//go:build windows

// Package win32 is the Windows backend: a window and a keyboard, in pure Go.
//
// HONEST CAVEAT, READ FIRST. This file was written on the airgapped Linux host the port is being
// developed on, which has no Windows toolchain to test with and no Windows to test on. It is still
// authored blind and still cannot be exercised from there: `GOOS=windows go build` and `go vet`
// pass for amd64 and arm64, the two pure pieces are unit-tested on Linux (see keys.go, which
// carries no build tag for exactly that reason), and the nearest-neighbour expansion below is
// platform.Expand -- the same function the x11 backend uses, tested at every scale and benchmarked.
//
// It was first run elsewhere on Windows Server 2025 (build 26100, amd64), on a real interactive
// desktop: 4,320 frames across six runs, and the pixels it put in its window were compared against
// the frames the same version renders on Linux and matched exactly. Window creation, the message
// pump, the blit and the scaled blit all work. docs/windows-first-run.md is the write-up.
//
// Since that run, Room sizes the window to the monitor, the window is placed in the middle of that
// monitor, and Present sent only what a frame changed (docs/IMPROVEMENTS.md 2.1, 2.76). v0.2.0 is
// the first release with all three. A build of its tree ran on the same desktop on 2026-09-30: Room
// chose 2x, the window was centred, and the title screen matched the Linux frame exactly. On one
// player's Windows 10 laptop the release showed a white window that filled in only where something
// moved, which is why Present now sends in bands and sweeps (2.81). Builds with that change ran on
// the same desktop that day, some made to lose sends on purpose, and a script typed keys into them.
// CI's service-session bench (below) asks for the placement, and reaches the Present only if its
// window opens; nothing checks what it drew, and no hermetic run calls Room.
//
// What that does NOT cover, and what a reader should still distrust: keyboard input beyond the
// arrow keys and Enter the script typed and the player steered with; arm64, which no machine here
// or there can run; and everything about a window the player interacts with -- resizing, focus
// loss, dragging, closing. ci.yml's `native` job compiles it on every push and benches it in a
// service session, which is a different case from a logged-in desktop and deliberately best-effort.
//
// No cgo and no dependencies, which is not a preference: the build host has no Go module proxy
// and no reachable GitHub, so go-sdl2, Ebitengine and golang.org/x/sys are simply unobtainable
// (docs/DEV_ENVIRONMENT.md 3). What is available is the standard library, and syscall is enough:
// syscall.NewLazyDLL resolves user32 and gdi32 entry points on first use, and syscall.NewCallback
// turns a Go function into something Windows can call as a window procedure.
// The result cross-compiles from Linux with CGO_ENABLED=0 and needs no runtime beyond the OS.
//
// The shape mirrors internal/platform/x11 deliberately, function for function, because the two
// have to behave identically for the game above them: one window at a fixed integer multiple of
// 640x480, an upload of what each frame changed, a held-key bitmap that survives losing focus,
// and Event.Text carrying what the host's own layout typed so the high-score screen works on a
// keyboard this port has never seen. Where the two differ it is Windows' doing and it is
// commented at the point of difference.
//
// TWO THINGS A READER SHOULD KNOW ABOUT THE MODEL.
//
// The goroutine that calls New must be the one that calls PollEvents, Present and Close. Win32
// delivers messages to the thread that created the window, so New calls runtime.LockOSThread and
// Close releases it. gliderGo's game loop is a single goroutine, so this costs nothing here; a
// caller who moved the loop to another goroutine would get a window that never reports a key.
//
// Dragging the window by its title bar freezes the game. Windows runs a modal message loop inside
// DefWindowProc for the duration of the drag, and it does not return until the mouse is released,
// so PollEvents cannot come back and the frame loop stops. That is inherent to a pump-driven
// backend without a second thread, it is not new (a 1994 Macintosh did the same thing), and the
// game state is unharmed -- the wall clock is what moves on. Worth knowing before treating it as
// a bug.
package win32

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/bwenstar/gliderGo/internal/platform"
)

