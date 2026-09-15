package game

// Grease.c's simulation half: the four modes, the four-frame tip, the rect that changes its
// own action mid-frame, and the repaint that puts a slick back after something erased part of
// it.
//
// The registration half is internal/render/grease_test.go. The split follows the code's, and
// the seam is worth naming because it is where the two coordinate systems meet: everything
// the render side writes -- Dest, Start, Stop -- is in *screen* pixels, and the one thing this
// side writes into the collision table, a slide rect's Bounds, is room-local. So the tests in
// this file that matter most are the ones asserting a number the other file's tests also
// assert, once with the play origin in it and once without.
//
// ---------------------------------------------------------------------------
// Why these tests build the grease table by hand
// ---------------------------------------------------------------------------
//
// Every test here appends a render.Grease directly rather than composing a room with a jar
// in it. Going through AddGrease would need art -- it bakes four cels out of the `bonus`
// sheet -- which would make the whole file skip on a bare checkout, and it would test the
// registration a second time instead of testing the simulation once. The fields a
// hand-built entry has to get right are Dest, Start, Stop, Frame and HotNum, and greaseJar
// below is the one place they are set so that a reader has one thing to check against
// AddGrease rather than a dozen.
//
// The consequence to keep in mind is that MapNum names no saved map, so greaseStrip returns
// nil and the falling arm's two blits are skipped. That is deliberate and it is the same
// bargain dynamics_test.go makes: what is under test is the bookkeeping -- the mode walk, the
// rect arithmetic, the dirty-rect registrations -- and every one of those runs identically
// with or without pixels. The blits themselves are the render side's golden images.

import (
	"testing"

	"glidergo/internal/game/player"
	"glidergo/internal/render"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// greaseJar puts one jar in the table with one kRewardIt hot spot pointing at it, arranged
// exactly as a composed room would have them: Dest in screen coordinates, the hot spot's
// Bounds room-local and the same 32x27 box, and Frame at -1.
//
// The numbers are AddGrease's, spelled out rather than derived: Start is four pixels clear of
// the jar on the leading side and Stop is `distance` pixels from the same edge. A test that
// called AddGrease to get them would pass even if that arithmetic were wrong.
//
// It returns the jar and its hot spot. Both are pointers into slices that nothing in these
// tests appends to, which is what makes holding them across frames safe.
func greaseJar(w *World, h, v, distance int16, isRight bool) (*render.Grease, *HotObject) {
	dest := render.Offset(render.SetRect(0, 0, 32, 27), h, v)

	g := render.Grease{
		Who: 0, Where: w.R.RoomNumber,
		Dest:    dest,
		MapNum:  -1, // no saved map: see the file comment
		Mode:    render.GreaseIdle,
		Frame:   -1,
		IsRight: isRight,
	}
	if isRight {
		g.Start, g.Stop = dest.Right+4, dest.Right+distance
	} else {
		g.Start, g.Stop = dest.Left-4, dest.Left-distance
	}
	// The jar's own reward rect, room-local as every hot spot is.
	w.R.Hot = append(w.R.Hot, HotObject{
		Bounds: render.Offset(dest, -w.R.V.OriginH, -w.R.V.OriginV),
		Action: RewardIt, Who: 0, IsOn: true,
	})

	// Index-matched, which is the one thing here a composed room would *not* do: in the real
	// game HotNum is written by SpillGrease from the master object's own record, and until
	// then it is a stale 0. Wiring it up front is what lets a test put two jars in one room
	// without the second silently addressing the first one's rect -- which is exactly the
	// confusion greaseHot's guard exists to survive, and which
	// TestRedrawAllGreaseSurvivesAnUnresolvableHotNum sets up deliberately instead.
	g.HotNum = int16(len(w.R.Hot) - 1)
	w.R.Grease = append(w.R.Grease, g)

	return &w.R.Grease[len(w.R.Grease)-1], &w.R.Hot[len(w.R.Hot)-1]
}

// greaseWorld is dynaWorld with the room number pinned, because two of the three functions in
// grease.go compare a jar's Where against it.
func greaseWorld(t *testing.T) *World {
	t.Helper()
	w := dynaWorld(t)
	w.R.RoomNumber = 0
	return w
}

// ---------------------------------------------------------------------------
// The tip
// ---------------------------------------------------------------------------

// TestGreaseTipIsFourFramesNotThree is the off-by-one the -1 initial Frame exists to create.
//
// `frame++` then `if frame >= 3` looks like three frames and is four: Frame is -1 on entry, so
// the cels *drawn* are 0, 1, 2 and then 3 on the frame the test fires, and the transition to
// spreading happens on that fourth frame rather than after it. Initialise Frame to 0 instead
// and the jar tips in three frames and skips its last cel -- a change no golden image catches,
// because every golden composes an upright jar.
//
// The walk is pinned alongside it because the two are the same mistake seen twice: the jar
// steps two pixels per frame in the direction it tips, and it steps on the frame the mode
// changes as well, so a right-facing jar's Dest has moved eight pixels by the time it is a
// slick. That eight is the same eight AddGrease subtracts -- see the render side.
func TestGreaseTipIsFourFramesNotThree(t *testing.T) {
	for _, c := range []struct {
		name    string
		isRight bool
		step    int16
	}{
		{"tipping right", true, 2},
		{"tipping left", false, -2},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := greaseWorld(t)
			g, _ := greaseJar(w, 200, 100, 64, c.isRight)
			g.Mode = render.GreaseFalling

			wantFrame := []int16{0, 1, 2, 3}
			for i, want := range wantFrame {
				if g.Mode != render.GreaseFalling {
					t.Fatalf("frame %d: the jar left GreaseFalling early, at Mode %d",
						i, g.Mode)
				}
				w.HandleGrease()
				if g.Frame != want {
					t.Errorf("frame %d: Frame = %d, want %d", i, g.Frame, want)
				}
				if got, wantLeft := g.Dest.Left, 200+int16(i+1)*c.step; got != wantLeft {
					t.Errorf("frame %d: Dest.Left = %d, want %d", i, got, wantLeft)
				}
			}
			if g.Mode != render.GreaseSpreading {
				t.Errorf("Mode = %d after four frames, want GreaseSpreading (%d)",
					g.Mode, render.GreaseSpreading)
			}
			if got, want := g.Dest.Left, 200+4*c.step; got != want {
				t.Errorf("Dest.Left = %d after the tip, want %d -- the jar steps on the "+
					"frame it becomes a slick too", got, want)
			}
		})
	}
}

