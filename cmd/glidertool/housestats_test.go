package main

// `house stats` at the command level.
//
// The measurement is tested in internal/profile: 10.2's table is checked cell by cell
// against the markdown it was transcribed from, the room graph is checked against the
// probe's figures for all 22 shipped houses and against hand-built one- and two-room
// houses, and the tier bands are checked against the two houses in levels/. None of that
// is repeated here. What is left is the part a printer can get wrong on its own -- a
// verdict column that disagrees with the package it is reporting, a flag that looks like it
// is checking something and is not, and "n/a" printed as 0.
//
// Every test runs with -no-assets. The eight houses that ship 'bnds' resources need the
// extracted tree to be walked correctly, and the fixture is not one of them: both its rooms
// carry a `bounds` field of their own, so their four openings are in the house file and the
// walk is exact with no art at all. A tool test that skips on a fresh checkout is a tool
// test nobody runs.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/profile"
	"github.com/bwenstar/gliderGo/internal/render"
)

// stats runs `house stats` over the package fixture and returns what it printed.
//
// The fixture is two rooms that cannot reach each other: room 1 is the start and opens left
// onto (floor 0, suite 60), where there is no room -- room 0 is at floor -1 -- and its one
// kInvisTrans is the dangling link houselint_test.go uses, so it is not an edge either. So
// the walk reports 1 of 2 reachable with room 0 unreachable, which is what makes this a
// fixture the graph rows can be read on.
func stats(t *testing.T, args ...string) (string, error) {
	t.Helper()
	_, housePath := writeFixture(t)
	return captureStdout(t, func() error {
		return houseStats(append(append([]string{"-no-assets"}, args...), housePath))
	})
}

// fixtureProfile is the same measurement the command makes, for the tests that compare the
// printed table against the package rather than against a hard-coded number.
func fixtureProfile(t *testing.T) profile.Profile {
	t.Helper()
	_, housePath := writeFixture(t)
	h, err := house.LoadFile(housePath)
	if err != nil {
		t.Fatal(err)
	}
	return profile.Measure(h, render.NewAssets(nil))
}

// statsLine finds the table row for one of 10.2's targets.
//
// Prefix-matched on the target name, which is unambiguous because no row's name is a prefix
// of another's -- "real rooms" and "rooms at the 24 ceiling" are the near miss.
func statsLine(t *testing.T, out, target string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, target+" ") {
			return line
		}
	}
	t.Fatalf("no %q row in:\n%s", target, out)
	return ""
}

// TestHouseStatsPrintsAllEighteenRows is the whole point of the command: a house author
// wants the numbers, and a table missing a row is a row they will not think about.
func TestHouseStatsPrintsAllEighteenRows(t *testing.T) {
	out, err := stats(t)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range profile.Rows() {
		statsLine(t, out, r.Target)
	}
	// The two graph numbers are not a row between them -- eccentricity is, reachable is
	// not -- and the reachable line has to name the room it counted from, because a count
	// from the wrong room is worse than no count.
	if !strings.Contains(out, "reachable: 1 of 2 rooms from 1 ") {
		t.Errorf("the reachable line does not name the start room:\n%s", out)
	}
}

// TestHouseStatsVerdictsAgreeWithTheProfilePackage holds the printer to the package.
//
// The verdict column could have re-tested each band where it prints it, and an early draft
// did. Two implementations of "is this value inside this band" is one more than the number
// of places the n/a rule can be stated: a tier whose cell is n/a has no target and so
// nothing to miss, and a row that cannot be *measured* is a miss only if the tier does give
// it a band. The command asks profile.Check instead, and this is the test that would catch
// it going back.
func TestHouseStatsVerdictsAgreeWithTheProfilePackage(t *testing.T) {
	p := fixtureProfile(t)
	for _, tier := range []profile.Tier{profile.Tutorial, profile.Small, profile.Epic} {
		t.Run(tier.String(), func(t *testing.T) {
			out, err := stats(t, "-tier", tier.String())
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "target") || !strings.Contains(out, tier.String()) {
				t.Errorf("the header does not name the tier it compared against:\n%s", out)
			}
			missed := map[string]bool{}
			for _, m := range profile.Check(p, tier) {
				missed[m.Row.Target] = true
			}
			for _, r := range profile.Rows() {
				want := "ok"
				switch {
				case r.Bands[tier].NA:
					want = "-"
				case missed[r.Target]:
					want = "miss"
				}
				if line := statsLine(t, out, r.Target); !strings.HasSuffix(line, want) {
					t.Errorf("%s reads %q; profile.Check says %q", r.Target, line, want)
				}
			}
		})
	}
}

// TestHouseStatsSaysNAAndNotZero is the one formatting rule worth a test of its own.
//
// The fixture has no enemies, so prize:enemy has no denominator. Printing 0 there would
// read as "this house has no prizes", which is the opposite of true -- it has a star -- and
// it is 10.2's own answer for the tutorial column, written "n/a".
func TestHouseStatsSaysNAAndNotZero(t *testing.T) {
	out, err := stats(t)
	if err != nil {
		t.Fatal(err)
	}
	if line := statsLine(t, out, "prize:enemy"); !strings.Contains(line, "n/a") {
		t.Errorf("prize:enemy on a house with no enemies reads %q, want n/a", line)
	}
}

