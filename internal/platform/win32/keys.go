package win32

// The two pure pieces of the Windows backend: virtual-key codes in, platform keys out, and
// WM_CHAR code units in, text out.
//
// This file carries no build tag, and that is deliberate rather than an oversight. Everything
// else in the package needs Windows to compile at all, so on the machine this port was written
// on -- Linux, airgapped, no Windows toolchain and no Windows to run one on -- none of it can be
// executed even once. What can be executed is the arithmetic and the two tables, which is where
// the mistakes actually live: an off-by-one in the letter range, a VK left out so a key silently
// stops working, a surrogate pair turned into two broken runes in a player's name. Splitting
// them out means `go test ./internal/platform/win32` runs on any host and covers them, and
// keys_test.go holds the VK table against platform's own key list so a key added to the enum
// cannot quietly go unmapped on Windows.
//
// Window creation, the message pump and the blit live in win32.go, and those have since been run on
// a real Windows machine and checked against Linux pixel for pixel. This file's own subject has
// not: that run was driven by -frames, so no key was ever pressed, and these two tables remain the
// least-exercised code in the package on the platform they exist for. See win32.go's note.

import "github.com/bwenstar/gliderGo/internal/platform"

// Virtual-key codes, from WinUser.h. Only the ones this port binds or needs are named; the OEM_n
// spellings are Microsoft's own and are kept so the table can be read against the header.
const (
	vkBack    = 0x08
	vkTab     = 0x09
	vkReturn  = 0x0D
	vkShift   = 0x10
	vkControl = 0x11
	vkMenu    = 0x12 // Alt, which Windows calls Menu
	vkEscape  = 0x1B
	vkSpace   = 0x20
	vkLeft    = 0x25
	vkUp      = 0x26
	vkRight   = 0x27
	vkDown    = 0x28
	vkDelete  = 0x2E

	vk0 = 0x30
	vk9 = 0x39
	vkA = 0x41
	vkZ = 0x5A

	vkLWin = 0x5B
	vkRWin = 0x5C

	vkNumpad0 = 0x60
	vkNumpad9 = 0x69

	vkF1  = 0x70
	vkF12 = 0x7B

	vkLShift    = 0xA0
	vkRShift    = 0xA1
	vkLControl  = 0xA2
	vkRControl  = 0xA3
	vkLMenu     = 0xA4
	vkRMenu     = 0xA5
	vkOEM1      = 0xBA // ;: on a US keyboard
	vkOEMPlus   = 0xBB // =+
	vkOEMComma  = 0xBC // ,<
	vkOEMMinus  = 0xBD // -_
	vkOEMPeriod = 0xBE // .>
	vkOEM2      = 0xBF // /?
	vkOEM3      = 0xC0 // `~
	vkOEM4      = 0xDB // [{
	vkOEM5      = 0xDC // \|
	vkOEM6      = 0xDD // ]}
	vkOEM7      = 0xDE // '"
)