// TestFallingGreaseRegistersOneWorkRectPerFrame is the tip's dirty-rect half.
//
// One work rect a frame and **no back rect**, which is the difference between grease and every
// mover: the falling arm writes the back map itself, so the tip is permanent and there is
// nothing to restore. A back rect here would put the upright jar back the frame after each cel
// and the animation would flicker between the first cel and the current one.
func TestFallingGreaseRegistersOneWorkRectPerFrame(t *testing.T) {
	w := greaseWorld(t)
	g, _ := greaseJar(w, 200, 100, 64, true)
	g.Mode = render.GreaseFalling
	clearRects(w)

	for i := 0; i < 4; i++ {
		w.HandleGrease()
		work, back := rectCounts(w)
		if work != i+1 || back != 0 {
			t.Fatalf("after %d frames: %d work rects and %d back rects, want %d and 0",
				i+1, work, back, i+1)
		}
	}
}

// ---------------------------------------------------------------------------
// The reward rect becoming a slide rect
// ---------------------------------------------------------------------------

// TestSlideRectIsRoomLocalAndTwoPixelsTall is the transition, and it is the one place in the
// game where a hot spot changes what it does while the room is running.
//
// Three things happen to the rect and all three are asserted, because each is a different way
// to get it wrong:
//
//	Action    kRewardIt -> kSlideIt. Miss this and the slick is still a prize: flying
//	          through it collects the jar a second time
//	Bounds    a 2-pixel-high strip on the floor line, **room-local**. Dest and Start are
//	          screen coordinates, so the arm subtracts playOrigin; drop that and the slick
//	          collides 64 pixels right and 79 pixels down from where it is drawn
//	IsOn      forced true, because the reward arm switched the rect off when the jar was
//	          knocked over
//
// The floor line is the subtle one. The bounds are computed from `Dest.Bottom` *before* the
// two-pixel walk at the bottom of the same arm, so the slick sits on the line the fourth cel
// was drawn at rather than two pixels along -- and since the walk is horizontal, that shows up
// in the rect's *left edge* being Start rather than Start±2.
func TestSlideRectIsRoomLocalAndTwoPixelsTall(t *testing.T) {
	for _, c := range []struct {
		name    string
		isRight bool
	}{
		{"tipping right", true},
		{"tipping left", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := greaseWorld(t)
			g, hot := greaseJar(w, 200, 100, 64, c.isRight)
			g.Mode = render.GreaseFalling

			// Three frames of falling: the fourth is the one under test.
			for i := 0; i < 3; i++ {
				w.HandleGrease()
			}
			if hot.Action != RewardIt {
				t.Fatalf("the rect changed action on frame %d; the transition is on the "+
					"fourth", 3)
			}
			startBefore, bottomBefore := g.Start, g.Dest.Bottom

			w.HandleGrease()

			if hot.Action != SlideIt {
				t.Errorf("Action = %s, want SlideIt", ActionName(hot.Action))
			}
			if !hot.IsOn {
				t.Error("IsOn is false; the transition forces it on because the reward " +
					"arm switched it off when the jar went over")
			}

			// Spelled out rather than built with the arm's own helpers: room-local, two
			// pixels tall, hanging above the floor line, growing from Start.
			var want Rect
			if c.isRight {
				want = Rect{Left: startBefore, Right: startBefore + 2}
			} else {
				want = Rect{Left: startBefore - 2, Right: startBefore}
			}
			want.Top, want.Bottom = bottomBefore-2, bottomBefore
			want = render.Offset(want, -w.R.V.OriginH, -w.R.V.OriginV)

			if hot.Bounds != want {
				t.Errorf("Bounds = %+v, want %+v", hot.Bounds, want)
			}
			if got := hot.Bounds.Tall(); got != 2 {
				t.Errorf("Bounds is %d tall, want 2 -- RedrawAllGrease uses that height "+
					"as its test for \"is there a slick here\"", got)
			}
		})
	}
}

