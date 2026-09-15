// Command glidergo is the game.
//
// Run with no arguments it comes up on the title screen, finds every house in
// assets/extracted/houses and waits for somebody to start a game -- which is the
// whole of what stage 1.7 adds, and the reason this is a game rather than a
// demonstration. internal/shell is that title screen; internal/game is the 1994 code;
// this file is the machine, and play.go is one game on it.
//
//	go run ./cmd/glidergo                          # the title screen
//	go run ./cmd/glidergo -house "Demo House"      # skip it and play, by name or path
//	go run ./cmd/glidergo -scale 2                 # 2x nearest-neighbour magnification
//	go run ./cmd/glidergo -room 12 -neighbors 3    # start elsewhere, smaller view
//	go run ./cmd/glidergo -two                     # two gliders on one keyboard
//	go run ./cmd/glidergo -bench                   # 300 frames unpaced, report the rate
//	go run ./cmd/glidergo -shot /tmp/splash.png    # draw a screen to a PNG and exit
//	go run ./cmd/glidergo -audio list              # which external players this machine has
//	go run ./cmd/glidergo -sound=false             # the original's dontLoadSounds
//	go run ./cmd/glidergo -wav /tmp/session.wav    # record the mix as well as play it
//	go run ./cmd/glidergo -prefs none              # this build's defaults, saving nothing
//	go run ./cmd/glidergo -import-prefs "Glider Prefs"   # bring 1994's settings across
//	go run -tags nullbackend ./cmd/glidergo -frames 300 -dump /tmp/f   # headless
//
// Any of -house, -frames, -bench and -dump means "play, do not stop at a title
// screen": the first because naming a house is asking for it, and the other three
// because a timed run, a benchmark and a frame dump are measurements, and a
// measurement that waits for a keypress is not one. Everything else is the shell.
//
// Sound goes to an external player's stdin -- pw-play, paplay, aplay, ffplay or sox,
// whichever is installed -- because the port is standard-library-only Go and cannot
// open a device directly. internal/audio/sink.go has the whole argument. A machine
// with none of them plays in silence and says so; -wav writes the same mix to a file,
// which is how a session on such a machine can be listened to somewhere else.
//
// Keys. On the title screen the arrows move and Return chooses, and every item has a
// letter (N, 2, L, S, A, Q); see internal/shell on why that differs from the original's
// arcade key map. **In a game the keys are settings**, eight bindings plus the pause key,
// and S on the title screen is where they are changed; internal/prefs holds them and
// prefs.Default is the list of what they start as.
//
// Out of the box player one has the four arrows, as the original does: left and right to
// steer, up to fire a rubber band, down for the battery. Player two has A and D to steer,
// W for bands and S for the battery, and **that is a deliberate departure** -- the
// original binds player two to Control, Command, Option and Shift
// (InterfaceInit.c:148-151), which a modern window manager intercepts before the
// application sees it and which many keyboards cannot report independently. A player who
// wants the 1994 bindings back can now have them, which is what docs/IMPROVEMENTS.md 2.3
// asked for.
//
// Three keys are the port's own and are not bindable. Tab pauses by default, as the
// original does (`isEscPauseKey` is false at Main.c:184), and the settings screen offers
// Escape instead because those are the two the artwork exists for -- but **Escape pauses
// either way**, so that the key a stranger reaches for cannot throw a game away
// (docs/IMPROVEMENTS.md 2.7). Q gives up a paused game, standing in for the original's
// Command-Q, which a window manager now owns; and Delete abandons a glider waiting in limbo
// for the other player. Closing the window ends the program.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"glidergo/internal/audio"
	"glidergo/internal/platform"
	"glidergo/internal/prefs"
	"glidergo/internal/render"
	"glidergo/internal/shell"
)

// version is what the title screen and a bug report quote. The Makefile sets it from
// `git describe`; a plain `go build` leaves it as it stands here.
var version = "dev"

