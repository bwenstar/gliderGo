// Package prefs is the player's saved settings: which house, which keys, how loud,
// and which of the 1994 bugs to keep.
//
// # What it replaces
//
// Glider PRO kept a 226-byte `prefsInfo` record in a resource in a file called
// "Glider Prefs" in the System Folder (`Prefs.c`, and docs/analysis/ui-dialogs.md 6.1
// has the byte-by-byte layout). Four things about that arrangement are worth naming,
// because this package is a deliberate answer to each:
//
//   - **It was versioned by exact equality, and a mismatch deleted the file.**
//     `LoadPrefs` compares the stored `prefVersion` against `kPrefsVersion` (0x0034)
//     with `!=`, puts up ALRT 160 and calls `FSpDelete` (Prefs.c). So every update that
//     touched the record threw away every setting the player had -- including their
//     high-score name. There is no migration path in the original at all. Here the
//     version is advisory: an older file is read for whatever fields it has, a newer one
//     is read for the fields this build knows, and nothing is ever deleted.
//
//   - **A single unreadable field lost the whole record.** A fixed binary struct is all
//     or nothing. This is JSON decoded over a struct pre-filled with defaults, so a
//     field that is missing takes its default, a field that is unknown is ignored, and a
//     field whose *value* is wrong is repaired one field at a time with a note saying
//     what happened (see Validate and Prefs.Notes).
//
//   - **It stored keys twice, and the two could disagree.** The record holds four
//     `Str15` display names at offsets 33..96 *and* four raw KeyMap bit offsets at
//     146..161. The names came from `GetKeyName`, which asks the current keyboard layout
//     what character a physical key produces, so a prefs file written on an AZERTY Mac
//     held "A" beside a bit offset that means Q. Here a binding is one canonical name
//     for the physical key, and internal/platform's KeyName is both what the file stores
//     and what the settings screen shows.
//
//   - **Two of its 226 bytes were never initialised.** Offsets 145 and 225 are structure
//     padding that the original writes straight out of an uninitialised local, so the
//     same settings produced different bytes on different runs. legacy.go reads that
//     layout and writes defined bytes for both.
//
// # What is not here
//
// The original's editor preferences -- the map window's position, the tool palette, the
// nine `was*` editor fields -- are read by ImportLegacy and dropped. There is no editor
// until stage 5, and inventing a home for settings nothing can honour would be worse
// than losing them: a player who sees "Zoom Windows" on a settings screen that does not
// zoom has been lied to. The fields are named in legacy.go so the gap is visible.
package prefs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bwenstar/gliderGo/internal/platform"
)

// Version is this schema's version. It goes in the file so a future build can tell an
// older one apart, and unlike the original's `kPrefsVersion` a difference is never
// fatal in either direction: Validate reports it and reads what it can.
//
// Bump it when a field changes *meaning*. Adding a field needs no bump, because an
// absent field already takes its default.
const Version = 1

// Name is the file's name inside the configuration directory.
const Name = "prefs.json"

// Controls is one player's four keys, exactly the four the original binds per glider
// (`gliderType.leftKey` and friends, InterfaceInit.c:139-151).
//
// They are names rather than platform.Keys because this struct is the file's shape and
// because the file has to be readable and editable by hand. Validate guarantees the
// invariant everything else relies on: **after a successful Load every one of these is
// a canonical name that platform.ParseKey accepts.** So the settings screen can display
// the string as it stands and store `platform.KeyName(k)` when the player rebinds.
type Controls struct {
	Left  string `json:"left"`
	Right string `json:"right"`
	Batt  string `json:"battery"`
	Band  string `json:"band"`
}

// Binding is Controls resolved: what the game is actually polled for.
type Binding struct {
	Left, Right, Batt, Band platform.Key
}

