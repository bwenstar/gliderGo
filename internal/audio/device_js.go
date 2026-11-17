//go:build js && wasm

package audio

// The browser's output: Web Audio, through the page.
//
// A page has no sound device to open and no program to pipe to, and it may not start an
// AudioContext at all until the player has clicked or pressed a key. So the page owns the context
// -- web/audio.js makes it inside the click that starts the game -- and this side only hands it
// PCM: little-endian s16 mono at Rate, which an AudioWorklet (web/audio-worklet.js) holds in a
// short jitter buffer and resamples to whatever rate the context runs at.
//
// The buffer is there because two clocks are involved. The mixer is paced by presents (pump.go's
// ClockTick), which is to say by the wall clock, and the context plays by the sound card's. The
// worklet starts once 50 ms are queued, goes quiet and waits for 50 ms again when it runs dry, and
// drops the oldest past 200 ms, so that a stall is heard as a gap and the latency stays bounded --
// the bargain maxCatchUp makes on the mixer's side.

import (
	"errors"
	"syscall/js"
)

const deviceName = "webaudio"

// haveDevice reports whether the page has the other half: web/audio.js's glidergoAudio.
func haveDevice() bool { return js.Global().Get("glidergoAudio").Truthy() }

func openDevice() (Stream, error) {
	api := js.Global().Get("glidergoAudio")
	if !api.Truthy() {
		return nil, errNoDevice
	}
	// open answers null, or why not.
	if why := api.Call("open", Rate); why.Truthy() {
		return nil, errors.New("webaudio: " + why.String())
	}
	return &webAudio{api: api, u8: js.Global().Get("Uint8Array")}, nil
}

// webAudio is the Stream: one message to the worklet per Write.
type webAudio struct {
	api, u8 js.Value
	b       []byte
	samples int64
	refused int64
}

// Write hands the samples to the page. It keeps every one once the worklet is running, and half a
// second of them while it is still loading; the rest are refused and counted.
func (w *webAudio) Write(s []int16) error {
	n := 2 * len(s)
	if cap(w.b) < n {
		w.b = make([]byte, n)
	}
	b := w.b[:n]
	for i, v := range s {
		b[2*i], b[2*i+1] = byte(v), byte(v>>8)
	}
	// A new array every time, because the page transfers it to the worklet rather than copying
	// it, and a transferred buffer is empty on this side afterwards.
	arr := w.u8.New(n)
	js.CopyBytesToJS(arr, b)
	took := w.api.Call("write", arr).Int()
	w.samples += int64(took)
	w.refused += int64(len(s) - took)
	return nil
}

func (w *webAudio) Close() error   { w.api.Call("close"); return nil }
func (w *webAudio) Name() string   { return deviceName }
func (w *webAudio) Samples() int64 { return w.samples }
func (w *webAudio) Err() error     { return nil }

// Dropped is what never played: what the page refused, and what the worklet threw away to keep
// its buffer under 200 ms, as it last reported.
func (w *webAudio) Dropped() int64 { return w.refused + w.stat("dropped") }

// Underruns is how many times the worklet ran dry, as it last reported: WaveOut's count, for the
// same line in the report on exit.
func (w *webAudio) Underruns() int64 { return w.stat("underruns") }

func (w *webAudio) stat(name string) int64 {
	v := w.api.Get("stats").Get(name)
	if v.Type() != js.TypeNumber {
		return 0
	}
	return int64(v.Int())
}
