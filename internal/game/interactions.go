package game

// The collision dispatcher: Interactions.c's world-facing half --
// HandleHotSpotCollision (:1198-1623), CheckForHotSpots (:1627-1686),
// HandleInteraction (:1691-1710), FlagStillOvers (:1715-1732), WebGlider (:1736-1777)
// and HandleMicrowaveAction (:1159-1193).
//
// The glider-facing half of the same file -- the ten escape checks, SectGlider,
// GliderInRect, GliderHitTop and BounceGlider -- is already in internal/game/player
// (escape.go and hitbox.go), because it only ever touches one glider and the room's
// four openings. What is left here is everything that needs the object graph, the
// inventory or the other player, and that is what makes this the file where a room
// stops being scenery.
//
// **There is no mode gate anywhere in this file, and that is deliberate.** Nothing here
// asks "is this glider in a state where it can be interacted with"; every arm of the
// dispatcher that cares tests the mode itself, and the arms that do not care -- kLiftIt,
// kDropIt, the two fans, kSlideIt, the three kIgnore* -- act on a glider that is
// dissolving, mailed, ducting or dead. So a glider being sucked into a mail slot is
// still lifted by the vent under it. Those writes land on HDesiredVel and VDesiredVel,
// which the transit modes do not read, so most of them are invisible; the three
// kIgnore* flags are cleared by FlagGliderNormal on the way out. A port that added the
// gate would be tidier and would differ, and the difference would show up in exactly
// the rooms that stack a vent under a duct.
//
// ---------------------------------------------------------------------------
// The two-player escape protocol
// ---------------------------------------------------------------------------
//
// Six of the arms below -- kMoveItUp, kMoveItDown, kTransportIt, kMailItLeft,
// kMailItRight, kDuctItDown and kDuctItUp -- carry the same three-branch shape:
//
//	if twoPlayerGame && !onePlayerLeft
//	    if otherPlayerEscaped == kNoOneEscaped        -> ARM: this glider goes first
//	    else if otherPlayerEscaped == <my own code>   -> FOLLOW: the second glider agrees
//	else
//	    -> just do it
//
// That is states 1, 3 and 4 of the limbo machine written out per exit kind (transit.go
// has the whole table). The `else if` compares against *this arm's own* code only, which
// is what makes a mismatched pair deadlock: a glider arriving at a duct while the other
// player is waiting at a staircase matches neither branch, so nothing happens at all --
// it cannot even arm, because kNoOneEscaped is no longer there. It walks into the duct
// and stays.
//
// The four *link* transports add a second test the two staircases do not have:
// `activeRectEscaped == index`. That is what makes the pair have to use the *same*
// object rather than merely the same kind of object, so two players cannot leave through
// two different mailboxes in the same room. World.ActiveRectEscaped has the note on why
// storing a hot-spot index across a frame is safe here and would not be in general.

import "glidergo/internal/game/player"

// Sound IDs and priorities used only from this file (GliderDefines.h:63-180). Declared
// here rather than centrally, following the player package's convention of putting each
// sound next to its one caller -- a sound constant three hundred lines from its use is
// how the wrong priority gets copied.
const (
	MicrowavedSound int16 = 8
	ChordSound      int16 = 23
	SizzleSound     int16 = 47
	WebTwangSound   int16 = 58
	TriggerSound    int16 = 63

	ChordPriority      int16 = 302
	WebTwangPriority   int16 = 310
	SizzlePriority     int16 = 413
	MicrowavedPriority int16 = 811

	// TriggerPriority is 999, the highest in the game, and it is the one priority
	// PlayPrioritySound treats specially rather than merely comparing (Sound.c:47-51).
	TriggerPriority int16 = 999
)

// ---------------------------------------------------------------------------
// HandleInteraction (Interactions.c:1691-1710)
// ---------------------------------------------------------------------------

