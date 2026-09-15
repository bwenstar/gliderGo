package platform

// Key names.
//
// Two things need to turn a key into text and back: the preferences file, which has
// to survive being written by one build and read by the next, and the settings screen,
// which has to show a player what a binding currently is. They are given one table
// rather than two on purpose -- the name on screen is byte for byte the name in the
// file, so somebody who can see "left" under Player One and wants to know what to type
// into prefs.json already knows.
//
// The names are lower case ASCII identifiers rather than the original's display forms.
// Glider PRO stored *both*: four `Str15` display names at offsets 33..96 of its
// preferences record and the four raw KeyMap offsets at 146..161, so a prefs file
// written by a Mac with a French keyboard held "A" beside a bit offset that meant Q
// (docs/analysis/ui-dialogs.md 6.1 and 6.10). Storing one canonical name for the
// physical key removes that whole class of disagreement, which matters here more than
// it did in 1994 because internal/platform's Key *is* the physical key -- backends
// translate host keycodes to it and no layout survives the trip.
//
// The original's own display strings are accepted as aliases by ParseKey, so a
// hand-written "lf arrow" or "esc" reads correctly and the legacy importer has
// something to fall back on when a record's bit offsets are unusable.

import "strings"

// keyNames is the whole table: canonical name first, then aliases.
//
// Every Key except KeyUnknown appears exactly once, and TestEveryKeyHasAName holds
// that true -- a key added to the enum with no entry here would otherwise persist as
// "unknown" and quietly reset itself on the next load.
var keyNames = []struct {
	key     Key
	name    string
	aliases []string
}{
	{KeyLeft, "left", []string{"lf arrow", "leftarrow", "arrowleft"}},
	{KeyRight, "right", []string{"rt arrow", "rightarrow", "arrowright"}},
	{KeyUp, "up", []string{"up arrow", "uparrow", "arrowup"}},
	{KeyDown, "down", []string{"dn arrow", "downarrow", "arrowdown"}},
	{KeySpace, "space", []string{"spacebar", " "}},
	{KeyReturn, "return", []string{"enter", "ret"}},
	{KeyEscape, "escape", []string{"esc"}},
	{KeyTab, "tab", nil},
	{KeyDelete, "delete", []string{"backspace", "del"}},

	{KeyMinus, "minus", []string{"-"}},
	{KeyEqual, "equal", []string{"="}},
	{KeyLeftBracket, "leftbracket", []string{"["}},
	{KeyRightBracket, "rightbracket", []string{"]"}},
	{KeyComma, "comma", []string{","}},
	{KeyPeriod, "period", []string{"."}},
	{KeySlash, "slash", []string{"/"}},
	{KeySemicolon, "semicolon", []string{";"}},
	{KeyQuote, "quote", []string{"'"}},
	{KeyBackslash, "backslash", []string{"\\"}},
	{KeyGrave, "grave", []string{"`", "tilde"}},

	// The four modifiers, with the Macintosh names they had in 1994 as aliases:
	// player two's original bindings are Control, Command, Option and Shift
	// (InterfaceInit.c:148-151), and a legacy prefs record or a reader of the
	// original's manual will call them that.
	{KeyShift, "shift", nil},
	{KeyControl, "control", []string{"ctrl"}},
	{KeyAlt, "alt", []string{"option", "opt"}},
	{KeySuper, "super", []string{"command", "cmd", "meta", "win"}},
}

// nameOf and keyOf are the table indexed both ways, built once.
var (
	nameOf = map[Key]string{}
	keyOf  = map[string]Key{}
)

func init() {
	add := func(k Key, name string, aliases ...string) {
		nameOf[k] = name
		keyOf[name] = k
		for _, a := range aliases {
			// Aliases never overwrite a canonical name: "enter" is an alias for
			// return and must not become return's own name, and no alias may shadow
			// a letter or digit.
			if _, taken := keyOf[a]; !taken {
				keyOf[a] = k
			}
		}
	}
	for _, e := range keyNames {
		add(e.key, e.name, e.aliases...)
	}
	// The letters, the digits and the function keys are ranges rather than rows,
	// because writing thirty-eight one-line entries invites exactly one typo.
	for k := KeyA; k <= KeyZ; k++ {
		add(k, string(rune('a'+int(k-KeyA))))
	}
	for k := Key0; k <= Key9; k++ {
		add(k, string(rune('0'+int(k-Key0))))
	}
	for k := KeyF1; k <= KeyF12; k++ {
		n := int(k-KeyF1) + 1
		if n < 10 {
			add(k, "f"+string(rune('0'+n)))
		} else {
			add(k, "f1"+string(rune('0'+n-10)))
		}
	}
}

// KeyName is the canonical name of a key: what the preferences file stores and what
// the settings screen shows. An unnamed key -- KeyUnknown, or one added to the enum
// with no table entry -- is "unknown", which is a legible thing to find in a config
// file and is not a name ParseKey will accept back.
func KeyName(k Key) string {
	if n, ok := nameOf[k]; ok {
		return n
	}
	return "unknown"
}

// ParseKey is KeyName's inverse, plus the aliases. It folds case and ignores
// surrounding space, because the file is meant to be editable by hand.
//
// It reports false rather than guessing. A binding that will not parse must fall back
// to a default the caller chooses and say so; silently binding the wrong key is the
// one outcome a player cannot diagnose.
func ParseKey(s string) (Key, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return KeyUnknown, false
	}
	k, ok := keyOf[s]
	return k, ok
}

// KeyNames lists every bindable key's canonical name, in enum order. It is what a
// `-help`-style listing and the settings screen's "press a key" prompt draw on, and
// the order is the enum's rather than alphabetical so the four arrows stay together.
func KeyNames() []string {
	out := make([]string, 0, len(nameOf))
	for k := KeyUnknown + 1; k < numKeys; k++ {
		if n, ok := nameOf[k]; ok {
			out = append(out, n)
		}
	}
	return out
}
