package web

// These run on any host, which is the point of keys.go having no build tag: they are the only part
// of the browser backend that `go test` can execute. The rest is checked in a real browser by
// tools/webcheck.

import (
	"math"
	"testing"

	"github.com/bwenstar/gliderGo/internal/platform"
)

// TestEveryPlatformKeyHasACode is win32's TestEveryPlatformKeyHasAVirtualKey for this backend: a
// key added to platform's enum and offered on the settings screen fails here, and not in a
// player's browser as a binding that does nothing.
func TestEveryPlatformKeyHasACode(t *testing.T) {
	mapped := map[platform.Key]bool{}
	for _, k := range codes {
		mapped[k] = true
	}
	for _, name := range platform.KeyNames() {
		k, ok := platform.ParseKey(name)
		if !ok {
			t.Fatalf("platform.KeyNames() returned %q, which platform.ParseKey rejects", name)
		}
		if !mapped[k] {
			t.Errorf("no KeyboardEvent.code maps to %q -- that key cannot be pressed in a "+
				"browser; add it to codes in keys.go", name)
		}
	}
}

// TestTheCodeTableMatchesUIEvents spot-checks the table against the W3C's list of code values, at
// both ends of every range. A table typed in from a specification is the kind of thing that is
// 95% right.
func TestTheCodeTableMatchesUIEvents(t *testing.T) {
	for _, tc := range []struct {
		code string
		want platform.Key
		note string
	}{
		{"ArrowLeft", platform.KeyLeft, "player one's default left"},
		{"ArrowRight", platform.KeyRight, ""},
		{"ArrowUp", platform.KeyUp, "player one's rubber band"},
		{"ArrowDown", platform.KeyDown, "player one's battery"},
		{"Enter", platform.KeyReturn, ""},
		{"NumpadEnter", platform.KeyReturn, "the numpad's own code, unlike Windows' one VK"},
		{"Escape", platform.KeyEscape, ""},
		{"Tab", platform.KeyTab, "the original's pause key"},
		{"Backspace", platform.KeyDelete, ""},
		{"Delete", platform.KeyDelete, "give up a glider in limbo"},
		{"Space", platform.KeySpace, ""},

		{"KeyA", platform.KeyA, "the bottom of the letter range"},
		{"KeyZ", platform.KeyZ, "the top of the letter range"},
		{"Digit0", platform.Key0, "the bottom of the digit range"},
		{"Digit2", platform.Key2, "which starts a two-player game"},
		{"Digit9", platform.Key9, "the top of the digit range"},
		{"Numpad0", platform.Key0, ""},
		{"Numpad9", platform.Key9, ""},
		{"F1", platform.KeyF1, ""},
		{"F10", platform.KeyF10, "two digits"},
		{"F12", platform.KeyF12, "the top of the function range"},

		{"Minus", platform.KeyMinus, ""},
		{"Equal", platform.KeyEqual, ""},
		{"BracketLeft", platform.KeyLeftBracket, ""},
		{"BracketRight", platform.KeyRightBracket, ""},
		{"Semicolon", platform.KeySemicolon, ""},
		{"Quote", platform.KeyQuote, ""},
		{"Backslash", platform.KeyBackslash, ""},
		{"Backquote", platform.KeyGrave, ""},
		{"Comma", platform.KeyComma, ""},
		{"Period", platform.KeyPeriod, ""},
		{"Slash", platform.KeySlash, ""},

		{"ShiftLeft", platform.KeyShift, ""},
		{"ShiftRight", platform.KeyShift, ""},
		{"ControlRight", platform.KeyControl, ""},
		{"AltLeft", platform.KeyAlt, "Option on a Mac"},
		{"AltRight", platform.KeyAlt, "AltGr, where a layout has one"},
		{"MetaLeft", platform.KeySuper, "Command, or the Windows key"},
		{"OSRight", platform.KeySuper, "Firefox's name for it until 118"},

		// Unmapped, each for its own reason, and all of them still type: see keyboard.event.
		{"IntlBackslash", platform.KeyUnknown, "the ISO key left of Z; Windows has no VK for it either"},
		{"ContextMenu", platform.KeyUnknown, ""},
		{"F13", platform.KeyUnknown, "past the enum's F12"},
		{"", platform.KeyUnknown, "what a synthetic event with no code says"},
	} {
		if got := codes[tc.code]; got != tc.want {
			t.Errorf("codes[%q] = %q, want %q %s",
				tc.code, platform.KeyName(got), platform.KeyName(tc.want), tc.note)
		}
	}
}

