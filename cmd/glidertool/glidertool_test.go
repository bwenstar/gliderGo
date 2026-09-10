package main

// The package's own tests. The house codec is tested exhaustively against the
// 22 shipped houses in internal/house; what is left to check here is the command
// plumbing -- that `dump` and `build` are actually inverses when driven through
// argument parsing and files, and that `check` reports the two conditions it is
// there to report. The fixture is synthesised rather than read from
// assets/extracted, so these tests pass on a checkout that has not run
// `make assets`.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"glidergo/internal/house"
)

// fixture is a two-room house exercising the things the tool has to survive: a
// room whose object slots have a hole in them, both field arities (`at` and
// `rect`), a negative floor, a link that addresses another room's slot, and a
// room name that needs Mac Roman (0xD5 is a right single quote).
const fixture = `format 1

house
    version    0x0200
    timestamp  743682113
    flags      0x00000006
    initial    107 49
    firstroom  1
    banner     "Glidertool fixture"
    trailer    "The end"

room 0 "Cellar"
    bounds     0x001f
    background 3000
    tiles      0 1 2 3 4 5 6 7
    floor      -1
    suite      60
    start      10 20
    object 0   kFloorVent   at 305 171  distance 5  initial 1 state 1 vector 1 tall 0
    object 1   kTable       rect 154 217 160 260  pict 0
    object 5   kStar        at 137 310  length 0  points 0  state 1  initial 1

room 1 "Attic\xd5s Reprise"
    bounds     0x0003
    background 3001
    floor      0
    suite      61
    object 0   kInvisTrans  at 100 200  tall 40  where 0  who 2  wide 30
`

func writeFixture(t *testing.T) (dir, housePath string) {
	t.Helper()
	dir = t.TempDir()
	h, err := house.ParseText(strings.NewReader(fixture))
	if err != nil {
		t.Fatalf("fixture does not parse: %v", err)
	}
	housePath = filepath.Join(dir, "fixture.house")
	if err := h.SaveFile(housePath); err != nil {
		t.Fatal(err)
	}
	return dir, housePath
}

// TestDumpBuildRoundTrip drives the two commands the plan names, through their
// flags, and requires the bytes back.
func TestDumpBuildRoundTrip(t *testing.T) {
	dir, housePath := writeFixture(t)
	original, err := os.ReadFile(housePath)
	if err != nil {
		t.Fatal(err)
	}

	for _, mode := range []struct {
		name string
		args []string
	}{
		{"clean", []string{"dump", "-o", filepath.Join(dir, "clean.txt"), housePath}},
		{"residue", []string{"dump", "-residue", "-o", filepath.Join(dir, "residue.txt"), housePath}},
		{"bare", []string{"dump", "-residue", "-bare", "-o", filepath.Join(dir, "bare.txt"), housePath}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			if err := houseCmd(mode.args); err != nil {
				t.Fatalf("dump: %v", err)
			}
			txt := mode.args[len(mode.args)-2]
			out := filepath.Join(dir, mode.name+".house")
			if err := houseCmd([]string{"build", "-q", "-o", out, txt}); err != nil {
				t.Fatalf("build: %v", err)
			}
			got, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(original) {
				t.Fatalf("rebuilt house is %d bytes, original is %d", len(got), len(original))
			}
			if d := firstDiff(original, got); d >= 0 {
				t.Errorf("byte %d differs: 0x%02X -> 0x%02X (%s)",
					d, original[d], got[d], locate(d, 2))
			}
		})
	}

	// The header is the only difference between -bare and the default, so a bare
	// dump must be a strict suffix of a full one.
	full, err := os.ReadFile(filepath.Join(dir, "residue.txt"))
	if err != nil {
		t.Fatal(err)
	}
	bare, err := os.ReadFile(filepath.Join(dir, "bare.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(full), string(bare)) {
		t.Error("-bare output is not the full output minus its header")
	}
	// -bare drops the explanatory preamble, not the per-field annotations: those
	// are what makes a dump readable next to the format document.
	if !strings.HasPrefix(string(bare), "format 1\n") {
		t.Errorf("-bare output starts with %q", firstLine(string(bare)))
	}
	if !strings.Contains(string(bare), "# kHouseVersion") {
		t.Error("-bare dropped the field annotations, which it should keep")
	}
}

