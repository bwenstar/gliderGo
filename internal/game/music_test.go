package game

// Tests for the score walk, which is the part of Music.c that is hard to read and easy to
// transcribe wrongly: two tables, three modes, two different wrap points and a table entry that
// is a jump instruction rather than a piece of music.
//
// No assets and no mixer -- the channel is a recorder. What is under test is the arithmetic.

import "testing"

// recorder is a MusicChannel that remembers what it was asked to play.
type recorder struct {
	queued    []int16
	silenced  int
	available bool
}

func (r *recorder) MusicAvailable() bool { return r.available }
func (r *recorder) QueueMusic(p int16)   { r.queued = append(r.queued, p) }
func (r *recorder) SilenceMusic()        { r.silenced++ }
func newMusicWorld() (*World, *recorder) {
	r := &recorder{available: true}
	w := &World{Music: r}
	w.InitMusic()
	return w, r
}

// TestInitMusic is the tail of the C's InitMusic: the cursor at the top of the whole score and
// the mode set to walk it.
func TestInitMusic(t *testing.T) {
	w, r := newMusicWorld()
	if w.DontLoadMusic {
		t.Error("InitMusic left DontLoadMusic set")
	}
	if w.MusicCursor != 0 || w.MusicSoundID != musicScore[0] || w.MusicMode != PlayWholeScoreMode {
		t.Errorf("cursor=%d piece=%d mode=%d, want 0/%d/%d",
			w.MusicCursor, w.MusicSoundID, w.MusicMode, musicScore[0], PlayWholeScoreMode)
	}
	if w.MusicOn {
		t.Error("MusicOn is set with isPlayMusicIdle false")
	}
	if len(r.queued) != 0 {
		t.Errorf("InitMusic queued %v with isPlayMusicIdle false", r.queued)
	}

	// With the idle preference on, InitMusic starts the splash score.
	w2 := &World{Music: &recorder{available: true}, PlayMusicIdle: true}
	w2.InitMusic()
	if !w2.MusicOn {
		t.Error("InitMusic did not start the score with isPlayMusicIdle set")
	}
}

// TestStartMusicQueuesTwo pins the two bufferCmds and the cursor advance between them.
//
// One queued piece would be the obvious transcription and it would be wrong: the second is the
// reserve that keeps the score seamless while the completion callback runs, and the cursor
// advance in the middle is why the callback's first answer is the *third* piece and not the
// second.
func TestStartMusicQueuesTwo(t *testing.T) {
	w, r := newMusicWorld()
	w.StartMusic()

	if want := []int16{musicScore[0], musicScore[1]}; len(r.queued) != 2 || r.queued[0] != want[0] || r.queued[1] != want[1] {
		t.Errorf("queued %v, want %v", r.queued, want)
	}
	if w.MusicCursor != 1 {
		t.Errorf("cursor = %d after StartMusic, want 1", w.MusicCursor)
	}
	if !w.MusicOn {
		t.Error("MusicOn is not set after StartMusic")
	}
	if got := w.NextMusicPiece(); got != musicScore[2] {
		t.Errorf("the first callback answered piece %d, want musicScore[2] = %d", got, musicScore[2])
	}
}

// TestStartMusicRefused covers the three ways StartMusic declines. The important one is the last:
// MusicOn must stay false, because both ladders in this file and ToggleMusicWhilePlaying all
// branch on it.
func TestStartMusicRefused(t *testing.T) {
	for _, c := range []struct {
		name  string
		setup func() (*World, *recorder)
	}{
		{"no music system", func() (*World, *recorder) {
			return &World{DontLoadMusic: true}, nil
		}},
		{"no channel", func() (*World, *recorder) {
			w := &World{}
			w.DontLoadMusic = false
			return w, nil
		}},
		{"machine muted", func() (*World, *recorder) {
			r := &recorder{available: false}
			w := &World{Music: r}
			w.InitMusic()
			return w, r
		}},
	} {
		w, r := c.setup()
		w.StartMusic()
		if w.MusicOn {
			t.Errorf("%s: MusicOn is set", c.name)
		}
		if r != nil && len(r.queued) != 0 {
			t.Errorf("%s: queued %v", c.name, r.queued)
		}
	}
}

