package player

// Leaving a room, from GliderPRO/Sources/Interactions.c:171-752.
//
// CheckGliderInRoom runs once per frame after the move and asks whether the glider
// has gone past any of the four boundaries. Each boundary then has three possible
// answers -- walk through into the next room, bounce off a wall, or die -- and which
// one applies depends on the room's own openness, on the background, and on the
// one-frame Ignore flags the interaction pass set.
//
// The original has eight functions here where four would do, because the two-player
// versions are the one-player versions with a race for the exit woven through them
// (CheckEscapeUpTwo against CheckEscapeUp, and so on). The port keeps them as eight
// so the pairs can be read against each other, and factors out only the two blocks
// that are genuinely character-for-character identical: raceForExit and burnOut.
//
// Two thresholds are involved per axis and they are not the same number.
// CheckGliderInRoom triggers on the *inner* limit -- CeilingLimit, FloorLimit,
// LeftThresh, RightThresh -- but the escape functions require the glider to be past
// the *outer* one, NoCeilingLimit and friends, before they will let it through. The
// gap between them is the band in which the glider is overlapping the boundary but
// has not yet cleared it, and it is what makes walking out of a room take several
// frames rather than happening the instant a pixel crosses.

// Background PICT resource IDs the escape logic branches on
// (GliderDefines.h:238, :241). thisBackground holds the resource ID itself, so these
// are compared as numbers rather than being an enum.
const (
	Dirt int16 = 2011 // a dirt room: the tiles decide where the ceiling and floor are
	Roof int16 = 2014 // a rooftop: a diagonal roof line, not a flat floor
)

// The four geographic escape codes this file writes -- PlayerEscapedRight, Left, Up
// and Down -- live in consts.go with the other nine, because the set is read as a
// whole by the hot spots and the transit handlers and splitting it across two files
// hid the handshake.

// Impact sounds (GliderDefines.h:55, :80, :101, :120, :123, :141).
const (
	HitWallSound int16 = 0
	FoilHitSound int16 = 25
	// DontExitSound is the refusal when the other player has already left by a
	// different exit: in a two-player game both must leave the same way.
	DontExitSound int16 = 46

	HitWallPriority  int16 = 100
	DontExitPriority int16 = 103
	FoilHitPriority  int16 = 400
)

// RoofCrashLow and RoofCrashHigh are the two y intercepts of the diagonal roof lines
// (Interactions.c:461, :471, :481, :491). A tile's roof surface is the line
// y = intercept - dx, where dx is the distance into the tile.
const (
	RoofCrashLow  int16 = 186
	RoofCrashHigh int16 = 250
)

// dirtTileOpenAbove reports whether a Dirt tile has sky at its top, so a glider
// beneath it can rise out of the room. Tiles 5 and 6 are the two that do
// (Interactions.c:206-209, :263-266).
func dirtTileOpenAbove(t int16) bool { return t == 5 || t == 6 }

// dirtTileOpenBelow is the same for the floor: tiles 2 and 3
// (Interactions.c:318-321, :396-397).
func dirtTileOpenBelow(t int16) bool { return t == 2 || t == 3 }

// tileUnder is `dest.left >> 6` and `dest.right >> 6`, the tile column an edge is in.
// The shift is arithmetic in both C and Go, so a negative x floors rather than
// truncating toward zero -- which is why the callers range-check the result instead of
// clamping it.
func tileUnder(x int16) int16 { return x >> 6 }

// tilesInRange is the `>= 0 && < 8` guard the dirt checks apply to both edges.
// Out of range means the glider is already half out of the room sideways, and every
// caller treats that as "no hole here", i.e. solid.
func tilesInRange(a, b int16) bool {
	return a >= 0 && a < NumTiles && b >= 0 && b < NumTiles
}

