package house

// Corpus tests: the Go loader is checked against the 22 shipped houses and, for
// every number it produces, against `docs/analysis/houses-inventory.md`.
//
// The golden numbers are *parsed out of that document at test time* rather than
// copied into this file. That is the point of the exercise. The inventory was
// produced by tools/probe_house.py -- a separate implementation, in a different
// language, written before this package existed -- so an agreement between the
// two is real evidence about the format, whereas a table copied from one into the
// other only proves that copying works. It also means the document cannot rot:
// edit a count in the markdown and this test fails.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// ------------------------------------------------------------------- locating

// repoRoot walks up from the package directory to the directory holding go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", dir)
		}
		dir = parent
	}
}

// corpus loads every shipped house once. The extracted houses are generated, not
// committed, so a missing directory is a skip with instructions rather than a
// failure -- a fresh clone should not look broken.
type corpusEntry struct {
	stem  string // "Art Museum"
	path  string
	raw   []byte
	house *House
}

func loadCorpus(t *testing.T) []corpusEntry {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "assets", "extracted", "houses")
	names, err := filepath.Glob(filepath.Join(dir, "*.house"))
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Skipf("no houses in %s -- run `make assets` first", dir)
	}
	sort.Strings(names)

	out := make([]corpusEntry, 0, len(names))
	for _, path := range names {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		h, err := Load(raw)
		if err != nil {
			t.Fatalf("%s: %v", filepath.Base(path), err)
		}
		out = append(out, corpusEntry{
			stem:  strings.TrimSuffix(filepath.Base(path), ".house"),
			path:  path,
			raw:   raw,
			house: h,
		})
	}
	return out
}

// --------------------------------------------------- reading the golden tables

// mdRows returns the data rows of the first markdown table after the line whose
// text contains `after`, split into trimmed cells. Separator rows (---) and the
// header row are dropped.
func mdRows(t *testing.T, path, after string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var rows [][]string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20) // the inventory has very long rows
	found, inTable, header := after == "", false, false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !found {
			if strings.Contains(line, after) {
				found = true
			}
			continue
		}
		if !strings.HasPrefix(line, "|") {
			if inTable {
				break // table ended
			}
			continue
		}
		inTable = true
		cells := splitRow(line)
		if !header {
			header = true
			continue
		}
		if strings.HasPrefix(cells[0], "---") {
			continue
		}
		rows = append(rows, cells)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatalf("%s: no table found after %q", filepath.Base(path), after)
	}
	return rows
}

func splitRow(line string) []string {
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func atoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(strings.ReplaceAll(s, ",", ""))
	if err != nil {
		t.Fatalf("not a number: %q", s)
	}
	return n
}

func inventoryPath(t *testing.T) string {
	return filepath.Join(repoRoot(t), "docs", "analysis", "houses-inventory.md")
}

// ----------------------------------------------------------------- the tests

// TestCorpusRoundTrip is the acceptance criterion for the codec: Save(Load(b))
// == b, byte for byte, for every shipped house -- residue, stale saved games,
// empty-slot union bytes and Sampler's PowerPC slack included.
func TestCorpusRoundTrip(t *testing.T) {
	for _, c := range loadCorpus(t) {
		out, err := c.house.Save()
		if err != nil {
			t.Errorf("%s: Save: %v", c.stem, err)
			continue
		}
		if len(out) != len(c.raw) {
			t.Errorf("%s: Save produced %d bytes, input was %d", c.stem, len(out), len(c.raw))
			continue
		}
		if diff := firstDiff(c.raw, out); diff >= 0 {
			t.Errorf("%s: byte %d differs: in=0x%02X out=0x%02X (%s)",
				c.stem, diff, c.raw[diff], out[diff], locate(diff))
		}
	}
}

func firstDiff(a, b []byte) int {
	for i := range a {
		if a[i] != b[i] {
			return i
		}
	}
	return -1
}

