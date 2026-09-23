package platform

import (
	"bytes"
	"math/rand/v2"
	"reflect"
	"testing"
)

// frame is a w x h framebuffer, every byte of which says where it is, so no two pixels match.
func frame(w, h int) *Framebuffer {
	fb := NewFramebuffer(w, h)
	for i := range fb.Pix {
		fb.Pix[i] = byte(i*7 + i/251)
	}
	return fb
}

func TestChangesReportsWhatAFrameChanged(t *testing.T) {
	var c Changes
	fb := frame(64, 48)
	whole := []Span{{0, 0, 64, 48}}
	if got := c.Diff(fb); !reflect.DeepEqual(got, whole) {
		t.Fatalf("first frame: %v, want the whole frame", got)
	}
	if got := c.Diff(fb); len(got) != 0 {
		t.Fatalf("the same frame again: %v, want nothing", got)
	}

	// One pixel, then a block, then two blocks with a gap between them.
	fb.Pix[(10*64+5)*4] ^= 0xFF
	if got, want := c.Diff(fb), []Span{{5, 10, 6, 11}}; !reflect.DeepEqual(got, want) {
		t.Errorf("one pixel: %v, want %v", got, want)
	}
	for y := 20; y < 24; y++ {
		fb.Pix[(y*64+30)*4+1] ^= 0xFF
		fb.Pix[(y*64+(y-10))*4+2] ^= 0xFF // a different column on each row
	}
	if got, want := c.Diff(fb), []Span{{10, 20, 31, 24}}; !reflect.DeepEqual(got, want) {
		t.Errorf("a block of rows: %v, want their rows, and the columns any of them changed: %v", got, want)
	}
	fb.Pix[(2*64+63)*4+3] ^= 0xFF // the X of BGRX counts too: it is sent like the rest
	fb.Pix[(40*64)*4] ^= 0xFF
	if got, want := c.Diff(fb), []Span{{63, 2, 64, 3}, {0, 40, 1, 41}}; !reflect.DeepEqual(got, want) {
		t.Errorf("two blocks: %v, want %v", got, want)
	}

	c.All()
	if got := c.Diff(fb); !reflect.DeepEqual(got, whole) {
		t.Errorf("after All: %v, want the whole frame", got)
	}
	if got := c.Diff(frame(32, 48)); !reflect.DeepEqual(got, []Span{{0, 0, 32, 48}}) {
		t.Errorf("a frame of another size: %v, want the whole of it", got)
	}
}

func TestChangesSendsAScatteredFrameAsOneBlock(t *testing.T) {
	var c Changes
	fb := frame(64, 200)
	c.Diff(fb)
	for y := 0; y < 200; y += 2 {
		fb.Pix[(y*64+y%64)*4] ^= 0xFF // columns 0 to 62, since y is even
	}
	got := c.Diff(fb)
	if want := []Span{{0, 0, 63, 199}}; !reflect.DeepEqual(got, want) {
		t.Errorf("100 separate rows: %v, want one block over them: %v", got, want)
	}
}

func TestChangesReadsAFrameWithPadding(t *testing.T) {
	var c Changes
	fb := &Framebuffer{Pix: make([]byte, 20*10), Stride: 20, W: 4, H: 10}
	c.Diff(fb)
	fb.Pix[5*20+16] = 1 // padding, past the last pixel: nothing the window is sent
	if got := c.Diff(fb); len(got) != 0 {
		t.Errorf("a change in the padding: %v, want nothing", got)
	}
	fb.Pix[5*20+12] = 1
	if got, want := c.Diff(fb), []Span{{3, 5, 4, 6}}; !reflect.DeepEqual(got, want) {
		t.Errorf("%v, want %v", got, want)
	}
}

// A frame Diff cannot read is not its problem to report: it says the whole frame changed, and
// ExpandSpan refuses it with the reason.
func TestChangesDoesNotReadPastAShortFrame(t *testing.T) {
	var c Changes
	c.Diff(frame(16, 12))
	short := &Framebuffer{Pix: make([]byte, 16*4*11), Stride: 16 * 4, W: 16, H: 12}
	if got := c.Diff(short); !reflect.DeepEqual(got, []Span{{0, 0, 16, 12}}) {
		t.Fatalf("a short frame: %v, want the whole frame", got)
	}
	if err := ExpandSpan(make([]byte, 16*4*12), 16*4, short, 1, Span{0, 0, 16, 12}); err == nil {
		t.Error("ExpandSpan took a frame too short for its size")
	}
	if got := c.Diff(frame(16, 12)); !reflect.DeepEqual(got, []Span{{0, 0, 16, 12}}) {
		t.Errorf("the frame after it: %v, want the whole frame, since the short one was never kept", got)
	}
}

