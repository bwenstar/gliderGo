package game

// Saved games: SavedGames.c's live half, and the two doors DoCommandKey opens.
//
// # What the original has
//
// Three functions and a global. `smallGame` (SavedGames.c:20) is the game a resume starts
// from; SaveGame2 writes a file, OpenSavedGame reads one back into that global, and
// SaveGame(Boolean) writes the same fields into the *house* instead. DoCommandKey
// (Input.c:53-73) is the only thing that calls any of them:
//
//	if Q:  playing = false; paused = false
//	       if (!twoPlayerGame && !demoGoing) && QuerySaveGame(): SaveGame2()
//	if S:  RefreshScoreboard(kSavingTitleMode); SaveGame2(); HideCursor()
//	       CopyRectWorkToMain(&workSrcRect); RefreshScoreboard(kNormalTitleMode)
//
// **None of it runs.** SaveGame2's body is commented out but for the note `// Add
// NavServices later.`; OpenSavedGame's first live statement is `return false;  // TEMP fix
// this iwth NavServices`; SaveGame is whole but its one call site (Menu.c:458) is commented
// out, and QueryResumeGame -- the dialogue that would have offered a resume -- is never
// called from anywhere. So in the shipped 1.1.2 a game could not be saved and
// kResumeGameMode was unreachable, which is why two houses ship with a saved game inside
// them that nothing has ever loaded (docs/analysis/scoring.md 9.5).
//
// That makes this stage a reconstruction rather than a transcription, and it is the only
// stage of 1.x that is. What is reconstructed is narrow: the *field list* below is
// SaveGame2's, in its order, and the restore half is the code that was already live and is
// already ported (InitGlider's resume arm, SetHouseToSavedRoom, WhereDoesGliderBegin,
// TestHighScore's `resumedSavedGame` bar). What is decided rather than transcribed is where
// the file goes and what wraps the bytes, and those decisions are in internal/house's
// savedgame.go (S1-S4) and internal/saved.
//
// # The split
//
// This file owns *what* a saved game is; the host owns where it goes. CaptureSavedGame
// builds one and hands it to World.SaveGame, which cmd/glidergo points at internal/saved.
// A nil hook is a build with nowhere to write -- a replay, a screenshot, `-saves none` --
// and DoSaveGame then draws the title, writes nothing and carries on, which is the honest
// rendering of a session that can play and cannot record.
//
// The other direction is ResumeSavedGame, and it is the only writer of World.SavedGame and
// World.ResumedSavedGame. Both of those are load-bearing in ways a caller cannot see:
// NewGame(ResumeGameMode) reads the first for the glider's position, the score, the lives
// and the whole inventory, and TestHighScore reads the second to bar a resumed game from
// the board. One function setting both is what keeps them agreeing.

import (
	"fmt"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
)

// CanSaveGame reports whether this game may be written to disk at all.
//
// The C spreads this over three places and they do not agree. SaveGame's own first
// statement is `if (twoPlayerGame) return;`; DoCommandKey's S arm tests `!twoPlayerGame`
// again; and only the Q arm tests `!demoGoing` as well. So in the original, pressing
// Command-S during the attract-mode demo would have saved the demo's game over the
// player's own -- GetDemoInput calls DoCommandKey on every frame outside the arcade build
// (Input.c:238). It cannot bite because SaveGame2 is a no-op, and it is not reproduced
// here: the demo is not a game anybody is playing, so the demo test belongs in both arms.
//
// Two players cannot be saved for a real reason rather than an arbitrary one. `gameType`
// has one glider's position, one mode and one facing (docs/analysis/scoring.md 9.5), so
// there is nowhere to put the second player; a two-player save would resume as a
// one-player game with two players' lives. Refusing is the original's answer and the right
// one. Stage 3's networked race is a separate format question and is noted in
// docs/IMPROVEMENTS.md.
//
// The nil-hook test is the port's third clause, and it is what lets the host advertise the
// key only when it works: a pause hint that offers a save this build cannot perform is
// worse than one that says nothing.
func (w *World) CanSaveGame() bool {
	return !w.TwoPlayer && !w.DemoGoing && w.SaveGame != nil
}

