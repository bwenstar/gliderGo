package main

// Where a setting comes from: the three layers, and the one property that has to hold for
// `make check` to mean anything -- a measurement reads nobody's preferences.
//
// These are the only tests in cmd/glidergo. Everything else here is wiring that can be seen
// to be right or is covered where the thing it wires lives (internal/prefs, internal/shell,
// internal/game). Precedence is different: it is a rule with four inputs and no single
// obvious reading, and getting it wrong makes a benchmark quietly depend on whoever ran it.

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"glidergo/internal/game"
	"glidergo/internal/prefs"
)

// TestHermeticRunsIgnoreThePlayersPreferences is the property `make check` rests on.
//
// -shot, -frames, -bench and -dump all have to produce the same bytes on any machine. If any
// of them read the preferences file, then a developer who had turned the volume down or
// picked a one-room view would get different golden images and a different frame rate from
// everyone else, and the Makefile would have to say so at every call site.
func TestHermeticRunsIgnoreThePlayersPreferences(t *testing.T) {
	// A preferences file that disagrees with the defaults about everything a measurement
	// could notice. If it is read, the assertions below fail.
	dir := t.TempDir()
	loud := prefs.Default()
	loud.Volume = 1
	loud.Scale = 4
	loud.Neighbors = 1
	loud.MusicInGame = false
	if err := loud.SaveFile(filepath.Join(dir, prefs.Name)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLIDERGO_CONFIG", dir)

	def := prefs.Default()
	for _, tc := range []struct {
		name string
		o    options
	}{
		{"-shot", options{shot: "out.png"}},
		{"-frames", options{frames: 300}},
		{"-bench", options{bench: true, frames: 300}},
		{"-dump", options{dump: "frames"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !hermetic(&tc.o) {
				t.Fatal("hermetic() is false; this run would read the player's settings")
			}
			p, canSave := loadPrefs(&tc.o)
			if canSave {
				t.Error("canSave is true for a measurement; it has nowhere it should write")
			}
			if p.Volume != def.Volume || p.Scale != def.Scale || p.Neighbors != def.Neighbors {
				t.Errorf("volume/scale/neighbors = %d/%d/%d, want this build's defaults %d/%d/%d",
					p.Volume, p.Scale, p.Neighbors, def.Volume, def.Scale, def.Neighbors)
			}
		})
	}

	// Playing is the other half of the same rule: a player who starts a game *does* get
	// their own settings, including when they name a house on the command line. -house is
	// not a measurement, and treating it as one would silently ignore a rebind.
	for _, o := range []options{{}, {house: "Slumberland"}, {two: true}} {
		if hermetic(&o) {
			t.Errorf("%+v is hermetic; a play session must read the player's settings", o)
		}
		if p, canSave := loadPrefs(&o); !canSave || p.Volume != loud.Volume {
			t.Errorf("%+v loaded volume %d, canSave %v; want %d and true",
				o, p.Volume, canSave, loud.Volume)
		}
	}
}

// TestPrefsNoneReadsAndWritesNothing: the escape hatch, and both halves of it matter. It has
// to ignore an existing file (so a golden image is reproducible on a machine that has one)
// and it has to refuse to save (so it cannot create one either).
func TestPrefsNoneReadsAndWritesNothing(t *testing.T) {
	dir := t.TempDir()
	quiet := prefs.Default()
	quiet.Volume = 2
	if err := quiet.SaveFile(filepath.Join(dir, prefs.Name)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLIDERGO_CONFIG", dir)

	o := options{prefsPath: prefsNone}
	p, canSave := loadPrefs(&o)
	if canSave {
		t.Error("canSave is true for -prefs none")
	}
	if p.Volume != prefs.Default().Volume {
		t.Errorf("volume = %d, want the default %d: -prefs none read the file",
			p.Volume, prefs.Default().Volume)
	}
}

// TestPrefsFileWinsEvenForAMeasurement: an explicit file is how a golden image of the
// settings screen gets settings to show, so it has to beat the hermetic rule rather than
// being beaten by it -- otherwise `-shot -prefs testdata/x.json` would draw the defaults and
// the test would be checking nothing.
func TestPrefsFileWinsEvenForAMeasurement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.json")
	want := prefs.Default()
	want.Volume = 3
	want.PauseKey = "escape"
	if err := want.SaveFile(path); err != nil {
		t.Fatal(err)
	}

	o := options{shot: "out.png", prefsPath: path}
	p, canSave := loadPrefs(&o)
	if !canSave {
		t.Error("canSave is false for an explicit -prefs; the file is right there")
	}
	if p.Volume != 3 || !p.EscPause() {
		t.Errorf("volume %d, EscPause %v; want 3 and true", p.Volume, p.EscPause())
	}
}

// TestMissingPrefsFileStillPlays: a file that cannot be read leaves usable settings in hand,
// because the alternative is refusing to start a game over a file the game itself wrote.
func TestMissingPrefsFileStillPlays(t *testing.T) {
	o := options{prefsPath: filepath.Join(t.TempDir(), "no", "such.json")}
	p, canSave := loadPrefs(&o)
	if p == nil {
		t.Fatal("loadPrefs returned nil settings")
	}
	if p.Volume != prefs.Default().Volume {
		t.Errorf("volume = %d, want the default %d", p.Volume, prefs.Default().Volume)
	}
	// Still saveable: the player named a path, and the reason it did not load may simply
	// be that it is not there yet.
	if !canSave {
		t.Error("canSave is false; an unreadable named file is still somewhere to write")
	}
}

// TestOverridesOnlyTouchWhatWasGiven is why the flag layer uses the set of names rather than
// comparing values against the defaults: `-volume 7` means "make it 7", and a comparison
// cannot tell that from a flag nobody typed.
func TestOverridesOnlyTouchWhatWasGiven(t *testing.T) {
	o := &options{volume: 7, scale: 1, neighbors: 9, music: true}

	// Nothing given: a file that says volume 2 and one room stays that way, even though
	// the options struct holds 7 and 9.
	p := prefs.Default()
	p.Volume, p.Neighbors, p.MusicInGame = 2, 1, false
	applyOverrides(o, p, nil)
	if p.Volume != 2 || p.Neighbors != 1 || p.MusicInGame {
		t.Errorf("an empty flag set changed the settings: volume %d, neighbors %d, music %v",
			p.Volume, p.Neighbors, p.MusicInGame)
	}

	// Given: each name moves its own setting and nothing else.
	applyOverrides(o, p, map[string]bool{"volume": true})
	if p.Volume != 7 {
		t.Errorf("volume = %d after -volume 7, want 7", p.Volume)
	}
	if p.Neighbors != 1 {
		t.Errorf("neighbors = %d; -volume moved it", p.Neighbors)
	}

	// -music is one flag over two settings, because `-music=false` plainly means silence
	// everywhere for this run.
	p.MusicInGame, p.MusicOnTitle = true, true
	o.music = false
	applyOverrides(o, p, map[string]bool{"music": true})
	if p.MusicInGame || p.MusicOnTitle {
		t.Errorf("-music=false left in-game %v and title %v", p.MusicInGame, p.MusicOnTitle)
	}
}

// TestVolumeZeroMutesThroughValidate: the sound flag is derived from the volume, both ways
// (prefs.Validate), so a flag override has to go through the same repair a hand-edited file
// does. `-volume 0` is the original's mute -- isSoundOn = (isVolume != 0) -- and a build
// where it merely turned the level down would still tick the scoreboard and play the music.
func TestVolumeZeroMutesThroughValidate(t *testing.T) {
	o := &options{volume: 0}
	p := prefs.Default()
	applyOverrides(o, p, map[string]bool{"volume": true})

	if p.Sound {
		t.Error("Sound is true at volume 0; Validate's derivation was not applied")
	}
	o.volume = 5
	applyOverrides(o, p, map[string]bool{"volume": true})
	if !p.Sound {
		t.Error("Sound is false at volume 5; the derivation only works one way")
	}
}

// TestImportRefusesToOverwrite. There is no prompt to give -- this runs before any window
// opens -- and a settings file is small and hand-editable, so telling somebody to move
// theirs aside is cheaper than inventing a backup scheme they would have to learn. What must
// not happen is losing the bindings they are using today to a file from 1994.
func TestImportRefusesToOverwrite(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "mine.json")
	mine := prefs.Default()
	mine.Volume = 4
	if err := mine.SaveFile(dst); err != nil {
		t.Fatal(err)
	}

	legacy := filepath.Join(dir, "Glider Prefs")
	if err := os.WriteFile(legacy, make([]byte, prefs.LegacySize), 0o644); err != nil {
		t.Fatal(err)
	}

	o := &options{importPrefs: legacy, prefsPath: dst}
	if err := importPrefs(o); err == nil {
		t.Fatal("importing over an existing file succeeded")
	}
	// And the existing file is untouched, which is the part that matters.
	back, err := prefs.LoadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if back.Volume != 4 {
		t.Errorf("volume = %d after a refused import, want the original 4", back.Volume)
	}

	// -prefs none has nowhere to write at all, which is an error rather than a silent
	// conversion thrown away.
	if err := importPrefs(&options{importPrefs: legacy, prefsPath: prefsNone}); err == nil {
		t.Error("-import-prefs with -prefs none succeeded; there is nowhere for it to go")
	}
}

