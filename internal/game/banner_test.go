package game

// The two screens Banner.c draws, and the wait protocol underneath them.
//
// Both halves are testable and neither is testable the same way. What is drawn is this
// package's business and is checked in pixels: the page lands centred, the message reaches
// the *screen* and not just the offscreen the original left it in, the star line is red and
// opt-in. How long it stays up is the host's, and is checked through a recording Wait hook
// -- because the one thing a port of a blocking wait can get wrong without any visible
// symptom is the unit, and Utilities.c:445 multiplies by sixty (docs/IMPROVEMENTS.md 2.32).
//
// The art is deliberately absent. render.Assets with no directory resolves every Plate to
// nil, loadScaledGraphic draws nothing, and what is left is exactly the part of these
// screens the port composes rather than loads. A test that needed the 1994 PICTs would be a
// test of the extractor.

import (
	"strings"
	"testing"

	"glidergo/internal/house"
	"glidergo/internal/render"
)

// waitLog records what the game asked the host to wait for.
//
// The pair is World.Wait's own: a tick count and whether input during it is dropped. Every
// wait in the game is one of four shapes -- (0,true) for a flush, (n,true) for a Delay,
// (60*s,false) for WaitForInputEvent, (2,false) for an animation frame -- so a log of the
// calls is a complete statement of a screen's pacing.
type waitLog struct {
	calls  []waitCall
	answer Waited // what every wait reports; the zero value is "the deadline expired"
}

type waitCall struct {
	ticks   int64
	discard bool
}

func (l *waitLog) hook() func(int64, bool) Waited {
	return func(ticks int64, discard bool) Waited {
		l.calls = append(l.calls, waitCall{ticks, discard})
		return l.answer
	}
}

// blocking returns just the waits that can actually be ended by the player, which is the
// part of the log a reader of Banner.c would recognise as its durations.
func (l *waitLog) blocking() []int64 {
	var out []int64
	for _, c := range l.calls {
		if !c.discard {
			out = append(out, c.ticks)
		}
	}
	return out
}

// bannerWorld is a world whose three surfaces are three distinguishable fills, so that
// "drawn", "restored" and "not touched at all" are three different answers.
func bannerWorld(t *testing.T, banner string, stars int16, starCount bool) *World {
	t.Helper()
	h := oneRoomHouse(house.ObjectIsEmpty)
	h.Banner.SetText(banner)
	if !starCount {
		h.Flags |= house.FlagNoStarCount
	}
	w := newTestWorld(h, "", "")
	w.StarsLeft = stars
	fillSurface(w.R.Work, 8)
	fillSurface(w.R.Back, 77)
	fillSurface(w.Main, 200)
	return w
}

// ---------------------------------------------------------------------------
// linesOfText
// ---------------------------------------------------------------------------

// TestLinesOfText covers the three separators and the two shapes the shipped corpus
// contains. The carriage return is the only one any 1994 house can hold; the other two are
// there for Stage 2's text-authored houses (see linesOfText).
func TestLinesOfText(t *testing.T) {
	for _, c := range []struct {
		name string
		in   string
		want []string
	}{
		{"one line", "Welcome to my house", []string{"Welcome to my house"}},
		{"carriage returns", "one\rtwo\rthree", []string{"one", "two", "three"}},
		{"line feeds", "one\ntwo", []string{"one", "two"}},
		{"crlf", "one\r\ntwo", []string{"one", "two"}},
		{"empty", "", []string{""}},
		{"trailing return", "one\r", []string{"one", ""}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := linesOfText(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("linesOfText(%q) = %q, want %q", c.in, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("linesOfText(%q) = %q, want %q", c.in, got, c.want)
				}
			}
		})
	}

	// The separator must not survive into the line. GetLineOfText returns it
	// (StringUtils.c:140-208), which on a Mac drew a glyph for 0x0D at the end of every
	// line but the last, and a port that kept it would draw one here too.
	for _, line := range linesOfText("one\rtwo") {
		if strings.ContainsAny(line, "\r\n") {
			t.Errorf("line %q still holds its separator", line)
		}
	}
}

