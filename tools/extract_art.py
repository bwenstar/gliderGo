#!/usr/bin/env python3
"""extract_art.py -- Glider PRO 1.0.4 sprite extractor: resource PICTs -> RGBA PNGs.

Stage 2 of the pipeline specified in docs/analysis/graphics-assets.md section 7.
Stage 0 is probe_rez.py (Rez text dump -> res/<TYPE>/<id>.bin).
Stage 1 is probe_pict.py (PICT opcode stream -> RGB canvas + palette-index plane).

Stdlib only (struct + zlib, via probe_pict).  No third-party dependencies.

Usage
-----
    python3 tools/probe_rez.py extract /tmp/gp/res
    python3 tools/extract_art.py /tmp/gp/res /tmp/gp/art

Writes <outdir>/sheet/<name>.png, <outdir>/object/<WHAT>_<constant>.png,
<outdir>/object/85_kFlower_<i>.png and <outdir>/manifest.json.

Every rect and every PICT ID in ATLAS/SHEETS below is transcribed from
GliderPRO/Sources/StructuresInit.c, StructuresInit2.c::InitSrcRects and
ObjectDraw2.c; see graphics-assets.md sections 6.2, 6.3 and 6.6 for the
per-entry citations.  Rects are (left, top, right, bottom) -- QSetRect order,
NOT the Mac Rect struct order (top, left, bottom, right).

kind column:
    sheet  srcRect is a real sub-rect of the named 8-bit sheet GWorld
    tmpl   srcRect is only a size/hit template; art comes from named sub-rects
    key    own PICT, transparency = white colour key (palette index 0)
    opaq   own PICT, drawn opaque
    pair   own PICT + own 1-bit mask PICT composited via a temp GWorld pair
    proc   drawn procedurally from rects/lines; there are no pixels to extract
    none   never drawn at all
"""
import collections
import json
import os
import struct
import sys
import zlib

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import probe_pict as PP


# --------------------------------------------------------------------------
# The 14 static sheet surfaces: name -> (GWorld, GWorld bounds, art, mask)
# graphics-assets 6.2; QSetRect sites StructuresInit.c:253,301,350,431,455,
# 501,538,623,632,641,650,659,668 and StructuresInit2.c:63.
# switch has no mask: PICT 5003 does not exist.
# --------------------------------------------------------------------------
SHEETS = {
    'appliance':  ('applianceSrcMap', '80x269', 4005, 5005),
    'ball':       ('ballSrcMap', '32x64', 4014, 5014),
    'balloon':    ('balloonSrcMap', '24x240', 4011, 5011),
    'blower':     ('blowerSrcMap', '48x402', 4000, 5000),
    'bonus':      ('bonusSrcMap', '88x378', 4002, 5002),
    'clutter':    ('clutterSrcMap', '128x69', 4018, 5018),
    'copter':     ('copterSrcMap', '32x300', 4012, 5012),
    'dart':       ('dartSrcMap', '64x76', 4013, 5013),
    'drip':       ('dripSrcMap', '16x72', 4015, 5015),
    'enemy':      ('enemySrcMap', '36x33', 4016, 5016),
    'furniture':  ('furnitureSrcMap', '64x278', 4001, 5001),
    'light':      ('lightSrcMap', '72x126', 4004, 5004),
    'switch':     ('switchSrcMap', '32x104', 4003, None),
    'trans':      ('transSrcMap', '56x32', 4008, 5008),
}

