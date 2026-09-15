package game

// The reward path: HandleRewards' fifteen prize types, and the saved-map restore ten of them
// depend on.
//
// The table below has a row per *case label* rather than per arm -- fifteen rows over the
// fourteen arms, because kGreaseRt and kGreaseLf share a body and a test still has to prove
// both directions reach it. Where the count matters, rewards.go's header keeps the two apart.
//
// The subject here is a *mapping*, not an algorithm. Fifteen rows, each of which touches
// some subset of {a sound, the score, one of four counters, the star count, a puff of light,
// a flying numeral, the glider's velocity, the room's background} and leaves the rest alone.
// There is no clever code to get wrong; what gets got wrong is an arm that awards the wrong
// counter, halves where it should quarter, or -- as happened in this port -- is simply
// missing.
//
// So the shape of this file is one table with one row per arm and every column of that
// subset asserted on every row, including the columns the row does not use. A test that
// only checked the effects an arm *has* would pass on an arm that also did something extra;
// checking the zeroes is what makes the table a specification. TestFifteenRewardArms is
// that table and it is the file's centre of gravity.
//
// Everything after it tests something the table cannot: the sign relationship between the
// battery and the helium balloon, the two-player doubling and what it excludes, the order
// dependency in the invisible bonus's point read, and the four behaviours of
// RestoreFromSavedMap -- which has no caller outside this path and the switch path.
//
// TestEveryDispatchableRewardIsConsumed is the one to read first if this file ever fails.
// It derives its list of prize types from CreateActiveRects rather than from a literal, so
// it fails when the two halves of the reward path disagree about which types exist -- which
// is exactly how the missing kHelium arm was found.

