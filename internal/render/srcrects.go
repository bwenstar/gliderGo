package render

// The object art atlas: InitSrcRects() from GliderPRO/Sources/StructuresInit.c
// and StructuresInit2.c, plus the classification of where each object's pixels
// actually come from.
//
// srcRects[] is a 256-entry array in the original, indexed by an object's `what`
// code, and only 116 of the entries are ever assigned. The other 140 are not
// spare capacity -- 28 fall inside the used code ranges and are read anyway by
// object types that are pure hit rectangles, and reading them yields the zero
// rect, which is what suppresses those objects' art. Modelling the table as a
// sparse map would therefore change behaviour, so it is an array here too.
//
// A srcRect serves two unrelated purposes depending on the object, which is the
// single most confusing thing about this table:
//
//   - For the `sheet` objects it is a genuine sub-rect of a shared sheet GWorld,
//     and blitting it is the whole of the object's art.
//   - For the `tmpl` objects it is only a size, used to lay out and hit-test the
//     object; the pixels come from separately named sub-rects (tableSrc, knobSrc,
//     lightSwitchSrc[], ...) and from procedural lines. srcRects[kTable] is 8
//     pixels tall because kTableThick is 8, while tableSrc is 22 pixels tall.
//
// Rects here are written in Rect field order. The extractor's tables and the
// original's QSetRect calls both use (left, top, right, bottom); this generator
// reorders them, so any hand edit must be made in QSetRect order in
// tools/extract_art.py and regenerated, not patched here.
//
// Generated from tools/extract_art.py's ATLAS/SHEETS/FLOWER_SRC. DO NOT EDIT.

// The object `what` codes (GliderPRO/Headers/GliderDefines.h). Spelled exactly
// as the original spells them, lower-case k prefix included, so that the draw
// switches below can be read against ObjectDrawAll.c line by line. house has the
// same codes as strings (house.ObjectName) but deliberately not as constants.
const (
	// Blower group -- data.a
	kFloorVent     = 0x01
	kCeilingVent   = 0x02
	kFloorBlower   = 0x03
	kCeilingBlower = 0x04
	kSewerGrate    = 0x05
	kLeftFan       = 0x06
	kRightFan      = 0x07
	kTaper         = 0x08
	kCandle        = 0x09
	kStubby        = 0x0A
	kTiki          = 0x0B
	kBBQ           = 0x0C
	kInvisBlower   = 0x0D
	kGrecoVent     = 0x0E
	kSewerBlower   = 0x0F
	kLiftArea      = 0x10

	// Furniture group -- data.b
	kTable         = 0x11
	kShelf         = 0x12
	kCabinet       = 0x13
	kFilingCabinet = 0x14
	kWasteBasket   = 0x15
	kMilkCrate     = 0x16
	kCounter       = 0x17
	kDresser       = 0x18
	kDeckTable     = 0x19
	kStool         = 0x1A
	kTrunk         = 0x1B
	kInvisObstacle = 0x1C
	kManhole       = 0x1D
	kBooks         = 0x1E
	kInvisBounce   = 0x1F

	// Bonus group -- data.c
	kRedClock    = 0x21
	kBlueClock   = 0x22
	kYellowClock = 0x23
	kCuckoo      = 0x24
	kPaper       = 0x25
	kBattery     = 0x26
	kBands       = 0x27
	kGreaseRt    = 0x28
	kGreaseLf    = 0x29
	kFoil        = 0x2A
	kInvisBonus  = 0x2B
	kStar        = 0x2C
	kSparkle     = 0x2D
	kHelium      = 0x2E
	kSlider      = 0x2F

	// Transport group -- data.d
	kUpStairs     = 0x31
	kDownStairs   = 0x32
	kMailboxLf    = 0x33
	kMailboxRt    = 0x34
	kFloorTrans   = 0x35
	kCeilingTrans = 0x36
	kDoorInLf     = 0x37
	kDoorInRt     = 0x38
	kDoorExRt     = 0x39
	kDoorExLf     = 0x3A
	kWindowInLf   = 0x3B
	kWindowInRt   = 0x3C
	kWindowExRt   = 0x3D
	kWindowExLf   = 0x3E
	kInvisTrans   = 0x3F
	kDeluxeTrans  = 0x40

	// Switch group -- data.e
	kLightSwitch   = 0x41
	kMachineSwitch = 0x42
	kThermostat    = 0x43
	kPowerSwitch   = 0x44
	kKnifeSwitch   = 0x45
	kInvisSwitch   = 0x46
	kTrigger       = 0x47
	kLgTrigger     = 0x48
	kSoundTrigger  = 0x49

	// Light group -- data.f
	kCeilingLight = 0x51
	kLightBulb    = 0x52
	kTableLamp    = 0x53
	kHipLamp      = 0x54
	kDecoLamp     = 0x55
	kFlourescent  = 0x56
	kTrackLight   = 0x57
	kInvisLight   = 0x58

	// Appliance group -- data.g
	kShredder    = 0x61
	kToaster     = 0x62
	kMacPlus     = 0x63
	kGuitar      = 0x64
	kTV          = 0x65
	kCoffee      = 0x66
	kOutlet      = 0x67
	kVCR         = 0x68
	kStereo      = 0x69
	kMicrowave   = 0x6A
	kCinderBlock = 0x6B
	kFlowerBox   = 0x6C
	kCDs         = 0x6D
	kCustomPict  = 0x6E

	// Enemy group -- data.h
	kBalloon  = 0x71
	kCopterLf = 0x72
	kCopterRt = 0x73
	kDartLf   = 0x74
	kDartRt   = 0x75
	kBall     = 0x76
	kDrip     = 0x77
	kFish     = 0x78
	kCobweb   = 0x79

	// Clutter group -- data.i
	kOzma       = 0x81
	kMirror     = 0x82
	kMousehole  = 0x83
	kFireplace  = 0x84
	kFlower     = 0x85
	kWallWindow = 0x86
	kBear       = 0x87
	kCalendar   = 0x88
	kVase1      = 0x89
	kVase2      = 0x8A
	kBulletin   = 0x8B
	kCloud      = 0x8C
	kFaucet     = 0x8D
	kRug        = 0x8E
	kChimes     = 0x8F
)

