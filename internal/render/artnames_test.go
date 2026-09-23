package render

// Every name the code asks Assets for, held to the table that answers it.
//
// Three accessors take a name: Sheet, Strip and Object. None of them stops anything on a name
// it does not know. Each records the first failure, returns nil and lets the draw skip, which
// is right for a half-extracted tree and wrong for a slip of the pen, because the slip then
// becomes a picture that is not there and an error nobody sees until the game ends.
// RenderShreds asked Sheet for "shred", which is a strip, from the commit that wrote it until
// the soak found it (docs/IMPROVEMENTS.md 2.73). Nothing in this package could have seen it.
// sheetSize in srcrects_test.go resolves a name in either table on purpose, and
// TestComposeEveryRoom composes rooms, not the animations that play in them.
//
// So this reads the calls rather than making them. Every non-test Go file in the module is
// parsed, every argument to the three accessors that is a constant is evaluated, and the value
// is looked up in the table that accessor reads:
//
//	Sheet(name)   sheetBounds, the sheets srcRects indexes into
//	Strip(name)   stripBounds, the strips reached through named globals
//	Object(what)  the atlas, where the kind must be artKey, artOpaque or artPair
//
// Object is the one a lookup alone would not catch. It fails only for a `what` with no name,
// and the extractor writes a crop of every artSheet object into object/ as well
// (tools/extract_art.py), so Object(kFloorVent) would load a picture and put it through the
// wrong painter with no error anywhere. Object's own comment says the sheet objects do not
// come through it; this is what makes that true.
//
// # Forwarders
//
// Most names never reach an accessor directly. maskSheet, opaqueSheet and bakeStrip take a
// sheet name and pass it on, and maskObject and DrawPictObject do the same with a `what`. A
// function that passes one of its own parameters straight to an accessor, or to another such
// function, is a forwarder for that parameter, and its callers are checked as though they had
// called the accessor. The forwarders are found by reading their bodies, not listed here,
// because a list is what goes stale: the next helper would be written, its callers would go
// unchecked, and this test would still pass.
//
// An argument that is not a constant -- thisObject.What, a loop variable -- is counted and left
// alone. It is data, and data is TestComposeEveryRoom's business.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
)

func TestEveryArtNameIsInItsTable(t *testing.T) {
	// A name in both tables would satisfy Sheet and Strip alike, and a call for it with the
	// wrong one would pass everything below.
	for name := range sheetBounds {
		if _, ok := stripBounds[name]; ok {
			t.Errorf("%q is in both sheetBounds and stripBounds: the two load different "+
				"files, and a call with the wrong accessor for it cannot be caught", name)
		}
	}
	if len(artKindNames) != int(artProc)+1 {
		t.Fatalf("artKindNames has %d names and artKind has %d values", len(artKindNames),
			int(artProc)+1)
	}

	// The atlas carries a sheet name for every object drawn out of one. Nothing loads through
	// that field, but the object draws use the sheet it names, and a row naming a strip or a
	// misspelling would be the table disagreeing with the code.
	for _, e := range atlas {
		if e.kind != artSheet && e.kind != artTmpl {
			continue
		}
		if _, ok := sheetBounds[e.sheet]; !ok {
			t.Errorf("the atlas row for %s is %s from %q, which is not in sheetBounds",
				house.ObjectName(e.what), artKindNames[e.kind], e.sheet)
		}
	}

	m := parseArtModule(t)
	if len(m.files) < 100 {
		t.Fatalf("parsed %d files; the module has about 150 non-test files, so the walk is not "+
			"reaching it", len(m.files))
	}
	r := m.check()
	for _, p := range r.problems {
		t.Error(p)
	}

	// A walk that found nothing passes everything, so each part has to have done something.
	for k := sheetTable; k <= objectTable; k++ {
		if r.checked[k] == 0 {
			t.Errorf("no constant reaches %s: either the module stopped using it or this "+
				"test stopped finding the calls, and either way it proved nothing", k.accessor())
		}
	}
	if r.inGame == 0 {
		t.Error("no checked name is in internal/game, which is where the shred bug was: the " +
			"walk is not reaching past this package")
	}
	t.Logf("%d constant names checked (%d sheet, %d strip, %d object), %d of them in "+
		"internal/game; %d arguments forwarded and %d not constant; forwarders %s",
		r.checked[sheetTable]+r.checked[stripTable]+r.checked[objectTable],
		r.checked[sheetTable], r.checked[stripTable], r.checked[objectTable], r.inGame,
		r.forwarded, r.dynamic, strings.Join(r.forwarderNames(), " "))
}

