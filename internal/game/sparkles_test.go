package game

// The effects layer: the two producers (DynamicMaps.c:165-254) and the two renderers
// (Render.c:322-418).
//
// Four functions over two three-slot tables, and almost everything that can go wrong in them
// is about the tables rather than the drawing. **Mode == -1 means free, so the zero value is
// not "empty"** -- an unswept table reads as three live effects and both producers then
// silently do nothing, with no counter drift and no error to see. That is the trap
// TestEffectTablesAreFreeListsNotZeroValues pins, and it is the reason InitGarbageRects is
// not optional.
//
// The rest divides into the arithmetic the producers do once (centre the cel, pick the three
// cels that spell this number) and the bookkeeping the renderers do per frame (walk the cels,
// grow the union, hand the slot back). The flying point's union is the mirror image of the six
// movers': grown at the leading edge here, pushed back at the trailing edge there, and the
// same rect either way -- see TestFlyingPointUnionGrowsAtTheLeadingEdge, which reuses
// movers_test.go's unionOf so that both files are checked against one independent derivation.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// ---------------------------------------------------------------------------
// The free lists
// ---------------------------------------------------------------------------

// TestEffectTablesAreFreeListsNotZeroValues is the trap, and the failure mode is the quiet
// kind.
//
// Both producers scan for `Mode == -1`. A freshly constructed World has Mode == 0 in every
// slot, which is a *valid live cel index*, so the scan finds nowhere to put anything and both
// producers fall out of their loop having done nothing. The counters stay at 0, so nothing
// looks wrong from the outside: sparkles and score numerals simply never appear for the rest
// of the game.
//
// InitGarbageRects is the only sweeper in the program and it is the last of ReadyLevel's five
// steps, which is what makes the tables usable. The original has the same requirement --
// StructuresInit2.c NewPtr's both tables without clearing them -- so this is transcribed
// behaviour and not a porting artefact.
func TestEffectTablesAreFreeListsNotZeroValues(t *testing.T) {
	// Deliberately not dynaWorld: that fixture calls InitGarbageRects for exactly this
	// reason, so it would hide what this test is about.
	w := newTestWorld(oneRoomHouse(house.ObjectIsEmpty), "", "")

	for i := 0; i < MaxSparkles; i++ {
		if w.Sparkles[i].Mode != 0 {
			t.Fatalf("Sparkles[%d].Mode = %d on a fresh World, want the zero value 0 -- this "+
				"test is about the zero value not being the free marker", i, w.Sparkles[i].Mode)
		}
	}

	w.AddSparkle(testWhere)
	w.AddFlyingPoint(testWhere, 1000, 0, -2)

	if w.NumSparkles != 0 || w.NumFlyingPts != 0 {
		t.Errorf("(NumSparkles, NumFlyingPts) = (%d, %d) on an unswept table, want (0, 0): "+
			"both producers have to fail silently", w.NumSparkles, w.NumFlyingPts)
	}
	for i := 0; i < MaxSparkles; i++ {
		if w.Sparkles[i].Bounds != (Rect{}) {
			t.Errorf("Sparkles[%d].Bounds = %+v, want the untouched zero rect",
				i, w.Sparkles[i].Bounds)
		}
	}

	// And with the sweep, the same two calls land.
	w.InitGarbageRects()
	w.AddSparkle(testWhere)
	w.AddFlyingPoint(testWhere, 1000, 0, -2)
	if w.NumSparkles != 1 || w.NumFlyingPts != 1 {
		t.Errorf("(NumSparkles, NumFlyingPts) = (%d, %d) after the sweep, want (1, 1)",
			w.NumSparkles, w.NumFlyingPts)
	}
}

// TestAddSparkleFillsTheFirstFreeSlot: the scan takes the lowest free index, so slots are
// reused rather than cycled. A sparkle that ends in slot 1 is followed into slot 1 by the next
// one, even though slot 2 has been free longer.
func TestAddSparkleFillsTheFirstFreeSlot(t *testing.T) {
	w := dynaWorld(t)

	for i := 0; i < MaxSparkles; i++ {
		w.AddSparkle(testWhere)
		if w.Sparkles[i].Mode != 0 || w.NumSparkles != int16(i)+1 {
			t.Fatalf("after %d adds: Sparkles[%d].Mode = %d and NumSparkles = %d",
				i+1, i, w.Sparkles[i].Mode, w.NumSparkles)
		}
	}

	// Hand slot 1 back, as the renderer's retire path does, and watch the next add take it
	// rather than appending past slot 2.
	w.Sparkles[1].Mode = -1
	w.NumSparkles--

	w.AddSparkle(render.SetRect(0, 0, 40, 40))
	if w.Sparkles[1].Mode != 0 {
		t.Errorf("Sparkles[1].Mode = %d, want 0 -- the freed slot was skipped",
			w.Sparkles[1].Mode)
	}
	if w.NumSparkles != MaxSparkles {
		t.Errorf("NumSparkles = %d, want %d", w.NumSparkles, MaxSparkles)
	}
}

