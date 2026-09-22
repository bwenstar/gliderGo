package profile

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/game"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// The corpus pin for the whole package, and the hand-built cases behind it.
//
// docs/analysis/original-houses.md 3.6 publishes fifteen columns for each of the 22
// shipped houses, produced by tools/probe_houses_inventory.py -- a Python transcription
// of DetermineRoomOpenings and GetNumberOfLights written before any of this existed.
// Eight of those columns are things Measure computes, so the document can be the test:
// it is parsed, not restated, which means a figure cannot be corrected in one place and
// left stale in the other.
//
// That matters most for the graph. A BFS eccentricity has no obvious right answer to
// eyeball -- "is Teddy World's longest shortest path 44 or 43" is not a question
// anybody can check by reading -- so the only way to know the Go walk is the same walk
// is to run it against 4,070 rooms and 22 published numbers. Along the way it also
// pins GetNumberOfLights across the whole corpus, which nothing else did.
//
// That pin is necessary and not sufficient. It says the Go walk and the Python probe
// agree about 22 houses; it does not say either is right about any single rule, because
// a rule that fires in no shipped house and a rule that fires in all of them both pass
// it unchanged. So the second half of this file builds one- and two-room houses in which
// exactly one thing is true -- a manhole, a dirt tile, a one-way wall, a link -- and
// checks that one thing. Between them: the corpus says the model is the probe's model,
// and these say what the model is.

const (
	corpusRoot = "../../assets"
	sectionDoc = "../../docs/analysis/original-houses.md"
)

// publishedRow is one row of 3.6, holding only the columns Profile can answer.
type publishedRow struct {
	name      string
	rooms     int
	objMean   string // one decimal place, compared as text so 7.1 is not 7.0999999
	fullest   int
	atCeiling int
	dark      int
	stars     int
	reachable int
	maxHop    int
}

// bold and code markers around the numbers in the table: 3.6 emphasises the extremes,
// and `**2**` is still 2.
var tableMarkup = regexp.MustCompile("[*`]")

// readSection36 parses the table under "### 3.6" out of the analysis document.
func readSection36(t *testing.T) []publishedRow {
	t.Helper()
	raw, err := os.ReadFile(sectionDoc)
	if err != nil {
		t.Skipf("no analysis document at %s: %v", sectionDoc, err)
	}

	lines := strings.Split(string(raw), "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "### 3.6") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("%s has no section 3.6; this test reads its table", sectionDoc)
	}

	num := func(s string) int {
		s = tableMarkup.ReplaceAllString(s, "")
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			t.Fatalf("3.6: %q is not a number: %v", s, err)
		}
		return n
	}

	var rows []publishedRow
	for _, l := range lines[start:] {
		if strings.HasPrefix(l, "## ") || strings.HasPrefix(l, "### 3.7") {
			break
		}
		if !strings.HasPrefix(l, "| ") {
			continue
		}
		cells := strings.Split(strings.Trim(l, "|"), "|")
		if len(cells) != 15 {
			continue
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if cells[0] == "House" || strings.HasPrefix(cells[0], "---") {
			continue
		}
		// reach/real is one cell holding two numbers; only the first is new
		// information, since `real` is the rooms column again.
		reach := strings.SplitN(tableMarkup.ReplaceAllString(cells[13], ""), "/", 2)
		if len(reach) != 2 {
			t.Fatalf("3.6: %q is not a reach/real pair", cells[13])
		}
		rows = append(rows, publishedRow{
			name:      cells[0],
			rooms:     num(cells[1]),
			objMean:   tableMarkup.ReplaceAllString(cells[2], ""),
			fullest:   num(cells[4]),
			atCeiling: num(cells[5]),
			dark:      num(cells[6]),
			stars:     num(cells[10]),
			reachable: num(reach[0]),
			maxHop:    num(cells[14]),
		})
		if got := num(reach[1]); got != rows[len(rows)-1].rooms {
			t.Errorf("3.6 %s: reach/real says %d real rooms, the rooms column says %d",
				cells[0], got, rows[len(rows)-1].rooms)
		}
	}
	if len(rows) == 0 {
		t.Fatalf("%s section 3.6 has no table rows", sectionDoc)
	}
	return rows
}

