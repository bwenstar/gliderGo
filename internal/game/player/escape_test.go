package player

import "testing"

// sealedRoom is an Env whose every boundary is solid: no open sides, walls at the
// normal limits, and a background that is neither Dirt nor Roof. It is the baseline
// for the boundary tests, so each one opens exactly the one boundary it is about.
func sealedRoom() *NopEnv {
	return &NopEnv{
		Left:  LeftWallLimit,
		Right: RightWallLimit,
		Back:  2010, // any PICT id that is not Dirt or Roof
	}
}

// twoPlayerRoom is sealedRoom plus a real otherPlayerEscaped slot, which NopEnv does
// not have: its zero value would be 0, and 0 is not NoOneEscaped.
type twoPlayerRoom struct {
	NopEnv
	escaped int16
}

func newTwoPlayerRoom() *twoPlayerRoom {
	r := &twoPlayerRoom{escaped: NoOneEscaped}
	r.NopEnv = *sealedRoom()
	return r
}

func (e *twoPlayerRoom) TwoPlayerGame() bool           { return true }
func (e *twoPlayerRoom) OnePlayerLeft() bool           { return false }
func (e *twoPlayerRoom) OtherPlayerEscaped() int16     { return e.escaped }
func (e *twoPlayerRoom) SetOtherPlayerEscaped(v int16) { e.escaped = v }

// TestBurningGliderDiesAtEveryBoundary: fire is a sentence with a two-second appeal,
// and the appeal has to be won inside the room you are in. Touching any of the four
// boundaries while alight kills the glider outright, whether or not that boundary
// would otherwise have been passable.
func TestBurningGliderDiesAtEveryBoundary(t *testing.T) {
	cases := []struct {
		name       string
		dest       Rect
		openTop    bool
		openBottom bool
	}{
		// Each is placed past the *outer* limit, so an unburnt glider in the same
		// place would be allowed straight through the open boundary.
		{"ceiling", Rect{Top: -20, Left: 100, Bottom: 6, Right: 148}, true, false},
		{"floor", Rect{Top: 314, Left: 100, Bottom: 340, Right: 148}, false, true},
		{"left wall", Rect{Top: 100, Left: -30, Bottom: 126, Right: 18}, false, false},
		{"right wall", Rect{Top: 100, Left: 492, Bottom: 126, Right: 540}, false, false},
	}
	for _, c := range cases {
		e := sealedRoom()
		e.Top, e.Bottom = c.openTop, c.openBottom
		g := newGliderAtRest(c.dest.Left, c.dest.Top)
		g.Mode = GliderBurning
		g.WasMode = FramesToBurn
		g.Dest = c.dest
		g.IgnoreLeft, g.IgnoreRight, g.IgnoreGround = true, true, true // maximum permission

		g.CheckGliderInRoom(e)

		if g.Mode != GliderFadingOut {
			t.Errorf("%s: Mode = %d, want GliderFadingOut", c.name, g.Mode)
		}
		if g.WasMode != 0 {
			t.Errorf("%s: WasMode = %d, want the fuse cleared", c.name, g.WasMode)
		}
		if len(e.Transitions) != 0 {
			t.Errorf("%s: left the room via %v; a burning glider may not", c.name, e.Transitions)
		}
	}
}

// TestFloorIsLethalWithoutPermission is the game's central cruelty: there is no
// landing. A solid floor kills, and the only way through it is IgnoreGround, which the
// interaction pass grants for one frame at a time over a hole.
func TestFloorIsLethalWithoutPermission(t *testing.T) {
	e := sealedRoom()
	g := newGliderAtRest(100, 320) // Dest.Bottom = 340, past NoFloorLimit
	g.CheckGliderInRoom(e)

	if g.Mode != GliderFadingOut {
		t.Errorf("Mode = %d, want GliderFadingOut: hitting the ground is fatal", g.Mode)
	}
	// Planted exactly on the floor rather than left mid-fall.
	if want := FloorLimit - g.Dest.Bottom; g.VVel != want {
		t.Errorf("VVel = %d, want %d", g.VVel, want)
	}
}

func TestIgnoreGroundFallsThrough(t *testing.T) {
	e := sealedRoom()
	g := newGliderAtRest(100, 320) // Dest.Bottom = 340
	g.IgnoreGround = true
	g.CheckGliderInRoom(e)

	if g.Mode != GliderNormal {
		t.Errorf("Mode = %d, want GliderNormal: the hole was open", g.Mode)
	}
	if len(e.Transitions) != 1 || e.Transitions[0] != "room" {
		t.Errorf("transitions = %v, want one room move", e.Transitions)
	}
}

