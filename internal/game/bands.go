package game

// RubberBands.c in full, plus Render.c's RenderBands -- the game's only projectile.
//
// A rubber band is the one thing the player can put into the world. Two at a time, sixteen
// pixels by six, fired horizontally at 20 px/frame with gravity catching up every fourth
// frame, and it can do almost everything the glider can: trip a switch, arm a fuse, knock a
// grease jar over, bounce off a wall, shoot down a balloon, and shove the glider that fired
// it. What it cannot do is collect a prize or leave the room.
//
// ---------------------------------------------------------------------------
// CheckBandCollision is five phases and they are not independent
// ---------------------------------------------------------------------------
//
//	1  the walls    if the room *has* the wall on that side, clamp and rebound
//	2  the hot spots  a sweep of the whole table, filtered to five of the 28 actions
//	3  the gliders  momentum transfer, one glider or two
//	4  ...          the second glider is phase 3 again under the escape protocol
//	5  the kill     out of bounds or below the floor and the band is deleted
//
// Phase 1 tests `leftThresh == kLeftWallLimit` -- "is there a wall here at all" -- and phase
// 5 tests the *constant*. So in a room with an open left side, phase 1 declines to clamp and
// phase 5 deletes the band the moment it passes x=12 anyway. **A rubber band cannot travel
// through a doorway**, which is a rule of the game and not an accident of the geometry: it
// makes bands a within-room tool and stops a player clearing a room they cannot see.
//
// And the two phases interlock the other way as well. Phase 1's clamp writes
// `dest.left = kLeftWallLimit` exactly, so phase 5's `dest.left < kLeftWallLimit` is false
// by one pixel and the rebounding band survives the frame it rebounds on. One pixel of slack
// is the whole margin; TestBandBouncesOffTheWallAndSurvivesTheKillTest pins it.
//
// ---------------------------------------------------------------------------
// The debounce does not work, and it is transcribed anyway
// ---------------------------------------------------------------------------
//
// `bandHitLast` is meant to stop a band resting against a switch from toggling it sixty
// times a second, and it is a single global holding a single hot-spot index. Three ways it
// fails, all reachable, all reproduced -- see docs/IMPROVEMENTS.md 2.41:
//
//	one band, two rects   after tripping hot spot 5 the sweep continues; hot spot 9 sees
//	                      bandHitLast == 5, so it trips too and leaves bandHitLast == 9.
//	                      Next frame 5 trips again because bandHitLast is 9. Two switches
//	                      within sixteen pixels toggle each other's latch for ever
//	two bands             the reset is per *band*: a band in free flight sets
//	                      nothingCollided and clears bandHitLast to -1, so the second
//	                      band in the air defeats the latch for the first one entirely.
//	                      This is the easy one to hit -- fire twice and hold one band
//	                      against a switch
//	the initial value     it is a zeroed global, not -1, and KillAllBands does not reset
//	                      it. The first collision of the first band of a session is
//	                      swallowed if it happens to be with hot spot 0
//
// World.BandHitLast is an int16 whose Go zero value is 0, which is the third of those for
// free. It is deliberately not initialised to -1.
//
// ---------------------------------------------------------------------------
// What a band is allowed to touch
// ---------------------------------------------------------------------------
//
// Five actions out of 28: kDissolveIt, kRewardIt, kSwitchIt, kTriggerIt, kBounceIt. Two of
// them do something, three of them do something conditionally:
//
//	kDissolveIt, kBounceIt  bounce or die, decided by whether the band's *previous*
//	                        position was already past the obstacle's edge. Both then
//	                        `break` out of the sweep -- the only arm that does
//	kRewardIt               grease only. A band cannot collect a clock, a battery or a
//	                        star; it knocks a jar over and switches the rect off
//	kSwitchIt               HandleSwitches, the same call the glider makes
//	kTriggerIt              ArmTrigger, likewise
//
// The kRewardIt arm is the one worth stating plainly, because "bands cannot collect prizes"
// is a design decision hidden inside a type test. If it were absent a player could farm a
// room's clocks from across it.

import (
	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/render"
)

// RubberBandVelocity is the fixed horizontal speed (RubberBands.c:12): 20 px/frame either
// way, and it never changes except by bouncing, which negates it, or by hitting a glider,
// which zeroes it.
//
// BandFallCount is the gravity divider: vVel is incremented once every four frames rather
// than every frame, so a band's trajectory is much flatter than a glider's. KillBandMode is
// a mode value doubling as a delete flag -- see HandleBands' sweep.
const (
	RubberBandVelocity int16 = 20
	BandFallCount      int16 = 4
	KillBandMode       int16 = -1
)

