package prefs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/platform"
)

// The defaults are a transcription of Main.c's no-prefs block, so they are checked
// against it field by field. A default that drifts is not a crash and not a test
// failure anywhere else -- it is a first run that plays subtly unlike the original, and
// the only place it can be caught is here.
func TestDefaultsAreTheOriginals(t *testing.T) {
	d := Default()

	if d.House != "Slumberland" {
		t.Errorf("default house is %q; Main.c:129 says Slumberland", d.House)
	}
	// PasStringCopy("\plf arrow", leftName) and theGlider.leftKey = kLeftArrowKeyMap,
	// Main.c:131-138.
	if got := d.Player1.Keys(); got != (Binding{
		Left: platform.KeyLeft, Right: platform.KeyRight,
		Batt: platform.KeyDown, Band: platform.KeyUp,
	}) {
		t.Errorf("player one's defaults are %+v; the original's are the four arrows", got)
	}
	// isEscPauseKey = false, Main.c:184. So Tab pauses, and the overlay is PICT 1016.
	if d.PauseKey != "tab" || d.EscPause() {
		t.Errorf("default pause key is %q (esc=%v); the original's isEscPauseKey is false",
			d.PauseKey, d.EscPause())
	}
	if d.Neighbors != 9 {
		t.Errorf("default neighbors is %d; Main.c:153 says 9", d.Neighbors)
	}
	if !d.MusicInGame || !d.MusicOnTitle {
		t.Errorf("music defaults are game=%v title=%v; Main.c:149-150 sets both true",
			d.MusicInGame, d.MusicOnTitle)
	}
	if !d.Sound {
		t.Error("sound defaults off; Main.c:147 sets isSoundOn true")
	}
	if d.HighName != "Your Name" || d.HighBanner != "Your Message Here" {
		t.Errorf("high-score placeholders are %q / %q; Main.c:132-133 has "+
			`"Your Name" / "Your Message Here"`, d.HighName, d.HighBanner)
	}
	if d.KeepRealTime {
		t.Error("KeepRealTime defaults true; the original has no catch-up at all (2.17)")
	}
	// The two deliberate deviations, asserted so that changing one is a decision rather
	// than an accident. Both are argued on their fields.
	if !d.PauseWhenUnfocused {
		t.Error("PauseWhenUnfocused defaults false; the deviation from doBackground is intentional")
	}
	if got := d.Player2.Keys(); got != (Binding{
		Left: platform.KeyA, Right: platform.KeyD,
		Batt: platform.KeyS, Band: platform.KeyW,
	}) {
		t.Errorf("player two's defaults are %+v; the port's are A/D/S/W (2.3)", got)
	}
}

// Validate must have nothing to say about Default(), or every fresh run would print
// complaints about settings the player never chose.
func TestDefaultsValidateClean(t *testing.T) {
	d := Default()
	d.Validate()
	if len(d.Notes) != 0 {
		t.Errorf("Validate had %d notes about the defaults:\n  %s",
			len(d.Notes), strings.Join(d.Notes, "\n  "))
	}
	// And twice, because Validate runs on every load of a file that has already been
	// validated and written back.
	d.Validate()
	if len(d.Notes) != 0 {
		t.Errorf("Validate is not idempotent: %v", d.Notes)
	}
}

func TestMissingFileIsTheDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefs.json")
	p, err := LoadFile(path)
	if err != nil {
		t.Fatalf("a first run must not be an error: %v", err)
	}
	if len(p.Notes) != 0 {
		t.Errorf("a first run produced notes: %v", p.Notes)
	}
	if !reflect.DeepEqual(normalize(p), normalize(Default())) {
		t.Errorf("a missing file gave %+v, want the defaults", p)
	}
	if p.Path() != path {
		t.Errorf("Path() is %q, want %q -- Save has to know where to put it", p.Path(), path)
	}
	// And nothing was created. A game that writes a config file it was only asked to
	// read is a game that cannot be run from a read-only image.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("LoadFile created the file")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prefs.json")

	p := Default()
	p.House = "Cheese Ball Cavern"
	p.Player1 = Controls{Left: "j", Right: "l", Batt: "k", Band: "i"}
	p.Player2 = Controls{Left: "f", Right: "h", Batt: "g", Band: "t"}
	p.PauseKey = "escape"
	p.Scale = 3
	p.Neighbors = 3
	p.Volume = 4
	p.MusicInGame = false
	p.KeepRealTime = true
	p.PauseWhenUnfocused = false
	p.HighName = "Bo"
	p.HighBanner = "up and to the left"
	p.Fixes = Fixes{MirrorFlame: true, SwitchSparkle: true}

	if err := p.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	q, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Notes) != 0 {
		t.Errorf("reading back what we wrote produced notes: %v", q.Notes)
	}
	if !reflect.DeepEqual(normalize(p), normalize(q)) {
		t.Errorf("round trip changed the settings:\n saved %+v\n read  %+v", normalize(p), normalize(q))
	}

	// The file is text a person can edit, and the keys in it are the names the settings
	// screen shows. If this ever became a binary blob or an opaque number per key, the
	// reason this package exists would be gone.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"house": "Cheese Ball Cavern"`, `"left": "j"`, `"pause_key": "escape"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("the file does not contain %s:\n%s", want, data)
		}
	}

	// Saving twice leaves one file: the temporary the rename went through must not be
	// left behind, or a config directory fills up with prefs.json.tmp1234567.
	if err := p.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 || ents[0].Name() != "prefs.json" {
		names := make([]string, len(ents))
		for i, e := range ents {
			names[i] = e.Name()
		}
		t.Errorf("the directory holds %v, want just prefs.json", names)
	}
}

// A file written by a build with fewer settings, or by hand with just the one line
// somebody wanted to change. Every absent field takes its default and every unknown
// field is ignored -- which together are what make it safe to add a setting later
// without a migration, and what the original could not do at all. Ignored on the way
// in is not dropped on the way out: TestANewerBuildsSettingsSurviveASave.
func TestPartialFileKeepsTheDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefs.json")
	if err := os.WriteFile(path, []byte(`{
	  "volume": 2,
	  "house": "Teddy World",
	  "an_option_from_2031": {"nested": [1, 2, 3]}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Notes) != 0 {
		t.Errorf("notes for a partial file: %v", p.Notes)
	}
	if p.Volume != 2 || p.House != "Teddy World" {
		t.Errorf("the file's own values were lost: volume %d house %q", p.Volume, p.House)
	}
	d := Default()
	if p.Player1 != d.Player1 || p.PauseKey != d.PauseKey || p.Neighbors != d.Neighbors ||
		!p.MusicInGame || p.Scale != d.Scale {
		t.Errorf("an absent field did not take its default: %+v", p)
	}
	// A file that does not declare a version is taken to be this one, because that is
	// what a hand-written file is: somebody typed the two settings they cared about and
	// no reasonable person types a schema number. The alternative -- version 0 -- would
	// make every hand-written file look like it came from an older build.
	if p.Version != Version {
		t.Errorf("Version is %d for a file that does not declare one; want the current %d",
			p.Version, Version)
	}
}

// A newer build's settings, read by this one and saved again. The game saves whenever
// the remembered house or the high-score name changes, so a player who runs an older
// release after a newer one would otherwise lose every setting the older one has no
// field for -- and the note Validate prints for a newer file says they are kept.
func TestANewerBuildsSettingsSurviveASave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefs.json")
	if err := os.WriteFile(path, []byte(`{
	  "version": 2,
	  "house": "Teddy World",
	  "Volume": 2,
	  "fixes": {"mirror_flame": true, "a_fix_from_2031": true},
	  "player1": "not an object any more",
	  "an_option_from_2031": {"nested": [1, 2, 3]},
	  "another": null
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(p.Notes, "\n"), "newer build") {
		t.Errorf("no note for a newer build's file: %v", p.Notes)
	}
	p.House = "Nemo's Market"
	p.Volume = 5
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("the saved file is not JSON: %v\n%s", err, data)
	}
	for k, want := range map[string]string{
		"an_option_from_2031": `{"nested":[1,2,3]}`,
		"another":             `null`,
		"version":             fmt.Sprint(Version),
	} {
		var b bytes.Buffer
		if err := json.Compact(&b, got[k]); err != nil || b.String() != want {
			t.Errorf("%s is %s after a save, want %s", k, got[k], want)
		}
	}
	// Read into a field, so not unknown: kept, it would win the next load back.
	if _, ok := got["Volume"]; ok {
		t.Errorf("the file keeps \"Volume\" beside \"volume\":\n%s", data)
	}
	var fixes map[string]bool
	if err := json.Unmarshal(got["fixes"], &fixes); err != nil || !fixes["a_fix_from_2031"] ||
		!fixes["mirror_flame"] {
		t.Errorf("the fixes object lost a key: %s", got["fixes"])
	}

	q, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Notes) != 0 {
		t.Errorf("reading back what we wrote produced notes: %v", q.Notes)
	}
	if !reflect.DeepEqual(normalize(p), normalize(q)) {
		t.Errorf("the save changed the settings:\n saved %+v\n read  %+v", normalize(p), normalize(q))
	}
	// This build's own keys come first, in the struct's order, as they always have.
	own, kept := bytes.Index(data, []byte(`"house"`)), bytes.Index(data, []byte(`"another"`))
	if own < 0 || kept < own {
		t.Errorf("a kept key comes before this build's own:\n%s", data)
	}

	// And a second save is the first one again.
	if err := q.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, data) {
		t.Errorf("a second save changed the file:\n first %s\n second %s", data, again)
	}
}

