package main

// The command line as a player types it, rather than as development types it.
//
// Every other flag in this program is for a measurement, a headless run or a migration, and gets
// its test beside the thing it configures. What is here is the one argument a *player* is likely
// to write, and the reason it needs a test of its own is that its failure mode is silence:
// `glidergo Slumberland` used to parse, run, and show the title screen, which is
// indistinguishable from the house having been refused. docs/IMPROVEMENTS.md 4.13 has the sweep
// that found it; this is the part that keeps it found.

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// parseArgs runs parseFlags over one command line, with a fresh FlagSet.
//
// The FlagSet has to be new each time for the reason TestResumeFlagsRefuseContradictions gives --
// parseFlags registers into flag.CommandLine and panics on a second registration of the same name
// -- and os.Args has to be restored, because a test that leaves it pointing at its own arguments
// breaks whichever test runs next rather than itself.
func parseArgs(t *testing.T, args ...string) (*options, error) {
	t.Helper()
	oldArgs, oldFlags := os.Args, flag.CommandLine
	t.Cleanup(func() { os.Args, flag.CommandLine = oldArgs, oldFlags })

	flag.CommandLine = flag.NewFlagSet("glidergo", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = append([]string{"glidergo"}, args...)
	return parseFlags()
}

// TestAHouseNamedWithoutAFlagIsStillAHouse: the gesture the 1994 program had.
//
// Glider PRO was a Mac application and a house was one of its documents, so opening one by naming
// it is the original's behaviour rather than a modern convenience -- and it is also the form an
// operating system uses when a file type is associated with a binary, which is what a
// double-clicked .house has to become when this ships for Windows and macOS.
//
// The assertion is on o.house and not on a game running, because what went wrong before was
// entirely in the parse: the name reached flag.Args() and nothing ever read it.
func TestAHouseNamedWithoutAFlagIsStillAHouse(t *testing.T) {
	o, err := parseArgs(t, "Slumberland")
	if err != nil {
		t.Fatalf("glidergo Slumberland: %v", err)
	}
	if o.house != "Slumberland" {
		t.Errorf("the house read %q; a name given without -house was dropped, which shows the "+
			"title screen and looks like the house was refused", o.house)
	}

	// A path, because that is what a file manager hands over and it must not be confused for a
	// house name to search the built-in tree for. parseFlags does not resolve it -- play does --
	// so what this pins is that the string arrives whole, separators and extension included.
	o, err = parseArgs(t, "-quiet", "/tmp/houses/Mine.house")
	if err != nil {
		t.Fatalf("glidergo /tmp/houses/Mine.house: %v", err)
	}
	if o.house != "/tmp/houses/Mine.house" {
		t.Errorf("the house read %q, want the path as given", o.house)
	}
}

// TestAHouseNamedBeforeItsFlagsIsStillAHouse is the second half of the same defect, and it
// outlived the first by three stages.
//
// 2.1 made `glidergo Slumberland` work. It did not make `glidergo Slumberland -scale 2` work, because
// the flag package stops at the first non-flag argument: -scale and 2 stayed in flag.Args(), so the
// count came to three and the player was told "one house at a time: 3 were named (Slumberland,
// -scale, 2)". That is a worse message than the silence it replaced -- it is confident, it is
// specific, and it is about the wrong thing, so the reader's next move is to try quoting the house
// name. internal/cliargs reorders the command line before Parse sees it (docs/IMPROVEMENTS.md 4.13).
//
// The last row is the reason the reordering emits its own `--`: it is the only way to name a house
// whose name begins with a dash, and it has to survive being moved.
func TestAHouseNamedBeforeItsFlagsIsStillAHouse(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		house string
		check func(*options) string
	}{
		{[]string{"Slumberland", "-quiet"}, "Slumberland", func(o *options) string {
			if !o.quiet {
				return "-quiet did not take"
			}
			return ""
		}},
		{[]string{"Slumberland", "-scale", "2"}, "Slumberland", func(o *options) string {
			if o.scale != 2 {
				return fmt.Sprintf("-scale is %d, want 2", o.scale)
			}
			return ""
		}},
		// Flags on both sides of the name, one of them a value that is a bare number, which is
		// the shape most likely to be mistaken for a second house.
		{[]string{"-two", "/tmp/Mine.house", "-room", "3"}, "/tmp/Mine.house", func(o *options) string {
			if !o.two || o.roomNum != 3 {
				return fmt.Sprintf("-two=%v -room=%d, want true and 3", o.two, o.roomNum)
			}
			return ""
		}},
		{[]string{"--", "-weird.house"}, "-weird.house", nil},
	} {
		o, err := parseArgs(t, tc.args...)
		if err != nil {
			t.Errorf("glidergo %s: %v", strings.Join(tc.args, " "), err)
			continue
		}
		if o.house != tc.house {
			t.Errorf("glidergo %s: the house read %q, want %q",
				strings.Join(tc.args, " "), o.house, tc.house)
		}
		if tc.check != nil {
			if bad := tc.check(o); bad != "" {
				t.Errorf("glidergo %s: %s", strings.Join(tc.args, " "), bad)
			}
		}
	}
}

