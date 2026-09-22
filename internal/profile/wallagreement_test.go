package profile

// The two copies of "which objects open a wall" have to agree.
//
// graph.go's opensLeftWall and opensRightWall are four-entry maps, written down because
// the room graph asks "could the glider ever leave this room to the west" of a room
// nobody is standing in -- a question the game never asks and therefore has no function
// for. The answer is in CreateActiveRects, spread over eight case arms that each build a
// 16-pixel strip and hand it an action, and there is no way to call it for a whole room.
//
// So the maps are a transcription, and this is the transcription most likely to be
// silently wrong in the whole package, because the names lie. kDoorExRt opens the
// **right** wall and kDoorExLf the left -- "Ex" is exterior artwork, not a side -- and
// the two arms are twenty lines apart in hotspots.go with identical bodies save one
// constant. Pairing "Lf" with left and "Rt" with right gets four of the eight backwards,
// and every one of those four is a wrong edge in the room graph rather than a crash.
//
// The sweep below drives every object code in the format's range through the dispatcher
// and demands that the set of codes producing each of the five actions roomSides
// overrides on is exactly the set graph.go carries. It is the same shape as
// internal/game's TestLinkPredicatesAgreeWithHouse, and for the same reason: two
// spellings of one fact, neither of which can be deleted, so the check is that they
// cannot drift.

import (
	"sort"
	"testing"

	"github.com/bwenstar/gliderGo/internal/game"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// TestDoorSetsMatchTheDispatcher pins graph.go's two door-and-window sets, and
// roomSides' three single-code arms, against CreateActiveRects.
//
// All five in one sweep rather than five tests, because the sweep is the expensive part
// and because the five belong together: they are exactly the object codes that can open
// a side of a room the background says is closed. A sixth appearing in hotspots.go --
// some new thing that hands out kIgnoreGround -- fails here, which is the point. The
// room graph would otherwise go on not knowing about it.
func TestDoorSetsMatchTheDispatcher(t *testing.T) {
	w := dispatchWorld()

	// action -> the set of codes that produce it. Built for the whole `what` space in
	// one pass, so an arm making two rects with different actions is seen as both.
	byAction := map[int16]map[int16]bool{}
	for what := int16(1); what <= game.Chimes; what++ {
		for _, action := range dispatch(w, what) {
			if byAction[action] == nil {
				byAction[action] = map[int16]bool{}
			}
			byAction[action][what] = true
		}
	}

	for _, c := range []struct {
		action int16
		want   map[int16]bool
		held   string
	}{
		{game.IgnoreLeftWall, opensLeftWall, "graph.go's opensLeftWall"},
		{game.IgnoreRightWall, opensRightWall, "graph.go's opensRightWall"},
		{game.IgnoreGround, map[int16]bool{game.Manhole: true}, "roomSides' kManhole arm"},
		{game.MoveItUp, map[int16]bool{game.UpStairs: true}, "roomSides' kUpStairs arm"},
		{game.MoveItDown, map[int16]bool{game.DownStairs: true}, "roomSides' kDownStairs arm"},
	} {
		got, want := codeNames(byAction[c.action]), codeNames(c.want)
		if got != want {
			t.Errorf("%s comes from %s; %s says %s -- the room graph is opening the "+
				"wrong sides", game.ActionName(c.action), got, c.held, want)
		}
	}
}

// codeNames renders a set of object codes as a sorted, named list, because a failure
// that reads `[kDoorExLf kDoorInLf kWindowExLf kWindowInLf]` against
// `[kDoorExRt kDoorInLf ...]` localises the mistake and `map[49:true 51:true]` does not.
func codeNames(set map[int16]bool) string {
	names := make([]string, 0, len(set))
	for what := range set {
		names = append(names, house.ObjectName(what))
	}
	sort.Strings(names)
	out := "["
	for i, n := range names {
		if i > 0 {
			out += " "
		}
		out += n
	}
	return out + "]"
}

// dispatch runs one object code through CreateActiveRects and returns the actions of
// the hot rects it made.
func dispatch(w *game.World, what int16) []int16 {
	o := house.Object{What: what}
	// Ten bytes read as a different struct by every family; only `what` decides which,
	// so one payload serves the whole sweep. A transport payload is what the eight
	// doors read, and the same first eight bytes are a sane rect for the manhole, which
	// reads them as furniture. Who is 1 rather than 255 so that the six link transports
	// are live as well: none of them produces a wall action, and a sweep that silently
	// skipped them would be checking less than it looks.
	o.SetTransport(house.Transport{
		TopLeft: house.Point{V: 100, H: 100}, Tall: 240, Where: 400, Who: 1, Wide: 64,
	})
	w.R.Master = []game.MasterObject{{
		RoomNum: 0, ObjectNum: 0,
		RoomLink: -1, ObjectLink: -1, LocalLink: -1,
		HotNum: -1, DynaNum: -1,
		TheObject: o,
	}}
	w.R.Hot = w.R.Hot[:0]
	w.CreateActiveRects(0)

	out := make([]int16, 0, len(w.R.Hot))
	for i := range w.R.Hot {
		out = append(out, w.R.Hot[i].Action)
	}
	return out
}

// dispatchWorld is a world on the smallest house that composes, with no art.
//
// One world for the whole sweep: CreateActiveRects reads a single master entry and
// appends to the hot list, and touches nothing else. A fresh world per code would
// allocate three 640x480 surfaces a hundred and forty times over for no gain, which is
// the reasoning internal/game's own sweeps give.
//
// The object under test does not go in the room. The dispatcher reads
// Master[who].TheObject and never looks at the house, so putting it in both places
// would invite the reader to think the room mattered.
func dispatchWorld() *game.World {
	h := &house.House{
		Version:   0x0200,
		NRooms:    1,
		FirstRoom: 0,
		Rooms: []house.Room{{
			Background: game.SimpleRoom,
			Floor:      0,
			Suite:      0,
		}},
	}
	for i := range h.Rooms[0].Objects {
		h.Rooms[0].Objects[i] = house.Object{What: house.ObjectIsEmpty}
	}
	sc := render.NewScene(render.DefaultView(), render.NewAssets(nil), h)
	return game.NewWorld(h, sc, 1)
}
