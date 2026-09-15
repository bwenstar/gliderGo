package scores

// The screen, checked against the three worked examples in docs/analysis/scoring.md 7.9.3-5.
//
// Those tables are the only evidence there is about this layout, because nothing in the
// original ever put it on a monitor: DoHighScores drew into the work map and returned without
// copying it forward (7.8). So the numbers here were derived from the source by hand, and
// checking the code against them -- rather than against a screenshot, which does not exist --
// is the whole of the available verification. Every coordinate in all three tables appears
// below.

import (
	"testing"

	"glidergo/internal/house"
	"glidergo/internal/render"
)

// ---------------------------------------------------------------------- anchors

func TestLayoutMatchesTheWorkedExamples(t *testing.T) {
	for _, tc := range []struct {
		what                         string
		w, h                         int
		left, drop, originH, originV int
		splash, plaque               render.Rect
		row0Shadow, footerV, footerH int
		rows19Shadow                 [9]int
	}{
		{
			// 7.9.3: the size this port draws into.
			what: "640x480", w: 640, h: 480,
			left: 144, drop: 129, originH: 0, originV: 0,
			splash:     render.SetRect(0, 0, 640, 480),
			plaque:     render.SetRect(154, 69, 486, 99),
			row0Shadow: 107, footerV: 308, footerH: 224,
			rows19Shadow: [9]int{147, 165, 183, 201, 219, 237, 255, 273, 291},
		},
		{
			// 7.9.4: the vertical anchor moves with splashOriginV, the horizontal one
			// recentres on the whole screen. 336 is not 144 + 192, which is the asymmetry.
			what: "1024x768", w: 1024, h: 768,
			left: 336, drop: 273, originH: 192, originV: 144,
			splash:     render.SetRect(192, 144, 832, 624),
			plaque:     render.SetRect(346, 213, 678, 243),
			row0Shadow: 251, footerV: 452, footerH: 416,
			rows19Shadow: [9]int{291, 309, 327, 345, 363, 381, 399, 417, 435},
		},
		{
			// 7.9.5: the smallest screen the original accepts. Both origins clamp to zero
			// (they would otherwise be -64 and -48), so dropIt is 129 again.
			what: "512x384", w: 512, h: 384,
			left: 80, drop: 129, originH: 0, originV: 0,
			splash:     render.SetRect(0, 0, 640, 480),
			plaque:     render.SetRect(90, 69, 422, 99),
			row0Shadow: 107, footerV: 308, footerH: 160,
			rows19Shadow: [9]int{147, 165, 183, 201, 219, 237, 255, 273, 291},
		},
	} {
		l := LayoutFor(tc.w, tc.h)
		if l.Left != tc.left || l.Drop != tc.drop {
			t.Errorf("%s: scoreLeft/dropIt = %d/%d, want %d/%d",
				tc.what, l.Left, l.Drop, tc.left, tc.drop)
		}
		if l.OriginH != tc.originH || l.OriginV != tc.originV {
			t.Errorf("%s: splashOrigin = %d,%d, want %d,%d",
				tc.what, l.OriginH, l.OriginV, tc.originH, tc.originV)
		}
		if got := l.Splash(); got != tc.splash {
			t.Errorf("%s: backdrop rect %v, want %v", tc.what, got, tc.splash)
		}
		if got := l.Plaque(); got != tc.plaque {
			t.Errorf("%s: plaque rect %v, want %v", tc.what, got, tc.plaque)
		}
		if got := l.RowBaseline(0); got != tc.row0Shadow {
			t.Errorf("%s: row 0 baseline %d, want %d", tc.what, got, tc.row0Shadow)
		}
		for i, want := range tc.rows19Shadow {
			if got := l.RowBaseline(i + 1); got != want {
				t.Errorf("%s: row %d baseline %d, want %d", tc.what, i+1, got, want)
			}
		}
		if got := l.FooterBaseline(); got != tc.footerV {
			t.Errorf("%s: footer baseline %d, want %d", tc.what, got, tc.footerV)
		}
		if got := l.Left + 80; got != tc.footerH {
			t.Errorf("%s: footer x %d, want %d", tc.what, got, tc.footerH)
		}
	}
}

