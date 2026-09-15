package game

// Room changes: GliderPRO/Sources/Transit.c, all nine functions.
//
// Four of them are the four ways the room under the glider is replaced --
// MoveRoomToRoom for a wall, a ceiling or a staircase, and TransportRoomToRoom,
// MoveDuctToDuct and MoveMailToMail for the three link transports. They are the point
// the game stops being one room. Everything else in the file exists to serve them:
// WhatAreWeLinkedTo and ReadyGliderFromTransit place a glider at the far end,
// HandleRoomVisitation pays for the room, and ForceKillGlider and FollowTheLeader are
// the two escapes from a two-player deadlock.
//
// All four end with the same five steps in the same order, and the order is the whole
// reason a room change looks like one event rather than several:
//
//	ForceThisRoom      the room's identity changes -- nothing is drawn
//	ReadyLevel         the new locale is composed into the work and back maps
//	RefreshScoreboard  the title reverts from "waiting" to the house name
//	WipeScreenOn       the work map is revealed on screen, four pixels at a time
//	RenderFrame        the gliders are drawn into it and shown
//
// Reordering any two of those five is visible. WipeScreenOn before ReadyLevel wipes on
// the *old* room; WipeScreenOn after RenderFrame paints the glider-free work map back
// over the gliders that were just blitted (see specDisagreement 3 in
// docs/analysis/stage15-raw/plans/transit.md).
//
// ---------------------------------------------------------------------------
// The two-player limbo machine
// ---------------------------------------------------------------------------
//
// Both players share one room. So when one of them reaches an exit the room cannot
// change yet, and the whole of Glider PRO's two-player mode is the six-state dance
// that follows. Nothing in the C names it; this is that machine written out, because
// four of its states are only reachable through combinations no single function shows.
//
// The state lives in three globals: `otherPlayerEscaped` (World.Escaped) holds *how*
// the waiting player left, `firstPlayer` (World.FirstPlayer) holds *who* they are, and
// the waiting glider's own mode is GliderInLimbo.
//
//  1. ARM. The first glider to reach an exit finds Escaped == NoOneEscaped. It writes
//     its own escape code there, shows the "waiting" banner
//     (RefreshScoreboard(EscapedTitleMode)) and calls FlagGliderInLimbo, which freezes
//     it and sets FirstPlayer to its own Which. Seventeen sites do this, one per way
//     out of a room per player. The room does not change.
//
//  2. WAIT. HandleGlider does nothing for a glider in limbo, so the other player keeps
//     playing in the same room, alone. Note that the frozen glider is usually still
//     *drawn*, half out of the doorway: FlagGliderInLimbo never sets DontDraw (see
//     Glider.DontDraw).
//
//  3. SHORT-CIRCUIT. The second glider reaches the *same kind* of exit. It finds its
//     own escape code already in Escaped, clears it back to NoOneEscaped, and calls the
//     transit handler directly -- which brings the waiting glider along by calling
//     UndoGliderLimbo on both. This is the intended path and the only one that costs
//     nothing.
//
//  4. REFUSE. The second glider reaches a *different* exit. The test is against its own
//     code only, so the mismatch falls through to the arm branch: this glider goes into
//     limbo too, overwriting Escaped and FirstPlayer with its own. Both gliders are now
//     frozen and neither can reach anything. The game is deadlocked, and the only way
//     out is state 5. One player leaving by the left door while the other leaves by the
//     right is enough to reach it.
//
//  5. GIVE UP. Player 1 presses the give-up key (Input.c:366-371 -- player 2 has no
//     such key, see docs/IMPROVEMENTS.md 2.23). ForceKillGlider fades out whichever glider
//     is *not* in limbo and sets playerSuicide, so the player who is still free spends
//     a mortal to release the one who is waiting: OffAMortal sees the suicide flag and
//     calls FollowTheLeader instead of respawning, and FollowTheLeader performs the
//     room change Escaped has been holding. In the deadlock of state 4 the two tests
//     are read in order, so it is glider 2 that is killed.
//
//  6. PERMANENT DEATH. A glider spends its last mortal. OffAMortal puts it in limbo
//     with DontDraw set, raises OnePlayerLeft and records PlayerDead, then moves the
//     *survivor* through whatever Escaped was holding and writes
//     PlayerIsDeadForever (-69) into it -- a code no arm of any dispatch matches, so no
//     later FollowTheLeader can fire. From then on the dead-glider guard at the head of
//     ReadyGliderFromTransit and OffsetGlider silently drops every attempt to move the
//     corpse, which is what keeps it from being dragged into rooms it is not in.
//
// The port keeps all six, bugs included. IMPROVEMENTS.md 2.22 offers the deadlock and the
// missing second give-up key as opt-in fixes behind a compatibility flag, because a
// released two-player mode that can deadlock is not shippable and a fidelity replay
// that cannot deadlock is not faithful.

