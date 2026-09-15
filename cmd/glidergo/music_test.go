package main

import (
	"testing"

	"glidergo/internal/game"
	"glidergo/internal/prefs"
)

// scoreLog is a music channel that records what it was asked for.
//
// It stands in for internal/audio.Engine, which is the real thing music.go talks to and
// which needs a sample bank, a sink and a mixer goroutine's worth of setup to say anything
// about a score. The three methods are the whole of game.MusicChannel.
type scoreLog struct {
	queued   []int16
	silenced int
	mute     bool // MusicAvailable false, which is what volume 0 looks like from here
}

func (l *scoreLog) MusicAvailable() bool   { return !l.mute }
func (l *scoreLog) QueueMusic(piece int16) { l.queued = append(l.queued, piece) }
func (l *scoreLog) SilenceMusic()          { l.silenced++ }

// titleApp is an app with nothing in it but the two things music.go reads: the preferences
// and a mixer. eng stays nil, so this covers everything except the one line that re-points
// Engine.NextPiece -- which is why startTitleMusic is not called here and the ladders are
// driven directly. See TestTitleMusicNeedsNoMixer for the nil-engine path.
func titleApp(t *testing.T, on bool, ch game.MusicChannel) (*app, *game.World) {
	t.Helper()
	p := prefs.Default()
	p.MusicOnTitle = on
	a := &app{p: p}
	a.title = &game.World{Music: ch}
	a.title.InitMusic()
	return a, a.title
}

// idle is startTitleMusic without the engine assignment: the two lines that decide
// anything. Keeping them together here is what makes the tests below read as the policy
// rather than as plumbing.
func idle(a *app) {
	a.title.PlayMusicIdle = a.p.MusicOnTitle
	a.title.StartIdleMusic()
}

func TestTitleScoreStartsAtTheTopAndCommitsTwoPieces(t *testing.T) {
	// InitMusic must arm the walk and commit nothing -- the preference is read when the
	// score starts, not when the World is built -- and then the first start queues the
	// two pieces StartMusic always commits, from the top of the score.
	log := &scoreLog{}
	a, w := titleApp(t, true, log)
	if len(log.queued) != 0 {
		t.Fatalf("building the title score queued %v: the preference is not read yet", log.queued)
	}
	if w.MusicOn {
		t.Error("the title score reports music on before anything started it")
	}

	idle(a)
	if !w.MusicOn {
		t.Fatal("music_on_title is on and StartIdleMusic did not start the score")
	}
	if len(log.queued) != 2 {
		t.Fatalf("started the score with %d pieces queued, want 2: StartMusic commits a "+
			"reserve so the score does not gap between pieces", len(log.queued))
	}
	// The whole score from the top, which is what idle music means -- as opposed to the
	// six-piece loop a game plays.
	if w.MusicMode != game.PlayWholeScoreMode {
		t.Errorf("idle music is in mode %d, want the whole score (%d)",
			w.MusicMode, game.PlayWholeScoreMode)
	}
}

func TestTitleScoreStartIsIdempotent(t *testing.T) {
	// The settings screen calls this on every keystroke, so a second start must not
	// commit a third piece.
	log := &scoreLog{}
	a, _ := titleApp(t, true, log)
	idle(a)
	idle(a)
	idle(a)
	if len(log.queued) != 2 {
		t.Errorf("three starts queued %d pieces, want 2: StartIdleMusic must only set the "+
			"mode once the score is already playing", len(log.queued))
	}
}

func TestTitleScoreStopsWhenThePreferenceGoesOff(t *testing.T) {
	log := &scoreLog{}
	a, w := titleApp(t, true, log)
	idle(a)

	a.p.MusicOnTitle = false
	idle(a)
	if w.MusicOn {
		t.Error("music_on_title went off and the score kept playing")
	}
	if log.silenced != 1 {
		t.Errorf("silenced %d times, want 1", log.silenced)
	}

	// And back on: the row is a toggle, and the score has to come back from it.
	a.p.MusicOnTitle = true
	idle(a)
	if !w.MusicOn {
		t.Error("music_on_title came back on and the score did not")
	}
	if len(log.queued) != 4 {
		t.Errorf("queued %d pieces over off-and-on, want 4 (two per start)", len(log.queued))
	}
}

