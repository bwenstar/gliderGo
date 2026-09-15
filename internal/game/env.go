package game

// This file is *World's implementation of player.Env: the 46 calls the glider makes
// outward at the moment it needs something that is not its own state.
//
// In the original there is no interface here at all. The player code calls free
// functions over file-scope globals, and Player.c, Modes.c and Input.c are in the same
// translation-unit soup as Play.c and Render.c. internal/game/player/env.go's own
// header explains why the port cuts there instead; this file is the other side of that
// cut, and its rule is that a method does exactly what the C function or global access
// it names does, and nothing else. Where a method cannot yet do that -- because the
// subsystem behind it belongs to a later sub-stage -- it says so in its own comment and
// names the sub-stage, rather than quietly approximating.
//
// The compile-time assertion at the bottom is the point of the file: it is what makes
// "*World implements player.Env in full" a fact the compiler checks rather than a claim
// in a commit message.

import (
	"glidergo/internal/game/player"
	"glidergo/internal/render"
)

// ---------------------------------------------------------------------------
// Sound
// ---------------------------------------------------------------------------

// PlayPrioritySound is Sound.c:40-85, reached through the World.SoundPlayer hook.
//
// A nil hook is not a degraded mode to apologise for: it is exactly the original's
// `dontLoadSounds` short circuit, which PlayPrioritySound tests first thing and returns
// on (Sound.c:44). A silent build takes the same path a Mac with no sound memory took.
func (w *World) PlayPrioritySound(sound, priority int16) {
	if w.SoundPlayer == nil {
		return
	}
	w.SoundPlayer(sound, priority)
}

// ---------------------------------------------------------------------------
// Inventory
// ---------------------------------------------------------------------------
//
// Six accessors and a flag over four World fields. They are methods rather than
// exported fields because the C reads and writes the globals directly and the player
// package cannot see World; see env.go's note on the ~20 accessor methods.

func (w *World) BatteryTotal() int16     { return w.Battery }
func (w *World) SetBatteryTotal(n int16) { w.Battery = n }
func (w *World) BandsTotal() int16       { return w.Bands }
func (w *World) SetBandsTotal(n int16)   { w.Bands = n }
func (w *World) FoilTotal() int16        { return w.Foil }
func (w *World) SetFoilTotal(n int16)    { w.Foil = n }

// SetShowFoil is the `showFoil` write plus the sheet reload around it:
// DeckGliderInFoil (Player.c:1142-1152) when on, RemoveFoilFromGlider
// (Player.c:1202-1213) when off.
//
// It is a graphics-state change and not just a flag, because the original swaps the
// entire glider sprite sheet rather than tinting the sprite. But *only in a two-player
// game*: both of the C's reload blocks are inside `if (twoPlayerGame)`. In a one-player
// game nothing is reloaded at all, because the foil sheet is permanently resident in
// the second slot and RenderGlider selects it from the flag instead. LoadGliderSheets
// has the full table and the reason it is arranged that way.
//
// So the reload is conditional and the flag is not. That asymmetry is the whole
// content of this function, and it is also why the flag has to be stored even where
// there is nothing to reload: the count and the artwork legitimately disagree for one
// frame while the dissolve runs, and the player code depends on that.
func (w *World) SetShowFoil(on bool) {
	w.ShowFoil = on
	if !w.TwoPlayer {
		return
	}
	if on {
		w.GlidSrc = w.R.A.Strip("gliderFoil")
		w.Glid2Src = w.R.A.Strip("gliderFoil2")
		return
	}
	w.GlidSrc = w.R.A.Strip("glider")
	w.Glid2Src = w.R.A.Strip("glider2")
}

// ---------------------------------------------------------------------------
// Room geometry
// ---------------------------------------------------------------------------

