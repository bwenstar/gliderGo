// Package scores is the high-score board: the ten rows, the file they persist in, the
// screen that shows them and the field a champion types their name into.
//
// # Why this is a package and not part of internal/house
//
// house owns the 292 bytes (`scoresType`, docs/analysis/scoring.md 7.2) and nothing else.
// It has to: it is the house codec, it is imported by the tools, and it must stay a
// codec -- no policy, no files of its own, no drawing. What the original spread across
// HighScores.c, HouseIO.c and Menu.c is *policy over* those bytes, and it lands here so
// that both callers can reach it. There are two, and they cannot see each other: the
// shell shows a board from the title screen (the original's Options > High Scores,
// Menu.c:418) and the game offers one at the end of a game (GameOver.c:67 and :503).
// internal/shell does not import internal/game and must not start.
//
// # The one deliberate structural change
//
// In 1994 a new high score was written back into the house file. `TestHighScore` set
// `gameDirty`, and `CloseHouse` answered that by calling `WriteHouse`, which rewrites the
// entire file -- every room, every object, the saved-game blob, all of it -- to record ten
// rows (docs/analysis/scoring.md 7.13). The port does not do that, for two reasons that
// have nothing to do with taste:
//
//   - The 22 shipped houses are not ours. They are read-only inputs to this port and the
//     licence position on them is unsettled (docs/IMPROVEMENTS.md 1.2); a game that
//     silently rewrites the files it was given is a game that cannot be shipped with a
//     clean conscience even if it never has a bug.
//   - A 98 KB whole-file rewrite to persist 292 bytes is one power cut away from a
//     destroyed house. The original has no temporary file and no rename.
//
// So the board a player fills in goes in a side-car file of exactly the same 292 bytes,
// under the player's own data directory, one per house -- which is the store the original
// *specified and then never used*, because its `houseIsReadOnly` was hard-wired false
// (docs/analysis/scoring.md 7.1 and 7.14). See store.go. The house's own board is still
// read, and is what a fresh side-car is seeded from, so a house that has had Ozma's 7400
// in it since 1995 shows Ozma's 7400 on the first run.
package scores

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwenstar/gliderGo/internal/house"
)

// Max is kMaxScores: the number of rows on a board. Ten, everywhere, and not a
// preference -- the array in the file is ten long.
const Max = house.MaxScores

// EmptyName is the placeholder `ZeroHighScores` writes into an unused row
// (HighScores.c:337, `"\p--------------"`). Fourteen hyphens, counted from the source
// byte by byte; the renderer never sees it, because a row is only drawn when its score
// is positive, but a file this port writes has to hold what 1994 would have held there
// or a board it seeds and a board it saves differ for no reason.
const EmptyName = "--------------"

// Zero is ZeroHighScores (HighScores.c:323-346): every row emptied and the banner reset
// to the house's name.
//
// The banner is the surprising half and it is what the C does: `PasStringCopy(thisHouseName,
// highScores.banner)`. A cleared board therefore advertises the house rather than the last
// champion's message, which is why five shipped houses have a banner that is just their own
// name (Empty House, Land of Illusion -- docs/analysis/scoring.md 7.3).
func Zero(s *house.Scores, houseName string) {
	s.Banner.SetText(houseName)
	zeroFrom(s, 0)
}

// ZeroAllButHighest is ZeroAllButHighestScore (HighScores.c:348-370): rows 1..9 emptied,
// row 0 and the banner left alone.
//
// The original reaches it from one place, the house-info dialog's "zero all but the
// highest" button (HouseInfo.c:335), and it assumes the board is sorted -- it keeps
// *row zero*, not the highest score. So does this. Sort first if that is in doubt.
func ZeroAllButHighest(s *house.Scores) { zeroFrom(s, 1) }

func zeroFrom(s *house.Scores, first int) {
	for i := first; i < Max; i++ {
		s.Names[i].SetText(EmptyName)
		s.Scores[i] = 0
		s.TimeStamps[i] = 0
		s.Levels[i] = 0
	}
}