func TestCheckPasses(t *testing.T) {
	_, housePath := writeFixture(t)
	problems, notes := checkFile(housePath)
	if len(problems) != 0 {
		t.Errorf("clean fixture reported problems: %v", problems)
	}
	// Room 0 leaves slots 2..4 empty, which is legal and worth saying.
	if !hasNote(notes, "holes in objects[]") {
		t.Errorf("expected a note about the non-compacted room, got %v", notes)
	}
}

// TestCheckFindsTruncation and TestCheckFindsUndefined cover the two kinds of
// finding: a file the codec cannot read at all, and a file it reads fine but
// which contains something no shipped house does.
func TestCheckFindsTruncation(t *testing.T) {
	dir, housePath := writeFixture(t)
	raw, err := os.ReadFile(housePath)
	if err != nil {
		t.Fatal(err)
	}
	short := filepath.Join(dir, "short.house")
	if err := os.WriteFile(short, raw[:len(raw)-10], 0o644); err != nil {
		t.Fatal(err)
	}
	problems, _ := checkFile(short)
	if len(problems) == 0 {
		t.Fatal("a truncated house passed check")
	}
	if !strings.Contains(problems[0], "truncated") {
		t.Errorf("problem is %q, expected it to mention truncation", problems[0])
	}
}

func TestCheckFindsUndefinedWhat(t *testing.T) {
	dir, _ := writeFixture(t)
	h, err := house.ParseText(strings.NewReader(fixture +
		"    object 3 0x50 raw 01 02 03 04 05 06 07 08 09 0a\n"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "odd.house")
	if err := h.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	problems, notes := checkFile(path)
	if len(problems) != 0 {
		t.Errorf("an undefined `what` is legal in the file; problems: %v", problems)
	}
	if !hasNote(notes, "none of the nine ranges") {
		t.Errorf("expected a note about the undefined type, got %v", notes)
	}
}

func hasNote(notes []string, substr string) bool {
	for _, n := range notes {
		if strings.Contains(n, substr) {
			return true
		}
	}
	return false
}

// TestLocate pins the offset-to-field mapping, since it is the thing a
// round-trip failure message is read for and it is easy to get off by one room.
func TestLocate(t *testing.T) {
	const room = house.SizeofHouseHeader
	const objects = room + house.SizeofRoom - house.MaxRoomObs*house.SizeofObject
	cases := []struct {
		off  int
		want string
	}{
		{0, "header byte 0"},
		{house.SizeofHouseHeader - 1, "header byte 865"},
		{room, "room 0 byte 0"},
		{objects, "room 0 object 0 byte 0"},
		{objects + house.SizeofObject + 3, "room 0 object 1 byte 3"},
		{room + house.SizeofRoom, "room 1 byte 0"},
		{room + 2*house.SizeofRoom, "trailing byte 0 past room 1"},
	}
	for _, tc := range cases {
		if got := locate(tc.off, 2); got != tc.want {
			t.Errorf("locate(%d, 2) = %q, want %q", tc.off, got, tc.want)
		}
	}
}

// TestGroupFieldsMatchParser is the anti-drift check for the `types` table: the
// fields it advertises must be exactly the fields the parser demands. It is
// checked by building an object line from the advertised list and parsing it.
func TestGroupFieldsMatchParser(t *testing.T) {
	for code, name := range house.ObjectNames() {
		g := house.GroupOf(code)
		var line strings.Builder
		line.WriteString("format 1\nroom 0 \"a\"\n    object 0 " + name)
		for _, f := range g.Fields() {
			line.WriteString(" " + f)
			for i := 0; i < house.FieldArity(f); i++ {
				line.WriteString(" 1")
			}
		}
		line.WriteString("\n")
		if _, err := house.ParseText(strings.NewReader(line.String())); err != nil {
			t.Errorf("0x%02X %s: the advertised fields %v do not satisfy the parser: %v",
				uint16(code), name, g.Fields(), err)
		}
		if got := groupFields(g); got == "" {
			t.Errorf("0x%02X %s: no field help", uint16(code), name)
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
