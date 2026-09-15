package game

// The dinahs table: registration, the eighteen-slot cap, one frame of each of the
// seventeen registrable types' handlers, and the census that proves the cap is reachable in
// shipped content.
//
// Almost none of it needs a house or art. A registration is a pure function of the type code,
// a room-local rect, ten bytes of object payload and the play origin; a handler frame is a
// pure function of the slot. Where art is genuinely the subject -- the blitter the fish
// switches -- the test lives in movers_test.go and skips without the extracted tree.

import (
	"path/filepath"
	"sort"
	"testing"

	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/render"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// dynaWorld is a world on an empty one-room house with no art: enough for
// AddDynamicObject, every handler, and every renderer's rect bookkeeping.
//
// No art is a feature rather than a limitation. Every renderer in dynamics_render.go is
// written so that a nil sheet still registers both dirty rects -- a missing art file should
// leave a hole, not desynchronise the lists -- so a world with no art exercises exactly the
// bookkeeping most of these tests are about and nothing else.
//
// The view is the default 640x480, whose play origin is (64, 79). Non-zero on both axes on
// purpose: that origin is what separates the ten movers' room-local Dest from the seven
// appliances' screen-space Dest, and a fixture with a zero origin would pass either way.
// The glider is parked far outside the room so that no handler's checkGliders can fire;
// collisions are dynacollide_test.go's subject, not this file's.
func dynaWorld(t *testing.T) *World {
	t.Helper()
	w := newTestWorld(oneRoomHouse(house.ObjectIsEmpty), "", "")
	if w.R.V.OriginH == 0 || w.R.V.OriginV == 0 {
		t.Fatalf("fixture play origin is (%d,%d); it has to be non-zero on both axes or the "+
			"room-local/screen-space distinction is untested", w.R.V.OriginH, w.R.V.OriginV)
	}
	w.P1.Dest = player.Rect{Top: -4000, Left: -4000, Bottom: -3980, Right: -3960}
	w.P2.Dest = w.P1.Dest

	// The last of ReadyLevel's five steps, and it is not optional here: it is the only
	// sweeper that writes Mode = -1 across the two effects tables, and Mode == -1 is what
	// makes a slot reusable. Without it a bare world reads as three live sparkles and
	// AddSparkle silently finds nowhere to put a fourth -- which is exactly the failure
	// HandleSparkleObject's puff would show up as.
	w.InitGarbageRects()
	return w
}

// enemyObj builds an enemyType payload: Length is the height or fall distance, Delay the
// reload period in units of two frames.
func enemyObj(what int16, length int16, delay byte) house.Object {
	o := house.Object{What: what}
	o.SetEnemy(house.Enemy{Length: length, Delay: delay, Initial: 1, State: 1})
	return o
}

// applianceObj builds an applianceType payload. The field *order* differs from enemyType's
// -- Height then Byte0 then Delay, against Length then Delay then Byte0 -- which is the only
// thing keeping a toaster's reload period from being a balloon's. See
// TestFishReadsTheSameBytesAsTheEnemyLength.
func applianceObj(what int16, height int16, delay byte) house.Object {
	o := house.Object{What: what}
	o.SetAppliance(house.Appliance{Height: height, Delay: delay, Initial: 1, State: 1})
	return o
}

// place returns a rect the size of src with its top-left corner at (left, top).
//
// Every expected rect in this file is built with it rather than by replaying the
// Offset/ZeroCorner/CenterIn composition the source uses. That is the point: re-invoking the
// same helpers in the same order would pass even if the composition were wrong, where
// spelling the arithmetic out independently pins the *numbers* -- the +10,+7 and +17,+10 and
// +56,+20 the author measured off his own art.
func place(src Rect, left, top int16) Rect {
	return Rect{Top: top, Left: left, Bottom: top + src.Tall(), Right: left + src.Wide()}
}

// centredAt centres a sprite-sized rect horizontally in `in` and hangs its top on in.Top,
// which is what the toaster's and the drip's registrations do.
func centredAt(src, in Rect) Rect {
	return place(src, in.Left+(in.Wide()-src.Wide())/2, in.Top)
}

// soundLog records PlayPrioritySound calls in order. Several traps in this slice are about
// *which* sound and in what order -- a fish plays DropSound then FishInSound on one frame --
// so the order matters as much as the count.
type soundLog struct{ played []int16 }

func (s *soundLog) install(w *World) {
	w.SoundPlayer = func(sound, priority int16) { s.played = append(s.played, sound) }
}

func (s *soundLog) is(t *testing.T, want ...int16) {
	t.Helper()
	if len(s.played) != len(want) {
		t.Fatalf("sounds %v, want %v", s.played, want)
	}
	for i := range want {
		if s.played[i] != want[i] {
			t.Fatalf("sounds %v, want %v", s.played, want)
		}
	}
}

// rectCounts returns the two dirty-rect list lengths, which is what most of the appliance
// and renderer tests assert on.
func rectCounts(w *World) (work, back int) { return len(w.Work2Main), len(w.Back2Work) }

func clearRects(w *World) {
	w.Work2Main = w.Work2Main[:0]
	w.Back2Work = w.Back2Work[:0]
}

// The room number and object slot every registration test passes, chosen so that a field
// holding the wrong one of the two is obvious in a failure message.
const (
	testRoom int16 = 5
	testSlot int16 = 7
)

// testWhere is the authored rect the registration table is written against: 60 wide by 30
// tall at (200, 100), room-local. Wide enough that the centring arithmetic has a non-zero
// answer for every sprite, and far enough from the room edges that no retire test fires on
// the first frame.
var testWhere = render.SetRect(200, 100, 260, 130)

// registerOne registers a single object of the given type into an otherwise empty table.
// `where` is room-local, exactly as the seventeen call sites inside DrawARoomsObjects pass
// it.
func registerOne(t *testing.T, what int16, obj house.Object, where Rect) *World {
	t.Helper()
	w := dynaWorld(t)
	w.R.NumLights = 3
	if slot := w.AddDynamicObject(what, where, obj, testRoom, testSlot, true); slot != 0 {
		t.Fatalf("AddDynamicObject(%#x) = %d, want slot 0", what, slot)
	}
	return w
}

// ---------------------------------------------------------------------------
// ZeroDinahs
// ---------------------------------------------------------------------------

// TestZeroDinahsClearsTwelveOfFourteen pins which fields the reset writes.
//
// The C writes twelve of the fourteen: Byte1 and Moving are not touched, and the table is
// NewPtr'd rather than NewPtrClear'd, so in the original those two hold whatever the memory
// manager left. Go's zero value covers them, which makes the port stricter in a way nothing
// can observe -- so this test asserts the *twelve* and deliberately does not assert the other
// two. Collapsing the function to `Dynamic{}` still passes; dropping one of the twelve does
// not.
func TestZeroDinahsClearsTwelveOfFourteen(t *testing.T) {
	w := dynaWorld(t)

	for i := range w.Dinahs {
		w.Dinahs[i] = Dynamic{
			Type: Balloon, Dest: render.SetRect(1, 2, 3, 4), Whole: render.SetRect(5, 6, 7, 8),
			HVel: 9, VVel: 10, Count: 11, Frame: 12, Timer: 13, Position: 14, Room: 15,
			Byte0: 16, Byte1: 17, Moving: true, Active: true,
		}
	}
	w.NumDynamics = MaxDynamicObs

	w.ZeroDinahs()

	if w.NumDynamics != 0 {
		t.Errorf("NumDynamics %d, want 0", w.NumDynamics)
	}
	var zero Rect
	for i := range w.Dinahs {
		d := &w.Dinahs[i]
		if d.Type != house.ObjectIsEmpty {
			t.Fatalf("slot %d: Type %#x, want ObjectIsEmpty", i, d.Type)
		}
		if d.Dest != zero || d.Whole != zero {
			t.Fatalf("slot %d: rects %v %v, want both empty", i, d.Dest, d.Whole)
		}
		if d.HVel|d.VVel|d.Count|d.Frame|d.Timer|d.Position|d.Room != 0 || d.Byte0 != 0 {
			t.Fatalf("slot %d: a numeric field survived: %+v", i, *d)
		}
		if d.Active {
			t.Fatalf("slot %d: Active survived", i)
		}
	}
}

// ---------------------------------------------------------------------------
// AddDynamicObject: the seventeen registrations
// ---------------------------------------------------------------------------

// TestRegistrationFieldsPerType is the acceptance criterion's "post-registration slot
// fields", table-driven over all seventeen registrable types.
//
// The `note` column marks the entries worth reading twice: three types discard part of the
// author's coordinate, two store a *negative* velocity in Count where the toaster stores a
// positive one, one is forced three times wider than its art, and one leaves Room alone.
func TestRegistrationFieldsPerType(t *testing.T) {
	where := testWhere
	// The seven appliances add the play origin straight back, putting Dest into screen
	// space; the other ten stay room-local.
	oh, ov := render.DefaultView().OriginH, render.DefaultView().OriginV
	appL, appT := where.Left+oh, where.Top+ov

	// The half-rate launch speed the ball and the fish solve for. Read from the helper
	// rather than hard-coded, because the loop itself is TestLaunchVelocityFromHeight's
	// subject and duplicating its answer here would only pin it twice.
	halfRate := dynaWorld(t).launchVelocityHalfRate

	type want struct {
		dest                                  Rect
		hvel, vvel, count, frame, timer, posn int16
		note                                  string
	}

	cases := []struct {
		name string
		what int16
		obj  house.Object
		want want
	}{{
		name: "sparkle", what: Sparkle, obj: applianceObj(Sparkle, 0, 0),
		want: want{
			dest: place(render.SparkleSrc[0], where.Left, where.Top),
			note: "Timer is RandomInt(60)+15, so it is asserted as a range below",
		},
	}, {
		name: "toaster", what: Toaster, obj: applianceObj(Toaster, 37, 15),
		want: want{
			// The bread, centred in the object and pushed down until its top sits on the
			// object's top, so the slice starts fully hidden inside the slot.
			dest:  centredAt(render.BreadSrc[0], where),
			hvel:  where.Top + 2, // the clip line, not a velocity
			vvel:  -9,            // launchVelocity(37): 1+..+9 = 45 >= 37, 1+..+8 = 36 < 37
			count: 9,             // positive here, unlike the ball's and the fish's
			frame: 45,            // delay 15 * 3, the idle countdown
			timer: 45,            // and the reload period it refills from
			note:  "HVel is a clip line and Frame a countdown; both change meaning in flight",
		},
	}, {
		name: "macplus", what: MacPlus, obj: applianceObj(MacPlus, 0, 0),
		want: want{
			dest: place(render.PlusScreen1, appL+10, appT+7),
			note: "Timer 0: a Mac Plus does nothing until ToggleMacPlus gives it 40 or 10",
		},
	}, {
		name: "tv", what: TV, obj: applianceObj(TV, 0, 0),
		want: want{dest: place(render.TVScreen1, appL+17, appT+10)},
	}, {
		name: "coffee", what: Coffee, obj: applianceObj(Coffee, 0, 0),
		want: want{
			dest: place(render.CoffeeLight1, appL+32, appT+57), timer: 200,
			note: "200 is not a reload period; HandleCoffee overwrites it and never settles",
		},
	}, {
		name: "outlet", what: Outlet, obj: applianceObj(Outlet, 0, 20),
		want: want{
			dest:  place(render.OutletSrc[0], appL, appT),
			hvel:  3,  // the room's light count, from Scene.NumLights
			count: 60, // delay 20 * 6 / TicksPerFrame
			timer: 60,
			note:  "HVel is a light count; UpdateOutletsLighting is its only later writer",
		},
	}, {
		name: "vcr", what: VCR, obj: applianceObj(VCR, 0, 0),
		want: want{
			dest: place(render.VCRTime1, appL+64, appT+6), timer: 115,
			note: "115/100 is the fifteen-frame blink, forever",
		},
	}, {
		name: "stereo", what: Stereo, obj: applianceObj(Stereo, 0, 0),
		want: want{dest: place(render.StereoLight1, appL+56, appT+20)},
	}, {
		name: "microwave", what: Microwave, obj: applianceObj(Microwave, 0, 0),
		want: want{
			dest: func() Rect {
				r := place(render.MicroOn, appL+14, appT+13)
				r.Right = r.Left + 48 // three times the art's width; HandleMicrowave tiles
				return r
			}(),
			note: "the slot is forced 48 wide over 16-wide art -- hazard H5",
		},
	}, {
		name: "balloon", what: Balloon, obj: enemyObj(Balloon, 0, 7),
		want: want{
			dest:  place(render.BalloonSrc[0], where.Left, BalloonStart-render.BalloonSrc[0].Tall()),
			vvel:  -2,
			count: 21, timer: 21, // delay 7 * 6 / 2
			note: "the authored y is DISCARDED -- a balloon always starts at the floor",
		},
	}, {
		name: "copter left", what: CopterLf, obj: enemyObj(CopterLf, 0, 7),
		want: want{
			dest: place(render.CopterSrc[0], where.Left, CopterStart),
			hvel: -1, vvel: 2, count: 21, timer: 21, posn: where.Left,
			note: "the authored y is discarded too, at the ceiling; HVel == 0 means SHOT",
		},
	}, {
		name: "copter right", what: CopterRt, obj: enemyObj(CopterRt, 0, 7),
		want: want{
			dest: place(render.CopterSrc[0], where.Left, CopterStart),
			hvel: 1, vvel: 2, count: 21, timer: 21, posn: where.Left,
		},
	}, {
		name: "dart left", what: DartLf, obj: enemyObj(DartLf, 0, 7),
		want: want{
			dest: place(render.DartSrc[0], RoomWide-render.DartSrc[0].Wide(), where.Top),
			hvel: -DartVelocity, vvel: 2, count: 21, frame: 0, timer: 21, posn: where.Top,
			note: "the authored x is DISCARDED, and Room is never assigned at all",
		},
	}, {
		name: "dart right", what: DartRt, obj: enemyObj(DartRt, 0, 7),
		want: want{
			dest: place(render.DartSrc[0], 0, where.Top),
			hvel: DartVelocity, vvel: 2, count: 21, frame: 2, timer: 21, posn: where.Top,
			note: "Frame 2 is a direction, not an animation phase",
		},
	}, {
		name: "ball", what: Ball, obj: enemyObj(Ball, 80, 0),
		want: want{
			dest: place(render.BallSrc[0], where.Left, where.Top),
			vvel: -halfRate(80), count: -halfRate(80),
			posn: where.Top + render.BallSrc[0].Tall(), // the floor is where it started
			note: "Count is NEGATIVE here, where the toaster's is positive; Timer is unused",
		},
	}, {
		name: "drip", what: Drip, obj: enemyObj(Drip, 53, 13),
		want: want{
			dest:  centredAt(render.DripSrc[0], where),
			hvel:  where.Top, // remember where to hang the next one back up
			count: 39, frame: 3, timer: 39,
			posn: where.Top + 53,
			note: "Frame 3 is the hanging drop, which is also what the static draw paints",
		},
	}, {
		name: "fish", what: Fish, obj: enemyObj(Fish, 80, 11),
		want: want{
			dest: place(render.FishSrc[0], where.Left+10, where.Top+8),
			hvel: 33, // the respawn delay: delay 11 * 6 / 2
			vvel: -halfRate(80), count: -halfRate(80), timer: 33,
			posn: where.Top + 8 + render.FishSrc[0].Tall(),
			note: "reads Enemy().Length where the C reads data.g.height -- the same two bytes",
		},
	}}

	if len(cases) != 17 {
		t.Fatalf("the table has %d entries, want all 17 registrable types", len(cases))
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := registerOne(t, c.what, c.obj, where)
			if w.NumDynamics != 1 {
				t.Fatalf("NumDynamics %d, want 1", w.NumDynamics)
			}
			d := &w.Dinahs[0]

			if d.Type != c.what {
				t.Errorf("Type %#x, want %#x", d.Type, c.what)
			}
			if d.Dest != c.want.dest {
				t.Errorf("Dest %+v, want %+v (%s)", d.Dest, c.want.dest, c.want.note)
			}
			// Every one of the seventeen sets Whole = Dest. Nothing registers with a trail.
			if d.Whole != d.Dest {
				t.Errorf("Whole %+v, want == Dest %+v", d.Whole, d.Dest)
			}
			if d.HVel != c.want.hvel {
				t.Errorf("HVel %d, want %d (%s)", d.HVel, c.want.hvel, c.want.note)
			}
			if d.VVel != c.want.vvel {
				t.Errorf("VVel %d, want %d", d.VVel, c.want.vvel)
			}
			if d.Count != c.want.count {
				t.Errorf("Count %d, want %d (%s)", d.Count, c.want.count, c.want.note)
			}
			if d.Frame != c.want.frame {
				t.Errorf("Frame %d, want %d (%s)", d.Frame, c.want.frame, c.want.note)
			}
			if c.what == Sparkle {
				// The one seeded field in the whole function, and the reason a room's
				// sparkle count shifts the RNG stream -- hazard H6.
				if d.Timer < 15 || d.Timer > 74 {
					t.Errorf("Timer %d, want RandomInt(60)+15, i.e. 15..74", d.Timer)
				}
			} else if d.Timer != c.want.timer {
				t.Errorf("Timer %d, want %d", d.Timer, c.want.timer)
			}
			if d.Position != c.want.posn {
				t.Errorf("Position %d, want %d", d.Position, c.want.posn)
			}
			if d.Byte0 != byte(testSlot) {
				t.Errorf("Byte0 %d, want %d, the object's slot within its room", d.Byte0, testSlot)
			}
			if d.Moving {
				t.Error("Moving is true; nothing registers already in motion")
			}
			if !d.Active {
				t.Error("Active is false after registering with isOn = true")
			}
			if c.what != DartLf && c.what != DartRt && d.Room != testRoom {
				t.Errorf("Room %d, want %d", d.Room, testRoom)
			}
		})
	}
}

