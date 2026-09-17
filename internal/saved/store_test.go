package saved

// The saved-game store: where a file goes, what happens to a file that is not what it
// should be, and the five gates between a save and a game.
//
// The refusal half is the larger half here for the same reason it is in internal/scores --
// this file comes out of the player's own data directory, where a full disk, an interrupted
// write and a restored backup can all reach it -- but the stakes are different. A half-right
// score board is still a score board; half a saved game would put the player in the wrong
// room with the wrong inventory and no way to tell that anything had happened. So nothing
// here repairs anything: a save is resumable or it is refused with a reason.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/datadir"
	"github.com/bwenstar/gliderGo/internal/house"
)

// A saved game with a distinguishable value in every field, so that a transposition shows up
// as a wrong number rather than as another zero.
func sample(name string, rooms int) *house.SavedGame {
	sg := &house.SavedGame{
		Format: house.SavedGameFormat,
		Game: house.Game{
			Version:      house.SavedGameVersion,
			WasStarsLeft: 4,
			TimeStamp:    0x2A3B4C5D,
			Where:        house.Point{V: 145, H: 216},
			Score:        8600,
			Energy:       -20,
			Bands:        7,
			RoomNumber:   int16(rooms / 2),
			GliderState:  1,
			NumGliders:   3,
			Foil:         1,
			Facing:       1,
			ShowFoil:     1,
		},
		Rooms: make([]house.SavedRoom, rooms),
	}
	sg.HouseName.SetText(name)
	for r := range sg.Rooms {
		sg.Rooms[r].Visited = byte(r % 2)
		sg.Rooms[r].Objects[0].What = int16(100 + r)
	}
	return sg
}

// houseFor is a house the sample above belongs to: the same stamp, the same room count.
func houseFor(sg *house.SavedGame) *house.House {
	return &house.House{
		TimeStamp: sg.Game.TimeStamp,
		NRooms:    int16(len(sg.Rooms)),
		Rooms:     make([]house.Room, len(sg.Rooms)),
	}
}

// ------------------------------------------------------------------ names and places

// The two kinds of player file are siblings with one stem, which is internal/datadir's job
// and is tested there. What is tested here is that this package actually asks it: a store
// that built its own path would drift from the score board's the first time a house was
// named something awkward.
func TestPathComesFromDatadir(t *testing.T) {
	st := OpenDir("/tmp/saves")
	for _, name := range []string{"Slumberland", "Fun House", "CON", "../../etc/passwd", "Café"} {
		want := filepath.Join("/tmp/saves", datadir.FileName(name, Ext))
		got := st.Path(name)
		if got != want {
			t.Errorf("Path(%q) = %q, want %q", name, got, want)
		}
		if !strings.HasSuffix(got, Ext) {
			t.Errorf("Path(%q) does not end in %q", name, Ext)
		}
	}
}

func TestDirHonoursGLIDERGO_DATA(t *testing.T) {
	t.Setenv("GLIDERGO_DATA", "/tmp/glider-portable")
	// Verbatim, with no sub-directory: the portable-install hatch puts Titanic.save next
	// to Titanic.scores. See internal/datadir.
	if got, err := Dir(); err != nil || got != "/tmp/glider-portable" {
		t.Errorf("Dir() = %q, %v", got, err)
	}
	t.Setenv("GLIDERGO_DATA", "")
	t.Setenv("GLIDERGO_CONFIG", "/tmp/glider-config")
	if got, err := Dir(); err != nil || got != filepath.Join("/tmp/glider-config", SubDir) {
		t.Errorf("Dir() = %q, %v", got, err)
	}
}

// ------------------------------------------------------------------ save and load

func TestSaveThenLoadReturnsTheSameGame(t *testing.T) {
	st := OpenDir(t.TempDir())
	sg := sample("Slumberland", 6)

	if st.Has("Slumberland") {
		t.Error("Has says there is a save before anything was written")
	}
	if err := st.Save(sg); err != nil {
		t.Fatal(err)
	}
	if !st.Has("Slumberland") {
		t.Error("Has says there is no save after one was written")
	}

	got, err := st.Load("Slumberland")
	if err != nil {
		t.Fatal(err)
	}
	// Encode writes the room count over UnusedShort, so the expectation has to as well.
	want := *sg
	want.Game.UnusedShort = int16(len(sg.Rooms))
	if got.Format != want.Format || got.HouseName != want.HouseName || got.Game != want.Game {
		t.Errorf("header round-tripped to %+v, want %+v", got.Game, want.Game)
	}
	if len(got.Rooms) != len(want.Rooms) {
		t.Fatalf("loaded %d rooms, want %d", len(got.Rooms), len(want.Rooms))
	}
	for r := range got.Rooms {
		if got.Rooms[r] != want.Rooms[r] {
			t.Errorf("room %d did not round-trip", r)
		}
	}
}

