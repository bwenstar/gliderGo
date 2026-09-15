package render

import "testing"

// TestScoreboardGeometry640 pins all twenty-three of InitScoreboardMap's rects at the
// resolution Glider PRO was written for.
//
// The numbers below were derived by hand from StructuresInit.c:59-163 before the code was
// written, which is the only way this test is worth having: a golden file regenerated from
// the implementation would agree with any arithmetic at all. Every value is the C's
// expression evaluated at screenW=640, screenH=480, and the comment on each line is that
// expression.
func TestScoreboardGeometry640(t *testing.T) {
	v := DefaultView()

	// houseRect is 640x460, so boardSrcRect is 640 wide by kScoreboardTall.
	want(t, "BoardSrc", v.BoardSrc, SetRect(0, 0, 640, 20))

	// boardDestRect = boardSrcRect offset by -kScoreboardTall: rows -20..0, entirely
	// above the window. The original's, unmoved -- the port's deviation is applied by
	// game.placeScoreboard and not here.
	want(t, "BoardDest", v.BoardDest, SetRect(0, -20, 640, 0))

	// hOffset #2 = (RectWide(houseRect) - 640) / 2 = 0, so the panel literals land where
	// they are written.
	want(t, "BoardTSrc", v.BoardTSrc, SetRect(0, 0, 256, 12))
	want(t, "BoardTDest", v.BoardTDest, SetRect(137, 5, 393, 17))
	want(t, "BoardGSrc", v.BoardGSrc, SetRect(0, 0, 20, 10))
	want(t, "BoardGDest", v.BoardGDest, SetRect(526, 5, 546, 15))
	want(t, "BoardPSrc", v.BoardPSrc, SetRect(0, 0, 64, 10))
	want(t, "BoardPDest", v.BoardPDest, SetRect(570, 5, 634, 15))

	// The two quick destinations: the panels' board positions, less the band's height.
	want(t, "BoardPQDest", v.BoardPQDest, SetRect(570, -15, 634, -5))
	want(t, "BoardGQDest", v.BoardGQDest, SetRect(526, -15, 546, -5))

	// The badge sheet: two 16-wide columns, four rows of 16, 16, 17, 17 = 66.
	want(t, "BadgeSrc", v.BadgeSrc, SetRect(0, 0, 32, 66))
	want(t, "BadgesBlank[foil]", v.BadgesBlank[FoilBadge], SetRect(0, 0, 16, 16))
	want(t, "BadgesBlank[bands]", v.BadgesBlank[BandsBadge], SetRect(0, 16, 16, 32))
	want(t, "BadgesBlank[battery]", v.BadgesBlank[BatteryBadge], SetRect(0, 32, 16, 49))
	want(t, "BadgesBlank[helium]", v.BadgesBlank[HeliumBadge], SetRect(0, 49, 16, 66))
	want(t, "BadgesBadges[foil]", v.BadgesBadges[FoilBadge], SetRect(16, 0, 32, 16))
	want(t, "BadgesBadges[bands]", v.BadgesBadges[BandsBadge], SetRect(16, 16, 32, 32))
	want(t, "BadgesBadges[battery]", v.BadgesBadges[BatteryBadge], SetRect(16, 32, 32, 49))
	want(t, "BadgesBadges[helium]", v.BadgesBadges[HeliumBadge], SetRect(16, 49, 32, 66))

	// The four destinations, at 2-kScoreboardTall and 1-kScoreboardTall.
	want(t, "BadgesDest[foil]", v.BadgesDest[FoilBadge], SetRect(432, -18, 448, -2))
	want(t, "BadgesDest[bands]", v.BadgesDest[BandsBadge], SetRect(449, -18, 465, -2))
	want(t, "BadgesDest[battery]", v.BadgesDest[BatteryBadge], SetRect(467, -19, 483, -2))

	// Battery and helium are the same rect. One signed counter drives both, so only one
	// can ever be lit; the equality is the reason QuickBatteryRefresh can blank the
	// battery slot to hide the helium badge.
	want(t, "BadgesDest[helium]", v.BadgesDest[HeliumBadge], v.BadgesDest[BatteryBadge])

	// Every panel has to fit inside the band it is composed into, or it would be silently
	// clipped and the score would lose digits.
	for _, c := range []struct {
		name string
		r    Rect
	}{
		{"BoardTDest", v.BoardTDest}, {"BoardGDest", v.BoardGDest}, {"BoardPDest", v.BoardPDest},
	} {
		if s, ok := Sect(c.r, v.BoardSrc); !ok || s != c.r {
			t.Errorf("%s %v is not contained in the band %v", c.name, c.r, v.BoardSrc)
		}
	}
}

