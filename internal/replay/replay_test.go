package replay_test

// The harness, and the 600-frame determinism trace Stage 1.5b asks for.
//
// The trace is a golden file rather than a hash constant, and that is the whole design
// decision worth explaining. A digest test says "frame something changed"; a golden trace
// says "frame 217 gained a dirty rect", which is the difference between a failure that
// starts an investigation and a failure that ends one. The digest is still checked -- it is
// in the golden file's header -- so a bug report can quote one line instead of six hundred.
//
// Regenerate with:
//
//	go test ./internal/replay/ -update
//
// and read the diff before committing it. Every line of that diff is a behaviour change.

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/replay"
)

var update = flag.Bool("update", false, "rewrite the golden traces in testdata/")

// assetRoot is the extracted asset tree, relative to this package.
const assetRoot = "../../assets/extracted"

// requireAssets skips rather than fails when the tree has not been extracted. `make assets`
// is a separate step and a fresh clone has none of it, so a hard failure here would mean a
// checkout that cannot run its own test suite.
func requireAssets(t *testing.T, sub string) string {
	t.Helper()
	dir := filepath.Join(assetRoot, sub)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no extracted assets at %s: run `make assets`", dir)
	}
	return dir
}

// script loads a script from testdata and points it at this checkout's asset tree.
//
// The directories are overridden rather than written into the file because they are a
// property of the machine, not of the run: a script that shipped with a bug report names a
// house and nothing about where the reporter kept it.
func script(t *testing.T, name string) *replay.Script {
	t.Helper()
	requireAssets(t, "houses")
	requireAssets(t, "art")

	f, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("open script: %v", err)
	}
	defer f.Close()
	s, err := replay.Parse(f)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	s.HouseDir = filepath.Join(assetRoot, "houses")
	s.ArtDir = filepath.Join(assetRoot, "art")
	s.HouseArtDir = filepath.Join(assetRoot, "houseart")
	return s
}

// TestSixHundredFramesMatchTheGoldenTrace is the acceptance criterion: six hundred frames of
// CD Demo House, pinned frame by frame.
//
// The script starts in room 4 and ducts into room 5 on frame 9, which is chosen for three
// things the trace has to be able to show:
//
//	a room transition, so that RenderFrames advancing by two on one frame is visible
//	a pendulum -- room 5's kCuckoo is the only pendulum source in the game -- so that
//	  Scene.Pendulums is exercised rather than merely reported as zero
//	an unattended glider, so that the rest of the run is the dynamics, the sparkles and
//	  the scoreboard moving on their own
//
// # The toaster, and why the trace changed in 1.5c
//
// Room 5's only dynamic object is its kToaster, and the golden file now records nine
// twenty-frame bursts on a sixty-four-frame period: 54-73, 118-137, ... 566-585. That is
// RenderToast, which was an empty stub until Stage 1.5c, and every number in it is
// derivable from the house:
//
//	delay 15   -> Frame = Timer = delay * 3 = 45 reload frames
//	height 37  -> launchVelocity solves 1+2+...+9 >= 37, so Count = 9, VVel = -9
//	           -> airborne for 2*Count+1 = 19 frames, VVel walking -9 up to 9
//	45 + 19    =  64, the period; the room is entered on frame 9, so the first launch
//	              is 9 + 45 = 54
//
// The last frame of each burst is `w2m` up one but `b2w` unchanged, which is the landing:
// HandleToast work-rects the trail and clears Moving, and RenderToast's gate then skips
// the draw, so that frame registers a work rect with no matching erase. Any future
// regeneration that keeps the +1/+1 shape but moves those boundaries is a change to the
// toaster's arithmetic and should be read as one.
//
// # The ClockFrame deviation
//
// PLAN.md's list of quantities to pin includes the original's `clockFrame`. The port has no
// such counter: the pendulum and clock animation is RenderPendulums, which is a named empty
// stub charged to Stage 1.5f (render_frame.go). len(Scene.Pendulums) is pinned in its place,
// which proves the pendulum was composed into the locale even though nothing swings it yet.
// When 1.5f lands, this trace will change on every frame with a pendulum in it, and that is
// the point of having it now.
func TestSixHundredFramesMatchTheGoldenTrace(t *testing.T) {
	s := script(t, "duct.script")
	if s.Frames != 600 {
		t.Fatalf("testdata/duct.script runs %d frames, want 600", s.Frames)
	}

	res, err := replay.Run(s)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var got bytes.Buffer
	if err := res.Trace(&got); err != nil {
		t.Fatalf("trace: %v", err)
	}

	golden := filepath.Join("testdata", "duct.trace")
	if *update {
		if err := os.WriteFile(golden, got.Bytes(), 0o666); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("wrote %s (%d lines)", golden, bytes.Count(got.Bytes(), []byte{'\n'}))
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v (run `go test ./internal/replay/ -update`)", err)
	}
	if bytes.Equal(got.Bytes(), want) {
		return
	}

	// The report is the first differing line and a count, not a 600-line dump: the point
	// of a per-frame trace is that the first divergence is the whole story, and everything
	// after it is that divergence propagating.
	gotLines := strings.Split(strings.TrimRight(got.String(), "\n"), "\n")
	wantLines := strings.Split(strings.TrimRight(string(want), "\n"), "\n")
	diffs := 0
	first := -1
	for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
		g, w := "", ""
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if g != w {
			diffs++
			if first < 0 {
				first = i
				t.Errorf("trace diverges at line %d:\n  got  %s\n  want %s", i+1, g, w)
			}
		}
	}
	t.Errorf("%d of %d lines differ; regenerate with `go test ./internal/replay/ -update` "+
		"and read the diff", diffs, len(wantLines))
}

