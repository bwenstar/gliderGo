package player

import "testing"

// The state machine and the input layer. Every expected number below was derived by
// hand from the C before the test was run, because a test that was tuned until it
// passed would only pin whatever this port happens to do.

//---------------------------------------------------------------- fades and fuses

// TestFadeInTakesSixteenFrames pins the length of a respawn and the exact frame the
// sound plays on.
//
// FadeGliderIn increments Frame *after* testing it for zero and *before* comparing it
// against LastFadeSequence, so the arithmetic works out to fifteen frames still fading
// and the sixteenth handing control back to the player. The sound is gated on
// `Frame == 0`, which is true only on the first of those, so a respawn goes ding once
// rather than sixteen times -- and the frame-0 sprite StartGliderFadingIn chose is
// never re-shown by the handler.
func TestFadeInTakesSixteenFrames(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{}
	g.StartGliderFadingIn(e)

	for i := 1; i <= 15; i++ {
		g.FadeGliderIn(e)
		if g.Mode != GliderFadingIn {
			t.Fatalf("frame %d: Mode = %d, want it still fading in", i, g.Mode)
		}
		if g.Frame != int16(i) {
			t.Fatalf("frame %d: Frame = %d, want %d", i, g.Frame, i)
		}
	}
	g.FadeGliderIn(e)
	if g.Mode != GliderNormal {
		t.Errorf("after %d frames Mode = %d, want GliderNormal", LastFadeSequence, g.Mode)
	}
	if g.EnteredRect != g.Dest {
		t.Errorf("EnteredRect = %+v, want it stamped with Dest %+v", g.EnteredRect, g.Dest)
	}
	if len(e.Sounds) != 1 || e.Sounds[0] != FadeInSound {
		t.Errorf("sounds = %v, want exactly one FadeInSound", e.Sounds)
	}
}

// TestBurnFuse pins the two seconds a burning glider has to find water.
//
// The fuse lives in WasMode, seeded at FramesToBurn by FlagGliderBurning and
// decremented once per frame; the flame animation lives in Frame at the same time.
// Two counters, two overloaded fields, one handler -- and if a port gave the fuse its
// own field it would have to remember to clear it, which is the bug WasMode's reuse
// makes impossible.
func TestBurnFuse(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{}
	g.FlagGliderBurning(e)

	if g.WasMode != FramesToBurn {
		t.Fatalf("fuse seeded at %d, want FramesToBurn %d", g.WasMode, FramesToBurn)
	}

	for i := 1; i < int(FramesToBurn); i++ {
		g.MoveGliderBurning(e)
		if g.Mode != GliderBurning {
			t.Fatalf("frame %d: Mode = %d, want it still burning", i, g.Mode)
		}
	}
	g.MoveGliderBurning(e)
	if g.Mode != GliderFadingOut {
		t.Errorf("after %d frames Mode = %d, want GliderFadingOut", FramesToBurn, g.Mode)
	}
	if len(e.Sounds) != 2 || e.Sounds[0] != CaughtFireSound || e.Sounds[1] != FadeOutSound {
		t.Errorf("sounds = %v, want [CaughtFire FadeOut]", e.Sounds)
	}
	// The flame animation cycled the whole time, and 60 is a multiple of 4, so the
	// phase is back where it started.
	if g.Dest.Tall() != GliderHigh {
		t.Errorf("Dest is %d tall after the fade started, want the flame shrunk away to %d",
			g.Dest.Tall(), GliderHigh)
	}
}

// TestBurningFlameCycles checks the other counter in isolation: Frame runs 0..3 and
// wraps, and the sprite run it picks depends on the facing.
func TestBurningFlameCycles(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{}
	g.FlagGliderBurning(e)

	for i, want := range []int16{1, 2, 3, 0, 1} {
		g.MoveGliderBurning(e)
		if g.Frame != want {
			t.Fatalf("frame %d: flame phase = %d, want %d", i+1, g.Frame, want)
		}
		if g.Src != GliderSrc[SpriteFirstBurning+want] {
			t.Errorf("frame %d: sprite is not burning frame %d", i+1, want)
		}
	}
}

//---------------------------------------------------------------- shredding

