package player

import "testing"

// newGliderAtRest is a glider sitting still, with Dest and DestShadow set the way
// InitGlider leaves them (Play.c:354-356): a 48x20 body, and a 48x9 shadow whose
// top is pinned at ShadowTop and which shares the body's left edge.
func newGliderAtRest(left, top int16) *Glider {
	g := &Glider{
		Dest: Rect{Top: top, Left: left, Bottom: top + GliderHigh, Right: left + GliderWide},
		DestShadow: Rect{
			Top: ShadowTop, Left: left,
			Bottom: ShadowTop + ShadowHigh, Right: left + GliderWide,
		},
		Facing: FaceRight,
		Which:  Player1,
	}
	g.Whole = g.Dest
	g.WholeShadow = g.DestShadow
	return g
}

// hold pins both velocities and both desired velocities to the same values, so
// the ramp is a no-op and the frame under test moves by exactly h and v.
//
// Needing this at all is the point of the whole design: there is no way to give
// the glider a velocity that survives a frame without also asserting the desired
// velocity, because MoveGlider ramps toward the desired value and then overwrites
// it. Tests that set only HVel measure the ramp, not the thing they meant to.
func (g *Glider) hold(h, v int16) {
	g.HVel, g.HDesiredVel = h, h
	g.VVel, g.VDesiredVel = v, v
}

// TestFreeFallTrace is player-physics.md §7.3's first worked trace: free fall from
// rest with no input. VVel must be 2 on the first frame, 3 on the second -- the
// overshoot clamp stopping it there rather than at 4 -- and 3 forever after, so
// the terminal fall speed is 3 px/frame reached in two frames.
//
// This is the steady state, which is what holds on every frame from the second one
// onwards: VDesiredVel is Gravity because MoveGlider itself reset it at the end of
// the previous call. The very first frame after a mode change is different, because
// the mode-entry functions leave VDesiredVel at 0; see TestFreeFallFromRespawn.
func TestFreeFallTrace(t *testing.T) {
	g := newGliderAtRest(100, 50)
	g.VDesiredVel = Gravity // as the previous MoveGlider would have left it

	want := []struct {
		vVel     int16
		topDelta int16
	}{
		{2, 2}, // +VImpulse
		{3, 3}, // +VImpulse would reach 4; clamped to VDesiredVel
		{3, 3},
		{3, 3},
		{3, 3},
	}
	for i, w := range want {
		before := g.Dest.Top
		g.MoveGlider()
		if g.VVel != w.vVel {
			t.Errorf("frame %d: VVel = %d, want %d", i+1, g.VVel, w.vVel)
		}
		if got := g.Dest.Top - before; got != w.topDelta {
			t.Errorf("frame %d: Dest.Top moved %d, want %d", i+1, got, w.topDelta)
		}
	}
	// Terminal fall is 3 px/frame = 90 px/s at 30 fps, so a 322px room takes
	// about 108 frames to fall through. Guard the arithmetic behind that claim.
	if TileHigh/Gravity != 107 {
		t.Errorf("full-room fall is %d frames, the analysis says ~108", TileHigh/Gravity)
	}
}

// TestFreeFallFromRespawn is the same fall entered the way the game actually enters
// it, through FlagGliderNormal rather than from a hand-set VDesiredVel.
//
// FlagGliderNormal leaves VDesiredVel at 0, not Gravity (Modes.c:354, via stopDead),
// so the first frame ramps toward 0, finds it is already there, moves nothing, and
// only then resets VDesiredVel to Gravity. The glider gets one frame of hang and the
// trace is 0, 2, 3, 3 -- one frame later than TestFreeFallTrace's steady state.
//
// That single frame is worth a test because it is exactly the kind of detail a port
// "tidies up" by initialising VDesiredVel to Gravity for consistency, which would
// make every respawn, every arrival from a staircase and every mail delivery start
// falling a frame early.
func TestFreeFallFromRespawn(t *testing.T) {
	g := newGliderAtRest(100, 50)
	g.FlagGliderNormal(&NopEnv{})

	if g.VDesiredVel != 0 {
		t.Fatalf("FlagGliderNormal left VDesiredVel = %d, want 0", g.VDesiredVel)
	}

	for i, want := range []int16{0, 2, 3, 3, 3} {
		before := g.Dest.Top
		g.MoveGlider()
		if g.VVel != want {
			t.Errorf("frame %d: VVel = %d, want %d", i+1, g.VVel, want)
		}
		if got := g.Dest.Top - before; got != want {
			t.Errorf("frame %d: Dest.Top moved %d, want %d", i+1, got, want)
		}
	}
}

