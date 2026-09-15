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
//
// The checked-in version of that question lives one package over, in internal/fidelity,
// which drives this one through Watch to hash the picture at the end of every frame. It is
// separated on purpose: a reference corpus is a file that a deliberate pixel change has to
// be re-blessed against, and putting it in the trace format would make every script on file
// a hostage to the next sub-stage's improvements.
//
// # Sound
//
// The audio is in the trace twice over, and the split is the same one: what happened is in the
// samples, what it sounded like is reported at the end.
//
//	snd=              per frame, in the hashed line: every request, its priority, the channel
//	                  it got and what it cut off. This is the sharp half -- a sound that was
//	                  requested and refused is a bug report's whole content, and it is
//	                  invisible in a recording
//	Audio.Digest      once, at the end: a hash of the mix. Two runs of one script must agree
//	Audio.Stats       once, at the end: the counters, including how many samples clipped
//
// Mixing is deterministic for the same reason the simulation is -- audio.Pump.FrameTick asks
// for exactly one frame's worth of samples per frame from an engine with no clock and no
// goroutine, so the stream is a function of the script and nothing else. RunTo will hand that
// stream to a WAV, which is how the audio path is checked at all on a build host with no sound
// card: `glidertool replay -wav` writes a file and somebody listens to it somewhere else.
//
// Sound can be switched off, and doing so is **not** the same run with the volume down: a build
// with no sound system composes rooms differently, because a sound trigger whose sample cannot
// be loaded gets no hot spot. See Script.Sound.
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

	"glidergo/internal/audio"
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

	// The four asset roots. Empty means the defaults below, which are the layout
	// `make assets` produces and the one cmd/glidergo defaults to.
	HouseDir    string
	ArtDir      string
	HouseArtDir string
	SoundDir    string

	Seed      int32 // the random stream; 0 is a legal seed here, unlike in cmd/glidergo
	Frames    int   // how many simulated frames to run before quitting
	Neighbors int   // 1, 3 or 9: how much of the house is composed around the player
	TwoPlayer bool
	Clock     time.Time // the calendar's month and the red clock's hands

	// Sound and Music are the two audio preferences, used verbatim: the zero value of a
	// Script is silent, and NewScript is what turns them on. That is Facing's convention
	// rather than Gliders' (see both below), and it is the right way round here because a
	// hand-built Script in a test wants no dependency on the sound tree until it asks for one.
	//
	// **Sound off is not merely quieter, it is a different simulation**, which is why it is a
	// script field at all and not a flag on Run. A build with no sound system composes rooms
	// differently: LoadTriggerSound cannot claim the reserved sample slot, so a sound trigger
	// gets no hot spot and the kSoundIt rect a glider would have crossed is not there. That is
	// the original's own `dontLoadSounds` behaviour (Sound.c:265-303 through
	// game/hotspots.go's loadTriggerSound), so a fidelity corpus needs to be able to record
	// both -- and a trace of one is not comparable with a trace of the other.
	//
	// Music off leaves the effects alone: no MusicChannel is wired, so the score never starts
	// and nothing else changes. It is separate because the music chain is the one part of the
	// audio path that calls back into the game (NextMusicPiece walks the score cursor), and a
	// bug report about the effects should be able to switch it off.
	Sound bool
	Music bool

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
		SoundDir:    "assets/extracted/sound",
		Frames:      frames,
		Neighbors:   9,
		Clock:       DefaultClock,
		Room:        -1,
		Facing:      1,
		Sound:       true,
		Music:       true,
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

	// Sounds is every PlayPrioritySound request made on this frame and what the mixer did
	// with it, already rendered as the trace's `snd=` field: `-` for a silent frame,
	// otherwise `name:priority:channel` per request, joined with commas, with `>name`
	// appended when the grant cut something off and `x` in place of the channel when the
	// request was refused. So `shred:903:0>tik` is a shred taking channel 0 from a score tik.
	//
	// A rendered string rather than a slice of events, for two reasons. It keeps Sample
	// comparable, which is what lets the determinism test compare two runs frame by frame
	// with `!=` and report the frame rather than a diff of two slices. And it is the text
	// the digest hashes either way -- the structured form is on Result.Events, with the
	// frame number attached, for a test that wants to assert on the mixer's decisions.
	//
	// The requests are attributed to the frame they were *made* on, which is exact: the hook
	// reads World.Frame at the moment PlayPrioritySound is called. The *audio* for the frame
	// is mixed slightly later -- see the note in RunTo -- so the two can disagree by less
	// than a frame, and this column is the one to trust about ordering.
	Sounds string

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
	snd := s.Sounds
	if snd == "" {
		// A run with no sound system at all, and a frame that requested nothing, both read
		// as `-`. They are distinguished by the header's `audio` line, not per frame.
		snd = "-"
	}
	return fmt.Sprintf(
		"f=%d even=%d w2m=%d b2w=%d rend=%d pend=%d clock=%d room=%d mode=%d dest=%d,%d,%d,%d score=%d mortals=%d stars=%d guarded=%d dropped=%d rand=%d snd=%s",
		s.Frame, b2i(s.Even), s.Work2Main, s.Back2Work, s.Renders, s.Pendulums, s.ClockFrame,
		s.Room, s.Mode, s.Dest.Top, s.Dest.Left, s.Dest.Bottom, s.Dest.Right,
		s.Score, s.Mortals, s.Stars, s.Guarded, s.Dropped, s.Rand, snd)
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Event is one sound request, with the frame it was made on.
//
// The audio package's own Event stamps requests with the mixer's sample cursor, which is what a
// live host has; a replay knows the frame, which is what a bug report quotes.
type Event struct {
	Frame int64
	audio.Event
}

