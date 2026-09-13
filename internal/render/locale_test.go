package render

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"glidergo/internal/house"
)

// update rewrites testdata/locale_golden.txt from the current renderer. It is
// spelled the way the standard library spells it, and it is the only way that
// file is meant to change: `go test ./internal/render -update`, then read the
// diff and justify every line of it in the commit message.
var update = flag.Bool("update", false, "rewrite the golden files from the current output")

const (
	assetRoot  = "../../assets/extracted"
	goldenFile = "testdata/locale_golden.txt"
)

// requireAssets skips rather than fails when the extracted art is absent. The
// art is derived from a copyrighted 1994 application and is not in the
// repository; `make assets` produces it. Everything in this file that needs a
// pixel therefore has to be skippable, and everything that does not need one is
// deliberately kept out of that set.
func requireAssets(t *testing.T, sub string) string {
	t.Helper()
	dir := filepath.Join(assetRoot, sub)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no extracted assets at %s (run `make assets`)", dir)
	}
	return dir
}

// ---------------------------------------------------------------------------
// The palette
// ---------------------------------------------------------------------------

// TestPaletteMatchesClut checks the generated palette against the shipped
// resource, byte for byte, for both copies.
//
// palette.go builds the table from a description of how Apple's 256-colour
// system palette is laid out rather than transcribing 256 triples, which is only
// legitimate if the description is right. It is checked against the real 'clut'
// because the indices are arithmetic operands: DrawTable ORs them together to
// dither a shadow, so an off-by-one anywhere in the table changes pixels that
// have nothing to do with the colour that moved.
//
// 'clut' 128 and 129 are separate resources with identical contents. Both are
// checked, because "they are the same" is the claim the renderer relies on when
// it uses one table for every surface.
func TestPaletteMatchesClut(t *testing.T) {
	dir := requireAssets(t, "res/clut")
	for _, id := range []int{128, 129} {
		path := filepath.Join(dir, fmt.Sprintf("%d.bin", id))
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("clut %d: %v", id, err)
		}
		// ColorTable: ctSeed(4) ctFlags(2) ctSize(2), then ctSize+1 ColorSpecs
		// of value(2) rgb(6). ctSize is the top index, so 255 means 256 entries.
		if len(b) < 8 {
			t.Fatalf("clut %d: %d bytes, too short for a ColorTable header", id, len(b))
		}
		size := int(binary.BigEndian.Uint16(b[6:8]))
		if size != 255 {
			t.Errorf("clut %d: ctSize %d, want 255", id, size)
		}
		if want := 8 + (size+1)*8; len(b) != want {
			t.Fatalf("clut %d: %d bytes, want %d", id, len(b), want)
		}
		for i := 0; i <= size; i++ {
			spec := b[8+i*8:]
			value := int(binary.BigEndian.Uint16(spec[0:2]))
			r := binary.BigEndian.Uint16(spec[2:4])
			g := binary.BigEndian.Uint16(spec[4:6])
			bl := binary.BigEndian.Uint16(spec[6:8])
			if value != i {
				t.Errorf("clut %d entry %d: value %d, not its own index", id, i, value)
			}
			// The Mac stores 16 bits per channel; an 8-bit source colour is
			// widened by repetition, so the low byte must equal the high byte
			// and either one is the 8-bit value.
			for _, c := range [3]uint16{r, g, bl} {
				if c>>8 != c&0xFF {
					t.Fatalf("clut %d entry %d: channel %04X is not an 8-bit value doubled", id, i, c)
				}
			}
			got := Palette[i]
			if got.R != uint8(r>>8) || got.G != uint8(g>>8) || got.B != uint8(bl>>8) {
				t.Errorf("clut %d entry %d: have #%02X%02X%02X, want #%02X%02X%02X",
					id, i, got.R, got.G, got.B, r>>8, g>>8, bl>>8)
			}
			if got.A != 0xFF {
				t.Errorf("clut %d entry %d: alpha %d, want 255", id, i, got.A)
			}
		}
	}

	// The two properties the rest of the renderer leans on.
	if c := Palette[White8]; c.R != 0xFF || c.G != 0xFF || c.B != 0xFF {
		t.Errorf("index 0 is #%02X%02X%02X, want pure white: it is the colour key", c.R, c.G, c.B)
	}
	seen := make(map[[3]uint8]int, 256)
	for i, c := range Palette {
		k := [3]uint8{c.R, c.G, c.B}
		if prev, dup := seen[k]; dup {
			t.Errorf("indices %d and %d are both #%02X%02X%02X: RGB -> index is not injective",
				prev, i, c.R, c.G, c.B)
		}
		seen[k] = i
	}
}

