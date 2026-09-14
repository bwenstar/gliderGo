package player

import "testing"

// newGliderAt is newGliderAtRest under a name that reads better in these tests, where
// the point is the box rather than the velocity. At (100, 100) the glider's Dest is
// {100,100,120,148} and its inset hit box is {105,105,115,143}; every expected number
// below is derived from those two.
func newGliderAt(left, top int16) *Glider { return newGliderAtRest(left, top) }

// TestSectGliderEdgesAreInclusive pins the QuickDraw convention: a rect that merely
// touches the glider's box is a hit, because the four comparisons are `<` and `>`
// rather than `<=` and `>=`.
//
// A port that used Go's image.Rectangle.Overlaps, or any half-open convention, would
// be off by one pixel on all four sides of every object in the game.
func TestSectGliderEdgesAreInclusive(t *testing.T) {
	g := newGliderAt(100, 100) // Dest = {100,100,120,148}

	touching := Rect{Top: 120, Left: 100, Bottom: 130, Right: 148}
	if !g.SectGlider(touching, false) {
		t.Error("a rect whose top touches the glider's bottom should hit")
	}
	clear := Rect{Top: 121, Left: 100, Bottom: 130, Right: 148}
	if g.SectGlider(clear, false) {
		t.Error("a rect one pixel below the glider should not hit")
	}
}

// TestSectGliderScrutinize checks the 5 px inset: the loose box is the full 48x20 and
// the tight one is 38x10, so there is a five-pixel margin all round in which an object
// hits one and not the other.
func TestSectGliderScrutinize(t *testing.T) {
	g := newGliderAt(100, 100)

	// Sits inside the loose box but outside the tight one: 3 px in from the left.
	edge := Rect{Top: 100, Left: 100, Bottom: 120, Right: 103}
	if !g.SectGlider(edge, false) {
		t.Error("loose box should hit")
	}
	if g.SectGlider(edge, true) {
		t.Errorf("tight box should not hit: %d px of inset", HitBoxInset)
	}
}

// TestSectGliderTrimsTheFlame is the reason BurningTopTrim exists. FlagGliderBurning
// grows Dest upward by 6 px to make room for the flame, and if the hit box were not
// trimmed back the flame itself would collide with the scenery -- a burning glider
// would be unable to fit through gaps it fitted through a frame earlier.
func TestSectGliderTrimsTheFlame(t *testing.T) {
	g := newGliderAt(100, 100)
	g.FlagGliderBurning(&NopEnv{})
	if g.Dest.Tall() != GliderBurningHigh {
		t.Fatalf("burning Dest is %d tall, want %d", g.Dest.Tall(), GliderBurningHigh)
	}

	// A rect occupying only the flame plume, above the glider's real body.
	flame := Rect{Top: g.Dest.Top, Left: 100, Bottom: g.Dest.Top + BurningTopTrim - 1, Right: 148}
	if g.SectGlider(flame, false) {
		t.Error("the flame plume should not collide")
	}
	// The same rect does overlap the raw Dest, which is what proves the trim did it.
	if !sect(flame, g.Dest) {
		t.Fatal("test is not exercising the trim: the rect misses Dest entirely")
	}
}

// TestGliderInRectIgnoresTheBurningTrim is the counterpart. GliderInRect uses the raw
// Dest with no trim and no inset, so a burning glider needs six more pixels of
// headroom to be considered "inside" something than a normal one -- which is why
// catching fire next to a transporter can stop it working.
func TestGliderInRectIgnoresTheBurningTrim(t *testing.T) {
	box := Rect{Top: 100, Left: 90, Bottom: 130, Right: 160}

	normal := newGliderAt(100, 100)
	if !normal.GliderInRect(box) {
		t.Error("a normal glider at the box's top edge should be inside it")
	}

	burning := newGliderAt(100, 100)
	burning.FlagGliderBurning(&NopEnv{})
	if burning.GliderInRect(box) {
		t.Error("the same glider, burning, should no longer fit: the plume sticks out")
	}
}

// TestGliderHitTopLandingOnTop is the true branch: the glider came down onto the
// object, so rewinding the horizontal move leaves the boxes still overlapping.
//
// Nothing is spent and nothing bounces here; the caller decides what true means, and
// for the one caller there is (kDissolveIt, Interactions.c:1223-1227) it is fatal.
func TestGliderHitTopLandingOnTop(t *testing.T) {
	g := newGliderAt(100, 100)
	g.WasHVel = 0 // fell straight down
	e := &NopEnv{Foil: 3}

	obstacle := Rect{Top: 112, Left: 100, Bottom: 140, Right: 200}
	if !g.GliderHitTop(e, obstacle) {
		t.Fatal("falling onto an object should report a top hit")
	}
	if e.Foil != 3 {
		t.Errorf("a top hit spent foil: %d, want 3", e.Foil)
	}
	if len(e.Sounds) != 0 {
		t.Errorf("a top hit played %v, want silence", e.Sounds)
	}
}