// Audio is what the run's audio path did, and it is a report rather than a fixture: every field
// is a count or a hash, and none of them is checked into a golden file except through the trace
// header.
type Audio struct {
	audio.Stats

	// Bank is how many bytes of samples were resident, and Triggers how many of them were the
	// house's own. A house whose trigger sounds failed to extract runs silently past every
	// sound trigger in it, and this is where that shows up.
	Bank     int
	Triggers int

	// Samples is what the pump produced. On this path it is FrameTick's exact arithmetic --
	// one frame's worth per recorded frame -- which is why the harness can assert it rather
	// than merely report it. See RunTo.
	Samples int64

	// Digest is a hash of the mix itself: sixteen hex characters over the little-endian bytes
	// a `-wav` run would have written. Two runs of one script must agree on it, and that is a
	// far stronger statement than the sample digest can make -- it covers every sample of
	// every channel, the priority policy's choice of channel, the clip point and the score.
	Digest string
}

// Result is what a run produced.
type Result struct {
	Script  *Script
	Samples []Sample

	// Events is every sound request the run made, in order, and the only place the mixer's
	// decisions survive in structured form. Empty for a run with sound off.
	Events []Event

	// Audio is the audio path's own report. Zero for a run with sound off, which is
	// distinguishable from a silent run with sound on by Audio.Samples being 0 rather than
	// the frame count times SamplesPerFrame.
	Audio Audio

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
	//
	// The sample line includes the frame's sound requests, so `sound off` gives a different
	// value here even where it gives the same gameplay -- which is deliberate. A sound request
	// is a decision the simulation made, at a frame, and a port that stopped playing the
	// toaster would otherwise pass every digest comparison in the suite. `music off` does not
	// change it, the score having no per-frame effect at all. Which of the two was in force is
	// in the trace's header for exactly this reason, and Audio.Digest is the separate claim
	// about the samples themselves.
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

// Watch observes the picture at the end of every recorded frame.
//
// It is how internal/fidelity gets a per-frame hash out of a run without this package
// growing an opinion about what a reference frame is. The three surfaces are the same
// three Planes reports, in the same order, and they hold the frame the sample describes:
// the state at the *last* Present of it, which for Work is the full composite before
// CopyRectsQD's Back2Work restore takes the animation back off again.
//
// Two rules, both because the surfaces handed over are a reused scratch copy rather than
// the game's own:
//
//   - They are valid for the duration of the call. A watcher that keeps one keeps a buffer
//     the next frame overwrites; Surface.Clone is what to do instead.
//   - Writing to them is pointless rather than dangerous -- they are copies, so the game
//     cannot see it -- but it makes the next frame's diff a lie. Read them.
//
// The scratch copy is the whole reason this is cheap enough to exist. See RunWatching.
type Watch func(sample Sample, main, work, back *render.Surface)

// shot is the scratch a Watch is handed: one reusable copy of each plane.
//
// Copying and then hashing once per frame is what makes a per-frame picture affordable at
// all. Present fires up to 160 times inside a transition frame and only the last of them
// describes what the player was left looking at, so a hash *at* Present would sweep three
// planes 160 times to keep one answer. A copy is several times cheaper than a hash of the
// same bytes, and this way the expensive half happens once per frame, in flush, on the
// snapshot the last Present left behind.
type shot struct {
	main, work, back *render.Surface
}

// take copies the three planes as they stand. A nil receiver is a run with no watcher and
// does nothing, which is what keeps the hook out of the way of every other caller.
func (sh *shot) take(main, work, back *render.Surface) {
	if sh == nil {
		return
	}
	sh.main = copyInto(sh.main, main)
	sh.work = copyInto(sh.work, work)
	sh.back = copyInto(sh.back, back)
}

// copyInto copies src into dst, reallocating only when dst is missing or the wrong size.
func copyInto(dst, src *render.Surface) *render.Surface {
	if src == nil {
		return nil
	}
	if dst == nil || dst.W != src.W || dst.H != src.H {
		dst = render.NewSurface(src.W, src.H)
	}
	copy(dst.Pix, src.Pix)
	// The masks of Main, Work and Back are all nil in practice -- they are destinations,
	// and only art carries a mask -- but a watcher that writes a PNG of one would produce
	// a differently transparent image if that ever stopped being true, so it is carried
	// rather than assumed away.
	switch {
	case src.Mask == nil:
		dst.Mask = nil
	case len(dst.Mask) != len(src.Mask):
		dst.Mask = append([]uint8(nil), src.Mask...)
	default:
		copy(dst.Mask, src.Mask)
	}
	return dst
}

// Run plays a script and returns its trace. The audio is mixed and hashed and then discarded,
// which is what a test wants; RunTo is how a caller keeps it.
func Run(s *Script) (*Result, error) { return RunWatching(s, nil, nil) }

// RunTo is Run with somewhere for the mix to go: a WAV, a player's stdin, an audio.Tee of both.
//
// A nil sink still mixes. That is not waste -- the mix is where a silent bug lives. A sound that
// was requested, granted a channel and then cut off after four milliseconds by a louder one on
// the same channel is a *correct* trace line and a wrong noise, and only the samples can tell the
// difference. Mixing always also means Audio.Digest is always available, so "is the audio
// deterministic" is answered by every run of every script rather than by the ones that asked.
//
// **RunTo closes the sink**, on every path out of this function. A WAV's header is patched on
// Close, so a harness that forgot would leave a file that says it holds no samples; and the
// alternative -- the caller closing after reading Result -- gets the order wrong for a Tee whose
// pipe wants draining before the report is printed. That includes the failures below and the case
// of a sink handed to a script with `sound off`, both of which produce a valid empty file rather
// than a truncated one. audio.WAV.Close is idempotent, so a `defer w.Close()` at the call site is
// still safe.
func RunTo(s *Script, sink audio.Sink) (*Result, error) { return RunWatching(s, sink, nil) }

// RunWatching is RunTo with a per-frame look at the picture. A nil watch is RunTo exactly:
// no snapshot is taken and no plane is copied.
func RunWatching(s *Script, sink audio.Sink, watch Watch) (*Result, error) {
	// Ownership of the sink passes to the pump when there is one; until then it is this
	// function's to close, including on the validation errors immediately below.
	owned := false
	if sink != nil {
		defer func() {
			if !owned {
				sink.Close()
			}
		}()
	}

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

	// soundsThisFrame is the sampler's view of the request log: the `snd=` field for the frame
	// being sampled. A run with no audio keeps this stub and every frame reads `-`.
	soundsThisFrame := func() string { return "" }

	// ---- audio ----
	//
	// The three effect hooks, the music channel and a pump. Everything here is off unless the
	// script asks for it, and a script that asks for it and cannot have it is an **error**: a
	// trace recorded against a half-extracted sound tree would be a plausible-looking file
	// whose `snd=` column is empty for a reason that has nothing to do with the game. That is
	// the same argument the sticky assets.Err() check at the bottom of this function makes,
	// made earlier because the bank is loaded before the run rather than during it.
	//
	// The pump is driven by FrameTick from flush() below -- exactly SamplesPerFrame per
	// recorded frame, from an engine with no clock -- which is what makes two runs of one
	// script produce the same bytes. cmd/glidergo drives ClockTick instead, and the two paths
	// share every line of the mixer.
	var eng *audio.Engine
	var pump *audio.Pump
	var mixHash *audio.Digest
	if s.Sound {
		dir := s.SoundDir
		if dir == "" {
			dir = "assets/extracted/sound"
		}
		bank, err := audio.LoadBank(dir)
		if err != nil {
			return nil, err
		}
		// The house's own trigger sounds, which is HouseIO.c's resource-fork swap. A house
		// with none is not an error; a broken manifest is.
		if err := bank.LoadHouse(name); err != nil {
			return nil, err
		}

		eng = audio.New(bank)
		mixHash = audio.NewDigest()
		var out audio.Sink = mixHash
		if sink != nil {
			out = audio.Tee{mixHash, sink}
		}
		pump = audio.NewPump(eng, out)
		owned = true

		// The request log. It keys on World.Frame at the moment of the call, which is why
		// the `snd=` column is exact about ordering even though the samples for the frame
		// are mixed a little later: the game requests a sound from inside the input pass or
		// the interaction pass, long before the frame's last Present.
		curFrame := int64(-1)
		var frameEvents []audio.Event
		eng.On = func(ev audio.Event) {
			if w.Frame != curFrame {
				curFrame, frameEvents = w.Frame, frameEvents[:0]
			}
			frameEvents = append(frameEvents, ev)
			res.Events = append(res.Events, Event{Frame: w.Frame, Event: ev})
		}
		soundsThisFrame = func() string {
			if curFrame != w.Frame {
				return ""
			}
			return formatSounds(frameEvents)
		}

		w.SoundPlayer = eng.PlayPrioritySound
		w.TriggerSoundExists = eng.LoadTriggerSound
		w.FlushTriggerSound = eng.FlushTriggerSound

		if s.Music {
			// Both preferences on and then InitMusic, which is the order cmd/glidergo
			// uses and the order Music.c's own InitMusic implies: it starts the splash
			// score if isPlayMusicIdle is already set.
			w.Music = eng
			eng.NextPiece = w.NextMusicPiece
			w.PlayMusicGame = true
			w.PlayMusicIdle = true
			w.InitMusic()
		}
	}

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
	var frame *shot
	if watch != nil {
		frame = &shot{}
	}
	flush := func() {
		if have {
			res.Samples = append(res.Samples, cur)
			have = false

			// The picture, to whoever asked for it, and before the audio for no reason
			// except that a watcher is the cheaper of the two to reason about: it sees
			// exactly the frame the sample describes, with none of the sub-frame offset the
			// mix below carries.
			if watch != nil {
				watch(cur, frame.main, frame.work, frame.back)
			}

			// One frame's audio per recorded frame, mixed here because this is the only
			// place in the harness that knows a frame is over.
			//
			// It is a *little* late, and the shift is worth stating: this runs at the
			// first Present of frame N+1, by which time N+1's input pass has already
			// asked for its sounds, so a sound requested on N+1 begins at the top of the
			// block that is nominally N's. The offset is under one frame -- 33
			// milliseconds -- it is constant, and it cannot reorder anything, because the
			// engine starts a sample the moment it is requested. The `snd=` column is
			// what carries exact per-frame attribution; this carries the samples.
			//
			// The alternative would be a hook at the bottom of PlayGame's loop, which
			// would be a second frame-boundary notion in World for the harness's benefit
			// alone. The package's one rule is that nothing here may affect the
			// simulation, and adding a hook to the loop to make a WAV 33 ms tidier is a
			// poor trade against it.
			//
			// pump is nil for a script that asked for silence, and every Pump method
			// tolerates a nil receiver on purpose (pump.go), so this needs no guard.
			pump.FrameTick()
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
			Sounds:     soundsThisFrame(),
			Rand:       w.RandSeed,
		}
		have = true

		// And the picture the counters above describe, for a run that asked for it. Taken
		// on every Present and overwritten, for the same reason cur is: the last one of the
		// frame is the one that describes it.
		frame.take(w.Main, scene.Work, scene.Back)

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
	//
	// A caller that wants the per-frame version anyway can have it -- that is what Watch is
	// -- and the cost argument is why it copies at Present and hashes in flush instead. The
	// two answers are not the same and both are worth having: this one is free and says
	// *whether*, a reference corpus costs a file on disk and says *when*.
	res.Planes = Planes{
		Main: planeDigest(w.Main),
		Work: planeDigest(scene.Work),
		Back: planeDigest(scene.Back),
	}

	// The audio report, after the last frame's mix and after the sink is closed so that a WAV
	// handed in by the caller has its header patched before anybody reads the file.
	if eng != nil {
		closeErr := pump.Close()
		res.Audio = Audio{
			Stats:    eng.Stats(),
			Bank:     eng.Bank().Bytes(),
			Triggers: len(eng.Bank().Triggers),
			Samples:  pump.Mixed(),
			Digest:   mixHash.Sum(),
		}
		// A sink error is reported and does not stop the run: the trace is still valid, and
		// "the WAV could not be written" is a different failure from "the game misbehaved".
		// The pump's own latched error comes first because it is the earlier one.
		if err := pump.Err; err != nil {
			return res, err
		}
		if closeErr != nil {
			return res, closeErr
		}
	}

	// Sticky asset errors, reported once at the end. A replay against a half-extracted
	// asset tree would otherwise produce a plausible trace of a game drawing nothing.
	if err := assets.Err(); err != nil {
		return res, err
	}
	return res, nil
}

// formatSounds renders one frame's requests as the `snd=` field. See Sample.Sounds.
func formatSounds(evs []audio.Event) string {
	if len(evs) == 0 {
		return ""
	}
	parts := make([]string, len(evs))
	for i, ev := range evs {
		ch := strconv.Itoa(ev.Channel)
		if ev.Channel < 0 {
			ch = "x" // refused: no channel was quiet enough
		}
		s := soundName(ev.Slot) + ":" + strconv.Itoa(int(ev.Priority)) + ":" + ch
		if ev.Displaced != audio.NoSoundPlaying {
			s += ">" + soundName(ev.Displaced)
		}
		parts[i] = s
	}
	return strings.Join(parts, ",")
}

// soundName is audio.Name with the spaces taken out, because the trace line is space-delimited
// and "score tik" would end a field in the middle of itself.
func soundName(slot int16) string {
	return strings.ReplaceAll(audio.Name(slot), " ", "-")
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
	// The audio settings go in the header and not the footer, because they change what the
	// samples below mean: sound off is a different composition (see Script.Sound), so a reader
	// comparing two traces has to know before the first frame line rather than after the last.
	fmt.Fprintf(bw, "# audio sound=%s music=%s\n", onOff(r.Script.Sound), onOff(r.Script.Music))
	fmt.Fprintf(bw, "# digest=%s\n", r.Digest)
	for _, s := range r.Samples {
		fmt.Fprintln(bw, s.line())
	}
	fmt.Fprintf(bw, "# end frames=%d gameOver=%v score=%d stars=%d mortals=%d room=%d\n",
		r.Frames, r.GameOver, r.Score, r.StarsLeft, r.Mortals, r.Room)
	fmt.Fprintf(bw, "# diag guarded=%d droppedWork=%d droppedBack=%d\n",
		r.Diag.Guarded, r.Diag.DroppedWorkRects, r.Diag.DroppedBackRects)
	if r.Script.Sound {
		a := r.Audio
		fmt.Fprintf(bw, "# sound requests=%d played=%d refused=%d trigger-refused=%d cut=%d music=%d triggers=%d\n",
			a.Requests, a.Granted, a.Refused, a.TriggerRefused, a.Displaced, a.MusicStarted, a.Triggers)
		fmt.Fprintf(bw, "# mix samples=%d clipped=%d digest=%s\n", a.Samples, a.Clipped, a.Digest)
	}
	for _, d := range r.Diag.Seen {
		fmt.Fprintf(bw, "# deviation %s\n", d)
	}
	return bw.Flush()
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
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
//	sound off                no sound system at all, which changes the composition
//	music off                effects but no score
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
		// word is num's counterpart for the keywords whose argument is a string, and it
		// exists because the four directory keywords used to index fields[1] directly: a
		// bug-report file with a bare `artdir` line panicked the tool that was meant to read
		// it. Every keyword in this switch now answers a missing argument with a sentence.
		word := func(i int) (string, error) {
			if i >= len(fields) {
				return "", fmt.Errorf("missing argument")
			}
			return fields[i], nil
		}
		// onOffArg reads the `on`/`off` the two audio keywords take. Nothing else is
		// accepted -- not `true`, not `1` -- because a script is read by people and a second
		// spelling is a second thing to remember.
		onOffArg := func() (bool, error) {
			v, err := word(1)
			if err != nil {
				return false, err
			}
			switch v {
			case "on":
				return true, nil
			case "off":
				return false, nil
			}
			return false, fmt.Errorf("want on or off, have %q", v)
		}
		set := func(dst *string, i int) error {
			v, err := word(i)
			if err != nil {
				return bad(err)
			}
			*dst = v
			return nil
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
			if err := set(&s.HouseDir, 1); err != nil {
				return nil, err
			}
		case "artdir":
			if err := set(&s.ArtDir, 1); err != nil {
				return nil, err
			}
		case "houseartdir":
			if err := set(&s.HouseArtDir, 1); err != nil {
				return nil, err
			}
		case "sounddir":
			if err := set(&s.SoundDir, 1); err != nil {
				return nil, err
			}
		case "sound":
			on, err := onOffArg()
			if err != nil {
				return nil, bad(err)
			}
			s.Sound = on
		case "music":
			on, err := onOffArg()
			if err != nil {
				return nil, bad(err)
			}
			s.Music = on
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
	// Written even when they are the defaults, like `neighbors` and `players` above: the whole
	// point of the emitted template is that a reporter can see what a run's settings were
	// without knowing what this package's defaults happen to be this month.
	fmt.Fprintf(bw, "sound %s\n", onOff(s.Sound))
	fmt.Fprintf(bw, "music %s\n", onOff(s.Music))
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
