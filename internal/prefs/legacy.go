package prefs

// Importing a 1994 preferences file.
//
// Glider PRO kept its settings in the data fork of a file called "Glider Prefs" (type
// 'gliP', creator 'ozm5') in the Preferences folder, written by one `FSWrite` of
// `sizeof(prefsInfo)` -- 226 bytes, no header, no resource fork, big-endian, mac68k
// alignment (Prefs.c:97-144). So an importer is a fixed-offset read and nothing more,
// and anybody with an old System Folder can bring their keys and their high-score name
// across.
//
// The layout is transcribed in docs/analysis/ui-dialogs.md 6.1 and reproduced in the
// offset constants below; the struct it comes from is at GliderPRO/Headers/Externs.h:233
// under `#pragma options align=mac68k`, which pads to 2 and gives the two dead bytes at
// 145 and 225. Both are written straight out of an uninitialised stack local, so two
// files with identical settings need not have identical bytes -- which is worth knowing
// if anybody ever diffs two of these.
//
// # What cannot be brought across
//
// Nineteen of the record's fields describe things this port does not have, and they are
// dropped rather than stored somewhere hopeful:
//
//   - The nine editor geometry fields (`wasEditH` through `isMapTop`), `wasToolGroup`,
//     `wasFloor`, `wasSuite`, `wasAutoEdit`, `wasMapOpen`, `wasToolsOpen`,
//     `wasCoordOpen`, `wasPrettyMap` and `wasMaxFiles`: the editor is stage 5. When it
//     exists it can grow its own file; guessing now at where its windows should sit
//     would be inventing data, not importing it.
//   - `wasDepthPref`, `wasScreen2`, `wasDoColorFade`, `wasZooms` and `wasQuickTrans`:
//     screen-depth switching, second-monitor play and the zoom-rectangle transitions
//     are Toolbox-era behaviours with no counterpart here (docs/IMPROVEMENTS.md 2.8).
//   - `smWarnings` and `wasBitchDialogs`: counters for alerts this port does not raise.
//   - `wasHouseChecks`: house integrity checking, which this port always does, because
//     internal/house validates on load and a malformed house is a diagnostic rather
//     than a crash.
//   - `wasDoAutoDemo`: the idle demo. Real, and coming with the shell's attract mode;
//     until there is something to switch off, a switch would be a lie.
//   - `wasMusicOn`: not a preference at all. `isMusicOn` means "the music channel is
//     playing right now" -- StopTheMusic clears it, StartMusic sets it (Music.c:107-121)
//     -- and it reached the file only because WriteOutPrefs copied every global it could
//     see. Importing it would let "I quit during a silent room" become "music is off".
//     The two flags that *are* preferences, `wasIdleMusic` and `wasGameMusic`, come
//     across.
//
// Notes records the dropped fields that a player might otherwise go looking for, so the
// import is honest about what it did rather than quietly lossy.

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/platform"
)

// LegacySize is the exact size of a `prefsInfo` record, which is the exact size of the
// original's preferences file. A shorter file is what `ReadPrefs` reports as `eofErr`,
// and LoadPrefs answers that by deleting it (Prefs.c:252-257).
const LegacySize = 226

// LegacyVersion is `kPrefsVersion`, the value at offset 164 in a file written by the
// shipped 1.1.2 build. The original compares it with `!=` and deletes the file on any
// difference; ImportLegacy reports a difference and carries on, because a field this
// importer reads is at the same offset in every version that ever shipped and a wrong
// guess about one field is cheaper than throwing away all of them.
const LegacyVersion = 0x0034

