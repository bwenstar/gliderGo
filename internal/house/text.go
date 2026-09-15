package house

// The text format: a line-oriented, hand-authorable rendering of a house.
//
// It exists for three jobs the binary format is bad at -- reviewing a house in a
// diff, authoring the new houses of Stage 2, and reading one in a bug report --
// and it is deliberately *field*-exact rather than byte-exact. The binary codec
// is the archival one (Save(Load(b)) == b for all 22 shipped houses); this one
// carries every named field and drops the unnamed bytes between them.
//
// That split is forced by the corpus, not chosen for convenience. Of the 66,240
// empty object slots in the 22 shipped houses, 66,236 hold non-zero residue --
// uninitialised memory left behind because CreateNewRoom only ever set `what`
// (GliderPRO/Sources/Room.c:181-182) -- and 4,008 of 4,070 room names have
// non-zero bytes past their length byte. A text format that reproduced all of it
// would be four fifths hex noise and useless for every one of its three jobs.
//
// So the guarantee is stated precisely, and tested that way:
//
//	ParseText(WriteText(h))  ==  h.Canonical()
//
// where Canonical zeroes exactly the bytes the text format does not carry, and
// nothing else. WriteText(TextOptions{Residue: true}) additionally emits that
// residue, which makes the round trip byte-exact for the price of a much noisier
// file; the parser always accepts it.
//
// Two conventions in the output are worth knowing before reading one:
//
//   - Coordinates are vertical first (`at 168 320` is v=168, h=320), because
//     that is what a QuickDraw Point is. The header comment says so in every
//     file written, since this is the single easiest thing to get backwards.
//   - Fields the original declared as Mac `Boolean` are written as integers, not
//     as yes/no. They are bytes, and the corpus contains blowers whose `state`
//     byte is 23 and houses whose `unusedBoolean` is 255. Rendering those as a
//     boolean would silently rewrite them.

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// TextFormatVersion is the version of this text syntax, written as the first
// directive of every file and checked on parse.
const TextFormatVersion = 1

// TextOptions controls what WriteText emits.
type TextOptions struct {
	// Residue emits the uninitialised bytes the format otherwise drops:
	// `residue` lines on Pascal strings and `raw` on empty object slots. With
	// it set, a parse of the output reproduces the input byte for byte.
	Residue bool

	// NoHeader suppresses the leading comment block, for tests and for diffing
	// two houses without the boilerplate.
	NoHeader bool
}

// Canonical returns a copy of h with exactly the bytes the text format does not
// carry set to zero: the residue past every Pascal string's length byte, and the
// 10 union bytes of every empty object slot. Every named field is untouched,
// including the ones the original never used and the stale saved game that
// twenty of the shipped houses carry behind hasGame == 0.
//
// It is also the answer to "how much of this file means anything?": a canonical
// house is what the same house would look like had it been written by a program
// that cleared its memory first.
func (h *House) Canonical() *House {
	c := *h
	c.Rooms = make([]Room, len(h.Rooms))
	copy(c.Rooms, h.Rooms)
	c.Slack = append([]byte(nil), h.Slack...)

	clearResidue(c.Banner[:])
	clearResidue(c.Trailer[:])
	clearResidue(c.HighScores.Banner[:])
	for i := range c.HighScores.Names {
		clearResidue(c.HighScores.Names[i][:])
	}
	for i := range c.Rooms {
		r := &c.Rooms[i]
		clearResidue(r.Name[:])
		for j := range r.Objects {
			if r.Objects[j].IsEmpty() {
				r.Objects[j].Data = [10]byte{}
			}
		}
	}
	return &c
}

func clearResidue(b []byte) {
	n := int(b[0])
	if n > len(b)-1 {
		n = len(b) - 1
	}
	for i := 1 + n; i < len(b); i++ {
		b[i] = 0
	}
}

// -------------------------------------------------------------------- writing

const textHeader = `# gliderGo house, text format %d
#
# Coordinates are VERTICAL FIRST: "at 168 320" means v=168, h=320, because a
# classic QuickDraw Point is {v, h}. Rects are top left bottom right.
#
# Fields the 1994 source declared as Boolean are written as integers, because
# they are bytes and the shipped houses contain values other than 0 and 1.
#
# Omitted fields default to zero. Object slots not listed are empty. Room and
# object numbers are load-bearing -- links address them by index -- so they are
# written explicitly and must stay in ascending order.
#
# See docs/analysis/house-format.md for what every field means.

`

type textWriter struct {
	w   *bufio.Writer
	err error
	opt TextOptions
}

func (t *textWriter) printf(format string, args ...any) {
	if t.err != nil {
		return
	}
	_, t.err = fmt.Fprintf(t.w, format, args...)
}

// kv writes an aligned key/value line. The width is chosen so the longest field
// name in the format ("unusedboolean") still leaves values in one column.
func (t *textWriter) kv(key, format string, args ...any) {
	t.printf("    %-14s %s\n", key, fmt.Sprintf(format, args...))
}