// The property a backend relies on: a window that was sent the whole first frame, and then only
// what Diff reported of each frame after it, shows exactly what Expand of that frame would.
func TestSendingOnlyTheChangesShowsTheWholeFrame(t *testing.T) {
	rng := rand.New(rand.NewPCG(2, 76))
	for scale := 1; scale <= 8; scale++ {
		w, h := 40, 30
		fb := frame(w, h)
		stride := w*scale*4 + 8 // padded, as a backend's surface may be
		shown := make([]byte, stride*h*scale)
		want := make([]byte, len(shown))
		var c Changes
		for n := range 60 {
			for range rng.IntN(6) {
				// A block of a few rows and columns, as a sprite moving would change.
				x, y := rng.IntN(w), rng.IntN(h)
				for dy := range 1 + rng.IntN(4) {
					for dx := range 1 + rng.IntN(5) {
						if x+dx < w && y+dy < h {
							fb.Pix[((y+dy)*w+x+dx)*4+rng.IntN(4)] = byte(rng.Uint32())
						}
					}
				}
			}
			for _, s := range c.Diff(fb) {
				if err := ExpandSpan(shown, stride, fb, scale, s); err != nil {
					t.Fatal(err)
				}
			}
			if err := Expand(want, stride, fb, scale); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(shown, want) {
				t.Fatalf("scale %d, frame %d: the window differs from the frame", scale, n)
			}
		}
	}
}

func TestExpandSpanWritesOnlyItsBlock(t *testing.T) {
	fb := frame(8, 6)
	const scale = 3
	stride := 8 * scale * 4
	dst := bytes.Repeat([]byte{0xEE}, stride*6*scale)
	if err := ExpandSpan(dst, stride, fb, scale, Span{2, 1, 5, 3}); err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 6*scale; y++ {
		for x := 0; x < 8*scale; x++ {
			got := dst[y*stride+x*4 : y*stride+x*4+4]
			sx, sy := x/scale, y/scale
			inside := sx >= 2 && sx < 5 && sy >= 1 && sy < 3
			want := []byte{0xEE, 0xEE, 0xEE, 0xEE}
			if inside {
				want = fb.Pix[(sy*8+sx)*4:][:4]
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("(%d,%d) is %x, want %x (inside the span: %v)", x, y, got, want, inside)
			}
		}
	}

	// A span off the edge is clipped rather than refused, and one wholly off it writes nothing.
	if err := ExpandSpan(dst, stride, fb, scale, Span{-4, -4, 100, 100}); err != nil {
		t.Errorf("a span larger than the frame: %v", err)
	}
	before := append([]byte(nil), dst...)
	if err := ExpandSpan(dst, stride, fb, scale, Span{8, 0, 12, 6}); err != nil || !bytes.Equal(dst, before) {
		t.Errorf("a span past the frame's right edge wrote something (err %v)", err)
	}
}

func TestFitIsTheLargestScaleThatFits(t *testing.T) {
	for _, tc := range []struct {
		name string
		room Room
		max  int
		want int
	}{
		// This host's desktop: a 2526x1312 work area, less the fixed title-bar allowance.
		{"the development desktop", Room{W: 2526 - 16, H: 1312 - 56}, 8, 2},
		{"1080p under a top bar", Room{W: 1920 - 16, H: 1080 - 32 - 56}, 8, 2},
		{"2560x1440 does not reach 3x", Room{W: 2560, H: 1440 - 48}, 8, 2},
		{"2560x1600 does", Room{W: 2560, H: 1600 - 48}, 8, 3},
		{"4K", Room{W: 3840, H: 2160 - 48}, 8, 4},
		{"4K, capped at 3", Room{W: 3840, H: 2160 - 48}, 3, 3},
		{"exactly 2x", Room{W: 1280, H: 960}, 8, 2},
		{"a pixel short of 2x", Room{W: 1280, H: 959}, 8, 1},
		{"1366x768", Room{W: 1366, H: 768 - 40}, 8, 1},
		{"smaller than the game", Room{W: 320, H: 200}, 8, 1},
		{"no room at all", Room{}, 8, 1},
		{"wider than 8x", Room{W: 1 << 20, H: 1 << 20}, 8, 8},
	} {
		if got := tc.room.Fit(ScreenWidth, ScreenHeight, tc.max); got != tc.want {
			t.Errorf("%s: %dx%d, max %d: Fit = %d, want %d", tc.name, tc.room.W, tc.room.H, tc.max, got, tc.want)
		}
	}
}
