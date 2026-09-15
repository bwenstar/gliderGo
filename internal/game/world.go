// Package game is the running Glider PRO game: the live state the original kept
// in file-scope globals across a dozen translation units, and the code that reads
// and writes it.
//
// It exists as one package for the reason the original's globals were one segment.
// Play.c's frame loop, Interactions.c's hot-spot dispatcher, Dynamics*.c's movers,
// Objects.c's state engine and Transit.c's room changes are mutually recursive:
// MoveGliderUpStairs calls MoveRoomToRoom, which rebuilds the room, which calls
// CreateActiveRects, which reads the same masterObjects table HandleSwitches
// writes. Splitting that into packages would need an interface at every cut, and
// the cuts are not where the seams are. So it is one package, and the seams that
// *are* real -- the renderer, the house codec, the player's own state machine --
// stay in their own packages behind narrow surfaces.
//
// Two types carry everything. World is the game; Room is the composed locale the
// glider is standing in. The split is not tidiness: Room is everything DrawLocale
// rebuilds from scratch on a room change, and World is everything that survives
// one. Putting a field on the wrong side of that line is the bug class this
// package is arranged to make visible, and there is one field in the original that
// is on the wrong side -- tvOn, see Room.TVInRoom.
package game

import (
	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/render"
)

// Rect is the QuickDraw rect, shared with internal/house and internal/render so no
// conversion is needed across those two boundaries. internal/game/player declares
// its own identical struct; convert with player.Rect(r) at that boundary, which is
// a no-op the compiler checks.
type Rect = house.Rect

// Point is the QuickDraw point: vertical first.
type Point = house.Point

