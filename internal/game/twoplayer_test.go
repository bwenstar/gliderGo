package game

// Two gliders in one room, at the level where the room is real.
//
// The handshake's *rules* are pinned in the player package, against a hand-built room with
// four openings and no objects: player.TestTwoPlayerRaceForTheCeiling walks the three
// outcomes of raceForExit and player.TestManholeSkipsTheRaceInTwoPlayer records the one
// exit that has no race at all. Those tests cannot reach the other two thirds of the
// protocol, because a transporter needs an object graph, a link and a destination room,
// and the shared inventory needs a scoreboard to refresh.
//
// So this file is the half that needs a house. Three things live here:
//
//  1. **The transit strictness.** A wall, ceiling or floor asks the second glider only
//     "did your partner leave the same *way*". A transporter, mailbox or duct also asks
//     "did they leave through this same *object*" -- `activeRectEscaped == index`
//     (Interactions.c:1381-1411) -- and it refuses a mismatch *silently*, where geography
//     refuses one audibly and bounces the glider back. Transits are therefore the
//     stricter of the two, which is the opposite of what docs/PLAN.md's 1.9 acceptance
//     used to say; the entry has been corrected.
//
//  2. **The arrival freeze.** ReadyGliderFromTransit idles whichever glider is not
//     FirstPlayer for thirty frames, and FirstPlayer means "the one who went into limbo
//     first" only after somebody has -- before that it is Go's and the C's zero value,
//     which is *player 2*.
//
//  3. **One inventory, one throttle.** Two gliders draw on one signed battery counter and
//     one player.Input, and the second is observable through the sound: a two-player game
//     with a single thruster plays the thrust sound every frame instead of every fourth.
//
// None of it can be reached by resuming a saved game, because NewGame's two-player arm
// passes NewGameMode to both InitGlider calls (play.go:145-150) -- a two-player game
// always begins at the house's authored start point and can never be resumed. So the
// fixtures below compose a room and place both gliders by hand.

import (
	"testing"

	"glidergo/internal/game/player"
)

// ---------------------------------------------------------------------------
// The fixture
// ---------------------------------------------------------------------------

// twoPlayerRoom is a two-player game standing in one named room of one named house, with
// every sound it plays recorded.
//
// It deliberately does not call NewGame. NewGame would put both gliders on the house's
// start point mid-fade and would run the banner and the stars panel; what these tests
// want is two gliders in a specific room, in normal mode, wherever the test puts them.
// InitGlider is called for each because it is what sets the sixteen fields a glider needs
// before anything may read it -- and because its two-player arm is what makes Mortals 4.
func twoPlayerRoom(t *testing.T, houseName string, room int16) (*World, *[]int16) {
	t.Helper()

	w := playTestWorld(t, houseName, 0)
	w.TwoPlayer = true

	var heard []int16
	w.SoundPlayer = func(sound, priority int16) { heard = append(heard, sound) }

	w.ForceThisRoom(room)
	w.SetObjectsToDefaults()
	w.Rebuild()

	// Identities first: InitGlider does not set Which, and an unset Which makes both
	// gliders Player2 -- which would silently disable the whole handshake, since
	// FirstPlayer would then match both of them.
	w.P1.Which = player.Player1
	w.P2.Which = player.Player2
	w.InitGlider(&w.P1, NewGameMode)
	w.InitGlider(&w.P2, NewGameMode)

	heard = heard[:0]
	return w, &heard
}

