package render

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"glidergo/internal/house"
)

// The font's data is a compile-time constant, so most of what could go wrong with it is a
// typo in a string literal. These tests are therefore shaped around the two questions a typo
// would answer wrongly: is the sheet still the shape the decoder assumes, and do the
// characters that matter still come out as the letters they are supposed to be.

// TestFontSheetShape re-checks the sheet's geometry from the outside.
//
// init already panics on the two coarse failures (wrong line count, wrong line length), and
// this covers the ones it cannot see: that the notation is only '.' and '#' -- a space would
// read as a blank and look right in the source while being a different character to the
// decoder -- and that the sixth column of every cell is empty, because the advance is the
// whole cell width and a glyph that painted its spacing column would touch its neighbour.
func TestFontSheetShape(t *testing.T) {
	lines := strings.Split(strings.Trim(fontSheet, "\n"), "\n")
	if len(lines) != 6*FontTall {
		t.Fatalf("sheet has %d lines, want %d", len(lines), 6*FontTall)
	}
	for i, line := range lines {
		if len(line) != 16*FontWide {
			t.Fatalf("line %d is %d characters, want %d", i, len(line), 16*FontWide)
		}
		for x := 0; x < len(line); x++ {
			if line[x] != '.' && line[x] != '#' {
				t.Fatalf("line %d column %d is %q, want '.' or '#'", i, x, line[x])
			}
			if x%FontWide == FontWide-1 && line[x] == '#' {
				t.Errorf("line %d column %d paints the spacing column of cell %d",
					i, x, x/FontWide)
			}
		}
	}

	// Every ASCII code from space to DEL has a cell, in order, at the front of the glyph
	// table. The extras follow, and nothing overwrote one of the first 96.
	for r := rune(0x20); r <= 0x7F; r++ {
		g, ok := glyphOf[r]
		if !ok {
			t.Errorf("no glyph for %q", r)
			continue
		}
		if want := int(r - 0x20); g != want {
			t.Errorf("%q is glyph %d, want %d", r, g, want)
		}
	}
	if fallback != int(0x7F-0x20) {
		t.Errorf("fallback is glyph %d, want the 0x7F cell", fallback)
	}
}

// TestGlyphPixels pins nine characters, pixel for pixel, in the same notation the font is
// authored in.
//
// The art below was read off the implementation and then checked by eye, which is the only
// verification a hand-authored font can have: there is no reference bitmap to diff against,
// because the original's font was Geneva and Geneva is not in the sources. What the test is
// for is the next change -- a shifted row, a fold rewritten, a mark moved -- showing up as a
// letter that no longer looks like itself.
//
// The nine are chosen one per mechanism: a digit and a capital from the sheet, a descender, a
// composed accent, an authored extra, a two-mark composition, a fold that expands to three
// cells, the glyph whose centre dot distinguishes 0 from O, and the box.
func TestGlyphPixels(t *testing.T) {
	for _, c := range []struct {
		text string
		art  []string
	}{
		{"1", []string{
			"..#...",
			".##...",
			"..#...",
			"..#...",
			"..#...",
			"..#...",
			".###..",
			"......",
			"......",
		}},
		{"A", []string{
			"..#...",
			".#.#..",
			"#...#.",
			"#...#.",
			"#####.",
			"#...#.",
			"#...#.",
			"......",
			"......",
		}},
		// A descender: rows 7 and 8, the two below the baseline.
		{"g", []string{
			"......",
			"......",
			".####.",
			"#...#.",
			"#...#.",
			"#...#.",
			".####.",
			"....#.",
			".###..",
		}},
		// 0 against O: the interior dots are what tells them apart at this size.
		{"0", []string{
			".###..",
			"#...#.",
			"#..##.",
			"#.#.#.",
			"##..#.",
			"#...#.",
			".###..",
			"......",
			"......",
		}},
		// Composed: the acute sits in rows 0 and 1, which the lowercase body leaves free.
		{"á", []string{
			"...#..",
			"..#...",
			".###..",
			"....#.",
			".####.",
			"#...#.",
			".####.",
			"......",
			"......",
		}},
		// The tilde is the one mark that has to read as a shape rather than as dots.
		{"ñ", []string{
			"#.##..",
			".#..#.",
			"####..",
			"#...#.",
			"#...#.",
			"#...#.",
			"#...#.",
			"......",
			"......",
		}},
		// An authored extra, and the one non-ASCII character the shipped houses are full
		// of: 58 of their room names end in it.
		{"…", []string{
			"......",
			"......",
			"......",
			"......",
			"......",
			"......",
			"#.#.#.",
			"......",
			"......",
		}},
		// A fold: one rune, three cells.
		{"©", []string{
			"...#.........#....",
			"..#...........#...",
			".#.....###.....#..",
			".#....#........#..",
			".#....#........#..",
			"..#...#.......#...",
			"...#...###...#....",
			"..................",
			"..................",
		}},
		// The box, for a rune with no representation at all.
		{"\x01", []string{
			"#####.",
			"#...#.",
			"#...#.",
			"#...#.",
			"#...#.",
			"#...#.",
			"#####.",
			"......",
			"......",
		}},
	} {
		got := drawnArt(c.text)
		for r := range c.art {
			if got[r] != c.art[r] {
				t.Errorf("%q row %d:\n got %s\nwant %s", c.text, r, got[r], c.art[r])
			}
		}
	}
}

