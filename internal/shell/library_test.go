package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
)

// houseBytes builds the smallest file that is a real house: a 866-byte header with
// kHouseVersion in it and n whole rooms after it. Save is the authority for the
// layout, so these tests cannot drift from the codec.
func houseBytes(t *testing.T, rooms int) []byte {
	t.Helper()
	h := &house.House{Version: house.HouseVersion, Rooms: make([]house.Room, rooms)}
	b, err := h.Save()
	if err != nil {
		t.Fatalf("building a %d-room house: %v", rooms, err)
	}
	if want := house.SizeofHouseHeader + rooms*house.SizeofRoom; len(b) != want {
		t.Fatalf("built %d bytes, want %d", len(b), want)
	}
	return b
}

func write(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// testLibrary lays out a directory with one of everything Discover has to cope
// with, and returns it.
func testLibrary(t *testing.T) (root string, lib *Library) {
	t.Helper()
	root = t.TempDir()

	write(t, filepath.Join(root, "Slumberland.house"), houseBytes(t, 3))
	write(t, filepath.Join(root, "Demo House"), houseBytes(t, 1)) // no extension, as on a Mac
	write(t, filepath.Join(root, "aardvark.house"), houseBytes(t, 2))
	write(t, filepath.Join(root, "sub", "Nested.glh"), houseBytes(t, 1))

	write(t, filepath.Join(root, "notes.txt"), []byte("read me"))           // ignored: extension
	write(t, filepath.Join(root, ".hidden.house"), houseBytes(t, 1))        // ignored: hidden
	write(t, filepath.Join(root, ".git", "Sneaky.house"), houseBytes(t, 1)) // ignored: hidden dir

	write(t, filepath.Join(root, "broken.house"), func() []byte {
		b := houseBytes(t, 1)
		b[0], b[1] = 'M', 'Z' // a Windows executable someone renamed
		return b
	}())
	write(t, filepath.Join(root, "tiny.house"), make([]byte, 100))

	lib, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	return root, lib
}

func TestDiscoverFindsHousesInSortOrder(t *testing.T) {
	_, lib := testLibrary(t)

	want := []string{"aardvark", "Demo House", "Nested", "Slumberland"}
	got := make([]string, len(lib.Houses))
	for i, h := range lib.Houses {
		got[i] = h.Name
	}
	if len(got) != len(want) {
		t.Fatalf("found %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("house %d is %q, want %q (whole list %v)", i, got[i], want[i], got)
		}
	}

	if r := lib.Houses[0].Rooms; r != 2 {
		t.Errorf("aardvark has %d rooms, want 2", r)
	}
	if !lib.Houses[1].Demo {
		t.Errorf("%q should be flagged as the demo house", lib.Houses[1].Name)
	}
	if lib.Houses[0].Demo {
		t.Errorf("aardvark should not be flagged as the demo house")
	}
	if lib.Houses[2].Rel != filepath.Join("sub", "Nested.glh") {
		t.Errorf("Nested's Rel is %q, want sub/Nested.glh", lib.Houses[2].Rel)
	}
}

// A file that looks like a house and is not has to be reported, not silently
// dropped: that is the whole point of Library.Skipped (docs/IMPROVEMENTS.md 2.33).
func TestDiscoverReportsRejects(t *testing.T) {
	_, lib := testLibrary(t)

	if len(lib.Skipped) != 2 {
		for _, s := range lib.Skipped {
			t.Logf("skipped %s: %v", s.Path, s.Why)
		}
		t.Fatalf("skipped %d files, want 2 (broken.house and tiny.house)", len(lib.Skipped))
	}
	for _, s := range lib.Skipped {
		if s.Why == nil {
			t.Errorf("%s was skipped with no reason", s.Path)
		}
		if base := filepath.Base(s.Path); base != "broken.house" && base != "tiny.house" {
			t.Errorf("unexpected reject %s: %v", s.Path, s.Why)
		}
	}
	// The reason has to name the file and say what was wrong with it, because it is
	// shown to a player who can see the file in the folder.
	if got := lib.Skipped[0].Why.Error(); !strings.Contains(got, "broken.house") || !strings.Contains(got, "version") {
		t.Errorf("broken.house's reason is %q; want the name and the version in it", got)
	}
}

func TestDiscoverIgnoresHiddenAndForeignFiles(t *testing.T) {
	_, lib := testLibrary(t)
	for _, h := range lib.Houses {
		switch h.Name {
		case ".hidden", "Sneaky", "notes":
			t.Errorf("%q should not have been listed (Rel %q)", h.Name, h.Rel)
		}
	}
	for _, s := range lib.Skipped {
		if base := filepath.Base(s.Path); base == "notes.txt" {
			t.Errorf("notes.txt should be ignored silently, not reported as a reject")
		}
	}
}

func TestDiscoverBadRoot(t *testing.T) {
	if _, err := Discover(""); err == nil {
		t.Error("an empty root should be an error")
	}
	if _, err := Discover(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("a missing root should be an error")
	}
	f := filepath.Join(t.TempDir(), "afile")
	write(t, f, []byte("x"))
	if _, err := Discover(f); err == nil {
		t.Error("a root that is a file should be an error")
	}
}

func TestFindIsCaseInsensitive(t *testing.T) {
	_, lib := testLibrary(t)
	if i := lib.Find("SLUMBERLAND"); i < 0 || lib.Houses[i].Name != "Slumberland" {
		t.Errorf("Find(SLUMBERLAND) = %d, want the index of Slumberland", i)
	}
	if i := lib.Find("demo house"); i < 0 || lib.Houses[i].Name != "Demo House" {
		t.Errorf("Find(demo house) = %d, want the index of Demo House", i)
	}
	if i := lib.Find("Nowhere"); i != -1 {
		t.Errorf("Find(Nowhere) = %d, want -1", i)
	}
}

// NameFirst is WhichStringFirst, and these are its three documented rules
// (StringUtils.c:34-87).
func TestNameFirstRules(t *testing.T) {
	cases := []struct {
		a, b  string
		first bool
		why   string
	}{
		{"aardvark", "Slumberland", true, "lowercase folds to uppercase, so a sorts before S"},
		{"AARDVARK", "aardvark", false, "the fold makes equal strings equal, so neither is first"},
		{"aardvark", "AARDVARK", false, "and the comparison is symmetric"},
		{"Demo", "Demo House", true, "a common prefix puts the shorter one first"},
		{"Demo House", "Demo", false, "and not the longer one"},
		{"Zoo", "École", true, "bytes at or above 0x80 are not folded, so they sort last"},
		{"Ant", "Bee", true, "ordinary case"},
		{"Bee", "Ant", false, "ordinary case, reversed"},
		{"", "Ant", true, "the empty string is shortest"},
		{"", "", false, "equal"},
	}
	for _, c := range cases {
		if got := NameFirst(c.a, c.b); got != c.first {
			t.Errorf("NameFirst(%q, %q) = %v, want %v -- %s", c.a, c.b, got, c.first, c.why)
		}
	}
}

// Two houses with the same name in two directories are two houses. The original
// collapses them (SelectHouse.c:527-528); this must not.
func TestSameNameInTwoDirectories(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a", "Demo House"), houseBytes(t, 1))
	write(t, filepath.Join(root, "b", "Demo House"), houseBytes(t, 2))

	lib, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(lib.Houses) != 2 {
		t.Fatalf("found %d houses, want 2", len(lib.Houses))
	}
	if lib.Houses[0].Rel > lib.Houses[1].Rel {
		t.Errorf("same-named houses are not ordered by path: %q then %q",
			lib.Houses[0].Rel, lib.Houses[1].Rel)
	}
}