// TestTheProfileMatchesThePublishedCorpusFigures measures all 22 shipped houses and
// compares eight columns each against 3.6.
//
// Every house is one subtest, so a single house drifting names itself rather than
// failing the lot, and every column is reported rather than the first mismatch --
// because a walk that is wrong is usually wrong about several houses at once and the
// pattern is the diagnosis.
func TestTheProfileMatchesThePublishedCorpusFigures(t *testing.T) {
	rows := readSection36(t)

	houseDir := filepath.Join(corpusRoot, "extracted", "houses")
	if _, err := os.Stat(houseDir); err != nil {
		t.Skipf("no extracted assets at %s (run `make assets`)", houseDir)
	}
	artFS := dirFS(filepath.Join(corpusRoot, "extracted", "art"))
	forkRoot := filepath.Join(corpusRoot, "extracted", "houseart")

	// Rooms across the whole corpus whose openings come out of a 'bnds' resource,
	// accumulated across the subtests and checked against 10.1 step 10 below. empty is
	// accumulated the same way, against a figure 3.6 does not publish per house.
	forkBounded, empty := 0, 0

	for _, want := range rows {
		t.Run(want.name, func(t *testing.T) {
			h, err := house.LoadFile(filepath.Join(houseDir, want.name+".house"))
			if err != nil {
				t.Fatalf("loading %s: %v", want.name, err)
			}

			// The house's own resources go in front of the application's, as
			// HouseIO.c's OpenHouseResFork does. Eight of the 22 ship 'bnds'
			// resources and without them those houses' custom-background rooms read
			// as sealed, which changes the reachable count.
			a := render.NewAssets(artFS)
			if fork := dirFS(filepath.Join(forkRoot, want.name)); fork != nil {
				a.OpenHouseResFork(want.name, fork)
			}

			p := Measure(h, a)
			forkBounded, empty = forkBounded+p.ForkBounded, empty+p.Empty

			eq := func(what string, got, exp int) {
				if got != exp {
					t.Errorf("%s: got %d, 3.6 says %d", what, got, exp)
				}
			}
			eq("real rooms", p.Rooms, want.rooms)
			eq("fullest room", p.Fullest, want.fullest)
			eq("rooms at the 24 ceiling", p.AtCeiling, want.atCeiling)
			eq("dark rooms", p.Dark, want.dark)
			eq("stars", p.Stars, want.stars)
			eq("rooms reachable from the first", p.Reachable, want.reachable)
			eq("BFS eccentricity", p.Eccentricity, want.maxHop)

			if got := strconv.FormatFloat(p.PerRoom(p.Objects), 'f', 1, 64); got != want.objMean {
				t.Errorf("objects per room: got %s, 3.6 says %s", got, want.objMean)
			}
			if len(p.DarkRooms) != p.Dark {
				t.Errorf("%d dark rooms counted but %d named", p.Dark, len(p.DarkRooms))
			}
			if len(p.Unreachable) != p.Rooms-p.Reachable {
				t.Errorf("%d rooms unreachable by the count but %d named",
					p.Rooms-p.Reachable, len(p.Unreachable))
			}
			// The room the reachable count counts from. All 22 name a live one, so a
			// Start of kRoomIsEmpty here would mean the walk began nowhere -- and a
			// reachable count of 0 for that reason looks exactly like a broken rule.
			if p.Start < 0 || int(p.Start) >= len(h.Rooms) ||
				h.Rooms[p.Start].Suite == house.RoomIsEmpty {
				t.Errorf("the walk started at room %d, which is not a live room", p.Start)
			}
		})
	}

	// 10.1 step 10: "155 shipped rooms rely on the 'bnds' fallback and zero of them are
	// missing the resource." This is the other half of why Measure takes assets at all,
	// and it is checked here rather than trusted because the number decides whether a
	// caller who passed nil got the right answer -- the ForkBounded field exists so that
	// `house stats -no-assets` can say how wrong it is, and a field that counts the wrong
	// thing would make that caveat confidently misleading.
	if want := 155; forkBounded != want {
		t.Errorf("%d rooms across the corpus read their openings from a 'bnds' resource; "+
			"original-houses.md 10.1 step 10 says %d", forkBounded, want)
	}

	// 5.4's class-signature table publishes "(empty room): 840, 20.6 %" as its commonest
	// signature, which is the only place the corpus's empty-room count appears -- 3.6 has
	// no column for it and 10.2 has a row whose cells turn out to be something else
	// (4.25). Pinned here because "no objects" is not as obvious a predicate as it sounds:
	// a room's 24 slots are ObjectIsEmpty-padded, a deleted room is a slot rather than a
	// room, and counting either wrong moves 10.2's empty-rooms row for every house at once
	// while leaving the other seventeen looking fine.
	if want := 840; empty != want {
		t.Errorf("%d rooms across the corpus hold no objects; original-houses.md 5.4 says "+
			"%d (20.6 %% of 4,070)", empty, want)
	}
}