// Fixes are the 1994 defects a player may choose to have corrected.
//
// All four default to **false**, which is to say the original's behaviour, and that is
// not timidity: stage 1.8's fidelity work compares this port's sparkle tables and dirty
// rects against the C's, so a build that quietly fixed them could not be checked
// against the thing it is a port of. They are offered here, one flag each, so a player
// who wants the game rather than the artefact can have it -- see docs/IMPROVEMENTS.md
// 2.19, 2.20, 2.39 and 2.23, each of which is one line of game code behind its flag.
type Fixes struct {
	// MirrorFlame clips the reflected glider's back rect to the mirrors, so a candle
	// flame or a pendulum sharing that rect stops blinking (2.19).
	MirrorFlame bool `json:"mirror_flame"`

	// MirrorFoil adds the `!twoPlayerGame` guard DrawReflection is missing, so a
	// mirror in a two-player game stops drawing player one's reflection with player
	// two's foil sheet (2.20).
	MirrorFoil bool `json:"mirror_foil"`

	// SwitchSparkle drops the second, unpositioned AddSparkle in the switch's prize
	// arm. It fires 145 times across the 22 shipped houses and puts a stray puff of
	// light in the corner of the play area every time (2.39).
	SwitchSparkle bool `json:"switch_sparkle"`

	// Player2GiveUp gives the abandon key (Delete) to player 2 as well as player 1, so
	// that either player can break a two-player deadlock rather than only the one whose
	// keyboard half it is. The one fix in this struct that is about what a player can
	// *do* rather than what they can see (2.23).
	Player2GiveUp bool `json:"player2_give_up"`
}