// putGliderIn places a glider fully inside a hot spot's bounds, in normal mode.
//
// "Fully inside" is the requirement GliderInRect imposes and SectGlider does not: the
// event arms of the collision dispatcher re-test with containment, so a glider merely
// overlapping a transporter is registered and then refused. Centring is the cheapest way
// to be certain, and the helper fails the test rather than silently placing a glider that
// cannot fit -- a transporter narrower than 48 pixels would otherwise look like a rule.
func putGliderIn(t *testing.T, g *player.Glider, r Rect) {
	t.Helper()

	if r.Right-r.Left < player.GliderWide || r.Bottom-r.Top < player.GliderHigh {
		t.Fatalf("hot spot %v is smaller than a %dx%d glider, so GliderInRect can never "+
			"be true inside it: pick another room", r, player.GliderWide, player.GliderHigh)
	}
	left := r.Left + (r.Right-r.Left-player.GliderWide)/2
	top := r.Top + (r.Bottom-r.Top-player.GliderHigh)/2

	g.Dest = player.Rect{Top: top, Left: left,
		Bottom: top + player.GliderHigh, Right: left + player.GliderWide}
	g.DestShadow = player.Rect{Top: player.ShadowTop, Left: left,
		Bottom: player.ShadowTop + player.ShadowHigh, Right: left + player.GliderWide}
	g.Whole = g.Dest
	g.WholeShadow = g.DestShadow
	g.Mode = player.GliderNormal
	g.DontDraw = false
}

// twoTransporters finds two live, non-overlapping transporters in the current room.
//
// It searches rather than hard-coding a pair of indices because the indices are an output
// of CreateActiveRects, not of the house file: they shift the moment the object list or
// the rect builder changes, and a test that hard-coded them would fail with "mode 0, want
// 5" rather than "this room no longer has two transporters".
func twoTransporters(t *testing.T, w *World) (a, b int16) {
	t.Helper()

	var found []int16
	for i := range w.R.Hot {
		if w.R.Hot[i].Action == TransportIt && w.R.Hot[i].IsOn {
			found = append(found, int16(i))
		}
	}
	for i := 0; i < len(found); i++ {
		for j := i + 1; j < len(found); j++ {
			ra, rb := w.R.Hot[found[i]].Bounds, w.R.Hot[found[j]].Bounds
			// Disjoint on either axis is enough, and it has to be checked: two
			// overlapping transporters would let one glider be inside both, which would
			// make "the same object" untestable.
			if ra.Right <= rb.Left || rb.Right <= ra.Left ||
				ra.Bottom <= rb.Top || rb.Bottom <= ra.Top {
				return found[i], found[j]
			}
		}
	}
	t.Fatalf("room %d has %d live transporters and no two of them are disjoint; "+
		"the transit-race test needs two the same glider cannot be inside at once",
		w.R.RoomNumber, len(found))
	return 0, 0
}

// dissolve runs one glider's transit animation to completion.
//
// The bound is a real assertion, not a safety net: TransportGliderOut counts Frame down
// from LastFadeSequence-1, so the dissolve is exactly sixteen frames and a loop that
// needed more than twice that has found something other than a dissolve.
func dissolve(t *testing.T, w *World, g *player.Glider) {
	t.Helper()

	for i := 0; i < 2*int(player.LastFadeSequence); i++ {
		if g.Mode != player.GliderTransporting {
			return
		}
		w.HandleGlider(g)
	}
	t.Fatalf("glider is still GliderTransporting after %d frames; the dissolve is "+
		"%d frames long", 2*player.LastFadeSequence, player.LastFadeSequence)
}

// ---------------------------------------------------------------------------
// 1. The transit race: same code *and* same object, refused silently
// ---------------------------------------------------------------------------