// TestTwoAnswersToOneQuestionAreRefused is the -resume/-room principle applied to the new
// spelling: both forms at once, or two houses, is two answers, and resolving it silently drops
// one of them.
//
// The messages are asserted to quote the names back. That is the part that makes the error worth
// more than a usage dump: a shell that globbed, or a house whose name has a space in it and lost
// its quotes, produces exactly this failure, and seeing the two strings is what tells the player
// which of those happened.
func TestTwoAnswersToOneQuestionAreRefused(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		quote []string
	}{
		{[]string{"-house", "Slumberland", "Titanic"}, []string{"Slumberland", "Titanic"}},
		// The unquoted-name case, which is the commonest way this happens by accident.
		{[]string{"CD", "Demo", "House"}, []string{"CD", "Demo", "House"}},
	} {
		_, err := parseArgs(t, tc.args...)
		if err == nil {
			t.Errorf("glidergo %s was accepted", strings.Join(tc.args, " "))
			continue
		}
		for _, want := range tc.quote {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("glidergo %s: %v, which does not quote %q back",
					strings.Join(tc.args, " "), err, want)
			}
		}
	}
}

// TestAHeadlessBuildWithNothingToStopItIsRefused is the guard on the failure a test suite cannot
// otherwise report, because the failure is a hang.
//
// A null-backend build has no window and no keyboard, and null.Window delivers only the events a
// script gave it, so a play with no frame limit never sees an EventQuit and never stops. It is not
// an obscure way to arrive there: `CGO_ENABLED=0 make run` does it, and CONTRIBUTING.md offers that
// to anybody who cannot install libx11-dev.
//
// The backend name is a parameter, so all four rows run on this host -- see endlessHeadlessRun for
// why that is the point rather than a convenience.
func TestAHeadlessBuildWithNothingToStopItIsRefused(t *testing.T) {
	for _, tc := range []struct {
		why     string
		name    string
		o       options
		refused bool
	}{
		{"a play with no end, on a build that cannot be quit", "null", options{}, true},
		{"the same with a house named, which changes nothing", "null", options{house: "X"}, true},
		{"-dump with no -frames, which would also fill the disk", "null",
			options{dump: "/tmp/frames"}, true},
		{"-frames, which finishes", "null", options{frames: 3}, false},
		{"-shot, which is one screen and an exit", "null", options{shot: "/tmp/s.png"}, false},
		{"a windowed build, where the player can close the window", "x11", options{}, false},
		{"a windowed build on Windows, for the same reason", "win32", options{}, false},
	} {
		if got := endlessHeadlessRun(tc.name, &tc.o); got != tc.refused {
			t.Errorf("%s: refused=%v, want %v", tc.why, got, tc.refused)
		}
	}
}

// TestVersionSurvivesACommandLineItCannotRun is the ordering this all has to respect.
//
// -version returns before every check, on purpose: it is the command a bug report is asked to run,
// so it has to work on a machine where nothing else does -- including one whose command line is
// wrong, which is a state a report arrives in more often than not. Putting the house check above
// the -version return would be the easy mistake, and it would break the one command written to be
// pasted somewhere else.
func TestVersionSurvivesACommandLineItCannotRun(t *testing.T) {
	if _, err := parseArgs(t, "-version", "a", "b"); err != nil {
		t.Errorf("-version alongside two houses: %v; it has to print the build anyway", err)
	}
	if _, err := parseArgs(t, "-version", "-resume", "-two"); err != nil {
		t.Errorf("-version alongside a contradiction: %v", err)
	}
}
