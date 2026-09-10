package house

import (
	"encoding/binary"
	"fmt"
)

// Object is objectType: a 2-byte `what` code and a 10-byte union whose
// interpretation `what` selects (GliderPRO/Headers/GliderStructs.h:90-105).
//
// The union is kept as raw bytes with typed views over it, rather than as nine
// Go structs, for three reasons that all come from the corpus:
//
//   - An empty slot (What == ObjectIsEmpty) still carries 10 bytes of live
//     residue, because CreateNewRoom set only `what`
//     (GliderPRO/Sources/Room.c:181-182). Those bytes are part of the file.
//   - Seven `what` values are undefined (0x00, 0x20, 0x30, 0x4A-0x50, 0x59-0x60,
//     0x6F-0x70, 0x7A-0x80) and select no variant at all. None occur in the 22
//     shipped houses, but a user house may contain one and must still load.
//   - It mirrors what the original actually is. A union is not a tagged union;
//     the editor writes through one member and the game reads another in at
//     least one place, and a model that cannot express that would be lying.
//
// All typed views are value types: read with Blower(), write with SetBlower().
type Object struct {
	What int16
	Data [10]byte
}

// IsEmpty reports whether this slot holds no object. Slots are never compacted
// and their indices are addressed by links, so an empty slot in the middle of a
// room is normal (six shipped rooms have one) and must be preserved in place.
func (o Object) IsEmpty() bool { return o.What == ObjectIsEmpty }

// Group is the family of object types that share a union variant. The variant is
// selected by an inclusive range on `what`, never by the high nibble: 0x10
// (kLiftArea) is a blower and 0x40 (kDeluxeTrans) is a transport, so nibble
// arithmetic gets both wrong (docs/analysis/house-format.md 5.1).
type Group uint8

// The nine groups, named for their union member in GliderStructs.h and lettered
// as the C declares them (data.a .. data.i).
const (
	GroupNone      Group = iota // undefined `what`, or an empty slot
	GroupBlower                 // data.a  blowerType     0x01-0x10
	GroupFurniture              // data.b  furnitureType  0x11-0x1F
	GroupBonus                  // data.c  bonusType      0x21-0x2F
	GroupTransport              // data.d  transportType  0x31-0x40
	GroupSwitch                 // data.e  switchType     0x41-0x49
	GroupLight                  // data.f  lightType      0x51-0x58
	GroupAppliance              // data.g  applianceType  0x61-0x6E
	GroupEnemy                  // data.h  enemyType      0x71-0x79
	GroupClutter                // data.i  clutterType    0x81-0x8F
)

var groupNames = [...]string{"none", "blower", "furniture", "bonus", "transport",
	"switch", "light", "appliance", "enemy", "clutter"}

func (g Group) String() string {
	if int(g) < len(groupNames) {
		return groupNames[g]
	}
	return fmt.Sprintf("Group(%d)", uint8(g))
}

// Member is the C union member letter this group decodes through, which is how
// the analysis documents and the original source refer to it.
func (g Group) Member() string {
	if g == GroupNone {
		return "-"
	}
	return string(rune('a' + int(g) - 1))
}

// groupFields names each variant's members in the order the text format writes
// them, which is also the order the union declares them. One table, read by both
// the writer's verification and the parser's required-field check, so the two
// cannot drift apart -- and exported because an authoring tool needs to be able
// to tell a person which fields an object line must carry.
var groupFields = [...][]string{
	GroupNone:      nil,
	GroupBlower:    {"at", "distance", "initial", "state", "vector", "tall"},
	GroupFurniture: {"rect", "pict"},
	GroupBonus:     {"at", "length", "points", "state", "initial"},
	GroupTransport: {"at", "tall", "where", "who", "wide"},
	GroupSwitch:    {"at", "delay", "where", "who", "type"},
	GroupLight:     {"at", "length", "byte0", "byte1", "initial", "state"},
	GroupAppliance: {"at", "height", "byte0", "delay", "initial", "state"},
	GroupEnemy:     {"at", "length", "delay", "byte0", "initial", "state"},
	GroupClutter:   {"rect", "pict"},
}

// Fields returns the text-format field names of a variant, in writing order.
// GroupNone returns nil: an undefined `what` has no named fields and is written
// as raw bytes. The result is freshly allocated, so a caller may sort it.
func (g Group) Fields() []string {
	if int(g) >= len(groupFields) || groupFields[g] == nil {
		return nil
	}
	return append([]string(nil), groupFields[g]...)
}

