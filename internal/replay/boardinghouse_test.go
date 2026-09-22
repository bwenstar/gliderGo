package replay_test

// The acceptance test for the port's second house.
//
// Open House establishes that a house can be written to a column of
// docs/analysis/original-houses.md 10.2 and then held there by a test. What it cannot
// establish on its own is whether that was a method or a coincidence, because one house
// fitted to one column is also what you get by drawing a house and then picking the
// column it happens to land in. So this one is written to the **small** column -- 45 to
// 85 rooms -- whose bands differ from the tutorial's in kind and not just in width: 7 to
// 19 objects per room against 2.2 to 3.1, enemies where the tutorial tier permits almost
// none, and up to 4 % dark rooms where the tutorial tier permits none at all. The two
// houses are measured by the same code (profile_test.go) and share no layout.
//
// The structure here follows openhouse_test.go exactly, including building the house
// from the checked-in text rather than loading the build product, for the reasons that
// file gives at length: the text is what is under version control, so a stale
// assets/levels/ copy cannot make a broken source pass, and a fresh clone needs no
// `make levels` before `make test` says something.
//
// # What a failure means
//
// A failure in TestBoardingHouseCanBeFinished is a flight-model change, not a house bug,
// unless the house changed -- in which case the diff says so. The route is
// 7 -> 8 -> 9 -> 20 -> 21 -> 30 -> 31 -> 38 -> 39 -> 45 -> 46 -> 49 and each pair is one
// staircase or one east crossing, so the room the run stopped in names the leg. The
// script's own header holds the four rules that regenerate its keystrokes.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/replay"
)

// boardingHouseSource is the authored text, relative to this package. The space in the
// name is the naming scheme, not an oversight -- see openhouse_test.go.
const boardingHouseSource = "../../levels/Boarding House.house.txt"

func boardingHouseDir(t *testing.T) string {
	t.Helper()
	h, err := house.ParseTextFile(boardingHouseSource)
	if err != nil {
		t.Fatalf("parse %s: %v", boardingHouseSource, err)
	}
	raw, err := h.Save()
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Boarding House.house"), raw, 0o666); err != nil {
		t.Fatalf("write house: %v", err)
	}
	return dir
}

func TestBoardingHouseCanBeFinished(t *testing.T) {
	dir := boardingHouseDir(t)

	s := script(t, "boarding-house.script")
	s.HouseDir = dir // after localAssets, which points it at the extracted tree

	res, err := replay.Run(s)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	room := -1
	if n := len(res.Samples); n > 0 {
		room = int(res.Samples[n-1].Room)
	}

	if !res.GameOver {
		t.Errorf("the run did not finish: gameOver=false after %d frames, in room %d "+
			"with %d star(s) left and %d glider(s)",
			res.Frames, room, res.StarsLeft, res.Mortals)
	}
	if res.StarsLeft != 0 {
		t.Errorf("stars left = %d, want 0: the route takes all three, so any remainder "+
			"means a leg diverged before the one that was missed", res.StarsLeft)
	}
	if want := int16(2); res.Mortals != want {
		t.Errorf("gliders = %d, want %d: nothing on this route can kill the glider, so a "+
			"loss here is a collision that did not used to happen", res.Mortals, want)
	}
	// 100 a room for the eleven entered after the first, and 5,000 for each of the three
	// stars. A prize picked up in passing or a room visited twice would move this, which is
	// worth hearing about even though neither would be a failure of the house.
	if want := int32(16100); res.Score != want {
		t.Errorf("score = %d, want %d (11 rooms at 100 plus three 5,000 stars)",
			res.Score, want)
	}
	// The three stars sit at v 14 directly over a vent, so they are collected at the top of
	// a ride the route was making anyway, and the last one ends the house. 1,447 frames is
	// what that takes; `frames 1500` in the script leaves 53 spare.
	if res.Frames > 1500 {
		t.Errorf("the run used all %d frames: the script budgets 1500 and the route "+
			"finishes in 1447, so this means it is no longer finishing", res.Frames)
	}
	t.Logf("finished in %d frames, room %d, score %d", res.Frames, room, res.Score)
}

