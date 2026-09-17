package render

// Assets.UI is the one accessor in this file for which a missing file is not a fault,
// and that exception is the whole of what needs testing here. Every other accessor
// draws a room, and a room without its art is a bug that Err has to report; UI draws
// the shell, which can always fall back to its own chrome, and a first run against a
// tree with no extracted assets has to reach a title screen that says so rather than an
// error (docs/IMPROVEMENTS.md 2.6). An exception nothing tests is an exception that
// quietly stops being one.

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writePlate puts a w x h PNG of palette-legal pixels at dir/<id>.png.
func writePlate(t *testing.T, dir string, id int16, w, h int) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// On the palette, because surfaceFromImage refuses anything else --
			// deliberately, so that art which no longer matches the palette the game
			// ORs indices against is a hard error rather than a snap to the nearest.
			c := Palette[uint8(1+(x+y)%8)]
			img.Set(x, y, color.RGBA{c.R, c.G, c.B, 0xFF})
		}
	}
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%d.png", id)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// A missing plate is nil and leaves Err clean. If it recorded, cmd/glidergo would
// print an asset error after every game on a build with no extracted art -- which is
// the supported first-run state, not a fault.
func TestUIMissingPlateIsNotAnError(t *testing.T) {
	a := NewAssets(dirFS(t.TempDir()))
	if got := a.UI(1000); got != nil {
		t.Errorf("UI(1000) returned a %dx%d surface from an empty tree", got.W, got.H)
	}
	if err := a.Err(); err != nil {
		t.Errorf("a missing plate recorded a sticky error: %v", err)
	}

	// A root that is not there at all takes the same path, because that is what
	// `-art /nonexistent` and a fresh clone both look like. make headless renders the
	// shell that way on purpose.
	b := NewAssets(dirFS(filepath.Join(t.TempDir(), "no", "such", "tree")))
	if got := b.UI(1000); got != nil {
		t.Error("a missing art root returned a surface")
	}
	if err := b.Err(); err != nil {
		t.Errorf("a missing art root recorded: %v", err)
	}

	// And the sticky error stays clean across the ids the shell actually asks for on
	// its way to a title screen, since one recording id would be enough to spoil it.
	for _, id := range []int16{150, 151, 153, 1000, 1001, 1002, 1003, 1004, 1015, 1016} {
		if got := b.UI(id); got != nil || b.Err() != nil {
			t.Fatalf("UI(%d) on an empty tree gave %v / %v", id, got, b.Err())
		}
	}
}

func TestUIReturnsThePlateThatIsThere(t *testing.T) {
	root := t.TempDir()
	writePlate(t, filepath.Join(root, "ui"), 1000, 8, 5)

	a := NewAssets(dirFS(root))
	s := a.UI(1000)
	if s == nil {
		t.Fatalf("UI(1000) is nil with the plate present: %v", a.Err())
	}
	if s.W != 8 || s.H != 5 {
		t.Errorf("plate is %dx%d, want 8x5", s.W, s.H)
	}
	if err := a.Err(); err != nil {
		t.Errorf("Err = %v after a load that worked", err)
	}
	// Cached like every other accessor, which matters because the shell asks for the
	// splash once a frame: an uncached UI would decode a 640x460 PNG thirty times a
	// second while the title screen sits there.
	if again := a.UI(1000); again != s {
		t.Error("UI decoded the plate twice; load's cache should have answered")
	}
	// An id next to a present one is still absent, i.e. the stat is per id and not a
	// test of whether the directory exists.
	if got := a.UI(1001); got != nil || a.Err() != nil {
		t.Errorf("UI(1001) beside a present 1000 gave %v / %v", got, a.Err())
	}
}

