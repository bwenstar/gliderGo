//go:build linux && cgo

// Package x11 is the Linux backend: a window and a keyboard, over raw Xlib.
//
// It deliberately uses only libX11, which is present on the development host
// (and on every Linux desktop) with no extra packages. XPutImage over the wire
// benchmarks at 533 fps for full 640x480x32 frames on this machine -- 8.9x the
// 60 fps budget -- so MIT-SHM (which would need libxext-dev) is not required.
// See docs/DEV_ENVIRONMENT.md §5.
package x11

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/XKBlib.h>
#include <X11/keysym.h>
#include <stdlib.h>
#include <string.h>

// XDestroyImage is a function-pointer macro in Xlib, which cgo cannot call.
static void destroy_image(XImage *img) { XDestroyImage(img); }

// Reading a member of the XEvent union from Go is awkward, and getting it wrong
// is a memory-safety bug rather than a compile error. These three accessors keep
// the union handling in C where the compiler checks it.
static int  ev_type(XEvent *e)     { return e->type; }
static unsigned int ev_keycode(XEvent *e) { return e->xkey.keycode; }
static unsigned long ev_msg0(XEvent *e)   { return (unsigned long)e->xclient.data.l[0]; }
static void ev_size(XEvent *e, int *w, int *h) { *w = e->xconfigure.width; *h = e->xconfigure.height; }
static int  ev_expose_count(XEvent *e) { return e->xexpose.count; }
*/
import "C"

import (
	"fmt"
	"unsafe"

	"glidergo/internal/platform"
)

// Window is an X11 window plus its presentation image.
type Window struct {
	dpy    *C.Display
	win    C.Window
	gc     C.GC
	img    *C.XImage
	buf    unsafe.Pointer // C-allocated: XImage retains it, so it may not be Go memory
	bufLen int

	w, h   int // framebuffer (logical) size
	scale  int
	pw, ph int // presented (physical) size

	wmDelete C.Atom
	down     [256]bool // by platform.Key
	keysyms  map[C.KeySym]platform.Key
	closed   bool
}

// New opens a window. The caller owns Close.
func New(cfg platform.Config) (*Window, error) {
	w, h, scale := cfg.Width, cfg.Height, cfg.Scale
	if w == 0 {
		w = platform.ScreenWidth
	}
	if h == 0 {
		h = platform.ScreenHeight
	}
	if scale < 1 {
		scale = 1
	}

	dpy := C.XOpenDisplay(nil)
	if dpy == nil {
		return nil, fmt.Errorf("x11: cannot open display (is DISPLAY set?)")
	}
	scr := C.XDefaultScreen(dpy)
	depth := C.XDefaultDepth(dpy, scr)
	if depth != 24 && depth != 32 {
		C.XCloseDisplay(dpy)
		return nil, fmt.Errorf("x11: unsupported display depth %d (need 24 or 32)", int(depth))
	}
	vis := C.XDefaultVisual(dpy, scr)
	root := C.XRootWindow(dpy, scr)

	pw, ph := w*scale, h*scale
	win := C.XCreateSimpleWindow(dpy, root, 0, 0, C.uint(pw), C.uint(ph), 0,
		C.XBlackPixel(dpy, scr), C.XBlackPixel(dpy, scr))

	// Ask the window manager not to let the user resize away from an integer
	// scale: the original game has no notion of a resizable playfield.
	hints := C.XAllocSizeHints()
	hints.flags = C.PMinSize | C.PMaxSize
	hints.min_width, hints.max_width = C.int(pw), C.int(pw)
	hints.min_height, hints.max_height = C.int(ph), C.int(ph)
	C.XSetWMNormalHints(dpy, win, hints)
	C.XFree(unsafe.Pointer(hints))

	C.XSelectInput(dpy, win,
		C.ExposureMask|C.KeyPressMask|C.KeyReleaseMask|C.StructureNotifyMask|C.FocusChangeMask)

	name := C.CString("WM_DELETE_WINDOW")
	defer C.free(unsafe.Pointer(name))
	wmDelete := C.XInternAtom(dpy, name, C.False)
	C.XSetWMProtocols(dpy, win, &wmDelete, 1)

	bufLen := pw * ph * 4
	buf := C.malloc(C.size_t(bufLen))
	if buf == nil {
		C.XCloseDisplay(dpy)
		return nil, fmt.Errorf("x11: out of memory for %dx%d image", pw, ph)
	}
	C.memset(buf, 0, C.size_t(bufLen))

	img := C.XCreateImage(dpy, vis, C.uint(depth), C.ZPixmap, 0,
		(*C.char)(buf), C.uint(pw), C.uint(ph), 32, 0)
	if img == nil {
		C.free(buf)
		C.XCloseDisplay(dpy)
		return nil, fmt.Errorf("x11: XCreateImage failed")
	}

	win2 := &Window{
		dpy: dpy, win: win, img: img, buf: buf, bufLen: bufLen,
		w: w, h: h, scale: scale, pw: pw, ph: ph,
		wmDelete: wmDelete, keysyms: keysymTable(),
	}
	win2.gc = C.XCreateGC(dpy, C.Drawable(win), 0, nil)
	if err := win2.SetTitle(cfg.Title); err != nil {
		win2.Close()
		return nil, err
	}
	C.XMapWindow(dpy, win)
	C.XFlush(dpy)
	return win2, nil
}

