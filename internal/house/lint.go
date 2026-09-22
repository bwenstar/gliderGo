// Link resolution and the house linter.
//
// The original ships its own validator -- CheckHouseForProblems and its twelve
// sub-checks (GliderPRO/Sources/HouseLegal.c:1052-1218, transcribed in
// docs/analysis/house-format.md part 10) -- but it runs only in edit mode, and nine
// of the twelve steps *repair* rather than report. Ten of them also run on save, so
// a house that has been through the 1994 editor already satisfies them.
//
// What that validator does **not** guarantee is the list at the end of §10.12:
// valid link targets, valid `snd `/`PICT` ids, in-range `who`, `tiles[]` within the
// background picture, and `firstRoom` being in range. Those are precisely the
// mistakes a person authoring a new house by hand will make, and every one of them
// fails silently at play time -- a transporter that does nothing looks like a
// transporter the player has not worked out yet.
//
// So Lint reports and never mutates, and it covers both halves: the original's own
// rules restated as diagnostics, plus the five gaps the original leaves open.
//
// Severity is calibrated against the 22 shipped houses rather than against an
// ideal, because those houses are the specification of what a playable Glider PRO
// house is. They contain 189 dangling links and a basement you cannot climb out of
// -- Slumberland's, and deliberate, with a test of its own -- so neither of those
// classes can be an error without failing the corpus on its own linter.
//
// The rule is narrower than "whatever the corpus does is legal", and the difference
// is the whole of it: **a class the corpus exercises deliberately cannot be an error;
// a class it exercises by overrunning a buffer can.** There is exactly one of the
// latter. CD Demo House room 72 holds a link whose `who` is 35, GenerateRetroLinks
// indexes retroLinkList[who] with no bound check (House.c:577, :600), and the C
// therefore reads into the following room's record. So `link-slot-range` is
// SeverityError and `house lint` does exit 1 on the originals, by design rather than
// by accident, and TestLintCorpus pins that count at 1 so it stays deliberate.
//
// An earlier draft of this paragraph also cited "staircases that lead nowhere". The
// four stair rules fire zero times across all 4,070 corpus rooms: the phrase belonged
// to CheckForStaircasePairs (HouseLegal.c:961-1045) existing -- Calhoun wrote a repair
// pass, so he had seen the problem -- and not to these 22 files, which it read as a
// measurement of. Dropped rather than rewritten, because at note and warn those rules
// need no corpus evidence to justify them. docs/IMPROVEMENTS.md 4.21 has the numbers.
package house

import (
	"fmt"
	"strings"
)

// NumUndergroundFloors is the bias in a packed floor/suite link value: floor 0 in a
// house file is eight storeys below ground (GliderDefines.h:508).
const NumUndergroundFloors = 8

// RoomIsEmpty is kRoomIsEmpty. It is both "no such room" as a return value and, in
// a room's own Suite field, the marker for a deleted room -- CompressHouse and
// LopOffExtraRooms normally remove those before a save, which is why 21 of the 22
// shipped houses are exactly 866 + 348*nRooms bytes long.
const RoomIsEmpty int16 = -1

// The legal grid, from ValidateRoomNumbers (HouseLegal.c:804-815). All four are
// hard-coded literals in the original and none is written in terms of kMaxNumRoomsH,
// kMaxNumRoomsV or kNumUndergroundFloors even though the values coincide, so they
// are copied as literals here too rather than derived.
const (
	MinFloor int16 = -7
	MaxFloor int16 = 56
	MinSuite int16 = 0
	MaxSuite int16 = 127
)

// MaxRoomNameLen is the clamp CheckRoomNameLength applies (HouseLegal.c:870-873).
// Str27 is 28 bytes, so a longer length byte reads into the following `bounds`
// field.
const MaxRoomNameLen = 27

// UntitledRoom is the name CountUntitledRooms looks for, compared
// case-insensitively (HouseLegal.c:846).
const UntitledRoom = "Untitled Room"

// FlowerVariants is how many pictures DrawFlower can select between
// (objectdraw2.go, from ObjectDraw.c). A kFlower whose pict is outside 0..5 draws
// nothing at all.
const FlowerVariants = 6

// ---------------------------------------------------------------------------
// The link encoding (Link.c:34-53)
// ---------------------------------------------------------------------------

// ExtractFloorSuite unpacks a link's floor and suite. Link.c:39-53.
//
// The two fields share one short, and houses before version 2.0 packed them the
// other way round; both orders are read because ExtractFloorSuite itself re-reads
// the house version. No shipped house is version 1, so only the second branch is
// exercised by the corpus -- but the first is what a house saved by an earlier
// build would need, and dropping it would silently transpose every link in it.
//
// This is the canonical implementation. internal/render and internal/game resolve
// links through it rather than repeating the arithmetic, because a port with two
// copies of a version-dependent bit-packing has two chances to get it wrong.
func (h *House) ExtractFloorSuite(combo int16) (floor, suite int16) {
	if h.Version < HouseVersion {
		return combo/100 - NumUndergroundFloors, combo % 100
	}
	return combo%100 - NumUndergroundFloors, combo / 100
}

// MergeFloorSuite packs a floor and suite back into a link value, and is a true
// inverse of ExtractFloorSuite.
//
// The original's MergeFloorSuite (Link.c:34-37) deliberately is *not*: it omits the
// +kNumUndergroundFloors bias and leaves every caller to add it, which three call
// sites do three different ways (Link.c:277, House.c:788 and :804, and an inlined
// fourth at Scrap.c:146). docs/analysis/constants.md 6204 recommends making the
// pair a real bijection that owns the bias, which is what this is.
func (h *House) MergeFloorSuite(floor, suite int16) int16 {
	if h.Version < HouseVersion {
		return (floor+NumUndergroundFloors)*100 + suite
	}
	return suite*100 + floor + NumUndergroundFloors
}

