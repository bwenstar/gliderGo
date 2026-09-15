package player

// Input, from GliderPRO/Sources/Input.c.
//
// The original reads the hardware key map into one file-scope KeyMap and then tests
// per-glider bit indices out of it, so the two players in a two-player game share a
// single poll: GetInput calls GetKeys only when `which == kPlayer1`, and player 2's
// pass a moment later reads the snapshot player 1 took. That is how two gliders are
// driven from one keyboard without either seeing a half-updated state.
//
// The port keeps the property and makes it explicit: the caller polls once per frame
// and hands each glider its own resolved Keys. That also removes the original's
// ordering hazard, where whichever glider happened to be handled first decided what
// the other one saw.

// Keys is one frame of input for one glider, already resolved from the key map.
//
// Left, Right, Batt and Band are this glider's four configurable keys. The other
// three are not per-player in the original either: Command, Delete and the pause key
// are tested against the raw key map regardless of whose turn it is.
//
// Pause is the resolved answer to "is the pause key down", because the original picks
// between Escape and Tab from the isEscPauseKey preference at every one of its four
// test sites (Input.c:93, :100, :115, :271, :373). The preference belongs to settings,
// not to the player, so it is resolved before it gets here.
type Keys struct {
	Left, Right, Batt, Band bool
	Command, Delete, Pause  bool
}

// DemoKey is the code the demo recorder logs for one action: the `char key` of a
// `demoType` record (GliderStructs.h:334-339).
//
// The four values are internal/demo.Key's, spelled out again here so that this package can
// stay import-free -- it has no imports at all, which is what lets the glider be tested
// without a house, a scene or an asset tree. A test in internal/game pins the two sets
// together, because a port that got them out of step would fly the demo mirror-imaged and
// nothing would fail to compile.
type DemoKey byte

// The four codes, in the order GetInput's branches log them.
//
// **Right is 0 and left is 1**, which is the opposite of what `GetDemoInput`'s own case
// comments say (Input.c:228, :235). The recorder's call sites are authoritative because they
// produced the shipped `'demo'` resource: `LogDemoKey(0)` is in the rightKey branch at
// Input.c:304 and `(1)` is in the leftKey branch at :321. docs/analysis/input.md §14.2.
const (
	DemoRight DemoKey = 0
	DemoLeft  DemoKey = 1
	DemoBatt  DemoKey = 2
	DemoBand  DemoKey = 3
)

// Input is Input.c's own file-scope state: the two variables that throttle the
// battery and helium sound effects.
//
// They are globals in the original, which means they are shared by both players --
// and so is batteryTotal, since the two players draw on one inventory. Keeping them
// together in a value the caller owns preserves the sharing without making them
// package-level, so a test can drive a glider without disturbing another one.
//
// The corollary is a caller obligation the type cannot enforce: a two-player game
// must pass the *same* Input to both gliders. Give each its own and the sound
// throttle below is wrong in a way that is easy to miss, because it is right for one
// player. See Play.c:452-453, where the two GetInput calls are back to back.
type Input struct {
	// batteryFrame counts 0..3 and re-triggers the thrust or hiss sound every
	// fourth frame, which is what makes a held battery key sound like a repeating
	// pulse rather than one sample restarting 30 times a second.
	//
	// That cadence is the one-player cadence. Because this field and the flag below
	// are shared, a two-player game with one thruster finds batteryWasEngaged false
	// at every DoBatteryEngaged call -- the other player's GetInput cleared it
	// (Input.c:341-342) -- so batteryFrame is forced back to 0 and the sound plays
	// every frame. With both thrusting it advances twice per frame, so the sound
	// plays every second frame and batteryTotal drains at 2 per frame.
	batteryFrame int16

	// batteryWasEngaged is true if the most recent GetInput call took the battery
	// branch -- for *either* glider, since PlayGame calls GetInput twice back to
	// back (Play.c:452-453). It is what resets batteryFrame to 0 on the first frame
	// of a new press, so a fresh press always sounds immediately rather than waiting
	// out the previous cycle's remainder.
	//
	// The exception is a burning glider, which returns before the reset, so one
	// player catching fire restores the other's ordinary four-frame cadence.
	batteryWasEngaged bool

	// LogDemoKey is the demo recorder, and it is the `#ifdef CREATEDEMODATA` call sites
	// of Input.c:304, :321, :334 and :348 turned into a nil-checked function pointer.
	//
	// nil in a normal game, which is the original's shipped configuration -- the recorder
	// was a compile-time option John Calhoun turned on once to make `'demo'` 128 and left
	// off in the released build. A hook rather than a build tag because there is no reason
	// a released binary should not be able to record its own demo (that is what
	// internal/demo's Recorder is for), and because a compile-time switch in a port is a
	// second binary nobody tests.
	//
	// It is called from *inside* the branches below rather than from the caller, and the
	// exact positions matter: the C logs before the both-keys test and before the
	// fire-held test, so an about-face records as a plain right press and a held band key
	// records a code on every frame. Both are why a replayed demo is not a replay of the
	// session that recorded it. See the four call sites, and internal/demo's Recorder for
	// the one-record-per-frame rule the stream needs and the C's recorder did not keep.
	LogDemoKey func(key DemoKey)
}

