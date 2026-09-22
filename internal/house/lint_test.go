package house

// Linter tests.
//
// Two halves, and they test different things on purpose.
//
// The synthetic half builds the smallest house Lint has nothing to say about and
// then breaks exactly one thing in it. That is the only way to exercise most of the
// catalogue: the 22 shipped houses have been through the 1994 editor, which repairs
// nine of the twelve checks in CheckHouseForProblems on every save, so a corpus-only
// test would leave two thirds of these checks unrun and unproven -- a linter whose
// checks have never fired is a linter nobody should trust.
//
// The corpus half pins the calibration. Severity here was *chosen* from what the
// shipped houses contain, so the totals are part of the design and not an incidental
// measurement: if a change to the link arithmetic turned 189 dangling links into 190,
// the arithmetic is wrong, not the number.

import (
	"fmt"
	"strings"
	"testing"
)

// ------------------------------------------------------------------- builders

// testHouse assembles a house that Lint reports nothing about, given rooms that
// are themselves clean. Every test starts here and introduces one defect, so a
// finding count is unambiguous.
func testHouse(rooms ...Room) *House {
	return &House{
		Version:   HouseVersion,
		FirstRoom: 0,
		NRooms:    int16(len(rooms)),
		Rooms:     rooms,
	}
}

// testRoom fills the 24 slots with empties, copies the given objects into the low
// ones and sets numObjects to agree -- the three things a room needs before any
// check about its contents means anything.
func testRoom(name string, floor, suite int16, objs ...Object) Room {
	r := Room{Floor: floor, Suite: suite, NumObjects: int16(len(objs))}
	r.Name.SetText(name)
	for i := range r.Objects {
		r.Objects[i] = Object{What: ObjectIsEmpty}
	}
	copy(r.Objects[:], objs)
	return r
}

// star is the object every house needs one of to be winnable, and so the object
// every test house carries to keep no-stars out of the way.
func star() Object { return Object{What: codeStar} }

func plain(name string) Object { return Object{What: mustCode(name)} }

// bgRoom is testRoom with a background, and with the identity tiling rather than
// testRoom's all-zeroes -- tiles[] of 0 0 0 0 0 0 0 0 is a legal permutation that
// draws one column eight times, so leaving it would trip starfield-tiles in every
// room the mounting tests build and make their assertions mean something else.
func bgRoom(name string, bg, floor, suite int16, objs ...Object) Room {
	r := testRoom(name, floor, suite, objs...)
	r.Background = bg
	for i := range r.Tiles {
		r.Tiles[i] = int16(i)
	}
	return r
}

// transportTo and switchTo write a link through the member the original writes it
// through, rather than poking the aliased bytes directly -- the point of the
// LinkWhere/LinkWho distinction is that the two families are addressed separately
// even though they overlap, and a test that bypassed that would not be testing it.
func transportTo(name string, where int16, who byte) Object {
	o := Object{What: mustCode(name)}
	o.SetTransport(Transport{Where: where, Who: who})
	return o
}

func switchTo(name string, where int16, who byte) Object {
	o := Object{What: mustCode(name)}
	o.SetSwitch(Switch{Where: where, Who: who})
	return o
}

// fullOptions supplies both predicates so that no check is skipped. The art tree
// it pretends to be holds one 640-pixel picture, PICT 1000, which is ten tile
// columns; everything else is absent except the eighteen built-in backgrounds,
// which every real asset tree has and which are all 512x322 -- measured, not
// assumed, from assets/extracted/art/bg. Without them a room with a built-in
// background would raise background-pict and no test about such a room could
// assert "exactly one finding".
func fullOptions() LintOptions {
	return LintOptions{
		PictSize: func(id int16) (int, int, bool) {
			if id == 1000 {
				return 640, 480, true
			}
			if id >= FirstBuiltInBackground && id <= LastBuiltInBackground {
				return 512, 322, true
			}
			return 0, 0, false
		},
		SoundStatus: func(id int16) SoundStatus {
			switch id {
			case 3000:
				return SoundOK
			case 3001:
				return SoundUnreadable
			}
			return SoundMissing
		},
	}
}

// ------------------------------------------------------------------- assertions

// checkIDs is the multiset of check ids a run produced, as a sorted, counted
// string -- the shape an assertion can be written against without depending on
// wording that is meant to change as the C is better understood.
func checkIDs(fs []Finding) string {
	counts := map[string]int{}
	for _, f := range fs {
		counts[fmt.Sprintf("%s/%s", f.Severity, f.Check)]++
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sortStrings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s=%d", k, counts[k])
	}
	return strings.Join(parts, " ")
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// ------------------------------------------------------------------- the baseline

// TestLintCleanHouse is the test the rest depend on. If the baseline is not silent
// then every "exactly one finding" assertion below is really asserting something
// else, so this runs first and says so plainly when it breaks.
func TestLintCleanHouse(t *testing.T) {
	h := testHouse(testRoom("Front Door", 0, 0, star()))
	if got := h.Lint(fullOptions()); len(got) != 0 {
		t.Fatalf("the baseline house is not clean, so no other test in this file means "+
			"what it says:\n%s", findingLines(got))
	}
}

func findingLines(fs []Finding) string {
	var b strings.Builder
	for _, f := range fs {
		fmt.Fprintf(&b, "    %s\n", f)
	}
	return b.String()
}

// ------------------------------------------------------------------- house level

func TestLintHouseChecks(t *testing.T) {
	tests := []struct {
		name string
		want string
		fix  func(h *House)
	}{
		{
			name: "room count disagrees with the file",
			want: "error/room-count=1",
			fix:  func(h *House) { h.NRooms = 7 },
		},
		{
			// no-rooms stops the house checks: nothing below it can be true of a
			// room that is not there, so no-stars must *not* also fire. room-count
			// does, because it is tested before -- nRooms still says 1.
			name: "no rooms at all",
			want: "error/no-rooms=1 error/room-count=1",
			fix:  func(h *House) { h.Rooms = nil },
		},
		{
			name: "first room out of range",
			want: "error/first-room=1",
			fix:  func(h *House) { h.FirstRoom = 9 },
		},
		{
			name: "first room is a deleted room",
			want: "error/first-room=1 note/deleted-room=1 warn/no-stars=1",
			fix: func(h *House) {
				h.Rooms[0].Suite = RoomIsEmpty
			},
		},
		{
			name: "pre-2.0 version",
			want: "note/house-version=1",
			fix:  func(h *House) { h.Version = 0x0100 },
		},
		{
			name: "no stars",
			want: "warn/no-stars=1",
			fix:  func(h *House) { h.Rooms[0].Objects[0] = Object{What: ObjectIsEmpty} },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := testHouse(testRoom("Front Door", 0, 0, star()))
			// numObjects is derived from the slots, so a fix that removes an object
			// must not be reported as a numObjects mismatch as well.
			tc.fix(h)
			for i := range h.Rooms {
				h.Rooms[i].NumObjects = int16(h.Rooms[i].LiveObjects())
			}
			if got := checkIDs(h.Lint(fullOptions())); got != tc.want {
				t.Errorf("got %q, want %q\n%s", got, tc.want, findingLines(h.Lint(fullOptions())))
			}
		})
	}
}

