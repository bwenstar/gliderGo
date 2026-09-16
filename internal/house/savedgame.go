package house

// The saved game: 1.10's file format, and the 40-byte block inside a house that it grew out
// of.
//
// Everything here is a reconstruction, and it is worth being blunt about why. The 1994
// source has two saved-game formats and not one working path to either:
//
//   - `gameType`, 40 bytes, lives *inside* the house file at offset 820 with a `hasGame`
//     flag at 860. This package has parsed it since 1.2, and two shipped houses carry a
//     live one (Titanic: room 104, score 4700, two gliders). The writer, `SaveGame`
//     (SavedGames.c:150-166 in effect), is fully implemented -- and its only call site is
//     commented out at Menu.c:458.
//   - `game2Type`, a separate file of type 'gliG' and creator 'ozm5', is the same 40 bytes
//     with `unusedShort` renamed `nRooms`, preceded by an `FSSpec` naming the house, and
//     followed by one 292-byte `savedRoom` per room. Its writer `SaveGame2` has its entire
//     body commented out (SavedGames.c:33-147; the one live statement is the comment
//     `// Add NavServices later.`), and its reader `OpenSavedGame` begins with
//     `return false;		// TEMP fix this iwth NavServices` (:169) above 130 more commented
//     lines. `QueryResumeGame` (Menu.c:710-758) is never called.
//
// So `smallGame` was never filled, `NewGame(kResumeGameMode)` was never reached, and no
// player of Glider PRO 1.1.2 ever saved a game. What survives is a complete and unambiguous
// *description* of what the author intended, in code that would have worked: the field
// filling, the four validation gates and their alerts, and the restore half -- which is
// live, because it only reads `smallGame` (see internal/game's resume arm, ported in 1.2).
// docs/analysis/scoring.md 9 has the offset tables and the pseudocode; this file implements
// them, and states the four places it had to decide something the dead code did not.
//
// Those four are numbered S1..S4 and this comment is their normative home -- the S is not
// decoration. docs/analysis/format-decisions.md already owns a D1..D7 about the *original's*
// ambiguities, and two numberings of unrelated things sharing one prefix is how a reader ends
// up reading the wrong ruling. S1 happens to be format-decisions.md's D1 and says so; S2..S4
// have no counterpart there, because they are decisions about what *this port writes* rather
// than about what 1994 meant.
//
// S1 (= format-decisions.md D1): the container is 110 bytes, not the header's 114.
// `sizeof(game2Type)` in a real compiler is 110 -- FSSpec is
// `{short vRefNum; long parID; Str63 name}` = 2+4+64 = 70, plus 40 -- and the 114 in the
// comment at GliderStructs.h counts padding no Mac compiler inserted. That decision was
// settled before 1.10 existed; this file only depends on it.
//
// S2: the FSSpec's first six bytes are replaced. A volume reference number and a directory
// ID identify a 1994 filesystem and mean nothing anywhere else, and writing zeros there
// would leave a file that says nothing about itself. In their place go a four-byte magic
// "gliG" -- the original's own type code, which is what a `file(1)` magic entry would look
// for -- and a two-byte container version. Everything from offset 6 on is `game2Type`
// exactly: the Str63 house name at 6, the 40-byte `gameType` at 70, `savedRoom[]` at 110.
// A 1994 build with `OpenSavedGame` uncommented would read one of these files correctly
// except for an FSSpec it would have had to re-resolve by name anyway.
//
// S3: both saved-game versions are accepted. `kSavedGameVersion` is 0x0200 and
// `OpenSavedGame` rejects anything else, but *both* saved games shipped inside the corpus
// hold 0x0100 (docs/analysis/scoring.md 9.5) -- so honouring that gate would mean Titanic's
// save could never be resumed, which is the one thing 1.10 is specified to do. The layout
// is identical either way; the field records which writer produced the record, and nothing
// downstream branches on it.
//
// S4: a house's own embedded block is resumed with the *house's* timeStamp, not the one the
// block carries. This is the decision that would otherwise make the feature unreachable, and
// the argument for it is long enough to belong next to the code that does it -- see
// EmbeddedGame.
//
// What is *not* here: this package never writes a saved game into a house file. Writing
// player state back into a house the port did not author is the thing 1.7c ruled out for
// high scores, for the same reason -- the vendored houses stay byte-identical and a house
// from anywhere stays playable. The embedded block is read, and honoured, and left alone.
// internal/saved owns the side-car; internal/game owns what the fields mean.