// TestShreddingIsTwoPhases is the fullest exercise of Frame's overloading in the
// whole package: it is a y coordinate for the first phase and a negative frame count
// for the second, and the handler tells them apart with `Frame > 0` alone.
//
// With the shredder at {200,100,300,200} and the glider entering at y=100:
// FlagGliderShredding teleports it to Left 136 (bounds.Left + ShredderInset) and sets
// Frame to 297 (bounds.Bottom - ShredderMouthInset), the y its whole body has to be
// ground past.
//
// The descent then has two speeds, and they are the opposite way round from their
// names: DropShredFast (4) applies while the glider is still falling toward the mouth,
// DropShredSlow (1) once the mouth has hold of it. Dest.Top climbs 100 -> 280 at 4
// px/frame -- 45 frames, silent -- and then 280 -> 296 at 1 px/frame with the shred
// sound on every one of those 17 frames. On the last of them nothing is left, the
// confetti is spawned, and Frame flips to ShredderCountdown.
func TestShreddingIsTwoPhases(t *testing.T) {
	g := newGliderAtRest(0, 100)
	e := &NopEnv{Shadow: true}
	g.FlagGliderShredding(e, Rect{Top: 200, Left: 100, Bottom: 300, Right: 200})

	if g.Dest.Left != 136 || g.Dest.Right != 184 || g.Dest.Bottom != 120 {
		t.Fatalf("teleported to %+v, want Left 136, Right 184, Bottom 120", g.Dest)
	}
	if g.Frame != 297 {
		t.Fatalf("grind target = %d, want 297", g.Frame)
	}

	// Phase one, falling: 4 px/frame, no sound, shadow still drawn.
	soundsAtStart := len(e.Sounds)
	for i := 1; i <= 45; i++ {
		top := g.Dest.Top
		g.MoveGliderShredding(e)
		if got := g.Dest.Top - top; got != DropShredFast {
			t.Fatalf("falling frame %d: moved %d, want DropShredFast %d", i, got, DropShredFast)
		}
	}
	if g.Dest.Top != 280 {
		t.Fatalf("after the fall Dest.Top = %d, want 280", g.Dest.Top)
	}
	if len(e.Sounds) != soundsAtStart {
		t.Errorf("the fall played %v, want silence until the mouth has hold",
			e.Sounds[soundsAtStart:])
	}
	if !e.Shadow {
		t.Error("the shadow was hidden before the shredder had hold")
	}

	// Phase one, grinding: 1 px/frame, a shred sound per frame, shadow hidden, and
	// Dest.Bottom pinned to the grind target as the body is eaten away.
	for i := 1; i <= 16; i++ {
		g.MoveGliderShredding(e)
		if g.Dest.Bottom != g.Frame {
			t.Fatalf("grinding frame %d: Dest.Bottom = %d, want it clipped to the target %d",
				i, g.Dest.Bottom, g.Frame)
		}
		if e.Sounds[len(e.Sounds)-1] != ShredSound {
			t.Fatalf("grinding frame %d played %d, want ShredSound", i, e.Sounds[len(e.Sounds)-1])
		}
		if e.Shadow {
			t.Fatalf("grinding frame %d: the shadow is still drawn", i)
		}
	}
	if g.Dest.Tall() != 1 {
		t.Fatalf("after 16 grinding frames the glider is %d tall, want 1", g.Dest.Tall())
	}

	// The 17th consumes what is left.
	g.MoveGliderShredding(e)
	if len(e.Shredded) != 1 {
		t.Fatalf("AddAShreddedGlider called %d times, want 1", len(e.Shredded))
	}
	if g.Frame != ShredderCountdown {
		t.Fatalf("Frame = %d, want ShredderCountdown %d", g.Frame, ShredderCountdown)
	}

	// Phase two: the same field counted back up to zero, and only then is the life
	// spent. The delay is what lets the confetti finish falling before the screen
	// changes.
	for i := 1; i < -int(ShredderCountdown); i++ {
		g.MoveGliderShredding(e)
		if e.MortalsSpent != 0 {
			t.Fatalf("life spent on countdown frame %d, want it to wait out all %d",
				i, -ShredderCountdown)
		}
	}
	g.MoveGliderShredding(e)
	if e.MortalsSpent != 1 {
		t.Errorf("MortalsSpent = %d after the countdown, want 1", e.MortalsSpent)
	}
}