// ---------------------------------------------------------------------------
// The hand-built cases
// ---------------------------------------------------------------------------

// sealedRoom is a room with a wall on all four sides, so that whatever a test then opens
// is the object or the tile it put there and nothing else.
//
// kSimpleRoom is one of the eleven interiors, which DoesRoomHaveFloor and
// DoesRoomHaveCeiling both answer "it has one" for; tile 0 on the left edge and tile 7 on
// the right are what interiorThresholds reads as walled. Those two tile values are not
// portable to kDirt -- its walled left tile is 1, not 0 -- which is why the dirt test
// below builds its own room rather than calling this.
func sealedRoom(floor, suite int16) house.Room {
	rm := house.Room{Background: game.SimpleRoom, Floor: floor, Suite: suite}
	for i := range rm.Objects {
		rm.Objects[i] = house.Object{What: house.ObjectIsEmpty}
	}
	rm.Tiles[0] = 0
	rm.Tiles[house.NumTiles-1] = house.NumTiles - 1
	return rm
}

// handBuiltScene wires a hand-built house up the way Measure does, with no art.
//
// No art is correct here rather than merely convenient: every room in this half of the
// file has a background below kUserBackground, so no 'bnds' resource is consulted and the
// openings are entirely the fixture's doing. A test that needed art would be testing the
// asset tree as well as the walk.
func handBuiltScene(h *house.House) *render.Scene {
	return render.NewScene(render.DefaultView(), render.NewAssets(nil), h)
}

func handBuilt(h *house.House) *game.World {
	return game.NewWorld(h, handBuiltScene(h), 1)
}

// oneSealedRoom is the single-room house the per-room tests start from.
func oneSealedRoom() *house.House {
	return &house.House{
		Version:   house.HouseVersion,
		NRooms:    1,
		FirstRoom: 0,
		Rooms:     []house.Room{sealedRoom(0, 0)},
	}
}