// TestTwoThresholdsPerBoundary is the gap between the inner and outer limits.
//
// CheckGliderInRoom triggers as soon as Dest.Bottom passes FloorLimit (312), but
// CheckEscapeDown will not hand the glider to the next room until it passes
// NoFloorLimit (332). In between, with permission, nothing at all happens -- the glider
// keeps falling and is checked again next frame. That twenty-pixel band is what makes
// dropping through a hole take three frames instead of being instantaneous, and a port
// that used one threshold for both would teleport the player.
func TestTwoThresholdsPerBoundary(t *testing.T) {
	e := sealedRoom()
	g := newGliderAtRest(100, 300) // Dest.Bottom = 320: past 312, short of 332
	g.IgnoreGround = true
	g.hold(0, 3)
	before := *g

	g.CheckGliderInRoom(e)

	if len(e.Transitions) != 0 {
		t.Errorf("transitions = %v, want none: not yet clear of the room", e.Transitions)
	}
	if g.Mode != before.Mode || g.VVel != before.VVel || g.Dest != before.Dest {
		t.Errorf("glider changed inside the threshold band: %+v -> %+v", before.Dest, g.Dest)
	}
	if FloorLimit >= NoFloorLimit {
		t.Fatal("the two floor thresholds are the wrong way round")
	}
}

// TestWallBouncesToTheWallFace: the bounce reflects HVel and adds the penetration
// measured against the *inner* limit, so the glider is returned to the wall's face
// rather than to the room's outer edge.
func TestWallBouncesToTheWallFace(t *testing.T) {
	e := sealedRoom()
	e.Foil = 0
	g := newGliderAtRest(0, 100) // Dest.Left = 0, inside LeftWallLimit 12
	g.hold(-6, 0)
	g.CheckGliderInRoom(e)

	// -(-6) + (12 - 0) = 18
	if g.HVel != 18 {
		t.Errorf("HVel = %d, want 18 (reversed 6, plus 12 of penetration)", g.HVel)
	}
	if len(e.Sounds) != 1 || e.Sounds[0] != HitWallSound {
		t.Errorf("sounds = %v, want HitWallSound", e.Sounds)
	}
	if len(e.Transitions) != 0 {
		t.Errorf("bounced through the wall: %v", e.Transitions)
	}
}

// TestOpenSideNeedsNoClearance is the asymmetry between the walls and the ceiling.
//
// A room with no wall on a side lets the glider out the instant CheckGliderInRoom's own
// test fires -- CheckEscapeLeft's open branch has no distance check at all. The ceiling
// and floor always require the outer limit. So walking sideways out of an open-sided
// room is immediate, while falling out of an open-bottomed one is not.
func TestOpenSideNeedsNoClearance(t *testing.T) {
	e := sealedRoom()
	e.Left = NoLeftWallLimit // no wall on this side
	g := newGliderAtRest(-30, 100)
	g.CheckGliderInRoom(e)

	if len(e.Transitions) != 1 {
		t.Fatalf("transitions = %v, want one room move", e.Transitions)
	}
}

// TestDirtTilesDecideTheCeiling: in a Dirt room the ceiling is per-tile, and *both*
// the tile under the glider's left edge and the one under its right must be open. A
// glider straddling the edge of a hole is stopped dead, which is why lining up under a
// gap in dirt takes care.
func TestDirtTilesDecideTheCeiling(t *testing.T) {
	// Dest.Left = 100 -> tile 1; Dest.Right = 148 -> tile 2.
	lined := sealedRoom()
	lined.Back = Dirt
	lined.Tiles[1], lined.Tiles[2] = 5, 6 // both open above
	g := newGliderAtRest(100, -20)        // Dest.Top = -20, past NoCeilingLimit
	g.CheckGliderInRoom(lined)
	if len(lined.Transitions) != 1 {
		t.Errorf("both tiles open: transitions = %v, want one", lined.Transitions)
	}

	straddling := sealedRoom()
	straddling.Back = Dirt
	straddling.Tiles[1], straddling.Tiles[2] = 5, 0 // right edge over solid dirt
	g2 := newGliderAtRest(100, -20)
	g2.hold(0, -4)
	g2.CheckGliderInRoom(straddling)
	if len(straddling.Transitions) != 0 {
		t.Errorf("straddling: transitions = %v, want none", straddling.Transitions)
	}
	// Stopped against the ceiling, not bounced: no sound, and VVel is assigned the
	// exact distance to CeilingLimit.
	if want := CeilingLimit - g2.Dest.Top; g2.VVel != want {
		t.Errorf("VVel = %d, want %d", g2.VVel, want)
	}
	if len(straddling.Sounds) != 0 {
		t.Errorf("a solid ceiling made a sound: %v", straddling.Sounds)
	}
}

