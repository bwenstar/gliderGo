package audio

// Sound.c, as a mixer: the three sampled channels, the priority policy that decides which
// one a request lands on, and the completion callback that frees it again.
//
// ---------------------------------------------------------------------------
// Why there is a mixer here at all
// ---------------------------------------------------------------------------
//
// There is no mixer in the original. `SndDoImmediate(channel0, bufferCmd)` hands a sample to
// the Sound Manager, the Sound Manager sums the open channels in an interrupt handler, and the
// hardware plays the result -- so the 1994 code contains the *policy* (which channel, at what
// priority) and none of the arithmetic. A port on a modern host has to supply the arithmetic,
// because every audio API on Linux, Windows and macOS wants a single stream of interleaved
// samples and will not sum four for you.
//
// That is the whole of what this file adds to Sound.c, and it is deliberately the *only*
// thing it adds. The policy below is transcribed statement for statement, including the
// asymmetry where channel 2 assigns its two globals after issuing its commands rather than
// before (Sound.c:210-211 against :144-145), because the difference is invisible and
// pretending it is not there would be one more place a reader has to trust the port.
//
// ---------------------------------------------------------------------------
// One goroutine, on purpose
// ---------------------------------------------------------------------------
//
// **Engine is not safe for concurrent use, and this is a design decision rather than an
// omission.** The Sound Manager mixed on an interrupt, which is the 1994 equivalent of a
// second thread, and a port that copied that shape would need a lock around every channel and
// -- much worse -- would have to call the music chain (NextPiece, below) from the mixing
// thread. NextPiece reads and writes the game's music cursor, so that lock would have to
// extend into internal/game and would put a mutex on the frame loop's path.
//
// Instead, both drivers call Mix from the goroutine that owns the World:
//
//	the recorded path   internal/replay mixes exactly SamplesPerFrame per game frame, so
//	                    the stream is a pure function of the simulation and two runs of one
//	                    script produce byte-identical audio. That is what makes the WAV a
//	                    test rather than a demo.
//	the live path       cmd/glidergo mixes what the wall clock says is due, from inside
//	                    Present -- which the game calls at least once a frame and 160 times
//	                    during a room transition, so the device stays fed even through a
//	                    wipe that takes longer than a frame.
//
// The goroutine boundary is pushed out to the sink, which sees bytes and no engine state at
// all. See sink.go.

// SamplesPerFrame is one game frame's worth of audio.
//
// kTicksPerFrame is 2 on the Mac's 60.15 Hz clock, so a frame is 30.075 of a second and
// 22254.5454.../30.075 is 739.98 samples. Rounding to 740 puts the recorded stream 0.003%
// slow -- 20 milliseconds over a ten-minute session -- against a game whose own frame clock
// is a tick-granular busy-wait. Nothing in the port measures audio against wall time, so the
// error has nowhere to accumulate into anything observable.
//
// The assets confirm the number independently, which is a pleasant surprise for a constant
// derived from two #defines: the Hiss sample is 2960 frames long, which is 4 x 740 exactly, and
// Input.c asks for it every fourth game frame. The author was cutting samples to this same
// arithmetic. See bank.go's note on the four continuous sounds.
const SamplesPerFrame = 740

// FullVolume is the top of the Macintosh volume scale that UnivGetSoundVolume reports
// (Music.c:55). The original reads the *system* volume and only ever compares it against
// zero, which is the one thing it can do with it: `bufferCmd` plays at the channel's own
// amplitude and the Sound Manager scales the mix. The port keeps the scale and does the
// scaling itself, so that -volume can turn the game down without turning the desktop down.
const FullVolume = 7

// Event is one request to play a sound and what became of it.
//
// It exists for the trace and for bug reports, and it is a *record* rather than a
// notification: nothing in the game or the mixer reads one back. That is what makes it safe
// for the replay harness to hang a hook on the engine without changing what the engine does.
type Event struct {
	// Sample is the mixer's cursor when the request arrived -- how many samples had been
	// mixed. Divided by SamplesPerFrame it is the frame, in the recorded path; in the live
	// path it is only a stamp.
	Sample int64

	Slot     int16 // which sound: an index into Names
	Priority int16 // the priority the caller asked at

	// Channel is 0, 1 or 2 for a granted request and -1 for a refused one. Which channel
	// is the interesting half: a sound that lands on the channel a long shred is holding
	// cuts the shred off, and that is audible in a way the request log alone cannot show.
	Channel int

	// Displaced is the slot the grant interrupted, or -1. Sound.c has no notion of this --
	// bufferCmd on a busy channel simply replaces what was playing -- and it is the single
	// most useful thing this record carries when somebody reports that a sound "did not
	// play": very often it did, for four milliseconds.
	Displaced int16
}