// RoomNumber is the index of the room at a floor and suite, or RoomIsEmpty.
// Room.c:737-760, a linear search as in the original.
//
// It does **not** skip rooms whose Suite is RoomIsEmpty, because the original does
// not: GetRoomNumber compares both fields and nothing else. That matters for one
// specific value. A link whose destination room was deleted holds where == -100,
// which is MergeFloorSuite(0, kRoomIsEmpty) and decodes to floor -8, suite -1 --
// so it resolves to a real index if and only if the house happens to contain a
// deleted room that was on floor -8. Reproducing the search exactly is the only way
// to agree with the game about which of those two things happens.
func (h *House) RoomNumber(floor, suite int16) int16 {
	for i := range h.Rooms {
		if h.Rooms[i].Floor == floor && h.Rooms[i].Suite == suite {
			return int16(i)
		}
	}
	return RoomIsEmpty
}

// The object codes this file needs, resolved from the name table rather than
// respelled as hex. objectNames is the port's single record of the vocabulary
// (GliderDefines.h:311-435); a second spelling of 0x31 here would be a second thing
// to get wrong, and silently so. A typo in one of these names panics the first time
// anything imports the package, which is every test in it.
var (
	codeUpStairs     = mustCode("kUpStairs")
	codeDownStairs   = mustCode("kDownStairs")
	codeSoundTrigger = mustCode("kSoundTrigger")
	codeStar         = mustCode("kStar")
	codeCustomPict   = mustCode("kCustomPict")
	codeFlower       = mustCode("kFlower")

	// The six transports that carry a link, from ObjectIsLinkTransport
	// (Objects.c:219-233). The other ten -- stairs, doors, windows -- move the
	// glider by their own geometry and store no destination.
	linkTransports = codeSet("kMailboxLf", "kMailboxRt", "kFloorTrans",
		"kCeilingTrans", "kInvisTrans", "kDeluxeTrans")

	// The eight switches that carry a link, from ObjectIsLinkSwitch
	// (Objects.c:236-250). kSoundTrigger is the exception and not because it has
	// no link: its `where` is a sound resource id, so resolving it as a room would
	// wire the trigger to whatever room that number happened to name.
	linkSwitches = codeSet("kLightSwitch", "kMachineSwitch", "kThermostat",
		"kPowerSwitch", "kKnifeSwitch", "kInvisSwitch", "kTrigger", "kLgTrigger")

	// The objects that stand on the floor, and the objects that hang from the
	// ceiling. Membership is not a guess from the names: each of these twelve
	// occupies exactly **one** vertical coordinate across all 4,070 corpus rooms,
	// and that coordinate plus the height of its artwork puts it against one
	// surface or the other.
	//
	//	kFloorVent     v 305 + 11 = 316    kCeilingVent    v  8, 11 tall
	//	kFloorBlower   v 304 + 15 = 319    kCeilingBlower  v  5, 15 tall
	//	kSewerGrate    v 303 + 17 = 320    kCeilingLight   v  4, 20 tall
	//	kGrecoVent     v 303 + 18 = 321    kFlourescent    v 12, 12 tall
	//	kSewerBlower   v 292 + 12 = 304    kTrackLight     v  5, 24 tall
	//	kHipLamp       v  23 + 276 = 299
	//	kDecoLamp      v  91 + 212 = 303
	//
	// The two lamps earn their place there rather than here for the same reason:
	// kHipLamp is 276 pixels tall and kDecoLamp 212, so both are standing lamps
	// whose bottoms land on the floor line, not pendants. kTableLamp, kLightBulb
	// and kInvisLight are deliberately in neither set -- their v does vary, because
	// a table lamp sits on whatever furniture the author put under it.
	//
	// The five updraughts and the two downdraughts are also exactly how
	// CreateActiveRects groups them (hotspots.go, from ObjectRects.c): an
	// updraught's lift column runs from `distance` pixels *above* the object down
	// to its top edge, a downdraught's runs down from its bottom. The five flames
	// -- kTaper, kCandle, kStubby, kTiki, kBBQ -- make a thermal column the same
	// way and are still left out, because the corpus does put them outdoors: 18
	// kTiki and 9 kBBQ stand in rooms with no ceiling, and one kCandle and one
	// kStubby in rooms with no floor. A torch on a lawn is a torch on a lawn.
	floorMounted = codeSet("kFloorVent", "kFloorBlower", "kSewerGrate",
		"kGrecoVent", "kSewerBlower", "kHipLamp", "kDecoLamp")

	ceilingMounted = codeSet("kCeilingVent", "kCeilingBlower", "kCeilingLight",
		"kFlourescent", "kTrackLight")
)

// The built-in background range: ids the application's own resource fork supplies,
// as against FirstUserBackground and up, which the house file carries itself
// (docs/analysis/house-format.md 4.2).
//
// Two properties of the range matter to the checks below. Every one of the 18 is
// exactly 512 pixels wide -- eight TileWide columns -- so a tiles[] entry in a
// built-in room can never be out of range and `tile-column` can never fire on one.
// And whether a room has a floor or a ceiling is decided by a fixed list of these
// ids, where a user-art room reads its own `bounds` field or the background's 'bnds'
// resource instead (Room.c:1138-1206).
const (
	FirstBuiltInBackground int16 = 2000
	LastBuiltInBackground  int16 = 2017
	FirstUserBackground    int16 = 3000
)

// floorlessBackgrounds is DoesRoomHaveFloor's list (Room.c:1138-1168): the built-in
// backgrounds a glider falls out of the bottom of. Named rather than bare ids so a
// finding can say kSky.
var floorlessBackgrounds = map[int16]string{
	2015: "kSky", 2016: "kStratosphere", 2017: "kStars",
}

// ceilinglessBackgrounds is DoesRoomHaveCeiling's list (Room.c:1172-1206): the three
// above, plus kRoof -- which has a floor, because a roof is one -- plus the three
// ground levels. So a garden is closed at the bottom and open at the top, and a sky
// room is open at both.
var ceilinglessBackgrounds = map[int16]string{
	2009: "kGarden", 2012: "kMeadow", 2013: "kField", 2014: "kRoof",
	2015: "kSky", 2016: "kStratosphere", 2017: "kStars",
}