// The two band sounds (GliderDefines.h:75-76, :102, :130). The rebound is priority 102 --
// nearly the lowest in the game, just above the wall thud -- because a band pinging around a
// room must never drown out a prize.
const (
	FireBandSound       int16 = 20
	BandReboundSound    int16 = 21
	FireBandPriority    int16 = 301
	BandReboundPriority int16 = 102
)

// AddBand is RubberBands.c:256-290, reached from the band key (Input.c:253, :352): fire one.
//
// h and v are the glider's Dest.Left+24 and Dest.Top+10 -- its middle -- and the band is
// then thrown 32 pixels clear in the direction of travel, so it starts outside the glider it
// came from and cannot hit it on the frame it is fired.
//
// Three details carry more weight than they look:
//
// **The recoil is real.** `thisGlider->hVel -= hVel/2` pushes the glider ten pixels a frame
// the other way. Firing is a movement technique, and firing repeatedly against a wall is how
// a player crosses a room with no battery.
//
// **A tipped glider fires upward.** vVel starts at -2 when the player is holding the key
// opposite to their facing, which is the only control the player has over a band's arc.
//
// **The false return is what makes a refused shot free.** Input.c decrements bandsTotal only
// when this returns true, so a third simultaneous band costs no ammunition and does not set
// FireHeld -- the key stays armed and the shot happens as soon as a slot frees.
func (w *World) AddBand(thisGlider *player.Glider, h, v int16, direction bool) bool {
	if w.NumBands >= MaxRubberBands {
		return false
	}

	b := &w.BandList[w.NumBands]
	b.Mode = 0
	b.Count = 0
	if thisGlider.Tipped {
		b.VVel = -2
	} else {
		b.VVel = 0
	}
	b.Dest = render.SetRect(h-8, v-3, h+8, v+3)

	if direction == player.FaceLeft {
		b.Dest = render.Offset(b.Dest, -32, 0)
		b.HVel = -RubberBandVelocity
	} else {
		b.Dest = render.Offset(b.Dest, 32, 0)
		b.HVel = RubberBandVelocity
	}

	thisGlider.HVel -= b.HVel / 2
	w.NumBands++

	w.PlayPrioritySound(FireBandSound, FireBandPriority)
	return true
}

// HandleBands is RubberBands.c:208-252, called from PlayGame's main loop outside every
// gameOver guard: advance, collide and reap every band in flight.
//
// The ordering inside the loop is what a reader has to get right, because it is the reason a
// band's trail is erased:
//
//	1  the animation cel advances, 0,1,2 and wrap
//	2  gravity, once every BandFallCount frames
//	3  **the rect the band is about to leave** is registered as a work rect
//	4  the band moves
//	5  it is collided against the room
//
// Step 3 uses the *pre-move* rect. That is the erase: the dirty rect covering where the band
// was is what lets RestoreWorkMap put the background back before RenderBands draws it in its
// new place. Register the post-move rect instead and every band leaves a permanent smear.
//
// The reaping loop at the bottom is a do-while over a table that shrinks underneath it, with
// an inner `while` rather than an `if` because KillBand fills the hole with the *last* band,
// which may itself be dying. `mode = 0` before the kill is dead in the original -- KillBand
// overwrites the slot -- but it is what makes the inner loop terminate when the slot being
// killed *is* the last one, so it is not removable.
func (w *World) HandleBands() {
	if w.NumBands == 0 {
		return
	}

	for i := int16(0); i < w.NumBands; i++ {
		b := &w.BandList[i]

		b.Mode++
		if b.Mode > 2 {
			b.Mode = 0
		}

		b.Count++
		if b.Count >= BandFallCount {
			b.VVel++
			b.Count = 0
		}

		// The pre-move rect. See the function comment: this is the erase.
		dest := render.Offset(b.Dest, w.R.V.OriginH, w.R.V.OriginV)
		w.AddRectToWorkRects(player.Rect(dest))

		b.Dest = render.Offset(b.Dest, b.HVel, b.VVel)

		w.CheckBandCollision(i)
	}

	count := int16(0)
	for {
		for w.BandList[count].Mode == KillBandMode {
			w.BandList[count].Mode = 0
			w.KillBand(count)
		}
		count++
		if count >= w.NumBands {
			break
		}
	}
}

