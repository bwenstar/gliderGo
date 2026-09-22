package profile

// The static room graph: which rooms can be walked to from which, and how far the
// furthest one is.
//
// This is 10.2's last two rows -- BFS eccentricity and the reachable count -- and it
// is the only part of a house profile that measures its *shape* rather than its
// contents. Everything else in this package is a tally; this is a walk.
//
// # Why "static" is doing real work in that sentence
//
// The running game decides whether the glider can leave a room per frame, and the
// answer depends on where the glider is standing. A door is a 16-pixel strip
// (hotspots.go's kDoorInLf and its seven siblings): the wall is solid everywhere
// else, and what the door does is set one Ignore flag for one frame while the glider
// overlaps it. A manhole is a hole in the floor that the glider has to be *entirely*
// over. A kDirt room's ceiling is open only above the two tile columns drawn with
// sky. None of that can be answered about a room nobody is in.
//
// So this model answers a weaker question: is there *any* position from which the
// glider could leave this room in this direction? It is an over-approximation of
// escape and an under-approximation of difficulty -- an edge here may take a skilled
// player four attempts, and the graph cannot tell. It is the same model
// tools/probe_houses_inventory.py used to produce the per-house figures in
// docs/analysis/original-houses.md 3.6, which is why those figures can be a test
// (see graph_test.go) rather than a second opinion.
//
// # Why the numbers are reported and never linted
//
// Every one of the 22 shipped houses has rooms this model calls unreachable, and not
// by a little: Art Museum reaches 76 of 109, Fun House reaches 5 of 43. Some of that
// is the approximation (a switch that opens a door has no edge here, because a switch
// is not a transport) and some of it is genuine authoring residue. Either way
// "unreachable" is a description and not a verdict, which is why it belongs in
// `glidertool house stats` with no severity attached and no effect on the exit code.
// docs/IMPROVEMENTS.md 4.1 declined static reachability as a lint check for the same
// reason and that decision stands; what changed is that a number an author can read
// is useful even when a rule that fails them would not be.

