package main

// The host's half of the saved game: where the file goes, which of the two possible saves a
// resume gets, and the two flag combinations that contradict themselves.
//
// The round trip itself is not here. internal/game's TestSaveAndResumeRoundTripsARealGame
// plays a real house, saves through the real encoder and resumes into a fresh World, which is
// docs/PLAN.md 1.10's acceptance criterion; internal/saved owns the five gates. What is left
// for this file is the policy that only the host can state, and all of it is about *choosing*:
// a house can carry a game of its own (two of the twenty-two do), the player can have one of
// their own, and the two are not the same offer.

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"glidergo/internal/house"
	"glidergo/internal/platform"
	"glidergo/internal/prefs"
	"glidergo/internal/saved"
	"glidergo/internal/shell"
)

// savesApp is an app with a saves directory and nothing else: no window, no audio, no scores.
func savesApp(t *testing.T, savesDir string) *app {
	t.Helper()
	a := newApp(&options{savesDir: savesDir, scoresDir: scoresNone}, prefs.Default(), false)
	return a
}

// A house file with a game inside it, which is what `hasGame` and the 40 bytes at offset 820
// are: Titanic's shape, authored rather than extracted so the numbers are legible here.
func houseWithGame(t *testing.T, dir, name string, hasGame bool, score int32, room int16) (string, *house.House) {
	t.Helper()
	h := &house.House{
		Version:   house.HouseVersion,
		TimeStamp: 0x2A3B4C5D,
		FirstRoom: 0,
		Rooms:     make([]house.Room, 4),
	}
	if hasGame {
		h.HasGame = 1
		h.SavedGame = house.Game{
			Version:    house.SavedGameVersion1, // both shipped blocks are 0x0100 -- S3
			Score:      score,
			RoomNumber: room,
			NumGliders: 2,
			Energy:     0,
			// The clock, as SaveGame wrote it (SavedGames.c:320). It is deliberately *not*
			// the house's stamp, so a test that lost EmbeddedGame's substitution would see
			// saved.Check refuse the block.
			TimeStamp: 0x11111111,
		}
	}
	path := filepath.Join(dir, name+".house")
	if err := h.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	return path, h
}

// A save of our own for the same house, with a score nothing else has.
func writeSave(t *testing.T, a *app, name string, score int32, rooms int) {
	t.Helper()
	sg := &house.SavedGame{
		Format: house.SavedGameFormat,
		Game: house.Game{
			Version:    house.SavedGameVersion,
			TimeStamp:  0x2A3B4C5D,
			Score:      score,
			RoomNumber: 1,
			NumGliders: 3,
		},
		Rooms: make([]house.SavedRoom, rooms),
	}
	sg.HouseName.SetText(name)
	if err := a.saves.Save(sg); err != nil {
		t.Fatal(err)
	}
}

// -saves is -scores' and -prefs' pattern, and `none` has to be a total opt-out rather than a
// quieter one: with no store the pause hint stops offering S, so a build that opened a
// directory anyway would advertise a key and then write a file the player asked it not to.
func TestOpenSavesHonoursTheFlag(t *testing.T) {
	dir := t.TempDir()

	if a := savesApp(t, savesNone); a.saves != nil {
		t.Errorf("-saves none opened %q", a.saves.Dir())
	}
	a := savesApp(t, dir)
	if a.saves == nil {
		t.Fatal("-saves <dir> opened no store")
	}
	if got := a.saves.Dir(); got != dir {
		t.Errorf("store directory is %q, want %q", got, dir)
	}
	// Nothing on disk yet: opening a store must not create anything, so a session that never
	// saves leaves no trace of the flag.
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Errorf("opening a store wrote %d entries (%v)", len(entries), err)
	}

	// A nil store answers every question rather than crashing, which is what lets the two
	// call sites below be the only two checks of a.saves in the program.
	none := savesApp(t, savesNone)
	if _, err := none.savedGame(shell.House{Name: "Slumberland"}); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a nil store's savedGame said %v, want os.ErrNotExist", err)
	}
}