// TestARoomsObjectsOpenTheSidesItsBackgroundClosed drives all eleven object codes
// roomSides overrides on, one at a time, into a room with no way out.
//
// Eleven cases and a control, and the control is the one that would catch the mistake
// worth catching. Every assertion here is about a side becoming open, so a roomSides that
// opened *every* side for *any* object would pass all eleven -- and produce a room graph
// in which any furnished room connects to all four neighbours. kTable is there to say no.
//
// The objects carry no payload, and that is the model rather than an omission: roomSides
// switches on the code alone, because the question it answers is "is there any position
// from which the glider could leave this way" and a door's 16-pixel strip is somewhere in
// the wall wherever the author dragged it. graph.go's own comment is where that
// approximation is argued; TestDoorSetsMatchTheDispatcher is what ties the eight codes to
// the arms that build those strips.
func TestARoomsObjectsOpenTheSidesItsBackgroundClosed(t *testing.T) {
	// The fixture before the cases: if the sealed room is not sealed, every case below
	// passes for the wrong reason.
	if got := roomSides(handBuilt(oneSealedRoom()), 0); got != (sides{}) {
		t.Fatalf("the sealed fixture reads as %+v, so the cases below prove nothing", got)
	}

	for _, c := range []struct {
		code int16
		want sides
	}{
		{game.DoorInLf, sides{left: true}},
		{game.DoorExLf, sides{left: true}},
		{game.WindowInLf, sides{left: true}},
		{game.WindowExLf, sides{left: true}},
		{game.DoorInRt, sides{right: true}},
		{game.DoorExRt, sides{right: true}},
		{game.WindowInRt, sides{right: true}},
		{game.WindowExRt, sides{right: true}},
		{game.Manhole, sides{down: true}},
		{game.UpStairs, sides{up: true}},
		{game.DownStairs, sides{down: true}},
		// The control: a table is something to stand on, not a way out.
		{game.Table, sides{}},
	} {
		t.Run(house.ObjectName(c.code), func(t *testing.T) {
			h := oneSealedRoom()
			h.Rooms[0].Objects[0] = house.Object{What: c.code}
			h.Rooms[0].NumObjects = 1
			if got := roomSides(handBuilt(h), 0); got != c.want {
				t.Errorf("roomSides = %+v, want %+v", got, c.want)
			}
		})
	}
}

// TestADirtRoomsTilesOpenItsCeilingAndItsFloor covers the one background whose openings
// are drawn rather than declared, and the one place the port has to choose between two
// fields that disagree.
func TestADirtRoomsTilesOpenItsCeilingAndItsFloor(t *testing.T) {
	// A sealed dirt room: tile 1 on the left edge, because kDirt's walled tile is 1 and
	// not 0, tile 7 on the right, and tile 4 in between. Tile 4 is inert --
	// DirtTileOpenAbove takes 5 and 6, DirtTileOpenBelow takes 2 and 3 -- so it is the
	// dirt tile that draws neither sky nor a pit.
	dirt := func(middle int16) *house.House {
		rm := house.Room{Background: game.Dirt, Floor: 0, Suite: 0}
		for i := range rm.Objects {
			rm.Objects[i] = house.Object{What: house.ObjectIsEmpty}
		}
		for i := range rm.Tiles {
			rm.Tiles[i] = 4
		}
		rm.Tiles[0] = 1
		rm.Tiles[house.NumTiles-1] = house.NumTiles - 1
		rm.Tiles[3] = middle
		return &house.House{
			Version: house.HouseVersion, NRooms: 1, FirstRoom: 0,
			Rooms: []house.Room{rm},
		}
	}

	for _, c := range []struct {
		tile int16
		want sides
	}{
		{4, sides{}},
		{5, sides{up: true}},
		{6, sides{up: true}},
		{2, sides{down: true}},
		{3, sides{down: true}},
	} {
		t.Run("tile "+strconv.Itoa(int(c.tile)), func(t *testing.T) {
			if got := roomSides(handBuilt(dirt(c.tile)), 0); got != c.want {
				t.Errorf("roomSides = %+v, want %+v", got, c.want)
			}
		})
	}

	// And the inconsistency roomSides had to pick a side of. A dirt room with tile 1 on
	// its left edge comes out of DetermineRoomOpenings with LeftThresh = LeftWallLimit
	// (there is a wall) and LeftOpen = true (there is not) at once; room.go:176-201 is
	// where that is argued and TestDirtInconsistencyIsReachableInShippedHouses is the
	// census of how often it happens. The graph follows the threshold, because the
	// threshold is what CheckGliderInRoom actually stops the glider with.
	w := handBuilt(dirt(4))
	got := roomSides(w, 0)
	if !w.R.LeftOpen {
		t.Fatal("tile 1 on the left edge no longer trips the kDirt inconsistency, so the " +
			"assertion below is vacuous")
	}
	if got.left {
		t.Error("a dirt room whose left tile is 1 has LeftOpen true and LeftThresh walled; " +
			"the graph is meant to follow the threshold and it followed the flag")
	}
}

