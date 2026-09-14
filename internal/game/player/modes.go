package player

// The mode-entry functions, from GliderPRO/Sources/Modes.c.
//
// These are how the rest of the game changes the glider's state: nothing outside
// this file assigns Mode directly. Each one sets Mode, seeds whatever Frame means
// in that mode, and picks the first sprite -- and several also resize Dest, because
// the burning glider is 26 px tall and everything else is 20.
//
// Three habits recur and are worth naming once:
//
//   - Most entry points begin by settling a foil dissolve in progress
//     (settleFoil below). What that rescues is not the sprite -- every successor
//     handler rewrites Src on the next frame anyway (Player.c:320-329, :663-672,
//     :760-769), and three of the entry functions overwrite it themselves. It is
//     the global: DeckGliderInFoil sets showFoil (Player.c:1144) and
//     RemoveFoilFromGlider clears it (:1204), each reloading both glider atlases in
//     a two-player game. Skip the prologue and showFoil stays stale for the rest of
//     the foil supply, so the renderer picks the wrong atlas (Render.c:507, :163)
//     and the wrong value is written into save games (SavedGames.c:106).
//   - Several read Mode before writing it, so the order of the checks matters and
//     they cannot be reordered into a switch.
//   - `Whole = Dest` appears whenever a mode change moves or resizes the glider
//     without a MoveGlider call, because Whole is otherwise only maintained by
//     MoveGlider and would still describe the previous frame's sweep.

// settleFoil finishes a foil dissolve that is still in progress, so a mode change
// during one leaves showFoil and the artwork consistent. Modes.c repeats this pair
// of checks verbatim near the top of seven functions -- in the ducting and
// transporting starters it sits after the sound (Modes.c:219/:221, :252/:254,
// :294/:296), and in StartGliderFadingOut after the already-fading early return
// (:79-85).
func (g *Glider) settleFoil(e Env) {
	switch g.Mode {
	case GliderGoingFoil:
		g.DeckGliderInFoil(e)
	case GliderLosingFoil:
		g.RemoveFoilFromGlider(e)
	}
}

// StartGliderFadingIn is Modes.c:25-46. The glider materialises where it already
// is, so Dest is untouched and Whole is collapsed onto it.
func (g *Glider) StartGliderFadingIn(e Env) {
	if e.FoilTotal() <= 0 {
		e.SetShowFoil(false)
	}
	g.Mode = GliderFadingIn
	g.Whole = g.Dest
	g.Frame = 0
	g.DontDraw = false
	g.setFadeSprite(g.Frame)
}

// StartGliderTransportingIn is Modes.c:50-71 and is character-for-character
// StartGliderFadingIn with a different mode. The two are kept apart because their
// handlers play different sounds, which is the only way a player can tell a
// respawn from a transporter arrival.
func (g *Glider) StartGliderTransportingIn(e Env) {
	if e.FoilTotal() <= 0 {
		e.SetShowFoil(false)
	}
	g.Mode = GliderTransportingIn
	g.Whole = g.Dest
	g.Frame = 0
	g.DontDraw = false
	g.setFadeSprite(g.Frame)
}

// StartGliderFadingOut is Modes.c:75-116: dying.
//
// The early return on an already-fading glider is load-bearing. Death has several
// simultaneous causes -- a glider can be burning, out of time and touching an enemy
// on the same frame -- and without the guard the second caller would restart the
// fade and the player would never die.
//
// The shrink-from-burning block runs only when Dest is taller than a normal glider.
// It hands the renderer the *old*, taller rect first, because Whole is about to be
// collapsed onto the new smaller Dest and the six pixels of flame above the glider
// would otherwise never be erased.
func (g *Glider) StartGliderFadingOut(e Env) {
	if g.Mode == GliderFadingOut {
		return
	}
	g.settleFoil(e)

	if g.Dest.Tall() > GliderHigh {
		e.AddRectToWorkRects(g.Dest)
		if e.HasMirror() {
			e.AddRectToWorkRects(g.Dest.Offset(MirrorOffsetH, MirrorOffsetV))
		}
		g.Dest.Right = g.Dest.Left + GliderWide
		g.Dest.Top = g.Dest.Bottom - GliderHigh
	}
	g.Mode = GliderFadingOut
	g.Whole = g.Dest
	// The fade out is the fade in read backwards, so it starts at the last entry
	// and FadeGliderOut decrements.
	g.Frame = LastFadeSequence - 1
	g.setFadeSprite(g.Frame)
}