import (
	"github.com/bwenstar/gliderGo/internal/game"
	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// roomGraph is one walk's result. The three fields are what Profile publishes; the
// adjacency itself is not kept, because nothing has needed it yet and a graph that
// escapes this file would have to document the approximation all over again.
type roomGraph struct {
	reachable    int
	unreachable  []RoomRef
	eccentricity int
	// start is the room the walk began at, or house.RoomIsEmpty if it began nowhere.
	// Reported because "379 of 383 rooms are reachable" is half a fact without it, and
	// because the room it names is not always the house's firstRoom field.
	start int16
}

// sides is the four directions a room can be left in, after the objects standing in
// it have had their say. Named rather than a [4]bool so the assignments read.
type sides struct{ left, right, up, down bool }

// The eight doors and windows, split by which wall they punch through.
//
// "In" and "Ex" are interior and exterior *artwork*, not left and right, and the
// pairing is not what the names suggest: kDoorExRt and kWindowExRt suppress the right
// wall while sitting at horizontal offset 0, and kDoorInRt sits at offset 128. The
// sets below are therefore transcribed from the dispatcher rather than from the
// names, and TestDoorSetsMatchTheDispatcher drives every object code through
// CreateActiveRects and fails if these two maps and hotspots.go ever disagree.
var (
	opensLeftWall = map[int16]bool{
		game.DoorInLf:   true,
		game.DoorExLf:   true,
		game.WindowInLf: true,
		game.WindowExLf: true,
	}
	opensRightWall = map[int16]bool{
		game.DoorInRt:   true,
		game.DoorExRt:   true,
		game.WindowInRt: true,
		game.WindowExRt: true,
	}
)

// walk builds the graph and breadth-first searches it from the house's first room.
//
// It takes the Scene rather than making one because the bounds of a room with a
// house's own background come out of that house's 'bnds' resource, which is reached
// through the Scene's Assets (GetOriginalBounding, room.go:260-301). Eight of the 22
// shipped houses ship 'bnds' resources, 70 in all, and 155 rooms across 7 of them read
// their openings from one, so a walk with no asset tree reads those rooms as closed on
// all four sides -- a real difference in the answer, not a cosmetic one, and the reason
// Measure takes assets at all.
func walk(h *house.House, sc *render.Scene) roomGraph {
	// A World is the only thing that can be asked DetermineRoomOpenings, and the seed
	// is irrelevant: nothing here advances the random stream.
	w := game.NewWorld(h, sc, 1)

	// The grid is the house's address space: (floor, suite) -> room index, real rooms
	// only. Two rooms may claim the same cell -- nothing in the format forbids it --
	// and the first wins, which is what the linear scan in House.RoomNumber does and
	// therefore what the game would find.
	type cell struct{ floor, suite int16 }
	grid := make(map[cell]int16, len(h.Rooms))
	for i := range h.Rooms {
		rm := &h.Rooms[i]
		if rm.Suite == house.RoomIsEmpty {
			continue
		}
		if _, taken := grid[cell{rm.Floor, rm.Suite}]; !taken {
			grid[cell{rm.Floor, rm.Suite}] = int16(i)
		}
	}

	// Edges are directed, and that is not a simplification. Leaving a room to the
	// east needs *this* room's right side open; the destination's left wall does not
	// stop the glider arriving, because CheckEscapeRight tests thisRoom's threshold
	// and MoveRoomToRoom asks the neighbour nothing (Interactions.c:662,
	// Transit.c:398-404). So a pair of rooms can be joined one way only, and 3.6's
	// one-way passage census is the count of exactly that.
	adj := make(map[int16][]int16, len(grid))
	for i := range h.Rooms {
		rm := &h.Rooms[i]
		if rm.Suite == house.RoomIsEmpty {
			continue
		}
		from := int16(i)
		s := roomSides(w, from)

		// North is floor+1 and south is floor-1 (Room.c:562-633), which is worth
		// stating because the floor field counts upward and the screen does not.
		link := func(open bool, floor, suite int16) {
			if !open {
				return
			}
			if to, ok := grid[cell{floor, suite}]; ok {
				adj[from] = append(adj[from], to)
			}
		}
		link(s.left, rm.Floor, rm.Suite-1)
		link(s.right, rm.Floor, rm.Suite+1)
		link(s.up, rm.Floor+1, rm.Suite)
		link(s.down, rm.Floor-1, rm.Suite)

		// A transport is an edge to wherever it points. All six gate on `who != 255`
		// and make no hot rect at all without it (hotspots.go's kMailboxLf,
		// kMailboxRt, kFloorTrans, kCeilingTrans, kInvisTrans and kDeluxeTrans),
		// so an unlinked duct is scenery and not a route. The switches are *not*
		// here: a switch with a `where` opens something in another room rather than
		// carrying the glider there, and nothing in this model can follow what it
		// opened.
		for j := range rm.Objects {
			o := rm.Objects[j]
			if !house.LinkCarryingTransport(o.What) {
				continue
			}
			if who, ok := house.LinkWho(o); !ok || who == house.UnlinkedWho {
				continue
			}
			to := h.RoomLinked(o)
			if to == house.RoomIsEmpty {
				continue
			}
			// RoomLinked can name a slot that is not a room: it resolves through
			// House.RoomNumber, which matches on (floor, suite) without checking
			// that the slot is live, and the dangling sentinel -100 extracts to a
			// suite of -1 -- kRoomIsEmpty itself. The game has the same hole and
			// walks into it (Objects.c:126-174 does not guard, and ForceThisRoom
			// accepts any in-range index), but an edge into a deleted room is a
			// defect in the house rather than a route through it, and the linter's
			// link-dangling check is where it gets reported.
			if h.Rooms[to].Suite == house.RoomIsEmpty {
				continue
			}
			adj[from] = append(adj[from], to)
		}
	}

	// GetFirstRoomNumber rather than the raw field: a house naming a first room it
	// does not contain starts the player in room 0, so the walk starts there too
	// (play.go:623-633).
	first := w.GetFirstRoomNumber()

	hop := map[int16]int{}
	maxHop := 0
	started := house.RoomIsEmpty
	if first >= 0 && h.Rooms[first].Suite != house.RoomIsEmpty {
		started = first
		hop[first] = 0
		queue := []int16{first}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, next := range adj[cur] {
				if _, seen := hop[next]; seen {
					continue
				}
				hop[next] = hop[cur] + 1
				if hop[next] > maxHop {
					maxHop = hop[next]
				}
				queue = append(queue, next)
			}
		}
	}

	g := roomGraph{reachable: len(hop), eccentricity: maxHop, start: started}
	// In file order, so two runs over the same house print the same list and a diff
	// between two versions of a house is readable.
	for i := range h.Rooms {
		if h.Rooms[i].Suite == house.RoomIsEmpty {
			continue
		}
		if _, ok := hop[int16(i)]; !ok {
			g.unreachable = append(g.unreachable, RoomRef{int16(i), h.Rooms[i].Name.Text()})
		}
	}
	if len(hop) == 0 {
		// A house with no start room has no eccentricity, and 0 would read as "one
		// room, no distance". -1 is what the probe reports and what the tier bands
		// are written against.
		g.eccentricity = -1
	}
	return g
}

