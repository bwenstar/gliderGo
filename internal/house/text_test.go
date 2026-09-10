package house

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// TestTextRoundTripFields is the acceptance criterion for the text format:
// parsing what the writer wrote reproduces every named field of every shipped
// house. The comparison target is Canonical(), which differs from the input only
// in the bytes the format deliberately drops -- see the package comment in
// text.go for why dropping them is the right call.
func TestTextRoundTripFields(t *testing.T) {
	for _, c := range loadCorpus(t) {
		txt, err := c.house.Text(TextOptions{})
		if err != nil {
			t.Errorf("%s: WriteText: %v", c.stem, err)
			continue
		}
		back, err := ParseText(strings.NewReader(txt))
		if err != nil {
			t.Errorf("%s: ParseText: %v", c.stem, err)
			continue
		}
		want := c.house.Canonical()
		if diff := diffHouses(want, back); diff != "" {
			t.Errorf("%s: text round trip lost a field: %s", c.stem, diff)
		}
	}
}

// TestTextRoundTripExact is the stronger claim for the residue-preserving mode:
// text out, text in, and the binary is byte-identical to the file we started
// from. This is what makes the text format usable for forensics, not just for
// authoring.
func TestTextRoundTripExact(t *testing.T) {
	for _, c := range loadCorpus(t) {
		txt, err := c.house.Text(TextOptions{Residue: true})
		if err != nil {
			t.Errorf("%s: WriteText: %v", c.stem, err)
			continue
		}
		back, err := ParseText(strings.NewReader(txt))
		if err != nil {
			t.Errorf("%s: ParseText: %v", c.stem, err)
			continue
		}
		out, err := back.Save()
		if err != nil {
			t.Errorf("%s: Save: %v", c.stem, err)
			continue
		}
		if len(out) != len(c.raw) {
			t.Errorf("%s: %d bytes after a text round trip, input was %d",
				c.stem, len(out), len(c.raw))
			continue
		}
		if d := firstDiff(c.raw, out); d >= 0 {
			t.Errorf("%s: byte %d differs after a text round trip: in=0x%02X out=0x%02X (%s)",
				c.stem, d, c.raw[d], out[d], locate(d))
		}
	}
}

// TestTextIdempotent checks that the writer is a fixed point: canonical house ->
// text -> house -> text produces the identical text. Without this, a house
// checked into git would churn on every save.
func TestTextIdempotent(t *testing.T) {
	for _, c := range loadCorpus(t) {
		for _, opt := range []TextOptions{{}, {Residue: true}} {
			first, err := c.house.Text(opt)
			if err != nil {
				t.Fatalf("%s: %v", c.stem, err)
			}
			back, err := ParseText(strings.NewReader(first))
			if err != nil {
				t.Fatalf("%s: %v", c.stem, err)
			}
			second, err := back.Text(opt)
			if err != nil {
				t.Fatalf("%s: %v", c.stem, err)
			}
			if first != second {
				t.Errorf("%s (residue=%v): writer is not idempotent: %s",
					c.stem, opt.Residue, firstLineDiff(first, second))
			}
		}
	}
}

// TestCanonicalIsMinimal checks that Canonical() zeroes exactly the bytes the
// text format drops and nothing more: canonicalising twice changes nothing, and
// a canonical house still saves to a file of the same size.
func TestCanonicalIsMinimal(t *testing.T) {
	for _, c := range loadCorpus(t) {
		once := c.house.Canonical()
		twice := once.Canonical()
		if !reflect.DeepEqual(once, twice) {
			t.Errorf("%s: Canonical is not idempotent", c.stem)
		}
		b, err := once.Save()
		if err != nil {
			t.Fatalf("%s: %v", c.stem, err)
		}
		if len(b) != len(c.raw) {
			t.Errorf("%s: canonical house saves to %d bytes, original is %d",
				c.stem, len(b), len(c.raw))
		}
		// Canonicalising must not touch the original.
		if d := firstDiff(c.raw, mustSave(t, c.house)); d >= 0 {
			t.Errorf("%s: Canonical mutated its receiver at byte %d", c.stem, d)
		}
	}
}

