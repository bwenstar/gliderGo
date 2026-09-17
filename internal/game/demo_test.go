package game

// The attract-mode input path: GetDemoInput against GetInput, plus the recorder hook.
//
// This file is organised around one claim: **the demo path is not the keyboard path with a
// different key source.** docs/analysis/input.md §14.3 lists eight differences between
// Input.c:186-277 and Input.c:281-379, and a port that unified the two functions would pass
// every test in handle_test.go and still fly the recorded demo off course. Five of the eight
// are observable from here and each has a test that runs the *same* input through both
// functions and asserts they disagree:
//
//	difference 2/3  no both-keys case, and Tipped cleared before the switch
//	difference 4    the battery works with no battery and outside normal mode
//	difference 5    the band works with no bands, and can drive the count negative
//	difference 6    a band record does not clear FireHeld, so a held key is one shot
//	difference 7    the abandon key is not tested
//
// The remaining three are structural -- no DoCommandKey (8), the arcade abort (1), the
// two-player arm not existing -- and the first two are here too.
//
// TestDemoKeyCodesMatchThePlayerPackage is the cheapest test in the file and the one most
// worth having: player.DemoKey exists only so that internal/game/player can stay import-free,
// and if the two enumerations ever drift the port records left as right and nothing fails to
// compile.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/demo"
	"github.com/bwenstar/gliderGo/internal/game/player"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// demoWorld is dynaWorld with a stream loaded and player one ready to be driven by it.
//
// Bands and Battery are stocked, because the interesting tests are the ones that empty them:
// a fixture that started at zero could not tell "the guard is missing" from "the guard is
// there and the inventory happened to be empty".
func demoWorld(t *testing.T, recs ...demo.Record) *World {
	t.Helper()
	w := dynaWorld(t)
	w.R.RoomNumber = 0
	w.Bands = 5
	w.Battery = 5
	w.Playing = true
	w.DemoGoing = true
	w.Demo = demo.Stream(recs).Cursor()
	w.P1.Which = player.Player1
	w.P1.Mode = player.GliderNormal
	w.P1.Facing = player.FaceRight
	return w
}

// rec is a record with the padding byte a recorder would write.
func rec(frame int64, k demo.Key) demo.Record { return demo.Record{Frame: frame, Key: k} }

// keysDown makes the world's keyboard answer the same thing on every frame, for both
// gliders, and counts the polls.
func keysDown(w *World, k player.Keys) *int {
	n := 0
	w.KeyPoll = func(*player.Glider) player.Keys {
		n++
		return k
	}
	return &n
}

// ---------------------------------------------------------------------------
// The wire codes
// ---------------------------------------------------------------------------

// TestDemoKeyCodesMatchThePlayerPackage pins the two copies of the four key codes together.
//
// internal/game/player declares player.DemoKey and internal/demo declares demo.Key with the
// same four values, because the player package has no imports at all -- that is what lets a
// glider be tested without a house, a scene or an asset tree (see player/input.go). The
// conversion in World.RecordDemo is a plain byte cast, so a drift here would be silent: the
// recorder would write 1 where it meant 0 and the demo would replay mirror-imaged.
func TestDemoKeyCodesMatchThePlayerPackage(t *testing.T) {
	for _, c := range []struct {
		name string
		mine player.DemoKey
		wire demo.Key
	}{
		{"right", player.DemoRight, demo.KeyRight},
		{"left", player.DemoLeft, demo.KeyLeft},
		{"batt", player.DemoBatt, demo.KeyBatt},
		{"band", player.DemoBand, demo.KeyBand},
	} {
		if demo.Key(c.mine) != c.wire {
			t.Errorf("%s: player.DemoKey %d, demo.Key %d", c.name, byte(c.mine), byte(c.wire))
		}
		if c.wire.String() != c.name {
			t.Errorf("demo.Key(%d).String() = %q, want %q", byte(c.wire), c.wire.String(), c.name)
		}
	}
	// And the absolute values, because they are a file format and not an internal
	// convention: right is 0 because LogDemoKey(0) is in the right-key branch.
	if player.DemoRight != 0 || player.DemoLeft != 1 || player.DemoBatt != 2 || player.DemoBand != 3 {
		t.Fatalf("the codes moved: %d %d %d %d",
			player.DemoRight, player.DemoLeft, player.DemoBatt, player.DemoBand)
	}
}