// CaptureGame is SaveGame2's seventeen assignments (SavedGames.c:89-106), in their order.
//
// The order is kept because it is the only description of the format that the original ever
// wrote down as code, and a field-by-field reading against it is how this was checked.
// Three of the seventeen are not here:
//
//   - `house`, the FSSpec. It is a volume reference, a directory ID and a file name, and
//     two of the three are meaningless off a 1994 Macintosh. internal/house keeps the
//     third -- the name -- at the offset the FSSpec's `name` field had, which is what makes
//     a gliderGo save byte-identical to a `game2Type` from offset 6 on (S2).
//   - `nRooms`, which shares its two bytes with gameType's `unusedShort`. Encode writes it
//     from the length of the room slice, so it cannot disagree with what follows it.
//   - `hasGame`, which lives in the house rather than in the game and is SaveGame's, not
//     SaveGame2's.
//
// Two are worth reading twice. `timeStamp` is **the house's**, not the clock's: it is the
// key OpenSavedGame's second gate compares, so a save that stamped itself with the moment
// it was written could never be reopened. (SaveGame, the other writer, does exactly that --
// `GetDateTime(&stamp)` at SavedGames.c:320 -- and since nothing ever read its output back,
// nothing ever noticed. See internal/saved's Check.) And `where` is the glider's *dest*
// rather than its position-plus-velocity, so a save taken mid-flight resumes standing
// still: the C's own resume arm reads no velocities at all.
func (w *World) CaptureGame() house.Game {
	g := house.Game{
		// savedGame->version = kSavedGameVersion. This port writes 0x0200 and reads
		// 0x0100 as well, because every save in the shipped corpus is a 0x0100 (S3).
		Version:      house.SavedGameVersion,
		WasStarsLeft: w.StarsLeft,
		TimeStamp:    w.H.TimeStamp,
		Where:        house.Point{V: w.P1.Dest.Top, H: w.P1.Dest.Left},
		Score:        w.Score,
		UnusedLong:   0,
		UnusedLong2:  0,
		Energy:       w.Battery,
		Bands:        w.Bands,
		RoomNumber:   w.R.RoomNumber,
		GliderState:  w.P1.Mode,
		NumGliders:   w.Mortals,
		Foil:         w.Foil,
		// UnusedShort is nRooms in a game2Type and 0 in a houseType's block. Encode
		// overwrites it either way; see house.SavedGame.Encode.
		//
		// Both of these are a Mac `Boolean` -- one byte -- and not the shorts the
		// surrounding fields are. `facing` is assigned straight from theGlider.facing,
		// whose kFaceRight is TRUE (GliderDefines.h:554), so the port's Facing bool maps
		// to 1 and 0 with no reinterpretation. InitGlider's resume arm reads it back as
		// `!= 0`, which is what makes any nonzero byte from a hand-edited save mean right.
		Facing:   boolByte(w.P1.Facing),
		ShowFoil: boolByte(w.ShowFoil),
	}
	return g
}

// CaptureSavedGame is CaptureGame plus the two things a file needs that a house's embedded
// block does not: the name of the house it belongs to, and every room's mutable state.
//
// The rooms are the difference between the two formats and between the two experiences. A
// `gameType` restores the player; a `game2Type` restores the *house* -- every switch thrown,
// every prize taken, every room visited -- and without them a resumed game would put the
// player back in room 40 with all forty rooms' prizes waiting to be collected again.
// house.CaptureSavedRooms is the snapshot and it copies exactly what a game can change.
func (w *World) CaptureSavedGame() *house.SavedGame {
	sg := &house.SavedGame{
		Format: house.SavedGameFormat,
		Game:   w.CaptureGame(),
		Rooms:  w.H.CaptureSavedRooms(),
	}
	// A name longer than 63 bytes is truncated rather than refused. It cannot happen from
	// a real house -- these are file names -- and a save the player cannot make is worse
	// than a name the player can already see is long.
	sg.HouseName.SetText(w.HouseName)
	return sg
}

// DoSaveGame is DoCommandKey's Command-S arm (Input.c:65-71): five statements, of which
// four are about the screen.
//
// The screen work is the whole reason this is a method on World rather than something the
// host does for itself. Saving happens from inside GetInput, mid-frame, with the pause
// placard possibly on top of the play area, and the C's answer is to flip the scoreboard's
// title to "Saving Game", let the file dialogue happen over the window, then put the play
// area back from the work map and flip the title back. Every one of those still applies
// here except the dialogue.
//
// Two departures:
//
//   - It presents. The C draws straight into the window, so RefreshScoreboard *is* the
//     player seeing it; here a composited frame has to be pushed. Without the first
//     present, "Saving Game" would be composed and overwritten in the same frame and the
//     player would never learn that the key did anything -- and on a slow or full disk
//     they would see nothing at all while the game appeared to hang.
//   - HideCursor is dropped, as everywhere else in the port: there is no Toolbox cursor.
//
// The error is discarded, which matches SaveGame2's void return and its caller's total
// disinterest. It is not lost: World.SaveGame's own comment says the host keeps it, and
// cmd/glidergo puts it in the pause hint, which is the one line a paused player is reading.
func (w *World) DoSaveGame() {
	// `if (twoPlayerGame)` twice over -- see CanSaveGame. A refused save still draws
	// nothing and says nothing here, because the host has already decided not to offer
	// the key; this is the backstop, not the message.
	if !w.CanSaveGame() {
		return
	}

	w.RefreshScoreboard(SavingTitleMode)
	w.present()

	_ = w.SaveGame(w.CaptureSavedGame())

	// CopyRectWorkToMain(&workSrcRect): the whole play area, back from the last
	// composited frame. In the C this is what erases the file dialogue; here it erases the
	// pause placard, which the pause loop's next pass draws again (see pause.go's paint
	// contract). The C had no such loop, so saving during a 1994 pause wiped the placard
	// for good and left a paused game looking live.
	w.CopyRectWorkToMain(player.Rect(w.R.V.WorkRect))
	w.RefreshScoreboard(NormalTitleMode)
	w.present()
}

