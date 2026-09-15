package audio

// Tests for the channel policy and the mixer.
//
// None of these touch the extracted assets: the bank is fabricated in memory, with samples
// whose values are chosen so that a mix can be read by eye. That is on purpose -- the policy in
// Sound.c is the part of the audio path most likely to be broken by a well-meant tidy-up, and a
// test that only runs on a machine that has run `make assets` is a test that does not run on a
// bare checkout.

import "testing"

// fakeBank builds a bank of flat samples: slot n plays the constant byte 128+n, so the mixer's
// output for slot n alone is n<<8 and a two-channel mix is legible at a glance.
func fakeBank(frames int) *Bank {
	b := &Bank{Triggers: map[int16]*Sound{}}
	for i := int16(0); i < TriggerSlot; i++ {
		data := make([]byte, frames)
		for j := range data {
			data[j] = byte(128 + i)
		}
		b.Effects[i] = &Sound{Slot: i, ID: BaseSoundID + i, Name: Name(i), Data: data}
	}
	for i := int16(0); i < MaxMusic; i++ {
		data := make([]byte, frames)
		for j := range data {
			// Music sits below zero so that a mix can tell a piece of music from an
			// effect by its sign.
			data[j] = byte(128 - 1 - i)
		}
		b.Music[i] = &Sound{Slot: i, ID: BaseMusicID + i, Name: "music", Data: data}
	}
	return b
}

// mix1 mixes one sample and returns it.
func mix1(e *Engine) int16 {
	var buf [1]int16
	e.Mix(buf[:])
	return buf[0]
}

// mixN mixes n samples and discards them, for advancing the channels.
func mixN(e *Engine, n int) {
	buf := make([]int16, n)
	e.Mix(buf)
}

// TestLowestPriorityWins is the heart of PlayPrioritySound: three channels, and a request goes
// to the quietest one.
func TestLowestPriorityWins(t *testing.T) {
	e := New(fakeBank(100))

	e.PlayPrioritySound(1, 300)
	e.PlayPrioritySound(2, 200)
	e.PlayPrioritySound(3, 400)
	if got, want := e.Playing(), [3]int16{1, 2, 3}; got != want {
		t.Fatalf("after filling the channels, playing = %v, want %v", got, want)
	}

	// Channel 1 holds the lowest priority (200), so a request at 250 lands there and takes
	// the sound that was on it.
	e.PlayPrioritySound(4, 250)
	if got, want := e.Playing(), [3]int16{1, 4, 3}; got != want {
		t.Errorf("playing = %v, want %v: 250 should displace the 200 on channel 1", got, want)
	}
	if e.stats.Displaced != 1 {
		t.Errorf("Displaced = %d, want 1", e.stats.Displaced)
	}

	// Now the lowest is 250 and a request under it is refused outright rather than
	// stealing a channel.
	e.PlayPrioritySound(5, 100)
	if got, want := e.Playing(), [3]int16{1, 4, 3}; got != want {
		t.Errorf("playing = %v after a refused request, want %v unchanged", got, want)
	}
	if e.stats.Refused != 1 {
		t.Errorf("Refused = %d, want 1", e.stats.Refused)
	}
}

