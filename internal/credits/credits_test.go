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
	for _, who := range People() {
		// The port's own two rows are not credits to a person and are not in the README.
		switch who {
		case "gliderGo", "Nobody but the people above":
			continue
		}
		// The README writes the publisher with an HTML entity for the ampersand, which is
		// a property of the README's markup and not of the name.
		want := strings.ReplaceAll(who, "&", "&amp;")
		if !strings.Contains(doc, want) {
			t.Errorf("credits.txt names %q, which GliderPRO/README.md does not", who)
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
	want := []string{"the game", "the houses", "the illustrations", "this port"}
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
