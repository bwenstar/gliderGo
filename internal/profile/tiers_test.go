package profile

import (
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestTheTierTableMatchesTheDocument parses 10.2 out of the analysis document and
// compares all ninety cells against tiers.go.
//
// tiers.go is a transcription of a table somebody measured, and a transcription with no
// check on it is a copy that will be corrected in one place. This is the check. It reads
// the markdown rather than a generated file on purpose: 10.2 is what an author reads, so
// 10.2 has to be the thing the tool agrees with.
//
// One cell is knowingly not a transcription -- see widened below -- and it is named here
// rather than tolerated by a loose comparison, so that the other eighty-nine are exact.
func TestTheTierTableMatchesTheDocument(t *testing.T) {
	raw, err := os.ReadFile(sectionDoc)
	if err != nil {
		t.Skipf("no analysis document at %s: %v", sectionDoc, err)
	}

	// The one deliberate divergence: 10.2's tutorial empty-rooms cell is "~20 %", a
	// midpoint, and a band needs two ends. rows widens it to 15-25.
	widened := map[string]Band{"empty rooms/tutorial": n(15, 25)}

	doc := parseSection102(t, string(raw))
	got := Rows()

	if len(doc) != len(got) {
		t.Fatalf("10.2 has %d rows, tiers.go has %d", len(doc), len(got))
	}
	for i, want := range doc {
		r := got[i]
		if r.Target != want.target {
			t.Errorf("row %d: tiers.go says %q, 10.2 says %q -- the two tables are in "+
				"different orders, which makes every comparison below meaningless",
				i, r.Target, want.target)
			continue
		}
		for tier := range TierNames {
			key := want.target + "/" + TierNames[tier]
			exp := want.bands[tier]
			if w, ok := widened[key]; ok {
				exp = w
			}
			if r.Bands[tier] != exp {
				t.Errorf("%s: tiers.go says %v, 10.2 says %v", key, r.Bands[tier], exp)
			}
		}
	}
	for k := range widened {
		if !strings.Contains(string(raw), "~20 %") {
			t.Errorf("tiers.go widens %s because 10.2 writes a midpoint, but 10.2 no "+
				"longer has one -- transcribe the range it has now", k)
		}
	}
}

// docRow is one parsed row of 10.2.
type docRow struct {
	target string
	bands  [len(TierNames)]Band
}

// parseSection102 reads the five-column table under "### 10.2".
func parseSection102(t *testing.T, doc string) []docRow {
	t.Helper()
	lines := strings.Split(doc, "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "### 10.2") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("%s has no section 10.2; this test reads its table", sectionDoc)
	}

	var out []docRow
	for _, l := range lines[start:] {
		if strings.HasPrefix(l, "### 10.3") || strings.HasPrefix(l, "## ") {
			break
		}
		if !strings.HasPrefix(l, "| ") {
			continue
		}
		cells := strings.Split(strings.Trim(l, "|"), "|")
		if len(cells) != 1+len(TierNames) {
			continue
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(tableMarkup.ReplaceAllString(cells[i], ""))
		}
		if cells[0] == "target" {
			// The heading row doubles as a check that the columns are the tiers we
			// think they are, in the order tiers.go indexes them by.
			for i, name := range TierNames {
				if cells[1+i] != name {
					t.Fatalf("10.2 column %d is %q, tiers.go calls that tier %q",
						i, cells[1+i], name)
				}
			}
			continue
		}
		if strings.HasPrefix(cells[0], "---") {
			continue
		}
		r := docRow{target: cells[0]}
		for i := range TierNames {
			r.bands[i] = parseBand(t, cells[0], cells[1+i])
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		t.Fatalf("%s section 10.2 has no table rows", sectionDoc)
	}
	return out
}

// parseBand reads one cell: "35-45", "0", "0-4 %", "~20 %" or "n/a".
func parseBand(t *testing.T, target, cell string) Band {
	t.Helper()
	cell = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(cell), "%"))
	if cell == "n/a" {
		return na
	}
	if strings.HasPrefix(cell, "~") {
		// A midpoint, not a range. Returned as a zero-width band so the comparison in
		// the caller fails unless that cell is in `widened` by name.
		v := mustFloat(t, target, strings.TrimPrefix(cell, "~"))
		return n(v)
	}
	if lo, hi, ok := strings.Cut(cell, "-"); ok {
		return Band{Lo: mustFloat(t, target, lo), Hi: mustFloat(t, target, hi)}
	}
	return n(mustFloat(t, target, cell))
}

