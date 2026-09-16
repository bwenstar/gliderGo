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
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"glidergo/internal/audio"
	"glidergo/internal/demo"
	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/render"
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
	f, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("open script: %v", err)
	}
	defer f.Close()
	s, err := replay.Parse(f)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	return localAssets(t, s)
}

// localAssets points a script at this checkout's asset tree and skips if it is not there.
//
// The sound tree is required only when the script asks for sound, because Run treats a missing
// bank as an error when it does -- a trace whose `snd=` column is empty for want of assets would
// be a fixture of nothing.
func localAssets(t *testing.T, s *replay.Script) *replay.Script {
	t.Helper()
	requireAssets(t, "houses")
	requireAssets(t, "art")
	s.HouseDir = filepath.Join(assetRoot, "houses")
	s.ArtDir = filepath.Join(assetRoot, "art")
	s.HouseArtDir = filepath.Join(assetRoot, "houseart")
	if s.Sound {
		requireAssets(t, "sound")
		s.SoundDir = filepath.Join(assetRoot, "sound")
	}
	// A demo is a file rather than a directory, and it is repointed for the same reason: the
	// script names the stream the way somebody standing in the repository root would type it,
	// and these tests run in this package's directory. The name inside the tree is kept, so a
	// second stream added later resolves to itself rather than to the shipped one.
	if s.Demo != "" {
		requireAssets(t, filepath.Join("res", "demo"))
		s.Demo = filepath.Join(assetRoot, strings.TrimPrefix(s.Demo, "assets/extracted/"))
	}
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
// # The invisible switch, and why the trace changed in 1.5d
//
// Room 5 has a kInvisSwitch at v=0 -- flat against the ceiling, object 22 -- with type 1,
// ForceOn, wired to object 0: the room-sized kDeluxeTrans whose rect is 1,5,313,501 and whose
// state in the house file is off. HandleSwitches was a stub until Stage 1.5d, so that
// transporter stayed off for the whole run and the unattended glider bobbed under the ceiling
// to the end of it. It is live now, and the trace goes somewhere:
//
//	f=224   the glider trips the switch. `w2m` up one and `b2w` unchanged, the same
//	        signature as a toaster landing and for the same reason: HandleSwitches blits
//	        back->work directly (CopyRectBackToWork is not queued) and queues one
//	        work->main rect. Silent, because an invisible switch has no lever and no click
//	f=225   the transporter's kTransportIt rect is live, so it fires: 225-239 dissolving
//	        (mode 10), TransportGliderOut counting the fade sequence down from 15
//	f=240   TransportRoomToRoom lands the glider in **room 70, "Welcome…"** -- the
//	        transporter's own link, and the house's start room -- then 16 frames
//	        materialising (mode 23) and 345 frames normal
//
// Three details of the second half are worth knowing before reading it as a bug:
//
//	score 100 -> 200 on the transition is HandleRoomVisitation's award for reaching a new
//	  room, not a prize. HandleRewards is never called anywhere in this run
//	`w2m` and `b2w` drop to 1 and stay there, where room 5 held 2. Room 70 is a sky room
//	  (bounds 0x0000, no floor), so the glider casts no shadow and there is one rect a
//	  frame instead of two
//	the glider never settles: room 70's kInvisBlower at v=139 h=236 holds it up, which is
//	  why dest oscillates over five values to the end of the run
//
// What the trace gave up for that is six of the toaster's nine bursts -- 246-265 onward, all
// of them after the glider leaves room 5. The three before it survive, and the arithmetic
// itself is pinned closed-form by TestLaunchVelocityFromHeight and by the dynamics table, so
// this costs an illustration rather than any coverage. The switch, the state change publishing
// a hot spot that did not exist, the transport and a second room are all new to it.
//
// # ClockFrame
//
// PLAN.md's list of quantities to pin includes the original's `clockFrame`, and until 1.5f
// there was nothing to pin: RenderPendulums was a named empty stub, so len(Scene.Pendulums)
// stood in for it and proved only that the pendulum had been composed into the locale. Both
// are in the trace now, and the pair is what makes the swing legible -- `pend` says a clock
// is in the room and `clock` says whether this was one of the two frames in fifteen it moved
// on. See the note on Sample.ClockFrame.
//
// 1.5f landing is also why this golden was regenerated: five new RandomInt draws in the
// registration pass shifted the whole `rand` column from the first room load onward, and the
// flames, stars and pendulums that now animate each register a work rect, so `w2m` moved
// too. Neither is a behaviour change to anything this file already covered: the diff against
// the previous golden is `clock`, `rand`, `w2m` and the digest header, and nothing else.
//
// `b2w` did not move, and that is the interesting half of the diff. The four animated
// families register **no back rects** -- their blit comes out of a filmstrip and is opaque,
// so it erases the last frame by covering it (game/anim.go). Shreds do register back rects,
// and the column's stillness says this script never shreds a glider; the confetti is covered
// by unit tests in game/shreds_test.go instead.
func TestSixHundredFramesMatchTheGoldenTrace(t *testing.T) {
	s := script(t, "duct.script")
	if s.Frames != 600 {
		t.Fatalf("testdata/duct.script runs %d frames, want 600", s.Frames)
	}

	res, err := replay.Run(s)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	checkGolden(t, res, "duct.trace")
}

// checkGolden compares a run's trace against a checked-in one, or rewrites it under -update.
//
// It is a helper rather than inline code in one test because there is more than one golden
// now -- the one-player duct run and the two-player race -- and the *reporting* is the part
// worth sharing. See the comment inside on why the first differing frame is singled out.
func checkGolden(t *testing.T, res *replay.Result, name string) {
	t.Helper()

	var got bytes.Buffer
	if err := res.Trace(&got); err != nil {
		t.Fatalf("trace: %v", err)
	}

	golden := filepath.Join("testdata", name)
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

	// The report is the first differing *frame* and a count, not a 600-line dump: the point
	// of a per-frame trace is that the first divergence is the whole story, and everything
	// after it is that divergence propagating.
	//
	// The comment lines are skipped when choosing which difference to print, and that is not
	// tidiness. The digest on line 4 is a hash of every sample, so it changes whenever
	// anything else does; reporting it as "the first divergence" names line 4 on every single
	// failure and buries the frame that actually moved. It is still reported, separately and
	// second, because a bug report can quote it in one line.
	gotLines := strings.Split(strings.TrimRight(got.String(), "\n"), "\n")
	wantLines := strings.Split(strings.TrimRight(string(want), "\n"), "\n")
	diffs, header := 0, 0
	first := -1
	for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
		g, w := "", ""
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if g == w {
			continue
		}
		diffs++
		if strings.HasPrefix(g, "#") || strings.HasPrefix(w, "#") {
			header++
			continue
		}
		if first < 0 {
			first = i
			t.Errorf("trace diverges at line %d:\n  got  %s\n  want %s", i+1, g, w)
		}
	}
	if first < 0 {
		// Only the header moved: the samples are identical and the digest is not, which
		// means the digest function changed rather than the game.
		t.Errorf("every sample matches but %d header line(s) differ -- the digest or the "+
			"trace header changed, not the run", header)
	}
	t.Errorf("%d of %d lines differ (%d of them header); regenerate with "+
		"`go test ./internal/replay/ -update` and read the diff",
		diffs, len(wantLines), header)
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

	// The audio half of the same property, and a strictly stronger claim than the loop above.
	// The sample lines carry the sound *requests*; this is the mix -- every sample of every
	// channel, the priority policy's choice of channel, the drop-sample rate conversion and the
	// clip point. Nothing in the recorded path may consult a clock, and the pump proves it here.
	if a.Audio.Digest != b.Audio.Digest {
		t.Errorf("mix digests differ: %s then %s", a.Audio.Digest, b.Audio.Digest)
	}
	if a.Audio.Digest == "" {
		t.Error("no mix digest: the run did not mix at all, so this test proved nothing")
	}
	// FrameTick is exact arithmetic rather than a clock, so this is an equality and not a
	// tolerance. One recorded frame is one block; the trace's own sample count is 601 for 600
	// frames, frame 0 being the state before the loop ran.
	if want := int64(len(a.Samples)) * audio.SamplesPerFrame; a.Audio.Samples != want {
		t.Errorf("mixed %d samples for %d frames, want %d exactly",
			a.Audio.Samples, len(a.Samples), want)
	}
	// CD Demo House's own 'snd ' resources, which is what makes its sound triggers hot spots at
	// all. Nine of them extract; a house whose trigger sounds stopped loading would run silently
	// past every sound trigger in it and nothing else in the suite would notice.
	if a.Audio.Triggers != 9 {
		t.Errorf("CD Demo House loaded %d trigger sounds, want 9", a.Audio.Triggers)
	}
	if a.Audio.Requests == 0 || a.Audio.Granted == 0 {
		t.Errorf("120 frames of this script asked for %d sounds and played %d; it should duct "+
			"between two rooms and pass a cuckoo clock", a.Audio.Requests, a.Audio.Granted)
	}
}

