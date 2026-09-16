package game

// The game: NewGame (Play.c:74-278) and PlayGame (:430-597), plus the four small
// functions they need -- InitGlider (:307-368), SetHouseToFirstRoom (:372-378),
// SetHouseToSavedRoom (:382-385), HandlePlayEvent (:392-426) -- and
// RestoreEntireGameScreen (:800-819). The three helpers Play.c reaches into other files
// for are here too, because nothing else calls them: GetFirstRoomNumber and
// WhereDoesGliderBegin (House.c:196-239) and CountStarsInHouse (Banner.c:89-110).
//
// This is the top of the call graph. Everything else in the package is reached from the
// twenty-odd lines of PlayGame's loop body, which is the whole game on one screen:
//
//	Frame++, EvenFrame = !EvenFrame        the two clocks
//	the event pump                         and the pause-on-deactivate spin
//	HandleTelephone                        ambience
//	HandleDynamics                         the room moves
//	GetInput, HandleInteraction            the player acts, the world answers
//	HandleTriggers, HandleBands            fuses and rubber bands
//	HandleGlider                           the player's state machine steps
//	RenderFrame, HandleDynamicScoreboard   draw it -- and wait for the frame's tick
//
// followed by a game-over countdown that runs *after* all of that, so the last glider's
// death finishes on screen before the game-over sequence takes the window.
//
// Read PlayGame before NewGame. NewGame is a 200-line preamble whose only interesting
// property is its order, and the order only makes sense once you know what the loop
// needs to be true on its first pass.

import (
	"glidergo/internal/game/player"
	"glidergo/internal/render"
)

// The three values of TheMode (GliderDefines.h:186-188). Not a named type, because the
// C stores it in the same `short` as everything else and 1.7's menu code compares it
// against these bare numbers.
const (
	SplashMode int16 = 0
	EditMode   int16 = 1
	PlayMode   int16 = 2
)

// The two values of NewGame's `mode` argument (GliderDefines.h:616-617).
//
// They are also InitGlider's and WhereDoesGliderBegin's argument, and there they matter
// more: **neither function has a default**. WhereDoesGliderBegin leaves `initialPt`
// uninitialised for any third value and then offsets a rect by whatever was on the
// stack, so calling either with anything but these two is undefined in the original.
// The port's versions say so at the call site rather than silently picking one.
const (
	ResumeGameMode int16 = 0
	NewGameMode    int16 = 1
)

// InitialGliders is kInitialGliders, #defined locally at Play.c:18: the lives a new
// game starts with, per player.
//
// Two, and the counter it seeds is shared -- a two-player game gets four between them,
// not four each. See InitGlider and World.Mortals.
const InitialGliders int16 = 2

// IdleSplashTicks is kIdleSplashTicks: two minutes of TickCount before the splash
// screen advances to the next attract-mode panel. NewGame's last statement and
// DoDemoGame's both set the deadline from it.
const IdleSplashTicks int64 = 7200

// ---------------------------------------------------------------------------
// NewGame (Play.c:74-278)
// ---------------------------------------------------------------------------