// One field of the wrong type costs that field and nothing else. This is a property of
// encoding/json -- a type mismatch calls saveError and decoding continues -- and the
// whole design rests on it, which is why it is pinned here rather than assumed. (It is
// also why no type in this package has a custom UnmarshalJSON: an error returned from
// one aborts the rest of the decode, silently discarding every later field.)
func TestOneBadFieldDoesNotCostTheOthers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefs.json")
	if err := os.WriteFile(path, []byte(
		`{"volume": "loud", "house": "Ivy University", "scale": 2, "neighbors": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadFile(path)
	if err != nil {
		t.Fatalf("a type error must not fail the load: %v", err)
	}
	if p.House != "Ivy University" {
		t.Errorf("the field before the bad one was lost: house %q", p.House)
	}
	if p.Scale != 2 || p.Neighbors != 1 {
		t.Errorf("the fields after the bad one were lost: scale %d neighbors %d", p.Scale, p.Neighbors)
	}
	if p.Volume != Default().Volume {
		t.Errorf("volume is %d; the unreadable value should have left the default", p.Volume)
	}
	if len(p.Notes) == 0 {
		t.Error("a type error produced no note; the player would never know a setting was ignored")
	}
}

// A file that is not JSON at all cannot be repaired field by field. It is kept, not
// deleted -- the original's answer to the same situation is FSpDelete.
func TestCorruptFileIsMovedAsideNotDeleted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prefs.json")
	junk := []byte("\x00\x01this was never JSON{{{")
	if err := os.WriteFile(path, junk, 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadFile(path)
	if err != nil {
		t.Fatalf("a corrupt file must still yield usable settings: %v", err)
	}
	if !reflect.DeepEqual(normalize(p), normalize(Default())) {
		t.Error("a corrupt file did not fall back to the defaults")
	}
	if len(p.Notes) != 1 || !strings.Contains(p.Notes[0], "not valid JSON") {
		t.Errorf("notes are %v; one of them has to say the file was unreadable", p.Notes)
	}
	kept, err := os.ReadFile(path + ".bad")
	if err != nil {
		t.Fatalf("the unreadable file was not kept: %v", err)
	}
	if string(kept) != string(junk) {
		t.Error("the kept copy is not what was there")
	}
	// And a save afterwards writes a good file rather than refusing.
	if err := p.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err != nil {
		t.Fatalf("the replacement file does not load: %v", err)
	}
}

func TestValidateRepairsAndSaysSo(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(*Prefs)
		want func(*Prefs) string // "" if the repair is right
		says string
	}{{
		name: "a key that does not exist on this build",
		set:  func(p *Prefs) { p.Player1.Left = "f13" },
		want: func(p *Prefs) string {
			if p.Player1.Left != "left" {
				return "left binding is " + p.Player1.Left
			}
			return ""
		},
		says: "f13",
	}, {
		name: "a binding that would take the pause key",
		set:  func(p *Prefs) { p.Player1.Band = "tab" },
		want: func(p *Prefs) string {
			if p.Player1.Band != "up" {
				return "band binding is " + p.Player1.Band
			}
			return ""
		},
		says: "pause",
	}, {
		name: "two controls on one key",
		set:  func(p *Prefs) { p.Player1.Batt = "left" },
		want: func(p *Prefs) string {
			if p.Player1.Batt != "down" || p.Player1.Left != "left" {
				return "bindings are " + p.Player1.Left + "/" + p.Player1.Batt
			}
			return ""
		},
		says: "already",
	}, {
		name: "one player's key taken by the other",
		set:  func(p *Prefs) { p.Player2.Right = "right" },
		want: func(p *Prefs) string {
			if p.Player2.Right != "d" {
				return "player two's right is " + p.Player2.Right
			}
			return ""
		},
		says: "player one",
	}, {
		name: "a volume off the scale",
		set:  func(p *Prefs) { p.Volume = 11 },
		want: func(p *Prefs) string {
			if p.Volume != Default().Volume {
				return "volume is not the default"
			}
			return ""
		},
		says: "volume",
	}, {
		name: "a negative scale",
		set:  func(p *Prefs) { p.Scale = -1 },
		want: func(p *Prefs) string {
			if p.Scale != ScaleAuto {
				return "scale is not auto"
			}
			return ""
		},
		says: "scale",
	}, {
		name: "a scale past the cap",
		set:  func(p *Prefs) { p.Scale = MaxScale + 1 },
		want: func(p *Prefs) string {
			if p.Scale != ScaleAuto {
				return "scale is not auto"
			}
			return ""
		},
		says: "scale",
	}, {
		name: "a neighbor count the renderer cannot compose",
		set:  func(p *Prefs) { p.Neighbors = 5 },
		want: func(p *Prefs) string {
			if p.Neighbors != 9 {
				return "neighbors is not 9"
			}
			return ""
		},
		says: "neighbors",
	}, {
		name: "a pause key that is neither tab nor escape",
		set:  func(p *Prefs) { p.PauseKey = "p" },
		want: func(p *Prefs) string {
			if p.PauseKey != "tab" {
				return "pause key is " + p.PauseKey
			}
			return ""
		},
		says: "pause key",
	}, {
		name: "a version from the future",
		set:  func(p *Prefs) { p.Version = Version + 40 },
		want: func(p *Prefs) string { return "" },
		says: "newer build",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			p := Default()
			tc.set(p)
			p.Validate()
			if bad := tc.want(p); bad != "" {
				t.Errorf("not repaired: %s", bad)
			}
			if len(p.Notes) == 0 {
				t.Fatal("repaired silently; a setting that reverts with no explanation is undiagnosable")
			}
			if !strings.Contains(strings.Join(p.Notes, "\n"), tc.says) {
				t.Errorf("the notes do not mention %q:\n  %s", tc.says, strings.Join(p.Notes, "\n  "))
			}
		})
	}
}

// Auto, and every scale up to the cap, is a setting and not a repair (docs/IMPROVEMENTS.md 2.1).
// Auto is the default, so a fresh install's window fits its monitor.
func TestEveryScaleIncludingAutoIsValid(t *testing.T) {
	if Default().Scale != ScaleAuto {
		t.Errorf("the default scale is %d, not auto", Default().Scale)
	}
	for scale := ScaleAuto; scale <= MaxScale; scale++ {
		p := Default()
		p.Scale = scale
		p.Validate()
		if p.Scale != scale || len(p.Notes) != 0 {
			t.Errorf("scale %d came back %d, with notes %q", scale, p.Scale, p.Notes)
		}
	}
}

// isSoundOn = (isVolume != 0), Main.c:196-199. Derived rather than stored, so a file
// cannot describe the contradiction of sound on at volume zero.
func TestVolumeZeroIsSilence(t *testing.T) {
	p := Default()
	p.Volume = 0
	p.Sound = true
	p.Validate()
	if p.Sound {
		t.Error("volume 0 with sound on: one of them is lying")
	}
	if len(p.Notes) != 0 {
		t.Errorf("the derivation is not a repair and needs no note: %v", p.Notes)
	}
}

func TestLongNamesAreClippedToTheOriginalsLimits(t *testing.T) {
	p := Default()
	p.HighName = strings.Repeat("é", 40)   // Str15, and not one byte per character
	p.HighBanner = strings.Repeat("z", 99) // Str31
	p.Validate()
	if n := len([]rune(p.HighName)); n != MaxHighName {
		t.Errorf("name is %d runes, want %d", n, MaxHighName)
	}
	if n := len([]rune(p.HighBanner)); n != MaxHighBanner {
		t.Errorf("banner is %d runes, want %d", n, MaxHighBanner)
	}
	if !strings.HasPrefix(p.HighName, "é") {
		t.Errorf("the clip cut a character in half: %q", p.HighName)
	}
}

// The XDG default, and the spec's rule that a relative $XDG_CONFIG_HOME counts as unset. The
// Go this is built with does not decide it: 1.23 uses the relative path, putting the settings
// under whatever directory the game was started from, and 1.26 refuses it, saving them nowhere.
func TestDirDefaultsUnderTheConfigDirectory(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("$XDG_CONFIG_HOME is the Linux branch")
	}
	t.Setenv("GLIDERGO_CONFIG", "")
	t.Setenv("HOME", "/home/tester")

	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	if got, err := Dir(); err != nil || got != "/tmp/xdg/glidergo" {
		t.Errorf("Dir() = %q, %v; want /tmp/xdg/glidergo", got, err)
	}
	for _, xdg := range []string{"", "relative/path", "."} {
		t.Setenv("XDG_CONFIG_HOME", xdg)
		if got, err := Dir(); err != nil || got != "/home/tester/.config/glidergo" {
			t.Errorf("with XDG_CONFIG_HOME=%q, Dir() = %q, %v; want /home/tester/.config/glidergo",
				xdg, got, err)
		}
	}
}

func TestDirHonoursTheEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLIDERGO_CONFIG", dir)
	got, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("Dir() = %q, want the override %q", got, dir)
	}
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, Name); path != want {
		t.Errorf("Path() = %q, want %q", path, want)
	}

	// Save through the override creates the directory and Load reads it back, which is
	// the whole first-run path for a player with no config directory yet.
	nested := filepath.Join(dir, "not", "yet", "there")
	t.Setenv("GLIDERGO_CONFIG", nested)
	p := Default()
	p.House = "Nemo's Market"
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	q, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if q.House != "Nemo's Market" {
		t.Errorf("Load after Save gave house %q", q.House)
	}
}

// normalize drops the fields that describe a load rather than a setting, so DeepEqual
// can be used on the rest.
func normalize(p *Prefs) Prefs {
	q := *p
	q.Notes = nil
	q.path = ""
	q.read = nil
	q.Version = Version
	return q
}
