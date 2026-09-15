package scores

// The side-car: one 292-byte file per house, under the player's own data directory.
//
// This is the store docs/analysis/scoring.md 7.14 specifies and 7.1 shows was never
// reached, because its only guard `houseIsReadOnly` came from a function whose whole body
// is `return false` (HouseIO.c:659-664). The format is the original's exactly -- the raw
// `scoresType`, big-endian, at offset 0, exactly `sizeof(scoresType)` bytes -- so a board
// this port writes could be read by a 1994 build that had `houseIsReadOnly` fixed, and
// one written by that build can be read here.
//
// Three things about it are not the original's, and each is a bug of the original's that
// the analysis names:
//
//   - **The read is clamped.** `ReadScoresFromDisk` does `GetEOF` then one `FSRead` of
//     that many bytes into `&highScores`, which is 292 bytes in the *middle* of the house
//     structure. A 400-byte side-car therefore overwrote the saved game, `hasGame`,
//     `firstRoom`, `nRooms` and the start of the room array, and a large enough one ran
//     off the end of the Memory Manager block (7.14). Here at most 292 bytes are taken.
//   - **A short file is an overlay, not a truncation.** The original's read left whatever
//     the house already held past the end of the file, because it read *into* the house.
//     That behaviour is worth keeping -- a file truncated by a full disk still yields a
//     usable board -- so the same thing is done deliberately: the house's own board is the
//     background and the file is painted over it.
//   - **The write is atomic.** `WriteScoresToDisk` writes in place and then calls SetEOF.
//     Here the bytes go to a temporary file in the same directory and are renamed over the
//     target, so an interrupted save leaves the previous board rather than half of one.
//
// Where it goes is a modern question the original could not have (docs/IMPROVEMENTS.md
// 3.1): 1994 had one place for everything, "the Preferences folder", and a save game and a
// score board went in the same drawer. See Dir.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"glidergo/internal/house"
)

// Ext is the side-car's extension. The original's files had none -- they were named after
// the house and identified by the type code 'gliS' (7.14) -- and on a filesystem with no
// type codes an extension is how a directory listing stays legible.
const Ext = ".scores"

// SubDir is the directory the boards live in, under Dir. It is the original's
// "G-PRO Scores ƒ" folder (HighScores.c:654, the trailing byte being Mac Roman 0xC4),
// renamed to something that can be typed.
const SubDir = "scores"

// Dir is where the boards go.
//
// Scores are *data*, not configuration: the player did not choose them, cannot usefully
// edit them, and would not expect them in the same place as a settings file. On Linux the
// XDG basedir spec says that means $XDG_DATA_HOME (~/.local/share), not $XDG_CONFIG_HOME,
// and the distinction is not pedantry -- a dotfile-syncing setup that tracks ~/.config
// should not be picking up a game's high scores. Go has os.UserConfigDir and os.UserCacheDir
// and no os.UserDataDir, so the Linux path is resolved by hand and every other platform
// falls back to the config directory, which is where its own conventions put both
// (~/Library/Application Support, %AppData%).
//
// Two overrides, in order:
//
//	GLIDERGO_DATA     used verbatim, for a portable install and for tests
//	GLIDERGO_CONFIG   if set, scores go under it, so that a player who has pointed the
//	                  whole game at one directory gets one directory
//
// The second is prefs.Dir's variable and is honoured here on purpose: somebody who runs
// `GLIDERGO_CONFIG=./glider-data glidergo` has said where they want the game's files, and
// answering that with settings in one place and scores in another would be a bug.
func Dir() (string, error) {
	if d := os.Getenv("GLIDERGO_DATA"); d != "" {
		return d, nil
	}
	if d := os.Getenv("GLIDERGO_CONFIG"); d != "" {
		return filepath.Join(d, SubDir), nil
	}
	if runtime.GOOS == "linux" {
		if d := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(d) {
			return filepath.Join(d, "glidergo", SubDir), nil
		}
		// $XDG_DATA_HOME unset, or set to a relative path, which the spec says to treat as
		// unset. ~/.local/share is the spec's own default.
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".local", "share", "glidergo", SubDir), nil
		}
		// No home directory either. Fall through rather than fail: os.UserConfigDir has
		// its own answer and its own error message.
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("scores: no data directory: %w", err)
	}
	return filepath.Join(base, "glidergo", SubDir), nil
}

// Store is a directory of boards.
//
// A nil *Store is a build with nowhere to keep scores -- `-shot`, the tests, a read-only
// installation -- and every method tolerates it: Load returns the house's own board and
// Save says why it cannot. That is the same shape as Host.SavePrefs being nil (see
// internal/shell): a session that can play and cannot record is a real thing and it should
// not be a special case at every call site.
type Store struct {
	dir string
}

// Open resolves Dir and returns a store for it. The directory is not created until a save
// needs it, so merely launching the game writes nothing.
func Open() (*Store, error) {
	d, err := Dir()
	if err != nil {
		return nil, err
	}
	return &Store{dir: d}, nil
}