// TestHoldRightTrace is §7.3's second and third traces: holding a direction key
// from rest ramps HVel 2, 4, 5 and stops at exactly 5 rather than overshooting to
// 6; releasing it decays 5, 3, 1, 0 over three frames.
//
// NormalThrust has to be re-applied on every frame because MoveGlider zeroes
// HDesiredVel itself. A port that set it once would see the glider coast to a
// stop, which is the most likely way to get this whole subsystem wrong.
func TestHoldRightTrace(t *testing.T) {
	g := newGliderAtRest(100, 50)

	for i, want := range []int16{2, 4, 5, 5, 5} {
		g.HDesiredVel = NormalThrust // GetInput does this every frame
		g.MoveGlider()
		if g.HVel != want {
			t.Errorf("holding right, frame %d: HVel = %d, want %d", i+1, g.HVel, want)
		}
	}
	// Cruise speed is 5 px/frame = 150 px/s.
	if g.HVel != NormalThrust {
		t.Fatalf("cruise HVel = %d, want NormalThrust %d", g.HVel, NormalThrust)
	}

	for i, want := range []int16{3, 1, 0, 0} {
		g.MoveGlider() // key released: HDesiredVel stays 0
		if g.HVel != want {
			t.Errorf("released, frame %d: HVel = %d, want %d", i+1, g.HVel, want)
		}
	}
}

// TestHeliumTrace is §7.3's fourth trace: helium from terminal fall ramps
// 3, 1, -1, -3, -4 -- four frames to full lift -- and rise speed is 4 px/frame,
// slightly faster than the fall speed of 3.
func TestHeliumTrace(t *testing.T) {
	g := newGliderAtRest(100, 50)
	g.VVel = Gravity // already at terminal fall

	for i, want := range []int16{1, -1, -3, -4, -4} {
		g.VDesiredVel = -HeliumLift // DoHeliumEngaged does this every frame
		g.MoveGlider()
		if g.VVel != want {
			t.Errorf("helium frame %d: VVel = %d, want %d", i+1, g.VVel, want)
		}
	}
	if HeliumLift <= Gravity {
		t.Errorf("HeliumLift %d should exceed Gravity %d: rising is faster than falling",
			HeliumLift, Gravity)
	}
}

// TestNoVerticalClamp pins §7.2 consequence 1. A ceiling vent assigns
// CeilingVentDrop (8) to VDesiredVel, which is well past anything the ramp would
// otherwise produce. Because it is a target and not a kick, the 8 must be reached
// over the three-frame ramp rather than felt at once, held for as long as the vent
// keeps re-asserting it, and only then decayed by VImpulse per frame back to Gravity
// -- which is what the two loops below assert. That the 8 is reached *in full* is
// the no-clamp property; the 40 px case at the end of this test pins it separately,
// where a clamp of any kind would show up.
func TestNoVerticalClamp(t *testing.T) {
	g := newGliderAtRest(100, 50)
	g.VVel = Gravity

	// Under the vent: VDesiredVel is 8, so the ramp climbs toward it.
	for i, want := range []int16{5, 7, 8, 8} {
		g.VDesiredVel = CeilingVentDrop
		g.MoveGlider()
		if g.VVel != want {
			t.Errorf("under vent, frame %d: VVel = %d, want %d", i+1, g.VVel, want)
		}
	}
	// Out from under it: decays by 2 per frame back to Gravity, not instantly.
	for i, want := range []int16{6, 4, 3, 3} {
		g.MoveGlider()
		if g.VVel != want {
			t.Errorf("past vent, frame %d: VVel = %d, want %d", i+1, g.VVel, want)
		}
	}

	// And a velocity far larger than the horizontal limit must be applied in
	// full. 40 px/frame is twice MaxHVel; a symmetric vertical clamp added "for
	// safety" would silently trim it to 16 and nothing else in the port would
	// complain.
	g2 := newGliderAtRest(100, 50)
	g2.hold(0, 40)
	top := g2.Dest.Top
	g2.MoveGlider()
	if got := g2.Dest.Top - top; got != 40 {
		t.Errorf("VVel 40 moved %d px, want 40: there is no vertical clamp", got)
	}
}