# --------------------------------------------------------------------------
# All 116 initialised srcRects[] entries (graphics-assets 6.3).
# The 28 unassigned slots are absent by construction; see 6.4/6.5.
# --------------------------------------------------------------------------
ATLAS = [
    # (what, constant, kind, key, drawHelper, srcRect(l,t,r,b), note)
    (0x01, 'kFloorVent',      'sheet',  'blower',      'DrawSimpleBlowers',         [0, 0, 48, 11],        ''),
    (0x02, 'kCeilingVent',    'sheet',  'blower',      'DrawSimpleBlowers',         [0, 11, 48, 22],       ''),
    (0x03, 'kFloorBlower',    'sheet',  'blower',      'DrawSimpleBlowers',         [0, 22, 48, 37],       ''),
    (0x04, 'kCeilingBlower',  'sheet',  'blower',      'DrawSimpleBlowers',         [0, 37, 48, 52],       ''),
    (0x05, 'kSewerGrate',     'sheet',  'blower',      'DrawSimpleBlowers',         [0, 52, 48, 69],       ''),
    (0x06, 'kLeftFan',        'sheet',  'blower',      'DrawSimpleBlowers',         [0, 69, 40, 124],      'tikiFlame[] overlays x40-48'),
    (0x07, 'kRightFan',       'sheet',  'blower',      'DrawSimpleBlowers',         [0, 124, 40, 179],     ''),
    (0x08, 'kTaper',          'sheet',  'blower',      'DrawSimpleBlowers',         [0, 209, 20, 268],     '+ flame[0..4] 16x15 stride 15 @(32,179)'),
    (0x09, 'kCandle',         'sheet',  'blower',      'DrawSimpleBlowers',         [0, 179, 32, 209],     '+ flame[0..4] 16x15 stride 15 @(32,179)'),
    (0x0A, 'kStubby',         'sheet',  'blower',      'DrawSimpleBlowers',         [0, 268, 20, 304],     '+ flame[0..4] 16x15 stride 15 @(32,179)'),
    (0x0B, 'kTiki',           'sheet',  'blower',      'DrawTiki',                  [21, 268, 48, 296],    '+ tikiFlame[0..4] 8x10 stride 10 @(40,69); pole drawn procedurally'),
    (0x0C, 'kBBQ',            'key',    '3988',        'DrawPictSansWhiteObject',   [0, 0, 64, 33],        '+ coals[0..3] 32x9 stride 9 @(0,304) from blower sheet'),
    (0x0D, 'kInvisBlower',    'none',   None,          '-',                         [0, 0, 24, 24],        'hit/size template only'),
    (0x0E, 'kGrecoVent',      'sheet',  'blower',      'DrawSimpleBlowers',         [0, 340, 48, 358],     ''),
    (0x0F, 'kSewerBlower',    'sheet',  'blower',      'DrawSimpleBlowers',         [0, 390, 32, 402],     ''),
    (0x10, 'kLiftArea',       'none',   None,          '-',                         [0, 0, 64, 32],        'size overridden by data.a.distance/tall'),
    (0x11, 'kTable',          'tmpl',   'furniture',   'DrawTable',                 [0, 0, 64, 8],         'art = tableSrc 64x22 @(0,0); rect is kTableThick=8 thick'),
    (0x12, 'kShelf',          'tmpl',   'furniture',   'DrawShelf',                 [0, 0, 64, 6],         'art = shelfSrc 16x21 @(0,22); rect is kShelfThick=6 thick'),
    (0x13, 'kCabinet',        'tmpl',   'furniture',   'DrawCabinet',               [0, 0, 64, 64],        'art = hingeSrc 4x16 @(16,22), handleSrc 4x21 @(20,22)'),
    (0x14, 'kFilingCabinet',  'opaq',   '3995',        'DrawPictObject',            [0, 0, 74, 107],       ''),
    (0x15, 'kWasteBasket',    'sheet',  'furniture',   'DrawSimpleFurniture',       [0, 43, 64, 104],      ''),
    (0x16, 'kMilkCrate',      'sheet',  'furniture',   'DrawSimpleFurniture',       [0, 104, 64, 162],     ''),
    (0x17, 'kCounter',        'proc',   None,          'DrawCounter',               [0, 0, 128, 64],       'no PICT pixels at all'),
    (0x18, 'kDresser',        'tmpl',   'furniture',   'DrawDresser',               [0, 0, 128, 64],       'art = knobSrc 8x8 @(24,22) srcCopy, leftFootSrc 16x16 @(32,22), rightFootSrc 16x16 @(48,22)'),
    (0x19, 'kDeckTable',      'tmpl',   'furniture',   'DrawDeckTable',             [0, 0, 64, 8],         'art = deckSrc 64x21 @(0,162)'),
    (0x1A, 'kStool',          'sheet',  'furniture',   'DrawStool',                 [0, 183, 48, 221],     '+ procedural legs to kStoolBase 304'),
    (0x1B, 'kTrunk',          'key',    '3987',        'DrawPictSansWhiteObject',   [0, 0, 144, 80],       ''),
    (0x1C, 'kInvisObstacle',  'none',   None,          '-',                         [0, 0, 64, 64],        'hit/size template only'),
    (0x1D, 'kManhole',        'key',    '3967',        'DrawPictSansWhiteObject',   [0, 0, 123, 22],       ''),
    (0x1E, 'kBooks',          'key',    '3964',        'DrawPictSansWhiteObject',   [0, 0, 64, 51],        ''),
    (0x1F, 'kInvisBounce',    'none',   None,          '-',                         [0, 0, 64, 64],        'hit/size template only'),
    (0x21, 'kRedClock',       'sheet',  'bonus',       'DrawRedClock',              [0, 0, 28, 17],        '+ digits[0..10] 4x6 stride 6 @(28,0)'),
    (0x22, 'kBlueClock',      'sheet',  'bonus',       'DrawBlueClock',             [0, 17, 28, 42],       '+ ColorLine hands'),
    (0x23, 'kYellowClock',    'sheet',  'bonus',       'DrawYellowClock',           [0, 42, 28, 70],       '+ ColorLine hands'),
    (0x24, 'kCuckoo',         'sheet',  'bonus',       'DrawCuckoo',                [0, 148, 40, 228],     '+ pendulumSrc[0..2] 32x28 stride 28 @(56,186)'),
    (0x25, 'kPaper',          'sheet',  'bonus',       'DrawSimplePrizes',          [0, 127, 48, 148],     ''),
    (0x26, 'kBattery',        'sheet',  'bonus',       'DrawSimplePrizes',          [32, 0, 48, 25],       ''),
    (0x27, 'kBands',          'sheet',  'bonus',       'DrawSimplePrizes',          [20, 70, 48, 93],      'bandRects[0..2] 16x6 stride 6 in bandsSrcMap 4007/5007 are the in-flight bands'),
    (0x28, 'kGreaseRt',       'sheet',  'bonus',       'DrawGreaseRt',              [0, 243, 32, 270],     'frames greaseSrcRt[0..3] 32x27 @(0,243),(0,270),(0,297),(32,297)'),
    (0x29, 'kGreaseLf',       'sheet',  'bonus',       'DrawGreaseLf',              [0, 324, 32, 351],     'frames greaseSrcLf[0..3] 32x27 @(0,324),(32,324),(0,351),(32,351)'),
    (0x2A, 'kFoil',           'sheet',  'bonus',       'DrawFoil',                  [0, 228, 55, 243],     ''),
    (0x2B, 'kInvisBonus',     'none',   None,          '-',                         [0, 0, 24, 24],        'hit/size template only'),
    (0x2C, 'kStar',           'sheet',  'bonus',       'DrawSimplePrizes',          [48, 0, 80, 31],       'starSrc[0..5] 32x31 stride 31 @(48,0)'),
    (0x2D, 'kSparkle',        'sheet',  'bonus',       '(AddDynamicObject)',        [0, 70, 20, 89],       'sparkleSrc[0..4] 20x19; frames 2,3,4 @(0,70),(0,89),(0,108); [0]=[4], [1]=[3]'),
    (0x2E, 'kHelium',         'sheet',  'bonus',       'DrawSimplePrizes',          [32, 270, 88, 286],    ''),
    (0x2F, 'kSlider',         'none',   None,          '-',                         [0, 0, 64, 16],        'hit/size template only'),
    (0x31, 'kUpStairs',       'key',    '3997',        'DrawPictSansWhiteObject',   [0, 0, 160, 267],      ''),
    (0x32, 'kDownStairs',     'opaq',   '3996',        'DrawPictObject',            [0, 0, 160, 267],      ''),
    (0x33, 'kMailboxLf',      'pair',   '3986/3904',   'DrawMailboxLeft',           [0, 0, 94, 80],        'anchored to kMailboxBase 296'),
    (0x34, 'kMailboxRt',      'pair',   '3985/3903',   'DrawMailboxRight',          [0, 0, 94, 80],        'anchored to kMailboxBase 296'),
    (0x35, 'kFloorTrans',     'sheet',  'trans',       'DrawSimpleTransport',       [0, 1, 56, 16],        ''),
    (0x36, 'kCeilingTrans',   'sheet',  'trans',       'DrawSimpleTransport',       [0, 16, 56, 31],       ''),
    (0x37, 'kDoorInLf',       'key',    '3984',        'DrawPictSansWhiteObject',   [0, 0, 144, 322],      ''),
    (0x38, 'kDoorInRt',       'key',    '3983',        'DrawPictSansWhiteObject',   [0, 0, 144, 322],      ''),
    (0x39, 'kDoorExRt',       'opaq',   '3982',        'DrawPictObject',            [0, 0, 16, 322],       ''),
    (0x3A, 'kDoorExLf',       'opaq',   '3981',        'DrawPictObject',            [0, 0, 16, 322],       ''),
    (0x3B, 'kWindowInLf',     'key',    '3980',        'DrawPictSansWhiteObject',   [0, 0, 20, 170],       ''),
    (0x3C, 'kWindowInRt',     'key',    '3979',        'DrawPictSansWhiteObject',   [0, 0, 20, 170],       ''),
    (0x3D, 'kWindowExRt',     'opaq',   '3977',        'DrawPictObject',            [0, 0, 16, 170],       ''),
    (0x3E, 'kWindowExLf',     'opaq',   '3978',        'DrawPictObject',            [0, 0, 16, 170],       ''),
    (0x3F, 'kInvisTrans',     'none',   None,          '-',                         [0, 0, 64, 32],        'hit/size template only'),
    (0x40, 'kDeluxeTrans',    'none',   None,          '-',                         [0, 0, 64, 64],        'hit/size template only'),
    (0x41, 'kLightSwitch',    'tmpl',   'switch',      'DrawLightSwitch',           [0, 0, 15, 24],        'art = lightSwitchSrc[0] 15x24 @(0,0) on / [1] @(16,0) off'),
    (0x42, 'kMachineSwitch',  'tmpl',   'switch',      'DrawMachineSwitch',         [0, 48, 16, 72],       'art = machineSwitchSrc[0] 16x24 @(0,24) / [1] @(16,24); rect y-offset 48 disagrees'),
    (0x43, 'kThermostat',     'tmpl',   'switch',      'DrawThermostat',            [0, 48, 15, 72],       'art = thermostatSrc[0] 15x24 @(0,48) / [1] @(16,48)'),
    (0x44, 'kPowerSwitch',    'tmpl',   'switch',      'DrawPowerSwitch',           [0, 72, 8, 80],        'art = powerSrc[0] 8x8 @(0,72) / [1] @(8,72)'),
    (0x45, 'kKnifeSwitch',    'tmpl',   'switch',      'DrawKnifeSwitch',           [0, 80, 16, 104],      'art = knifeSwitchSrc[0] 16x24 @(0,80) / [1] @(16,80)'),
    (0x46, 'kInvisSwitch',    'none',   None,          '-',                         [0, 0, 12, 12],        'hit/size template only'),
    (0x47, 'kTrigger',        'none',   None,          '-',                         [0, 0, 12, 12],        'hit/size template only'),
    (0x48, 'kLgTrigger',      'none',   None,          '-',                         [0, 0, 48, 48],        'hit/size template only'),
    (0x49, 'kSoundTrigger',   'none',   None,          '-',                         [0, 0, 32, 32],        'hit/size template only'),
    (0x51, 'kCeilingLight',   'sheet',  'light',       'DrawSimpleLight',           [0, 0, 64, 20],        ''),
    (0x52, 'kLightBulb',      'sheet',  'light',       'DrawSimpleLight',           [0, 20, 16, 48],       ''),
    (0x53, 'kTableLamp',      'sheet',  'light',       'DrawSimpleLight',           [16, 20, 64, 90],      ''),
    (0x54, 'kHipLamp',        'key',    '3994',        'DrawPictSansWhiteObject',   [0, 0, 72, 276],       ''),
    (0x55, 'kDecoLamp',       'key',    '3993',        'DrawPictSansWhiteObject',   [0, 0, 64, 212],       ''),
    (0x56, 'kFlourescent',    'tmpl',   'light',       'DrawFlourescent',           [0, 0, 64, 12],        'art = flourescentSrc1 16x12 @(0,78) off / flourescentSrc2 @(0,90) on, tiled'),
    (0x57, 'kTrackLight',     'tmpl',   'light',       'DrawTrackLight',            [0, 0, 64, 24],        'art = trackLightSrc[0..2] 24x24 stride 24 in x @(0,102); kTrackLightSpacing 64'),
    (0x58, 'kInvisLight',     'none',   None,          '-',                         [0, 0, 16, 16],        'hit/size template only'),
    (0x61, 'kShredder',       'sheet',  'appliance',   'DrawSimpleAppliance',       [0, 0, 73, 22],        'shredSrcMap 4010/5010 40x35 holds the shredded-paper animation'),
    (0x62, 'kToaster',        'sheet',  'appliance',   'DrawSimpleAppliance',       [0, 22, 48, 49],       'toastSrcMap 4009/5009 32x174 holds breadSrc[0..5] 32x29 stride 29'),
    (0x63, 'kMacPlus',        'sheet',  'appliance',   'DrawMacPlus',               [0, 49, 48, 107],      '+ plusScreen1 32x22 @(48,127) off / plusScreen2 @(48,149) on, srcCopy at +10,+7'),
    (0x64, 'kGuitar',         'key',    '3991',        'DrawPictSansWhiteObject',   [0, 0, 64, 172],       ''),
    (0x65, 'kTV',             'pair',   '3992/3912',   'DrawTV',                    [0, 0, 92, 77],        '+ tvScreen1 64x49 @(0,171) off / tvScreen2 @(0,220) on, from applianceSrcMap'),
    (0x66, 'kCoffee',         'sheet',  'appliance',   'DrawCoffee',                [0, 107, 43, 171],     '+ coffeeLight1 8x4 @(72,171) off / coffeeLight2 @(72,175) on, srcCopy at +32,+57'),
    (0x67, 'kOutlet',         'sheet',  'appliance',   'DrawOutlet',                [64, 22, 80, 46],      'outletSrc[0..3] 16x24 stride 24 @(64,22) is the spark animation'),
    (0x68, 'kVCR',            'pair',   '3990/3913',   'DrawVCR',                   [0, 0, 96, 22],        '+ vcrTime1 16x4 @(64,179) / vcrTime2 @(64,183) from applianceSrcMap'),
    (0x69, 'kStereo',         'pair',   '3989/3914',   'DrawStereo',                [0, 0, 128, 53],       '+ stereoLight1 4x1 @(68,171) / stereoLight2 @(68,172)'),
    (0x6A, 'kMicrowave',      'pair',   '3971/3915',   'DrawMicrowave',             [0, 0, 92, 59],        '+ microOff 16x35 @(64,187) / microOn @(64,222)'),
    (0x6B, 'kCinderBlock',    'key',    '3960',        'DrawPictSansWhiteObject',   [0, 0, 40, 62],        ''),
    (0x6C, 'kFlowerBox',      'key',    '3959',        'DrawPictSansWhiteObject',   [0, 0, 80, 32],        ''),
    (0x6D, 'kCDs',            'sheet',  'appliance',   'DrawSimpleAppliance',       [48, 22, 64, 52],      ''),
    (0x6E, 'kCustomPict',     'key',    'data.g.height', 'DrawCustPictSansWhite',     [0, 0, 72, 34],        'default art PICT 10000 is exactly 72x34'),
    (0x71, 'kBalloon',        'sheet',  'balloon',     '(AddDynamicObject)',        [0, 0, 24, 30],        'balloonSrc[0..7] 24x30 stride 30'),
    (0x72, 'kCopterLf',       'sheet',  'copter',      '(AddDynamicObject)',        [0, 0, 32, 30],        'copterSrc[0..9] 32x30 stride 30'),
    (0x73, 'kCopterRt',       'sheet',  'copter',      '(AddDynamicObject)',        [0, 0, 32, 30],        'copterSrc[0..9] 32x30 stride 30'),
    (0x74, 'kDartLf',         'sheet',  'dart',        '(AddDynamicObject)',        [0, 0, 64, 19],        'dartSrc[0..3] 64x19 stride 19'),
    (0x75, 'kDartRt',         'sheet',  'dart',        '(AddDynamicObject)',        [0, 0, 64, 19],        'dartSrc[0..3] 64x19 stride 19'),
    (0x76, 'kBall',           'sheet',  'ball',        '(AddDynamicObject)',        [0, 0, 32, 32],        'ballSrc[0..1] 32x32 stride 32'),
    (0x77, 'kDrip',           'sheet',  'drip',        'DrawDrip',                  [0, 0, 16, 12],        'static draw uses dripSrc[3] @(0,36), NOT srcRects[kDrip]; dripSrc[0..5] 16x12 stride 12'),
    (0x78, 'kFish',           'sheet',  'enemy',       'DrawFish',                  [0, 0, 36, 33],        'static draw reads enemySrcMap 4016/5016; the 8-frame fishSrcMap 4017/5017 16x16 stride 16 is animation-only'),
    (0x79, 'kCobweb',         'pair',   '3958/3927',   'DrawPictWithMaskObject',    [0, 0, 54, 45],        ''),
    (0x81, 'kOzma',           'opaq',   '3975',        'DrawPictObject',            [0, 0, 102, 92],       ''),
    (0x82, 'kMirror',         'proc',   None,          'DrawMirror',                [0, 0, 64, 64],        'four ColorFrameRect insets, no PICT'),
    (0x83, 'kMousehole',      'sheet',  'clutter',     'DrawSimpleClutter',         [0, 0, 10, 11],        ''),
    (0x84, 'kFireplace',      'key',    '3973',        'DrawPictSansWhiteObject',   [0, 0, 180, 142],      ''),
    (0x86, 'kWallWindow',     'proc',   None,          'DrawWallWindow',            [0, 0, 64, 80],        'kWindowSillThick 7, no PICT'),
    (0x87, 'kBear',           'key',    '3972',        'DrawPictSansWhiteObject',   [0, 0, 56, 58],        ''),
    (0x88, 'kCalendar',       'opaq',   '3970',        'DrawCalendar',              [0, 0, 63, 92],        '+ month name from STR# 1005 at +((64-w)/2), +55'),
    (0x89, 'kVase1',          'key',    '3969',        'DrawPictSansWhiteObject',   [0, 0, 36, 45],        ''),
    (0x8A, 'kVase2',          'key',    '3968',        'DrawPictSansWhiteObject',   [0, 0, 35, 57],        ''),
    (0x8B, 'kBulletin',       'opaq',   '3966',        'DrawBulletin',              [0, 0, 80, 58],        ''),
    (0x8C, 'kCloud',          'pair',   '3965/3921',   'DrawPictWithMaskObject',    [0, 0, 128, 30],       ''),
    (0x8D, 'kFaucet',         'sheet',  'clutter',     'DrawSimpleClutter',         [0, 51, 56, 69],       ''),
    (0x8E, 'kRug',            'key',    '3962',        'DrawPictSansWhiteObject',   [0, 0, 144, 18],       ''),
    (0x8F, 'kChimes',         'key',    '3961',        'DrawPictSansWhiteObject',   [0, 0, 28, 74],        ''),
]