// The order that is the whole of the policy: the player's own save wins over the game the
// house shipped with, and the row is told which of the two it is holding.
func TestSavedGamePrefersThePlayersOwnSave(t *testing.T) {
	dir := t.TempDir()
	path, _ := houseWithGame(t, dir, "Titanic", true, 4700, 3)
	h := shell.House{Name: "Titanic", Rel: "Titanic.house", Path: path}

	a := savesApp(t, filepath.Join(dir, "saves"))

	// With no save of our own, the house's own block -- which in 1994 nothing could open,
	// because the menu item that would have (Menu.c:458) is commented out.
	info, err := a.savedGame(h)
	if err != nil {
		t.Fatalf("a house carrying a game reported %v", err)
	}
	if !info.FromHouse {
		t.Error("FromHouse is false for the house's own block; the row would tell the player " +
			"this was their save")
	}
	if info.Score != 4700 || info.Room != 3 || info.Gliders != 2 {
		t.Errorf("the house's block described as %+v", info)
	}
	if info.Rooms != 0 {
		t.Errorf("the house's block reported %d rooms; a houseType carries no snapshot", info.Rooms)
	}

	// Now a save of our own for the same house.
	writeSave(t, a, "Titanic", 8600, 4)
	info, err = a.savedGame(h)
	if err != nil {
		t.Fatal(err)
	}
	if info.FromHouse {
		t.Error("FromHouse is true with a save of our own in the store")
	}
	if info.Score != 8600 || info.Rooms != 4 {
		t.Errorf("our own save described as %+v", info)
	}

	// And deleting it brings the house's block back, because nothing ever wrote the house.
	if err := a.saves.Remove("Titanic"); err != nil {
		t.Fatal(err)
	}
	if info, err := a.savedGame(h); err != nil || !info.FromHouse || info.Score != 4700 {
		t.Errorf("after removing our save: %+v, %v", info, err)
	}
}

// A house with no game of its own and no save is the ordinary case, and the error has to be
// the store's os.ErrNotExist so the menu row can say "no saved game for Slumberland -- press S
// while paused" rather than something about house files.
func TestSavedGameWithNothingToResume(t *testing.T) {
	dir := t.TempDir()
	path, _ := houseWithGame(t, dir, "Slumberland", false, 0, 0)
	a := savesApp(t, filepath.Join(dir, "saves"))

	_, err := a.savedGame(shell.House{Name: "Slumberland", Path: path})
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("savedGame said %v, want os.ErrNotExist", err)
	}

	// A house that is not a house at all: still the store's error, because the row's question
	// is whether there is a game to resume and a file that cannot be read is a problem the
	// player meets when they play it, with a better message than a menu row can hold.
	bad := filepath.Join(dir, "Nonsense.house")
	if err := os.WriteFile(bad, []byte("MZ not a house"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.savedGame(shell.House{Name: "Nonsense", Path: bad}); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("an unreadable house reported %v, want the store's os.ErrNotExist", err)
	}
}

