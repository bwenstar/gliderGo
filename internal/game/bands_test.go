package game

// The band path: RubberBands.c end to end, plus RenderBands' two registrations.
//
// A rubber band is the only object the player can create, and almost everything interesting
// about it is an interaction between two phases of CheckBandCollision that read as
// independent. So this file is organised by those interactions rather than by function:
//
//   - **The wall and the floor.** Phase 1 clamps and phase 5 kills, and they are one pixel
//     apart. TestBandBouncesOffTheWallAndSurvivesTheKillTest is the pin bands.go names.
//   - **The doorway.** Phase 5 tests the wall *constants* where phase 1 tests the room's
//     thresholds, so a band dies at x=12 even where the room is open. That is a rule of the
//     game and TestBandDiesAtAnOpenDoorway is where it is written down.
//   - **The debounce.** One global for two bands and every hot spot in the room. Three tests
//     here defeat it three different ways, and all three are reachable in the shipped game.
//   - **The gliders.** Two phases that share one `if hVel != 0`, which is why the *second*
//     glider gets a thud and no shove.
//
// The two-glider tests are the ones to read carefully, because the escape protocol's guards
// look inverted and are not. `DeadWhich == Player2` gates glider *one*: player 2 being the
// dead one is exactly what makes player 1 the survivor. See interactions.go for the same note
// on the same idiom.
//
// **What StillOver does to a band test.** HandleSwitches and ArmTrigger both have their own
// once-per-step-on latch, and in a running game it is CheckForHotSpots -- the *glider* sweep
// -- that clears it for rects the glider is not standing on. These tests do not run that
// sweep, so they clear StillOver by hand at the top of each frame, exactly where the glider
// sweep would have. Without that, HandleSwitches' latch masks the band debounce entirely and
// the debounce tests would pass for the wrong reason.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// bandWorld is dynaWorld with both walls present and the gliders left where dynaWorld put
// them: at (-4000,-4000), far enough out that no band test trips the momentum phases by
// accident.
//
// The thresholds are set explicitly rather than composed by DetermineRoomOpenings, because
// "does this room have a wall on this side" is the *input* to phase 1 and half the tests here
// want the other answer. The assertion that they are not already equal to the wall limits
// would be the wrong assertion -- a bare World has them at zero, which is neither -- so what
// is checked instead is that the four constants are still distinct, which is what makes the
// doorway test meaningful.
// **BandHitLast is seeded to -1**, and that is a decision worth explaining rather than a
// convenience. A fresh World has it at 0, which is the original's zeroed global and which
// swallows the first collision against hot spot 0 -- so a fixture that left it alone would make
// every single-hot-spot test in this file silently assert nothing. -1 is not an invented state
// either: it is what the very next frame of free flight writes, so it is the value in force for
// all but the first frame of a session. TestFirstCollisionWithHotSpotZeroIsSwallowed builds its
// own world and owns the zero.
func bandWorld(t *testing.T) *World {
	t.Helper()
	if LeftWallLimit == NoLeftWallLimit || RightWallLimit == NoRightWallLimit {
		t.Fatal("the wall limits and the no-wall limits have to differ or phase 1 and " +
			"phase 5 cannot disagree, and this whole file is about that disagreement")
	}
	w := dynaWorld(t)
	w.R.RoomNumber = 0
	w.R.LeftThresh = LeftWallLimit
	w.R.RightThresh = RightWallLimit
	w.BandHitLast = -1
	return w
}

// putBand installs one band directly, bypassing AddBand: the muzzle geometry is
// TestAddBandGeometryAndRecoil's subject and every other test wants a band somewhere AddBand
// cannot put one.
//
// `left` and `top` are room-local, and the size is fixed at the 16x6 AddBand produces --
// phase 1's clamp writes `right = left + 16` unconditionally, so a band of any other width
// would be silently resized by the first wall it touched.
func putBand(w *World, left, top, hVel, vVel int16) *Band {
	i := w.NumBands
	w.BandList[i] = Band{
		Dest: render.SetRect(left, top, left+16, top+6),
		HVel: hVel, VVel: vVel,
	}
	w.NumBands++
	return &w.BandList[i]
}

// clearStillOvers is the part of CheckForHotSpots these tests need and do not run: the glider
// sweep sets stillOver false on every rect the glider is not touching, once a frame. See the
// file comment.
func clearStillOvers(w *World) {
	for i := range w.R.Hot {
		w.R.Hot[i].StillOver = false
	}
}

// switchPlate wires master[0] as a light switch whose link names room 0's object slot 1, and
// puts a deluxe transporter there for it to toggle.
//
// LocalLink is -1 -- "the target is not in the locale" -- which is the shortest graph that
// still makes SetObjectState report a change and therefore still plays the click.
//
// **The target has to be a transporter and not a prize**, and that is not a cosmetic choice.
// The debounce tests measure *how many times* a band throws one switch, so the click has to
// be countable across frames -- and HandleSwitches plays it only when SetObjectState returns
// true. The prize family (setstate.go:171) computes `changed = state != 0` and then writes
// zero, so a prize reports a change exactly once and is silent for ever after: a test using
// one would see a single click no matter how many times the latch was defeated, and would
// pass whether the debounce worked or not. DeluxeTrans' Toggle arm sets `changed = true`
// unconditionally, and its switchLinkedObject arm is one of the four explicit nothings, so
// each successful throw costs exactly one SwitchSound and nothing else.
func switchPlate(w *World) house.Object {
	plate := house.Object{What: LightSwitch}
	plate.SetSwitch(house.Switch{Who: 1, Type: byte(Toggle)})

	target := house.Object{What: DeluxeTrans}
	target.SetTransport(house.Transport{Where: house.UnlinkedWhere, Wide: 0})

	w.Room(0).Objects[0] = plate
	w.Room(0).Objects[1] = target
	w.Room(0).NumObjects = 2
	w.R.Master = []MasterObject{{
		RoomNum: 0, ObjectNum: 0,
		RoomLink: 0, ObjectLink: 1, LocalLink: -1,
		HotNum: 0, DynaNum: 0, TheObject: plate,
	}}
	return plate
}

