package game

// HandleRewards (Interactions.c:756-979) -- the prizes.
//
// Two hundred and twenty lines, and the largest single function in Interactions.c: one arm
// per collectable, and the collectables are where most of the game's inventory, scoring
// and sound live. The clocks, the paper, the batteries, the bands, the grease, the foil,
// the stars, the helium, the sliders and the invisible bonus all pass through it.
//
// ---------------------------------------------------------------------------
// One shape, nine times
// ---------------------------------------------------------------------------
//
// The switch has **fifteen case labels over fourteen arms** -- kGreaseRt and kGreaseLf share
// one body -- and the two counts are worth keeping apart, because the deviations below are
// counted per label (a house has fifteen kinds of collectable) while the shape is a property
// of the arms. Nine of the fourteen arms are the same seven steps in the same order:
//
//	if SetObjectState(...)              the prize is switched off, and this is the
//	    PlayPrioritySound               *only* test -- a second glider touching the
//	    RestoreFromSavedMap             same prize on the same frame gets nothing,
//	    AddSparkle or AddFlyingPoint    because the state byte is already clear
//	    slow the glider down
//	    award the points or the supply
//	    RedrawAllGrease
//	who.IsOn = false
//
// The other five account for themselves: the star omits the velocity change, the invisible
// bonus omits the restore and the regrease, the two grease labels are one arm that does
// almost nothing, and the sparkle and the slider are empty. Six of the nine carry exactly one
// extra call -- a pendulum to stop, a counter to refresh, the foil animation to start -- and
// the three clocks carry none.
//
// It is transcribed longhand rather than collapsed into a table, for two reasons. The
// deviations are not decoration -- four of the labels award neither points nor supply, five
// award a supply and no points, one takes no saved map, one does not slow the glider at all,
// and two do not even clear IsOn -- and every one of those is a thing a reader will want to
// check against the C line by line. And the arms differ in *which* counter they touch and
// which quarter of the scoreboard they refresh, which is exactly the sort of mapping a table
// hides: the battery and the helium balloon write the same field with opposite signs, which
// is the one pair a table would have merged and must not.
//
// ---------------------------------------------------------------------------
// The three deviations worth knowing before reading
// ---------------------------------------------------------------------------
//
//	the two grease jars  no sound, no points, no restore: SpillGrease does all of it,
//	                     because a knocked-over jar is not removed, it becomes a slick
//	kSparkle, kSlider    bare breaks. **Neither clears IsOn**, so their reward rects
//	                     stay live and are re-dispatched every frame forever -- which is
//	                     harmless only because both arms are empty
//	kStar                no velocity change. Every other prize slows the glider; the
//	                     star, the most valuable thing in the game, does not touch it
//
// ---------------------------------------------------------------------------
// Two-player doubling
// ---------------------------------------------------------------------------
//
// Five arms add their supply twice when `twoPlayerGame && !onePlayerLeft`, because there
// is one inventory for both players and a prize has to be worth the same per glider. Once
// one player is out, the doubling stops -- so the survivor of a two-player game collects
// at single rate. Note what is *not* doubled: points, and the star count.

import "glidergo/internal/game/player"

// What a prize is worth (GliderDefines.h:537-541), and what a supply refills
// (Interactions.c:16-19, which are file-local #defines there and file-local here for the
// same reason -- nothing else in the game refers to them).
//
// The point values are duplicated as literals in the AddFlyingPoint calls below, exactly as
// the C duplicates them, because AddFlyingPoint switches on the number to pick its art: the
// constant is what the player scores and the literal is which numerals fly away, and they
// agree only because someone kept them in step. See AddFlyingPoint's note on the default
// arm, which is what an author's 700-point invisible bonus lands in.
const (
	RedClockPoints    int32 = 100
	BlueClockPoints   int32 = 300
	YellowClockPoints int32 = 500
	CuckooClockPoints int32 = 1000
	StarPoints        int32 = 5000

	// BatterySupply is "about 2 rooms worth of thrust", per the original's comment.
	// HeliumSupply is three times that, because helium is spent faster.
	BatterySupply int16 = 50
	HeliumSupply  int16 = 150
	BandsSupply   int16 = 8
	FoilSupply    int16 = 8
)

