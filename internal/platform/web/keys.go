package web

// The pure half of the browser backend: what a KeyboardEvent means to the game, and how large the
// canvas is drawn.
//
// This file carries no build tag, for win32/keys.go's reason. web.go needs syscall/js, which
// compiles only for GOOS=js, and `go test` cannot run a js/wasm test binary on a machine with no
// JavaScript engine to run it in -- which is the machine this port is written on. What can run
// anywhere is the translation and the arithmetic, and that is where the mistakes would be: a key
// left out of the table, a key reported on the way down and lost on the way up, a high-score name
// with a control code in it. keys_test.go holds the table against platform's own key list, so a
// key added to the enum cannot quietly go unmapped in a browser. web.go is the rest: listeners
// that read a few fields off each event and hand them to keyboard.event.

import (
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/bwenstar/gliderGo/internal/platform"
)

// codes is KeyboardEvent.code to platform.Key.
//
// A code names a key by where it is. The names are what a US keyboard engraves there and they do
// not change with the layout: "Semicolon" is the key right of L whether it types ; or m or ñ.
// That is the choice win32/keys.go makes for the OEM keys and x11 makes by taking keysym index 0,
// for the same reason: the game binds keys, and a binding must not move when the player switches
// layout. The letters are the exception, and keyOf has them.
var codes = func() map[string]platform.Key {
	m := map[string]platform.Key{
		"ArrowLeft": platform.KeyLeft, "ArrowRight": platform.KeyRight,
		"ArrowUp": platform.KeyUp, "ArrowDown": platform.KeyDown,
		"Space": platform.KeySpace, "Enter": platform.KeyReturn, "NumpadEnter": platform.KeyReturn,
		"Escape": platform.KeyEscape, "Tab": platform.KeyTab,

		// Both, as on X11 and Windows: Delete gives up a glider waiting in limbo, and Backspace
		// is what a player reaches for to correct a high-score name.
		"Backspace": platform.KeyDelete, "Delete": platform.KeyDelete,

		"Minus": platform.KeyMinus, "Equal": platform.KeyEqual,
		"BracketLeft": platform.KeyLeftBracket, "BracketRight": platform.KeyRightBracket,
		"Comma": platform.KeyComma, "Period": platform.KeyPeriod, "Slash": platform.KeySlash,
		"Semicolon": platform.KeySemicolon, "Quote": platform.KeyQuote,
		"Backslash": platform.KeyBackslash, "Backquote": platform.KeyGrave,

		// The modifiers by side, both sides to one key. OSLeft and OSRight are what Firefox
		// called the Windows key until version 118, and a player on an older one still has it.
		"ShiftLeft": platform.KeyShift, "ShiftRight": platform.KeyShift,
		"ControlLeft": platform.KeyControl, "ControlRight": platform.KeyControl,
		"AltLeft": platform.KeyAlt, "AltRight": platform.KeyAlt,
		"MetaLeft": platform.KeySuper, "MetaRight": platform.KeySuper,
		"OSLeft": platform.KeySuper, "OSRight": platform.KeySuper,
	}
	for i := 0; i < 10; i++ {
		d := strconv.Itoa(i)
		m["Digit"+d] = platform.Key0 + platform.Key(i)
		m["Numpad"+d] = platform.Key0 + platform.Key(i)
	}
	for i := 0; i < 12; i++ {
		m["F"+strconv.Itoa(i+1)] = platform.KeyF1 + platform.Key(i)
	}
	for i := 0; i < 26; i++ {
		m["Key"+string(rune('A'+i))] = platform.KeyA + platform.Key(i)
	}
	return m
}()