// CheckBandCollision is RubberBands.c:37-204: one band against the whole room.
//
// The five phases are laid out in the file comment. What is not there, because it belongs
// next to the code, is that the C's `collided` is declared uninitialised and every path that
// reads it assigns it first -- the wall phase's `collided = true` is never read by anything.
// Go's zero value makes that explicit rather than merely true.
func (w *World) CheckBandCollision(who int16) {
	b := &w.BandList[who]
	nothingCollided := true
	collided := false

	// ---- phase 1: the walls -------------------------------------------------
	//
	// Guarded on the threshold *being* the wall limit, which is how the port asks "is
	// there a wall on this side" -- see Room.LeftThresh. An open side is not clamped, and
	// phase 5 kills the band there instead.
	//
	// The `if hVel < 0` inside the left arm is not redundant with the position test: a
	// band that has already rebounded and is travelling right can still be sitting left
	// of the limit for one frame, and negating its velocity again would trap it. The
	// clamp still runs, and it is the clamp that phase 5 depends on.
	if w.R.LeftThresh == LeftWallLimit && b.Dest.Left < LeftWallLimit {
		if b.HVel < 0 {
			b.HVel = -b.HVel
		}
		b.Dest.Left = LeftWallLimit
		b.Dest.Right = b.Dest.Left + 16
		w.PlayPrioritySound(BandReboundSound, BandReboundPriority)
		collided = true
	} else if w.R.RightThresh == RightWallLimit && b.Dest.Right > RightWallLimit {
		if b.HVel > 0 {
			b.HVel = -b.HVel
		}
		b.Dest.Right = RightWallLimit
		b.Dest.Left = b.Dest.Right - 16
		w.PlayPrioritySound(BandReboundSound, BandReboundPriority)
		collided = true
	}

	// ---- phase 2: the hot spots ---------------------------------------------
	//
	// The label is the C's `break`. In C that break is inside a chain of `if`s and so
	// leaves the *loop*; here the arm sits in a `switch`, where a bare break would leave
	// only the switch and let the sweep continue against a rect that has just moved.
sweep:
	for i := range w.R.Hot {
		hot := &w.R.Hot[i]
		if !hot.IsOn {
			continue
		}

		action := hot.Action
		if action != DissolveIt && action != RewardIt &&
			action != SwitchIt && action != TriggerIt && action != BounceIt {
			continue
		}

		// An open-coded SectRect, the same four-test shape as DidBandHitDynamic and
		// with no inset -- so DoScrutinize is ignored here. A band gets the loose hit
		// box against a hazard where a glider gets the tight one.
		switch {
		case b.Dest.Bottom < hot.Bounds.Top:
			collided = false
		case b.Dest.Top > hot.Bounds.Bottom:
			collided = false
		case b.Dest.Right < hot.Bounds.Left:
			collided = false
		case b.Dest.Left > hot.Bounds.Right:
			collided = false
		default:
			collided = true
		}
		if !collided {
			continue
		}

		nothingCollided = false
		if w.BandHitLast == int16(i) {
			// The debounce, such as it is. Note that it suppresses the *effect* and
			// not the collision: nothingCollided is already false above, so a band
			// held against a switch keeps the latch alive.
			continue
		}
		w.BandHitLast = int16(i)

		switch {
		case action == DissolveIt || action == BounceIt:
			// Bounce or die, and the test is whether the band's *previous* position
			// was already clear of the obstacle's near edge. If it was, this frame's
			// step carried it into the face of the thing and it rebounds off that
			// face; if it was not, the band was already inside the obstacle when the
			// frame began and it is deleted instead.
			//
			// So a band fired at a wall bounces and a band fired at a table leg it
			// is already overlapping is absorbed. Twenty pixels a frame is enough
			// for either to happen.
			if b.HVel > 0 {
				if b.Dest.Right-b.HVel < hot.Bounds.Left {
					b.HVel = -b.HVel
					b.Dest.Right = hot.Bounds.Left
					b.Dest.Left = b.Dest.Right - 16
				} else {
					b.Mode = KillBandMode
				}
			} else {
				if b.Dest.Left-b.HVel > hot.Bounds.Right {
					b.HVel = -b.HVel
					b.Dest.Left = hot.Bounds.Right
					b.Dest.Right = b.Dest.Left + 16
				} else {
					b.Mode = KillBandMode
				}
			}
			w.PlayPrioritySound(BandReboundSound, BandReboundPriority)
			// The only arm that stops the sweep. A band that bounced off one
			// obstacle is not also collided against the next, which matters because
			// its rect has just been moved.
			break sweep

		case action == RewardIt:
			// **Grease only.** Every other prize is unreachable by a band, which is
			// the design decision this type test is hiding.
			//
			// The body is HandleRewards' grease arm inlined, minus the IsOn write,
			// which is done here instead and unconditionally: the rect is switched
			// off even when SetObjectState refused, where the reward path leaves
			// that to `who.IsOn = false` at the end of its arm. Same outcome,
			// different route.
			whoLinked := hot.Who
			if !w.masterValid(whoLinked) {
				break
			}
			m := &w.R.Master[whoLinked]
			if m.TheObject.What == GreaseRt || m.TheObject.What == GreaseLf {
				if w.SetObjectState(w.R.RoomNumber, m.ObjectNum, Toggle, whoLinked) {
					w.SpillGrease(m.DynaNum, m.HotNum)
				}
				hot.IsOn = false
			}

		case action == SwitchIt:
			// The same call the glider makes, with the same StillOver latch inside
			// it -- so a band and a glider on one switch share the edge detector.
			w.HandleSwitches(hot)

		case action == TriggerIt:
			w.ArmTrigger(hot)
		}
	}
	if nothingCollided {
		w.BandHitLast = -1
	}

	// ---- phases 3 and 4: the gliders ----------------------------------------
	//
	// Gated on `hVel != 0`, so a band that has already given its momentum to one glider
	// cannot give it to the other -- and a band travelling straight down hits nobody.
	if b.HVel != 0 {
		switch {
		case b.Dest.Bottom < w.P1.Dest.Top:
			collided = false
		case b.Dest.Top > w.P1.Dest.Bottom:
			collided = false
		case b.Dest.Right < w.P1.Dest.Left:
			collided = false
		case b.Dest.Left > w.P1.Dest.Right:
			collided = false
		default:
			collided = true
		}

		if collided {
			// Half the band's velocity to the glider, and the band stops dead --
			// it does not bounce off a glider, it sticks and falls. The escape
			// protocol reads backwards and is correct: `playerDead == kPlayer2`
			// gates glider *1*, because player 2 being dead is what makes player 1
			// the survivor. See CheckForHotSpots' note.
			if !w.TwoPlayer || !w.OneLeft || w.DeadWhich == player.Player2 {
				w.P1.HVel += b.HVel / 2
				b.HVel = 0
				w.PlayPrioritySound(player.HitWallSound, player.HitWallPriority)
			}
		}

		if w.TwoPlayer {
			switch {
			case b.Dest.Bottom < w.P2.Dest.Top:
				collided = false
			case b.Dest.Top > w.P2.Dest.Bottom:
				collided = false
			case b.Dest.Right < w.P2.Dest.Left:
				collided = false
			case b.Dest.Left > w.P2.Dest.Right:
				collided = false
			default:
				collided = true
			}

			if collided {
				if !w.OneLeft || w.DeadWhich == player.Player1 {
					w.P2.HVel += b.HVel / 2
					b.HVel = 0
					w.PlayPrioritySound(player.HitWallSound, player.HitWallPriority)
				}
			}
		}
	}

	// ---- phase 5: the kill --------------------------------------------------
	//
	// The constants, not the thresholds. See the file comment: this is what stops a band
	// leaving through a doorway, and it is one pixel clear of phase 1's clamp.
	//
	// It runs even for a band phase 2 has already marked for deletion, which is harmless
	// -- KillBandMode is the value it would write.
	if b.Dest.Left < LeftWallLimit || b.Dest.Right > RightWallLimit {
		b.Mode = KillBandMode
	} else if b.Dest.Bottom > FloorLimit {
		b.Mode = KillBandMode
	}
}

