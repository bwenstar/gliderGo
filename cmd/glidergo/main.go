// Command glidergo is the game.
//
// Run with no arguments it comes up on the title screen, lists every house built
// into this executable and waits for somebody to start a game -- which is the
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
//	go run ./cmd/glidergo -audio list              # which sound outputs this machine has
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
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bwenstar/gliderGo/assets"
	"github.com/bwenstar/gliderGo/internal/assetfs"
	"github.com/bwenstar/gliderGo/internal/audio"
	"github.com/bwenstar/gliderGo/internal/cliargs"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/platform/backend"
	"github.com/bwenstar/gliderGo/internal/prefs"
	"github.com/bwenstar/gliderGo/internal/project"
	"github.com/bwenstar/gliderGo/internal/render"
	"github.com/bwenstar/gliderGo/internal/saved"
	"github.com/bwenstar/gliderGo/internal/scores"
	"github.com/bwenstar/gliderGo/internal/shell"
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

		// One line under a fatal error, because a released binary travels without this
		// repository and the person reading the message has no other way to find out where
		// to send it. It is split two ways on purpose: somebody who mistyped a flag is told
		// about -h, and everybody else is pointed at the tracker. Sending the mistyped-flag
		// case to an issue tracker would be the fastest way to make the footer something
		// nobody reads.
		//
		// The second line is conditional -- "if that is not something you can fix" -- and
		// that wording is load-bearing rather than polite. Most of what reaches here is the
		// player's to fix: a house that is not where they said it was, a directory that does
		// not exist. A footer that called every one of those a bug would be wrong most of the
		// time, and a footer that is wrong most of the time is invisible by the time it is
		// right.
		me := filepath.Base(os.Args[0])
		var u usageErr
		if errors.As(err, &u) {
			fmt.Fprintf(os.Stderr, "glidergo: `%s -h` lists the flags\n", me)
		} else {
			fmt.Fprintf(os.Stderr, "glidergo: if that is not something you can fix, %s "+
				"-- paste the output of `%s -version`\n", project.Issues, me)
		}
		os.Exit(1)
	}
}

// usageErr marks the one class of failure that must not carry the bug-report footer: the
// command line was wrong. It wraps rather than replaces, so the message is still the specific
// one the check wrote -- `-neighbors must be 1, 3 or 9` and not a generic "bad usage".
type usageErr struct{ error }

func (u usageErr) Unwrap() error { return u.error }

// options is the command line, parsed and checked once.
type options struct {
	house string

	// The four asset roots. **Empty means the copy built into this executable**, which is
	// what every one of them is unless somebody says otherwise, and the reason a downloaded
	// binary needs no files beside it. A named directory replaces that root rather than
	// layering over it; see internal/assetfs, and read the four of them through the methods
	// below rather than reaching for the strings.
	houses   string
	artDir   string
	houseArt string

	// levels is the fifth root: the houses written for this port, listed in the picker beside
	// the twenty-two as a separate level set. A directory replaces the built-in copy exactly as
	// the four above do -- what is *added* is the set, not the root. See sources below, and
	// internal/shell/sets.go for what a set claims.
	levels string

	// tree is the built-in tree the four roots above resolve against: assets.Tree(). It is a
	// field and not a package call so that a test can build an options with none.
	tree fs.FS

	// levelTree is the second one, assets.Levels(), and it is separate for the reason
	// internal/assetpack.LevelsName gives: the 1994 data carries a claim that `make
	// assets-check` proves, and a house written this week does not belong inside it. A nil
	// levelTree with no -levels flag is a build with no houses of its own, which is a state
	// tests use and nothing released is.
	levelTree fs.FS

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
	savesDir    string
	resume      bool

	sound    bool
	sounds   string // the fourth asset root, and empty means built-in like the rest
	music    bool
	volume   int
	audioOut string
	wav      string

	showVersion bool
}

// The five asset roots, each as a filesystem and a name for messages. They are resolved at
// every use rather than once into fields, because resolving one is an os.DirFS or an fs.Sub
// and neither reads anything -- and because a root that is a live directory should be looked
// at when it is needed, not cached at startup. See internal/assetfs.
//
// Four are subtrees of one archive and levelsRoot is the whole of a second, which is the only
// reason it calls a different function.

func (o *options) artRoot() (fs.FS, string)      { return assetfs.Root(o.tree, o.artDir, "art") }
func (o *options) houseArtRoot() (fs.FS, string) { return assetfs.Root(o.tree, o.houseArt, "houseart") }
func (o *options) housesRoot() (fs.FS, string)   { return assetfs.Root(o.tree, o.houses, "houses") }
func (o *options) levelsRoot() (fs.FS, string)   { return assetfs.Whole(o.levelTree, o.levels, "levels") }
func (o *options) soundRoot() (fs.FS, string)    { return assetfs.Root(o.tree, o.sounds, "sound") }