// locate turns a file offset into a human-readable position, so a round-trip
// failure names the field instead of just an offset.
func locate(off int) string {
	if off < SizeofHouseHeader {
		fields := []struct {
			at   int
			name string
		}{
			{offVersion, "version"}, {offUnusedShort, "unusedShort"},
			{offTimeStamp, "timeStamp"}, {offFlags, "flags"},
			{offInitial, "initial"}, {offBanner, "banner"},
			{offTrailer, "trailer"}, {offHighScores, "highScores"},
			{offSavedGame, "savedGame"}, {offHasGame, "hasGame"},
			{offUnusedBoolean, "unusedBoolean"}, {offFirstRoom, "firstRoom"},
			{offNRooms, "nRooms"},
		}
		name := "?"
		for _, f := range fields {
			if off >= f.at {
				name = f.name
			}
		}
		return "header." + name
	}
	rel := off - SizeofHouseHeader
	room, within := rel/SizeofRoom, rel%SizeofRoom
	if within >= offRoomObjects {
		o := within - offRoomObjects
		return fmt.Sprintf("room %d object %d byte %d", room, o/SizeofObject, o%SizeofObject)
	}
	return fmt.Sprintf("room %d byte %d", room, within)
}

// TestCorpusInventory checks the loader against the per-house table: file size,
// version, room count, live-object count, distinct `what` codes, and the name of
// the first room -- which incidentally exercises PStr28 and Mac Roman decoding.
func TestCorpusInventory(t *testing.T) {
	corpus := loadCorpus(t)
	byStem := make(map[string]corpusEntry, len(corpus))
	for _, c := range corpus {
		byStem[c.stem] = c
	}

	rows := mdRows(t, inventoryPath(t), "| # | House file |")
	if len(rows) != len(corpus) {
		t.Fatalf("inventory lists %d houses, corpus has %d", len(rows), len(corpus))
	}

	for _, r := range rows {
		// | # | House file | .binhex B | data fork B | rsrc fork B | ver |
		// rooms | floors | suites | objects | distinct types | start room | author |
		stem := strings.TrimSuffix(strings.Trim(r[1], "`"), ".binhex")
		c, ok := byStem[stem]
		if !ok {
			t.Errorf("inventory names %q but no %s.house was extracted", stem, stem)
			continue
		}
		h := c.house

		if got, want := len(c.raw), atoi(t, r[3]); got != want {
			t.Errorf("%s: data fork is %d bytes, inventory says %d", stem, got, want)
		}
		if got, want := fmt.Sprintf("0x%04x", uint16(h.Version)), strings.ToLower(r[5]); got != want {
			t.Errorf("%s: version %s, inventory says %s", stem, got, want)
		}
		if got, want := len(h.Rooms), atoi(t, r[6]); got != want {
			t.Errorf("%s: %d rooms, inventory says %d", stem, got, want)
		}

		live, kinds := 0, map[int16]bool{}
		for i := range h.Rooms {
			for _, o := range h.Rooms[i].Objects {
				if !o.IsEmpty() {
					live++
					kinds[o.What] = true
				}
			}
		}
		if want := atoi(t, r[9]); live != want {
			t.Errorf("%s: %d live objects, inventory says %d", stem, live, want)
		}
		if want := atoi(t, r[10]); len(kinds) != want {
			t.Errorf("%s: %d distinct what codes, inventory says %d", stem, len(kinds), want)
		}

		// start room: `91 "Art Museum"`, with the name elided as `…` when long.
		num, name := splitStartRoom(t, r[11])
		if int(h.FirstRoom) != num {
			t.Errorf("%s: firstRoom %d, inventory says %d", stem, h.FirstRoom, num)
		}
		if num >= 0 && num < len(h.Rooms) {
			got := h.Rooms[num].Name.Text()
			if trunc := strings.TrimSuffix(name, "…"); trunc != name {
				if !strings.HasPrefix(got, trunc) {
					t.Errorf("%s: room %d is %q, inventory says %q…", stem, num, got, trunc)
				}
			} else if got != name {
				t.Errorf("%s: room %d is %q, inventory says %q", stem, num, got, name)
			}
		}
	}
}

