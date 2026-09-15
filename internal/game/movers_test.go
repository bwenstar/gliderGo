package game

// The six movers (Dynamics2.c) and the seven renderers (Dynamics.c:112-286).
//
// The movers sort into three families by what "done" means -- fly-and-respawn for the
// balloon, copter and dart; bounce for the ball; drop-and-rest for the drip and fish -- and
// the tests are grouped the same way, with the two cross-cutting invariants first.
//
// Those two account for most of what these 618 lines do:
//
//	Whole == union(Dest before, Dest after)   for every mover in flight
//	EvenFrame gating follows no rule at all   and so has to be pinned per handler
//
// The first is the trailing-edge union written five different ways -- `Whole.Bottom -= VVel`
// for a rising object, `Whole.Top -= VVel` for a falling one, and one of Left/Right for a
// drifting one -- and one assertion covers all of them. The second is the opposite: the
// balloon animates on even frames and the copter every frame; the ball, drip and fish gain
// gravity on even frames and the dart never; the drip flips cels on even frames and the dart
// has no cels to flip. There is no principle to factor out, so each gate gets its own row.

import (
	"testing"

	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/render"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// unionOf is the rect union, spelled out rather than borrowed from render, so that the
// invariant below is an independent derivation and not a re-run of the code it checks.
func unionOf(a, b Rect) Rect {
	u := a
	if b.Top < u.Top {
		u.Top = b.Top
	}
	if b.Left < u.Left {
		u.Left = b.Left
	}
	if b.Bottom > u.Bottom {
		u.Bottom = b.Bottom
	}
	if b.Right > u.Right {
		u.Right = b.Right
	}
	return u
}

// inFlight registers one mover and puts it in the air at a known place and velocity, far
// enough from every retire line that the caller can run several frames without one firing.
//
// The placement is the point of the helper. Four of the six have retire tests against
// absolute room-local lines -- 8 and 310 for the balloon and the copter, 0 and RoomWide for
// the dart -- and a registration puts three of them *on* one of those lines, so a test that
// registered and stepped would retire on frame one and measure the reset instead of the
// flight.
func inFlight(t *testing.T, what int16, prep func(*Dynamic)) *World {
	t.Helper()
	w := registerOne(t, what, typicalObject(what), testWhere)
	d := &w.Dinahs[0]
	d.Moving = true
	prep(d)
	return w
}

// playerRectOf is a glider-sized box centred on r, so that SectGlider's 5px inset on all
// four sides still leaves an overlap. Glider-sized rather than merely large because
// StartGliderFadingOut resizes any Dest taller than GliderHigh, which would move the glider
// out from under the object mid-test.
//
// TestBalloonPopStopsHurtingYou opens with the control that proves this rect does collide,
// so the negative assertions that follow are not vacuous.
func playerRectOf(r Rect) player.Rect {
	left := r.Left + r.Wide()/2 - player.GliderWide/2
	top := r.Top + r.Tall()/2 - player.GliderHigh/2
	return player.Rect{
		Top: top, Left: left,
		Bottom: top + player.GliderHigh, Right: left + player.GliderWide,
	}
}

// ---------------------------------------------------------------------------
// The two cross-cutting invariants
// ---------------------------------------------------------------------------

// TestMoverWholeIsTheTrailingUnion is the one assertion that covers all six.
//
// Every mover rebuilds Whole from Dest each frame by pushing the trailing edge back one
// frame's travel, and the five spellings of that -- Bottom for a riser, Top for a faller,
// Right for a leftward drifter, Left for a rightward one, and both edges for the diagonal
// copter and dart -- all come to the same thing: the union of where it was and where it is.
// Recomputed rather than accumulated, so it cannot drift.
//
// Asserted over several frames because the interesting failure is a handler that gets the
// first frame right and then unions against a stale rect.
func TestMoverWholeIsTheTrailingUnion(t *testing.T) {
	cases := []struct {
		name string
		what int16
		fn   func(*World, int16)
		prep func(*Dynamic)
	}{
		{"balloon rising", Balloon, (*World).HandleBalloon, func(d *Dynamic) {
			d.Dest = Rect{Top: 180, Left: 200, Bottom: 210, Right: 224}
			d.VVel = -2
		}},
		{"balloon popped", Balloon, (*World).HandleBalloon, func(d *Dynamic) {
			d.Dest = Rect{Top: 100, Left: 200, Bottom: 130, Right: 224}
			d.VVel = EnemyDropSpeed
			d.Frame = 6
		}},
		{"copter leftward", CopterLf, (*World).HandleCopter, func(d *Dynamic) {
			d.Dest = Rect{Top: 100, Left: 200, Bottom: 130, Right: 232}
			d.HVel, d.VVel = -1, 2
		}},
		{"copter rightward", CopterRt, (*World).HandleCopter, func(d *Dynamic) {
			d.Dest = Rect{Top: 100, Left: 200, Bottom: 130, Right: 232}
			d.HVel, d.VVel = 1, 2
		}},
		{"dart leftward", DartLf, (*World).HandleDart, func(d *Dynamic) {
			d.Dest = Rect{Top: 100, Left: 200, Bottom: 119, Right: 264}
			d.HVel, d.VVel = -DartVelocity, 2
		}},
		{"dart rightward", DartRt, (*World).HandleDart, func(d *Dynamic) {
			d.Dest = Rect{Top: 100, Left: 200, Bottom: 119, Right: 264}
			d.HVel, d.VVel = DartVelocity, 2
		}},
		{"ball rising", Ball, (*World).HandleBall, func(d *Dynamic) {
			d.Dest = Rect{Top: 100, Left: 200, Bottom: 132, Right: 232}
			d.VVel, d.Position = -6, 300
		}},
		{"ball falling", Ball, (*World).HandleBall, func(d *Dynamic) {
			d.Dest = Rect{Top: 100, Left: 200, Bottom: 132, Right: 232}
			d.VVel, d.Position = 4, 300
		}},
		{"drip falling", Drip, (*World).HandleDrip, func(d *Dynamic) {
			d.Dest = Rect{Top: 100, Left: 200, Bottom: 112, Right: 208}
			d.VVel, d.Position, d.Frame = 3, 300, 4
		}},
		{"fish rising", Fish, (*World).HandleFish, func(d *Dynamic) {
			d.Dest = Rect{Top: 200, Left: 200, Bottom: 216, Right: 216}
			d.VVel, d.Position = -9, 290
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := inFlight(t, tc.what, tc.prep)

			for frame := 0; frame < 4; frame++ {
				w.EvenFrame = frame%2 == 0
				before := w.Dinahs[0].Dest
				tc.fn(w, 0)
				if !w.Dinahs[0].Moving {
					t.Fatalf("frame %d: retired; inFlight placed it too close to a stop line",
						frame)
				}
				after := w.Dinahs[0].Dest
				if before == after {
					t.Fatalf("frame %d: Dest did not move", frame)
				}
				if got, want := w.Dinahs[0].Whole, unionOf(before, after); got != want {
					t.Errorf("frame %d: Whole = %+v, want union(%+v, %+v) = %+v",
						frame, got, before, after, want)
				}
			}
		})
	}
}

// TestEvenFrameGatingIsPerHandler pins the pattern that has no rule behind it.
//
// Each row is one handler's answer to "does this frame's parity matter", and the value of
// the table is that it is asymmetric: a reader who assumed the gates were uniform would
// "fix" three of these seven rows.
func TestEvenFrameGatingIsPerHandler(t *testing.T) {
	cases := []struct {
		name string
		what int16
		fn   func(*World, int16)
		prep func(*Dynamic)
		// read is the field the gate is supposed to control.
		read func(*Dynamic) int16
		// gated is whether an odd frame leaves that field alone; never is the stronger
		// claim that no frame touches it at all.
		gated bool
		never bool
		note  string
	}{
		{
			name: "balloon animation", what: Balloon, fn: (*World).HandleBalloon,
			prep: func(d *Dynamic) {
				d.Dest = Rect{Top: 180, Left: 200, Bottom: 210, Right: 224}
				d.VVel = -2
			},
			read: func(d *Dynamic) int16 { return d.Frame }, gated: true,
			note: "the balloon animates at 15 fps against the copter's 30",
		},
		{
			name: "copter animation", what: CopterLf, fn: (*World).HandleCopter,
			prep: func(d *Dynamic) {
				d.Dest = Rect{Top: 100, Left: 200, Bottom: 130, Right: 232}
				d.HVel, d.VVel = -1, 2
			},
			read: func(d *Dynamic) int16 { return d.Frame },
			note: "the rotor spins every frame",
		},
		{
			name: "dart animation", what: DartRt, fn: (*World).HandleDart,
			prep: func(d *Dynamic) {
				d.Dest = Rect{Top: 100, Left: 200, Bottom: 119, Right: 264}
				d.HVel, d.VVel = DartVelocity, 2
			},
			read: func(d *Dynamic) int16 { return d.Frame }, never: true,
			note: "a dart has no frame counter at all: its four cels are two directions",
		},
		{
			name: "ball gravity", what: Ball, fn: (*World).HandleBall,
			prep: func(d *Dynamic) {
				d.Dest = Rect{Top: 100, Left: 200, Bottom: 132, Right: 232}
				d.VVel, d.Position = -6, 300
			},
			read: func(d *Dynamic) int16 { return d.VVel }, gated: true,
			note: "half-rate gravity is what makes the arc take twice as long to come down",
		},
		{
			name: "dart gravity", what: DartRt, fn: (*World).HandleDart,
			prep: func(d *Dynamic) {
				d.Dest = Rect{Top: 100, Left: 200, Bottom: 119, Right: 264}
				d.HVel, d.VVel = DartVelocity, 2
			},
			read: func(d *Dynamic) int16 { return d.VVel }, never: true,
			note: "a dart has no gravity either, so it crosses the room on a constant slope",
		},
		{
			name: "drip gravity", what: Drip, fn: (*World).HandleDrip,
			prep: func(d *Dynamic) {
				d.Dest = Rect{Top: 100, Left: 200, Bottom: 112, Right: 208}
				d.VVel, d.Position = 3, 300
			},
			read: func(d *Dynamic) int16 { return d.VVel }, gated: true,
		},
		{
			name: "fish gravity", what: Fish, fn: (*World).HandleFish,
			prep: func(d *Dynamic) {
				d.Dest = Rect{Top: 200, Left: 200, Bottom: 216, Right: 216}
				d.VVel, d.Position = -9, 290
			},
			read: func(d *Dynamic) int16 { return d.VVel }, gated: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			step := func(even bool) bool {
				w := inFlight(t, tc.what, tc.prep)
				w.EvenFrame = even
				was := tc.read(&w.Dinahs[0])
				tc.fn(w, 0)
				return tc.read(&w.Dinahs[0]) != was
			}
			onOdd, onEven := step(false), step(true)

			switch {
			case tc.never:
				if onOdd || onEven {
					t.Errorf("changed on odd = %v, on even = %v, want neither (%s)",
						onOdd, onEven, tc.note)
				}
			case tc.gated:
				if onOdd || !onEven {
					t.Errorf("changed on odd = %v, on even = %v, want false and true (%s)",
						onOdd, onEven, tc.note)
				}
			default:
				if !onOdd || !onEven {
					t.Errorf("changed on odd = %v, on even = %v, want both true (%s)",
						onOdd, onEven, tc.note)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// The three that fly and respawn
// ---------------------------------------------------------------------------

// TestEnemyWaitingPuffsExactlyOnce is the shared idle arm, and the two-branch structure in
// it is what makes the guarantee.
//
// With a reload period of more than four frames the timer passes through StartSparkle on the
// way down, so the puff fires there and the launch itself is silent. With a shorter period
// the timer never equals 4, so `Count < StartSparkle` emits the puff at launch instead.
// Neither case can produce two, and the player always gets a sparkle -- which is the whole
// point, because an enemy that appeared unannounced would be unavoidable.
//
// The periods here are the reachable ones: registration computes Count as
// `(Delay * 6) / TicksPerFrame`, so every period in a real house is a multiple of three. See
// TestEnemyWaitingHasOneSilentPeriodAndItIsUnreachable for the hole that leaves.
func TestEnemyWaitingPuffsExactlyOnce(t *testing.T) {
	for _, period := range []int16{3, 6, 9, 33} {
		for _, what := range []int16{Balloon, CopterLf, DartRt} {
			w := registerOne(t, what, typicalObject(what), testWhere)
			var snd soundLog
			snd.install(w)
			w.Dinahs[0].Count = period
			w.Dinahs[0].Timer = period
			w.Dinahs[0].Moving = false

			puffs, launched := 0, -1
			for frame := 1; frame <= int(period)+2 && launched == -1; frame++ {
				before := w.NumSparkles
				w.enemyWaiting(0)
				if w.NumSparkles != before {
					puffs++
				}
				if w.Dinahs[0].Moving {
					launched = frame
				}
			}

			if launched != int(period) {
				t.Errorf("%#x period %d: launched on frame %d, want %d",
					what, period, launched, period)
			}
			if puffs != 1 {
				t.Errorf("%#x period %d: %d puffs, want exactly 1", what, period, puffs)
			}
			if len(snd.played) != 1 || snd.played[0] != EnemyInSound {
				t.Errorf("%#x period %d: sounds %v, want one EnemyInSound",
					what, period, snd.played)
			}
		}
	}
}

// TestEnemyWaitingHasOneSilentPeriodAndItIsUnreachable pins an off-by-one and the reason it
// never fires.
//
// A period of exactly 4 satisfies neither branch: the timer counts 3, 2, 1, 0 and so never
// *equals* StartSparkle, and `Count < StartSparkle` is false for 4. So an enemy on a
// four-frame reload would launch in complete silence with no warning puff -- the one case the
// two branches were meant to cover between them.
//
// It cannot happen. Count comes from `(Delay * 6) / TicksPerFrame` with Delay a single
// authored byte, which is `Delay * 3`, so every reachable period is a multiple of three and 4
// is not one. Asserted over all 256 authorable delays rather than argued, because this is
// exactly the kind of gap a later "improvement" to the registration arithmetic would open up.
func TestEnemyWaitingHasOneSilentPeriodAndItIsUnreachable(t *testing.T) {
	w := registerOne(t, Balloon, typicalObject(Balloon), testWhere)
	var snd soundLog
	snd.install(w)
	w.Dinahs[0].Count = 4
	w.Dinahs[0].Timer = 4
	w.Dinahs[0].Moving = false

	for i := 0; i < 4; i++ {
		w.enemyWaiting(0)
	}
	if !w.Dinahs[0].Moving {
		t.Fatalf("a period of 4 did not launch after 4 frames")
	}
	if w.NumSparkles != 0 || len(snd.played) != 0 {
		t.Errorf("period 4: %d sparkles and sounds %v; the point of this test is that it is "+
			"the silent one", w.NumSparkles, snd.played)
	}

	for delay := 0; delay < 256; delay++ {
		if period := (int16(delay) * 6) / TicksPerFrame; period == StartSparkle {
			t.Fatalf("authored delay %d yields period %d == StartSparkle, so the silent case "+
				"is reachable after all", delay, period)
		}
	}
}

// TestEnemyWaitingFreezesWhileSwitchedOff: nothing happens at all while Active is false, so
// switching an enemy back on resumes its countdown rather than restarting it. There is no
// `else` in the C, which is what makes this true.
func TestEnemyWaitingFreezesWhileSwitchedOff(t *testing.T) {
	w := registerOne(t, Balloon, typicalObject(Balloon), testWhere)
	w.Dinahs[0].Active = false
	w.Dinahs[0].Timer = 9
	before := w.Dinahs[0]

	for i := 0; i < 20; i++ {
		w.enemyWaiting(0)
	}

	if w.Dinahs[0] != before {
		t.Errorf("a switched-off enemy's slot changed over 20 frames\nbefore %+v\nafter  %+v",
			before, w.Dinahs[0])
	}
}

// TestEnemyRetireOffsetsTheRectButNotTheSparkle pins two adjacent lines that disagree, on
// purpose.
//
// The work rect is Whole offset by playOrigin; the sparkle is Dest *un-offset*, because
// AddSparkle adds playOrigin itself. Handing it a screen rect would put the puff one whole
// origin down and to the right. The C makes the asymmetry look like a slip by reusing one
// local named `dest` for both.
func TestEnemyRetireOffsetsTheRectButNotTheSparkle(t *testing.T) {
	w := inFlight(t, Balloon, func(d *Dynamic) {
		d.Dest = Rect{Top: 100, Left: 200, Bottom: 130, Right: 224}
		d.Whole = Rect{Top: 96, Left: 200, Bottom: 130, Right: 224}
	})
	clearRects(w)
	oh, ov := w.R.V.OriginH, w.R.V.OriginV

	w.enemyRetire(0)

	if got, want := w.Work2Main[0], render.Offset(w.Dinahs[0].Whole, oh, ov); got != want {
		t.Errorf("work rect = %+v, want Whole offset by the play origin = %+v", got, want)
	}
	// AddSparkle offsets what it is given, so it has to be given a room-local rect.
	want := render.CenterIn(render.SparkleSrc[0], render.Offset(w.Dinahs[0].Dest, oh, ov))
	if got := w.Sparkles[0].Bounds; got != want {
		t.Errorf("sparkle bounds = %+v, want %+v -- the play origin was added twice", got, want)
	}
}

// TestBalloonPopStopsHurtingYou is the VVel state trick and its consequence.
//
// There is no popped flag: a rising balloon has a negative velocity, and popping it writes
// EnemyDropSpeed into VVel, which is how the next frame takes the other arm. That arm calls
// neither the band test nor the collision check -- **a popped balloon is harmless** and falls
// through the player without touching them.
func TestBalloonPopStopsHurtingYou(t *testing.T) {
	rising := func(d *Dynamic) {
		d.Dest = Rect{Top: 180, Left: 200, Bottom: 210, Right: 224}
		d.VVel = -2
	}

	// The control. Without it every assertion below would pass against a glider that was
	// simply out of reach.
	t.Run("an intact balloon kills", func(t *testing.T) {
		w := inFlight(t, Balloon, rising)
		var snd soundLog
		snd.install(w)
		w.Foil = 0
		w.P1.Mode = player.GliderNormal
		w.P1.Dest = playerRectOf(w.Dinahs[0].Dest)

		w.HandleBalloon(0)

		if w.P1.Mode != player.GliderFadingOut {
			t.Fatalf("glider mode = %d, want GliderFadingOut = %d -- playerRectOf does not "+
				"overlap, so the rest of this test would be vacuous",
				w.P1.Mode, player.GliderFadingOut)
		}
		snd.is(t, player.FadeOutSound)
	})

	w := inFlight(t, Balloon, rising)
	var snd soundLog
	snd.install(w)

	// A band across the balloon. NumBands is 0 for the whole game until 1.5e, so the three
	// band arms are only reachable from a test until then.
	w.NumBands = 1
	w.BandList[0].Dest = w.Dinahs[0].Dest

	w.EvenFrame = false
	w.HandleBalloon(0)

	if w.Dinahs[0].Frame != 6 {
		t.Errorf("Frame = %d after the pop, want the first burst cel 6", w.Dinahs[0].Frame)
	}
	if w.Dinahs[0].VVel != EnemyDropSpeed {
		t.Errorf("VVel = %d, want EnemyDropSpeed = %d", w.Dinahs[0].VVel, EnemyDropSpeed)
	}
	snd.is(t, PopSound)

	// The pop frame does not also move: the band arm and the move arm are the two halves of
	// one if/else.
	if got := w.Dinahs[0].Dest; got != (Rect{Top: 180, Left: 200, Bottom: 210, Right: 224}) {
		t.Errorf("Dest = %+v; the pop frame must not also move", got)
	}

	// Now park a glider inside the falling balloon. It has to pass straight through.
	w.Foil = 0
	w.P1.Mode = player.GliderNormal
	snd = soundLog{}
	snd.install(w)

	for frame := 0; frame < 3; frame++ {
		w.EvenFrame = frame%2 == 0
		w.P1.Dest = playerRectOf(w.Dinahs[0].Dest)
		w.HandleBalloon(0)
	}

	if w.P1.Mode != player.GliderNormal {
		t.Errorf("glider mode = %d, want GliderNormal -- a popped balloon killed the player",
			w.P1.Mode)
	}
	snd.is(t)

	// And the band cannot re-pop it: the falling arm has no band test.
	if w.Dinahs[0].VVel != EnemyDropSpeed {
		t.Errorf("VVel = %d, want it still falling at %d", w.Dinahs[0].VVel, EnemyDropSpeed)
	}
}

// TestBalloonRetiresAtEitherEnd: the retire test fires at the ceiling and at the floor, so a
// balloon that made it up and a popped one that came down take the same exit and both sparkle
// out. The reset puts it back on the floor, and **only Dest.Bottom is placed** -- Left and
// Right are left wherever the balloon was, which is where it started, since nothing ever
// moves a balloon sideways.
func TestBalloonRetiresAtEitherEnd(t *testing.T) {
	for _, tc := range []struct {
		name string
		prep func(*Dynamic)
	}{
		{"reached the ceiling", func(d *Dynamic) {
			d.Dest = Rect{Top: BalloonStop + 2, Left: 200, Bottom: BalloonStop + 32, Right: 224}
			d.VVel = -2
		}},
		{"popped and reached the floor", func(d *Dynamic) {
			d.Dest = Rect{Top: BalloonStart - 34, Left: 200, Bottom: BalloonStart - 4, Right: 224}
			d.VVel = EnemyDropSpeed
			d.Frame = 6
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := inFlight(t, Balloon, tc.prep)
			var snd soundLog
			snd.install(w)
			clearRects(w)
			w.Dinahs[0].Count = 33
			w.Dinahs[0].Timer = 0
			left, right := w.Dinahs[0].Dest.Left, w.Dinahs[0].Dest.Right

			w.EvenFrame = true
			w.HandleBalloon(0)

			d := w.Dinahs[0]
			if d.Moving {
				t.Fatalf("Moving = true; the balloon did not retire")
			}
			if d.VVel != -2 {
				t.Errorf("VVel = %d, want -2 (intact and rising again)", d.VVel)
			}
			if d.Timer != d.Count {
				t.Errorf("Timer = %d, want Count = %d", d.Timer, d.Count)
			}
			if d.Dest.Bottom != BalloonStart {
				t.Errorf("Dest.Bottom = %d, want BalloonStart = %d", d.Dest.Bottom, BalloonStart)
			}
			if want := BalloonStart - render.BalloonSrc[0].Tall(); d.Dest.Top != want {
				t.Errorf("Dest.Top = %d, want %d", d.Dest.Top, want)
			}
			if d.Dest.Left != left || d.Dest.Right != right {
				t.Errorf("Dest horizontal = (%d, %d), want the untouched (%d, %d)",
					d.Dest.Left, d.Dest.Right, left, right)
			}
			if d.Whole != d.Dest {
				t.Errorf("Whole = %+v, want Dest = %+v", d.Whole, d.Dest)
			}

			// enemyRetire: erase the trail, puff where it was, play the leaving sound.
			work, back := rectCounts(w)
			if work != 1 || back != 0 {
				t.Errorf("rects (work %d, back %d), want (1, 0)", work, back)
			}
			if w.NumSparkles != 1 {
				t.Errorf("NumSparkles = %d, want 1", w.NumSparkles)
			}
			snd.is(t, EnemyOutSound)
		})
	}
}

// TestCopterShotDropsHarmlessly is the same state trick on the other axis: shooting a copter
// zeroes HVel, so it stops drifting sideways and the next frame takes the falling arm --
// which, like the balloon's, neither tests for bands nor checks the gliders.
func TestCopterShotDropsHarmlessly(t *testing.T) {
	w := inFlight(t, CopterLf, func(d *Dynamic) {
		d.Dest = Rect{Top: 100, Left: 200, Bottom: 130, Right: 232}
		d.HVel, d.VVel = -1, 2
	})
	var snd soundLog
	snd.install(w)
	w.NumBands = 1
	w.BandList[0].Dest = w.Dinahs[0].Dest

	w.HandleCopter(0)

	if w.Dinahs[0].Frame != 8 {
		t.Errorf("Frame = %d, want the first crumple cel 8", w.Dinahs[0].Frame)
	}
	if w.Dinahs[0].HVel != 0 || w.Dinahs[0].VVel != EnemyDropSpeed {
		t.Errorf("(HVel, VVel) = (%d, %d), want (0, %d)",
			w.Dinahs[0].HVel, w.Dinahs[0].VVel, EnemyDropSpeed)
	}
	snd.is(t, PaperCrunchSound)

	// The crumple cels loop: 8, 9, 8, 9 all the way down. There is no "gone" cel.
	seen := map[int16]int{}
	for i := 0; i < 6; i++ {
		w.HandleCopter(0)
		seen[w.Dinahs[0].Frame]++
	}
	if len(seen) != 2 || seen[8] == 0 || seen[9] == 0 {
		t.Errorf("crumple cels seen = %v, want 8 and 9 alternating", seen)
	}
}

// TestCopterResetPlacesFourEdges pins the mixture of a derived height and a literal width.
//
// Three edges come from CopterStart and Position and the fourth is `Left + 32`, a bare
// literal where the vertical uses RectTall(copterSrc[0]). Both are 32 and 30 for the shipped
// art, so the mixture is harmless -- and it is transcribed as written because that is what a
// copter sheet of a different width would have to match.
func TestCopterResetPlacesFourEdges(t *testing.T) {
	for _, what := range []int16{CopterLf, CopterRt} {
		w := inFlight(t, what, func(d *Dynamic) {
			d.Dest = Rect{Top: CopterStop - 2, Left: 200, Bottom: CopterStop + 28, Right: 232}
			d.HVel, d.VVel = 1, 2
			d.Position = 77
			d.Count = 33
		})
		w.HandleCopter(0)

		d := w.Dinahs[0]
		if d.Moving {
			t.Fatalf("%#x: did not retire at CopterStop", what)
		}
		if d.Dest.Top != CopterStart {
			t.Errorf("%#x: Dest.Top = %d, want CopterStart = %d", what, d.Dest.Top, CopterStart)
		}
		if want := CopterStart + render.CopterSrc[0].Tall(); d.Dest.Bottom != want {
			t.Errorf("%#x: Dest.Bottom = %d, want %d", what, d.Dest.Bottom, want)
		}
		if d.Dest.Left != 77 {
			t.Errorf("%#x: Dest.Left = %d, want Position = 77", what, d.Dest.Left)
		}
		if d.Dest.Right != 77+32 {
			t.Errorf("%#x: Dest.Right = %d, want Left + the literal 32", what, d.Dest.Right)
		}
		wantHVel := int16(1)
		if what == CopterLf {
			wantHVel = -1
		}
		if d.HVel != wantHVel || d.VVel != 2 {
			t.Errorf("%#x: (HVel, VVel) = (%d, %d), want (%d, 2)", what, d.HVel, d.VVel, wantHVel)
		}
	}
}

// TestDartFliesAConstantSlope is the consequence of the dart having no gravity: it drops the
// same two pixels every frame, so a dart always arrives at the same height on the far wall.
// That is what makes rooms full of them learnable.
func TestDartFliesAConstantSlope(t *testing.T) {
	w := inFlight(t, DartRt, func(d *Dynamic) {
		d.Dest = Rect{Top: 60, Left: 40, Bottom: 79, Right: 104}
		d.HVel, d.VVel = DartVelocity, 2
	})

	var drops []int16
	for frame := 0; frame < 8; frame++ {
		w.EvenFrame = frame%2 == 0
		before := w.Dinahs[0].Dest.Top
		w.HandleDart(0)
		if !w.Dinahs[0].Moving {
			t.Fatalf("frame %d: retired early", frame)
		}
		drops = append(drops, w.Dinahs[0].Dest.Top-before)
		if w.Dinahs[0].VVel != 2 {
			t.Fatalf("frame %d: VVel = %d, want a constant 2", frame, w.Dinahs[0].VVel)
		}
	}
	for i, d := range drops {
		if d != 2 {
			t.Errorf("frame %d fell %d pixels, want a constant 2 (drops %v)", i, d, drops)
		}
	}
}

// TestDartRetiresByThreeExits is the only three-way retire test in Dynamics2.c, because a
// dart can leave by either side wall or by the floor.
//
// RoomWide is the right-hand test even though the dart is drawn from a 64-wide cel: it is
// Dest.Right that has to reach 512, so the visible dart is already outside the room by the
// time it retires.
func TestDartRetiresByThreeExits(t *testing.T) {
	cases := []struct {
		name string
		what int16
		prep func(*Dynamic)
	}{
		{"left wall", DartLf, func(d *Dynamic) {
			d.Dest = Rect{Top: 60, Left: 4, Bottom: 79, Right: 68}
			d.HVel, d.VVel = -DartVelocity, 2
		}},
		{"right wall", DartRt, func(d *Dynamic) {
			d.Dest = Rect{Top: 60, Left: RoomWide - 68, Bottom: 79, Right: RoomWide - 4}
			d.HVel, d.VVel = DartVelocity, 2
		}},
		{"floor", DartRt, func(d *Dynamic) {
			d.Dest = Rect{Top: DartStop - 21, Left: 200, Bottom: DartStop - 2, Right: 264}
			d.HVel, d.VVel = DartVelocity, 2
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := inFlight(t, tc.what, tc.prep)
			w.Dinahs[0].Count = 33
			w.Dinahs[0].Position = 55
			var snd soundLog
			snd.install(w)

			w.HandleDart(0)

			d := w.Dinahs[0]
			if d.Moving {
				t.Fatalf("Moving = true; the dart did not retire")
			}
			if d.VVel != 2 || d.Timer != d.Count {
				t.Errorf("(VVel, Timer) = (%d, %d), want (2, Count = %d)", d.VVel, d.Timer, d.Count)
			}
			if d.Dest.Top != 55 {
				t.Errorf("Dest.Top = %d, want Position = 55", d.Dest.Top)
			}
			if want := int16(55) + render.DartSrc[0].Tall(); d.Dest.Bottom != want {
				t.Errorf("Dest.Bottom = %d, want %d", d.Dest.Bottom, want)
			}
			// The reset flies it in from the wall its type points away from, and Frame is
			// the direction rather than an animation phase.
			if tc.what == DartLf {
				if d.Frame != 0 || d.HVel != -DartVelocity || d.Dest.Right != RoomWide {
					t.Errorf("leftward reset: Frame %d, HVel %d, Right %d; want 0, %d, %d",
						d.Frame, d.HVel, d.Dest.Right, -DartVelocity, RoomWide)
				}
				if want := RoomWide - render.DartSrc[0].Wide(); d.Dest.Left != want {
					t.Errorf("leftward reset: Left = %d, want %d", d.Dest.Left, want)
				}
			} else {
				if d.Frame != 2 || d.HVel != DartVelocity || d.Dest.Left != 0 {
					t.Errorf("rightward reset: Frame %d, HVel %d, Left %d; want 2, %d, 0",
						d.Frame, d.HVel, d.Dest.Left, DartVelocity)
				}
				if want := render.DartSrc[0].Wide(); d.Dest.Right != want {
					t.Errorf("rightward reset: Right = %d, want %d", d.Dest.Right, want)
				}
			}
			snd.is(t, EnemyOutSound)
		})
	}
}

// ---------------------------------------------------------------------------
// The ball
// ---------------------------------------------------------------------------

// TestRestingBallStillCollides is the ball's first oddity, and it is the one that shows up as
// level design: the collision test sits at the very top of the function, outside the moving
// check, so **a ball at rest still kills.** Every other mover only touches you while it is
// moving. A ball that has run down its bounces sits on the floor as a permanent hazard that
// looks like scenery -- which is exactly what the rooms that use one are built around.
func TestRestingBallStillCollides(t *testing.T) {
	w := registerOne(t, Ball, typicalObject(Ball), testWhere)
	var snd soundLog
	snd.install(w)

	w.Dinahs[0].Moving = false
	w.Dinahs[0].Active = false // and it will not start moving either
	w.Foil = 0
	w.P1.Mode = player.GliderNormal
	w.P1.Dest = playerRectOf(w.Dinahs[0].Dest)

	w.HandleBall(0)

	if w.P1.Mode != player.GliderFadingOut {
		t.Errorf("glider mode = %d, want GliderFadingOut; a resting ball has to be lethal",
			w.P1.Mode)
	}
	if w.Dinahs[0].Moving {
		t.Errorf("Moving = true; an inactive resting ball must stay put")
	}
	snd.is(t, player.FadeOutSound)
}

// TestBallIdleArmWritesEvenFrame is the ball's second oddity: a write to the global frame
// parity flag from inside an object handler.
//
// It knocks the flag out of step with Frame, so the candles in that room can flicker on the
// frames a room without a ball flickers its stars. What it is *for* is the ball itself: the
// write makes the ball's own first frame of gravity land immediately, which is what lets the
// reverse-engineered launch velocity reach the height the author typed.
//
// How long the shift lasts is TestBallResetsParityRatherThanTogglingIt's subject, and the
// answer is not "forever" -- the write is an assignment, so it depends on the parity it lands
// on, and in every shipped ball room it cancels a second write made one frame earlier.
func TestBallIdleArmWritesEvenFrame(t *testing.T) {
	w := registerOne(t, Ball, typicalObject(Ball), testWhere)
	w.Dinahs[0].Moving = false
	w.Dinahs[0].Active = true
	w.Dinahs[0].Count = -9
	w.EvenFrame = false

	w.HandleBall(0)

	if !w.EvenFrame {
		t.Errorf("EvenFrame = false; the idle arm's global write is missing")
	}
	if !w.Dinahs[0].Moving || w.Dinahs[0].VVel != -9 {
		t.Errorf("(Moving, VVel) = (%v, %d), want (true, Count = -9)",
			w.Dinahs[0].Moving, w.Dinahs[0].VVel)
	}

	// An *inactive* resting ball leaves the parity alone -- the write is inside the Active
	// test, not beside it.
	w = registerOne(t, Ball, typicalObject(Ball), testWhere)
	w.Dinahs[0].Moving = false
	w.Dinahs[0].Active = false
	w.EvenFrame = false
	w.HandleBall(0)
	if w.EvenFrame {
		t.Errorf("EvenFrame = true; an inactive ball must not touch the global parity")
	}
}

// TestBallResetsParityRatherThanTogglingIt is how long the shift above actually lasts, and the
// answer is "it depends", which is worth pinning because it is easy to overstate in either
// direction.
//
// Both writers of EvenFrame outside the loop head assign `true`; neither toggles. So the effect
// of a write is not "flip the parity" but "force it to a known phase", and whether that is a
// visible desynchronisation depends entirely on the parity it lands on. A second write on a
// later frame does not compound the first -- it replaces it.
//
// The two writers are one frame apart in every shipped ball room, and that is the case this
// test walks:
//
//	composition   launchVelocityHalfRate assigns true   (hazard H1, before frame 1)
//	loop head     toggles it to false
//	frame 1       HandleBall's idle arm assigns true    (the launch phase-lock)
//
// which lands on exactly the value an undisturbed toggle would have produced. So the divergence
// is one frame wide, at composition time, and then gone. Every kBall in CD Demo House is
// authored `initial 1`, so this is the only sequence the shipped game ever runs -- see
// replay.TestABallBreaksTheEvenFrameInvariant, which observes the same cancellation end to end
// in three real rooms.
//
// A ball switched on by a trigger dozens of frames later has no second write to cancel it, and
// that shift does persist. Nothing here relies on the shipped houses' authoring, so a new house
// that switches a ball on mid-room is free to hit the other case.
func TestBallResetsParityRatherThanTogglingIt(t *testing.T) {
	// An assignment, not a toggle: from either parity the idle arm lands on true.
	for _, from := range []bool{false, true} {
		w := registerOne(t, Ball, typicalObject(Ball), testWhere)
		w.Dinahs[0].Moving = false
		w.Dinahs[0].Active = true
		w.Dinahs[0].Count = -9
		w.EvenFrame = from

		w.HandleBall(0)

		if !w.EvenFrame {
			t.Errorf("from EvenFrame = %v the idle arm produced false; it assigns true "+
				"unconditionally and does not toggle", from)
		}
	}

	// Registration writes it too, and that is hazard H1: nothing in AddDynamicObject reads
	// EvenFrame, so this write is collateral -- the author's `evenFrame` resolved to the global
	// where he meant a second loop-local.
	w := dynaWorld(t)
	w.EvenFrame = false
	w.R.NumLights = 3
	if slot := w.AddDynamicObject(Ball, testWhere, typicalObject(Ball),
		testRoom, testSlot, true); slot != 0 {
		t.Fatalf("AddDynamicObject(ball) = %d, want slot 0", slot)
	}
	if !w.EvenFrame {
		t.Fatalf("composing a ball left EvenFrame false; the H1 write is missing and the " +
			"cancellation below would not be what the shipped game does")
	}

	// The sequence, one frame at a time. The ball registered Moving = false and Active = true,
	// which is what `initial 1` gives, so frame 1's idle arm fires.
	if w.Dinahs[0].Moving || !w.Dinahs[0].Active {
		t.Fatalf("(Moving, Active) = (%v, %v) after composition, want (false, true)",
			w.Dinahs[0].Moving, w.Dinahs[0].Active)
	}
	w.EvenFrame = !w.EvenFrame // the loop head, PlayGame's `evenFrame = !evenFrame`
	if w.EvenFrame {
		t.Fatalf("the toggle did not run")
	}
	w.HandleBall(0)

	// Frame 1's expected parity for an undisturbed run is true, and that is where the pair of
	// writes has landed: the second undid the first.
	if !w.EvenFrame {
		t.Errorf("EvenFrame = false at the end of frame 1, want true -- the idle arm's write " +
			"is what cancels the composition's")
	}
	if !w.Dinahs[0].Moving {
		t.Errorf("the ball did not launch on frame 1, so there was no second write")
	}

	// And from here the flag toggles undisturbed, because a moving ball never takes the idle
	// arm again. Four frames is enough to show it tracks Frame's parity.
	for frame := int64(2); frame <= 5; frame++ {
		w.EvenFrame = !w.EvenFrame
		w.HandleBall(0)
		if want := frame%2 == 1; w.EvenFrame != want {
			t.Errorf("frame %d: EvenFrame = %v, want %v -- a moving ball must not write it "+
				"again", frame, w.EvenFrame, want)
		}
	}
}

// TestBallBounceRecurrenceTruncatesTowardZero is the third oddity, and it is exact.
//
// An inactive ball keeps three quarters of its impact speed with C's truncation toward zero,
// so the chain of impact speeds is 8, 6, 4, 3, 2, 1 and the ball stops on the sixth bounce.
// `VVel == 0` is the stop test rather than `>= 0`, and it is reached exactly because 3/4 of 1
// truncates to 0 -- widen the test to `>= 0` and the ball stops two bounces early.
//
// Each row drives one bounce directly, because a free-running ball's impact speeds depend on
// the half-rate gravity and would obscure the arithmetic this is about.
// TestBallComesToRestWhenInactive is the free run.
func TestBallBounceRecurrenceTruncatesTowardZero(t *testing.T) {
	cases := []struct {
		impact, want int16
		stops        bool
	}{
		{8, -6, false},
		{6, -4, false}, // 18/4 truncates to 4, not 5
		{4, -3, false},
		{3, -2, false}, // 9/4 truncates to 2
		{2, -1, false},
		{1, 0, true}, // 3/4 truncates to 0, which is the stop
	}

	for _, tc := range cases {
		impact := tc.impact
		w := inFlight(t, Ball, func(d *Dynamic) {
			d.Position = 300
			d.Dest = Rect{Top: 300 - 32 - impact, Left: 200, Bottom: 300 - impact, Right: 232}
			d.VVel, d.Count = impact, -8
			d.Active = false
		})
		var snd soundLog
		snd.install(w)

		w.EvenFrame = true
		w.HandleBall(0)

		d := w.Dinahs[0]
		if d.Dest.Bottom != 300 {
			t.Fatalf("impact %d: Dest.Bottom = %d, want the floor line 300 -- no bounce",
				tc.impact, d.Dest.Bottom)
		}
		if d.VVel != tc.want {
			t.Errorf("impact %d: VVel = %d, want %d", tc.impact, d.VVel, tc.want)
		}
		if d.Moving == tc.stops {
			t.Errorf("impact %d: Moving = %v, want %v", tc.impact, d.Moving, !tc.stops)
		}
		if d.Dest.Top != 300-32 {
			t.Errorf("impact %d: Dest.Top = %d, want Bottom - the literal 32",
				tc.impact, d.Dest.Top)
		}
		snd.is(t, BounceSound)
	}
}

// TestBallComesToRestWhenInactive is the free run: a ball switched off does stop, and the
// bounce that stops it is the one whose impact speed truncates to zero.
func TestBallComesToRestWhenInactive(t *testing.T) {
	w := inFlight(t, Ball, func(d *Dynamic) {
		d.Position = 300
		d.Dest = Rect{Top: 268, Left: 200, Bottom: 300, Right: 232}
		d.VVel, d.Count = -8, -8
		d.Active = false
	})

	frames := 0
	for ; frames < 2000 && w.Dinahs[0].Moving; frames++ {
		w.EvenFrame = frames%2 == 0
		w.HandleBall(0)
	}
	if w.Dinahs[0].Moving {
		t.Fatalf("still bouncing after %d frames; an inactive ball has to run down", frames)
	}
	if w.Dinahs[0].VVel != 0 {
		t.Errorf("VVel = %d at rest, want 0", w.Dinahs[0].VVel)
	}
	if w.Dinahs[0].Dest.Bottom != 300 {
		t.Errorf("Dest.Bottom = %d at rest, want the floor line 300", w.Dinahs[0].Dest.Bottom)
	}
}

// TestBallBouncesForeverWhenActive: at each bounce an active ball's velocity is reset to
// Count -- the launch speed -- so it never loses energy, and the only sound it ever makes is
// the bounce.
func TestBallBouncesForeverWhenActive(t *testing.T) {
	w := inFlight(t, Ball, func(d *Dynamic) {
		d.Position = 300
		d.Dest = Rect{Top: 268, Left: 200, Bottom: 300, Right: 232}
		d.VVel, d.Count = -8, -8
		d.Active = true
	})
	var snd soundLog
	snd.install(w)

	for frame := 0; frame < 300; frame++ {
		w.EvenFrame = frame%2 == 0
		w.HandleBall(0)
		if !w.Dinahs[0].Moving {
			t.Fatalf("frame %d: an active ball stopped bouncing", frame)
		}
	}

	bounces := 0
	for _, s := range snd.played {
		if s != BounceSound {
			t.Fatalf("sounds %v, want only BounceSound", snd.played)
		}
		bounces++
	}
	if bounces < 5 {
		t.Errorf("%d bounces in 300 frames, want several", bounces)
	}
}

// TestBallSquashCelLastsOneFrame: cel 1 is the squashed ball, set on the bounce frame only if
// the ball is still moving, and cleared to 0 by the very next airborne frame.
//
// The `if Moving` around it is why a ball that stops on its last bounce rests as a *round*
// ball rather than a permanently squashed one.
func TestBallSquashCelLastsOneFrame(t *testing.T) {
	w := inFlight(t, Ball, func(d *Dynamic) {
		d.Position = 300
		d.Dest = Rect{Top: 266, Left: 200, Bottom: 298, Right: 232}
		d.VVel, d.Count = 4, -8
		d.Active = true
	})

	w.EvenFrame = true
	w.HandleBall(0) // the bounce
	if w.Dinahs[0].Frame != 1 {
		t.Fatalf("Frame = %d on the bounce frame, want the squashed cel 1", w.Dinahs[0].Frame)
	}
	w.EvenFrame = false
	w.HandleBall(0)
	if w.Dinahs[0].Frame != 0 {
		t.Errorf("Frame = %d one frame later, want the round cel 0", w.Dinahs[0].Frame)
	}

	// The last bounce of a dying ball leaves it round: Frame is only written when Moving
	// survived the bounce, and the preceding airborne frame had already set it to 0.
	w = inFlight(t, Ball, func(d *Dynamic) {
		d.Position = 300
		d.Dest = Rect{Top: 267, Left: 200, Bottom: 299, Right: 232}
		d.VVel, d.Count, d.Frame = 1, -8, 0
		d.Active = false
	})
	w.EvenFrame = true
	w.HandleBall(0)
	if w.Dinahs[0].Moving {
		t.Fatalf("an impact speed of 1 truncates to 0, so this bounce has to stop the ball")
	}
	if w.Dinahs[0].Frame != 0 {
		t.Errorf("Frame = %d on the last bounce, want 0 -- a resting ball is round",
			w.Dinahs[0].Frame)
	}
}

// ---------------------------------------------------------------------------
// The drip and the fish
// ---------------------------------------------------------------------------

// TestDripSwellNeedsExactTimerValues: the idle arm is a chain of `==` tests, not ranges, so a
// drip whose reload period is shorter than 6 frames skips the cels it steps over. There is no
// clamping.
func TestDripSwellNeedsExactTimerValues(t *testing.T) {
	w := registerOne(t, Drip, typicalObject(Drip), testWhere)
	w.Dinahs[0].Moving = false
	w.Dinahs[0].Active = true
	w.Dinahs[0].Timer = 8
	w.Dinahs[0].Frame = 3

	// Timer walks 7, 6, 5, 4, 3, 2, 1, 0 -- cels at 6, 4 and 2, and the launch at 0.
	want := []int16{3, 0, 0, 1, 1, 2, 2, 4}
	for i, wantFrame := range want {
		w.HandleDrip(0)
		if got := w.Dinahs[0].Frame; got != wantFrame {
			t.Errorf("after %d idle frames Timer = %d and Frame = %d, want %d",
				i+1, w.Dinahs[0].Timer, got, wantFrame)
		}
	}
	if !w.Dinahs[0].Moving {
		t.Errorf("Moving = false after the timer reached 0; the drop never let go")
	}

	// A three-frame period never sees 6 or 4, so the first two cels of the swell are never
	// drawn.
	w = registerOne(t, Drip, typicalObject(Drip), testWhere)
	w.Dinahs[0].Moving = false
	w.Dinahs[0].Active = true
	w.Dinahs[0].Timer = 3
	w.Dinahs[0].Frame = 3
	seen := map[int16]bool{}
	for i := 0; i < 3; i++ {
		w.HandleDrip(0)
		seen[w.Dinahs[0].Frame] = true
	}
	if seen[0] || seen[1] {
		t.Errorf("cels seen = %v; a three-frame period must skip cels 0 and 1", seen)
	}
	if !w.Dinahs[0].Moving {
		t.Errorf("a three-frame period never launched")
	}
}

// TestDripFlipsTwoCelsOnEvenFrames: `Frame = 9 - Frame` is a two-cel flip, not arithmetic on
// an index -- 4 becomes 5 and 5 becomes 4.
func TestDripFlipsTwoCelsOnEvenFrames(t *testing.T) {
	w := inFlight(t, Drip, func(d *Dynamic) {
		d.Dest = Rect{Top: 100, Left: 200, Bottom: 112, Right: 208}
		d.VVel, d.Position, d.Frame = 3, 300, 4
	})

	want := []int16{5, 5, 4, 4, 5, 5}
	for i, wantFrame := range want {
		w.EvenFrame = i%2 == 0
		w.HandleDrip(0)
		if got := w.Dinahs[0].Frame; got != wantFrame {
			t.Fatalf("frame %d (even %v): Frame = %d, want %d", i, i%2 == 0, got, wantFrame)
		}
	}
}

// TestDripSplashdownHangsTheNextOne: the drop is put back on the ceiling at HVel -- the Y the
// registration recorded -- with a 12-pixel box, and the reload period restarts.
func TestDripSplashdownHangsTheNextOne(t *testing.T) {
	w := inFlight(t, Drip, func(d *Dynamic) {
		d.Dest = Rect{Top: 290, Left: 200, Bottom: 298, Right: 208}
		d.Whole = Rect{Top: 280, Left: 200, Bottom: 298, Right: 208}
		d.VVel, d.Position, d.HVel, d.Count = 3, 300, 111, 33
	})
	var snd soundLog
	snd.install(w)
	clearRects(w)

	w.EvenFrame = false
	w.HandleDrip(0)

	d := w.Dinahs[0]
	if d.Dest.Top != 111 || d.Dest.Bottom != 111+12 {
		t.Errorf("Dest = %+v, want Top = HVel = 111 and Bottom = Top + 12", d.Dest)
	}
	if d.VVel != 0 || d.Timer != 33 || d.Frame != 3 || d.Moving {
		t.Errorf("(VVel, Timer, Frame, Moving) = (%d, %d, %d, %v), want (0, 33, 3, false)",
			d.VVel, d.Timer, d.Frame, d.Moving)
	}
	snd.is(t, DropSound)
	if work, back := rectCounts(w); work != 1 || back != 0 {
		t.Errorf("rects (work %d, back %d), want (1, 0) -- the fall's trail is erased",
			work, back)
	}
}

// TestDripIdleArmLeavesWholeStale pins a transcribed inefficiency, so that a later efficiency
// pass has to remove it deliberately.
//
// The idle arm never touches Whole, so from the frame after a splashdown until the next launch
// it still holds the union of the last fall: the full column from ceiling to floor. RenderDrip
// is not gated on Moving, so it registers that stale column as a work rect on every single
// resting frame. Harmless -- the work map there is a clean copy of the background -- but it is
// a large blit and two of the 47 rect slots, held for the whole reload period. See
// docs/IMPROVEMENTS.md.
func TestDripIdleArmLeavesWholeStale(t *testing.T) {
	w := inFlight(t, Drip, func(d *Dynamic) {
		d.Dest = Rect{Top: 290, Left: 200, Bottom: 298, Right: 208}
		d.Whole = Rect{Top: 100, Left: 200, Bottom: 298, Right: 208}
		d.VVel, d.Position, d.HVel, d.Count = 3, 300, 111, 33
		d.Active = true
	})
	w.HandleDrip(0) // splashdown

	column := w.Dinahs[0].Whole
	if column.Tall() < 100 {
		t.Fatalf("Whole = %+v after splashdown; the fixture did not build a tall column", column)
	}
	for i := 0; i < 10; i++ {
		w.HandleDrip(0)
		if w.Dinahs[0].Whole != column {
			t.Fatalf("idle frame %d rewrote Whole to %+v; the idle arm must leave it stale",
				i+1, w.Dinahs[0].Whole)
		}
	}
	if w.Dinahs[0].Whole.Tall() <= w.Dinahs[0].Dest.Tall() {
		t.Errorf("Whole %+v is no taller than Dest %+v; the stale column is the point",
			w.Dinahs[0].Whole, w.Dinahs[0].Dest)
	}
}

// TestFishAnimatesOnlyOnTheWayDown: `VVel >= 0 && Frame < 7` walks cels 4..7 as the fish
// falls, so the rise holds cel 4 -- one pose going up, four coming down. It stops at 7 rather
// than wrapping, so the last cel is held until splashdown.
func TestFishAnimatesOnlyOnTheWayDown(t *testing.T) {
	w := inFlight(t, Fish, func(d *Dynamic) {
		d.Dest = Rect{Top: 200, Left: 200, Bottom: 216, Right: 216}
		d.VVel, d.Position, d.Frame, d.Count = -9, 290, 4, -9
	})

	// Rising: VVel is negative, so the cel is held.
	for frame := 0; w.Dinahs[0].VVel < 0; frame++ {
		w.EvenFrame = frame%2 == 0
		w.HandleFish(0)
		if !w.Dinahs[0].Moving {
			t.Fatalf("frame %d: splashed down while still rising", frame)
		}
		if w.Dinahs[0].Frame != 4 {
			t.Fatalf("Frame = %d while rising (VVel %d), want the single rise cel 4",
				w.Dinahs[0].Frame, w.Dinahs[0].VVel)
		}
	}

	// Falling: 5, 6, 7 and then held at 7.
	var got []int16
	for frame := 0; frame < 6 && w.Dinahs[0].Moving; frame++ {
		w.EvenFrame = frame%2 == 0
		w.HandleFish(0)
		got = append(got, w.Dinahs[0].Frame)
	}
	if len(got) < 4 {
		t.Fatalf("only %d falling frames (%v); the fixture splashed down too early", len(got), got)
	}
	for i, f := range got {
		want := int16(5 + i)
		if want > 7 {
			want = 7
		}
		if f != want {
			t.Fatalf("falling cels = %v, want 5, 6, 7 then held at 7", got)
		}
	}
}

// TestFishSplashdownPlaysTwoSounds: DropSound then FishInSound, one after the other with
// nothing between them.
//
// Not a mistake to tidy -- which one the player hears is PlayPrioritySound's decision, and 1.6
// will make that call. Transcribed as two calls, and the order is part of it.
func TestFishSplashdownPlaysTwoSounds(t *testing.T) {
	w := inFlight(t, Fish, func(d *Dynamic) {
		d.Dest = Rect{Top: 280, Left: 200, Bottom: 296, Right: 216}
		d.Whole = Rect{Top: 270, Left: 200, Bottom: 296, Right: 216}
		d.VVel, d.Position, d.Count, d.HVel = 6, 300, -9, 33
	})
	var snd soundLog
	snd.install(w)
	clearRects(w)

	w.EvenFrame = false
	w.HandleFish(0)

	snd.is(t, DropSound, FishInSound)

	d := w.Dinahs[0]
	if d.Dest.Bottom != 300 || d.Dest.Top != 300-16 {
		t.Errorf("Dest = %+v, want Bottom = Position = 300 and Top = Bottom - 16", d.Dest)
	}
	if d.Whole.Top != d.Dest.Top-2 {
		t.Errorf("Whole.Top = %d, want Dest.Top - 2 = %d", d.Whole.Top, d.Dest.Top-2)
	}
	if d.VVel != -9 || d.Timer != 33 || d.Frame != 0 || d.Moving {
		t.Errorf("(VVel, Timer, Frame, Moving) = (%d, %d, %d, %v), want (Count -9, HVel 33, 0, false)",
			d.VVel, d.Timer, d.Frame, d.Moving)
	}
	if work, back := rectCounts(w); work != 1 || back != 0 {
		t.Errorf("rects (work %d, back %d), want (1, 0)", work, back)
	}
}

// TestFishBobsWhileSwitchedOff is the closest thing in Dynamics2.c to a visible bug, and it is
// reachable in the shipped game.
//
// The idle bob runs *outside* the Active test: `Whole = Dest` and the `Timer & 3` animation are
// unconditional, and only the countdown is gated. So a switched-off fish still animates using a
// frozen timer -- and if that frozen timer happens to be 3 modulo 4, the bob condition is true
// on every frame and the fish twitches forever, one pixel at a time, in place.
//
// The cel cycle is 1, 2, 3, 0 and the moves are down, down, up, up, so it is a two-pixel twitch
// that returns to where it started every four frames. Asserting the return is what shows it is
// a twitch rather than a drift.
func TestFishBobsWhileSwitchedOff(t *testing.T) {
	w := registerOne(t, Fish, typicalObject(Fish), testWhere)
	w.Dinahs[0].Moving = false
	w.Dinahs[0].Active = false
	w.Dinahs[0].Timer = 3 // 3 & 3 == 3, and nothing will ever decrement it
	w.Dinahs[0].Frame = 0
	start := w.Dinahs[0].Dest

	var offsets []int16
	for i := 0; i < 8; i++ {
		w.HandleFish(0)
		if w.Dinahs[0].Timer != 3 {
			t.Fatalf("frame %d: Timer = %d, want a frozen 3", i+1, w.Dinahs[0].Timer)
		}
		offsets = append(offsets, w.Dinahs[0].Dest.Top-start.Top)
	}

	want := []int16{1, 2, 1, 0, 1, 2, 1, 0}
	for i := range want {
		if offsets[i] != want[i] {
			t.Fatalf("Dest.Top offsets = %v, want %v (down, down, up, up, forever)", offsets, want)
		}
	}
	if w.Dinahs[0].Dest != start {
		t.Errorf("Dest = %+v after eight frames, want the original %+v -- the twitch drifted",
			w.Dinahs[0].Dest, start)
	}

	// The bob keeps its own union: a one-pixel move always has a two-pixel rect behind it.
	if w.Dinahs[0].Whole == w.Dinahs[0].Dest {
		t.Errorf("Whole == Dest = %+v; the bob's union does not cover the move",
			w.Dinahs[0].Dest)
	}
}

// TestFishBobIsSilentAtOtherTimerValues: the frozen-timer twitch needs Timer == 3 mod 4, so a
// fish switched off at any other moment simply sits still. That is what makes the bug
// intermittent rather than universal, and it is why it survived 1994.
func TestFishBobIsSilentAtOtherTimerValues(t *testing.T) {
	for _, timer := range []int16{0, 1, 2, 4, 5, 6} {
		w := registerOne(t, Fish, typicalObject(Fish), testWhere)
		w.Dinahs[0].Moving = false
		w.Dinahs[0].Active = false
		w.Dinahs[0].Timer = timer
		start := w.Dinahs[0].Dest

		for i := 0; i < 8; i++ {
			w.HandleFish(0)
		}
		if w.Dinahs[0].Dest != start {
			t.Errorf("Timer %d: the fish moved to %+v; only 3 mod 4 twitches",
				timer, w.Dinahs[0].Dest)
		}
	}
}

// ---------------------------------------------------------------------------
// The seven renderers (Dynamics.c:112-286)
// ---------------------------------------------------------------------------

// clearOfTheSlot is a toaster clip line far below any test rect, so that RenderToast's
// `vClip = Dest.Bottom - HVel` is negative and the full cel is drawn. HVel is the slot line for
// a toaster and nothing else, so every other renderer ignores it.
const clearOfTheSlot int16 = 1000

// renderCases is one row per renderer: the type, the call, and whether it is gated on Moving.
var renderCases = []struct {
	name  string
	what  int16
	fn    func(*World, int16)
	gated bool
	note  string
}{
	{"toast", Toaster, (*World).RenderToast, true, ""},
	{"balloon", Balloon, (*World).RenderBalloon, true, ""},
	{"copter", CopterLf, (*World).RenderCopter, true, ""},
	{"dart", DartRt, (*World).RenderDart, true, ""},
	{"ball", Ball, (*World).RenderBall, false,
		"a ball that ran down its bounces sits on the floor forever, is still lethal, and is " +
			"not in the static background"},
	{"drip", Drip, (*World).RenderDrip, false,
		"cel 3, the hanging drop, has to be drawn every resting frame"},
	{"fish", Fish, (*World).RenderFish, false,
		"a resting fish draws the four idle ripple cels"},
}

// TestRenderersRegisterDestOnBackAndWholeOnWork is the six-line shape, and the two rects are
// not interchangeable.
//
// Dest -- where the sprite is now -- goes on the back list, so exactly that much of the work
// map is restored from the clean background next frame. Whole -- the union of now and last
// frame -- goes on the work list, so the screen copy covers the trail. Getting them the wrong
// way round leaves either a permanent smear or a flicker.
func TestRenderersRegisterDestOnBackAndWholeOnWork(t *testing.T) {
	for _, tc := range renderCases {
		t.Run(tc.name, func(t *testing.T) {
			w := registerOne(t, tc.what, typicalObject(tc.what), testWhere)
			d := &w.Dinahs[0]
			d.Moving = true
			d.Frame = 0
			d.Dest = Rect{Top: 100, Left: 200, Bottom: 130, Right: 230}
			d.Whole = Rect{Top: 96, Left: 198, Bottom: 130, Right: 230}
			d.HVel = clearOfTheSlot
			clearRects(w)

			tc.fn(w, 0)

			work, back := rectCounts(w)
			if work != 1 || back != 1 {
				t.Fatalf("rects (work %d, back %d), want (1, 1)", work, back)
			}
			oh, ov := w.R.V.OriginH, w.R.V.OriginV
			if got, want := w.Back2Work[0], render.Offset(d.Dest, oh, ov); got != want {
				t.Errorf("back rect = %+v, want the offset Dest %+v", got, want)
			}
			if got, want := w.Work2Main[0], render.Offset(d.Whole, oh, ov); got != want {
				t.Errorf("work rect = %+v, want the offset Whole %+v", got, want)
			}
		})
	}
}

// TestRenderGatesOnMovingExceptBallDripAndFish pins which three of the seven have no gate.
//
// The gate on the other four is what makes their static room draw sufficient: a waiting
// balloon, copter or dart is painted into the background by the composition and the renderer
// leaves it alone until it launches. The three ungated ones are ungated for three different
// reasons -- see the notes in renderCases -- which is why this cannot be reduced to a rule.
func TestRenderGatesOnMovingExceptBallDripAndFish(t *testing.T) {
	ungated := 0
	for _, tc := range renderCases {
		if !tc.gated {
			ungated++
		}
		t.Run(tc.name, func(t *testing.T) {
			w := registerOne(t, tc.what, typicalObject(tc.what), testWhere)
			w.Dinahs[0].Moving = false
			w.Dinahs[0].Frame = 0
			w.Dinahs[0].Dest = Rect{Top: 100, Left: 200, Bottom: 130, Right: 230}
			w.Dinahs[0].Whole = w.Dinahs[0].Dest
			w.Dinahs[0].HVel = clearOfTheSlot
			clearRects(w)

			tc.fn(w, 0)

			work, back := rectCounts(w)
			if tc.gated {
				if work != 0 || back != 0 {
					t.Errorf("a resting %s registered (work %d, back %d), want nothing",
						tc.name, work, back)
				}
			} else if work != 1 || back != 1 {
				t.Errorf("a resting %s registered (work %d, back %d), want (1, 1) -- %s",
					tc.name, work, back, tc.note)
			}
		})
	}
	if ungated != 3 {
		t.Errorf("%d ungated renderers, want 3 (ball, drip, fish)", ungated)
	}
}

// TestRenderToastClipsToTheSlot is the one renderer written longhand.
//
// **HVel is not a velocity for a toaster; it is the clip line** -- the Y of the top of the
// slot, recorded at registration. `vClip = Dest.Bottom - HVel` is how far the bread's bottom
// has sunk past that line, and while it is positive both the source and the destination lose
// that many rows from the bottom, so the slice appears to emerge from the slot instead of
// floating in front of it.
//
// The clipped destination bottom therefore lands exactly on the slot line, which is a cleaner
// statement of the same arithmetic and is what this asserts. The clip is applied to src and
// dest and never to Whole, so the work rect covers the unclipped travel -- that is what erases
// the bread's trail once it is above the slot.
func TestRenderToastClipsToTheSlot(t *testing.T) {
	toast := func(t *testing.T, dest, whole Rect, hVel int16) *World {
		t.Helper()
		w := registerOne(t, Toaster, typicalObject(Toaster), testWhere)
		d := &w.Dinahs[0]
		d.Moving = true
		d.Frame = 0
		d.Dest, d.Whole, d.HVel = dest, whole, hVel
		clearRects(w)
		w.RenderToast(0)
		return w
	}

	t.Run("still inside the slot", func(t *testing.T) {
		dest := Rect{Top: 100, Left: 200, Bottom: 120, Right: 216}
		whole := Rect{Top: 100, Left: 200, Bottom: 130, Right: 216}
		w := toast(t, dest, whole, 110) // the slot line, ten pixels above the bread's bottom
		oh, ov := w.R.V.OriginH, w.R.V.OriginV

		if got, want := w.Back2Work[0].Bottom, 110+ov; got != want {
			t.Errorf("clipped back rect bottom = %d, want the slot line + origin = %d", got, want)
		}
		if got, want := w.Work2Main[0], render.Offset(whole, oh, ov); got != want {
			t.Errorf("work rect = %+v, want the *unclipped* Whole %+v", got, want)
		}
	})

	t.Run("clear of the slot", func(t *testing.T) {
		dest := Rect{Top: 60, Left: 200, Bottom: 80, Right: 216}
		whole := Rect{Top: 60, Left: 200, Bottom: 100, Right: 216}
		w := toast(t, dest, whole, 110) // vClip is negative now
		oh, ov := w.R.V.OriginH, w.R.V.OriginV

		if got, want := w.Back2Work[0], render.Offset(dest, oh, ov); got != want {
			t.Errorf("back rect = %+v, want the unclipped Dest %+v", got, want)
		}
	})

	t.Run("the clip can consume the whole cel", func(t *testing.T) {
		// On the frame the bread is launched Dest is still entirely inside the slot, so
		// dest.Bottom drops below dest.Top and the rect inverts. QuickDraw draws nothing for
		// an inverted rect and Surface.Copy returns early on any non-positive extent, so the
		// two agree -- but the rect is still registered on both lists, exactly as the C
		// registers it, because CopyRectsQD has to see the same rects.
		dest := Rect{Top: 100, Left: 200, Bottom: 120, Right: 216}
		w := toast(t, dest, dest, 50) // vClip = 70, far more than the cel is tall

		if work, back := rectCounts(w); work != 1 || back != 1 {
			t.Errorf("rects (work %d, back %d), want (1, 1) even for an inverted rect", work, back)
		}
		if got := w.Back2Work[0]; got.Bottom >= got.Top {
			t.Errorf("back rect = %+v, want it inverted", got)
		}
	})
}

// TestRenderFishSwitchesBlitters is the only renderer that changes transfer mode, and the
// strip it draws from is what makes the switch necessary rather than arbitrary.
//
// The eight cels split cleanly in half. Cels 0-3, the idle ripple the resting animation
// cycles through, are **fully opaque** 16x16 tiles -- they are the water, not something
// sitting in it. Cels 4-7, the leap, are a fish silhouette with about half their pixels
// clear. So SrcCopy for a resting fish paints the water tile it is meant to paint, and Masked
// for a leaping one lets the room show through around the fish. Swap them and both halves
// break: a masked resting fish leaves the room's background showing where the water should be,
// and an unmasked leaping one drags a 16x16 block of the strip's own backdrop across the room.
//
// Which is also why the switch cannot be tested with the cel the handler would pair with it.
// Masking a fully opaque cel is a no-op, so a resting fish draws identical pixels either way
// and the two modes are indistinguishable there. The mode is isolated by holding Frame on a
// leap cel and toggling only Moving -- a combination the handler never produces, which is the
// point: it is the renderer's decision under test, not the handler's cel choice.
//
// Asserted on pixels rather than on a recorded mode, because the mode is a local: the test
// predicts both blits from the strip's own mask and checks which prediction the work map
// matches. It has to be pixels for a second reason -- copy1to1 takes the same memmove fast path
// for SrcCopy and for Masked with a mask-less source, so a test that only checked the argument
// could pass while drawing the wrong thing.
func TestRenderFishSwitchesBlitters(t *testing.T) {
	artDir := requireAssets(t, "art")
	w := newTestWorld(oneRoomHouse(house.ObjectIsEmpty), artDir, "")
	w.InitGarbageRects()
	w.R.NumLights = 3
	if slot := w.AddDynamicObject(Fish, testWhere, typicalObject(Fish),
		testRoom, testSlot, true); slot != 0 {
		t.Fatalf("AddDynamicObject(fish) = %d, want slot 0", slot)
	}

	art := w.R.A.Strip("fish")
	if art == nil {
		t.Skip("no fish strip in the extracted art")
	}
	if art.Mask == nil {
		t.Skip("the extracted fish strip carries no mask, so Masked and SrcCopy are the same " +
			"blit and this test could not tell them apart")
	}

	// The opacity split the mode switch is built around, pinned against the shipped art. This
	// is the reason the two branches exist, so it is asserted rather than assumed.
	for cel, src := range render.FishSrc {
		clear := clearPixels(art, src)
		if cel < 4 && clear != 0 {
			t.Errorf("idle cel %d has %d transparent pixels, want a solid water tile",
				cel, clear)
		}
		if cel >= 4 && clear == 0 {
			t.Errorf("leap cel %d is fully opaque, so Masked would have nothing to do", cel)
		}
	}

	// Cel 4 is the one a leaping fish holds all the way up, and is where the modes disagree.
	const leapCel = 4
	src := render.FishSrc[leapCel]
	d := &w.Dinahs[0]
	d.Frame = leapCel
	d.Dest = Rect{Top: 100, Left: 200, Bottom: 100 + src.Tall(), Right: 200 + src.Wide()}
	d.Whole = d.Dest
	dest := render.Offset(d.Dest, w.R.V.OriginH, w.R.V.OriginV)

	// A sentinel background, so that "left alone" is visible.
	w.R.Work.Fill(dest, render.Black8)
	before := workPixels(w, dest)
	wantMasked := blitPrediction(art, src, dest, before, render.Masked)
	wantSrcCopy := blitPrediction(art, src, dest, before, render.SrcCopy)
	if samePixels(wantMasked, wantSrcCopy) {
		t.Fatalf("the two modes predict the same pixels for leap cel %d, so this test cannot "+
			"tell them apart; either the cel lost its mask or the strip's backdrop is now "+
			"Black8", leapCel)
	}

	for _, tc := range []struct {
		name   string
		moving bool
		want   []uint8
	}{
		{"leaping is masked", true, wantMasked},
		{"held still is unmasked", false, wantSrcCopy},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w.R.Work.Fill(dest, render.Black8)
			w.Dinahs[0].Moving = tc.moving
			clearRects(w)

			w.RenderFish(0)

			if got := workPixels(w, dest); !samePixels(got, tc.want) {
				t.Errorf("the work map does not match the %q prediction", tc.name)
			}
			// Both branches register the same two rects, so the switch is purely a transfer
			// mode.
			if work, back := rectCounts(w); work != 1 || back != 1 {
				t.Errorf("rects (work %d, back %d), want (1, 1)", work, back)
			}
		})
	}
}

// TestRenderDynamicsHasNoDefault: the fan-out has seven arms covering nine of the seventeen
// types, and an empty `default` -- so the other eight, the seven appliances aside from the
// toaster plus the sparkle emitter, draw nothing and register nothing.
//
// Which is right: the appliances have their own handlers that blit and register directly
// (dynamics_appliances.go), and a sparkle is drawn by RenderSparkles from a different table. A
// `default` that fell through to renderDinah would give every appliance a second, wrongly
// offset blit.
func TestRenderDynamicsHasNoDefault(t *testing.T) {
	rendered := map[int16]bool{
		Toaster: true, Balloon: true, CopterLf: true, CopterRt: true,
		DartLf: true, DartRt: true, Ball: true, Drip: true, Fish: true,
	}
	if len(rendered) != 9 {
		t.Fatalf("%d rendered types listed, want the 9 the seven arms cover", len(rendered))
	}

	all := allRegistrableTypes()
	for _, what := range all {
		w := registerOne(t, what, typicalObject(what), testWhere)
		w.Dinahs[0].Moving = true
		w.Dinahs[0].Frame = 0
		w.Dinahs[0].HVel = clearOfTheSlot
		clearRects(w)

		w.RenderDynamics()

		work, back := rectCounts(w)
		if rendered[what] {
			if work != 1 || back != 1 {
				t.Errorf("%#x: rects (work %d, back %d), want (1, 1)", what, work, back)
			}
		} else if work != 0 || back != 0 {
			t.Errorf("%#x: rects (work %d, back %d), want nothing -- RenderDynamics has no "+
				"default arm", what, work, back)
		}
	}
	if want := len(all) - len(rendered); want != 8 {
		t.Errorf("%d registrable types have no renderer, want 8", want)
	}
}

// ---------------------------------------------------------------------------
// Pixel helpers for TestRenderFishSwitchesBlitters
// ---------------------------------------------------------------------------

// workPixels reads a rect out of the work map, row-major.
func workPixels(w *World, r Rect) []uint8 {
	out := make([]uint8, 0, int(r.Wide())*int(r.Tall()))
	for y := int(r.Top); y < int(r.Bottom); y++ {
		for x := int(r.Left); x < int(r.Right); x++ {
			out = append(out, w.R.Work.Pix[y*w.R.Work.W+x])
		}
	}
	return out
}

// clearPixels counts the transparent pixels of one cel: how much of it Masked would skip.
func clearPixels(art *render.Surface, src Rect) int {
	if art.Mask == nil {
		return 0
	}
	n := 0
	for y := int(src.Top); y < int(src.Bottom); y++ {
		for x := int(src.Left); x < int(src.Right); x++ {
			if y < 0 || y >= art.H || x < 0 || x >= art.W {
				continue
			}
			if art.Mask[y*art.W+x] == 0 {
				n++
			}
		}
	}
	return n
}

// blitPrediction is what the dest rect would hold after copying src from art in the given
// mode, computed from the rect's contents beforehand. Same-size rects only, which is every
// blit here.
func blitPrediction(art *render.Surface, src, dst Rect, before []uint8, mode render.CopyMode) []uint8 {
	out := append([]uint8(nil), before...)
	width := int(dst.Wide())
	for y := 0; y < int(dst.Tall()); y++ {
		for x := 0; x < width; x++ {
			sx, sy := int(src.Left)+x, int(src.Top)+y
			if sx < 0 || sx >= art.W || sy < 0 || sy >= art.H {
				continue
			}
			if mode == render.Masked && art.Mask != nil && art.Mask[sy*art.W+sx] == 0 {
				continue
			}
			out[y*width+x] = art.Pix[sy*art.W+sx]
		}
	}
	return out
}

func samePixels(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