//---------------------------------------------------------------- dispatch

// TestHandleGliderClearsIgnoreFlags pins where the three one-frame permissions are
// cleared: in the dispatcher, after the handler, for every mode without exception.
//
// GliderInLimbo is the case that proves it. That mode has no handler at all -- an
// empty case in the original and here -- so if the clearing lived in the handlers a
// limboed glider would keep whatever permissions it was granted on the frame it left
// the room, and would be able to walk through a wall on the frame it came back.
//
// Running every mode through the dispatcher also catches any handler that panics on a
// zero-valued glider, which is why this loop covers all twenty-four rather than the
// one that makes the point.
func TestHandleGliderClearsIgnoreFlags(t *testing.T) {
	for m := GliderNormal; m <= GliderTransportingIn; m++ {
		g := newGliderAtRest(100, 50)
		g.Mode = m
		g.IgnoreLeft, g.IgnoreRight, g.IgnoreGround = true, true, true

		e := &NopEnv{Alive: g, BandAdded: true}
		g.HandleGlider(e)

		if g.IgnoreLeft || g.IgnoreRight || g.IgnoreGround {
			t.Errorf("mode %d: ignore flags left set (%v %v %v)",
				m, g.IgnoreLeft, g.IgnoreRight, g.IgnoreGround)
		}
	}
}

// TestLimboIsFrozenNotBroken is the other half of the above: a limboed glider must
// come out of the dispatcher having moved nowhere and animated nothing, because it is
// waiting for the other player and the room around it is the other player's.
func TestLimboIsFrozenNotBroken(t *testing.T) {
	g := newGliderAtRest(100, 50)
	g.Mode = GliderInLimbo
	g.hold(9, 9) // momentum it would carry if anything integrated
	before := *g

	e := &NopEnv{}
	g.HandleGlider(e)

	if g.Dest != before.Dest || g.Frame != before.Frame || g.Mode != GliderInLimbo {
		t.Errorf("limbo advanced: Dest %+v -> %+v, Frame %d -> %d, Mode %d",
			before.Dest, g.Dest, before.Frame, g.Frame, g.Mode)
	}
	if len(e.Sounds) != 0 || len(e.Transitions) != 0 {
		t.Errorf("limbo had side effects: sounds %v, transitions %v", e.Sounds, e.Transitions)
	}
}

// TestIdleCountdownLivesInHVel pins the last of the overloaded fields. TagGliderIdle
// stores a 30-frame freeze in HVel, which is only safe because GliderIdle is not one
// of the momentum modes -- nothing integrates it, so the field is free.
func TestIdleCountdownLivesInHVel(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{}
	g.TagGliderIdle(e)

	if g.Mode != GliderIdle || g.HVel != IdleFrames || g.WasMode != GliderNormal {
		t.Fatalf("Mode = %d, HVel = %d, WasMode = %d; want GliderIdle, %d, GliderNormal",
			g.Mode, g.HVel, g.WasMode, IdleFrames)
	}
	if HasMomentum(GliderIdle) {
		t.Fatal("GliderIdle integrates velocity, so HVel cannot hold a countdown")
	}

	for i := 1; i < int(IdleFrames); i++ {
		g.HandleGlider(e)
		if g.Mode != GliderIdle {
			t.Fatalf("frame %d: Mode = %d, want it still idle", i, g.Mode)
		}
	}
	g.HandleGlider(e)
	if g.Mode != GliderNormal {
		t.Errorf("after %d frames Mode = %d, want it restored to GliderNormal",
			IdleFrames, g.Mode)
	}
	if g.DontDraw {
		t.Error("DontDraw still set after the freeze")
	}
}

//---------------------------------------------------------------- input

