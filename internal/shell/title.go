package shell

import (
	"hash/fnv"
	"strings"

	"github.com/bwenstar/gliderGo/internal/render"
)

// The name on the splash.
//
// PICT 1000's logo says "Glider PRO": "Glider" in Calhoun's dark brown, "PRO" stood on end
// beside it in a lighter one, and "by john calhoun" underneath. This program is not Glider
// PRO, and the title screen is the first thing it says about itself, so the shell paints PRO
// out, writes "Go" where it stood, and adds a line under Calhoun's credit naming who ported
// it. Everything else on the plate -- his "Glider", his credit, the plane -- is drawn exactly
// as 1994 drew it, and TestTheTitleKeepsCalhounsLogo holds every one of those pixels to the
// PICT.
//
// All of it is taken from the plate's own hand rather than from the port's font, which would
// have been the one thing on the screen that did not look drawn. The G of Go is Calhoun's G
// and the o is the bowl of his d turned round, both averaged down to 0.6 of their size; the
// credit line's b, y, o, n and a are the ones in "by john calhoun", pixel for pixel, and its
// p, r, t, e and d are drawn to match them. And all of it is in PRO's lighter brown, which
// keeps a distinction the logo already made: the dark ink is Calhoun's, and the light ink is
// what was added to it.
//
// This paints the screen and never the plate. assets/extracted is still the 1994 art byte
// for byte, which `make assets-check` proves, and the About box and the credits still say
// whose game this is.

// The ink, by palette index as cream and label are:
//
//	10 = (0,1,4) = #FFCC33, the sky behind the logo
//	52 = (1,2,4) = #CC9933, PRO's outer edge
//	53 = (1,2,5) = #CC9900, PRO's inner edge
//	95 = (2,3,5) = #996600, PRO itself
const (
	titleSky  = 10
	titleEdge = 52
	titleHalf = 53
	titleInk  = 95
)

// titleColour is what a cell of titleMark or creditStrip draws. '.' draws nothing.
func titleColour(c byte) (uint8, bool) {
	switch c {
	case '+':
		return titleEdge, true
	case 'x':
		return titleHalf, true
	case '#':
		return titleInk, true
	}
	return 0, false
}

// Where it goes. titlePRO is the box PRO stands in, with a few pixels of sky round it. The
// mark is centred on the row PRO was, with three pixels of sky between it and both the r and
// the plane (TestTheTitleClearsTheArt). The credit ends where Calhoun's does, at column 281,
// a line lower.
var titlePRO = render.SetRect(287, 15, 322, 88)

const (
	titleMarkH = 287
	titleMarkV = 33

	creditText  = "ported by brendan ta"
	creditRight = 282 // one past the last column, as a Rect's Right is
	creditTop   = 103
	creditSpace = 5 // between words, as between "by" and "john"
)

// titleArea holds every pixel stampTitle writes, and titleSum is FNV-1a over the shipped PICT
// 1000's pixels in it, as palette indices, row by row.
//
// The stamp is at fixed coordinates, which is right only over the picture it was drawn
// against. -art can name any tree, and a splash in it can be anything: a picture with sky
// where Calhoun had no PRO would get a brown smudge. So if the plate under the area is not
// the 1994 one, stampTitle leaves the screen as the plate drew it.
var titleArea = render.SetRect(120, 15, 352, 116)

const titleSum = 0x005d4bd08a4438fb

// stampTitle turns the 1994 title on scr into this port's. art is the plate that has just
// been copied to scr at the origin.
func stampTitle(scr, art *render.Surface) {
	if !isShippedTitle(art) {
		return
	}
	scr.Fill(titlePRO, titleSky)
	eachTitlePixel(func(h, v int, idx uint8) { scr.Pix[v*scr.W+h] = idx })
}

// eachTitlePixel calls f for every pixel of the mark and the credit line, with its ink. The
// screen is always 640x480 (New refuses any other) and titleArea is inside it, so nothing
// here needs clipping.
func eachTitlePixel(f func(h, v int, idx uint8)) {
	eachCell(titleMarkH, titleMarkV, titleMark[:], f)
	h := creditRight - creditWidth()
	for _, r := range creditText {
		if r == ' ' {
			h += creditSpace
			continue
		}
		g := creditGlyphs[r]
		eachCell(h, creditTop, g, f)
		h += len(g[0]) + 1
	}
}

// eachCell is eachTitlePixel for one grid, with its top-left at (h, v).
func eachCell(h, v int, rows []string, f func(h, v int, idx uint8)) {
	for y, row := range rows {
		for x := 0; x < len(row); x++ {
			if idx, ok := titleColour(row[x]); ok {
				f(h+x, v+y, idx)
			}
		}
	}
}

// isShippedTitle reports whether art is the splash the stamp was drawn against.
func isShippedTitle(art *render.Surface) bool {
	a := titleArea
	if art == nil || art.W < int(a.Right) || art.H < int(a.Bottom) {
		return false
	}
	sum := fnv.New64a()
	for v := int(a.Top); v < int(a.Bottom); v++ {
		sum.Write(art.Pix[v*art.W+int(a.Left) : v*art.W+int(a.Right)])
	}
	return sum.Sum64() == titleSum
}