// TestGameScoreSettlesOnOneRefrain follows the -1 in gameScore, which is the most surprising
// four bytes in Music.c.
//
// From the kick at cursor 2 the score plays sparse2, chorus, chorus, wraps to sparse1, then
// reads the -1, jumps the cursor back one and plays sparse1 again -- and keeps playing sparse1
// for the rest of the game. The in-game music is a three-piece flourish and then one quiet
// refrain on repeat, which is what background music under a game should be.
func TestGameScoreSettlesOnOneRefrain(t *testing.T) {
	w, _ := newMusicWorld()
	w.MusicMode = PlayGameScoreMode
	w.SetMusicalMode(KickGameScoreMode) // cursor = 2

	want := []int16{
		PlayRefrainSparse2, // [3]
		PlayChorus,         // [4]
		PlayChorus,         // [5]
		PlayRefrainSparse1, // wrapped to [1]
		PlayRefrainSparse1, // [2] is -1: back to [1]
		PlayRefrainSparse1,
		PlayRefrainSparse1,
		PlayRefrainSparse1,
	}
	for i, w2 := range want {
		if got := w.NextMusicPiece(); got != w2 {
			t.Fatalf("piece %d = %d, want %d (cursor %d)", i, got, w2, w.MusicCursor)
		}
	}
	if w.MusicCursor != 1 {
		t.Errorf("the score settled on cursor %d, want 1", w.MusicCursor)
	}
}

// TestProdGameScoreMode is the other entry into the game score: cursor -1, so the walk's
// increment lands on gameScore[0] and the flourish is skipped.
func TestProdGameScoreMode(t *testing.T) {
	w, _ := newMusicWorld()
	w.MusicMode = PlayGameScoreMode
	w.SetMusicalMode(ProdGameScoreMode)
	if w.MusicCursor != -1 {
		t.Fatalf("cursor = %d after a prod, want -1", w.MusicCursor)
	}
	if got := w.NextMusicPiece(); got != gameScore[0] {
		t.Errorf("first piece after a prod = %d, want gameScore[0] = %d", got, gameScore[0])
	}
	// And it settles in the same place, two pieces later.
	w.NextMusicPiece()
	if got := w.NextMusicPiece(); got != PlayRefrainSparse1 {
		t.Errorf("third piece = %d, want the sparse refrain it settles on", got)
	}
}

// TestWholeScoreWrapsEarly pins the asymmetry between StartMusic's wrap and the callback's.
//
// The callback wraps at kLastMusicPiece-1, so musicScore[15] is unreachable from it; StartMusic
// wraps at kLastMusicPiece, so it can reach [15] and nothing else can. Both entries are the
// chorus, so no listener could tell -- but the port transcribes the asymmetry rather than
// tidying it, and this test is what stops somebody tidying it later.
func TestWholeScoreWrapsEarly(t *testing.T) {
	w, _ := newMusicWorld()
	w.MusicMode = PlayWholeScoreMode
	w.MusicCursor = 13

	if got := w.NextMusicPiece(); got != musicScore[14] || w.MusicCursor != 14 {
		t.Fatalf("piece = %d cursor = %d, want %d/14", got, w.MusicCursor, musicScore[14])
	}
	if got := w.NextMusicPiece(); got != musicScore[0] || w.MusicCursor != 0 {
		t.Errorf("after [14] the callback gave piece %d at cursor %d, want musicScore[0] = %d at 0; "+
			"the callback wraps at 15 and never reaches [15]", got, w.MusicCursor, musicScore[0])
	}

	// StartMusic, by contrast, walks all sixteen.
	w.MusicCursor = 14
	w.MusicSoundID = musicScore[14]
	w.StartMusic()
	if w.MusicCursor != 15 {
		t.Errorf("StartMusic left the cursor at %d, want 15: its wrap is at 16", w.MusicCursor)
	}
}

// TestModeIsAPieceNumber is MusicCallBack's default branch: a mode that is not one of the two
// score modes is a piece index, so the score stops walking and one piece repeats.
func TestModeIsAPieceNumber(t *testing.T) {
	w, _ := newMusicWorld()
	w.SetMusicalMode(PlayChorus)
	if w.MusicMode != PlayChorus || w.MusicCursor != 0 {
		t.Fatalf("mode = %d cursor = %d, want %d/0", w.MusicMode, w.MusicCursor, PlayChorus)
	}
	for i := 0; i < 3; i++ {
		if got := w.NextMusicPiece(); got != PlayChorus {
			t.Fatalf("piece %d = %d, want the chorus on repeat", i, got)
		}
	}
	if w.MusicCursor != 0 {
		t.Errorf("cursor moved to %d; a fixed mode does not walk the score", w.MusicCursor)
	}
}