// ---------------------------------------------------------------------------
// BringUpBanner
// ---------------------------------------------------------------------------

// TestDrawBannerCentresThePage pins the rect every other banner test measures against.
//
// 330x220 centred in 640x480 is (155,130)-(485,350), and it is a constant rather than a
// computation because CenterRectInRect's (Bwide-Awide)/2 and the more obvious
// Bwide/2-Awide/2 agree here and disagree by a pixel on odd differences (render.CenterIn).
func TestDrawBannerCentresThePage(t *testing.T) {
	w := bannerWorld(t, "hello", 0, false)
	if got, want := w.DrawBanner(), render.SetRect(155, 130, 485, 350); got != want {
		t.Errorf("DrawBanner() = %+v, want %+v", got, want)
	}
}

// TestBringUpBannerReachesTheScreen is the test for the line the original is missing.
//
// The shipped BringUpBanner composes the whole banner into workSrcMap and then copies the
// clean background back over it, with both DissBits calls commented out -- so the player
// sees a room for fifteen seconds (docs/analysis/rendering.md 15.5). This port adds the
// blit, and this pins all three consequences of adding it: the message is on Main, the work
// map is clean again afterwards, and the clean pixels came from Back rather than from a
// fill.
func TestBringUpBannerReachesTheScreen(t *testing.T) {
	w := bannerWorld(t, "Welcome", 0, false)
	var log waitLog
	w.Wait = log.hook()

	presents := 0
	w.Present = func() { presents++ }

	page := render.SetRect(155, 130, 485, 350)
	w.BringUpBanner()

	if n := countIndex(w.Main, page, render.Black8); n == 0 {
		t.Error("no black pixels inside the page on Main: the banner never reached the screen")
	}
	if presents != 1 {
		t.Errorf("presents = %d, want 1", presents)
	}

	// The erase, and that it is a restore. Back is filled with 77, so a work map that
	// still holds 8 was never copied over and one that holds 255 kept the text.
	if n := countIndex(w.R.Work, page, 77); n != int(page.Wide())*int(page.Tall()) {
		t.Errorf("%d of %d pixels inside the page came from Back; the erase is not a restore",
			n, int(page.Wide())*int(page.Tall()))
	}

	// Outside the page nothing moved. The banner is a 330x220 sheet and the room around it
	// is the room the player is about to be dropped into.
	if n := countIndex(w.Main, render.SetRect(0, 0, 100, 100), 200); n != 100*100 {
		t.Errorf("%d of 10000 pixels outside the page still hold the screen fill", n)
	}
}

// TestBringUpBannerWaitsInSeconds is the unit test in the literal sense.
//
// WaitForInputEvent's parameter is seconds and its deadline is `TickCount() + 60 * seconds`,
// so the banner's 15 is 900 ticks and a demo's 4 is 240. A port that read the parameter as
// ticks would show the banner for a quarter of a second and nothing about the screen would
// look wrong -- which is why this is checked as a number and not as an appearance.
func TestBringUpBannerWaitsInSeconds(t *testing.T) {
	for _, c := range []struct {
		name string
		demo bool
		want int64
	}{
		{"a game", false, 15 * 60},
		{"a demo", true, 4 * 60},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := bannerWorld(t, "Welcome", 0, false)
			w.DemoGoing = c.demo
			var log waitLog
			w.Wait = log.hook()

			w.BringUpBanner()

			got := log.blocking()
			if len(got) != 1 || got[0] != c.want {
				t.Fatalf("blocking waits = %v ticks, want [%d]", got, c.want)
			}
			// FlushEvents at both ends of the wait, which is what consumes the keystroke
			// that dismissed the banner so it cannot also steer the glider.
			if n := len(log.calls); n != 3 {
				t.Errorf("%d waits in total, want 3 (flush, wait, flush): %+v", n, log.calls)
			}
			if log.calls[0] != (waitCall{0, true}) || log.calls[2] != (waitCall{0, true}) {
				t.Errorf("the wait is not flushed at both ends: %+v", log.calls)
			}
		})
	}
}

