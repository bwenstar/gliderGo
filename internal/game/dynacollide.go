package game

// What a dynamic object does when it touches you: Dynamics.c:34-106, plus the
// two-player fan-out that its eight callers repeat verbatim.
//
// Seventy-three lines of C, kept in their own file because every mover and the outlet
// call into them and because they are the one part of 1.5c that can end a life.

import (
	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/render"
)

// ShoveVelocity is kShoveVelocity (Dynamics.c:17), a file-local #define rather than one
// of GliderDefines.h's. player.ShoveVelocity is the same 8 and the same meaning; this
// package cannot use that one without importing the constant into a comparison it never
// makes, so the citation is here and the value is read from player to keep one copy.
const ShoveVelocity = player.ShoveVelocity

// ---------------------------------------------------------------------------
// CheckDynamicCollision (Dynamics.c:34-74)
// ---------------------------------------------------------------------------

// CheckDynamicCollision decides what one dynamic object does to one glider.
//
// **This is not kDissolveIt and must never share an implementation with it.** Put it
// beside interactions.go's DissolveIt arm and the skeleton looks identical -- foil
// absorbs, no foil kills -- but four things differ and every one of them is observable:
//
//	                 kDissolveIt                   here
//	mode gate        != GliderFadingOut            six named modes (player.CollidesWithDynamics)
//	foil drain       every frame                   EvenFrame only, so half rate
//	survived a hit   silent                        shove +/-8, inherit upward VVel, FoilHitSound
//	from above       GliderHitTop kills through foil   no such case
//
// The mode gate is the sharpest of the four. kDissolveIt blacklists one mode, so a glider
// that is shredding, burning up, ducting, sinking or mid-transit still interacts with a
// blade; the whitelist here means a dart passes harmlessly through all of those. And the
// whitelist admits GliderBurning, so a glider on fire can still be knocked sideways by a
// piece of toast.
//
// The half-rate drain is why a dart is survivable at all. A blade and a dart both touch
// you for as long as you are inside them, but the dart charges one sheet per two frames
// and is simultaneously shoving you away, so one sheet of foil usually carries you
// through one dart where a blade eats the sheet at once.
//
// Three fine points:
//
// **scrutinize is always true.** Dynamics are the only collision family that always
// takes the 5px inset on all four sides; hot spots take it per rect from DoScrutinize
// (interactions.go). A dart has to get 5px inside the glider's box before it counts,
// which is what makes near-misses feel like near-misses.
//
// **VDesiredVel is inherited only when the object is moving up** (`VVel < 0`), and it is
// assigned rather than added, so it replaces what the player asked for. A rising balloon
// lifts you; a falling piece of toast does not press you down and leaves the vertical
// alone entirely.
//
// **doOffset converts screen coordinates back to room-local.** Dest is stored in screen
// space for the seven appliances and room-local for the movers (see AddDynamicObject), and
// this flag is what reconciles the two. It is the outlet -- an appliance -- that passes
// true, and HandleOutlet is where it goes wrong; see checkGliders.
func (w *World) CheckDynamicCollision(who int16, thisGlider *player.Glider, doOffset bool) {
	dinahRect := w.Dinahs[who].Dest
	if doOffset {
		dinahRect = render.Offset(dinahRect, -w.R.V.OriginH, -w.R.V.OriginV)
	}

	if !thisGlider.SectGlider(player.Rect(dinahRect), true) {
		return
	}
	if !player.CollidesWithDynamics(thisGlider.Mode) {
		return
	}

	if w.Foil > 0 || thisGlider.Mode == player.GliderLosingFoil {
		// The LosingFoil disjunct is the same grace window the three hazards in
		// interactions.go have: the dissolve animation for losing the last sheet takes
		// several frames, during which Foil is already 0.
		if render.IsRectLeftOfRect(dinahRect, Rect(thisGlider.Dest)) {
			thisGlider.HDesiredVel = ShoveVelocity
		} else {
			thisGlider.HDesiredVel = -ShoveVelocity
		}
		if w.Dinahs[who].VVel < 0 {
			thisGlider.VDesiredVel = w.Dinahs[who].VVel
		}
		w.PlayPrioritySound(player.FoilHitSound, player.FoilHitPriority)
		if w.EvenFrame && w.Foil > 0 {
			w.Foil--
			if w.Foil <= 0 {
				thisGlider.StartGliderFoilLosing(w)
			}
		}
	} else {
		thisGlider.StartGliderFadingOut(w)
		w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
	}
}