// A save that exists and is broken must not fall through to the house's block. Silently
// resuming somebody else's 1994 game because this morning's save is corrupt is the one outcome
// worse than saying so.
func TestABrokenSaveIsReportedAndNotSubstituted(t *testing.T) {
	dir := t.TempDir()
	path, h := houseWithGame(t, dir, "Titanic", true, 4700, 3)
	a := savesApp(t, filepath.Join(dir, "saves"))

	if err := os.MkdirAll(a.saves.Dir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a.saves.Path("Titanic"), []byte("gliGnot really"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := a.savedGame(shell.House{Name: "Titanic", Path: path})
	if err == nil {
		t.Fatalf("a broken save peeked as %+v", info)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Error("a broken save reported os.ErrNotExist; the row would say there is no save")
	}
	if info.FromHouse {
		t.Error("a broken save was replaced by the house's own block")
	}
	// And the same at the load side, which is what a resume actually goes through.
	if _, err := a.savedGameFor("Titanic", h); err == nil {
		t.Error("savedGameFor read a broken save without complaint")
	}
}

// savedGameFor is savedGame's order applied to the whole file, and the fallback it produces
// has to survive the gates: the house's block carries the clock's stamp and the second gate
// compares the house's, so EmbeddedGame's substitution is what makes a resume of it possible
// at all.
func TestSavedGameForFallsBackToTheHousesOwnBlock(t *testing.T) {
	dir := t.TempDir()
	_, h := houseWithGame(t, dir, "Titanic", true, 4700, 3)
	a := savesApp(t, filepath.Join(dir, "saves"))

	sg, err := a.savedGameFor("Titanic", h)
	if err != nil {
		t.Fatal(err)
	}
	if sg.Game.Score != 4700 || sg.Game.RoomNumber != 3 {
		t.Errorf("the house's block loaded as %+v", sg.Game)
	}
	if sg.Rooms != nil {
		t.Error("the house's block came with a room snapshot")
	}
	if err := saved.Check(sg, "Titanic", h); err != nil {
		t.Errorf("the house's own game does not pass the gates: %v", err)
	}

	// Our own save wins here too, and it is the whole file: rooms and all.
	writeSave(t, a, "Titanic", 8600, len(h.Rooms))
	sg, err = a.savedGameFor("Titanic", h)
	if err != nil {
		t.Fatal(err)
	}
	if sg.Game.Score != 8600 || len(sg.Rooms) != len(h.Rooms) {
		t.Errorf("our own save loaded as score %d with %d rooms", sg.Game.Score, len(sg.Rooms))
	}
	if err := saved.Check(sg, "Titanic", h); err != nil {
		t.Errorf("our own save does not pass the gates: %v", err)
	}

	// And a house with neither: os.ErrNotExist, wrapped in words a player can read.
	_, plain := houseWithGame(t, dir, "Slumberland", false, 0, 0)
	_, err = a.savedGameFor("Slumberland", plain)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("savedGameFor said %v, want os.ErrNotExist", err)
	}
	if err == nil || !strings.Contains(err.Error(), "Slumberland has no saved game to resume") {
		t.Errorf("the error reads %v", err)
	}
}

// ------------------------------------------------------------------ the keys, in a real game

// playApp is an app that can play: a scripted window, no audio, no art, and a game that runs
// flat out. -bench with no frame limit is what makes it fast and also what makes it quiet --
// with no tick source there is no pacing, and with no Wait hook the banner and the
// stars-remaining panel do not block (see play.go's five hooks).
func playApp(t *testing.T, savesDir string, script ...[]platform.Event) (*app, *fakeWin) {
	t.Helper()
	o := &options{
		bench:     true,
		quiet:     true,
		roomNum:   -1,
		savesDir:  savesDir,
		scoresDir: scoresNone,
	}
	a := newApp(o, prefs.Default(), false)
	win := &fakeWin{script: script, held: map[platform.Key]bool{}}
	a.win = win
	a.fb = platform.NewFramebuffer(640, 480)
	return a, win
}

// idlePolls pads a script so that a poll after the game has ended is not read as a closed window.
func idlePolls(n int) [][]platform.Event {
	s := make([][]platform.Event, n)
	for i := range s {
		s[i] = []platform.Event{}
	}
	return s
}

// The two keys 1.10 adds to the pause, driven through the real game: S saves, and Q asks the
// question alert 1041 asked, whose two buttons were "Save First" and "Don't Save".
//
// These arms are closures inside play() and this is the only test of them, because the whole
// point of them is the order they happen in -- a save from *inside* the pause loop, with the
// placard on the screen and the game still live. So each case is a whole game, and what it
// asserts is what a player would find afterwards: a file or no file, and a game that ended
// without the window closing.
//
// The three cases are chosen so that each key is the only thing that could have written the
// file. "Q then N" is the one that would catch a save arm wired to both answers.
func TestSavingAndGivingUpFromThePause(t *testing.T) {
	cases := []struct {
		name     string
		keys     []platform.Key
		wantSave bool
	}{
		{"S saves and N leaves without saving again", []platform.Key{
			platform.KeyS, platform.KeyQ, platform.KeyN}, true},
		{"Y is Save First", []platform.Key{platform.KeyQ, platform.KeyY}, true},
		{"N is Don't Save", []platform.Key{platform.KeyQ, platform.KeyN}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path, h := houseWithGame(t, dir, "Slumberland", false, 0, 0)

			// Three empty polls first: the frame loop's, whose KeyPoll then sees the press
			// edge and pauses, the pause loop's own first pass -- where the C's release loop
			// (A) waits -- and the pass that sees the key released. Then one key per pass,
			// with a gap between them so that nothing depends on two arriving together.
			script := [][]platform.Event{{}, {}, {}}
			for _, k := range c.keys {
				script = append(script, pressed(k), []platform.Event{})
			}
			script = append(script, idlePolls(20)...)

			a, win := playApp(t, filepath.Join(dir, "saves"), script...)
			pauseKey := a.p.Pause()
			win.onPoll = func(f *fakeWin) { f.held[pauseKey] = f.pass < 2 }

			out, err := a.play("Slumberland", path, false, false)
			if err != nil {
				t.Fatal(err)
			}
			if out.Closed {
				t.Error("the game reported a closed window; Q is a give-up, not a quit")
			}
			if win.pass > len(script) {
				t.Errorf("the script ran out after %d polls: the pause never ended", win.pass)
			}

			sg, err := a.saves.Load("Slumberland")
			if !c.wantSave {
				if err == nil {
					t.Fatalf("a save was written anyway: %+v", sg.Game)
				}
				return
			}
			if err != nil {
				t.Fatalf("no save was written from the pause: %v", err)
			}
			if got := sg.HouseName.Text(); got != "Slumberland" {
				t.Errorf("the save names %q; World.HouseName is what writes this", got)
			}
			if len(sg.Rooms) != len(h.Rooms) {
				t.Errorf("the save carries %d rooms, want %d: a game2Type restores the house "+
					"and not just the player", len(sg.Rooms), len(h.Rooms))
			}
			if sg.Game.Version != house.SavedGameVersion {
				t.Errorf("the save is version %#04x, want kSavedGameVersion %#04x",
					sg.Game.Version, house.SavedGameVersion)
			}
			if sg.Game.TimeStamp != h.TimeStamp {
				t.Errorf("the save is stamped %#x, want the house's %#x -- the second gate "+
					"compares the house's stamp", sg.Game.TimeStamp, h.TimeStamp)
			}
			if err := saved.Check(sg, "Slumberland", h); err != nil {
				t.Errorf("the game this port saved does not pass its own gates: %v", err)
			}
		})
	}
}