// KillBand is RubberBands.c:294-303: delete one band by moving the last one into its slot.
//
// An unordered swap-remove, which is why HandleBands' reaping loop re-tests the same index
// rather than advancing: the band that just landed there has not been examined.
func (w *World) KillBand(which int16) {
	lastBand := w.NumBands - 1
	if which != lastBand {
		w.BandList[which] = w.BandList[lastBand]
	}
	w.NumBands--
}

// KillAllBands is RubberBands.c:307-317, the third line of DrawLocale's reset head: every
// band in flight is deleted on a room change.
//
// It zeroes all MaxRubberBands modes rather than just the live ones, which is the C's and is
// the only reason the reaping loop can read one slot past NumBands safely. It does **not**
// reset bandHitLast; see the file comment on the debounce.
func (w *World) KillAllBands() {
	for i := range w.BandList {
		w.BandList[i].Mode = 0
	}
	w.NumBands = 0
}

// RenderBands is Render.c:534-555, the last call in RenderFrame: draw every band in flight.
//
// Last means on top of everything, gliders included -- a band crossing in front of the
// glider that fired it is drawn over it. It registers both a work rect and a back rect,
// which is the pair a moving object needs: the work rect puts this frame on screen and the
// back rect asks for the background to be restored here before the next one.
func (w *World) RenderBands() {
	if w.NumBands == 0 {
		return
	}

	art := w.R.A.Strip("bands")
	for i := int16(0); i < w.NumBands; i++ {
		dest := render.Offset(w.BandList[i].Dest, w.R.V.OriginH, w.R.V.OriginV)
		if art != nil {
			w.R.Work.Copy(art, render.BandRects[w.BandList[i].Mode], dest, render.Masked)
		}
		w.AddRectToWorkRects(player.Rect(dest))
		w.AddRectToBackRects(player.Rect(dest))
	}
}
