package game

// The shredded glider: the four-slot table, the cloud that grows out of a shredder slot and
// falls, and the two places the original is wrong.
//
// No art and no house. RenderShreds is a pure function of a Shred, the play origin and the
// sheet, and with no sheet loaded every blit is skipped while the whole animation -- the two
// arms, the rect walk, the sounds, the sparkle and the dirty rects -- runs unchanged. That is
// the same bargain grease_test.go and dynamics_test.go make.
//
// The replay golden does not cover any of this: the recorded script never flies a glider into
// a shredder, which is how the `b2w` column stayed byte-identical when this subsystem landed.
// So these tests are the only thing holding it.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/render"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// shredWorld is dynaWorld with the sound player recording, because both arms of RenderShreds
// are as much a sound as a picture -- 35 restarts of kShredSound and one kFadeOutSound.
func shredWorld(t *testing.T) (*World, *[][2]int16) {
	t.Helper()
	w := dynaWorld(t)
	var heard [][2]int16
	w.SoundPlayer = func(sound, priority int16) {
		heard = append(heard, [2]int16{sound, priority})
	}
	return w, &heard
}

// gliderAt is a glider-sized rect, which is what AddAShreddedGlider takes: the position the
// last of the sprite disappeared into the slot at.
func gliderAt(h, v int16) player.Rect {
	return player.Rect{
		Top: v, Left: h,
		Bottom: v + player.GliderHigh, Right: h + player.GliderWide,
	}
}

// ---------------------------------------------------------------------------
// The table
// ---------------------------------------------------------------------------

// TestAShreddedGliderStartsWithZeroHeight pins the four offsets, and the odd one is the
// height.
//
// Left and Top are the glider's plus (4, 14) -- the slot is inset from the sprite -- and the
// width is the sprite's 40. But Bottom is set *equal* to Top, so a fresh cloud has no height
// at all and the growth arm's first act is to add one row. A fixture that gave it a height
// would skip the first frame of the animation and nothing would look wrong.
func TestAShreddedGliderStartsWithZeroHeight(t *testing.T) {
	w, _ := shredWorld(t)
	w.AddAShreddedGlider(gliderAt(100, 200))

	if w.NumShredded != 1 {
		t.Fatalf("NumShredded = %d, want 1", w.NumShredded)
	}
	want := Rect{Top: 214, Left: 104, Bottom: 214, Right: 144}
	if got := w.Shreds[0].Bounds; got != want {
		t.Errorf("Bounds = %+v, want %+v: (+4, +14) from the glider, 40 wide, no height",
			got, want)
	}
	if w.Shreds[0].Frame != 0 {
		t.Errorf("Frame = %d, want 0", w.Shreds[0].Frame)
	}
}

// TestTheBoundsStayRoomLocalAndTheDrawIsOffset is the coordinate-space claim, and it is the
// one every rect in this file depends on.
//
// AddAShreddedGlider takes the glider's Dest, which is room-local -- RenderGlider is what adds
// the play origin, not the physics -- and stores it unchanged. RenderShreds then offsets a
// *copy* for the draw and the dirty rects. So Bounds is room-local for the cloud's whole life
// and the rect that reaches the screen is Bounds plus the origin, which is what lets the cloud
// survive a scroll.
//
// The failure this guards is the natural one: offsetting in place, once, at the top of the
// frame. That draws the first frame in the right place and then walks the cloud away from the
// shredder by a play origin per frame.
func TestTheBoundsStayRoomLocalAndTheDrawIsOffset(t *testing.T) {
	w, _ := shredWorld(t)
	w.AddAShreddedGlider(gliderAt(100, 200))
	w.RenderShreds()

	// One frame of growth: a single row, room-local, at the glider's offsets.
	wantBounds := Rect{Top: 214, Left: 104, Bottom: 215, Right: 144}
	if got := w.Shreds[0].Bounds; got != wantBounds {
		t.Errorf("Bounds = %+v, want %+v: still room-local after a frame", got, wantBounds)
	}
	if len(w.Back2Work) != 1 {
		t.Fatalf("%d back rects, want 1", len(w.Back2Work))
	}
	want := render.Offset(wantBounds, w.R.V.OriginH, w.R.V.OriginV)
	if got := w.Back2Work[0]; got != want {
		t.Errorf("back rect = %+v, want %+v -- Bounds plus the play origin (%d, %d)",
			got, want, w.R.V.OriginH, w.R.V.OriginV)
	}
}