import (
	"glidergo/internal/game/player"
	"glidergo/internal/render"
)

// ---------------------------------------------------------------------------
// WhatAreWeLinkedTo (Transit.c:31-61) and the link resolution around it
// ---------------------------------------------------------------------------

// WhatAreWeLinkedTo asks what kind of thing a transit object's far end is.
//
// It exists because arriving somewhere is not the same operation as leaving: a glider
// that steps into a floor duct might come out of a ceiling duct, a mail slot facing
// left, or a transporter, and each needs a different mode, position and clip. The
// answer is the destination *object's* type, not the source's, and
// ReadyGliderFromTransit switches on it.
//
// Only three of the five LinkedTo codes are ever returned. There is no kFloorTrans
// case, so a duct wired to a floor transporter comes back as LinkedToOther and the
// glider materialises out of it as though it were a transporter -- which makes
// ReadyGliderFromTransit's LinkedToFloorDuct arm dead code. It is transcribed there
// anyway; see that switch.
//
// **The C has no bounds check and cannot.** `who` is declared Byte, and the value
// handed to it is masterObjects[].objectLink, which is -1 for an object with no link.
// -1 as a Byte is 255, so the C indexes objects[255] of a 24-slot array and reads
// 2.3 KB past the end of the room -- in practice the middle of some later room, whose
// `what` is unlikely to be a mailbox or a ceiling transporter, so the read almost
// always lands on the default arm and returns LinkedToOther.
//
// This port guards the index and returns LinkedToOther, which is the answer that read
// almost always produced, arrived at honestly. It is a deliberate divergence, and the
// case that reaches it -- an unlinked transit object -- is an authoring error a house
// linter should report (IMPROVEMENTS.md 4.1).
func (w *World) WhatAreWeLinkedTo(where int16, who int) int16 {
	rm := w.Room(where)
	if rm == nil || w.badIndex(devRoomObject, who, MaxRoomObs) {
		return player.LinkedToOther
	}

	switch rm.Objects[who].What {
	case MailboxLf:
		return player.LinkedToLeftMailbox

	case MailboxRt:
		return player.LinkedToRightMailbox

	case CeilingTrans:
		return player.LinkedToCeilingDuct

	default:
		return player.LinkedToOther
	}
}

// resolveTransitLink is the eleven-line block that heads StartGliderMailingIn,
// StartGliderDuctingDown, StartGliderDuctingUp and StartGliderTransporting
// (Modes.c:163-171, :226-234, :259-267, :301-309), character-identical in all four.
//
// It has no name in the original because it is written out four times. It is hoisted
// here because it is object-graph work and those four functions live in the player
// package, which cannot see the object graph -- so the caller resolves the link and
// passes a player.Link in. See player.Link.
//
// It writes the answer to **two places**, and both are needed:
//
//	World.Pending  is the C's three globals -- transRoom, transRect, linkedToWhat.
//	g.Transit      is the per-glider copy the player package's own handlers read.
//
// The globals are what make the two-player handshake work. StartGliderMailingOut is
// called for *both* gliders on arrival, including the follower who never touched a
// mailbox and whose own Transit still holds whatever it last entered -- and it picks
// its exit side from linkedToWhat. So the follower has to see the leader's link, which
// is exactly what ReadyGliderFromTransit's `g.Transit = w.Pending` does.
//
// The last-writer-wins is therefore reproduced rather than gated. If both players step
// into transit objects on the same frame the second overwrites the first's Pending, and
// both then arrive at the second's destination. That is the original's behaviour and
// the reason the shared slot exists at all.
func (w *World) resolveTransitLink(who *HotObject) player.Link {
	var link player.Link

	if who != nil && !w.badIndex(devMasterObject, int(who.Who), len(w.R.Master)) {
		mo := &w.R.Master[who.Who]
		link.Room = mo.RoomLink
		objLinked := mo.ObjectLink
		link.What = w.WhatAreWeLinkedTo(link.Room, int(objLinked))

		// GetObjectRect(&(*thisHouse)->rooms[transRoom].objects[objLinked], &transRect).
		// Note that this is the destination object's rect in *its own room's* local
		// coordinates, which is what makes it usable directly: the destination room is
		// about to become the central room, so its local coordinates become the
		// glider's.
		if rm := w.Room(link.Room); rm != nil && !w.badIndex(devRoomObject, int(objLinked), MaxRoomObs) {
			var r Rect
			w.R.A.GetObjectRect(&rm.Objects[objLinked], &r)
			link.Rect = player.Rect(r)
		}
	}

	w.Pending = link
	return link
}

// ---------------------------------------------------------------------------
// HandleRoomVisitation (Transit.c:430-445)
// ---------------------------------------------------------------------------

