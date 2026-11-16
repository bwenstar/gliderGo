// Command webcheck plays the browser build in headless Chrome and holds it to the native one.
//
//	go run ./tools/webcheck -site bin/web -null-wasm bin/webcheck/glidergo-null.wasm \
//	    -native bin/webcheck/glidergo-null     # what `make web-check` runs
//
// # Why
//
// The browser backend's own tests (internal/platform/web) run on any host, because the half of
// it that decides things has no build tag. The other half is syscall/js, and the page around it
// is JavaScript, and neither runs anywhere but a browser. This is that browser: it serves -site
// the way a static host does, opens it in Chrome with no window, and plays.
//
// # What it checks
//
//   - **The title screen.** The page loads, the key that starts it starts it, and the canvas
//     comes to hold the pixels the native build's -shot draws of the same screen: all of them but
//     the Race... row, which a browser greys because it cannot race.
//   - **The game.** 120 frames of the null backend built for js/wasm, dumped into the page's own
//     storage, are byte for byte the native build's 120. This is the claim that the browser plays
//     the same game: the same simulation, the same composition, the same PNG encoder.
//   - **The keyboard and the storage.** A game started from the keyboard, paused with Tab and
//     saved with S leaves its save in localStorage, and after a reload O resumes it. Sound runs
//     through Web Audio meanwhile.
//   - **The room wipes.** A paced game steered across rooms never leaves the page more than
//     wipeBudget without a paint (docs/IMPROVEMENTS.md 3.10). It is the one check that times
//     anything, and its limit is set far above what a fixed build takes so that a slow runner
//     does not fail it: it is there for a regression back to a frozen page, which took half a
//     second here.
//   - **The stored files.** web/fs.js, through the calls Go's runtime makes: a change keeps an
//     entry it could not read and another tab's files, a store the browser refuses is an error
//     for the call that needed it, and a tree that is not a tree is set aside, not written over.
//   - **A stranger's link.** Reached by a name that is not this machine's, as on github.io, the
//     page ignores ?arg=, which can name flags that write files.
//
// # Where it runs
//
// Chrome gets a temporary profile and home of its own and no display (chrome.go), and the native
// runs get a temporary GLIDERGO_CONFIG and GLIDERGO_DATA, so that nothing here reads or writes
// anything of the user's. Everything temporary is removed afterwards; -out keeps what a failure
// needs looked at, and -shots keeps screenshots of the cover and the title screen as well.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bwenstar/gliderGo/internal/cliargs"
)

// wipeBudget is the longest the page may go without a paint during play. A frame is 33 ms and a
// paint 16.7; a room's wipe once took 483 ms on the machine this was written on.
const wipeBudget = 250 * time.Millisecond

// dumpFrames is how many frames the game check compares. -frames 120 dumps 122 PNGs, because a
// dump is a PNG a present and a game presents twice as it opens, before its first frame.
const dumpFrames = 120

func main() {
	var cfg config
	flag.StringVar(&cfg.site, "site", "bin/web", "the browser build, as `make web` writes it")
	flag.StringVar(&cfg.nullWasm, "null-wasm", "", "the game built for js/wasm with -tags nullbackend")
	flag.StringVar(&cfg.native, "native", "", "the game built for this machine with -tags nullbackend")
	flag.StringVar(&cfg.chrome, "chrome", "", "the Chrome to run; empty finds one on PATH")
	flag.StringVar(&cfg.out, "out", "", "a directory for what a failed check leaves to be looked at")
	flag.BoolVar(&cfg.shots, "shots", false, "keep screenshots of the cover and the title screen in -out, pass or fail")
	// Every flag takes a value or is a switch, and there are no positional arguments; FlagsFirst
	// is called all the same, as every command here calls it (internal/cliargs).
	flag.CommandLine.Parse(cliargs.FlagsFirst(flag.CommandLine, os.Args[1:]))
	if cfg.nullWasm == "" || cfg.native == "" || flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "usage: webcheck -site DIR -null-wasm FILE -native FILE [-chrome PATH] [-out DIR [-shots]]")
		os.Exit(2)
	}
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "webcheck: %v\n", err)
		os.Exit(1)
	}
}

type config struct {
	site, nullWasm, native, chrome, out string
	shots                               bool
}