// roomSides asks one room which of its four sides the glider could leave by.
//
// The first four lines are the whole of DetermineRoomOpenings, reused rather than
// re-derived -- including its kDirt inconsistency, where LeftThresh says "walled" and
// LeftOpen says "open" for the same tile. This reads the *thresholds* for the sides
// and the *flags* for top and bottom, which is what the escape checks read:
// CheckEscapeLeft tests LeftThresh (Interactions.c:572), CheckEscapeUp tests TopOpen
// (:244). On the two dirt tiles where the pair disagrees the threshold is the one
// that decides whether the glider can fly out, so it is the one the graph follows.
//
// Everything after that is an object or a tile overriding the background, and each of
// the three has a citation because each is a different mechanism: a tile that is a
// hole, an object that is a hole, and an object that is a staircase.
func roomSides(w *game.World, roomNum int16) sides {
	w.ForceThisRoom(roomNum)
	w.DetermineRoomOpenings()
	rm := w.ThisRoom()
	if rm == nil {
		return sides{}
	}

	s := sides{
		left:  w.R.LeftThresh == game.NoLeftWallLimit,
		right: w.R.RightThresh == game.NoRightWallLimit,
		up:    w.R.TopOpen,
		down:  w.R.BottomOpen,
	}

	// A dirt room's ceiling and floor are drawn by its tiles, so two of the eight
	// tile pictures are sky and two are a pit. One such tile anywhere in the row
	// makes the side passable, because the glider can be standing under it.
	if rm.Background == game.Dirt {
		for _, t := range rm.Tiles {
			if player.DirtTileOpenAbove(t) {
				s.up = true
			}
			if player.DirtTileOpenBelow(t) {
				s.down = true
			}
		}
	}

	for j := range rm.Objects {
		switch what := rm.Objects[j].What; {
		// A manhole's rect is kIgnoreGround: standing over it suppresses the floor
		// for a frame and the glider drops through (hotspots.go's kManhole,
		// Interactions.c:404-441).
		case what == game.Manhole:
			s.down = true

		// A staircase is a two-object pair -- kUpStairs here and kDownStairs in the
		// room above -- and the hot rect moves the glider between them without
		// either room's ceiling being open (hotspots.go's kUpStairs/kDownStairs,
		// Transit.c's MoveItUp/MoveItDown). They carry no link: the destination is
		// the geometric neighbour, which is why this sets a direction rather than
		// adding an edge of its own.
		case what == game.UpStairs:
			s.up = true
		case what == game.DownStairs:
			s.down = true

		case opensLeftWall[what]:
			s.left = true
		case opensRightWall[what]:
			s.right = true
		}
	}
	return s
}