// Offsets into the record. Named rather than inlined because every one of them is a
// claim about a 32-year-old struct layout and TestLegacyOffsets checks them against a
// record built from the same list.
const (
	offHouseName  = 0   // Str32  wasDefaultName
	offLeftName   = 33  // Str15  wasLeftName
	offRightName  = 49  // Str15  wasRightName
	offBattName   = 65  // Str15  wasBattName
	offBandName   = 81  // Str15  wasBandName
	offHighName   = 97  // Str15  wasHighName
	offHighBanner = 113 // Str31  wasHighBanner
	offPad1       = 145 // alignment, uninitialised in the original
	offLeftMap    = 146 // long   wasLeftMap   (a KeyMap *bit offset*, not a keycode)
	offRightMap   = 150 // long   wasRightMap
	offBattMap    = 154 // long   wasBattMap
	offBandMap    = 158 // long   wasBandMap
	offVolume     = 162 // short  wasVolume     0..7
	offVersion    = 164 // short  prefVersion
	offMaxFiles   = 166 // short  wasMaxFiles
	offEditorRect = 168 // 14 shorts of editor window geometry, dropped
	offNeighbors  = 196 // short  wasNumNeighbors
	offDepthPref  = 198 // short  wasDepthPref
	offToolGroup  = 200 // short  wasToolGroup
	offWarnings   = 202 // short  smWarnings
	offFloor      = 204 // short  wasFloor
	offSuite      = 206 // short  wasSuite
	offZooms      = 208 // Boolean wasZooms
	offMusicOn    = 209 // Boolean wasMusicOn      (runtime state; see the note above)
	offAutoEdit   = 210 // Boolean wasAutoEdit
	offColorFade  = 211 // Boolean wasDoColorFade
	offMapOpen    = 212 // Boolean wasMapOpen
	offToolsOpen  = 213 // Boolean wasToolsOpen
	offCoordOpen  = 214 // Boolean wasCoordOpen
	offQuickTrans = 215 // Boolean wasQuickTrans
	offIdleMusic  = 216 // Boolean wasIdleMusic
	offGameMusic  = 217 // Boolean wasGameMusic
	offEscPause   = 218 // Boolean wasEscPauseKey
	offAutoDemo   = 219 // Boolean wasDoAutoDemo
	offScreen2    = 220 // Boolean wasScreen2
	offBackground = 221 // Boolean wasDoBackground
	offChecks     = 222 // Boolean wasHouseChecks
	offPrettyMap  = 223 // Boolean wasPrettyMap
	offBitchDlogs = 224 // Boolean wasBitchDialogs
	offPad2       = 225 // tail alignment, uninitialised in the original
)