import (
	"errors"
	"fmt"
)

// Sizes and identifiers of the saved-game format.
const (
	// SizeofSavedRoom is sizeof(savedRoom): two bytes of padding the original never
	// initialised, the `visited` flag, and all 24 object slots.
	SizeofSavedRoom = 292

	// SizeofSavedGameHeader is everything before the first room: magic, container
	// version, the Str63 house name and the 40-byte gameType. This is S1's 110.
	SizeofSavedGameHeader = 110

	// SavedGameMagic replaces the FSSpec's vRefNum and parID. It is the original's own
	// file type code, so a magic-file entry can name the format after what made it.
	SavedGameMagic = "gliG"

	// SavedGameFormat is the container version -- this file's own numbering, not the
	// original's. 1 is the layout described above. A reader that meets a larger number
	// should refuse the file rather than guess.
	SavedGameFormat = 1

	// SavedGameVersion is kSavedGameVersion, the version SaveGame2 would have written
	// and OpenSavedGame would have demanded, and SavedGameVersion1 is what both shipped
	// records actually hold. See S3.
	SavedGameVersion  = 0x0200
	SavedGameVersion1 = 0x0100
)

// Offsets inside an encoded saved game, so a test can assert them rather than trusting
// that the writer below and the offset table in docs/analysis/scoring.md 9.5 agree.
const (
	offSavedMagic     = 0
	offSavedFormat    = 4
	offSavedHouseName = 6   // FSSpec.name: the reason the first six bytes are ours to spend
	offSavedGameBlock = 70  // gameType
	offSavedRooms     = 110 // savedRoom[]
)

// ErrNotSavedGame is returned for a file that is not a saved game at all, as distinct from
// one that is damaged: the difference decides whether a caller reports "this is not a save"
// or "your save is broken", and those want different words in front of a player.
var ErrNotSavedGame = errors.New("house: not a saved game")

// SavedRoom is savedRoom: the mutable half of a room, which is the object bytes plus
// whether the player has been there.
//
// Twenty-four slots always, holes included, because a link addresses an object by slot index
// and nothing may be renumbered (see Room). The two dead fields are carried for the same
// reason House carries its unused ones: a round-trip that drops bytes is not a round-trip.
// In a file this port wrote they are zero; in one a 1994 build wrote they would have been
// whatever the heap held, because SaveGame2 only ever set `visited` and the objects.
type SavedRoom struct {
	UnusedShort int16
	UnusedByte  byte
	Visited     byte
	Objects     [MaxRoomObs]Object
}

// SavedGame is game2Type with S2's header: which house, the 40-byte game state, and one
// SavedRoom per room of that house.
//
// Format is the container version and is *not* Game.Version -- the two numbers count
// different things, which is exactly why the original's single `version` field could not
// tell a bad file from an old one.
type SavedGame struct {
	Format    int16
	HouseName PStr64
	Game      Game
	Rooms     []SavedRoom
}

// Size is the exact number of bytes Encode will produce.
func (sg *SavedGame) Size() int {
	return SizeofSavedGameHeader + len(sg.Rooms)*SizeofSavedRoom
}

// Encode serialises a saved game.
//
// `nRooms` is written from len(Rooms), never from the Game struct's UnusedShort, so the
// count in the file and the rooms in the file cannot disagree -- which is the whole reason
// `game2Type` renamed that field. The caller's own value there is overwritten rather than
// checked, because there is no reading under which it would be more authoritative than the
// slice.
func (sg *SavedGame) Encode() ([]byte, error) {
	if len(sg.Rooms) > 0x7FFF {
		return nil, fmt.Errorf("house: %d rooms exceeds the short nRooms field", len(sg.Rooms))
	}
	g := sg.Game
	g.UnusedShort = int16(len(sg.Rooms))

	e := &encoder{b: make([]byte, sg.Size())}
	e.bytes([]byte(SavedGameMagic))
	e.i16(sg.Format)
	e.bytes(sg.HouseName[:])
	e.game(&g)
	for i := range sg.Rooms {
		e.savedRoom(&sg.Rooms[i])
	}
	if e.off != len(e.b) {
		// Unreachable: Size and the writers below describe one layout, and the
		// round-trip test would fail first. Checked anyway -- a short write here yields a
		// file that decodes to a different game.
		panic(fmt.Sprintf("house: internal: SavedGame.Encode wrote %d of %d bytes",
			e.off, len(e.b)))
	}
	return e.b, nil
}

