package main

// Starting Chrome, and driving one page of it.
//
// The machine this is run on may have somebody's own Chrome open on it, and nothing here may touch
// that one. So the browser started here gets a profile and a home directory of its own under a
// temporary directory, which keeps it out of their profile and out of their running browser's
// way. It is cut off from their desktop as well: no DISPLAY or WAYLAND_DISPLAY, so that it can
// never open a window on anybody's; no XDG_RUNTIME_DIR, DBUS_SESSION_BUS_ADDRESS or PULSE_SERVER,
// so that it finds neither their sound server nor their session bus; and --mute-audio, so that
// the game it plays makes no sound even where it could. It is stopped with Browser.close, and
// killed by its own process handle if that does not work -- never by name, which would match
// every Chrome on the machine.

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// chromeNames are the commands a Chrome is found under, in the order they are tried.
var chromeNames = []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"}

// findChrome is the -chrome flag, or the first of chromeNames on PATH.
func findChrome(flagValue string) (string, error) {
	if flagValue != "" {
		return exec.LookPath(flagValue)
	}
	for _, name := range chromeNames {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no Chrome on PATH (tried %s); -chrome names one",
		strings.Join(chromeNames, ", "))
}

// The page's size, and the screen's. Fixed, so that auto magnification comes to the same 2x on
// every machine this runs on: 1400x1000 fits 1280x960 and not 1920x1440.
const (
	viewW, viewH = 1400, 1000
)

// strangerHost is a name for the page that is not localhost: what the page sees on github.io. The
// .test domain is reserved, so it can be nobody's real host.
const strangerHost = "glidergo.test"

type browser struct {
	cmd  *exec.Cmd
	c    *conn
	pipe *os.File // our end of fd 4, closed after the browser is
}

// startChrome starts a headless Chrome whose profile and home are under dir. noSandbox is for a
// machine that refuses Chrome's sandbox; see sandboxRefused.
//
// Linux and macOS only. The debugging pipe is two file descriptors the child inherits at 3 and 4,
// and Windows has no way to hand a child a descriptor by number (exec.Cmd.ExtraFiles), so there
// this says so rather than starting a Chrome that will never answer.
func startChrome(path, dir string, noSandbox bool) (*browser, error) {
	if runtime.GOOS == "windows" {
		return nil, errors.New("webcheck runs on Linux and macOS: Chrome's debugging pipe is two " +
			"inherited file descriptors, which Windows cannot pass to a child")
	}
	home := filepath.Join(dir, "home")
	profile := filepath.Join(dir, "profile")
	for _, d := range []string{home, profile} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, err
		}
	}
	toChrome, ourW, err := os.Pipe() // Chrome reads fd 3
	if err != nil {
		return nil, err
	}
	ourR, fromChrome, err := os.Pipe() // Chrome writes fd 4
	if err != nil {
		return nil, err
	}

	args := []string{
		"--headless", "--remote-debugging-pipe", "--user-data-dir=" + profile,
		"--no-first-run", "--no-default-browser-check", "--disable-extensions",
		"--disable-background-networking", "--disable-sync", "--disable-component-update",
		"--disable-default-apps", "--disable-gpu", "--mute-audio", "--password-store=basic",
		fmt.Sprintf("--window-size=%d,%d", viewW, viewH),
		// A name that is not this machine's, for the page as a stranger's link reaches it
		// (checkQuery); every lookup of it is the loopback server.
		"--host-resolver-rules=MAP " + strangerHost + " 127.0.0.1",
	}
	if noSandbox {
		args = append(args, "--no-sandbox")
	}
	cmd := exec.Command(path, append(args, "about:blank")...)
	cmd.ExtraFiles = []*os.File{toChrome, fromChrome} // fds 3 and 4
	var env []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		switch k {
		case "DISPLAY", "WAYLAND_DISPLAY", "HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME",
			"XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS", "PULSE_SERVER":
			continue
		}
		env = append(env, kv)
	}
	cmd.Env = append(env, "HOME="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_CACHE_HOME="+filepath.Join(home, ".cache"), "XDG_DATA_HOME="+filepath.Join(home, ".local", "share"))
	log, err := os.Create(filepath.Join(dir, "chrome.log"))
	if err != nil {
		return nil, err
	}
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting %s: %w", path, err)
	}
	toChrome.Close()
	fromChrome.Close()
	log.Close()

	b := &browser{cmd: cmd, c: newConn(ourW, ourR, ourR), pipe: ourR}
	if _, err := b.c.call("", "Browser.getVersion", nil, 30*time.Second); err != nil {
		b.close()
		return nil, fmt.Errorf("%s did not answer on its debugging pipe (%v); its output is in %s",
			path, err, filepath.Join(dir, "chrome.log"))
	}
	return b, nil
}