// defaultHouse is the house the original opens with -- Slumberland is what its
// shipped preferences name (PrefsInit, docs/analysis/ui-dialogs.md P1). It is the
// shell's opening selection when it is present, and the house the measurement flags
// use when they are given without one.
const defaultHouse = "Slumberland"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "glidergo: %v\n", err)
		os.Exit(1)
	}
}

// options is the command line, parsed and checked once.
type options struct {
	house     string
	houses    string
	artDir    string
	houseArt  string
	roomNum   int
	neighbors int
	scale     int
	two       bool
	seed      int64
	frames    int
	bench     bool
	dump      string
	quiet     bool

	shot       string
	shotScreen string

	prefsPath   string
	importPrefs string
	scoresDir   string

	sound    bool
	sounds   string
	music    bool
	volume   int
	audioOut string
	wav      string
}

func parseFlags() (*options, error) {
	o := &options{}
	flag.StringVar(&o.house, "house", "", "play this house at once instead of showing the title screen (name or path; default "+defaultHouse+" for -frames/-bench/-dump)")
	flag.StringVar(&o.houses, "houses", "assets/extracted/houses", "directory to search for houses")
	flag.StringVar(&o.artDir, "art", "assets/extracted/art", "extracted application art tree")
	flag.StringVar(&o.houseArt, "houseart", "assets/extracted/houseart", "extracted per-house resource forks")
	flag.IntVar(&o.roomNum, "room", -1, "start in this room number instead of the house's first")
	flag.IntVar(&o.neighbors, "neighbors", 9, "how much of the house to compose around the player: 1, 3 or 9")
	flag.IntVar(&o.scale, "scale", 1, "integer nearest-neighbour magnification of the 640x480 image")
	flag.BoolVar(&o.two, "two", false, "two players on one keyboard (the title screen's Two Player Game does the same)")
	flag.Int64Var(&o.seed, "seed", 0, "fix the random stream for a reproducible run (0 = use the clock)")
	flag.IntVar(&o.frames, "frames", 0, "quit after N frames, for headless and timed runs (0 = play)")
	flag.BoolVar(&o.bench, "bench", false, "run with no frame pacing and report the rate the machine sustains")
	flag.StringVar(&o.dump, "dump", "", "with -tags nullbackend, write each frame as a PNG into this directory")
	flag.BoolVar(&o.quiet, "quiet", false, "do not print the startup and shutdown summaries")

	flag.StringVar(&o.shot, "shot", "", "draw one title-screen frame to this PNG and exit; needs no display")
	flag.StringVar(&o.shotScreen, "shot-screen", "splash", "which screen -shot draws: splash, houses, settings, about, credits or scores")

	flag.StringVar(&o.prefsPath, "prefs", "", "preferences file to use instead of the one in the config directory (\""+prefsNone+"\" = this build's defaults, saving nothing)")
	flag.StringVar(&o.importPrefs, "import-prefs", "", "convert an original 226-byte \"Glider Prefs\" file into this port's settings, then exit")
	flag.StringVar(&o.scoresDir, "scores", "", "directory for the high-score files, one per house (\""+scoresNone+"\" = play without recording any)")

	flag.BoolVar(&o.sound, "sound", true, "load the sound bank; -sound=false is the original's dontLoadSounds")
	flag.StringVar(&o.sounds, "sounds", "assets/extracted/sound", "directory of extracted sound assets")
	flag.BoolVar(&o.music, "music", true, "play the score as well as the effects")
	flag.IntVar(&o.volume, "volume", 7, "output volume, 0 to 7; 0 is silence and also stops the score")
	flag.StringVar(&o.audioOut, "audio", "", "external player to pipe the mix to, or \"list\" for what this machine has")
	flag.StringVar(&o.wav, "wav", "", "write the mix to this WAV file")
	flag.Parse()

	if o.volume < 0 || o.volume > audio.FullVolume {
		return nil, fmt.Errorf("-volume must be 0 to %d", audio.FullVolume)
	}
	if o.scale < 1 {
		return nil, errors.New("-scale must be at least 1")
	}
	switch o.neighbors {
	case 1, 3, 9:
	default:
		return nil, errors.New("-neighbors must be 1, 3 or 9")
	}
	if o.dump != "" {
		os.Setenv("GLIDERGO_FRAMEDUMP", o.dump)
	}
	if o.bench && o.frames == 0 {
		// An unpaced game with no end is not a benchmark and not playable either --
		// the glider crosses the room in a few milliseconds. Three hundred frames is
		// ten seconds of game time, which is long enough for the rate to settle.
		o.frames = 300
	}
	return o, nil
}

