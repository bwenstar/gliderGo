package assetfs

// ArtRoots tests: the search list for one house's own pictures.
//
// This is the one thing in the package that searches instead of substituting, so the order it
// searches in and the words it says about a miss are the whole of its behaviour
// (docs/IMPROVEMENTS.md 4.15 and 4.17). Both are easy to get subtly wrong in a way no other test
// would notice: a list that searched the built-in forks first would silently give a new house the
// 1994 art for whatever PICT id it happened to reuse, and a Complaint that fired on "there is no
// fork here" rather than on "there is no fork here and this house wanted one" is the bug 4.17 is
// about.

import (
	"strings"
	"testing"
	"testing/fstest"
)

// artFor is a root holding one house's fork, with a single picture in it so that IsDir has a
// directory to find. The id is not read by anything here; a fork is located by the house's name.
func artFor(house string) fstest.MapFS {
	return fstest.MapFS{house + "/pict/3000.png": {Data: []byte("not really a png")}}
}

func TestForkTakesTheFirstRootThatHasTheHouse(t *testing.T) {
	roots := ArtRoots{
		{FS: artFor("Titanic"), Label: "first"},
		{FS: artFor("Gallery"), Label: "second"},
		{FS: artFor("Gallery"), Label: "third"},
	}
	fork := roots.Fork("Gallery")
	if !fork.Found() {
		t.Fatal("Gallery is in two of the three roots")
	}
	if fork.Label != "second/Gallery" {
		t.Errorf("Label = %q, want second/Gallery -- first hit wins, and the label says which",
			fork.Label)
	}
}

// A nil FS is skipped without being searched or named. Nil roots are ordinary: the built-in tree is
// nil in every test binary and in every library caller, which is deliberate (see the package
// comment), so a list of three roots is routinely a list of one.
func TestForkSkipsNilRoots(t *testing.T) {
	roots := ArtRoots{
		{Label: "no tree here"},
		{FS: artFor("Titanic"), Label: "built-in:houseart"},
	}
	fork := roots.Fork("Titanic")
	if fork.Label != "built-in:houseart/Titanic" {
		t.Errorf("Label = %q, want built-in:houseart/Titanic", fork.Label)
	}
	if strings.Contains(fork.Label, "no tree here") {
		t.Error("a root with no filesystem was named as though it had been searched")
	}
}

// Passed is the whole of what Named buys: a directory somebody typed that did not have the house
// they are playing is worth a line, and a built-in root that did not is not. Both kinds are skipped
// over identically; only the named one is remembered.
func TestForkRemembersOnlyTheNamedRootsItPassed(t *testing.T) {
	roots := ArtRoots{
		{FS: artFor("Somebody Else"), Label: "/tmp/art", Named: true},
		{FS: artFor("Nobody"), Label: "built-in:levels/houseart"},
		{FS: artFor("Titanic"), Label: "built-in:houseart"},
	}
	fork := roots.Fork("Titanic")
	if !fork.Found() {
		t.Fatal("Titanic is in the third root")
	}
	if len(fork.Passed) != 1 || fork.Passed[0] != "/tmp/art/Titanic" {
		t.Fatalf("Passed = %v, want exactly [/tmp/art/Titanic] -- the built-in root that also "+
			"missed is not a mistake anybody made", fork.Passed)
	}
}

// Nothing found names every root that was searched, because the message it feeds has to say where
// to put a fork and not only that there is not one.
func TestForkNamesEveryRootSearchedWhenNothingHasTheHouse(t *testing.T) {
	roots := ArtRoots{
		{FS: artFor("Somebody Else"), Label: "/tmp/art", Named: true},
		{FS: artFor("Nobody"), Label: "built-in:houseart"},
	}
	fork := roots.Fork("Titanic")
	if fork.Found() {
		t.Fatal("no root has Titanic")
	}
	if fork.Label != "/tmp/art, built-in:houseart" {
		t.Errorf("Label = %q, want both roots named", fork.Label)
	}
}

// An empty list is a state a caller can reach -- no flag, no built-in tree, no levels archive --
// and it has to be a quiet "no fork" rather than a panic or a message about nowhere.
func TestForkOnNoRootsAtAll(t *testing.T) {
	fork := ArtRoots{}.Fork("Titanic")
	if fork.Found() || fork.Label != "" {
		t.Fatalf("Found = %v, Label = %q; want false and empty", fork.Found(), fork.Label)
	}
	if why := fork.Complaint("Titanic", false); why != "" {
		t.Errorf("Complaint = %q, want nothing: no roots and no art wanted", why)
	}
}

func TestComplaint(t *testing.T) {
	found := ArtRoots{{FS: artFor("Titanic"), Label: "built-in:houseart"}}.Fork("Titanic")
	missed := ArtRoots{{FS: artFor("Nobody"), Label: "built-in:houseart"}}.Fork("Titanic")
	passed := ArtRoots{
		{FS: artFor("Nobody"), Label: "/tmp/art", Named: true},
		{FS: artFor("Titanic"), Label: "built-in:houseart"},
	}.Fork("Titanic")

	for _, c := range []struct {
		name  string
		fork  Fork
		wants bool
		want  string // a substring, or "" for silence
		why   string
	}{
		{
			name: "found, and wanted", fork: found, wants: true, want: "",
			why: "the ordinary case for half the shipped houses, and it says nothing",
		},
		{
			name: "found, not wanted", fork: found, wants: false, want: "",
			why: "a fork that is there and unused is not news; nine of the 22 are shaped so",
		},
		{
			name: "missing, not wanted", fork: missed, wants: false, want: "",
			why: "this is 4.17: `Open House` is drawn from built-in backgrounds alone and used " +
				"to be the only house warned at on a complete tree",
		},
		{
			name: "missing, and wanted", fork: missed, wants: true,
			want: "Titanic has no resource fork in built-in:houseart",
			why:  "the message the port exists to print, naming the house and where it looked",
		},
		{
			name: "found elsewhere after a named root missed", fork: passed, wants: false,
			want: "no resource fork at /tmp/art/Titanic; using built-in:houseart/Titanic",
			why: "said even though the house wants no art, because the claim being checked is " +
				"about the directory: this is how a half-extracted tree is caught",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := c.fork.Complaint("Titanic", c.wants)
			switch {
			case c.want == "" && got != "":
				t.Errorf("said %q; %s", got, c.why)
			case c.want != "" && !strings.Contains(got, c.want):
				t.Errorf("said %q, want it to contain %q; %s", got, c.want, c.why)
			}
		})
	}
}

// A fork holding only 'bnds' and no pict/ is a real thing -- a house can describe which sides of a
// *built-in* background are openings -- so the test that locates one is IsDir on the house's
// directory and not "does pict/ exist".
func TestForkFindsAForkWithNoPicturesInIt(t *testing.T) {
	only := fstest.MapFS{"Titanic/bnds/2011.bin": {Data: []byte{0, 0, 0, 0}}}
	if !(ArtRoots{{FS: only, Label: "built-in:houseart"}}).Fork("Titanic").Found() {
		t.Fatal("a fork of bounds and no pictures is still the house's fork")
	}
}

func TestNamedArtIsNamed(t *testing.T) {
	r := NamedArt("/tmp/art")
	if !r.Named || r.Label != "/tmp/art" || r.FS == nil {
		t.Fatalf("NamedArt(%q) = %+v; the flag's value is the root, and it is a named one",
			"/tmp/art", r)
	}
	if Built(r.Label) {
		t.Error("a directory somebody typed is not the built-in copy")
	}
}