// TestAddSparkleDropsTheFourth: over the cap it silently does nothing, and the fourth request
// in a frame is dropped rather than queued.
//
// Reachable in the shipped game -- a room where two enemies retire on the frame the player
// collects a prize loses one of the three puffs -- and kept, because a queue would change what
// the player sees in exactly the busy rooms where the difference shows.
func TestAddSparkleDropsTheFourth(t *testing.T) {
	w := dynaWorld(t)
	for i := 0; i < MaxSparkles; i++ {
		w.AddSparkle(testWhere)
	}
	before := w.Sparkles

	w.AddSparkle(render.SetRect(0, 0, 40, 40))

	if w.NumSparkles != MaxSparkles {
		t.Errorf("NumSparkles = %d after a fourth add, want %d", w.NumSparkles, MaxSparkles)
	}
	if w.Sparkles != before {
		t.Errorf("the fourth add overwrote a live slot\nbefore %+v\nafter  %+v",
			before, w.Sparkles)
	}
}

// TestAddFlyingPointDropsTheFourth is the same cap on the other table. Three numerals is the
// whole budget, which is why a glider that flies through a row of prizes shows points for the
// first three and nothing for the rest until one expires.
func TestAddFlyingPointDropsTheFourth(t *testing.T) {
	w := dynaWorld(t)
	for i := 0; i < MaxFlyingPts; i++ {
		w.AddFlyingPoint(testWhere, 100, 0, -2)
	}
	before := w.FlyingPoints

	w.AddFlyingPoint(testWhere, 500, 3, -4)

	if w.NumFlyingPts != MaxFlyingPts {
		t.Errorf("NumFlyingPts = %d after a fourth add, want %d", w.NumFlyingPts, MaxFlyingPts)
	}
	if w.FlyingPoints != before {
		t.Errorf("the fourth add overwrote a live slot")
	}
}

// TestRoomChangeDropsLiveEffects: InitGarbageRects frees both tables outright, so effects do
// **not** follow the player through a door.
//
// That is why the score numerals floating up from a prize vanish the instant the room changes
// rather than drifting on in the new room, and why a sparkle interrupted by a door is simply
// gone. The function's name is about rectangles and three of its five jobs are not.
func TestRoomChangeDropsLiveEffects(t *testing.T) {
	w := dynaWorld(t)
	w.AddSparkle(testWhere)
	w.AddSparkle(testWhere)
	w.AddFlyingPoint(testWhere, 500, 2, -3)
	w.RenderSparkles()
	w.RenderFlyingPoints()
	if w.NumSparkles != 2 || w.NumFlyingPts != 1 {
		t.Fatalf("fixture: (NumSparkles, NumFlyingPts) = (%d, %d), want (2, 1)",
			w.NumSparkles, w.NumFlyingPts)
	}
	if work, back := rectCounts(w); work == 0 || back == 0 {
		t.Fatalf("fixture: rects (work %d, back %d), want both non-zero", work, back)
	}

	w.InitGarbageRects()

	if w.NumSparkles != 0 || w.NumFlyingPts != 0 {
		t.Errorf("(NumSparkles, NumFlyingPts) = (%d, %d) after the sweep, want (0, 0)",
			w.NumSparkles, w.NumFlyingPts)
	}
	for i := 0; i < MaxSparkles; i++ {
		if w.Sparkles[i].Mode != -1 {
			t.Errorf("Sparkles[%d].Mode = %d, want the free marker -1", i, w.Sparkles[i].Mode)
		}
	}
	for i := 0; i < MaxFlyingPts; i++ {
		if w.FlyingPoints[i].Mode != -1 {
			t.Errorf("FlyingPoints[%d].Mode = %d, want the free marker -1",
				i, w.FlyingPoints[i].Mode)
		}
	}
	if work, back := rectCounts(w); work != 0 || back != 0 {
		t.Errorf("rects (work %d, back %d) after the sweep, want (0, 0)", work, back)
	}
}

