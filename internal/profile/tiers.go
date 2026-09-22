package profile

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The profile a new house is held to: docs/analysis/original-houses.md 10.2, "Numeric
// targets", eighteen rows by five size tiers.
//
// It is a transcription and is meant to stay one. TestTheTierTableMatchesTheDocument
// parses 10.2's markdown and compares it cell by cell against the table below, so the
// numbers exist in one place and the document is the place -- an author reads 10.2 and a
// tool checks 10.2, and neither can drift from the other without a test failing.
//
// # What the bands are and are not
//
// They are the *observed spread of the 1994 houses*, tier by tier, and nothing more.
// 10.2 says "pick a tier, then hit these", and the honest reading of a miss is "no
// shipped house of this size looked like that", which is a reason to look and not a
// verdict. That is why Check returns misses rather than errors and why `house stats`
// prints every row whether it passes or not: a house that is deliberately unlike the
// originals is a legitimate thing to build, and the tool's job is to say so out loud
// rather than to refuse.
//
// Two consequences of being a measurement rather than a rule:
//
//   - **The tiers overlap and are not monotone.** Small's objects/room band is 7-19 and
//     medium's is 5.2-8.2, so a house at 9 objects a room is small-shaped and not
//     medium-shaped, which is the opposite of what "bigger tier, more of everything"
//     would predict. That is Sampler and Fun House being dense little houses, and it is
//     real; docs/IMPROVEMENTS.md 4.18 is the entry about where the tiers contradict
//     10.3's construction procedure.
//   - **A tier with one or two houses in it publishes their exact values as a band.**
//     Tutorial's "stars 1" is not a design rule, it is that both tutorial-sized houses
//     ship one star. 4.18 narrowed that column for exactly this reason.

// Tier is one of 10.2's five columns.
type Tier int

const (
	Tutorial Tier = iota
	Small
	Medium
	Large
	Epic
)

// TierNames are the column headings, in column order and spelled as 10.2 spells them --
// which is also what `glidertool house stats -tier` takes.
var TierNames = [...]string{"tutorial", "small", "medium", "large", "epic"}

func (t Tier) String() string {
	if t < 0 || int(t) >= len(TierNames) {
		return "tier(" + strconv.Itoa(int(t)) + ")"
	}
	return TierNames[t]
}

// ParseTier resolves a tier by name, case-insensitively.
func ParseTier(s string) (Tier, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	for i, n := range TierNames {
		if n == s {
			return Tier(i), true
		}
	}
	return 0, false
}

// Unit is how a row's numbers are written, which decides how both the band and the
// measurement are printed. Getting this wrong is how a table comes to say a house is at
// "0 dark rooms" when it means 0.4 %.
type Unit int

const (
	UnitCount   Unit = iota // whole things: rooms, objects, stars, points
	UnitRatio               // per-room means and the grid density
	UnitPercent             // the two rows 10.2 writes with a % sign
)

