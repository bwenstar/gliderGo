package render

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

// The 'bnds' resource, which decides whether a room with a house's own background has
// walls, a floor and a ceiling.
//
// There was no test here, and that is the whole story of the bug it now guards: the
// loader read the record as an eight-byte QuickDraw Rect because the C calls the handle
// `boundsHand`, every shipped resource is four bytes, so every lookup failed the length
// check and answered "no resource" -- which GetOriginalBounding turns into "closed on
// all four sides". 155 rooms in the shipped corpus were sealed, and nothing said so,
// because the only thing that could have noticed was a walk over the room graph and
// nothing walked it.

// TestBndsIsFourBooleansAndNotARect drives the decoder from bytes.
//
// boundsType is `Boolean left, top, right, bottom` (GliderStructs.h:266-272) -- four
// bytes in that order, each a flag. The cases below are chosen to fail under the old
// Rect reading rather than merely to pass under this one: `01 00 01 00` is the commonest
// pattern in the corpus and is left+right open, which as big-endian int16s would read
// as top = 256 and left = 256 and mean something else entirely.
func TestBndsIsFourBooleansAndNotARect(t *testing.T) {
	cases := []struct {
		raw  []byte
		want Bounds
		why  string
	}{
		{[]byte{0, 0, 0, 0}, Bounds{}, "sealed: the two The Asylum Pro resources"},
		{[]byte{1, 0, 1, 0}, Bounds{Left: true, Right: true},
			"open sideways, floor and ceiling intact -- a corridor"},
		{[]byte{1, 1, 1, 1}, Bounds{true, true, true, true}, "open on all four sides"},
		{[]byte{0, 1, 0, 0}, Bounds{Top: true}, "Leviathan 3000: a hole in the ceiling only"},
		{[]byte{0, 0, 1, 1}, Bounds{Right: true, Bottom: true}, "Slumberland 3302"},
		// The bytes are flags, not counts: the C tests them with `if (x)`, so any
		// non-zero is open. No shipped resource does this, and the decoder must not
		// start caring if one ever does.
		{[]byte{2, 0, 255, 0}, Bounds{Left: true, Right: true}, "non-zero is open, not just 1"},
		// Longer than four bytes is not a reason to refuse: the original's
		// GetResource hands back a handle and the struct reads the first four bytes
		// of it whatever its size.
		{[]byte{1, 0, 0, 0, 9, 9, 9, 9}, Bounds{Left: true}, "trailing bytes are ignored"},
	}

	for _, c := range cases {
		a := NewAssets(nil)
		a.OpenHouseResFork("test", fstest.MapFS{
			"bnds/3000.bin": &fstest.MapFile{Data: c.raw},
		})
		got, ok := a.Bnds(3000)
		if !ok {
			t.Errorf("Bnds(%v): not found, want %s", c.raw, c.why)
			continue
		}
		if got != c.want {
			t.Errorf("Bnds(%v) = %+v, want %+v (%s)", c.raw, got, c.want, c.why)
		}
	}

	// Too short, absent, and no fork at all are the three "no resource" answers, and
	// all three have to be the *same* answer, because GetOriginalBounding's 0 is
	// documented as "the author never saved openings".
	short := NewAssets(nil)
	short.OpenHouseResFork("test", fstest.MapFS{
		"bnds/3000.bin": &fstest.MapFile{Data: []byte{1, 0, 1}},
	})
	if _, ok := short.Bnds(3000); ok {
		t.Error("a three-byte record was accepted; boundsType is four bytes")
	}
	if _, ok := short.Bnds(3001); ok {
		t.Error("a resource that is not there was found")
	}
	if _, ok := NewAssets(nil).Bnds(3000); ok {
		t.Error("a Bnds with no house fork open found something")
	}
}

// TestEveryShippedBndsResourceLoads reads all 70 'bnds' resources in the committed
// asset tree through the loader.
//
// It is a census and a regression guard at once. The census is the interesting half:
// every one of the 70 is exactly four bytes and every byte is 0 or 1, which is the
// evidence that the struct and not the Rect is the layout -- the corpus would not be
// uniformly four bytes wide if a Rect had ever been written. The guard is that all 70
// come back found, which is precisely what did not happen before.
func TestEveryShippedBndsResourceLoads(t *testing.T) {
	forkRoot := filepath.Join(assetRoot, "houseart")
	if _, err := os.Stat(forkRoot); err != nil {
		t.Skipf("no extracted house art at %s (run `make assets`)", forkRoot)
	}
	entries, err := os.ReadDir(forkRoot)
	if err != nil {
		t.Fatal(err)
	}

	total, houses := 0, 0
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(forkRoot, e.Name(), "bnds")
		files, err := os.ReadDir(dir)
		if err != nil {
			continue // a house with no 'bnds' of its own, which is fourteen of them
		}
		houses++
		names = append(names, e.Name())

		fork := filepath.Join(forkRoot, e.Name())
		a := NewAssets(nil)
		a.OpenHouseResFork(e.Name(), dirFS(fork))

		for _, f := range files {
			base := strings.TrimSuffix(f.Name(), ".bin")
			id, err := strconv.Atoi(base)
			if err != nil {
				t.Errorf("%s: %q is not a resource id", e.Name(), f.Name())
				continue
			}
			total++

			raw, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if len(raw) != 4 {
				t.Errorf("%s bnds %d: %d bytes, want 4 (boundsType is four Booleans)",
					e.Name(), id, len(raw))
			}
			for i, b := range raw {
				if b > 1 {
					t.Errorf("%s bnds %d byte %d = %d, want 0 or 1", e.Name(), id, i, b)
				}
			}

			got, ok := a.Bnds(int16(id))
			if !ok {
				t.Errorf("%s bnds %d: the loader did not find it", e.Name(), id)
				continue
			}
			want := Bounds{raw[0] != 0, raw[1] != 0, raw[2] != 0, raw[3] != 0}
			if got != want {
				t.Errorf("%s bnds %d = %+v, want %+v", e.Name(), id, got, want)
			}
		}
	}

	sort.Strings(names)
	t.Logf("%d 'bnds' resources across %d houses: %s", total, houses, strings.Join(names, ", "))
	// Pinned, because the interesting failure is the corpus shrinking to nothing and
	// the loop above then passing without reading a byte.
	if total != 70 || houses != 8 {
		t.Errorf("got %d resources across %d houses, want 70 across 8 "+
			"(docs/analysis/original-houses.md 3.5's bnds column)", total, houses)
	}
}