// Prefs is the whole of what is saved.
//
// Every field's default is the original's, where the original had one, and the two
// places that deviate say so on the field. The zero value is *not* usable -- use
// Default() -- because half of these settings are booleans whose useful default is
// true and a forgotten initialisation would silently turn off the sound.
type Prefs struct {
	// Version is the schema this file was written by. See Version.
	Version int `json:"version"`

	// House is the house to open, by name and without its extension. The original
	// stores the same thing in `wasDefaultName` and ships "Slumberland" in it.
	// An empty string means "let the shell choose", which is what a player who has
	// deleted their house gets rather than an error.
	House string `json:"house"`

	// Player1 and Player2 are the eight bindings. Player one's default is the
	// original's -- the four arrows -- and player two's is **not**: the original binds
	// Control, Command, Option and Shift (InterfaceInit.c:148-151), which a modern
	// window manager intercepts and many keyboards cannot report independently, so the
	// port's default is A/D/S/W. docs/IMPROVEMENTS.md 2.3, and the whole reason this
	// file exists is that a player who wants the 1994 bindings can now have them.
	Player1 Controls `json:"player1"`
	Player2 Controls `json:"player2"`

	// PauseKey is "tab" or "escape", the original's `isEscPauseKey` as the name of the
	// key rather than a boolean. Its default is tab, because `isEscPauseKey` is false
	// at Main.c:184 and because in this port Escape leaves the game.
	PauseKey string `json:"pause_key"`

	// Scale is the integer magnification of the 640x480 image, applied by the backend
	// and nowhere else (docs/IMPROVEMENTS.md 2.8). The original had no equivalent: its
	// window was one screen pixel per game pixel on a 640x480 display.
	Scale int `json:"scale"`

	// Neighbors is `numNeighbors`: 1, 3 or 9 rooms composed around the player. The
	// original's default is 9, forced to 1 on a screen 512 pixels wide or narrower.
	Neighbors int `json:"neighbors"`

	// Sound is `isSoundOn` and Volume is `isVolume`, 0 to 7. The original derives the
	// first from the second (`isSoundOn = (isVolume != 0)`, Main.c) and takes the
	// volume from the *system* setting clamped to 1..3 on a fresh install; a port has
	// no business reading the desktop's master volume, so the default here is the
	// scale's top.
	Sound  bool `json:"sound"`
	Volume int  `json:"volume"`

	// MusicInGame is `isPlayMusicGame` and MusicOnTitle is `isPlayMusicIdle`. Two
	// flags rather than one because they are two choices: the score during play and
	// the score on the title screen are different pieces of music in different places
	// and a player can want one without the other. Both default true.
	MusicInGame  bool `json:"music_in_game"`
	MusicOnTitle bool `json:"music_on_title"`

	// PauseWhenUnfocused is the original's `doBackground`, the Brains pane's
	// "Background Tasks" (docs/analysis/ui-dialogs.md 6.11) -- the flag PlayGame checks
	// before it stops simulating while the window is not frontmost. Its name is inverted
	// from the original's because the original's is about 1994's cooperative
	// multitasking and this one is about what happens to your glider when you alt-tab.
	//
	// **Its default is inverted too**, and this is a deliberate deviation: the original
	// defaults it false, so a player who switched away kept falling. On a system where
	// switching away is one keystroke and happens by accident, coming back to a dead
	// glider reads as a bug. A player who wants the 1994 behaviour sets this false, and
	// an imported 1994 file gets whatever it actually said.
	PauseWhenUnfocused bool `json:"pause_when_unfocused"`

	// KeepRealTime chooses what happens when a frame overruns: false is the
	// original's, which is to let the game run slower and lose no animation, and true
	// skips ahead to keep wall-clock time. docs/IMPROVEMENTS.md 2.17 -- the original
	// reseeds its deadline from the clock after the wait, so it has no catch-up at
	// all, and that is the right default but not the only defensible one.
	//
	// **Nothing reads it yet.** The limiter is still the original's no-catch-up form
	// (internal/game's awaitFrame), because a catch-up that is worth having needs a
	// resync clamp -- otherwise the first long stall, a room wipe or a pause, is followed
	// by a burst of unpaced frames -- and that is frame-pacing work, which belongs with
	// 1.8's timing pass. The field is here so that an imported file and a hand-edited
	// one have somewhere to say it, and it is the one setting in this struct that is
	// declared ahead of its implementation.
	KeepRealTime bool `json:"keep_real_time"`

	// HighName and HighBanner are `wasHighName` and `wasHighBanner`: the name a new
	// high score is entered under and the one-line message that goes with it. They
	// live in preferences and not in the house, which is why they are here -- the
	// board is per house (1.7c) and the identity is per player.
	HighName   string `json:"high_name"`
	HighBanner string `json:"high_banner"`

	// Fixes are the opt-in corrections. See Fixes.
	Fixes Fixes `json:"fixes"`

	// Notes is what Load had to repair, in the order it found it. Not saved: it
	// describes one load of one file. A caller shows these to whoever is watching --
	// stderr, or the shell's status line -- because a setting that has silently
	// reverted is indistinguishable from one that never saved.
	Notes []string `json:"-"`

	// path is where this came from, so Save can put it back without being told.
	path string
}

// Default is the settings a player who has never opened the game has.
//
// The values are the original's no-prefs block (Main.c's `else` arm of ReadInPrefs)
// wherever the original has an opinion this port can honour. Three deviations, each
// argued on its field: player two's bindings, the volume, and Scale, which is new.
func Default() *Prefs {
	return &Prefs{
		Version: Version,
		House:   "Slumberland",
		Player1: Controls{Left: "left", Right: "right", Batt: "down", Band: "up"},
		Player2: Controls{Left: "a", Right: "d", Batt: "s", Band: "w"},

		PauseKey:  "tab",
		Scale:     1,
		Neighbors: 9,

		Sound:        true,
		Volume:       7,
		MusicInGame:  true,
		MusicOnTitle: true,

		PauseWhenUnfocused: true,
		KeepRealTime:       false,

		// The original's own placeholders (Main.c:132-133). They are placeholders on
		// purpose: they appear on the shipped high-score boards, so a board that still
		// says "Your Name" is telling the truth about never having been played.
		HighName:   "Your Name",
		HighBanner: "Your Message Here",
	}
}

