package replay_test

// The acceptance test for the port's own house.
//
// docs/PLAN.md Stage 2 asks that "every new house passes the validator, is completable
// headlessly by a scripted run, and is playable start to finish by hand". The validator
// half is `glidertool house lint`, which `make levels` runs. This is the second half: it
// flies levels/Open House.house.txt from the Furnace Room to the star in the Belfry and
// insists the run reach game over without losing a glider.
//
// # Why it builds the house instead of loading one
//
// The house is built here, in memory, from the checked-in text. Pointing the test at
// assets/levels/Open House.house would have been shorter and would have tested the wrong
// file: that one is a build product, so a stale copy could pass while the source it came
// from was broken, and a fresh clone would have to run `make levels` before `make test`
// could say anything. The text is what is under version control and the text is what is
// flown. house.Save is the same encoder `glidertool house build` uses, so nothing about
// the binary is special-cased either.
//
// The temporary file exists only because replay.Run resolves a house by name or by path
// and has nowhere to accept one already parsed. Writing it to t.TempDir() and pointing
// HouseDir there keeps the script itself free of machine paths, which is the same reason
// duct.script says `house CD Demo House` and lets the harness say where that is.
//
// # What a failure means
//
// **A failure here is a flight-model change, not a house bug** -- unless the house
// changed, in which case the diff says so. The test reports the room the glider was in
// when the run ended and the last room it reached, because that names the leg that
// stopped working: the route is 4 -> 10 -> 11 -> 21 -> 22 -> 30 -> 31 -> 36 -> 37 -> 41
// and each pair is one staircase or one east crossing.
//
// It deliberately does not pin the digest or a golden trace. duct.trace exists to catch
// any change at all, frame by frame, and a second file of that kind would fail on every
// change to the flight model while saying less than it does. What this asserts is the one
// thing about the house that has to stay true: that it can be finished. Mortals is
// checked as well as the star, because a route that dies twice on the way up and finishes
// on its last glider is not a house anybody would call completable.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/game"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/replay"
)

// openHouseSource is the authored text, relative to this package.
//
// The space in the file name is deliberate and is the whole naming scheme: a house is
// called whatever its file is called -- houseType has no name field, and the picker reads
// the file name (internal/shell/library.go) -- so `levels/X.house.txt` builds to
// `assets/levels/X.house` and lists as "X" with no mapping table anywhere. Naming the
// source open-house.house.txt would have needed one.
const openHouseSource = "../../levels/Open House.house.txt"

// openHouseDir builds the authored text into a directory of one house and returns it.
//
// Both tests in this file need a root to resolve a name in, and this package cannot reach the
// archive inside the executables: nothing under internal/ imports assets/, which is the rule that
// keeps 15 MiB of PNGs out of every test binary. So the root is made here, from the text, for the
// reason the file comment gives -- the text is what is under version control.
func openHouseDir(t *testing.T) string {
	t.Helper()
	h, err := house.ParseTextFile(openHouseSource)
	if err != nil {
		t.Fatalf("parse %s: %v", openHouseSource, err)
	}
	raw, err := h.Save()
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Open House.house"), raw, 0o666); err != nil {
		t.Fatalf("write house: %v", err)
	}
	return dir
}

func TestOpenHouseCanBeFinished(t *testing.T) {
	dir := openHouseDir(t)

	s := script(t, "open-house.script")
	s.HouseDir = dir // after localAssets, which points it at the extracted tree

	res, err := replay.Run(s)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Where the glider finished, for the failure messages below. The run stops on game
	// over, so the last sample is the frame the star was collected on.
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
		t.Errorf("stars left = %d, want 0: the star in the Belfry was not collected",
			res.StarsLeft)
	}
	if want := int16(2); res.Mortals != want {
		t.Errorf("gliders = %d, want %d: the route lost one, so it is not a clean run",
			res.Mortals, want)
	}
	// 100 a room for the nine rooms entered after the first, and 5000 for the star. A
	// prize picked up by accident or a room visited twice would move this, which is worth
	// hearing about even though neither would be a failure of the house.
	if want := int32(5900); res.Score != want {
		t.Errorf("score = %d, want %d (9 rooms at 100 plus the 5000 star)", res.Score, want)
	}
	t.Logf("finished in %d frames, room %d, score %d", res.Frames, room, res.Score)
}