// artKind says where an object's pixels come from, which decides both how the
// asset loader finds them and which CopyBits/CopyMask variant moves them.
type artKind uint8

const (
	// artNone: never drawn. Invisible obstacles, triggers, lift areas. The
	// srcRect is a hit rectangle and there is no art anywhere.
	artNone artKind = iota

	// artSheet: srcRect is a sub-rect of the named shared sheet, blitted with
	// CopyMask through the sheet's own 1-bit mask PICT.
	artSheet

	// artTmpl: srcRect is a size template only. The draw helper composes the
	// object from named sub-rects of the named sheet and from lines.
	artTmpl

	// artKey: the object has its own PICT, drawn with CopyBits in `transparent`
	// mode -- white (palette index 0) is the colour key.
	artKey

	// artOpaque: own PICT, drawn with DrawPicture/srcCopy. Every pixel lands,
	// white included.
	artOpaque

	// artPair: own PICT plus its own 1-bit mask PICT, composited through a
	// temporary GWorld pair by CopyMask. Where a mask exists it is
	// authoritative: several of these PICTs contain opaque white that a colour
	// key would wrongly drop.
	artPair

	// artProc: drawn entirely from rects and lines. kCounter, kMirror and
	// kWallWindow have no pixels to extract at all.
	artProc
)

// atlasEntry is one initialised srcRects[] slot together with its provenance.
type atlasEntry struct {
	what  int16
	kind  artKind
	sheet string // artSheet/artTmpl: the sheet the art or sub-rects live in
	pict  int16  // artKey/artOpaque/artPair: the art PICT id, 0 otherwise
	src   Rect
}