// Dir is the directory the preferences file lives in.
//
// os.UserConfigDir is the native answer on every platform this port targets --
// $XDG_CONFIG_HOME or ~/.config on Linux, ~/Library/Application Support on macOS,
// %AppData% on Windows -- which is the modern equivalent of the original's "a file in
// the System Folder". GLIDERGO_CONFIG overrides it wholesale, for a portable install
// on a memory stick and for anybody who does not want a game writing under ~/.config.
func Dir() (string, error) {
	if d := os.Getenv("GLIDERGO_CONFIG"); d != "" {
		return d, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("prefs: no configuration directory: %w", err)
	}
	return filepath.Join(base, "glidergo"), nil
}

// Path is the preferences file.
func Path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, Name), nil
}

// Load reads the preferences file, or returns the defaults if there is not one yet.
//
// A missing file is the ordinary first run and not an error. Anything else that goes
// wrong -- an unreadable file, a syntax error, a value out of range -- leaves usable
// settings in hand: the return is never nil, and what could not be honoured is in
// Notes. That is the whole contract, and it is the one the original did not offer.
func Load() (*Prefs, error) {
	path, err := Path()
	if err != nil {
		p := Default()
		p.Notes = append(p.Notes, err.Error())
		return p, err
	}
	return LoadFile(path)
}

// LoadFile is Load from a named file, for the -prefs flag and for tests.
func LoadFile(path string) (*Prefs, error) {
	p := Default()
	p.path = path

	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// The first run. Not a note either: there is nothing wrong and nothing to
		// tell anybody about.
		return p, nil
	case err != nil:
		p.Notes = append(p.Notes, fmt.Sprintf("cannot read %s (%v); using defaults", path, err))
		return p, err
	}

	// Decoded over the defaults, so an absent field keeps its default and an unknown
	// one is ignored. encoding/json reports the *first* type error and carries on
	// decoding the rest, which is exactly the behaviour wanted here: one bad field
	// costs that field and nothing else.
	if err := json.Unmarshal(data, p); err != nil {
		if syntax := new(json.SyntaxError); errors.As(err, &syntax) {
			// A file that is not JSON at all cannot be repaired field by field, and
			// silently overwriting it on the next save would destroy settings the
			// player may have spent time on. So it is moved aside, named, and kept.
			// The original's answer to an unreadable prefs file is FSpDelete.
			bad := path + ".bad"
			note := fmt.Sprintf("%s is not valid JSON (%v); using defaults", path, err)
			if mvErr := os.Rename(path, bad); mvErr == nil {
				note += "; the old file is at " + filepath.Base(bad)
			}
			q := Default()
			q.path = path
			q.Notes = append(q.Notes, note)
			return q, nil
		}
		// A type error, and the fields around it survived. Say which one.
		p.Notes = append(p.Notes, fmt.Sprintf("%s: %v; that setting is back to its default", path, err))
	}

	p.Validate()
	return p, nil
}