func splitStartRoom(t *testing.T, cell string) (int, string) {
	t.Helper()
	i := strings.Index(cell, `"`)
	if i < 0 {
		t.Fatalf("unparseable start room cell %q", cell)
	}
	return atoi(t, strings.TrimSpace(cell[:i])), strings.Trim(cell[i:], `"`)
}

// TestCorpusWhatHistogram checks every one of the 117 defined object types,
// twice over: the total number of instances and the number of houses using it.
// It also checks that the Go name table agrees with the document's, which is how
// a typo in one of 117 hand-transcribed identifiers gets caught.
func TestCorpusWhatHistogram(t *testing.T) {
	corpus := loadCorpus(t)

	counts := map[int16]int{}
	houses := map[int16]map[string]bool{}
	for _, c := range corpus {
		for i := range c.house.Rooms {
			for _, o := range c.house.Rooms[i].Objects {
				if o.IsEmpty() {
					continue
				}
				counts[o.What]++
				if houses[o.What] == nil {
					houses[o.What] = map[string]bool{}
				}
				houses[o.What][c.stem] = true
			}
		}
	}

	rows := mdRows(t, inventoryPath(t), "### Full corpus `what` histogram")
	seen := map[int16]bool{}
	total := 0
	for _, r := range rows {
		code, err := strconv.ParseUint(strings.TrimPrefix(r[0], "0x"), 16, 16)
		if err != nil {
			t.Fatalf("bad code %q", r[0])
		}
		what := int16(code)
		seen[what] = true
		want, wantHouses := atoi(t, r[2]), atoi(t, r[3])
		total += want

		if name := ObjectName(what); name != r[1] {
			t.Errorf("0x%02X: name table says %q, inventory says %q", what, name, r[1])
		}
		if got := counts[what]; got != want {
			t.Errorf("0x%02X %s: %d instances, inventory says %d", what, r[1], got, want)
		}
		if got := len(houses[what]); got != wantHouses {
			t.Errorf("0x%02X %s: used by %d houses, inventory says %d", what, r[1], got, wantHouses)
		}
		if GroupOf(what) == GroupNone {
			t.Errorf("0x%02X %s: classified as GroupNone but it is a shipped object", what, r[1])
		}
	}

	// Every code the corpus contains must appear in the histogram, and vice
	// versa: a code in one and not the other means the two readers disagree
	// about which bytes are a `what`.
	for what := range counts {
		if !seen[what] {
			t.Errorf("corpus contains 0x%02X (%d times) but the histogram does not list it",
				what, counts[what])
		}
	}
	if len(rows) != len(ObjectNames()) {
		t.Errorf("histogram lists %d codes, the Go name table defines %d",
			len(rows), len(ObjectNames()))
	}
	if total != 31440 {
		t.Errorf("histogram counts sum to %d, corpus totals say 31440", total)
	}
}

// TestCorpusTotals pins the four numbers quoted throughout the documentation, so
// that anything which changes them has to change them here too.
func TestCorpusTotals(t *testing.T) {
	corpus := loadCorpus(t)

	rooms, objects, stars := 0, 0, 0
	kinds := map[int16]bool{}
	for _, c := range corpus {
		rooms += len(c.house.Rooms)
		for i := range c.house.Rooms {
			for _, o := range c.house.Rooms[i].Objects {
				if o.IsEmpty() {
					continue
				}
				objects++
				kinds[o.What] = true
				if o.What == 0x2C { // kStar
					stars++
				}
			}
		}
	}

	// Parsed from the "## Corpus totals" bullet list rather than hardcoded.
	want := map[string]int{}
	body, err := os.ReadFile(inventoryPath(t))
	if err != nil {
		t.Fatal(err)
	}
	after := string(body)[strings.Index(string(body), "## Corpus totals"):]
	for _, line := range strings.Split(after, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			if strings.HasPrefix(line, "#") && len(want) > 0 {
				break
			}
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(line, "- "), ":", 2)
		if len(parts) != 2 {
			continue
		}
		num := strings.Fields(strings.TrimSpace(parts[1]))
		if len(num) == 0 {
			continue
		}
		if n, err := strconv.Atoi(num[0]); err == nil {
			want[parts[0]] = n
		}
	}

	for _, tc := range []struct {
		key string
		got int
	}{
		{"houses", len(corpus)},
		{"rooms", rooms},
		{"live objects", objects},
		{"distinct `what` codes used", len(kinds)},
		{"stars", stars},
	} {
		w, ok := want[tc.key]
		if !ok {
			t.Errorf("corpus totals do not mention %q", tc.key)
			continue
		}
		if tc.got != w {
			t.Errorf("%s: loader says %d, corpus totals say %d", tc.key, tc.got, w)
		}
	}
}