// The column offsets, as the shadow/face pairs the worked examples list. Face is one pixel
// left of shadow in every column.
func TestColumnOffsetsMatchTheWorkedExamples(t *testing.T) {
	l := LayoutFor(640, 480)
	for _, tc := range []struct {
		dx           int
		shadow, face int
	}{
		{colPlacing, 145, 144},
		{colName, 175, 174},
		{colRooms, 305, 304},
		{colWord, 337, 336},
		{colScore, 435, 434},
	} {
		if got := l.Left + tc.dx; got != tc.shadow {
			t.Errorf("column at +%d: shadow x %d, want %d", tc.dx, got, tc.shadow)
		}
		if got := l.Left + tc.dx - 1; got != tc.face {
			t.Errorf("column at +%d: face x %d, want %d", tc.dx, got, tc.face)
		}
	}

	// And the 1024x768 face positions, which is the check that the offsets are relative to
	// scoreLeft rather than to the screen.
	big := LayoutFor(1024, 768)
	for i, want := range []int{336, 366, 496, 528, 626} {
		dx := []int{colPlacing, colName, colRooms, colWord, colScore}[i]
		if got := big.Left + dx - 1; got != want {
			t.Errorf("1024x768 column %d face x %d, want %d", i+1, got, want)
		}
	}
}

// Row one really is above the banner box, which is the layout's most surprising property and
// the easiest to lose in a rewrite that "tidied" RowBaseline into a multiplication.
func TestRowOneIsDrawnAboveTheBannerBox(t *testing.T) {
	l := LayoutFor(640, 480)
	// The yellow box's top, from 7.9.3.
	const bannerBoxTop = 111
	if got := l.RowBaseline(0); got >= bannerBoxTop {
		t.Errorf("row 0's baseline is %d; the banner box starts at %d", got, bannerBoxTop)
	}
	if got := l.RowBaseline(1); got <= bannerBoxTop+18 {
		t.Errorf("row 1's baseline is %d; it should be below the box", got)
	}
	// And the lift is kKimsLifted more than a plain -18.
	if got, want := l.RowBaseline(0), l.Drop-scoreSpacing-kimsLifted; got != want {
		t.Errorf("row 0 baseline %d, want %d", got, want)
	}
}

// ----------------------------------------------------------------------- pixels

// tally counts the palette indices in a rect: enough to say "this band holds white and no
// cyan" without pinning the shape of a glyph.
func tally(s *render.Surface, r render.Rect) map[uint8]int {
	out := map[uint8]int{}
	for y := int(r.Top); y < int(r.Bottom) && y < s.H; y++ {
		for x := int(r.Left); x < int(r.Right) && x < s.W; x++ {
			if x < 0 || y < 0 {
				continue
			}
			out[s.Pix[y*s.W+x]]++
		}
	}
	return out
}

// faceBand is the rows a coloured face occupies for a row baseline: the font puts a capital
// in v-7..v-1, and the face sits one pixel above the shadow.
func faceBand(l Layout, i int, x0, x1 int) render.Rect {
	v := l.RowBaseline(i) - 1
	return render.SetRect(int16(x0), int16(v-7), int16(x1), int16(v+2))
}

func newScreen() *render.Surface {
	s := render.NewSurface(640, 480)
	s.Fill(s.Bounds(), render.Black8)
	return s
}