// starfieldBackgrounds are the two built-ins whose eight columns are interchangeable,
// so that permuting tiles[] neither breaks nor improves them. See starfieldTiles.
//
// Each carries its own corpus tally because the two are not equally unanimous, and a
// finding that quoted one house's numbers at the other would be the kind of imprecise
// citation docs/IMPROVEMENTS.md 4.21 is about.
var starfieldBackgrounds = map[int16]struct {
	name            string
	identity, rooms int
}{
	2016: {"kStratosphere", 62, 62},
	2017: {"kStars", 243, 246},
}

func mustCode(name string) int16 {
	c, ok := ObjectCode(name)
	if !ok {
		panic("house: no object code named " + name)
	}
	return c
}

func codeSet(names ...string) map[int16]bool {
	m := make(map[int16]bool, len(names))
	for _, n := range names {
		m[mustCode(n)] = true
	}
	return m
}

// LinkCarryingTransport reports whether a `what` code is one of the six transports
// with a stored destination.
func LinkCarryingTransport(what int16) bool { return linkTransports[what] }

// LinkCarryingSwitch reports whether a `what` code is one of the eight switches
// with a stored target.
func LinkCarryingSwitch(what int16) bool { return linkSwitches[what] }

// LinkWhere reads an object's stored destination, and reports false for an object
// that stores none.
//
// The two families read the same two bytes through different union members --
// data.d.where for transports, data.e.where for switches -- which alias at payload
// offset 6. The distinction is documentation rather than arithmetic, and it is kept
// because a future divergence in either struct would otherwise apply to both
// silently. Scrap.c:145-154 has the two branches backwards and gets away with it in
// C for exactly this reason.
func LinkWhere(o Object) (int16, bool) {
	switch {
	case LinkCarryingTransport(o.What):
		return o.Transport().Where, true
	case LinkCarryingSwitch(o.What):
		return o.Switch().Where, true
	}
	return 0, false
}

// LinkWho reads an object's stored target slot, and reports false for an object
// that stores none. Note the asymmetric sentinel: `who` is a Byte, so "unlinked" is
// UnlinkedWho (255) and not -1, while the `where` beside it in the same struct uses
// -1 (Link.c:333-334).
func LinkWho(o Object) (byte, bool) {
	switch {
	case LinkCarryingTransport(o.What):
		return o.Transport().Who, true
	case LinkCarryingSwitch(o.What):
		return o.Switch().Who, true
	}
	return 0, false
}

// RoomLinked resolves a link-carrying object's destination to a room index, or
// RoomIsEmpty. Objects.c:126-174.
//
// An unlinked object, and an object whose link names a room the house does not
// contain, both come back as RoomIsEmpty -- the original's graceful degradation,
// which a port must reproduce rather than error on: 6 of the 22 shipped houses
// contain dangling links, Slumberland 69 of them.
func (h *House) RoomLinked(o Object) int16 {
	combo, ok := LinkWhere(o)
	if !ok || combo == UnlinkedWhere {
		return RoomIsEmpty
	}
	return h.RoomNumber(h.ExtractFloorSuite(combo))
}

// ObjectLinked resolves a link-carrying object's target slot, or -1. Objects.c:177-216.
//
// It range-checks `who` against MaxRoomObs, which the original does not: the corpus
// holds one link with who == 35 (CD Demo House room 72, object 22), and where the C
// reads 132 bytes into the following room's record, Go would panic. Treating an
// out-of-range slot as unlinked is the safest faithful choice.
func (h *House) ObjectLinked(o Object) int16 {
	who, ok := LinkWho(o)
	if !ok || who == UnlinkedWho || int(who) >= MaxRoomObs {
		return -1
	}
	return int16(who)
}

// ---------------------------------------------------------------------------
// Findings
// ---------------------------------------------------------------------------

// Severity ranks a finding by what it costs whoever plays the house.
type Severity int

const (
	// SeverityNote is residue or redundancy: the house plays as authored, but
	// something in it is dead, duplicated, or the leftovers of an edit. Every
	// class of note here occurs in the shipped houses.
	SeverityNote Severity = iota

	// SeverityWarn is something a player can walk into: a transit object that
	// will not transit, a staircase that lands badly, a sound that will not play.
	// The shipped houses contain these too -- Slumberland's basement has no way
	// back up, which commit history records as the original's design, not a bug
	// -- so warn is as far as those can go.
	SeverityWarn

	// SeverityError is a house the loader has to repair, or range-check, before it
	// can be played at all. These are the ones a new house must have none of.
	SeverityError
)

var severityNames = [...]string{"note", "warn", "error"}

func (s Severity) String() string {
	if int(s) < len(severityNames) {
		return severityNames[s]
	}
	return fmt.Sprintf("Severity(%d)", int(s))
}

// ParseSeverity is the inverse, for a command-line threshold flag.
func ParseSeverity(s string) (Severity, bool) {
	for i, n := range severityNames {
		if strings.EqualFold(s, n) {
			return Severity(i), true
		}
	}
	return 0, false
}

// Finding is one thing the linter noticed. Room is -1 for a house-level finding and
// Slot is -1 for anything that is not about one object slot.
//
// Check is a stable kebab-case identifier, so that a house can be held to "no
// link-slot-range findings" in a test or a CI step without matching on prose.
type Finding struct {
	Severity Severity
	Check    string
	Room     int
	Slot     int
	RoomName string
	Message  string
}

// String renders a finding the way the tool prints it: severity, check, location,
// message. The location reads as the room's index *and* name because a link names
// the index and a person reads the name.
func (f Finding) String() string {
	var where string
	switch {
	case f.Room < 0:
		where = "house"
	case f.Slot < 0:
		where = fmt.Sprintf("room %d %q", f.Room, f.RoomName)
	default:
		where = fmt.Sprintf("room %d %q slot %d", f.Room, f.RoomName, f.Slot)
	}
	return fmt.Sprintf("%-5s %-22s %s: %s", f.Severity, f.Check, where, f.Message)
}

