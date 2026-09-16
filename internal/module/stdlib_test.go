package module

// gliderGo depends on the standard library and nothing else. This file is what makes that a
// checked property rather than a habit.
//
// It matters more than a preference about dependencies. Three separate things in this
// repository are true only because the import graph stops at the standard library:
//
//   - The airgapped build host works at all. That network has no Go module proxy of any kind
//     -- not an empty one, none -- so the first `require` line would make the build
//     unreproducible there, and the person who added it would very likely not be the person
//     who found out.
//   - GOPROXY=off is set unconditionally in the Makefile and in scripts/env.sh. That is a
//     policy, and a policy nothing verifies is just a comment.
//   - There is no go.sum, so there is no supply chain to audit before a public release. That
//     is a real security property of the shipped binary and it is worth keeping deliberately
//     rather than by accident.
//
// The check runs entirely offline: it parses go.mod as text and every .go file with go/parser.
// It never shells out to `go list`, which would need a module cache and, in the failure case
// this test exists to catch, a network. That is the point -- the assertion has to hold on the
// machine that cannot download anything.
//
// Parsing rather than listing also makes the check *stronger* than `go list ./...` would be.
// go/parser reads a file's imports whatever its build constraints say, so an import reachable
// only under GOOS=windows, or only behind a `nullbackend` tag, is caught here. `go list` for
// this host would not see it, and CI would find it on the day someone cross-compiles.

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// modulePath is what go.mod declares, and therefore the one non-stdlib import prefix that is
// legitimate: gliderGo's own packages.
const modulePath = "glidergo"

// repoRoot walks up from the package directory to the directory holding go.mod.
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

// ---------------------------------------------------------------------------
// go.mod itself
// ---------------------------------------------------------------------------

// checkGoMod reports every reason the given go.mod is not dependency-free, as one string per
// problem, each prefixed with its line number.
//
// It is a pure function over bytes rather than a test body, and that is not tidiness. Feeding
// the real go.mod a `require` line to see this fire does not work: the *go command* refuses to
// build the package first, with "missing go.sum entry", so the test never runs. A pure function
// can be handed a synthetic go.mod, which is what TestTheGoModCheckCatchesEachWayIn does. The
// alternative was a check nobody had ever seen fail.
//
// Hand-parsing a format that has a real parser in golang.org/x/mod also looks like the wrong
// call, and would be, except that importing golang.org/x/mod to prove nothing imports
// golang.org/x/anything is a contradiction. The grammar for the directives that matter here is
// a line and an optional parenthesised block, which is small enough to read correctly.
func checkGoMod(raw []byte) []string {
	// Every directive that can pull in code from outside this module. `replace` and `exclude`
	// only ever apply to a require, so on their own they are already a sign something is off.
	banned := map[string]string{
		"require": "adds a dependency",
		"replace": "redirects a dependency",
		"exclude": "excludes a dependency version",
		"tool":    "adds a tool dependency",
	}

	var problems []string
	var sawModule, sawGo bool
	var block string // the directive whose ( ... ) block we are inside, if any

	add := func(lineno int, format string, args ...any) {
		problems = append(problems, fmt.Sprintf("go.mod:%d: ", lineno)+fmt.Sprintf(format, args...))
	}

	for i, line := range strings.Split(string(raw), "\n") {
		lineno := i + 1
		// Strip comments and surrounding space. A `// require` in prose is not a require.
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if block != "" {
			if line == ")" {
				block = ""
				continue
			}
			add(lineno, "inside a %s block: %q -- gliderGo is standard library only", block, line)
			continue
		}

		fields := strings.Fields(line)
		directive := fields[0]

		if why, bad := banned[directive]; bad {
			if len(fields) > 1 && fields[1] == "(" {
				block = directive
			}
			add(lineno, "%q %s: %q", directive, why, line)
			continue
		}

		switch directive {
		case "module":
			sawModule = true
			if len(fields) < 2 || fields[1] != modulePath {
				add(lineno, "module path is %q, want %q -- if it was renamed on purpose, "+
					"update modulePath in this file too", line, modulePath)
			}
		case "go":
			sawGo = true
		case "godebug", "toolchain", "retract":
			// Harmless: none of these can introduce code from another module.
		default:
			add(lineno, "unrecognised directive %q. This test does not know whether it can "+
				"pull in outside code, which is reason enough to look: %q", directive, line)
		}
	}

	if block != "" {
		problems = append(problems, fmt.Sprintf("go.mod: unterminated %s block", block))
	}
	// Without these two the parse above proves nothing -- an empty or unreadable go.mod would
	// satisfy every check so far.
	if !sawModule {
		problems = append(problems, "go.mod has no module directive; this parse did not read "+
			"what it thinks it did")
	}
	if !sawGo {
		problems = append(problems, "go.mod has no go directive; the toolchain floor is what "+
			"scripts/bootstrap-dev-env.sh reads")
	}
	return problems
}

