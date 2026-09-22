package cliargs_test

// The reordering, tested as a property rather than as an output.
//
// Almost every case below asserts that two spellings of one command line parse to the same
// thing, not that FlagsFirst returns a particular slice. That is deliberate: the returned
// slice is an implementation detail -- it carries a `--` the caller never typed -- and what
// the package promises is about meaning. A test that pinned the slice would have to be
// rewritten to change the separator and would still not say whether Parse agreed.
//
// The one thing asserted directly is what is *not* reordered, because that is where a
// well-meaning parser does damage: an unknown flag has to stay refusable by name, `-h` has to
// stay flag.ErrHelp, and a bool flag must not start eating the token after it.

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/cliargs"
)

// probe is a FlagSet shaped like the subcommands this exists for: two flags that take a
// value, two that do not, and -fail next to -f so that a name matched by prefix rather than
// in full would show up as a swallowed argument.
type probe struct {
	fs    *flag.FlagSet
	tier  *string
	out   *string
	fail  *bool
	rooms *bool
}

func newProbe() *probe {
	fs := flag.NewFlagSet("probe", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return &probe{
		fs:    fs,
		tier:  fs.String("tier", "", "a flag that takes a value"),
		out:   fs.String("o", "", "the short spelling of one"),
		fail:  fs.Bool("fail", false, "a flag that does not"),
		rooms: fs.Bool("rooms", false, "another"),
	}
}

// parse is what every caller of this package does: reorder, then hand it to Parse.
func (p *probe) parse(args []string) error {
	return p.fs.Parse(cliargs.FlagsFirst(p.fs, args))
}

// state is the whole result of a parse, as one comparable string -- the four flags and the
// positional arguments. Two command lines that mean the same thing produce the same state,
// which is the assertion this file is built on.
func (p *probe) state() string {
	return fmt.Sprintf("tier=%q o=%q fail=%v rooms=%v args=%q",
		*p.tier, *p.out, *p.fail, *p.rooms, p.fs.Args())
}

// parseState runs one command line through a fresh probe.
func parseState(t *testing.T, args ...string) (string, error) {
	t.Helper()
	p := newProbe()
	err := p.parse(args)
	return p.state(), err
}

// TestTheOrderingAShellUserTypesMeansTheSameAsTheDocumentedOne is the defect this package was
// written for.
//
// Each pair is one command line twice: the way `glidertool -h` writes it, and the way a
// person types it when they are thinking about the file first. Before this package the second
// column parsed and the first did not, and the failures were silent or misleading rather than
// refusals -- `house stats house.file -tier tutorial` printed the eighteen rows with no tier
// column and then said `open -tier: no such file or directory`.
func TestTheOrderingAShellUserTypesMeansTheSameAsTheDocumentedOne(t *testing.T) {
	for _, tc := range []struct {
		documented, habit []string
	}{
		// The two real lines from docs/IMPROVEMENTS.md 4.13, with this probe's flag names.
		{[]string{"-tier", "tutorial", "a.house"}, []string{"a.house", "-tier", "tutorial"}},
		{[]string{"-o", "out.txt", "a.house"}, []string{"a.house", "-o", "out.txt"}},

		// A value attached with = needs no lookahead at all, and must not get one.
		{[]string{"-tier=small", "-fail", "a.house"}, []string{"a.house", "-tier=small", "-fail"}},

		// Flags on both sides of the files, and more than one file: the flags keep their own
		// order and so do the files, because -o out.txt twice is the caller's last word
		// winning and a reordering that shuffled them would change which.
		{
			[]string{"-fail", "-rooms", "-tier", "epic", "a.house", "b.house"},
			[]string{"-fail", "a.house", "-rooms", "b.house", "-tier", "epic"},
		},

		// A value that looks like a flag. This is the case that makes the lookahead worth
		// having: -o is asked whether it wants the next token, and it does, so -weird.txt
		// never gets a chance to be mistaken for a flag nobody defined.
		{[]string{"-o", "-weird.txt", "a.house"}, []string{"a.house", "-o", "-weird.txt"}},

		// `-` is stdin, which `house build -` depends on, and is a positional in both
		// orderings because parseOne's own test is len(arg) < 2.
		{[]string{"-o", "out.house", "-"}, []string{"-", "-o", "out.house"}},

		// The double spelling. Go accepts -flag and --flag alike; so must the reordering,
		// or the habit ordering would work for one spelling and not the other.
		{[]string{"--tier", "large", "a.house"}, []string{"a.house", "--tier", "large"}},
	} {
		want, err := parseState(t, tc.documented...)
		if err != nil {
			t.Errorf("%s: %v, and it is the ordering the documents use", quote(tc.documented), err)
			continue
		}
		got, err := parseState(t, tc.habit...)
		if err != nil {
			t.Errorf("%s: %v", quote(tc.habit), err)
			continue
		}
		if got != want {
			t.Errorf("%s parsed as\n\t%s\nand %s as\n\t%s\nwhich are the same command line",
				quote(tc.habit), got, quote(tc.documented), want)
		}
	}
}

// TestABoolFlagDoesNotEatTheTokenAfterIt pins the half of the lookahead that has to say no.
//
// `-fail true` is not how the flag package spells a true bool -- that is `-fail=true`, and
// `true` is left over as a positional argument. Which is surprising, and is exactly why this
// package must not fix it: a reordering that decided -fail wanted a value would consume a
// file name in the habit ordering (`glidertool house check a.house -q b.house` losing
// b.house) while the documented ordering kept it, and the two orderings would stop meaning
// the same thing. IsBoolFlag is asked of the FlagSet so that the answer is the parser's own.
func TestABoolFlagDoesNotEatTheTokenAfterIt(t *testing.T) {
	got, err := parseState(t, "-fail", "true", "a.house")
	if err != nil {
		t.Fatal(err)
	}
	if want := `tier="" o="" fail=true rooms=false args=["true" "a.house"]`; got != want {
		t.Errorf("-fail true a.house parsed as\n\t%s\nwant\n\t%s", got, want)
	}
}

// TestTheCallersOwnTerminatorIsObeyed: after `--` the flag package treats every token as
// positional however it is spelled, and so does this.
//
// It is the only way a file whose name begins with a dash can be named at all, and it is
// also the only way a positional argument can reach the output looking like a flag -- which
// is the reason FlagsFirst emits a `--` of its own.
func TestTheCallersOwnTerminatorIsObeyed(t *testing.T) {
	got, err := parseState(t, "-fail", "--", "-tier", "a.house")
	if err != nil {
		t.Fatal(err)
	}
	if want := `tier="" o="" fail=true rooms=false args=["-tier" "a.house"]`; got != want {
		t.Errorf("-fail -- -tier a.house parsed as\n\t%s\nwant\n\t%s", got, want)
	}

	// And a file that got there that way survives the round trip, which is what the emitted
	// separator is for: reordering this without one would hand Parse a bare -tier again.
	got, err = parseState(t, "--", "-tier")
	if err != nil {
		t.Fatal(err)
	}
	if want := `tier="" o="" fail=false rooms=false args=["-tier"]`; got != want {
		t.Errorf("-- -tier parsed as\n\t%s\nwant\n\t%s", got, want)
	}
}

// TestAFlagNobodyDefinedIsStillRefusedByName is the guard on the tempting mistake.
//
// The alternative implementation treats anything it does not recognise as a positional
// argument, which is how a getopt wrapper usually behaves and which here would turn a typo
// into a file name: `house stats -tier smal a.house` is one story, `house stats -teir small
// a.house` must not become "no such file: -teir". Unknown flags stay in the flag stream, and
// Parse produces its own message naming the flag.
func TestAFlagNobodyDefinedIsStillRefusedByName(t *testing.T) {
	for _, args := range [][]string{
		{"-teir", "small", "a.house"},
		{"a.house", "-teir", "small"},
	} {
		_, err := parseState(t, args...)
		if err == nil {
			t.Errorf("%s was accepted", quote(args))
			continue
		}
		if !strings.Contains(err.Error(), "teir") {
			t.Errorf("%s: %v, which does not name the flag that was wrong", quote(args), err)
		}
	}
}

// TestHelpStillReachesErrHelp, in both orderings.
//
// -h is not a registered flag anywhere in this project; it is the flag package's own, and it
// is reported as flag.ErrHelp, which is how both commands know to exit 2 with usage rather
// than to log an error. Passing it through untouched is what keeps that working, and the
// habit ordering is the likelier of the two -- a reader who has just been shown a command
// line appends -h to it.
func TestHelpStillReachesErrHelp(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"a.house", "-h"}, {"-fail", "a.house", "--help"}} {
		if _, err := parseState(t, args...); !errors.Is(err, flag.ErrHelp) {
			t.Errorf("%s: %v, want flag.ErrHelp", quote(args), err)
		}
	}
}