// sandboxRefused reports whether a Chrome that would not start said why in its log: a kernel that
// will not give it the namespaces its sandbox is built from. Ubuntu 24.04 refuses them to any
// program without an AppArmor profile, which a hosted CI runner may well be running. The answer
// is --no-sandbox, and taking it here is safe in a way it would not be on a desktop, because the
// only page this browser ever opens is this build's own, from a loopback server.
func sandboxRefused(log string) bool {
	b, err := os.ReadFile(log)
	return err == nil && strings.Contains(string(b), "No usable sandbox")
}

func (b *browser) version() string {
	r, err := b.c.call("", "Browser.getVersion", nil, 10*time.Second)
	if err != nil {
		return "?"
	}
	var v struct{ Product string }
	json.Unmarshal(r, &v)
	return v.Product
}

// close asks the browser to quit and waits, and kills it -- this process, by its handle -- if it
// has not gone in fifteen seconds.
func (b *browser) close() {
	b.c.call("", "Browser.close", nil, 10*time.Second)
	kill := time.AfterFunc(15*time.Second, func() { b.cmd.Process.Kill() })
	b.cmd.Wait()
	kill.Stop()
	b.pipe.Close()
}

// page is one tab, in a browser context of its own: its own localStorage, so that no check sees
// what another left behind.
type page struct {
	c       *conn
	session string
	context string
}

func (b *browser) newPage() (*page, error) {
	r, err := b.c.call("", "Target.createBrowserContext", nil, 30*time.Second)
	if err != nil {
		return nil, err
	}
	var ctx struct{ BrowserContextID string }
	json.Unmarshal(r, &ctx)
	r, err = b.c.call("", "Target.createTarget",
		map[string]any{"url": "about:blank", "browserContextId": ctx.BrowserContextID}, 30*time.Second)
	if err != nil {
		return nil, err
	}
	var t struct{ TargetID string }
	json.Unmarshal(r, &t)
	r, err = b.c.call("", "Target.attachToTarget", map[string]any{"targetId": t.TargetID, "flatten": true}, 30*time.Second)
	if err != nil {
		return nil, err
	}
	var s struct{ SessionID string }
	json.Unmarshal(r, &s)
	p := &page{c: b.c, session: s.SessionID, context: ctx.BrowserContextID}
	for _, m := range []string{"Page.enable", "Runtime.enable"} {
		if _, err := p.cmd(m, nil); err != nil {
			return nil, err
		}
	}
	if _, err := p.cmd("Emulation.setDeviceMetricsOverride", map[string]any{
		"width": viewW, "height": viewH, "deviceScaleFactor": 1, "mobile": false,
		"screenWidth": viewW, "screenHeight": viewH,
	}); err != nil {
		return nil, err
	}
	// A headless page is never the focused window, and a game that pauses when it loses focus
	// would otherwise be paused before it started.
	if _, err := p.cmd("Emulation.setFocusEmulationEnabled", map[string]any{"enabled": true}); err != nil {
		return nil, err
	}
	return p, nil
}

// close disposes of the page's browser context, and so of the page and the game it was running:
// a check that left its game playing would be load on the next check's clock.
func (p *page) close() {
	p.c.call("", "Target.disposeBrowserContext", map[string]any{"browserContextId": p.context}, 30*time.Second)
	p.c.drop(p.session, func(message) bool { return false })
}

func (p *page) cmd(method string, params any) (json.RawMessage, error) {
	return p.c.call(p.session, method, params, 60*time.Second)
}

// open loads a URL and waits for its load event.
func (p *page) open(url string) error {
	if _, err := p.cmd("Page.navigate", map[string]any{"url": url}); err != nil {
		return err
	}
	_, err := p.c.event(p.session, "Page.loadEventFired", 60*time.Second)
	return err
}

// eval runs an expression in the page, awaiting it if it is a promise, and unmarshals its value
// into v (which may be nil).
func (p *page) eval(expr string, v any) error { return p.evalFor(expr, v, 60*time.Second) }

// evalFor is eval with a timeout of its own: a page whose game never sleeps answers nothing until
// the game is over, so a wait for one has to be a single long call.
func (p *page) evalFor(expr string, v any, timeout time.Duration) error {
	r, err := p.c.call(p.session, "Runtime.evaluate", map[string]any{
		"expression": expr, "returnByValue": true, "awaitPromise": true,
	}, timeout)
	if err != nil {
		return err
	}
	var out struct {
		Result           struct{ Value json.RawMessage }
		ExceptionDetails *struct {
			Text      string
			Exception *struct{ Description string }
		}
	}
	if err := json.Unmarshal(r, &out); err != nil {
		return err
	}
	if e := out.ExceptionDetails; e != nil {
		if e.Exception != nil {
			return fmt.Errorf("in the page: %s", e.Exception.Description)
		}
		return fmt.Errorf("in the page: %s", e.Text)
	}
	if v == nil || len(out.Result.Value) == 0 {
		return nil
	}
	return json.Unmarshal(out.Result.Value, v)
}

