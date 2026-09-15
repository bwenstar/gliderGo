// Command glidergo is the game.
//
// It is the whole of the port's host layer: it loads a house and the extracted art,
// builds a World, hangs the five host hooks off it, and calls NewGame. Everything after
// that call happens inside internal/game, which is where the 1994 code lives; this file
// contains no game logic and is the place to look for anything that is about *this*
// machine rather than about Glider PRO.
//
//	go run ./cmd/glidergo                          # Slumberland, the original's default
//	go run ./cmd/glidergo -house "Demo House"      # any house in assets/extracted/houses
//	go run ./cmd/glidergo -scale 2                 # 2x nearest-neighbour magnification
//	go run ./cmd/glidergo -room 12 -neighbors 3    # start elsewhere, smaller view
//	go run ./cmd/glidergo -two                     # two gliders on one keyboard
//	go run ./cmd/glidergo -bench                   # 300 frames unpaced, report the rate
//	go run -tags nullbackend ./cmd/glidergo -frames 300 -dump /tmp/f   # headless
//
// Keys. Player one has the four arrows, as the original does: left and right to steer, up
// to fire a rubber band, down for the battery. Tab pauses -- which is the original's
// default, `isEscPauseKey` being false at Main.c:184 -- and Delete abandons a glider that
// is waiting in limbo for the other player. Escape quits.
//
// Player two has A and D to steer, W for bands and S for the battery. **That is a
// deliberate departure.** The original binds player two to Control, Command, Option and
// Shift (InterfaceInit.c:148-151), which a modern window manager intercepts before the
// application sees it and which many keyboards cannot report independently. The bindings
// are per-glider data in player.Glider precisely so that 1.7's settings screen can make
// all eight of them the player's choice; see docs/IMPROVEMENTS.md 2.3.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"glidergo/internal/game"
	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/platform"
	"glidergo/internal/platform/backend"
	"glidergo/internal/render"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "glidergo: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		houseName = flag.String("house", "Slumberland", "house to play, by name or by path")
		houses    = flag.String("houses", "assets/extracted/houses", "directory of extracted .house files")
		artDir    = flag.String("art", "assets/extracted/art", "extracted application art tree")
		houseArt  = flag.String("houseart", "assets/extracted/houseart", "extracted per-house resource forks")
		roomNum   = flag.Int("room", -1, "start in this room number instead of the house's first")
		neighbors = flag.Int("neighbors", 9, "how much of the house to compose around the player: 1, 3 or 9")
		scale     = flag.Int("scale", 1, "integer nearest-neighbour magnification of the 640x480 image")
		two       = flag.Bool("two", false, "two players on one keyboard (the shell around this is stage 1.9)")
		seed      = flag.Int64("seed", 0, "fix the random stream for a reproducible run (0 = use the clock)")
		frames    = flag.Int("frames", 0, "quit after N frames, for headless and timed runs (0 = play)")
		bench     = flag.Bool("bench", false, "run with no frame pacing and report the rate the machine sustains")
		dump      = flag.String("dump", "", "with -tags nullbackend, write each frame as a PNG into this directory")
		quiet     = flag.Bool("quiet", false, "do not print the startup and shutdown summaries")
	)
	flag.Parse()

	if *scale < 1 {
		return fmt.Errorf("-scale must be at least 1")
	}
	switch *neighbors {
	case 1, 3, 9:
	default:
		return fmt.Errorf("-neighbors must be 1, 3 or 9")
	}
	if *dump != "" {
		os.Setenv("GLIDERGO_FRAMEDUMP", *dump)
	}
	if *bench && *frames == 0 {
		// An unpaced game with no end is not a benchmark and not playable either -- the
		// glider crosses the room in a few milliseconds. Three hundred frames is ten
		// seconds of game time, which is long enough for the rate to settle.
		*frames = 300
	}

	// ---- the house and its art ------------------------------------------------

	path := *houseName
	if filepath.Ext(path) != ".house" {
		path = filepath.Join(*houses, path+".house")
	}
	h, err := house.LoadFile(path)
	if err != nil {
		return err
	}
	if len(h.Rooms) == 0 {
		return fmt.Errorf("%s: house has no rooms", path)
	}

	// The house's name is its file name: houseType has no name field, because on a Mac the
	// document's name was the file's. Every place the original shows a house name it is
	// reading the FSSpec.
	name := strings.TrimSuffix(filepath.Base(path), ".house")

	assets := render.NewAssets(*artDir)
	// The house's own resource fork shadows the application's for as long as the house is
	// open, which is what HouseIO.c does. Without it every custom background in the house
	// falls back to PICT 2000 and half the shipped houses look wrong.
	fork := filepath.Join(*houseArt, name)
	if st, err := os.Stat(fork); err == nil && st.IsDir() {
		assets.OpenHouseResFork(fork)
	} else if !*quiet {
		fmt.Fprintf(os.Stderr, "glidergo: no extracted resource fork at %s; custom art will fall back\n", fork)
	}

	view := render.DefaultView()
	scene := render.NewScene(view, assets, h)
	scene.NumNeighbors = *neighbors
	scene.Clock = time.Now()

	// ---- the window -----------------------------------------------------------

	win, err := backend.Open(platform.Config{
		Title:  "gliderGo -- " + name,
		Width:  int(view.Screen.Wide()),
		Height: int(view.Screen.Tall()),
		Scale:  *scale,
	})
	if err != nil {
		return err
	}
	defer win.Close()

	fb := platform.NewFramebuffer(int(view.Screen.Wide()), int(view.Screen.Tall()))

	// ---- the world ------------------------------------------------------------

	// A zero seed means "use the clock", which is what the original does: InitializeRandom
	// seeds from the time at launch. A fixed seed makes a whole run reproducible, which is
	// what 1.8's replay tests will want.
	s := int32(*seed)
	if s == 0 {
		s = int32(time.Now().UnixNano())
	}
	w := game.NewWorld(h, scene, s)
	w.TwoPlayer = *two

	// **DoBackground is true here and false in the original** (Main.c:186). It is the
	// preference that decides whether PlayGame pumps host events at all, and in a port it
	// is not optional: with it false the window never sees a keystroke, never repaints on
	// exposure and never notices that it lost the foreground. The original could get away
	// with it because the Toolbox drew the window's contents from the WindowRecord; a
	// modern compositor cannot. docs/IMPROVEMENTS.md 2.21 is the longer form of this: a
	// released build should always pause on focus loss and should not offer the choice.
	w.DoBackground = true

	// ---- the five host hooks --------------------------------------------------
	//
	// This is the entire seam between the game and the machine. Each hook is documented on
	// its field in internal/game/world.go; what follows is only this host's answer.

	// The 60.15 Hz Mac tick. The original's clock is the Time Manager's, incremented by
	// the vertical retrace; wall-clock milliseconds scaled by 60/1000.66 is the same thing
	// to within a tick, and the game measures nothing in absolute time.
	start := time.Now()
	if !*bench {
		w.TickCount = func() int64 {
			return int64(time.Since(start).Seconds() * 60.15)
		}

		// The frame limiter's loop body. The original's is empty -- a busy-wait that pins a
		// core for whatever fraction of the two ticks the frame did not need. Sleeping for a
		// tick instead is docs/IMPROVEMENTS.md 2.17, and it is safe because it changes only
		// *how* the wait is spent, not when it ends: awaitFrame still returns on the same tick.
		w.WaitTick = func() { time.Sleep(time.Millisecond) }
	}
	// -bench leaves both hooks nil, which is how the game is told to run flat out: with no
	// tick source awaitFrame skips its wait entirely and Ticks answers from the frame
	// counter, so the *game* still keeps nominal time -- Frame * TicksPerFrame -- while the
	// wall clock is free to run ahead. Nothing about the simulation changes, which is what
	// makes the number it prints a measurement of the composition and blit path rather than
	// of the limiter. It is also exactly the clock the headless build and every test in
	// internal/game uses, so a benchmark and a replay see the same frames.

	w.Present = func() {
		w.Main.ToBGRX(fb.Pix, fb.Stride)
		if err := win.Present(fb); err != nil {
			// A failed present is a dead window, and there is nothing useful to do with
			// the error from inside a void hook. Ending the game is the honest response
			// and it goes through the same door the Quit menu item does.
			w.Quitting = true
			w.SwitchedOut = false
		}
	}

	// The event pump. HandlePlayEvent calls this and the game's three arms are
	// RefreshGameWindow, Suspend and Resume; only the *classification* below is ours.
	w.PlayEvent = func() {
		for _, ev := range win.PollEvents() {
			switch {
			case ev.Kind == platform.EventQuit,
				ev.Kind == platform.EventKeyDown && ev.Key == platform.KeyEscape:
				// SwitchedOut is cleared alongside Quitting because PlayGame's pump loop
				// spins on SwitchedOut alone -- so a window closed while the game is in
				// the background would otherwise never be noticed. The original has the
				// same hole and cannot fall into it, because its Quit is a menu command
				// and a background application has no menu bar.
				w.Quitting = true
				w.SwitchedOut = false

			case ev.Kind == platform.EventFocus:
				if ev.Focused {
					w.Resume()
				} else {
					w.Suspend()
				}

			case ev.Kind == platform.EventResize:
				// The backend owns the scale transform; the game's surfaces are always
				// 640x480. See docs/IMPROVEMENTS.md 2.8 on inserting the resize transform at
				// the present step and nowhere else.

			case ev.Kind == platform.EventNone:
			}
		}

		// HandlePlayEvent's `sleep = 2`, and the only place it belongs. WaitNextEvent
		// yields for up to two ticks when the queue is empty; during play that is
		// invisible because awaitFrame is already spending the frame's budget, so
		// reproducing it there would install a second pacer. The one case where it
		// matters is this one: PlayGame spins on this hook while switched out, and
		// without a yield a backgrounded game burns a core.
		if w.SwitchedOut {
			time.Sleep(2 * time.Second / 60)
		}
	}

	// One poll, two gliders. The original snapshots the hardware KeyMap once, inside
	// player one's GetInput, and player two reads that snapshot -- so both calls in a frame
	// see the same keys. Reading the backend's held-key map has the same property for the
	// same reason: it is only updated by PollEvents, which ran once, above.
	w.KeyPoll = func(g *player.Glider) player.Keys {
		if g.Which == player.Player2 {
			return player.Keys{
				Left:  win.KeyDown(platform.KeyA),
				Right: win.KeyDown(platform.KeyD),
				Batt:  win.KeyDown(platform.KeyS),
				Band:  win.KeyDown(platform.KeyW),
				// No Command, no Delete and no Pause. Player two has no give-up key and
				// no way to reach the menus in the original either, which is one of the
				// two-player asymmetries docs/IMPROVEMENTS.md 2.23 asks 1.9 to fix.
			}
		}
		return player.Keys{
			Left:    win.KeyDown(platform.KeyLeft),
			Right:   win.KeyDown(platform.KeyRight),
			Batt:    win.KeyDown(platform.KeyDown),
			Band:    win.KeyDown(platform.KeyUp),
			Delete:  win.KeyDown(platform.KeyDelete),
			Pause:   win.KeyDown(platform.KeyTab),
			Command: false, // 1.7's menus; the game asks DoCommandKey, which is a stub
		}
	}

	// Sound is 1.6. Left nil, which PlayPrioritySound treats as "no channels", so every
	// call is dropped rather than queued -- see internal/game/env.go.

	// ---- play -----------------------------------------------------------------

	if *roomNum >= 0 {
		if *roomNum >= len(h.Rooms) {
			return fmt.Errorf("-room %d out of range (house has %d)", *roomNum, len(h.Rooms))
		}
		// The house's authored start room, overridden before NewGame reads it. Cleaner
		// than reaching past SetHouseToFirstRoom, and it is exactly what the editor's
		// "set as first room" command writes.
		h.FirstRoom = int16(*roomNum)
	}

	// -frames is the headless and timed path: a hook on the frame limiter is the only
	// place that can end a game from outside without inventing a key. It sets the same
	// flag the Quit menu item does.
	if *frames > 0 {
		w.Present = wrapPresentLimit(w, int64(*frames))
	}

	if !*quiet {
		fmt.Printf("glidergo: %s -- %d rooms, first %d, %d stars\n",
			name, len(h.Rooms), h.FirstRoom, w.CountStarsInHouse())
		fmt.Printf("glidergo: backend=%s surface=%dx%d scale=%d neighbors=%d seed=%d\n",
			backend.Name, fb.W, fb.H, *scale, *neighbors, s)
	}

	w.NewGame(game.NewGameMode)

	if !*quiet {
		el := time.Since(start)
		rate := float64(w.Frame) / el.Seconds()
		fmt.Printf("glidergo: %d frames in %v (%.1f fps), score %d, %d stars left\n",
			w.Frame, el.Round(time.Millisecond), rate, w.Score, w.StarsLeft)
		if *bench {
			// The original's target, for something to compare against: kTicksPerFrame is
			// 2 on a 60.15 Hz clock, so a Mac that kept up ran at 30.07 frames a second.
			fmt.Printf("glidergo: unpaced -- %.1fx the original's 30.07 fps target\n", rate/30.07)
		}
	}

	// Sticky asset errors, reported once at the end rather than at every blit, because the
	// draw helpers are void in the original and a missing PICT there ends in RedAlert.
	if err := assets.Err(); err != nil {
		return err
	}
	return nil
}

// wrapPresentLimit ends the game after n frames.
//
// It wraps Present rather than counting inside the loop because Present is called once per
// visible frame, which is what "-frames 300" means to someone timing the port. The
// alternative -- counting World.Frame -- also counts the frames PlayGame simulates and does
// not draw, and would make a headless dump and a windowed run disagree.
func wrapPresentLimit(w *game.World, n int64) func() {
	prev := w.Present
	var count int64
	return func() {
		if prev != nil {
			prev()
		}
		count++
		if count >= n {
			w.Quitting = true
			w.SwitchedOut = false
		}
	}
}
