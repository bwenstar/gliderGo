package shell

// The saved game on the menu: "Open Saved Game..." -- MENU 129's third item, with the ⌘O it
// carried in 1994 -- and the one line under it that says what resuming would get you.
//
// **The original's item did nothing.** It dispatches to OpenSavedGame (SavedGames.c:160),
// whose first live statement is `return false;		// TEMP fix this iwth NavServices`, so
// choosing it in the shipped 1.1.2 returned to the menu having read no file and started no
// game. internal/game/savegame.go's file comment has the rest of the inventory -- SaveGame2's
// commented-out body, SaveGame's commented-out call site, QueryResumeGame called from
// nowhere. That is why this row is a reconstruction, and why the two places it does more than
// the original are things the original's own arrangement made impossible rather than
// improvements for their own sake:
//
//   - **It says what it is offering before it is chosen.** The row is live only when there
//     is a save, and the status band shows the save's own summary -- three gliders, 8600
//     points, room 6 -- while the cursor is on it. QueryResumeGame (Menu.c:710-758) had
//     exactly that text, from `smallGame`, and could only show it *after* the file was open,
//     in a dialogue asking whether to throw the save away. A row that knows what it holds
//     needs no such dialogue.
//   - **There is no "beginning a new game will overwrite your saved game" warning**, because
//     nothing here overwrites a save except another save. The original needed the warning
//     because SaveGame(true) wrote the game into the *house file* (SavedGames.c:300-341) and
//     a new game would have clobbered it; this port never writes a house file it did not
//     author, so a new game leaves the save exactly where it was. Dying and resuming from
//     the same save is therefore allowed, which is what a save is for.
//   - **A house that ships with a game in it can be resumed.** Two of the twenty-two do --
//     `hasGame` and the 40 bytes at offset 820 -- and in 1994 nothing could open them,
//     because the menu item that would have (Menu.c:458) is commented out. The host offers
//     the house's own block when this installation has no save of its own, and the row says
//     which of the two it is holding. See house.EmbeddedGame.
//
// The row is greyed and says why rather than disappearing when there is nothing to resume.
// An item that appears only sometimes is an item nobody learns is there -- and the reason is
// the useful part: "no saved game for Slumberland" and "this build has nowhere to keep saved
// games" send a player to two different places.

import (
	"errors"
	"os"

	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/saved"
)

// errNoSaves is what a shell with no Saved hook answers: -shot, `-saves none`, the tests, a
// read-only installation. It is an error rather than a fourth return value so that the three
// unavailable cases below are one switch on one thing.
var errNoSaves = errors.New("shell: this build has no saved games")

// savedEntry is one cached answer from Host.Saved.
//
// The error is cached as well as the header, because "there is no save" is the answer for
// most houses most of the time and is worth exactly as much to a menu drawn sixty times a
// second as the other answer is. Both are dropped after every game; see Shell.saves.
type savedEntry struct {
	info saved.Info
	err  error
}

// saveOf is Host.Saved for one house, cached.
//
// Keyed on Rel and not on Name, for the reason board is (scores.go): Rel identifies the
// *file*, so two houses of the same name in two directories are two entries here. They do
// share one save, since the store is keyed by house name the way the original's would have
// been, and that is worth knowing rather than papering over in the cache.
func (s *Shell) saveOf(h House) (saved.Info, error) {
	if s.host.Saved == nil {
		return saved.Info{}, errNoSaves
	}
	if e, hit := s.saves[h.Rel]; hit {
		return e.info, e.err
	}
	info, err := s.host.Saved(h)
	if s.saves == nil {
		s.saves = make(map[string]savedEntry)
	}
	s.saves[h.Rel] = savedEntry{info: info, err: err}
	return info, err
}

// resumeItem is the menu row, which is four rows in one: it is live, or it is unavailable for
// one of three reasons, and each reason is something the player can act on.
func (s *Shell) resumeItem() item {
	it := item{key: platform.KeyO, label: "Open Saved Game...", do: s.resume}
	h, have := s.House()
	if !have {
		// No why: choose falls back to the shell's own, which says there are no houses and
		// how to get some. A house is the first thing this row needs.
		return it
	}

	info, err := s.saveOf(h)
	switch {
	case errors.Is(err, errNoSaves):
		it.why = "this build has nowhere to keep saved games"
	case errors.Is(err, os.ErrNotExist):
		it.why = "no saved game for " + h.Name + " -- press S while paused in a game to make one"
	case err != nil:
		// A save that exists and cannot be read. The store's errors name the file first and
		// the reason second, and both belong here: the reason says what is wrong and the path
		// says which file to move out of the way.
		it.why = "the saved game for " + h.Name + " cannot be read: " + err.Error()
	default:
		it.ok = true
		it.note = "resume " + h.Name + " -- " + info.Summary()
		if info.FromHouse {
			// Not this installation's save: the game the house file itself carries, which
			// is what `hasGame` means and what Titanic and ImagineHouse PRO II have held since 1994
			// with nothing able to open them (house.EmbeddedGame). Saying so is the
			// difference between a curiosity somebody chooses on purpose and a player
			// thinking their own save has been replaced by a stranger's.
			it.note = "resume the game " + h.Name + " shipped with -- " + info.Summary()
		}
	}
	return it
}

// resume asks the host for the house's saved game.
//
// One player, always: see Choice.Resume. The validation is not repeated here -- the file is
// read once, by whoever answers Play, and a shell that had checked it a second time would
// only be checking an older copy of the answer.
func (s *Shell) resume() {
	h, ok := s.House()
	if !ok {
		s.msg = s.why("Open Saved Game...")
		return
	}
	s.start(Choice{House: h, Resume: true})
}
