// Command glidergo will be the game. Today it is the platform-layer smoke test:
// it opens a window at the original's exact 640x480, drives it at the original's
// tick rate, and draws a moving test pattern so that a new session can confirm
// in one command that the toolchain, cgo, X11 and the frame pacing all work.
//
//	go run ./cmd/glidergo                    # windowed, 60 ticks/s, Esc to quit
//	go run ./cmd/glidergo -scale 2           # 2x nearest-neighbour magnification
//	go run ./cmd/glidergo -frames 120 -bench # timed run, no interaction
//	go run -tags nullbackend ./cmd/glidergo -frames 5 -dump /tmp/frames
package main

import (
	"flag"
	"fmt"
	"image/color"
	"os"
	"time"

	"glidergo/internal/platform"
	"glidergo/internal/platform/backend"
)

// The original runs its world off the Mac tick timer at 60 ticks per second;
// docs/analysis/architecture.md records how many ticks it spends per frame.
// Until that is ported, 60 Hz is the placeholder the smoke test paces to.
const tickHz = 60

func main() {
	scale := flag.Int("scale", 1, "integer magnification of the 640x480 image")
	frames := flag.Int("frames", 0, "exit after N frames (0 = run until quit)")
	bench := flag.Bool("bench", false, "run flat out and report frame rate instead of pacing to the tick")
	dump := flag.String("dump", "", "with -tags nullbackend, write each frame as a PNG into this directory")
	flag.Parse()

	if *dump != "" {
		os.Setenv("GLIDERGO_FRAMEDUMP", *dump)
	}

	win, err := backend.Open(platform.Config{
		Title: "gliderGo",
		Scale: *scale,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "glidergo: %v\n", err)
		os.Exit(1)
	}
	defer win.Close()

	fb := platform.NewFramebuffer(platform.ScreenWidth, platform.ScreenHeight)
	fmt.Printf("glidergo: backend=%s surface=%dx%d scale=%d\n",
		backend.Name, fb.W, fb.H, *scale)

	tick := time.Duration(int64(time.Second) / tickHz)
	next := time.Now()
	start := time.Now()
	n := 0

	for {
		for _, ev := range win.PollEvents() {
			switch {
			case ev.Kind == platform.EventQuit,
				ev.Kind == platform.EventKeyDown && ev.Key == platform.KeyEscape,
				ev.Kind == platform.EventKeyDown && ev.Key == platform.KeyQ:
				report(start, n)
				return
			case ev.Kind == platform.EventKeyDown:
				fmt.Printf("key down: %d\n", ev.Key)
			}
		}

		drawTestPattern(fb, n, win)

		if err := win.Present(fb); err != nil {
			fmt.Fprintf(os.Stderr, "glidergo: present: %v\n", err)
			os.Exit(1)
		}
		n++
		if *frames > 0 && n >= *frames {
			report(start, n)
			return
		}
		if !*bench {
			next = next.Add(tick)
			if d := time.Until(next); d > 0 {
				time.Sleep(d)
			} else {
				next = time.Now() // we fell behind; do not accumulate debt
			}
		}
	}
}

func report(start time.Time, n int) {
	el := time.Since(start)
	if n == 0 || el == 0 {
		return
	}
	fmt.Printf("glidergo: %d frames in %v (%.1f fps)\n", n, el.Round(time.Millisecond), float64(n)/el.Seconds())
}

// drawTestPattern exercises the paths the real renderer will use: a full-surface
// fill, per-pixel writes, and a rectangle whose position depends on held keys.
func drawTestPattern(fb *platform.Framebuffer, n int, win platform.Window) {
	fb.Fill(color.RGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 0xff})

	// Horizon band, standing in for a room's floor line.
	for y := 360; y < 368; y++ {
		for x := 0; x < fb.W; x++ {
			fb.Set(x, y, color.RGBA{R: 0x44, G: 0x44, B: 0x66, A: 0xff})
		}
	}

	// A "glider" the arrow keys nudge, so input is visibly wired up.
	x := 300 + int(hOffset)
	if win.KeyDown(platform.KeyLeft) {
		hOffset -= 3
	}
	if win.KeyDown(platform.KeyRight) {
		hOffset += 3
	}
	y := 200 + int(8*sin(float64(n)/12))
	for dy := 0; dy < 12; dy++ {
		for dx := 0; dx < 48; dx++ {
			fb.Set(x+dx, y+dy, color.RGBA{R: 0xe8, G: 0xe8, B: 0xf0, A: 0xff})
		}
	}

	// Frame counter as a moving pixel ruler along the top: cheap, and it makes
	// dropped frames obvious to the eye.
	for i := 0; i < 8; i++ {
		fb.Set((n*2+i)%fb.W, 4, color.RGBA{R: 0xff, G: 0xcc, B: 0x33, A: 0xff})
	}
}

var hOffset float64

// sin avoids importing math for one call in a smoke test; a 3-term Taylor
// expansion is plenty for a wobble.
func sin(t float64) float64 {
	for t > 3.14159265 {
		t -= 2 * 3.14159265
	}
	for t < -3.14159265 {
		t += 2 * 3.14159265
	}
	t2 := t * t
	return t * (1 - t2/6*(1-t2/20*(1-t2/42)))
}
