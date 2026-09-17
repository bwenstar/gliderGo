// Package saved is where a game in progress lives between sessions: one file per house,
// under the player's own data directory.
//
// The original put its saved games wherever a Standard File dialogue landed -- SaveGame2
// would have called `StandardPutFile` (SavedGames.c, in the commented body) and
// OpenSavedGame `StandardGetFile`, so the file could be anywhere and the player had to find
// it again. This port keys them by house instead, in a fixed place, for one reason: it makes
// "resume" a single keystroke instead of a file browser. That is the same trade
// internal/scores makes, and the two live side by side under internal/datadir's roof.
//
// One save per house, and the newer one wins. The original's dialogue would have allowed
// twenty saves of one house under twenty names; a keyed store allows one. That is a real
// loss and it is the right default -- Glider PRO is a game you resume, not one you branch --
// and it is recorded in docs/IMPROVEMENTS.md rather than pretended away.
//
// Nothing here decides what the fields *mean*; internal/house owns the format and
// internal/game owns the semantics. What this package owns is the four validation gates,
// which is the part the original wrote out in full and never ran (see Check).
package saved

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwenstar/gliderGo/internal/datadir"
	"github.com/bwenstar/gliderGo/internal/house"
)

// Ext is the file's extension. The original identified its saves by the type code 'gliG'
// with no extension at all; on a filesystem with no type codes an extension is how a
// directory listing stays legible, and the magic in the file's first four bytes is the same
// 'gliG' (see house.SavedGameMagic).
const Ext = ".save"

// SubDir is the directory the saves live in, under Dir.
const SubDir = "saves"

// Dir is where the saves go: internal/datadir's answer for SubDir. See that package for the
// reasoning and the two environment overrides.
func Dir() (string, error) {
	d, err := datadir.Dir(SubDir)
	if err != nil {
		return "", fmt.Errorf("saved: %w", err)
	}
	return d, nil
}

// Store is a directory of saved games.
//
// A nil *Store is a build with nowhere to save -- `-shot`, the tests, `-saves none`, a
// read-only installation -- and every method tolerates it: Load finds nothing, Save says
// why it cannot, Has is false. That is the same shape as scores.Store and Host.SavePrefs,
// and it is what lets the game's save hook be installed unconditionally and the menu row be
// greyed out from one test.
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

// OpenDir is Open for a directory the caller has chosen: `-saves <dir>`, and the tests.
func OpenDir(dir string) *Store { return &Store{dir: dir} }

// Dir is where this store keeps its files, or "" for a nil store.
func (st *Store) Dir() string {
	if st == nil {
		return ""
	}
	return st.dir
}

// Path is the file one house's save lives in, or "" for a nil store.
func (st *Store) Path(houseName string) string {
	if st == nil {
		return ""
	}
	return filepath.Join(st.dir, datadir.FileName(houseName, Ext))
}

// Has reports whether a house has a save, without reading or validating it.
//
// This is the menu's question, and it is deliberately the cheap one: the shell asks it for
// every house in the library every time it draws, so it must not read files. A save that
// exists and turns out to be unusable is reported when it is opened, which is where the
// player can be told what is wrong with it.
func (st *Store) Has(houseName string) bool {
	if st == nil {
		return false
	}
	fi, err := os.Stat(st.Path(houseName))
	return err == nil && fi.Mode().IsRegular()
}

