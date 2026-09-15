package house

// What PeekFile has to get right is not "does it parse a house" -- Load already
// answers that against all 22 shipped houses (corpus_test.go). It is the boundary:
// which files it accepts that Load would refuse, and which it refuses that a picker
// would otherwise list. Both directions are load-bearing, because internal/shell
// builds its list from this function and reports Load's failure separately
// (docs/IMPROVEMENTS.md 2.33), and a sniff that is too strict makes a real house
// vanish from the list with no message at all.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// peekFixture writes a syntactically valid house of n rooms and returns its path.
func peekFixture(t *testing.T, name string, n int, edit func(b []byte) []byte) string {
	t.Helper()
	h := &House{Version: HouseVersion, Rooms: make([]Room, n)}
	h.NRooms = int16(n)
	b, err := h.Save()
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if edit != nil {
		b = edit(b)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPeekReadsTheHeaderAndNotTheRooms(t *testing.T) {
	path := peekFixture(t, "Peek.house", 3, nil)

	s, err := PeekFile(path)
	if err != nil {
		t.Fatalf("PeekFile: %v", err)
	}
	if s.NRooms != 3 {
		t.Errorf("NRooms = %d, want 3", s.NRooms)
	}
	if s.Version != HouseVersion {
		t.Errorf("Version = 0x%04X, want 0x%04X", uint16(s.Version), HouseVersion)
	}
	if want := int64(SizeofHouseHeader + 3*SizeofRoom); s.Size != want {
		t.Errorf("Size = %d, want %d", s.Size, want)
	}
	if s.Path != path {
		t.Errorf("Path = %q, want %q", s.Path, path)
	}
}

// The header carries the board, and reading it is the whole reason the picker can
// show a house's high score before the house is opened.
func TestPeekCarriesTheBannerAndTheBoard(t *testing.T) {
	h := &House{Version: HouseVersion, Rooms: make([]Room, 1), NRooms: 1}
	h.Banner.SetText("A house by Ada")
	h.HighScores.Names[0].SetText("Ada")
	h.HighScores.Scores[0] = 4200
	b, err := h.Save()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "Banner.house")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := PeekFile(path)
	if err != nil {
		t.Fatalf("PeekFile: %v", err)
	}
	if s.Banner != "A house by Ada" {
		t.Errorf("Banner = %q", s.Banner)
	}
	if s.Scores.Scores[0] != 4200 || s.Scores.Names[0].Text() != "Ada" {
		t.Errorf("the board did not survive the peek: %q %d",
			s.Scores.Names[0].Text(), s.Scores.Scores[0])
	}
}

// Sampler is the reason the length test is `<=` rather than `==`: a PowerPC build
// padded houseType to 868 bytes and wrote two slack bytes past the last room, so the
// only shipped house that fails an equality test is a real house the original opens
// without complaint.
func TestPeekAcceptsThePowerPCSlack(t *testing.T) {
	path := peekFixture(t, "Slack.house", 2, func(b []byte) []byte {
		return append(b, make([]byte, PowerPCSlack)...)
	})
	if _, err := PeekFile(path); err != nil {
		t.Errorf("PeekFile refused a house with the PowerPC slack: %v", err)
	}

	// And any other trailing bytes are accepted too, deliberately: the sniff decides
	// what the picker lists, Load decides what plays, and the two are allowed to
	// disagree. A file that sniffs and will not load becomes a message on screen.
	junk := peekFixture(t, "Junk.house", 1, func(b []byte) []byte {
		return append(b, 1, 2, 3)
	})
	if _, err := PeekFile(junk); err != nil {
		t.Errorf("PeekFile refused a house with three trailing bytes: %v", err)
	}
	if _, err := LoadFile(junk); err == nil {
		t.Error("Load should still refuse it -- if both accept it, this test proves nothing")
	}
}

func TestPeekRefusalsSayWhy(t *testing.T) {
	dir := t.TempDir()

	// A Windows executable dropped in the houses directory: the version test's case,
	// and the error has to quote the two bytes it actually found.
	mz := filepath.Join(dir, "setup.house")
	if err := os.WriteFile(mz, append([]byte("MZ"), make([]byte, SizeofHouseHeader)...), 0o644); err != nil {
		t.Fatal(err)
	}

	// A truncated download: the header is fine and the rooms are not there.
	short := peekFixture(t, "Cut.house", 40, func(b []byte) []byte {
		return b[:SizeofHouseHeader+SizeofRoom]
	})

	// Too small to hold a header at all.
	tiny := filepath.Join(dir, "tiny.house")
	if err := os.WriteFile(tiny, make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		path string
		want []string
	}{
		{"wrong version", mz, []string{"version", "0x4D5A"}},
		{"truncated", short, []string{"40 rooms", "needs"}},
		{"too small", tiny, []string{"100 bytes", "too small"}},
		{"missing", filepath.Join(dir, "nope.house"), []string{"nope.house"}},
		{"a directory", dir, []string{"directory"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PeekFile(tc.path)
			if err == nil {
				t.Fatal("PeekFile accepted it")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error is %q; it should contain %q -- this string is what a\n"+
						"picker shows a player who can see the file and not the house", err, want)
				}
			}
		})
	}
}

// The 22 shipped houses all sniff, which is the claim the picker rests on. Skipped
// rather than failed with no assets, for corpus_test.go's reason.
func TestPeekAcceptsEveryShippedHouse(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "assets", "extracted", "houses")
	paths, err := filepath.Glob(filepath.Join(dir, "*.house"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Skipf("no extracted houses in %s -- run `make assets`", dir)
	}
	for _, p := range paths {
		s, err := PeekFile(p)
		if err != nil {
			t.Errorf("PeekFile(%s): %v", filepath.Base(p), err)
			continue
		}
		// The room count must agree with the loader's, or the picker shows one number
		// and the game plays another.
		h, err := LoadFile(p)
		if err != nil {
			t.Errorf("LoadFile(%s): %v", filepath.Base(p), err)
			continue
		}
		if int(s.NRooms) != len(h.Rooms) {
			t.Errorf("%s: peek says %d rooms, load found %d",
				filepath.Base(p), s.NRooms, len(h.Rooms))
		}
		if s.Unlocked != h.Unlocked() {
			t.Errorf("%s: peek says unlocked=%v, load says %v",
				filepath.Base(p), s.Unlocked, h.Unlocked())
		}
	}
}