var atlas = []atlasEntry{
	{kFloorVent, artSheet, "blower", 0, Rect{Top: 0, Left: 0, Bottom: 11, Right: 48}},
	{kCeilingVent, artSheet, "blower", 0, Rect{Top: 11, Left: 0, Bottom: 22, Right: 48}},
	{kFloorBlower, artSheet, "blower", 0, Rect{Top: 22, Left: 0, Bottom: 37, Right: 48}},
	{kCeilingBlower, artSheet, "blower", 0, Rect{Top: 37, Left: 0, Bottom: 52, Right: 48}},
	{kSewerGrate, artSheet, "blower", 0, Rect{Top: 52, Left: 0, Bottom: 69, Right: 48}},
	{kLeftFan, artSheet, "blower", 0, Rect{Top: 69, Left: 0, Bottom: 124, Right: 40}}, // tikiFlame[] overlays x40-48
	{kRightFan, artSheet, "blower", 0, Rect{Top: 124, Left: 0, Bottom: 179, Right: 40}},
	{kTaper, artSheet, "blower", 0, Rect{Top: 209, Left: 0, Bottom: 268, Right: 20}},  // + flame[0..4] 16x15 stride 15 @(32,179)
	{kCandle, artSheet, "blower", 0, Rect{Top: 179, Left: 0, Bottom: 209, Right: 32}}, // + flame[0..4] 16x15 stride 15 @(32,179)
	{kStubby, artSheet, "blower", 0, Rect{Top: 268, Left: 0, Bottom: 304, Right: 20}}, // + flame[0..4] 16x15 stride 15 @(32,179)
	{kTiki, artSheet, "blower", 0, Rect{Top: 268, Left: 21, Bottom: 296, Right: 48}},  // + tikiFlame[0..4] 8x10 stride 10 @(40,69); pole drawn procedurally
	{kBBQ, artKey, "", 3988, Rect{Top: 0, Left: 0, Bottom: 33, Right: 64}},            // + coals[0..3] 32x9 stride 9 @(0,304) from blower sheet
	{kInvisBlower, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 24, Right: 24}},      // hit/size template only
	{kGrecoVent, artSheet, "blower", 0, Rect{Top: 340, Left: 0, Bottom: 358, Right: 48}},
	{kSewerBlower, artSheet, "blower", 0, Rect{Top: 390, Left: 0, Bottom: 402, Right: 32}},
	{kLiftArea, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 32, Right: 64}},         // size overridden by data.a.distance/tall
	{kTable, artTmpl, "furniture", 0, Rect{Top: 0, Left: 0, Bottom: 8, Right: 64}},    // art = tableSrc 64x22 @(0,0); rect is kTableThick=8 thick
	{kShelf, artTmpl, "furniture", 0, Rect{Top: 0, Left: 0, Bottom: 6, Right: 64}},    // art = shelfSrc 16x21 @(0,22); rect is kShelfThick=6 thick
	{kCabinet, artTmpl, "furniture", 0, Rect{Top: 0, Left: 0, Bottom: 64, Right: 64}}, // art = hingeSrc 4x16 @(16,22), handleSrc 4x21 @(20,22)
	{kFilingCabinet, artOpaque, "", 3995, Rect{Top: 0, Left: 0, Bottom: 107, Right: 74}},
	{kWasteBasket, artSheet, "furniture", 0, Rect{Top: 43, Left: 0, Bottom: 104, Right: 64}},
	{kMilkCrate, artSheet, "furniture", 0, Rect{Top: 104, Left: 0, Bottom: 162, Right: 64}},
	{kCounter, artProc, "", 0, Rect{Top: 0, Left: 0, Bottom: 64, Right: 128}},           // no PICT pixels at all
	{kDresser, artTmpl, "furniture", 0, Rect{Top: 0, Left: 0, Bottom: 64, Right: 128}},  // art = knobSrc 8x8 @(24,22) srcCopy, leftFootSrc 16x16 @(32,22), rightFootSrc 16x16 @(48,22)
	{kDeckTable, artTmpl, "furniture", 0, Rect{Top: 0, Left: 0, Bottom: 8, Right: 64}},  // art = deckSrc 64x21 @(0,162)
	{kStool, artSheet, "furniture", 0, Rect{Top: 183, Left: 0, Bottom: 221, Right: 48}}, // + procedural legs to kStoolBase 304
	{kTrunk, artKey, "", 3987, Rect{Top: 0, Left: 0, Bottom: 80, Right: 144}},
	{kInvisObstacle, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 64, Right: 64}}, // hit/size template only
	{kManhole, artKey, "", 3967, Rect{Top: 0, Left: 0, Bottom: 22, Right: 123}},
	{kBooks, artKey, "", 3964, Rect{Top: 0, Left: 0, Bottom: 51, Right: 64}},
	{kInvisBounce, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 64, Right: 64}},        // hit/size template only
	{kRedClock, artSheet, "bonus", 0, Rect{Top: 0, Left: 0, Bottom: 17, Right: 28}},     // + digits[0..10] 4x6 stride 6 @(28,0)
	{kBlueClock, artSheet, "bonus", 0, Rect{Top: 17, Left: 0, Bottom: 42, Right: 28}},   // + ColorLine hands
	{kYellowClock, artSheet, "bonus", 0, Rect{Top: 42, Left: 0, Bottom: 70, Right: 28}}, // + ColorLine hands
	{kCuckoo, artSheet, "bonus", 0, Rect{Top: 148, Left: 0, Bottom: 228, Right: 40}},    // + pendulumSrc[0..2] 32x28 stride 28 @(56,186)
	{kPaper, artSheet, "bonus", 0, Rect{Top: 127, Left: 0, Bottom: 148, Right: 48}},
	{kBattery, artSheet, "bonus", 0, Rect{Top: 0, Left: 32, Bottom: 25, Right: 48}},
	{kBands, artSheet, "bonus", 0, Rect{Top: 70, Left: 20, Bottom: 93, Right: 48}},     // bandRects[0..2] 16x6 stride 6 in bandsSrcMap 4007/5007 are the in-flight bands
	{kGreaseRt, artSheet, "bonus", 0, Rect{Top: 243, Left: 0, Bottom: 270, Right: 32}}, // frames greaseSrcRt[0..3] 32x27 @(0,243),(0,270),(0,297),(32,297)
	{kGreaseLf, artSheet, "bonus", 0, Rect{Top: 324, Left: 0, Bottom: 351, Right: 32}}, // frames greaseSrcLf[0..3] 32x27 @(0,324),(32,324),(0,351),(32,351)
	{kFoil, artSheet, "bonus", 0, Rect{Top: 228, Left: 0, Bottom: 243, Right: 55}},
	{kInvisBonus, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 24, Right: 24}},     // hit/size template only
	{kStar, artSheet, "bonus", 0, Rect{Top: 0, Left: 48, Bottom: 31, Right: 80}},    // starSrc[0..5] 32x31 stride 31 @(48,0)
	{kSparkle, artSheet, "bonus", 0, Rect{Top: 70, Left: 0, Bottom: 89, Right: 20}}, // SparkleSrc[0..4] 20x19; frames 2,3,4 @(0,70),(0,89),(0,108); [0]=[4], [1]=[3]
	{kHelium, artSheet, "bonus", 0, Rect{Top: 270, Left: 32, Bottom: 286, Right: 88}},
	{kSlider, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 16, Right: 64}}, // hit/size template only
	{kUpStairs, artKey, "", 3997, Rect{Top: 0, Left: 0, Bottom: 267, Right: 160}},
	{kDownStairs, artOpaque, "", 3996, Rect{Top: 0, Left: 0, Bottom: 267, Right: 160}},
	{kMailboxLf, artPair, "", 3986, Rect{Top: 0, Left: 0, Bottom: 80, Right: 94}}, // anchored to kMailboxBase 296
	{kMailboxRt, artPair, "", 3985, Rect{Top: 0, Left: 0, Bottom: 80, Right: 94}}, // anchored to kMailboxBase 296
	{kFloorTrans, artSheet, "trans", 0, Rect{Top: 1, Left: 0, Bottom: 16, Right: 56}},
	{kCeilingTrans, artSheet, "trans", 0, Rect{Top: 16, Left: 0, Bottom: 31, Right: 56}},
	{kDoorInLf, artKey, "", 3984, Rect{Top: 0, Left: 0, Bottom: 322, Right: 144}},
	{kDoorInRt, artKey, "", 3983, Rect{Top: 0, Left: 0, Bottom: 322, Right: 144}},
	{kDoorExRt, artOpaque, "", 3982, Rect{Top: 0, Left: 0, Bottom: 322, Right: 16}},
	{kDoorExLf, artOpaque, "", 3981, Rect{Top: 0, Left: 0, Bottom: 322, Right: 16}},
	{kWindowInLf, artKey, "", 3980, Rect{Top: 0, Left: 0, Bottom: 170, Right: 20}},
	{kWindowInRt, artKey, "", 3979, Rect{Top: 0, Left: 0, Bottom: 170, Right: 20}},
	{kWindowExRt, artOpaque, "", 3977, Rect{Top: 0, Left: 0, Bottom: 170, Right: 16}},
	{kWindowExLf, artOpaque, "", 3978, Rect{Top: 0, Left: 0, Bottom: 170, Right: 16}},
	{kInvisTrans, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 32, Right: 64}},           // hit/size template only
	{kDeluxeTrans, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 64, Right: 64}},          // hit/size template only
	{kLightSwitch, artTmpl, "switch", 0, Rect{Top: 0, Left: 0, Bottom: 24, Right: 15}},    // art = lightSwitchSrc[0] 15x24 @(0,0) on / [1] @(16,0) off
	{kMachineSwitch, artTmpl, "switch", 0, Rect{Top: 48, Left: 0, Bottom: 72, Right: 16}}, // art = machineSwitchSrc[0] 16x24 @(0,24) / [1] @(16,24); rect y-offset 48 disagrees
	{kThermostat, artTmpl, "switch", 0, Rect{Top: 48, Left: 0, Bottom: 72, Right: 15}},    // art = thermostatSrc[0] 15x24 @(0,48) / [1] @(16,48)
	{kPowerSwitch, artTmpl, "switch", 0, Rect{Top: 72, Left: 0, Bottom: 80, Right: 8}},    // art = powerSrc[0] 8x8 @(0,72) / [1] @(8,72)
	{kKnifeSwitch, artTmpl, "switch", 0, Rect{Top: 80, Left: 0, Bottom: 104, Right: 16}},  // art = knifeSwitchSrc[0] 16x24 @(0,80) / [1] @(16,80)
	{kInvisSwitch, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 12, Right: 12}},          // hit/size template only
	{kTrigger, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 12, Right: 12}},              // hit/size template only
	{kLgTrigger, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 48, Right: 48}},            // hit/size template only
	{kSoundTrigger, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 32, Right: 32}},         // hit/size template only
	{kCeilingLight, artSheet, "light", 0, Rect{Top: 0, Left: 0, Bottom: 20, Right: 64}},
	{kLightBulb, artSheet, "light", 0, Rect{Top: 20, Left: 0, Bottom: 48, Right: 16}},
	{kTableLamp, artSheet, "light", 0, Rect{Top: 20, Left: 16, Bottom: 90, Right: 64}},
	{kHipLamp, artKey, "", 3994, Rect{Top: 0, Left: 0, Bottom: 276, Right: 72}},
	{kDecoLamp, artKey, "", 3993, Rect{Top: 0, Left: 0, Bottom: 212, Right: 64}},
	{kFlourescent, artTmpl, "light", 0, Rect{Top: 0, Left: 0, Bottom: 12, Right: 64}},    // art = flourescentSrc1 16x12 @(0,78) off / flourescentSrc2 @(0,90) on, tiled
	{kTrackLight, artTmpl, "light", 0, Rect{Top: 0, Left: 0, Bottom: 24, Right: 64}},     // art = trackLightSrc[0..2] 24x24 stride 24 in x @(0,102); kTrackLightSpacing 64
	{kInvisLight, artNone, "", 0, Rect{Top: 0, Left: 0, Bottom: 16, Right: 16}},          // hit/size template only
	{kShredder, artSheet, "appliance", 0, Rect{Top: 0, Left: 0, Bottom: 22, Right: 73}},  // shredSrcMap 4010/5010 40x35 holds the shredded-paper animation
	{kToaster, artSheet, "appliance", 0, Rect{Top: 22, Left: 0, Bottom: 49, Right: 48}},  // toastSrcMap 4009/5009 32x174 holds BreadSrc[0..5] 32x29 stride 29
	{kMacPlus, artSheet, "appliance", 0, Rect{Top: 49, Left: 0, Bottom: 107, Right: 48}}, // + PlusScreen1 32x22 @(48,127) off / PlusScreen2 @(48,149) on, srcCopy at +10,+7
	{kGuitar, artKey, "", 3991, Rect{Top: 0, Left: 0, Bottom: 172, Right: 64}},
	{kTV, artPair, "", 3992, Rect{Top: 0, Left: 0, Bottom: 77, Right: 92}},               // + TVScreen1 64x49 @(0,171) off / TVScreen2 @(0,220) on, from applianceSrcMap
	{kCoffee, artSheet, "appliance", 0, Rect{Top: 107, Left: 0, Bottom: 171, Right: 43}}, // + CoffeeLight1 8x4 @(72,171) off / CoffeeLight2 @(72,175) on, srcCopy at +32,+57
	{kOutlet, artSheet, "appliance", 0, Rect{Top: 22, Left: 64, Bottom: 46, Right: 80}},  // OutletSrc[0..3] 16x24 stride 24 @(64,22) is the spark animation
	{kVCR, artPair, "", 3990, Rect{Top: 0, Left: 0, Bottom: 22, Right: 96}},              // + VCRTime1 16x4 @(64,179) / VCRTime2 @(64,183) from applianceSrcMap
	{kStereo, artPair, "", 3989, Rect{Top: 0, Left: 0, Bottom: 53, Right: 128}},          // + StereoLight1 4x1 @(68,171) / StereoLight2 @(68,172)
	{kMicrowave, artPair, "", 3971, Rect{Top: 0, Left: 0, Bottom: 59, Right: 92}},        // + MicroOff 16x35 @(64,187) / MicroOn @(64,222)
	{kCinderBlock, artKey, "", 3960, Rect{Top: 0, Left: 0, Bottom: 62, Right: 40}},
	{kFlowerBox, artKey, "", 3959, Rect{Top: 0, Left: 0, Bottom: 32, Right: 80}},
	{kCDs, artSheet, "appliance", 0, Rect{Top: 22, Left: 48, Bottom: 52, Right: 64}},
	{kCustomPict, artKey, "", 0, Rect{Top: 0, Left: 0, Bottom: 34, Right: 72}},       // default art PICT 10000 is exactly 72x34
	{kBalloon, artSheet, "balloon", 0, Rect{Top: 0, Left: 0, Bottom: 30, Right: 24}}, // balloonSrc[0..7] 24x30 stride 30
	{kCopterLf, artSheet, "copter", 0, Rect{Top: 0, Left: 0, Bottom: 30, Right: 32}}, // copterSrc[0..9] 32x30 stride 30
	{kCopterRt, artSheet, "copter", 0, Rect{Top: 0, Left: 0, Bottom: 30, Right: 32}}, // copterSrc[0..9] 32x30 stride 30
	{kDartLf, artSheet, "dart", 0, Rect{Top: 0, Left: 0, Bottom: 19, Right: 64}},     // dartSrc[0..3] 64x19 stride 19
	{kDartRt, artSheet, "dart", 0, Rect{Top: 0, Left: 0, Bottom: 19, Right: 64}},     // dartSrc[0..3] 64x19 stride 19
	{kBall, artSheet, "ball", 0, Rect{Top: 0, Left: 0, Bottom: 32, Right: 32}},       // ballSrc[0..1] 32x32 stride 32
	{kDrip, artSheet, "drip", 0, Rect{Top: 0, Left: 0, Bottom: 12, Right: 16}},       // static draw uses DripSrc[3] @(0,36), NOT srcRects[kDrip]; DripSrc[0..5] 16x12 stride 12
	{kFish, artSheet, "enemy", 0, Rect{Top: 0, Left: 0, Bottom: 33, Right: 36}},      // static draw reads enemySrcMap 4016/5016; the 8-frame fishSrcMap 4017/5017 16x16 stride 16 is animation-only
	{kCobweb, artPair, "", 3958, Rect{Top: 0, Left: 0, Bottom: 45, Right: 54}},
	{kOzma, artOpaque, "", 3975, Rect{Top: 0, Left: 0, Bottom: 92, Right: 102}},
	{kMirror, artProc, "", 0, Rect{Top: 0, Left: 0, Bottom: 64, Right: 64}}, // four ColorFrameRect insets, no PICT
	{kMousehole, artSheet, "clutter", 0, Rect{Top: 0, Left: 0, Bottom: 11, Right: 10}},
	{kFireplace, artKey, "", 3973, Rect{Top: 0, Left: 0, Bottom: 142, Right: 180}},
	{kWallWindow, artProc, "", 0, Rect{Top: 0, Left: 0, Bottom: 80, Right: 64}}, // kWindowSillThick 7, no PICT
	{kBear, artKey, "", 3972, Rect{Top: 0, Left: 0, Bottom: 58, Right: 56}},
	{kCalendar, artOpaque, "", 3970, Rect{Top: 0, Left: 0, Bottom: 92, Right: 63}}, // + month name from STR# 1005 at +((64-w)/2), +55
	{kVase1, artKey, "", 3969, Rect{Top: 0, Left: 0, Bottom: 45, Right: 36}},
	{kVase2, artKey, "", 3968, Rect{Top: 0, Left: 0, Bottom: 57, Right: 35}},
	{kBulletin, artOpaque, "", 3966, Rect{Top: 0, Left: 0, Bottom: 58, Right: 80}},
	{kCloud, artPair, "", 3965, Rect{Top: 0, Left: 0, Bottom: 30, Right: 128}},
	{kFaucet, artSheet, "clutter", 0, Rect{Top: 51, Left: 0, Bottom: 69, Right: 56}},
	{kRug, artKey, "", 3962, Rect{Top: 0, Left: 0, Bottom: 18, Right: 144}},
	{kChimes, artKey, "", 3961, Rect{Top: 0, Left: 0, Bottom: 74, Right: 28}},
}

