package platform

import (
	"image/color"
	"testing"
)

func TestFramebufferFillAndSet(t *testing.T) {
	fb := NewFramebuffer(8, 4)
	if got, want := len(fb.Pix), 8*4*4; got != want {
		t.Fatalf("Pix length = %d, want %d", got, want)
	}

	fb.Fill(color.RGBA{R: 1, G: 2, B: 3, A: 255})
	for y := 0; y < fb.H; y++ {
		for x := 0; x < fb.W; x++ {
			o := y*fb.Stride + x*4
			if fb.Pix[o] != 3 || fb.Pix[o+1] != 2 || fb.Pix[o+2] != 1 || fb.Pix[o+3] != 255 {
				t.Fatalf("Fill left BGRX %v at (%d,%d), want {3 2 1 255}", fb.Pix[o:o+4], x, y)
			}
		}
	}

	fb.Set(3, 2, color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 255})
	o := 2*fb.Stride + 3*4
	if fb.Pix[o] != 0x30 || fb.Pix[o+1] != 0x20 || fb.Pix[o+2] != 0x10 {
		t.Errorf("Set wrote BGRX %v, want {0x30 0x20 0x10 ...}", fb.Pix[o:o+4])
	}
}

// Out-of-range writes must be dropped, not panic: the original draws sprites
// that hang off the edge of a room and relies on the clip.
func TestFramebufferSetClipsInsteadOfPanicking(t *testing.T) {
	fb := NewFramebuffer(4, 4)
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {4, 0}, {0, 4}, {100, 100}} {
		fb.Set(p[0], p[1], color.RGBA{R: 255, A: 255})
	}
	for i, b := range fb.Pix {
		if b != 0 {
			t.Fatalf("out-of-range Set touched byte %d (value %d)", i, b)
		}
	}
}