func mustSave(t *testing.T, h *House) []byte {
	t.Helper()
	b, err := h.Save()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// diffHouses reports the first field that differs, named, because a DeepEqual
// failure on a 500-room house is unreadable.
func diffHouses(a, b *House) string {
	if a.Version != b.Version {
		return fmt.Sprintf("version %d != %d", a.Version, b.Version)
	}
	if a.UnusedShort != b.UnusedShort {
		return fmt.Sprintf("unusedShort %d != %d", a.UnusedShort, b.UnusedShort)
	}
	if a.TimeStamp != b.TimeStamp {
		return fmt.Sprintf("timeStamp %d != %d", a.TimeStamp, b.TimeStamp)
	}
	if a.Flags != b.Flags {
		return fmt.Sprintf("flags 0x%x != 0x%x", a.Flags, b.Flags)
	}
	if a.Initial != b.Initial {
		return fmt.Sprintf("initial %v != %v", a.Initial, b.Initial)
	}
	if a.Banner != b.Banner {
		return fmt.Sprintf("banner %q != %q", a.Banner.Text(), b.Banner.Text())
	}
	if a.Trailer != b.Trailer {
		return fmt.Sprintf("trailer %q != %q", a.Trailer.Text(), b.Trailer.Text())
	}
	if a.HighScores != b.HighScores {
		return diffScores(&a.HighScores, &b.HighScores)
	}
	if a.SavedGame != b.SavedGame {
		return fmt.Sprintf("savedGame %+v != %+v", a.SavedGame, b.SavedGame)
	}
	if a.HasGame != b.HasGame {
		return fmt.Sprintf("hasGame %d != %d", a.HasGame, b.HasGame)
	}
	if a.UnusedBoolean != b.UnusedBoolean {
		return fmt.Sprintf("unusedBoolean %d != %d", a.UnusedBoolean, b.UnusedBoolean)
	}
	if a.FirstRoom != b.FirstRoom {
		return fmt.Sprintf("firstRoom %d != %d", a.FirstRoom, b.FirstRoom)
	}
	if a.NRooms != b.NRooms {
		return fmt.Sprintf("nRooms %d != %d", a.NRooms, b.NRooms)
	}
	if len(a.Rooms) != len(b.Rooms) {
		return fmt.Sprintf("%d rooms != %d", len(a.Rooms), len(b.Rooms))
	}
	if !reflect.DeepEqual(a.Slack, b.Slack) {
		return fmt.Sprintf("slack %v != %v", a.Slack, b.Slack)
	}
	for i := range a.Rooms {
		if d := diffRooms(&a.Rooms[i], &b.Rooms[i]); d != "" {
			return fmt.Sprintf("room %d (%q): %s", i, a.Rooms[i].Name.Text(), d)
		}
	}
	return ""
}

func diffScores(a, b *Scores) string {
	if a.Banner != b.Banner {
		return fmt.Sprintf("scores banner %q != %q", a.Banner.Text(), b.Banner.Text())
	}
	for i := range a.Names {
		if a.Names[i] != b.Names[i] {
			return fmt.Sprintf("score %d name %q != %q", i, a.Names[i].Text(), b.Names[i].Text())
		}
		if a.Scores[i] != b.Scores[i] {
			return fmt.Sprintf("score %d value %d != %d", i, a.Scores[i], b.Scores[i])
		}
		if a.TimeStamps[i] != b.TimeStamps[i] {
			return fmt.Sprintf("score %d timestamp %d != %d", i, a.TimeStamps[i], b.TimeStamps[i])
		}
		if a.Levels[i] != b.Levels[i] {
			return fmt.Sprintf("score %d rooms %d != %d", i, a.Levels[i], b.Levels[i])
		}
	}
	return "scores differ in an unnamed way"
}

func diffRooms(a, b *Room) string {
	switch {
	case a.Name != b.Name:
		return fmt.Sprintf("name %q != %q", a.Name.Text(), b.Name.Text())
	case a.Bounds != b.Bounds:
		return fmt.Sprintf("bounds 0x%x != 0x%x", uint16(a.Bounds), uint16(b.Bounds))
	case a.LeftStart != b.LeftStart:
		return fmt.Sprintf("leftStart %d != %d", a.LeftStart, b.LeftStart)
	case a.RightStart != b.RightStart:
		return fmt.Sprintf("rightStart %d != %d", a.RightStart, b.RightStart)
	case a.UnusedByte != b.UnusedByte:
		return fmt.Sprintf("unusedByte %d != %d", a.UnusedByte, b.UnusedByte)
	case a.Visited != b.Visited:
		return fmt.Sprintf("visited %d != %d", a.Visited, b.Visited)
	case a.Background != b.Background:
		return fmt.Sprintf("background %d != %d", a.Background, b.Background)
	case a.Tiles != b.Tiles:
		return fmt.Sprintf("tiles %v != %v", a.Tiles, b.Tiles)
	case a.Floor != b.Floor:
		return fmt.Sprintf("floor %d != %d", a.Floor, b.Floor)
	case a.Suite != b.Suite:
		return fmt.Sprintf("suite %d != %d", a.Suite, b.Suite)
	case a.Openings != b.Openings:
		return fmt.Sprintf("openings %d != %d", a.Openings, b.Openings)
	case a.NumObjects != b.NumObjects:
		return fmt.Sprintf("numObjects %d != %d", a.NumObjects, b.NumObjects)
	}
	for i := range a.Objects {
		if a.Objects[i] != b.Objects[i] {
			return fmt.Sprintf("object %d (%s): %v != %v",
				i, a.Objects[i], a.Objects[i], b.Objects[i])
		}
	}
	return ""
}

func firstLineDiff(a, b string) string {
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(la) && i < len(lb); i++ {
		if la[i] != lb[i] {
			return fmt.Sprintf("line %d:\n  -%s\n  +%s", i+1, la[i], lb[i])
		}
	}
	return fmt.Sprintf("%d lines vs %d", len(la), len(lb))
}

// ------------------------------------------------------- hand-authoring checks

// TestParseMinimalHouse is the authoring case: the smallest thing a person could
// reasonably type that is still a playable house. Everything unstated must
// default, which is the property that makes the format writable by hand.
func TestParseMinimalHouse(t *testing.T) {
	const src = `
format 1

house
    version    0x0200
    firstroom  0
    banner     "Hello"
    trailer    "Bye"
    initial    100 200

room 0 "Start"
    bounds     0x001f
    background 2000
    tiles      0 1 2 3 4 5 6 7
    floor      0
    suite      60
    start      10 20
    object 0   kFloorVent  at 200 64  distance 5  initial 1 state 1 vector 0 tall 0
    object 4   kTable      rect 100 50 110 150  pict 0
`
	h, err := ParseText(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Rooms) != 1 || h.NRooms != 1 {
		t.Fatalf("got %d rooms (nRooms=%d), want 1", len(h.Rooms), h.NRooms)
	}
	r := &h.Rooms[0]
	if got := r.Name.Text(); got != "Start" {
		t.Errorf("room name %q", got)
	}
	if r.NumObjects != 2 {
		t.Errorf("numObjects derived as %d, want 2", r.NumObjects)
	}
	if r.LiveObjects() != 2 {
		t.Errorf("%d live objects, want 2", r.LiveObjects())
	}
	if r.Compacted() {
		t.Error("slots 0 and 4 are used; Compacted should be false")
	}
	if b := r.Objects[0].Blower(); b.TopLeft != (Point{V: 200, H: 64}) || b.Distance != 5 {
		t.Errorf("blower decoded as %+v", b)
	}
	if f := r.Objects[4].Furniture(); f.Bounds != (Rect{100, 50, 110, 150}) {
		t.Errorf("furniture decoded as %+v", f)
	}
	for _, slot := range []int{1, 2, 3, 5, 23} {
		if !r.Objects[slot].IsEmpty() {
			t.Errorf("slot %d should be empty, got %s", slot, r.Objects[slot])
		}
	}
	if h.Initial != (Point{V: 100, H: 200}) {
		t.Errorf("initial %+v: coordinates are vertical first", h.Initial)
	}
	// A minimal house must survive a trip through binary too.
	if _, err := h.Save(); err != nil {
		t.Fatal(err)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"no format", "house\n  version 0x200\n", "expected a `format 1` directive first"},
		{"bad format version", "format 99\n", "text format version 99"},
		{"unknown section field", "format 1\nhouse\n  nonsense 1\n", `unknown house field "nonsense"`},
		{"field outside section", "format 1\nversion 1\n", `"version" outside any section`},
		{"rooms out of order", "format 1\nroom 1 \"x\"\n", "room 1 out of order"},
		{"room skipped", "format 1\nroom 0 \"a\"\nroom 2 \"c\"\n", "room 2 out of order"},
		{"slot out of range", "format 1\nroom 0 \"a\"\n object 24 kStar at 1 2 length 0 points 0 state 0 initial 0\n", "out of range"},
		{"slots descending", "format 1\nroom 0 \"a\"\n" +
			" object 5 kStar at 1 2 length 0 points 0 state 0 initial 0\n" +
			" object 2 kStar at 1 2 length 0 points 0 state 0 initial 0\n", "out of order"},
		{"unknown object type", "format 1\nroom 0 \"a\"\n object 0 kNoSuchThing at 1 2\n", "unknown object type"},
		{"missing object field", "format 1\nroom 0 \"a\"\n object 0 kFloorVent at 1 2\n", `missing field "distance"`},
		{"extra object field", "format 1\nroom 0 \"a\"\n" +
			" object 0 kTable rect 1 2 3 4 pict 0 nonsense 1\n", `unknown field "nonsense"`},
		{"wrong arity", "format 1\nroom 0 \"a\"\n object 0 kTable rect 1 2 3 pict 0\n", `takes 4 value(s)`},
		{"defined code as number", "format 1\nroom 0 \"a\"\n object 0 0x01 raw 00 00 00 00 00 00 00 00 00 00\n", "write it as kFloorVent"},
		{"tiles arity", "format 1\nroom 0 \"a\"\n tiles 1 2 3\n", "needs exactly 8"},
		{"unterminated string", "format 1\nhouse\n banner \"oops\n", "unterminated quoted string"},
		{"bad escape", `format 1` + "\nhouse\n banner \"a\\qb\"\n", "unknown escape"},
		{"short doesn't fit", "format 1\nhouse\n version 70000\n", "does not fit a short"},
		{"byte doesn't fit", "format 1\nhouse\n hasgame 300\n", "does not fit a byte"},
		{"bad slack length", "format 1\nhouse\n slack 00\n", "expected 2 hex bytes"},
		{"non-hex residue", "format 1\nhouse\n slack zz zz\n", "not a hex byte"},
		{"duplicate format", "format 1\nformat 1\n", "duplicate format"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseText(strings.NewReader(tc.src))
			if err == nil {
				t.Fatalf("expected an error containing %q, got none", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err, tc.want)
			}
		})
	}
}

