package player

// The glider's hit box, from GliderPRO/Sources/Interactions.c:54-167.
//
// There is no separate collision shape: every test below starts from Dest, the same
// rect the sprite is drawn into, and adjusts it. Two adjustments recur:
//
//   - a 5 px inset on all four sides, applied when the caller asks to "scrutinize".
//     The glider is 48x20 and the artwork does not fill it, so the honest box is
//     38x10 and the loose box is the full 48x20. Which one a given object uses is
//     that object's decision, and it is how a glider can brush past a spike but not
//     a wall.
//   - a 6 px trim off the top while burning, because FlagGliderBurning grew Dest
//     upward to make room for the flame. Without it the flame itself would collide.
//
// All four tests are inclusive on every edge -- `theRect->bottom < glideBounds.top`
// rather than `<=` -- so rects that merely touch are treated as overlapping. That is
// QuickDraw's convention for lines rather than for rects, and reproducing it matters:
// a one-pixel gap is a hit in this game.

// HitBoxInset is the 5 px taken off each side for a scrutinized test
// (Interactions.c:60-63, :112-115).
const HitBoxInset int16 = 5

// BurningTopTrim undoes FlagGliderBurning's 6 px flame plume so the flame does not
// collide with anything (Interactions.c:108).
const BurningTopTrim int16 = 6

// SectGlider is Interactions.c:101-130: does the glider overlap r?
//
// `scrutinize` picks the tight box over the loose one. The burning trim is applied
// first and unconditionally, so a burning glider gets the same 20 px-tall body as a
// normal one either way.
func (g *Glider) SectGlider(r Rect, scrutinize bool) bool {
	b := g.Dest
	if g.Mode == GliderBurning {
		b.Top += BurningTopTrim
	}
	if scrutinize {
		b.Left += HitBoxInset
		b.Top += HitBoxInset
		b.Right -= HitBoxInset
		b.Bottom -= HitBoxInset
	}
	return sect(r, b)
}

// GliderInRect is Interactions.c:134-150: is the glider entirely inside r?
//
// Containment, not overlap, and it uses the raw Dest with no inset and no burning
// trim -- so a burning glider needs 6 px more headroom to satisfy it than a normal
// one. It is the test for "is the player lined up with this thing", used by the
// transporters and the mail slots, which is why they refuse a glider that is only
// half in.
func (g *Glider) GliderInRect(r Rect) bool {
	b := g.Dest
	return b.Top >= r.Top && b.Bottom <= r.Bottom && b.Left >= r.Left && b.Right <= r.Right
}

