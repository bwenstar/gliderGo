package audio

// Tests for the music channel: the two-deep queue StartMusic fills, the chain that keeps it
// full, and the independence from the three effect channels that is the whole reason music is a
// fourth channel and not a fourth priority.

import "testing"

func TestMusicQueueAndChain(t *testing.T) {
	const frames = 4
	e := New(fakeBank(frames))

	// The two bufferCmds StartMusic issues (Music.c:59-83), with the score cursor advanced
	// between them.
	e.QueueMusic(0)
	e.QueueMusic(1)
	if got := e.MusicQueued(); got != 2 {
		t.Fatalf("queued %d pieces, want 2", got)
	}

	// The score, from the callback's point of view: whatever the game's cursor walk hands
	// back. Here, ascending pieces.
	next := int16(2)
	e.NextPiece = func() int16 {
		p := next
		next++
		if next >= MaxMusic {
			next = 0
		}
		return p
	}

	// Six pieces of four frames each: the two queued, then four from the chain.
	want := []int16{0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5}
	for i, w := range want {
		if got := e.MusicPlaying(); i%frames == 0 && got != w && got != NoPiece {
			// Checked properly by the sample value below; this only catches a piece
			// starting on the wrong sample.
			t.Fatalf("sample %d: MusicPlaying = %d, want %d", i, got, w)
		}
		// fakeBank's music sits below zero, one step per piece, so the sample names the
		// piece: piece p mixes to -(p+1)<<8.
		if got, wantSample := mix1(e), int16(-(int32(w)+1)<<8); got != wantSample {
			t.Fatalf("sample %d: mixed %d, want %d (piece %d)", i, got, wantSample, w)
		}
	}
	if e.stats.MusicStarted != 6 {
		t.Errorf("MusicStarted = %d, want 6", e.stats.MusicStarted)
	}
	if got := e.MusicQueued(); got != 0 {
		t.Errorf("%d pieces still queued; the chain should hold nothing in reserve", got)
	}
}

// TestMusicStallsOnNoPiece checks the latch. A chain whose NextPiece declines must stop asking:
// without the latch it would call into game code 22,255 times a second while the score is
// stopped.
func TestMusicStallsOnNoPiece(t *testing.T) {
	e := New(fakeBank(2))

	calls := 0
	e.NextPiece = func() int16 { calls++; return NoPiece }

	e.QueueMusic(0)
	mixN(e, 100)

	if calls != 1 {
		t.Errorf("NextPiece was called %d times after declining once; it must be asked once and then left alone", calls)
	}
	if e.MusicPlaying() != NoPiece {
		t.Errorf("MusicPlaying = %d after the score declined, want %d", e.MusicPlaying(), NoPiece)
	}

	// Queueing re-arms it: this is StartMusic being called again after the score stopped.
	e.NextPiece = func() int16 { calls++; return 3 }
	e.QueueMusic(2)
	mixN(e, 3)
	if calls != 2 {
		t.Errorf("NextPiece was called %d times, want 2: QueueMusic must clear the stall", calls)
	}
	if got := e.MusicPlaying(); got != 3 {
		t.Errorf("MusicPlaying = %d, want piece 3 from the re-armed chain", got)
	}
}

// TestMusicWithoutAChain is a build with no game behind the engine: the queue plays out and the
// channel goes quiet rather than looping or panicking.
func TestMusicWithoutAChain(t *testing.T) {
	e := New(fakeBank(2))
	e.QueueMusic(0)
	mixN(e, 2)
	if e.MusicPlaying() != NoPiece {
		t.Errorf("MusicPlaying = %d with no NextPiece, want silence", e.MusicPlaying())
	}
	if got := mix1(e); got != 0 {
		t.Errorf("mixed %d with the score exhausted, want silence", got)
	}
}

// TestSilenceMusic is StopTheMusic: what is queued is dropped and what is playing stops.
func TestSilenceMusic(t *testing.T) {
	e := New(fakeBank(100))
	e.NextPiece = func() int16 { return 0 }
	e.QueueMusic(0)
	e.QueueMusic(1)
	mixN(e, 10)

	e.SilenceMusic()
	if got := e.MusicPlaying(); got != NoPiece {
		t.Errorf("MusicPlaying = %d after SilenceMusic", got)
	}
	if got := e.MusicQueued(); got != 0 {
		t.Errorf("%d pieces still queued after SilenceMusic", got)
	}
	if got := mix1(e); got != 0 {
		t.Errorf("mixed %d after SilenceMusic, want silence", got)
	}
	// The chain is disarmed too, so a stopped score stays stopped even though NextPiece
	// would happily supply another piece.
	mixN(e, 10)
	if e.MusicPlaying() != NoPiece {
		t.Error("the score restarted itself after SilenceMusic")
	}
}