// TestGoModDeclaresNoDependencies runs that check against the real go.mod.
func TestGoModDeclaresNoDependencies(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range checkGoMod(raw) {
		t.Errorf("%s\n\n"+
			"gliderGo depends on the standard library only. If a dependency is genuinely\n"+
			"wanted, that is a project decision rather than a build fix: it ends\n"+
			"GOPROXY=off, it puts a go.sum in the release, and it has to work on a network\n"+
			"with no module proxy. See the comment at the top of this file.", p)
	}
}

// TestTheGoModCheckCatchesEachWayIn is the negative half: one case per route a dependency can
// take into go.mod, plus the two cases where the parser must not cry wolf.
func TestTheGoModCheckCatchesEachWayIn(t *testing.T) {
	const good = "module glidergo\n\ngo 1.23\n"

	for _, tc := range []struct {
		name  string
		mod   string
		wantN int
	}{
		{"the real shape", good, 0},
		{"a comment mentioning require", good + "\n// we deliberately require nothing\n", 0},
		{"a toolchain line", good + "toolchain go1.23.12\n", 0},

		{"a single-line require", good + "require github.com/x/y v1.2.3\n", 1},
		{"a require block", good + "require (\n\tgithub.com/x/y v1.2.3\n)\n", 2},
		{"two in a block", good + "require (\n\ta.com/b v1.0.0\n\tc.com/d v2.0.0\n)\n", 3},
		{"a replace", good + "replace github.com/x/y => ../y\n", 1},
		{"an exclude", good + "exclude github.com/x/y v1.0.0\n", 1},
		{"a tool directive", good + "tool github.com/x/y/cmd/z\n", 1},
		{"an inline require after a comment", good + "require a.com/b v1.0.0 // indirect\n", 1},

		// Three: the require itself, the line inside it, and the missing ")".
		{"an unterminated block", good + "require (\n\ta.com/b v1.0.0\n", 3},
		{"an unknown directive", good + "vendorise all\n", 1},
		{"the wrong module path", "module notglidergo\n\ngo 1.23\n", 1},
		{"no module line", "go 1.23\n", 1},
		{"no go line", "module glidergo\n", 1},
		{"empty", "", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := checkGoMod([]byte(tc.mod))
			if len(got) != tc.wantN {
				t.Errorf("checkGoMod reported %d problems, want %d:\n  %s",
					len(got), tc.wantN, strings.Join(got, "\n  "))
			}
		})
	}
}

// TestTheImportRuleAcceptsTheStandardLibraryAndRefusesModules pins allowedImport, which is the
// one line of judgement in this file: "no dot in the first path element" as the test for
// standard library.
func TestTheImportRuleAcceptsTheStandardLibraryAndRefusesModules(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"os", true},
		{"path/filepath", true},
		{"encoding/binary", true},
		{"C", true},                      // cgo, used by internal/platform/x11
		{"glidergo", true},               // the module itself
		{"glidergo/internal/game", true}, // and its packages

		{"github.com/user/repo", false},
		{"golang.org/x/image/draw", false},       // the one a renderer would reach for first
		{"gitlab.internal.example.com/x", false}, // a self-hosted forge, dots and all
		{"example.com", false},
		{"./local", false},
		{"../sibling", false},

		// Accepted, and worth saying why rather than leaving it to look like a hole. A path
		// with no dot in its first element is stdlib or it is nothing: no one else may publish
		// such a module path, so `go build` fails with "package glidergofake is not in std",
		// which is a better message than this test could write. The only ways a no-dot path
		// resolves to outside code are a `replace`, a vendor tree, or GOPATH mode -- and the
		// other two tests in this file rule out all three. That is why both halves exist.
		{"glidergofake", true},
	} {
		got, why := allowedImport(tc.path)
		if got != tc.want {
			t.Errorf("allowedImport(%q) = %v (%s), want %v", tc.path, got, why, tc.want)
		}
	}
}