// TestSoundOffIsADifferentRunNotAQuieterOne pins the one thing about `sound off` that is easy to
// get wrong in a bug report: it changes the digest.
//
// It has to, because a sound request is a decision the simulation made at a frame and the sample
// line records it. A port that stopped playing the toaster would otherwise pass every digest
// comparison in the suite.
//
// What the second half shows is the *limit* of that: in this script the two runs differ in the
// sound column and nowhere else, because rooms 4, 5 and 70 have no sound trigger between them. In
// a house that has one they would differ in the composition as well -- a sound trigger whose
// sound cannot load gets no hot spot at all (game/hotspots.go) -- which is why the package
// documents silence as a different simulation rather than a quieter one, and why a trace records
// which of the two it was in its header.
func TestSoundOffIsADifferentRunNotAQuieterOne(t *testing.T) {
	loud := script(t, "duct.script")
	loud.Frames = 120
	on, err := replay.Run(loud)
	if err != nil {
		t.Fatalf("with sound: %v", err)
	}

	quiet := script(t, "duct.script")
	quiet.Frames = 120
	quiet.Sound = false
	off, err := replay.Run(quiet)
	if err != nil {
		t.Fatalf("without sound: %v", err)
	}

	if len(off.Events) != 0 {
		t.Errorf("a silent run logged %d sound events", len(off.Events))
	}
	if off.Audio != (replay.Audio{}) {
		t.Errorf("a silent run reported audio: %+v", off.Audio)
	}
	if off.Digest == on.Digest {
		t.Errorf("both runs digest to %s: the sound column is not reaching the digest, so a "+
			"regression that silenced the game would pass every determinism test here", on.Digest)
	}

	if len(on.Samples) != len(off.Samples) {
		t.Fatalf("%d frames with sound, %d without", len(on.Samples), len(off.Samples))
	}
	for i := range on.Samples {
		x, y := on.Samples[i], off.Samples[i]
		if x.Sounds == "" && y.Sounds == "" && x == y {
			continue
		}
		x.Sounds, y.Sounds = "", ""
		if x != y {
			t.Fatalf("frame %d differs in more than its sound:\n  %+v\n  %+v",
				on.Samples[i].Frame, x, y)
		}
	}
	// Vacuously true if nothing ever asked for a sound.
	sounded := 0
	for _, s := range on.Samples {
		if s.Sounds != "" {
			sounded++
		}
	}
	if sounded == 0 {
		t.Error("no frame of the loud run asked for a sound; the comparison above proved nothing")
	}
}