// One save per house, and the newer one wins. This is the deliberate loss the package
// comment records: the original's Standard File dialogue would have allowed twenty saves of
// one house under twenty names.
func TestSavingTwiceReplacesTheFirstSave(t *testing.T) {
	dir := t.TempDir()
	st := OpenDir(dir)

	first := sample("Slumberland", 3)
	if err := st.Save(first); err != nil {
		t.Fatal(err)
	}
	second := sample("Slumberland", 3)
	second.Game.Score = 99999
	if err := st.Save(second); err != nil {
		t.Fatal(err)
	}

	got, err := st.Load("Slumberland")
	if err != nil {
		t.Fatal(err)
	}
	if got.Game.Score != 99999 {
		t.Errorf("score = %d, want the second save's 99999", got.Game.Score)
	}

	// And no temporaries: an interrupted save must leave the previous save intact, which
	// means the write goes to a temporary file, but a *finished* save must not leave one
	// behind for the player to find in their data directory.
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 || ents[0].Name() != datadir.FileName("Slumberland", Ext) {
		names := make([]string, len(ents))
		for i, e := range ents {
			names[i] = e.Name()
		}
		t.Errorf("the directory holds %v, want just the one save", names)
	}
}

// The file's name and the name inside it come from the same place, so they cannot disagree.
// A save whose header named one house and whose path named another would fail Check for a
// reason no player could act on.
func TestSaveIsKeyedByTheNameInTheSave(t *testing.T) {
	dir := t.TempDir()
	st := OpenDir(dir)
	if err := st.Save(sample("Titanic", 2)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, datadir.FileName("Titanic", Ext))); err != nil {
		t.Errorf("the save is not at Titanic's path: %v", err)
	}
}

// The directory is made when a save needs it and not before, so merely launching the game
// writes nothing at all.
func TestSaveCreatesTheDirectoryAndOpenDoesNot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "not", "there", "yet")
	st := OpenDir(dir)
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the directory exists before any save: %v", err)
	}
	if err := st.Save(sample("Slumberland", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("Save did not create the directory: %v", err)
	}
}

func TestLoadOfAMissingSaveIsErrNotExist(t *testing.T) {
	st := OpenDir(t.TempDir())
	_, err := st.Load("Slumberland")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Load of a missing save = %v, want os.ErrNotExist -- "+
			"a caller has to be able to tell `no save yet` from `the save is broken`", err)
	}
}

// Everything a file in that directory can turn out to be. Each has to be refused, and the
// two that mean "this is not a saved game" have to say so, because a player who put the
// wrong file there wants different words from one whose save was truncated.
func TestLoadRefusesEverythingThatIsNotASave(t *testing.T) {
	good, err := sample("Slumberland", 4).Encode()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		data     []byte
		notASave bool
	}{
		{name: "empty", data: nil, notASave: true},
		{name: "a truncated header", data: good[:20], notASave: true},
		{name: "a house file", data: make([]byte, 2048), notASave: true},
		{name: "a truncated body", data: good[:len(good)-8]},
		{name: "a save with a byte glued on", data: append(append([]byte(nil), good...), 0)},
	}
	for _, c := range cases {
		dir := t.TempDir()
		st := OpenDir(dir)
		if err := os.WriteFile(st.Path("Slumberland"), c.data, 0o644); err != nil {
			t.Fatal(err)
		}

		// Has is deliberately not fooled into saying no: it is the menu's cheap question
		// and answers from a stat, so a broken save is reported when it is opened, where
		// there is somewhere to put the reason.
		if !st.Has("Slumberland") {
			t.Errorf("%s: Has = false; it must not read the file", c.name)
		}

		_, err := st.Load("Slumberland")
		if err == nil {
			t.Errorf("%s: loaded", c.name)
			continue
		}
		if errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s: reported as missing rather than as broken: %v", c.name, err)
		}
		if is := errors.Is(err, house.ErrNotSavedGame); is != c.notASave {
			t.Errorf("%s: errors.Is(err, ErrNotSavedGame) = %v, want %v (%v)",
				c.name, is, c.notASave, err)
		}
		if !strings.Contains(err.Error(), st.Path("Slumberland")) {
			t.Errorf("%s: the error does not name the file: %v", c.name, err)
		}
	}
}