// The three libraries, resolved on first use. Nothing is loaded until a proc below is called, so
// this costs nothing at start-up and nothing at all in a build that never opens a window.
//
// A word on DLL preloading, because loading a library by bare name is a real attack surface and
// this backend does it three times. LoadLibrary's search order begins with the directory the
// executable was loaded from -- and a release archive tells the player to run the binary from the
// directory they unpacked it into, which is precisely the arrangement a planted `user32.dll`
// relies on. What makes these three safe is that user32, gdi32 and kernel32 are all on Windows'
// KnownDLLs list, which is consulted before any path search and is not overridable by a file on
// disk. It is not a general licence: a DLL that is *not* a KnownDLL must be loaded by absolute
// path, and nothing here should grow a fourth entry without checking which kind it is.
// golang.org/x/sys/windows.NewLazySystemDLL pins the system directory explicitly and is the
// belt-and-braces answer; it is a module, and this port has no dependencies (internal/module).
var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procShowWindow          = user32.NewProc("ShowWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procPeekMessageW        = user32.NewProc("PeekMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procAdjustWindowRect    = user32.NewProc("AdjustWindowRect")
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procSetWindowTextW      = user32.NewProc("SetWindowTextW")
	procBeginPaint          = user32.NewProc("BeginPaint")
	procEndPaint            = user32.NewProc("EndPaint")
	procLoadCursorW         = user32.NewProc("LoadCursorW")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procMonitorFromRect     = user32.NewProc("MonitorFromRect")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
	procIsIconic            = user32.NewProc("IsIconic")

	// The two ways to say "do not scale my pixels". Neither is on every supported Windows, so
	// both are looked up and neither is required; see dpiAware.
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	procStretchDIBits = gdi32.NewProc("StretchDIBits")
)

// Window messages, class and window styles, and the handful of GDI constants, from WinUser.h and
// WinGDI.h. Named rather than inlined so the switch in wndProc reads as the documentation does.
const (
	wmDestroy    = 0x0002
	wmSize       = 0x0005
	wmSetFocus   = 0x0007
	wmKillFocus  = 0x0008
	wmPaint      = 0x000F
	wmClose      = 0x0010
	wmQuit       = 0x0012
	wmEraseBkgnd = 0x0014
	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmChar       = 0x0102
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105

	csVRedraw = 0x0001
	csHRedraw = 0x0002
	csOwnDC   = 0x0020

	// WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX, and deliberately not WS_THICKFRAME or
	// WS_MAXIMIZEBOX: the original game has no notion of a resizable playfield, so the window
	// cannot be dragged bigger, maximised, or snapped to half a screen. This is the same
	// decision the x11 backend makes with min-size and max-size hints, taken the Windows way --
	// by leaving the styles out, which a window manager cannot decline to honour.
	wsWindow     = 0x00C00000 | 0x00080000 | 0x00020000
	cwUseDefault = 0x80000000

	swShow = 5

	monitorDefaultToNearest = 2

	pmRemove = 0x0001

	idcArrow = 32512

	biRGB        = 0
	dibRGBColors = 0
	srcCopy      = 0x00CC0020

	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2, which is -4 as a handle-shaped value.
	dpiPerMonitorAwareV2 = ^uintptr(3)
)

// The Win32 structures this backend passes across the boundary. Each is laid out to match its C
// definition exactly on both amd64 and arm64, where a pointer is 8 bytes and Go's field
// alignment agrees with the Microsoft ABI's.
type (
	wndClassExW struct {
		size       uint32
		style      uint32
		wndProc    uintptr
		clsExtra   int32
		wndExtra   int32
		instance   uintptr
		icon       uintptr
		cursor     uintptr
		background uintptr
		menuName   *uint16
		className  *uint16
		iconSm     uintptr
	}

	msg struct {
		hwnd    uintptr
		message uint32
		wParam  uintptr
		lParam  uintptr
		time    uint32
		pt      struct{ x, y int32 }
	}

	rect struct{ left, top, right, bottom int32 }

	point struct{ x, y int32 }

	// monitorInfo is MONITORINFO: a monitor's rectangle, and its work area, which is the part
	// the taskbar and any docked toolbar leave free.
	monitorInfo struct {
		size    uint32
		monitor rect
		work    rect
		flags   uint32
	}

	paintStruct struct {
		hdc         uintptr
		erase       int32
		paint       rect
		restore     int32
		incUpdate   int32
		rgbReserved [32]byte
	}

	// bitmapInfoHeader is a 32-bit BI_RGB DIB header, which is the one description of memory
	// that makes this backend a copy rather than a conversion: a BI_RGB 32bpp scan line is
	// 0x00RRGGBB per pixel as a little-endian DWORD, so its bytes are B, G, R, X -- exactly
	// platform.Framebuffer's documented layout. height is stored negative; see send.
	bitmapInfoHeader struct {
		size          uint32
		width         int32
		height        int32
		planes        uint16
		bitCount      uint16
		compression   uint32
		sizeImage     uint32
		xPelsPerMeter int32
		yPelsPerMeter int32
		clrUsed       uint32
		clrImportant  uint32
	}
)