// HandleRoomVisitation credits the player with a room, once ever.
//
// **It credits the room being left, not the one being entered.** Every caller is the
// first statement of a transit handler, before ForceThisRoom, so `thisRoom` is still
// the room the glider is walking out of. The consequence is visible on the scoreboard:
// the first room of a house scores nothing until the player leaves it, and a player who
// dies without ever leaving the first room finishes with a score of zero.
//
// The C writes `visited` twice -- once into the house handle and once into the
// `thisRoom` working copy -- because those are two separate structures that both have
// the field. This port has no separate copy (see World.ThisRoom), so the two writes
// collapse into one and the copy cannot go stale. The two indices the C uses,
// localNumbers[kCentralRoom] and thisRoom, always name the same room here, because
// DrawLocale sets the former from the latter and nothing between them can run.
//
// `visited` is a byte on disk and is written 1 rather than true, which is what the
// house codec round-trips (see internal/house's corpus test, which pins every shipped
// room's value at 0 or 1).
func (w *World) HandleRoomVisitation() {
	rm := w.ThisRoom()
	if rm == nil || rm.Visited != 0 {
		return
	}
	rm.Visited = 1
	w.Score += RoomVisitScore
}

// ---------------------------------------------------------------------------
// ReadyGliderFromTransit (Transit.c:65-147)
// ---------------------------------------------------------------------------

// ReadyGliderFromTransit places a glider at the far end of a link transport.
//
// Called once per glider by each of the three link handlers, with the *shared*
// linkedToWhat as its argument -- so in a two-player game both gliders arrive out of
// the same kind of object, from the same rect, whether or not they both entered one.
//
// The four live arms differ in more than position. Two of them leave Dest degenerate on
// purpose: the mailbox arms make it zero-*width* at the slot's mouth and the ceiling
// duct arm makes it zero-*height* just above the duct, because the arrival animation
// grows the glider out of the opening over the following frames rather than fading it
// in. A port that "fixed" either into a full 48x20 rect would have the glider appear
// whole and then be re-clipped by its own handler on the next frame.
//
// Only the transporter arm sets EnteredRect. So a glider that dies after arriving by
// mail or by duct respawns at wherever it last became normal *in a previous room* --
// which, because FlagGliderNormal is called at the top of this function and does not
// touch EnteredRect either, can be a room it is no longer in. That is reachable in the
// shipped houses and is reproduced.
//
// The literals 64, 79, 25 and 4 are the original's, are undocumented there, and are
// not any of the named glider or slot dimensions. They are kept as literals rather than
// given invented names.
func (w *World) ReadyGliderFromTransit(g *player.Glider, toWhat int16) {
	// The dead-glider guard, identical to the one heading OffsetGlider: in a
	// two-player game where one player is finished, the corpse is not moved.
	if w.TwoPlayer && w.OneLeft && g.Which == w.DeadWhich {
		return
	}

	// The C reads the transRect global from here down. This is the assignment that
	// stands in for that, and it is why the follower in a two-player game exits the
	// correct side of the destination mailbox -- see resolveTransitLink. It has to be
	// after the guard, so that a corpse's stale link is left alone, and before the
	// switch, because StartGliderMailingOut reads it.
	g.Transit = w.Pending

	g.FlagGliderNormal(w)

	switch toWhat {
	case player.LinkedToOther:
		g.StartGliderTransportingIn(w)

		// CenterRectInRect(&tempRect, &transRect) on a copy of dest, which
		// FlagGliderNormal has just snapped to 48x20.
		tempRect := player.Rect(render.CenterIn(Rect(g.Dest), Rect(g.Transit.Rect)))

		// The C assigns all four edges of dest individually; a struct assignment is
		// the same four writes, and nothing between them reads dest.
		g.Dest = tempRect
		// The shadow takes only the horizontal pair. Its Top and Bottom are left
		// where FlagGliderNormal put them, on the floor plane of the *new* room.
		g.DestShadow.Left = tempRect.Left
		g.DestShadow.Right = tempRect.Right
		g.Whole = g.Dest
		g.WholeShadow = g.DestShadow
		g.EnteredRect = g.Dest

	case player.LinkedToLeftMailbox:
		g.StartGliderMailingOut(w)

		g.Clip = g.Transit.Rect
		g.Clip.Right -= 64
		g.Clip.Bottom -= 25

		tempRect := g.Dest
		g.Dest.Left = g.Clip.Right
		g.Dest.Right = g.Dest.Left // zero width: the glider slides out leftward
		g.Dest.Bottom = g.Clip.Bottom - 4
		g.Dest.Top = g.Dest.Bottom - tempRect.Tall()
		g.DestShadow.Left = g.Dest.Left
		g.DestShadow.Right = g.Dest.Right
		g.Whole = g.Dest
		g.WholeShadow = g.DestShadow

	case player.LinkedToRightMailbox:
		g.StartGliderMailingOut(w)

		g.Clip = g.Transit.Rect
		g.Clip.Left += 79
		g.Clip.Bottom -= 25

		tempRect := g.Dest
		g.Dest.Right = g.Clip.Left
		g.Dest.Left = g.Dest.Right // zero width: the glider slides out rightward
		g.Dest.Bottom = g.Clip.Bottom - 4
		g.Dest.Top = g.Dest.Bottom - tempRect.Tall()
		g.DestShadow.Left = g.Dest.Left
		g.DestShadow.Right = g.Dest.Right
		g.Whole = g.Dest
		g.WholeShadow = g.DestShadow

	case player.LinkedToCeilingDuct:
		// The one Start* that takes no Env: it neither plays a sound nor settles foil,
		// because the glider is already positioned and merely becomes visible.
		g.StartGliderDuctingIn()

		tempRect := player.Rect(render.CenterIn(Rect(g.Dest), Rect(g.Transit.Rect)))
		g.Dest.Left = tempRect.Left
		g.Dest.Right = tempRect.Right
		g.Dest.Top = tempRect.Top
		g.Dest.Bottom = g.Dest.Top // zero height, then lifted clear of the duct
		g.Dest = g.Dest.Offset(0, -tempRect.Tall())
		g.DestShadow.Left = tempRect.Left
		g.DestShadow.Right = tempRect.Right
		g.Whole = g.Dest
		g.WholeShadow = g.DestShadow

	case player.LinkedToFloorDuct:
		// Dead code: WhatAreWeLinkedTo has no kFloorTrans case and so never returns
		// this. A duct linked to a floor transporter arrives as LinkedToOther instead.
		// Transcribed because its emptiness is the evidence -- an author wiring a duct
		// to a floor duct gets a transporter materialisation, not nothing.

	default:
	}

	// The arrival freeze. The glider that is *not* FirstPlayer is handed to
	// TagGliderIdle, which holds it still for 30 frames so the two do not both start
	// flying the instant the room appears.
	//
	// FirstPlayer is never initialised in the C -- it is a BSS Boolean, false, i.e.
	// kPlayer2 -- and World.FirstPlayer is left at Go's false to match. So before any
	// glider has ever gone into limbo the *player 1* glider is the one frozen on
	// arrival, which is the opposite of what the name suggests. See World.FirstPlayer.
	if w.TwoPlayer && g.Which != w.FirstPlayer {
		g.TagGliderIdle(w)
	}
}