// TestGliderHitTopSideImpact is the false branch, and the false return means "already
// handled" rather than "no collision": the function spends the foil and reflects HVel
// itself.
//
// The arithmetic is worth pinning exactly. The glider is 5 px into the obstacle's left
// face after moving +5 this frame; rewinding puts its right edge at 138, two pixels
// clear of the obstacle at 140, so the rewound test misses and this is a side hit.
// offset is then 2 + 143 - 140 = 5, and HVel becomes -5 - 5 = -10: reversed, plus the
// penetration paid back, so the next frame ends clear instead of grinding along inside.
func TestGliderHitTopSideImpact(t *testing.T) {
	g := newGliderAt(100, 100)
	g.HVel, g.WasHVel = 5, 5
	e := &NopEnv{Foil: 3}

	wall := Rect{Top: 100, Left: 140, Bottom: 120, Right: 200}
	if g.GliderHitTop(e, wall) {
		t.Fatal("running into an object's side should not report a top hit")
	}
	if g.HVel != -10 {
		t.Errorf("HVel = %d, want -10 (reversed 5, plus 5 of penetration)", g.HVel)
	}
	if e.Foil != 2 {
		t.Errorf("foil = %d, want 2: a side impact spends a sheet", e.Foil)
	}
	if len(e.Sounds) != 1 || e.Sounds[0] != FoilHitSound {
		t.Errorf("sounds = %v, want just FoilHitSound", e.Sounds)
	}
	if g.Mode != GliderNormal {
		t.Errorf("Mode = %d, want GliderNormal: foil remained", g.Mode)
	}
}

// TestGliderHitTopSpendsTheLastFoil: when the sheet spent was the last one, the
// dissolve starts. That is the only way foil is lost through use rather than through
// a timer.
func TestGliderHitTopSpendsTheLastFoil(t *testing.T) {
	g := newGliderAt(100, 100)
	g.HVel, g.WasHVel = 5, 5
	e := &NopEnv{Foil: 1}

	g.GliderHitTop(e, Rect{Top: 100, Left: 140, Bottom: 120, Right: 200})
	if e.Foil != 0 {
		t.Fatalf("foil = %d, want 0", e.Foil)
	}
	if g.Mode != GliderLosingFoil {
		t.Errorf("Mode = %d, want GliderLosingFoil", g.Mode)
	}
	// FoilHitSound from the impact, then FizzleSound from StartGliderFoilLosing.
	if len(e.Sounds) != 2 || e.Sounds[0] != FoilHitSound || e.Sounds[1] != FizzleSound {
		t.Errorf("sounds = %v, want [FoilHit Fizzle]", e.Sounds)
	}
}

// TestBounceGliderPicksTheNearerSide checks that the shove goes the short way out, and
// that HVel is assigned the penetration depth rather than reflected -- so the speed the
// glider leaves with is proportional to how deep it got.
func TestBounceGliderPicksTheNearerSide(t *testing.T) {
	// Clipped the obstacle's left face by 8 px: pushed left by 8.
	g := newGliderAt(100, 100) // right edge 148
	g.BounceGlider(&NopEnv{Foil: 1}, Rect{Top: 100, Left: 140, Bottom: 120, Right: 200})
	if g.HVel != -8 {
		t.Errorf("HVel = %d, want -8", g.HVel)
	}

	// Clipped the obstacle's right face by 10 px: pushed right by 10.
	g = newGliderAt(100, 100) // left edge 100
	g.BounceGlider(&NopEnv{Foil: 1}, Rect{Top: 100, Left: 60, Bottom: 120, Right: 110})
	if g.HVel != 10 {
		t.Errorf("HVel = %d, want 10", g.HVel)
	}
}

// TestBounceGliderSound: the sound is how the player learns whether they had foil.
func TestBounceGliderSound(t *testing.T) {
	wall := Rect{Top: 100, Left: 140, Bottom: 120, Right: 200}

	withFoil := &NopEnv{Foil: 2}
	newGliderAt(100, 100).BounceGlider(withFoil, wall)
	if len(withFoil.Sounds) != 1 || withFoil.Sounds[0] != FoilHitSound {
		t.Errorf("with foil: %v, want FoilHitSound", withFoil.Sounds)
	}

	bare := &NopEnv{Foil: 0}
	newGliderAt(100, 100).BounceGlider(bare, wall)
	if len(bare.Sounds) != 1 || bare.Sounds[0] != HitWallSound {
		t.Errorf("bare: %v, want HitWallSound", bare.Sounds)
	}
}