// TestCursorOutOfRange is the port's own guard, not the C's: a cursor from a corrupt saved game
// or a hand-written replay script costs the player a piece of music rather than crashing.
func TestCursorOutOfRange(t *testing.T) {
	w, _ := newMusicWorld()
	w.MusicMode = PlayWholeScoreMode
	w.MusicCursor = 900
	if got := w.NextMusicPiece(); got != musicScore[0] {
		// 900 increments to 901, which is >= 15, so the wrap catches it first.
		t.Errorf("piece = %d, want the wrap to bring it back to musicScore[0] = %d", got, musicScore[0])
	}

	// The direct read, past the wrap: only reachable through the game score's jump.
	w.MusicMode = PlayGameScoreMode
	w.MusicCursor = -400
	if got := w.NextMusicPiece(); got != PlayChorus {
		t.Errorf("piece = %d from a cursor of -400, want the chorus", got)
	}
}

// TestStopTheMusic covers the isMusicOn guard: a second stop is free, which matters because the
// two ladders and the stereo toggle can all reach it in one frame.
func TestStopTheMusic(t *testing.T) {
	w, r := newMusicWorld()
	w.StartMusic()
	w.StopTheMusic()
	if w.MusicOn {
		t.Error("MusicOn is still set after StopTheMusic")
	}
	if r.silenced != 1 {
		t.Errorf("silenced %d times, want 1", r.silenced)
	}
	w.StopTheMusic()
	if r.silenced != 1 {
		t.Errorf("a second stop reached the channel: silenced %d times", r.silenced)
	}
}

// TestToggleMusicWhilePlaying is the stereo's toggle and the suspend/resume pair, which are the
// same four lines seen from two places.
func TestToggleMusicWhilePlaying(t *testing.T) {
	w, r := newMusicWorld()

	// A stereo switched on with the game-music preference on starts the score.
	w.PlayMusicGame = true
	w.ToggleMusicWhilePlaying()
	if !w.MusicOn {
		t.Fatal("the score did not start")
	}
	queued := len(r.queued)

	// Again, with the score already playing: nothing happens, which is what makes two
	// stereos in one room wired to one switch cancel out rather than stacking.
	w.ToggleMusicWhilePlaying()
	if len(r.queued) != queued {
		t.Errorf("a second toggle queued %d more pieces", len(r.queued)-queued)
	}

	// The preference off stops it, and again is free.
	w.PlayMusicGame = false
	w.ToggleMusicWhilePlaying()
	if w.MusicOn {
		t.Error("the score did not stop")
	}
	w.ToggleMusicWhilePlaying()
	if r.silenced != 1 {
		t.Errorf("silenced %d times, want 1", r.silenced)
	}
}

// TestDontLoadMusicIsInert is the state a build with no audio runs in, and every function in
// music.go has to tolerate it -- the game must be playable with no sound system at all.
func TestDontLoadMusicIsInert(t *testing.T) {
	r := &recorder{available: true}
	w := &World{Music: r, DontLoadMusic: true, PlayMusicGame: true, PlayMusicIdle: true}

	w.SetMusicalMode(PlayGameScoreMode)
	w.StartMusic()
	w.StopTheMusic()
	w.ToggleMusicWhilePlaying()
	w.StartGameMusic()
	w.StartIdleMusic()

	if w.MusicOn || w.MusicMode != 0 || w.MusicCursor != 0 {
		t.Errorf("state moved with music off: on=%v mode=%d cursor=%d", w.MusicOn, w.MusicMode, w.MusicCursor)
	}
	if len(r.queued) != 0 || r.silenced != 0 {
		t.Errorf("the channel was touched with music off: queued %v silenced %d", r.queued, r.silenced)
	}
}

// TestNilChannel is the same tolerance one level down: DontLoadMusic clear but no mixer, which
// is what a headless fidelity run with sound off looks like.
func TestNilChannel(t *testing.T) {
	w := &World{PlayMusicGame: true}
	w.InitMusic()
	w.StartMusic()
	w.StopTheMusic()
	w.ToggleMusicWhilePlaying()
	w.StartGameMusic()
	if w.MusicOn {
		t.Error("MusicOn is set with no channel behind it")
	}
	// The score still walks: NextMusicPiece is pure state, and with no mixer nothing calls
	// it. StartGameMusic left the mode on the game score, so this is gameScore[1].
	if got := w.NextMusicPiece(); got != gameScore[1] {
		t.Errorf("piece = %d, want gameScore[1] = %d", got, gameScore[1])
	}
}

// TestScoreTables pins the two tables against Music.c:332-354. They are read by nothing else and
// are the easiest thing in the file to fat-finger.
func TestScoreTables(t *testing.T) {
	want := [16]int16{0, 1, 2, 3, 4, 4, 0, 1, 2, 3, 4, 4, 5, 6, 4, 4}
	if musicScore != want {
		t.Errorf("musicScore = %v, want %v", musicScore, want)
	}
	wantGame := [6]int16{6, 5, -1, 6, 4, 4}
	if gameScore != wantGame {
		t.Errorf("gameScore = %v, want %v", gameScore, wantGame)
	}
}
