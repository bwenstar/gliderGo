// boot.js: fetch and compile the game while the cover is up, run it on a click or a key, and say
// what happened when it stops.
//
// Two query parameters, for testing: ?arg=-x&arg=y appends command-line arguments (tools/webcheck
// asks for -frames and -dump this way), and ?wasm=name.wasm loads another binary from beside this
// page. Both are honoured only on a page served from this machine. Anywhere else a link is all a
// stranger needs to make one, and the game's flags can write files: `-shot` over a saved game,
// `-prefs` moving one aside. And the binary only ever from beside the page, because a URL there
// would run anybody's code with this site's storage, which on github.io is every one of this
// account's projects.
(function () {
  "use strict";
  const build = globalThis.glidergoBuild || { version: "", source: "", wasmBytes: 0 };
  const local = ["localhost", "127.0.0.1", "[::1]"].includes(location.hostname);
  const params = new URLSearchParams(local ? location.search : "");
  if (!local && location.search) console.warn("glidergo: the query is for testing, on a page served from this machine, and is ignored here");
  const cover = document.getElementById("cover");
  const play = document.getElementById("play");
  const note = document.getElementById("note");
  const report = document.getElementById("report");

  // Where the game's files live, and so what it calls them on screen: "settings saved in
  // /browser storage/prefs.json" says where they are to somebody who has never heard of
  // localStorage. One directory for all of it, as datadir does for a GLIDERGO_CONFIG.
  const STORAGE = "/browser storage";

  if (build.version) document.getElementById("version").textContent = "gliderGo " + build.version;
  if (build.source) document.getElementById("source").href = build.source;

  function say(text) { note.textContent = text; }

  // Keyboard only, as in 1994. A device with nothing but a touch screen is told so, and can still
  // try: a tablet with a keyboard attached reports the same pointer.
  if (matchMedia("(hover: none) and (pointer: coarse)").matches) {
    say("gliderGo is played with a keyboard, and this device may not have one.");
  }

  let wasm = "glidergo.wasm";
  if (params.has("wasm")) {
    wasm = params.get("wasm");
    if (!/^[A-Za-z0-9_-]+\.wasm$/.test(wasm)) {
      play.textContent = "Not loaded";
      say("?wasm= names a file beside this page, such as glidergo.wasm, and not " + JSON.stringify(wasm) + ".");
      return;
    }
  }

  const go = new Go();
  go.argv = ["glidergo"].concat(params.getAll("arg"));
  go.env = { GLIDERGO_CONFIG: STORAGE };
  go.exit = (code) => { globalThis.glidergoExit = code; };
  globalThis.glidergoGo = go;

  // The download, with its progress on the button. The binary is the game and every 1994 picture
  // and sound inside it, about 19 MB, and a page that says "Loading" for that long without a
  // number looks broken. build.js knows its size; the server may be sending it compressed, and a
  // byte count of the compressed stream measured against the real size would stop short of 100.
  async function fetchWasm(url) {
    const r = await fetch(url);
    if (!r.ok) throw new Error(url + ": " + r.status + " " + r.statusText);
    let body = r.body;
    if (body && build.wasmBytes > 0) {
      let got = 0;
      body = body.pipeThrough(new TransformStream({
        transform(chunk, ctrl) {
          got += chunk.length;
          play.textContent = "Loading… " + Math.min(99, Math.floor(100 * got / build.wasmBytes)) + "%";
          ctrl.enqueue(chunk);
        },
      }));
    }
    // compileStreaming compiles while it downloads, and wants the type a server may not send; the
    // fallback is the whole file first and then the compile.
    if (WebAssembly.compileStreaming && body) {
      return WebAssembly.compileStreaming(new Response(body, { headers: { "Content-Type": "application/wasm" } }));
    }
    return WebAssembly.compile(await new Response(body).arrayBuffer());
  }

  const t0 = performance.now();
  const compiled = fetchWasm(wasm).then((m) => {
    globalThis.glidergoCompileMs = performance.now() - t0;
    play.textContent = "Play";
    play.disabled = false;
    play.focus();
    return m;
  });
  compiled.catch((e) => {
    play.textContent = "Not loaded";
    say("This browser could not load the game: " + e);
  });

  // The report a bug wants: everything the game printed on standard error, which is where a panic
  // goes, and the head of its crash.log, which names the build and the machine. A native build
  // keeps the panic in crash.log as well (cmd/glidergo/crash.go), but that takes a second file
  // descriptor for the same file and js/wasm has no dup. Both are inside the page and nowhere a
  // player could find them, so they are put on the cover to be copied.
  function showReport() {
    const lines = globalThis.glidergoErr.slice(-200);
    const log = globalThis.glidergoFS.nodes.get(STORAGE + "/crash.log");
    let text = lines.join("\n");
    if (log && log.data) text += (text ? "\n\n" : "") + new TextDecoder().decode(log.data);
    if (!text) return;
    document.getElementById("report-text").textContent = text;
    report.hidden = false;
  }

  // The line to lead with: the runtime's own first line if it died, else the last thing it said.
  function cause() {
    const err = globalThis.glidergoErr;
    return err.find((l) => /^(panic|fatal error): /.test(l)) || err[err.length - 1] || "";
  }

  function stopped(code, err) {
    globalThis.glidergoRunning = false;
    // Whatever was written and not yet closed, which is what a run that died leaves behind.
    globalThis.glidergoFS.persist();
    cover.hidden = false;
    play.textContent = "Play again";
    play.disabled = false;
    play.onclick = () => location.reload();
    play.focus();
    if (err || code !== 0) {
      say("gliderGo stopped with an error: " + (err || cause() || "exit status " + code));
      showReport();
    } else if (globalThis.glidergoFS.state.failed) {
      say("Thanks for playing. This browser would not keep the game's files, so nothing from this " +
          "session was saved: " + globalThis.glidergoFS.state.failed);
    } else {
      say("Thanks for playing.");
    }
  }

  let begun = false;
  async function begin() {
    if (begun) return;
    begun = true;
    // Inside the click or the key press, which is the only place it works. A page whose audio.js
    // did not load plays in silence, as a machine with no sound output does.
    if (globalThis.glidergoAudio) glidergoAudio.unlock();
    // Ask that the files be kept when the disk is short of space. A browser may say no, or ask
    // the player; either way the game runs.
    if (navigator.storage && navigator.storage.persist) navigator.storage.persist().catch(() => {});
    let inst;
    try {
      inst = await WebAssembly.instantiate(await compiled, go.importObject);
    } catch (e) {
      stopped(1, e);
      return;
    }
    cover.hidden = true;
    note.textContent = "";
    document.getElementById("glidergo").focus();
    globalThis.glidergoRunning = true;
    try {
      await go.run(inst);
    } catch (e) {
      stopped(1, e);
      return;
    }
    stopped(globalThis.glidergoExit === undefined ? 0 : globalThis.glidergoExit);
  }

  // The button, and Space or Return anywhere while the cover is up: a focused button takes both
  // already, and this is for a click that moved the focus somewhere else first.
  play.onclick = begin;
  addEventListener("keydown", (e) => {
    if (cover.hidden || play.disabled || begun) return;
    if (e.code === "Space" || e.code === "Enter" || e.code === "NumpadEnter") {
      e.preventDefault();
      play.click();
    }
  });
})();
