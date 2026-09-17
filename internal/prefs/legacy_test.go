package prefs

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/platform"
)

// ---------------------------------------------------------------------------
// The KeyMap bit offsets
// ---------------------------------------------------------------------------

// Externs.h defines thirty-four of these offsets by hand, one per key the original ever
// needed to test. They are the best possible check on keyMapOffset and macKeys, because
// they were written by the person who wrote the code that consumed them: if this table
// passes, the involution and the keycode table are both right for every key the game
// itself used.
//
// The values are transcribed from GliderPRO/Headers/Externs.h:116-163. The keypad and
// caps-lock rows have no platform.Key, and are checked through describeKeyMapOffset
// instead -- which is exactly the path an imported binding takes when it names a key
// this port cannot poll.
func TestKeyMapOffsetsMatchTheHeader(t *testing.T) {
	for _, tc := range []struct {
		constant string
		off      int32
		key      platform.Key // KeyUnknown for the keys this port does not model
		label    string       // what describeKeyMapOffset should say when key is Unknown
	}{
		{"kUpArrowKeyMap", 121, platform.KeyUp, ""},
		{"kDownArrowKeyMap", 122, platform.KeyDown, ""},
		{"kRightArrowKeyMap", 123, platform.KeyRight, ""},
		{"kLeftArrowKeyMap", 124, platform.KeyLeft, ""},

		{"kAKeyMap", 7, platform.KeyA, ""},
		{"kBKeyMap", 12, platform.KeyB, ""},
		{"kCKeyMap", 15, platform.KeyC, ""},
		{"kDKeyMap", 5, platform.KeyD, ""},
		{"kEKeyMap", 9, platform.KeyE, ""},
		{"kFKeyMap", 4, platform.KeyF, ""},
		{"kGKeyMap", 2, platform.KeyG, ""},
		{"kHKeyMap", 3, platform.KeyH, ""},
		{"kMKeyMap", 41, platform.KeyM, ""},
		{"kNKeyMap", 42, platform.KeyN, ""},
		{"kOKeyMap", 24, platform.KeyO, ""},
		{"kPKeyMap", 36, platform.KeyP, ""},
		{"kQKeyMap", 11, platform.KeyQ, ""},
		{"kRKeyMap", 8, platform.KeyR, ""},
		{"kSKeyMap", 6, platform.KeyS, ""},
		{"kTKeyMap", 22, platform.KeyT, ""},
		{"kVKeyMap", 14, platform.KeyV, ""},
		{"kWKeyMap", 10, platform.KeyW, ""},
		{"kXKeyMap", 0, platform.KeyX, ""},
		{"kZKeyMap", 1, platform.KeyZ, ""},
		{"kPeriodKeyMap", 40, platform.KeyPeriod, ""},

		// The one documented case that takes the `lo <= 7` branch, and the modifiers
		// player two was bound to in 1994.
		{"kCommandKeyMap", 48, platform.KeySuper, ""},
		{"kControlKeyMap", 60, platform.KeyControl, ""},
		{"kOptionKeyMap", 61, platform.KeyAlt, ""},
		{"kShiftKeyMap", 63, platform.KeyShift, ""},

		{"kEscKeyMap", 50, platform.KeyEscape, ""},
		{"kDeleteKeyMap", 52, platform.KeyDelete, ""},
		{"kSpaceBarMap", 54, platform.KeySpace, ""},
		{"kTabKeyMap", 55, platform.KeyTab, ""},

		{"kCapsLockKeyMap", 62, platform.KeyUnknown, "caps lock"},

		{"kPlusKeypadMap", 66, platform.KeyUnknown, "keypad's +"},
		{"kMinusKeypadMap", 73, platform.KeyUnknown, "keypad's -"},
		{"kTimesKeypadMap", 68, platform.KeyUnknown, "keypad's *"},
		{"k0KeypadMap", 85, platform.KeyUnknown, "keypad's 0"},
		{"k1KeypadMap", 84, platform.KeyUnknown, "keypad's 1"},
		{"k2KeypadMap", 83, platform.KeyUnknown, "keypad's 2"},
		{"k3KeypadMap", 82, platform.KeyUnknown, "keypad's 3"},
		{"k4KeypadMap", 81, platform.KeyUnknown, "keypad's 4"},
		{"k5KeypadMap", 80, platform.KeyUnknown, "keypad's 5"},
		{"k6KeypadMap", 95, platform.KeyUnknown, "keypad's 6"},
		{"k7KeypadMap", 94, platform.KeyUnknown, "keypad's 7"},
		{"k8KeypadMap", 92, platform.KeyUnknown, "keypad's 8"},
		{"k9KeypadMap", 91, platform.KeyUnknown, "keypad's 9"},
	} {
		got, ok := keyFromKeyMapOffset(tc.off)
		if tc.key == platform.KeyUnknown {
			if ok {
				t.Errorf("%s (offset %d) resolved to %s; this port does not model that key",
					tc.constant, tc.off, platform.KeyName(got))
			}
			if d := describeKeyMapOffset(tc.off); !strings.Contains(d, tc.label) {
				t.Errorf("%s (offset %d) is described as %q; it should mention %q",
					tc.constant, tc.off, d, tc.label)
			}
			continue
		}
		if !ok {
			t.Errorf("%s (offset %d) did not resolve; want %s",
				tc.constant, tc.off, platform.KeyName(tc.key))
			continue
		}
		if got != tc.key {
			t.Errorf("%s (offset %d) = %s, want %s",
				tc.constant, tc.off, platform.KeyName(got), platform.KeyName(tc.key))
		}
	}
}