// Stats is what a session can say about its own audio.
//
// Cumulative and monotonic, like game.Diagnostics, and read by nothing inside the engine for
// the same reason: a counter the mixer consulted would be a counter that could change the
// mix.
type Stats struct {
	Requests  int64 // every call to PlayPrioritySound that got past the dontLoad guard
	Granted   int64 // ...of which these reached a channel
	Refused   int64 // ...and these lost to a busier one
	Muted     int64 // granted a channel but dropped by isSoundOn being false
	Displaced int64 // grants that cut off a sound already playing on that channel

	// TriggerRefused counts the requests the exclusivity rule turned down: a second
	// trigger sound while one is still playing (Sound.c:47-51). Separated out because it
	// is the one refusal that is a *feature* -- it stops a room full of sound triggers
	// from drowning itself -- so it should not inflate the number that means "the mix is
	// too busy".
	TriggerRefused int64

	// MusicStarted counts pieces the music channel began. The score is seven pieces on a
	// sixteen-entry loop, so this rising steadily is the score playing and this stuck at 1
	// or 2 while the game runs is StartMusic's two queued buffers having drained with
	// nothing chaining behind them -- which is the failure this counter exists to name.
	MusicStarted int64

	Samples int64 // total samples mixed, on both paths

	// Clipped counts samples that hit the rail. The Macintosh clipped at eight bits and so
	// does this, at the same absolute level; a nonzero count is not a defect, it is two
	// loud sounds at once, and the number is here so that a report of "distorted audio"
	// can be answered with how often. See Mix.
	Clipped int64
}

// channel is channel0, channel1 or channel2 together with the two globals that shadow it:
// priorityN (Sound.c:31) and soundPlayingN (:32).
//
// The Sound Manager holds the sample and the position; the C holds the priority and the slot
// beside it, and the two are kept in step by hand. Keeping all four in one struct is the one
// structural liberty this file takes with Sound.c, and it is what makes the completion
// callback a method rather than three near-identical functions.
type channel struct {
	priority int16 // priorityN. 0 is idle, and is what the callback restores.
	playing  int16 // soundPlayingN. NoSoundPlaying when idle.

	data []byte // the sample being played, nil when idle

	// pos is the read cursor into data and step is how far it moves per output sample, both
	// in 16.16 fixed point. For every sound the application ships, step is exactly FixedOne
	// and pos is therefore an integer at every sample -- the arithmetic is a copy. It is only
	// the houses' odd-rate trigger sounds that ever land between two bytes, and what happens
	// then is drop-sample conversion: the byte to the left. See Sound.Step.
	pos, step int64
}

// NoSoundPlaying is kNoSoundPlaying (Sound.c:16).
const NoSoundPlaying int16 = -1

// next advances the channel one frame and returns its contribution to the mix.
//
// The `pos >= len` branch is CallBack0/1/2 (Sound.c:217-261): the Sound Manager's completion
// callback, whose entire body is `priorityN = 0; soundPlayingN = kNoSoundPlaying`. Running it
// here, at the moment the last frame is consumed, is what makes the port's channel bookkeeping
// track the C's -- and it is why the *length of every sample* is part of the port's behaviour
// rather than a detail of its assets: the shred is 1934 frames long, so it holds priority 903
// for 1934 samples -- two and a half game frames -- and every request below 903 during that
// time depends on it.
func (c *channel) next() int32 {
	if c.data == nil {
		return 0
	}
	// 128 is silence in an unsigned 8-bit sample, and the shift is the only conversion in
	// the mixer: one channel at full amplitude spans -32768..+32512, which is the same
	// absolute level the Macintosh's own full scale had. Two of them therefore clip in the
	// port exactly where they clipped in 1994.
	v := (int32(c.data[c.pos>>16]) - 128) << 8
	c.pos += c.step
	if c.pos>>16 >= int64(len(c.data)) {
		c.data = nil
		c.pos = 0
		c.priority = 0
		c.playing = NoSoundPlaying
	}
	return v
}

// start is the bufferCmd: hand the channel a sample, replacing whatever it was playing.
//
// SndDoImmediate does not wait its turn, which is why a busy channel is simply overwritten and
// why Event.Displaced is worth recording.
func (c *channel) start(s *Sound, slot, priority int16) (displaced int16) {
	displaced = NoSoundPlaying
	if c.data != nil {
		displaced = c.playing
	}
	c.priority = priority
	c.playing = slot
	c.data = s.Data
	c.pos = 0

	// A zero Step is the Macintosh rate, so that a Sound assembled by hand -- in a test, or by
	// a future editor that has not filled the field in -- plays at its natural speed rather
	// than standing still forever on byte zero.
	c.step = s.Step
	if c.step <= 0 {
		c.step = FixedOne
	}
	return displaced
}