// keyOf is the key one press is, from the event's code and key.
//
// A letter is the layout's letter and not its place, which is the one way this differs from codes
// and the reason it is a function. It is what the other two backends do: a Windows virtual-key
// code and X11's keysym index 0 both follow the layout for letters, so on a French keyboard the
// key engraved A is KeyA on both, although it sits where a US keyboard has Q. And it is what the
// screens assume. The title screen says Q quits and the pause says S saves, and a player looks
// for those letters on their own keyboard. So when key is a Latin letter, in either case, it is
// the answer.
//
// Two cases fall back on the place. A letter outside ASCII -- Cyrillic, Greek -- is reported as
// the key a US keyboard has there, which is what Windows does for those layouts too, so a Russian
// player still has all 26. And a letter's place that types something else on this layout, such as
// AZERTY's comma where a US keyboard has M, is no key at all. Its letter has moved, keyOf finds it
// where it went, and the game must not see two keys called M.
//
// The keypad is the other place key decides. With Num Lock off its digits are the arrows and the
// editing keys, and a Windows virtual-key code says so (VK_LEFT, not VK_NUMPAD4), so here too 4 is
// left and . is Delete; the rest of them are keys the game has no name for.
func keyOf(code, key string) platform.Key {
	if len(key) == 1 {
		c := key[0] | 0x20 // ASCII lower case, which turns no other byte into a letter
		switch {
		case c >= 'a' && c <= 'z':
			return platform.KeyA + platform.Key(c-'a')
		case key[0] > ' ' && key[0] < 0x7F && strings.HasPrefix(code, "Key"):
			return platform.KeyUnknown
		}
	}
	if strings.HasPrefix(code, "Numpad") {
		switch key {
		case "ArrowLeft":
			return platform.KeyLeft
		case "ArrowRight":
			return platform.KeyRight
		case "ArrowUp":
			return platform.KeyUp
		case "ArrowDown":
			return platform.KeyDown
		case "Delete":
			return platform.KeyDelete
		case "Home", "End", "PageUp", "PageDown", "Insert", "Clear":
			return platform.KeyUnknown
		}
	}
	return codes[code]
}

// typed is what a press typed: the character KeyboardEvent.key says it produced, when it is one
// character and it prints, and "" otherwise.
//
// key names a key that types nothing -- "Enter", "Shift", "Dead" for the first half of an accent
// -- and every such name is longer than one character, so a single rune is always the character
// itself. The control-code bands are dropped as x11's eventText and win32's charAccum drop them,
// so that the three backends agree on what typing nothing means and the high-score field behaves
// the same on all three. A browser should never report one; this says what would happen if it did.
func typed(key string) string {
	r, n := utf8.DecodeRuneInString(key)
	if n == 0 || n != len(key) || r == utf8.RuneError {
		return ""
	}
	if r < 0x20 || (r >= 0x7F && r < 0xA0) {
		return ""
	}
	return key
}

// browserKeys are left to the browser although the game could name them: reload, full screen and
// the developer tools. A player who binds one gets it in the game and the browser's action as
// well, which is better than a page that cannot be reloaded.
var browserKeys = map[string]bool{"F5": true, "F11": true, "F12": true}

// press is the part of a KeyboardEvent this backend reads.
type press struct {
	code, key string
	down      bool // keydown rather than keyup
	repeat    bool

	// The modifier state, as the event reports it. altGraph is getModifierState("AltGraph"),
	// which Windows sets with ctrl and alt both: see chord.
	ctrl, meta, altGraph bool

	// composing is an input method's composition in progress, whose keystrokes are its own.
	composing bool
}

// chord reports whether a press belongs to the browser rather than to the game: one made with
// Control or Command held, which is how every browser shortcut is spelled -- reload, close the
// tab, find, zoom. Such a press still reaches the game as a key, so a Control binding works, but
// it types nothing and the browser's action is not stopped.
//
// AltGr is not a chord, although Windows reports it as Control and Alt together. It is how a
// German keyboard types @ and a Polish one types ą, and a high-score name with either in it is
// a name.
func chord(p press) bool { return (p.ctrl || p.meta) && !p.altGraph }

// keyboard is the key state the listeners keep, and the whole of what KeyDown reads.
type keyboard struct {
	// held is each key down now, by its code, and the key its press was reported as. A key's
	// release is reported as whatever its press was, whatever the modifiers have done in
	// between: AltGr pressed while A is held changes what key A's place reports, and the
	// release of a key the game never saw released is a key held down for ever.
	held map[string]platform.Key

	// count is how many held places report each key. Both Shifts are KeyShift, so the left one's
	// release must not let go of a Shift the right one is still holding.
	count [256]int
}