import (
	"testing"

	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/render"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// prizeAt is the room-local rect every prize in this file occupies: 48x48 at (100, 100).
//
// Room-local, because that is what a hot spot's Bounds is -- CreateActiveRects offsets by
// the object's topLeft and nothing else (hotspots.go:303). Both AddSparkle and
// AddFlyingPoint add the play origin themselves, which is why the reward arms can pass
// `who.Bounds` straight through.
var prizeAt = render.SetRect(100, 100, 148, 148)

// The glider's velocity on entry to every reward. 8 and 16 are chosen so that both the
// halving arms and the quartering arms land on distinct exact integers -- 4/8 and 2/4 -- so
// a row that halves where it should quarter cannot pass by rounding.
const (
	startHVel int16 = 8
	startVVel int16 = 16
)

// startStars is what StarsLeft holds before a star is collected. Three, so that the arm's
// `<= 0` branch is not taken: TestLastStarFlagsGameOver covers the other side.
const startStars int16 = 3

// rewardWorld builds the smallest world HandleRewards will walk: one room, one prize in
// object slot 0, one master entry pointing at it, one live kRewardIt hot spot, and one
// saved map underneath so the restore has something to paint.
//
// The master entry and the house object are separate copies of the same object, which is
// not a convenience -- it is the arrangement SetObjectState's reproduced bug needs. It
// clears the house's state byte and writes the *master* copy through the wrong union
// member, and a fixture that shared one object would hide that.
//
// `points` is only read by the kInvisBonus arm; every other row ignores it.
func rewardWorld(t *testing.T, what, points int16) (*World, *player.Glider, *HotObject) {
	t.Helper()
	w := dynaWorld(t)

	obj := house.Object{What: what}
	obj.SetBonus(house.Bonus{
		TopLeft: house.Point{V: prizeAt.Top, H: prizeAt.Left},
		Length:  48,
		Points:  points,
		State:   1,
		Initial: 1,
	})
	w.Room(0).Objects[0] = obj

	w.R.RoomNumber = 0
	w.R.Master = []MasterObject{{
		RoomNum: 0, ObjectNum: 0,
		RoomLink: -1, ObjectLink: -1, LocalLink: -1,
		HotNum: 0, DynaNum: -1,
		TheObject: obj,
	}}
	w.R.Hot = []HotObject{{Bounds: prizeAt, Action: RewardIt, Who: 0, IsOn: true}}
	w.R.SavedMaps = []render.SavedMap{savedMapUnder(0, 0, prizeAt, w.R.V.OriginH, w.R.V.OriginV)}

	w.StarsLeft = startStars

	g := &w.P1
	g.Mode = player.GliderNormal
	g.HVel, g.VVel = startHVel, startVVel

	clearRects(w)
	return w, g, &w.R.Hot[0]
}

// savedMapUnder builds the slot DrawARoomsObjects would have claimed for an object at
// `local` (room-local) whose room is `where` and object slot `who`.
//
// Dest is in *screen* coordinates, because that is what backUpToSavedMap recorded: it is
// handed the rect after the composition has already offset it by the play origin. Getting
// that wrong would put every restore one play origin away from the prize, which is a
// mistake a test with a zero origin could not catch -- see dynaWorld on why the fixture
// origin is non-zero on both axes.
func savedMapUnder(where, who int16, local Rect, originH, originV int16) render.SavedMap {
	return render.SavedMap{
		Map:   render.NewSurface(int(local.Wide()), int(local.Tall())),
		Dest:  render.Offset(local, originH, originV),
		Where: where,
		Who:   who,
	}
}

// ---------------------------------------------------------------------------
// The fifteen prize types
// ---------------------------------------------------------------------------

// rewardCase is one case label of HandleRewards, and every field is asserted on every row --
// including the zeroes. See the file comment on why the zeroes are the point.
//
// The four counters are absolute rather than deltas because all four start at zero, and
// absolute values read against the constants in rewards.go without arithmetic.
type rewardCase struct {
	name string
	what int16

	// sound is the one PlayPrioritySound the arm makes, or 0 for the four silent arms.
	// One, never two: the restore only makes a sound on the switch path.
	sound int16

	score   int32
	mortals int16
	battery int16
	bands   int16
	foil    int16
	// stars is StarsLeft afterwards. startStars on every row but the star's.
	stars int16

	sparkles int
	flying   int
	// art is FlyingPoints[0].Start -- which three cels of the 24x120 strip the numeral
	// comes from, and therefore which number the player sees. Asserted because it is
	// duplicated from the point value by hand at every call site.
	art int16

	// The glider's velocity afterwards: quartered, halved, or untouched.
	hVel, vVel int16

	// workRects is 1 for the ten arms that restore a saved map and 0 for the five that
	// do not. It is the only observable RestoreFromSavedMap leaves in a world with no
	// art, and it doubles as the "did this arm erase the prize" column.
	workRects int

	// cleared is whether the *house* state byte went to zero, i.e. whether the arm
	// reached SetObjectState at all. False for the two bare breaks.
	cleared bool
	// isOn is who.IsOn afterwards. True for the two bare breaks, which is the bug.
	isOn bool

	note string
}

// theFifteenRewards is Interactions.c:764-980 in the original's order.
//
// The order is worth keeping. It is not alphabetical and not grouped by effect; it is the
// order the object codes were assigned in, which puts the three clocks together, the
// supplies together, and -- the detail this table exists to pin -- kHelium between kSparkle
// and kSlider, twenty lines after the kBattery arm it is the mirror image of. That distance
// is why the arm went missing in transcription.
var theFifteenRewards = []rewardCase{{
	name: "RedClock", what: RedClock,
	sound: BeepsSound, score: 100, stars: startStars,
	flying: 1, art: 12, hVel: 2, vVel: 4, workRects: 1, cleared: true,
	note: "quarters the velocity, like all four clocks",
}, {
	name: "BlueClock", what: BlueClock,
	sound: BuzzerSound, score: 300, stars: startStars,
	flying: 1, art: 6, hVel: 2, vVel: 4, workRects: 1, cleared: true,
}, {
	name: "YellowClock", what: YellowClock,
	sound: DingSound, score: 500, stars: startStars,
	flying: 1, art: 3, hVel: 2, vVel: 4, workRects: 1, cleared: true,
}, {
	name: "Cuckoo", what: Cuckoo,
	sound: CuckooSound, score: 1000, stars: startStars,
	flying: 1, art: 0, hVel: 2, vVel: 4, workRects: 1, cleared: true,
	note: "art 0 comes from AddFlyingPoint's default arm, not a 1000 case",
}, {
	name: "Paper", what: Paper,
	sound: EnergizeSound, mortals: 1, stars: startStars,
	sparkles: 1, hVel: 4, vVel: 8, workRects: 1, cleared: true,
	note: "the first supply arm: a sparkle instead of numerals, and halves rather than quarters",
}, {
	name: "Battery", what: Battery,
	sound: EnergizeSound, battery: BatterySupply, stars: startStars,
	sparkles: 1, hVel: 4, vVel: 8, workRects: 1, cleared: true,
}, {
	name: "Bands", what: Bands,
	sound: EnergizeSound, bands: BandsSupply, stars: startStars,
	sparkles: 1, hVel: 4, vVel: 8, workRects: 1, cleared: true,
}, {
	name: "GreaseRt", what: GreaseRt,
	stars: startStars, hVel: startHVel, vVel: startVVel, cleared: true,
	note: "no sound, no restore, no velocity change: a jar is knocked over, not collected",
}, {
	name: "GreaseLf", what: GreaseLf,
	stars: startStars, hVel: startHVel, vVel: startVVel, cleared: true,
}, {
	name: "Foil", what: Foil,
	sound: EnergizeSound, foil: FoilSupply, stars: startStars,
	sparkles: 1, hVel: 4, vVel: 8, workRects: 1, cleared: true,
	note: "the one supply with no scoreboard counter; the glider's own art changes instead",
}, {
	name: "InvisBonus", what: InvisBonus,
	sound: BonusSound, score: 250, stars: startStars,
	flying: 1, art: 9, hVel: 2, vVel: 4, cleared: true,
	note: "workRects 0: nothing was ever drawn over the background, so nothing is restored",
}, {
	name: "Star", what: Star,
	sound: EnergizeSound, score: StarPoints, stars: startStars - 1,
	sparkles: 1, hVel: startHVel, vVel: startVVel, workRects: 1, cleared: true,
	note: "the only prize that does not slow the glider down",
}, {
	name: "Sparkle", what: Sparkle,
	stars: startStars, hVel: startHVel, vVel: startVVel, isOn: true,
	note: "a bare break: nothing happens and IsOn is never cleared, so it re-dispatches forever",
}, {
	name: "Helium", what: Helium,
	sound: EnergizeSound, battery: -HeliumSupply, stars: startStars,
	sparkles: 1, hVel: 4, vVel: 8, workRects: 1, cleared: true,
	note: "the battery arm with every sign inverted; the counter is one field for both",
}, {
	name: "Slider", what: Slider,
	stars: startStars, hVel: startHVel, vVel: startVVel, isOn: true,
	note: "the other bare break",
}}

// TestFifteenRewardArms is the table: one collection of each prize type, with every
// observable effect and every absence of one asserted.
func TestFifteenRewardArms(t *testing.T) {
	if len(theFifteenRewards) != 15 {
		t.Fatalf("%d rows, want 15: the C's switch has fifteen case labels "+
			"(Interactions.c:766-978)", len(theFifteenRewards))
	}

	for _, c := range theFifteenRewards {
		t.Run(c.name, func(t *testing.T) {
			w, g, who := rewardWorld(t, c.what, 250)
			var log soundLog
			log.install(w)

			w.HandleRewards(g, who)

			if c.sound == 0 {
				log.is(t)
			} else {
				log.is(t, c.sound)
			}
			if w.Score != c.score {
				t.Errorf("Score = %d, want %d", w.Score, c.score)
			}
			if w.Mortals != c.mortals {
				t.Errorf("Mortals = %d, want %d", w.Mortals, c.mortals)
			}
			if w.Battery != c.battery {
				t.Errorf("Battery = %d, want %d", w.Battery, c.battery)
			}
			if w.Bands != c.bands {
				t.Errorf("Bands = %d, want %d", w.Bands, c.bands)
			}
			if w.Foil != c.foil {
				t.Errorf("Foil = %d, want %d", w.Foil, c.foil)
			}
			if w.StarsLeft != c.stars {
				t.Errorf("StarsLeft = %d, want %d", w.StarsLeft, c.stars)
			}
			if int(w.NumSparkles) != c.sparkles {
				t.Errorf("NumSparkles = %d, want %d", w.NumSparkles, c.sparkles)
			}
			if int(w.NumFlyingPts) != c.flying {
				t.Errorf("NumFlyingPts = %d, want %d", w.NumFlyingPts, c.flying)
			}
			if c.flying == 1 {
				fp := &w.FlyingPoints[0]
				if fp.Start != c.art {
					t.Errorf("FlyingPoints[0].Start = %d, want %d: the wrong numeral art "+
						"for %d points", fp.Start, c.art, c.score)
				}
				// Half the entry velocity, not half the exit velocity: the numerals
				// inherit the speed the glider had when it hit the prize.
				if fp.HVel != startHVel/2 || fp.VVel != startVVel/2 {
					t.Errorf("FlyingPoints[0] velocity = (%d,%d), want (%d,%d): the "+
						"numerals take half the *entry* velocity",
						fp.HVel, fp.VVel, startHVel/2, startVVel/2)
				}
			}
			if g.HVel != c.hVel || g.VVel != c.vVel {
				t.Errorf("glider velocity = (%d,%d), want (%d,%d)",
					g.HVel, g.VVel, c.hVel, c.vVel)
			}
			if got := len(w.Work2Main); got != c.workRects {
				t.Errorf("%d work rects, want %d: %q either restored a saved map when it "+
					"should not have, or did not when it should", got, c.workRects, c.name)
			}
			if len(w.Back2Work) != 0 {
				t.Errorf("%d back rects, want 0: a restore writes the background itself, "+
					"so there is nothing to restore *from* it", len(w.Back2Work))
			}
			if got := w.Room(0).Objects[0].Bonus().State != 0; got == c.cleared {
				t.Errorf("house state byte set = %v, want %v", got, !c.cleared)
			}
			if who.IsOn != c.isOn {
				t.Errorf("who.IsOn = %v, want %v", who.IsOn, c.isOn)
			}
			if who.StillOver {
				t.Error("who.StillOver = true: HandleRewards has no StillOver guard and " +
					"must not write one -- IsOn is what stops the re-dispatch")
			}
		})
	}
}

// TestEveryDispatchableRewardIsConsumed is the cross-check between the two halves of the
// reward path, and it is the test that found the missing kHelium arm.
//
// CreateActiveRects decides which object types get a kRewardIt hot spot; HandleRewards
// decides what happens when one fires. Nothing in the compiler connects them: a type with a
// rect and no arm produces a prize the player can touch forever and never collect, and the
// only symptom is a prize that does not go away.
//
// So rather than a literal list, the type set is *measured* off CreateActiveRects, and the
// requirement is the weakest one that catches the failure: every type that can produce a
// kRewardIt rect must have its house state byte cleared when that rect fires. That is true
// of all thirteen and cannot be true of a type with no arm.
//
// kSparkle and kSlider are correctly absent from the measured set -- neither gets a
// kRewardIt rect (hotspots.go:298-299) -- which is why their empty arms are consistent
// rather than broken.
func TestEveryDispatchableRewardIsConsumed(t *testing.T) {
	types := dispatchableRewardTypes(t)
	if len(types) != 13 {
		t.Fatalf("%d types make a kRewardIt rect (%v), want 13: the eleven prizes at "+
			"hotspots.go:300 plus the two grease jars", len(types), types)
	}

	for _, what := range types {
		t.Run(house.ObjectName(what), func(t *testing.T) {
			w, g, who := rewardWorld(t, what, 250)
			w.HandleRewards(g, who)

			if w.Room(0).Objects[0].Bonus().State != 0 {
				t.Errorf("state byte still set after collecting %s: HandleRewards has no "+
					"arm for it, so the prize can be touched but never taken",
					house.ObjectName(what))
			}
			if who.IsOn {
				t.Errorf("who.IsOn still true after collecting %s: its rect stays in the "+
					"collision table and re-dispatches every frame", house.ObjectName(what))
			}
		})
	}
}

// dispatchableRewardTypes runs CreateActiveRects over every object code in the house
// format's range and returns the ones that produce at least one kRewardIt rect.
//
// One world, reused: a fresh one per code would allocate three 640x480 surfaces a hundred
// and forty times over for no gain, since CreateActiveRects reads only the master entry and
// writes only the hot-spot list.
func dispatchableRewardTypes(t *testing.T) []int16 {
	t.Helper()
	w := dynaWorld(t)

	var out []int16
	for what := int16(1); what <= 0x8F; what++ {
		obj := house.Object{What: what}
		// A bonus payload with a live state byte, which is what CreateActiveRects tests
		// to decide between a jar's reward rect and a spill's slide rect. Every other
		// family reads the same ten bytes as something else and makes some rect from
		// them; only the value of `what` decides which.
		obj.SetBonus(house.Bonus{
			TopLeft: house.Point{V: prizeAt.Top, H: prizeAt.Left},
			Length:  48, Points: 250, State: 1, Initial: 1,
		})
		w.R.Master = []MasterObject{{
			RoomNum: 0, ObjectNum: 0,
			RoomLink: -1, ObjectLink: -1, LocalLink: -1,
			HotNum: -1, DynaNum: -1,
			TheObject: obj,
		}}
		w.R.Hot = w.R.Hot[:0]
		w.CreateActiveRects(0)

		for i := range w.R.Hot {
			if w.R.Hot[i].Action == RewardIt {
				out = append(out, what)
				break
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// The trigger condition
// ---------------------------------------------------------------------------

// TestACollectedPrizeCannotBeCollectedTwice pins SetObjectState's return value as the arms'
// only gate.
//
// There is no StillOver guard on a reward and no other test: the second call finds the state
// byte already clear, SetObjectState reports no change, and the arm's body is skipped
// entirely. That is what makes the reward path safe to run twice on one frame -- which it
// is, in a two-player game where both gliders overlap the same clock.
func TestACollectedPrizeCannotBeCollectedTwice(t *testing.T) {
	w, g, who := rewardWorld(t, RedClock, 0)
	var log soundLog
	log.install(w)

	w.HandleRewards(g, who)
	if w.Score != RedClockPoints {
		t.Fatalf("Score = %d after the first collection, want %d", w.Score, RedClockPoints)
	}

	// IsOn is what takes the rect out of the collision scan in the real game, but the
	// dispatcher is not in the loop here: this calls the arm directly, exactly as a second
	// glider on the *same* frame would reach it, before the cleared IsOn has been read.
	who.IsOn = true
	g.HVel, g.VVel = startHVel, startVVel
	w.HandleRewards(g, who)

	if w.Score != RedClockPoints {
		t.Errorf("Score = %d after the second collection, want %d: the state byte was "+
			"already clear", w.Score, RedClockPoints)
	}
	log.is(t, BeepsSound)
	if int(w.NumFlyingPts) != 1 {
		t.Errorf("NumFlyingPts = %d, want 1: the second collection must add nothing",
			w.NumFlyingPts)
	}
	if g.HVel != startHVel || g.VVel != startVVel {
		t.Errorf("glider velocity = (%d,%d), want (%d,%d): the second collection must not "+
			"slow the glider either", g.HVel, g.VVel, startHVel, startVVel)
	}
	if who.IsOn {
		t.Error("who.IsOn = true: the clear is outside the `if`, so it happens either way")
	}
}

// TestAPrizeThatStartsCollectedDoesNothing is the same gate from the other end: an author
// can ship an object whose initial state byte is zero, and CreateActiveRects still makes its
// rect (with IsOn false). If the dispatcher ever reaches it -- and IsThisValid is what stops
// it -- the arm has to be inert.
func TestAPrizeThatStartsCollectedDoesNothing(t *testing.T) {
	w, g, who := rewardWorld(t, YellowClock, 0)
	obj := w.Room(0).Objects[0]
	c := obj.Bonus()
	c.State = 0
	obj.SetBonus(c)
	w.Room(0).Objects[0] = obj

	var log soundLog
	log.install(w)
	w.HandleRewards(g, who)

	log.is(t)
	if w.Score != 0 {
		t.Errorf("Score = %d, want 0", w.Score)
	}
	if len(w.Work2Main) != 0 {
		t.Errorf("%d work rects, want 0: nothing was erased", len(w.Work2Main))
	}
	if who.IsOn {
		t.Error("who.IsOn = true: the clear runs even when nothing was collected")
	}
}

// TestBadMasterIndexLeavesIsOnAlone pins the port's early return as exact.
//
// The C indexes masterObjects[who->who] unguarded, so a bad index reads garbage and falls
// off the end of the switch -- and `who->isOn = false` lives inside every arm, so it is
// *not* cleared. The port returns early, which reproduces that: same nothing, including the
// nothing that looks like an oversight.
func TestBadMasterIndexLeavesIsOnAlone(t *testing.T) {
	for _, bad := range []int16{-1, 1, 999} {
		w, g, who := rewardWorld(t, RedClock, 0)
		var log soundLog
		log.install(w)
		who.Who = bad

		w.HandleRewards(g, who)

		log.is(t)
		if w.Score != 0 {
			t.Errorf("who.Who = %d: Score = %d, want 0", bad, w.Score)
		}
		if !who.IsOn {
			t.Errorf("who.Who = %d: IsOn was cleared; the C's clear is inside the arms, "+
				"so an unresolvable index leaves the rect live", bad)
		}
	}
}

// ---------------------------------------------------------------------------
// The battery and the helium balloon: one field, two signs
// ---------------------------------------------------------------------------

// TestBatteryAndHeliumShareOneSignedCounter is the pair of arms a table would have merged.
//
// Battery is positive charges and negative helium in one int16, so the two prizes do not
// accumulate -- they replace each other. Both arms test the sign they *are* (`> 0` for the
// battery, `< 0` for helium) and both take the assignment branch otherwise, which means an
// exactly-zero counter is assigned rather than added to. That last case is the one worth a
// row: it is the state a new game starts in, so it is the first battery every player
// collects.
func TestBatteryAndHeliumShareOneSignedCounter(t *testing.T) {
	for _, c := range []struct {
		name  string
		what  int16
		start int16
		want  int16
		note  string
	}{
		{"battery from empty", Battery, 0, BatterySupply,
			"zero is not `> 0`, so it is assigned; same answer, different branch"},
		{"battery tops up battery", Battery, 20, 20 + BatterySupply, ""},
		{"battery replaces helium", Battery, -100, BatterySupply,
			"the helium is thrown away, not netted off"},
		{"helium from empty", Helium, 0, -HeliumSupply,
			"zero is not `< 0` either, so the mirror branch is the assignment too"},
		{"helium tops up helium", Helium, -20, -20 - HeliumSupply,
			"more helium is more negative"},
		{"helium replaces battery", Helium, 100, -HeliumSupply, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, g, who := rewardWorld(t, c.what, 0)
			w.Battery = c.start
			w.HandleRewards(g, who)
			if w.Battery != c.want {
				t.Errorf("Battery %d -> %d, want %d (%s)", c.start, w.Battery, c.want, c.note)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Two-player doubling
// ---------------------------------------------------------------------------

// TestTwoPlayerDoublesTheFiveSupplies pins which arms double and, more importantly, which
// do not.
//
// There is one inventory for two gliders, so a supply has to be worth the same per glider as
// it is in a one-player game -- hence the second add. Points are not doubled, because a
// score is shared too and doubling it would inflate the high-score table; and the star count
// is not doubled, because it is a count of objects in the house and has nothing to do with
// how many people are playing.
func TestTwoPlayerDoublesTheFiveSupplies(t *testing.T) {
	for _, c := range []struct {
		name  string
		what  int16
		get   func(*World) int16
		one   int16
		two   int16
		doubl bool
	}{
		{"Paper", Paper, func(w *World) int16 { return w.Mortals }, 1, 2, true},
		{"Battery", Battery, func(w *World) int16 { return w.Battery },
			BatterySupply, 2 * BatterySupply, true},
		{"Bands", Bands, func(w *World) int16 { return w.Bands },
			BandsSupply, 2 * BandsSupply, true},
		{"Foil", Foil, func(w *World) int16 { return w.Foil },
			FoilSupply, 2 * FoilSupply, true},
		{"Helium", Helium, func(w *World) int16 { return w.Battery },
			-HeliumSupply, -2 * HeliumSupply, true},
		{"Star's count", Star, func(w *World) int16 { return startStars - w.StarsLeft },
			1, 1, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			// One player.
			w, g, who := rewardWorld(t, c.what, 0)
			w.HandleRewards(g, who)
			if got := c.get(w); got != c.one {
				t.Fatalf("one player: %d, want %d", got, c.one)
			}

			// Two players, both alive.
			w, g, who = rewardWorld(t, c.what, 0)
			w.TwoPlayer = true
			w.HandleRewards(g, who)
			if got := c.get(w); got != c.two {
				t.Errorf("two players: %d, want %d (doubles: %v)", got, c.two, c.doubl)
			}

			// Two players, one of them out. The survivor collects at single rate.
			w, g, who = rewardWorld(t, c.what, 0)
			w.TwoPlayer, w.OneLeft = true, true
			w.HandleRewards(g, who)
			if got := c.get(w); got != c.one {
				t.Errorf("two players, one left: %d, want %d: the doubling stops when a "+
					"player is out", got, c.one)
			}
		})
	}
}

// TestTwoPlayerDoesNotDoublePoints is the other half of the rule above, on the six arms
// that award a score.
func TestTwoPlayerDoesNotDoublePoints(t *testing.T) {
	for _, c := range []struct {
		what int16
		want int32
	}{
		{RedClock, RedClockPoints},
		{BlueClock, BlueClockPoints},
		{YellowClock, YellowClockPoints},
		{Cuckoo, CuckooClockPoints},
		{InvisBonus, 250},
		{Star, StarPoints},
	} {
		t.Run(house.ObjectName(c.what), func(t *testing.T) {
			w, g, who := rewardWorld(t, c.what, 250)
			w.TwoPlayer = true
			w.HandleRewards(g, who)
			if w.Score != c.want {
				t.Errorf("Score = %d in a two-player game, want %d", w.Score, c.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// The star
// ---------------------------------------------------------------------------

// TestLastStarFlagsGameOver is the acceptance criterion for the win condition: the frame the
// last star is taken is the frame the game ends.
//
// Two details are pinned besides the flag. The score is added *after* FlagGameOver, so the
// last star still pays -- reverse those two lines and the best possible run loses 5000
// points. And the countdown is seeded, because FlagGameOver is not the end of the game: it
// is the start of sixteen frames during which the glider keeps flying.
func TestLastStarFlagsGameOver(t *testing.T) {
	w, g, who := rewardWorld(t, Star, 0)
	w.StarsLeft = 1

	w.HandleRewards(g, who)

	if !w.GameOver {
		t.Error("GameOver = false after the last star")
	}
	if w.CountDown != NumCountDownFrames {
		t.Errorf("CountDown = %d, want %d", w.CountDown, NumCountDownFrames)
	}
	if w.StarsLeft != 0 {
		t.Errorf("StarsLeft = %d, want 0", w.StarsLeft)
	}
	if w.Score != StarPoints {
		t.Errorf("Score = %d, want %d: the score is added after the flag, so the last "+
			"star pays", w.Score, StarPoints)
	}
}

// TestNotTheLastStarDoesNotFlagGameOver is the else arm. It exists mostly to pin that the
// branch is on the count and not on the object, since DisplayStarsRemaining is a no-op until
// 1.7 and would otherwise be untestable in either direction.
func TestNotTheLastStarDoesNotFlagGameOver(t *testing.T) {
	w, g, who := rewardWorld(t, Star, 0)
	w.StarsLeft = 2
	w.HandleRewards(g, who)
	if w.GameOver {
		t.Error("GameOver = true with a star still to collect")
	}
	if w.StarsLeft != 1 {
		t.Errorf("StarsLeft = %d, want 1", w.StarsLeft)
	}
}

// TestStarCountBelowZeroStillFlagsGameOver pins `<= 0` rather than `== 0`.
//
// CountStarsInHouse cannot produce a house whose count disagrees with its objects, but a
// saved game and a two-player resync both can, and the difference between the two operators
// is whether such a house is unwinnable or merely odd.
func TestStarCountBelowZeroStillFlagsGameOver(t *testing.T) {
	w, g, who := rewardWorld(t, Star, 0)
	w.StarsLeft = 0
	w.HandleRewards(g, who)
	if !w.GameOver {
		t.Errorf("StarsLeft went to %d and GameOver = false: `== 0` would strand a house "+
			"whose count and objects disagree", w.StarsLeft)
	}
}

// TestStarStopsItsSpin pins the second thing the star arm does that no other prize does.
//
// A star owns two saved-map slots: the background under it, and a strip holding six frames
// of spin. Collecting it restores the first and retires the animation that would otherwise
// keep painting over the restored background every frame.
func TestStarStopsItsSpin(t *testing.T) {
	w, g, who := rewardWorld(t, Star, 0)
	w.R.Stars = []render.Anim{
		{Where: 0, Who: 0, SavedMap: 1},
		{Where: 0, Who: 1, SavedMap: 2},
	}

	w.HandleRewards(g, who)

	if !w.R.Stars[0].Stopped {
		t.Error("Stars[0].Stopped = false: the collected star keeps spinning over the " +
			"background that was just put back")
	}
	if w.R.Stars[1].Stopped {
		t.Error("Stars[1].Stopped = true: a different star was retired")
	}
}

// TestCuckooStopsItsPendulum is the same for the one clock with a moving part.
//
// The order inside the arm matters and is not observable here: the restore has to run first,
// because stopping the pendulum only stops *future* frames -- reversed, the last cel drawn
// would be frozen on the wall under a restored background. What is observable is that both
// happen, which is what this asserts.
func TestCuckooStopsItsPendulum(t *testing.T) {
	w, g, who := rewardWorld(t, Cuckoo, 0)
	w.R.Pendulums = []render.Anim{
		{Where: 0, Who: 0, SavedMap: 1},
		{Where: 1, Who: 0, SavedMap: 2},
	}

	w.HandleRewards(g, who)

	if !w.R.Pendulums[0].Stopped {
		t.Error("Pendulums[0].Stopped = false: the swing outlives the clock")
	}
	if w.R.Pendulums[1].Stopped {
		t.Error("Pendulums[1].Stopped = true: a clock in another room was retired")
	}
	if int(w.NumSparkles) != 0 {
		t.Errorf("NumSparkles = %d, want 0: the cuckoo is a points prize and gets "+
			"numerals, not a puff", w.NumSparkles)
	}
}

// ---------------------------------------------------------------------------
// The invisible bonus
// ---------------------------------------------------------------------------

// TestInvisBonusReadsPointsBeforeTheStateWriteCorruptsThem is an order dependency between
// two functions, and the kind of thing that survives a refactor only if something pins it.
//
// SetObjectState's prize family writes a zero into the master copy through the *blower*
// union member -- payload offset 7, which in bonusType is the low byte of `points`. So
// HandleRewards reads the author's value out of the master copy *before* calling
// SetObjectState. Swap the two lines and every invisible bonus loses its low byte: 500
// becomes 256, 100 becomes 0.
//
// The test drives that directly. It awards a bonus whose value has a non-zero low byte, and
// asserts both that the player got the whole number and that the master copy's low byte was
// indeed destroyed -- so it fails if the read moves, and it also fails if somebody "fixes"
// setstate.go's offset, which would be a change to what the object graph says about every
// collected prize in the locale.
func TestInvisBonusReadsPointsBeforeTheStateWriteCorruptsThem(t *testing.T) {
	const authored int16 = 500

	w, g, who := rewardWorld(t, InvisBonus, authored)
	w.HandleRewards(g, who)

	if w.Score != int32(authored) {
		t.Errorf("Score = %d, want %d: the point value was read after SetObjectState "+
			"clobbered its low byte", w.Score, authored)
	}
	if got := w.R.Master[0].TheObject.Bonus().Points; got != authored&^0xFF {
		t.Errorf("master copy's Points = %d, want %d: setstate.go's reproduced "+
			"offset-7 write is what makes the read order matter, and it is gone",
			got, authored&^0xFF)
	}
	if w.Room(0).Objects[0].Bonus().Points != authored {
		t.Error("the house copy's Points changed: only the master copy is corrupted")
	}
}

// TestInvisBonusPointArtFallsBackToTheThousandCels is the author-facing consequence of
// AddFlyingPoint's switch having five cases and a default.
//
// An invisible bonus's value is whatever the author typed. 100, 250, 300 and 500 have their
// own numerals; anything else -- 700, 1000, 42 -- draws the 1000 art while scoring the
// authored number. Reproduced, and worth a test because it is the one place in the game
// where the score and the number the player sees can disagree.
func TestInvisBonusPointArtFallsBackToTheThousandCels(t *testing.T) {
	for _, c := range []struct {
		points int16
		art    int16
	}{
		{100, 12}, {250, 9}, {300, 6}, {500, 3}, {1000, 0},
		{700, 0}, {42, 0},
	} {
		w, g, who := rewardWorld(t, InvisBonus, c.points)
		w.HandleRewards(g, who)
		if w.Score != int32(c.points) {
			t.Errorf("%d points: Score = %d", c.points, w.Score)
		}
		if w.FlyingPoints[0].Start != c.art {
			t.Errorf("%d points: numeral art %d, want %d",
				c.points, w.FlyingPoints[0].Start, c.art)
		}
	}
}

// ---------------------------------------------------------------------------
// The foil
// ---------------------------------------------------------------------------

// TestFoilStartsTheDissolveAndRefreshesNoCounter pins the one supply with no scoreboard
// number: what the player sees is the glider's own art changing.
func TestFoilStartsTheDissolveAndRefreshesNoCounter(t *testing.T) {
	w, g, who := rewardWorld(t, Foil, 0)
	w.HandleRewards(g, who)

	if w.Foil != FoilSupply {
		t.Errorf("Foil = %d, want %d", w.Foil, FoilSupply)
	}
	if g.Mode != player.GliderGoingFoil {
		t.Errorf("glider Mode = %d, want GliderGoingFoil (%d)", g.Mode, player.GliderGoingFoil)
	}
	if g.Frame != 0 {
		t.Errorf("glider Frame = %d, want 0: the dissolve starts from the first cel", g.Frame)
	}
}

// TestFoilOnAGliderAlreadyDissolvingIsStillCollected separates the two halves of the arm.
//
// StartGliderFoilGoing refuses a glider that is already mid-dissolve, but the supply is
// added before that call and unconditionally -- so flying through two foil sheets in
// consecutive frames banks both, and only the first restarts the animation. Worth pinning
// because the natural "fix" is to move the supply inside the guard.
func TestFoilOnAGliderAlreadyDissolvingIsStillCollected(t *testing.T) {
	w, g, who := rewardWorld(t, Foil, 0)
	g.Mode = player.GliderGoingFoil
	g.Frame = 3

	w.HandleRewards(g, who)

	if w.Foil != FoilSupply {
		t.Errorf("Foil = %d, want %d: the supply is added outside the mode guard",
			w.Foil, FoilSupply)
	}
	if g.Frame != 3 {
		t.Errorf("glider Frame = %d, want 3: the dissolve must not restart", g.Frame)
	}
}

// ---------------------------------------------------------------------------
// RestoreFromSavedMap
// ---------------------------------------------------------------------------

// TestRestoreFromSavedMapWritesBothMaps is the whole point of the function, and the half
// that is easy to drop is the back map.
//
// Work only: the prize reappears the moment the glider flies away and the frame's
// back->work restore paints it again. Back only: the prize is gone from the next frame
// onwards but is still on screen for this one. The C writes both and so must this, which is
// what makes an erase permanent without recomposing the room.
func TestRestoreFromSavedMapWritesBothMaps(t *testing.T) {
	w := dynaWorld(t)
	w.R.RoomNumber = 0

	// A patch of a colour neither map holds, so "the patch landed here" is a positive
	// observation rather than the absence of one.
	const ink uint8 = 0x5A
	sm := savedMapUnder(0, 0, prizeAt, w.R.V.OriginH, w.R.V.OriginV)
	fill(sm.Map, ink)
	w.R.SavedMaps = []render.SavedMap{sm}
	clearRects(w)

	w.RestoreFromSavedMap(0, 0, false)

	for _, m := range []struct {
		name string
		surf *render.Surface
	}{{"Back", w.R.Back}, {"Work", w.R.Work}} {
		if got := at(m.surf, sm.Dest.Left, sm.Dest.Top); got != ink {
			t.Errorf("%s at the patch's top-left = %#x, want %#x", m.name, got, ink)
		}
		if got := at(m.surf, sm.Dest.Right-1, sm.Dest.Bottom-1); got != ink {
			t.Errorf("%s at the patch's bottom-right = %#x, want %#x", m.name, got, ink)
		}
		if got := at(m.surf, sm.Dest.Right, sm.Dest.Top); got == ink {
			t.Errorf("%s one pixel right of the patch = %#x: the blit overran", m.name, got)
		}
	}
	if len(w.Work2Main) != 1 {
		t.Errorf("%d work rects, want 1", len(w.Work2Main))
	}
	if len(w.Back2Work) != 0 {
		t.Errorf("%d back rects, want 0: the background *is* what was just written",
			len(w.Back2Work))
	}
}

// TestRestoreFromSavedMapStopsAtTheFirstMatch is the load-bearing `break`.
//
// A star and a cuckoo each own two slots with the same (where, who): the object's swatch,
// claimed by the composition, and the animation's frame strip, claimed by AddStar or
// AddPendulum afterwards. The strip's Dest is anchored at the *corner of the screen*,
// because that is the rect it was handed. Without the break, collecting a star would paint
// six frames of star art over the top-left of the play area.
func TestRestoreFromSavedMapStopsAtTheFirstMatch(t *testing.T) {
	w := dynaWorld(t)
	w.R.RoomNumber = 0

	const swatchInk, stripInk uint8 = 0x11, 0x22
	swatch := savedMapUnder(0, 0, prizeAt, w.R.V.OriginH, w.R.V.OriginV)
	fill(swatch.Map, swatchInk)

	// AddStar's slot, verbatim: SetRect(0, 0, 32, 31*NumStarFrames) -- never offset, so
	// its Dest really is the screen corner.
	strip := render.SavedMap{
		Map:   render.NewSurface(32, 31*6),
		Dest:  render.SetRect(0, 0, 32, 31*6),
		Where: 0, Who: 0,
	}
	fill(strip.Map, stripInk)

	w.R.SavedMaps = []render.SavedMap{swatch, strip}
	clearRects(w)

	w.RestoreFromSavedMap(0, 0, false)

	if got := at(w.R.Back, swatch.Dest.Left, swatch.Dest.Top); got != swatchInk {
		t.Errorf("Back at the prize = %#x, want the swatch %#x", got, swatchInk)
	}
	if got := at(w.R.Back, 0, 0); got == stripInk {
		t.Fatal("Back at (0,0) holds the frame strip: the loop did not stop at the first " +
			"match, and six cels of animation have been painted over the corner of the " +
			"play area")
	}
	if len(w.Work2Main) != 1 {
		t.Errorf("%d work rects, want 1: one restore, one rect", len(w.Work2Main))
	}
}

// TestRestoreFromSavedMapMatchesOnBothCoordinates. The table is keyed by (room, object) and
// a house has 24 object slots per room, so matching on either half alone would erase the
// wrong thing in about one room in twenty-four.
func TestRestoreFromSavedMapMatchesOnBothCoordinates(t *testing.T) {
	for _, c := range []struct {
		name        string
		where, who  int16
		wantRestore bool
	}{
		{"both match", 3, 7, true},
		{"wrong room", 4, 7, false},
		{"wrong object", 3, 8, false},
		{"neither", 9, 9, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := dynaWorld(t)
			sm := savedMapUnder(3, 7, prizeAt, w.R.V.OriginH, w.R.V.OriginV)
			w.R.SavedMaps = []render.SavedMap{sm}
			clearRects(w)

			w.RestoreFromSavedMap(c.where, c.who, false)

			if got := len(w.Work2Main) == 1; got != c.wantRestore {
				t.Errorf("restored = %v, want %v", got, c.wantRestore)
			}
		})
	}
}

// TestRestoreFromSavedMapSkipsAnUnclaimedSlot. backUpToSavedMap's redraw branch can leave a
// slot with a nil Map -- it returns -1 without appending -- and 1.5f's animators will add
// more slots than they fill. A nil patch has to be skipped rather than dereferenced, and it
// must not consume the match: the *next* slot with the same key is the real one.
func TestRestoreFromSavedMapSkipsAnUnclaimedSlot(t *testing.T) {
	w := dynaWorld(t)
	const ink uint8 = 0x3C

	real := savedMapUnder(0, 0, prizeAt, w.R.V.OriginH, w.R.V.OriginV)
	fill(real.Map, ink)
	w.R.SavedMaps = []render.SavedMap{{Where: 0, Who: 0}, real}
	clearRects(w)

	w.RestoreFromSavedMap(0, 0, false)

	if got := at(w.R.Back, real.Dest.Left, real.Dest.Top); got != ink {
		t.Errorf("Back at the prize = %#x, want %#x: the nil slot swallowed the match",
			got, ink)
	}
}

// TestRestoreFromSavedMapSparklesOnlyWhenAsked pins the doSparkle argument, which is the
// only difference between the two call sites.
//
// False on the reward path: the player flew into the prize and gets a flying numeral or the
// arm's own sparkle. True on the switch path: the prize vanishes somewhere the player may
// not be looking, so a puff of light and a fade-out sound say what happened.
func TestRestoreFromSavedMapSparklesOnlyWhenAsked(t *testing.T) {
	for _, doSparkle := range []bool{false, true} {
		w := dynaWorld(t)
		w.R.SavedMaps = []render.SavedMap{
			savedMapUnder(0, 0, prizeAt, w.R.V.OriginH, w.R.V.OriginV)}
		var log soundLog
		log.install(w)
		clearRects(w)

		w.RestoreFromSavedMap(0, 0, doSparkle)

		wantSparkles := 0
		if doSparkle {
			wantSparkles = 1
		}
		if int(w.NumSparkles) != wantSparkles {
			t.Errorf("doSparkle = %v: NumSparkles = %d, want %d",
				doSparkle, w.NumSparkles, wantSparkles)
		}
		if doSparkle {
			log.is(t, player.FadeOutSound)
		} else {
			log.is(t)
		}
	}
}

// TestRestoreFromSavedMapUndoesItsOwnScreenOffset is the round trip that looks pointless
// and is not.
//
// SavedMap.Dest is in screen coordinates. AddSparkle takes a room-local rect and adds the
// play origin itself. So the sparkle's rect has to be converted *back* to room-local purely
// to be converted forward again -- the C's line for line, and the one place in the port
// where that happens. A port that dropped the subtraction would put the puff two play
// origins from the prize, which on a 640x480 screen is 128 pixels right and 158 down.
func TestRestoreFromSavedMapUndoesItsOwnScreenOffset(t *testing.T) {
	w := dynaWorld(t)
	sm := savedMapUnder(0, 0, prizeAt, w.R.V.OriginH, w.R.V.OriginV)
	w.R.SavedMaps = []render.SavedMap{sm}

	w.RestoreFromSavedMap(0, 0, true)

	if w.NumSparkles != 1 {
		t.Fatalf("NumSparkles = %d, want 1", w.NumSparkles)
	}
	// The puff is a 20x19 cel centred in the patch, so its corner is the patch's corner
	// plus half the difference in each axis. Spelled out here rather than by re-invoking
	// CenterIn, which would pass whatever the offset arithmetic did; the cel's size comes
	// from the source rect because that is a measurement of the art and not of the code
	// under test.
	//
	// The halves truncate, and the cel's odd height means the puff sits one pixel above
	// the geometric centre. Worth spelling out rather than tolerating: a test that allowed
	// a pixel of slack would also allow an off-by-one introduced later.
	src := render.SparkleSrc[0]
	wantLeft := sm.Dest.Left + (sm.Dest.Wide()-src.Wide())/2
	wantTop := sm.Dest.Top + (sm.Dest.Tall()-src.Tall())/2
	got := w.Sparkles[0].Bounds
	if got.Left != wantLeft || got.Top != wantTop {
		t.Errorf("sparkle at (%d,%d), want (%d,%d): off by (%d,%d), which is a play "+
			"origin if it is (%d,%d)",
			got.Left, got.Top, wantLeft, wantTop,
			got.Left-wantLeft, got.Top-wantTop, w.R.V.OriginH, w.R.V.OriginV)
	}
}

// ---------------------------------------------------------------------------
// Surface helpers
// ---------------------------------------------------------------------------

// fill paints a whole surface one colour. Index 0 is what NewSurface leaves, so every ink
// these tests use is non-zero and "the patch landed" is a positive observation.
func fill(s *render.Surface, v uint8) {
	s.Fill(s.Bounds(), v)
}

// at reads one pixel, returning 0 for a coordinate outside the surface so that an
// off-by-one in a test's own arithmetic reads as "not the ink" rather than panicking.
func at(s *render.Surface, x, y int16) uint8 {
	if int(x) < 0 || int(x) >= s.W || int(y) < 0 || int(y) >= s.H {
		return 0
	}
	return s.Pix[int(y)*s.W+int(x)]
}