// The one property that makes a single function serve both directions.
func TestKeyMapOffsetIsItsOwnInverse(t *testing.T) {
	for v := int32(0); v < 128; v++ {
		if got := keyMapOffset(keyMapOffset(v)); got != v {
			t.Fatalf("keyMapOffset twice on %d gave %d", v, got)
		}
		// And it stays inside the byte it started in, which is the whole point of the
		// shuffle: the high nibble selects the KeyMap byte and must not move.
		if got := keyMapOffset(v); got&0xF0 != v&0xF0 {
			t.Fatalf("keyMapOffset(%d) = %d crossed a byte boundary", v, got)
		}
	}
	// Out of range is not a key. The field is a signed long in the file and nothing
	// stops it holding anything.
	for _, bad := range []int32{-1, 128, 1 << 20} {
		if _, ok := keyFromKeyMapOffset(bad); ok {
			t.Errorf("offset %d resolved to a key", bad)
		}
	}
}

// ---------------------------------------------------------------------------
// The record layout
// ---------------------------------------------------------------------------

// legacyFields is the record as a list of (offset, size), transcribed independently of
// the constants in legacy.go from the struct at GliderPRO/Headers/Externs.h:233. The
// test below checks that the two agree and that the result tiles exactly 226 bytes with
// nothing missing and nothing overlapping -- which is the only way to be sure an offset
// typo has not shifted every field after it by a byte.
var legacyFields = []struct {
	name   string
	off    int
	size   int
	scalar bool // a short or a long, and so subject to alignment; not a byte array
}{
	{"wasDefaultName", offHouseName, 33, false},
	{"wasLeftName", offLeftName, 16, false},
	{"wasRightName", offRightName, 16, false},
	{"wasBattName", offBattName, 16, false},
	{"wasBandName", offBandName, 16, false},
	{"wasHighName", offHighName, 16, false},
	{"wasHighBanner", offHighBanner, 32, false},
	{"(alignment)", offPad1, 1, false},
	{"wasLeftMap", offLeftMap, 4, true},
	{"wasRightMap", offRightMap, 4, true},
	{"wasBattMap", offBattMap, 4, true},
	{"wasBandMap", offBandMap, 4, true},
	{"wasVolume", offVolume, 2, true},
	{"prefVersion", offVersion, 2, true},
	{"wasMaxFiles", offMaxFiles, 2, true},
	{"editor geometry", offEditorRect, 28, true}, // 14 shorts: wasEditH..isMapTop
	{"wasNumNeighbors", offNeighbors, 2, true},
	{"wasDepthPref", offDepthPref, 2, true},
	{"wasToolGroup", offToolGroup, 2, true},
	{"smWarnings", offWarnings, 2, true},
	{"wasFloor", offFloor, 2, true},
	{"wasSuite", offSuite, 2, true},
	{"wasZooms", offZooms, 1, false},
	{"wasMusicOn", offMusicOn, 1, false},
	{"wasAutoEdit", offAutoEdit, 1, false},
	{"wasDoColorFade", offColorFade, 1, false},
	{"wasMapOpen", offMapOpen, 1, false},
	{"wasToolsOpen", offToolsOpen, 1, false},
	{"wasCoordOpen", offCoordOpen, 1, false},
	{"wasQuickTrans", offQuickTrans, 1, false},
	{"wasIdleMusic", offIdleMusic, 1, false},
	{"wasGameMusic", offGameMusic, 1, false},
	{"wasEscPauseKey", offEscPause, 1, false},
	{"wasDoAutoDemo", offAutoDemo, 1, false},
	{"wasScreen2", offScreen2, 1, false},
	{"wasDoBackground", offBackground, 1, false},
	{"wasHouseChecks", offChecks, 1, false},
	{"wasPrettyMap", offPrettyMap, 1, false},
	{"wasBitchDialogs", offBitchDlogs, 1, false},
	{"(alignment)", offPad2, 1, false},
}

