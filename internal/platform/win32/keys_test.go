package win32

// These tests run on Linux, which is the whole point of keys.go having no build tag: they are the
// only part of the Windows backend that can be executed on the machine it was written on.

import (
	"testing"
	"unicode/utf16"

	"github.com/bwenstar/gliderGo/internal/platform"
)

// TestEveryPlatformKeyHasAVirtualKey is the test that earns its keep over time. platform.Key is
// the port's own enum and it grows: the day someone adds a key to it and binds it in the settings
// screen, this fails on Linux rather than in a bug report from a Windows player whose new binding
// does nothing.
//
// It walks platform's own list of names rather than a copy of the enum, so there is no second
// list here to fall out of date.
func TestEveryPlatformKeyHasAVirtualKey(t *testing.T) {
	mapped := map[platform.Key]int{}
	for vk, k := range vkKeys {
		if k != platform.KeyUnknown {
			mapped[k]++
			_ = vk
		}
	}
	for _, name := range platform.KeyNames() {
		k, ok := platform.ParseKey(name)
		if !ok {
			t.Fatalf("platform.KeyNames() returned %q, which platform.ParseKey rejects", name)
		}
		if mapped[k] == 0 {
			t.Errorf("no virtual-key code maps to %q -- that key cannot be pressed on Windows; "+
				"add it to vkKeys in keys.go", name)
		}
	}
}

// TestTheVirtualKeyTableMatchesWinUser spot-checks the codes against the header they came from,
// with the ranges checked at both ends and in the middle. A table transcribed from documentation
// is exactly the kind of thing that is 95% right.
func TestTheVirtualKeyTableMatchesWinUser(t *testing.T) {
	for _, tc := range []struct {
		vk   uintptr
		want platform.Key
		note string
	}{
		{0x25, platform.KeyLeft, "VK_LEFT"},
		{0x26, platform.KeyUp, "VK_UP"},
		{0x27, platform.KeyRight, "VK_RIGHT"},
		{0x28, platform.KeyDown, "VK_DOWN"},
		{0x0D, platform.KeyReturn, "VK_RETURN, which is also the numpad's Enter"},
		{0x1B, platform.KeyEscape, "VK_ESCAPE"},
		{0x09, platform.KeyTab, "VK_TAB -- the original's pause key"},
		{0x08, platform.KeyDelete, "VK_BACK"},
		{0x2E, platform.KeyDelete, "VK_DELETE -- give up a glider in limbo"},
		{0x20, platform.KeySpace, "VK_SPACE"},

		{0x41, platform.KeyA, "VK_A, and player two's left"},
		{0x44, platform.KeyD, "VK_D, player two's right"},
		{0x53, platform.KeyS, "VK_S, player two's battery and the pause screen's save"},
		{0x57, platform.KeyW, "VK_W, player two's rubber band"},
		{0x51, platform.KeyQ, "VK_Q, quit"},
		{0x5A, platform.KeyZ, "the top of the letter range"},

		{0x30, platform.Key0, "the bottom of the digit range"},
		{0x32, platform.Key2, "VK_2, which starts a two-player game"},
		{0x39, platform.Key9, "the top of the digit range"},
		{0x60, platform.Key0, "VK_NUMPAD0"},
		{0x69, platform.Key9, "VK_NUMPAD9"},

		{0x70, platform.KeyF1, "VK_F1"},
		{0x7B, platform.KeyF12, "VK_F12, the top of the function range"},

		{0x10, platform.KeyShift, "VK_SHIFT"},
		{0xA0, platform.KeyShift, "VK_LSHIFT"},
		{0xA1, platform.KeyShift, "VK_RSHIFT"},
		{0x11, platform.KeyControl, "VK_CONTROL"},
		{0x12, platform.KeyAlt, "VK_MENU, which is Alt"},
		{0xA5, platform.KeyAlt, "VK_RMENU"},
		{0x5B, platform.KeySuper, "VK_LWIN"},

		{0xBD, platform.KeyMinus, "VK_OEM_MINUS"},
		{0xBB, platform.KeyEqual, "VK_OEM_PLUS"},
		{0xDB, platform.KeyLeftBracket, "VK_OEM_4"},
		{0xDD, platform.KeyRightBracket, "VK_OEM_6"},
		{0xBA, platform.KeySemicolon, "VK_OEM_1"},
		{0xDE, platform.KeyQuote, "VK_OEM_7"},
		{0xDC, platform.KeyBackslash, "VK_OEM_5"},
		{0xC0, platform.KeyGrave, "VK_OEM_3"},
		{0xBF, platform.KeySlash, "VK_OEM_2"},
		{0xBC, platform.KeyComma, "VK_OEM_COMMA"},
		{0xBE, platform.KeyPeriod, "VK_OEM_PERIOD"},

		// Unmapped, and each for its own reason. 0x00 is not a key; 0x01 is the left mouse
		// button, which this game has no use for; 0x5D is the context-menu key.
		{0x00, platform.KeyUnknown, "not a virtual-key code"},
		{0x01, platform.KeyUnknown, "VK_LBUTTON -- Glider PRO is keyboard only"},
		{0x5D, platform.KeyUnknown, "VK_APPS"},
	} {
		if got := keyFor(tc.vk); got != tc.want {
			t.Errorf("keyFor(%#02x) = %q, want %q (%s)",
				tc.vk, platform.KeyName(got), platform.KeyName(tc.want), tc.note)
		}
	}
}