// TestDartRegistrationLeavesRoomZero is the one registration case with a field missing, and
// it is pinned rather than fixed.
//
// The two dart arms never assign Room, and that is harmless *by construction*. Room has
// exactly one reader in the whole game, UpdateOutletsLighting, which filters on
// Type == Outlet -- and an outlet is one of the seven appliances, the only types that can
// register from a room other than the central one. A dart is gated on
// neighbor == kCentralRoom at its call site, so its Room could only ever hold the central
// room's number, the one value that filter never has to tell apart.
//
// Do not "complete" the dart's registration. See trip.go's UpdateOutletsLighting.
func TestDartRegistrationLeavesRoomZero(t *testing.T) {
	for _, what := range []int16{DartLf, DartRt} {
		w := dynaWorld(t)
		w.Dinahs[0].Room = 99 // a value a Room assignment would have to overwrite
		if got := w.AddDynamicObject(what, testWhere, enemyObj(what, 0, 7),
			testRoom, 0, true); got != 0 {
			t.Fatalf("AddDynamicObject(%#x) = %d, want 0", what, got)
		}
		if w.Dinahs[0].Room != 99 {
			t.Errorf("type %#x: Room %d, want the stale 99 -- the dart's case must not assign "+
				"Room", what, w.Dinahs[0].Room)
		}
	}
}