// LintOptions supplies the two things a house file cannot answer about itself.
//
// This package deliberately knows nothing about art or sound -- it imports no other
// package in the port, which is what lets internal/render and internal/game both
// depend on it. So the checks that need an asset tree take a predicate, and a nil
// predicate means *skip*, never "answer no". Lint says which checks it skipped, so
// a clean report cannot be mistaken for a complete one.
type LintOptions struct {
	// SoundStatus reports what the asset tree can say about a `snd ` resource id.
	// Used for kSoundTrigger, whose `where` is a resource id rather than a room.
	SoundStatus func(id int16) SoundStatus

	// PictSize reports a PICT's pixel dimensions. Used for room backgrounds, for
	// the tile columns indexed into them, and for kCustomPict.
	PictSize func(id int16) (w, h int, ok bool)
}

// SoundStatus is what an asset tree can say about a house's own `snd ` resource, and it
// has three answers rather than two because two of them are not the house's fault in the
// same way.
//
// A sound the house does not carry is an authoring mistake, and one the original Mac
// punished identically: the trigger got no hot spot there either. A sound the house does
// carry in a form this port cannot decode is *our* gap, and a linter that reported it as a
// defect in a 1994 house would be telling the reader something false about the house. So
// the two get separate checks with separate ids, and one of them points at our tracker
// instead of at the author.
type SoundStatus int

const (
	// SoundMissing: the house carries no resource of that id.
	SoundMissing SoundStatus = iota
	// SoundUnreadable: it carries one, but this port cannot decode it.
	SoundUnreadable
	// SoundOK: it carries one and it loads.
	SoundOK
)

// Lint reports everything questionable about a house, in file order: the
// house-level checks first, then each room, then each room's slots.
//
// The order is deterministic and the output is never mutated into the house, so two
// runs over the same bytes give the same findings and a diff of two reports is
// meaningful.
func (h *House) Lint(opt LintOptions) []Finding {
	l := &linter{h: h, opt: opt}
	l.house()
	for i := range h.Rooms {
		l.room(i)
	}
	return l.out
}

type linter struct {
	h   *House
	opt LintOptions
	out []Finding
}

func (l *linter) add(sev Severity, check string, room, slot int, format string, args ...any) {
	name := ""
	if room >= 0 && room < len(l.h.Rooms) {
		name = l.h.Rooms[room].Name.Text()
	}
	l.out = append(l.out, Finding{
		Severity: sev, Check: check, Room: room, Slot: slot,
		RoomName: name, Message: fmt.Sprintf(format, args...),
	})
}

// ---------------------------------------------------------------------------
// House-level checks
// ---------------------------------------------------------------------------

func (l *linter) house() {
	h := l.h

	var skipped []string
	if l.opt.SoundStatus == nil {
		skipped = append(skipped, "sound-id, sound-unreadable")
	}
	if l.opt.PictSize == nil {
		skipped = append(skipped, "background-pict, tile-column, custom-pict")
	}
	if len(skipped) > 0 {
		l.add(SeverityNote, "checks-skipped", -1, -1,
			"no asset tree was supplied, so these checks did not run: %s",
			strings.Join(skipped, ", "))
	}

	if int(h.NRooms) != len(h.Rooms) {
		l.add(SeverityError, "room-count", -1, -1,
			"the header says %d rooms and the file holds %d; ValidateNumberOfRooms "+
				"resolves this in favour of the file length and rewrites nRooms "+
				"(HouseLegal.c:629-636)", h.NRooms, len(h.Rooms))
	}

	if len(h.Rooms) == 0 {
		l.add(SeverityError, "no-rooms", -1, -1, "the house contains no rooms")
		return
	}

	switch {
	case h.FirstRoom < 0 || int(h.FirstRoom) >= len(h.Rooms):
		l.add(SeverityError, "first-room", -1, -1,
			"firstRoom is %d, outside 0..%d; GetFirstRoomNumber substitutes 0 into a "+
				"local variable and never writes the correction back to the header "+
				"(House.c:209-213), so the house plays from room 0 and still claims "+
				"otherwise", h.FirstRoom, len(h.Rooms)-1)
	case h.Rooms[h.FirstRoom].Suite == RoomIsEmpty:
		l.add(SeverityError, "first-room", -1, -1,
			"firstRoom is %d, which is a deleted room; GetFirstRoomNumber range-checks "+
				"the index but not whether the room is live (House.c:209-213)", h.FirstRoom)
	}

	if h.Version != HouseVersion {
		l.add(SeverityNote, "house-version", -1, -1,
			"version is 0x%04X, not the 0x%04X every shipped house carries; links are "+
				"unpacked the pre-2.0 way below 0x0200, with floor and suite swapped "+
				"(Link.c:39-53)", uint16(h.Version), HouseVersion)
	}

	// CheckDuplicateFloorSuite, HouseLegal.c:646-682. The keeper is the
	// lowest-indexed room at a cell, which is also what RoomNumber's linear search
	// finds, so the later room is addressable by nothing at all.
	type cell struct{ floor, suite int16 }
	seen := make(map[cell]int, len(h.Rooms))
	stars := 0
	for i := range h.Rooms {
		rm := &h.Rooms[i]
		if rm.Suite == RoomIsEmpty {
			continue
		}
		c := cell{rm.Floor, rm.Suite}
		if first, dup := seen[c]; dup {
			l.add(SeverityError, "duplicate-cell", i, -1,
				"floor %d suite %d is already room %d %q; RoomNumber returns the lower "+
					"index for both, so this room is unreachable by any link, staircase or "+
					"neighbour lookup, and CheckDuplicateFloorSuite deletes it on the next "+
					"edit-mode save (HouseLegal.c:665-672)",
				rm.Floor, rm.Suite, first, h.Rooms[first].Name.Text())
		} else {
			seen[c] = i
		}
		for slot := range rm.Objects {
			if rm.Objects[slot].What == codeStar {
				stars++
			}
		}
	}

	// CountStarsInHouse < 1, HouseLegal.c:1203. 69 stars across the 22 shipped
	// houses, and 21 of them have at least one -- Fun House has none, so the
	// original's own validator would flag it in red too. That one house is the
	// reason this is a warning and not an error.
	if stars == 0 {
		l.add(SeverityWarn, "no-stars", -1, -1,
			"the house contains no kStar, so it cannot be won; CheckHouseForProblems "+
				"warns about exactly this (HouseLegal.c:1203)")
	}
}