// StartGliderGoingUpStairs is Modes.c:120-133.
//
// Frame is seeded with WasBurning when the glider is on fire, so that the arrival
// handlers could in principle put it back on fire in the next room. That is why
// Frame is checked against a sentinel here rather than used as a counter: nothing
// animates while walking up stairs, so the field is free.
//
// In practice the branch is dead. It fires only when Mode == GliderBurning
// (Modes.c:127), and every caller of the stairs starters (Interactions.c:1268,
// :1276, :1281 and :1303, :1311, :1316) has already intercepted a burning glider
// and sent it to StartGliderFadingOut (:1253-1258, :1288-1293). Transcribed
// verbatim anyway, because the reachability argument lives in the interaction gates
// -- Stage 1.5 -- and not here; the corresponding reads at Player.c:417, :546,
// :879, :917 and :949 therefore always take the FlagGliderNormal arm.
func (g *Glider) StartGliderGoingUpStairs(e Env) {
	g.settleFoil(e)
	if g.Mode == GliderBurning {
		g.Frame = WasBurning
	} else {
		g.Frame = 0
	}
	g.Mode = GliderGoingUp
}

// StartGliderGoingDownStairs is Modes.c:137-151. As above, plus it measures the
// staircase now, while the glider is still in the room that contains it.
func (g *Glider) StartGliderGoingDownStairs(e Env) {
	g.settleFoil(e)
	if g.Mode == GliderBurning {
		g.Frame = WasBurning
	} else {
		g.Frame = 0
	}
	g.Mode = GliderGoingDown
	g.RightClip = e.GetUpStairsRightEdge()
}

// StartGliderMailingIn is Modes.c:155-177: the glider is being sucked into a mail
// slot. `bounds` is the slot's rect and `link` where it leads.
//
// Note what this does *not* do: it never assigns Mode. The caller in Interactions.c
// picks GliderMailInLeft or GliderMailInRight from which side the slot faces, and
// this function only sets up the clip. StartGliderDuctingDown and ...Up, which are
// otherwise the same shape, do set their own mode.
func (g *Glider) StartGliderMailingIn(e Env, bounds Rect, link Link) {
	e.PlayPrioritySound(TransOutSound, TransOutPriority)
	g.Transit = link
	g.Frame = 0
	g.Clip = bounds
	// The glider settles to sit on the slot's floor. Measured from Dest's current
	// height, so a burning glider aims 6 px higher than a normal one.
	g.Clip.Top = bounds.Bottom - g.Dest.Tall()
}

// StartGliderMailingOut is Modes.c:181-209: arriving at the far slot. Which
// direction it comes out is decided by the destination slot's own facing, not by
// the way the glider went in.
func (g *Glider) StartGliderMailingOut(e Env) {
	g.settleFoil(e)
	if g.Transit.What == LinkedToLeftMailbox {
		g.Facing = FaceLeft
		g.Mode = GliderMailOutLeft
		g.setSprite(SpriteLeft)
	} else {
		g.Facing = FaceRight
		g.Mode = GliderMailOutRight
		g.setSprite(SpriteRight)
	}
	g.stopDead()
	g.DontDraw = false
}

// StartGliderDuctingDown is Modes.c:213-242: entering a floor duct. The glider is
// centred on the duct horizontally over the following frames -- this only records
// the target, in Clip.Left.
func (g *Glider) StartGliderDuctingDown(e Env, bounds Rect, link Link) {
	e.PlayPrioritySound(TransOutSound, TransOutPriority)
	g.settleFoil(e)
	g.Transit = link
	g.Frame = 0
	g.Clip = bounds
	g.Clip.Left = bounds.Left + (bounds.Wide()-GliderWide)/2
	g.Mode = GliderDuctingDown
}

// StartGliderDuctingUp is Modes.c:246-275, the same for a ceiling duct.
func (g *Glider) StartGliderDuctingUp(e Env, bounds Rect, link Link) {
	e.PlayPrioritySound(TransOutSound, TransOutPriority)
	g.settleFoil(e)
	g.Transit = link
	g.Frame = 0
	g.Clip = bounds
	g.Clip.Left = bounds.Left + (bounds.Wide()-GliderWide)/2
	g.Mode = GliderDuctingUp
}

// StartGliderDuctingIn is Modes.c:279-284: dropping out of a ceiling duct at the
// far end. Deliberately does not settle foil or touch velocity -- the glider is
// arriving, and whatever set it up has already positioned Dest.
func (g *Glider) StartGliderDuctingIn() {
	g.Mode = GliderDuctingIn
	g.Whole = g.Dest
	g.DontDraw = false
}