// TestTransitRaceNeedsTheSameObjectNotJustTheSameKind is the strictness a wall does not
// have.
//
// The room is "Every Teddy's a Transport!", chosen for two invisible transporters far
// enough apart that no glider can be inside both. Player one dissolves into one of them
// and waits in limbo; player two then walks into the *other* one, which matches the
// escape code and not the object, and nothing happens at all -- no mode change, no sound,
// no room change, and not even a write to the pending link. Walking into the first one
// instead is accepted immediately.
//
// The contrast with player.TestTwoPlayerRaceForTheCeiling's third case is the whole point
// of the test. A ceiling refuses with kDontExitSound, reverses the glider's velocity and
// claws back the overshoot, so the player is *told*. A transporter refuses by falling off
// the end of an `else if`, so the player is told nothing and the glider simply stands in
// the transporter. That asymmetry is docs/IMPROVEMENTS.md 2.22's first bullet, and this
// test is the demonstration of it.
func TestTransitRaceNeedsTheSameObjectNotJustTheSameKind(t *testing.T) {
	w, heard := twoPlayerRoom(t, "CD Demo House", 143)
	a, b := twoTransporters(t, w)

	// -- player one arms the race -------------------------------------------
	putGliderIn(t, &w.P1, w.R.Hot[a].Bounds)
	w.HandleHotSpotCollision(&w.P1, &w.R.Hot[a], a)

	if w.P1.Mode != player.GliderTransporting {
		t.Fatalf("P1 Mode %d after entering a transporter, want GliderTransporting %d",
			w.P1.Mode, player.GliderTransporting)
	}
	if w.ActiveRectEscaped != a {
		t.Fatalf("ActiveRectEscaped %d, want %d: the arming glider records which "+
			"transporter, and the follower has to match it", w.ActiveRectEscaped, a)
	}
	// Escaped is *not* set yet, and the gap is real: the code is written when the
	// dissolve finishes, so for the sixteen frames in between both gliders can still
	// take the arming branch.
	if w.Escaped != NoOneEscaped {
		t.Errorf("Escaped %d on the frame the transporter was entered, want NoOneEscaped "+
			"%d: escapeOrWait writes the code when the dissolve ends, not when it starts",
			w.Escaped, NoOneEscaped)
	}

	dissolve(t, w, &w.P1)

	if w.Escaped != player.PlayerTransportedOut {
		t.Fatalf("Escaped %d after the dissolve, want PlayerTransportedOut %d",
			w.Escaped, player.PlayerTransportedOut)
	}
	if w.P1.Mode != player.GliderInLimbo {
		t.Fatalf("P1 Mode %d, want GliderInLimbo %d: the first one out waits",
			w.P1.Mode, player.GliderInLimbo)
	}
	if w.FirstPlayer != player.Player1 {
		t.Errorf("FirstPlayer %v, want %v: FlagGliderInLimbo stamps whoever went first",
			w.FirstPlayer, player.Player1)
	}
	if w.R.RoomNumber != 143 {
		t.Fatalf("the room changed to %d while one glider was still in it", w.R.RoomNumber)
	}
	pendingAfterArming := w.Pending

	// -- player two walks into the wrong transporter ------------------------
	putGliderIn(t, &w.P2, w.R.Hot[b].Bounds)
	*heard = (*heard)[:0]
	w.HandleHotSpotCollision(&w.P2, &w.R.Hot[b], b)

	if w.P2.Mode != player.GliderNormal {
		t.Errorf("P2 Mode %d after the wrong transporter, want GliderNormal %d: "+
			"activeRectEscaped is %d and the index was %d, so the follow branch must "+
			"not fire", w.P2.Mode, player.GliderNormal, w.ActiveRectEscaped, b)
	}
	if len(*heard) != 0 {
		t.Errorf("the refusal played %v; a transit mismatch is silent, unlike a wall's "+
			"kDontExitSound -- see player.TestTwoPlayerRaceForTheCeiling", *heard)
	}
	if w.P2.HVel != 0 || w.P2.VVel != 0 {
		t.Errorf("P2 velocity (%d,%d) after the refusal, want (0,0): a transit refusal "+
			"does not bounce the glider the way BounceGlider does",
			w.P2.HVel, w.P2.VVel)
	}
	if w.ActiveRectEscaped != a || w.Escaped != player.PlayerTransportedOut {
		t.Errorf("the refusal moved the race to (%d,%d), want (%d,%d) left alone",
			w.Escaped, w.ActiveRectEscaped, player.PlayerTransportedOut, a)
	}
	if w.Pending != pendingAfterArming {
		t.Errorf("the refusal overwrote the pending link: %+v, want %+v. "+
			"resolveTransitLink is evaluated as StartGliderTransporting's argument, so a "+
			"refused follower must not reach it", w.Pending, pendingAfterArming)
	}

	// This is the deadlock in docs/IMPROVEMENTS.md 2.22: one glider in limbo, the other
	// standing in a transporter that will never take it, and no frame counter anywhere
	// that ends it. Nothing below the refusal changes on its own, so repeating it is the
	// proof rather than a re-test.
	for i := 0; i < 60; i++ {
		w.HandleHotSpotCollision(&w.P2, &w.R.Hot[b], b)
	}
	if w.P2.Mode != player.GliderNormal || w.P1.Mode != player.GliderInLimbo {
		t.Errorf("after sixty more frames the modes are %d/%d; the deadlock is supposed "+
			"to be permanent, and the only way out is the give-up key",
			w.P1.Mode, w.P2.Mode)
	}

	// -- and into the right one ---------------------------------------------
	putGliderIn(t, &w.P2, w.R.Hot[a].Bounds)
	*heard = (*heard)[:0]
	w.HandleHotSpotCollision(&w.P2, &w.R.Hot[a], a)

	if w.P2.Mode != player.GliderTransporting {
		t.Fatalf("P2 Mode %d in the *same* transporter, want GliderTransporting %d",
			w.P2.Mode, player.GliderTransporting)
	}
	if len(*heard) == 0 || (*heard)[0] != player.TransOutSound {
		t.Errorf("sounds %v, want TransOutSound %d first: the acceptance is audible even "+
			"though the refusal is not", *heard, player.TransOutSound)
	}
}

