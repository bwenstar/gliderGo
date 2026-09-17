package game

// The switch path: HandleSwitches' latch and lever, and switchLinkedObject's twenty-three
// arms.
//
// A switch is the game's one general-purpose mechanism -- an author can wire one to any of
// forty-odd object types -- so the subject here is coverage of a dispatch table rather than
// the behaviour of any one arm. TestTwentyThreeSwitchArms is that coverage: one row per arm,
// every object type in the arm exercised, and the same five observables asserted on all of
// them.
//
// Three things this file establishes that reading switches.go does not make obvious.
//
// **Three arms are unreachable.** kSlider, kSoundTrigger and kGuitar are all types
// SetObjectState refuses, and the caller only reaches the second switch when SetObjectState
// returned true. So those three arms are transcribed dead code -- worth keeping, because
// they say what the author intended, and worth pinning, because a later change to
// SetObjectState would silently bring them to life.
// TestThreeSwitchArmsCannotBeReached is the assertion.
//
// **The lever is drawn from the *linked* object's state, not the switch's own.** That works
// only because SetObjectState publishes the state it computed through World.NewState and
// HandleSwitches reads it back. It is the regression this stage fixed and it is invisible
// without art, so TestLeverShowsTheLinkedObjectsState renders both cases and compares
// pixels; TestLeverStateComesFromTheLinkedObject pins the same thing on the field, for the
// runs where the art is not extracted.
//
// **A switch wired to a star makes the house unwinnable.** The star is grouped with the
// eight other prizes, so it gets neither StopStar nor a decrement of the star count, and
// nothing else in the game can decrement it once the object is gone.
// TestSwitchOnAStarStrandsTheHouse pins the bug so that the eventual fix is a deliberate
// change to a test rather than an accident.

