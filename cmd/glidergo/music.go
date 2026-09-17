package main

// The score on the title screen -- `music_on_title`, the last thing 1.7 owed.
//
// The original plays music while nobody is playing: `StartIdleMusic` asks for the whole
// score from the top, and the splash screen, the house picker and the settings dialog all
// have it behind them. It is a preference of its own (`isPlayMusicIdle`) precisely because
// wanting the score during a game and wanting it while choosing a house are different
// wants.
//
// The reason it took until now is one line of package layout. On the Mac the cursor into
// the score is a file-scope global (Music.c:31-35), so *anything* can walk it with no game
// in existence. This port hangs that state on a `game.World`, deliberately -- the cursor is
// game state, six room-change handlers nudge it, and a saved game will have to carry it
// (internal/game/music.go) -- and the shell has no World and is not allowed to have one:
// internal/shell does not import internal/game, which is what keeps the whole way into the
// game testable with no house and no sound card (docs/PLAN.md, 1.7a's first decision).
//
// So the bridge lives here, in the one package that already imports both, and it is much
// smaller than the stage note feared. A `World` needs *nothing* except a music channel to
// walk a score: no house, no scene, no surfaces, no assets. The title screen's cursor is a
// bare `&game.World{Music: a.eng}`, and every function below is one of the transcribed
// ladders called on it. Nothing about the score is reimplemented here, which is the point
// -- `StartIdleMusic` keeps being the only thing that knows what idle music means.

import "github.com/bwenstar/gliderGo/internal/game"

// titleScore returns the World that owns the title screen's cursor into the score,
// building it on first use, or nil for a session with no mixer.
//
// `InitMusic` is what arms a World's music at all -- `NewWorld` sets `DontLoadMusic` and
// only this clears it -- and it is called here with the idle preference still false on
// purpose: that seeds the cursor at the top of the score without committing a piece, so
// starting and stopping afterwards is `StartIdleMusic`'s decision alone and reads the
// preference at the moment it matters rather than at the moment the window opened.
func (a *app) titleScore() *game.World {
	if a.eng == nil {
		return nil
	}
	if a.title == nil {
		a.title = &game.World{Music: a.eng}
		a.title.InitMusic()
	}
	return a.title
}

// startTitleMusic starts, stops or leaves the score alone according to `music_on_title`.
//
// It is the whole of the title screen's music policy, and it is called from three places
// -- the shell starting up, a game handing control back, and the settings screen changing
// something -- because in all three the question is the same one and the answer is
// `StartIdleMusic`'s: start it if the preference is on and nothing is playing, stop it if
// the preference is off and something is, and otherwise just make sure the mode is the
// whole score rather than the six-piece in-game loop.
//
// Calling it repeatedly is free, which is why the settings screen can call it on every
// keystroke. It is also how the score recovers from a mute: at volume zero
// `Engine.MusicAvailable` refuses to start and `MusicOn` stays false, so raising the
// volume and calling again starts it (internal/audio/music.go).
func (a *app) startTitleMusic() {
	w := a.titleScore()
	if w == nil {
		return
	}

	// Re-pointed here rather than once at startup, because a game points it at its own
	// World (bindAudio) and that World is finished by the time control comes back. Before
	// this existed nothing ever pointed it back: the mixer kept asking a retired game's
	// World for pieces, and the score the player heard over the title screen was being
	// walked by a game that had ended -- which also meant every game of a session was
	// still reachable from the engine and could not be collected. The assignment is safe
	// unsynchronised for the reason internal/audio/music.go gives: NextPiece is called
	// from inside Mix, and Mix runs on whichever goroutine clocks the pump, which is this
	// one.
	a.eng.NextPiece = w.NextMusicPiece

	w.PlayMusicIdle = a.p.MusicOnTitle
	w.StartIdleMusic()
}

// stopTitleMusic silences the score on the way into a game.
//
// The C does not need this -- one global cursor means `StartGameMusic` finds the music
// already on and only changes mode, so the score crosses into a game without a gap. Here
// the game gets its own World whose `InitMusic` seeds a fresh cursor, and leaving the
// title screen's two committed pieces in the queue underneath it would play the whole
// score and the game score at once out of one channel. So the transition costs a moment
// of silence, and that is the trade: a doubled queue is a bug a player can hear, and a
// gap is not.
func (a *app) stopTitleMusic() {
	if a.title != nil {
		a.title.StopTheMusic()
	}
}

// adoptScore hands a finished game's music state to the title screen's cursor.
//
// `NewGame`'s teardown ends every game by calling that World's own `StartIdleMusic`, so on
// the Mac the title screen is *already* playing idle music by the time control comes back,
// out of the same global. Copying the four fields is how two Worlds reproduce that, and the
// field that earns it is `MusicOn`: coming across true, it tells the next `StartIdleMusic`
// there is nothing to start, only a mode to set, and the queued reserve plays on without a
// gap. Coming across false -- a game played with the score switched off -- it starts the
// score, which is also what the C does.
//
// The cursor is copied for completeness and not for effect. `StartIdleMusic` ends in
// `SetMusicalMode(PlayWholeScoreMode)`, whose default arm resets the cursor to zero
// (internal/game/music.go), so the title screen resumes at the top of the score rather than
// where the game left off -- in the C as well. What it does not do is jump: the two pieces
// already committed are what the player hears while the cursor goes back to the beginning.
func (a *app) adoptScore(w *game.World) {
	if a.title == nil || w == nil {
		return
	}
	a.title.MusicCursor = w.MusicCursor
	a.title.MusicSoundID = w.MusicSoundID
	a.title.MusicMode = w.MusicMode
	a.title.MusicOn = w.MusicOn
}
