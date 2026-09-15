package render

// Text: a hand-authored bitmap font, and the DrawString/StringWidth pair that
// Scoreboard.Panel and DrawCalendar are written against.
//
// **This font is not the original's font, and it cannot be.** Every string Glider PRO
// draws goes out in the Mac's application font -- TextFont(applFont); TextSize(12);
// TextFace(bold), set three times in InitScoreboardMap (StructuresInit.c:109,118,131) --
// which on every machine the game shipped for resolved to Geneva 12. Geneva lives in the
// System file, not in Glider PRO's resource fork, so there is nothing in the vendored
// sources to extract: a port either authors a font or draws no text at all. The scoreboard
// spent one commit drawing no text at all, and this is the other half of that.
//
// What is here is a fixed-width 5x7 core in a 6x9 cell, which is smaller and plainer than
// Geneva 12 bold. The consequences are visible and bounded:
//
//   - Every character advances six pixels, so a room title is wider or narrower than the
//     original's by however much Geneva's proportional metrics differ. The title panel is
//     256 pixels -- 42 characters at this advance -- against a 27-character room name, so
//     nothing shipped can overflow it. See docs/IMPROVEMENTS.md 2.29.
//   - There is no bold. The original's boldface is a smear of the plain face one pixel
//     right, which at this size would close every counter in the alphabet.
//   - The drop shadow is not this file's business. Scoreboard.c draws each string twice,
//     black at (1,10) then white at (0,9), and the caller keeps doing exactly that.
//
// Coverage is the one place this file is deliberately generous rather than faithful. A
// house is a user-supplied file whose room names are Mac Roman, so all 128 upper-half
// characters can turn up in a title panel, and a box where a letter should be is the kind
// of thing that gets a released game a bug report. Every one of the 128 resolves to
// something legible: most through the sheet, the accented lowercase forms by composing a
// mark over a base letter, and the rest by folding to the nearest ASCII the font does have
// (© -> "(c)", Æ -> "AE", ≤ -> "<="). font_test.go walks the whole Mac Roman upper half and
// fails if any rune falls through to the box.

import "strings"

// The cell. Five columns of glyph and one of spacing; seven rows at or above the baseline
// and two below it for descenders.
//
// fontAscent is the mapping from a cell row to a screen row: row r of a glyph drawn with
// its baseline at v lands at v-fontAscent+r, so row 6 -- the bottom of a capital -- is at
// v-1 and rows 7 and 8 hang at v and v+1. That is QuickDraw's convention, in which the
// baseline row itself carries no part of an unaccented capital, and it is what makes the
// original's pen positions land correctly in panels only ten pixels tall: black at v=10
// puts a digit in rows 3..9 of a 10-row panel, exactly filling it, and clips only the
// descenders that digits do not have.
const (
	FontWide = 6
	FontTall = 9

	fontCore   = 5
	fontAscent = 7
)

// glyph is one cell: nine rows of five bits, bit 4 leftmost.
type glyph [FontTall]uint8

// cell is what a rune resolves to -- a glyph, and optionally a mark drawn over it at the
// same pen position. A mark is a separate glyph rather than a pre-composed one because
// there are 27 accented lowercase letters in Mac Roman and thirteen marks, and composing
// costs one extra draw where pre-composing would cost 27 hand-authored cells.
type cell struct{ g, mark int }

const noMark = -1

// The marks, indexed by these constants so that the table below can be read without
// counting. Only the first seven are reachable from Mac Roman's accented letters; the rest
// exist because Mac Roman also has the twelve marks as standalone characters (´ ¨ ˆ ˜ ¯ ˘
// ˙ ˚ ¸ ˝ ˛ ˇ), and drawing those as a mark over a space is both the least code and the
// most honest rendering of what they are.
const (
	markAcute = iota
	markGrave
	markCircumflex
	markTilde
	markUmlaut
	markRing
	markCedilla
	markCaron
	markMacron
	markBreve
	markDot
	markDoubleAcute
	markOgonek
	numMarks
)