// TestTheSameScriptTwiceIsTheSameRun is the property the whole package rests on.
//
// It would be satisfied trivially if the run were pinned by construction, so what it
// actually tests is that nothing reachable from Run consults the wall clock, the map
// iteration order or an uninitialised random source. Two runs in the same process, back to
// back, is the cheapest arrangement that would catch any of those.
func TestTheSameScriptTwiceIsTheSameRun(t *testing.T) {
	s := script(t, "duct.script")
	s.Frames = 120 // enough to cover the transition; the golden test covers the long run

	a, err := replay.Run(s)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	b, err := replay.Run(s)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if a.Digest != b.Digest {
		t.Fatalf("digests differ: %s then %s", a.Digest, b.Digest)
	}
	if len(a.Samples) != len(b.Samples) {
		t.Fatalf("sample counts differ: %d then %d", len(a.Samples), len(b.Samples))
	}
	for i := range a.Samples {
		if a.Samples[i] != b.Samples[i] {
			t.Fatalf("frame %d differs:\n  %+v\n  %+v", a.Samples[i].Frame, a.Samples[i], b.Samples[i])
		}
	}
}

// TestDifferentSeedsAreDifferentRuns is the other half: the seed has to reach the
// simulation.
//
// A harness that pinned everything including the randomness would pass the test above and
// be useless -- it would reproduce one run of the game and no bug that depended on a dice
// roll. CD Demo House's rooms have sparkles and dynamics driven from World.Rand, so two
// seeds must diverge somewhere in 600 frames.
func TestDifferentSeedsAreDifferentRuns(t *testing.T) {
	s := script(t, "duct.script")
	// Both seeds are recorded here rather than read back from Result.Script, because
	// Result.Script is the same pointer that gets mutated below -- a failure message that
	// said `a.Script.Seed` would report the second seed twice and read as a harness that
	// forgot to change anything.
	first := s.Seed
	a, err := replay.Run(s)
	if err != nil {
		t.Fatalf("seed %d: %v", first, err)
	}
	s.Seed = first + 1000
	second := s.Seed
	b, err := replay.Run(s)
	if err != nil {
		t.Fatalf("seed %d: %v", second, err)
	}
	if a.Digest == b.Digest {
		t.Errorf("seeds %d and %d produced the same digest %s: the seed is not reaching "+
			"the simulation", first, second, a.Digest)
	}
}