// TestBringUpBannerWithoutAHostDoesNotWait: a nil hook is a headless build, and every wait
// in the game has to be over before it starts or a replay cannot cross this screen. The
// drawing still happens -- which is what makes a -shot of the banner possible at all.
func TestBringUpBannerWithoutAHostDoesNotWait(t *testing.T) {
	w := bannerWorld(t, "Welcome", 0, false)
	w.BringUpBanner()
	if n := countIndex(w.Main, render.SetRect(155, 130, 485, 350), render.Black8); n == 0 {
		t.Error("a headless banner drew nothing; -shot would capture an empty room")
	}
}

// ---------------------------------------------------------------------------
// DrawBannerMessage
// ---------------------------------------------------------------------------

// TestBannerStarCountIsOptInAndInverted covers the flag and the singular/plural pair.
//
// The flag is house.FlagNoStarCount and it is *clear* to show the count (HouseIO.c:418),
// which is the sort of inversion a port gets backwards silently: twelve of the twenty
// shipped houses would gain two lines of red text nobody wrote. The count line is red --
// ForeColor(redColor), render.QDRed -- so counting red pixels inside the page is the whole
// test.
func TestBannerStarCountIsOptInAndInverted(t *testing.T) {
	page := render.SetRect(155, 130, 485, 350)

	for _, c := range []struct {
		name      string
		stars     int16
		starCount bool
		wantRed   bool
		wantText  string
	}{
		{"seven stars", 7, true, true, "There are 7 stars in the house."},
		{"one star", 1, true, true, "There is 1 star in the house."},
		{"no stars", 0, true, true, "There are 0 stars in the house."},
		{"flag set", 7, false, false, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := bannerWorld(t, "Welcome", c.stars, c.starCount)
			w.DrawBannerMessage(w.DrawBanner())

			red := countIndex(w.R.Work, page, render.QDRed)
			if c.wantRed && red == 0 {
				t.Error("no red pixels inside the page: the star count is missing")
			}
			if !c.wantRed && red != 0 {
				t.Errorf("%d red pixels inside the page, want none: FlagNoStarCount is inverted", red)
			}
			if !c.wantRed {
				return
			}

			// The plural is picked by `!= 1`, so zero stars reads "There are 0 stars".
			// Pinned as a string because the pixels cannot tell the two apart and
			// because the wording is a localized resource the port has inlined
			// (STR# 150, indices 1-4).
			want := strThereAre + "0" + strStarsInHouse
			switch c.stars {
			case 1:
				want = strThereIs + "1" + strStarInHouse
			case 7:
				want = strThereAre + "7" + strStarsInHouse
			}
			if want != c.wantText {
				t.Fatalf("the test's own expectation is inconsistent: %q vs %q", want, c.wantText)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DisplayStarsRemaining
// ---------------------------------------------------------------------------

// starsPanelRect is 256x64 centred on the screen and lifted twenty rows (Banner.c:212-215).
var starsPanelRect = render.SetRect(192, 188, 448, 252)

// snapshotOnPresent captures Main the first time the game shows it.
//
// Necessary rather than convenient: DisplayStarsRemaining draws the panel, presents, waits,
// and then restores the room over it, so by the time the call returns there is nothing left
// to look at. What the player saw is the first present, and this is the only way a test can
// see it too.
func snapshotOnPresent(w *World, presents *int) *[]uint8 {
	shot := new([]uint8)
	w.Present = func() {
		*presents++
		if *presents == 1 {
			*shot = append([]uint8(nil), w.Main.Pix...)
		}
	}
	return shot
}

// TestDisplayStarsRemainingDrawsTheCountOnlyInThePlural covers both of Banner.c's oddities
// at once: the plate is chosen on `< 2` rather than `== 1`, and the number is drawn in one
// arm of the branch only.
//
// The singular plate spells the count out in its artwork ("Last star!"), so there is nothing
// to place on it; the C computes the string before the branch anyway, which is what makes
// the omission look like a bug rather than a choice. Palette index 4 (#FFFF33) is what
// ColorText(theStr, 4L) selects and is the only place in the game that names it.
func TestDisplayStarsRemainingDrawsTheCountOnlyInThePlural(t *testing.T) {
	for _, c := range []struct {
		name  string
		stars int16
		want  bool
	}{
		{"seven left", 7, true},
		{"two left", 2, true},
		{"one left", 1, false},
		{"none left", 0, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := bannerWorld(t, "Welcome", c.stars, true)
			var log waitLog
			w.Wait = log.hook()
			presents := 0
			shot := snapshotOnPresent(w, &presents)

			w.DisplayStarsRemaining()

			panel := &render.Surface{W: w.Main.W, H: w.Main.H, Pix: *shot}
			n := countIndex(panel, starsPanelRect, 4)
			if c.want && n == 0 {
				t.Error("no index-4 pixels in the panel: the count was not drawn")
			}
			if !c.want && n != 0 {
				t.Errorf("%d index-4 pixels in the panel, want none: the singular plate carries its own number", n)
			}
		})
	}
}

// TestDisplayStarsRemainingRestoresTheRoom: the panel is drawn into Main and the work map
// still holds the room underneath it, so the tail's CopyRectWorkToMain is a restore. That
// is the same arrangement the pause placard uses and the reason neither needs a saved
// bitmap.
func TestDisplayStarsRemainingRestoresTheRoom(t *testing.T) {
	w := bannerWorld(t, "Welcome", 7, true)
	var log waitLog
	w.Wait = log.hook()
	presents := 0
	snapshotOnPresent(w, &presents)

	w.DisplayStarsRemaining()

	if n, all := countIndex(w.Main, starsPanelRect, 8), 256*64; n != all {
		t.Errorf("%d of %d pixels under the panel came from the work map; the restore is incomplete", n, all)
	}
	// Once for the panel, once for the restore.
	if presents != 2 {
		t.Errorf("presents = %d, want 2", presents)
	}

	// One second with input ignored, then thirty seconds waiting for it -- 60 ticks and
	// 1800, and the first of the two is the one a player cannot skip.
	want := []waitCall{{60, true}, {0, true}, {1800, false}, {0, true}}
	if len(log.calls) != len(want) {
		t.Fatalf("waits = %+v, want %+v", log.calls, want)
	}
	for i := range want {
		if log.calls[i] != want[i] {
			t.Fatalf("waits = %+v, want %+v", log.calls, want)
		}
	}
}

// TestDisplayStarsRemainingRebuildsAfterAResume is the one place in the game that reads what
// a wait returned.
//
// WaitForInputEvent answers `didResume` and not "input ended it", and the reason it is asked
// here is that another application having been in front of the window means the screen is
// gone rather than merely stale -- so the answer is a full rebuild. A port that read the
// Boolean as "the player pressed a key" would rebuild the screen on every dismissal, which
// is invisible on a Mac and a visible flash here.
func TestDisplayStarsRemainingRebuildsAfterAResume(t *testing.T) {
	for _, resumed := range []bool{false, true} {
		name := "dismissed"
		if resumed {
			name = "resumed"
		}
		t.Run(name, func(t *testing.T) {
			w := bannerWorld(t, "Welcome", 7, true)
			log := waitLog{answer: Waited{Resumed: resumed}}
			w.Wait = log.hook()
			presents := 0
			snapshotOnPresent(w, &presents)

			w.DisplayStarsRemaining()

			// Outside the panel: untouched on a dismissal, rebuilt on a resume.
			// RestoreEntireGameScreen fills the whole screen black before it redraws,
			// so the screen fill cannot survive it.
			outside := render.SetRect(0, 0, 100, 100)
			kept := countIndex(w.Main, outside, 200)
			if resumed && kept != 0 {
				t.Errorf("%d pixels of the old screen survived a resume; the screen was not rebuilt", kept)
			}
			if !resumed && kept != 100*100 {
				t.Errorf("only %d of 10000 pixels survived a dismissal; the screen was rebuilt for the wrong reason", kept)
			}
		})
	}
}