// ---------------------------------------------------------------------------
// AddSparkle's arithmetic
// ---------------------------------------------------------------------------

// TestAddSparkleCentresRatherThanPlaces is the one piece of arithmetic in AddSparkle, and it
// is why every call site passes an object's bounds instead of a point.
//
// The cel is 20x19 and CenterRectInRect keeps that size, so a sparkle over a 64-wide prize and
// one over a 24-wide balloon are the same size and both sit on their object's centre. Asserted
// against a rect built by hand rather than by re-invoking CenterIn, so that the *numbers* are
// pinned and not the composition.
func TestAddSparkleCentresRatherThanPlaces(t *testing.T) {
	cel := render.SparkleSrc[0]
	if cel.Wide() != 20 || cel.Tall() != 19 {
		t.Fatalf("SparkleSrc[0] is %dx%d, want the 20x19 the centring is written against",
			cel.Wide(), cel.Tall())
	}

	for _, where := range []Rect{
		render.SetRect(200, 100, 264, 130), // a wide prize
		render.SetRect(200, 100, 224, 130), // a narrow balloon, same centre column
		render.SetRect(0, 0, 1, 1),         // degenerate: the cel hangs off the top-left
	} {
		w := dynaWorld(t)
		oh, ov := w.R.V.OriginH, w.R.V.OriginV
		w.AddSparkle(where)

		// The producer offsets *then* centres, so the answer is the screen-space centre of
		// the object with a 20x19 box hung on it. Odd extents truncate the same way
		// CenterRectInRect does.
		cx := where.Left + oh + (where.Wide()-20)/2
		cy := where.Top + ov + (where.Tall()-19)/2
		want := Rect{Top: cy, Left: cx, Bottom: cy + 19, Right: cx + 20}

		if got := w.Sparkles[0].Bounds; got != want {
			t.Errorf("where %+v: Bounds = %+v, want %+v", where, got, want)
		}
		if got := w.Sparkles[0].Bounds; got.Wide() != 20 || got.Tall() != 19 {
			t.Errorf("where %+v: Bounds is %dx%d, want the cel's own 20x19 -- it is centred, "+
				"not stretched", where, got.Wide(), got.Tall())
		}
	}
}

// TestSparkleBoundsAreScreenSpaceAlready: both producers store an offset rect, so neither
// renderer offsets anything. That is the opposite of the six movers and the same as the seven
// appliances, and it is why enemyRetire offsets its work rect on one line and passes the
// sparkle an un-offset rect on the next.
//
// A regression here would put every puff one whole play origin -- (64, 79) on the default view
// -- down and to the right of the thing it is meant to mark.
func TestSparkleBoundsAreScreenSpaceAlready(t *testing.T) {
	w := dynaWorld(t)
	oh, ov := w.R.V.OriginH, w.R.V.OriginV
	w.AddSparkle(testWhere)
	roomLocal := render.CenterIn(render.SparkleSrc[0], testWhere)

	if got := w.Sparkles[0].Bounds; got == roomLocal {
		t.Fatalf("Bounds = %+v, which is the room-local rect: the play origin was never added",
			got)
	}
	if got, want := w.Sparkles[0].Bounds, render.Offset(roomLocal, oh, ov); got != want {
		t.Errorf("Bounds = %+v, want the room-local rect plus the origin = %+v", got, want)
	}

	// And the renderer leaves it alone: the rect it registers is the stored one, not the
	// stored one offset again.
	clearRects(w)
	w.RenderSparkles()
	if got, want := w.Work2Main[0], w.Sparkles[0].Bounds; got != want {
		t.Errorf("work rect = %+v, want Bounds unchanged = %+v", got, want)
	}
}

// ---------------------------------------------------------------------------
// RenderSparkles
// ---------------------------------------------------------------------------

