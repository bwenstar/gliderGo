package player

// The per-frame mode handlers and the dispatcher, from GliderPRO/Sources/Player.c.
//
// HandleGlider runs exactly one of these per frame per player. Six of them end in
// MoveGlider and are the only modes with momentum; the rest move Dest by a
// hard-coded speed, which is why a glider on the stairs cannot be blown sideways by
// a fan.
//
// The transit handlers all share one shape:
//
//  1. pick a sprite from Facing;
//  2. move Dest (and usually DestShadow) by a fixed speed, writing the trailing
//     edge of Whole before the move and the leading edge after, exactly as
//     MoveGlider does;
//  3. compute how much of the glider is still on the visible side of a clipping
//     plane -- `notClipped` -- and then either trim Dest, Src and Mask in lockstep
//     so the glider appears to slide behind the scenery, or, once nothing is left,
//     hand over to the next room.
//
// Step 3 is why Src and Mask are separate fields on the glider rather than one
// index: the sprite is genuinely cropped, per frame, in the same rect arithmetic.

// escapeOrWait is the twenty-two line block that ends every transit
// (Player.c:345-370, :473-498, :609-636, :716-741, :813-838, :1012-1037,
// :1103-1128). Six near-identical copies in the original, differing only in the
// escape code and which transition function to call.
//
// In a one-player game it just makes the transition. In two players it is a race:
// the first glider to reach the exit sets the escape code, gets a scoreboard banner
// and is parked in limbo; when the second one arrives it finds its own code already
// set, clears it, and both travel together. That is how the original keeps two
// players in one room without splitting the view.
func (g *Glider) escapeOrWait(e Env, code int16, move func(*Glider)) {
	if !e.TwoPlayerGame() {
		move(g)
		return
	}
	if e.OnePlayerLeft() {
		// Move whichever glider is still alive, not necessarily this one.
		move(e.Survivor())
		return
	}
	if e.OtherPlayerEscaped() == code {
		e.SetOtherPlayerEscaped(NoOneEscaped)
		move(g)
		return
	}
	e.SetOtherPlayerEscaped(code)
	e.RefreshScoreboard(EscapedTitleMode)
	g.FlagGliderInLimbo(e, true)
}

// vanish blanks the glider: it hands both swept rects to the renderer so the
// background is restored, then stops drawing. Precedes every transition that is not
// a walk between rooms (Player.c:601-607 and the four duct/mail copies).
func (g *Glider) vanish(e Env) {
	h, v := e.PlayOriginH(), e.PlayOriginV()
	e.CopyRectWorkToMain(g.Whole.Offset(h, v))
	e.CopyRectWorkToMain(g.WholeShadow.Offset(h, v))
	g.DontDraw = true
}

// moveH translates Dest and DestShadow horizontally by d and rebuilds the
// horizontal half of both swept rects. This is the block MoveGlider inlines twice
// and the transit handlers repeat verbatim eight more times.
func (g *Glider) moveH(d int16) {
	if d < 0 {
		g.Whole.Right = g.Dest.Right
		g.WholeShadow.Right = g.DestShadow.Right
	} else {
		g.Whole.Left = g.Dest.Left
		g.WholeShadow.Left = g.DestShadow.Left
	}
	g.Dest.Left += d
	g.Dest.Right += d
	g.DestShadow.Left += d
	g.DestShadow.Right += d
	if d < 0 {
		g.Whole.Left = g.Dest.Left
		g.WholeShadow.Left = g.DestShadow.Left
	} else {
		g.Whole.Right = g.Dest.Right
		g.WholeShadow.Right = g.DestShadow.Right
	}
}

// moveV translates Dest vertically by d and rebuilds the vertical half of Whole.
// DestShadow is not moved: as in MoveGlider, the shadow never leaves the floor.
func (g *Glider) moveV(d int16) {
	if d < 0 {
		g.Whole.Bottom = g.Dest.Bottom
	} else {
		g.Whole.Top = g.Dest.Top
	}
	g.Dest.Top += d
	g.Dest.Bottom += d
	if d < 0 {
		g.Whole.Top = g.Dest.Top
	} else {
		g.Whole.Bottom = g.Dest.Bottom
	}
}

