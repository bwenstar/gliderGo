package house

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Binary codec for the original house format: big-endian, no padding, no magic
// number, no checksum. A house file is
//
//	houseType header      866 bytes   (offsetof(houseType, rooms))
//	roomType  × nRooms    348 each
//	slack                 0 or 2 bytes
//
// with this header layout (byte offsets, from GliderStructs.h; the analysis in
// docs/analysis/house-format.md 4 derives the same table independently):
//
//	  0  version        short
//	  2  unusedShort    short
//	  4  timeStamp      long        Mac epoch, low bit = locked
//	  8  flags          long
//	 12  initial        Point       {v, h}
//	 16  banner         Str255      256
//	272  trailer        Str255      256
//	528  highScores     scoresType  292
//	820  savedGame      gameType     40
//	860  hasGame        Boolean
//	861  unusedBoolean  Boolean
//	862  firstRoom      short
//	864  nRooms         short
//	866  rooms[]
//
// There is no version negotiation to do on load: 1.1.2 wrote 0x0200 and read
// anything, and all 22 shipped houses are 0x0200. Load therefore accepts any
// version and records it, leaving policy to the caller -- the alternative is a
// loader that rejects a file the original would have opened.

// Header field offsets. Named because tests assert against them and because a
// hand-inspected hex dump is a routine debugging step for this format.
const (
	offVersion       = 0
	offUnusedShort   = 2
	offTimeStamp     = 4
	offFlags         = 8
	offInitial       = 12
	offBanner        = 16
	offTrailer       = 272
	offHighScores    = 528
	offSavedGame     = 820
	offHasGame       = 860
	offUnusedBoolean = 861
	offFirstRoom     = 862
	offNRooms        = 864
	offRooms         = SizeofHouseHeader
)

// Room field offsets, same reasoning.
const (
	offRoomName       = 0
	offRoomBounds     = 28
	offRoomLeftStart  = 30
	offRoomRightStart = 31
	offRoomUnusedByte = 32
	offRoomVisited    = 33
	offRoomBackground = 34
	offRoomTiles      = 36
	offRoomFloor      = 52
	offRoomSuite      = 54
	offRoomOpenings   = 56
	offRoomNumObjects = 58
	offRoomObjects    = 60
)

// ErrShort is returned when a file ends inside a structure. It is wrapped, so
// callers can distinguish "truncated" from "malformed" with errors.Is.
var ErrShort = errors.New("house: file truncated")

// decoder is a big-endian cursor. Every read is bounds-checked once, at the
// point of use, so a truncated file produces one clear error naming the field
// rather than a panic or a zero-filled struct.
type decoder struct {
	b   []byte
	off int
}

func (d *decoder) need(n int, what string) error {
	if d.off+n > len(d.b) {
		return fmt.Errorf("%w: need %d bytes for %s at offset %d, have %d",
			ErrShort, n, what, d.off, len(d.b)-d.off)
	}
	return nil
}

func (d *decoder) u8() byte {
	v := d.b[d.off]
	d.off++
	return v
}

func (d *decoder) i16() int16 {
	v := be16(d.b[d.off:])
	d.off += 2
	return v
}

func (d *decoder) i32() int32 {
	v := int32(uint32(d.b[d.off])<<24 | uint32(d.b[d.off+1])<<16 |
		uint32(d.b[d.off+2])<<8 | uint32(d.b[d.off+3]))
	d.off += 4
	return v
}

func (d *decoder) u32() uint32 { return uint32(d.i32()) }

func (d *decoder) bytes(dst []byte) {
	copy(dst, d.b[d.off:d.off+len(dst)])
	d.off += len(dst)
}

func (d *decoder) point() Point { return Point{V: d.i16(), H: d.i16()} }

func (d *decoder) rect() Rect {
	return Rect{Top: d.i16(), Left: d.i16(), Bottom: d.i16(), Right: d.i16()}
}

// encoder is the mirror image. It writes into a pre-sized buffer, because the
// output length is known exactly before the first byte is written -- if it were
// not, this codec would not be byte-exact.
type encoder struct {
	b   []byte
	off int
}

func (e *encoder) u8(v byte) {
	e.b[e.off] = v
	e.off++
}

func (e *encoder) i16(v int16) {
	putBE16(e.b[e.off:], v)
	e.off += 2
}