// quiet is the quietCmd/flushCmd pair, plus the one thing the pair does not do in the
// original. See FlushAnyTriggerPlaying.
func (c *channel) quiet() {
	c.data = nil
	c.pos = 0
	c.priority = 0
	c.playing = NoSoundPlaying
}

// Engine is InitSound's world: the bank, the three channels, the music channel and the two
// preferences that gate them.
type Engine struct {
	bank *Bank

	ch    [3]channel
	music musicChannel

	// trigger is theSoundData[kMaxSounds-1]: the one sample a house's sound trigger gets
	// to occupy, loaded per room by LoadTriggerSound and freed by DumpTriggerSound.
	trigger *Sound

	// soundOn is isSoundOn and dontLoad is dontLoadSounds. Both are the C's, and they are
	// two flags rather than one because they are refused at different depths: dontLoad
	// short-circuits PlayPrioritySound before it looks at a channel (Sound.c:44), while
	// soundOn is tested inside PlaySound0/1/2 *after* the channel has been chosen (:142).
	// So a muted game still runs the whole policy and still counts a request against it.
	// That is invisible in the original and it is exactly why it is worth keeping: it is
	// the difference between "the mix was too busy" and "the player turned sound off",
	// which is a question a bug report actually asks.
	soundOn  bool
	dontLoad bool

	// volume is 0..FullVolume. Zero is what StartMusic reads as "the machine is muted" and
	// declines to start the score for (Music.c:57).
	volume int16

	// NextPiece is the music channel's callBackCmd (Music.c:170-216): the score has run
	// out of queued pieces and the game is being asked for the next one. It is called from
	// inside Mix, on Mix's goroutine, which is the whole reason Mix has only one -- see the
	// file comment. nil leaves the music channel to fall silent when its queue empties,
	// which is what a build with no game attached should do.
	NextPiece func() int16

	// On, if set, is called once per request with what became of it. The replay harness
	// sets it to build the trace's sound column; a released build leaves it nil.
	On func(Event)

	stats Stats
}

// New is InitSound (Sound.c:438-474) and the channel half of InitMusic (Music.c:312-369):
// take a loaded bank and open the channels.
//
// A nil bank is `dontLoadSounds`, not an error. That is the state a build with no extracted
// assets is in, it is the state the original is in on a Mac that could not spare the memory,
// and every guard downstream of it is the C's own -- so a silent Engine is a supported
// configuration rather than a degraded one, and the game cannot tell the difference between it
// and no Engine at all.
func New(b *Bank) *Engine {
	e := &Engine{
		bank:     b,
		soundOn:  true,
		dontLoad: b == nil,
		volume:   FullVolume,
	}
	e.reset()
	return e
}

// reset is InitSound's five assignments (Sound.c:451-456).
func (e *Engine) reset() {
	for i := range e.ch {
		e.ch[i] = channel{playing: NoSoundPlaying}
	}
	e.music = musicChannel{channel: channel{playing: NoSoundPlaying}}
}

// SetSoundOn is the isSoundOn preference and SetVolume is the system volume. Both take effect
// on the next request rather than on the current mix, which is the original's behaviour for
// the same reason: neither one reaches into a channel that is already playing.
func (e *Engine) SetSoundOn(on bool) { e.soundOn = on }
func (e *Engine) SoundOn() bool      { return e.soundOn }

func (e *Engine) SetVolume(v int16) {
	if v < 0 {
		v = 0
	}
	if v > FullVolume {
		v = FullVolume
	}
	e.volume = v
}

func (e *Engine) Volume() int16 { return e.volume }

// Bank is the samples this engine plays, for a caller that wants to report on them.
func (e *Engine) Bank() *Bank { return e.bank }

// Stats is the session's counters. A copy, so that a caller cannot reach into the engine and
// zero one.
func (e *Engine) Stats() Stats { return e.stats }