// FieldArity is how many numbers a field name takes. Two of them are composites
// of the Toolbox types -- `at` is a Point and `rect` is a Rect -- because writing
// a rectangle as four separate keys would let an author give three of them.
func FieldArity(name string) int {
	switch name {
	case "at":
		return 2 // Point: vertical, horizontal
	case "rect":
		return 4 // Rect: top, left, bottom, right
	}
	return 1
}

// groupRanges is the inclusive-range table from KeepObjectLegal's switch
// (GliderPRO/Sources/HouseLegal.c), the only place in the original where every
// code is enumerated against its union member.
var groupRanges = []struct {
	lo, hi int16
	group  Group
}{
	{0x01, 0x10, GroupBlower},
	{0x11, 0x1F, GroupFurniture},
	{0x21, 0x2F, GroupBonus},
	{0x31, 0x40, GroupTransport},
	{0x41, 0x49, GroupSwitch},
	{0x51, 0x58, GroupLight},
	{0x61, 0x6E, GroupAppliance},
	{0x71, 0x79, GroupEnemy},
	{0x81, 0x8F, GroupClutter},
}

// GroupOf classifies a `what` code. GroupNone means the code is undefined: it
// falls through every switch in the original and indexes an unset srcRects[]
// entry, so a port should treat it as invalid rather than guess.
func GroupOf(what int16) Group {
	for _, r := range groupRanges {
		if what >= r.lo && what <= r.hi {
			return r.group
		}
	}
	return GroupNone
}

// Group returns the object's union family.
func (o Object) Group() Group { return GroupOf(o.What) }

// objectNames maps every defined `what` to the exact C identifier from
// GliderPRO/Headers/GliderDefines.h:311-435. All 117 occur at least once across
// the 22 shipped houses and no undefined code occurs anywhere
// (docs/analysis/house-format.md 5.2), so this table is also the complete
// vocabulary a new house may use.
//
// The C names are kept verbatim rather than prettified: the extracted sprite
// files are named <HEX>_<kConstant>.png, so a room dump joins against the art
// directory with no lookup table.
var objectNames = map[int16]string{
	// Blower -- data.a
	0x01: "kFloorVent",
	0x02: "kCeilingVent",
	0x03: "kFloorBlower",
	0x04: "kCeilingBlower",
	0x05: "kSewerGrate",
	0x06: "kLeftFan",
	0x07: "kRightFan",
	0x08: "kTaper",
	0x09: "kCandle",
	0x0A: "kStubby",
	0x0B: "kTiki",
	0x0C: "kBBQ",
	0x0D: "kInvisBlower",
	0x0E: "kGrecoVent",
	0x0F: "kSewerBlower",
	0x10: "kLiftArea",

	// Furniture -- data.b
	0x11: "kTable",
	0x12: "kShelf",
	0x13: "kCabinet",
	0x14: "kFilingCabinet",
	0x15: "kWasteBasket",
	0x16: "kMilkCrate",
	0x17: "kCounter",
	0x18: "kDresser",
	0x19: "kDeckTable",
	0x1A: "kStool",
	0x1B: "kTrunk",
	0x1C: "kInvisObstacle",
	0x1D: "kManhole",
	0x1E: "kBooks",
	0x1F: "kInvisBounce",

	// Bonus -- data.c
	0x21: "kRedClock",
	0x22: "kBlueClock",
	0x23: "kYellowClock",
	0x24: "kCuckoo",
	0x25: "kPaper",
	0x26: "kBattery",
	0x27: "kBands",
	0x28: "kGreaseRt",
	0x29: "kGreaseLf",
	0x2A: "kFoil",
	0x2B: "kInvisBonus",
	0x2C: "kStar",
	0x2D: "kSparkle",
	0x2E: "kHelium",
	0x2F: "kSlider",

	// Transport -- data.d
	0x31: "kUpStairs",
	0x32: "kDownStairs",
	0x33: "kMailboxLf",
	0x34: "kMailboxRt",
	0x35: "kFloorTrans",
	0x36: "kCeilingTrans",
	0x37: "kDoorInLf",
	0x38: "kDoorInRt",
	0x39: "kDoorExRt",
	0x3A: "kDoorExLf",
	0x3B: "kWindowInLf",
	0x3C: "kWindowInRt",
	0x3D: "kWindowExRt",
	0x3E: "kWindowExLf",
	0x3F: "kInvisTrans",
	0x40: "kDeluxeTrans",

	// Switch -- data.e
	0x41: "kLightSwitch",
	0x42: "kMachineSwitch",
	0x43: "kThermostat",
	0x44: "kPowerSwitch",
	0x45: "kKnifeSwitch",
	0x46: "kInvisSwitch",
	0x47: "kTrigger",
	0x48: "kLgTrigger",
	0x49: "kSoundTrigger",

	// Light -- data.f
	0x51: "kCeilingLight",
	0x52: "kLightBulb",
	0x53: "kTableLamp",
	0x54: "kHipLamp",
	0x55: "kDecoLamp",
	0x56: "kFlourescent",
	0x57: "kTrackLight",
	0x58: "kInvisLight",

	// Appliance -- data.g
	0x61: "kShredder",
	0x62: "kToaster",
	0x63: "kMacPlus",
	0x64: "kGuitar",
	0x65: "kTV",
	0x66: "kCoffee",
	0x67: "kOutlet",
	0x68: "kVCR",
	0x69: "kStereo",
	0x6A: "kMicrowave",
	0x6B: "kCinderBlock",
	0x6C: "kFlowerBox",
	0x6D: "kCDs",
	0x6E: "kCustomPict",

	// Enemy -- data.h
	0x71: "kBalloon",
	0x72: "kCopterLf",
	0x73: "kCopterRt",
	0x74: "kDartLf",
	0x75: "kDartRt",
	0x76: "kBall",
	0x77: "kDrip",
	0x78: "kFish",
	0x79: "kCobweb",

	// Clutter -- data.i
	0x81: "kOzma",
	0x82: "kMirror",
	0x83: "kMousehole",
	0x84: "kFireplace",
	0x85: "kFlower",
	0x86: "kWallWindow",
	0x87: "kBear",
	0x88: "kCalendar",
	0x89: "kVase1",
	0x8A: "kVase2",
	0x8B: "kBulletin",
	0x8C: "kCloud",
	0x8D: "kFaucet",
	0x8E: "kRug",
	0x8F: "kChimes",
}