// TestTheTransitionFrameRendersTwice pins the one thing in the trace that is easy to read as
// a bug and is not.
//
// PlayGame renders at the bottom of the loop; MoveDuctToDuct renders from inside the
// interaction pass, before that. So the frame a glider changes room on advances
// World.RenderFrames by two while World.Frame advances by one, and the second render is the
// one whose dirty-rect lists describe what the player is looking at.
func TestTheTransitionFrameRendersTwice(t *testing.T) {
	s := script(t, "duct.script")
	s.Frames = 30
	res, err := replay.Run(s)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var transition, prev replay.Sample
	found := false
	for i, sm := range res.Samples {
		if i > 0 && sm.Room != res.Samples[i-1].Room {
			transition, prev, found = sm, res.Samples[i-1], true
			break
		}
	}
	if !found {
		t.Fatal("no room change in 30 frames; the script no longer ducts out of room 4")
	}
	if transition.Frame != 9 {
		t.Errorf("room changed on frame %d, want 9", transition.Frame)
	}
	if d := transition.Renders - prev.Renders; d != 2 {
		t.Errorf("frame %d advanced RenderFrames by %d, want 2", transition.Frame, d)
	}
	if transition.Pendulums != 1 {
		t.Errorf("frame %d composed %d pendulums, want 1 (room 5's kCuckoo)",
			transition.Frame, transition.Pendulums)
	}
	// Every other frame of the run advances it by exactly one, which is what makes the
	// two above a fact about transitions rather than about frame 9.
	for i := 1; i < len(res.Samples); i++ {
		sm, pv := res.Samples[i], res.Samples[i-1]
		if sm.Frame == transition.Frame {
			continue
		}
		if d := sm.Renders - pv.Renders; d != 1 {
			t.Errorf("frame %d advanced RenderFrames by %d, want 1", sm.Frame, d)
		}
	}
}

// TestEvenFrameIsAStoredFlag guards a shortcut somebody will eventually take.
//
// World.EvenFrame is a field the loop toggles, not `Frame & 1`, and in a run that never
// composes a ball the two agree -- which is exactly why replacing the field with the
// expression would look like a simplification. The trace records the field, so this test
// states the invariant the current code happens to satisfy and will fail if a future
// sub-stage ever skips a toggle, rather than silently changing what the trace means.
//
// **The invariant is conditional, and duct.script is one of the runs it holds for.** Rooms 4
// and 5 contain no kBall, and a ball is what breaks it -- see
// TestABallBreaksTheEvenFrameInvariant, which is the other half of this pair and the reason
// this one cannot be read as a universal law.
func TestEvenFrameIsAStoredFlag(t *testing.T) {
	s := script(t, "duct.script")
	s.Frames = 60
	res, err := replay.Run(s)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, sm := range res.Samples {
		if want := sm.Frame%2 == 1; sm.Even != want {
			t.Errorf("frame %d: Even %v, want %v -- a toggle was skipped", sm.Frame, sm.Even, want)
		}
	}
}

// roomScript is a bare run that starts in one room and holds no keys, for the handful of tests
// that are about what a room's own contents do rather than about a recorded script.
func roomScript(t *testing.T, room int16, frames int) *replay.Script {
	t.Helper()
	requireAssets(t, "houses")
	requireAssets(t, "art")

	s := replay.NewScript("CD Demo House", frames)
	s.Room = room
	// Mid-room and high up, so that in every room below the glider starts in free air and
	// falls -- no transport under it, and nothing to collide with on the way down.
	s.Where.H, s.Where.V = 240, 40
	s.HouseDir = filepath.Join(assetRoot, "houses")
	s.ArtDir = filepath.Join(assetRoot, "art")
	s.HouseArtDir = filepath.Join(assetRoot, "houseart")
	return s
}