// PlayPrioritySound is Sound.c:40-85, and it is the whole of the port's channel policy.
//
// Read the two guards in order, because they refuse for different reasons:
//
//	dontLoadSounds     there is no sound system. Every request is dropped and nothing is
//	                   counted, because a build with no audio should not accumulate
//	                   statistics about audio.
//	the trigger rule   a request at kTriggerPriority is refused outright while *any*
//	                   channel is already playing one. Not "the lowest channel" -- any.
//	                   This is the only place a sound's priority changes the *rule* and not
//	                   just the comparison, and it is what stops a room whose triggers all
//	                   fire at once from playing three of them over each other.
//
// Then the policy: find the channel with the lowest priority -- ties going to the
// lowest-numbered channel, because the comparisons are strict -- and play there if the
// request is at least as loud. `>=` and not `>`, so a request ties into the channel and
// displaces what is on it, which is what makes a repeated sound (the battery's thrust, the
// shredder) restart every frame instead of playing once.
func (e *Engine) PlayPrioritySound(which, priority int16) {
	if e == nil || e.dontLoad {
		return
	}
	e.stats.Requests++

	if priority == TriggerPriority {
		for i := range e.ch {
			if e.ch[i].priority == TriggerPriority {
				e.stats.TriggerRefused++
				e.stats.Refused++
				e.note(Event{Slot: which, Priority: priority, Channel: -1, Displaced: NoSoundPlaying})
				return
			}
		}
	}

	whosLowest, lowest := 0, e.ch[0].priority
	if e.ch[1].priority < lowest {
		lowest, whosLowest = e.ch[1].priority, 1
	}
	if e.ch[2].priority < lowest {
		lowest, whosLowest = e.ch[2].priority, 2
	}

	if priority < lowest {
		e.stats.Refused++
		e.note(Event{Slot: which, Priority: priority, Channel: -1, Displaced: NoSoundPlaying})
		return
	}
	e.playSound(whosLowest, which, priority)
}

// playSound is PlaySound0/1/2 (Sound.c:133-213), which are three copies of one function.
//
// The `isSoundOn` test is inside, exactly where the C has it, and the reason to keep it there
// rather than lift it into PlayPrioritySound is stated on Engine.soundOn: a muted request has
// already chosen its channel, and only the two assignments and the bufferCmd are skipped. So
// with sound off the channels never leave priority 0 and every subsequent request is
// "granted" to channel 0 -- which is unobservable, and is the original's.
func (e *Engine) playSound(n int, slot, priority int16) {
	if !e.soundOn {
		e.stats.Muted++
		e.stats.Granted++
		e.note(Event{Slot: slot, Priority: priority, Channel: n, Displaced: NoSoundPlaying})
		return
	}

	s := e.sample(slot)
	if s == nil {
		// The C cannot reach this: theSoundData[] is fully loaded or failedSound is set
		// and PlayPrioritySound returned above. The port can, through exactly one door --
		// a sound trigger whose sample was freed by a room change between the hot spot
		// being made and the trigger firing -- so it answers the way the guarded reads in
		// internal/game do, by declining rather than by inventing a sample.
		e.stats.Refused++
		e.note(Event{Slot: slot, Priority: priority, Channel: -1, Displaced: NoSoundPlaying})
		return
	}

	displaced := e.ch[n].start(s, slot, priority)
	if displaced != NoSoundPlaying {
		e.stats.Displaced++
	}
	e.stats.Granted++
	e.note(Event{Slot: slot, Priority: priority, Channel: n, Displaced: displaced})
}

// sample is theSoundData[slot], with the trigger slot's indirection.
func (e *Engine) sample(slot int16) *Sound {
	if slot == TriggerSlot {
		return e.trigger
	}
	return e.bank.Effect(slot)
}

func (e *Engine) note(ev Event) {
	if e.On == nil {
		return
	}
	ev.Sample = e.stats.Samples
	e.On(ev)
}

// LoadTriggerSound is Sound.c:265-303: claim the one reserved slot for a house's own sound.
//
// It reports whether the sound loaded, which is all CreateActiveRects asks -- and the answer
// decides whether a sound trigger gets a hot spot at all, so **this function changes what the
// room looks like**, not just what it sounds like. internal/game/hotspots.go's
// loadTriggerSound is the caller and has the other half of the note.
//
// The `e.trigger != nil` refusal is the C's `theSoundData[kMaxSounds - 1] != nil`: the second
// sound trigger in a room fails, whatever sound it names. The port's game side reproduces the
// same limit independently through Room.TriggerSoundHeld, which is not redundant -- that flag
// is what a silent build (no Engine at all) still honours, so a room composes the same way
// with and without audio in that one respect.
func (e *Engine) LoadTriggerSound(id int16) bool {
	if e == nil || e.dontLoad || e.trigger != nil {
		return false
	}
	s := e.bank.Trigger(id)
	if s == nil {
		return false
	}
	e.trigger = s
	return true
}

