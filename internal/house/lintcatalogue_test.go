package house

// The catalogue test.
//
// LintChecks() is what `glidertool house checks` prints, and it is the only place a
// person can look up a check id they have just seen in a report. A hand-maintained
// list like that rots the first time someone adds a check and forgets it, and the
// failure is silent: the report still prints the id, and the lookup table just does
// not have it.
//
// So this reads lint.go's own syntax tree and holds the two together. Every `l.add`
// call site in the file names its check id as a string literal in the second
// argument, and its severity as a `Severity*` identifier in the first, which is
// enough to reconstruct the whole emittable catalogue from the code itself.
//
// This is a stronger test than exercising the linter and collecting what comes out.
// A behavioural test only proves the ids it happened to trigger exist; this proves
// there are no others.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"testing"
)

// addCallSites parses lint.go and returns, for each check id, the set of severities
// the code can report it at.
func addCallSites(t *testing.T) map[string]map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "lint.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing lint.go: %v", err)
	}

	out := map[string]map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "add" {
			return true
		}
		if len(call.Args) < 2 {
			t.Errorf("%s: l.add with %d arguments", fset.Position(call.Pos()), len(call.Args))
			return true
		}

		sev, ok := call.Args[0].(*ast.Ident)
		if !ok {
			t.Errorf("%s: l.add's severity is not a plain identifier, so this test can no "+
				"longer see it; keep the call sites literal or teach this test the new shape",
				fset.Position(call.Pos()))
			return true
		}
		lit, ok := call.Args[1].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			t.Errorf("%s: l.add's check id is not a string literal, so it cannot be "+
				"catalogued; a computed id would also be unsearchable in a report",
				fset.Position(call.Pos()))
			return true
		}
		id, err := strconv.Unquote(lit.Value)
		if err != nil {
			t.Fatalf("%s: %v", fset.Position(lit.Pos()), err)
		}
		if out[id] == nil {
			out[id] = map[string]bool{}
		}
		out[id][sev.Name] = true
		return true
	})

	if len(out) == 0 {
		t.Fatal("found no l.add call sites in lint.go; this test is not testing anything")
	}
	return out
}

func TestLintCatalogueIsComplete(t *testing.T) {
	emitted := addCallSites(t)

	catalogue := map[string]LintCheck{}
	for _, c := range LintChecks() {
		if _, dup := catalogue[c.ID]; dup {
			t.Errorf("LintChecks lists %q twice", c.ID)
		}
		catalogue[c.ID] = c
	}

	for id := range emitted {
		if _, ok := catalogue[id]; !ok {
			t.Errorf("lint.go can report %q and LintChecks does not list it, so "+
				"`glidertool house checks` cannot explain it", id)
		}
	}
	for id := range catalogue {
		if _, ok := emitted[id]; !ok {
			t.Errorf("LintChecks lists %q and no l.add call site emits it, so the "+
				"catalogue documents a check that does not exist", id)
		}
	}

	// The declared severity must be the worst the code actually uses. Too low and a
	// `-fail warn` step lets a real error through unadvertised; too high and an
	// author chases something the linter would only ever call a note.
	names := map[string]Severity{
		"SeverityNote": SeverityNote, "SeverityWarn": SeverityWarn, "SeverityError": SeverityError,
	}
	for id, sevs := range emitted {
		c, ok := catalogue[id]
		if !ok {
			continue
		}
		worst, known := SeverityNote, true
		for name := range sevs {
			s, ok := names[name]
			if !ok {
				t.Errorf("%s is reported with severity %q, which is a variable rather "+
					"than one of the three constants; write the call sites literally so "+
					"the levels a check reports at can be read off the code", id, name)
				known = false
				continue
			}
			if s > worst {
				worst = s
			}
		}
		if !known {
			continue
		}
		if c.Severity != worst {
			t.Errorf("LintChecks says %q is a %s and its worst call site reports %s",
				id, c.Severity, worst)
		}
	}
}

// TestLintChecksIsSorted keeps the catalogue in the order `house checks` prints it,
// so that adding a check is a one-line diff rather than a re-sort.
func TestLintChecksIsSorted(t *testing.T) {
	checks := LintChecks()
	ids := make([]string, len(checks))
	for i, c := range checks {
		ids[i] = c.ID
	}
	if !sort.StringsAreSorted(ids) {
		t.Errorf("LintChecks is not sorted by id: %v", ids)
	}
	for _, c := range checks {
		if c.What == "" {
			t.Errorf("%q has no description, so `house checks` would print a blank line", c.ID)
		}
	}
}
