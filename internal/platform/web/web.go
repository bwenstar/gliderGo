//go:build js && wasm

// Package web is the browser backend: a <canvas> for the pixels, and the page's keyboard events
// for the keys and the text.
//
// The page is web/ at the top of the repository, and it owns everything a browser asks of a page
// before a program may run: loading the binary, the click or key press that is allowed to start
// the sound, the storage the game's files live in. This package is the window, and only that. New
// binds to the page's canvas, whose id is the one contract between the two, and Present puts
// pixels on it.
//
// A canvas keeps what it was drawn, so Present sends only what changed (platform.Changes) and a
// paused game costs nothing. The canvas is always the game's own 640x480 and CSS magnifies it, so
// a magnified frame costs the program nothing more than a 1x one: the browser does the scaling, on
// the GPU where it has one. That is also why there is no Expand here.
//
// What a key means and how large the canvas is drawn are decided in keys.go, which has no build
// tag and is tested on any host. This file reads the browser's events and hands them over, and it
// is tested in a browser, by tools/webcheck.
package web

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/bwenstar/gliderGo/internal/platform"
)

// CanvasID is the id of the page's <canvas>. web/index.html has the element.
const CanvasID = "glidergo"

// Window is the page's canvas.
type Window struct {
	// mu is for what the listeners write and the game reads. A listener runs while the game is
	// asleep -- js/wasm has one thread, and Go hands the browser its event loop only when every
	// goroutine is parked -- so the two never actually contend, and the lock is what makes that
	// a guarantee rather than an observation about the scheduler.
	mu      sync.Mutex
	queue   []platform.Event
	keys    keyboard
	focused bool
	mac     bool // whose browsers send no keyup while Command is held (keyboard.commandUp)

	canvas, ctx, img, data js.Value
	rgba                   []byte // the canvas's pixels, as the ImageData holds them
	w, h                   int
	scale                  int // Config.Scale: the most the canvas is shown at
	changes                platform.Changes
	sent                   time.Time // when Present last reached the canvas

	listeners []listener
}

// burst is how soon after a present another is not worth sending: half a paint at 60 Hz.
const burst = 8 * time.Millisecond

// Coalesce is platform.Coalescer: a present within burst of the last one sent may wait for the
// host's next sleep, which is the soonest the page could paint it anyway.
//
// Not every present may wait, although none of them can be seen before that sleep. A host that
// never sleeps -- -bench, or a machine too slow to keep up, whose frame limiter never has time to
// wait -- would then never send one, and its page would show nothing until the end. With burst,
// such a run still sends one every 8 ms, and the page has the latest of them to paint the moment
// it gets a turn. Play and a pause present 33 ms apart or more, so what waits is a burst: a
// wipe's strips, and the few presents a game opens with.
func (w *Window) Coalesce() bool { return time.Since(w.sent) < burst }

type listener struct {
	target js.Value
	name   string
	fn     js.Func
}

// New binds to the page's canvas and listens to the page.
func New(cfg platform.Config) (*Window, error) {
	w, h := cfg.Width, cfg.Height
	if w == 0 {
		w = platform.ScreenWidth
	}
	if h == 0 {
		h = platform.ScreenHeight
	}
	global := js.Global()
	doc := global.Get("document")
	canvas := doc.Call("getElementById", CanvasID)
	if !canvas.Truthy() {
		return nil, fmt.Errorf("web: the page has no <canvas id=%q>", CanvasID)
	}
	canvas.Set("width", w)
	canvas.Set("height", h)

	// No alpha: a canvas that has none is composited as opaque, and the game has no use for one.
	ctx := canvas.Call("getContext", "2d", map[string]any{"alpha": false})
	if !ctx.Truthy() {
		return nil, errors.New("web: this browser gave the canvas no 2D context")
	}
	img := ctx.Call("createImageData", w, h)
	win := &Window{
		canvas: canvas, ctx: ctx, img: img, data: img.Get("data"),
		rgba: make([]byte, w*h*4), w: w, h: h, scale: cfg.Scale, focused: true,
		// navigator.platform is deprecated and still everywhere, and it is what an iPad with a
		// keyboard calls a Mac too, which for this is right.
		mac: strings.HasPrefix(global.Get("navigator").Get("platform").String(), "Mac"),
	}
	// The alpha bytes are written once, here. Present writes the other three.
	for i := 3; i < len(win.rgba); i += 4 {
		win.rgba[i] = 0xff
	}
	if cfg.Title != "" {
		win.SetTitle(cfg.Title)
	}

	win.listen(global, "keydown", func(e js.Value) { win.key(e, true) })
	win.listen(global, "keyup", func(e js.Value) { win.key(e, false) })

	// Three ways to lose the keyboard. blur and focus are another window or the address bar;
	// visibilitychange is another tab, or a minimised window, which on some browsers sends no
	// blur at all.
	win.listen(global, "blur", func(js.Value) { win.focus(false) })
	win.listen(global, "focus", func(js.Value) { win.focus(true) })
	win.listen(doc, "visibilitychange", func(js.Value) {
		win.focus(doc.Get("visibilityState").String() == "visible")
	})

	// A 2D canvas can lose its pixels -- the GPU reset, or the browser needed the memory back --
	// and this is the event that says it has them again, blank. It is EventExpose's own case: a
	// paused game draws no frames, so without the expose nothing would ever repaint it. Chrome
	// 99 and later send it; a browser that loses a canvas and does not say so is not one this
	// can do anything about.
	win.listen(canvas, "contextrestored", func(js.Value) {
		win.mu.Lock()
		win.changes.All()
		win.queue = append(win.queue, platform.Event{Kind: platform.EventExpose})
		win.mu.Unlock()
	})

	win.listen(global, "resize", func(js.Value) { win.layout() })
	win.layout()
	return win, nil
}

