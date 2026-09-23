//go:build linux && cgo

// Package x11 is the Linux backend: a window and a keyboard, over raw Xlib.
//
// It deliberately uses only libX11, which is present on the development host
// (and on every Linux desktop) with no extra packages. Frames go to the server with
// XPutImage, and not MIT-SHM, which would need libxext-dev: Present sends only the
// rows a frame changed (platform.Changes), so a frame that moved a glider is a few
// dozen rows and not the 19.7 MB a whole frame is at 4x. docs/IMPROVEMENTS.md 2.76
// has the measurements, and docs/DEV_ENVIRONMENT.md §5 the reasoning.
//
// It asks the server for one thing that is not the default, XKB's detectable
// auto-repeat, and Event.Repeat depends on it: without it X reports a held key as a
// stream of releases and presses that cannot be told from taps (New says more). Xephyr
// and the DCV desktop on the development host both support it. A server that does not
// is not refused, because the game still plays -- KeyDown reads the bitmap, and the
// pairs leave it held -- but on one, Repeat is never set and a held Escape on the house
// picker quits the game. GLFW's answer for such a server, peeking at the queue on a
// release for a press with the same keycode and time, is not written until a server
// needs it (docs/IMPROVEMENTS.md 2.72).
package x11

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/XKBlib.h>
#include <X11/keysym.h>
#include <X11/Xatom.h>
#include <stdlib.h>
#include <string.h>

enum { atom_cardinal = XA_CARDINAL, atom_window = XA_WINDOW };

// root_longs reads a list property of the root window, of 32-bit values of type want, into
// out: at most max of them. It returns how many it read, or -1 if the root has no such
// property. Xlib hands format-32 data back as C longs whatever a long's width, which is why
// the copy is here and not in Go.
static int root_longs(Display *d, const char *name, Atom want, long *out, int max) {
	Atom a = XInternAtom(d, name, True);
	if (a == None) return -1;
	Atom type = None;
	int format = 0;
	unsigned long n = 0, after = 0;
	unsigned char *p = NULL;
	if (XGetWindowProperty(d, DefaultRootWindow(d), a, 0, max, False, want,
			&type, &format, &n, &after, &p) != Success) return -1;
	int k = -1;
	if (p != NULL && type == want && format == 32) {
		k = n < (unsigned long)max ? (int)n : max;
		memcpy(out, p, (size_t)k * sizeof(long));
	}
	if (p != NULL) XFree(p);
	return k;
}

// pointer_at is where the pointer is on the root window, or -1,-1 if it is on another screen.
static void pointer_at(Display *d, int *x, int *y) {
	Window r, c;
	int wx, wy;
	unsigned int m;
	if (!XQueryPointer(d, DefaultRootWindow(d), &r, &c, x, y, &wx, &wy, &m)) *x = *y = -1;
}

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

// ev_text is XLookupString: what this keystroke typed, given the modifiers and the
// keyboard mapping the server is using. It is the only part of this backend that
// knows about characters at all -- the game binds physical keys -- and it exists so
// that the high-score screen can be typed into on a keyboard that is not American.
//
// The bytes come back as ISO Latin-1 and not in the locale's encoding, which is
// exactly what the caller wants: Latin-1 is the first 256 code points of Unicode, so
// each byte is its own rune with no conversion table.
static int ev_text(XEvent *e, char *buf, int n) {
	KeySym ignored;
	return XLookupString(&e->xkey, buf, n, &ignored, NULL);
}
*/
import "C"

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"github.com/bwenstar/gliderGo/internal/platform"
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

	// changes is what the window was last sent, so that Present sends only what differs.
	// mapped is whether the window is on screen: an unmapped one is not sent frames, and
	// what it showed is gone when it comes back, as it is after an Expose.
	changes platform.Changes
	mapped  bool

	wmDelete C.Atom
	down     [256]bool // by platform.Key
	held     [256]bool // by X keycode, for Event.Repeat
	keysyms  map[C.KeySym]platform.Key
	closed   bool
}

// noDisplay is the error for a display that will not open. XOpenDisplay reports failure and
// nothing else -- no errno, no reason -- so the only diagnosis available is the environment it
// was asked to connect to. See platform.DisplayAdvice for the three cases and why one sentence
// was wrong about two of them.
func noDisplay() error {
	return fmt.Errorf("x11: cannot open a window: %s",
		platform.DisplayAdvice(os.Getenv("DISPLAY"), os.Getenv("WAYLAND_DISPLAY")))
}

