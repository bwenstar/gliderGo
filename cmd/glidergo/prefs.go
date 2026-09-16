package main

// Where a setting comes from, and which of three answers wins.
//
// There are three sources and they are tried in this order, latest wins:
//
//  1. **This build's defaults** (prefs.Default). What somebody who has never run the
//     game gets, and what a measurement gets on purpose -- see hermetic.
//  2. **The preferences file** (prefs.Load, or -prefs). The player's own settings, and
//     the only one of the three that is remembered.
//  3. **A flag actually given on the command line** (overrideFromFlags). For this run
//     only, and detected with flag.Visit rather than by comparing against the default,
//     because `-volume 7` means "make it 7" and a comparison cannot tell it from silence.
//
// The original has two of the three: ReadInPrefs and its no-prefs `else` arm (Main.c). It
// has no command line at all, so the third layer is the port's, and it is the layer that
// makes `make check` reproducible on a machine whose owner has been playing the game.
//
// # What is not written back
//
// Nothing here saves. Saving happens in exactly two places -- the settings screen closing
// (internal/shell/settings.go) and runShell noticing that the player chose a different
// house -- because those are the two moments a player changed a setting *in the game*. The
// original saves once, at quit (WriteOutPrefs, Main.c:381), which loses everything if it
// crashes; saving at the change costs one file write and cannot lose one.
//
// A flag override does land in the file if the player then changes something in the
// settings screen, because the screen shows the session's settings and saving writes what
// the screen shows. That is the honest behaviour: the alternative is a screen showing
// volume 2 that saves volume 7.

import (
	"flag"
	"fmt"
	"os"

	"glidergo/internal/game"
	"glidergo/internal/prefs"
)

// prefsNone is what -prefs takes to mean "the defaults, and read and write nothing". It
// is a word rather than an empty string because an empty -prefs is a mistyped path and
// should not silently become a different mode.
const prefsNone = "none"

// hermetic says this run is a measurement rather than a play session.
//
// A screenshot, a timed run, a benchmark and a frame dump are all things a Makefile or a
// test does, and every one of them has to produce the same bytes on any machine. Reading
// the player's preferences would make `make check` depend on whether whoever ran it has
// ever turned the volume down -- so a measurement starts from the defaults unless it is
// pointed at a file explicitly, which is what makes `make check` hermetic with no special
// pleading in the Makefile.
//
// -house is deliberately not in this list. Naming a house is asking to play, and a player
// who plays that way wants their own key bindings.
func hermetic(o *options) bool {
	return o.shot != "" || o.frames > 0 || o.bench || o.dump != ""
}

// loadPrefs answers with the settings and whether they can be saved.
//
// It never fails. A preferences file that cannot be read, or a configuration directory
// that does not exist, leaves usable settings in hand and a note on stderr -- because the
// alternative is refusing to start a game over a file the game itself wrote.
//
// The second return is "is there anywhere to put these": false for the defaults-only
// modes, which the shell turns into a settings screen that says so on the way out rather
// than one that pretends to have saved.
func loadPrefs(o *options) (*prefs.Prefs, bool) {
	switch {
	case o.prefsPath == prefsNone:
		return prefs.Default(), false

	case o.prefsPath != "":
		// An explicit file wins even for a measurement: `-shot -prefs testdata/x.json`
		// is how a golden screenshot of the settings screen gets settings to show.
		p, err := prefs.LoadFile(o.prefsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "glidergo: %v\n", err)
		}
		return p, true

	case hermetic(o):
		return prefs.Default(), false
	}

	p, err := prefs.Load()
	if err != nil {
		// Load's contract is that p is usable anyway. The commonest cause is a machine
		// with no configuration directory, where the game still plays and only the
		// remembering is lost.
		fmt.Fprintf(os.Stderr, "glidergo: %v\n", err)
	}
	return p, true
}

// overrideFromFlags lays the flags that were actually given over the loaded settings.
//
// Four flags name a preference: -volume, -scale, -neighbors and -music. The rest either
// have no preference behind them (-house, -room, -seed, the paths) or are about the
// machine rather than the player (-audio, -wav, -sound, which is the original's
// dontLoadSounds and decides whether the bank is loaded at all, not how loud it is).
//
// Validate runs at the end rather than each field being checked here, so that a flag and
// a hand-edited file are repaired by the same code -- including the sound-follows-volume
// derivation, which is why `-volume 0` mutes rather than merely turning the level down.
func overrideFromFlags(o *options, p *prefs.Prefs) {
	given := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { given[f.Name] = true })
	applyOverrides(o, p, given)
}