// logDemo is the nil check, so the four call sites read like the C's one-liners.
func (in *Input) logDemo(k DemoKey) {
	if in.LogDemoKey != nil {
		in.LogDemoKey(k)
	}
}

// GetInput is Input.c:281-379: one frame of input for one glider.
//
// The structure is one big if/else on whether the glider is burning, and the burning
// branch does exactly one thing: apply full thrust in the direction the glider faces.
// Everything else -- the direction keys, the battery, the bands, the abandon key and
// the pause key -- is in the else. So a burning glider cannot be steered, cannot fire,
// cannot use its battery, and cannot even be paused. The two seconds it has left play
// out at full speed whatever the player does.
//
// Note also what the burning branch does not touch: HeldLeft, HeldRight, Tipped and
// FireHeld all keep whatever value they had on the frame the fire started, for the
// whole burn. HeldLeft and HeldRight gate the stair transitions, so a glider that
// caught fire while walking right is still refused the up-stairs for the rest of its
// life.
func (in *Input) GetInput(g *Glider, e Env, k Keys) {
	if g.Which == Player1 && k.Command {
		e.DoCommandKey()
	}

	if g.Mode == GliderBurning {
		if g.Facing == FaceLeft {
			g.HDesiredVel -= NormalThrust
		} else {
			g.HDesiredVel += NormalThrust
		}
		return
	}

	// ---- direction keys (Input.c:299-328) ----
	//
	// Right is tested first and its branch is exclusive, so with both keys down the
	// left-key branch never runs. Both down is the about-face gesture instead.
	//
	// Tipped is `facing == the opposite of the key`, i.e. "thrusting against the way
	// I am pointing", which is the bank. It picks the tipped sprite and it inverts
	// the battery's direction.
	g.HeldLeft = false
	g.HeldRight = false
	//
	// The recorder is logged in both right-hand arms and logs DemoRight for both, because
	// the C's LogDemoKey(0) sits at the top of `if (rightKey)` -- above the both-keys test
	// at Input.c:306. So the about-face gesture is recorded as an ordinary right press and
	// replays as one: the demo stream has no way to express it at all
	// (docs/analysis/input.md §14.3, difference 3).
	switch {
	case k.Right && k.Left:
		in.logDemo(DemoRight)
		g.ToggleGliderFacing()
		// HeldLeft, not HeldRight: an about-face is refused the down-stairs and
		// allowed the up-stairs. Both keys are down, so either would be defensible;
		// the original picks left (Input.c:309) and the choice is observable.
		g.HeldLeft = true
	case k.Right:
		in.logDemo(DemoRight)
		g.HDesiredVel += NormalThrust
		g.Tipped = g.Facing == FaceLeft
		g.HeldRight = true
	case k.Left:
		in.logDemo(DemoLeft)
		g.HDesiredVel -= NormalThrust
		g.Tipped = g.Facing == FaceRight
		g.HeldLeft = true
	default:
		g.Tipped = false
	}

	// ---- battery and helium (Input.c:330-342) ----
	//
	// One key, two power-ups, decided by the sign of the shared counter. Both are
	// refused outside normal mode, so neither works during an about-face or a foil
	// dissolve even though those modes do run the integrator.
	if k.Batt && e.BatteryTotal() != 0 && g.Mode == GliderNormal {
		in.logDemo(DemoBatt)
		if e.BatteryTotal() > 0 {
			in.DoBatteryEngaged(g, e)
		} else {
			in.DoHeliumEngaged(g, e)
		}
	} else {
		in.batteryWasEngaged = false
	}

	// ---- rubber bands (Input.c:344-364) ----
	//
	// FireHeld is set only when AddBand succeeded, so a shot refused because the
	// band array is full costs nothing and will be retried next frame. It is cleared
	// in the else, which is what makes the key one-shot: releasing it for a single
	// frame re-arms.
	if k.Band && e.BandsTotal() > 0 && g.Mode == GliderNormal {
		// Above the fire-held test, as at Input.c:348: a held key logs a record every
		// frame even though only the first one fires. Playback reproduces the single shot
		// anyway -- case 3 does not clear FireHeld -- so the extra records are harmless
		// there and merely make the stream longer than the session's shots.
		in.logDemo(DemoBand)
		if !g.FireHeld {
			if e.AddBand(g, g.Dest.Left+BandSpawnH, g.Dest.Top+BandSpawnV, g.Facing) {
				e.SetBandsTotal(e.BandsTotal() - 1)
				if e.BandsTotal() <= 0 {
					e.QuickBandsRefresh(false)
				}
				g.FireHeld = true
			}
		}
	} else {
		g.FireHeld = false
	}

	// ---- abandon the other player (Input.c:366-371) ----
	//
	// Two-player only, and only player 1 may press it: `thisGlider->which` is the
	// test, and Player1 is the true value. It is the escape hatch for when one player
	// has already left the room and the other cannot or will not follow -- it kills
	// the straggler so the game can continue.
	if e.OtherPlayerEscaped() != NoOneEscaped && k.Delete && g.Which == Player1 && !e.OnePlayerLeft() {
		e.ForceKillGlider()
	}

	if k.Pause {
		e.DoPause()
	}
}

