package main

// One game, from a file name to a score.
//
// This is the host half of playing: it opens a house, builds the art and the World
// around it, hangs the five hooks off it and calls NewGame, which does not return
// until the game is over. Everything in here is about *this machine*; nothing in
// here is about Glider PRO.
//
// It is a method on app rather than a function because two things outlive a single
// game and must not be rebuilt around each one. The window is one: a shell that
// closed and reopened its window between games would flash the desktop every time
// somebody died. The audio chain is the other, and for a stronger reason -- the mix
// goes to an external player's stdin (internal/audio/sink.go), so a pump per game
// would mean spawning and killing pw-play around every game and a click at both
// ends. The bank is a megabyte of samples that only has to be decoded once, too.
//
// What *is* per game: the house, its resource fork, the art cache, the Scene, the
// World and every hook on it. So play rebinds the four audio hooks and the score
// walk onto each new World (bindAudio), which is the one piece of wiring that is
// easy to get wrong -- all the music state lives on the World (internal/game/music.go),
// so an engine still calling last game's NextMusicPiece would be walking a dead
// game's score.

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bwenstar/gliderGo/internal/assetfs"
	"github.com/bwenstar/gliderGo/internal/audio"
	"github.com/bwenstar/gliderGo/internal/game"
	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/platform/backend"
	"github.com/bwenstar/gliderGo/internal/prefs"
	"github.com/bwenstar/gliderGo/internal/render"
	"github.com/bwenstar/gliderGo/internal/saved"
	"github.com/bwenstar/gliderGo/internal/scores"
	"github.com/bwenstar/gliderGo/internal/shell"
)

// app is the host: the things that outlive one game.
type app struct {
	o   *options
	win platform.Window
	fb  *platform.Framebuffer

	// p is the settings, shared with the settings screen, which edits it in place. So
	// this is not a snapshot and must not be copied: everything that reads a preference
	// reads it through here, and reads it at the moment it needs it.
	p *prefs.Prefs

	// canSave says there is somewhere to write p. See loadPrefs.
	canSave bool

	// store is the high-score boards, one file per house. A nil store is a session that
	// can play and cannot record -- -scores none, or no data directory -- and every method
	// on it tolerates that, so this is never checked at a call site (internal/scores).
	store *scores.Store

	// saves is the games in progress, one file per house, and it is nil under the same two
	// conditions -- -saves none, or nowhere to write. Unlike store it *is* checked at two
	// call sites, and both checks are the same decision: with nowhere to write, the game
	// gets no SaveGame hook (so CanSaveGame is false and the pause hint stops offering the
	// key) and the shell gets no Saved hook (so the menu row says why rather than offering
	// a resume nothing could perform).
	saves *saved.Store

	bank  *audio.Bank
	eng   *audio.Engine
	pump  *audio.Pump
	sink  audio.Sink
	where string // one word for the startup line: which player, or "none"

	// title is the cursor into the score that the title screen walks, and the only
	// World in the process that is not a game. See music.go.
	title *game.World

	// randSeed is the random stream, held here because the original holds it in
	// QuickDraw's globals. `qd.randSeed` is process-lifetime: the second game of a session
	// carries on from wherever the first one left the sequence, so its candles start on
	// different frames and its telephone rings after a different delay. A World that were
	// seeded afresh each game would replay the same "random" opening every time, which is
	// the one way a fixed seed can be less faithful than a clock. Resolved once by
	// resolveSeed and handed back at the end of every game; see play.
	randSeed int32

	// artErr is the first sticky asset error from any game this session. It is
	// reported rather than returned mid-session, because losing a finished game's
	// score to a missing PICT would be a worse trade than a line on stderr.
	artErr error
}

func newApp(o *options, p *prefs.Prefs, canSave bool) *app {
	a := &app{o: o, p: p, canSave: canSave, randSeed: resolveSeed(o.seed)}
	// The original's launch spends one draw before any game does: VariableInit picks the
	// editor's default flower out of the same stream (game.AdvanceRandSeed says where). So
	// the first game of a session starts on 16807 rather than on 1, and matching that is
	// what makes its first candle flame and its first pendulum agree with 1994's.
	a.randSeed = game.AdvanceRandSeed(a.randSeed)
	a.openScores()
	a.openSaves()
	return a
}

// resolveSeed turns -seed into the state the random stream starts the process at.
//
// The default is 1, and that is a fidelity decision rather than a convenience one: the
// original seeds `qd.randSeed` from the clock at `Utilities.c:61`, but that call sits inside
// `#if !TARGET_CARBON` and `GliderPRO/Prefix.h:1` sets TARGET_CARBON, so the shipped build
// begins every launch on the 1 that InitGraf left in the QuickDraw globals. Matching it is
// what makes this port's candles start on the frames the 1994 build's did, and it is the seed
// docs/analysis/toolbox-primitives.md §1.6's verified draw table is a table of.
//
// `-seed 0` asks for the clock instead. That is the 68k build's behaviour -- a real thing the
// game did on other hardware, which is why it is offered rather than removed -- and it is
// what a run that should not be the same run twice wants. The value is folded into the
// generator's legal range here rather than left to World.Random's guard, so that the two
// nanosecond readings in two billion that land on the recurrence's dead ends do not silently
// become the default seed and make a clock run look like a fixed one.
func resolveSeed(flag int64) int32 {
	if flag != 0 {
		return int32(flag)
	}
	seed := int32(uint32(time.Now().UnixNano()) & 0x7FFFFFFF)
	if seed == 0 || seed == 0x7FFFFFFF {
		seed = 16807 // 0 is a fixed point and 2^31-1 maps to it; any other state will do
	}
	return seed
}