// TestSlideRectSurvivesAnUnresolvableHotNum is greaseHot's guard, which the C does not have.
//
// HotNum is zero on a jar nothing has touched and can name a rect from a *previous* locale on
// one the table has outlived, because DrawLocale clears the grease table while AddGrease never
// rewrites HotNum. In the C both land inside a fixed-size array and the value is discarded;
// here the table is a slice and the second would panic mid-frame.
//
// The jar still tips. That is the point: the guard skips the rect rewrite and nothing else, so
// a jar whose hot spot cannot be found becomes a slick that is drawn and cannot be slipped on,
// rather than crashing the game.
func TestSlideRectSurvivesAnUnresolvableHotNum(t *testing.T) {
	w := greaseWorld(t)
	g, _ := greaseJar(w, 200, 100, 64, true)
	g.Mode = render.GreaseFalling
	g.HotNum = int16(len(w.R.Hot)) + 5 // past the end

	for i := 0; i < 4; i++ {
		w.HandleGrease()
	}
	if g.Mode != render.GreaseSpreading {
		t.Errorf("Mode = %d, want GreaseSpreading -- the guard should skip the rect "+
			"rewrite, not the transition", g.Mode)
	}
}

// ---------------------------------------------------------------------------
// The slick
// ---------------------------------------------------------------------------