// Window is a Win32 window plus the DIB it presents through.
type Window struct {
	hwnd uintptr
	hdc  uintptr // private and permanent, because the class is CS_OWNDC

	w, h   int // framebuffer (logical) size
	scale  int
	pw, ph int // presented (physical) size

	buf  []byte // pw*ph*4 BGRX bytes, the scaled surface StretchDIBits reads
	bmi  bitmapInfoHeader
	sub  bitmapInfoHeader // the header of the band being sent; see send
	band []byte           // a narrow band's pixels, copied out of buf; see send

	changes platform.Changes // what the window was last sent
	sweep   int              // the next row Present sends whether it changed or not

	events  []platform.Event
	pending int // index into events of the WM_KEYDOWN a WM_CHAR belongs to, or -1
	chars   charAccum
	down    [256]bool // by platform.Key
	closed  bool
	quit    bool
}

// The window registry. A window procedure is a C function pointer with no user data of its own,
// so the HWND it is handed has to be turned back into a *Window somehow.
//
// A map guarded by a mutex, rather than the usual GWLP_USERDATA trick of storing the pointer in
// the window itself. Storing a Go pointer in memory the Go runtime cannot see and casting it back
// through uintptr is precisely the pattern unsafe.Pointer's rules forbid -- nothing would keep
// the object reachable, and go vet says so. A map keeps every live window reachable from a
// package variable, which is both correct and boring.
var (
	mu       sync.Mutex
	windows  = map[uintptr]*Window{}
	creating *Window // the window whose CreateWindowExW has not returned yet; see windowFor

	classOnce sync.Once
	dpiOnce   sync.Once
	classErr  error
	classPtr  *uint16
	instance  uintptr

	// One callback for the process, made once. syscall.NewCallback consumes a slot from a
	// fixed-size table (a few thousand) and never releases it, so making one per window would
	// be a leak with a hard ceiling.
	wndProcPtr = syscall.NewCallback(wndProc)
)

// workArea is the work area of the monitor under the mouse pointer, or of the one nearest it,
// in physical pixels. It is where New puts the window, since that is the monitor the player is
// looking at, and so it is also what Room measures.
func workArea() (rect, error) {
	dpiAware() // before any monitor is asked about, or the answer is in scaled pixels
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))) // on failure, the monitor nearest 0,0
	at := rect{pt.x, pt.y, pt.x + 1, pt.y + 1}
	mon, _, _ := procMonitorFromRect.Call(uintptr(unsafe.Pointer(&at)), monitorDefaultToNearest)
	mi := monitorInfo{size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if mon == 0 {
		return rect{}, fmt.Errorf("win32: no monitor to put a window on")
	}
	if r, _, err := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi))); r == 0 {
		return rect{}, fmt.Errorf("win32: GetMonitorInfoW failed: %v", err)
	}
	return mi.work, nil
}

// Room is the space a window may have (platform.Room): the work area of the monitor under the
// pointer, less the caption and frame this window's style adds around the game.
func Room() (platform.Room, error) {
	work, err := workArea()
	if err != nil {
		return platform.Room{}, err
	}
	var frame rect
	procAdjustWindowRect.Call(uintptr(unsafe.Pointer(&frame)), wsWindow, 0)
	return platform.Room{
		W:    int(work.right-work.left) - int(frame.right-frame.left),
		H:    int(work.bottom-work.top) - int(frame.bottom-frame.top),
		From: "the work area of the monitor under the pointer, less a title bar",
	}, nil
}