func run() error {
	o, err := parseFlags()
	if err != nil {
		return err
	}

	// -audio list answers and exits, before anything is opened: somebody who has just
	// been told there is no sound wants the answer now, not after a megabyte of
	// samples has loaded.
	if o.audioOut == "list" {
		found := audio.Players()
		if len(found) == 0 {
			fmt.Println("glidergo: no audio player found; -wav writes a file instead")
			return nil
		}
		fmt.Printf("glidergo: audio players on this machine, best first: %s\n",
			strings.Join(found, " "))
		return nil
	}

	// The other one-shot, and it exits for a stronger reason: it *writes* the
	// preferences file, and a run that imported somebody's 1994 bindings and then came
	// up on a title screen would leave them wondering whether it had worked.
	if o.importPrefs != "" {
		return importPrefs(o)
	}

	// The settings, before anything reads one. See cmd/glidergo/prefs.go for the three
	// sources and their order.
	p, canSave := loadPrefs(o)
	overrideFromFlags(o, p)
	reportPrefsNotes(p)

	switch {
	case o.shot != "":
		return shot(o, p)
	case o.house != "" || o.frames > 0 || o.bench || o.dump != "":
		return playDirect(o, p)
	default:
		return runShell(o, p, canSave)
	}
}

// ---------------------------------------------------------------------------
// The three ways to start
// ---------------------------------------------------------------------------

// runShell is the ordinary one: a window, a title screen, and games started from it.
func runShell(o *options, p *prefs.Prefs, canSave bool) error {
	lib, err := shell.Discover(o.houses)
	if err != nil {
		// **Not fatal.** A missing or empty houses directory is what a fresh clone
		// has, and the useful place to say so is the screen the player is looking at
		// -- which is exactly what the shell does with an empty library. Exiting here
		// would put the one piece of information they need on a terminal they may
		// never see (docs/IMPROVEMENTS.md 2.6).
		fmt.Fprintf(os.Stderr, "glidergo: %v\n", err)
	}

	a := newApp(o, p, canSave)
	if err := a.openWindow("gliderGo"); err != nil {
		return err
	}
	defer a.close()
	if err := a.openAudio(); err != nil {
		return err
	}

	sh, err := shell.New(a.shellHost(), lib)
	if err != nil {
		return err
	}

	// The house the player last played, which is what the original opens with too:
	// `wasDefaultName` (Main.c:130), shipped as Slumberland. A name that is no longer
	// there is not an error -- houses are files and files get moved -- so it falls back
	// to the shipped default and says what happened.
	if p.House != "" && !sh.Select(p.House) {
		fmt.Fprintf(os.Stderr, "glidergo: %s is not in %s any more\n", p.House, o.houses)
		sh.Select(defaultHouse)
	} else if p.House == "" {
		sh.Select(defaultHouse)
	}

	if err := sh.Run(); err != nil {
		return err
	}

	// WriteOutPrefs' `PasStringCopy(thisHouseName, prefs.wasDefaultName)` (Main.c:377):
	// the house you were last on is remembered. Only when it changed, so that quitting
	// the title screen is not a file write, and only when there is somewhere to put it.
	if h, ok := sh.House(); ok && h.Name != p.House && canSave {
		p.House = h.Name
		if err := p.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "glidergo: cannot save the settings: %v\n", err)
		}
	}
	return a.artErr
}