import (
	"bytes"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// switchAt is the room-local rect of the plate: 32x20 at (40, 40), which is roughly the
// size of the light-switch artwork and far enough from prizeAt that a rect registered for
// one cannot be mistaken for the other in a failure message.
var switchAt = render.SetRect(40, 40, 72, 60)

// switchWorld builds the two-entry master graph HandleSwitches walks: a plate in object
// slot 0 whose link names a target in object slot 1, both in the one room.
//
// The graph is spelt out rather than composed from a house because the composition is not
// the subject and would fail for a dozen unrelated reasons -- see trip_test.go's header,
// which makes the same argument for FireTrigger.
//
// `action` is the plate's own data.e.type: Toggle, ForceOn or ForceOff. `localLink` is
// master[0].LocalLink, and -1 is the "target is not in the locale" case that skips the
// second dispatch entirely. `withDinah` registers the target in the dinahs table and points
// master[1].DynaNum at it, which the fourteen appliance and enemy arms need.
func switchWorld(t *testing.T, switchWhat int16, target house.Object, action int16,
	localLink int16, withDinah bool) (*World, *HotObject) {
	t.Helper()
	w := dynaWorld(t)
	w.R.RoomNumber = 0

	// Zero rather than dynaWorld's default, because the eight light arms turn on the
	// room's only lamp and RedrawRoomLighting only redraws on the unlit->lit crossing.
	// A non-zero count here would make every light row assert nothing.
	w.R.NumLights = 0

	plate := house.Object{What: switchWhat}
	plate.SetSwitch(house.Switch{
		TopLeft: house.Point{V: switchAt.Top, H: switchAt.Left},
		Delay:   0,
		Where:   0,
		Who:     1,
		Type:    byte(action),
	})
	w.Room(0).Objects[0] = plate
	w.Room(0).Objects[1] = target
	w.Room(0).NumObjects = 2

	dynaNum := int16(-1)
	if withDinah {
		slot := w.AddDynamicObject(target.What, testWhere, target, 0, 1, true)
		if slot != 0 {
			t.Fatalf("AddDynamicObject(%s) = %d, want slot 0",
				house.ObjectName(target.What), slot)
		}
		dynaNum = 0
		// ToggleStereos is the one poke of the fourteen that refuses to run while the
		// slot's timer is non-zero. Zeroing it here keeps that refusal out of the
		// dispatch table's business; TestToggleStereosRefusesWhileTimerRuns owns it.
		w.Dinahs[0].Timer = 0
	}

	w.R.Master = []MasterObject{{
		RoomNum: 0, ObjectNum: 0,
		RoomLink: 0, ObjectLink: 1, LocalLink: localLink,
		// A switch's DynaNum is its own HotNum, not a dinahs slot. See
		// MasterObject.DynaNum on the field's three meanings.
		HotNum: 0, DynaNum: 0,
		TheObject: plate,
	}, {
		RoomNum: 0, ObjectNum: 1,
		RoomLink: -1, ObjectLink: -1, LocalLink: -1,
		HotNum: 1, DynaNum: dynaNum,
		TheObject: target,
	}}
	w.R.Hot = []HotObject{
		{Bounds: switchAt, Action: SwitchIt, Who: 0, IsOn: true},
		{Bounds: prizeAt, Action: RewardIt, Who: 1, IsOn: true},
	}
	w.R.SavedMaps = []render.SavedMap{
		savedMapUnder(0, 1, prizeAt, w.R.V.OriginH, w.R.V.OriginV)}

	clearRects(w)
	return w, &w.R.Hot[0]
}

// The four target payload builders. Which one a type needs is decided by the union member
// its family reads, and getting it wrong is silent: every family reads the same ten bytes,
// so a light built as an appliance still has *a* state byte, just not where SetObjectState
// looks. See TestFishReadsTheSameBytesAsTheEnemyLength in dynamics_test.go.

// prizeObj is a live, uncollected prize: State non-zero is what makes SetObjectState report
// a change, and therefore the only thing that lets a switch remove it.
func prizeObj(what int16) house.Object {
	o := house.Object{What: what}
	o.SetBonus(house.Bonus{
		TopLeft: house.Point{V: prizeAt.Top, H: prizeAt.Left},
		Length:  48, Points: 250, State: 1, Initial: 1,
	})
	return o
}

// lightObj is a lamp that is *off*, so that a Toggle turns it on and crosses the
// unlit -> lit boundary RedrawRoomLighting tests.
func lightObj(what int16) house.Object {
	o := house.Object{What: what}
	o.SetLight(house.Light{
		TopLeft: house.Point{V: 40, H: 200}, Length: 48, Initial: 0, State: 0,
	})
	return o
}

// transportObj is a deluxe transporter with a live link. Its state lives in the low nibble
// of `wide`, and 0 there means off -- so a Toggle turns it on.
func transportObj(what int16) house.Object {
	o := house.Object{What: what}
	o.SetTransport(house.Transport{
		TopLeft: house.Point{V: 100, H: 200}, Tall: 32, Where: 0, Who: 1, Wide: 0,
	})
	return o
}

// plainObj is for the types with no payload worth building: a switch, a trigger, a piece of
// furniture. Ten zero bytes, which is what an author's freshly placed object holds.
func plainObj(what int16) house.Object { return house.Object{What: what} }

// applianceTarget and enemyTarget adapt dynamics_test.go's two builders to the one-argument
// shape the table uses. The height/length and delay are arbitrary but non-zero: a zero delay
// would give some handlers a zero reload period, and although no row here runs a frame,
// a fixture that could not survive one is a trap for the next test in this file.
func applianceTarget(what int16) house.Object { return applianceObj(what, 48, 4) }
func enemyTarget(what int16) house.Object     { return enemyObj(what, 48, 4) }

// ---------------------------------------------------------------------------
// The twenty-three arms
// ---------------------------------------------------------------------------

// switchArm is one arm of switchLinkedObject, with every object type the arm covers.
//
// The five observables are the same on every row, including the rows that produce none of
// them: a work-rect count, the sounds in order, a sparkle count, whether SetObjectState
// reported a change, and an arm-specific check. Asserting the zeroes is what makes the
// table a specification rather than a smoke test -- see rewards_test.go's header, which
// makes the argument at length.
type switchArm struct {
	name  string
	types []int16
	obj   func(int16) house.Object

	// dinah registers the target in the dinahs table. True for exactly the fourteen arms
	// that end in a Toggle*, because a dinahs slot is the only thing those fourteen touch.
	dinah bool

	// reached is whether SetObjectState returns true for this arm's types, i.e. whether
	// the arm runs at all. False on three rows and that is the interesting part of the
	// table: see TestThreeSwitchArmsCannotBeReached.
	reached bool

	// noStateWrite marks a row that is reached and still leaves the target's ten payload
	// bytes alone. Exactly one row: SetObjectState's kStereo arm flips a *game* flag rather
	// than an object's state and returns true unconditionally, which is what makes the
	// stereo the only target a switch can throw twice with the same effect.
	noStateWrite bool

	// workRects counts every AddRectToWorkRects. One for the lever, which every reached
	// row pays; two where the arm also restores a saved map or recomposes the room.
	workRects int
	sounds    []int16
	sparkles  int

	check func(t *testing.T, w *World)
	note  string
}

// theSwitchArms is Interactions.c:1038-1148 in the original's order.
//
// Twenty-four rows for twenty-three arms: kInvisBonus and kSlider share one `break` in the
// C, and they are split here because only one of the two can be reached. A single row would
// have had to assert the weaker of the two behaviours.
var theSwitchArms = []switchArm{{
	name:  "nine prizes",
	types: []int16{RedClock, BlueClock, YellowClock, Paper, Battery, Bands, Foil, Star, Helium},
	obj:   prizeObj, reached: true,
	workRects: 2, sounds: []int16{SwitchSound, player.FadeOutSound}, sparkles: 2,
	check: func(t *testing.T, w *World) {
		if w.R.Hot[1].IsOn {
			t.Error("the prize's own hot spot is still on: SetObjectState clears it, and " +
				"without that the player could still collect a prize a switch removed")
		}
	},
	note: "two sparkles, and the second is the bug: see TestTheSpuriousSparkleLandsAtTheCorner",
}, {
	name:  "the cuckoo",
	types: []int16{Cuckoo}, obj: prizeObj, reached: true,
	workRects: 2, sounds: []int16{SwitchSound, player.FadeOutSound}, sparkles: 1,
	check: func(t *testing.T, w *World) {
		if len(w.R.Pendulums) > 0 && !w.R.Pendulums[0].Stopped {
			t.Error("Pendulums[0].Stopped = false: the swing outlives the clock")
		}
	},
	note: "one sparkle, not two: the cuckoo arm has no AddSparkle of its own, spurious or otherwise",
}, {
	name:  "the two grease jars",
	types: []int16{GreaseRt, GreaseLf}, obj: prizeObj, reached: true,
	workRects: 1, sounds: []int16{SwitchSound},
	note: "no restore: a knocked-over jar becomes a slick in place. SpillGrease is 1.5e",
}, {
	name:  "the invisible bonus",
	types: []int16{InvisBonus}, obj: prizeObj, reached: true,
	workRects: 1, sounds: []int16{SwitchSound},
	note: "explicitly nothing: it was never drawn, so there is nothing to put back",
}, {
	name:  "the slider",
	types: []int16{Slider}, obj: prizeObj, reached: false,
	note: "unreachable: SetObjectState's kSlider arm is a bare break and returns false",
}, {
	name:  "the deluxe transporter",
	types: []int16{DeluxeTrans}, obj: transportObj, reached: true,
	workRects: 1, sounds: []int16{SwitchSound},
	check: func(t *testing.T, w *World) {
		if w.Room(0).Objects[1].Data[offTransportWide]&0x0F == 0 {
			t.Error("the transporter's state nibble is still clear")
		}
	},
	note: "no redraw: its state is read at use time",
}, {
	name:  "the shredder",
	types: []int16{Shredder}, obj: applianceTarget, reached: true,
	workRects: 1, sounds: []int16{SwitchSound},
	note: "the blades are drawn by its own dynamic handler, so nothing to do here",
}, {
	name:  "the sound trigger",
	types: []int16{SoundTrigger}, obj: plainObj, reached: false,
	note: "unreachable: a sound trigger is in SetObjectState's *switch* family, which " +
		"returns false. The arm would have played the house's custom sound",
}, {
	name: "the eight lights",
	types: []int16{CeilingLight, LightBulb, TableLamp, HipLamp, DecoLamp, Flourescent,
		TrackLight, InvisLight},
	obj: lightObj, reached: true,
	workRects: 2, sounds: []int16{SwitchSound},
	check: func(t *testing.T, w *World) {
		if w.R.NumLights != 1 {
			t.Errorf("NumLights = %d, want 1: the lamp was switched on and the count "+
				"is recomputed from the house", w.R.NumLights)
		}
		if !w.R.ShadowVisible {
			t.Error("ShadowVisible = false: a lit room casts the glider's shadow, and " +
				"the recache is the last line of RedrawRoomLighting")
		}
	},
	note: "the second work rect is the whole central room: the unlit->lit crossing recomposes it",
}, {
	name:  "the guitar",
	types: []int16{Guitar}, obj: applianceTarget, reached: false,
	note: "unreachable: SetObjectState's comment is \"really no point to change this state\"",
}, {
	name: "the toaster", types: []int16{Toaster}, obj: applianceTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the Mac Plus", types: []int16{MacPlus}, obj: applianceTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the TV", types: []int16{TV}, obj: applianceTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the coffee pot", types: []int16{Coffee}, obj: applianceTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the outlet", types: []int16{Outlet}, obj: applianceTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the VCR", types: []int16{VCR}, obj: applianceTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the stereo", types: []int16{Stereo}, obj: applianceTarget, dinah: true,
	reached: true, noStateWrite: true, workRects: 1, sounds: []int16{SwitchSound},
	check: func(t *testing.T, w *World) {
		dinahWentIdle(t, w)
		if !w.R.PlayMusicGame {
			t.Error("PlayMusicGame = false: the stereo's SetObjectState arm is the one " +
				"that changes a game flag rather than an object, and it toggles " +
				"whatever the action asked for")
		}
	},
	note: "the one arm whose SetObjectState writes no object state and still returns true",
}, {
	name: "the microwave", types: []int16{Microwave}, obj: applianceTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the balloon", types: []int16{Balloon}, obj: enemyTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the two copters", types: []int16{CopterLf, CopterRt}, obj: enemyTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the two darts", types: []int16{DartLf, DartRt}, obj: enemyTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the ball", types: []int16{Ball}, obj: enemyTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the drip", types: []int16{Drip}, obj: enemyTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}, {
	name: "the fish", types: []int16{Fish}, obj: enemyTarget, dinah: true,
	reached: true, workRects: 1, sounds: []int16{SwitchSound}, check: dinahWentIdle,
}}

// dinahWentIdle is the check the fourteen Toggle* arms share.
//
// Every fixture registers with isOn true, so AddDynamicObject leaves Active true and the
// first toggle switches *off*. That is the observation, and it is the whole of what the
// dispatch has to get right: the arm has to reach the Toggle* for its own type with the
// object's DynaNum, and any of the fourteen would flip this flag -- so this proves the
// dispatch happened, while trip_test.go proves each of the fourteen writes the right fields.
func dinahWentIdle(t *testing.T, w *World) {
	t.Helper()
	if w.NumDynamics != 1 {
		t.Fatalf("NumDynamics = %d, want 1: the fixture did not register the target",
			w.NumDynamics)
	}
	if w.Dinahs[0].Active {
		t.Error("Dinahs[0].Active = true: the switch did not reach this type's Toggle*, " +
			"or reached it with the wrong index")
	}
}

// TestTwentyThreeSwitchArms is the table.
func TestTwentyThreeSwitchArms(t *testing.T) {
	if len(theSwitchArms) != 24 {
		t.Fatalf("%d rows, want 24: twenty-three arms, with kInvisBonus and kSlider split",
			len(theSwitchArms))
	}

	for _, arm := range theSwitchArms {
		for _, what := range arm.types {
			t.Run(arm.name+"/"+house.ObjectName(what), func(t *testing.T) {
				w, who := switchWorld(t, LightSwitch, arm.obj(what), Toggle, 1, arm.dinah)
				w.R.Pendulums = []render.Anim{{Where: 0, Who: 1, SavedMap: 1}}
				w.R.Stars = []render.Anim{{Where: 0, Who: 1, SavedMap: 1}}
				before := w.Room(0).Objects[1].Data

				var log soundLog
				log.install(w)

				w.HandleSwitches(who)

				if !who.StillOver {
					t.Error("who.StillOver = false: the latch is written unconditionally, " +
						"so holding the glider against a switch flips it once")
				}
				// The state write, which is what "reached" is measured by on every row but
				// the stereo's -- and the stereo is exactly why it is measured rather than
				// assumed from the sounds.
				wantWrite := arm.reached && !arm.noStateWrite
				if got := w.Room(0).Objects[1].Data != before; got != wantWrite {
					t.Errorf("target payload changed = %v, want %v: SetObjectState %s "+
						"have written %s's state byte", got, wantWrite,
						map[bool]string{true: "should", false: "should not"}[wantWrite],
						house.ObjectName(what))
				}
				if got := len(w.Work2Main); got != arm.workRects {
					t.Errorf("%d work rects, want %d", got, arm.workRects)
				}
				log.is(t, arm.sounds...)
				if int(w.NumSparkles) != arm.sparkles {
					t.Errorf("NumSparkles = %d, want %d", w.NumSparkles, arm.sparkles)
				}
				if arm.check != nil {
					arm.check(t, w)
				}
			})
		}
	}
}

// TestEveryStatefulTypeIsEitherHandledOrDeliberatelyNot is the structural guard, and it is
// the switch path's version of TestEveryDispatchableRewardIsConsumed -- the test that found
// the missing kHelium reward arm.
//
// The risk it covers is the one that bug was an instance of: two halves of the same feature
// drifting apart. SetObjectState decides which types have a state a switch can change;
// switchLinkedObject decides what has to be redrawn when one does. A type in the first and not
// the second changes silently and leaves its old art on screen, which looks like a rendering
// bug and is a dispatch bug.
//
// Rather than carry a list of types, it measures the first half -- sweeping every object code
// through SetObjectState -- and then requires every type it finds to be accounted for. Three
// answers count as accounted for, and the second and third are claims about the C worth
// stating once, here, where a reader who notices the gap will come looking:
//
// **The blowers have no arm and need none.** A blower's presence on screen does not depend on
// its state: the vent and the fan are drawn by the room composition, and the air column is not
// drawn at all. So switching one off changes nothing that has to be repainted, and the silent
// absence is correct. Eleven of the sixteen data.a types reach this test at all -- the five
// flames are among the types SetObjectState refuses outright, so a switch cannot blow a candle
// out.
//
// **kSparkle has no arm either, and that is the same argument.** A sparkle emitter draws
// nothing permanent -- the puffs are transient effects re-emitted every frame -- so there is no
// background to put back. It is the only one of the fifteen bonus types with no arm at all,
// which sits oddly beside kInvisBonus: the two are in the same position and the C gives one an
// explicit empty case and the other nothing. Same behaviour, differently spelt.
func TestEveryStatefulTypeIsEitherHandledOrDeliberatelyNot(t *testing.T) {
	handled := map[int16]bool{}
	for _, arm := range theSwitchArms {
		for _, what := range arm.types {
			if handled[what] {
				t.Errorf("%s appears in two rows of theSwitchArms: the arms are disjoint in "+
					"the C, so a duplicate means one row is asserting the wrong arm",
					house.ObjectName(what))
			}
			handled[what] = true
		}
	}

	// All sixteen of data.a, kLiftArea included -- it is the last member of the blower union
	// and carries a state byte like the rest, however little it looks like a fan.
	noRedraw := map[int16]bool{}
	for _, what := range []int16{FloorVent, CeilingVent, FloorBlower, CeilingBlower, SewerGrate,
		LeftFan, RightFan, Taper, Candle, Stubby, Tiki, BBQ, InvisBlower, GrecoVent,
		SewerBlower, LiftArea,
		// And the one bonus that draws nothing to put back.
		Sparkle} {
		noRedraw[what] = true
	}

	// One world, reused: SetObjectState reads the object's type and its payload and nothing
	// about the room, so a fresh fixture per type would assert the same thing more slowly.
	w := dynaWorld(t)
	w.R.RoomNumber = 0

	stateful, inTable, excused, unhandled := 0, 0, 0, 0
	for what := int16(1); what <= 0x8F; what++ {
		// Every payload shape in turn. A type whose state lives in a union member this
		// fixture did not build would read a zero and could report no change, so the type
		// counts as stateful if *any* shape makes SetObjectState return true.
		changed := false
		for _, build := range []func(int16) house.Object{prizeObj, lightObj, transportObj,
			applianceTarget, enemyTarget, plainObj} {
			w.Room(0).Objects[1] = build(what)
			if w.SetObjectState(0, 1, ForceOn, -1) {
				changed = true
				break
			}
		}
		if !changed {
			continue
		}
		stateful++
		switch {
		case handled[what]:
			inTable++
		case noRedraw[what]:
			excused++
		default:
			unhandled++
			t.Errorf("%s (0x%02X) has a state a switch can change and no arm in "+
				"theSwitchArms: either switchLinkedObject grew a case the table does not "+
				"cover, or a type gained a state byte without gaining the redraw that has "+
				"to go with it", house.ObjectName(what), what)
		}
	}

	t.Logf("%d object types have a switchable state: %d covered by theSwitchArms, %d with "+
		"nothing to redraw, %d unaccounted for", stateful, inTable, excused, unhandled)
	// The other direction. Three of the table's rows are for types SetObjectState refuses, so
	// the table is expected to be exactly three wider than the stateful set it covers.
	if want := len(handled) - 3; inTable != want {
		t.Errorf("%d of the table's %d types are stateful, want %d: the three unreachable "+
			"arms are kSlider, kSoundTrigger and kGuitar, and a fourth means "+
			"TestThreeSwitchArmsCannotBeReached is now understating the dead code",
			inTable, len(handled), want)
	}

	if stateful == 0 {
		t.Fatal("no type reported a state change: the sweep is broken, so the accounting " +
			"above proves nothing")
	}
}

// TestThreeSwitchArmsCannotBeReached states the dead code as a fact rather than a comment.
//
// All three arms are in switches.go because the original has them, and all three are
// unreachable for the same reason: SetObjectState returns false for a slider, for a sound
// trigger and for a guitar, and the second dispatch runs only when it returned true. The
// author clearly meant a switched sound trigger to play its sound and a switched guitar to
// strum; neither ever has.
//
// Pinned because the coupling is at a distance. Someone adding a state byte to the guitar --
// a reasonable thing to want, and one line in setstate.go -- would bring an untested arm to
// life, and this is the test that says so.
func TestThreeSwitchArmsCannotBeReached(t *testing.T) {
	for _, c := range []struct {
		what int16
		obj  func(int16) house.Object
		why  string
	}{
		{Slider, prizeObj, "a bare break in SetObjectState's bonus family"},
		{SoundTrigger, plainObj, "grouped with the switches, which have no state of their own"},
		{Guitar, applianceTarget, "\"really no point to change this state\""},
	} {
		t.Run(house.ObjectName(c.what), func(t *testing.T) {
			w := dynaWorld(t)
			w.R.RoomNumber = 0
			w.Room(0).Objects[1] = c.obj(c.what)
			if w.SetObjectState(0, 1, Toggle, -1) {
				t.Errorf("SetObjectState reported a change for %s, so switchLinkedObject's "+
					"arm is now live and untested (%s)", house.ObjectName(c.what), c.why)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// The latch
// ---------------------------------------------------------------------------

// TestSwitchLatchesOnceEvenWhenNothingHappens is the StillOver guard, which is read at the
// top and written at the bottom with nothing between them that can skip the write.
//
// The unconditional write is the part worth pinning: a switch wired to a table changes
// nothing, and still latches -- so the glider has to step off and back on to try again. A
// port that wrote the flag inside the `if` would give a table-wired switch a
// sixty-times-a-second retry loop, which is invisible until something in the chain starts
// making a sound.
func TestSwitchLatchesOnceEvenWhenNothingHappens(t *testing.T) {
	for _, c := range []struct {
		name    string
		target  house.Object
		reached bool
	}{
		{"a lamp, which the switch can change", lightObj(TableLamp), true},
		{"a table, which it cannot", plainObj(Table), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, who := switchWorld(t, LightSwitch, c.target, Toggle, 1, false)
			var log soundLog
			log.install(w)

			w.HandleSwitches(who)
			if !who.StillOver {
				t.Fatal("StillOver = false after the first throw")
			}
			firstRects := len(w.Work2Main)
			firstSounds := len(log.played)

			// A second frame with the glider still touching it.
			w.HandleSwitches(who)
			if len(w.Work2Main) != firstRects || len(log.played) != firstSounds {
				t.Errorf("the second frame did something: rects %d -> %d, sounds %d -> %d",
					firstRects, len(w.Work2Main), firstSounds, len(log.played))
			}

			// Stepping off and back on. CheckForHotSpots clears StillOver when the
			// glider leaves the rect; a light switch is meant to be flippable twice.
			who.StillOver = false
			w.HandleSwitches(who)
			if c.reached && len(w.Work2Main) == firstRects {
				t.Error("the switch did nothing on a fresh step-on")
			}
		})
	}
}

// TestBadMasterIndexStillLatches pins where the port put its bounds check.
//
// The C indexes masterObjects[who->who] unguarded; the port guards, and the guard is
// *inside* the latch rather than around it. That ordering is the whole content of the
// decision: outside it, an unresolvable switch would be re-dispatched on every frame the
// glider touched it, forever.
func TestBadMasterIndexStillLatches(t *testing.T) {
	for _, bad := range []int16{-1, 2, 999} {
		w, who := switchWorld(t, LightSwitch, lightObj(TableLamp), Toggle, 1, false)
		var log soundLog
		log.install(w)
		who.Who = bad

		w.HandleSwitches(who)

		log.is(t)
		if len(w.Work2Main) != 0 {
			t.Errorf("who.Who = %d: %d work rects, want 0", bad, len(w.Work2Main))
		}
		if !who.StillOver {
			t.Errorf("who.Who = %d: StillOver = false, so this switch would be "+
				"re-dispatched every frame", bad)
		}
	}
}

// ---------------------------------------------------------------------------
// The lever
// ---------------------------------------------------------------------------

// TestLeverStateComesFromTheLinkedObject is the art-free half of this stage's regression
// test.
//
// SetObjectState computes the state it writes into World.NewState -- a file-scope global in
// the C -- and HandleSwitches reads it back to draw the plate. So the lever shows the
// *lamp's* new state, not the switch's own, and a switch has no state of its own to show:
// SetObjectState returns false for all five plate types.
//
// A port that made newState a local would draw every lever from a zero value, so a light
// switch would animate to "off" while turning a lamp on. This asserts the field, which is
// the only observable without art; TestLeverShowsTheLinkedObjectsState asserts the pixels.
func TestLeverStateComesFromTheLinkedObject(t *testing.T) {
	for _, c := range []struct {
		name  string
		start byte
		want  bool
	}{
		{"an unlit lamp switches on", 0, true},
		{"a lit lamp switches off", 1, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			lamp := lightObj(TableLamp)
			lamp.Data[offLightState] = c.start
			w, who := switchWorld(t, LightSwitch, lamp, Toggle, 1, false)

			// A value neither outcome can produce by accident, so "was written" and
			// "happened to already be right" are distinguishable.
			w.NewState = !c.want

			w.HandleSwitches(who)

			if w.NewState != c.want {
				t.Errorf("World.NewState = %v, want %v: this is what DrawSwitch is handed, "+
					"and it has to be the lamp's state and not the plate's",
					w.NewState, c.want)
			}
			if got := w.Room(0).Objects[1].Data[offLightState] != 0; got != c.want {
				t.Errorf("the lamp's own state byte = %v, want %v", got, c.want)
			}
			if w.Room(0).Objects[0].Data[offLightState] != 0 {
				t.Error("the plate's payload was written: a switch has no state, and " +
					"SetObjectState's switch family returns false without touching one")
			}
		})
	}
}

// TestLeverShowsTheLinkedObjectsState is the pixel half, and it needs the extracted art
// because DrawSwitch is a blit out of the switch sheet.
//
// The two cases differ only in the *lamp's* starting state; the plate is identical in both.
// So if the two plates come out looking the same, the lever is not being drawn from the
// linked object -- which is exactly the bug this stage fixed, and the only way to see it is
// to look at the pixels.
func TestLeverShowsTheLinkedObjectsState(t *testing.T) {
	artDir := requireAssets(t, "art")

	plate := make([][]uint8, 2)
	for i, start := range []byte{0, 1} {
		lamp := lightObj(TableLamp)
		lamp.Data[offLightState] = start

		w := newTestWorld(oneRoomHouse(house.ObjectIsEmpty), artDir, "")
		w.InitGarbageRects()
		w.P1.Dest = player.Rect{Top: -4000, Left: -4000, Bottom: -3980, Right: -3960}
		w.P2.Dest = w.P1.Dest
		w.R.RoomNumber = 0
		w.R.NumLights = 0
		who := installSwitchGraph(t, w, LightSwitch, lamp)

		w.HandleSwitches(who)

		plate[i] = backPixels(w, render.Offset(switchAt, w.R.V.OriginH, w.R.V.OriginV))
	}

	if bytes.Equal(plate[0], plate[1]) {
		t.Error("the plate looks identical whether the lamp went on or off: the lever is " +
			"not being drawn from World.NewState, so it shows the switch's own (absent) " +
			"state instead of the linked object's")
	}
	if allSame(plate[0]) {
		t.Error("the plate is a flat colour: DrawSwitch found no art, so the comparison " +
			"above proves nothing")
	}
}

// installSwitchGraph is switchWorld's graph-building half, for the one test that needs a
// world with art rather than dynaWorld's.
func installSwitchGraph(t *testing.T, w *World, switchWhat int16, target house.Object) *HotObject {
	t.Helper()
	plate := house.Object{What: switchWhat}
	plate.SetSwitch(house.Switch{
		TopLeft: house.Point{V: switchAt.Top, H: switchAt.Left}, Where: 0, Who: 1,
		Type: byte(Toggle),
	})
	w.Room(0).Objects[0] = plate
	w.Room(0).Objects[1] = target
	w.Room(0).NumObjects = 2
	w.R.Master = []MasterObject{{
		RoomNum: 0, ObjectNum: 0, RoomLink: 0, ObjectLink: 1, LocalLink: 1,
		HotNum: 0, DynaNum: 0, TheObject: plate,
	}, {
		RoomNum: 0, ObjectNum: 1, RoomLink: -1, ObjectLink: -1, LocalLink: -1,
		HotNum: 1, DynaNum: -1, TheObject: target,
	}}
	w.R.Hot = []HotObject{
		{Bounds: switchAt, Action: SwitchIt, Who: 0, IsOn: true},
		{Bounds: prizeAt, Action: RewardIt, Who: 1, IsOn: true},
	}
	clearRects(w)
	return &w.R.Hot[0]
}

// backPixels reads a rect out of the *background* map, which is where DrawSwitch draws: a
// thrown lever is a permanent change to the room, not something the next frame's
// back->work restore is allowed to undo.
func backPixels(w *World, r Rect) []uint8 {
	out := make([]uint8, 0, int(r.Wide())*int(r.Tall()))
	for y := r.Top; y < r.Bottom; y++ {
		for x := r.Left; x < r.Right; x++ {
			out = append(out, at(w.R.Back, x, y))
		}
	}
	return out
}

func allSame(pix []uint8) bool {
	for _, p := range pix {
		if p != pix[0] {
			return false
		}
	}
	return true
}

// TestInvisibleSwitchIsSilentAndStillPaysForTwoBlits is the sixth plate case, which is a
// bare break in the C.
//
// The sound is inside the five drawing arms rather than above them, so an invisible switch
// is silent as well as invisible. But the two blits below the inner switch are outside it,
// so an invisible switch still promotes an unchanged rect from the background into the work
// map and still spends one of the 47 dirty-rect slots publishing it. That is the original's
// behaviour and it is docs/IMPROVEMENTS.md 2.11.
func TestInvisibleSwitchIsSilentAndStillPaysForTwoBlits(t *testing.T) {
	w, who := switchWorld(t, InvisSwitch, transportObj(DeluxeTrans), Toggle, 1, false)
	var log soundLog
	log.install(w)

	w.HandleSwitches(who)

	log.is(t)
	if len(w.Work2Main) != 1 {
		t.Errorf("%d work rects, want 1: the blits are outside the inner switch, so an "+
			"invisible switch pays for them too", len(w.Work2Main))
	}
	if w.Room(0).Objects[1].Data[offTransportWide]&0x0F == 0 {
		t.Error("the target was not switched: invisible is not inert")
	}
}

// TestKnifeSwitchThrowsButItsOwnStateNeverChanges is the one type outside the state machine
// in both directions.
//
// kKnifeSwitch is missing from SetObjectState's switch family and from GetObjectState's, but
// it *is* in CreateActiveRects' list and it *is* in HandleSwitches' drawing arm. So it can be
// thrown, it clicks, it draws a lever, and nothing anywhere records which way it is pointing
// -- so a knife switch reverts to its authored appearance the next time the room is composed.
func TestKnifeSwitchThrowsButItsOwnStateNeverChanges(t *testing.T) {
	w, who := switchWorld(t, KnifeSwitch, lightObj(TableLamp), Toggle, 1, false)
	var log soundLog
	log.install(w)
	before := w.Room(0).Objects[0].Data

	w.HandleSwitches(who)

	log.is(t, SwitchSound)
	if w.Room(0).Objects[0].Data != before {
		t.Error("the knife switch's own payload changed: it has no state byte anywhere in " +
			"the state machine")
	}
	if w.Room(0).Objects[1].Data[offLightState] == 0 {
		t.Error("the lamp was not switched on: a knife switch works, it just does not " +
			"remember")
	}
}

// TestForceOnAndForceOffAreTheAuthorsChoice pins data.e.type as the third argument to
// SetObjectState.
//
// It is what makes a "switch" able to be a one-way switch: an author who sets ForceOn gets a
// plate that turns a lamp on and then reports no change forever after, so the second throw is
// silent and draws nothing. Worth a test because the value comes out of the plate's payload
// and every other caller of SetObjectState passes a literal.
func TestForceOnAndForceOffAreTheAuthorsChoice(t *testing.T) {
	for _, c := range []struct {
		name         string
		action       int16
		start        byte
		wantState    bool
		wantReported bool
	}{
		{"Toggle on", Toggle, 0, true, true},
		{"Toggle off", Toggle, 1, false, true},
		{"ForceOn from off", ForceOn, 0, true, true},
		{"ForceOn from on", ForceOn, 1, true, false},
		{"ForceOff from on", ForceOff, 1, false, true},
		{"ForceOff from off", ForceOff, 0, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			lamp := lightObj(TableLamp)
			lamp.Data[offLightState] = c.start
			w, who := switchWorld(t, LightSwitch, lamp, c.action, 1, false)
			var log soundLog
			log.install(w)

			w.HandleSwitches(who)

			if got := w.Room(0).Objects[1].Data[offLightState] != 0; got != c.wantState {
				t.Errorf("the lamp is %v, want %v", got, c.wantState)
			}
			// No change reported means the whole body is skipped: no click, no lever, no
			// rects, and no second dispatch.
			if got := len(log.played) > 0; got != c.wantReported {
				t.Errorf("clicked = %v, want %v", got, c.wantReported)
			}
			if got := len(w.Work2Main) > 0; got != c.wantReported {
				t.Errorf("%d work rects, want %v", len(w.Work2Main), c.wantReported)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// The local/remote split
// ---------------------------------------------------------------------------

// TestSwitchToARoomOutsideTheLocaleWritesTheHouseAndNothingElse is the `linkIndex != -1`
// half of the function, and it is the same split FireTrigger has.
//
// A switch wired to a room the player cannot see writes the house copy, clicks, draws its
// own lever, and stops: the second dispatch needs a master index and there isn't one. So the
// prize at the other end is gone -- the state byte says so -- but nothing erased it, nothing
// sparkled, and the player finds the change on arrival.
//
// This is what makes a house able to change a room the player has not reached, and it is why
// SetObjectState's `local` argument has to be optional.
//
// The fixture is artificial in one way worth naming: its target is in the central room *and*
// has LocalLink -1, which a real house cannot produce -- an object in the locale always has a
// master index. That is deliberate, because it is the only way to watch the -1 suppress the
// publication rather than merely have nothing to publish to. The last assertion below is the
// one that depends on the artifice.
func TestSwitchToARoomOutsideTheLocaleWritesTheHouseAndNothingElse(t *testing.T) {
	w, who := switchWorld(t, LightSwitch, prizeObj(RedClock), Toggle, -1, false)
	w.R.Stars = []render.Anim{{Where: 0, Who: 1, SavedMap: 1}}
	var log soundLog
	log.install(w)

	w.HandleSwitches(who)

	if w.Room(0).Objects[1].Bonus().State != 0 {
		t.Error("the prize's state byte was not cleared: the house is written either way")
	}
	// The click and the lever, and nothing from switchLinkedObject.
	log.is(t, SwitchSound)
	if len(w.Work2Main) != 1 {
		t.Errorf("%d work rects, want 1: the lever's, with no restore", len(w.Work2Main))
	}
	if w.NumSparkles != 0 {
		t.Errorf("NumSparkles = %d, want 0: nobody is there to see it", w.NumSparkles)
	}
	// And the publication did not happen: `local == -1` is SetObjectState's "write the house
	// and tell nobody", so the live hot spot is left exactly as it was. In a real house that
	// is invisible -- an object outside the locale has no hot spot to leave alone -- and this
	// fixture's target does have one, which is what makes the suppression observable.
	if !w.R.Hot[1].IsOn {
		t.Error("the target's hot spot was cleared: local == -1 must suppress every live " +
			"update, because the indices it would need are the ones it does not have")
	}
}

// TestSwitchWithAnOutOfRangeLinkStopsBeforeTheSecondDispatch. LocalLink comes off disk in a
// house file and nothing validates it on load, so a hand-edited or truncated house can name
// a master index that does not exist. The C would index past the table; the port refuses,
// and the refusal has to land *after* the lever, not instead of it.
func TestSwitchWithAnOutOfRangeLinkStopsBeforeTheSecondDispatch(t *testing.T) {
	w, who := switchWorld(t, LightSwitch, prizeObj(RedClock), Toggle, 99, false)
	var log soundLog
	log.install(w)

	w.HandleSwitches(who)

	log.is(t, SwitchSound)
	if len(w.Work2Main) != 1 {
		t.Errorf("%d work rects, want 1: the lever still draws", len(w.Work2Main))
	}
	if w.NumSparkles != 0 {
		t.Errorf("NumSparkles = %d, want 0", w.NumSparkles)
	}
}

// ---------------------------------------------------------------------------
// The two reproduced bugs
// ---------------------------------------------------------------------------

// TestSwitchOnAStarStrandsTheHouse is the more serious of the two, and the reason it is
// pinned rather than quietly fixed is that fixing it changes what a shipped house does.
//
// The star is grouped with the eight other prizes in switchLinkedObject, so a switch-removed
// star gets the restore and the sparkle and nothing else: no StopStar, so its six-cel spin
// keeps animating over the background that was just put back, and no decrement of the star
// count. The object is gone, so it can never be collected, and nothing else in the game
// decrements that count -- the house is unwinnable from the moment the switch is thrown.
//
// **An author can therefore build a house that cannot be won**, and the editor gives no
// warning. That is a release-quality concern for Stage 2's new houses and Stage 5's editor;
// see docs/IMPROVEMENTS.md.
func TestSwitchOnAStarStrandsTheHouse(t *testing.T) {
	w, who := switchWorld(t, LightSwitch, prizeObj(Star), Toggle, 1, false)
	w.R.Stars = []render.Anim{{Where: 0, Who: 1, SavedMap: 1}}
	w.StarsLeft = 1

	w.HandleSwitches(who)

	if w.Room(0).Objects[1].Bonus().State != 0 {
		t.Fatal("the star was not removed, so this test is not testing anything")
	}
	if w.StarsLeft != 1 {
		t.Errorf("StarsLeft = %d, want 1 (unchanged): the switch path does not decrement "+
			"it. If this now reads 0 the bug has been fixed -- update the comments in "+
			"switches.go and docs/IMPROVEMENTS.md rather than this line alone", w.StarsLeft)
	}
	if w.GameOver {
		t.Error("GameOver = true: nothing in the switch path can flag it")
	}
	if w.R.Stars[0].Stopped {
		t.Error("Stars[0].Stopped = true: the switch path has no StopStar either. If this " +
			"is now false the bug has been fixed; see above")
	}
	// The half that says "unwinnable" rather than merely "wrong": the object is gone, so
	// HandleRewards can never run for it, so the count can never reach zero.
	w.R.Hot[1].IsOn = true
	before := w.StarsLeft
	w.HandleRewards(&w.P1, &w.R.Hot[1])
	if w.StarsLeft != before {
		t.Errorf("StarsLeft = %d after trying to collect the removed star, want %d: "+
			"if the reward path can still take it, the house is not stranded after all",
			w.StarsLeft, before)
	}
}

// TestTheSpuriousSparkleLandsAtTheCorner is the other reproduced bug, and the harmless one.
//
// `bounds` is declared in HandleSwitches and never assigned before the prize arm's
// AddSparkle reads it. On 68k that was whatever was on the stack; in Go it is the zero rect,
// so the puff lands at the top-left corner of the play area rather than somewhere random.
//
// The sparkle the player is *meant* to see comes from RestoreFromSavedMap's own doSparkle
// arm a line earlier and is correctly placed on the prize, which is what makes this one pure
// noise -- and cheap to remove later behind a "modern" option, since nothing depends on it.
func TestTheSpuriousSparkleLandsAtTheCorner(t *testing.T) {
	w, who := switchWorld(t, LightSwitch, prizeObj(RedClock), Toggle, 1, false)

	w.HandleSwitches(who)

	if w.NumSparkles != 2 {
		t.Fatalf("NumSparkles = %d, want 2: the restore's and the spurious one",
			w.NumSparkles)
	}

	// Slot 0 is the restore's, on the prize; slot 1 is the spurious one, on the zero rect
	// offset by the play origin. The sprite is centred, so the second sits a little above
	// and left of the origin itself.
	onPrize, spurious := w.Sparkles[0].Bounds, w.Sparkles[1].Bounds
	prizeOnScreen := render.Offset(prizeAt, w.R.V.OriginH, w.R.V.OriginV)
	if onPrize.Left < prizeOnScreen.Left || onPrize.Left >= prizeOnScreen.Right {
		t.Errorf("Sparkles[0] at %+v is not over the prize at %+v: the puff the player is "+
			"meant to see has moved", onPrize, prizeOnScreen)
	}
	if spurious.Left >= w.R.V.OriginH || spurious.Top >= w.R.V.OriginV {
		t.Errorf("Sparkles[1] at %+v is not at the play origin (%d,%d): the reproduced "+
			"glitch draws from an unassigned rect, so it must land at Go's zero",
			spurious, w.R.V.OriginH, w.R.V.OriginV)
	}
}

// ---------------------------------------------------------------------------
// The corpus
// ---------------------------------------------------------------------------

// TestShippedHousesWireSwitchesToPrizes is the measurement that decides how much the two
// reproduced bugs above matter, and it is the answer this stage needed before recording them.
//
// A *switch* wired to a bonus is the only way to reach either -- both live in
// switchLinkedObject's prize group -- and a trigger wired to one reaches neither, because
// FireTrigger has no prize arm at all: a trigger wired to a red clock arms, counts down,
// fires and does nothing. So the two families are counted separately and only the switch
// figure is the one that matters.
//
// **The answer is 145 switch links into the prize arm, in the houses John Calhoun shipped.**
// The corner sparkle is therefore not a latent curiosity: it is something a player of the
// original content sees. That is what promotes it from a footnote to an entry in
// docs/IMPROVEMENTS.md with a fix behind the modern-options flag.
//
// The star answer is the other way round -- zero, in 22 houses -- so no shipped house is
// stranded and the missing decrement is a hazard for Stage 2's new houses and Stage 5's
// editor rather than a defect in the originals.
//
// Counts are logged rather than asserted, because the corpus is the evidence and a number
// baked into an assertion would just have to be updated. Two things do fail: a switch wired
// to a star, which would mean a shipped house cannot be completed, and a predicate that
// resolves nothing, which is the failure mode this test had in its first version.
func TestShippedHousesWireSwitchesToPrizes(t *testing.T) {
	houseDir := requireAssets(t, "houses")
	paths, _ := filepath.Glob(filepath.Join(houseDir, "*.house"))
	if len(paths) == 0 {
		t.Skipf("no houses in %s", houseDir)
	}
	sort.Strings(paths)

	// All fifteen bonus types, so that the log says which ones authors actually wire and not
	// merely how many. kSparkle and kSlider are in the list and have no reachable arm, which
	// is worth counting for the same reason.
	prizes := map[int16]bool{}
	for _, what := range []int16{RedClock, BlueClock, YellowClock, Cuckoo, Paper, Battery,
		Bands, GreaseRt, GreaseLf, Foil, InvisBonus, Star, Sparkle, Helium, Slider} {
		prizes[what] = true
	}
	// The nine bonus types in switchLinkedObject's first arm: the only ones that reach the
	// restore and the spurious sparkle.
	inPrizeArm := map[int16]bool{}
	for _, what := range []int16{RedClock, BlueClock, YellowClock, Paper, Battery, Bands,
		Foil, Star, Helium} {
		inPrizeArm[what] = true
	}
	// The six plates HandleSwitches serves, and the two pressure plates it does not. Both are
	// walked because both are link-carrying objects an author can wire to a prize, and the
	// point of splitting them is that only one family has an arm for it.
	plate := map[int16]bool{}
	for _, what := range []int16{LightSwitch, MachineSwitch, Thermostat, PowerSwitch,
		KnifeSwitch, InvisSwitch} {
		plate[what] = true
	}
	pressure := map[int16]bool{Trigger: true, LgTrigger: true}

	found := map[int16]int{}
	plates, wired, links, viaSwitch, sparkling, stars := 0, 0, 0, 0, 0, 0
	for _, path := range paths {
		h, err := house.LoadFile(path)
		if err != nil {
			continue // not a loadable house; the corpus tests own that complaint
		}
		name := strings.TrimSuffix(filepath.Base(path), ".house")
		// The link has to be resolved the way the game resolves it. data.e.where is a
		// *packed floor/suite pair*, not a room index -- 6709 is floor 26 suite 53, not room
		// 6709 -- so reading it as an index finds nothing and would have made this survey
		// silently report a clean corpus. GetRoomLinked is the real arithmetic, and using it
		// is also what makes this test track any future change to it.
		w := newTestWorld(h, "", "")
		for r := range h.Rooms {
			rm := &h.Rooms[r]
			for o := 0; o < int(rm.NumObjects) && o < len(rm.Objects); o++ {
				obj := rm.Objects[o]
				if !plate[obj.What] && !pressure[obj.What] {
					continue
				}
				plates++
				where, who := w.GetRoomLinked(obj), w.GetObjectLinked(obj)
				if where < 0 || int(where) >= len(h.Rooms) || who < 0 {
					continue // unlinked, or pointing at a room this house does not contain
				}
				dst := &h.Rooms[where]
				if int(who) >= int(dst.NumObjects) || int(who) >= len(dst.Objects) {
					continue
				}
				wired++
				target := dst.Objects[who]
				if !prizes[target.What] {
					continue
				}
				links++
				found[target.What]++
				if !plate[obj.What] {
					continue // a trigger: no arm for this, so nothing to count
				}
				viaSwitch++
				if inPrizeArm[target.What] {
					sparkling++
				}
				if target.What == Star {
					stars++
					t.Errorf("%s room %d object %d is a %s wired to a star (room %d "+
						"object %d): throwing it removes the star without decrementing "+
						"the count, so the house can no longer be finished",
						name, r, o, house.ObjectName(obj.What), where, who)
				}
			}
		}
	}

	whats := make([]int16, 0, len(found))
	for what := range found {
		whats = append(whats, what)
	}
	sort.Slice(whats, func(i, j int) bool { return whats[i] < whats[j] })
	for _, what := range whats {
		t.Logf("%4d link(s) to a %s", found[what], house.ObjectName(what))
	}
	t.Logf("%d plates in %d houses; %d wired to a resolvable object; %d of those to a bonus, "+
		"%d of them from a switch rather than a pressure plate",
		plates, len(paths), wired, links, viaSwitch)

	// The guard that keeps the survey honest. A predicate that resolves nothing would report
	// "no shipped house does this" just as loudly as a corpus that really does not, and the
	// first version of this test did exactly that: it read data.e.where as a room index when
	// it is a packed floor/suite pair, resolved nothing, and concluded the corpus was clean.
	if wired == 0 {
		t.Fatalf("%d plates and not one resolvable link: the predicate is broken, so the "+
			"conclusions below would be meaningless", plates)
	}

	if sparkling == 0 {
		t.Log("no shipped house wires a switch into the prize arm, so the corner sparkle is " +
			"latent in the original content")
	} else {
		t.Logf("%d switch links reach the prize arm, so the spurious corner sparkle fires "+
			"%d different times across the original houses: it is visible in shipped "+
			"content, not merely reachable", sparkling, sparkling)
	}
	if stars == 0 {
		t.Log("and none wires one to a star, so no shipped house is stranded -- the missing " +
			"StarsLeft decrement is a hazard for new houses and for the editor, not a " +
			"defect a player of the originals can hit")
	}
}