// TestBaselinePlacement is the reason the original's pen positions work in panels only ten
// pixels tall.
//
// Scoreboard.c draws every string at v=10 and again at v=9, into panels that are 12 rows for
// the room title and 10 for the glider count and the score. With a seven-row ascent the
// bottom of a capital lands at v-1 = row 9, exactly the last row of a 10-row panel: one row
// lower and every digit on the scoreboard would lose its base.
func TestBaselinePlacement(t *testing.T) {
	// A capital occupies the seven rows at and above the baseline, and no more.
	tall := NewSurface(FontWide, 20)
	tall.DrawString(0, 10, "A", Black8)
	if top, bot := rowExtent(tall); top != 3 || bot != 9 {
		t.Errorf("'A' at v=10 fills rows %d..%d, want 3..9", top, bot)
	}

	// A descender reaches two rows past that.
	tall.Fill(tall.Bounds(), White8)
	tall.DrawString(0, 10, "g", Black8)
	if top, bot := rowExtent(tall); top != 5 || bot != 11 {
		t.Errorf("'g' at v=10 fills rows %d..%d, want 5..11", top, bot)
	}

	// And a digit drawn into a real ten-row panel at the real pen loses nothing: the same
	// glyph in a surface with room to spare has the same pixels.
	panel := NewSurface(int(DefaultView().BoardGSrc.Wide()), int(DefaultView().BoardGSrc.Tall()))
	panel.DrawString(1, 10, "0", Black8)
	roomy := NewSurface(panel.W, 20)
	roomy.DrawString(1, 10, "0", Black8)
	for y := 0; y < panel.H; y++ {
		for x := 0; x < panel.W; x++ {
			if panel.Pix[y*panel.W+x] != roomy.Pix[y*roomy.W+x] {
				t.Fatalf("'0' at (1,10) differs at (%d,%d) between a 10-row panel and a 20-row one", x, y)
			}
		}
	}
	if _, bot := rowExtent(panel); bot != panel.H-1 {
		t.Errorf("'0' in a %d-row panel ends at row %d, want the last row", panel.H, bot)
	}
}

// TestPanelDropShadow checks the one thing about Panel that is an ordering and not a
// position: black at (1,10) first, white at (0,9) second, so the white wins where the two
// overlap and the shadow falls down and to the right.
//
// It calls Panel on a bare Scoreboard because Panel touches nothing but the surface it is
// given -- no view, no assets, no board -- and constructing the real thing would drag the
// extracted art into a test about two pen positions.
func TestPanelDropShadow(t *testing.T) {
	p := NewSurface(20, 10)
	(&Scoreboard{}).Panel(p, "1")

	at := func(x, y int) uint8 { return p.Pix[y*p.W+x] }

	// (2,4) takes the digit's (col 1, row 1) from the black pass and its (col 2, row 2)
	// from the white pass. Drawn in the C's order it is white.
	if got := at(2, 4); got != White8 {
		t.Errorf("the overlapping pixel (2,4) is %d, want white (%d) -- the two draws are in the wrong order",
			got, White8)
	}
	// (3,5) is the black pass only: the white pass's glyph column there is blank.
	if got := at(3, 5); got != Black8 {
		t.Errorf("the shadow pixel (3,5) is %d, want black (%d)", got, Black8)
	}
	// Everything the two draws did not touch is still the gray the panel was cleared to.
	lit := 0
	for i := range p.Pix {
		if p.Pix[i] != kGrayBackgroundColor {
			lit++
		}
	}
	if lit == 0 {
		t.Fatal("Panel drew nothing")
	}
	if lit >= p.W*p.H {
		t.Fatalf("Panel painted all %d pixels; it should have left the gray around the digit", lit)
	}
}

