package platform

import (
	"strings"
	"testing"
)

// TestEachReasonAWindowDidNotOpenGetsItsOwnAnswer is the whole point of DisplayAdvice being a
// function of two strings: every branch is reachable from a host that has a working display, which
// is the only kind of host this project is developed on.
//
// The assertions are on what each answer must *not* say as much as on what it must. A message that
// asks whether DISPLAY is set when the reader can see that it is, or that talks about X when the
// reader is on Wayland, does not merely fail to help -- it costs the reader the time they spend
// believing it, and then the trust they would have given the next message.
func TestEachReasonAWindowDidNotOpenGetsItsOwnAnswer(t *testing.T) {
	for _, tc := range []struct {
		why              string
		display, wayland string
		says             []string
		doesNotSay       []string
	}{
		{
			why:  "no graphical session at all",
			says: []string{"DISPLAY is unset", "-shot", "-frames"},
			// The headless flags are the actual answer here, and naming Wayland would invent a
			// session that is not running.
			doesNotSay: []string{"Wayland", "ssh"},
		},
		{
			why:     "a Wayland session with no XWayland",
			wayland: "wayland-0",
			says:    []string{"wayland-0", "XWayland", "xwayland"},
			// Not "is DISPLAY set?": it is not set, and it is not supposed to be. Telling this
			// reader to set it by hand is telling them to point at a server that is not there.
			doesNotSay: []string{"ssh", "the value is wrong"},
		},
		{
			why:     "DISPLAY set and refused",
			display: ":0",
			says:    []string{`":0"`, "refused", "-X"},
			// The one case where asking whether DISPLAY is set is worse than saying nothing.
			doesNotSay: []string{"DISPLAY is unset", "Wayland"},
		},
	} {
		got := DisplayAdvice(tc.display, tc.wayland)
		for _, want := range tc.says {
			if strings.Contains(got, want) {
				continue
			}
			// One `says` entry may be satisfied by any spelling of it, which is what the
			// XWayland/xwayland pair is doing above: the sentence names the package and the
			// technology and they are capitalised differently.
			if strings.Contains(strings.ToLower(got), strings.ToLower(want)) {
				continue
			}
			t.Errorf("%s:\n  %s\nwhich does not mention %q", tc.why, got, want)
		}
		for _, never := range tc.doesNotSay {
			if strings.Contains(got, never) {
				t.Errorf("%s:\n  %s\nwhich mentions %q, and sends the reader the wrong way",
					tc.why, got, never)
			}
		}
	}
}

// TestNoAdviceIsEverEmptyOrUnpunctuated is the shape check, and it is here because this string is
// pasted into an error with a `: ` before it. An empty branch would print "cannot open a window: "
// and a branch ending in a full stop would read as two sentences with a colon between them.
func TestNoAdviceIsEverEmptyOrUnpunctuated(t *testing.T) {
	for _, tc := range [][2]string{{"", ""}, {"", "wayland-0"}, {":0", ""}, {":0", "wayland-0"}} {
		got := DisplayAdvice(tc[0], tc[1])
		if got == "" {
			t.Errorf("DisplayAdvice(%q, %q) is empty", tc[0], tc[1])
			continue
		}
		if strings.HasSuffix(got, ".") {
			t.Errorf("DisplayAdvice(%q, %q) ends in a full stop; it is the tail of a sentence "+
				"that already has one: %s", tc[0], tc[1], got)
		}
	}

	// And the fourth combination, which the three-way switch has no branch of its own for: both
	// set means there is an X server to talk to and it said no, so it takes the refused answer.
	if both, refused := DisplayAdvice(":0", "wayland-0"), DisplayAdvice(":0", ""); both != refused {
		t.Errorf("with both variables set the advice is %q, want the refused-connection answer "+
			"%q -- DISPLAY being set is what decides, because that is what XOpenDisplay read",
			both, refused)
	}
}
