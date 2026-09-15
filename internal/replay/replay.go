// Package replay runs the game headlessly from a script and reports, one line per frame,
// what the simulation did.
//
// It exists because of what a released game's bug report has to be able to say. "The
// glider gets stuck in the third room of Leviathan sometimes" is not reproducible; a seed
// and a keystroke log is, exactly, on any machine, with no display and no sound. That is
// releasePolish 4 in docs/analysis/stage15-raw/plans/spec-normative.md, and it is the
// reason this is a package with a command in front of it (`glidertool replay`) rather than
// a helper inside a _test.go file.
//
// The design has one rule: **nothing here may affect the simulation.** The harness sets
// World.KeyPoll and wraps World.Present, which are the two hooks a host is supposed to
// own, and reads counters afterwards. It does not reach into the game's state, and the
// script cannot express anything a player could not do -- with one deliberate exception,
// the resume start point, which is how a replay can begin in the room where the bug is
// instead of walking there for four minutes.
//
// # Determinism
//
// Three things have to be pinned or two runs of the same script diverge:
//
//	the random seed          World.Rand is a field for this reason (world.go:59-64)
//	the calendar clock       DrawCalendar reads the month and DrawRedClock the hands
//	the frame clock          nil TickCount makes Ticks derive from Frame, so the run is
//	                        unpaced and the game keeps nominal time regardless of the
//	                        wall clock -- see World.TickCount and awaitFrame
//
// The first two are script fields with fixed defaults. The third is the absence of a
// hook, which is why Run installs neither TickCount nor WaitTick.
//
// What is deliberately not pinned *in the trace* is the screen. Every pixel is still
// composed into World.Main -- a headless run and a windowed run produce byte-identical
// surfaces, which is World.Present's contract -- but the samples digest the simulation, not
// the image. Hashing the framebuffer per frame is a better test and a much worse bug-report
// format: a one-pixel change in a later sub-stage would invalidate every script on file. The
// samples below are the quantities the spec asks for (§5.3) plus the two diagnostics from
// game.Diagnostics, and they change only when behaviour does.
//
// The picture is reported once, at the end, as Result.Planes -- a hash of the three index
// planes as the last frame left them. That is a run-to-run comparison and not a checked-in
// constant, which is the distinction that makes it safe to have both: no file on disk holds
// the value, so a deliberate pixel change costs nothing, while two runs of one script are
// still required to agree on it.
package replay

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"glidergo/internal/game"
	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/render"
)

// DefaultClock is the instant a script runs at unless it says otherwise.
//
// It is Glider PRO's own release month, and the value matters only in that it must not be
// time.Now(): the calendar object draws the month name and the red clock draws the hands,
// so a run in September and a run in March compose different pixels. A zero time.Time
// would also be deterministic and would draw JANUARY at midnight, which is a state no
// player is ever in; a real date is the same cost and reads as intentional.
var DefaultClock = time.Date(1994, time.September, 14, 10, 9, 0, 0, time.UTC)

// Script is a replay: what to load, where to start, and which keys are held when.
//
// The zero value is not runnable -- House is required and Frames must be positive -- and
// NewScript is what fills in the defaults.
type Script struct {
	// House is a house name (looked up in HouseDir) or a path ending in ".house".
	House string

	// The three asset roots. Empty means the defaults below, which are the layout
	// `make assets` produces and the one cmd/glidergo defaults to.
	HouseDir    string
	ArtDir      string
	HouseArtDir string

	Seed      int32 // the random stream; 0 is a legal seed here, unlike in cmd/glidergo
	Frames    int   // how many simulated frames to run before quitting
	Neighbors int   // 1, 3 or 9: how much of the house is composed around the player
	TwoPlayer bool
	Clock     time.Time // the calendar's month and the red clock's hands

	// Room and Where are the start override, and they are the one thing in a script a
	// player could not have done.
	//
	// Room < 0 starts the house the way the player would: NewGameMode from
	// House.FirstRoom at House.Initial.
	//
	// Room >= 0 with a zero Where sets House.FirstRoom instead, which is what
	// `glidergo -room` does -- the glider still begins at the house's authored initial
	// point, so this is only useful for rooms where that point is inside the room.
	//
	// Room >= 0 with a nonzero Where resumes: the game starts in ResumeGameMode with a
	// synthesised saved game, which is the supported way into an arbitrary room at an
	// arbitrary point (see World.SavedGame and InitGlider's resume arm). Gliders and
	// Stars fill the two saved-game fields that change how the game ends.
	Room  int16
	Where house.Point

	// Facing is the saved game's byte: nonzero is right, which is what a new game
	// starts as, and 0 is left. It is used verbatim, so the zero value of a Script
	// means left -- NewScript is what supplies the right-facing default.
	Facing byte

	// Gliders and Stars are the resume path's kSavedGame fields, and here 0 does mean
	// "use the default": InitialGliders, and the house's own star count. That is safe
	// in a way it is not for Facing -- a run starting with no lives is not a run, and
	// a house with no stars counts 0 anyway.
	Gliders int16
	Stars   int16

	// Input is the keystroke log: a timeline of *changes*, not of presses. The entry
	// with the largest Frame at or below the current frame is the one in force, so a
	// player who holds right for two seconds is one line, and a script with no entries
	// at all is a run with every key up.
	//
	// Run sorts a copy by Frame, so a hand-written script need not be ordered.
	Input []Hold
}