// event is what one press means: the event PollEvents hands out, if it is one the game has any
// use for, and whether the browser's own action for it is to be stopped.
//
// prevent is the arrows scrolling the page and Space paging it, Tab moving the focus off the
// canvas and Backspace, once, going back a page. It is set for every key the game can name except
// the browser's own and chords, and never for a key the game cannot name: whatever a browser
// does with those, the game has no claim on it.
func (kb *keyboard) event(p press) (ev platform.Event, ok, prevent bool) {
	if kb.held == nil {
		kb.held = map[string]platform.Key{}
	}
	k, isHeld := kb.held[p.code]
	if !isHeld {
		k = keyOf(p.code, p.key)
	}
	switch {
	case p.down && !isHeld:
		kb.held[p.code] = k
		kb.count[k]++
	case !p.down && isHeld:
		delete(kb.held, p.code)
		kb.count[k]--
	}

	c := chord(p)
	text := ""
	if p.down && !c && !p.composing {
		text = typed(p.key)
	}
	if k == platform.KeyUnknown {
		// A key the game has no name for still types, which is how an accented letter on a
		// layout nobody here has seen reaches the high-score field. x11.go says the same of its
		// own unmapped keys.
		if text == "" {
			return platform.Event{}, false, false
		}
		return platform.Event{Kind: platform.EventKeyDown, Repeat: p.repeat, Text: text}, true, false
	}
	kind := platform.EventKeyUp
	if p.down {
		kind = platform.EventKeyDown
	}
	ev = platform.Event{Kind: kind, Key: k, Repeat: p.down && p.repeat, Text: text}
	return ev, true, !c && !browserKeys[p.code]
}

// down reports whether k is held.
func (kb *keyboard) down(k platform.Key) bool {
	return k > platform.KeyUnknown && int(k) < len(kb.count) && kb.count[k] > 0
}

// release lets go of everything. The page lost the keyboard, and the keyups will go wherever it
// went: without this, a glider steered left when the player switched tabs is still steering left
// when they come back. x11.go's FocusOut arm is the same argument.
func (kb *keyboard) release() {
	clear(kb.held)
	kb.count = [256]int{}
}

// modifiers are the places a Mac still sends a keyup for while Command is held.
var modifiers = map[string]bool{
	"ShiftLeft": true, "ShiftRight": true, "ControlLeft": true, "ControlRight": true,
	"AltLeft": true, "AltRight": true, "MetaLeft": true, "MetaRight": true,
	"OSLeft": true, "OSRight": true, "CapsLock": true,
}

// commandUp is what a Mac needs when Command comes up. Its browsers send no keyup for a key let go
// of while Command was held, so a player two bound to Command who steered with it held would leave
// the key down for good. So every key still held except the modifiers is let go of now, and the
// releases the game never got are returned, in key order, for the backend to queue. On every
// other system those keyups do arrive, and web.go does not call this.
func (kb *keyboard) commandUp() []platform.Event {
	var out []platform.Event
	for code, k := range kb.held {
		if modifiers[code] {
			continue
		}
		delete(kb.held, code)
		kb.count[k]--
		if k != platform.KeyUnknown && kb.count[k] == 0 {
			out = append(out, platform.Event{Kind: platform.EventKeyUp, Key: k})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// isCommand reports whether a code is either Command key, by either of its names.
func isCommand(code string) bool {
	return code == "MetaLeft" || code == "MetaRight" || code == "OSLeft" || code == "OSRight"
}

// shown is the size the canvas is drawn at, in CSS pixels, for a w x h game at scale in a viewport
// of vw x vh CSS pixels, at dpr device pixels to the CSS pixel.
//
// scale is the most it is drawn at, in device pixels to the game's pixel: Config.Scale, which is
// the player's setting or auto, settled against the whole screen (Room). The page shows the game
// at that size when the viewport has room, and shrinks it to fit when it has not, by however much
// fitting takes rather than by whole steps. A browser window is a window on a screen, with its
// own toolbars: on a 1080p screen 2x needs 960 rows, a maximised window has about 950, and the
// whole step below is a postage stamp. Full screen gives the toolbars back, and the game returns
// to exactly scale.
//
// A viewport too small even for 1x shrinks it below 1x, which is the only way a phone shows the
// whole of a 640x480 game. A viewport that cannot be measured is ignored, and the game is drawn at
// scale.
func shown(scale, w, h int, vw, vh, dpr float64) (float64, float64) {
	if dpr <= 0 {
		dpr = 1
	}
	s := float64(max(scale, 1))
	if vw > 0 && vh > 0 {
		s = min(s, vw*dpr/float64(w), vh*dpr/float64(h))
	}
	return float64(w) * s / dpr, float64(h) * s / dpr
}