// SetShadowVisible writes the `shadowVisible` global (Player.c:53).
//
// The *getter* of the same name is not here: it is IsShadowVisible in room.go, and it
// is the Room.c:1103 predicate, not this field. That asymmetry is deliberate and
// player/env.go explains it at length -- the one call site is
// `shadowVisible = IsShadowVisible()`, so the pair has to be predicate-in,
// global-out or the assignment degenerates into a no-op.
func (w *World) SetShadowVisible(v bool) { w.R.ShadowVisible = v }

func (w *World) HasMirror() bool   { return w.R.HasMirror }
func (w *World) TopOpen() bool     { return w.R.TopOpen }
func (w *World) BottomOpen() bool  { return w.R.BottomOpen }
func (w *World) LeftThresh() int16 { return w.R.LeftThresh }

func (w *World) RightThresh() int16 { return w.R.RightThresh }

// Background is `thisBackground` and Tile is `thisTiles[i]` -- the *cached* copies
// DrawRoomBackground writes for the central room (RoomGraphics.c:162-253), not the
// house record's fields.
//
// The distinction matters in the same way the shadow pair's does. DetermineRoomOpenings
// reads the house record because it runs as part of composing the room; the escape
// checks read these caches because they run per frame, and on a frame where the room
// has changed identity but has not yet been recomposed the two differ. Reading the
// house record here would make the glider collide against the room it is arriving in
// one frame before that room is drawn.
func (w *World) Background() int16 { return w.R.ThisBackground }

// Tile does not range-check i, because the original does not: callers test the index
// themselves (`(offset >= 0) && (offset <= 7)`, Interactions.c:455) and Go's own bounds
// check is a strictly better failure than the C's silent out-of-array read. Keeping the
// check at the call sites is what keeps those transcriptions statement-for-statement.
func (w *World) Tile(i int16) int16 { return w.R.ThisTiles[i] }

// GetUpStairsRightEdge is ObjectRects.c:1135-1159 and GetDownStairsLeftEdge is
// :1163-1185: the x a glider is clipped against while it walks behind a staircase.
//
// **Read the object codes twice.** GetUpStairs...  searches for kDownStairs and
// GetDownStairs... searches for kUpStairs. That is the C, verbatim, and it is not a
// typo in either language: the two staircase objects are the two ends of one flight, so
// a glider on its way *up* out of this room is behind the banister of the object that
// leads *down* into it. A port that "corrected" the pairing would clip both staircases
// against the wrong edge, and in a room with only one of the two would clip against the
// default instead -- which is kRoomWide and 0, i.e. no clipping at all.
//
// Both scan all 24 slots and break on the first match, so a room with two down
// staircases uses the lower-numbered slot. Both read the house record rather than
// Master, because the C reads `(*thisHouse)->rooms[thisRoomNumber].objects[i]` directly
// -- it is looking for an object in *this* room only, and Master holds all nine.
func (w *World) GetUpStairsRightEdge() int16 {
	rightEdge := RoomWide
	rm := w.ThisRoom()
	if rm == nil {
		return rightEdge
	}
	for i := 0; i < MaxRoomObs; i++ {
		if rm.Objects[i].What == DownStairs {
			rightEdge = rm.Objects[i].TopLeft().H + render.SrcRect(DownStairs).Right - 1
			break
		}
	}
	return rightEdge
}

func (w *World) GetDownStairsLeftEdge() int16 {
	leftEdge := int16(0)
	rm := w.ThisRoom()
	if rm == nil {
		return leftEdge
	}
	for i := 0; i < MaxRoomObs; i++ {
		if rm.Objects[i].What == UpStairs {
			leftEdge = rm.Objects[i].TopLeft().H + 1
			break
		}
	}
	return leftEdge
}

// SetTakingTheStairs writes `takingTheStairs` (RoomGraphics.c:38), which suppresses the
// glider's own draw for the frame a staircase transition completes.
func (w *World) SetTakingTheStairs(v bool) { w.R.TakingTheStairs = v }

