package game

// The two endings, and the three things a test can hold them to.
//
// **They must end.** Both are `while` loops with no frame counter, ended by a condition the
// animation itself has to reach -- the angel leaving the screen, eight sheets of paper
// sticking -- and a transcription error in either termination test is a hang rather than a
// wrong pixel. That is the one bug in this file's C that a reader cannot see and a test can,
// so every case here runs a whole animation to its end with no clock and no keyboard.
//
// **They must end the play loop.** `playing = false` is what stops PlayGame calling the
// ending again on the next frame; the win path does it first and the loss path last.
//
// **The tail differs between them and between a game and a demo**, in a way that decides
// whether the attract mode can post high scores. Those four combinations are the rest of the
// file.
//
// What is *drawn* is not pinned here. With no extracted art every plate is nil, so the
// starfield, the paper, the letters and the angel are all absent and what remains is the
// dirty-rect protocol -- which render_frame_test.go already covers, at the level where a
// rect list is the subject rather than a side effect. Pinning the composition of these two
// screens is 1.8's job and needs the 1994 PICTs to be worth anything.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// gameOverWorld is a world mid-game: a trailer to draw, a score to offer, and three
// distinguishable surface fills.
//
// Main is filled rather than left white because DoDiedGameOver *reads it back* -- the room
// the player died in is on the window and nowhere else -- so a fill is standing in for the
// last frame of a lost game.
func gameOverWorld(t *testing.T, trailer string) *World {
	t.Helper()
	h := oneRoomHouse(house.ObjectIsEmpty)
	h.Trailer.SetText(trailer)
	w := newTestWorld(h, "", "")
	w.Playing = true
	w.Score = 12345
	fillSurface(w.R.Work, 8)
	fillSurface(w.R.Back, 77)
	fillSurface(w.Main, 200)
	return w
}

