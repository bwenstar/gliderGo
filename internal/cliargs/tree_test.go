package cliargs_test

// The sweep that keeps the next subcommand from forgetting.
//
// FlagsFirst is only worth anything at the call sites, and there are fifteen of them: fourteen
// subcommands in cmd/glidertool and one flag.CommandLine in cmd/glidergo. A behavioural test
// over today's fifteen would pass forever while the sixteenth -- written by somebody who
// copied a neighbouring subcommand's first ten lines and not its eleventh -- quietly kept the
// old behaviour. That is the same class of defect as a package missing from the README's map
// (internal/citations): nothing fails, and the only person who finds out is the one who typed
// the file name first and was told their flag is not a file.
//
// So the invariant is checked against the source rather than against the behaviour: every
// FlagSet this tree parses is reordered first, and nothing calls flag.Parse.
//
// Test files are skipped on purpose. The probe in cliargs_test.go is a FlagSet built to be
// parsed both ways, and a sweep that included it would either fail or have to name an
// exception, which is how a check acquires a list of exceptions.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot walks up from the package directory to the directory holding go.mod, as
// internal/module's own sweep does.
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

func TestEveryCommandLineInTheTreeIsReorderedBeforeItIsParsed(t *testing.T) {
	root := repoRoot(t)

	var files, parses int
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			// The vendored 1994 data and the build output hold no Go, and assets/ holds a
			// generated file big enough to be worth not parsing.
			if name := d.Name(); name == ".git" || name == "GliderPRO" || name == "bin" {
				return filepath.SkipDir
			}
			return nil
		case !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go"):
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		files++

		rel, _ := filepath.Rel(root, path)
		for _, bad := range checkFile(f, flagSetNames(f)) {
			parses++
			if bad.why != "" {
				t.Errorf("%s:%d: %s", rel, fset.Position(bad.pos).Line, bad.why)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// A sweep that finds nothing to check has stopped being a sweep, which is how the four
	// stale test names internal/citations found came to be green. Fifteen is the count today;
	// the assertion is that there is more than one, so adding a subcommand needs no edit here
	// and deleting cmd/glidertool does.
	if parses < 2 {
		t.Errorf("only %d FlagSet parse(s) found across %d files: this sweep no longer "+
			"recognises how the tree parses its arguments", parses, files)
	}
	t.Logf("%d FlagSet parse(s) across %d non-test files", parses, files)
}

// found is one Parse call: where it is, and what is wrong with it, or "" if nothing is.
type found struct {
	pos token.Pos
	why string
}

// flagSetNames collects the identifiers a file assigned from flag.NewFlagSet.
//
// This is the part that keeps the sweep from having an opinion about time.Parse,
// house.ParseText or strconv.ParseInt: a Parse call is only interesting if its receiver is
// known to be a FlagSet, and the only two ways to have one here are NewFlagSet and
// flag.CommandLine.
func flagSetNames(f *ast.File) map[string]bool {
	names := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		var lhs []ast.Expr
		var rhs []ast.Expr
		switch s := n.(type) {
		case *ast.AssignStmt:
			lhs, rhs = s.Lhs, s.Rhs
		case *ast.ValueSpec:
			for _, name := range s.Names {
				lhs = append(lhs, name)
			}
			rhs = s.Values
		default:
			return true
		}
		for i, r := range rhs {
			if i >= len(lhs) || !isSelector(r, "flag", "NewFlagSet") {
				continue
			}
			if id, ok := lhs[i].(*ast.Ident); ok {
				names[id.Name] = true
			}
		}
		return true
	})
	return names
}

// checkFile returns every FlagSet parse in one file, judged.
func checkFile(f *ast.File, sets map[string]bool) []found {
	var out []found
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		// flag.Parse() is the one spelling that cannot be fixed in place, because it takes
		// no arguments to reorder: os.Args is read inside it.
		if isSelector(call.Fun, "flag", "Parse") {
			out = append(out, found{call.Pos(), "flag.Parse() reads os.Args itself, so there is " +
				"nowhere to reorder the arguments: call " +
				"flag.CommandLine.Parse(cliargs.FlagsFirst(flag.CommandLine, os.Args[1:])) " +
				"instead (internal/cliargs, docs/IMPROVEMENTS.md 4.13)"})
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Parse" || !isFlagSet(sel.X, sets) {
			return true
		}

		why := ""
		if len(call.Args) != 1 || !isSelector(call.Args[0], "cliargs", "FlagsFirst") {
			why = "this FlagSet is parsed with the arguments in the order they were given, so a " +
				"flag written after a file name is read as another file: wrap them in " +
				"cliargs.FlagsFirst(fs, args) (docs/IMPROVEMENTS.md 4.13)"
		}
		out = append(out, found{call.Pos(), why})
		return true
	})
	return out
}

// isFlagSet reports whether an expression is something this file knows to be a FlagSet.
func isFlagSet(x ast.Expr, sets map[string]bool) bool {
	switch e := x.(type) {
	case *ast.Ident:
		return sets[e.Name]
	case *ast.SelectorExpr:
		// flag.CommandLine, and also a FlagSet held in a struct field named for one --
		// p.fs.Parse -- which is how a test would spell it and is why this is not narrower.
		return isSelector(x, "flag", "CommandLine") || sets[e.Sel.Name]
	}
	return false
}

// isSelector reports whether an expression is exactly pkg.name, or a call to it.
func isSelector(x ast.Expr, pkg, name string) bool {
	if call, ok := x.(*ast.CallExpr); ok {
		x = call.Fun
	}
	sel, ok := x.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg
}