// Present but undecodable is a broken extraction rather than an absent one, and
// silence would hide it. This is the case that separates "no assets" from "bad
// assets", which are two different bug reports with two different fixes.
func TestUIBrokenPlateRecords(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "1000.png"), []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := NewAssets(dirFS(root))
	if got := a.UI(1000); got != nil {
		t.Error("UI returned a surface for a file that is not a PNG")
	}
	err := a.Err()
	if err == nil {
		t.Fatal("a plate that is present and will not decode must record; otherwise a\n" +
			"half-extracted tree is indistinguishable from an unextracted one")
	}
	// The message has to name the file, because the player's fix is to re-extract that
	// one plate and nothing in the shell will tell them which.
	if got := err.Error(); !strings.Contains(got, "1000.png") {
		t.Errorf("error is %q; it should name ui/1000.png", got)
	}
}

// The open house's fork is deliberately not consulted, unlike Pict and Plate. Thirteen of
// the twenty shipped houses carry their own 1991-1993, four their own 1017 or 1018 and
// Teddy World its own 1015 and 1016; every one of those is drawn *over a running game* and
// is asked for through Plate. The shell's own chrome is the application's, because a house
// that could repaint the title screen could hide the way out of it.
func TestUIIgnoresTheOpenHouseFork(t *testing.T) {
	// 2000 because it is an id both accessors can resolve: Pict finds it as a room
	// background under bg/, UI as a plate under ui/. The two directories are how the
	// extractor lays the tree out, and using one id through both is what makes the
	// difference below about the *fork* rather than about the file name.
	app := t.TempDir()
	writePlate(t, filepath.Join(app, "bg"), 2000, 8, 5)
	writePlate(t, filepath.Join(app, "ui"), 2000, 8, 5)

	fork := t.TempDir()
	writePlate(t, filepath.Join(fork, "pict"), 2000, 40, 40)

	a := NewAssets(dirFS(app))
	a.OpenHouseResFork(fork, dirFS(fork))

	// Pict honours the fork: this is the control, and without it the test below would
	// pass just as well on a broken fixture.
	p := a.Pict(2000)
	if p == nil {
		t.Fatalf("Pict(2000) is nil: %v", a.Err())
	}
	if p.W != 40 || p.H != 40 {
		t.Errorf("Pict returned the %dx%d picture; the open fork's 40x40 should win", p.W, p.H)
	}

	// UI does not, for the same id, from the same loader, with the same fork open.
	// The house's copy is under pict/ rather than ui/, so "ignores it" here means UI
	// looks only at the application root -- which is the property, not the layout.
	writePlate(t, filepath.Join(fork, "ui"), 2000, 40, 40)
	u := a.UI(2000)
	if u == nil {
		t.Fatalf("UI(2000) is nil with the application plate present: %v", a.Err())
	}
	if u.W != 8 || u.H != 5 {
		t.Errorf("UI returned the %dx%d plate; it must be the application's 8x5 one and "+
			"not the open house's", u.W, u.H)
	}
	if err := a.Err(); err != nil {
		t.Errorf("Err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// Plate
// ---------------------------------------------------------------------------

// Plate is the third resolution order in this file and it exists because the other two are
// each wrong for the pictures drawn *over a running game*: the pause placards, the
// stars-remaining panels and the banner plates.
//
// UI is wrong because on a Mac the open house's fork really does sit in front of the
// application's for these ids, and the shipped houses use it -- Teddy World has its own 1015
// and 1016. Pict is wrong because a missing file there is a fault worth recording, and these
// have a drawn fallback (internal/game/pause.go).
//
// So this pins both halves at once: the fork wins, and an absent plate is silent.
func TestPlatePrefersTheOpenHouseAndStaysSilent(t *testing.T) {
	app := t.TempDir()
	writePlate(t, filepath.Join(app, "ui"), 1015, 8, 5)

	fork := t.TempDir()
	writePlate(t, filepath.Join(fork, "pict"), 1015, 40, 40)

	a := NewAssets(dirFS(app))

	// Closed: the application's, which is what a house with no 1015 of its own gets and
	// what nineteen of the twenty shipped houses get.
	s := a.Plate(1015)
	if s == nil {
		t.Fatalf("Plate(1015) is nil with the application plate present: %v", a.Err())
	}
	if s.W != 8 || s.H != 5 {
		t.Errorf("with no fork open Plate returned %dx%d, want the application's 8x5", s.W, s.H)
	}

	// Open: the house's, which is Teddy World's case.
	a.OpenHouseResFork(fork, dirFS(fork))
	if s := a.Plate(1015); s == nil {
		t.Fatalf("Plate(1015) is nil with the fork open: %v", a.Err())
	} else if s.W != 40 || s.H != 40 {
		t.Errorf("with the fork open Plate returned %dx%d, want the house's 40x40", s.W, s.H)
	}

	// And UI, for the same id with the same fork open, still answers the application's --
	// which is the distinction the two accessors exist to draw.
	if u := a.UI(1015); u == nil || u.W != 8 {
		t.Errorf("UI(1015) = %v with the fork open; the shell's chrome is not a house's to "+
			"redefine", u)
	}

	// Absent from both is nil and no recorded error: the game must still be playable
	// against a checkout with no extracted art (docs/IMPROVEMENTS.md 2.6).
	a.CloseHouseResFork()
	if got := a.Plate(1016); got != nil {
		t.Errorf("Plate(1016) returned a %dx%d surface from a tree without one", got.W, got.H)
	}
	if err := a.Err(); err != nil {
		t.Errorf("a missing plate recorded a sticky error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// MaskedPlate
// ---------------------------------------------------------------------------

// writePlateFunc is writePlate with the caller choosing the pixels, which the mask tests
// need: a mask is only a mask if its two colours are where the test put them.
func writePlateFunc(t *testing.T, dir string, id int16, w, h int, at func(x, y int) uint8) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := Palette[at(x, y)]
			img.Set(x, y, color.RGBA{c.R, c.G, c.B, 0xFF})
		}
	}
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%d.png", id)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// maskedCount is how many pixels of a surface are opaque, which is the whole of what
// MaskedPlate produces that Plate does not.
func maskedCount(s *Surface) int {
	n := 0
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			if s.opaque(x, y) {
				n++
			}
		}
	}
	return n
}

// The basic contract: the art's pixels, the companion plate's non-white pixels as the
// mask, and the art left alone.
func TestMaskedPlateAppliesTheCompanionPlate(t *testing.T) {
	app := t.TempDir()
	ui := filepath.Join(app, "ui")
	writePlateFunc(t, ui, 1994, 8, 4, func(int, int) uint8 { return Yellow })
	// Left half white (clear), right half index 1, which is what the extractor writes
	// for a set bit in a 1-bit mask PICT.
	writePlateFunc(t, ui, 1998, 8, 4, func(x, y int) uint8 {
		if x < 4 {
			return White8
		}
		return 1
	})

	a := NewAssets(dirFS(app))
	s := a.MaskedPlate(1994)
	if s == nil {
		t.Fatalf("MaskedPlate(1994) is nil with both plates present: %v", a.Err())
	}
	if s.W != 8 || s.H != 4 {
		t.Fatalf("masked plate is %dx%d, want the art's 8x4", s.W, s.H)
	}
	for i, v := range s.Pix {
		if v != Yellow {
			t.Fatalf("pixel %d is %d, want the art's %d: the mask must not touch the colour plane", i, v, Yellow)
		}
	}
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			if got, want := s.opaque(x, y), x >= 4; got != want {
				t.Fatalf("opaque(%d,%d) = %v, want %v: a set mask bit is a non-white pixel", x, y, got, want)
			}
		}
	}

	// The art itself must come back unmasked from Plate, or the cache is aliased and the
	// second caller of Plate(1994) gets a surface with holes in it.
	if art := a.Plate(1994); maskedCount(art) != art.W*art.H {
		t.Errorf("Plate(1994) is %d/%d opaque; MaskedPlate mutated the cached art",
			maskedCount(art), art.W*art.H)
	}
	// Cached, because the game-over pages are drawn once a frame while they are up.
	if again := a.MaskedPlate(1994); again != s {
		t.Error("MaskedPlate rebuilt the pair on the second call")
	}
}

