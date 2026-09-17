package scores

// The board as a data structure: the sort, the placing rule, and what happens to a board
// that has been tampered with.
//
// One of these tests is evidence about 1994 rather than about this code:
// TestSortOfAShippedBoardChangesNothing runs the transcribed sort over all 22 boards that
// came in the shipped houses and requires that it move nothing. Those boards are the output
// of the original sort, so a transcription that reordered any of them would be wrong -- and
// because the comparison is on bytes, it also catches a sort that got the right order by
// losing the residue.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bwenstar/gliderGo/internal/house"
)

// ------------------------------------------------------------------- fixtures

// board builds a descending board from a list of scores, with a distinguishable name,
// timestamp, room count and residue per row. The residue matters: several of these tests
// are about which bytes travel with which row.
func board(scores ...int32) house.Scores {
	var s house.Scores
	s.Banner.SetText("A Banner")
	for i := 0; i < Max; i++ {
		if i < len(scores) && scores[i] > 0 {
			s.Names[i].SetText(rowName(i))
			s.Scores[i] = scores[i]
			s.TimeStamps[i] = uint32(0x50000000 + i)
			s.Levels[i] = int16(10 + i)
		} else {
			s.Names[i].SetText(EmptyName)
		}
		// Residue past the length byte, unique per row, so a row that ends up somewhere
		// else can be identified by its garbage alone.
		for j := 1 + int(s.Names[i][0]); j < len(s.Names[i]); j++ {
			s.Names[i][j] = byte(0x80 + i)
		}
	}
	return s
}

func rowName(i int) string { return "Player " + string(rune('A'+i)) }

// names is a board's ten names, for comparing orders without printing 292 bytes.
func names(s *house.Scores) []string {
	out := make([]string, Max)
	for i := range out {
		out[i] = s.Names[i].Text()
	}
	return out
}

func scoreList(s *house.Scores) []int32 {
	out := make([]int32, Max)
	copy(out, s.Scores[:])
	return out
}

// ------------------------------------------------------------------- clearing

func TestZeroEmptiesEveryRowAndNamesTheHouse(t *testing.T) {
	s := board(900, 800, 700)
	Zero(&s, "Ice Palace")

	if got := s.Banner.Text(); got != "Ice Palace" {
		t.Errorf("banner is %q, want the house name", got)
	}
	for i := 0; i < Max; i++ {
		if s.Scores[i] != 0 || s.TimeStamps[i] != 0 || s.Levels[i] != 0 {
			t.Errorf("row %d survived Zero: score %d, stamp %d, rooms %d",
				i, s.Scores[i], s.TimeStamps[i], s.Levels[i])
		}
		if got := s.Names[i].Text(); got != EmptyName {
			t.Errorf("row %d is named %q, want the placeholder", i, got)
		}
	}
	if Occupied(&s) != 0 {
		t.Errorf("a zeroed board reports %d occupied rows", Occupied(&s))
	}
}

// The placeholder is fourteen hyphens, which is what HighScores.c:337 writes. Checked as a
// number because it is the sort of constant that loses a character in transcription and
// nothing else in the port would notice: the renderer never draws it.
func TestEmptyNameIsFourteenHyphens(t *testing.T) {
	if EmptyName != strings.Repeat("-", 14) {
		t.Errorf("EmptyName is %q", EmptyName)
	}
}

// ZeroAllButHighestScore keeps row zero *and the banner*. The banner is the easy half to
// get wrong, because ZeroHighScores -- the function five lines above it in the original --
// does reset it.
func TestZeroAllButHighestKeepsRowZeroAndTheBanner(t *testing.T) {
	s := board(900, 800, 700)
	before := s.Names[0]
	ZeroAllButHighest(&s)

	if got := s.Banner.Text(); got != "A Banner" {
		t.Errorf("banner became %q; ZeroAllButHighestScore does not touch it", got)
	}
	if s.Scores[0] != 900 || s.Levels[0] != 10 || s.Names[0] != before {
		t.Errorf("row 0 changed: %+v", s.Names[0])
	}
	if n := Occupied(&s); n != 1 {
		t.Errorf("%d rows left occupied, want 1", n)
	}
}

// ------------------------------------------------------------------ qualifying