// TestLintDuplicateCell covers the one house-level check that needs two rooms: two
// rooms at one floor/suite, where RoomNumber's linear search can only ever find the
// first, so the second is addressable by nothing at all.
func TestLintDuplicateCell(t *testing.T) {
	h := testHouse(
		testRoom("Front Door", 0, 0, star()),
		testRoom("The Same Place", 0, 0),
	)
	got := h.Lint(fullOptions())
	if want := "error/duplicate-cell=1"; checkIDs(got) != want {
		t.Fatalf("got %q, want %q\n%s", checkIDs(got), want, findingLines(got))
	}
	if got[0].Room != 1 {
		t.Errorf("the finding is on room %d; it belongs on the *later* room, room 1, "+
			"because that is the unreachable one", got[0].Room)
	}
}

// TestLintSkippedChecks pins the promise that a nil predicate means skip and never
// "answer no". A report with no asset tree must say which checks did not run, or a
// clean report is indistinguishable from an incomplete one.
func TestLintSkippedChecks(t *testing.T) {
	// The house is deliberately full of things the asset checks would catch: a
	// background that no tree holds, a kCustomPict naming nothing, a sound trigger
	// naming a sound nobody has, and -- in a second room, because a column can only
	// be judged against a background that *does* resolve -- a tile column past the
	// end of one that does.
	first := testRoom("Front Door", 0, 0, star(),
		plain("kCustomPict"), switchTo("kSoundTrigger", 7777, UnlinkedWho))
	first.Background = 4242
	second := testRoom("Back Door", 0, 1)
	second.Background = 1000
	second.Tiles[3] = 99
	h := testHouse(first, second)

	got := h.Lint(LintOptions{})
	if want := "note/checks-skipped=1"; checkIDs(got) != want {
		t.Fatalf("with no predicates, got %q, want %q\n%s",
			checkIDs(got), want, findingLines(got))
	}
	for _, id := range []string{"sound-id", "sound-unreadable", "background-pict",
		"tile-column", "custom-pict"} {
		if !strings.Contains(got[0].Message, id) {
			t.Errorf("the checks-skipped note does not name %q, so a reader cannot tell "+
				"what the report is missing: %s", id, got[0].Message)
		}
	}

	// And with the predicates supplied, every one of them fires and the note does not.
	want := "warn/background-pict=1 warn/custom-pict=1 warn/sound-id=1 warn/tile-column=1"
	if full := h.Lint(fullOptions()); checkIDs(full) != want {
		t.Errorf("with predicates, got %q, want %q\n%s",
			checkIDs(full), want, findingLines(full))
	}
}

// ------------------------------------------------------------------- room level

func TestLintRoomChecks(t *testing.T) {
	tests := []struct {
		name string
		want string
		fix  func(r *Room)
	}{
		{
			// A deleted room returns early: nothing about its contents is checked,
			// because CompressHouse is about to remove it -- which is also why the
			// star inside it stops counting. first-room fires because the house
			// starts in this room and GetFirstRoomNumber does not notice.
			name: "deleted room",
			want: "error/first-room=1 note/deleted-room=1 warn/no-stars=1",
			fix:  func(r *Room) { r.Suite = RoomIsEmpty },
		},
		{
			name: "floor below the grid",
			want: "error/grid-range=1",
			fix:  func(r *Room) { r.Floor = MinFloor - 1 },
		},
		{
			name: "floor above the grid",
			want: "error/grid-range=1",
			fix:  func(r *Room) { r.Floor = MaxFloor + 1 },
		},
		{
			name: "suite above the grid",
			want: "error/grid-range=1",
			fix:  func(r *Room) { r.Suite = MaxSuite + 1 },
		},
		{
			// ValidateRoomNumbers tests floor and suite sequentially rather than
			// exclusively, so a room wrong in both ways is wrong twice.
			name: "both coordinates off the grid",
			want: "error/grid-range=2",
			fix:  func(r *Room) { r.Floor, r.Suite = MaxFloor+1, MaxSuite+1 },
		},
		{
			name: "name length past Str27",
			want: "error/name-length=1",
			fix:  func(r *Room) { r.Name[0] = MaxRoomNameLen + 1 },
		},
		{
			name: "untitled room",
			want: "note/untitled-room=1",
			fix:  func(r *Room) { r.Name.SetText(UntitledRoom) },
		},
		{
			name: "untitled room in another case",
			want: "note/untitled-room=1",
			fix:  func(r *Room) { r.Name.SetText("UNTITLED ROOM") },
		},
		{
			name: "numObjects disagrees",
			want: "note/num-objects=1",
			fix:  func(r *Room) { r.NumObjects = 17 },
		},
		{
			name: "undefined what",
			want: "error/undefined-what=1",
			fix:  func(r *Room) { r.Objects[1] = Object{What: 0x20}; r.NumObjects = 2 },
		},
		{
			name: "flower variant out of range",
			want: "warn/flower-variant=1",
			fix: func(r *Room) {
				f := plain("kFlower")
				f.SetClutter(Clutter{Pict: FlowerVariants})
				r.Objects[1] = f
				r.NumObjects = 2
			},
		},
		{
			name: "two sound triggers in one room",
			want: "warn/sound-trigger-crowded=1",
			fix: func(r *Room) {
				r.Objects[1] = switchTo("kSoundTrigger", 3000, UnlinkedWho)
				r.Objects[2] = switchTo("kSoundTrigger", 3000, UnlinkedWho)
				r.NumObjects = 3
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			room := testRoom("Front Door", 0, 0, star())
			tc.fix(&room)
			h := testHouse(room)
			if got := checkIDs(h.Lint(fullOptions())); got != tc.want {
				t.Errorf("got %q, want %q\n%s",
					got, tc.want, findingLines(h.Lint(fullOptions())))
			}
		})
	}
}