// New opens a window. The caller owns Close, and must call it, PollEvents and Present from this
// same goroutine -- see the package comment.
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
	pw, ph := w*scale, h*scale

	// Win32 delivers a window's messages to the thread that created it, and this backend has no
	// thread of its own to pump them on: PollEvents runs the pump on whichever goroutine calls
	// it. Locking here is what makes "the same goroutine" mean "the same thread".
	runtime.LockOSThread()
	unlock := runtime.UnlockOSThread

	if err := registerClass(); err != nil {
		unlock()
		return nil, err
	}

	title := cfg.Title
	if title == "" {
		title = "gliderGo"
	}
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		unlock()
		return nil, fmt.Errorf("win32: window title %q: %w", title, err)
	}

	// The style has no resize border, so the outer window is the client area plus a caption and
	// a frame of whatever thickness this Windows draws. AdjustWindowRect is what turns the
	// client size the game needs into the window size CreateWindowExW takes; computing it by
	// hand is how a backend ends up with a playfield a few pixels short and a blit that clips.
	r := rect{0, 0, int32(pw), int32(ph)}
	procAdjustWindowRect.Call(uintptr(unsafe.Pointer(&r)), wsWindow, 0)
	outerW, outerH := int(r.right-r.left), int(r.bottom-r.top)

	// In the middle of the work area, and never above or left of it, so that a window too big
	// for the monitor still has its title bar on screen to be dragged by. CW_USEDEFAULT would
	// cascade it down from the top left of the primary monitor, and a window sized to fit the
	// work area would then run off the bottom of it.
	x, y := uintptr(cwUseDefault), uintptr(cwUseDefault)
	if work, err := workArea(); err == nil {
		x = uintptr(max(int(work.left)+(int(work.right-work.left)-outerW)/2, int(work.left)))
		y = uintptr(max(int(work.top)+(int(work.bottom-work.top)-outerH)/2, int(work.top)))
	}

	win := &Window{
		w: w, h: h, scale: scale, pw: pw, ph: ph,
		buf:     make([]byte, pw*ph*4),
		band:    make([]byte, max(maxSend, pw*4)),
		pending: -1,
	}
	win.bmi = bitmapInfoHeader{
		size:   uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		width:  int32(pw),
		height: -int32(ph), // top-down; see send
		planes: 1, bitCount: 32,
		compression: biRGB,
		sizeImage:   uint32(pw * ph * 4),
	}

	// The window procedure runs before CreateWindowExW returns, so the handle has to be
	// bindable from inside it. See windowFor.
	mu.Lock()
	creating = win
	mu.Unlock()

	hwnd, _, callErr := procCreateWindowExW.Call(
		0, // no extended style
		uintptr(unsafe.Pointer(classPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		wsWindow,
		x, y,
		uintptr(outerW), uintptr(outerH),
		0, 0, instance, 0)

	mu.Lock()
	creating = nil
	if hwnd != 0 {
		win.hwnd = hwnd
		windows[hwnd] = win
	}
	mu.Unlock()

	if hwnd == 0 {
		unlock()
		return nil, fmt.Errorf("win32: CreateWindowExW failed: %v", callErr)
	}

	// One DC for the window's whole life, which is what CS_OWNDC buys: no GetDC/ReleaseDC pair
	// on the frame path, and no chance of running the system out of cache DCs at 60 Hz.
	hdc, _, callErr := procGetDC.Call(hwnd)
	if hdc == 0 {
		win.Close()
		return nil, fmt.Errorf("win32: GetDC failed: %v", callErr)
	}
	win.hdc = hdc

	procShowWindow.Call(hwnd, swShow)
	// Best effort, and allowed to fail: Windows refuses to steal the foreground for a process
	// the user did not just interact with. Without it a window launched from a shell can come
	// up behind the terminal, which for a keyboard-only game reads as "the controls do not
	// work".
	procSetForegroundWindow.Call(hwnd)
	return win, nil
}

// registerClass registers the window class once per process and makes the process DPI aware.
func registerClass() error {
	classOnce.Do(func() {
		classPtr, classErr = syscall.UTF16PtrFromString("gliderGoWindow")
		if classErr != nil {
			return
		}
		dpiAware()
		instance, _, _ = procGetModuleHandleW.Call(0)
		cursor, _, _ := procLoadCursorW.Call(0, idcArrow)

		wc := wndClassExW{
			// CS_OWNDC for the permanent DC. The two redraw flags invalidate the whole
			// window on a size change, which cannot happen with this style but costs
			// nothing to be right about.
			style:    csHRedraw | csVRedraw | csOwnDC,
			wndProc:  wndProcPtr,
			instance: instance,
			cursor:   cursor,
			// No background brush, deliberately. Windows would otherwise paint the whole
			// window with it before every WM_PAINT, which is a flash of grey between
			// frames; WM_ERASEBKGND is claimed in wndProc for the same reason. Every
			// pixel of this window comes from the game.
			background: 0,
			className:  classPtr,
		}
		wc.size = uint32(unsafe.Sizeof(wc))

		if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
			classErr = fmt.Errorf("win32: RegisterClassExW failed: %v", err)
		}
	})
	return classErr
}