// TestTheGiveUpKeyIsTheOnlyWayOutOfTheDeadlock is the payoff of the state the test above
// leaves behind, and the reason 1.9 owes player 2 a second binding.
//
// One glider waits in limbo at a transporter, the other stands in a transporter that will
// never accept it, and nothing in the game times either of them out. ForceKillGlider is the
// only exit: it spends the free glider's mortal, and OffAMortal then sees Suicide and calls
// FollowTheLeader, which performs the room change the waiting glider had been holding. So
// the deadlock costs a life and the game goes on.
//
// In the original that key is player 1's alone. The last case is the correction --
// Fixes.Player2GiveUp -- and it is here rather than only in
// player.TestDeleteAbandonsOnlyForPlayerOne because the unit test can prove the call
// happens and only a real house can prove what the call is *for*.
func TestTheGiveUpKeyIsTheOnlyWayOutOfTheDeadlock(t *testing.T) {
	for _, c := range []struct {
		name    string
		presser bool
		fix     bool
		rescued bool
	}{
		{"player one presses it", player.Player1, false, true},
		{"player two presses it", player.Player2, false, false},
		{"player two presses it with the fix on", player.Player2, true, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, _ := twoPlayerRoom(t, "CD Demo House", 143)
			w.Fix.Player2GiveUp = c.fix
			a, b := twoTransporters(t, w)

			// The deadlock, built exactly as the test above builds it and asserted there
			// rather than here: player one waiting at transporter `a`, player two standing
			// in `b` and being refused in silence.
			putGliderIn(t, &w.P1, w.R.Hot[a].Bounds)
			w.HandleHotSpotCollision(&w.P1, &w.R.Hot[a], a)
			dissolve(t, w, &w.P1)
			putGliderIn(t, &w.P2, w.R.Hot[b].Bounds)
			w.HandleHotSpotCollision(&w.P2, &w.R.Hot[b], b)

			if w.P1.Mode != player.GliderInLimbo || w.P2.Mode != player.GliderNormal {
				t.Fatalf("the fixture did not reach the deadlock: modes %d/%d",
					w.P1.Mode, w.P2.Mode)
			}
			destination := w.Pending.Room
			presser := &w.P1
			if c.presser == player.Player2 {
				presser = &w.P2
			}

			w.KeyPoll = func(g *player.Glider) player.Keys {
				return player.Keys{Delete: g.Which == c.presser}
			}
			w.GetInput(&w.P1)
			w.GetInput(&w.P2)

			if !c.rescued {
				if w.Suicide || w.P2.Mode == player.GliderFadingOut {
					t.Fatalf("the press was accepted: Suicide=%v P2 mode %d. Only "+
						"player one has the key unless Fixes.Player2GiveUp is set",
						w.Suicide, w.P2.Mode)
				}
				if w.Mortals != 2*InitialGliders {
					t.Errorf("Mortals %d, want %d: a refused press must not cost a life",
						w.Mortals, 2*InitialGliders)
				}
				return
			}

			// The straggler is killed, not the presser: ForceKillGlider tests one glider
			// for limbo and fades out the *other*, so pressing it is asking for your
			// partner's life and not your own.
			if w.P2.Mode != player.GliderFadingOut {
				t.Fatalf("P2 mode %d after %v pressed Delete, want GliderFadingOut %d",
					w.P2.Mode, presser.Which, player.GliderFadingOut)
			}
			if !w.Suicide {
				t.Error("Suicide is not set, so OffAMortal will respawn the glider " +
					"instead of calling FollowTheLeader")
			}

			// Run the fade-out out. The last frame of it reaches OffAMortal, which spends
			// the mortal and dispatches the room change.
			for i := 0; i < 2*int(player.LastFadeSequence); i++ {
				if w.P2.Mode != player.GliderFadingOut {
					break
				}
				w.HandleGlider(&w.P2)
			}

			if w.Mortals != 2*InitialGliders-1 {
				t.Errorf("Mortals %d, want %d: breaking the deadlock costs exactly one life",
					w.Mortals, 2*InitialGliders-1)
			}
			if destination != 143 && w.R.RoomNumber != destination {
				t.Errorf("room %d, want %d: FollowTheLeader owes the waiting glider the "+
					"room change it was holding", w.R.RoomNumber, destination)
			}
			if w.Escaped != NoOneEscaped {
				t.Errorf("Escaped %d, want NoOneEscaped %d: FollowTheLeader clears the "+
					"slot before it dispatches", w.Escaped, NoOneEscaped)
			}
			if w.P1.Mode == player.GliderInLimbo {
				t.Error("player one is still waiting after the give-up key was pressed")
			}
		})
	}
}