// TestImportWritesWhereThePrefsWouldGo: with no -prefs, the destination is the configuration
// directory's own file, so that importing and then playing picks the import up. An import
// that landed somewhere else would look like it had done nothing.
func TestImportWritesWhereThePrefsWouldGo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLIDERGO_CONFIG", dir)

	legacy := filepath.Join(dir, "Glider Prefs")
	if err := os.WriteFile(legacy, make([]byte, prefs.LegacySize), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := importPrefs(&options{importPrefs: legacy}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, prefs.Name)); err != nil {
		t.Errorf("no %s in the configuration directory after an import: %v", prefs.Name, err)
	}
}

// TestEveryOptInFixIsCopiedToTheGame is the only thing that makes gameFixes' promise real.
//
// prefs.Fixes and game.Fixes are deliberately separate types -- internal/game must not
// import internal/prefs -- so the mapping between them is a hand-written struct literal,
// and a named-field literal does not fail to compile when a field is added to either side.
// It copies what it names and leaves the rest zero, which means the failure mode of adding
// a fifth fix is a setting a player can turn on that does nothing whatsoever, with no
// compiler error, no test failure and no log line.
//
// So this compares the two field lists by reflection and then checks that every prefs field
// set true arrives true. It is the reason a new fix cannot be half-wired. Both halves are
// needed: the name comparison catches a field added to one side only, and the copy check
// catches a field added to both sides and forgotten here.
func TestEveryOptInFixIsCopiedToTheGame(t *testing.T) {
	pt := reflect.TypeOf(prefs.Fixes{})
	gt := reflect.TypeOf(game.Fixes{})

	names := func(t reflect.Type) []string {
		var out []string
		for i := 0; i < t.NumField(); i++ {
			out = append(out, t.Field(i).Name)
		}
		return out
	}
	pn, gn := names(pt), names(gt)
	if !reflect.DeepEqual(pn, gn) {
		t.Fatalf("prefs.Fixes has %v and game.Fixes has %v; the two must carry the same "+
			"fixes under the same names, or gameFixes cannot be checked at all", pn, gn)
	}

	// Every field true. Booleans only, which is what the type is for -- a non-boolean fix
	// would need this test rewritten, and the fatal below says so rather than skipping it.
	all := prefs.Fixes{}
	v := reflect.ValueOf(&all).Elem()
	for i := 0; i < pt.NumField(); i++ {
		if v.Field(i).Kind() != reflect.Bool {
			t.Fatalf("prefs.Fixes.%s is %s, not a bool; this test only knows how to set "+
				"booleans", pt.Field(i).Name, v.Field(i).Kind())
		}
		v.Field(i).SetBool(true)
	}

	got := reflect.ValueOf(gameFixes(all))
	for i := 0; i < gt.NumField(); i++ {
		if !got.Field(i).Bool() {
			t.Errorf("gameFixes leaves %s false with every preference set: the fix is in "+
				"both structs and is not copied, so turning it on does nothing",
				gt.Field(i).Name)
		}
	}

	// And the other direction, so that a copy written `true` by mistake is caught too.
	if zero := gameFixes(prefs.Fixes{}); zero != (game.Fixes{}) {
		t.Errorf("gameFixes(zero) = %+v, want the zero value: every fix defaults to the "+
			"original's behaviour", zero)
	}
}