# flowerSrc[0..5] -- StructuresInit2.c:72-88, clutter sheet.  kFlower has no
# srcRects entry; these six rects are both the art and the object's size.
FLOWER_SRC = [(0, 23, 10, 51), (10, 16, 34, 51), (34, 16, 68, 51),
              (68, 14, 95, 37), (68, 37, 95, 51), (95, 0, 127, 51)]


# --------------------------------------------------------------------------
# The 14 strip surfaces: art frames or scrolling bands that srcRects[] never
# indexes, so they are reached through the named globals of graphics-assets 6.6
# rather than through ATLAS.  name -> (GWorld, art PICT, mask PICT or None,
# expected WxH).  The size column is an assertion, not a hint: all 14 were
# measured and all 14 agree, so a mismatch means the resource changed.
#
# glider/glider2/gliderFoil/gliderFoil2 are four different 48x668 colour
# sheets that share ONE mask (4999) because their frame geometry is identical;
# Play.c:124-135 and Player.c:1146-1152 swap them between the two glider
# GWorlds at runtime.
# --------------------------------------------------------------------------
STRIPS = [
    ('glider',      'glidSrcMap',   3999, 4999, (48, 668)),
    ('glider2',     'glid2SrcMap',  3974, 4999, (48, 668)),
    ('gliderFoil',  'glidSrcMap',   3976, 4999, (48, 668)),
    ('gliderFoil2', 'glid2SrcMap',  3963, 4999, (48, 668)),
    ('shadow',      'shadowSrcMap', 3998, 4998, (48, 18)),
    ('bands',       'bandsSrcMap',  4007, 5007, (16, 18)),
    ('points',      'pointsSrcMap', 4006, 5006, (24, 120)),
    ('toast',       'toastSrcMap',  4009, 5009, (32, 174)),
    ('shred',       'shredSrcMap',  4010, 5010, (40, 35)),
    ('fish',        'fishSrcMap',   4017, 5017, (16, 128)),
    ('supp',        'suppSrcMap',   1999, None, (512, 44)),
    ('angel',       'angelSrcMap',  1019, 1020, (96, 44)),
    ('badge',       'badgeSrcMap',  1996, None, (32, 66)),
    ('board',       'boardSrcMap',  1997, None, (1536, 20)),
]

