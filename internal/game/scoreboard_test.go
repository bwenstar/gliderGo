package game

// The scoreboard: where it lands, when each badge blinks, and how the score rolls.
//
// Three of these tests need no art at all, because placement is arithmetic over the view.
// The blink and the band-on-screen tests do need it: they assert on *pixels in Main*, which
// is the only way to prove that a badge was drawn rather than merely that a function was
// called. Reproducing the C's control flow is not the goal -- putting the right cell of the
// right sheet at the right place on the right frame is.
//
// The last two tests exist because there is a font now. Before it, every panel was a flat
// gray patch and the two things the panels are actually for -- the glider count's clamp and
// the score roll's intermediate numbers -- were unobservable from outside: "0" and "-1" drew
// the same nothing. Both are asserted against a panel drawn from the string spelled out in
// the test, which is what makes them tests of the value chosen rather than of the drawing.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// scoreboardWorld is a world on the smallest house that will build, with art if `art` is
// non-empty. The house does not matter to any test here -- none of them composes a room --
// but NewWorld needs one, and it is NewWorld that seeds the scoreboard rects.
func scoreboardWorld(art string) *World {
	return newTestWorld(oneRoomHouse(house.ObjectIsEmpty), art, "")
}

// ---------------------------------------------------------------------------
// Placement
// ---------------------------------------------------------------------------

// TestScoreboardPlacementHigh pins the port's one scoreboard deviation.
//
// The original's High offset is -localRoomsDest[kCentralRoom].top, which moves the band from
// rows -20..0 to rows -99..-79 -- from just off the top of the window to further off it. The
// port's is +screenRect.bottom, which puts it in the twenty rows below the house rect: the
// one strip of a 640x480 screen that no room can draw into, because houseRect is the screen
// less exactly that many rows. Every one of the seven movable rects moves by the same amount,
// so the layout InitScoreboardMap built survives the move intact.
func TestScoreboardPlacementHigh(t *testing.T) {
	w := scoreboardWorld("")
	v := w.R.V

	w.placeScoreboard(ScoreboardHigh)

	// The band occupies exactly the strip between the house rect and the bottom of the
	// screen. Both halves matter: one row higher and it would cover a room's last row,
	// one row lower and it would be off screen -- which is the bug being fixed.
	wantRect(t, "BoardDestRect", w.BoardDestRect, render.SetRect(0, 460, 640, 480))
	if w.BoardDestRect.Top != v.House.Bottom || w.BoardDestRect.Bottom != v.Screen.Bottom {
		t.Errorf("band %v does not fill the strip between house %v and screen %v",
			w.BoardDestRect, v.House, v.Screen)
	}

	wantRect(t, "BoardGQDestRect", w.BoardGQDestRect, render.SetRect(526, 465, 546, 475))
	wantRect(t, "BoardPQDestRect", w.BoardPQDestRect, render.SetRect(570, 465, 634, 475))

	wantRect(t, "badge[foil]", w.BadgesDestRects[render.FoilBadge], render.SetRect(432, 462, 448, 478))
	wantRect(t, "badge[bands]", w.BadgesDestRects[render.BandsBadge], render.SetRect(449, 462, 465, 478))
	wantRect(t, "badge[battery]", w.BadgesDestRects[render.BatteryBadge], render.SetRect(467, 461, 483, 478))
	wantRect(t, "badge[helium]", w.BadgesDestRects[render.HeliumBadge], w.BadgesDestRects[render.BatteryBadge])

	// Everything the band carries has to be inside the band, or it lands on a room.
	for name, r := range map[string]render.Rect{
		"gliders": w.BoardGQDestRect, "score": w.BoardPQDestRect,
		"foil": w.BadgesDestRects[render.FoilBadge], "bands": w.BadgesDestRects[render.BandsBadge],
		"battery": w.BadgesDestRects[render.BatteryBadge],
	} {
		if s, ok := render.Sect(r, w.BoardDestRect); !ok || s != r {
			t.Errorf("%s panel %v escapes the band %v", name, r, w.BoardDestRect)
		}
	}

	// In this layout every room is drawn, so the whole work map is live.
	wantRect(t, "JustRoomsRect", v.JustRoomsRect, render.ZeroCorner(v.House))
}