// NewGame sets a game up, plays it to its end, and puts the shell back.
//
// It is one function containing three, and the middle one is the whole game: everything
// before `w.Playing = true` is set-up, PlayGame does not return until the game is over,
// and everything after it is teardown. The original's own comment marks the seam
// (Play.c:212-213) and it is the only comment in the function.
//
// **This does not call ReadyLevel, and must not be refactored to.** The two do the same
// four things in a different order with fifty lines of other work interleaved, and each
// difference is load-bearing:
//
//		ReadyLevel:  NilSavedMaps, DetermineRoomOpenings, DrawLocale, InitGarbageRects
//		NewGame:     DetermineRoomOpenings, NilSavedMaps, ... DrawLocale, ... InitGarbageRects
//
//	  - The first two are *swapped*. Here the openings are derived before the saved-map
//	    table is dropped, which is safe only because SetHouseToFirstRoom has already
//	    changed the room and nothing has composed anything yet.
//	  - DrawLocale is fifty-three lines later, after the gliders are initialised and the
//	    screen is painted black, because it draws into a work map that has to be black
//	    first (a room narrower than the screen leaves the margins showing).
//	  - InitGarbageRects is after BringUpBanner, which *blocks* on a modal alert. Seeding
//	    the frame clock before a blocking dialogue would leave the deadline minutes in
//	    the past, and the first frame of the game would be un-paced.
//	  - StartGliderFadingIn is after InitGarbageRects, so the fade's first dirty rect
//	    lands in a list that has just been cleared rather than one that is about to be.
//
// Collapsing any of it into ReadyLevel would compile, run, and be wrong in four ways at
// once, none of which shows up as a crash. See readylevel.go for the other half of this
// argument.
func (w *World) NewGame(mode int16) {
	// The scoreboard's position depends on how many neighbouring rooms are drawn, and
	// this is the only call. See scoreboard.go -- in the shipped configuration it does
	// nothing at all, which is why the original shows no scoreboard.
	w.AdjustScoreboardHeight()

	w.GameOver = false
	w.TheMode = PlayMode
	w.StartGameMusic()

	// A resumed game keeps the object states it was saved with -- switches thrown,
	// prizes taken, grease spilt. Note that this runs *before* the room is chosen, so
	// it is a whole-house reset and not a room one.
	if mode != ResumeGameMode {
		w.SetObjectsToDefaults()
	}

	// HideCursor(). The port has no cursor to hide until 1.7 gives the shell one.

	if mode == ResumeGameMode {
		w.SetHouseToSavedRoom()
	} else if mode == NewGameMode {
		w.SetHouseToFirstRoom()
	}

	// Swapped relative to ReadyLevel; see the note above.
	w.DetermineRoomOpenings()
	w.R.SavedMaps = w.R.SavedMaps[:0] // NilSavedMaps()

	w.Frame = 0
	// numBands = 0 -- the band table is 1.5e's; HandleBands is a no-op below.
	// `demoIndex = 0`: the attract-mode cursor, reset here whether or not this is a demo,
	// exactly as the C does. A world with no stream loaded has nothing to reset.
	if w.Demo != nil {
		w.Demo.Reset()
	}
	w.SaidFollowCount = 0
	w.Escaped = NoOneEscaped
	w.OneLeft = false
	w.Suicide = false

	// The gliders. Note that the two-player arm passes NewGameMode to *both* calls and
	// discards NewGame's own `mode`, so **a two-player game can never be resumed** --
	// which is consistent with SavedGames.c, where a saved game records one glider.
	// See InitGlider for what the two calls do to each other.
	if w.TwoPlayer {
		w.InitGlider(&w.P1, NewGameMode)
		w.InitGlider(&w.P2, NewGameMode)
	} else {
		w.InitGlider(&w.P1, mode)
	}

	// The four LoadGraphic calls, which differ between the two arms only in which sheet
	// goes into the second slot. LoadGliderSheets reads TwoPlayer and does both cases;
	// its comment has the table and the reason RenderGlider's foil test is shaped the
	// way it is.
	w.LoadGliderSheets()

	// "Paint strip on screen black" (Play.c:141): the bottom twenty rows of the screen
	// rect.
	//
	// In the C, mainWindowRect *is* thisMac.screen (MainWindow.c:222), so this blackens
	// the bottom of the display. In the port, Screen is 640x480 and the house rect is
	// the top 640x460 of it (render.NewView), so the strip is exactly rows 460..480 --
	// the scoreboard band, and the only part of Main the room path can never reach. So
	// the statement that reads like housekeeping in the C is, here, the scoreboard's
	// own clear, and it has to happen before anything blits a board into it.
	strip := w.R.V.Screen
	strip.Top = strip.Bottom - ScoreboardTall
	w.Main.Fill(strip, render.Black8)

	// SetPort(workSrcMap); PaintRect(&workSrcRect). Redundant in the port and kept: the
	// composition below ends in RestoreWorkMap, which overwrites the whole work map
	// from a background that DrawLocale itself filled black. It is transcribed because
	// it is the C's guarantee rather than a consequence, and because a future
	// composition that stops covering every pixel would need it back.
	w.R.Work.Fill(w.R.V.WorkRect, render.Black8)

	w.Rebuild() // DrawLocale()
	w.RefreshScoreboard(NormalTitleMode)

	// The three-way ladder. All three arms dump the same rect; they differ only in what
	// is shown over it first, and both of those *block* -- BringUpBanner puts up a modal
	// alert with the author's message, DisplayStarsRemaining a countdown. That is why
	// InitGarbageRects comes after this and not before.
	switch mode {
	case NewGameMode:
		w.BringUpBanner()
	case ResumeGameMode:
		w.DisplayStarsRemaining()
	}
	w.DumpScreenOn(w.R.V.JustRoomsRect)

	w.InitGarbageRects()

	w.P1.StartGliderFadingIn(w)
	if w.TwoPlayer {
		// Player 2 fades in and is then immediately idled and hidden, so a two-player
		// game starts with one glider on screen and the second waiting. TagGliderIdle
		// freezes it for thirty frames with no visual tell at all, which is a real
		// usability problem in the original -- see docs/IMPROVEMENTS.md 2.16.
		w.P2.StartGliderFadingIn(w)
		w.P2.TagGliderIdle(w)
		w.P2.DontDraw = true
	}

	// Once per game, not once per room: the phone's schedule outlives every room change
	// and every death. See telephone.go.
	w.InitTelephone()

	// `wasPlayMusicPref = isPlayMusicGame` (Play.c:207), restored at :237. The
	// round-trip exists because the game-over sequence can change the preference --
	// TestHighScore's dialogue has a music checkbox -- and the change is meant to apply
	// to the next game rather than retroactively to this one's teardown.
	wasPlayMusicPref := w.PlayMusicGame

	// MaxMem(&growBytes) is a Toolbox heap compaction whose result is assigned to a
	// local and never read. Nothing to port.

	// StopMovie/SetMovieActive on the room's TV (COMPILEQT). Movie playback is out of
	// scope; see ReadyLevel's note.

	// ---- everything before this line is set-up ----
	w.Playing = true
	w.PlayGame()
	// ---- everything after it is after a game has ended ----

	w.PlayMusicGame = wasPlayMusicPref

	// ZeroMirrorRegion(). Inlined for the same reason ReadyLevel inlines NilSavedMaps:
	// the C is freeing a region handle and there is nothing to free here. Room.HasMirror
	// is cleared with the list because Rebuild derives one from the other and leaving
	// them disagreed is the bug that invariant exists to prevent.
	w.R.MirrorRects = w.R.MirrorRects[:0]
	w.R.HasMirror = false

	// TwoPlayer is cleared *here* and nowhere else, so a two-player game leaves the
	// world one-player. 1.9's menu is what sets it, once, immediately before calling
	// this function.
	w.TwoPlayer = false
	w.TheMode = SplashMode
	// InitCursor() -- see HideCursor above.
	w.StartIdleMusic()
	w.R.SavedMaps = w.R.SavedMaps[:0] // NilSavedMaps()
	w.BlackenScoreboard()
	// UpdateMenus(false) -- 1.7's menu bar.

	// `if (!gameOver)` guards a splash restore here (Play.c:257-273), and it reads oddly
	// and is right: a game that ended with GameOver set has already been followed by
	// DoGameOver or DoDiedGameOver, and *those* redraw the splash themselves
	// (GameOver.c:68). This arm is the other way out of PlayGame -- the player quit --
	// where nothing has redrawn anything and the window still holds the room.
	//
	// **Not transcribed, deliberately.** What the C is doing is scaling the splash art
	// into workSrcMap and invalidating the window so the update event paints the title
	// screen from it. In this port the title screen is internal/shell's: it is composed on
	// the shell's own surface and presented on the pass after this function returns
	// (shell.go's Run), so nothing ever reads the game's work map again, and filling it
	// with splash art paints 294,400 pixels that no host can display. A `-dump` run does
	// not see them either -- restoreSplashScreen presents nothing, and a dump writes on
	// Present.
	//
	// Leaving them out is what lets the fidelity harness state the invariant it exists to
	// check: after the last frame the work map equals the background map in a room where
	// nothing animates, so any draw that forgot to register a back rect shows up as a
	// hash mismatch (internal/replay's Result.Planes, and the room-70 case in
	// replay_test.go). A teardown that repaints Work would make every script's work hash
	// the same constant. See docs/IMPROVEMENTS.md 2.62.
	//
	// The two endings keep their call, because there the splash is on screen a moment
	// later: DoGameOver composes the starfield over it and presents (gameover.go:140).

	// WaitCommandQReleased() blocks until the player physically lets go of Command-Q.
	// It exists so that the keystroke that ended the game does not immediately register
	// again on the splash screen. It is not transcribed: the port has no raw key map at
	// this level, and requiring a physical release to leave a game is a thing a modern
	// build should not do -- see docs/IMPROVEMENTS.md 2.15.

	w.DemoGoing = false
	w.IncrementModeTime = w.Ticks() + IdleSplashTicks
}

