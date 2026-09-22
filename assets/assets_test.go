package assets

// What the executables are carrying, checked against what is committed.
//
// This is the only test in the repository that links the two archives, and it is here because it
// is the only one whose subject is the archives. Everything under internal/ reads the working copy
// of extracted/ instead, so `go test ./internal/...` links none of these 11 MiB.
//
// For the levels archive that is not a nicety, it is the entire coverage. Nothing under internal/
// can see the built-in houses of this port -- internal/shell takes its sources from whoever calls
// it and internal/fidelity builds its own -- so `make check` passed unchanged on the commit that
// first embedded them. A wrongly-prefixed archive and a stale one are both invisible from there
// all the way to the player: the game would list twenty-two houses, or an old build of the
// twenty-third, and say nothing. The last two tests in this file are what notice.

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/assetpack"
	"github.com/bwenstar/gliderGo/internal/house"
)

// tree is the extracted tree beside this file, as a path for the on-disk half of the comparison.
const treeDir = "extracted"

// levelSources is where the port's own houses are authored: text, one file per house, committed.
//
// It is reached with a relative path out of this package because it is not in it. assets/levels/
// -- the directory `make levels` builds and the archive is packed from -- is deliberately not
// committed (.gitignore says why), so a test that compared the archive against *that* would skip
// in a fresh clone and in CI, which is every run that matters. The text is what is always there.
const levelSources = "../levels"