// ImportLegacyFile reads an original "Glider Prefs" file.
func ImportLegacyFile(path string) (*Prefs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	p, err := ImportLegacy(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return p, nil
}

// ImportLegacy converts a 226-byte `prefsInfo` record into this port's preferences.
//
// It starts from Default(), so every field the record has no opinion about -- scale, the
// three fidelity switches, whether time is kept in real seconds -- gets this port's
// default rather than a zero. What the record does say wins, then Validate repairs
// anything out of range, and Notes holds the running commentary.
//
// The only error is a record too short to read, which is the original's own `eofErr`
// case. Everything else is a note.
func ImportLegacy(data []byte) (*Prefs, error) {
	if len(data) < LegacySize {
		return nil, fmt.Errorf("not a Glider Prefs record: %d bytes, want at least %d",
			len(data), LegacySize)
	}

	p := Default()
	if len(data) > LegacySize {
		// Not fatal. The original itself reads exactly 226 bytes and ignores the rest,
		// and a longer file most likely means somebody concatenated something or a
		// later build grew the struct -- in which case the prefix is still ours.
		p.note("the record is %d bytes; reading the first %d", len(data), LegacySize)
		data = data[:LegacySize]
	}

	if v := int(int16(binary.BigEndian.Uint16(data[offVersion:]))); v != LegacyVersion {
		p.note("this record says version 0x%04X and the shipped Glider PRO wrote 0x%04X; "+
			"reading it anyway", uint16(v), LegacyVersion)
	}

	if n := pstr(data[offHouseName:], 33); n != "" {
		p.House = n
	}
	if n := pstr(data[offHighName:], 16); n != "" {
		p.HighName = n
	}
	if n := pstr(data[offHighBanner:], 32); n != "" {
		p.HighBanner = n
	}

	// The four bindings. Each has two representations in the record and they can
	// disagree; the bit offset is the one the original actually played with, so it wins,
	// and the display name is the fallback. See internal/platform/keys.go for why
	// storing both was a mistake worth not repeating.
	for _, b := range []struct {
		label   string
		mapOff  int
		nameOff int
		set     func(*Controls, string)
	}{
		{"left", offLeftMap, offLeftName, func(c *Controls, s string) { c.Left = s }},
		{"right", offRightMap, offRightName, func(c *Controls, s string) { c.Right = s }},
		{"battery", offBattMap, offBattName, func(c *Controls, s string) { c.Batt = s }},
		{"band", offBandMap, offBandName, func(c *Controls, s string) { c.Band = s }},
	} {
		off := int32(binary.BigEndian.Uint32(data[b.mapOff:]))
		shown := pstr(data[b.nameOff:], 16)

		if k, ok := keyFromKeyMapOffset(off); ok {
			b.set(&p.Player1, platform.KeyName(k))
			continue
		}
		// The offset named a key this port does not model -- a keypad key, Help, F13,
		// caps lock. The display name is worth a try: the original's GetKeyName strings
		// are ParseKey aliases precisely for this.
		if k, ok := platform.ParseKey(shown); ok {
			p.note("the %s key was %s, which this build cannot bind; using %s from the record's own label",
				b.label, describeKeyMapOffset(off), platform.KeyName(k))
			b.set(&p.Player1, platform.KeyName(k))
			continue
		}
		p.note("the %s key was %s, which this build cannot bind; leaving it as %s",
			b.label, describeKeyMapOffset(off), keyOfControl(&p.Player1, b.label))
	}

	p.Volume = int(int16(binary.BigEndian.Uint16(data[offVolume:])))
	p.Neighbors = int(int16(binary.BigEndian.Uint16(data[offNeighbors:])))

	// wasIdleMusic and wasGameMusic, the two real music preferences. Any non-zero byte
	// is true: Boolean was a byte and the original assigned Booleans to it, but a
	// hand-edited or half-written record can hold anything.
	p.MusicOnTitle = data[offIdleMusic] != 0
	p.MusicInGame = data[offGameMusic] != 0

	if data[offEscPause] != 0 {
		p.PauseKey = "escape"
	} else {
		p.PauseKey = "tab"
	}

	// `doBackground` -- the Brains pane's "Background Tasks" -- is the one editor-era
	// flag with a live counterpart: it is what World.DoBackground gates, the spin that
	// stops the simulation while the window is not frontmost (Play.c's PlayGame). The
	// port's default inverts the original's, so an import is also the one place a
	// player's old answer is honoured exactly. See Prefs.PauseWhenUnfocused.
	p.PauseWhenUnfocused = data[offBackground] != 0

	// The dropped fields, named once, only where a player would notice the loss.
	if data[offAutoDemo] == 0 {
		p.note("the idle demo was switched off; this build has no attract mode yet, so there is nothing to switch")
	}
	if mf := int(int16(binary.BigEndian.Uint16(data[offMaxFiles:]))); mf != 48 {
		p.note("the house file limit was %d; this build has no limit and no editor to need one", mf)
	}
	if data[offMapOpen] != 0 || data[offToolsOpen] != 0 || data[offCoordOpen] != 0 || data[offAutoEdit] != 0 {
		p.note("editor window positions and tool settings are in this record; they are kept out of prefs.json until there is an editor to use them")
	}

	p.Validate()
	return p, nil
}

// keyOfControl is a small helper for the note above: what a binding currently says.
func keyOfControl(c *Controls, label string) string {
	for _, f := range controlFields {
		if f.label == label {
			return f.get(c)
		}
	}
	return "unknown"
}

// pstr reads a Pascal string out of a fixed-size field: one length byte then that many
// Mac OS Roman characters. A length past the field is clamped, which is what the
// original's own validation does with room names and is the only sane answer for a file
// this old (internal/house/pstr.go).
func pstr(b []byte, size int) string {
	if len(b) < size {
		size = len(b)
	}
	n := int(b[0])
	if n > size-1 {
		n = size - 1
	}
	return house.MacRomanToUTF8(b[1 : 1+n])
}

// ---------------------------------------------------------------------------
// KeyMap bit offsets
// ---------------------------------------------------------------------------

// keyMapOffset converts between a Macintosh virtual keycode and the bit offset that
// `BitTst` wants for a `KeyMap`, which is what the record stores.
//
// It is Utilities.c:504-517 verbatim:
//
//	hi = raw & 0xF0; lo = raw & 0x0F
//	off = lo <= 7 ? hi + (7 - lo) : hi + (0x17 - lo)
//
// The reason for the shuffle is that KeyMap is four big-endian-addressed 32-bit words
// scanned by a bit-test instruction that numbers bits from the most significant end, so
// a keycode's byte stays put while its low nibble is mirrored inside the byte. The
// function is its own inverse -- 7-(7-lo) is lo, and 0x17-(0x17-lo) is lo -- which is
// why one function serves both directions here and why the original could get away with
// having only the one.
func keyMapOffset(v int32) int32 {
	hi := v & 0xF0
	lo := v & 0x0F
	if lo <= 7 {
		return hi + (7 - lo)
	}
	return hi + (0x17 - lo)
}

// keyFromKeyMapOffset turns a stored bit offset into a key this port can poll, or
// reports false if the offset names a key internal/platform does not model.
func keyFromKeyMapOffset(off int32) (platform.Key, bool) {
	if off < 0 || off > 127 {
		return platform.KeyUnknown, false
	}
	k, ok := macKeys[keyMapOffset(off)]
	return k, ok
}

// describeKeyMapOffset names an offset for a note. Keys this port does model are named
// by their port name; the rest get the label the original's GetKeyName would have shown
// where there is one, and a raw keycode where there is not.
func describeKeyMapOffset(off int32) string {
	if off < 0 || off > 127 {
		return fmt.Sprintf("bit offset %d, which is not a key at all", off)
	}
	raw := keyMapOffset(off)
	if k, ok := macKeys[raw]; ok {
		return platform.KeyName(k)
	}
	if n, ok := macKeyLabels[raw]; ok {
		return n
	}
	return fmt.Sprintf("Macintosh keycode 0x%02X", raw)
}

// macKeys is the Macintosh virtual keycode table, restricted to the keys
// internal/platform models.
//
// The codes are the ANSI layout's, which is the one the shipped Glider PRO was played
// on and the one every later Apple keyboard kept for the physical positions. They are
// derived from Inside Macintosh's table and cross-checked against every worked example
// in docs/analysis/input.md: the four arrows at offsets 121-124, Tab at 55, Escape at
// 50, Delete at 52, Space at 54, Command at 48 (the only documented case that takes
// the `lo <= 7` branch), Shift at 63, and A/X/Z/C/R at 7/0/1/15/8.
//
// Codes deliberately absent: 0x0A (the ISO section key, not on the keyboards this
// shipped for), 0x39 caps lock, 0x47 clear, the keypad at 0x41-0x5C, and Help, Home,
// End, Page Up, Page Down, forward delete and F13-F15. Every one of them is bindable in
// the original and none is modelled by internal/platform, so an import that finds one
// falls back to the record's display name and says so.
var macKeys = map[int32]platform.Key{
	0x00: platform.KeyA, 0x01: platform.KeyS, 0x02: platform.KeyD, 0x03: platform.KeyF,
	0x04: platform.KeyH, 0x05: platform.KeyG, 0x06: platform.KeyZ, 0x07: platform.KeyX,
	0x08: platform.KeyC, 0x09: platform.KeyV, 0x0B: platform.KeyB, 0x0C: platform.KeyQ,
	0x0D: platform.KeyW, 0x0E: platform.KeyE, 0x0F: platform.KeyR, 0x10: platform.KeyY,
	0x11: platform.KeyT,

	0x12: platform.Key1, 0x13: platform.Key2, 0x14: platform.Key3, 0x15: platform.Key4,
	0x16: platform.Key6, 0x17: platform.Key5, 0x19: platform.Key9, 0x1A: platform.Key7,
	0x1C: platform.Key8, 0x1D: platform.Key0,

	0x18: platform.KeyEqual, 0x1B: platform.KeyMinus,
	0x1E: platform.KeyRightBracket, 0x21: platform.KeyLeftBracket,

	0x1F: platform.KeyO, 0x20: platform.KeyU, 0x22: platform.KeyI, 0x23: platform.KeyP,
	0x25: platform.KeyL, 0x26: platform.KeyJ, 0x28: platform.KeyK,
	0x2D: platform.KeyN, 0x2E: platform.KeyM,

	0x24: platform.KeyReturn,
	0x27: platform.KeyQuote, 0x29: platform.KeySemicolon, 0x2A: platform.KeyBackslash,
	0x2B: platform.KeyComma, 0x2C: platform.KeySlash, 0x2F: platform.KeyPeriod,
	0x30: platform.KeyTab, 0x31: platform.KeySpace, 0x32: platform.KeyGrave,
	0x33: platform.KeyDelete, 0x35: platform.KeyEscape,

	// The four modifiers, and the reason player two's original bindings are unusable on
	// a modern desktop: Command is the window manager's, and on many keyboards the pair
	// of Shifts and the pair of Options cannot be told apart (docs/IMPROVEMENTS.md 2.3).
	0x37: platform.KeySuper, 0x38: platform.KeyShift,
	0x3A: platform.KeyAlt, 0x3B: platform.KeyControl,

	0x7A: platform.KeyF1, 0x78: platform.KeyF2, 0x63: platform.KeyF3, 0x76: platform.KeyF4,
	0x60: platform.KeyF5, 0x61: platform.KeyF6, 0x62: platform.KeyF7, 0x64: platform.KeyF8,
	0x65: platform.KeyF9, 0x6D: platform.KeyF10, 0x67: platform.KeyF11, 0x6F: platform.KeyF12,

	0x7B: platform.KeyLeft, 0x7C: platform.KeyRight,
	0x7D: platform.KeyDown, 0x7E: platform.KeyUp,
}

// macKeyLabels names the keys macKeys leaves out, so a note can say "the battery key
// was the keypad's 5" instead of "keycode 0x57". The strings are the original's own
// GetKeyName output where it had one (docs/analysis/ui-dialogs.md 6.10).
var macKeyLabels = map[int32]string{
	0x0A: "the ISO section key",
	0x39: "caps lock",
	0x47: "clear",
	0x4C: "the keypad's enter",
	0x69: "F13", 0x6B: "F14", 0x71: "F15",
	0x72: "help", 0x73: "home", 0x74: "pg up", 0x75: "frwd del",
	0x77: "end", 0x79: "pg dn",
}

func init() {
	// The keypad, named as a range rather than thirteen rows. The codes are not
	// contiguous and the gaps are the arithmetic keys, which is why this is a lookup and
	// not a formula.
	for raw, label := range map[int32]string{
		0x52: "the keypad's 0", 0x53: "the keypad's 1", 0x54: "the keypad's 2",
		0x55: "the keypad's 3", 0x56: "the keypad's 4", 0x57: "the keypad's 5",
		0x58: "the keypad's 6", 0x59: "the keypad's 7", 0x5B: "the keypad's 8",
		0x5C: "the keypad's 9", 0x41: "the keypad's decimal point",
		0x43: "the keypad's *", 0x45: "the keypad's +",
		0x4B: "the keypad's /", 0x4E: "the keypad's -", 0x51: "the keypad's =",
	} {
		macKeyLabels[raw] = label
	}
}