// burnOut is the four-line block CheckGliderInRoom applies at every boundary when the
// glider is on fire (Interactions.c:698-703, :711-716, :727-732, :740-745).
//
// A burning glider cannot leave the room by any route: touching any boundary kills it
// outright. That is the counterweight to fire being survivable at all -- you have two
// seconds to find water, and you have to find it here.
//
// WasMode = 0 clears the burn fuse. It has no observable effect in the original's own
// flow, because StartGliderFadingOut is about to change the mode and its early return
// cannot fire from GliderBurning, so nothing reads the fuse again. Kept because it is
// in the source and costs nothing.
func (g *Glider) burnOut(e Env) {
	g.WasMode = 0
	g.StartGliderFadingOut(e)
	e.PlayPrioritySound(FadeOutSound, FadeOutPriority)
}

// raceForExit is the three-branch block at the heart of all four CheckEscape*Two
// functions (Interactions.c:179-195 and its six other copies).
//
// It is a stricter race than the one the transit handlers in handle.go run. There,
// arriving second simply lets both players through. Here there is a third outcome:
// if the other player has already left by a *different* exit, this glider is refused
// and bounced back with the "don't exit" sound. Both players must leave a room the
// same way, so the first one out effectively chooses the route for both.
//
// The bounce is passed in because it is the only part that differs per boundary --
// which velocity to reflect and against which limit.
func (g *Glider) raceForExit(e Env, code, where int16, bounce func()) {
	switch e.OtherPlayerEscaped() {
	case NoOneEscaped:
		e.SetOtherPlayerEscaped(code)
		e.RefreshScoreboard(EscapedTitleMode)
		g.FlagGliderInLimbo(e, true)
	case code:
		e.SetOtherPlayerEscaped(NoOneEscaped)
		e.MoveRoomToRoom(g, where)
	default:
		e.PlayPrioritySound(DontExitSound, DontExitPriority)
		bounce()
	}
}

// bounceOffCeiling and its three siblings are the `vel = -vel + offset` reflections.
// The offset is the penetration depth past the outer limit, so the reflection both
// reverses the velocity and asks for the overshoot back.
//
// Asks, rather than gets: every velocity write in this file is attenuated before it
// moves anything. CheckGliderInRoom runs inside the interaction pass, which precedes
// HandleGlider in the same frame (Play.c:482 then :487), so MoveGlider's ramp eats up
// to VImpulse or HImpulse of whatever is written here on the way to the move. None of
// these four functions lands the glider exactly where its arithmetic says.
func (g *Glider) bounceOffCeiling() { g.VVel = -g.VVel + (NoCeilingLimit - g.Dest.Top) }
func (g *Glider) bounceOffFloor()   { g.VVel = -g.VVel + (NoFloorLimit - g.Dest.Bottom) }

// stopAtCeiling is `vVel = kCeilingLimit - dest.top`: not a bounce but a hard stop,
// assigning the velocity that *would* put the glider against the ceiling. The ramp
// takes its cut first, so the stop is never applied in full -- a snap of 5 or more
// leaves Dest.Top at 6, one of 3 leaves it at 8, one of 1 leaves it at 10 -- and
// touching a ceiling is usually a two-frame event. It is what a solid ceiling does,
// and unlike the wall case it makes no sound at all.
func (g *Glider) stopAtCeiling() { g.VVel = CeilingLimit - g.Dest.Top }

// crashIntoGround is the floor's version, and it is fatal: the glider is aimed at the
// floor -- two pixels short of it, after the ramp -- and the fade-out starts. Falling
// to the ground kills you in this game; there is no landing.
func (g *Glider) crashIntoGround(e Env) {
	g.VVel = FloorLimit - g.Dest.Bottom
	g.StartGliderFadingOut(e)
	e.PlayPrioritySound(FadeOutSound, FadeOutPriority)
}

// hitWall plays the wall impact and reflects HVel against `limit`, which is the inner
// wall limit rather than the outer one -- so a glider that bounces off a wall is
// pushed back to the wall's face, not to the room's edge.
func (g *Glider) hitWall(e Env, limit, edge int16) {
	if e.FoilTotal() > 0 {
		e.PlayPrioritySound(FoilHitSound, FoilHitPriority)
	} else {
		e.PlayPrioritySound(HitWallSound, HitWallPriority)
	}
	g.HVel = -g.HVel + (limit - edge)
}

//---------------------------------------------------------------------- up

