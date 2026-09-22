package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// line is one command line lifted from a document.
type line struct {
	doc  string // "README.md"
	num  int    // 1-based, so that README.md:292 is a thing an editor can open
	text string // verbatim, comment and all, which is what gets executed
	key  string // the comment-stripped form the rules table is keyed by
}

// scrapeAll reads every document's command lines, keyed by document.
func scrapeAll(root string) (map[string][]line, error) {
	out := make(map[string][]line, len(docs))
	for _, doc := range docs {
		ls, err := scrape(root, doc)
		if err != nil {
			return nil, err
		}
		if len(ls) == 0 {
			// A document with no command lines is either a document that stopped having any or a
			// fence convention that changed under this program's feet. Both want saying out loud
			// rather than passing as "nothing to check".
			return nil, fmt.Errorf("%s has no command lines in any ```bash fence", doc)
		}
		out[doc] = ls
	}
	return out, nil
}

func scrape(root, doc string) ([]line, error) {
	b, err := os.ReadFile(filepath.Join(root, doc))
	if err != nil {
		return nil, err
	}
	return scrapeText(doc, string(b))
}

// scrapeText finds the command lines in one document: every non-empty, non-comment line inside a
// ```bash fence. Fences are tracked in pairs rather than by looking for the opening tag alone, so
// that a ```bash that appears *inside* a block of quoted sample output could not open one -- which
// is not hypothetical in a repository whose documents quote their own documents.
//
// Separate from scrape so that the fence rule can be tested against a document written to break
// it, rather than only against the two that currently obey it.
func scrapeText(doc, content string) ([]line, error) {
	var out []line
	open, info := false, ""
	for i, raw := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(raw)
		if strings.HasPrefix(t, "```") {
			open, info = !open, strings.TrimPrefix(t, "```")
			continue
		}
		if !open || info != "bash" || t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		out = append(out, line{doc: doc, num: i + 1, text: t, key: command(t)})
	}
	if open {
		return nil, fmt.Errorf("%s: a fence is never closed", doc)
	}
	return out, nil
}

// command returns the shell command a documented line names, with its trailing `#` comment
// removed and its internal whitespace collapsed. This is the key the rules table is written
// against; see the package comment for why it is the command and not the whole line.
//
// The quote tracking is there so that a `#` inside an argument -- no line has one today, and
// house names being what they are, one is a matter of time -- is not read as the start of a
// comment. It is not a shell parser and does not want to be: it knows that quotes come in pairs
// and that a comment starts after whitespace, which is the whole of what these documents use.
func command(text string) string {
	inSingle, inDouble := false, false
	for i, r := range text {
		switch {
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case r == '#' && !inSingle && !inDouble && i > 0 && (text[i-1] == ' ' || text[i-1] == '\t'):
			return collapse(text[:i])
		}
	}
	return collapse(text)
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// unclassified are the documented command lines no rule names, in document order.
func unclassified(lines map[string][]line) []line {
	var out []line
	for _, doc := range docs {
		for _, l := range lines[doc] {
			if _, ok := rules[l.key]; !ok {
				out = append(out, l)
			}
		}
	}
	return out
}

// stale are the rules no document names any more, sorted.
//
// The half of this check that is easy to leave out, and the half that decides whether it is still
// a check in a year. A rule for a line that has been deleted or reworded reads exactly like
// coverage and is not any: the line it was written for is now unclassified, which the other
// direction catches, but a rule that says `skip: "opens a window"` about a command nobody
// documents will sit there looking like a considered decision for as long as nobody checks.
func stale(lines map[string][]line) []string {
	used := make(map[string]bool, len(rules))
	for _, ls := range lines {
		for _, l := range ls {
			used[l.key] = true
		}
	}
	var out []string
	for key := range rules {
		if !used[key] {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

// classified is both directions at once, as the one error a run stops on.
func classified(lines map[string][]line) error {
	missing, dead := unclassified(lines), stale(lines)
	if len(missing) == 0 && len(dead) == 0 {
		return nil
	}
	var b strings.Builder
	if len(missing) > 0 {
		fmt.Fprintf(&b, "%d documented command line(s) are not in tools/docscheck's rules "+
			"table.\nAn empty rule{} runs the line; rule{skip: \"...\"} says in words why it "+
			"cannot be run here:\n", len(missing))
		for _, l := range missing {
			fmt.Fprintf(&b, "\n  %s:%d  %s\n      key: %q", l.doc, l.num, l.text, l.key)
		}
	}
	if len(dead) > 0 {
		if len(missing) > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "%d rule(s) name a command line no document gives any more, so they "+
			"look like coverage and are not:\n", len(dead))
		for _, key := range dead {
			fmt.Fprintf(&b, "\n  %q", key)
		}
	}
	return fmt.Errorf("%s", b.String())
}

// repoRoot walks up from the working directory to the go.mod, the same way internal/module does,
// so that this runs the same from `make docs-check` and from `go test ./tools/docscheck`.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod above %s", dir)
		}
		dir = parent
	}
}