// TestSparkleCelsAreAPalindrome pins the aliasing the five-cel table is built on.
//
// Only three distinct pictures exist: [0] aliases [4] and [1] aliases [3]
// (StructuresInit.c:397-401), so the strip plays out and back and a sparkle grows and shrinks
// from one set of three. Mode is the index into that palindrome, which is why the cap is
// NumSparkleModes rather than a count of pictures.
func TestSparkleCelsAreAPalindrome(t *testing.T) {
	if len(render.SparkleSrc) != int(NumSparkleModes) {
		t.Fatalf("SparkleSrc has %d cels but NumSparkleModes is %d; the renderer walks one "+
			"with the other", len(render.SparkleSrc), NumSparkleModes)
	}
	if render.SparkleSrc[0] != render.SparkleSrc[4] {
		t.Errorf("SparkleSrc[0] = %+v and [4] = %+v, want them aliased",
			render.SparkleSrc[0], render.SparkleSrc[4])
	}
	if render.SparkleSrc[1] != render.SparkleSrc[3] {
		t.Errorf("SparkleSrc[1] = %+v and [3] = %+v, want them aliased",
			render.SparkleSrc[1], render.SparkleSrc[3])
	}
	distinct := map[Rect]bool{}
	for _, r := range render.SparkleSrc {
		distinct[r] = true
	}
	if len(distinct) != 3 {
		t.Errorf("%d distinct cels, want 3 pictures played out and back", len(distinct))
	}
}

// TestSparkleCostsSixFramesAndElevenRects pins the whole life of the simplest renderer in the
// game: five drawing frames at two rects each, then one frame that only erases.
//
// Eleven rects out of the 47 usable slots for one puff is the practical reason the cap is
// three: three at once is a third of the frame's dirty-rect budget. Worth having in a test
// because a later efficiency pass will be tempted by the fact that Bounds never moves, and the
// second rect per frame is what stages the erase.
func TestSparkleCostsSixFramesAndElevenRects(t *testing.T) {
	w := dynaWorld(t)
	w.AddSparkle(testWhere)
	bounds := w.Sparkles[0].Bounds
	clearRects(w)

	var perFrame [][2]int
	frames := 0
	for ; frames < 20 && w.Sparkles[0].Mode != -1; frames++ {
		work, back := rectCounts(w)
		w.RenderSparkles()
		nowWork, nowBack := rectCounts(w)
		perFrame = append(perFrame, [2]int{nowWork - work, nowBack - back})
	}

	if frames != 6 {
		t.Errorf("a sparkle lived %d frames, want 6 (five cels plus the erase)", frames)
	}
	want := [][2]int{{1, 1}, {1, 1}, {1, 1}, {1, 1}, {1, 1}, {1, 0}}
	if len(perFrame) != len(want) {
		t.Fatalf("rects per frame %v, want %v", perFrame, want)
	}
	for i := range want {
		if perFrame[i] != want[i] {
			t.Fatalf("rects per frame %v, want %v", perFrame, want)
		}
	}
	if work, _ := rectCounts(w); work != 6 {
		t.Errorf("%d work rects in total, want 6", work)
	}
	if _, back := rectCounts(w); back != 5 {
		t.Errorf("%d back rects in total, want 5", back)
	}

	// Every one of the eleven is the same rect: a sparkle is the one effect with no velocity,
	// so Dest and Whole are the same thing and one rect serves both lists.
	for i, r := range w.Work2Main {
		if r != bounds {
			t.Errorf("work rect %d = %+v, want the unmoving %+v", i, r, bounds)
		}
	}
	if w.NumSparkles != 0 {
		t.Errorf("NumSparkles = %d after the erase frame, want 0", w.NumSparkles)
	}
}

// TestRenderSparklesEarlyOutsOnTheCounter pins which of the two representations wins when they
// disagree.
//
// The counter is redundant with the table -- NumSparkles is always the number of slots whose
// Mode is not -1 -- and it is kept because it is what the early-out reads. So a table with a
// live slot and a zero counter draws nothing at all, and the slot is never handed back either:
// the sparkle is stuck. Only InitGarbageRects can recover it, which it does by clearing the
// table rather than the counter.
//
// Not a bug to fix, but the reason the two must be written together, and the failure a future
// refactor that dropped the counter would silently repair while changing the frame cost of an
// idle room.
func TestRenderSparklesEarlyOutsOnTheCounter(t *testing.T) {
	w := dynaWorld(t)
	w.AddSparkle(testWhere)
	w.NumSparkles = 0 // the counter and the table now disagree
	clearRects(w)

	w.RenderSparkles()

	if work, back := rectCounts(w); work != 0 || back != 0 {
		t.Errorf("rects (work %d, back %d), want nothing: the counter is the early-out",
			work, back)
	}
	if w.Sparkles[0].Mode != 0 {
		t.Errorf("Sparkles[0].Mode = %d, want the stranded 0 -- the slot cannot be recovered "+
			"by the renderer", w.Sparkles[0].Mode)
	}
}