func (w *Window) listen(target js.Value, name string, f func(js.Value)) {
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		f(args[0])
		return nil
	})
	target.Call("addEventListener", name, fn)
	w.listeners = append(w.listeners, listener{target, name, fn})
}

// key is both key listeners.
func (w *Window) key(e js.Value, down bool) {
	p := press{
		code:      e.Get("code").String(),
		key:       e.Get("key").String(),
		down:      down,
		repeat:    e.Get("repeat").Bool(),
		ctrl:      e.Get("ctrlKey").Bool(),
		meta:      e.Get("metaKey").Bool(),
		altGraph:  e.Call("getModifierState", "AltGraph").Bool(),
		composing: e.Get("isComposing").Bool(),
	}
	w.mu.Lock()
	ev, ok, prevent := w.keys.event(p)
	if ok {
		w.queue = append(w.queue, ev)
	}
	if !down && w.mac && isCommand(p.code) {
		w.queue = append(w.queue, w.keys.commandUp()...)
	}
	w.mu.Unlock()
	if prevent {
		e.Call("preventDefault")
	}
}

func (w *Window) focus(f bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if f == w.focused {
		return // blur and visibilitychange can both report one loss
	}
	w.focused = f
	if !f {
		w.keys.release()
	}
	w.queue = append(w.queue, platform.Event{Kind: platform.EventFocus, Focused: f})
}

// layout sizes the canvas for the viewport it is in now (keys.go's shown).
func (w *Window) layout() {
	g := js.Global()
	cw, ch := shown(w.scale, w.w, w.h,
		g.Get("innerWidth").Float(), g.Get("innerHeight").Float(), g.Get("devicePixelRatio").Float())
	style := w.canvas.Get("style")
	style.Set("width", fmt.Sprintf("%.3fpx", cw))
	style.Set("height", fmt.Sprintf("%.3fpx", ch))
}

// Present swizzles what changed from BGRX to RGBA and puts only that on the canvas.
func (w *Window) Present(fb *platform.Framebuffer) error {
	if fb.W != w.w || fb.H != w.h {
		return fmt.Errorf("web: a %dx%d frame for a %dx%d canvas", fb.W, fb.H, w.w, w.h)
	}
	w.sent = time.Now()
	for _, s := range w.changes.Diff(fb) {
		for y := s.Y0; y < s.Y1; y++ {
			src := fb.Pix[y*fb.Stride+s.X0*4 : y*fb.Stride+s.X1*4]
			dst := w.rgba[(y*w.w+s.X0)*4 : (y*w.w+s.X1)*4]
			for i := 0; i+3 < len(src); i += 4 {
				dst[i], dst[i+1], dst[i+2] = src[i+2], src[i+1], src[i]
			}
		}
		// Whole rows go across, because the ImageData is the whole canvas and a row is the
		// unit it is contiguous in; putImageData's dirty rectangle then draws only the span.
		off, end := s.Y0*w.w*4, s.Y1*w.w*4
		js.CopyBytesToJS(w.data.Call("subarray", off, end), w.rgba[off:end])
		w.ctx.Call("putImageData", w.img, 0, 0, s.X0, s.Y0, s.X1-s.X0, s.Y1-s.Y0)
	}
	return nil
}

// PollEvents drains what the listeners queued. It never blocks.
func (w *Window) PollEvents() []platform.Event {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := w.queue
	w.queue = nil
	return out
}

// KeyDown reports whether k is held.
func (w *Window) KeyDown(k platform.Key) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.keys.down(k)
}

// SetTitle is the tab's title.
func (w *Window) SetTitle(s string) error {
	js.Global().Get("document").Set("title", s)
	return nil
}

// Close stops listening. The canvas keeps the last frame, which is what the page shows under
// whatever it says next.
func (w *Window) Close() error {
	for _, l := range w.listeners {
		l.target.Call("removeEventListener", l.name, l.fn)
		l.fn.Release()
	}
	w.listeners = nil
	return nil
}

// Room is the screen the browser is on, in device pixels: what full screen would give the game.
//
// Not the viewport, which is smaller by the browser's own toolbars and changes whenever the window
// does. Auto is the most the game is drawn at, and the page shrinks it to fit a smaller viewport
// and grows it back when there is room again (shown, in keys.go), so the answer wanted here is
// the most there could be. A 1080p screen is then auto 2x, which full screen shows exactly and a
// maximised window shows a little smaller, rather than auto 1x everywhere because 960 rows do not
// quite fit under a toolbar.
func Room() (platform.Room, error) {
	g := js.Global()
	dpr := g.Get("devicePixelRatio").Float()
	if dpr <= 0 {
		dpr = 1
	}
	screen := g.Get("screen")
	return platform.Room{
		W:    int(screen.Get("width").Float() * dpr),
		H:    int(screen.Get("height").Float() * dpr),
		From: "the screen the browser is on",
	}, nil
}