// TestRoofSlopesOppositeWays pins CheckRoofCollision's mirrored tiles. Tiles 1 and 2
// measure into the tile from its left edge; 5 and 6 measure from its right. So the same
// glider at the same height crashes through one and clears the other.
func TestRoofSlopesOppositeWays(t *testing.T) {
	// Dest.Left = 60 puts the glider's centre at 84, i.e. 20 px into tile 1.
	// Tile 2:  reach = 20,      intercept 186 -> crash when Bottom > 166.
	// Tile 5:  reach = 64-20=44, intercept 186 -> crash when Bottom > 142.
	// Dest.Bottom = 160 is between the two.
	for _, c := range []struct {
		tile  int16
		crash bool
	}{{2, false}, {5, true}} {
		e := sealedRoom()
		e.Back = Roof
		e.Tiles[1] = c.tile
		g := newGliderAtRest(60, 140) // Dest.Bottom = 160, past RoofLimit 122
		g.CheckGliderInRoom(e)

		if got := g.Mode == GliderFadingOut; got != c.crash {
			t.Errorf("tile %d: crashed = %v, want %v", c.tile, got, c.crash)
		}
	}
}

// TestRoofIgnoresAnUnknownTile: any tile that is not one of the four roof profiles is
// an immediate crash, with no geometry at all. That is the original's default branch,
// and it means a roof room with a stray tile number is a death trap rather than a
// pass-through.
func TestRoofIgnoresAnUnknownTile(t *testing.T) {
	e := sealedRoom()
	e.Back = Roof
	e.Tiles[1] = 3 // not 1, 2, 5 or 6
	g := newGliderAtRest(60, 140)
	g.CheckGliderInRoom(e)
	if g.Mode != GliderFadingOut {
		t.Errorf("Mode = %d, want GliderFadingOut", g.Mode)
	}
}

// TestGreaseSlidesOverTheRoof is the only thing Sliding does outside picking a sprite:
// it suppresses the roof collision entirely, so a glider on grease slides along a roof
// it would otherwise fall through.
func TestGreaseSlidesOverTheRoof(t *testing.T) {
	e := sealedRoom()
	e.Back = Roof
	e.Tiles[1] = 3 // the fatal default branch
	g := newGliderAtRest(60, 140)
	g.Sliding = true
	g.CheckGliderInRoom(e)
	if g.Mode != GliderNormal {
		t.Errorf("Mode = %d, want GliderNormal: grease should carry the glider over", g.Mode)
	}
}

// TestSlidingOutlivesItsFrame pins the half of Sliding that reads like a one-frame flag
// and is not one. Only MoveGliderNormal clears it, and HandleGlider reaches that in mode
// Normal alone -- while CheckGliderInRoom, and so the roof check that consults the flag,
// runs in four modes. A glider that slips on grease and then tumbles about-face keeps its
// immunity to the roof for the three frames of the tumble.
func TestSlidingOutlivesItsFrame(t *testing.T) {
	for _, m := range []Mode{GliderFaceLeft, GliderFaceRight, GliderBurning} {
		e := sealedRoom()
		g := newGliderAtRest(100, 140)
		g.Mode = m
		g.Sliding = true
		g.HandleGlider(e)
		if !g.Sliding {
			t.Errorf("mode %d cleared Sliding; only Normal does", m)
		}
	}

	e := sealedRoom()
	g := newGliderAtRest(100, 140)
	g.Sliding = true
	g.HandleGlider(e)
	if g.Sliding {
		t.Error("mode Normal left Sliding set; MoveGliderNormal must consume it")
	}
}

// TestNonInRoomModesAreNotBounded is the mechanism that lets the transit modes work at
// all. A glider walking up the stairs is deliberately driving Dest past the ceiling
// limit every frame; if CheckGliderInRoom applied to it, it would be killed or bounced
// on its way out.
func TestNonInRoomModesAreNotBounded(t *testing.T) {
	for _, m := range []Mode{GliderGoingUp, GliderDuctingDown, GliderMailInLeft, GliderInLimbo} {
		e := sealedRoom()
		g := newGliderAtRest(100, 320) // deep past the floor: fatal in a normal mode
		g.Mode = m
		g.CheckGliderInRoom(e)
		if g.Mode != m {
			t.Errorf("mode %d was changed to %d by a boundary check", m, g.Mode)
		}
		if len(e.Sounds) != 0 || len(e.Transitions) != 0 {
			t.Errorf("mode %d triggered %v / %v", m, e.Sounds, e.Transitions)
		}
	}
}