// With nothing to save, Q ends the game on the keystroke rather than asking a question with no
// good answer. That is where the C puts its own guard too -- around the question, not around
// the save (Input.c:55-63).
func TestGivingUpWithNowhereToSave(t *testing.T) {
	dir := t.TempDir()
	path, _ := houseWithGame(t, dir, "Slumberland", false, 0, 0)

	script := append([][]platform.Event{
		{}, {}, {},
		pressed(platform.KeyQ),
	}, idlePolls(20)...)

	a, win := playApp(t, savesNone, script...)
	if a.saves != nil {
		t.Fatal("-saves none opened a store")
	}
	pauseKey := a.p.Pause()
	win.onPoll = func(f *fakeWin) { f.held[pauseKey] = f.pass < 2 }

	out, err := a.play("Slumberland", path, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Closed {
		t.Error("the game reported a closed window")
	}
	if win.pass > len(script) {
		t.Errorf("the script ran out after %d polls: Q did not end the game", win.pass)
	}
	if _, err := os.Stat(filepath.Join(dir, "saves")); !os.IsNotExist(err) {
		t.Errorf("a saves directory appeared under -saves none: %v", err)
	}
}

// The other direction, end to end through play(): the mode. A resumed game starts with the
// save's score and a new one starts at nothing, and nothing else in the two calls differs --
// which is what makes this a test of the one line that chooses kResumeGameMode.
func TestResumeStartsFromTheSavedGame(t *testing.T) {
	dir := t.TempDir()
	path, h := houseWithGame(t, dir, "Slumberland", false, 0, 0)
	savesDir := filepath.Join(dir, "saves")

	a, _ := playApp(t, savesDir)
	writeSave(t, a, "Slumberland", 8600, len(h.Rooms))

	// An empty script: the first poll reports the window closed, so each game ends after a
	// frame or two -- long enough for NewGame to have read the save and not long enough for
	// the glider to reach anything that would change the score.
	a, _ = playApp(t, savesDir)
	out, err := a.play("Slumberland", path, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if out.Score != 8600 {
		t.Errorf("a resumed game started with score %d, want the save's 8600", out.Score)
	}

	a, _ = playApp(t, savesDir)
	out, err = a.play("Slumberland", path, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Score != 0 {
		t.Errorf("a new game started with score %d; the save was applied to it", out.Score)
	}
}

// A refused resume never starts a game, and says which of the gates refused it. The C runs
// these checks after it has begun tearing the world down, which is why it can only answer with
// a yellow alert over a half-started game (see internal/saved's Check).
func TestResumeRefusalsHappenBeforeTheGameStarts(t *testing.T) {
	dir := t.TempDir()
	path, h := houseWithGame(t, dir, "Slumberland", false, 0, 0)
	savesDir := filepath.Join(dir, "saves")

	// Nothing to resume.
	a, win := playApp(t, savesDir)
	if _, err := a.play("Slumberland", path, false, true); err == nil {
		t.Error("a resume with no saved game started a game")
	} else if !strings.Contains(err.Error(), "no saved game to resume") {
		t.Errorf("%v", err)
	}
	if win.presents != 0 {
		t.Errorf("%d frames were presented for a resume that could not happen", win.presents)
	}

	// A save for a house that has been edited since: gate 2, and the one a player is most
	// likely to meet, because it is what re-saving a house in the editor does.
	a, win = playApp(t, savesDir)
	writeSave(t, a, "Slumberland", 8600, len(h.Rooms))
	sg, err := a.saves.Load("Slumberland")
	if err != nil {
		t.Fatal(err)
	}
	sg.Game.TimeStamp++
	if err := a.saves.Save(sg); err != nil {
		t.Fatal(err)
	}
	if _, err := a.play("Slumberland", path, false, true); err == nil {
		t.Error("a save from before the house was edited was resumed")
	} else if !strings.Contains(err.Error(), "has been modified") {
		t.Errorf("%v", err)
	}
	if win.presents != 0 {
		t.Errorf("%d frames were presented for a refused resume", win.presents)
	}
}

// The two flag combinations that ask for two different things at once. Both are refused rather
// than resolved, because either resolution silently drops something the command line asked
// for -- which is the failure mode that costs an hour of wondering why -room did nothing.
func TestResumeFlagsRefuseContradictions(t *testing.T) {
	oldArgs, oldFlags := os.Args, flag.CommandLine
	defer func() { os.Args, flag.CommandLine = oldArgs, oldFlags }()

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"-resume", "-two"}, "-resume and -two"},
		{[]string{"-resume", "-room", "3"}, "-resume and -room"},
		{[]string{"-resume"}, ""},                // on its own, fine
		{[]string{"-resume", "-room", "-1"}, ""}, // -1 is the flag's own "no override"
	}
	for _, c := range cases {
		// A fresh FlagSet per call: parseFlags registers into flag.CommandLine and would
		// panic on the second registration of the same name.
		flag.CommandLine = flag.NewFlagSet("glidergo", flag.ContinueOnError)
		flag.CommandLine.SetOutput(io.Discard)
		os.Args = append([]string{"glidergo"}, c.args...)

		_, err := parseFlags()
		switch {
		case c.want == "" && err != nil:
			t.Errorf("glidergo %v: %v", c.args, err)
		case c.want != "" && err == nil:
			t.Errorf("glidergo %v was accepted", c.args)
		case c.want != "" && err != nil && !strings.Contains(err.Error(), c.want):
			t.Errorf("glidergo %v: %v, want it to mention %q", c.args, err, c.want)
		}
	}
}