// TestGreaseSpreadStopsOneStepShortOfStop walks a slick from the first two pixels to
// SpiltIdle, checking the rect on every frame.
//
// Start is advanced *before* the termination test, so the test is against the position the
// next frame would paint from and a slick stops one step short of Stop rather than one past
// it. With distance 64 and Start beginning at Dest.Right+4, the reachable span is 60 pixels
// and the loop runs 30 times.
//
// The hot spot grows in lockstep with the paint, one edge per frame, and the *other* edge
// never moves -- a right-facing slick's Left is the jar and its Right is the leading edge. So
// the collision rect and the black pixels are the same shape throughout, which is what makes
// the slick feel like it is where it looks.
func TestGreaseSpreadStopsOneStepShortOfStop(t *testing.T) {
	for _, c := range []struct {
		name    string
		isRight bool
		sign    int16
	}{
		{"spreading right", true, 1},
		{"spreading left", false, -1},
	} {
		t.Run(c.name, func(t *testing.T) {
			const distance int16 = 64
			w := greaseWorld(t)
			g, hot := greaseJar(w, 200, 100, distance, c.isRight)

			// Hand the jar straight to the spreading arm, with the slide rect the
			// transition would have made: a 2-pixel seed at the jar's edge.
			g.Mode = render.GreaseSpreading
			g.Frame = 3
			seed := Rect{Top: g.Dest.Bottom - 2, Bottom: g.Dest.Bottom}
			if c.isRight {
				seed.Left, seed.Right = g.Start, g.Start+2
			} else {
				seed.Left, seed.Right = g.Start-2, g.Start
			}
			hot.Action, hot.IsOn = SlideIt, true
			hot.Bounds = render.Offset(seed, -w.R.V.OriginH, -w.R.V.OriginV)
			anchor := hot.Bounds

			frames := 0
			for g.Mode == render.GreaseSpreading {
				w.HandleGrease()
				frames++
				if frames > 100 {
					t.Fatalf("the slick never stopped; Start = %d, Stop = %d",
						g.Start, g.Stop)
				}
				// One edge advances two pixels, the other is untouched.
				want := anchor
				if c.isRight {
					want.Right += int16(frames) * 2
				} else {
					want.Left -= int16(frames) * 2
				}
				if hot.Bounds != want {
					t.Fatalf("frame %d: Bounds = %+v, want %+v",
						frames, hot.Bounds, want)
				}
			}

			if g.Mode != render.GreaseSpiltIdle {
				t.Fatalf("Mode = %d, want GreaseSpiltIdle", g.Mode)
			}

			// 60 reachable pixels of the authored 64, two at a time. The four the slick
			// never covers are the four Start was seeded clear of the jar by.
			if want := (distance - 4) / 2; int16(frames) != want {
				t.Errorf("the slick took %d frames, want %d -- distance %d less the "+
					"4-pixel gap, two pixels a frame", frames, want, distance)
			}
			// One step short: Start has just crossed Stop, not landed on it.
			if c.isRight && g.Start < g.Stop {
				t.Errorf("Start = %d stopped before Stop = %d", g.Start, g.Stop)
			}
			if !c.isRight && g.Start > g.Stop {
				t.Errorf("Start = %d stopped before Stop = %d", g.Start, g.Stop)
			}
		})
	}
}

// TestSpreadingGreaseRegistersTheRectItPainted is the correctly-converted half of
// docs/IMPROVEMENTS.md 2.34.
//
// The paint and the registration name the same rect at the same call site, three lines apart,
// so the dirty rect tells the truth about where the pixels went. HandleOutlet's PaintRect is
// the half that does not, and this is the model it should have followed -- which is why the
// assertion is on the *screen-space* rect: the registration takes the rect Fill took, with no
// conversion between them.
func TestSpreadingGreaseRegistersTheRectItPainted(t *testing.T) {
	w := greaseWorld(t)
	g, hot := greaseJar(w, 200, 100, 64, true)
	g.Mode = render.GreaseSpreading
	g.Frame = 3
	hot.Action, hot.IsOn = SlideIt, true
	hot.Bounds = render.Offset(
		Rect{Top: g.Dest.Bottom - 2, Left: g.Start, Bottom: g.Dest.Bottom, Right: g.Start + 2},
		-w.R.V.OriginH, -w.R.V.OriginV)

	start, bottom := g.Start, g.Dest.Bottom
	clearRects(w)
	w.HandleGrease()

	work, back := rectCounts(w)
	if work != 1 || back != 0 {
		t.Fatalf("%d work rects and %d back rects, want 1 and 0", work, back)
	}
	want := Rect{Top: bottom - 2, Left: start, Bottom: bottom, Right: start + 2}
	if w.Work2Main[0] != want {
		t.Errorf("work rect = %+v, want %+v (screen coordinates, exactly the rect Fill "+
			"was handed)", w.Work2Main[0], want)
	}
}