// ---------------------------------------------------------------------------
// Playback
// ---------------------------------------------------------------------------

func TestGetDemoInputAppliesTheFramesRecord(t *testing.T) {
	w := demoWorld(t, rec(1, demo.KeyRight), rec(3, demo.KeyLeft))

	// Frame 1: the right record. Facing right, so no bank.
	w.Frame = 1
	w.P1.FireHeld = true
	w.GetDemoInput(&w.P1)
	if got := w.P1.HDesiredVel; got != player.NormalThrust {
		t.Errorf("HDesiredVel %d, want %d", got, player.NormalThrust)
	}
	if !w.P1.HeldRight || w.P1.HeldLeft || w.P1.Tipped {
		t.Errorf("right record: held %v/%v tipped %v", w.P1.HeldLeft, w.P1.HeldRight, w.P1.Tipped)
	}
	if w.P1.FireHeld {
		t.Error("a direction record must clear FireHeld")
	}

	// Frame 2 has no record. That is not "no input": it clears the held flags and FireHeld,
	// which is the state a keyless frame leaves behind and is observable through the stairs.
	w.Frame = 2
	w.P1.HDesiredVel = 0
	w.P1.FireHeld = true
	w.GetDemoInput(&w.P1)
	if w.P1.HDesiredVel != 0 {
		t.Errorf("a keyless frame thrust by %d", w.P1.HDesiredVel)
	}
	if w.P1.HeldRight || w.P1.HeldLeft || w.P1.FireHeld {
		t.Errorf("a keyless frame left held %v/%v fire %v",
			w.P1.HeldLeft, w.P1.HeldRight, w.P1.FireHeld)
	}
	if got, want := w.Demo.Index(), 1; got != want {
		t.Errorf("cursor at %d, want %d: a keyless frame must not consume a record", got, want)
	}

	// Frame 3: the left record, still facing right, so this one banks.
	w.Frame = 3
	w.P1.HDesiredVel = 0
	w.GetDemoInput(&w.P1)
	if got := w.P1.HDesiredVel; got != -player.NormalThrust {
		t.Errorf("HDesiredVel %d, want %d", got, -player.NormalThrust)
	}
	if !w.P1.HeldLeft || w.P1.HeldRight || !w.P1.Tipped {
		t.Errorf("left record while facing right: held %v/%v tipped %v",
			w.P1.HeldLeft, w.P1.HeldRight, w.P1.Tipped)
	}
	if !w.Demo.Done() {
		t.Error("the stream should be consumed")
	}
}

// TestGetDemoInputHasNoAboutFace is differences 2 and 3 together.
//
// The stream cannot express both keys at once, so the about-face gesture is unreachable from
// a demo -- and because Tipped is cleared *above* the switch rather than in a default arm, no
// demo frame can carry a bank over from the frame before either. GetInput's both-keys arm
// does exactly the opposite of both: it flips the facing and leaves Tipped alone.
func TestGetDemoInputHasNoAboutFace(t *testing.T) {
	// The keyboard path, for comparison: both keys down starts the about-face tumble and
	// preserves the bank. The *facing* does not flip here -- ToggleGliderFacing sets the
	// three-frame GliderFaceLeft mode and MoveGliderFacing is what flips it at the end of
	// the tumble -- so the mode is what says the gesture was accepted.
	w := demoWorld(t)
	w.P1.Tipped = true
	keysDown(w, player.Keys{Left: true, Right: true})
	w.GetInput(&w.P1)
	if w.P1.Mode != player.GliderFaceLeft {
		t.Fatalf("GetInput's both-keys arm left mode %d, want the about-face tumble %d",
			w.P1.Mode, player.GliderFaceLeft)
	}
	if !w.P1.Tipped {
		t.Fatal("GetInput's both-keys arm cleared Tipped; the demo comparison below is void")
	}
	if !w.P1.HeldLeft || w.P1.HeldRight {
		t.Fatalf("GetInput's both-keys arm set held %v/%v, want left only",
			w.P1.HeldLeft, w.P1.HeldRight)
	}

	// The demo path, with the same keyboard held down and the recorded key that an
	// about-face is *logged* as (LogDemoKey(0), from above the both-keys test).
	d := demoWorld(t, rec(1, demo.KeyRight))
	d.Frame = 1
	d.P1.Tipped = true
	keysDown(d, player.Keys{Left: true, Right: true})
	d.GetDemoInput(&d.P1)
	if d.P1.Mode != player.GliderNormal || d.P1.Facing != player.FaceRight {
		t.Errorf("the demo started an about-face (mode %d, facing %v); the stream has no way "+
			"to say that", d.P1.Mode, d.P1.Facing)
	}
	if d.P1.Tipped {
		t.Error("the demo preserved a bank across a frame; Tipped is cleared before the switch")
	}
	if d.P1.HeldLeft || !d.P1.HeldRight {
		t.Errorf("the demo's right record set held %v/%v, want right only",
			d.P1.HeldLeft, d.P1.HeldRight)
	}
}