// until polls an expression until it is true, or says what it was waiting for. Each poll may
// take as long as is left, so that a page that cannot answer until its game ends -- an unpaced
// run -- is waited for for all of timeout and not for one call's worth.
func (p *page) until(expr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		var ok bool
		if err := p.evalFor("!!("+expr+")", &ok, max(time.Until(deadline), time.Second)); err != nil {
			if errors.Is(err, errTimeout) {
				return fmt.Errorf("waited %v for %s", timeout, expr)
			}
			return err
		}
		if ok {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("waited %v for %s", timeout, expr)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// keyNames is what a key event needs beyond its code: the key it reports and the Windows code
// Chrome wants. Only the keys a check presses.
var keyNames = map[string]struct {
	key string
	vk  int
}{
	"Enter": {"Enter", 13}, "Space": {" ", 32}, "Escape": {"Escape", 27}, "Tab": {"Tab", 9},
	"ArrowLeft": {"ArrowLeft", 37}, "ArrowUp": {"ArrowUp", 38},
	"ArrowRight": {"ArrowRight", 39}, "ArrowDown": {"ArrowDown", 40},
}

func keyFields(code string) (key string, vk int, err error) {
	if k, ok := keyNames[code]; ok {
		return k.key, k.vk, nil
	}
	if l, ok := strings.CutPrefix(code, "Key"); ok && len(l) == 1 && l[0] >= 'A' && l[0] <= 'Z' {
		return strings.ToLower(l), int(l[0]), nil
	}
	return "", 0, fmt.Errorf("no key %q in webcheck's table", code)
}

// down and up are the two halves of a key press, for a key that is held.
func (p *page) down(code string) error { return p.keyEvent("keyDown", code) }
func (p *page) up(code string) error   { return p.keyEvent("keyUp", code) }

func (p *page) keyEvent(kind, code string) error {
	key, vk, err := keyFields(code)
	if err != nil {
		return err
	}
	ev := map[string]any{"type": kind, "code": code, "key": key,
		"windowsVirtualKeyCode": vk, "nativeVirtualKeyCode": vk}
	if kind == "keyDown" && len(key) == 1 {
		ev["text"] = key
	}
	_, err = p.cmd("Input.dispatchKeyEvent", ev)
	return err
}

// tap is a press and a release, with the release a little later: the game reads held keys once
// a frame, and a press and release inside one frame is a press it never sees.
func (p *page) tap(code string) error {
	if err := p.down(code); err != nil {
		return err
	}
	time.Sleep(120 * time.Millisecond)
	return p.up(code)
}

// hold keeps a key down for d.
func (p *page) hold(code string, d time.Duration) error {
	if err := p.down(code); err != nil {
		return err
	}
	time.Sleep(d)
	return p.up(code)
}

// screenshot is the page as it is drawn, as a PNG.
func (p *page) screenshot() ([]byte, error) {
	r, err := p.cmd("Page.captureScreenshot", map[string]any{"format": "png"})
	if err != nil {
		return nil, err
	}
	var shot struct{ Data string }
	if err := json.Unmarshal(r, &shot); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(shot.Data)
}

// canvas is the canvas's pixels, RGBA, as getImageData reads them.
func (p *page) canvas() ([]byte, error) {
	var s string
	if err := p.eval(`(() => {
		const cv = document.getElementById("glidergo");
		const d = cv.getContext("2d").getImageData(0, 0, cv.width, cv.height).data;
		let s = "";
		for (let i = 0; i < d.length; i += 0x8000) s += String.fromCharCode.apply(null, d.subarray(i, i + 0x8000));
		return btoa(s);
	})()`, &s); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(s)
}

// console is what the page logged that was not the game's own lines, which go to glidergoLog,
// plus anything it threw.
func (p *page) console() []string {
	var out []string
	for _, m := range p.c.drop(p.session, func(m message) bool {
		return m.Method == "Runtime.exceptionThrown" || m.Method == "Runtime.consoleAPICalled"
	}) {
		var e struct {
			Type             string
			Args             []struct{ Value any }
			ExceptionDetails struct {
				Text      string
				Exception *struct{ Description string }
			}
		}
		json.Unmarshal(m.Params, &e)
		if m.Method == "Runtime.exceptionThrown" {
			d := e.ExceptionDetails.Text
			if e.ExceptionDetails.Exception != nil {
				d = e.ExceptionDetails.Exception.Description
			}
			out = append(out, "exception: "+d)
			continue
		}
		var parts []string
		for _, a := range e.Args {
			parts = append(parts, fmt.Sprint(a.Value))
		}
		line := strings.Join(parts, " ")
		if !strings.HasPrefix(line, "glidergo: ") {
			out = append(out, e.Type+": "+line)
		}
	}
	return out
}