var (
	// glyphs is the sheet's 96 cells followed by the authored extras; glyphOf maps a rune
	// to an index into it.
	glyphs  []glyph
	glyphOf map[rune]int

	// marks is indexed by the mark constants above.
	marks [numMarks]glyph

	// composed covers the runes drawn as a base plus a mark: the accented lowercase
	// letters, and the standalone diacritics as a mark over a space.
	composed map[rune]cell

	// folds is the last resort before the box: a rune replaced by a short run of
	// characters the font does have. Expansion is one level deep by construction -- see
	// eachCell -- so no entry here can loop.
	folds map[rune]string

	// fallback is the hollow box, drawn for anything none of the above resolves.
	fallback int
)

// init builds all five tables in one function, in dependency order, rather than as var
// initializers that reference each other. Doing it here is deliberate: the tables cross-
// reference (composed and folds are written in terms of glyphOf), and this package has
// already been bitten once by initializer ordering -- see the note on bgrxLUT in
// surface.go, which spent a while being 256 zeroes because the compiler could not see the
// dependency it had. One init with the order written out cannot go wrong that way.
func init() {
	lines := strings.Split(strings.Trim(fontSheet, "\n"), "\n")
	if len(lines) != 6*FontTall {
		panic("render: font sheet has the wrong number of lines")
	}
	glyphs = make([]glyph, 0, 128)
	glyphOf = make(map[rune]int, 256)
	for block := 0; block < 6; block++ {
		for c := 0; c < 16; c++ {
			var g glyph
			for r := 0; r < FontTall; r++ {
				line := lines[block*FontTall+r]
				if len(line) != 16*FontWide {
					panic("render: font sheet line is not 96 characters")
				}
				g[r] = fontBits(line[c*FontWide : (c+1)*FontWide])
			}
			addGlyph(rune(0x20+block*16+c), g)
		}
	}
	fallback = glyphOf[0x7F]

	// The marks. Lowercase bodies occupy rows 2..6, so a mark at rows 0..1 never collides
	// with the letter it sits over; the two that hang below (cedilla, ogonek) use rows 7
	// and 8, which no base letter that takes them uses either.
	marks[markAcute] = authored(0, "...#.", "..#..")
	marks[markGrave] = authored(0, ".#...", "..#..")
	marks[markCircumflex] = authored(0, "..#..", ".#.#.")
	marks[markTilde] = authored(0, "#.##.", ".#..#")
	marks[markUmlaut] = authored(0, ".#.#.")
	marks[markRing] = authored(0, degreeRows...)
	marks[markCedilla] = authored(7, "..#..", ".##..")
	marks[markCaron] = authored(0, ".#.#.", "..#..")
	marks[markMacron] = authored(0, ".###.")
	marks[markBreve] = authored(0, "#...#", ".###.")
	marks[markDot] = authored(0, "..#..")
	marks[markDoubleAcute] = authored(0, ".#.#.", "#.#..")
	marks[markOgonek] = authored(7, "..#..", "..##.")

	// The authored extras: the non-ASCII characters worth drawing properly rather than
	// folding. Everything here is either something a house name plausibly contains (the
	// ellipsis is in 58 of the shipped rooms' names, the dashes and the bullet in a few
	// more) or something whose ASCII fold would be a lie about what the character is.
	//
	// The dotless i is not drawn for its own sake -- it is the base the accented i's are
	// composed over, because a mark on top of a dotted i is two dots.
	for _, e := range []struct {
		r rune
		g glyph
	}{
		{'ı', authored(2, "..#..", "..#..", "..#..", "..#..", "..#..")},
		{'…', authored(6, "#.#.#")},
		{'–', authored(3, "####.")},
		{'—', authored(3, "#####")},
		{'•', authored(3, ".##..", ".##..")},
		{'°', authored(0, degreeRows...)},
		{'¢', authored(1, "..#..", ".###.", "#.#..", "#.#..", ".###.", "..#..")},
		{'£', authored(1, "..##.", ".#...", "###..", ".#...", ".#...", "####.")},
		{'§', authored(0, ".###.", "#....", ".##..", "#..#.", ".##..", "...#.", "###..")},
		{'¶', authored(0, ".####", "##.#.", "##.#.", ".#.#.", ".#.#.", ".#.#.", ".#.#.")},
		{'†', authored(0, "..#..", "#####", "..#..", "..#..", "..#..", "..#..", "..#..")},
		{'‡', authored(0, "..#..", "#####", "..#..", "..#..", "#####", "..#..", "..#..")},
		{'◊', authored(2, "..#..", ".#.#.", "#...#", ".#.#.", "..#..")},
		{'π', authored(2, "#####", ".#.#.", ".#.#.", ".#.#.", ".#.##")},
		{'µ', authored(2, "#...#", "#...#", "#...#", "#...#", "####.", "#....", "#....")},
		{'¬', authored(3, "#####", "....#")},

		// U+F8FF is the Apple logo, Mac Roman 0xF0, and it is in the table because the
		// port's own text format can round-trip it. Five pixels is not much apple.
		{'\uf8ff', authored(0, "...#.", "..#..", ".###.", "#####", "#####", "#####", ".#.#.")},
	} {
		addGlyph(e.r, e.g)
	}

	// The composed forms. Written one per line rather than grouped by mark because the
	// base is not always the obvious letter -- the accented i's sit over a dotless i --
	// and a table this shape can be checked against Mac Roman's own listing by eye.
	composed = make(map[rune]cell, 64)
	for _, a := range []struct {
		r, base rune
		mark    int
	}{
		{'à', 'a', markGrave}, {'á', 'a', markAcute}, {'â', 'a', markCircumflex},
		{'ä', 'a', markUmlaut}, {'ã', 'a', markTilde}, {'å', 'a', markRing},
		{'è', 'e', markGrave}, {'é', 'e', markAcute}, {'ê', 'e', markCircumflex},
		{'ë', 'e', markUmlaut},
		{'ì', 'ı', markGrave}, {'í', 'ı', markAcute}, {'î', 'ı', markCircumflex},
		{'ï', 'ı', markUmlaut},
		{'ò', 'o', markGrave}, {'ó', 'o', markAcute}, {'ô', 'o', markCircumflex},
		{'ö', 'o', markUmlaut}, {'õ', 'o', markTilde},
		{'ù', 'u', markGrave}, {'ú', 'u', markAcute}, {'û', 'u', markCircumflex},
		{'ü', 'u', markUmlaut},
		{'ñ', 'n', markTilde}, {'ç', 'c', markCedilla}, {'ÿ', 'y', markUmlaut},

		// The twelve standalone diacritics, as the mark alone.
		{'´', ' ', markAcute}, {'¨', ' ', markUmlaut}, {'ˆ', ' ', markCircumflex},
		{'˜', ' ', markTilde}, {'¯', ' ', markMacron}, {'˘', ' ', markBreve},
		{'˙', ' ', markDot}, {'˚', ' ', markRing}, {'¸', ' ', markCedilla},
		{'˝', ' ', markDoubleAcute}, {'˛', ' ', markOgonek}, {'ˇ', ' ', markCaron},
	} {
		g, ok := glyphOf[a.base]
		if !ok {
			panic("render: composed form " + string(a.r) + " has no base glyph")
		}
		composed[a.r] = cell{g, a.mark}
	}

	folds = make(map[rune]string, 96)

	// Accented capitals fold to the bare capital. A seven-row capital reaches the top of
	// the cell, so there is nowhere to put the mark: the choice is between dropping the
	// accent and drawing a letter with a mark inside it, and a plain É reads better than
	// an E with a stripe through the top bar. IMPROVEMENTS 2.29 owes a taller font.
	for _, p := range []struct{ base, forms string }{
		{"A", "ÁÀÂÄÃÅ"}, {"C", "Ç"}, {"E", "ÉÈÊË"}, {"I", "ÍÌÎÏ"}, {"N", "Ñ"},
		{"O", "ÓÒÔÖÕØ"}, {"U", "ÚÙÛÜ"}, {"Y", "Ÿ"},
	} {
		for _, r := range p.forms {
			folds[r] = p.base
		}
	}

	// Ligatures and the letters that are two letters.
	for r, s := range map[rune]string{
		'Æ': "AE", 'æ': "ae", 'Œ': "OE", 'œ': "oe", 'ø': "o", 'ß': "ss",
		'ﬁ': "fi", 'ﬂ': "fl",
	} {
		folds[r] = s
	}

	// Punctuation and symbols. Curly quotes lose their curl, the guillemets become
	// chevrons, and the arithmetic that needs two glyphs gets two.
	for r, s := range map[rune]string{
		'\u00a0': " ", // the no-break space, escaped because it is invisible
		'‘':      "'", '’': "'", '‚': ",", '“': "\"", '”': "\"", '„': ",,",
		'«': "<<", '»': ">>", '‹': "<", '›': ">",
		'·': "•", '‰': "%", '⁄': "/", '÷': "/",
		'¿': "?", '¡': "!",
		'®': "(R)", '©': "(c)", '™': "TM",
		'≠': "!=", '≤': "<=", '≥': ">=", '≈': "~", '±': "+-",
		'¥': "Y", '€': "E", 'ƒ': "f", 'ª': "a", 'º': "o",
		'∞': "oo", '√': "v", '∂': "d", '∑': "S", '∏': "P", '∫': "S", '∆': "D", 'Ω': "O",
	} {
		folds[r] = s
	}
}