// TestTheFifthCloudIsDroppedRatherThanWrittenOutOfBounds is the port's one divergence in this
// file, and it is a memory-safety divergence rather than a behavioural one.
//
// The original guards with `if (numShredded > kMaxShredded) return;` -- strictly greater --
// with kMaxShredded == 4. So a call with numShredded already 4 passes the guard and writes
// shreds[4] of a four-element NewPtr (Environ.c:654), then leaves the counter at 5. Reaching
// it needs four live clouds and a fifth glider shredded before any of them is removed, which
// is why it shipped: a room with two shredders and a player determined to use them.
//
// The port tests `>=`, through badIndex so that the refusal is *reported* rather than silent:
// a bug report that says the build declined a write the 1994 one performed is worth more than
// a cloud that failed to appear. Reproducing the overrun would mean reproducing a bug whose
// observable behaviour is "corrupt whatever the allocator put next", which is not a behaviour
// a port can be faithful to. docs/IMPROVEMENTS.md 2.43 records it.
func TestTheFifthCloudIsDroppedRatherThanWrittenOutOfBounds(t *testing.T) {
	w, _ := shredWorld(t)
	for i := int16(0); i < MaxShredded; i++ {
		w.AddAShreddedGlider(gliderAt(100+i*8, 200))
	}
	if w.NumShredded != MaxShredded {
		t.Fatalf("NumShredded = %d after four, want %d", w.NumShredded, MaxShredded)
	}
	if w.Diag.Guarded != 0 {
		t.Fatalf("four clouds recorded %d deviations, want none: the first four are the "+
			"C's own behaviour", w.Diag.Guarded)
	}
	fourth := w.Shreds[MaxShredded-1]

	w.AddAShreddedGlider(gliderAt(999, 999))

	if w.NumShredded != MaxShredded {
		t.Errorf("NumShredded = %d after a fifth, want %d: the guard is >= and the fifth "+
			"cloud is dropped", w.NumShredded, MaxShredded)
	}
	if w.Shreds[MaxShredded-1] != fourth {
		t.Errorf("the fifth call overwrote slot %d: %+v -> %+v",
			MaxShredded-1, fourth, w.Shreds[MaxShredded-1])
	}
	// And it is on the record. This is the only guarded *write* in the port, so the kind is
	// asserted by name: a refusal filed under some other array would read, in a bug report,
	// as a malformed house rather than as the cap being reached.
	want := Deviation{Kind: devShred, Index: int(MaxShredded), Limit: int(MaxShredded)}
	if w.Diag.Guarded != 1 || len(w.Diag.Seen) != 1 || w.Diag.Seen[0] != want {
		t.Errorf("the fifth cloud recorded guarded=%d seen=%v, want one %v",
			w.Diag.Guarded, w.Diag.Seen, want)
	}
}

// ---------------------------------------------------------------------------
// RemoveShreds
// ---------------------------------------------------------------------------