// SetBanner writes the board's banner, the line a first-place finish earns the right to
// change (HighScores.c:406-408).
//
// Text longer than 31 Mac Roman bytes is truncated, which is `PasStringCopyNum(tempStr,
// highBanner, 31)` at HighScores.c:633. A rune with no Mac Roman equivalent becomes '?',
// which is house.UTF8ToMacRoman's answer and is why the entry field refuses those runes
// up front rather than letting them arrive here (see Field.Insert).
func SetBanner(s *house.Scores, banner string) { s.Banner.SetText(banner) }

// Occupied counts the rows a board actually holds, which is the number the renderer draws:
// `if (scores[i] > 0L)` at HighScores.c:176. A score of zero is an empty row and a
// negative one is impossible in play -- the awards are all positive multiples of 100
// (docs/analysis/scoring.md 7.3) -- so this is also the test for "has anybody ever
// finished this house".
func Occupied(s *house.Scores) int {
	n := 0
	for i := 0; i < Max; i++ {
		if s.Scores[i] > 0 {
			n++
		}
	}
	return n
}

// Qualify is TestHighScore's search (HighScores.c:389-397): the row a score would take,
// or -1 for a score that does not make the board.
//
// The comparison is strictly greater, so a score that *ties* the tenth place does not
// get on -- and, higher up the board, a tie sits below the row it tied. That is why Fun
// House's board reads 1300, 1300 with the older of the two above (docs/analysis/scoring.md
// 7.3): the second 1300 could not beat the first, so it went in below it.
//
// It assumes descending order, as the original does. Repair is how a board that has been
// hand-edited into some other order gets back to one this can be trusted on.
func Qualify(s *house.Scores, score int32) int {
	for i := 0; i < Max; i++ {
		if score > s.Scores[i] {
			return i
		}
	}
	return -1
}

// Insert puts one finished game on the board and returns the row it ended up in, or -1 if
// it did not qualify.
//
// This is the second half of TestHighScore (HighScores.c:399-415) and it is deliberately
// the same odd shape: the new score is written into the *last* row and the board is then
// sorted, rather than the rows below the placing being shifted down. Same result, and
// keeping the shape means keeping the property that falls out of it -- the row that gets
// pushed off the bottom is the one the new score overwrites, so nothing has to decide
// what to discard.
//
// The returned row is `placing` from Qualify, which is where the sort puts the new score.
// That is provable rather than hopeful: every row above `placing` holds a score >= this
// one and comes first, and the sort takes the first of equals, so all of them are picked
// before row 9 is; every row from `placing` to 8 holds strictly less. `lastHighScore`, the
// white highlight on the board, is that index (see Draw's highlight argument).
func Insert(s *house.Scores, name string, score int32, rooms int16, when time.Time) int {
	placing := Qualify(s, score)
	if placing < 0 {
		return -1
	}
	s.Names[Max-1].SetText(name)
	s.Scores[Max-1] = score
	s.TimeStamps[Max-1] = uint32(house.MacTimeFrom(when))
	s.Levels[Max-1] = rooms
	Sort(s)
	return placing
}

// Sort is SortHighScores (HighScores.c:281-318): a selection sort, descending, ties
// resolved in favour of the lower index.
//
// Ten passes, each taking the greatest remaining score and marking it consumed with -1.
// The tie rule is not a choice either -- `scores[i] > greatest` only replaces the
// candidate on a strict improvement, so the first of two equal scores is taken first --
// and it is visible in the shipped boards, which is the evidence that this is the sort
// that produced them.
//
// Two deliberate differences from the C, both in the same direction:
//
//   - **The scratch board is zero, not stack garbage.** The C declares `scoresType
//     tempScores` on the stack, copies each winning row into it with `PasStringCopy` --
//     which writes length+1 bytes and no more -- and then assigns the whole struct back.
//     So every sort replaced the padding after each name with whatever was on the stack
//     at that moment, and that garbage was written to the house file: it is the source
//     of the residue in all 22 shipped boards (docs/analysis/scoring.md 7.2). Here the
//     rows are copied whole, so a name's residue travels with the name it belongs to and
//     a sort of an unchanged board changes no bytes at all.
//   - **A row whose score is already -1 is kept rather than lost.** In the C such a row
//     is never the greatest, so it is never picked, and its slot in tempScores stays
//     uninitialised -- the sort silently replaces it with garbage. Only a corrupt file
//     can produce one (Repair clamps them first), but "silently replaces with garbage" is
//     not a behaviour worth transcribing, so unpicked rows are left empty instead.
func Sort(s *house.Scores) {
	var temp house.Scores
	work := *s // the copy the -1 markers are scribbled on, so s is not damaged mid-sort

	for h := 0; h < Max; h++ {
		greatest, which := int32(-1), -1
		for i := 0; i < Max; i++ {
			if work.Scores[i] > greatest {
				greatest, which = work.Scores[i], i
			}
		}
		if which < 0 {
			continue
		}
		temp.Names[h] = work.Names[which]
		temp.Scores[h] = work.Scores[which]
		temp.TimeStamps[h] = work.TimeStamps[which]
		temp.Levels[h] = work.Levels[which]
		work.Scores[which] = -1
	}

	// The banner is copied separately in the C too (HighScores.c:314), because it is the
	// board's and not a row's -- sorting the rows must not move it.
	temp.Banner = s.Banner
	*s = temp
}