func mustFloat(t *testing.T, target, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		t.Fatalf("10.2 row %q: %q is not a number: %v", target, s, err)
	}
	return v
}

// TestEveryTierRowCanBeMeasured drives all eighteen rows against a real house, because a
// table of bands with no measurement behind it would pass the test above and be useless.
//
// The check is not "the values are right" -- 3.6's corpus pin does that for eight of the
// rows -- but "every row answers, and answers in its own unit". A row whose Measure
// returned a raw count where the band is a percentage would sail through every other
// test in this package.
func TestEveryTierRowCanBeMeasured(t *testing.T) {
	p := corpusProfile(t, "Slumberland")

	for _, r := range Rows() {
		v, ok := r.Measure(p)
		if !ok {
			t.Errorf("%s: cannot be measured for a 383-room house", r.Target)
			continue
		}
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Errorf("%s: measured %v", r.Target, v)
		}
		if r.Unit == UnitPercent && (v < 0 || v > 100) {
			t.Errorf("%s: %v is not a percentage -- the row's Measure is probably "+
				"returning the ratio or the count", r.Target, v)
		}
		if r.Unit == UnitCount && v != math.Trunc(v) {
			t.Errorf("%s: %v is not a whole number", r.Target, v)
		}
	}
}

// TestSlumberlandIsAnEpicHouseByItsOwnFigures is the end-to-end use of the table: the
// largest shipped house, checked against the tier its own room count puts it in.
//
// It does not demand zero misses, and that is the point of the test rather than a
// weakness in it. Only one shipped house satisfies its own tier on every row -- 10.2's
// bands are per-column extremes drawn from different houses, so hitting all eighteen means
// being the extreme house in eighteen ways at once, which Leviathan nearly is and nobody
// else is (TestTheBandsExcludeTheHousesTheyWereMeasuredFrom counts them;
// docs/IMPROVEMENTS.md 4.18 and 4.25 are the entries). What is asserted is that the rows
// this house misses are few and named, so that a change to Measure or to the table shows up
// here as a house drifting rather than as silence.
func TestSlumberlandIsAnEpicHouseByItsOwnFigures(t *testing.T) {
	p := corpusProfile(t, "Slumberland")

	misses := Check(p, Epic)
	for _, m := range misses {
		t.Logf("misses %s (epic wants %s)", m, m.Row.Bands[Epic].String(m.Row.Unit))
	}
	// Exactly one, and it is worth naming rather than counting: Slumberland is 13.6 %
	// empty and epic's band is 20-30 %, so **the largest shipped house is outside the
	// empty-rooms band of its own tier**. That is 4.18's complaint in one line -- a
	// column assembled from four houses' extremes need not contain any of the four -- and
	// it is why Check reports and does not judge.
	if len(misses) != 1 || misses[0].Row.Target != "empty rooms" {
		got := make([]string, len(misses))
		for i, m := range misses {
			got[i] = m.Row.Target
		}
		t.Errorf("Slumberland misses %v of the epic rows; it used to miss only "+
			"[empty rooms], so either the table moved or Measure did", got)
	}

	// And the row that could not exist before the room graph did.
	ecc := rowNamed(t, "BFS eccentricity")
	if !ecc.Bands[Epic].Contains(float64(p.Eccentricity)) {
		t.Errorf("BFS eccentricity %d is outside epic's %s, and Slumberland is the house "+
			"the top of that band was measured from",
			p.Eccentricity, ecc.Bands[Epic].String(ecc.Unit))
	}
}

