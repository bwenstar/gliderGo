package shell

// The high-score screen, reached from the menu: Options > High Scores (Menu.c:417-419).
//
// The screen itself is internal/scores' -- Draw there is DrawHighScores, transcribed
// coordinate by coordinate from docs/analysis/scoring.md 7.9. What this file does is the
// part that is the shell's business: which house's board is up, where the board came from,
// and how the player gets out.
//
// **The original could not do this.** Its menu item calls DoHighScores, which draws the
// whole screen into an offscreen buffer and then returns without ever copying it to the
// window (7.8) -- so choosing High Scores in 1994 waited a second and a half and showed you
// nothing. The layout in the analysis was read out of the source rather than off a
// screenshot, because no screenshot of it can exist. This is therefore the one screen in
// the port that is *more* than a port, and the only honest way to build it was to transcribe
// the drawing exactly and fix nothing but the missing blit.
//
// One deliberate difference from the drawing code's own contract, and it is about the white
// highlight. `lastHighScore` is a module global set by TestHighScore and reset only at
// launch (7.5 note 3), so in the original a board opened from this menu still highlights the
// row from the last qualifying game -- in a different house, if a different house is loaded,
// because the global is a bare index with no house attached to it. That is an uninitialised
// state read rather than a behaviour, and reproducing it here would mean carrying a
// cross-house index around for the sake of highlighting somebody else's score. The menu's
// screen highlights nothing; the post-game screen, which cmd/glidergo draws, highlights the
// row the score actually landed in. Noted in docs/IMPROVEMENTS.md.

import (
	"fmt"

	"glidergo/internal/house"
	"glidergo/internal/scores"
)

// openScores puts the board up for the selected house.
//
// It refuses when there is no house, for the same reason the menu item is unavailable then:
// a high-score board belongs to a house and there is no such thing as the game's own board
// (7.1 -- the three candidate stores are all per-house).
func (s *Shell) openScores() {
	h, ok := s.House()
	if !ok {
		s.msg = s.why("High Scores...")
		return
	}
	s.mode = modeScores
	s.msg = s.boardLine(h)
}

// board is the merged board for one house, cached.
//
// With no Scores hook it is the house file's own table, which house.PeekFile already read
// on the way past -- so a build with no side-car still shows the twenty-two shipped boards
// rather than ten empty rows. That is not a degraded mode: those boards are 1994's, they
// are the reason the screen is worth having before anybody has played, and §7.3 lists every
// one of them.
func (s *Shell) board(h House) house.Scores {
	if s.host.Scores == nil {
		return h.Scores
	}
	// Keyed on Rel and not on Name, because Rel is what identifies the *file*: two houses
	// with the same name in two directories are two houses here (see library.go). They
	// will share a side-car, since the side-car is named after the house the way the
	// original's was, and that is a consequence worth knowing about rather than a bug to
	// paper over in the cache.
	if b, hit := s.boards[h.Rel]; hit {
		return b
	}
	b := s.host.Scores(h)
	if s.boards == nil {
		s.boards = make(map[string]house.Scores)
	}
	s.boards[h.Rel] = b
	return b
}

// boardLine is the status line under the board: how many rows are filled, and the banner
// the champion left, which is the one part of a board that is a message rather than a
// number.
func (s *Shell) boardLine(h House) string {
	b := s.board(h)
	n := scores.Occupied(&b)
	switch {
	case n == 0:
		return h.Name + " -- no scores yet"
	case b.Banner.Text() != "":
		return fmt.Sprintf("%s -- %d of %d, banner: %s",
			h.Name, n, scores.Max, b.Banner.Text())
	default:
		return fmt.Sprintf("%s -- %d of %d scores", h.Name, n, scores.Max)
	}
}

// drawScores paints the board over the whole screen.
//
// It replaces the splash backdrop rather than sitting on it as a panel, which is what the
// original does: DrawHighScores fills the work map with PICT 1995's starfield and puts the
// plaque, the title and the ten rows straight onto it (7.9.1). There is nothing to put a
// panel over.
//
// The status band still goes at the bottom, in the twenty rows the original's window did
// not have (see screens.go). The board's own footer -- "Hit a Key to Exit" -- lands at row
// 308 on a 640x480 screen, so the two do not touch.
//
// It answers whether it drew. False means the mode was wrong for it and has been corrected,
// so the caller draws the splash instead of presenting a blank frame.
func (s *Shell) drawScores() bool {
	h, ok := s.House()
	if !ok {
		// Unreachable through openScores, and reachable by a unit test or a future caller
		// that sets the mode itself. Recover rather than draw nothing.
		s.mode = modeSplash
		s.msg = s.why("High Scores...")
		return false
	}
	b := s.board(h)
	scores.Draw(s.host.Screen, s.host.Assets, h.Name, &b, -1)
	return true
}