# The 18 room backgrounds, 512x322 each, drawn with srcCopy into workSrcMap as
# the first step of DrawRoomBackground (RoomGraphics.c:238-239).  Nothing can
# render a room without these.  roomType.background holds the ID (offset 34).
BACKGROUNDS = list(range(2000, 2018))

# Room art that is neither a sheet, a strip nor an object: the manhole seen
# through the floor.  LoadScaledGraphic maps its picFrame onto the caller's
# rect (RoomGraphics.c:281) -- one of only two scaling sites in the game.
MISC = {3957: 'manhole_thru_floor'}

# The 38 UI plates: about box, splash, dialog banners, editor tool palettes,
# Room Info thumbnails, game-over and scoreboard art.  All opaque -- every one
# reaches the screen through DrawPicture or a srcCopy CopyBits.
UI = ([150, 151, 153] + list(range(1000, 1019)) + [1021, 1022, 1023]
      + [1202, 1211, 1216, 1217] + list(range(1988, 1996)) + [1998])

# kCustomPict (0x6E) names its art at runtime -- the house supplies the ID, so
# ATLAS carries 'data.g.height' rather than a number.  PICT 10000 is the
# placeholder the editor shows for an unresolved custom picture, and it is what
# the extractor emits (graphics-assets 7.9 rule 5).  Keyed on the object code so
# the fallback is impossible to confuse with a real constant ID.
KEY_DEFAULT_PICT = {0x6E: 10000}