// ---------------------------------------------------------------------------
// The room queries, with no art at all
// ---------------------------------------------------------------------------

// testHouse builds a small house by hand: a 3x3 block of rooms on floor 1 plus
// one room each on floors 0 and 2, so that every neighbour slot has something
// findable and a few have nothing.
//
//	floor 2                 suite 10 (room 9, garden -- lit, not a structure)
//	floor 1  suite 9 (r0)   suite 10 (r1)      suite 11 (r2)
//	floor 0  suite 9 (r3)   suite 10 (r4)      suite 11 (r5)
//
// Rooms 6..8 are decoys on floors nothing asks about, to prove the search is by
// (floor, suite) and not by index.
func testHouse() *house.House {
	h := &house.House{Version: 0x0200, FirstRoom: 1}
	add := func(name string, floor, suite, background int16, bounds int16) {
		var r house.Room
		r.Name.SetText(name)
		r.Floor, r.Suite = floor, suite
		r.Background = background
		r.Bounds = bounds
		h.Rooms = append(h.Rooms, r)
	}
	add("west", 1, 9, kSimpleRoom, 0)     // 0
	add("centre", 1, 10, kPaneledRoom, 0) // 1
	add("east", 1, 11, kSimpleRoom, 0)    // 2
	add("sw", 0, 9, kDirt, 0)             // 3
	add("south", 0, 10, kBasement, 0)     // 4
	add("se", 0, 11, kDirt, 0)            // 5
	add("decoy a", 4, 40, kSimpleRoom, 0) // 6
	add("decoy b", 5, 41, kSimpleRoom, 0) // 7
	add("decoy c", 6, 42, kSimpleRoom, 0) // 8
	add("above", 2, 10, kGarden, 0)       // 9
	h.NRooms = int16(len(h.Rooms))
	return h
}

