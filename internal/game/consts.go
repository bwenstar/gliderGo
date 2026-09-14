package game

// The caps on the runtime tables (GliderDefines.h:250-266).
//
// Every one of these is a hard cap in the original: the array is fixed-size and
// the registrar returns -1 or silently drops when it is full. The port keeps that
// behaviour rather than growing a slice, because overflow is *observable* -- a
// 25th animated object in view is invisible, a 57th hot spot is inert -- and a
// port that quietly grew the table would play differently from the original in
// exactly the crowded rooms where the difference matters.
const (
	MaxRoomObs       = 24  // kMaxRoomObs: object slots per room
	MaxMasterObjects = 216 // kMaxMasterObjects: MaxRoomObs * 9
	MaxHotSpots      = 56  // kMaxHotSpots
	MaxSavedMaps     = 24  // kMaxSavedMaps
	MaxRubberBands   = 2   // kMaxRubberBands
	MaxGrease        = 16  // kMaxGrease
	MaxTempManholes  = 8   // kMaxTempManholes (Objects.c:12, not GliderDefines.h)
	MaxGarbageRects  = 48  // kMaxGarbageRects
)

// The nine local room slots (GliderDefines.h:217-225). The numbering is the
// original's and is load-bearing three times over: it indexes localNumbers[],
// it is what VerticalRoomOffset switches on, and DrawLocale's draw order is
// written in terms of it.
const (
	CentralRoom = iota
	NorthRoom
	NorthEastRoom
	EastRoom
	SouthEastRoom
	SouthRoom
	SouthWestRoom
	WestRoom
	NorthWestRoom
	NumLocalRooms
)

// The 28 hot-spot actions (GliderDefines.h:282-309). These are the whole
// vocabulary of "what happens when the glider touches this": CreateActiveRects
// assigns one to every rect it makes, and HandleHotSpotCollision is a switch over
// exactly this list.
//
// IgnoreIt is 0 and therefore the zero value, which is deliberate in the C too:
// a hot spot that has been consumed is left in the table with its isOn cleared
// rather than removed, so the indices other tables hold stay valid.
const (
	IgnoreIt          int16 = 0
	LiftIt            int16 = 1
	DropIt            int16 = 2
	PushItLeft        int16 = 3
	PushItRight       int16 = 4
	DissolveIt        int16 = 5
	RewardIt          int16 = 6
	MoveItUp          int16 = 7
	MoveItDown        int16 = 8
	SwitchIt          int16 = 9
	ShredIt           int16 = 10
	StrumIt           int16 = 11
	TriggerIt         int16 = 12
	BurnIt            int16 = 13
	SlideIt           int16 = 14
	TransportIt       int16 = 15
	IgnoreLeftWall    int16 = 16
	IgnoreRightWall   int16 = 17
	MailItLeft        int16 = 18
	MailItRight       int16 = 19
	DuctItDown        int16 = 20
	DuctItUp          int16 = 21
	MicrowaveIt       int16 = 22
	IgnoreGround      int16 = 23
	BounceIt          int16 = 24
	ChimeIt           int16 = 25
	WebIt             int16 = 26
	SoundIt           int16 = 27
	NumHotSpotActions       = 28
)

// actionNames is the C identifier for each action, for test failure messages and
// for the glidertool dump. A table of names is worth more than it looks: a hot-spot
// diff that says "kBurnIt where kLiftIt expected" localises a bug in the flame
// split instantly, where "13 != 1" does not.
var actionNames = [NumHotSpotActions]string{
	"kIgnoreIt", "kLiftIt", "kDropIt", "kPushItLeft", "kPushItRight",
	"kDissolveIt", "kRewardIt", "kMoveItUp", "kMoveItDown", "kSwitchIt",
	"kShredIt", "kStrumIt", "kTriggerIt", "kBurnIt", "kSlideIt",
	"kTransportIt", "kIgnoreLeftWall", "kIgnoreRightWall", "kMailItLeft",
	"kMailItRight", "kDuctItDown", "kDuctItUp", "kMicrowaveIt", "kIgnoreGround",
	"kBounceIt", "kChimeIt", "kWebIt", "kSoundIt",
}