// StartGliderTransporting is Modes.c:288-330: dissolving into a transporter.
//
// Unlike the fade out, this snaps Dest back to normal size unconditionally rather
// than only when it is oversized, and it does not hand the old rect to the
// renderer first, so an oversized glider leaves the difference on the background
// until something else dirties it -- an original artefact, reproduced.
//
// Not, however, via fire: a burning glider cannot step onto a transporter, because
// kTransportIt intercepts it first (Interactions.c:1382-1387). The reachable cause
// is a foil dissolve, which is still 26 tall in GliderGoingFoil and which
// kTransportIt does not gate on (:1388-1390). So the six stray pixels of flame are
// real, but they belong to a glider that caught fire and then grabbed foil.
func (g *Glider) StartGliderTransporting(e Env, link Link) {
	e.PlayPrioritySound(TransOutSound, TransOutPriority)
	g.settleFoil(e)
	g.Transit = link

	g.Dest.Right = g.Dest.Left + GliderWide
	g.Dest.Bottom = g.Dest.Top + GliderHigh
	g.DestShadow.Right = g.DestShadow.Left + GliderWide
	g.DestShadow.Bottom = g.DestShadow.Top + ShadowHigh
	g.Mode = GliderTransporting
	g.Whole = g.Dest
	g.Frame = LastFadeSequence - 1
	g.setFadeSprite(g.Frame)
}

// FlagGliderNormal is Modes.c:334-362: hand the glider back to the player.
//
// Every transit mode ends here or in FlagGliderBurning, so this is where the
// invariants are re-established: Dest and DestShadow are snapped back to their
// canonical sizes from their top-left corners, the four velocities are zeroed and
// the three one-frame ignore flags are cleared.
//
// VDesiredVel is set to 0, not Gravity. That is not a tidy-up: it buys the glider
// one frame of hang before gravity takes hold, because the first MoveGlider ramps
// toward 0, moves nothing, and only then resets VDesiredVel to Gravity. The trace
// out of a respawn is 0, 2, 3, 3 rather than 2, 3, 3, 3.
func (g *Glider) FlagGliderNormal(e Env) {
	g.Dest.Right = g.Dest.Left + GliderWide
	g.Dest.Bottom = g.Dest.Top + GliderHigh
	g.DestShadow.Right = g.DestShadow.Left + GliderWide
	g.DestShadow.Bottom = g.DestShadow.Top + ShadowHigh
	g.Mode = GliderNormal
	g.setNormalSprite()
	g.stopDead()
	g.IgnoreLeft = false
	g.IgnoreRight = false
	g.IgnoreGround = false
	g.DontDraw = false
	g.Frame = 0
	// Modes.c:361 is `shadowVisible = IsShadowVisible()`: a cached global recomputed
	// from the room. Becoming normal is the only place it is refreshed, which is why
	// a room whose light is switched off mid-flight keeps drawing the shadow.
	e.SetShadowVisible(e.IsShadowVisible())
}

// FlagGliderShredding is Modes.c:366-402: the glider has been caught by a paper
// shredder. `bounds` is the shredder's rect.
//
// Frame stops being a counter here and becomes a y coordinate -- the height the
// glider has to be ground down to -- which is why MoveGliderShredding compares it
// against Dest.Top rather than against a frame limit.
//
// The Whole bookkeeping is unusual: the glider is teleported sideways into the
// shredder's mouth, so the swept rect has to be extended in whichever direction it
// jumped, and the comparison is against Whole.Left rather than the old Dest.Left.
func (g *Glider) FlagGliderShredding(e Env, bounds Rect) {
	e.PlayPrioritySound(CaughtFireSound, CaughtFirePriority)
	g.Dest.Left = bounds.Left + ShredderInset
	g.Dest.Right = g.Dest.Left + GliderWide
	g.Dest.Bottom = g.Dest.Top + GliderHigh
	if g.Dest.Left > g.Whole.Left {
		g.Whole.Right = g.Dest.Right
		g.WholeShadow.Right = g.Dest.Right
	} else {
		g.Whole.Left = g.Dest.Left
		g.WholeShadow.Left = g.Dest.Left
	}
	g.DestShadow.Left = g.Dest.Left
	g.DestShadow.Right = g.DestShadow.Left + GliderWide
	g.DestShadow.Bottom = g.DestShadow.Top + ShadowHigh
	g.Mode = GliderShredding
	g.setNormalSprite()
	g.stopDead()
	g.Frame = bounds.Bottom - ShredderMouthInset
}