// Strictly greater, so a tie does not get on the board. This is the rule that produced Fun
// House's 1300, 1300 with the older score above (docs/analysis/scoring.md 7.3).
func TestQualifyNeedsStrictlyMore(t *testing.T) {
	s := board(900, 800, 700)
	for _, tc := range []struct {
		score int32
		want  int
	}{
		{901, 0}, {900, 1}, {899, 1}, {800, 2}, {701, 2}, {700, 3}, {1, 3}, {0, -1}, {-5, -1},
	} {
		if got := Qualify(&s, tc.score); got != tc.want {
			t.Errorf("Qualify(%d) = %d, want %d", tc.score, got, tc.want)
		}
	}

	// A full board is the case that can turn a tie into a place it should not have.
	full := board(10, 9, 8, 7, 6, 5, 4, 3, 2, 1)
	if got := Qualify(&full, 1); got != -1 {
		t.Errorf("a score tying tenth place qualified at row %d", got)
	}
	if got := Qualify(&full, 2); got != 9 {
		// Beating the tenth place but only tying the ninth: last row, not second-to-last.
		t.Errorf("Qualify(2) on a full board = %d, want 9", got)
	}
	if got := Qualify(&full, 3); got != 8 {
		t.Errorf("Qualify(3) on a full board = %d, want 8", got)
	}
}

// --------------------------------------------------------------------- insert

// The property Insert's return value rests on: the score really is in the row it says.
// Worth a table rather than one case, because the claim is about every position including
// both ends and both sides of a tie.
func TestInsertLandsInTheRowItReturns(t *testing.T) {
	when := time.Date(1996, time.March, 3, 12, 0, 0, 0, time.UTC)
	for _, score := range []int32{5000, 901, 900, 899, 800, 701, 700, 500, 1} {
		s := board(900, 800, 700)
		placing := Insert(&s, "Champion", score, 42, when)
		if placing < 0 {
			t.Errorf("Insert(%d) did not qualify", score)
			continue
		}
		if s.Scores[placing] != score {
			t.Errorf("Insert(%d) returned row %d, which holds %d",
				score, placing, s.Scores[placing])
		}
		if got := s.Names[placing].Text(); got != "Champion" {
			t.Errorf("Insert(%d) returned row %d, which is named %q", score, placing, got)
		}
		if s.Levels[placing] != 42 {
			t.Errorf("Insert(%d): row %d has %d rooms, want 42", score, placing, s.Levels[placing])
		}
		if got := house.MacTime(int32(s.TimeStamps[placing])); !got.Equal(when) {
			t.Errorf("Insert(%d): row %d is stamped %v, want %v", score, placing, got, when)
		}
		if !Sorted(&s) {
			t.Errorf("Insert(%d) left the board out of order: %v", score, scoreList(&s))
		}
	}
}

// A score that does not qualify must change nothing at all -- not the rows, not the residue,
// not the banner. The original returns before it writes anything; a port that wrote the row
// first and then checked would leave the loser's name in the tenth slot.
func TestInsertOfALoserChangesNoBytes(t *testing.T) {
	s := board(10, 9, 8, 7, 6, 5, 4, 3, 2, 1)
	before := house.EncodeScores(&s)

	if got := Insert(&s, "Nobody", 1, 99, time.Now()); got != -1 {
		t.Fatalf("a tying score was placed at row %d", got)
	}
	if after := house.EncodeScores(&s); !bytes.Equal(after, before) {
		t.Error("a failed Insert modified the board")
	}
}

// The row that falls off the bottom is the row the new score replaced, and the nine above
// it are unchanged. This is what the write-to-row-9-then-sort shape buys, and it is worth
// pinning because it is the part that looks like a mistake.
func TestInsertPushesExactlyOneRowOff(t *testing.T) {
	s := board(10, 9, 8, 7, 6, 5, 4, 3, 2, 1)
	if got := Insert(&s, "New", 100, 1, time.Now()); got != 0 {
		t.Fatalf("Insert returned %d, want 0", got)
	}
	want := []string{"New", rowName(0), rowName(1), rowName(2), rowName(3),
		rowName(4), rowName(5), rowName(6), rowName(7), rowName(8)}
	if got := names(&s); !equalStrings(got, want) {
		t.Errorf("board is %v,\n want %v", got, want)
	}
	if s.Scores[9] != 2 {
		t.Errorf("tenth row holds %d, want the old ninth (2)", s.Scores[9])
	}
}

// ----------------------------------------------------------------------- sort

// Ties keep their original order, because the C only replaces its candidate on a strict
// improvement. Without this rule a resort of a board with two equal scores would swap them,
// which would make saving a board a diff even when nothing was played.
func TestSortKeepsEqualScoresInOrder(t *testing.T) {
	s := board(500, 500, 500, 500)
	Sort(&s)
	want := []string{rowName(0), rowName(1), rowName(2), rowName(3),
		EmptyName, EmptyName, EmptyName, EmptyName, EmptyName, EmptyName}
	if got := names(&s); !equalStrings(got, want) {
		t.Errorf("equal scores were reordered: %v", got)
	}
}