// TestRemoveShredsRemovesExactlyOne is the second place the original is wrong, and unlike the
// overrun this one is transcribed as-is.
//
// The name says "shreds" and the caller reads as a sweep -- OffAMortal calls it under
// `if (numShredded > 0)` -- but the body finds the single entry with the largest frame and
// swap-removes it. A death with two clouds on screen therefore leaves one behind, still
// animating, while the new glider fades in over the top of it.
//
// Kept because it is visible in the original, reachable in any house with two shredders in one
// room, and because "fixing" it changes what a replay looks like. The swap is asserted too:
// the last live entry is copied over the chosen one, so the surviving cloud changes index.
func TestRemoveShredsRemovesExactlyOne(t *testing.T) {
	w, _ := shredWorld(t)
	w.AddAShreddedGlider(gliderAt(100, 200))
	w.AddAShreddedGlider(gliderAt(300, 200))
	w.AddAShreddedGlider(gliderAt(500, 200))
	w.Shreds[0].Frame = 5
	w.Shreds[1].Frame = 19 // the most advanced
	w.Shreds[2].Frame = 3
	survivor0, survivor2 := w.Shreds[0], w.Shreds[2]

	w.RemoveShreds()

	if w.NumShredded != 2 {
		t.Fatalf("NumShredded = %d, want 2: one cloud removed, not the table cleared",
			w.NumShredded)
	}
	if w.Shreds[0] != survivor0 {
		t.Errorf("slot 0 moved: %+v -> %+v", survivor0, w.Shreds[0])
	}
	if w.Shreds[1] != survivor2 {
		t.Errorf("slot 1 = %+v, want the swapped-down slot 2 %+v", w.Shreds[1], survivor2)
	}
	// The vacated slot keeps its rect and loses only its frame, which is why
	// AddAShreddedGlider writes all four bounds fields rather than offsetting them.
	if w.Shreds[2].Frame != 0 {
		t.Errorf("the vacated slot's Frame = %d, want 0", w.Shreds[2].Frame)
	}
}

// TestRemoveShredsRemovesNothingFromAGrowingCloud is the consequence of `frame > largest`
// starting from largest = 0, and it is the sharp edge of the two.
//
// An entry still in its growth arm has frame == 0, which can never beat a `largest` that
// starts at 0. So a death with exactly one cloud on screen and that cloud still growing
// removes **nothing**: OffAMortal's guard fires, RemoveShreds no-ops, and the confetti keeps
// growing through the respawn.
//
// Initialising `largest` to -1 would be a one-character fix and would change this. It stays,
// because 35 frames of a cloud that outlives its glider is what the original shows.
func TestRemoveShredsRemovesNothingFromAGrowingCloud(t *testing.T) {
	w, _ := shredWorld(t)
	w.AddAShreddedGlider(gliderAt(100, 200))
	before := w.Shreds[0]

	w.RemoveShreds()

	if w.NumShredded != 1 {
		t.Errorf("NumShredded = %d, want the unchanged 1: frame 0 can never beat a "+
			"`largest` seeded at 0", w.NumShredded)
	}
	if w.Shreds[0] != before {
		t.Errorf("the growing cloud changed: %+v -> %+v", before, w.Shreds[0])
	}

	// And it keeps animating afterwards, which is the visible half of the same fact.
	w.RenderShreds()
	if w.Shreds[0].Bounds.Bottom != before.Bounds.Bottom+1 {
		t.Error("the cloud stopped growing; the point is that it does not")
	}
}

// TestRemoveShredsOnAnEmptyTableIsSafe covers the who == -1 return with the counter at zero,
// which is the path a death in a room with no shredder in it takes -- OffAMortal's guard
// means it should not be reached at all, and it is guarded anyway.
func TestRemoveShredsOnAnEmptyTableIsSafe(t *testing.T) {
	w, _ := shredWorld(t)
	w.RemoveShreds()
	if w.NumShredded != 0 {
		t.Errorf("NumShredded = %d, want 0", w.NumShredded)
	}
}

