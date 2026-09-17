package game

// Death: OffAMortal (Player.c:1482-1604), HideGlider (Play.c:712-729) and
// FlagGameOver (GameOver.c:236-241).
//
// One function does almost all of it. OffAMortal is called from exactly two places --
// Player.c:295, a glider that finished dissolving, and Player.c:1317, a glider that
// finished burning or being shredded -- and from those two calls it has to decide
// between four outcomes: respawn here, release a player waiting in limbo, retire one
// player of two, or end the game. Which one depends on `mortals` *after* the
// decrement, on twoPlayerGame, on playerSuicide and on onePlayerLeft, and no two of
// those are checked in the same place.
//
// The shape below is the C's, including the fact that its two `if (mortals ...)` blocks
// are sequential rather than nested, so the second one re-tests a variable the first
// one has already branched on. Flattening them changes nothing; keeping them apart
// makes the -1 case -- where the first block retires a player and the second moves the
// survivor -- readable as the two separate events it is.

import "github.com/bwenstar/gliderGo/internal/game/player"

// OffAMortal is Player.c:1482-1604: a glider has finished dying. Spend a life.
//
// Reading order matters here, because `mortals` is decremented in the middle and both
// halves test it:
//
//	mortals >= 0 after the decrement   there is another glider to fly. Respawn it at
//	                                   enteredRect -- or, if the death was a give-up,
//	                                   hand off to FollowTheLeader instead.
//	mortals == -1, two players         the first of the two is out. It goes into limbo
//	                                   with DontDraw, and the survivor is moved through
//	                                   whatever room change was pending.
//	mortals < -1, two players          both are out. Game over.
//	mortals < 0, one player            game over.
//
// **The mortals counter is shared between the two players**, which is why "-1" means
// "the first player of two is finished" rather than "player 1". A two-player game has
// one pool of lives and each death takes from it, so the two players do not get an
// equal number of tries -- the better player gets more. That is the original's design
// and not a bug; the scoreboard shows the shared pool.
//
// The early return on GameOver is what stops a second glider dying during the 16-frame
// countdown from decrementing past the end and re-flagging.
func (w *World) OffAMortal(g *player.Glider) {
	if w.GameOver {
		return
	}

	// The shred particles are dropped *before* anything else, because a glider that
	// died in a shredder leaves its own pieces on screen and the respawn below would
	// otherwise draw the new glider into them.
	//
	// "Dropped", singular, and not "cleared": RemoveShreds takes out the single
	// most-advanced cloud and can take out none at all. That is the original's and is
	// documented in shreds.go rather than worked around here -- the guard reads as
	// though the two lines together empty the table, and they do not.
	if w.NumShredded > 0 {
		w.RemoveShreds()
	}

	w.Mortals--
	if w.Mortals < 0 {
		w.HideGlider(g)
		if w.TwoPlayer {
			if w.Mortals < -1 {
				// Both players are now dead.
				w.FlagGameOver()
				g.DontDraw = true
			} else {
				// This is state 6 of the limbo machine -- see transit.go. Note that
				// this is the one FlagGliderInLimbo call that is followed by
				// DontDraw: a glider frozen at a doorway is still drawn, but one that
				// is out of lives is not.
				g.FlagGliderInLimbo(w, false)
				g.DontDraw = true
				w.OneLeft = true
				w.DeadWhich = g.Which
			}
		} else {
			w.FlagGameOver()
			g.DontDraw = true
		}
	} else {
		w.QuickGlidersRefresh()
		w.HideGlider(g)
	}

	if w.Mortals >= 0 {
		// A glider that died *while gaining foil* has a foil the sheets do not know
		// about yet, so the deck is settled before the mode is reset. Skipping this
		// leaves the player with the foil in their inventory and no foil in the
		// sprite -- and, in a two-player game, with the wrong sheet loaded for the
		// rest of the room.
		if g.Mode == player.GliderGoingFoil {
			g.DeckGliderInFoil(w)
		}

		g.FlagGliderNormal(w)
		if w.Suicide {
			// The give-up path. No fade-in and no reposition: FollowTheLeader is about
			// to change the room, and MoveRoomToRoom will place this glider itself.
			w.FollowTheLeader()
		} else {
			g.StartGliderFadingIn(w)
			g.Dest = g.EnteredRect
			g.Whole = g.Dest
			g.DestShadow.Left = g.Dest.Left
			g.DestShadow.Right = g.Dest.Right
			g.WholeShadow = g.DestShadow
		}
	} else if w.Mortals == -1 && w.OneLeft && !w.GameOver {
		// The survivor is dragged through the room change the player who just died had
		// been waiting on. This is the same seven-arm dispatch as FollowTheLeader and
		// the same three-codes-per-direction grouping, but it selects the glider from
		// PlayerDead rather than from who is in limbo -- so the two cannot be shared,
		// even though they read alike. See transit.go's limbo table, state 6.
		survivor := &w.P1
		if w.DeadWhich == player.Player1 {
			survivor = &w.P2
		}

		switch w.Escaped {
		case player.PlayerEscapedUp, player.PlayerEscapingUpStairs, player.PlayerEscapedUpStairs:
			w.MoveRoomToRoom(survivor, player.Above)

		case player.PlayerEscapedDown, player.PlayerEscapingDownStairs, player.PlayerEscapedDownStairs:
			w.MoveRoomToRoom(survivor, player.Below)

		case player.PlayerEscapedLeft:
			w.MoveRoomToRoom(survivor, player.ToLeft)

		case player.PlayerEscapedRight:
			w.MoveRoomToRoom(survivor, player.ToRight)

		case player.PlayerTransportedOut:
			w.TransportRoomToRoom(survivor)

		case player.PlayerMailedOut:
			w.MoveMailToMail(survivor)

		case player.PlayerDuckedOut:
			w.MoveDuctToDuct(survivor)

		default:
			// Including NoOneEscaped, which is the common case: the player who died was
			// not waiting for anything, so the survivor stays in the room and simply
			// carries on alone.
		}

		// The terminal write. PlayerIsDeadForever (-69) matches no arm of this switch
		// nor of FollowTheLeader's, so once it is here no later death or give-up can
		// move anyone through a room. It is not cleared until the next NewGame.
		w.Escaped = player.PlayerIsDeadForever
	}
}