// HandleInteraction is the whole collision phase of a frame: hot spots first, then the
// room's boundaries.
//
// The order is not arbitrary. CheckForHotSpots is what sets IgnoreLeft, IgnoreRight and
// IgnoreGround, and CheckGliderInRoom is what reads them -- so a glider standing in a
// doorway is let through the wall only because the doorway's kIgnoreLeftWall rect was
// visited earlier in the same call. Swapping the two makes every door in the game
// solid.
//
// The play loop calls this once per frame, after the gliders have moved and before
// RenderFrame.
func (w *World) HandleInteraction() {
	w.CheckForHotSpots()

	if w.TwoPlayer {
		if w.OneLeft {
			if w.DeadWhich == player.Player1 {
				w.P2.CheckGliderInRoom(w)
			} else {
				w.P1.CheckGliderInRoom(w)
			}
		} else {
			w.P1.CheckGliderInRoom(w)
			w.P2.CheckGliderInRoom(w)
		}
	} else {
		w.P1.CheckGliderInRoom(w)
	}
}

// ---------------------------------------------------------------------------
// CheckForHotSpots (Interactions.c:1627-1686)
// ---------------------------------------------------------------------------

// CheckForHotSpots tests both gliders against every live hot spot.
//
// The one-player branch is four lines and the two-player branch is forty, and the
// difference is entirely `hitObject`. StillOver is per *rect*, not per glider, so with
// two players in the room it can only be cleared when *neither* of them is over the
// rect -- otherwise player 1 stepping off a guitar would re-arm the chord for player 2
// standing on it. The one-player branch clears it in an else and the two-player branch
// has to accumulate first.
//
// **The consequence is that two players share one edge detector.** Both standing on the
// same guitar produce one chord between them, and whichever arrives second gets nothing.
// Every `if !stillOver` action in the dispatcher behaves that way; it is the original's
// and is reproduced.
//
// The onePlayerLeft tests read backwards and are correct: `playerDead == kPlayer2` gates
// glider *1*, because when player 2 is the one who is dead, player 1 is the one still
// playing. So the effect is to skip the corpse -- and note that it is skipped *after*
// SectGlider has already been evaluated, so a dead glider's rect still contributes to
// hitObject and can hold StillOver set for the survivor. A glider that died standing on
// a guitar therefore mutes it for the rest of the room.
//
// The index is passed through to the dispatcher because four of the arms store it in
// ActiveRectEscaped. Ranging by index rather than by value is required: the arms write
// to the entry.
func (w *World) CheckForHotSpots() {
	for i := range w.R.Hot {
		who := &w.R.Hot[i]
		if !who.IsOn {
			continue
		}

		if w.TwoPlayer {
			hitObject := false

			if w.P1.SectGlider(player.Rect(who.Bounds), who.DoScrutinize) {
				if !w.OneLeft || w.DeadWhich == player.Player2 {
					w.HandleHotSpotCollision(&w.P1, who, int16(i))
					hitObject = true
				}
			}

			if w.P2.SectGlider(player.Rect(who.Bounds), who.DoScrutinize) {
				if !w.OneLeft || w.DeadWhich == player.Player1 {
					w.HandleHotSpotCollision(&w.P2, who, int16(i))
					hitObject = true
				}
			}

			if !hitObject {
				who.StillOver = false
			}
		} else {
			if w.P1.SectGlider(player.Rect(who.Bounds), who.DoScrutinize) {
				w.HandleHotSpotCollision(&w.P1, who, int16(i))
			} else {
				who.StillOver = false
			}
		}
	}
}

// ---------------------------------------------------------------------------
// FlagStillOvers (Interactions.c:1715-1732)
// ---------------------------------------------------------------------------

// FlagStillOvers pre-arms the edge detector on every rect the glider is already inside,
// so that arriving somewhere does not count as stepping onto it.
//
// One caller: FinishGliderDuctingIn (Player.c:954), the moment a glider drops out of a
// ceiling duct. Without it a duct whose exit sits over a guitar would strike a chord on
// arrival, and -- much worse -- a duct exit placed over another duct's kDuctItUp rect
// would suck the glider straight back in, because that arm is the one gated on
// `!who->stillOver` rather than on a mode.
//
// It writes StillOver for switched-*off* rects too, in the else, which is the one place
// in the game that touches a dark rect's flags. Harmless -- the dispatcher never reaches
// them -- and transcribed because leaving it out would leave a stale true on a rect that
// is switched on later in the same room.
func (w *World) FlagStillOvers(g *player.Glider) {
	for i := range w.R.Hot {
		who := &w.R.Hot[i]
		if who.IsOn {
			who.StillOver = g.SectGlider(player.Rect(who.Bounds), who.DoScrutinize)
		} else {
			who.StillOver = false
		}
	}
}