// Room is the space a window may have on this display (platform.Room). It opens a connection
// of its own, because it is asked before New, which needs the answer to size the window.
//
// libX11 alone knows the size of the screen and no more; which monitor is where is XRandR's or
// Xinerama's, and each is a library this port does not link. So it reads what the window
// manager publishes on the root window instead. GNOME's mutter publishes each monitor's work
// area, the part a panel or a dock does not cover, as _GTK_WORKAREAS_D<desktop>; every EWMH
// window manager publishes _NET_WORKAREA, which on more than one monitor is their union. With
// neither there is no window manager to reserve anything and the screen is the answer.
func Room() (platform.Room, error) {
	spare.Lock()
	defer spare.Unlock()
	if spare.dpy == nil {
		if spare.dpy = C.XOpenDisplay(nil); spare.dpy == nil {
			return platform.Room{}, noDisplay()
		}
	}
	dpy := spare.dpy

	scr := C.XDefaultScreen(dpy)
	desk := 0
	if v := rootLongs(dpy, "_NET_CURRENT_DESKTOP", C.atom_cardinal, 1); len(v) == 1 && v[0] >= 0 && v[0] < 64 {
		desk = v[0]
	}
	monitors := rootLongs(dpy, fmt.Sprintf("_GTK_WORKAREAS_D%d", desk), C.atom_cardinal, 4*16)
	work := rootLongs(dpy, "_NET_WORKAREA", C.atom_cardinal, 4*64)
	if len(work) >= 4*(desk+1) {
		work = work[4*desk:]
	}
	wm := len(rootLongs(dpy, "_NET_SUPPORTING_WM_CHECK", C.atom_window, 1)) == 1
	var px, py C.int
	C.pointer_at(dpy, &px, &py)
	return chooseRoom(int(C.XDisplayWidth(dpy, scr)), int(C.XDisplayHeight(dpy, scr)),
		monitors, work, int(px), int(py), wm), nil
}

// spare is the connection Room opened, which is closed once New has one of its own and not
// before. A server with no other client -- Xvfb under xvfb-run, as CI runs the game, or a bare
// Xephyr -- resets when its last client leaves and refuses connections while it does, so a Room
// that closed its connection straight away could have the New after it refused. This package's
// tests, which do the same thing, were refused every time on a bare Xephyr.
var spare struct {
	sync.Mutex
	dpy *C.Display
}

// closeSpare closes the connection Room kept, if it kept one.
func closeSpare() {
	spare.Lock()
	defer spare.Unlock()
	if spare.dpy != nil {
		C.XCloseDisplay(spare.dpy)
		spare.dpy = nil
	}
}

// rootLongs is root_longs as a slice, nil when the root has no such property.
func rootLongs(dpy *C.Display, name string, want C.Atom, max int) []int {
	c := C.CString(name)
	defer C.free(unsafe.Pointer(c))
	buf := make([]C.long, max)
	n := int(C.root_longs(dpy, c, want, &buf[0], C.int(max)))
	if n <= 0 {
		return nil
	}
	out := make([]int, n)
	for i := range out {
		out[i] = int(buf[i])
	}
	return out
}

// frameW and frameH are what a window manager's frame is allowed: a border either side and a
// title bar. A window's real frame is only published once it is mapped (_NET_FRAME_EXTENTS),
// which is after the size that has to fit inside the work area was chosen, so this is a fixed
// allowance and a generous one: mutter's title bar is 37 pixels, and a border is rarely more
// than 4 a side.
const frameW, frameH = 16, 56

// chooseRoom is Room's decision, from what the root window said: the screen's size, a work area
// per monitor (x, y, w, h each; nil without them), the desktop's work area (nil without it),
// where the pointer is, and whether a window manager is running.
//
// The monitor is the one the pointer is on, since that is where a window manager puts a new
// window, or the nearest one to it: a pointer on a top bar is on no monitor's work area.
func chooseRoom(screenW, screenH int, monitors, work []int, px, py int, wm bool) platform.Room {
	r := platform.Room{W: screenW, H: screenH, From: "the screen"}
	best := -1
	for i := 0; i+4 <= len(monitors); i += 4 {
		x, y, w, h := monitors[i], monitors[i+1], monitors[i+2], monitors[i+3]
		if w <= 0 || h <= 0 {
			continue
		}
		dx := max(x-px, 0, px-(x+w-1))
		dy := max(y-py, 0, py-(y+h-1))
		if d := dx*dx + dy*dy; best < 0 || d < best {
			best = d
			r = platform.Room{W: w, H: h, From: "the work area of the monitor under the pointer"}
		}
	}
	if best < 0 && len(work) >= 4 && work[2] > 0 && work[3] > 0 {
		r = platform.Room{W: work[2], H: work[3], From: "the desktop's work area"}
	}
	if wm {
		r.W, r.H = r.W-frameW, r.H-frameH
		r.From += ", less a title bar"
	}
	return r
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
		return nil, noDisplay()
	}
	closeSpare()

	// Detectable auto-repeat, before this connection has asked for a single event. By
	// default an X server sends each repeat of a held key as a KeyRelease and a KeyPress
	// with the same timestamp, so the bitmap has always just seen the key go up and the
	// rule in PollEvents that marks a repeat -- a press while the key is already down --
	// never fires. Every `!ev.Repeat` guard above this backend was dead on Linux as a
	// result: a held Escape took the house picker back to the title screen and then quit
	// (docs/IMPROVEMENTS.md 2.72). With the flag the server sends repeats as presses and
	// nothing else, which is what Windows does, and the rule then marks the presses
	// win32.go's lParam bit 30 marks.
	//
	// XKB is part of libX11 -- XkbKeycodeToKeysym in PollEvents already uses it -- so this
	// costs no library. The answer is not acted on: a server that cannot do it is not
	// refused, for the reason the package comment gives, so there is nothing to do
	// differently here. supported is only somewhere for Xlib to write, and the test asks
	// the server again rather than trusting it (detectableAutoRepeat).
	var supported C.Bool
	C.XkbSetDetectableAutoRepeat(dpy, C.True, &supported)

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

