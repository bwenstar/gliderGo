package game

// Render.c's background-animation passes, simulation half: the cel walk, the wrap, the
// pendulum's four-step swing, and the two clocks that decide which frames any of it happens
// on.
//
// The registration half is internal/render/anim_test.go, and the seam between them is a
// single number per family -- the cel height. That file asserts the strip a registration
// bakes is `frames` cels of `celH` tall; this one asserts the animator steps by `celH` and
// wraps after `frames`. Both have to be right for a flame to animate, and getting one wrong
// alone produces a flame that walks off the bottom of its own strip, which draws nothing and
// looks like a missing art file.
//
// ---------------------------------------------------------------------------
// Why these tests build the tables by hand
// ---------------------------------------------------------------------------
//
// Same bargain grease_test.go makes, for the same reason. Going through the five add*
// functions would need art -- they bake cels out of the `blower` and `bonus` sheets -- so the
// whole file would skip on a bare checkout, and it would be testing the registration a second
// time instead of testing the animation once.
//
// The consequence is that SavedMap names no strip, so animStrip returns nil and the one blit
// per entry is skipped. Everything these tests are about runs identically either way: the
// Mode walk, the Src arithmetic, the sounds, and the dirty-rect registrations. The pixels are
// the render side's golden images.

import (
	"testing"

	"glidergo/internal/render"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// animEntry is one table entry arranged as a registration would leave it, minus the strip:
// Dest a cel-sized box in screen coordinates well inside justRoomsRect, Mode the seeded cel
// index with Src the matching cel, and SavedMap pointing nowhere.
//
// Mode and Src are set from the same argument on purpose. The C stores the cel index and the
// cel rect separately and steps them in adjacent lines, so a fixture that set only one of them
// would be starting the animator from a state no registration produces -- and, worse, would
// pass most of the assertions below anyway.
//
// Dest being clear of the clamp is also deliberate. AddRectToWorkRects clips to
// justRoomsRect, so an entry parked near an edge would register a rect that is not its Dest
// and every bookkeeping assertion below would be testing the clamp instead of the animator.
func animEntry(celW, celH, mode int16) render.Anim {
	return render.Anim{
		Dest:     render.Offset(render.SetRect(0, 0, celW, celH), 200, 200),
		Src:      render.Offset(render.SetRect(0, 0, celW, celH), 0, mode*celH),
		Mode:     mode,
		SavedMap: -1, // no strip: see the file comment
		Where:    0,
		Who:      3,
	}
}

// animWorld is dynaWorld with the sound player recording instead of discarding, because two
// of the pendulum's four steps are sounds and a nil hook makes them invisible.
func animWorld(t *testing.T) (*World, *[][2]int16) {
	t.Helper()
	w := dynaWorld(t)
	var heard [][2]int16
	w.SoundPlayer = func(sound, priority int16) {
		heard = append(heard, [2]int16{sound, priority})
	}
	return w, &heard
}

// theFourWrappingFamilies is the table stepStrip's four call sites reduce to: a slice on the
// Scene, a cel size and a frame count. Each test below runs over all four rather than picking
// a representative, because the C's four copies have already drifted from each other once
// (RenderStars tests a literal 6 where its siblings use a named constant) and a fifth
// divergence is exactly what a representative test would miss.
var theFourWrappingFamilies = []struct {
	name       string
	celW, celH int16
	frames     int16
	set        func(w *World, t []render.Anim)
	get        func(w *World) []render.Anim
	render     func(w *World)
}{
	{
		name: "candle flames", celW: 16, celH: 15, frames: render.NumCandleFrames,
		set:    func(w *World, t []render.Anim) { w.R.Flames = t },
		get:    func(w *World) []render.Anim { return w.R.Flames },
		render: (*World).RenderFlames,
	},
	{
		name: "tiki flames", celW: 8, celH: 10, frames: render.NumTikiFrames,
		set:    func(w *World, t []render.Anim) { w.R.TikiFlames = t },
		get:    func(w *World) []render.Anim { return w.R.TikiFlames },
		render: (*World).RenderFlames,
	},
	{
		name: "bbq coals", celW: 32, celH: 9, frames: render.NumCoalFrames,
		set:    func(w *World, t []render.Anim) { w.R.Coals = t },
		get:    func(w *World) []render.Anim { return w.R.Coals },
		render: (*World).RenderFlames,
	},
	{
		name: "stars", celW: 32, celH: 31, frames: render.NumStarFrames,
		set:    func(w *World, t []render.Anim) { w.R.Stars = t },
		get:    func(w *World) []render.Anim { return w.R.Stars },
		render: (*World).RenderStars,
	},
}

// ---------------------------------------------------------------------------
// The cel walk
// ---------------------------------------------------------------------------

// TestEachFamilyWalksItsStripAndWraps is the core of the file: two full cycles of every
// family, with Mode and Src pinned on every single frame.
//
// Two things are being asserted at once and they are separable failures. Mode is the cel
// index and drives the wrap; Src is the rect actually blitted. The C stores both and steps
// them independently -- `mode++` and `SetRect`/`OffsetRect` in adjacent lines -- so they can
// disagree, and a port that derived one from the other would be untestable here and subtly
// wrong at the wrap. See TestTheWrapResetsSrcAbsolutelyNotBySubtraction.
//
// The frame-one value is the one worth reading closely: **an entry seeded at cel 0 shows cel
// 1 on its first frame.** The step precedes the blit, so cel 0 is drawn once per cycle -- at
// the wrap -- and never on the frame after a registration.
func TestEachFamilyWalksItsStripAndWraps(t *testing.T) {
	for _, f := range theFourWrappingFamilies {
		t.Run(f.name, func(t *testing.T) {
			w, _ := animWorld(t)
			f.set(w, []render.Anim{animEntry(f.celW, f.celH, 0)})

			for frame := int16(1); frame <= 2*f.frames; frame++ {
				f.render(w)

				wantMode := frame % f.frames
				got := f.get(w)[0]
				if got.Mode != wantMode {
					t.Fatalf("frame %d: Mode = %d, want %d", frame, got.Mode, wantMode)
				}
				wantSrc := render.SetRect(0, wantMode*f.celH, f.celW,
					wantMode*f.celH+f.celH)
				if got.Src != wantSrc {
					t.Fatalf("frame %d: Src = %+v, want %+v (cel %d of a %d-pixel stride)",
						frame, got.Src, wantSrc, wantMode, f.celH)
				}
			}
		})
	}
}

// TestTheWrapResetsSrcAbsolutelyNotBySubtraction is what makes a one-past-the-end seed
// harmless, and it is why the wrap cannot be written as `Src = Offset(Src, 0, -frames*celH)`.
//
// render.Anim.Mode explains the seed: RandomInt(n) can return n, so an entry can arrive with
// Mode == frames and a Src rect one cel *below* the bottom of its own strip. Nothing clamps
// it. The wrap rescues it because it assigns Src.Top and Src.Bottom outright, so the
// out-of-range rect is gone before the blit reads it -- where a subtracting wrap would leave
// it one cel out for the whole of the next cycle, blitting cel 1 while Mode says 0.
//
// Both starting points are checked, so the test states the equivalence the original relies
// on: a seed of `frames` and a seed of `frames-1` draw the same first cel.
func TestTheWrapResetsSrcAbsolutelyNotBySubtraction(t *testing.T) {
	for _, f := range theFourWrappingFamilies {
		t.Run(f.name, func(t *testing.T) {
			for _, seed := range []int16{f.frames - 1, f.frames} {
				w, _ := animWorld(t)
				f.set(w, []render.Anim{animEntry(f.celW, f.celH, seed)})
				f.render(w)

				got := f.get(w)[0]
				if got.Mode != 0 {
					t.Errorf("seeded at %d: Mode = %d after one frame, want the wrapped 0",
						seed, got.Mode)
				}
				want := render.SetRect(0, 0, f.celW, f.celH)
				if got.Src != want {
					t.Errorf("seeded at %d: Src = %+v, want %+v -- the wrap assigns Top "+
						"and Bottom rather than offsetting", seed, got.Src, want)
				}
			}
		})
	}
}

// TestAStoppedEntryDoesNothingAtAll covers the gate that only two of the four families can
// ever have set.
//
// Collecting a star stops it (`theStars[i].mode = -1`) and it keeps its slot and its cel for
// the rest of the locale. The three flame families have no way to reach this state -- StopStar
// and StopPendulum are the only writers of the flag -- so for them the gate is dead code, and
// it is folded in anyway because that is what lets one function serve all four. Asserted for
// all four so that the dead code stays dead rather than being deleted from three of them.
//
// "Nothing at all" is three things: no step, no blit, and **no work rect.** The rect is the
// one with a visible consequence, because a stopped entry that still registered its Dest
// would copy that patch of work map to the screen every other frame forever -- which is
// invisible until something else draws there, and then it is a rectangle of stale pixels that
// follows nothing.
func TestAStoppedEntryDoesNothingAtAll(t *testing.T) {
	for _, f := range theFourWrappingFamilies {
		t.Run(f.name, func(t *testing.T) {
			w, _ := animWorld(t)
			e := animEntry(f.celW, f.celH, 2)
			e.Stopped = true
			f.set(w, []render.Anim{e})

			for i := 0; i < 10; i++ {
				f.render(w)
			}

			got := f.get(w)[0]
			if got.Mode != 2 || got.Src != e.Src {
				t.Errorf("a stopped entry moved: Mode %d -> %d, Src %+v -> %+v",
					e.Mode, got.Mode, e.Src, got.Src)
			}
			if len(w.Work2Main) != 0 {
				t.Errorf("%d work rects registered for a stopped entry, want 0",
					len(w.Work2Main))
			}
		})
	}
}

// TestTheFourFamiliesRegisterAWorkRectAndNoBackRect is the whole point of the filmstrip
// design, stated as the two lists.
//
// Every other moving thing in the game costs two registrations: a work rect to get it on
// screen and a back rect to restore the wall underneath before the next draw. These cost one,
// because a cel *is* the wall with the flame on it, so an opaque blit of this frame's cel
// erases last frame's without help.
//
// Adding the missing-looking back rect is the failure this pins. It would restore bare wall
// under the flame on the frame after every draw, and since RenderFlames and RenderStars run
// on alternate frames the result is a flame visible half the time -- a 15fps strobe rather
// than a candle.
func TestTheFourFamiliesRegisterAWorkRectAndNoBackRect(t *testing.T) {
	for _, f := range theFourWrappingFamilies {
		t.Run(f.name, func(t *testing.T) {
			w, _ := animWorld(t)
			e := animEntry(f.celW, f.celH, 0)
			f.set(w, []render.Anim{e})

			f.render(w)

			if len(w.Work2Main) != 1 {
				t.Fatalf("%d work rects, want exactly 1", len(w.Work2Main))
			}
			if got := w.Work2Main[0]; got != e.Dest {
				t.Errorf("work rect = %+v, want the entry's Dest %+v: it is registered "+
					"unshrunk, because the cel covers all of it", got, e.Dest)
			}
			if len(w.Back2Work) != 0 {
				t.Errorf("%d back rects, want 0 -- the strip does that job",
					len(w.Back2Work))
			}
		})
	}
}

// TestRenderFlamesEntersForAnyOfItsThreeTables covers the shape of the original's early
// return, which is one triple test rather than three separate ones.
//
// The distinction is only observable per table: a room with coals and no candles still
// animates its coals. A port that returned on `len(Flames) == 0` would give a barbecue with
// permanently static embers and nothing else wrong, in a room that is otherwise correct.
func TestRenderFlamesEntersForAnyOfItsThreeTables(t *testing.T) {
	cases := []struct {
		name string
		set  func(w *World, e render.Anim)
		get  func(w *World) []render.Anim
		celH int16
	}{
		{"candles only", func(w *World, e render.Anim) { w.R.Flames = []render.Anim{e} },
			func(w *World) []render.Anim { return w.R.Flames }, 15},
		{"tikis only", func(w *World, e render.Anim) { w.R.TikiFlames = []render.Anim{e} },
			func(w *World) []render.Anim { return w.R.TikiFlames }, 10},
		{"coals only", func(w *World, e render.Anim) { w.R.Coals = []render.Anim{e} },
			func(w *World) []render.Anim { return w.R.Coals }, 9},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, _ := animWorld(t)
			c.set(w, animEntry(16, c.celH, 0))
			w.RenderFlames()

			if got := c.get(w)[0].Mode; got != 1 {
				t.Errorf("Mode = %d, want 1: the early return is a triple test, so one "+
					"non-empty table is enough to enter", got)
			}
		})
	}
}