// ---------------------------------------------------------------------------
// HandleHotSpotCollision (Interactions.c:1198-1623)
// ---------------------------------------------------------------------------

// HandleHotSpotCollision is the switch that turns a rect the glider is touching into
// something happening. Twenty-four of the 28 actions appear; the other four --
// kIgnoreIt, kRewardIt's neighbours and so on -- are covered below.
//
// Two families of arm, and they behave completely differently:
//
//   - The *continuous* arms (kLiftIt, kDropIt, the fans, kSlideIt, the three kIgnore*,
//     kBounceIt) just write a field, every frame, with no test at all. SectGlider
//     overlapping is the whole condition.
//   - The *event* arms re-test with GliderInRect -- a stricter, fully-contained test --
//     or with `!who.stillOver`, and most also exclude the modes in which they would be
//     re-entered. So getting the outer SectGlider is necessary and not sufficient, and
//     the two tests are not interchangeable: a glider clipping the corner of a
//     transporter is registered here and refused by GliderInRect.
//
// **Every one of the seven arms that can move the glider to another room begins by
// checking for fire**, and fire wins: a burning glider that reaches a staircase, a
// transporter, a mailbox or a duct dies on the spot instead of travelling. That is what
// stops fire being carried between rooms, and it is why the fuse is worth outrunning.
// The `wasMode = 0` before each StartGliderFadingOut clears the burn counter, which
// shares the field.
//
// `index` is only read by the four link arms. `who` is a pointer because six arms write
// through it.
func (w *World) HandleHotSpotCollision(thisGlider *player.Glider, who *HotObject, index int16) {
	bounds := player.Rect(who.Bounds)

	switch who.Action {
	// -- the continuous arms ------------------------------------------------

	case LiftIt:
		// Assigned, not added: a floor vent's lift is a target velocity the ramp
		// approaches, so two vents are no stronger than one.
		thisGlider.VDesiredVel = player.FloorVentLift

	case DropIt:
		thisGlider.VDesiredVel = player.CeilingVentDrop

	case PushItLeft:
		// Added, not assigned -- the one asymmetry in the blower set. Two fans facing
		// each other cancel exactly, and a fan plus the player's own input is the sum.
		thisGlider.HDesiredVel += -player.FanStrength

	case PushItRight:
		thisGlider.HDesiredVel += player.FanStrength

	case SlideIt:
		// Spilt grease. Sliding is cleared inside MoveGliderNormal, so it has to be
		// re-set every frame the glider is on the spill -- which is why this is a
		// continuous arm. The velocity write seats the glider exactly on the spill's
		// top edge in one frame rather than letting gravity settle it.
		thisGlider.Sliding = true
		thisGlider.VVel = who.Bounds.Top - thisGlider.Dest.Bottom

	case IgnoreLeftWall:
		thisGlider.IgnoreLeft = true

	case IgnoreRightWall:
		thisGlider.IgnoreRight = true

	case IgnoreGround:
		thisGlider.IgnoreGround = true

	case BounceIt:
		// The only continuous arm that reads the glider's velocity as well as writing
		// it; see player.BounceGlider for the four-way edge selection.
		thisGlider.BounceGlider(w, bounds)

	// -- the one-shot arms --------------------------------------------------

	case StrumIt:
		if !who.StillOver {
			w.PlayPrioritySound(ChordSound, ChordPriority)
			who.StillOver = true
		}

	case ChimeIt:
		if !who.StillOver {
			w.StrikeChime()
			who.StillOver = true
		}

	case SoundIt:
		// kSoundTrigger's own rect, as distinct from kTriggerIt: this one plays the
		// house's custom trigger sound and arms nothing.
		if !who.StillOver {
			w.PlayPrioritySound(TriggerSound, TriggerPriority)
			who.StillOver = true
		}

	case TriggerIt, LgTrigger:
		// **`case kLgTrigger` is dead code in the original.** kLgTrigger is 0x48 -- an
		// *object* type, not an action -- and hot-spot actions only ever run 0..27, so
		// nothing can ever equal it. A large trigger's rect is created with kTriggerIt
		// like any other. Transcribed with its citation because the mistake is
		// informative: it says the author expected the two to need different handling
		// and then found they did not.
		w.ArmTrigger(who)

	case RewardIt:
		w.HandleRewards(thisGlider, who)

	case SwitchIt:
		w.HandleSwitches(who)

	case MicrowaveIt:
		if thisGlider.GliderInRect(bounds) {
			w.HandleMicrowaveAction(who, thisGlider)
		}

	// -- the hazards --------------------------------------------------------

	case DissolveIt:
		// Foil absorbs a dissolve, one sheet per contact -- except from *above*.
		// GliderHitTop is what makes landing on top of the hazard fatal regardless,
		// and it is the only asymmetric hazard in the game.
		if thisGlider.Mode != player.GliderFadingOut {
			if w.Foil > 0 || thisGlider.Mode == player.GliderLosingFoil {
				if thisGlider.GliderHitTop(w, bounds) {
					thisGlider.StartGliderFadingOut(w)
					w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
				} else if w.Foil > 0 {
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

	case ShredIt:
		// The `mode == GliderLosingFoil` disjunct in all three hazards is a grace
		// window: the dissolve animation for losing the last sheet takes several
		// frames, and during them Foil is already 0, so without it the glider would be
		// killed by the very hazard that took its last sheet.
		if thisGlider.Mode != player.GliderShredding && thisGlider.GliderInRect(bounds) {
			if w.Foil > 0 || thisGlider.Mode == player.GliderLosingFoil {
				w.PlayPrioritySound(player.FoilHitSound, player.FoilHitPriority)
				if w.Foil > 0 {
					w.Foil--
					if w.Foil <= 0 {
						thisGlider.StartGliderFoilLosing(w)
					}
				}
			} else {
				thisGlider.FlagGliderShredding(w, bounds)
			}
		}

	case BurnIt:
		// Foil against fire does something the other two hazards do not: it *lifts*.
		// A foil-wrapped glider over a flame is pushed up out of it at the floor
		// vent's speed, which is the game's one piece of positive feedback for holding
		// foil. Note that the sound is inside the `Foil > 0` test but the lift is
		// outside it, so the grace window still lifts and does so silently.
		if thisGlider.Mode != player.GliderBurning && thisGlider.Mode != player.GliderFadingOut {
			if w.Foil > 0 || thisGlider.Mode == player.GliderLosingFoil {
				thisGlider.VDesiredVel = player.FloorVentLift
				if w.Foil > 0 {
					w.PlayPrioritySound(SizzleSound, SizzlePriority)
					w.Foil--
					if w.Foil <= 0 {
						thisGlider.StartGliderFoilLosing(w)
					}
				}
			} else {
				thisGlider.FlagGliderBurning(w)
			}
		}

	case WebIt:
		// The two branches are the same test twice with opposite mode conditions, and
		// the second is what WebGlider's own first three lines already handle -- so a
		// burning glider entering a web is killed by this arm, and would be killed by
		// WebGlider if it got there. The redundancy is the original's.
		if thisGlider.GliderInRect(bounds) && thisGlider.Mode != player.GliderBurning {
			w.WebGlider(thisGlider, bounds)
		} else if thisGlider.Mode == player.GliderBurning && thisGlider.GliderInRect(bounds) {
			thisGlider.WasMode = 0
			thisGlider.StartGliderFadingOut(w)
			w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
		}

	// -- the exits ----------------------------------------------------------

	case MoveItUp:
		// `!heldRight` lets the player refuse the staircase by holding right, which is
		// the only way to walk past an up staircase without taking it. Note that the
		// two staircase arms do *not* test ActiveRectEscaped, so two players can pair
		// up on different staircases in the same room as long as both go the same way.
		if !thisGlider.HeldRight && thisGlider.GliderInRect(bounds) {
			if thisGlider.Mode == player.GliderBurning {
				thisGlider.WasMode = 0
				thisGlider.StartGliderFadingOut(w)
				w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
			} else if w.TwoPlayer && !w.OneLeft {
				if w.Escaped == NoOneEscaped {
					if thisGlider.Mode != player.GliderGoingUp && thisGlider.Mode != player.GliderInLimbo {
						// EscapingUpStairs, not EscapedUpStairs: the walk takes
						// several frames and the code is upgraded when it completes.
						// Both are accepted by MoveRoomToRoom's kAbove arm.
						w.Escaped = player.PlayerEscapingUpStairs
						w.RefreshScoreboard(EscapedTitleMode)
						thisGlider.StartGliderGoingUpStairs(w)
					}
				} else if w.Escaped == player.PlayerEscapedUpStairs {
					if thisGlider.Mode != player.GliderGoingUp && thisGlider.Mode != player.GliderInLimbo {
						thisGlider.StartGliderGoingUpStairs(w)
					}
				}
			} else {
				thisGlider.StartGliderGoingUpStairs(w)
			}
		}

	case MoveItDown:
		// `!heldLeft` -- the mirror of the arm above, and the reason down staircases
		// are refused by holding *left*. The asymmetry follows the artwork: an up
		// staircase is entered walking left and a down staircase walking right.
		if !thisGlider.HeldLeft && thisGlider.GliderInRect(bounds) {
			if thisGlider.Mode == player.GliderBurning {
				thisGlider.WasMode = 0
				thisGlider.StartGliderFadingOut(w)
				w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
			} else if w.TwoPlayer && !w.OneLeft {
				if w.Escaped == NoOneEscaped {
					if thisGlider.Mode != player.GliderGoingDown && thisGlider.Mode != player.GliderInLimbo {
						w.Escaped = player.PlayerEscapingDownStairs
						w.RefreshScoreboard(EscapedTitleMode)
						thisGlider.StartGliderGoingDownStairs(w)
					}
				} else if w.Escaped == player.PlayerEscapedDownStairs {
					if thisGlider.Mode != player.GliderGoingDown && thisGlider.Mode != player.GliderInLimbo {
						thisGlider.StartGliderGoingDownStairs(w)
					}
				}
			} else {
				thisGlider.StartGliderGoingDownStairs(w)
			}
		}

	case TransportIt:
		if thisGlider.Mode == player.GliderBurning {
			thisGlider.WasMode = 0
			thisGlider.StartGliderFadingOut(w)
			w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
		} else if thisGlider.GliderInRect(bounds) &&
			thisGlider.Mode != player.GliderTransporting &&
			thisGlider.Mode != player.GliderFadingOut {
			if w.TwoPlayer && !w.OneLeft {
				if w.Escaped == NoOneEscaped {
					if thisGlider.Mode != player.GliderInLimbo {
						// The arming glider records *which* transporter, and the
						// follower has to match it. This is also the only write to
						// ActiveRectEscaped outside the other three link arms.
						w.ActiveRectEscaped = index
						thisGlider.StartGliderTransporting(w, w.resolveTransitLink(who))
					}
				} else if w.Escaped == player.PlayerTransportedOut {
					if thisGlider.Mode != player.GliderInLimbo && w.ActiveRectEscaped == index {
						thisGlider.StartGliderTransporting(w, w.resolveTransitLink(who))
					}
				}
			} else {
				thisGlider.StartGliderTransporting(w, w.resolveTransitLink(who))
			}
		}

	case MailItLeft:
		// kMailItLeft is a mailbox open to the *right*, and the facing test is what
		// makes a mail slot directional: the glider must be moving into the mouth. The
		// two-term condition reads oddly and is exact -- Facing is where the sprite
		// points and Tipped means the player is pressing the opposite way, so
		// `(FaceRight && !Tipped) || (FaceLeft && Tipped)` is "actually heading right".
		if thisGlider.Mode == player.GliderBurning {
			thisGlider.WasMode = 0
			thisGlider.StartGliderFadingOut(w)
			w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
		} else if thisGlider.GliderInRect(bounds) &&
			thisGlider.Mode != player.GliderMailOutRight &&
			thisGlider.Mode != player.GliderMailInLeft &&
			thisGlider.Mode != player.GliderFadingOut &&
			((thisGlider.Facing == player.FaceRight && !thisGlider.Tipped) ||
				(thisGlider.Facing == player.FaceLeft && thisGlider.Tipped)) {
			if w.TwoPlayer && !w.OneLeft {
				if w.Escaped == NoOneEscaped {
					if thisGlider.Mode != player.GliderInLimbo {
						w.ActiveRectEscaped = index
						thisGlider.StartGliderMailingIn(w, bounds, w.resolveTransitLink(who))
						// The mode is the caller's job for the mail slots and only for
						// them; see player.StartGliderMailingIn.
						thisGlider.Mode = player.GliderMailInLeft
					}
				} else if w.Escaped == player.PlayerMailedOut {
					if thisGlider.Mode != player.GliderInLimbo && w.ActiveRectEscaped == index {
						thisGlider.StartGliderMailingIn(w, bounds, w.resolveTransitLink(who))
						thisGlider.Mode = player.GliderMailInLeft
					}
				}
			} else {
				thisGlider.StartGliderMailingIn(w, bounds, w.resolveTransitLink(who))
				thisGlider.Mode = player.GliderMailInLeft
			}
		}

	case MailItRight:
		// A mailbox open to the left: the same arm with both facing terms inverted.
		if thisGlider.Mode == player.GliderBurning {
			thisGlider.WasMode = 0
			thisGlider.StartGliderFadingOut(w)
			w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
		} else if thisGlider.GliderInRect(bounds) &&
			thisGlider.Mode != player.GliderMailOutLeft &&
			thisGlider.Mode != player.GliderMailInRight &&
			thisGlider.Mode != player.GliderFadingOut &&
			((thisGlider.Facing == player.FaceRight && thisGlider.Tipped) ||
				(thisGlider.Facing == player.FaceLeft && !thisGlider.Tipped)) {
			if w.TwoPlayer && !w.OneLeft {
				if w.Escaped == NoOneEscaped {
					if thisGlider.Mode != player.GliderInLimbo {
						w.ActiveRectEscaped = index
						thisGlider.StartGliderMailingIn(w, bounds, w.resolveTransitLink(who))
						thisGlider.Mode = player.GliderMailInRight
					}
				} else if w.Escaped == player.PlayerMailedOut {
					if thisGlider.Mode != player.GliderInLimbo && w.ActiveRectEscaped == index {
						thisGlider.StartGliderMailingIn(w, bounds, w.resolveTransitLink(who))
						thisGlider.Mode = player.GliderMailInRight
					}
				}
			} else {
				thisGlider.StartGliderMailingIn(w, bounds, w.resolveTransitLink(who))
				thisGlider.Mode = player.GliderMailInRight
			}
		}

	case DuctItDown:
		if thisGlider.Mode == player.GliderBurning {
			thisGlider.WasMode = 0
			thisGlider.StartGliderFadingOut(w)
			w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
		} else if thisGlider.GliderInRect(bounds) &&
			thisGlider.Mode != player.GliderDuctingDown &&
			thisGlider.Mode != player.GliderFadingOut {
			if w.TwoPlayer && !w.OneLeft {
				if w.Escaped == NoOneEscaped {
					if thisGlider.Mode != player.GliderInLimbo {
						w.ActiveRectEscaped = index
						thisGlider.StartGliderDuctingDown(w, bounds, w.resolveTransitLink(who))
					}
				} else if w.Escaped == player.PlayerDuckedOut {
					if thisGlider.Mode != player.GliderInLimbo && w.ActiveRectEscaped == index {
						thisGlider.StartGliderDuctingDown(w, bounds, w.resolveTransitLink(who))
					}
				}
			} else {
				thisGlider.StartGliderDuctingDown(w, bounds, w.resolveTransitLink(who))
			}
		}

	case DuctItUp:
		// The ceiling duct is the one exit with a `!who.stillOver` guard, because it is
		// the one a glider can arrive *out of*: FlagStillOvers pre-arms it so dropping
		// out of a ceiling duct does not immediately re-enter the one below.
		//
		// **The two-player branches do not set StillOver and the one-player branch
		// does** -- an asymmetry in the original with no evident reason. In a
		// two-player game the guard is therefore carried only by the mode test, which
		// holds because StartGliderDuctingUp sets GliderDuctingUp on the same frame.
		if thisGlider.Mode == player.GliderBurning {
			thisGlider.WasMode = 0
			thisGlider.StartGliderFadingOut(w)
			w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
		} else if thisGlider.GliderInRect(bounds) &&
			thisGlider.Mode != player.GliderDuctingUp &&
			thisGlider.Mode != player.GliderDuctingIn &&
			thisGlider.Mode != player.GliderFadingOut &&
			!who.StillOver {
			if w.TwoPlayer && !w.OneLeft {
				if w.Escaped == NoOneEscaped {
					if thisGlider.Mode != player.GliderInLimbo {
						w.ActiveRectEscaped = index
						thisGlider.StartGliderDuctingUp(w, bounds, w.resolveTransitLink(who))
					}
				} else if w.Escaped == player.PlayerDuckedOut {
					if thisGlider.Mode != player.GliderInLimbo && w.ActiveRectEscaped == index {
						thisGlider.StartGliderDuctingUp(w, bounds, w.resolveTransitLink(who))
					}
				}
			} else {
				thisGlider.StartGliderDuctingUp(w, bounds, w.resolveTransitLink(who))
				who.StillOver = true
			}
		}

	default:
		// kIgnoreIt and nothing else. The C has no default either, so an action code
		// outside 0..27 -- which CreateActiveRects cannot produce -- would fall through
		// silently.
	}
}

// ---------------------------------------------------------------------------
// HandleMicrowaveAction (Interactions.c:1159-1193)
// ---------------------------------------------------------------------------

// HandleMicrowaveAction destroys the glider's inventory. It is the only object in the
// game that takes things away without touching the glider itself.
//
// Which things is a bit field in the appliance's byte0, set by the house author: bit 0
// bands, bit 1 battery, bit 2 foil. So a microwave can be wired to erase any subset, and
// the sound plays once for the whole set rather than per item.
//
// **It reads StillOver and never writes it.** Nothing in the game sets StillOver for a
// microwave rect, so the early return only fires when FlagStillOvers pre-armed it -- a
// glider dropping out of a ceiling duct inside a microwave's radiation box. Every other
// time, this runs on *every frame* the glider is inside. It is idempotent by accident:
// after the first frame all three counters are already zero, so `killed` stays false and
// nothing happens. Adding the missing `who.StillOver = true` would be a behaviour change
// only for a glider that picks up bands while standing in a live microwave, which the
// shipped houses do not arrange -- so it is left as it is.
//
// The battery test is `!= 0`, not `> 0`, because the counter is signed and negative
// means helium. A microwave wired to bands erases a helium charge too.
func (w *World) HandleMicrowaveAction(who *HotObject, thisGlider *player.Glider) {
	if who.StillOver {
		return
	}

	killed := false

	if who.Who >= 0 && int(who.Who) < len(w.R.Master) {
		obj := &w.R.Master[who.Who].TheObject
		if obj.Data[offApplianceState] != 0 {
			kills := int16(obj.Data[offApplianceByte0])

			if kills&0x0001 == 0x0001 && w.Bands > 0 {
				w.Bands = 0
				killed = true
				w.QuickBandsRefresh(false)
			}
			if kills&0x0002 == 0x0002 && w.Battery != 0 {
				w.Battery = 0
				killed = true
				w.QuickBatteryRefresh(false)
			}
			if kills&0x0004 == 0x0004 && w.Foil > 0 {
				w.Foil = 0
				killed = true
				// No QuickFoilRefresh here: StartGliderFoilLosing's dissolve is the
				// feedback, and the scoreboard catches up when the mode ends.
				thisGlider.StartGliderFoilLosing(w)
			}
		}
	}

	if killed {
		w.PlayPrioritySound(MicrowavedSound, MicrowavedPriority)
	}
}

// offApplianceByte0 is applianceType.byte0 (GliderStructs.h): the microwave's kill
// mask, a toaster's launch strength, a TV's movie number. Offset 6, alongside the
// offsets in setstate.go.
const offApplianceByte0 = 6

// ---------------------------------------------------------------------------
// WebGlider (Interactions.c:1736-1777)
// ---------------------------------------------------------------------------

// WebGlider is the spider web: it drags the glider toward the web's centre and kills it
// after 150 frames.
//
// The pull is a proportional controller written in two lines. hDist and vDist are the
// sum of the two edge differences shifted right by 3, i.e. one eighth of twice the
// centre offset -- a quarter of the distance to the centre, per axis. Assigning that
// straight to hVel and vVel makes the glider converge geometrically on the middle of the
// web and stay there.
//
// Two things gate it, and together they are the entire mechanic:
//
//   - The pull only happens `if hDesiredVel != 0`, i.e. while the player is *pressing a
//     direction key*. Struggling is what the web responds to. Let go and the else
//     branch zeroes both desired velocities, so the glider hangs still and silent.
//   - Even when struggling, the pull and the twang only happen on even frames, so the
//     sound plays at 15 Hz rather than 30.
//
// The fuse runs regardless. WasMode is the counter -- shared with the burn fuse and with
// FlagGliderInLimbo's saved mode, which is safe only because a glider cannot be webbed
// and burning and in limbo at once -- and it is incremented on *every* frame, struggling
// or not. So holding still does not extend the 150 frames; it only makes them quiet.
//
// The kill is StartGliderFadingOut with WasMode reset first, the same three lines the
// seven exit arms use for a burning glider.
func (w *World) WebGlider(thisGlider *player.Glider, webBounds player.Rect) {
	// KillWebbedGlider is #defined inside the function in the original
	// (Interactions.c:1739) and used at :1774. Five seconds at 30.07 fps.
	const KillWebbedGlider int16 = 150

	// Unreachable from HandleHotSpotCollision, whose kWebIt arm has already routed a
	// burning glider to its own copy of these four lines. Transcribed because it is the
	// function's own precondition and the arm's duplicate is what is redundant.
	if thisGlider.Mode == player.GliderBurning && thisGlider.GliderInRect(webBounds) {
		thisGlider.WasMode = 0
		thisGlider.StartGliderFadingOut(w)
		w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
		return
	}

	hDist := ((webBounds.Right - thisGlider.Dest.Right) +
		(webBounds.Left - thisGlider.Dest.Left)) >> 3
	vDist := ((webBounds.Bottom - thisGlider.Dest.Bottom) +
		(webBounds.Top - thisGlider.Dest.Top)) >> 3

	if thisGlider.HDesiredVel != 0 {
		if w.EvenFrame {
			thisGlider.HVel = hDist
			thisGlider.VVel = vDist
			w.PlayPrioritySound(WebTwangSound, WebTwangPriority)
		}
	} else {
		// Already 0 by the test above; vDesiredVel is the one that matters, because it
		// cancels the gravity FlagGliderNormal would otherwise have restored and is
		// what makes a webbed glider hang rather than sag.
		thisGlider.HDesiredVel = 0
		thisGlider.VDesiredVel = 0
	}

	thisGlider.WasMode++
	if thisGlider.WasMode >= KillWebbedGlider {
		thisGlider.WasMode = 0
		thisGlider.StartGliderFadingOut(w)
		w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
	}
}
