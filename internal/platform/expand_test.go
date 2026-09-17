package platform

import (
	"bytes"
	"image/color"
	"testing"
)

// bgrx is one pixel as Expand and every backend see it.
func bgrx(c color.RGBA) []byte { return []byte{c.B, c.G, c.R, 0xff} }

// gradient is a framebuffer whose every pixel is distinguishable from every other, which is what
// makes a misplaced block visible instead of merely possible.
func gradient(w, h, pad int) *Framebuffer {
	fb := &Framebuffer{Stride: w*4 + pad, W: w, H: h}
	fb.Pix = make([]byte, fb.Stride*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			o := y*fb.Stride + x*4
			fb.Pix[o+0] = byte(x)
			fb.Pix[o+1] = byte(y)
			fb.Pix[o+2] = byte(x ^ y)
			fb.Pix[o+3] = 0xff
		}
	}
	// A recognisable value in the padding, so a scale-1 copy that walks the source stride
	// instead of the source width shows up as a wrong pixel rather than as a black one.
	for y := 0; y < h; y++ {
		for i := 0; i < pad; i++ {
			fb.Pix[y*fb.Stride+w*4+i] = 0xEE
		}
	}
	return fb
}

// TestExpandPlacesEveryPixelAtEveryScale is the whole contract: destination pixel (x, y) is
// source pixel (x/scale, y/scale), for every pixel of every scale, with a padded source stride
// and a padded destination stride to prove neither is confused with a width.
func TestExpandPlacesEveryPixelAtEveryScale(t *testing.T) {
	const w, h = 7, 5
	for _, scale := range []int{1, 2, 3, 4} {
		for _, dstPad := range []int{0, 12} {
			fb := gradient(w, h, 8)
			dstStride := w*scale*4 + dstPad
			dst := bytes.Repeat([]byte{0xAA}, dstStride*h*scale)
			if err := Expand(dst, dstStride, fb, scale); err != nil {
				t.Fatalf("scale %d pad %d: %v", scale, dstPad, err)
			}
			for y := 0; y < h*scale; y++ {
				for x := 0; x < w*scale; x++ {
					so := (y/scale)*fb.Stride + (x/scale)*4
					do := y*dstStride + x*4
					if !bytes.Equal(dst[do:do+4], fb.Pix[so:so+4]) {
						t.Fatalf("scale %d pad %d: dst(%d,%d) = %v, want src(%d,%d) = %v",
							scale, dstPad, x, y, dst[do:do+4], x/scale, y/scale, fb.Pix[so:so+4])
					}
				}
			}
			// The destination padding must be untouched, or a backend with an aligned
			// surface would have this function writing over somebody else's bytes.
			for y := 0; y < h*scale; y++ {
				tail := dst[y*dstStride+w*scale*4 : (y+1)*dstStride]
				for i, b := range tail {
					if b != 0xAA {
						t.Fatalf("scale %d: destination padding byte %d of row %d was written (%#x)",
							scale, i, y, b)
					}
				}
			}
		}
	}
}

// TestExpandAgreesWithSetAndFill ties Expand to the two writers in this package, so the pixel
// order it copies is checked against the definition of the format rather than against itself.
func TestExpandAgreesWithSetAndFill(t *testing.T) {
	fb := NewFramebuffer(4, 3)
	red := color.RGBA{R: 0xC0, G: 0x10, B: 0x20, A: 0xff}
	fb.Fill(red)
	blue := color.RGBA{R: 0x01, G: 0x02, B: 0x03, A: 0xff}
	fb.Set(2, 1, blue)

	const scale = 2
	dstStride := fb.W * scale * 4
	dst := make([]byte, dstStride*fb.H*scale)
	if err := Expand(dst, dstStride, fb, scale); err != nil {
		t.Fatal(err)
	}
	// The one Set pixel became a 2x2 block of blue at (4..5, 2..3), and everything else is red.
	for y := 0; y < fb.H*scale; y++ {
		for x := 0; x < fb.W*scale; x++ {
			want := bgrx(red)
			if x/scale == 2 && y/scale == 1 {
				want = bgrx(blue)
			}
			o := y*dstStride + x*4
			if !bytes.Equal(dst[o:o+4], want) {
				t.Fatalf("(%d,%d) = %v, want %v", x, y, dst[o:o+4], want)
			}
		}
	}
}