// FlagGliderBurning is Modes.c:406-434: the glider catches fire.
//
// Dest grows upward by 6 px for the flame, keeping Bottom fixed, so a burning
// glider stands on the same ground as a normal one. WasMode becomes the 60-frame
// fuse -- two seconds -- and Frame becomes the 4-frame flame animation phase.
func (g *Glider) FlagGliderBurning(e Env) {
	e.PlayPrioritySound(CaughtFireSound, CaughtFirePriority)
	g.Dest.Right = g.Dest.Left + GliderWide
	g.Dest.Top = g.Dest.Bottom - GliderBurningHigh
	g.DestShadow.Right = g.DestShadow.Left + GliderWide
	g.DestShadow.Bottom = g.DestShadow.Top + ShadowHigh
	g.Mode = GliderBurning
	if g.Facing == FaceLeft {
		g.setSprite(SpriteBurningLeft)
	} else {
		g.setSprite(SpriteFirstBurning)
	}
	g.stopDead()
	g.Frame = 0
	g.WasMode = FramesToBurn
}

// FlagGliderFaceLeft is Modes.c:438-444. Frame seeds at the *last* about-face
// sprite and MoveGliderFaceLeft counts down to the first, so the tumble runs
// 20, 19, 18. Facing itself is not changed until the tumble finishes.
func (g *Glider) FlagGliderFaceLeft() {
	g.Mode = GliderFaceLeft
	g.Frame = LastAboutFaceFrame
	g.setSprite(LastAboutFaceFrame)
}

// FlagGliderFaceRight is Modes.c:448-454, counting up 18, 19, 20 instead.
func (g *Glider) FlagGliderFaceRight() {
	g.Mode = GliderFaceRight
	g.Frame = FirstAboutFaceFrame
	g.setSprite(FirstAboutFaceFrame)
}

// FlagGliderInLimbo is Modes.c:458-468: two-player only. The first player out of a
// room waits here while the other catches up. WasMode remembers what it was doing
// so UndoGliderLimbo can put it back.
//
// The "follow me" prompt is capped at three per game by saidFollow, which is a
// global in the original and a counter on the Env here.
func (g *Glider) FlagGliderInLimbo(e Env, sayIt bool) {
	g.WasMode = g.Mode
	g.Mode = GliderInLimbo
	if sayIt && e.SaidFollow() < MaxSaidFollow {
		e.PlayPrioritySound(FollowSound, FollowPriority)
		e.SetSaidFollow(e.SaidFollow() + 1)
	}
	e.SetFirstPlayer(g.Which)
}

// UndoGliderLimbo is Modes.c:472-480. Note DontDraw is cleared whether or not the
// glider was in limbo, and that the dead player in a one-player-left game is
// skipped entirely -- the guard that appears at the top of six functions in this
// file.
func (g *Glider) UndoGliderLimbo(e Env) {
	if g.deadAndDone(e) {
		return
	}
	if g.Mode == GliderInLimbo {
		g.Mode = g.WasMode
	}
	g.DontDraw = false
}

// ToggleGliderFacing is Modes.c:484-493, called when both direction keys are held.
// The early return means the about-face is only available from normal mode: it
// cannot interrupt a burn, a transit or a dissolve.
func (g *Glider) ToggleGliderFacing() {
	if g.Mode != GliderNormal {
		return
	}
	if g.Facing == FaceLeft {
		g.FlagGliderFaceRight()
	} else {
		g.FlagGliderFaceLeft()
	}
}

// InsureGliderFacingRight is Modes.c:497-504: turn the glider to face right if it
// is not already, used when the world needs it pointing a particular way. A burning
// glider is exempt, because its sprite set has no about-face frames.
func (g *Glider) InsureGliderFacingRight(e Env) {
	if g.deadAndDone(e) {
		return
	}
	if g.Facing == FaceLeft && g.Mode != GliderBurning {
		g.FlagGliderFaceRight()
	}
}

// InsureGliderFacingLeft is Modes.c:508-515.
func (g *Glider) InsureGliderFacingLeft(e Env) {
	if g.deadAndDone(e) {
		return
	}
	if g.Facing == FaceRight && g.Mode != GliderBurning {
		g.FlagGliderFaceLeft()
	}
}