// ---------------------------------------------------------------------------
// AddFlyingPoint's strip selection
// ---------------------------------------------------------------------------

// TestFlyingPointStripSelection is how one 24x120 sheet spells five different numbers: each
// value owns three consecutive cels, in descending order of value.
func TestFlyingPointStripSelection(t *testing.T) {
	cases := []struct {
		points      int16
		start, stop int16
	}{
		{1000, 0, 2},
		{500, 3, 5},
		{300, 6, 8},
		{250, 9, 11},
		{100, 12, 14},
	}

	for _, tc := range cases {
		w := dynaWorld(t)
		w.AddFlyingPoint(testWhere, tc.points, 0, -2)
		p := w.FlyingPoints[0]

		if p.Start != tc.start || p.Stop != tc.stop {
			t.Errorf("%d points: (Start, Stop) = (%d, %d), want (%d, %d)",
				tc.points, p.Start, p.Stop, tc.start, tc.stop)
		}
		if p.Mode != tc.start {
			t.Errorf("%d points: Mode = %d, want Start = %d", tc.points, p.Mode, tc.start)
		}
		if p.Stop-p.Start != 2 {
			t.Errorf("%d points: %d cels, want 3", tc.points, p.Stop-p.Start+1)
		}
		if int(p.Stop) >= len(render.PointsSrc) {
			t.Errorf("%d points: Stop = %d is past the %d-cel strip",
				tc.points, p.Stop, len(render.PointsSrc))
		}
	}

	// The five ranges tile the strip exactly, with nothing spare: fifteen cels, five numbers,
	// three each. That is what makes an out-of-range Mode the immediate consequence of an
	// off-by-one in the walk -- see TestFlyingPointNeverIndexesPastTheStrip.
	covered := map[int16]bool{}
	for _, tc := range cases {
		for cel := tc.start; cel <= tc.stop; cel++ {
			if covered[cel] {
				t.Errorf("cel %d is claimed by two numbers", cel)
			}
			covered[cel] = true
		}
	}
	if len(covered) != len(render.PointsSrc) {
		t.Errorf("the five ranges cover %d of %d cels, want all of them",
			len(covered), len(render.PointsSrc))
	}
}

// TestFlyingPointDefaultArmIsNotAFallback: the switch has no error case, so any value that is
// not one of the four named ones draws the 1000 art.
//
// That is reached deliberately -- a kCuckoo is worth 1000 and gets there through the default --
// and accidentally: a kInvisBonus is worth whatever the author typed, so an author who types
// 700 gets a numeral that reads "1000". Transcribed as written, because it is visible in at
// least one shipped house and a level built around it would change if this were "fixed".
func TestFlyingPointDefaultArmIsNotAFallback(t *testing.T) {
	for _, points := range []int16{1000, 700, 1, 0, -5, 32767} {
		w := dynaWorld(t)
		w.AddFlyingPoint(testWhere, points, 0, -2)
		p := w.FlyingPoints[0]
		if p.Start != 0 || p.Stop != 2 {
			t.Errorf("%d points: (Start, Stop) = (%d, %d), want the 1000 art (0, 2)",
				points, p.Start, p.Stop)
		}
	}
}

// TestFlyingPointStartsCentredAndStationary: the producer centres the 24x8 cel on the object,
// seeds Whole equal to Dest so the first frame's union has something to grow from, and stores
// the velocities it was handed.
//
// The velocities are half the glider's at the moment of collection, which is why the numerals
// inherit the player's motion and a stationary glider's points rise straight up.
func TestFlyingPointStartsCentredAndStationary(t *testing.T) {
	cel := render.PointsSrc[0]
	if cel.Wide() != 24 || cel.Tall() != 8 {
		t.Fatalf("PointsSrc[0] is %dx%d, want 24x8", cel.Wide(), cel.Tall())
	}

	w := dynaWorld(t)
	oh, ov := w.R.V.OriginH, w.R.V.OriginV
	w.AddFlyingPoint(testWhere, 250, 3, -4)
	p := w.FlyingPoints[0]

	cx := testWhere.Left + oh + (testWhere.Wide()-24)/2
	cy := testWhere.Top + ov + (testWhere.Tall()-8)/2
	want := Rect{Top: cy, Left: cx, Bottom: cy + 8, Right: cx + 24}

	if p.Dest != want {
		t.Errorf("Dest = %+v, want %+v", p.Dest, want)
	}
	if p.Whole != p.Dest {
		t.Errorf("Whole = %+v, want Dest = %+v -- the first frame's union grows from it",
			p.Whole, p.Dest)
	}
	if p.HVel != 3 || p.VVel != -4 {
		t.Errorf("(HVel, VVel) = (%d, %d), want (3, -4)", p.HVel, p.VVel)
	}
	if p.Loops != 0 {
		t.Errorf("Loops = %d, want 0", p.Loops)
	}
}