// sources is the list of places the picker's houses come from, and the set each one declares.
// It is the one place in the program that decides what a level set *means*, so it is worth
// reading the three decisions in it.
//
// The houses root is Original when it is the built-in one and Other when -houses named a
// directory. That is not caution for its own sake: the game cannot know what somebody put in a
// directory, and a list that called an arbitrary folder "Original" would be making a
// provenance claim out of a flag. Other is the true answer, and the picker shows the root's
// label beside the count so it is checkable.
//
// The levels root is a second source rather than a substitute for the first, because two houses
// do not resolve against each other the way two copies of PICT 1000 do -- see
// internal/shell/library.go. A player who types -levels replaces the houses in the New set,
// which is the same rule the other four roots follow; what neither flag can do is fold the two
// sets into one.
//
// It is added only when there is something there. That is no longer the interesting case, since
// every release now carries the port's own houses and the New set is always drawn -- but a build
// whose levels archive is empty, or a test that passes no levelTree, still has one set, and a
// picker with one set draws no chooser at all.
func (o *options) sources() []shell.Source {
	housesFS, housesName := o.housesRoot()
	set := shell.SetOriginal
	if o.houses != "" {
		set = shell.SetOther
	}
	out := []shell.Source{{FS: housesFS, Label: housesName, Set: set}}

	// A -levels directory that is missing is still listed, so Discover reports it by name:
	// somebody who typed the flag wants the error rather than a silently unchanged picker.
	// A built-in levels root that is absent is not an error and not mentioned, because no
	// caller asked for it.
	if levelsFS, levelsName := o.levelsRoot(); o.levels != "" || assetfs.IsDir(levelsFS, ".") {
		out = append(out, shell.Source{FS: levelsFS, Label: levelsName, Set: shell.SetNew})
	}
	return out
}