// clipRight trims the glider's right edge, and its sprite's, to n pixels wide, so
// what remains is the leftmost n pixels. The shadow follows.
func (g *Glider) clipRight(n int16) {
	g.Dest.Right = g.Dest.Left + n
	g.Src.Right = g.Src.Left + n
	g.Mask.Right = g.Mask.Left + n
	g.DestShadow.Right = g.Dest.Right
}

// clipLeft keeps the rightmost n pixels.
func (g *Glider) clipLeft(n int16) {
	g.Dest.Left = g.Dest.Right - n
	g.Src.Left = g.Src.Right - n
	g.Mask.Left = g.Mask.Right - n
	g.DestShadow.Left = g.Dest.Left
}

// clipTop keeps the bottom n pixels. No shadow adjustment: the shadow has no
// vertical extent to trim.
func (g *Glider) clipTop(n int16) {
	g.Dest.Top = g.Dest.Bottom - n
	g.Src.Top = g.Src.Bottom - n
	g.Mask.Top = g.Mask.Bottom - n
}

// clipBottom keeps the top n pixels.
func (g *Glider) clipBottom(n int16) {
	g.Dest.Bottom = g.Dest.Top + n
	g.Src.Bottom = g.Src.Top + n
	g.Mask.Bottom = g.Mask.Top + n
}

// facingSprite picks between a right-facing and a left-facing sprite index, the
// two-branch if that opens twelve of these handlers.
func (g *Glider) facingSprite(right, left int16) {
	if g.Facing == FaceLeft {
		g.setSprite(left)
	} else {
		g.setSprite(right)
	}
}

//---------------------------------------------------------------- normal & burning

// MoveGliderNormal is Player.c:151-199: the interactive mode.
//
// All it does is choose one of six sprites and integrate. Every force acting on the
// glider was already applied to HDesiredVel and VDesiredVel by GetInput and the
// interaction pass before this runs.
//
// Sliding is consumed here rather than in HandleGlider: it is cleared inside the
// branch that uses it, so it survives exactly one frame and only if that frame was
// spent in normal mode. The three Ignore flags are cleared for every mode instead.
func (g *Glider) MoveGliderNormal() {
	if g.Facing == FaceLeft {
		if g.Sliding {
			g.setSprite(SpriteSlideLeft)
			g.Sliding = false
		} else if g.Tipped {
			g.setSprite(SpriteLeftTipped)
		} else {
			g.setSprite(SpriteLeft)
		}
	} else {
		if g.Sliding {
			g.setSprite(SpriteSlideRight)
			g.Sliding = false
		} else if g.Tipped {
			g.setSprite(SpriteRightTipped)
		} else {
			g.setSprite(SpriteRight)
		}
	}
	g.MoveGlider()
}

// MoveGliderBurning is Player.c:203-227.
//
// Two counters run at once in two overloaded fields: Frame cycles 0..3 for the flame
// animation, and WasMode counts the 60-frame fuse down to death. The glider is still
// fully mobile -- MoveGlider runs -- but GetInput forces the thrust in whichever
// direction it faces, so a burning glider cannot be steered, only watched.
func (g *Glider) MoveGliderBurning(e Env) {
	g.Frame++
	if g.Frame > BurningFrames-1 {
		g.Frame = 0
	}
	if g.Facing == FaceLeft {
		g.setSprite(SpriteBurningLeft + g.Frame)
	} else {
		g.setSprite(SpriteFirstBurning + g.Frame)
	}

	g.WasMode--
	if g.WasMode <= 0 {
		g.StartGliderFadingOut(e)
		e.PlayPrioritySound(FadeOutSound, FadeOutPriority)
	}
	g.MoveGlider()
}

//---------------------------------------------------------------- fades

// FadeGliderIn is Player.c:231-257: materialising after a respawn.
//
// The sound is played on the frame Frame is still 0, i.e. the first frame of the
// fade, and the counter is incremented before the sprite is chosen -- so frame 0's
// sprite, set by StartGliderFadingIn, is shown once and then never again by this
// function.
//
// EnteredRect is stamped when the fade completes, not when it starts. That makes the
// respawn point the place the glider finished materialising, which matters because
// the glider does not move during a fade but the room around it may have.
func (g *Glider) FadeGliderIn(e Env) {
	if g.Frame == 0 {
		e.PlayPrioritySound(FadeInSound, FadeInPriority)
	}
	g.Frame++
	if g.Frame >= LastFadeSequence {
		g.FlagGliderNormal(e)
		g.EnteredRect = g.Dest
	} else {
		g.setFadeSprite(g.Frame)
	}
}