// Hold is one entry in a keystroke timeline: from this frame on, these keys are down.
type Hold struct {
	Frame  int64
	P1, P2 player.Keys
}

// NewScript returns a runnable script for a house with every default filled in.
func NewScript(houseName string, frames int) *Script {
	return &Script{
		House:       houseName,
		HouseDir:    "assets/extracted/houses",
		ArtDir:      "assets/extracted/art",
		HouseArtDir: "assets/extracted/houseart",
		Frames:      frames,
		Neighbors:   9,
		Clock:       DefaultClock,
		Room:        -1,
		Facing:      1,
	}
}

// Sample is one frame of the trace.
//
// It is sampled at the *last* Present of each frame, which is not the same as the first
// and matters exactly once: on a transition frame the game renders twice, and only the
// second render's counters describe the frame the player was left looking at. See Run.
type Sample struct {
	Frame     int64
	Even      bool  // World.EvenFrame, which is a stored flag and not Frame&1
	Work2Main int   // len(World.Work2Main) as CopyRectsQD replayed it
	Back2Work int   //
	Renders   int64 // World.RenderFrames: 2 more than the last sample on a transition
	Pendulums int   // len(Scene.Pendulums): the pendulum was composed into the locale

	// ClockFrame is Scene.ClockFrame, the pendulum phase counter, and it is the one
	// field here that is not monotonic and not a count.
	//
	// It advances only in rooms that have a pendulum -- RenderPendulums returns before
	// the increment otherwise -- and the swing fires at exactly 10 and exactly 15. **The
	// sampled values are 0..14 and never 15**, because the frame that reaches 15 resets
	// it to 0 before returning. So `clock=0` and `clock=10` are the two frames in fifteen
	// the clock moved on and everything else is a stall, which is how the *uneven*
	// tick-tock shows up in the trace: the gaps alternate five frames and ten. A port
	// that tidied the two magic numbers into one interval would appear here as a column
	// stepping 0..9 forever. See game/anim.go's RenderPendulums.
	//
	// On a transition frame it advances by two, like Renders, because the pendulum is
	// stepped once per render and not once per game frame.
	//
	// It is also the pendulum's only observable phase. Anim.Mode is not sampled, and
	// three cels ping-ponging produce the same work rect every time, so `w2m` cannot
	// distinguish a swing from a stall.
	ClockFrame int16

	Room    int16
	Mode    int16 // player one's glider mode
	Dest    player.Rect
	Score   int32
	Mortals int16
	Stars   int16
	Guarded int64 // World.Diag.Guarded, cumulative
	Dropped int64 // dropped work + back rects, cumulative

	// Rand is World.RandSeed: the state of the one random stream.
	//
	// This is the strongest determinism claim in the trace and the only field that is
	// not a fact about the picture. Every flame phase, pendulum start, balloon jitter,
	// telephone ring and wind chime draws from that stream in composition order, so two
	// runs whose Rand agrees on all 600 frames consumed the same sequence in the same
	// order -- which is a much sharper statement than "the glider ended up in the same
	// place". It is also what makes the seed observable at all: a glider left alone in a
	// room may land identically under any seed, and without this the trace could not
	// tell a pinned run from an unseeded one.
	Rand int32
}

