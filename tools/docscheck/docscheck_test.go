// The cheap half of docs-check, in a form `go test ./...` reaches.
//
// `make docs-check` runs the documented command lines, which takes a built pair of binaries and a
// few seconds. These cases take neither: they check that the *table* still describes the
// documents, in both directions, which is the half that rots. A command line added to the README
// in a hurry is caught by `go test ./tools/docscheck` with no assets, no binaries and no display,
// and the failure names the file to edit -- so the person who added the line is the person who
// decides whether it can be run unattended, which is the only person who knows.
//
// The last case is not about the table. `CONTRIBUTING.md` prints `make check`'s prerequisite list
// verbatim, which makes it a copied line rather than a sentence, and a copied line wants holding to
// its source. It lives here because this is the package whose subject is what the documents claim.
package main

import (
	"os"
	"path/filepath"
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

// TestTheStepListInContributingIsTheMakefilesOwn holds a copied line to the line it was copied
// from. "## `make check` is the gate" quotes the target's prerequisites exactly -- that is the
// point of it, since a reader comparing a CI log against the document should be comparing the same
// words -- and quoting something exactly is a thing that stays true only while somebody notices.
// It had already drifted once: `docs-check` was added to `check` and the document still listed the
// twelve steps before it, which is a document quietly describing last week's gate.
//
// Deliberately a substring match on the whole file rather than a search for the fence. The list is
// in an untagged fence, because it is output and not a command (see scrapeText), so the rest of
// this package cannot see it; and what matters is that the line is *somewhere* in the document
// rather than in one particular place, which leaves the author free to move the section.
func TestTheStepListInContributingIsTheMakefilesOwn(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	want := ""
	makefile := read(t, root, "Makefile")
	for _, l := range strings.Split(makefile, "\n") {
		if rest, ok := strings.CutPrefix(l, "check:"); ok {
			want = strings.TrimSpace(rest)
			break
		}
	}
	if want == "" {
		t.Fatal("the Makefile has no `check:` line with prerequisites on it, so this test cannot " +
			"say what CONTRIBUTING.md should be quoting")
	}
	if doc := read(t, root, "CONTRIBUTING.md"); !strings.Contains(doc, "\n"+want+"\n") {
		t.Errorf("CONTRIBUTING.md does not quote `make check`'s steps as the Makefile lists them.\n"+
			"  the Makefile says: %s\n  put that line, on its own, in the fence under "+
			"\"`make check` is the gate\"", want)
	}
}

func read(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
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