// ---------------------------------------------------------------------------
// AddBand
// ---------------------------------------------------------------------------

// TestAddBandGeometryAndRecoil pins the muzzle, the velocity and the kick.
//
// The 32-pixel throw is what keeps a band from hitting the glider that fired it on frame one,
// and the recoil is what makes firing a movement technique -- a player with no battery can
// cross a room on rubber bands alone. Both numbers are asserted absolutely rather than as
// deltas, because both start from zero.
func TestAddBandGeometryAndRecoil(t *testing.T) {
	for _, c := range []struct {
		name     string
		facing   bool
		want     Rect
		wantVel  int16
		wantKick int16
	}{
		{
			name: "right", facing: player.FaceRight,
			// (200,100) is the glider's middle; the band is 16x6 around it and then
			// thrown 32 right, so 200-8+32 = 224.
			want: render.SetRect(224, 97, 240, 103), wantVel: RubberBandVelocity,
			wantKick: -RubberBandVelocity / 2,
		},
		{
			name: "left", facing: player.FaceLeft,
			want: render.SetRect(160, 97, 176, 103), wantVel: -RubberBandVelocity,
			wantKick: RubberBandVelocity / 2,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := bandWorld(t)
			var sounds soundLog
			sounds.install(w)

			if !w.AddBand(&w.P1, 200, 100, c.facing) {
				t.Fatal("AddBand refused the first band")
			}
			if w.NumBands != 1 {
				t.Fatalf("NumBands = %d, want 1", w.NumBands)
			}
			b := &w.BandList[0]
			if b.Dest != c.want {
				t.Errorf("Dest = %+v, want %+v", b.Dest, c.want)
			}
			if b.HVel != c.wantVel {
				t.Errorf("HVel = %d, want %d", b.HVel, c.wantVel)
			}
			if b.VVel != 0 || b.Mode != 0 || b.Count != 0 {
				t.Errorf("VVel/Mode/Count = %d/%d/%d, want 0/0/0 for an untipped shot",
					b.VVel, b.Mode, b.Count)
			}
			if w.P1.HVel != c.wantKick {
				t.Errorf("glider HVel = %d, want %d -- the recoil is half the band's "+
					"velocity, the other way", w.P1.HVel, c.wantKick)
			}
			sounds.is(t, FireBandSound)
		})
	}
}

// TestAddBandTippedFiresUpward pins the one control the player has over a band's arc.
//
// Tipped means the direction key opposite to the facing is held, which is also what makes the
// glider hover rather than accelerate. So the upward shot and the stationary hover are the
// same input, and a player who wants a flat shot has to be moving.
func TestAddBandTippedFiresUpward(t *testing.T) {
	w := bandWorld(t)
	w.P1.Tipped = true
	if !w.AddBand(&w.P1, 200, 100, player.FaceRight) {
		t.Fatal("AddBand refused the first band")
	}
	if got := w.BandList[0].VVel; got != -2 {
		t.Errorf("VVel = %d, want -2 for a tipped glider", got)
	}
}

// TestBandCapIsTwoAndARefusedShotIsFree pins both halves of AddBand's `return false`.
//
// The refusal is what Input.c reads to decide whether to charge the player a band, so a third
// simultaneous shot has to leave the inventory, the recoil and the sound log *all* untouched.
// A stub that returned true and did nothing would have cost the player ammunition for a shot
// that never appeared, which is why the old env.go stub returned false.
func TestBandCapIsTwoAndARefusedShotIsFree(t *testing.T) {
	w := bandWorld(t)
	var sounds soundLog
	sounds.install(w)

	if MaxRubberBands != 2 {
		t.Fatalf("kMaxRubberBands = %d, want 2", MaxRubberBands)
	}
	for i := 0; i < MaxRubberBands; i++ {
		if !w.AddBand(&w.P1, 200, 100, player.FaceRight) {
			t.Fatalf("AddBand refused band %d", i)
		}
	}
	kickAfterTwo := w.P1.HVel

	if w.AddBand(&w.P1, 200, 100, player.FaceRight) {
		t.Error("AddBand accepted a third band; the cap is two")
	}
	if w.NumBands != MaxRubberBands {
		t.Errorf("NumBands = %d, want %d", w.NumBands, MaxRubberBands)
	}
	if w.P1.HVel != kickAfterTwo {
		t.Errorf("glider HVel = %d after the refused shot, want %d -- a refused shot "+
			"must not recoil either", w.P1.HVel, kickAfterTwo)
	}
	sounds.is(t, FireBandSound, FireBandSound)
}

// ---------------------------------------------------------------------------
// HandleBands: the cel, the gravity and the reaping
// ---------------------------------------------------------------------------