// The five prize sounds and their priorities (GliderDefines.h:58-116, :160-172). The three
// clocks each get their own, which is the game's clearest piece of audio feedback: the
// pitch tells you what you just took without looking at the scoreboard.
const (
	BeepsSound    int16 = 3
	BuzzerSound   int16 = 4
	DingSound     int16 = 5
	EnergizeSound int16 = 6
	CuckooSound   int16 = 11
	BonusSound    int16 = 61

	BeepsPriority    int16 = 800
	BuzzerPriority   int16 = 801
	DingPriority     int16 = 802
	EnergizePriority int16 = 803
	CuckooPriority   int16 = 805
	BonusPriority    int16 = 812
)

// HandleRewards is Interactions.c:756-979: the glider has touched a prize.
//
// It is reached from the kRewardIt arm of the collision dispatcher, once per frame per
// overlapping glider -- there is no StillOver guard anywhere in it. What makes it fire once
// instead of every frame is `who.IsOn = false` at the end of each arm, which takes the rect
// out of CheckForHotSpots' scan, plus SetObjectState's return value as a second line of
// defence for the two frames in between.
func (w *World) HandleRewards(thisGlider *player.Glider, who *HotObject) {
	// The C indexes masterObjects[who->who] unguarded. A bad index there falls off the
	// end of the switch and does nothing at all -- not even clearing IsOn, since that
	// line is inside every arm -- so returning early is exact, not merely safe.
	if !w.masterValid(who.Who) {
		return
	}
	whoLinked := who.Who
	bounds := who.Bounds
	m := &w.R.Master[whoLinked]

	switch m.TheObject.What {
	case RedClock:
		if w.takePrize(m, whoLinked) {
			w.PlayPrioritySound(BeepsSound, BeepsPriority)
			w.RestoreFromSavedMap(w.R.RoomNumber, m.ObjectNum, false)
			// Half the velocity to the numerals, then a quarter of it left to the
			// glider. The order is the C's and it matters: the points inherit the
			// speed the glider had when it hit the clock, not the speed it leaves
			// with, so a fast pass throws the score across the room.
			w.AddFlyingPoint(bounds, 100, thisGlider.HVel/2, thisGlider.VVel/2)
			thisGlider.HVel /= 4
			thisGlider.VVel /= 4
			w.Score += RedClockPoints
			w.RedrawAllGrease()
		}
		who.IsOn = false

	case BlueClock:
		if w.takePrize(m, whoLinked) {
			w.PlayPrioritySound(BuzzerSound, BuzzerPriority)
			w.RestoreFromSavedMap(w.R.RoomNumber, m.ObjectNum, false)
			w.AddFlyingPoint(bounds, 300, thisGlider.HVel/2, thisGlider.VVel/2)
			thisGlider.HVel /= 4
			thisGlider.VVel /= 4
			w.Score += BlueClockPoints
			w.RedrawAllGrease()
		}
		who.IsOn = false

	case YellowClock:
		if w.takePrize(m, whoLinked) {
			w.PlayPrioritySound(DingSound, DingPriority)
			w.RestoreFromSavedMap(w.R.RoomNumber, m.ObjectNum, false)
			w.AddFlyingPoint(bounds, 500, thisGlider.HVel/2, thisGlider.VVel/2)
			thisGlider.HVel /= 4
			thisGlider.VVel /= 4
			w.Score += YellowClockPoints
			w.RedrawAllGrease()
		}
		who.IsOn = false

	case Cuckoo:
		if w.takePrize(m, whoLinked) {
			w.PlayPrioritySound(CuckooSound, CuckooPriority)
			w.RestoreFromSavedMap(w.R.RoomNumber, m.ObjectNum, false)
			// The one clock with a moving part. StopPendulum has to come *after* the
			// restore: the restore puts the wall back and this stops the animator
			// painting the swing over it again next frame. Reversed, the last cel of
			// the pendulum would be frozen on the wall for the rest of the room.
			w.R.StopPendulum(w.R.RoomNumber, m.ObjectNum)
			w.AddFlyingPoint(bounds, 1000, thisGlider.HVel/2, thisGlider.VVel/2)
			thisGlider.HVel /= 4
			thisGlider.VVel /= 4
			w.Score += CuckooClockPoints
			w.RedrawAllGrease()
		}
		who.IsOn = false

	case Paper:
		if w.takePrize(m, whoLinked) {
			w.PlayPrioritySound(EnergizeSound, EnergizePriority)
			w.RestoreFromSavedMap(w.R.RoomNumber, m.ObjectNum, false)
			w.AddSparkle(bounds)
			// A supply arm halves the velocity where a points arm quarters it, and
			// gets a sparkle where a points arm gets numerals. Both differences run
			// through all ten arms and neither has an exception.
			thisGlider.HVel /= 2
			thisGlider.VVel /= 2
			w.Mortals++
			if w.TwoPlayer && !w.OneLeft {
				w.Mortals++
			}
			w.QuickGlidersRefresh()
			w.RedrawAllGrease()
		}
		who.IsOn = false

	case Battery:
		if w.takePrize(m, whoLinked) {
			w.PlayPrioritySound(EnergizeSound, EnergizePriority)
			w.RestoreFromSavedMap(w.R.RoomNumber, m.ObjectNum, false)
			w.AddSparkle(bounds)
			thisGlider.HVel /= 2
			thisGlider.VVel /= 2
			// One signed counter for two power-ups, so picking up a battery while
			// carrying helium *replaces* the helium rather than adding to it -- and
			// the test is `> 0`, not `>= 0`, so an exactly-empty counter is also
			// replaced rather than topped up. Same value either way; the branch only
			// matters for the sign.
			if w.Battery > 0 {
				w.Battery += BatterySupply
			} else {
				w.Battery = BatterySupply
			}
			if w.TwoPlayer && !w.OneLeft {
				w.Battery += BatterySupply
			}
			w.QuickBatteryRefresh(false)
			w.RedrawAllGrease()
		}
		who.IsOn = false

	case Bands:
		if w.takePrize(m, whoLinked) {
			w.PlayPrioritySound(EnergizeSound, EnergizePriority)
			w.RestoreFromSavedMap(w.R.RoomNumber, m.ObjectNum, false)
			w.AddSparkle(bounds)
			thisGlider.HVel /= 2
			thisGlider.VVel /= 2
			w.Bands += BandsSupply
			if w.TwoPlayer && !w.OneLeft {
				w.Bands += BandsSupply
			}
			w.QuickBandsRefresh(false)
			w.RedrawAllGrease()
		}
		who.IsOn = false

	case GreaseRt, GreaseLf:
		// The shortest arm and the odd one out: no sound, no points, no sparkle and no
		// restore. A grease jar is not collected, it is knocked over -- SpillGrease
		// redraws the object as a slick in place, so the background under it must
		// *not* be put back. Charged to 1.5e; see triggers.go.
		if w.takePrize(m, whoLinked) {
			w.SpillGrease(m.DynaNum, m.HotNum)
		}
		who.IsOn = false

	case Foil:
		if w.takePrize(m, whoLinked) {
			w.PlayPrioritySound(EnergizeSound, EnergizePriority)
			w.RestoreFromSavedMap(w.R.RoomNumber, m.ObjectNum, false)
			w.AddSparkle(bounds)
			thisGlider.HVel /= 2
			thisGlider.VVel /= 2
			w.Foil += FoilSupply
			if w.TwoPlayer && !w.OneLeft {
				w.Foil += FoilSupply
			}
			// No scoreboard refresh: the foil count is not on the board. What the
			// player sees is the glider's own art changing, and that is a mode with a
			// dissolve, not a counter.
			thisGlider.StartGliderFoilGoing(w)
			w.RedrawAllGrease()
		}
		who.IsOn = false

	case InvisBonus:
		// **The read of `points` before the SetObjectState call is load-bearing.**
		// SetObjectState's prize family writes a zero through the *blower* union member
		// of this same master copy -- payload offset 7, which in bonusType is the low
		// byte of `points` (see setstate.go, where the bug is reproduced and explained).
		// So reading the value afterwards would give an author's 500-point bonus back as
		// 256 and a 100-point one as 0. The original reads it first and so does this;
		// the two lines cannot be swapped.
		points := m.TheObject.Bonus().Points
		if w.takePrize(m, whoLinked) {
			w.PlayPrioritySound(BonusSound, BonusPriority)
			// No RestoreFromSavedMap, and correctly so: an invisible bonus is one of
			// the two prize types the composition draws nothing for, so it never
			// claimed a saved map to restore. No RedrawAllGrease either, for the same
			// reason -- nothing was erased, so nothing needs repairing.
			w.AddFlyingPoint(bounds, points, thisGlider.HVel/2, thisGlider.VVel/2)
			thisGlider.HVel /= 4
			thisGlider.VVel /= 4
			w.Score += int32(points)
		}
		who.IsOn = false

	case Star:
		if w.takePrize(m, whoLinked) {
			w.PlayPrioritySound(EnergizeSound, EnergizePriority)
			w.RestoreFromSavedMap(w.R.RoomNumber, m.ObjectNum, false)
			w.AddSparkle(bounds)
			w.R.StopStar(w.R.RoomNumber, m.ObjectNum)
			// **No velocity change.** The only prize in the game that does not slow
			// the glider down, which is what makes a run of stars collectable in one
			// pass -- and is presumably deliberate, since the star is the object the
			// whole house is built around.
			w.StarsLeft--
			if w.StarsLeft <= 0 {
				// `<= 0` and not `== 0`: the guard is against a house whose star count
				// and star objects disagree, which CountStarsInHouse cannot produce but
				// a two-player resync or a saved game can.
				w.FlagGameOver()
			} else {
				w.DisplayStarsRemaining()
			}
			w.RedrawAllGrease()
			// The score comes *after* the game-over flag, so the last star still pays.
			w.Score += StarPoints
		}
		who.IsOn = false

	case Sparkle:
		// A bare break in the C, and note what is missing: `who.IsOn = false`. A
		// sparkle emitter's reward rect therefore stays live and is re-dispatched on
		// every frame the glider overlaps it, forever. Harmless because the arm is
		// empty, and kept because "collecting" a sparkle would be a behaviour change --
		// the emitter is switched off by IsThisValid when its state byte is cleared,
		// and nothing here clears it.

	case Helium:
		if w.takePrize(m, whoLinked) {
			w.PlayPrioritySound(EnergizeSound, EnergizePriority)
			w.RestoreFromSavedMap(w.R.RoomNumber, m.ObjectNum, false)
			w.AddSparkle(bounds)
			thisGlider.HVel /= 2
			thisGlider.VVel /= 2
			// The mirror image of the battery arm, and the sign is the whole of it:
			// one signed counter, positive for battery charges and negative for
			// helium, so the two prizes cancel rather than accumulate. The test is
			// `< 0` here against the battery's `> 0` -- both mean "already the kind I
			// am", both leave an exactly-zero counter to the assignment branch -- and
			// the two-player doubling *subtracts*, because more helium is more
			// negative. Get any one of those three signs wrong and a helium balloon
			// hands the player battery thrust instead of buoyancy.
			if w.Battery < 0 {
				w.Battery -= HeliumSupply
			} else {
				w.Battery = -HeliumSupply
			}
			if w.TwoPlayer && !w.OneLeft {
				w.Battery -= HeliumSupply
			}
			w.QuickBatteryRefresh(false)
			w.RedrawAllGrease()
		}
		who.IsOn = false

	case Slider:
		// The other bare break, for the same reason: a slide strip is not a prize. It
		// shares bonusType, which is the only thing that puts it in this switch at all.
	}
}

// takePrize is the SetObjectState call all twelve non-empty arms open with, spelled
// once because the argument list is what a reader has to check and it is identical every
// time.
//
// Two things in it are worth naming. The action is a literal 0 -- Toggle -- and it is
// **ignored**: the prize family of SetObjectState clears the state byte whatever the
// action says, which is why a prize can never be put back and why a house has to be
// reloaded to replay it. And the room is `thisRoomNumber`, not the object's own RoomNum,
// which is safe only because a kRewardIt hot spot exists solely for the central room --
// CreateActiveRects never builds one for a neighbour. The two are always equal here; if
// they ever stop being, this is the line that breaks.
func (w *World) takePrize(m *MasterObject, whoLinked int16) bool {
	return w.SetObjectState(w.R.RoomNumber, m.ObjectNum, Toggle, whoLinked)
}