// TestIdleAndSpiltGreaseCostNothing is the reason there is no separate list of active spills.
//
// Both ends of the mode range fall through HandleGrease's switch with no arm, so a room full
// of untouched jars and a room full of spent ones are the same zero-cost frame. Asserting it
// as "no rects and no sound" is the only observable there is, and it is the one that would
// break if someone gave the spilt arm a repaint -- which is RedrawAllGrease's job and is
// called from somewhere else entirely.
func TestIdleAndSpiltGreaseCostNothing(t *testing.T) {
	for _, mode := range []int16{render.GreaseIdle, render.GreaseSpiltIdle} {
		w := greaseWorld(t)
		g, _ := greaseJar(w, 200, 100, 64, true)
		g.Mode = mode
		before := *g

		var sounds soundLog
		sounds.install(w)
		clearRects(w)
		w.HandleGrease()

		if *g != before {
			t.Errorf("mode %d: the jar changed: %+v -> %+v", mode, before, *g)
		}
		if work, back := rectCounts(w); work != 0 || back != 0 {
			t.Errorf("mode %d: %d work and %d back rects, want none", mode, work, back)
		}
		sounds.is(t)
	}
}

// TestHandleGreaseWithNoJarsIsANoOp is the early return, and the whole of the optimisation the
// original bothered with. Worth a line because the guard is `len(...) == 0` on a slice where
// the C tests a counter, and a `<= 0` typo would be invisible.
func TestHandleGreaseWithNoJarsIsANoOp(t *testing.T) {
	w := greaseWorld(t)
	if len(w.R.Grease) != 0 {
		t.Fatalf("fixture already has %d jars", len(w.R.Grease))
	}
	clearRects(w)
	w.HandleGrease()
	if work, back := rectCounts(w); work != 0 || back != 0 {
		t.Errorf("%d work and %d back rects on an empty table", work, back)
	}
}

// ---------------------------------------------------------------------------
// SpillGrease
// ---------------------------------------------------------------------------

// TestSpillGreaseIsOneShot is the idle test, which is the only guard the function has -- and
// is the reason HandleRewards' grease arm needs no StillOver latch of its own.
//
// A glider resting on a knocked-over jar re-enters the reward arm every frame. Without the
// test the animation would restart and the spill sound would fire sixty times a second; with
// it, the second call and every call after it is silent. The three non-idle modes are all
// tested because "already falling" and "already spilt" are different states that have to give
// the same answer.
func TestSpillGreaseIsOneShot(t *testing.T) {
	w := greaseWorld(t)
	g, _ := greaseJar(w, 200, 100, 64, true)

	var sounds soundLog
	sounds.install(w)

	w.SpillGrease(0, 0)
	if g.Mode != render.GreaseFalling {
		t.Fatalf("Mode = %d after the first spill, want GreaseFalling", g.Mode)
	}
	sounds.is(t, GreaseSpillSound)

	for _, mode := range []int16{render.GreaseFalling, render.GreaseSpreading, render.GreaseSpiltIdle} {
		g.Mode = mode
		g.HotNum = 7
		w.SpillGrease(0, 99)
		if g.Mode != mode {
			t.Errorf("mode %d: a second spill changed it to %d", mode, g.Mode)
		}
		if g.HotNum != 7 {
			t.Errorf("mode %d: a second spill rewrote HotNum to %d; the refused call "+
				"must not touch the rect the jar already owns", mode, g.HotNum)
		}
	}
	sounds.is(t, GreaseSpillSound)
}

// TestSpillGreaseRefusesAnOutOfRangeIndex is FireTrigger's remote half, and the one place this
// port declines to reproduce a read the original performs.
//
// Triggers.c passes a dynamic index its own branch condition has just proved is -1, which in C
// is a read of grease[-1] -- in-bounds-ish on a 68k global array, undefined everywhere, and a
// panic in Go. The refusal is the fix, and it is safe because the only thing the C's read
// could have done is spill a jar the author never wired up: the -1 means "the target is not in
// this locale", so there is no jar to spill.
//
// hotNum is deliberately *not* checked here. It is only stored; greaseHot guards the read.
func TestSpillGreaseRefusesAnOutOfRangeIndex(t *testing.T) {
	w := greaseWorld(t)
	g, _ := greaseJar(w, 200, 100, 64, true)

	var sounds soundLog
	sounds.install(w)

	for _, dyna := range []int16{-1, -100, 1, int16(len(w.R.Grease)), 500} {
		w.SpillGrease(dyna, 0)
		if g.Mode != render.GreaseIdle {
			t.Fatalf("dynaNum %d spilled the jar at index 0", dyna)
		}
	}
	sounds.is(t)

	// And a negative hotNum is stored rather than refused, because greaseHot is what
	// guards it -- so the jar still tips and simply never gets a slide rect.
	w.SpillGrease(0, -1)
	if g.Mode != render.GreaseFalling || g.HotNum != -1 {
		t.Errorf("Mode/HotNum = %d/%d, want GreaseFalling/-1: hotNum is stored unchecked",
			g.Mode, g.HotNum)
	}
	for i := 0; i < 4; i++ {
		w.HandleGrease()
	}
	if g.Mode != render.GreaseSpreading {
		t.Errorf("Mode = %d, want GreaseSpreading -- a jar with no resolvable rect still "+
			"tips", g.Mode)
	}
}