// printVersion answers -version. It prints more than the version string because the
// thing it is for is the first line of a bug report, and the four facts underneath
// are the ones that decide whether a report is even about the same program: which
// backend was compiled in (the null one draws nothing and is chosen silently by any
// build without cgo, or off Linux), which Go built it, which OS and architecture, and
// where the game is reading its assets from. That last one is the built-in copy unless a
// flag says otherwise, so "missing" means a flag names a directory that is not there --
// which explains a large class of "it starts and there is nothing there" reports on its own.
//
// It deliberately does not open a window, load a sound bank or read the preferences,
// so it answers on a machine where the game itself cannot start.
func printVersion(o *options) {
	fmt.Printf("glidergo %s\n", version)
	fmt.Printf("  backend   %s\n", backend.Name)
	fmt.Printf("  built by  %s\n", runtime.Version())
	fmt.Printf("  platform  %s/%s\n", runtime.GOOS, runtime.GOARCH)
	for _, r := range vcsRows() {
		fmt.Printf("  %-9s %s\n", r[0], r[1])
	}

	// What the executable is carrying, first, because it is the answer to "does this need
	// files beside it" and because a build with no assets in it -- which nothing released is,
	// but a `go build` of a stripped tree could be -- explains everything below it.
	//
	// Both archives, counted together. Two lines would have invited the reading that a binary
	// can have one and not the other, and it cannot: they are two //go:embed directives in one
	// package, so a build has both or does not compile. The levels root gets its own row below,
	// where the question is which houses rather than how many bytes.
	files, bytes := assetfs.Measure(o.tree)
	if lf, lb := assetfs.Measure(o.levelTree); lf > 0 {
		files, bytes = files+lf, bytes+lb
	}
	if files > 0 {
		fmt.Printf("  assets    built in (%d files, %d KiB)\n", files, bytes/1024)
	} else {
		fmt.Printf("  assets    none built in\n")
	}

	// Then each root, named individually rather than as one yes/no: the four are extracted by
	// separate passes of tools/extract_all.py, and a half-extracted directory somebody pointed
	// a flag at is a real state rather than a hypothetical one (docs/IMPROVEMENTS.md 5.2).
	artFS, artName := o.artRoot()
	soundFS, soundName := o.soundRoot()
	housesFS, housesName := o.housesRoot()
	houseArtFS, houseArtName := o.houseArtRoot()
	// A named type and not an anonymous one, because the levels row below is appended
	// conditionally and an anonymous struct would have to be spelled out twice to do it.
	//
	// rebuild is the target that regenerates this root, and it is a field rather than one
	// sentence at the bottom because the two archives are rebuilt by different targets and the
	// old single remedy told a stale levels archive to run `make assets`.
	type row struct {
		what    string
		flag    string
		ok      bool
		where   string
		rebuild string
	}
	rows := []row{
		{"art", "art", assetfs.Exists(artFS, "manifest.json"), artName, "make assets"},
		{"sound", "sounds", assetfs.Exists(soundFS, "manifest.tsv"), soundName, "make assets"},
		{"houses", "houses", assetfs.IsDir(housesFS, "."), housesName, "make assets"},
		{"houseart", "houseart", assetfs.IsDir(houseArtFS, "."), houseArtName, "make assets"},
	}

	// The fifth root, and only when there is one. It is conditional where the other four are
	// unconditional because it is the only root a build can legitimately lack: an executable
	// with an empty levels archive is complete, and a "levels none" row would read as something
	// missing rather than as something not asked for. A release has houses of its own, so in
	// practice this row is always drawn -- it is the line that says which ones, and the one that
	// says "missing (some/dir)" when -levels was a typo.
	if levelsFS, levelsName := o.levelsRoot(); o.levels != "" || assetfs.IsDir(levelsFS, ".") {
		rows = append(rows, row{"levels", "levels", assetfs.IsDir(levelsFS, "."),
			levelsName, "make levels-zip"})
	}

	for _, t := range rows {
		switch {
		case t.where == "":
			fmt.Printf("  %-9s none -- name a directory with -%s\n", t.what, t.flag)
		case t.ok:
			fmt.Printf("  %-9s found (%s)\n", t.what, t.where)
		case assetfs.Built(t.where):
			// The archive in this executable is short of a root, which is a broken build
			// and not anything the person running it did.
			fmt.Printf("  %-9s missing (%s) -- `%s` rebuilds it\n", t.what, t.where, t.rebuild)
		default:
			// A directory a flag named, so there is nothing to rebuild: the path is wrong,
			// or it is right and the directory is not the root it was taken for. This is
			// the commonest way to see this line at all, and the old message sent every
			// mistyped -art at `make assets` in a source tree the player may not have.
			fmt.Printf("  %-9s missing (%s) -- -%s named it, and there is no %s root there\n",
				t.what, t.where, t.flag, t.what)
		}
	}

	// Then where the sound would go, which is the one row here that is a *prediction* rather
	// than a reading. It earns its place because "no sound" is the commonest report a port of a
	// 1994 Mac game gets and because the causes are four different things -- the flag, the bank,
	// the platform and what happens to be installed -- and only two of them are visible from
	// outside the process.
	fmt.Printf("  audio     %s\n", audioRoute(o))

	// Then this installation's own three files, which is the question nobody can answer from
	// outside the process. The original kept one 226-byte resource in the System Folder; this
	// port keeps settings, score boards and saved games in three places under
	// os.UserConfigDir and os.UserHomeDir, and which directories those are depends on the
	// platform, on four XDG variables and on three flags. "nowhere" is a real answer and not
	// a fault -- `-prefs none` and a machine with no configuration directory both land there
	// -- and it is the answer to "it forgot my settings again", which is otherwise
	// unanswerable without reading this source. Paths only: nothing here opens a file, so
	// the block still works on the machine where the game will not start.
	for _, r := range statePaths(o) {
		fmt.Printf("  %-9s %s\n", r[0], r[1])
	}

	// Last, and the reason the whole block is worth pasting: where this came from and where
	// the paste goes. A report that arrives with everything above it and no way to tell which
	// project it is about has happened to every program that ships more than one binary, and
	// the lines cost nothing.
	//
	// The provenance is four rows rather than one sentence because the one sentence had to
	// choose an article for an SPDX identifier -- it read "under the GPL-2.0-only" -- and
	// because these are four separate facts that get quoted separately: which project, where
	// the bugs go, which revision of whose C this is a transcription of, and who holds what.
	fmt.Printf("  home      %s\n", project.Home)
	fmt.Printf("  bugs      %s\n", project.Issues)
	fmt.Printf("  licence   %s\n", project.Licence)
	fmt.Printf("  port of   %s -- %s / %s, %s\n", project.Original, project.OriginalAuthor,
		project.OriginalPublisher, project.OriginalYear)
	fmt.Printf("  from      %s @ %s\n", project.Upstream, project.UpstreamCommit)
	fmt.Printf("  %s; %s %s\n", project.Copyright, project.Original, project.OriginalCopyright)
}

// vcsRows is what the toolchain stamped into this executable, as label/value pairs.
//
// It is here because `version` above is a linker variable the Makefile sets from `git
// describe`, and a plain `go build ./cmd/glidergo` -- which is what somebody trying the project
// for the first time runs -- leaves it as the literal string "dev". A report from such a binary
// was unattributable. Since Go 1.18 the toolchain stamps `vcs.revision`, `vcs.time` and
// `vcs.modified` into any build made inside a repository without being asked, so the commit is
// recoverable anyway, and `modified true` is the single most useful line in a bug report: it
// says the binary is not any commit at all and the diff has to come from the reporter.
//
// Empty is a normal answer, not a failure: a build from a release tarball, from a module cache
// or with -buildvcs=false has no stamps, and the rows are simply absent rather than printed as
// "unknown", because a row that says nothing is worse than no row.
func vcsRows() [][2]string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}
	// Fixed order, looked up rather than ranged over, because the order Settings comes in is
	// not part of the toolchain's contract and this is output somebody reads.
	labels := [][2]string{
		{"vcs.revision", "commit"},
		{"vcs.time", "committed"},
		{"vcs.modified", "modified"},
	}
	var rows [][2]string
	for _, want := range labels {
		for _, s := range info.Settings {
			if s.Key == want[0] {
				rows = append(rows, [2]string{want[1], s.Value})
				break
			}
		}
	}
	return rows
}