// World is the game. Every field names a file-scope global of the original, and
// the comment gives the C identifier where the Go name differs.
//
// It is deliberately one flat struct rather than a nest of subsystems. In the C
// these are 40-odd globals that any function may touch, and a nested arrangement
// would suggest an encapsulation that does not exist -- HandleInteraction writes
// theScore, batteryTotal, bandsTotal, foilTotal, starsLeft, mortals and the room
// itself, all in one dispatcher. Flat and honest beats nested and wrong.
type World struct {
	// H is thisHouse: the house handle. Object state lives here and nowhere else,
	// which is why SetObjectState writes through it and every other copy is a
	// cache that has to be invalidated by hand.
	H *house.House

	// R is the current locale. It is a field, not a pointer, and it is *rebuilt in
	// place* by Rebuild rather than replaced -- see the note on Rebuild for why
	// that has to be a method.
	R Room

	// RandSeed is qd.randSeed: the state of the one random stream, which every
	// RandomInt call in the game advances. It is a field rather than a package
	// global because the *order* of draws is observable -- the flame phases, the
	// pendulum starts, the balloon jitter, the telephone and the wind chimes all
	// pull from it in composition order, so two runs that consume the same sequence
	// look identical and a replay can be pinned.
	//
	// Seeded once per game and never re-seeded on a room change, exactly as the
	// original seeds qd.randSeed from the clock at launch (Utilities.c:60) and
	// leaves it alone. A fixed seed makes a whole game reproducible; see Random.
	RandSeed int32

	// P1 and P2 are theGlider and theGlider2. Values, not pointers, so that
	// &w.P1 is a stable address for the whole game -- the C passes &theGlider
	// into every handler and stores no gliders anywhere else, and a slice would
	// invalidate those addresses on growth.
	P1, P2 player.Glider

	// TwoPlayer is twoPlayerGame. OneLeft is onePlayerLeft: one player has used
	// their last mortal and the survivor is finishing alone. DeadWhich is
	// playerDead, whose true value is player.Player1 -- and which names the player
	// who *is* dead, so every read of it selects the *other* glider.
	TwoPlayer bool
	OneLeft   bool
	DeadWhich bool

	// FirstPlayer (Transit.c:18) is which of the two triggered the pending room
	// change, and is what decides who is placed first on arrival.
	FirstPlayer bool

	// Escaped is otherPlayerEscaped: the boundary the other player left through
	// while this one was still in the room, or NoOneEscaped.
	Escaped int16

	// Suicide is playerSuicide: the player pressed the give-up key, which skips
	// the death animation but still spends a mortal.
	Suicide bool

	// ActiveRectEscaped is activeRectEscaped (Interactions.c): the index into
	// Room.Hot of the transport rect the first player left through, so the second
	// has to use the *same object* rather than merely the same kind of object.
	// Written and read by the four link arms of HandleHotSpotCollision only.
	//
	// Storing a hot-spot index across frames is normally unsafe -- the table is
	// rebuilt from scratch on every room change, so an index outlives its meaning.
	// It is safe here for one reason: the only window in which it is read is
	// between the first glider arming and the second agreeing, and both happen in
	// the same room, because the room cannot change until they agree. On any other
	// path out of that window the value is simply never read again.
	//
	// **It is never initialised in the original**, in the C's BSS or in NewGame, so
	// its value before the first transport is 0 -- and Go's zero matches. That
	// matters exactly once: if a player reaches a transporter while the other is
	// already waiting at a *different* transporter with no arm having happened
	// (which the deadlock of state 4 can produce), hot spot 0 is the one that
	// accidentally matches.
	ActiveRectEscaped int16

	// Triggers is triggers[] (Triggers.c:27): sixteen fuses, each armed by the
	// glider touching a trigger plate and fired by HandleTriggers when its timer
	// runs out. Fixed-size, because FindEmptyTriggerSlot returning -1 -- and the
	// seventeenth simultaneous trigger silently not firing -- is observable.
	// Cleared by ZeroTriggers on every room change.
	Triggers [MaxTriggers]TriggerSlot

	// Phone is thePhone and Chimes is theChimes (Play.c): the two ambience clocks.
	// Both are the same struct and the chimes use only its first field; see
	// PhoneState. Phone survives every room change and every death, Chimes is
	// gated per room on Room.NumChimes.
	Phone  PhoneState
	Chimes PhoneState

	// Score is theScore, a long in the C and therefore int32 here. Rooms visited,
	// not points, is what the high-score board sorts on; see internal/house.Scores.
	Score int32

	// GameOver and CountDown are the game-over latch and its delay (GameOver.c:239-240).
	//
	// FlagGameOver sets both and nothing else happens for CountDown frames: the loop
	// keeps rendering, so the last glider's fade finishes on screen before the
	// game-over sequence takes the window (Play.c:499-502). Sixteen frames, about half
	// a second.
	//
	// GameOver is also a guard: OffAMortal returns immediately when it is set, which is
	// what stops a second death in those sixteen frames from spending another mortal.
	GameOver  bool
	CountDown int16

	// NumShredded is numShredded (DynamicMaps.c:33): how many shredded-glider
	// confetti clouds are live.
	//
	// The table itself and every writer of this counter are 1.5f's. It is here now
	// because OffAMortal's first act is to drop them (`if (numShredded > 0)
	// RemoveShreds()`), and a counter that is always zero with a no-op RemoveShreds
	// beside it keeps that statement transcribed in place rather than remembered.
	NumShredded int16

	// MusicMode and MusicCursor are musicMode and musicCursor (Music.c): which score
	// is playing and where in it. DontLoadMusic is the C's flag of the same name --
	// music is off, either by preference or because the bank failed to load -- and it
	// short-circuits SetMusicalMode. It starts true because this stage has no music at
	// all; 1.6 clears it when a bank loads.
	MusicMode     int16
	MusicCursor   int16
	DontLoadMusic bool

	// The inventory. Battery is signed and is one counter for two power-ups:
	// positive is battery charges, negative is helium (player.Env.BatteryTotal has
	// the full note). Foil is sheets remaining and ShowFoil is whether the foil
	// sprite sheet is loaded, which are separate because the sheet swap is a
	// graphics change that lags the count by a frame.
	Mortals   int16
	Battery   int16
	Bands     int16
	Foil      int16
	ShowFoil  bool
	StarsLeft int16

	// SaidFollowCount is saidFollow (Modes.c:13): how many times the two-player
	// "follow me" prompt has played, capped at three for the whole game rather than
	// per room (Modes.c:462-466).
	//
	// The C's name is `saidFollow` and the Env accessor keeps it; the field carries
	// the -Count suffix because Go will not let a struct have a field and a method of
	// the same name, and the method is the one whose name is fixed by the interface.
	// The same applies to SoundPlayer below.
	SaidFollowCount int16

	// Frame is gameFrame: frames since the game began, the clock every timer and
	// animation phase is measured against. NextFrame is the TickCount the next frame
	// is due at, which InitGarbageRects seeds (Render.c:690) and awaitFrame reseeds.
	//
	// EvenFrame is evenFrame, and it is **not** `Frame & 1`. It is a stored flag with
	// four writers: PlayGame's loop head, which toggles it beside Frame (Play.c:434-435),
	// a stalled ball or fish being kicked into motion mid-frame (Dynamics2.c:420), and
	// the kBall and kFish arms of AddDynamicObject at room-build time (Dynamics3.c:474,
	// :524), plus the launch-time init at InterfaceInit.c:131. Any
	// of the last three desynchronises the flame/star alternation from frame parity,
	// and nothing ever resynchronises it -- so a room with a ball in it animates its
	// candles on the frames a room without one animates its stars, permanently.
	// Deriving it from Frame would be a plausible-looking simplification that changes
	// what the player sees.
	Frame     int64
	EvenFrame bool
	NextFrame int64

	// RenderFrames counts calls to RenderFrame and has no counterpart in the C.
	//
	// It exists because Frame does not answer the question. A room transition renders
	// several frames inside one game frame (Transit.c's four RenderFrame calls), so
	// Frame counts simulation steps and this counts presentations. The fidelity
	// replays pin both, which is how a transition that draws the wrong number of
	// intermediate frames is caught.
	RenderFrames int64

	// TickCount is the 60.15 Hz tick source. nil derives it from Frame; see Ticks.
	TickCount func() int64

	// WaitTick is the body of the frame limiter's wait loop (Render.c:662).
	//
	// nil is the original: an empty loop body, i.e. a busy-wait that burns a core for
	// up to two ticks per frame. A released build passes something that sleeps. It is
	// a hook rather than an edit because the *end* of the wait must not change --
	// awaitFrame explains why the deadline arithmetic is load-bearing and the spinning
	// is not.
	WaitTick func()

	// Main is mainWindow: the pixels the player sees.
	//
	// In the original this is not a buffer at all -- it is the window, and CopyBits
	// into it lands in the frame buffer, so a rect copied to main is on screen the
	// instant the copy returns. The port cannot have that, because every backend it
	// will ever have (X11 shared memory, SDL, a browser canvas) takes a whole image
	// and uploads it. So Main is a real surface at the *screen's* size -- 640x480,
	// not the 640x460 the two offscreens are -- and Present is what makes it visible.
	//
	// It being screen-sized is what makes the scoreboard possible: rows 460..480 are
	// only ever written by Scoreboard.c, and nothing in the room path can reach them.
	Main *render.Surface

	// Present is the hook that pushes Main to a display. nil is a headless build and
	// is not a degraded one: every pixel is still composed into Main, so a headless
	// run and a windowed run produce byte-identical screens, which is what lets the
	// fidelity corpus check the display without a display.
	//
	// It is called at exactly the four points where the original's writes to
	// mainWindow first became visible to a player who was watching:
	//
	//   - the tail of CopyRectsQD, once per rendered frame;
	//   - DumpScreenOn, the whole-screen dump;
	//   - each of WipeScreenOn's 116 or 160 strips, which is what makes the wipe an
	//     animation rather than a jump;
	//   - HideGlider, whose erase is followed by a modal dialogue and not by a frame.
	//
	// Everything else -- CopyRectWorkToMain from a mode change, the dirty rects
	// themselves -- writes Main and waits, because the C's next visible moment is one
	// of those four and never sooner. See screen.go.
	Present func()

	// GlidSrc, Glid2Src and ShadowSrc are glidSrcMap, glid2SrcMap and shadowSrcMap:
	// the sprite sheets RenderGlider and DrawReflection blit out of.
	//
	// Two slots for four glider sheets, because the original reloads their *contents*
	// rather than holding all four open, and which sheet is in the second slot depends
	// on the player count. LoadGliderSheets has the table and SetShowFoil has the
	// two-player reload; between them they are the reason RenderGlider's foil test
	// reads `(!twoPlayerGame) && showFoil`.
	//
	// The masks are not separate fields. The C's one glidMaskMap is shared by all four
	// sheets, and the extractor baked it into each strip's own mask, so each surface's
	// Mask already *is* glidMaskMap. See LoadGliderSheets.
	GlidSrc   *render.Surface
	Glid2Src  *render.Surface
	ShadowSrc *render.Surface

	// Work2Main and Back2Work are the two dirty-rect lists (Render.c): what has to
	// be copied from the work map to the screen this frame, and what has to be
	// restored from the clean background to the work map first. Both are cleared by
	// InitGarbageRects and truncated by RenderFrame once CopyRectsQD has replayed them.
	//
	// Order matters twice over. Within a list, order is z-order: CopyRectsQD replays
	// Work2Main in append order, so a rect appended later covers one appended earlier,
	// which is why RenderFrame's call order is transcribed exactly. Between the lists,
	// the screen copy runs before the erase -- publish first, erase second.
	//
	// Overflow is observable and reproduced: all three adders guard on
	// `numWork2Main < kMaxGarbageRects - 1` and, when that fails, **silently drop the
	// rect**. Nothing merges it into a neighbour. A dropped work rect is a patch of
	// screen that is never copied forward; a dropped back rect is a patch of work map
	// that is never erased, so whatever was drawn there smears until something else
	// dirties it.
	//
	// Note the `- 1`: the guard stops at 47 of the 48 slots, so the last one is
	// unreachable. That is the original's off-by-one and the effective cap is 47.
	Work2Main []Rect
	Back2Work []Rect

	// Diag counts the two things about a frame that are otherwise invisible: the dirty
	// rects the port drops as faithfully as the original did, and the out-of-range reads
	// it declines to perform where the original went ahead. Nothing in the game reads it.
	// See guards.go.
	Diag Diagnostics

	// Pending is the resolved destination of the transit the glider is currently
	// inside: transRect, transRoom and linkedToWhat as one value. The player code
	// takes it as an argument rather than reaching for the object graph, which is
	// why player.Link exists at all.
	Pending player.Link

	// PrevRoom is previousRoom, which ForceThisRoom sets and the map window reads.
	PrevRoom int16

	// Ward and PhoneBitSet are the house's two flag bits, cached because they are read
	// per frame. HasMovie is whether this build found a QuickTime movie for the
	// house at all.
	//
	// PhoneBitSet is `phoneBitSet` (Play.c:54) and it **suppresses** the telephone
	// rather than enabling it: HandleTelephone's whole body is inside `if
	// (!phoneBitSet)` (Play.c:748). A house with the bit set is a house where the phone
	// never rings. The field carries the C's full name because `Phone` names the ring
	// timer (see below) and the two are opposites.
	Ward        bool
	PhoneBitSet bool
	HasMovie    bool

	// TVOn is tvOn, and it is the one global in the original that is on the wrong
	// side of the World/Room line: nothing ever resets it on a room change, so it
	// stays true after the player leaves the room whose TV they switched on.
	//
	// That is not a bug in the C, because every one of the seven sites that reads
	// it also tests tvInRoom, which *is* room-scoped and *is* reset. The port
	// keeps the field here, keeps Room.TVInRoom there, and keeps both conjuncts at
	// all seven read sites. Dropping either one -- "simplifying" TVOn into the
	// room, or dropping the TVInRoom test as redundant -- changes behaviour.
	TVOn bool

	// TriggerSoundExists stands in for the 'snd ' resource lookup inside
	// LoadTriggerSound (Sound.c:265-303). It exists as a hook rather than a call
	// because whether a sound trigger gets a hot spot at all depends on whether
	// its sound loads, so a silent build composes measurably different rooms from
	// a build with audio -- and the fidelity corpus has to be able to choose.
	//
	// nil means "no sound system", which is exactly the original's dontLoadSounds
	// short circuit: LoadTriggerSound returns -1 and no kSoundIt rect is made.
	// That is the state of this stage; 1.6 supplies a real one.
	TriggerSoundExists func(soundID int16) bool

	// SoundPlayer is the injected implementation of PlayPrioritySound (Sound.c:40-85):
	// request a sound, which is granted only if nothing of higher priority is already
	// playing. A hook for the same reason as TriggerSoundExists -- 1.6 fills it in --
	// and nil means silence.
	//
	// It is not called `PlayPrioritySound`, even though that is the C's name, because
	// the method that satisfies player.Env owns that name (see env.go). The field is
	// the backend; the method is the call site's view of it.
	//
	// Larger is higher: PlayPrioritySound finds the quietest of the three channels
	// and plays only if `priority >= lowestPriority` (Sound.c:53-67). The scale runs
	// from 100 for a wall bump to the 800s for the noisy appliances, so a blower's
	// 701 displaces most things and is displaced by few.
	SoundPlayer func(sound, priority int16)

	// ---------------------------------------------------------------------
	// Play.c's lifecycle globals
	// ---------------------------------------------------------------------

	// Playing is `playing` (Play.c:50): the frame loop's run flag. NewGame sets it
	// true on the line before it calls PlayGame and the only things that clear it are
	// DoGameOver and DoDiedGameOver, so it answers "is a game in progress" and nothing
	// else. PlayGame tests it twice per frame -- once as the loop condition and once
	// again before rendering -- and the second test is why the last simulated frame of
	// a game is never drawn. See PlayGame.
	Playing bool

	// Quitting is `quitting`, a Main.c global: the application is shutting down.
	// PlayGame's loop condition tests it every frame, and **nothing inside the loop
	// can set it** in this port, because the C's only writer is the Quit menu item.
	// The test is transcribed anyway; see PlayGame.
	Quitting bool

	// TheMode is `theMode`: SplashMode, EditMode or PlayMode. NewGame writes it twice,
	// PlayMode at the top and SplashMode at the bottom, which is the whole of this
	// stage's interest in it. 1.7's shell is the other reader.
	TheMode int16

	// DemoGoing is `demoGoing`: this game is the attract-mode demo, so the one-player
	// arm reads its input from the recorded stream instead of the keyboard. Set by
	// DoDemoGame, cleared by NewGame's tail.
	DemoGoing bool

	// DoBackground is `doBackground`: the preference "keep playing while switched
	// out". It gates the entire event pump -- with it false, PlayGame never calls
	// HandlePlayEvent at all, so the game neither pauses on deactivation nor processes
	// an update event. See PlayGame and docs/IMPROVEMENTS.md 2.21.
	DoBackground bool

	// SwitchedOut is `switchedOut`: the application is in the background. The event
	// pump spins on HandlePlayEvent while it is set, which is the original's
	// pause-on-deactivate, and only a resume event clears it.
	SwitchedOut bool

	// NoRoomAtAll is `noRoomAtAll` (House.c:203): GetFirstRoomNumber was asked for the
	// first room of a house that has none. Nothing in this stage reads it -- the C's
	// readers are the editor and the house loader -- but it is written where the C
	// writes it so that 1.7 finds the flag rather than the symptom.
	NoRoomAtAll bool

	// IncrementModeTime is `incrementModeTime`: the tick at which the idle splash
	// screen advances to the next attract-mode panel. NewGame's last statement writes
	// it and DoDemoGame writes it again; 1.7's shell reads it.
	IncrementModeTime int64

	// SavedGame is `smallGame` (SavedGames.c:20), and it is **not** H.SavedGame.
	//
	// The two are easy to confuse and only one of them is live. `houseType.savedGame`
	// -- the 40 bytes at offset 820 of every house file -- is never read anywhere in
	// 1.1.2: House.c:139 and SavedGames.c:337/:341 write `hasGame` beside it and
	// nothing ever loads it back. The saved game the resume path actually uses comes
	// from a *separate file* (SavedGames.c:261-275) into this global. So a house's
	// embedded saved game is residue, and the port keeps both fields for that reason:
	// house.House.SavedGame round-trips the residue, and this one is the game.
	//
	// Filled by 1.10's loader. Until then it is zero, which makes ResumeGameMode start
	// the glider at (0,0) of room 0 with no lives -- so 1.10 is what makes resume mean
	// anything, and NewGameMode is the only mode this stage exercises.
	SavedGame house.Game

	// In is Input.c's file-scope pair of sound-throttle variables.
	//
	// One value, deliberately: the C's are globals and are therefore shared by both
	// gliders, and the sharing is audible (see player.Input). A two-player game must
	// pass the *same* Input to both GetInput calls, and having exactly one on World is
	// how that obligation is discharged rather than remembered.
	In player.Input

	// KeyPoll is the keyboard. It answers with one glider's four keys plus the three
	// unbound ones already resolved -- see player.Keys, which explains why resolution
	// happens above this call rather than inside it.
	//
	// nil is "no keyboard": every key up, every frame. That is the right default for a
	// headless run and for the fidelity corpus, and it is not a degraded mode -- a
	// glider with no input still falls, burns, drifts on a fan and dies, so most of the
	// simulation is exercised without one.
	//
	// The frame loop calls it once per glider, back to back, which is the C's
	// arrangement at Play.c:452-453. The C polls the hardware only on player 1's call
	// and lets player 2 read the snapshot; a hook implementation must give the same
	// answer to both calls in one frame, i.e. it must not pump host events between
	// them. The loop guarantees it makes no other call in between.
	KeyPoll func(g *player.Glider) player.Keys

	// PlayEvent is the host event pump: the WaitNextEvent at HandlePlayEvent's head.
	//
	// Only the *classification* of events is out here. The three arms the C acts on --
	// window update, suspend, resume -- are transcribed as RefreshGameWindow, Suspend
	// and Resume on this type, and a host implementation calls them. That split is why
	// this is a bare func() rather than something returning an event: the part that
	// differs per platform is which host message means "we lost the foreground", and
	// the part that must not differ is what the game does about it.
	//
	// nil is a headless build. SwitchedOut can then never become true, so the pump's
	// do/while runs exactly once and the loop cannot stall -- see PlayGame.
	PlayEvent func()

	// The four music preferences and states (Music.c, Prefs.c). All read by NewGame's
	// two music ladders and nowhere else in this stage.
	//
	// PlayMusicGame and PlayMusicIdle are `isPlayMusicGame` and `isPlayMusicIdle`: the
	// player's two separate choices about music during a game and music on the splash
	// screen. MusicOn is `isMusicOn`, whether a channel is actually open, and
	// FailedMusic is `failedMusic`, set once when StartMusic has failed so the alert is
	// not repeated.
	//
	// They stay false through this stage and DontLoadMusic keeps the ladders inert; see
	// StartGameMusic.
	PlayMusicGame bool
	PlayMusicIdle bool
	MusicOn       bool
	FailedMusic   bool

	// WasScoreboardMode is `wasScoreboardMode` (StructuresInit.c:70): which of the two
	// scoreboard layouts the board rects are currently positioned for. Initialised to
	// ScoreboardHigh at launch and written only by AdjustScoreboardHeight.
	//
	// It is a latch guarding an offset accumulation in the C, which is why it exists at
	// all -- see AdjustScoreboardHeight, where the port makes the body idempotent and
	// keeps the latch anyway.
	WasScoreboardMode int16

	// Board is the scoreboard's five offscreen maps (StructuresInit.c's boardSrcMap and
	// friends). Created once per world, deliberately not per locale: the score does not
	// reset when the player walks through a door, so it cannot live on the Scene that
	// DrawLocale rebuilds.
	Board *render.Scoreboard

	// DisplayedScore is `displayedScore` and DoRollScore is `doRollScore`
	// (Scoreboard.c:40-42): the score the board is currently showing, and whether it is
	// allowed to walk up to Score a few points at a time instead of jumping.
	//
	// The pair is what makes collecting a prize feel like collecting a prize: the number
	// climbs at ScoreRollAmount a frame with a tick of sound per step. DoRollScore is set
	// by every RefreshScoreboard and is never cleared anywhere in the shipped source, so
	// the "jump straight to the total" branch is dead code -- see HandleDynamicScoreboard.
	DisplayedScore int32
	DoRollScore    bool

	// The seven rects AdjustScoreboardHeight moves. Their unmoved forms are on
	// render.View, computed once from the screen's size; these are the placed copies, and
	// they are on World because the placement is a property of the game's chosen view.
	//
	// BoardGQDestRect and BoardPQDestRect are `boardGQDestRect` and `boardPQDestRect`:
	// where the glider count and the score go when either is refreshed on its own,
	// straight to the screen. BadgesDestRects is `badgesDestRects`, indexed by
	// render.FoilBadge and its three companions.
	BoardGQDestRect render.Rect
	BoardPQDestRect render.Rect
	BadgesDestRects [4]render.Rect

	// BoardDestRect is `boardDestRect` (StructuresInit.c:97-98): where on the screen the
	// scoreboard is blitted.
	//
	// **The port's initial value deviates from the original's, deliberately, and it is
	// the deviation that makes the score visible.** The C sets this to the board's source
	// rect offset up by ScoreboardTall, i.e. rows -20..0 of the window -- above the top
	// edge, outside every clip. With the default nine-neighbour view
	// AdjustScoreboardHeight never moves it, so **the shipped game draws no scoreboard at
	// all**: every blit into it is clipped away. The player sees no score, no lives and no
	// inventory, and the only reason that was ever tolerable is that the nine-neighbour
	// view fills all 640x460 with rooms and there is nowhere to put a board.
	//
	// The port has somewhere to put it. render.NewView makes Screen 640x480 and House the
	// top 640x460 of it, so rows 460..480 are a band the room path can never write --
	// which is also exactly the strip NewGame paints black (Play.c:141). So this starts at
	// (460,0,480,640) and the scoreboard is on screen. See docs/IMPROVEMENTS.md 2.9.
	BoardDestRect render.Rect
}

