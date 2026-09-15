package game

// Music.c, less the mixing: the mode, the score, the walk over it, and the three functions
// that start and stop the channel.
//
// The channel itself is internal/audio -- four sampled voices and the arithmetic that sums
// them -- and the line between the two packages is drawn at MusicChannel, three methods
// down the file. What stays here is the *score*: which piece is next, which is a question
// about game state and not about audio, since the cursor is nudged from six room-change
// handlers and will have to survive into a saved game.
//
// ---------------------------------------------------------------------------
// SetMusicalMode (Music.c:146-166)
// ---------------------------------------------------------------------------
//
// What it does is *not* what its name suggests, and the asymmetry is worth reading before
// the rest of the file:
//
//	kKickGameScoreMode  -> musicCursor = 2       and musicMode is left alone
//	kProdGameScoreMode  -> musicCursor = -1      and musicMode is left alone
//	anything else       -> musicMode = newMode, musicCursor = 0
//
// So the two modes the transit code passes never change the mode at all. They nudge
// the cursor, and the callback -- which only walks the score while musicMode is
// kPlayGameScoreMode or kPlayWholeScoreMode -- picks the nudge up on its next tick.
// A "kick" jumps to piece 2 of the game score; a "prod" sets the cursor to -1, which
// the callback's increment turns into 0 and then reads gameScore[0].
//
// So calling this with kKickGameScoreMode while the music is stopped does nothing
// observable at all, and with the music playing it changes what the *next* piece will
// be and not what is playing now. Either way it never interrupts a note, which is why
// six room-change handlers can call it freely.

// The four musical modes (GliderDefines.h:47-50).
//
// The first two are commands rather than states, for the reason above. The last two
// are the states the callback switches on: PlayGameScoreMode loops the in-game score
// from piece 1, PlayWholeScoreMode plays the whole thing from piece 0 and is what
// FlagGameOver switches to.
const (
	ProdGameScoreMode  int16 = -4
	KickGameScoreMode  int16 = -3
	PlayGameScoreMode  int16 = -2
	PlayWholeScoreMode int16 = -1
)

// SetMusicalMode is Music.c:146-166.
//
// The DontLoadMusic guard is the C's first statement and is not a courtesy: with music
// off, the mode and the cursor must stay where they were, because the mixer reads them
// through NextMusicPiece and a stale cursor from a game played before the player turned
// music on is what the original leaves there too.
func (w *World) SetMusicalMode(newMode int16) {
	if w.DontLoadMusic {
		return
	}

	switch newMode {
	case KickGameScoreMode:
		w.MusicCursor = 2

	case ProdGameScoreMode:
		w.MusicCursor = -1

	default:
		w.MusicMode = newMode
		w.MusicCursor = 0
	}
}

// ---------------------------------------------------------------------------
// The two ladders NewGame opens and closes with (Play.c:83-101, :237-252)
// ---------------------------------------------------------------------------
//
// They are here rather than in play.go because they are Music.c's logic seen from a
// caller, and because the second one is not the inverse of the first: the game score
// and the splash score are separate preferences, so a player can have music in menus
// and silence in play, or the reverse. Each ladder is five lines in the C and they
// differ only in the preference read and the mode passed.

// StartGameMusic is Play.c:83-101: begin the in-game score, or stop the music.
//
// The `!isMusicOn` guard is what stops a second channel being opened when the splash
// score was already playing -- StartMusic opens the channel, SetMusicalMode only
// steers it. And the `else` branch is not a no-op: coming from a splash screen with
// music into a game with the game-music preference off has to *stop* the score, which
// is the one path in the game that silences music mid-session.
//
// The YellowAlert on failure sets failedMusic and plays on, so a machine with no sound
// memory got one alert per game and then silence. The port keeps the flag and drops the
// alert until 1.7 has somewhere to put it.
func (w *World) StartGameMusic() {
	if w.PlayMusicGame {
		if !w.MusicOn {
			w.StartMusic()
		}
		w.SetMusicalMode(PlayGameScoreMode)
	} else if w.MusicOn {
		w.StopTheMusic()
	}
}

// StartIdleMusic is Play.c:237-252: the same ladder on the way out, reading the idle
// preference and asking for the whole score from the top rather than the in-game loop.
func (w *World) StartIdleMusic() {
	if w.PlayMusicIdle {
		if !w.MusicOn {
			w.StartMusic()
		}
		w.SetMusicalMode(PlayWholeScoreMode)
	} else if w.MusicOn {
		w.StopTheMusic()
	}
}

// ---------------------------------------------------------------------------
// The channel, and the score
// ---------------------------------------------------------------------------