// kvc is kv with a trailing explanatory comment.
func (t *textWriter) kvc(key, value, comment string) {
	if comment == "" {
		t.printf("    %-14s %s\n", key, value)
		return
	}
	t.printf("    %-14s %-20s # %s\n", key, value, comment)
}

// Text renders the house as a string.
func (h *House) Text(opt TextOptions) (string, error) {
	var sb strings.Builder
	if err := h.WriteText(&sb, opt); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// WriteText renders the house in the text format.
func (h *House) WriteText(w io.Writer, opt TextOptions) error {
	t := &textWriter{w: bufio.NewWriter(w), opt: opt}
	if !opt.NoHeader {
		t.printf(textHeader, TextFormatVersion)
	}
	t.printf("format %d\n\n", TextFormatVersion)

	t.printf("house\n")
	t.kvc("version", fmt.Sprintf("0x%04x", uint16(h.Version)), versionComment(h.Version))
	t.kvc("timestamp", strconv.FormatInt(int64(h.TimeStamp), 10), macMaskedTimeString(h.TimeStamp))
	t.kvc("flags", fmt.Sprintf("0x%08x", uint32(h.Flags)), flagsComment(h.Flags))
	t.kv("initial", "%d %d", h.Initial.V, h.Initial.H)
	t.kv("firstroom", "%d", h.FirstRoom)
	t.pstr("banner", h.Banner[:])
	t.pstr("trailer", h.Trailer[:])
	if h.HasGame != 0 {
		t.kv("hasgame", "%d", h.HasGame)
	}
	if h.UnusedShort != 0 {
		t.kv("unusedshort", "%d", h.UnusedShort)
	}
	if h.UnusedBoolean != 0 {
		t.kv("unusedboolean", "%d", h.UnusedBoolean)
	}
	if len(h.Slack) > 0 {
		t.kvc("slack", hexBytes(h.Slack), "PowerPC tail, see house-format.md 12.4")
	}
	t.printf("\n")

	t.scores(&h.HighScores)
	t.game(&h.SavedGame)
	for i := range h.Rooms {
		t.room(i, &h.Rooms[i])
	}

	if t.err != nil {
		return t.err
	}
	return t.w.Flush()
}

// pstr writes a Pascal string, plus its residue when asked for.
func (t *textWriter) pstr(key string, b []byte) {
	t.kv(key, "%s", QuoteMacRoman(pstrText(b)))
	if t.opt.Residue {
		if res := pstrResidue(b); hasResidue(b) {
			t.kv(key+".residue", "%s", hexBytes(res))
		}
	}
}

func (t *textWriter) scores(s *Scores) {
	if *s == (Scores{}) {
		return
	}
	t.printf("scores\n")
	t.pstr("banner", s.Banner[:])
	t.printf("    #     %-19s %10s  %12s  %s\n", "name", "score", "timestamp", "rooms")
	for i := range s.Names {
		// The date is a comment because nothing reads it back and nothing ever drew it
		// (docs/IMPROVEMENTS.md 2.57) -- but it is the only record of when the shipped
		// boards were set, so a dump is where somebody can see it.
		row := fmt.Sprintf("    entry %-19s %10d  %12d  %d",
			QuoteMacRoman(s.Names[i].Text()), s.Scores[i], s.TimeStamps[i], s.Levels[i])
		if c := timeComment(int32(s.TimeStamps[i])); c != "" {
			t.printf("%-56s # %s\n", row, c)
		} else {
			t.printf("%s\n", row)
		}
		if t.opt.Residue && s.Names[i].HasResidue() {
			t.kv(fmt.Sprintf("entry.%d.residue", i), "%s", hexBytes(s.Names[i].Residue()))
		}
	}
	t.printf("\n")
}

func (t *textWriter) game(g *Game) {
	if *g == (Game{}) {
		return
	}
	t.printf("game\n")
	t.kv("version", "%d", g.Version)
	t.kv("wasstarsleft", "%d", g.WasStarsLeft)
	t.kvc("timestamp", strconv.FormatInt(int64(g.TimeStamp), 10), timeComment(g.TimeStamp))
	t.kv("where", "%d %d", g.Where.V, g.Where.H)
	t.kv("score", "%d", g.Score)
	t.kv("energy", "%d", g.Energy)
	t.kv("bands", "%d", g.Bands)
	t.kv("roomnumber", "%d", g.RoomNumber)
	t.kv("gliderstate", "%d", g.GliderState)
	t.kv("numgliders", "%d", g.NumGliders)
	t.kv("foil", "%d", g.Foil)
	t.kv("facing", "%d", g.Facing)
	t.kv("showfoil", "%d", g.ShowFoil)
	if g.UnusedLong != 0 {
		t.kv("unusedlong", "%d", g.UnusedLong)
	}
	if g.UnusedLong2 != 0 {
		t.kv("unusedlong2", "%d", g.UnusedLong2)
	}
	if g.UnusedShort != 0 {
		t.kv("unusedshort", "%d", g.UnusedShort)
	}
	t.printf("\n")
}

func (t *textWriter) room(n int, r *Room) {
	t.printf("room %d %s\n", n, QuoteMacRoman(r.Name.Text()))
	if t.opt.Residue && r.Name.HasResidue() {
		t.kv("name.residue", "%s", hexBytes(r.Name.Residue()))
	}
	t.kvc("bounds", fmt.Sprintf("0x%04x", uint16(r.Bounds)), boundsComment(r.Bounds))
	t.kvc("background", strconv.Itoa(int(r.Background)), backgroundComment(r.Background))
	t.kv("tiles", "%s", intsJoin(r.Tiles[:]))
	t.kv("floor", "%d", r.Floor)
	t.kv("suite", "%d", r.Suite)
	t.kv("start", "%d %d", r.LeftStart, r.RightStart)
	if r.Visited != 0 {
		t.kv("visited", "%d", r.Visited)
	}
	if r.UnusedByte != 0 {
		t.kv("unusedbyte", "%d", r.UnusedByte)
	}
	if r.Openings != 0 {
		t.kvc("openings", strconv.Itoa(int(r.Openings)),
			"dead field; the original recomputes openings")
	}
	// numObjects is advisory -- the original recomputes it from the live slots
	// (MakeSureNumObjectsJives) and it agrees in all 4,070 shipped rooms, so the
	// parser derives it. Written only if this house disagrees, so that even a
	// malformed count survives a round trip.
	if int(r.NumObjects) != r.LiveObjects() {
		t.kvc("numobjects", strconv.Itoa(int(r.NumObjects)),
			fmt.Sprintf("disagrees with the %d live slots; preserved verbatim",
				r.LiveObjects()))
	}
	for i := range r.Objects {
		t.object(i, r.Objects[i])
	}
	t.printf("\n")
}

func (t *textWriter) object(slot int, o Object) {
	if o.IsEmpty() {
		if t.opt.Residue && o.Data != [10]byte{} {
			t.printf("    object %-2d empty  raw %s\n", slot, hexBytes(o.Data[:]))
		}
		return
	}

	name := ObjectName(o.What)
	if name == "" {
		// Undefined `what`. None occur in the shipped houses; a user house may
		// still contain one and must survive being read and written.
		t.printf("    object %-2d 0x%02x  raw %s   # undefined type\n",
			slot, uint16(o.What), hexBytes(o.Data[:]))
		return
	}

	var f string
	switch o.Group() {
	case GroupBlower:
		b := o.Blower()
		f = fmt.Sprintf("at %d %d  distance %d  initial %d  state %d  vector %d  tall %d",
			b.TopLeft.V, b.TopLeft.H, b.Distance, b.Initial, b.State, b.Vector, b.Tall)
	case GroupFurniture:
		b := o.Furniture()
		f = fmt.Sprintf("rect %d %d %d %d  pict %d",
			b.Bounds.Top, b.Bounds.Left, b.Bounds.Bottom, b.Bounds.Right, b.Pict)
	case GroupBonus:
		c := o.Bonus()
		f = fmt.Sprintf("at %d %d  length %d  points %d  state %d  initial %d",
			c.TopLeft.V, c.TopLeft.H, c.Length, c.Points, c.State, c.Initial)
	case GroupTransport:
		d := o.Transport()
		f = fmt.Sprintf("at %d %d  tall %d  where %d  who %d  wide %d",
			d.TopLeft.V, d.TopLeft.H, d.Tall, d.Where, d.Who, d.Wide)
	case GroupSwitch:
		e := o.Switch()
		f = fmt.Sprintf("at %d %d  delay %d  where %d  who %d  type %d",
			e.TopLeft.V, e.TopLeft.H, e.Delay, e.Where, e.Who, e.Type)
	case GroupLight:
		l := o.Light()
		f = fmt.Sprintf("at %d %d  length %d  byte0 %d  byte1 %d  initial %d  state %d",
			l.TopLeft.V, l.TopLeft.H, l.Length, l.Byte0, l.Byte1, l.Initial, l.State)
	case GroupAppliance:
		g := o.Appliance()
		f = fmt.Sprintf("at %d %d  height %d  byte0 %d  delay %d  initial %d  state %d",
			g.TopLeft.V, g.TopLeft.H, g.Height, g.Byte0, g.Delay, g.Initial, g.State)
	case GroupEnemy:
		h := o.Enemy()
		f = fmt.Sprintf("at %d %d  length %d  delay %d  byte0 %d  initial %d  state %d",
			h.TopLeft.V, h.TopLeft.H, h.Length, h.Delay, h.Byte0, h.Initial, h.State)
	case GroupClutter:
		i := o.Clutter()
		f = fmt.Sprintf("rect %d %d %d %d  pict %d",
			i.Bounds.Top, i.Bounds.Left, i.Bounds.Bottom, i.Bounds.Right, i.Pict)
	}
	t.printf("    object %-2d %-15s %s\n", slot, name, f)
}

// ---------------------------------------------------------------- annotations

func versionComment(v int16) string {
	switch v {
	case HouseVersion:
		return "kHouseVersion"
	case NewHouseVersion:
		return "kNewHouseVersion"
	}
	return ""
}

// timeComment renders a Mac timestamp as a date, as a comment only -- the number
// is the value, and no timezone assumption is baked into the format.
func timeComment(mac int32) string {
	if mac == 0 {
		return ""
	}
	return macTimeString(mac)
}

func flagsComment(f int32) string {
	var on []string
	if f&FlagWard != 0 {
		on = append(on, "ward")
	}
	if f&FlagPhone != 0 {
		on = append(on, "phone")
	}
	if f&FlagNoStarCount != 0 {
		on = append(on, "no star count")
	}
	if rest := f &^ (FlagWard | FlagPhone | FlagNoStarCount); rest != 0 {
		on = append(on, fmt.Sprintf("unknown 0x%x", uint32(rest)))
	}
	if len(on) == 0 {
		return ""
	}
	return strings.Join(on, ", ")
}

// boundsComment decodes the room bounds bitfield (house-format.md 4.2). Bit 0 is
// a marker: when clear, the whole field is meaningless and the original falls
// back to a 'bnds' resource.
func boundsComment(b int16) string {
	if b == 0 {
		return "unset: falls back to the 'bnds' resource"
	}
	var on []string
	if b&0x02 != 0 {
		on = append(on, "left")
	}
	if b&0x04 != 0 {
		on = append(on, "top")
	}
	if b&0x08 != 0 {
		on = append(on, "right")
	}
	if b&0x10 != 0 {
		on = append(on, "bottom")
	}
	if b&0x20 != 0 {
		on = append(on, "structure")
	}
	if b&0x01 == 0 {
		on = append(on, "!! marker bit clear")
	}
	if len(on) == 0 {
		return "all closed"
	}
	return "open: " + strings.Join(on, " ")
}

func backgroundComment(id int16) string {
	if id == 0 {
		return "none"
	}
	if id >= 2000 && id <= 2017 {
		return "built-in"
	}
	return "house PICT"
}

func hexBytes(b []byte) string {
	var sb strings.Builder
	for i, c := range b {
		if i > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%02x", c)
	}
	return sb.String()
}

func intsJoin(v []int16) string {
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = strconv.Itoa(int(n))
	}
	return strings.Join(parts, " ")
}

