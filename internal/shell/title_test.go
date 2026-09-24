package shell

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/credits"
	"github.com/bwenstar/gliderGo/internal/render"
)

// shippedArt is the working tree's extracted art, or a skip: a checkout ships it, and one
// that has run `make clean-assets` has nothing here to compare against.
func shippedArt(t *testing.T) *render.Assets {
	t.Helper()
	dir := filepath.Join("..", "..", "assets", "extracted", "art")
	if _, err := os.Stat(filepath.Join(dir, "ui", "1000.png")); err != nil {
		t.Skipf("no extracted art to test the title against: %v", err)
	}
	return render.NewAssets(os.DirFS(dir))
}

func shippedSplash(t *testing.T) *render.Surface {
	t.Helper()
	a := shippedArt(t)
	art := a.UI(1000)
	if art == nil {
		t.Fatalf("ui/1000.png is there and did not load: %v", a.Err())
	}
	return art
}

// stamped is the screen drawBackdrop leaves over art, before any panel is drawn on it.
func stamped(art *render.Surface) *render.Surface {
	scr := render.NewSurface(screenWide, screenTall)
	scr.Copy(art, art.Bounds(), art.Bounds(), render.SrcCopy)
	stampTitle(scr, art)
	return scr
}

// stampInk is every pixel the stamp inks, keyed by v*screenWide+h.
func stampInk() map[int]uint8 {
	m := map[int]uint8{}
	eachTitlePixel(func(h, v int, idx uint8) { m[v*screenWide+h] = idx })
	return m
}

func in(r render.Rect, h, v int) bool {
	return h >= int(r.Left) && h < int(r.Right) && v >= int(r.Top) && v < int(r.Bottom)
}

// The claim the whole overlay rests on: it adds to Calhoun's logo and takes nothing from it
// except PRO. Outside PRO's box a pixel is either the PICT's or ink laid on the PICT's sky,
// so his "Glider", his credit and the plane are his to the pixel; inside the box nothing of
// PRO is left, only sky and the mark.
func TestTheTitleKeepsCalhounsLogo(t *testing.T) {
	art := shippedSplash(t)
	scr := stamped(art)
	ink := stampInk()
	if len(ink) == 0 {
		t.Fatal("the stamp inks nothing")
	}

	changed, bad := 0, 0
	for v := 0; v < art.H; v++ {
		for h := 0; h < art.W; h++ {
			i := v*screenWide + h
			got, was := scr.Pix[i], art.Pix[v*art.W+h]
			want, inked := ink[i]
			switch {
			case in(titlePRO, h, v):
				if !inked {
					want = titleSky
				}
			case inked:
				if was != titleSky {
					if bad++; bad <= 5 {
						t.Errorf("(%d,%d): the stamp inks over the PICT's %d, not over sky", h, v, was)
					}
				}
			default:
				want = was
			}
			if got != want {
				if bad++; bad <= 5 {
					t.Errorf("(%d,%d) = %d, want %d", h, v, got, want)
				}
			}
			if got != was {
				changed++
			}
		}
	}
	if changed == 0 {
		t.Error("the shipped splash came out unchanged: stampTitle did not recognise it")
	}
}

// And the ink is PRO's, which is what makes the light brown read as a thing added: every
// pixel PRO has is one of the three the mark and the credit are drawn in.
func TestTheTitleIsInPROsInk(t *testing.T) {
	art := shippedSplash(t)
	want := map[uint8]color.RGBA{
		titleSky:  {0xFF, 0xCC, 0x33, 0xFF},
		titleEdge: {0xCC, 0x99, 0x33, 0xFF},
		titleHalf: {0xCC, 0x99, 0x00, 0xFF},
		titleInk:  {0x99, 0x66, 0x00, 0xFF},
	}
	for idx, c := range want {
		if render.Palette[idx] != c {
			t.Errorf("palette index %d is %v; the comment says %v", idx, render.Palette[idx], c)
		}
	}
	pro := 0
	for v := int(titlePRO.Top); v < int(titlePRO.Bottom); v++ {
		for h := int(titlePRO.Left); h < int(titlePRO.Right); h++ {
			idx := art.Pix[v*art.W+h]
			if _, ok := want[idx]; !ok {
				t.Fatalf("PRO has a pixel of index %d at (%d,%d), which the stamp has no ink for",
					idx, h, v)
			}
			if idx != titleSky {
				pro++
			}
		}
	}
	if pro == 0 {
		t.Error("there is no PRO in titlePRO to paint out")
	}
}

// The mark keeps three pixels of sky between itself and anything the PICT drew, which is
// what the free rows between the r and the plane allowed at 0.6; the credit keeps one, below
// Calhoun's descenders. Anything closer reads as touching at 1:1.
func TestTheTitleClearsTheArt(t *testing.T) {
	art := shippedSplash(t)
	eachTitlePixel(func(h, v int, _ uint8) {
		r := 3
		if v >= creditTop {
			r = 1
		}
		for y := v - r; y <= v+r; y++ {
			for x := h - r; x <= h+r; x++ {
				if in(titlePRO, x, y) {
					continue
				}
				if idx := art.Pix[y*art.W+x]; idx != titleSky {
					t.Fatalf("the stamp's (%d,%d) is within %d of the PICT's %d at (%d,%d)",
						h, v, r, idx, x, y)
				}
			}
		}
	})
}

