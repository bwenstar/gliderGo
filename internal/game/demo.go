package game

// The attract-mode demo: GetDemoInput (Input.c:186-277) and DoDemoGame (Play.c:282-303).
//
// The stream itself -- the six-byte records, the cursor, the recorder -- is internal/demo.
// This file is the twenty lines that sit between it and the glider, and every one of them is a
// place where the original's demo path differs from its keyboard path. `GetDemoInput` is not
// `GetInput` with a different source of keys; it is a second, sloppier copy of it, and
// docs/analysis/input.md §14.3 lists the eight differences a port must not unify away. Five of
// them are visible in the function below and marked where they happen.
//
// Read it beside player.Input.GetInput. The temptation, when both are on screen, is to build
// one function with a `demo bool` -- and that would quietly re-add the guards the demo path
// does not have, which is exactly how a port ends up with an attract mode that cannot spend a
// band it does not own and therefore diverges from the recording on frame 1442.

import (
	"github.com/bwenstar/gliderGo/internal/demo"
	"github.com/bwenstar/gliderGo/internal/game/player"
)

// GetDemoInput is Input.c:186-277: drive player one's glider from the recorded stream.
//
// One record per frame at most, and the only frame a record applies to is the one whose number
// it holds exactly -- see demo.Cursor.Key. A frame with no record is not "no input": it clears
// FireHeld, which is what re-arms the band key, so the difference between a gap and a record is
// observable.
//
// With no stream loaded this is the whole of the original's behaviour with an empty one: every
// frame reads past the end, nothing is applied, and the glider flies on physics alone. That is
// the state the port shipped in before this stage and it is still reachable -- a build with no
// extracted assets has no `'demo'` resource -- so it is a supported path rather than an error.
func (w *World) GetDemoInput(g *player.Glider) {
	// GetKeys(theKeys), inside the player-one test exactly as at Input.c:188-190. Only
	// player one is ever passed here (PlayGame's demo branch is in the one-player arm), so
	// the test is transcription rather than logic.
	var k player.Keys
	if g.Which == player.Player1 {
		if w.KeyPoll != nil {
			k = w.KeyPoll(g)
		}

		// BUILD_ARCADE_VERSION (Input.c:192-201): **any of the four game keys ends the
		// demo**, and Command-Q/S do nothing. The arcade build is the shipped
		// configuration -- see arcadeBlackenBoard -- so this is the live arm, and it is
		// what makes an attract mode feel right: a player who touches the controls gets
		// the menu back instead of watching a ghost fly.
		//
		// Note what it does *not* do: return. The frame's recorded input is still
		// applied and the pause key is still tested, because the C only sets the two
		// flags and falls through. PlayGame's loop condition is what actually stops,
		// one render later.
		if k.Left || k.Right || k.Batt || k.Band {
			w.Playing = false
			w.Paused = false
		}
	}

	// Difference 8: DoCommandKey is not reached in the arcade build, so Command-Q cannot
	// quit a demo and Command-S cannot save one. The non-arcade arm (Input.c:205-206) is
	// deliberately not transcribed -- it is the other half of a #if, and the port picks the
	// same half everywhere else.

	if g.Mode == player.GliderBurning {
		// Identical to GetInput's burning branch, and it is the one thing the two paths
		// really do share: a burning glider is unsteerable whatever is driving it.
		if g.Facing == player.FaceLeft {
			g.HDesiredVel -= player.NormalThrust
		} else {
			g.HDesiredVel += player.NormalThrust
		}
		return
	}

	// Difference 2: Tipped is cleared here, *before* the switch, where GetInput clears it
	// only in the no-key arm of its own switch. Combined with difference 3 -- there is no
	// both-keys case -- that means a demo can never about-face and never carries a bank
	// across a keyless frame.
	g.HeldLeft = false
	g.HeldRight = false
	g.Tipped = false

	if key, ok := w.demoKey(); ok {
		// No default arm, deliberately. A record with a key outside 0..3 is consumed and
		// does nothing, and in particular does **not** clear FireHeld -- the clear lives
		// in the else below, not in the switch (Input.c:226-269). Adding a default that
		// cleared it would be tidier and wrong.
		switch key {
		case demo.KeyRight:
			// Right, despite the C's comment saying left. See demo.Key.
			g.HDesiredVel += player.NormalThrust
			g.Tipped = g.Facing == player.FaceLeft
			g.HeldRight = true
			g.FireHeld = false
		case demo.KeyLeft:
			g.HDesiredVel -= player.NormalThrust
			g.Tipped = g.Facing == player.FaceRight
			g.HeldLeft = true
			g.FireHeld = false
		case demo.KeyBatt:
			// Difference 4: no `BatteryTotal != 0` and no `Mode == GliderNormal`. With
			// the counter at zero this calls DoHeliumEngaged, which increments it to 1
			// and hisses -- so a demo record can hand the glider a helium charge it
			// never picked up. Unreachable with the shipped stream, which contains no
			// battery records at all (910 right, 198 left, 9 band, 0 batt), and
			// reachable by any stream this port records.
			if w.BatteryTotal() > 0 {
				w.In.DoBatteryEngaged(g, w)
			} else {
				w.In.DoHeliumEngaged(g, w)
			}
			g.FireHeld = false
		case demo.KeyBand:
			// Difference 5: no `BandsTotal > 0` and no mode test, so this can drive the
			// band count negative -- AddBand only needs a free slot in the array. The
			// scoreboard's refresh is called on `<= 0` and would then be called again on
			// every subsequent shot.
			if !g.FireHeld {
				if w.AddBand(g, g.Dest.Left+player.BandSpawnH, g.Dest.Top+player.BandSpawnV, g.Facing) {
					w.SetBandsTotal(w.BandsTotal() - 1)
					if w.BandsTotal() <= 0 {
						w.QuickBandsRefresh(false)
					}
					g.FireHeld = true
				}
			}
			// No FireHeld = false here, which is difference 6's other half: a band
			// record on the frame after a successful shot finds the flag still set and
			// fires nothing, so a held key is one shot even in a stream that logged it
			// every frame.
		}
	} else {
		g.FireHeld = false
	}

	// Difference 7: the delete-suicide key is not tested at all, so a demo cannot abandon a
	// player -- there is no second player in a demo anyway (PlayGame reaches this only in
	// its one-player arm).

	if k.Pause {
		w.DoPause()
	}
}