// ReadyGliderForTripUpStairs is Modes.c:519-546: the glider emerges from the
// staircase in the room above.
//
// It places Dest by taking the *sprite* rect, zeroing its corner and offsetting it,
// which is the original's idiom for "a glider-sized rect at this position". Then it
// immediately calls the handler for one frame, so the glider is already a step out
// of the stairs on the frame it appears rather than sitting flush with the edge.
func (g *Glider) ReadyGliderForTripUpStairs(e Env) {
	if g.deadAndDone(e) {
		return
	}
	g.Facing = FaceLeft
	g.Mode = GliderComingUp
	g.setSprite(SpriteLeft)
	g.stopDead()

	g.RightClip = e.GetUpStairsRightEdge()
	g.Dest = g.Src.ZeroCorner().Offset(g.RightClip, GliderAppearsComingUp)
	g.Whole = g.Dest
	g.DestShadow.Left = g.Dest.Left
	g.DestShadow.Right = g.Dest.Right
	g.WholeShadow = g.DestShadow

	g.FinishGliderUpStairs(e)
}

// ReadyGliderForTripDownStairs is Modes.c:550-577. The mirror image, except that
// the glider is placed a full glider-width to the *left* of the clip so that it
// emerges rightward from behind the staircase.
func (g *Glider) ReadyGliderForTripDownStairs(e Env) {
	if g.deadAndDone(e) {
		return
	}
	g.Facing = FaceRight
	g.Mode = GliderComingDown
	g.setSprite(SpriteRight)
	g.stopDead()

	g.LeftClip = e.GetDownStairsLeftEdge()
	g.Dest = g.Src.ZeroCorner().Offset(g.LeftClip-GliderWide, GliderAppearsComingDown)
	g.Whole = g.Dest
	g.DestShadow.Left = g.Dest.Left
	g.DestShadow.Right = g.Dest.Right
	g.WholeShadow = g.DestShadow

	g.FinishGliderDownStairs(e)
}

// StartGliderFoilGoing is Modes.c:581-601: the glider has picked up foil.
//
// The guard skips a dissolve already in progress and skips limbo, but not any other
// mode: a burning glider that grabs foil enters the dissolve and comes out of it
// normal, which is how foil puts out a fire.
func (g *Glider) StartGliderFoilGoing(e Env) {
	if g.Mode == GliderGoingFoil || g.Mode == GliderInLimbo {
		return
	}
	e.QuickFoilRefresh(false)
	g.Mode = GliderGoingFoil
	g.Whole = g.Dest
	g.Frame = 0
	g.setFoilSprite(g.Frame)
}

// StartGliderFoilLosing is Modes.c:605-627: the foil has been used up or shredded.
// Identical but for the mode and the extra fizzle.
func (g *Glider) StartGliderFoilLosing(e Env) {
	if g.Mode == GliderLosingFoil || g.Mode == GliderInLimbo {
		return
	}
	e.QuickFoilRefresh(false)
	e.PlayPrioritySound(FizzleSound, FizzlePriority)
	g.Mode = GliderLosingFoil
	g.Whole = g.Dest
	g.Frame = 0
	g.setFoilSprite(g.Frame)
}

// TagGliderIdle is Modes.c:631-639: freeze the glider for 30 frames, one second.
// The countdown is stored in HVel, which is safe only because nothing integrates
// velocity in this mode -- HandleGlider dispatches to HandleIdleGlider, which never
// calls MoveGlider.
func (g *Glider) TagGliderIdle(e Env) {
	if g.deadAndDone(e) {
		return
	}
	g.WasMode = g.Mode
	g.Mode = GliderIdle
	g.HVel = IdleFrames
}

// stopDead zeroes both velocities and both desired velocities, the four-line block
// that appears in six of the functions above. Note it leaves VDesiredVel at 0
// rather than Gravity; see FlagGliderNormal.
func (g *Glider) stopDead() {
	g.HVel = 0
	g.VVel = 0
	g.HDesiredVel = 0
	g.VDesiredVel = 0
	g.Tipped = false
}

// deadAndDone is the guard at the top of UndoGliderLimbo, InsureGliderFacing*,
// ReadyGliderForTrip* and TagGliderIdle: in a two-player game that has come down to
// one survivor, the dead player's glider is left completely alone. Without it the
// corpse would be turned, moved and un-limboed along with the survivor.
func (g *Glider) deadAndDone(e Env) bool {
	return e.TwoPlayerGame() && e.OnePlayerLeft() && g.Which == e.PlayerDead()
}