func TestLegacyLayoutTilesTheRecord(t *testing.T) {
	next := 0
	for _, f := range legacyFields {
		if f.off != next {
			t.Errorf("%s is at %d; the field before it ends at %d", f.name, f.off, next)
		}
		// mac68k alignment: a scalar sits on an even offset, a byte array sits wherever
		// the field before it ended. That distinction is the entire reason there is a
		// pad at 145 and not one at 33: the six Pascal strings are `unsigned char[16]`
		// and align to 1, and the first `long` after them does not.
		if f.scalar && f.off%2 != 0 {
			t.Errorf("%s is a %d-byte scalar at odd offset %d", f.name, f.size, f.off)
		}
		next = f.off + f.size
	}
	if next != LegacySize {
		t.Errorf("the fields cover %d bytes; sizeof(prefsInfo) is %d", next, LegacySize)
	}
}

// ---------------------------------------------------------------------------
// Importing
// ---------------------------------------------------------------------------

// record builds a 226-byte prefsInfo. It starts from what the original would have
// written on a fresh install (Main.c:122-190) so that each test only has to say what it
// is testing.
//
// There is no shipped "Glider Prefs" file to check any of this against -- it was written
// per machine and none is in the source tree -- so this is a synthetic record built from
// the documented layout, and the honest limit of what these tests prove is "the importer
// reads the layout docs/analysis/ui-dialogs.md 6.1 describes". The layout itself is
// checked against the C struct by TestLegacyLayoutTilesTheRecord and against the header
// by TestKeyMapOffsetsMatchTheHeader; what remains unverifiable here is only whether a
// real 1994 file differs from its own source code.
func record() []byte {
	b := make([]byte, LegacySize)
	putPStr(b, offHouseName, 33, "Slumberland")
	putPStr(b, offLeftName, 16, "lf arrow")
	putPStr(b, offRightName, 16, "rt arrow")
	putPStr(b, offBattName, 16, "dn arrow")
	putPStr(b, offBandName, 16, "up arrow")
	putPStr(b, offHighName, 16, "Your Name")
	putPStr(b, offHighBanner, 32, "Your Message Here")

	putLong(b, offLeftMap, 124)  // kLeftArrowKeyMap
	putLong(b, offRightMap, 123) // kRightArrowKeyMap
	putLong(b, offBattMap, 122)  // kDownArrowKeyMap
	putLong(b, offBandMap, 121)  // kUpArrowKeyMap

	putShort(b, offVolume, 3) // a fresh install clamps the system volume to 1..3
	putShort(b, offVersion, LegacyVersion)
	putShort(b, offMaxFiles, 48)
	putShort(b, offNeighbors, 9)

	b[offZooms] = 1
	b[offMusicOn] = 1
	b[offColorFade] = 1
	b[offIdleMusic] = 1
	b[offGameMusic] = 1
	b[offEscPause] = 0 // isEscPauseKey false, so Tab pauses
	b[offAutoDemo] = 1
	b[offBackground] = 0 // doBackground false: the original keeps running unfocused
	b[offChecks] = 1
	b[offBitchDlogs] = 1

	// The editor's fresh-install state (Main.c:180-183). These are the fields that
	// cannot come across, and a record without them would not exercise the note that
	// says so.
	b[offAutoEdit] = 1
	b[offMapOpen] = 1
	b[offToolsOpen] = 1
	b[offCoordOpen] = 0

	// The two bytes the original never initialised. Filled with something recognisable
	// so that a test which accidentally reads one gets a legible wrong answer.
	b[offPad1] = 0xEE
	b[offPad2] = 0xEE
	return b
}

