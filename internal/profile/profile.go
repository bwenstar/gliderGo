// Package profile measures a house against the shape of the 1994 ones.
//
// docs/PLAN.md Stage 2 asks for houses "designed against the quantitative profile of
// the originals in docs/analysis/original-houses.md -- comparable room counts, object
// vocabulary and difficulty curve, not just 'some rooms'". Section 10.2 of that
// document is the profile: eighteen rows of bands across five size tiers, derived from
// all 4,070 shipped rooms. This package is the walk that fills them in.
//
// # Why it is a package and not a test helper
//
// The walk had two callers and no home. internal/replay's two house tests each hold a
// tier's column and shared the counting between them, which is the right arrangement
// for a test -- a divergence between two houses is then always a difference in the
// houses -- and the wrong one for an author, because a test only speaks when it fails.
// The thing somebody writing a house wants is the numbers, printed, before there is a
// test to fail: "you are at 2.9 objects a room and the ceiling is 3.1". That is
// `glidertool house stats`, and it and the tests now measure with the same code
// (docs/IMPROVEMENTS.md 4.16).
//
// # What it can and cannot see
//
// Sixteen of the eighteen rows are arithmetic over the house file. One needs the
// renderer -- a dark room is a room GetNumberOfLights answers 0 for, and that is a
// method on *render.Scene because it reads the background's self-lighting table and
// each lamp's switch state. One needs the room graph, which is graph.go's business and
// the reason this package imports internal/game.
//
// So this package sits above house, render and game, and nothing in those three knows
// it exists. That is deliberate: a measurement must not become something the game or
// the codecs depend on.
package profile