// TestThereIsNoLockFileOrVendorTree: both are artefacts of having dependencies, so either one
// appearing means the property this file protects has already been broken -- possibly by a
// stray `go get` or `go mod tidy` that someone ran and did not commit the go.mod half of.
func TestThereIsNoLockFileOrVendorTree(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"go.sum", "vendor", "go.work", "go.work.sum"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			t.Errorf("%s exists. gliderGo has no dependencies, so nothing should have "+
				"created it; if a `go get` did, revert go.mod too.", name)
		}
	}
}

// ---------------------------------------------------------------------------
// The import graph
// ---------------------------------------------------------------------------

// TestEveryImportIsTheStandardLibraryOrThisModule is the check with teeth. go.mod can be clean
// while a file imports something -- the build then fails rather than silently downloading, but
// it fails with a message about a missing module, and this says the same thing in one line.
func TestEveryImportIsTheStandardLibraryOrThisModule(t *testing.T) {
	root := repoRoot(t)

	var files, imports int
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch {
			// GliderPRO is the vendored 1994 C source, read-only and full of no Go.
			case d.Name() == "GliderPRO":
				return fs.SkipDir
			// Generated or transient trees: extracted assets, build output, the toolchain
			// cache the bootstrap script writes.
			case d.Name() == ".git", d.Name() == "bin", d.Name() == ".toolchain":
				return fs.SkipDir
			// testdata is skipped for the same reason the go command skips it: files in it
			// are fixtures, not code, and are never built. A dependency hidden there could
			// not reach a binary.
			case d.Name() == "testdata":
				return fs.SkipDir
			// The go command ignores directories starting with _ or . -- so must this, or
			// the two disagree about what the module contains.
			case path != root && (strings.HasPrefix(d.Name(), "_") || strings.HasPrefix(d.Name(), ".")):
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			t.Errorf("%s: %v", rel(root, path), perr)
			return nil
		}
		files++

		for _, spec := range f.Imports {
			p, uerr := strconv.Unquote(spec.Path.Value)
			if uerr != nil {
				t.Errorf("%s: unparsable import %s", rel(root, path), spec.Path.Value)
				continue
			}
			imports++
			if ok, why := allowedImport(p); !ok {
				t.Errorf("%s:%d: imports %q -- %s\n\n"+
					"gliderGo is standard library only; see the comment at the top of "+
					"internal/module/stdlib_test.go.",
					rel(root, path), fset.Position(spec.Pos()).Line, p, why)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// A walk that silently found nothing would pass every assertion above, which is the one
	// way this test could rot into decoration. The floors are far below the real counts
	// (roughly 200 files and 700 imports when this was written), so they catch a broken walk
	// without needing an update every time a file is added.
	if files < 50 {
		t.Errorf("only walked %d .go files; the walk is broken, not the module", files)
	}
	if imports < 100 {
		t.Errorf("only saw %d imports across %d files; the walk is broken", imports, files)
	}
}

// allowedImport decides whether one import path is standard library, this module, or cgo's
// pseudo-package.
//
// The test for "standard library" is that the first element of the path has no dot in it.
// That is the same rule the go command itself uses to tell a standard library path from a
// module path -- a module path's first element is a domain name, and domain names have dots.
// It is exact rather than a heuristic: the standard library reserves every first element
// without a dot, which is precisely why module paths are required to have one.
func allowedImport(p string) (bool, string) {
	if p == "C" {
		// cgo. Real and used: internal/platform/x11 links libX11 through it. It is not a Go
		// dependency and cannot appear in go.mod.
		return true, ""
	}
	if p == modulePath || strings.HasPrefix(p, modulePath+"/") {
		return true, ""
	}
	first := p
	if i := strings.Index(p, "/"); i >= 0 {
		first = p[:i]
	}
	if strings.Contains(first, ".") {
		return false, "the first path element contains a dot, so this is an external module"
	}
	if strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../") {
		return false, "relative imports are not legal in a module"
	}
	// No dot and not ours: the standard library. If the name is misspelled the build says so
	// far more clearly than this test could.
	return true, ""
}

func rel(root, path string) string {
	if r, err := filepath.Rel(root, path); err == nil {
		return r
	}
	return path
}