// dpiAware tells Windows not to scale this process's windows, and it matters more here than the
// one line of code suggests.
//
// A DPI-unaware process on a 200% display does not get a 640x480 window: it gets a 320x240 one
// that Windows stretches to 640x480 with a bilinear filter. Every pixel of 1994 art would arrive
// on screen blurred, by a step this port has no say in and cannot compensate for -- which is the
// exact opposite of what -scale does, and of what the whole render path is for.
//
// Both entry points are optional and neither is checked. SetProcessDpiAwarenessContext is
// Windows 10 1703 and later; SetProcessDPIAware is Vista and later but is the older, per-process
// form. If the awareness is already set (by a manifest, say, or by a launcher) both fail
// harmlessly and the setting that is already there is the right one to keep.
//
// It runs once, before the first monitor query or the window class, whichever comes first:
// Room asks about the monitor before New registers anything, and a process that is not yet DPI
// aware is told the size of a monitor in scaled pixels.
func dpiAware() {
	dpiOnce.Do(func() {
		if procSetProcessDpiAwarenessContext.Find() == nil {
			if r, _, _ := procSetProcessDpiAwarenessContext.Call(dpiPerMonitorAwareV2); r != 0 {
				return
			}
		}
		if procSetProcessDPIAware.Find() == nil {
			procSetProcessDPIAware.Call()
		}
	})
}

// windowFor maps an HWND back to its Window.
//
// The adoption branch is the awkward part of Win32 and not an optimisation: a window receives
// WM_NCCREATE, WM_CREATE and often WM_SIZE from *inside* CreateWindowExW, before that call has
// returned the handle there would be anything to key a map on. So New parks the half-built
// Window in `creating` and the first message binds it. Passing the pointer through
// CREATESTRUCT's lpCreateParams is the other way, and it is the same unsafe.Pointer-through-a-
// uintptr problem the registry comment describes, for no gain.
func windowFor(hwnd uintptr) *Window {
	mu.Lock()
	defer mu.Unlock()
	if w, ok := windows[hwnd]; ok {
		return w
	}
	if creating != nil {
		w := creating
		w.hwnd = hwnd
		windows[hwnd] = w
		creating = nil
		return w
	}
	return nil
}

// wndProc is the window procedure. It runs on the thread that dispatched the message, which is
// the thread PollEvents was called from, so it appends to w.events with no synchronisation and
// none is needed.
func wndProc(hwnd, message, wparam, lparam uintptr) uintptr {
	w := windowFor(hwnd)
	if w == nil {
		// A message for a window this package does not own, or for one already closed.
		r, _, _ := procDefWindowProcW.Call(hwnd, message, wparam, lparam)
		return r
	}

	switch message {
	case wmClose:
		// Report it and do nothing else. Destroying the window here would leave the game
		// drawing into a dead HWND for the rest of the frame; the shell sees EventQuit,
		// finishes what it is doing and calls Close, which is the same path the Quit menu
		// item takes. This mirrors the x11 backend's WM_DELETE_WINDOW handling exactly.
		w.quit = true
		w.events = append(w.events, platform.Event{Kind: platform.EventQuit})
		return 0

	case wmDestroy:
		w.closed = true
		return 0

	case wmEraseBkgnd:
		// Claimed, so Windows does not paint over the frame that is already there. Returning
		// nonzero means "erased" -- and it has been, by the game.
		return 1

	case wmPaint:
		// BeginPaint/EndPaint is not optional: they are what clears the update region, and
		// without them Windows would send WM_PAINT again immediately, forever. Between them
		// the damaged part is put back from buf, which holds the frame the window was sent
		// last, so the repair does not wait on the game. All and EventExpose as well, so the
		// next Present sends the whole of the current frame and a paused game redraws; a paint
		// that arrives before the first Present puts back black, not whatever was there.
		var ps paintStruct
		if dc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps))); dc != 0 {
			x0, y0 := max(int(ps.paint.left), 0), max(int(ps.paint.top), 0)
			x1, y1 := min(int(ps.paint.right), w.pw), min(int(ps.paint.bottom), w.ph)
			if x1 > x0 && y1 > y0 {
				w.send(dc, x0, y0, x1-x0, y1-y0)
			}
		}
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		w.changes.All()
		w.events = append(w.events, platform.Event{Kind: platform.EventExpose})
		return 0

	case wmSetFocus:
		w.events = append(w.events, platform.Event{Kind: platform.EventFocus, Focused: true})
		return 0

	case wmKillFocus:
		// Losing focus must clear the held keys or the glider flies off on its own when the
		// player alt-tabs away mid-press: Windows sends no WM_KEYUP for a key released while
		// another window has the focus, so the bitmap would say "still held" forever.
		w.down = [256]bool{}
		w.events = append(w.events, platform.Event{Kind: platform.EventFocus, Focused: false})
		return 0

	case wmSize:
		// The style forbids resizing, so this is the initial size and the minimise/restore
		// pair. Reported for the same reason x11 reports ConfigureNotify: the framebuffer is
		// unaffected either way (platform.EventResize).
		w.events = append(w.events, platform.Event{
			Kind: platform.EventResize,
			W:    int(lparam & 0xFFFF), H: int((lparam >> 16) & 0xFFFF),
		})
		return 0

	case wmKeyDown, wmSysKeyDown, wmKeyUp, wmSysKeyUp:
		w.key(message == wmKeyDown || message == wmSysKeyDown, wparam, lparam)
		if message == wmKeyDown || message == wmKeyUp {
			return 0
		}
		// The Sys pair is Alt and F10, and it goes on to DefWindowProc afterwards rather than
		// being swallowed: that is what keeps Alt-F4 closing the window and Alt-Space opening
		// the system menu. The game sees the key as well, which is right -- Alt is a
		// bindable key like any other here.
		r, _, _ := procDefWindowProcW.Call(hwnd, message, wparam, lparam)
		return r

	case wmChar:
		w.char(uint16(wparam))
		return 0
	}

	// WM_SYSCHAR is deliberately not handled: it is what Alt-key chords produce, and a chord
	// types nothing into a high-score name. Letting it reach DefWindowProc keeps the menu
	// behaviour Windows users expect.
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wparam, lparam)
	return r
}