// ---------------------------------------------------------------------------
// PlayGame (Play.c:430-597)
// ---------------------------------------------------------------------------

// PlayGame is the frame loop. It returns when the game is over.
//
// The C has the body twice, once per player count, and the two copies differ in exactly
// three places: the second GetInput, the second HandleGlider, and the demo-input branch
// that only the one-player arm has. Everything else -- eleven calls and four `gameOver`
// guards -- is duplicated verbatim. This is one body with guards, because two copies of
// eleven calls is two places for a later stage to insert a call into one of them.
//
// Four things about the loop are easy to get wrong and each is called out where it
// happens:
//
//  1. The `if (playing)` before RenderFrame is a *second* test of the loop condition,
//     and it means the last simulated frame of a game is never drawn.
//  2. HandleTriggers and HandleBands are outside every gameOver guard, so fuses keep
//     burning and bands keep flying through the sixteen-frame death countdown.
//  3. HandleDynamics runs before the player, not after, and is also ungated.
//  4. The frame's pacing is inside RenderFrame, at its end. A frame that returns early
//     -- game over, or `!playing` -- is not paced at all.
func (w *World) PlayGame() {
	// `quitting` cannot become true from inside this loop: in the C it is set by the
	// Quit menu item, and the event pump below cannot reach a menu because
	// HandlePlayEvent handles only update and suspend/resume events. It is transcribed
	// because the loop condition is the loop condition, and because 1.7's shell will
	// want exactly this door -- a window close while a game is running has to end the
	// game without going through the game-over sequence.
	for w.Playing && !w.Quitting {
		// The two clocks, and they are separate. Frame is monotonic and is what every
		// timer in the game is measured against; EvenFrame is a stored flag that three
		// other writers can flip out of step with it, permanently. See World.EvenFrame,
		// and note that RenderFrame reads EvenFrame and writes neither.
		w.Frame++
		w.EvenFrame = !w.EvenFrame

		// The event pump, and it is the whole of the game's pause-on-deactivate. Two
		// things about it are worth stating:
		//
		// The `do { } while (switchedOut)` spins here for as long as the application is
		// in the background, so a switched-out game stops advancing mid-frame -- after
		// the two clocks have already been bumped. So every deactivation costs the game
		// exactly one frame's worth of animation phase, which is invisible but real.
		//
		// The whole thing is inside `if (doBackground)`, and doBackground is a *user
		// preference*. With it off the game never pumps events at all: it does not
		// pause when it loses the foreground, and it never repaints on an update event.
		// That is a bad default to ship -- see docs/IMPROVEMENTS.md 2.21 -- and it is
		// transcribed rather than fixed because the fix changes when the simulation
		// advances.
		//
		// The loop body is pumpWhileSwitchedOut, in pause.go, because it draws. The C
		// leaves a backgrounded game holding a frozen frame with nothing on it, which is
		// indistinguishable from a hung one (2.28).
		if w.DoBackground {
			w.pumpWhileSwitchedOut()
		}

		w.HandleTelephone()

		// ---- the shared body of the C's two arms ----

		w.HandleDynamics()

		if !w.GameOver {
			// Back to back and with nothing in between, which is the contract
			// World.KeyPoll documents: the C polls the hardware only inside player 1's
			// call and lets player 2 read the snapshot.
			if !w.TwoPlayer && w.DemoGoing {
				w.GetDemoInput(&w.P1)
			} else {
				w.GetInput(&w.P1)
				if w.TwoPlayer {
					w.GetInput(&w.P2)
				}
			}
			w.HandleInteraction()
		}

		// Ungated on purpose. A trigger armed on the frame the last glider died still
		// fires during the countdown, and a band already in flight still lands.
		w.HandleTriggers()
		w.HandleBands()

		if !w.GameOver {
			w.HandleGlider(&w.P1)
			if w.TwoPlayer {
				w.HandleGlider(&w.P2)
			}
		}

		// The second test of the loop condition. It is `playing`, **not** `!gameOver`:
		// a game that has just been flagged over keeps rendering for sixteen frames,
		// which is the point of the countdown. What this guard actually catches is
		// DoGameOver having already run inside this same frame -- which cannot happen,
		// because the countdown block is below -- and the frame after PlayGame's caller
		// has cleared Playing. Either way the effect is that the *final* simulated
		// frame of a game is never drawn, and never paced.
		if w.Playing {
			// MoviesTask on the room's TV (COMPILEQT). Out of scope; see ReadyLevel.
			w.RenderFrame()
			w.HandleDynamicScoreboard()
		}

		// ---- the game-over countdown (Play.c:499-546) ----
		if w.GameOver {
			w.CountDown--
			if w.CountDown <= 0 {
				// HideGlider erases the glider from the screen for good. It is called on
				// P1 unconditionally, even in a two-player game where P2 may be the one
				// still on screen -- so a two-player game over leaves player 2's glider
				// painted into the last frame. That is the original's, and it is
				// invisible because the game-over screen paints over everything a moment
				// later.
				w.HideGlider(&w.P1)
				w.RefreshScoreboard(NormalTitleMode)

				// BUILD_ARCADE_VERSION: "Need to paint over the scoreboard black."
				w.arcadeBlackenBoard()

				// Mortals is -1 or lower here. Below -1 means both players of a
				// two-player game are out; exactly -1 with one player means the house
				// was not finished. DoGameOver is the *win* -- the player completed the
				// house -- and DoDiedGameOver is the loss. Both clear Playing, which is
				// what ends the loop (GameOver.c:62 and :492).
				if w.Mortals < 0 {
					w.DoDiedGameOver()
				} else {
					w.DoGameOver()
				}
			}
		}
	}

	// The unconditional arcade block (Play.c:551-593), which is the same two halves as
	// the one inside the countdown and runs whether or not the game ended in a game
	// over. So a player who quits mid-game also gets the board blacked and redrawn.
	w.arcadeBlackenBoard()
}

