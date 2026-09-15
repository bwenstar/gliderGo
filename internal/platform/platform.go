// Package platform is gliderGo's port layer: the complete boundary between the
// game and the machine it runs on.
//
// It exists because this project cannot use a Go game engine. The development
// network has no Go module proxy and no reachable GitHub (see
// docs/DEV_ENVIRONMENT.md §3), so Ebitengine, go-sdl2 and friends are simply not
// obtainable. Everything here is therefore standard library plus, per backend, a
// small hand-written cgo or syscall shim.
//
// That constraint turns out to suit the subject. Glider PRO (1994) is a
// fixed-resolution 2D blitter: no scaling, no rotation, no alpha compositing,
// one 640x480 indexed-colour frame at a time. A software Framebuffer that the
// backend uploads once per frame is not a compromise, it is an exact model of
// the original.
//
// Backends live in subpackages and are selected at build time by file-level
// build tags, never by runtime probing:
//
//	x11    cgo + Xlib + XPutImage      Linux (measured 533 fps at 640x480x32)
//	win32  pure Go syscall to gdi32    Windows, no cgo and no dependencies
//	sdl2   hand-written cgo binding    macOS/iOS/Android escape hatch
//	null   headless, writes PNG/WAV    tests, CI, frame-diff fidelity checks
//
// The interfaces are deliberately narrow: a backend is a few hundred lines, and
// adding one must never require touching game code.
package platform

import "image/color"

// Screen geometry of the original game. These are not preferences; the original
// composes at exactly this size and the port reproduces it pixel for pixel,
// leaving window scaling to the backend. See docs/analysis/rendering.md.
const (
	ScreenWidth  = 640
	ScreenHeight = 480
)

// Framebuffer is a 32-bit BGRX software surface, which is the layout X11, GDI
// and SDL all accept without conversion on little-endian hosts. The original's
// 8-bit indexed pixels are expanded through its 'clut' palette during asset
// extraction, so the runtime never carries an indexed path.
//
// Pix is row-major with Stride bytes per row; Stride may exceed 4*W when a
// backend needs alignment padding. Byte order within a pixel is B,G,R,X.
type Framebuffer struct {
	Pix    []byte
	Stride int
	W, H   int
}

// NewFramebuffer allocates a tightly packed framebuffer.
func NewFramebuffer(w, h int) *Framebuffer {
	return &Framebuffer{Pix: make([]byte, w*h*4), Stride: w * 4, W: w, H: h}
}

// Set writes one pixel. Game code blits spans rather than calling this per
// pixel; it exists for tests and for the few places the original draws single
// dots.
func (f *Framebuffer) Set(x, y int, c color.RGBA) {
	if x < 0 || y < 0 || x >= f.W || y >= f.H {
		return
	}
	o := y*f.Stride + x*4
	f.Pix[o+0] = c.B
	f.Pix[o+1] = c.G
	f.Pix[o+2] = c.R
	f.Pix[o+3] = 0xff
}

// Fill paints the whole surface one colour.
func (f *Framebuffer) Fill(c color.RGBA) {
	if f.H == 0 {
		return
	}
	row := f.Pix[:f.Stride]
	for x := 0; x+4 <= f.Stride; x += 4 {
		row[x+0], row[x+1], row[x+2], row[x+3] = c.B, c.G, c.R, 0xff
	}
	for y := 1; y < f.H; y++ {
		copy(f.Pix[y*f.Stride:(y+1)*f.Stride], row)
	}
}

// Key is a physical key identity, independent of layout and of the host's
// keycode numbering. Backends translate to these; the game only ever sees them.
// The set covers what Glider PRO binds (see docs/analysis/input.md) plus what a
// modern shell needs.
type Key int

const (
	KeyUnknown Key = iota
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeySpace
	KeyReturn
	KeyEscape
	KeyTab
	KeyDelete
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ
	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeyMinus
	KeyEqual
	KeyLeftBracket
	KeyRightBracket
	KeyComma
	KeyPeriod
	KeySlash
	KeySemicolon
	KeyQuote
	KeyBackslash
	KeyGrave
	KeyShift
	KeyControl
	KeyAlt
	KeySuper
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	numKeys
)

// EventKind discriminates Event.
type EventKind int

const (
	EventNone EventKind = iota
	EventKeyDown
	EventKeyUp
	EventQuit   // window closed or platform asked us to exit
	EventFocus  // Focused reports whether we gained or lost focus
	EventResize // backend surface changed; Framebuffer size is unaffected

	// EventExpose says the window's contents were damaged and have to be drawn
	// again. It is the original's update event (Play.c's updateEvt arm calls
	// RefreshGameWindow) and it matters for exactly the reason it did in 1994: a
	// paused game draws no frames, so if something obscures the window while it is
	// paused, nothing else will ever repaint it. A backend that composites and
	// retains window contents may never send one; a backend that does not, will.
	EventExpose
)