// TestMusicIsIndependentOfEffects is the acceptance criterion for the music half of this stage:
// music must start, stop and survive a room change independently of the three effect channels.
//
// The room change is the interesting one. DrawLocale calls FlushAnyTriggerPlaying and
// DumpTriggerSound on every transition, and if the music channel were reachable from either the
// score would restart -- or stop -- every time the player walked through a door.
func TestMusicIsIndependentOfEffects(t *testing.T) {
	e := New(fakeBank(1000))
	e.trigger = e.bank.Effect(7)
	e.NextPiece = func() int16 { return 0 }
	e.QueueMusic(0)

	// Fill all three effect channels, including one with a house's trigger sound.
	e.PlayPrioritySound(TriggerSlot, TriggerPriority)
	e.PlayPrioritySound(1, 300)
	e.PlayPrioritySound(2, 300)
	mixN(e, 10)

	before := e.MusicPlaying()
	frame := e.music.pos >> 16 // pos is 16.16 fixed point; see Sound.Step

	// A room change.
	e.FlushTriggerSound()

	if got := e.MusicPlaying(); got != before {
		t.Errorf("the music changed from piece %d to %d across a room change", before, got)
	}
	if got := e.music.pos >> 16; got != frame {
		t.Errorf("the music jumped from frame %d to %d across a room change", frame, got)
	}

	// And the reverse: silencing the music leaves the effects alone.
	e.SilenceMusic()
	if got := e.Playing(); got[1] != 1 || got[2] != 2 {
		t.Errorf("effects = %v after SilenceMusic, want slots 1 and 2 still playing", got)
	}

	// Music also has no priority and takes no channel: a fourth sound is still refused by
	// the three-channel policy and not given the music channel.
	e.QueueMusic(0)
	mixN(e, 1)
	e.PlayPrioritySound(3, 100)
	if e.MusicPlaying() == 3 {
		t.Error("an effect was played on the music channel")
	}
}

// TestMusicAvailable covers StartMusic's two refusals: no sound system, and a system volume of
// zero. The game reads the answer into isMusicOn, so getting it wrong makes every later
// ToggleMusicWhilePlaying act on a lie.
func TestMusicAvailable(t *testing.T) {
	e := New(fakeBank(4))
	if !e.MusicAvailable() {
		t.Error("music is unavailable with a loaded bank at full volume")
	}
	e.SetVolume(0)
	if e.MusicAvailable() {
		t.Error("music is available at volume 0; StartMusic refuses to begin the score on a muted machine")
	}
	e.SetVolume(1)
	if !e.MusicAvailable() {
		t.Error("music is unavailable at volume 1")
	}

	// A bank with effects but no music -- what a Glider PRO Lite fork extracts to.
	lite := fakeBank(4)
	for i := range lite.Music {
		lite.Music[i] = nil
	}
	if New(lite).MusicAvailable() {
		t.Error("music is available with no music in the bank")
	}
}

// TestMusicPieceConstants pins the three piece numbers that double as modes against the
// resource names in the extracted bank, which is a genuinely independent source: the C header
// says kPlayChorus is 4, and the fork says 'snd ' 2004 is called "Chorus.22".
func TestMusicPieceConstants(t *testing.T) {
	b := requireBank(t)
	for _, c := range []struct {
		piece  int16
		prefix string
	}{
		{PlayChorus, "Chorus"},
		{PlayRefrainSparse1, "RefrainSparse1"},
		{PlayRefrainSparse2, "RefrainSparse2"},
	} {
		s := b.Piece(c.piece)
		if s == nil {
			t.Errorf("piece %d is missing", c.piece)
			continue
		}
		if len(s.Name) < len(c.prefix) || s.Name[:len(c.prefix)] != c.prefix {
			t.Errorf("piece %d is the resource %q, but the header calls it %s",
				c.piece, s.Name, c.prefix)
		}
	}
}