// Format prints one number in this row's unit.
func (u Unit) Format(v float64) string {
	switch u {
	case UnitPercent:
		return trimZeros(strconv.FormatFloat(v, 'f', 1, 64)) + " %"
	case UnitRatio:
		return trimZeros(strconv.FormatFloat(v, 'f', 2, 64))
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// trimZeros turns "0.60" into "0.6" and "20.0" into "20", so a transcribed band prints
// the way 10.2 writes it rather than padded out to the formatter's width.
func trimZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// Band is one cell of 10.2: an inclusive range. Lo == Hi is a cell written as a single
// number, and NA is the one cell written "n/a".
type Band struct {
	Lo, Hi float64
	NA     bool
}

// Contains is the test Check applies. Inclusive at both ends, because 10.2's bands are
// the corpus extremes themselves -- Slumberland is 383 rooms and epic's band opens at
// 383, so an exclusive test would fail the house the band was measured from.
func (b Band) Contains(v float64) bool {
	if b.NA || math.IsNaN(v) {
		return false
	}
	return v >= b.Lo && v <= b.Hi
}

// String prints a band the way 10.2 writes it.
func (b Band) String(u Unit) string {
	switch {
	case b.NA:
		return "n/a"
	case b.Lo == b.Hi:
		return u.Format(b.Lo)
	}
	lo, hi := u.Format(b.Lo), u.Format(b.Hi)
	// A percent range carries one sign, at the end: "10-25 %", not "10 %-25 %".
	if u == UnitPercent {
		lo = strings.TrimSuffix(lo, " %")
	}
	return lo + "-" + hi
}

// Row is one of 10.2's eighteen rows: its label, how to print it, the five bands, and
// how to get the number out of a Profile.
//
// Measure returning (value, false) means the row cannot be evaluated for this house
// rather than that it failed -- prize:enemy on a house with no enemies is the case, and
// it is the same case 10.2 writes as "n/a" in the tutorial column.
type Row struct {
	Target  string
	Unit    Unit
	Bands   [len(TierNames)]Band
	Measure func(Profile) (float64, bool)
}

// n builds a band from a low and a high; one argument means a single-valued cell.
func n(lo float64, hi ...float64) Band {
	if len(hi) == 0 {
		return Band{Lo: lo, Hi: lo}
	}
	return Band{Lo: lo, Hi: hi[0]}
}

// na is 10.2's "n/a": prize:enemy in the tutorial column, where the two houses have no
// enemies at all and the ratio has no denominator.
var na = Band{NA: true}

// ratio is the measurement behind every per-room row: a division that reports itself
// unmeasurable on the empty house rather than returning a NaN nobody asked for.
func ratio(num func(Profile) int) func(Profile) (float64, bool) {
	return func(p Profile) (float64, bool) {
		if p.Rooms == 0 {
			return 0, false
		}
		return float64(num(p)) / float64(p.Rooms), true
	}
}

// count is the measurement behind every whole-number row.
func count(f func(Profile) int) func(Profile) (float64, bool) {
	return func(p Profile) (float64, bool) { return float64(f(p)), true }
}

// pct is the two rows 10.2 writes with a % sign.
func pct(f func(Profile) int) func(Profile) (float64, bool) {
	r := ratio(f)
	return func(p Profile) (float64, bool) {
		v, ok := r(p)
		return 100 * v, ok
	}
}

// rows is 10.2's table, in 10.2's order. The order is not cosmetic: it goes shape, then
// contents, then difficulty, then the two totals, which is the order an author fills a
// house in, and `house stats` prints it unsorted for that reason.
//
// The one cell that is not a transcription is **tutorial's empty-rooms band**. 10.2
// writes "~20 %", which is a midpoint and not a range, and a band has to have ends. It
// is widened to 15-25 -- the same +-5 the neighbouring small column spans either side
// of its own midpoint -- and TestTheTierTableMatchesTheDocument knows about this one
// exception by name so that the rest of the table stays a strict comparison.
var rows = [...]Row{
	{"real rooms", UnitCount, [5]Band{n(35, 45), n(45, 85), n(85, 140), n(175, 303), n(383, 531)},
		count(func(p Profile) int { return p.Rooms })},
	{"occupied floors", UnitCount, [5]Band{n(6, 7), n(5, 14), n(10, 14), n(12, 20), n(18, 35)},
		count(func(p Profile) int { return p.Floors })},
	{"occupied suites", UnitCount, [5]Band{n(9, 10), n(13, 20), n(10, 42), n(27, 39), n(45, 101)},
		count(func(p Profile) int { return p.Suites })},
	{"grid density", UnitRatio, [5]Band{n(0.6, 0.65), n(0.2, 0.65), n(0.5, 1.0), n(0.44, 0.57), n(0.22, 0.64)},
		func(p Profile) (float64, bool) {
			if p.Floors*p.Suites == 0 {
				return 0, false
			}
			return p.Density(), true
		}},
	{"total objects", UnitCount, [5]Band{n(78, 138), n(300, 620), n(570, 1100), n(1280, 1820), n(2990, 5840)},
		count(func(p Profile) int { return p.Objects })},
	{"objects/room", UnitRatio, [5]Band{n(2.2, 3.1), n(7, 19), n(5.2, 8.2), n(6.0, 7.5), n(6.6, 14.5)},
		ratio(func(p Profile) int { return p.Objects })},
	{"empty rooms", UnitPercent, [5]Band{n(15, 25), n(10, 25), n(15, 25), n(20, 25), n(20, 30)},
		pct(func(p Profile) int { return p.Empty })},
	{"rooms at the 24 ceiling", UnitCount, [5]Band{n(0), n(0, 6), n(0, 3), n(5, 22), n(4, 77)},
		count(func(p Profile) int { return p.AtCeiling })},
	{"enemies/room", UnitRatio, [5]Band{n(0.0, 0.1), n(0.5, 1.3), n(0.3, 1.0), n(0.2, 0.8), n(0.25, 0.73)},
		ratio(func(p Profile) int { return p.Enemies })},
	{"prizes/room", UnitRatio, [5]Band{n(0.03, 0.29), n(0.4, 0.95), n(0.5, 1.4), n(0.37, 0.73), n(0.42, 1.10)},
		ratio(func(p Profile) int { return p.Prizes })},
	{"prize:enemy", UnitRatio, [5]Band{na, n(0.7, 1.5), n(0.5, 2.4), n(0.7, 2.7), n(1.1, 2.2)},
		func(p Profile) (float64, bool) {
			if p.Enemies == 0 {
				return 0, false
			}
			return float64(p.Prizes) / float64(p.Enemies), true
		}},
	{"stars", UnitCount, [5]Band{n(1), n(1, 4), n(1, 6), n(3, 5), n(1, 6)},
		count(func(p Profile) int { return p.Stars })},
	{"batteries", UnitCount, [5]Band{n(0, 1), n(1, 6), n(1, 8), n(0, 15), n(2, 27)},
		count(func(p Profile) int { return p.Batteries })},
	{"rubber bands", UnitCount, [5]Band{n(0, 1), n(0, 4), n(1, 6), n(1, 10), n(8, 32)},
		count(func(p Profile) int { return p.Bands })},
	{"dark rooms", UnitPercent, [5]Band{n(0), n(0, 4), n(0, 8), n(0, 18), n(0, 11)},
		pct(func(p Profile) int { return p.Dark })},
	{"BFS eccentricity", UnitCount, [5]Band{n(11, 15), n(2, 23), n(10, 39), n(24, 46), n(24, 59)},
		func(p Profile) (float64, bool) {
			// -1 is walk's "no room to start from", which is not a length.
			if p.Eccentricity < 0 {
				return 0, false
			}
			return float64(p.Eccentricity), true
		}},
	{"distinct what codes", UnitCount, [5]Band{n(15, 48), n(50, 69), n(50, 103), n(85, 110), n(60, 114)},
		count(func(p Profile) int { return p.Kinds })},
	{"total points", UnitCount, [5]Band{n(8500, 12500), n(7900, 45500), n(30700, 58900), n(44600, 86300), n(73000, 141400)},
		count(func(p Profile) int { return int(p.Points) })},
}

// Rows is 10.2's table. A copy, because a caller holding the package's own array could
// edit the bands out from under every other caller -- and one of the callers is a test
// that asserts they match the document.
func Rows() []Row { return append([]Row(nil), rows[:]...) }

// Miss is one row a house does not satisfy, with enough in it to print a line without
// going back to the table.
type Miss struct {
	Row   Row
	Value float64
	// Measured is false when the row could not be evaluated -- prize:enemy on a house
	// with no enemies. Such a row is reported as a miss only when the tier does give it
	// a band, since otherwise there is nothing to have missed.
	Measured bool
}

// String is the one-line form: `objects/room 4.40, tutorial wants 2.2-3.1`.
func (m Miss) String() string {
	if !m.Measured {
		return fmt.Sprintf("%s cannot be measured for this house", m.Row.Target)
	}
	return fmt.Sprintf("%s %s", m.Row.Target, m.Row.Unit.Format(m.Value))
}

// Check measures a house against one tier and returns the rows it misses, in 10.2's
// order.
//
// An empty result means every row lands inside its band, which exactly one of the 22
// shipped houses manages for its own tier: Leviathan, which set eight of epic's own bounds.
// The other 21 are outside at least one band of the tier the document assigns them to
// (TestTheBandsExcludeTheHousesTheyWereMeasuredFrom, docs/IMPROVEMENTS.md 4.25). So this is
// a report and not a gate, and a caller that turns a non-empty result into an exit status is
// choosing to be stricter than the corpus.
func Check(p Profile, t Tier) []Miss {
	if t < 0 || int(t) >= len(TierNames) {
		return nil
	}
	var out []Miss
	for _, r := range rows {
		b := r.Bands[t]
		if b.NA {
			// No target at this tier, so there is nothing to miss. A tutorial house
			// that does have enemies is not thereby wrong; it is outside what the two
			// tutorial-sized originals measured.
			continue
		}
		v, ok := r.Measure(p)
		if !ok {
			out = append(out, Miss{Row: r, Measured: false})
			continue
		}
		if !b.Contains(v) {
			out = append(out, Miss{Row: r, Value: v, Measured: true})
		}
	}
	return out
}
