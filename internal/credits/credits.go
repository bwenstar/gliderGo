// Package credits carries the authorship of Glider PRO, as data.
//
// It exists because of one sentence in docs/IMPROVEMENTS.md 1.2: the five house authors and
// both illustrators have to be named in the shipped build, whatever is decided about the
// assets. That is a legal obligation, not a nicety, and the thing about an obligation
// discharged by a Go string literal in a drawing function is that nobody ever diffs it
// against the document it came from.
//
// So the names live in credits.txt, embedded, and credits_test.go pins that file against
// GliderPRO/README.md -- the only statement of authorship the source release makes. A name
// added here that is not in the README fails the test; a house credited in the README and
// missing here fails it too. The file cannot quietly stop being true.
//
// The format is deliberately one that cannot fail to parse: a line in brackets opens a
// section, a line with a `|` is a row, a line without one is a note, and `#` and blanks are
// ignored. There is no error return anywhere in this package, because there is no input it
// could reject -- the input is compiled in.
package credits

import (
	_ "embed" // for the directive below; the package itself is not called
	"strings"
)

//go:embed credits.txt
var raw string

// Row is one line of a section: somebody, and what they did. A note has no Who and its
// What is the whole line, which is how the two sentences the README needs -- an alias and
// the six houses nobody is credited for -- sit among the rows they qualify.
type Row struct {
	Who  string
	What string
}

// Note reports whether this row is prose rather than a credit.
func (r Row) Note() bool { return r.Who == "" }

// Section is a heading and its rows, in the order the file gives them. Order is content
// here: the game before its houses, the houses before the pictures in them, and this port
// last, because it is the least of what is on the screen.
type Section struct {
	Title string
	Rows  []Row
}

// Sections parses the embedded file. It parses on every call rather than caching, because
// it is called when a screen opens and never in a frame, and a package-level cache is a
// thing tests have to reason about for no gain.
func Sections() []Section {
	var out []Section
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			out = append(out, Section{Title: strings.TrimSpace(line[1 : len(line)-1])})
			continue
		}
		// A row before any heading would be a row with nowhere to go, and the file has
		// none. Dropping it rather than inventing a section keeps the invariant the
		// drawing code relies on: every row belongs to a heading.
		if len(out) == 0 {
			continue
		}
		sec := &out[len(out)-1]
		if who, what, found := strings.Cut(line, "|"); found {
			sec.Rows = append(sec.Rows, Row{
				Who:  strings.TrimSpace(who),
				What: strings.TrimSpace(what),
			})
			continue
		}
		sec.Rows = append(sec.Rows, Row{What: line})
	}
	return out
}

// People is every person named in a Who field, deduplicated, in the order they first
// appear. It is what a check that "everybody who is owed a credit got one" needs, and it
// is the reason Who is a comma-separated list rather than one name per row: the README
// credits people in groups, and a row per person would say Slumberland four times.
func People() []string {
	var out []string
	seen := map[string]bool{}
	for _, sec := range Sections() {
		for _, r := range sec.Rows {
			if r.Note() {
				continue
			}
			for _, who := range strings.Split(r.Who, ",") {
				who = strings.TrimSpace(who)
				if who == "" || seen[who] {
					continue
				}
				seen[who] = true
				out = append(out, who)
			}
		}
	}
	return out
}
