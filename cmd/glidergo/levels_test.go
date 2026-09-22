package main

// What `-levels DIR` does to the list a player sees.
//
// This is the one decision in the level-set design a reader is most likely to assume the other
// way round, and the assumption is reasonable: a *set* is a union -- the picker's Original and New
// rows come from two roots at once -- so a flag that names a directory of houses reads like a
// third source being added. It is not. All five asset flags replace the root they name, and
// -levels replaces the levels root, which is the one whose contents are otherwise invisible: they
// are inside the executable, so nobody can see what they were replaced with by looking at a
// directory.
//
// The deciding case is this repository's own documented workflow, and it is why these tests exist
// rather than a comment. `make levels` builds levels/*.house.txt into assets/levels/ and the way
// an author plays what they just wrote is `bin/glidergo -levels assets/levels` -- a directory
// holding a copy of every house already in the binary. Under additive semantics that lists every
// one of them twice, and the two rows are not merely ugly: a save file and a score board are keyed
// on the house's name and its timestamp (internal/saved/store.go:85, internal/scores.FileName), so
// the duplicates share both and the second row is a way to lose a saved game. That is not an edge
// case somebody has to go looking for; it is what happens the first time anybody follows the
// instructions. docs/IMPROVEMENTS.md 4.14's amendment is the argument.
//
// Nothing else in `make check` can see this. internal/shell takes its sources from its caller and
// so cannot tell what the caller decided, and the flag is parsed here -- so the test belongs here,
// on the real command line, with the real embedded archives behind it.

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/shell"
)

// oneBuiltInLevel copies one house out of the embedded levels archive into a fresh directory and
// returns the directory and the house's name.
//
// A copy rather than a house written for the test, because the copy is the case that matters: it
// reproduces what `make levels` leaves on disk, where the same house exists twice over -- once in
// the directory and once inside the binary running.
func oneBuiltInLevel(t *testing.T, tree fs.FS) (dir, name string) {
	t.Helper()
	ents, err := fs.ReadDir(tree, ".")
	if err != nil {
		t.Fatalf("the built-in levels archive will not list: %v", err)
	}
	if len(ents) == 0 {
		t.Skip("this build carries no houses of its own, so there is no replacement to test")
	}
	rel := ents[0].Name()
	b, err := fs.ReadFile(tree, rel)
	if err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, rel), b, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, strings.TrimSuffix(rel, ".house")
}

// discover builds the library the picker would show for one command line.
func discover(t *testing.T, args ...string) *shell.Library {
	t.Helper()
	o, err := parseArgs(t, args...)
	if err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	lib, err := shell.Discover(o.sources()...)
	if err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	lib.Sort()
	return lib
}

// TestLevelsFlagReplacesTheBuiltInSetRatherThanAddingToIt is the documented workflow, run.
func TestLevelsFlagReplacesTheBuiltInSetRatherThanAddingToIt(t *testing.T) {
	plain, err := parseArgs(t)
	if err != nil {
		t.Fatal(err)
	}
	dir, name := oneBuiltInLevel(t, plain.levelTree)

	before := discover(t)
	after := discover(t, "-levels", dir)

	// The one number a player sees on the status band, and the whole point: a directory
	// holding a copy of a house already in the binary must not make two houses of it.
	if len(after.Houses) != len(before.Houses) {
		t.Errorf("-levels %s lists %d houses where the same houses built in list %d: the "+
			"directory is being added to the levels root instead of replacing it",
			dir, len(after.Houses), len(before.Houses))
	}

	// Said again per name, because the count alone would also pass if the flag had replaced
	// the *houses* root by mistake and the totals happened to match.
	seen := 0
	for _, h := range after.Houses {
		if h.Name != name {
			continue
		}
		seen++
		if h.Set != shell.SetNew {
			t.Errorf("%q is in the %v set, want %v: a -levels directory is the New set",
				h.Name, h.Set, shell.SetNew)
		}
		if !strings.HasPrefix(h.Path, dir) {
			t.Errorf("%q was read from %q, want it under %s: the built-in copy is still "+
				"being listed", h.Name, h.Path, dir)
		}
	}
	if seen != 1 {
		t.Errorf("%q appears %d times with -levels %s, want once -- two rows of one house "+
			"share one save file and one score board", name, seen, dir)
	}

	// And the other four roots are untouched by a flag that names none of them.
	if a, b := after.Count(shell.SetOriginal), before.Count(shell.SetOriginal); a != b {
		t.Errorf("-levels changed the Original set from %d houses to %d", b, a)
	}
}

// TestAnEmptyLevelsDirectoryLeavesNoNewSetAtAll is the same rule from the other side, and the
// crisper half of the proof: under additive semantics the houses in the archive would still be
// listed, because an empty directory adds nothing. They are not listed, because it replaced them.
//
// It is also a real state and not a contrivance -- it is `make clean` and then `-levels
// assets/levels`, or a typo'd path that happens to exist -- and the thing to check is that it
// degrades to exactly the 1994 game rather than to an error or to a chooser with a dead position
// in it (internal/shell.Library.Sets).
func TestAnEmptyLevelsDirectoryLeavesNoNewSetAtAll(t *testing.T) {
	before := discover(t)
	if before.Count(shell.SetNew) == 0 {
		t.Skip("this build carries no houses of its own, so there is nothing to replace")
	}

	after := discover(t, "-levels", t.TempDir())
	if n := after.Count(shell.SetNew); n != 0 {
		t.Errorf("an empty -levels directory still lists %d house(s) in the New set: the "+
			"built-in archive is being added to rather than replaced", n)
	}
	if a, b := after.Count(shell.SetOriginal), before.Count(shell.SetOriginal); a != b {
		t.Errorf("the Original set went from %d houses to %d, and -levels names neither root",
			b, a)
	}
	if sets := after.Sets(); len(sets) != 1 || sets[0] != shell.SetOriginal {
		t.Errorf("the picker offers the sets %v, want just %v: an empty set is not a choice",
			sets, []shell.Set{shell.SetOriginal})
	}
}