// TestHorizontalClampInsideSignBranches pins §7.2 consequences 2 and 3. The
// battery adds HyperThrust straight to HVel, bypassing the ramp, so HVel can
// arrive at MoveGlider already past the limit; the clamp still fires because the
// branch is chosen on the sign of HVel, not on whether the ramp moved anything.
// Steady-state battery flight is therefore exactly MaxHVel.
func TestHorizontalClampInsideSignBranches(t *testing.T) {
	g := newGliderAtRest(100, 50)
	for i := 0; i < 20; i++ {
		g.HDesiredVel = NormalThrust // key held
		g.HVel += HyperThrust        // DoBatteryEngaged, bypassing the ramp
		g.MoveGlider()
		if g.HVel > MaxHVel {
			t.Fatalf("frame %d: HVel = %d exceeds MaxHVel %d", i+1, g.HVel, MaxHVel)
		}
	}
	if g.HVel != MaxHVel {
		t.Errorf("steady-state battery HVel = %d, want exactly MaxHVel %d", g.HVel, MaxHVel)
	}

	// The same leftward, where the branch under test is the negative one.
	g = newGliderAtRest(100, 50)
	for i := 0; i < 20; i++ {
		g.HDesiredVel = -NormalThrust
		g.HVel -= HyperThrust
		g.MoveGlider()
		if g.HVel < -MaxHVel {
			t.Fatalf("frame %d: HVel = %d exceeds -MaxHVel", i+1, g.HVel)
		}
	}
	if g.HVel != -MaxHVel {
		t.Errorf("steady-state leftward HVel = %d, want -%d", g.HVel, MaxHVel)
	}
}

// TestWasVelWrittenUnconditionally pins §7.2 consequence 3, which is the one
// place the analysis document contradicts itself: §7.1 and §7.2 say WasHVel and
// WasVVel are refreshed on every call, while §23 porting note 2 says they are
// "only updated on a non-zero-velocity frame". The C settles it -- both branches
// of both move blocks assign them (Player.c:99, :116, :131, :140) and the split is
// `if vel < 0 {} else {}`, so a zero velocity takes the positive branch and the
// assignment runs.
//
// It matters because GliderHitTop un-sweeps the hit box by WasHVel
// (Interactions.c:65): a glider that was stationary when MoveGlider last ran must
// un-sweep by 0, not by its last non-zero velocity.
func TestWasVelWrittenUnconditionally(t *testing.T) {
	g := newGliderAtRest(100, 50)

	// Build up a non-zero history, then come to a stop.
	g.hold(-10, -6)
	g.MoveGlider()
	if g.WasHVel != -10 || g.WasVVel != -6 {
		t.Fatalf("WasHVel/WasVVel = %d/%d, want -10/-6", g.WasHVel, g.WasVVel)
	}

	g.hold(0, 0) // VDesiredVel 0 as well, so VVel really is 0 through the move
	g.MoveGlider()
	if g.WasHVel != 0 {
		t.Errorf("WasHVel = %d after a zero-velocity frame, want 0: the else branch must write it",
			g.WasHVel)
	}
	if g.WasVVel != 0 {
		t.Errorf("WasVVel = %d after a zero-velocity frame, want 0", g.WasVVel)
	}
}

// TestWholeIsSweptNotAccumulated pins §7.2 consequences 4 and 5. Whole is built
// by writing the trailing edge before the move and the leading edge after, so it
// spans both positions; it is fully rewritten every call, so it does not
// accumulate; and with both velocities zero it collapses to exactly Dest.
func TestWholeIsSweptNotAccumulated(t *testing.T) {
	// Moving right and down: Whole spans from the old top-left to the new
	// bottom-right.
	g := newGliderAtRest(100, 50)
	g.hold(6, 4)
	g.MoveGlider()
	if want := (Rect{Top: 50, Left: 100, Bottom: 50 + GliderHigh + 4, Right: 100 + GliderWide + 6}); g.Whole != want {
		t.Errorf("moving right/down: Whole = %+v, want %+v", g.Whole, want)
	}

	// Moving left and up: the trailing edges are now right and bottom.
	g = newGliderAtRest(100, 50)
	g.hold(-6, -4)
	g.MoveGlider()
	if want := (Rect{Top: 50 - 4, Left: 100 - 6, Bottom: 50 + GliderHigh, Right: 100 + GliderWide}); g.Whole != want {
		t.Errorf("moving left/up: Whole = %+v, want %+v", g.Whole, want)
	}

	// Stationary: Whole collapses to Dest, proving it does not accumulate the
	// sweep from the frames above.
	g = newGliderAtRest(100, 50)
	g.hold(-20, -20)
	g.MoveGlider() // a big sweep, the horizontal half of it clamped to -16
	g.hold(0, 0)
	g.MoveGlider() // now stand still
	if g.Whole != g.Dest {
		t.Errorf("stationary: Whole = %+v, want it collapsed to Dest %+v", g.Whole, g.Dest)
	}
}

