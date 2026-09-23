package credits

// The point of these tests is that credits.txt is a transcription, and a transcription is
// only worth anything while it still matches. GliderPRO/README.md is vendored read-only, so
// every name here can be checked against the only document that authorises it.
//
// The one thing GliderPRO/ is not is required: the game's data is committed, so a checkout can
// have the whole 1994 source deleted and still play. These tests skip in that case rather than
// failing, because nothing is wrong -- there is simply nothing left to compare against. They
// still fail if the directory is there and the README is not, which is damage and not a choice.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readme(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "GliderPRO")); os.IsNotExist(err) {
		t.Skip("no GliderPRO/ in this checkout: credits.txt cannot be checked against its source")
	}
	b, err := os.ReadFile(filepath.Join(root, "GliderPRO", "README.md"))
	if err != nil {
		t.Fatalf("the vendored README is the source for this file: %v", err)
	}
	return string(b)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", dir)
		}
		dir = parent
	}
}

// Nobody is named here who is not named there. This is the check that matters: a credits
// screen that invents a contributor is worse than one that omits a real one, because the
// omission is an oversight and the invention is a claim.
func TestEverybodyNamedIsNamedInTheREADME(t *testing.T) {
	doc := readme(t)
	for _, sec := range Sections() {
		// [this port] is not in the README, which is upstream's and older than the port. Its
		// rows are the port's own: the project, the person who ported it -- named at the
		// owner's request, as the title screen names them (docs/IMPROVEMENTS.md 1.5) -- and
		// whoever wrote a house in levels/. The section is skipped whole rather than name by
		// name, so that a contributor's credit does not need an edit here too, and its rows
		// are held to levels/ instead (TestEveryHouseThisPortShipsIsCredited). The port's
		// names are no longer skipped wherever they appear, as a list of them was: one
		// written into a 1994 section is checked there, and fails.
		if sec.Title == "this port" {
			continue
		}
		for _, r := range sec.Rows {
			if r.Note() {
				continue
			}
			for _, who := range strings.Split(r.Who, ",") {
				who = strings.TrimSpace(who)
				// Eliot is deliberately not checked here: upstream's README says PICT 153
				// "features a portion of this Little Nemo comic" and says nothing about the
				// line of verse set across the same plate, so the README is simply not the
				// authority for that row. TestEliotIsCreditedFromThePlateItself is.
				if who == "T.S. Eliot" {
					continue
				}
				// The README writes the publisher with an HTML entity for the ampersand,
				// which is a property of the README's markup and not of the name.
				want := strings.ReplaceAll(who, "&", "&amp;")
				if !strings.Contains(doc, want) {
					t.Errorf("credits.txt names %q under [%s], which GliderPRO/README.md does not",
						who, sec.Title)
				}
			}
		}
	}
}

// Every house authored in levels/ ships inside every executable, so each has a row under
// [this port] saying whose it is. CONTRIBUTING.md's "Whose house it is" is the rule, and this is
// what holds a patch to it. A house is found by a whole field of the row, as rowFor finds the
// 1994 ones, so "House" cannot be credited by "Open House" or "Boarding House" by a longer name.
func TestEveryHouseThisPortShipsIsCredited(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoRoot(t), "levels", "*.house.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("levels/ has no houses; the port ships two")
	}
	credited := map[string]bool{}
	for _, sec := range Sections() {
		if sec.Title != "this port" {
			continue
		}
		for _, r := range sec.Rows {
			if r.Note() {
				continue
			}
			for _, field := range strings.Split(r.What, ",") {
				credited[strings.TrimSpace(field)] = true
			}
		}
	}
	for _, f := range files {
		if name := strings.TrimSuffix(filepath.Base(f), ".house.txt"); !credited[name] {
			t.Errorf("levels/ ships %q, and credits.txt's [this port] credits nobody for it", name)
		}
	}
}

// And the five house authors item 1.2 lists really are all there, by name. Spelled out
// rather than derived, because this is the list the obligation is stated in terms of.
func TestTheFiveHouseAuthorsAndBothIllustratorsAreNamed(t *testing.T) {
	named := map[string]bool{}
	for _, who := range People() {
		named[who] = true
	}
	for _, who := range []string{
		"Kim Money", "Jonathan Chin", "Ward Hartenstein", "Steve Sullivan",
		"Shawn Brenneman",               // the five houses authors
		"John R. Neill", "Winsor McCay", // the two illustrators
		"John Calhoun", "Casady & Greene Inc.", // and the game itself
	} {
		if !named[who] {
			t.Errorf("%q is not named in credits.txt; docs/IMPROVEMENTS.md 1.2 requires it", who)
		}
	}
}