// srcRects is the original's srcRects[256]. Unassigned slots are the zero rect,
// exactly as the original's uninitialised static array is.
var srcRects [256]Rect

// objArt is srcRects' companion: the kind and sheet for each assigned slot.
var objArt [256]atlasEntry

func init() {
	for _, e := range atlas {
		srcRects[e.what] = e.src
		objArt[e.what] = e
	}
}

// SrcRect returns srcRects[what], or the zero rect for an unassigned code --
// including for out-of-range codes, where the original would read out of bounds.
func SrcRect(what int16) Rect {
	if what < 0 || what > 255 {
		return Rect{}
	}
	return srcRects[what]
}

// flowerSrc is StructuresInit2.c:72-88. kFlower is the one object with no
// srcRects entry at all: these six rects of the clutter sheet are simultaneously
// its art and its size, selected by data.i.pict, which is why the six extracted
// PNGs all differ in size.
var flowerSrc = [6]Rect{
	{Top: 23, Left: 0, Bottom: 51, Right: 10},
	{Top: 16, Left: 10, Bottom: 51, Right: 34},
	{Top: 16, Left: 34, Bottom: 51, Right: 68},
	{Top: 14, Left: 68, Bottom: 37, Right: 95},
	{Top: 37, Left: 68, Bottom: 51, Right: 95},
	{Top: 0, Left: 95, Bottom: 51, Right: 127},
}