// TestGetDemoInputBatteryIgnoresItsGuards is difference 4.
//
// GetInput refuses the battery unless `batteryTotal != 0 && mode == kGliderNormal`.
// GetDemoInput tests neither, so a record can spend a battery the glider does not have --
// which lands in DoHeliumEngaged, because the sign of the shared counter is what chooses --
// and can spend one during an about-face tumble.
func TestGetDemoInputBatteryIgnoresItsGuards(t *testing.T) {
	t.Run("no battery becomes helium", func(t *testing.T) {
		// The keyboard first: an empty counter is refused outright.
		w := demoWorld(t)
		w.Battery = 0
		keysDown(w, player.Keys{Batt: true})
		w.GetInput(&w.P1)
		if w.Battery != 0 || w.P1.VDesiredVel != 0 {
			t.Fatalf("GetInput with no battery: counter %d, lift %d -- want both 0",
				w.Battery, w.P1.VDesiredVel)
		}

		// The demo: the same state, and the glider gets a helium charge it never
		// collected. The counter goes *up*, which is the negative half of the slot.
		d := demoWorld(t, rec(1, demo.KeyBatt))
		d.Frame = 1
		d.Battery = 0
		d.GetDemoInput(&d.P1)
		if d.Battery != 1 {
			t.Errorf("counter %d after a battery record with none in stock, want 1", d.Battery)
		}
		if d.P1.VDesiredVel != -player.HeliumLift {
			t.Errorf("VDesiredVel %d, want %d (helium)", d.P1.VDesiredVel, -player.HeliumLift)
		}
	})

	t.Run("outside normal mode", func(t *testing.T) {
		// GliderFaceLeft is the about-face tumble: a real, reachable, three-frame mode
		// that the integrator runs and that GetInput refuses the battery in.
		w := demoWorld(t)
		w.P1.Mode = player.GliderFaceLeft
		keysDown(w, player.Keys{Batt: true})
		w.GetInput(&w.P1)
		if w.Battery != 5 || w.P1.HVel != 0 {
			t.Fatalf("GetInput mid-tumble: counter %d, HVel %d -- want 5 and 0",
				w.Battery, w.P1.HVel)
		}

		d := demoWorld(t, rec(1, demo.KeyBatt))
		d.Frame = 1
		d.P1.Mode = player.GliderFaceLeft
		d.GetDemoInput(&d.P1)
		if d.Battery != 4 {
			t.Errorf("counter %d after a battery record mid-tumble, want 4", d.Battery)
		}
		if d.P1.HVel != player.HyperThrust {
			t.Errorf("HVel %d, want %d", d.P1.HVel, player.HyperThrust)
		}
	})
}