# Masks are consumed, never written out; the manifest records which sprite used
# which.  Listed so the accounting self-test below can see all 152.
SHEET_MASKS = [5000, 5001, 5002, 5004, 5005, 5008, 5011, 5012, 5013, 5014,
               5015, 5016, 5018]
STRIP_MASKS = [1020, 4998, 4999, 5006, 5007, 5009, 5010, 5017]
OBJ_MASKS = [3903, 3904, 3912, 3913, 3914, 3915, 3921, 3927]


# --------------------------------------------------------------------------
# The only nine opcodes that occur in Glider PRO's 152 PICTs
# (graphics-assets 3.3).  Anything else is a hard error, not a warning:
# a picture that needs DirectBitsRect, BitsRgn, a PixPat, a QuickTime payload
# or any vector/text op is not Glider PRO art.
# --------------------------------------------------------------------------

ALLOWED_OPS = frozenset((
    0x0001,   # Clip          (always rgnSize 10, i.e. a bare rect)
    0x0011,   # VersionOp
    0x001E,   # DefHilite     (no data; skip)
    0x0090,   # BitsRect      (uncompressed CopyBits, rect source)
    0x0098,   # PackBitsRect  (PackBits CopyBits, rect source)
    0x0099,   # PackBitsRgn   (PackBits CopyBits, region mask)
    0x00A0,   # ShortComment  (2 bytes; skip)
    0x00FF,   # OpEndPic
    0x0C00,   # HeaderOp      (v2 header, 24 bytes)
))