// Present displays fb, magnifying by the configured integer scale with
// nearest-neighbour sampling (the only scaling that keeps 1994 pixel art honest).
//
// It sends what changed since the last frame and nothing else: each block
// platform.Changes reports is expanded into the image and put with one XPutImage
// of that block, since the image keeps every pixel it was given and the rest of it
// is still the frame the window shows. The expansion is platform.ExpandSpan rather
// than a loop here, because the win32 backend does exactly the same thing into a
// DIB and the arithmetic is worth having in one place -- one that can be
// unit-tested on a machine with no display, which this file cannot be. XPutImage's
// own errors are asynchronous and arrive at the X error handler, so there is
// nothing to check here.
func (w *Window) Present(fb *platform.Framebuffer) error {
	if fb.W != w.w || fb.H != w.h {
		return fmt.Errorf("x11: framebuffer is %dx%d, window expects %dx%d", fb.W, fb.H, w.w, w.h)
	}
	if !w.mapped {
		// Minimised, or not yet on screen. The server would throw the frame away.
		w.changes.All()
		return nil
	}
	img := unsafe.Slice((*byte)(w.buf), w.bufLen)
	s := w.scale
	for _, sp := range w.changes.Diff(fb) {
		if err := platform.ExpandSpan(img, w.pw*4, fb, s, sp); err != nil {
			return err
		}
		x, y := C.int(sp.X0*s), C.int(sp.Y0*s)
		C.XPutImage(w.dpy, C.Drawable(w.win), w.gc, w.img, x, y, x, y,
			C.uint((sp.X1-sp.X0)*s), C.uint((sp.Y1-sp.Y0)*s))
	}
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
			kc := C.ev_keycode(&ev)
			// Index 0 is the unshifted keysym, which is what we want: the game
			// binds physical keys, not characters.
			ks := C.XkbKeycodeToKeysym(w.dpy, C.KeyCode(kc), 0, 0)
			k := w.keysyms[ks]

			// A press while the same physical key is already down is a repeat, now
			// that New asks for detectable auto-repeat. It is tracked by keycode, not
			// by k, for the two cases k gets wrong: keys that share a platform.Key
			// (BackSpace and Delete, Return and KP_Enter, a digit and its keypad twin),
			// where Delete tapped with BackSpace held would read as a repeat and be
			// dropped, and keys with no platform.Key at all, which have no slot in
			// down. Windows' lParam bit 30 is per key too, so the two backends agree.
			// X keycodes are 8..255, so the array is always in range.
			repeat := isDown && w.held[kc]
			w.held[kc] = bool(isDown)

			// What the keystroke typed, on presses only -- a key release types
			// nothing, and a held key types repeatedly, so auto-repeat carries text
			// too. This is a second question about the same event and not a
			// replacement for the first: the glider is steered by Key and a player's
			// name is spelled by Text (see platform.Event).
			text := ""
			if isDown {
				text = eventText(&ev)
			}

			if k == platform.KeyUnknown {
				// A key this port has no name for still types. On a French layout the
				// keysym under the physical Q is `a`, and every accented key and every
				// key on a layout nobody here has seen is unmapped; dropping those
				// would make the high-score screen unusable outside the US. The event
				// goes out with KeyUnknown, which no binding matches, so a text field
				// sees the character and the game sees nothing.
				if text == "" {
					continue
				}
				out = append(out, platform.Event{Kind: platform.EventKeyDown, Repeat: repeat, Text: text})
				continue
			}

			w.down[k] = bool(isDown)
			kind := platform.EventKeyUp
			if isDown {
				kind = platform.EventKeyDown
			}
			out = append(out, platform.Event{Kind: kind, Key: k, Repeat: repeat, Text: text})
		case C.ClientMessage:
			if C.Atom(C.ev_msg0(&ev)) == w.wmDelete {
				w.closed = true
				out = append(out, platform.Event{Kind: platform.EventQuit})
			}
		case C.FocusOut:
			// Losing focus must clear held keys or the glider flies off on its
			// own when the user alt-tabs away mid-press.
			// held goes too, or a key released while another window had focus would
			// mark its next real press as a repeat and the shell would drop it.
			w.down = [256]bool{}
			w.held = [256]bool{}
			out = append(out, platform.Event{Kind: platform.EventFocus, Focused: false})
		case C.FocusIn:
			out = append(out, platform.Event{Kind: platform.EventFocus, Focused: true})
		case C.Expose:
			// X sends one Expose per damaged rectangle, with `count` counting the
			// ones still to come; the last of a series has count 0. The next
			// Present sends the whole frame, so there is nothing to gain from the
			// individual rects, and only the last one is reported: a series
			// becomes a single redraw. That redraw is the same frame the window
			// was last sent, when the game is paused, which is why All matters.
			w.changes.All()
			if C.ev_expose_count(&ev) == 0 {
				out = append(out, platform.Event{Kind: platform.EventExpose})
			}
		case C.MapNotify:
			w.mapped = true
			w.changes.All()
		case C.UnmapNotify:
			w.mapped = false
		case C.ConfigureNotify:
			var cw, ch C.int
			C.ev_size(&ev, &cw, &ch)
			out = append(out, platform.Event{Kind: platform.EventResize, W: int(cw), H: int(ch)})
		}
	}
	return out
}

