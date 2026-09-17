package shell

// The shell's half of the high-score screen: which board is up, where it came from, how
// long it is trusted for, and that it replaces the title screen rather than sitting on it.
//
// What the board *looks like* is internal/scores' and is tested there, pixel by pixel
// against the coordinates in docs/analysis/scoring.md 7.9. What is tested here is only what
// this package decides.

import (
	"errors"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/render"
)

// board is a filled-in high-score table, for a host hook to hand back.
func board(name string, score int32, rooms int16, banner string) house.Scores {
	var b house.Scores
	b.Names[0].SetText(name)
	b.Scores[0] = score
	b.Levels[0] = rooms
	b.Banner.SetText(banner)
	return b
}

// ------------------------------------------------------------------- getting there

func TestHighScoresOpensOnHAndAnyKeyLeaves(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"},
		key(platform.KeyH), key(platform.KeyX))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if s.mode != modeSplash {
		t.Errorf("mode is %v after H then X, want the splash screen back", s.mode)
	}

	// And H on its own gets there and stays there, so the test above is not passing
	// because H did nothing at all.
	open, _ := shellOver(t, []string{"Slumberland"})
	open.event(key(platform.KeyH)[0])
	if open.mode != modeScores {
		t.Errorf("mode is %v after H, want the high scores", open.mode)
	}
}

// The item is in the menu and reachable with the arrows, not only by accelerator.
func TestTheMenuHasAHighScoresItem(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	at := -1
	for i, it := range s.menu() {
		if it.label == "High Scores..." {
			at = i
			if !it.ok {
				t.Error("High Scores... is unavailable with a house selected")
			}
			if it.key != platform.KeyH {
				t.Errorf("High Scores... is on %v, want H (Menu.c:417's Options item)", it.key)
			}
		}
	}
	if at < 0 {
		t.Fatal("the menu has no High Scores... item")
	}
	for i := 0; i < at; i++ {
		s.event(key(platform.KeyDown)[0])
	}
	s.event(key(platform.KeyReturn)[0])
	if s.mode != modeScores {
		t.Errorf("choosing the item with Return left the mode at %v", s.mode)
	}
}

// With nothing to show a board for, it refuses and says why -- rather than opening a screen
// whose title is a blank space.
func TestHighScoresNeedsAHouse(t *testing.T) {
	s, _ := shellOver(t, nil, key(platform.KeyH))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if s.mode == modeScores {
		t.Fatal("the high-score screen opened with no house selected")
	}
	if s.msg == "" {
		t.Error("pressing H with no houses said nothing")
	}
}

func TestShowOpensTheScoresScreen(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	if err := s.Show("scores"); err != nil {
		t.Fatalf("Show(scores): %v", err)
	}
	if s.mode != modeScores {
		t.Errorf("Show(scores) left the mode at %v", s.mode)
	}

	empty, _ := shellOver(t, nil)
	if err := empty.Show("scores"); err == nil {
		t.Error("Show(scores) with no house should fail rather than draw an empty board")
	}

	// The name list in the error is what -shot's own help promises, so a new screen that
	// is not listed there is a flag nobody can discover.
	err := s.Show("leaderboard")
	if err == nil || !strings.Contains(err.Error(), "scores") {
		t.Errorf("the unknown-screen error is %v; it should list scores", err)
	}
}

// ------------------------------------------------------------------ where it comes from

// No side-car in the build: the screen shows the house file's own rows, which is what makes
// the twenty-two shipped boards visible before anybody has played (docs/analysis/scoring.md
// 7.3).
func TestWithNoHostHookTheBoardIsTheHouseFilesOwn(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	s.lib.Houses[0].Scores = board("Kim", 4200, 12, "shipped in 1994")

	h, _ := s.House()
	got := s.board(h)
	if got.Names[0].Text() != "Kim" || got.Scores[0] != 4200 {
		t.Errorf("board = %q %d, want the house file's own Kim 4200",
			got.Names[0].Text(), got.Scores[0])
	}
}