// ---------------------------------------------------------------------------
// RedrawAllGrease
// ---------------------------------------------------------------------------

// TestRedrawAllGreaseTestsAllThree walks the truth table of the three conditions, because the
// function's whole job is deciding what *not* to repaint.
//
// It is called by the ten reward arms that restore a saved map, for a reason not in the name:
// RestoreFromSavedMap has just written a rectangle of original background over whatever was
// there, and a slick crossing that rectangle now has a hole in it. This paints the lot back --
// so a false positive repaints a jar that has no slick (black smear where the jar was) and a
// false negative leaves the hole.
//
// The middle test is the interesting one. `height == 2` is a second, independent way of asking
// "is there a slick?", and it is the one that catches a jar whose hot spot was reused by a room
// rebuild: the mode says spreading but the rect is 27 pixels tall because it belongs to
// something else now, and repainting it would fill a jar-sized box with black.
func TestRedrawAllGreaseTestsAllThree(t *testing.T) {
	for _, c := range []struct {
		name        string
		room        int16 // the jar's Where
		tall        int16 // the hot spot's height
		mode        int16
		wantRepaint bool
	}{
		{"a spilt slick in this room", 0, 2, render.GreaseSpiltIdle, true},
		{"a spreading slick in this room", 0, 2, render.GreaseSpreading, true},
		{"a falling jar in this room", 0, 2, render.GreaseFalling, true},
		{"an untouched jar", 0, 2, render.GreaseIdle, false},
		{"a slick in a neighbouring room", 1, 2, render.GreaseSpiltIdle, false},
		{"a spilt jar still owning its 27px reward rect", 0, 27, render.GreaseSpiltIdle, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := greaseWorld(t)
			g, hot := greaseJar(w, 200, 100, 64, true)
			g.Where = c.room
			g.Mode = c.mode
			hot.Bounds = Rect{Top: 50, Left: 100, Bottom: 50 + c.tall, Right: 160}
			if c.mode != render.GreaseIdle {
				hot.Action = SlideIt
			}

			clearRects(w)
			w.RedrawAllGrease()

			work, back := rectCounts(w)
			if back != 0 {
				t.Errorf("%d back rects; the repaint writes the back map itself and has "+
					"nothing to restore", back)
			}
			if (work == 1) != c.wantRepaint {
				t.Fatalf("%d work rects, want %d", work, map[bool]int{true: 1}[c.wantRepaint])
			}
			if !c.wantRepaint {
				return
			}
			// Room-local Bounds, screen-space rect: the conversion RedrawAllGrease adds
			// and HandleGrease's transition removes. Asserting the number is what pins
			// the direction; a sign error passes every "is it repainted" test.
			want := Rect{
				Top:    hot.Bounds.Top + w.R.V.OriginV,
				Left:   hot.Bounds.Left + w.R.V.OriginH,
				Bottom: hot.Bounds.Bottom + w.R.V.OriginV,
				Right:  hot.Bounds.Right + w.R.V.OriginH,
			}
			if w.Work2Main[0] != want {
				t.Errorf("work rect = %+v, want %+v", w.Work2Main[0], want)
			}
		})
	}
}