// ---------------------------------------------------------------------------
// MoveRoomToRoom (Transit.c:151-310)
// ---------------------------------------------------------------------------

// MoveRoomToRoom is the room change through a wall, a ceiling, a floor or a staircase.
//
// `where` is Above, Below, ToRight or ToLeft and means the direction of *travel*, so it
// selects both the neighbour room and, at the end, the direction the wipe runs.
//
// Three orderings inside it are load-bearing:
//
//  1. HandleRoomVisitation is called before ForceThisRoom, which is what makes the
//     score credit the room being left. See that function.
//
//  2. The facing fix-ups (InsureGliderFacingRight/Left) run before ForceThisRoom, and
//     OffsetGlider runs after it. The first pair care only about the glider; the
//     enterRect that follows reads `thisRoom->leftStart`, which must be the *new*
//     room's.
//
//  3. UndoGliderLimbo runs before OffsetGlider. Undoing limbo restores the glider's
//     mode and rects from where it froze, so offsetting first would move the frozen
//     position and then throw it away.
//
// The horizontal and vertical arms are not symmetric, and neither is their music. The
// two side doors send ProdGameScoreMode -- a nudge back to the top of the score -- and
// the two vertical exits send KickGameScoreMode, which jumps to piece 2. Walking
// sideways through a house therefore restarts the music and climbing does not.
//
// **The takingTheStairs bug is reproduced.** The flag is set at Player.c:344 and :472
// and cleared only in DrawLocale (RoomGraphics.c:127), so it survives a glider going
// into limbo at the top of a staircase. If the other player then leaves through the
// *ceiling* rather than by the same stairs, this function's kAbove arm still sees the
// flag set and hands both gliders to ReadyGliderForTripUpStairs -- so they arrive
// walking up a staircase in a room that may not have one. Reachable in any room with
// both an up staircase and an open ceiling.
func (w *World) MoveRoomToRoom(g *player.Glider, where int16) {
	w.HandleRoomVisitation()

	switch where {
	case player.ToRight:
		w.SetMusicalMode(ProdGameScoreMode)
		if w.TwoPlayer {
			w.P1.UndoGliderLimbo(w)
			w.P2.UndoGliderLimbo(w)
			w.P1.InsureGliderFacingRight(w)
			w.P2.InsureGliderFacingRight(w)
		} else {
			g.InsureGliderFacingRight(w)
		}
		w.ForceThisRoom(w.R.LocalNumbers[EastRoom])
		if w.TwoPlayer {
			w.P1.OffsetGlider(w, player.ToLeft)
			w.P2.OffsetGlider(w, player.ToLeft)
			enterRect := w.sideEnterRect(false)
			w.P1.EnteredRect = enterRect
			w.P2.EnteredRect = enterRect
		} else {
			g.OffsetGlider(w, player.ToLeft)
			g.EnteredRect = w.sideEnterRect(false)
		}

	case player.ToLeft:
		w.SetMusicalMode(ProdGameScoreMode)
		if w.TwoPlayer {
			w.P1.UndoGliderLimbo(w)
			w.P2.UndoGliderLimbo(w)
			w.P1.InsureGliderFacingLeft(w)
			w.P2.InsureGliderFacingLeft(w)
		} else {
			g.InsureGliderFacingLeft(w)
		}
		w.ForceThisRoom(w.R.LocalNumbers[WestRoom])
		if w.TwoPlayer {
			w.P1.OffsetGlider(w, player.ToRight)
			w.P2.OffsetGlider(w, player.ToRight)
			enterRect := w.sideEnterRect(true)
			w.P1.EnteredRect = enterRect
			w.P2.EnteredRect = enterRect
		} else {
			g.OffsetGlider(w, player.ToRight)
			g.EnteredRect = w.sideEnterRect(true)
		}

	case player.Above:
		// Note the difference from the two side arms: ForceThisRoom comes *first*,
		// before any glider is touched, because the stairs branch below measures the
		// destination room's staircase.
		w.SetMusicalMode(KickGameScoreMode)
		w.ForceThisRoom(w.R.LocalNumbers[NorthRoom])
		if !w.R.TakingTheStairs {
			if w.TwoPlayer {
				w.P1.UndoGliderLimbo(w)
				w.P2.UndoGliderLimbo(w)
				w.P1.OffsetGlider(w, player.Below)
				w.P2.OffsetGlider(w, player.Below)
				w.P1.EnteredRect = w.P1.Dest
				w.P2.EnteredRect = w.P2.Dest
			} else {
				g.OffsetGlider(w, player.Below)
				g.EnteredRect = g.Dest
			}
		} else {
			// No UndoGliderLimbo here, in either branch. ReadyGliderForTripUpStairs
			// assigns the mode itself, so the limbo mode is overwritten rather than
			// undone -- and it is the only path out of limbo that does not go through
			// UndoGliderLimbo, which is why a glider arriving by stairs keeps
			// DontDraw if something had set it.
			if w.TwoPlayer {
				w.P1.ReadyGliderForTripUpStairs(w)
				w.P2.ReadyGliderForTripUpStairs(w)
			} else {
				g.ReadyGliderForTripUpStairs(w)
			}
		}

	case player.Below:
		w.SetMusicalMode(KickGameScoreMode)
		w.ForceThisRoom(w.R.LocalNumbers[SouthRoom])
		if !w.R.TakingTheStairs {
			if w.TwoPlayer {
				w.P1.UndoGliderLimbo(w)
				w.P2.UndoGliderLimbo(w)
				w.P1.OffsetGlider(w, player.Above)
				w.P2.OffsetGlider(w, player.Above)
				w.P1.EnteredRect = w.P1.Dest
				w.P2.EnteredRect = w.P2.Dest
			} else {
				g.OffsetGlider(w, player.Above)
				g.EnteredRect = g.Dest
			}
		} else {
			if w.TwoPlayer {
				w.P1.ReadyGliderForTripDownStairs(w)
				w.P2.ReadyGliderForTripDownStairs(w)
			} else {
				g.ReadyGliderForTripDownStairs(w)
			}
		}

	default:
		// An unrecognised direction changes no room and moves no glider -- and still
		// runs all five steps below, so the level is recomposed, the screen is wiped
		// on and a frame is rendered. Unreachable from the escape checks, which only
		// ever pass the four; reproduced because the fall-through is the C's.
	}

	// The arrival freeze, the same as ReadyGliderFromTransit's tail but written the
	// other way round -- there it tests the glider it was handed, here it selects one
	// from FirstPlayer. The two agree, and the extra `!onePlayerLeft` here is what
	// stops a corpse being frozen (it is already frozen).
	if w.TwoPlayer && !w.OneLeft {
		if w.FirstPlayer == player.Player1 {
			w.P2.TagGliderIdle(w)
		} else {
			w.P1.TagGliderIdle(w)
		}
	}

	w.ReadyLevel()
	w.RefreshScoreboard(NormalTitleMode)
	w.WipeScreenOn(where, w.R.V.JustRoomsRect)

	w.RenderFrame()
	w.restartRoomMovie()
}