// TestOpenHouseMatchesTheTutorialProfile checks the house against the measurements of the
// 1994 corpus, which is the other half of the Stage 2 bullet: "designed against the
// quantitative profile of the originals in docs/analysis/original-houses.md -- comparable
// room counts, object vocabulary and difficulty curve, not just 'some rooms'".
//
// Every band below is a cell of that document's 10.2 table, tutorial column, and the header
// comment of levels/Open House.house.txt quotes the same numbers. Without this test that
// comment is a claim: someone adds four objects to make a room look better, the house still
// lints clean and still finishes, and it has quietly stopped being a tutorial-tier house.
// The bands are wide because the corpus's are -- this is not pinning one design, it is
// keeping an edit inside the tier it was written for.
//
// Two of 10.2's rows are not here. Dark rooms (target 0 %) needs GetNumberOfLights, which
// lives in internal/render and cannot be reached from a house without duplicating its rule;
// see docs/IMPROVEMENTS.md 4.16. BFS eccentricity (target 11-15) needs a graph walk over the
// openings, which is real work for one number and is filed with it.
func TestOpenHouseMatchesTheTutorialProfile(t *testing.T) {
	h, err := house.ParseTextFile(openHouseSource)
	if err != nil {
		t.Fatalf("parse %s: %v", openHouseSource, err)
	}

	prizeValue := map[string]int32{
		"kRedClock": game.RedClockPoints, "kBlueClock": game.BlueClockPoints,
		"kYellowClock": game.YellowClockPoints, "kCuckoo": game.CuckooClockPoints,
		"kStar": game.StarPoints,
	}
	points := map[int16]int32{}
	for name, v := range prizeValue {
		code, ok := house.ObjectCode(name)
		if !ok {
			t.Fatalf("no object code for %s", name)
		}
		points[code] = v
	}

	var objects, empties, enemies, prizes, stars, fullest int
	var prizePoints int32
	floors, suites := map[int16]bool{}, map[int16]bool{}
	kinds := map[int16]bool{}
	starCode, _ := house.ObjectCode("kStar")
	for i := range h.Rooms {
		rm := &h.Rooms[i]
		floors[rm.Floor], suites[rm.Suite] = true, true
		n := rm.LiveObjects()
		objects += n
		if n == 0 {
			empties++
		}
		if n > fullest {
			fullest = n
		}
		for j := range rm.Objects {
			o := rm.Objects[j]
			if o.IsEmpty() {
				continue
			}
			kinds[o.What] = true
			switch o.Group() {
			case house.GroupEnemy:
				enemies++
			case house.GroupBonus:
				prizes++
				prizePoints += points[o.What]
				if o.What == starCode {
					stars++
				}
			}
		}
	}
	rooms := len(h.Rooms)
	total := int32(100*rooms) + prizePoints
	density := float64(rooms) / float64(len(floors)*len(suites))

	// Each row: what it is, the value, and 10.2's tutorial band.
	for _, c := range []struct {
		what     string
		got      float64
		lo, hi   float64
		integral bool
	}{
		{"real rooms", float64(rooms), 35, 45, true},
		{"occupied floors", float64(len(floors)), 6, 7, true},
		{"occupied suites", float64(len(suites)), 9, 10, true},
		{"grid density", density, 0.60, 0.65, false},
		{"total objects", float64(objects), 78, 138, true},
		{"objects per room", float64(objects) / float64(rooms), 2.2, 3.1, false},
		{"empty rooms %", 100 * float64(empties) / float64(rooms), 15, 25, false},
		{"objects in the fullest room", float64(fullest), 0, 23, true},
		{"enemies per room", float64(enemies) / float64(rooms), 0.0, 0.1, false},
		{"prizes per room", float64(prizes) / float64(rooms), 0.03, 0.29, false},
		{"stars", float64(stars), 1, 1, true},
		{"distinct object codes", float64(len(kinds)), 15, 48, true},
		{"total points", float64(total), 8500, 12500, true},
	} {
		if c.got < c.lo || c.got > c.hi {
			if c.integral {
				t.Errorf("%s = %d, want %d..%d (original-houses.md 10.2, tutorial)",
					c.what, int(c.got), int(c.lo), int(c.hi))
			} else {
				t.Errorf("%s = %.3f, want %.2f..%.2f (original-houses.md 10.2, tutorial)",
					c.what, c.got, c.lo, c.hi)
			}
		}
	}

	// The start room is a separate rule (10.3 step 9): richer than average, and no enemies
	// in it, because the first thing a player sees should not be able to kill them.
	if h.FirstRoom < 0 || int(h.FirstRoom) >= rooms {
		t.Fatalf("firstRoom = %d, out of range", h.FirstRoom)
	}
	first := &h.Rooms[h.FirstRoom]
	if n, avg := first.LiveObjects(), float64(objects)/float64(rooms); float64(n) < avg {
		t.Errorf("the start room %q holds %d objects, below the %.2f average: 10.3 step 9 "+
			"wants it richer than average", first.Name.Text(), n, avg)
	}
	for j := range first.Objects {
		if o := first.Objects[j]; !o.IsEmpty() && o.Group() == house.GroupEnemy {
			t.Errorf("the start room %q holds an enemy (%s)", first.Name.Text(),
				house.ObjectName(o.What))
		}
	}

	t.Logf("%d rooms, %d objects (%.2f/room), %d empty, %d enemies, %d prizes, "+
		"%d distinct codes, %d points, density %.3f",
		rooms, objects, float64(objects)/float64(rooms), empties, enemies, prizes,
		len(kinds), total, density)
}

