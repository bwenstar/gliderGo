package audio

// The two ways to drive Mix: by the frame, and by the clock.
//
// This file is where time enters the package, and it is deliberately the *only* file where it
// does. The Engine has no clock, no goroutine and no timer: it turns requests and a sample
// count into samples. That is what makes the recorded path bit-exact -- FrameTick asks for
// exactly one frame's worth, every frame, so the stream is a function of the simulation and
// nothing else -- and it is what makes the live path's imprecision a property of thirty lines
// here rather than of the whole audio path.
//
//	FrameTick   internal/replay, and any headless run. Exactly SamplesPerFrame per game
//	            frame. Two runs of one script produce byte-identical audio, which is what
//	            lets a WAV be a fixture.
//	ClockTick   cmd/glidergo, from inside Present. Mixes what the wall clock says is due.
//	            Present is called once per frame in play and once per strip during a room
//	            wipe -- 116 to 160 times -- so the device keeps being fed through the one
//	            part of the game that takes much longer than a frame to draw.
//
// The live path is driven from Present rather than from a ticker goroutine for the reason given
// at length in engine.go: the music chain calls back into the game to walk the score cursor,
// and a ticker would put that call on another goroutine.

import "time"

// Pump drives an Engine into a Sink.
//
// Not safe for concurrent use, like the Engine it drives, and for the same reason.
type Pump struct {
	e    *Engine
	sink Sink

	buf   []int16
	mixed int64

	// cursor is where the stream is *supposed* to be, and mixed is how much was actually
	// produced. They are the same number on the recorded path and they come apart on the
	// live one, by exactly Skipped: after a stall the cursor jumps forward to meet the wall
	// clock while mixed does not. Keeping them separate is what lets Mixed() stay an honest
	// count of samples rather than a position that sometimes lies about one.
	cursor int64

	start time.Time

	// Now is the clock, injectable so that pump_test.go can advance time by hand instead
	// of sleeping. Defaults to time.Now.
	Now func() time.Time

	// Skipped counts samples the clock said were due but which were never mixed, because
	// the game stalled for longer than maxCatchUp. See ClockTick: this is the counter that
	// distinguishes "the audio glitched because the mixer is wrong" from "the audio
	// glitched because the frame loop was blocked for half a second", and the two have
	// entirely different fixes.
	Skipped int64

	// Err latches the first sink error. The game is never told: a failed sink is silence,
	// not a crash, which is the original's failedSound behaviour.
	Err error
}

// maxCatchUp is the most one ClockTick will mix, in samples: about four frames.
//
// When the game stalls -- a garbage collection, a window manager freeze, a debugger breakpoint
// -- the clock keeps running and the mixer does not. On the next tick the deficit could be
// seconds, and mixing all of it would dump seconds of stale audio into the player at once,
// which sounds like a machine-gun burst of everything that was playing when the stall began.
// Mixing at most four frames and then jumping the cursor forward drops the rest instead, so a
// stall is heard as a gap. A gap is what actually happened.
const maxCatchUp = 4 * SamplesPerFrame

// NewPump returns a Pump. A nil Engine or Sink is allowed and does nothing, which is what a
// build with sound off uses so that the hosts need no `if audio != nil` around every call.
func NewPump(e *Engine, s Sink) *Pump {
	return &Pump{e: e, sink: s, Now: time.Now}
}

// FrameTick mixes exactly one frame's worth of audio. The deterministic path.
func (p *Pump) FrameTick() {
	p.mix(SamplesPerFrame)
}

// ClockTick mixes whatever the wall clock says is due since the first call.
//
// The first call only starts the clock: there is no deficit yet, and mixing on it would emit a
// buffer of silence before the game has asked for a single sound.
func (p *Pump) ClockTick() {
	if p == nil || p.e == nil {
		return
	}
	now := p.Now()
	if p.start.IsZero() {
		p.start = now
		return
	}

	// Integer arithmetic on nanoseconds, so the cursor cannot drift the way a float
	// accumulator would over a long session: `due` is recomputed from the start time every
	// tick rather than accumulated.
	due := now.Sub(p.start).Nanoseconds() * Rate / int64(time.Second)
	want := due - p.cursor
	if want <= 0 {
		return
	}
	if want > maxCatchUp {
		p.Skipped += want - maxCatchUp
		p.cursor = due - maxCatchUp
		want = maxCatchUp
	}
	p.mix(int(want))
}

// mix fills the buffer and hands it to the sink.
func (p *Pump) mix(n int) {
	if p == nil || p.e == nil || n <= 0 {
		return
	}
	if cap(p.buf) < n {
		p.buf = make([]int16, n)
	}
	p.buf = p.buf[:n]
	p.e.Mix(p.buf)
	p.mixed += int64(n)
	p.cursor += int64(n)

	if p.sink == nil {
		return
	}
	if err := p.sink.Write(p.buf); err != nil && p.Err == nil {
		p.Err = err
	}
}

// Mixed is how many samples have actually been produced, which on the recorded path is the
// frame count times SamplesPerFrame and is worth asserting on for exactly that reason. On the
// live path it falls short of the elapsed time by Skipped.
func (p *Pump) Mixed() int64 {
	if p == nil {
		return 0
	}
	return p.mixed
}

// Close closes the sink. Safe on a nil Pump and on one with no sink, so a host can call it
// unconditionally on the way out.
func (p *Pump) Close() error {
	if p == nil || p.sink == nil {
		return nil
	}
	return p.sink.Close()
}