// TestUndefinedWhatSurvives covers the one form of object the shipped houses
// never contain: a `what` code outside all nine ranges. A user house may hold
// one, and neither codec may drop or reinterpret it.
func TestUndefinedWhatSurvives(t *testing.T) {
	const src = "format 1\nroom 0 \"a\"\n" +
		"    object 0 0x50 raw 01 02 03 04 05 06 07 08 09 0a   # undefined\n"
	h, err := ParseText(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	o := h.Rooms[0].Objects[0]
	if o.What != 0x50 {
		t.Fatalf("what = 0x%02x, want 0x50", uint16(o.What))
	}
	if o.Group() != GroupNone {
		t.Errorf("0x50 classified as %s, want none", o.Group())
	}
	if o.Data != [10]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		t.Errorf("union bytes %v", o.Data)
	}
	if got := o.String(); !strings.Contains(got, "undefined") {
		t.Errorf("String() = %q, want it to say undefined", got)
	}

	txt, err := h.Text(TextOptions{NoHeader: true})
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseText(strings.NewReader(txt))
	if err != nil {
		t.Fatalf("re-parsing our own output: %v\n%s", err, txt)
	}
	if back.Rooms[0].Objects[0] != o {
		t.Errorf("undefined object did not survive: %v", back.Rooms[0].Objects[0])
	}
}

