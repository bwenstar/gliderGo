package main

// The host's high-score flow, driven through a fake window.
//
// What is worth testing here is not the drawing -- internal/scores tests that pixel by pixel
// -- but the policy this file adds on top of it: who gets asked for a name, what gets written
// to disk, what the preferences remember, and that a closed window gets out of a modal dialog
// that has no Cancel button.
//
// The three loops are timed in ticks, so these tests take about a second each: the board's
// hold (DelayTicks(60)) is unconditional in the original and reproduced here, and a test that
// skipped it would not be testing the code that ships.

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bwenstar/gliderGo/internal/game"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/prefs"
	"github.com/bwenstar/gliderGo/internal/render"
	"github.com/bwenstar/gliderGo/internal/scores"
)

// fakeWin is a platform.Window that hands out a scripted event batch per poll and counts
// presents. Past the end of the script it reports the window closed, so a loop that is
// waiting for something the script never sends fails instead of hanging.
type fakeWin struct {
	script [][]platform.Event
	pass   int

	presents int
	held     map[platform.Key]bool

	// onPoll runs before each batch is handed out, with the pass about to be served in
	// f.pass. It is how a test presses a key that is *polled* rather than delivered: the
	// pause key is read through KeyDown and the give-up keys arrive as events, and the pause
	// loop reads both in the same pass, so holding one for exactly some of the passes cannot
	// be expressed by the script alone. See saved_test.go.
	onPoll func(f *fakeWin)
}

func (f *fakeWin) Present(*platform.Framebuffer) error { f.presents++; return nil }

func (f *fakeWin) PollEvents() []platform.Event {
	if f.onPoll != nil {
		f.onPoll(f)
	}
	if f.pass >= len(f.script) {
		f.pass++
		return []platform.Event{{Kind: platform.EventQuit}}
	}
	evs := f.script[f.pass]
	f.pass++
	return evs
}

func (f *fakeWin) KeyDown(k platform.Key) bool { return f.held[k] }
func (f *fakeWin) SetTitle(string) error       { return nil }
func (f *fakeWin) Close() error                { return nil }

// typed is one batch: a run of characters. The x11 backend fills Text from XLookupString,
// which is what a real keyboard arrives as.
func typed(s string) []platform.Event {
	var evs []platform.Event
	for _, r := range s {
		evs = append(evs, platform.Event{
			Kind: platform.EventKeyDown,
			Key:  platform.KeyUnknown,
			Text: string(r),
		})
	}
	return evs
}

func pressed(k platform.Key) []platform.Event {
	return []platform.Event{{Kind: platform.EventKeyDown, Key: k}}
}

// scoreApp is an app with a window, a store in a temporary directory and no audio.
func scoreApp(t *testing.T, script ...[]platform.Event) (*app, *fakeWin, *game.World) {
	t.Helper()
	o := &options{scoresDir: t.TempDir()}
	p := prefs.Default()
	a := newApp(o, p, false)

	win := &fakeWin{script: script, held: map[platform.Key]bool{}}
	a.win = win
	a.fb = platform.NewFramebuffer(640, 480)

	h := &house.House{NRooms: 1, Rooms: []house.Room{{}}}
	w := &game.World{H: h, Main: render.NewSurface(640, 480)}
	return a, win, w
}

// keyEventually is a script that presses a key on every poll, which is how the board's
// thirty-second wait is ended: the unconditional one-second hold swallows whatever arrives
// during it, so a single scripted press would be eaten.
func keyEventually(n int) [][]platform.Event {
	var s [][]platform.Event
	for i := 0; i < n; i++ {
		s = append(s, pressed(platform.KeySpace))
	}
	return s
}

// A score that does not make the board asks nothing, writes nothing and says nothing. The
// original has an alert for this and never shows it (docs/analysis/scoring.md 7.12).
func TestAScoreThatDoesNotQualifyIsSilent(t *testing.T) {
	a, win, w := scoreApp(t)
	for i := range w.H.HighScores.Scores {
		w.H.HighScores.Scores[i] = 10000
	}

	if a.highScoreHook(w, "Slumberland", nil, new(bool))(500, 3) {
		t.Error("a score below every row on the board qualified")
	}
	if win.presents != 0 {
		t.Errorf("%d frames were presented for a score that did not qualify", win.presents)
	}
	if _, err := os.Stat(a.store.Path("Slumberland")); !os.IsNotExist(err) {
		t.Errorf("a file was written for a score that did not qualify: %v", err)
	}
}

// The ordinary path: type a name, press Return, accept the banner (an empty board means
// first place, so the second dialog comes up), see the board, press a key.
func TestAQualifyingScoreIsTypedInAndRecorded(t *testing.T) {
	script := append([][]platform.Event{
		typed("Ozma"),
		pressed(platform.KeyReturn),
		pressed(platform.KeyReturn),
	}, keyEventually(200)...)

	a, _, w := scoreApp(t, script...)
	closed := false
	if !a.highScoreHook(w, "Slumberland", nil, &closed)(4200, 12) {
		t.Fatal("a score onto an empty board did not qualify")
	}
	if closed {
		t.Error("the window was reported closed")
	}

	// What went to disk, read back through the same door the shell reads it through.
	b, notes := a.store.Load("Slumberland", house.Scores{})
	if len(notes) != 0 {
		t.Errorf("the board this program just wrote came back with repairs: %v", notes)
	}
	if got := b.Names[0].Text(); got != "Ozma" {
		t.Errorf("row 0's name is %q, want Ozma", got)
	}
	if b.Scores[0] != 4200 {
		t.Errorf("row 0's score is %d, want 4200", b.Scores[0])
	}
	if b.Levels[0] != 12 {
		t.Errorf("row 0's room count is %d, want 12", b.Levels[0])
	}
	if b.TimeStamps[0] == 0 {
		t.Error("row 0 has no timestamp")
	}
	if got := scores.Occupied(&b); got != 1 {
		t.Errorf("the board holds %d rows, want 1", got)
	}

	// And the name is remembered for the next game, which is the original's `highName`
	// (7.5 note 5). Not written to disk here: this app has canSave false.
	if a.p.HighName != "Ozma" {
		t.Errorf("the remembered name is %q, want Ozma", a.p.HighName)
	}
}

