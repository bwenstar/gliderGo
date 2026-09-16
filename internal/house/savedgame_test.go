package house

// The saved-game format. Three things are worth pinning here and nothing else is:
//
//   - the offsets, against docs/analysis/scoring.md 9.5's table, because the whole point of
//     S2 is that a gliderGo save is `game2Type` from offset 6 on and a table that drifted
//     from the writer would make that claim false without breaking anything;
//   - the round trip, because a save that does not decode to the game that was saved is
//     worse than no save at all;
//   - the refusals, because this file comes out of the player's own data directory, which is
//     somewhere a truncated write, a half-restored backup and a hex editor can all reach.
//
// The corpus test at the bottom is the one that says the reconstruction is right: the
// 40-byte block is decoded from all 22 shipped houses and re-encoded to the same bytes.

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// titanicStamp is the timeStamp in Titanic.house's live saved game. It is worth spelling out
// rather than writing the negative: a Mac epoch second in 1995 has bit 31 set, so a signed
// `long timeStamp` was already wrapping when the file was written, and every comparison the
// port makes against it has to be an equality rather than an ordering.
var titanicStamp = func() int32 { var u uint32 = 2880300087; return int32(u) }()

// A saved game with distinguishable values everywhere, so a transposed field shows up as a
// wrong number rather than as another zero.
func sampleSavedGame(rooms int) *SavedGame {
	sg := &SavedGame{
		Format: SavedGameFormat,
		Game: Game{
			Version:      SavedGameVersion,
			WasStarsLeft: 7,
			TimeStamp:    titanicStamp,
			Where:        Point{V: 85, H: 240},
			Score:        4700,
			UnusedLong:   0x11223344,
			UnusedLong2:  0x55667788,
			Energy:       -150, // ImagineHouse PRO II's, i.e. genuinely negative
			Bands:        3,
			RoomNumber:   104,
			GliderState:  2,
			NumGliders:   5,
			Foil:         1,
			UnusedShort:  -999, // overwritten with the room count; see Encode
			Facing:       1,
			ShowFoil:     1,
		},
		Rooms: make([]SavedRoom, rooms),
	}
	sg.HouseName.SetText("Titanic")
	for r := range sg.Rooms {
		sg.Rooms[r].Visited = byte(r % 2)
		for i := range sg.Rooms[r].Objects {
			sg.Rooms[r].Objects[i].What = int16(r*100 + i)
			for j := range sg.Rooms[r].Objects[i].Data {
				sg.Rooms[r].Objects[i].Data[j] = byte(r + i + j)
			}
		}
	}
	return sg
}

// The layout, byte by byte, against the offsets docs/analysis/scoring.md 9.5 records for
// game2Type. Everything from offSavedHouseName on has to be at the offset the original's
// struct put it at -- that is S2's entire claim.
func TestEncodedLayoutMatchesGame2Type(t *testing.T) {
	sg := sampleSavedGame(3)
	b, err := sg.Encode()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(b), SizeofSavedGameHeader+3*SizeofSavedRoom; got != want {
		t.Errorf("Encode produced %d bytes, want %d", got, want)
	}
	if got := string(b[offSavedMagic:offSavedFormat]); got != SavedGameMagic {
		t.Errorf("magic = %q, want %q", got, SavedGameMagic)
	}
	if got := be16(b[offSavedFormat:]); got != SavedGameFormat {
		t.Errorf("container version = %d, want %d", got, SavedGameFormat)
	}
	if !bytes.Equal(b[offSavedHouseName:offSavedHouseName+64], sg.HouseName[:]) {
		t.Error("the house name is not the Str63 at offset 6")
	}

	// The 40-byte block, at 70, is the same bytes EncodeGame produces -- with nRooms
	// substituted at 106. If those two ever disagree the port has two descriptions of
	// gameType, which is the thing this package exists to avoid.
	want := sg.Game
	want.UnusedShort = 3
	if got := b[offSavedGameBlock : offSavedGameBlock+SizeofGame]; !bytes.Equal(got, EncodeGame(&want)) {
		t.Error("the 40 bytes at offset 70 are not the gameType block")
	}
	if got := be16(b[offSavedGameBlock+36:]); got != 3 {
		t.Errorf("nRooms at absolute offset %d = %d, want 3", offSavedGameBlock+36, got)
	}
	if offSavedRooms != SizeofSavedGameHeader {
		t.Errorf("the first room is at %d but the header is %d bytes",
			offSavedRooms, SizeofSavedGameHeader)
	}

	// One field of one room, positioned by hand: `visited` is the fourth byte of a
	// savedRoom and room 1's is 1, so room 1's objects begin three bytes later.
	if got := b[offSavedRooms+SizeofSavedRoom+3]; got != 1 {
		t.Errorf("room 1's visited byte = %d, want 1", got)
	}
	if got := be16(b[offSavedRooms+SizeofSavedRoom+4:]); got != 100 {
		t.Errorf("room 1's object 0 `what` = %d, want 100", got)
	}
}