// Every letter the credit shares with "by john calhoun" is that letter, found where Calhoun's
// line has it, 13 rows up, in the ink his line uses for the same cell.
func TestTheCreditReusesCalhounsLetters(t *testing.T) {
	art := shippedSplash(t)
	dark := map[uint8]byte{137: '#', 95: 'x', 52: '+'} // #663300, #996600, #CC9933
	cell := func(h, v int) byte {
		if c, ok := dark[art.Pix[v*art.W+h]]; ok {
			return c
		}
		return '.'
	}
	const top = creditTop - 13
	for _, r := range creditLetters {
		if !strings.ContainsRune("by john calhoun", r) {
			continue
		}
		g := creditGlyphs[r]
		found := false
		for h := 160; h+len(g[0]) <= 290 && !found; h++ {
			found = true
			for y, row := range g {
				for x := 0; x < len(row) && found; x++ {
					found = cell(h+x, top+y) == row[x]
				}
			}
		}
		if !found {
			t.Errorf("the credit's %q is not the one in Calhoun's line", r)
		}
	}
}

// The strip is data a person edits by hand, so its shape is checked rather than trusted, and
// so is everything the stamp inks: inside titleArea, which is what titleSum covers, and clear
// of the menu panel's halo, which would dim it.
func TestTheTitleGridsAreWellFormed(t *testing.T) {
	for i, row := range creditStrip {
		if n := strings.Count(row, "|") + 1; n != len(creditLetters) {
			t.Errorf("creditStrip row %d has %d glyphs, want %d", i, n, len(creditLetters))
		}
	}
	for _, r := range creditLetters {
		g := creditGlyphs[r]
		if len(g) != len(creditStrip) {
			t.Errorf("%q has %d rows, want %d", r, len(g), len(creditStrip))
			continue
		}
		for i, row := range g {
			if len(row) != len(g[0]) {
				t.Errorf("%q row %d is %d wide, row 0 is %d", r, i, len(row), len(g[0]))
			}
		}
	}
	for _, r := range creditText {
		if _, ok := creditGlyphs[r]; !ok && r != ' ' {
			t.Errorf("creditText has %q and creditStrip does not", r)
		}
	}
	for i, row := range titleMark {
		if len(row) != len(titleMark[0]) {
			t.Errorf("titleMark row %d is %d wide, row 0 is %d", i, len(row), len(titleMark[0]))
		}
	}
	for _, rows := range [][]string{titleMark[:], creditStrip[:]} {
		for _, row := range rows {
			if s := strings.Trim(row, ".+x#|"); s != "" {
				t.Errorf("a grid row has %q in it, which draws nothing: %s", s, row)
			}
		}
	}

	halo := menuRect(1)
	eachTitlePixel(func(h, v int, _ uint8) {
		if !in(titleArea, h, v) {
			t.Fatalf("the stamp inks (%d,%d), outside titleArea %v", h, v, titleArea)
		}
		if h >= int(halo.Left)-8 && v >= int(halo.Top)-8 {
			t.Fatalf("the stamp inks (%d,%d), under the menu panel's halo", h, v)
		}
	})
}

// A splash that is not the one the stamp was drawn against is drawn as it is. One pixel is
// enough to tell, and one pixel is what this changes.
func TestTheTitleLeavesOtherSplashesAlone(t *testing.T) {
	art := shippedSplash(t).Clone()
	art.Pix[50*art.W+300] ^= 1 // in PRO
	scr := stamped(art)
	for v := 0; v < art.H; v++ {
		for h := 0; h < art.W; h++ {
			if scr.Pix[v*screenWide+h] != art.Pix[v*art.W+h] {
				t.Fatalf("(%d,%d) was stamped on a splash that is not the shipped one", h, v)
			}
		}
	}
	if isShippedTitle(nil) || isShippedTitle(render.NewSurface(200, 100)) {
		t.Error("no plate, or one too small to hold the logo, is not the shipped splash")
	}
}

// And the screen a player sees has it: Draw goes through drawBackdrop, which is the only
// caller, and every pixel of the mark and the credit is on the splash as drawn.
func TestTheSplashCarriesTheTitle(t *testing.T) {
	a := shippedArt(t)
	s, f := shellOver(t, []string{"Slumberland"})
	s.host.Assets = a
	s.Draw()
	eachTitlePixel(func(h, v int, idx uint8) {
		if got := f.scr.Pix[v*screenWide+h]; got != idx {
			t.Fatalf("the splash has %d at (%d,%d), and the stamp inks %d there", got, h, v, idx)
		}
	})
}

// A name painted into pixels is one nothing can read, which is credits.txt's own complaint
// about PICT 153. So whoever the splash says ported this is also a row under [this port],
// which is text: the credits screen draws it, and a search of the source finds it. It is
// the row the no-art title screen reads its "ported by" line from, too (porter).
func TestTheTitlesCreditIsInTheCredits(t *testing.T) {
	who, ok := strings.CutPrefix(creditText, "ported by ")
	if !ok {
		t.Fatalf("creditText is %q; this test reads the name after \"ported by \"", creditText)
	}
	if !strings.EqualFold(porter(), who) {
		t.Errorf("the splash says %q, and the no-art title screen says \"ported by %s\"",
			creditText, porter())
	}
	for _, sec := range credits.Sections() {
		if sec.Title != "this port" {
			continue
		}
		for _, r := range sec.Rows {
			if strings.EqualFold(r.Who, who) {
				return
			}
		}
	}
	t.Errorf("the splash says %q, and credits.txt's [this port] names nobody called %q",
		creditText, who)
}