// key records one key transition and queues the event.
func (w *Window) key(isDown bool, wparam, lparam uintptr) {
	k := keyFor(wparam)

	// Bit 30 of lParam is the key's previous state, so a set bit on a press means auto-repeat.
	// Windows reports this directly. The x11 backend infers it from its own record -- a press
	// while the same keycode is already down -- and that works only because x11.New asks for
	// XKB's detectable auto-repeat: by default X sends each repeat as a release and then a
	// press, and the inference never fires (docs/IMPROVEMENTS.md 2.72). The bit is still the
	// better answer, because it stays correct across a lost WM_KEYUP and a record does not.
	repeat := isDown && lparam&0x40000000 != 0

	kind := platform.EventKeyUp
	if isDown {
		kind = platform.EventKeyDown
	}
	if k != platform.KeyUnknown {
		w.down[k] = isDown
	}
	w.events = append(w.events, platform.Event{Kind: kind, Key: k, Repeat: repeat})

	// Where the text of this press will go when its WM_CHAR arrives, which it does as a
	// separate message a moment later -- TranslateMessage in PollEvents is what generates it,
	// and it lands in the same drain. A release types nothing, so it clears the slot.
	if isDown {
		w.pending = len(w.events) - 1
	} else {
		w.pending = -1
	}
}

// char attaches what a press typed to the event that reported it.
func (w *Window) char(unit uint16) {
	text := w.chars.add(unit)
	if text == "" {
		return
	}
	if w.pending >= 0 && w.pending < len(w.events) {
		w.events[w.pending].Text += text
		return
	}
	// A character with no key press of ours in front of it: an input method committing text, or
	// a WM_CHAR posted directly. It still has to reach the high-score field, so it goes out as
	// a press of no key at all -- which is what the x11 backend does with a keysym it has no
	// name for, and KeyUnknown matches no binding.
	w.events = append(w.events, platform.Event{Kind: platform.EventKeyDown, Text: text})
}