// TestRoomQueries exercises the five lookups DrawLocale is built on. It needs no
// art, so it runs on a bare checkout -- which matters, because these are the
// functions that decide *which* room is drawn where, and a regression in them
// would show up in the golden hashes as an unexplainable pixel change.
func TestRoomQueries(t *testing.T) {
	h := testHouse()
	s := NewScene(DefaultView(), NewAssets(""), h)
	s.RoomNumber = 1 // "centre", floor 1 suite 10

	t.Run("GetRoomNumber", func(t *testing.T) {
		cases := []struct{ floor, suite, want int16 }{
			{1, 9, 0},
			{1, 10, 1},
			{1, 11, 2},
			{0, 10, 4},
			{2, 10, 9},
			{1, 12, kRoomIsEmpty}, // no such suite on that floor
			{3, 10, kRoomIsEmpty}, // no such floor
			{4, 10, kRoomIsEmpty}, // floor 4 exists, but at suite 40
		}
		for _, c := range cases {
			if got := s.GetRoomNumber(c.floor, c.suite); got != c.want {
				t.Errorf("GetRoomNumber(%d,%d) = %d, want %d", c.floor, c.suite, got, c.want)
			}
		}
	})

	t.Run("GetNeighborRoomNumber", func(t *testing.T) {
		// The slot order is the original's: central, N, NE, E, SE, S, SW, W, NW.
		want := [9]int16{1, 9, kRoomIsEmpty, 2, 5, 4, 3, 0, kRoomIsEmpty}
		for slot, w := range want {
			if got := s.GetNeighborRoomNumber(slot); got != w {
				t.Errorf("GetNeighborRoomNumber(%d) = %d, want %d", slot, got, w)
			}
		}
	})

	t.Run("IsRoomAStructure", func(t *testing.T) {
		cases := []struct {
			room int16
			want bool
		}{
			{0, true},  // kSimpleRoom
			{1, true},  // kPaneledRoom
			{4, false}, // kBasement is interior but not a "structure"
			{3, false}, // kDirt
			{9, false}, // kGarden
			{kRoomIsEmpty, false},
			{99, false}, // out of range
		}
		for _, c := range cases {
			if got := s.IsRoomAStructure(c.room); got != c.want {
				t.Errorf("IsRoomAStructure(%d) = %v, want %v", c.room, got, c.want)
			}
		}

		// A house-supplied background decides by the bounds bits when it has
		// any, and by the id otherwise. Both branches, on one room.
		h.Rooms[0].Background = kUserBackground
		h.Rooms[0].Bounds = 0
		if !s.IsRoomAStructure(0) {
			t.Errorf("background %d, bounds 0: want a structure (below kUserStructureRange)", kUserBackground)
		}
		h.Rooms[0].Background = kUserStructureRange
		if s.IsRoomAStructure(0) {
			t.Errorf("background %d, bounds 0: want not a structure", kUserStructureRange)
		}
		h.Rooms[0].Bounds = 32
		if !s.IsRoomAStructure(0) {
			t.Error("bounds bit 32 set: want a structure whatever the background")
		}
		h.Rooms[0].Bounds = 16
		if s.IsRoomAStructure(0) {
			t.Error("bounds set but bit 32 clear: want not a structure")
		}
		h.Rooms[0].Background, h.Rooms[0].Bounds = kSimpleRoom, 0
	})

	t.Run("GetNumberOfLights", func(t *testing.T) {
		if got := s.GetNumberOfLights(9); got != 1 {
			t.Errorf("garden: %d lights, want 1 -- outdoors is lit by definition", got)
		}
		if got := s.GetNumberOfLights(3); got != 1 {
			t.Errorf("dirt with all-zero tiles: %d lights, want 1", got)
		}
		h.Rooms[3].Tiles[5] = 2
		if got := s.GetNumberOfLights(3); got != 0 {
			t.Errorf("dirt with a solid tile: %d lights, want 0 -- that room is underground", got)
		}
		h.Rooms[3].Tiles[5] = 0

		if got := s.GetNumberOfLights(1); got != 0 {
			t.Errorf("bare panelled room: %d lights, want 0", got)
		}
		// An interior window counts unconditionally; a lamp only when on.
		h.Rooms[1].Objects[0].What = kWindowInLf
		h.Rooms[1].Objects[1].What = kCeilingLight
		if got := s.GetNumberOfLights(1); got != 1 {
			t.Errorf("window plus an off ceiling light: %d lights, want 1", got)
		}
		lit := h.Rooms[1].Objects[1].Light()
		lit.State = 1
		h.Rooms[1].Objects[1].SetLight(lit)
		if got := s.GetNumberOfLights(1); got != 2 {
			t.Errorf("window plus an on ceiling light: %d lights, want 2", got)
		}
		h.Rooms[1].Objects[0].What = house.ObjectIsEmpty
		h.Rooms[1].Objects[1].What = house.ObjectIsEmpty

		if got := s.GetNumberOfLights(kRoomIsEmpty); got != 0 {
			t.Errorf("GetNumberOfLights(kRoomIsEmpty) = %d, want 0", got)
		}
	})

	t.Run("ExtractFloorSuite", func(t *testing.T) {
		// Version 2.0 and later: suite is the hundreds, floor the remainder,
		// biased by the eight underground floors.
		if f, su := s.ExtractFloorSuite(1008); f != 0 || su != 10 {
			t.Errorf("2.0 combo 1008 -> floor %d suite %d, want 0 and 10", f, su)
		}
		if f, su := s.ExtractFloorSuite(1009); f != 1 || su != 10 {
			t.Errorf("2.0 combo 1009 -> floor %d suite %d, want 1 and 10", f, su)
		}
		// Before 2.0 the two halves were the other way round.
		h.Version = 0x0100
		if f, su := s.ExtractFloorSuite(1008); f != 2 || su != 8 {
			t.Errorf("1.0 combo 1008 -> floor %d suite %d, want 2 and 8", f, su)
		}
		h.Version = 0x0200
	})

	t.Run("IsThisValid", func(t *testing.T) {
		if s.IsThisValid(1, 0) {
			t.Error("an empty slot is not valid")
		}
		h.Rooms[1].Objects[0].What = kTable
		if !s.IsThisValid(1, 0) {
			t.Error("a table is always valid")
		}
		// A collectable is valid only while it is still there.
		h.Rooms[1].Objects[0].What = kRedClock
		b := h.Rooms[1].Objects[0].Bonus()
		b.State = 0
		h.Rooms[1].Objects[0].SetBonus(b)
		if s.IsThisValid(1, 0) {
			t.Error("a taken clock must not be valid: it would reappear")
		}
		b.State = 1
		h.Rooms[1].Objects[0].SetBonus(b)
		if !s.IsThisValid(1, 0) {
			t.Error("an untaken clock is valid")
		}
		h.Rooms[1].Objects[0].What = house.ObjectIsEmpty
	})
}

