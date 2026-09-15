package audio

// Tests for the WAV writer, the pump's two pacing modes and the encoder.
//
// The Pipe is not tested here: it starts a subprocess and writes to a sound card, and a test
// that did either would either be skipped on every build machine or would make noise on
// somebody's desktop. What is testable about it -- that the encoding is right and that a full
// queue drops instead of blocking -- is covered by the encoder tests below and by the bounded
// channel's own semantics.

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWAVRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.wav")

	w, err := CreateWAV(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []int16{0, 1, -1, 32767, -32768, 1234, -4321}
	if err := w.Write(want); err != nil {
		t.Fatal(err)
	}
	if err := w.Write(nil); err != nil {
		t.Fatal(err)
	}
	if got := w.Samples(); got != int64(len(want)) {
		t.Errorf("Samples() = %d, want %d", got, len(want))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("a second Close returned %v, want nil", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != wavHeader+len(want)*2 {
		t.Fatalf("file is %d bytes, want %d", len(b), wavHeader+len(want)*2)
	}

	// The header fields a player actually reads. Getting the rate or the channel count
	// wrong is the difference between the game's audio and a chipmunk.
	for _, c := range []struct {
		name string
		got  uint32
		want uint32
	}{
		{"RIFF size", binary.LittleEndian.Uint32(b[4:]), uint32(36 + len(want)*2)},
		{"fmt size", binary.LittleEndian.Uint32(b[16:]), 16},
		{"format", uint32(binary.LittleEndian.Uint16(b[20:])), 1},
		{"channels", uint32(binary.LittleEndian.Uint16(b[22:])), 1},
		{"sample rate", binary.LittleEndian.Uint32(b[24:]), Rate},
		{"byte rate", binary.LittleEndian.Uint32(b[28:]), Rate * 2},
		{"block align", uint32(binary.LittleEndian.Uint16(b[32:])), 2},
		{"bits", uint32(binary.LittleEndian.Uint16(b[34:])), 16},
		{"data size", binary.LittleEndian.Uint32(b[40:]), uint32(len(want) * 2)},
	} {
		if c.got != c.want {
			t.Errorf("header %s = %d, want %d", c.name, c.got, c.want)
		}
	}
	if string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" || string(b[12:16]) != "fmt " || string(b[36:40]) != "data" {
		t.Errorf("chunk tags are wrong: %q", b[:44])
	}

	if got := Decode(b[wavHeader:]); len(got) != len(want) {
		t.Fatalf("decoded %d samples, want %d", len(got), len(want))
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("sample %d = %d, want %d", i, got[i], want[i])
			}
		}
	}
}

func TestWAVWriteAfterClose(t *testing.T) {
	w, err := CreateWAV(filepath.Join(t.TempDir(), "out.wav"))
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Write([]int16{1}); err == nil {
		t.Error("writing to a closed WAV succeeded")
	}
}

// TestPumpFrameTick is the deterministic path: one frame in, one frame's samples out, and the
// same stream twice.
//
// The repeatability is the point. It is what lets a replay script carry an audio digest, and it
// is only true because the Engine has no clock in it.
func TestPumpFrameTick(t *testing.T) {
	run := func() []int16 {
		e := New(fakeBank(2000))
		var out []int16
		sink := &collect{}
		p := NewPump(e, sink)
		for f := 0; f < 5; f++ {
			if f == 1 {
				e.PlayPrioritySound(26, 903)
			}
			if f == 3 {
				e.PlayPrioritySound(4, 400)
			}
			p.FrameTick()
		}
		if got, want := p.Mixed(), int64(5*SamplesPerFrame); got != want {
			t.Fatalf("Mixed() = %d, want %d", got, want)
		}
		out = append(out, sink.samples...)
		return out
	}

	a, b := run(), run()
	if len(a) != 5*SamplesPerFrame {
		t.Fatalf("got %d samples, want %d", len(a), 5*SamplesPerFrame)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("two identical runs differ at sample %d: %d against %d", i, a[i], b[i])
		}
	}
	// The first frame is before any sound was asked for, so it must be silent -- a pump
	// that primed itself with a partial buffer would show up here.
	for i := 0; i < SamplesPerFrame; i++ {
		if a[i] != 0 {
			t.Fatalf("frame 0 sample %d = %d, want silence", i, a[i])
		}
	}
	if a[SamplesPerFrame] == 0 {
		t.Error("frame 1 is silent; the sound requested on it did not reach the mix")
	}
}