// TestTheWalkFollowsAOneWayPassageOneWay is the directedness of the adjacency, which is
// the single most surprising thing about the model and the one a reader is most likely to
// assume away.
//
// Two rooms side by side: the west room's right wall is open and the east room's left wall
// is not. That is a legal and common shape rather than a broken house -- CheckEscapeRight
// tests the room being left and MoveRoomToRoom asks the destination nothing -- so the
// glider goes east and cannot come back, and 3.6's one-way passage column is the count of
// exactly this. The test builds the same two rooms twice and moves only the first room.
func TestTheWalkFollowsAOneWayPassageOneWay(t *testing.T) {
	build := func(first int16) *house.House {
		west, east := sealedRoom(0, 0), sealedRoom(0, 1)
		// Anything but tile 7 on the right edge is no right wall.
		west.Tiles[house.NumTiles-1] = 0
		return &house.House{
			Version: house.HouseVersion, NRooms: 2, FirstRoom: first,
			Rooms: []house.Room{west, east},
		}
	}

	h := build(0)
	if g := walk(h, handBuiltScene(h)); g.reachable != 2 || g.eccentricity != 1 || g.start != 0 {
		t.Errorf("from the west room: %d reachable at eccentricity %d, starting at %d; "+
			"want 2 at 1 starting at 0", g.reachable, g.eccentricity, g.start)
	}
	h = build(1)
	g := walk(h, handBuiltScene(h))
	if g.reachable != 1 || g.eccentricity != 0 || g.start != 1 {
		t.Errorf("from the east room: %d reachable at eccentricity %d, starting at %d; "+
			"want 1 at 0 starting at 1", g.reachable, g.eccentricity, g.start)
	}
	if len(g.unreachable) != 1 {
		t.Errorf("%d rooms named unreachable, want 1 -- the count and the list have to "+
			"agree or `house stats` prints a number it cannot back up", len(g.unreachable))
	}
}

// TestOnlyALinkedTransportIsAnEdge covers walk's four gates on a link: the object has to
// be one of the six transports, its `who` has to be something other than 255, its `where`
// has to resolve to a room the house contains, and that room has to be a live one.
//
// Two rooms that do not touch -- floor 0 suite 0 and floor 3 suite 7 -- so no geometric
// edge can exist and the only thing the walk can find is the link.
func TestOnlyALinkedTransportIsAnEdge(t *testing.T) {
	build := func(what int16, who byte, floor, suite int16) *house.House {
		h := &house.House{
			Version: house.HouseVersion, NRooms: 2, FirstRoom: 0,
			Rooms: []house.Room{sealedRoom(0, 0), sealedRoom(3, 7)},
		}
		o := house.Object{What: what}
		// Transport and Switch put `where` at payload offset 6 and `who` at 8, so one
		// setter serves both families; LinkWhere and LinkWho are what read them back,
		// each through its own union member. See internal/house's note on the aliasing.
		o.SetTransport(house.Transport{
			TopLeft: house.Point{V: 100, H: 100},
			Where:   h.MergeFloorSuite(floor, suite),
			Who:     who,
		})
		h.Rooms[0].Objects[0] = o
		h.Rooms[0].NumObjects = 1
		return h
	}

	for _, c := range []struct {
		name string
		h    *house.House
		want int
	}{
		{"a linked transporter", build(game.InvisTrans, 0, 3, 7), 2},
		// 255 and not -1, because `who` is a byte. An unlinked duct makes no hot rect
		// at all in hotspots.go, so it is scenery and not a route.
		{"an unlinked transporter", build(game.InvisTrans, house.UnlinkedWho, 3, 7), 1},
		// A switch with a target opens something in another room rather than carrying
		// the glider there, and nothing in this model can follow what it opened.
		{"a linked switch", build(game.LightSwitch, 0, 3, 7), 1},
		// A link naming a floor and suite the house does not contain. Six of the 22
		// shipped houses have these, Slumberland 69 of them.
		{"a link to a room that is not there", build(game.InvisTrans, 0, 9, 9), 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			if g := walk(c.h, handBuiltScene(c.h)); g.reachable != c.want {
				t.Errorf("%d rooms reachable, want %d", g.reachable, c.want)
			}
		})
	}
}