// ---------------------------------------------------------------------------
// HandlePlayEvent (Play.c:392-426) and the three arms it dispatches to
// ---------------------------------------------------------------------------

// HandlePlayEvent pumps one host event.
//
// The C is a WaitNextEvent with `sleep = 2` and three arms. **The sleep is not a frame
// limiter and must not become one.** It is the Toolbox's "yield the CPU for up to two
// ticks if there is nothing to do", it only has any effect when the event queue is
// empty, and the game's real pacing is the busy-wait at the end of RenderFrame
// (awaitFrame). A port that turned this into a second sleep would halve the frame rate
// on an idle queue and leave the two pacers fighting. So the port has one pacer, and
// this function does not wait for anything.
//
// Only the *classification* of events is outside the game: which host message means "we
// lost the foreground" is a platform question, and what the game does about it is not.
// So the hook is a bare func() and the three arms are the three methods below, which a
// host implementation calls.
func (w *World) HandlePlayEvent() {
	if w.PlayEvent == nil {
		// No host: no events. SwitchedOut can never become true, so PlayGame's pump
		// loop runs exactly once and cannot stall. That is required rather than
		// convenient -- a headless run drives the whole frame loop through here.
		return
	}
	w.PlayEvent()
}

// RefreshGameWindow is HandlePlayEvent's updateEvt arm (Play.c:400-411): the window was
// exposed, so put the room back on it.
//
// It is the *only* thing that repairs the screen after something has drawn over it, and
// that makes it more important than it looks. RestoreEntireGameScreen -- the teardown
// after a modal dialogue -- recomposes the work map and never copies it to the screen;
// it relies on the dialogue's disappearance generating an update event that lands here.
// With DoBackground false, that event is never pumped, so the screen stays black behind
// the dialogue until the dirty rects happen to cover it. See RestoreEntireGameScreen.
func (w *World) RefreshGameWindow() {
	r := w.R.V.JustRoomsRect
	w.Main.Copy(w.R.Work, r, r, render.SrcCopy)
	w.RefreshScoreboard(NormalTitleMode)
	w.present()
}