// eventText is the printable part of what a key press typed.
//
// Everything unprintable is dropped rather than passed on, because the caller is a text
// field and the keys that produce control codes here are keys it handles itself: Return
// is 0x0D, Escape 0x1B, Tab 0x09 and Delete 0x7F, and all four already arrive as a Key.
// A field that appended Text blindly would put a carriage return in the middle of a
// player's name.
//
// The 0x80..0x9F band is dropped for the same reason: those are C1 control codes in
// Latin-1, not characters, whatever a Mac Roman table might make of the same bytes.
func eventText(ev *C.XEvent) string {
	// Sixteen is generous. XLookupString returns one byte for a character key and a
	// handful for the few keysyms with multi-character mappings; it truncates rather
	// than overflowing, and a truncated keystroke is not a case worth a heap
	// allocation per key press.
	var buf [16]C.char
	n := int(C.ev_text(ev, &buf[0], C.int(len(buf))))
	if n <= 0 {
		return ""
	}
	var rs []rune
	for i := 0; i < n && i < len(buf); i++ {
		b := byte(buf[i])
		if b < 0x20 || (b >= 0x7F && b < 0xA0) {
			continue
		}
		rs = append(rs, rune(b)) // Latin-1 byte == Unicode code point
	}
	return string(rs)
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

// detectableAutoRepeat asks the server whether this connection has detectable
// auto-repeat, and whether it could have. It is here rather than in the test because
// cgo is not allowed in a _test.go file, and it asks rather than remembering what New
// was told so that the test checks the server's state and not this package's account
// of it.
func (w *Window) detectableAutoRepeat() (on, supported bool) {
	var s C.Bool
	on = C.XkbGetDetectableAutoRepeat(w.dpy, &s) != 0
	return on, s != 0
}

// sync waits until the server has done everything asked of it, so that a benchmark's time
// includes the server's share of a Present and not only the request's.
func (w *Window) sync() { C.XSync(w.dpy, C.False) }

// shown is what the window shows, read back from the server as BGRX rows with no padding, and
// whether the window is on screen yet. Like detectableAutoRepeat it is for the test, and here
// because cgo is not allowed in a _test.go file.
func (w *Window) shown() ([]byte, bool) {
	w.PollEvents() // for the MapNotify
	if !w.mapped {
		return nil, false
	}
	C.XSync(w.dpy, C.False)
	img := C.XGetImage(w.dpy, C.Drawable(w.win), 0, 0, C.uint(w.pw), C.uint(w.ph), C.AllPlanes, C.ZPixmap)
	if img == nil {
		return nil, false
	}
	defer C.destroy_image(img)
	stride := int(img.bytes_per_line)
	data := unsafe.Slice((*byte)(unsafe.Pointer(img.data)), stride*w.ph)
	out := make([]byte, 0, w.pw*w.ph*4)
	for y := 0; y < w.ph; y++ {
		out = append(out, data[y*stride:][:w.pw*4]...)
	}
	return out, true
}

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