// Room is the composed locale: the central room the glider is in, the eight
// neighbours drawn around it, and every table DrawLocale rebuilds from scratch.
//
// Everything here is discarded and recomputed on a room change. That is the
// definition of the type: if a field survives a room change it belongs on World.
type Room struct {
	// Scene is the renderer's half of the same locale -- the surfaces, the nine
	// room numbers, the animation tables and the draw order. It is embedded rather
	// than referenced because in the original there is no distinction: numLights
	// and thisTiles are globals that both the drawing code and the collision code
	// read, and the embedding reproduces that shared reach without duplicating the
	// fields into two places that can disagree.
	*render.Scene

	// Hot is hotSpots[], the collision table. Cap MaxHotSpots; AddActiveRect
	// returns -1 rather than growing it.
	Hot []HotObject

	// Master is masterObjects[], the object graph: every object of all nine local
	// rooms, with its links resolved into indices into this same slice. Cap
	// MaxMasterObjects.
	Master []MasterObject

	// NumLocalMaster is numLocalMasterObjects: how many of Master belong to the
	// central room. The central room is listed first, so Master[:NumLocalMaster]
	// is exactly the central room's 24 slots -- which is what the dispatchers that
	// only care about the room the glider is in iterate over.
	NumLocalMaster int

	// LeftThresh and RightThresh are each both a coordinate and a flag: the escape
	// checks compare the glider against them, and compare them against
	// LeftWallLimit/RightWallLimit to decide whether there is a wall at all.
	LeftThresh, RightThresh int16

	// The four openings, from DetermineRoomOpenings. Note that these are not
	// symmetric in how they are derived: left and right come from the first and
	// last background tile, top and bottom from the background alone.
	TopOpen, BottomOpen, LeftOpen, RightOpen bool

	// ShadowVisible is shadowVisible: the *cached* flag, set once by ReadyLevel
	// and by RedrawRoomLighting, not the IsShadowVisible predicate. The two differ
	// in the original and the difference is load-bearing; see IsShadowVisible.
	ShadowVisible bool

	// TakingTheStairs suppresses the glider's own drawing for one frame while a
	// staircase transition completes.
	TakingTheStairs bool

	// HasMirror is true in a room holding a kMirror, which doubles every dirty
	// rect at a fixed offset (Modes.c:92-97).
	HasMirror bool

	// TVInRoom and TVMovieNumber are room-scoped and are reset here on every room
	// change, which is what makes World.TVOn safe to leave stale. See that field.
	TVInRoom      bool
	TVMovieNumber int16

	// NumChimes is numChimes: how many kChimes objects the *central* room holds.
	// It is incremented inside CreateActiveRects, which only runs for the central
	// room, so a chime in a neighbouring room does not shorten the interval --
	// a detail worth stating because the count reads like a locale-wide total.
	NumChimes int

	// TriggerSoundHeld is theSoundData[kMaxSounds-1] != nil, i.e. whether the one
	// reserved sound slot is occupied. It lives on Room and not on World because
	// DrawLocale calls DumpTriggerSound() in its reset head (RoomGraphics.c:58),
	// which frees the slot on every room change -- so its lifetime is exactly a
	// locale's even though the C's array is a file-scope global.
	//
	// It is what makes a room's *second* kSoundTrigger inert: the first one takes
	// the slot and every later one fails LoadTriggerSound's `!= nil` test and gets
	// no hot spot. That is a real one-per-room limit and not an artefact of the
	// port; see loadTriggerSound.
	TriggerSoundHeld bool
}