// CheckEscapeUp is Interactions.c:244-279: the one-player ceiling.
//
// Three cases. An open top lets the glider through once it has cleared
// NoCeilingLimit. A Dirt room asks its tiles instead: both the tile under the
// glider's left edge and the one under its right must be open above, so a glider
// straddling the edge of a hole is stopped. Anything else is a solid ceiling.
func (g *Glider) CheckEscapeUp(e Env) {
	if e.TopOpen() {
		if g.Dest.Top < NoCeilingLimit {
			e.MoveRoomToRoom(g, Above)
		}
		return
	}
	if e.Background() != Dirt {
		g.stopAtCeiling()
		return
	}
	l, r := tileUnder(g.Dest.Left), tileUnder(g.Dest.Right)
	if !tilesInRange(l, r) {
		g.stopAtCeiling()
		return
	}
	if dirtTileOpenAbove(e.Tile(l)) && dirtTileOpenAbove(e.Tile(r)) {
		if g.Dest.Top < NoCeilingLimit {
			e.MoveRoomToRoom(g, Above)
		}
		return
	}
	g.stopAtCeiling()
}

// CheckEscapeUpTwo is Interactions.c:171-240, the same with the race woven in.
//
// Note what is *not* different: the solid-ceiling outcomes are identical, because a
// ceiling does not care how many players there are. Only the two paths that actually
// leave the room gain the race.
func (g *Glider) CheckEscapeUpTwo(e Env) {
	if e.TopOpen() {
		if g.Dest.Top < NoCeilingLimit {
			g.raceForExit(e, PlayerEscapedUp, Above, g.bounceOffCeiling)
		}
		return
	}
	if e.Background() != Dirt {
		g.stopAtCeiling()
		return
	}
	l, r := tileUnder(g.Dest.Left), tileUnder(g.Dest.Right)
	if !tilesInRange(l, r) {
		g.stopAtCeiling()
		return
	}
	if dirtTileOpenAbove(e.Tile(l)) && dirtTileOpenAbove(e.Tile(r)) {
		if g.Dest.Top < NoCeilingLimit {
			g.raceForExit(e, PlayerEscapedUp, Above, g.bounceOffCeiling)
		}
		return
	}
	g.stopAtCeiling()
}

//---------------------------------------------------------------------- down

// CheckEscapeDown is Interactions.c:378-446: the one-player floor.
//
// The extra ingredient here is IgnoreGround, which the interaction pass sets when the
// glider is over an open manhole or a hole in the floor. Without it the solid-floor
// branch kills the player; with it, the same branch lets them fall through into the
// room below. So the floor of a room is lethal by default and passable only by
// explicit permission, renewed every frame.
//
// The three solid-floor branches -- wrong background, tiles out of range, tiles not
// open below -- are character-for-character identical in the original.
func (g *Glider) CheckEscapeDown(e Env) {
	if e.BottomOpen() {
		if g.Dest.Bottom > NoFloorLimit {
			e.MoveRoomToRoom(g, Below)
		}
		return
	}
	if e.Background() == Dirt {
		l, r := tileUnder(g.Dest.Left), tileUnder(g.Dest.Right)
		if tilesInRange(l, r) && dirtTileOpenBelow(e.Tile(l)) && dirtTileOpenBelow(e.Tile(r)) {
			if g.Dest.Bottom > NoFloorLimit {
				e.MoveRoomToRoom(g, Below)
			}
			return
		}
	}
	g.hitGroundOrFallThrough(e)
}

// hitGroundOrFallThrough is the solid-floor outcome, repeated four times in
// CheckEscapeDown and three in CheckEscapeDownTwo (Interactions.c:346-357 etc.).
func (g *Glider) hitGroundOrFallThrough(e Env) {
	if g.IgnoreGround {
		if g.Dest.Bottom > NoFloorLimit {
			e.MoveRoomToRoom(g, Below)
		}
		return
	}
	g.crashIntoGround(e)
}