// An id with no companion is Plate exactly, so a call site does not need to know which of
// the plates it draws happens to have a mask.
func TestMaskedPlateIsPlateForAnUnpairedID(t *testing.T) {
	app := t.TempDir()
	writePlate(t, filepath.Join(app, "ui"), 1015, 8, 5)

	a := NewAssets(dirFS(app))
	if got, want := a.MaskedPlate(1015), a.Plate(1015); got != want {
		t.Errorf("MaskedPlate(1015) = %v, want the identical surface Plate returns %v", got, want)
	}
	if got := a.MaskedPlate(1016); got != nil || a.Err() != nil {
		t.Errorf("MaskedPlate(1016) on a tree without it gave %v / %v", got, a.Err())
	}
}

// A missing mask draws the art opaque rather than nothing. A partial extraction is a
// degraded picture; it is not a missing screen (docs/IMPROVEMENTS.md 2.6).
func TestMaskedPlateWithoutItsMaskIsTheOpaqueArt(t *testing.T) {
	app := t.TempDir()
	writePlate(t, filepath.Join(app, "ui"), 1994, 8, 4)

	a := NewAssets(dirFS(app))
	s := a.MaskedPlate(1994)
	if s == nil {
		t.Fatalf("MaskedPlate(1994) is nil with the art present and no mask: %v", a.Err())
	}
	if n := maskedCount(s); n != 8*4 {
		t.Errorf("%d of 32 pixels opaque with no mask plate, want all of them", n)
	}
	if err := a.Err(); err != nil {
		t.Errorf("a missing mask recorded a sticky error: %v", err)
	}
}

