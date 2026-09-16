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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"glidergo/internal/datadir"
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

// Dir is where the boards go: internal/datadir's answer for SubDir. That package holds the
// reasoning (data rather than config, the two environment overrides, the per-platform
// bases); it is shared with internal/saved so that a house's board and its save land in the
// same tree under the same rules.
func Dir() (string, error) {
	d, err := datadir.Dir(SubDir)
	if err != nil {
		return "", fmt.Errorf("scores: %w", err)
	}
	return d, nil
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

// FileName is the file a house's board lives in: the house's name, percent-escaped, with
// Ext. The escaper is internal/datadir's, shared with internal/saved so that one house's
// two files are named alike.
func FileName(houseName string) string { return datadir.FileName(houseName, Ext) }