// GiveUpGame is DoCommandKey's Command-Q arm (Input.c:55-63), minus the question.
//
// The two assignments are the arm's whole effect on the game: `playing = false` ends
// PlayGame's loop at the top of the next frame and `paused = false` ends the pause loop the
// keystroke arrived in. Clearing Playing rather than Quitting is what the C does and is the
// more faithful of the two, because PlayGame tests it a second time mid-frame (Play.c:499)
// -- so the frame the player gave up on is simulated and never drawn, exactly as in 1994.
//
// `save` is QuerySaveGame's answer (alert 1041, "Save game?" with a Yes button), asked by
// the host because it is a question and this package has no dialogues. The gate stays here:
// a host that asked the question of a two-player game would get a false yes, so the answer
// is filtered through CanSaveGame rather than trusted.
//
// The C's `!twoPlayerGame && !demoGoing` guard is CanSaveGame's, and note where the C puts
// it -- around the *question*, not around the save. A two-player game is never asked.
// cmd/glidergo keeps that: with nothing to save, Q gives up immediately.
func (w *World) GiveUpGame(save bool) {
	w.Playing = false
	w.Paused = false
	if save {
		w.DoSaveGame()
	}
}

// ResumeSavedGame installs a saved game, and is the only way a game can be resumed.
//
// It is OpenSavedGame's tail (SavedGames.c:261-284): the `smallGame` fill and the per-room
// copy back into the house. Everything before that tail in the C -- the file, the four
// validation gates, the alerts -- is the host's, and the gates are in internal/saved's
// Check, which is where a message for the player can be written.
//
// Two things it does that the C does not, both cheap and both preventing a half-started
// game:
//
//   - The room number is checked against the house. The C's fourth gate compares room
//     *counts* and then trusts the number, so a save naming room 40 of a 40-room house
//     reached ForceThisRoom, which raises kYellowIllegalRoomNum from inside NewGame with
//     the world already half torn down (see game/room.go). Here it is refused before
//     anything is written.
//   - Nothing is assigned until everything has been validated, so a refused resume leaves
//     the World fit to start a new game. house.ApplySavedRooms has the same property.
//
// A nil Rooms slice means "this save carries no room snapshot", which is not the same as a
// save carrying zero rooms (that is a mismatch, and is refused). It is how a house's own
// embedded 40 bytes are resumed -- the two shipped ones, Titanic and ImagineHouse PRO II -- where
// the objects the game starts from are the ones the house file itself holds. That is
// precisely what kResumeGameMode would have done in 1994, since nothing there ever loaded a
// room snapshot into a house at all.
func (w *World) ResumeSavedGame(sg *house.SavedGame) error {
	if sg == nil {
		return fmt.Errorf("game: no saved game to resume")
	}
	if w.H == nil {
		return fmt.Errorf("game: no house to resume into")
	}
	if n := sg.Game.RoomNumber; n < 0 || int(n) >= len(w.H.Rooms) {
		return fmt.Errorf("game: saved game names room %d of %d", n, len(w.H.Rooms))
	}
	if sg.Rooms != nil {
		if err := w.H.ApplySavedRooms(sg.Rooms); err != nil {
			return fmt.Errorf("game: %w", err)
		}
	}

	w.SavedGame = sg.Game
	// resumedSavedGame (HighScores.c:33), and the reason it is set here rather than by the
	// caller: it is the only thing standing between a resumed game and the high-score
	// board, and a game that reached the board by a route that forgot to set it would be
	// cheating with the port's help.
	w.ResumedSavedGame = true
	return nil
}

// boolByte is the port's implicit `Boolean` conversion: the last two fields of a gameType
// are single bytes and the C assigns Booleans straight into them. Writing 1 and 0 rather
// than any other nonzero is what keeps a save this port wrote legible to the 1994 field
// list, which compares `facing` for equality in one place (the editor's glider marker) and
// for truth everywhere else.
func boolByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}