// DumpTriggerSound is Sound.c:307-312, and FlushAnyTriggerPlaying is :89-129. DrawLocale calls
// them as a pair on every room change (RoomGraphics.c:57-58), so this method is the pair.
//
// **The port resets the flushed channel's priority and the original does not.** That is a
// deliberate deviation and the reasoning is worth having in full, because it is the only place
// in the audio path where the port declines to reproduce what the 1994 build did:
//
// The C issues quietCmd to stop the sample and flushCmd to empty the channel's command queue.
// The callBackCmd that PlaySoundN queued behind the bufferCmd is *in* that queue, so flushing
// it means the completion callback never runs -- and the callback is the only thing that ever
// writes priorityN back to 0. The channel is therefore left claiming kTriggerPriority, which
// is 999, for the rest of the session: InitSound runs once, at launch (Main.c:337), and
// nothing else resets it. One interrupted trigger sound costs a channel; three leave
// PlayPrioritySound with lowestPriority == 999, which refuses every ordinary request, while
// the trigger rule above refuses every trigger. The game goes silent and stays silent.
//
// Reaching it needs a house with a sound trigger, a sound long enough to still be playing on
// the way out of the room, and a player who leaves -- which is a description of Art Museum,
// whose "Security" is 2.2 seconds and whose triggers are in corridors. So the port resets the
// three fields, as though the callback had run.
//
// Two things make that safe rather than presumptuous. Nothing in the simulation reads channel
// state, so no replay and no digest can move. And the deviation can only ever make the port
// louder than the original, never quieter, so it cannot hide a sound the 1994 build played.
// Recorded in docs/IMPROVEMENTS.md 2.47, along with the one piece of evidence that the author
// knew something was wrong here: the call to FlushAnyTriggerPlaying inside LoadTriggerSound is
// commented out (Sound.c:275).
func (e *Engine) FlushTriggerSound() {
	if e == nil {
		return
	}
	for i := range e.ch {
		if e.ch[i].priority == TriggerPriority {
			e.ch[i].quiet()
		}
	}
	e.trigger = nil
}

// Mix fills dst with the sum of the four channels and advances every one of them by len(dst)
// frames.
//
// It is the only place in the package that produces samples, and it is deliberately
// arithmetic-free beyond the sum: no filtering, no interpolation, no panning, and no rate
// conversion beyond the one the original asked for. Every channel is opened with
// `initNoInterp + initMono` (Sound.c:379), which is a Sound Manager channel that converts the
// header's sample rate to the hardware's by *dropping and repeating samples* and does not
// interpolate between them, in mono. That is exactly what channel.next does with Sound.Step, and
// it is the whole of the port's resampling: a filter or a pan here would be adding something the
// game never had, and honouring the header rate is something it always did.
//
// The clip is at the same absolute level the Macintosh's was: one channel at full amplitude
// spans very nearly the whole int16 range, so two loud sounds together distort here exactly as
// they did in 1994. Stats.Clipped counts it rather than hiding it, and the alternative --
// scaling the sum by a quarter so four channels can never clip -- is rejected on purpose: it
// would make every single sound in the game a quarter as loud as the original's to buy
// headroom the game almost never uses.
func (e *Engine) Mix(dst []int16) {
	if e == nil {
		for i := range dst {
			dst[i] = 0
		}
		return
	}
	for i := range dst {
		acc := e.ch[0].next() + e.ch[1].next() + e.ch[2].next() + e.music.next(e)

		if e.volume != FullVolume {
			acc = acc * int32(e.volume) / FullVolume
		}
		switch {
		case acc > 32767:
			acc = 32767
			e.stats.Clipped++
		case acc < -32768:
			acc = -32768
			e.stats.Clipped++
		}
		dst[i] = int16(acc)
	}
	e.stats.Samples += int64(len(dst))
}

// Busy reports whether anything is playing. Used only by reports and tests: the game never
// asks, because the original has nothing to ask with.
func (e *Engine) Busy() bool {
	if e == nil {
		return false
	}
	for i := range e.ch {
		if e.ch[i].data != nil {
			return true
		}
	}
	return e.music.data != nil
}

// Playing is the three soundPlayingN globals, in order, for a report that wants to say what
// was on the channels when something went wrong.
func (e *Engine) Playing() [3]int16 {
	var out [3]int16
	if e == nil {
		return [3]int16{NoSoundPlaying, NoSoundPlaying, NoSoundPlaying}
	}
	for i := range e.ch {
		out[i] = e.ch[i].playing
	}
	return out
}

// Priorities is the three priorityN globals, in order. Beside Playing it is the whole of the
// channel state, and the pair is what makes a "my sound did not play" report answerable.
func (e *Engine) Priorities() [3]int16 {
	var out [3]int16
	if e == nil {
		return out
	}
	for i := range e.ch {
		out[i] = e.ch[i].priority
	}
	return out
}