// The reproduced oversight, pinned so that a later "fix" has to be a deliberate one: on the
// highlighted row, four columns turn white and the word "rooms" stays cyan.
func TestTheHighlightedRowLeavesColumnFourCyan(t *testing.T) {
	dst := newScreen()
	l := LayoutFor(640, 480)
	s := board(900, 800)

	Draw(dst, nil, "House", &s, 0)

	white := []struct {
		what   string
		x0, x1 int
	}{
		{"placing", l.Left + colPlacing - 1, l.Left + colName - 1},
		{"name", l.Left + colName - 1, l.Left + colRooms - 1},
		{"rooms count", l.Left + colRooms - 1, l.Left + colWord - 1},
		{"score", l.Left + colScore - 1, l.Left + scoreWide},
	}
	for _, c := range white {
		got := tally(dst, faceBand(l, 0, c.x0, c.x1))
		if got[render.White8] == 0 {
			t.Errorf("the highlighted row's %s column has no white pixels: %v", c.what, got)
		}
		if got[render.QDCyan] != 0 || got[render.QDYellow] != 0 {
			t.Errorf("the highlighted row's %s column is still coloured: %v", c.what, got)
		}
	}

	word := tally(dst, faceBand(l, 0, l.Left+colWord-1, l.Left+colScore-1))
	if word[render.QDCyan] == 0 {
		t.Errorf("column four is not cyan on the highlighted row: %v", word)
	}
	if word[render.White8] != 0 {
		t.Error("column four turned white; HighScores.c:240 has no lastHighScore test, " +
			"so the highlighted row reads white white white cyan white")
	}
}

// An unhighlighted row: cyan placing, yellow name, yellow rooms, cyan word, yellow score.
func TestAnOrdinaryRowUsesTheOriginalsColours(t *testing.T) {
	dst := newScreen()
	l := LayoutFor(640, 480)
	s := board(900, 800)

	Draw(dst, nil, "House", &s, -1)

	for _, c := range []struct {
		what   string
		x0, x1 int
		want   uint8
	}{
		{"placing", l.Left + colPlacing - 1, l.Left + colName - 1, render.QDCyan},
		{"name", l.Left + colName - 1, l.Left + colRooms - 1, render.QDYellow},
		{"rooms count", l.Left + colRooms - 1, l.Left + colWord - 1, render.QDYellow},
		{"the word rooms", l.Left + colWord - 1, l.Left + colScore - 1, render.QDCyan},
		{"score", l.Left + colScore - 1, l.Left + scoreWide, render.QDYellow},
	} {
		got := tally(dst, faceBand(l, 1, c.x0, c.x1))
		if got[c.want] == 0 {
			t.Errorf("the %s column has no pixels of index %d: %v", c.what, c.want, got)
		}
		if got[render.White8] != 0 {
			t.Errorf("the %s column has white pixels with no row highlighted: %v", c.what, got)
		}
	}
}

// Every face has a black shadow behind it, one pixel down and right of nothing -- the faces
// are up and left of their shadows here, which is the opposite of the in-game scoreboard.
// This checks the shadow exists and is where the analysis says.
func TestEachRowFaceHasABlackShadowBelowIt(t *testing.T) {
	dst := render.NewSurface(640, 480)
	dst.Fill(dst.Bounds(), render.LtGray8) // not black, so a black pixel is evidence
	l := LayoutFor(640, 480)
	s := board(900)

	Draw(dst, nil, "House", &s, -1)

	// The shadow band is one pixel lower than the face band.
	v := l.RowBaseline(0)
	band := render.SetRect(int16(l.Left), int16(v-7), int16(l.Left+scoreWide), int16(v+2))
	got := tally(dst, band)
	if got[render.Black8] == 0 {
		t.Errorf("no shadow pixels in row 0's band: %v", got)
	}
	// The bottom row of the shadow's capital is at v-1, and the face's is at v-2, so the
	// pixels at v-1 in the placing column can only be shadow.
	only := tally(dst, render.SetRect(int16(l.Left+colPlacing), int16(v-1),
		int16(l.Left+colPlacing+5), int16(v)))
	if only[render.Black8] == 0 {
		t.Errorf("the shadow's own bottom row is empty: %v", only)
	}
}

// Only rows with a score are drawn (HighScores.c:176), which is why the fourteen-hyphen
// placeholder never reaches the screen.
func TestEmptyRowsAreNotDrawn(t *testing.T) {
	dst := newScreen()
	l := LayoutFor(640, 480)
	s := board(900, 800, 700)

	Draw(dst, nil, "House", &s, -1)

	for i := 3; i < Max; i++ {
		band := faceBand(l, i, l.Left, l.Left+scoreWide)
		got := tally(dst, band)
		for idx, n := range got {
			if idx != render.Black8 && n > 0 {
				t.Errorf("row %d was drawn although its score is 0: %v", i, got)
				break
			}
		}
	}
	// And the three that do have scores are there.
	for i := 0; i < 3; i++ {
		got := tally(dst, faceBand(l, i, l.Left, l.Left+scoreWide))
		if got[render.QDYellow] == 0 && got[render.QDCyan] == 0 {
			t.Errorf("row %d has a score but was not drawn: %v", i, got)
		}
	}
}