// -------------------------------------------------------------------- parsing

// ParseText reads the text format. Errors name the line number and the offending
// token, because this format is meant to be typed by hand.
func ParseText(r io.Reader) (*House, error) {
	p := &textParser{sc: bufio.NewScanner(r), h: &House{}}
	p.sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	if err := p.run(); err != nil {
		return nil, err
	}
	return p.finish()
}

// ParseTextFile is ParseText on a file, naming the file in any error.
func ParseTextFile(path string) (*House, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h, err := ParseText(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return h, nil
}

type textParser struct {
	sc   *bufio.Scanner
	line int
	h    *House

	sawFormat bool
	section   string // "", "house", "scores", "game", "room"
	room      *Room  // current room when section == "room"
	roomIndex int
	scoreRow  int
	lastSlot  int
	explicitN map[int]int16 // room index -> explicit numobjects override
}

func (p *textParser) errf(format string, args ...any) error {
	return fmt.Errorf("line %d: %s", p.line, fmt.Sprintf(format, args...))
}

func (p *textParser) run() error {
	p.roomIndex = -1
	p.explicitN = map[int]int16{}
	for p.sc.Scan() {
		p.line++
		toks, err := tokenize(p.sc.Text())
		if err != nil {
			return p.errf("%v", err)
		}
		if len(toks) == 0 {
			continue
		}
		if err := p.directive(toks); err != nil {
			return err
		}
	}
	return p.sc.Err()
}

func (p *textParser) directive(toks []string) error {
	switch toks[0] {
	case "format":
		if p.sawFormat {
			return p.errf("duplicate format directive")
		}
		v, err := p.int(toks, 1, "format version")
		if err != nil {
			return err
		}
		if v != TextFormatVersion {
			return p.errf("text format version %d, this build reads %d", v, TextFormatVersion)
		}
		p.sawFormat = true
		return nil

	case "house", "scores", "game":
		if !p.sawFormat {
			return p.errf("expected a `format %d` directive first", TextFormatVersion)
		}
		if len(toks) != 1 {
			return p.errf("`%s` takes no arguments", toks[0])
		}
		p.section, p.room, p.scoreRow = toks[0], nil, 0
		return nil

	case "room":
		if !p.sawFormat {
			return p.errf("expected a `format %d` directive first", TextFormatVersion)
		}
		return p.beginRoom(toks)
	}

	switch p.section {
	case "house":
		return p.houseField(toks)
	case "scores":
		return p.scoresField(toks)
	case "game":
		return p.gameField(toks)
	case "room":
		return p.roomField(toks)
	}
	return p.errf("%q outside any section", toks[0])
}

func (p *textParser) beginRoom(toks []string) error {
	if len(toks) < 2 {
		return p.errf("`room` needs a number")
	}
	n, err := p.int(toks, 1, "room number")
	if err != nil {
		return err
	}
	if n != p.roomIndex+1 {
		return p.errf("room %d out of order: expected room %d "+
			"(room numbers are addressed by links and cannot be renumbered)",
			n, p.roomIndex+1)
	}
	p.roomIndex = n
	p.h.Rooms = append(p.h.Rooms, Room{})
	p.room = &p.h.Rooms[len(p.h.Rooms)-1]
	p.section, p.lastSlot = "room", -1

	// Empty slots are the default, so every slot starts empty and only the
	// listed ones are filled in.
	for i := range p.room.Objects {
		p.room.Objects[i].What = ObjectIsEmpty
	}

	if len(toks) > 2 {
		name, err := UnquoteMacRoman(toks[2])
		if err != nil {
			return p.errf("room name: %v", err)
		}
		if !p.room.Name.SetText(name) {
			return p.errf("room name %q does not fit a Str27", name)
		}
	}
	if len(toks) > 3 {
		return p.errf("unexpected %q after the room name", toks[3])
	}
	return nil
}

func (p *textParser) houseField(toks []string) error {
	h := p.h
	switch toks[0] {
	case "version":
		return p.i16(toks, &h.Version, "version")
	case "timestamp":
		return p.i32(toks, &h.TimeStamp, "timestamp")
	case "flags":
		return p.i32(toks, &h.Flags, "flags")
	case "initial":
		return p.point(toks, &h.Initial)
	case "firstroom":
		return p.i16(toks, &h.FirstRoom, "firstroom")
	case "banner":
		return p.pstr(toks, h.Banner[:], "banner")
	case "banner.residue":
		return p.residue(toks, h.Banner[:])
	case "trailer":
		return p.pstr(toks, h.Trailer[:], "trailer")
	case "trailer.residue":
		return p.residue(toks, h.Trailer[:])
	case "hasgame":
		return p.u8(toks, &h.HasGame, "hasgame")
	case "unusedshort":
		return p.i16(toks, &h.UnusedShort, "unusedshort")
	case "unusedboolean":
		return p.u8(toks, &h.UnusedBoolean, "unusedboolean")
	case "slack":
		b, err := p.hex(toks, PowerPCSlack)
		if err != nil {
			return err
		}
		h.Slack = b
		return nil
	}
	return p.errf("unknown house field %q", toks[0])
}

func (p *textParser) scoresField(toks []string) error {
	s := &p.h.HighScores
	switch toks[0] {
	case "banner":
		return p.pstr(toks, s.Banner[:], "scores banner")
	case "banner.residue":
		return p.residue(toks, s.Banner[:])
	case "entry":
		if p.scoreRow >= MaxScores {
			return p.errf("more than %d score entries", MaxScores)
		}
		if len(toks) != 5 {
			return p.errf("`entry` needs name, score, timestamp and rooms")
		}
		i := p.scoreRow
		p.scoreRow++
		if err := p.pstr(toks, s.Names[i][:], "score name"); err != nil {
			return err
		}
		v, err := p.int(toks, 2, "score")
		if err != nil {
			return err
		}
		s.Scores[i] = int32(v)
		if v, err = p.int(toks, 3, "score timestamp"); err != nil {
			return err
		}
		s.TimeStamps[i] = uint32(v)
		if v, err = p.int(toks, 4, "score rooms"); err != nil {
			return err
		}
		s.Levels[i] = int16(v)
		return nil
	}
	if row, field, ok := parseEntryResidue(toks[0]); ok && field == "residue" {
		if row < 0 || row >= MaxScores {
			return p.errf("score entry %d out of range", row)
		}
		return p.residue(toks, s.Names[row][:])
	}
	return p.errf("unknown scores field %q", toks[0])
}

// parseEntryResidue splits "entry.3.residue".
func parseEntryResidue(key string) (row int, field string, ok bool) {
	parts := strings.Split(key, ".")
	if len(parts) != 3 || parts[0] != "entry" {
		return 0, "", false
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, "", false
	}
	return n, parts[2], true
}

func (p *textParser) gameField(toks []string) error {
	g := &p.h.SavedGame
	switch toks[0] {
	case "version":
		return p.i16(toks, &g.Version, "version")
	case "wasstarsleft":
		return p.i16(toks, &g.WasStarsLeft, "wasstarsleft")
	case "timestamp":
		return p.i32(toks, &g.TimeStamp, "timestamp")
	case "where":
		return p.point(toks, &g.Where)
	case "score":
		return p.i32(toks, &g.Score, "score")
	case "unusedlong":
		return p.i32(toks, &g.UnusedLong, "unusedlong")
	case "unusedlong2":
		return p.i32(toks, &g.UnusedLong2, "unusedlong2")
	case "energy":
		return p.i16(toks, &g.Energy, "energy")
	case "bands":
		return p.i16(toks, &g.Bands, "bands")
	case "roomnumber":
		return p.i16(toks, &g.RoomNumber, "roomnumber")
	case "gliderstate":
		return p.i16(toks, &g.GliderState, "gliderstate")
	case "numgliders":
		return p.i16(toks, &g.NumGliders, "numgliders")
	case "foil":
		return p.i16(toks, &g.Foil, "foil")
	case "unusedshort":
		return p.i16(toks, &g.UnusedShort, "unusedshort")
	case "facing":
		return p.u8(toks, &g.Facing, "facing")
	case "showfoil":
		return p.u8(toks, &g.ShowFoil, "showfoil")
	}
	return p.errf("unknown game field %q", toks[0])
}

func (p *textParser) roomField(toks []string) error {
	r := p.room
	switch toks[0] {
	case "name.residue":
		return p.residue(toks, r.Name[:])
	case "bounds":
		return p.i16(toks, &r.Bounds, "bounds")
	case "background":
		return p.i16(toks, &r.Background, "background")
	case "floor":
		return p.i16(toks, &r.Floor, "floor")
	case "suite":
		return p.i16(toks, &r.Suite, "suite")
	case "visited":
		return p.u8(toks, &r.Visited, "visited")
	case "unusedbyte":
		return p.u8(toks, &r.UnusedByte, "unusedbyte")
	case "openings":
		return p.i16(toks, &r.Openings, "openings")
	case "numobjects":
		v, err := p.int(toks, 1, "numobjects")
		if err != nil {
			return err
		}
		p.explicitN[p.roomIndex] = int16(v)
		return nil
	case "start":
		if len(toks) != 3 {
			return p.errf("`start` needs leftStart and rightStart")
		}
		l, err := p.int(toks, 1, "leftStart")
		if err != nil {
			return err
		}
		rt, err := p.int(toks, 2, "rightStart")
		if err != nil {
			return err
		}
		r.LeftStart, r.RightStart = byte(l), byte(rt)
		return nil
	case "tiles":
		if len(toks) != 1+NumTiles {
			return p.errf("`tiles` needs exactly %d values, got %d", NumTiles, len(toks)-1)
		}
		for i := 0; i < NumTiles; i++ {
			v, err := p.int(toks, 1+i, "tile")
			if err != nil {
				return err
			}
			r.Tiles[i] = int16(v)
		}
		return nil
	case "object":
		return p.object(toks)
	}
	return p.errf("unknown room field %q", toks[0])
}

func (p *textParser) object(toks []string) error {
	if len(toks) < 3 {
		return p.errf("`object` needs a slot and a type")
	}
	slot, err := p.int(toks, 1, "object slot")
	if err != nil {
		return err
	}
	if slot < 0 || slot >= MaxRoomObs {
		return p.errf("object slot %d out of range 0..%d", slot, MaxRoomObs-1)
	}
	if slot <= p.lastSlot {
		return p.errf("object slot %d out of order: slots must ascend "+
			"(links address objects by slot)", slot)
	}
	p.lastSlot = slot
	o := &p.room.Objects[slot]

	// `empty [raw ...]` and `0xNN raw ...` are the two literal forms; anything
	// else is a type name with keyword fields.
	fields := toks[3:]
	if toks[2] == "empty" {
		o.What = ObjectIsEmpty
		return p.objectRaw(o, fields, true)
	}

	what, ok := ObjectCode(toks[2])
	if !ok {
		n, err := parseInt(toks[2])
		if err != nil {
			return p.errf("unknown object type %q", toks[2])
		}
		o.What = int16(n)
		if GroupOf(o.What) != GroupNone {
			return p.errf("0x%02x is %s; write it as %s, not as a number",
				uint16(o.What), ObjectName(o.What), ObjectName(o.What))
		}
		return p.objectRaw(o, fields, false)
	}
	o.What = what

	kv, err := p.keywords(fields)
	if err != nil {
		return err
	}
	return p.objectFields(o, kv)
}

// objectRaw handles the two forms whose 10 union bytes are written literally.
func (p *textParser) objectRaw(o *Object, fields []string, optional bool) error {
	if len(fields) == 0 {
		if optional {
			return nil
		}
		return p.errf("an undefined object type needs a `raw` value")
	}
	if fields[0] != "raw" {
		return p.errf("expected `raw`, got %q", fields[0])
	}
	b, err := p.hex(fields, SizeofObject-2)
	if err != nil {
		return err
	}
	copy(o.Data[:], b)
	return nil
}

// keywords turns "at 168 320 distance 5 ..." into a name -> values map. A field
// name is a token that is not a number; its values are the numbers after it.
func (p *textParser) keywords(fields []string) (map[string][]int, error) {
	kv := map[string][]int{}
	key := ""
	for _, tok := range fields {
		if n, err := parseInt(tok); err == nil && key != "" {
			kv[key] = append(kv[key], n)
			continue
		}
		if _, dup := kv[tok]; dup {
			return nil, p.errf("duplicate object field %q", tok)
		}
		key = tok
		kv[key] = nil
	}
	return kv, nil
}

// objectFields writes the keyword fields into the union. Every field of the
// group must be present: a partially specified object would silently take zeros
// for the rest, and a zero `distance` or `where` is a real, playable value.
//
// The field names come from Group.Fields, the same table the writer's order
// follows, so this only has to know how to spend the values it gets back.
func (p *textParser) objectFields(o *Object, kv map[string][]int) error {
	names := o.Group().Fields()
	if names == nil {
		return p.errf("0x%02x selects no union variant", uint16(o.What))
	}
	v := make([]int, 0, len(names)+3)
	for _, n := range names {
		got, ok := kv[n]
		if !ok {
			return p.errf("%s: missing field %q", ObjectName(o.What), n)
		}
		if arity := FieldArity(n); len(got) != arity {
			return p.errf("%s: field %q takes %d value(s), got %d",
				ObjectName(o.What), n, arity, len(got))
		}
		v = append(v, got...)
	}
	for k := range kv {
		if !contains(names, k) {
			return p.errf("%s: unknown field %q", ObjectName(o.What), k)
		}
	}

	switch o.Group() {
	case GroupBlower:
		o.SetBlower(Blower{TopLeft: Point{int16(v[0]), int16(v[1])}, Distance: int16(v[2]),
			Initial: byte(v[3]), State: byte(v[4]), Vector: byte(v[5]), Tall: byte(v[6])})
	case GroupFurniture:
		o.SetFurniture(Furniture{Bounds: Rect{int16(v[0]), int16(v[1]), int16(v[2]), int16(v[3])},
			Pict: int16(v[4])})
	case GroupBonus:
		o.SetBonus(Bonus{TopLeft: Point{int16(v[0]), int16(v[1])}, Length: int16(v[2]),
			Points: int16(v[3]), State: byte(v[4]), Initial: byte(v[5])})
	case GroupTransport:
		o.SetTransport(Transport{TopLeft: Point{int16(v[0]), int16(v[1])}, Tall: int16(v[2]),
			Where: int16(v[3]), Who: byte(v[4]), Wide: byte(v[5])})
	case GroupSwitch:
		o.SetSwitch(Switch{TopLeft: Point{int16(v[0]), int16(v[1])}, Delay: int16(v[2]),
			Where: int16(v[3]), Who: byte(v[4]), Type: byte(v[5])})
	case GroupLight:
		o.SetLight(Light{TopLeft: Point{int16(v[0]), int16(v[1])}, Length: int16(v[2]),
			Byte0: byte(v[3]), Byte1: byte(v[4]), Initial: byte(v[5]), State: byte(v[6])})
	case GroupAppliance:
		o.SetAppliance(Appliance{TopLeft: Point{int16(v[0]), int16(v[1])}, Height: int16(v[2]),
			Byte0: byte(v[3]), Delay: byte(v[4]), Initial: byte(v[5]), State: byte(v[6])})
	case GroupEnemy:
		o.SetEnemy(Enemy{TopLeft: Point{int16(v[0]), int16(v[1])}, Length: int16(v[2]),
			Delay: byte(v[3]), Byte0: byte(v[4]), Initial: byte(v[5]), State: byte(v[6])})
	case GroupClutter:
		o.SetClutter(Clutter{Bounds: Rect{int16(v[0]), int16(v[1]), int16(v[2]), int16(v[3])},
			Pict: int16(v[4])})
	}
	return nil
}

func contains(names []string, k string) bool {
	for _, n := range names {
		if n == k {
			return true
		}
	}
	return false
}

// finish fills in the derived fields and checks the whole-house invariants that
// only make sense once every room has been read.
func (p *textParser) finish() (*House, error) {
	if !p.sawFormat {
		return nil, fmt.Errorf("not a gliderGo house: no `format` directive")
	}
	h := p.h
	if len(h.Rooms) > 0x7FFF {
		return nil, fmt.Errorf("%d rooms exceeds the short nRooms field", len(h.Rooms))
	}
	h.NRooms = int16(len(h.Rooms))
	for i := range h.Rooms {
		r := &h.Rooms[i]
		if n, ok := p.explicitN[i]; ok {
			r.NumObjects = n
		} else {
			r.NumObjects = int16(r.LiveObjects())
		}
	}
	return h, nil
}

// ------------------------------------------------------------ scalar plumbing

func (p *textParser) int(toks []string, i int, what string) (int, error) {
	if i >= len(toks) {
		return 0, p.errf("missing %s", what)
	}
	n, err := parseInt(toks[i])
	if err != nil {
		return 0, p.errf("%s: %v", what, err)
	}
	return n, nil
}

// parseInt accepts decimal and 0x hex, with an optional sign. The only limit
// applied here is 32 bits: several fields are legitimately written as unsigned
// hex (flags, bounds), as signed decimal (coordinates) and as values above 2^31
// (Mac timestamps, which are ~2.9e9 for the mid-1990s dates the shipped houses
// carry) in the same file. Narrower range checks belong to the caller, which
// knows the field's width.
func parseInt(s string) (int, error) {
	neg := false
	t := s
	switch {
	case strings.HasPrefix(t, "-"):
		neg, t = true, t[1:]
	case strings.HasPrefix(t, "+"):
		t = t[1:]
	}
	base := 10
	if strings.HasPrefix(t, "0x") || strings.HasPrefix(t, "0X") {
		base, t = 16, t[2:]
	}
	if t == "" {
		return 0, fmt.Errorf("not a number: %q", s)
	}
	v, err := strconv.ParseUint(t, base, 64)
	if err != nil {
		return 0, fmt.Errorf("not a number: %q", s)
	}
	if v > 0xFFFFFFFF {
		return 0, fmt.Errorf("out of range for 32 bits: %q", s)
	}
	if neg {
		return -int(v), nil
	}
	return int(v), nil
}

func (p *textParser) i16(toks []string, dst *int16, what string) error {
	v, err := p.int(toks, 1, what)
	if err != nil {
		return err
	}
	if v < -0x8000 || v > 0xFFFF {
		return p.errf("%s: %d does not fit a short", what, v)
	}
	*dst = int16(uint16(v))
	if len(toks) > 2 {
		return p.errf("unexpected %q after %s", toks[2], what)
	}
	return nil
}

func (p *textParser) i32(toks []string, dst *int32, what string) error {
	v, err := p.int(toks, 1, what)
	if err != nil {
		return err
	}
	*dst = int32(uint32(v))
	if len(toks) > 2 {
		return p.errf("unexpected %q after %s", toks[2], what)
	}
	return nil
}

func (p *textParser) u8(toks []string, dst *byte, what string) error {
	v, err := p.int(toks, 1, what)
	if err != nil {
		return err
	}
	if v < 0 || v > 0xFF {
		return p.errf("%s: %d does not fit a byte", what, v)
	}
	*dst = byte(v)
	if len(toks) > 2 {
		return p.errf("unexpected %q after %s", toks[2], what)
	}
	return nil
}

func (p *textParser) point(toks []string, dst *Point) error {
	if len(toks) != 3 {
		return p.errf("`%s` needs two values, v then h", toks[0])
	}
	v, err := p.int(toks, 1, "v")
	if err != nil {
		return err
	}
	hh, err := p.int(toks, 2, "h")
	if err != nil {
		return err
	}
	*dst = Point{V: int16(v), H: int16(hh)}
	return nil
}

func (p *textParser) pstr(toks []string, dst []byte, what string) error {
	if len(toks) < 2 {
		return p.errf("missing %s", what)
	}
	s, err := UnquoteMacRoman(toks[1])
	if err != nil {
		return p.errf("%s: %v", what, err)
	}
	if !pstrSet(dst, s) {
		return p.errf("%s: %q does not fit %d bytes, or contains characters "+
			"Mac OS Roman cannot represent", what, s, len(dst)-1)
	}
	return nil
}

// residue writes bytes past a Pascal string's length. It refuses to write into
// the declared text, so a residue line cannot silently corrupt a name.
func (p *textParser) residue(toks []string, dst []byte) error {
	n := int(dst[0])
	if n > len(dst)-1 {
		n = len(dst) - 1
	}
	b, err := p.hex(toks, len(dst)-1-n)
	if err != nil {
		return err
	}
	copy(dst[1+n:], b)
	return nil
}

// hex reads `key aa bb cc ...` and requires exactly want bytes.
func (p *textParser) hex(toks []string, want int) ([]byte, error) {
	got := toks[1:]
	if len(got) != want {
		return nil, p.errf("%s: expected %d hex bytes, got %d", toks[0], want, len(got))
	}
	b := make([]byte, want)
	for i, tok := range got {
		v, err := strconv.ParseUint(tok, 16, 8)
		if err != nil {
			return nil, p.errf("%s: %q is not a hex byte", toks[0], tok)
		}
		b[i] = byte(v)
	}
	return b, nil
}

// tokenize splits a line on whitespace, keeping "quoted strings" whole and
// discarding a `#` comment. A `#` inside a quoted string is literal.
func tokenize(line string) ([]string, error) {
	var toks []string
	i := 0
	for i < len(line) {
		c := line[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case c == '#':
			return toks, nil
		case c == '"':
			j := i + 1
			for j < len(line) {
				if line[j] == '\\' {
					j += 2
					continue
				}
				if line[j] == '"' {
					break
				}
				j++
			}
			if j >= len(line) {
				return nil, fmt.Errorf("unterminated quoted string")
			}
			toks = append(toks, line[i:j+1])
			i = j + 1
		default:
			j := i
			for j < len(line) && line[j] != ' ' && line[j] != '\t' &&
				line[j] != '\r' && line[j] != '#' {
				j++
			}
			toks = append(toks, line[i:j])
			i = j
		}
	}
	return toks, nil
}