// TestArchiveIsTheTree is the reason two copies of the assets can be committed without one of
// them going stale.
//
// The archive is generated from the tree by `make assets-zip` and nothing enforces that anybody
// ran it: an extractor change, or a hand-edited PNG, leaves a tree that is right and an archive
// that is what the game will actually draw. So the check is here, in the ordinary test run, and
// it compares contents rather than archive bytes -- see assetpack.Compare for why.
func TestArchiveIsTheTree(t *testing.T) {
	if _, err := fs.Stat(tree, "manifest.json"); err != nil {
		t.Fatalf("the built-in archive has no manifest.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(treeDir, "manifest.json")); err != nil {
		t.Skipf("no extracted assets at %s (run `make assets`)", treeDir)
	}

	diffs, err := assetpack.Compare(tree, treeDir, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range diffs {
		t.Errorf("%s", d)
	}
	if len(diffs) > 0 {
		t.Fatalf("%s does not match %s: run `make assets-zip`", assetpack.Name, treeDir)
	}
}

// TestBuiltInRootsAreThere pins the four roots the game asks for by name, because a build that
// has lost one of them starts, draws nothing and says the art is missing -- a bug report that
// looks like a renderer fault and is a packaging fault.
func TestBuiltInRootsAreThere(t *testing.T) {
	for _, name := range []string{
		"art/manifest.json",
		"art/ui/1000.png",
		"sound/manifest.tsv",
		"houses/Demo House.house",
		"houseart/manifest.json",
		"res/demo/128.bin", // the attract-mode input stream, which the shell replays
	} {
		if _, err := fs.Stat(tree, name); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestApostropheNamesSurvived is the regression test for the trap that decided the archive.
//
// go:embed accepts only names that are valid module file paths, which rules out an apostrophe. A
// pattern naming one of these files fails the build, which is survivable; a pattern naming its
// *directory* skips it in silence, which is not -- that is a game missing three of its
// twenty-two houses and three houses' worth of custom art, with nothing anywhere to say so.
//
// The archive has no such rule, and this is what says the archive still has no such rule. It
// does not skip when the tree is absent: the claim is about the bytes in this binary.
func TestApostropheNamesSurvived(t *testing.T) {
	for _, name := range []string{"Castle o' the Air", "Nemo's Market", "Rainbow's End"} {
		if _, err := fs.Stat(tree, path.Join("houses", name+".house")); err != nil {
			t.Errorf("house %q is not in the archive: %v", name, err)
		}
		fork := path.Join("houseart", name)
		st, err := fs.Stat(tree, fork)
		if err != nil {
			t.Errorf("%s is not in the archive: %v", fork, err)
			continue
		}
		if !st.IsDir() {
			t.Errorf("%s is not a directory", fork)
		}
	}
}

// TestHouseCount is the whole shipped set, counted. Twenty-two houses went into the extraction
// and twenty-two have to come out of the executable; the three above are the ones most likely to
// go missing, but the count is what notices any of them going.
func TestHouseCount(t *testing.T) {
	ents, err := fs.ReadDir(tree, "houses")
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 22 {
		t.Errorf("the archive holds %d houses, want 22", len(ents))
	}
	// And no .rsrc: the per-house resource forks are extractor input, 24 MB of it, and
	// assetpack.Skip keeps them out. One slipping in would double the download.
	for _, e := range ents {
		if path.Ext(e.Name()) == ".rsrc" {
			t.Errorf("%s is in the archive; the resource forks are not shipped", e.Name())
		}
	}
}

// TestLevelArchiveHoldsEveryAuthoredHouseAtItsRoot is the shape of the second archive.
//
// Two things can be wrong with it and neither is visible from anywhere else. A member packed one
// directory down -- "levels/Open House.house", which is what a `-tree assets` pack would write --
// leaves a levels root whose only entry is a directory, so shell.Discover walks into it, finds the
// house and lists it with a path nobody can read; or, if the prefix is something the walk skips,
// finds nothing and says nothing. And a house authored but never packed is simply absent. Both
// ship a game that starts, plays and is missing a level.
//
// So this compares the two lists by name and does not skip: levels/*.house.txt is committed, the
// archive is committed, and the claim is about the bytes in this binary.
func TestLevelArchiveHoldsEveryAuthoredHouseAtItsRoot(t *testing.T) {
	srcs, err := filepath.Glob(filepath.Join(levelSources, "*.house.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(srcs) == 0 {
		t.Fatalf("no houses authored in %s, so this archive should not exist", levelSources)
	}
	want := make([]string, 0, len(srcs))
	for _, src := range srcs {
		want = append(want, strings.TrimSuffix(filepath.Base(src), ".txt"))
	}
	sort.Strings(want)

	ents, err := fs.ReadDir(levelTree, ".")
	if err != nil {
		t.Fatalf("the levels archive will not list: %v", err)
	}
	got := make([]string, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() {
			t.Errorf("%s/ is a directory at the root of %s: the houses go in flat, "+
				"one level set is one root (internal/assetpack.LevelsName)",
				e.Name(), assetpack.LevelsName)
			continue
		}
		got = append(got, e.Name())
	}
	sort.Strings(got)

	if len(got) != len(want) {
		t.Errorf("%s holds %d houses, want %d: %v vs %v",
			assetpack.LevelsName, len(got), len(want), got, want)
	}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Errorf("%s does not hold %q (it holds %v): `make levels` builds it, then "+
				"`make levels-zip` packs it -- two commands, because the first exits 1 "+
				"on purpose once the archive and the directory disagree",
				assetpack.LevelsName, want[i], got)
			break
		}
	}
}

// shippedStamps is every shipped house's published `timestamp` field, and it is a table of
// promises rather than a table of facts: the number beside a name is what that house's players
// have in their saved games, so it is not the port's to change once the house is out.
//
// A house is added here in the same commit that ships it, which is the point -- see the test.
var shippedStamps = map[string]int32{
	"Open House": 1725439552,
}

// TestShippedHousesKeepTheStampTheirSavesAreKeyedOn is a guard on one field of one line, and the
// only test in the repository whose subject is a promise to strangers.
//
// A saved game is refused unless the house's TimeStamp still equals the one the save carries
// (internal/saved/store.go:401-407, the original's kYellowSavedTimeWrong). That gate is right --
// a save stores object states by room and object index, and an edited house may have moved both
// -- and it is what makes `timestamp 1725439552` in an authored house a published constant. Nothing
// else notices an edit to it: the house still lints, still finishes, still plays, and every saved
// game of it in the world stops loading. So the number is pinned, and the failure says what the
// change costs rather than that two numbers differ. See docs/IMPROVEMENTS.md 4.19.
//
// The pin is read off the authored *text*, which is the file somebody would edit;
// TestLevelArchiveIsFresh is what ties that text to the bytes a player actually has.
//
// It fails just as loudly for a house with no row, because the decision a new house needs is
// exactly this one -- pick a stamp, publish it, never move it -- and a table that quietly covers
// only the houses somebody remembered is not a guard.
func TestShippedHousesKeepTheStampTheirSavesAreKeyedOn(t *testing.T) {
	srcs, err := filepath.Glob(filepath.Join(levelSources, "*.house.txt"))
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool, len(srcs))
	for _, src := range srcs {
		name := strings.TrimSuffix(filepath.Base(src), ".house.txt")
		seen[name] = true
		h, err := house.ParseTextFile(src)
		if err != nil {
			t.Errorf("%s: %v", src, err)
			continue
		}
		want, ok := shippedStamps[name]
		if !ok {
			t.Errorf("%s has no row in shippedStamps: a house that ships publishes its "+
				"`timestamp` (%d here), because that field is the key every saved game of it "+
				"is checked against. Add the line.", src, h.TimeStamp)
			continue
		}
		if h.TimeStamp != want {
			t.Errorf("%s is authored with timestamp %d, but %d shipped: every saved game of "+
				"%q would be refused with \"has been modified since this game was saved\". "+
				"Put %d back, or decide deliberately to strand the saves and move the pin.",
				src, h.TimeStamp, want, name, want)
		}
		if h.TimeStamp&1 != 0 {
			t.Errorf("%s is authored with timestamp %d, which is odd, so the house is locked "+
				"against its own editor (house.Unlocked reads bit 0)", src, h.TimeStamp)
		}
	}
	for name := range shippedStamps {
		if !seen[name] {
			t.Errorf("shippedStamps has a row for %q and %s/%s.house.txt does not exist: "+
				"a house that has shipped cannot be unshipped from the table, so either the "+
				"house came back under another name or the row is the one that is wrong",
				name, levelSources, name)
		}
	}
}

// TestLevelArchiveIsFresh is TestArchiveIsTheTree for the houses, and it is the one test that
// cannot be replaced by looking at a directory.
//
// It builds each authored house here, in memory, with the same encoder `glidertool house build`
// uses, and compares the bytes with the archive member. That is deliberately a different route to
// the same answer than `make levels` takes -- that target runs bin/glidertool, writes
// assets/levels/, and then asks packassets whether the archive still matches the directory. Both
// checks are worth having and only this one runs in CI's `native` job, which never builds the
// tool or the tree.
//
// **A failure here means the archive is stale, not that the house is wrong.** Somebody edited
// levels/X.house.txt and committed it without `make levels-zip`, so every executable still
// carries the previous build of X. The remedy is in the message because the failure is silent
// everywhere else: the house lints, finishes and plays, it is just not the house in the text.
func TestLevelArchiveIsFresh(t *testing.T) {
	srcs, err := filepath.Glob(filepath.Join(levelSources, "*.house.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, src := range srcs {
		member := strings.TrimSuffix(filepath.Base(src), ".txt")
		h, err := house.ParseTextFile(src)
		if err != nil {
			t.Errorf("%s: %v", src, err)
			continue
		}
		want, err := h.Save()
		if err != nil {
			t.Errorf("%s: encode: %v", src, err)
			continue
		}
		got, err := fs.ReadFile(levelTree, member)
		if err != nil {
			t.Errorf("%s: %v -- `make levels` builds it, then `make levels-zip` packs it",
				member, err)
			continue
		}
		if len(got) != len(want) {
			t.Errorf("%s in %s is %d bytes, but %s builds to %d: the archive is stale, "+
				"`make levels-zip` repacks it",
				member, assetpack.LevelsName, len(got), src, len(want))
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s in %s differs from a build of %s at byte %d (%#02x, want %#02x): "+
					"the archive is stale, `make levels-zip` repacks it",
					member, assetpack.LevelsName, src, i, got[i], want[i])
				break
			}
		}
	}
}