func TestSortPutsAScrambledBoardInOrder(t *testing.T) {
	var s house.Scores
	order := []int32{300, 900, 0, 100, 700, 0, 500, 200, 0, 400}
	for i, v := range order {
		s.Scores[i] = v
		if v > 0 {
			s.Names[i].SetText(rowName(i))
		} else {
			s.Names[i].SetText(EmptyName)
		}
		s.Levels[i] = int16(v / 100)
	}
	Sort(&s)

	if !Sorted(&s) {
		t.Fatalf("still out of order: %v", scoreList(&s))
	}
	// The room count travelled with the score, not with the row index: this is the check
	// that catches a sort that moves one array and forgets another.
	for i := 0; i < Max; i++ {
		if want := int16(s.Scores[i] / 100); s.Levels[i] != want {
			t.Errorf("row %d: score %d but %d rooms; the arrays came apart",
				i, s.Scores[i], s.Levels[i])
		}
	}
}

// The banner belongs to the board, not to a row, so sorting must not move it. The C copies
// it across separately for exactly this reason (HighScores.c:314) -- its scratch board is
// otherwise uninitialised, so a sort that forgot the copy would replace the banner with
// stack garbage.
func TestSortLeavesTheBannerAlone(t *testing.T) {
	s := board(100, 900, 500)
	before := s.Banner
	Sort(&s)
	if s.Banner != before {
		t.Errorf("the banner became %q", s.Banner.Text())
	}
}

// A name's residue is part of that name and has to travel with it. The original lost this
// -- PasStringCopy copies length+1 bytes into an uninitialised local, so every sort
// scribbled fresh stack garbage past every name -- and reproducing *that* would mean a
// board could never be compared with the board it was seeded from.
func TestSortMovesResidueWithItsName(t *testing.T) {
	s := board(100, 900)
	row0, row1 := s.Names[0], s.Names[1]
	Sort(&s)
	if s.Names[0] != row1 || s.Names[1] != row0 {
		t.Errorf("residue did not follow its name: row 0 is % X, row 1 is % X",
			s.Names[0][:], s.Names[1][:])
	}
}

// Sorting a board that is already sorted must be a no-op on the bytes, which is what makes
// it safe for the side-car to sort on load: an unchanged board that is written back is
// byte-identical to the one that was read.
func TestSortOfASortedBoardChangesNoBytes(t *testing.T) {
	s := board(900, 800, 700, 100)
	before := house.EncodeScores(&s)
	Sort(&s)
	if after := house.EncodeScores(&s); !bytes.Equal(after, before) {
		t.Error("sorting an already-sorted board changed its bytes")
	}
}

// The evidence: 22 boards written by the 1994 sort, and this sort moves nothing in any of
// them, byte for byte. If the tie rule were the other way round, or the residue were
// rebuilt rather than carried, some of these would fail.
func TestSortOfAShippedBoardChangesNothing(t *testing.T) {
	for _, c := range shippedBoards(t) {
		before := house.EncodeScores(&c.board)
		s := c.board
		Sort(&s)
		if after := house.EncodeScores(&s); !bytes.Equal(after, before) {
			t.Errorf("%s: the sort reordered a board the original had already sorted\n"+
				" before %v\n after  %v", c.name, scoreList(&c.board), scoreList(&s))
		}
		if !Sorted(&c.board) {
			t.Errorf("%s: the shipped board is not in descending order: %v",
				c.name, scoreList(&c.board))
		}
		if notes := Repair(&s); len(notes) != 0 {
			t.Errorf("%s: Repair wanted to change a shipped board: %v", c.name, notes)
		}
	}
}

// --------------------------------------------------------------------- repair

func TestRepairClearsANegativeScore(t *testing.T) {
	s := board(900, 800)
	s.Scores[5] = -7
	s.Names[5].SetText("Cheater")
	s.Levels[5] = 3

	notes := Repair(&s)
	if len(notes) == 0 {
		t.Fatal("Repair said nothing about a negative score")
	}
	if s.Scores[5] != 0 || s.Levels[5] != 0 {
		t.Errorf("row 5 is still %d / %d rooms", s.Scores[5], s.Levels[5])
	}
	if got := s.Names[5].Text(); got != EmptyName {
		t.Errorf("row 5 is named %q", got)
	}
	if !Sorted(&s) {
		t.Errorf("Repair left the board out of order: %v", scoreList(&s))
	}
}

