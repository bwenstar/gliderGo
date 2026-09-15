package game

// The poke path: trip.go's fourteen Toggle*, its eight Trigger* and its one
// UpdateOutletsLighting, plus triggers.go's arm/tick/fire machinery. Since 1.5d the other way
// in is HandleSwitches, and switches_test.go owns that side.
//
// Two shapes of test here and the difference is worth naming, because it decides what a
// failure means.
//
// The Toggle*/Trigger* tests call the pokes **directly**, with a dinah registered by
// AddDynamicObject and nothing else. That is deliberate: each of the twenty-two is one or
// two lines and what has to be pinned is which fields those lines write and which they
// leave alone. A test that reached them through a composed house would prove the same
// thing more slowly and would fail for a dozen unrelated reasons.
//
// The triggers.go tests go the other way and drive FireTrigger through a hand-built
// two-entry master graph, because the dispatch *is* the subject: thirteen arms, one
// remote arm, and a `default` whose emptiness is a design decision (see
// TestSevenTypesAreNotTriggerable, which trip.go's header points at by name).
//
// TriggerSwitch is tested here for its *bound* -- which table it indexes -- and not for its
// effect, even though HandleSwitches has been real since 1.5d. The bound is the half that
// matters at this seam, because it is the thing hazard H2 is about: `who` is a hotSpots index
// and not a dinahs slot, and every other Trigger* takes the other kind. The effects belong to
// switches_test.go, which drives HandleSwitches directly.

