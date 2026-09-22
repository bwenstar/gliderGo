package main

// `house lint` and `house checks` at the command level.
//
// The checks themselves are tested in internal/house, against synthetic houses that
// break one rule at a time. What is left here is the plumbing: that the severity
// thresholds do what their flags say, that the exit status is usable in a CI step,
// that a misspelled -check is refused rather than silently reporting a clean house,
// and that the catalogue prints.
//
// Every test runs with -no-assets. The asset-backed checks depend on `make assets`
// having run, and a tool test that skips on a fresh checkout is a tool test nobody
// runs.

import (
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
)

// lint runs `house lint` over the package fixture and returns what it printed.
//
// The fixture has exactly one defect, which is what makes it useful here: room 1's
// kInvisTrans carries `where 0`, and because the fixture is a version 0x0200 house
// that unpacks to floor -8 suite 0, where the fixture has no room. So the worst
// finding is a warning, and the two thresholds can be told apart.
func lint(t *testing.T, args ...string) (string, error) {
	t.Helper()
	_, housePath := writeFixture(t)
	return captureStdout(t, func() error {
		return houseLint(append(append([]string{"-no-assets"}, args...), housePath))
	})
}

func TestHouseLintReportsTheFixture(t *testing.T) {
	out, err := lint(t)
	if err != nil {
		t.Fatalf("the fixture has no errors, so -fail error should pass: %v", err)
	}
	// The one defect: a dangling link. where 0 unpacks to floor -8, suite 0.
	if !strings.Contains(out, "link-dangling-room") {
		t.Errorf("no dangling-link finding in:\n%s", out)
	}
	if !strings.Contains(out, "floor -8 suite 0") {
		t.Errorf("the finding does not say where the link resolved to:\n%s", out)
	}
	// And the honesty note, because -no-assets was given.
	if !strings.Contains(out, "the art and sound checks did not run") {
		t.Errorf("a report with no asset tree must say so:\n%s", out)
	}
	if !strings.Contains(out, "checks-skipped") {
		t.Errorf("Lint must name the checks it skipped:\n%s", out)
	}
}

func TestHouseLintMinSeverity(t *testing.T) {
	// -min note shows the checks-skipped note; -min warn does not.
	notes, err := lint(t, "-min", "note")
	if err != nil {
		t.Fatal(err)
	}
	warns, err := lint(t, "-min", "warn")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes, "checks-skipped") {
		t.Errorf("-min note hid a note:\n%s", notes)
	}
	if strings.Contains(warns, "checks-skipped") {
		t.Errorf("-min warn printed a note:\n%s", warns)
	}
	if !strings.Contains(warns, "link-dangling-room") {
		t.Errorf("-min warn hid a warning:\n%s", warns)
	}
}

func TestHouseLintFailThreshold(t *testing.T) {
	// The fixture's worst finding is a warning, so -fail warn must exit non-zero and
	// -fail error must not. That is the whole contract a CI step depends on.
	if _, err := lint(t, "-fail", "warn"); err == nil {
		t.Error("-fail warn passed a house with a warning")
	}
	if _, err := lint(t, "-fail", "error"); err != nil {
		t.Errorf("-fail error failed a house with no errors: %v", err)
	}
	if _, err := lint(t, "-fail", "never", "-min", "note"); err != nil {
		t.Errorf("-fail never returned an error: %v", err)
	}
	if _, err := lint(t, "-fail", "sometimes"); err == nil {
		t.Error("-fail accepted a value that is not a severity")
	}
	if _, err := lint(t, "-min", "nit"); err == nil {
		t.Error("-min accepted a value that is not a severity")
	}
}

func TestHouseLintCheckFilter(t *testing.T) {
	out, err := lint(t, "-check", "link-dangling-room")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "link-dangling-room") {
		t.Errorf("-check hid the check it named:\n%s", out)
	}
	if strings.Contains(out, "checks-skipped") {
		t.Errorf("-check printed a check it did not name:\n%s", out)
	}

	// A misspelled id is the one failure a linter must not have: it would filter
	// every finding out and print a clean-looking report.
	if _, err := lint(t, "-check", "link-dangling-rooms"); err == nil {
		t.Error("-check accepted an id that is not a check, which would report a " +
			"clean house by filtering everything away")
	}
}

func TestHouseLintSummary(t *testing.T) {
	out, err := lint(t, "-summary")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "link-dangling-room") {
		t.Errorf("-summary printed a finding:\n%s", out)
	}
	if !strings.Contains(out, "1 warnings") {
		t.Errorf("-summary did not tally the warning:\n%s", out)
	}
}

func TestHouseLintNeedsAFile(t *testing.T) {
	if err := houseLint(nil); err == nil {
		t.Error("house lint with no arguments succeeded")
	}
}

// TestHouseChecksPrintsTheCatalogue makes sure the lookup table a reader is sent to
// actually prints, and prints every id. internal/house holds it to the code that
// emits the findings; this holds the command to the table.
func TestHouseChecksPrintsTheCatalogue(t *testing.T) {
	out, err := captureStdout(t, func() error { return houseLintChecks(nil) })
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range house.LintChecks() {
		if !strings.Contains(out, c.ID) {
			t.Errorf("`house checks` does not print %q", c.ID)
		}
		if !strings.Contains(out, c.What) {
			t.Errorf("`house checks` does not explain %q", c.ID)
		}
	}
	if err := houseLintChecks([]string{"extra"}); err == nil {
		t.Error("house checks accepted an argument")
	}
}

// TestHouseSubcommandsAreDispatched pins that the subcommands added after dump, build
// and check are reachable, which a dispatcher switch makes easy to forget: a command
// with tests of its own still does nothing if `house <name>` does not route to it.
func TestHouseSubcommandsAreDispatched(t *testing.T) {
	for _, sub := range []string{"lint", "checks", "stats"} {
		if err := houseCmd([]string{sub, "--help"}); err == nil {
			t.Errorf("house %s --help did not return flag.ErrHelp", sub)
		}
	}
	if err := houseCmd([]string{"lnit"}); err == nil {
		t.Error("an unknown house subcommand succeeded")
	}
}
