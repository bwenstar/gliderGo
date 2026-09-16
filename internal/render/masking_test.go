package render

// Which of the four masking strategies each object type is drawn with, counted.
//
// This is item 15 of the fidelity contract (docs/ORIGINAL_GAME.md §19), and it is the one
// contract item that is a *census* rather than a behaviour: the original has four painters and
// `graphics-assets.md` §5 measured which objects reach each one -- 21 colour-keyed PICTs
// (including `kCustomPict`'s default 10000), 9 opaque, and the mask-paired sheets. Getting an
// object into the wrong column is not subtle when you see it (a colour key punches 681 holes
// in the angel's white robe; a mask on a keyed PICT paints a white box) but it is completely
// invisible to every other test in this package, because both paths draw *something* of the
// right size in the right place and `TestComposeEveryRoom` only asks whether pixels changed.
//
// So this reads the dispatch instead of the pixels. `DrawARoomsObjects` is a transcription of
// `ObjectDrawAll.c`'s switch, one case group per object type, and the census is which painter
// each group calls. That makes the test fail on the change that is actually dangerous --
// moving a `case` label from one group to another, or adding a new object type to whichever
// group is nearest -- and not on a palette or geometry change, which the pixel tests own.
//
// The three procedural objects are asserted the same way but from the other side: they must
// call *no* painter, because their art does not exist as a resource at all (`kMirror` is a
// flood fill plus a frame, `kCounter` and `kWallWindow` are rectangles). An object that
// acquired a painter here would be an object whose art was invented.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// The four painters, named as they are called on the Scene receiver. The first two are the
// original's `DrawPictObject` / `DrawPictWithMaskObject` / `DrawPictSansWhiteObject` /
// `DrawCustPictSansWhite` (ObjectDraw2.c:1197, :1253, :1302, :1412); see objectdraw2.go for
// why the middle two collapse to one blit in this port and are still two functions here.
const (
	opaquePainter   = "DrawPictObject"
	maskedPainter   = "DrawPictWithMaskObject"
	keyedPainter    = "DrawPictSansWhiteObject"
	custPictPainter = "DrawCustPictSansWhite"
	calendarPainter = "DrawCalendar"
	bulletinPainter = "DrawBulletin"
)

func TestTheMaskingStrategyOfEveryObjectType(t *testing.T) {
	strategy := objectStrategies(t)

	// The 21 colour-keyed objects of graphics-assets.md §5.3 and line 1844's census: 20
	// object types through DrawPictSansWhiteObject, plus kCustomPict through the by-id
	// version of the same painter. The count is the assertion as much as the membership --
	// §19 row 15 says "the white colour key applies only to the 21 colour-keyed objects".
	keyed := []string{
		"kBBQ", "kManhole",
		"kUpStairs", "kDoorInLf", "kDoorInRt", "kWindowInLf", "kWindowInRt",
		"kTrunk", "kBooks", "kHipLamp", "kDecoLamp", "kGuitar", "kCinderBlock",
		"kFlowerBox", "kFireplace", "kBear", "kVase1", "kVase2", "kRug", "kChimes",
	}
	assertGroup(t, strategy, keyedPainter, keyed)
	assertGroup(t, strategy, custPictPainter, []string{"kCustomPict"})
	if got := len(keyed) + 1; got != 21 {
		t.Errorf("the colour-keyed census is %d types, want 21 (graphics-assets.md §5, line 1844)", got)
	}

	// The 9 opaque PICTs are 7 object types through DrawPictObject plus two that have their
	// own painter for the text they carry and are opaque underneath it: the calendar's month
	// name (STR# 1005) and the bulletin board. Counting them here rather than only in the
	// keyed/opaque split is what makes the total come out at §5's 9 rather than 7.
	assertGroup(t, strategy, opaquePainter, []string{
		"kFilingCabinet", "kOzma",
		"kDownStairs", "kDoorExRt", "kDoorExLf", "kWindowExRt", "kWindowExLf",
	})
	assertGroup(t, strategy, calendarPainter, []string{"kCalendar"})
	assertGroup(t, strategy, bulletinPainter, []string{"kBulletin"})

	// The two objects whose 1-bit mask carries information their colour plane does not.
	// Everything else with a mask is a *sheet* rather than an object PICT (§5.4's table of
	// eight pairs), which is why this group is two and not eight.
	assertGroup(t, strategy, maskedPainter, []string{"kCobweb", "kCloud"})

	// The three procedural objects: no painter, no PICT, no mask.
	for _, name := range []string{"kMirror", "kCounter", "kWallWindow"} {
		if painters, ok := strategy[name]; !ok {
			t.Errorf("%s is not a case in DrawARoomsObjects any more", name)
		} else if len(painters) > 0 {
			t.Errorf("%s now calls %s; it is drawn procedurally and has no art resource "+
				"(§19 row 15)", name, strings.Join(painters, ", "))
		}
	}
}