// Suspend and Resume are the two halves of HandlePlayEvent's osEvt arm (Play.c:412-425).
//
// They are asymmetric in the C and the asymmetry is transcribed: resume clears the flag
// *before* toggling the music, suspend sets it *after*. ToggleMusicWhilePlaying reads
// the flag, so the two orders mean the same thing -- "the music follows the flag" -- and
// only by accident. Written down because a reader who tidied them into the same shape
// would be tidying a coincidence.
func (w *World) Resume() {
	w.SwitchedOut = false
	w.ToggleMusicWhilePlaying()
	// HideCursor()
}

// Suspend is the deactivation half. See Resume.
func (w *World) Suspend() {
	// InitCursor()
	w.SwitchedOut = true
	w.ToggleMusicWhilePlaying()
}

// ---------------------------------------------------------------------------
// InitGlider (Play.c:307-368)
// ---------------------------------------------------------------------------

// InitGlider puts one glider at the start of a game.
//
// Its name undersells it: two thirds of what it writes are not the glider's. `theScore`,
// `mortals`, `batteryTotal`, `bandsTotal`, `foilTotal`, `showFoil` and
// `numStarsRemaining` are all whole-game state, and they are written here because in a
// one-player game "initialise the glider" and "initialise the game" happen at the same
// moment.
//
// **In a two-player game they are written twice**, and that is the reason this function
// is worth reading closely. NewGame calls it for both gliders, so:
//
//   - CountStarsInHouse runs twice and walks the whole house twice. Wasteful, harmless.
//   - `mortals` is set to InitialGliders and then, because TwoPlayer is true, has
//     InitialGliders added -- giving 4. The second call repeats that from scratch, so
//     the answer is 4 and not 8. The `mortals = kInitialGliders` assignment is what
//     saves it; an idiomatic `mortals += kInitialGliders` would have given 8.
//   - The score and the three inventories are zeroed twice, which is why they are
//     shared rather than per-player. Two players draw on one battery.
//
// The resume arm reads World.SavedGame, which is the port's `smallGame` and is *not* the
// house's embedded saved game -- see that field. ResumeSavedGame (savegame.go) is the only
// thing that fills it, which is what makes "nobody calls NewGame(ResumeGameMode) without a
// saved game" true by construction rather than by convention: with the field left zero this
// arm would give the player no lives, no stars and a glider at (0,0).
func (w *World) InitGlider(g *player.Glider, mode int16) {
	g.Dest = w.WhereDoesGliderBegin(mode)

	if mode == ResumeGameMode {
		w.StarsLeft = w.SavedGame.WasStarsLeft
	} else if mode == NewGameMode {
		w.StarsLeft = w.CountStarsInHouse()
	}

	if mode == ResumeGameMode {
		w.Score = w.SavedGame.Score
		w.Mortals = w.SavedGame.NumGliders
		w.Battery = w.SavedGame.Energy
		w.Bands = w.SavedGame.Bands
		w.Foil = w.SavedGame.Foil
		g.Mode = w.SavedGame.GliderState
		g.Facing = w.SavedGame.Facing != 0
		w.ShowFoil = w.SavedGame.ShowFoil != 0

		// A two-arm switch with a default, which is the C's way of saying "burning is
		// the only mode worth restoring". Every other saved mode -- mid-transporter,
		// half way up a staircase, dissolving into foil -- is discarded and the glider
		// simply stands up. Note that FlagGliderNormal overwrites the Mode just read
		// out of the save, so the field's only real use is choosing this branch.
		switch g.Mode {
		case player.GliderBurning:
			g.FlagGliderBurning(w)
		default:
			g.FlagGliderNormal(w)
		}
	} else {
		w.Score = 0
		w.Mortals = InitialGliders
		if w.TwoPlayer {
			w.Mortals += InitialGliders
		}
		w.Battery = 0
		w.Bands = 0
		w.Foil = 0
		g.Mode = player.GliderNormal
		g.Facing = player.FaceRight
		g.Src = player.GliderSrc[player.SpriteRight]
		g.Mask = player.GliderSrc[player.SpriteRight]
		w.ShowFoil = false
	}

	// The shadow, and note what it takes from Dest: the horizontal edges only. Its top
	// is the fixed ShadowTop and its height the fixed ShadowHigh, so the shadow starts
	// on the floor plane however high up the room the glider begins. See
	// Glider.DestShadow.
	g.DestShadow = player.Rect{
		Top: player.ShadowTop, Left: g.Dest.Left,
		Bottom: player.ShadowTop + player.ShadowHigh, Right: g.Dest.Left + player.GliderWide,
	}
	g.WholeShadow = g.DestShadow

	g.HVel = 0
	g.VVel = 0
	g.HDesiredVel = 0
	g.VDesiredVel = 0

	g.Tipped = false
	g.Sliding = false
	g.DontDraw = false

	// Not reset here, and each absence is a real one: Whole keeps whatever the previous
	// game left in it until the first MoveGlider recomputes it; EnteredRect keeps the
	// previous game's respawn point until FlagGliderNormal writes it; Frame and WasMode
	// keep their overloaded contents. The Flag* call in the resume arm covers the first
	// two, and the new-game arm reaches StartGliderFadingIn in NewGame a few lines
	// later, which covers Frame. Nothing covers WasMode.
}