// TestTheWAVHoldsTheMix is 1.6's acceptance criterion, minus the ear.
//
// The build host has no sound card, so "the audio works" cannot be checked here at all -- what can
// be checked is that the file a developer carries to a machine that does have one holds exactly the
// samples the run reported, at the length the frame count implies, with a header that describes
// them. The listening is the part a human does; this is the part that makes the file worth
// carrying.
//
// The re-hash at the end is the load-bearing assertion. Audio.Digest is taken from a Digest sink
// inside the run, and this reads the bytes back off the disk through the WAV's own header offsets
// and hashes them again: if the two agree, then the digest a bug report quotes is a checksum of a
// file anybody can produce and verify with sha256sum. See audio.Digest.
func TestTheWAVHoldsTheMix(t *testing.T) {
	s := script(t, "duct.script")
	s.Frames = 60 // past the duct, the cuckoo's first tick and the toaster's first launch

	path := filepath.Join(t.TempDir(), "run.wav")
	w, err := audio.CreateWAV(path)
	if err != nil {
		t.Fatal(err)
	}
	// No defer w.Close(): RunTo closes the sink, and the patched header below is the evidence
	// that it did. WAV.Close is idempotent, so a call here would be harmless -- and would also
	// hide the bug this asserts against.
	res, err := replay.RunTo(s, w)
	if err != nil {
		t.Fatalf("RunTo: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const header = 44 // RIFF 12 + fmt 24 + data 8
	want := header + int(res.Audio.Samples)*2
	if len(body) != want {
		t.Fatalf("the file is %d bytes, want %d: %d samples of 16-bit mono after a %d-byte header",
			len(body), want, res.Audio.Samples, header)
	}
	if got := binary.LittleEndian.Uint32(body[40:]); int64(got) != res.Audio.Samples*2 {
		t.Errorf("the data chunk says %d bytes, the run mixed %d: the length fields are patched "+
			"on Close, so this is a file nobody closed", got, res.Audio.Samples*2)
	}
	if got := binary.LittleEndian.Uint32(body[24:]); got != audio.Rate {
		t.Errorf("the header says %d Hz, want %d", got, audio.Rate)
	}

	samples := audio.Decode(body[header:])
	d := audio.NewDigest()
	if err := d.Write(samples); err != nil {
		t.Fatal(err)
	}
	if d.Sum() != res.Audio.Digest {
		t.Errorf("the file hashes to %s, the run reported %s", d.Sum(), res.Audio.Digest)
	}

	loudest := int16(0)
	for _, v := range samples {
		if v > loudest {
			loudest = v
		}
	}
	if loudest == 0 {
		t.Errorf("%d samples and not one of them positive: the WAV is silence, and %d sound "+
			"requests went somewhere else", len(samples), res.Audio.Requests)
	}
}

// TestTwelveHundredFramesTwiceAreTheSamePicture is 1.5f's last acceptance clause, and it is
// a strictly stronger claim than the digest test above.
//
// The sample trace records dirty-rect *counts*. It cannot see a pixel: a flame stuck on cel
// 3, a pendulum swinging the wrong way, a filmstrip baked with the wrong frame masked over it
// and a shred emerging top-first all register the same rect as the correct version and all
// leave `w2m` and `b2w` alone. Everything 1.5f added is therefore invisible to the digest by
// construction -- which is exactly why the plan asks for planes here rather than more samples.
//
// Twelve hundred frames is twice the golden's run and about twenty seconds of play. What the
// length buys is not that a divergence would still be visible at the end -- this script
// settles into a steady state, and a difference that self-corrected would show up in the
// samples rather than in the planes -- but that the planes are reached through a thousand
// frames of animation, a transport and two rooms rather than through a single composition.
// The digest is compared here as well, at this length, for the divergence that does not last.
//
// The three checks after the comparison are what stop it passing vacuously, and two of them
// are facts about the frame protocol worth having pinned in their own right:
//
//	room 70, Work == Back    every renderer that draws into the work map registers a back
//	                        rect, and CopyRectsQD's second loop replays all of them -- so a
//	                        room where nothing animates ends each frame exactly as composed.
//	                        A renderer that forgot its back rect fails here
//	room 5, Work != Back     except the animated families, whose cels come out of a filmstrip
//	                        and register **no** back rect (game/anim.go). Room 5's cuckoo has
//	                        been drawing over the composition since the glider arrived and
//	                        none of it was ever erased
//	the two runs differ      all three planes, between a run that ends in room 5 and one that
//	                        ends in room 70, which is what shows the hash reads the picture
//	                        rather than the house
func TestTwelveHundredFramesTwiceAreTheSamePicture(t *testing.T) {
	s := script(t, "duct.script")
	s.Frames = 1200

	a, err := replay.Run(s)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	b, err := replay.Run(s)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if a.Frames < 1200 {
		t.Fatalf("the run stopped after %d frames; the script ends early and the long tail "+
			"this test is about was never simulated", a.Frames)
	}

	// Named one at a time rather than as a struct compare, because *which* plane moved is
	// the whole diagnosis: Back is the composition, Work is Back plus everything that
	// animated, Main is Work as the dirty rects delivered it. See replay.Planes.
	for _, p := range []struct{ name, x, y string }{
		{"back", a.Planes.Back, b.Planes.Back},
		{"work", a.Planes.Work, b.Planes.Work},
		{"main", a.Planes.Main, b.Planes.Main},
	} {
		if p.x == "" {
			t.Errorf("%s plane hashed to nothing: the surface was nil", p.name)
			continue
		}
		if p.x != p.y {
			t.Errorf("the %s plane differs between two runs of one script: %s then %s",
				p.name, p.x, p.y)
		}
	}
	if a.Digest != b.Digest {
		t.Errorf("digests differ over 1200 frames: %s then %s -- the samples diverged too, "+
			"so read the trace before the planes", a.Digest, b.Digest)
	}

	// Room 70 is a sky room with nothing animated in it, so the erase pass is complete.
	if a.Room != 70 {
		t.Logf("the long run now ends in room %d, not 70; the two checks below describe "+
			"room 70 and room 5, and a failure may mean the script goes somewhere else now",
			a.Room)
	} else if a.Planes.Work != a.Planes.Back {
		t.Errorf("room %d ends with work %s over back %s: nothing in that room animates, so "+
			"something drew into the work map without registering a back rect and the next "+
			"frame will draw on top of it", a.Room, a.Planes.Work, a.Planes.Back)
	}

	short := script(t, "duct.script")
	short.Frames = 120
	c, err := replay.Run(short)
	if err != nil {
		t.Fatalf("short run: %v", err)
	}
	if c.Room == a.Room {
		t.Fatalf("both runs ended in room %d; this test's sharpness check assumes the "+
			"120-frame run stops somewhere else (it used to be room 5, against room 70)", a.Room)
	}
	if c.Planes.Work == c.Planes.Back {
		t.Errorf("room %d ends with the work map identical to the composition, but its cuckoo "+
			"registers no back rect and should have left its cels behind; nothing was drawn "+
			"over the composition at all", c.Room)
	}
	for _, p := range []struct{ name, long, short string }{
		{"back", a.Planes.Back, c.Planes.Back},
		{"work", a.Planes.Work, c.Planes.Work},
		{"main", a.Planes.Main, c.Planes.Main},
	} {
		if p.long == p.short {
			t.Errorf("the %s plane is %s in both rooms %d and %d; the hash is not reading "+
				"the picture", p.name, p.long, c.Room, a.Room)
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

// TestTheCuckooTicksUnevenly reads the `clock` column of the same run as a statement about
// the pendulum rather than as 600 opaque numbers.
//
// The golden pins every value; this pins the *shape*, which is the thing a reader would want
// to check against the original and the thing a plausible-looking tidy-up would break.
// RenderPendulums swings when ClockFrame reads exactly 10 or exactly 15 and resets to 0 on
// the 15, so the swing frames are 10 apart and then 5 apart, forever. Collapse the two
// constants into a single interval -- the obvious simplification, and the counter still
// counts -- and this test fails while the room still looks like a working clock.
//
// The frame the glider ducts into room 5 is skipped: RenderFrames advances by two there, so
// the counter advances by two as well and the gap either side of it is not a gap between
// swings. That is asserted in TestTheTransitionFrameRendersTwice above.
func TestTheCuckooTicksUnevenly(t *testing.T) {
	s := script(t, "duct.script")
	res, err := replay.Run(s)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// A swing is a sample whose counter reads 10, or one whose counter reads 0 having
	// been reset from 15. `clock=0` in a room with no pendulum is not a swing, hence the
	// pend gate.
	var swings []int64
	for i, sm := range res.Samples {
		if sm.Pendulums == 0 {
			continue
		}
		if sm.ClockFrame == 10 || (sm.ClockFrame == 0 && i > 0 && res.Samples[i-1].ClockFrame == 14) {
			swings = append(swings, sm.Frame)
		}
	}
	if len(swings) < 20 {
		t.Fatalf("%d swings in 600 frames, want dozens; room 5's kCuckoo is not swinging",
			len(swings))
	}
	if sm := res.Samples[len(res.Samples)-1]; sm.ClockFrame > 14 {
		t.Errorf("last sample has clock=%d; 15 is reset to 0 before the function returns "+
			"and must never appear in the trace", sm.ClockFrame)
	}

	// Gaps of 5 and 10, alternating. The transition frame is excluded by dropping any gap
	// that straddles it, which is at most one of them.
	for i := 1; i < len(swings); i++ {
		gap := swings[i] - swings[i-1]
		if swings[i-1] <= 9 && swings[i] >= 9 {
			continue
		}
		if gap != 5 && gap != 10 {
			t.Fatalf("swings at frames %d and %d are %d apart, want 5 or 10",
				swings[i-1], swings[i], gap)
		}
		if i >= 2 {
			if prev := swings[i-1] - swings[i-2]; prev == gap && swings[i-2] > 9 {
				t.Errorf("two %d-frame gaps in a row ending at frame %d: the tick-tock "+
					"is even, so the 10-and-15 pair has been collapsed into one interval",
					gap, swings[i])
			}
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
	s := replay.NewScript("CD Demo House", frames)
	s.Room = room
	// Mid-room and high up, so that in every room below the glider starts in free air and
	// falls -- no transport under it, and nothing to collide with on the way down.
	s.Where.H, s.Where.V = 240, 40
	return localAssets(t, s)
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
	// A path with a space in it, because `demo` takes the rest of the line for the same reason
	// `house` does and a quoted-string reading of it would break on the first Mac file name.
	orig.Demo = "res/demo/my recording.bin"
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
	if back.Demo != orig.Demo {
		t.Errorf("demo path %q, want %q", back.Demo, orig.Demo)
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
		"house H\nsound maybe\n",     // not on or off
		"house H\nartdir\n",          // a keyword with its argument left off
		"house H\ndemo\n",            // and the same for the demo path
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
	// base takes the *subtest's* t rather than closing over this one. Closing over the parent
	// looks equivalent and is not: localAssets calls requireAssets, whose t.Skipf on a checkout
	// with no extracted assets would then call Goexit on the parent from the subtest's
	// goroutine. That is not a skip, it is
	//     testing.go: test executed panic(nil) or runtime.Goexit: subtest may have called
	//     FailNow on a parent test
	// -- a hard failure, on the one machine that most needs the suite to be runnable: a fresh
	// clone. `cases` is a map, so which subtest tripped it varied from run to run, which is
	// exactly as confusing as it sounds.
	base := func(t *testing.T) *replay.Script {
		t.Helper()
		return localAssets(t, replay.NewScript("CD Demo House", 10))
	}
	cases := map[string]func(*replay.Script){
		"no house":       func(s *replay.Script) { s.House = "" },
		"zero frames":    func(s *replay.Script) { s.Frames = 0 },
		"bad neighbours": func(s *replay.Script) { s.Neighbors = 4 },
		"missing house":  func(s *replay.Script) { s.House = "No Such House" },
		"room past end":  func(s *replay.Script) { s.Room = 30000; s.Where = house.Point{H: 1, V: 1} },
		// A two-player demo would otherwise run silently on the keyboard timeline, because
		// PlayGame only reaches GetDemoInput in its one-player arm. The path need not exist:
		// the arity check comes first, deliberately, so the complaint is about the script.
		"two-player demo": func(s *replay.Script) {
			s.Demo = filepath.Join(assetRoot, demo.ShippedPath)
			s.TwoPlayer = true
		},
		"missing demo": func(s *replay.Script) {
			s.Demo = filepath.Join(t.TempDir(), "no-such-demo.bin")
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := base(t)
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
	s := localAssets(t, replay.NewScript("CD Demo House", 5))
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

// TestWatchSeesEveryFrameOnce pins the Watch contract, which internal/fidelity's whole corpus
// rests on: one call per recorded frame, in order, with the sample the frame's counters were
// taken from.
//
// The point being pinned is that Present is not a frame. It fires once per wipe strip inside a
// transition -- and this script's glider takes a duct on frame 9, so the run has one -- and a
// hook that reported per Present would hand a corpus 160 rows for one frame and pin the middle
// of the wipe as if it were the frame.
func TestWatchSeesEveryFrameOnce(t *testing.T) {
	s := script(t, "duct.script")
	s.Frames = 40

	var seen []int64
	var mains, works, backs []string
	var kept *render.Surface
	var keptHash string
	res, err := replay.RunWatching(s, nil, func(sample replay.Sample, main, work, back *render.Surface) {
		seen = append(seen, sample.Frame)
		mains = append(mains, planeHash(main))
		works = append(works, planeHash(work))
		backs = append(backs, planeHash(back))
		if kept == nil {
			kept, keptHash = main, planeHash(main)
		}
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(seen) != len(res.Samples) {
		t.Fatalf("the watch saw %d frames and the trace has %d samples", len(seen), len(res.Samples))
	}
	for i, f := range seen {
		if f != res.Samples[i].Frame {
			t.Fatalf("watch call %d was frame %d, the trace's is %d", i, f, res.Samples[i].Frame)
		}
	}
	if got := res.Samples[9].Renders; got < 2 {
		t.Errorf("frame 9 rendered %d times: this script is supposed to cross a transition "+
			"there, and without one the test is not pinning anything", got)
	}

	// The Back plane changes on the transition frame and nowhere else in this run, which is
	// what proves the watch is being handed *this frame's* composition rather than one
	// end-of-run surface: room 4's background is one hash for frames 0-8 and room 5's is
	// another from frame 9 on.
	if backs[8] == backs[9] {
		t.Errorf("the composition is %s on both sides of the room change: the watch is not "+
			"seeing a per-frame Back", backs[8])
	}
	for i := 10; i < len(backs); i++ {
		if backs[i] != backs[9] {
			t.Errorf("frame %d composes room 5 differently (%s, was %s): nothing in this "+
				"run redraws the background after the transition", i, backs[i], backs[9])
			break
		}
	}
	// And frame 0 is the composition with nothing over it yet: the first render happens
	// before anything has animated, so the composite *is* the background.
	if works[0] != backs[0] {
		t.Errorf("frame 0's work map (%s) already differs from the background (%s) before "+
			"anything moved", works[0], backs[0])
	}

	// The last frame's Main is what the player was left looking at, and it is deliberately
	// **not** Result.Planes.Main. Two things happen after that last Present: CopyRectsQD
	// restores Back over Work (so the end state's Work is the erase, not the frame), and
	// then PlayGame's unconditional arcade block blackens the scoreboard band and blits it
	// to the screen (Play.c:551-593, scoreboard.go's arcadeBlackenBoard) -- so the end state
	// includes twenty rows no frame ever presented.
	//
	// That gap is the reason a per-frame corpus is worth building at all: Planes is the
	// state the process was left in, and internal/fidelity's rows are the pictures the game
	// actually showed. Pinned as an inequality because the day they *do* match, either the
	// arcade block stopped running or the shot is being taken after the erase.
	last := len(mains) - 1
	if mains[last] == res.Planes.Main {
		t.Errorf("the last frame's Main and the end state both hash %s: the teardown's "+
			"scoreboard blit is missing, or the watch is being called too late", mains[last])
	}
	if works[last] == res.Planes.Work {
		t.Errorf("the last frame's work map and the end state both hash %s: the frame's "+
			"composite is supposed to be captured before the Back2Work restore", works[last])
	}

	// And the surfaces are a reused scratch copy, which is the contract a watcher has to
	// obey by cloning: the pointer from the first frame does not still hold the first frame.
	if kept != nil && planeHash(kept) == keptHash {
		t.Error("the surface handed to the watch still holds frame 0 at the end of the run; " +
			"Watch documents a reused buffer and internal/fidelity clones on that basis")
	}
}

// TestWatchingDoesNotChangeTheRun is the package's one rule, applied to its newest hook. A
// harness that observes the picture must not be able to move a pixel of it.
func TestWatchingDoesNotChangeTheRun(t *testing.T) {
	s := script(t, "duct.script")
	s.Frames = 60

	plain, err := replay.Run(s)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	// A watcher that does the worst thing a watcher can do: writes to what it was handed.
	// Because the surfaces are copies, this must be invisible to the run.
	watched, err := replay.RunWatching(s, nil, func(_ replay.Sample, main, work, back *render.Surface) {
		for i := range main.Pix {
			main.Pix[i] = 0xFF
		}
		for i := range work.Pix {
			work.Pix[i] = 0xFF
		}
		for i := range back.Pix {
			back.Pix[i] = 0xFF
		}
	})
	if err != nil {
		t.Fatalf("run watching: %v", err)
	}
	if plain.Digest != watched.Digest {
		t.Errorf("watching changed the trace: %s became %s", plain.Digest, watched.Digest)
	}
	if plain.Planes != watched.Planes {
		t.Errorf("watching changed the picture:\n unwatched %+v\n watched   %+v", plain.Planes, watched.Planes)
	}
	if plain.Audio.Digest != watched.Audio.Digest {
		t.Errorf("watching changed the mix: %s became %s", plain.Audio.Digest, watched.Audio.Digest)
	}
}

// The attract mode.
//
// A `demo` script replays a recorded keystroke log through the ordinary physics, which makes it
// the sharpest determinism test in the package: the recording is 1117 records long and any drift
// in the frame clock, the input pass or the random stream shows up as a glider going somewhere
// the recording never went. See testdata/demo.script, which also explains why these tests assert
// the *shape* of the run and not the frame numbers it currently reaches.

// TestTheDemoReplaysTheSameWayTwice is TestTheSameScriptTwiceIsTheSameRun with the input coming
// from a 1994 file instead of from the script.
//
// It is a strictly stronger claim than the duct script's, for one reason: the keystroke timeline
// is a lookup keyed on the frame number, so a run whose frame counter drifted would still read
// the same keys, while the demo cursor is an *equality* test against a recorded frame. A cursor
// that arrives one frame late consumes nothing for the rest of the stream, and Consumed says so.
func TestTheDemoReplaysTheSameWayTwice(t *testing.T) {
	s := script(t, "demo.script")

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
	if a.Planes != b.Planes {
		t.Errorf("pictures differ:\n  %+v\n  %+v", a.Planes, b.Planes)
	}
	if a.Demo != b.Demo {
		t.Errorf("demo reports differ:\n  %+v\n  %+v", a.Demo, b.Demo)
	}

	// The stream itself: the file the script named, whole, and actually consumed. Records is
	// the one exact number here, because it is a property of the resource rather than of the
	// simulation -- if it ever changes, the extractor or the codec changed.
	if a.Demo.Records != demo.ShippedRecords {
		t.Errorf("replayed %d records, want the shipped %d", a.Demo.Records, demo.ShippedRecords)
	}
	if a.Demo.Consumed == 0 {
		t.Fatal("the run consumed no records: the cursor never matched a frame, so this test " +
			"compared two runs of a game with no input")
	}
	if a.Demo.Path != s.Demo {
		t.Errorf("Demo.Path is %q, want the script's %q", a.Demo.Path, s.Demo)
	}
	// Deliberately not an equality: how far the recording gets is a fidelity fact about the
	// physics (docs/IMPROVEMENTS.md 2.18), and pinning it here would make an improvement to
	// the physics fail a test about determinism. Logged instead, so that `go test -v` reports
	// the number and a bisect can watch it move.
	t.Logf("consumed %d of %d records in %d frames, %d frames past the end; game over %v, mortals %d",
		a.Demo.Consumed, a.Demo.Records, a.Frames, a.Demo.PastEnd, a.GameOver, a.Mortals)

	// And the trace says all of it, because that is what somebody comparing two machines sends.
	var buf bytes.Buffer
	if err := a.Trace(&buf); err != nil {
		t.Fatalf("trace: %v", err)
	}
	text := buf.String()
	for _, want := range []string{
		"# demo path=" + s.Demo,
		fmt.Sprintf("records=%d", demo.ShippedRecords),
		fmt.Sprintf("# demo consumed=%d of %d", a.Demo.Consumed, a.Demo.Records),
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the trace does not report %q:\n%s", want, firstLines(text, 6))
		}
	}
}

// TestTheDemoChangesTheRun is the other half: the recording is reaching the simulation.
//
// Without it, every assertion above would pass on a harness that loaded the stream, reported it
// and never applied a key -- which is exactly the bug a `demo` keyword can have.
func TestTheDemoChangesTheRun(t *testing.T) {
	with := script(t, "demo.script")
	with.Frames = 400 // long past the first record at frame 46
	played, err := replay.Run(with)
	if err != nil {
		t.Fatalf("with the demo: %v", err)
	}

	without := script(t, "demo.script")
	without.Frames = 400
	without.Demo = ""
	idle, err := replay.Run(without)
	if err != nil {
		t.Fatalf("without the demo: %v", err)
	}

	if played.Digest == idle.Digest {
		t.Errorf("both runs digest to %s: the recording is loaded and reported but never "+
			"reaches the glider", played.Digest)
	}
	if played.Planes == idle.Planes {
		t.Error("both runs end on the same picture; 400 frames of recorded input moved nothing")
	}
	if idle.Demo != (replay.Demo{}) {
		t.Errorf("a run with no demo reported %+v, want the zero value", idle.Demo)
	}
	if played.Demo.Consumed == 0 {
		t.Error("400 frames consumed no records; the first is frame 46")
	}
}

// TestAGameKeyAbortsTheDemo covers the arcade behaviour, through the whole stack.
//
// Input.c's BUILD_ARCADE_VERSION block is on in the original's shipped build: during a demo, any
// of player one's four game keys sets `playing = false`, which is how a passer-by who touches the
// keyboard gets a real game instead of watching the recording. So a `demo` script that also holds
// a key ends when the key goes down -- and the harness must not be the thing that swallows it,
// because in the shell that keypress is the difference between an attract mode and a stuck game.
func TestAGameKeyAbortsTheDemo(t *testing.T) {
	const abortAt = 100

	full := script(t, "demo.script")
	full.Frames = 400
	long, err := replay.Run(full)
	if err != nil {
		t.Fatalf("unaborted: %v", err)
	}
	if long.Frames < 400 {
		t.Fatalf("the control run ended early at frame %d; it is the baseline for the abort",
			long.Frames)
	}

	s := script(t, "demo.script")
	s.Frames = 400
	s.Input = []replay.Hold{{Frame: abortAt, P1: player.Keys{Band: true}}}
	short, err := replay.Run(s)
	if err != nil {
		t.Fatalf("aborted: %v", err)
	}

	// PlayGame's loop tests `playing` at the top, so the frame the key went down on is the last
	// one simulated: a couple of frames of slack, not a range.
	if short.Frames < abortAt || short.Frames > abortAt+2 {
		t.Errorf("the run ended at frame %d, want the demo to abort at %d", short.Frames, abortAt)
	}
	if short.Demo.Consumed == 0 || short.Demo.Consumed >= long.Demo.Consumed {
		t.Errorf("the aborted run consumed %d records and the full one %d; want a prefix",
			short.Demo.Consumed, long.Demo.Consumed)
	}
	// Not a game over: the demo ended, the game did not. In the shell this is where the glider
	// is handed to the player.
	if short.GameOver {
		t.Error("aborting the demo reported game over; the arcade path stops the demo, not the game")
	}
}

// Two players on one keyboard.
//
// Stage 1.9's determinism half. The unit tests for the handshake live where the code does --
// player.TestTwoPlayerRaceForTheCeiling, game.TestTheSurvivorIsDraggedThroughTheDoorway and the
// rest -- and each pins one rule from a world built for it. These pin the *whole* thing running:
// one house, one keyboard, two gliders, six deaths out of one counter, and a game that ends by
// itself. See testdata/two.script, which reads the run frame by frame and is the document to
// start from if one of these fails.

// TestTwoPlayersMatchTheGoldenTrace is the two-player golden, and it also asserts the one thing
// about the run that no single line of the golden states: that it stopped on its own.
//
// The script asks for six hundred frames and the trace is 426 long, because the sixth death
// flagged the game over and PlayGame's loop noticed. That is why the budget is unreachable on
// purpose -- a regression that left a two-player game running after both players were out of
// lives would show up here as a longer trace rather than as a subtly wrong line in the middle of
// this one.
func TestTwoPlayersMatchTheGoldenTrace(t *testing.T) {
	s := script(t, "two.script")
	if !s.TwoPlayer {
		t.Fatal("testdata/two.script is not a two-player script; the tail columns this golden " +
			"exists for are gated on Sample.Two")
	}

	res, err := replay.Run(s)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	checkGolden(t, res, "two.trace")

	if !res.GameOver {
		t.Errorf("the run ended at frame %d of %d without a game over; two players out of "+
			"lives must end the game", res.Frames, s.Frames)
	}
	if res.Frames >= int64(s.Frames) {
		t.Errorf("the run used its whole %d-frame budget, so it was cut off rather than "+
			"finished; the game over is what is being tested", s.Frames)
	}
	// -2 and not 0: OffAMortal decrements past zero once per player, and the second time
	// through is what distinguishes "one player is out" from "the game is over".
	if res.Mortals != -2 {
		t.Errorf("the run ended with mortals %d, want -2", res.Mortals)
	}
}

// TestTheTwoPlayerTailIsOnlyInTwoPlayerTraces pins the trace format's one conditional, which is a
// compatibility promise rather than a matter of taste.
//
// Sample.line appends nine columns when the run had two gliders and none when it had one. The
// alternative -- always emit them -- would put seven constant, meaningless columns in every
// one-player trace in the repository and force every golden on file to be re-blessed to gain
// them. So the shared prefix has to stay a prefix, byte for byte, and that is what this checks:
// the field *names* of a two-player line are a one-player line's names followed by exactly the
// nine that two players add.
func TestTheTwoPlayerTailIsOnlyInTwoPlayerTraces(t *testing.T) {
	line := func(twoPlayer bool) (string, []string) {
		s := localAssets(t, replay.NewScript("CD Demo House", 20))
		s.Room = 2
		s.TwoPlayer = twoPlayer
		res, err := replay.Run(s)
		if err != nil {
			t.Fatalf("two=%v: %v", twoPlayer, err)
		}
		var buf bytes.Buffer
		if err := res.Trace(&buf); err != nil {
			t.Fatalf("trace: %v", err)
		}
		for _, ln := range strings.Split(buf.String(), "\n") {
			if strings.HasPrefix(ln, "f=") {
				var names []string
				for _, field := range strings.Fields(ln) {
					names = append(names, field[:strings.Index(field, "=")])
				}
				return ln, names
			}
		}
		t.Fatalf("two=%v: the trace has no sample lines", twoPlayer)
		return "", nil
	}

	solo, soloNames := line(false)
	duo, duoNames := line(true)

	if strings.Contains(solo, "mode2=") {
		t.Errorf("a one-player sample carries the two-player tail:\n  %s", solo)
	}
	if !strings.Contains(duo, "mode2=") {
		t.Fatalf("a two-player sample has no tail at all, so the comparison below would pass "+
			"vacuously:\n  %s", duo)
	}
	tail := []string{"mode2", "dest2", "esc", "arect", "first", "oneleft", "dead"}
	if len(duoNames) != len(soloNames)+len(tail) {
		t.Fatalf("a two-player line has %d fields and a one-player line %d; want exactly %d more"+
			"\n  solo %v\n  duo  %v", len(duoNames), len(soloNames), len(tail), soloNames, duoNames)
	}
	for i, name := range soloNames {
		if duoNames[i] != name {
			t.Fatalf("field %d is %q in a one-player line and %q in a two-player one; the shared "+
				"prefix has to stay a prefix\n  solo %v\n  duo  %v",
				i, name, duoNames[i], soloNames, duoNames)
		}
	}
	for i, name := range tail {
		if got := duoNames[len(soloNames)+i]; got != name {
			t.Errorf("tail field %d is %q, want %q", i, got, name)
		}
	}

	// And the promise kept, on disk: the one-player goldens in this directory predate the tail
	// and must not have gained it. A failure here means somebody regenerated them from a
	// two-player run, which would make the diff unreadable rather than merely wrong.
	for _, name := range []string{"duct.trace", "demo.trace"} {
		body, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			continue // demo.trace is optional; the demo tests report its absence
		}
		if bytes.Contains(body, []byte("mode2=")) {
			t.Errorf("testdata/%s is a one-player golden and carries two-player columns", name)
		}
	}
}

// TestEachPlayersKeysAreReadSeparately is the harness half of "two players on one keyboard": the
// script has two key columns and they have to arrive at two different gliders.
//
// Every assertion in the golden test above would pass on a harness that read only the first
// column and drove both gliders from it, or that merged the two into one timeline -- the run
// would still be deterministic, still end in a game over, still match whatever golden it
// produced. What that harness could not do is tell these four runs apart.
func TestEachPlayersKeysAreReadSeparately(t *testing.T) {
	run := func(name string, mutate func([]replay.Hold)) *replay.Result {
		t.Helper()
		s := script(t, "two.script")
		s.Frames = 200 // past the second hold at frame 100 and the divergence it causes
		if len(s.Input) != 2 {
			t.Fatalf("testdata/two.script has %d holds, want 2 (frames 0 and 100)", len(s.Input))
		}
		mutate(s.Input)
		res, err := replay.Run(s)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return res
	}

	base := run("as written", func([]replay.Hold) {})
	// Player one turns around and player two's column is untouched.
	movedP1 := run("player one changed", func(h []replay.Hold) {
		h[0].P1 = player.Keys{Left: true}
		h[1].P1 = player.Keys{Left: true}
	})
	// Player two never takes the hint from the refusal and keeps walking into the closed wall.
	movedP2 := run("player two changed", func(h []replay.Hold) {
		h[1].P2 = player.Keys{Left: true}
	})
	// The same two timelines, swapped between the players. Not symmetric, because NewGame idles
	// player two for the first thirty frames and player one is free from frame one -- so a
	// harness that merged the columns into one would produce the same run here and a faithful
	// one cannot.
	swapped := run("swapped", func(h []replay.Hold) {
		for i := range h {
			h[i].P1, h[i].P2 = h[i].P2, h[i].P1
		}
	})

	for _, c := range []struct {
		name, why string
		digest    string
	}{
		{"player one changed", "player one's column never reaches player one's glider", movedP1.Digest},
		{"player two changed", "player two's column never reaches player two's glider", movedP2.Digest},
		{"swapped", "the two columns are being merged into one timeline", swapped.Digest},
	} {
		if c.digest == base.Digest {
			t.Errorf("%q digests the same as the script as written (%s): %s", c.name, c.why, c.digest)
		}
	}
	if movedP1.Digest == movedP2.Digest {
		t.Errorf("changing player one's keys and changing player two's produce the same run %s; "+
			"the keys are reaching one glider, not two", movedP1.Digest)
	}
}

// TestTheTwoPlayerTraceSeesTheHandshake reads the golden run as statements about the game rather
// than as 426 opaque lines.
//
// The golden pins every value, which makes it a perfect regression test and a poor explanation:
// a failure says "line 78 changed" and leaves the reader to work out what line 78 was for. This
// asserts the *shape* -- two idle freezes of the same documented length, one limbo wait, one
// refusal at the far wall, one joint crossing, one counter drained by six deaths, one terminal
// -69 -- so that a change to the handshake fails with a sentence.
//
// The frame numbers are found rather than written down, for the reason testdata/demo.script
// gives: how far a glider gets is a fidelity fact, and an improvement to the physics would move
// every number here while leaving every rule intact.
func TestTheTwoPlayerTraceSeesTheHandshake(t *testing.T) {
	res, err := replay.Run(script(t, "two.script"))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(res.Samples) < 400 {
		t.Fatalf("%d samples; the run is meant to reach a two-player game over around frame 426",
			len(res.Samples))
	}

	// The room change, which everything else is placed relative to.
	change := -1
	for i := 1; i < len(res.Samples); i++ {
		if res.Samples[i].Room != res.Samples[i-1].Room {
			change = i
			break
		}
	}
	if change < 0 {
		t.Fatal("the run never changes room, so the mismatched-exit race it exists to show " +
			"never resolved")
	}

	// # The two idle freezes
	//
	// TagGliderIdle stores its countdown in HVel and HandleIdleGlider decrements it, so the
	// freeze lasts IdleFrames-1 *sampled* frames: it is armed before the first frame runs, and
	// the frame that empties the counter has already left the mode by the time Present sees it.
	// Both freezes are the same length, and that is the point of counting them together --
	// NewGame's is at the start of the game (Play.c:198-203) and MoveRoomToRoom's is at the
	// arrival, and one implementation serves both.
	type block struct{ from, to int }
	var idle []block
	for i, sm := range res.Samples {
		if sm.Mode2 != player.GliderIdle {
			continue
		}
		if n := len(idle); n > 0 && idle[n-1].to == i-1 {
			idle[n-1].to = i
			continue
		}
		idle = append(idle, block{i, i})
	}
	if len(idle) != 2 {
		t.Errorf("player two was idled %d times, want 2 (the start of the game and the arrival "+
			"in the next room); blocks %v", len(idle), idle)
	}
	for _, b := range idle {
		if n := b.to - b.from + 1; n != int(player.IdleFrames)-1 {
			t.Errorf("the freeze at frame %d lasted %d frames, want IdleFrames-1 = %d",
				res.Samples[b.from].Frame, n, player.IdleFrames-1)
		}
	}
	if len(idle) > 0 && res.Samples[idle[0].from].Frame != 1 {
		t.Errorf("the first freeze starts on frame %d, want 1: a two-player game begins with "+
			"both gliders on the same pixel and freezes one of them",
			res.Samples[idle[0].from].Frame)
	}
	if len(idle) > 1 && idle[1].from != change {
		t.Errorf("the second freeze starts at frame %d and the room changes at frame %d; the "+
			"arrival freeze is supposed to be the room change",
			res.Samples[idle[1].from].Frame, res.Samples[change].Frame)
	}
	// Frozen means frozen: player two does not move for the whole of the first block. Without
	// this the length above would pass on a mode label that nothing acted on.
	for _, b := range idle {
		for i := b.from + 1; i <= b.to; i++ {
			if res.Samples[i].Dest2 != res.Samples[b.from].Dest2 {
				t.Errorf("player two moved from %v to %v during the freeze at frame %d",
					res.Samples[b.from].Dest2, res.Samples[i].Dest2, res.Samples[i].Frame)
				break
			}
		}
	}

	// # The limbo wait, and who chose the door
	limbo := -1
	for i, sm := range res.Samples {
		if sm.Mode == player.GliderInLimbo {
			limbo = i
			break
		}
	}
	if limbo < 0 {
		t.Fatal("player one never reached limbo; nobody waited at a doorway and the race below " +
			"cannot have happened")
	}
	first := res.Samples[limbo]
	if first.Escaped != player.PlayerEscapedRight {
		t.Errorf("frame %d put player one in limbo with Escaped %d, want PlayerEscapedRight (%d): "+
			"the glider walked out of the open right wall",
			first.Frame, first.Escaped, player.PlayerEscapedRight)
	}
	if first.First != player.Player1 {
		t.Errorf("frame %d put player one in limbo and FirstPlayer is player two; the glider that "+
			"escapes first is the one that chooses", first.Frame)
	}
	if limbo >= change {
		t.Errorf("player one entered limbo at frame %d and the room changed at frame %d; the "+
			"wait is supposed to come first", first.Frame, res.Samples[change].Frame)
	}

	// # The refusal
	//
	// The acceptance clause: a wall exit refuses a glider whose partner left by a different
	// route. Player two walks into the *other* open wall while player one is waiting at this
	// one, and kDontExitSound is the whole of the game's feedback for it.
	refused := -1
	for i := limbo; i < change; i++ {
		if strings.Contains(res.Samples[i].Sounds, "dont-exit") {
			refused = i
			break
		}
	}
	if refused < 0 {
		t.Error("no dont-exit sound between the limbo wait and the room change: player two " +
			"reached the far wall and was let through, or never reached it")
	} else {
		sm := res.Samples[refused]
		if sm.Mode != player.GliderInLimbo || sm.Escaped != player.PlayerEscapedRight {
			t.Errorf("frame %d played dont-exit with mode %d and Escaped %d; the refusal is only "+
				"meaningful while the other glider is still waiting at its own door",
				sm.Frame, sm.Mode, sm.Escaped)
		}
		if sm.Room != first.Room {
			t.Errorf("frame %d played dont-exit in room %d, having started in room %d",
				sm.Frame, sm.Room, first.Room)
		}
	}

	// # The crossing
	//
	// One frame, both gliders, and four things that have to be true at once.
	cross, before := res.Samples[change], res.Samples[change-1]
	if before.Mode != player.GliderInLimbo {
		t.Errorf("the frame before the room change has player one in mode %d, not limbo; the "+
			"room changed for some other reason than the second glider agreeing", before.Mode)
	}
	if cross.Escaped != player.NoOneEscaped {
		t.Errorf("the room changed with Escaped still %d; the handshake is not cleared, so the "+
			"next doorway will admit a glider on a stale code", cross.Escaped)
	}
	if cross.Mode == player.GliderInLimbo {
		t.Error("player one is still in limbo after the room changed")
	}
	if cross.Mode2 != player.GliderIdle {
		t.Errorf("player two arrived in mode %d, want GliderIdle: MoveRoomToRoom freezes "+
			"whoever is not FirstPlayer, and FirstPlayer is player one here", cross.Mode2)
	}
	if cross.First != player.Player1 {
		t.Error("FirstPlayer changed across the room change; the glider that chose the door is " +
			"the one that is placed rather than frozen")
	}
	if d := cross.Renders - before.Renders; d != 2 {
		t.Errorf("the transition frame advanced RenderFrames by %d, want 2", d)
	}

	// # One counter, six deaths
	//
	// The shared-inventory rule as the whole game shows it: Mortals starts at two players'
	// worth of one three-life counter and walks down one step at a time, and *both* gliders
	// take steps out of it. A port that gave each player their own counter would produce a
	// column that stalled at 1 while somebody still had lives left.
	want := []int16{4, 3, 2, 1, 0, -1, -2}
	var got []int16
	byGlider := map[string]int{}
	for i, sm := range res.Samples {
		if i == 0 || sm.Mortals != res.Samples[i-1].Mortals {
			got = append(got, sm.Mortals)
		}
		if i == 0 || sm.Mortals >= res.Samples[i-1].Mortals {
			continue
		}
		// Who paid. OffAMortal runs at the end of a fade-out, so the glider that was dying on
		// the previous frame is the one the counter came out of.
		switch {
		case res.Samples[i-1].Mode == player.GliderFadingOut:
			byGlider["player one"]++
		case res.Samples[i-1].Mode2 == player.GliderFadingOut:
			byGlider["player two"]++
		default:
			t.Errorf("mortals dropped to %d on frame %d with neither glider fading out "+
				"(modes %d and %d)", sm.Mortals, sm.Frame,
				res.Samples[i-1].Mode, res.Samples[i-1].Mode2)
		}
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("the mortal counter went %v, want %v: one counter, one step per death", got, want)
	}
	for _, who := range []string{"player one", "player two"} {
		if byGlider[who] == 0 {
			t.Errorf("every death in the run came out of the other glider; %s never drew on the "+
				"shared counter, so this run does not show it being shared", who)
		}
	}
	t.Logf("six deaths out of one counter: %v", byGlider)

	// # Dead forever
	//
	// PlayerIsDeadForever is the original's own constant and the thing that stops the survivor
	// waiting at doorways for a player who cannot come. It is written once and never cleared,
	// which is what the tail of this loop asserts.
	//
	// `dead=0` in the trace is not "nobody": player.Player2 *is* false, so the column reads the
	// same whether player two is the dead one or the game has not come down to one player at
	// all. That is why it is only read here, inside the OneLeft gate -- see Sample.Dead.
	out := -1
	for i, sm := range res.Samples {
		if sm.OneLeft {
			out = i
			break
		}
	}
	if out < 0 {
		t.Fatal("no frame reported OneLeft; both players kept their lives, and the six deaths " +
			"above went somewhere else")
	}
	last := res.Samples[out]
	if last.Escaped != player.PlayerIsDeadForever {
		t.Errorf("frame %d is the first with one player left and Escaped is %d, want "+
			"PlayerIsDeadForever (%d)", last.Frame, last.Escaped, player.PlayerIsDeadForever)
	}
	if last.Dead != player.Player2 {
		t.Errorf("frame %d says player one is the one out of lives; player two dies first in "+
			"this script", last.Frame)
	}
	if last.Mode2 != player.GliderInLimbo {
		t.Errorf("player two is in mode %d after spending its last mortal, want GliderInLimbo "+
			"(%d): OffAMortal parks it in limbo with DontDraw rather than removing it",
			last.Mode2, player.GliderInLimbo)
	}
	if last.Mortals != -1 {
		t.Errorf("frame %d reports one player left with mortals %d, want -1", last.Frame, last.Mortals)
	}
	for _, sm := range res.Samples[out:] {
		if !sm.OneLeft {
			t.Errorf("frame %d cleared OneLeft; a player out of lives does not get any back",
				sm.Frame)
			break
		}
		if sm.Escaped != player.PlayerIsDeadForever {
			t.Errorf("frame %d cleared Escaped to %d; PlayerIsDeadForever is written once and "+
				"survives to the next NewGame", sm.Frame, sm.Escaped)
			break
		}
	}
}

// planeHash is the test's own hash of a surface: the same shape as the package's internal one
// and the corpus's, kept separate because a test that shares the function it is checking
// cannot catch a change to it.
func planeHash(s *render.Surface) string {
	if s == nil {
		return "-"
	}
	sum := sha256.New()
	fmt.Fprintf(sum, "%dx%d\n", s.W, s.H)
	sum.Write(s.Pix)
	return hex.EncodeToString(sum.Sum(nil))[:16]
}

// firstLines keeps a failure message readable: a trace is hundreds of lines and the header is
// where the answer is.
func firstLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