// With a hook, the hook wins -- and is asked once per house, not once per frame. A board
// read off disk every frame would be sixty file reads a second for a screen that cannot
// change while it is up.
func TestTheHostsBoardIsAskedForOncePerHouse(t *testing.T) {
	s, _ := shellOver(t, []string{"Amy", "Beth"})
	s.lib.Houses[0].Scores = board("stale", 1, 1, "")

	asked := []string{}
	s.host.Scores = func(h House) house.Scores {
		asked = append(asked, h.Rel)
		return board("merged "+h.Name, 9000, 3, "")
	}

	h, _ := s.House()
	for i := 0; i < 3; i++ {
		if got := s.board(h); got.Names[0].Text() != "merged "+h.Name {
			t.Fatalf("pass %d: board came back as %q, want the host's",
				i, got.Names[0].Text())
		}
	}
	if len(asked) != 1 {
		t.Errorf("the host was asked %d times for one house: %v", len(asked), asked)
	}

	// A different house is a different cache entry, keyed on the path and not the name.
	other := s.lib.Houses[1]
	if got := s.board(other); got.Names[0].Text() != "merged "+other.Name {
		t.Errorf("the second house got %q, want its own board", got.Names[0].Text())
	}
	if len(asked) != 2 || asked[1] != other.Rel {
		t.Errorf("the host was asked for %v, want a second call for %s", asked, other.Rel)
	}
}

// A game can put a score on the board, so every board this shell is holding is suspect the
// moment a game ends. Including one that ended badly: the error path drops the cache too.
func TestTheCacheIsDroppedAfterAGame(t *testing.T) {
	for _, name := range []string{"ok", "error"} {
		s, f := shellOver(t, []string{"Slumberland"})
		if name == "error" {
			f.err = errors.New("the house went away mid-game")
		}
		calls := 0
		s.host.Scores = func(House) house.Scores {
			calls++
			return board("Kim", 4200, 12, "")
		}

		h, _ := s.House()
		s.board(h)
		if calls != 1 {
			t.Fatalf("%s: %d calls before the game, want 1", name, calls)
		}

		s.play(false)
		s.board(h)
		if calls != 2 {
			t.Errorf("%s: %d calls after a game, want the board to have been re-read",
				name, calls)
		}
	}
}

// ---------------------------------------------------------------------- the status line

func TestTheStatusLineDescribesTheBoard(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	h, _ := s.House()

	// Nothing on it yet, which is the state a house the player has never finished is in.
	s.host.Scores = func(House) house.Scores { return house.Scores{} }
	if got := s.boardLine(h); !strings.Contains(got, "no scores yet") {
		t.Errorf("an empty board reads %q", got)
	}

	// The banner is the one part of a board that is a message rather than a number, so it
	// goes in the line where a player will read it even before the screen is drawn.
	s.boards = nil
	s.host.Scores = func(House) house.Scores { return board("Kim", 4200, 12, "you can't win") }
	got := s.boardLine(h)
	for _, want := range []string{"Slumberland", "1 of 10", "you can't win"} {
		if !strings.Contains(got, want) {
			t.Errorf("the status line %q does not mention %q", got, want)
		}
	}

	// A board with scores and no banner says so without a trailing "banner:".
	s.boards = nil
	s.host.Scores = func(House) house.Scores { return board("Kim", 4200, 12, "") }
	if got := s.boardLine(h); strings.Contains(got, "banner") {
		t.Errorf("a board with no banner reads %q", got)
	}
}

// -------------------------------------------------------------------------- the drawing

