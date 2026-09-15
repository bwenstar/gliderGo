package game

// The game's half of the high-score subsystem: the room count, the eligibility rules, and
// which of the two game-over paths asks for a score.
//
// The host's half -- the dialogs, the board, the file -- is internal/scores' and is tested
// there. What is tested here is the *contract* on World.HighScore: when it is called, what
// it is handed, and what this package does with its answer.

import (
	"testing"

	"glidergo/internal/house"
)

// scoreWorld is a headless world on a house with n rooms, none of them visited.
func scoreWorld(n int) *World {
	h := oneRoomHouse(house.ObjectIsEmpty)
	for len(h.Rooms) < n {
		h.Rooms = append(h.Rooms, house.Room{Background: SimpleRoom})
	}
	h.NRooms = int16(len(h.Rooms))
	return newTestWorld(h, "", "")
}

// spy records what the hook was handed and answers what it was told to.
type spy struct {
	calls  int
	score  int32
	rooms  int16
	answer bool
}

func (s *spy) hook(score int32, rooms int16) bool {
	s.calls++
	s.score, s.rooms = score, rooms
	return s.answer
}

// ---------------------------------------------------------------- room counting

func TestCountRoomsVisitedCountsTheFlagAndNothingElse(t *testing.T) {
	w := scoreWorld(6)
	if got := w.CountRoomsVisited(); got != 0 {
		t.Errorf("a fresh house has %d rooms visited", got)
	}

	w.H.Rooms[0].Visited = 1
	w.H.Rooms[3].Visited = 1
	w.H.Rooms[5].Visited = 1
	if got := w.CountRoomsVisited(); got != 3 {
		t.Errorf("three visited rooms counted as %d", got)
	}

	// The flag is a byte on disk and the C tests it for truth, not for equality with 1.
	// A house edited by hand can hold any non-zero value there.
	w.H.Rooms[1].Visited = 200
	if got := w.CountRoomsVisited(); got != 4 {
		t.Errorf("a non-1 visited byte was not counted: %d", got)
	}
}

// The number on the board is the number the score agrees with: HandleRoomVisitation credits
// the room being *left*, so walking out of the first room is what makes it count. A player
// who never leaves the first room finishes with zero rooms and zero points, and those two
// zeroes are the same fact.
func TestTheRoomCountAgreesWithTheScore(t *testing.T) {
	w := scoreWorld(3)
	w.R.RoomNumber = 0 // the room the glider is standing in

	if got := w.CountRoomsVisited(); got != 0 || w.Score != 0 {
		t.Fatalf("before anything: %d rooms, %d points", got, w.Score)
	}

	w.HandleRoomVisitation()
	if got := w.CountRoomsVisited(); got != 1 {
		t.Errorf("after leaving one room: %d rooms visited, want 1", got)
	}
	if w.Score != RoomVisitScore {
		t.Errorf("after leaving one room: %d points, want %d", w.Score, RoomVisitScore)
	}

	// And a second departure from the same room credits neither, which is what keeps a
	// player from farming a doorway.
	w.HandleRoomVisitation()
	if got := w.CountRoomsVisited(); got != 1 {
		t.Errorf("re-leaving the same room counted it twice: %d", got)
	}
	if w.Score != RoomVisitScore {
		t.Errorf("re-leaving the same room scored twice: %d", w.Score)
	}
}

// ------------------------------------------------------------------ eligibility

// A build with no host cannot ask for a name, so it does not qualify anybody. This is the
// case the fidelity corpus runs in: a replay's outcome must not depend on whether the
// machine running it has a board file.
func TestTestHighScoreWithoutAHostIsFalseAndSilent(t *testing.T) {
	w := scoreWorld(2)
	w.Score = 10000
	if w.TestHighScore() {
		t.Error("a headless world claimed a high score")
	}
}

// The one hard rule in the whole subsystem (HighScores.c:381): a resumed game is
// ineligible however well it goes, and the hook is never even called -- so the player is
// not asked for a name and then quietly denied.
func TestAResumedSavedGameIsIneligible(t *testing.T) {
	w := scoreWorld(2)
	w.Score = 1000000
	w.H.Rooms[0].Visited = 1
	var s spy
	s.answer = true
	w.HighScore = s.hook
	w.ResumedSavedGame = true

	if w.TestHighScore() {
		t.Error("a resumed game qualified")
	}
	if s.calls != 0 {
		t.Errorf("the hook was called %d times for an ineligible game", s.calls)
	}

	// And clearing the flag makes the same world eligible, so the guard is the flag and
	// not something else about the fixture.
	w.ResumedSavedGame = false
	if !w.TestHighScore() {
		t.Error("the same world did not qualify with the flag clear")
	}
	if s.calls != 1 {
		t.Errorf("the hook was called %d times, want 1", s.calls)
	}
}