// TestTransitFollowerCompletesTheRoomChange is the other half: the follow branch does not
// merely change a mode, it hands both gliders to TransportRoomToRoom.
//
// Split from the test above because it runs the whole transition -- ForceThisRoom,
// ReadyLevel, a wipe and two RenderFrames -- and a failure in any of that would otherwise
// be reported as a failure of the race.
func TestTransitFollowerCompletesTheRoomChange(t *testing.T) {
	w, _ := twoPlayerRoom(t, "CD Demo House", 143)
	a, _ := twoTransporters(t, w)

	putGliderIn(t, &w.P1, w.R.Hot[a].Bounds)
	w.HandleHotSpotCollision(&w.P1, &w.R.Hot[a], a)
	dissolve(t, w, &w.P1)

	destination := w.Pending.Room
	if destination == 143 {
		t.Skip("this transporter links back into its own room, so there is no room " +
			"change to observe")
	}

	putGliderIn(t, &w.P2, w.R.Hot[a].Bounds)
	w.HandleHotSpotCollision(&w.P2, &w.R.Hot[a], a)
	dissolve(t, w, &w.P2)

	if w.R.RoomNumber != destination {
		t.Errorf("room %d after both gliders transported, want %d",
			w.R.RoomNumber, destination)
	}
	// The slot is cleared by escapeOrWait's matching arm, which is what lets the next
	// room be raced for. A code left behind is the takingTheStairs-style leak.
	if w.Escaped != NoOneEscaped {
		t.Errorf("Escaped %d after the pair left, want NoOneEscaped %d",
			w.Escaped, NoOneEscaped)
	}
	// Neither glider may still be in limbo: TransportRoomToRoom calls UndoGliderLimbo on
	// both before it readies either.
	if w.P1.Mode == player.GliderInLimbo || w.P2.Mode == player.GliderInLimbo {
		t.Errorf("modes %d/%d after arrival; UndoGliderLimbo runs for both gliders",
			w.P1.Mode, w.P2.Mode)
	}
}

// ---------------------------------------------------------------------------
// 2. The arrival freeze
// ---------------------------------------------------------------------------