// TestBoardingHouseMatchesTheSmallProfile checks the house against 10.2's small column.
//
// Every band below is a cell of that table and the header comment of
// levels/Boarding House.house.txt quotes the same numbers. Without this test that comment
// is a claim: someone fills a room out to look better, the house still lints clean and
// still finishes, and it has quietly stopped being a small-tier house. The bands are wide
// because the corpus's are -- this is not pinning one design, it is keeping an edit inside
// the tier it was written for.
//
// Two rows earned their place by catching something rather than by being in the table.
// prizes/room came out at 0.941 against a 0.95 ceiling on the first pass, which is not a
// margin. And dark rooms was at 3.9 % after the Bell Turret stopped being a kRoof room,
// because kRoof lights itself and kPaneledRoom does not -- 3.9 % is *inside* 0-4 %, so
// the band did not catch it and the row printed green. The lesson is in the t.Logf at the
// bottom: the numbers are printed so they can be read, not only bounded.
func TestBoardingHouseMatchesTheSmallProfile(t *testing.T) {
	h, err := house.ParseTextFile(boardingHouseSource)
	if err != nil {
		t.Fatalf("parse %s: %v", boardingHouseSource, err)
	}
	p := measure(t, h)

	checkRows(t, "small", []profileRow{
		{"real rooms", float64(p.rooms), 45, 85, true},
		{"occupied floors", float64(p.floors), 5, 14, true},
		{"occupied suites", float64(p.suites), 13, 20, true},
		{"grid density", p.density, 0.20, 0.65, false},
		{"total objects", float64(p.objects), 300, 620, true},
		{"objects per room", p.perRoom(p.objects), 7.0, 19.0, false},
		{"empty rooms %", 100 * p.perRoom(p.empties), 10, 25, false},
		{"rooms at the 24-object ceiling", float64(p.atCeiling), 0, 6, true},
		{"enemies per room", p.perRoom(p.enemies), 0.5, 1.3, false},
		{"prizes per room", p.perRoom(p.prizes), 0.4, 0.95, false},
		{"stars", float64(p.stars), 1, 4, true},
		{"batteries", float64(p.batteries), 1, 6, true},
		{"rubber bands", float64(p.bands), 0, 4, true},
		{"dark rooms %", 100 * p.perRoom(p.dark), 0, 4, false},
		{"distinct object codes", float64(p.kinds), 50, 69, true},
		{"total points", float64(p.total), 7900, 45500, true},
	})

	// The prize:enemy ratio is a row of 10.2 and not a division of two other rows, because
	// a house can sit inside both of those bands and still be either a shooting gallery or
	// a museum. Guarded rather than divided, so a house with no enemies fails on the
	// enemies row above and not on a division by zero here.
	if p.enemies == 0 {
		t.Errorf("no enemies at all: the small tier wants 0.5-1.3 per room")
	} else {
		ratio := float64(p.prizes) / float64(p.enemies)
		if ratio < 0.7 || ratio > 1.5 {
			t.Errorf("prize:enemy = %.3f, want 0.70..1.50 (original-houses.md 10.2, small)",
				ratio)
		}
	}

	checkStartRoom(t, h, p)

	t.Logf("%d rooms, %d objects (%.2f/room), %d empty, %d at the ceiling (fullest %d), "+
		"%d enemies, %d prizes (%d stars, %d batteries, %d bands), %d distinct codes, "+
		"%d points, density %.3f, %d dark %v",
		p.rooms, p.objects, p.perRoom(p.objects), p.empties, p.atCeiling, p.fullest,
		p.enemies, p.prizes, p.stars, p.batteries, p.bands, p.kinds, p.total, p.density,
		p.dark, p.darkNames)
}

// TestBoardingHouseUsesEveryBuiltInBackground is the one row of this house's design that
// is not a band in 10.2 and is worth a test anyway.
//
// 10.3 step 3 asks for a background mix and the port cannot satisfy the user-art half of
// it (docs/IMPROVEMENTS.md 4.15). What it can do, and what this house is the first to do
// in the repository -- ours or 1994's -- is use all eighteen built-in backgrounds. That is
// a coverage claim about the renderer as much as about the house: eighteen backgrounds
// means eighteen tile sets, four different wall rules, three different ceiling rules and
// the kRoof collision surface all exercised by one file that a person can also play.
//
// If a future edit drops one, the profile rows above will not notice, because a background
// is not an object.
//
// # Why the whole histogram is pinned and not just the eighteen-of-eighteen claim
//
// A background change moves numbers the header quotes and no other test reads. The Bell
// Turret proved it twice over: it was kRoof and became kPaneledRoom, which put out its light
// (caught, because the dark-room row exists) *and* moved one room from the open-air side of
// the house to the ceilinged side, which silently falsified four figures in the header --
// the interior count, its object mean, the open-air count and its mean. Nothing caught that,
// and it was found by recomputing the header's own numbers by hand.
//
// Pinning the histogram is the cheap fix. The alternative was to compute "has a ceiling" per
// room in the test, which means either duplicating DoesRoomHaveCeiling's switch
// (internal/game/room.go:358-371 -- it is a method on the running World, not a function of a
// background) or driving a World from room to room, and the derived figures are all
// downstream of this table plus the object counts the rows above already pin. One line of
// data catches any background edit in any room.
func TestBoardingHouseUsesEveryBuiltInBackground(t *testing.T) {
	h, err := house.ParseTextFile(boardingHouseSource)
	if err != nil {
		t.Fatalf("parse %s: %v", boardingHouseSource, err)
	}
	seen := map[int16]int{}
	for i := range h.Rooms {
		seen[h.Rooms[i].Background]++
	}
	// 2000..2017 is kSimpleRoom..kStars, the built-in range; >= 3000 is user art.
	var missing []int16
	for bg := int16(2000); bg <= 2017; bg++ {
		if seen[bg] == 0 {
			missing = append(missing, bg)
		}
	}
	if len(missing) != 0 {
		t.Errorf("backgrounds %v are unused: this house's claim is all 18 of 2000-2017",
			missing)
	}
	for bg, n := range seen {
		if bg < 2000 || bg > 2017 {
			t.Errorf("background %d is outside the built-in range (%d room(s)): a house of "+
				"ours may not carry art, see docs/IMPROVEMENTS.md 4.15", bg, n)
		}
	}

	// 30 rooms of interior art (2000-2008) and 21 outdoor (2009-2017), of which the seven
	// backgrounds DoesRoomHaveCeiling excludes account for 16. Those three figures and the
	// object means the header derives from them are what this table protects.
	want := map[int16]int{
		2000: 5, 2001: 5, 2002: 4, 2003: 3, 2004: 2, 2005: 4, 2006: 2, 2007: 2, 2008: 3,
		2009: 1, 2010: 1, 2011: 4, 2012: 2, 2013: 1, 2014: 3, 2015: 6, 2016: 2, 2017: 1,
	}
	for bg := int16(2000); bg <= 2017; bg++ {
		if seen[bg] != want[bg] {
			t.Errorf("background %d is used by %d room(s), want %d: the header's indoor/"+
				"outdoor mix and its per-room object means are computed from this split, and "+
				"nothing else would notice them going stale", bg, seen[bg], want[bg])
		}
	}

	t.Logf("%d backgrounds over %d rooms: %v", len(seen), len(h.Rooms), seen)
}