// TestBurningLocksOutInput pins the shape of GetInput: one if/else on whether the
// glider is burning, and the burning branch does exactly one thing before returning.
//
// So a burning glider cannot steer, fire, use its battery -- or pause. The command key
// is the single exception, because DoCommandKey is tested *before* the burning check.
//
// The subtler half is what the branch leaves alone. HeldLeft, HeldRight, Tipped and
// FireHeld all keep whatever they held on the frame the fire started, for the whole
// burn. HeldLeft and HeldRight gate the staircases, so a glider that caught fire while
// walking right is refused the up-stairs for the rest of its life.
func TestBurningLocksOutInput(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{Battery: 5, Bands: 3, BandAdded: true}
	g.FlagGliderBurning(e)
	g.HeldLeft, g.HeldRight, g.Tipped, g.FireHeld = true, true, true, true

	var in Input
	in.GetInput(g, e, Keys{
		Left: true, Right: true, Batt: true, Band: true,
		Command: true, Delete: true, Pause: true,
	})

	if g.HDesiredVel != NormalThrust {
		t.Errorf("HDesiredVel = %d, want NormalThrust %d: full thrust in the facing direction",
			g.HDesiredVel, NormalThrust)
	}
	if e.Battery != 5 || e.Bands != 3 {
		t.Errorf("battery %d, bands %d: a burning glider spent inventory", e.Battery, e.Bands)
	}
	if e.Pauses != 0 {
		t.Error("a burning glider was allowed to pause")
	}
	if e.Commands != 1 {
		t.Errorf("Commands = %d, want 1: the command key is tested before the burn check",
			e.Commands)
	}
	if !g.HeldLeft || !g.HeldRight || !g.Tipped || !g.FireHeld {
		t.Errorf("the burning branch rewrote a held-key flag: %v %v %v %v",
			g.HeldLeft, g.HeldRight, g.Tipped, g.FireHeld)
	}
}

// TestBothKeysAboutFace pins the gesture and the one observable arbitrary choice in
// it. Both direction keys down is neither left nor right: it starts an about-face,
// contributes no thrust, and sets HeldLeft rather than HeldRight.
//
// That last is a coin toss the original makes at Input.c:309, and it is visible in
// play: HeldLeft is what the up-stairs requires and the down-stairs refuses, so
// standing on a staircase and tapping both keys walks you up.
func TestBothKeysAboutFace(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{}

	var in Input
	in.GetInput(g, e, Keys{Left: true, Right: true})

	if g.Mode != GliderFaceLeft {
		t.Errorf("Mode = %d, want GliderFaceLeft: a right-facing glider turns left", g.Mode)
	}
	if g.Frame != LastAboutFaceFrame {
		t.Errorf("Frame = %d, want the tumble seeded at %d", g.Frame, LastAboutFaceFrame)
	}
	if !g.HeldLeft || g.HeldRight {
		t.Errorf("HeldLeft/HeldRight = %v/%v, want true/false", g.HeldLeft, g.HeldRight)
	}
	if g.HDesiredVel != 0 {
		t.Errorf("HDesiredVel = %d, want 0: neither key thrusts", g.HDesiredVel)
	}
}

// TestBatteryFollowsTheBank is the detail most likely to be got backwards. The
// battery's direction comes from Tipped -- the bank -- and not from Facing, so holding
// right while facing left gives +HyperThrust, not -HyperThrust.
//
// Read as a physical claim it is obvious: the thrust goes where the player is pushing.
// Read as code it is a double negative, because Tipped is defined as "facing the
// opposite way to the key" and DoBatteryEngaged then inverts on it.
func TestBatteryFollowsTheBank(t *testing.T) {
	g := newGliderAtRest(100, 50)
	g.Facing = FaceLeft
	e := &NopEnv{Battery: 5}

	var in Input
	in.GetInput(g, e, Keys{Right: true, Batt: true})

	if !g.Tipped {
		t.Fatal("Tipped should be set: the glider faces left and the key is right")
	}
	if g.HVel != HyperThrust {
		t.Errorf("HVel = %d, want +HyperThrust %d, not -%d", g.HVel, HyperThrust, HyperThrust)
	}
	if g.HDesiredVel != NormalThrust {
		t.Errorf("HDesiredVel = %d, want NormalThrust %d: the key still ramps too",
			g.HDesiredVel, NormalThrust)
	}
	if e.Battery != 4 {
		t.Errorf("battery = %d, want 4", e.Battery)
	}
	if len(e.Sounds) != 1 || e.Sounds[0] != ThrustSound {
		t.Errorf("sounds = %v, want ThrustSound", e.Sounds)
	}
}