// TestLintBackground covers the two checks that read a picture's width: a background
// nobody has, and a tile column past the end of one somebody does.
func TestLintBackground(t *testing.T) {
	// PICT 1000 is 640 pixels, so columns 0..9 are in range and 10 is not.
	room := testRoom("Front Door", 0, 0, star())
	room.Background = 1000
	room.Tiles[0] = 9
	room.Tiles[1] = 10
	room.Tiles[2] = -1
	h := testHouse(room)
	got := h.Lint(fullOptions())
	if want := "warn/tile-column=2"; checkIDs(got) != want {
		t.Fatalf("got %q, want %q\n%s", checkIDs(got), want, findingLines(got))
	}

	// A background of 0 means "no background" and must not be looked up at all,
	// which is also what keeps the baseline house silent. The bad columns stay put:
	// the assertion is that they are not even read.
	h.Rooms[0].Background = 0
	if got := h.Lint(fullOptions()); len(got) != 0 {
		t.Errorf("background 0 should be left alone, got:\n%s", findingLines(got))
	}
}

// TestLintMounting is the first check in this file that relates an object to the room
// it stands in rather than to the room's own fields.
//
// It is also the first that can only be justified by the corpus, because nothing goes
// wrong mechanically: GetObjectRect places these objects at absolute coordinates, the
// lift column is computed from the object's data, and the art composes. So the cases
// below are paired -- every "this is wrong" row has a "this is the same object in a
// room that has the surface" row beside it, because a check with no negative case is
// a check that might be firing on the object alone.
func TestLintMounting(t *testing.T) {
	tests := []struct {
		name  string
		bg    int16
		obj   string
		want  string
		about string
	}{
		// The floor rule. kSky, kStratosphere and kStars are DoesRoomHaveFloor's three.
		{"floor vent in the sky", 2015, "kFloorVent",
			"warn/mount-no-floor=1", "no floor at all"},
		{"sewer grate in the stratosphere", 2016, "kSewerGrate",
			"warn/mount-no-floor=1", "kStratosphere"},
		{"greco vent among the stars", 2017, "kGrecoVent",
			"warn/mount-no-floor=1", "kStars"},
		// A standing lamp is in the set for the same reason a vent is: kHipLamp is 276
		// pixels tall and its v is 23 in all 26 corpus placements, so its foot is on the
		// floor line and not its head on the ceiling.
		{"hip lamp in the sky", 2015, "kHipLamp",
			"warn/mount-no-floor=1", "rests on the floor"},
		{"deco lamp in the sky", 2015, "kDecoLamp",
			"warn/mount-no-floor=1", "rests on the floor"},

		// kRoof is the one that catches a reader out, and it is why the two lists are
		// separate maps rather than one predicate: a roof has a floor -- you walk on it --
		// and no ceiling. So a vent on a roof is odd but not floorless, and the linter
		// says nothing, which is also what the corpus does with its 181 kRoof rooms.
		{"floor vent on a roof", 2014, "kFloorVent", "", ""},
		{"floor vent in a simple room", 2000, "kFloorVent", "", ""},
		{"sewer grate in a meadow", 2012, "kSewerGrate", "", ""},

		// The ceiling rule, over seven backgrounds rather than three.
		{"ceiling vent in the sky", 2015, "kCeilingVent",
			"warn/mount-no-ceiling=1", "no ceiling"},
		{"ceiling blower on a roof", 2014, "kCeilingBlower",
			"warn/mount-no-ceiling=1", "kRoof"},
		{"ceiling light in a garden", 2009, "kCeilingLight",
			"warn/mount-no-ceiling=1", "kGarden"},
		{"flourescent in a meadow", 2012, "kFlourescent",
			"warn/mount-no-ceiling=1", "kMeadow"},
		{"track light in a field", 2013, "kTrackLight",
			"warn/mount-no-ceiling=1", "kField"},
		{"ceiling light in a simple room", 2000, "kCeilingLight", "", ""},
		{"ceiling vent in a basement", 2001, "kCeilingVent", "", ""},

		// kSkywalk and kDirt look outdoor and are not: both are in neither list, which
		// is a fact about Room.c and not a guess, and a check derived from how a
		// background looks would get both wrong.
		{"ceiling light on a skywalk", 2010, "kCeilingLight", "", ""},
		{"floor vent in the dirt", 2011, "kFloorVent", "", ""},

		// A room with no floor *and* no ceiling holding one of each: two findings from
		// one room, because the two rules are independent and a room can fail both.
		{"both at once", 2015, "", "", ""},

		// The five flames are deliberately in neither set. 18 kTiki stand in corpus
		// rooms with no ceiling and 1 kStubby in a room with no floor, so a rule that
		// included them would fail the corpus on houses that meant it.
		{"tiki on a roof", 2014, "kTiki", "", ""},
		{"candle in the sky", 2015, "kCandle", "", ""},
		{"table lamp in the sky", 2015, "kTableLamp", "", ""},

		// User art is out of scope and silently so: its openings come from the room's
		// own bounds field or the background's 'bnds' resource, neither of which this
		// package can read. A false negative here is the documented limit
		// (docs/IMPROVEMENTS.md 4.22); a false positive would be a bug.
		//
		// background-pict is expected and is not the subject: fullOptions' stub tree
		// carries no PICT 3000, and a house that really carried its own art would not
		// raise it. It is asserted rather than suppressed because a row whose want is
		// "" would have to be read as "the mount checks are silent", and this one says
		// the stronger thing -- the only finding is the one about the missing picture.
		{"floor vent in user art", 3000, "kFloorVent", "warn/background-pict=1", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var objs []Object
			if tc.obj == "" {
				objs = []Object{star(), plain("kFloorVent"), plain("kCeilingVent")}
			} else {
				objs = []Object{star(), plain(tc.obj)}
			}
			h := testHouse(bgRoom("Up High", tc.bg, 0, 0, objs...))
			want := tc.want
			if tc.obj == "" {
				want = "warn/mount-no-ceiling=1 warn/mount-no-floor=1"
			}
			got := h.Lint(fullOptions())
			if checkIDs(got) != want {
				t.Fatalf("got %q, want %q\n%s", checkIDs(got), want, findingLines(got))
			}
			if tc.about != "" && !strings.Contains(got[0].Message, tc.about) {
				t.Errorf("message does not mention %q: %s", tc.about, got[0].Message)
			}
		})
	}
}