// The board is a screen, not a panel: it replaces the backdrop, so no house label and no
// menu survive under it. The status band still runs along the bottom, because those twenty
// rows are the port's own and the board draws nothing in them.
func TestTheScoresScreenReplacesTheBackdrop(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"})
	s.lib.Houses[0].Scores = board("Kim", 4200, 12, "you can't win")
	if err := s.Show("scores"); err != nil {
		t.Fatal(err)
	}
	s.Draw()

	// `label` is drawn by nothing but drawHouseLabel and the fallback title screen, which
	// are both parts of the backdrop this screen replaces.
	if n := count(f.scr, label); n != 0 {
		t.Errorf("%d pixels of the splash screen's house label survived under the board", n)
	}
	// The footer is the only blue text on the screen (HighScores.c:266).
	if count(f.scr, render.QDBlue) == 0 {
		t.Error("no blue ink: the board's \"Hit a Key to Exit\" footer was not drawn")
	}
	// And the title, which is cyan over a black shadow.
	if count(f.scr, render.QDCyan) == 0 {
		t.Error("no cyan ink: the board's title row was not drawn")
	}
	if !messageIsOnScreen(f.scr) {
		t.Error("the status band is empty; it should still be drawn under the board")
	}
	// Nothing of the board may reach into the band, or the two would overprint. The
	// footer's baseline is 308 on this screen and the band starts at 460.
	for v := splashTall - 1; v >= splashTall-4; v-- {
		for x := 0; x < screenWide; x++ {
			if got := f.scr.Pix[v*f.scr.W+x]; got != render.Black8 {
				t.Fatalf("row %d column %d is %d; the board should be black down here", v, x, got)
			}
		}
	}
}

// A mode set from outside without a house must not present a blank frame.
func TestDrawingTheBoardWithNoHouseFallsBackToTheSplash(t *testing.T) {
	s, f := shellOver(t, nil)
	s.mode = modeScores
	s.Draw()
	if s.mode != modeSplash {
		t.Errorf("mode is %v; drawing a board with no house should have recovered", s.mode)
	}
	if count(f.scr, label) == 0 && count(f.scr, cream) == 0 {
		t.Error("nothing was drawn at all")
	}
}

// The picker's footer and the board must not disagree about the same house. The footer
// therefore reads the merged board, which this checks by rendering the same footer two ways:
// once with the score in a side-car and once with it in the house file.
func TestThePickerFooterShowsTheMergedBoard(t *testing.T) {
	merged := board("Ada", 4200, 9, "")

	withHook, hf := shellOver(t, []string{"Slumberland"})
	withHook.lib.Houses[0].Scores = board("Bob", 900, 2, "")
	withHook.host.Scores = func(House) house.Scores { return merged }
	withHook.openPicker()
	withHook.Draw()

	inFile, ff := shellOver(t, []string{"Slumberland"})
	inFile.lib.Houses[0].Scores = merged
	inFile.openPicker()
	inFile.Draw()

	// The footer's baseline, and the rows a 5x7 glyph with a shadow can reach.
	for v := pickFootV - render.FontTall; v <= pickFootV+2; v++ {
		for x := 0; x < screenWide; x++ {
			if hf.scr.Pix[v*screenWide+x] != ff.scr.Pix[v*screenWide+x] {
				t.Fatalf("the footer differs at (%d,%d): the picker is showing the house "+
					"file's best score rather than the merged board's", x, v)
			}
		}
	}

	// And the two boards really are different, so the comparison above is not comparing a
	// footer with itself.
	if who, _, _ := bestOf(withHook.lib.Houses[0].Scores); who != "Bob" {
		t.Fatalf("the fixture's house file says %q, want Bob", who)
	}
}

func TestBestOfIgnoresEmptyRows(t *testing.T) {
	var b house.Scores
	if _, _, ok := bestOf(b); ok {
		t.Error("an empty board has a best score")
	}
	// A row with a name and no score is empty: the original tests the score (7.9.1).
	b.Names[4].SetText("Nobody")
	if _, _, ok := bestOf(b); ok {
		t.Error("a named row with a zero score counted as a score")
	}
}