// TestTheBandsExcludeTheHousesTheyWereMeasuredFrom is the whole corpus read against the
// whole table, and it is the test that keeps one sentence of `house stats` honest.
//
// The command prints, whenever -tier is given, that "21 of the 22 shipped houses are
// themselves outside a band of their own tier". An earlier draft of that line said *no*
// shipped house is inside all eighteen, which was a guess made from three houses and is
// wrong: Leviathan is inside all eighteen of epic's bands, which it manages by being the
// house that set eight of them. A sentence a tool prints to an author about the corpus has
// to be measured over the corpus, so it is measured here.
//
// Three mechanisms produce the 21, and all three are properties of the table rather than of
// the houses (docs/IMPROVEMENTS.md 4.25):
//
//   - **An edge rounded inward excludes the house it was taken from.** Empty House holds
//     one prize in 35 rooms, 0.0286 a room, and tutorial's prizes/room band opens at 0.03.
//     SpacePods is 14.53 objects a room against an epic band closing at 14.5. Art Museum
//     holds 569 objects against a medium band opening at 570.
//   - **The empty-rooms row is not a spread at all.** Eighteen of the 22 are outside it.
//     Its five cells run 10-30 % while the houses run 0 % to 53.3 %, and the tutorial cell
//     is written "~20 %" -- which is 5.4's corpus-wide figure, 840 of 4,070 rooms, not
//     anything about tutorial-sized houses.
//   - **10.2's room bands disagree with 8.3's own grouping.** 8.3 puts California or Bust!
//     (16 rooms) and Fun House (43) in a tier it describes as 16-65 rooms; 10.2's small
//     column says 45-85, so both houses are outside the room band of the tier the document
//     assigns them to. Same for Sampler at 2 rooms in a tutorial column of 35-45.
//
// The numbers are asserted exactly rather than as "most". A loose assertion here would
// survive Measure changing under it, which is the failure this test exists to catch.
func TestTheBandsExcludeTheHousesTheyWereMeasuredFrom(t *testing.T) {
	grouped := readSection83Grouping(t)

	inside, missedEmpty, total := []string{}, 0, 0
	for tier, names := range grouped {
		for _, name := range names {
			total++
			p := corpusProfile(t, name)
			misses := Check(p, Tier(tier))
			if len(misses) == 0 {
				inside = append(inside, name)
			}
			for _, m := range misses {
				if m.Row.Target == "empty rooms" {
					missedEmpty++
				}
			}
		}
	}

	if total != 22 {
		t.Fatalf("8.3's grouping accounts for %d houses, not 22", total)
	}
	if len(inside) != 1 || inside[0] != "Leviathan" {
		t.Errorf("the houses inside all eighteen of their own tier's bands are %v; "+
			"`house stats` prints \"21 of the 22\" and internal/profile's own comments "+
			"say the one is Leviathan", inside)
	}
	if missedEmpty != 18 {
		t.Errorf("%d houses are outside their tier's empty-rooms band, not 18 -- if that "+
			"is now a smaller number then 10.2's empty-rooms row has been rewritten and "+
			"the widened cell in TestTheTierTableMatchesTheDocument wants rereading",
			missedEmpty)
	}

	// The third mechanism, checked against the two documents rather than against houses:
	// three of 8.3's five room ranges are exactly 10.2's real-rooms band and two are not.
	rooms := rowNamed(t, "real rooms")
	for tier, want := range readSection83Rooms(t) {
		got := rooms.Bands[tier]
		agree := got == want
		switch Tier(tier) {
		case Medium, Large, Epic:
			if !agree {
				t.Errorf("%s: 8.3 groups %s rooms, 10.2's real-rooms band is %s -- these "+
					"two used to agree, so one of the documents has moved",
					Tier(tier), want.String(UnitCount), got.String(UnitCount))
			}
		default:
			if agree {
				t.Errorf("%s: 8.3 and 10.2 now agree at %s. That is the fix 4.25 asks "+
					"for; delete this arm and the entry's third bullet",
					Tier(tier), got.String(UnitCount))
			}
		}
	}
}

// readSection83Grouping reads the "Grouping by intent" table's houses column, tier by tier.
//
// The mapping from that table's rows to 10.2's columns is by *order* and cannot be by name:
// 8.3 calls them "template / tech demo", "showcase / short", "medium campaign", "large
// campaign" and "epic", and 10.2's headings are tutorial, small, medium, large, epic. Five
// rows in increasing size against five columns in increasing size is the only reading
// available, and the room-range comparison at the end of the test above is what holds it --
// three of the five ranges are identical to 10.2's own room band, which no accidental
// alignment would produce.
func readSection83Grouping(t *testing.T) [len(TierNames)][]string {
	t.Helper()
	var out [len(TierNames)][]string
	for i, cells := range readSection83(t) {
		if i >= len(out) {
			t.Fatalf("8.3's grouping has more than %d tiers", len(out))
		}
		for _, h := range strings.Split(cells[2], ",") {
			// "Sampler (2)" -- the room count is repeated in three of the five rows.
			if name, _, found := strings.Cut(strings.TrimSpace(h), " ("); found {
				out[i] = append(out[i], name)
			} else {
				out[i] = append(out[i], strings.TrimSpace(h))
			}
		}
	}
	return out
}

// readSection83Rooms reads the same table's rooms column as a band per tier.
func readSection83Rooms(t *testing.T) [len(TierNames)]Band {
	t.Helper()
	var out [len(TierNames)]Band
	for i, cells := range readSection83(t) {
		if i >= len(out) {
			t.Fatalf("8.3's grouping has more than %d tiers", len(out))
		}
		out[i] = parseBand(t, "8.3 rooms", cells[1])
	}
	return out
}