// TestGetDemoInputBandIgnoresItsGuards is difference 5: no ammunition test, so the count
// goes negative and the scoreboard's `<= 0` refresh is called on every shot after the first.
func TestGetDemoInputBandIgnoresItsGuards(t *testing.T) {
	w := demoWorld(t)
	w.Bands = 0
	keysDown(w, player.Keys{Band: true})
	w.GetInput(&w.P1)
	if w.NumBands != 0 || w.Bands != 0 {
		t.Fatalf("GetInput with no bands fired %d (count %d)", w.NumBands, w.Bands)
	}

	d := demoWorld(t, rec(1, demo.KeyBand), rec(2, demo.KeyBand))
	d.Frame = 1
	d.Bands = 0
	d.GetDemoInput(&d.P1)
	if d.NumBands != 1 {
		t.Fatalf("%d bands in flight after a band record with none in stock, want 1", d.NumBands)
	}
	if d.Bands != -1 {
		t.Errorf("band count %d, want -1: the demo path has no ammunition test", d.Bands)
	}
	if !d.P1.FireHeld {
		t.Error("a successful shot must set FireHeld")
	}

	// And difference 6: the band case does not clear FireHeld, so the record on the very
	// next frame fires nothing. A held key is one shot however many records it logged.
	d.Frame = 2
	d.GetDemoInput(&d.P1)
	if d.NumBands != 1 {
		t.Errorf("%d bands after a second consecutive band record, want 1", d.NumBands)
	}
	if d.Bands != -1 {
		t.Errorf("band count %d after a refused shot, want -1", d.Bands)
	}
	if !d.Demo.Done() {
		t.Error("the second record should still have been consumed")
	}

	// A keyless frame re-arms, exactly as releasing the key does on the keyboard.
	d.Frame = 3
	d.GetDemoInput(&d.P1)
	if d.P1.FireHeld {
		t.Error("a keyless frame did not re-arm the band key")
	}
}

// TestGetDemoInputUnknownKeyIsConsumedAndDoesNothing pins the missing default arm.
//
// A record with a key outside 0..3 is consumed by the frame test and then falls through the
// switch: no thrust, no shot -- and, because the FireHeld clear lives in the `else` of the
// frame test rather than in the switch, **the fire flag survives**. That is why
// demo.Key.Valid exists as a report rather than as a filter.
func TestGetDemoInputUnknownKeyIsConsumedAndDoesNothing(t *testing.T) {
	w := demoWorld(t, demo.Record{Frame: 1, Key: demo.Key(7)})
	w.Frame = 1
	w.P1.FireHeld = true
	w.P1.HDesiredVel = 0
	w.GetDemoInput(&w.P1)

	if w.P1.HDesiredVel != 0 || w.NumBands != 0 || w.Battery != 5 {
		t.Errorf("key 7 did something: thrust %d, bands %d, battery %d",
			w.P1.HDesiredVel, w.NumBands, w.Battery)
	}
	if !w.P1.FireHeld {
		t.Error("key 7 cleared FireHeld; the clear is in the else of the frame test, not the switch")
	}
	if !w.Demo.Done() {
		t.Error("key 7 was not consumed")
	}
	if w.Diag.Guarded != 0 {
		t.Errorf("Guarded %d: an unknown key is not an out-of-range read", w.Diag.Guarded)
	}
}

// TestGetDemoInputBurningIgnoresTheStream is the one branch the two paths share, and it has a
// consequence worth pinning: a burning glider returns *before* the cursor is read, so the
// records for those frames are never consumed and are then unreachable for ever. A demo whose
// glider catches fire has diverged, and this is the mechanism.
func TestGetDemoInputBurningIgnoresTheStream(t *testing.T) {
	w := demoWorld(t, rec(1, demo.KeyBand), rec(2, demo.KeyBand))
	w.Frame = 1
	w.P1.Mode = player.GliderBurning
	w.P1.Facing = player.FaceLeft
	w.P1.FireHeld = true
	w.P1.HeldRight = true

	w.GetDemoInput(&w.P1)

	if got := w.P1.HDesiredVel; got != -player.NormalThrust {
		t.Errorf("HDesiredVel %d, want %d: a burning glider thrusts the way it faces",
			got, -player.NormalThrust)
	}
	if w.NumBands != 0 {
		t.Errorf("a burning glider fired %d bands", w.NumBands)
	}
	if w.Demo.Index() != 0 {
		t.Errorf("cursor at %d; the burning branch returns before the cursor is read", w.Demo.Index())
	}
	// And the flags it does not touch, which is what keeps a glider that caught fire while
	// walking right out of the up-stairs for the rest of its life.
	if !w.P1.HeldRight || !w.P1.FireHeld {
		t.Errorf("the burning branch cleared held %v or fire %v", w.P1.HeldRight, w.P1.FireHeld)
	}

	// Frame 2's record is now stranded behind frame 1's, which the equality test can never
	// match again.
	w.Frame = 2
	w.P1.Mode = player.GliderNormal
	w.GetDemoInput(&w.P1)
	if w.NumBands != 0 {
		t.Error("frame 2's record fired after the cursor was stranded on frame 1's")
	}
}