// TestTransitArrivalFreezesWhoeverIsNotFirstPlayer pins ReadyGliderFromTransit's last
// three lines, and the surprise in them is the default.
//
// The test is a table over FirstPlayer because the field's *interesting* value is the one
// nobody assigns. `firstPlayer` is a BSS Boolean in the C, so it is false -- kPlayer2 --
// until some glider goes into limbo, and the freeze is `which != firstPlayer`. So on the
// first transit of a game where nobody has waited for anybody, it is **player one** who
// arrives frozen. That is the opposite of what the name suggests and it is reachable: a
// two-player game whose first room change is a manhole (no race, no limbo, see
// player.TestManholeSkipsTheRaceInTwoPlayer) and whose second is a transporter gets it.
func TestTransitArrivalFreezesWhoeverIsNotFirstPlayer(t *testing.T) {
	for _, c := range []struct {
		name        string
		firstPlayer bool
		why         string
	}{
		{"nobody has waited yet", player.Player2,
			"FirstPlayer's zero value is Player2, so player one is the one frozen"},
		{"player one waited", player.Player1,
			"the follower is frozen, which is the case a real handshake produces"},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, _ := twoPlayerRoom(t, "CD Demo House", 143)
			w.FirstPlayer = c.firstPlayer

			frozen, free := &w.P1, &w.P2
			if c.firstPlayer == player.Player1 {
				frozen, free = &w.P2, &w.P1
			}

			// A bare LinkedToOther arrival: no link rect is needed, because the freeze is
			// past the switch and applies to every kind of transit.
			w.ReadyGliderFromTransit(&w.P1, player.LinkedToOther)
			w.ReadyGliderFromTransit(&w.P2, player.LinkedToOther)

			if frozen.Mode != player.GliderIdle {
				t.Errorf("Mode %d, want GliderIdle %d (%s)",
					frozen.Mode, player.GliderIdle, c.why)
			}
			// The countdown lives in HVel, which is why a frozen glider cannot also be
			// moving: TagGliderIdle overwrites the velocity with the frame count.
			if frozen.HVel != player.IdleFrames {
				t.Errorf("HVel %d, want IdleFrames %d: the freeze counter shares the "+
					"horizontal velocity field (Modes.c:638)",
					frozen.HVel, player.IdleFrames)
			}
			if free.Mode == player.GliderIdle {
				t.Errorf("both gliders were frozen; only `which != FirstPlayer` is (%s)",
					c.why)
			}
		})
	}
}

// TestOnePlayerTransitDoesNotFreezeAnybody is the guard the test above needs.
//
// The freeze is gated on TwoPlayer, and a port that dropped the gate would idle the glider
// for a second after every transporter in the game everybody actually plays -- which would
// look like a stutter and would pass any test that only asserted the two-player case.
func TestOnePlayerTransitDoesNotFreezeAnybody(t *testing.T) {
	w, _ := twoPlayerRoom(t, "CD Demo House", 143)
	w.TwoPlayer = false
	w.FirstPlayer = player.Player2 // the value that freezes player one when it counts

	w.ReadyGliderFromTransit(&w.P1, player.LinkedToOther)

	if w.P1.Mode == player.GliderIdle {
		t.Errorf("a one-player arrival was frozen; the freeze exists only to stop two " +
			"gliders flying out of the same pixel")
	}
}

// ---------------------------------------------------------------------------
// 3. One inventory and one throttle
// ---------------------------------------------------------------------------

// pressBatt drives n frames of input with the battery key held by whoever `p1` and `p2`
// say, in PlayGame's order: player one's GetInput, then player two's, back to back
// (Play.c:452-453).
//
// The order is the whole reason this is a helper. player.Input's throttle state is shared,
// so what the second call sees depends on what the first one did -- and a test that polled
// them in the other order, or only polled the thrusting glider, would measure a game that
// does not exist.
func pressBatt(w *World, n int, p1, p2 bool) {
	w.KeyPoll = func(g *player.Glider) player.Keys {
		if g.Which == player.Player1 {
			return player.Keys{Batt: p1}
		}
		return player.Keys{Batt: p2}
	}
	for i := 0; i < n; i++ {
		w.GetInput(&w.P1)
		if w.TwoPlayer {
			w.GetInput(&w.P2)
		}
	}
}