// TransportGliderIn is Player.c:261-287, identical to FadeGliderIn but for the
// arrival sound.
func (g *Glider) TransportGliderIn(e Env) {
	if g.Frame == 0 {
		e.PlayPrioritySound(TransInSound, TransInPriority)
	}
	g.Frame++
	if g.Frame >= LastFadeSequence {
		g.FlagGliderNormal(e)
		g.EnteredRect = g.Dest
	} else {
		g.setFadeSprite(g.Frame)
	}
}

// FadeGliderOut is Player.c:291-313: dying. The same table walked backwards, and
// the life is spent when the counter goes negative.
func (g *Glider) FadeGliderOut(e Env) {
	g.Frame--
	if g.Frame < 0 {
		e.OffAMortal(g)
	} else {
		g.setFadeSprite(g.Frame)
	}
}

// TransportGliderOut is Player.c:594-653: dissolving into a transporter. The fade
// runs down and then the glider is gone from this room entirely.
func (g *Glider) TransportGliderOut(e Env) {
	g.Frame--
	if g.Frame < 0 {
		g.vanish(e)
		g.escapeOrWait(e, PlayerTransportedOut, e.TransportRoomToRoom)
	} else {
		g.setFadeSprite(g.Frame)
	}
}

//---------------------------------------------------------------- stairs

// MoveGliderUpStairs is Player.c:315-379: walking up and out of the top of the room.
//
// The sprite pair is the odd one out. Facing left uses SpriteLeft as expected, but
// facing right uses SpriteRightTipped -- the banked frame -- rather than SpriteRight,
// so a glider climbing rightward is drawn leaning. Faithfully reproduced; it is
// almost certainly deliberate, since a level glider would look like it was floating.
//
// The clip plane is Dest.Bottom - 29 rather than a named constant, and it shrinks the
// glider from the top as it disappears behind the top of the staircase.
func (g *Glider) MoveGliderUpStairs(e Env) {
	g.facingSprite(SpriteRightTipped, SpriteLeft)
	g.moveV(ClimbStairsSpeed)

	vNotClipped := g.Dest.Bottom - StairsClipOffset
	if vNotClipped >= GliderHigh {
		return
	}
	if vNotClipped > 0 {
		g.clipTop(vNotClipped)
		return
	}
	// Nothing left of the glider: collapse it to zero height and go up a room.
	g.Dest.Top = g.Dest.Bottom
	g.Src.Top = g.Src.Bottom
	g.Mask.Top = g.Mask.Bottom
	e.SetTakingTheStairs(true)
	g.escapeOrWait(e, PlayerEscapedUpStairs, func(who *Glider) {
		e.MoveRoomToRoom(who, Above)
	})
}

// FinishGliderUpStairs is Player.c:383-427: emerging from a staircase in the room
// above, walking up and to the left.
//
// The handoff at the end sets both HVel and HDesiredVel, and both VVel and
// VDesiredVel, to the transit speeds. Setting the desired velocities too is what
// stops the ramp from immediately eating the momentum, so the glider leaves the
// stairs already gliding rather than starting from rest.
//
// Frame is read as the kWasBurning sentinel here, having been set three modes ago by
// StartGliderGoingUpStairs -- the mechanism by which a fire would survive a
// staircase. It never does: the sentinel is never actually set, because kMoveItUp
// fades a burning glider out before the stairs begin (Interactions.c:1253-1258), so
// this branch is dead and FlagGliderNormal always wins. Kept for fidelity.
func (g *Glider) FinishGliderUpStairs(e Env) {
	g.setSprite(SpriteLeft)
	g.moveV(VClimbStairsSpeed)
	g.moveH(HClimbStairsSpeed)

	hNotClipped := g.RightClip - g.Dest.Left
	if hNotClipped < GliderWide {
		g.clipRight(hNotClipped)
		return
	}
	if g.Frame == WasBurning {
		g.FlagGliderBurning(e)
	} else {
		g.FlagGliderNormal(e)
	}
	g.HVel = HClimbStairsSpeed
	g.HDesiredVel = HClimbStairsSpeed
	g.VVel = VClimbStairsSpeed
	g.VDesiredVel = VClimbStairsSpeed
	g.EnteredRect = g.Dest
}