// ---------------------------------------------------------------------------
// The named sub-rects. These are the art the `tmpl` objects and the animation
// helpers actually blit, and they are the reason srcRects alone cannot render a
// room. All were verified against the QSetRect calls in StructuresInit.c rather
// than inferred from the extracted PNGs.
// ---------------------------------------------------------------------------

// furnitureSrcMap, 64x278. Note the PICT is only 64x221; rows 221..277 are
// unwritten GWorld (white) and no sub-rect reads there.
var (
	tableSrc     = Rect{Top: 0, Left: 0, Bottom: 22, Right: 64}    // the 22-pixel-tall top a kTable is drawn from, over an 8-pixel-thick rect
	shelfSrc     = Rect{Top: 22, Left: 0, Bottom: 43, Right: 16}   // tiled across a kShelf
	hingeSrc     = Rect{Top: 22, Left: 16, Bottom: 38, Right: 20}  // kCabinet door hinges
	handleSrc    = Rect{Top: 22, Left: 20, Bottom: 43, Right: 24}  // kCabinet door handle
	knobSrc      = Rect{Top: 22, Left: 24, Bottom: 30, Right: 32}  // kDresser drawer knobs, blitted srcCopy
	leftFootSrc  = Rect{Top: 22, Left: 32, Bottom: 38, Right: 48}  // kDresser left foot
	rightFootSrc = Rect{Top: 22, Left: 48, Bottom: 38, Right: 64}  // kDresser right foot
	deckSrc      = Rect{Top: 162, Left: 0, Bottom: 183, Right: 64} // tiled across a kDeckTable
)