// TestOneBatteryServesBothGliders is the shared inventory from the side a player notices:
// what one glider spends, the other does not have.
//
// The counter is doubled at pickup (TestTwoPlayerDoublesTheFiveSupplies) precisely because
// it is shared -- two gliders drawing on one number need twice as much in it to be worth
// the same each. Nothing rations it, though, so one player holding the key drains both
// players' charges, and this is the test that says so.
func TestOneBatteryServesBothGliders(t *testing.T) {
	// Neither glider is placed anywhere in particular: the battery cares only that the
	// mode is GliderNormal, which is what InitGlider leaves behind.
	w, heard := twoPlayerRoom(t, "CD Demo House", 143)

	const charges = 3
	w.Battery = charges

	pressBatt(w, charges, true, false)

	if w.Battery != 0 {
		t.Fatalf("Battery %d after player one held the key for %d frames, want 0",
			w.Battery, charges)
	}
	if want := int16(charges * player.HyperThrust); w.P1.HVel != want {
		t.Errorf("P1 HVel %d, want %d: %d frames of HyperThrust",
			w.P1.HVel, want, charges)
	}
	// The fizzle is the audible half of running dry, and it is played once.
	fizzles := 0
	for _, s := range *heard {
		if s == player.FizzleSound {
			fizzles++
		}
	}
	if fizzles != 1 {
		t.Errorf("heard %d fizzles in %v, want exactly 1", fizzles, *heard)
	}

	// Now player two tries. There is nothing left, and the emptiness was not player
	// two's doing.
	before := w.P2.HVel
	*heard = (*heard)[:0]
	pressBatt(w, 4, false, true)

	if w.P2.HVel != before {
		t.Errorf("P2 HVel %d -> %d on an empty counter, want no change: `batteryTotal != 0` "+
			"is the gate and player one already spent it", before, w.P2.HVel)
	}
	if len(*heard) != 0 {
		t.Errorf("an empty battery still played %v", *heard)
	}
	if w.Battery != 0 {
		t.Errorf("Battery %d, want 0: a refused press must not decrement", w.Battery)
	}
}

// TestOneThrusterInTwoPlayerSoundsEveryFrame is the sharpest observable consequence of
// two gliders sharing one player.Input, and it is a defect rather than a design.
//
// batteryFrame counts 0..3 and re-triggers the thrust sound on 0, which is what makes a
// held key pulse instead of machine-gunning. batteryWasEngaged is what resets the counter
// on a fresh press -- and it is cleared by *any* GetInput call that did not take the
// battery branch, including the other player's. So in a two-player game with one thruster,
// the non-thrusting glider's poll clears the flag every frame, the counter is forced back
// to 0 every frame, and the sound plays every frame: four times the intended rate, from a
// player who is not touching the key.
//
// With both players thrusting the counter advances twice per frame instead, so the sound
// arrives every second frame and the charges drain at two per frame. All three rates are
// in one table because it is the *relationship* between them that identifies the bug: any
// one of them alone looks like a tuning choice.
func TestOneThrusterInTwoPlayerSoundsEveryFrame(t *testing.T) {
	const frames = 8

	for _, c := range []struct {
		name    string
		two     bool
		p1, p2  bool
		thrusts int
		spent   int16
		why     string
	}{
		{"one player", false, true, false, frames / int(player.BatteryFrames), frames,
			"the intended cadence: one sound every fourth frame"},
		{"two players, one thrusting", true, true, false, frames, frames,
			"the other player's poll clears batteryWasEngaged, so the cycle restarts " +
				"every frame and the sound plays every frame"},
		{"two players, both thrusting", true, true, true, frames / 2, 2 * frames,
			"the counter advances twice per frame, so the sound halves and the drain " +
				"doubles"},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, heard := twoPlayerRoom(t, "CD Demo House", 143)
			w.TwoPlayer = c.two

			// Comfortably more than the run can spend, so no arm reaches the fizzle and
			// every sound in the list is a thrust.
			w.Battery = 1000

			pressBatt(w, frames, c.p1, c.p2)

			thrusts := 0
			for _, s := range *heard {
				if s == player.ThrustSound {
					thrusts++
				}
			}
			if thrusts != c.thrusts {
				t.Errorf("%d thrust sounds in %d frames, want %d (%s)",
					thrusts, frames, c.thrusts, c.why)
			}
			if spent := 1000 - w.Battery; spent != c.spent {
				t.Errorf("spent %d charges in %d frames, want %d",
					spent, frames, c.spent)
			}
		})
	}
}