// HideGlider is Play.c:712-729: erase a glider from the screen, now.
//
// It is the one thing in the game that draws by *not* drawing: it copies the clean work
// map over the three rects the glider occupies -- body, mirror reflection, shadow -- so
// what is left is the room without it. It is called at three points, all of them a
// glider that is about to stop existing: twice in OffAMortal and once at the end of the
// game-over countdown (Play.c:509).
//
// The mirror rect is the same body rect offset by (-20, -16), which is DrawReflection's
// offset. Note that it is the *already offset* tempRect being offset again, so the two
// copies overlap by all but 20x16 pixels of their area -- the original does not
// bother computing the union.
//
// **The port has to present and the original does not.** In the C, CopyRectWorkToMain
// is a CopyBits straight to the window, so the erase is on the glass the instant it
// runs. Here Main is an offscreen surface and the last of the three calls has to be
// followed by present(), because HideGlider's third caller -- the countdown tail -- is
// the last thing to touch the play screen before the game-over sequence takes over, and
// there is no RenderFrame after it to flush the change.
func (w *World) HideGlider(g *player.Glider) {
	tempRect := g.Whole.Offset(w.PlayOriginH(), w.PlayOriginV())
	w.CopyRectWorkToMain(tempRect)

	if w.R.HasMirror {
		tempRect = tempRect.Offset(-20, -16)
		w.CopyRectWorkToMain(tempRect)
	}

	tempRect = g.WholeShadow.Offset(w.PlayOriginH(), w.PlayOriginV())
	w.CopyRectWorkToMain(tempRect)

	w.present()
}

// FlagGameOver is GameOver.c:236-241: the game is over in sixteen frames.
//
// The delay is the whole point, and the comment above it in the original says so: the
// player has just watched their last glider come apart, and cutting to the game-over
// screen on the same frame would swallow that. So this sets a latch and a counter and
// returns, the play loop keeps rendering, and the tail at Play.c:499-503 counts down.
//
// GameOver is more than a countdown flag. It gates HandleGlider (Play.c:487), so no
// glider moves during the sixteen frames, and it is the early return at the head of
// OffAMortal, so nothing can die twice. Setting the two fields without also honouring
// those two gates would let a burning glider spend the whole remaining pool of mortals
// during the delay.
//
// The music switch is to PlayWholeScoreMode -- the full score from the top, not the
// in-game loop -- and it is the audible signal that the game has ended. It does nothing
// while DontLoadMusic is set; see SetMusicalMode.
func (w *World) FlagGameOver() {
	w.GameOver = true
	w.CountDown = NumCountDownFrames
	w.SetMusicalMode(PlayWholeScoreMode)
}

// NumCountDownFrames is kNumCountDownFrames, #defined locally at GameOver.c:17 and used
// at exactly one other line, :240. Sixteen frames is a little over half a second at the
// original's 30.07 fps.
const NumCountDownFrames int16 = 16

// RemoveShreds landed in 1.5f and is in shreds.go with the rest of the confetti. Read its
// comment before trusting the call in OffAMortal above: despite the plural it removes one
// cloud, and if the only cloud on screen is still growing it removes none.