// TestAnEmptyRoomCostsNothing is the early return itself, and it is the common case: most
// rooms in most houses have none of these five things in them.
func TestAnEmptyRoomCostsNothing(t *testing.T) {
	w, heard := animWorld(t)
	before := w.R.ClockFrame

	for i := 0; i < 5; i++ {
		w.RenderFlames()
		w.RenderStars()
		w.RenderPendulums()
	}

	if len(w.Work2Main) != 0 || len(w.Back2Work) != 0 {
		t.Errorf("an empty room registered %d work and %d back rects, want none",
			len(w.Work2Main), len(w.Back2Work))
	}
	if len(*heard) != 0 {
		t.Errorf("an empty room played %v", *heard)
	}
	// The clock counter does not advance either, because the early return is above the
	// increment. So a locale with no pendulum leaves ClockFrame frozen wherever the last
	// one left it -- which is fine only because the next registration reseeds it.
	if w.R.ClockFrame != before {
		t.Errorf("ClockFrame advanced to %d in a room with no pendulum; the early return "+
			"is above the increment", w.R.ClockFrame)
	}
}

// ---------------------------------------------------------------------------
// The pendulum
// ---------------------------------------------------------------------------

// pendulumWorld puts n pendulums in the room, all at cel 1 swinging the same way, and sets
// ClockFrame to what addPendulum seeds it to.
func pendulumWorld(t *testing.T, n int, toOrFro bool) (*World, *[][2]int16) {
	t.Helper()
	w, heard := animWorld(t)
	for i := 0; i < n; i++ {
		e := animEntry(32, 28, 1)
		e.Who = int16(i)
		e.ToOrFro = toOrFro
		w.R.Pendulums = append(w.R.Pendulums, e)
	}
	w.R.ClockFrame = 10
	return w, heard
}