func (e *encoder) i32(v int32) {
	u := uint32(v)
	e.b[e.off] = byte(u >> 24)
	e.b[e.off+1] = byte(u >> 16)
	e.b[e.off+2] = byte(u >> 8)
	e.b[e.off+3] = byte(u)
	e.off += 4
}

func (e *encoder) u32(v uint32) { e.i32(int32(v)) }

func (e *encoder) bytes(src []byte) {
	copy(e.b[e.off:], src)
	e.off += len(src)
}

func (e *encoder) point(p Point) { e.i16(p.V); e.i16(p.H) }

func (e *encoder) rect(r Rect) { e.i16(r.Top); e.i16(r.Left); e.i16(r.Bottom); e.i16(r.Right) }

// ------------------------------------------------------------------- decoding

// Load parses a house file. The returned House holds no reference to b.
//
// Load is strict about structure and permissive about content: a short file, a
// negative nRooms or a trailing-byte count other than 0 or PowerPCSlack is an
// error, but out-of-range coordinates, undefined `what` codes and nonsense
// Boolean bytes all load unchanged. That split is deliberate. The original's own
// validation (HouseLegal.c) ran on demand from the editor, not on load, so any
// house it could open must open here too; clamping belongs in a separate,
// explicit pass, not silently inside the reader.
func Load(b []byte) (*House, error) {
	d := &decoder{b: b}
	h := &House{}
	if err := d.header(h); err != nil {
		return nil, err
	}

	if err := d.need(int(h.NRooms)*SizeofRoom, "rooms"); err != nil {
		return nil, err
	}
	h.Rooms = make([]Room, h.NRooms)
	for i := range h.Rooms {
		d.room(&h.Rooms[i])
	}

	// Trailing bytes. Two is the documented PowerPC slack (house.go); anything
	// else means we have misread the file, and saying so beats loading a house
	// that will not round-trip.
	switch tail := len(b) - d.off; tail {
	case 0:
	case PowerPCSlack:
		h.Slack = append([]byte(nil), b[d.off:]...)
	default:
		return nil, fmt.Errorf("house: %d bytes past the last of %d rooms; "+
			"expected 0 or %d (see docs/analysis/house-format.md 12.4)",
			tail, h.NRooms, PowerPCSlack)
	}
	return h, nil
}

// header decodes the 866 bytes before rooms[] and leaves d positioned at the first
// room. It is separate from Load only so that PeekFile can stop here: a picker that
// wants a name and a room count has no business reading 4,070 rooms, and the header
// is also the only part a file has to have to be recognisable as a house at all.
func (d *decoder) header(h *House) error {
	if err := d.need(SizeofHouseHeader, "house header"); err != nil {
		return err
	}
	h.Version = d.i16()
	h.UnusedShort = d.i16()
	h.TimeStamp = d.i32()
	h.Flags = d.i32()
	h.Initial = d.point()
	d.bytes(h.Banner[:])
	d.bytes(h.Trailer[:])
	d.scores(&h.HighScores)
	d.game(&h.SavedGame)
	h.HasGame = d.u8()
	h.UnusedBoolean = d.u8()
	h.FirstRoom = d.i16()
	h.NRooms = d.i16()

	if d.off != offRooms { // unreachable unless the code above drifts
		return fmt.Errorf("house: internal: header decoded to %d bytes, want %d",
			d.off, offRooms)
	}
	if h.NRooms < 0 {
		return fmt.Errorf("house: nRooms is negative (%d)", h.NRooms)
	}
	return nil
}

func (d *decoder) scores(s *Scores) {
	d.bytes(s.Banner[:])
	for i := range s.Names {
		d.bytes(s.Names[i][:])
	}
	for i := range s.Scores {
		s.Scores[i] = d.i32()
	}
	for i := range s.TimeStamps {
		s.TimeStamps[i] = d.u32()
	}
	for i := range s.Levels {
		s.Levels[i] = d.i16()
	}
}

func (d *decoder) game(g *Game) {
	g.Version = d.i16()
	g.WasStarsLeft = d.i16()
	g.TimeStamp = d.i32()
	g.Where = d.point()
	g.Score = d.i32()
	g.UnusedLong = d.i32()
	g.UnusedLong2 = d.i32()
	g.Energy = d.i16()
	g.Bands = d.i16()
	g.RoomNumber = d.i16()
	g.GliderState = d.i16()
	g.NumGliders = d.i16()
	g.Foil = d.i16()
	g.UnusedShort = d.i16()
	g.Facing = d.u8()
	g.ShowFoil = d.u8()
}