// ---------------------------------------------------------------------------
// Room-level checks
// ---------------------------------------------------------------------------

func (l *linter) room(i int) {
	rm := &l.h.Rooms[i]

	if rm.Suite == RoomIsEmpty {
		l.add(SeverityNote, "deleted-room", i, -1,
			"suite is kRoomIsEmpty, so this room is deleted; CompressHouse and "+
				"LopOffExtraRooms run unconditionally on an edit-mode save "+
				"(HouseLegal.c:1103-1105), which is why 21 of the 22 shipped houses are "+
				"exactly 866 + 348*nRooms bytes")
		return
	}

	// ValidateRoomNumbers, HouseLegal.c:804-821. The original deletes the room
	// rather than clamping, and the two tests are sequential rather than exclusive,
	// so a room with both wrong counts twice -- reproduced here as two findings.
	if rm.Floor < MinFloor || rm.Floor > MaxFloor {
		l.add(SeverityError, "grid-range", i, -1,
			"floor is %d, outside %d..%d; ValidateRoomNumbers deletes the room rather "+
				"than clamping it (HouseLegal.c:804-811)", rm.Floor, MinFloor, MaxFloor)
	}
	if rm.Suite < MinSuite || rm.Suite > MaxSuite {
		l.add(SeverityError, "grid-range", i, -1,
			"suite is %d, outside %d..%d; ValidateRoomNumbers deletes the room rather "+
				"than clamping it (HouseLegal.c:814-821)", rm.Suite, MinSuite, MaxSuite)
	}

	// CheckRoomNameLength, HouseLegal.c:870-874. Str27 is 28 bytes, so a longer
	// length byte reads past the name into `bounds`.
	if rm.Name[0] > MaxRoomNameLen {
		l.add(SeverityError, "name-length", i, -1,
			"the name's length byte is %d, past the %d Str27 holds; the reader would "+
				"run into the bounds field, and CheckRoomNameLength clamps it to %d "+
				"(HouseLegal.c:870-874)", rm.Name[0], MaxRoomNameLen, MaxRoomNameLen)
	}

	// CountUntitledRooms, HouseLegal.c:846, case-insensitive and warn-only.
	if strings.EqualFold(rm.Name.Text(), UntitledRoom) {
		l.add(SeverityNote, "untitled-room", i, -1,
			"the room is still called %q; CountUntitledRooms counts these so that an "+
				"author can find the ones they have not got to yet (HouseLegal.c:834-851)",
			UntitledRoom)
	}

	// MakeSureNumObjectsJives, HouseLegal.c:899-907. LiveObjects is the authority.
	if int(rm.NumObjects) != rm.LiveObjects() {
		l.add(SeverityNote, "num-objects", i, -1,
			"numObjects says %d and %d slots are live; the field is advisory and "+
				"MakeSureNumObjectsJives overwrites it from the count (HouseLegal.c:899-907)",
			rm.NumObjects, rm.LiveObjects())
	}

	l.background(i, rm)
	l.starfieldTiles(i, rm)

	// The per-room tallies. Two sound triggers cannot both work, and a doubled
	// staircase is decided by slot order rather than by position.
	soundTriggers, upStairs, downStairs := 0, 0, 0

	for slot := range rm.Objects {
		o := rm.Objects[slot]
		if o.IsEmpty() {
			continue
		}

		if o.Group() == GroupNone {
			l.add(SeverityError, "undefined-what", i, slot,
				"what is 0x%02X, which selects no union variant and falls through every "+
					"switch in the original; it would index an unset srcRects[] entry "+
					"(docs/analysis/house-format.md 5.1). No shipped house contains one",
				uint16(o.What))
			// Nothing below can read this object's payload meaningfully.
			continue
		}

		switch o.What {
		case codeSoundTrigger:
			soundTriggers++
			l.soundTrigger(i, slot, o)
		case codeUpStairs:
			upStairs++
			l.stairs(i, slot, rm, 1, codeDownStairs,
				"GetUpStairsRightEdge finds no kDownStairs to measure and returns its "+
					"default of kRoomWide, so the glider walks up out of the right-hand "+
					"wall (ObjectRects.c:1135-1159)")
		case codeDownStairs:
			downStairs++
			l.stairs(i, slot, rm, -1, codeUpStairs,
				"GetDownStairsLeftEdge finds no kUpStairs to measure and returns 0, so "+
					"the glider is placed a full glider-width off the left edge of the "+
					"room (ObjectRects.c:1163-1185)")
		case codeCustomPict:
			l.customPict(i, slot, o)
		case codeFlower:
			if v := o.Clutter().Pict; v < 0 || v >= FlowerVariants {
				l.add(SeverityWarn, "flower-variant", i, slot,
					"pict is %d, outside 0..%d; DrawFlower range-checks it and draws "+
						"nothing at all, so the flower is invisible", v, FlowerVariants-1)
			}
		}

		l.mounting(i, rm, slot, o)
		l.link(i, slot, o)
	}

	if soundTriggers > 1 {
		l.add(SeverityWarn, "sound-trigger-crowded", i, -1,
			"%d kSoundTrigger objects in one room; the loader holds a single reserved "+
				"sound slot, freed at every room change (RoomGraphics.c:58), so only one "+
				"of them has a sound to play", soundTriggers)
	}
	if upStairs > 1 {
		l.add(SeverityNote, "stairs-doubled", i, -1,
			"%d kUpStairs in one room; GetDownStairsLeftEdge breaks on the first match, "+
				"so the lower-numbered slot decides where an arriving glider appears and "+
				"the others are decoration (ObjectRects.c:1163-1185)", upStairs)
	}
	if downStairs > 1 {
		l.add(SeverityNote, "stairs-doubled", i, -1,
			"%d kDownStairs in one room; GetUpStairsRightEdge breaks on the first match, "+
				"so the lower-numbered slot decides where an arriving glider appears and "+
				"the others are decoration (ObjectRects.c:1135-1159)", downStairs)
	}
}