// TestEmptySlotResidueSurvives covers the other unusual form: an empty slot that
// still holds bytes. 66,236 of the corpus's 66,240 empty slots are like this.
func TestEmptySlotResidueSurvives(t *testing.T) {
	const src = "format 1\nroom 0 \"a\"\n" +
		"    object 3 empty  raw ff ee dd cc bb aa 99 88 77 66\n"
	h, err := ParseText(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	o := h.Rooms[0].Objects[3]
	if !o.IsEmpty() {
		t.Fatal("slot 3 should be empty")
	}
	if o.Data[0] != 0xFF || o.Data[9] != 0x66 {
		t.Errorf("residue %v", o.Data)
	}
	if h.Rooms[0].NumObjects != 0 {
		t.Errorf("numObjects %d: an empty slot with residue is still empty",
			h.Rooms[0].NumObjects)
	}
	// Without Residue the writer drops it; with Residue it keeps it.
	plain, _ := h.Text(TextOptions{NoHeader: true})
	if strings.Contains(plain, "empty") {
		t.Errorf("default output should omit empty slots:\n%s", plain)
	}
	exact, _ := h.Text(TextOptions{NoHeader: true, Residue: true})
	back, err := ParseText(strings.NewReader(exact))
	if err != nil {
		t.Fatal(err)
	}
	if back.Rooms[0].Objects[3] != o {
		t.Errorf("residue lost: %v", back.Rooms[0].Objects[3])
	}
}

// TestNumObjectsOverride covers the malformed-count case the corpus does not
// contain: a stored numObjects that disagrees with the live slots. The original
// would silently recompute it; the text format records it so a round trip is
// still exact, and says why in a comment.
func TestNumObjectsOverride(t *testing.T) {
	h := &House{Version: HouseVersion, Rooms: []Room{{}}}
	for i := range h.Rooms[0].Objects {
		h.Rooms[0].Objects[i].What = ObjectIsEmpty
	}
	h.Rooms[0].Name.SetText("a")
	h.Rooms[0].NumObjects = 7 // a lie: no live slots

	txt, err := h.Text(TextOptions{NoHeader: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(txt, "numobjects") {
		t.Fatalf("expected a numobjects override line:\n%s", txt)
	}
	back, err := ParseText(strings.NewReader(txt))
	if err != nil {
		t.Fatal(err)
	}
	if back.Rooms[0].NumObjects != 7 {
		t.Errorf("numObjects %d, want the preserved 7", back.Rooms[0].NumObjects)
	}
}

func TestTokenize(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"# just a comment", nil},
		{"a b c", []string{"a", "b", "c"}},
		{"a  b\tc\r", []string{"a", "b", "c"}},
		{"a # b c", []string{"a"}},
		{"a#b", []string{"a"}},
		{`banner "hello world"`, []string{"banner", `"hello world"`}},
		{`banner "with # hash"`, []string{"banner", `"with # hash"`}},
		{`banner "esc \" quote" x`, []string{"banner", `"esc \" quote"`, "x"}},
		{`x "a" # c`, []string{"x", `"a"`}},
	}
	for _, tc := range cases {
		got, err := tokenize(tc.in)
		if err != nil {
			t.Errorf("tokenize(%q): %v", tc.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("tokenize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if _, err := tokenize(`x "unterminated`); err == nil {
		t.Error("expected an error for an unterminated string")
	}
}