// degreeRows is the ring, used twice: as the ° glyph and as the mark over å and ˚.
var degreeRows = []string{".##..", ".##.."}

// addGlyph appends a cell and registers the rune that draws it.
func addGlyph(r rune, g glyph) {
	glyphOf[r] = len(glyphs)
	glyphs = append(glyphs, g)
}

// authored builds a glyph from rows given top-down, the first of them at row top. It is
// how everything outside the sheet is written, and it takes the same '.' and '#' notation
// so that a glyph reads the same wherever it appears in this file.
func authored(top int, rows ...string) glyph {
	var g glyph
	if top < 0 || top+len(rows) > FontTall {
		panic("render: authored glyph does not fit the cell")
	}
	for i, row := range rows {
		if len(row) != fontCore {
			panic("render: authored glyph row is not 5 characters")
		}
		g[top+i] = fontBits(row)
	}
	return g
}

// fontBits packs one row of a cell. It is strict about three things, all of which are the
// kind of mistake that would otherwise show up as one wrong pixel in one letter: only '.'
// and '#' are accepted, so a stray space cannot read as a blank; the row may not be longer
// than the cell; and column five -- the spacing column -- must be empty, because the
// advance is the whole cell width and a glyph that used it would touch its neighbour.
func fontBits(row string) uint8 {
	var v uint8
	for c := 0; c < len(row); c++ {
		switch row[c] {
		case '.':
		case '#':
			if c >= fontCore {
				panic("render: font glyph uses the spacing column")
			}
			v |= 1 << (fontCore - 1 - c)
		default:
			panic("render: font glyph row contains something other than '.' and '#'")
		}
	}
	return v
}