// TestLintStarfieldTiles pins the weakest of the three background checks, and the one
// whose severity moved while it was being written.
//
// Permuting a starfield's columns is harmless -- that is the whole finding -- so what
// the note has to do is fire on the permutation without claiming it is a defect. The
// wording matters more than usual here and is asserted, not just the count.
func TestLintStarfieldTiles(t *testing.T) {
	tests := []struct {
		name  string
		bg    int16
		tiles [NumTiles]int16
		want  string
		about string
	}{
		{"stars, identity", 2017, [NumTiles]int16{0, 1, 2, 3, 4, 5, 6, 7}, "", ""},
		{"stratosphere, identity", 2016, [NumTiles]int16{0, 1, 2, 3, 4, 5, 6, 7}, "", ""},
		// Leviathan room 221's actual tiling, which is the clearest of the corpus's
		// three exceptions that the author meant it.
		{"stars, reversed", 2017, [NumTiles]int16{7, 6, 5, 4, 3, 2, 1, 0},
			"note/starfield-tiles=1", "243 of the corpus's 246 kStars rooms"},
		// One column out of place is the same finding: the rule is the identity, not
		// "mostly the identity", because a leftover tiling rarely differs in one place.
		{"stars, one column swapped", 2017, [NumTiles]int16{0, 1, 2, 3, 4, 5, 7, 6},
			"note/starfield-tiles=1", "star field either way"},
		// Each background quotes its own tally. kStratosphere is 62 of 62 and kStars
		// 243 of 246, and a message that gave one house's numbers for the other would
		// be exactly the imprecise citation IMPROVEMENTS 4.21 is about.
		{"stratosphere, shuffled", 2016, [NumTiles]int16{1, 0, 2, 3, 4, 5, 6, 7},
			"note/starfield-tiles=1", "62 of the corpus's 62 kStratosphere rooms"},
		// All zeroes is a legal permutation -- it draws column 0 eight times -- and is
		// the shape a hand-written or generated room starts out in, which is the case
		// this check exists to catch.
		{"stars, all zeroes", 2017, [NumTiles]int16{},
			"note/starfield-tiles=1", "background changed after the tiles"},
		// kSky is not a starfield and is the reason the set has two members and not
		// three: its 595 corpus rooms use 98 distinct patterns and none is the identity,
		// because rearranging cloud columns is how one picture becomes 595 skies.
		{"sky, shuffled", 2015, [NumTiles]int16{5, 2, 7, 0, 1, 1, 3, 4}, "", ""},
		{"sky, all zeroes", 2015, [NumTiles]int16{}, "", ""},
		{"simple room, shuffled", 2000, [NumTiles]int16{5, 2, 7, 0, 1, 1, 3, 4}, "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			room := testRoom("Up High", 0, 0, star())
			room.Background = tc.bg
			room.Tiles = tc.tiles
			got := testHouse(room).Lint(fullOptions())
			if checkIDs(got) != tc.want {
				t.Fatalf("got %q, want %q\n%s", checkIDs(got), tc.want, findingLines(got))
			}
			if tc.about != "" && !strings.Contains(got[0].Message, tc.about) {
				t.Errorf("message does not mention %q: %s", tc.about, got[0].Message)
			}
		})
	}
}

// ------------------------------------------------------------------- sound

// TestLintSoundTrigger is the check that had to be split. A sound the house does not
// carry is the author's mistake and the original punished it identically; a sound the
// house carries that this port cannot decode is our gap, and reporting the two the
// same way would have the tool telling a reader something false about a 1994 house.
func TestLintSoundTrigger(t *testing.T) {
	tests := []struct {
		name  string
		id    int16
		want  string
		blame string // a word the message must contain, so the two do not drift
	}{
		{"loads", 3000, "", ""},
		{"carried but MACE-compressed", 3001, "warn/sound-unreadable=1", "gliderGo"},
		{"not carried at all", 9999, "warn/sound-id=1", "does not carry"},
		{"names no sound", UnlinkedWhere, "note/sound-id=1", "names no sound"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := testHouse(testRoom("Front Door", 0, 0, star(),
				switchTo("kSoundTrigger", tc.id, UnlinkedWho)))
			got := h.Lint(fullOptions())
			if checkIDs(got) != tc.want {
				t.Fatalf("got %q, want %q\n%s", checkIDs(got), tc.want, findingLines(got))
			}
			if tc.blame != "" && !strings.Contains(got[0].Message, tc.blame) {
				t.Errorf("the message does not say %q, so it does not say whose fault "+
					"this is: %s", tc.blame, got[0].Message)
			}
		})
	}
}