# --------------------------------------------------------------------------
# Accounting: every PICT in the fork must land in exactly one bucket
# --------------------------------------------------------------------------

def pict_buckets():
    """The role partition of graphics-assets 7.8, computed from the tables
    above rather than restated, so the two cannot drift apart."""
    b = collections.OrderedDict()
    b['sheet_art'] = sorted(art for _, _, art, _ in SHEETS.values())
    b['sheet_mask'] = sorted(SHEET_MASKS)
    b['strip_art'] = sorted(art for _, _, art, _, _ in STRIPS)
    b['strip_mask'] = sorted(STRIP_MASKS)
    obj = set()
    for what, name, kind, key, fn, rect, note in ATLAS:
        if kind in ('key', 'opaq', 'pair'):
            for part in str(key).split('/'):
                if part.isdigit():
                    obj.add(int(part))
    obj |= set(KEY_DEFAULT_PICT.values())
    b['obj_art'] = sorted(obj - set(OBJ_MASKS))
    b['obj_mask'] = sorted(OBJ_MASKS)
    b['bg'] = sorted(BACKGROUNDS)
    b['misc'] = sorted(MISC)
    b['ui'] = sorted(UI)
    return b


def check_accounting(res, expect_total=152):
    """Fail loudly if a PICT is unaccounted for, double-counted or missing.

    This is the acceptance criterion for the asset step: 'every PICT either
    extracts or is listed as deliberately skipped'.  Asserting it in the tool
    means a resource nobody classified cannot slip through as silently
    ignored -- which is exactly how the 18 room backgrounds went missing from
    the first version of this pipeline.
    """
    buckets = pict_buckets()
    seen = collections.Counter()
    for ids in buckets.values():
        seen.update(ids)
    dupes = {i: n for i, n in seen.items() if n > 1}
    # 4999 legitimately serves all four glider sheets, but it is listed once.
    if dupes:
        raise ValueError("PICT(s) in more than one bucket: %s" % dupes)

    on_disk = set(int(f[:-4]) for f in os.listdir(os.path.join(res.dir, "PICT"))
                  if f.endswith(".bin"))
    classified = set(seen)
    missing = sorted(on_disk - classified)
    phantom = sorted(classified - on_disk)
    if missing:
        raise ValueError("%d PICT(s) present in the fork but classified by no "
                         "bucket: %s" % (len(missing), missing))
    if phantom:
        raise ValueError("%d PICT(s) named by a bucket but absent from the "
                         "fork: %s" % (len(phantom), phantom))
    if len(classified) != expect_total:
        raise ValueError("expected %d PICTs, accounted for %d"
                         % (expect_total, len(classified)))
    return buckets


# --------------------------------------------------------------------------
# Resource access (stage 0 output)
# --------------------------------------------------------------------------

class Res(object):
    def __init__(self, resdir):
        self.dir = resdir
        self.cache = {}

    def raw(self, typ, rid):
        with open(os.path.join(self.dir, typ, "%d.bin" % rid), "rb") as f:
            return f.read()

    def has(self, typ, rid):
        return os.path.exists(os.path.join(self.dir, typ, "%d.bin" % rid))

    def palette(self):
        """clut 128.  Only a fallback: 167 of 168 8-bit raster ops carry an
        embedded ColorTable byte-identical to this one (graphics-assets 4.1)."""
        return PP.ctab_to_palette(PP.parse_clut(self.raw("clut", 128)))

    def pict(self, rid, pal):
        """Decode one PICT.  Returns (w, h, rgb bytearray, info dict).
        info["idxmap"] is a list of H rows of W palette indices, None where
        the canvas was never painted."""
        if rid in self.cache:
            return self.cache[rid]
        pic = PP.Picture(self.raw("PICT", rid)).parse()
        bad = sorted(set("0x%04X" % op for _, op, _, _ in pic.ops
                         if op not in ALLOWED_OPS))
        if bad:
            raise ValueError("PICT %d uses opcodes outside the measured "
                             "Glider PRO subset: %s" % (rid, ", ".join(bad)))
        out = PP.rasterise(pic, pal)
        if out[3]["unhandled"]:
            raise ValueError("PICT %d has undrawn vector/text ops: %s"
                             % (rid, out[3]["unhandled"]))
        self.cache[rid] = out
        return out


# --------------------------------------------------------------------------
# The four alpha rules of graphics-assets section 5, unified into RGBA
# --------------------------------------------------------------------------

def rgba_from_mask(art, mask):
    """Path 1/3: alpha = 255 where the 1-bit mask index == 1, else 0.

    Mask and art are pixel-aligned and the same size, with one measured
    exception: mask 5005 is one row shorter than art 4005.  A short mask row
    is treated as fully transparent, which reproduces the original because
    nothing samples that row through the mask (graphics-assets 3.12.6)."""
    aw, ah, argb, _ = art
    mw, mh, _, mi = mask
    idx = mi["idxmap"]
    out = bytearray(aw * ah * 4)
    for y in range(ah):
        row = idx[y] if y < mh else None
        for x in range(aw):
            o3 = (y * aw + x) * 3
            o4 = (y * aw + x) * 4
            out[o4:o4 + 3] = argb[o3:o3 + 3]
            v = row[x] if (row is not None and x < mw) else None
            out[o4 + 3] = 255 if v == 1 else 0
    return out