func (d *decoder) room(r *Room) {
	d.bytes(r.Name[:])
	r.Bounds = d.i16()
	r.LeftStart = d.u8()
	r.RightStart = d.u8()
	r.UnusedByte = d.u8()
	r.Visited = d.u8()
	r.Background = d.i16()
	for i := range r.Tiles {
		r.Tiles[i] = d.i16()
	}
	r.Floor = d.i16()
	r.Suite = d.i16()
	r.Openings = d.i16()
	r.NumObjects = d.i16()
	for i := range r.Objects {
		r.Objects[i].What = d.i16()
		d.bytes(r.Objects[i].Data[:])
	}
}

// LoadFile reads and parses a house file, naming the file in any error.
func LoadFile(path string) (*House, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	h, err := Load(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return h, nil
}

// LoadFS is LoadFile against a filesystem: one house out of the copy built into the
// executable, or out of a directory somebody named. name is a slash-separated path
// within fsys and is what any error calls the house.
func LoadFS(fsys fs.FS, name string) (*House, error) {
	if fsys == nil {
		return nil, fmt.Errorf("%s: no houses to read it from", name)
	}
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	h, err := Load(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return h, nil
}

// ------------------------------------------------------------------- encoding

// Size returns the exact number of bytes Save will produce.
func (h *House) Size() int {
	return SizeofHouseHeader + len(h.Rooms)*SizeofRoom + len(h.Slack)
}

// Save serialises the house. For any house produced by Load, Save reproduces the
// input byte for byte -- residue, unused fields, empty-slot union bytes and the
// PowerPC slack included. That is the property TestCorpusRoundTrip pins across
// all 22 shipped houses.
//
// NRooms is written from len(Rooms) rather than from the field, so the two cannot
// drift; Load guarantees they start out equal.
func (h *House) Save() ([]byte, error) {
	if len(h.Rooms) > 0x7FFF {
		return nil, fmt.Errorf("house: %d rooms exceeds the short nRooms field",
			len(h.Rooms))
	}
	if n := len(h.Slack); n != 0 && n != PowerPCSlack {
		return nil, fmt.Errorf("house: slack is %d bytes; expected 0 or %d",
			n, PowerPCSlack)
	}

	e := &encoder{b: make([]byte, h.Size())}
	e.i16(h.Version)
	e.i16(h.UnusedShort)
	e.i32(h.TimeStamp)
	e.i32(h.Flags)
	e.point(h.Initial)
	e.bytes(h.Banner[:])
	e.bytes(h.Trailer[:])
	e.scores(&h.HighScores)
	e.game(&h.SavedGame)
	e.u8(h.HasGame)
	e.u8(h.UnusedBoolean)
	e.i16(h.FirstRoom)
	e.i16(int16(len(h.Rooms)))

	if e.off != offRooms {
		return nil, fmt.Errorf("house: internal: header encoded to %d bytes, want %d",
			e.off, offRooms)
	}
	for i := range h.Rooms {
		e.room(&h.Rooms[i])
	}
	e.bytes(h.Slack)

	if e.off != len(e.b) {
		return nil, fmt.Errorf("house: internal: encoded %d of %d bytes",
			e.off, len(e.b))
	}
	return e.b, nil
}

func (e *encoder) scores(s *Scores) {
	e.bytes(s.Banner[:])
	for i := range s.Names {
		e.bytes(s.Names[i][:])
	}
	for i := range s.Scores {
		e.i32(s.Scores[i])
	}
	for i := range s.TimeStamps {
		e.u32(s.TimeStamps[i])
	}
	for i := range s.Levels {
		e.i16(s.Levels[i])
	}
}

func (e *encoder) game(g *Game) {
	e.i16(g.Version)
	e.i16(g.WasStarsLeft)
	e.i32(g.TimeStamp)
	e.point(g.Where)
	e.i32(g.Score)
	e.i32(g.UnusedLong)
	e.i32(g.UnusedLong2)
	e.i16(g.Energy)
	e.i16(g.Bands)
	e.i16(g.RoomNumber)
	e.i16(g.GliderState)
	e.i16(g.NumGliders)
	e.i16(g.Foil)
	e.i16(g.UnusedShort)
	e.u8(g.Facing)
	e.u8(g.ShowFoil)
}

func (e *encoder) room(r *Room) {
	e.bytes(r.Name[:])
	e.i16(r.Bounds)
	e.u8(r.LeftStart)
	e.u8(r.RightStart)
	e.u8(r.UnusedByte)
	e.u8(r.Visited)
	e.i16(r.Background)
	for i := range r.Tiles {
		e.i16(r.Tiles[i])
	}
	e.i16(r.Floor)
	e.i16(r.Suite)
	e.i16(r.Openings)
	e.i16(r.NumObjects)
	for i := range r.Objects {
		e.i16(r.Objects[i].What)
		e.bytes(r.Objects[i].Data[:])
	}
}

// SaveFile writes the house to path via a temporary file and a rename, so an
// interrupted save cannot leave a half-written house where a good one was.
func (h *House) SaveFile(path string) error {
	b, err := h.Save()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".glidergo-house-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name) // no-op once the rename has succeeded
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// WriteTo makes House an io.WriterTo, for callers that already have a stream.
func (h *House) WriteTo(w io.Writer) (int64, error) {
	b, err := h.Save()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(b)
	return int64(n), err
}

// ------------------------------------------------- the high-score board, alone

// The board is the one part of a house that has a life outside the file, and it has
// had one since 1994. Glider PRO writes a house's high scores back into the house
// itself (HouseIO.c), which means a read-only house -- one on a CD, one a player
// downloaded, one shipped inside an application -- cannot record a score at all.
// The original grew a second path for exactly that: a 292-byte side-car in the
// Preferences folder holding nothing but a scoresType, creator 'ozm5', type 'gliS'
// (docs/analysis/scoring.md 7.14). The code to read it survives in HighScores.c and
// is dead, because nothing ever wrote one.
//
// This port takes the side-car as the only place it writes (internal/scores), so the
// vendored houses stay byte-identical and a house from anywhere is playable. That
// needs the board's 292 bytes on their own, without a house around them, which is
// what these two are for. Save and Load keep using the unexported halves, so there
// is exactly one description of the layout and the side-car cannot drift from the
// header it came out of.

// EncodeScores serialises one board as the same SizeofScores big-endian bytes it
// occupies inside a house header.
//
// Every byte is written from the struct, including the residue past each Pascal
// string's length -- so a board that was read out of a shipped house and never
// modified encodes back to the bytes it came from, garbage included. That is not
// tidiness: SortHighScores sorted through an uninitialised local, so the residue in
// the shipped boards is real 1994 stack garbage, and reproducing it is what lets
// TestScoresCodecMatchesEveryShippedHeader compare bytes rather than fields.
func EncodeScores(s *Scores) []byte {
	e := &encoder{b: make([]byte, SizeofScores)}
	e.scores(s)
	if e.off != len(e.b) {
		// Unreachable: SizeofScores and encoder.scores describe the same layout, and
		// the corpus round-trip would fail first. Checked anyway, because a silent
		// short write here would produce a side-car that decodes to another board.
		panic(fmt.Sprintf("house: internal: EncodeScores wrote %d of %d bytes",
			e.off, len(e.b)))
	}
	return e.b
}

// DecodeScores parses a board from exactly SizeofScores bytes.
//
// The length is required rather than tolerated, which is the same split Load draws:
// strict about structure, permissive about content. Any ten names, any ten scores and
// any ten timestamps decode unchanged, however impossible -- Repair, not the reader,
// decides what a board with a negative score means.
//
// The original is the counter-example. HighScores.c reads its side-car with GetEOF
// followed by one FSRead of that many bytes straight into the house handle's
// highScores member, with no check that the file is 292 bytes long: a longer file
// overwrites the saved game and the room count behind it, and a shorter one leaves
// the tail of the board holding whatever the house already had. A caller here that
// wants to survive a truncated file overlays it onto a known-good board and says so
// (internal/scores), which is a decision at the call site rather than an accident in
// the reader.
func DecodeScores(b []byte) (Scores, error) {
	var s Scores
	if len(b) != SizeofScores {
		return s, fmt.Errorf("house: high-score board is %d bytes, want %d",
			len(b), SizeofScores)
	}
	d := &decoder{b: b}
	d.scores(&s)
	if d.off != SizeofScores { // unreachable, as above
		return s, fmt.Errorf("house: internal: DecodeScores read %d of %d bytes",
			d.off, SizeofScores)
	}
	return s, nil
}