// MoveGliderDownStairs is Player.c:431-508: walking down and to the right, behind
// the staircase, and out of the bottom of the room.
//
// Unlike the up-stairs case this one moves horizontally as well as vertically, and
// clips from the right against RightClip, which StartGliderGoingDownStairs measured
// before the walk began.
func (g *Glider) MoveGliderDownStairs(e Env) {
	g.facingSprite(SpriteRight, SpriteLeft)
	g.moveH(HDropStairsSpeed)
	g.moveV(VDropStairsSpeed)

	hNotClipped := g.RightClip - g.Dest.Left
	if hNotClipped >= GliderWide {
		return
	}
	if hNotClipped > 0 {
		g.clipRight(hNotClipped)
		return
	}
	g.clipRight(0)
	e.SetTakingTheStairs(true)
	g.escapeOrWait(e, PlayerEscapedDownStairs, func(who *Glider) {
		e.MoveRoomToRoom(who, Below)
	})
}

// FinishGliderDownStairs is Player.c:512-556: arriving in the room below and walking
// out from behind the staircase, rightward. Clips from the left against LeftClip.
func (g *Glider) FinishGliderDownStairs(e Env) {
	g.setSprite(SpriteRight)
	g.moveH(HDropStairsSpeed)
	g.moveV(VDropStairsSpeed)

	hNotClipped := g.Dest.Right - g.LeftClip
	if hNotClipped < GliderWide {
		g.clipLeft(hNotClipped)
		return
	}
	if g.Frame == WasBurning {
		g.FlagGliderBurning(e)
	} else {
		g.FlagGliderNormal(e)
	}
	g.HVel = HDropStairsSpeed
	g.HDesiredVel = HDropStairsSpeed
	g.VVel = VDropStairsSpeed
	g.VDesiredVel = VDropStairsSpeed
	g.EnteredRect = g.Dest
}

//---------------------------------------------------------------- about-face

// MoveGliderFaceLeft is Player.c:560-573: the three-frame tumble to face left.
//
// The sprite is set from Frame *before* the decrement and the mode change is tested
// after, so the sequence shown is 20, 19, 18 and the mode ends on the frame after 18
// was drawn. MoveGlider still runs, so the glider keeps its momentum through the
// tumble -- an about-face mid-air does not stop you.
func (g *Glider) MoveGliderFaceLeft() {
	g.setSprite(g.Frame)
	g.MoveGlider()
	g.Frame--
	if g.Frame < FirstAboutFaceFrame {
		g.Mode = GliderNormal
		g.Facing = FaceLeft
	}
}

// MoveGliderFaceRight is Player.c:577-590, counting up instead: 18, 19, 20.
func (g *Glider) MoveGliderFaceRight() {
	g.setSprite(g.Frame)
	g.MoveGlider()
	g.Frame++
	if g.Frame > LastAboutFaceFrame {
		g.Mode = GliderNormal
		g.Facing = FaceRight
	}
}

//---------------------------------------------------------------- ducts

// centreOnDuct nudges the glider one pixel per frame toward Clip.Left, the duct's
// centre line (Player.c:674-697, :771-794). One pixel, not a jump: a glider that
// enters a duct off-centre visibly shuffles into place as it slides.
func (g *Glider) centreOnDuct() {
	if g.Dest.Left < g.Clip.Left {
		g.moveH(1)
	} else if g.Dest.Left > g.Clip.Left {
		g.moveH(-1)
	}
}

// MoveGliderDownDuct is Player.c:657-750: sliding down into a floor duct.
func (g *Glider) MoveGliderDownDuct(e Env) {
	g.facingSprite(SpriteRight, SpriteLeft)
	g.centreOnDuct()
	g.moveV(VDropDuctSpeed)

	vNotClipped := DuctFloorLimit - g.Dest.Top
	if vNotClipped >= GliderHigh {
		return
	}
	if vNotClipped > 0 {
		g.clipBottom(vNotClipped)
		return
	}
	g.vanish(e)
	g.escapeOrWait(e, PlayerDuckedOut, e.MoveDuctToDuct)
}