// TestTheArtNameCheckCatchesEachWayANameCanBeWrong is the test that keeps the one above honest.
//
// The module passes it now, which is exactly what an empty loop would also do. So the checker
// is handed a small module of its own with the mistakes planted in it, the first being the one
// that was really committed, and each has to come back as a problem that names it. The lines
// that are right have to come back as nothing, or the check is noise.
//
// The planted files are named without .go on purpose. internal/citations holds every path in
// this repository's text that looks like one of our files to being one, and these are not.
func TestTheArtNameCheckCatchesEachWayANameCanBeWrong(t *testing.T) {
	m := newArtModule()
	for rel, src := range map[string]string{
		"internal/game/planted": `package game

import "github.com/bwenstar/gliderGo/internal/render"

func (w *World) RenderShreds() { w.R.A.Sheet("shred") }
func (w *World) wrongWay()     { w.R.A.Strip("appliance") }
func (w *World) neither()      { w.R.A.Sheet("shreds") }
func (w *World) imported()     { w.R.A.Sheet(render.ShredName) }
func (w *World) right()        { w.R.A.Strip("shred"); w.R.A.Sheet("appliance") }
func (w *World) data(n string) { w.R.A.Sheet(n + "s") }
`,
		"internal/render/planted": `package render

const ShredName = "shred"

const (
	kFloorVent = 0x01
	kMailboxLf = 0x33
	sheetName  = "toast"
)

func (s *Scene) maskSheet(sheet string, src, dst Rect) { s.A.Sheet(sheet) }
func (s *Scene) paint(dst Rect, name string)           { s.maskSheet(name, Rect{}, dst) }
func (s *Scene) maskObject(what int16, src, dst Rect)  { s.A.Object(what) }

func (s *Scene) viaTwo()     { s.paint(Rect{}, "fish") }
func (s *Scene) viaConst()   { s.maskSheet(sheetName, Rect{}, Rect{}) }
func (s *Scene) sheetObject() { s.maskObject(kFloorVent, Rect{}, Rect{}) }
func (s *Scene) noObject()   { s.A.Object(0x7F) }
func (s *Scene) fine()       { s.paint(Rect{}, "blower"); s.maskObject(kMailboxLf, Rect{}, Rect{}) }
`,
		"internal/other/planted": `package other

type Book struct{}

func (b *Book) Sheet(name string) {}
func maskSheet()                  {}
`,
	} {
		if err := m.add(rel, src); err != nil {
			t.Fatalf("parse %s: %v", rel, err)
		}
	}
	r := m.check()

	// Line numbers are part of the want: a problem reported at the wrong line would send
	// somebody to read the wrong call.
	wants := []struct{ why, at, says string }{
		{"a strip asked for as a sheet", "internal/game/planted:5",
			`Sheet is given "shred", which is in stripBounds and not sheetBounds: it wants Strip`},
		{"a sheet asked for as a strip", "internal/game/planted:6",
			`Strip is given "appliance", which is in sheetBounds and not stripBounds: it wants Sheet`},
		{"a name in neither table", "internal/game/planted:7", `"shreds", which is in neither table`},
		{"a constant from another package", "internal/game/planted:8", `"shred", which is in stripBounds`},
		{"a strip through two forwarders", "internal/render/planted:15",
			`paint, which reaches Sheet, is given "fish"`},
		{"a strip through a local constant", "internal/render/planted:16", `"toast", which is in stripBounds`},
		{"a sheet object through Object", "internal/render/planted:17", "kFloorVent, which is artSheet"},
		{"a what with no object", "internal/render/planted:18", "0x7F, which is no object"},
		{"a second type with an accessor's name", "internal/other/planted", "declares Book.Sheet"},
		{"a forwarder declared twice", "maskSheet", "declared 2 times"},
	}
	used := make([]bool, len(r.problems))
	for _, w := range wants {
		found := false
		for i, p := range r.problems {
			if !used[i] && strings.HasPrefix(p, w.at) && strings.Contains(p, w.says) {
				used[i], found = true, true
				break
			}
		}
		if !found {
			t.Errorf("%s is not caught: want a problem at %s saying %q", w.why, w.at, w.says)
		}
	}
	for i, p := range r.problems {
		if !used[i] {
			t.Errorf("a problem nothing was planted for: %s", p)
		}
	}

	// And the counts say the right lines were read rather than skipped: in each file, one
	// constant on each of the four wrong lines and two on the line that is right.
	if got := r.checked[stripTable] + r.checked[sheetTable] + r.checked[objectTable]; got != 12 {
		t.Errorf("%d constants checked, want 12", got)
	}
	if r.dynamic != 1 {
		t.Errorf("%d arguments counted as not constant, want 1 (data's `n + \"s\"`)", r.dynamic)
	}
}