// TestAHouseInTheLevelsRootIsFoundByName is the lookup a bug report about one of this port's own
// houses depends on.
//
// Those houses ship *inside* the executable (assets/levels.zip), so a player who says "it sticks
// in the third room of Open House" has a name and no file, and until the levels root was searched
// here `glidertool replay -house "Open House"` answered `open Open House.house: file does not
// exist` -- naming a file nobody on earth has. The houses root is the 1994 set and does not hold
// it, which is exactly the state a release binary is in.
//
// The order is asserted the other way round too, in cmd/glidergo's levels_test.go: the houses root
// is searched first, so an original wins a name it shares with a new house.
func TestAHouseInTheLevelsRootIsFoundByName(t *testing.T) {
	s := localAssets(t, replay.NewScript("Open House", 30))

	// First the state this fixes, so that the test cannot pass by the house having quietly
	// arrived in the houses root: with no levels root there is nothing to find, and the error
	// has to name where it looked rather than repeat the file name back.
	if _, err := replay.Run(s); err == nil {
		t.Fatalf("Open House resolved with no levels root: it is not one of the 22")
	} else if !strings.Contains(err.Error(), s.HouseDir) {
		t.Errorf("the error is %q; it should name the root it searched (%s), because "+
			"\"file does not exist\" about a name somebody typed leaves out where",
			err, s.HouseDir)
	}

	s.LevelDir = openHouseDir(t)
	res, err := replay.Run(s)
	if err != nil {
		t.Fatalf("run with -levels %s: %v", s.LevelDir, err)
	}
	// That it ran at all is the claim; how many samples thirty frames produce is
	// TestSixHundredFramesMatchTheGoldenTrace's business and not this test's.
	if res.Frames == 0 || len(res.Samples) == 0 {
		t.Errorf("the house resolved and then nothing ran: frames=%d, samples=%d",
			res.Frames, len(res.Samples))
	}
}