// ---------------------------------------------------------------------------
// RenderFlyingPoints
// ---------------------------------------------------------------------------

// TestFlyingPointUnionGrowsAtTheLeadingEdge is the mirror image of
// TestMoverWholeIsTheTrailingUnion, and it produces the same rect by the opposite method.
//
// The movers recompute Whole from Dest every frame by pushing the trailing edge *back* one
// frame's travel. This grows Whole *forward* to the new leading edge and then, on the last
// line of the loop, resets it to Dest -- so Whole enters each frame holding the previous
// position and leaves it covering both.
//
// The two cannot be swapped. Accumulating is only safe because a numeral's velocity never
// changes; a mover that accumulated would keep a rect from before its last direction change
// forever. Asserted with movers_test.go's unionOf, so both files check against one derivation.
func TestFlyingPointUnionGrowsAtTheLeadingEdge(t *testing.T) {
	cases := []struct {
		name       string
		hVel, vVel int16
	}{
		{"up and to the right", 3, -2},
		{"down and to the left", -3, 2},
		{"straight up from a stationary glider", 0, -2},
		{"straight down", 0, 4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := dynaWorld(t)
			w.AddFlyingPoint(testWhere, 500, tc.hVel, tc.vVel)

			for frame := 0; frame < 5; frame++ {
				before := w.FlyingPoints[0].Dest
				clearRects(w)
				w.RenderFlyingPoints()
				after := w.FlyingPoints[0].Dest

				if w.FlyingPoints[0].Mode == -1 {
					t.Fatalf("frame %d: retired after five frames of a 72-frame life", frame)
				}
				// Whole is reset to Dest on the last line, so it has to be read from the
				// rects the frame registered rather than from the slot afterwards.
				if got, want := w.Work2Main[0], unionOf(before, after); got != want {
					t.Errorf("frame %d: work rect = %+v, want union(%+v, %+v) = %+v",
						frame, got, before, after, want)
				}
				if got, want := w.Back2Work[0], after; got != want {
					t.Errorf("frame %d: back rect = %+v, want the new Dest %+v",
						frame, got, want)
				}
				if got, want := w.FlyingPoints[0].Whole, after; got != want {
					t.Errorf("frame %d: Whole = %+v after the frame, want Dest = %+v so the "+
						"next union starts from here", frame, got, want)
				}
			}
		})
	}
}

// TestFlyingPointShowsThreeCelsPerLoop pins the ordering of the two tests at the top of the
// loop body: the `Mode > Stop` reset runs before the loop count is checked, and both run before
// the draw, so the frame that would have drawn cel Stop+1 draws Start instead.
//
// Read the (Mode, Loops) pairs one frame back to recover the cel that was drawn: Mode is
// incremented on the last line, so a frame that ends with Mode == Start+1 drew Start. The
// sequence therefore says the drawn cels are Start, Start+1, Start+2 repeating, three per loop
// and never four.
func TestFlyingPointShowsThreeCelsPerLoop(t *testing.T) {
	w := dynaWorld(t)
	w.AddFlyingPoint(testWhere, 300, 0, -2) // Start 6, Stop 8
	start, stop := w.FlyingPoints[0].Start, w.FlyingPoints[0].Stop
	if start != 6 || stop != 8 {
		t.Fatalf("fixture: (Start, Stop) = (%d, %d), want (6, 8)", start, stop)
	}

	want := [][2]int16{
		{7, 0}, {8, 0}, {9, 0}, // drew 6, 7, 8
		{7, 1}, {8, 1}, {9, 1}, // reset to 6, then 7, 8
		{7, 2}, {8, 2}, {9, 2},
	}
	for i, wantPair := range want {
		clearRects(w)
		w.RenderFlyingPoints()
		got := [2]int16{w.FlyingPoints[0].Mode, w.FlyingPoints[0].Loops}
		if got != wantPair {
			t.Fatalf("frame %d: (Mode, Loops) = %v, want %v", i+1, got, wantPair)
		}
		if drew := got[0] - 1; drew < start || drew > stop {
			t.Fatalf("frame %d drew cel %d, outside this number's %d..%d", i+1, drew, start, stop)
		}
	}
}