// TestZeroShredsClearsTheCounterAndNotTheTable states the exact scope of DrawLocale's single
// assignment, because the stale rects it leaves behind are load-bearing for nothing and would
// be a bug if anything read them.
//
// Every slot is written in full by AddAShreddedGlider before it is read, so what is below the
// counter is unreachable. Asserting the rects survive is asserting that the port did not
// "helpfully" zero the array -- which would be harmless here and would diverge from the C in
// a place a future change might come to depend on.
func TestZeroShredsClearsTheCounterAndNotTheTable(t *testing.T) {
	w, _ := shredWorld(t)
	w.AddAShreddedGlider(gliderAt(100, 200))
	w.Shreds[0].Frame = 7
	stale := w.Shreds[0]

	w.ZeroShreds()

	if w.NumShredded != 0 {
		t.Errorf("NumShredded = %d, want 0", w.NumShredded)
	}
	if w.Shreds[0] != stale {
		t.Errorf("slot 0 was cleared as well: %+v -> %+v", stale, w.Shreds[0])
	}
}

// ---------------------------------------------------------------------------
// The animation
// ---------------------------------------------------------------------------

// TestTheGrowthArmEmergesBottomFirst walks all 35 frames of the first arm, and the assertion
// that matters is the *source* rect rather than the destination.
//
// The rect's bottom edge advances one pixel a frame with the top pinned, and the blit takes
// the bottom `high` rows of the 40x35 sprite -- `src.top = src.bottom - high`. So the confetti
// slides out of the slot bottom edge first, the way paper leaves a shredder. Take the top
// `high` rows instead and it emerges upside down, which is a one-character change and looks
// almost right.
//
// Src is a local in RenderShreds and never reaches a field, so what is asserted here is the
// height that drives it, one frame at a time, plus the invariant the arithmetic rests on:
// ShredGrowHeight and the sheet's own height are the same number, which is what makes the
// growth end exactly as the last row of the sprite becomes visible. If those two ever
// disagree the arm either stops early or reads above the top of the sheet.
func TestTheGrowthArmEmergesBottomFirst(t *testing.T) {
	if got := ShredSrc.Bottom - ShredSrc.Top; got != ShredGrowHeight {
		t.Fatalf("the shred sheet is %d tall and the growth arm runs to %d; the src "+
			"arithmetic assumes they are equal", got, ShredGrowHeight)
	}

	w, heard := shredWorld(t)
	w.AddAShreddedGlider(gliderAt(100, 200))
	top := w.Shreds[0].Bounds.Top

	for frame := int16(1); frame <= ShredGrowHeight; frame++ {
		w.RenderShreds()
		s := w.Shreds[0]

		if s.Bounds.Top != top {
			t.Fatalf("frame %d: Top moved to %d, want the pinned %d",
				frame, s.Bounds.Top, top)
		}
		if got := s.Bounds.Bottom - s.Bounds.Top; got != frame {
			t.Fatalf("frame %d: height = %d, want %d (one row a frame)", frame, got, frame)
		}
		wantFrame := int16(0)
		if frame == ShredGrowHeight {
			wantFrame = 1 // the last growth frame also promotes to the fall arm
		}
		if s.Frame != wantFrame {
			t.Fatalf("frame %d: Frame = %d, want %d", frame, s.Frame, wantFrame)
		}
	}

	// kShredSound on every one of the 35, not once. Thirty-five restarts of a one-shot is
	// what makes the noise continuous, and it is why the shred holds a channel for over a
	// second at priority 903.
	if len(*heard) != int(ShredGrowHeight) {
		t.Fatalf("heard %d sounds in 35 growth frames, want 35", len(*heard))
	}
	for i, s := range *heard {
		if s != [2]int16{player.ShredSound, player.ShredPriority} {
			t.Fatalf("growth sound %d = %v, want {%d %d}",
				i, s, player.ShredSound, player.ShredPriority)
		}
	}
}