// TestDemoAbortsOnAnyGameKey is difference 1: the arcade build's abort.
//
// BUILD_ARCADE_VERSION is on in the released source drop, so any of player one's four game
// keys ends the demo. Two things about it are easy to get wrong and both are asserted here:
// it does **not** return, so the frame's recorded input is still applied; and it clears
// Paused as well as Playing, which is what lets a paused demo be ended by a keypress.
func TestDemoAbortsOnAnyGameKey(t *testing.T) {
	for _, c := range []struct {
		name string
		k    player.Keys
		stop bool
	}{
		{"left", player.Keys{Left: true}, true},
		{"right", player.Keys{Right: true}, true},
		{"batt", player.Keys{Batt: true}, true},
		{"band", player.Keys{Band: true}, true},
		// Command is not one of the four: in the arcade build DoCommandKey is not even
		// reached from here (difference 8), so Command-Q cannot quit a demo. That call is
		// unobservable in this port -- World.DoCommandKey is deliberately empty, see
		// env.go -- so what is asserted is the part that matters: the demo runs on.
		{"command", player.Keys{Command: true}, false},
		{"delete", player.Keys{Delete: true}, false},
		{"none", player.Keys{}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := demoWorld(t, rec(1, demo.KeyRight))
			w.Frame = 1
			w.Paused = true
			keysDown(w, c.k)

			w.GetDemoInput(&w.P1)

			if got := !w.Playing; got != c.stop {
				t.Errorf("Playing %v, want stopped = %v", w.Playing, c.stop)
			}
			if w.Paused != !c.stop {
				t.Errorf("Paused %v; the abort clears it too", w.Paused)
			}
			// The record is applied either way: the C sets two flags and falls through,
			// and PlayGame's loop condition is what actually stops it.
			if got := w.P1.HDesiredVel; got != player.NormalThrust {
				t.Errorf("HDesiredVel %d, want %d: the frame's record must still apply",
					got, player.NormalThrust)
			}
		})
	}
}

// TestGetDemoInputOnlyPollsPlayerOne is the transcription of Input.c:188: the keyboard is read
// inside the player-one test. A demo is one-player by construction -- PlayGame reaches this
// only in its `!twoPlayer` arm -- so this is the belt to that braces.
func TestGetDemoInputOnlyPollsPlayerOne(t *testing.T) {
	w := demoWorld(t, rec(1, demo.KeyRight))
	w.Frame = 1
	polls := keysDown(w, player.Keys{Left: true})

	w.P2.Which = player.Player2
	w.P2.Mode = player.GliderNormal
	w.GetDemoInput(&w.P2)
	if *polls != 0 {
		t.Errorf("player two's pass polled the keyboard %d times", *polls)
	}
	if !w.Playing {
		t.Error("player two's keys aborted the demo")
	}
	// The record was still applied to player two, which is the C's behaviour and is
	// unreachable in a real game.
	if w.P2.HDesiredVel != player.NormalThrust {
		t.Errorf("player two's thrust %d, want %d", w.P2.HDesiredVel, player.NormalThrust)
	}

	w.Frame = 2
	w.GetDemoInput(&w.P1)
	if *polls != 1 {
		t.Errorf("player one's pass polled %d times, want 1", *polls)
	}
}