// TestShadowTracksHorizontallyOnly pins §7.2 consequence 6. MoveGlider moves
// DestShadow left and right with the glider and never touches its Top or Bottom:
// the shadow is a fixed-height blob sliding along the floor, with no scaling or
// fading with altitude.
func TestShadowTracksHorizontallyOnly(t *testing.T) {
	g := newGliderAtRest(100, 50)
	g.HVel, g.VVel = 7, 9

	for i := 0; i < 10; i++ {
		g.HDesiredVel = NormalThrust
		g.MoveGlider()
		if g.DestShadow.Top != ShadowTop {
			t.Fatalf("frame %d: DestShadow.Top = %d, want it pinned at %d",
				i+1, g.DestShadow.Top, ShadowTop)
		}
		if h := g.DestShadow.Bottom - g.DestShadow.Top; h != ShadowHigh {
			t.Fatalf("frame %d: shadow is %d tall, want %d", i+1, h, ShadowHigh)
		}
		if g.DestShadow.Left != g.Dest.Left {
			t.Fatalf("frame %d: shadow left %d, glider left %d: they must stay aligned",
				i+1, g.DestShadow.Left, g.Dest.Left)
		}
	}
	if g.Dest.Top == 50 {
		t.Error("the glider never moved vertically, so this test proved nothing")
	}
}

// TestDestNeverChangesSize is the assumption that makes the one-sided sweep equal
// a true union (§7.2 consequence 4). MoveGlider must translate Dest, never resize
// it -- the height change between normal and burning happens at mode entry, not
// during a move.
func TestDestNeverChangesSize(t *testing.T) {
	g := newGliderAtRest(100, 50)
	for i, v := range []int16{5, -5, 0, 16, -16, 3} {
		g.HVel, g.VVel = v, -v
		g.VDesiredVel = -v
		g.MoveGlider()
		if w := g.Dest.Right - g.Dest.Left; w != GliderWide {
			t.Errorf("step %d: Dest is %d wide, want %d", i, w, GliderWide)
		}
		if h := g.Dest.Bottom - g.Dest.Top; h != GliderHigh {
			t.Errorf("step %d: Dest is %d tall, want %d", i, h, GliderHigh)
		}
	}
}

// TestModeClassification pins the three mode predicates against the line counts
// the original's own guards imply: four modes are "in the room" for wall logic,
// six carry momentum, six collide with dynamic objects.
func TestModeClassification(t *testing.T) {
	var inRoom, momentum, dynamics int
	for m := GliderNormal; m <= GliderTransportingIn; m++ {
		if InRoom(m) {
			inRoom++
		}
		if HasMomentum(m) {
			momentum++
		}
		if CollidesWithDynamics(m) {
			dynamics++
		}
	}
	if inRoom != 4 {
		t.Errorf("InRoom accepts %d modes, want 4 (Interactions.c:691-694)", inRoom)
	}
	if momentum != 6 {
		t.Errorf("HasMomentum accepts %d modes, want 6", momentum)
	}
	if dynamics != 6 {
		t.Errorf("CollidesWithDynamics accepts %d modes, want 6 (Dynamics.c:45-50)", dynamics)
	}

	// Every momentum mode must also be a mode the interaction pass considers,
	// or a glider could accelerate somewhere the world cannot push back.
	for m := GliderNormal; m <= GliderTransportingIn; m++ {
		if HasMomentum(m) && !CollidesWithDynamics(m) {
			t.Errorf("mode %d has momentum but is exempt from dynamic collision", m)
		}
	}
}