// swing runs frames until the pendulum next moves, and returns how many that took. The gate
// means most frames do nothing, so every assertion about the swing has to be phrased in
// swings rather than in frames.
func swing(t *testing.T, w *World) int {
	t.Helper()
	before := w.R.Pendulums[0].Mode
	for n := 1; n <= 40; n++ {
		w.RenderPendulums()
		if w.R.Pendulums[0].Mode != before {
			return n
		}
	}
	t.Fatalf("no swing in 40 frames from ClockFrame %d", w.R.ClockFrame)
	return 0
}

// TestThePendulumSwingsCentreEndCentreEnd pins the four-step cycle and the sound at each end.
//
// It is four steps and not three because the direction flip happens *after* the step, so each
// end is visited once and the centre twice:
//
//	1 -> 2  flip, kTikSound
//	2 -> 1
//	1 -> 0  flip, kTokSound
//	0 -> 1
//
// So the cels drawn per cycle are 2, 1, 0, 1. That is the opposite of a real pendulum, which
// lingers at its extremes, and it is simply what ping-ponging across three cels gets you.
// Two full cycles are walked, because a swing that flipped on the wrong side of the step
// would still produce a plausible-looking first cycle and then stick at one end.
func TestThePendulumSwingsCentreEndCentreEnd(t *testing.T) {
	w, heard := pendulumWorld(t, 1, true)

	want := []struct {
		mode  int16
		sound int16 // 0 for none
	}{
		{2, TikSound}, {1, 0}, {0, TokSound}, {1, 0},
		{2, TikSound}, {1, 0}, {0, TokSound}, {1, 0},
	}
	for i, step := range want {
		swing(t, w)
		p := w.R.Pendulums[0]
		if p.Mode != step.mode {
			t.Fatalf("swing %d: Mode = %d, want %d", i+1, p.Mode, step.mode)
		}
		// Src tracks Mode by offset alone here -- there is no absolute reset, because
		// Mode never leaves 0..2 and so never needs rescuing.
		if p.Src.Top != p.Mode*28 {
			t.Fatalf("swing %d: Src.Top = %d, want %d for cel %d",
				i+1, p.Src.Top, p.Mode*28, p.Mode)
		}
		if step.sound == 0 {
			if len(*heard) != 0 {
				t.Fatalf("swing %d: heard %v mid-swing, want silence", i+1, *heard)
			}
			continue
		}
		if len(*heard) != 1 || (*heard)[0][0] != step.sound {
			t.Fatalf("swing %d: heard %v, want one sound %d", i+1, *heard, step.sound)
		}
		*heard = nil
	}
}