// artTable is which lookup answers a name.
type artTable int

const (
	sheetTable artTable = iota
	stripTable
	objectTable
)

// accessor is the Assets method that reads the table, for messages.
func (k artTable) accessor() string { return [...]string{"Sheet", "Strip", "Object"}[k] }

// artAccessors are the three methods, by name. Each takes the name as its only parameter.
var artAccessors = map[string]artTable{
	"Sheet":  sheetTable,
	"Strip":  stripTable,
	"Object": objectTable,
}

// artKindNames is artKind's const block as text, in its order.
var artKindNames = [...]string{"artNone", "artSheet", "artTmpl", "artKey", "artOpaque", "artPair", "artProc"}

// artParam is one parameter that ends up as a name: where it is in the parameter list and
// which table the name is looked up in at the end of the chain.
type artParam struct {
	index int
	table artTable
}

// artFile is one parsed file and what is needed to read a constant in it.
type artFile struct {
	rel     string            // slash path from the module root
	dir     string            // the package, which is what scopes a constant
	f       *ast.File         // parsed without object resolution; constant does what little is needed
	imports map[string]string // local name to package dir, for this module's imports only
}

// artModule is the module's source, parsed.
type artModule struct {
	root  string // the directory holding go.mod, which add reads from; empty for a planted module
	fset  *token.FileSet
	files []*artFile

	// consts is every package-level constant: package dir, then name, then the expression
	// it was declared with. An iota continuation has no expression of its own and is nil.
	consts map[string]map[string]ast.Expr

	// declared counts function declarations by name, because forwarders are matched by
	// name and a name declared twice would make that ambiguous.
	declared map[string]int
}

// artReport is what check found.
type artReport struct {
	problems   []string
	checked    map[artTable]int
	forwarders map[string][]artParam
	forwarded  int // arguments that are the enclosing forwarder's own parameter
	dynamic    int // arguments that are not constants
	inGame     int // checked names in internal/game
}

// forwarderNames lists the forwarders as name->Accessor, sorted, for the log line.
func (r artReport) forwarderNames() []string {
	var out []string
	for name, ps := range r.forwarders {
		for _, p := range ps {
			out = append(out, name+"->"+p.table.accessor())
		}
	}
	sort.Strings(out)
	return out
}

func newArtModule() *artModule {
	return &artModule{
		fset:     token.NewFileSet(),
		consts:   map[string]map[string]ast.Expr{},
		declared: map[string]int{},
	}
}

// modulePath is go.mod's module line, the prefix that marks an import as one of ours.
const modulePath = "github.com/bwenstar/gliderGo/"

// add parses one file, named by its slash path from the module root, and indexes its imports,
// constants and function names. src is what parser.ParseFile takes: nil to read rel from
// disk, or the source itself.
func (m *artModule) add(rel string, src any) error {
	path := rel
	if src == nil {
		path = filepath.Join(m.root, filepath.FromSlash(rel))
	}
	f, err := parser.ParseFile(m.fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return err
	}
	af := &artFile{rel: rel, dir: pathDir(rel), f: f, imports: map[string]string{}}
	for _, spec := range f.Imports {
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil || !strings.HasPrefix(p, modulePath) {
			continue
		}
		local := p[strings.LastIndex(p, "/")+1:]
		if spec.Name != nil {
			local = spec.Name.Name
		}
		af.imports[local] = strings.TrimPrefix(p, modulePath)
	}
	consts := m.consts[af.dir]
	if consts == nil {
		consts = map[string]ast.Expr{}
		m.consts[af.dir] = consts
	}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			m.declared[d.Name.Name]++
		case *ast.GenDecl:
			if d.Tok != token.CONST {
				continue
			}
			for _, spec := range d.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, n := range vs.Names {
					var v ast.Expr
					if i < len(vs.Values) {
						v = vs.Values[i]
					}
					consts[n.Name] = v
				}
			}
		}
	}
	m.files = append(m.files, af)
	return nil
}