// scoresNone is what -scores takes to mean "play, and record nothing". It is spelled the
// same as -prefs none and for the same reason: a run that must not touch the player's files
// -- a bisect, a bug report's reproduction, a shared machine -- should be one word away.
const scoresNone = "none"

// openScores decides where the boards live.
//
// A failure to resolve the directory is not fatal and does not even wait for a score to be
// earned before it is reported: somebody whose data directory cannot be found wants to know
// at startup, not after the one game they were going to get onto the board with.
func (a *app) openScores() {
	switch a.o.scoresDir {
	case scoresNone:
		return
	case "":
		st, err := scores.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "glidergo: high scores will not be recorded: %v\n", err)
			return
		}
		a.store = st
	default:
		a.store = scores.OpenDir(a.o.scoresDir)
	}
}

// board is the shell's Host.Scores: one house's board as a player should see it, which is the
// side-car laid over whatever the house file itself carries.
func (a *app) board(h shell.House) house.Scores {
	b, notes := a.store.Load(h.Name, h.Scores)
	a.reportScoreNotes(h.Name, notes)
	return b
}

// savesNone is -saves' opt-out, spelled like -prefs none and -scores none.
const savesNone = "none"

// openSaves decides where the games in progress live, on openScores' pattern and with its
// reasoning: a failure to resolve the directory is reported at startup rather than at the
// moment somebody presses S, because that moment is the one where being told "there is
// nowhere to put this" is least useful.
func (a *app) openSaves() {
	switch a.o.savesDir {
	case savesNone:
		return
	case "":
		st, err := saved.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "glidergo: games cannot be saved: %v\n", err)
			return
		}
		a.saves = st
	default:
		a.saves = saved.OpenDir(a.o.savesDir)
	}
}

// savedGame is the shell's Host.Saved: what the "Open Saved Game..." row would resume, or
// why there is nothing to resume.
//
// Two sources, in this order, and the order is the whole of the policy: **this
// installation's save wins over the one the house shipped with.** A player's own game is the
// one they mean, and a house's embedded block does not change, so a save laid over it is
// always the newer of the two. The house's block stays where it is either way -- nothing
// here writes a house file -- so deleting a save brings it back.
//
// The error from the store is the one that reaches the menu, not the house's: "no saved game
// for Slumberland" is the answer when neither source has anything, and it is the store's
// wording because the store is where a save the player makes will go.
func (a *app) savedGame(h shell.House) (saved.Info, error) {
	info, err := a.saves.Peek(h.Name)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return info, err
	}

	// No save of our own. PeekFile reads the 866-byte header and no rooms, which is the
	// same read the picker already does for every house it lists, so the cost of asking is
	// one header. A house that will not peek is not reported here: the row's job is to say
	// whether there is a game to resume, and a house that cannot be read at all is a
	// problem the moment it is *played*, with a better message than this row has room for.
	housesFS, _ := a.o.housesRoot()
	sum, perr := libraryHouse(housesFS, h).peek()
	if perr != nil || !sum.HasGame {
		return saved.Info{}, err
	}
	from := saved.InfoOf(house.EmbeddedGame(h.Name, sum.TimeStamp, sum.Game))
	from.FromHouse = true
	return from, nil
}