// TestStringWidth checks the advance against what DrawString actually paints. The two are
// separate functions and a fold makes them easy to disagree: one rune can be three cells.
func TestStringWidth(t *testing.T) {
	for _, c := range []struct {
		text string
		want int16
	}{
		{"", 0},
		{" ", FontWide},
		{"A", FontWide},
		{"AB", 2 * FontWide},
		{"á", FontWide},         // composed: base plus mark, one cell
		{"©", 3 * FontWide},     // folded to "(c)"
		{"≤", 2 * FontWide},     // folded to "<="
		{"\x01", FontWide},      // the box still advances
		{"Café…", 5 * FontWide}, // one composed, one extra, three plain
		{"\u00a0", FontWide},    // no-break space folds to a space
	} {
		if got := StringWidth(c.text); got != c.want {
			t.Errorf("StringWidth(%q) = %d, want %d", c.text, got, c.want)
		}
	}

	// And nothing is painted outside the advance, nor short of the last cell.
	for _, text := range []string{"A", "Glider PRO", "©", "El Niño…", "≥≤±"} {
		w := StringWidth(text)
		s := NewSurface(int(w)+FontWide, FontTall)
		s.DrawString(0, fontAscent, text, Black8)
		right := -1
		for x := 0; x < s.W; x++ {
			for y := 0; y < s.H; y++ {
				if s.Pix[y*s.W+x] == Black8 {
					right = x
				}
			}
		}
		if right < 0 {
			t.Errorf("%q painted nothing", text)
			continue
		}
		if int16(right) >= w {
			t.Errorf("%q painted column %d, past its advance of %d", text, right, w)
		}
		if int16(right) < w-FontWide {
			t.Errorf("%q painted no further than column %d, short of its last cell at %d",
				text, right, w-FontWide)
		}
	}
}

// TestDrawStringClips draws off all four edges. plot is the clipping primitive, so this is
// really a test that DrawString goes through it rather than indexing Pix itself: a room name
// wider than its panel is a user-supplied file's doing, and it must not panic.
func TestDrawStringClips(t *testing.T) {
	s := NewSurface(10, 10)
	s.DrawString(-100, 5, "Glider", Black8)
	s.DrawString(100, 5, "Glider", Black8)
	s.DrawString(2, -50, "Glider", Black8)
	s.DrawString(2, 200, "Glider", Black8)
	for i, v := range s.Pix {
		if v != White8 {
			t.Fatalf("a fully clipped string painted pixel %d", i)
		}
	}

	// A partial overlap paints its visible sliver and nothing else. 'l' is a single stem in
	// column 2, so at h=-2 it lands in column 0.
	s.DrawString(-2, fontAscent+6, "l", Black8)
	if s.Pix[6*s.W+0] != Black8 {
		t.Error("the visible column of a half-clipped glyph was not painted")
	}
}

// TestFoldsAreOneLevel is the invariant that makes eachCell's fold arm terminate without a
// depth counter: a fold's expansion is resolved through the glyphs and the composed forms
// only, so an entry that expanded to another folded rune would silently draw a box.
func TestFoldsAreOneLevel(t *testing.T) {
	for r, s := range folds {
		if s == "" {
			t.Errorf("%q folds to nothing", r)
			continue
		}
		for _, f := range s {
			if _, ok := folds[f]; ok {
				t.Errorf("%q folds to %q, which contains the folded rune %q", r, s, f)
			}
			_, isGlyph := glyphOf[f]
			_, isComposed := composed[f]
			if !isGlyph && !isComposed {
				t.Errorf("%q folds to %q, which contains the unrenderable rune %q", r, s, f)
			}
		}
		if _, ok := glyphOf[r]; ok {
			t.Errorf("%q has both a glyph and a fold", r)
		}
		if _, ok := composed[r]; ok {
			t.Errorf("%q has both a composed form and a fold", r)
		}
	}
}

// TestEveryMacRomanRuneIsRenderable is the coverage promise this font makes, and the reason
// the fold table is as long as it is.
//
// A house is a user-supplied file and its room names are Mac Roman, so all 256 byte values
// can reach the title panel. Every one of them has to come out as something legible -- a box
// in a room name is the kind of thing that gets a released game a bug report -- and the only
// bytes exempt are the control codes below 0x20, which are not characters and which the
// original's own text format escapes rather than draws.
//
// The corpus is built with the same decoder the house loader uses, so this cannot drift from
// what a house can actually contain.
func TestEveryMacRomanRuneIsRenderable(t *testing.T) {
	b := make([]byte, 0, 224)
	for c := 0x20; c <= 0xFF; c++ {
		b = append(b, byte(c))
	}
	text := house.MacRomanToUTF8(b)
	if bad := UnrenderableRunes(text); len(bad) != 0 {
		t.Errorf("%d Mac Roman characters have no representation: %q", len(bad), string(bad))
	}
	for _, r := range text {
		if w := StringWidth(string(r)); w < FontWide || w%FontWide != 0 {
			t.Errorf("%q is %d pixels wide, want a positive multiple of %d", r, w, FontWide)
		}
	}
}