// TestScoreboardPlacementLow is the original's small-screen layout, unchanged.
//
// The band goes in the black gap above the central room -- rows 59..79, since the room's top
// is 79 and the band is 20 tall -- and justRoomsRect shrinks to the central room's rows,
// which is what stops the whole-screen blit from touching eight neighbours that are not
// being drawn.
func TestScoreboardPlacementLow(t *testing.T) {
	w := scoreboardWorld("")
	v := w.R.V
	central := v.LocalRoomsDest[CentralRoom]

	w.placeScoreboard(ScoreboardLow)

	wantRect(t, "BoardDestRect", w.BoardDestRect, render.SetRect(0, 59, 640, 79))
	if w.BoardDestRect.Bottom != central.Top {
		t.Errorf("band %v does not sit directly above the central room %v", w.BoardDestRect, central)
	}

	wantRect(t, "BoardGQDestRect", w.BoardGQDestRect, render.SetRect(526, 64, 546, 74))
	wantRect(t, "BoardPQDestRect", w.BoardPQDestRect, render.SetRect(570, 64, 634, 74))
	wantRect(t, "badge[foil]", w.BadgesDestRects[render.FoilBadge], render.SetRect(432, 61, 448, 77))
	wantRect(t, "badge[bands]", w.BadgesDestRects[render.BandsBadge], render.SetRect(449, 61, 465, 77))
	wantRect(t, "badge[battery]", w.BadgesDestRects[render.BatteryBadge], render.SetRect(467, 60, 483, 77))

	wantRect(t, "JustRoomsRect", v.JustRoomsRect,
		render.SetRect(0, central.Top, v.House.Right, central.Bottom))
}

// TestScoreboardPlacementIsIdempotent is the property the C does not have.
//
// AdjustScoreboardHeight's body is seven QOffsetRects, so calling it twice with the same mode
// would move the board twice; the `wasScoreboardMode != newMode` guard is load-bearing for
// correctness and not just for speed. The port assigns instead of accumulating, which makes
// the guard an optimisation -- so a future caller (a resize, a second game in one process, a
// two-player restart) cannot push the board off screen by asking for what it already has.
func TestScoreboardPlacementIsIdempotent(t *testing.T) {
	w := scoreboardWorld("")

	for _, mode := range []int16{ScoreboardHigh, ScoreboardLow} {
		w.placeScoreboard(mode)
		board, gq, pq, badges := w.BoardDestRect, w.BoardGQDestRect, w.BoardPQDestRect, w.BadgesDestRects
		rooms := w.R.V.JustRoomsRect

		for i := 0; i < 3; i++ {
			w.placeScoreboard(mode)
		}

		if w.BoardDestRect != board || w.BoardGQDestRect != gq || w.BoardPQDestRect != pq ||
			w.BadgesDestRects != badges || w.R.V.JustRoomsRect != rooms {
			t.Errorf("mode %d: four calls do not agree with one: band %v then %v",
				mode, board, w.BoardDestRect)
		}
	}
}