// DecodeSavedGame parses a saved game.
//
// Strict about structure, permissive about content, which is the split Load draws and for
// the same reason: a file whose length does not match its own room count has been truncated
// or concatenated and cannot be interpreted, whereas a negative score or a room number past
// the end of the house is a *claim* to be checked against a particular house -- and that
// check needs the house, so it belongs to the caller (internal/saved.Check), not here.
func DecodeSavedGame(b []byte) (*SavedGame, error) {
	if len(b) < SizeofSavedGameHeader {
		return nil, fmt.Errorf("%w: %d bytes, less than the %d-byte header",
			ErrNotSavedGame, len(b), SizeofSavedGameHeader)
	}
	if string(b[offSavedMagic:offSavedMagic+len(SavedGameMagic)]) != SavedGameMagic {
		return nil, fmt.Errorf("%w: it does not begin with %q", ErrNotSavedGame, SavedGameMagic)
	}

	sg := &SavedGame{}
	d := &decoder{b: b, off: offSavedFormat}
	sg.Format = d.i16()
	if sg.Format != SavedGameFormat {
		// Refused, not guessed. A newer gliderGo may add fields, and reading them as the
		// current layout would restore a game that is not the one that was saved.
		return nil, fmt.Errorf("house: saved-game container version %d; this build reads %d",
			sg.Format, SavedGameFormat)
	}
	d.bytes(sg.HouseName[:])
	d.game(&sg.Game)

	// nRooms is the file's own claim; the length is the fact. Cross-check them rather than
	// trusting either, because the original's reader trusted the claim and would have
	// walked off the end of a short file exactly as ReadScoresFromDisk did (see
	// DecodeScores' note).
	n := int(sg.Game.UnusedShort)
	body := len(b) - SizeofSavedGameHeader
	switch {
	case n < 0:
		return nil, fmt.Errorf("house: saved game claims %d rooms", n)
	case body != n*SizeofSavedRoom:
		return nil, fmt.Errorf("house: saved game claims %d rooms, which is %d bytes, "+
			"but holds %d", n, n*SizeofSavedRoom, body)
	}

	sg.Rooms = make([]SavedRoom, n)
	for i := range sg.Rooms {
		d.savedRoom(&sg.Rooms[i])
	}
	if d.off != len(b) { // unreachable, as above
		return nil, fmt.Errorf("house: internal: DecodeSavedGame read %d of %d bytes",
			d.off, len(b))
	}
	return sg, nil
}

func (e *encoder) savedRoom(r *SavedRoom) {
	e.i16(r.UnusedShort)
	e.u8(r.UnusedByte)
	e.u8(r.Visited)
	for i := range r.Objects {
		e.i16(r.Objects[i].What)
		e.bytes(r.Objects[i].Data[:])
	}
}

func (d *decoder) savedRoom(r *SavedRoom) {
	r.UnusedShort = d.i16()
	r.UnusedByte = d.u8()
	r.Visited = d.u8()
	for i := range r.Objects {
		r.Objects[i].What = d.i16()
		d.bytes(r.Objects[i].Data[:])
	}
}

// ------------------------------------------------- the 40-byte block, alone

// EncodeGame serialises one gameType as the same SizeofGame big-endian bytes it occupies
// inside a house header at offset 820.
//
// The pair to this and DecodeGame exist for the same reason EncodeScores and DecodeScores
// do: the block has a life outside the file it is embedded in, and there must be exactly one
// description of its layout for the two to share. Every byte comes from the struct, unused
// longs included, so a block read out of a shipped house encodes back to the bytes it came
// from -- which is what lets the tests compare bytes rather than fields.
func EncodeGame(g *Game) []byte {
	e := &encoder{b: make([]byte, SizeofGame)}
	e.game(g)
	if e.off != len(e.b) {
		panic(fmt.Sprintf("house: internal: EncodeGame wrote %d of %d bytes", e.off, len(e.b)))
	}
	return e.b
}

