package render

// Hostile bytes, on purpose (docs/IMPROVEMENTS.md 4.30).
//
// A house can carry its own pictures, and -levels, -houseart and -art all read PNGs somebody else
// drew. image/png is the standard library's and is fuzzed there. What is this package's is
// everything around it: the size check from the header, the second open, and the two converters
// that turn whatever image type the decoder chose into a Surface. So FuzzPicture puts arbitrary
// bytes through both loaders. Plain `go test` runs its seeds; `make fuzz` runs the engine.

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"testing/fstest"
)

// FuzzPicture is a picture file from anywhere, loaded as art and as a house's picture.
//
// What must hold:
//   - a loader returns a surface or records an error, never both and never neither;
//   - a surface is the size the file's header says, and inside the limits (4.36);
//   - its planes are exactly that size: an art surface has a mask, a house picture is opaque and
//     has none;
//   - a house picture's pixel whose colour is on the palette gets that entry, not an approximation.
func FuzzPicture(f *testing.F) {
	f.Add(pngHeader(1, 1))
	f.Add(pngHeader(maxPictSide, 1))
	f.Add(pngHeader(16384, 16384))
	white := color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}
	for _, img := range []image.Image{
		fill(image.NewNRGBA(image.Rect(0, 0, 2, 2)), white),
		fill(image.NewNRGBA(image.Rect(0, 0, 3, 1)), color.NRGBA{}),              // transparent
		fill(image.NewNRGBA(image.Rect(0, 0, 2, 1)), color.NRGBA{1, 2, 3, 0xFF}), // off the palette
		fill(image.NewNRGBA(image.Rect(0, 0, 1, 1)), color.NRGBA{9, 9, 9, 0x80}), // partial alpha
		fill(image.NewGray(image.Rect(0, 0, 2, 2)), color.Gray{0x80}),
		fill(image.NewRGBA64(image.Rect(0, 0, 1, 2)), color.RGBA64{0xFFFF, 0, 0, 0xFFFF}),
		image.NewPaletted(image.Rect(0, 0, 3, 2), color.Palette{Palette[0], Palette[35], Palette[255]}),
	} {
		var b bytes.Buffer
		if err := png.Encode(&b, img); err != nil {
			f.Fatal(err)
		}
		f.Add(b.Bytes())
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		fsys := fstest.MapFS{"p.png": &fstest.MapFile{Data: b}}
		for _, house := range []bool{false, true} {
			a := NewAssets(fsys)
			var s *Surface
			if house {
				s = a.loadHousePict(fsys, "fuzz", "p.png")
			} else {
				s = a.load("p.png")
			}
			err := a.Err()
			if (s == nil) == (err == nil) {
				t.Fatalf("house=%v: a surface %v and an error %v", house, s != nil, err)
			}
			if s == nil {
				continue
			}
			cfg, cerr := png.DecodeConfig(bytes.NewReader(b))
			switch {
			case cerr != nil:
				t.Fatalf("house=%v: a surface from a file whose header does not decode: %v", house, cerr)
			case s.W != cfg.Width || s.H != cfg.Height:
				t.Fatalf("house=%v: %dx%d from a header of %dx%d", house, s.W, s.H, cfg.Width, cfg.Height)
			case s.W > maxPictSide || s.H > maxPictSide || s.W*s.H > maxPictPixels:
				t.Fatalf("house=%v: %dx%d got past the limits", house, s.W, s.H)
			case len(s.Pix) != s.W*s.H:
				t.Fatalf("house=%v: %d pixels for %dx%d", house, len(s.Pix), s.W, s.H)
			case !house && len(s.Mask) != s.W*s.H:
				t.Fatalf("an art surface with a %d-byte mask for %dx%d", len(s.Mask), s.W, s.H)
			case house && s.Mask != nil:
				t.Fatal("a house picture with a mask; they are opaque")
			}
			if !house {
				continue
			}
			img, err := png.Decode(bytes.NewReader(b))
			if err != nil {
				t.Fatalf("a house picture loaded from a file that does not decode: %v", err)
			}
			for y := range s.H {
				for x := range s.W {
					r, g, bl, _ := img.At(x, y).RGBA()
					if idx, ok := IndexOf(uint8(r>>8), uint8(g>>8), uint8(bl>>8)); ok && s.Pix[y*s.W+x] != idx {
						t.Fatalf("(%d,%d) is palette entry %d and became %d", x, y, idx, s.Pix[y*s.W+x])
					}
				}
			}
		}
	})
}

// fill paints every pixel of img one colour and returns it.
func fill[T interface {
	image.Image
	Set(x, y int, c color.Color)
}](img T, c color.Color) image.Image {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}