// playDirect skips the shell: one house, one game, then exit.
//
// It saves nothing. -house is a flag and not a choice the player made in the game, so
// remembering it would let `-house "Fun House"` quietly change what the title screen opens
// with next time.
func playDirect(o *options, p *prefs.Prefs) error {
	name := o.house
	if name == "" {
		name = defaultHouse
	}
	path := housePath(o, name)
	name = houseName(path)

	a := newApp(o, p, false)
	if err := a.openWindow("gliderGo -- " + name); err != nil {
		return err
	}
	defer a.close()
	if err := a.openAudio(); err != nil {
		return err
	}
	if _, err := a.play(name, path, o.two); err != nil {
		return err
	}
	return a.artErr
}

// shot draws one title-screen frame into a PNG and exits.
//
// It opens no window, no audio and no house, which is the point: it is how the shell
// gets tested on a machine with no display and how `make headless` covers the screens
// a player actually meets first. A frame of the *game* has had that since 1.5
// (-frames with -dump); this is the same idea for the part of the program that is not
// the game.
func shot(o *options, p *prefs.Prefs) error {
	lib, err := shell.Discover(o.houses)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glidergo: %v\n", err)
	}

	view := render.DefaultView()
	scr := render.NewSurface(int(view.Screen.Wide()), int(view.Screen.Tall()))
	host := shell.Host{
		Screen:  scr,
		Assets:  render.NewAssets(o.artDir),
		Present: func() {},
		Poll:    func() []platform.Event { return nil },
		Play: func(shell.Choice) (shell.Outcome, error) {
			return shell.Outcome{}, errors.New("-shot does not play")
		},

		// The settings, so that -shot-screen settings has something to draw and so that
		// the About box quotes the real bindings. No SavePrefs and no ApplyPrefs: a
		// screenshot changes nothing and there is no mixer to tell. Without -prefs these
		// are this build's defaults, which is what makes the image reproducible.
		Prefs: p,

		// No Scores hook, so -shot-screen scores draws the board the *house file* carries
		// and not this machine's. That is deliberate and it is the same argument as the
		// scale below: a screenshot has to be reproducible, and a golden image that
		// changed the first time somebody on the build machine got onto the board would be
		// a test that fails for the best possible reason and still fails.

		Version: version,
	}
	sh, err := shell.New(host, lib)
	if err != nil {
		return err
	}
	if o.house != "" {
		sh.Select(houseName(housePath(o, o.house)))
	} else {
		sh.Select(defaultHouse)
	}
	if err := sh.Show(o.shotScreen); err != nil {
		return err
	}

	sh.Draw()

	// o.scale and not p.Scale, which are the same number whenever -scale was given and
	// differ only when a preferences file names one. A screenshot's magnification is a
	// property of the file being asked for rather than of the player's window, and
	// `-shot -prefs some.json` is how a golden image of the settings screen gets settings
	// to show -- that file must not be able to change the image's size out from under the
	// comparison.
	if err := writePNG(o.shot, scr, o.scale); err != nil {
		return err
	}
	if !o.quiet {
		fmt.Printf("glidergo: wrote %s -- the %s screen, %d houses, %d skipped\n",
			o.shot, o.shotScreen, len(lib.Houses), len(lib.Skipped))
	}
	return nil
}

// ---------------------------------------------------------------------------
// The shell's host
// ---------------------------------------------------------------------------