// The bound. The original read whatever the filesystem reported straight into a fixed
// structure; this refuses by arithmetic before allocating, so a file that grew to fill a
// disk is a message rather than an out-of-memory.
func TestLoadRefusesAnAbsurdlyLargeFile(t *testing.T) {
	st := OpenDir(t.TempDir())
	big := make([]byte, house.SizeofSavedGameHeader+8193*house.SizeofSavedRoom)
	copy(big, house.SavedGameMagic)
	if err := os.WriteFile(st.Path("Slumberland"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Load("Slumberland"); err == nil {
		t.Error("a 2.4MB save was read")
	}
}

func TestRemove(t *testing.T) {
	st := OpenDir(t.TempDir())
	if err := st.Remove("Slumberland"); err != nil {
		t.Errorf("removing a save that is not there = %v, want nil", err)
	}
	if err := st.Save(sample("Slumberland", 2)); err != nil {
		t.Fatal(err)
	}
	if err := st.Remove("Slumberland"); err != nil {
		t.Fatal(err)
	}
	if st.Has("Slumberland") {
		t.Error("the save is still there")
	}
}

// A nil store is a build with nowhere to keep saves: -shot, a replay, -saves none, a
// read-only installation. Every method has to tolerate it, because that is what lets the
// game's hook be installed unconditionally and the menu row be greyed from one test.
func TestANilStoreIsAnEmptyStore(t *testing.T) {
	var st *Store
	if st.Dir() != "" || st.Path("Slumberland") != "" || st.Has("Slumberland") {
		t.Error("a nil store claims to have something")
	}
	if err := st.Save(sample("Slumberland", 1)); err == nil {
		t.Error("a nil store saved a game")
	} else if !strings.Contains(err.Error(), "nowhere") {
		t.Errorf("the error does not say why: %v", err)
	}
	if _, err := st.Load("Slumberland"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("nil Load = %v, want os.ErrNotExist", err)
	}
	if _, err := st.Peek("Slumberland"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("nil Peek = %v, want os.ErrNotExist", err)
	}
	if err := st.Remove("Slumberland"); err != nil {
		t.Errorf("nil Remove = %v, want nil", err)
	}
}

// ------------------------------------------------------------------ peeking

func TestPeekDescribesASaveWithoutReadingItsRooms(t *testing.T) {
	st := OpenDir(t.TempDir())
	sg := sample("Slumberland", 12)
	if err := st.Save(sg); err != nil {
		t.Fatal(err)
	}

	info, err := st.Peek("Slumberland")
	if err != nil {
		t.Fatal(err)
	}
	want := Info{
		HouseName: "Slumberland",
		Version:   house.SavedGameVersion,
		Score:     8600,
		Gliders:   3,
		Room:      6,
		Stars:     4,
		Bands:     7,
		Battery:   -20,
		Rooms:     12,
		Size:      int64(house.SizeofSavedGameHeader + 12*house.SizeofSavedRoom),
	}
	if info != want {
		t.Errorf("Peek =\n\t%+v\nwant\n\t%+v", info, want)
	}
	if got, want := info.Summary(), "3 gliders, 8600 points, room 6"; got != want {
		t.Errorf("Summary = %q, want %q", got, want)
	}

	// The original's own pluralisation, from QueryResumeGame's ParamText: one glider, not
	// one gliders.
	one := Info{Gliders: 1, Score: 10, Room: 2}
	if got, want := one.Summary(), "1 glider, 10 points, room 2"; got != want {
		t.Errorf("Summary = %q, want %q", got, want)
	}
}

// Peek's room count comes from the file's size and is then compared with the header's
// claim, so a truncated save is caught before the player has chosen it -- which is the
// opposite of the original's order, where the field was trusted.
func TestPeekCatchesATruncatedSave(t *testing.T) {
	good, err := sample("Slumberland", 5).Encode()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		data []byte
	}{
		{"a whole room short", good[:len(good)-house.SizeofSavedRoom]},
		{"half a room short", good[:len(good)-house.SizeofSavedRoom/2]},
		{"header only", good[:house.SizeofSavedGameHeader]},
		{"less than a header", good[:12]},
	}
	for _, c := range cases {
		st := OpenDir(t.TempDir())
		if err := os.WriteFile(st.Path("Slumberland"), c.data, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := st.Peek("Slumberland"); err == nil {
			t.Errorf("%s: peeked without complaint", c.name)
		}
	}
}

// ------------------------------------------------------------------ the five gates

// OpenSavedGame's four validations plus the port's fifth, each triggered on its own, and
// -- the part worth pinning -- in the original's order, because the order decides which
// complaint a player sees when more than one thing is wrong.
func TestCheckIsTheOriginalsGatesInTheOriginalsOrder(t *testing.T) {
	sg := sample("Slumberland", 8)
	h := houseFor(sg)

	if err := Check(sg, "Slumberland", h); err != nil {
		t.Fatalf("a matching save was refused: %v", err)
	}
	// EqualString(..., true, true): the C's comparison ignores case. The file is keyed by
	// the escaped name anyway, so on a case-folding filesystem the two spellings are
	// already one file.
	if err := Check(sg, "SLUMBERLAND", h); err != nil {
		t.Errorf("the name comparison is case-sensitive: %v", err)
	}

	cases := []struct {
		name  string
		house string
		setUp func(sg *house.SavedGame, h *house.House)
		want  string
	}{
		{name: "gate 1: another house's save", house: "Titanic", want: `"Slumberland", not "Titanic"`},
		{name: "gate 2: the house has been edited since", house: "Slumberland",
			setUp: func(_ *house.SavedGame, h *house.House) { h.TimeStamp++ },
			want:  "has been modified"},
		{name: "gate 3: a version from the future", house: "Slumberland",
			setUp: func(sg *house.SavedGame, _ *house.House) { sg.Game.Version = 0x0300 },
			want:  "version 0x0300"},
		{name: "gate 4: the house grew", house: "Slumberland",
			setUp: func(_ *house.SavedGame, h *house.House) {
				h.Rooms = append(h.Rooms, house.Room{})
			},
			want: "has 8 rooms and Slumberland has 9"},
		{name: "gate 5: a room the house does not have", house: "Slumberland",
			setUp: func(sg *house.SavedGame, _ *house.House) { sg.Game.RoomNumber = 8 },
			want:  "is in room 8"},
		{name: "gate 5: a negative room", house: "Slumberland",
			setUp: func(sg *house.SavedGame, _ *house.House) { sg.Game.RoomNumber = -1 },
			want:  "is in room -1"},
	}
	for _, c := range cases {
		sg := sample("Slumberland", 8)
		h := houseFor(sg)
		if c.setUp != nil {
			c.setUp(sg, h)
		}
		err := Check(sg, c.house, h)
		if err == nil {
			t.Errorf("%s: accepted", c.name)
			continue
		}
		var mm *Mismatch
		if !errors.As(err, &mm) {
			t.Errorf("%s: %T, want a *Mismatch -- a caller may offer a new game for these "+
				"and not for a parse failure", c.name, err)
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want it to mention %q", c.name, err, c.want)
		}
	}

	// The order, stated as a test: with the wrong house *and* a changed stamp *and* a bad
	// version, the player is told about the house, because that is the one they can act on.
	wrong := sample("Titanic", 8)
	wrong.Game.Version = 0x0300
	wrong.Game.TimeStamp = 0
	if err := Check(wrong, "Slumberland", h); err == nil ||
		!strings.Contains(err.Error(), "not \"Slumberland\"") {
		t.Errorf("with three things wrong the complaint was %v, want the house name", err)
	}
}

// S3: the shipped corpus's saves are all 0x0100, which the original's own gate would have
// rejected -- so both versions pass and a third does not.
func TestCheckAcceptsBothVersionsThatExist(t *testing.T) {
	for _, v := range []int16{house.SavedGameVersion, house.SavedGameVersion1} {
		sg := sample("Slumberland", 8)
		sg.Game.Version = v
		if err := Check(sg, "Slumberland", houseFor(sg)); err != nil {
			t.Errorf("version %#04x refused: %v", v, err)
		}
	}
}

// A nil house is "check what you can": a menu listing saves has the name and not the file.
func TestCheckWithoutAHouseChecksTheNameOnly(t *testing.T) {
	sg := sample("Slumberland", 8)
	sg.Game.Version = 0x0300
	sg.Game.RoomNumber = 4000
	if err := Check(sg, "Slumberland", nil); err != nil {
		t.Errorf("with no house to check against: %v", err)
	}
	if err := Check(sg, "Titanic", nil); err == nil {
		t.Error("the name is not checked without a house")
	}
}

// InfoOf is Peek without the file, and the two must not drift: the menu row's line is
// written by one of them and read by a player who cannot tell which.
func TestInfoOfAgreesWithPeek(t *testing.T) {
	st := OpenDir(t.TempDir())
	sg := sample("Slumberland", 12)
	if err := st.Save(sg); err != nil {
		t.Fatal(err)
	}
	peeked, err := st.Peek("Slumberland")
	if err != nil {
		t.Fatal(err)
	}

	// Encode wrote the room count over UnusedShort, so the in-memory game has to be reloaded
	// rather than described as it was handed to Save -- which is the point: InfoOf describes
	// whatever it is given, and the two callers give it the same thing.
	loaded, err := st.Load("Slumberland")
	if err != nil {
		t.Fatal(err)
	}
	got := InfoOf(loaded)
	got.Size = peeked.Size // the one field only a file has
	if got != peeked {
		t.Errorf("InfoOf =\n\t%+v\nPeek =\n\t%+v", got, peeked)
	}

	// The house's own embedded block: no file, no rooms, and a summary all the same. This is
	// the case InfoOf exists for -- see house.EmbeddedGame.
	embedded := house.EmbeddedGame("Titanic", 0x2A3B4C5D, sg.Game)
	info := InfoOf(embedded)
	if info.Rooms != 0 {
		t.Errorf("an embedded game reported %d rooms", info.Rooms)
	}
	if info.HouseName != "Titanic" || info.Score != 8600 || info.Gliders != 3 {
		t.Errorf("InfoOf(embedded) = %+v", info)
	}
	if info.FromHouse {
		// Set by the host, which is the only thing that knows where it looked. A package
		// that opens saves must not claim a save came from somewhere else.
		t.Error("InfoOf set FromHouse; that is the host's word, not this package's")
	}
	if got := InfoOf(nil); got != (Info{}) {
		t.Errorf("InfoOf(nil) = %+v, want the zero Info", got)
	}
}

// Gate 4's nil arm: a save with no room snapshot at all, which is what a house's own 40 bytes
// are. There is nothing to compare and nothing to corrupt, so the count is not checked -- and
// the marker cannot come out of a file, which is what keeps that from being a hole.
func TestCheckAcceptsASaveWithNoRoomSnapshot(t *testing.T) {
	h := &house.House{
		TimeStamp: 0x2A3B4C5D,
		NRooms:    8,
		Rooms:     make([]house.Room, 8),
	}
	sg := house.EmbeddedGame("Slumberland", h.TimeStamp, sample("ignored", 8).Game)
	if sg.Rooms != nil {
		t.Fatal("EmbeddedGame carried a room snapshot")
	}
	if err := Check(sg, "Slumberland", h); err != nil {
		t.Errorf("a house's own embedded game was refused: %v", err)
	}

	// Every other gate still applies to it. The nil is a licence about the rooms and nothing
	// else, which is what the fifth gate below is for: an embedded block naming a room the
	// house has since lost is still refused.
	sg.Game.RoomNumber = 8
	if err := Check(sg, "Slumberland", h); err == nil {
		t.Error("an embedded game naming room 8 of an 8-room house was accepted")
	}

	// Zero rooms is not the same as no rooms, and a file cannot say the second: DecodeSavedGame
	// allocates the slice from the count, so a header-only save decodes to an empty non-nil one
	// and is caught by the count comparison.
	empty, err := (&house.SavedGame{
		Format: house.SavedGameFormat,
		Game:   sg.Game,
		Rooms:  []house.SavedRoom{},
	}).Encode()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := house.DecodeSavedGame(empty)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Rooms == nil {
		t.Fatal("a decoded save with no rooms has a nil slice: the nil marker can be forged " +
			"by a file, and gate 4 can be skipped by truncating one")
	}
	if err := Check(decoded, "Slumberland", h); err == nil {
		t.Error("a save carrying zero rooms was accepted for an 8-room house")
	}
}

// The empty-house case, which the fifth gate covers for free and which no earlier gate
// would: every room number is out of range when there are no rooms.
func TestCheckRefusesAnyRoomOfAnEmptyHouse(t *testing.T) {
	sg := sample("Slumberland", 0)
	h := &house.House{TimeStamp: sg.Game.TimeStamp}
	if err := Check(sg, "Slumberland", h); err == nil {
		t.Error("a save was accepted for a house with no rooms")
	}
}