// GliderHitTop is Interactions.c:54-97, and the name is the opposite way round from
// what it does.
//
// It answers "did the glider land on top of r, rather than run into its side?", and
// it answers it by rewinding: the hit box is moved back by WasHVel, undoing this
// frame's horizontal motion, and re-tested. If it still overlaps, the overlap was
// not caused by moving sideways: the glider landed on the object, rose into it, or
// was engulfed by it. That is the true return.
//
// True is not survivable. The function's only caller in the original is the
// kDissolveIt case (Interactions.c:1223), and on true it kills the glider outright
// -- StartGliderFadingOut plus kFadeOutSound (:1225-1226) -- foil or no foil. Foil
// saves you only on the side path handled below, which is why the name reads
// backwards twice over.
//
// If rewinding makes the overlap vanish, the sideways motion is what caused it. That
// is a side impact, and this function handles it itself rather than reporting it: it
// plays the foil-hit sound, spends a sheet of foil, and reflects HVel. So the false
// return means "already dealt with".
//
// The reflection is `HVel = -HVel - offset`, where offset is measured from the
// still-rewound box plus 2. It therefore both reverses the velocity and adds the
// penetration depth, so the glider ends the next frame clear of the obstacle instead
// of bouncing along inside it.
//
// Note the foil is spent whether or not the glider has any: foilTotal is decremented
// unconditionally and StartGliderFoilLosing is called whenever the result is <= 0.
// The count is *not* guaranteed positive on entry -- the only caller's gate is
// `(foilTotal > 0) || (mode == kGliderLosingFoil)` (Interactions.c:1221), so a
// glider already at 0 and dissolving walks in here, leaves foilTotal at -1 and calls
// StartGliderFoilLosing a second time. That is harmless only because the function
// early-returns on modes 19 and 21. Do not clamp at 0: later code reads the value.
//
// And the caller decrements again (Interactions.c:1230-1235), so one frame of side
// contact with a kDissolveIt object costs two sheets rather than one. That half of
// the arithmetic belongs to Stage 1.5.
func (g *Glider) GliderHitTop(e Env, r Rect) bool {
	b := Rect{
		Left:   g.Dest.Left + HitBoxInset,
		Top:    g.Dest.Top + HitBoxInset,
		Right:  g.Dest.Right - HitBoxInset,
		Bottom: g.Dest.Bottom - HitBoxInset,
	}
	// Rewind this frame's horizontal move. Vertical motion is deliberately not
	// rewound: that asymmetry is the whole mechanism.
	b.Left -= g.WasHVel
	b.Right -= g.WasHVel

	if sect(r, b) {
		return true
	}

	e.PlayPrioritySound(FoilHitSound, FoilHitPriority)
	e.SetFoilTotal(e.FoilTotal() - 1)
	if e.FoilTotal() <= 0 {
		g.StartGliderFoilLosing(e)
	}

	b.Left += g.WasHVel
	b.Right += g.WasHVel
	var offset int16
	if g.HVel > 0 {
		offset = 2 + b.Right - r.Left
	} else {
		offset = 2 + b.Left - r.Right
	}
	g.HVel = -g.HVel - offset
	return false
}

// BounceGlider is Interactions.c:154-167: shove the glider back out of r
// horizontally, choosing whichever side it is nearer to.
//
// HVel is *assigned* the overlap depth rather than reflected, so the glider leaves
// with a speed that depends on how deep it got: a shallow clip barely moves it, a
// deep one flings it. That is why brushing a wall feels different from slamming
// into it.
//
// What it does not do is land the glider exactly clear. HandleInteraction runs
// before HandleGlider in the same frame (Play.c:482, :487), and MoveGlider ramps
// HVel toward HDesiredVel by HImpulse and clamps it to +/-MaxHVel *before* adding it
// to Dest (Player.c:66-77, :96-97, :119-120). With HDesiredVel at 0 the bounce frame
// therefore displaces max(min(|overlap|, 18) - 2, 0) -- two pixels short -- and an
// overlap wider than 18 px takes several frames, with BounceGlider and its sound
// re-firing on each. Contrast GliderHitTop, whose offset is `2 + penetration`: that
// +2 pre-pays the ramp, and there is no such compensation here. HDesiredVel need not
// be 0 either (Input.c:214/:216 give +/-5, Interactions.c:1211/:1215 give +/-12), so
// how far the bounce actually carries depends on the input and the fans in the same
// frame.
//
// The sound tells the player which it was: foil rings, bare glider thuds.
func (g *Glider) BounceGlider(e Env, r Rect) {
	b := g.Dest
	if (r.Right - b.Left) < (b.Right - r.Left) {
		g.HVel = r.Right - b.Left
	} else {
		g.HVel = r.Left - b.Right
	}
	if e.FoilTotal() > 0 {
		e.PlayPrioritySound(FoilHitSound, FoilHitPriority)
	} else {
		e.PlayPrioritySound(HitWallSound, HitWallPriority)
	}
}

// sect is the four-comparison overlap test the three functions above share
// (Interactions.c:68-77, :118-127). Inclusive on every edge, as noted above.
func sect(a, b Rect) bool {
	switch {
	case a.Bottom < b.Top:
		return false
	case a.Top > b.Bottom:
		return false
	case a.Right < b.Left:
		return false
	case a.Left > b.Right:
		return false
	}
	return true
}