// Every house the README credits is credited here, to the same people. Checked by finding
// the house's name in the README's line and comparing the surnames in it -- which catches
// the failure this is really guarding against: a house moved to the wrong author's row.
func TestEveryHouseTheREADMECreditsIsHere(t *testing.T) {
	doc := readme(t)

	var lines []string
	for _, l := range strings.Split(doc, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "* ") && strings.Contains(l, " by ") {
			lines = append(lines, l)
		}
	}
	if len(lines) < 9 {
		t.Fatalf("found %d credit lines in the README; it had 9", len(lines))
	}

	for _, houseName := range []string{
		"Demo House", "CD Demo House", "Davis Station", "Metropolis", "Titanic",
		"Grand Prix", "Leviathan", "ImagineHouse PRO II", "In The Mirror",
		"Land of Illusion", "Nemo's Market", "Rainbow's End", "SpacePods",
		"Slumberland", "Teddy World", "The Asylum Pro",
	} {
		src := ""
		for _, l := range lines {
			if strings.Contains(l, houseName) {
				src = l
				break
			}
		}
		if src == "" {
			t.Errorf("the README does not credit %q; this test's list is out of date", houseName)
			continue
		}

		row, found := rowFor(houseName)
		if !found {
			t.Errorf("credits.txt does not credit %q, which the README does", houseName)
			continue
		}
		for _, who := range strings.Split(row.Who, ",") {
			last := lastWord(strings.TrimSpace(who))
			if !strings.Contains(src, last) {
				t.Errorf("credits.txt gives %q to %s; the README's line is %q",
					houseName, who, strings.TrimSpace(src))
			}
		}
	}
}

// rowFor finds the row crediting one house. Whole-field matching, so that "Demo House" does
// not answer with "CD Demo House"'s row by being a substring of it.
func rowFor(houseName string) (Row, bool) {
	for _, sec := range Sections() {
		for _, r := range sec.Rows {
			if r.Note() {
				continue
			}
			for _, w := range strings.Split(r.What, ",") {
				if strings.TrimSpace(w) == houseName {
					return r, true
				}
			}
		}
	}
	return Row{}, false
}

func lastWord(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return s
	}
	return f[len(f)-1]
}

// TestEliotIsCreditedFromThePlateItself covers the one row whose authority is not the README.
//
// The 1994 About plate sets a line of verse along its bottom edge. Upstream's README credits the
// picture the plate is built from and not the words on it, so a port that only transcribes the
// README loses the quotation -- which is precisely what this one did, for four stages, while also
// drawing the wrong PICT and so not even showing it. The transcription lives in
// docs/analysis/ui-dialogs.md 10.6 (internal/shell/provenance_test.go holds the plate id against
// the same section), and that is what the row is checked against.
func TestEliotIsCreditedFromThePlateItself(t *testing.T) {
	spec := filepath.Join(repoRoot(t), "docs", "analysis", "ui-dialogs.md")
	b, err := os.ReadFile(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "chambers of the sea") {
		t.Fatalf("%s no longer transcribes PICT 153's text, which is the source for the "+
			"Eliot row in credits.txt", spec)
	}

	var row Row
	for _, sec := range Sections() {
		for _, r := range sec.Rows {
			if strings.Contains(r.Who, "Eliot") {
				row = r
			}
		}
	}
	if row.Who == "" {
		t.Fatal("credits.txt credits nobody for the line of verse on PICT 153")
	}
	// The plate is the thing being credited, so the row has to point at it. "Prufrock" alone
	// would be a literary note; what a reader needs is which pixels it came off.
	if !strings.Contains(row.What, "plate") {
		t.Errorf("the Eliot row says %q, which does not say where the line appears", row.What)
	}
}

// The shape the drawing code relies on: sections in the file's order, every row under a
// heading, no row both empty and not a note.
func TestTheFileParsesIntoSectionsWithRows(t *testing.T) {
	secs := Sections()
	var titles []string
	for _, s := range secs {
		titles = append(titles, s.Title)
		if len(s.Rows) == 0 {
			t.Errorf("section %q has no rows", s.Title)
		}
		for i, r := range s.Rows {
			if r.What == "" {
				t.Errorf("section %q row %d has nothing in it", s.Title, i)
			}
			if strings.Contains(r.Who, "|") || strings.Contains(r.What, "|") {
				t.Errorf("section %q row %d still holds a separator: %+v", s.Title, i, r)
			}
		}
	}
	want := []string{"the game", "the houses", "borrowed from elsewhere", "this port"}
	if strings.Join(titles, "/") != strings.Join(want, "/") {
		t.Errorf("the sections are %v, want %v", titles, want)
	}
}

// A note is prose that qualifies the rows around it, and each belongs to the section it is
// about: the alias and the six uncredited houses under the houses, and what this port does
// not ship under this port.
func TestTheNotesSayWhatTheirSectionNeedsThemTo(t *testing.T) {
	notes := map[string]string{}
	for _, sec := range Sections() {
		for _, r := range sec.Rows {
			if r.Note() {
				notes[sec.Title] += r.What + " "
			}
		}
	}
	for _, tc := range []struct {
		section string
		want    []string
	}{
		{"the houses", []string{"Paul Finn", "Art Museum", "Fun House", "Omid"}},
		{"this port", []string{"make assets"}},
	} {
		for _, want := range tc.want {
			if !strings.Contains(notes[tc.section], want) {
				t.Errorf("%s's notes do not mention %q: %q", tc.section, want, notes[tc.section])
			}
		}
	}
	// Paul Finn is the alias, and the README says so too. If it ever stops saying so, the
	// note is the thing that is wrong.
	if !strings.Contains(readme(t), "Paul Finn") {
		t.Error("the README no longer mentions Paul Finn")
	}
}

// A comment is not a credit. The file leads with fourteen lines of them and none may reach
// a screen.
func TestCommentsAndBlanksAreNotRows(t *testing.T) {
	for _, sec := range Sections() {
		for _, r := range sec.Rows {
			if strings.HasPrefix(r.What, "#") || strings.HasPrefix(r.Who, "#") {
				t.Errorf("a comment became a row: %+v", r)
			}
		}
	}
	if n := len(People()); n < 9 {
		t.Errorf("only %d people are named: %v", n, People())
	}
}