// The banner's double box: black at dropIt-17, yellow one pixel up and left of it. Checked at
// the corners, which is where a frame that was drawn at the wrong offset shows.
func TestTheBannerBoxIsFramedTwice(t *testing.T) {
	dst := newScreen()
	l := LayoutFor(640, 480)
	var s house.Scores
	s.Banner.SetText("Hi")
	for i := range s.Names {
		s.Names[i].SetText(EmptyName)
	}

	Draw(dst, nil, "House", &s, -1)

	bw := int(render.StringWidth("Hi"))
	blackLeft := l.Left - 3 + (scoreWide-bw)/2
	blackTop := l.Drop - 17

	at := func(x, y int) uint8 { return dst.Pix[y*dst.W+x] }
	// The yellow frame is up and left, so its top-left corner is the outermost pixel.
	if got := at(blackLeft-1, blackTop-1); got != render.QDYellow {
		t.Errorf("the yellow box's top-left corner is index %d, want %d",
			got, render.QDYellow)
	}
	// The black frame's bottom-right corner is outside the yellow one.
	if got := at(blackLeft+bw+7, blackTop+17); got != render.Black8 {
		t.Errorf("the black box's bottom-right corner is index %d, want %d",
			got, render.Black8)
	}
	// 7.9.3: black box (T=112, B=130), yellow box (T=111, B=129).
	if blackTop != 112 {
		t.Errorf("the black box's top is %d, want 112", blackTop)
	}
}

// The footer is blue and 9 pt in the original, so it is the one line with no shadow. What
// matters here is the colour and that it says something a keyboard-only port can honour.
func TestTheFooterIsBlueAndDoesNotMentionTheMouse(t *testing.T) {
	dst := newScreen()
	l := LayoutFor(640, 480)
	var s house.Scores

	Draw(dst, nil, "House", &s, -1)

	v := l.FooterBaseline()
	got := tally(dst, render.SetRect(int16(l.Left+80), int16(v-7),
		int16(l.Left+80+int(render.StringWidth(ExitWord))), int16(v+2)))
	if got[render.QDBlue] == 0 {
		t.Errorf("the footer has no blue pixels: %v", got)
	}

	// The original's STR# 150 index 8 is "Click Mouse or Hit a Key to Exit". This port has
	// no pointer input at all, so promising the mouse would be an instruction that does
	// not work -- a deviation worth pinning rather than leaving to a future tidy-up.
	if ExitWord != "Hit a Key to Exit" {
		t.Errorf("the footer says %q", ExitWord)
	}
}

// The title is cyan with a black shadow up and to the *left*, which is unique to this screen.
func TestTheTitleShadowIsUpAndToTheLeft(t *testing.T) {
	dst := newScreen()
	l := LayoutFor(640, 480)
	var s house.Scores
	Draw(dst, nil, "Slumberland", &s, -1)

	// The face's band at dropIt-65 and the shadow's at dropIt-66. A scale-2 capital
	// occupies fourteen rows above the baseline, so take a generous band and require both
	// colours in it -- the offset itself is checked by the leftmost column below.
	band := render.SetRect(int16(l.Left), int16(l.Drop-66-14),
		int16(l.Left+scoreWide), int16(l.Drop-63))
	got := tally(dst, band)
	if got[render.QDCyan] == 0 {
		t.Errorf("the title has no cyan pixels: %v", got)
	}
	if got[render.Black8] == 0 {
		t.Errorf("the title has no shadow: %v", got)
	}

	// The shadow starts one pixel to the left of the face, so the leftmost non-background
	// column in the band is the shadow's.
	title := "• Slumberland •"
	scale := 2
	if int(render.StringWidthScaled(title, scale)) > scoreWide {
		scale = 1
	}
	w := int(render.StringWidthScaled(title, scale))
	faceX := l.Left + (scoreWide-w)/2
	if faceX <= 0 {
		t.Fatalf("the title starts at %d", faceX)
	}
	shadowOnly := tally(dst, render.SetRect(int16(faceX-1), int16(l.Drop-66-14),
		int16(faceX), int16(l.Drop-63)))
	if shadowOnly[render.Black8] == 0 {
		t.Error("nothing in the column left of the title's face; the shadow is not up-left")
	}
	if shadowOnly[render.QDCyan] != 0 {
		t.Error("the cyan face reaches a pixel left of where it was asked to start")
	}
}