// blowerSrcMap, 48x402. The three animation strips live to the right of and
// below the objects that use them.
var (
	// flame[0..4], 16x15, stride 15 in v. Shared by kTaper, kCandle and kStubby.
	flameSrc = [5]Rect{
		{Top: 179, Left: 32, Bottom: 194, Right: 48},
		{Top: 194, Left: 32, Bottom: 209, Right: 48},
		{Top: 209, Left: 32, Bottom: 224, Right: 48},
		{Top: 224, Left: 32, Bottom: 239, Right: 48},
		{Top: 239, Left: 32, Bottom: 254, Right: 48},
	}

	// tikiFlame[0..4], 8x10, stride 10 in v. kTiki only; its pole is lines.
	tikiFlameSrc = [5]Rect{
		{Top: 69, Left: 40, Bottom: 79, Right: 48},
		{Top: 79, Left: 40, Bottom: 89, Right: 48},
		{Top: 89, Left: 40, Bottom: 99, Right: 48},
		{Top: 99, Left: 40, Bottom: 109, Right: 48},
		{Top: 109, Left: 40, Bottom: 119, Right: 48},
	}

	// coals[0..3], 32x9, stride 9 in v. Overlaid on kBBQ, whose body is PICT 3988.
	coalsSrc = [4]Rect{
		{Top: 304, Left: 0, Bottom: 313, Right: 32},
		{Top: 313, Left: 0, Bottom: 322, Right: 32},
		{Top: 322, Left: 0, Bottom: 331, Right: 32},
		{Top: 331, Left: 0, Bottom: 340, Right: 32},
	}
)

// bonusSrcMap, 88x378.
var (
	// digits[0..10], 4x6, stride 6 in v. The eleventh is the blank/colon cell.
	digitSrc = [11]Rect{
		{Top: 0, Left: 28, Bottom: 6, Right: 32},
		{Top: 6, Left: 28, Bottom: 12, Right: 32},
		{Top: 12, Left: 28, Bottom: 18, Right: 32},
		{Top: 18, Left: 28, Bottom: 24, Right: 32},
		{Top: 24, Left: 28, Bottom: 30, Right: 32},
		{Top: 30, Left: 28, Bottom: 36, Right: 32},
		{Top: 36, Left: 28, Bottom: 42, Right: 32},
		{Top: 42, Left: 28, Bottom: 48, Right: 32},
		{Top: 48, Left: 28, Bottom: 54, Right: 32},
		{Top: 54, Left: 28, Bottom: 60, Right: 32},
		{Top: 60, Left: 28, Bottom: 66, Right: 32},
	}

	// pendulumSrc[0..2], 32x28, stride 28 in v. kCuckoo.
	pendulumSrc = [3]Rect{
		{Top: 186, Left: 56, Bottom: 214, Right: 88},
		{Top: 214, Left: 56, Bottom: 242, Right: 88},
		{Top: 242, Left: 56, Bottom: 270, Right: 88},
	}

	// starSrc[0..5], 32x31, stride 31 in v. kStar spins through all six; the
	// static draw uses frame 0, which is also srcRects[kStar].
	starSrc = [6]Rect{
		{Top: 0, Left: 48, Bottom: 31, Right: 80},
		{Top: 31, Left: 48, Bottom: 62, Right: 80},
		{Top: 62, Left: 48, Bottom: 93, Right: 80},
		{Top: 93, Left: 48, Bottom: 124, Right: 80},
		{Top: 124, Left: 48, Bottom: 155, Right: 80},
		{Top: 155, Left: 48, Bottom: 186, Right: 80},
	}

	// greaseSrcRt[0..3] and greaseSrcLf[0..3], 32x27. Laid out irregularly --
	// two down one column then two across, and the left and right sets do not
	// mirror each other's layout, so these are transcribed, not computed.
	greaseSrcRt = [4]Rect{
		{Top: 243, Left: 0, Bottom: 270, Right: 32},
		{Top: 270, Left: 0, Bottom: 297, Right: 32},
		{Top: 297, Left: 0, Bottom: 324, Right: 32},
		{Top: 297, Left: 32, Bottom: 324, Right: 64},
	}
	greaseSrcLf = [4]Rect{
		{Top: 324, Left: 0, Bottom: 351, Right: 32},
		{Top: 324, Left: 32, Bottom: 351, Right: 64},
		{Top: 351, Left: 0, Bottom: 378, Right: 32},
		{Top: 351, Left: 32, Bottom: 378, Right: 64},
	}

	// SparkleSrc[0..4], 20x19. Only three distinct frames exist: the strip is
	// played out and back, so [0] aliases [4] and [1] aliases [3].
	SparkleSrc = [5]Rect{
		{Top: 108, Left: 0, Bottom: 127, Right: 20},
		{Top: 89, Left: 0, Bottom: 108, Right: 20},
		{Top: 70, Left: 0, Bottom: 89, Right: 20},
		{Top: 89, Left: 0, Bottom: 108, Right: 20},
		{Top: 108, Left: 0, Bottom: 127, Right: 20},
	}
)