// assertGroup requires that exactly the named object types reach the named painter.
func assertGroup(t *testing.T, strategy map[string][]string, painter string, want []string) {
	t.Helper()
	var got []string
	for name, painters := range strategy {
		for _, p := range painters {
			if p == painter {
				got = append(got, name)
				break
			}
		}
	}
	sort.Strings(got)
	sorted := append([]string(nil), want...)
	sort.Strings(sorted)
	if strings.Join(got, " ") != strings.Join(sorted, " ") {
		t.Errorf("%s draws %d object types:\n  got  %s\n  want %s",
			painter, len(got), strings.Join(got, " "), strings.Join(sorted, " "))
	}
}

// objectStrategies reads DrawARoomsObjects and returns, for each object type its switch names,
// the painters that type's case group calls.
//
// The outer switch is the one whose tag is `thisObject.What`, and it is taken by source order:
// two case groups (the clocks and the appliances) switch on the same expression again inside
// themselves, and those inner cases belong to the outer group's census rather than being
// groups of their own. Walking each outer clause whole is what attributes them correctly.
//
// A group's painter list is every Scene method it calls whose name is one of the six above, so
// a case that draws two things -- the cuckoo's clock case draws its own body and registers a
// pendulum -- contributes both and neither is silently dropped.
func objectStrategies(t *testing.T) map[string][]string {
	t.Helper()
	const src = "locale.go"

	f, err := parser.ParseFile(token.NewFileSet(), src, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", src, err)
	}
	var body *ast.BlockStmt
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "DrawARoomsObjects" {
			body = fd.Body
		}
	}
	if body == nil {
		t.Fatal("no DrawARoomsObjects in " + src)
	}

	painters := map[string]bool{
		opaquePainter: true, maskedPainter: true, keyedPainter: true,
		custPictPainter: true, calendarPainter: true, bulletinPainter: true,
	}

	out := map[string][]string{}
	var found bool
	ast.Inspect(body, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok || found {
			return !found
		}
		sel, ok := sw.Tag.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "What" {
			return true
		}
		found = true
		for _, stmt := range sw.Body.List {
			clause, ok := stmt.(*ast.CaseClause)
			if !ok {
				continue
			}
			var called []string
			for _, s := range clause.Body {
				ast.Inspect(s, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if fn, ok := call.Fun.(*ast.SelectorExpr); ok && painters[fn.Sel.Name] {
						called = append(called, fn.Sel.Name)
					}
					return true
				})
			}
			for _, label := range clause.List {
				id, ok := label.(*ast.Ident)
				if !ok { // house.ObjectIsEmpty
					continue
				}
				out[id.Name] = called
			}
		}
		return false
	})
	if !found {
		t.Fatal("no `switch thisObject.What` in DrawARoomsObjects")
	}
	if len(out) < 60 {
		t.Fatalf("only %d object types found in the switch; the census cannot be right", len(out))
	}
	return out
}