// TestHeliumIsTheNegativeHalf pins the single signed counter. One key and one slot
// serve two power-ups: positive is battery charges, negative is helium, and the same
// keypress does completely different things either side of zero -- HVel directly for
// one, VDesiredVel through the ramp for the other.
//
// Both walk the counter toward zero, so the increment here is not a typo.
func TestHeliumIsTheNegativeHalf(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{Battery: -3}

	var in Input
	in.GetInput(g, e, Keys{Batt: true})

	if g.VDesiredVel != -HeliumLift {
		t.Errorf("VDesiredVel = %d, want -HeliumLift %d", g.VDesiredVel, -HeliumLift)
	}
	if g.HVel != 0 {
		t.Errorf("HVel = %d, want 0: helium does not thrust sideways", g.HVel)
	}
	if e.Battery != -2 {
		t.Errorf("battery = %d, want -2: helium counts up toward zero", e.Battery)
	}
	if len(e.Sounds) != 1 || e.Sounds[0] != HissSound {
		t.Errorf("sounds = %v, want HissSound", e.Sounds)
	}
}

// TestBatteryFizzlesAtZero: the frame that empties the counter plays the fizzle
// *instead* of the thrust, because the early return skips the sound throttle
// altogether. The thrust for that frame is still applied -- the last charge is not
// wasted, it just goes out quietly.
func TestBatteryFizzlesAtZero(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{Battery: 1}

	var in Input
	in.GetInput(g, e, Keys{Batt: true})

	if e.Battery != 0 {
		t.Fatalf("battery = %d, want 0", e.Battery)
	}
	if g.HVel != HyperThrust {
		t.Errorf("HVel = %d, want %d: the last charge still thrusts", g.HVel, HyperThrust)
	}
	if len(e.Sounds) != 1 || e.Sounds[0] != FizzleSound {
		t.Errorf("sounds = %v, want just FizzleSound", e.Sounds)
	}
}

// TestThrustSoundEveryFourthFrame pins the throttle that turns a held key into a
// pulsing engine note rather than one sample restarting thirty times a second.
//
// Nine held frames give three sounds, on frames 1, 5 and 9: the counter is reset on
// the first frame of a press so it always sounds immediately, then fires on frame 0 of
// each four-frame cycle.
//
// This is the one-player cadence, which is why one Input drives one glider here. The
// throttle state is shared in the original, so two players change it -- see the
// Input struct's own comments.
func TestThrustSoundEveryFourthFrame(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{Battery: 20}

	var in Input
	var soundedOn []int
	for i := 1; i <= 9; i++ {
		before := len(e.Sounds)
		in.GetInput(g, e, Keys{Batt: true})
		if len(e.Sounds) > before {
			soundedOn = append(soundedOn, i)
		}
	}
	want := []int{1, 5, 9}
	if len(soundedOn) != len(want) {
		t.Fatalf("sounded on frames %v, want %v", soundedOn, want)
	}
	for i := range want {
		if soundedOn[i] != want[i] {
			t.Fatalf("sounded on frames %v, want %v", soundedOn, want)
		}
	}
}

// TestFreshPressRestartsTheCycle is the reason batteryWasEngaged exists. Releasing the
// key for a single frame and pressing it again must sound at once, not wait out the
// remainder of the previous cycle -- otherwise a player tapping the battery in short
// bursts would hear nothing at all.
func TestFreshPressRestartsTheCycle(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{Battery: 20}

	var in Input
	in.GetInput(g, e, Keys{Batt: true}) // sounds, cycle now at 1
	in.GetInput(g, e, Keys{Batt: true}) // silent, cycle at 2
	if len(e.Sounds) != 1 {
		t.Fatalf("sounds = %v, want one so far", e.Sounds)
	}

	in.GetInput(g, e, Keys{})           // released: clears batteryWasEngaged
	in.GetInput(g, e, Keys{Batt: true}) // fresh press: must sound immediately
	if len(e.Sounds) != 2 {
		t.Errorf("sounds = %v, want a second one on the fresh press", e.Sounds)
	}
}