// TestTheTikAndTokPrioritiesAreAdjacentAndLow states the two constants, which is worth doing
// because they are the quietest sounds in the game and that is deliberate.
//
// At 200 and 201 a tick loses to almost everything else in the room, so a clock in a busy
// room is heard between events rather than over them.
func TestTheTikAndTokPrioritiesAreAdjacentAndLow(t *testing.T) {
	w, heard := pendulumWorld(t, 1, true)
	swing(t, w) // 1 -> 2, tik
	swing(t, w) // 2 -> 1
	swing(t, w) // 1 -> 0, tok

	if len(*heard) != 2 {
		t.Fatalf("heard %v, want a tik and a tok", *heard)
	}
	if (*heard)[0] != [2]int16{TikSound, TikPriority} {
		t.Errorf("tik = %v, want {%d %d}", (*heard)[0], TikSound, TikPriority)
	}
	if (*heard)[1] != [2]int16{TokSound, TokPriority} {
		t.Errorf("tok = %v, want {%d %d}", (*heard)[1], TokSound, TokPriority)
	}
}

// TestTheClockTicksUnevenly is the reason ClockFrame is a separate counter rather than a
// parity test on the frame number, and it is the most easily "fixed" thing in the file.
//
// The body runs only when the counter reads exactly 10 or exactly 15, and only the 15 resets
// it. So the gaps between swings alternate five frames and ten:
//
//	seed 10 -> 11 12 13 14 15  swing, reset       5
//	         0  1 ..     9 10  swing, no reset   10
//	           11 ..    14 15  swing, reset       5
//
// Collapsing 10-and-15 into one interval is a one-line simplification that makes the clock
// tick evenly. It would look right, sound different, and pass every other test in this file.
//
// The replay side asserts the same shape end-to-end over a real house from the trace's
// `clock` column; see internal/replay's TestTheCuckooTicksUnevenly. This one asserts it on a
// synthetic room, so a failure says which of the two numbers moved.
func TestTheClockTicksUnevenly(t *testing.T) {
	w, _ := pendulumWorld(t, 1, true)

	var gaps []int
	for i := 0; i < 8; i++ {
		gaps = append(gaps, swing(t, w))
	}

	want := []int{5, 10, 5, 10, 5, 10, 5, 10}
	for i := range want {
		if gaps[i] != want[i] {
			t.Fatalf("gaps between swings = %v, want %v", gaps, want)
		}
	}
}