// demoKey is the cursor read, with the original's out-of-bounds read stood in for.
//
// The C indexes `demoData[demoIndex]` unguarded (Input.c:224). Past the last record that is a
// read off the end of a NewPtr block: harmless on the Mac, where the comparison against heap
// slack fails and playback simply stops producing input, and a panic in Go. The cursor answers
// "no record" instead and counts the refusals.
//
// The deviation is reported **once**, on the first refusal, and that is a departure from the
// convention every other guard in this package follows. Diagnostics.Guarded counts every call,
// and a demo that outlives its stream asks once per frame until the glider dies -- hundreds of
// identical events, which would bury the room-object and trigger guards that a bug report is
// actually about. One event says the same thing: this run went past the end of its demo. See
// guards.go, and docs/IMPROVEMENTS.md 2.33 for why the exclusions are listed rather than
// silent.
func (w *World) demoKey() (demo.Key, bool) {
	if w.Demo == nil {
		return 0, false
	}
	key, ok := w.Demo.Key(w.Frame)
	if !ok && w.Demo.PastEnd() == 1 {
		w.badIndex(devDemoRecord, w.Demo.Index(), w.Demo.Len())
	}
	return key, ok
}

// DoDemoGame is Play.c:282-303: play the attract-mode demo.
//
// The C's version is nine lines of house juggling around one call: it closes the current house,
// swaps `thisHouseIndex` for `demoHouseIndex`, opens and reads that house, sets `demoGoing`,
// calls `NewGame(kNewGameMode)`, and then puts the previous house back. **Only the middle of
// that is here**, because in this port a World *is* a house -- it is built around one
// house.House and one render.Scene (NewWorld) -- so the swap is the caller's, and the caller is
// internal/shell, which already owns house selection and already knows which houses exist.
//
// That split is not just tidiness. `demoHouseIndex` is **-1 unless a file called exactly
// "Demo House" was found during the house scan** (SelectHouse.c:636-644), and the C then
// indexes `theHousesSpecs[-1]` without checking -- the attract mode of a copy of Glider PRO
// whose demo house had been moved would read one struct before the array. The port's guard is
// structural: shell.Library records the demo house as a *found file* (shell.DemoHouse), the
// menu item is disabled when there is none, and this function is never reached without a world
// built on it. See docs/analysis/unresolved-format-decisions.md §4.1.
//
// d may be nil, and then the demo is a game with no input -- see GetDemoInput. It is a real
// state, not an error: a build whose assets have not been extracted has no stream.
func (w *World) DoDemoGame(d *demo.Cursor) {
	w.Demo = d
	w.DemoGoing = true

	// NewGame resets the cursor, sets Playing, runs the whole game and clears DemoGoing on
	// its way out (Play.c:275-277). So a demo ends the way a game ends -- the glider dies,
	// the countdown runs, DoDiedGameOver takes the screen -- and the only thing that makes
	// it a demo is where the input came from.
	w.NewGame(NewGameMode)

	// `incrementModeTime = TickCount() + kIdleSplashTicks` a second time (Play.c:302).
	// NewGame's tail has already done it; the C does it again because the house reopening
	// between the two takes real time, and re-arming the idle timer afterwards is what
	// stops the splash screen from starting another demo the instant this one ends.
	w.IncrementModeTime = w.Ticks() + IdleSplashTicks
}

// RecordDemo makes this world record its own input, and returns the recorder.
//
// The hook is player.Input's, called from the four places `GetInput` calls `LogDemoKey`
// (Input.c:304, :321, :334, :348), so what is recorded is what the *keyboard path* did -- a
// demo game records nothing, which is right: GetDemoInput has no LogDemoKey calls, and a demo
// that recorded itself would append a second copy of the stream it was playing.
//
// Frame numbers are World.Frame at the moment of the press, which is what playback compares
// against. That is the whole of the format's synchronisation: no timestamps, no duration, and
// therefore no dependence on how fast the recording machine was (docs/analysis/input.md §14.8,
// which also explains how the shipped resource was produced and why it ends in garbage).
func (w *World) RecordDemo() *demo.Recorder {
	rec := demo.NewRecorder()
	w.In.LogDemoKey = func(k player.DemoKey) { rec.Log(w.Frame, demo.Key(k)) }
	return rec
}