// TestAdjustScoreboardHeightLatch covers the two things the latch has to get right: the early
// return has to leave the rects correct, and a real change has to move them.
//
// The early return is the case that would have been a bug had NewWorld not seeded the rects.
// wasScoreboardMode starts at High and numNeighbors defaults to 9, so on every normal screen
// AdjustScoreboardHeight returns without doing anything at all -- and the band has to already
// be where High wants it, not at the raw boardDestRect of rows -20..0.
func TestAdjustScoreboardHeightLatch(t *testing.T) {
	w := scoreboardWorld("")

	if w.WasScoreboardMode != ScoreboardHigh || w.R.NumNeighbors != 9 {
		t.Fatalf("expected the shipped default of High with 9 neighbours, got mode %d with %d",
			w.WasScoreboardMode, w.R.NumNeighbors)
	}

	w.AdjustScoreboardHeight() // returns early
	wantRect(t, "BoardDestRect after the early return", w.BoardDestRect, render.SetRect(0, 460, 640, 480))

	// A one- or three-neighbour view is the small-screen layout, and the board moves.
	w.R.NumNeighbors = 3
	w.AdjustScoreboardHeight()
	if w.WasScoreboardMode != ScoreboardLow {
		t.Errorf("mode = %d, want ScoreboardLow with 3 neighbours", w.WasScoreboardMode)
	}
	wantRect(t, "BoardDestRect at Low", w.BoardDestRect, render.SetRect(0, 59, 640, 79))

	w.AdjustScoreboardHeight()
	wantRect(t, "BoardDestRect after a repeat call", w.BoardDestRect, render.SetRect(0, 59, 640, 79))
}

// ---------------------------------------------------------------------------
// The blink
// ---------------------------------------------------------------------------

// TestBlinkSchedule walks one whole eight-frame cycle and checks, per frame, which badge slot
// was painted and with which cell of the sheet.
//
// This is the test that could not be written against the C's control flow, because the
// interesting part is not which function ran -- it is that frames 3 and 6 paint nothing, that
// the battery's show/hide pair is inverted relative to the other two, and that the hide half
// draws the *blank* cell rather than skipping the blit. All three are visible only in pixels.
func TestBlinkSchedule(t *testing.T) {
	art := requireAssets(t, "art")
	w := scoreboardWorld(art)
	sheet := w.Board.Badge
	if sheet == nil {
		t.Fatalf("no badge sheet: %v", w.R.A.Err())
	}

	// Every counter low but not empty, so all three badges are eligible to blink.
	w.Foil, w.Bands, w.Battery = 1, 1, 1

	const (
		none = -1
		sent = 200 // a palette index the sheet does not use in these cells
	)
	for _, c := range []struct {
		frame int64
		slot  int  // which badge destination was painted, or none
		cell  int  // which badge's cell was the source
		lit   bool // the lit column, or the blank one
	}{
		{frame: 0, slot: render.FoilBadge, cell: render.FoilBadge, lit: true},
		{frame: 1, slot: render.BatteryBadge, cell: render.BatteryBadge, lit: false},
		{frame: 2, slot: render.BandsBadge, cell: render.BandsBadge, lit: true},
		{frame: 3, slot: none},
		{frame: 4, slot: render.BatteryBadge, cell: render.BatteryBadge, lit: true},
		{frame: 5, slot: render.FoilBadge, cell: render.FoilBadge, lit: false},
		{frame: 6, slot: none},
		{frame: 7, slot: render.BandsBadge, cell: render.BandsBadge, lit: false},
	} {
		w.Main.Fill(w.Main.Bounds(), sent)
		w.Frame = c.frame
		w.HandleDynamicScoreboard()

		// The three destinations are distinct rects, so "painted" and "untouched" are
		// separable for each -- which is how the empty frames are provable.
		for _, slot := range []int{render.FoilBadge, render.BandsBadge, render.BatteryBadge} {
			dest := w.BadgesDestRects[slot]
			if slot != c.slot {
				if !regionIs(w.Main, dest, sent) {
					t.Errorf("frame %d: badge slot %d was painted and should not have been",
						c.frame, slot)
				}
				continue
			}
			src := w.R.V.BadgesBlank[c.cell]
			if c.lit {
				src = w.R.V.BadgesBadges[c.cell]
			}
			if !regionEquals(w.Main, dest, sheet, src) {
				t.Errorf("frame %d: badge slot %d is not the %s cell of badge %d",
					c.frame, slot, litness(c.lit), c.cell)
			}
		}
	}
}