// sideEnterRect is the four lines the two horizontal arms of MoveRoomToRoom each write
// out twice:
//
//	QSetRect(&enterRect, 0, 0, 48, 20);
//	QOffsetRect(&enterRect, 0, kGliderStartsDown + (short)thisRoom->leftStart - 2);
//
// It is the respawn rect for a glider that walked in through a side wall, and it is the
// room's, not the glider's: the author picks the entry height per side per room, in
// leftStart and rightStart. `fromLeftWall` selects which of the two, and with it the
// horizontal offset -- 0 for a glider entering at the left wall, kRoomWide - 48 at the
// right.
//
// 48 and 20 are written as literals in the C even though they equal kGliderWide and
// kGliderHigh, and are left as literals here.
//
// leftStart and rightStart are read *after* ForceThisRoom, so they are the destination
// room's. A destination that is out of range leaves ThisRoom nil and this returns the
// rect at offset 0; the C would read the previous room's field or, past the end of the
// array, whatever followed it.
func (w *World) sideEnterRect(fromLeftWall bool) player.Rect {
	var start, h int16
	if rm := w.ThisRoom(); rm != nil {
		if fromLeftWall {
			start = int16(rm.RightStart)
		} else {
			start = int16(rm.LeftStart)
		}
	}
	if fromLeftWall {
		h = RoomWide - 48
	}
	return player.Rect(render.Offset(render.SetRect(0, 0, 48, 20), h, player.GliderStartsDown+start-2))
}