// env is what every check is given.
type env struct {
	cfg  config
	b    *browser
	base string // the server's URL, with no trailing slash
	tmp  string
}

func run(cfg config) error {
	if _, err := os.Stat(filepath.Join(cfg.site, "index.html")); err != nil {
		return fmt.Errorf("%s has no index.html: run `make web` first", cfg.site)
	}
	chrome, err := findChrome(cfg.chrome)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "glidergo-webcheck-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	srv, base, err := serve(cfg.site, map[string]string{"/glidergo-null.wasm": cfg.nullWasm})
	if err != nil {
		return err
	}
	defer srv.Close()

	b, err := startChrome(chrome, filepath.Join(tmp, "chrome"), false)
	if err != nil && sandboxRefused(filepath.Join(tmp, "chrome", "chrome.log")) {
		fmt.Println("webcheck: this machine refuses Chrome's sandbox, as its log says, so it runs " +
			"with --no-sandbox; the only page it opens is this build's own, from loopback")
		b, err = startChrome(chrome, filepath.Join(tmp, "chrome-no-sandbox"), true)
	}
	if err != nil {
		return err
	}
	defer b.close()
	fmt.Printf("webcheck: %s, serving %s at %s\n", b.version(), cfg.site, base)

	e := &env{cfg: cfg, b: b, base: base, tmp: tmp}
	checks := []struct {
		name string
		f    func(*env) (string, error)
	}{
		{"title screen", checkTitle},
		{"the game", checkFrames},
		{"keyboard, storage and sound", checkSave},
		{"room wipes", checkWipes},
		{"the stored files", checkStorage},
		{"a stranger's link", checkQuery},
	}
	failed := 0
	for _, c := range checks {
		msg, err := c.f(e)
		if err != nil {
			failed++
			fmt.Printf("webcheck: FAIL %s: %v\n", c.name, err)
			continue
		}
		fmt.Printf("webcheck: ok   %s: %s\n", c.name, msg)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d checks failed", failed, len(checks))
	}
	fmt.Printf("webcheck: all %d checks passed\n", len(checks))
	return nil
}

// serve is the static host: root's files, plus extra names mapped to files outside it, on a
// loopback port of the kernel's choosing.
//
// It sends application/wasm by name rather than trusting the platform's MIME table, since
// compileStreaming refuses anything else and that table is the machine's own. No caching, so
// that a run never sees an earlier build's files.
func serve(root string, extra map[string]string) (*http.Server, string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	files := http.FileServer(http.Dir(root))
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if strings.HasSuffix(r.URL.Path, ".wasm") {
			w.Header().Set("Content-Type", "application/wasm")
		}
		if f, ok := extra[r.URL.Path]; ok {
			http.ServeFile(w, r, f)
			return
		}
		files.ServeHTTP(w, r)
	})}
	go srv.Serve(ln) // the one goroutine this command starts; Close ends it
	return srv, "http://" + ln.Addr().String(), nil
}

// native runs the native build in a configuration directory of its own.
func (e *env) native(args ...string) error {
	dir := filepath.Join(e.tmp, "native")
	cmd := exec.Command(e.cfg.native, args...)
	var env []string
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); k != "DISPLAY" && !strings.HasPrefix(k, "GLIDERGO_") {
			env = append(env, kv)
		}
	}
	cmd.Env = append(env, "GLIDERGO_CONFIG="+filepath.Join(dir, "config"), "GLIDERGO_DATA="+filepath.Join(dir, "data"))
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %v\n%s", e.cfg.native, strings.Join(args, " "), err, out.String())
	}
	return nil
}

// start opens the page with a query, waits for it to compile and presses the key that starts it.
// before runs first, for a check that has something to set on the page before the game sees it.
func (e *env) start(query string, before string) (*page, error) {
	p, err := e.b.newPage()
	if err != nil {
		return nil, err
	}
	started := false
	defer func() {
		if !started {
			p.close()
		}
	}()
	if err := p.open(e.base + "/" + query); err != nil {
		return nil, err
	}
	if err := p.until("globalThis.glidergoCompileMs !== undefined", 120*time.Second); err != nil {
		return nil, e.why(p, err)
	}
	if query == "" {
		e.shoot(p, "cover.png")
	}
	if before != "" {
		if err := p.eval(before, nil); err != nil {
			return nil, err
		}
	}
	if err := p.tap("Space"); err != nil {
		return nil, err
	}
	if err := p.until("globalThis.glidergoRunning === true || globalThis.glidergoExit !== undefined", 30*time.Second); err != nil {
		return nil, e.why(p, err)
	}
	started = true
	return p, nil
}