// TestRedrawAllGreaseSurvivesAnUnresolvableHotNum is the other side of greaseHot's guard.
//
// The C reads hotSpots[hotNum] *before* testing any of the three conditions, which is safe
// there because hotSpots is allocated at its cap and an unwritten hotNum of 0 indexes garbage
// that is immediately discarded. Here the table is a slice whose length grows during
// composition, so the read is guarded -- and the reordering is unobservable because none of
// the three tests has a side effect. This is the assertion that the guard skips the jar rather
// than the whole loop, so one bad entry does not stop the rest of the room being repaired.
func TestRedrawAllGreaseSurvivesAnUnresolvableHotNum(t *testing.T) {
	w := greaseWorld(t)

	bad, _ := greaseJar(w, 200, 100, 64, true)
	bad.Mode = render.GreaseSpiltIdle
	bad.HotNum = 500

	good, goodHot := greaseJar(w, 300, 100, 64, true)
	good.Mode = render.GreaseSpiltIdle
	goodHot.Bounds = Rect{Top: 50, Left: 100, Bottom: 52, Right: 160}

	clearRects(w)
	w.RedrawAllGrease()

	if work, _ := rectCounts(w); work != 1 {
		t.Errorf("%d work rects, want 1: the bad entry is skipped and the good one is "+
			"still repainted", work)
	}
}

// ---------------------------------------------------------------------------
// The frame the slick is not collidable
// ---------------------------------------------------------------------------

// TestSlideRectIsNotCollidableOnTheFrameItAppears is the one-frame delay the file comment
// promises, measured rather than asserted from the call order.
//
// HandleGrease runs inside RenderFrame, after the interaction sweep has finished with this
// frame's hot spots (Render.c:647). So on the frame a jar finishes tipping, the rect it just
// rewrote has already been swept: a glider standing exactly where the slick appears is not
// sliding until the next frame. TestRenderFrameOrder in play_test.go owns the call order; what
// this owns is the *consequence*, which is the thing a well-meaning reorder would break.
//
// The measurement is Glider.Sliding, which kSlideIt sets and which MoveGliderNormal clears
// every frame -- so it is a per-frame answer to "am I on grease right now" and not a latch.
// That makes it exactly the right probe: the test clears it by hand before each sweep, which
// is what the movement code would have done, and reads whether the sweep put it back.
func TestSlideRectIsNotCollidableOnTheFrameItAppears(t *testing.T) {
	w := greaseWorld(t)
	g, hot := greaseJar(w, 200, 100, 64, true)
	g.Mode = render.GreaseFalling

	// Park the glider over where the slick will be: room-local, on the jar's floor line.
	// kSlideIt is not one of the five scrutinized actions, so the plain rect overlap is
	// enough and there is no 5px inset to allow for.
	slick := render.Offset(
		Rect{Top: g.Dest.Bottom - 2, Left: g.Start, Bottom: g.Dest.Bottom, Right: g.Start + 40},
		-w.R.V.OriginH, -w.R.V.OriginV)
	w.P1.Dest = player.Rect{
		Top: slick.Top - 18, Left: slick.Left, Bottom: slick.Bottom, Right: slick.Left + 48,
	}
	w.P1.Mode = player.GliderNormal

	// Three frames of falling, so the next call is the transition.
	for i := 0; i < 3; i++ {
		w.HandleGrease()
	}

	// The frame the slick appears: sweep first, as RenderFrame does, then HandleGrease.
	w.P1.Sliding = false
	w.CheckForHotSpots()
	w.HandleGrease()
	if hot.Action != SlideIt {
		t.Fatalf("the transition did not happen; Action = %s", ActionName(hot.Action))
	}
	if w.P1.Sliding {
		t.Error("the glider is sliding on the frame the slick appeared; the sweep had " +
			"already run when HandleGrease rewrote the rect")
	}

	// The next frame, same order, and now it bites.
	w.P1.Sliding = false
	w.CheckForHotSpots()
	w.HandleGrease()
	if !w.P1.Sliding {
		t.Error("the glider is still not sliding on the frame after; the slide rect " +
			"should be collidable by now")
	}
	// And the arm's other half: the velocity that seats the glider on the slick's top edge
	// in one frame rather than letting gravity settle it.
	if want := hot.Bounds.Top - w.P1.Dest.Bottom; w.P1.VVel != want {
		t.Errorf("VVel = %d, want %d -- the arm seats the glider on the spill's top edge",
			w.P1.VVel, want)
	}
}