// ActionName returns the C identifier for a hot-spot action.
func ActionName(action int16) string {
	if action < 0 || int(action) >= len(actionNames) {
		return "kUnknownAction"
	}
	return actionNames[action]
}

// The three SetObjectState actions (GliderDefines.h:441-443).
//
// Toggle is 0 and is what a switch sends. ForceOn and ForceOff are what a trigger
// and the two-player synchroniser send. Note that four of the nine families in
// SetObjectState ignore this argument entirely; see setstate.go.
const (
	Toggle   int16 = 0
	ForceOn  int16 = 1
	ForceOff int16 = 2
)

// Room geometry (GliderDefines.h:496-509, :548-549). Duplicated from
// internal/game/player/consts.go on purpose: these are two packages that both
// need the numbers, and importing the player package here would invert the
// dependency the Env interface exists to establish.
const (
	NumTiles        int16 = 8
	TileWide        int16 = 64
	TileHigh        int16 = 322
	RoomWide        int16 = 512
	VertLocalOffset int16 = 322

	FloorLimit       int16 = 312
	LeftWallLimit    int16 = 12
	RightWallLimit   int16 = 500
	NoLeftWallLimit  int16 = -24
	NoRightWallLimit int16 = 536

	FloorSupportTall int16 = 44
	GliderWide       int16 = 48
	GliderHigh       int16 = 20
)

// The geometry constants CreateActiveRects uses, all seven of them local to
// ObjectRects.c:11-17 and used nowhere else in the original.
//
// FloorColumnWide is the one to notice: a floor vent's lift column is **four
// pixels wide**, centred on the vent, not the width of the vent's artwork. A port
// that used the sprite width would make every vent in the game roughly six times
// easier to ride.
const (
	FloorColumnWide    int16 = 4
	CeilingColumnWide  int16 = 24
	FanColumnThick     int16 = 16
	FanColumnDown      int16 = 20
	DeadlyFlameHeight  int16 = 24
	StoolThick         int16 = 25
	ShredderActiveHigh int16 = 40
)

// Where a house's own resources begin, and where a custom background stops being
// assumed to be an interior (GliderDefines.h:245-246).
const (
	UserBackground     int16 = 3000
	UserStructureRange int16 = 3300
)

// The eighteen built-in room backgrounds (GliderDefines.h:227-244). The three
// DetermineRoomOpenings and the two floor/ceiling predicates switch on these, so
// the whole set is needed here even though internal/render declares its own copy
// for drawing.
const (
	SimpleRoom     int16 = 2000
	PaneledRoom    int16 = 2001
	Basement       int16 = 2002
	ChildsRoom     int16 = 2003
	AsianRoom      int16 = 2004
	UnfinishedRoom int16 = 2005
	SwingersRoom   int16 = 2006
	Bathroom       int16 = 2007
	Library        int16 = 2008
	Garden         int16 = 2009
	Skywalk        int16 = 2010
	Dirt           int16 = 2011
	Meadow         int16 = 2012
	Field          int16 = 2013
	Roof           int16 = 2014
	Sky            int16 = 2015
	Stratosphere   int16 = 2016
	Stars          int16 = 2017
)