// TestFlyingPointNeverIndexesPastTheStrip is the reason the ordering above is not merely
// cosmetic.
//
// The 100-point numeral owns the last three cels of a fifteen-cel array, so Stop is 14 and a
// Mode of 15 would index one past the end. In Go that panics; in the original it read whatever
// followed the array and drew garbage. So an off-by-one in the reset is not a wrong cel, it is
// a crash -- and the only thing standing between the two is that the reset is checked before
// the draw.
//
// Needs real art, because the index is only evaluated inside the `art != nil` guard.
func TestFlyingPointNeverIndexesPastTheStrip(t *testing.T) {
	artDir := requireAssets(t, "art")
	w := newTestWorld(oneRoomHouse(house.ObjectIsEmpty), artDir, "")
	w.InitGarbageRects()
	if w.R.A.Strip("points") == nil {
		t.Skip("no points strip in the extracted art, so the index is never evaluated")
	}

	w.AddFlyingPoint(testWhere, 100, 0, -2) // Start 12, Stop 14: hard against the end
	if w.FlyingPoints[0].Stop != int16(len(render.PointsSrc))-1 {
		t.Fatalf("Stop = %d, want the last cel %d; this test is about the boundary",
			w.FlyingPoints[0].Stop, len(render.PointsSrc)-1)
	}

	// The whole life, drawing every frame. A panic here is the failure.
	for frame := 0; frame < 200 && w.FlyingPoints[0].Mode != -1; frame++ {
		clearRects(w)
		w.RenderFlyingPoints()
	}
	if w.FlyingPoints[0].Mode != -1 {
		t.Errorf("the numeral never retired")
	}
}

// TestFlyingPointLivesSeventyTwoFrames: twenty-four replays of three cels, so a score numeral
// is on screen for 72 frames -- about two and a half seconds -- drifting the whole time.
//
// The count is exact and the arithmetic is easy to get wrong by one, because Loops is
// incremented on the frame the cel index wraps rather than on the frame after: Loops reaches k
// on frame 3k+1, so it reaches MaxFlyingPointsLoop on frame 73 and that frame erases instead of
// drawing.
func TestFlyingPointLivesSeventyTwoFrames(t *testing.T) {
	w := dynaWorld(t)
	w.AddFlyingPoint(testWhere, 1000, 0, -2)

	drawn := 0
	retiredOn := -1
	for frame := 1; frame <= 200 && retiredOn == -1; frame++ {
		before := w.FlyingPoints[0].Dest
		clearRects(w)
		w.RenderFlyingPoints()
		if w.FlyingPoints[0].Mode == -1 {
			retiredOn = frame
			if w.FlyingPoints[0].Dest != before {
				t.Errorf("the retire frame also moved the numeral, from %+v to %+v",
					before, w.FlyingPoints[0].Dest)
			}
		} else {
			drawn++
		}
	}

	if drawn != 72 {
		t.Errorf("%d drawing frames, want 72 (%d loops of 3)", drawn, MaxFlyingPointsLoop)
	}
	if retiredOn != 73 {
		t.Errorf("retired on frame %d, want 73", retiredOn)
	}
	if w.NumFlyingPts != 0 {
		t.Errorf("NumFlyingPts = %d, want 0", w.NumFlyingPts)
	}
}