// MoveGliderUpDuct is Player.c:754-847: rising into a ceiling duct.
func (g *Glider) MoveGliderUpDuct(e Env) {
	g.facingSprite(SpriteRight, SpriteLeft)
	g.centreOnDuct()
	g.moveV(VRiseDuctSpeed)

	vNotClipped := g.Dest.Bottom - (CeilingTransTop + 1)
	if vNotClipped >= GliderHigh {
		return
	}
	if vNotClipped > 0 {
		g.clipTop(vNotClipped)
		return
	}
	g.vanish(e)
	g.escapeOrWait(e, PlayerDuckedOut, e.MoveDuctToDuct)
}

// FinishGliderDuctingIn is Player.c:927-956: dropping out of a ceiling duct.
//
// The arrival sound fires on the frame Dest is still zero-height, which is the state
// the transition left it in. That is the same trick the two mail-out handlers use --
// a collapsed rect as an implicit "this is my first frame" flag, so the mode needs no
// counter at all.
//
// Alone among the finish handlers it calls FlagStillOvers, because a glider dropped
// out of a duct lands without moving and would otherwise never notice it is standing
// on something.
func (g *Glider) FinishGliderDuctingIn(e Env) {
	if g.Dest.Top == g.Dest.Bottom {
		e.PlayPrioritySound(TransInSound, TransInPriority)
	}
	g.moveV(VDropDuctSpeed)

	vNotClipped := g.Dest.Bottom - (CeilingTransTop + 1)
	if vNotClipped < GliderHigh {
		g.clipTop(vNotClipped)
		return
	}
	if g.Frame == WasBurning {
		g.FlagGliderBurning(e)
	} else {
		g.FlagGliderNormal(e)
	}
	g.EnteredRect = g.Dest
	e.FlagStillOvers(g)
}

//---------------------------------------------------------------- mail

// settleInSlot drops the glider toward Clip.Top at up to VMailDropSpeed per frame,
// so a glider sucked into a mail slot above or below its own height slides to the
// slot's floor instead of snapping (Player.c:978-988, :1069-1079).
//
// The clamp is a min against the remaining distance, so the last step is partial and
// the glider lands exactly on Clip.Top rather than overshooting and bouncing.
func (g *Glider) settleInSlot() {
	if g.Dest.Top >= g.Clip.Top {
		return
	}
	d := g.Clip.Top - g.Dest.Top
	if d > VMailDropSpeed {
		d = VMailDropSpeed
	}
	g.moveV(d)
}

// MoveGliderInMailLeft is Player.c:960-1047: being pulled rightward into a slot
// whose opening faces left. Clips from the right against Clip.Right.
func (g *Glider) MoveGliderInMailLeft(e Env) {
	g.facingSprite(SpriteRight, SpriteLeft)
	g.settleInSlot()
	g.moveH(HMailPullSpeed)

	hNotClipped := g.Clip.Right - g.Dest.Left
	if hNotClipped >= GliderWide {
		return
	}
	if hNotClipped > 0 {
		g.clipRight(hNotClipped)
		return
	}
	g.vanish(e)
	g.escapeOrWait(e, PlayerMailedOut, e.MoveMailToMail)
}

// MoveGliderInMailRight is Player.c:1051-1138, the mirror: pulled leftward, clipped
// from the left against Clip.Left.
func (g *Glider) MoveGliderInMailRight(e Env) {
	g.facingSprite(SpriteRight, SpriteLeft)
	g.settleInSlot()
	g.moveH(HMailPullRtSpeed)

	hNotClipped := g.Dest.Right - g.Clip.Left
	if hNotClipped >= GliderWide {
		return
	}
	if hNotClipped > 0 {
		g.clipLeft(hNotClipped)
		return
	}
	g.vanish(e)
	g.escapeOrWait(e, PlayerMailedOut, e.MoveMailToMail)
}