// TestAddDynamicObjectRefusesPastEighteen pins the cap.
//
// The nineteenth registration returns -1 and leaves the counter at eighteen, and that -1 is
// what the composition writes back into the object's DynaNum -- which is why every Toggle*
// and Trigger* has to be able to refuse it. See World.dinah and
// TestTogglesRefuseUnregisteredObjects.
func TestAddDynamicObjectRefusesPastEighteen(t *testing.T) {
	w := dynaWorld(t)
	obj := enemyObj(Balloon, 0, 7)

	for i := 0; i < MaxDynamicObs; i++ {
		if got := w.AddDynamicObject(Balloon, testWhere, obj, testRoom, int16(i), true); got != int16(i) {
			t.Fatalf("registration %d returned %d", i, got)
		}
	}
	if w.NumDynamics != MaxDynamicObs {
		t.Fatalf("NumDynamics %d, want %d", w.NumDynamics, MaxDynamicObs)
	}

	if got := w.AddDynamicObject(Balloon, testWhere, obj, testRoom, 19, true); got != -1 {
		t.Errorf("the 19th registration returned %d, want -1", got)
	}
	if w.NumDynamics != MaxDynamicObs {
		t.Errorf("NumDynamics %d after a refused registration, want %d",
			w.NumDynamics, MaxDynamicObs)
	}
}