// TestGetDemoInputPauses pins the last line of the function: the pause key still works during
// a demo, and it is tested after the record has been applied.
func TestGetDemoInputPauses(t *testing.T) {
	w := demoWorld(t, rec(1, demo.KeyRight))
	w.Frame = 1
	paused := 0
	w.Pause = func(paint func()) { paused++ }
	keysDown(w, player.Keys{Pause: true})

	w.GetDemoInput(&w.P1)

	if paused != 1 {
		t.Errorf("the pause loop ran %d times, want 1", paused)
	}
	if w.P1.HDesiredVel != player.NormalThrust {
		t.Error("the record was not applied before the pause")
	}
	if !w.Playing {
		t.Error("the pause key aborted the demo; it is not one of the four game keys")
	}
}

// TestDemoPastEndIsReportedOnce is the deviation demoKey documents.
//
// The original reads `demoData[demoIndex]` off the end of its block once per frame for the
// rest of the game. The port refuses and counts -- but only the first time, because
// Diagnostics.Guarded counts every call and a few hundred identical events would bury the
// room-object and trigger guards that a bug report is actually about.
func TestDemoPastEndIsReportedOnce(t *testing.T) {
	w := demoWorld(t, rec(1, demo.KeyRight))
	var seen []Deviation
	w.Diag.On = func(d Deviation) { seen = append(seen, d) }

	for f := int64(1); f <= 50; f++ {
		w.Frame = f
		w.GetDemoInput(&w.P1)
	}

	if got := w.Demo.PastEnd(); got != 49 {
		t.Errorf("PastEnd %d, want 49: every frame after the last record reads past the end", got)
	}
	if w.Diag.Guarded != 1 {
		t.Errorf("Guarded %d, want 1: the demo overrun is reported once per run", w.Diag.Guarded)
	}
	if len(seen) != 1 || seen[0].Kind != devDemoRecord {
		t.Fatalf("deviations %v, want one %q", seen, devDemoRecord)
	}
	if seen[0].Index != 1 || seen[0].Limit != 1 {
		t.Errorf("deviation %v, want demo record[1] of 1", seen[0])
	}
}

// TestDemoWithNoStreamIsAGameWithNoInput pins the nil-cursor path, which is what a build with
// no extracted assets has: `'demo'` 128 is not there, the shell has no stream, and the attract
// mode still has to run.
func TestDemoWithNoStreamIsAGameWithNoInput(t *testing.T) {
	w := demoWorld(t)
	w.Demo = nil
	w.P1.FireHeld = true

	for f := int64(1); f <= 10; f++ {
		w.Frame = f
		w.GetDemoInput(&w.P1)
	}

	if w.P1.HDesiredVel != 0 || w.NumBands != 0 || w.Battery != 5 {
		t.Errorf("a demo with no stream did something: thrust %d bands %d battery %d",
			w.P1.HDesiredVel, w.NumBands, w.Battery)
	}
	if w.P1.FireHeld {
		t.Error("a demo with no stream must still clear FireHeld every frame")
	}
	if w.Diag.Guarded != 0 {
		t.Errorf("Guarded %d: having no demo is not a deviation", w.Diag.Guarded)
	}
}

// ---------------------------------------------------------------------------
// Recording
// ---------------------------------------------------------------------------