// TestBandModeCyclesAndGravityTicksEveryFourthFrame walks eight frames of a band going
// nowhere horizontally.
//
// Two independent counters run here and it is worth seeing them side by side: Mode is the
// three-cel spin and wraps every third frame, Count is the gravity divider and fires every
// fourth. They are coprime, so the pattern does not repeat for twelve frames -- which is why
// a band's fall looks smoother than the glider's despite being coarser.
//
// The band is given HVel 0 so that phases 1 and 3 cannot fire and the only thing moving it is
// the gravity under test.
func TestBandModeCyclesAndGravityTicksEveryFourthFrame(t *testing.T) {
	w := bandWorld(t)
	putBand(w, 200, 100, 0, 0)

	wantMode := []int16{1, 2, 0, 1, 2, 0, 1, 2}

	// Count starts at 0 and is pre-incremented, so `count >= 4` is first true on the
	// *fourth* frame and every fourth frame after it -- 4 and 8, not 3 and 7. The reset to
	// zero rather than a subtract is what makes the period exactly four with no drift.
	wantVVel := []int16{0, 0, 0, 1, 1, 1, 1, 2}

	// The tick and the move are in the same frame -- vVel is incremented above the
	// Offset -- so there is no lag: the first frame vVel is non-zero is the first frame the
	// band descends. The whole eight-frame fall is six pixels, against a glider's ~28.
	wantTop := []int16{100, 100, 100, 101, 102, 103, 104, 106}

	for frame := 0; frame < len(wantMode); frame++ {
		w.HandleBands()
		if w.NumBands != 1 {
			t.Fatalf("frame %d: the band died; NumBands = %d", frame+1, w.NumBands)
		}
		b := &w.BandList[0]
		if b.Mode != wantMode[frame] || b.VVel != wantVVel[frame] || b.Dest.Top != wantTop[frame] {
			t.Errorf("frame %d: Mode/VVel/Top = %d/%d/%d, want %d/%d/%d",
				frame+1, b.Mode, b.VVel, b.Dest.Top,
				wantMode[frame], wantVVel[frame], wantTop[frame])
		}
	}
}

// TestHandleBandsRegistersThePreMoveRect is the erase.
//
// The dirty rect covers where the band *was*, not where it is going, which is what lets
// RestoreWorkMap put the background back before RenderBands draws the new position. Register
// the post-move rect instead and every band leaves a permanent smear down the room -- a bug
// that would look like a compositor problem and be hunted in entirely the wrong file.
func TestHandleBandsRegistersThePreMoveRect(t *testing.T) {
	w := bandWorld(t)
	before := putBand(w, 200, 100, RubberBandVelocity, 0).Dest
	clearRects(w)

	w.HandleBands()

	if w.BandList[0].Dest.Left != before.Left+RubberBandVelocity {
		t.Fatalf("the band did not move: Left = %d, want %d",
			w.BandList[0].Dest.Left, before.Left+RubberBandVelocity)
	}
	want := render.Offset(before, w.R.V.OriginH, w.R.V.OriginV)
	if got, _ := rectCounts(w); got != 1 {
		t.Fatalf("work rects = %d, want 1", got)
	}
	if w.Work2Main[0] != want {
		t.Errorf("work rect = %+v, want %+v (the *pre-move* rect in screen coordinates)",
			w.Work2Main[0], want)
	}
}

// TestRenderBandsRegistersBothLists pins the pair every moving object needs: the work rect
// puts this frame on screen, the back rect asks for the background to be restored here before
// the next one.
//
// RenderBands is last in RenderFrame and therefore draws over everything, gliders included.
// That ordering is not observable without art, so this test asserts the registrations and
// TestRenderFrameOrder in play_test.go owns the position in the sequence.
func TestRenderBandsRegistersBothLists(t *testing.T) {
	w := bandWorld(t)
	dest := putBand(w, 200, 100, RubberBandVelocity, 0).Dest
	clearRects(w)

	w.RenderBands()

	want := render.Offset(dest, w.R.V.OriginH, w.R.V.OriginV)
	work, back := rectCounts(w)
	if work != 1 || back != 1 {
		t.Fatalf("work/back rects = %d/%d, want 1/1", work, back)
	}
	if w.Work2Main[0] != want || w.Back2Work[0] != want {
		t.Errorf("work/back rect = %+v/%+v, want %+v for both",
			w.Work2Main[0], w.Back2Work[0], want)
	}
}

// ---------------------------------------------------------------------------
// The wall, the floor and the doorway
// ---------------------------------------------------------------------------

// TestBandBouncesOffTheWallAndSurvivesTheKillTest is the pin bands.go's header names.
//
// Phase 1 writes `left = kLeftWallLimit` exactly and phase 5 kills on `left < kLeftWallLimit`,
// so the rebounding band survives by one pixel. Change the clamp to `left = limit - 1`, or the
// kill to `<=`, and every band that touches a wall dies on contact instead of bouncing -- and
// because a band that dies is simply gone, the symptom would be "bands sometimes do not
// bounce", which is a much harder thing to find than this test failing.
func TestBandBouncesOffTheWallAndSurvivesTheKillTest(t *testing.T) {
	for _, c := range []struct {
		name      string
		left, vel int16
		wantLeft  int16
	}{
		{"left wall", LeftWallLimit + 8, -RubberBandVelocity, LeftWallLimit},
		{"right wall", RightWallLimit - 24, RubberBandVelocity, RightWallLimit - 16},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := bandWorld(t)
			var sounds soundLog
			sounds.install(w)
			putBand(w, c.left, 100, c.vel, 0)

			w.HandleBands()

			if w.NumBands != 1 {
				t.Fatalf("the band died on the frame it rebounded; NumBands = %d. "+
					"Phase 1's clamp and phase 5's kill are one pixel apart and this "+
					"is that pixel", w.NumBands)
			}
			b := &w.BandList[0]
			if b.Dest.Left != c.wantLeft || b.Dest.Right != c.wantLeft+16 {
				t.Errorf("Dest = %+v, want left %d and a 16-wide band",
					b.Dest, c.wantLeft)
			}
			if b.HVel != -c.vel {
				t.Errorf("HVel = %d, want %d -- the wall negates it", b.HVel, -c.vel)
			}
			sounds.is(t, BandReboundSound)
		})
	}
}