// switchSrcMap, 32x104. Column 0 is the "on"/up art, column 16 the "off"/down
// art, except powerSrc whose two 8x8 cells sit side by side at y=72.
//
// srcRects[kMachineSwitch] and srcRects[kThermostat] are both at y=48 while
// machineSwitchSrc is at y=24. That disagreement is in the original and it is
// harmless: the srcRect is only a size template for these two.
var (
	lightSwitchSrc   = [2]Rect{Rect{Top: 0, Left: 0, Bottom: 24, Right: 15}, Rect{Top: 0, Left: 16, Bottom: 24, Right: 31}}
	machineSwitchSrc = [2]Rect{Rect{Top: 24, Left: 0, Bottom: 48, Right: 16}, Rect{Top: 24, Left: 16, Bottom: 48, Right: 32}}
	thermostatSrc    = [2]Rect{Rect{Top: 48, Left: 0, Bottom: 72, Right: 15}, Rect{Top: 48, Left: 16, Bottom: 72, Right: 31}}
	powerSrc         = [2]Rect{Rect{Top: 72, Left: 0, Bottom: 80, Right: 8}, Rect{Top: 72, Left: 8, Bottom: 80, Right: 16}}
	knifeSwitchSrc   = [2]Rect{Rect{Top: 80, Left: 0, Bottom: 104, Right: 16}, Rect{Top: 80, Left: 16, Bottom: 104, Right: 32}}
)

// lightSrcMap, 72x126.
var (
	flourescentSrc1 = Rect{Top: 78, Left: 0, Bottom: 90, Right: 16}  // off
	flourescentSrc2 = Rect{Top: 90, Left: 0, Bottom: 102, Right: 16} // on, tiled across the fixture

	// trackLightSrc[0..2], 24x24, stride 24 in h: off, on, and the lit cone.
	trackLightSrc = [3]Rect{
		{Top: 102, Left: 0, Bottom: 126, Right: 24},
		{Top: 102, Left: 24, Bottom: 126, Right: 48},
		{Top: 102, Left: 48, Bottom: 126, Right: 72},
	}
)

// applianceSrcMap, 80x269. Every one of these is an overlay blitted onto an
// appliance that has already been drawn, and the on/off pairs are adjacent so
// that a boolean state selects between them.
var (
	PlusScreen1  = Rect{Top: 127, Left: 48, Bottom: 149, Right: 80} // kMacPlus screen off, srcCopy at +10,+7
	PlusScreen2  = Rect{Top: 149, Left: 48, Bottom: 171, Right: 80} // kMacPlus screen on
	TVScreen1    = Rect{Top: 171, Left: 0, Bottom: 220, Right: 64}  // kTV screen off
	TVScreen2    = Rect{Top: 220, Left: 0, Bottom: 269, Right: 64}  // kTV screen on
	CoffeeLight1 = Rect{Top: 171, Left: 72, Bottom: 175, Right: 80} // kCoffee lamp off, srcCopy at +32,+57
	CoffeeLight2 = Rect{Top: 175, Left: 72, Bottom: 179, Right: 80} // kCoffee lamp on
	VCRTime1     = Rect{Top: 179, Left: 64, Bottom: 183, Right: 80} // kVCR clock blank
	VCRTime2     = Rect{Top: 183, Left: 64, Bottom: 187, Right: 80} // kVCR clock 12:00
	StereoLight1 = Rect{Top: 171, Left: 68, Bottom: 172, Right: 72} // kStereo LED off -- four pixels by one
	StereoLight2 = Rect{Top: 172, Left: 68, Bottom: 173, Right: 72} // kStereo LED on
	MicroOff     = Rect{Top: 187, Left: 64, Bottom: 222, Right: 80} // kMicrowave door closed
	MicroOn      = Rect{Top: 222, Left: 64, Bottom: 257, Right: 80} // kMicrowave door open

	// OutletSrc[0..3], 16x24, stride 24 in v. Frame 0 is the unlit outlet and
	// is also srcRects[kOutlet]; 1..3 are the spark.
	OutletSrc = [4]Rect{
		{Top: 22, Left: 64, Bottom: 46, Right: 80},
		{Top: 46, Left: 64, Bottom: 70, Right: 80},
		{Top: 70, Left: 64, Bottom: 94, Right: 80},
		{Top: 94, Left: 64, Bottom: 118, Right: 80},
	}
)

// toastSrcMap, 32x174 -- a strip, not a sheet: nothing in srcRects indexes it.
// BreadSrc[0..5], 32x29, stride 29 in v, is the slice rising out of a kToaster.
var BreadSrc = [6]Rect{
	{Top: 0, Left: 0, Bottom: 29, Right: 32},
	{Top: 29, Left: 0, Bottom: 58, Right: 32},
	{Top: 58, Left: 0, Bottom: 87, Right: 32},
	{Top: 87, Left: 0, Bottom: 116, Right: 32},
	{Top: 116, Left: 0, Bottom: 145, Right: 32},
	{Top: 145, Left: 0, Bottom: 174, Right: 32},
}

// dripSrcMap, 16x72. DripSrc[0..5], 16x12, stride 12 in v. The static room draw
// uses frame 3 -- the hanging drop -- and not srcRects[kDrip], which is frame 0.
var DripSrc = [6]Rect{
	{Top: 0, Left: 0, Bottom: 12, Right: 16},
	{Top: 12, Left: 0, Bottom: 24, Right: 16},
	{Top: 24, Left: 0, Bottom: 36, Right: 16},
	{Top: 36, Left: 0, Bottom: 48, Right: 16},
	{Top: 48, Left: 0, Bottom: 60, Right: 16},
	{Top: 60, Left: 0, Bottom: 72, Right: 16},
}