// TestHouseStatsFailNeedsATier pins the exit-status contract, which is the only part of
// this command a CI step can depend on.
//
// -fail with no -tier is refused rather than accepted and ignored. Accepting it would make
// a step that looks like it is checking something and never fails, which is worse than not
// having the flag: the house drifts and the pipeline stays green.
func TestHouseStatsFailNeedsATier(t *testing.T) {
	if _, err := stats(t, "-fail"); err == nil {
		t.Error("-fail with no -tier succeeded, so a CI step could check nothing and pass")
	}
	// The fixture is two rooms, so it misses every tier's real-rooms band by a mile.
	if _, err := stats(t, "-tier", "tutorial", "-fail"); err == nil {
		t.Error("-fail passed a two-room house against the 35-45 room tutorial band")
	}
	// And without -fail the same run reports and exits zero, because 10.2's bands are a
	// measurement and not a rule -- the command says so in its own output.
	out, err := stats(t, "-tier", "tutorial")
	if err != nil {
		t.Errorf("-tier without -fail returned an error: %v", err)
	}
	if !strings.Contains(out, "not rules") {
		t.Errorf("a tier comparison must say what a band is:\n%s", out)
	}
	if _, err := stats(t, "-tier", "enormous"); err == nil {
		t.Error("-tier accepted a name that is not one of 10.2's five columns")
	}
}

// TestHouseStatsNamesTheRoomsBehindTheCounts is the difference between a number an author
// can act on and one they cannot.
//
// Both lists are index-and-name, which is RoomRef's doing: the shipped houses reuse names
// freely -- Titanic's 43 unreachable rooms are 24 "Murmur", 9 "Dirt" and 8 "Pitch" -- so a
// list of names alone says something is wrong and gives no way to find it.
func TestHouseStatsNamesTheRoomsBehindTheCounts(t *testing.T) {
	quiet, err := stats(t)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(quiet, "-rooms names them") {
		t.Errorf("a house with an unreachable room does not offer the list:\n%s", quiet)
	}
	if strings.Contains(quiet, `"Cellar"`) {
		t.Errorf("the list printed without -rooms:\n%s", quiet)
	}

	loud, err := stats(t, "-rooms")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(loud, "unreachable (1):") {
		t.Errorf("-rooms did not head the list with its length:\n%s", loud)
	}
	if !strings.Contains(loud, `0 "Cellar"`) {
		t.Errorf("-rooms named no room, or named it without its index:\n%s", loud)
	}
}

// TestHouseStatsSummaryIsOneLinePerHouse is the shape that makes
// `house stats -summary assets/extracted/houses/*.house` a readable answer to "how do the
// 22 compare", and the columns are 3.6's so the output can be read against the document.
func TestHouseStatsSummaryIsOneLinePerHouse(t *testing.T) {
	_, housePath := writeFixture(t)
	out, err := captureStdout(t, func() error {
		return houseStats([]string{"-no-assets", "-summary", housePath, housePath})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"reach/real", "ecc", "points"} {
		if !strings.Contains(out, want) {
			t.Errorf("the summary header has no %q column:\n%s", want, out)
		}
	}
	if n := strings.Count(out, "fixture"); n != 2 {
		t.Errorf("two houses produced %d summary lines:\n%s", n, out)
	}
	// One header for the run and not one per house, which is what makes it a table.
	if n := strings.Count(out, "reach/real"); n != 1 {
		t.Errorf("the summary header printed %d times:\n%s", n, out)
	}
}

// forkBoundFixture is one room with a house's own background and no `bounds` field, which
// is the only room that needs the art tree to know which of its sides are open: with no
// 'bnds' resource to read, GetOriginalBounding answers 0 and the room is sealed
// (internal/game/room.go:139-150). 155 of the corpus's 4,070 rooms are like this.
const forkBoundFixture = `format 1

house
    version    0x0200
    timestamp  743682113
    initial    107 49
    firstroom  0

room 0 "Needs A Resource"
    background 3000
    floor      0
    suite      60
    object 0   kStar  at 137 310  length 0  points 0  state 1  initial 1
`

// TestHouseStatsSaysWhatNoAssetsCostThisHouse is the difference between a caveat and a
// disclaimer.
//
// The closing line used to warn about the two graph rows whenever -no-assets was given,
// which is wrong for fifteen of the 22 shipped houses and for every house the port ships:
// a room that carries its own `bounds` field needs no resource, so the walk with no art is
// the same walk. A note that fires on every house is a note nobody reads, and this one was
// telling most of its readers to discount a number that was exact.
func TestHouseStatsSaysWhatNoAssetsCostThisHouse(t *testing.T) {
	// The package fixture's two rooms both carry a bounds field, so nothing is lost.
	exact, err := stats(t)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(exact, "costs nothing here") {
		t.Errorf("a house whose rooms all carry bounds got the caveat anyway:\n%s", exact)
	}

	h, err := house.ParseText(strings.NewReader(forkBoundFixture))
	if err != nil {
		t.Fatalf("fixture does not parse: %v", err)
	}
	path := filepath.Join(t.TempDir(), "forkbound.house")
	if err := h.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	costly, err := captureStdout(t, func() error {
		return houseStats([]string{"-no-assets", path})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(costly, "1 room(s) read their openings from a 'bnds' resource") {
		t.Errorf("a room that needs a resource got no caveat, or no count:\n%s", costly)
	}
}

func TestHouseStatsNeedsAFile(t *testing.T) {
	if err := houseStats(nil); err == nil {
		t.Error("house stats with no arguments succeeded")
	}
}