// TestOneSoundPerFrameAcrossEveryPendulumInTheRoom covers playedTikTok, whose scope is the
// frame and not the entry.
//
// Three grandfather clocks in one room therefore tick *once*, and which one wins is table
// order. It is faithful, so it is pinned rather than repaired -- a per-entry sound would make
// a room of clocks a wall of noise, and the flag exists precisely to stop that. See
// TestTheFirstPendulumsSoundAlwaysWins for the audible consequence.
func TestOneSoundPerFrameAcrossEveryPendulumInTheRoom(t *testing.T) {
	w, heard := pendulumWorld(t, 3, true)

	swing(t, w) // all three step 1 -> 2 on the same frame, all three want a tik

	if len(*heard) != 1 {
		t.Errorf("three pendulums reaching an end on one frame played %v, want one sound",
			*heard)
	}
	for i := range w.R.Pendulums {
		if w.R.Pendulums[i].Mode != 2 {
			t.Errorf("pendulum %d is at cel %d, want 2: the throttle is on the sound, "+
				"not on the swing", i, w.R.Pendulums[i].Mode)
		}
	}
	if len(w.Work2Main) != 3 {
		t.Errorf("%d work rects for three pendulums, want 3", len(w.Work2Main))
	}
}

// TestTheFirstPendulumsSoundAlwaysWins is the audible consequence of the throttle, and it is
// the part that reads as a bug.
//
// Two clocks in opposite phase reach their ends on the *same* frames -- one arriving at cel 2
// while the other arrives at cel 0 -- so both want a sound on every acting frame and the
// earlier table entry takes it every time. The second clock is therefore **silent for the
// whole locale**: it swings, it draws, and it is never heard.
//
// What the room plays is exactly what the first clock alone would play, which is a proper
// alternating tick-tock, so nothing sounds wrong. That is what makes it worth a test: the
// symptom is not a strange noise, it is one of two visibly swinging clocks being mute, which
// nobody would notice and a per-entry "fix" would change into a room playing two sounds a
// frame.
func TestTheFirstPendulumsSoundAlwaysWins(t *testing.T) {
	w, heard := pendulumWorld(t, 2, true)
	w.R.Pendulums[1].ToOrFro = false // opposite phase: it is heading for cel 0

	seen := map[int16]bool{}
	for i := 0; i < 8; i++ {
		swing(t, w)
		seen[w.R.Pendulums[1].Mode] = true
	}

	// The first clock flips on every other acting frame, alternating ends, so four sounds
	// in eight swings -- and that is the entire room.
	want := [][2]int16{
		{TikSound, TikPriority}, {TokSound, TokPriority},
		{TikSound, TikPriority}, {TokSound, TokPriority},
	}
	if len(*heard) != len(want) {
		t.Fatalf("heard %v, want %v: one sound per acting frame, and both clocks act on "+
			"the same frames", *heard, want)
	}
	for i := range want {
		if (*heard)[i] != want[i] {
			t.Fatalf("heard %v, want %v", *heard, want)
		}
	}
	// Both really did swing -- the second one reached both ends, twice each. It just never
	// got a turn at the speaker. Without this the test would pass on a room where the
	// second pendulum was frozen, which is a different bug with the same sound.
	if !seen[0] || !seen[2] {
		t.Errorf("the second pendulum visited cels %v, want both ends: it is meant to be "+
			"swinging and mute, not stuck and mute", seen)
	}
}