// ---------------------------------------------------------------------------
// TransportRoomToRoom (Transit.c:314-348), MoveDuctToDuct (:352-387) and
// MoveMailToMail (:391-426)
// ---------------------------------------------------------------------------
//
// These three are **character-identical**. Not similar: the same thirty lines three
// times, with only the function name differing. All the behaviour that distinguishes a
// transporter from a duct from a mail slot is in ReadyGliderFromTransit's switch on
// linkedToWhat, which all three call.
//
// They are transcribed as three functions with three citations rather than factored
// into one, and that is a deliberate choice. Each has its own callers in the player
// package's Env, each may need to diverge (the mail slot's arrival sound, the duct's
// second glider), and collapsing them would leave three names pointing at one body,
// where a future edit meant for one silently changes all three. The comment on each is
// short because this block is the explanation.
//
// What they all do that MoveRoomToRoom does not is test `sameRoom`, and the skip is
// three separate statements rather than one block:
//
//	if (!sameRoom) ForceThisRoom(transRoom);
//	...
//	if (!sameRoom) ReadyLevel();
//	RefreshScoreboard(kNormalTitleMode);
//	if (!sameRoom) WipeScreenOn(kAbove, &justRoomsRect);
//
// So a transporter that leads to another object in the same room moves the glider and
// recomposes *nothing*. That is a real optimisation with real consequences, because
// ReadyLevel is what re-derives the room's state:
//
//   - takingTheStairs is not cleared (RoomGraphics.c:127 does not run);
//   - shadowVisible is not recomputed;
//   - the four openings are not recomputed, so a same-room hop cannot change them;
//   - DrawLocale's tables -- flames, pendulums, bands, the mirror region, the temporary
//     manholes -- are not rebuilt, which is why a band in flight survives a same-room
//     transport and does not survive any other room change;
//   - InitGarbageRects does not run, so neither dirty-rect list is cleared and
//     nextFrame is not reseeded. The frame the transport happens on therefore renders
//     twice against one budget -- once here, once from the play loop -- and every
//     effect that steps its own animation inside RenderFrame steps twice. In a room
//     with a pendulum, clockFrame double-steps.
//
// That last one is the only place in the game where a double RenderFrame lands on
// un-cleared lists, and it is what the fidelity replay's transition test is built to
// catch. All four RenderFrame calls in this file are outside the sameRoom test and
// outside the movie block, so every one of them fires; see restartRoomMovie.

// TransportRoomToRoom is Transit.c:314-348: the glider dissolves into a transporter and
// re-materialises at its far end.
func (w *World) TransportRoomToRoom(g *player.Glider) {
	w.SetMusicalMode(KickGameScoreMode)
	w.HandleRoomVisitation()

	sameRoom := w.Pending.Room == w.R.RoomNumber
	if !sameRoom {
		w.ForceThisRoom(w.Pending.Room)
	}

	if w.TwoPlayer {
		w.P1.UndoGliderLimbo(w)
		w.P2.UndoGliderLimbo(w)
		w.ReadyGliderFromTransit(&w.P1, w.Pending.What)
		w.ReadyGliderFromTransit(&w.P2, w.Pending.What)
	} else {
		w.ReadyGliderFromTransit(g, w.Pending.What)
	}

	if !sameRoom {
		w.ReadyLevel()
	}
	w.RefreshScoreboard(NormalTitleMode)
	if !sameRoom {
		// kAbove, always: a link transport has no direction, so the wipe always runs
		// downward whatever the geometry of the two objects.
		w.WipeScreenOn(player.Above, w.R.V.JustRoomsRect)
	}

	w.RenderFrame()
	w.restartRoomMovie()
}