func TestSavedGameRoundTrips(t *testing.T) {
	for _, rooms := range []int{0, 1, 3, 58} {
		sg := sampleSavedGame(rooms)
		b, err := sg.Encode()
		if err != nil {
			t.Fatalf("%d rooms: %v", rooms, err)
		}
		got, err := DecodeSavedGame(b)
		if err != nil {
			t.Fatalf("%d rooms: %v", rooms, err)
		}

		// Encode overwrites UnusedShort with the room count on the way out, so the
		// expectation has to as well: that field is the room count and the -999 the sample
		// puts there is deliberately not preserved.
		want := *sg
		want.Game.UnusedShort = int16(rooms)
		if got.Format != want.Format || got.HouseName != want.HouseName || got.Game != want.Game {
			t.Errorf("%d rooms: header round-tripped to %+v, want %+v",
				rooms, got.Game, want.Game)
		}
		if len(got.Rooms) != rooms {
			t.Fatalf("%d rooms: decoded %d", rooms, len(got.Rooms))
		}
		for r := range got.Rooms {
			if got.Rooms[r] != want.Rooms[r] {
				t.Errorf("%d rooms: room %d round-tripped wrong", rooms, r)
			}
		}

		// And the bytes, so that re-encoding a decoded save is a fixed point. A save this
		// port writes and reads and writes again must not drift, or a player who resumes
		// and re-saves loses something a byte at a time.
		again, err := got.Encode()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, again) {
			t.Errorf("%d rooms: re-encoding a decoded save changed the bytes", rooms)
		}
	}
}

// Everything a damaged or foreign file can be. Each has to be refused with a legible
// reason, and the two that are "this is not a save at all" have to be distinguishable from
// the rest, because a player who opened the wrong file wants different words than one whose
// save is broken.
func TestDecodeSavedGameRefusals(t *testing.T) {
	good, err := sampleSavedGame(2).Encode()
	if err != nil {
		t.Fatal(err)
	}

	mutate := func(f func([]byte)) []byte {
		b := append([]byte(nil), good...)
		f(b)
		return b
	}

	cases := []struct {
		name     string
		in       []byte
		notASave bool
		want     string
	}{
		{name: "empty", in: nil, notASave: true, want: "less than"},
		{name: "header only, truncated", in: good[:SizeofSavedGameHeader-1],
			notASave: true, want: "less than"},
		{name: "wrong magic", in: mutate(func(b []byte) { copy(b, "gliS") }),
			notASave: true, want: "does not begin with"},
		{name: "a house file", in: make([]byte, SizeofHouseHeader),
			notASave: true, want: "does not begin with"},
		{name: "a newer container", in: mutate(func(b []byte) { putBE16(b[offSavedFormat:], 99) }),
			want: "container version 99"},
		{name: "truncated body", in: good[:len(good)-1], want: "claims 2 rooms"},
		{name: "extra bytes", in: append(append([]byte(nil), good...), 0), want: "claims 2 rooms"},
		{name: "nRooms lies high",
			in:   mutate(func(b []byte) { putBE16(b[offSavedGameBlock+36:], 3) }),
			want: "claims 3 rooms"},
		{name: "nRooms is negative",
			in:   mutate(func(b []byte) { putBE16(b[offSavedGameBlock+36:], -1) }),
			want: "claims -1 rooms"},
	}
	for _, tc := range cases {
		got, err := DecodeSavedGame(tc.in)
		if err == nil {
			t.Errorf("%s: decoded to %+v, want an error", tc.name, got)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want it to mention %q", tc.name, err, tc.want)
		}
		if is := errors.Is(err, ErrNotSavedGame); is != tc.notASave {
			t.Errorf("%s: errors.Is(err, ErrNotSavedGame) = %v, want %v; %v",
				tc.name, is, tc.notASave, err)
		}
	}
}

