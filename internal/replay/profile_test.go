package replay_test

// The measurement half of docs/PLAN.md Stage 2's "designed against the quantitative
// profile of the originals in docs/analysis/original-houses.md".
//
// 10.2 of that document turns the measurements of all 4,070 shipped rooms into a table
// of bands per size tier, and each of the port's own houses is written against one
// column of it. What this file holds is the counting; what the per-house files hold is
// the column. They are separated for one reason: the houses are at different tiers, so
// if each test counted for itself, a difference between them could be a difference in
// the houses or a difference in the arithmetic, and there would be no way to tell from
// a failure which. Counted here, a divergence is always the house.
//
// # Why dark rooms is in here and not duplicated
//
// Every row below is computable from internal/house alone except one. A dark room is a
// room GetNumberOfLights answers 0 for, and that function is a method on
// *render.Scene (internal/render/locale.go:1350-1384) because it reads the background's
// self-lighting table and each light object's switch state. internal/house cannot
// import internal/render -- and would not want to, since the rule is a rendering rule.
//
// openhouse_test.go used to say the row therefore "cannot be computed", citing
// docs/IMPROVEMENTS.md 4.16, and that was wrong: *this* package already imports
// internal/render (replay.go needs it to run a game at all), so a test here can build a
// Scene over a parsed house and ask it. It costs three lines and it catches the one
// thing a band on its own does not -- a house whose light went out because its
// background changed. The Bell Turret did exactly that: it was kRoof, which lights
// itself, and became kPaneledRoom, which does not.
//
// What is still not computed is 10.2's BFS eccentricity (a graph walk over the room
// openings), which remains filed as 4.16.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/game"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// houseProfile is one row per cell of 10.2 that a house can be measured for.
type houseProfile struct {
	rooms       int
	floors      int
	suites      int
	density     float64
	objects     int
	empties     int
	fullest     int
	atCeiling   int // rooms holding all MaxRoomObs objects, which 10.2 counts separately
	enemies     int
	prizes      int
	stars       int
	batteries   int
	bands       int
	kinds       int
	dark        int
	darkNames   []string
	prizePoints int32
	total       int32
}

// perRoom is the division 10.2 states most of its bands as, guarding the empty house so
// a parse that silently produced nothing fails on a band rather than on a NaN.
func (p houseProfile) perRoom(n int) float64 {
	if p.rooms == 0 {
		return 0
	}
	return float64(n) / float64(p.rooms)
}

// measure walks a parsed house and fills in the profile.
//
// The prize values come from internal/game's constants rather than from a table written
// here, so a change to what a clock is worth moves the house's expected total points
// instead of silently disagreeing with it.
func measure(t *testing.T, h *house.House) houseProfile {
	t.Helper()

	points := map[int16]int32{}
	for name, v := range map[string]int32{
		"kRedClock": game.RedClockPoints, "kBlueClock": game.BlueClockPoints,
		"kYellowClock": game.YellowClockPoints, "kCuckoo": game.CuckooClockPoints,
		"kStar": game.StarPoints,
	} {
		code, ok := house.ObjectCode(name)
		if !ok {
			t.Fatalf("no object code for %s", name)
		}
		points[code] = v
	}
	code := func(name string) int16 {
		c, ok := house.ObjectCode(name)
		if !ok {
			t.Fatalf("no object code for %s", name)
		}
		return c
	}
	starCode, batteryCode, bandsCode := code("kStar"), code("kBattery"), code("kBands")

	// One Scene over the whole house, for the dark-room row. The view is the default 640x480
	// and the assets are empty: GetNumberOfLights reads the room's background and its light
	// objects' states and touches no art, so there is nothing for an asset tree to supply.
	scene := render.NewScene(render.DefaultView(), render.NewAssets(nil), h)

	var p houseProfile
	floors, suites, kinds := map[int16]bool{}, map[int16]bool{}, map[int16]bool{}
	for i := range h.Rooms {
		rm := &h.Rooms[i]
		floors[rm.Floor], suites[rm.Suite] = true, true

		n := rm.LiveObjects()
		p.objects += n
		if n == 0 {
			p.empties++
		}
		if n > p.fullest {
			p.fullest = n
		}
		if n == house.MaxRoomObs {
			p.atCeiling++
		}
		if scene.GetNumberOfLights(int16(i)) == 0 {
			p.dark++
			p.darkNames = append(p.darkNames, rm.Name.Text())
		}

		for j := range rm.Objects {
			o := rm.Objects[j]
			if o.IsEmpty() {
				continue
			}
			kinds[o.What] = true
			switch o.Group() {
			case house.GroupEnemy:
				p.enemies++
			case house.GroupBonus:
				p.prizes++
				p.prizePoints += points[o.What]
				switch o.What {
				case starCode:
					p.stars++
				case batteryCode:
					p.batteries++
				case bandsCode:
					p.bands++
				}
			}
		}
	}

	p.rooms, p.floors, p.suites, p.kinds = len(h.Rooms), len(floors), len(suites), len(kinds)
	if cells := p.floors * p.suites; cells > 0 {
		p.density = float64(p.rooms) / float64(cells)
	}
	// 100 a room for every room entered after the first, plus the prizes. This is the score
	// a player who cleared the house would hold, which is the quantity 10.2 tabulates.
	p.total = int32(100*p.rooms) + p.prizePoints
	return p
}

// profileRow is one cell of 10.2: a name, the measured value and the tier's band.
type profileRow struct {
	what     string
	got      float64
	lo, hi   float64
	integral bool
}

// checkRows reports every row that left its band, naming the tier so the failure says
// which column of 10.2 it was read from.
//
// It reports all of them rather than stopping at the first, because the rows are not
// independent: adding four objects to a room moves objects/room, total objects, the
// empty-room percentage and possibly prizes/room, and seeing the set is how you tell an
// edit that went slightly too far from a house that has changed tier.
func checkRows(t *testing.T, tier string, rows []profileRow) {
	t.Helper()
	for _, c := range rows {
		if c.got >= c.lo && c.got <= c.hi {
			continue
		}
		if c.integral {
			t.Errorf("%s = %d, want %d..%d (original-houses.md 10.2, %s)",
				c.what, int(c.got), int(c.lo), int(c.hi), tier)
		} else {
			t.Errorf("%s = %.3f, want %.2f..%.2f (original-houses.md 10.2, %s)",
				c.what, c.got, c.lo, c.hi, tier)
		}
	}
}

// checkStartRoom is 10.3 step 9, which both houses obey and which is not a 10.2 band:
// the first room a player sees is richer than average and cannot kill them.
func checkStartRoom(t *testing.T, h *house.House, p houseProfile) {
	t.Helper()
	if h.FirstRoom < 0 || int(h.FirstRoom) >= p.rooms {
		t.Fatalf("firstRoom = %d, out of range", h.FirstRoom)
	}
	first := &h.Rooms[h.FirstRoom]
	if n, avg := first.LiveObjects(), p.perRoom(p.objects); float64(n) < avg {
		t.Errorf("the start room %q holds %d objects, below the %.2f average: 10.3 step 9 "+
			"wants it richer than average", first.Name.Text(), n, avg)
	}
	for j := range first.Objects {
		if o := first.Objects[j]; !o.IsEmpty() && o.Group() == house.GroupEnemy {
			t.Errorf("the start room %q holds an enemy (%s): 10.3 step 9 forbids it",
				first.Name.Text(), house.ObjectName(o.What))
		}
	}
}