// MoveDuctToDuct is Transit.c:352-387. See the block above: identical to
// TransportRoomToRoom.
func (w *World) MoveDuctToDuct(g *player.Glider) {
	w.SetMusicalMode(KickGameScoreMode)
	w.HandleRoomVisitation()

	sameRoom := w.Pending.Room == w.R.RoomNumber
	if !sameRoom {
		w.ForceThisRoom(w.Pending.Room)
	}

	if w.TwoPlayer {
		w.P1.UndoGliderLimbo(w)
		w.P2.UndoGliderLimbo(w)
		w.ReadyGliderFromTransit(&w.P1, w.Pending.What)
		w.ReadyGliderFromTransit(&w.P2, w.Pending.What)
	} else {
		w.ReadyGliderFromTransit(g, w.Pending.What)
	}

	if !sameRoom {
		w.ReadyLevel()
	}
	w.RefreshScoreboard(NormalTitleMode)
	if !sameRoom {
		w.WipeScreenOn(player.Above, w.R.V.JustRoomsRect)
	}

	w.RenderFrame()
	w.restartRoomMovie()
}

// MoveMailToMail is Transit.c:391-426. See the block above: identical to
// TransportRoomToRoom.
func (w *World) MoveMailToMail(g *player.Glider) {
	w.SetMusicalMode(KickGameScoreMode)
	w.HandleRoomVisitation()

	sameRoom := w.Pending.Room == w.R.RoomNumber
	if !sameRoom {
		w.ForceThisRoom(w.Pending.Room)
	}

	if w.TwoPlayer {
		w.P1.UndoGliderLimbo(w)
		w.P2.UndoGliderLimbo(w)
		w.ReadyGliderFromTransit(&w.P1, w.Pending.What)
		w.ReadyGliderFromTransit(&w.P2, w.Pending.What)
	} else {
		w.ReadyGliderFromTransit(g, w.Pending.What)
	}

	if !sameRoom {
		w.ReadyLevel()
	}
	w.RefreshScoreboard(NormalTitleMode)
	if !sameRoom {
		w.WipeScreenOn(player.Above, w.R.V.JustRoomsRect)
	}

	w.RenderFrame()
	w.restartRoomMovie()
}

// restartRoomMovie is the second half of the COMPILEQT block that ends all four
// transit handlers:
//
//	if ((thisMac.hasQT) && (hasMovie) && (tvInRoom) && (tvOn))
//	{
//	    GoToBeginningOfMovie(theMovie);
//	    StartMovie(theMovie);
//	}
//
// A television is the only object in the game that plays video, and this is what makes
// walking back into a room restart it from the beginning rather than resume it. All
// four conditions are read: the build has QuickTime, the house shipped a movie, the new
// room contains a TV, and the TV is switched on -- and that last flag is World.TVOn,
// which is *not* reset on a room change, so a TV switched on in one room restarts the
// movie in every later room that has one. See that field.
//
// A named no-op rather than a comment at four call sites, so that 1.5e has one body to
// fill and cannot fill it in three places out of four. The RenderFrame above it is
// inside the same #ifdef but outside this test, which is why it is not in here.
func (w *World) restartRoomMovie() {}

// ---------------------------------------------------------------------------
// ForceKillGlider (Transit.c:449-469)
// ---------------------------------------------------------------------------

// ForceKillGlider is the give-up key: kill the glider that is *not* waiting, so that
// the one that is can go on.
//
// It reads as though it kills the player who pressed the key and it does the opposite.
// Both arms test one glider for limbo and then fade out the *other*, because the point
// is not to quit -- it is to break state 4 or 5 of the limbo machine by spending the
// free player's mortal. OffAMortal then sees playerSuicide and calls FollowTheLeader,
// which performs the room change the waiting glider has been holding.
//
// Three details:
//
// **`mode != kGliderFadingOut` is a debounce**, not a safety check. The key is polled
// every frame while it is held (Input.c:366-371), so without it a single press would
// queue a fade-out per frame and spend several mortals.
//
// **Only player 1 has the key.** Input.c:366-371 reads one key map slot and calls this
// unconditionally, so in a game where player 2 is the one waiting, player 1 is still the
// only one who can release them. IMPROVEMENTS.md 2.23 has the second binding.
//
// **In the deadlock both gliders are in limbo**, so the first test wins and glider 2 is
// the one killed. Nothing here is aware of that case; it works by accident, and it is
// the only way out of state 4.
//
// If neither glider is in limbo this does nothing at all -- which is why the give-up key
// is inert in a one-player game and why player.Input already gates the call on
// TwoPlayerGame (player/input.go:165).
func (w *World) ForceKillGlider() {
	if w.P1.Mode == player.GliderInLimbo {
		if w.P2.Mode != player.GliderFadingOut {
			w.P2.StartGliderFadingOut(w)
			w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
			w.Suicide = true
		}
	} else if w.P2.Mode == player.GliderInLimbo {
		if w.P1.Mode != player.GliderFadingOut {
			w.P1.StartGliderFadingOut(w)
			w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
			w.Suicide = true
		}
	}
}