// objectCodes is the reverse of objectNames, for the text parser.
var objectCodes = func() map[string]int16 {
	m := make(map[string]int16, len(objectNames))
	for code, name := range objectNames {
		m[name] = code
	}
	return m
}()

// ObjectName returns the C identifier for a `what` code, or "" if undefined.
// Empty slots return "kObjectIsEmpty", the original's own name for the sentinel.
func ObjectName(what int16) string {
	if what == ObjectIsEmpty {
		return "kObjectIsEmpty"
	}
	return objectNames[what]
}

// ObjectCode is the inverse of ObjectName.
func ObjectCode(name string) (int16, bool) {
	if name == "kObjectIsEmpty" {
		return ObjectIsEmpty, true
	}
	c, ok := objectCodes[name]
	return c, ok
}

// ObjectNames returns the full defined vocabulary, for tests and for the
// editor's palette. The map is not exported directly so nothing can mutate it.
func ObjectNames() map[int16]string {
	m := make(map[int16]string, len(objectNames))
	for k, v := range objectNames {
		m[k] = v
	}
	return m
}

func (o Object) String() string {
	if o.IsEmpty() {
		return "kObjectIsEmpty"
	}
	if n := ObjectName(o.What); n != "" {
		return n
	}
	return fmt.Sprintf("what(0x%02X, undefined)", uint16(o.What))
}

// ---------------------------------------------------------------- union views

// Blower is blowerType (data.a): vents, blowers, fans, candles and lift areas.
// `Vector` is a bitfield -- 1 up, 2 right, 4 down, 8 left -- and is only read for
// kLiftArea; the other fifteen have hard-coded directions. `Tall` is likewise
// only meaningful for kLiftArea (docs/analysis/house-format.md 6.1).
type Blower struct {
	TopLeft  Point
	Distance int16
	Initial  byte
	State    byte
	Vector   byte
	Tall     byte
}

// Vector bits of Blower.Vector, from the comment on blowerType in
// GliderPRO/Headers/GliderStructs.h:17.
const (
	VectorUp    byte = 1
	VectorRight byte = 2
	VectorDown  byte = 4
	VectorLeft  byte = 8
)