// Present displays fb, magnifying by the configured integer scale with nearest-neighbour
// sampling. It sends what changed since the last frame, block by block as platform.Changes
// reports it and as the x11 backend does, and then one band of the window whether it changed or
// not.
//
// That band is the sweep, and it is there because a send can go missing without a word. On one
// player's Windows 10 laptop the whole-frame sends v0.2.0 made -- 4.9 MB each at 2x -- by all the
// signs never reached the screen, while the smaller ones for a moving glider or a menu's highlight
// did: the window came up white and filled in only where something moved (docs/IMPROVEMENTS.md
// 2.81). What StretchDIBits returned went unread, and nothing would have sent those pixels again.
// Now nothing goes in a piece bigger than maxSend, and the sweep works down the window a 64th of it
// a frame, so whatever is lost regardless is sent again within 64 frames: about 1.5 s in the
// menus, two in play. On the one Windows machine this has been timed on, that costs paced play
// nothing measurable and the unpaced bench a tenth at 2x and a fifth at 4x, and a sideways wipe
// between rooms, 160 Presents in one frame with a band each, 237 to 278 ms rather than 205 to 215.
//
// A failed send is not an error either. A Present error ends the game (cmd/glidergo's Present
// hook treats one as a dead window), and GDI refusing a frame drawn while the window is
// mid-transition would be a game that quits itself. It is noted instead, and the next Present
// sends the whole frame again.
func (w *Window) Present(fb *platform.Framebuffer) error {
	if w.hwnd == 0 {
		return fmt.Errorf("win32: the window is closed")
	}
	if fb.W != w.w || fb.H != w.h {
		return fmt.Errorf("win32: framebuffer is %dx%d, window expects %dx%d", fb.W, fb.H, w.w, w.h)
	}
	if r, _, _ := procIsIconic.Call(w.hwnd); r != 0 {
		// Minimised: nothing shows, and what showed is gone when the window is restored.
		w.changes.All()
		return nil
	}

	s := w.scale
	for _, sp := range w.changes.Diff(fb) {
		if err := platform.ExpandSpan(w.buf, w.pw*4, fb, s, sp); err != nil {
			return err
		}
		if !w.send(w.hdc, sp.X0*s, sp.Y0*s, (sp.X1-sp.X0)*s, (sp.Y1-sp.Y0)*s) {
			w.changes.All()
		}
	}

	rows := min((w.ph+63)/64, w.ph-w.sweep)
	if !w.send(w.hdc, 0, w.sweep, w.pw, rows) {
		w.changes.All()
	}
	if w.sweep += rows; w.sweep >= w.ph {
		w.sweep = 0
	}
	return nil
}

// maxSend is the most one StretchDIBits call is given: 64 KiB of DIB, which is 12 whole rows of
// the window at 2x and 6 at 4x, or a wipe's column whole at 2x and in two at 3x and 4x. The
// largest send the laptop in Present's comment is known to have drawn is v0.2.0's for a menu
// highlight, a DIB of 38 whole rows at 2x and 190 KiB, and the only ones known lost there are
// whole frames. The cap is a third of the first. What it costs is calls: a whole frame at 2x is
// 80 of them rather than one, 3 ms against 1.6 on the one Windows machine this has been timed on.
const maxSend = 64 << 10

// send puts the block of buf with its top left at x, y, cw by ch pixels, into the window through
// dc, a band of at most maxSend bytes at a time. It reports whether GDI took every band.
//
// Each band is sent as a DIB of its own, and the source rectangle is all of it. A band as wide as
// the window is buf's own rows. A narrower one is copied into band first, so that its DIB is as
// wide as the block: a DIB of buf's rows would be whole rows of the window, and a wipe's column 4
// pixels wide would be a call for every few rows of it. The source rectangle being the whole DIB
// is deliberate. Which way StretchDIBits counts ySrc in a top-down DIB is something Windows' own
// documentation and its drivers have disagreed on, and a source rectangle that is the whole of its
// DIB is the same pixels whichever way it is counted.
//
// The DIB header's height is negative, which is what makes this a copy of the framebuffer rather
// than a vertical flip of it: a positive height means a bottom-up DIB, the layout every Windows
// bitmap file uses and no framebuffer in this port does. StretchDIBits with equal source and
// destination rectangles is a straight blit -- the scaling has already happened, in Go, because
// GDI's own stretch would smooth 1994 pixel art (see platform.Expand).
func (w *Window) send(dc uintptr, x, y, cw, ch int) bool {
	stride, ok := cw*4, true
	rows := max(maxSend/stride, 1)
	for top := y; top < y+ch; top += rows {
		n := min(rows, y+ch-top)
		bits := w.band
		if cw == w.pw {
			bits = w.buf[top*stride:]
		} else {
			for i := range n {
				at := ((top+i)*w.pw + x) * 4
				copy(bits[i*stride:(i+1)*stride], w.buf[at:at+stride])
			}
		}
		w.sub = w.bmi
		w.sub.width = int32(cw)
		w.sub.height = -int32(n)
		w.sub.sizeImage = uint32(stride * n)

		// Both pointers are into fields of a live *Window, so nothing here can be collected
		// while the call runs -- which is the rule that makes passing Go memory to a syscall
		// safe, and the reason buf, band and sub are fields rather than locals.
		r, _, _ := procStretchDIBits.Call(dc,
			uintptr(x), uintptr(top), uintptr(cw), uintptr(n), // destination rectangle
			0, 0, uintptr(cw), uintptr(n), // source: all of the band's own DIB
			uintptr(unsafe.Pointer(&bits[0])),
			uintptr(unsafe.Pointer(&w.sub)),
			dibRGBColors, srcCopy)

		// The rows copied, as a C int, so only the low 32 bits mean anything. Zero is a refusal,
		// or nothing copied, and either way the band may not be on the screen. GDI_ERROR is for
		// JPEG and PNG DIBs, and a negative count is a success on a mirrored DC.
		if int32(r) == 0 {
			ok = false
		}
	}
	return ok
}