// TestBlinkHelium is the same schedule with the counter negative, which is how the game
// stores helium: one signed number for two power-ups, so the *cell* changes and the
// destination does not.
//
// The hide frame is the transcribed oddity -- QuickBatteryRefresh blanks the battery cell
// even when it is helium that is lit -- and it is invisible only because
// render.TestBadgeBlankCellsAreIdentical holds. This test asserts the battery blank cell
// specifically, so if that ever stops being true, both tests fail rather than neither.
func TestBlinkHelium(t *testing.T) {
	art := requireAssets(t, "art")
	w := scoreboardWorld(art)
	sheet := w.Board.Badge
	if sheet == nil {
		t.Fatalf("no badge sheet: %v", w.R.A.Err())
	}

	w.Battery = -1 // one puff of helium left: negative, and above HeliumLow
	dest := w.BadgesDestRects[render.BatteryBadge]

	const sent = 200
	for _, c := range []struct {
		frame int64
		cell  int
		lit   bool
	}{
		{frame: 4, cell: render.HeliumBadge, lit: true},   // show: the helium badge
		{frame: 1, cell: render.BatteryBadge, lit: false}, // hide: the *battery* blank
	} {
		w.Main.Fill(w.Main.Bounds(), sent)
		w.Frame = c.frame
		w.HandleDynamicScoreboard()

		src := w.R.V.BadgesBlank[c.cell]
		if c.lit {
			src = w.R.V.BadgesBadges[c.cell]
		}
		if !regionEquals(w.Main, dest, sheet, src) {
			t.Errorf("frame %d: the battery slot is not the %s cell of badge %d",
				c.frame, litness(c.lit), c.cell)
		}
	}
}

// TestBlinkOnlyWhenLow pins both ends of every threshold, and there are eight of them.
//
// A badge blinks only while its counter is *both* non-zero and strictly below its low mark.
// The strictness is the off-by-one the constants document: two sheets of foil do not blink and
// one does, even though two is the 25% the original's comment claims. An empty badge stops
// blinking entirely and simply stays dark, which is a different thing from blinking slowly.
func TestBlinkOnlyWhenLow(t *testing.T) {
	art := requireAssets(t, "art")
	w := scoreboardWorld(art)
	if w.Board.Badge == nil {
		t.Fatalf("no badge sheet: %v", w.R.A.Err())
	}

	const sent = 200
	for _, c := range []struct {
		name       string
		foil       int16
		bands      int16
		battery    int16
		frame      int64
		wantPaint  bool
		wantedSlot int
	}{
		{name: "foil at 1 blinks", foil: 1, frame: 0, wantPaint: true, wantedSlot: render.FoilBadge},
		{name: "foil at FoilLow does not", foil: FoilLow, frame: 0, wantedSlot: render.FoilBadge},
		{name: "foil empty does not", foil: 0, frame: 0, wantedSlot: render.FoilBadge},
		{name: "bands at 1 blinks", bands: 1, frame: 2, wantPaint: true, wantedSlot: render.BandsBadge},
		{name: "bands at BandsLow does not", bands: BandsLow, frame: 2, wantedSlot: render.BandsBadge},
		{name: "battery at 16 blinks", battery: BatteryLow - 1, frame: 4, wantPaint: true, wantedSlot: render.BatteryBadge},
		{name: "battery at BatteryLow does not", battery: BatteryLow, frame: 4, wantedSlot: render.BatteryBadge},
		{name: "battery empty does not", battery: 0, frame: 4, wantedSlot: render.BatteryBadge},
		{name: "helium at -37 blinks", battery: HeliumLow + 1, frame: 4, wantPaint: true, wantedSlot: render.BatteryBadge},
		{name: "helium at HeliumLow does not", battery: HeliumLow, frame: 4, wantedSlot: render.BatteryBadge},
	} {
		w.Foil, w.Bands, w.Battery = c.foil, c.bands, c.battery
		w.Main.Fill(w.Main.Bounds(), sent)
		w.Frame = c.frame
		w.HandleDynamicScoreboard()

		painted := !regionIs(w.Main, w.BadgesDestRects[c.wantedSlot], sent)
		if painted != c.wantPaint {
			t.Errorf("%s: painted = %v, want %v", c.name, painted, c.wantPaint)
		}
	}
}

