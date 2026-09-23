package main

// The race's host half, on the parts of it a player reads rather than the parts a socket does.
//
// internal/netplay tests the protocol and internal/shell tests the screen that arranges a race.
// What is left over is this file's own: the flags' refusals, and the lines the waiting host reads
// out to the other player. Both are here for the same reason args_test.go gives -- their failure
// mode is silence. A wrong line on the waiting screen is a command the other player pastes and is
// refused by, and nothing in the program is in a position to notice.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/netplay"
	"github.com/bwenstar/gliderGo/internal/prefs"
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
	// A private address is marked as reachable from this network only (addressLine), and the
	// mark is not part of what gets typed.
	addr, _, _ := strings.Cut(lines[1], " ")
	if strings.Contains(addr, "glidergo") || !strings.HasSuffix(addr, ":1138") {
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

// Every fix a player can turn on reaches the other player's machine as its own rule, named the
// way the prefs file names it -- so that the other side is told "fixes.mirror_flame", the words
// it would look for in its own settings. raceRules is a hand-written list, and a fix added to
// prefs.Fixes and not to it is one the other player never hears about; this is what catches it.
//
// And every one of today's fixes is a rule that is only reported, because none of them changes
// what a race's simulation does (netplay.Rules has the four reasons). A fix that does change it
// needs a bit in netplay.RulesGated instead, and this test to say which fixes are allowed there.
func TestEveryFixHasARaceRule(t *testing.T) {
	typ := reflect.TypeOf(prefs.Fixes{})
	seen := map[netplay.Rules]string{}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		var one prefs.Fixes
		reflect.ValueOf(&one).Elem().Field(i).SetBool(true)
		r := raceRules(one)
		want := "fixes." + strings.Split(f.Tag.Get("json"), ",")[0]
		switch {
		case r == 0:
			t.Errorf("fixes.%s is not in raceRules, so a race never tells the other side about it",
				f.Name)
			continue
		case r&(r-1) != 0:
			t.Errorf("fixes.%s sets %v, more than one rule", f.Name, r)
		case r.String() != want:
			t.Errorf("fixes.%s goes out as %q; the other player's prefs file calls it %q",
				f.Name, r, want)
		case r&netplay.RulesGated != 0:
			t.Errorf("fixes.%s goes out as a rule that refuses a race; none of today's fixes "+
				"changes a race's simulation", f.Name)
		}
		if other, dup := seen[r]; dup {
			t.Errorf("fixes.%s and fixes.%s go out as the same rule", other, f.Name)
		}
		seen[r] = f.Name
	}
	if r := raceRules(prefs.Fixes{}); r != 0 {
		t.Errorf("no fixes go out as %v", r)
	}
}

// The engine fingerprint this build's races carry, pinned.
//
// Not because it is wrong to change -- it is measured, and changes whenever the simulation does
// (replay.Engine) -- but because **a change here is news**: from this build on, races with every
// release before it are refused. When this fails, the change that moved it is a change to how the
// game flies, and that is either a bug or a decision. If it is a decision, write the new value
// here and say in CHANGELOG.md that this release does not race the ones before it.
func TestTheEngineFingerprintIsPinned(t *testing.T) {
	const pinned = 0xE82F4D7565428514
	got, err := engineFingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if got != pinned {
		t.Errorf("the engine fingerprint is %016X, was %016X: this build flies differently from "+
			"the last, and will not race it", got, uint64(pinned))
	}
}