// TestPumpClockTick drives the live path with a hand-cranked clock.
func TestPumpClockTick(t *testing.T) {
	e := New(fakeBank(100000))
	sink := &collect{}
	p := NewPump(e, sink)

	now := time.Unix(1000, 0)
	p.Now = func() time.Time { return now }

	// The first tick only starts the clock: mixing on it would emit a buffer of silence
	// before the game has asked for anything.
	p.ClockTick()
	if p.Mixed() != 0 {
		t.Fatalf("the first ClockTick mixed %d samples, want 0", p.Mixed())
	}

	// A tenth of a second is 2225.5 samples; the pump mixes the whole part.
	now = now.Add(100 * time.Millisecond)
	p.ClockTick()
	if got, want := p.Mixed(), int64(2225); got != want {
		t.Errorf("after 100ms, Mixed() = %d, want %d", got, want)
	}

	// No time passing means nothing to mix, which is the common case: Present is called
	// many times per frame during a room wipe.
	p.ClockTick()
	if got, want := p.Mixed(), int64(2225); got != want {
		t.Errorf("a tick with no elapsed time mixed up to %d, want %d", got, want)
	}

	// The cursor is recomputed from the start time rather than accumulated, so it cannot
	// drift: after a full second the total is the rate, not the sum of nine roundings.
	for i := 0; i < 9; i++ {
		now = now.Add(100 * time.Millisecond)
		p.ClockTick()
	}
	if got, want := p.Mixed(), int64(Rate); got != want {
		t.Errorf("after one second, Mixed() = %d, want exactly %d", got, want)
	}
	if p.Skipped != 0 {
		t.Errorf("Skipped = %d with the clock kept up", p.Skipped)
	}
	if len(sink.samples) != int(p.Mixed()) {
		t.Errorf("the sink got %d samples but the pump mixed %d", len(sink.samples), p.Mixed())
	}
}

// TestPumpClampsCatchUp is the stall case: a game blocked for a second must not dump a second of
// stale audio into the player when it comes back.
func TestPumpClampsCatchUp(t *testing.T) {
	e := New(fakeBank(100))
	p := NewPump(e, &collect{})
	now := time.Unix(1000, 0)
	p.Now = func() time.Time { return now }

	p.ClockTick()
	now = now.Add(time.Second)
	p.ClockTick()

	if got := p.Mixed(); got != maxCatchUp {
		t.Errorf("after a one-second stall the pump mixed %d samples, want the clamp at %d", got, maxCatchUp)
	}
	if want := int64(Rate - maxCatchUp); p.Skipped != want {
		t.Errorf("Skipped = %d, want %d", p.Skipped, want)
	}

	// And it resynchronises rather than staying a second behind: the next tick asks for the
	// samples due since the stall, not since the start.
	now = now.Add(100 * time.Millisecond)
	p.ClockTick()
	if got, want := p.Mixed()-maxCatchUp, int64(2225); got != want {
		t.Errorf("the tick after a stall mixed %d samples, want %d", got, want)
	}
}

