package game

// The game's half of the high-score subsystem: TestHighScore (HighScores.c:374-426) and
// CountRoomsVisited (House.c:511-531), transcribed from docs/analysis/scoring.md 7.5.
//
// Almost all of that C is not this package's business. Of the twenty numbered steps in the
// analysis's pseudocode, fourteen are a modal dialog, a selection sort, a screen and a
// dirty flag -- and every one of those has moved to internal/scores, which owns the board's
// model, its file and its two prompts, or to the host, which owns the window. What is left
// here is the four things only a running game knows:
//
//	is this game eligible          resumedSavedGame, the one hard rule (7.13)
//	what did the player score      theScore, unclamped and unvalidated (7.5 note 4)
//	how many rooms did they see    CountRoomsVisited, over the *in-memory* house
//	does the caller carry on       the Boolean DoGameOver tests (8.5)
//
// The line between the two halves is the same one PlayEvent and Pause are drawn along, and
// it is drawn here for the same reason: what the game knows does not vary by platform, and
// what blocks does. See World.HighScore.
//
// One thing the C does here is deliberately *not* transcribed. Its step 18 sets
// `gameDirty = true`, which is what eventually makes CloseHouse rewrite the house file with
// the new board inside it (7.13). This port never writes a house file it did not author: the
// twenty-two shipped houses are vendored read-only, and a board earned in one of them goes
// into a side-car of the port's own instead. So there is no dirty flag to set and nothing in
// this package touches w.H. internal/scores.Store is where the write happens, and the reason
// it is a separate file rather than a field in the house is docs/analysis/scoring.md 7.1 and
// 7.14: the original had a side-car too, and its reader was the bug that made it dead code.

// CountRoomsVisited is House.c:511-531: how many of the house's rooms the player has been
// in this game.
//
// It is the `levels` column on the high-score board -- "12 rooms" -- and the only place that
// column comes from. Three properties of it are worth keeping in mind, all from the
// analysis's note 8:
//
//   - **It counts the in-memory house**, which is this port's only house: there is no
//     separate working copy to disagree with (see HandleRoomVisitation).
//   - **It includes the room the player is standing in**, because HandleRoomVisitation
//     marked it on the way in.
//   - **It excludes rooms only seen through a window.** Looking into a room is not visiting
//     it; the flag is set by walking out of one.
//
// So the number on the board is one *less* than a player who has walked through n doors
// would count, in the same way their score is: the first room of a house is credited when
// they leave it, not when the game starts. The two numbers are consistent with each other,
// which is what matters -- see docs/analysis/scoring.md 2.3.
//
// The C's HGetState/HLock/HSetState around the walk is a Memory Manager handle lock and has
// no counterpart here. The return type is the C's `short`, kept because the board's field is
// an `int16` on disk and a house with more than 32767 rooms cannot be loaded anyway.
func (w *World) CountRoomsVisited() int16 {
	var n int16
	for i := range w.H.Rooms {
		if w.H.Rooms[i].Visited != 0 {
			n++
		}
	}
	return n
}

// TestHighScore is HighScores.c:374-426: decide whether this game's score belongs on the
// board and, if it does, let the host collect a name and show it.
//
// It answers `placing != -1` -- true if the score qualified. That Boolean has exactly one
// reader, DoGameOver, and one thing it means there: whether the splash screen still needs
// redrawing (GameOver.c:68). DoDiedGameOver calls this and throws the answer away, redrawing
// the splash either way, which is an inconsistency between the two paths rather than a
// design; both are transcribed as they are.
//
// The two guards are the whole of the port's own logic:
//
//   - ResumedSavedGame is the C's own first line and is a rule, not an implementation
//     detail: a game continued from a save is ineligible however well it goes (7.13).
//   - A nil hook is a headless build, and false is the right answer for one. Nobody can
//     type a name with no keyboard, and -- more to the point -- a fidelity replay's outcome
//     must not depend on whether a board file happens to exist on the machine running it.
//
// Everything the C does between those two lines and its return happens on the far side of
// the hook. What arrives there is a score and a room count; what comes back is a Boolean.
// The host is free to decide the score does not qualify, which is what makes the split
// honest: qualification is a property of a board this package cannot see.
func (w *World) TestHighScore() bool {
	// `if (resumedSavedGame) return false` -- HighScores.c:381, before the house is even
	// locked. Nothing else in the function runs, so an ineligible game shows no dialog and
	// no board.
	if w.ResumedSavedGame {
		return false
	}
	if w.HighScore == nil {
		return false
	}
	// `theScore` is passed as it stands. It is not clamped and not validated (7.5 note 4):
	// whatever the score roll left behind is what goes on the board, and the roll can be
	// mid-flight when a game ends -- see HandleDynamicScoreboard, where DisplayedScore
	// chases Score and only Score is authoritative.
	return w.HighScore(w.Score, w.CountRoomsVisited())
}