// FinishGliderMailingLeft is Player.c:851-885: being pushed leftward out of a slot.
// No sprite is chosen -- StartGliderMailingOut set it once and it does not animate.
func (g *Glider) FinishGliderMailingLeft(e Env) {
	if g.Dest.Left == g.Dest.Right {
		e.PlayPrioritySound(TransInSound, TransInPriority)
	}
	g.moveH(HPushMailSpeed)

	hNotClipped := g.Clip.Right - g.Dest.Left
	if hNotClipped < GliderWide {
		g.clipRight(hNotClipped)
		return
	}
	if g.Frame == WasBurning {
		g.FlagGliderBurning(e)
	} else {
		g.FlagGliderNormal(e)
	}
	g.EnteredRect = g.Dest
}

// FinishGliderMailingRight is Player.c:889-923, pushed rightward instead.
func (g *Glider) FinishGliderMailingRight(e Env) {
	if g.Dest.Left == g.Dest.Right {
		e.PlayPrioritySound(TransInSound, TransInPriority)
	}
	g.moveH(HPushMailRtSpeed)

	hNotClipped := g.Dest.Right - g.Clip.Left
	if hNotClipped < GliderWide {
		g.clipLeft(hNotClipped)
		return
	}
	if g.Frame == WasBurning {
		g.FlagGliderBurning(e)
	} else {
		g.FlagGliderNormal(e)
	}
	g.EnteredRect = g.Dest
}

//---------------------------------------------------------------- foil

// DeckGliderInFoil is Player.c:1142-1166: swap in the foil-clad artwork.
//
// The sprite index is Frame+2, which only lands on a sensible frame because the
// callers reach here with Frame in 5..8 -- giving 7..10, the tail of the dissolve.
// Called both from the foil dissolve's own handler and from settleFoil when some
// other mode change interrupts it.
func (g *Glider) DeckGliderInFoil(e Env) {
	e.SetShowFoil(true)
	i := g.Frame + 2
	if g.Facing == FaceLeft {
		i += LeftFadeOffset
	}
	g.setSprite(i)
}

// RemoveFoilFromGlider is Player.c:1202-1226, the same in reverse.
func (g *Glider) RemoveFoilFromGlider(e Env) {
	e.SetShowFoil(false)
	i := g.Frame + 2
	if g.Facing == FaceLeft {
		i += LeftFadeOffset
	}
	g.setSprite(i)
}

// MoveGliderFoilGoing is Player.c:1170-1198: the nine-frame dissolve that puts foil
// on.
//
// The halfway point is the actual swap: frames 1-4 dissolve the bare glider out,
// then from frame 5 DeckGliderInFoil has changed the artwork and the same indices
// dissolve the foil-clad one back in. Both halves show sprites 10-Frame, so the two
// dissolves are visually one continuous fade through a different sprite sheet.
//
// MoveGlider runs on every frame, so foil is picked up without stopping.
func (g *Glider) MoveGliderFoilGoing(e Env) {
	g.Frame++
	if g.Frame > FoilLastFrame {
		g.FlagGliderNormal(e)
	} else if g.Frame < FoilDeckFrame {
		g.setFoilSprite(g.Frame)
	} else {
		g.DeckGliderInFoil(e)
	}
	g.MoveGlider()
}

// MoveGliderFoilLosing is Player.c:1230-1256, the same dissolve taking foil off.
func (g *Glider) MoveGliderFoilLosing(e Env) {
	g.Frame++
	if g.Frame > FoilLastFrame {
		g.FlagGliderNormal(e)
	} else if g.Frame < FoilDeckFrame {
		g.setFoilSprite(g.Frame)
	} else {
		g.RemoveFoilFromGlider(e)
	}
	g.MoveGlider()
}

//---------------------------------------------------------------- shredding

