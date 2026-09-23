package project

// The half that makes package project worth having.
//
// Constants that six callers share are only an improvement over six literals if something
// checks that they are still the same as the tree around them. go.mod is the authority for the
// module path; README.md is the authority for what the project says it is in prose; and a link
// in a document is the one kind of string in this repository that nothing else validates -- a
// misspelled import path does not compile, while a misspelled URL is a dead link that ships.
//
// The sweep below reads the whole tree as text rather than asking the toolchain anything, for
// the same reason internal/module does: it has to hold on a machine with no network and no
// module cache.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot walks up from the package directory to the directory holding go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the package directory")
		}
		dir = parent
	}
}

// TestModuleMatchesGoMod is the one assertion that has an authority to check against. Home is
// built out of Module, so a module path that has moved and a Home that has not is not a
// possible state -- which is the reason Home is a concatenation and not its own literal.
func TestModuleMatchesGoMod(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	var got string
	for _, line := range strings.Split(string(b), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			got = strings.TrimSpace(rest)
			break
		}
	}
	if got == "" {
		t.Fatal("go.mod declares no module path")
	}
	if got != Module {
		t.Errorf("go.mod says %q and project.Module says %q; every URL in the About box, "+
			"the usage text and the issue templates is built out of the second one", got, Module)
	}
	if want := "https://" + Module; Home != want {
		t.Errorf("Home = %q, want %q", Home, want)
	}
}