// The five enemy strips and the flying-points strip: art that only a *moving*
// object reads, which is why none of it was needed before 1.5c.
//
// Every one is a single column with a constant stride, from the loops at
// StructuresInit.c:688-720 and :576-580. Written out rather than computed for the
// same reason BreadSrc and DripSrc are: TestStripRectsTileTheirSheet checks each
// array against its sheet's declared bounds, and a transcribed table is what that
// test can actually catch a mistake in.
var (
	// balloonSrcMap 24x240. BalloonSrc[0..7], 24x30, stride 30
	// (kNumBalloonFrames = 8).
	BalloonSrc = [8]Rect{
		{Top: 0, Left: 0, Bottom: 30, Right: 24},
		{Top: 30, Left: 0, Bottom: 60, Right: 24},
		{Top: 60, Left: 0, Bottom: 90, Right: 24},
		{Top: 90, Left: 0, Bottom: 120, Right: 24},
		{Top: 120, Left: 0, Bottom: 150, Right: 24},
		{Top: 150, Left: 0, Bottom: 180, Right: 24},
		{Top: 180, Left: 0, Bottom: 210, Right: 24},
		{Top: 210, Left: 0, Bottom: 240, Right: 24},
	}

	// copterSrcMap 32x300. CopterSrc[0..9], 32x30, stride 30
	// (kNumCopterFrames = 10). The longest enemy strip in the game, and the only
	// one whose frame count is not a power of two.
	CopterSrc = [10]Rect{
		{Top: 0, Left: 0, Bottom: 30, Right: 32},
		{Top: 30, Left: 0, Bottom: 60, Right: 32},
		{Top: 60, Left: 0, Bottom: 90, Right: 32},
		{Top: 90, Left: 0, Bottom: 120, Right: 32},
		{Top: 120, Left: 0, Bottom: 150, Right: 32},
		{Top: 150, Left: 0, Bottom: 180, Right: 32},
		{Top: 180, Left: 0, Bottom: 210, Right: 32},
		{Top: 210, Left: 0, Bottom: 240, Right: 32},
		{Top: 240, Left: 0, Bottom: 270, Right: 32},
		{Top: 270, Left: 0, Bottom: 300, Right: 32},
	}

	// dartSrcMap 64x76. DartSrc[0..3], 64x19, stride 19 (kNumDartFrames = 4).
	//
	// The four frames are two *directions*, not an animation: a dart's registration
	// sets frame 0 for kDartLf and frame 2 for kDartRt and nothing ever advances it.
	// See HandleDart.
	DartSrc = [4]Rect{
		{Top: 0, Left: 0, Bottom: 19, Right: 64},
		{Top: 19, Left: 0, Bottom: 38, Right: 64},
		{Top: 38, Left: 0, Bottom: 57, Right: 64},
		{Top: 57, Left: 0, Bottom: 76, Right: 64},
	}

	// ballSrcMap 32x64. BallSrc[0..1], 32x32, stride 32 (kNumBallFrames = 2).
	BallSrc = [2]Rect{
		{Top: 0, Left: 0, Bottom: 32, Right: 32},
		{Top: 32, Left: 0, Bottom: 64, Right: 32},
	}

	// fishSrcMap 16x128. FishSrc[0..7], 16x16, stride 16 (kNumFishFrames = 8).
	//
	// Distinct from srcRects[kFish], which is the 36x33 still on the enemy sheet:
	// the static room draw uses that one and only the leaping fish uses this strip.
	// FishSrc[0] is already at the origin, which is why kFish is the one
	// registration case in AddDynamicObject with no ZeroRectCorner.
	FishSrc = [8]Rect{
		{Top: 0, Left: 0, Bottom: 16, Right: 16},
		{Top: 16, Left: 0, Bottom: 32, Right: 16},
		{Top: 32, Left: 0, Bottom: 48, Right: 16},
		{Top: 48, Left: 0, Bottom: 64, Right: 16},
		{Top: 64, Left: 0, Bottom: 80, Right: 16},
		{Top: 80, Left: 0, Bottom: 96, Right: 16},
		{Top: 96, Left: 0, Bottom: 112, Right: 16},
		{Top: 112, Left: 0, Bottom: 128, Right: 16},
	}

	// pointsSrcMap 24x120. PointsSrc[0..14], 24x8, stride 8
	// (StructuresInit.c:412-416, which loops to 15 with no #define behind it).
	//
	// Fifteen rows of numerals, read in pairs: AddFlyingPoint picks a start index
	// and an end index out of this strip and RenderFlyingPoints walks between them,
	// which is how one 24x120 sheet spells 100, 250, 300, 500 and 1000. The walking
	// index is flyingPoints[].mode and it indexes this array directly
	// (Render.c:369).
	PointsSrc = [15]Rect{
		{Top: 0, Left: 0, Bottom: 8, Right: 24},
		{Top: 8, Left: 0, Bottom: 16, Right: 24},
		{Top: 16, Left: 0, Bottom: 24, Right: 24},
		{Top: 24, Left: 0, Bottom: 32, Right: 24},
		{Top: 32, Left: 0, Bottom: 40, Right: 24},
		{Top: 40, Left: 0, Bottom: 48, Right: 24},
		{Top: 48, Left: 0, Bottom: 56, Right: 24},
		{Top: 56, Left: 0, Bottom: 64, Right: 24},
		{Top: 64, Left: 0, Bottom: 72, Right: 24},
		{Top: 72, Left: 0, Bottom: 80, Right: 24},
		{Top: 80, Left: 0, Bottom: 88, Right: 24},
		{Top: 88, Left: 0, Bottom: 96, Right: 24},
		{Top: 96, Left: 0, Bottom: 104, Right: 24},
		{Top: 104, Left: 0, Bottom: 112, Right: 24},
		{Top: 112, Left: 0, Bottom: 120, Right: 24},
	}
)