// TestNothingIsDroppedOrInvented is the arithmetic check, and it is here because the two ways
// to get this wrong are both off-by-one in a loop that consumes a lookahead token: dropping
// the last argument, or emitting it twice.
//
// It also covers the spellings the flag package calls bad syntax -- `---tier` and `-=x` --
// which this package leaves alone on purpose, so they arrive at Parse to be refused there.
// They are checked for preservation rather than for a message, because the message is Parse's
// to choose.
func TestNothingIsDroppedOrInvented(t *testing.T) {
	for _, args := range [][]string{
		{"a.house", "-tier", "small"},
		{"a.house", "-tier"},            // a value flag with nothing after it
		{"-o"},                          // the same, alone
		{"a.house", "-fail"},            // a bool flag last
		{"---tier", "small", "a.house"}, // bad syntax, left for Parse
		{"-=x", "a.house"},              //  "
		{"-", "--", "-"},                // stdin, a terminator, and a file called -
		{},                              // nothing at all
	} {
		got := cliargs.FlagsFirst(newProbe().fs, args)
		counts := map[string]int{}
		for _, a := range args {
			counts[a]++
		}
		for _, a := range got {
			counts[a]--
		}
		// One "--" more than was given is the separator, and the only difference allowed.
		delete(counts, "--")
		for a, n := range counts {
			if n != 0 {
				t.Errorf("%s -> %s: %q appears %d time(s) fewer than it was given",
					quote(args), quote(got), a, n)
			}
		}
		if extra := len(got) - len(args); extra < 0 || extra > 1 {
			t.Errorf("%s -> %s: %d arguments became %d", quote(args), quote(got),
				len(args), len(got))
		}
	}
}

// TestACommandLineWithNoFilesIsUntouched: the separator is only emitted when there is
// something to separate, so `house checks` and `replay -script -` come out byte for byte as
// they went in.
//
// Which matters for one reason: a `--` in a command line that never had a positional argument
// would be a thing to explain in every error message and every -h example, for no gain.
func TestACommandLineWithNoFilesIsUntouched(t *testing.T) {
	for _, args := range [][]string{{}, {"-fail"}, {"-tier", "small", "-rooms"}} {
		got := cliargs.FlagsFirst(newProbe().fs, args)
		if quote(got) != quote(args) {
			t.Errorf("%s came back as %s", quote(args), quote(got))
		}
	}
}

// quote renders a command line the way a person would have typed it, so that a failure can be
// pasted into a shell.
func quote(args []string) string {
	if len(args) == 0 {
		return "(no arguments)"
	}
	return strings.Join(args, " ")
}