// Sorted reports whether a board is in the descending order everything else here assumes.
func Sorted(s *house.Scores) bool {
	for i := 1; i < Max; i++ {
		if s.Scores[i] > s.Scores[i-1] {
			return false
		}
	}
	return true
}

// Repair makes a board safe to use and returns one line per thing it had to change.
//
// This exists because the side-car is a file in the player's own data directory. It can be
// truncated by a full disk, copied between machines, restored from a backup half-written,
// or edited by hand by somebody who wants to see their name at the top -- and the answer to
// all of those has to be a playable game, not a crash and not a silently wrong board.
// The original's answer was to read the file with one unclamped FSRead straight over the
// middle of the house structure in memory (docs/analysis/scoring.md 7.14), which for a file
// of the wrong size corrupted the room array and then the heap.
//
// What it repairs, and why each one matters:
//
//   - **A negative score becomes zero.** -1 is the sort's own "consumed" marker, so a
//     stored -1 is a row the sort cannot see; and a negative score sorts below the empty
//     rows, which puts a visible row underneath invisible ones.
//   - **A negative room count becomes zero.** It is drawn, so it has to be drawable, and
//     "-3 rooms" is not a thing that happened.
//   - **A row with a score and no name gets the placeholder**, so the board does not have
//     a blank line in the middle of it.
//   - **A board out of order is sorted.** Qualify and Insert are both defined in terms of
//     descending order; a board that is not in it puts scores in the wrong rows rather
//     than merely looking odd.
//
// It does not repair the residue, the timestamps or the banner: none of them can make the
// game do anything wrong, and rewriting bytes that are merely strange is how a port loses
// the ability to say a file round-trips.
func Repair(s *house.Scores) []string {
	var notes []string
	for i := 0; i < Max; i++ {
		if s.Scores[i] < 0 {
			notes = append(notes, fmt.Sprintf("row %d had a negative score (%d); cleared it",
				i+1, s.Scores[i]))
			s.Scores[i] = 0
			s.TimeStamps[i] = 0
			s.Levels[i] = 0
			s.Names[i].SetText(EmptyName)
			continue
		}
		if s.Levels[i] < 0 {
			notes = append(notes, fmt.Sprintf("row %d claimed %d rooms; called it 0",
				i+1, s.Levels[i]))
			s.Levels[i] = 0
		}
		if s.Scores[i] > 0 && strings.TrimSpace(s.Names[i].Text()) == "" {
			notes = append(notes, fmt.Sprintf("row %d had a score and no name", i+1))
			s.Names[i].SetText(EmptyName)
		}
	}
	if !Sorted(s) {
		notes = append(notes, "the rows were out of order; sorted them")
		Sort(s)
	}
	return notes
}

// Name is a row's name, with a legible stand-in for the two ways it can be unreadable: the
// dash placeholder an empty row carries, and a name that is blank or all spaces.
//
// The placeholder is only ever seen on a row with no score, which the board does not draw,
// so this is for the callers that show a single row out of context -- the picker's footer
// and the status line.
func Name(s *house.Scores, i int) string {
	if i < 0 || i >= Max {
		return ""
	}
	who := s.Names[i].Text()
	if who == EmptyName || strings.TrimSpace(who) == "" {
		return "(nameless)"
	}
	return who
}