// Save writes a game.
//
// The house name in the file and the name of the file are taken from the same place, so they
// cannot disagree -- a save whose header named one house and whose path named another would
// fail Check for a reason no player could act on.
//
// The write is atomic: the bytes go to a temporary file in the same directory and are
// renamed over the target. An interrupted save therefore leaves the *previous* save intact,
// which matters more here than it does for a score board. Saving is what a player does
// before quitting, so a half-written file would be discovered hours later with nothing to
// fall back on.
func (st *Store) Save(sg *house.SavedGame) error {
	if st == nil {
		return errors.New("saved: this build has nowhere to keep saved games")
	}
	data, err := sg.Encode()
	if err != nil {
		return err
	}

	path := st.Path(sg.HouseName.Text())
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".glidergo-save-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name) // no-op once the rename has succeeded
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	// Synced before the rename, not after. Without this the rename can be durable while
	// the contents are not, which on a crash gives a save that exists and is zero-length
	// -- strictly worse than the one it replaced.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// Load reads a house's save. A missing file is os.ErrNotExist, so a caller can tell "no save
// yet" from "the save is broken" with errors.Is.
//
// Unlike scores.Load this does not repair anything and does not fall back. A score board
// that is half wrong is still a score board; half a saved game is a game the player did not
// play, and putting them in the wrong room with the wrong inventory is worse than telling
// them the file is broken.
func (st *Store) Load(houseName string) (*house.SavedGame, error) {
	if st == nil {
		return nil, os.ErrNotExist
	}
	path := st.Path(houseName)

	// Bounded. The file's own header says how big it should be and the limit below is far
	// past any real house, so a file that grew to fill a disk is refused rather than read
	// into memory. The original read whatever GetEOF reported straight into a fixed
	// structure (see house.DecodeScores' note), which is the bug this avoids by arithmetic
	// rather than by luck.
	const maxRooms = 8192
	max := int64(house.SizeofSavedGameHeader + maxRooms*house.SizeofSavedRoom)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("%s: larger than %d bytes; this is not a saved game", path, max)
	}

	sg, err := house.DecodeSavedGame(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return sg, nil
}

// Remove deletes a house's save. A missing file is not an error, because the only caller
// this can have is a player asking for the file to go, and "it was already gone" is that
// request granted.
//
// **Nothing in the port calls it automatically, and in particular dying does not.** The
// obvious-looking rule -- consume the save when the game it holds is finished -- gets the
// player exactly backwards: dying is the moment somebody wants their save, and a resume that
// worked once should work again. So a save is replaced only by another save, and a house's
// save outlives every game played from it. That is also what makes it safe for the menu to
// offer a resume with no warning attached (internal/shell/saved.go).
func (st *Store) Remove(houseName string) error {
	if st == nil {
		return nil
	}
	err := os.Remove(st.Path(houseName))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// ---------------------------------------------------------------------------
// Peeking, for the menu
// ---------------------------------------------------------------------------

// Info is what a save says about itself: enough to put a line under a menu row so the player
// knows what they are about to resume before they resume it.
//
// The original offered nothing of the kind. QueryResumeGame (Menu.c:710-758) filled a
// dialogue with "^0 glider^1 ... ^2 points" from `smallGame` -- but only after the save had
// been opened, and only to ask whether to *discard* it. Showing it before the choice is a
// port addition; see docs/IMPROVEMENTS.md.
type Info struct {
	HouseName string // as the save records it, which is what Check compares
	Version   int16  // the saved-game version: 0x0200, or 0x0100 from a 1994 writer
	Score     int32
	Gliders   int16
	Room      int16 // room number, not a name -- naming it needs the house
	Stars     int16 // stars still to collect
	Bands     int16
	Battery   int16
	Rooms     int   // how many rooms the save carries, from the file's length
	Size      int64 // the file's size, for a diagnostic

	// FromHouse says this header came out of a house file's own 40 bytes -- `hasGame`,
	// which SaveGame(true) would have set and which two shipped houses hold -- rather than
	// out of a file in this store. Nothing here produces one; the host does, with
	// house.EmbeddedGame and InfoOf, because finding it means opening a house and this
	// package opens saves.
	//
	// It is one bool and it changes one line of text, and that line is worth it: "the game
	// Titanic shipped with" and "the game you saved last night" are different offers, and a
	// player who is shown the second when they are being handed the first will think their
	// save was lost.
	FromHouse bool
}

// InfoOf describes a saved game that is already in memory.
//
// Peek is this plus a file. It is separate so that the one caller with no file -- a house's
// own embedded block, which never was one -- describes itself through the same code, and so
// that Summary cannot end up with two definitions of what "3 gliders" means.
func InfoOf(sg *house.SavedGame) Info {
	if sg == nil {
		return Info{}
	}
	return Info{
		HouseName: sg.HouseName.Text(),
		Version:   sg.Game.Version,
		Score:     sg.Game.Score,
		Gliders:   sg.Game.NumGliders,
		Room:      sg.Game.RoomNumber,
		Stars:     sg.Game.WasStarsLeft,
		Bands:     sg.Game.Bands,
		Battery:   sg.Game.Energy,
		Rooms:     len(sg.Rooms),
	}
}

// Peek reads a save's header without reading its rooms.
//
// It is the menu's second question, after Has, and it reads 110 bytes and one stat rather
// than the whole file, because the shell asks it once per house and a large house's save is
// sixty kilobytes of object state that no menu row needs.
//
// The room count comes from the file's *size*, not from the header's nRooms, and then the
// two are compared -- so a truncated save is caught here too, before the player has chosen
// it. That ordering is the opposite of the original's, which trusted the field.
func (st *Store) Peek(houseName string) (Info, error) {
	var info Info
	if st == nil {
		return info, os.ErrNotExist
	}
	path := st.Path(houseName)

	f, err := os.Open(path)
	if err != nil {
		return info, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return info, err
	}
	head := make([]byte, house.SizeofSavedGameHeader)
	if _, err := io.ReadFull(f, head); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return info, fmt.Errorf("%s: %w: only %d bytes",
				path, house.ErrNotSavedGame, fi.Size())
		}
		return info, fmt.Errorf("%s: %w", path, err)
	}

	// DecodeSavedGame on the header alone plus a body of the right length. Rather than a
	// second, looser parser -- which could disagree with the real one -- the header is
	// padded out to the size the file claims and handed to the same function. The padding
	// decodes to rooms that are then thrown away.
	body := fi.Size() - int64(len(head))
	if body < 0 || body%house.SizeofSavedRoom != 0 ||
		body > int64(8192*house.SizeofSavedRoom) {
		return info, fmt.Errorf("%s: %d bytes is not a header plus whole rooms",
			path, fi.Size())
	}
	sg, err := house.DecodeSavedGame(append(head, make([]byte, body)...))
	if err != nil {
		return info, fmt.Errorf("%s: %w", path, err)
	}

	info = InfoOf(sg)
	info.Size = fi.Size()
	return info, nil
}

