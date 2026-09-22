// The cheap half of docs-check, in a form `go test ./...` reaches.
//
// `make docs-check` runs the documented command lines, which takes a built pair of binaries and a
// few seconds. These cases take neither: they check that the *table* still describes the
// documents, in both directions, which is the half that rots. A command line added to the README
// in a hurry is caught by `go test ./tools/docscheck` with no assets, no binaries and no display,
// and the failure names the file to edit -- so the person who added the line is the person who
// decides whether it can be run unattended, which is the only person who knows.
package main

import (
	"strings"
	"testing"
)

func TestEveryCommandLineTheDocumentsGiveIsClassified(t *testing.T) {
	lines := scraped(t)
	if bad := unclassified(lines); len(bad) > 0 {
		for _, l := range bad {
			t.Errorf("%s:%d is not in tools/docscheck's rules table\n  line: %s\n  key:  %q\n"+
				"  add `%q: {},` to run it, or `%q: {skip: \"why not\"},`",
				l.doc, l.num, l.text, l.key, l.key, l.key)
		}
	}
}

func TestNoRuleNamesACommandLineTheDocumentsNoLongerGive(t *testing.T) {
	for _, key := range stale(scraped(t)) {
		t.Errorf("no document gives %q any more, so its rule is not coverage; delete it", key)
	}
}

func TestBothDocumentsStillPutTheirCommandsInBashFences(t *testing.T) {
	lines := scraped(t)
	for _, doc := range docs {
		if len(lines[doc]) == 0 {
			t.Errorf("%s has no command lines in any ```bash fence; either it lost them or the "+
				"fence convention this program reads by has changed", doc)
		}
	}
}

// TestMostOfTheDocumentedLinesAreActuallyRun is the floor. Every rule in the table is a decision,
// and `skip` is the decision that costs nothing today and everything later: a table that has
// grown a reason for each of thirty-eight lines not to run would pass every case above while
// checking nothing at all. Nineteen run as this is written; the floor is set below that so that
// adding a documented command that genuinely cannot run in a check does not trip it, and low
// enough that it can only be reached by a habit rather than by a line.
func TestMostOfTheDocumentedLinesAreActuallyRun(t *testing.T) {
	lines := scraped(t)
	runnable, total := 0, 0
	for _, doc := range docs {
		for _, l := range lines[doc] {
			total++
			if rules[l.key].skip == "" {
				runnable++
			}
		}
	}
	if runnable < 12 {
		t.Errorf("only %d of %d documented command lines are run; a docs-check that skips most "+
			"of the documents is a caveat wearing a target's name", runnable, total)
	}
	t.Logf("%d of %d documented command lines are run", runnable, total)
}

func TestASkippedLineIsSkippedForOneStatedReason(t *testing.T) {
	for key, r := range rules {
		if r.skip == "" {
			continue
		}
		// needs and setup are both about running the line, so either of them beside a skip means
		// two answers to one question -- and the reader of the table cannot tell which won.
		if len(r.needs) > 0 || len(r.setup) > 0 {
			t.Errorf("%q is skipped and also has needs/setup, which only matter to a line that "+
				"runs", key)
		}
		if len(r.skip) < 20 || strings.HasSuffix(r.skip, ".") {
			t.Errorf("%q's reason is %q; it is printed under the line in a list of what was not "+
				"checked, so it wants a clause and not a word", key, r.skip)
		}
	}
}

func TestTheCommentBesideALineIsNotPartOfTheKey(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"make check      # fmt, vet, tests, build", "make check"},
		{"bin/glidertool types                # the 117 object types", "bin/glidertool types"},
		{"make\thelp\t# tabs too", "make help"},
		{"bin/glidertool house checks", "bin/glidertool house checks"},
		// The two forms a shell would not treat as a comment, and neither does this: no
		// whitespace in front of the #, and a # inside quotes. Today's documents contain
		// neither, which is why they are here -- the case that matters is the one added later.
		{"bin/glidertool types vent#3", "bin/glidertool types vent#3"},
		{`bin/glidertool house info "My #1 House.house"`, `bin/glidertool house info "My #1 House.house"`},
		{`tr '\r' '\n' < "GliderPRO/Sources/Player.c" > /tmp/Player.c`, `tr '\r' '\n' < "GliderPRO/Sources/Player.c" > /tmp/Player.c`},
	} {
		if got := command(c.text); got != c.want {
			t.Errorf("command(%q) = %q, want %q", c.text, got, c.want)
		}
	}
}

// TestSampleOutputIsNotMistakenForACommand is the fence rule: these documents put commands in
// ```bash and everything a command *printed* in an untagged fence, and the untagged ones are full
// of lines that would run if anybody ran them.
func TestSampleOutputIsNotMistakenForACommand(t *testing.T) {
	const doc = "```bash\nmake help    # a command\n```\n\n" +
		"```\n$ bin/glidertool house info x.house\nx.house: 3 rooms\nrm -rf /\n```\n\n" +
		"```bash\nmake doctor\n```\n"
	lines, err := scrapeText("synthetic.md", doc)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, l := range lines {
		got = append(got, l.key)
	}
	want := "make help, make doctor"
	if strings.Join(got, ", ") != want {
		t.Errorf("scraped %q, want %q -- an untagged fence's contents are sample output, "+
			"including any fence it quotes", strings.Join(got, ", "), want)
	}
}

// TestAnUnclosedFenceIsAnError covers the one case the pair tracking cannot resolve on its own.
// Fences are counted in pairs, so a ```bash written *inside* another fence closes it rather than
// opening a block of commands -- which leaves the tags unbalanced, and an unbalanced document is
// reported rather than guessed at. The alternative is a scraper that reads a document's prose as
// commands and says nothing.
func TestAnUnclosedFenceIsAnError(t *testing.T) {
	if _, err := scrapeText("synthetic.md", "```bash\nmake help\n"); err == nil {
		t.Error("a document whose fence is never closed scraped cleanly; every line after the " +
			"opening tag would be read as a command")
	}
}

func scraped(t *testing.T) map[string][]line {
	t.Helper()
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	lines, err := scrapeAll(root)
	if err != nil {
		t.Fatal(err)
	}
	return lines
}