// why adds what the page said to an error, which is usually the reason for it.
func (e *env) why(p *page, err error) error {
	var lines []string
	p.eval("globalThis.glidergoLog || []", &lines)
	lines = append(lines, p.console()...)
	if len(lines) > 12 {
		lines = lines[len(lines)-12:]
	}
	if len(lines) == 0 {
		return err
	}
	return fmt.Errorf("%v; the page said:\n    %s", err, strings.Join(lines, "\n    "))
}

// ---------------------------------------------------------------------------- the checks

func checkTitle(e *env) (string, error) {
	shot := filepath.Join(e.tmp, "title.png")
	if err := e.native("-shot", shot, "-sound=false", "-prefs", "none", "-quiet"); err != nil {
		return "", err
	}
	want, w, h, err := readRGBA(shot)
	if err != nil {
		return "", err
	}
	p, err := e.start("?arg=-prefs&arg=none", "")
	if err != nil {
		return "", err
	}
	defer p.close()
	// The canvas is blank until the first present, and the first present is the title screen, so
	// it is read until it matches or there has been time for it to have done.
	//
	// One difference is meant. A browser cannot race, so its Race... row is greyed and the native
	// one is not (cmd/glidergo's cannotRace), and the two screens must differ there and nowhere
	// else: inside one menu row, which is the browser's half of that row's test.
	deadline := time.Now().Add(15 * time.Second)
	for {
		got, err := p.canvas()
		if err != nil {
			return "", err
		}
		d := compare(got, want, w, h)
		if d.n > 0 && d.y1-d.y0 < menuPitch && d.x1-d.x0 < raceWide {
			e.shoot(p, "title.png")
			return fmt.Sprintf("the canvas is the native -shot in all but %d of %d pixels, and those "+
				"are the Race... row, greyed (rows %d-%d)", d.n, w*h, d.y0, d.y1), nil
		}
		if time.Now().After(deadline) {
			e.keep("title-browser.png", got, w, h)
			e.keep("title-native.png", want, w, h)
			if d.n == 0 {
				return "", fmt.Errorf("the canvas is the native -shot exactly, so the Race... row " +
					"is not greyed, and a browser cannot race")
			}
			return "", e.why(p, fmt.Errorf("the canvas differs from the native -shot in %s, which is "+
				"more than the Race... row", d))
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// The Race... row's size, from internal/shell/screens.go: menuPitch rows apart, and seven
// characters of 12 pixels at menuScale 2.
const (
	menuPitch = 21
	raceWide  = 7*12 + 4
)

func checkFrames(e *env) (string, error) {
	// Unpaced, on both sides: -bench changes how the wait between frames is spent and nothing
	// about the frames (cmd/glidergo's five hooks), and it is the difference between four seconds
	// and a fifth of one. The page answers nothing while an unpaced run is going, since the run
	// never sleeps, so the wait below is one long call rather than a poll.
	dir := filepath.Join(e.tmp, "frames")
	n := fmt.Sprint(dumpFrames)
	if err := e.native("-bench", "-frames", n, "-sound=false", "-dump", dir, "-quiet"); err != nil {
		return "", err
	}
	want, err := hashDir(dir)
	if err != nil {
		return "", err
	}
	// glidergoKeepNothing: 122 PNGs are 6 MB, more than localStorage holds, and writing the whole
	// tree after each of them would be the slowest part of the run.
	p, err := e.start("?wasm=glidergo-null.wasm&arg=-bench&arg=-frames&arg="+n+"&arg=-sound%3Dfalse&arg=-dump&arg=%2Fdump&arg=-quiet",
		"globalThis.glidergoKeepNothing = true")
	if err != nil {
		return "", err
	}
	defer p.close()
	if err := p.until("globalThis.glidergoExit !== undefined", 300*time.Second); err != nil {
		return "", e.why(p, err)
	}
	var code int
	if err := p.eval("globalThis.glidergoExit", &code); err != nil {
		return "", err
	}
	if code != 0 {
		return "", e.why(p, fmt.Errorf("the run exited %d", code))
	}
	var got []string
	if err := p.eval(`(async () => {
		const out = [];
		for (const [path, n] of glidergoFS.nodes) {
			if (!path.startsWith("/dump/") || n.dir) continue;
			const h = new Uint8Array(await crypto.subtle.digest("SHA-256", n.data));
			out.push([...h].map((b) => b.toString(16).padStart(2, "0")).join("") + "  " + path.slice(6));
		}
		return out.sort((a, b) => a.slice(66).localeCompare(b.slice(66)));
	})()`, &got); err != nil {
		return "", err
	}
	if len(got) != len(want) {
		return "", fmt.Errorf("the browser dumped %d frames and the native build %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			return "", fmt.Errorf("frame %d differs: native %s, browser %s (and maybe others)",
				i, want[i][:12], got[i][:12])
		}
	}
	return fmt.Sprintf("all %d frame PNGs are byte for byte the native build's", len(want)), nil
}

func checkSave(e *env) (string, error) {
	p, err := e.start("", "")
	if err != nil {
		return "", err
	}
	defer p.close()
	const save = "/browser storage/saves/Slumberland.save"
	started := `glidergoLog.some((l) => /^glidergo: Slumberland -- .* rooms, first/.test(l))`
	// Each key, then what it is waiting for: a line the game prints or a file it writes, where
	// there is one, and a pause long enough for a frame or two where there is not.
	steps := []struct {
		key, until string
		pause      time.Duration
	}{
		{"", "document.title.includes('Slumberland')", 500 * time.Millisecond}, // the title screen
		{"KeyN", started, 0},                                                               // New Game, and its banner
		{"Space", "", time.Second},                                                         // past the banner, flying
		{"Tab", "", 500 * time.Millisecond},                                                // pause
		{"KeyS", "glidergoFS.nodes.has(" + jsString(save) + ")", 0},                        // save
		{"KeyQ", "", 500 * time.Millisecond},                                               // give up...
		{"KeyN", "glidergoLog.some((l) => / frames in /.test(l))", 500 * time.Millisecond}, // ...without saving again
	}
	for _, s := range steps {
		if s.key != "" {
			if err := p.tap(s.key); err != nil {
				return "", err
			}
		}
		if s.until != "" {
			if err := p.until(s.until, 15*time.Second); err != nil {
				return "", e.why(p, fmt.Errorf("after %s: %v", s.key, err))
			}
		}
		time.Sleep(s.pause)
	}
	var size int
	if err := p.eval(`(() => {
		const o = JSON.parse(localStorage.getItem(glidergoFS.key) || "{}");
		const n = o[`+jsString(save)+`];
		return n ? atob(n.data).length : 0;
	})()`, &size); err != nil {
		return "", err
	}
	if size == 0 {
		return "", e.why(p, fmt.Errorf("S in the pause left no %s in localStorage", save))
	}
	var audio struct {
		State     string
		Played    int
		Underruns int
	}
	if err := p.eval("glidergoAudio.stats", &audio); err != nil {
		return "", err
	}
	if audio.State != "running" || audio.Played == 0 {
		return "", fmt.Errorf("the sound did not run: the AudioContext is %q and %d samples played",
			audio.State, audio.Played)
	}

	// The next visit: the same storage, a new page. O is Open Saved Game..., and a game
	// starting is the proof that the shell offered the save and the game read it.
	if err := p.open(e.base + "/"); err != nil {
		return "", err
	}
	if err := p.until("globalThis.glidergoCompileMs !== undefined", 120*time.Second); err != nil {
		return "", err
	}
	if err := p.tap("Space"); err != nil {
		return "", err
	}
	if err := p.until("document.title.includes('Slumberland')", 15*time.Second); err != nil {
		return "", e.why(p, err)
	}
	time.Sleep(500 * time.Millisecond)
	if err := p.tap("KeyO"); err != nil {
		return "", err
	}
	if err := p.until(started, 10*time.Second); err != nil {
		return "", e.why(p, fmt.Errorf("O after a reload did not resume the saved game: %v", err))
	}
	return fmt.Sprintf("a %d-byte save outlived the page and resumed; sound ran, %d samples played, %d underruns",
		size, audio.Played, audio.Underruns), nil
}

// wipeFrames is how long the wipe check flies: long enough to cross into the next room and back.
const wipeFrames = 240

func checkWipes(e *env) (string, error) {
	// Two things are watched from the page's side: the time between paints, and the presents
	// that changed most of the canvas at once, which in play is a room's wipe and nothing else.
	// The second is the proof that the first covered one.
	p, err := e.start("?arg=-frames&arg="+fmt.Sprint(wipeFrames), `(() => {
		globalThis.paints = [];
		globalThis.wipes = 0;
		let last = 0;
		const tick = (t) => {
			if (last) globalThis.paints.push(t - last);
			last = t;
			if (globalThis.glidergoExit === undefined) requestAnimationFrame(tick);
		};
		requestAnimationFrame(tick);
		const ctx = document.getElementById("glidergo").getContext("2d");
		const put = ctx.putImageData.bind(ctx);
		ctx.putImageData = (img, dx, dy, x, y, w, h) => {
			if (w * h > 640 * 480 / 2) globalThis.wipes++;
			return put(img, dx, dy, x, y, w, h);
		};
	})()`)
	if err != nil {
		return "", err
	}
	defer p.close()
	// Out of the first room to the right and back, which is a wipe each way. The count starts
	// after the game's own opening, whose banner and first room are whole screens too.
	time.Sleep(500 * time.Millisecond)
	if err := p.eval("globalThis.wipes = 0", nil); err != nil {
		return "", err
	}
	for _, h := range []struct {
		key string
		d   time.Duration
	}{{"ArrowRight", 3 * time.Second}, {"ArrowLeft", 3 * time.Second}} {
		if err := p.hold(h.key, h.d); err != nil {
			return "", err
		}
	}
	if err := p.until("globalThis.glidergoExit !== undefined", 120*time.Second); err != nil {
		return "", e.why(p, err)
	}
	var r struct {
		N     int
		Worst float64
		Rate  string
		Wipes int
	}
	if err := p.eval(`(() => {
		const a = globalThis.paints.slice(5);
		const line = glidergoLog.find((l) => / frames in /.test(l)) || "";
		const m = line.match(/\(([0-9.]+) fps\)/);
		return {n: a.length, worst: Math.max(0, ...a), rate: m ? m[1] : "?", wipes: globalThis.wipes};
	})()`, &r); err != nil {
		return "", err
	}
	worst := time.Duration(r.Worst * float64(time.Millisecond))
	msg := fmt.Sprintf("%d frames at %s fps and %d rooms' wipes; the slowest of %d paint gaps was %v",
		wipeFrames, r.Rate, r.Wipes, r.N, worst.Round(100*time.Microsecond))
	switch {
	case r.N < 100:
		return "", fmt.Errorf("%s, which is too few paints to say anything", msg)
	case worst > wipeBudget:
		return "", fmt.Errorf("%s, over the %v this allows", msg, wipeBudget)
	case r.Wipes == 0:
		// Counted only once a wipe reaches the canvas whole, which coalescing is what does; a
		// wipe sent strip by strip is the stall above.
		return "", fmt.Errorf("%s, and without a wipe that measured nothing", msg)
	}
	return msg, nil
}

// checkStorage holds web/fs.js to what it promises, through the same fs calls Go's runtime makes:
// that only what changed is written, over the tree as it is stored now, so that one entry it cannot
// read and another tab's files both survive a change; that a store the browser refuses is an error
// for the call that needed it, and not a save that silently vanishes; and that a tree that is not
// a tree at all is set aside, not written over.
func checkStorage(e *env) (string, error) {
	p, err := e.b.newPage()
	if err != nil {
		return "", err
	}
	defer p.close()
	if err := p.open(e.base + "/"); err != nil {
		return "", err
	}
	const tree = `{"/browser storage":{"dir":1,"mtime":1},` +
		`"/browser storage/bad":{"data":"%%%","mode":420,"mtime":1},` +
		`"/browser storage/good":{"data":"aGVsbG8=","mode":420,"mtime":1}}`
	if err := p.eval("localStorage.setItem(glidergoFS.key, "+jsString(tree)+")", nil); err != nil {
		return "", err
	}
	if err := p.open(e.base + "/"); err != nil { // fs.js reads the tree as it loads
		return "", err
	}

	var kept struct {
		Bad               string
		Good, Other, Made bool
	}
	if err := p.eval(`(async () => {
		const key = glidergoFS.key;
		const other = JSON.parse(localStorage.getItem(key));
		other["/browser storage/other"] = {data: btoa("another tab"), mode: 420, mtime: 2};
		localStorage.setItem(key, JSON.stringify(other));
		await new Promise((ok, no) => fs.mkdir("/browser storage/made", 0o755, (e) => e ? no(e) : ok()));
		const o = JSON.parse(localStorage.getItem(key));
		const f = (p) => !!o["/browser storage/" + p];
		return {bad: f("bad") ? o["/browser storage/bad"].data : "", good: f("good"), other: f("other"), made: f("made")};
	})()`, &kept); err != nil {
		return "", err
	}
	if kept.Bad != "%%%" || !kept.Good || !kept.Other || !kept.Made {
		return "", fmt.Errorf("after a change the stored tree has bad=%q good=%v another tab's=%v "+
			"the change=%v; want the unreadable entry as it was and the other three there",
			kept.Bad, kept.Good, kept.Other, kept.Made)
	}

	var full struct {
		Code string
		Full bool
	}
	if err := p.eval(`(async () => {
		const fill = [];
		for (let n = 1 << 20; n >= 16; n >>= 1) {
			const s = "x".repeat(n);
			try { for (;;) { const k = "glidergo:test:fill" + fill.length; localStorage.setItem(k, s); fill.push(k); } }
			catch (e) { /* this size no longer fits */ }
		}
		const code = await new Promise((ok) => fs.mkdir("/browser storage/full", 0o755, (e) => ok(e ? e.code : "")));
		for (const k of fill) localStorage.removeItem(k);
		return {code, full: glidergoFS.state.full};
	})()`, &full); err != nil {
		return "", err
	}
	if full.Code != "ENOSPC" || !full.Full {
		return "", fmt.Errorf("a full store answered the call that needed it with %q (full=%v), "+
			"want ENOSPC, which Go reports as a full disk", full.Code, full.Full)
	}

	if err := p.eval("localStorage.setItem(glidergoFS.key, 'not a tree')", nil); err != nil {
		return "", err
	}
	if err := p.open(e.base + "/"); err != nil {
		return "", err
	}
	var aside struct {
		Aside string
		Main  *string
	}
	if err := p.eval(`({aside: localStorage.getItem(glidergoFS.key + ":unreadable"), main: localStorage.getItem(glidergoFS.key)})`, &aside); err != nil {
		return "", err
	}
	if aside.Aside != "not a tree" || aside.Main != nil {
		return "", fmt.Errorf("an unreadable tree left %q set aside and %v in place; want it moved "+
			"aside whole", aside.Aside, aside.Main)
	}
	return "a change keeps an unreadable entry and another tab's files, a full store is ENOSPC, " +
		"and an unreadable tree is set aside", nil
}

// checkQuery: the page honours ?arg= only when it is served from this machine. Reached by any
// other name, as a stranger's link would reach it on github.io, a query that would have written a
// screenshot over a file starts the game as if it were not there.
func checkQuery(e *env) (string, error) {
	_, port, err := net.SplitHostPort(strings.TrimPrefix(e.base, "http://"))
	if err != nil {
		return "", err
	}
	p, err := e.b.newPage()
	if err != nil {
		return "", err
	}
	defer p.close()
	target := "/browser storage/from-a-link.png"
	if err := p.open("http://" + strangerHost + ":" + port + "/?arg=-shot&arg=" +
		strings.ReplaceAll(target, " ", "%20")); err != nil {
		return "", err
	}
	if err := p.until("globalThis.glidergoCompileMs !== undefined", 120*time.Second); err != nil {
		return "", e.why(p, err)
	}
	if err := p.tap("Space"); err != nil {
		return "", err
	}
	if err := p.until("globalThis.glidergoRunning === true || globalThis.glidergoExit !== undefined", 30*time.Second); err != nil {
		return "", e.why(p, err)
	}
	time.Sleep(time.Second)
	var r struct {
		Running bool
		Wrote   bool
	}
	if err := p.eval(`({running: globalThis.glidergoExit === undefined, wrote: glidergoFS.nodes.has(`+jsString(target)+`)})`, &r); err != nil {
		return "", err
	}
	if !r.Running || r.Wrote {
		return "", e.why(p, fmt.Errorf("from %s a query naming -shot left the game running=%v, "+
			"and the screenshot written=%v; want it ignored", strangerHost, r.Running, r.Wrote))
	}
	return "reached as " + strangerHost + ", a query naming -shot is ignored and the game starts", nil
}

// ---------------------------------------------------------------------------- pixels and files

// diff is where two images disagree.
type diff struct{ n, x0, y0, x1, y1 int }

func (d diff) String() string {
	return fmt.Sprintf("%d pixels, in rows %d-%d and columns %d-%d", d.n, d.y0, d.y1, d.x0, d.x1)
}

// compare counts the pixels whose colour differs between two RGBA images of w x h, ignoring alpha:
// the canvas has none to speak of, and a PNG decoder's idea of it is not the point.
func compare(a, b []byte, w, h int) diff {
	d := diff{x0: w, y0: h, x1: -1, y1: -1}
	if len(a) != w*h*4 || len(b) != w*h*4 {
		return diff{n: w * h, x1: w - 1, y1: h - 1}
	}
	for i := 0; i < w*h; i++ {
		o := 4 * i
		if a[o] == b[o] && a[o+1] == b[o+1] && a[o+2] == b[o+2] {
			continue
		}
		x, y := i%w, i/w
		d.n++
		d.x0, d.y0, d.x1, d.y1 = min(d.x0, x), min(d.y0, y), max(d.x1, x), max(d.y1, y)
	}
	return d
}

// readRGBA decodes a PNG to RGBA bytes.
func readRGBA(path string) ([]byte, int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("%s: %w", path, err)
	}
	r := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			rgba.Set(x, y, img.At(r.Min.X+x, r.Min.Y+y))
		}
	}
	return rgba.Pix, r.Dx(), r.Dy(), nil
}