// ---------------------------------------------------------------------------
// The room the game starts in (Play.c:372-385, House.c:196-239)
// ---------------------------------------------------------------------------

// SetHouseToFirstRoom is Play.c:372-378: two statements, and the only caller of
// GetFirstRoomNumber outside the editor.
func (w *World) SetHouseToFirstRoom() {
	w.ForceThisRoom(w.GetFirstRoomNumber())
}

// SetHouseToSavedRoom is Play.c:382-385: one statement. Note that it does not validate the
// saved room number -- ForceThisRoom's own -1 test is the only guard, and a save naming a
// room past the end of a house that has since been edited smaller would land on
// ThisRoom() == nil. See World.Room.
//
// It cannot be reached with such a number, because the one function that fills SavedGame
// refuses it first: ResumeSavedGame checks the room against the house before it assigns
// anything, which is where the C would have raised kYellowIllegalRoomNum from here instead
// -- mid-NewGame, with the locale half built. The check is there rather than added here so
// that this stays the transcription it is.
func (w *World) SetHouseToSavedRoom() {
	w.ForceThisRoom(w.SavedGame.RoomNumber)
}

// GetFirstRoomNumber is House.c:196-217: which room a new game starts in.
//
// It is three lines of arithmetic around one decision the author made in the editor, and
// it is careful in a way the rest of House.c is not: an out-of-range `firstRoom` is
// silently clamped to room 0 rather than reported, so a house whose start room was
// deleted still opens. That tolerance is why the port does not need to validate the
// field on load.
//
// The empty-house case returns -1 and sets NoRoomAtAll. Nothing in this stage reads the
// flag, and the -1 flows into ForceThisRoom, which treats it as a no-op -- so opening an
// empty house leaves the world on room 0 rather than crashing.
func (w *World) GetFirstRoomNumber() int16 {
	if len(w.H.Rooms) <= 0 {
		w.NoRoomAtAll = true
		return -1
	}
	firstRoom := w.H.FirstRoom
	if firstRoom >= int16(len(w.H.Rooms)) || firstRoom < 0 {
		firstRoom = 0
	}
	return firstRoom
}