// TestRepeatedSoundSpreadsAcrossChannels is how the four continuous sounds actually behave, and
// it is not what you would guess.
//
// A sound asked for again while it is still playing does *not* restart on its own channel: the
// policy sends every request to the quietest channel, and an idle channel at priority 0 is
// quieter than the one already playing. So a per-frame sound fills channel 0, then 1, then 2,
// and only then begins replacing itself -- from channel 0, because the ties in the channel
// search are broken by the strict `<` in favour of the lowest-numbered channel.
//
// Two consequences worth having pinned. The overlap is what makes Shred and Sizzle sound
// continuous when the samples are only two and three frames long (see bank.go). And a repeated
// loud sound takes the *whole* mixer, which is why a shredding glider silences the room.
func TestRepeatedSoundSpreadsAcrossChannels(t *testing.T) {
	e := New(fakeBank(100))

	want := [][3]int16{
		{26, NoSoundPlaying, NoSoundPlaying},
		{26, 26, NoSoundPlaying},
		{26, 26, 26},
	}
	for i, w := range want {
		e.PlayPrioritySound(26, 903)
		if got := e.Playing(); got != w {
			t.Fatalf("request %d: playing = %v, want %v", i+1, got, w)
		}
	}

	// All three channels now hold 903. The fourth request ties, and `priority >= lowest`
	// grants it -- to channel 0, the lowest-numbered of the tied channels.
	mixN(e, 10)
	e.PlayPrioritySound(26, 903)
	// pos is 16.16 fixed point, so the whole part is the frame the channel has reached. These
	// sounds carry no rate, which stepFor reads as the Macintosh rate, so ten output samples is
	// ten frames on the nose.
	frame := func(i int) int64 { return e.ch[i].pos >> 16 }
	if frame(0) != 0 {
		t.Errorf("channel 0 is %d frames in; the fourth request should have restarted it", frame(0))
	}
	if frame(1) != 10 || frame(2) != 10 {
		t.Errorf("channels 1 and 2 are at %d and %d frames; only channel 0 should have restarted",
			frame(1), frame(2))
	}
	if e.stats.Refused != 0 {
		t.Errorf("Refused = %d, want 0: an equal priority must be granted, not refused. "+
			"With `>` instead of `>=` in the policy, every continuous sound in the game would "+
			"play three times and then stop", e.stats.Refused)
	}
	if e.stats.Displaced != 1 {
		t.Errorf("Displaced = %d, want 1", e.stats.Displaced)
	}
}

// TestCompletionCallbackFreesTheChannel is CallBack0/1/2: when the last frame is consumed the
// channel drops to priority 0 and reports nothing playing.
//
// The exact frame matters, so this counts. A channel freed one sample early would let a quiet
// sound in over the tail of a loud one; freed one sample late and the last sample of every
// sound in the game would be dropped.
func TestCompletionCallbackFreesTheChannel(t *testing.T) {
	const frames = 8
	e := New(fakeBank(frames))
	e.PlayPrioritySound(5, 900)

	for i := 0; i < frames; i++ {
		if got := e.Priorities()[0]; got != 900 {
			t.Fatalf("frame %d: priority = %d, want 900 while the sound is playing", i, got)
		}
		if got, want := mix1(e), int16(5<<8); got != want {
			t.Fatalf("frame %d: sample = %d, want %d", i, got, want)
		}
	}
	if got := e.Priorities()[0]; got != 0 {
		t.Errorf("after the last frame, priority = %d, want 0", got)
	}
	if got := e.Playing()[0]; got != NoSoundPlaying {
		t.Errorf("after the last frame, playing = %d, want %d", got, NoSoundPlaying)
	}
	if got := mix1(e); got != 0 {
		t.Errorf("a finished channel mixed %d, want silence", got)
	}
	if e.Busy() {
		t.Error("Busy() is true with nothing playing")
	}
}

// TestTriggerExclusivity is Sound.c:47-51, the one rule that depends on the priority rather
// than just comparing it.
func TestTriggerExclusivity(t *testing.T) {
	e := New(fakeBank(100))
	e.trigger = e.bank.Effect(7) // stand in for a house's loaded sound

	e.PlayPrioritySound(TriggerSlot, TriggerPriority)
	if got := e.Playing()[0]; got != TriggerSlot {
		t.Fatalf("the first trigger did not play: playing = %v", e.Playing())
	}

	// A second trigger is refused even though two channels are idle -- which is the whole
	// point of the rule, and is why it cannot be expressed as a priority comparison.
	e.PlayPrioritySound(TriggerSlot, TriggerPriority)
	if got, want := e.Playing(), [3]int16{TriggerSlot, NoSoundPlaying, NoSoundPlaying}; got != want {
		t.Errorf("playing = %v, want %v: a second trigger must be refused", got, want)
	}
	if e.stats.TriggerRefused != 1 {
		t.Errorf("TriggerRefused = %d, want 1", e.stats.TriggerRefused)
	}

	// Everything quieter still plays, on the other channels.
	e.PlayPrioritySound(1, 100)
	if got := e.Playing()[1]; got != 1 {
		t.Errorf("an ordinary sound was refused while a trigger played: playing = %v", e.Playing())
	}
}

