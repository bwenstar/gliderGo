package main

// The parts of webcheck that need no browser: the protocol's framing, the static server, and the
// arithmetic the checks rest on. The server is this command's one goroutine and conn is read
// under a deadline, which is why `make race` runs this package.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeBrowser answers commands on a pipe pair the way Chrome does: an event first, then the reply.
// It stops at the first read error, which is the test closing its end.
func fakeBrowser(t *testing.T, cmds io.Reader, replies io.Writer, reply func(message) []message) {
	r := bufio.NewReader(cmds)
	for {
		b, err := r.ReadBytes(0)
		if err != nil {
			return
		}
		var m message
		if err := json.Unmarshal(bytes.TrimSuffix(b, []byte{0}), &m); err != nil {
			t.Errorf("the client sent %q, which is not JSON: %v", b, err)
			return
		}
		for _, out := range reply(m) {
			b, _ := json.Marshal(out)
			replies.Write(append(b, 0))
		}
	}
}

// pipes is the pair Chrome is given, made of net.Pipe rather than os.Pipe: a read deadline works on
// it on every system, where an os.Pipe on Windows takes none, and CI runs these tests on Windows
// although webcheck itself cannot run there (startChrome). It holds nothing, so every write waits
// for its read, and a test that writes to a side nobody reads has to read it itself.
func pipes(t *testing.T) (ourW, theirR, theirW, ourR net.Conn) {
	t.Helper()
	ourW, theirR = net.Pipe()
	theirW, ourR = net.Pipe()
	t.Cleanup(func() {
		for _, c := range []net.Conn{ourW, theirR, theirW, ourR} {
			c.Close()
		}
	})
	return ourW, theirR, theirW, ourR
}

func TestACallGetsItsReplyAndKeepsTheEventsBeforeIt(t *testing.T) {
	ourW, theirR, theirW, ourR := pipes(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fakeBrowser(t, theirR, theirW, func(m message) []message {
			if m.Method != "Page.navigate" || m.SessionID != "s1" {
				return []message{{ID: m.ID, Error: &struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{-32601, "no such method"}}}
			}
			return []message{
				{Method: "Page.loadEventFired", SessionID: "s1", Params: json.RawMessage(`{}`)},
				{ID: m.ID, SessionID: "s1", Result: json.RawMessage(`{"frameId":"f"}`)},
			}
		})
	}()
	c := newConn(ourW, ourR, ourR)

	r, err := c.call("s1", "Page.navigate", map[string]any{"url": "about:blank"}, 5*time.Second)
	if err != nil || string(r) != `{"frameId":"f"}` {
		t.Fatalf("call = %s, %v", r, err)
	}
	// The event arrived before the reply, and is still there for whoever waits for it.
	if _, err := c.event("s1", "Page.loadEventFired", time.Second); err != nil {
		t.Fatalf("the event that came first was not kept: %v", err)
	}
	// An error reply is an error with the browser's words in it.
	if _, err := c.call("s1", "Nope.nothing", nil, 5*time.Second); err == nil || !strings.Contains(err.Error(), "no such method") {
		t.Errorf("an error reply came back as %v", err)
	}
	ourW.Close() // the fake stops at the end of its input
	<-done
}

func TestABrowserThatSaysNothingIsATimeout(t *testing.T) {
	ourW, theirR, _, ourR := pipes(t)
	go io.Copy(io.Discard, theirR) // a browser that reads the command and never answers it
	c := newConn(ourW, ourR, ourR)
	start := time.Now()
	_, err := c.call("", "Browser.getVersion", nil, 200*time.Millisecond)
	if !errors.Is(err, errTimeout) {
		t.Fatalf("a silent browser gave %v, want a timeout", err)
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("the timeout took %v to arrive", took)
	}
}

// TestTheServerIsAStaticHost: the site's files, the extra names mapped in, wasm with the type
// compileStreaming insists on, and nothing cached.
func TestTheServerIsAStaticHost(t *testing.T) {
	site := t.TempDir()
	os.WriteFile(filepath.Join(site, "index.html"), []byte("<!doctype html>"), 0o644)
	os.WriteFile(filepath.Join(site, "glidergo.wasm"), []byte("\x00asm"), 0o644)
	other := filepath.Join(t.TempDir(), "elsewhere.wasm")
	os.WriteFile(other, []byte("\x00asm-null"), 0o644)

	srv, base, err := serve(site, map[string]string{"/glidergo-null.wasm": other})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	for _, tc := range []struct{ path, body, typ string }{
		{"/", "<!doctype html>", "text/html"},
		{"/glidergo.wasm", "\x00asm", "application/wasm"},
		{"/glidergo-null.wasm", "\x00asm-null", "application/wasm"},
	} {
		resp, err := http.Get(base + tc.path)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(b) != tc.body || !strings.HasPrefix(resp.Header.Get("Content-Type"), tc.typ) {
			t.Errorf("%s: %q as %q, want %q as %q", tc.path, b, resp.Header.Get("Content-Type"), tc.body, tc.typ)
		}
		if resp.Header.Get("Cache-Control") != "no-store" {
			t.Errorf("%s may be cached, and a run could see an earlier build", tc.path)
		}
	}
	if resp, err := http.Get(base + "/absent.js"); err == nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("a missing file is %d", resp.StatusCode)
		}
	}
}