def rgba_from_key(art):
    """Path 2: QuickDraw `transparent` mode against a fresh GWorld whose
    background is white.  alpha = 0 where the palette index is 0 (white) or
    the pixel was never painted at all."""
    aw, ah, argb, ai = art
    idx = ai["idxmap"]
    out = bytearray(aw * ah * 4)
    for y in range(ah):
        for x in range(aw):
            o3 = (y * aw + x) * 3
            o4 = (y * aw + x) * 4
            out[o4:o4 + 3] = argb[o3:o3 + 3]
            v = idx[y][x]
            out[o4 + 3] = 0 if (v is None or v == 0) else 255
    return out


def rgba_opaque(art):
    """Path 4: srcCopy / DrawPicture.  Everything is opaque."""
    aw, ah, argb, _ = art
    out = bytearray(aw * ah * 4)
    for i in range(aw * ah):
        out[i * 4:i * 4 + 3] = argb[i * 3:i * 3 + 3]
        out[i * 4 + 3] = 255
    return out


def crop(w, h, rgba, l, t, r, b):
    cw, ch = r - l, b - t
    out = bytearray(cw * ch * 4)
    for y in range(ch):
        sy = t + y
        if sy < 0 or sy >= h:
            continue
        for x in range(cw):
            sx = l + x
            if sx < 0 or sx >= w:
                continue
            out[(y * cw + x) * 4:(y * cw + x) * 4 + 4] = \
                rgba[(sy * w + sx) * 4:(sy * w + sx) * 4 + 4]
    return cw, ch, out


def opaque_count(w, h, rgba):
    return sum(1 for i in range(w * h) if rgba[i * 4 + 3] == 255)


def write_png_rgba(path, w, h, rgba):
    raw = bytearray()
    for y in range(h):
        raw.append(0)
        raw += rgba[y * w * 4:(y + 1) * w * 4]
    ihdr = struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0)
    d = os.path.dirname(path)
    if d:
        os.makedirs(d, exist_ok=True)
    with open(path, "wb") as f:
        f.write(b"\x89PNG\r\n\x1a\n")
        f.write(PP._chunk(b"IHDR", ihdr))
        f.write(PP._chunk(b"IDAT", zlib.compress(bytes(raw), 9)))
        f.write(PP._chunk(b"IEND", b""))


# --------------------------------------------------------------------------
# Driver
# --------------------------------------------------------------------------