// ---------------------------------------------------------------------------
// The score roll
// ---------------------------------------------------------------------------

// TestScoreRoll counts the steps and the ticks.
//
// The clamp on the last step is what makes the displayed number land on the score exactly
// rather than overshoot and stick there -- 40 points is three full steps of 13 and a
// remainder of 1 -- and the tick has to fire once per redraw including that short last one,
// because it sits outside the DoRollScore branch and inside the `Score > DisplayedScore` test.
func TestScoreRoll(t *testing.T) {
	w := scoreboardWorld("")

	var ticks []int16
	w.SoundPlayer = func(sound, priority int16) {
		if sound == ScoreTikSound {
			if priority != ScoreTikPriority {
				t.Errorf("tick priority = %d, want %d", priority, ScoreTikPriority)
			}
			ticks = append(ticks, sound)
		}
	}

	w.Frame = 3 // an empty frame of the blink cycle, so only the roll runs
	w.Score, w.DisplayedScore, w.DoRollScore = 40, 0, true

	var steps []int32
	for i := 0; i < 6; i++ {
		w.HandleDynamicScoreboard()
		steps = append(steps, w.DisplayedScore)
	}

	for i, want := range []int32{13, 26, 39, 40, 40, 40} {
		if steps[i] != want {
			t.Errorf("step %d: displayed = %d, want %d (all: %v)", i, steps[i], want, steps)
			break
		}
	}
	if len(ticks) != 4 {
		t.Errorf("got %d ticks, want 4 -- one per redraw, and none once the roll has arrived",
			len(ticks))
	}
}

// TestScoreRollJumpsWhenDisarmed is the branch the shipped game cannot reach.
//
// Nothing anywhere in the original clears doRollScore, so the else arm is dead code there. It
// is transcribed because of where the tick and the redraw sit relative to it: a port that
// hoisted them into the roll branch would look identical today and would silently stop
// sounding the score the moment anyone added the setting this branch exists for. One tick,
// not one per thirteen points.
func TestScoreRollJumpsWhenDisarmed(t *testing.T) {
	w := scoreboardWorld("")

	ticks := 0
	w.SoundPlayer = func(sound, priority int16) { ticks++ }

	w.Frame = 6
	w.Score, w.DisplayedScore, w.DoRollScore = 500, 0, false
	w.HandleDynamicScoreboard()

	if w.DisplayedScore != 500 {
		t.Errorf("displayed = %d, want the score outright at 500", w.DisplayedScore)
	}
	if ticks != 1 {
		t.Errorf("got %d ticks, want exactly 1", ticks)
	}

	// And nothing at all once they agree: no tick, no redraw, no drift.
	w.HandleDynamicScoreboard()
	if ticks != 1 {
		t.Errorf("a settled score ticked again: %d ticks", ticks)
	}
}

// TestRefreshScoreboardArmsAndEmptiesTheRoll is why walking into a room does not replay a
// count-up that was already in progress.
//
// RefreshScoreboard sets doRollScore -- arming the roll -- and then draws the real score and
// assigns displayedScore = theScore, which empties it. The two happen in that order in the
// same call, so the flag being set never costs a frame of counting.
func TestRefreshScoreboardArmsAndEmptiesTheRoll(t *testing.T) {
	w := scoreboardWorld("")

	ticks := 0
	w.SoundPlayer = func(sound, priority int16) { ticks++ }

	w.Score, w.DisplayedScore, w.DoRollScore = 500, 100, false
	w.RefreshScoreboard(NormalTitleMode)

	if !w.DoRollScore {
		t.Error("RefreshScoreboard left the roll disarmed")
	}
	if w.DisplayedScore != w.Score {
		t.Errorf("displayed = %d, want the score at %d", w.DisplayedScore, w.Score)
	}

	w.Frame = 3
	w.HandleDynamicScoreboard()
	if ticks != 0 {
		t.Errorf("the roll replayed after a refresh: %d ticks", ticks)
	}
}