// background checks the room's PICT and the eight tile columns indexed into it.
//
// tiles[i] is a column index, not a pixel offset: src.left = tiles[i] * kTileWide
// (Room.c:291-292). There is no bounds check in the original, so a column past the
// end of the picture reads off the right edge of the GWorld.
func (l *linter) background(i int, rm *Room) {
	if l.opt.PictSize == nil || rm.Background == 0 {
		return
	}
	w, _, ok := l.opt.PictSize(rm.Background)
	if !ok {
		l.add(SeverityWarn, "background-pict", i, -1,
			"background is PICT %d, which is in neither the house's own resources nor "+
				"the application's; LoadGraphicSpecial falls back to PICT 2000 "+
				"(RoomGraphics.c:145), so the room draws as the wrong place rather than "+
				"failing", rm.Background)
		return
	}
	columns := w / TileWide
	for t, col := range rm.Tiles {
		if col < 0 || int(col) >= columns {
			l.add(SeverityWarn, "tile-column", i, -1,
				"tiles[%d] is column %d and PICT %d is only %d columns wide (%d px); "+
					"src.left = tiles[i] * %d is not bounds-checked anywhere, so this "+
					"reads off the right edge of the background (Room.c:291-292)",
				t, col, rm.Background, columns, w, TileWide)
		}
	}
}

// TileWide is kTileWide: the width of one background column (GliderDefines.h:497).
const TileWide = 64

// starfieldTiles notes a kStratosphere or kStars room tiled other than 0..7.
//
// This is the weakest of the three background checks and it is a note for a reason
// worth writing down, because the reasoning went the other way first.
//
// Every built-in background is one 512-pixel picture, so tiles[] selects eight
// columns out of eight: for most backgrounds that is the whole point -- kSky's 595
// corpus rooms use 98 distinct patterns and not one of them is the identity, because
// rearranging cloud columns is how one picture becomes 595 different skies. For the
// two starfields it is not, because their columns are interchangeable. A permuted
// star field is another star field. The corpus agrees, unanimously for one of them:
// 62 of 62 kStratosphere rooms are the identity, and 243 of 246 kStars rooms.
//
// So a permutation here is harmless, which is exactly why it is worth a note: harmless
// is what a leftover looks like. The three exceptions are all Leviathan's -- rooms 172,
// 206 and 221, one of them a straight reversal -- and they are the author's choice.
// What this catches is the other case, which is a room whose background was changed to
// kStars after its tiles were authored for something else; the port's own generator did
// precisely that and nothing in the toolchain objected (docs/IMPROVEMENTS.md 4.22).
//
// Not a warning, by the rule this file's header states: the corpus exercises the class
// deliberately, so the finding cannot claim more than "look at this".
func (l *linter) starfieldTiles(i int, rm *Room) {
	bg, ok := starfieldBackgrounds[rm.Background]
	if !ok {
		return
	}
	for t, col := range rm.Tiles {
		if int(col) != t {
			l.add(SeverityNote, "starfield-tiles", i, -1,
				"background is %s (PICT %d) and tiles[] is %v rather than the identity; "+
					"its eight columns are interchangeable, so this draws a star field "+
					"either way -- but %d of the corpus's %d %s rooms are the identity, and "+
					"the usual cause of a permuted one is a background changed after the "+
					"tiles were authored", bg.name, rm.Background, rm.Tiles,
				bg.identity, bg.rooms, bg.name)
			return
		}
	}
}

// mounting reports an object standing against a surface the room does not have.
//
// The two rules are the mirror of each other and neither is in the original's
// validator, which has no check that relates an object to its background at all --
// CheckHouseForProblems asks only whether each room's own fields are legal. Nothing
// breaks at play time either: GetObjectRect places every one of these objects at the
// absolute coordinates it stores (ObjectRects.c:32-273, no term in it derives from the
// room's openings), LiftIt reads the object and not the floor, and the art composes
// fine. The room draws, the house lints, the lift works, and a player sees a ventilation
// grille bolted to the open sky.
//
// Warn rather than error, and warn rather than note, on the corpus: across the 2,273
// rooms with a built-in background -- 903 with no floor and 1,220 with no ceiling --
// there is not one instance of either rule. Zero of 903 for the floor rule covers 903
// kFloorVent, 336 kSewerGrate, 105 kSewerBlower, 70 kFloorBlower, 25 kGrecoVent, 40
// kDecoLamp and 23 kHipLamp placements; zero of 1,220 for the ceiling rule covers 112
// kCeilingLight, 79 kFlourescent, 23 kCeilingVent, 21 kTrackLight and 11 kCeilingBlower.
// That is not a house style, it is a rule every author in 1994 followed
// without being told, so no shipped house is made to fail by saying it out loud -- and
// warn is what `-min warn` shows an author who has not asked for notes.
//
// Only built-in backgrounds are checked. A user-art room's openings come from its own
// `bounds` field, or from the background's 'bnds' resource when that field is 0
// (Room.c:1138-1206, boundsCode), and this package cannot read a resource fork -- it
// has PictSize and nothing analogous for 'bnds'. Rather than check half the rule on
// half the rooms, the limit is stated here and in docs/IMPROVEMENTS.md 4.22.
func (l *linter) mounting(i int, rm *Room, slot int, o Object) {
	if rm.Background < FirstBuiltInBackground || rm.Background > LastBuiltInBackground {
		return
	}
	if name, ok := floorlessBackgrounds[rm.Background]; ok && floorMounted[o.What] {
		l.add(SeverityWarn, "mount-no-floor", i, slot,
			"%s rests on the floor and this room's background is %s (PICT %d), which "+
				"DoesRoomHaveFloor gives no floor at all (Room.c:1138-1168); the object "+
				"draws and works, so the only symptom is machinery standing on open air. "+
				"No floor-mounted object appears in any of the corpus's %d floorless rooms",
			ObjectName(o.What), name, rm.Background, 903)
	}
	if name, ok := ceilinglessBackgrounds[rm.Background]; ok && ceilingMounted[o.What] {
		l.add(SeverityWarn, "mount-no-ceiling", i, slot,
			"%s hangs from the ceiling and this room's background is %s (PICT %d), which "+
				"DoesRoomHaveCeiling gives no ceiling (Room.c:1172-1206); the object draws "+
				"and works, so the only symptom is a fixture suspended from nothing. No "+
				"ceiling-mounted object appears in any of the corpus's %d ceilingless rooms",
			ObjectName(o.What), name, rm.Background, 1220)
	}
}