// shellHost is the shell's side of this machine. It is short because the shell asks
// for almost nothing: a surface, a way to show it, events, and a way to play.
func (a *app) shellHost() shell.Host {
	view := render.DefaultView()
	scr := render.NewSurface(int(view.Screen.Wide()), int(view.Screen.Tall()))

	// dead is how a failed present reaches a shell that has no error path: the next
	// poll reports the window closed, which is true, and the shell stops. A void hook
	// with nowhere to put an error is the same problem play.go's Present has, and it
	// gets the same answer.
	dead := false

	// The saver, or nothing at all. Method-valued rather than wrapped so that a
	// build with nowhere to write hands the shell a nil it can test, instead of a
	// function that quietly does nothing -- the settings screen tells the player
	// which of the two it has.
	var save func() error
	if a.canSave {
		save = a.p.Save
	}

	return shell.Host{
		Screen: scr,

		// The application's art, with no house resource fork open. The shell's chrome
		// is the application's even when a house redefines the same PICT ids: see
		// render.Assets.UI.
		Assets: render.NewAssets(a.o.artDir),

		Present: func() {
			scr.ToBGRX(a.fb.Pix, a.fb.Stride)
			if err := a.win.Present(a.fb); err != nil {
				fmt.Fprintf(os.Stderr, "glidergo: %v\n", err)
				dead = true
			}
			// The mixer's pacer, for the same reason play.go's Present calls it: the
			// splash screen is silent in this build, but the pump still has to be
			// clocked or the tail of the last game's audio would sit in the buffer
			// unplayed. When 1.7b makes music-on-the-splash a preference, this is
			// what makes it audible.
			a.pump.ClockTick()
		},

		Poll: func() []platform.Event {
			if dead {
				return []platform.Event{{Kind: platform.EventQuit}}
			}
			return a.win.PollEvents()
		},

		// A title screen has no reason to burn a core. One tick of the original's
		// clock is well under what a static screen needs and keeps the audio pump
		// clocked at about the rate the game clocks it.
		Idle: func() { time.Sleep(16 * time.Millisecond) },

		Play: func(c shell.Choice) (shell.Outcome, error) {
			return a.play(c.House.Name, c.House.Path, c.TwoPlayer)
		},

		// The board the High Scores screen shows: the side-car over the house file's own
		// rows. The shell caches whatever this answers and drops the cache after every
		// game, so this is a file read per house per visit and not per frame.
		Scores: a.board,

		// The settings screen edits this in place, so the next game reads whatever it
		// left behind -- which is the whole of how a rebind takes effect. The bindings
		// are resolved once per game (play.go), because the only way to reach this screen
		// is from the title screen and there is no game running while it is up.
		Prefs: a.p,

		// Nil when there is nowhere to write: -prefs none, or a machine with no
		// configuration directory. The screen says so on the way out.
		SavePrefs: save,

		// The one setting the machine holds its own copy of. The volume lives in the
		// mixer (internal/audio/engine.go) because every sample is scaled by it, so a
		// change made on the settings screen has to be pushed rather than polled -- and
		// this is what makes the volume audibly change while the screen is still up
		// instead of at the next game.
		ApplyPrefs: func() {
			if a.eng == nil {
				return
			}
			a.eng.SetVolume(int16(a.p.Volume))
			a.eng.SetSoundOn(a.p.Sound)
		},

		Title: func(s string) {
			if a.win != nil {
				a.win.SetTitle(s)
			}
		},
		Notify:  func(s string) { fmt.Fprintln(os.Stderr, s) },
		Version: version,
	}
}

// ---------------------------------------------------------------------------
// Odds and ends
// ---------------------------------------------------------------------------

// housePath turns whatever -house was given into a path. A name is looked up in the
// houses directory; anything with a separator or an extension in it is taken as a
// path, so a house sitting anywhere on the disk can be played without moving it.
func housePath(o *options, name string) string {
	if strings.ContainsRune(name, filepath.Separator) || filepath.Ext(name) != "" {
		return name
	}
	return filepath.Join(o.houses, name+".house")
}