// TestCorpusNumObjectsJives checks the claim that `numObjects` agrees with the
// live-slot count in every shipped room. The field is advisory -- the original
// recomputes it (HouseLegal.c:899-907) -- so this is not a format requirement.
// It is pinned anyway: if it ever fails for a house we ship, the count we render
// from is the wrong one.
func TestCorpusNumObjectsJives(t *testing.T) {
	for _, c := range loadCorpus(t) {
		for i := range c.house.Rooms {
			r := &c.house.Rooms[i]
			if int(r.NumObjects) != r.LiveObjects() {
				t.Errorf("%s room %d (%q): numObjects=%d, live slots=%d",
					c.stem, i, r.Name.Text(), r.NumObjects, r.LiveObjects())
			}
		}
	}
}

// TestCorpusNonCompacted records the rooms whose object slots have holes. These
// are the reason Objects is a fixed [24]Object and not a slice: compacting them
// would renumber slots that links address by index. The count is asserted so
// that a "helpful" normalisation somewhere in the loader shows up as a failure
// here rather than as broken links in a house nobody tests.
func TestCorpusNonCompacted(t *testing.T) {
	var holes []string
	for _, c := range loadCorpus(t) {
		for i := range c.house.Rooms {
			if !c.house.Rooms[i].Compacted() {
				holes = append(holes, fmt.Sprintf("%s room %d (%q)",
					c.stem, i, c.house.Rooms[i].Name.Text()))
			}
		}
	}
	const want = 6
	if len(holes) != want {
		t.Errorf("%d rooms have holes in objects[], expected %d:\n  %s",
			len(holes), want, strings.Join(holes, "\n  "))
	}
}

// TestCorpusSlack pins the one house with trailing bytes. Sampler was saved by a
// PowerPC build that sized the write with sizeof(houseType) rather than
// offsetof(houseType, rooms) -- see house.go and house-format.md 12.4.
func TestCorpusSlack(t *testing.T) {
	for _, c := range loadCorpus(t) {
		want := 0
		if c.stem == "Sampler" {
			want = PowerPCSlack
		}
		if got := len(c.house.Slack); got != want {
			t.Errorf("%s: %d slack bytes, expected %d", c.stem, got, want)
		}
	}
}

// TestCorpusRoomInvariants pins the four room fields the analysis found to be
// constant or near-constant across all 4,070 rooms (house-format.md 8.4). They
// are the fields a renderer would otherwise be tempted to interpret.
func TestCorpusRoomInvariants(t *testing.T) {
	visited := map[byte]int{}
	for _, c := range loadCorpus(t) {
		for i := range c.house.Rooms {
			r := &c.house.Rooms[i]
			if r.UnusedByte != 0 {
				t.Errorf("%s room %d: unusedByte is %d, expected 0", c.stem, i, r.UnusedByte)
			}
			if r.Openings != 0 {
				t.Errorf("%s room %d: openings is %d, expected 0", c.stem, i, r.Openings)
			}
			if r.Visited > 1 {
				t.Errorf("%s room %d: visited is %d, expected 0 or 1", c.stem, i, r.Visited)
			}
			visited[r.Visited]++
			if n := int(r.Name[0]); n < 1 || n > 27 {
				t.Errorf("%s room %d: name length byte is %d, expected 1..27", c.stem, i, n)
			}
		}
	}
	// house-format.md:310 counted these exactly.
	if visited[0] != 3634 || visited[1] != 436 {
		t.Errorf("visited distribution is %v, house-format.md 6.2 says 3634 zeros / 436 ones",
			visited)
	}
}
