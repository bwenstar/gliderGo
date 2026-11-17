// audio.js: globalThis.glidergoAudio, the page's half of internal/audio/device_js.go.
//
// A browser starts an AudioContext only inside a click or a key press, so boot.js calls unlock()
// from the one that starts the game; the game calls open() and write() whenever it likes after
// that. Samples written before the worklet has loaded are held, up to half a second of them, and
// handed over when it has.
(function () {
  "use strict";
  const RATE = 22255; // internal/audio.Rate, which open() checks
  let ctx = null, node = null, pending = [], pendingLen = 0;

  // What the report on exit and tools/webcheck read. posted and refused are this side's: every
  // sample written, and the ones there was no room for while the worklet loaded. played,
  // underruns, dropped and queued are the worklet's, as it last said.
  const stats = {
    state: "none", contextRate: 0, posted: 0, refused: 0,
    played: 0, underruns: 0, dropped: 0, queued: 0,
  };

  globalThis.glidergoAudio = {
    stats,

    unlock() {
      if (!ctx) {
        const AC = globalThis.AudioContext || globalThis.webkitAudioContext;
        if (!AC) { stats.state = "this browser has no Web Audio"; return; }
        // The game's own rate if the browser will run a context at it, which saves resampling;
        // the worklet resamples from whatever rate it is given.
        try { ctx = new AC({ sampleRate: RATE, latencyHint: "interactive" }); }
        catch (e) { ctx = new AC({ latencyHint: "interactive" }); }
        stats.contextRate = ctx.sampleRate;
        ctx.onstatechange = () => { stats.state = ctx.state; };
        if (!ctx.audioWorklet) { stats.state = "this browser has no AudioWorklet"; return; }
        ctx.audioWorklet.addModule("audio-worklet.js").then(() => {
          node = new AudioWorkletNode(ctx, "glider-out", {
            numberOfInputs: 0, numberOfOutputs: 1, outputChannelCount: [1],
            processorOptions: { rate: RATE },
          });
          node.port.onmessage = (e) => Object.assign(stats, e.data);
          node.connect(ctx.destination);
          for (const s of pending) node.port.postMessage(s, [s.buffer]);
          pending = []; pendingLen = 0;
        }).catch((e) => { stats.state = "the worklet did not load: " + e; });
      }
      if (ctx.state !== "running") ctx.resume();
      stats.state = ctx.state;
    },

    // open answers null, or why not.
    open(rate) { return rate === RATE ? null : "the page plays " + RATE + " Hz, not " + rate; },

    // write takes little-endian s16 mono and answers how many samples it kept.
    write(u8) {
      const s = new Int16Array(u8.buffer, u8.byteOffset, u8.byteLength >> 1);
      const n = s.length; // read before the transfer, after which the array is empty
      stats.posted += n;
      if (node) { node.port.postMessage(s, [s.buffer]); return n; }
      if (pendingLen + n > RATE / 2) { stats.refused += n; return 0; }
      pending.push(s); pendingLen += n;
      return n;
    },

    close() { if (node) { node.disconnect(); node = null; } },
  };
})();