// TestRecordDemoLogsTheKeyboardPath drives GetInput -- the keyboard path -- and checks the
// stream that comes out, including the two places where what is recorded is not what happened.
func TestRecordDemoLogsTheKeyboardPath(t *testing.T) {
	w := demoWorld(t)
	r := w.RecordDemo()

	// Frame 1: right. Frame 2: an about-face, which logs as a plain right press because
	// LogDemoKey(0) sits above the both-keys test. Frame 3: left. Frame 4: the battery.
	// Frames 5 and 6: the band key held, which logs twice and fires once.
	type frame struct {
		f int64
		k player.Keys
	}
	for _, fr := range []frame{
		{1, player.Keys{Right: true}},
		{2, player.Keys{Right: true, Left: true}},
		{3, player.Keys{Left: true}},
		{4, player.Keys{Batt: true}},
		{5, player.Keys{Band: true}},
		{6, player.Keys{Band: true}},
		{7, player.Keys{}},
	} {
		w.Frame = fr.f
		keysDown(w, fr.k)
		// The mode is put back to normal at the top of every frame because frame 2's
		// about-face leaves the glider in the three-frame GliderFaceLeft tumble, and both
		// the battery and the band are refused outside normal mode. In a real session those
		// three frames pass; here there is no HandleGlider to pass them, so a fixture that
		// left the mode alone would record nothing after frame 3 and the interesting half of
		// this test would silently not run.
		w.P1.Mode = player.GliderNormal
		w.GetInput(&w.P1)
	}

	want := demo.Stream{
		{Frame: 1, Key: demo.KeyRight},
		{Frame: 2, Key: demo.KeyRight}, // the about-face, flattened
		{Frame: 3, Key: demo.KeyLeft},
		{Frame: 4, Key: demo.KeyBatt},
		{Frame: 5, Key: demo.KeyBand},
		{Frame: 6, Key: demo.KeyBand}, // logged, and fired nothing
	}
	got := r.Stream()
	if len(got) != len(want) {
		t.Fatalf("%d records, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("record %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if r.Dropped() != 0 {
		t.Errorf("Dropped %d, want 0: no frame here had two actions", r.Dropped())
	}
	if err := got.Validate(); err != nil {
		t.Errorf("the recorded stream does not validate: %v", err)
	}

	// Frame 6 logged a band record and fired nothing, because FireHeld was still set. That
	// is the asymmetry player/input.go's LogDemoKey comment describes, and it is why a
	// stream has more band records than the session had shots.
	if w.NumBands != 1 {
		t.Errorf("%d bands fired across two held frames, want 1", w.NumBands)
	}
}

// TestRecordDemoKeepsOneRecordPerFrame is the invariant the original's recorder could not
// keep: LogDemoKey is called from four independent branches, so a player holding right while
// firing a band logged two records on one frame -- and playback's equality test would then
// stall on the second for ever. The port's recorder keeps the first and counts the rest.
func TestRecordDemoKeepsOneRecordPerFrame(t *testing.T) {
	w := demoWorld(t)
	r := w.RecordDemo()

	// Right, battery and band all down on one frame: three LogDemoKey calls, in the C's
	// order -- direction, battery, band.
	w.Frame = 1
	keysDown(w, player.Keys{Right: true, Batt: true, Band: true})
	w.GetInput(&w.P1)

	got := r.Stream()
	if len(got) != 1 {
		t.Fatalf("%d records from one frame: %+v", len(got), got)
	}
	if got[0] != (demo.Record{Frame: 1, Key: demo.KeyRight}) {
		t.Errorf("record %+v, want frame 1 right: the first branch to log wins", got[0])
	}
	if r.Dropped() != 2 {
		t.Errorf("Dropped %d, want 2", r.Dropped())
	}

	// All three actions still happened -- the recorder is a tap, not a filter.
	if w.P1.HDesiredVel != player.NormalThrust || w.Battery != 4 || w.NumBands != 1 {
		t.Errorf("the frame itself was altered: thrust %d battery %d bands %d",
			w.P1.HDesiredVel, w.Battery, w.NumBands)
	}
}

// TestGetDemoInputRecordsNothing pins the other half of the recorder's wiring: the demo path
// has no LogDemoKey calls at all, so a demo game that happened to have a recorder installed
// appends nothing. A demo that recorded itself would write a second copy of the stream it was
// playing.
func TestGetDemoInputRecordsNothing(t *testing.T) {
	w := demoWorld(t, rec(1, demo.KeyRight), rec(2, demo.KeyBand))
	r := w.RecordDemo()

	for f := int64(1); f <= 2; f++ {
		w.Frame = f
		w.GetDemoInput(&w.P1)
	}

	if n := len(r.Stream()); n != 0 {
		t.Errorf("%d records from a demo game, want 0: %+v", n, r.Stream())
	}
	if w.P1.HDesiredVel == 0 || w.NumBands == 0 {
		t.Fatal("the demo did nothing, so this test proved nothing")
	}
}
