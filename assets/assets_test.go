package assets

// What the executables are carrying, checked against what is committed.
//
// This is the only test in the repository that links the archive, and it is here because it is
// the only one whose subject is the archive. Everything under internal/ reads the working copy of
// extracted/ instead, so `go test ./internal/...` links none of these 11 MiB.

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/bwenstar/gliderGo/internal/assetpack"
)

// tree is the extracted tree beside this file, as a path for the on-disk half of the comparison.
const treeDir = "extracted"

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