// eachCell walks text and calls fn once for every cell that will be drawn, in order.
//
// src is the rune the cell came from, which is the same for every cell of a folded rune,
// and ok is false when nothing resolved and the box was substituted. The fold arm resolves
// its expansion through the glyphs and the composed forms but never through folds again,
// which is what bounds this at one level and makes a cycle in the table impossible rather
// than merely absent.
func eachCell(text string, fn func(src rune, c cell, ok bool)) {
	for _, r := range text {
		if g, ok := glyphOf[r]; ok {
			fn(r, cell{g, noMark}, true)
			continue
		}
		if c, ok := composed[r]; ok {
			fn(r, c, true)
			continue
		}
		if s, ok := folds[r]; ok {
			for _, f := range s {
				if g, ok := glyphOf[f]; ok {
					fn(r, cell{g, noMark}, true)
				} else if c, ok := composed[f]; ok {
					fn(r, c, true)
				} else {
					fn(r, cell{fallback, noMark}, false)
				}
			}
			continue
		}
		fn(r, cell{fallback, noMark}, false)
	}
}

// DrawString is QuickDraw's DrawString: draw text with its left edge at h and its baseline
// at v, in the given colour index, clipped to the surface.
//
// The pen is not advanced, because nothing in the game draws two strings in a row without
// a MoveTo in between; StringWidth is there for the one caller that needs to know how wide
// a string came out (DrawCalendar centres its month).
func (s *Surface) DrawString(h, v int16, text string, idx uint8) {
	x := h
	eachCell(text, func(_ rune, c cell, _ bool) {
		s.drawGlyph(x, v, glyphs[c.g], idx)
		if c.mark != noMark {
			s.drawGlyph(x, v, marks[c.mark], idx)
		}
		x += FontWide
	})
}