// TestALetterIsTheLayoutsLetter: what keyOf adds to the table, in the layouts that disagree with
// a US keyboard about where the letters are.
func TestALetterIsTheLayoutsLetter(t *testing.T) {
	for _, tc := range []struct {
		code, key string
		want      platform.Key
		why       string
	}{
		{"KeyQ", "q", platform.KeyQ, "US: the place and the letter agree"},
		{"KeyQ", "Q", platform.KeyQ, "with Shift, or Caps Lock"},
		{"KeyQ", "a", platform.KeyA, "AZERTY's A, where US has Q, as on X11 and Windows"},
		{"KeyA", "q", platform.KeyQ, "AZERTY's Q, which quits on the title screen"},
		{"KeyZ", "w", platform.KeyW, "AZERTY's W, player two's rubber band"},
		{"Semicolon", "m", platform.KeyM, "AZERTY's M, right of L"},
		{"KeyY", "z", platform.KeyZ, "QWERTZ"},

		// A letter's place that types something else has given its letter away.
		{"KeyM", ",", platform.KeyUnknown, "AZERTY's comma, where US has M: M is right of L"},
		{"KeyQ", "@", platform.KeyUnknown, "AltGr+Q on a German keyboard"},

		// The place, where the key says nothing a Latin layout would.
		{"KeyQ", "й", platform.KeyQ, "Russian: the letters are where US has them"},
		{"KeyE", "€", platform.KeyE, "AltGr+E: the euro is not a key"},
		{"KeyA", "å", platform.KeyA, "Option+A on a Mac"},
		{"KeyE", "Dead", platform.KeyE, "the first half of an accent"},
		{"KeyE", "Unidentified", platform.KeyE, "a browser that would not say"},
		{"Digit2", "é", platform.Key2, "AZERTY's 2, unshifted"},
		{"Digit2", "@", platform.Key2, "US Shift+2: not a letter's place"},
		{"Semicolon", ";", platform.KeySemicolon, "US"},
		{"Space", " ", platform.KeySpace, ""},
		{"ArrowLeft", "ArrowLeft", platform.KeyLeft, ""},
	} {
		if got := keyOf(tc.code, tc.key); got != tc.want {
			t.Errorf("keyOf(%q, %q) = %q, want %q (%s)",
				tc.code, tc.key, platform.KeyName(got), platform.KeyName(tc.want), tc.why)
		}
	}
	// And on a US keyboard the two halves of keyOf agree for every letter, so the exception
	// costs a US player nothing.
	for c := 'a'; c <= 'z'; c++ {
		code := "Key" + string(c-'a'+'A')
		if got, want := keyOf(code, string(c)), codes[code]; got != want {
			t.Errorf("keyOf(%q, %q) = %q, but the table says %q",
				code, string(c), platform.KeyName(got), platform.KeyName(want))
		}
	}
}