// TestAddDynamicObjectRejectsUnknownTypes pins the default arm, and the one write it leaves
// behind.
//
// The Type assignment is above the switch, so an unknown code *does* leave Type set on the
// slot NumDynamics still points at. Harmless -- the next successful registration overwrites
// it and nothing reads past NumDynamics -- and asserted here so that a reader who finds the
// stale Type does not report it as a bug.
func TestAddDynamicObjectRejectsUnknownTypes(t *testing.T) {
	w := dynaWorld(t)
	for _, what := range []int16{Table, RedClock, Mirror, LightSwitch, GreaseRt,
		house.ObjectIsEmpty} {
		before := w.NumDynamics
		if got := w.AddDynamicObject(what, testWhere, house.Object{What: what},
			testRoom, 0, true); got != -1 {
			t.Errorf("type %#x returned %d, want -1", what, got)
		}
		if w.NumDynamics != before {
			t.Errorf("type %#x advanced NumDynamics to %d", what, w.NumDynamics)
		}
		if w.Dinahs[before].Type != what {
			t.Errorf("type %#x: slot Type is %#x, want the stale write the C leaves behind",
				what, w.Dinahs[before].Type)
		}
	}
}

// TestShredderIsNotRegistrable is worth its own name, because kShredder looks like it belongs
// in the table and does not.
//
// It is an appliance in the house format, it has a hot spot, it has animation art, and its
// cels are drawn by RenderShreds out of the *shreds* table -- 1.5f's work -- not out of
// dinahs. An arm for it here would silently steal one of the eighteen slots from every room
// that has one.
func TestShredderIsNotRegistrable(t *testing.T) {
	w := dynaWorld(t)
	if got := w.AddDynamicObject(Shredder, testWhere, applianceObj(Shredder, 0, 0),
		testRoom, 0, true); got != -1 {
		t.Errorf("kShredder registered into slot %d; it belongs to the shreds table", got)
	}
	if isRegistrableType(Shredder) {
		t.Error("isRegistrableType says kShredder is registrable; the census would be wrong")
	}
}