// OpenDir is Open for a directory the caller has chosen: `-scores <dir>`, and the tests.
func OpenDir(dir string) *Store { return &Store{dir: dir} }

// Dir is where this store keeps its files, or "" for a nil store.
func (st *Store) Dir() string {
	if st == nil {
		return ""
	}
	return st.dir
}

// Path is the file one house's board lives in, or "" for a nil store.
func (st *Store) Path(houseName string) string {
	if st == nil {
		return ""
	}
	return filepath.Join(st.dir, FileName(houseName))
}

// Load reads a house's board, seeded from the board the house file itself carries.
//
// The seed is the point. A house that has had a score in it since 1995 shows that score
// the first time it is opened here, because the side-car starts as a copy of what the file
// holds and the player's own games accumulate on top of it. It also means the answer to a
// deleted side-car is the 1994 board rather than an empty one, which is the behaviour
// somebody would expect of a file they deleted on purpose.
//
// Nothing here fails. A missing file is an ordinary first run; anything else that goes
// wrong -- unreadable, truncated, the wrong length, hand-edited into nonsense -- yields a
// board that can be played and shown, plus one note per thing that had to be worked
// around. The notes are for the status line and stderr, not for the player's attention.
func (st *Store) Load(houseName string, seed house.Scores) (house.Scores, []string) {
	if st == nil {
		return seed, nil
	}
	path := st.Path(houseName)
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return seed, nil
	case err != nil:
		return seed, []string{fmt.Sprintf("cannot read %s (%v); using the board in the house", path, err)}
	}

	board, notes := Overlay(seed, data)
	if len(notes) > 0 {
		for i, n := range notes {
			notes[i] = filepath.Base(path) + ": " + n
		}
	}
	return board, notes
}

// Overlay is the read path of docs/analysis/scoring.md 7.14 with the clamp it lacks: the
// file's bytes are painted over the seed board and the result is decoded and repaired.
//
// Exported because it is the part worth testing on its own and the part a caller might want
// without a file -- a board arriving over the Stage 3 network connection is the same
// problem.
func Overlay(seed house.Scores, data []byte) (house.Scores, []string) {
	var notes []string

	b := house.EncodeScores(&seed)
	n := copy(b, data)
	switch {
	case n < len(b):
		notes = append(notes, fmt.Sprintf("only %d of %d bytes; the rest is the board in the house",
			n, len(b)))
	case len(data) > len(b):
		notes = append(notes, fmt.Sprintf("%d bytes; read the first %d and ignored the rest",
			len(data), len(b)))
	}

	board, err := house.DecodeScores(b)
	if err != nil {
		// Unreachable: b is EncodeScores' own buffer, so it is exactly the right length and
		// DecodeScores' only error is the length. Reported rather than panicked because
		// this is a file path and a panic in one is never the right answer.
		return seed, append(notes, err.Error())
	}
	return board, append(notes, Repair(&board)...)
}

// Save writes a house's board.
//
// It creates the directory on the way, which is `CreateScoresFolder` (HighScores.c:642-661)
// doing the same thing with FSpDirCreate, and it is the only moment this port writes
// anything anywhere on its own initiative.
func (st *Store) Save(houseName string, s *house.Scores) error {
	if st == nil {
		return errors.New("scores: this build has nowhere to keep high scores")
	}
	path := st.Path(houseName)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data := house.EncodeScores(s)
	tmp, err := os.CreateTemp(dir, "board.tmp*")
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

// ---------------------------------------------------------------------------
// House name -> file name
// ---------------------------------------------------------------------------

// safeByte is the set a file name may hold unescaped. Deliberately narrow: it is the
// intersection of what Linux, macOS and Windows all accept in any position, so the same
// house produces the same file name on all three and a scores directory can be copied
// between them.
func safeByte(c byte) bool {
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

// FileName is the file a house's board lives in: the house's name, percent-escaped.
//
// The original had no such problem. Its side-car was named with `thisHouseName` verbatim
// (HighScores.c:723), because HFS accepted every byte but ':' and a house name came from a
// file name in the first place. This port will meet house names from three places -- a file
// stem, a `.house` text file's own declaration, and Stage 2's new houses -- so the encoding
// has to be total.
//
// It is injective, which is the property that matters: two different houses cannot share a
// board. `%` is itself escaped, so a `%XX` in the output can only have come from escaping,
// and the two special cases below cannot be produced any other way.
//
// One thing it is not is case-sensitive on a case-folding filesystem: "Titanic" and
// "titanic" are one file on macOS and Windows and two on Linux. That is the original's
// behaviour rather than a regression -- its folder scan compared names with
// `EqualString(..., true, true)`, case- and diacritical-insensitively (HighScores.c:696) --
// and treating two spellings of one house as one board is the better of the two answers
// anyway.
func FileName(houseName string) string {
	var sb strings.Builder
	sb.Grow(len(houseName) + 8)
	for i := 0; i < len(houseName); i++ {
		c := houseName[i]
		if safeByte(c) {
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
	case reserved[strings.ToUpper(base)]:
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
	return base + Ext
}