// DoBatteryEngaged is Input.c:121-156: one frame of the battery.
//
// HyperThrust is added straight to HVel, bypassing the ramp, which is why the battery
// feels like a kick rather than an acceleration. MoveGlider still clamps the result to
// MaxHVel, so with a direction key also held the steady state is exactly 16 px/frame
// and the surplus is thrown away every frame.
//
// The direction is the thrust direction, not the facing: a tipped glider is banked
// against the way it points, and the battery follows the bank. So holding right while
// facing left gets you +8, not -8.
func (in *Input) DoBatteryEngaged(g *Glider, e Env) {
	if g.Facing == FaceLeft {
		if g.Tipped {
			g.HVel += HyperThrust
		} else {
			g.HVel -= HyperThrust
		}
	} else {
		if g.Tipped {
			g.HVel -= HyperThrust
		} else {
			g.HVel += HyperThrust
		}
	}

	e.SetBatteryTotal(e.BatteryTotal() - 1)

	if e.BatteryTotal() == 0 {
		e.QuickBatteryRefresh(false)
		e.PlayPrioritySound(FizzleSound, FizzlePriority)
		// No `batteryWasEngaged = false` here, unlike DoHeliumEngaged. The
		// asymmetry is in the original (Input.c:140-144 against :165-170) and is
		// reproduced. It is reachable: it leaves the flag set on the frame the
		// battery runs dry, so if the player then picks up helium and holds the key
		// again, batteryFrame is not reset and the hiss starts out of phase.
		return
	}
	in.tickBatterySound(e, ThrustSound, ThrustPriority)
}

// DoHeliumEngaged is Input.c:160-182: one frame of helium.
//
// Nothing like the battery mechanically. Helium assigns VDesiredVel rather than
// touching VVel, so it goes through the ramp and it does not stack -- it is the same
// channel a floor vent writes, and the last writer in the frame wins. Rising is
// therefore 4 px/frame against falling's 3, reached in two frames.
//
// The counter is incremented, not decremented: BatteryTotal is one signed slot for
// both power-ups and helium is the negative half, so both functions walk it toward
// zero and both fizzle when they arrive.
func (in *Input) DoHeliumEngaged(g *Glider, e Env) {
	g.VDesiredVel = -HeliumLift
	e.SetBatteryTotal(e.BatteryTotal() + 1)

	if e.BatteryTotal() == 0 {
		e.QuickBatteryRefresh(false)
		e.PlayPrioritySound(FizzleSound, FizzlePriority)
		in.batteryWasEngaged = false
		return
	}
	in.tickBatterySound(e, HissSound, HissPriority)
}

// tickBatterySound is the eight lines both power-ups end with (Input.c:146-155,
// :171-181): sound on frame 0 of a four-frame cycle, and a fresh press restarts the
// cycle so it always sounds at once.
func (in *Input) tickBatterySound(e Env, sound, priority int16) {
	if !in.batteryWasEngaged {
		in.batteryFrame = 0
	}
	if in.batteryFrame == 0 {
		e.PlayPrioritySound(sound, priority)
	}
	in.batteryFrame++
	if in.batteryFrame >= BatteryFrames {
		in.batteryFrame = 0
	}
	in.batteryWasEngaged = true
}