// PollEvents runs the message pump until the queue is empty and returns what it produced.
//
// PeekMessageW rather than GetMessageW because this must never block: the game loop calls it once
// a frame and then goes on to simulate one, whether anything was typed or not. The hwnd filter is
// zero, which takes messages for every window on this thread plus thread messages like WM_QUIT --
// a filtered pump is how a program ends up unable to be quit.
func (w *Window) PollEvents() []platform.Event {
	// Fresh each call rather than a reused buffer, so a caller may hold on to the slice.
	w.events = nil
	w.pending = -1

	var m msg
	for {
		r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove)
		if r == 0 {
			break
		}
		if m.message == wmQuit {
			// Nothing here posts WM_QUIT, so this is Windows or a launcher asking the
			// process to end. It never reaches a window procedure, hence the special case.
			w.quit = true
			w.events = append(w.events, platform.Event{Kind: platform.EventQuit})
			continue
		}
		// TranslateMessage is what turns a WM_KEYDOWN into the WM_CHAR that carries the
		// character the player's own layout, modifiers and dead keys produced. It posts that
		// message to this queue, so the loop above picks it up before it finishes -- which is
		// why w.pending can be an index into a slice that is still being built.
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	// Drop the presses and releases of keys this port has no name for that typed nothing
	// either: a key event with no Key matches no binding and a text event with no Text says
	// nothing, so passing them on would only make every consumer test for them. The x11
	// backend drops the same ones at the same point, for the same reason.
	kept := w.events[:0]
	for _, ev := range w.events {
		if ev.Key == platform.KeyUnknown && ev.Text == "" &&
			(ev.Kind == platform.EventKeyDown || ev.Kind == platform.EventKeyUp) {
			continue
		}
		kept = append(kept, ev)
	}
	w.events = nil
	return kept
}

// KeyDown reports whether k is held right now. The original polls the keyboard rather than
// consuming events (GetKeys) and the physics depends on it, so this bitmap is the backend's real
// output and the event queue is the sideline.
func (w *Window) KeyDown(k platform.Key) bool {
	if k <= 0 || int(k) >= len(w.down) {
		return false
	}
	return w.down[k]
}

// SetTitle sets the window caption.
func (w *Window) SetTitle(s string) error {
	if s == "" {
		s = "gliderGo"
	}
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return fmt.Errorf("win32: window title %q: %w", s, err)
	}
	if w.hwnd == 0 {
		return fmt.Errorf("win32: the window is closed")
	}
	procSetWindowTextW.Call(w.hwnd, uintptr(unsafe.Pointer(p)))
	return nil
}

// Closed reports whether the window has been destroyed or asked to quit.
func (w *Window) Closed() bool { return w.closed || w.quit }

// Close destroys the window and releases the thread New locked. It is safe to call twice, which
// matters because New calls it on its own failure path.
func (w *Window) Close() error {
	if w.hwnd == 0 {
		return nil
	}
	if w.hdc != 0 {
		procReleaseDC.Call(w.hwnd, w.hdc)
		w.hdc = 0
	}
	hwnd := w.hwnd
	w.hwnd = 0

	// DestroyWindow sends WM_DESTROY synchronously, so the registry entry has to outlive the
	// call: wndProc looks the window up to record that it is closed. Delete afterwards, and
	// hold no lock across the call.
	procDestroyWindow.Call(hwnd)

	mu.Lock()
	delete(windows, hwnd)
	mu.Unlock()

	runtime.UnlockOSThread()
	return nil
}