def run(resdir, outdir):
    res = Res(resdir)
    pal = res.palette()
    manifest = {"sheets": {}, "strips": {}, "objects": {}, "backgrounds": {},
                "ui": {}, "misc": {}, "buckets": {}, "stats": {}}
    stats = collections.Counter()

    buckets = check_accounting(res)
    manifest["buckets"] = {k: v for k, v in buckets.items()}
    print("PICT accounting: %d resources, %d buckets, all classified"
          % (sum(len(v) for v in buckets.values()), len(buckets)))
    print("  " + "  ".join("%s=%d" % (k, len(v)) for k, v in buckets.items()))
    print()

    sheet_rgba = {}
    for name in sorted(SHEETS):
        gw, size, art_id, mask_id = SHEETS[name]
        art = res.pict(art_id, pal)
        if mask_id is None:
            rgba, how = rgba_opaque(art), "opaque (no mask resource exists)"
        else:
            rgba, how = rgba_from_mask(art, res.pict(mask_id, pal)), "mask %d" % mask_id
        w, h = art[0], art[1]
        sheet_rgba[name] = (w, h, rgba)
        write_png_rgba(os.path.join(outdir, "sheet", name + ".png"), w, h, rgba)
        op = opaque_count(w, h, rgba)
        manifest["sheets"][name] = {
            "gworld": gw, "gworld_size": size, "art_pict": art_id,
            "mask_pict": mask_id, "art_size": [w, h], "alpha": how,
            "opaque_px": op, "total_px": w * h}
        print("sheet %-10s %3dx%-3d art %d %-30s opaque %6d/%6d (%.1f%%)"
              % (name, w, h, art_id, how, op, w * h, 100.0 * op / (w * h)))

    for what, name, kind, key, fn, rect, note in ATLAS:
        l, t, r, b = rect
        slug = "%02X_%s" % (what, name)
        rec = {"what": what, "kind": kind, "srcRect": rect,
               "w": r - l, "h": b - t, "draw": fn, "note": note}
        png = os.path.join(outdir, "object", slug + ".png")
        if kind == "sheet":
            w, h, rgba = sheet_rgba[key]
            cw, ch, cr = crop(w, h, rgba, l, t, r, b)
            write_png_rgba(png, cw, ch, cr)
            rec["from"] = "sheet:" + key
            rec["file"] = "object/%s.png" % slug
        elif kind == "tmpl":
            rec["from"] = "sheet:%s (named sub-rects; graphics-assets 6.6)" % key
        elif kind == "key":
            head_ = key.split("/")[0]
            # A numeric key is a constant PICT ID.  A non-numeric one names a
            # runtime field, in which case KEY_DEFAULT_PICT supplies the
            # placeholder the original itself falls back to.
            pict_id = int(head_) if head_.isdigit() else KEY_DEFAULT_PICT.get(what)
            if pict_id is not None:
                art = res.pict(pict_id, pal)
                rgba = rgba_from_key(art)
                write_png_rgba(png, art[0], art[1], rgba)
                n = art[0] * art[1]
                rec.update({"from": "pict:%d" % pict_id, "file": "object/%s.png" % slug,
                            "alpha": "colour key index 0", "total_px": n,
                            "transparent_px": n - opaque_count(art[0], art[1], rgba)})
                if not head_.isdigit():
                    rec["runtime_source"] = key
                    rec["placeholder"] = True
            else:
                rec["from"] = "pict:%s" % key
        elif kind == "opaq":
            art = res.pict(int(key), pal)
            rgba = rgba_opaque(art)
            write_png_rgba(png, art[0], art[1], rgba)
            rec.update({"from": "pict:%s" % key, "file": "object/%s.png" % slug,
                        "alpha": "none"})
        elif kind == "pair":
            ap, mp = key.split("/")
            art = res.pict(int(ap), pal)
            rgba = rgba_from_mask(art, res.pict(int(mp), pal))
            write_png_rgba(png, art[0], art[1], rgba)
            n = art[0] * art[1]
            rec.update({"from": "pict:%s" % ap, "mask_pict": int(mp),
                        "file": "object/%s.png" % slug, "alpha": "mask " + mp,
                        "opaque_px": opaque_count(art[0], art[1], rgba),
                        "total_px": n})
        stats[kind] += 1
        manifest["objects"][slug] = rec

    # kFlower (0x85) has no srcRects entry at all: sized from flowerSrc[pict].
    w, h, rgba = sheet_rgba["clutter"]
    for i, (l, t, r, b) in enumerate(FLOWER_SRC):
        cw, ch, cr = crop(w, h, rgba, l, t, r, b)
        write_png_rgba(os.path.join(outdir, "object", "85_kFlower_%d.png" % i),
                       cw, ch, cr)
    manifest["objects"]["85_kFlower"] = {
        "what": 0x85, "kind": "frames", "srcRect": None, "from": "sheet:clutter",
        "frames": [list(f) for f in FLOWER_SRC],
        "note": "srcRects[kFlower] is never assigned; size comes from "
                "flowerSrc[data.i.pict] (graphics-assets 6.5)"}
    stats["frames"] += 1

    # ---- strips: frame bands srcRects never indexes (graphics-assets 6.6) ----
    print()
    for name, gw, art_id, mask_id, expect in STRIPS:
        art = res.pict(art_id, pal)
        w, h = art[0], art[1]
        if (w, h) != expect:
            raise ValueError("strip %s: PICT %d is %dx%d, expected %dx%d"
                             % (name, art_id, w, h, expect[0], expect[1]))
        if mask_id is None:
            rgba, how = rgba_opaque(art), "opaque (no mask resource)"
        else:
            rgba, how = rgba_from_mask(art, res.pict(mask_id, pal)), "mask %d" % mask_id
        write_png_rgba(os.path.join(outdir, "strip", name + ".png"), w, h, rgba)
        op = opaque_count(w, h, rgba)
        manifest["strips"][name] = {
            "gworld": gw, "art_pict": art_id, "mask_pict": mask_id,
            "size": [w, h], "alpha": how, "opaque_px": op, "total_px": w * h,
            "file": "strip/%s.png" % name}
        stats["strip"] += 1
        print("strip %-12s %4dx%-4d art %d %-24s opaque %6d/%6d (%.1f%%)"
              % (name, w, h, art_id, how, op, w * h, 100.0 * op / (w * h)))

    # ---- room backgrounds: srcCopy into workSrcMap, always opaque ----
    print()
    for rid in BACKGROUNDS:
        art = res.pict(rid, pal)
        w, h = art[0], art[1]
        if (w, h) != (512, 322):
            raise ValueError("background %d is %dx%d, expected 512x322"
                             % (rid, w, h))
        write_png_rgba(os.path.join(outdir, "bg", "%d.png" % rid), w, h,
                       rgba_opaque(art))
        manifest["backgrounds"][str(rid)] = {
            "size": [w, h], "alpha": "none (srcCopy)",
            "file": "bg/%d.png" % rid}
        stats["bg"] += 1
    print("bg          %d backgrounds %d..%d, all 512x322"
          % (len(BACKGROUNDS), BACKGROUNDS[0], BACKGROUNDS[-1]))

    # ---- UI plates and the one misc room picture ----
    for rid in UI:
        art = res.pict(rid, pal)
        w, h = art[0], art[1]
        write_png_rgba(os.path.join(outdir, "ui", "%d.png" % rid), w, h,
                       rgba_opaque(art))
        manifest["ui"][str(rid)] = {"size": [w, h], "alpha": "none",
                                    "file": "ui/%d.png" % rid}
        stats["ui"] += 1
    print("ui          %d plates" % len(UI))

    for rid, name in sorted(MISC.items()):
        art = res.pict(rid, pal)
        w, h = art[0], art[1]
        slug = "%d_%s" % (rid, name)
        write_png_rgba(os.path.join(outdir, "misc", slug + ".png"), w, h,
                       rgba_opaque(art))
        manifest["misc"][str(rid)] = {"size": [w, h], "alpha": "none",
                                      "file": "misc/%s.png" % slug}
        stats["misc"] += 1
    print("misc        %d (%s)" % (len(MISC), ", ".join(MISC.values())))

    manifest["stats"] = dict(stats)
    with open(os.path.join(outdir, "manifest.json"), "w") as f:
        json.dump(manifest, f, indent=1, sort_keys=True)
    print()
    print("kinds      :", dict(stats))
    for d in ("sheet", "object", "strip", "bg", "ui", "misc"):
        p = os.path.join(outdir, d)
        print("%-11s: %d PNGs" % (d + " PNGs", len(os.listdir(p))))
    return manifest


def main(argv):
    if len(argv) != 3:
        print(__doc__)
        return 2
    run(argv[1], argv[2])
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