// ---------------------------------------------------------------------------
// The two-player fan-out
// ---------------------------------------------------------------------------

// checkGliders is the twelve-line block that appears eight times, verbatim except for
// one argument: Dynamics.c:333-349 (HandleToast), :534-550 (HandleOutlet), and
// Dynamics2.c:44-60, :145-161, :250-266, :361-376, :435-451, :509-525.
//
// **Seven of the eight callers use this helper; HandleOutlet writes it out longhand** and
// that is deliberate. Dynamics.c:539 and :541 pass doOffset = false where :545, :546 and
// :550 pass true, so the surviving player of a two-player game is tested against an
// unconverted screen rect. Giving this helper two boolean parameters to paper over the
// difference would hide a bug 1.5c is not allowed to fix; see HandleOutlet.
//
// The OneLeft test reads *forwards* here -- `DeadWhich == P1's Which` means glider 1 is
// the corpse, so check glider 2 -- where CheckForHotSpots' reads backwards
// (interactions.go). Both are correct; they are asking different questions, and a reader
// arriving from interactions.go will assume one of them is a bug.
func (w *World) checkGliders(who int16, doOffset bool) {
	if w.TwoPlayer {
		if w.OneLeft {
			if w.DeadWhich == w.P1.Which {
				w.CheckDynamicCollision(who, &w.P2, doOffset)
			} else {
				w.CheckDynamicCollision(who, &w.P1, doOffset)
			}
		} else {
			w.CheckDynamicCollision(who, &w.P1, doOffset)
			w.CheckDynamicCollision(who, &w.P2, doOffset)
		}
	} else {
		w.CheckDynamicCollision(who, &w.P1, doOffset)
	}
}

// ---------------------------------------------------------------------------
// DidBandHitDynamic (Dynamics.c:80-106)
// ---------------------------------------------------------------------------

// Band is bandType (GliderStructs.h:274-279): one rubber band in flight.
//
// Declared here rather than with the rest of RubberBands.c's port because DidBandHitDynamic
// is 1.5c's and reads Dest, and a second declaration in 1.5e would have been a second table
// that could disagree with this one. 1.5e added the *writers* -- AddBand, HandleBands,
// RenderBands, KillAllBands, all in bands.go -- and they use these same five fields
// unchanged, which is what that arrangement was betting on.
//
// So the three band tests in HandleBalloon, HandleCopter and HandleDart are reachable from
// 1.5e onward. Before it they could not fire at all, because NumBands was never non-zero.
type Band struct {
	Dest        Rect
	Mode, Count int16
	HVel, VVel  int16
}

// DidBandHitDynamic reports whether any live rubber band overlaps this object.
//
// An open-coded SectRect -- the same four-test shape as SectGlider with no inset and no
// burning adjustment -- returning on the first hit.
//
// **Dest is not converted here.** There is no doOffset parameter and no offset, and that
// is right: all three callers are movers, whose Dest is room-local, and bands are stored
// room-local too. Worth saying because the function next door does convert and the
// asymmetry reads like an omission.
//
// **Only balloons, copters and darts can be shot down.** Toast, ball, drip and fish are
// not in the caller list (Dynamics2.c:62, :162, :267 are all of them), so a rubber band
// passes straight through them. That is a rule of the game rather than an implementation
// detail, and it is the kind of thing an "improvement" would quietly break.
//
// One latent bug not reproduced, because Go cannot: the C's `collided` is uninitialised
// and is read when numBands is 0. All three callers guard with `numBands > 0` and C's &&
// short-circuits, so it is unreachable in the shipped game; Go's zero value is false,
// which is the answer the code wanted anyway. The callers keep the guard.
func (w *World) DidBandHitDynamic(who int16) bool {
	dinahRect := w.Dinahs[who].Dest

	collided := false
	for i := int16(0); i < w.NumBands; i++ {
		switch {
		case w.BandList[i].Dest.Bottom < dinahRect.Top:
			collided = false
		case w.BandList[i].Dest.Top > dinahRect.Bottom:
			collided = false
		case w.BandList[i].Dest.Right < dinahRect.Left:
			collided = false
		case w.BandList[i].Dest.Left > dinahRect.Right:
			collided = false
		default:
			collided = true
		}
		if collided {
			break
		}
	}
	return collided
}
