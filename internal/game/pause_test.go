package game

// The pause placard, and the two halves of it that can go wrong independently: what is
// drawn (this package's business) and how long it stays up (the host's).
//
// The host half is exercised here with hooks that stand in for cmd/glidergo's wait loop --
// one that paints once and returns, one that paints several times, one that re-enters -- so
// that the *contract* on World.Pause is tested rather than any one host's answer to it.
// The three C release loops themselves are cmd/glidergo's and are not reachable from here.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// pauseWorld is a headless world with a painted work map and a painted screen, so that a
// restore from one to the other is visible pixel by pixel.
//
// The two fills are deliberately different indices and neither is white: a bug that copies
// nothing, a bug that copies the wrong surface and a bug that clears instead of copying all
// look the same against a white background.
func pauseWorld(t *testing.T, artDir string) *World {
	t.Helper()
	w := newTestWorld(oneRoomHouse(house.ObjectIsEmpty), artDir, "")
	fillSurface(w.R.Work, 8)
	fillSurface(w.Main, 200)
	return w
}

func fillSurface(s *render.Surface, idx uint8) {
	for i := range s.Pix {
		s.Pix[i] = idx
	}
}

// countIndex is how many pixels of one index a rect holds, which is enough to tell "the
// placard is up" from "the placard is gone" without pinning artwork.
func countIndex(s *render.Surface, r Rect, idx uint8) int {
	n := 0
	for v := r.Top; v < r.Bottom; v++ {
		for h := r.Left; h < r.Right; h++ {
			if s.Pix[int(v)*s.W+int(h)] == idx {
				n++
			}
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// The host contract
// ---------------------------------------------------------------------------

// TestDoPauseWithoutAHostDoesNothing: a nil Pause hook is a headless build, and the answer
// has to be "no pause" rather than "pause for ever".
//
// This is the case the fidelity corpus depends on. A replay has no keyboard, so nothing
// could ever end a pause; a faithful DoPause that waited would hang the first recorded
// frame in which the pause key is down, and a hang in a test suite is far more expensive to
// diagnose than a skipped placard.
func TestDoPauseWithoutAHostDoesNothing(t *testing.T) {
	w := pauseWorld(t, "")
	before := append([]uint8(nil), w.Main.Pix...)

	presents := 0
	w.Present = func() { presents++ }
	w.DoPause()

	if w.Paused {
		t.Error("Paused is true after a no-op DoPause; nothing would ever clear it")
	}
	if presents != 0 {
		t.Errorf("presents = %d, want 0: a build with no pause loop should not repaint", presents)
	}
	for i := range before {
		if w.Main.Pix[i] != before[i] {
			t.Fatalf("Main changed at pixel %d; a no-op DoPause must not draw", i)
		}
	}
}

// TestDoPauseRaisesTheFlagForTheHostAndLowersItAfter: Paused is the loop condition, so its
// value *inside* the hook is the whole of what it is for.
//
// The C's DoCommandKey clears `paused` from underneath the loop and 1.10's save-and-quit
// will do the same, which is only possible if the flag is visible from in there.
func TestDoPauseRaisesTheFlagForTheHostAndLowersItAfter(t *testing.T) {
	w := pauseWorld(t, "")
	seen := false
	w.Pause = func(paint func()) {
		seen = w.Paused
		paint()
	}
	w.DoPause()

	if !seen {
		t.Error("Paused was false inside the Pause hook; the host has no loop condition to test")
	}
	if w.Paused {
		t.Error("Paused is still true after DoPause returned")
	}
}

// TestDoPauseDrawsThePlacardThenRestoresFromTheWorkMap is the whole of the visible
// behaviour: something covers the middle of the screen while the game is paused, and
// afterwards those pixels are the frame the work map is still holding.
//
// The restore is checked against the work map's fill rather than against Main's previous
// contents on purpose. That is what the C does -- CopyBits(workSrcMap -> mainWindow) -- and
// it is why the placard needs no saved-bits buffer; a port that stashed and replayed the
// old screen would pass a "Main is unchanged" test and still be wrong the first time the
// pause was raised over a stale window.
func TestDoPauseDrawsThePlacardThenRestoresFromTheWorkMap(t *testing.T) {
	w := pauseWorld(t, "")
	plate := w.pausePlateRect()

	full := int(plate.Wide()) * int(plate.Tall())
	if n := countIndex(w.Main, plate, 200); n != full {
		t.Fatalf("the fixture is wrong: %d of %d screen pixels are the old frame", n, full)
	}

	var duringPause int
	w.Pause = func(paint func()) {
		paint()
		duringPause = countIndex(w.Main, plate, 200)
	}
	w.DoPause()

	if duringPause == full {
		t.Error("the placard's rect still holds the old frame while paused; nothing was drawn")
	}
	if n := countIndex(w.Main, plate, 8); n != full {
		t.Errorf("%d of %d pixels restored from the work map; the erase is incomplete", n, full)
	}
}

// TestDoPauseRestoresTheHintRowToo: the hint is this port's own row and the C's single-rect
// restore knows nothing about it, so the union is the one piece of arithmetic that has to be
// got right or a paused game leaves a permanent stripe under where the placard was.
func TestDoPauseRestoresTheHintRowToo(t *testing.T) {
	w := pauseWorld(t, "")
	w.PauseHint = "press Q to quit"
	hint := w.pauseHintRect()
	if render.Empty(hint) {
		t.Fatal("pauseHintRect is empty with a hint set")
	}

	w.Pause = func(paint func()) { paint() }
	w.DoPause()

	if n := countIndex(w.Main, hint, 8); n != int(hint.Wide())*int(hint.Tall()) {
		t.Errorf("%d of %d hint pixels restored; pauseArea does not cover the hint",
			n, int(hint.Wide())*int(hint.Tall()))
	}
}

// TestDoPausePaintsOncePerPass: the host is required to call paint every pass through its
// loop, and this pins the other side of that bargain -- paint is repeatable, and each call
// presents.
//
// The reason it must be repeatable is RefreshGameWindow: an expose event answered mid-pause
// copies the whole play area up from the work map and wipes the placard, and the next pass
// putting it back is what makes that invisible instead of a case the host has to notice.
func TestDoPausePaintsOncePerPass(t *testing.T) {
	w := pauseWorld(t, "")
	plate := w.pausePlateRect()

	presents := 0
	w.Present = func() { presents++ }
	w.Pause = func(paint func()) {
		for pass := 0; pass < 3; pass++ {
			// Stand in for an expose: RefreshGameWindow's copy, over the placard.
			w.Main.Copy(w.R.Work, plate, plate, render.SrcCopy)
			paint()
			if countIndex(w.Main, plate, 8) == int(plate.Wide())*int(plate.Tall()) {
				t.Errorf("pass %d: the placard was not repainted after an expose", pass)
			}
		}
	}
	w.DoPause()

	// Three paints and the restore, each presenting once.
	if presents != 4 {
		t.Errorf("presents = %d, want 4 (three paints and the restore)", presents)
	}
}

// TestDoPauseIsNotReentrant: a second DoPause from inside the first must return at once.
//
// 1.10's save-and-quit runs from inside the pause loop and will reach code that polls, and a
// poll can report the pause key. Nesting would leave the inner call's `Paused = false` on
// the way out, which would end the outer pause and leave the placard on the screen with the
// game running behind it.
func TestDoPauseIsNotReentrant(t *testing.T) {
	w := pauseWorld(t, "")
	calls := 0
	w.Pause = func(paint func()) {
		calls++
		if calls == 1 {
			w.DoPause()
			if !w.Paused {
				t.Error("the re-entrant call cleared Paused; the outer pause has ended early")
			}
		}
		paint()
	}
	w.DoPause()

	if calls != 1 {
		t.Errorf("the Pause hook ran %d times, want 1: DoPause re-entered", calls)
	}
}

// ---------------------------------------------------------------------------
// The other pause: switched out
// ---------------------------------------------------------------------------

// TestASwitchedOutGameSaysSoAndPutsItBack is docs/IMPROVEMENTS.md 2.28. The C's pump loop
// spins on switchedOut with nothing on the screen, which is what a hung program looks like;
// this one draws the panel every pass and restores the frame on the way out.
func TestASwitchedOutGameSaysSoAndPutsItBack(t *testing.T) {
	w := pauseWorld(t, "")
	plate := w.pausePlateRect()
	full := int(plate.Wide()) * int(plate.Tall())

	// Three passes in the background, then focus comes back -- which is what the host's
	// EventFocus arm does, from inside this hook.
	passes := 0
	w.SwitchedOut = true
	w.PlayEvent = func() {
		if passes++; passes >= 3 {
			w.SwitchedOut = false
		}
	}
	painted := 0
	w.Present = func() {
		if countIndex(w.Main, plate, 200) != full {
			painted++
		}
	}

	w.pumpWhileSwitchedOut()

	if passes != 3 {
		t.Errorf("the pump ran %d times, want 3: the loop does not spin on SwitchedOut", passes)
	}
	if painted < 2 {
		t.Errorf("the panel was on screen for %d presents; it must be drawn every pass so "+
			"that an expose answered mid-suspend cannot wipe it", painted)
	}
	if n := countIndex(w.Main, plate, 8); n != full {
		t.Errorf("%d of %d pixels restored from the work map after resuming", n, full)
	}
}

// TestASwitchedOutGameWithNothingToDrawOnIsTheCsLoop: no Present hook is a headless run, and
// there the loop has to be exactly the C's -- no drawing, no restore, no extra work per
// frame in the one case that runs millions of them.
func TestASwitchedOutGameWithNothingToDrawOnIsTheCsLoop(t *testing.T) {
	w := pauseWorld(t, "")
	w.Present = nil
	before := append([]uint8(nil), w.Main.Pix...)

	w.SwitchedOut = true
	passes := 0
	w.PlayEvent = func() {
		if passes++; passes >= 2 {
			w.SwitchedOut = false
		}
	}
	w.pumpWhileSwitchedOut()

	if string(w.Main.Pix) != string(before) {
		t.Error("Main changed with no Present hook; a headless suspend must not draw")
	}
}

// TestTheSwitchedOutPanelIsNotThePausePlacard: the artwork says "Press Tab to Continue" and
// a switched-out game does not resume on Tab, so the two panels must differ even on a
// checkout that has the art. This is the one place the port refuses the original's picture.
func TestTheSwitchedOutPanelIsNotThePausePlacard(t *testing.T) {
	w := pauseWorld(t, requireAssets(t, "art"))
	plate := w.pausePlateRect()
	if w.R.A.Plate(kTabPausePictID) == nil {
		t.Fatalf("PICT %d is missing from the app art: %v", kTabPausePictID, w.R.A.Err())
	}

	w.Present = func() {}
	w.paintPause()
	paused := append([]uint8(nil), w.Main.Pix...)

	w.drawPausePanel(plate, "click the window to resume")
	if string(w.Main.Pix) == string(paused) {
		t.Error("the switched-out panel is the pause placard; it would tell a player to press " +
			"a key that does nothing")
	}
}

// ---------------------------------------------------------------------------
// The placard itself
// ---------------------------------------------------------------------------

// TestPausePlateRectIsTheCsRect: 214x54 centred in houseRect (Input.c:82-83).
func TestPausePlateRectIsTheCsRect(t *testing.T) {
	w := pauseWorld(t, "")
	r := w.pausePlateRect()

	if r.Wide() != kPausePlateWide || r.Tall() != kPausePlateTall {
		t.Errorf("plate is %dx%d, want %dx%d", r.Wide(), r.Tall(), kPausePlateWide, kPausePlateTall)
	}
	house := w.R.V.House
	if got, want := r.Left-house.Left, house.Right-r.Right; got != want {
		t.Errorf("plate is not horizontally centred: %d left, %d right", got, want)
	}
	if got, want := r.Top-house.Top, house.Bottom-r.Bottom; got != want {
		t.Errorf("plate is not vertically centred: %d above, %d below", got, want)
	}
}

// TestPausePanelIsDrawnWithoutArt: docs/IMPROVEMENTS.md 2.6 -- the absence of a decoration
// is not the absence of a game. With no extracted art the player must still be told that
// the game is paused and which key resumes it, in the same 214x54 the picture would have
// filled, so that nothing else on screen depends on whether the art is there.
func TestPausePanelIsDrawnWithoutArt(t *testing.T) {
	w := pauseWorld(t, "") // no art dir, so Assets.Plate answers nil
	plate := w.pausePlateRect()

	if w.R.A.Plate(kTabPausePictID) != nil {
		t.Skip("this checkout has extracted art; the no-art fallback cannot be reached")
	}

	w.Pause = func(paint func()) {
		paint()
		if n := countIndex(w.Main, plate, render.Black8); n == 0 {
			t.Error("no panel: the placard's rect has no black in it")
		}
		if n := countIndex(w.Main, plate, render.White8); n == 0 {
			t.Error("no text or frame: the placard's rect has no white in it")
		}
		if n := countIndex(w.Main, plate, 200); n != 0 {
			t.Errorf("%d pixels of the old screen show through the panel", n)
		}
	}
	w.DoPause()
}

// TestPausePanelNamesTheKeyThatResumes: the fallback panel has to say Esc or Tab, and which
// one is the player's isEscPauseKey. Compared as pixels because the two panels differ only
// in that word.
func TestPausePanelNamesTheKeyThatResumes(t *testing.T) {
	if newTestWorld(oneRoomHouse(house.ObjectIsEmpty), "", "").R.A.Plate(kTabPausePictID) != nil {
		t.Skip("this checkout has extracted art; the no-art fallback cannot be reached")
	}
	// Captured inside the hook, not from Present: the last thing DoPause presents is the
	// *restored* frame, which is identical either way and would make this test vacuous.
	draw := func(esc bool) []uint8 {
		w := pauseWorld(t, "")
		w.EscPause = esc
		var out []uint8
		w.Pause = func(paint func()) {
			paint()
			out = append([]uint8(nil), w.Main.Pix...)
		}
		w.DoPause()
		return out
	}
	if a, b := draw(true), draw(false); string(a) == string(b) {
		t.Error("the Esc and Tab panels are identical; the panel does not name the key")
	}
}

// TestPausePlateComesFromTheHouseFirst: Teddy World ships its own PICT 1015 and 1016, and
// on a Mac its resource fork sits in front of the application's for those ids. Assets.Plate
// is what reproduces that, and this is the pause path's end of it.
func TestPausePlateComesFromTheHouseFirst(t *testing.T) {
	art := requireAssets(t, "art")
	fork := requireAssets(t, "houseart/Teddy World")

	w := newTestWorld(oneRoomHouse(house.ObjectIsEmpty), art, fork)
	appOnly := newTestWorld(oneRoomHouse(house.ObjectIsEmpty), art, "")

	for _, id := range []int16{kEscPausePictID, kTabPausePictID} {
		got, want := w.R.A.Plate(id), appOnly.R.A.Plate(id)
		if got == nil || want == nil {
			t.Fatalf("PICT %d is missing from the app art or from Teddy World's fork", id)
		}
		if string(got.Pix) == string(want.Pix) {
			t.Errorf("PICT %d is the application's in Teddy World; the fork is not "+
				"being consulted first", id)
		}
	}
}

// ---------------------------------------------------------------------------
// The hint row
// ---------------------------------------------------------------------------

// TestPauseHintRectIsEmptyWithoutAHint: no hint draws nothing and restores nothing, which
// is what a fidelity replay wants -- the C has no such row.
func TestPauseHintRectIsEmptyWithoutAHint(t *testing.T) {
	w := pauseWorld(t, "")
	if r := w.pauseHintRect(); !render.Empty(r) {
		t.Errorf("pauseHintRect = %+v with no hint, want empty", r)
	}
	if got, want := w.pauseArea(), w.pausePlateRect(); got != want {
		t.Errorf("pauseArea = %+v with no hint, want the placard's %+v", got, want)
	}
}

// TestPauseHintRectStaysInsideTheHouse: a hint longer than the play area is wide would
// otherwise put pixels where the restore does not reach -- off the edge, or on the
// scoreboard, which the pause never covered and RefreshGameWindow will not repair.
func TestPauseHintRectStaysInsideTheHouse(t *testing.T) {
	w := pauseWorld(t, "")
	house := w.R.V.House

	for _, hint := range []string{
		"Q",
		"press Q to quit -- this port has no Command key",
		"press Q to quit, and this is a hint long enough to be wider than the whole of the " +
			"play area, which is the case that has to clamp rather than overflow",
	} {
		w.PauseHint = hint
		r := w.pauseHintRect()
		if render.Empty(r) {
			t.Errorf("%q: empty rect", hint)
			continue
		}
		if r.Left < house.Left || r.Right > house.Right || r.Bottom > house.Bottom {
			t.Errorf("%q: rect %+v is outside the house %+v", hint, r, house)
		}
		if plate := w.pausePlateRect(); r.Top < plate.Bottom {
			t.Errorf("%q: rect %+v overlaps the placard %+v", hint, r, plate)
		}
	}
}

// TestPauseAreaIsTheUnion: the restore's rect is what was drawn and nothing more. A rect
// that is too small leaves a stripe; one that is too large copies work-map pixels over a
// scoreboard the pause never touched.
func TestPauseAreaIsTheUnion(t *testing.T) {
	w := pauseWorld(t, "")
	w.PauseHint = "press Q to quit -- this port has no Command key"

	plate, hint, area := w.pausePlateRect(), w.pauseHintRect(), w.pauseArea()
	for _, r := range []Rect{plate, hint} {
		if s, ok := render.Sect(r, area); !ok || s != r {
			t.Errorf("pauseArea %+v does not contain %+v", area, r)
		}
	}
	if area.Tall() > plate.Tall()+kPauseHintGap+hint.Tall() {
		t.Errorf("pauseArea %+v is taller than the placard and the hint together", area)
	}
}