// TestTwoPlayerRaceForTheCeiling walks all three outcomes of raceForExit.
//
// The third is the one that has no counterpart in a one-player game: if the other
// player has already left by a different exit, this glider is refused. Both players
// must leave a room the same way, so whoever gets out first chooses the route.
func TestTwoPlayerRaceForTheCeiling(t *testing.T) {
	t.Run("first one out waits", func(t *testing.T) {
		e := newTwoPlayerRoom()
		e.Top = true
		g := newGliderAtRest(100, -20)
		g.CheckGliderInRoom(e)

		if e.escaped != PlayerEscapedUp {
			t.Errorf("escaped = %d, want PlayerEscapedUp", e.escaped)
		}
		if g.Mode != GliderInLimbo {
			t.Errorf("Mode = %d, want GliderInLimbo", g.Mode)
		}
		if g.WasMode != GliderNormal {
			t.Errorf("WasMode = %d, want the mode it came from", g.WasMode)
		}
		if len(e.Transitions) != 0 {
			t.Errorf("left without waiting: %v", e.Transitions)
		}
		if len(e.Sounds) != 1 || e.Sounds[0] != FollowSound {
			t.Errorf("sounds = %v, want FollowSound", e.Sounds)
		}
	})

	t.Run("second one out takes both", func(t *testing.T) {
		e := newTwoPlayerRoom()
		e.Top = true
		e.escaped = PlayerEscapedUp
		g := newGliderAtRest(100, -20)
		g.CheckGliderInRoom(e)

		if e.escaped != NoOneEscaped {
			t.Errorf("escaped = %d, want the slot cleared", e.escaped)
		}
		if len(e.Transitions) != 1 {
			t.Errorf("transitions = %v, want one room move", e.Transitions)
		}
	})

	t.Run("wrong exit is refused", func(t *testing.T) {
		e := newTwoPlayerRoom()
		e.Top = true
		e.escaped = PlayerEscapedLeft // the other player went sideways
		g := newGliderAtRest(100, -20)
		g.hold(0, -5)
		g.CheckGliderInRoom(e)

		if e.escaped != PlayerEscapedLeft {
			t.Errorf("escaped = %d, want it left alone", e.escaped)
		}
		if len(e.Transitions) != 0 {
			t.Errorf("got out anyway: %v", e.Transitions)
		}
		if len(e.Sounds) != 1 || e.Sounds[0] != DontExitSound {
			t.Errorf("sounds = %v, want DontExitSound", e.Sounds)
		}
		// -(-5) + (-10 - -20) = 5 + 10 = 15: reversed, plus the overshoot repaid.
		if g.VVel != 15 {
			t.Errorf("VVel = %d, want 15", g.VVel)
		}
	})
}

// TestManholeSkipsTheRaceInTwoPlayer records an asymmetry in the original rather than a
// design choice. Falling through a hole in a dirt floor with IgnoreGround set calls
// MoveRoomToRoom directly even in a two-player game (Interactions.c:346-350), so the
// player who drops through leaves at once and the other gets no banner and no limbo --
// unlike every other exit in the game.
func TestManholeSkipsTheRaceInTwoPlayer(t *testing.T) {
	e := newTwoPlayerRoom()
	e.Back = Dirt
	e.Tiles[1], e.Tiles[2] = 0, 0 // solid dirt: the "not open below" branch
	g := newGliderAtRest(100, 320)
	g.IgnoreGround = true
	g.CheckGliderInRoom(e)

	if len(e.Transitions) != 1 {
		t.Fatalf("transitions = %v, want one immediate room move", e.Transitions)
	}
	if e.escaped != NoOneEscaped {
		t.Errorf("escaped = %d, want the race untouched", e.escaped)
	}
	if g.Mode != GliderNormal {
		t.Errorf("Mode = %d, want GliderNormal: no limbo", g.Mode)
	}
}

// TestCornerExitGetsTwoVerdicts: the vertical and horizontal tests are two separate
// if/else chains, not one, so a glider leaving through a corner is judged on both axes
// in the same frame.
func TestCornerExitGetsTwoVerdicts(t *testing.T) {
	e := sealedRoom()
	e.Top = true
	g := newGliderAtRest(0, -20) // out through the ceiling and into the left wall
	g.hold(-6, -6)
	g.CheckGliderInRoom(e)

	// Vertical: through the open ceiling. Horizontal: bounced off the wall.
	if len(e.Transitions) != 1 {
		t.Errorf("transitions = %v, want the ceiling exit", e.Transitions)
	}
	if len(e.Sounds) != 1 || e.Sounds[0] != HitWallSound {
		t.Errorf("sounds = %v, want the wall impact as well", e.Sounds)
	}
}