// Furniture is furnitureType (data.b): tables, shelves, obstacles. Unlike every
// other variant this one carries a full Rect, not a Point, which is why
// furniture can be stretched in the editor and a candle cannot.
type Furniture struct {
	Bounds Rect
	Pict   int16
}

// Bonus is bonusType (data.c): prizes, clocks, bands, grease, stars.
// Note that State precedes Initial here, the opposite order from Blower --
// a transposition that would compile fine and corrupt every prize in the house.
type Bonus struct {
	TopLeft Point
	Length  int16 // grease spill length
	Points  int16 // kInvisBonus score value
	State   byte
	Initial byte
}

// Transport is transportType (data.d): stairs, doors, windows, mailboxes.
// Where is the destination room and Who the destination object slot; the
// sentinels are asymmetric -- Where uses -1 because it is a short, Who uses 255
// because it is a byte (GliderPRO/Sources/Link.c:333-334).
type Transport struct {
	TopLeft Point
	Tall    int16
	Where   int16
	Who     byte
	Wide    byte
}

// UnlinkedWhere and UnlinkedWho are the two "not linked" sentinels.
const (
	UnlinkedWhere int16 = -1
	UnlinkedWho   byte  = 255
)

// Switch is switchType (data.e): light switches, triggers, sound triggers.
type Switch struct {
	TopLeft Point
	Delay   int16
	Where   int16
	Who     byte
	Type    byte
}

// Light is lightType (data.f).
type Light struct {
	TopLeft Point
	Length  int16
	Byte0   byte
	Byte1   byte
	Initial byte
	State   byte
}

// Appliance is applianceType (data.g). Height doubles as a PICT id for the
// kCustomPict object, and Byte0 precedes Delay -- the reverse of Enemy.
type Appliance struct {
	TopLeft Point
	Height  int16
	Byte0   byte
	Delay   byte
	Initial byte
	State   byte
}

// Enemy is enemyType (data.h). Delay precedes Byte0 here, the reverse of
// Appliance; the two structs are otherwise identical, so the order is the only
// thing keeping a balloon from being a toaster.
type Enemy struct {
	TopLeft Point
	Length  int16
	Delay   byte
	Byte0   byte
	Initial byte
	State   byte
}

// Clutter is clutterType (data.i): purely decorative, drawn and never touched.
// Same 10-byte shape as Furniture.
type Clutter struct {
	Bounds Rect
	Pict   int16
}

// TopLeft is the object's anchor, whichever variant it is. Every one of the nine
// begins with two shorts that are a vertical then a horizontal coordinate: seven
// declare a `Point topLeft` and the other two a `Rect bounds`, whose first two
// members are top and left. So the anchor can be read before the variant is
// known, which is what a room listing or a renderer's sort wants.
//
// It is meaningless for an empty slot or an undefined `what`; check Group first.
func (o Object) TopLeft() Point { return o.point() }

func be16(b []byte) int16       { return int16(binary.BigEndian.Uint16(b)) }
func putBE16(b []byte, v int16) { binary.BigEndian.PutUint16(b, uint16(v)) }

func (o Object) point() Point { return Point{V: be16(o.Data[0:]), H: be16(o.Data[2:])} }
func (o Object) rect() Rect {
	return Rect{Top: be16(o.Data[0:]), Left: be16(o.Data[2:]),
		Bottom: be16(o.Data[4:]), Right: be16(o.Data[6:])}
}

// Blower decodes data.a. The caller is responsible for checking Group(); reading
// the wrong view is not an error, it is exactly what a C union permits, and the
// bytes come back reinterpreted rather than zeroed.
func (o Object) Blower() Blower {
	return Blower{TopLeft: o.point(), Distance: be16(o.Data[4:]),
		Initial: o.Data[6], State: o.Data[7], Vector: o.Data[8], Tall: o.Data[9]}
}

func (o *Object) SetBlower(b Blower) {
	putBE16(o.Data[0:], b.TopLeft.V)
	putBE16(o.Data[2:], b.TopLeft.H)
	putBE16(o.Data[4:], b.Distance)
	o.Data[6], o.Data[7], o.Data[8], o.Data[9] = b.Initial, b.State, b.Vector, b.Tall
}

func (o Object) Furniture() Furniture {
	return Furniture{Bounds: o.rect(), Pict: be16(o.Data[8:])}
}