// First place, and only first place, is asked for a banner (HighScores.c:404-408).
func TestOnlyFirstPlaceIsAskedForABanner(t *testing.T) {
	for _, tc := range []struct {
		name       string
		existing   int32 // a score already on the board
		score      int32
		wantBanner string
	}{
		{"first", 100, 4200, "you can't win"},
		{"second", 9000, 4200, prefs.Default().HighBanner},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := append([][]platform.Event{
				typed("Ozma"),
				pressed(platform.KeyReturn),
				typed("you can't win"),
				pressed(platform.KeyReturn),
			}, keyEventually(200)...)

			a, _, w := scoreApp(t, script...)
			w.H.HighScores.Scores[0] = tc.existing
			w.H.HighScores.Names[0].SetText("Somebody")
			scores.SetBanner(&w.H.HighScores, prefs.Default().HighBanner)

			if !a.highScoreHook(w, "House", nil, new(bool))(tc.score, 5) {
				t.Fatal("the score did not qualify")
			}
			b, _ := a.store.Load("House", house.Scores{})
			if got := b.Banner.Text(); got != tc.wantBanner {
				t.Errorf("the banner is %q, want %q", got, tc.wantBanner)
			}
		})
	}
}

// The name prompt starts on the remembered name, selected, so a player who wants the same
// name as last time presses Return and a player who does not just types.
func TestReturnAloneKeepsTheRememberedName(t *testing.T) {
	script := append([][]platform.Event{
		pressed(platform.KeyReturn), // the name: keep it
		pressed(platform.KeyReturn), // the banner: first place on an empty board
	}, keyEventually(200)...)
	a, _, w := scoreApp(t, script...)
	a.p.HighName = "Kim"

	a.highScoreHook(w, "House", nil, new(bool))(4200, 5)

	b, _ := a.store.Load("House", house.Scores{})
	if got := b.Names[0].Text(); got != "Kim" {
		t.Errorf("row 0's name is %q, want the remembered Kim", got)
	}
}

// A closed window gets out of a dialog that has no Cancel button, and tells play.go what
// happened -- otherwise the shell comes back up and draws a title screen into a dead window.
func TestAClosedWindowLeavesTheNamePrompt(t *testing.T) {
	a, _, w := scoreApp(t) // an empty script: the first poll reports the window closed
	closed := false

	done := make(chan bool, 1)
	go func() { done <- a.highScoreHook(w, "House", nil, &closed)(4200, 5) }()
	select {
	case got := <-done:
		if !got {
			t.Error("the score qualified, so the hook should still answer true")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the hook did not return: a closed window cannot get out of the dialog")
	}

	if !closed {
		t.Error("play.go's closed flag was not set")
	}
	if !w.Quitting {
		t.Error("the world was not told to quit")
	}
	// The score is still recorded. It was earned before the window went away, and the name
	// is the least important part of it.
	b, _ := a.store.Load("House", house.Scores{})
	if b.Scores[0] != 4200 {
		t.Errorf("row 0's score is %d; a closed window lost the score", b.Scores[0])
	}
}

// -scores none plays and records nothing, and the game does not care: a nil store is a
// supported state everywhere (internal/scores.Store).
func TestScoresNoneRecordsNothing(t *testing.T) {
	script := append([][]platform.Event{
		pressed(platform.KeyReturn), pressed(platform.KeyReturn),
	}, keyEventually(200)...)
	a, _, w := scoreApp(t, script...)
	dir := a.store.Dir()
	a.store = nil

	if !a.highScoreHook(w, "House", nil, new(bool))(4200, 5) {
		t.Error("a qualifying score was refused because there was nowhere to write it")
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) != 0 {
		t.Errorf("%d files were written with no store: %v", len(entries), entries)
	}
	if _, err := os.Stat(filepath.Join(dir, "House"+scores.Ext)); !os.IsNotExist(err) {
		t.Errorf("a board file exists: %v", err)
	}
}

// The board's own board is never written to. The side-car is the write path; the house file
// is a read-only input (internal/scores' package comment).
func TestTheHousesOwnBoardIsNotModified(t *testing.T) {
	script := append([][]platform.Event{
		typed("Ozma"), pressed(platform.KeyReturn),
		typed("mine now"), pressed(platform.KeyReturn),
	}, keyEventually(200)...)

	a, _, w := scoreApp(t, script...)
	w.H.HighScores.Names[0].SetText("Somebody Else")
	w.H.HighScores.Scores[0] = 100
	before := house.EncodeScores(&w.H.HighScores)

	a.highScoreHook(w, "House", nil, new(bool))(4200, 5)

	if after := house.EncodeScores(&w.H.HighScores); string(after) != string(before) {
		t.Error("the house's own high-score table was modified")
	}
	// And the side-car did get it, seeded from the house's own row.
	b, _ := a.store.Load("House", w.H.HighScores)
	if b.Names[0].Text() != "Ozma" || b.Names[1].Text() != "Somebody Else" {
		t.Errorf("the side-car reads %q then %q; want Ozma above Somebody Else",
			b.Names[0].Text(), b.Names[1].Text())
	}
}
