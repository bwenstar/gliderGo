package game

// SetMusicalMode (Music.c:146-166), which is the only part of Music.c the transit
// path touches.
//
// The rest of Music.c -- the two sound channels, the score tables, the callback that
// walks them and StartTheMusic/StopTheMusic -- is 1.6's. This one function is here
// because all six room-change handlers open with a call to it, and leaving it out
// would leave six calls to comment out and put back.
//
// What it does is *not* what its name suggests, and the asymmetry is the reason it is
// transcribed rather than stubbed:
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
// Calling this with kKickGameScoreMode when the music is stopped therefore does
// nothing observable at all, which is what makes it safe to have live now.

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
// off, the mode and the cursor must stay where they were, because 1.6's music thread
// will read them and a stale cursor from a game played before the player turned music
// on is what the original leaves there too.
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

// StartMusic, StopTheMusic and ToggleMusicWhilePlaying are Music.c's channel
// management and belong to 1.6.
//
// Named no-ops, and the important thing about them is what they do *not* do: neither
// StartMusic nor StopTheMusic touches MusicOn here. That is what keeps both ladders
// above inert -- MusicOn stays false, so `!w.MusicOn` is always true and `else if
// w.MusicOn` never fires -- and it is a truer stub than one that flipped the flag,
// because a flag flipped without a channel behind it would make 1.6 debug a lie.
func (w *World) StartMusic()   {}
func (w *World) StopTheMusic() {}

// ToggleMusicWhilePlaying is Music.c: mute on deactivation, unmute on return. Called
// from both arms of the suspend/resume handler, which is why one function serves both
// directions -- it reads switchedOut rather than taking an argument.
func (w *World) ToggleMusicWhilePlaying() {}