// SetTitle sets the window caption.
func (w *Window) SetTitle(s string) error {
	if s == "" {
		s = "gliderGo"
	}
	c := C.CString(s)
	defer C.free(unsafe.Pointer(c))
	C.XStoreName(w.dpy, w.win, c)
	return nil
}

// Present uploads fb and displays it, magnifying by the configured integer
// scale with nearest-neighbour sampling (the only scaling that keeps 1994 pixel
// art honest).
func (w *Window) Present(fb *platform.Framebuffer) error {
	if fb.W != w.w || fb.H != w.h {
		return fmt.Errorf("x11: framebuffer is %dx%d, window expects %dx%d", fb.W, fb.H, w.w, w.h)
	}
	dst := unsafe.Slice((*byte)(w.buf), w.bufLen)
	dstStride := w.pw * 4

	if w.scale == 1 {
		for y := 0; y < fb.H; y++ {
			copy(dst[y*dstStride:(y+1)*dstStride], fb.Pix[y*fb.Stride:y*fb.Stride+fb.W*4])
		}
	} else {
		for y := 0; y < fb.H; y++ {
			src := fb.Pix[y*fb.Stride : y*fb.Stride+fb.W*4]
			row := dst[y*w.scale*dstStride:][:dstStride]
			for x := 0; x < fb.W; x++ {
				px := src[x*4 : x*4+4]
				for s := 0; s < w.scale; s++ {
					copy(row[(x*w.scale+s)*4:], px)
				}
			}
			// Replicate the expanded row for the remaining scale-1 lines.
			for s := 1; s < w.scale; s++ {
				copy(dst[(y*w.scale+s)*dstStride:][:dstStride], row)
			}
		}
	}

	C.XPutImage(w.dpy, C.Drawable(w.win), w.gc, w.img, 0, 0, 0, 0, C.uint(w.pw), C.uint(w.ph))
	C.XFlush(w.dpy)
	return nil
}

// PollEvents drains the X queue without blocking.
func (w *Window) PollEvents() []platform.Event {
	var out []platform.Event
	var ev C.XEvent
	for C.XPending(w.dpy) > 0 {
		C.XNextEvent(w.dpy, &ev)
		switch C.ev_type(&ev) {
		case C.KeyPress, C.KeyRelease:
			isDown := C.ev_type(&ev) == C.KeyPress
			// Index 0 is the unshifted keysym, which is what we want: the game
			// binds physical keys, not characters.
			ks := C.XkbKeycodeToKeysym(w.dpy, C.KeyCode(C.ev_keycode(&ev)), 0, 0)
			k := w.keysyms[ks]
			if k == platform.KeyUnknown {
				continue
			}
			repeat := isDown && w.down[k]
			w.down[k] = bool(isDown)
			kind := platform.EventKeyUp
			if isDown {
				kind = platform.EventKeyDown
			}
			out = append(out, platform.Event{Kind: kind, Key: k, Repeat: repeat})
		case C.ClientMessage:
			if C.Atom(C.ev_msg0(&ev)) == w.wmDelete {
				w.closed = true
				out = append(out, platform.Event{Kind: platform.EventQuit})
			}
		case C.FocusOut:
			// Losing focus must clear held keys or the glider flies off on its
			// own when the user alt-tabs away mid-press.
			w.down = [256]bool{}
			out = append(out, platform.Event{Kind: platform.EventFocus, Focused: false})
		case C.FocusIn:
			out = append(out, platform.Event{Kind: platform.EventFocus, Focused: true})
		case C.Expose:
			// X sends one Expose per damaged rectangle, with `count` counting the
			// ones still to come; the last of a series has count 0. Since Present
			// uploads the whole framebuffer there is nothing to gain from the
			// individual rects, so only the last one is reported and a series
			// becomes a single redraw.
			if C.ev_expose_count(&ev) == 0 {
				out = append(out, platform.Event{Kind: platform.EventExpose})
			}
		case C.ConfigureNotify:
			var cw, ch C.int
			C.ev_size(&ev, &cw, &ch)
			out = append(out, platform.Event{Kind: platform.EventResize, W: int(cw), H: int(ch)})
		}
	}
	return out
}