// CheckEscapeDownTwo is Interactions.c:283-374.
//
// One asymmetry against CheckEscapeUpTwo is worth noting: falling through a hole in
// the floor with IgnoreGround set does *not* run the race. Interactions.c:346-350
// calls MoveRoomToRoom directly even in a two-player game, so a player who drops
// through a manhole leaves immediately and the other is left behind without a banner
// or a limbo. Whether that is deliberate or an oversight, it is what the original
// does.
//
// It is also structurally different: the Dirt branch has no `else` for tiles out of
// range, so a two-player glider straddling the room edge in a dirt room falls out of
// the function having done nothing at all -- no crash, no stop, no transition -- and
// keeps falling until the next frame's check. CheckEscapeDown does have that else.
func (g *Glider) CheckEscapeDownTwo(e Env) {
	if e.BottomOpen() {
		if g.Dest.Bottom > NoFloorLimit {
			g.raceForExit(e, PlayerEscapedDown, Below, g.bounceOffFloor)
		}
		return
	}
	if e.Background() == Dirt {
		l, r := tileUnder(g.Dest.Left), tileUnder(g.Dest.Right)
		if !tilesInRange(l, r) {
			// Interactions.c:315-358 -- deliberately no else branch. See above.
			return
		}
		if dirtTileOpenBelow(e.Tile(l)) && dirtTileOpenBelow(e.Tile(r)) {
			if g.Dest.Bottom > NoFloorLimit {
				g.raceForExit(e, PlayerEscapedDown, Below, g.bounceOffFloor)
			}
			return
		}
	}
	g.hitGroundOrFallThrough(e)
}

//---------------------------------------------------------------------- roof

// CheckRoofCollision is Interactions.c:450-505: the sloped rooftop.
//
// A Roof room has no flat floor. Each of the eight tiles carries a diagonal roof
// surface, and the test is whether the glider's feet have gone below the line
// y = intercept - dx, where dx is how far into the tile the glider's centre is.
// Tiles 1 and 6 sit lower (intercept 250) and tiles 2 and 5 higher (186); tiles 5
// and 6 measure dx from the tile's right edge, so their lines slope the other way.
//
// Only the glider's horizontal centre is tested, not both edges -- a glider is
// treated as a point for roof purposes, which is why you can hang half off a gable.
//
// Any tile that is not one of the four is an immediate crash. And the whole function
// is skipped while Sliding: a glider on grease slides along the roof instead of
// falling through it, which is the only place Sliding does anything outside choosing
// a sprite. Note that the skip is not limited to the frame the grease was touched --
// only mode Normal clears Sliding, and this function also runs in FaceLeft, FaceRight
// and Burning, so a glider that slips and then tumbles about-face stays immune for
// those frames as well. See the Sliding field comment.
func (g *Glider) CheckRoofCollision(e Env) {
	col := tileUnder(g.Dest.Left + HalfGliderWide)
	if col < 0 || col > NumTiles-1 || g.Sliding {
		return
	}

	dx := (g.Dest.Left + HalfGliderWide) - (col << 6)
	var reach, intercept int16
	switch e.Tile(col) {
	case 1:
		reach, intercept = dx, RoofCrashHigh
	case 2:
		reach, intercept = dx, RoofCrashLow
	case 5:
		reach, intercept = TileWide-dx, RoofCrashLow
	case 6:
		reach, intercept = TileWide-dx, RoofCrashHigh
	default:
		g.crashIntoGround(e)
		return
	}
	if reach > intercept-g.Dest.Bottom {
		g.crashIntoGround(e)
	}
}

//---------------------------------------------------------------------- sides

// CheckEscapeLeft is Interactions.c:572-595: the left wall.
//
// LeftThresh doubles as the answer to "is there a wall here": it equals
// LeftWallLimit in a walled room and something else in an open one. If the room is
// open the glider simply leaves, with no distance requirement at all -- the caller's
// `Dest.Left < LeftThresh` was enough.
//
// In a walled room IgnoreLeft is the doorway permission, and it works like
// IgnoreGround: without it the glider bounces off the wall, with it the glider passes
// through once it has cleared NoLeftWallLimit.
func (g *Glider) CheckEscapeLeft(e Env) {
	if e.LeftThresh() != LeftWallLimit {
		e.MoveRoomToRoom(g, ToLeft)
		return
	}
	if g.IgnoreLeft {
		if g.Dest.Left < NoLeftWallLimit {
			e.MoveRoomToRoom(g, ToLeft)
		}
		return
	}
	g.hitWall(e, LeftWallLimit, g.Dest.Left)
}