// WhereDoesGliderBegin is House.c:223-239: the glider's starting rect, room-local.
//
// The C takes a `Rect *` and returns void; this returns the rect, because the C's only
// two callers both hand it the address of something they are about to overwrite anyway
// and the out-parameter carries no information the return value does not.
//
// **The C has no default arm and reads an uninitialised Point when given one.** `mode`
// selects between the saved game's stored position and the house's authored one, and any
// third value leaves `initialPt` holding stack garbage which is then offset into a rect.
// The port returns the authored position instead, and says so: it is the answer the
// caller almost certainly wanted, it is deterministic, and the alternative is
// reproducing undefined behaviour that no shipped path reaches.
//
// The rect is GliderWide x GliderHigh at the point, so the authored position is the
// glider's *top left* and not its centre -- an author placing a glider near the right
// wall in the editor gets it 48px further right than the marker suggests.
func (w *World) WhereDoesGliderBegin(mode int16) player.Rect {
	initialPt := w.H.Initial
	if mode == ResumeGameMode {
		initialPt = w.SavedGame.Where
	}
	return player.Rect{
		Top: 0, Left: 0, Bottom: player.GliderHigh, Right: player.GliderWide,
	}.Offset(initialPt.H, initialPt.V)
}

// CountStarsInHouse is Banner.c:89-110: how many stars the whole house holds.
//
// Called once per InitGlider -- so twice in a two-player game -- and it walks every
// object slot of every room. That is 24 * nRooms comparisons, up to about 7000 for the
// largest shipped house, once at the start of a game. Nobody has ever noticed.
//
// The `suite != kRoomIsEmpty` test is what makes a deleted room's stars not count, and
// it matters: a room slot is emptied by setting its suite to -1 and its objects are left
// in place, so without the test a house that had rooms deleted would ask the player for
// stars that cannot be reached. See house.Room and RoomExists, which uses the same
// sentinel for the same reason.
func (w *World) CountStarsInHouse() int16 {
	var numStars int16
	for i := range w.H.Rooms {
		rm := &w.H.Rooms[i]
		if rm.Suite == house_kRoomIsEmpty {
			continue
		}
		for h := 0; h < MaxRoomObs; h++ {
			if rm.Objects[h].What == Star {
				numStars++
			}
		}
	}
	return numStars
}