// statePaths answers "where does this build keep my things", for the three stores the game
// writes, under the flags this invocation was given. Label and value, so the caller owns the
// formatting.
//
// It mirrors loadPrefs, app.openScores and app.openSaves rather than calling them, and that is
// a deliberate duplication of three switch statements: those three open files, print notes and
// build stores, and -version must do none of that. The duplication is small, it is in one
// place, and cmd/glidergo's own test pins the three `none` spellings against it.
// audioRoute says where this build would send the mix, and opens nothing on the way.
//
// Nothing here is allowed to open a device, which is the whole difficulty: the honest answer is
// what openSink returns and openSink starts a player. So this mirrors its three cases against
// audio.Outputs(), which only asks the platform what exists -- exec.LookPath for the five
// external players, waveOutGetNumDevs on Windows. A -version run must be safe beside a running
// game, and on the native device it would not be: waveout refuses a second open in one process,
// so asking properly would print a failure caused by the asking.
//
// Which makes this a prediction, and the wording says so where it can be wrong. `-audio` names
// an output that Open insists on, so the row says the run stops if it is missing rather than
// claiming it is there; the unflagged case names what Open would *try* first and then the rest,
// because the rest is the list somebody about to type `-audio` wants.
func audioRoute(o *options) string {
	// Two flags answer before any output is considered, and both are states a bug report
	// arrives in: -sound=false is the original's dontLoadSounds and a run with it is silent for
	// a reason that has nothing to do with this machine's players, and -audio list is checked
	// after -version in run() so the pair is reachable and would otherwise read as a request
	// for an output called "list".
	switch {
	case !o.sound:
		return "off (-sound=false: no bank is loaded, as the original's dontLoadSounds)"
	case o.audioOut == "list":
		return "nothing opened (-audio list prints this machine's outputs and exits)"
	}

	var to []string
	if o.wav != "" {
		to = append(to, o.wav+" (a file: -wav records the mix)")
	}
	// openSink's condition, exactly: -wav on its own records and opens no player, because
	// starting one on a build machine would be a surprise.
	if o.audioOut != "" || o.wav == "" {
		found := audio.Outputs()
		switch {
		case o.audioOut != "":
			to = append(to, o.audioOut+" (named with -audio, so the run stops if it is not there)")
		case len(found) == 0:
			to = append(to, "nothing -- no sound output on this machine; "+
				"`-audio list` names what it looks for, and -wav writes a file instead")
		case len(found) == 1:
			to = append(to, found[0])
		default:
			to = append(to, found[0]+" (also installed: "+strings.Join(found[1:], " ")+")")
		}
	}
	return strings.Join(to, " + ")
}

func statePaths(o *options) [][2]string {
	// resolve turns one flag into the line to print. The empty flag is the interesting case:
	// it means "the usual place", and the usual place is a computation that can fail on a
	// machine with no home directory -- which is a state worth naming rather than hiding.
	resolve := func(name, value, none string, usual func() (string, error)) string {
		switch value {
		case none:
			return fmt.Sprintf("nowhere (-%s %s: nothing is read or written)", name, none)
		case "":
			p, err := usual()
			if err != nil {
				return fmt.Sprintf("nowhere (%v)", err)
			}
			return p
		default:
			return value
		}
	}

	rows := [][2]string{
		{"prefs", resolve("prefs", o.prefsPath, prefsNone, prefs.Path)},
		{"scores", resolve("scores", o.scoresDir, scoresNone, scores.Dir)},
		{"saves", resolve("saves", o.savesDir, savesNone, saved.Dir)},
	}
	// The fourth case loadPrefs has, and the one that would otherwise make this block lie:
	// a measurement -- -shot, -frames, -bench or -dump -- reads the defaults and writes
	// nothing, unless -prefs named a file outright. See hermetic in prefs.go.
	if o.prefsPath == "" && hermetic(o) {
		rows[0][1] = "nowhere (a measurement run uses this build's defaults)"
	}
	return rows
}