// MusicChannel is the mixer's music channel as the game sees it: three methods, none of
// which knows what a score is.
//
// internal/audio.Engine satisfies it. The split is drawn here rather than one method
// wider because the *cursor* is game state -- SetMusicalMode nudges it from six
// room-change handlers, and a saved session will have to carry it -- so the walk that
// reads and writes it stays in this package and the mixer asks for a piece number when
// it needs one. See NextMusicPiece.
type MusicChannel interface {
	// MusicAvailable is StartMusic's two guards (Music.c:52-57): there is a music
	// system and the machine is not muted. The game reads it into MusicOn, so an
	// optimistic answer would make every later ToggleMusicWhilePlaying act on a lie.
	MusicAvailable() bool

	// QueueMusic is one of StartMusic's bufferCmds: commit a piece to play after
	// whatever is already queued.
	QueueMusic(piece int16)

	// SilenceMusic is StopTheMusic's flushCmd and quietCmd.
	SilenceMusic()
}

// The score, from InitMusic (Music.c:332-354). Written once there and never again, so
// they are package-level and immutable rather than fields on World.
//
// Read them as music: pieces 0..3 are the four refrains, 4 is the chorus (which is twice
// as long as anything else), and 5 and 6 are the two sparse refrains. So musicScore is
// four refrains, the chorus twice, the four refrains again, the chorus twice, the two
// sparse refrains, and the chorus twice more -- about a minute and a half, which is the
// splash screen's loop.
//
// gameScore is the in-game loop and is only five entries long with a -1 in the middle,
// which is a jump instruction rather than a piece; NextMusicPiece has the note on what it
// does and on the loop it produces.
var (
	musicScore = [16]int16{
		0, 1, 2, 3, PlayChorus, PlayChorus, 0, 1, 2, 3,
		PlayChorus, PlayChorus, PlayRefrainSparse1, PlayRefrainSparse2, PlayChorus, PlayChorus,
	}
	gameScore = [6]int16{
		PlayRefrainSparse2, PlayRefrainSparse1, -1, PlayRefrainSparse2, PlayChorus, PlayChorus,
	}
)

// The three piece numbers the score tables name (GliderDefines.h:51-53). They are indices
// into the seven pieces of music, and the extracted resources confirm them: 'snd ' 2004 is
// "Chorus.22", 2005 is "RefrainSparse1.22" and 2006 is "RefrainSparse2.22".
const (
	PlayChorus         int16 = 4
	PlayRefrainSparse1 int16 = 5
	PlayRefrainSparse2 int16 = 6
)

// The two lengths the score walks wrap at (Music.c:17-18). They are the *table* lengths
// and the walk does not use them symmetrically -- see NextMusicPiece.
const (
	lastMusicPiece = 16
	lastGamePiece  = 6
)

// InitMusic is the tail of Music.c's InitMusic (:356-368): the score is loaded, so set the
// cursor to the top, take the first piece and start the splash score if the preference
// wants it.
//
// The port calls this from the host once a bank has loaded, which is also where
// DontLoadMusic is cleared -- the flag that has kept every function in this file inert
// since Stage 1.2. Everything above the cursor assignment in the C is asset loading and
// channel opening, which is internal/audio's.
func (w *World) InitMusic() {
	w.DontLoadMusic = false
	w.MusicOn = false
	w.MusicCursor = 0
	w.MusicSoundID = musicScore[w.MusicCursor]
	w.MusicMode = PlayWholeScoreMode

	if w.PlayMusicIdle {
		w.StartMusic()
	}
}

// StartMusic is Music.c:44-94: queue two pieces and arm the callback.
//
// Two, not one, and the second one advances the cursor. That is the original's way of
// keeping a piece in reserve so the score does not gap between pieces while the completion
// callback runs, and the port keeps it because the cursor arithmetic is visible in the
// result: the first piece played is the one the cursor is on now, and the callback's first
// question is answered from one step further along.
//
// Note the wrap here is at kLastMusicPiece, all sixteen entries, where the callback's is at
// kLastMusicPiece-1. So musicScore[15] can only ever be reached by this function. It is a
// kPlayChorus, as is [14], so the audible consequence is nil -- but the asymmetry is real
// and transcribed rather than tidied, because tidying it would change which piece follows
// which for a listener who knows the tune.
func (w *World) StartMusic() {
	if w.DontLoadMusic || w.Music == nil {
		return
	}
	if !w.Music.MusicAvailable() {
		return
	}

	w.Music.QueueMusic(w.MusicSoundID)

	w.MusicCursor++
	if w.MusicCursor >= lastMusicPiece {
		w.MusicCursor = 0
	}
	w.MusicSoundID = musicScore[w.MusicCursor]

	w.Music.QueueMusic(w.MusicSoundID)
	w.MusicOn = true
}