// TestLintSoundTriggerCarriesNoRoomLink pins the reason kSoundTrigger is excluded
// from linkSwitches: its `where` is a resource id, and resolving it as a room would
// wire the trigger to whatever room that number happened to name.
func TestLintSoundTriggerCarriesNoRoomLink(t *testing.T) {
	o := switchTo("kSoundTrigger", 3000, UnlinkedWho)
	if LinkCarryingSwitch(o.What) {
		t.Fatal("kSoundTrigger is in linkSwitches, so its sound id is being resolved " +
			"as a floor/suite pair")
	}
	if _, ok := LinkWhere(o); ok {
		t.Error("LinkWhere answers for a kSoundTrigger")
	}
	h := testHouse(testRoom("Front Door", 0, 0, star()))
	if got := h.RoomLinked(o); got != RoomIsEmpty {
		t.Errorf("RoomLinked(kSoundTrigger) = %d, want %d", got, RoomIsEmpty)
	}
}

// ------------------------------------------------------------------- links

// TestLintLinks walks the five ways a stored link fails, in the order they stop
// mattering. The destination room in each case is floor 1 suite 0, which packs to
// where = 0*100 + 1 + 8 = 9.
func TestLintLinks(t *testing.T) {
	const destWhere = 9 // floor 1, suite 0, version 2.0 packing

	tests := []struct {
		name  string
		obj   Object
		want  string
		about string // a phrase the message must carry
	}{
		{
			name: "unlinked the way DoUnlink writes it",
			obj:  transportTo("kFloorTrans", UnlinkedWhere, UnlinkedWho),
			want: "note/link-unlinked=1",
		},
		{
			name:  "no room but still names a slot",
			obj:   transportTo("kFloorTrans", UnlinkedWhere, 3),
			want:  "warn/link-unlinked=1",
			about: "misleading residue",
		},
		{
			name:  "room does not exist",
			obj:   transportTo("kFloorTrans", 4242, 0),
			want:  "warn/link-dangling-room=1",
			about: "silently does nothing",
		},
		{
			// The corpus' commonest finding, and a note rather than a warning
			// because it is inert twice over: this is what a deleted destination
			// leaves behind.
			name:  "room does not exist and who is 255",
			obj:   transportTo("kFloorTrans", 4242, UnlinkedWho),
			want:  "note/link-dangling-room=1",
			about: "already dead before the room went",
		},
		{
			name:  "room resolves but no target slot",
			obj:   transportTo("kFloorTrans", destWhere, UnlinkedWho),
			want:  "note/link-no-target=1",
			about: "inert even though the room resolves",
		},
		{
			name:  "target slot past the 24 a room holds",
			obj:   transportTo("kFloorTrans", destWhere, 35),
			want:  "error/link-slot-range=1",
			about: "no bound check",
		},
		{
			name:  "target slot is empty",
			obj:   transportTo("kFloorTrans", destWhere, 5),
			want:  "warn/link-slot-empty=1",
			about: "leaves forward links pointing at the hole",
		},
		{
			name:  "transport linked to something that is not a transport",
			obj:   transportTo("kFloorTrans", destWhere, 0),
			want:  "note/link-target-kind=1",
			about: "LinkedToOther",
		},
		{
			// A switch linked to a non-transport is the normal case -- that is what
			// a light switch is for -- so link-target-kind must not fire for one.
			name: "switch linked to an appliance is fine",
			obj:  switchTo("kLightSwitch", destWhere, 0),
			want: "",
		},
		{
			name: "transport linked to a transport is fine",
			obj:  transportTo("kFloorTrans", destWhere, 1),
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := testHouse(
				testRoom("Front Door", 0, 0, star(), tc.obj),
				// Slot 0 is a light, slot 1 a transport, slot 5 empty: one target of
				// each kind the checks distinguish between. The transport is a
				// kDoorInLf rather than a kCeilingTrans because a door is in the
				// transport group but stores no link of its own, so it cannot add a
				// finding about itself and confuse the count.
				testRoom("Upstairs", 1, 0, plain("kFlourescent"), plain("kDoorInLf")),
			)
			got := h.Lint(fullOptions())
			if checkIDs(got) != tc.want {
				t.Fatalf("got %q, want %q\n%s", checkIDs(got), tc.want, findingLines(got))
			}
			if tc.about != "" && !strings.Contains(got[0].Message, tc.about) {
				t.Errorf("message does not mention %q: %s", tc.about, got[0].Message)
			}
		})
	}
}

// TestLintLinkVersionDependence is the reason ExtractFloorSuite lives in this
// package rather than being written out twice. The same `where` names a different
// room in a pre-2.0 house, and a linter that assumed one packing would report every
// link in such a house as dangling.
func TestLintLinkVersionDependence(t *testing.T) {
	// where = 900: 2.0 unpacks floor 900%100-8 = -8, suite 9; 1.x unpacks
	// floor 900/100-8 = 1, suite 0.
	const where = 900

	h := testHouse(
		testRoom("Front Door", 0, 0, star(),
			transportTo("kFloorTrans", where, 0)),
		testRoom("Upstairs", 1, 0, plain("kDoorInLf")),
	)

	// As a 2.0 house the link dangles.
	if got, want := checkIDs(h.Lint(fullOptions())), "warn/link-dangling-room=1"; got != want {
		t.Errorf("version 0x%04X: got %q, want %q", uint16(h.Version), got, want)
	}
	// As a 1.x house it resolves, to a transport, so nothing is reported at all.
	h.Version = 0x0100
	if got, want := checkIDs(h.Lint(fullOptions())), "note/house-version=1"; got != want {
		t.Errorf("version 0x%04X: got %q, want %q\n%s", uint16(h.Version), got, want,
			findingLines(h.Lint(fullOptions())))
	}
}