import (
	"testing"

	"glidergo/internal/house"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// poisonTimer is a value no Toggle* ever writes, parked in Timer before each toggle so
// that "did not write Timer" is an assertion rather than an absence.
const poisonTimer int16 = 999

// deviations installs the Diagnostics hook and returns the growing list of refusals.
//
// The hook rather than Diag.Seen, because Seen is distinct-by-kind and samples only the
// first sixteen: two refusals of the same kind are one entry there, and several of these
// tests count refusals.
func deviations(w *World) *[]Deviation {
	var seen []Deviation
	w.Diag.On = func(d Deviation) { seen = append(seen, d) }
	return &seen
}

// linkedPair is the smallest master graph FireTrigger will walk: a trigger plate at index
// 0 whose LocalLink names a target at index 1.
//
// The plate carries a real switchType payload so that ArmTrigger's big-endian delay read
// has something to read. The target's DynaNum is the caller's, because which table it
// indexes depends on the target's own type -- a dinahs slot for twelve of the arms, a
// hotSpots index for the six switch arms. See MasterObject.DynaNum.
func linkedPair(targetWhat, dynaNum, delay int16) []MasterObject {
	plate := house.Object{What: Trigger}
	plate.SetSwitch(house.Switch{Delay: delay, Where: 0, Who: 1})
	return []MasterObject{
		{RoomNum: 0, ObjectNum: 0, RoomLink: 0, ObjectLink: 1, LocalLink: 1,
			HotNum: -1, DynaNum: -1, TheObject: plate},
		{RoomNum: 0, ObjectNum: 1, RoomLink: -1, ObjectLink: -1, LocalLink: -1,
			HotNum: -1, DynaNum: dynaNum, TheObject: house.Object{What: targetWhat}},
	}
}

// fireAt installs linkedPair's graph, fills trigger slot 0 as ArmTrigger would have, and
// fires it immediately -- skipping the countdown, which TestTriggerTimingAtThreeDelays
// covers separately.
func fireAt(w *World, targetWhat, dynaNum int16) {
	w.R.Master = linkedPair(targetWhat, dynaNum, 0)
	w.Triggers[0] = TriggerSlot{Room: 0, Object: 1, Index: 0, What: targetWhat, Armed: true}
	w.FireTrigger(0)
}

// greaseAt puts a spillable grease jar in room 0's object slot 1, which is where
// linkedPair's target lives in house coordinates. State != 0 is what makes
// SetObjectState report a change, and therefore what makes the grease arm reach
// SpillGrease.
func greaseAt(w *World, what int16) {
	o := house.Object{What: what}
	o.SetBonus(house.Bonus{State: 1})
	w.Room(0).Objects[1] = o
}

// ---------------------------------------------------------------------------
// The fourteen Toggle* (Trip.c:20-142)
// ---------------------------------------------------------------------------

// toggleCase is one of the fourteen: the type whose dinah it is dispatched for, the
// function, and Timer after each of the first two calls.
//
// The first call always switches *off*, because AddDynamicObject sets Active = isOn and
// every fixture here registers with isOn true. So timerOff is the first observation and
// timerOn the second, which reads backwards and is the honest order.
type toggleCase struct {
	name              string
	what              int16
	fn                func(*World, int16)
	timerOff, timerOn int16
	// zeroFirst is ToggleStereos alone: it refuses while Timer is non-zero, so it
	// cannot be handed poisonTimer. See TestToggleStereosRefusesWhileTimerRuns.
	zeroFirst bool
	note      string
}

// theFourteenToggles is HandleSwitches' dispatch table (Interactions.c:1090-1148), which
// is the only caller of all fourteen in the shipped game.
//
// The `what` column matters: each toggle is exercised against a dinah registered by the
// type it is dispatched for, so that the fields it leaves alone are that type's real
// post-registration values rather than zeroes. None of the fourteen actually looks at
// Type -- they index and write -- and the invariant test below is what proves that.
var theFourteenToggles = []toggleCase{
	{"toaster", Toaster, (*World).ToggleToaster, poisonTimer, poisonTimer,
		false, "Timer is the reload period; a switch must not reset it"},
	{"macPlus", MacPlus, (*World).ToggleMacPlus, 10, 40,
		false, "the only asymmetric toggle: four times as long to wake as to sleep"},
	{"tv", TV, (*World).ToggleTV, 4, 4, false, ""},
	{"coffee", Coffee, (*World).ToggleCoffee, 4, 4,
		false, "unconditional in both directions, so a mid-cycle switch restarts the cycle"},
	{"outlet", Outlet, (*World).ToggleOutlet, poisonTimer, poisonTimer,
		false, "Timer is the reload countdown; switching off freezes it where it stands"},
	{"vcr", VCR, (*World).ToggleVCR, 4, 4, false, ""},
	{"stereo", Stereo, (*World).ToggleStereos, 4, 4,
		true, "the only toggle that can refuse; needs Timer == 0 to act at all"},
	{"microwave", Microwave, (*World).ToggleMicrowave, 4, 4, false, ""},
	{"balloon", Balloon, (*World).ToggleBalloon, poisonTimer, poisonTimer,
		false, "a balloon already in flight keeps flying; Active gates the idle arm only"},
	{"copter", CopterLf, (*World).ToggleCopter, poisonTimer, poisonTimer, false, ""},
	{"dart", DartRt, (*World).ToggleDart, poisonTimer, poisonTimer, false, ""},
	{"ball", Ball, (*World).ToggleBall, poisonTimer, poisonTimer,
		false, "switching a ball off does not make it safe -- HandleBall collides above its Moving test"},
	{"drip", Drip, (*World).ToggleDrip, poisonTimer, poisonTimer, false, ""},
	{"fish", Fish, (*World).ToggleFish, poisonTimer, poisonTimer, false, ""},
}

// TestTogglesFlipActiveAndNothingElse is the fourteen-way table: each toggle inverts
// Active, writes Timer if and only if the table says so, and touches no other field.
//
// The third clause is the one worth having. Nine of the fourteen are a single line in the
// C and it is easy to believe a port of one of them; what is hard to see by reading is
// that none of the fourteen quietly resets Frame, or Moving, or a rect -- which several
// of the *handlers* do, and which a plausible-looking toggle could be written to do.
func TestTogglesFlipActiveAndNothingElse(t *testing.T) {
	for _, tc := range theFourteenToggles {
		t.Run(tc.name, func(t *testing.T) {
			w := registerOne(t, tc.what, typicalObject(tc.what), testWhere)
			if !w.Dinahs[0].Active {
				t.Fatalf("registered with isOn true and Active is false; the fixture is wrong")
			}

			for call, want := range []int16{tc.timerOff, tc.timerOn} {
				if tc.zeroFirst {
					w.Dinahs[0].Timer = 0
				} else {
					w.Dinahs[0].Timer = poisonTimer
				}
				wantActive := call == 1

				before := w.Dinahs[0]
				tc.fn(w, 0)
				after := w.Dinahs[0]

				if after.Active != wantActive {
					t.Errorf("call %d: Active = %v, want %v", call+1, after.Active, wantActive)
				}
				if after.Timer != want {
					t.Errorf("call %d: Timer = %d, want %d (%s)", call+1, after.Timer, want, tc.note)
				}

				// Everything except the two fields the toggle is allowed to write.
				before.Active, before.Timer = after.Active, after.Timer
				if before != after {
					t.Errorf("call %d: a toggle wrote a field other than Active and Timer\n"+
						"before %+v\nafter  %+v", call+1, before, after)
				}
			}
		})
	}
}

// TestToggleMacPlusIsTheOnlyAsymmetricToggle reads the table rather than the world.
//
// The claim is about the *set*: exactly one of the fourteen writes a different Timer going
// on than going off. Trip.c is fourteen near-identical functions and the Mac's 40/10 is
// the single place the author varied, so a second asymmetric entry appearing in the table
// is either a discovery worth arguing about or a typo, and both want stopping here.
func TestToggleMacPlusIsTheOnlyAsymmetricToggle(t *testing.T) {
	var asymmetric []string
	for _, tc := range theFourteenToggles {
		if tc.timerOff != tc.timerOn {
			asymmetric = append(asymmetric, tc.name)
		}
	}
	if len(asymmetric) != 1 || asymmetric[0] != "macPlus" {
		t.Errorf("asymmetric toggles = %v, want [macPlus] alone", asymmetric)
	}
	if theFourteenToggles[1].timerOn != 40 || theFourteenToggles[1].timerOff != 10 {
		t.Errorf("macPlus timers = on %d / off %d, want on 40 / off 10",
			theFourteenToggles[1].timerOn, theFourteenToggles[1].timerOff)
	}
}

// TestTheFourteenTogglesAreFourteen pins the count and the absence of duplicates, so that
// the table above cannot drift out of step with Trip.c by a copy-paste.
func TestTheFourteenTogglesAreFourteen(t *testing.T) {
	if len(theFourteenToggles) != 14 {
		t.Fatalf("%d toggles in the table, want 14", len(theFourteenToggles))
	}
	seen := map[string]bool{}
	for _, tc := range theFourteenToggles {
		if seen[tc.name] {
			t.Errorf("duplicate toggle %q", tc.name)
		}
		seen[tc.name] = true
	}
}

// TestToggleStereosRefusesWhileTimerRuns is the one Toggle* with a precondition.
//
// HandleStereo holds Timer non-zero for the whole four-frame fade, so this is the
// mechanism that gives a player mashing a stereo switch one toggle per four frames instead
// of one per press. Every other appliance honours the switch mid-cycle; asserting the
// refusal *and* that nothing at all moved is what separates "ignored" from "half applied".
func TestToggleStereosRefusesWhileTimerRuns(t *testing.T) {
	w := registerOne(t, Stereo, typicalObject(Stereo), testWhere)

	w.Dinahs[0].Timer = 3
	before := w.Dinahs[0]
	w.ToggleStereos(0)
	if w.Dinahs[0] != before {
		t.Errorf("a stereo with Timer 3 was touched\nbefore %+v\nafter  %+v", before, w.Dinahs[0])
	}

	// And it acts the instant the fade is over.
	w.Dinahs[0].Timer = 0
	w.ToggleStereos(0)
	if w.Dinahs[0].Active {
		t.Errorf("Active = true after a toggle at Timer 0, want false")
	}
	if w.Dinahs[0].Timer != 4 {
		t.Errorf("Timer = %d after a toggle at Timer 0, want 4", w.Dinahs[0].Timer)
	}
}

// TestTogglesRefuseUnregisteredObjects drives all fourteen through World.dinah's guard.
//
// Both bad indices are reachable in the shipped game, which is why this is not a
// paranoia test: DynaNum is -1 for any object that never registered -- one past the
// eighteen-slot cap, or one composed with redraw = true -- and a switch wired to it hands
// that -1 straight here. The C reads dinahs[-1]. The upper bound is the same story from
// the other end: NumDynamics, not MaxDynamicObs, so slot 1 of an 18-slot table with one
// live object is already out of range.
func TestTogglesRefuseUnregisteredObjects(t *testing.T) {
	for _, tc := range theFourteenToggles {
		t.Run(tc.name, func(t *testing.T) {
			for _, index := range []int16{-1, 1, MaxDynamicObs} {
				w := registerOne(t, tc.what, typicalObject(tc.what), testWhere)
				seen := deviations(w)
				before := w.Dinahs

				tc.fn(w, index)

				if w.Dinahs != before {
					t.Errorf("index %d: the dinahs table was written", index)
				}
				if len(*seen) != 1 {
					t.Fatalf("index %d: %d deviations, want 1", index, len(*seen))
				}
				if got := (*seen)[0]; got.Kind != devDynamic || got.Index != int(index) {
					t.Errorf("index %d: deviation %v, want %s[%d]", index, got, devDynamic, index)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// The eight Trigger* (Trip.c:146-231)
// ---------------------------------------------------------------------------

// TestTriggersStartTheirObject is the seven dinah-based pokes on a freshly registered
// object of their own type. TriggerSwitch is the eighth and is below, because its argument
// is not a dinahs slot.
//
// The `want` closures assert against values derived from typicalObject's height 37 and
// delay 15, spelled out in each comment rather than recomputed, so that a change to the
// registration arithmetic fails here loudly instead of moving both sides together.
func TestTriggersStartTheirObject(t *testing.T) {
	cases := []struct {
		name  string
		what  int16
		fn    func(*World, int16)
		sound []int16
		want  func(*testing.T, *Dynamic, Dynamic)
	}{
		{
			// Count = launchVelocity(37) = 9, so VVel = -9. Frame drops to 0 from the
			// idle countdown's Delay*3 = 45.
			"toast", Toaster, (*World).TriggerToast, []int16{ToastLaunchSound},
			func(t *testing.T, d *Dynamic, was Dynamic) {
				if d.VVel != -was.Count {
					t.Errorf("VVel = %d, want -Count = %d", d.VVel, -was.Count)
				}
				if was.Count != 9 {
					t.Errorf("Count = %d, want launchVelocity(37) = 9", was.Count)
				}
				if d.Frame != 0 || !d.Moving {
					t.Errorf("Frame = %d, Moving = %v, want 0 and true", d.Frame, d.Moving)
				}
			},
		},
		{
			// Position is the outlet's launch enum; LengthOfZap is 30 frames.
			"outlet", Outlet, (*World).TriggerOutlet, []int16{ZapSound},
			func(t *testing.T, d *Dynamic, was Dynamic) {
				if d.Position != 1 {
					t.Errorf("Position = %d, want 1", d.Position)
				}
				if d.Timer != LengthOfZap {
					t.Errorf("Timer = %d, want LengthOfZap = %d", d.Timer, LengthOfZap)
				}
			},
		},
		{
			// The only poke that *shortens* a countdown. A drip reads the enemy union's
			// delay -- offset 6, not the appliance's 7 -- so typicalObject's 11 gives
			// (11*6)/2 = 33, down to 7: the length of HandleDrip's swell animation.
			"drip", Drip, (*World).TriggerDrip, nil,
			func(t *testing.T, d *Dynamic, was Dynamic) {
				if was.Timer != 33 {
					t.Errorf("registered Timer = %d, want (11*6)/2 = 33", was.Timer)
				}
				if d.Timer != 7 {
					t.Errorf("Timer = %d, want 7", d.Timer)
				}
			},
		},
		{
			// The only poke that touches a rect, and the only one that starts a mover
			// directly rather than through the sparkle countdown.
			"fish", Fish, (*World).TriggerFish, []int16{FishOutSound},
			func(t *testing.T, d *Dynamic, was Dynamic) {
				if d.Whole != d.Dest {
					t.Errorf("Whole = %+v, Dest = %+v; the union was not collapsed", d.Whole, d.Dest)
				}
				if d.Frame != 4 || !d.Moving {
					t.Errorf("Frame = %d, Moving = %v, want 4 and true", d.Frame, d.Moving)
				}
			},
		},
		{
			"balloon", Balloon, (*World).TriggerBalloon, nil,
			func(t *testing.T, d *Dynamic, was Dynamic) { wantSparkleLead(t, d) },
		},
		{
			"copter", CopterLf, (*World).TriggerCopter, nil,
			func(t *testing.T, d *Dynamic, was Dynamic) { wantSparkleLead(t, d) },
		},
		{
			"dart", DartRt, (*World).TriggerDart, nil,
			func(t *testing.T, d *Dynamic, was Dynamic) { wantSparkleLead(t, d) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := registerOne(t, tc.what, typicalObject(tc.what), testWhere)
			var snd soundLog
			snd.install(w)

			was := w.Dinahs[0]
			tc.fn(w, 0)
			tc.want(t, &w.Dinahs[0], was)
			snd.is(t, tc.sound...)
		})
	}
}

// wantSparkleLead is the shared assertion for the three enemy pokes, and the `+ 1` is the
// whole of it.
//
// The idle arms decrement Timer and then test `<= 0` before `== StartSparkle`, so
// StartSparkle + 1 gives a puff on the next frame and a launch four frames after that.
// Setting StartSparkle itself would take Timer to 3 on the first decrement, skip the
// `== 4` arm entirely, and the enemy would appear with no puff at all.
func wantSparkleLead(t *testing.T, d *Dynamic) {
	t.Helper()
	if d.Timer != StartSparkle+1 {
		t.Errorf("Timer = %d, want StartSparkle + 1 = %d", d.Timer, StartSparkle+1)
	}
	if d.Timer == StartSparkle {
		t.Errorf("Timer = StartSparkle exactly, which skips the puff -- see TriggerBalloon")
	}
}

// TestTriggerToastOnInactiveRestartsTheCountdown is the else arm, and it is the one place
// in the game where poking something makes it later.
//
// For a toaster Frame is the idle countdown and Timer the reload period it refills from,
// so Frame = Timer moves Frame *up*: triggering a switched-off toaster mid-countdown makes
// the bread it was about to pop wait another whole period.
func TestTriggerToastOnInactiveRestartsTheCountdown(t *testing.T) {
	w := registerOne(t, Toaster, typicalObject(Toaster), testWhere)
	var snd soundLog
	snd.install(w)

	w.Dinahs[0].Active = false
	w.Dinahs[0].Frame = 2 // two frames from popping
	w.TriggerToast(0)

	if got, want := w.Dinahs[0].Frame, w.Dinahs[0].Timer; got != want {
		t.Errorf("Frame = %d, want Timer = %d", got, want)
	}
	if w.Dinahs[0].Frame != 45 {
		t.Errorf("Frame = %d, want the full Delay*3 = 45 period", w.Dinahs[0].Frame)
	}
	if w.Dinahs[0].Moving {
		t.Errorf("Moving = true; a switched-off toaster must not launch")
	}
	snd.is(t)
}

// TestTriggerOutletOnInactiveRestartsTheCountdown is the outlet's twin of the above, in
// this type's own two fields: Timer from Count where the toaster resets Frame from Timer.
func TestTriggerOutletOnInactiveRestartsTheCountdown(t *testing.T) {
	w := registerOne(t, Outlet, typicalObject(Outlet), testWhere)
	var snd soundLog
	snd.install(w)

	w.Dinahs[0].Active = false
	w.Dinahs[0].Timer = 2
	w.TriggerOutlet(0)

	if got, want := w.Dinahs[0].Timer, w.Dinahs[0].Count; got != want {
		t.Errorf("Timer = %d, want Count = %d", got, want)
	}
	if w.Dinahs[0].Position != 0 {
		t.Errorf("Position = %d; a switched-off outlet must not zap", w.Dinahs[0].Position)
	}
	snd.is(t)
}

// TestTriggerPokesRefuseAnObjectAlreadyInMotion pins the shared gate.
//
// Six of the seven are gated -- the toaster and the fish on Moving, the outlet on
// Position, the three enemies on Moving -- and the point of the gate is that a fuse fired
// at something already in flight does not extend its life or restart its animation. The
// drip is the exception in *form* only (its gate is `!Moving && Timer > 7`) and is
// covered here too.
func TestTriggerPokesRefuseAnObjectAlreadyInMotion(t *testing.T) {
	cases := []struct {
		name string
		what int16
		fn   func(*World, int16)
		busy func(*Dynamic)
	}{
		{"toast", Toaster, (*World).TriggerToast, func(d *Dynamic) { d.Moving = true }},
		{"outlet", Outlet, (*World).TriggerOutlet, func(d *Dynamic) { d.Position = 1 }},
		{"drip", Drip, (*World).TriggerDrip, func(d *Dynamic) { d.Moving = true }},
		{"fish", Fish, (*World).TriggerFish, func(d *Dynamic) { d.Moving = true }},
		{"balloon", Balloon, (*World).TriggerBalloon, func(d *Dynamic) { d.Moving = true }},
		{"copter", CopterLf, (*World).TriggerCopter, func(d *Dynamic) { d.Moving = true }},
		{"dart", DartRt, (*World).TriggerDart, func(d *Dynamic) { d.Moving = true }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := registerOne(t, tc.what, typicalObject(tc.what), testWhere)
			var snd soundLog
			snd.install(w)

			tc.busy(&w.Dinahs[0])
			before := w.Dinahs[0]
			tc.fn(w, 0)

			if w.Dinahs[0] != before {
				t.Errorf("an object already in motion was written\nbefore %+v\nafter  %+v",
					before, w.Dinahs[0])
			}
			snd.is(t)
		})
	}
}

// TestTriggerDripOnlyShortensAndIsIdempotent is the `Timer > 7` half of the drip's gate.
//
// A drip already inside its last seven frames is left alone, which is what stops repeated
// triggers from holding it at 7 forever -- the difference between a drip that eventually
// falls and one pinned at the start of its swell animation by a fuse chain.
func TestTriggerDripOnlyShortensAndIsIdempotent(t *testing.T) {
	w := registerOne(t, Drip, typicalObject(Drip), testWhere)

	w.TriggerDrip(0)
	if w.Dinahs[0].Timer != 7 {
		t.Fatalf("Timer = %d after the first trigger, want 7", w.Dinahs[0].Timer)
	}

	// Idempotent at 7, and never lengthening below it.
	for _, start := range []int16{7, 6, 4, 2, 1, 0} {
		w.Dinahs[0].Timer = start
		w.TriggerDrip(0)
		if w.Dinahs[0].Timer != start {
			t.Errorf("Timer %d -> %d; a drip inside its last seven frames must not be touched",
				start, w.Dinahs[0].Timer)
		}
	}
}

// TestTriggerFishRefusesWhenSwitchedOff is the one gate on Active among the eight.
//
// A switched-off toaster restarts its countdown and a switched-off outlet restarts its
// reload, but a switched-off fish does nothing at all -- the asymmetry is the author's and
// is what makes a fish switch an off switch rather than a mode switch.
func TestTriggerFishRefusesWhenSwitchedOff(t *testing.T) {
	w := registerOne(t, Fish, typicalObject(Fish), testWhere)
	var snd soundLog
	snd.install(w)

	w.Dinahs[0].Active = false
	before := w.Dinahs[0]
	w.TriggerFish(0)

	if w.Dinahs[0] != before {
		t.Errorf("a switched-off fish was written\nbefore %+v\nafter  %+v", before, w.Dinahs[0])
	}
	snd.is(t)
}

// TestTriggerSwitchBoundsAgainstHotSpotsNotDinahs is the whole of what can be tested
// about the eighth Trigger* while HandleSwitches is a stub -- and it is the half that
// matters, because the bound is what hazard H2 is about.
//
// `who` is a hotSpots index, not a dinahs slot: the six switch types carry one in DynaNum
// instead of a dinah number. So an index that is valid for the hot-spot table has to be
// accepted even when the dinahs table is empty, and an index that would be valid for the
// dinahs table has to be refused when the hot-spot table is shorter. Both directions are
// asserted, because getting this wrong in either one produces a game that mostly works.
func TestTriggerSwitchBoundsAgainstHotSpotsNotDinahs(t *testing.T) {
	// Three hot spots, no dinahs at all: every valid hot-spot index is accepted.
	w := dynaWorld(t)
	w.R.Hot = make([]HotObject, 3)
	seen := deviations(w)
	for who := int16(0); who < 3; who++ {
		w.TriggerSwitch(who)
	}
	if w.NumDynamics != 0 {
		t.Fatalf("fixture registered %d dinahs; it has to be empty for this to prove anything",
			w.NumDynamics)
	}
	if len(*seen) != 0 {
		t.Errorf("hot spots 0..2 of 3 produced %v; TriggerSwitch is bounded on the wrong table",
			*seen)
	}

	// And the other direction: a dinah exists at slot 0, the hot-spot table is empty,
	// so slot 0 must be refused -- and refused as a hot spot. A switch is not one of the
	// seventeen registrable types, so the dinah here is a balloon; what matters is only
	// that the dinahs table is longer than the hot-spot table.
	w = dynaWorld(t)
	w.AddDynamicObject(Balloon, testWhere, typicalObject(Balloon), testRoom, testSlot, true)
	if w.NumDynamics != 1 {
		t.Fatalf("NumDynamics = %d, want 1", w.NumDynamics)
	}
	w.R.Hot = nil
	seen = deviations(w)
	w.TriggerSwitch(0)
	if len(*seen) != 1 {
		t.Fatalf("%d deviations, want 1 -- index 0 is valid for dinahs and must still be refused",
			len(*seen))
	}
	if got := (*seen)[0]; got.Kind != devHotSpot || got.Limit != 0 {
		t.Errorf("deviation %v, want %s[0] of 0", got, devHotSpot)
	}
}

// ---------------------------------------------------------------------------
// UpdateOutletsLighting (Trip.c:235-244)
// ---------------------------------------------------------------------------

// TestUpdateOutletsLightingFiltersOnRoom pins the two filters and the bound.
//
// The room filter is load-bearing rather than decorative. numLights is a global that
// DrawLocale reassigns before each of the nine room draws and AddDynamicObject reads as it
// registers, so each room's outlets carry their own room's count. Without the filter,
// turning the central room's lights on would overwrite a neighbouring dark room's outlets
// with a non-zero count and its sockets would stop painting themselves out.
//
// This is also the only reader of Dynamic.Room in all 67 sources, which is why the
// dart's missing Room assignment is safe -- see TestDartRegistrationLeavesRoomZero.
func TestUpdateOutletsLightingFiltersOnRoom(t *testing.T) {
	w := dynaWorld(t)
	w.R.NumLights = 2

	if got := w.AddDynamicObject(Outlet, testWhere, typicalObject(Outlet), 3, 0, true); got != 0 {
		t.Fatalf("outlet in room 3 got slot %d, want 0", got)
	}
	if got := w.AddDynamicObject(Outlet, testWhere, typicalObject(Outlet), 4, 1, true); got != 1 {
		t.Fatalf("outlet in room 4 got slot %d, want 1", got)
	}
	if got := w.AddDynamicObject(Balloon, testWhere, typicalObject(Balloon), 3, 2, true); got != 2 {
		t.Fatalf("balloon in room 3 got slot %d, want 2", got)
	}

	// A slot past NumDynamics, to pin the loop's bound. Nothing composes this; it stands
	// in for a stale slot left by a previous, busier room.
	w.Dinahs[5] = Dynamic{Type: Outlet, Room: 3, HVel: 99}

	for _, d := range w.Dinahs[:3] {
		if d.Type == Outlet && d.HVel != 2 {
			t.Fatalf("an outlet registered with HVel %d, want the room's 2 lights", d.HVel)
		}
	}

	w.UpdateOutletsLighting(3, 7)

	if got := w.Dinahs[0].HVel; got != 7 {
		t.Errorf("room 3's outlet HVel = %d, want 7", got)
	}
	if got := w.Dinahs[1].HVel; got != 2 {
		t.Errorf("room 4's outlet HVel = %d, want its own 2 -- the room filter leaked", got)
	}
	if got := w.Dinahs[2].HVel; got != 0 {
		t.Errorf("room 3's balloon HVel = %d, want 0 -- the type filter leaked", got)
	}
	if got := w.Dinahs[5].HVel; got != 99 {
		t.Errorf("slot 5 HVel = %d, want 99 -- the loop ran past NumDynamics", got)
	}
}

// ---------------------------------------------------------------------------
// ArmTrigger and the countdown (Triggers.c:34-74, 137-148)
// ---------------------------------------------------------------------------

// TestArmTriggerCopiesTheLinkAndTheDelay pins the four fields ArmTrigger writes, at three
// author delays.
//
// The delay is a big-endian *short* at payload offset 4 -- the only multi-byte field any of
// these transcriptions reads -- and is multiplied by three, so the editor's 0..127 covers
// roughly nought to twelve seconds. Room and Object are the *linked* object's house
// coordinates and not the plate's, which is what lets FireTrigger hand them straight to
// SetObjectState.
func TestArmTriggerCopiesTheLinkAndTheDelay(t *testing.T) {
	for _, delay := range []int16{0, 1, 42} {
		w := dynaWorld(t)
		w.R.Master = linkedPair(Guitar, -1, delay)
		hot := HotObject{Who: 0, Action: TriggerIt}

		w.ArmTrigger(&hot)

		got := w.Triggers[0]
		if !got.Armed {
			t.Fatalf("delay %d: slot 0 is not armed", delay)
		}
		if got.Timer != delay*3 {
			t.Errorf("delay %d: Timer = %d, want delay*3 = %d", delay, got.Timer, delay*3)
		}
		if got.Room != 0 || got.Object != 1 {
			t.Errorf("delay %d: (Room, Object) = (%d, %d), want the link's (0, 1)",
				delay, got.Room, got.Object)
		}
		if got.Index != 0 {
			t.Errorf("delay %d: Index = %d, want the plate's own master index 0", delay, got.Index)
		}
		if !hot.StillOver {
			t.Errorf("delay %d: StillOver = false; the rect was not marked used", delay)
		}
	}
}

// TestArmTriggerFiresOncePerStepOn is the StillOver guard, which is both the guard and the
// flag: the early return makes a plate arm once per step-on rather than once per frame.
func TestArmTriggerFiresOncePerStepOn(t *testing.T) {
	w := dynaWorld(t)
	w.R.Master = linkedPair(Guitar, -1, 5)
	hot := HotObject{Who: 0, Action: TriggerIt}

	w.ArmTrigger(&hot)
	w.ArmTrigger(&hot)
	w.ArmTrigger(&hot)

	armed := 0
	for i := range w.Triggers {
		if w.Triggers[i].Armed {
			armed++
		}
	}
	if armed != 1 {
		t.Errorf("%d slots armed after three frames of standing on the plate, want 1", armed)
	}

	// Stepping off and back on arms it again.
	hot.StillOver = false
	w.ArmTrigger(&hot)
	armed = 0
	for i := range w.Triggers {
		if w.Triggers[i].Armed {
			armed++
		}
	}
	if armed != 2 {
		t.Errorf("%d slots armed after stepping off and back on, want 2", armed)
	}
}

// TestArmTriggerConsumesTheSeventeenthTrigger pins the silent drop at the cap.
//
// FindEmptyTriggerSlot returns -1 when all sixteen are busy and ArmTrigger then does
// nothing *except* set StillOver -- so a seventeenth simultaneous trigger is consumed and
// never fires, and stays dropped until the glider steps off and back on. Kept faithful,
// like the other caps.
func TestArmTriggerConsumesTheSeventeenthTrigger(t *testing.T) {
	w := dynaWorld(t)
	w.R.Master = linkedPair(Guitar, -1, 5)

	for i := 0; i < MaxTriggers; i++ {
		hot := HotObject{Who: 0, Action: TriggerIt}
		w.ArmTrigger(&hot)
		if !w.Triggers[i].Armed {
			t.Fatalf("slot %d did not arm; first-fit is not filling in order", i)
		}
	}
	if got := w.FindEmptyTriggerSlot(); got != -1 {
		t.Fatalf("FindEmptyTriggerSlot = %d with all %d busy, want -1", got, MaxTriggers)
	}

	before := w.Triggers
	hot := HotObject{Who: 0, Action: TriggerIt}
	w.ArmTrigger(&hot)

	if w.Triggers != before {
		t.Errorf("the seventeenth trigger wrote the table")
	}
	if !hot.StillOver {
		t.Errorf("StillOver = false; a dropped trigger has to stay dropped until the glider " +
			"steps off, which is what the unconditional write is for")
	}
}

// TestTriggerTimingAtThreeDelays runs the countdown to the fire.
//
// HandleTriggers decrements and *then* tests `<= 0`, so a delay of 0 gives Timer 0, which
// is already `<= 0` on the very first tick: a zero-delay trigger fires on the frame after
// it is armed rather than instantly. That one-frame lag is the original's and is visible on
// a trigger wired to a light. Delay n fires on frame 3n for n > 0.
//
// The "exactly one sound" assertion is also the disarm test: FireTrigger can reach
// SetObjectState, which can re-enter this world, so HandleTriggers clears Armed *before*
// firing. Without that ordering the slot re-fires on every subsequent frame.
func TestTriggerTimingAtThreeDelays(t *testing.T) {
	for _, tc := range []struct{ delay, wantFrame int16 }{{0, 1}, {1, 3}, {42, 126}} {
		w := dynaWorld(t)
		w.R.Master = linkedPair(Guitar, -1, tc.delay)
		var snd soundLog
		snd.install(w)

		hot := HotObject{Who: 0, Action: TriggerIt}
		w.ArmTrigger(&hot)

		fired := int16(-1)
		for f := int16(1); f <= tc.wantFrame+3; f++ {
			w.HandleTriggers()
			if fired == -1 && len(snd.played) > 0 {
				fired = f
			}
		}

		if fired != tc.wantFrame {
			t.Errorf("delay %d fired on frame %d, want %d", tc.delay, fired, tc.wantFrame)
		}
		if len(snd.played) != 1 {
			t.Errorf("delay %d: %d sounds, want 1 -- the slot re-fired, so it was not "+
				"disarmed before firing", tc.delay, len(snd.played))
		}
		if w.Triggers[0].Armed {
			t.Errorf("delay %d: slot 0 is still armed after firing", tc.delay)
		}
		if w.Triggers[0].Timer != 0 {
			t.Errorf("delay %d: Timer = %d after firing, want it clamped to 0",
				tc.delay, w.Triggers[0].Timer)
		}
	}
}

// TestZeroTriggersClearsOnlyArmed is DrawLocale's reset head: every room change drops
// fuses that were still burning, so a trigger armed on the way out of a room never fires.
//
// It leaves Timer and the three indices stale, which is correct rather than sloppy --
// ArmTrigger overwrites all of them, and leaving them readable is what makes a post-mortem
// dump of the table informative.
func TestZeroTriggersClearsOnlyArmed(t *testing.T) {
	w := dynaWorld(t)
	w.Triggers[3] = TriggerSlot{Object: 9, Room: 8, Index: 7, Timer: 6, What: 5, Armed: true}
	want := w.Triggers[3]
	want.Armed = false

	w.ZeroTriggers()

	if w.Triggers[3] != want {
		t.Errorf("slot 3 = %+v, want %+v -- only Armed may be cleared", w.Triggers[3], want)
	}
	for i := range w.Triggers {
		if w.Triggers[i].Armed {
			t.Errorf("slot %d is still armed", i)
		}
	}
}

// TestArmTriggerWhatReadsTheWrongTable pins a transcribed bug, deliberately.
//
// The C reads `masterObjects[triggers[where].object].theObject.what`, where `.object` is
// an object *number within its room*, 0..23 -- so it names whichever object of the first
// room in the locale happens to sit at that slot, not the linked object. The correct index
// is localLink, which FireTrigger uses.
//
// Nothing reads TriggerSlot.What, so the wrong value is inert. It is kept because a port
// that "fixed" it would make the field meaningful and invite a later reader to use it, and
// because a house test that dumps the trigger table has to match the original's numbers.
// This test is the marker: it fails if someone corrects the read.
func TestArmTriggerWhatReadsTheWrongTable(t *testing.T) {
	w := dynaWorld(t)
	plate := house.Object{What: Trigger}
	plate.SetSwitch(house.Switch{Delay: 1, Where: 0, Who: 1})

	// Object number 1 lives at master index 2, so the buggy read (master[1]) and the
	// correct one (master[localLink] = master[2]) name different types.
	w.R.Master = []MasterObject{
		{RoomNum: 0, ObjectNum: 0, RoomLink: 0, ObjectLink: 1, LocalLink: 2,
			HotNum: -1, DynaNum: -1, TheObject: plate},
		{RoomNum: 9, ObjectNum: 0, RoomLink: -1, ObjectLink: -1, LocalLink: -1,
			HotNum: -1, DynaNum: -1, TheObject: house.Object{What: Table}},
		{RoomNum: 0, ObjectNum: 1, RoomLink: -1, ObjectLink: -1, LocalLink: -1,
			HotNum: -1, DynaNum: -1, TheObject: house.Object{What: Guitar}},
	}

	hot := HotObject{Who: 0, Action: TriggerIt}
	w.ArmTrigger(&hot)

	if got := w.Triggers[0].What; got != Table {
		t.Errorf("What = %#x, want the bug's %#x (kTable at master[1]) and not the "+
			"correct %#x (kGuitar at master[localLink])", got, Table, Guitar)
	}

	// And the bug is inert: FireTrigger looks the target up itself, through localLink.
	var snd soundLog
	snd.install(w)
	w.HandleTriggers()
	w.HandleTriggers()
	w.HandleTriggers()
	snd.is(t, ChordSound)
}

// ---------------------------------------------------------------------------
// FireTrigger's dispatch (Triggers.c:100-192)
// ---------------------------------------------------------------------------

// noDinah is the DynaNum of a target that never registered, which is what an object past
// the eighteen-slot cap carries.
const noDinah int16 = -1

// TestFireTriggerDispatch walks every live arm of the local half.
//
// Thirteen arms and one `default`, and the interesting thing about the shape is how
// lopsided the local/remote split is: twelve of the thirteen need a dynamic-object slot
// and only objects in the locale have one. So a trigger wired to a balloon two rooms away
// does nothing at all, while a trigger wired to grease two rooms away does spill it,
// invisibly, and the player finds it on arrival -- which is the only way a house can change
// a room the player has not reached. See TestFireTriggerRemoteHalfSpillsGreaseOnly.
func TestFireTriggerDispatch(t *testing.T) {
	cases := []struct {
		name string
		// target is the linked object's type; register is a type to put in dinah slot 0,
		// or noDinah for none; dynaNum is what the target's DynaNum holds.
		target, register, dynaNum int16
		hotSpots                  int
		setup                     func(*World)
		sounds                    []int16
		check                     func(*testing.T, *World)
	}{
		{
			name: "greaseRt spills", target: GreaseRt, register: noDinah, dynaNum: 4,
			setup: func(w *World) { greaseAt(w, GreaseRt) },
			check: func(t *testing.T, w *World) { wantSpilled(t, w) },
		},
		{
			name: "greaseLf spills", target: GreaseLf, register: noDinah, dynaNum: 4,
			setup: func(w *World) { greaseAt(w, GreaseLf) },
			check: func(t *testing.T, w *World) { wantSpilled(t, w) },
		},
		// The six switch arms. DynaNum is a hotSpots index here, not a dinahs slot, and
		// with the hot-spot table empty the refusal is the proof the arm was taken. That
		// is still the right observable now HandleSwitches is real: a fixture with a
		// populated hot-spot table would prove the dispatch through the switch's own
		// effects, which is switches_test.go's job.
		{name: "lightSwitch throws", target: LightSwitch, register: noDinah, dynaNum: 0,
			check: wantForwardedToASwitch},
		{name: "machineSwitch throws", target: MachineSwitch, register: noDinah, dynaNum: 0,
			check: wantForwardedToASwitch},
		{name: "thermostat throws", target: Thermostat, register: noDinah, dynaNum: 0,
			check: wantForwardedToASwitch},
		{name: "powerSwitch throws", target: PowerSwitch, register: noDinah, dynaNum: 0,
			check: wantForwardedToASwitch},
		{name: "knifeSwitch throws", target: KnifeSwitch, register: noDinah, dynaNum: 0,
			check: wantForwardedToASwitch},
		{name: "invisSwitch throws", target: InvisSwitch, register: noDinah, dynaNum: 0,
			check: wantForwardedToASwitch},
		{
			// The author's own "// Change me": the custom sound loader was never
			// written, so a remotely triggered sound trigger strums the guitar.
			name: "soundTrigger strums", target: SoundTrigger, register: noDinah,
			dynaNum: noDinah, sounds: []int16{ChordSound},
		},
		{
			name: "guitar strums", target: Guitar, register: noDinah, dynaNum: noDinah,
			sounds: []int16{ChordSound},
		},
		{
			// No dinah needed: this arm only plays the percolator.
			name: "coffee percolates", target: Coffee, register: noDinah, dynaNum: noDinah,
			sounds: []int16{CoffeeSound},
		},
		{
			name: "toaster launches", target: Toaster, register: Toaster, dynaNum: 0,
			sounds: []int16{ToastLaunchSound},
			check: func(t *testing.T, w *World) {
				if !w.Dinahs[0].Moving || w.Dinahs[0].Frame != 0 {
					t.Errorf("Moving = %v, Frame = %d, want true and 0",
						w.Dinahs[0].Moving, w.Dinahs[0].Frame)
				}
			},
		},
		{
			name: "outlet zaps", target: Outlet, register: Outlet, dynaNum: 0,
			sounds: []int16{ZapSound},
			check: func(t *testing.T, w *World) {
				if w.Dinahs[0].Position != 1 || w.Dinahs[0].Timer != LengthOfZap {
					t.Errorf("Position = %d, Timer = %d, want 1 and %d",
						w.Dinahs[0].Position, w.Dinahs[0].Timer, LengthOfZap)
				}
			},
		},
		{
			name: "balloon rises", target: Balloon, register: Balloon, dynaNum: 0,
			check: wantSparkleLeadOnSlotZero,
		},
		{
			name: "copterLf drops", target: CopterLf, register: CopterLf, dynaNum: 0,
			check: wantSparkleLeadOnSlotZero,
		},
		{
			name: "copterRt drops", target: CopterRt, register: CopterRt, dynaNum: 0,
			check: wantSparkleLeadOnSlotZero,
		},
		{
			name: "dartLf flies", target: DartLf, register: DartLf, dynaNum: 0,
			check: wantSparkleLeadOnSlotZero,
		},
		{
			name: "dartRt flies", target: DartRt, register: DartRt, dynaNum: 0,
			check: wantSparkleLeadOnSlotZero,
		},
		{
			name: "drip swells", target: Drip, register: Drip, dynaNum: 0,
			check: func(t *testing.T, w *World) {
				if w.Dinahs[0].Timer != 7 {
					t.Errorf("Timer = %d, want 7", w.Dinahs[0].Timer)
				}
			},
		},
		{
			name: "fish leaps", target: Fish, register: Fish, dynaNum: 0,
			sounds: []int16{FishOutSound},
			check: func(t *testing.T, w *World) {
				if !w.Dinahs[0].Moving || w.Dinahs[0].Frame != 4 {
					t.Errorf("Moving = %v, Frame = %d, want true and 4",
						w.Dinahs[0].Moving, w.Dinahs[0].Frame)
				}
			},
		},
		// The default arm. A trigger wired to a table, a clock or -- notably -- another
		// trigger plate arms, counts down, fires and does nothing. Triggers chain through
		// *switches*, not through each other.
		{name: "table does nothing", target: Table, register: noDinah, dynaNum: noDinah},
		{name: "trigger does not chain", target: Trigger, register: noDinah, dynaNum: 0},
		{name: "lgTrigger does not chain", target: LgTrigger, register: noDinah, dynaNum: 0},
		{name: "redClock does nothing", target: RedClock, register: noDinah, dynaNum: noDinah},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := dynaWorld(t)
			if tc.register != noDinah {
				if got := w.AddDynamicObject(tc.register, testWhere,
					typicalObject(tc.register), testRoom, testSlot, true); got != 0 {
					t.Fatalf("AddDynamicObject(%#x) = %d, want 0", tc.register, got)
				}
			}
			if tc.hotSpots > 0 {
				w.R.Hot = make([]HotObject, tc.hotSpots)
			}
			if tc.setup != nil {
				tc.setup(w)
			}
			var snd soundLog
			snd.install(w)

			fireAt(w, tc.target, tc.dynaNum)

			snd.is(t, tc.sounds...)
			if tc.check != nil {
				tc.check(t, w)
			}
		})
	}
}

// wantSpilled is the grease arms' assertion. ForceOn rather than Toggle, so a trigger can
// only ever spill grease and never un-spill it -- and the bonus family ignores `action`
// anyway and always clears the state, which is what makes a spill one-way twice over.
func wantSpilled(t *testing.T, w *World) {
	t.Helper()
	if got := w.Room(0).Objects[1].Bonus().State; got != 0 {
		t.Errorf("the jar's state byte = %d, want 0 -- the grease was not spilled", got)
	}
	// The master copy is written through the *blower* union member, which is the
	// transcribed bug SetObjectState's bonus family documents.
	if got := w.R.Master[1].TheObject.Data[offBlowerState]; got != 0 {
		t.Errorf("master copy's blower-state byte = %d, want 0", got)
	}
}

// wantForwardedToASwitch is the six switch arms' assertion: TriggerSwitch was reached with
// the target's DynaNum, and refused it because the hot-spot table is empty. The refusal is
// what makes the forward observable without also dragging in everything a real switch does.
func wantForwardedToASwitch(t *testing.T, w *World) {
	t.Helper()
	if len(w.Diag.Seen) != 1 || w.Diag.Seen[0].Kind != devHotSpot {
		t.Errorf("deviations %v, want one %s -- the switch arm was not taken",
			w.Diag.Seen, devHotSpot)
	}
}

func wantSparkleLeadOnSlotZero(t *testing.T, w *World) {
	t.Helper()
	wantSparkleLead(t, &w.Dinahs[0])
}

// TestSevenTypesAreNotTriggerable is the test trip.go's header points at by name.
//
// Six appliances have a switch but no fuse -- a trigger cannot start a Mac, a TV, a VCR, a
// stereo or a microwave, and cannot start a ball -- and kSparkle has neither. The asymmetry
// is by design: it is what makes a ball a fixture of a room's layout rather than a hazard a
// fuse can spring.
//
// Pinned as no-ops so that a later editor of FireTrigger's switch cannot quietly add a
// fall-through. The assertion is the whole dinahs table byte-for-byte, plus silence, plus
// no guarded read -- so an arm that merely *indexed* the slot would fail too.
func TestSevenTypesAreNotTriggerable(t *testing.T) {
	for _, what := range []int16{Sparkle, MacPlus, TV, VCR, Stereo, Microwave, Ball} {
		t.Run(house.ObjectName(what), func(t *testing.T) {
			w := registerOne(t, what, typicalObject(what), testWhere)
			var snd soundLog
			snd.install(w)
			seen := deviations(w)
			before := w.Dinahs

			fireAt(w, what, 0)

			if w.Dinahs != before {
				t.Errorf("the dinahs table was written\nbefore %+v\nafter  %+v",
					before[0], w.Dinahs[0])
			}
			snd.is(t)
			if len(*seen) != 0 {
				t.Errorf("deviations %v, want none -- the slot was indexed", *seen)
			}
		})
	}

	// And the count: seven, not six or eight. FireTrigger's switch covers twelve
	// registrable types across its arms; these are the other five plus kSparkle... which
	// is to say, the arithmetic is worth stating rather than trusting.
	triggerable := 0
	for _, what := range allRegistrableTypes() {
		switch what {
		case Sparkle, MacPlus, TV, VCR, Stereo, Microwave, Ball:
		default:
			triggerable++
		}
	}
	if want := len(allRegistrableTypes()) - 7; triggerable != want {
		t.Errorf("%d triggerable registrable types, want %d", triggerable, want)
	}
}

// TestFireTriggerRemoteHalfSpillsGreaseOnly is the lopsided other half.
//
// LocalLink == -1 means the target is not one of the nine local rooms, and the remote half
// has exactly one arm: grease. Everything else -- a balloon, an outlet, a fish two rooms
// away -- does nothing, because it would need a dynamic-object slot and only objects in the
// locale have one.
//
// Note what the C does with localLink after having proved it is -1: it passes it to
// SetObjectState as `local`, and -1 there means "write the house, publish nothing". So the
// apparently pointless line is load-bearing, and the master copy must be left alone.
func TestFireTriggerRemoteHalfSpillsGreaseOnly(t *testing.T) {
	remote := func(w *World, targetWhat int16) {
		w.R.Master = linkedPair(targetWhat, noDinah, 0)
		w.R.Master[0].LocalLink = -1
		w.Triggers[0] = TriggerSlot{Room: 0, Object: 1, Index: 0, What: targetWhat, Armed: true}
		w.FireTrigger(0)
	}

	t.Run("grease spills invisibly", func(t *testing.T) {
		w := dynaWorld(t)
		greaseAt(w, GreaseRt)
		var snd soundLog
		snd.install(w)

		remote(w, GreaseRt)

		if got := w.Room(0).Objects[1].Bonus().State; got != 0 {
			t.Errorf("the jar's state byte = %d, want 0 -- a remote trigger has to spill", got)
		}
		// local == -1, so no publication: the master copy keeps its own state.
		if got := w.R.Master[1].TheObject.Data[offBlowerState]; got != 0 {
			t.Errorf("master blower-state byte = %d; the fixture's copy should be 0 already", got)
		}
		snd.is(t)
	})

	t.Run("a remote balloon does nothing", func(t *testing.T) {
		w := registerOne(t, Balloon, typicalObject(Balloon), testWhere)
		var snd soundLog
		snd.install(w)
		before := w.Dinahs

		// The remote switch reads the *room's* object, not the master copy, so the
		// balloon has to be in the room for this to be the balloon case at all.
		w.Room(0).Objects[1] = typicalObject(Balloon)
		remote(w, Balloon)

		if w.Dinahs != before {
			t.Errorf("a remote balloon trigger reached the dinahs table")
		}
		snd.is(t)
	})

	t.Run("an empty slot does nothing", func(t *testing.T) {
		w := dynaWorld(t)
		var snd soundLog
		snd.install(w)
		remote(w, GreaseRt) // room 0 slot 1 is still kObjectIsEmpty
		snd.is(t)
	})
}

// TestFireTriggerGuardsEveryIndex covers the three guarded reads on the way in.
//
// All three are reachable from a malformed or hand-edited house, and all three are
// unguarded in the C. The trigger index is bounded against the sixteen-slot table; the
// plate's own master index and the target's localLink are bounded against the composed
// graph, whose length depends on how many neighbours the locale has -- so a house saved
// from a nine-room locale and reopened in a one-room one has links past the end.
func TestFireTriggerGuardsEveryIndex(t *testing.T) {
	t.Run("trigger index", func(t *testing.T) {
		w := dynaWorld(t)
		seen := deviations(w)
		w.FireTrigger(MaxTriggers)
		if len(*seen) != 1 || (*seen)[0].Kind != devTrigger {
			t.Errorf("deviations %v, want one %s", *seen, devTrigger)
		}
	})

	t.Run("the plate's own master index", func(t *testing.T) {
		w := dynaWorld(t)
		w.R.Master = linkedPair(Guitar, noDinah, 0)
		w.Triggers[0] = TriggerSlot{Index: 99, Armed: true}
		seen := deviations(w)
		var snd soundLog
		snd.install(w)

		w.FireTrigger(0)

		if len(*seen) != 1 || (*seen)[0].Kind != devMasterObject {
			t.Errorf("deviations %v, want one %s", *seen, devMasterObject)
		}
		snd.is(t)
	})

	t.Run("the target's localLink", func(t *testing.T) {
		w := dynaWorld(t)
		w.R.Master = linkedPair(Guitar, noDinah, 0)
		w.R.Master[0].LocalLink = 99
		w.Triggers[0] = TriggerSlot{Index: 0, Armed: true}
		seen := deviations(w)
		var snd soundLog
		snd.install(w)

		w.FireTrigger(0)

		if len(*seen) != 1 || (*seen)[0].Kind != devMasterObject {
			t.Errorf("deviations %v, want one %s", *seen, devMasterObject)
		}
		snd.is(t)
	})

	t.Run("the remote half's room object slot", func(t *testing.T) {
		w := dynaWorld(t)
		w.R.Master = linkedPair(GreaseRt, noDinah, 0)
		w.R.Master[0].LocalLink = -1
		w.Triggers[0] = TriggerSlot{Room: 0, Object: MaxRoomObs, Index: 0, Armed: true}
		seen := deviations(w)

		w.FireTrigger(0)

		if len(*seen) != 1 || (*seen)[0].Kind != devRoomObject {
			t.Errorf("deviations %v, want one %s", *seen, devRoomObject)
		}
	})
}