// DecodeGame parses a gameType from exactly SizeofGame bytes.
func DecodeGame(b []byte) (Game, error) {
	var g Game
	if len(b) != SizeofGame {
		return g, fmt.Errorf("house: saved-game block is %d bytes, want %d", len(b), SizeofGame)
	}
	d := &decoder{b: b}
	d.game(&g)
	if d.off != SizeofGame { // unreachable, as above
		return g, fmt.Errorf("house: internal: DecodeGame read %d of %d bytes", d.off, SizeofGame)
	}
	return g, nil
}

// EmbeddedGame is the game a house carries inside itself, as a SavedGame a resume can take:
// `hasGame` and the 40 bytes at offset 820, which is what SaveGame(true) would have written
// and what two shipped houses hold (Titanic, room 104, score 4700, two gliders;
// ImagineHouse PRO II, room 45, score 5900, five).
//
// The room slice stays nil, and that is the point rather than an omission. A house's own
// block has no room snapshot -- SaveGame writes 40 bytes into the header and nothing else --
// so the objects a resumed game starts from are the ones the house file itself holds, which
// is exactly what kResumeGameMode would have done in 1994. game.ResumeSavedGame reads nil
// that way and internal/saved.Check skips the room-count gate for it; a file always decodes
// to a non-nil slice, so the two cases cannot be confused.
//
// **The timestamp is the house's and not the block's.** SaveGame stamps the game with
// `GetDateTime(&stamp)` (SavedGames.c:320) -- the moment it was written -- while the gate
// that reads it back compares the *house's* timeStamp (OpenSavedGame's second gate). The
// two can never be equal, so the original's own validation would have rejected every game
// its own SaveGame wrote. Nothing noticed, because nothing ever read one back. Taking the
// house's stamp here is the only reading under which the gate means what it was written to
// mean: this game came out of this very file, so the file cannot have changed since.
//
// Callers with only a header can pass Summary's TimeStamp and Game; callers with a loaded
// House pass its own two fields. The three arguments are separate rather than a *House for
// that reason -- the menu asks this question of a file it has only peeked at.
func EmbeddedGame(houseName string, timeStamp int32, g Game) *SavedGame {
	g.TimeStamp = timeStamp
	sg := &SavedGame{Format: SavedGameFormat, Game: g}
	sg.HouseName.SetText(houseName)
	return sg
}

// ------------------------------------------------- house <-> saved rooms

// CaptureSavedRooms copies the mutable half of every room out of a loaded house.
//
// This is SaveGame2's inner loop (SavedGames.c:100-113 in the commented body), and what it
// captures is exactly the state the game mutates in place: object bytes, because a switch
// that has been thrown or a band that has been taken is recorded in the object's own union,
// and `visited`, because the star count and the room's first-entry behaviour depend on it.
// Nothing else in a room can change while a game is played.
func (h *House) CaptureSavedRooms() []SavedRoom {
	out := make([]SavedRoom, len(h.Rooms))
	for i := range h.Rooms {
		out[i].Visited = h.Rooms[i].Visited
		out[i].Objects = h.Rooms[i].Objects
	}
	return out
}

// ApplySavedRooms is the inverse, and it is `OpenSavedGame`'s success arm: for each room,
// `visited` and all kMaxRoomObs objects are copied in, and every other field is left as the
// house file has it.
//
// The room count has to match. That is the fourth of the original's four validation gates
// (`savedGame->nRooms != thisHousePtr->nRooms` -> `kYellowSavedRoomsWrong`), and it is the
// one that would corrupt a game rather than merely start the wrong one: applying a
// 40-room save to a 60-room house would leave twenty rooms holding another house's
// switches. internal/saved.Check reports it before anything is applied; this is the
// backstop.
func (h *House) ApplySavedRooms(rooms []SavedRoom) error {
	if len(rooms) != len(h.Rooms) {
		return fmt.Errorf("house: the saved game has %d rooms and the house has %d",
			len(rooms), len(h.Rooms))
	}
	for i := range rooms {
		h.Rooms[i].Visited = rooms[i].Visited
		h.Rooms[i].Objects = rooms[i].Objects
	}
	return nil
}