// TestUnrenderableRunes checks the report itself: distinct runes, in the order they first
// appear, and only the ones that really fall through to the box.
func TestUnrenderableRunes(t *testing.T) {
	if got := UnrenderableRunes("Café… (c) 1994"); len(got) != 0 {
		t.Errorf("UnrenderableRunes of renderable text returned %q", string(got))
	}
	got := UnrenderableRunes("\x01ok\u3042ok\x01")
	if want := []rune{'\x01', '\u3042'}; string(got) != string(want) {
		t.Errorf("UnrenderableRunes = %q, want %q", string(got), string(want))
	}
	// And the box is what actually gets drawn for one.
	if a, b := drawnArt("\u3042"), drawnArt("\x7f"); strings.Join(a, "\n") != strings.Join(b, "\n") {
		t.Error("an unrenderable rune did not draw the 0x7F box")
	}
}

// TestShippedTextFitsItsPanel walks every room name in the 22 shipped houses, plus the two
// strings Scoreboard.c hard-codes, and checks that each one is renderable and fits the title
// panel at the pen the original draws it with.
//
// Fitting is not automatic: this font is fixed-width where Geneva was proportional, so a name
// that fit in 1994 could overflow here. It does not, and by a wide margin -- the widest name
// in the corpus is nowhere near the 256-pixel panel -- but that is a fact about the shipped
// houses and the check is what will catch the first authored house that pushes past it. See
// docs/IMPROVEMENTS.md 2.29 and 4.1.
func TestShippedTextFitsItsPanel(t *testing.T) {
	houseDir := requireAssets(t, "houses")
	paths, err := filepath.Glob(filepath.Join(houseDir, "*.house"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Skipf("no houses in %s", houseDir)
	}
	sort.Strings(paths)

	v := DefaultView()
	// The pen's h is 1 for the black pass, so a string fits when 1+width is inside the
	// panel.
	budget := v.BoardTSrc.Wide() - 1

	var widest int16
	var widestName string
	names := 0
	for _, path := range paths {
		h, err := house.LoadFile(path)
		if err != nil {
			t.Errorf("%s: %v", filepath.Base(path), err)
			continue
		}
		for i := range h.Rooms {
			name := h.Rooms[i].Name.Text()
			names++
			if bad := UnrenderableRunes(name); len(bad) != 0 {
				t.Errorf("%s room %d %q: unrenderable %q",
					filepath.Base(path), i, name, string(bad))
			}
			if w := StringWidth(name); w > widest {
				widest, widestName = w, name
			}
		}
	}
	if names == 0 {
		t.Fatal("no room names in the corpus")
	}
	if widest > budget {
		t.Errorf("the widest room name %q is %d pixels, past the title panel's %d",
			widestName, widest, budget)
	}
	t.Logf("%d room names, widest %q at %d of %d pixels", names, widestName, widest, budget)

	for _, s := range []string{"Hit Delete key if unable to Follow", "Saving Game…"} {
		if bad := UnrenderableRunes(s); len(bad) != 0 {
			t.Errorf("%q: unrenderable %q", s, string(bad))
		}
		if w := StringWidth(s); w > budget {
			t.Errorf("%q is %d pixels, past the title panel's %d", s, w, budget)
		}
	}

	// The two number panels have their own budgets, and the numbers that go in them are
	// bounded: the score is at most seven digits before the game's own 9,999,999 cap and
	// the glider count is two characters at its widest ("-2", which refreshNumGliders
	// clamps away but QuickGlidersRefresh does not).
	if w := StringWidth("1234567"); w > v.BoardPSrc.Wide() {
		t.Errorf("a seven-digit score is %d pixels, past the score panel's %d", w, v.BoardPSrc.Wide())
	}
	if w := StringWidth("-2"); w > v.BoardGSrc.Wide() {
		t.Errorf("\"-2\" is %d pixels, past the glider panel's %d", w, v.BoardGSrc.Wide())
	}
}

// drawnArt renders text at the top-left of a surface exactly its advance wide and returns it
// in the font's own notation, so a failure prints as the letter that came out.
func drawnArt(text string) []string {
	s := NewSurface(int(StringWidth(text)), FontTall)
	s.DrawString(0, fontAscent, text, Black8)
	rows := make([]string, FontTall)
	for y := 0; y < FontTall; y++ {
		var b strings.Builder
		for x := 0; x < s.W; x++ {
			if s.Pix[y*s.W+x] == Black8 {
				b.WriteByte('#')
			} else {
				b.WriteByte('.')
			}
		}
		rows[y] = b.String()
	}
	return rows
}

// rowExtent returns the first and last rows with any non-white pixel, or -1, -1.
func rowExtent(s *Surface) (top, bottom int) {
	top, bottom = -1, -1
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			if s.Pix[y*s.W+x] != White8 {
				if top < 0 {
					top = y
				}
				bottom = y
				break
			}
		}
	}
	return top, bottom
}