// TestFlyingPointRetireErasesDestNotWhole pins the last frame, which does three things and
// draws nothing.
//
// It registers **Dest** -- not Whole -- on the work list, which is the final erase, then frees
// the slot. Whole would also work here and would be one rect's worth larger; Dest is what the
// original registers, and it is correct because the numeral has already stopped: the previous
// frame's `Whole = Dest` left the two equal in everything but the last move, and that move's
// trail was covered by the previous frame's own work rect.
func TestFlyingPointRetireErasesDestNotWhole(t *testing.T) {
	w := dynaWorld(t)
	w.AddFlyingPoint(testWhere, 500, 3, -2)

	// Jump to the last frame rather than running 73: Mode is left at Start so the reset arm is
	// not taken, and the loop count is what retires it.
	w.FlyingPoints[0].Loops = MaxFlyingPointsLoop
	w.FlyingPoints[0].Whole = Rect{Top: 0, Left: 0, Bottom: 400, Right: 400}
	dest := w.FlyingPoints[0].Dest
	clearRects(w)

	w.RenderFlyingPoints()

	if work, back := rectCounts(w); work != 1 || back != 0 {
		t.Fatalf("rects (work %d, back %d), want (1, 0)", work, back)
	}
	if got := w.Work2Main[0]; got != dest {
		t.Errorf("work rect = %+v, want Dest %+v and not the deliberately oversized Whole",
			got, dest)
	}
	if w.FlyingPoints[0].Dest != dest {
		t.Errorf("Dest = %+v, want it unmoved at %+v", w.FlyingPoints[0].Dest, dest)
	}
	if w.FlyingPoints[0].Mode != -1 || w.NumFlyingPts != 0 {
		t.Errorf("(Mode, NumFlyingPts) = (%d, %d), want (-1, 0)",
			w.FlyingPoints[0].Mode, w.NumFlyingPts)
	}
}

// TestRenderFlyingPointsEarlyOutsOnTheCounter is RenderSparkles' counter trap on the other
// table, and it is worth its own test because the two renderers are the only readers of the two
// counters: everything else in the game works off Mode.
func TestRenderFlyingPointsEarlyOutsOnTheCounter(t *testing.T) {
	w := dynaWorld(t)
	w.AddFlyingPoint(testWhere, 100, 0, -2)
	dest := w.FlyingPoints[0].Dest
	w.NumFlyingPts = 0
	clearRects(w)

	w.RenderFlyingPoints()

	if work, back := rectCounts(w); work != 0 || back != 0 {
		t.Errorf("rects (work %d, back %d), want nothing", work, back)
	}
	if w.FlyingPoints[0].Dest != dest {
		t.Errorf("Dest = %+v, want it unmoved at %+v", w.FlyingPoints[0].Dest, dest)
	}
}

// TestEffectsRunIndependentlyPerSlot: three numerals started on different frames with different
// velocities each keep their own cel walk, loop count and union. The renderers are per-slot
// loops with no shared state, and the only thing they share is the rect lists.
func TestEffectsRunIndependentlyPerSlot(t *testing.T) {
	w := dynaWorld(t)
	w.AddFlyingPoint(testWhere, 1000, 2, -2) // slot 0, cels 0..2
	for i := 0; i < 5; i++ {
		clearRects(w)
		w.RenderFlyingPoints()
	}
	w.AddFlyingPoint(testWhere, 100, -2, 3) // slot 1, cels 12..14

	if w.NumFlyingPts != 2 {
		t.Fatalf("NumFlyingPts = %d, want 2", w.NumFlyingPts)
	}
	if w.FlyingPoints[1].Loops != 0 {
		t.Errorf("the new numeral inherited Loops = %d", w.FlyingPoints[1].Loops)
	}

	for i := 0; i < 4; i++ {
		clearRects(w)
		w.RenderFlyingPoints()
		if a, b := w.FlyingPoints[0], w.FlyingPoints[1]; a.Mode-a.Start == b.Mode-b.Start &&
			a.Loops == b.Loops {
			t.Errorf("frame %d: the two slots are in lockstep (%d/%d and %d/%d)",
				i, a.Mode-a.Start, a.Loops, b.Mode-b.Start, b.Loops)
		}
	}

	// Opposite velocities, so the two drift apart rather than sharing a position.
	if w.FlyingPoints[0].Dest.Left <= w.FlyingPoints[1].Dest.Left {
		t.Errorf("the rightward numeral is at %d and the leftward one at %d",
			w.FlyingPoints[0].Dest.Left, w.FlyingPoints[1].Dest.Left)
	}
	if w.FlyingPoints[0].Dest.Top >= w.FlyingPoints[1].Dest.Top {
		t.Errorf("the rising numeral is at %d and the falling one at %d",
			w.FlyingPoints[0].Dest.Top, w.FlyingPoints[1].Dest.Top)
	}
}