// TestLaunchVelocityFromHeight pins both reverse-engineering loops, as closed forms.
//
// The house stores how *high* a toaster's bread or a ball's bounce should go, and these two
// loops solve for the launch speed by simulating the fall until the steps cover it. The sum of
// the first v steps is what each one is really computing, so the answer is the smallest v
// whose triangular sum reaches the height -- which is worth asserting as arithmetic rather
// than as a second copy of the loop.
//
//	full rate  position falls by 1,2,3,...   after v steps: v(v+1)/2
//	half rate  by 1,1,2,2,3,3,...           after 2v steps: v(v+1)
//
// **So the half-rate loop returns the SMALLER velocity**, which is the opposite of what the
// name suggests until you read it as physics: gravity at half strength means a longer flight,
// so less launch speed reaches the same height. A ball authored to bounce 80 pixels leaves at
// 9, where a toaster authored to throw bread 80 pixels leaves at 13.
//
// A height of 0 still yields 1 in both, because the C is a do/while -- a toaster with height 0
// launches, one pixel -- and the port's break is at the bottom for exactly that reason.
func TestLaunchVelocityFromHeight(t *testing.T) {
	// The smallest v >= 1 whose steps cover the height, computed by counting rather than by
	// simulating.
	solve := func(height int16, sum func(v int16) int16) int16 {
		for v := int16(1); ; v++ {
			if sum(v) >= height {
				return v
			}
		}
	}
	fullSum := func(v int16) int16 { return v * (v + 1) / 2 }
	halfSum := func(v int16) int16 { return v * (v + 1) }

	// A handful of the answers spelled out, so a failure names a number rather than a formula.
	for _, c := range []struct{ height, full, half int16 }{
		{0, 1, 1}, // the do/while's one iteration
		{1, 1, 1},
		{2, 2, 1},
		{8, 4, 3},
		{37, 9, 6},
		{45, 9, 7}, // the half-rate loop breaks on the FIRST step of the pair: 42 + 7
		{46, 10, 7},
		{80, 13, 9},
		{200, 20, 14},
	} {
		if got := launchVelocity(c.height); got != c.full {
			t.Errorf("launchVelocity(%d) = %d, want %d", c.height, got, c.full)
		}
		if got := dynaWorld(t).launchVelocityHalfRate(c.height); got != c.half {
			t.Errorf("launchVelocityHalfRate(%d) = %d, want %d", c.height, got, c.half)
		}
		if c.half > c.full {
			t.Errorf("height %d: the half-rate answer %d must not exceed the full-rate %d",
				c.height, c.half, c.full)
		}
	}

	// And the closed forms across the whole range a house can author, which is what would
	// catch an off-by-one in either loop's break placement.
	w := dynaWorld(t)
	for height := int16(0); height <= 400; height++ {
		if got, want := launchVelocity(height), solve(height, fullSum); got != want {
			t.Fatalf("launchVelocity(%d) = %d, want %d (v(v+1)/2 >= height)", height, got, want)
		}
		if got, want := w.launchVelocityHalfRate(height), solve(height, halfSum); got != want {
			t.Fatalf("launchVelocityHalfRate(%d) = %d, want %d (v(v+1) >= height)",
				height, got, want)
		}
	}
}

// TestBallAndFishRegistrationSetEvenFrame is hazard H1, asserted rather than left to be
// rediscovered.
//
// launchVelocityHalfRate writes the global EvenFrame, because in the C `lilFrame` is a local
// and `evenFrame` is an extern the author reached by accident. It is vestigial *inside*
// AddDynamicObject -- nothing there reads it -- but the collateral effect is visible:
// EvenFrame selects RenderFlames versus RenderStars and gates every half-rate mover, so
// composing a room that registers a ball or a fish resynchronises every flame and star in
// the locale.
//
// The toaster's loop is full rate, needs no toggle, and writes neither flag.
func TestBallAndFishRegistrationSetEvenFrame(t *testing.T) {
	for _, c := range []struct {
		name  string
		what  int16
		obj   house.Object
		wants bool
	}{
		{"ball", Ball, enemyObj(Ball, 80, 0), true},
		{"fish", Fish, enemyObj(Fish, 80, 11), true},
		{"toaster", Toaster, applianceObj(Toaster, 37, 15), false},
		{"balloon", Balloon, enemyObj(Balloon, 0, 7), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := dynaWorld(t)
			w.EvenFrame = false
			w.AddDynamicObject(c.what, testWhere, c.obj, testRoom, 0, true)
			if w.EvenFrame != c.wants {
				t.Errorf("EvenFrame %v after registering a %s, want %v",
					w.EvenFrame, c.name, c.wants)
			}
		})
	}
}

// TestFishReadsTheSameBytesAsTheEnemyLength pins the union alias the fish's registration
// depends on.
//
// The C's fish arm reads `who->data.g.height` -- the *appliance* view -- where every other
// enemy reads `data.h.length`. It is not a bug: applianceType.height and enemyType.length are
// both the big-endian short at payload offset 4, so the wrong union member names the right two
// bytes. The port reads Enemy().Length because that is the family a fish belongs to, and this
// test is what keeps the two from drifting apart if either struct is edited.
//
// The two *delay* fields do not alias -- offset 7 against offset 6 -- which is why this test
// asserts the one and denies the other.
func TestFishReadsTheSameBytesAsTheEnemyLength(t *testing.T) {
	o := house.Object{What: Fish}
	o.SetEnemy(house.Enemy{Length: 1234, Delay: 11, Initial: 1, State: 1})

	if got, want := o.Appliance().Height, o.Enemy().Length; got != want {
		t.Errorf("Appliance().Height = %d, Enemy().Length = %d; they must alias", got, want)
	}
	if o.Appliance().Delay == o.Enemy().Delay {
		t.Errorf("both Delay fields read %d; they are at different payload offsets and this "+
			"fixture has to be able to tell them apart", o.Enemy().Delay)
	}

	// And the consequence: a fish registers the launch speed the *length* asks for.
	w := registerOne(t, Fish, enemyObj(Fish, 80, 11), testWhere)
	if want := -w.launchVelocityHalfRate(80); w.Dinahs[0].VVel != want {
		t.Errorf("VVel %d, want %d -- the fish must solve for Length, not for Delay",
			w.Dinahs[0].VVel, want)
	}
}

// TestFishSrcIsAlreadyAtTheOrigin turns AddDynamicObject's comment into an assertion.
//
// The fish is the one registration with no ZeroCorner call. That is only correct because
// FishSrc[0] happens to sit at the top-left of its strip; if the strip were ever re-cut with
// a margin, the fish would register at an offset nobody would think to look for, and every
// other case would still be right.
func TestFishSrcIsAlreadyAtTheOrigin(t *testing.T) {
	if render.FishSrc[0].Left != 0 || render.FishSrc[0].Top != 0 {
		t.Errorf("FishSrc[0] = %+v, want its top-left at the origin -- AddDynamicObject's "+
			"kFish case omits ZeroCorner and depends on it", render.FishSrc[0])
	}
}

// ---------------------------------------------------------------------------
// One frame of each of the seventeen handlers
// ---------------------------------------------------------------------------