// creditWidth is creditText's width in pixels: each letter's own and one between letters,
// and creditSpace for a space.
func creditWidth() int {
	w := 0
	for _, r := range creditText {
		if r == ' ' {
			w += creditSpace
		} else {
			w += len(creditGlyphs[r][0]) + 1
		}
	}
	return w - 1
}

// titleMark is "Go", 64x40. Its G is PICT 1000's own (columns 16-75, rows 17-82) and its o is
// the left half of the d's bowl (columns 142-162, rows 39-82) mirrored into a whole one, the
// two set six pixels apart on Glider's baseline. Each pixel's ink is how far its luminance
// lies from the sky's towards #663300's; the pair was box-filtered to 0.6 of its size and the
// result cut into PRO's three shades at 18%, 40% and 62%.
var titleMark = [...]string{
	"...............+xx#######xx+....................................",
	".............+################..................................",
	"...........x####################+...............................",
	".........x######xx+..++x#########x..............................",
	"........x#####...........+########..............................",
	".......#####+.............+######x..............................",
	"......#####...............x######...............................",
	".....x####................######................................",
	"....+####...............+######.................................",
	"....####x..............+######..................................",
	"...#####.............x#####x....................................",
	"...####+............#####x+.....................................",
	"..x####...........x#####........................................",
	"..#####..........####x..........................................",
	".x####x........####x..........................x#########x.......",
	".#####+.......####...........................#############......",
	".#####+.....+###+...........................######xxx######+....",
	"+#####......###............................#####x......#####x...",
	"x#####.....x##x......+++........+++.......######.......+#####...",
	"x#####+....x###+....+#######x#######.....x#####.........######..",
	"x#####x....x####.....##############+....+#####x..........######.",
	"x#####x.....####......############+.....######...........x#####.",
	"x######.....+##x.......x#########x......######...........x#####x",
	"x######.................#########.......######............######",
	"x######+................+#######.......+#####x............######",
	"+#######................+#######.......x#####x............######",
	".#######x...............+#######.......x#####x............######",
	".########...............+#######.......x#####x............######",
	".x########..............+#######.......x######............######",
	"..#########.............+#######.......+######............######",
	"..##########............+#######.......+######...........x######",
	"...###########x.........########........######+..........######x",
	"...##############++++x##########........#######.........+######x",
	"...+############################........#######x.......+#######.",
	"....+###########################........+########xxxxx########x.",
	".....+###########################+.......#####################..",
	"......x###########################........###################+..",
	".......+##########################x........#################x...",
	".........#########################..........###############+....",
	"...........x########xxxxxxx#####x............x###########x......",
}

// creditStrip is the credit line's alphabet, one glyph per '|'-separated column in the order
// of creditLetters, on a common grid whose row 0 is creditTop and whose baseline is row 9.
// Calhoun's credit is a 7-pixel x-height, 2-pixel-stem serif, and the five letters he did
// not need are drawn to it.
const creditLetters = "abdenoprty"

var creditStrip = [...]string{
	"........|+++.....|.....+++.|........|........|........|........|.......|......|.......",
	"........|.##.....|......##.|........|........|........|........|.......|..+...|.......",
	"........|.##.....|......##.|........|........|........|........|.......|.x#...|.......",
	"..++.++.|.##.++..|...++.##.|..+++...|++++.+..|..+++...|+++.++..|++++.++|.##...|+++++++",
	".+#++x#.|.##x.x#.|.x#++###.|.x#++#x.|.##xx##.|.x#++#x.|.##x#x#.|.##x###|####x.|x#x+x.#",
	"...xxx#.|.##..+#x|+#x..x##.|+#x..x#.|.##..x#.|+#x..x#.|.##..+#x|.##+.x+|.##...|.#x..+x",
	"..#x.x#.|.##...#x|x#....##.|x######+|.##..x#.|x#....#+|.##...#x|.##....|.##...|.x#..x+",
	".x#..x#.|.##..+#x|x#+...##.|x#+.....|.##..x#.|x#+..+#+|.##..+#x|.##....|.##...|..#x.x.",
	".x#x.##+|.##++##+|+#x++###.|+#x..+#.|.##..x#.|+#x++x#.|.###+##+|.##....|.##+.x|..x##+.",
	"..######|.x####x.|.x####x#x|.x####x.|x##x+##x|.x####x.|.##x##x.|x##x+..|..x##+|..+#x..",
	"........|........|.........|........|........|........|.##.....|.......|......|.+x#x..",
	"........|........|.........|........|........|........|.##.....|.......|......|.+##x..",
	"........|........|.........|........|........|........|x##x+...|.......|......|...x+..",
}

// creditGlyphs is creditStrip cut into letters.
var creditGlyphs = func() map[rune][]string {
	m := make(map[rune][]string, len(creditLetters))
	for _, row := range creditStrip {
		for i, cell := range strings.Split(row, "|") {
			r := rune(creditLetters[i])
			m[r] = append(m[r], cell)
		}
	}
	return m
}()