// line is the sample's canonical one-line form. It is what the digest hashes and what
// `glidertool replay -trace` prints, and the two must be the same text or a digest
// mismatch cannot be diffed.
func (s Sample) line() string {
	return fmt.Sprintf(
		"f=%d even=%d w2m=%d b2w=%d rend=%d pend=%d clock=%d room=%d mode=%d dest=%d,%d,%d,%d score=%d mortals=%d stars=%d guarded=%d dropped=%d rand=%d",
		s.Frame, b2i(s.Even), s.Work2Main, s.Back2Work, s.Renders, s.Pendulums, s.ClockFrame,
		s.Room, s.Mode, s.Dest.Top, s.Dest.Left, s.Dest.Bottom, s.Dest.Right,
		s.Score, s.Mortals, s.Stars, s.Guarded, s.Dropped, s.Rand)
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Result is what a run produced.
type Result struct {
	Script  *Script
	Samples []Sample

	// The end state, which is the part a bug report leads with.
	Frames    int64
	GameOver  bool
	Score     int32
	StarsLeft int16
	Mortals   int16
	Room      int16
	Diag      game.Diagnostics

	// Digest is a hex SHA-256 of every sample line, truncated to 16 characters. Two
	// runs of the same script agree on it or the port is not deterministic; two
	// *different* scripts agreeing on it is meaningless, so it is always reported
	// beside the script it came from.
	Digest string

	// Planes hashes the three index planes as the last frame left them: the pixel half
	// of determinism, which Digest cannot state.
	//
	// It is **not** folded into Digest and not written into Trace, for the reason the
	// package comment gives -- a one-pixel change in a later sub-stage would invalidate
	// every script on file, and a script is meant to outlive the sub-stage it was
	// recorded in. It is reported separately instead, where a caller that wants to
	// compare pictures can ask for it and a caller comparing behaviour is unaffected.
	Planes Planes
}

// Planes is a hash of each of the run's surfaces at the moment it stopped.
//
// Three rather than one, because they answer different questions. Back is the composition:
// a difference there is a locale that was drawn differently, before anything moved. Work is
// Back plus everything the frame loop blitted over it -- the flames, the gliders, the
// confetti -- so a difference in Work with Back identical is the *animation* diverging.
// Main is what a display would have been handed, which is Work as the dirty rects delivered
// it, so a difference in Main with Work identical is a dirty-rect bug: the right pixels
// composed and the wrong ones copied.
type Planes struct {
	Main string // World.Main, at the screen's size
	Work string // Scene.Work, the per-frame composite
	Back string // Scene.Back, the static room under it
}

// planeDigest hashes a surface's index plane, dimensions first.
//
// The indices are hashed and not the RGB, for internal/render's reason: the indices are what
// the game composites and what the palette maps, so two different indices that happen to
// share a colour are still a divergence. The dimensions go in because a surface that came
// back the wrong size would otherwise hash equal to a correctly sized one whose extra rows
// happened to be the white a fresh GWorld holds.
func planeDigest(s *render.Surface) string {
	if s == nil {
		return ""
	}
	sum := sha256.New()
	fmt.Fprintf(sum, "%dx%d\n", s.W, s.H)
	sum.Write(s.Pix)
	return hex.EncodeToString(sum.Sum(nil))[:16]
}

// Run plays a script and returns its trace.
func Run(s *Script) (*Result, error) {
	if s.House == "" {
		return nil, fmt.Errorf("replay: no house")
	}
	if s.Frames <= 0 {
		return nil, fmt.Errorf("replay: frames must be positive, have %d", s.Frames)
	}
	switch s.Neighbors {
	case 1, 3, 9:
	default:
		return nil, fmt.Errorf("replay: neighbors must be 1, 3 or 9, have %d", s.Neighbors)
	}

	path := s.House
	if filepath.Ext(path) != ".house" {
		path = filepath.Join(s.HouseDir, path+".house")
	}
	h, err := house.LoadFile(path)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSuffix(filepath.Base(path), ".house")

	assets := render.NewAssets(s.ArtDir)
	if s.HouseArtDir != "" {
		fork := filepath.Join(s.HouseArtDir, name)
		if st, err := os.Stat(fork); err == nil && st.IsDir() {
			assets.OpenHouseResFork(fork)
		}
	}

	scene := render.NewScene(render.DefaultView(), assets, h)
	scene.NumNeighbors = s.Neighbors
	scene.Clock = s.Clock

	w := game.NewWorld(h, scene, s.Seed)
	w.TwoPlayer = s.TwoPlayer

	// DoBackground false is the original's own default (Main.c:186) and is what a headless
	// run wants: PlayGame's event-pump block is skipped entirely, so nothing outside the
	// simulation can advance or stall a frame. With World.PlayEvent nil -- which it is
	// here, there being no host -- the two settings are indistinguishable anyway, because
	// HandlePlayEvent returns immediately and SwitchedOut can never become true.
	w.DoBackground = false

	// No TickCount and no WaitTick. That is not an omission: with no tick source
	// awaitFrame skips its wait and Ticks answers from the frame counter, so the game
	// keeps nominal time (Frame * TicksPerFrame) while the wall clock runs free. A replay
	// is therefore as fast as the machine allows and gives the same answer on a slow one.

	// One poll, two gliders, from the timeline. The C snapshots the hardware once inside
	// player one's GetInput and lets player two read the snapshot; a timeline lookup has
	// the same property for free, because both calls in a frame see the same World.Frame.
	holds := make([]Hold, len(s.Input))
	copy(holds, s.Input)
	sort.SliceStable(holds, func(i, j int) bool { return holds[i].Frame < holds[j].Frame })
	w.KeyPoll = func(g *player.Glider) player.Keys {
		k := keysAt(holds, w.Frame)
		if g.Which == player.Player2 {
			return k.P2
		}
		return k.P1
	}

	res := &Result{Script: s}

	// The sampler. Present is the *sampling point* and Frame is the quantity, and they
	// are not the same number: a transition presents once per wipe strip (116 or 160
	// times inside one frame) and HideGlider presents on its own. So each frame's record
	// is overwritten until the frame changes, which leaves the *last* present of the
	// frame in the trace.
	//
	// That choice is what makes the transition frame legible. On the frame a glider takes
	// a door, Transit.c renders once from inside MoveRoomToRoom and PlayGame renders
	// again at the bottom of the loop, so Renders advances by two while Frame advances by
	// one -- and it is the second render whose dirty-rect lists describe what the player
	// is looking at. Sampling the first present would report the middle of the wipe.
	var cur Sample
	have := false
	flush := func() {
		if have {
			res.Samples = append(res.Samples, cur)
			have = false
		}
	}
	w.Present = func() {
		if have && cur.Frame != w.Frame {
			flush()
		}
		cur = Sample{
			Frame:      w.Frame,
			Even:       w.EvenFrame,
			Work2Main:  len(w.Work2Main),
			Back2Work:  len(w.Back2Work),
			Renders:    w.RenderFrames,
			Pendulums:  len(scene.Pendulums),
			ClockFrame: scene.ClockFrame,
			Room:       w.R.RoomNumber,
			Mode:       w.P1.Mode,
			Dest:       w.P1.Dest,
			Score:      w.Score,
			Mortals:    w.Mortals,
			Stars:      w.StarsLeft,
			Guarded:    w.Diag.Guarded,
			Dropped:    w.Diag.DroppedWorkRects + w.Diag.DroppedBackRects,
			Rand:       w.RandSeed,
		}
		have = true

		if w.Frame >= int64(s.Frames) {
			// The same door the Quit menu item uses. SwitchedOut is cleared beside it
			// because PlayGame's pump loop spins on that flag alone.
			w.Quitting = true
			w.SwitchedOut = false
		}
	}

	mode := game.NewGameMode
	if s.Room >= 0 {
		if int(s.Room) >= len(h.Rooms) {
			return nil, fmt.Errorf("replay: room %d out of range (%s has %d)", s.Room, name, len(h.Rooms))
		}
		if s.Where == (house.Point{}) {
			h.FirstRoom = s.Room
		} else {
			mode = game.ResumeGameMode
			gliders := s.Gliders
			if gliders == 0 {
				gliders = game.InitialGliders
			}
			stars := s.Stars
			if stars == 0 {
				stars = w.CountStarsInHouse()
			}
			w.SavedGame = house.Game{
				RoomNumber:   s.Room,
				Where:        s.Where,
				NumGliders:   gliders,
				WasStarsLeft: stars,
				GliderState:  player.GliderNormal,
				// Facing verbatim, with no zero-means-default fixup: 0 *is* left
				// (player.FaceLeft is false), so treating it as "unset" would make
				// `facing left` silently start the glider facing right. The default
				// lives in NewScript instead, which is the only place a Script is
				// meant to be created.
				Facing: s.Facing,
			}
			// NewGame's resume arm skips SetObjectsToDefaults, because a real resume is
			// restoring object state that the saved game carried. A synthesised one has
			// none, so the objects have to be defaulted here or every clock in the house
			// starts collected.
			w.SetObjectsToDefaults()
		}
	}

	w.NewGame(mode)
	flush()

	res.Frames = w.Frame
	res.GameOver = w.GameOver
	res.Score = w.Score
	res.StarsLeft = w.StarsLeft
	res.Mortals = w.Mortals
	res.Room = w.R.RoomNumber
	res.Diag = w.Diag
	res.Digest = digest(res.Samples)

	// The surfaces, read after the run rather than sampled during it. Present is where a
	// per-frame hash would have to go and it fires up to 160 times in a transition frame,
	// so hashing there would cost 160 sweeps of a 640x480 plane on those frames and pin the
	// middle of a wipe. The end state is the cheap statement and the sharp one: 1,200 frames
	// of divergence cannot reconverge to the same three planes by accident.
	res.Planes = Planes{
		Main: planeDigest(w.Main),
		Work: planeDigest(scene.Work),
		Back: planeDigest(scene.Back),
	}

	// Sticky asset errors, reported once at the end. A replay against a half-extracted
	// asset tree would otherwise produce a plausible trace of a game drawing nothing.
	if err := assets.Err(); err != nil {
		return res, err
	}
	return res, nil
}

// keysAt is the timeline lookup: the last entry at or below frame, or all keys up.
func keysAt(holds []Hold, frame int64) Hold {
	var out Hold
	for _, h := range holds {
		if h.Frame > frame {
			break
		}
		out = h
	}
	return out
}

func digest(samples []Sample) string {
	sum := sha256.New()
	for _, s := range samples {
		fmt.Fprintln(sum, s.line())
	}
	return hex.EncodeToString(sum.Sum(nil))[:16]
}

// Trace writes the whole trace: a header naming the script, then one line per frame.
//
// The frame lines are exactly the text the digest hashed, so a mismatch between two runs
// is a diff between two of these.
func (r *Result) Trace(out io.Writer) error {
	bw := bufio.NewWriter(out)
	fmt.Fprintf(bw, "# gliderGo replay trace\n")
	fmt.Fprintf(bw, "# house=%s seed=%d frames=%d neighbors=%d players=%d\n",
		r.Script.House, r.Script.Seed, r.Script.Frames, r.Script.Neighbors, players(r.Script))
	if r.Script.Room >= 0 {
		fmt.Fprintf(bw, "# start room=%d where=%d,%d\n", r.Script.Room, r.Script.Where.H, r.Script.Where.V)
	}
	fmt.Fprintf(bw, "# digest=%s\n", r.Digest)
	for _, s := range r.Samples {
		fmt.Fprintln(bw, s.line())
	}
	fmt.Fprintf(bw, "# end frames=%d gameOver=%v score=%d stars=%d mortals=%d room=%d\n",
		r.Frames, r.GameOver, r.Score, r.StarsLeft, r.Mortals, r.Room)
	fmt.Fprintf(bw, "# diag guarded=%d droppedWork=%d droppedBack=%d\n",
		r.Diag.Guarded, r.Diag.DroppedWorkRects, r.Diag.DroppedBackRects)
	for _, d := range r.Diag.Seen {
		fmt.Fprintf(bw, "# deviation %s\n", d)
	}
	return bw.Flush()
}

func players(s *Script) int {
	if s.TwoPlayer {
		return 2
	}
	return 1
}

// ---------------------------------------------------------------------------
// The text format
// ---------------------------------------------------------------------------
//
// A script is line-oriented, `#` to end of line is a comment, and every line is a keyword
// and its arguments. That is the same shape as the house text format `glidertool house
// dump` writes, and for the same reason: a bug report is a file somebody pastes into an
// email, and a diff between two of them should be readable.
//
//	house CD Demo House      the house, by name or by path
//	seed 12345               the random stream
//	frames 600               how long to run
//	neighbors 9              1, 3 or 9
//	players 2                1 or 2
//	room 4                   start room; with `where`, a resume
//	where 423 20             h then v, the glider's top-left in room coordinates
//	facing right             or left
//	gliders 2                the resume path's mortal count
//	stars 5                  the resume path's remaining stars
//	clock 1994-09-14T10:09:00Z
//	at 0 right               from frame 0, player one holds right
//	at 45 - band             player one lets go, player two fires a band
//
// The key names are the fields of player.Keys, lower-cased: left, right, batt, band,
// command, delete, pause. Several are joined with commas and `-` means none. An `at` line
// takes player one's keys and then, optionally, player two's.

// keyNames is the whole vocabulary, in the order FormatKeys emits. Each entry carries
// both directions so that ParseKeys and FormatKeys cannot drift apart: adding a key to
// player.Keys means adding one line here, and the round-trip test then covers it.
var keyNames = []struct {
	name string
	set  func(*player.Keys)
	get  func(player.Keys) bool
}{
	{"left", func(k *player.Keys) { k.Left = true }, func(k player.Keys) bool { return k.Left }},
	{"right", func(k *player.Keys) { k.Right = true }, func(k player.Keys) bool { return k.Right }},
	{"batt", func(k *player.Keys) { k.Batt = true }, func(k player.Keys) bool { return k.Batt }},
	{"band", func(k *player.Keys) { k.Band = true }, func(k player.Keys) bool { return k.Band }},
	{"command", func(k *player.Keys) { k.Command = true }, func(k player.Keys) bool { return k.Command }},
	{"delete", func(k *player.Keys) { k.Delete = true }, func(k player.Keys) bool { return k.Delete }},
	{"pause", func(k *player.Keys) { k.Pause = true }, func(k player.Keys) bool { return k.Pause }},
}

// ParseKeys reads one key field: "-" or "none" for all up, otherwise comma-separated
// names.
func ParseKeys(s string) (player.Keys, error) {
	var k player.Keys
	if s == "" || s == "-" || s == "none" {
		return k, nil
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		found := false
		for _, kn := range keyNames {
			if kn.name == part {
				kn.set(&k)
				found = true
				break
			}
		}
		if !found {
			return k, fmt.Errorf("unknown key %q", part)
		}
	}
	return k, nil
}

// FormatKeys is ParseKeys' inverse, and round-trips.
func FormatKeys(k player.Keys) string {
	var on []string
	for _, kn := range keyNames {
		if kn.get(k) {
			on = append(on, kn.name)
		}
	}
	if len(on) == 0 {
		return "-"
	}
	return strings.Join(on, ",")
}

// Parse reads a script in the text format. Unknown keywords are an error rather than a
// warning: a typo in a bug report's script must not silently run a different game.
func Parse(in io.Reader) (*Script, error) {
	s := NewScript("", 0)
	sc := bufio.NewScanner(in)
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		if i := strings.IndexByte(text, '#'); i >= 0 {
			text = text[:i]
		}
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		bad := func(err error) error { return fmt.Errorf("line %d: %s: %w", line, fields[0], err) }
		num := func(i int) (int64, error) {
			if i >= len(fields) {
				return 0, fmt.Errorf("missing argument")
			}
			return strconv.ParseInt(fields[i], 10, 64)
		}
		switch fields[0] {
		case "house":
			// Unquoted and space-bearing: "CD Demo House" is a real file name, so the
			// rest of the line is the value.
			s.House = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "house"))
			if s.House == "" {
				return nil, bad(fmt.Errorf("missing argument"))
			}
		case "housedir":
			s.HouseDir = fields[1]
		case "artdir":
			s.ArtDir = fields[1]
		case "houseartdir":
			s.HouseArtDir = fields[1]
		case "seed":
			n, err := num(1)
			if err != nil {
				return nil, bad(err)
			}
			s.Seed = int32(n)
		case "frames":
			n, err := num(1)
			if err != nil {
				return nil, bad(err)
			}
			s.Frames = int(n)
		case "neighbors":
			n, err := num(1)
			if err != nil {
				return nil, bad(err)
			}
			s.Neighbors = int(n)
		case "players":
			n, err := num(1)
			if err != nil {
				return nil, bad(err)
			}
			if n != 1 && n != 2 {
				return nil, bad(fmt.Errorf("must be 1 or 2"))
			}
			s.TwoPlayer = n == 2
		case "room":
			n, err := num(1)
			if err != nil {
				return nil, bad(err)
			}
			s.Room = int16(n)
		case "where":
			h, err := num(1)
			if err != nil {
				return nil, bad(err)
			}
			v, err := num(2)
			if err != nil {
				return nil, bad(err)
			}
			s.Where = house.Point{H: int16(h), V: int16(v)}
		case "facing":
			switch {
			case len(fields) < 2:
				return nil, bad(fmt.Errorf("missing argument"))
			case fields[1] == "right":
				s.Facing = 1
			case fields[1] == "left":
				s.Facing = 0
			default:
				return nil, bad(fmt.Errorf("want left or right, have %q", fields[1]))
			}
		case "gliders":
			n, err := num(1)
			if err != nil {
				return nil, bad(err)
			}
			s.Gliders = int16(n)
		case "stars":
			n, err := num(1)
			if err != nil {
				return nil, bad(err)
			}
			s.Stars = int16(n)
		case "clock":
			if len(fields) < 2 {
				return nil, bad(fmt.Errorf("missing argument"))
			}
			t, err := time.Parse(time.RFC3339, fields[1])
			if err != nil {
				return nil, bad(err)
			}
			s.Clock = t
		case "at":
			f, err := num(1)
			if err != nil {
				return nil, bad(err)
			}
			hold := Hold{Frame: f}
			if len(fields) > 2 {
				if hold.P1, err = ParseKeys(fields[2]); err != nil {
					return nil, bad(err)
				}
			}
			if len(fields) > 3 {
				if hold.P2, err = ParseKeys(fields[3]); err != nil {
					return nil, bad(err)
				}
			}
			s.Input = append(s.Input, hold)
		default:
			return nil, fmt.Errorf("line %d: unknown keyword %q", line, fields[0])
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return s, nil
}