// TestALinkIntoADeletedRoomIsNotAnEdge is the fourth gate on its own, because it needs a
// house shaped in a way the helper above cannot make.
//
// The dangling sentinel a deleted link leaves behind is -100, which is
// MergeFloorSuite(0, kRoomIsEmpty) and decodes back to floor -8, suite -1. RoomNumber
// matches on (floor, suite) without checking that the slot is live, so it will happily
// return the index of an empty record -- and the original does the same and walks into it.
// An edge into a deleted room is a defect in the house rather than a route through it, so
// the walk drops it and the linter's link-dangling check is what reports it.
func TestALinkIntoADeletedRoomIsNotAnEdge(t *testing.T) {
	deleted := house.Room{Floor: -8, Suite: house.RoomIsEmpty}
	for i := range deleted.Objects {
		deleted.Objects[i] = house.Object{What: house.ObjectIsEmpty}
	}
	h := &house.House{
		Version: house.HouseVersion, NRooms: 2, FirstRoom: 0,
		Rooms: []house.Room{sealedRoom(0, 0), deleted},
	}
	o := house.Object{What: game.InvisTrans}
	o.SetTransport(house.Transport{TopLeft: house.Point{V: 100, H: 100}, Where: -100, Who: 0})
	h.Rooms[0].Objects[0] = o
	h.Rooms[0].NumObjects = 1

	// The fixture has to actually resolve to the empty slot, or the guard is untested.
	if got := h.RoomLinked(o); got != 1 {
		t.Fatalf("the dangling link resolves to room %d, not the empty slot at 1; "+
			"MergeFloorSuite or ExtractFloorSuite has moved", got)
	}
	g := walk(h, handBuiltScene(h))
	if g.reachable != 1 {
		t.Errorf("%d rooms reachable, want 1 -- the walk followed a link into a deleted "+
			"room", g.reachable)
	}
	// One real room, so nothing is unreachable: an empty slot is not a room the author
	// has lost, and counting it as one would put a blank name in `house stats`.
	if len(g.unreachable) != 0 {
		t.Errorf("unreachable names %v; an empty slot is not a room", g.unreachable)
	}
}

// corpusProfile measures one shipped house by name, with its own resource fork open --
// the loading dance every test in this package needs and the same one `house stats` does
// by file name.
func corpusProfile(t *testing.T, name string) Profile {
	t.Helper()
	path := filepath.Join(corpusRoot, "extracted", "houses", name+".house")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no extracted assets at %s (run `make assets`)", path)
	}
	h, err := house.LoadFile(path)
	if err != nil {
		t.Fatalf("loading %s: %v", name, err)
	}
	a := render.NewAssets(dirFS(filepath.Join(corpusRoot, "extracted", "art")))
	if fork := dirFS(filepath.Join(corpusRoot, "extracted", "houseart", name)); fork != nil {
		a.OpenHouseResFork(name, fork)
	}
	return Measure(h, a)
}

// dirFS is an asset root as the loaders take it; a directory that is not there is "no
// art", which is a legitimate input and not a failure.
func dirFS(dir string) fs.FS {
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil
	}
	return os.DirFS(dir)
}