// keep writes RGBA bytes as a PNG under -out, for a failure to be looked at. Without -out it
// keeps nothing.
func (e *env) keep(name string, pix []byte, w, h int) {
	if e.cfg.out == "" || len(pix) != w*h*4 {
		return
	}
	if err := os.MkdirAll(e.cfg.out, 0o755); err != nil {
		return
	}
	img := &image.RGBA{Pix: pix, Stride: 4 * w, Rect: image.Rect(0, 0, w, h)}
	f, err := os.Create(filepath.Join(e.cfg.out, name))
	if err != nil {
		return
	}
	defer f.Close()
	png.Encode(f, img)
	fmt.Printf("webcheck: kept %s\n", filepath.Join(e.cfg.out, name))
}

// shoot keeps a screenshot under -out when -shots asks for them: what a person reviewing a run, or
// a CI log, can look at to see the page as a player would.
func (e *env) shoot(p *page, name string) {
	if !e.cfg.shots || e.cfg.out == "" {
		return
	}
	b, err := p.screenshot()
	if err == nil && os.MkdirAll(e.cfg.out, 0o755) == nil && os.WriteFile(filepath.Join(e.cfg.out, name), b, 0o644) == nil {
		fmt.Printf("webcheck: kept %s\n", filepath.Join(e.cfg.out, name))
	}
}

// hashDir is each file's SHA-256 and name, sorted by name, as sha256sum prints them.
func hashDir(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, ent := range ents {
		b, err := os.ReadFile(filepath.Join(dir, ent.Name()))
		if err != nil {
			return nil, err
		}
		h := sha256.Sum256(b)
		out = append(out, hex.EncodeToString(h[:])+"  "+ent.Name())
	}
	if len(out) == 0 {
		return nil, errors.New(dir + " is empty")
	}
	sort.Slice(out, func(i, j int) bool { return out[i][66:] < out[j][66:] })
	return out, nil
}

// jsString quotes s for an expression.
func jsString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x20:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