// drawGlyph plots one cell. Rows that are entirely blank -- most of them, in most glyphs
// -- cost one test.
func (s *Surface) drawGlyph(h, v int16, g glyph, idx uint8) {
	for r := 0; r < FontTall; r++ {
		bits := g[r]
		if bits == 0 {
			continue
		}
		y := int(v) - fontAscent + r
		for c := 0; c < fontCore; c++ {
			if bits&(1<<(fontCore-1-c)) != 0 {
				s.plot(int(h)+c, y, idx)
			}
		}
	}
}

// StringWidth is the Toolbox StringWidth: how many pixels DrawString will advance over.
// Fixed-width, so this is a cell count times the cell width -- but it is a cell count and
// not a rune count, because a folded rune draws more than one cell.
func StringWidth(text string) int16 {
	var w int16
	eachCell(text, func(rune, cell, bool) { w += FontWide })
	return w
}

// UnrenderableRunes reports the distinct runes in text that the font has no representation
// for and would draw as the hollow box, in the order they first appear.
//
// It exists for two callers. font_test.go asserts it is empty for every room name in the
// 22 shipped houses and for the whole Mac Roman upper half, which is what keeps the fold
// table honest as it grows. The house linter of IMPROVEMENTS 4.1 will want the same answer
// about a house someone else wrote.
func UnrenderableRunes(text string) []rune {
	var out []rune
	var seen map[rune]bool
	eachCell(text, func(src rune, _ cell, ok bool) {
		if ok || seen[src] {
			return
		}
		if seen == nil {
			seen = make(map[rune]bool, 4)
		}
		seen[src] = true
		out = append(out, src)
	})
	return out
}