// TestRefreshScoreboardPutsTheBandOnScreen is the whole deviation, end to end: after a
// refresh, the twenty rows below the house rect hold the composed band.
//
// It is one assertion and it is the one that matters, because every other test in this file
// would still pass if blitBoardToScreen were deleted. Rows 460..480 of a 640x480 screen are
// the strip no room draws into, so a byte-for-byte match against the band's own map means the
// player can see a scoreboard -- which, in the original's shipped configuration, they cannot.
func TestRefreshScoreboardPutsTheBandOnScreen(t *testing.T) {
	art := requireAssets(t, "art")
	w := scoreboardWorld(art)

	w.Main.Fill(w.Main.Bounds(), 200)
	w.RefreshScoreboard(NormalTitleMode)

	if !regionEquals(w.Main, w.BoardDestRect, w.Board.Board, w.R.V.BoardSrc) {
		t.Errorf("the band at %v is not the composed board map", w.BoardDestRect)
	}
}

// ---------------------------------------------------------------------------
// NumToString
// ---------------------------------------------------------------------------

// TestItoa covers the two things the scoreboard's number formatting is actually depended on
// for: the sign, which refreshNumGliders and QuickGlidersRefresh deliberately disagree about,
// and the most negative value, which a naive negate-in-place would print wrong.
func TestItoa(t *testing.T) {
	for _, c := range []struct {
		in   int32
		want string
	}{
		{0, "0"}, {1, "1"}, {9, "9"}, {10, "10"}, {13, "13"}, {999, "999"},
		{-1, "-1"}, {-2, "-2"}, {32767, "32767"}, {-32768, "-32768"},
		{2147483647, "2147483647"}, {-2147483648, "-2147483648"},
	} {
		if got := itoa32(c.in); got != c.want {
			t.Errorf("itoa32(%d) = %q, want %q", c.in, got, c.want)
		}
	}

	// The clamp and the lack of one, as the two panels see them. Mortals reaches -1 on the
	// last death of a one-player game and -2 on the last of a two-player game.
	if got := itoa16(-1); got != "-1" {
		t.Errorf("itoa16(-1) = %q; the quick path draws this and must not clamp", got)
	}
	if got := itoa16(0); got != "0" {
		t.Errorf("itoa16(0) = %q; the clamped path draws this", got)
	}
}

// ---------------------------------------------------------------------------
// The two panels, now that there are glyphs in them
// ---------------------------------------------------------------------------

// TestGliderCountClamp is the difference between Scoreboard.c's two glider-count refreshes,
// made visible.
//
// refreshNumGliders clamps Mortals at zero; QuickGlidersRefresh does not. Mortals reaches -1
// on the last death of a one-player game, so the clamp is what stops the board from reading
// "-1 gliders" in the moment before the game-over sequence takes the window -- and the quick
// path's lack of one is safe because its only caller is the branch where a glider remains.
//
// The expected panels are drawn by the same Panel that the code under test uses, with the
// string spelled out here. That makes this a test of which number each path chose, which is
// the thing that differs, and not a second copy of the font.
func TestGliderCountClamp(t *testing.T) {
	w := scoreboardWorld("")
	panel := func(text string) *render.Surface {
		s := render.NewSurface(w.Board.Gliders.W, w.Board.Gliders.H)
		w.Board.Panel(s, text)
		return s
	}
	zero, minusOne := panel("0"), panel("-1")
	if regionEquals(zero, zero.Bounds(), minusOne, minusOne.Bounds()) {
		t.Fatal(`"0" and "-1" draw the same pixels; this test cannot tell the paths apart`)
	}

	w.Mortals = -1
	w.refreshNumGliders()
	if !regionEquals(w.Board.Gliders, w.Board.Gliders.Bounds(), zero, zero.Bounds()) {
		t.Error(`refreshNumGliders with Mortals = -1 did not draw "0"; the clamp is gone`)
	}

	w.QuickGlidersRefresh()
	if !regionEquals(w.Board.Gliders, w.Board.Gliders.Bounds(), minusOne, minusOne.Bounds()) {
		t.Error(`QuickGlidersRefresh with Mortals = -1 did not draw "-1"; it has grown a clamp`)
	}

	// And the quick path puts it on the screen, at the destination the band's height moved.
	if !regionEquals(w.Main, w.BoardGQDestRect, minusOne, minusOne.Bounds()) {
		t.Errorf("the glider count at %v is not what the quick path drew", w.BoardGQDestRect)
	}
}

