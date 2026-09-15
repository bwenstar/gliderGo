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

	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/replay"
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

// The `replay` subcommand.
//
// internal/replay tests the harness; what is left here is the argument handling, and the
// one piece of it that is genuinely easy to get wrong: the flags are an overlay on a script
// file, so a flag nobody typed must not silently reset a field the file set. That is why
// the numeric flags default to out-of-range sentinels, and it is the kind of thing that
// works until somebody "tidies" a default and then quietly reproduces the wrong run.
//
// These use the synthesised fixture where they can, so that a checkout without
// `make assets` still runs them.

// TestReplayResolvesScriptThenFlags is the overlay rule, in both directions.
func TestReplayResolvesScriptThenFlags(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "bug.script")
	const original = "house CD Demo House\nseed 7\nframes 120\nneighbors 1\nroom 4\nwhere 420 20\nfacing left\nat 0 right\nat 45 -\n"
	if err := os.WriteFile(scriptPath, []byte(original), 0o666); err != nil {
		t.Fatal(err)
	}

	// -script writes the resolved script and exits before loading anything, so this needs
	// no house and no art on disk.
	resolve := func(t *testing.T, args ...string) *replay.Script {
		t.Helper()
		outPath := filepath.Join(t.TempDir(), "resolved.script")
		if err := run(append([]string{"replay", "-script", outPath}, args...)); err != nil {
			t.Fatalf("replay -script: %v", err)
		}
		f, err := os.Open(outPath)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		s, err := replay.Parse(f)
		if err != nil {
			t.Fatalf("the tool wrote a script it cannot read back: %v", err)
		}
		return s
	}

	// Untouched: every field survives the trip through the tool.
	kept := resolve(t, scriptPath)
	if kept.House != "CD Demo House" || kept.Seed != 7 || kept.Frames != 120 || kept.Neighbors != 1 {
		t.Errorf("run settings changed: %q/%d/%d/%d", kept.House, kept.Seed, kept.Frames, kept.Neighbors)
	}
	if kept.Room != 4 || kept.Where.H != 420 || kept.Where.V != 20 || kept.Facing != 0 {
		t.Errorf("start point changed: room %d at %v facing %d", kept.Room, kept.Where, kept.Facing)
	}
	if len(kept.Input) != 2 {
		t.Fatalf("%d holds, want 2", len(kept.Input))
	}
	if !kept.Input[0].P1.Right || kept.Input[1].P1 != (player.Keys{}) {
		t.Errorf("keystroke log changed: %+v", kept.Input)
	}

	// Overridden: only the fields named on the command line move.
	over := resolve(t, "-frames", "600", "-seed", "1", "-two", "-room", "9", scriptPath)
	if over.Frames != 600 || over.Seed != 1 || !over.TwoPlayer || over.Room != 9 {
		t.Errorf("overrides not applied: %d frames, seed %d, two %v, room %d",
			over.Frames, over.Seed, over.TwoPlayer, over.Room)
	}
	if over.Neighbors != 1 || over.House != "CD Demo House" || over.Where.H != 420 {
		t.Errorf("overrode fields nobody named: %d-room, house %q, where %v",
			over.Neighbors, over.House, over.Where)
	}
	// Seed 0 is a legal seed and the sentinel is -1, so `-seed 0` has to reach the script.
	if z := resolve(t, "-seed", "0", scriptPath); z.Seed != 0 {
		t.Errorf("-seed 0 gave seed %d: zero is a seed, not an absent flag", z.Seed)
	}
}

// TestReplayNeedsAHouse: the one flag with no default worth guessing.
func TestReplayNeedsAHouse(t *testing.T) {
	if err := run([]string{"replay", "-script", "-"}); err == nil {
		t.Error("replay with no house succeeded; want an error")
	}
	if err := run([]string{"replay", "-where", "1,2", "-script", "-", "-house", "H"}); err == nil {
		t.Error("-where without -room succeeded; want an error")
	}
	if err := run([]string{"replay", filepath.Join(t.TempDir(), "absent.script")}); err == nil {
		t.Error("replay of a missing script succeeded; want an error")
	}
}

// TestReplayRunsTheFixtureHouse drives the whole command against a real house file, with no
// art tree at all.
//
// The missing art is the point of the case, not a shortcut: replay's contract is that a
// sticky asset failure still returns a usable result, because "every room composed empty"
// is a symptom the missing PICT explains and the trace is the evidence. So this asserts
// both halves -- the command reports the failure, and it printed the run first.
func TestReplayRunsTheFixtureHouse(t *testing.T) {
	_, housePath := writeFixture(t)
	outPath := filepath.Join(t.TempDir(), "trace.txt")
	err := run([]string{"replay", "-house", housePath, "-frames", "12", "-neighbors", "1",
		"-art", filepath.Join(t.TempDir(), "no-art"), "-trace", "-o", outPath})
	if err == nil {
		t.Error("a missing art tree did not report an error")
	}

	body, rerr := os.ReadFile(outPath)
	if rerr != nil {
		t.Fatalf("no trace was written despite the run completing: %v", rerr)
	}
	text := string(body)
	for _, want := range []string{"# gliderGo replay trace", "# digest=", "f=0 ", "f=12 ", "# end frames=12"} {
		if !strings.Contains(text, want) {
			t.Errorf("trace has no %q:\n%s", want, text)
		}
	}
	// Thirteen lines for twelve frames: frame 0 is the state before the loop ran, so a
	// trace of N frames has N+1 samples. Pinned because it is the sort of off-by-one that
	// makes two traces of the same run look different lengths.
	if n := strings.Count(text, "\nf="); n != 13 {
		t.Errorf("%d frame lines, want 13 (frames 0 through 12)", n)
	}
}

// TestReplayDigestIsStableAndShort is what a bug report quotes, so it has to be one token.
func TestReplayDigestIsStableAndShort(t *testing.T) {
	_, housePath := writeFixture(t)
	noArt := filepath.Join(t.TempDir(), "no-art")
	digest := func(t *testing.T) string {
		t.Helper()
		outPath := filepath.Join(t.TempDir(), "d.txt")
		// The error is the missing art tree and is asserted in the case above.
		_ = run([]string{"replay", "-house", housePath, "-frames", "8", "-neighbors", "1",
			"-art", noArt, "-digest", "-o", outPath})
		body, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(body))
	}
	got := digest(t)
	if len(got) != 16 {
		t.Errorf("digest %q is %d characters, want 16", got, len(got))
	}
	if strings.ContainsAny(got, " \t\n") {
		t.Errorf("digest %q is not a single token", got)
	}
	if again := digest(t); again != got {
		t.Errorf("the same command gave %s then %s", got, again)
	}
}

// TestReplaySummaryIsTheDefault, and -o does not change which output is produced.
//
// -o used to imply -trace, which meant a summary could only ever reach the terminal. The
// coupling is worth a test because it is the sort of convenience that gets re-added.
func TestReplaySummaryIsTheDefault(t *testing.T) {
	_, housePath := writeFixture(t)
	outPath := filepath.Join(t.TempDir(), "summary.txt")
	_ = run([]string{"replay", "-house", housePath, "-frames", "6", "-neighbors", "1",
		"-art", filepath.Join(t.TempDir(), "no-art"), "-o", outPath})
	body, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "f=0 ") {
		t.Errorf("-o alone produced a trace; it should only redirect the summary:\n%s", text)
	}
	for _, want := range []string{"6 frames", "1-room", "1 player(s)", "ended   room", "digest "} {
		if !strings.Contains(text, want) {
			t.Errorf("summary has no %q:\n%s", want, text)
		}
	}
}