// pathDir is path.Dir for the slash paths add is given.
func pathDir(rel string) string {
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[:i]
	}
	return "."
}

// check runs every rule over the parsed files.
func (m *artModule) check() artReport {
	r := artReport{checked: map[artTable]int{}}

	// The accessors are recognised by method name, which is only sound while nothing else in
	// the module has a method by one of those names.
	for _, af := range m.files {
		for _, d := range af.f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv == nil {
				continue
			}
			if _, ok := artAccessors[fn.Name.Name]; ok && receiverName(fn) != "Assets" {
				r.problems = append(r.problems, af.rel+": declares "+receiverName(fn)+"."+
					fn.Name.Name+", and this test finds the accessors by name alone: it now "+
					"needs to know the receiver's type")
			}
		}
	}

	r.forwarders = m.forwarders()
	var names []string
	for name := range r.forwarders {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if n := m.declared[name]; n != 1 {
			r.problems = append(r.problems, name+" forwards an art name and is declared "+
				strconv.Itoa(n)+" times: forwarders are matched by name, so this test "+
				"cannot tell which calls reach an accessor")
		}
	}

	for _, af := range m.files {
		for _, d := range af.f.Decls {
			params := map[string]int{}
			if fn, ok := d.(*ast.FuncDecl); ok {
				params = paramIndex(fn)
			}
			ast.Inspect(d, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				callee, targets := m.targets(af, call, r.forwarders)
				for _, p := range targets {
					if p.index >= len(call.Args) {
						continue
					}
					arg := call.Args[p.index]
					if id, ok := arg.(*ast.Ident); ok {
						if _, ok := params[id.Name]; ok {
							r.forwarded++
							continue
						}
					}
					lit, name, isConst := m.constant(af, arg)
					if !isConst {
						r.dynamic++
						continue
					}
					at := af.rel + ":" + strconv.Itoa(m.fset.Position(arg.Pos()).Line)
					problem, read := checkArtName(callee, p.table, lit, name)
					if problem != "" {
						r.problems = append(r.problems, at+": "+problem)
					}
					if read {
						r.checked[p.table]++
						if strings.HasPrefix(af.dir, "internal/game") {
							r.inGame++
						}
					}
				}
				return true
			})
		}
	}
	return r
}

// checkArtName holds one constant to its table. It returns the problem, if any, and whether
// the constant could be read at all.
func checkArtName(callee string, table artTable, lit *ast.BasicLit, name string) (problem string, read bool) {
	via := table.accessor()
	if callee != via {
		via = callee + ", which reaches " + via + ","
	}

	if table == objectTable {
		var what int16
		if lit != nil && lit.Kind == token.INT {
			v, err := strconv.ParseInt(lit.Value, 0, 16)
			if err != nil {
				return via + " is given " + lit.Value + ", which is not an int16", false
			}
			what = int16(v)
		} else {
			code, ok := house.ObjectCode(name)
			if !ok {
				return via + " is given the constant " + name + ", which this test cannot " +
					"evaluate", false
			}
			what = code
		}
		label := house.ObjectName(what)
		if label == "" {
			return via + " is given " + name + ", which is no object", true
		}
		if what < 0 || int(what) >= len(objArt) || objArt[what].what != what {
			return via + " is given " + label + ", which has no atlas row", true
		}
		switch k := objArt[what].kind; k {
		case artKey, artOpaque, artPair:
			return "", true
		default:
			return via + " is given " + label + ", which is " + artKindNames[k] + ": its " +
				"pixels are not its own PICT, and Object would load the extractor's crop of " +
				"them and draw it the wrong way", true
		}
	}

	if lit == nil || lit.Kind != token.STRING {
		return via + " is given the constant " + name + ", which this test cannot evaluate", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return via + " is given " + lit.Value + ": " + err.Error(), false
	}
	want, other := sheetBounds, stripBounds
	wantName, otherName, otherAccessor := "sheetBounds", "stripBounds", "Strip"
	if table == stripTable {
		want, other = stripBounds, sheetBounds
		wantName, otherName, otherAccessor = "stripBounds", "sheetBounds", "Sheet"
	}
	q := strconv.Quote(s)
	if _, ok := want[s]; ok {
		return "", true
	}
	if _, ok := other[s]; ok {
		return via + " is given " + q + ", which is in " + otherName + " and not " + wantName +
			": it wants " + otherAccessor, true
	}
	return via + " is given " + q + ", which is in neither table, so nothing can load it", true
}