// Summary is Info as one line for a menu row: what QueryResumeGame's dialogue would have
// said, in the space a menu has.
//
// The original's wording is kept where it fits -- "glider"/"gliders" pluralised from the
// count, and "points" -- because it is the only phrasing the game ever used for this and
// there is no reason to invent another.
func (i Info) Summary() string {
	glider := "gliders"
	if i.Gliders == 1 {
		glider = "glider"
	}
	return fmt.Sprintf("%d %s, %d points, room %d", i.Gliders, glider, i.Score, i.Room)
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// Mismatch is a save that is well-formed and does not belong to the house it was offered
// with. It is separated from an ordinary error because the four cases below are the
// original's four YellowAlerts, each of which had its own wording, and because a caller may
// reasonably want to offer "start a new game instead" for these and not for a parse failure.
type Mismatch struct {
	Reason string
}

func (m *Mismatch) Error() string { return m.Reason }

// Check is OpenSavedGame's validation (SavedGames.c:180-200, in the commented body), in the
// original's order, with its reasons and one addition.
//
// The order is worth keeping because it decides which complaint a player sees when more than
// one thing is wrong, and the original's order is from most specific to least: the wrong
// house first, then the right house edited, then the wrong version, then the wrong size.
//
//	gate 1  house name      SavedGameMismatchError -> alert 1044, which named the house
//	gate 2  house timeStamp  kYellowSavedTimeWrong
//	gate 3  saved version    kYellowSavedVersWrong
//	gate 4  nRooms           kYellowSavedRoomsWrong, whose parameter was the *difference*
//
// Two departures, both argued in internal/house's savedgame.go:
//
//   - Gate 3 accepts 0x0100 as well as kSavedGameVersion's 0x0200 (S3). Every saved game in
//     the shipped corpus holds 0x0100, so the original's own gate would reject the only real
//     saves in existence -- including Titanic's, which this stage is specified to resume.
//   - A fifth gate: the saved room number must name a room of the house. The original had no
//     such check here and instead raised kYellowIllegalRoomNum from ForceThisRoom, deep
//     inside game start-up with the world already half torn down. Checking it before
//     anything is applied means a bad save cannot half-start a game.
//
// Gate 4's own backstop lives in house.ApplySavedRooms, so a caller that skips Check cannot
// scatter one house's objects through another.
func Check(sg *house.SavedGame, houseName string, h *house.House) error {
	// EqualString(..., true, true) -- case- *and* diacritical-insensitive. Go's EqualFold
	// is the first and not the second, which is the right amount of leniency here for a
	// reason the original did not have: the file is keyed by the escaped house name, and on
	// a case-folding filesystem two spellings of one house are already one file
	// (see datadir.FileName).
	if !strings.EqualFold(sg.HouseName.Text(), houseName) {
		return &Mismatch{Reason: fmt.Sprintf("this saved game is for the house %q, not %q",
			sg.HouseName.Text(), houseName)}
	}

	if h == nil {
		// Nothing further can be checked without the house. Not an error: a caller
		// listing saves for a menu has the name and not the file.
		return nil
	}

	if h.TimeStamp != sg.Game.TimeStamp {
		// kYellowSavedTimeWrong. The house has been edited -- or is a different copy of
		// it -- since the game was saved, and the object states in the save no longer
		// describe the rooms they would be applied to.
		return &Mismatch{Reason: fmt.Sprintf(
			"%s has been modified since this game was saved", houseName)}
	}

	switch sg.Game.Version {
	case house.SavedGameVersion, house.SavedGameVersion1:
	default:
		return &Mismatch{Reason: fmt.Sprintf(
			"this saved game is version %#04x; gliderGo reads %#04x and %#04x",
			sg.Game.Version, house.SavedGameVersion1, house.SavedGameVersion)}
	}

	// A nil room slice is a save with no snapshot at all: a house's own embedded 40 bytes,
	// where the objects a resume starts from are the ones the house file holds
	// (house.EmbeddedGame). There is nothing to compare, and nothing to corrupt either. The
	// marker cannot be forged by a file, because DecodeSavedGame always allocates the slice
	// -- a file claiming zero rooms decodes to an empty one and is caught here.
	if sg.Rooms != nil && len(sg.Rooms) != len(h.Rooms) {
		// kYellowSavedRoomsWrong, whose alert parameter was the signed difference. Kept,
		// because "three rooms too few" says more about what happened to the house than
		// two absolute numbers do -- and both are here anyway.
		return &Mismatch{Reason: fmt.Sprintf(
			"this saved game has %d rooms and %s has %d",
			len(sg.Rooms), houseName, len(h.Rooms))}
	}

	// The fifth gate. Note that the empty-house case is covered: len(h.Rooms) == 0 makes
	// every room number out of range.
	if n := sg.Game.RoomNumber; n < 0 || int(n) >= len(h.Rooms) {
		return &Mismatch{Reason: fmt.Sprintf(
			"this saved game is in room %d, and %s has %d rooms",
			n, houseName, len(h.Rooms))}
	}
	return nil
}