// TestFlushTriggerSound is the documented deviation from the original: the port resets the
// flushed channel's priority, where the C loses the callback that would have.
//
// The test asserts the port's behaviour and names the original's, so that anybody who changes
// this on purpose has to change a test that explains itself.
func TestFlushTriggerSound(t *testing.T) {
	e := New(fakeBank(1000))
	e.trigger = e.bank.Effect(7)

	e.PlayPrioritySound(TriggerSlot, TriggerPriority)
	e.PlayPrioritySound(1, 100)
	mixN(e, 10)

	e.FlushTriggerSound()

	if got := e.Priorities()[0]; got != 0 {
		t.Errorf("priority = %d after a flush, want 0; in the original this stays at %d "+
			"for the rest of the session because flushCmd discards the callBackCmd", got, TriggerPriority)
	}
	if got := e.Playing()[0]; got != NoSoundPlaying {
		t.Errorf("playing = %d after a flush, want %d", got, NoSoundPlaying)
	}
	if e.trigger != nil {
		t.Error("the trigger sample is still loaded after DumpTriggerSound")
	}

	// The other channels are untouched: a room change silences the house's sound and
	// nothing else.
	if got := e.Playing()[1]; got != 1 {
		t.Errorf("channel 1 lost its sound to the flush: playing = %v", e.Playing())
	}

	// And the freed channel is usable again, which is the point of the deviation.
	e.PlayPrioritySound(2, 100)
	if got := e.Playing()[0]; got != 2 {
		t.Errorf("the flushed channel refused a new sound: playing = %v", e.Playing())
	}
}

// TestLoadTriggerSoundIsOneAtATime is the `theSoundData[kMaxSounds-1] != nil` refusal, which is
// what limits a room to one sound trigger.
func TestLoadTriggerSoundIsOneAtATime(t *testing.T) {
	b := fakeBank(50)
	b.Triggers[3000] = &Sound{ID: 3000, Slot: TriggerSlot, Name: "A-hem!", Data: []byte{200, 200}}
	b.Triggers[3001] = &Sound{ID: 3001, Slot: TriggerSlot, Name: "Shhhh!", Data: []byte{100, 100}}
	e := New(b)

	if !e.LoadTriggerSound(3000) {
		t.Fatal("the first trigger sound failed to load")
	}
	if e.LoadTriggerSound(3001) {
		t.Error("a second trigger sound loaded; a room may hold only one")
	}
	if e.LoadTriggerSound(9999) {
		t.Error("a trigger sound the house does not have reported success")
	}

	e.FlushTriggerSound()
	if !e.LoadTriggerSound(3001) {
		t.Error("after a room change the slot is still held")
	}
	if got := e.trigger.Name; got != "Shhhh!" {
		t.Errorf("the loaded trigger is %q, want Shhhh!", got)
	}
}

// TestSoundOffStillRunsThePolicy pins the placement of the isSoundOn test *inside* PlaySoundN,
// which is where Sound.c has it.
//
// With sound off, a request is counted, a channel is chosen, and then nothing is played and no
// priority is assigned -- so the channels never leave priority 0 and nothing is ever refused.
// That is unobservable in the original and worth keeping because it makes Stats.Muted mean
// something precise.
func TestSoundOffStillRunsThePolicy(t *testing.T) {
	e := New(fakeBank(100))
	e.SetSoundOn(false)

	e.PlayPrioritySound(1, 900)
	e.PlayPrioritySound(2, 100)

	if got := e.Playing(); got != [3]int16{NoSoundPlaying, NoSoundPlaying, NoSoundPlaying} {
		t.Errorf("playing = %v with sound off, want nothing", got)
	}
	if got := e.Priorities(); got != [3]int16{0, 0, 0} {
		t.Errorf("priorities = %v with sound off, want all zero", got)
	}
	if e.stats.Requests != 2 || e.stats.Muted != 2 || e.stats.Refused != 0 {
		t.Errorf("requests=%d muted=%d refused=%d, want 2/2/0",
			e.stats.Requests, e.stats.Muted, e.stats.Refused)
	}
	if got := mix1(e); got != 0 {
		t.Errorf("mixed %d with sound off, want silence", got)
	}
}