func TestTitleScoreStaysRecoverableWhileMuted(t *testing.T) {
	// Volume 0 makes Engine.MusicAvailable false, and the port must treat that as "not
	// yet" rather than "no": ApplyPrefs pushes the volume and then asks for the score, so
	// raising the volume on the settings screen has to start it.
	log := &scoreLog{mute: true}
	a, w := titleApp(t, true, log)
	idle(a)
	if w.MusicOn || len(log.queued) != 0 {
		t.Fatalf("a muted mixer was asked for %v and the score thinks it is on: %v",
			log.queued, w.MusicOn)
	}

	log.mute = false
	idle(a)
	if !w.MusicOn || len(log.queued) != 2 {
		t.Errorf("the volume came back up and the score did not start: on=%v queued=%v",
			w.MusicOn, log.queued)
	}
}

func TestAdoptScoreKeepsTheScorePlayingAcrossAGame(t *testing.T) {
	// A game ends with NewGame's teardown calling its own StartIdleMusic, so on the Mac --
	// one global cursor -- the title screen is already playing when control comes back.
	// Copying is how two Worlds get there, and MusicOn is the field that earns it: it is
	// what tells the next StartIdleMusic there is nothing to start, so the reserve plays on
	// and the player hears no gap.
	log := &scoreLog{}
	a, w := titleApp(t, true, log)
	idle(a)
	before := len(log.queued)

	// A finished game, as NewGame's teardown leaves it: its own cursor, somewhere in the
	// middle of the score, and its music still on.
	g := &game.World{Music: log}
	g.InitMusic()
	g.MusicCursor, g.MusicSoundID, g.MusicMode, g.MusicOn = 9, 4, game.PlayWholeScoreMode, true
	a.adoptScore(g)

	if w.MusicCursor != 9 || w.MusicSoundID != 4 {
		t.Errorf("the title score sits at cursor %d piece %d, want the game's 9 and 4",
			w.MusicCursor, w.MusicSoundID)
	}
	idle(a)
	if len(log.queued) != before {
		t.Errorf("coming back from a game queued %d more pieces: the score was already "+
			"playing and should have been left alone", len(log.queued)-before)
	}

	// And then the cursor goes back to the top, because StartIdleMusic ends in
	// SetMusicalMode, whose default arm resets it -- the C does this too, which is why the
	// adopted cursor is not the point. Pinned because it looks like a bug in adoptScore
	// and is not: the two committed pieces are what covers the rewind.
	if w.MusicCursor != 0 {
		t.Errorf("the title score resumed at cursor %d; SetMusicalMode resets it to 0",
			w.MusicCursor)
	}
	if got := w.NextMusicPiece(); w.MusicCursor != 1 {
		t.Errorf("the next piece came from cursor %d (piece %d), want 1", w.MusicCursor, got)
	}
}

func TestAdoptScoreWithoutATitleScoreDoesNothing(t *testing.T) {
	// -shot and any run with no mixer never build one, and a game still ends.
	a := &app{p: prefs.Default()}
	g := &game.World{}
	g.MusicCursor = 3
	a.adoptScore(g)
	a.stopTitleMusic()
}

func TestTitleMusicNeedsNoMixer(t *testing.T) {
	// The nil-engine path: a session with -sound off, or a machine with no sink, must
	// reach every one of these and build no World at all.
	a := &app{p: prefs.Default()}
	if w := a.titleScore(); w != nil {
		t.Fatal("a session with no mixer built a World to walk a score with")
	}
	a.startTitleMusic()
	a.stopTitleMusic()
	if a.title != nil {
		t.Error("startTitleMusic built a title score with no mixer to play it")
	}
}

func TestTitleMusicIsOnByDefault(t *testing.T) {
	// The original's default (Main.c:149-150), and the reason the row exists at all.
	if !prefs.Default().MusicOnTitle {
		t.Error("music_on_title defaults off; the original plays the score while idle")
	}
}
