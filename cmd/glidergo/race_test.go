package main

// The race's host half, on the parts of it a player reads rather than the parts a socket does.
//
// internal/netplay tests the protocol and internal/shell tests the screen that arranges a race.
// What is left over is this file's own: the flags' refusals, and the lines the waiting host reads
// out to the other player. Both are here for the same reason args_test.go gives -- their failure
// mode is silence. A wrong line on the waiting screen is a command the other player pastes and is
// refused by, and nothing in the program is in a position to notice.

import (
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/netplay"
)

// The house name reaches the pasteable line quoted, which it did not until the Race screen made
// this line the second-best thing on the screen rather than the only one.
//
// `glidergo -join 10.0.0.5:1138 Fun House` is three arguments, and parseFlags answers more than
// one house with "one house at a time" (TestTwoAnswersToOneQuestionAreRefused). So the unquoted
// version of this line was a worked example that could not work for any of the eight 1994 houses
// whose names have a space in them.
func TestTheHostingLinesCanBePasted(t *testing.T) {
	lines := hostingLines("0.0.0.0:1138", "Fun House")
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, `"Fun House"`) {
		t.Errorf("the hosting lines are\n%s\nand none of them quotes the house name; the "+
			"command they offer would be refused as two houses", joined)
	}
	if !strings.Contains(joined, ":1138") {
		t.Errorf("the hosting lines are\n%s\nand none of them says the port", joined)
	}

	// The address on its own, before the command line, because since the Race screen exists
	// that is what the other player types. The first line is the sentence, so the second is
	// the first thing read out, and it must not be a command.
	if len(lines) < 2 {
		t.Fatalf("only %d hosting lines: %v", len(lines), lines)
	}
	if strings.Contains(lines[1], "glidergo") || !strings.HasSuffix(lines[1], ":1138") {
		t.Errorf("the first thing read out is %q; it should be an address and a port and "+
			"nothing else, because it is typed into a box", lines[1])
	}
	if !strings.Contains(lines[0], "Race") {
		t.Errorf("the opening line is %q; it should name the screen the other player opens",
			lines[0])
	}
}

// A listener on a port nobody asked for still has to be able to say which one it got, which is why
// the port comes from the listener's address and not from the flag.
func TestTheHostingLinesUseThePortTheListenerGot(t *testing.T) {
	if got := strings.Join(hostingLines("0.0.0.0:49913", "Slumberland"), "\n"); !strings.Contains(got, ":49913") {
		t.Errorf("the hosting lines are\n%s\nand none of them says port 49913", got)
	}
	// An address that cannot be split is not a reason to print no port at all: the default is
	// what a build with no listener would have used.
	if got := strings.Join(hostingLines("nonsense", "Slumberland"), "\n"); !strings.Contains(got, ":"+netplay.DefaultPort) {
		t.Errorf("an unparseable listener address produced\n%s\nwant the default port", got)
	}
}

func TestShellQuoteOnlyQuotesWhatNeedsIt(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Slumberland", "Slumberland"},
		{"Fun House", `"Fun House"`},
		{"", `""`},
		{"CD Demo House", `"CD Demo House"`},
		// A name that is already going to confuse a shell whatever is done to it. Left
		// exactly as it is rather than quoted wrongly for one of the two shells this line
		// gets read on: see shellQuote.
		{`Say "Ah"`, `Say "Ah"`},
	} {
		if got := shellQuote(tc.in); got != tc.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// -port on its own used to be refused, because a port with no -host and no -join was a flag with
// nothing to act on. The Race screen gave it something: it is the port the title screen hosts and
// dials on, so a player who has to use a different one now has a reason to set it before the shell
// ever opens.
func TestAPortWithoutAFlagToUseItIsAccepted(t *testing.T) {
	o, err := parseArgs(t, "-port", "2000")
	if err != nil {
		t.Fatalf("glidergo -port 2000: %v", err)
	}
	if o.port != "2000" {
		t.Errorf("-port 2000 parsed as %q", o.port)
	}
	if raceRequested(o) {
		t.Error("-port on its own asked for a race; it only says which port one would use")
	}
}

// The command line's arrangement and the title screen's are the same type, so that play() has one
// path through it rather than two. asRace is the whole of the conversion, and what this checks is
// that it agrees with shell.Race.Wanted about what a race is.
func TestTheFlagsAndTheScreenArrangeTheSameThing(t *testing.T) {
	for _, tc := range []struct {
		o    options
		want bool
	}{
		{options{}, false},
		{options{host: true}, true},
		{options{join: "10.0.0.5"}, true},
	} {
		o := tc.o
		r := asRace(&o)
		if r.Wanted() != tc.want {
			t.Errorf("asRace(%+v).Wanted() = %v, want %v", tc.o, r.Wanted(), tc.want)
		}
		if r.Wanted() != raceRequested(&o) {
			t.Errorf("asRace(%+v) and raceRequested disagree about whether this is a race",
				tc.o)
		}
	}
}