// A house with a very long name gets a smaller title rather than one that runs off both ends
// of the block. The original never had to decide this: a 14 pt proportional font drew
// narrower than a scaled bitmap can.
func TestALongHouseNameShrinksTheTitleRatherThanOverflowing(t *testing.T) {
	l := LayoutFor(640, 480)
	long := "A House With A Preposterously Long Name Indeed"
	title := "• " + long + " •"

	if int(render.StringWidthScaled(title, 2)) <= scoreWide {
		t.Fatal("the sample name is not long enough to force the smaller size")
	}
	if got := int(render.StringWidthScaled(title, 1)); got > scoreWide {
		t.Skipf("even at scale 1 the title is %d wide; that case is clipped, not shrunk", got)
	}

	dst := newScreen()
	var s house.Scores
	Draw(dst, nil, long, &s, -1)

	// Nothing in the eight columns outside the block on either side.
	for _, side := range []render.Rect{
		render.SetRect(int16(l.Left-8), int16(l.Drop-80), int16(l.Left), int16(l.Drop-60)),
		render.SetRect(int16(l.Left+scoreWide), int16(l.Drop-80),
			int16(l.Left+scoreWide+8), int16(l.Drop-60)),
	} {
		got := tally(dst, side)
		area := int(side.Right-side.Left) * int(side.Bottom-side.Top)
		if got[render.QDCyan] != 0 || got[render.Black8] != area {
			t.Errorf("the title spilled outside the block: %v", got)
		}
	}
}

// ----------------------------------------------------------------- robustness

// The screen has to draw on a surface that is not 640x480 and against a tree with no art,
// because both happen: `-shot` sizes its own surface and a fresh clone has no assets. What
// must not happen is a panic or a write outside the buffer.
func TestDrawSurvivesOddSurfacesAndMissingArt(t *testing.T) {
	s := board(900, 800, 700, 600, 500, 400, 300, 200, 100, 50)
	s.Banner.SetText("A rather long banner, as banners go")
	for _, size := range [][2]int{
		{640, 480}, {512, 384}, {1024, 768}, {320, 240}, {64, 64}, {1, 1}, {800, 480},
	} {
		dst := render.NewSurface(size[0], size[1])
		Draw(dst, nil, "Some House", &s, 3)
		if len(dst.Pix) != size[0]*size[1] {
			t.Errorf("%dx%d: the surface changed size", size[0], size[1])
		}
	}
}

// A missing plaque leaves a heading rather than a blank strip: docs/IMPROVEMENTS.md 2.6, the
// absence of a decoration is not the absence of a screen.
func TestAMissingPlaqueDrawsItsOwnHeading(t *testing.T) {
	dst := newScreen()
	l := LayoutFor(640, 480)
	var s house.Scores
	Draw(dst, nil, "House", &s, -1)

	got := tally(dst, l.Plaque())
	if got[render.QDYellow] == 0 {
		t.Errorf("nothing was drawn where the plaque goes: %v", got)
	}
}

// Draw is a pure function of its arguments: it must not modify the board it is given, or
// showing the screen would be a reason to write the side-car back.
func TestDrawDoesNotModifyTheBoard(t *testing.T) {
	s := board(900, 800, 700)
	before := house.EncodeScores(&s)
	Draw(newScreen(), nil, "House", &s, 1)
	after := house.EncodeScores(&s)
	if string(after) != string(before) {
		t.Error("Draw modified the board")
	}
}