// A mask smaller than its art leaves the uncovered pixels transparent, which is what
// CopyMask reads out of the unwritten part of a fresh 1-bit GWorld. The alternative --
// treating "no mask bit" as opaque -- would draw the whole plate on a checkout whose mask
// PICT decoded short, and that is precisely the case the polarity has to survive.
func TestMaskedPlateTreatsAShortMaskAsTransparent(t *testing.T) {
	app := t.TempDir()
	ui := filepath.Join(app, "ui")
	writePlateFunc(t, ui, 1992, 8, 4, func(int, int) uint8 { return Yellow })
	writePlateFunc(t, ui, 1991, 8, 2, func(int, int) uint8 { return 1 })

	a := NewAssets(dirFS(app))
	s := a.MaskedPlate(1992)
	if s == nil {
		t.Fatalf("MaskedPlate(1992) is nil: %v", a.Err())
	}
	if n, want := maskedCount(s), 8*2; n != want {
		t.Errorf("%d pixels opaque, want %d: only the rows the mask covers", n, want)
	}
}

// The reason this lives in render at all rather than in the extractor: the art and the mask
// are separate resources with separate ids, so a house may override one and not the other.
// Thirteen of the twenty shipped houses carry their own 1991-1993, and on a Mac such a
// house's 1992 pairs with the *application's* 1991.
func TestMaskedPlateResolvesEachIDThroughTheForkSeparately(t *testing.T) {
	app := t.TempDir()
	ui := filepath.Join(app, "ui")
	writePlateFunc(t, ui, 1992, 8, 4, func(int, int) uint8 { return Yellow })
	writePlateFunc(t, ui, 1991, 8, 4, func(int, int) uint8 { return 1 }) // application mask: all opaque

	// The house redefines the art and *not* the mask, which is the shipped case.
	fork := t.TempDir()
	writePlateFunc(t, filepath.Join(fork, "pict"), 1992, 8, 4, func(int, int) uint8 { return QDCyan })

	a := NewAssets(dirFS(app))
	a.MaskedPlate(1992) // fork closed first, so a leaking cache would be caught below
	a.OpenHouseResFork(fork, dirFS(fork))

	s := a.MaskedPlate(1992)
	if s == nil {
		t.Fatalf("MaskedPlate(1992) is nil with the fork open: %v", a.Err())
	}
	if s.Pix[0] != QDCyan {
		t.Errorf("pixel 0 is %d, want the house's %d: the art did not come from the fork", s.Pix[0], QDCyan)
	}
	if n := maskedCount(s); n != 8*4 {
		t.Errorf("%d of 32 pixels opaque, want all: the application's mask should still apply", n)
	}

	// And the other way round: the house redefines only the mask, so the application's art
	// is drawn through it. This is the pairing an extractor that baked the two together
	// could not produce.
	fork2 := t.TempDir()
	writePlateFunc(t, filepath.Join(fork2, "pict"), 1991, 8, 4, func(x, y int) uint8 {
		if y == 0 {
			return 1
		}
		return White8
	})
	a.OpenHouseResFork(fork2, dirFS(fork2))
	s2 := a.MaskedPlate(1992)
	if s2 == nil {
		t.Fatalf("MaskedPlate(1992) is nil with the second fork open: %v", a.Err())
	}
	if s2.Pix[0] != Yellow {
		t.Errorf("pixel 0 is %d, want the application's %d", s2.Pix[0], Yellow)
	}
	if n, want := maskedCount(s2), 8; n != want {
		t.Errorf("%d pixels opaque, want %d: the house's mask should be the one applied", n, want)
	}
}