// TestAStoppedPendulumIsSkippedButStillLetsTheRoomTick separates the two gates, which sit at
// different levels: ClockFrame decides whether the *frame* does anything, and Stopped decides
// whether an *entry* does.
//
// Collecting the cuckoo clock stops its pendulum. In a room with two clocks the other one has
// to keep going, and it has to keep the *sound*, which the C only gets right because
// playedTikTok is set inside the swing rather than at the top of the loop.
func TestAStoppedPendulumIsSkippedButStillLetsTheRoomTick(t *testing.T) {
	w, heard := pendulumWorld(t, 2, true)
	w.R.Pendulums[0].Stopped = true
	frozen := w.R.Pendulums[0]

	swing := 0
	for i := 0; i < 40 && swing == 0; i++ {
		w.RenderPendulums()
		if w.R.Pendulums[1].Mode != 1 {
			swing = i + 1
		}
	}
	if swing == 0 {
		t.Fatal("the live pendulum never swung")
	}
	if w.R.Pendulums[0] != frozen {
		t.Errorf("the stopped pendulum moved: %+v -> %+v", frozen, w.R.Pendulums[0])
	}
	if len(*heard) != 1 || (*heard)[0][0] != TikSound {
		t.Errorf("heard %v, want the live pendulum's tik: the throttle flag is set inside "+
			"the swing, so a skipped entry does not consume it", *heard)
	}
	if len(w.Work2Main) != 1 {
		t.Errorf("%d work rects, want 1 -- only the live pendulum's", len(w.Work2Main))
	}
}

// TestThePendulumRegistersNoBackRectEither is the filmstrip contract again, for the one
// family that draws on every frame it acts rather than every other one.
func TestThePendulumRegistersNoBackRectEither(t *testing.T) {
	w, _ := pendulumWorld(t, 1, true)
	for i := 0; i < 40; i++ {
		w.RenderPendulums()
	}
	if len(w.Back2Work) != 0 {
		t.Errorf("%d back rects, want 0", len(w.Back2Work))
	}
	if len(w.Work2Main) == 0 {
		t.Error("no work rects either; the pendulum never drew")
	}
}

// ---------------------------------------------------------------------------
// The strip lookup
// ---------------------------------------------------------------------------

// TestAnimStripRefusesAnIndexItCannotResolve is the port's guard rather than the original's,
// which indexes savedMaps[] unconditionally.
//
// Two things make that unsafe here and only one of them is a bug: an out-of-range index would
// be, and a slot whose Map is nil is the ordinary state of a Scene composed without art --
// which every test in this file and most of dynamics_test.go relies on. The alternative is a
// nil dereference in the middle of a frame instead of a missing flame.
func TestAnimStripRefusesAnIndexItCannotResolve(t *testing.T) {
	w, _ := animWorld(t)
	w.R.SavedMaps = append(w.R.SavedMaps, render.SavedMap{Map: nil})

	for _, i := range []int{-1, 1, 500, 0} {
		if got := w.animStrip(i); got != nil {
			t.Errorf("animStrip(%d) = %v, want nil", i, got)
		}
	}

	// And the animators survive it: a whole cycle of every family with an unresolvable
	// slot steps the cels, registers the rects, and skips only the blit.
	for _, f := range theFourWrappingFamilies {
		f.set(w, []render.Anim{animEntry(f.celW, f.celH, 0)})
	}
	w.R.Pendulums = append(w.R.Pendulums, animEntry(32, 28, 1))
	for i := 0; i < 40; i++ {
		w.RenderFlames()
		w.RenderStars()
		w.RenderPendulums()
	}
	if len(w.Work2Main) == 0 {
		t.Error("nothing registered a work rect; the bookkeeping is meant to run with or " +
			"without pixels")
	}
}