// A save is 110 bytes plus 292 per room and nothing else -- no alignment, no padding, no
// version-dependent size. S1's numbers, asserted directly, because they were the one thing
// the original's own header got wrong (it says 114).
func TestSavedGameSizeIsD1(t *testing.T) {
	if SizeofSavedGameHeader != 110 {
		t.Errorf("SizeofSavedGameHeader = %d; savedgame.go's S1 -- format-decisions.md D1 -- says 110",
			SizeofSavedGameHeader)
	}
	if SizeofSavedRoom != 4+MaxRoomObs*SizeofObject {
		t.Errorf("SizeofSavedRoom = %d, but 4 + %d*%d = %d",
			SizeofSavedRoom, MaxRoomObs, SizeofObject, 4+MaxRoomObs*SizeofObject)
	}
	if got, want := len(SavedGameMagic)+2+64+SizeofGame, SizeofSavedGameHeader; got != want {
		t.Errorf("the header's parts sum to %d, not %d", got, want)
	}
	for _, rooms := range []int{0, 1, 58, 200} {
		sg := sampleSavedGame(rooms)
		b, err := sg.Encode()
		if err != nil {
			t.Fatal(err)
		}
		if len(b) != sg.Size() || len(b) != 110+292*rooms {
			t.Errorf("%d rooms: %d bytes, Size() = %d, want %d",
				rooms, len(b), sg.Size(), 110+292*rooms)
		}
	}
}

// ------------------------------------------------- the 40-byte block

func TestGameBlockRoundTrips(t *testing.T) {
	want := sampleSavedGame(0).Game
	b := EncodeGame(&want)
	if len(b) != SizeofGame {
		t.Fatalf("EncodeGame produced %d bytes, want %d", len(b), SizeofGame)
	}
	got, err := DecodeGame(b)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("round-tripped to %+v, want %+v", got, want)
	}
	for _, n := range []int{0, SizeofGame - 1, SizeofGame + 1} {
		if _, err := DecodeGame(make([]byte, n)); err == nil {
			t.Errorf("DecodeGame accepted %d bytes", n)
		}
	}
}

// The block this port writes and the block the original wrote are the same 40 bytes. Decoded
// out of each shipped house at offset 820 and re-encoded: any difference means the field
// order or a width is wrong, and the two live records in the corpus would then resume into
// the wrong room.
func TestGameCodecMatchesEveryShippedHeader(t *testing.T) {
	live := 0
	for _, e := range loadCorpus(t) {
		want := e.raw[offSavedGame : offSavedGame+SizeofGame]
		g, err := DecodeGame(want)
		if err != nil {
			t.Fatalf("%s: %v", e.stem, err)
		}
		if got := EncodeGame(&g); !bytes.Equal(got, want) {
			t.Errorf("%s: the block at +%d does not re-encode to itself\n got %x\nwant %x",
				e.stem, offSavedGame, got, want)
		}
		if g != e.house.SavedGame {
			t.Errorf("%s: DecodeGame and Load disagree about the block", e.stem)
		}
		if e.house.HasGame != 0 {
			live++
			// Both of them: S3's evidence. If a corpus ever shows up with 0x0200 this
			// still passes -- what would not is a *third* value, which would mean the
			// field is not a version at all.
			switch g.Version {
			case SavedGameVersion, SavedGameVersion1:
			default:
				t.Errorf("%s carries a live save of version %#04x, which is neither "+
					"kSavedGameVersion nor the 0x0100 both shipped saves hold",
					e.stem, g.Version)
			}
			if g.RoomNumber < 0 || int(g.RoomNumber) >= len(e.house.Rooms) {
				t.Errorf("%s's live save names room %d of %d",
					e.stem, g.RoomNumber, len(e.house.Rooms))
			}
		}
	}
	if live != 2 {
		t.Errorf("%d shipped houses set hasGame; docs/analysis/scoring.md 9.5 says 2", live)
	}
}

// EmbeddedGame turns a house's own 40 bytes into something a resume can take, and it makes
// exactly two changes to them. Both matter and neither is obvious.
func TestEmbeddedGameSubstitutesTheHousesStamp(t *testing.T) {
	g := sampleSavedGame(4).Game
	g.TimeStamp = 0x11111111 // the clock, as SaveGame wrote it (SavedGames.c:320)

	sg := EmbeddedGame("Titanic", 0x2A3B4C5D, g)

	if sg.Format != SavedGameFormat {
		t.Errorf("Format is %d, want %d", sg.Format, SavedGameFormat)
	}
	if got := sg.HouseName.Text(); got != "Titanic" {
		// The name is the one thing the block has no room for -- houseType has no name field
		// -- so a resume of it cannot get past gate 1 without this.
		t.Errorf("HouseName is %q, want %q", got, "Titanic")
	}
	if sg.Game.TimeStamp != 0x2A3B4C5D {
		t.Errorf("TimeStamp is %#x, want the house's %#x: SaveGame stamped the block with the "+
			"clock while the gate that reads it back compares the house's stamp, so the "+
			"original's own validation would refuse every game its own SaveGame wrote",
			sg.Game.TimeStamp, 0x2A3B4C5D)
	}
	if sg.Rooms != nil {
		t.Error("EmbeddedGame carried a room snapshot; a houseType has none, and the nil is " +
			"what tells the room-count gate there is nothing to compare")
	}

	// Everything else is the house's bytes, untouched.
	want := g
	want.TimeStamp = 0x2A3B4C5D
	if sg.Game != want {
		t.Errorf("the block changed:\n got %+v\nwant %+v", sg.Game, want)
	}
}