// TestTypedIsOnePrintableCharacter covers what reaches the high-score name field, on the same
// terms as win32's TestCharAccumBuildsTheTextAKeyTyped.
func TestTypedIsOnePrintableCharacter(t *testing.T) {
	for _, tc := range []struct{ key, want string }{
		{"a", "a"},
		{"B", "B"},
		{" ", " "},
		{"é", "é"},
		{"α", "α"},
		{"漢", "漢"},
		{"\U0001F600", "\U0001F600"},

		// Names of keys that type nothing. Every one is longer than a character.
		{"Enter", ""},
		{"Backspace", ""},
		{"Shift", ""},
		{"Dead", ""},
		{"Process", ""},
		{"", ""},

		// What no browser sends, held to what x11 and win32 drop.
		{"\r", ""},
		{"\x1b", ""},
		{"\x7f", ""},
		{"\u0085", ""},
		{"\u009f", ""},
		{"\u00a0", "\u00a0"}, // a character again, as in charAccum
		{"ab", ""},
		{"e\u0301", ""}, // an e and a combining acute: two runes, which no key types
	} {
		if got := typed(tc.key); got != tc.want {
			t.Errorf("typed(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func down(code, key string) press { return press{code: code, key: key, down: true} }
func up(code, key string) press   { return press{code: code, key: key} }

// TestAReleaseIsWhatThePressWas: a key's release reports the key its press did, however the
// modifiers changed what its place reports in between, and only then is it let go of.
func TestAReleaseIsWhatThePressWas(t *testing.T) {
	var kb keyboard
	ev, ok, _ := kb.event(down("KeyQ", "a")) // AZERTY's A
	if !ok || ev.Kind != platform.EventKeyDown || ev.Key != platform.KeyA {
		t.Fatalf("the press was %+v, ok=%v; want KeyA down", ev, ok)
	}
	if !kb.down(platform.KeyA) {
		t.Fatal("KeyA is not held after its press")
	}
	// AltGr goes down while A is held, and A comes up as æ.
	ev, ok, _ = kb.event(up("KeyQ", "æ"))
	if !ok || ev.Kind != platform.EventKeyUp || ev.Key != platform.KeyA {
		t.Errorf("the release was %+v, ok=%v; want KeyA up", ev, ok)
	}
	if kb.down(platform.KeyA) || kb.down(platform.KeyQ) {
		t.Error("a key is still held after its only press was released")
	}
}

// TestARepeatIsTheSameKey: auto-repeat is reported as a repeat of the key that went down, and
// types again, as on X11.
func TestARepeatIsTheSameKey(t *testing.T) {
	var kb keyboard
	kb.event(down("KeyA", "a"))
	p := down("KeyA", "a")
	p.repeat = true
	ev, _, _ := kb.event(p)
	if ev.Key != platform.KeyA || !ev.Repeat || ev.Text != "a" {
		t.Errorf("the repeat was %+v; want KeyA, Repeat, typing a", ev)
	}
	kb.event(up("KeyA", "a"))
	if kb.down(platform.KeyA) {
		t.Error("one release did not let go of a key that repeated")
	}
	r := up("KeyA", "a")
	r.repeat = true
	if ev, _, _ := kb.event(r); ev.Repeat {
		t.Error("a release was reported as a repeat")
	}
}

// TestBothShiftsMustBeLetGo: two places report one key, and it is held while either is.
func TestBothShiftsMustBeLetGo(t *testing.T) {
	var kb keyboard
	kb.event(down("ShiftLeft", "Shift"))
	kb.event(down("ShiftRight", "Shift"))
	kb.event(up("ShiftLeft", "Shift"))
	if !kb.down(platform.KeyShift) {
		t.Error("letting go of the left Shift let go of the right one too")
	}
	kb.event(up("ShiftRight", "Shift"))
	if kb.down(platform.KeyShift) {
		t.Error("Shift is held with both Shifts up")
	}
	// A release the game never saw pressed -- the key went down before the page had the
	// keyboard -- is reported, and holds nothing.
	ev, ok, _ := kb.event(up("KeyD", "d"))
	if !ok || ev.Kind != platform.EventKeyUp || ev.Key != platform.KeyD || kb.down(platform.KeyD) {
		t.Errorf("an unseen press's release was %+v, ok=%v", ev, ok)
	}
}

// TestAChordIsTheBrowsers: Control-R and Command-W still reach the game as keys, type nothing, and
// do what the browser does with them. AltGr is not a chord.
func TestAChordIsTheBrowsers(t *testing.T) {
	var kb keyboard
	p := down("KeyR", "r")
	p.ctrl = true
	ev, ok, prevent := kb.event(p)
	if !ok || ev.Key != platform.KeyR || ev.Text != "" || prevent {
		t.Errorf("Control-R was %+v, ok=%v, prevent=%v; want KeyR, no text, not stopped",
			ev, ok, prevent)
	}

	p = down("KeyW", "w")
	p.meta = true
	if _, _, prevent := kb.event(p); prevent {
		t.Error("Command-W was stopped, and closing the tab is the browser's")
	}

	// AltGr+Q on a German keyboard: Windows reports Control and Alt as well.
	p = down("KeyQ", "@")
	p.ctrl, p.altGraph = true, true
	ev, ok, _ = kb.event(p)
	if !ok || ev.Text != "@" || ev.Key != platform.KeyUnknown {
		t.Errorf("AltGr+Q was %+v, ok=%v; want an @ typed by no key", ev, ok)
	}

	// And composition belongs to the input method.
	p = down("KeyK", "k")
	p.composing = true
	if ev, _, _ := kb.event(p); ev.Text != "" {
		t.Errorf("a key pressed mid-composition typed %q", ev.Text)
	}
}

// TestWhatTheBrowserKeeps: which presses have the browser's own action stopped.
func TestWhatTheBrowserKeeps(t *testing.T) {
	for _, tc := range []struct {
		p    press
		want bool
		why  string
	}{
		{down("ArrowDown", "ArrowDown"), true, "it would scroll the page"},
		{up("ArrowDown", "ArrowDown"), true, "both halves of the press"},
		{down("Space", " "), true, "it would page down"},
		{down("Tab", "Tab"), true, "the pause key, and it would move the focus"},
		{down("Backspace", "Backspace"), true, "it went back a page once"},
		{down("KeyQ", "q"), true, "the game's"},
		{down("F5", "F5"), false, "reload"},
		{down("F11", "F11"), false, "full screen"},
		{down("F12", "F12"), false, "the developer tools"},
		{down("F1", "F1"), true, "bindable, and the browser's help is not worth it"},
		{down("IntlBackslash", "<"), false, "not the game's to stop"},
	} {
		var kb keyboard
		if _, _, got := kb.event(tc.p); got != tc.want {
			t.Errorf("%+v: prevent = %v, want %v (%s)", tc.p, got, tc.want, tc.why)
		}
	}
	// A key the game cannot name and that typed nothing is not an event at all.
	var kb keyboard
	if _, ok, _ := kb.event(down("ContextMenu", "ContextMenu")); ok {
		t.Error("the context-menu key was handed to the game")
	}
	if ev, ok, _ := kb.event(down("IntlBackslash", "<")); !ok || ev.Text != "<" {
		t.Errorf("the ISO key's < was %+v, ok=%v; want it typed", ev, ok)
	}
}

// TestReleaseLetsGoOfEverything is the lost-focus arm.
func TestReleaseLetsGoOfEverything(t *testing.T) {
	var kb keyboard
	kb.event(down("ArrowLeft", "ArrowLeft"))
	kb.event(down("ShiftLeft", "Shift"))
	kb.release()
	if kb.down(platform.KeyLeft) || kb.down(platform.KeyShift) {
		t.Fatal("a key is still held after release")
	}
	// And the next press of the same key is a press, not a repeat of one that was let go.
	ev, _, _ := kb.event(down("ArrowLeft", "ArrowLeft"))
	if ev.Kind != platform.EventKeyDown || ev.Repeat || !kb.down(platform.KeyLeft) {
		t.Errorf("the press after release was %+v", ev)
	}
}

// TestShown is the canvas's size in the cases a player will meet.
func TestShown(t *testing.T) {
	for _, tc := range []struct {
		name         string
		scale        int
		vw, vh, dpr  float64
		wantW, wantH float64
	}{
		{"full screen at 1080p fits auto's 2x", 2, 1920, 1080, 1, 1280, 960},
		{"a maximised window at 1080p shrinks it a little", 2, 1920, 950, 1, 950.0 * 4 / 3, 950},
		{"a wide window is fitted by its height", 3, 3000, 1200, 1, 1600, 1200},
		{"a tall window by its width", 3, 1000, 2000, 1, 1000, 750},
		{"room to spare is not used", 1, 1920, 1080, 1, 640, 480},
		{"at 125% a device pixel is 0.8 CSS pixels", 2, 1536, 864, 1.25, 1024, 768},
		{"at 125% in a maximised window", 2, 1536, 760, 1.25, 760.0 * 4 / 3, 760},
		{"a Retina laptop at 2x is 640x480 CSS pixels", 2, 1440, 900, 2, 640, 480},
		{"a small window shrinks it below 1x", 1, 500, 400, 1, 500, 375},
		{"a phone draws 1x in device pixels", 1, 390, 844, 3, 640.0 / 3, 160},
		{"0 is 1:1, as in platform.Config", 0, 1920, 1080, 1, 640, 480},
		{"no viewport to measure", 2, 0, 0, 1, 1280, 960},
		{"no device pixel ratio", 2, 1920, 1080, 0, 1280, 960},
	} {
		gotW, gotH := shown(tc.scale, 640, 480, tc.vw, tc.vh, tc.dpr)
		if math.Abs(gotW-tc.wantW) > 1e-9 || math.Abs(gotH-tc.wantH) > 1e-9 {
			t.Errorf("%s: shown(%d, 640, 480, %v, %v, %v) = %v x %v, want %v x %v",
				tc.name, tc.scale, tc.vw, tc.vh, tc.dpr, gotW, gotH, tc.wantW, tc.wantH)
		}
	}
}

// TestTheKeypadWithNumLockOff: its digits are the arrows and the editing keys then, as a Windows
// virtual-key code says.
func TestTheKeypadWithNumLockOff(t *testing.T) {
	for _, tc := range []struct {
		code, key string
		want      platform.Key
	}{
		{"Numpad4", "ArrowLeft", platform.KeyLeft},
		{"Numpad6", "ArrowRight", platform.KeyRight},
		{"Numpad8", "ArrowUp", platform.KeyUp},
		{"Numpad2", "ArrowDown", platform.KeyDown},
		{"NumpadDecimal", "Delete", platform.KeyDelete},
		{"Numpad7", "Home", platform.KeyUnknown},
		{"Numpad3", "PageDown", platform.KeyUnknown},
		{"Numpad5", "Clear", platform.KeyUnknown},
		// And with it on, the digits are the digits.
		{"Numpad4", "4", platform.Key4},
		{"Numpad0", "0", platform.Key0},
		{"Numpad7", "Unidentified", platform.Key7},
	} {
		if got := keyOf(tc.code, tc.key); got != tc.want {
			t.Errorf("keyOf(%q, %q) = %q, want %q", tc.code, tc.key, platform.KeyName(got), platform.KeyName(tc.want))
		}
	}
}

// TestALetGoCommandLetsGoOfTheKeysItHid: a Mac sends no keyup for a key released while Command is
// held, so Command's own release lets go of every key but the modifiers.
func TestALetGoCommandLetsGoOfTheKeysItHid(t *testing.T) {
	var kb keyboard
	kb.event(down("ShiftLeft", "Shift"))
	kb.event(down("ArrowLeft", "ArrowLeft"))
	kb.event(down("KeyD", "d"))
	kb.event(down("MetaLeft", "Meta"))
	// ArrowLeft and D come up while Command is held, and the Mac says nothing.
	kb.event(up("MetaLeft", "Meta"))
	evs := kb.commandUp()
	if len(evs) != 2 || evs[0].Key != platform.KeyLeft || evs[1].Key != platform.KeyD ||
		evs[0].Kind != platform.EventKeyUp || evs[1].Kind != platform.EventKeyUp {
		t.Fatalf("commandUp = %+v, want ArrowLeft's and D's releases, in key order", evs)
	}
	if kb.down(platform.KeyLeft) || kb.down(platform.KeyD) {
		t.Error("a key Command hid the release of is still held")
	}
	if !kb.down(platform.KeyShift) {
		t.Error("Shift, whose release a Mac does send, was let go of too")
	}
	if evs := kb.commandUp(); len(evs) != 0 {
		t.Errorf("a second commandUp released %+v", evs)
	}
}
