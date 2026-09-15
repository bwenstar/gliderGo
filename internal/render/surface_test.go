package render

import (
	"image/color"
	"testing"
)

// TestBGRXLUTMatchesPalette is a package-initialisation regression test, and it exists
// because this table was silently all-black.
//
// bgrxLUT began life as `var bgrxLUT = func() [256]uint32 { ... range Palette ... }()`.
// Go runs every package-level variable initializer before any init function, ordering them
// by the dependencies visible in the initializer expressions -- and Palette has no
// initializer expression, because palette.go's init fills it. So there was no dependency
// edge, the closure ran first, and it built 256 entries out of a zero-valued Palette:
// every one 0xFF000000, i.e. opaque black.
//
// The consequence was that Surface.ToBGRX -- the one function that turns a composed screen
// into something a window can show -- returned an all-black image for every possible input.
// Nothing in this package calls it, so no test here failed; the game's own tests assert on
// palette indices, so none of those failed either. It was found by looking at the first
// frame the port ever put in a window.
//
// The general shape of the bug is "a derived table built from a table filled by init", and
// the general fix is "derive it in the same init". This test is the specific guard: it
// compares the LUT against Palette element by element, which is a claim no initialisation
// order can satisfy accidentally.
func TestBGRXLUTMatchesPalette(t *testing.T) {
	var nonBlack int
	for i, c := range Palette {
		want := uint32(c.B) | uint32(c.G)<<8 | uint32(c.R)<<16 | 0xFF000000
		if bgrxLUT[i] != want {
			t.Fatalf("bgrxLUT[%d] = %#08x, want %#08x (palette %+v)", i, bgrxLUT[i], want, c)
		}
		if want != 0xFF000000 {
			nonBlack++
		}
	}
	// 255 of the 256 entries are non-black -- index 255 is the palette's only pure black.
	// Asserting the count as well as the values is what makes this test fail loudly rather
	// than pass vacuously if Palette itself is ever left empty.
	if nonBlack != 255 {
		t.Errorf("%d non-black entries, want 255; the palette is not populated", nonBlack)
	}
}

// TestToBGRXExpandsEveryPixel pins the conversion itself: the destination's byte order, its
// alpha, and its tolerance of a stride wider than the surface.
func TestToBGRXExpandsEveryPixel(t *testing.T) {
	s := NewSurface(3, 2)
	s.Pix[0] = 0   // white: the palette's first entry
	s.Pix[1] = 255 // black: its last
	s.Pix[2] = 17
	s.Pix[3] = 200
	s.Pix[4] = 99
	s.Pix[5] = 1

	// A stride two pixels wider than the surface, pre-filled with a marker, so that the
	// promise in ToBGRX's doc comment -- pixels beyond the width are left alone -- is
	// tested rather than assumed. A backend whose rows are padded depends on it.
	const stride = 5 * 4
	dst := make([]byte, stride*2)
	for i := range dst {
		dst[i] = 0xAA
	}
	s.ToBGRX(dst, stride)

	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			c := Palette[s.Pix[y*3+x]]
			o := y*stride + x*4
			got := color.RGBA{B: dst[o], G: dst[o+1], R: dst[o+2], A: dst[o+3]}
			if got != (color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xFF}) {
				t.Errorf("(%d,%d) index %d -> %+v, want %+v", x, y, s.Pix[y*3+x], got, c)
			}
		}
		for o := y*stride + 3*4; o < y*stride+stride; o++ {
			if dst[o] != 0xAA {
				t.Errorf("byte %d of row %d was overwritten; padding must be left alone", o, y)
			}
		}
	}
}