// forwarders finds every function that passes one of its own parameters straight to an
// accessor or to another forwarder, by name, repeating until a pass finds nothing new.
func (m *artModule) forwarders() map[string][]artParam {
	fwd := map[string][]artParam{}
	for grew := true; grew; {
		grew = false
		for _, af := range m.files {
			for _, d := range af.f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				params := paramIndex(fn)
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					_, targets := m.targets(af, call, fwd)
					for _, p := range targets {
						if p.index >= len(call.Args) {
							continue
						}
						id, ok := call.Args[p.index].(*ast.Ident)
						if !ok {
							continue
						}
						i, ok := params[id.Name]
						if !ok {
							continue
						}
						found := artParam{i, p.table}
						name := fn.Name.Name
						known := false
						for _, q := range fwd[name] {
							known = known || q == found
						}
						if !known {
							fwd[name] = append(fwd[name], found)
							grew = true
						}
					}
					return true
				})
			}
		}
	}
	return fwd
}

// targets says which of a call's arguments end up as names, and what the call is called.
// An accessor is a method, so a package-qualified call is never one; a forwarder can be
// either.
func (m *artModule) targets(af *artFile, call *ast.CallExpr, fwd map[string][]artParam) (string, []artParam) {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		name := fun.Sel.Name
		if x, ok := fun.X.(*ast.Ident); ok {
			if _, isPkg := af.imports[x.Name]; isPkg {
				return name, fwd[name]
			}
		}
		if k, ok := artAccessors[name]; ok && len(call.Args) == 1 {
			return name, []artParam{{0, k}}
		}
		return name, fwd[name]
	case *ast.Ident:
		return fun.Name, fwd[fun.Name]
	}
	return "", nil
}

// constant evaluates an argument as far as this test needs to: a literal, or an identifier
// naming a package-level constant of the file's own package or of one the file imports from
// this module. isConst false is an argument that is not a constant at all. A constant whose
// declaration is not a literal comes back with lit nil and its name, so the caller can try
// the one other thing it knows, which is house.ObjectCode for an object's C name.
func (m *artModule) constant(af *artFile, e ast.Expr) (lit *ast.BasicLit, name string, isConst bool) {
	if b, ok := e.(*ast.BasicLit); ok {
		return b, b.Value, true
	}
	dir := af.dir
	switch x := e.(type) {
	case *ast.Ident:
		name = x.Name
	case *ast.SelectorExpr:
		pkg, ok := x.X.(*ast.Ident)
		if !ok {
			return nil, "", false
		}
		if dir, ok = af.imports[pkg.Name]; !ok {
			return nil, "", false
		}
		name = x.Sel.Name
	default:
		return nil, "", false
	}
	v, ok := m.consts[dir][name]
	if !ok {
		return nil, "", false
	}
	b, _ := v.(*ast.BasicLit)
	return b, name, true
}

// paramIndex numbers a function's parameters, receiver excluded, in declaration order.
func paramIndex(fn *ast.FuncDecl) map[string]int {
	out := map[string]int{}
	i := 0
	for _, field := range fn.Type.Params.List {
		if len(field.Names) == 0 {
			i++
			continue
		}
		for _, n := range field.Names {
			out[n.Name] = i
			i++
		}
	}
	return out
}

// receiverName is the type a method is declared on, pointer or not.
func receiverName(fn *ast.FuncDecl) string {
	e := fn.Recv.List[0].Type
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	if idx, ok := e.(*ast.IndexExpr); ok {
		e = idx.X
	}
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// parseArtModule parses every non-test Go file in the module, skipping what the go command
// skips. The directory rules are internal/module's stdlib walk, for its reasons.
func parseArtModule(t *testing.T) *artModule {
	t.Helper()
	m := newArtModule()
	m.root = repoRoot(t)
	err := filepath.WalkDir(m.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch n := d.Name(); {
			case n == "GliderPRO", n == ".git", n == "bin", n == ".toolchain", n == "testdata":
				return fs.SkipDir
			case path != m.root && (strings.HasPrefix(n, "_") || strings.HasPrefix(n, ".")):
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(m.root, path)
		if err != nil {
			return err
		}
		return m.add(filepath.ToSlash(rel), nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// repoRoot is the directory holding go.mod, found by walking up from this package.
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