// TestKeyForRefusesAnImpossibleWParam: wParam is a uintptr, so nothing but this stops a garbage
// message indexing off the end of a 256-entry array.
func TestKeyForRefusesAnImpossibleWParam(t *testing.T) {
	for _, vk := range []uintptr{256, 0xFFFF, ^uintptr(0)} {
		if got := keyFor(vk); got != platform.KeyUnknown {
			t.Errorf("keyFor(%#x) = %q, want unknown", vk, platform.KeyName(got))
		}
	}
}

// TestCharAccumBuildsTheTextAKeyTyped covers what reaches the high-score name field.
func TestCharAccumBuildsTheTextAKeyTyped(t *testing.T) {
	for _, tc := range []struct {
		name  string
		units []uint16
		want  string
	}{
		{"a letter", []uint16{'B'}, "B"},
		{"a name", []uint16{'O', 'z', 'm', 'a'}, "Ozma"},
		{"a space", []uint16{' '}, " "},

		// Every one of these arrives as a WM_CHAR after a WM_KEYDOWN that already reported
		// the key, so passing it on would put a control code in a player's name.
		{"Return types nothing", []uint16{0x0D}, ""},
		{"Escape types nothing", []uint16{0x1B}, ""},
		{"Tab types nothing", []uint16{0x09}, ""},
		{"Backspace types nothing", []uint16{0x08}, ""},
		{"DEL types nothing", []uint16{0x7F}, ""},
		{"a C1 control types nothing", []uint16{0x85}, ""},
		{"0x9F is the last dropped one", []uint16{0x9F}, ""},

		// The point of doing this at all: a player whose name is not ASCII.
		{"0xA0 is a character again", []uint16{0xA0}, " "},
		{"an e-acute", []uint16{0xE9}, "é"},
		{"a Greek letter", []uint16{0x03B1}, "α"},
		{"a CJK character", []uint16{0x6F22}, "漢"},

		// Outside the BMP, so two messages for one character.
		{"a surrogate pair", []uint16{0xD83D, 0xDE00}, "\U0001F600"},
		{"a high-plane pair", []uint16{0xD869, 0xDEB2}, "\U0002A6B2"},
		{"a pair after a letter", []uint16{'a', 0xD83D, 0xDE00}, "a\U0001F600"},

		// Broken streams. None of these can happen from a working Windows, and all three are
		// one bad cast away from putting U+FFFD into a name, so each is pinned.
		{"a lone high surrogate", []uint16{0xD83D}, ""},
		{"a lone low surrogate", []uint16{0xDE00}, ""},
		{"a high surrogate then a letter", []uint16{0xD83D, 'x'}, "x"},
		{"two high surrogates then a low", []uint16{0xD800, 0xD83D, 0xDE00}, "\U0001F600"},
		{"a low surrogate then a pair", []uint16{0xDE00, 0xD83D, 0xDE00}, "\U0001F600"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var acc charAccum
			var got string
			for _, u := range tc.units {
				got += acc.add(u)
			}
			if got != tc.want {
				t.Errorf("units %#04x typed %q, want %q", tc.units, got, tc.want)
			}
		})
	}
}

// TestCharAccumAgreesWithUTF16 runs every printable code point in the BMP plus a slice of the
// astral planes through the encoder Windows uses and back through charAccum, which is a stronger
// statement than any table of examples: the two directions agree, so nothing is lost or shifted.
func TestCharAccumAgreesWithUTF16(t *testing.T) {
	check := func(r rune) {
		var acc charAccum
		var got string
		for _, u := range utf16.Encode([]rune{r}) {
			got += acc.add(u)
		}
		if got != string(r) {
			t.Fatalf("U+%04X round-tripped as %q", r, got)
		}
	}
	for r := rune(0xA0); r <= 0xFFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue // not characters; utf16.Encode substitutes U+FFFD for them
		}
		check(r)
	}
	for r := rune(0x10000); r <= 0x10FFFF; r += 0x101 {
		check(r)
	}
}
