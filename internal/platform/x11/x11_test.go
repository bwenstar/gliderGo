//go:build linux && cgo

package x11

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bwenstar/gliderGo/internal/platform"
)

// TestMain keeps a connection to the server open while the tests open and close windows. A
// server with no other client resets when a test closes its window, and refuses the next test's
// connection while it does. Room's connection is the one that stays, and New closes it, so a test
// that opens a window asks for it again straight after (hold).
func TestMain(m *testing.M) {
	hold()
	os.Exit(m.Run())
}

func hold() {
	if os.Getenv("DISPLAY") != "" {
		Room()
	}
}

// TestNewAsksForDetectableAutoRepeat is what makes Event.Repeat mean anything on Linux. By
// default an X server sends a held key's repeats as release-and-press pairs, the bitmap sees the
// key go up before every one, and a held Escape on the house picker quits the game. New asks for
// XKB's detectable auto-repeat to stop that (docs/IMPROVEMENTS.md 2.72).
//
// What it cannot do is hold a key. That needs XTest to fake a press the server will repeat, and
// XTest is a library this port does not link, so the test checks the server's own answer after
// New instead, which is the one thing New changes.
func TestNewAsksForDetectableAutoRepeat(t *testing.T) {
	// DISPLAY is what decides, as it is for `make smoke`. Unset, there is no server to ask, which
	// is every CI runner and every SSH session without -X. Set, a window is expected, and one that
	// will not open is a failure rather than a skip.
	display := os.Getenv("DISPLAY")
	if display == "" {
		t.Skip("DISPLAY is unset, so there is no X server to ask")
	}
	w, err := New(platform.Config{Title: "gliderGo x11 test"})
	if err != nil {
		t.Fatalf("DISPLAY=%q is set and New failed: %v", display, err)
	}
	hold()
	defer w.Close()

	on, supported := w.detectableAutoRepeat()
	if !supported {
		t.Skipf("the server on DISPLAY=%q does not support detectable auto-repeat, so this cannot "+
			"tell whether New asked for it", display)
	}
	if !on {
		t.Error("the server supports detectable auto-repeat and it is off after New, so every " +
			"repeat of a held key will read as a fresh press")
	}
}

// TestPresentSendsWhatTheFrameChanged is the changed-rows present end to end: a whole frame,
// then a frame that differs in two places, read back from the server each time and held to what
// Expand of that frame says the window should show (docs/IMPROVEMENTS.md 2.76). Needs a display,
// as the test above does, and a window nothing covers, which a server with no window manager or
// one that composites both give.
func TestPresentSendsWhatTheFrameChanged(t *testing.T) {
	display := os.Getenv("DISPLAY")
	if display == "" {
		t.Skip("DISPLAY is unset, so there is no X server to draw on")
	}
	// XGetImage refuses, with a BadMatch that Xlib's default handler turns into exit(1), a
	// window that runs off the screen -- and CI's Xvfb has been 640x480. So the scale is the
	// largest up to 2 that fits, and a screen that fits no window at all is a skip.
	room, err := Room()
	if err != nil {
		t.Fatal(err)
	}
	scale := room.Fit(platform.ScreenWidth, platform.ScreenHeight, 2)
	if room.W < platform.ScreenWidth*scale || room.H < platform.ScreenHeight*scale {
		t.Skipf("%s is %dx%d, too small to read a %dx%d window back from",
			room.From, room.W, room.H, platform.ScreenWidth, platform.ScreenHeight)
	}
	t.Logf("at %dx, in %s (%dx%d)", scale, room.From, room.W, room.H)
	w, err := New(platform.Config{Title: "gliderGo x11 test", Scale: scale})
	if err != nil {
		t.Fatalf("DISPLAY=%q is set and New failed: %v", display, err)
	}
	hold()
	defer w.Close()
	for start := time.Now(); !w.mapped; time.Sleep(10 * time.Millisecond) {
		if time.Since(start) > 5*time.Second {
			t.Fatal("the window was not mapped within 5 s")
		}
		w.PollEvents()
	}

	fb := platform.NewFramebuffer(platform.ScreenWidth, platform.ScreenHeight)
	for i := range fb.Pix {
		fb.Pix[i] = byte(i*7 + i/251)
	}
	want := make([]byte, fb.W*scale*4*fb.H*scale)
	check := func(what string) {
		t.Helper()
		if err := w.Present(fb); err != nil {
			t.Fatal(err)
		}
		got, ok := w.shown()
		if !ok {
			t.Fatalf("%s: could not read the window back", what)
		}
		if err := platform.Expand(want, fb.W*scale*4, fb, scale); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < len(want); i += 4 {
			// The fourth byte is padding to a 24-bit server, which returns what it likes.
			if got[i] != want[i] || got[i+1] != want[i+1] || got[i+2] != want[i+2] {
				p := i / 4
				t.Fatalf("%s: pixel (%d,%d) is %x, want %x", what,
					p%(fb.W*scale), p/(fb.W*scale), got[i:i+3], want[i:i+3])
			}
		}
	}
	check("the first frame")
	for y := 100; y < 140; y++ {
		for x := 200; x < 260; x++ {
			fb.Pix[y*fb.Stride+x*4+1] ^= 0xFF
		}
	}
	fb.Pix[479*fb.Stride+639*4] ^= 0xFF
	check("a frame that changed a block and the last pixel")
	check("the same frame again")
}