// TestTheFallArmDropsFourPixelsAFrame walks the second arm and ends on the sparkle.
//
// Nineteen frames, and the nineteenth is the one whose increment reaches 20: it moves the
// cloud a last time, draws nothing, and emits the sparkle at the position it moved to. That
// shared iteration is why the whole animation is 54 frames and not 55 -- see the file comment
// on shreds.go, which counted it twice until this test was written.
func TestTheFallArmDropsFourPixelsAFrame(t *testing.T) {
	w, heard := shredWorld(t)
	w.AddAShreddedGlider(gliderAt(100, 200))
	for i := int16(0); i < ShredGrowHeight; i++ {
		w.RenderShreds() // through the growth arm
	}
	*heard = nil
	start := w.Shreds[0].Bounds

	for step := int16(1); step <= ShredLastFrame-1; step++ {
		w.RenderShreds()
		s := w.Shreds[0]

		if s.Bounds.Top != start.Top+4*step {
			t.Fatalf("fall %d: Top = %d, want %d", step, s.Bounds.Top, start.Top+4*step)
		}
		if s.Bounds.Bottom != start.Bottom+4*step {
			t.Fatalf("fall %d: Bottom = %d, want %d -- the whole rect moves, unlike the "+
				"growth arm", step, s.Bounds.Bottom, start.Bottom+4*step)
		}
		if got := s.Bounds.Bottom - s.Bounds.Top; got != ShredGrowHeight {
			t.Fatalf("fall %d: height = %d, want the full %d", step, got, ShredGrowHeight)
		}
		if s.Frame != step+1 {
			t.Fatalf("fall %d: Frame = %d, want %d", step, s.Frame, step+1)
		}
	}

	if w.Shreds[0].Frame != ShredLastFrame {
		t.Fatalf("Frame = %d after 19 fall frames, want %d",
			w.Shreds[0].Frame, ShredLastFrame)
	}
	// One sound in the whole fall, on the last frame.
	if len(*heard) != 1 || (*heard)[0] != [2]int16{player.FadeOutSound, player.FadeOutPriority} {
		t.Errorf("heard %v in the fall, want one {%d %d} on the last frame",
			*heard, player.FadeOutSound, player.FadeOutPriority)
	}
	// The sparkle is centred in the cloud's final position, not placed at its corner.
	if w.NumSparkles != 1 {
		t.Fatalf("NumSparkles = %d, want 1", w.NumSparkles)
	}
	final := render.Offset(w.Shreds[0].Bounds, w.R.V.OriginH, w.R.V.OriginV)
	puff := w.Sparkles[0].Bounds
	if puff != render.CenterIn(render.SparkleSrc[0], final) {
		t.Errorf("sparkle at %+v, want %v centred in the cloud's last rect %+v",
			puff, render.SparkleSrc[0], final)
	}
}

// TestASpentCloudIsInert covers the frame the two arms both decline, which is the state a
// cloud spends the rest of the locale in.
//
// Neither case matches at Frame 20, so the entry stops moving, stops drawing and stops
// registering rects -- but keeps its slot. That is the reason the four-slot cap bites harder
// than it looks: four spent clouds will refuse a fifth even with nothing on screen.
func TestASpentCloudIsInert(t *testing.T) {
	w, heard := shredWorld(t)
	w.AddAShreddedGlider(gliderAt(100, 200))
	w.Shreds[0].Frame = ShredLastFrame
	spent := w.Shreds[0]

	for i := 0; i < 20; i++ {
		w.RenderShreds()
	}

	if w.Shreds[0] != spent {
		t.Errorf("a spent cloud moved: %+v -> %+v", spent, w.Shreds[0])
	}
	if len(w.Work2Main) != 0 || len(w.Back2Work) != 0 {
		t.Errorf("a spent cloud registered %d work and %d back rects, want none",
			len(w.Work2Main), len(w.Back2Work))
	}
	if len(*heard) != 0 {
		t.Errorf("a spent cloud played %v", *heard)
	}
	// And it still holds its slot, which is the cap's real cost.
	if w.NumShredded != 1 {
		t.Errorf("NumShredded = %d, want 1: a spent cloud is not reclaimed", w.NumShredded)
	}
}