// The 117 object type codes (GliderDefines.h:311-435), in the original's own
// spelling minus the k prefix. internal/house carries the same vocabulary as a
// name table for its text codec; this is the switchable form, and the two are
// pinned against each other by TestObjectCodesMatchHouse.
const (
	// Blowers -- data.a
	FloorVent     int16 = 0x01
	CeilingVent   int16 = 0x02
	FloorBlower   int16 = 0x03
	CeilingBlower int16 = 0x04
	SewerGrate    int16 = 0x05
	LeftFan       int16 = 0x06
	RightFan      int16 = 0x07
	Taper         int16 = 0x08
	Candle        int16 = 0x09
	Stubby        int16 = 0x0A
	Tiki          int16 = 0x0B
	BBQ           int16 = 0x0C
	InvisBlower   int16 = 0x0D
	GrecoVent     int16 = 0x0E
	SewerBlower   int16 = 0x0F
	LiftArea      int16 = 0x10

	// Furniture -- data.b
	Table         int16 = 0x11
	Shelf         int16 = 0x12
	Cabinet       int16 = 0x13
	FilingCabinet int16 = 0x14
	WasteBasket   int16 = 0x15
	MilkCrate     int16 = 0x16
	Counter       int16 = 0x17
	Dresser       int16 = 0x18
	DeckTable     int16 = 0x19
	Stool         int16 = 0x1A
	Trunk         int16 = 0x1B
	InvisObstacle int16 = 0x1C
	Manhole       int16 = 0x1D
	Books         int16 = 0x1E
	InvisBounce   int16 = 0x1F

	// Bonuses -- data.c
	RedClock    int16 = 0x21
	BlueClock   int16 = 0x22
	YellowClock int16 = 0x23
	Cuckoo      int16 = 0x24
	Paper       int16 = 0x25
	Battery     int16 = 0x26
	Bands       int16 = 0x27
	GreaseRt    int16 = 0x28
	GreaseLf    int16 = 0x29
	Foil        int16 = 0x2A
	InvisBonus  int16 = 0x2B
	Star        int16 = 0x2C
	Sparkle     int16 = 0x2D
	Helium      int16 = 0x2E
	Slider      int16 = 0x2F

	// Transports -- data.d
	UpStairs     int16 = 0x31
	DownStairs   int16 = 0x32
	MailboxLf    int16 = 0x33
	MailboxRt    int16 = 0x34
	FloorTrans   int16 = 0x35
	CeilingTrans int16 = 0x36
	DoorInLf     int16 = 0x37
	DoorInRt     int16 = 0x38
	DoorExRt     int16 = 0x39
	DoorExLf     int16 = 0x3A
	WindowInLf   int16 = 0x3B
	WindowInRt   int16 = 0x3C
	WindowExRt   int16 = 0x3D
	WindowExLf   int16 = 0x3E
	InvisTrans   int16 = 0x3F
	DeluxeTrans  int16 = 0x40

	// Switches -- data.e
	LightSwitch   int16 = 0x41
	MachineSwitch int16 = 0x42
	Thermostat    int16 = 0x43
	PowerSwitch   int16 = 0x44
	KnifeSwitch   int16 = 0x45
	InvisSwitch   int16 = 0x46
	Trigger       int16 = 0x47
	LgTrigger     int16 = 0x48
	SoundTrigger  int16 = 0x49

	// Lights -- data.f
	CeilingLight int16 = 0x51
	LightBulb    int16 = 0x52
	TableLamp    int16 = 0x53
	HipLamp      int16 = 0x54
	DecoLamp     int16 = 0x55
	Flourescent  int16 = 0x56
	TrackLight   int16 = 0x57
	InvisLight   int16 = 0x58

	// Appliances -- data.g
	Shredder    int16 = 0x61
	Toaster     int16 = 0x62
	MacPlus     int16 = 0x63
	Guitar      int16 = 0x64
	TV          int16 = 0x65
	Coffee      int16 = 0x66
	Outlet      int16 = 0x67
	VCR         int16 = 0x68
	Stereo      int16 = 0x69
	Microwave   int16 = 0x6A
	CinderBlock int16 = 0x6B
	FlowerBox   int16 = 0x6C
	CDs         int16 = 0x6D
	CustomPict  int16 = 0x6E

	// Enemies -- data.h
	Balloon  int16 = 0x71
	CopterLf int16 = 0x72
	CopterRt int16 = 0x73
	DartLf   int16 = 0x74
	DartRt   int16 = 0x75
	Ball     int16 = 0x76
	Drip     int16 = 0x77
	Fish     int16 = 0x78
	Cobweb   int16 = 0x79

	// Clutter -- data.i
	Ozma       int16 = 0x81
	Mirror     int16 = 0x82
	Mousehole  int16 = 0x83
	Fireplace  int16 = 0x84
	Flower     int16 = 0x85
	WallWindow int16 = 0x86
	Bear       int16 = 0x87
	Calendar   int16 = 0x88
	Vase1      int16 = 0x89
	Vase2      int16 = 0x8A
	Bulletin   int16 = 0x8B
	Cloud      int16 = 0x8C
	Faucet     int16 = 0x8D
	Rug        int16 = 0x8E
	Chimes     int16 = 0x8F
)