// fontSheet is the glyph data: 96 cells in ASCII order, sixteen to a block and nine lines
// each, so every line below is exactly 96 characters and each block spells out sixteen
// consecutive characters of the ASCII table.
//
// Blank pixels are '.' and not spaces. That is not a style choice: spaces would put
// trailing whitespace at the end of most of these lines, and the first editor or hook that
// stripped it would silently delete the right-hand column of every glyph in the font.
//
// The last cell -- 0x7F, where DEL would be -- is the hollow box drawn for any character
// the font cannot render. It is deliberately conspicuous.
const fontSheet = "" +
	// 20..2F   space ! " # $ % & ' ( ) * + , - . /
	"........#....#.#...#.#....#...##.....##.....#......#...#......................................#.\n" +
	"........#....#.#...#.#...####.##..#.#..#....#.....#.....#.....#.....#.........................#.\n" +
	"........#.........#####.#.#......#..#..#.........#.......#..#.#.#...#........................#..\n" +
	"........#..........#.#...###....#....##..........#.......#...###..#####........###..........#...\n" +
	"........#.........#####...#.#..#....#.#.#........#.......#..#.#.#...#......................#....\n" +
	"...................#.#..####..#..##.#..#..........#.....#.....#.....#.....##..............#.....\n" +
	"........#..........#.#....#......##..##.#..........#...#..................#...........#...#.....\n" +
	".........................................................................#......................\n" +
	"................................................................................................\n" +
	// 30..3F   0 1 2 3 4 5 6 7 8 9 : ; < = > ?
	".###....#....###..#####....#..#####...##..#####..###...###..................#.......#......###..\n" +
	"#...#..##...#...#....#....##..#......#........#.#...#.#...#...#.....#......#.........#....#...#.\n" +
	"#..##...#.......#...#....#.#..####..#........#..#...#.#...#...............#...#####...#.......#.\n" +
	"#.#.#...#......#.....#..#..#......#.####....#....###...####..............#.............#.....#..\n" +
	"##..#...#.....#.......#.#####.....#.#...#..#....#...#.....#...............#...#####...#.....#...\n" +
	"#...#...#....#....#...#....#..#...#.#...#..#....#...#....#....#.....##.....#.........#..........\n" +
	".###...###..#####..###.....#...###...###...#.....###...##...........#.......#.......#.......#...\n" +
	"...................................................................#............................\n" +
	"................................................................................................\n" +
	// 40..4F   @ A B C D E F G H I J K L M N O
	".###....#...####...###..####..#####.#####..###..#...#..###....###.#...#.#.....#...#.#...#..###..\n" +
	"#...#..#.#..#...#.#...#.#...#.#.....#.....#...#.#...#...#......#..#..#..#.....##.##.##..#.#...#.\n" +
	"....#.#...#.#...#.#.....#...#.#.....#.....#.....#...#...#......#..#.#...#.....#.#.#.#.#.#.#...#.\n" +
	".##.#.#...#.####..#.....#...#.####..####..#..##.#####...#......#..##....#.....#...#.#..##.#...#.\n" +
	"#.#.#.#####.#...#.#.....#...#.#.....#.....#...#.#...#...#......#..#.#...#.....#...#.#...#.#...#.\n" +
	"#.#.#.#...#.#...#.#...#.#...#.#.....#.....#...#.#...#...#...#..#..#..#..#.....#...#.#...#.#...#.\n" +
	".####.#...#.####...###..####..#####.#......###..#...#..###...##...#...#.#####.#...#.#...#..###..\n" +
	"................................................................................................\n" +
	"................................................................................................\n" +
	// 50..5F   P Q R S T U V W X Y Z [ \ ] ^ _
	"####...###..####...###..#####.#...#.#...#.#...#.#...#.#...#.#####...###.#.....###.....#.........\n" +
	"#...#.#...#.#...#.#...#...#...#...#.#...#.#...#.#...#.#...#.....#...#...#.......#....#.#........\n" +
	"#...#.#...#.#...#.#.......#...#...#.#...#.#...#..#.#...#.#.....#....#....#......#...#...#.......\n" +
	"####..#...#.####...###....#...#...#.#...#.#.#.#...#.....#.....#.....#.....#.....#...............\n" +
	"#.....#.#.#.#.#.......#...#...#...#.#...#.#.#.#..#.#....#....#......#......#....#...............\n" +
	"#.....#..#..#..#..#...#...#...#...#..#.#..##.##.#...#...#...#.......#.......#...#...............\n" +
	"#......##.#.#...#..###....#....###....#...#...#.#...#...#...#####...###.....#.###.........#####.\n" +
	"................................................................................................\n" +
	"................................................................................................\n" +
	// 60..6F   ` a b c d e f g h i j k l m n o
	".#..........#...............#.........##........#.......#......#..#......##.....................\n" +
	"..#.........#...............#........#..........#.................#.......#.....................\n" +
	".......###..####...###...####..###...#.....####.####....#......#..#..#....#...##.#..####...###..\n" +
	"..........#.#...#.#.....#...#.#...#.####..#...#.#...#...#......#..#.#.....#...#.#.#.#...#.#...#.\n" +
	".......####.#...#.#.....#...#.#####..#....#...#.#...#...#......#..##......#...#.#.#.#...#.#...#.\n" +
	"......#...#.#...#.#.....#...#.#......#....#...#.#...#...#......#..#.#.....#...#.#.#.#...#.#...#.\n" +
	".......####.####...###...####..###...#.....####.#...#...#......#..#..#...###..#.#.#.#...#..###..\n" +
	"..............................................#.............#..#................................\n" +
	"...........................................###...............##.................................\n" +
	// 70..7F   p q r s t u v w x y z { | } ~ box
	".........................#...........................................##...#...##..........#####.\n" +
	".........................#..........................................#.....#.....#.........#...#.\n" +
	"####...####.#.##...####.####..#...#.#...#.#...#.#...#.#...#.#####...#.....#.....#....#..#.#...#.\n" +
	"#...#.#...#.##..#.#......#....#...#.#...#.#...#..#.#..#...#....#...#......#......#..#.#.#.#...#.\n" +
	"#...#.#...#.#......###...#....#...#.#...#.#.#.#...#...#...#...#.....#.....#.....#...#..#..#...#.\n" +
	"#...#.#...#.#.........#..#..#.#...#..#.#..#.#.#..#.#..#...#..#......#.....#.....#.........#...#.\n" +
	"####...####.#.....####....##...####...#....#.#..#...#..####.#####....##...#...##..........#####.\n" +
	"#.........#...............................................#.....................................\n" +
	"#.........#............................................###......................................\n"