// CheckEscapeLeftTwo is Interactions.c:509-568.
func (g *Glider) CheckEscapeLeftTwo(e Env) {
	bounce := func() { g.HVel = -g.HVel + (NoLeftWallLimit - g.Dest.Left) }
	if e.LeftThresh() != LeftWallLimit {
		g.raceForExit(e, PlayerEscapedLeft, ToLeft, bounce)
		return
	}
	if g.IgnoreLeft {
		if g.Dest.Left < NoLeftWallLimit {
			g.raceForExit(e, PlayerEscapedLeft, ToLeft, bounce)
		}
		return
	}
	g.hitWall(e, LeftWallLimit, g.Dest.Left)
}

// CheckEscapeRight is Interactions.c:662-685, the mirror of CheckEscapeLeft.
func (g *Glider) CheckEscapeRight(e Env) {
	if e.RightThresh() != RightWallLimit {
		e.MoveRoomToRoom(g, ToRight)
		return
	}
	if g.IgnoreRight {
		if g.Dest.Right > NoRightWallLimit {
			e.MoveRoomToRoom(g, ToRight)
		}
		return
	}
	g.hitWall(e, RightWallLimit, g.Dest.Right)
}

// CheckEscapeRightTwo is Interactions.c:599-658.
func (g *Glider) CheckEscapeRightTwo(e Env) {
	bounce := func() { g.HVel = -g.HVel + (NoRightWallLimit - g.Dest.Right) }
	if e.RightThresh() != RightWallLimit {
		g.raceForExit(e, PlayerEscapedRight, ToRight, bounce)
		return
	}
	if g.IgnoreRight {
		if g.Dest.Right > NoRightWallLimit {
			g.raceForExit(e, PlayerEscapedRight, ToRight, bounce)
		}
		return
	}
	g.hitWall(e, RightWallLimit, g.Dest.Right)
}

//---------------------------------------------------------------------- dispatch

// CheckGliderInRoom is Interactions.c:689-752: has the glider left the room?
//
// Only the four InRoom modes are tested, which is the mechanism that stops a glider
// on the stairs or in a duct from also falling out of the world: those modes are
// moving Dest past every one of these limits on purpose.
//
// The vertical tests are an if/else chain, so a glider that is somehow past both the
// ceiling and the floor is treated as having hit the ceiling; the roof test is the
// chain's last arm and so is skipped entirely on any frame the glider is also out
// through the top or bottom. The horizontal tests are a *separate* chain, so a glider
// leaving through a corner gets both a vertical and a horizontal verdict in the same
// frame -- and, in a two-player game, can set otherPlayerEscaped for one direction
// and then immediately be refused for the other.
func (g *Glider) CheckGliderInRoom(e Env) {
	if !InRoom(g.Mode) {
		return
	}
	twoPlayer := e.TwoPlayerGame() && !e.OnePlayerLeft()

	switch {
	case g.Dest.Top < CeilingLimit:
		switch {
		case g.Mode == GliderBurning:
			g.burnOut(e)
		case twoPlayer:
			g.CheckEscapeUpTwo(e)
		default:
			g.CheckEscapeUp(e)
		}
	case g.Dest.Bottom > FloorLimit:
		switch {
		case g.Mode == GliderBurning:
			g.burnOut(e)
		case twoPlayer:
			g.CheckEscapeDownTwo(e)
		default:
			g.CheckEscapeDown(e)
		}
	case e.Background() == Roof && g.Dest.Bottom > RoofLimit:
		g.CheckRoofCollision(e)
	}

	switch {
	case g.Dest.Left < e.LeftThresh():
		switch {
		case g.Mode == GliderBurning:
			g.burnOut(e)
		case twoPlayer:
			g.CheckEscapeLeftTwo(e)
		default:
			g.CheckEscapeLeft(e)
		}
	case g.Dest.Right > e.RightThresh():
		switch {
		case g.Mode == GliderBurning:
			g.burnOut(e)
		case twoPlayer:
			g.CheckEscapeRightTwo(e)
		default:
			g.CheckEscapeRight(e)
		}
	}
}