// vkKeys is the whole translation, indexed by virtual-key code.
//
// An array and not a map: wParam is a byte-ranged value on every message, the table is 256
// entries, and this is on the path of every key event. It is built once at init.
//
// The OEM keys are worth a word, because they are the one place this differs in character from
// the X11 table it mirrors. A VK_OEM_n code identifies a *position* on the keyboard, and which
// character that position produces depends on the layout -- VK_OEM_1 is `;` on a US keyboard and
// `$` on a French one. That is exactly right here and is the same choice the X11 backend makes by
// taking keysym index 0: the game binds physical keys, so a binding must not move when the
// player switches layout. The character a key produces is Event.Text's job, and that comes from
// WM_CHAR, which is layout-aware. See platform.Event.
//
// The seven modifiers are mapped both as the generic VK (VK_SHIFT) and as the two sided ones
// (VK_LSHIFT, VK_RSHIFT). Windows delivers the sided codes in WM_KEYDOWN's wParam on modern
// systems and the generic ones through other paths; both are handled rather than one being
// assumed, which costs three table entries.
var vkKeys = func() [256]platform.Key {
	var t [256]platform.Key
	for vk, k := range map[int]platform.Key{
		vkLeft: platform.KeyLeft, vkRight: platform.KeyRight,
		vkUp: platform.KeyUp, vkDown: platform.KeyDown,
		vkSpace: platform.KeySpace, vkReturn: platform.KeyReturn,
		vkEscape: platform.KeyEscape, vkTab: platform.KeyTab,

		// Both, as the X11 backend does: Delete is the original's key for giving up a glider
		// stuck in limbo, and Backspace is what a player reaches for to correct a high-score
		// name. The text field wants one and the game wants the other, and neither cares
		// which physical key it came from.
		vkBack: platform.KeyDelete, vkDelete: platform.KeyDelete,

		vkOEMMinus: platform.KeyMinus, vkOEMPlus: platform.KeyEqual,
		vkOEM4: platform.KeyLeftBracket, vkOEM6: platform.KeyRightBracket,
		vkOEMComma: platform.KeyComma, vkOEMPeriod: platform.KeyPeriod,
		vkOEM2: platform.KeySlash, vkOEM1: platform.KeySemicolon,
		vkOEM7: platform.KeyQuote, vkOEM5: platform.KeyBackslash,
		vkOEM3: platform.KeyGrave,

		vkShift: platform.KeyShift, vkLShift: platform.KeyShift, vkRShift: platform.KeyShift,
		vkControl: platform.KeyControl, vkLControl: platform.KeyControl, vkRControl: platform.KeyControl,
		vkMenu: platform.KeyAlt, vkLMenu: platform.KeyAlt, vkRMenu: platform.KeyAlt,
		vkLWin: platform.KeySuper, vkRWin: platform.KeySuper,
	} {
		t[vk] = k
	}
	for i := 0; i <= vkZ-vkA; i++ {
		t[vkA+i] = platform.KeyA + platform.Key(i)
	}
	for i := 0; i <= vk9-vk0; i++ {
		t[vk0+i] = platform.Key0 + platform.Key(i)
		t[vkNumpad0+i] = platform.Key0 + platform.Key(i)
	}
	for i := 0; i <= vkF12-vkF1; i++ {
		t[vkF1+i] = platform.KeyF1 + platform.Key(i)
	}
	return t
}()

// keyFor translates one virtual-key code. Anything out of range or unmapped is KeyUnknown,
// which no binding matches -- and which still reaches a text field if the press typed something,
// the way an accented key on a layout this table has never heard of must.
func keyFor(wparam uintptr) platform.Key {
	if wparam >= uintptr(len(vkKeys)) {
		return platform.KeyUnknown
	}
	return vkKeys[wparam]
}

// charAccum turns the WM_CHAR stream into text.
//
// Two things make this more than a cast. WM_CHAR carries one UTF-16 code unit, so a character
// outside the Basic Multilingual Plane arrives as two messages that have to be recombined --
// pass them through separately and a text field gets two replacement characters instead of one
// emoji. And the control codes have to go: Return, Escape, Tab and Backspace all produce a
// WM_CHAR (0x0D, 0x1B, 0x09, 0x08) *as well as* the WM_KEYDOWN that already reported them as a
// Key, so a field that appended Text blindly would put a carriage return inside a player's name.
// The 0x7F..0x9F band goes for the same reason: DEL and the C1 controls are not characters.
//
// This mirrors the X11 backend's eventText, which drops the same two ranges from XLookupString's
// Latin-1 bytes. The two backends agreeing on what "typed nothing" means is what keeps the
// high-score screen behaving identically on both.
type charAccum struct{ high uint16 }

// add reports what this code unit typed: a one-rune string, or "" for a code unit that typed
// nothing on its own -- a high surrogate awaiting its pair, a control code, or an orphaned low
// surrogate.
func (c *charAccum) add(u uint16) string {
	switch {
	case u >= 0xD800 && u <= 0xDBFF:
		// A high surrogate is half a character. Hold it; the low half is the next message.
		c.high = u
		return ""
	case u >= 0xDC00 && u <= 0xDFFF:
		if c.high == 0 {
			// A low surrogate with nothing before it is a broken stream, not a character.
			return ""
		}
		r := 0x10000 + (rune(c.high-0xD800) << 10) + rune(u-0xDC00)
		c.high = 0
		return string(r)
	}
	// Anything else ends a pending pair rather than completing it: the high surrogate was
	// orphaned, and dropping it is better than emitting the replacement character into a name.
	c.high = 0
	r := rune(u)
	if r < 0x20 || (r >= 0x7F && r < 0xA0) {
		return ""
	}
	return string(r)
}