// houseName is the house's name: its file name without the extension. houseType has
// no name field, because on a Mac the document's name was the file's, and every place
// the original shows a house name it is reading the FSSpec.
func houseName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// writePNG saves a surface, optionally upscaled by an integer factor, so that 640x480
// of 1994 pixels can be looked at without a viewer's own smoothing in the way.
func writePNG(path string, s *render.Surface, scale int) error {
	img := s.ToRGBA()
	if scale > 1 {
		b := img.Bounds()
		big := image.NewRGBA(image.Rect(0, 0, b.Dx()*scale, b.Dy()*scale))
		for y := 0; y < big.Rect.Dy(); y++ {
			for x := 0; x < big.Rect.Dx(); x++ {
				big.Set(x, y, img.At(x/scale, y/scale))
			}
		}
		img = big
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// openSink decides where the mix goes, and returns a one-word description of it for
// the startup line.
//
// Three cases, and the middle one is the one worth stating:
//
//	-wav alone          write the file and look for no player. Recording is the intent, and
//	                    starting a player as well would be a surprise on a build machine.
//	-audio and -wav     both, through a Tee: play it and keep what was played.
//	neither             the first player that exists. An empty answer is not an error here --
//	                    this host has no sound card at all -- so the caller warns and plays on.
func openSink(prefer, wav string) (audio.Sink, string, error) {
	var sinks []audio.Sink
	var where []string

	if wav != "" {
		f, err := audio.CreateWAV(wav)
		if err != nil {
			return &audio.Discard{}, "none", err
		}
		sinks = append(sinks, f)
		where = append(where, wav)
	}

	if prefer != "" || wav == "" {
		pipe, err := audio.OpenPipe(prefer)
		if err != nil {
			if len(sinks) == 0 {
				return &audio.Discard{}, "none", err
			}
			// A WAV is already open, so the session is not silent and the missing
			// player is worth less than the recording: report it and keep the file.
			return sinks[0], where[0], err
		}
		sinks = append(sinks, pipe)
		where = append(where, pipe.Name())
	}

	switch len(sinks) {
	case 0:
		return &audio.Discard{}, "none", nil
	case 1:
		return sinks[0], where[0], nil
	default:
		return audio.Tee(sinks), strings.Join(where, "+"), nil
	}
}

// reportAudio is the audio half of the shutdown summary.
//
// Three of these numbers are the ones a bug report needs and cannot get any other way.
// *refused* is the channel policy doing its job -- three channels and a busy room -- and a
// large number there is not a defect. *skipped* is the frame loop having stalled for more than
// four frames, which is a game problem wearing an audio problem's clothes. *dropped* is the
// external player having stopped reading, which is the player's problem or the machine's. They
// have three different fixes, which is why they are three different counters.
func reportAudio(eng *audio.Engine, pump *audio.Pump, sink audio.Sink) {
	if eng == nil {
		return
	}
	st := eng.Stats()
	fmt.Printf("glidergo: sound -- %d requests, %d played, %d refused, %d cut off, %d music pieces\n",
		st.Requests, st.Granted, st.Refused+st.TriggerRefused, st.Displaced, st.MusicStarted)

	line := fmt.Sprintf("glidergo: mix -- %.1fs of audio", float64(pump.Mixed())/audio.Rate)
	if st.Clipped > 0 {
		line += fmt.Sprintf(", %d samples clipped", st.Clipped)
	}
	if pump.Skipped > 0 {
		line += fmt.Sprintf(", %.2fs skipped after stalls", float64(pump.Skipped)/audio.Rate)
	}
	if p, ok := sink.(*audio.Pipe); ok {
		if n := p.Dropped(); n > 0 {
			line += fmt.Sprintf(", %d samples dropped by %s", n, p.Name())
		}
		if err := p.Err(); err != nil {
			line += fmt.Sprintf(", %s stopped reading (%v)", p.Name(), err)
		}
	}
	if pump.Err != nil {
		line += fmt.Sprintf(", sink error (%v)", pump.Err)
	}
	fmt.Println(line)
}