// TestBandDebounce pins the one-shot: FireHeld makes a held key fire once, and it is
// cleared in the else, so releasing for a single frame re-arms.
func TestBandDebounce(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{Bands: 3, BandAdded: true}

	var in Input
	in.GetInput(g, e, Keys{Band: true})
	if e.Bands != 2 || !g.FireHeld {
		t.Fatalf("bands = %d, FireHeld = %v; want 2, true", e.Bands, g.FireHeld)
	}
	in.GetInput(g, e, Keys{Band: true}) // still held: no second shot
	if e.Bands != 2 {
		t.Errorf("bands = %d after a held frame, want 2", e.Bands)
	}

	in.GetInput(g, e, Keys{}) // released
	if g.FireHeld {
		t.Fatal("FireHeld survived a released frame")
	}
	in.GetInput(g, e, Keys{Band: true})
	if e.Bands != 1 {
		t.Errorf("bands = %d after re-arming, want 1", e.Bands)
	}
}

// TestRefusedBandCostsNothing: AddBand returns false when the band array is full, and
// the whole block is inside that success test -- so the shot costs no band and FireHeld
// stays clear, meaning the key is still armed and the shot goes off the moment a slot
// frees up. A port that spent the band first would silently eat ammunition in a busy
// room.
func TestRefusedBandCostsNothing(t *testing.T) {
	g := newGliderAtRest(100, 50)
	e := &NopEnv{Bands: 3, BandAdded: false}

	var in Input
	in.GetInput(g, e, Keys{Band: true})

	if e.Bands != 3 {
		t.Errorf("bands = %d, want 3: a refused shot is free", e.Bands)
	}
	if g.FireHeld {
		t.Error("FireHeld set on a refused shot, so it will never retry")
	}
}

// TestDeleteAbandonsOnlyForPlayerOne pins the escape hatch for a stuck two-player
// game: when one player has left the room and the other cannot follow, Delete kills
// the straggler so the game can go on.
//
// Only player 1 may press it, and the reason is `thisGlider->which` rather than whose
// keyboard the key is on: GetInput runs for both gliders and both passes read the same
// key, so it is the identity test and nothing else that refuses player 2. That matters
// because it is the whole of the fix -- Input.Player2GiveUp removes the identity test and
// leaves every other guard in place (docs/IMPROVEMENTS.md 2.23).
//
// The last row is the one worth having: with the flag on, the key still does nothing
// unless somebody is actually waiting, so the correction cannot turn Delete into a suicide
// button. TestDeleteNeedsTheOtherPlayerGone is the same assertion for the unflagged path.
func TestDeleteAbandonsOnlyForPlayerOne(t *testing.T) {
	for _, tc := range []struct {
		name    string
		which   bool
		giveUp  bool
		escaped int16
		want    int
	}{
		{"player 1 may abandon", Player1, false, PlayerEscapedUp, 1},
		{"player 2 may not", Player2, false, PlayerEscapedUp, 0},
		{"player 1 with the fix on", Player1, true, PlayerEscapedUp, 1},
		{"player 2 with the fix on", Player2, true, PlayerEscapedUp, 1},
		{"the fix does not bypass the other guards", Player2, true, NoOneEscaped, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			room := newTwoPlayerRoom()
			room.escaped = tc.escaped // PlayerEscapedUp: the other player is already out

			g := newGliderAtRest(100, 50)
			g.Which = tc.which

			in := Input{Player2GiveUp: tc.giveUp}
			in.GetInput(g, room, Keys{Delete: true})

			if room.Kills != tc.want {
				t.Errorf("ForceKillGlider called %d times, want %d", room.Kills, tc.want)
			}
		})
	}
}

// TestDeleteNeedsTheOtherPlayerGone: the same key with nobody escaped does nothing,
// which is what stops it being a suicide button in normal play.
func TestDeleteNeedsTheOtherPlayerGone(t *testing.T) {
	room := newTwoPlayerRoom() // escaped == NoOneEscaped

	g := newGliderAtRest(100, 50)
	var in Input
	in.GetInput(g, room, Keys{Delete: true})

	if room.Kills != 0 {
		t.Errorf("ForceKillGlider called %d times, want 0", room.Kills)
	}
}