// TestMergeFloorSuiteRoundTrips pins the bijection the original does not have. The
// C's MergeFloorSuite omits the +kNumUndergroundFloors bias and leaves three call
// sites to add it three different ways; this pair owns it, so the two are inverses.
//
// The two packings do not cover the same grid, and that is a property of the format
// rather than of this code. Both put one field in the low two decimal digits, which
// can only hold 0..99. Version 2.0 puts the *floor* there -- floor+8 runs 1..64, so
// every legal floor fits and suites run to 127 in the high digits. Version 1.x puts
// the *suite* there, so a pre-2.0 house cannot express suite 100 or above at all:
// MergeFloorSuite(-7, 100) and MergeFloorSuite(-6, 0) are the same short. That is
// very likely why 2.0 swapped them, and it is why the pre-2.0 loop below stops at 99
// -- testing past it would be asserting that a 1994 encoding was wider than it was.
func TestMergeFloorSuiteRoundTrips(t *testing.T) {
	for _, tc := range []struct {
		version  int16
		maxSuite int16
	}{
		{0x0100, 99},
		{HouseVersion, MaxSuite},
	} {
		h := &House{Version: tc.version}
		for floor := MinFloor; floor <= MaxFloor; floor++ {
			for suite := MinSuite; suite <= tc.maxSuite; suite++ {
				combo := h.MergeFloorSuite(floor, suite)
				gotF, gotS := h.ExtractFloorSuite(combo)
				if gotF != floor || gotS != suite {
					t.Fatalf("version 0x%04X: (%d,%d) -> %d -> (%d,%d)",
						uint16(tc.version), floor, suite, combo, gotF, gotS)
				}
			}
		}
	}

	// And the collision itself, pinned so that nobody later "fixes" the pre-2.0
	// branch into something the original would not have written.
	old := &House{Version: 0x0100}
	if a, b := old.MergeFloorSuite(-7, 100), old.MergeFloorSuite(-6, 0); a != b {
		t.Errorf("pre-2.0 packing of (-7,100) is %d and of (-6,0) is %d; they are the "+
			"same short in the original's encoding", a, b)
	}
}

// TestDeletedDestinationSentinel pins the one value where RoomNumber's refusal to
// skip deleted rooms is load-bearing. A link whose destination was deleted holds
// where == -100, which decodes to floor -8 suite -1 -- so it resolves to a real
// index if and only if the house happens to contain a deleted room on floor -8, and
// the linter has to agree with the game about which of those two things happens.
func TestDeletedDestinationSentinel(t *testing.T) {
	h := testHouse(testRoom("Front Door", 0, 0, star()))
	const deleted = -100
	if got, want := h.MergeFloorSuite(-NumUndergroundFloors, RoomIsEmpty), int16(deleted); got != want {
		t.Fatalf("MergeFloorSuite(-8, -1) = %d, want %d -- the sentinel is not what "+
			"the editor writes", got, want)
	}
	floor, suite := h.ExtractFloorSuite(deleted)
	if floor != -NumUndergroundFloors || suite != RoomIsEmpty {
		t.Fatalf("ExtractFloorSuite(-100) = (%d,%d), want (-8,-1)", floor, suite)
	}
	if got := h.RoomNumber(floor, suite); got != RoomIsEmpty {
		t.Errorf("RoomNumber(-8,-1) = %d in a house with no deleted rooms, want %d",
			got, RoomIsEmpty)
	}

	// Add a deleted room on floor -8 and the sentinel now names it, exactly as
	// GetRoomNumber would. This is not a defect to fix; it is the behaviour to
	// reproduce.
	dead := testRoom("Gone", -NumUndergroundFloors, RoomIsEmpty)
	h.Rooms = append(h.Rooms, dead)
	if got := h.RoomNumber(floor, suite); got != 1 {
		t.Errorf("RoomNumber(-8,-1) = %d with a deleted room on floor -8, want 1 -- "+
			"GetRoomNumber compares both fields and nothing else", got)
	}
}

// ------------------------------------------------------------------- staircases