// TestDrawLocaleWithoutArt is the degenerate composition: no assets at all.
//
// It has to complete, because that is the state a fresh checkout is in, and it
// has to record the failure, because the game's only way to report a missing
// resource is Assets.Err -- the draw helpers are void, exactly as the original's
// are. What it must not do is panic on a nil surface part way through and leave
// the caller with no error to report.
//
// The picture that comes out is instructive rather than blank. An unlit room is
// black-filled without consulting any art, so the central room here is black; a
// lit one blits eight tiles out of the scratch map, which without a background
// PICT to decode into it is still the white a fresh GWorld holds. So the visible
// result of a missing background is a white room, which is also what the original
// would show if its RedAlert were removed.
func TestDrawLocaleWithoutArt(t *testing.T) {
	h := testHouse()
	s := NewScene(DefaultView(), NewAssets(filepath.Join(t.TempDir(), "absent")), h)
	s.RoomNumber = 1 // "centre": a panelled room with no light sources
	s.DrawLocale()

	if s.A.Err() == nil {
		t.Error("composing with no art should have recorded an error")
	}
	if s.Back.W != 640 || s.Back.H != 460 {
		t.Errorf("back map is %dx%d, want 640x460", s.Back.W, s.Back.H)
	}
	if s.NumLights != 0 {
		t.Fatalf("central room has %d lights, want 0: the rest of this test assumes it is dark", s.NumLights)
	}
	central := s.V.LocalRoomsDest[kCentralRoom]
	for v := central.Top; v < central.Bottom; v++ {
		for h := central.Left; h < central.Right; h++ {
			if px := s.Back.Pix[int(v)*s.Back.W+int(h)]; px != Black8 {
				t.Fatalf("(%d,%d) is index %d, want black: an unlit room does not need art", h, v, px)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// The golden hashes
// ---------------------------------------------------------------------------

// TestComposeEveryRoom composes every room of every shipped house and compares
// each one against a checked-in hash.
//
// What this does and does not prove is worth being exact about. It is a
// regression guard, not a fidelity test: the hashes are of *our* output, since
// there is no 1994 framebuffer to hash against. Fidelity was established by
// looking at the pictures beside the original's, and it is that judgement the
// hashes freeze. So a diff here is not automatically a bug -- it is a change
// that has to be looked at and explained.
//
// It is also the only test that exercises the whole composition. All 22 houses
// and every one of their rooms go through it, which reaches the paths a
// hand-written case never would: the houses that redefine application art, the
// off-palette pictures, the rooms whose saved-map table overflows, the
// off-the-map neighbours at every elevation.
func TestComposeEveryRoom(t *testing.T) {
	artDir := requireAssets(t, "art")
	houseDir := requireAssets(t, "houses")
	forkRoot := filepath.Join(assetRoot, "houseart")

	paths, err := filepath.Glob(filepath.Join(houseDir, "*.house"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Skipf("no houses in %s", houseDir)
	}
	sort.Strings(paths)

	// A fixed clock, so that the three clock faces and the calendar are the same
	// on every run. The date is the one on the shipped Demo House.
	clock := time.Date(1994, time.October, 14, 10, 9, 0, 0, time.UTC)

	var lines []string
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".house")
		h, err := house.LoadFile(path)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		assets := NewAssets(artDir)
		fork := filepath.Join(forkRoot, name)
		if st, err := os.Stat(fork); err == nil && st.IsDir() {
			assets.OpenHouseResFork(fork)
		}
		s := NewScene(DefaultView(), assets, h)
		s.Clock = clock

		for n := range h.Rooms {
			s.RoomNumber = int16(n)
			s.DrawLocale()
			lines = append(lines, fmt.Sprintf("%s|%d|%s|%s", name, n, digest(s.Back), tally(s)))
		}
		if err := assets.Err(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}

	if *update {
		if err := writeGolden(lines); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %d lines to %s", len(lines), goldenFile)
		return
	}

	want, err := readGolden()
	if err != nil {
		t.Fatalf("%v (run `go test ./internal/render -update` to create it)", err)
	}
	if len(want) != len(lines) {
		t.Errorf("composed %d rooms, golden file has %d: the corpus changed", len(lines), len(want))
	}
	bad := 0
	for i, got := range lines {
		if i >= len(want) {
			break
		}
		if got == want[i] {
			continue
		}
		bad++
		if bad <= 20 {
			t.Errorf("line %d:\n  have %s\n  want %s", i+1, got, want[i])
		}
	}
	if bad > 20 {
		t.Errorf("... and %d more differing rooms", bad-20)
	}
}

// digest hashes a surface's index plane. The indices are hashed rather than the
// RGB, because indices are what the game composites and what the palette maps;
// two different indices that happen to share a colour are still a difference.
func digest(s *Surface) string {
	sum := sha256.New()
	var hdr [8]byte
	binary.BigEndian.PutUint32(hdr[0:4], uint32(s.W))
	binary.BigEndian.PutUint32(hdr[4:8], uint32(s.H))
	sum.Write(hdr[:])
	sum.Write(s.Pix)
	return hex.EncodeToString(sum.Sum(nil))[:16]
}

// tally records the bookkeeping the composition did but did not draw. The
// dynamic tables are as much a product of DrawLocale as the pixels are -- the
// 24-slot saved-map cap decides which clocks appear at all -- so they are pinned
// beside the hash where a diff will show them.
func tally(s *Scene) string {
	return fmt.Sprintf("sm=%d fl=%d tk=%d co=%d pd=%d st=%d dy=%d mh=%d mr=%d",
		len(s.SavedMaps), len(s.Flames), len(s.TikiFlames), len(s.Coals),
		len(s.Pendulums), len(s.Stars), len(s.Dynamics), len(s.TempManholes),
		len(s.MirrorRects))
}

func readGolden() ([]string, error) {
	f, err := os.Open(goldenFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, sc.Err()
}

func writeGolden(lines []string) error {
	if err := os.MkdirAll(filepath.Dir(goldenFile), 0o777); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# gliderGo: one line per composed room, written by\n")
	b.WriteString("#   go test ./internal/render -update\n")
	b.WriteString("#\n")
	b.WriteString("# house|room|sha256-16(back map index plane)|dynamic-table sizes\n")
	b.WriteString("#\n")
	b.WriteString("# These pin our own output so that an unintended change in the renderer\n")
	b.WriteString("# shows up as a diff. They are not the original's pixels -- there is no\n")
	b.WriteString("# 1994 framebuffer to hash -- so a diff here is a change to explain, not\n")
	b.WriteString("# automatically a bug.\n")
	for _, l := range lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return os.WriteFile(goldenFile, []byte(b.String()), 0o666)
}