// PlayOriginH and PlayOriginV are `playOriginH` and `playOriginV`
// (InterfaceInit.c:213-214): the screen position of room-local (0,0), which at 640x480
// is (64,79).
//
// They are on the interface because the player package works in room-local coordinates
// and hands rects to AddRectToWorkRects and CopyRectWorkToMain in *screen* coordinates,
// so it has to do the addition itself -- exactly as the C does, at Modes.c:88-96 and
// Player.c:60-64. See player.Env's contract note for why the offset is not folded into
// the adders.
func (w *World) PlayOriginH() int16 { return w.R.V.OriginH }
func (w *World) PlayOriginV() int16 { return w.R.V.OriginV }

// ---------------------------------------------------------------------------
// Two-player state
// ---------------------------------------------------------------------------

func (w *World) TwoPlayerGame() bool { return w.TwoPlayer }
func (w *World) OnePlayerLeft() bool { return w.OneLeft }

// PlayerDead is `playerDead` (Player.c:53), and it does not mean "a player is dead".
// It names *which* player is dead, in the same true=Player1 encoding as Glider.Which,
// and is only meaningful when OnePlayerLeft is set. Every read of it selects the other
// glider; see Survivor.
func (w *World) PlayerDead() bool { return w.DeadWhich }

func (w *World) OtherPlayerEscaped() int16     { return w.Escaped }
func (w *World) SetOtherPlayerEscaped(v int16) { w.Escaped = v }
func (w *World) SetFirstPlayer(which bool)     { w.FirstPlayer = which }
func (w *World) SaidFollow() int16             { return w.SaidFollowCount }
func (w *World) SetSaidFollow(n int16)         { w.SaidFollowCount = n }

// Survivor returns the glider of whichever player is not dead.
//
// It is the inverse of PlayerDead, and the inversion is the whole point: Player.c:349-352
// reads `if (playerDead == kPlayer1) MoveRoomToRoom(&theGlider2, ...) else
// MoveRoomToRoom(&theGlider, ...)` -- it moves the one playerDead does *not* name. Every
// one-player-left branch in every transit handler does this, so that a dead player's
// corpse does not drag the survivor's view around.
func (w *World) Survivor() *player.Glider {
	if w.DeadWhich == player.Player1 {
		return &w.P2
	}
	return &w.P1
}

// ---------------------------------------------------------------------------
// Not yet implemented: the nine that belong to later work
// ---------------------------------------------------------------------------
//
// Two of the 48, both with empty bodies. These are honest stubs, not approximations. Each
// says which commit or sub-stage fills it and what the stub's behaviour means in the
// meantime, because a stub that silently does something plausible is worse than one that
// does nothing: the first hides a gap and the second is visible in a test.
//
// The one this commit removed is AddAShreddedGlider, now real in shreds.go. Its empty body
// was the least visible of any stub in the file: a glider still flew into the shredder, was
// still clipped away four pixels a frame, and still died -- it simply left no confetti. The
// one before that was AddBand, whose `return false` was load-bearing rather than merely safe
// (the C's false is what makes a refused shot *not cost a band*). The four before that were
// Scoreboard.c's -- QuickBatteryRefresh, QuickBandsRefresh, QuickFoilRefresh and
// RefreshScoreboard -- and the nine before them were the dirty-rect protocol, the four
// transit handlers, OffAMortal, FlagStillOvers and ForceKillGlider.

// DoPause and DoCommandKey belong to 1.7, the shell: both open modal UI that does not
// exist yet. DoPause in particular *blocks* in the original, called from inside GetInput,
// which is why a paused game does not advance a frame -- see docs/IMPROVEMENTS.md 2.5 for
// why a released build needs more than a faithful transcription of it, and 2.32 for the two
// other places (BringUpBanner, DisplayStarsRemaining) that stop the world the same way and
// want the same answer: a pause the frame loop knows about, not a sleep.
func (w *World) DoPause()      {}
func (w *World) DoCommandKey() {}

// _ asserts the whole interface at compile time. This is the acceptance criterion for
// 1.5b's Env work in executable form: if a method is missing or its signature drifts,
// the package does not build.
var _ player.Env = (*World)(nil)