// Event is one input or window notification.
type Event struct {
	Kind    EventKind
	Key     Key
	Repeat  bool
	Focused bool
	W, H    int

	// Text is what this key press *typed*, if anything: one rune for an ordinary
	// character, empty for a key that types nothing. It is set on EventKeyDown only,
	// including auto-repeat, since a held key types repeatedly.
	//
	// Key and Text answer two different questions and the game needs both. Key is the
	// physical key, layout-independent, because that is what the original binds: the
	// glider's controls come out of GetKeys, so "the key left of X" has to stay the key
	// left of X whatever it is engraved with. Text is the character the host's own
	// layout, modifiers and compose state produced, because the high-score screen asks
	// the player to type their name (docs/analysis/scoring.md 7.11) and a name typed
	// through a physical-key table would be mojibake on any layout but the one the
	// table was written for.
	//
	// A backend that cannot report text leaves this empty; KeyChar is the US-layout
	// fallback for that case, and the caller decides which it prefers.
	Text string
}

// KeyChar is what a key types on a US layout: the fallback for a backend that reports
// keys but not text.
//
// It is deliberately not the whole story and is not meant to be. Text entry belongs to
// the host, which knows the player's layout, their dead keys and their input method;
// this knows one arrangement of one keyboard. It exists so that a backend with no text
// support at all -- and the scripted null backend, which has no keyboard -- can still
// reach the one screen in this game that needs typing.
//
// The letters and digits are the enum's own ranges, the way keys.go names them; the
// punctuation is a table, and TestKeyCharAgreesWithTheKeyNames holds it to the engravings
// keys.go already lists as aliases, so the two descriptions of a US keyboard cannot drift
// apart. Shift is written out rather than computed, because "the character above" is not
// arithmetic.
func KeyChar(k Key, shift bool) (rune, bool) {
	var r rune
	switch {
	case k == KeySpace:
		return ' ', true // shifted or not, and the one key where that is worth saying
	case k >= KeyA && k <= KeyZ:
		r = 'a' + rune(k-KeyA)
	case k >= Key0 && k <= Key9:
		r = '0' + rune(k-Key0)
	default:
		var ok bool
		if r, ok = usUnshifted[k]; !ok {
			return 0, false
		}
	}
	if !shift {
		return r, true
	}
	if r >= 'a' && r <= 'z' {
		return r - 'a' + 'A', true
	}
	if up, ok := usShifted[r]; ok {
		return up, true
	}
	return r, true
}

// usUnshifted is the punctuation a US keyboard engraves on the keys that are not letters,
// digits or space.
var usUnshifted = map[Key]rune{
	KeyMinus: '-', KeyEqual: '=', KeyLeftBracket: '[', KeyRightBracket: ']',
	KeyComma: ',', KeyPeriod: '.', KeySlash: '/', KeySemicolon: ';',
	KeyQuote: '\'', KeyBackslash: '\\', KeyGrave: '`',
}

// usShifted is the same keys plus the digits, shifted.
var usShifted = map[rune]rune{
	'1': '!', '2': '@', '3': '#', '4': '$', '5': '%',
	'6': '^', '7': '&', '8': '*', '9': '(', '0': ')',
	'-': '_', '=': '+', '[': '{', ']': '}', '\\': '|',
	';': ':', '\'': '"', ',': '<', '.': '>', '/': '?', '`': '~',
}

// Window is a host surface the game can present frames to.
//
// The contract is intentionally frame-at-a-time: Present uploads the whole
// framebuffer. The original game does dirty-rectangle updates internally
// (docs/analysis/rendering.md) for reasons that no longer apply -- at 533 fps
// measured for full-screen uploads, correctness beats cleverness here.
type Window interface {
	// Present uploads and displays the framebuffer.
	Present(fb *Framebuffer) error
	// PollEvents drains pending host events. It never blocks.
	PollEvents() []Event
	// KeyDown reports whether a key is currently held. The original polls the
	// keyboard state directly (GetKeys) rather than consuming key events, and
	// the physics depends on that, so backends must maintain this bitmap.
	KeyDown(k Key) bool
	// SetTitle updates the window caption.
	SetTitle(s string) error
	// Close releases host resources.
	Close() error
}

// Config describes the window a backend should create.
type Config struct {
	Title  string
	Width  int // framebuffer width; 0 means ScreenWidth
	Height int // framebuffer height; 0 means ScreenHeight
	Scale  int // integer magnification of the presented image; 0/1 means 1:1
}

// AudioSink accepts interleaved 16-bit signed PCM at the mixer's sample rate.
// The original's sounds are Sound Manager 'snd ' resources decoded to PCM at
// extraction time (docs/analysis/audio.md), so playback is a pure mixing
// problem with no format handling at runtime.
//
// A sink is also allowed to be a file: the development host has no audio
// hardware at all (no /dev/snd), so the WAV-writing sink is how audio
// correctness gets verified here.
type AudioSink interface {
	// SampleRate reports the rate the sink consumes.
	SampleRate() int
	// Channels reports 1 for mono, 2 for interleaved stereo.
	Channels() int
	// Write queues samples. It must not block for longer than one buffer.
	Write(samples []int16) error
	// Close flushes and releases the device.
	Close() error
}
