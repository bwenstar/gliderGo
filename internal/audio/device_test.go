package audio

// The output-choosing rules, tested in the form that holds on every platform.
//
// Nothing here opens a real output. `Open("")` deliberately takes the best thing the machine has,
// which on a developer's box means starting a player and making a noise in the middle of a test run,
// so the tests below exercise the paths that resolve a *name* and the accounting around them.

import (
	"errors"
	"strings"
	"testing"
)

// *Pipe has to satisfy Stream or the shutdown report loses the drop counter, and openDevice's
// signature makes the compiler check the same thing for the native sink on the platform that has one.
var _ Stream = (*Pipe)(nil)

// TestOpenRejectsAnUnknownName checks that a name is insisted on rather than quietly replaced.
//
// The flag exists for somebody diagnosing one specific output, so silently using a different one
// would waste their afternoon -- and the message has to carry both the name they typed and the names
// that exist, because "unknown output" on its own does not tell them what to type instead.
func TestOpenRejectsAnUnknownName(t *testing.T) {
	s, err := Open("no-such-output")
	if err == nil {
		s.Close()
		t.Fatal("Open accepted an output that does not exist")
	}
	for _, want := range []string{"no-such-output", "known outputs"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// TestOutputNamesAreOfferable checks that every name the error message suggests is a name Open would
// accept. Suggesting a spelling that is then rejected is the one failure this message can have.
func TestOutputNamesAreOfferable(t *testing.T) {
	names := outputNames()
	if len(names) == 0 {
		t.Fatal("outputNames is empty; the unknown-output message would suggest nothing")
	}
	if deviceName != "" && names[0] != deviceName {
		t.Errorf("outputNames starts with %q, want the native device %q first", names[0], deviceName)
	}
	// Membership is checked against the tables rather than by opening each one, because a name
	// that resolves would start a player and make a noise in the middle of a test run.
	for _, name := range names {
		if name == "" {
			t.Error("outputNames contains an empty name")
			continue
		}
		if name == deviceName {
			continue
		}
		known := false
		for _, p := range players {
			if p.name == name {
				known = true
				break
			}
		}
		if !known {
			t.Errorf("outputNames suggests %q, which OpenPipe would reject as unknown", name)
		}
	}
}

// TestNoDeviceIsASentinel is what lets Open fall through to the external players: a platform with no
// native driver, and a Windows machine with no sound card, both have to answer errNoDevice rather
// than an ordinary error, or a Linux run would report a device failure it never had.
func TestNoDeviceIsASentinel(t *testing.T) {
	if haveDevice() {
		t.Skip("this machine has a native device, and opening it would make a noise")
	}
	s, err := openDevice()
	if err == nil {
		s.Close()
		t.Fatal("openDevice succeeded although haveDevice said there is none")
	}
	if !errors.Is(err, errNoDevice) {
		t.Errorf("openDevice failed with %v, want something wrapping errNoDevice", err)
	}
}

// TestOutputsListsWhatIsThere checks the list `-audio list` prints: the native device first when
// there is one, then the players, and nothing invented.
func TestOutputsListsWhatIsThere(t *testing.T) {
	got, players := Outputs(), Players()
	want := len(players)
	if haveDevice() {
		want++
		if len(got) == 0 || got[0] != deviceName {
			t.Errorf("Outputs = %v, want %q first", got, deviceName)
		}
	}
	if len(got) != want {
		t.Errorf("Outputs = %v (%d entries), want %d", got, len(got), want)
	}
	if deviceName == "" && haveDevice() {
		t.Error("a platform with no device name reports having one; -audio list would print an empty word")
	}
}