// NoOneEscaped is the sentinel for World.Escaped (GliderDefines.h).
const NoOneEscaped int16 = -1

// NewWorld starts a world on a house. Nothing is composed until Rebuild.
//
// The Scene is supplied rather than created here because a Scene needs a View and
// an Assets, and a headless test wants to choose both. `seed` is the initial state
// of the RandomInt stream -- pass a fixed value to make a whole run reproducible,
// or the clock to match the original's behaviour at launch. Zero and 0x7FFFFFFF are
// both fixed points of the generator and are nudged to 1; see Random.
func NewWorld(h *house.House, sc *render.Scene, seed int32) *World {
	w := &World{
		H:           h,
		RandSeed:    seed,
		Ward:        h.Ward(),
		PhoneBitSet: h.Phone(),
		Escaped:     NoOneEscaped,
		PrevRoom:    -1,
		// theMode = kSplashMode (InterfaceInit.c:154). A world exists before a game
		// does.
		TheMode: SplashMode,
		// dontLoadMusic. True until 1.6 loads a music bank; see SetMusicalMode.
		DontLoadMusic: true,
		// StructuresInit.c:70. High is the launch value and the nine-neighbour default
		// keeps it there, which is why AdjustScoreboardHeight is a no-op in practice.
		WasScoreboardMode: ScoreboardHigh,
	}

	// InterfaceInit.c:147-152, the launch-time glider identity. It is *not* in
	// NewGame, and putting it there would be wrong in a way that is invisible for one
	// player and fatal for two: NewGame runs once per game, and these are the
	// per-glider facts that outlive a game.
	//
	// Which is the one of the six that the simulation reads. GetInput tests it to
	// decide who owns the Command key (Input.c:283), OffAMortal stores it in
	// DeadWhich, and eight sites in Dynamics*.c compare it against PlayerDead to
	// decide which glider an enemy is allowed to hit. Leaving it unset makes both
	// gliders Player2, because Player2 is false -- so a two-player game would give the
	// Command key to nobody and let every enemy target the dead player's glider.
	//
	// The four key indices are the default bindings and nothing in the port reads them
	// yet; see the note on player.LeftArrowKeyMap for what they are and why player 2's
	// four modifier keys have to be rebindable before release.
	w.P1.Which = player.Player1
	w.P1.LeftKey = player.LeftArrowKeyMap
	w.P1.RightKey = player.RightArrowKeyMap
	w.P1.BattKey = player.DownArrowKeyMap
	w.P1.BandKey = player.UpArrowKeyMap

	w.P2.Which = player.Player2
	w.P2.LeftKey = player.ControlKeyMap
	w.P2.RightKey = player.CommandKeyMap
	w.P2.BattKey = player.OptionKeyMap
	w.P2.BandKey = player.ShiftKeyMap

	w.R.Scene = sc
	w.R.Hot = make([]HotObject, 0, MaxHotSpots)
	w.R.Master = make([]MasterObject, 0, MaxMasterObjects)
	// Main is the screen rect and not the house rect: see the field's own note. It
	// starts white, as a freshly created GWorld did; NewGame paints it before the
	// first frame reaches it.
	w.Main = render.NewSurface(int(sc.V.Screen.Wide()), int(sc.V.Screen.Tall()))

	// InitScoreboardMap's tail (StructuresInit.c:74-160): the five offscreen maps, and the
	// seven movable rects put where WasScoreboardMode already claims they are.
	//
	// The seeding is not redundant with AdjustScoreboardHeight, it is what makes that
	// function's latch tell the truth. The latch starts at ScoreboardHigh and the
	// nine-neighbour default *is* High, so NewGame's one call returns without doing
	// anything -- which in the C leaves the rects at their construction values, rows
	// -20..0, and is the whole reason the shipped game shows no scoreboard. Placing them
	// here means the latch and the rects agree from the first frame, and the port's
	// deviated High position is stated in exactly one place, placeScoreboard.
	w.Board = render.NewScoreboard(sc.V, sc.A)
	w.placeScoreboard(w.WasScoreboardMode)
	return w
}

// ThisRoom is thisRoom: the central room's record. It is read out of the house
// rather than kept as a separate copy, which is a deliberate divergence -- see
// SetObjectState for why the original's separate thisRoom copy exists and why not
// having it removes a whole class of cache-coherence bug rather than creating one.
func (w *World) ThisRoom() *house.Room {
	return w.Room(w.R.RoomNumber)
}

// Room returns a room by index, or nil if the index is out of range.
//
// The nil is not defensive padding. GetNeighborRoomNumber returns -1 for a room
// that does not exist, and the original then indexes rooms[-1] -- which on 68k
// read the tail of the preceding room's struct and, for the object slots, aliased
// tiles[6] and the low byte of openings (see the note in hotspots.go). Returning
// nil and having callers test it is the port's chosen divergence, and every caller
// that tests it says which of the C's out-of-range reads it is standing in for.
func (w *World) Room(n int16) *house.Room {
	if n < 0 || int(n) >= len(w.H.Rooms) {
		return nil
	}
	return &w.H.Rooms[n]
}

// NumberRooms is the numberRooms global: how many rooms the house really has.
// The stored NRooms field is advisory and RoomExists walks this count.
func (w *World) NumberRooms() int16 { return int16(len(w.H.Rooms)) }