// TestABallBreaksTheEvenFrameInvariant is the other half of the pair above, and it is here
// rather than in a unit test because the point is that this is reachable in shipped content:
// three rooms of CD Demo House do it, on the frame the room is composed, with no input.
//
// Two writers put EvenFrame out of step with Frame, and **both are assignments to true, never
// toggles**:
//
//	launchVelocityHalfRate, from the kBall and kFish arms of AddDynamicObject -- so it
//	fires at composition time, before the first frame runs (hazard H1)
//
//	HandleBall's idle arm, on the frame a resting ball is kicked into motion -- which is
//	deliberate, because it phase-locks the half-rate gravity to the launch
//
// Every ball in CD Demo House is authored `initial 1`, so the two fire exactly one frame
// apart: the composition sets true, the loop head toggles it to false, and the ball's first
// idle frame sets it back to true. **The second write undoes the first**, which is why the
// divergence below is one frame wide and not permanent.
//
// That cancellation is arithmetic, not luck, and it is also parity-dependent: it holds because
// these runs compose the room before frame 1, whose expected parity is the true the ball
// writes. A ball switched on mid-game by a trigger has no second write one frame later to
// cancel it, and a room ducted into on the other parity would keep the shift -- see
// game.TestBallResetsParityRatherThanTogglingIt for the mechanism at one frame's resolution.
//
// So the honest statement is: EvenFrame is a stored flag, it *is* knocked out of step by real
// shipped rooms, and deriving it from Frame would change the flame-versus-star alternation on
// the frame a ball room is entered.
func TestABallBreaksTheEvenFrameInvariant(t *testing.T) {
	// The three rooms of CD Demo House that compose a kBall, and two controls that do not.
	// The controls are what make this a fact about balls rather than about room composition.
	cases := []struct {
		room  int16
		name  string
		balls bool
	}{
		{30, "Ball Illusion", true},
		{172, "Attica, Greece", true},
		{175, "Dodgeball", true},
		{4, "Sticky Fly Paper", false},
		{5, "the pendulum room", false},
	}

	for _, tc := range cases {
		res, err := replay.Run(roomScript(t, tc.room, 30))
		if err != nil {
			t.Fatalf("room %d %q: %v", tc.room, tc.name, err)
		}
		if len(res.Samples) < 30 {
			t.Fatalf("room %d %q: %d samples, want the whole run", tc.room, tc.name,
				len(res.Samples))
		}

		var diverged []int64
		for _, sm := range res.Samples {
			if sm.Even != (sm.Frame%2 == 1) {
				diverged = append(diverged, sm.Frame)
			}
		}

		if !tc.balls {
			if len(diverged) != 0 {
				t.Errorf("room %d %q has no ball but diverged on frames %v",
					tc.room, tc.name, diverged)
			}
			continue
		}

		// Frame 0 is the composed state, before the loop has run once, so this is the
		// composition's write and nothing else.
		if len(diverged) != 1 || diverged[0] != 0 {
			t.Errorf("room %d %q diverged on frames %v, want exactly [0]: the composition's "+
				"write and then the idle arm's write cancelling it",
				tc.room, tc.name, diverged)
			continue
		}
		if !res.Samples[0].Even {
			t.Errorf("room %d %q: frame 0 Even = false, want the true the composition wrote",
				tc.room, tc.name)
		}
	}
}