// MoveGliderShredding is Player.c:1260-1319: being fed through a paper shredder.
//
// Frame is a y coordinate here, not a counter -- the height the glider is being
// ground down to, stamped by FlagGliderShredding -- until the glider is fully
// consumed, at which point it becomes ShredderCountdown, a negative frame count
// before the life is actually lost. The `Frame > 0` test at the top is what
// distinguishes the two phases, and it works only because a room-local y is always
// positive while the countdown is always negative.
//
// The two descent speeds are the opposite way round from what the names suggest:
// DropShredSlow applies once the glider has *reached* the shredder and is being
// eaten, and DropShredFast while it is still falling toward it. So the glider drops
// quickly, then grinds down a pixel at a time with the shred sound on every frame.
//
// The distance to the mouth is measured twice, before and after the move, because
// the move itself may be what closed the gap.
func (g *Glider) MoveGliderShredding(e Env) {
	if g.Frame <= 0 {
		// Phase two: the glider is gone, count up to zero and spend the life.
		g.Frame++
		if g.Frame >= 0 {
			e.OffAMortal(g)
		}
		return
	}

	g.facingSprite(SpriteRight, SpriteLeft)

	if g.Frame-g.Dest.Top < GliderHigh {
		g.moveV(DropShredSlow)
		e.SetShadowVisible(false)
		e.PlayPrioritySound(ShredSound, ShredPriority)
	} else {
		g.moveV(DropShredFast)
	}

	vNotClipped := g.Frame - g.Dest.Top
	if vNotClipped >= GliderHigh {
		return
	}
	if vNotClipped > 0 {
		g.clipBottom(vNotClipped)
		return
	}
	e.AddAShreddedGlider(g.Dest)
	g.Frame = ShredderCountdown
}

//---------------------------------------------------------------- idle & dispatch

// HandleIdleGlider is Player.c:1323-1331: the two-player freeze. The countdown lives
// in HVel, which is safe because this mode never integrates.
func (g *Glider) HandleIdleGlider() {
	g.HVel--
	if g.HVel <= 0 {
		g.Mode = g.WasMode
		g.DontDraw = false
	}
}

// HandleGlider is Player.c:1335-1439: one frame of the glider.
//
// GliderInLimbo is listed with an empty body in the original and is kept that way
// here rather than folded into the default: a glider in limbo is frozen but is not
// an error, and the distinction between "no handler" and "not a mode" is worth
// keeping. GliderComingUp and GliderComingDown dispatch to the Finish* handlers
// rather than to Move* ones, which is why those two names do not match their modes.
//
// The three Ignore flags are cleared afterwards for every mode, including the ones
// with no handler. They are single-frame permissions granted by the interaction pass
// -- "there is a doorway here", "there is a manhole under you" -- so anything that
// wants to keep one has to re-grant it next frame.
func (g *Glider) HandleGlider(e Env) {
	switch g.Mode {
	case GliderNormal:
		g.MoveGliderNormal()
	case GliderFadingIn:
		g.FadeGliderIn(e)
	case GliderFadingOut:
		g.FadeGliderOut(e)
	case GliderGoingUp:
		g.MoveGliderUpStairs(e)
	case GliderComingUp:
		g.FinishGliderUpStairs(e)
	case GliderGoingDown:
		g.MoveGliderDownStairs(e)
	case GliderComingDown:
		g.FinishGliderDownStairs(e)
	case GliderFaceLeft:
		g.MoveGliderFaceLeft()
	case GliderFaceRight:
		g.MoveGliderFaceRight()
	case GliderBurning:
		g.MoveGliderBurning(e)
	case GliderTransporting:
		g.TransportGliderOut(e)
	case GliderDuctingDown:
		g.MoveGliderDownDuct(e)
	case GliderDuctingUp:
		g.MoveGliderUpDuct(e)
	case GliderDuctingIn:
		g.FinishGliderDuctingIn(e)
	case GliderMailInLeft:
		g.MoveGliderInMailLeft(e)
	case GliderMailOutLeft:
		g.FinishGliderMailingLeft(e)
	case GliderMailInRight:
		g.MoveGliderInMailRight(e)
	case GliderMailOutRight:
		g.FinishGliderMailingRight(e)
	case GliderGoingFoil:
		g.MoveGliderFoilGoing(e)
	case GliderLosingFoil:
		g.MoveGliderFoilLosing(e)
	case GliderShredding:
		g.MoveGliderShredding(e)
	case GliderInLimbo:
		// Frozen, waiting for the other player. No handler in the original either.
	case GliderIdle:
		g.HandleIdleGlider()
	case GliderTransportingIn:
		g.TransportGliderIn(e)
	}

	g.IgnoreLeft = false
	g.IgnoreRight = false
	g.IgnoreGround = false
}
