package datadir

// The properties that only matter now that there is more than one kind of player data.
// internal/scores has its own tests for the boards -- including a spot-check table pinning
// the exact spelling of every `.scores` file a player might already have on disk -- and
// internal/saved has its own for the saves. What is left, and what lives here, is the part
// neither of them can see from inside: that the two kinds agree.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The two kinds land in sibling directories under one root, never in each other's, and never
// one inside the other. A save that got written into the scores directory would be read back
// by nothing.
func TestKindsAreSiblings(t *testing.T) {
	t.Setenv("GLIDERGO_DATA", "")
	t.Setenv("GLIDERGO_CONFIG", "/tmp/glider-portable")

	scores, err := Dir("scores")
	if err != nil {
		t.Fatalf("Dir(scores): %v", err)
	}
	saves, err := Dir("saves")
	if err != nil {
		t.Fatalf("Dir(saves): %v", err)
	}
	if scores == saves {
		t.Fatalf("both kinds resolve to %q", scores)
	}
	if filepath.Dir(scores) != filepath.Dir(saves) {
		t.Errorf("scores go to %q and saves to %q, which are not siblings", scores, saves)
	}
	for _, c := range []struct{ a, b string }{{scores, saves}, {saves, scores}} {
		if strings.HasPrefix(c.a, c.b+string(filepath.Separator)) {
			t.Errorf("%q is inside %q", c.a, c.b)
		}
	}
}

// GLIDERGO_DATA is used verbatim, with no sub component, which is the one place the two
// kinds share a directory. That is deliberate -- it is the portable-install escape hatch,
// and a stick with Titanic.scores next to Titanic.save is easier to reason about -- so it
// only works because the extensions differ. Check that it does.
func TestGLIDERGO_DATAIsVerbatimAndTheKindsStillDoNotCollide(t *testing.T) {
	t.Setenv("GLIDERGO_DATA", "/tmp/glider-data")
	t.Setenv("GLIDERGO_CONFIG", "/tmp/glider-config")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg")

	for _, sub := range []string{"scores", "saves"} {
		got, err := Dir(sub)
		if err != nil || got != "/tmp/glider-data" {
			t.Errorf("Dir(%q) = %q, %v; want the variable verbatim", sub, got, err)
		}
	}
	if a, b := FileName("Titanic", ".scores"), FileName("Titanic", ".save"); a == b {
		t.Errorf("one house's board and save are both %q", a)
	}
}

// The XDG default, and the spec's rule that a relative $XDG_DATA_HOME counts as unset --
// which matters because a relative path would put a player's saves wherever they happened to
// launch the game from.
func TestDirDefaultsUnderTheDataDirectory(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the hand-rolled XDG path is the Linux branch")
	}
	t.Setenv("GLIDERGO_DATA", "")
	t.Setenv("GLIDERGO_CONFIG", "")
	t.Setenv("HOME", "/home/tester")

	t.Setenv("XDG_DATA_HOME", "/tmp/xdg")
	want := filepath.Join("/tmp/xdg", AppDir, "saves")
	if got, err := Dir("saves"); err != nil || got != want {
		t.Errorf("Dir(saves) = %q, %v; want %q", got, err, want)
	}

	want = filepath.Join("/home/tester", ".local", "share", AppDir, "saves")
	for _, xdg := range []string{"", "relative/path"} {
		t.Setenv("XDG_DATA_HOME", xdg)
		if got, err := Dir("saves"); err != nil || got != want {
			t.Errorf("with XDG_DATA_HOME=%q, Dir(saves) = %q, %v; want %q", xdg, got, err, want)
		}
	}
}

// One escaper, so one answer: whatever a house is called, its two files differ in the
// extension and in nothing else. The alternative -- two copies of the escaper that drifted
// apart -- would show up as a house whose board is found and whose save is not.
func TestOneHouseGetsOneStemForEveryKind(t *testing.T) {
	names := []string{
		"Titanic", "Art Museum", "Ozma's Revenge", "", "CON", "con", "a/b", "Caf\xE9",
		strings.Repeat("long", 40),
	}
	for _, n := range names {
		a, b := FileName(n, ".scores"), FileName(n, ".save")
		if strings.TrimSuffix(a, ".scores") != strings.TrimSuffix(b, ".save") {
			t.Errorf("FileName(%q) is %q for a board and %q for a save; the stems differ",
				n, a, b)
		}
	}
}

// The escaper is total and injective for every kind, and each name it emits is one all three
// target platforms will open. scores' tests assert this for `.scores`; the extension is part
// of what a Windows device name is checked against ("CON.save" is as unopenable as "CON"),
// so it is worth asserting for the new one too.
func TestFileNameIsInjectiveAndPortableForEveryExt(t *testing.T) {
	names := []string{
		"Titanic", "titanic", "TITANIC", "Art Museum", "Art%20Museum", "Ozma's Revenge",
		"", "%", ".", "..", "CON", "con", "NUL", "a/b", "a\\b", "a:b", "Caf\xE9",
		strings.Repeat("x", 99), strings.Repeat("x", 100), strings.Repeat("x", 101),
		strings.Repeat("x", 100) + "a", strings.Repeat("x", 100) + "b",
	}
	for _, ext := range []string{".scores", ".save"} {
		seen := map[string]string{}
		for _, n := range names {
			f := FileName(n, ext)
			if prev, dup := seen[f]; dup {
				t.Errorf("%q and %q both map to %q", prev, n, f)
			}
			seen[f] = n

			if !strings.HasSuffix(f, ext) || f == ext {
				t.Errorf("FileName(%q, %q) = %q", n, ext, f)
			}
			if f == "." || f == ".." {
				t.Errorf("FileName(%q, %q) = %q", n, ext, f)
			}
			for i := 0; i < len(f); i++ {
				if c := f[i]; !SafeByte(c) && c != '%' {
					t.Errorf("FileName(%q, %q) = %q, which holds byte %02X at %d",
						n, ext, f, c, i)
				}
			}
			if Reserved(strings.TrimSuffix(f, ext)) {
				t.Errorf("FileName(%q, %q) = %q, a Windows device name", n, ext, f)
			}
			if len(f) > 120 {
				t.Errorf("FileName(%q, %q) is %d bytes long", n, ext, len(f))
			}
		}
	}
}

// A name the escaper emits can actually be created, on this filesystem at least. The
// portability check above is by inspection; this one is by trying it.
func TestEveryNameCanBeCreated(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"Titanic", "Art Museum", "", "CON", "a/b", "Caf\xE9",
		strings.Repeat("long", 40)} {
		p := filepath.Join(dir, FileName(n, ".save"))
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Errorf("house %q -> %q: %v", n, filepath.Base(p), err)
		}
	}
}