// TestLintStairs covers the pairing, including the crossing that is easy to get
// backwards: a kUpStairs needs a kDownStairs in the room *above* to arrive behind,
// because the two are the two ends of one flight.
func TestLintStairs(t *testing.T) {
	tests := []struct {
		name  string
		rooms []Room
		want  string
		about string
	}{
		{
			name: "paired both ways",
			rooms: []Room{
				testRoom("Ground", 0, 0, star(), plain("kUpStairs")),
				testRoom("First", 1, 0, plain("kDownStairs")),
			},
			want: "",
		},
		{
			name: "up stairs to a floor with no room",
			rooms: []Room{
				testRoom("Ground", 0, 0, star(), plain("kUpStairs")),
			},
			want:  "warn/stairs-no-room=1",
			about: "no room there",
		},
		{
			name: "down stairs to a floor with no room",
			rooms: []Room{
				testRoom("Ground", 0, 0, star(), plain("kDownStairs")),
			},
			want:  "warn/stairs-no-room=1",
			about: "no room there",
		},
		{
			// The crossing. A kDownStairs in the room above is not a counterpart
			// for a kUpStairs below -- it is the *same* flight seen from the top,
			// and what the room above needs is one of those, not another kUpStairs.
			// The third room exists only so that the middle room's own kUpStairs is
			// properly paired and does not add a second finding.
			name: "up stairs whose destination holds another up stairs",
			rooms: []Room{
				testRoom("Ground", 0, 0, star(), plain("kUpStairs")),
				testRoom("First", 1, 0, plain("kUpStairs")),
				testRoom("Second", 2, 0, plain("kDownStairs")),
			},
			want:  "warn/stairs-unpaired=1",
			about: "walks up out of the right-hand wall",
		},
		{
			name: "down stairs whose destination holds another down stairs",
			rooms: []Room{
				testRoom("Ground", 0, 0, star(), plain("kDownStairs")),
				testRoom("Basement", -1, 0, plain("kDownStairs")),
				testRoom("Sub-basement", -2, 0, plain("kUpStairs")),
			},
			want:  "warn/stairs-unpaired=1",
			about: "off the left edge",
		},
		{
			// A staircase only pairs within its own suite: the destination is
			// (floor+1, suite), not any room on the floor above. The kDownStairs
			// next door is correctly paired with its own suite's room below, so the
			// only finding is that it is no help to the Ground floor's stairs.
			name: "up stairs paired in the wrong suite",
			rooms: []Room{
				testRoom("Ground", 0, 0, star(), plain("kUpStairs")),
				testRoom("Next Door", 1, 1, plain("kDownStairs")),
				testRoom("Below Next Door", 0, 1, plain("kUpStairs")),
			},
			want:  "warn/stairs-no-room=1",
			about: "floor 1 suite 0",
		},
		{
			name: "two up stairs in one room",
			rooms: []Room{
				testRoom("Ground", 0, 0, star(),
					plain("kUpStairs"), plain("kUpStairs")),
				testRoom("First", 1, 0, plain("kDownStairs")),
			},
			want:  "note/stairs-doubled=1",
			about: "the lower-numbered slot decides",
		},
		{
			name: "two down stairs in one room",
			rooms: []Room{
				testRoom("Ground", 0, 0, star(),
					plain("kDownStairs"), plain("kDownStairs")),
				testRoom("Basement", -1, 0, plain("kUpStairs")),
			},
			want:  "note/stairs-doubled=1",
			about: "the lower-numbered slot decides",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := testHouse(tc.rooms...)
			got := h.Lint(fullOptions())
			if checkIDs(got) != tc.want {
				t.Fatalf("got %q, want %q\n%s", checkIDs(got), tc.want, findingLines(got))
			}
			if tc.about != "" && !strings.Contains(got[0].Message, tc.about) {
				t.Errorf("message does not mention %q: %s", tc.about, got[0].Message)
			}
		})
	}
}

// ------------------------------------------------------------------- summarising

func TestCountsAndAtLeast(t *testing.T) {
	fs := []Finding{
		{Severity: SeverityNote, Check: "a"},
		{Severity: SeverityWarn, Check: "b"},
		{Severity: SeverityWarn, Check: "c"},
		{Severity: SeverityError, Check: "d"},
	}
	notes, warns, errs := Counts(fs)
	if notes != 1 || warns != 2 || errs != 1 {
		t.Errorf("Counts = %d,%d,%d, want 1,2,1", notes, warns, errs)
	}
	if got := AtLeast(fs, SeverityWarn); len(got) != 3 {
		t.Errorf("AtLeast(warn) kept %d findings, want 3", len(got))
	}
	// Order is preserved, because two reports are meant to be diffable.
	if got := AtLeast(fs, SeverityWarn); got[0].Check != "b" || got[2].Check != "d" {
		t.Errorf("AtLeast reordered: %v", got)
	}
}

func TestParseSeverity(t *testing.T) {
	for _, s := range []string{"note", "WARN", "Error"} {
		if _, ok := ParseSeverity(s); !ok {
			t.Errorf("ParseSeverity(%q) failed; the flag is documented as case-insensitive", s)
		}
	}
	if _, ok := ParseSeverity("fatal"); ok {
		t.Error("ParseSeverity accepted \"fatal\"")
	}
	for i, want := range []string{"note", "warn", "error"} {
		if got := Severity(i).String(); got != want {
			t.Errorf("Severity(%d) = %q, want %q", i, got, want)
		}
	}
}

// ------------------------------------------------------------------- the corpus