func TestCompareSaysWhereTheImagesDiffer(t *testing.T) {
	const w, h = 4, 3
	a := make([]byte, w*h*4)
	b := make([]byte, w*h*4)
	if d := compare(a, b, w, h); d.n != 0 {
		t.Errorf("two equal images differ in %v", d)
	}
	b[4*(1*w+2)] = 9   // (2,1)
	b[4*(2*w+3)+1] = 9 // (3,2)
	b[4*(0*w+0)+3] = 9 // alpha only, which does not count
	d := compare(a, b, w, h)
	if d.n != 2 || d.x0 != 2 || d.x1 != 3 || d.y0 != 1 || d.y1 != 2 {
		t.Errorf("compare = %+v, want 2 pixels in columns 2-3 and rows 1-2", d)
	}
	if d := compare(a, b[:8], w, h); d.n != w*h {
		t.Errorf("a short image differs in %d pixels, want all %d", d.n, w*h)
	}
}

func TestJSStringQuotes(t *testing.T) {
	for in, want := range map[string]string{
		"/browser storage/saves/Slumberland.save": `"/browser storage/saves/Slumberland.save"`,
		`a"b\c`:     `"a\"b\\c"`,
		"tab\there": `"tab\u0009here"`,
	} {
		if got := jsString(in); got != want {
			t.Errorf("jsString(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestKeyFields(t *testing.T) {
	for code, want := range map[string]struct {
		key string
		vk  int
	}{
		"KeyQ": {"q", 'Q'}, "Space": {" ", 32}, "Tab": {"Tab", 9}, "ArrowRight": {"ArrowRight", 39},
	} {
		key, vk, err := keyFields(code)
		if err != nil || key != want.key || vk != want.vk {
			t.Errorf("keyFields(%q) = %q, %d, %v", code, key, vk, err)
		}
	}
	if _, _, err := keyFields("Digit2"); err == nil {
		t.Error("a key the table does not have was accepted")
	}
}

// TestARefusedSandboxIsReadOffTheLog: the one failure to start that webcheck answers by trying
// again, and only that one.
func TestARefusedSandboxIsReadOffTheLog(t *testing.T) {
	log := filepath.Join(t.TempDir(), "chrome.log")
	os.WriteFile(log, []byte("[FATAL:zygote_host_impl_linux.cc(127)] No usable sandbox! Update your kernel"), 0o644)
	if !sandboxRefused(log) {
		t.Error("Chrome's own words for a refused sandbox were not recognised")
	}
	os.WriteFile(log, []byte("ERROR:ui/ozone/common/gl_ozone_egl.cc:26] GLDisplayEGL::Initialize failed."), 0o644)
	if sandboxRefused(log) {
		t.Error("a GPU complaint, which every headless run makes, was read as a refused sandbox")
	}
	if sandboxRefused(log + ".absent") {
		t.Error("a log that is not there was read as a refused sandbox")
	}
}

// TestAMessageSplitByATimeoutIsStillOneMessage: a deadline that runs out halfway through a message
// must leave its first half for the next read, or every check after a slow one fails on JSON
// that is the back half of a reply.
func TestAMessageSplitByATimeoutIsStillOneMessage(t *testing.T) {
	ourW, _, theirW, ourR := pipes(t)
	c := newConn(ourW, ourR, ourR)
	whole := []byte(`{"id":7,"result":{"product":"Chrome/157"}}`)
	rest := make(chan struct{})
	go func() {
		theirW.Write(whole[:15])
		<-rest
		theirW.Write(append(whole[15:], 0))
	}()
	if _, err := c.read(time.Now().Add(100 * time.Millisecond)); !errors.Is(err, errTimeout) {
		t.Fatalf("half a message read as %v, want a timeout", err)
	}
	close(rest)
	m, err := c.read(time.Now().Add(5 * time.Second))
	if err != nil || m.ID != 7 || string(m.Result) != `{"product":"Chrome/157"}` {
		t.Fatalf("the message after the timeout was %+v, %v", m, err)
	}
}