// The two shipped houses that carry a live game, resumed as far as this package can take them:
// the block names a room the house has and the whole thing round-trips through a file. This is
// the corpus half of the claim in docs/PLAN.md 1.10 -- that Titanic's shipped save can be
// resumed -- and it is here rather than in internal/saved because that package has no corpus.
func TestEveryLiveShippedGameIsResumable(t *testing.T) {
	live := 0
	for _, e := range loadCorpus(t) {
		if e.house.HasGame == 0 {
			continue
		}
		live++
		sg := EmbeddedGame(e.stem, e.house.TimeStamp, e.house.SavedGame)
		if int(sg.Game.RoomNumber) >= len(e.house.Rooms) {
			t.Errorf("%s: room %d of %d", e.stem, sg.Game.RoomNumber, len(e.house.Rooms))
		}

		// Round-tripped through the file format, because that is what the store will do with
		// it the first time the player saves the game they resumed. The room slice does not
		// survive -- there was none -- and Encode writes the count over UnusedShort, so the
		// comparison has to allow for both.
		b, err := sg.Encode()
		if err != nil {
			t.Fatalf("%s: %v", e.stem, err)
		}
		got, err := DecodeSavedGame(b)
		if err != nil {
			t.Fatalf("%s: %v", e.stem, err)
		}
		want := sg.Game
		want.UnusedShort = 0
		if got.Game != want || got.HouseName != sg.HouseName {
			t.Errorf("%s: the embedded game did not survive a file", e.stem)
		}
		if len(got.Rooms) != 0 {
			t.Errorf("%s: %d rooms came back out of a save that had none", e.stem, len(got.Rooms))
		}
	}
	if live != 2 {
		t.Errorf("%d shipped houses carry a live game; docs/analysis/scoring.md 9.5 says 2", live)
	}
}

// ------------------------------------------------- house <-> saved rooms

func TestCaptureAndApplySavedRooms(t *testing.T) {
	h := &House{Rooms: make([]Room, 4)}
	for r := range h.Rooms {
		h.Rooms[r].Visited = byte(r % 2)
		h.Rooms[r].Name.SetText("room")
		h.Rooms[r].Background = int16(1000 + r)
		for i := range h.Rooms[r].Objects {
			h.Rooms[r].Objects[i].What = int16(r*10 + i)
		}
	}
	snap := h.CaptureSavedRooms()

	// Play a bit: throw a switch, visit a room, and -- the case that matters -- change a
	// field a save must *not* restore.
	h.Rooms[0].Objects[3].Data[0] = 0xFF
	h.Rooms[2].Visited = 1
	h.Rooms[1].Background = 4242

	if err := h.ApplySavedRooms(snap); err != nil {
		t.Fatal(err)
	}
	if h.Rooms[0].Objects[3].Data[0] != 0 {
		t.Error("the object byte was not restored")
	}
	if h.Rooms[2].Visited != 0 {
		t.Error("visited was not restored")
	}
	if h.Rooms[1].Background != 4242 {
		t.Error("Background was restored; a saved game holds no room layout")
	}

	// The room-count gate: the original's fourth validation, and the one that would
	// otherwise leave rooms holding another house's switches.
	for _, n := range []int{3, 5} {
		if err := h.ApplySavedRooms(make([]SavedRoom, n)); err == nil {
			t.Errorf("applied %d saved rooms to a 4-room house", n)
		}
	}
}

// The captured half is the mutable half: what a game can change and nothing else. Checked by
// mutating every byte of a room and seeing which of them survive a capture-and-apply, which
// is a stronger statement than the field list in CaptureSavedRooms' comment.
func TestCaptureCoversEverythingAGameCanChange(t *testing.T) {
	h := &House{Rooms: make([]Room, 1)}
	snap := h.CaptureSavedRooms()

	// Objects and visited: restored.
	for i := range h.Rooms[0].Objects {
		h.Rooms[0].Objects[i].What = 99
		for j := range h.Rooms[0].Objects[i].Data {
			h.Rooms[0].Objects[i].Data[j] = 0xAA
		}
	}
	h.Rooms[0].Visited = 1
	if err := h.ApplySavedRooms(snap); err != nil {
		t.Fatal(err)
	}
	if h.Rooms[0] != (Room{}) {
		t.Error("a capture-and-apply did not restore every object byte and visited")
	}
}