// -1 is the sort's own "already taken" marker, so a stored -1 is the value most likely to
// make a hand-edited file behave strangely rather than merely look wrong. Repair has to
// clear it before Sort ever sees it.
func TestRepairClearsTheSortsOwnMarker(t *testing.T) {
	var s house.Scores
	s.Scores = [Max]int32{900, -1, -1, -1, -1, -1, -1, -1, -1, -1}
	s.Names[0].SetText("Champ")
	Repair(&s)
	for i := 1; i < Max; i++ {
		if s.Scores[i] != 0 {
			t.Errorf("row %d still holds %d", i, s.Scores[i])
		}
	}
	if s.Scores[0] != 900 || s.Names[0].Text() != "Champ" {
		t.Errorf("the real row was lost: %d %q", s.Scores[0], s.Names[0].Text())
	}
}

func TestRepairNamesAScoredBlankRow(t *testing.T) {
	s := board(900, 800)
	s.Names[1].SetText("   ")
	notes := Repair(&s)
	if len(notes) != 1 || !strings.Contains(notes[0], "row 2") {
		t.Errorf("notes are %v, want one about row 2", notes)
	}
	if got := s.Names[1].Text(); got != EmptyName {
		t.Errorf("row 1 is named %q", got)
	}
}

func TestRepairSortsAnUnsortedBoard(t *testing.T) {
	s := board(100, 900, 500)
	notes := Repair(&s)
	if len(notes) != 1 || !strings.Contains(notes[0], "order") {
		t.Errorf("notes are %v, want one about the order", notes)
	}
	if !Sorted(&s) {
		t.Errorf("still unsorted: %v", scoreList(&s))
	}
}

func TestRepairOfAGoodBoardSaysNothingAndChangesNothing(t *testing.T) {
	s := board(900, 800, 700)
	before := house.EncodeScores(&s)
	if notes := Repair(&s); len(notes) != 0 {
		t.Errorf("Repair complained about a good board: %v", notes)
	}
	if after := house.EncodeScores(&s); !bytes.Equal(after, before) {
		t.Error("Repair changed a good board")
	}
}

// Repair deliberately does not tidy residue, timestamps or the banner: they cannot make the
// game misbehave, and rewriting bytes that are merely strange is how a port loses the
// ability to say a file round-trips.
func TestRepairLeavesHarmlessOddityAlone(t *testing.T) {
	s := board(900, 800)
	s.TimeStamps[0] = 0xFFFFFFFF // a date in 2040-something
	s.Banner = house.PStr32{31, 0xC4, 0xA5, 0}
	for i := 4; i < len(s.Banner); i++ {
		s.Banner[i] = 0xFE
	}
	before := house.EncodeScores(&s)
	if notes := Repair(&s); len(notes) != 0 {
		t.Errorf("Repair complained: %v", notes)
	}
	if after := house.EncodeScores(&s); !bytes.Equal(after, before) {
		t.Error("Repair tidied something it should have left alone")
	}
}

// ----------------------------------------------------------------------- name

func TestNameStandsInForTheUnreadable(t *testing.T) {
	s := board(900)
	s.Names[1].SetText(EmptyName)
	s.Names[2].SetText("")
	s.Names[3].SetText("    ")
	for _, tc := range []struct {
		row  int
		want string
	}{
		{0, rowName(0)}, {1, "(nameless)"}, {2, "(nameless)"}, {3, "(nameless)"},
		{-1, ""}, {Max, ""},
	} {
		if got := Name(&s, tc.row); got != tc.want {
			t.Errorf("Name(row %d) = %q, want %q", tc.row, got, tc.want)
		}
	}
}

// -------------------------------------------------------------------- helpers

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type shipped struct {
	name  string
	board house.Scores
}

// shippedBoards loads the high-score board out of every extracted house. The extracted
// assets are generated rather than committed, so a missing directory is a skip with
// instructions -- the same rule internal/house's corpus test follows.
func shippedBoards(t *testing.T) []shipped {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "assets", "extracted", "houses")
	paths, err := filepath.Glob(filepath.Join(dir, "*.house"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Skipf("no extracted houses in %s; run `make assets` first", dir)
	}
	out := make([]shipped, 0, len(paths))
	for _, p := range paths {
		h, err := house.LoadFile(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		out = append(out, shipped{
			name:  strings.TrimSuffix(filepath.Base(p), ".house"),
			board: h.HighScores,
		})
	}
	return out
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", dir)
		}
		dir = parent
	}
}