// KeyDown reports whether k is held right now.
func (w *Window) KeyDown(k platform.Key) bool {
	if k <= 0 || int(k) >= len(w.down) {
		return false
	}
	return w.down[k]
}

// Closed reports whether the window manager has asked us to quit.
func (w *Window) Closed() bool { return w.closed }

// Close releases the display connection and image memory.
func (w *Window) Close() error {
	if w.dpy == nil {
		return nil
	}
	if w.img != nil {
		// XDestroyImage would free our malloc'd buffer too; do it by hand so the
		// ownership is obvious.
		w.img.data = nil
		C.destroy_image(w.img)
		w.img = nil
	}
	if w.buf != nil {
		C.free(w.buf)
		w.buf = nil
	}
	if w.gc != nil {
		C.XFreeGC(w.dpy, w.gc)
	}
	C.XDestroyWindow(w.dpy, w.win)
	C.XCloseDisplay(w.dpy)
	w.dpy = nil
	return nil
}

// keysymTable maps X keysyms to platform keys. Built once per window; the table
// is small enough that a map lookup per event is irrelevant next to a blit.
func keysymTable() map[C.KeySym]platform.Key {
	m := map[C.KeySym]platform.Key{
		C.XK_Left: platform.KeyLeft, C.XK_Right: platform.KeyRight,
		C.XK_Up: platform.KeyUp, C.XK_Down: platform.KeyDown,
		C.XK_space: platform.KeySpace, C.XK_Return: platform.KeyReturn,
		C.XK_KP_Enter: platform.KeyReturn, C.XK_Escape: platform.KeyEscape,
		C.XK_Tab: platform.KeyTab, C.XK_BackSpace: platform.KeyDelete,
		C.XK_Delete: platform.KeyDelete,
		C.XK_minus:  platform.KeyMinus, C.XK_equal: platform.KeyEqual,
		C.XK_bracketleft: platform.KeyLeftBracket, C.XK_bracketright: platform.KeyRightBracket,
		C.XK_comma: platform.KeyComma, C.XK_period: platform.KeyPeriod,
		C.XK_slash: platform.KeySlash, C.XK_semicolon: platform.KeySemicolon,
		C.XK_apostrophe: platform.KeyQuote, C.XK_backslash: platform.KeyBackslash,
		C.XK_grave:   platform.KeyGrave,
		C.XK_Shift_L: platform.KeyShift, C.XK_Shift_R: platform.KeyShift,
		C.XK_Control_L: platform.KeyControl, C.XK_Control_R: platform.KeyControl,
		C.XK_Alt_L: platform.KeyAlt, C.XK_Alt_R: platform.KeyAlt,
		C.XK_Super_L: platform.KeySuper, C.XK_Super_R: platform.KeySuper,
		C.XK_F1: platform.KeyF1, C.XK_F2: platform.KeyF2, C.XK_F3: platform.KeyF3,
		C.XK_F4: platform.KeyF4, C.XK_F5: platform.KeyF5, C.XK_F6: platform.KeyF6,
		C.XK_F7: platform.KeyF7, C.XK_F8: platform.KeyF8, C.XK_F9: platform.KeyF9,
		C.XK_F10: platform.KeyF10, C.XK_F11: platform.KeyF11, C.XK_F12: platform.KeyF12,
	}
	for i := 0; i < 26; i++ {
		m[C.KeySym(C.XK_a+C.int(i))] = platform.KeyA + platform.Key(i)
	}
	for i := 0; i < 10; i++ {
		m[C.KeySym(C.XK_0+C.int(i))] = platform.Key0 + platform.Key(i)
		m[C.KeySym(C.XK_KP_0+C.int(i))] = platform.Key0 + platform.Key(i)
	}
	return m
}
