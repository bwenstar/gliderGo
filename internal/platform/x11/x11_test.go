//go:build linux && cgo

package x11

import (
	"os"
	"testing"

	"github.com/bwenstar/gliderGo/internal/platform"
)

// TestNewAsksForDetectableAutoRepeat is what makes Event.Repeat mean anything on Linux. By
// default an X server sends a held key's repeats as release-and-press pairs, the bitmap sees the
// key go up before every one, and a held Escape on the house picker quits the game. New asks for
// XKB's detectable auto-repeat to stop that (docs/IMPROVEMENTS.md 2.72).
//
// What it cannot do is hold a key. That needs XTest to fake a press the server will repeat, and
// XTest is a library this port does not link, so the test checks the server's own answer after
// New instead, which is the one thing New changes.
func TestNewAsksForDetectableAutoRepeat(t *testing.T) {
	// DISPLAY is what decides, as it is for `make smoke`. Unset, there is no server to ask, which
	// is every CI runner and every SSH session without -X. Set, a window is expected, and one that
	// will not open is a failure rather than a skip.
	display := os.Getenv("DISPLAY")
	if display == "" {
		t.Skip("DISPLAY is unset, so there is no X server to ask")
	}
	w, err := New(platform.Config{Title: "gliderGo x11 test"})
	if err != nil {
		t.Fatalf("DISPLAY=%q is set and New failed: %v", display, err)
	}
	defer w.Close()

	on, supported := w.detectableAutoRepeat()
	if !supported {
		t.Skipf("the server on DISPLAY=%q does not support detectable auto-repeat, so this cannot "+
			"tell whether New asked for it", display)
	}
	if !on {
		t.Error("the server supports detectable auto-repeat and it is off after New, so every " +
			"repeat of a held key will read as a fresh press")
	}
}