func TestChooseRoom(t *testing.T) {
	// This host's desktop: one 2560x1344 monitor, a dock down the left and a top bar.
	gnome := []int{34, 32, 2526, 1312}
	two := []int{0, 27, 1920, 1053, 1920, 0, 2560, 1400} // a 1080p monitor and a 1440p one beside it
	for _, tc := range []struct {
		name           string
		monitors, work []int
		px, py         int
		wm             bool
		want           platform.Room
	}{
		{"mutter", gnome, []int{34, 32, 2526, 1312}, 900, 600, true,
			platform.Room{W: 2526 - frameW, H: 1312 - frameH,
				From: "the work area of the monitor under the pointer, less a title bar"}},
		{"the pointer on the top bar", gnome, nil, 900, 5, true,
			platform.Room{W: 2526 - frameW, H: 1312 - frameH,
				From: "the work area of the monitor under the pointer, less a title bar"}},
		{"the left of two monitors", two, []int{0, 0, 4480, 1400}, 100, 500, true,
			platform.Room{W: 1920 - frameW, H: 1053 - frameH,
				From: "the work area of the monitor under the pointer, less a title bar"}},
		{"the right of two monitors", two, []int{0, 0, 4480, 1400}, 3000, 500, true,
			platform.Room{W: 2560 - frameW, H: 1400 - frameH,
				From: "the work area of the monitor under the pointer, less a title bar"}},
		{"the pointer on another screen", two, nil, -1, -1, false,
			platform.Room{W: 1920, H: 1053, From: "the work area of the monitor under the pointer"}},
		{"EWMH without mutter", nil, []int{0, 24, 1920, 1056}, 10, 10, true,
			platform.Room{W: 1920 - frameW, H: 1056 - frameH, From: "the desktop's work area, less a title bar"}},
		{"a monitor list that is nonsense", []int{0, 0, 0, 0, 5}, []int{0, 24, 1920, 1056}, 10, 10, false,
			platform.Room{W: 1920, H: 1056, From: "the desktop's work area"}},
		{"Xvfb", nil, nil, 0, 0, false, platform.Room{W: 2600, H: 1980, From: "the screen"}},
	} {
		got := chooseRoom(2600, 1980, tc.monitors, tc.work, tc.px, tc.py, tc.wm)
		if got != tc.want {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

// TestRoom asks the display for real. Whatever the host, the answer has to be a size, and it has
// to say where it came from.
func TestRoom(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("DISPLAY is unset, so there is no X server to ask")
	}
	r, err := Room()
	if err != nil {
		t.Fatal(err)
	}
	if r.W <= 0 || r.H <= 0 || r.From == "" {
		t.Errorf("Room() = %+v", r)
	}
	t.Logf("DISPLAY=%s: %dx%d, from %s: the largest window that fits is %dx", os.Getenv("DISPLAY"),
		r.W, r.H, r.From, r.Fit(platform.ScreenWidth, platform.ScreenHeight, 8))
}

// BenchmarkPresent is what a frame costs to show, the server's share included, for the three
// kinds of frame a changed-rows present sees (docs/IMPROVEMENTS.md 2.76): one that changed
// nothing, a title screen or a pause; one that moved a sprite, which is most of a game; and one
// that changed everything, which is a shell screen appearing, an expose, or a room entered
// without a wipe. `make bench` is the second kind almost throughout -- its glider never leaves
// the first room -- so the third is here to be the worst case the bench row does not show.
//
//	DISPLAY=:57 go test -run - -bench Present ./internal/platform/x11/
func BenchmarkPresent(b *testing.B) {
	if os.Getenv("DISPLAY") == "" {
		b.Skip("DISPLAY is unset, so there is no X server to draw on")
	}
	room, err := Room()
	if err != nil {
		b.Fatal(err)
	}
	for scale := 1; scale <= 4; scale++ {
		if room.W < platform.ScreenWidth*scale || room.H < platform.ScreenHeight*scale {
			// Off the screen is never drawn, and would be measured as free.
			b.Logf("%dx does not fit %s (%dx%d); not measured", scale, room.From, room.W, room.H)
			continue
		}
		w, err := New(platform.Config{Title: "gliderGo x11 bench", Scale: scale})
		if err != nil {
			b.Fatal(err)
		}
		hold()
		for start := time.Now(); !w.mapped; time.Sleep(10 * time.Millisecond) {
			if time.Since(start) > 5*time.Second {
				b.Fatal("the window was not mapped within 5 s")
			}
			w.PollEvents()
		}
		fb := platform.NewFramebuffer(platform.ScreenWidth, platform.ScreenHeight)
		for i := range fb.Pix {
			fb.Pix[i] = byte(i*7 + i/251)
		}
		for _, kind := range []string{"unchanged", "a sprite", "the whole frame"} {
			b.Run(fmt.Sprintf("%s/%dx", kind, scale), func(b *testing.B) {
				w.changes.All()
				if err := w.Present(fb); err != nil {
					b.Fatal(err)
				}
				w.sync()
				b.ResetTimer()
				for n := range b.N {
					switch kind {
					case "a sprite":
						// A 32x32 block that moves a pixel a frame, as a glider does.
						x0, y0 := 100+n%200, 200
						for y := y0; y < y0+32; y++ {
							fb.Pix[(y*fb.W+x0)*4] ^= 0xFF
						}
					case "the whole frame":
						w.changes.All()
					}
					if err := w.Present(fb); err != nil {
						b.Fatal(err)
					}
					w.sync()
				}
			})
		}
		w.Close()
	}
}