func putPStr(b []byte, off, size int, s string) {
	if len(s) > size-1 {
		s = s[:size-1]
	}
	b[off] = byte(len(s))
	copy(b[off+1:off+size], s)
}

func putShort(b []byte, off int, v int16) {
	binary.BigEndian.PutUint16(b[off:], uint16(v))
}

func putLong(b []byte, off int, v int32) {
	binary.BigEndian.PutUint32(b[off:], uint32(v))
}

// A record holding the original's own fresh-install settings has to import to this
// port's own defaults, or one of the two transcriptions is wrong. The exceptions are
// listed in the test, and each one is a documented deviation rather than a surprise.
func TestImportLegacyDefaultsMatchOurDefaults(t *testing.T) {
	p, err := ImportLegacy(record())
	if err != nil {
		t.Fatal(err)
	}
	d := Default()

	if p.House != d.House {
		t.Errorf("house %q, want %q", p.House, d.House)
	}
	if p.Player1 != d.Player1 {
		t.Errorf("player one %+v, want %+v", p.Player1, d.Player1)
	}
	if p.PauseKey != "tab" || p.EscPause() {
		t.Errorf("pause key %q", p.PauseKey)
	}
	if p.Neighbors != 9 {
		t.Errorf("neighbors %d", p.Neighbors)
	}
	if !p.MusicInGame || !p.MusicOnTitle {
		t.Errorf("music game=%v title=%v", p.MusicInGame, p.MusicOnTitle)
	}
	if p.HighName != "Your Name" || p.HighBanner != "Your Message Here" {
		t.Errorf("high score identity %q / %q", p.HighName, p.HighBanner)
	}
	// The volume is the one number that does *not* match, and deliberately: the original
	// reads it from the system's output volume and clamps to 1..3, and this port has no
	// business reading the desktop's master volume. An import takes the file's number.
	if p.Volume != 3 {
		t.Errorf("volume %d, want the record's 3", p.Volume)
	}
	if !p.Sound {
		t.Error("a non-zero volume must leave the sound on")
	}
	// wasDoBackground false, which the port's own default inverts. An import is the one
	// place the 1994 answer wins.
	if p.PauseWhenUnfocused {
		t.Error("the record says doBackground false, so the game should keep running unfocused")
	}
	// Nothing the record cannot describe was invented.
	if p.Scale != 1 || p.KeepRealTime || p.Fixes != (Fixes{}) {
		t.Errorf("a field the record has no opinion about was not left at its default: %+v", p)
	}

	// One note, about the editor settings that could not come across. Fidelity to the
	// record is not the same as pretending nothing was lost.
	if len(p.Notes) != 1 || !strings.Contains(p.Notes[0], "editor") {
		t.Errorf("notes are %v; want exactly one, about the editor", p.Notes)
	}
}

// The record stores each binding twice and the two can disagree, because the display
// name came from the keyboard layout in force when it was written. The bit offset is
// what the original actually tested against, so it wins.
func TestImportLegacyPrefersTheOffsetOverTheLabel(t *testing.T) {
	b := record()
	putLong(b, offLeftMap, 7) // kAKeyMap: the offset says A
	putPStr(b, offLeftName, 16, "lf arrow")

	p, err := ImportLegacy(b)
	if err != nil {
		t.Fatal(err)
	}
	if p.Player1.Left != "a" {
		t.Errorf("left is %q; the offset said A and the label said left arrow, and the "+
			"offset is the one the game played with", p.Player1.Left)
	}
	// Nothing was lost, so nothing is said about player one's left key: the label was
	// never authoritative and disagreeing with it is not a repair.
	// Checked by prefix, because the note about *player two's* collision names player
	// one's key as the thing it collided with.
	for _, n := range p.Notes {
		if strings.HasPrefix(n, "player one's left") {
			t.Errorf("unexpected note about player one's left key: %q", n)
		}
	}

	// Player two, though, defaults to A for *its* left thruster, so importing A for
	// player one collides. Player one keeps it -- an imported binding is a choice and a
	// default is not -- and player two's is unbound rather than moved to some third key
	// nobody picked.
	if p.Player2.Left != Unbound {
		t.Errorf("player two's left is %q; it collided with player one's A and should be unbound",
			p.Player2.Left)
	}
	if !noteMentions(p, "player two's left") || !noteMentions(p, "unbound") {
		t.Errorf("the collision was not explained: %v", p.Notes)
	}
	if got := p.Player2.Keys().Left; got != platform.KeyUnknown {
		t.Errorf("an unbound control resolved to %s; it has to be a key no backend reports",
			platform.KeyName(got))
	}
	// And player two's other three are untouched: one collision must not cascade.
	if d := Default(); p.Player2.Right != d.Player2.Right || p.Player2.Batt != d.Player2.Batt ||
		p.Player2.Band != d.Player2.Band {
		t.Errorf("the collision spread to the other controls: %+v", p.Player2)
	}

	// Validating again must be silent. A file saved in this state and reloaded would
	// otherwise complain on every launch about a binding the player has already been
	// told about.
	q := *p
	q.Notes = nil
	q.Validate()
	if len(q.Notes) != 0 {
		t.Errorf("re-validating a repaired set of bindings complained again: %v", q.Notes)
	}
	if q.Player2.Left != Unbound {
		t.Errorf("re-validating changed the unbound key to %q", q.Player2.Left)
	}
}