// ---------------------------------------------------------------------------
// Object-level checks
// ---------------------------------------------------------------------------

// link is the check the whole file exists for: the five ways a stored link can fail
// to be a link, in the order they stop mattering.
func (l *linter) link(room, slot int, o Object) {
	where, ok := LinkWhere(o)
	if !ok {
		return
	}
	who, _ := LinkWho(o)
	name := ObjectName(o.What)

	if where == UnlinkedWhere {
		if who != UnlinkedWho {
			l.add(SeverityWarn, "link-unlinked", room, slot,
				"%s has no destination room (where is -1) but still names slot %d; "+
					"GetRoomLinked returns -1 first, so the object does nothing and the "+
					"slot number is misleading residue", name, who)
		} else {
			l.add(SeverityNote, "link-unlinked", room, slot,
				"%s is not linked to anything (where -1, who 255), which is the state "+
					"DoUnlink writes (Link.c:246-247)", name)
		}
		return
	}

	floor, suite := l.h.ExtractFloorSuite(where)
	dest := l.h.RoomNumber(floor, suite)
	if dest == RoomIsEmpty {
		// Both severities are written as literal calls rather than as one call with
		// a computed severity, so that the two levels this check reports at are
		// visible in the code and can be read off it -- which is what
		// TestLintCatalogueIsComplete does.
		const dangling = "%s links to where %d, which unpacks to floor %d suite %d and " +
			"no room is there; GetRoomNumber returns kRoomIsEmpty and the object " +
			"silently does nothing%s"
		if who == UnlinkedWho {
			// 165 of the 189 dangling links in the shipped corpus are exactly this,
			// and all of them are inert for the second reason as well.
			l.add(SeverityNote, "link-dangling-room", room, slot, dangling,
				name, where, floor, suite,
				", and who is 255 as well, so the link was already dead before the "+
					"room went; this is the residue a deleted destination leaves, and "+
					"it is the commonest finding in the shipped houses")
		} else {
			l.add(SeverityWarn, "link-dangling-room", room, slot, dangling,
				name, where, floor, suite, "")
		}
		return
	}

	if who == UnlinkedWho {
		l.add(SeverityNote, "link-no-target", room, slot,
			"%s links to room %d %q but names no object in it (who is 255), so the link "+
				"is inert even though the room resolves (Link.c:333-334)",
			name, dest, l.h.Rooms[dest].Name.Text())
		return
	}

	if int(who) >= MaxRoomObs {
		l.add(SeverityError, "link-slot-range", room, slot,
			"%s links to slot %d of room %d %q, which holds %d slots; GenerateRetroLinks "+
				"indexes retroLinkList[who] with no bound check (House.c:577, :600) and "+
				"the original reads into the following room's record. The shipped corpus "+
				"holds exactly one of these",
			name, who, dest, l.h.Rooms[dest].Name.Text(), MaxRoomObs)
		return
	}

	target := l.h.Rooms[dest].Objects[who]
	if target.IsEmpty() {
		l.add(SeverityWarn, "link-slot-empty", room, slot,
			"%s links to slot %d of room %d %q, which is empty; DeleteObject clears the "+
				"reverse-link list but leaves forward links pointing at the hole "+
				"(docs/analysis/editor-object-manipulation.md 9)",
			name, who, dest, l.h.Rooms[dest].Name.Text())
		return
	}

	if LinkCarryingTransport(o.What) && target.Group() != GroupTransport {
		l.add(SeverityNote, "link-target-kind", room, slot,
			"%s links to %s, which is not a transit object; WhatAreWeLinkedTo returns "+
				"LinkedToOther and the glider materialises at that object's own rect, "+
				"with nothing there to send it back (Transit.c:31-61)",
			name, target.String())
	}
}

// stairs is CheckForStaircasePairs, HouseLegal.c:961-1045, restated per instance.
//
// dFloor is +1 for kUpStairs and -1 for kDownStairs, and counterpart is the code the
// destination room must hold. **Read those two together and note the crossing**: a
// glider arriving from below emerges from behind the kDownStairs of the room above,
// because the two objects are the two ends of one flight. GetUpStairsRightEdge
// searches for kDownStairs and GetDownStairsLeftEdge for kUpStairs, which is the C
// verbatim and not a transcription slip in either language.
func (l *linter) stairs(room, slot int, rm *Room, dFloor int16, counterpart int16, consequence string) {
	name := ObjectName(rm.Objects[slot].What)
	dest := l.h.RoomNumber(rm.Floor+dFloor, rm.Suite)
	if dest == RoomIsEmpty {
		l.add(SeverityWarn, "stairs-no-room", room, slot,
			"%s leads to floor %d suite %d and there is no room there; "+
				"CheckForStaircasePairs warns about exactly this and changes nothing "+
				"(HouseLegal.c:982-989, :1013-1020)", name, rm.Floor+dFloor, rm.Suite)
		return
	}
	target := &l.h.Rooms[dest]
	for i := range target.Objects {
		if target.Objects[i].What == counterpart {
			return
		}
	}
	l.add(SeverityWarn, "stairs-unpaired", room, slot,
		"%s leads to room %d %q, which holds no %s to arrive on; %s",
		name, dest, target.Name.Text(), ObjectName(counterpart), consequence)
}