// What the hook is handed: the score as it stands, unclamped, and the room count.
func TestTestHighScorePassesTheScoreAndTheRoomCount(t *testing.T) {
	w := scoreWorld(5)
	w.Score = 4230
	w.DisplayedScore = 100 // mid-roll: the board shows one number and Score is the other
	w.H.Rooms[0].Visited = 1
	w.H.Rooms[2].Visited = 1
	var s spy
	w.HighScore = s.hook

	w.TestHighScore()
	if s.score != 4230 {
		t.Errorf("the hook was handed %d, want the authoritative Score of 4230", s.score)
	}
	if s.rooms != 2 {
		t.Errorf("the hook was handed %d rooms, want 2", s.rooms)
	}

	// Unclamped and unvalidated (docs/analysis/scoring.md 7.5 note 4). A negative score
	// cannot arise in play, but nothing here stops one, and the board's own Qualify is
	// what refuses it.
	w.Score = -50
	w.TestHighScore()
	if s.score != -50 {
		t.Errorf("the score was adjusted on the way through: %d", s.score)
	}
}

// The answer is passed through, because DoGameOver's splash-redraw decision is exactly it.
func TestTestHighScoreReturnsWhatTheHostSays(t *testing.T) {
	for _, want := range []bool{true, false} {
		w := scoreWorld(2)
		s := spy{answer: want}
		w.HighScore = s.hook
		if got := w.TestHighScore(); got != want {
			t.Errorf("the host said %v and TestHighScore said %v", want, got)
		}
	}
}

// --------------------------------------------------------------- the two paths

// The win path asks, always. Its Boolean is what suppresses the splash redraw
// (GameOver.c:68), which is not observable until 1.7d draws a splash screen; what is
// observable now is that the question gets asked and that the loop is ended either way.
func TestDoGameOverAsksForAHighScore(t *testing.T) {
	for _, answer := range []bool{true, false} {
		w := scoreWorld(2)
		w.Score = 5000
		w.Playing = true
		s := spy{answer: answer}
		w.HighScore = s.hook

		w.DoGameOver()
		if s.calls != 1 {
			t.Errorf("host answer %v: the hook was called %d times, want 1", answer, s.calls)
		}
		if w.Playing {
			t.Errorf("host answer %v: DoGameOver left the loop running", answer)
		}
	}
}

// The loss path asks too -- and a demo does not. An attract mode that could put scores on
// the board would fill it on an unattended machine.
func TestDoDiedGameOverAsksExceptInADemo(t *testing.T) {
	for _, tc := range []struct {
		demo bool
		want int
	}{{false, 1}, {true, 0}} {
		w := scoreWorld(2)
		w.Score = 5000
		w.Playing = true
		w.DemoGoing = tc.demo
		var s spy
		w.HighScore = s.hook

		w.DoDiedGameOver()
		if s.calls != tc.want {
			t.Errorf("demo=%v: the hook was called %d times, want %d",
				tc.demo, s.calls, tc.want)
		}
		if w.Playing {
			t.Errorf("demo=%v: DoDiedGameOver left the loop running", tc.demo)
		}
	}
}

// Neither path may write to the house. The original's TestHighScore edits the board inside
// the house handle and sets gameDirty, which is what would eventually rewrite the house
// file; this port keeps the twenty-two shipped houses read-only and puts a new board in a
// side-car instead (docs/analysis/scoring.md 7.1). A regression here would be a game
// silently modifying vendored data.
func TestNeitherGameOverPathTouchesTheHousesOwnBoard(t *testing.T) {
	for _, name := range []string{"win", "loss"} {
		w := scoreWorld(3)
		w.Score = 999999
		w.Playing = true
		w.H.HighScores.Banner.SetText("the house's own banner")
		w.H.HighScores.Names[0].SetText("Somebody Else")
		w.H.HighScores.Scores[0] = 100
		before := house.EncodeScores(&w.H.HighScores)

		// A host that would have recorded the score, if this package let it.
		w.HighScore = func(int32, int16) bool { return true }
		if name == "win" {
			w.DoGameOver()
		} else {
			w.DoDiedGameOver()
		}

		if after := house.EncodeScores(&w.H.HighScores); string(after) != string(before) {
			t.Errorf("%s: the house's own high-score table was modified", name)
		}
	}
}