// TestLintCorpus pins the calibration.
//
// These totals are not an incidental measurement -- every severity in the linter was
// chosen so that the shipped houses produce exactly this and no errors beyond the one
// the analysis documents. Three of the numbers cross-check the link arithmetic
// against figures derived independently by tools/probe_house.py and recorded in
// docs/analysis/: 189 dangling links, one out-of-range `who`, 69 stars in 21 houses.
// If a change to ExtractFloorSuite turned 189 into 190, the change is wrong.
//
// It runs without asset predicates, so the counts are a property of the house files
// alone and cannot move when the art or sound extraction changes.
func TestLintCorpus(t *testing.T) {
	corpus := loadCorpus(t)

	counts := map[string]int{}
	var errors []string
	for _, e := range corpus {
		for _, f := range e.house.Lint(LintOptions{}) {
			if f.Check == "checks-skipped" {
				continue
			}
			counts[f.Check]++
			counts[f.Severity.String()]++
			if f.Severity == SeverityError {
				errors = append(errors, fmt.Sprintf("%s: %s", e.stem, f))
			}
		}
	}

	// The one error in 4,070 rooms: CD Demo House room 72 slot 22 links to slot 35
	// of a room that holds 24. Any second error is a regression in the linter or a
	// change in the corpus, and either way this test should say which.
	if len(errors) != 1 {
		t.Errorf("the corpus produces %d errors, want exactly 1:\n    %s",
			len(errors), strings.Join(errors, "\n    "))
	}

	for _, tc := range []struct {
		check string
		want  int
		why   string
	}{
		{"link-dangling-room", 189,
			"docs/analysis/original-houses.md counts 189 dangling links across the 22 " +
				"houses; this is the same number reached by resolving each link through " +
				"ExtractFloorSuite and RoomNumber"},
		{"link-slot-range", 1,
			"exactly one link in the corpus has who >= 24: CD Demo House room 72 slot 22, " +
				"who = 35 (docs/analysis/house-format.md 12.x)"},
		{"no-stars", 1,
			"Fun House is the only shipped house with no kStar, which is why no-stars is " +
				"a warning and not an error"},
		// The three stair rules, pinned at zero on purpose. This file's own header cites
		// "staircases that lead nowhere" among the four corpus defects that justify the
		// severity calibration, and across 326 stair objects in 4,070 rooms not one of
		// these fires: every flight in the corpus is paired and both ends land in a room
		// that exists. The rules are exercised by TestLintStairs on houses written to
		// provoke them, so a zero here is a fact about the corpus and not a dead check.
		// See docs/IMPROVEMENTS.md 4.21, which is what these three rows are the evidence
		// for -- the claim went unexamined because nothing counted it.
		{"stairs-no-room", 0,
			"no shipped staircase leads to a floor/suite with no room in it"},
		{"stairs-unpaired", 0,
			"every shipped kUpStairs has a kDownStairs to arrive on and vice versa; the " +
				"pairing is crossed, which is the C verbatim (HouseLegal.c:961-1045)"},
		{"stairs-doubled", 0,
			"no shipped room holds two kUpStairs or two kDownStairs, so GetUpStairsRightEdge " +
				"and GetDownStairsLeftEdge breaking on the first match is never observable"},
		// The three background checks, and these rows are the reason they exist. The two
		// mount rules are calibrated at warn on being zero here: a warning that fired on
		// the originals would be the linter telling a reader something false about a 1994
		// house, and IMPROVEMENTS 4.22 proposed warn on exactly this measurement.
		//
		// The zeroes cover real placements rather than an empty search -- 903 kFloorVent,
		// 336 kSewerGrate, 105 kSewerBlower, 70 kFloorBlower, 40 kDecoLamp, 25 kGrecoVent
		// and 23 kHipLamp in built-in-background rooms for the floor rule, and 112
		// kCeilingLight, 79 kFlourescent, 23 kCeilingVent, 21 kTrackLight and 11
		// kCeilingBlower for the ceiling rule, against 903 floorless and 1,220 ceilingless
		// rooms for them to have landed in. TestLintMounting fires both on houses written
		// to provoke them, so these are facts about the corpus and not dead checks.
		{"mount-no-floor", 0,
			"no floor-standing object stands in any of the corpus's 903 kSky, kStratosphere " +
				"or kStars rooms; zero of 903 is not a house style, it is a rule every " +
				"author in 1994 followed without being told"},
		{"mount-no-ceiling", 0,
			"no ceiling fixture hangs in any of the corpus's 1,220 rooms with no ceiling, " +
				"the seven backgrounds of DoesRoomHaveCeiling (Room.c:1172-1206)"},
		// starfield-tiles is the one of the three that is not zero, and its three are why
		// it is a note. All three are Leviathan's -- rooms 172, 206 and 221 -- and one is
		// a straight reversal of 0..7, which is a choice and not a leftover. A class the
		// corpus exercises deliberately cannot be more than a note; that is the rule
		// lint.go's own header now states, and this row is what holds it to it.
		{"starfield-tiles", 3,
			"Leviathan permutes the star field in three rooms and no other shipped house " +
				"does it once; 62 of 62 kStratosphere rooms and 243 of 246 kStars rooms are " +
				"the identity tiling"},
		{"undefined-what", 0,
			"no shipped house contains a `what` outside the nine ranges"},
		{"room-count", 0,
			"every shipped house's nRooms agrees with its file length"},
		{"duplicate-cell", 0,
			"no shipped house has two live rooms at one floor/suite"},
		{"deleted-room", 0,
			"CompressHouse runs on every edit-mode save, so no shipped house carries a " +
				"deleted room"},
		{"grid-range", 0, "every shipped room is inside floor -7..56, suite 0..127"},
		{"name-length", 0, "no shipped room name overruns Str27"},
		{"num-objects", 0,
			"numObjects agrees with the live slot count in all 4,070 shipped rooms, " +
				"which internal/house's corpus test also pins"},
	} {
		if counts[tc.check] != tc.want {
			t.Errorf("%s: got %d, want %d -- %s", tc.check, counts[tc.check], tc.want, tc.why)
		}
	}

	// And the shape of the whole: no errors beyond the one, and the notes vastly
	// outnumbering the warnings, which is what makes `-min warn` a usable default
	// for someone authoring a new house.
	if counts["warn"] >= counts["note"] {
		t.Errorf("the corpus produces %d notes and %d warnings; the warning classes "+
			"were calibrated to be the rarer ones", counts["note"], counts["warn"])
	}
}

// TestLintCorpusIsStable pins that Lint never mutates the house and always reports
// the same findings in the same order -- the property that makes a diff of two lint
// reports mean something.
func TestLintCorpusIsStable(t *testing.T) {
	for _, e := range loadCorpus(t) {
		first := e.house.Lint(fullOptions())
		second := e.house.Lint(fullOptions())
		if len(first) != len(second) {
			t.Fatalf("%s: %d findings then %d", e.stem, len(first), len(second))
		}
		for i := range first {
			if first[i] != second[i] {
				t.Fatalf("%s: finding %d differs between runs:\n    %s\n    %s",
					e.stem, i, first[i], second[i])
			}
		}
		// Re-saving must produce the file it read: Lint took no liberties.
		again, err := e.house.Save()
		if err != nil {
			t.Fatalf("%s: %v", e.stem, err)
		}
		if len(again) != len(e.raw) {
			t.Fatalf("%s: re-saved %d bytes, read %d", e.stem, len(again), len(e.raw))
		}
		for i := range again {
			if again[i] != e.raw[i] {
				t.Fatalf("%s: Lint changed byte %d", e.stem, i)
			}
		}
	}
}