// TestScriptRoundTrips: Write then Parse is the identity.
//
// It matters because Write is how `glidertool replay -script` produces the template a
// reporter fills in, so a field Write emits and Parse rejects would hand somebody a file
// that does not load.
func TestScriptRoundTrips(t *testing.T) {
	orig := replay.NewScript("CD Demo House", 600)
	orig.Seed = 7
	orig.Neighbors = 3
	orig.TwoPlayer = true
	orig.Room = 4
	orig.Where = house.Point{H: 420, V: 20}
	orig.Facing = 0
	orig.Gliders = 3
	orig.Stars = 5
	orig.Clock = time.Date(1995, time.March, 2, 4, 5, 6, 0, time.UTC)
	orig.Input = []replay.Hold{
		{Frame: 0, P1: player.Keys{Right: true}},
		{Frame: 45, P1: player.Keys{Left: true, Batt: true}, P2: player.Keys{Band: true}},
		{Frame: 90},
	}

	var buf bytes.Buffer
	if err := orig.Write(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Keep the text: Parse consumes the buffer, and both the failure messages below and
	// the second-write comparison need it afterwards.
	text := buf.String()
	back, err := replay.Parse(strings.NewReader(text))
	if err != nil {
		t.Fatalf("parse what we wrote: %v\n%s", err, text)
	}

	// Field by field on the parts a reader would check by eye, then a byte comparison of
	// the second write against the first. The second write is the stronger assertion --
	// it covers every field including ones added later -- and the named checks are what
	// make a failure legible.
	if back.House != orig.House || back.Seed != orig.Seed || back.Frames != orig.Frames ||
		back.Neighbors != orig.Neighbors || back.TwoPlayer != orig.TwoPlayer {
		t.Errorf("run settings changed: %q/%d/%d/%d/%v, want %q/%d/%d/%d/%v",
			back.House, back.Seed, back.Frames, back.Neighbors, back.TwoPlayer,
			orig.House, orig.Seed, orig.Frames, orig.Neighbors, orig.TwoPlayer)
	}
	if back.Room != orig.Room || back.Where != orig.Where || back.Facing != orig.Facing ||
		back.Gliders != orig.Gliders || back.Stars != orig.Stars {
		t.Errorf("start point changed: room %d at %v facing %d, %d gliders, %d stars; "+
			"want room %d at %v facing %d, %d gliders, %d stars",
			back.Room, back.Where, back.Facing, back.Gliders, back.Stars,
			orig.Room, orig.Where, orig.Facing, orig.Gliders, orig.Stars)
	}
	if !back.Clock.Equal(orig.Clock) {
		t.Errorf("clock %v, want %v", back.Clock, orig.Clock)
	}
	if len(back.Input) != len(orig.Input) {
		t.Fatalf("%d holds, want %d\n%s", len(back.Input), len(orig.Input), text)
	}
	for i := range orig.Input {
		if back.Input[i] != orig.Input[i] {
			t.Errorf("hold %d: %+v, want %+v", i, back.Input[i], orig.Input[i])
		}
	}

	var again bytes.Buffer
	if err := back.Write(&again); err != nil {
		t.Fatalf("write again: %v", err)
	}
	if again.String() != text {
		t.Errorf("second write differs from the first:\n got %s\nwant %s", again.String(), text)
	}
}

// TestFacingLeftSurvivesTheScript is a regression test for a bug this package had.
//
// Facing is a byte where 0 means left, and Run used to treat 0 as "unset" and substitute
// right -- so `facing left` was silently ignored and every left-door script started the
// glider the wrong way round. The symptom was subtle in exactly the way that matters here:
// the run still worked, the glider still left through the left door, and it arrived
// mid-about-face instead of walking, which reads as a port bug rather than a harness bug.
func TestFacingLeftSurvivesTheScript(t *testing.T) {
	s, err := replay.Parse(strings.NewReader("house H\nroom 4\nwhere 1 2\nfacing left\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.Facing != 0 {
		t.Errorf("`facing left` gave Facing %d, want 0", s.Facing)
	}
	if s = replay.NewScript("H", 1); s.Facing == 0 {
		t.Errorf("NewScript defaults to Facing 0 (left); want right, so that a script " +
			"that says nothing starts the way a new game does")
	}
}

func TestKeysRoundTrip(t *testing.T) {
	cases := []struct {
		text string
		keys player.Keys
	}{
		{"-", player.Keys{}},
		{"left", player.Keys{Left: true}},
		{"right,batt", player.Keys{Right: true, Batt: true}},
		{"left,right,batt,band,command,delete,pause", player.Keys{
			Left: true, Right: true, Batt: true, Band: true,
			Command: true, Delete: true, Pause: true}},
	}
	for _, c := range cases {
		got, err := replay.ParseKeys(c.text)
		if err != nil {
			t.Errorf("ParseKeys(%q): %v", c.text, err)
			continue
		}
		if got != c.keys {
			t.Errorf("ParseKeys(%q) = %+v, want %+v", c.text, got, c.keys)
		}
		if back := replay.FormatKeys(c.keys); back != c.text {
			t.Errorf("FormatKeys(%+v) = %q, want %q", c.keys, back, c.text)
		}
	}
	// "none" parses and is not what FormatKeys emits, which is deliberate: a reporter
	// writing the file by hand should not have to guess which spelling is accepted.
	if k, err := replay.ParseKeys("none"); err != nil || k != (player.Keys{}) {
		t.Errorf("ParseKeys(\"none\") = %+v, %v; want all-up and no error", k, err)
	}
}

// TestBadScriptsAreRejected: a typo must not silently run a different game.
//
// The alternative -- warn and continue -- is what makes a bug report unfalsifiable: the
// maintainer runs the script, gets a different frame, and has no way to tell whether the
// port changed or the file was misread.
func TestBadScriptsAreRejected(t *testing.T) {
	bad := []string{
		"housse CD Demo House\n",     // misspelled keyword
		"house H\nfarmes 60\n",       //
		"house H\nfacing sideways\n", // not left or right
		"house H\nframes abc\n",      // not a number
		"house H\nplayers 3\n",       // out of range
		"house H\nclock yesterday\n", // not RFC3339
		"house H\nat 0 jump\n",       // unknown key name
		"house H\nat\n",              // missing frame
		"house\n",                    // missing value
		"house H\nwhere 1\n",         // where takes two numbers
	}
	for _, text := range bad {
		if s, err := replay.Parse(strings.NewReader(text)); err == nil {
			t.Errorf("Parse(%q) succeeded, giving %+v; want an error", text, s)
		}
	}
}

// TestRunRejectsUnrunnableScripts covers the checks Run makes before it loads anything, so
// that a bad script fails with a sentence instead of a panic three packages down.
func TestRunRejectsUnrunnableScripts(t *testing.T) {
	requireAssets(t, "houses")
	base := func() *replay.Script {
		s := replay.NewScript("CD Demo House", 10)
		s.HouseDir = filepath.Join(assetRoot, "houses")
		s.ArtDir = filepath.Join(assetRoot, "art")
		s.HouseArtDir = filepath.Join(assetRoot, "houseart")
		return s
	}
	cases := map[string]func(*replay.Script){
		"no house":       func(s *replay.Script) { s.House = "" },
		"zero frames":    func(s *replay.Script) { s.Frames = 0 },
		"bad neighbours": func(s *replay.Script) { s.Neighbors = 4 },
		"missing house":  func(s *replay.Script) { s.House = "No Such House" },
		"room past end":  func(s *replay.Script) { s.Room = 30000; s.Where = house.Point{H: 1, V: 1} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := base()
			mutate(s)
			if _, err := replay.Run(s); err == nil {
				t.Errorf("Run succeeded; want an error")
			}
		})
	}
}

// TestFirstRoomStartIsNotAResume pins the difference between the two ways a script can name
// a room, because it is the one part of the format that could be read as redundant.
//
// `room N` alone moves the house's own starting room and lets NewGameMode run: the glider
// appears at the house's authored initial point, the banner shows, and nothing is
// synthesised. `room N` plus `where` resumes instead. Both are useful -- the first is how
// you look at a room the way its author meant it to be entered -- and confusing them would
// silently change what a bug report reproduces.
func TestFirstRoomStartIsNotAResume(t *testing.T) {
	requireAssets(t, "art")
	s := replay.NewScript("CD Demo House", 5)
	s.HouseDir = filepath.Join(assetRoot, "houses")
	s.ArtDir = filepath.Join(assetRoot, "art")
	s.HouseArtDir = filepath.Join(assetRoot, "houseart")
	s.Room = 4

	res, err := replay.Run(s)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(res.Samples) == 0 {
		t.Fatal("no frames")
	}
	if res.Samples[0].Room != 4 {
		t.Errorf("started in room %d, want 4", res.Samples[0].Room)
	}
	// The authored initial point, not the resume point: the glider is nowhere near the
	// ceiling duct, so it must still be in room 4 five frames later.
	if res.Room != 4 {
		t.Errorf("ended in room %d, want 4 -- `room` without `where` must not place the "+
			"glider at a transit", res.Room)
	}
}