// TestOneHandlerFramePerType is the second half of the acceptance criterion: "one frame of
// its handler", for all seventeen labels over the fifteen handlers -- CopterLf/CopterRt share
// HandleCopter and DartLf/DartRt share HandleDart.
//
// Each case registers its type, optionally pokes the one field that makes the next frame
// interesting rather than a plain decrement, runs HandleDynamics once, and asserts what
// changed. The pokes are all of the form "set the timer to where the schedule gets
// interesting", which is what the toggles do in the real game.
func TestOneHandlerFramePerType(t *testing.T) {
	cases := []struct {
		name  string
		what  int16
		obj   house.Object
		poke  func(d *Dynamic)
		check func(t *testing.T, w *World, d *Dynamic, snd *soundLog)
	}{{
		name: "sparkle idles", what: Sparkle, obj: applianceObj(Sparkle, 0, 0),
		poke: func(d *Dynamic) { d.Timer = 9 },
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if d.Timer != 8 || d.Frame != 0 {
				t.Errorf("Timer %d Frame %d, want 8 and 0", d.Timer, d.Frame)
			}
			if w.NumSparkles != 0 {
				t.Errorf("NumSparkles %d, want 0 -- nothing puffs until the timer expires",
					w.NumSparkles)
			}
			snd.is(t)
		},
	}, {
		name: "sparkle emits", what: Sparkle, obj: applianceObj(Sparkle, 0, 0),
		poke: func(d *Dynamic) { d.Timer = 1 },
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			// Frame is a five-frame lockout, not an animation index, and it is
			// NumSparkleModes because that is how long RenderSparkles takes to run one puff
			// down.
			if d.Frame != NumSparkleModes {
				t.Errorf("Frame %d, want the %d-frame lockout", d.Frame, NumSparkleModes)
			}
			if d.Timer < 60 || d.Timer > 299 {
				t.Errorf("Timer %d, want RandomInt(240)+60, i.e. 60..299", d.Timer)
			}
			if w.NumSparkles != 1 {
				t.Errorf("NumSparkles %d, want 1", w.NumSparkles)
			}
			snd.is(t, MysticSound)
		},
	}, {
		name: "toaster counts down", what: Toaster, obj: applianceObj(Toaster, 37, 15),
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if d.Frame != 44 || d.Moving {
				t.Errorf("Frame %d Moving %v, want 44 and false", d.Frame, d.Moving)
			}
			snd.is(t)
		},
	}, {
		name: "toaster launches", what: Toaster, obj: applianceObj(Toaster, 37, 15),
		poke: func(d *Dynamic) { d.Frame = 1 },
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if !d.Moving || d.VVel != -d.Count || d.Frame != 0 {
				t.Errorf("Moving %v VVel %d Frame %d, want true, %d and 0",
					d.Moving, d.VVel, d.Frame, -d.Count)
			}
			snd.is(t, ToastLaunchSound)
		},
	}, {
		name: "macplus chimes at thirty", what: MacPlus, obj: applianceObj(MacPlus, 0, 0),
		poke: func(d *Dynamic) { d.Timer = 31 },
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			// Ten frames before the screen lights up, which is a Mac Plus booting.
			if d.Timer != 30 {
				t.Errorf("Timer %d, want 30", d.Timer)
			}
			snd.is(t, MacOnSound)
		},
	}, {
		name: "tv stages its screen", what: TV, obj: applianceObj(TV, 0, 0),
		poke: func(d *Dynamic) { d.Timer = 2 },
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			// The staging half of the two-frame reveal: new art into the *back* map plus a
			// back rect, so next frame's back->work carries it into the work map. The reveal
			// itself is a work rect one frame later.
			if work, back := rectCounts(w); work != 0 || back != 1 {
				t.Errorf("rects work %d back %d, want 0 and 1 -- staging is a back rect",
					work, back)
			}
			snd.is(t, TVOnSound)
		},
	}, {
		name: "tv reveals", what: TV, obj: applianceObj(TV, 0, 0),
		poke: func(d *Dynamic) { d.Timer = 1 },
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if work, back := rectCounts(w); work != 1 || back != 0 {
				t.Errorf("rects work %d back %d, want 1 and 0 -- the reveal is a work rect",
					work, back)
			}
			snd.is(t)
		},
	}, {
		name: "coffee gurgles", what: Coffee, obj: applianceObj(Coffee, 0, 0),
		poke: func(d *Dynamic) { d.Timer = 101 },
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			// The self-restart at Timer == 100 is what makes 1 and 0 unreachable for as long
			// as the percolator stays on.
			if d.Timer < 200 || d.Timer > 399 {
				t.Errorf("Timer %d, want 200 + RandomInt(200)", d.Timer)
			}
			snd.is(t, CoffeeSound)
		},
	}, {
		name: "outlet starts a zap", what: Outlet, obj: applianceObj(Outlet, 0, 20),
		poke: func(d *Dynamic) { d.Timer = 1 },
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			// Position is the launch/idle enum, where the toaster uses Moving.
			if d.Position != 1 || d.Timer != LengthOfZap {
				t.Errorf("Position %d Timer %d, want 1 and %d", d.Position, d.Timer, LengthOfZap)
			}
			snd.is(t, ZapSound)
		},
	}, {
		name: "vcr blinks", what: VCR, obj: applianceObj(VCR, 0, 0),
		poke: func(d *Dynamic) { d.Timer = 101 },
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			// Frame here is a two-state toggle flipped as 1 - Frame, the third distinct
			// meaning of Frame in Dynamics.c.
			if d.Timer != 115 || d.Frame != 1 {
				t.Errorf("Timer %d Frame %d, want 115 and 1", d.Timer, d.Frame)
			}
			snd.is(t)
		},
	}, {
		name: "stereo reveals its led", what: Stereo, obj: applianceObj(Stereo, 0, 0),
		poke: func(d *Dynamic) { d.Timer = 1 },
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if work, _ := rectCounts(w); work != 1 {
				t.Errorf("work rects %d, want 1", work)
			}
			// ToggleMusicWhilePlaying is 1.6's work and is still a stub, so the music half
			// of this frame is unobservable; the reveal is what is testable now.
			snd.is(t)
		},
	}, {
		name: "microwave tiles its window", what: Microwave, obj: applianceObj(Microwave, 0, 0),
		poke: func(d *Dynamic) { d.Timer = 2 },
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if _, back := rectCounts(w); back != 1 {
				t.Errorf("back rects %d, want 1 -- the full 48-wide slot once, not three "+
					"16-wide scratches", back)
			}
			snd.is(t, MacOnSound)
		},
	}, {
		name: "balloon waits", what: Balloon, obj: enemyObj(Balloon, 0, 7),
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if d.Timer != 20 || d.Moving {
				t.Errorf("Timer %d Moving %v, want 20 and false", d.Timer, d.Moving)
			}
			snd.is(t)
		},
	}, {
		name: "copter left waits", what: CopterLf, obj: enemyObj(CopterLf, 0, 7),
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if d.Timer != 20 || d.Moving {
				t.Errorf("Timer %d Moving %v, want 20 and false", d.Timer, d.Moving)
			}
		},
	}, {
		name: "copter right waits", what: CopterRt, obj: enemyObj(CopterRt, 0, 7),
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if d.Timer != 20 {
				t.Errorf("Timer %d, want 20", d.Timer)
			}
		},
	}, {
		name: "dart left waits", what: DartLf, obj: enemyObj(DartLf, 0, 7),
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if d.Timer != 20 || d.Moving {
				t.Errorf("Timer %d Moving %v, want 20 and false", d.Timer, d.Moving)
			}
		},
	}, {
		name: "dart right waits", what: DartRt, obj: enemyObj(DartRt, 0, 7),
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if d.Timer != 20 {
				t.Errorf("Timer %d, want 20", d.Timer)
			}
		},
	}, {
		name: "ball kicks itself off", what: Ball, obj: enemyObj(Ball, 80, 0),
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			// The idle arm is three statements and one of them writes a global.
			if !d.Moving || d.VVel != d.Count {
				t.Errorf("Moving %v VVel %d, want true and Count %d", d.Moving, d.VVel, d.Count)
			}
			if !w.EvenFrame {
				t.Error("EvenFrame is false; HandleBall's idle arm writes the global true")
			}
		},
	}, {
		name: "drip hangs", what: Drip, obj: enemyObj(Drip, 53, 13),
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			if d.Timer != 38 || d.Frame != 3 || d.Moving {
				t.Errorf("Timer %d Frame %d Moving %v, want 38, 3 and false",
					d.Timer, d.Frame, d.Moving)
			}
			snd.is(t)
		},
	}, {
		name: "fish waits in the water", what: Fish, obj: enemyObj(Fish, 80, 11),
		check: func(t *testing.T, w *World, d *Dynamic, snd *soundLog) {
			// 33 & 3 == 1, so the bob does not fire on this frame and only the countdown
			// moves. TestFishBobsWhileSwitchedOff covers the other half.
			if d.Timer != 32 || d.Moving {
				t.Errorf("Timer %d Moving %v, want 32 and false", d.Timer, d.Moving)
			}
			snd.is(t)
		},
	}}

	// Every one of the seventeen labels has to appear, or the acceptance criterion is not
	// met -- and a label silently dropped from the table is exactly the gap a hand-maintained
	// list develops.
	covered := map[int16]bool{}
	for _, c := range cases {
		covered[c.what] = true
	}
	for _, what := range allRegistrableTypes() {
		if !covered[what] {
			t.Errorf("no handler-frame case for type %#x", what)
		}
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := registerOne(t, c.what, c.obj, testWhere)
			w.EvenFrame = true
			if c.poke != nil {
				c.poke(&w.Dinahs[0])
			}
			var snd soundLog
			snd.install(w)
			clearRects(w)

			w.HandleDynamics()

			c.check(t, w, &w.Dinahs[0], &snd)
		})
	}
}