// TestTheWorkRectTrailsTheBackRect is the one place in the renderer where the two dirty rects
// are deliberately *different*, and the difference is the trail.
//
// The back rect is exactly where the sprite was drawn, so next frame restores the wall under
// it. The work rect is that rect extended backwards over where the sprite was *last* frame, so
// the screen copy covers the streak it would otherwise leave. Register the same rect for both
// and the cloud smears a four-pixel band of stale confetti down the screen behind itself.
//
// The extension is four pixels in the fall arm, which is exactly the step, and one pixel in
// the growth arm, which is not -- the growth arm's top edge never moves, so the extra row is
// above the cloud and always blank. Harmless overdraw, transcribed rather than trimmed.
func TestTheWorkRectTrailsTheBackRect(t *testing.T) {
	w, _ := shredWorld(t)
	w.AddAShreddedGlider(gliderAt(100, 200))

	check := func(t *testing.T, phase string, extend int16) {
		t.Helper()
		if len(w.Work2Main) != 1 || len(w.Back2Work) != 1 {
			t.Fatalf("%s: %d work and %d back rects, want one each",
				phase, len(w.Work2Main), len(w.Back2Work))
		}
		back, work := w.Back2Work[0], w.Work2Main[0]
		if work.Left != back.Left || work.Right != back.Right ||
			work.Bottom != back.Bottom {
			t.Errorf("%s: work %+v and back %+v differ on an edge other than the top",
				phase, work, back)
		}
		if work.Top != back.Top-extend {
			t.Errorf("%s: work rect top = %d, want %d (back %d extended back by %d)",
				phase, work.Top, back.Top-extend, back.Top, extend)
		}
		w.Work2Main, w.Back2Work = w.Work2Main[:0], w.Back2Work[:0]
	}

	w.RenderShreds()
	check(t, "growth", 1)

	for i := int16(1); i < ShredGrowHeight; i++ {
		w.RenderShreds()
		w.Work2Main, w.Back2Work = w.Work2Main[:0], w.Back2Work[:0]
	}

	w.RenderShreds()
	check(t, "fall", 4)

	for i := int16(2); i < ShredLastFrame-1; i++ {
		w.RenderShreds()
		w.Work2Main, w.Back2Work = w.Work2Main[:0], w.Back2Work[:0]
	}

	// The sparkle frame registers both even though it draws nothing: the back rect is a
	// no-op there, and the work rect is what pushes the previous frame's erase to screen.
	w.RenderShreds()
	check(t, "sparkle", 4)
}

// TestFourCloudsAnimateIndependently is the loop bound, which is `numShredded` and not the
// table length -- so a slot above the counter is never touched even though it holds a rect.
func TestFourCloudsAnimateIndependently(t *testing.T) {
	w, _ := shredWorld(t)
	for i := int16(0); i < 3; i++ {
		w.AddAShreddedGlider(gliderAt(100+i*64, 200))
		w.Shreds[i].Frame = i * 5 // three clouds at three different points
	}
	// A fourth slot, written and then abandoned above the counter.
	w.AddAShreddedGlider(gliderAt(400, 200))
	w.NumShredded = 3
	untouched := w.Shreds[3]

	w.RenderShreds()

	if w.Shreds[0].Bounds.Bottom-w.Shreds[0].Bounds.Top != 1 {
		t.Error("cloud 0 (frame 0) did not grow")
	}
	for i := int16(1); i < 3; i++ {
		if w.Shreds[i].Frame != i*5+1 {
			t.Errorf("cloud %d: Frame = %d, want %d", i, w.Shreds[i].Frame, i*5+1)
		}
	}
	if w.Shreds[3] != untouched {
		t.Errorf("the slot above the counter animated: %+v -> %+v",
			untouched, w.Shreds[3])
	}
	if len(w.Back2Work) != 3 {
		t.Errorf("%d back rects, want 3 -- one per live cloud", len(w.Back2Work))
	}
}