// TestNilBankIsSilent is `dontLoadSounds`: the configuration a build with no extracted assets
// runs in. Every entry point must tolerate it, because the alternative is the game refusing to
// start without sounds.
func TestNilBankIsSilent(t *testing.T) {
	e := New(nil)

	e.PlayPrioritySound(1, 900)
	e.QueueMusic(0)
	if e.LoadTriggerSound(3000) {
		t.Error("a trigger sound loaded from a nil bank")
	}
	e.FlushTriggerSound()
	e.SilenceMusic()

	if e.stats.Requests != 0 {
		t.Errorf("Requests = %d with no sound system; requests should not be counted at all", e.stats.Requests)
	}
	if e.Busy() {
		t.Error("Busy() with no sound system")
	}
	buf := make([]int16, 16)
	for i := range buf {
		buf[i] = 999 // Mix must write silence, not leave the caller's buffer alone.
	}
	e.Mix(buf)
	for i, v := range buf {
		if v != 0 {
			t.Fatalf("sample %d = %d, want silence", i, v)
		}
	}
	if e.MusicAvailable() {
		t.Error("MusicAvailable with no sound system")
	}
}

// TestNilEngine covers the same ground one level up: a host with sound turned off holds a nil
// *Engine and calls the same methods, so every one of them has a nil receiver guard.
func TestNilEngine(t *testing.T) {
	var e *Engine
	e.PlayPrioritySound(1, 900)
	e.FlushTriggerSound()
	e.SilenceMusic()
	e.QueueMusic(0)
	if e.LoadTriggerSound(3000) || e.Busy() || e.MusicAvailable() {
		t.Error("a nil Engine claimed to do something")
	}
	if got := e.Playing(); got != [3]int16{NoSoundPlaying, NoSoundPlaying, NoSoundPlaying} {
		t.Errorf("nil Engine playing = %v", got)
	}
	if got := e.MusicPlaying(); got != NoPiece {
		t.Errorf("nil Engine MusicPlaying = %d", got)
	}
	buf := make([]int16, 4)
	buf[0] = 7
	e.Mix(buf)
	if buf[0] != 0 {
		t.Error("a nil Engine's Mix did not write silence")
	}
}

// TestMixSumsAndClips checks the arithmetic, including the rail.
//
// The clip level is the Macintosh's: one channel at full amplitude very nearly fills an int16,
// so two loud sounds together distort in the port exactly where they distorted in 1994. The
// test asserts that rather than assuming it, because the tempting "fix" -- scaling the sum down
// so four channels cannot clip -- would quietly make every sound in the game a quarter as loud
// as the original's.
func TestMixSumsAndClips(t *testing.T) {
	b := fakeBank(8)
	// Two channels at nearly full positive amplitude: 127<<8 each, which sums past the top
	// of an int16.
	loud := make([]byte, 8)
	for i := range loud {
		loud[i] = 255
	}
	b.Effects[1] = &Sound{Slot: 1, Data: loud}
	b.Effects[2] = &Sound{Slot: 2, Data: loud}
	e := New(b)

	e.PlayPrioritySound(1, 100)
	if got, want := mix1(e), int16(127<<8); got != want {
		t.Fatalf("one loud channel mixed %d, want %d", got, want)
	}
	e.PlayPrioritySound(2, 100) // channel 1, since channel 0 is busy at 100
	if got := mix1(e); got != 32767 {
		t.Errorf("two loud channels mixed %d, want the rail at 32767", got)
	}
	if e.stats.Clipped != 1 {
		t.Errorf("Clipped = %d, want 1", e.stats.Clipped)
	}

	// The negative rail, which is one larger in magnitude and therefore a separate case.
	quiet := make([]byte, 8)
	e2 := New(&Bank{Effects: [MaxSounds]*Sound{1: {Data: quiet}, 2: {Data: quiet}}})
	e2.PlayPrioritySound(1, 100)
	e2.PlayPrioritySound(2, 100)
	if got, want := mix1(e2), int16(-32768); got != want {
		t.Errorf("two channels at the negative extreme mixed %d, want %d", got, want)
	}
}