// TestHandleDynamicsIgnoresEmptySlots pins the default arm of the dispatch loop.
//
// The C has no default at all, and does not need one: the loop runs to NumDynamics, and
// every slot below it registered successfully. The port's default is therefore unreachable
// in the shipped game -- except through the one path that can leave a live-looking slot
// behind, an unknown type code, which sets Type and does not advance the counter.
func TestHandleDynamicsIgnoresEmptySlots(t *testing.T) {
	w := dynaWorld(t)
	// A slot with a plausible Type but no registration behind it, inside the loop's range.
	w.Dinahs[0] = Dynamic{Type: house.ObjectIsEmpty, Timer: 500, Active: true}
	w.Dinahs[1] = Dynamic{Type: Table, Timer: 500, Active: true}
	w.NumDynamics = 2

	w.HandleDynamics()

	for i := 0; i < 2; i++ {
		if w.Dinahs[i].Timer != 500 {
			t.Errorf("slot %d: Timer %d, want the untouched 500", i, w.Dinahs[i].Timer)
		}
	}
}

// ---------------------------------------------------------------------------
// The cap against shipped content
// ---------------------------------------------------------------------------

// TestBusiestShippedLocaleSaturates is the acceptance criterion's census.
//
// The point is not that some house happens to be busy. It is that eighteen is *reachable in
// shipped content*, which is what makes AddDynamicObject's -1 and every Toggle*'s refusal
// live code rather than defensive padding.
//
// The pressure is per locale, not per room, and the two halves are different: the seven
// appliances register from all nine rooms, and the other ten types only from the central one
// (every non-appliance call site is gated on neighbor == kCentralRoom). So the count below is
// appliances across the whole nine-room locale plus movers in the middle. It is an upper
// bound -- registration also needs IsThisValid and a window intersection -- which is the safe
// direction: if the bound never reached eighteen, the cap would be unreachable for certain.
//
// The census is computed rather than checked in as a number, because a checked-in number
// would only pin the extractor. The busiest locale is logged by name so that a change to the
// drawing order -- which decides *which* objects get the eighteen slots -- shows up here as a
// different winner rather than as silence.
func TestBusiestShippedLocaleSaturates(t *testing.T) {
	houseDir := requireAssets(t, "houses")
	paths, _ := filepath.Glob(filepath.Join(houseDir, "*.house"))
	if len(paths) == 0 {
		t.Skipf("no houses in %s", houseDir)
	}
	sort.Strings(paths)

	type census struct {
		house string
		room  int16
		name  string
		count int
	}
	var busiest census
	saturated, houses := 0, 0

	for _, path := range paths {
		h, err := house.LoadFile(path)
		if err != nil {
			continue
		}
		houses++
		w := newTestWorld(h, "", "")
		hit := false
		for i := range h.Rooms {
			w.R.RoomNumber = int16(i)
			n := 0
			for slot := 0; slot < 9; slot++ {
				rm := w.Room(w.R.GetNeighborRoomNumber(slot))
				if rm == nil {
					continue
				}
				for j := range rm.Objects {
					what := rm.Objects[j].What
					if !isRegistrableType(what) {
						continue
					}
					// The split that makes this a locale count and not nine room counts.
					if slot == CentralRoom || isApplianceType(what) {
						n++
					}
				}
			}
			if n > busiest.count {
				busiest = census{filepath.Base(path), int16(i), h.Rooms[i].Name.Text(), n}
			}
			if n >= MaxDynamicObs {
				saturated++
				hit = true
			}
		}
		if hit {
			t.Logf("%s has at least one saturating locale", filepath.Base(path))
		}
	}

	t.Logf("busiest locale over %d houses: %s room %d %q, %d registrable objects against a cap "+
		"of %d", houses, busiest.house, busiest.room, busiest.name, busiest.count, MaxDynamicObs)
	t.Logf("%d shipped locales reach the cap", saturated)

	if busiest.count == 0 {
		t.Fatal("no shipped locale holds a registrable object; the census found nothing")
	}
	if saturated == 0 {
		t.Errorf("no shipped locale reaches %d registrable objects (busiest is %d), so "+
			"AddDynamicObject's -1 would be unreachable in the shipped game -- if this is a "+
			"deliberate change to the corpus, the note on World.dinah needs revisiting",
			MaxDynamicObs, busiest.count)
	}
}