// ---------------------------------------------------------------------------
// FollowTheLeader (Transit.c:473-557)
// ---------------------------------------------------------------------------

// FollowTheLeader performs the room change a glider has been waiting in limbo for,
// dragging the other glider along to the same place.
//
// It is called from exactly one place -- OffAMortal, when playerSuicide is set -- and it
// is the payoff of the give-up key. It reads the pending escape code, clears it, copies
// the limbo glider's position onto the other one, and dispatches to whichever of the
// four transit handlers matches.
//
// **This is the function that genuinely needs world scope**, and it is the reason
// World.Pending exists alongside the per-glider Transit. It writes one glider's rects
// from the other's and then hands *the other one* to a handler that reads the shared
// link -- so the two gliders' state is deliberately entangled here, and no per-glider
// arrangement can express it.
//
// Two things about it are wrong in the original and are kept:
//
// **`oneOrTwo` is uninitialised.** If neither glider is in limbo, neither of the two
// blocks below runs and the switch dispatches on a stack Boolean. The port uses false,
// which selects theGlider -- one of the two values the C could have had, and the one an
// empty 68k stack frame most often produced. The path is reachable: ForceKillGlider is
// the only writer of playerSuicide and only fires with a glider in limbo, but
// OffAMortal is entered from a fade-out that a *previous* frame's ForceKillGlider armed,
// and UndoGliderLimbo can have run in between.
//
// **The position copy is largely vestigial.** Every handler this dispatches to either
// calls OffsetGlider (which rewrites Dest from the glider's own previous Dest) or
// ReadyGliderFromTransit (which rewrites it from the destination rect), so the copied
// rects are overwritten before they are drawn. The exception is the stairs branch of
// MoveRoomToRoom, where ReadyGliderForTripUpStairs positions from Dest -- so the copy
// matters only when the leader escaped up or down a staircase. It is transcribed in
// full because that case is real.
//
// The three stairs codes are grouped with the plain vertical escapes, three per case:
// Escaped, Escaping and EscapedStairs all mean "went up", because a glider can be
// interrupted mid-staircase and the pending code has to survive that.
func (w *World) FollowTheLeader() {
	w.Suicide = false
	wasEscaped := w.Escaped
	w.Escaped = NoOneEscaped

	// oneOrTwo: true means glider 1 is the one in limbo, so glider 2 is the one handed
	// to the handler. The C leaves this uninitialised; see the note above.
	oneOrTwo := false

	if w.P1.Mode == player.GliderInLimbo {
		oneOrTwo = true
		w.P2.Dest = w.P1.Dest
		w.P2.DestShadow = w.P1.DestShadow
		w.P2.Whole = w.P2.Dest
		w.P2.WholeShadow = w.P2.DestShadow
	} else if w.P2.Mode == player.GliderInLimbo {
		oneOrTwo = false
		w.P1.Dest = w.P2.Dest
		w.P1.DestShadow = w.P2.DestShadow
		w.P1.Whole = w.P1.Dest
		w.P1.WholeShadow = w.P1.DestShadow
	}

	// follower is the glider passed to the handler: the one that is *not* in limbo.
	// The handlers use it only in the one-player branches, which cannot be taken from
	// here -- FollowTheLeader is two-player-only -- so this selection is what the C
	// spends fourteen if/else pairs on and nothing reads.
	follower := &w.P1
	if oneOrTwo {
		follower = &w.P2
	}

	switch wasEscaped {
	case player.PlayerEscapedUp, player.PlayerEscapingUpStairs, player.PlayerEscapedUpStairs:
		w.MoveRoomToRoom(follower, player.Above)

	case player.PlayerEscapedDown, player.PlayerEscapingDownStairs, player.PlayerEscapedDownStairs:
		w.MoveRoomToRoom(follower, player.Below)

	case player.PlayerEscapedLeft:
		w.MoveRoomToRoom(follower, player.ToLeft)

	case player.PlayerEscapedRight:
		w.MoveRoomToRoom(follower, player.ToRight)

	case player.PlayerTransportedOut:
		w.TransportRoomToRoom(follower)

	case player.PlayerMailedOut:
		w.MoveMailToMail(follower)

	case player.PlayerDuckedOut:
		w.MoveDuctToDuct(follower)

	default:
		// Including PlayerIsDeadForever (-69), which is what makes state 6 of the limbo
		// machine terminal: once OffAMortal has written it, no give-up can move
		// anybody.
	}
}