// TestScoreboardTwoOffsets is the reason initScoreboard writes its two hOffsets out
// separately instead of reusing one local the way the C does.
//
// The C's first hOffset centres a 1536-pixel picture in the board's width and its second
// centres a 640-pixel layout in the house rect. At 640x480 they are -448 and 0; at
// 1536x1026 they are 0 and 448. A port that folded them into one value would be correct at
// exactly one resolution, and 640 is the one where the mistake is invisible.
func TestScoreboardTwoOffsets(t *testing.T) {
	small, big := NewView(640, 480), NewView(1536, 1026)

	// hOffset #2 shows up as the panels' horizontal position. 137 at 640, 137+448 at 1536.
	if got, wantH := small.BoardTDest.Left, int16(137); got != wantH {
		t.Errorf("640: title left = %d, want %d", got, wantH)
	}
	if got, wantH := big.BoardTDest.Left, int16(137+448); got != wantH {
		t.Errorf("1536: title left = %d, want %d", got, wantH)
	}

	// hOffset #1 is not stored -- it is where ResetBoardArt puts the picture -- so it is
	// checked through the width it is derived from.
	for _, c := range []struct {
		v    *View
		want int16
	}{{small, -448}, {big, 0}} {
		if got := (c.v.BoardSrc.Wide() - kMaxViewWidth) / 2; got != c.want {
			t.Errorf("board art hOffset at %d wide = %d, want %d", c.v.BoardSrc.Wide(), got, c.want)
		}
	}

	// The band is always as wide as the drawable area and always 20 tall, at both.
	for _, v := range []*View{small, big} {
		if v.BoardSrc.Wide() != v.House.Wide() || v.BoardSrc.Tall() != kScoreboardTall {
			t.Errorf("band %v does not span the house rect %v", v.BoardSrc, v.House)
		}
	}
}

// TestScoreboardSurfaces checks that the five offscreen maps are the size of the rects that
// index them. A map one pixel short would clip a panel rather than fail.
func TestScoreboardSurfaces(t *testing.T) {
	art := requireAssets(t, "art")
	v := DefaultView()
	a := NewAssets(art)
	s := NewScoreboard(v, a)
	if err := a.Err(); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		name string
		surf *Surface
		r    Rect
	}{
		{"Board", s.Board, v.BoardSrc},
		{"Badge", s.Badge, v.BadgeSrc},
		{"Title", s.Title, v.BoardTSrc},
		{"Gliders", s.Gliders, v.BoardGSrc},
		{"Points", s.Points, v.BoardPSrc},
	} {
		if c.surf == nil {
			t.Errorf("%s is nil", c.name)
			continue
		}
		if int16(c.surf.W) != c.r.Wide() || int16(c.surf.H) != c.r.Tall() {
			t.Errorf("%s is %dx%d, want %dx%d", c.name, c.surf.W, c.surf.H, c.r.Wide(), c.r.Tall())
		}
	}

	// ResetBoardArt drew the picture at hOffset #1, which at 640 wide is -448: the band
	// shows the middle 640 columns of a 1536-wide image. Checking every pixel against the
	// strip is what pins that offset -- and it cannot be done by looking for unpainted
	// white, because the board's engraved highlights are index 0 themselves.
	strip := a.Strip("board")
	if strip == nil {
		t.Fatal("no board strip")
	}
	const hOffset = -448
	for y := 0; y < s.Board.H; y++ {
		for x := 0; x < s.Board.W; x++ {
			got, expect := s.Board.Pix[y*s.Board.W+x], strip.Pix[y*strip.W+x-hOffset]
			if got != expect {
				t.Fatalf("band pixel (%d,%d) = %d, want strip column %d = %d",
					x, y, got, x-hOffset, expect)
			}
		}
	}
}

// TestBadgeBlankCellsAreIdentical is what makes one transcribed oddity provably harmless.
//
// QuickBatteryRefresh blanks the *battery* cell even when it is the helium badge being
// hidden, because the two destination rects are identical. That leaves
// BadgesBlank[HeliumBadge] as the one rect InitScoreboardMap computes that no code path
// ever reads. If the two blank cells differ by a pixel the shortcut is a visible bug and
// this test is how anyone would find out.
func TestBadgeBlankCellsAreIdentical(t *testing.T) {
	art := requireAssets(t, "art")
	v := DefaultView()
	a := NewAssets(art)
	sheet := a.Strip("badge")
	if err := a.Err(); err != nil {
		t.Fatal(err)
	}

	batt, heli := v.BadgesBlank[BatteryBadge], v.BadgesBlank[HeliumBadge]
	if batt.Wide() != heli.Wide() || batt.Tall() != heli.Tall() {
		t.Fatalf("blank cells are different sizes: %v and %v", batt, heli)
	}
	for y := int16(0); y < batt.Tall(); y++ {
		for x := int16(0); x < batt.Wide(); x++ {
			b := sheet.Pix[int(batt.Top+y)*sheet.W+int(batt.Left+x)]
			h := sheet.Pix[int(heli.Top+y)*sheet.W+int(heli.Left+x)]
			if b != h {
				t.Fatalf("blank cells differ at (%d,%d): battery %d, helium %d", x, y, b, h)
			}
		}
	}
}

func want(t *testing.T, name string, got, expect Rect) {
	t.Helper()
	if got != expect {
		t.Errorf("%s = {t:%d l:%d b:%d r:%d}, want {t:%d l:%d b:%d r:%d}",
			name, got.Top, got.Left, got.Bottom, got.Right,
			expect.Top, expect.Left, expect.Bottom, expect.Right)
	}
}
