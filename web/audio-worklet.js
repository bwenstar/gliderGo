// audio-worklet.js: plays the game's 22255 Hz mono mix at the AudioContext's rate.
//
// The game mixes by the wall clock and the sound card plays by its own, so this holds a short
// jitter buffer (internal/audio/device_js.go says why): it starts once 50 ms are queued, goes
// quiet and waits for 50 ms again when it runs dry, and drops the oldest samples past 200 ms so
// that the latency stays bounded. Resampling is linear, which is plenty for 1994's 8-bit sounds.
class GliderOut extends AudioWorkletProcessor {
  constructor(opts) {
    super();
    this.rate = opts.processorOptions.rate;
    this.step = this.rate / sampleRate;
    this.buf = new Float32Array(1 << 15); // 1.47 s at 22255 Hz, far past the 200 ms cap
    this.mask = this.buf.length - 1;
    this.w = 0; // samples written, ever
    this.r = 0; // where playing has got to, fractionally
    this.prebuffer = this.rate * 0.05;
    this.cap = this.rate * 0.2;
    this.running = false;
    this.played = 0;
    this.underruns = 0;
    this.dropped = 0;
    this.quanta = 0;
    this.port.onmessage = (e) => {
      const s = e.data;
      for (let i = 0; i < s.length; i++) this.buf[(this.w + i) & this.mask] = s[i] / 32768;
      this.w += s.length;
      const over = this.w - this.r - this.cap;
      if (over > 0) { this.r += over; this.dropped += Math.round(over); }
    };
  }

  process(inputs, outputs) {
    const out = outputs[0][0];
    if (!this.running && this.w - this.r >= this.prebuffer) this.running = true;
    for (let i = 0; i < out.length; i++) {
      if (this.running && this.w - this.r >= 2) {
        const i0 = Math.floor(this.r), f = this.r - i0;
        const a = this.buf[i0 & this.mask], b = this.buf[(i0 + 1) & this.mask];
        out[i] = a + (b - a) * f;
        this.r += this.step;
        this.played += this.step;
      } else {
        if (this.running) { this.running = false; this.underruns++; }
        out[i] = 0;
      }
    }
    // About three times a second at 44.1 kHz: often enough for the report, rarely enough to cost
    // nothing.
    if (++this.quanta % 100 === 0) {
      this.port.postMessage({
        played: Math.round(this.played), underruns: this.underruns,
        dropped: this.dropped, queued: Math.round(this.w - this.r),
      });
    }
    return true;
  }
}
registerProcessor("glider-out", GliderOut);