// savedGameFor is the game a resume starts from, in savedGame's order: this installation's
// save, then the house's own block. It takes the loaded house rather than a path because
// every gate but the first needs it.
//
// The distinction from savedGame is what each one costs. That one reads a header for a menu
// row; this one reads the whole save -- every room's object state -- because it is about to
// be applied to a house.
func (a *app) savedGameFor(name string, h *house.House) (*house.SavedGame, error) {
	sg, err := a.saves.Load(name)
	if err == nil {
		return sg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if h.HasGame == 0 {
		// os.ErrNotExist is wrapped rather than replaced: a caller that wants to offer
		// "start a new game instead" can still tell this apart from a broken save, and the
		// text is what a player needs to read.
		return nil, fmt.Errorf("%s has no saved game to resume: %w", name, os.ErrNotExist)
	}
	return house.EmbeddedGame(name, h.TimeStamp, h.SavedGame), nil
}

// openWindow opens the one window. The surfaces inside are always 640x480 -- the
// game's own screen -- and the magnification is the backend's business
// (docs/IMPROVEMENTS.md 2.8).
//
// The scale comes from the settings rather than from -scale, because -scale is only one of
// the ways it can be set; overrideFromFlags has already folded the flag into them. It is
// read once, here, which is why the settings screen's magnification row says "next launch":
// resizing a window mid-session is a backend change (2.8) and not a preference change.
func (a *app) openWindow(title string) error {
	view := render.DefaultView()
	win, err := backend.Open(platform.Config{
		Title:  title,
		Width:  int(view.Screen.Wide()),
		Height: int(view.Screen.Tall()),
		Scale:  a.p.Scale,
	})
	if err != nil {
		return err
	}
	a.win = win
	a.fb = platform.NewFramebuffer(int(view.Screen.Wide()), int(view.Screen.Tall()))
	return nil
}

func (a *app) close() {
	if a.pump != nil {
		a.pump.Close()
	}
	if a.win != nil {
		a.win.Close()
	}
}

// openAudio is InitSound and InitMusic (Sound.c:438-474, Music.c:312-369), which in
// the original run once each at launch (Main.c:337) and are the reason a Mac that
// could not spare the memory played the whole game in silence rather than refusing
// to start.
//
// Every failure here takes that same path -- a line on stderr and a silent game,
// never a returned error -- with one exception: a player named on the command line
// and not installed *is* an error, because the flag exists for somebody diagnosing
// one specific player and quietly using a different one would waste their afternoon.
func (a *app) openAudio() error {
	o := a.o
	if !o.sound {
		return nil
	}
	soundFS, soundName := o.soundRoot()
	bank, err := audio.LoadBank(soundFS)
	if err != nil {
		// The bank is normally the one inside the executable, so the usual cause is a bad
		// -sounds path; in a working tree it is a `make clean-assets` that `make assets`
		// puts back. The game is fully playable without it.
		if soundName == "" {
			fmt.Fprintf(os.Stderr, "glidergo: no sound: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "glidergo: no sound from %s: %v\n", soundName, err)
		}
		return nil
	}

	sink, where, err := openSink(o.audioOut, o.wav)
	if err != nil {
		if o.audioOut != "" {
			return err
		}
		fmt.Fprintf(os.Stderr, "glidergo: %v\n", err)
	}

	a.bank, a.sink, a.where = bank, sink, where
	a.eng = audio.New(bank)
	// The player's volume, and `isSoundOn` with it. The bank is loaded either way: a
	// volume of zero is a mute the settings screen can undo mid-session, so a run that
	// started silent must still have the samples in hand.
	a.eng.SetVolume(int16(a.p.Volume))
	a.eng.SetSoundOn(a.p.Sound)
	a.pump = audio.NewPump(a.eng, sink)

	if !o.quiet {
		fmt.Printf("glidergo: audio=%s %d sounds + %d music = %d KiB, rate %d Hz, volume %d/%d\n",
			where, audio.TriggerSlot, audio.MaxMusic, bank.Bytes()/1024,
			audio.Rate, a.p.Volume, audio.FullVolume)
	}
	return nil
}

// bindAudio points the engine at a new World and loads the house's own sounds.
//
// The house half of the resource-fork swap done for art in play: for as long as a
// house is open its own 'snd ' resources are the ones GetResource finds. Thirteen of
// the twenty-two shipped houses have any and twelve have one that can be read (five
// resources are MACE 6:1 compressed), and a house with none is not an error -- it is
// a house whose sound triggers get no hot spot at all, which is the C's behaviour
// and is why this runs before the first room is composed.
func (a *app) bindAudio(w *game.World, name string) {
	if a.eng == nil {
		return
	}
	if err := a.bank.LoadHouse(name); err != nil {
		fmt.Fprintf(os.Stderr, "glidergo: %s: no custom sounds: %v\n", name, err)
	}

	// The three requests the game makes of the mixer, and the one it makes of the
	// score. All four are documented on their fields in internal/game/world.go;
	// TriggerSoundExists is the one that is not just sound, because whether a sound
	// trigger loads decides whether the room gets a kSoundIt hot spot at all -- so a
	// house with custom sounds composes differently with audio than without it.
	w.SoundPlayer = a.eng.PlayPrioritySound
	w.TriggerSoundExists = a.eng.LoadTriggerSound
	w.FlushTriggerSound = a.eng.FlushTriggerSound
	w.Music = a.eng

	// The one call that goes the other way: the mixer asks the game which piece of
	// the score is next, from inside Mix, on this goroutine. See
	// internal/audio/music.go for why the score walk stays on the game's side.
	a.eng.NextPiece = w.NextMusicPiece

	// The two music preferences, each from its own setting now. They are separate in the
	// original because a player can want the score during a game and not on the title
	// screen; -music is the one switch that covers both, for a run that wants silence.
	// MusicOnTitle is set here as well as on the title screen's own World, because it is
	// this World's teardown that starts the idle score when the game ends (NewGame,
	// internal/game/play.go) and music.go's adoptScore is what carries the cursor across.
	w.PlayMusicGame = a.p.MusicInGame
	w.PlayMusicIdle = a.p.MusicOnTitle
	w.InitMusic()
}

// play runs one game and returns its result. An error means the game never started
// -- the house would not load, the art is unreadable, the room number is out of
// range, the saved game does not belong to this house -- and the shell shows it and stays
// up, because the player's next move is to choose a different house
// (docs/IMPROVEMENTS.md 2.33).
func (a *app) play(ref houseRef, two, resume bool) (shell.Outcome, error) {
	o := a.o
	name := ref.Name

	h, err := ref.open()
	if err != nil {
		return shell.Outcome{}, err
	}
	if len(h.Rooms) == 0 {
		return shell.Outcome{}, fmt.Errorf("%s: house has no rooms", ref.Path)
	}

	// The saved game, read and validated before anything is built.
	//
	// Both halves are deliberately here rather than after the World exists. internal/saved's
	// Check is OpenSavedGame's four gates plus one, and the C runs them *after* it has begun
	// tearing the world down for a resume -- so a mismatched save there left the player
	// looking at a yellow alert and a half-started game (see game.ResumeSavedGame's note on
	// kYellowIllegalRoomNum). Failing before the window's title changes means a refused
	// resume is a message on the title screen's status band with the menu still under it.
	var sg *house.SavedGame
	if resume {
		if sg, err = a.savedGameFor(name, h); err != nil {
			return shell.Outcome{}, err
		}
		if err := saved.Check(sg, name, h); err != nil {
			return shell.Outcome{}, err
		}
	}
	if o.roomNum >= 0 {
		if o.roomNum >= len(h.Rooms) {
			return shell.Outcome{}, fmt.Errorf("-room %d out of range (%s has %d)",
				o.roomNum, name, len(h.Rooms))
		}
		// The house's authored start room, overridden before NewGame reads it.
		// Cleaner than reaching past SetHouseToFirstRoom, and it is exactly what the
		// editor's "set as first room" command writes.
		h.FirstRoom = int16(o.roomNum)
	}

	artFS, _ := o.artRoot()
	assets := render.NewAssets(artFS)
	// The house's own resource fork shadows the application's for as long as the
	// house is open, which is what HouseIO.c does. Without it every custom
	// background in the house falls back to PICT 2000 and half the shipped houses
	// look wrong.
	houseArtFS, houseArtName := o.houseArtRoot()
	fork := assetfs.Name(houseArtName, name)
	if assetfs.IsDir(houseArtFS, name) {
		assets.OpenHouseResFork(fork, assetfs.Sub(houseArtFS, name))
	} else if !o.quiet {
		fmt.Fprintf(os.Stderr, "glidergo: no extracted resource fork at %s; custom art will fall back\n", fork)
	}

	view := render.DefaultView()
	scene := render.NewScene(view, assets, h)
	scene.NumNeighbors = a.p.Neighbors
	scene.Clock = time.Now()

	// The stream is the process's, not this game's: it was resolved at startup by
	// resolveSeed and the last game handed back where it had got to. See app.randSeed.
	seed := a.randSeed
	w := game.NewWorld(h, scene, seed)
	w.TwoPlayer = two

	// The house's name, which the World has no other way to learn: houseType has no name
	// field and never did (a Mac document was named by its file), so a save has to be told
	// what to write into its own header. Set for every game and not only a resumed one,
	// because it is what a save made *during* this game will carry.
	w.HouseName = name

	// Where a save goes, and the one thing the game does not decide. A nil hook is a build
	// with nowhere to write: CanSaveGame reads it, so the pause hint below stops offering a
	// key that could not work.
	//
	// The error is kept rather than returned, because DoSaveGame's caller is a keystroke
	// inside the pause loop and there is nothing there to return to. The pause hint is where
	// it surfaces -- the one line a paused player is already reading.
	var saveErr error
	if a.saves != nil {
		w.SaveGame = func(sg *house.SavedGame) error {
			saveErr = a.saves.Save(sg)
			if saveErr != nil {
				fmt.Fprintf(os.Stderr, "glidergo: %v\n", saveErr)
			}
			return saveErr
		}
	}

	// The saved game itself, applied to the World and to the house's rooms before NewGame
	// reads either. Everything that could refuse has already refused above; this cannot
	// fail for a reason the player did not already see, and is checked because a resume
	// that silently did not happen would start a new game with the save's own score line.
	if resume {
		if err := w.ResumeSavedGame(sg); err != nil {
			return shell.Outcome{}, err
		}
	}

	// **DoBackground is true here and false in the original** (Main.c:186), and it is
	// *not* the preference it looks like. It decides whether PlayGame pumps host events
	// at all, and in a port that is not optional: with it false the window never sees a
	// keystroke, never repaints on exposure and never notices that it lost the
	// foreground. The original could get away with it because the Toolbox drew the
	// window's contents from the WindowRecord; a modern compositor cannot.
	//
	// The *player's* half of the original's flag -- "keep playing while switched out" --
	// is prefs.PauseWhenUnfocused, and it gates the Suspend call in the focus arm below
	// instead. That is the split docs/IMPROVEMENTS.md 2.21 asks for: the pump is the
	// port's business and the pausing is the player's.
	w.DoBackground = true

	// Which placard the pause draws, and what the placard cannot say for itself. Both
	// pictures read "or Cmd-Q to Quit the game" and there is no Command key here, so the
	// hint is the substitute -- see internal/game/pause.go. It is also the confirmation
	// docs/IMPROVEMENTS.md 2.7 asks for: Escape pauses rather than quitting, so the only way
	// to throw a game away is to read this line first.
	w.EscPause = a.p.EscPause()

	// The hint's resting text, which is also the menu of keys the pause accepts. It names S
	// only when a save would actually happen: CanSaveGame is false for a two-player game (one
	// glider's worth of fields, see game/savegame.go) and for a build with nowhere to write,
	// and a line that offered a key which silently did nothing would be worse than one that
	// never mentioned it. The arms below replace this line while the pause is up and put it
	// back when the pause ends.
	baseHint := "no Command key here -- press Q to give up the game"
	if w.CanSaveGame() {
		baseHint = "no Command key here -- press S to save the game, Q to give it up"
	}
	w.PauseHint = baseHint

	w.Fix = gameFixes(a.p.Fixes)

	a.bindAudio(w, name)
	if a.win != nil {
		a.win.SetTitle("gliderGo -- " + name)
	}

	// ---- the five host hooks --------------------------------------------------
	//
	// This is the entire seam between the game and the machine. Each hook is
	// documented on its field in internal/game/world.go; what follows is only this
	// host's answer.

	// The 60.15 Hz Mac tick. The original's clock is the Time Manager's, incremented
	// by the vertical retrace; wall-clock milliseconds scaled by 60/1000.66 is the
	// same thing to within a tick, and the game measures nothing in absolute time.
	start := time.Now()
	if !o.bench {
		w.TickCount = func() int64 {
			return int64(time.Since(start).Seconds() * 60.15)
		}

		// The frame limiter's loop body. The original's is empty -- a busy-wait that
		// pins a core for whatever fraction of the two ticks the frame did not need.
		// Sleeping for a tick instead is docs/IMPROVEMENTS.md 2.17, and it is safe
		// because it changes only *how* the wait is spent, not when it ends:
		// awaitFrame still returns on the same tick.
		w.WaitTick = func() { time.Sleep(time.Millisecond) }
	}
	// -bench leaves both hooks nil, which is how the game is told to run flat out:
	// with no tick source awaitFrame skips its wait entirely and Ticks answers from
	// the frame counter, so the *game* still keeps nominal time -- Frame *
	// TicksPerFrame -- while the wall clock is free to run ahead. Nothing about the
	// simulation changes, which is what makes the number it prints a measurement of
	// the composition and blit path rather than of the limiter. It is also exactly
	// the clock the headless build and every test in internal/game uses, so a
	// benchmark and a replay see the same frames.

	w.Present = func() {
		w.Main.ToBGRX(a.fb.Pix, a.fb.Stride)
		if err := a.win.Present(a.fb); err != nil {
			// A failed present is a dead window, and there is nothing useful to do
			// with the error from inside a void hook. Ending the game is the honest
			// response and it goes through the same door the Quit menu item does.
			w.Quitting = true
			w.SwitchedOut = false
		}

		// The mixer's only pacer on the live path, and it is *here* rather than on
		// the frame loop for one reason: Present is called once per frame during play
		// and once per strip during a room wipe -- 116 or 160 times inside a single
		// frame, which is the one part of the game that takes far longer than a frame
		// to draw. A pump driven by the frame counter would starve the player through
		// every door the glider takes. Nil until the bank loads, and safe on a nil
		// receiver.
		a.pump.ClockTick()
	}

	// closed distinguishes the two ways out of a game that look identical to the
	// World. Giving up (Q from a pause) ends the game and gives the shell back; a closed
	// window ends everything, because there is nowhere to draw a title screen. The original
	// needs no such distinction: its Quit is a menu command and its window is the desktop.
	closed := false

	// asking is QuerySaveGame (alert 1041, "Do you want to save the state of the game before
	// quitting?"), asked in the hint row rather than in a dialogue because that row is the
	// only place this port has to say something to a paused player.
	//
	// The alert has two buttons and no Cancel -- "Save First" and "Don't Save" -- because
	// DoCommandKey assigns `playing = false` *before* it asks (Input.c:55-58), so in 1994 the
	// question was only ever about the save and never about the quitting. Y and N are those
	// two buttons. The pause key is a third answer the alert did not have, and it is here
	// because the keystroke that opens the question is different from the one that did in
	// 1994: Command-Q is a chord nobody hits by accident and a bare Q is one letter away from
	// the controls. Resuming is what a player who did not mean it wants, and it costs the
	// faithful path nothing -- neither button moved.
	asking := false

	// The keys, resolved once. Doing it here and not per poll is safe because the
	// settings screen is only reachable from the title screen, so no binding can change
	// between this line and the end of the game -- and a rebind is therefore never half
	// applied. prefs.Controls.Keys is what makes an unbound control a dead key rather
	// than a wrong one.
	keys1, keys2 := a.p.Player1.Keys(), a.p.Player2.Keys()

	// Two keys pause, and only one of them is a preference.
	//
	// The pause key is the player's, tab or escape -- the original's isEscPauseKey, which is
	// also what picked the placard above. Escape pauses whether or not it is that key,
	// because Escape used to *end the game here*: one keystroke, no prompt, which is worse
	// than the original, where giving up meant going to a menu (docs/IMPROVEMENTS.md 2.7).
	// Now it asks the question instead, and the pause is where the question is asked -- Q
	// gives up, either key resumes. A stranger's guess at "get me out of here" costs them
	// nothing, and the way out is written on the screen they land on.
	pauseKey := a.p.Pause()
	pauseDown := func() bool {
		return a.win.KeyDown(pauseKey) || a.win.KeyDown(platform.KeyEscape)
	}

	// The event pump. HandlePlayEvent calls this and the game's three arms are
	// RefreshGameWindow, Suspend and Resume; only the *classification* below is ours.
	w.PlayEvent = func() {
		for _, ev := range a.win.PollEvents() {
			switch {
			case ev.Kind == platform.EventQuit:
				closed = true
				// SwitchedOut is cleared alongside Quitting because PlayGame's pump
				// loop spins on SwitchedOut alone -- so a window closed while the
				// game is in the background would otherwise never be noticed. The
				// original has the same hole and cannot fall into it, because its
				// Quit is a menu command and a background application has no menu
				// bar.
				w.Quitting = true
				w.SwitchedOut = false

			case ev.Kind == platform.EventKeyDown && w.Paused && asking:
				// The two buttons of alert 1041, and nothing else: any other key is
				// swallowed here so that a mistyped answer cannot fall through to the Q
				// and S arms below and re-ask a question that is already up. The pause
				// key is not a key event -- pauseDown polls the key state -- so the
				// third answer is unaffected by this arm consuming the keyboard.
				//
				// Both answers end the game, which is the C's ordering and not an
				// oversight of it: `playing = false` is already assigned by the time the
				// alert appears. GiveUpGame is where those two assignments live, and it
				// filters the save through CanSaveGame so that a yes from a game that
				// cannot be saved writes nothing.
				switch ev.Key {
				case platform.KeyY:
					w.GiveUpGame(true)
				case platform.KeyN:
					w.GiveUpGame(false)
				}

			case ev.Kind == platform.EventKeyDown && ev.Key == platform.KeyQ && w.Paused:
				// DoCommandKey's Command-Q arm (Input.c:55-63), which in the original is
				// reachable from inside the pause loop and nowhere else that matters.
				// Plain Q here because the window manager owns Command-Q on every
				// platform this builds for, and only while paused, so that Q stays
				// bindable as a control.
				//
				// With nothing to save the question is skipped entirely and the game ends
				// on this keystroke. That is where the C puts its `!twoPlayerGame &&
				// !demoGoing` guard too -- around the question, not around the save -- so
				// a two-player game has always given up in one key.
				if !w.CanSaveGame() {
					w.GiveUpGame(false)
					break
				}
				asking = true
				w.SetPauseHint("give up: Y saves the game first, N does not, " +
					platform.KeyName(pauseKey) + " keeps playing")

			case ev.Kind == platform.EventKeyDown && ev.Key == platform.KeyS && w.Paused &&
				!ev.Repeat:
				// DoCommandKey's Command-S arm (Input.c:65-71). Everything it does to the
				// screen is DoSaveGame's; what is left here is the sentence afterwards,
				// because the C's save could not fail (it did not happen) and this one can
				// -- a full disk, a read-only home directory. saveErr is the hook's answer
				// and the hint row is where a paused player is already looking.
				//
				// !ev.Repeat because a held S would otherwise write the file once per poll,
				// each time with two presents of the "Saving Game" title over the top of a
				// pause. A press is a save.
				w.DoSaveGame()
				if saveErr != nil {
					w.SetPauseHint("the game could not be saved: " + saveErr.Error())
				} else {
					w.SetPauseHint("game saved -- S again replaces it, Q gives up the game")
				}

			case ev.Kind == platform.EventFocus:
				if ev.Focused {
					w.Resume()
				} else if o.frames == 0 && a.p.PauseWhenUnfocused {
					// **A timed run does not pause.** Suspending on focus loss is
					// the original's behaviour and the right behaviour for a play
					// session, but it makes the loop depend on the window manager:
					// PlayGame spins on this hook while SwitchedOut and presents
					// nothing, so a -frames run can never reach its limit once the
					// window is backgrounded. That is not hypothetical -- on this
					// host the WM hands focus back to the terminal about a second
					// after the window opens, which is why `-frames 300` used to
					// hang at frame 34 while `-frames 30` passed. A timed run is a
					// measurement or a replay, and neither has a user to pause for.
					w.Suspend()
				}

			case ev.Kind == platform.EventExpose:
				// The original's updateEvt arm. It is needed for the same reason it
				// was in 1994 and for one more: a suspended game draws no frames at
				// all, so without this the window keeps whatever the X server
				// happened to retain for as long as the game is paused. See
				// docs/IMPROVEMENTS.md 2.27.
				w.RefreshGameWindow()

			case ev.Kind == platform.EventResize:
				// The backend owns the scale transform; the game's surfaces are
				// always 640x480. See docs/IMPROVEMENTS.md 2.8 on inserting the
				// resize transform at the present step and nowhere else.

			case ev.Kind == platform.EventNone:
			}
		}

		// HandlePlayEvent's `sleep = 2`, and the only place it belongs.
		// WaitNextEvent yields for up to two ticks when the queue is empty; during
		// play that is invisible because awaitFrame is already spending the frame's
		// budget, so reproducing it there would install a second pacer. The one case
		// where it matters is this one: PlayGame spins on this hook while switched
		// out, and without a yield a backgrounded game burns a core.
		if w.SwitchedOut {
			time.Sleep(2 * time.Second / 60)
		}
	}

	// One poll, two gliders. The original snapshots the hardware KeyMap once, inside
	// player one's GetInput, and player two reads that snapshot -- so both calls in a
	// frame see the same keys. Reading the backend's held-key map has the same
	// property for the same reason: it is only updated by PollEvents, which ran once,
	// above.
	//
	// pauseHeld is the exception, and it is the one piece of state this hook keeps. The
	// pause key is reported on its *press edge* rather than while it is held, because
	// DoPause is called from inside GetInput and returns having resumed: a level-triggered
	// bit would pause again on the same press, over and over, which looks exactly like a
	// pause key that does not work. internal/game/pause.go's file comment has the C's
	// version of this argument -- the three GetKeys release loops -- and why one edge is
	// the whole of it here.
	pauseHeld := false

	w.KeyPoll = func(g *player.Glider) player.Keys {
		if g.Which == player.Player2 {
			return player.Keys{
				Left:  a.win.KeyDown(keys2.Left),
				Right: a.win.KeyDown(keys2.Right),
				Batt:  a.win.KeyDown(keys2.Batt),
				Band:  a.win.KeyDown(keys2.Band),

				// Delete is reported for player two too, and the *game* decides
				// whether it counts: GetInput's `which == kPlayer1` gate is the
				// original's, and prefs.Fixes.Player2GiveUp is what drops it (1.9,
				// docs/IMPROVEMENTS.md 2.23). Reporting it unconditionally keeps one
				// decision in one place -- a host that filtered the key here as well
				// would make the setting depend on two files agreeing.
				//
				// Player two still has no Command and no Pause, and those are not
				// oversights: there is no Command key on this host at all, and a
				// second pause poll would pause twice on one press, because
				// KeyPoll's edge detector is a single variable. See
				// docs/analysis/input.md 10.3.
				Delete: a.win.KeyDown(platform.KeyDelete),
			}
		}

		down := pauseDown()
		edge := down && !pauseHeld
		pauseHeld = down

		return player.Keys{
			Left:   a.win.KeyDown(keys1.Left),
			Right:  a.win.KeyDown(keys1.Right),
			Batt:   a.win.KeyDown(keys1.Batt),
			Band:   a.win.KeyDown(keys1.Band),
			Delete: a.win.KeyDown(platform.KeyDelete),
			Pause:  edge,

			// Command stays false on every host this port has. The two chords the
			// original watches for arrive by other doors now -- Command-Q as the window's
			// quit event, and the pause loop's give-up as plain Q -- so DoCommandKey is
			// never called. internal/game/env.go says the rest.
			Command: false,
		}
	}

	// The pause loop: Input.c:89-116 with the spinning replaced by a pump.
	//
	// DoPause draws the placard and hands this the job of waiting, which is the only part
	// a Macintosh could do by polling the keyboard flat out and a windowed program cannot.
	// The three things it owes the game are in World.Pause: pump, paint once per pass
	// afterwards, and return only when the pause is over.
	//
	// The C's three release loops collapse into `held`. A pause begins on a press edge, so
	// the key is down when this is entered; the first release clears held, and the next
	// press resumes. That is loops (A) and (B), and loop (C) -- the C's wait for the
	// resuming press to be released -- is unnecessary because KeyPoll is edge-triggered
	// too, so a key still held on the way out cannot pause again.
	w.Pause = func(paint func()) {
		// Every pause starts from the resting hint and no question outstanding, however the
		// last one ended. Without this a player who gave up, was asked, resumed with the
		// pause key and paused again would find the question still on the screen and Y still
		// live -- a keystroke they answered a minute ago, waiting for them.
		defer func() {
			asking = false
			w.SetPauseHint(baseHint)
		}()

		held := true
		for {
			// The same pump the frame loop uses, so that a pause answers an expose, a
			// focus change, a closed window and the give-up key exactly as play does.
			// Nothing else may consume events: the Q arm above is inside this call.
			w.PlayEvent()

			if down := pauseDown(); !down {
				held = false
			} else if !held {
				return
			}
			// Quitting is the window or the give-up key; Paused going false by itself is
			// what 1.10's save-and-quit will do, and is the C's `paused = false` from
			// inside DoCommandKey.
			if w.Quitting || !w.Paused {
				return
			}

			paint()
			time.Sleep(2 * time.Second / 60)
		}
	}

	// The blocking waits: WaitForInputEvent (Utilities.c:439-478) and Delay, which the
	// banner, the stars-remaining panel and both game-over animations are paced by.
	//
	// Installed only for a session with somebody in front of it, and for the same reason
	// the high scores are: a measurement or a replay must not spend fifteen real seconds
	// looking at a banner. A nil hook is "the deadline expired immediately"
	// (internal/game/wait.go), so -frames, -bench and -dump runs draw every pixel these
	// screens draw and none of their duration -- which is what makes a game that ends in
	// death byte-comparable between this host and the headless one.
	//
	// Three differences from the C, all of them the host's half of wait.go's list:
	//
	//   - **A held modifier does not end a wait.** The C tests the key map, so a finger
	//     resting on Shift skips the banner and both endings instantly. This takes a key
	//     press, which is what a screen the player is meant to read wants.
	//   - **The window keeps repainting.** The C dequeues an update event and drops it
	//     without answering it; this re-presents Main every pass, which the port can do
	//     and 1994 could not because there was no Main to present. Note that it is
	//     emphatically *not* RefreshGameWindow -- that rebuilds the screen from the work
	//     map, which is exactly what these four screens have drawn over.
	//   - **A suspend does not suspend the game.** The C's osEvt arm calls InitCursor and
	//     nothing else -- no music toggle, no SwitchedOut -- so a wait crossed by a
	//     switch out keeps its clock and its sound. Transcribed as-is: World.Suspend here
	//     would stop the mixer and, worse, park the game in a state PlayGame's pump loop
	//     is the only thing that clears.
	//
	// A resume *does* end the wait and is the one thing the return value carries, because
	// DisplayStarsRemaining reads it to decide whether to rebuild the screen. It is
	// gated on having seen the matching focus loss: the C cannot receive a resume without
	// a suspend before it and an X11 window manager can hand out focus twice.
	if o.frames == 0 && !o.bench && o.dump == "" {
		w.Wait = func(ticks int64, discard bool) game.Waited {
			deadline := time.Now().Add(time.Duration(ticks) * tick)
			lostFocus := false
			var r game.Waited

			// At least one pass, always: FlushEvents is a zero-tick discarding wait and
			// draining the queue is the whole of what it is for.
			for {
				for _, ev := range a.win.PollEvents() {
					switch {
					case ev.Kind == platform.EventQuit:
						// The same door PlayEvent's arm uses. Nothing in the game reads
						// Quitting out of a Waited, so it is set directly.
						closed = true
						w.Quitting = true
						w.SwitchedOut = false
						return r

					case ev.Kind == platform.EventKeyDown && !ev.Repeat:
						// `theEvent.what == keyDown`, minus the mouse (this port reports
						// no pointer events) and minus auto-repeat, which the Mac's queue
						// would have coalesced anyway. A discarding wait swallows it: see
						// DelayTicks on why the panel's first second is not skippable.
						if !discard {
							r.Input = true
						}

					case ev.Kind == platform.EventFocus:
						if !ev.Focused {
							lostFocus = true
						} else if lostFocus && !discard {
							r.Resumed = true
						}

					case ev.Kind == platform.EventExpose, ev.Kind == platform.EventResize,
						ev.Kind == platform.EventNone:
						// Nothing to do for any of them. The present below repaints the
						// window whether it was damaged or not, and the backend owns the
						// scale transform.
					}
				}

				a.present(w)
				if r.Input || r.Resumed || w.Quitting {
					return r
				}
				left := time.Until(deadline)
				if left <= 0 {
					return r
				}
				// HandlePlayEvent's `sleep = 2`, in slices, so that the deadline is
				// honoured to within a poll rather than overshot by one.
				if left > pollWait {
					left = pollWait
				}
				time.Sleep(left)
			}
		}
	}

	// The high scores, and only for a session with somebody in front of it.
	//
	// A measurement is not offered a board: the two dialogs block until Okay is pressed and
	// there is no Cancel in either of them (docs/analysis/scoring.md 7.11.1), so a -frames
	// run that happened to die would hang forever waiting for a name, and one that got past
	// that would sit on the board for thirty seconds. Both are the reason
	// World.TestHighScore treats a nil hook as "does not qualify" rather than as an error:
	// a replay's outcome must not depend on the machine it runs on. `make headless` covers
	// the screen through -shot instead.
	if o.frames == 0 && !o.bench && o.dump == "" {
		w.HighScore = a.highScoreHook(w, name, assets, &closed)
	}

	// -frames is the headless and timed path: a hook on the frame limiter is the only
	// place that can end a game from outside without inventing a key. It sets the
	// same flag the Quit menu item does.
	if o.frames > 0 {
		w.Present = wrapPresentLimit(w, int64(o.frames))
	}

	// ---- play -----------------------------------------------------------------

	if !o.quiet {
		fmt.Printf("glidergo: %s -- %d rooms, first %d, %d stars\n",
			name, len(h.Rooms), h.FirstRoom, w.CountStarsInHouse())
		// The version is on this line because it is the line a bug report quotes
		// (docs/IMPROVEMENTS.md 4.2), and a report that does not say which build it came
		// from costs a round trip before anything can be looked at. The title screen
		// shows the same string on its status band, for a player who never sees stdout.
		fmt.Printf("glidergo: version=%s backend=%s surface=%dx%d scale=%d neighbors=%d seed=%d\n",
			version, backend.Name, a.fb.W, a.fb.H, a.p.Scale, a.p.Neighbors, seed)
	}

	// kNewGameMode or kResumeGameMode (Play.c:100-190), which is the only difference a resume
	// makes to the game after ResumeSavedGame has installed the fields. The mode decides three
	// things inside NewGame: where the glider begins (WhereDoesGliderBegin reads the saved
	// `where` instead of the first room's start), whether the score and the inventory are
	// zeroed, and which of the two opening screens is shown -- a new game gets the author's
	// banner, a resumed one gets DisplayStarsRemaining, which is the count of stars still to
	// find and therefore the one thing a returning player needs.
	mode := game.NewGameMode
	if resume {
		mode = game.ResumeGameMode
	}
	w.NewGame(mode)

	// NewGame's teardown has just started the idle score on this World, which is about to
	// go out of scope. Hand its place in the score to the title screen's cursor before it
	// does, so the score carries on across the transition the way one global would. See
	// music.go's adoptScore.
	a.adoptScore(w)

	// And the same hand-off for the random stream, which is a QuickDraw global in the
	// original and so spans games in one process. Read back after NewGame returns rather
	// than while the game runs, because that is the one moment the World is finished with
	// it; a game that never started drew nothing and leaves the stream where it was.
	a.randSeed = w.RandSeed

	if !o.quiet {
		el := time.Since(start)
		rate := float64(w.Frame) / el.Seconds()
		fmt.Printf("glidergo: %d frames in %v (%.1f fps), score %d, %d stars left\n",
			w.Frame, el.Round(time.Millisecond), rate, w.Score, w.StarsLeft)
		if o.bench {
			// The original's target, for something to compare against:
			// kTicksPerFrame is 2 on a 60.15 Hz clock, so a Mac that kept up ran at
			// 30.07 frames a second.
			fmt.Printf("glidergo: unpaced -- %.1fx the original's 30.07 fps target\n", rate/30.07)
		}
		reportAudio(a.eng, a.pump, a.sink)
	}

	// Sticky asset errors, collected rather than raised: the draw helpers are void in
	// the original and a missing PICT there ends in RedAlert, so this is the port's
	// equivalent -- reported once, at the end, with the score intact.
	if err := assets.Err(); err != nil && a.artErr == nil {
		a.artErr = err
	}

	return shell.Outcome{
		Score:     w.Score,
		StarsLeft: w.StarsLeft,
		Frames:    w.Frame,
		Closed:    closed,
	}, nil
}

// wrapPresentLimit ends the game once the simulation has run n frames.
//
// Present is the sampling point and World.Frame is the quantity, and they are not
// the same number: **Present is not called once per frame.** NewGame's DumpScreenOn
// presents before the loop starts, HideGlider presents on its own, and a room
// transition presents once per wipe strip -- 116 or 160 times inside a single frame.
// This used to count its own calls, and the arithmetic showed: `-frames 300`
// reported 298 in a static room and 137 in a run where the glider happened to take a
// door, because one 160-strip wipe spent more than half the budget. Counting
// World.Frame makes the flag mean what it says, makes two runs of the same house
// comparable, and makes the fps line above -- w.Frame over elapsed -- a rate of
// frames rather than of blits. It is the same mistake, and the same fix, as
// docs/IMPROVEMENTS.md 2.4 in the test harness.
//
// One consequence worth stating: with -dump the PNG count is still the *present*
// count, so a run that crosses a room boundary writes a file per wipe strip. That is
// the useful behaviour for looking at a transition frame by frame, and it is why the
// limit lives here rather than in the dumper.
func wrapPresentLimit(w *game.World, n int64) func() {
	prev := w.Present
	return func() {
		if prev != nil {
			prev()
		}
		if w.Frame >= n {
			w.Quitting = true
			w.SwitchedOut = false
		}
	}
}