// TestScoreRollDrawsTheIntermediateNumber is the other half of TestScoreRoll: that one counts
// the steps the roll takes, this one checks that the number each step draws is the rolling
// one and not the real score.
//
// quickScoreRefresh is the only function in the file that draws DisplayedScore, and the roll
// is the only reason the two counters exist separately, so a version that drew Score would
// pass every other test here and never show the count-up.
func TestScoreRollDrawsTheIntermediateNumber(t *testing.T) {
	w := scoreboardWorld("")
	panel := func(text string) *render.Surface {
		s := render.NewSurface(w.Board.Points.W, w.Board.Points.H)
		w.Board.Panel(s, text)
		return s
	}

	w.Score, w.DisplayedScore, w.DoRollScore = 40, 0, true
	w.Frame = 3 // a frame the dynamic handler rolls on
	w.HandleDynamicScoreboard()

	if w.DisplayedScore != ScoreRollAmount {
		t.Fatalf("DisplayedScore = %d after one step, want %d", w.DisplayedScore, ScoreRollAmount)
	}
	thirteen, forty := panel("13"), panel("40")
	if !regionEquals(w.Board.Points, w.Board.Points.Bounds(), thirteen, thirteen.Bounds()) {
		t.Error(`the score panel does not read "13" after one roll step`)
	}
	if regionEquals(w.Board.Points, w.Board.Points.Bounds(), forty, forty.Bounds()) {
		t.Error(`the score panel reads "40"; quickScoreRefresh is drawing Score, not DisplayedScore`)
	}
	if !regionEquals(w.Main, w.BoardPQDestRect, thirteen, thirteen.Bounds()) {
		t.Errorf("the score at %v is not the intermediate number", w.BoardPQDestRect)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func wantRect(t *testing.T, name string, got, expect render.Rect) {
	t.Helper()
	if got != expect {
		t.Errorf("%s = {t:%d l:%d b:%d r:%d}, want {t:%d l:%d b:%d r:%d}",
			name, got.Top, got.Left, got.Bottom, got.Right,
			expect.Top, expect.Left, expect.Bottom, expect.Right)
	}
}

// regionIs reports whether every pixel of a rect holds one index -- how "nothing was drawn
// here" is asserted, given a sentinel fill.
func regionIs(s *render.Surface, r render.Rect, idx uint8) bool {
	for y := int(r.Top); y < int(r.Bottom); y++ {
		for x := int(r.Left); x < int(r.Right); x++ {
			if s.Pix[y*s.W+x] != idx {
				return false
			}
		}
	}
	return true
}

// regionEquals compares a rect of one surface against a rect of another, pixel for pixel. The
// two rects must be the same size; a mismatch is a failure and not a clip, because every blit
// on the scoreboard path copies a cell whose size is fixed at launch.
func regionEquals(dst *render.Surface, dr render.Rect, src *render.Surface, sr render.Rect) bool {
	if dr.Wide() != sr.Wide() || dr.Tall() != sr.Tall() {
		return false
	}
	for y := 0; y < int(dr.Tall()); y++ {
		for x := 0; x < int(dr.Wide()); x++ {
			d := dst.Pix[(int(dr.Top)+y)*dst.W+int(dr.Left)+x]
			s := src.Pix[(int(sr.Top)+y)*src.W+int(sr.Left)+x]
			if d != s {
				return false
			}
		}
	}
	return true
}

func litness(lit bool) string {
	if lit {
		return "lit"
	}
	return "blank"
}