// textFiles walks the tree and hands back every file worth reading as prose. The excluded
// directories are the ones whose contents are not authored here: the git database, the vendored
// 1994 data, the extracted assets (1,877 generated files) and build output.
func textFiles(t *testing.T) map[string]string {
	t.Helper()
	root := repoRoot(t)
	out := map[string]string{}

	skipDir := map[string]bool{
		".git": true, "GliderPRO": true, "assets": true, "bin": true, "__pycache__": true,
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// A NUL byte is the cheap and sufficient test for "not text", and it agrees with
		// what git itself uses to decide whether to print a diff.
		if strings.IndexByte(string(b), 0) >= 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 50 {
		t.Fatalf("only %d text files found; the walk is not reaching the tree", len(out))
	}
	return out
}

// linkRE matches a GitHub repository link as a reader would click it, which is deliberately
// narrower than what matches an import path: the https:// prefix is what distinguishes "a link
// somebody will follow" from "a Go package path the compiler already checked".
//
// The dot has to be in the character class, because GitHub repository names may contain one
// (`someone/someone.github.io` is the common case), and that means a URL at the end of an English
// sentence matches with the full stop attached. The first run of this test found exactly that in
// SECURITY.md and reported a link that differed from the real one by a period, which is a false
// positive of the worst kind: it accuses a correct document. `trimLink` takes the punctuation back
// off. A repository name cannot end in a dot, so nothing legitimate is lost.
var linkRE = regexp.MustCompile(`https://github\.com/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)`)

// trimLink strips the trailing punctuation that prose leaves glued to a URL. Markdown adds its
// own: a link inside `[text](url)` cannot end in `)`, and one at the end of a table cell or a
// parenthetical routinely does.
func trimLink(url string) string {
	return strings.TrimRight(url, ".,;:!?)\"'")
}

// TestEveryGitHubLinkIsOneWeMean sweeps the tree for repository links and requires each to be
// one of the two this project has any business naming: its own, and the upstream C it is a port
// of. Anything else is either a typo or a dependency that arrived without a decision.
//
// This is the check that would have caught the one real risk in shipping a URL inside the game:
// the About box tells a player where to file a bug, and a link that 404s sends them nowhere
// while looking exactly as authoritative as one that works.
func TestEveryGitHubLinkIsOneWeMean(t *testing.T) {
	allowed := map[string]bool{
		Home:     true,
		Upstream: true,
	}

	seen := map[string][]string{}
	for name, body := range textFiles(t) {
		for _, m := range linkRE.FindAllStringSubmatch(body, -1) {
			// The pattern stops at the second path segment, so a deep link -- into a file, a
			// release, an issue -- reduces to the repository root and is judged as that root.
			// Which is the whole question here: this test asks which repositories the tree
			// names, not which pages of them.
			url := trimLink(m[0])
			if !allowed[url] {
				seen[url] = append(seen[url], name)
			}
		}
	}
	for url, files := range seen {
		t.Errorf("%s is linked from %v and is neither project.Home (%s) nor "+
			"project.Upstream (%s); if it is meant, add it to this test's allowlist",
			url, files, Home, Upstream)
	}
}

// TestTheReadmeAndTheGameAgree pins the facts the About box states against the README, which is
// the other document a reader forms an impression from. They are two audiences for one set of
// claims -- somebody who found the repository and somebody who only has the binary -- and the
// second one cannot check the first.
func TestTheReadmeAndTheGameAgree(t *testing.T) {
	readme := textFiles(t)["README.md"]
	if readme == "" {
		t.Fatal("no README.md")
	}
	for _, c := range []struct{ what, value string }{
		{"Home", Home},
		{"Upstream", Upstream},
		{"UpstreamCommit", UpstreamCommit},
		{"Original", Original},
		{"OriginalAuthor", OriginalAuthor},
		{"OriginalPublisher", OriginalPublisher},
		{"OriginalYear", OriginalYear},
		{"Copyright", Copyright},
		{"GoLicence", GoLicence},
	} {
		if !strings.Contains(readme, c.value) {
			t.Errorf("README.md does not mention project.%s (%q), so the game and the "+
				"repository are telling a reader different things", c.what, c.value)
		}
	}
}

// TestTheOriginalsCopyrightIsQuotedAndNotParaphrased is the other constant with a document
// behind it, and the only one whose document is a 1994 resource rather than anything written
// here. DITL 150 item 6 is a statText with fixed content; docs/analysis/ui-dialogs.md 10.6
// transcribes it inside backticks, and a notice that has been tidied up -- an ASCII "(c)", a
// dropped "Inc.", 1995 for 2000 -- is no longer the notice GPLv2 section 1 asks to be kept
// intact. So it is compared to the transcription character for character.
func TestTheOriginalsCopyrightIsQuotedAndNotParaphrased(t *testing.T) {
	doc := textFiles(t)[filepath.Join("docs", "analysis", "ui-dialogs.md")]
	if doc == "" {
		t.Fatal("docs/analysis/ui-dialogs.md is the source for OriginalCopyright and is missing")
	}
	if !strings.Contains(doc, "`"+OriginalCopyright+"`") {
		t.Errorf("ui-dialogs.md does not quote OriginalCopyright (%q) as the About dialog's "+
			"statText; DITL 150 item 6 is the authority and section 10.6 transcribes it",
			OriginalCopyright)
	}
}

// TestEveryExportedConstantHereIsReadBySomething turns this package's own argument into a check.
// The doc comment says these constants exist because a fact spelled out separately in six places
// will eventually disagree with itself -- which is true of a constant with six readers and says
// nothing about one with none. An exported constant nobody reads is not a shared fact; it is a
// string, in a package whose every other string is compared to something, and it inherits the
// credibility of its neighbours without earning it.
//
// It was found rather than imagined. `Releases` was first here for a release page that did not
// exist yet: spelled correctly, built out of Home like the rest, and read by nothing, so no test
// could have told anyone if it had been wrong (docs/IMPROVEMENTS.md 4.13). It went, and came back
// with the page, read by -version and -help (5.4); this is what stops the next one.
//
// go/parser for the declarations and a text sweep for the readers, and the split is deliberate. A
// list of what this package exports has to be exact, and gofmt's column alignment is not something
// to build a regexp on. A reader, on the other hand, is exactly `project.X` written somewhere else
// -- no type information, no module cache and no network, which is what internal/module settled on
// for the same reason. The one thing the sweep cannot see is an aliased import, and nothing in this
// tree aliases this package; if something ever does, this test will ask for a reader that is
// already there, which is a failure that explains itself.
func TestEveryExportedConstantHereIsReadBySomething(t *testing.T) {
	root := repoRoot(t)
	self := filepath.Join(root, "internal", "project", "project.go")
	f, err := parser.ParseFile(token.NewFileSet(), self, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var declared []string
	uses := map[string][]string{} // constant -> the constants its own value is built from
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, s := range g.Specs {
			v, ok := s.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, n := range v.Names {
				if !n.IsExported() {
					continue
				}
				declared = append(declared, n.Name)
				for _, val := range v.Values {
					ast.Inspect(val, func(n2 ast.Node) bool {
						if id, ok := n2.(*ast.Ident); ok && id.IsExported() {
							uses[n.Name] = append(uses[n.Name], id.Name)
						}
						return true
					})
				}
			}
		}
	}
	// A floor, because a parse that silently found nothing would pass every case below. There are
	// fourteen as this is written, in two const blocks.
	if len(declared) < 10 {
		t.Fatalf("parsed only %d exported constants out of project.go, which has two const "+
			"blocks of them; the parse is not reaching the declarations", len(declared))
	}

	// Every .go file in the tree except this package's own two, which read the constants
	// unqualified and so say nothing about whether anything outside uses them.
	elsewhere := map[string]string{}
	for name, body := range textFiles(t) {
		if filepath.Ext(name) == ".go" && filepath.Dir(name) != filepath.Join("internal", "project") {
			elsewhere[name] = body
		}
	}
	read := map[string]bool{}
	for _, name := range declared {
		re := regexp.MustCompile(`\bproject\.` + name + `\b`)
		for _, body := range elsewhere {
			if re.MatchString(body) {
				read[name] = true
				break
			}
		}
	}

	// A constant that only one of its neighbours reads counts, as long as that neighbour is read:
	// `Home` is `"https://" + Module`, so `Module` has no caller of its own and is still the string
	// every URL in the game is built out of. Reached by walking the dependency edges from the
	// constants that do have callers, which terminates because a const block cannot be cyclic.
	for changed := true; changed; {
		changed = false
		for name := range read {
			for _, dep := range uses[name] {
				if !read[dep] {
					read[dep], changed = true, true
				}
			}
		}
	}

	for _, name := range declared {
		if !read[name] {
			t.Errorf("project.%s is exported, nothing outside this package reads it, and no "+
				"constant that is read is built from it -- so nothing checks it either; give it "+
				"the caller it was written for, or delete it and say in docs/IMPROVEMENTS.md when "+
				"it comes back", name)
		}
	}
}

// TestLicenceIsGPLv2Only guards the one constant with a legal consequence. Upstream's grant
// names version 2 with no "or any later version" clause (README.md's Licence section quotes
// it), so an SPDX identifier ending in `-or-later` here would be a claim nobody made.
func TestLicenceIsGPLv2Only(t *testing.T) {
	if Licence != "GPL-2.0-only" {
		t.Errorf("Licence = %q; upstream's grant has no "+
			"\"or any later version\" clause, so this port is GPLv2-only", Licence)
	}
	licence := textFiles(t)["LICENSE"]
	if !strings.Contains(licence, "GNU GENERAL PUBLIC LICENSE") ||
		!strings.Contains(licence, "Version 2, June 1991") {
		t.Error("LICENSE is not the GPLv2 text that project.Licence claims it is")
	}
}