// applyOverrides is the half of it that does not need a command line: which flags were
// given, as a set of names. Separated so it can be tested -- the flag package's default
// CommandLine is process-wide and can only be parsed once, so a test that went through
// overrideFromFlags could check exactly one case.
func applyOverrides(o *options, p *prefs.Prefs, given map[string]bool) {
	if given["volume"] {
		p.Volume = o.volume
	}
	if given["scale"] {
		p.Scale = o.scale
	}
	if given["neighbors"] {
		p.Neighbors = o.neighbors
	}
	if given["music"] {
		// One flag over both preferences. They are two settings and a player can want one
		// without the other (prefs.Prefs), but `-music=false` plainly means silence
		// everywhere for this run.
		p.MusicInGame, p.MusicOnTitle = o.music, o.music
	}
	p.Validate()
}

// reportPrefsNotes prints what the preferences layer had to repair, and forgets it.
//
// Notes are how prefs.Load says "your file asked for something I could not do" without
// failing, and stderr is the only place a player who has hand-edited the file will think
// to look. Cleared afterwards because the settings screen reports its *own* notes on the
// status line by watching the slice grow (internal/shell/settings.go), and a stale note
// from launch would be a puzzle rather than a message.
func reportPrefsNotes(p *prefs.Prefs) {
	for _, n := range p.Notes {
		fmt.Fprintf(os.Stderr, "glidergo: %s\n", n)
	}
	p.Notes = nil
}

// gameFixes copies the opt-in corrections from the player's settings into the game's own
// struct, because internal/game must not import internal/prefs -- the game has no
// preferences, it has a caller that had some.
//
// A function rather than a struct literal at the one call site so that
// TestEveryOptInFixIsCopiedToTheGame can reach it. A named-field literal does *not* fail to
// compile when a field is added to either side; it silently copies three of four, and the
// setting then does nothing at all, which is the worst way for a preference to be broken.
// The test compares the two field lists by reflection and is the only thing that actually
// enforces the mapping.
func gameFixes(f prefs.Fixes) game.Fixes {
	return game.Fixes{
		MirrorFlame:   f.MirrorFlame,
		MirrorFoil:    f.MirrorFoil,
		SwitchSparkle: f.SwitchSparkle,
		Player2GiveUp: f.Player2GiveUp,
	}
}

// importPrefs is the -import-prefs one-shot: read a 1994 "Glider Prefs" file, convert it,
// and write it where this port keeps its own.
//
// It exists because the settings are the one part of a 1994 installation that is *about
// the player* rather than about the game -- eight key bindings somebody chose and got used
// to. internal/prefs/legacy.go does the conversion and lists every dropped field with a
// reason; this only decides where the result goes and refuses to destroy anything.
//
// A destination that already exists is an error rather than a backup or a prompt. There is
// no prompt to give -- this runs before any window opens -- and a settings file is small
// and hand-editable, so telling somebody to move theirs aside is cheaper than inventing a
// backup scheme they would then have to learn.
func importPrefs(o *options) error {
	p, err := prefs.ImportLegacyFile(o.importPrefs)
	if err != nil {
		return err
	}

	dst := o.prefsPath
	switch dst {
	case prefsNone:
		return fmt.Errorf("-import-prefs has nowhere to write with -prefs %s", prefsNone)
	case "":
		if dst, err = prefs.Path(); err != nil {
			return err
		}
	}
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("%s already exists; move it aside, or use -prefs to write somewhere else", dst)
	}

	for _, n := range p.Notes {
		fmt.Fprintf(os.Stderr, "glidergo: %s\n", n)
	}
	if err := p.SaveFile(dst); err != nil {
		return err
	}

	fmt.Printf("glidergo: imported %s -> %s\n", o.importPrefs, dst)
	fmt.Printf("glidergo: player one %s/%s/%s/%s, player two %s/%s/%s/%s, %s pauses\n",
		p.Player1.Left, p.Player1.Right, p.Player1.Band, p.Player1.Batt,
		p.Player2.Left, p.Player2.Right, p.Player2.Band, p.Player2.Batt,
		p.PauseKey)
	return nil
}