// allRegistrableTypes is the seventeen codes AddDynamicObject has an arm for, in the order
// the switch lists them.
//
// Written out rather than derived by probing AddDynamicObject, so that an arm added to the
// switch without a thought for the census or the handler-frame table fails a test instead of
// widening one silently.
func allRegistrableTypes() []int16 {
	return []int16{Sparkle, Toaster, MacPlus, TV, Coffee, Outlet, VCR, Stereo, Microwave,
		Balloon, CopterLf, CopterRt, DartLf, DartRt, Ball, Drip, Fish}
}

func isRegistrableType(what int16) bool {
	for _, t := range allRegistrableTypes() {
		if t == what {
			return true
		}
	}
	return false
}

// isApplianceType is the seven that put Dest into screen space, register from all nine rooms,
// draw into the back map and therefore have no renderer. One property, four consequences --
// see AddDynamicObject's header.
func isApplianceType(what int16) bool {
	switch what {
	case MacPlus, TV, Coffee, Outlet, VCR, Stereo, Microwave:
		return true
	}
	return false
}

// typicalObject is a plausible ten-byte payload for each of the seventeen, so that a test can
// iterate allRegistrableTypes without a per-type literal.
//
// Which family a type's payload belongs to is not cosmetic -- Height/Delay and Length/Delay sit
// at different offsets -- so the split is spelled out. kSparkle is in neither: its payload is
// bonusType, and AddDynamicObject's kSparkle arm reads none of it, so anything serves.
func typicalObject(what int16) house.Object {
	switch what {
	case Toaster, MacPlus, TV, Coffee, Outlet, VCR, Stereo, Microwave, Sparkle:
		return applianceObj(what, 37, 15)
	default:
		return enemyObj(what, 80, 11)
	}
}

// TestOnlyTheSevenAppliancesTrackThePlayOrigin is the coordinate-convention split, derived
// rather than restated.
//
// Registering the same object into two worlds that differ only in their play origin isolates
// the one thing under test: an appliance's Dest moves with the origin because its arm writes
// `where.Left + originH + dx`, and the other ten do not because theirs write `where.Left`. So
// the seven live in screen space and the ten in room space, which is why CheckDynamicCollision
// needs a doOffset flag at all.
//
// This is stronger than checking the numbers in TestRegistrationFieldsPerType, which would
// still pass if a type were moved from one convention to the other and its expected rect
// edited to match.
func TestOnlyTheSevenAppliancesTrackThePlayOrigin(t *testing.T) {
	registerAt := func(originH, originV int16, what int16) Rect {
		h := oneRoomHouse(house.ObjectIsEmpty)
		v := render.DefaultView()
		v.OriginH, v.OriginV = originH, originV
		w := NewWorld(h, render.NewScene(v, render.NewAssets(""), h), 1)
		if got := w.AddDynamicObject(what, testWhere, typicalObject(what),
			testRoom, testSlot, true); got != 0 {
			t.Fatalf("type %#x: AddDynamicObject = %d, want 0", what, got)
		}
		return w.Dinahs[0].Dest
	}

	oh, ov := render.DefaultView().OriginH, render.DefaultView().OriginV
	appliances := 0
	for _, what := range allRegistrableTypes() {
		atZero := registerAt(0, 0, what)
		moved := registerAt(oh, ov, what)

		wantH, wantV := int16(0), int16(0)
		if isApplianceType(what) {
			wantH, wantV = oh, ov
			appliances++
		}
		if gotH, gotV := moved.Left-atZero.Left, moved.Top-atZero.Top; gotH != wantH || gotV != wantV {
			t.Errorf("type %#x: moving the play origin by (%d,%d) moved Dest by (%d,%d), want "+
				"(%d,%d)", what, oh, ov, gotH, gotV, wantH, wantV)
		}
	}
	if appliances != 7 {
		t.Errorf("%d of the seventeen tracked the origin, want 7", appliances)
	}
}

// TestTheSeventeenSplitIntoSevenAndTen pins the count the whole file leans on.
func TestTheSeventeenSplitIntoSevenAndTen(t *testing.T) {
	appliances := 0
	for _, what := range allRegistrableTypes() {
		if isApplianceType(what) {
			appliances++
		}
	}
	if len(allRegistrableTypes()) != 17 || appliances != 7 {
		t.Errorf("%d registrable types of which %d appliances, want 17 and 7",
			len(allRegistrableTypes()), appliances)
	}
}