// A binding on a key this port cannot poll. Two rungs: the record's own label, then the
// default. Both say so.
func TestImportLegacyUnmodelledKeyFallsBack(t *testing.T) {
	t.Run("to the label", func(t *testing.T) {
		b := record()
		putLong(b, offBattMap, 80) // k5KeypadMap
		putPStr(b, offBattName, 16, "j")

		p, err := ImportLegacy(b)
		if err != nil {
			t.Fatal(err)
		}
		if p.Player1.Batt != "j" {
			t.Errorf("battery is %q, want j from the label", p.Player1.Batt)
		}
		if !noteMentions(p, "keypad's 5") {
			t.Errorf("no note about the key that could not be bound: %v", p.Notes)
		}
	})

	t.Run("to the default", func(t *testing.T) {
		b := record()
		putLong(b, offBattMap, 80)
		putPStr(b, offBattName, 16, "keypad 5") // not a name ParseKey knows

		p, err := ImportLegacy(b)
		if err != nil {
			t.Fatal(err)
		}
		if p.Player1.Batt != "down" {
			t.Errorf("battery is %q, want the default down", p.Player1.Batt)
		}
		if !noteMentions(p, "keypad's 5") || !noteMentions(p, "leaving it") {
			t.Errorf("the note does not say what happened: %v", p.Notes)
		}
	})

	t.Run("an offset that is not a key at all", func(t *testing.T) {
		b := record()
		putLong(b, offBandMap, 4242)
		putPStr(b, offBandName, 16, "\xff\xfe")

		p, err := ImportLegacy(b)
		if err != nil {
			t.Fatal(err)
		}
		if p.Player1.Band != "up" {
			t.Errorf("band is %q, want the default up", p.Player1.Band)
		}
		if !noteMentions(p, "not a key at all") {
			t.Errorf("notes are %v", p.Notes)
		}
	})
}

func TestImportLegacyShortRecordIsAnError(t *testing.T) {
	if _, err := ImportLegacy(record()[:LegacySize-1]); err == nil {
		t.Error("a 225-byte record was accepted; this is the original's own eofErr case")
	}
	if _, err := ImportLegacy(nil); err == nil {
		t.Error("an empty record was accepted")
	}
	// Longer is fine: the original reads the first 226 bytes and ignores the rest.
	long := append(record(), make([]byte, 40)...)
	p, err := ImportLegacy(long)
	if err != nil {
		t.Fatalf("a longer record should still import: %v", err)
	}
	if p.House != "Slumberland" {
		t.Errorf("house %q from a long record", p.House)
	}
	if !noteMentions(p, "266 bytes") {
		t.Errorf("the extra bytes were not mentioned: %v", p.Notes)
	}
}

// The version mismatch that made the original delete the file. Here it is a sentence.
func TestImportLegacyVersionMismatchIsANoteNotAFailure(t *testing.T) {
	b := record()
	putShort(b, offVersion, 0x0031) // some earlier build
	putPStr(b, offHighName, 16, "Calhoun")

	p, err := ImportLegacy(b)
	if err != nil {
		t.Fatal(err)
	}
	if p.HighName != "Calhoun" {
		t.Errorf("high name %q; the point of reading an older version is to keep this", p.HighName)
	}
	if !noteMentions(p, "0x0031") {
		t.Errorf("the version difference was not reported: %v", p.Notes)
	}
}