// TestBandDiesAtAnOpenDoorway is the rule of the game hidden in phase 5's choice of constant.
//
// With the wall absent the room's threshold is NoLeftWallLimit (-24), so phase 1 declines to
// clamp -- and phase 5 still tests kLeftWallLimit (12) and deletes the band. A band therefore
// cannot be fired through a doorway into the next room, which is what keeps bands a
// within-room tool: a player cannot clear a room they cannot see.
//
// The rebound sound is asserted absent as well as the band being gone, because that is the
// difference between "phase 1 did not run" and "phase 1 ran and phase 5 undid it".
func TestBandDiesAtAnOpenDoorway(t *testing.T) {
	for _, c := range []struct {
		name      string
		left, vel int16
		setup     func(*World)
	}{
		{"left doorway", LeftWallLimit + 8, -RubberBandVelocity,
			func(w *World) { w.R.LeftThresh = NoLeftWallLimit }},
		{"right doorway", RightWallLimit - 24, RubberBandVelocity,
			func(w *World) { w.R.RightThresh = NoRightWallLimit }},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := bandWorld(t)
			c.setup(w)
			var sounds soundLog
			sounds.install(w)
			putBand(w, c.left, 100, c.vel, 0)

			w.HandleBands()

			if w.NumBands != 0 {
				t.Errorf("NumBands = %d, want 0 -- a band cannot leave through a "+
					"doorway", w.NumBands)
			}
			sounds.is(t)
		})
	}
}

// TestBandDiesOnTheFloor is phase 5's `else if`: below kFloorLimit and the band is gone.
//
// It is an else-if rather than a second test, so a band that is both out of bounds sideways
// and below the floor is killed by the first arm. Nothing observable depends on which, since
// the outcome is the same value -- but it does mean the floor arm is unreachable for a band
// leaving through a doorway, which is why this test gives the band no horizontal velocity.
func TestBandDiesOnTheFloor(t *testing.T) {
	w := bandWorld(t)
	// Count 3 so the gravity tick fires this frame and carries the band past the floor in
	// one step, rather than needing four frames of setup to prove one branch.
	b := putBand(w, 200, FloorLimit-12, 0, 10)
	b.Count = BandFallCount - 1

	w.HandleBands()

	if w.NumBands != 0 {
		t.Errorf("NumBands = %d, want 0 -- the band fell past y=%d", w.NumBands, FloorLimit)
	}
}

