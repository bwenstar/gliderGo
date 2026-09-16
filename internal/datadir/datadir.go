// Package datadir answers two questions that every kind of player data has to answer the
// same way: where does it live, and what is it called.
//
// It exists because there is now more than one kind. internal/scores had both answers to
// itself since 1.7c; 1.10's saved games need exactly the same ones, and a second copy of a
// seventy-line file-name escaper is a second thing to get wrong. The escaper in particular
// has to be *identical* between the two: a house whose board is Art%20Museum.scores and
// whose save is "Art Museum.save" would be a bug nobody would notice until they moved the
// directory to Windows.
//
// The original had none of this. 1994 had one place for everything -- the Preferences
// folder -- and a high-score board and a saved game went in the same drawer under the
// house's own name, because HFS accepted every byte but ':' (see internal/scores' notes on
// HighScores.c:723 and docs/analysis/scoring.md 7.14, 9.2).
package datadir

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// AppDir is the game's own directory component, used under whichever base directory the
// platform hands us. Lower case and one word so it is the same string on all three.
const AppDir = "glidergo"

// Dir is where a kind of player data goes; sub is the subdirectory that names the kind
// ("scores", "saves").
//
// Both are *data*, not configuration: the player did not choose them, cannot usefully edit
// them, and would not expect them in the same place as a settings file. On Linux the XDG
// basedir spec says that means $XDG_DATA_HOME (~/.local/share), not $XDG_CONFIG_HOME, and
// the distinction is not pedantry -- a dotfile-syncing setup that tracks ~/.config should
// not be picking up a game's high scores or its saves. Go has os.UserConfigDir and
// os.UserCacheDir and no os.UserDataDir, so the Linux path is resolved by hand and every
// other platform falls back to the config directory, which is where its own conventions put
// both (~/Library/Application Support, %AppData%).
//
// Two overrides, in order:
//
//	GLIDERGO_DATA     used verbatim, for a portable install and for tests
//	GLIDERGO_CONFIG   if set, the data goes under it, so that a player who has pointed the
//	                  whole game at one directory gets one directory
//
// The second is prefs.Dir's variable and is honoured here on purpose: somebody who runs
// `GLIDERGO_CONFIG=./glider-data glidergo` has said where they want the game's files, and
// answering that with settings in one place and scores in another would be a bug.
//
// GLIDERGO_DATA is deliberately *not* given a sub component. It is the portable-install
// escape hatch, and a USB stick with Titanic.scores next to Titanic.save is easier to
// reason about than one with two near-empty subdirectories. The two kinds cannot collide
// because their extensions differ.
func Dir(sub string) (string, error) {
	if d := os.Getenv("GLIDERGO_DATA"); d != "" {
		return d, nil
	}
	if d := os.Getenv("GLIDERGO_CONFIG"); d != "" {
		return filepath.Join(d, sub), nil
	}
	if runtime.GOOS == "linux" {
		if d := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(d) {
			return filepath.Join(d, AppDir, sub), nil
		}
		// $XDG_DATA_HOME unset, or set to a relative path, which the spec says to treat as
		// unset. ~/.local/share is the spec's own default.
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".local", "share", AppDir, sub), nil
		}
		// No home directory either. Fall through rather than fail: os.UserConfigDir has
		// its own answer and its own error message.
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("no data directory for %s: %w", sub, err)
	}
	return filepath.Join(base, AppDir, sub), nil
}

// ---------------------------------------------------------------------------
// House name -> file name
// ---------------------------------------------------------------------------

// SafeByte is the set a file name may hold unescaped. Deliberately narrow: it is the
// intersection of what Linux, macOS and Windows all accept in any position, so the same
// house produces the same file name on all three and a data directory can be copied
// between them.
func SafeByte(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '_', c == '.':
		return true
	}
	return false
}

// maxBase is how long an escaped name may be before it is shortened. 100 leaves room for
// the extension and the hash inside every filesystem limit worth caring about (255 bytes on
// ext4 and APFS, 255 UTF-16 units on NTFS).
const maxBase = 100

// reserved is the Windows device names, which are refused *with any extension* and in any
// case, so "CON.scores" is as unopenable as "CON". A house called Con is not likely; a
// port that only works for likely house names is not finished.
var reserved = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// Reserved reports whether an escaped base name is one Windows refuses. Exported for the
// tests, which assert that every name FileName can emit is openable.
func Reserved(base string) bool { return reserved[strings.ToUpper(base)] }

// FileName is the file one house's data lives in: the house's name, percent-escaped, plus
// ext (".scores", ".save").
//
// The original had no such problem. Its side-car was named with `thisHouseName` verbatim
// (HighScores.c:723), because HFS accepted every byte but ':' and a house name came from a
// file name in the first place. This port will meet house names from three places -- a file
// stem, a `.house` text file's own declaration, and Stage 2's new houses -- so the encoding
// has to be total.
//
// It is injective, which is the property that matters: two different houses cannot share a
// board or a save. `%` is itself escaped, so a `%XX` in the output can only have come from
// escaping, and the two special cases below cannot be produced any other way.
//
// One thing it is not is case-sensitive on a case-folding filesystem: "Titanic" and
// "titanic" are one file on macOS and Windows and two on Linux. That is the original's
// behaviour rather than a regression -- its folder scan compared names with
// `EqualString(..., true, true)`, case- and diacritical-insensitively (HighScores.c:696) --
// and treating two spellings of one house as one board is the better of the two answers
// anyway.
func FileName(houseName, ext string) string {
	var sb strings.Builder
	sb.Grow(len(houseName) + 8)
	for i := 0; i < len(houseName); i++ {
		c := houseName[i]
		if SafeByte(c) {
			sb.WriteByte(c)
			continue
		}
		const hexDigits = "0123456789ABCDEF"
		sb.WriteByte('%')
		sb.WriteByte(hexDigits[c>>4])
		sb.WriteByte(hexDigits[c&0x0F])
	}
	base := sb.String()

	switch {
	case base == "":
		// A house with no name at all. `%` alone is not something the escaper can emit,
		// so it cannot collide with any real name.
		base = "%"
	case Reserved(base):
		// Escape the first byte, which is always safe to do and always unambiguous: an
		// escape in that position could not have come from a safe byte.
		base = fmt.Sprintf("%%%02X%s", base[0], base[1:])
	case len(base) > maxBase:
		// Too long for some filesystem, somewhere. Shorten it and append a digest of the
		// *whole* name, so two houses that share a hundred-character prefix still get two
		// files. Twelve hex digits of SHA-256 is 48 bits.
		sum := sha256.Sum256([]byte(houseName))
		base = base[:maxBase] + "-" + hex.EncodeToString(sum[:6])
	}
	return base + ext
}