// House names are Mac OS Roman, and the shipped houses include names with characters
// that are not ASCII. A name decoded as Latin-1 would not match a file on disk.
func TestImportLegacyDecodesMacRoman(t *testing.T) {
	b := record()
	// 0x8E is é in Mac OS Roman (and Ž in Latin-1), 0xA5 is a bullet.
	b[offHouseName] = 6
	copy(b[offHouseName+1:], []byte{'C', 'a', 'f', 0x8E, ' ', 0xA5})

	p, err := ImportLegacy(b)
	if err != nil {
		t.Fatal(err)
	}
	if p.House != "Café •" {
		t.Errorf("house is %q, want %q", p.House, "Café •")
	}
}

func TestImportLegacyEscPauseAndBackground(t *testing.T) {
	b := record()
	b[offEscPause] = 1
	b[offBackground] = 1
	b[offIdleMusic] = 0
	b[offGameMusic] = 0

	p, err := ImportLegacy(b)
	if err != nil {
		t.Fatal(err)
	}
	if p.PauseKey != "escape" || !p.EscPause() {
		t.Errorf("pause key %q; wasEscPauseKey was set", p.PauseKey)
	}
	if !p.PauseWhenUnfocused {
		t.Error("wasDoBackground was set, so the game should pause when it loses focus")
	}
	if p.MusicInGame || p.MusicOnTitle {
		t.Error("both music preferences were off in the record")
	}
}

// wasMusicOn is runtime state, not a preference: it means "the music channel is playing
// right now". A player who quit from a silent screen must not find the music switched
// off next time.
func TestImportLegacyIgnoresTheRuntimeMusicFlag(t *testing.T) {
	b := record()
	b[offMusicOn] = 0
	b[offIdleMusic] = 1
	b[offGameMusic] = 1

	p, err := ImportLegacy(b)
	if err != nil {
		t.Fatal(err)
	}
	if !p.MusicInGame || !p.MusicOnTitle {
		t.Error("wasMusicOn was allowed to switch off the two real music preferences")
	}
}

// A record of zeroes: a truncated write, a file from a different program, a disk error.
// Every offset resolves to X (offset 0 is keycode 0x07), the volume is silence and the
// neighbour count is impossible. Nothing here may panic and nothing may pass silently.
func TestImportLegacyZeroRecordIsRepairedAndReported(t *testing.T) {
	p, err := ImportLegacy(make([]byte, LegacySize))
	if err != nil {
		t.Fatal(err)
	}
	if p.House != "Slumberland" {
		t.Errorf("an empty house name should leave the default, got %q", p.House)
	}
	if p.Player1.Left != "x" {
		t.Errorf("left is %q; bit offset 0 is keycode 0x07, which is X", p.Player1.Left)
	}
	// The other three offsets are also zero, so they collide with the left key and fall
	// back one at a time.
	if p.Player1.Right != "right" || p.Player1.Batt != "down" || p.Player1.Band != "up" {
		t.Errorf("the collisions were not repaired: %+v", p.Player1)
	}
	if p.Volume != 0 || p.Sound {
		t.Errorf("volume %d sound %v; a zeroed record means silence", p.Volume, p.Sound)
	}
	if p.Neighbors != 9 {
		t.Errorf("neighbors %d; 0 is not a value the renderer can compose", p.Neighbors)
	}
	if len(p.Notes) < 3 {
		t.Errorf("only %d notes for a record that is entirely zeroes: %v", len(p.Notes), p.Notes)
	}
}

func TestImportLegacyFileFromDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Glider Prefs")
	if err := os.WriteFile(path, record(), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := ImportLegacyFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.House != "Slumberland" {
		t.Errorf("house %q", p.House)
	}
	// The imported settings save to this port's own file, which is the whole point:
	// import once, then never think about the old one again.
	out := filepath.Join(dir, "prefs.json")
	if err := p.SaveFile(out); err != nil {
		t.Fatal(err)
	}
	q, err := LoadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if q.Player1 != p.Player1 || q.HighName != p.HighName {
		t.Errorf("the imported settings did not survive a save: %+v vs %+v", q, p)
	}

	if _, err := ImportLegacyFile(filepath.Join(dir, "no such file")); err == nil {
		t.Error("importing a file that is not there succeeded")
	}
}

func noteMentions(p *Prefs, s string) bool {
	return strings.Contains(strings.Join(p.Notes, "\n"), s)
}