// soundTrigger checks the one switch whose `where` is not a room.
func (l *linter) soundTrigger(room, slot int, o Object) {
	id := o.Switch().Where
	if id == UnlinkedWhere {
		l.add(SeverityNote, "sound-id", room, slot,
			"kSoundTrigger names no sound (where is -1), so touching it does nothing")
		return
	}
	if l.opt.SoundStatus == nil {
		return
	}
	switch l.opt.SoundStatus(id) {
	case SoundOK:
	case SoundUnreadable:
		l.add(SeverityWarn, "sound-unreadable", room, slot,
			"kSoundTrigger names 'snd ' %d, which the house does carry and gliderGo "+
				"cannot decode (MACE 6:1; docs/IMPROVEMENTS.md 2.49). This one is ours, "+
				"not the author's: the sound played on a 1994 Mac. Here LoadTriggerSound "+
				"fails, so CreateActiveRects composes the room with no hot spot and the "+
				"trigger cannot be touched at all", id)
	default:
		l.add(SeverityWarn, "sound-id", room, slot,
			"kSoundTrigger names 'snd ' %d, which the house does not carry; "+
				"LoadTriggerSound looks only in the house's own resources, never the "+
				"application's (Sound.c:265-303), and CreateActiveRects then composes the "+
				"room with no hot spot at all -- so the trigger is not merely silent, it "+
				"cannot be touched", id)
	}
}

// customPict checks kCustomPict, whose PICT id lives in data.g.height rather than in
// a pict field -- the one object where the id is stored in a member named for
// something else (docs/analysis/house-format.md 12.9).
func (l *linter) customPict(room, slot int, o Object) {
	if l.opt.PictSize == nil {
		return
	}
	id := o.Appliance().Height
	if _, _, ok := l.opt.PictSize(id); !ok {
		l.add(SeverityWarn, "custom-pict", room, slot,
			"kCustomPict names PICT %d, which is in neither the house's own resources "+
				"nor the application's; GetObjectRect rewrites the id to 10000 and the "+
				"object draws as whatever that is (ObjectRects.c:216)", id)
	}
}

// ---------------------------------------------------------------------------
// Summarising
// ---------------------------------------------------------------------------

// Counts tallies findings by severity, for a caller that needs an exit code or a
// one-line summary.
func Counts(findings []Finding) (notes, warns, errs int) {
	for _, f := range findings {
		switch f.Severity {
		case SeverityNote:
			notes++
		case SeverityWarn:
			warns++
		case SeverityError:
			errs++
		}
	}
	return notes, warns, errs
}

// AtLeast filters findings to those at or above a severity, preserving order.
func AtLeast(findings []Finding, min Severity) []Finding {
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		if f.Severity >= min {
			out = append(out, f)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// The catalogue
// ---------------------------------------------------------------------------

// LintCheck is one entry in the catalogue: what a check id means, in one line.
//
// Severity is the *worst* the check can report, because three of them report at two
// levels -- a dangling link is a note when its `who` is already 255 and a warning
// when it is not, which is the difference between residue and something a player can
// walk into.
type LintCheck struct {
	ID       string
	Severity Severity
	What     string
}

// LintChecks is every check Lint can emit, sorted by id.
//
// It lives here rather than in the command that prints it for one reason: a check id
// that appears in a report with no explanation anywhere is worse than no check, and
// keeping the list beside the code that emits it is what lets a test hold the two
// together. TestLintCatalogueIsComplete parses this file's own syntax tree and fails
// if any l.add call names an id that is not in this table, or if any table entry
// claims a severity no call site actually uses.
func LintChecks() []LintCheck {
	return []LintCheck{
		{"background-pict", SeverityWarn, "the room's background PICT is in neither resource chain"},
		{"checks-skipped", SeverityNote, "no asset tree was supplied, so some checks did not run"},
		{"custom-pict", SeverityWarn, "a kCustomPict names a PICT that is in neither chain"},
		{"deleted-room", SeverityNote, "a room marked deleted that a save would have compressed away"},
		{"duplicate-cell", SeverityError, "two rooms claim one floor/suite, so the later one is unreachable"},
		{"first-room", SeverityError, "firstRoom is out of range, or names a deleted room"},
		{"flower-variant", SeverityWarn, "a kFlower's pict is outside 0..5, so it draws nothing"},
		{"grid-range", SeverityError, "floor outside -7..56 or suite outside 0..127"},
		{"house-version", SeverityNote, "not version 0x0200, so links unpack the pre-2.0 way"},
		{"link-dangling-room", SeverityWarn, "a link whose destination room does not exist"},
		{"link-no-target", SeverityNote, "a link with a room but no object slot (who is 255)"},
		{"link-slot-empty", SeverityWarn, "a link to a slot that holds no object"},
		{"link-slot-range", SeverityError, "a link to a slot past the 24 a room holds"},
		{"link-target-kind", SeverityNote, "a transport linked to something that is not a transport"},
		{"link-unlinked", SeverityWarn, "a transit object or switch with no destination"},
		{"mount-no-ceiling", SeverityWarn, "a ceiling fixture in a background that has no ceiling"},
		{"mount-no-floor", SeverityWarn, "a floor-standing object in a background that has no floor"},
		{"name-length", SeverityError, "the room name's length byte runs past Str27"},
		{"no-rooms", SeverityError, "the house has no rooms at all"},
		{"no-stars", SeverityWarn, "no kStar anywhere, so the house cannot be won"},
		{"num-objects", SeverityNote, "numObjects disagrees with the live slot count"},
		{"room-count", SeverityError, "the header's nRooms disagrees with the file length"},
		{"sound-id", SeverityWarn, "a kSoundTrigger naming a sound the house does not carry"},
		{"sound-trigger-crowded", SeverityWarn, "more than one kSoundTrigger in a room, which holds one"},
		{"sound-unreadable", SeverityWarn, "a sound the house carries that gliderGo cannot decode (our gap)"},
		{"stairs-doubled", SeverityNote, "two staircases of one kind in a room; the lower slot wins"},
		{"stairs-no-room", SeverityWarn, "a staircase leading to a floor with no room on it"},
		{"stairs-unpaired", SeverityWarn, "a staircase whose destination has no counterpart to arrive on"},
		{"starfield-tiles", SeverityNote, "a kStars or kStratosphere room tiled other than 0..7"},
		{"tile-column", SeverityWarn, "a tiles[] column is past the right edge of the background"},
		{"undefined-what", SeverityError, "an object code that selects no union variant"},
		{"untitled-room", SeverityNote, "the room is still called Untitled Room"},
	}
}