// Save writes the file, creating its directory. It is atomic: the bytes go to a
// temporary file in the same directory and are renamed over the target, so a crash or a
// full disk mid-write leaves the old settings intact rather than a truncated file. The
// original wrote its resource in place.
func (p *Prefs) Save() error {
	path := p.path
	if path == "" {
		var err error
		if path, err = Path(); err != nil {
			return err
		}
		p.path = path
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	p.Version = Version
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, Name+".tmp*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// SaveFile writes to a named file, for the -prefs flag and for tests.
func (p *Prefs) SaveFile(path string) error {
	p.path = path
	return p.Save()
}

// Path is where this set of preferences was loaded from and will be saved to.
func (p *Prefs) Path() string { return p.path }

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// Validate repairs anything a file may hold that the game cannot use, appending one
// note per repair. It is idempotent and it never fails: the point is that Load's caller
// always has settings it can run with.
//
// The order matters in one place. The pause key is resolved before the eight bindings,
// because a binding that collides with it is the one that gives way -- a player who has
// bound Tab to their left thruster cannot pause, and losing pause is worse than losing
// one thruster, which is why the original reserves Tab, Escape and Delete from binding
// outright (docs/analysis/ui-dialogs.md 6.10).
func (p *Prefs) Validate() {
	d := Default()

	if p.Version > Version {
		p.note("written by a newer build (version %d, this is %d); settings this build does not know are kept as they are",
			p.Version, Version)
	}

	// The pause key: the original offers exactly these two and so does this.
	switch p.PauseKey {
	case "tab", "escape":
	default:
		p.note("pause key %q is not tab or escape; using %s", p.PauseKey, d.PauseKey)
		p.PauseKey = d.PauseKey
	}
	pause, _ := platform.ParseKey(p.PauseKey)

	// The eight bindings. Reserved keys first, then duplicates, and both sides of a
	// clash are named in the note so a player can see what to change.
	reserved := map[platform.Key]string{
		pause:               "the pause key",
		platform.KeyEscape:  "the key that leaves a game",
		platform.KeyDelete:  "the key that gives up a waiting glider",
		platform.KeyReturn:  "the key that confirms",
		platform.KeyUnknown: "not a key",
	}
	taken := map[platform.Key]string{}
	for _, who := range []struct {
		name string
		c    *Controls
		def  *Controls
	}{
		// Player one first, and the order is load-bearing: whoever is checked first
		// keeps the key. Player one's bindings are the ones an import brings across and
		// the ones a single player uses, so they outrank player two's -- which, at their
		// defaults, are this port's invention rather than anybody's choice.
		{"player one", &p.Player1, &d.Player1},
		{"player two", &p.Player2, &d.Player2},
	} {
		for _, f := range controlFields {
			got := f.get(who.c)
			def := f.get(who.def)
			defKey, _ := platform.ParseKey(def)

			// Deliberately unbound. A legitimate state, not a fault, so no note: it is
			// where a collision that could not be resolved ends up, and re-validating a
			// file that has one must be silent or every launch would repeat the
			// complaint.
			if got == "" || got == Unbound {
				f.set(who.c, Unbound)
				continue
			}

			k, ok := platform.ParseKey(got)
			if !ok {
				p.note("%s's %s key %q is not a key on this build; using %s", who.name, f.label, got, def)
				f.set(who.c, def)
				k = defKey
			}
			if why, bad := reserved[k]; bad {
				p.note("%s's %s key cannot be %s -- it is %s; using %s", who.name, f.label, platform.KeyName(k), why, def)
				f.set(who.c, def)
				k = defKey
			}

			// Two controls on one key. The original refuses the second binding with a
			// beep and keeps the dialog open (docs/analysis/ui-dialogs.md 6.10); a file
			// has nobody to beep at, so the second claimant falls back to its own
			// default, and if that is taken as well it is left **unbound**.
			//
			// Unbound rather than shifted onto some third key nobody asked for: a dead
			// control is visible on the settings screen and fixable in one keystroke,
			// while an invented one is a key that does something the player never chose
			// and cannot guess. It also terminates -- an invented binding can collide
			// again on the next load and walk from key to key.
			if prev, dup := taken[k]; dup {
				_, defReserved := reserved[defKey]
				_, defTaken := taken[defKey]
				if k != defKey && !defTaken && !defReserved {
					p.note("%s's %s key is %s, which is already %s; using %s",
						who.name, f.label, platform.KeyName(k), prev, def)
					f.set(who.c, def)
					k = defKey
				} else {
					p.note("%s's %s key is %s, which is already %s, and %s is taken too; "+
						"it is unbound until you set it on the settings screen",
						who.name, f.label, platform.KeyName(k), prev, def)
					f.set(who.c, Unbound)
					continue
				}
			}
			taken[k] = who.name + "'s " + f.label + " key"
		}
	}

	if p.Volume < 0 || p.Volume > MaxVolume {
		p.note("volume %d is outside 0..%d; using %d", p.Volume, MaxVolume, d.Volume)
		p.Volume = d.Volume
	}
	// `isSoundOn = (isVolume != 0)` (Main.c). Kept as a derivation rather than an
	// independent flag, because a saved file with sound on and volume zero describes
	// silence either way and a settings screen that shows both would be showing a
	// contradiction.
	//
	// Derived in **both** directions, which matters more than it looks: a one-way rule
	// that only ever clears the flag turns "the player slid the volume to zero and back
	// up" into permanent silence, because nothing would ever set it again. Volume is the
	// setting a player touches; this follows it.
	p.Sound = p.Volume != 0

	if p.Scale < 1 || p.Scale > MaxScale {
		p.note("scale %d is outside 1..%d; using %d", p.Scale, MaxScale, d.Scale)
		p.Scale = d.Scale
	}
	switch p.Neighbors {
	case 1, 3, 9:
	default:
		p.note("neighbors %d is not 1, 3 or 9; using %d", p.Neighbors, d.Neighbors)
		p.Neighbors = d.Neighbors
	}

	// The two strings that reach the screen. The original's are Str15 and Str31 and
	// cannot overflow; these can, and a 4000-character banner would draw over the
	// whole high-score board.
	p.HighName = clip(p.HighName, MaxHighName)
	p.HighBanner = clip(p.HighBanner, MaxHighBanner)
}

// Unbound is what a binding says when there is no key on it: a control the player can
// see is unset rather than one that quietly does nothing.
//
// It exists because a collision has to end somewhere. It is also what the settings
// screen writes when a player clears a binding -- not platform.KeyName(KeyUnknown),
// which is "unknown" and reads like a fault, and which Validate treats as one.
const Unbound = "none"

// The limits, named because both the settings screen and Validate need them.
const (
	// MaxVolume is the original's 0..7 scale (UnivSetSoundVolume clamps to it).
	MaxVolume = 7

	// MaxScale is a cap on the window rather than a fidelity matter: 8x of 640x480 is
	// 5120x3840, past any display this will run on, and a mistyped 800 should not ask
	// the backend for a 4-gigabyte surface.
	MaxScale = 8

	// The original's Str15 wasHighName and Str31 wasHighBanner, in runes.
	MaxHighName   = 15
	MaxHighBanner = 31
)

// controlFields is the four bindings as data, so Validate and the settings screen walk
// the same list in the same order and neither can forget one.
var controlFields = []struct {
	label string
	get   func(*Controls) string
	set   func(*Controls, string)
}{
	{"left", func(c *Controls) string { return c.Left }, func(c *Controls, s string) { c.Left = s }},
	{"right", func(c *Controls) string { return c.Right }, func(c *Controls, s string) { c.Right = s }},
	{"battery", func(c *Controls) string { return c.Batt }, func(c *Controls, s string) { c.Batt = s }},
	{"band", func(c *Controls) string { return c.Band }, func(c *Controls, s string) { c.Band = s }},
}

func (p *Prefs) note(format string, a ...any) {
	p.Notes = append(p.Notes, fmt.Sprintf(format, a...))
}

// clip shortens a string to n runes. Runes and not bytes: a name typed in a language
// this port will meet is not one byte per character, and half a UTF-8 sequence draws as
// a replacement glyph.
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// ---------------------------------------------------------------------------
// Reading the settings back out
// ---------------------------------------------------------------------------

// Keys resolves one player's four bindings. Unbound answers KeyUnknown, which no
// backend ever reports as held, so the control is dead rather than wrong; so does any
// name that will not parse, which after Validate should not happen but must not become
// a wrong key if it does.
func (c Controls) Keys() Binding {
	k := func(s string) platform.Key {
		key, _ := platform.ParseKey(s)
		return key
	}
	return Binding{Left: k(c.Left), Right: k(c.Right), Batt: k(c.Batt), Band: k(c.Band)}
}

// Pause is the pause key as a key.
func (p *Prefs) Pause() platform.Key {
	k, _ := platform.ParseKey(p.PauseKey)
	return k
}

// EscPause is the original's `isEscPauseKey`, which is what decides between the two
// pause overlays (PICT 1015 and 1016) as well as which key pauses.
func (p *Prefs) EscPause() bool { return p.PauseKey == "escape" }
