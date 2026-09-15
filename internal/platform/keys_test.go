package platform

import "testing"

// The property the preferences file rests on: every key the enum has can be written
// down and read back as the same key. A key with no name would be saved as "unknown",
// refused by ParseKey on the next launch, and silently reset to a default -- a binding
// that will not stay bound, which is the worst kind of settings bug because it looks
// like the save failed.
func TestEveryKeyHasANameAndRoundTrips(t *testing.T) {
	for k := KeyUnknown + 1; k < numKeys; k++ {
		name := KeyName(k)
		if name == "unknown" {
			t.Errorf("key %d has no name in keyNames", int(k))
			continue
		}
		got, ok := ParseKey(name)
		if !ok {
			t.Errorf("ParseKey(%q) failed for key %d", name, int(k))
			continue
		}
		if got != k {
			t.Errorf("ParseKey(KeyName(%d)) = %d, want %d (via %q)", int(k), int(got), int(k), name)
		}
	}

	// And no two keys share a name, or one of them cannot be expressed.
	seen := map[string]Key{}
	for k := KeyUnknown + 1; k < numKeys; k++ {
		n := KeyName(k)
		if prev, dup := seen[n]; dup {
			t.Errorf("keys %d and %d are both called %q", int(prev), int(k), n)
		}
		seen[n] = k
	}
	if n := len(KeyNames()); n != int(numKeys)-1 {
		t.Errorf("KeyNames lists %d names for %d keys", n, int(numKeys)-1)
	}
}

func TestKeyNameSpotChecks(t *testing.T) {
	for _, tc := range []struct {
		k    Key
		want string
	}{
		{KeyLeft, "left"}, {KeyDown, "down"}, {KeyTab, "tab"}, {KeyEscape, "escape"},
		{KeyA, "a"}, {KeyZ, "z"}, {Key0, "0"}, {Key9, "9"},
		{KeyF1, "f1"}, {KeyF9, "f9"}, {KeyF10, "f10"}, {KeyF12, "f12"},
		{KeySuper, "super"}, {KeyGrave, "grave"}, {KeyUnknown, "unknown"},
	} {
		if got := KeyName(tc.k); got != tc.want {
			t.Errorf("KeyName(%d) = %q, want %q", int(tc.k), got, tc.want)
		}
	}
}

// The aliases exist for two readers: somebody editing prefs.json by hand, and the
// legacy importer, which finds the original's own display strings in a 1994 record.
func TestParseKeyAliasesAndRejections(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Key
	}{
		{"left", KeyLeft},
		{"LEFT", KeyLeft},
		{"  left  ", KeyLeft},
		{"lf arrow", KeyLeft}, // the original's GetKeyName output
		{"rt arrow", KeyRight},
		{"dn arrow", KeyDown},
		{"up arrow", KeyUp},
		{"esc", KeyEscape},
		{"command", KeySuper}, // player two's original battery key
		{"option", KeyAlt},
		{"ctrl", KeyControl},
		{"enter", KeyReturn},
		{"A", KeyA},
		{"/", KeySlash},
	} {
		got, ok := ParseKey(tc.in)
		if !ok || got != tc.want {
			t.Errorf("ParseKey(%q) = %d,%v; want %d,true", tc.in, int(got), ok, int(tc.want))
		}
	}

	// An alias must not have stolen a canonical name. "enter" is return's alias and
	// KeyReturn is still "return"; if the table were built the other way round a
	// saved file would flip between the two spellings on every write.
	if KeyName(KeyReturn) != "return" {
		t.Errorf("an alias overwrote a canonical name: KeyReturn is %q", KeyName(KeyReturn))
	}

	for _, bad := range []string{"", " ", "unknown", "f13", "hyper", "left arrow key", "å"} {
		if k, ok := ParseKey(bad); ok {
			t.Errorf("ParseKey(%q) accepted it as key %d", bad, int(k))
		}
	}
}

// KeyChar is the fallback for a backend that reports keys but not text, so what it must
// not do is invent characters for keys that type nothing: a text field that appended
// KeyChar's answer for Escape or F1 would put junk in the player's name.
func TestKeyCharTypesOnlyWhatTypes(t *testing.T) {
	for _, tc := range []struct {
		k       Key
		plain   rune
		shifted rune
	}{
		{KeyA, 'a', 'A'},
		{KeyZ, 'z', 'Z'},
		{Key0, '0', ')'},
		{Key1, '1', '!'},
		{Key9, '9', '('},
		{KeySpace, ' ', ' '},
		{KeyMinus, '-', '_'},
		{KeyEqual, '=', '+'},
		{KeyComma, ',', '<'},
		{KeyPeriod, '.', '>'},
		{KeySlash, '/', '?'},
		{KeySemicolon, ';', ':'},
		{KeyQuote, '\'', '"'},
		{KeyBackslash, '\\', '|'},
		{KeyGrave, '`', '~'},
		{KeyLeftBracket, '[', '{'},
		{KeyRightBracket, ']', '}'},
	} {
		if got, ok := KeyChar(tc.k, false); !ok || got != tc.plain {
			t.Errorf("KeyChar(%q, false) = %q,%v; want %q,true", KeyName(tc.k), got, ok, tc.plain)
		}
		if got, ok := KeyChar(tc.k, true); !ok || got != tc.shifted {
			t.Errorf("KeyChar(%q, true) = %q,%v; want %q,true", KeyName(tc.k), got, ok, tc.shifted)
		}
	}

	for _, k := range []Key{
		KeyUnknown, KeyLeft, KeyRight, KeyUp, KeyDown, KeyReturn, KeyEscape, KeyTab,
		KeyDelete, KeyShift, KeyControl, KeyAlt, KeySuper, KeyF1, KeyF12, numKeys,
	} {
		if r, ok := KeyChar(k, false); ok {
			t.Errorf("KeyChar(%q) invented %q; that key types nothing", KeyName(k), r)
		}
	}

	// Every key that types something types exactly one rune, in both states. The
	// original's name field is a Str15 of Mac Roman bytes, so a keystroke that produced
	// two characters would silently change how many a name has room for.
	for k := KeyUnknown + 1; k < numKeys; k++ {
		for _, shift := range []bool{false, true} {
			if r, ok := KeyChar(k, shift); ok && (r < 0x20 || r > 0x7E) {
				t.Errorf("KeyChar(%q, %v) = %U, which is not printable ASCII", KeyName(k), shift, r)
			}
		}
	}
}

// The two descriptions of a US keyboard in this package -- keyNames' aliases, which are
// the engravings, and usUnshifted, which is what those keys type -- have to agree, or the
// settings screen and the text field disagree about the same physical key.
func TestKeyCharAgreesWithTheKeyNames(t *testing.T) {
	for _, e := range keyNames {
		r, ok := KeyChar(e.key, false)
		if !ok {
			continue
		}
		if e.key == KeySpace {
			continue // its engraving is a word, and " " is the alias
		}
		if _, isAlias := keyOf[string(r)]; !isAlias {
			t.Errorf("KeyChar says %s types %q, but %q is not one of its names",
				e.name, r, string(r))
			continue
		}
		if got := keyOf[string(r)]; got != e.key {
			t.Errorf("KeyChar says %s types %q, but ParseKey(%q) is %s",
				e.name, r, string(r), KeyName(got))
		}
	}

	// And the other direction, so a key added to usUnshifted without a name is caught:
	// every entry there is a key keyNames lists.
	for k := range usUnshifted {
		if KeyName(k) == "unknown" {
			t.Errorf("usUnshifted has key %d, which keyNames does not name", int(k))
		}
	}
}