// ---------------------------------------------------------------------------
// RestoreEntireGameScreen (Play.c:800-819)
// ---------------------------------------------------------------------------

// RestoreEntireGameScreen puts the room back after a dialogue has been over the window.
//
// Its callers are the pause and the command-key handlers -- anything that put a modal
// window up mid-game. It black-fills the whole screen, recomposes the locale, and
// redraws the scoreboard.
//
// **It never copies the work map to the screen**, and that is not an omission in the
// port. The C does not either: it paints the *window* black, composes into the *work
// map*, and stops. What makes the room reappear is the update event the vanishing
// dialogue generates, which HandlePlayEvent turns into a whole-justRoomsRect blit
// (RefreshGameWindow). So the redraw is delivered by the event pump, one frame later,
// and with DoBackground false it is not delivered at all -- the screen stays black
// except where the dirty rects happen to fall. That is a visible bug in the original in
// its default configuration; docs/IMPROVEMENTS.md 2.5 covers it with the rest of
// DoPause.
//
// It calls Rebuild and not ReadyLevel: no NilSavedMaps before, no InitGarbageRects
// after, so the dirty-rect state and the frame clock survive a pause. See readylevel.go.
func (w *World) RestoreEntireGameScreen() {
	// HideCursor()
	w.Main.Fill(w.R.V.Screen, render.Black8)
	w.Rebuild()
	w.RefreshScoreboard(NormalTitleMode)
}

// ---------------------------------------------------------------------------
// Input, one step above the player package
// ---------------------------------------------------------------------------

// GetInput is Input.c:281-379 reached through the KeyPoll hook: resolve one glider's
// keys and hand them to its own GetInput.
//
// The split is player.Input's: the raw key map is a platform fact, the mapping from keys
// to thrust is not. See World.KeyPoll for why one poll must serve both gliders, and
// player.Keys for why the three unbound keys are resolved before they get here.
func (w *World) GetInput(g *player.Glider) {
	var k player.Keys
	if w.KeyPoll != nil {
		k = w.KeyPoll(g)
	}

	// One assignment rather than a constructor: NewWorld does not take a Fixes, because
	// the host sets w.Fix after the world exists, so there is no single moment at which
	// the two could be tied together once. Doing it here costs a bool store per glider per
	// frame and cannot go stale, which is the trade a settings screen that can be opened
	// mid-game needs anyway. See game.Fixes.Player2GiveUp.
	w.In.Player2GiveUp = w.Fix.Player2GiveUp

	w.In.GetInput(g, w, k)
}

// HandleGlider is Player.c:1339, and exists here only so PlayGame's two call sites read
// like the C's. The glider's state machine is entirely in the player package.
func (w *World) HandleGlider(g *player.Glider) {
	g.HandleGlider(w)
}

// GetDemoInput landed with 1.8b and lives in demo.go, beside DoDemoGame and the recorder
// hook. It is not GetInput with a different key source -- the original's demo path is a
// second copy of the function with five of its guards missing -- which is why it is a file
// of its own rather than a branch above.
//
// Still true and still worth knowing: replaying the original's own `'demo'` stream against
// the *original's* frames requires the random stream to match the Mac's, and rand.go's
// Random() is a reconstruction that has never been checked against a real trace. What the
// port can check -- and does, in internal/replay -- is that the stream replays identically
// twice here. See docs/IMPROVEMENTS.md 2.18.

// ---------------------------------------------------------------------------
// The stubs PlayGame and NewGame call, each with the stage that fills it
// ---------------------------------------------------------------------------
//
// Same convention as the nine renderers in render_frame.go: the call is transcribed in
// its real position now, so that filling it in later is a change to one function body
// and cannot move anything else.

// HandleDynamics landed with 1.5c and lives in dynamics.go.

// HandleBands landed with 1.5e and lives in bands.go. It stays ungated like
// HandleDynamics: a band in flight when the last mortal is spent keeps flying, and can
// still trip a switch after the glider that fired it is dead.

// DoGameOver and DoDiedGameOver landed with 1.7d and live in gameover.go, along with
// restoreSplashScreen. Both clear Playing, which is what ends PlayGame's loop; the win
// path does it first (GameOver.c:62) and the loss path last (GameOver.c:492).

// BringUpBanner and DisplayStarsRemaining landed with 1.7d and live in banner.go. Both
// block, which is why InitGarbageRects follows them in NewGame rather than preceding
// them, and neither blocks *here*: the waiting is the host's, through World.Wait.