// TestBothBandsDyingOnOneFrameAreBothReaped exercises the reaping loop's inner `while`.
//
// KillBand fills the hole with the *last* band, which may itself be dying, so the sweep has
// to re-test the same index rather than advance -- and the `mode = 0` before the kill, which
// looks dead because KillBand overwrites the slot, is what terminates that inner loop when
// the slot being killed *is* the last one. Remove it and this test hangs rather than fails,
// which is why it is worth having a case where both bands die on the same frame.
//
// The second half of the test is the vacated slot, and it is the opposite assertion from the
// one that looks right. KillBand copies the last band down and decrements numBands without
// clearing what it copied *from*, so after both bands die slot 1 still reads -1 for ever.
// That is faithful and it is also the reason AddBand opens with `mode = 0` instead of trusting
// the slot it is handed: a band fired into slot 1 after this frame would otherwise be born
// already marked for deletion and would be reaped before it was ever drawn.
func TestBothBandsDyingOnOneFrameAreBothReaped(t *testing.T) {
	w := bandWorld(t)
	for i := 0; i < MaxRubberBands; i++ {
		b := putBand(w, 200+int16(i)*32, FloorLimit-12, 0, 10)
		b.Count = BandFallCount - 1
	}
	if w.NumBands != MaxRubberBands {
		t.Fatalf("fixture put %d bands in the air, want %d", w.NumBands, MaxRubberBands)
	}

	w.HandleBands()

	if w.NumBands != 0 {
		t.Errorf("NumBands = %d, want 0 -- both bands were below the floor", w.NumBands)
	}
	if w.BandList[0].Mode == KillBandMode {
		t.Errorf("slot 0 still holds the kill sentinel; the inner while did not re-test it " +
			"after KillBand moved the second dying band into it")
	}
	if w.BandList[MaxRubberBands-1].Mode != KillBandMode {
		t.Errorf("slot %d = %d, want the stale %d: KillBand does not clear the slot it "+
			"copies out of, and AddBand's `mode = 0` is what covers for that",
			MaxRubberBands-1, w.BandList[MaxRubberBands-1].Mode, KillBandMode)
	}

	// And the cover works: the next shot into that slot is a live band.
	g := &w.P1
	g.Dest = player.Rect{Top: 90, Left: 200, Bottom: 110, Right: 248}
	for i := 0; i < MaxRubberBands; i++ {
		if !w.AddBand(g, 224, 100, player.FaceRight) {
			t.Fatalf("AddBand refused shot %d into a table of %d empty slots", i, MaxRubberBands)
		}
	}
	for i := range w.BandList {
		if w.BandList[i].Mode == KillBandMode {
			t.Errorf("slot %d is still marked for deletion after AddBand wrote it", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Obstacles
// ---------------------------------------------------------------------------

// TestBandBouncesOffAnObstacleItReached and TestBandIsAbsorbedByAnObstacleItWasInside are the
// two halves of one test in the source: `dest.right - hVel < bounds.left`.
//
// The subtraction reconstructs where the band was *before* this frame's step. If it was clear
// of the obstacle's face, the step drove it into that face and it rebounds; if it was already
// overlapping, it is inside the thing and is absorbed. At twenty pixels a frame both happen
// often, and the difference is visible: a band that bounces off a table leg comes back and a
// band fired point-blank into one vanishes.
func TestBandBouncesOffAnObstacleItReached(t *testing.T) {
	w := bandWorld(t)
	var sounds soundLog
	sounds.install(w)
	w.R.Hot = []HotObject{{
		Bounds: render.SetRect(200, 90, 260, 130), Action: BounceIt, Who: -1, IsOn: true,
	}}
	// Pre-move right edge is 190, clear of the obstacle's left edge at 200; the step
	// carries it to 210 and into the face.
	putBand(w, 174, 100, RubberBandVelocity, 0)

	w.HandleBands()

	if w.NumBands != 1 {
		t.Fatalf("the band was absorbed; it should have bounced. NumBands = %d", w.NumBands)
	}
	b := &w.BandList[0]
	if b.HVel != -RubberBandVelocity {
		t.Errorf("HVel = %d, want %d", b.HVel, -RubberBandVelocity)
	}
	if b.Dest.Right != 200 || b.Dest.Left != 184 {
		t.Errorf("Dest = %+v, want the band flush against the obstacle's left edge at 200",
			b.Dest)
	}
	sounds.is(t, BandReboundSound)
}

func TestBandIsAbsorbedByAnObstacleItWasInside(t *testing.T) {
	w := bandWorld(t)
	w.R.Hot = []HotObject{{
		Bounds: render.SetRect(200, 90, 260, 130), Action: BounceIt, Who: -1, IsOn: true,
	}}
	// Pre-move right edge is 212, already past the obstacle's left edge at 200.
	putBand(w, 196, 100, RubberBandVelocity, 0)

	w.HandleBands()

	if w.NumBands != 0 {
		t.Errorf("NumBands = %d, want 0 -- a band already inside an obstacle is absorbed, "+
			"not bounced", w.NumBands)
	}
}

// TestBounceStopsTheSweep pins the one arm of the five that leaves the hot-spot loop.
//
// It has to: the arm has just moved the band's rect, so continuing the sweep would collide
// the band against the *next* obstacle from its new position. The port spells this as a
// labelled `break sweep` because the arm sits in a Go switch, where a bare break would leave
// only the switch -- see bands.go. The observable is that the second obstacle's rebound sound
// never plays.
func TestBounceStopsTheSweep(t *testing.T) {
	w := bandWorld(t)
	var sounds soundLog
	sounds.install(w)
	// Two obstacles the band's post-move rect overlaps at once.
	w.R.Hot = []HotObject{
		{Bounds: render.SetRect(200, 90, 260, 130), Action: BounceIt, Who: -1, IsOn: true},
		{Bounds: render.SetRect(205, 90, 265, 130), Action: BounceIt, Who: -1, IsOn: true},
	}
	putBand(w, 174, 100, RubberBandVelocity, 0)

	w.HandleBands()

	sounds.is(t, BandReboundSound)
	if w.BandHitLast != 0 {
		t.Errorf("BandHitLast = %d, want 0 -- the sweep stopped at the first obstacle",
			w.BandHitLast)
	}
}

// ---------------------------------------------------------------------------
// Prizes and grease
// ---------------------------------------------------------------------------

// greaseRewardWorld builds the smallest graph the kRewardIt arm walks: a hot spot at `at`
// whose Who names master[0], master[0] naming room 0's object slot 1, and a matching entry in
// the render side's grease table for SpillGrease to find.
func greaseRewardWorld(t *testing.T, what int16, at Rect) *World {
	t.Helper()
	w := bandWorld(t)

	obj := house.Object{What: what}
	obj.SetBonus(house.Bonus{State: 1, Initial: 1, Points: 100})
	w.Room(0).Objects[1] = obj
	w.Room(0).NumObjects = 2

	w.R.Master = []MasterObject{{
		RoomNum: 0, ObjectNum: 1,
		RoomLink: -1, ObjectLink: -1, LocalLink: -1,
		HotNum: 0, DynaNum: 0, TheObject: obj,
	}}
	w.R.Hot = []HotObject{{Bounds: at, Action: RewardIt, Who: 0, IsOn: true}}
	w.R.Grease = []render.Grease{{
		Where: 0, Who: 0, Mode: render.GreaseIdle, Frame: -1,
		Dest: render.SetRect(0, 0, 32, 27),
	}}
	if w.R.Master[0].HotNum != 0 {
		t.Fatal("the fixture's HotNum has to name the reward rect; SpillGrease is handed " +
			"the master's HotNum and HandleGrease writes the slide rect through it")
	}
	return w
}

// TestBandSpillsGreaseButCannotCollectAPrize is the design decision hidden in a type test.
//
// The kRewardIt arm checks that the linked object is kGreaseRt or kGreaseLf and does nothing
// otherwise, so a band knocks a jar over and passes straight through a clock, a battery and a
// star. Remove the type test and a player could farm a room's prizes from across it without
// ever flying there, which is a different game.
//
// The clock row asserts that *nothing at all* happened -- not the state byte, not the rect's
// IsOn, not a sound -- because the arm's failure mode is a partial effect: switching the rect
// off without collecting would make the prize disappear unclaimed.
func TestBandSpillsGreaseButCannotCollectAPrize(t *testing.T) {
	at := render.SetRect(200, 90, 260, 130)

	t.Run("grease spills", func(t *testing.T) {
		w := greaseRewardWorld(t, GreaseRt, at)
		var sounds soundLog
		sounds.install(w)
		putBand(w, 174, 100, RubberBandVelocity, 0)

		w.HandleBands()

		if got := w.R.Grease[0].Mode; got != render.GreaseFalling {
			t.Errorf("grease mode = %d, want %d (falling)", got, render.GreaseFalling)
		}
		if got := w.R.Grease[0].HotNum; got != 0 {
			t.Errorf("grease HotNum = %d, want 0 -- SpillGrease is handed the *master's* "+
				"HotNum, which is how HandleGrease finds the slide rect to write", got)
		}
		if w.R.Hot[0].IsOn {
			t.Error("the reward rect is still on; a spilt jar must not be spillable twice")
		}
		if got := w.Room(0).Objects[1].Bonus().State; got != 0 {
			t.Errorf("the jar's state byte = %d, want 0", got)
		}
		sounds.is(t, GreaseSpillSound)
	})

	t.Run("clock is untouched", func(t *testing.T) {
		w := greaseRewardWorld(t, RedClock, at)
		var sounds soundLog
		sounds.install(w)
		putBand(w, 174, 100, RubberBandVelocity, 0)

		w.HandleBands()

		if !w.R.Hot[0].IsOn {
			t.Error("the reward rect was switched off for a clock; a band cannot collect " +
				"a prize and must not make it vanish either")
		}
		if got := w.Room(0).Objects[1].Bonus().State; got != 1 {
			t.Errorf("the clock's state byte = %d, want 1 -- untouched", got)
		}
		if got := w.R.Grease[0].Mode; got != render.GreaseIdle {
			t.Errorf("grease mode = %d, want idle", got)
		}
		sounds.is(t)
	})
}

// ---------------------------------------------------------------------------
// The debounce, defeated three ways
// ---------------------------------------------------------------------------

// TestFirstCollisionWithHotSpotZeroIsSwallowed is the smallest of the three.
//
// bandHitLast is a zeroed global rather than -1, and KillAllBands does not reset it, so the
// first band collision of a session against hot spot **0** matches the latch and is
// suppressed. Go's zero value reproduces this for free, which is why World.BandHitLast has no
// initialiser -- see the field's comment.
//
// This is the one test in the file that does not go through bandWorld, because bandWorld seeds
// the latch to -1 precisely so that the swallow does not silently absorb every other test's
// first frame. Here the fresh zero is the subject, and the control is the frame after: once the
// latch has been cleared by a band touching nothing, the same band on the same rect does fire.
// So the swallow costs one click, not the hot spot.
func TestFirstCollisionWithHotSpotZeroIsSwallowed(t *testing.T) {
	w := dynaWorld(t)
	w.R.RoomNumber = 0
	w.R.LeftThresh = LeftWallLimit
	w.R.RightThresh = RightWallLimit

	var sounds soundLog
	sounds.install(w)
	switchPlate(w)
	w.R.Hot = []HotObject{{
		Bounds: render.SetRect(190, 90, 260, 130), Action: SwitchIt, Who: 0, IsOn: true,
	}}
	if w.BandHitLast != 0 {
		t.Fatalf("BandHitLast = %d on a fresh World, want 0 -- the swallow is the *point* "+
			"of that zero value and an initialiser on the field would hide it", w.BandHitLast)
	}
	// HVel 0 so the band stays on the plate and neither wall nor glider phase fires.
	putBand(w, 200, 100, 0, 0)

	clearStillOvers(w)
	w.HandleBands()
	sounds.is(t)
	if w.BandHitLast != 0 {
		t.Fatalf("BandHitLast = %d, want 0 still", w.BandHitLast)
	}

	// The control. Nothing about the band changed; only the latch did.
	w.BandHitLast = -1
	clearStillOvers(w)
	w.HandleBands()
	sounds.is(t, SwitchSound)
}

// TestBandDebounceHoldsForOneRect is the case the latch was written for, and the only one it
// handles: one band, one hot spot, three frames, one click.
func TestBandDebounceHoldsForOneRect(t *testing.T) {
	w := bandWorld(t)
	var sounds soundLog
	sounds.install(w)
	switchPlate(w)
	w.R.Hot = []HotObject{
		{Bounds: render.SetRect(190, 90, 260, 130), Action: SwitchIt, Who: 0, IsOn: true},
	}
	putBand(w, 200, 100, 0, 0)

	for frame := 0; frame < 3; frame++ {
		clearStillOvers(w)
		w.HandleBands()
	}
	sounds.is(t, SwitchSound)
}

// TestSecondBandDefeatsTheDebounce is the easy one to hit in a real game: fire twice, hold one
// band against a switch, and the switch chatters.
//
// bandHitLast is one global and the reset is per *band*. The free-flying band collides with
// nothing, sets nothingCollided, and clears the latch to -1 -- so the parked band's
// suppression is undone every frame by a band on the other side of the room. A player with two
// bands in the air gets a switch throwing sixty times a second.
//
// Compare against TestBandDebounceHoldsForOneRect: same three frames, same one plate, one
// click there and three here. Nothing about the second band touches the switch.
func TestSecondBandDefeatsTheDebounce(t *testing.T) {
	w := bandWorld(t)
	var sounds soundLog
	sounds.install(w)
	switchPlate(w)
	w.R.Hot = []HotObject{
		{Bounds: render.SetRect(190, 90, 260, 130), Action: SwitchIt, Who: 0, IsOn: true},
	}
	putBand(w, 200, 100, 0, 0) // parked on the plate
	putBand(w, 300, 200, 0, 0) // free air, touching nothing

	for frame := 0; frame < 3; frame++ {
		clearStillOvers(w)
		w.HandleBands()
	}
	sounds.is(t, SwitchSound, SwitchSound, SwitchSound)
}

// TestTwoRectsDefeatTheDebounceForEachOther is the subtlest of the three and needs no second
// band at all.
//
// The latch holds one index. A band overlapping two switch plates trips the first, which sets
// the latch to 1; the sweep continues to the second, which does not match 1, so it trips too
// and leaves the latch at 2. Next frame the first does not match 2 either. The two plates
// alternate the latch between them for ever and both chatter.
//
// It needs the two rects within sixteen pixels of each other, which is close, but a
// light switch and a trigger plate on the same wall would do it -- and the kSwitchIt and
// kTriggerIt arms are both in the filtered five, so a switch and a fuse count as two rects for
// this purpose.
func TestTwoRectsDefeatTheDebounceForEachOther(t *testing.T) {
	w := bandWorld(t)
	var sounds soundLog
	sounds.install(w)
	switchPlate(w)
	w.R.Hot = []HotObject{
		{Bounds: render.SetRect(190, 90, 260, 130), Action: SwitchIt, Who: 0, IsOn: true},
		{Bounds: render.SetRect(195, 90, 265, 130), Action: SwitchIt, Who: 0, IsOn: true},
	}
	putBand(w, 200, 100, 0, 0)

	for frame := 0; frame < 2; frame++ {
		clearStillOvers(w)
		w.HandleBands()
	}
	// Four clicks where the latch intends one.
	sounds.is(t, SwitchSound, SwitchSound, SwitchSound, SwitchSound)
}

// TestKillAllBandsLeavesTheDebounceLatched is the third failure's other half.
//
// A room change deletes every band in flight and does *not* reset bandHitLast, so the latch
// arrives in the new room still naming a hot spot from the old one -- and hot-spot indices are
// per-locale, so it names whatever now happens to sit at that index. The port keeps this
// because the alternative is a table that disagrees with the original's on the first frame of
// every room.
func TestKillAllBandsLeavesTheDebounceLatched(t *testing.T) {
	w := bandWorld(t)
	putBand(w, 200, 100, RubberBandVelocity, 0)
	putBand(w, 300, 100, -RubberBandVelocity, 0)
	w.BandHitLast = 7

	w.KillAllBands()

	if w.NumBands != 0 {
		t.Errorf("NumBands = %d, want 0", w.NumBands)
	}
	for i := range w.BandList {
		if w.BandList[i].Mode != 0 {
			t.Errorf("slot %d Mode = %d, want 0 -- KillAllBands zeroes all "+
				"kMaxRubberBands, not just the live ones", i, w.BandList[i].Mode)
		}
	}
	if w.BandHitLast != 7 {
		t.Errorf("BandHitLast = %d, want 7 -- KillAllBands does not reset it, and the "+
			"stale index crossing into the next room is the original's behaviour",
			w.BandHitLast)
	}
}

// ---------------------------------------------------------------------------
// The gliders
// ---------------------------------------------------------------------------

// bandAtGlider parks glider 1 (and optionally glider 2) where a band fired right from
// (174,100) will reach them, and returns the world.
func bandAtGlider(t *testing.T, twoPlayer bool) *World {
	t.Helper()
	w := bandWorld(t)
	w.TwoPlayer = twoPlayer
	w.P1.Dest = player.Rect{Top: 90, Left: 200, Bottom: 130, Right: 250}
	w.P2.Dest = w.P1.Dest
	w.P1.HVel, w.P2.HVel = 0, 0
	putBand(w, 174, 100, RubberBandVelocity, 0)
	return w
}

// TestBandShovesTheGlider pins the momentum transfer: half the band's velocity to the glider,
// and the band stops dead.
//
// It does not bounce off a glider -- hVel goes to zero and the band falls straight down from
// there, which is what makes a point-blank shot at a partner both a shove and a wasted band.
func TestBandShovesTheGlider(t *testing.T) {
	w := bandAtGlider(t, false)
	var sounds soundLog
	sounds.install(w)

	w.HandleBands()

	if w.P1.HVel != RubberBandVelocity/2 {
		t.Errorf("glider HVel = %d, want %d", w.P1.HVel, RubberBandVelocity/2)
	}
	if got := w.BandList[0].HVel; got != 0 {
		t.Errorf("band HVel = %d, want 0 -- a band sticks to a glider rather than "+
			"bouncing off it", got)
	}
	sounds.is(t, player.HitWallSound)
}

// TestBandCannotShoveBothGlidersButThudsTwice is the consequence of phases 3 and 4 sharing one
// `if hVel != 0`.
//
// The guard is evaluated once, at the top, so glider 2's arm still runs after glider 1 zeroed
// the velocity -- and `hVel / 2` is then 0. Glider 2 gains nothing, but the sound sits beside
// the transfer rather than being conditional on it having any effect, so it plays anyway:
// **two thuds, one shove.** In a two-player game with the gliders touching, one player is
// pushed and both hear the hit.
//
// The escape guard is what decides whether the arm runs at all, and here neither player is
// dead so both guards pass. TestOnlyTheSurvivingGliderIsShoved is the case where one does not.
func TestBandCannotShoveBothGlidersButThudsTwice(t *testing.T) {
	w := bandAtGlider(t, true)
	var sounds soundLog
	sounds.install(w)

	w.HandleBands()

	if w.P1.HVel != RubberBandVelocity/2 {
		t.Errorf("glider 1 HVel = %d, want %d", w.P1.HVel, RubberBandVelocity/2)
	}
	if w.P2.HVel != 0 {
		t.Errorf("glider 2 HVel = %d, want 0 -- glider 1 already zeroed the band's "+
			"velocity and half of zero is zero", w.P2.HVel)
	}
	sounds.is(t, player.HitWallSound, player.HitWallSound)
}

// TestOnlyTheSurvivingGliderIsShoved pins the escape protocol's two guards, which read
// backwards and are correct.
//
// `DeadWhich == Player2` gates glider **one**. Player 2 being the dead one is exactly what
// makes player 1 the survivor, so the test that looks like it names the wrong glider names the
// right one. The consequence is what matters: a dead player's corpse cannot be pushed around
// by the survivor's rubber bands.
//
// One thud, not two, and that is the sharp end of the test. The sound is *inside* the escape
// guard, so the arm the guard closes is silent as well as inert -- unlike the case next door,
// where the second arm runs, thuds and transfers zero. So the two tests together pin the
// difference between "the arm ran and had no effect" and "the arm did not run": the first is
// audible, the second is not. Move the sound above the guard to tidy it and only this test
// notices.
//
// The interesting half is the case that survives *and* the case that shoves being different
// gliders across the two rows, which is what makes the backwards-reading guards observable at
// all: get either one wrong and one row of this table swaps its two velocities.
func TestOnlyTheSurvivingGliderIsShoved(t *testing.T) {
	for _, c := range []struct {
		name           string
		dead           bool
		wantP1, wantP2 int16
		wantThuds      int
	}{
		{"player 1 is dead", player.Player1, 0, RubberBandVelocity / 2, 1},
		{"player 2 is dead", player.Player2, RubberBandVelocity / 2, 0, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := bandAtGlider(t, true)
			w.OneLeft = true
			w.DeadWhich = c.dead
			var sounds soundLog
			sounds.install(w)

			w.HandleBands()

			if w.P1.HVel != c.wantP1 || w.P2.HVel != c.wantP2 {
				t.Errorf("glider HVels = %d/%d, want %d/%d",
					w.P1.HVel, w.P2.HVel, c.wantP1, c.wantP2)
			}
			if len(sounds.played) != c.wantThuds {
				t.Errorf("sounds %v, want %d thud -- the sound is inside the escape "+
					"guard, so the closed arm is silent", sounds.played, c.wantThuds)
			}
		})
	}
}

// TestABandTravellingStraightDownHitsNobody pins the `if hVel != 0` guard itself.
//
// A band that has already spent its momentum on one glider, or one fired at a wall and stopped
// by it, passes through a glider without touching it. That is not an edge case: it is what
// happens to the *second* glider in every two-player collision above, and it is why a band
// resting on a glider's back does nothing.
func TestABandTravellingStraightDownHitsNobody(t *testing.T) {
	w := bandAtGlider(t, false)
	w.BandList[0].HVel = 0
	// Put it inside the glider so the only thing stopping the transfer is the guard.
	w.BandList[0].Dest = render.SetRect(210, 100, 226, 106)
	var sounds soundLog
	sounds.install(w)

	w.HandleBands()

	if w.P1.HVel != 0 {
		t.Errorf("glider HVel = %d, want 0", w.P1.HVel)
	}
	sounds.is(t)
}

// ---------------------------------------------------------------------------
// The filter
// ---------------------------------------------------------------------------

// TestOnlyFiveActionsSeeABand pins the filter at the top of the hot-spot sweep.
//
// Twenty-eight actions exist and a band is offered to five. The other twenty-three are the
// glider's alone, and the list is worth pinning rather than trusting because the failure is
// silent in both directions: adding kShredIt would let a band shred itself, and dropping
// kTriggerIt would make some houses' fuses unreachable by band with nothing to show for it.
//
// The observable is BandHitLast, which is written just above the action dispatch and so
// records "the sweep accepted this rect" without depending on what the arm then did -- which
// matters, because Who is -1 here and three of the five arms bail immediately on that.
//
// **It has to be read as `== 0` and not as `!= seed`.** The sweep ends with
// `if nothingCollided { bandHitLast = -1 }`, so a filtered action leaves the latch at -1
// rather than at whatever the test put there: a probe watching for the seed to *change* reads
// that reset as a hit and reports all 28 actions as reached. Seeding -1 and looking for the
// hot spot's own index 0 is the version the reset cannot forge.
func TestOnlyFiveActionsSeeABand(t *testing.T) {
	reached := map[int16]bool{
		DissolveIt: true, RewardIt: true, SwitchIt: true, TriggerIt: true, BounceIt: true,
	}

	// All 28, driven off NumHotSpotActions rather than a written-out list, so that a
	// twenty-ninth added to consts.go without a decision about bands fails here rather
	// than in a house.
	for action := int16(0); action < NumHotSpotActions; action++ {
		w := bandWorld(t)
		// Who -1 so no arm can walk the master graph and the only thing measured is
		// whether the arm was entered at all.
		w.R.Hot = []HotObject{{
			Bounds: render.SetRect(190, 90, 260, 130), Action: action, Who: -1, IsOn: true,
		}}
		putBand(w, 200, 100, 0, 0)

		w.HandleBands()

		got := w.BandHitLast == 0
		if got != reached[action] {
			t.Errorf("%s: band reached the arm = %v (BandHitLast = %d), want %v",
				ActionName(action), got, w.BandHitLast, reached[action])
		}
	}
}