// TestPumpTolerates covers the two nil configurations a host with sound off uses.
func TestPumpTolerates(t *testing.T) {
	var nilPump *Pump
	nilPump.FrameTick()
	nilPump.ClockTick()
	if nilPump.Mixed() != 0 {
		t.Error("a nil Pump mixed something")
	}
	if err := nilPump.Close(); err != nil {
		t.Errorf("closing a nil Pump: %v", err)
	}

	p := NewPump(nil, nil)
	p.FrameTick()
	p.ClockTick()
	if err := p.Close(); err != nil {
		t.Errorf("closing an engineless Pump: %v", err)
	}

	// An Engine with no sink still mixes -- which is what a headless fidelity run does when
	// it only wants the event log.
	e := New(fakeBank(10))
	p2 := NewPump(e, nil)
	p2.FrameTick()
	if p2.Mixed() != SamplesPerFrame {
		t.Errorf("a sinkless pump mixed %d samples, want %d", p2.Mixed(), SamplesPerFrame)
	}
}

func TestDiscardSink(t *testing.T) {
	d := &Discard{}
	e := New(fakeBank(10))
	p := NewPump(e, d)
	p.FrameTick()
	p.FrameTick()
	if got, want := d.Samples(), int64(2*SamplesPerFrame); got != want {
		t.Errorf("Discard counted %d samples, want %d", got, want)
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestPumpLatchesSinkError checks that a failing sink is recorded and then tolerated: audio
// failure must never stop the game.
func TestPumpLatchesSinkError(t *testing.T) {
	e := New(fakeBank(10))
	bad := &failing{}
	p := NewPump(e, bad)
	p.FrameTick()
	p.FrameTick()

	if p.Err == nil {
		t.Fatal("the pump did not record the sink's error")
	}
	if bad.writes != 2 {
		t.Errorf("the sink was written %d times, want 2: a failed sink must not stop the pump", bad.writes)
	}
	if p.Mixed() != 2*SamplesPerFrame {
		t.Errorf("Mixed() = %d; mixing must continue after a sink error", p.Mixed())
	}
}

// TestEncodeReusesItsBuffer is a small allocation check on the one function on the hot path: it
// runs once per Write, which on the live path is thirty times a second for the whole session.
func TestEncodeReusesItsBuffer(t *testing.T) {
	buf := make([]byte, 0, 8)
	got := encode(buf, []int16{1, 2, 3, 4})
	if &got[0] != &buf[:1][0] {
		t.Error("encode allocated a new buffer when the old one was big enough")
	}
	if len(got) != 8 {
		t.Fatalf("encoded %d bytes, want 8", len(got))
	}
	// Little-endian, always, whatever the host is.
	if got[0] != 1 || got[1] != 0 {
		t.Errorf("first sample encoded as %v, want little-endian 01 00", got[:2])
	}

	grown := encode(got[:0], []int16{1, 2, 3, 4, 5})
	if len(grown) != 10 {
		t.Errorf("encoded %d bytes after growing, want 10", len(grown))
	}
}

func TestPlayersIsOrdered(t *testing.T) {
	// Not "a player exists" -- the build box has no sound card. Only that the table is
	// well formed, since a typo in an argument list is otherwise found by a person with
	// headphones on.
	for _, p := range players {
		if p.name == "" || len(p.args) == 0 || p.note == "" {
			t.Errorf("player %+v is incomplete", p)
		}
		var hasRate bool
		for _, a := range p.args {
			if a != "" && containsRate(a) {
				hasRate = true
			}
		}
		if !hasRate {
			t.Errorf("player %s has no %%r argument, so it would play at its own default rate", p.name)
		}
	}
	if _, err := OpenPipe("no-such-player"); err == nil {
		t.Error("OpenPipe accepted an unknown player")
	}
}

func containsRate(s string) bool {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '%' && s[i+1] == 'r' {
			return true
		}
	}
	return s == "%r"
}

// collect is a Sink that keeps everything, for the pump tests.
type collect struct{ samples []int16 }

func (c *collect) Write(s []int16) error { c.samples = append(c.samples, s...); return nil }
func (c *collect) Close() error          { return nil }

// failing is a Sink that always errors.
type failing struct{ writes int }

func (f *failing) Write(s []int16) error { f.writes++; return os.ErrClosed }
func (f *failing) Close() error          { return nil }