// TestExpandRefusesASurfaceThatIsTooSmall covers the reason the checks are up front and return
// errors: a backend gets its geometry from a window manager and its buffer from its own
// bookkeeping, and the frame where those two disagree must end the window rather than the
// process. Each case would be an out-of-range write without the guard.
func TestExpandRefusesASurfaceThatIsTooSmall(t *testing.T) {
	ok := func(w, h int) *Framebuffer { return NewFramebuffer(w, h) }

	for _, tc := range []struct {
		name      string
		dst       int
		dstStride int
		fb        *Framebuffer
		scale     int
		wantErr   bool
	}{
		{"exactly right", 16 * 4 * 12, 16 * 4, ok(16, 12), 1, false},
		{"exactly right, doubled", 32 * 4 * 24, 32 * 4, ok(16, 12), 2, false},
		{"one row short", 16 * 4 * 11, 16 * 4, ok(16, 12), 1, true},
		{"one byte short", 16*4*12 - 1, 16 * 4, ok(16, 12), 1, true},
		{"stride too narrow", 16 * 4 * 12, 16*4 - 4, ok(16, 12), 1, true},
		{"stride fits 1:1 but not 2:1", 32 * 4 * 24, 16 * 4, ok(16, 12), 2, true},
		{"source stride lies about its width",
			16 * 4 * 12, 16 * 4, &Framebuffer{Pix: make([]byte, 16*4*12), Stride: 16*4 - 4, W: 16, H: 12}, 1, true},
		{"source shorter than its own geometry",
			16 * 4 * 12, 16 * 4, &Framebuffer{Pix: make([]byte, 16*4*11), Stride: 16 * 4, W: 16, H: 12}, 1, true},
		{"negative height", 16 * 4 * 12, 16 * 4, &Framebuffer{Stride: 64, W: 16, H: -1}, 1, true},

		// Zero pixels is not an error: it is a window that has nothing to show yet, and
		// returning early is what keeps the arithmetic below it free of empty-slice cases.
		{"no width", 0, 0, &Framebuffer{Stride: 0, W: 0, H: 12}, 1, false},
		{"no height", 0, 64, &Framebuffer{Stride: 64, W: 16, H: 0}, 1, false},

		// A scale below 1 is clamped rather than refused, because Config.Scale documents 0
		// as meaning 1:1 and a backend passes it through.
		{"scale zero means 1:1", 16 * 4 * 12, 16 * 4, ok(16, 12), 0, false},
		{"scale minus one means 1:1", 16 * 4 * 12, 16 * 4, ok(16, 12), -1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := Expand(make([]byte, tc.dst), tc.dstStride, tc.fb, tc.scale)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Expand = %v, want error: %v", err, tc.wantErr)
			}
		})
	}
}

// BenchmarkExpand is the frame budget for the two backends' upload path: a 640x480 frame at the
// three scales the shell offers. Measured on the development host: 64 us at 1:1, 2.2 ms at 2:1
// and 3.3 ms at 3:1, against a 16.7 ms frame. So a per-frame full-surface expansion is
// affordable at every scale, which is the question this answers -- and the jump from 1:1 to 2:1
// is 34x rather than 4x because 1:1 is one copy per row and anything above it is one per pixel.
// Worth knowing before optimising the wrong end: at 3:1 this is a fifth of the frame.
func BenchmarkExpand(b *testing.B) {
	for _, scale := range []int{1, 2, 3} {
		fb := NewFramebuffer(ScreenWidth, ScreenHeight)
		fb.Fill(color.RGBA{R: 0x40, G: 0x80, B: 0xC0, A: 0xff})
		dstStride := ScreenWidth * scale * 4
		dst := make([]byte, dstStride*ScreenHeight*scale)
		b.Run(map[int]string{1: "1x", 2: "2x", 3: "3x"}[scale], func(b *testing.B) {
			b.SetBytes(int64(len(dst)))
			for i := 0; i < b.N; i++ {
				if err := Expand(dst, dstStride, fb, scale); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