// The measured pairs, over the shipped art. This is the evidence behind uiMaskPairs and it
// is a test rather than a comment because the numbers are the argument: two of the three
// masks carry information the colour plane does not, so "key out white instead" is wrong in
// general -- and for the high-score plaque the two happen to agree exactly, which is worth
// knowing when a screenshot of it is compared against a colour-keyed one.
func TestTheThreeShippedMaskPairsAreMeasured(t *testing.T) {
	a := NewAssets(dirFS(requireAssets(t, "art")))

	for _, c := range []struct {
		art, mask                       int16
		w, h                            int
		opaque, maskOpaqueButArtIsWhite int
	}{
		{1990, 1989, 32, 448, 9356, 4475},
		{1992, 1991, 330, 30, 9340, 5400},
		{1994, 1998, 332, 30, 3409, 0},
	} {
		s := a.MaskedPlate(c.art)
		if s == nil {
			t.Fatalf("MaskedPlate(%d) is nil: %v", c.art, a.Err())
		}
		if s.W != c.w || s.H != c.h {
			t.Errorf("PICT %d is %dx%d, want %dx%d", c.art, s.W, s.H, c.w, c.h)
		}
		if n := maskedCount(s); n != c.opaque {
			t.Errorf("PICT %d/%d: %d opaque pixels, want %d", c.art, c.mask, n, c.opaque)
		}

		// The cross-tab against a white colour key, which is the substitute a port is
		// tempted by: how many pixels the mask keeps that keying on white would drop.
		holes := 0
		for y := 0; y < s.H; y++ {
			for x := 0; x < s.W; x++ {
				if s.opaque(x, y) && s.Pix[y*s.W+x] == White8 {
					holes++
				}
			}
		}
		if holes != c.maskOpaqueButArtIsWhite {
			t.Errorf("PICT %d/%d: %d pixels are white and mask-opaque, want %d",
				c.art, c.mask, holes, c.maskOpaqueButArtIsWhite)
		}
	}
}

// A house whose plate will not decode records, because that is a broken extraction rather
// than an absent one -- the same line Pict and UI draw. The fallback panel is for art that
// is not there, not for art that is there and wrong.
func TestPlateBrokenHousePlateRecords(t *testing.T) {
	fork := t.TempDir()
	dir := filepath.Join(fork, "pict")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "1015.png"), []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := NewAssets(dirFS(t.TempDir()))
	a.OpenHouseResFork(fork, dirFS(fork))

	if got := a.Plate(1015); got != nil {
		t.Error("Plate returned a surface for a file that is not a PNG")
	}
	err := a.Err()
	if err == nil {
		t.Fatal("a house plate that is present and will not decode must record")
	}
	if got := err.Error(); !strings.Contains(got, "1015.png") {
		t.Errorf("error is %q; it should name the file", got)
	}
}