// StopTheMusic is Music.c:98-121.
//
// The `isMusicOn` guard means a second stop is free, which matters because both ladders
// above and ToggleMusicWhilePlaying can all reach it in the same frame.
func (w *World) StopTheMusic() {
	if w.DontLoadMusic || w.Music == nil {
		return
	}
	if w.MusicOn {
		w.Music.SilenceMusic()
		w.MusicOn = false
	}
}

// ToggleMusicWhilePlaying is Music.c:125-142: mute on deactivation, unmute on return. It
// takes no argument and serves both directions because it reads isPlayMusicGame and
// isMusicOn rather than being told which way to go -- start if the preference is on and
// nothing is playing, stop if the preference is off and something is.
//
// **Four call sites, two of them a stereo.** Play.c:413 and :421 are the suspend/resume
// arms, and Dynamics.c:681 and :698 are HandleStereo's -- once when a stereo finishes
// switching on and once when it finishes switching off. The name says "while playing" and
// means "while a game is in progress", not "while music is playing".
//
// The stereo pair is the reason it is a *toggle* rather than a setter, and the reason two
// stereos in one room wired to one switch cancel out; HandleStereo has the full note.
func (w *World) ToggleMusicWhilePlaying() {
	if w.DontLoadMusic {
		return
	}
	if w.PlayMusicGame {
		if !w.MusicOn {
			w.StartMusic()
		}
	} else if w.MusicOn {
		w.StopTheMusic()
	}
}

// NextMusicPiece is MusicCallBack (Music.c:170-216) with the two SndDoCommands removed: walk
// the score by one and answer with the piece to play.
//
// The mixer calls it, synchronously, from the goroutine that owns this World -- see
// internal/audio/music.go. It is the only function in the game whose caller is the audio
// path, and everything it touches is these two fields.
//
// Three modes, and the third is the trick:
//
//	kPlayGameScoreMode   walk gameScore, wrapping to 1 rather than 0
//	kPlayWholeScoreMode  walk musicScore, wrapping at 15 rather than 16
//	anything else        the mode *is* the piece number, so the score stops walking and
//	                     one piece repeats. That is what SetMusicalMode's default branch
//	                     is for and why musicMode is a short rather than an enum.
//
// The negative entry in gameScore is a jump: gameScore[2] is -1, and `musicCursor +=
// musicSoundID` moves the cursor *back* by one before re-reading. Follow it through and the
// in-game score is not a loop over five pieces at all -- from the kick at cursor 2 it plays
// [3] sparse2, [4] chorus, [5] chorus, wraps to [1] sparse1, then reads [2], jumps back to
// [1], and plays sparse1 for the rest of the game. So the game music is a flourish and then
// one quiet refrain on repeat, which is exactly what background music under a game should
// be, and it is achieved with a two-character table entry.
func (w *World) NextMusicPiece() int16 {
	switch w.MusicMode {
	case PlayGameScoreMode:
		w.MusicCursor++
		if w.MusicCursor >= lastGamePiece {
			w.MusicCursor = 1
		}
		w.MusicSoundID = w.gamePiece(w.MusicCursor)
		if w.MusicSoundID < 0 {
			w.MusicCursor += w.MusicSoundID
			w.MusicSoundID = w.gamePiece(w.MusicCursor)
		}

	case PlayWholeScoreMode:
		w.MusicCursor++
		if w.MusicCursor >= lastMusicPiece-1 {
			w.MusicCursor = 0
		}
		w.MusicSoundID = w.musicPiece(w.MusicCursor)

	default:
		w.MusicSoundID = w.MusicMode
	}
	return w.MusicSoundID
}

// gamePiece and musicPiece are the two table reads, range-checked.
//
// The C indexes both tables unchecked, and in the C that is safe: the cursor is only ever
// written by SetMusicalMode and the walk above, and every one of those writes lands in
// range. It is checked here anyway, because a saved game or a replay script will be able to
// set the cursor in 1.10, and a corrupt cursor should cost the player a piece of music
// rather than crash the game. An out-of-range read answers with the chorus, which is the
// piece the score spends the most time on.
func (w *World) gamePiece(i int16) int16 {
	if i < 0 || int(i) >= len(gameScore) {
		return PlayChorus
	}
	return gameScore[i]
}

func (w *World) musicPiece(i int16) int16 {
	if i < 0 || int(i) >= len(musicScore) {
		return PlayChorus
	}
	return musicScore[i]
}