// Write writes a script in the text format Parse reads, and round-trips through it.
func (s *Script) Write(out io.Writer) error {
	bw := bufio.NewWriter(out)
	fmt.Fprintf(bw, "# gliderGo replay script\n")
	fmt.Fprintf(bw, "house %s\n", s.House)
	fmt.Fprintf(bw, "seed %d\n", s.Seed)
	fmt.Fprintf(bw, "frames %d\n", s.Frames)
	fmt.Fprintf(bw, "neighbors %d\n", s.Neighbors)
	fmt.Fprintf(bw, "players %d\n", players(s))
	fmt.Fprintf(bw, "clock %s\n", s.Clock.Format(time.RFC3339))
	if s.Room >= 0 {
		fmt.Fprintf(bw, "room %d\n", s.Room)
		if s.Where != (house.Point{}) {
			fmt.Fprintf(bw, "where %d %d\n", s.Where.H, s.Where.V)
			if s.Facing == 0 {
				fmt.Fprintf(bw, "facing left\n")
			} else {
				fmt.Fprintf(bw, "facing right\n")
			}
			if s.Gliders != 0 {
				fmt.Fprintf(bw, "gliders %d\n", s.Gliders)
			}
			if s.Stars != 0 {
				fmt.Fprintf(bw, "stars %d\n", s.Stars)
			}
		}
	}
	for _, h := range s.Input {
		if s.TwoPlayer {
			fmt.Fprintf(bw, "at %d %s %s\n", h.Frame, FormatKeys(h.P1), FormatKeys(h.P2))
		} else {
			fmt.Fprintf(bw, "at %d %s\n", h.Frame, FormatKeys(h.P1))
		}
	}
	return bw.Flush()
}