import (
	"fmt"

	"github.com/bwenstar/gliderGo/internal/game"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// RoomRef names one room, by index as well as by name.
//
// The index is not decoration. The shipped houses reuse names freely -- Titanic has 24
// rooms called "Murmur" and 9 called "Dirt", 43 unreachable rooms between them -- so a
// report that lists names alone tells an author that something is wrong and gives them no
// way to find it. Both lists Profile publishes are lists of these for that reason.
type RoomRef struct {
	Index int16
	Name  string
}

func (r RoomRef) String() string { return fmt.Sprintf("%d %q", r.Index, r.Name) }

// Profile is one house, measured. Every field is either a row of 10.2 or something a
// row is computed from, plus the names behind two of the counts -- because "3 dark
// rooms" sends an author looking and "3 dark rooms: Bell Turret, Cellar, Attic" sends
// them to the right ones.
type Profile struct {
	Rooms  int // real rooms: slots whose suite is not kRoomIsEmpty
	Slots  int // rooms plus the empty slots, i.e. what the file holds
	Floors int
	Suites int

	Objects   int
	Empty     int
	Fullest   int
	AtCeiling int // rooms holding all MaxRoomObs objects, which 10.2 counts separately

	Enemies   int
	Prizes    int
	Stars     int
	Batteries int
	Bands     int
	Kinds     int

	Dark      int
	DarkRooms []RoomRef

	// Points is CountTotalHousePoints (HouseInfo.c:51-105): what a player who cleared
	// the house would hold. PrizePoints is the part of it the objects contribute.
	Points      int32
	PrizePoints int32

	// Reachable, Unreachable and Eccentricity come from the static room graph. See
	// graph.go for what the word "static" is carrying.
	Reachable    int
	Unreachable  []RoomRef
	Eccentricity int

	// Start is the room the walk began at, or house.RoomIsEmpty for a house it could
	// not start in at all. It is GetFirstRoomNumber's answer and not the firstRoom
	// field: a house naming a room it does not contain starts the player in room 0, and
	// a reachable count is not readable without knowing which room it counts from.
	Start int16

	// ForkBounded counts the rooms that read their four openings from the house's own
	// 'bnds' resource: a background of the house's own, and no bounds field in the room
	// to override it (room.go:139-150).
	//
	// Not a row of 10.2. It is here because it is exactly the number of rooms a
	// measurement made with no assets gets wrong, which lets a caller say how much that
	// cost instead of warning unconditionally -- and a caveat that fires on every house
	// is a caveat nobody reads. 10.1 step 10 puts the corpus total at 155.
	ForkBounded int
}

// Density is 10.2's grid density: how much of the floor-by-suite rectangle a house
// actually occupies. A house of 43 rooms on 7 floors and 10 suites has 70 cells and
// fills 61 % of them.
func (p Profile) Density() float64 {
	if cells := p.Floors * p.Suites; cells > 0 {
		return float64(p.Rooms) / float64(cells)
	}
	return 0
}

// PerRoom is the division 10.2 states most of its bands as. The empty house is guarded
// so a parse that silently produced nothing fails on a band rather than on a NaN.
func (p Profile) PerRoom(n int) float64 {
	if p.Rooms == 0 {
		return 0
	}
	return float64(n) / float64(p.Rooms)
}

// Object codes this file counts by name. Resolved once, and a miss panics at init
// rather than being handled at every use: the names are compiled into internal/house's
// table, so a failure here is a rename in that table and not anything a caller did.
var (
	star      = mustCode("kStar")
	battery   = mustCode("kBattery")
	rubber    = mustCode("kBands")
	invisible = mustCode("kInvisBonus")

	// The four clocks and the star are the prizes with a fixed value; every other
	// bonus is worth what its own `points` field says. See Points below.
	fixedPoints = map[int16]int32{
		mustCode("kRedClock"):    game.RedClockPoints,
		mustCode("kBlueClock"):   game.BlueClockPoints,
		mustCode("kYellowClock"): game.YellowClockPoints,
		mustCode("kCuckoo"):      game.CuckooClockPoints,
		mustCode("kStar"):        game.StarPoints,
	}
)

func mustCode(name string) int16 {
	c, ok := house.ObjectCode(name)
	if !ok {
		panic("profile: no object code named " + name)
	}
	return c
}

// Measure walks a house and fills in every row of 10.2.
//
// It builds one Scene and one World for the whole house, which is what the two rows
// that are not arithmetic need.
//
// # The assets argument is not optional in the way it looks
//
// Pass nil and you get a measurement, and for fifteen of the 22 shipped houses it is
// the same measurement. For the other seven it is not: a room with a house's own
// background and no `bounds` field of its own reads its four openings from that
// house's 'bnds' resource, and a resource that cannot be read means "closed on all four
// sides" (GetOriginalBounding, room.go:260-301). 155 rooms take that path, 146 of them
// with at least one side open, so a walk with no art can only lose edges, never gain
// them, and the reachable count comes out low. That is not hypothetical: it is the shape
// of the bug this package found (docs/IMPROVEMENTS.md 4.24), where the resource was
// unreadable for a different reason.
//
// The caller is also responsible for having opened the house's own fork -- Assets
// carries one house's resources at a time, exactly as HouseIO.c's OpenHouseResFork
// does, and which house it is holding is not something this package can guess from a
// *house.House. `glidertool house stats` does it by file name; the corpus test does it
// by house name.
func Measure(h *house.House, assets *render.Assets) Profile {
	if assets == nil {
		assets = render.NewAssets(nil)
	}
	scene := render.NewScene(render.DefaultView(), assets, h)

	var p Profile
	p.Slots = len(h.Rooms)
	floors, suites, kinds := map[int16]bool{}, map[int16]bool{}, map[int16]bool{}

	for i := range h.Rooms {
		rm := &h.Rooms[i]
		if rm.Suite == house.RoomIsEmpty {
			// A slot in the file that is not a room. 10.2's counts are all of "real
			// rooms", and a deleted room still occupies a record.
			continue
		}
		p.Rooms++
		floors[rm.Floor], suites[rm.Suite] = true, true
		if rm.Background >= game.UserBackground && rm.Bounds == 0 {
			p.ForkBounded++
		}

		n := rm.LiveObjects()
		p.Objects += n
		if n == 0 {
			p.Empty++
		}
		if n > p.Fullest {
			p.Fullest = n
		}
		if n == house.MaxRoomObs {
			p.AtCeiling++
		}
		if scene.GetNumberOfLights(int16(i)) == 0 {
			p.Dark++
			p.DarkRooms = append(p.DarkRooms, RoomRef{int16(i), rm.Name.Text()})
		}

		for j := range rm.Objects {
			o := rm.Objects[j]
			if o.IsEmpty() {
				continue
			}
			kinds[o.What] = true
			switch o.Group() {
			case house.GroupEnemy:
				p.Enemies++
			case house.GroupBonus:
				p.Prizes++
				p.PrizePoints += bonusPoints(o)
				switch o.What {
				case star:
					p.Stars++
				case battery:
					p.Batteries++
				case rubber:
					p.Bands++
				}
			}
		}
	}

	p.Floors, p.Suites, p.Kinds = len(floors), len(suites), len(kinds)
	// HouseInfo.c:51-105: a hundred a room, plus what the prizes are worth. The room
	// term is every real room, including the first -- which the player is never
	// credited with, because HandleRoomVisitation scores the room being *left*. The
	// discrepancy is the original's and is what its own house-info dialog reported.
	p.Points = int32(100*p.Rooms) + p.PrizePoints

	g := walk(h, scene)
	p.Reachable, p.Unreachable, p.Eccentricity = g.reachable, g.unreachable, g.eccentricity
	p.Start = g.start
	return p
}

// bonusPoints is what one prize adds to CountTotalHousePoints.
//
// Four clocks and the star have fixed values in the C; kInvisBonus is worth whatever
// the author typed into its `points` field, which is why a house's total cannot be
// derived from its object histogram alone. Every other bonus -- the foil, the bands,
// the battery, the helium -- is worth nothing on the scoreboard and is counted as a
// prize all the same, because 10.2's prizes/room row is about what a room offers and
// not about what it pays.
func bonusPoints(o house.Object) int32 {
	if v, ok := fixedPoints[o.What]; ok {
		return v
	}
	if o.What == invisible {
		return int32(o.Bonus().Points)
	}
	return 0
}