func (o *Object) SetFurniture(f Furniture) {
	putBE16(o.Data[0:], f.Bounds.Top)
	putBE16(o.Data[2:], f.Bounds.Left)
	putBE16(o.Data[4:], f.Bounds.Bottom)
	putBE16(o.Data[6:], f.Bounds.Right)
	putBE16(o.Data[8:], f.Pict)
}

func (o Object) Bonus() Bonus {
	return Bonus{TopLeft: o.point(), Length: be16(o.Data[4:]), Points: be16(o.Data[6:]),
		State: o.Data[8], Initial: o.Data[9]}
}

func (o *Object) SetBonus(c Bonus) {
	putBE16(o.Data[0:], c.TopLeft.V)
	putBE16(o.Data[2:], c.TopLeft.H)
	putBE16(o.Data[4:], c.Length)
	putBE16(o.Data[6:], c.Points)
	o.Data[8], o.Data[9] = c.State, c.Initial
}

func (o Object) Transport() Transport {
	return Transport{TopLeft: o.point(), Tall: be16(o.Data[4:]), Where: be16(o.Data[6:]),
		Who: o.Data[8], Wide: o.Data[9]}
}

func (o *Object) SetTransport(d Transport) {
	putBE16(o.Data[0:], d.TopLeft.V)
	putBE16(o.Data[2:], d.TopLeft.H)
	putBE16(o.Data[4:], d.Tall)
	putBE16(o.Data[6:], d.Where)
	o.Data[8], o.Data[9] = d.Who, d.Wide
}

func (o Object) Switch() Switch {
	return Switch{TopLeft: o.point(), Delay: be16(o.Data[4:]), Where: be16(o.Data[6:]),
		Who: o.Data[8], Type: o.Data[9]}
}

func (o *Object) SetSwitch(e Switch) {
	putBE16(o.Data[0:], e.TopLeft.V)
	putBE16(o.Data[2:], e.TopLeft.H)
	putBE16(o.Data[4:], e.Delay)
	putBE16(o.Data[6:], e.Where)
	o.Data[8], o.Data[9] = e.Who, e.Type
}

func (o Object) Light() Light {
	return Light{TopLeft: o.point(), Length: be16(o.Data[4:]),
		Byte0: o.Data[6], Byte1: o.Data[7], Initial: o.Data[8], State: o.Data[9]}
}

func (o *Object) SetLight(f Light) {
	putBE16(o.Data[0:], f.TopLeft.V)
	putBE16(o.Data[2:], f.TopLeft.H)
	putBE16(o.Data[4:], f.Length)
	o.Data[6], o.Data[7], o.Data[8], o.Data[9] = f.Byte0, f.Byte1, f.Initial, f.State
}

func (o Object) Appliance() Appliance {
	return Appliance{TopLeft: o.point(), Height: be16(o.Data[4:]),
		Byte0: o.Data[6], Delay: o.Data[7], Initial: o.Data[8], State: o.Data[9]}
}

func (o *Object) SetAppliance(g Appliance) {
	putBE16(o.Data[0:], g.TopLeft.V)
	putBE16(o.Data[2:], g.TopLeft.H)
	putBE16(o.Data[4:], g.Height)
	o.Data[6], o.Data[7], o.Data[8], o.Data[9] = g.Byte0, g.Delay, g.Initial, g.State
}

func (o Object) Enemy() Enemy {
	return Enemy{TopLeft: o.point(), Length: be16(o.Data[4:]),
		Delay: o.Data[6], Byte0: o.Data[7], Initial: o.Data[8], State: o.Data[9]}
}

func (o *Object) SetEnemy(h Enemy) {
	putBE16(o.Data[0:], h.TopLeft.V)
	putBE16(o.Data[2:], h.TopLeft.H)
	putBE16(o.Data[4:], h.Length)
	o.Data[6], o.Data[7], o.Data[8], o.Data[9] = h.Delay, h.Byte0, h.Initial, h.State
}

func (o Object) Clutter() Clutter {
	return Clutter{Bounds: o.rect(), Pict: be16(o.Data[8:])}
}

func (o *Object) SetClutter(i Clutter) {
	putBE16(o.Data[0:], i.Bounds.Top)
	putBE16(o.Data[2:], i.Bounds.Left)
	putBE16(o.Data[4:], i.Bounds.Bottom)
	putBE16(o.Data[6:], i.Bounds.Right)
	putBE16(o.Data[8:], i.Pict)
}
