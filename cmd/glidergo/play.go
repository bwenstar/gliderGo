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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"glidergo/internal/audio"
	"glidergo/internal/game"
	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/platform"
	"glidergo/internal/platform/backend"
	"glidergo/internal/render"
	"glidergo/internal/shell"
)

// app is the host: the things that outlive one game.
type app struct {
	o   *options
	win platform.Window
	fb  *platform.Framebuffer

	bank  *audio.Bank
	eng   *audio.Engine
	pump  *audio.Pump
	sink  audio.Sink
	where string // one word for the startup line: which player, or "none"

	// artErr is the first sticky asset error from any game this session. It is
	// reported rather than returned mid-session, because losing a finished game's
	// score to a missing PICT would be a worse trade than a line on stderr.
	artErr error
}

func newApp(o *options) *app { return &app{o: o} }

// openWindow opens the one window. The surfaces inside are always 640x480 -- the
// game's own screen -- and -scale is the backend's business (docs/IMPROVEMENTS.md 2.8).
func (a *app) openWindow(title string) error {
	view := render.DefaultView()
	win, err := backend.Open(platform.Config{
		Title:  title,
		Width:  int(view.Screen.Wide()),
		Height: int(view.Screen.Tall()),
		Scale:  a.o.scale,
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
	bank, err := audio.LoadBank(o.sounds)
	if err != nil {
		// The usual cause is a checkout with no assets: assets/extracted is
		// gitignored, being reproducible from GliderPRO/, so `make assets` is the
		// fix. The game is fully playable without it.
		fmt.Fprintf(os.Stderr, "glidergo: no sound: %v\n", err)
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
	a.eng.SetVolume(int16(o.volume))
	a.pump = audio.NewPump(a.eng, sink)

	if !o.quiet {
		fmt.Printf("glidergo: audio=%s %d sounds + %d music = %d KiB, rate %d Hz, volume %d/%d\n",
			where, audio.TriggerSlot, audio.MaxMusic, bank.Bytes()/1024,
			audio.Rate, o.volume, audio.FullVolume)
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

	// One flag for both preferences. The original has two -- music in a game and
	// music on the splash screen -- and they are separate because a player can want
	// one and not the other. The splash screen is silent in this build either way:
	// see the note on shellHost in main.go, and docs/IMPROVEMENTS.md.
	w.PlayMusicGame = a.o.music
	w.PlayMusicIdle = a.o.music
	w.InitMusic()
}

// play runs one game and returns its result. An error means the game never started
// -- the house would not load, the art is unreadable, the room number is out of
// range -- and the shell shows it and stays up, because the player's next move is to
// choose a different house (docs/IMPROVEMENTS.md 2.33).
func (a *app) play(name, path string, two bool) (shell.Outcome, error) {
	o := a.o

	h, err := house.LoadFile(path)
	if err != nil {
		return shell.Outcome{}, err
	}
	if len(h.Rooms) == 0 {
		return shell.Outcome{}, fmt.Errorf("%s: house has no rooms", path)
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

	assets := render.NewAssets(o.artDir)
	// The house's own resource fork shadows the application's for as long as the
	// house is open, which is what HouseIO.c does. Without it every custom
	// background in the house falls back to PICT 2000 and half the shipped houses
	// look wrong.
	fork := filepath.Join(o.houseArt, name)
	if st, err := os.Stat(fork); err == nil && st.IsDir() {
		assets.OpenHouseResFork(fork)
	} else if !o.quiet {
		fmt.Fprintf(os.Stderr, "glidergo: no extracted resource fork at %s; custom art will fall back\n", fork)
	}

	view := render.DefaultView()
	scene := render.NewScene(view, assets, h)
	scene.NumNeighbors = o.neighbors
	scene.Clock = time.Now()

	// A zero seed means "use the clock", which is what the original does:
	// InitializeRandom seeds from the time at launch. A fixed seed makes a whole run
	// reproducible, which is what 1.8's replay tests will want.
	seed := int32(o.seed)
	if seed == 0 {
		seed = int32(time.Now().UnixNano())
	}
	w := game.NewWorld(h, scene, seed)
	w.TwoPlayer = two

	// **DoBackground is true here and false in the original** (Main.c:186). It is the
	// preference that decides whether PlayGame pumps host events at all, and in a
	// port it is not optional: with it false the window never sees a keystroke, never
	// repaints on exposure and never notices that it lost the foreground. The
	// original could get away with it because the Toolbox drew the window's contents
	// from the WindowRecord; a modern compositor cannot. docs/IMPROVEMENTS.md 2.21 is
	// the longer form of this: a released build should always pause on focus loss and
	// should not offer the choice.
	w.DoBackground = true

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
	// World. Escape ends the game and gives the shell back; a closed window ends
	// everything, because there is nowhere to draw a title screen. The original needs
	// no such distinction: its Quit is a menu command and its window is the desktop.
	closed := false

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

			case ev.Kind == platform.EventKeyDown && ev.Key == platform.KeyEscape:
				w.Quitting = true
				w.SwitchedOut = false

			case ev.Kind == platform.EventFocus:
				if ev.Focused {
					w.Resume()
				} else if o.frames == 0 {
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
	w.KeyPoll = func(g *player.Glider) player.Keys {
		if g.Which == player.Player2 {
			return player.Keys{
				Left:  a.win.KeyDown(platform.KeyA),
				Right: a.win.KeyDown(platform.KeyD),
				Batt:  a.win.KeyDown(platform.KeyS),
				Band:  a.win.KeyDown(platform.KeyW),
				// No Command, no Delete and no Pause. Player two has no give-up key
				// and no way to reach the menus in the original either, which is one
				// of the two-player asymmetries docs/IMPROVEMENTS.md 2.23 asks 1.9
				// to fix.
			}
		}
		return player.Keys{
			Left:    a.win.KeyDown(platform.KeyLeft),
			Right:   a.win.KeyDown(platform.KeyRight),
			Batt:    a.win.KeyDown(platform.KeyDown),
			Band:    a.win.KeyDown(platform.KeyUp),
			Delete:  a.win.KeyDown(platform.KeyDelete),
			Pause:   a.win.KeyDown(platform.KeyTab),
			Command: false, // 1.7b's pause and command keys; DoCommandKey is a stub
		}
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
			version, backend.Name, a.fb.W, a.fb.H, o.scale, o.neighbors, seed)
	}

	w.NewGame(game.NewGameMode)

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