func parseFlags() (*options, error) {
	o := &options{tree: assets.Tree(), levelTree: assets.Levels()}
	flag.StringVar(&o.house, "house", "", "play this house at once instead of showing the title screen (name or path; default "+defaultHouse+" for -frames/-bench/-dump)")
	flag.StringVar(&o.houses, "houses", "", "directory to search for houses instead of the ones built in")
	flag.StringVar(&o.levels, "levels", "", "directory of houses to list as the New level set instead of the ones built in")
	flag.StringVar(&o.artDir, "art", "", "extracted application art tree to use instead of the one built in")
	flag.StringVar(&o.houseArt, "houseart", "", "extracted per-house resource forks to use instead of the ones built in")
	flag.IntVar(&o.roomNum, "room", -1, "start in this room number instead of the house's first")
	flag.IntVar(&o.neighbors, "neighbors", 9, "how much of the house to compose around the player: 1, 3 or 9")
	flag.IntVar(&o.scale, "scale", 1, "integer nearest-neighbour magnification of the 640x480 image")
	flag.BoolVar(&o.two, "two", false, "two players on one keyboard (the title screen's Two Player Game does the same)")
	flag.Int64Var(&o.seed, "seed", 1, "the random stream's starting state; 1 is what the 1994 build launched with (0 = use the clock instead)")
	flag.IntVar(&o.frames, "frames", 0, "quit after N frames, for headless and timed runs (0 = play)")
	flag.BoolVar(&o.bench, "bench", false, "run with no frame pacing and report the rate the machine sustains")
	flag.StringVar(&o.dump, "dump", "", "with -tags nullbackend, write each frame as a PNG into this directory")
	flag.BoolVar(&o.quiet, "quiet", false, "do not print the startup and shutdown summaries")

	flag.StringVar(&o.shot, "shot", "", "draw one title-screen frame to this PNG and exit; needs no display")
	flag.StringVar(&o.shotScreen, "shot-screen", "splash", "which screen -shot draws: splash, houses, settings, about, credits or scores")

	flag.StringVar(&o.prefsPath, "prefs", "", "preferences file to use instead of the one in the config directory (\""+prefsNone+"\" = this build's defaults, saving nothing)")
	flag.StringVar(&o.importPrefs, "import-prefs", "", "convert an original 226-byte \"Glider Prefs\" file into this port's settings, then exit")
	flag.StringVar(&o.scoresDir, "scores", "", "directory for the high-score files, one per house (\""+scoresNone+"\" = play without recording any)")
	flag.StringVar(&o.savesDir, "saves", "", "directory for saved games, one per house (\""+savesNone+"\" = play without saving any)")
	flag.BoolVar(&o.resume, "resume", false, "with -house, resume its saved game instead of starting a new one")

	flag.BoolVar(&o.sound, "sound", true, "load the sound bank; -sound=false is the original's dontLoadSounds")
	flag.StringVar(&o.sounds, "sounds", "", "directory of extracted sound assets to use instead of the ones built in")
	flag.BoolVar(&o.music, "music", true, "play the score as well as the effects")
	flag.IntVar(&o.volume, "volume", 7, "output volume, 0 to 7; 0 is silence and also stops the score")
	flag.StringVar(&o.audioOut, "audio", "", "sound output to use, or \"list\" for what this machine has")
	flag.StringVar(&o.wav, "wav", "", "write the mix to this WAV file")

	flag.BoolVar(&o.showVersion, "version", false, "print the build, the compiled-in backend and where the assets are coming from, then exit")

	// `-h` is the other thing a player reaches for before they reach for a search engine, and
	// the stock message -- "Usage of /tmp/glidergo:" followed by thirty flags -- says neither
	// what the program is nor where it came from. Everything the flags do is optional: the
	// binary plays the game with no arguments at all, which is worth saying once at the top.
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "%s -- a port of %s (%s, %s).\n\n",
			project.Name, project.Original, project.OriginalAuthor, project.OriginalYear)
		fmt.Fprintf(w, "usage: %s [flags] [house]\n\nRun it with no flags to play. The rest is for\n"+
			"development, headless runs and moving a 1994 installation across.\n\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintf(w, "\n%s -- bugs to %s\n", project.Home, project.Issues)
	}
	// flag.Parse() with the arguments reordered, rather than flag.Parse(), so that
	// `glidergo Slumberland -scale 2` works: the flag package stops at the first
	// non-flag argument, which made that one "one house at a time: 3 were named
	// (Slumberland, -scale, 2)" -- a message about the wrong thing entirely. The house
	// name is the argument a *player* types, which is what makes this worth the
	// indirection here and not only in glidertool (internal/cliargs, IMPROVEMENTS 4.13).
	//
	// flag.CommandLine is read at the point of use and not captured, because
	// args_test.go's parseArgs swaps it for a fresh FlagSet to keep one test's flags out
	// of the next one's: a reordering done against the old one would consult a FlagSet
	// that has none of the flags above registered, and would then leave every flag's
	// value stranded in the positional list.
	flag.CommandLine.Parse(cliargs.FlagsFirst(flag.CommandLine, os.Args[1:]))

	// Before every other check, because -version has to work on a machine where nothing
	// else does -- including one where the flags it is given alongside are wrong.
	if o.showVersion {
		return o, nil
	}

	// A house named without -house. Until 2.1 this was silently dropped -- `glidergo Slumberland`
	// showed the title screen, which looks like the house was refused rather than never read, and
	// costs the hour the -resume/-room checks below are worded to save. It is accepted rather than
	// refused for two reasons: the 1994 program was a Mac application and a house was one of its
	// documents, so opening one by naming it is the original's gesture, not a new convenience; and
	// it is the form an operating system uses when a file type is associated with a binary, which
	// is what a double-clicked .house has to become on Windows and macOS.
	//
	// Both spellings at once is still an error, on the -resume/-room principle: two answers were
	// given to one question and guessing which was meant is how an argument gets ignored quietly.
	switch {
	case flag.NArg() > 1:
		return nil, fmt.Errorf("one house at a time: %d were named (%s)",
			flag.NArg(), strings.Join(flag.Args(), ", "))
	case flag.NArg() == 1 && o.house != "":
		return nil, fmt.Errorf("-house %q and %q name two houses: give one", o.house, flag.Arg(0))
	case flag.NArg() == 1:
		o.house = flag.Arg(0)
	}

	if o.volume < 0 || o.volume > audio.FullVolume {
		return nil, fmt.Errorf("-volume must be 0 to %d", audio.FullVolume)
	}
	if o.resume && o.two {
		// A gameType holds one glider's room, position, mode and facing, so there is
		// nowhere in the format for a second player and the original refuses the save for
		// the same reason (game.CanSaveGame). Refusing here rather than quietly dropping
		// one of the two flags: both were asked for and they contradict each other.
		return nil, errors.New("-resume and -two cannot both be given: a saved game holds one glider")
	}
	if o.resume && o.roomNum >= 0 {
		// -room rewrites the house's first room, and a resume does not start in the house's
		// first room -- it starts where the save says. Accepting both would silently ignore
		// one of them, which is the failure mode that costs an hour of wondering why.
		return nil, errors.New("-resume and -room cannot both be given: a saved game names its own room")
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

// endlessHeadlessRun reports whether this run would draw frames into nothing, forever.
//
// A parameter rather than a reference to backend.Name, because that is a build-time constant: on
// the host `go test` runs on it is "x11", so a condition written against it directly could only be
// exercised by a second test binary built with -tags nullbackend, and nothing builds one. Taking
// the name makes every case reachable from an ordinary run of the suite -- which is the same
// argument platform.DisplayAdvice is built on, and it is worth repeating because this is a guard
// whose failure mode is a hang, and a hang is the one failure a test suite cannot report.
//
// -shot is exempt and the one-shots have already returned by the call site: each of those is a
// whole job that finishes, and finishing with no window is exactly what the null backend is for.
// -bench needs no mention because parseFlags has already turned it into 300 frames, which is the
// only reason this reads as two conditions rather than four.
func endlessHeadlessRun(backendName string, o *options) bool {
	return backendName == "null" && o.frames == 0 && o.shot == ""
}

func run() error {
	o, err := parseFlags()
	if err != nil {
		// Everything parseFlags rejects is the command line, so the wrap is one site rather
		// than seven `usageErr{...}` constructions inside the checks.
		return usageErr{err}
	}

	if o.showVersion {
		printVersion(o)
		return nil
	}

	// -audio list answers and exits, before anything is opened: somebody who has just
	// been told there is no sound wants the answer now, not after a megabyte of
	// samples has loaded.
	if o.audioOut == "list" {
		found := audio.Outputs()
		if len(found) == 0 {
			fmt.Println("glidergo: no sound output found; -wav writes a file instead")
			return nil
		}
		fmt.Printf("glidergo: sound outputs on this machine, best first: %s\n",
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

	// A null-backend build has no window and no keyboard, so nothing can ever ask it to stop:
	// null.Window returns only the events a script handed it, and a play with no script never sees
	// an EventQuit, so Closed() stays false forever. Without this check the run draws frames into
	// nothing, silently, until somebody types Ctrl-C -- and it is not an obscure way to arrive
	// there. It is what `CGO_ENABLED=0 make run` does, which is what CONTRIBUTING.md offers to
	// anybody who cannot install libx11-dev, and it is what a cross-compiled macOS or arm64 binary
	// does on the machine it was built for. A first five minutes that ends in a hang with no output
	// teaches nothing about which of the two it was.
	//
	if endlessHeadlessRun(backend.Name, o) {
		return usageErr{errors.New("this build has no window and no keyboard (backend null), so " +
			"nothing can ask it to quit: give it -frames N, -bench, or -shot FILE to draw one " +
			"screen and exit. A windowed build needs libx11-dev on Linux -- `make doctor` checks")}
	}

	switch {
	case o.shot != "":
		return shot(o, p)
	case o.house != "" || o.frames > 0 || o.bench || o.dump != "" || o.resume:
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
	lib, err := shell.Discover(o.sources()...)
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
	// The score behind the title screen, if the player wants it: the original plays it
	// while nobody is playing, and it is a preference of its own. See music.go.
	a.startTitleMusic()

	// The house the player last played, which is what the original opens with too:
	// `wasDefaultName` (Main.c:130), shipped as Slumberland. A name that is no longer
	// there is not an error -- houses are files and files get moved -- so it falls back
	// to the shipped default and says what happened.
	// lib.Root and not the houses root: with a levels root as well there is more than one
	// place the house could have been, and naming only the first would send somebody looking
	// in the wrong one.
	if p.House != "" && !sh.Select(p.House) {
		fmt.Fprintf(os.Stderr, "glidergo: %s is not in %s any more\n", p.House, lib.Root)
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
	ref, err := resolveHouse(o, name)
	if err != nil {
		return err
	}

	a := newApp(o, p, false)
	if err := a.openWindow("gliderGo -- " + ref.Name); err != nil {
		return err
	}
	defer a.close()
	if err := a.openAudio(); err != nil {
		return err
	}
	if _, err := a.play(ref, o.two, o.resume); err != nil {
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
	lib, err := shell.Discover(o.sources()...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glidergo: %v\n", err)
	}

	artFS, _ := o.artRoot()
	view := render.DefaultView()
	scr := render.NewSurface(int(view.Screen.Wide()), int(view.Screen.Tall()))
	host := shell.Host{
		Screen:  scr,
		Assets:  render.NewAssets(artFS),
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
		// The name, not the file: -shot never opens a house, and houseName answers for a
		// path, a file name and a bare name alike.
		sh.Select(houseName(o.house))
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
	artFS, _ := a.o.artRoot()

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

	// The same shape for the saved game, and for the same reason: a nil hook is what the
	// menu row reads as "this build has nowhere to keep saved games", and it says so on the
	// status band instead of offering a resume that could never work. -saves none and a
	// machine with no data directory both land here.
	var peek func(shell.House) (saved.Info, error)
	if a.saves != nil {
		peek = a.savedGame
	}

	return shell.Host{
		Screen: scr,

		// The application's art, with no house resource fork open. The shell's chrome
		// is the application's even when a house redefines the same PICT ids: see
		// render.Assets.UI.
		Assets: render.NewAssets(artFS),

		Present: func() {
			scr.ToBGRX(a.fb.Pix, a.fb.Stride)
			if err := a.win.Present(a.fb); err != nil {
				fmt.Fprintf(os.Stderr, "glidergo: %v\n", err)
				dead = true
			}
			// The mixer's pacer, for the same reason play.go's Present calls it,
			// and now for two: it is what makes the title screen's score audible
			// (music.go), and it would still have to be called for a silent one or
			// the tail of the last game's audio would sit in the buffer unplayed.
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
			// The title screen's score does not cross into a game: the game gets
			// its own cursor and its own two committed pieces, and both queues
			// coming out of one channel is the one thing here a player could hear
			// going wrong. music.go has the whole trade.
			a.stopTitleMusic()
			// The house's own filesystem, not its Path: Path is a message
			// ("built-in:houses/Titanic.house" for a house inside the executable) and
			// FS-and-Rel are what open the file. The FS travels on the house because
			// a library is a union of roots now and the houses root is not the only
			// one a listed house can have come from. See shell.House.
			out, err := a.play(libraryHouse(c.House), c.TwoPlayer, c.Resume)
			a.startTitleMusic()
			return out, err
		},

		// The board the High Scores screen shows: the side-car over the house file's own
		// rows. The shell caches whatever this answers and drops the cache after every
		// game, so this is a file read per house per visit and not per frame.
		Scores: a.board,

		// What the "Open Saved Game..." row offers, and what the band says under it: this
		// installation's save for the house, or the game the house file itself carries.
		// Cached by the shell per house per visit and dropped after every game, so this is
		// a header read and not a file read per frame -- see app.savedGame.
		Saved: peek,

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
			// Music on the title screen is the second setting that has to be
			// pushed, and the first whose effect the player is listening to while
			// the screen is still up: the row turns the score on and off where it
			// stands. It follows the volume rather than leading it because at
			// volume zero the mixer refuses to start the score at all, so raising
			// the volume and starting in that order is what makes a mute
			// recoverable in one keystroke.
			a.startTitleMusic()
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

// houseRef is one house and where to read it from.
//
// FS and Rel are the pair that opens the file: one of the houses roots -- a directory, or a
// subtree of the copy built into this executable -- and the name within it. FS nil means Path
// is a file on this machine and nothing but Path opens it, which is the case a player who typed
// a path is in.
//
// Name is the house's name, its file name without the extension, and it is not decoration:
// the high-score side-car, the saved game and the house's own 'snd ' resources are all keyed
// on it, so two copies of Slumberland in two directories share a score board on purpose.
type houseRef struct {
	Name string
	FS   fs.FS
	Rel  string
	Path string // what a message calls it; a real path only when FS is nil
}

// open reads the house.
func (r houseRef) open() (*house.House, error) {
	if r.FS == nil {
		return house.LoadFile(r.Path)
	}
	return house.LoadFS(r.FS, r.Rel)
}

// peek reads its 866-byte header and no rooms.
func (r houseRef) peek() (*house.Summary, error) {
	if r.FS == nil {
		return house.PeekFile(r.Path)
	}
	return house.PeekFS(r.FS, r.Rel)
}

// libraryHouse is a house the picker listed, in the root it was found in. It takes no
// filesystem because the house carries its own: a library is a union of roots, and the one the
// caller happens to have at hand is not necessarily the one this house came out of.
func libraryHouse(h shell.House) houseRef {
	return houseRef{Name: h.Name, FS: h.FS, Rel: h.Rel, Path: h.Path}
}

// inSources looks for one file name in each houses root in turn and answers the first that has
// it. The order is sources' order, so an original beats a new house of the same name -- which
// is the tie the picker's sort breaks the same way (internal/shell/library.go).
func inSources(o *options, rel string) (houseRef, bool) {
	for _, src := range o.sources() {
		if assetfs.Exists(src.FS, rel) {
			return houseRef{Name: houseName(rel), FS: src.FS, Rel: rel,
				Path: assetfs.Name(src.Label, rel)}, true
		}
	}
	return houseRef{}, false
}

// diskHouse is a house named by a path on this machine: no filesystem, so nothing but the path
// opens it, and the name is the path's own.
func diskHouse(path string) houseRef {
	return houseRef{Name: houseName(path), Path: path}
}

// resolveHouse turns whatever -house was given into one of those. A name is looked up in every
// houses root; anything with a separator or an extension in it is taken as a path, so a house
// sitting anywhere on the disk can be played without moving it.
//
// **Every root, and not just the houses one**, because the picker lists the new houses beside
// the originals and a flag that could not name one of them would make -house a way of playing
// some of the list. That also means -house is the one place a level set is invisible: a house
// is a house, and which collection it came from is a thing the picker says and not a thing that
// decides whether it can be played.
//
// The extension test needs the fallback below, because `-house Titanic.house` is what somebody
// who has just listed the houses will type, and it has an extension and no separator -- so the
// rule above resolved it against the working directory and the game said `open Titanic.house:
// no such file or directory` while the file sat where it was always going to be. A name with a
// separator in it is still taken at its word: there the player has said where to look.
func resolveHouse(o *options, name string) (houseRef, error) {
	if strings.ContainsRune(name, filepath.Separator) {
		return diskHouse(name), nil
	}

	if filepath.Ext(name) != "" {
		if _, err := os.Stat(name); err == nil {
			return diskHouse(name), nil
		}
		if ref, ok := inSources(o, name); ok {
			return ref, nil
		}
		// Neither. Report the path the player named rather than the one they did not, so
		// the error names something they can go and look for.
		return diskHouse(name), nil
	}

	rel := name + ".house"
	if ref, ok := inSources(o, rel); ok {
		ref.Name = name // the name as typed, so the score board and the saved game key on it
		return ref, nil
	}

	fsys, label := o.housesRoot()
	if fsys == nil {
		// No houses root at all: every flag was empty and this build carries no assets.
		// Saying so beats `open Slumberland.house: no such file or directory`, which sends
		// the reader looking in the working directory for a file that was never there.
		return houseRef{}, fmt.Errorf("no houses to look for %s in: this build has none "+
			"built in, so -houses has to name a directory", name)
	}
	// Nowhere has it. Point at the houses root anyway rather than erroring here, so the
	// failure comes out of the open with the path in it -- which is the message that says
	// where the game looked, and is what it said before there was more than one root.
	return houseRef{Name: name, FS: fsys, Rel: rel, Path: assetfs.Name(label, rel)}, nil
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
//	-wav alone          write the file and open no output. Recording is the intent, and
//	                    starting a player as well would be a surprise on a build machine.
//	-audio and -wav     both, through a Tee: play it and keep what was played.
//	neither             the best output there is -- the native device where the platform has
//	                    one, the first installed player otherwise (audio.Open). An empty answer
//	                    is not an error here, because a host can genuinely have no sound card,
//	                    so the caller warns and plays on.
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
		out, err := audio.Open(prefer)
		if err != nil {
			if len(sinks) == 0 {
				return &audio.Discard{}, "none", err
			}
			// A WAV is already open, so the session is not silent and the missing
			// output is worth less than the recording: report it and keep the file.
			return sinks[0], where[0], err
		}
		sinks = append(sinks, out)
		where = append(where, out.Name())
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
// Four of these numbers are the ones a bug report needs and cannot get any other way.
// *refused* is the channel policy doing its job -- three channels and a busy room -- and a
// large number there is not a defect. *skipped* is the frame loop having stalled for more than
// four frames, which is a game problem wearing an audio problem's clothes. *dropped* is the
// output having stopped taking samples, which is the device's problem or the machine's, and
// *gaps* is the sound device having run dry, which is the same stall heard from the far end.
// They have four different fixes, which is why they are four different counters.
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
	if s := streamIn(sink); s != nil {
		if n := s.Dropped(); n > 0 {
			line += fmt.Sprintf(", %d samples dropped by %s", n, s.Name())
		}
		// Underruns are the native device's to report; an external player's buffer hides
		// them, which is why this asks rather than assuming the method is there.
		if u, ok := s.(interface{ Underruns() int64 }); ok {
			if n := u.Underruns(); n > 0 {
				line += fmt.Sprintf(", %d gaps at %s", n, s.Name())
			}
		}
		if err := s.Err(); err != nil {
			line += fmt.Sprintf(", %s stopped taking samples (%v)", s.Name(), err)
		}
	}
	if pump.Err != nil {
		line += fmt.Sprintf(", sink error (%v)", pump.Err)
	}
	fmt.Println(line)
}

// streamIn finds the output inside whatever openSink built, or nil if there is none.
//
// The Tee case is the one that matters: with `-audio` and `-wav` together the sink is a Tee, and
// asking it directly used to find nothing -- so the run that was recording *because* the sound was
// wrong was the one run whose drop and gap counters went unreported.
func streamIn(sink audio.Sink) audio.Stream {
	switch s := sink.(type) {
	case audio.Stream:
		return s
	case audio.Tee:
		for _, inner := range s {
			if found := streamIn(inner); found != nil {
				return found
			}
		}
	}
	return nil
}