// readSection83 returns the four cells of each row of 8.3's grouping table.
//
// 8.3 holds two tables -- the per-house profile first, then the grouping -- so the row
// width is what tells them apart: four columns against nine.
func readSection83(t *testing.T) [][]string {
	t.Helper()
	raw, err := os.ReadFile(sectionDoc)
	if err != nil {
		t.Skipf("no analysis document at %s: %v", sectionDoc, err)
	}
	lines := strings.Split(string(raw), "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "### 8.3") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("%s has no section 8.3; this test reads its grouping table", sectionDoc)
	}

	var out [][]string
	for _, l := range lines[start:] {
		if strings.HasPrefix(l, "### 8.4") || strings.HasPrefix(l, "## ") {
			break
		}
		if !strings.HasPrefix(l, "| ") {
			continue
		}
		cells := strings.Split(strings.Trim(l, "|"), "|")
		if len(cells) != 4 {
			continue
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(tableMarkup.ReplaceAllString(cells[i], ""))
		}
		if cells[0] == "tier" || strings.HasPrefix(cells[0], "---") {
			continue
		}
		out = append(out, cells)
	}
	if len(out) != len(TierNames) {
		t.Fatalf("8.3's grouping table has %d tiers, tiers.go has %d",
			len(out), len(TierNames))
	}
	return out
}

// rowNamed finds one row by its label, so a test can name the row it means rather than
// its index -- an index that silently means a different row after an insertion is exactly
// the kind of thing this package exists to stop.
func rowNamed(t *testing.T, target string) Row {
	t.Helper()
	for _, r := range Rows() {
		if r.Target == target {
			return r
		}
	}
	t.Fatalf("10.2 has no row called %q", target)
	return Row{}
}

// TestATierIsResolvedByName covers the flag parsing `house stats -tier` does.
func TestATierIsResolvedByName(t *testing.T) {
	for i, name := range TierNames {
		for _, spelling := range []string{name, strings.ToUpper(name), " " + name + " "} {
			got, ok := ParseTier(spelling)
			if !ok || got != Tier(i) {
				t.Errorf("ParseTier(%q) = %v, %v; want %v, true", spelling, got, ok, Tier(i))
			}
			if got.String() != name {
				t.Errorf("Tier(%d).String() = %q, want %q", i, got.String(), name)
			}
		}
	}
	for _, bad := range []string{"", "huge", "tutorials", "0"} {
		if _, ok := ParseTier(bad); ok {
			t.Errorf("ParseTier(%q) accepted a tier that does not exist", bad)
		}
	}
	if got := Tier(99).String(); got != "tier(99)" {
		t.Errorf("an out-of-range tier prints %q", got)
	}
	if Check(Profile{}, Tier(99)) != nil {
		t.Error("Check accepted an out-of-range tier")
	}
}

// TestABandPrintsTheWayTheDocumentWritesIt pins the formatting, because every number
// `house stats` shows an author goes through it and a band that prints "0.60-0.65" beside
// a document that says "0.6-0.65" invites them to think the tool has a different table.
func TestABandPrintsTheWayTheDocumentWritesIt(t *testing.T) {
	cases := []struct {
		b    Band
		u    Unit
		want string
	}{
		{n(35, 45), UnitCount, "35-45"},
		{n(0), UnitCount, "0"},
		{n(1), UnitCount, "1"},
		{n(0.6, 0.65), UnitRatio, "0.6-0.65"},
		{n(6.0, 7.5), UnitRatio, "6-7.5"},
		{n(0.42, 1.10), UnitRatio, "0.42-1.1"},
		{n(0, 4), UnitPercent, "0-4 %"},
		{n(0), UnitPercent, "0 %"},
		{na, UnitRatio, "n/a"},
	}
	for _, c := range cases {
		if got := c.b.String(c.u); got != c.want {
			t.Errorf("Band{%v,%v}.String(%v) = %q, want %q", c.b.Lo, c.b.Hi, c.u, got, c.want)
		}
	}

	// Inclusive at both ends: epic's room band opens at 383 because Slumberland is 383.
	if !n(383, 531).Contains(383) || !n(383, 531).Contains(531) {
		t.Error("a band has to include its own ends, which are corpus houses")
	}
	if n(383, 531).Contains(382) || na.Contains(0) {
		t.Error("a band is including something it should not")
	}
}