// countTicks is how many waits of one shape the log holds. The animations' pacing is a
// stream of two-tick polls, so their length is measurable as a count.
func countTicks(l *waitLog, ticks int64, discard bool) int {
	n := 0
	for _, c := range l.calls {
		if c == (waitCall{ticks, discard}) {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// The win ending
// ---------------------------------------------------------------------------

// TestDoGameOverEndsTheGameAndRestoresTheSplash walks the whole win animation with a host
// that never reports input, which is the case the fidelity corpus runs: 369 passes for the
// angel to cross a 640-wide screen two pixels at a time, then 80 more, then five seconds.
//
// The five seconds are asserted because they are the only wait in the animation measured in
// seconds, and the C's `WaitForInputEvent(5)` is 300 ticks rather than 5 (Utilities.c:445).
func TestDoGameOverEndsTheGameAndRestoresTheSplash(t *testing.T) {
	w := gameOverWorld(t, "Thanks for playing")
	var log waitLog
	w.Wait = log.hook()

	w.DoGameOver()

	if w.Playing {
		t.Error("Playing is still true; PlayGame would run the ending again next frame")
	}

	// The angel starts at left = -96 and is drawn while left <= WorkRect.Right+2 = 642,
	// advancing two pixels a pass, and every pass it is on screen resets the trailing
	// count: 370 passes, inclusive of both ends. Then the counter climbs to 80, and the
	// pass that finds it there is the one that waits and leaves -- so 80 more, not 79.
	// One poll per pass throughout.
	if got, want := countTicks(&log, 2, false), 370+80; got != want {
		t.Errorf("%d two-tick polls, want %d: the angel's crossing or the trailing count is wrong", got, want)
	}
	if got := log.blocking(); len(got) == 0 || got[len(got)-1] != 5*60 {
		t.Errorf("the animation's last wait is %v, want 300 ticks (five seconds)", got[len(got)-1:])
	}

	// The starfield and the trailer were promoted into Back and are still there; the work
	// map has been given over to the splash screen. Both fills are gone, and they are
	// different fills, so neither result can be the other's.
	if n := countIndex(w.R.Back, w.R.V.WorkRect, 77); n != 0 {
		t.Errorf("%d pixels of the old background survived; the starfield was not promoted", n)
	}
	if n, all := countIndex(w.R.Work, w.R.V.WorkRect, render.Black8), 640*460; n != all {
		t.Errorf("%d of %d work-map pixels are black; the splash screen was not restored", n, all)
	}
}

// TestDoGameOverStopsWhenThePlayerTouchesSomething: one poll reporting input ends the whole
// animation, on the pass it arrives.
//
// It is worth pinning because the flag is read once per pass and the C's loop condition is
// the only thing that acts on it -- an ending that finished its 449 passes anyway would look
// identical on a machine nobody is sitting at.
func TestDoGameOverStopsWhenThePlayerTouchesSomething(t *testing.T) {
	w := gameOverWorld(t, "Thanks for playing")
	log := waitLog{answer: Waited{Input: true}}
	w.Wait = log.hook()

	w.DoGameOver()

	if got := countTicks(&log, 2, false); got != 1 {
		t.Errorf("%d two-tick polls, want 1: the animation ran on after the player asked it to stop", got)
	}
	if n := countTicks(&log, 5*60, false); n != 0 {
		t.Error("the five-second wait ran after an abort")
	}
	if w.Playing {
		t.Error("Playing is still true after an aborted ending")
	}
}

// TestDoGameOverStopsWhenTheWindowCloses is the same test through the other door. Quitting is
// checked alongside the poll's answer because a host that reports a closed window through
// World.Quitting rather than through Waited must still be able to stop an animation -- see
// cmd/glidergo's Wait hook, which sets the flag directly.
func TestDoGameOverStopsWhenTheWindowCloses(t *testing.T) {
	w := gameOverWorld(t, "Thanks for playing")
	var log waitLog
	w.Wait = log.hook()
	w.Quitting = true

	w.DoGameOver()

	if got := countTicks(&log, 2, false); got != 1 {
		t.Errorf("%d two-tick polls, want 1: a closed window did not end the animation", got)
	}
}

// TestDoGameOverSkipsTheSplashWhenAScoreQualifies: TestHighScore answering true means the
// host is about to draw a board over the window, so the splash restore is skipped
// (GameOver.c:68). The loss path now does the same thing, which it did not in 1994 -- see
// docs/IMPROVEMENTS.md 2.60.
//
// That the hook is called at all, on both paths, is highscore_test.go's -- it was written
// against these two functions while they were still stubs. What is new here is what the
// answer decides.
func TestDoGameOverSkipsTheSplashWhenAScoreQualifies(t *testing.T) {
	w := gameOverWorld(t, "Thanks for playing")
	var log waitLog
	w.Wait = log.hook()

	offered := 0
	w.HighScore = func(score int32, rooms int16) bool {
		offered++
		if score != 12345 {
			t.Errorf("the board was offered score %d, want 12345", score)
		}
		return true
	}

	w.DoGameOver()

	if offered != 1 {
		t.Errorf("the high-score hook was called %d times, want 1", offered)
	}
	// The splash restore fills the work map black. It did not run, so the starfield the
	// animation composed is still there -- which is what the player is looking at while
	// they type their name.
	if n, all := countIndex(w.R.Work, w.R.V.WorkRect, render.Black8), 640*460; n == all {
		t.Error("the splash screen was restored over a high-score board")
	}
}

// ---------------------------------------------------------------------------
// The loss ending
// ---------------------------------------------------------------------------

// TestDoDiedGameOverTerminates is the hang test.
//
// Eight sheets of paper flutter down and stick when `dest.bottom + RandomInt(8) > stopPages`,
// which is a random test applied to every page every pass -- so the loop's length is not a
// constant and its *termination* is the property worth asserting. A cap is used rather than
// an equality: what matters is that it ends, and ends in a plausible number of passes rather
// than in millions.
func TestDoDiedGameOverTerminates(t *testing.T) {
	w := gameOverWorld(t, "Thanks for playing")
	var log waitLog
	w.Wait = log.hook()

	w.DoDiedGameOver()

	if w.Playing {
		t.Error("Playing is still true; PlayGame would run the ending again next frame")
	}
	passes := countTicks(&log, 2, false)
	if passes < 8 || passes > 2000 {
		t.Errorf("the animation took %d passes; expected tens, not that", passes)
	}

	// The room the player died in was read back off the window into both offscreens
	// (GameOver.c:449-450), which is what the pages flutter over. Back still holds it;
	// the work map has been given to the splash screen.
	if n, all := countIndex(w.R.Back, w.R.V.WorkRect, 200), 640*460; n != all {
		t.Errorf("%d of %d background pixels came from the window; the grab is incomplete", n, all)
	}
	if n, all := countIndex(w.R.Work, w.R.V.WorkRect, render.Black8), 640*460; n != all {
		t.Errorf("%d of %d work-map pixels are black; the splash screen was not restored", n, all)
	}
}

// TestDoDiedGameOverWaitsTenSecondsOrOneInADemo pins the loss path's tail, whose two halves
// are one difference: a demo has to get back to attracting somebody, so it looks at the dead
// glider for a second instead of ten and never touches the board.
//
// The durations are the new part -- ten seconds is 600 ticks, not 10 -- and the board offer is
// asserted alongside them because the two must travel together: a demo that waited a second
// and then posted a score would fill an unattended machine's board with its own games.
// highscore_test.go covers the offer on its own.
func TestDoDiedGameOverWaitsTenSecondsOrOneInADemo(t *testing.T) {
	for _, c := range []struct {
		name       string
		demo       bool
		wantWait   int64
		wantOffers int
	}{
		{"a game", false, 10 * 60, 1},
		{"a demo", true, 1 * 60, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := gameOverWorld(t, "Thanks for playing")
			w.DemoGoing = c.demo
			var log waitLog
			w.Wait = log.hook()

			offers := 0
			w.HighScore = func(int32, int16) bool { offers++; return false }

			w.DoDiedGameOver()

			if offers != c.wantOffers {
				t.Errorf("the high-score hook was called %d times, want %d", offers, c.wantOffers)
			}
			if n := countTicks(&log, c.wantWait, false); n != 1 {
				t.Errorf("%d waits of %d ticks, want 1: %v", n, c.wantWait, log.blocking())
			}
		})
	}
}

// TestDoDiedGameOverAbortSkipsTheWaitButNotTheBoard: `if (!userAborted)` guards the ten
// seconds and nothing else, so a player who cut the animation short is still offered the
// board they earned.
//
// This is the one place the two endings disagree about what an abort means, and it is the
// right way round: the animation is a flourish and the board is the reward.
func TestDoDiedGameOverAbortSkipsTheWaitButNotTheBoard(t *testing.T) {
	w := gameOverWorld(t, "Thanks for playing")
	log := waitLog{answer: Waited{Input: true}}
	w.Wait = log.hook()

	offers := 0
	w.HighScore = func(int32, int16) bool { offers++; return false }

	w.DoDiedGameOver()

	if got := countTicks(&log, 2, false); got != 1 {
		t.Errorf("%d two-tick polls, want 1: the animation ran on after the abort", got)
	}
	if n := countTicks(&log, 10*60, false); n != 0 {
		t.Error("the ten-second wait ran after an abort")
	}
	if offers != 1 {
		t.Errorf("the high-score hook was called %d times, want 1: an abort must not cost the player their score", offers)
	}
}