// TestVolumeScales checks the port's own volume control, which the original does not have: the
// Macintosh read the system volume and let the Sound Manager do the scaling.
func TestVolumeScales(t *testing.T) {
	e := New(fakeBank(16))
	e.SetVolume(FullVolume)
	e.PlayPrioritySound(4, 100)
	full := mix1(e)
	if full != 4<<8 {
		t.Fatalf("at full volume, sample = %d, want %d", full, 4<<8)
	}

	e.SetVolume(3)
	if got, want := mix1(e), int16(int32(full)*3/FullVolume); got != want {
		t.Errorf("at volume 3, sample = %d, want %d", got, want)
	}

	e.SetVolume(0)
	if got := mix1(e); got != 0 {
		t.Errorf("at volume 0, sample = %d, want silence", got)
	}

	e.SetVolume(99)
	if e.Volume() != FullVolume {
		t.Errorf("SetVolume(99) gave %d, want a clamp to %d", e.Volume(), FullVolume)
	}
	e.SetVolume(-5)
	if e.Volume() != 0 {
		t.Errorf("SetVolume(-5) gave %d, want a clamp to 0", e.Volume())
	}
}

// TestEventsRecordTheOutcome checks the hook the replay harness builds its sound column from.
func TestEventsRecordTheOutcome(t *testing.T) {
	e := New(fakeBank(100))
	var got []Event
	e.On = func(ev Event) { got = append(got, ev) }

	e.PlayPrioritySound(1, 300) // channel 0
	mixN(e, 5)
	e.PlayPrioritySound(2, 400) // channel 1, both idle-ish
	e.PlayPrioritySound(3, 500) // channel 2
	e.PlayPrioritySound(4, 350) // displaces slot 1 on channel 0
	e.PlayPrioritySound(5, 100) // refused

	want := []Event{
		{Sample: 0, Slot: 1, Priority: 300, Channel: 0, Displaced: NoSoundPlaying},
		{Sample: 5, Slot: 2, Priority: 400, Channel: 1, Displaced: NoSoundPlaying},
		{Sample: 5, Slot: 3, Priority: 500, Channel: 2, Displaced: NoSoundPlaying},
		{Sample: 5, Slot: 4, Priority: 350, Channel: 0, Displaced: 1},
		{Sample: 5, Slot: 5, Priority: 100, Channel: -1, Displaced: NoSoundPlaying},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestMissingSampleIsRefused covers the one door the C cannot walk through: a request for the
// trigger slot when the sample has been freed. It must decline rather than panic, because the
// number comes from a house file.
func TestMissingSampleIsRefused(t *testing.T) {
	e := New(fakeBank(10))
	e.PlayPrioritySound(TriggerSlot, TriggerPriority)
	if e.Busy() {
		t.Error("a trigger played with no trigger sample loaded")
	}
	if e.stats.Refused != 1 {
		t.Errorf("Refused = %d, want 1", e.stats.Refused)
	}
	// Out-of-range slots come from the same place and must be as harmless.
	e.PlayPrioritySound(200, 100)
	e.PlayPrioritySound(-3, 100)
	if e.Busy() {
		t.Error("an out-of-range slot played something")
	}
}
