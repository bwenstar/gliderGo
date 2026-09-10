# Glider PRO 1.0.4 — Rendering Pipeline and Screen Composition

## Scope

This document reverse-documents, at porting fidelity, **how Glider PRO 1.0.4 gets pixels onto the
screen**: the offscreen surfaces it allocates, the order in which every layer is composited, the
dirty-rectangle double-buffer that limits per-frame blitting, the room→screen coordinate transform
(including the 3×3 neighbour viewport), the scoreboard overlay, room transitions, text drawing, and
the exact pixel dimensions of every surface, tile and sprite.

In scope:

* The three-surface model (`backSrcMap` / `workSrcMap` / on-screen `mainWindow`) and the two
  dirty-rect queues that shuttle data between them.
* Every offscreen GWorld the game creates, its rect, its colour depth, and the PICT resource(s)
  loaded into it.
* `RenderFrame()` — the exact per-frame draw order, all 11 phases.
* Room background composition: `DrawLocale`, the 8×(64×322) vertical tile strip engine,
  `DrawFloorSupport`, and the elevation-dependent outdoor backdrops.
* The object layer: `DrawARoomsObjects` and the **four distinct masking strategies** the game uses
  (pre-built 1-bit mask GWorld, on-demand temporary mask GWorld, `transparent`-mode white-keying,
  and plain rectangular `srcCopy`).
* The room→screen transform: `playOriginH/V`, `localRoomsDest[9]`, `OffsetRectRoomRelative`,
  `VerticalRoomOffset`, `justRoomsRect`.
* Scoreboard geometry and its two vertical modes (`kScoreboardHigh` / `kScoreboardLow`).
* Transitions (`WipeScreenOn`, `DumpScreenOn`, the dead `PourScreenOn`), and the fact that **no
  dissolve survives in 1.0.4**.
* Text drawing: scoreboard strings, banner, high scores, game-over trailer, coordinate windoid.
* The editor's XOR marching-ants marquee and the map-window thumbnails.
* Frame pacing (`TickCount()` busy-wait, `kTicksPerFrame`).
* Empirically measured resource-fork facts: 152 PICTs, their `picFrame`s, the PICT opcode subset
  actually used, the 256-entry CLUT, the 7 marquee patterns.

Out of scope (covered by sibling documents in this directory): physics/collision
(`object-dynamics.md`, `interactions.md`), the house file format (`house-format.md`), the audio
engine (`audio.md`), and the raw asset inventory (`graphics-assets.md`). This document duplicates a
small amount of asset data where it is load-bearing for composition.

**Naming convention used throughout.** All citations are of the form
`GliderPRO/Sources/Render.c:638`, with paths relative to the repository root and line
numbers taken from a copy of the file whose classic-Mac CR line terminators have been converted to
LF. The originals under `GliderPRO/Sources/` and `GliderPRO/Headers/` use bare `\r`, so `wc -l`
reports 0 and naive line numbering fails; convert first:

```sh
for f in Sources/*.c Headers/*.h; do tr '\r' '\n' < "$f" > /tmp/wf-render/$(basename "$f"); done
```

## Sources read

Read in full (CR→LF converted):

| File | LF lines | Why |
|---|---|---|
| `GliderPRO/Sources/Render.c` | 772 | The per-frame compositor and the dirty-rect queues |
| `GliderPRO/Sources/RoomGraphics.c` | 462 | Room background assembly, tiles, floor support |
| `GliderPRO/Sources/ObjectDraw.c` | 1406 | Object sprite drawing, part 1 |
| `GliderPRO/Sources/ObjectDraw2.c` | 1437 | Object sprite drawing, part 2 + all masking strategies |
| `GliderPRO/Sources/ObjectDrawAll.c` | 966 | `DrawARoomsObjects` — the object-layer dispatcher |
| `GliderPRO/Sources/Coordinates.c` | 196 | The floating coordinate windoid (editor) |
| `GliderPRO/Sources/Transitions.c` | 145 | `PourScreenOn`, `WipeScreenOn`, `DumpScreenOn` |
| `GliderPRO/Sources/Marquee.c` | 511 | Editor marching-ants selection marquee |
| `GliderPRO/Sources/MainWindow.c` | 601 | Window creation, splash drawing, menu-bar cover window |
| `GliderPRO/Headers/RoomGraphics.h` | 11 | (declares only `extern GWorldPtr suppSrcMap;`) |
| `GliderPRO/Headers/MainWindow.h` | 11 | (declares only `extern GWorldPtr workSrcMap;`) |

Read for the definitions the above depend on:

| File | What was taken from it |
|---|---|
| `GliderPRO/Headers/GliderDefines.h` | Every geometry / object / colour / animation constant |
| `GliderPRO/Headers/GliderStructs.h` | `roomType`, `objectType`, `gliderType`, `savedType`, `sparkleType`, `flyingPtType`, `flameType`, `pendulumType`, `bandType`, `greaseType`, `starType`, `shredType`, `dynaType` |
| `GliderPRO/Headers/GliderVars.h` | The global rect / GWorld declarations |
| `GliderPRO/Headers/Externs.h` | `kPreferredDepth` |
| `GliderPRO/Headers/Environ.h` | `macEnviron` |
| `GliderPRO/Sources/InterfaceInit.c` | `VariableInit` — `houseRect`, `playOriginH/V`, `localRoomsDest[9]`, `fadeInSequence[16]` |
| `GliderPRO/Sources/StructuresInit.c` | `InitScoreboardMap`, `InitGliderMap`, `InitBlowers`, `InitFurniture`, `InitPrizes`, `InitTransports`, `InitSwitches`, `InitLights`, `InitAppliances`, `InitEnemies` |
| `GliderPRO/Sources/StructuresInit2.c` | `InitClutter`, `InitSupport`, `InitAngel`, `CreateOffscreens`, `CreatePointers`, `InitSrcRects` |
| `GliderPRO/Sources/Utilities.c` | `CreateOffScreenGWorld`, `LoadGraphic`, `LoadScaledGraphic`, `WaitForInputEvent` |
| `GliderPRO/Sources/Environ.c` | `GetDeviceRect`, `CheckMemorySize` (an independent inventory of every GWorld's byte count) |
| `GliderPRO/Sources/RectUtils.c` | `ZeroRectCorner`, `CenterRectInRect`, `QSetRect`, `QOffsetRect`, `QUnionSimilarRect`, `RectWide`, `RectTall` |
| `GliderPRO/Sources/Scoreboard.c` | Scoreboard refresh, `AdjustScoreboardHeight`, `HandleDynamicScoreboard` |
| `GliderPRO/Sources/Play.c` | `NewGame`, `PlayGame` main loop, exit path |
| `GliderPRO/Sources/Player.c` | Glider fade sequences, `RenderGlider` inputs |
| `GliderPRO/Sources/Dynamics.c`, `Dynamics3.c` | The 7 dynamic-object renderers and `RenderDynamics` |
| `GliderPRO/Sources/DynamicMaps.c` | `savedMaps[24]` pre-composited animation strips, `AddSparkle` |
| `GliderPRO/Sources/Grease.c` | Grease spill drawing (writes to *both* buffers) |
| `GliderPRO/Sources/ObjectRects.c` | `GetObjectRect`, `OffsetRectRoomRelative`, `VerticalRoomOffset` |
| `GliderPRO/Sources/Room.c` | `DetermineRoomOpenings`, `GetOriginalBounding` |
| `GliderPRO/Sources/Transit.c` | The four callers of `WipeScreenOn` |
| `GliderPRO/Sources/Banner.c` | Opening banner composition |
| `GliderPRO/Sources/HighScores.c` | High-score screen composition |
| `GliderPRO/Sources/GameOver.c` | Falling-pages and angel/star game-over animations |
| `GliderPRO/Sources/Map.c` | Map-window room thumbnails |
| `GliderPRO/Sources/WindowUtils.c` | `GetLocalWindowRect` |
| `GliderPRO/Sources/Settings.c`, `Main.c` | Defaults for `numNeighbors`, `doBackground`, `isDoColorFade` |
| `GliderPRO/Sources/ColorUtils.c` | `ColorText`, `ColorRect`, `ColorOval`, `ColorRegion`, `ColorLine`, `ColorFrameRect`, `ColorFrameOval`, `HiliteRect` — the whole `Index2Color`+`RGBForeColor` family (§4.5, §23.5) |
| `GliderPRO/Sources/StringUtils.c` | `GetLocalizedString` = `GetIndString(…, STR# 150, i)` (§23.6) |
| `GliderPRO/Sources/Objects.c` | The `digits[11]`, `pendulumSrc[3]`, `greaseSrcRt[4]`, `greaseSrcLf[4]` definitions |
| `GliderPRO/Sources/Dynamics2.c` | The seven `evenFrame` readers: ball/drip/fish gravity, balloon and drip animation (§20.5) |
| `GliderPRO/Sources/Events.c`, `Input.c` | `BUILD_ARCADE_VERSION` key remapping and demo abort (§21.3, §21.4); the idle-splash timer |
| `GliderPRO/Sources/ObjectEdit.c` | The edit-mode object drawing path and the `srcOr` darkness stipple (§6.12, §23.1.1) |
| `GliderPRO/Sources/ObjectInfo.c`, `DialogUtils.c`, `About.c`, `RoomInfo.c` | The remaining `patXor` / diagonal-`Line` sites, for the exhaustive transfer-mode and line inventories (§23.1.1, §23.4.2) |
| `GliderPRO/Sources/AnimCursor.c` | `SpinCursor` — the `acur` 128 progress cursor (§23.9) |
| `GliderPRO/Sources/Interactions.c` | The spider-web `evenFrame` reader (§20.5) |

Binary evidence parsed with `python3` (results reported inline, section 5):

| Artefact | What was parsed |
|---|---|
| `GliderPRO/Glider PRO.r` | 15 MB Rez text dump of the entire resource fork; 538 resources decoded to bytes, including all 152 `PICT`s, `clut 128`/`129`, `PAT# 128`, `WIND 128`/`129`/`130` |
| `GliderPRO/Houses/*.binhex` | 22 BinHex 4.0 house files decoded; 4070 `roomType` records parsed to confirm tile indices and background IDs |

---

## 1. The three-surface model

Glider PRO composites every frame through exactly three full-screen-sized surfaces. Everything else
is a sprite sheet.

| Surface | C name | Rect global | Depth | Lifetime | Role |
|---|---|---|---|---|---|
| Background | `backSrcMap` | `backSrcRect` | `kPreferredDepth` = 8 | Whole session | The **static** room picture: backdrop tiles + every non-animating object of the visible 3×3 room block. Rebuilt only when the room changes (`DrawLocale`). |
| Work / compose | `workSrcMap` | `workSrcRect` | `kPreferredDepth` = 8 | Whole session | The **current frame**. Starts each frame as a copy of `backSrcMap` in the regions that changed last frame; animated sprites are drawn on top. Also doubles as the scratch surface for decoding full-screen PICTs (splash, star field, banner page). |
| Screen | `mainWindow` (a `WindowPtr`) | `mainWindowRect` | Device depth (8 expected) | Whole session | The visible window. Written only by `CopyRectsQD()` and by the transition/overlay routines. |

Declarations:

* `GliderPRO/Sources/MainWindow.c:35-38` — `Rect workSrcRect; GWorldPtr workSrcMap; Rect mainWindowRect; WindowPtr mainWindow, menuWindow;`
* `GliderPRO/Sources/RoomGraphics.c:28` — `GWorldPtr suppSrcMap;` (floor-support strip)
* `GliderPRO/Headers/MainWindow.h:11` — `extern GWorldPtr workSrcMap;` (the entire header)
* `GliderPRO/Headers/RoomGraphics.h:11` — `extern GWorldPtr suppSrcMap;` (the entire header)
* `GliderPRO/Headers/Externs.h:15` — `#define kPreferredDepth 8`

### 1.1 Why three and not two

The third surface exists so that **erase-behind is a blit, not a redraw**. A conventional
double-buffer would have to re-rasterise the room background under every moving sprite. Glider PRO
instead:

1. Draws a sprite into `workSrcMap` and records the sprite's destination rect in
   `back2WorkRects[]`.
2. Records the sprite's *swept* rect (union of this frame's and last frame's position — the `whole`
   field of the sprite struct) in `work2MainRects[]`.
3. At the end of the frame, blits every `work2MainRects[i]` from `workSrcMap` to the screen, then
   blits every `back2WorkRects[i]` from `backSrcMap` back into `workSrcMap`, wiping the sprite so
   the next frame starts clean.

So each moving pixel is touched exactly twice per frame (once work→screen, once back→work) and the
background is never re-rasterised.

`GliderPRO/Sources/Render.c:616-635`:

```c
void CopyRectsQD (void)
{
    short i;
    for (i = 0; i < numWork2Main; i++)
        CopyBits(workSrcMap, mainWindow, &work2MainRects[i], &work2MainRects[i], srcCopy, nil);
    for (i = 0; i < numBack2Work; i++)
        CopyBits(backSrcMap, workSrcMap, &back2WorkRects[i], &back2WorkRects[i], srcCopy, nil);
}
```

Note that **source and destination rects are identical in every one of these blits** — the three
surfaces share one coordinate system (see §2.6). That is the single most important simplifying
invariant in the renderer.

### 1.2 Coordinate identity of the three surfaces

`CreateOffscreens` (`GliderPRO/Sources/StructuresInit2.c:145-181`) builds both offscreen surfaces
from the *same* `houseRect`, each with its corner forced to (0,0):

```c
justRoomsRect = houseRect;  ZeroRectCorner(&justRoomsRect);
workSrcRect   = houseRect;  ZeroRectCorner(&workSrcRect);
CreateOffScreenGWorld(&workSrcMap, &workSrcRect, kPreferredDepth);
backSrcRect   = houseRect;  ZeroRectCorner(&backSrcRect);
CreateOffScreenGWorld(&backSrcMap, &backSrcRect, kPreferredDepth);
```

`ZeroRectCorner` (`GliderPRO/Sources/RectUtils.c:57-63`):

```c
void ZeroRectCorner (Rect *theRect)
{
    theRect->right  -= theRect->left;
    theRect->bottom -= theRect->top;
    theRect->left = 0;
    theRect->top  = 0;
}
```

`mainWindowRect` is likewise zero-cornered and is the window's *local* port rect
(`GliderPRO/Sources/MainWindow.c:222-224`), so window-local coordinates equal offscreen
coordinates. There is exactly one caveat: `mainWindowRect.bottom` is `houseRect.bottom` only when
the screen is at most `kMaxViewHeight` tall — see §2.3.

### 1.3 What is *not* in this model

* **Sprite sheets** (`glidSrcMap`, `blowerSrcMap`, …) are separate GWorlds, always the source of a
  blit and never a destination during play. They are filled once at init from PICTs.
* **1-bit mask GWorlds** (`glidMaskMap`, `blowerMaskMap`, …) pair 1:1 with the sheets and are only
  ever the mask argument of `CopyMask`.
* **`savedMaps[24]`** are small per-animation-cell GWorlds holding *pre-composited* frames
  (background swatch + masked sprite). See §11.2.
* **Temporary GWorlds** created and destroyed inside a single draw call for one-off large PICTs.
  See §8.2 (the create/blit/dispose idiom) and §11.3/§11.4 (the renderers that consume the
  `savedMaps` those temporaries feed).

---

## 2. Geometry, constants and coordinate spaces

### 2.1 The complete geometry constant table

All from `GliderPRO/Headers/GliderDefines.h`. Decimal, with hex where the value is used as a mask
or an object code.

| Constant | Value | Line | Meaning |
|---|---|---|---|
| `kNumTiles` | 8 | :496 | Vertical tile strips per room |
| `kTileWide` | 64 | :497 | Width of one tile strip, px |
| `kTileHigh` | 322 | :498 | Height of one tile strip = height of a room, px |
| `kRoomWide` | 512 | :499 | `kNumTiles * kTileWide`; room interior width, px |
| `kFloorSupportTall` | 44 | :500 | Height of the between-floors support band, px |
| `kVertLocalOffset` | 322 | :501 | Vertical spacing between stacked rooms (== `kTileHigh`) |
| `kScoreboardHigh` | 0 | :513 | Scoreboard mode: overlay at the top of the work map |
| `kScoreboardLow` | 1 | :514 | Scoreboard mode: directly above the central room |
| `kScoreboardTall` | 20 | :515 | Scoreboard band height, px |
| `kMaxViewWidth` | 1536 | :267 | Hard cap on `houseRect` width |
| `kMaxViewHeight` | 1026 | :268 | Hard cap on `houseRect` height |
| `kMapRoomWidth` | 32 | :247 | Map-window thumbnail width, px |
| `kMapRoomHeight` | 20 | :246 | Map-window thumbnail height, px |
| `kCeilingLimit` | 8 | :503 | Room-space y of a solid ceiling |
| `kFloorLimit` | 312 | :504 | Room-space y of a solid floor |
| `kRoofLimit` | 122 | :505 | Room-space y of an outdoor roof |
| `kLeftWallLimit` | 12 | :506 | Room-space x of a solid left wall |
| `kNoLeftWallLimit` | −24 | :507 | Room-space x when the left wall is open |
| `kRightWallLimit` | 500 | :508 | Room-space x of a solid right wall |
| `kNoRightWallLimit` | 536 | :509 | Room-space x when the right wall is open |
| `kNoCeilingLimit` | −10 | :510 | Room-space y when the ceiling is open |
| `kNoFloorLimit` | 332 | :511 | Room-space y when the floor is open |
| `kTicksPerFrame` | 2 | :533 | Frame period in 1/60.15 s ticks → ~30.07 fps |
| `kMaxSavedMaps` | 24 | :260 | Pre-composited animation cells |
| `kMaxRoomObs` | 24 | :250 | Objects per room |
| `kMaxSparkles` | 3 | :251 | Concurrent sparkles |
| `kNumSparkleModes` | 5 | :252 | Sparkle animation length |
| `kMaxFlyingPts` | 3 | :253 | Concurrent flying score numbers |
| `kMaxFlyingPointsLoop` | 24 | :254 | Flying-points animation loop budget |
| `kMaxCandles` | 20 | :255 | Concurrent candle flames |
| `kMaxTikis` | 8 | :256 | Concurrent tiki flames |
| `kMaxCoals` | 8 | :257 | Concurrent BBQ coals |
| `kMaxPendulums` | 8 | :258 | Concurrent clock pendulums |
| `kMaxHotSpots` | 56 | :259 | Interaction rects |
| `kMaxRubberBands` | 2 | :261 | Concurrent rubber bands in flight |
| `kMaxGrease` | 16 | :262 | Grease spills |
| `kMaxStars` | 4 | :263 | Animated stars |
| `kMaxShredded` | 4 | :264 | Shredded-glider animations |
| `kMaxDynamicObs` | 18 | :265 | Dynamic objects (`dinahs[]`) |
| `kMaxMasterObjects` | 216 | :266 | Master object list capacity (`= kMaxRoomObs * 9`) |
| `kNumSrcRects` | 0x90 = 144 | :437 | Size of the `srcRects[]` array |

Animation frame counts (`GliderPRO/Headers/GliderDefines.h:445-460`):

| Constant | Value |
|---|---|
| `kNumTrackLights` | 3 |
| `kNumOutletPicts` | 4 |
| `kNumCandleFlames` | 5 |
| `kNumTikiFlames` | 5 |
| `kNumBBQCoals` | 4 |
| `kNumBreadPicts` | 6 |
| `kNumBalloonFrames` | 8 |
| `kNumCopterFrames` | 10 |
| `kNumDartFrames` | 4 |
| `kNumBallFrames` | 2 |
| `kNumDripFrames` | 6 |
| `kNumFishFrames` | 8 |
| `kNumFlowers` | 6 |
| `kNumMarqueePats` | 7 |

Glider sprite metrics (`GliderPRO/Headers/GliderDefines.h:548-569`):

| Constant | Value | Meaning |
|---|---|---|
| `kGliderWide` | 48 | Glider sprite width, px |
| `kGliderHigh` | 20 | Glider sprite height, px |
| `kHalfGliderWide` | 24 | |
| `kGliderBurningHigh` | 26 | Height of the burning-glider frames |
| `kShadowHigh` | 9 | Shadow sprite height, px |
| `kShadowTop` | 306 | Room-space y at which the shadow is drawn |
| `kNumGliderSrcRects` | 31 | `gliderSrc[]` entry count |
| `kNumShadowSrcRects` | 2 | `shadowSrc[]` entry count (facing right / left) |
| `kFirstAboutFaceFrame` | 18 | |
| `kLastAboutFaceFrame` | 20 | |
| `kLeftFadeOffset` | 7 | Added to a fade frame index to get the left-facing variant |
| `kLastFadeSequence` | 16 | Length of `fadeInSequence[]` |
| `kGliderStartsDown` | 32 | |
| `kGliderFoil2PictID` | 3963 | |
| `kGlider2PictID` | 3974 | |
| `kGliderFoilPictID` | 3976 | |
| `kGliderPictID` | 3999 | |

Room-object placement constants (`GliderPRO/Headers/GliderDefines.h:467-494`) — these are
**room-space** y or x values baked into `GetObjectRect`:

| Constant | Value | Constant | Value |
|---|---|---|---|
| `kFloorVentTop` | 305 | `kDoorInTop` | 0 |
| `kCeilingVentTop` | 8 | `kDoorInLfLeft` | 0 |
| `kFloorBlowerTop` | 304 | `kDoorInRtLeft` | 368 |
| `kCeilingBlowerTop` | 5 | `kDoorExTop` | 0 |
| `kSewerGrateTop` | 303 | `kDoorExLfLeft` | 0 |
| `kCeilingTransTop` | 6 | `kDoorExRtLeft` | 496 |
| `kFloorTransTop` | 302 | `kWindowInTop` | 64 |
| `kStairsTop` | 28 | `kWindowInLfLeft` | 0 |
| `kCounterBottom` | 304 | `kWindowInRtLeft` | 492 |
| `kDresserBottom` | 293 | `kWindowExTop` | 64 |
| `kCeilingLightTop` | 4 | `kWindowExLfLeft` | 0 |
| `kHipLampTop` | 23 | `kWindowExRtLeft` | 496 |
| `kDecoLampTop` | 91 | `kTableThick` | 8 |
| `kFlourescentTop` | 12 | `kShelfThick` | 6 |
| `kTrackLightTop` | 5 | | |

Resource / scoring constants relevant to drawing
(`GliderPRO/Headers/GliderDefines.h:517-545`, `:623`, `:625`):

| Constant | Value | Meaning |
|---|---|---|
| `kHouseVersion` | 0x0200 | On-disk house version (all 22 shipped houses) |
| `kNewHouseVersion` | 0x0300 | Version written by newer editors (never seen in the corpus) |
| `kBaseBackgroundID` | 2000 | First built-in background PICT |
| `kFirstOutdoorBack` | 2009 | First background considered "outdoors" |
| `kNumBackgrounds` | 18 | Count of built-in backgrounds (2000..2017) |
| `kUserBackground` | 3000 | First user-supplied background PICT ID |
| `kUserStructureRange` | 3300 | User backgrounds ≥ this are "structures" |
| `kSplash8BitPICT` | 1000 | Splash screen PICT |
| `kStarPictID` | 1995 | Star-field backdrop PICT (high scores) |
| `kRoomIsEmpty` | −1 | Sentinel |
| `kObjectIsEmpty` | −1 | Sentinel |
| `kRedOrangeColor8` | 23 | Palette index used for scoreboard highlight |
| `kNumUndergroundFloors` | 8 | |
| `kMaxNumRoomsH` | 128 | |
| `kMaxNumRoomsV` | 64 | |
| `kStartSparkle` | 4 | Initial sparkle mode |
| `kScoreboardPictID` | 1997 | Scoreboard strip PICT |
| `kDemoLength` | 6702 | Bytes of `demo 128` |

Neighbour indices (`GliderPRO/Headers/GliderDefines.h:217-225`) — these index
`localRoomsDest[]`, `localNumbers[]`, `isStructure[]`:

| Name | Value | Direction |
|---|---|---|
| `kCentralRoom` | 0 | this room |
| `kNorthRoom` | 1 | up |
| `kNorthEastRoom` | 2 | up-right |
| `kEastRoom` | 3 | right |
| `kSouthEastRoom` | 4 | down-right |
| `kSouthRoom` | 5 | down |
| `kSouthWestRoom` | 6 | down-left |
| `kWestRoom` | 7 | left |
| `kNorthWestRoom` | 8 | up-left |

Built-in background PICT IDs (`GliderPRO/Headers/GliderDefines.h:227-244`):

| Name | ID | Name | ID |
|---|---|---|---|
| `kSimpleRoom` | 2000 | `kGarden` | 2009 |
| `kPaneledRoom` | 2001 | `kSkywalk` | 2010 |
| `kBasement` | 2002 | `kDirt` | 2011 |
| `kChildsRoom` | 2003 | `kMeadow` | 2012 |
| `kAsianRoom` | 2004 | `kField` | 2013 |
| `kUnfinishedRoom` | 2005 | `kRoof` | 2014 |
| `kSwingersRoom` | 2006 | `kSky` | 2015 |
| `kBathroom` | 2007 | `kStratosphere` | 2016 |
| `kLibrary` | 2008 | `kStars` | 2017 |

`kFirstOutdoorBack` (2009) is therefore exactly `kGarden`: 2000-2008 are the indoor backgrounds
and 2009-2017 the outdoor ones.

### 2.2 Where the screen rect comes from

`thisMac.screen` is the **full device rect of the chosen GDevice, including the menu-bar strip**.
`GliderPRO/Sources/Environ.c:334-343`:

```c
void GetDeviceRect (Rect *theRect)
{
    if (thisGDevice == nil)
        *theRect = qd.screenBits.bounds;
    else
        *theRect = (*thisGDevice)->gdRect;
}
```

`macEnviron` (`GliderPRO/Headers/Environ.h`) — fields relevant here:

| Field | Type | Meaning |
|---|---|---|
| `screen` | `Rect` | Full device rect (includes menu bar) |
| `gray` | `Rect` | The desktop "gray region" bounding box |
| `wasDepth` | `short` | Device depth before the game switched it |
| `isDepth` | `short` | Current device depth (4 or 8 are the two branches in the code) |
| `numScreens` | `short` | |
| `can1Bit`, `can4Bit`, `can8Bit` | `Boolean` | Depth capability |
| `hasColor`, `canColor`, `canSwitch` | `Boolean` | |
| `hasQT` | `Boolean` | QuickTime available |

Anything the code calls "menu bar height" is hard-coded to 20 in the rendering path:
`GliderPRO/Sources/MainWindow.c:224` (`mainWindowRect.bottom -= 20; // thisMac.menuHigh`),
`GliderPRO/Sources/MainWindow.c:228-229` (`MoveWindow(mainWindow, ..., thisMac.screen.top + 20, true)`),
and `GliderPRO/Sources/Play.c:144` (`tempRect.top = tempRect.bottom - 20; // thisMac.menuHigh`).
`kScoreboardTall` is also 20 (`GliderPRO/Headers/GliderDefines.h:515`). **These two 20s are
numerically equal but semantically unrelated**; a port must not conflate them.

### 2.3 `houseRect`, `workSrcRect`, `justRoomsRect`

`VariableInit` (`GliderPRO/Sources/InterfaceInit.c:196-201`):

```c
houseRect = thisMac.screen;
houseRect.bottom -= kScoreboardTall;        /* -20 */
if (houseRect.right  > kMaxViewWidth)  houseRect.right  = kMaxViewWidth;   /* 1536 */
if (houseRect.bottom > kMaxViewHeight) houseRect.bottom = kMaxViewHeight;  /* 1026 */
```

Notes:

1. `houseRect` is *not* zero-cornered — it retains `thisMac.screen`'s origin, which on a secondary
   monitor is non-zero. `CreateOffscreens` zero-corners copies of it
   (`GliderPRO/Sources/StructuresInit2.c:150-160`), so `workSrcRect` and `backSrcRect` are always
   `(0, 0, width, height)`.
2. The `-= kScoreboardTall` is what makes the offscreen surfaces exactly as tall as the window: the
   window loses 20 px to the menu bar, and the offscreen loses 20 px "to the scoreboard". They agree
   *by construction on ≤1026-px-tall screens*; on a taller screen the clamp to `kMaxViewHeight`
   breaks the identity and the window becomes taller than the offscreen (see Open questions).
3. `justRoomsRect` starts equal to `workSrcRect` and is later narrowed to just the central room band
   in `kScoreboardLow` mode (§13.10). It is the clip rect used by `AddRectToWorkRects`.

`CheckMemorySize` independently confirms the intended sizes
(`GliderPRO/Sources/Environ.c:585-655`): it computes the work map and back map each as
`RectWide(&houseRect) * (RectTall(&houseRect) + 1) * depth / 8` bytes, and the scoreboard map as
`RectWide(&houseRect) * 21 * depth / 8`. The `+1` row is slack for QuickDraw row alignment.

### 2.4 `playOriginH` / `playOriginV`

`GliderPRO/Sources/InterfaceInit.c:203-204`:

```c
playOriginH = (RectWide(&thisMac.screen) - kRoomWide) / 2;   /* (W - 512) / 2 */
playOriginV = (RectTall(&thisMac.screen) - kTileHigh) / 2;   /* (H - 322) / 2 */
```

Critically these are derived from **`thisMac.screen`, not `houseRect`**. Because
`houseRect.bottom == thisMac.screen.bottom - 20`, the central room is *not* vertically centred in
the offscreen surface: it sits 10 px lower than centre. On a 640×480 screen `playOriginV = 79`,
while centring in the 460-tall work map would have given 69.

`RectWide`/`RectTall` (`GliderPRO/Sources/RectUtils.c:95-107`) are plain `right-left` /
`bottom-top`.

### 2.5 `localRoomsDest[9]` — the 3×3 viewport

`GliderPRO/Sources/InterfaceInit.c:206-218`:

```c
for (i = 0; i < 9; i++) {
    QSetRect(&localRoomsDest[i], 0, 0, kRoomWide, kTileHigh);   /* 512 x 322 */
    QOffsetRect(&localRoomsDest[i], playOriginH, playOriginV);
}
QOffsetRect(&localRoomsDest[kNorthRoom],      0,          -kVertLocalOffset);
QOffsetRect(&localRoomsDest[kNorthEastRoom],  kRoomWide,  -kVertLocalOffset);
QOffsetRect(&localRoomsDest[kEastRoom],       kRoomWide,   0);
QOffsetRect(&localRoomsDest[kSouthEastRoom],  kRoomWide,   kVertLocalOffset);
QOffsetRect(&localRoomsDest[kSouthRoom],      0,           kVertLocalOffset);
QOffsetRect(&localRoomsDest[kSouthWestRoom], -kRoomWide,   kVertLocalOffset);
QOffsetRect(&localRoomsDest[kWestRoom],      -kRoomWide,   0);
QOffsetRect(&localRoomsDest[kNorthWestRoom], -kRoomWide,  -kVertLocalOffset);
```

So the nine destination rects tile a 1536×966 area whose centre cell is at
`(playOriginH, playOriginV)`. Only the parts that fall inside `workSrcRect` are visible; on a
640×480 screen the eight neighbours are almost entirely clipped away, which is exactly the intent —
you see a sliver of the adjacent rooms through open doorways.

`numNeighbors` selects how many of the nine are actually drawn: **1, 3 or 9**.

* 9 → all of `localRoomsDest`.
* 3 → central + `kWestRoom` + `kEastRoom`.
* 1 → central only.

Defaults: `GliderPRO/Sources/Settings.c:888` (`DisplayDefaults`) and
`GliderPRO/Sources/Settings.c:1257` both set `numNeighbors = 9`.
`GliderPRO/Sources/Main.c:192` forces `numNeighbors = 1` when
`thisMac.screen.right <= 512` (i.e. a 512×342 Mac Plus/SE-class screen).

### 2.6 The room→work-map transform

Object rects, glider rects and dynamic-object rects are all stored in **room space**: origin at the
top-left of the room interior, x in `[0, 512)`, y in `[0, 322)`. Two functions map room space to
work-map space.

`VerticalRoomOffset` (`GliderPRO/Sources/ObjectRects.c:1067`):

```
neighbor in {kNorthRoom, kNorthEastRoom, kNorthWestRoom}  ->  -kVertLocalOffset  (-322)
neighbor in {kSouthRoom, kSouthEastRoom, kSouthWestRoom}  ->  +kVertLocalOffset  (+322)
otherwise                                                 ->   0
```

`OffsetRectRoomRelative` (`GliderPRO/Sources/ObjectRects.c:1093`):

1. `QOffsetRect(theRect, playOriginH, playOriginV)`
2. then the same 8-way switch as `localRoomsDest`, adding `±kRoomWide` horizontally and
   `±kVertLocalOffset` vertically.

For the central room this reduces to the single addition `(playOriginH, playOriginV)`, which is why
every runtime renderer in `Render.c` and `Dynamics.c` just does
`QOffsetRect(&dest, playOriginH, playOriginV)` — the player and all dynamic objects only ever exist
in the central room.

**The transform is a pure translation. There is no scaling anywhere in the room rendering path.**
The room is always 512×322 device pixels. Larger screens show *more* of the neighbouring rooms, not
a bigger room. The only scaling in the whole program is `LoadScaledGraphic` (§4.3), used for
full-screen art (splash, star field, banner) and for the map-window "pretty" thumbnails.

### 2.7 Worked example: 640×480, 8-bit, `numNeighbors == 9`

| Quantity | Value |
|---|---|
| `thisMac.screen` | `(top 0, left 0, bottom 480, right 640)` |
| `houseRect` | `(0, 0, 460, 640)` |
| `workSrcRect` = `backSrcRect` = `justRoomsRect`(initial) | `(0, 0, 460, 640)` |
| `mainWindowRect` | `(0, 0, 460, 640)` |
| window position on screen | `(0, 20)` |
| `playOriginH` | `(640 − 512) / 2 = 64` |
| `playOriginV` | `(480 − 322) / 2 = 79` |
| `localRoomsDest[kCentralRoom]` | `(top 79, left 64, bottom 401, right 576)` |
| `localRoomsDest[kNorthRoom]` | `(−243, 64, 79, 576)` |
| `localRoomsDest[kSouthRoom]` | `(401, 64, 723, 576)` |
| `localRoomsDest[kWestRoom]` | `(79, −448, 401, 64)` |
| `localRoomsDest[kEastRoom]` | `(79, 576, 401, 1088)` |
| central room on the physical screen | y 99..421, x 64..576 |
| `boardSrcRect` | `(0, 0, 20, 640)` |
| `boardDestRect` (initial) | `(−20, 0, 0, 640)` |

Room-space point (0,0) therefore lands at work-map (64, 79) and physical screen (64, 99).

### 2.8 Worked example: 832×624

| Quantity | Value |
|---|---|
| `thisMac.screen` | `(0, 0, 624, 832)` |
| `houseRect` | `(0, 0, 604, 832)` |
| `workSrcRect` | `(0, 0, 604, 832)` |
| `playOriginH` | `(832 − 512) / 2 = 160` |
| `playOriginV` | `(624 − 322) / 2 = 151` |
| `localRoomsDest[kCentralRoom]` | `(151, 160, 473, 672)` |
| Visible slice of `kNorthRoom` | y −171..151 → rows 0..150 visible (151 px of the room above) |
| Visible slice of `kWestRoom` | x −352..160 → columns 0..159 visible (160 px of the room to the left) |

### 2.9 Worked example: 512×342 (`numNeighbors` forced to 1)

| Quantity | Value |
|---|---|
| `thisMac.screen` | `(0, 0, 342, 512)` |
| `houseRect` | `(0, 0, 322, 512)` |
| `workSrcRect` | `(0, 0, 322, 512)` |
| `playOriginH` | `(512 − 512) / 2 = 0` |
| `playOriginV` | `(342 − 322) / 2 = 10` |
| `localRoomsDest[kCentralRoom]` | `(10, 0, 332, 512)` |

Note the central room's bottom (332) exceeds `workSrcRect.bottom` (322) — the bottom 10 rows of the
room are clipped off. In `kScoreboardLow` mode `justRoomsRect` becomes `(10, 0, 332, 512)` whose
bottom is likewise outside the surface; `AddRectToWorkRects` therefore does not clamp it, and the
blit itself is clipped by QuickDraw. A Go port must clip explicitly here.

---

## 3. Complete offscreen GWorld inventory

Every offscreen surface the program allocates, in creation order. "Depth 8" means
`kPreferredDepth`; "Depth 1" means a `CopyMask` mask. PICT column gives the resource loaded via
`LoadGraphic` (which normalises `picFrame` to the origin, §4.2). "Measured w×h" is the PICT's actual
`picFrame` size as parsed from `Glider PRO.r` (§5.3).

### 3.1 The two compositing surfaces

| Global | Rect | Size | Depth | PICT | Site |
|---|---|---|---|---|---|
| `workSrcMap` | `workSrcRect` | `houseRect` w × h | 8 | — (painted / composited) | `GliderPRO/Sources/StructuresInit2.c:151-158` |
| `backSrcMap` | `backSrcRect` | `houseRect` w × h | 8 | — (composited) | `GliderPRO/Sources/StructuresInit2.c:160-162` |

### 3.2 Scoreboard surfaces — `InitScoreboardMap`, `GliderPRO/Sources/StructuresInit.c:59-166`

| Global | Rect | Size | Depth | PICT | Measured w×h |
|---|---|---|---|---|---|
| `boardSrcMap` | `boardSrcRect` | `houseRect` w × **20** | 8 | 1997 `kScoreboardPictID` | 1536 × 20 |
| `badgeSrcMap` | `badgeSrcRect` | 32 × 66 | 8 | 1996 `kBadgePictID` | 32 × 66 |
| `boardTSrcMap` | `boardTSrcRect` | 256 × 12 | 8 | — (text drawn at runtime) | — |
| `boardGSrcMap` | `boardGSrcRect` | 20 × 10 | 8 | — (text) | — |
| `boardPSrcMap` | `boardPSrcRect` | 64 × 10 | 8 | — (text) | — |

`boardSrcRect` is derived as `houseRect` zero-cornered with `bottom = kScoreboardTall`. PICT 1997 is
**1536 px wide** and is blitted into `boardSrcMap` at a negative horizontal offset so that its
central 640 px land in the visible band (`GliderPRO/Sources/StructuresInit.c:76-90`):

```c
if (boardSrcRect.right >= 640)
    hOffset = (RectWide(&boardSrcRect) - kMaxViewWidth) / 2;   /* (W - 1536)/2, negative for W<1536 */
else
    hOffset = -576;
```

For a 640-wide screen `hOffset = (640 − 1536)/2 = −448`, i.e. columns 448..1087 of the 1536-px strip
are shown. For a 1536-wide screen `hOffset = 0` and the whole strip is used.

### 3.3 Glider surfaces — `InitGliderMap`, `GliderPRO/Sources/StructuresInit.c:169-238`

| Global | Rect | Size | Depth | PICT | Measured |
|---|---|---|---|---|---|
| `glidSrcMap` | `glidSrcRect` | 48 × 668 | 8 | 3999 `kGliderPictID` | 48 × 668 |
| `glid2SrcMap` | `glidSrcRect` | 48 × 668 | 8 | 3974 `kGlider2PictID` | 48 × 668 |
| `glidMaskMap` | `glidSrcRect` | 48 × 668 | **1** | 4999 (= 3999 + 1000) | 48 × 668, PICT v1 |
| `shadowSrcMap` | `shadowSrcRect` | 48 × 18 | 8 | 3998 `kShadowPictID` | 48 × 18, PICT v1 |
| `shadowMaskMap` | `shadowSrcRect` | 48 × 18 | **1** | 4998 | 48 × 18, PICT v1 |
| `bandsSrcMap` | `bandsSrcRect` | 16 × 18 | 8 | 4007 `kRubberBandsPictID` | 16 × 18 |
| `bandsMaskMap` | `bandsSrcRect` | 16 × 18 | **1** | 5007 | 16 × 18, PICT v1 |

`glidSrcRect` is `QSetRect(&glidSrcRect, 0, 0, kGliderWide, 668)` with the comment
`// 32112 pixels` (`GliderPRO/Sources/StructuresInit.c:178`); 48 × 668 = 32064, so the comment is
off by 48 (one row) — harmless, it is only used by the memory estimator.

`gliderSrc[31]` sub-rects (`GliderPRO/Sources/StructuresInit.c:192-205`):

| Index range | Size | y | Purpose |
|---|---|---|---|
| 0..20 | 48 × 20 | `20 * i` | Normal frames; 0..6 right-facing, 7..13 left-facing (`kLeftFadeOffset` = 7), 14..20 the remaining poses incl. about-face 18..20 |
| 21..28 | 48 × 26 | `420 + 26 * (i − 21)` | Burning glider (`kGliderBurningHigh` = 26) |
| 29 | 48 × 20 | 628 | |
| 30 | 48 × 20 | 648 | |

Total consumed: 21·20 + 8·26 + 2·20 = 420 + 208 + 40 = 668. Exactly the sheet height.

`shadowSrc[i]` = 48 × 9 at `y = 9 * i`, i ∈ {0, 1}
(`GliderPRO/Sources/StructuresInit.c:216-220`). `bandRects[i]` = 16 × 6 at `y = 6 * i`, i ∈ {0,1,2}
(`GliderPRO/Sources/StructuresInit.c:231-235`).

### 3.4 Blower surfaces — `InitBlowers`, `GliderPRO/Sources/StructuresInit.c:244-287`

| Global | Rect | Size | Depth | PICT | Measured |
|---|---|---|---|---|---|
| `blowerSrcMap` | `blowerSrcRect` | 48 × 402 | 8 | 4000 | 48 × 402 |
| `blowerMaskMap` | `blowerSrcRect` | 48 × 402 | **1** | 5000 | 48 × 402, PICT v1 |

Sub-rects:

| Array | Count | Size | Position |
|---|---|---|---|
| `flame[0..4]` | 5 | 16 × 15 | `(32, 179 + 15 i)` |
| `tikiFlame[0..4]` | 5 | 8 × 10 | `(40, 69 + 10 i)` |
| `coals[0..3]` | 4 | 32 × 9 | `(0, 304 + 9 i)` |
| `leftStartGliderSrc` | 1 | 48 × 16 | `(0, 358)` |
| `rightStartGliderSrc` | 1 | 48 × 16 | `(0, 374)` |

`leftStartGliderSrc` is reused by the **editor marquee** as an XOR stencil out of
`blowerMaskMap` (§16.7).

### 3.5 Furniture — `InitFurniture`, `GliderPRO/Sources/StructuresInit.c:293-335`

| Global | Rect | Size | Depth | PICT | Measured |
|---|---|---|---|---|---|
| `furnitureSrcMap` | `furnitureSrcRect` | 64 × **278** | 8 | 4001 | 64 × **221** |
| `furnitureMaskMap` | `furnitureSrcRect` | 64 × 278 | **1** | 5001 | 64 × **221**, PICT v1 |

**Anomaly (verified by parsing the resource fork):** the GWorld is 278 rows tall but PICT 4001/5001
are only 221 rows. Rows 221..277 are therefore whatever `NewGWorld` left there (uninitialised /
zero). No `srcRect` addresses them, but the sheet is used right up to row 221: the lowest-reaching
source rect is `srcRects[kStool]` (48 × 38 at y 183..221, `GliderPRO/Sources/StructuresInit2.c:349-350`),
followed by `deckSrc` at y 162..183, `srcRects[kMilkCrate]` at y 104..162 and
`srcRects[kWasteBasket]` at y 43..104. 64 × 221 is therefore exactly the tight allocation for a port.

Sub-rects (`GliderPRO/Sources/StructuresInit.c:310-332`):

| Name | Size | Position |
|---|---|---|
| `tableSrc` | 64 × 22 | (0, 0) |
| `shelfSrc` | 16 × 21 | (0, 22) |
| `hingeSrc` | 4 × 16 | (16, 22) |
| `handleSrc` | 4 × 21 | (20, 22) |
| `knobSrc` | 8 × 8 | (24, 22) |
| `leftFootSrc` | 16 × 16 | (32, 22) |
| `rightFootSrc` | 16 × 16 | (48, 22) |
| `deckSrc` | 64 × 21 | (0, 162) |

### 3.6 Prizes / bonus — `InitPrizes`, `GliderPRO/Sources/StructuresInit.c:341-421`

| Global | Rect | Size | Depth | PICT | Measured |
|---|---|---|---|---|---|
| `bonusSrcMap` | `bonusSrcRect` | 88 × 378 | 8 | 4002 | 88 × 378 |
| `bonusMaskMap` | `bonusSrcRect` | 88 × 378 | **1** | 5002 | 88 × 378, PICT v1 |
| `pointsSrcMap` | `pointsSrcRect` | 24 × 120 | 8 | 4006 | 24 × 120 |
| `pointsMaskMap` | `pointsSrcRect` | 24 × 120 | **1** | 5006 | 24 × 120, **picFrame `(0,91,120,115)`** |

Sub-rects:

| Array | Count | Size | Position |
|---|---|---|---|
| `digits[0..10]` | 11 | 4 × 6 | `(28, 6 i)` |
| `pendulumSrc[0..2]` | 3 | 32 × 28 | `(56, 186 + 28 i)` |
| `greaseSrcRt[0..3]` | 4 | 32 × 27 | (0,243), (0,270), (0,297), (32,297) |
| `greaseSrcLf[0..3]` | 4 | 32 × 27 | (0,324), (32,324), (0,351), (32,351) |
| `starSrc[0..5]` | 6 | 32 × 31 | `(48, 31 i)` |
| `sparkleSrc[2..4]` | — | 20 × 19 | `(0, 70 + 19 i)` for i = 2,3,4 |
| `sparkleSrc[0]` | | | `= sparkleSrc[4]` (aliased) |
| `sparkleSrc[1]` | | | `= sparkleSrc[3]` (aliased) |
| `pointsSrc[0..14]` | 15 | 24 × 8 | `(0, 8 i)` |

The sparkle aliasing (`GliderPRO/Sources/StructuresInit.c:391-392`) means the 5-frame sparkle
animation plays cells 4, 3, 2, 3, 4 — grow-then-shrink out of only three distinct images.

### 3.7 Transports — `InitTransports`, `GliderPRO/Sources/StructuresInit.c:425-443`

| Global | Rect | Size | Depth | PICT | Measured |
|---|---|---|---|---|---|
| `transSrcMap` | `transSrcRect` | 56 × 32 | 8 | 4008 | 56 × 32 |
| `transMaskMap` | `transSrcRect` | 56 × 32 | **1** | 5008 | 56 × 32, PICT v1 |

**Original bug:** `InitTransports` never calls `GetGWorld(&wasCPort, &wasWorld)` at entry, yet calls
`SetGWorld(wasCPort, wasWorld)` at exit with the two uninitialised locals
(`GliderPRO/Sources/StructuresInit.c:441`). In practice `CreateOffscreens` calls it between
`InitPrizes` and `InitSwitches` and the next initialiser sets the port anyway, so the garbage
`SetGWorld` is survivable on 68k/PPC. A port should just not do this.

### 3.8 Switches — `InitSwitches`, `GliderPRO/Sources/StructuresInit.c:447-486`

| Global | Rect | Size | Depth | PICT | Measured |
|---|---|---|---|---|---|
| `switchSrcMap` | `switchSrcRect` | 32 × 104 | 8 | 4003 | 32 × 104 |
| — | — | — | — | **no mask GWorld** | PICT 5003 **does not exist** |

This is correct, not a bug: every switch is drawn with a plain rectangular `srcCopy`
(`GliderPRO/Sources/ObjectDraw2.c:301-372`), so no mask is needed. Verified empirically: the
resource fork contains PICTs 5000, 5001, 5002, **5004**, 5005 … — 5003 is absent (§5.9, anomaly 2).

Sub-rects — each switch has an off (index 0, x = 0) and on (index 1, x = 8 or 16) cell:

| Array | Size | Off position | On position |
|---|---|---|---|
| `lightSwitchSrc` | 15 × 24 | (0, 0) | (16, 0) |
| `machineSwitchSrc` | 16 × 24 | (0, 24) | (16, 24) |
| `thermostatSrc` | 15 × 24 | (0, 48) | (16, 48) |
| `powerSrc` | 8 × 8 | (0, 72) | (8, 72) |
| `knifeSwitchSrc` | 16 × 24 | (0, 80) | (16, 80) |

### 3.9 Lights — `InitLights`, `GliderPRO/Sources/StructuresInit.c:492-523`

| Global | Rect | Size | Depth | PICT | Measured |
|---|---|---|---|---|---|
| `lightSrcMap` | `lightSrcRect` | 72 × 126 | 8 | 4004 | 72 × 126 |
| `lightMaskMap` | `lightSrcRect` | 72 × 126 | **1** | 5004 | 72 × 126, PICT v1 |

| Array | Size | Position |
|---|---|---|
| `flourescentSrc1` | 16 × 12 | (0, 78) |
| `flourescentSrc2` | 16 × 12 | (0, 90) |
| `trackLightSrc[0..2]` | 24 × 24 | `(24 i, 102)` |

### 3.10 Appliances — `InitAppliances`, `GliderPRO/Sources/StructuresInit.c:529-612`

| Global | Rect | Size | Depth | PICT | Measured |
|---|---|---|---|---|---|
| `applianceSrcMap` | `applianceSrcRect` | 80 × 269 | 8 | 4005 | 80 × 269 |
| `applianceMaskMap` | `applianceSrcRect` | 80 × 269 | **1** | 5005 | 80 × **268** |
| `toastSrcMap` | `toastSrcRect` | 32 × 174 | 8 | 4009 | 32 × 174 |
| `toastMaskMap` | `toastSrcRect` | 32 × 174 | **1** | 5009 | 32 × 174, PICT v1 |
| `shredSrcMap` | `shredSrcRect` | 40 × 35 | 8 | 4010 | 40 × 35 |
| `shredMaskMap` | `shredSrcRect` | 40 × 35 | **1** | 5010 | 40 × 35, **picFrame `(195,0,230,40)`** |

**Anomaly:** mask PICT 5005 is 268 rows where the colour PICT 4005 is 269. The last row of the mask
GWorld is therefore uninitialised. The highest appliance sub-rect used is `microOn` at
y 222..257 and `microOff` at y 187..222, so row 268 is never sampled.

Sub-rects:

| Name | Size | Position |
|---|---|---|
| `plusScreen1` / `plusScreen2` | 32 × 22 | (48, 127) / (48, 149) |
| `tvScreen1` / `tvScreen2` | 64 × 49 | (0, 171) / (0, 220) |
| `coffeeLight1` / `coffeeLight2` | 8 × 4 | (72, 171) / (72, 175) |
| `outletSrc[0..3]` | 16 × 24 | `(64, 22 + 24 i)` |
| `breadSrc[0..5]` | 32 × 29 | `(0, 29 i)` — from `toastSrcMap` |
| `vcrTime1` / `vcrTime2` | 16 × 4 | (64, 179) / (64, 183) |
| `stereoLight1` / `stereoLight2` | 4 × 1 | (68, 171) / (68, 172) |
| `microOn` | 16 × 35 | (64, 222) |
| `microOff` | 16 × 35 | (64, 187) |

### 3.11 Enemies — `InitEnemies`, `GliderPRO/Sources/StructuresInit.c:614-724`

| Global | Rect | Size | Depth | PICT | Measured | Frames |
|---|---|---|---|---|---|---|
| `balloonSrcMap` | `balloonSrcRect` | 24 × 240 = 24 × (30·8) | 8 | 4011 | 24 × 240 | `kNumBalloonFrames` 8 |
| `balloonMaskMap` | | 24 × 240 | 1 | 5011 | 24 × 240, v1 | |
| `copterSrcMap` | `copterSrcRect` | 32 × 300 = 32 × (30·10) | 8 | 4012 | 32 × 300 | `kNumCopterFrames` 10 |
| `copterMaskMap` | | 32 × 300 | 1 | 5012 | 32 × 300, v1 | |
| `dartSrcMap` | `dartSrcRect` | 64 × 76 = 64 × (19·4) | 8 | 4013 | 64 × 76 | `kNumDartFrames` 4 |
| `dartMaskMap` | | 64 × 76 | 1 | 5013 | 64 × 76, v1 | |
| `ballSrcMap` | `ballSrcRect` | 32 × 64 = 32 × (32·2) | 8 | 4014 | 32 × 64 | `kNumBallFrames` 2 |
| `ballMaskMap` | | 32 × 64 | 1 | 5014 | 32 × 64, v1 | |
| `dripSrcMap` | `dripSrcRect` | 16 × 72 = 16 × (12·6) | 8 | 4015 | 16 × 72 | `kNumDripFrames` 6 |
| `dripMaskMap` | | 16 × 72 | 1 | 5015 | 16 × 72, v1 | |
| `enemySrcMap` | `enemySrcRect` | 36 × 33 | 8 | 4016 | 36 × 33 | (cobweb, 1 cell) |
| `enemyMaskMap` | | 36 × 33 | 1 | 5016 | 36 × 33, v1 | |
| `fishSrcMap` | `fishSrcRect` | **16** × 128 = 16 × (16·8) | 8 | 4017 | 16 × 128 | `kNumFishFrames` 8 |
| `fishMaskMap` | | 16 × 128 | 1 | 5017 | 16 × 128, v1 | |

Note the discrepancy between `fishSrcRect` (16 × 128, `GliderPRO/Sources/StructuresInit.c:677`) and
`srcRects[kFish]` (36 × 33, `GliderPRO/Sources/StructuresInit2.c:457`). `srcRects[kFish]` is used
only for editor hit-testing / object placement; the fish *renderer* uses `fishSrc[i]` which is 16 ×
16 at `y = 16 i`.

### 3.12 Clutter, support, angel — `GliderPRO/Sources/StructuresInit2.c:55-140`

| Global | Rect | Size | Depth | PICT | Measured |
|---|---|---|---|---|---|
| `clutterSrcMap` | `clutterSrcRect` | 128 × 69 | 8 | 4018 `kClutterPictID` | 128 × 69 |
| `clutterMaskMap` | `clutterSrcRect` | 128 × 69 | **1** | 5018 (= 4018 + 1000) | 128 × 69, PICT v1 |
| `suppSrcMap` | `suppSrcRect` | **512 × 44** (`kRoomWide` × `kFloorSupportTall`) | 8 | 1999 `kSupportPictID` | 512 × 44 |
| — | | | | **no mask** | |
| `angelSrcMap` | `angelSrcRect` | 96 × 44 | 8 | 1019 `kAngelPictID` | 96 × 44 |
| `angelMaskMap` | `angelSrcRect` | 96 × 44 | **1** | **1020** = `kAngelPictID + 1` | 96 × 44, PICT v1 |

**The angel is the one exception to the "+1000" mask-ID convention** — its mask is
`kAngelPictID + 1` (`GliderPRO/Sources/StructuresInit2.c:134`), i.e. PICT 1020, not 2019.

`flowerSrc[0..5]` from `clutterSrcMap` (`GliderPRO/Sources/StructuresInit2.c:72-89`):

| Index | Size | Position |
|---|---|---|
| 0 | 10 × 28 | (0, 23) |
| 1 | 24 × 35 | (10, 16) |
| 2 | 34 × 35 | (34, 16) |
| 3 | 27 × 23 | (68, 14) |
| 4 | 27 × 14 | (68, 37) |
| 5 | 32 × 51 | (95, 0) |

### 3.13 Editor-only and transient surfaces

| Global | Rect | Size | Depth | PICT | Site |
|---|---|---|---|---|---|
| `tileSrcMap` | `tileSrcRect` | 128 × 80, **`nil` at init** | 8 | — | `GliderPRO/Sources/StructuresInit2.c:178-179` |
| `nailSrcMap` | `nailSrcRect` | 32 × `kMapRoomHeight * (kNumBackgrounds+1)` = 32 × 380 | 8 | 1010 `kThumbnailPictID` | `GliderPRO/Sources/Map.c:743-746` |
| `toolSrcMap` | — | — | 8 | tools palette | `Tools.c` |
| `gameOverSrcMap` | `pageSrcRect` | 25 × 256 = 25 × (32·8) | 8 | 1988 `kLettersPictID` | `GliderPRO/Sources/GameOver.c:261-264` |
| `pageSrcMap` | `pageSrcRect` | 32 × 448 = 32 × (32·14) | 8 | 1990 `kPagesPictID` | `GliderPRO/Sources/GameOver.c:266-269` |
| `pageMaskMap` | `pageSrcRect` | 32 × 448 | **1** | 1989 `kPagesMaskID` | `GliderPRO/Sources/GameOver.c:271-273` |

`gameOverSrcMap`/`pageSrcMap`/`pageMaskMap` are created in `InitDiedGameOver` and disposed at the
end of `DoDiedGameOver` (`GliderPRO/Sources/GameOver.c:484-491`). Note that `pageSrcRect` is reused:
it is first set to 25 × 256 for the letters sheet then overwritten with 32 × 448 for the pages
sheet, so `gameOverSrcMap`'s own rect is not retained — the letter sub-rects `lettersSrc[0..7]` are
25 × 32 at `y = 32 i` (`GliderPRO/Sources/GameOver.c:300-304`) and `pageSrc[0..13]` are 32 × 32 at
`y = 32 i` (`GliderPRO/Sources/GameOver.c:275-279`).

### 3.14 Independent cross-check: `CheckMemorySize`

`GliderPRO/Sources/Environ.c:585-655` sums an estimate of every GWorld before allocating. Its
numbers are a second, independent statement of the intended sizes (colour pixels + mask bits):

| Surface | Colour pixels | Mask bits |
|---|---|---|
| work map | `W * (H+1)` | — |
| back map | `W * (H+1)` | — |
| scoreboard | `W * 21` | — |
| "more scoreboard" | 6396 | — |
| glider ×2 | 32112 each | 32064 |
| shadow | 912 | 864 |
| bands | 304 | 288 |
| blower | 19344 | 19344 |
| furniture | 17856 | 17792 |
| prizes | 33264 | 33176 |
| points | 2904 | 2880 |
| transport | 1848 | 1792 |
| switches | 3360 | — (none) |
| lights | 9144 | 9072 |
| appliances | 21600 | 21520 |
| toast | 5600 | 5568 |
| shredded | 1440 | 1400 |
| balloon | 5784 | 5760 |
| copter | 9632 | 9600 |
| dart | 4928 | 4864 |
| ball | 2080 | 2048 |
| drip | 1168 | 1152 |
| enemy | 1224 | 1188 |
| fish | 2064 | 2048 |
| clutter | 8960 | 8832 |
| support | 23040 | — (none) |
| angel | 4320 | 4224 |

`W = RectWide(&houseRect)`, `H = RectTall(&houseRect)`. Switches and support have **no mask entry**
(`Environ.c:614-615`, `:639-640` go straight from the map line to the next map line), independently
confirming they have no mask GWorld.

I checked every one of those 24 literals against the `QSetRect(&<x>SrcRect, 0, 0, w, h)` calls in
`StructuresInit.c` / `StructuresInit2.c` with a script. The rule is exact, not approximate:

- **colour map literal = `w * (h + 1)`** for all 24 surfaces except one;
- **mask literal = `w * h`** for all 21 surfaces that have a mask, except two.

The three exceptions, all in the prizes/blower group:

| Surface | `w × h` (from `QSetRect`) | `w*h` | `w*(h+1)` | `Environ.c` map | `Environ.c` mask | Anomaly |
|---|---|---|---|---|---|---|
| `bonusSrcMap` (prizes) | 88 × 378 (`StructuresInit.c:350`) | 33264 | 33352 | 33264 = `w*h` | 33176 = `w*(h-1)` | map has **no** `+1` row; mask is **one row short** |
| `blowerSrcMap` | 48 × 402 (`StructuresInit.c:253`) | 19296 | 19344 | 19344 = `w*(h+1)` | 19344 = `w*(h+1)` | mask literal copy-pasted from the map literal |
| `switchSrcMap` | 32 × 104 (`StructuresInit.c:455`) | 3328 | 3360 | 3360 = `w*(h+1)` | — | no mask exists (correct) |

Every other surface matches to the pixel, e.g. `applianceSrcMap` 80 × 269 → map `80*270 = 21600` ✓,
mask `80*269 = 21520` ✓; `suppSrcMap` 512 × 44 → map `512*45 = 23040` ✓; `angelSrcMap` 96 × 44 → map
`96*45 = 4320` ✓, mask `96*44 = 4224` ✓; `fishSrcMap` 16 × 128 → map `16*129 = 2064` ✓, mask
`16*128 = 2048` ✓. This is a strong independent confirmation of the whole sheet-dimension table in
3.2–3.12: two people (the estimator author and the `QSetRect` author) wrote the same numbers twice.

Two bugs in the estimator itself, both harmless in the shipped build:

1. Every colour-map term is scaled by `thisMac.isDepth` (4 or 8), but `CreateOffScreenGWorld` is
   *always* called with `kPreferredDepth` = 8 (every call site in the grep at 3.13). On a 16-colour
   screen the estimate is therefore exactly half of what is actually allocated.
2. `bytesNeeded += 4L * (long)thisMac.screen.bottom;` (`Environ.c:585`) and the two
   `4L * houseRect.bottom` terms (`:589`, `:593`) are meant to be the `rowBytes` arrays; they are one
   `long` per *scanline* rather than per row of the pixmap, which is coincidentally the right shape
   but the wrong surface (`screen.bottom` for the work map).

---

## 4. Surface creation and PICT loading helpers

### 4.1 `CreateOffScreenGWorld` — `GliderPRO/Sources/Utilities.c:266`

```c
OSErr CreateOffScreenGWorld (GWorldPtr *theGWorld, Rect *bounds, short depth)
{
    OSErr theErr;
    theErr = NewGWorld(theGWorld, depth, bounds, nil, nil, useTempMem);
    if (theErr != noErr)
        theErr = NewGWorld(theGWorld, depth, bounds, nil, nil, 0);
    LockPixels(GetGWorldPixMap(*theGWorld));
    return theErr;
}
```

Key facts:

1. `useTempMem` first, then plain heap. On a Go port both branches collapse to "allocate a pixel
   buffer".
2. `nil` for the colour table means "use the current device's CLUT". The game **never installs a
   palette** — there is no live `SetEntries` / `NewPalette` / `SetPalette` anywhere (§22.1) — so this
   is the *system* 8-bit CLUT that `SwitchToDepth(8, true)` leaves in place. That CLUT is the standard
   Macintosh 256-colour table, which is exactly what `clut` 128 in the app's own resource fork
   contains (measured structure in §5.6; `clut` 128 and `clut` 129 proved byte-identical in §22.2).
   The net effect is the one a port cares about: every 8-bit GWorld in the program shares one fixed
   256-entry palette, and `clut` 128 is a faithful copy of it.
3. `LockPixels` is called and **never balanced with `UnlockPixels`** — the pixel base address is
   assumed stable forever.
4. The `bounds` rect passed in is used verbatim as the GWorld's `portRect` *and* its
   `pixMap.bounds`, so a GWorld whose rect has a non-zero origin would have a shifted coordinate
   system. Every call site in Glider PRO passes a rect starting at (0,0) — except
   `CreateOffScreenGWorld(&workSrcMap, &workSrcRect, ...)` where `workSrcRect` has already been
   zero-cornered. So all GWorlds are origin-based.

### 4.2 `LoadGraphic` — `GliderPRO/Sources/Utilities.c:317`

```c
void LoadGraphic (short resID)
{
    Rect        bounds;
    PicHandle   thePicture;

    thePicture = GetPicture(resID);
    if (thePicture == nil)
        RedAlert(kErrFailedGraphicLoad);

    HLock((Handle)thePicture);
    bounds = (*thePicture)->picFrame;
    HUnlock((Handle)thePicture);
    OffsetRect(&bounds, -bounds.left, -bounds.top);
    DrawPicture(thePicture, &bounds);

    ReleaseResource((Handle)thePicture);
}
```

**Behaviour that matters:** the PICT's `picFrame` is *translated to the origin but not resized*.
Therefore:

* A PICT whose `picFrame` is `(195, 0, 230, 40)` (PICT 5010, the shredded-glider mask) is drawn at
  (0,0) with size 40 × 35 — the odd origin is harmless.
* A PICT whose `picFrame` is `(0, 91, 120, 115)` (PICT 5006, the flying-points mask) is drawn at
  (0,0) with size 24 × 120 — likewise harmless.
* A PICT whose *size* differs from the destination GWorld (4001 at 64 × 221 into a 64 × 278 GWorld)
  is **not** stretched; the extra rows stay uninitialised.

### 4.3 `LoadScaledGraphic` — `GliderPRO/Sources/Utilities.c:340`

```c
void LoadScaledGraphic (short resID, Rect *theRect)
{
    PicHandle thePicture = GetPicture(resID);
    if (thePicture == nil)
        RedAlert(kErrFailedGraphicLoad);
    DrawPicture(thePicture, theRect);
    ReleaseResource((Handle)thePicture);
}
```

`DrawPicture` with a destination rect different from `picFrame` **scales** (QuickDraw stretches the
`BitsRect`/`PackBitsRect` blits). Call sites:

| Site | PICT | Destination |
|---|---|---|
| `GliderPRO/Sources/MainWindow.c:106`, `:143`, `:255` | 1000 `kSplash8BitPICT` | `(0,0,460,640)` offset by `(splashOriginH, splashOriginV)` |
| `GliderPRO/Sources/Play.c:274` | 1000 | `(0,0,460,640)` offset by splash origin (game-exit repaint) |
| `GliderPRO/Sources/HighScores.c:67` | 1995 `kStarPictID` | `(0,0,640,480)` offset by splash origin |
| `GliderPRO/Sources/GameOver.c:87` | 1021 `kMilkywayPictID` | `(0,0,640,460)` centred in `workSrcRect` |
| `GliderPRO/Sources/Banner.c:61` | 1993 `kBannerPageTopPICT` | top 190 rows of a 330 × 220 page |
| `GliderPRO/Sources/Banner.c:224/227` | 1018 / 1017 | 256 × 64 centred, shifted up 20 |
| `GliderPRO/Sources/RoomGraphics.c:373` | 3957 `kManholeThruFloor` | a `tempManholes[i]` rect |
| `GliderPRO/Sources/Map.c:178` (`LoadGraphicPlus`) | user background | a 32 × 20 thumbnail cell |

Since PICTs 1000, 1021, 1995 are all natively 640 × 460 / 640 × 460 / 640 × 460 (measured, §5.3) and
1995 is drawn into a 640 × **480** rect, the star field is stretched vertically by 480/460 = 1.043.
This is a real, visible non-uniform scale a port must reproduce (or deliberately not).

`splashOriginH` / `splashOriginV` (`GliderPRO/Sources/MainWindow.c:245-252`):

```c
splashOriginH = ((thisMac.screen.right  - thisMac.screen.left) - 640) / 2;
if (splashOriginH < 0) splashOriginH = 0;
splashOriginV = ((thisMac.screen.bottom - thisMac.screen.top)  - 480) / 2;
if (splashOriginV < 0) splashOriginV = 0;
```

On 640×480 both are 0. On 832×624 they are 96 and 72.

### 4.4 `LoadGraphicSpecial` — `GliderPRO/Sources/RoomGraphics.c:134-157`

The room-background loader, with a three-level fallback chain:

```
1. thePicture = GetPicture(resID)                      /* 'PICT' resource */
2. if nil: thePicture = (PicHandle)GetResource('Date', resID)
3. if nil: thePicture = GetPicture(2000)                /* kSimpleRoom */
4. if still nil: RedAlert(kErrFailedGraphicLoad)
5. bounds = (*thePicture)->picFrame;
   OffsetRect(&bounds, -bounds.left, -bounds.top);
   DrawPicture(thePicture, &bounds);
   ReleaseResource(...)
```

The `'Date'` type is how user houses smuggle custom backgrounds past the Resource Manager's
`GetPicture` (which only looks at `'PICT'`); a house's resource fork stores its custom art as
`'Date'` resources with PICT IDs ≥ 3000. **A Go port must accept both type codes for the same
payload.** `LoadGraphicPlus` (`GliderPRO/Sources/Map.c:165-180`) does the same two-step lookup but
scales and returns silently on failure.

### 4.5 Colour-index drawing helpers — `GliderPRO/Sources/ColorUtils.c`

QuickDraw's `ForeColor` accepts only the eight classic colour constants (`blackColor`, `whiteColor`,
`redColor`, `greenColor`, `blueColor`, `cyanColor`, `magentaColor`, `yellowColor`). To draw in an
arbitrary palette index, Glider PRO wraps every primitive:

| Helper | Line | Primitive |
|---|---|---|
| `ColorText(StringPtr, long color)` | :20 | `DrawString` |
| `ColorRect(Rect *, long color)` | :36 | `PaintRect` |
| `ColorOval(Rect *, long color)` | :52 | `PaintOval` |
| `ColorRegion(RgnHandle, long color)` | :68 | `PaintRgn` |
| `ColorLine(h0, v0, h1, v1, long color)` | :84 | `MoveTo` + `LineTo` |
| `HiliteRect(Rect *, short color1, short color2)` | :103 | Four `ColorLine`s: top and left in `color1`, right and bottom in `color2` |
| `ColorFrameRect(Rect *, long color)` | :120 | `FrameRect` |
| `ColorFrameWHRect(left, top, wide, high, long color)` | :137 | `FrameRect` on a width/height rect |

Every one has the identical five-step body, e.g. `GliderPRO/Sources/ColorUtils.c:36-45`:

```c
GetForeColor(&wasColor);
Index2Color(color, &theRGBColor);
RGBForeColor(&theRGBColor);
PaintRect(theRect);
RGBForeColor(&wasColor);
```

`HiliteRect` is the only non-trivial one — its exact line endpoints
(`GliderPRO/Sources/ColorUtils.c:105-112`) are:

```
(left,      top)        -> (right-2,  top)         color1
(left,      top)        -> (left,     bottom-2)    color1
(right-1,   top)        -> (right-1,  bottom-2)    color2
(left+1,    bottom-1)   -> (right-1,  bottom-1)    color2
```

In a Go port with an indexed framebuffer this all collapses to "write palette index N directly",
because `Index2Color(N)` is by definition `clut128[N]` (the palette is measured entry-by-entry in
§5.6; the `Index2Color` → `RGBForeColor` round trip is exact because the 256 entries have no duplicate
RGB values, §23.5).

### 4.6 The mask-PICT ID convention

For every sprite sheet loaded at init, the 1-bit mask is `colourPictID + 1000`:

| Sheet PICT | Mask PICT |
|---|---|
| 3998 shadow | 4998 |
| 3999 glider | 4999 |
| 4000 blower | 5000 |
| 4001 furniture | 5001 |
| 4002 prizes | 5002 |
| 4003 switches | **none (5003 absent — by design)** |
| 4004 lights | 5004 |
| 4005 appliances | 5005 |
| 4006 points | 5006 |
| 4007 bands | 5007 |
| 4008 transport | 5008 |
| 4009 toast | 5009 |
| 4010 shredded | 5010 |
| 4011 balloon | 5011 |
| 4012 copter | 5012 |
| 4013 dart | 5013 |
| 4014 ball | 5014 |
| 4015 drip | 5015 |
| 4016 enemy (cobweb) | 5016 |
| 4017 fish | 5017 |
| 4018 clutter | 5018 |
| 1019 angel | **1020 (`+1`, not `+1000`)** |
| 1999 support | **none (opaque)** |

## 5. Measured resource-fork data

Everything in this section was produced by parsing
`GliderPRO/Glider PRO.r` (the 15 MB `derez` text dump of the
application's resource fork, 199843 LF-terminated lines) with a purpose-written Python
parser, and then decoding the resulting binary blobs with a from-scratch PICT v1/v2 +
PackBits decoder. No claim in this section is inferred from the C source; all of it is
observed bytes.

### 5.1 Resource inventory

**538 resources** in 36 types:

| Type | Count | Type | Count | Type | Count |
|---|---|---|---|---|---|
| `ALRT` | 26 | `BNDL` | 1 | `CDEF` | 1 |
| `CNTL` | 5 | `CURS` | 16 | `DITL` | 54 |
| `DLGX` | 1 | `DLOG` | 28 | `FREF` | 6 |
| `ICN#` | 6 | `ICON` | 35 | `MENU` | 6 |
| `PAT#` | 1 | `PICT` | 152 | `STR#` | 10 |
| `WDEF` | 2 | `WIND` | 3 | `acur` | 1 |
| `cctb` | 1 | `cicn` | 44 | `clut` | 2 |
| `crsr` | 12 | `dctb` | 20 | `demo` | 1 |
| `icl4` | 6 | `icl8` | 6 | `ics#` | 4 |
| `ics4` | 4 | `ics8` | 4 | `ictb` | 1 |
| `mctb` | 5 | `ozm5` | 1 | `snd ` | 70 |
| `vers` | 2 | `wctb` | 1 | `(total)` | 538 |

Only three of these types matter to the renderer: **PICT** (152 of them - every pixel the
game draws), **clut** (2, the 256-entry palette), and **PAT#** (1, the seven marquee
patterns). `cicn`/`ICON`/`CURS`/`crsr`/`acur` belong to the cursor and Finder-icon code;
`snd ` to the Sound Manager; `DITL`/`DLOG`/`ALRT`/`dctb`/`ictb`/`CNTL`/`mctb`/`cctb`/`wctb`
to the Dialog Manager; `demo` (1 resource, id 128) is the recorded demo input stream loaded
by `CreatePointers` (`GliderPRO/Sources/StructuresInit2.c:283-296`); `ozm5` is the Ozma
easter egg payload; `WDEF` 2048 is the coordinate-windoid definition procedure
(`GliderPRO/Sources/Coordinates.c:16`).

### 5.2 The PICT dialect Glider PRO actually uses

PICT resource header, big-endian, identical in v1 and v2:

| Offset | Size | Field | Notes |
|---|---|---|---|
| 0 | 2 | `picSize` | **signed** 16-bit. Garbage for pictures > 32767 bytes - see 5.9. |
| 2 | 8 | `picFrame` | `Rect` = top, left, bottom, right, four signed 16-bit |
| 10 | 2 | version | `0x1101` = v1 (opcodes are 1 byte) |
| 10 | 4 | version | `0x001102FF` -> v2 (opcodes are 2 bytes, word-aligned) |

(the v2 signature is the four bytes `00 11 02 FF`.)

Observed split: **36 version-1 PICTs, 116 version-2 PICTs.**

Across all 152 PICTs, only **eight distinct opcodes** ever appear:

| Opcode | Name | Occurrences | Payload handling |
|---|---|---|---|
| `0x0001` | `Clip` | 152 | 2-byte length *inclusive of itself*, then a Region; skip `length` bytes total |
| `0x001E` | `DefHilite` | 95 | zero bytes |
| `0x0090` | `BitsRect` | 17 | uncompressed rows |
| `0x0098` | `PackBitsRect` | 185 | PackBits rows |
| `0x0099` | `PackBitsRgn` | 3 | as `0x0098` plus a leading maskRgn |
| `0x00A0` | `ShortComment` | 269 | 2-byte payload |
| `0x00FF` | `OpEndPic` | 152 | terminates |
| `0x0C00` | `HeaderOp` | 116 | 24-byte payload, v2 only |

There is **no `0x009A` (`DirectBitsRect`)**, no `0x0091`/`0x009B` (`BitsRgn`/`PackBitsRgn`
direct), no text opcodes, no line/rect/oval opcodes, no `0x00A1 LongComment`. A Go PICT
decoder for this game needs to implement exactly those eight opcodes and nothing else.

Image-op geometry, in order, for `0x0090`/`0x0098`/`0x0099`:

```
uint16 rowBytesRaw          ; bit 15 set => PixMap follows, else BitMap
                            ; rowBytes = rowBytesRaw & 0x3FFF
Rect   bounds               ; 8 bytes
--- if PixMap (bit 15 set): 36 more bytes of PixMap ---
  int16 pmVersion, packType
  int32 packSize, hRes, vRes
  int16 pixelType, pixelSize, cmpCount, cmpSize
  int32 planeBytes; int32 pmTable; int32 pmReserved
  ColorTable: int32 ctSeed; uint16 ctFlags; int16 ctSize;
              (ctSize+1) * { uint16 value; uint16 r,g,b }
--- end PixMap ---
Rect   srcRect              ; 8 bytes
Rect   dstRect              ; 8 bytes
int16  mode                 ; transfer mode; always 0 (srcCopy) in this game
[ if op == 0x0099: uint16 rgnLen; skip rgnLen bytes ]
rows: for each of (bounds.bottom - bounds.top) rows:
    if op == 0x0090:  rowBytes raw bytes
    else:             byteCount, then PackBits data
                      byteCount is uint16 if rowBytes > 250, else uint8
```

The `rowBytes > 250` rule for the per-row byte-count width is the classic QuickDraw rule and
it matters: the 512-wide room backgrounds (`rowBytes` 512) use 2-byte counts, while a 16-wide
sprite strip (`rowBytes` 16) uses 1-byte counts. Getting this wrong desynchronises the whole
stream.

Observed image formats (205 image ops in 152 PICTs):

| op | pixelSize | packType | ColorTable | count | what it is |
|---|---|---|---|---|---|
| `0x0090` | 1 | - | none (BitMap) | 16 | uncompressed 1-bit masks |
| `0x0090` | 1 | 1 | 2 entries | 1 | PICT 1003 only (1-bit *PixMap*, unpacked) |
| `0x0098` | 1 | - | none (BitMap) | 21 | PackBits 1-bit masks |
| `0x0098` | 8 | 0 | 256 entries | 164 | the normal case: 8-bit indexed art |
| `0x0099` | 8 | 0 | 256 entries | 3 | 8-bit indexed art with a mask region |

Two structural facts proved by exhaustive scan:

1. **version 1 <=> no ColorTable <=> 1-bit.** The set of PICTs containing no ColorTable is
   *exactly* the set of 36 v1 PICTs: 150, 151, 1009, 1020, 1989, 1991, 1998, 3903, 3904,
   3912, 3913, 3914, 3915, 3921, 3927, 3998, 4998, 4999, 5000, 5001, 5002, 5004, 5005, 5006,
   5007, 5008, 5009, 5010, 5011, 5012, 5013, 5014, 5015, 5016, 5017, 5018. Every one of them
   is a 1-bit image, and every one of them is used as a `CopyMask` mask or as 4-bit-era art.
   (37 mono images in 36 PICTs, because PICT 4999 has two image ops.)
2. **Every 8-bit PICT carries the same palette.** All 167 8-bit image ops embed a 256-entry
   ColorTable whose 256 RGB triples are *byte-identical* to `clut` resource 128. They differ
   only in the per-entry `value` field: 144 image ops (92 PICTs) store `value == 0` for every
   entry, while 23 image ops (23 PICTs: 1000, 1001, 1002, 1004, 1006, 1007, 1008, 1010, 1012,
   1013, 1014, 1994, 1995, 1997, 2002, 2016, 2017, 3963, 3974, 3975, 3976, 3997, 3999) store
   `value == position`. Both encodings mean 'index n is entry n'. **A Go port can therefore
   ignore every embedded ColorTable and use one hard-coded 256-entry palette** (5.6).

The single exception is PICT 1003 (32x32, 282 bytes), whose 2-entry table is
`{value 0 -> FFFF/FFFF/FFFF, value 1 -> 0000/0000/0000}` - white, black.

### 5.3 Every PICT, measured

`picFrame` and byte length are read straight from the resource; `w x h` is derived.
`picSize (stored)` is the raw signed 16-bit header field, printed as-is to show where it
lies.

| PICT | picFrame (t,l,b,r) | w x h | picSize (stored) | actual bytes | ver |
|---|---|---|---|---|---|
| 150 | (0,0,63,63) | 63 x 63 | 632 | 632 | v1 |
| 151 | (0,0,63,63) | 63 x 63 | 632 | 632 | v1 |
| 153 | (0,0,100,372) | 372 x 100 | 15194 | 15194 | v2 |
| 1000 | (0,0,460,640) | 640 x 460 | -22590 | 108482 | v2 |
| 1001 | (0,0,32,431) | 431 x 32 | 5998 | 5998 | v2 |
| 1002 | (0,0,32,257) | 257 x 32 | 4986 | 4986 | v2 |
| 1003 | (0,0,32,32) | 32 x 32 | 282 | 282 | v2 |
| 1004 | (0,0,32,32) | 32 x 32 | 2938 | 2938 | v2 |
| 1005 | (0,0,32,313) | 313 x 32 | 5226 | 5226 | v2 |
| 1006 | (0,0,32,333) | 333 x 32 | 5870 | 5870 | v2 |
| 1007 | (0,0,32,385) | 385 x 32 | 5708 | 5708 | v2 |
| 1008 | (0,0,32,316) | 316 x 32 | 5580 | 5580 | v2 |
| 1009 | (0,0,28,76) | 76 x 28 | 329 | 329 | v1 |
| 1010 | (0,0,380,32) | 32 x 380 | 6850 | 6850 | v2 |
| 1011 | (0,0,216,360) | 360 x 216 | -16808 | 48728 | v2 |
| 1012 | (0,0,32,316) | 316 x 32 | 5644 | 5644 | v2 |
| 1013 | (0,0,32,289) | 289 x 32 | 5080 | 5080 | v2 |
| 1014 | (0,0,32,316) | 316 x 32 | 5582 | 5582 | v2 |
| 1015 | (0,0,54,214) | 214 x 54 | 6092 | 6092 | v2 |
| 1016 | (0,0,54,214) | 214 x 54 | 6106 | 6106 | v2 |
| 1017 | (0,0,64,256) | 256 x 64 | 9770 | 9770 | v2 |
| 1018 | (0,0,64,256) | 256 x 64 | 9742 | 9742 | v2 |
| 1019 | (0,0,44,96) | 96 x 44 | 4064 | 4064 | v2 |
| 1020 | (0,0,44,96) | 96 x 44 | 457 | 457 | v1 |
| 1021 | (0,0,460,640) | 640 x 460 | -14016 | 51520 | v2 |
| 1022 | (0,0,32,279) | 279 x 32 | 5066 | 5066 | v2 |
| 1023 | (0,0,32,280) | 280 x 32 | 5016 | 5016 | v2 |
| 1202 | (0,0,80,128) | 128 x 80 | 5826 | 5826 | v2 |
| 1211 | (0,0,80,128) | 128 x 80 | 5272 | 5272 | v2 |
| 1216 | (0,0,80,128) | 128 x 80 | 3328 | 3328 | v2 |
| 1217 | (0,0,80,128) | 128 x 80 | 2884 | 2884 | v2 |
| 1988 | (0,0,256,25) | 25 x 256 | 5900 | 5900 | v2 |
| 1989 | (0,0,448,32) | 32 x 448 | 1851 | 1851 | v1 |
| 1990 | (0,0,448,32) | 32 x 448 | 10052 | 10052 | v2 |
| 1991 | (0,0,30,330) | 330 x 30 | 477 | 477 | v1 |
| 1992 | (0,0,30,330) | 330 x 30 | 9360 | 9360 | v2 |
| 1993 | (0,0,190,330) | 330 x 190 | -21600 | 43936 | v2 |
| 1994 | (0,0,30,332) | 332 x 30 | 4972 | 4972 | v2 |
| 1995 | (0,0,460,640) | 640 x 460 | 20746 | 20746 | v2 |
| 1996 | (0,0,66,32) | 32 x 66 | 3144 | 3144 | v2 |
| 1997 | (0,0,20,1536) | 1536 x 20 | 13296 | 13296 | v2 |
| 1998 | (0,0,30,332) | 332 x 30 | 1049 | 1049 | v1 |
| 1999 | (0,0,44,512) | 512 x 44 | 16850 | 16850 | v2 |
| 2000 | (0,0,322,512) | 512 x 322 | 31838 | 31838 | v2 |
| 2001 | (0,0,322,512) | 512 x 322 | 23802 | 89338 | v2 |
| 2002 | (0,0,322,512) | 512 x 322 | -13208 | 52328 | v2 |
| 2003 | (0,0,322,512) | 512 x 322 | -22378 | 43158 | v2 |
| 2004 | (0,0,322,512) | 512 x 322 | 17740 | 83276 | v2 |
| 2005 | (0,0,322,512) | 512 x 322 | 5036 | 70572 | v2 |
| 2006 | (0,0,322,512) | 512 x 322 | -25932 | 105140 | v2 |
| 2007 | (0,0,322,512) | 512 x 322 | -3610 | 61926 | v2 |
| 2008 | (0,0,322,512) | 512 x 322 | 17386 | 82922 | v2 |
| 2009 | (0,0,322,512) | 512 x 322 | -30680 | 34856 | v2 |
| 2010 | (0,0,322,512) | 512 x 322 | -22516 | 43020 | v2 |
| 2011 | (0,0,322,512) | 512 x 322 | 15500 | 81036 | v2 |
| 2012 | (0,0,322,512) | 512 x 322 | -27746 | 37790 | v2 |
| 2013 | (0,0,322,512) | 512 x 322 | -32570 | 32966 | v2 |
| 2014 | (0,0,322,512) | 512 x 322 | 9596 | 75132 | v2 |
| 2015 | (0,0,322,512) | 512 x 322 | 23748 | 23748 | v2 |
| 2016 | (0,0,322,512) | 512 x 322 | 10640 | 76176 | v2 |
| 2017 | (0,0,322,512) | 512 x 322 | 12624 | 12624 | v2 |
| 3903 | (0,0,80,94) | 94 x 80 | 774 | 774 | v1 |
| 3904 | (0,0,80,94) | 94 x 80 | 784 | 784 | v1 |
| 3912 | (0,0,77,92) | 92 x 77 | 598 | 598 | v1 |
| 3913 | (0,0,22,96) | 96 x 22 | 212 | 212 | v1 |
| 3914 | (0,0,53,128) | 128 x 53 | 440 | 440 | v1 |
| 3915 | (0,0,59,92) | 92 x 59 | 477 | 477 | v1 |
| 3921 | (0,0,30,128) | 128 x 30 | 338 | 338 | v1 |
| 3927 | (0,0,45,54) | 54 x 45 | 450 | 450 | v1 |
| 3957 | (0,0,44,123) | 123 x 44 | 2970 | 2970 | v2 |
| 3958 | (0,0,45,54) | 54 x 45 | 3208 | 3208 | v2 |
| 3959 | (0,0,32,80) | 80 x 32 | 2668 | 2668 | v2 |
| 3960 | (0,0,62,40) | 40 x 62 | 3170 | 3170 | v2 |
| 3961 | (0,0,74,28) | 28 x 74 | 3954 | 3954 | v2 |
| 3962 | (0,0,18,144) | 144 x 18 | 2750 | 2750 | v2 |
| 3963 | (0,0,668,48) | 48 x 668 | 15258 | 15258 | v2 |
| 3964 | (0,0,51,64) | 64 x 51 | 4762 | 4762 | v2 |
| 3965 | (0,0,30,128) | 128 x 30 | 2946 | 2946 | v2 |
| 3966 | (0,0,58,80) | 80 x 58 | 4964 | 4964 | v2 |
| 3967 | (0,0,22,123) | 123 x 22 | 2800 | 2800 | v2 |
| 3968 | (0,0,57,35) | 35 x 57 | 3000 | 3000 | v2 |
| 3969 | (0,0,45,36) | 36 x 45 | 3096 | 3096 | v2 |
| 3970 | (0,0,92,63) | 63 x 92 | 4748 | 4748 | v2 |
| 3971 | (0,0,59,92) | 92 x 59 | 4702 | 4702 | v2 |
| 3972 | (0,0,58,56) | 56 x 58 | 3392 | 3392 | v2 |
| 3973 | (0,0,142,180) | 180 x 142 | 10946 | 10946 | v2 |
| 3974 | (0,0,668,48) | 48 x 668 | 17604 | 17604 | v2 |
| 3975 | (0,0,92,102) | 102 x 92 | 6552 | 6552 | v2 |
| 3976 | (0,0,668,48) | 48 x 668 | 14846 | 14846 | v2 |
| 3977 | (0,0,170,16) | 16 x 170 | 3270 | 3270 | v2 |
| 3978 | (0,0,170,16) | 16 x 170 | 3270 | 3270 | v2 |
| 3979 | (0,0,170,20) | 20 x 170 | 5386 | 5386 | v2 |
| 3980 | (0,0,170,20) | 20 x 170 | 5386 | 5386 | v2 |
| 3981 | (0,0,322,16) | 16 x 322 | 3794 | 3794 | v2 |
| 3982 | (0,0,322,16) | 16 x 322 | 3798 | 3798 | v2 |
| 3983 | (0,0,322,144) | 144 x 322 | 15272 | 15272 | v2 |
| 3984 | (0,0,322,144) | 144 x 322 | 15282 | 15282 | v2 |
| 3985 | (0,0,80,94) | 94 x 80 | 3396 | 3396 | v2 |
| 3986 | (0,0,80,94) | 94 x 80 | 4056 | 4056 | v2 |
| 3987 | (0,0,80,144) | 144 x 80 | 4294 | 4294 | v2 |
| 3988 | (0,0,33,64) | 64 x 33 | 3160 | 3160 | v2 |
| 3989 | (0,0,53,128) | 128 x 53 | 5144 | 5144 | v2 |
| 3990 | (0,0,22,96) | 96 x 22 | 2814 | 2814 | v2 |
| 3991 | (0,0,172,64) | 64 x 172 | 5726 | 5726 | v2 |
| 3992 | (0,0,77,92) | 92 x 77 | 6124 | 6124 | v2 |
| 3993 | (0,0,212,64) | 64 x 212 | 3740 | 3740 | v2 |
| 3994 | (0,0,276,72) | 72 x 276 | 5832 | 5832 | v2 |
| 3995 | (0,0,107,74) | 74 x 107 | 4706 | 4706 | v2 |
| 3996 | (0,0,267,160) | 160 x 267 | 5986 | 5986 | v2 |
| 3997 | (0,0,267,160) | 160 x 267 | 10950 | 10950 | v2 |
| 3998 | (0,0,18,48) | 48 x 18 | 167 | 167 | v1 |
| 3999 | (0,0,668,48) | 48 x 668 | 18688 | 18688 | v2 |
| 4000 | (0,0,402,48) | 48 x 402 | 11800 | 11800 | v2 |
| 4001 | (0,0,221,64) | 64 x 221 | 9416 | 9416 | v2 |
| 4002 | (0,0,378,88) | 88 x 378 | 15432 | 15432 | v2 |
| 4003 | (0,0,104,32) | 32 x 104 | 4650 | 4650 | v2 |
| 4004 | (0,0,126,72) | 72 x 126 | 5284 | 5284 | v2 |
| 4005 | (0,0,269,80) | 80 x 269 | 12578 | 12578 | v2 |
| 4006 | (0,0,120,24) | 24 x 120 | 4682 | 4682 | v2 |
| 4007 | (0,0,18,16) | 16 x 18 | 2384 | 2384 | v2 |
| 4008 | (0,0,32,56) | 56 x 32 | 2740 | 2740 | v2 |
| 4009 | (0,0,174,32) | 32 x 174 | 4768 | 4768 | v2 |
| 4010 | (0,0,35,40) | 40 x 35 | 3308 | 3308 | v2 |
| 4011 | (0,0,240,24) | 24 x 240 | 5618 | 5618 | v2 |
| 4012 | (0,0,300,32) | 32 x 300 | 5770 | 5770 | v2 |
| 4013 | (0,0,76,64) | 64 x 76 | 4434 | 4434 | v2 |
| 4014 | (0,0,64,32) | 32 x 64 | 3526 | 3526 | v2 |
| 4015 | (0,0,72,16) | 16 x 72 | 2822 | 2822 | v2 |
| 4016 | (0,0,33,36) | 36 x 33 | 2752 | 2752 | v2 |
| 4017 | (0,0,128,16) | 16 x 128 | 3902 | 3902 | v2 |
| 4018 | (0,0,69,128) | 128 x 69 | 4864 | 4864 | v2 |
| 4998 | (0,0,18,48) | 48 x 18 | 167 | 167 | v1 |
| 4999 | (0,0,668,48) | 48 x 668 | 4102 | 4102 | v1 |
| 5000 | (0,0,402,48) | 48 x 402 | 2471 | 2471 | v1 |
| 5001 | (0,0,221,64) | 64 x 221 | 1908 | 1908 | v1 |
| 5002 | (0,0,378,88) | 88 x 378 | 4500 | 4500 | v1 |
| 5004 | (0,0,126,72) | 72 x 126 | 1334 | 1334 | v1 |
| 5005 | (0,0,268,80) | 80 x 268 | 2124 | 2124 | v1 |
| 5006 | (0,91,120,115) | 24 x 120 | 539 | 539 | v1 |
| 5007 | (0,0,18,16) | 16 x 18 | 95 | 95 | v1 |
| 5008 | (0,0,32,56) | 56 x 32 | 313 | 313 | v1 |
| 5009 | (0,0,174,32) | 32 x 174 | 755 | 755 | v1 |
| 5010 | (195,0,230,40) | 40 x 35 | 269 | 269 | v1 |
| 5011 | (0,0,240,24) | 24 x 240 | 1019 | 1019 | v1 |
| 5012 | (0,0,300,32) | 32 x 300 | 1259 | 1259 | v1 |
| 5013 | (0,0,76,64) | 64 x 76 | 702 | 702 | v1 |
| 5014 | (0,0,64,32) | 32 x 64 | 315 | 315 | v1 |
| 5015 | (0,0,72,16) | 16 x 72 | 203 | 203 | v1 |
| 5016 | (0,0,33,36) | 36 x 33 | 257 | 257 | v1 |
| 5017 | (0,0,128,16) | 16 x 128 | 315 | 315 | v1 |
| 5018 | (0,0,69,128) | 128 x 69 | 1026 | 1026 | v1 |
| 10000 | (0,0,34,72) | 72 x 34 | 4182 | 4182 | v2 |

### 5.4 PICTs with more than one image op

19 PICTs are stored as several horizontal bands rather than one image. A decoder that stops
after the first `PackBitsRect` renders the top slice of these and leaves the rest blank -
which for the room backgrounds means the top 100 of 322 rows.

| PICT | image ops | what it is |
|---|---|---|
| 1011 | 2 | 360x216 dialog art |
| 1021 | 6 | 640x460 Milky Way (game-over background) |
| 1993 | 2 | 330x190 banner page top |
| 2000 | 4 | room background `kSimpleRoom` |
| 2001 | 4 | `kPaneledRoom` |
| 2003 | 4 | `kChildsRoom` |
| 2004 | 4 | `kAsianRoom` |
| 2005 | 4 | `kUnfinishedRoom` |
| 2006 | 4 | `kSwingersRoom` |
| 2007 | 4 | `kBathroom` |
| 2008 | 4 | `kLibrary` |
| 2009 | 4 | `kGarden` (= `kFirstOutdoorBack`) |
| 2010 | 4 | `kSkywalk` |
| 2011 | 4 | `kDirt` |
| 2012 | 4 | `kMeadow` |
| 2013 | 4 | `kField` |
| 2014 | 4 | `kRoof` |
| 2015 | 4 | `kSky` |
| 4999 | 2 | 48x668 glider mask |

The three room backgrounds that are *not* banded are 2002 (`kBasement`), 2016
(`kStratosphere`) and 2017 (`kStars`) - each is a single `PackBitsRect` covering all 322 rows.

PICT 2000's four bands, exactly as stored (note `bounds == srcRect == dstRect` in every one,
`mode == 0` = `srcCopy`, `rowBytes == 512`):

| band | bounds / srcRect / dstRect | rows |
|---|---|---|
| 0 | (0, 0, 100, 512) | 100 |
| 1 | (100, 0, 200, 512) | 100 |
| 2 | (200, 0, 300, 512) | 100 |
| 3 | (300, 0, 322, 512) | 22 |

So the correct decode loop is: allocate `picFrame` width x height, then for every image op
blit its rows to `dstRect` translated by `-picFrame.topLeft`. The bands tile vertically and
do not overlap; 100 + 100 + 100 + 22 = 322.

### 5.5 PICT 2000 assembled - proof that the eight tile columns are distinct

Decoding PICT 2000 into a 512x322 index buffer and hashing each 64-pixel-wide column
(md5, first 12 hex digits):

| tile column | x range | md5 | row 0 first 8 indices | row 321 first 8 indices |
|---|---|---|---|---|
| 0 | 0-63 | `59c8f3e167f9` | 223, 254, 223, 254, 223, 253, 9, 254 | 254, 223, 254, 223, 254, 253, 9, 253 |
| 1 | 64-127 | `2498ac4c522e` | 2, 2, 2, 2, 2, 2, 2, 2 | 94, 94, 94, 94, 94, 94, 94, 94 |
| 2 | 128-191 | `d866bc14d5c8` | 2, 2, 2, 2, 2, 2, 2, 2 | 94, 94, 94, 94, 94, 94, 94, 94 |
| 3 | 192-255 | `24f3fcd6d273` | 2, 2, 2, 2, 2, 2, 2, 2 | 94, 94, 94, 94, 94, 94, 94, 94 |
| 4 | 256-319 | `d6f04e25d98c` | 2, 2, 2, 2, 2, 2, 2, 2 | 94, 94, 94, 94, 94, 94, 94, 94 |
| 5 | 320-383 | `0c34c9322bb3` | 2, 2, 2, 2, 2, 2, 2, 2 | 94, 94, 94, 94, 94, 94, 94, 94 |
| 6 | 384-447 | `e67296a9f84f` | 2, 2, 2, 2, 2, 2, 2, 2 | 94, 94, 94, 94, 94, 94, 94, 94 |
| 7 | 448-511 | `b1a1458a2c5e` | 2, 2, 2, 2, 2, 2, 2, 2 | 94, 94, 94, 94, 94, 94, 94, 94 |

All eight md5s differ, so all eight tiles are genuinely different art - the tile index array
in a room really does select between eight distinct 64x322 vertical strips. Columns 1-7 share
the same wall colours (index 2 = `#FFFFCC` at the ceiling, index 94 = `#996633` at the floor);
column 0 is the left-wall variant, which is why `DetermineRoomOpenings` tests `leftTile == 0`
for indoor rooms (`GliderPRO/Sources/Room.c:816-925`).

Coarse ASCII thumbnail of the assembled 512x322 image (one character per 8x14 block, sampled
at the block centre; `#` = index > 200, `+` > 128, `.` > 40, space otherwise):

```
|##                                                            +#|
|#...                                                         # #|
|#.+.................+......++.....................++++.......  #|
|#.+.................+......++......#..............###........  #|
|#.+.................+......++......................# ........  #|
|#.+.................+......++.....................+ .........  #|
|#.+.................+......++................................  #|
|#.+.................+......++................................  #|
|#.+.................+......++................................  #|
|#.+.................+......++................................  #|
|#.+.................+......++................................  #|
|#.+.................+......++................................  #|
|#.+.................+......++................................  #|
|#.+.................+......++................................  #|
|#.+.................+......#+................................  #|
|#.+.................+......++................................  #|
|#.+.................+......++................................  #|
|#.+.................+......++................................  #|
|#.+.................+......++................................  #|
|#.+.................+......++................................  #|
|#.+#+++++++++++++++++++++++##+++++++++++++++++++++++++++++++++ #|
|#+#+.........................................................++#|
|#+............................................................+#|
```

(The vertical bars at columns 0 and 63 are the left and right walls; the horizontal band near
the bottom is the floor line at about y = 300.)

### 5.6 The palette: `clut` 128

| Property | Observed value |
|---|---|
| resource length | 2056 bytes (8-byte header + 256 * 8) |
| `ctSeed` | 0 |
| `ctFlags` | 0x0000 |
| `ctSize` | 255 (so 256 entries) |
| `clut 129` | **byte-identical to `clut 128`** |
| entry `value` fields | sequential 0..255 |

The palette is the standard Macintosh 8-bit system palette, and its structure is fully
mechanical - verified entry by entry with zero mismatches:

* **Indices 0-214: a 6x6x6 colour cube.** For index `i`, `ri = i / 36`, `gi = (i % 36) / 6`,
  `bi = i % 6`, and each 16-bit channel is `0xFFFF - 0x3333 * component`, i.e. the six levels
  are `FFFF, CCCC, 9999, 6666, 3333, 0000` (8-bit: `FF, CC, 99, 66, 33, 00`). Equivalently:
  `index = 36*ri + 6*gi + bi` where 0 = brightest. Verified: 215 of 215 entries match exactly.
  (The cube is *truncated*: a full 6^3 cube is 216 entries, but index 215 - which would be
  black - is stolen for the start of the red ramp, and black lives at 255 instead.)
* **Indices 215-224: pure red ramp.** `(level, 0, 0)` for level in
  `EEEE, DDDD, BBBB, AAAA, 8888, 7777, 5555, 4444, 2222, 1111`.
* **Indices 225-234: pure green ramp**, same ten levels in the green channel.
* **Indices 235-244: pure blue ramp**, same ten levels in the blue channel.
* **Indices 245-254: gray ramp**, same ten levels in all three channels.
* **Index 255: black** `(0,0,0)`.

The ten ramp levels in 8 bits are `EE, DD, BB, AA, 88, 77, 55, 44, 22, 11` - note they skip
`CC, 99, 66, 33` precisely because those already exist in the cube.

Every palette-index constant used by the drawing code, with its measured RGB:

| Constant | Index | RGB | Where defined |
|---|---|---|---|
| `k8WhiteColor` | 0 | `#FFFFFF` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `kYellowColor` | 5 | `#FFFF00` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `kGoldColor` | 11 | `#FFCC00` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `kRedOrangeColor8` | 23 | `#FF6600` | `GliderDefines.h:542` |
| `k8RedColor` | 35 | `#FF0000` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `kPaleVioletColor` | 42 | `#CCCCFF` | `ObjectDraw2.c:19-37` |
| `k8LtstGray3Color` | 43 | `#CCCCCC` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8LtTanColor` | 52 | `#CC9933` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8BambooColor` | 53 | `#CC9900` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `kDarkFleshColor` | 58 | `#CC6633` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8OrangeColor` | 59 | `#CC6600` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8TanColor` | 94 | `#996633` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8PissYellowColor` | 95 | `#996600` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8PumpkinColor` | 101 | `#993300` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8BrownColor` | 137 | `#663300` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8Red4Color` | 143 | `#660000` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8SkyColor` | 150 | `#33CCFF` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8EarthBlueColor` | 170 | `#333399` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8DkGray3Color` | 172 | `#333333` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8DkRedColor` | 222 | `#440000` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8DkRed2Color` | 223 | `#220000` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `kIntenseGreenColor` | 225 | `#00EE00` | `ObjectDraw2.c:19-37` |
| `kIntenseBlueColor` | 235 | `#0000EE` | `ObjectDraw2.c:19-37` |
| `(none - literal 244)` | 244 | `#000011` | literal at `GameOver.c:66`, `GameOver.c:82` |
| `k8LtstGrayColor` | 245 | `#EEEEEE` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8LtstGray2Color` | 246 | `#DDDDDD` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8LtstGray4Color` | 247 | `#BBBBBB` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8LtstGray5Color` | 248 | `#AAAAAA` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8LtGrayColor` | 249 | `#888888` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8GrayColor` | 250 | `#777777` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8Gray2Color` | 251 | `#555555` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8DkGrayColor` | 252 | `#444444` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8DkGray2Color` | 253 | `#222222` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8DkstGrayColor` | 254 | `#111111` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |
| `k8BlackColor` | 255 | `#000000` | `ObjectDraw.c:16-48`, `ObjectDraw2.c:19-37` |

Two extra depth-4 palette constants exist for the 16-colour fallback path:
`kGrayBackgroundColor 251` / `kGrayBackgroundColor4 10` (`GliderPRO/Sources/Scoreboard.c:15-16`).
At 4 bits the palette is the standard 16-entry Mac CLUT and the code hard-codes small indices
(9, 10, 11, 13) instead.

Complete 256-entry palette as 24-bit RGB (index: hex), which is all a Go port needs:

| idx | RGB | idx | RGB | idx | RGB | idx | RGB |
|---|---|---|---|---|---|---|---|
| 0 | `FFFFFF` | 64 | `CC3333` | 128 | `666699` | 192 | `0099FF` |
| 1 | `FFFFCC` | 65 | `CC3300` | 129 | `666666` | 193 | `0099CC` |
| 2 | `FFFF99` | 66 | `CC00FF` | 130 | `666633` | 194 | `009999` |
| 3 | `FFFF66` | 67 | `CC00CC` | 131 | `666600` | 195 | `009966` |
| 4 | `FFFF33` | 68 | `CC0099` | 132 | `6633FF` | 196 | `009933` |
| 5 | `FFFF00` | 69 | `CC0066` | 133 | `6633CC` | 197 | `009900` |
| 6 | `FFCCFF` | 70 | `CC0033` | 134 | `663399` | 198 | `0066FF` |
| 7 | `FFCCCC` | 71 | `CC0000` | 135 | `663366` | 199 | `0066CC` |
| 8 | `FFCC99` | 72 | `99FFFF` | 136 | `663333` | 200 | `006699` |
| 9 | `FFCC66` | 73 | `99FFCC` | 137 | `663300` | 201 | `006666` |
| 10 | `FFCC33` | 74 | `99FF99` | 138 | `6600FF` | 202 | `006633` |
| 11 | `FFCC00` | 75 | `99FF66` | 139 | `6600CC` | 203 | `006600` |
| 12 | `FF99FF` | 76 | `99FF33` | 140 | `660099` | 204 | `0033FF` |
| 13 | `FF99CC` | 77 | `99FF00` | 141 | `660066` | 205 | `0033CC` |
| 14 | `FF9999` | 78 | `99CCFF` | 142 | `660033` | 206 | `003399` |
| 15 | `FF9966` | 79 | `99CCCC` | 143 | `660000` | 207 | `003366` |
| 16 | `FF9933` | 80 | `99CC99` | 144 | `33FFFF` | 208 | `003333` |
| 17 | `FF9900` | 81 | `99CC66` | 145 | `33FFCC` | 209 | `003300` |
| 18 | `FF66FF` | 82 | `99CC33` | 146 | `33FF99` | 210 | `0000FF` |
| 19 | `FF66CC` | 83 | `99CC00` | 147 | `33FF66` | 211 | `0000CC` |
| 20 | `FF6699` | 84 | `9999FF` | 148 | `33FF33` | 212 | `000099` |
| 21 | `FF6666` | 85 | `9999CC` | 149 | `33FF00` | 213 | `000066` |
| 22 | `FF6633` | 86 | `999999` | 150 | `33CCFF` | 214 | `000033` |
| 23 | `FF6600` | 87 | `999966` | 151 | `33CCCC` | 215 | `EE0000` |
| 24 | `FF33FF` | 88 | `999933` | 152 | `33CC99` | 216 | `DD0000` |
| 25 | `FF33CC` | 89 | `999900` | 153 | `33CC66` | 217 | `BB0000` |
| 26 | `FF3399` | 90 | `9966FF` | 154 | `33CC33` | 218 | `AA0000` |
| 27 | `FF3366` | 91 | `9966CC` | 155 | `33CC00` | 219 | `880000` |
| 28 | `FF3333` | 92 | `996699` | 156 | `3399FF` | 220 | `770000` |
| 29 | `FF3300` | 93 | `996666` | 157 | `3399CC` | 221 | `550000` |
| 30 | `FF00FF` | 94 | `996633` | 158 | `339999` | 222 | `440000` |
| 31 | `FF00CC` | 95 | `996600` | 159 | `339966` | 223 | `220000` |
| 32 | `FF0099` | 96 | `9933FF` | 160 | `339933` | 224 | `110000` |
| 33 | `FF0066` | 97 | `9933CC` | 161 | `339900` | 225 | `00EE00` |
| 34 | `FF0033` | 98 | `993399` | 162 | `3366FF` | 226 | `00DD00` |
| 35 | `FF0000` | 99 | `993366` | 163 | `3366CC` | 227 | `00BB00` |
| 36 | `CCFFFF` | 100 | `993333` | 164 | `336699` | 228 | `00AA00` |
| 37 | `CCFFCC` | 101 | `993300` | 165 | `336666` | 229 | `008800` |
| 38 | `CCFF99` | 102 | `9900FF` | 166 | `336633` | 230 | `007700` |
| 39 | `CCFF66` | 103 | `9900CC` | 167 | `336600` | 231 | `005500` |
| 40 | `CCFF33` | 104 | `990099` | 168 | `3333FF` | 232 | `004400` |
| 41 | `CCFF00` | 105 | `990066` | 169 | `3333CC` | 233 | `002200` |
| 42 | `CCCCFF` | 106 | `990033` | 170 | `333399` | 234 | `001100` |
| 43 | `CCCCCC` | 107 | `990000` | 171 | `333366` | 235 | `0000EE` |
| 44 | `CCCC99` | 108 | `66FFFF` | 172 | `333333` | 236 | `0000DD` |
| 45 | `CCCC66` | 109 | `66FFCC` | 173 | `333300` | 237 | `0000BB` |
| 46 | `CCCC33` | 110 | `66FF99` | 174 | `3300FF` | 238 | `0000AA` |
| 47 | `CCCC00` | 111 | `66FF66` | 175 | `3300CC` | 239 | `000088` |
| 48 | `CC99FF` | 112 | `66FF33` | 176 | `330099` | 240 | `000077` |
| 49 | `CC99CC` | 113 | `66FF00` | 177 | `330066` | 241 | `000055` |
| 50 | `CC9999` | 114 | `66CCFF` | 178 | `330033` | 242 | `000044` |
| 51 | `CC9966` | 115 | `66CCCC` | 179 | `330000` | 243 | `000022` |
| 52 | `CC9933` | 116 | `66CC99` | 180 | `00FFFF` | 244 | `000011` |
| 53 | `CC9900` | 117 | `66CC66` | 181 | `00FFCC` | 245 | `EEEEEE` |
| 54 | `CC66FF` | 118 | `66CC33` | 182 | `00FF99` | 246 | `DDDDDD` |
| 55 | `CC66CC` | 119 | `66CC00` | 183 | `00FF66` | 247 | `BBBBBB` |
| 56 | `CC6699` | 120 | `6699FF` | 184 | `00FF33` | 248 | `AAAAAA` |
| 57 | `CC6666` | 121 | `6699CC` | 185 | `00FF00` | 249 | `888888` |
| 58 | `CC6633` | 122 | `669999` | 186 | `00CCFF` | 250 | `777777` |
| 59 | `CC6600` | 123 | `669966` | 187 | `00CCCC` | 251 | `555555` |
| 60 | `CC33FF` | 124 | `669933` | 188 | `00CC99` | 252 | `444444` |
| 61 | `CC33CC` | 125 | `669900` | 189 | `00CC66` | 253 | `222222` |
| 62 | `CC3399` | 126 | `6666FF` | 190 | `00CC33` | 254 | `111111` |
| 63 | `CC3366` | 127 | `6666CC` | 191 | `00CC00` | 255 | `000000` |

### 5.7 `PAT#` 128 - the seven marquee patterns

58 bytes total: `int16 count = 7`, then 7 x 8 bytes. Each `Pattern` is 8 rows of 8 bits,
MSB = leftmost pixel, 1 = black.

| Index (1-based, as passed to `GetIndPattern`) | Bytes | Row 0 bit pattern |
|---|---|---|
| 1 | `F8 F1 E3 C7 8F 1F 3E 7C` | `#####...` |
| 2 | `3E 7C F8 F1 E3 C7 8F 1F` | `..#####.` |
| 3 | `1F 3E 7C F8 F1 E3 C7 8F` | `...#####` |
| 4 | `8F 1F 3E 7C F8 F1 E3 C7` | `#...####` |
| 5 | `C7 8F 1F 3E 7C F8 F1 E3` | `##...###` |
| 6 | `E3 C7 8F 1F 3E 7C F8 F1` | `###...##` |
| 7 | `F1 E3 C7 8F 1F 3E 7C F8` | `####...#` |

Read as a group these are one diagonal 'barber pole'. **Verified with python3:**

* **Within** a pattern, row *r+1* is row *r* rotated **left by exactly 1 bit** — true for all 7
  patterns, 8/8 rows each. So every pattern is a 45° diagonal of 5 set bits and 3 clear bits.
* **Between** patterns, pattern *n+1* is pattern *n* shifted **down** by *s* rows, cyclically, with
  the measured shift sequence

  | step | 1→2 | 2→3 | 3→4 | 4→5 | 5→6 | 6→7 | 7→1 |
  |---|---|---|---|---|---|---|---|
  | rows | **2** | 1 | 1 | 1 | 1 | 1 | 1 |

  summing to **8** — exactly one full period of the 8-row diagonal.

So the cycle is seven steps covering eight rows: six single steps plus one double step at 1→2. The
marching ants therefore advance unevenly, with one visible "hitch" per 7-frame cycle. See Open
questions. Stepping `index` 0..6 and back to 0 every frame produces the animation
(`GliderPRO/Sources/Marquee.c:35-51`).

### 5.8 `WIND` resources - the real window sizes

| id | `boundsRect` (t,l,b,r) | w x h | `procID` | visible | goAway | title |
|---|---|---|---|---|---|---|
| 128 | (0, 0, 384, 512) | 512 x 384 | 2 (`plainDBox`) | 0 | 0 | "Main Window" |
| 129 | (0, 0, 322, 512) | 512 x 322 | 4 (`noGrowDocProc`) | 0 | 0 | "Main Window" |
| 130 | (0, 0, 20, 640) | 640 x 20 | 2 (`plainDBox`) | 0 | 0 | "New Window" |

`kMainWindowID 128`, `kEditWindowID 129`, `kMenuWindowID 130`
(`GliderPRO/Sources/MainWindow.c:16-18`). All three are created invisible and then resized
and shown by `OpenMainWindow`, so the stored sizes only matter as a sanity check:

* WIND 128's 512x384 is never used as-is - `OpenMainWindow` immediately does `SizeWindow` to
  `mainWindowRect` (screen width x screen height - 20) and `MoveWindow` to `(screen.left,
  screen.top + 20)` (`GliderPRO/Sources/MainWindow.c:222-229`). `plainDBox` means no title
  bar, no border, no drag region - a full-bleed play surface.
* WIND 129's **512x322 is exactly `kRoomWide x kTileHigh`** and is used as-is in edit mode
  (`QSetRect(&mainWindowRect, 0, 0, 512, 322)`, `GliderPRO/Sources/MainWindow.c:191`).
* WIND 130's height of 20 is the menu-bar cover strip; its width is overridden to the screen
  width. Note 20 here is `kScoreboardTall` *and* the assumed `thisMac.menuHigh` - see 2.2.

### 5.9 Resource anomalies a porter will trip over

| # | Anomaly | Evidence | Consequence |
|---|---|---|---|
| 1 | **`picSize` is signed and wrong for 19 PICTs.** | 1000 stores -22590 but is 108482 bytes; 1011 -16808 / 48728; 1021 -14016 / 51520; 1993 -21600 / 43936; 2001 23802 / 89338; 2002 -13208 / 52328; 2003 -22378 / 43158; 2004 17740 / 83276; 2005 5036 / 70572; 2006 -25932 / 105140; 2007 -3610 / 61926; 2008 17386 / 82922; 2009 -30680 / 34856; 2010 -22516 / 43020; 2011 15500 / 81036; 2012 -27746 / 37790; 2013 -32570 / 32966; 2014 9596 / 75132; 2016 10640 / 76176 | **Never trust `picSize`. Use the resource length.** |
| 2 | **PICT 5003 does not exist.** | full PICT id list: 150, 151, 153, 1000-1023, 1202, 1211, 1216, 1217, 1988-2017, 3903, 3904, 3912-3915, 3921, 3927, 3957-4018, 4998-5002, **5004**-5018, 10000 | 5003 would be the depth-4 twin of 4003 (switches). Switches are drawn with a plain rectangular `srcCopy` and never need a mask or a 4-bit variant, so nothing loads it (`GliderPRO/Sources/ObjectDraw2.c:301-372`; `InitSwitches` creates no mask GWorld, `GliderPRO/Sources/StructuresInit.c:447-486`; `CheckMemorySize` has no switches-mask line, `GliderPRO/Sources/Environ.c:614`). |
| 3 | **PICT 4001 (64x221) is smaller than its GWorld (64x278).** | measured `picFrame` (0,0,221,64) vs `QSetRect(&furnitureSrcRect, 0, 0, 64, 278)` (`GliderPRO/Sources/StructuresInit.c:301`) | rows 221-277 of `furnitureSrcMap` are whatever `NewGWorld` left there. In practice no `srcRects[]` entry reaches past y=221; a Go port should zero-fill. |
| 4 | **PICT 5005 (80x268) is one row shorter than PICT 4005 (80x269).** | measured | harmless: the depth-4 art is only ever `DrawPicture`d into the same 80x269 GWorld and the bottom row stays background. |
| 5 | **PICTs 5006 and 5010 have non-zero `picFrame` origins.** | 5006 = (0, 91, 120, 115); 5010 = (195, 0, 230, 40) | harmless *only because* `LoadGraphic` normalises with `OffsetRect(&bounds, -bounds.left, -bounds.top)` before `DrawPicture` (`GliderPRO/Sources/Utilities.c:317-337`). `LoadScaledGraphic` does **not** normalise - it passes the caller's rect - so it is immune too. A Go loader must replicate the normalisation. |
| 6 | **PICT 1997 (the scoreboard) is 1536 px wide** - `kMaxViewWidth`. | measured `picFrame` (0,0,20,1536) | only the leftmost `screenWidth` pixels are ever blitted. The GWorld is `screenWidth x 20`. |
| 7 | **PICT 1995 (high-scores background) is 640x460 but is drawn into a 640x480 rect.** | `QSetRect(&tempRect, 0, 0, 640, 480); LoadScaledGraphic(kStarPictID, &tempRect)` (`GliderPRO/Sources/HighScores.c:64-67`) | vertical stretch of 480/460 = 1.0435. A faithful port must stretch it too. |
| 8 | **20 of the 28 mask PICT ids 3900-3927 do not exist.** | present: 3903, 3904, 3912, 3913, 3914, 3915, 3921, 3927. absent: 3900, 3901, 3902, 3905, 3906, 3907, 3908, 3909, 3910, 3911, 3916, 3917, 3918, 3919, 3920, 3922, 3923, 3924, 3925, 3926 | the 20 absent ids correspond exactly to the 20 object types drawn by `DrawPictSansWhiteObject`, which keys on white instead of using a mask (`GliderPRO/Sources/ObjectDraw2.c:1302-1408`). The `#define`s for the absent masks are still in the source (`GliderPRO/Sources/ObjectDraw2.c:39-66`) - dead constants. |
| 9 | **PICT 1020 is the angel mask, breaking the `+1000` convention.** | `GetPicture(kAngelPictID)` = 1019, mask `GetPicture(kAngelPictID + 1)` = 1020 (`GliderPRO/Sources/StructuresInit2.c:130,134`); 1020 measured as 96x44 v1, 457 bytes (1-bit) | every other mask is `art + 1000`. |
| 10 | **PICT 1003 is the only 1-bit *PixMap*.** | packType 1, 2-entry ColorTable | a decoder that assumes 'PixMap => 8bpp' will crash on it. |

## 6. Room background composition — building `backSrcMap`

`backSrcMap` is rebuilt from scratch exactly once per room change. Everything in this section
runs at that moment and never during a frame.

### 6.1 Entry point: `ReadyLevel`

`GliderPRO/Sources/RoomGraphics.c:402-418`:

```
1  NilSavedMaps()                       /* dispose all 24 savedMaps GWorlds        */
2  #ifdef COMPILEQT
3    if (thisMac.hasQT && hasMovie && tvInRoom):
4        tvInRoom = false; tvWithMovieNumber = -1; StopMovie(theMovie)
5  #endif
6  DetermineRoomOpenings()              /* sets leftOpen/rightOpen/topOpen/bottomOpen */
7  DrawLocale()                         /* <- all of section 6                     */
8  InitGarbageRects()                   /* clears both dirty-rect queues           */
```

### 6.2 `DrawLocale` — the room compositor

`GliderPRO/Sources/RoomGraphics.c:44-130`. Numbered exactly as the C flows:

```
 1  ZeroFlamesAndTheLike()      /* numFlames = numTikiFlames = numCoals = 0 ...   */
 2  ZeroDinahs()                /* numDinahs = 0                                  */
 3  KillAllBands()
 4  ZeroMirrorRegion()          /* hasMirror = false, empty mirrorRgn             */
 5  ZeroTriggers()
 6  numTempManholes = 0
 7  FlushAnyTriggerPlaying(); DumpTriggerSound()
 8  tvInRoom = false;  tvWithMovieNumber = -1
 9  roomV = thisHouse->rooms[thisRoomNumber].floor         /* :64                  */
10  for i in 0..8:
11      localNumbers[i] = GetNeighborRoomNumber(i)         /* :69                  */
12      isStructure[i]  = IsRoomAStructure(localNumbers[i])/* :70                  */
13  ListAllLocalObjects()                                  /* :72                  */
14  GetGWorld(&wasCPort, &wasWorld);  SetGWorld(backSrcMap, nil)
15  PaintRect(&backSrcRect)            /* :76  fills the WHOLE back map with the   */
                                       /*      port's pen pattern+fgColor = BLACK  */
16  if numNeighbors > 3:               /* :78  i.e. only in 9-room mode            */
17      for nb in [NW, NE, N] with elevation roomV+1:
18          numLights = GetNumberOfLights(localNumbers[nb])
19          DrawRoomBackground(localNumbers[nb], nb, roomV+1)
20          DrawARoomsObjects(nb, false)
21      for nb in [SW, SE, S] with elevation roomV-1:
22          numLights = GetNumberOfLights(localNumbers[nb])
23          DrawRoomBackground(localNumbers[nb], nb, roomV-1)
24          DrawARoomsObjects(nb, false)
25  if numNeighbors > 1:                /* :105 i.e. 3-room and 9-room mode         */
26      for nb in [W, E] with elevation roomV:
27          numLights = GetNumberOfLights(localNumbers[nb])
28          DrawRoomBackground(localNumbers[nb], nb, roomV)
29          DrawARoomsObjects(nb, false)
30          DrawLighting()             /* no-op, see 6.9                           */
31  numLights = GetNumberOfLights(localNumbers[kCentralRoom])   /* :118            */
32  DrawRoomBackground(localNumbers[kCentralRoom], kCentralRoom, roomV)
33  DrawARoomsObjects(kCentralRoom, false)
34  DrawLighting()
35  if numNeighbors > 3:  DrawFloorSupport()                    /* :123-124        */
36  RestoreWorkMap()                   /* :125  back -> work, whole map            */
37  shadowVisible = IsShadowVisible()  /* :126                                     */
38  takingTheStairs = false            /* :127                                     */
39  SetGWorld(wasCPort, wasWorld)
```

Three things a porter must not reorder:

* **The central room is drawn LAST** (steps 31-34), after all eight neighbours. Neighbour rooms
  are 512 px wide and are placed exactly one room-width left/right, so they cannot overlap the
  centre horizontally; but a neighbour's *objects* can (a 144-wide door in the west room reaching
  east). Drawing the centre last guarantees the playfield wins.
* **`numLights` is a global that is re-assigned before each room** and read inside both
  `DrawRoomBackground` and `DrawARoomsObjects`. It is not a parameter. A Go port that passes it
  as a parameter is fine, but must pass the *neighbour's* light count, not the centre's.
* **Floor supports are drawn after everything else** (step 35) so they overlay the ceiling and
  floor of the rooms above/below.

`GetNeighborRoomNumber(which)` (`GliderPRO/Sources/Room.c:562-635`) maps the 9 slot indices to
(hDelta, vDelta) then does a linear search over all `numberRooms` for a room with matching
`suite`/`floor`, returning `kRoomIsEmpty` (−1) if none:

| slot | constant | value | hDelta (suite) | vDelta (floor) |
|---|---|---|---|---|
| centre | `kCentralRoom` | 0 | 0 | 0 |
| up | `kNorthRoom` | 1 | 0 | **+1** |
| up-right | `kNorthEastRoom` | 2 | +1 | +1 |
| right | `kEastRoom` | 3 | +1 | 0 |
| down-right | `kSouthEastRoom` | 4 | +1 | −1 |
| down | `kSouthRoom` | 5 | 0 | −1 |
| down-left | `kSouthWestRoom` | 6 | −1 | −1 |
| left | `kWestRoom` | 7 | −1 | 0 |
| up-left | `kNorthWestRoom` | 8 | −1 | +1 |

Note the sign flip: `floor` **increases upward** in house data, but screen y increases downward,
which is why `localRoomsDest[kNorthRoom]` is offset by `-kVertLocalOffset` (see 2.5).

### 6.3 `numNeighbors` — the viewport size

`numNeighbors` is a global `short` (`GliderPRO/Sources/RoomGraphics.c:31`) holding **1, 3 or 9**.

| Value | Rooms drawn | `justRoomsRect` |
|---|---|---|
| 1 | centre only | the central room band only |
| 3 | W, centre, E | the central room band only |
| 9 | all nine | the whole `houseRect` |

Defaults and the only writers:

| Site | Statement |
|---|---|
| `GliderPRO/Sources/Settings.c:888` | `numNeighbors = 9;` (`BringUpDefaultPrefs`) |
| `GliderPRO/Sources/Settings.c:1257` | `numNeighbors = 9;` (hard default) |
| `GliderPRO/Sources/Settings.c:1153` | `numNeighbors = 1;` (prefs radio button) |
| `GliderPRO/Sources/Settings.c:1164` | `numNeighbors = 3;` |
| `GliderPRO/Sources/Settings.c:1176` | `numNeighbors = 9;` |
| `GliderPRO/Sources/Settings.c:1146` | `numNeighbors = wasNeighbors;` (cancel) |

So **9 is the shipped default**, and everything about the visible geometry (including the
scoreboard being clipped off-screen, 2.9) follows from that.

### 6.4 `DrawRoomBackground(who, where, elevation)`

`GliderPRO/Sources/RoomGraphics.c:162-253`. `who` is a room number or `kRoomIsEmpty` (−1),
`where` is a 0..8 slot index, `elevation` is the *floor* number of the room being drawn.

```
 1  if where == kCentralRoom:                                   /* :169            */
 2      thisBackground = thisHouse->rooms[who].background        /* cache for later */
 3      thisTiles[0..7] = thisHouse->rooms[who].tiles[0..7]
 4  if (numLights == 0) && (who != kRoomIsEmpty):               /* :179            */
 5      SetGWorld(backSrcMap); PaintRect(&localRoomsDest[where]); return
        /* an existing but UNLIT room is a solid black rectangle                    */
 6  if who == kRoomIsEmpty:                                     /* :193            */
 7      if wardBitSet:                                          /* :195            */
 8          SetGWorld(backSrcMap); PaintRect(&localRoomsDest[where]); return
 9      if elevation > 1:   pictID = kSky   (2015); tiles[0..7] = 2   /* :209-214   */
10      elif elevation == 1: pictID = kMeadow (2012); tiles[0..7] = 0 /* :215-220   */
11      else:                pictID = kDirt  (2011); tiles[0..7] = 0 /* :221-226   */
12  else:
13      pictID    = thisHouse->rooms[who].background            /* :232            */
14      tiles[i]  = thisHouse->rooms[who].tiles[i]              /* :234            */
15  SetPort((GrafPtr)workSrcMap)                                /* :238            */
16  LoadGraphicSpecial(pictID)     /* decodes the 512x322 PICT into the WORK map    */
                                  /* at (0,0)-(512,322), clobbering it             */
17  src  = (0, 0, kTileWide, kTileHigh)          = (0,0,64,322)
18  dest = (0, 0, 64, 322) offset to localRoomsDest[where].topLeft
19  for i in 0..7:                                              /* :244            */
20      src.left  = tiles[i] * kTileWide           /* tiles[i] in 0..7             */
21      src.right = src.left + kTileWide
22      CopyBits(workSrcMap -> backSrcMap, &src, &dest, srcCopy, nil)
23      QOffsetRect(&dest, kTileWide, 0)
```

Consequences a porter must reproduce:

* **The work map is used as PICT-decode scratch.** During a room change `workSrcMap` is
  repeatedly overwritten with whole 512x322 backgrounds. That is safe only because step 36 of
  `DrawLocale` copies the finished back map over the whole work map afterwards.
* **Off-screen background PICTs are still fully decoded.** Even for a neighbour whose tiles land
  entirely outside `houseRect`, `LoadGraphicSpecial` decodes the full 512x322 image. This is the
  single biggest cost of a room change (up to 9 backgrounds x ~105 KB of PackBits).
* `wardBitSet` (`GliderPRO/Sources/RoomGraphics.c:33`) forces *all* empty neighbours to black.
* **The `who != kRoomIsEmpty` in the dark test at :179 is deliberate**: an empty (non-existent)
  neighbour is never blackened by the darkness rule, because `numLights` for an empty room is 0
  and the outdoor fallbacks would otherwise never be reachable.

`GetNumberOfLights(where)` (`GliderPRO/Sources/Room.c:970-1099`) returns 1 for the eight
always-lit backgrounds `kGarden (2009)`, `kSkywalk (2010)`, `kMeadow (2012)`, `kField (2013)`,
`kRoof (2014)`, `kSky (2015)`, `kStratosphere (2016)`, `kStars (2017)`; for `kDirt (2011)` it
returns 1 only if **all eight** tile indices are 0; otherwise 0. If still 0 it counts objects:
`kDoorInLf`, `kDoorInRt`, `kWindowInLf`, `kWindowInRt`, `kWallWindow` always count, and the eight
light types (`kCeilingLight`, `kLightBulb`, `kTableLamp`, `kHipLamp`, `kDecoLamp`,
`kFlourescent`, `kTrackLight`, `kInvisLight`) count when their state byte is set. Note the play
path reads `data.f.state` (`GliderPRO/Sources/Room.c:1091`) while the edit path reads
`data.f.initial` (`GliderPRO/Sources/Room.c:1026`).

### 6.5 `LoadGraphicSpecial` — the user-art fallback chain

`GliderPRO/Sources/RoomGraphics.c:134-158`:

```
1  thePicture = GetPicture(resID)                        /* app resource fork      */
2  if nil: thePicture = (PicHandle)GetResource('Date', resID)   /* :142 user house */
3  if nil: thePicture = GetPicture(2000)                        /* :145 kSimpleRoom*/
4  if nil: RedAlert(kErrFailedGraphicLoad)
5  HLock; bounds = (*thePicture)->picFrame; HUnlock
6  OffsetRect(&bounds, -bounds.left, -bounds.top)        /* normalise to (0,0)      */
7  DrawPicture(thePicture, &bounds)                      /* 1:1, no scaling        */
8  ReleaseResource((Handle)thePicture)
```

The four-character resource type is literally `'Date'` — an in-joke type used to smuggle
user-supplied PICTs inside a house file so the Resource Manager's PICT cache does not collide
with the application's own IDs. Any house-supplied background lives under `'Date'`, not `'PICT'`.
**A Go port must therefore treat house resource type `'Date'` as PICT data.** (Compare
`LoadGraphicPlus`, `GliderPRO/Sources/Map.c:165-180`, which does the same but returns silently on
failure, and `LoadGraphicSpecial`'s cousin `LoadGraphic`, `GliderPRO/Sources/Utilities.c:317`,
which has no `'Date'` fallback at all.)

### 6.6 Tiles

| Constant | Value | Meaning |
|---|---|---|
| `kNumTiles` | 8 | tile columns per room (`GliderDefines.h:496`) |
| `kTileWide` | 64 | tile width in px (`GliderDefines.h:497`) |
| `kTileHigh` | 322 | tile height = full room height (`GliderDefines.h:498`) |
| `kRoomWide` | 512 | `= 8 * 64` (`GliderDefines.h:499`) |

A room's `tiles[8]` array (`short tiles[kNumTiles]` inside `roomType`) holds eight indices into
the background sheet. Index *n* selects source x range `[64n, 64n+64)`. Verified empirically: over
**32560 tile values in 4070 rooms across 22 shipped houses, every value is in 0..7** — the
renderer can index without clamping, but a defensive port should clamp anyway since nothing in
the file format enforces it.

Because `kTileHigh == 322` equals the whole background height, tiling is purely horizontal: a
room is eight full-height vertical strips chosen from eight candidates. There is no vertical
tiling and no sub-tile offsetting anywhere in the game.

### 6.7 `DrawFloorSupport` — the between-floors band

`suppSrcRect` is `(0,0,44,512)` = **512 x 44** (`kRoomWide` x `kFloorSupportTall`,
`GliderPRO/Sources/StructuresInit2.c:106`), loaded from PICT 1999 with **no mask**
(`GliderPRO/Sources/StructuresInit2.c:110`), measured `picFrame` (0,0,44,512) — an exact match.

`GliderPRO/Sources/RoomGraphics.c:257-376` performs up to six `srcCopy` blits of the full
`suppSrcRect` into `backSrcMap`, each gated on one `isStructure[]` flag:

| # | Gate | dest x (from) | dest y (from) | Line |
|---|---|---|---|---|
| 1 | `isStructure[kNorthWestRoom]` | `localRoomsDest[kWestRoom].left` | `localRoomsDest[kCentralRoom].top - 44` | :268-275 |
| 2 | `isStructure[kWestRoom]` | `localRoomsDest[kWestRoom].left` | `localRoomsDest[kCentralRoom].bottom` | :286-293 |
| 3 | `isStructure[kNorthRoom]` | `localRoomsDest[kCentralRoom].left` | `localRoomsDest[kCentralRoom].top - 44` | :304-311 |
| 4 | `isStructure[kCentralRoom]` | `localRoomsDest[kCentralRoom].left` | `localRoomsDest[kCentralRoom].bottom` | :321-328 |
| 5 | `isStructure[kNorthEastRoom]` | `localRoomsDest[kEastRoom].left` | `localRoomsDest[kCentralRoom].top - 44` | :339-346 |
| 6 | `isStructure[kEastRoom]` | `localRoomsDest[kEastRoom].left` | `localRoomsDest[kCentralRoom].bottom` | :357-364 |

Read as a picture: three 512-wide bands immediately **above** the central room (from the NW, N
and NE rooms' floors) and three immediately **below** it (from the W, central and E rooms'
floors). The gate is "is the room whose floor this is a *structure*", i.e. an indoor room that
should have a visible floor slab. `IsRoomAStructure` (`GliderPRO/Sources/Room.c:763-812`) returns
true for `kPaneledRoom (2001)`, `kSimpleRoom (2000)`, `kChildsRoom (2003)`, `kAsianRoom (2004)`,
`kUnfinishedRoom (2005)`, `kSwingersRoom (2006)`, `kBathroom (2007)`, `kLibrary (2008)`,
`kSkywalk (2010)` and `kRoof (2014)`; for user backgrounds (>= 3000) it uses
`(bounds & 32) == 32` when `bounds != 0`, else `background < kUserStructureRange (3300)`.

After **each** of the six blits, every registered temp manhole that intersects the band is
snapped to the band's vertical extent and re-drawn through it:

```
for i in 0..numTempManholes-1:                       /* e.g. :277-283              */
    if SectRect(&dest, &tempManholes[i], &whoCares):
        tempManholes[i].top    = dest.top
        tempManholes[i].bottom = dest.bottom
        LoadScaledGraphic(kManholeThruFloor /* 3957 */, &tempManholes[i])
```

`kManholeThruFloor` is `#define`d as **3957** at `GliderPRO/Sources/RoomGraphics.c:16`; PICT 3957
measures 123 x 44, exactly the manhole width by the support height, so `LoadScaledGraphic`'s
scaling is a no-op in the vertical direction and only stretches horizontally if the manhole
object was resized. Temp manholes are registered from the object pass
(`AddTempManholeRect`, called at `GliderPRO/Sources/ObjectDrawAll.c:277`).

**Note the mutation:** `tempManholes[i]` is *modified in place* inside the loop, so a manhole that
intersects two bands ends up snapped to the second one. With `numNeighbors == 9` the two bands at
the same x are 322 px apart, so this cannot happen in practice.

### 6.8 `ReadyBackMap` and `RestoreWorkMap`

Both are whole-surface `srcCopy` blits with identical src and dst rects
(`GliderPRO/Sources/RoomGraphics.c:380-398`):

| Function | Direction | Rect | Line |
|---|---|---|---|
| `ReadyBackMap` | `workSrcMap` -> `backSrcMap` | `workSrcRect` | :382-384 |
| `RestoreWorkMap` | `backSrcMap` -> `workSrcMap` | `backSrcRect` | :395-397 |

`RestoreWorkMap` declares a local `Rect dest = backSrcRect` at :393 and never uses it — harmless
dead code.

`ReadyBackMap` is the "promote the current composited frame to the new static background"
operation; it is used by the editor and by the game-over/high-score screens, never during play.

### 6.9 `DrawLighting` is a no-op

`GliderPRO/Sources/RoomGraphics.c:422-430`:

```c
void DrawLighting (void)
{
	if (numLights == 0)
		return;
	else
	{
		// for future construction
	}
}
```

It is called three times in `DrawLocale` (:110, :115, :121) and once in `RedrawRoomLighting`
(:452) and does nothing in either case. **A Go port should omit it entirely.** All actual
darkness handling is the `PaintRect` at `DrawRoomBackground:187` plus the per-object `isLit`
gates.

### 6.10 `RedrawRoomLighting` — the only partial background rebuild

`GliderPRO/Sources/RoomGraphics.c:434-461`. Called when a light switch toggles.

```
1  roomV  = thisHouse->rooms[thisRoomNumber].floor
2  wasLit = (numLights > 0)
3  numLights = GetNumberOfLights(localNumbers[kCentralRoom])
4  isLit  = (numLights > 0)
5  if wasLit == isLit: return                      /* nothing to do               */
6  DrawRoomBackground(localNumbers[kCentralRoom], kCentralRoom, roomV)
7  DrawARoomsObjects(kCentralRoom, true)           /* redraw = TRUE               */
8  DrawLighting()                                  /* no-op                       */
9  UpdateOutletsLighting(localNumbers[kCentralRoom], numLights)
10 if numNeighbors > 3: DrawFloorSupport()         /* :455-456                    */
11 RestoreWorkMap()                                /* whole map back -> work      */
12 AddRectToWorkRects(&localRoomsDest[kCentralRoom])   /* :458 push to screen     */
13 shadowVisible = IsShadowVisible()
```

Only the **central** room is redrawn, but `RestoreWorkMap` copies the *entire* map and only the
central room rect is queued for the screen — so neighbours keep their old pixels, which is correct
because their lighting did not change. The `redraw = true` argument makes the object pass call
`ReBackUp*` instead of `Add*` for every animated element so the `savedMaps` swatches are refreshed
against the new background rather than allocating new ones (see section 12).

### 6.11 `IsShadowVisible`

`GliderPRO/Sources/Room.c:1103-1134`. Returns whether the central room has a floor, i.e. whether
the glider's drop shadow should be drawn at all:

* user background (>= 3000): `boundsCode = bounds ? (bounds >> 1) : GetOriginalBounding(background)`,
  then `hasFloor = ((boundsCode & 0x0008) != 0x0008)`.
* otherwise `false` for `kRoof (2014)`, `kSky (2015)`, `kStratosphere (2016)`, `kStars (2017)`;
  `true` for every other background.

The result is cached in the global `shadowVisible` and read by `RenderGlider`
(`GliderPRO/Sources/Render.c:452`).

### 6.12 The editor draws the room through a *different* code path

`DrawThisRoomsObjects` (`GliderPRO/Sources/ObjectEdit.c:2347-2729`) is a second, parallel
object-layer dispatcher used only in edit mode. Differences that matter:

| Aspect | Play (`DrawARoomsObjects`) | Edit (`DrawThisRoomsObjects`) |
|---|---|---|
| Rooms drawn | 1, 3 or 9 | central only |
| Rect source | `GetObjectRect` + `OffsetRectRoomRelative` | `roomObjectRects[i]` (`ObjectEdit.c:28`), already in window coords |
| Origin | `playOriginH`, `playOriginV` | 0, 0 (the edit window *is* the room, 512x322) |
| Darkness | solid black `PaintRect` per room | `PenMode(srcOr); PenPat(gray); PaintRect(&backSrcRect); PenNormal();` over the whole map (`ObjectEdit.c:2362-2368`) — a 50% stipple, not black |
| Invisible objects | not drawn | drawn as outlines: `DrawInvisibleBlower`, `DrawLiftArea`, `DrawInvisObstacle`, `DrawInvisBounce`, `DrawInvisBonus`, `DrawSlider`, `DrawInvisTransport`, `DrawInvisibleSwitch`, `DrawTrigger`, `DrawSoundTrigger`, `DrawInvisLight` |
| Appliance state | `data.g.state` | `data.g.initial`, and `isLit` forced to `true` (`ObjectEdit.c:2612-2641`) |
| Animated elements | registered into `savedMaps`, flames, pendulums, dynamics | none; a single static frame is drawn |
| Extra overlays | none | initial-glider ghost + the two start-position glider ghosts, then a whole-map back->work copy (`ObjectEdit.c:2708-2728`) |

The three editor-only ghosts:

```
if (isFirstRoom)                                         /* ObjectEdit.c:2708      */
    CopyMask(glidSrcMap, glidMaskMap, backSrcMap,
             &gliderSrc[0], &gliderSrc[0], &initialGliderRect);
CopyMask(blowerSrcMap, blowerMaskMap, backSrcMap,        /* :2716                  */
         &leftStartGliderSrc,  &leftStartGliderSrc,  &leftStartGliderDest);
CopyMask(blowerSrcMap, blowerMaskMap, backSrcMap,        /* :2721                  */
         &rightStartGliderSrc, &rightStartGliderSrc, &rightStartGliderDest);
CopyBits(backSrcMap -> workSrcMap, &backSrcRect, &backSrcRect, srcCopy, nil); /* :2726 */
```

with (`GliderPRO/Sources/StructuresInit.c:280-284`, `GliderPRO/Sources/ObjectEdit.c:2061-2067`):

| Rect | Value |
|---|---|
| `leftStartGliderSrc` | 48 x 16 at (0, 358) in `blowerSrcMap` |
| `rightStartGliderSrc` | 48 x 16 at (0, 374) in `blowerSrcMap` |
| `leftStartGliderDest` | 48 x 16 at (0, `kGliderStartsDown` + `thisRoom->leftStart`) = (0, 32 + leftStart) |
| `rightStartGliderDest` | 48 x 16 at (`kRoomWide - 48`, 32 + rightStart) = (464, 32 + rightStart) |

So the two "glider enters here" markers are 48x16 outline sprites living inside the *blower*
sheet at y 358 and 374, and `kGliderStartsDown` is **32** (`GliderDefines.h:569`).

---

## 7. The object layer

Objects are drawn into `backSrcMap` (never into `workSrcMap` directly, never to screen) as part of
the room build. There is no per-frame object drawing: an object either becomes part of the static
background, or it registers itself as an *animated element* (flame, dynamic, pendulum, band,
sparkle, ...) which the per-frame renderer handles from a `savedMaps` swatch.

### 7.1 The rect pipeline

Every object case in the play-mode dispatcher runs the same two-step transform
(`GliderPRO/Sources/ObjectDrawAll.c:65-66` and 110 other places):

```
GetObjectRect(&thisObject, &itsRect);            /* room-local rect, origin (0,0)  */
OffsetRectRoomRelative(&itsRect, neighbor);      /* -> window coords               */
```

`GetObjectRect` (`GliderPRO/Sources/ObjectRects.c:32-273`) is a 117-way switch. Nine distinct
derivations:

| Kind | Rule | Line |
|---|---|---|
| `kObjectIsEmpty` (0) | `(0,0,0,0)` | :40 |
| blowers (0x01-0x0F, not `kLiftArea`) | `srcRects[what]`, `ZeroRectCorner`, offset by `data.a.topLeft` | :58-60 |
| `kLiftArea` (0x10) | `(0,0,data.a.distance, data.a.tall*2)` offset by `data.a.topLeft` | :64-65 |
| furniture (0x11-0x1F) | `data.b.bounds` verbatim — the rect is stored in the file | :88 |
| prizes (0x21-0x2E) | `srcRects[what]`, `ZeroRectCorner`, offset by `data.c.topLeft` | :105-109 |
| `kSlider` (0x2F) | as prizes, then `right = left + data.c.length` | :113-118 |
| transport (0x31-0x3E) | `srcRects[what]`, `ZeroRectCorner`, offset by `data.d.topLeft` | :135-139 |
| `kInvisTrans` (0x3F) | as transport, then `bottom = top + data.d.tall`, `right += data.d.wide` | :143-149 |
| `kDeluxeTrans` (0x40) | `wide = (data.d.tall & 0xFF00) >> 8`, `tall = data.d.tall & 0x00FF`, rect = `(0,0,wide*4,tall*4)` offset by `data.d.topLeft` | :153-158 |
| switches (0x41-0x49) | `srcRects[what]`, `ZeroRectCorner`, offset by `data.e.topLeft` | :170-174 |
| lights except the two long ones (0x51-0x55, 0x58) | `srcRects[what]`, `ZeroRectCorner`, offset by `data.f.topLeft` | :183-187 |
| `kFlourescent` (0x56), `kTrackLight` (0x57) | `srcRects[what]`, `ZeroRectCorner`, **`right = data.f.length`** (absolute, not `left+length`), then offset by `data.f.topLeft` | :192-197 |
| appliances (0x61-0x6D) | `srcRects[what]`, `ZeroRectCorner`, offset by `data.g.topLeft` | :213-217 |
| `kCustomPict` (0x6E) | `GetPicture(data.g.height)`; if nil, force `data.g.height = 10000` and use `srcRects[kCustomPict]` (72x34); else use the PICT's own `picFrame`. `ZeroRectCorner`, offset by `data.g.topLeft` | :221-236 |
| enemies (0x71-0x79) | `srcRects[what]`, `ZeroRectCorner`, offset by `data.h.topLeft` | :248-252 |
| clutter (0x81-0x8F) | `data.i.bounds` verbatim | :270 |

`ZeroRectCorner(r)` normalises the rect so its top-left is (0,0) while preserving width and height
(`GliderPRO/Sources/RectUtils.c`). This is what lets the same `srcRects[]` entry serve both as a
*source* rect inside a sprite sheet and as a *size* template for a destination rect.

The `kDeluxeTrans` packing is the only bit-packed geometry in the object format: one 16-bit field
carries two 8-bit dimensions, each multiplied by 4 (so up to 1020x1020).

`OffsetRectRoomRelative(theRect, neighbor)` (`GliderPRO/Sources/ObjectRects.c:1093-1131`):

```
QOffsetRect(theRect, playOriginH, playOriginV)     /* room-local -> window          */
switch neighbor:
  kNorthRoom      : ( 0,          -kVertLocalOffset)
  kNorthEastRoom  : (+kRoomWide,  -kVertLocalOffset)
  kEastRoom       : (+kRoomWide,   0)
  kSouthEastRoom  : (+kRoomWide,  +kVertLocalOffset)
  kSouthRoom      : ( 0,          +kVertLocalOffset)
  kSouthWestRoom  : (-kRoomWide,  +kVertLocalOffset)
  kWestRoom       : (-kRoomWide,   0)
  kNorthWestRoom  : (-kRoomWide,  -kVertLocalOffset)
  kCentralRoom    : (no extra offset)
```

with `kRoomWide` = **512** and `kVertLocalOffset` = **kTileHigh + kFloorSupportTall = 322 + 44 =
366**. `VerticalRoomOffset(neighbor)` (`GliderPRO/Sources/ObjectRects.c:1067-1089`) returns just
the vertical part: `-366` for the three north slots, `+366` for the three south slots, `0`
otherwise. It exists because some draw helpers need a *scalar* room origin rather than a rect
(they compute a shadow or a reflection y from it): `DrawTiki`, `DrawTable`, `DrawDeckTable`,
`DrawStool`, `DrawMailboxLeft`, `DrawMailboxRight` all take `playOriginV + VerticalRoomOffset(neighbor)`.

**This is a pure translation. There is no scaling anywhere in the room/object path.**

### 7.2 The clipping test

Almost every case is guarded by

```c
testRect = houseRect;  ZeroRectCorner(&testRect);       /* ObjectDrawAll.c:36-37    */
...
if (SectRect(&itsRect, &testRect, &whoCares))  ...
```

i.e. "does this object's window rect intersect a rect the size of `houseRect` placed at (0,0)".
Note `houseRect` is already at (0,0) in 9-room mode (`(0,0,1054,1536)`), so `ZeroRectCorner` is a
no-op there; the call matters only if a port changes `houseRect`'s origin. Objects entirely outside
the composited house are skipped — this is the *only* culling in the renderer.

Three cases deliberately have **no** `SectRect` guard and are always drawn:
`kMailboxLf` (`GliderPRO/Sources/ObjectDrawAll.c:485-489`), `kMailboxRt` (:491-495), and everything
that only registers an animated element rather than drawing.

### 7.3 `IsThisValid` — the collected-prize filter

`GliderPRO/Sources/Objects.c:89-122`. Called once per slot before the switch
(`GliderPRO/Sources/ObjectDrawAll.c:48`):

* `kObjectIsEmpty` -> false.
* The twelve state-bearing prizes `kRedClock`, `kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`,
  `kBattery`, `kBands`, `kFoil`, `kInvisBonus`, `kStar`, `kSparkle`, `kHelium` -> `data.c.state`
  (a `Boolean`; false once collected).
* everything else -> true.

Note `kGreaseRt`/`kGreaseLf` are **not** in that list even though they are prizes: grease uses
`data.c.state` for "has it been spilled" and is handled by `AddGrease` instead.

### 7.4 `srcRects[144]` — every sprite-sheet source rect

`srcRects` is `Rect srcRects[kNumSrcRects]` with `kNumSrcRects` = `0x90` = **144**
(`GliderPRO/Headers/GliderDefines.h:437`); it is indexed directly by the object's `what` byte, so
the array is sparse. Initialised in one function, `InitSrcRects`
(`GliderPRO/Sources/StructuresInit2.c:306-475`). The table below was generated mechanically by
re-executing every `QSetRect`/`QOffsetRect` in that function, so the rects are exactly what the
program computes (line numbers are the `QSetRect` line):

| `what` | hex | constant | src w x h | src rect (t,l,b,r) | sheet | line |
|---|---|---|---|---|---|---|
| 0 | 0x00 | *(unused code)* | - | - | - | - |
| 1 | 0x01 | `kFloorVent` | 48 x 11 | (0,0,11,48) | blower | :308 |
| 2 | 0x02 | `kCeilingVent` | 48 x 11 | (11,0,22,48) | blower | :310 |
| 3 | 0x03 | `kFloorBlower` | 48 x 15 | (22,0,37,48) | blower | :312 |
| 4 | 0x04 | `kCeilingBlower` | 48 x 15 | (37,0,52,48) | blower | :314 |
| 5 | 0x05 | `kSewerGrate` | 48 x 17 | (52,0,69,48) | blower | :316 |
| 6 | 0x06 | `kLeftFan` | 40 x 55 | (69,0,124,40) | blower | :318 |
| 7 | 0x07 | `kRightFan` | 40 x 55 | (124,0,179,40) | blower | :320 |
| 8 | 0x08 | `kTaper` | 20 x 59 | (209,0,268,20) | blower | :322 |
| 9 | 0x09 | `kCandle` | 32 x 30 | (179,0,209,32) | blower | :324 |
| 10 | 0x0A | `kStubby` | 20 x 36 | (268,0,304,20) | blower | :326 |
| 11 | 0x0B | `kTiki` | 27 x 28 | (268,21,296,48) | blower | :328 |
| 12 | 0x0C | `kBBQ` | 64 x 33 | (0,0,33,64) | blower | :330 |
| 13 | 0x0D | `kInvisBlower` | 24 x 24 | (0,0,24,24) | blower | :331 |
| 14 | 0x0E | `kGrecoVent` | 48 x 18 | (340,0,358,48) | blower | :332 |
| 15 | 0x0F | `kSewerBlower` | 32 x 12 | (390,0,402,32) | blower | :334 |
| 16 | 0x10 | `kLiftArea` | 64 x 32 | (0,0,32,64) | blower | :336 |
| 17 | 0x11 | `kTable` | 64 x 8 | (0,0,8,64) | furniture | :338 |
| 18 | 0x12 | `kShelf` | 64 x 6 | (0,0,6,64) | furniture | :339 |
| 19 | 0x13 | `kCabinet` | 64 x 64 | (0,0,64,64) | furniture | :340 |
| 20 | 0x14 | `kFilingCabinet` | 74 x 107 | (0,0,107,74) | furniture | :341 |
| 21 | 0x15 | `kWasteBasket` | 64 x 61 | (43,0,104,64) | furniture | :342 |
| 22 | 0x16 | `kMilkCrate` | 64 x 58 | (104,0,162,64) | furniture | :344 |
| 23 | 0x17 | `kCounter` | 128 x 64 | (0,0,64,128) | furniture | :346 |
| 24 | 0x18 | `kDresser` | 128 x 64 | (0,0,64,128) | furniture | :347 |
| 25 | 0x19 | `kDeckTable` | 64 x 8 | (0,0,8,64) | furniture | :348 |
| 26 | 0x1A | `kStool` | 48 x 38 | (183,0,221,48) | furniture | :349 |
| 27 | 0x1B | `kTrunk` | 144 x 80 | (0,0,80,144) | furniture | :351 |
| 28 | 0x1C | `kInvisObstacle` | 64 x 64 | (0,0,64,64) | furniture | :352 |
| 29 | 0x1D | `kManhole` | 123 x 22 | (0,0,22,123) | furniture | :353 |
| 30 | 0x1E | `kBooks` | 64 x 51 | (0,0,51,64) | furniture | :354 |
| 31 | 0x1F | `kInvisBounce` | 64 x 64 | (0,0,64,64) | furniture | :355 |
| 32 | 0x20 | *(unused code)* | - | - | - | - |
| 33 | 0x21 | `kRedClock` | 28 x 17 | (0,0,17,28) | prize | :357 |
| 34 | 0x22 | `kBlueClock` | 28 x 25 | (17,0,42,28) | prize | :358 |
| 35 | 0x23 | `kYellowClock` | 28 x 28 | (42,0,70,28) | prize | :360 |
| 36 | 0x24 | `kCuckoo` | 40 x 80 | (148,0,228,40) | prize | :362 |
| 37 | 0x25 | `kPaper` | 48 x 21 | (127,0,148,48) | prize | :364 |
| 38 | 0x26 | `kBattery` | 16 x 25 | (0,32,25,48) | prize | :366 |
| 39 | 0x27 | `kBands` | 28 x 23 | (70,20,93,48) | prize | :368 |
| 40 | 0x28 | `kGreaseRt` | 32 x 27 | (243,0,270,32) | prize | :370 |
| 41 | 0x29 | `kGreaseLf` | 32 x 27 | (324,0,351,32) | prize | :372 |
| 42 | 0x2A | `kFoil` | 55 x 15 | (228,0,243,55) | prize | :374 |
| 43 | 0x2B | `kInvisBonus` | 24 x 24 | (0,0,24,24) | prize | :376 |
| 44 | 0x2C | `kStar` | 32 x 31 | (0,48,31,80) | prize | :377 |
| 45 | 0x2D | `kSparkle` | 20 x 19 | (70,0,89,20) | prize | :379 |
| 46 | 0x2E | `kHelium` | 56 x 16 | (270,32,286,88) | prize | :381 |
| 47 | 0x2F | `kSlider` | 64 x 16 | (0,0,16,64) | prize | :383 |
| 48 | 0x30 | *(unused code)* | - | - | - | - |
| 49 | 0x31 | `kUpStairs` | 160 x 267 | (0,0,267,160) | transport | :385 |
| 50 | 0x32 | `kDownStairs` | 160 x 267 | (0,0,267,160) | transport | :386 |
| 51 | 0x33 | `kMailboxLf` | 94 x 80 | (0,0,80,94) | transport | :387 |
| 52 | 0x34 | `kMailboxRt` | 94 x 80 | (0,0,80,94) | transport | :388 |
| 53 | 0x35 | `kFloorTrans` | 56 x 15 | (1,0,16,56) | transport | :389 |
| 54 | 0x36 | `kCeilingTrans` | 56 x 15 | (16,0,31,56) | transport | :391 |
| 55 | 0x37 | `kDoorInLf` | 144 x 322 | (0,0,322,144) | transport | :393 |
| 56 | 0x38 | `kDoorInRt` | 144 x 322 | (0,0,322,144) | transport | :394 |
| 57 | 0x39 | `kDoorExRt` | 16 x 322 | (0,0,322,16) | transport | :395 |
| 58 | 0x3A | `kDoorExLf` | 16 x 322 | (0,0,322,16) | transport | :396 |
| 59 | 0x3B | `kWindowInLf` | 20 x 170 | (0,0,170,20) | transport | :397 |
| 60 | 0x3C | `kWindowInRt` | 20 x 170 | (0,0,170,20) | transport | :398 |
| 61 | 0x3D | `kWindowExRt` | 16 x 170 | (0,0,170,16) | transport | :399 |
| 62 | 0x3E | `kWindowExLf` | 16 x 170 | (0,0,170,16) | transport | :400 |
| 63 | 0x3F | `kInvisTrans` | 64 x 32 | (0,0,32,64) | transport | :401 |
| 64 | 0x40 | `kDeluxeTrans` | 64 x 64 | (0,0,64,64) | transport | :402 |
| 65 | 0x41 | `kLightSwitch` | 15 x 24 | (0,0,24,15) | switch | :404 |
| 66 | 0x42 | `kMachineSwitch` | 16 x 24 | (48,0,72,16) | switch | :405 |
| 67 | 0x43 | `kThermostat` | 15 x 24 | (48,0,72,15) | switch | :407 |
| 68 | 0x44 | `kPowerSwitch` | 8 x 8 | (72,0,80,8) | switch | :409 |
| 69 | 0x45 | `kKnifeSwitch` | 16 x 24 | (80,0,104,16) | switch | :411 |
| 70 | 0x46 | `kInvisSwitch` | 12 x 12 | (0,0,12,12) | switch | :413 |
| 71 | 0x47 | `kTrigger` | 12 x 12 | (0,0,12,12) | switch | :414 |
| 72 | 0x48 | `kLgTrigger` | 48 x 48 | (0,0,48,48) | switch | :415 |
| 73 | 0x49 | `kSoundTrigger` | 32 x 32 | (0,0,32,32) | switch | :416 |
| 74 | 0x4A | *(unused code)* | - | - | - | - |
| 75 | 0x4B | *(unused code)* | - | - | - | - |
| 76 | 0x4C | *(unused code)* | - | - | - | - |
| 77 | 0x4D | *(unused code)* | - | - | - | - |
| 78 | 0x4E | *(unused code)* | - | - | - | - |
| 79 | 0x4F | *(unused code)* | - | - | - | - |
| 80 | 0x50 | *(unused code)* | - | - | - | - |
| 81 | 0x51 | `kCeilingLight` | 64 x 20 | (0,0,20,64) | light | :418 |
| 82 | 0x52 | `kLightBulb` | 16 x 28 | (20,0,48,16) | light | :420 |
| 83 | 0x53 | `kTableLamp` | 48 x 70 | (20,16,90,64) | light | :422 |
| 84 | 0x54 | `kHipLamp` | 72 x 276 | (0,0,276,72) | light | :424 |
| 85 | 0x55 | `kDecoLamp` | 64 x 212 | (0,0,212,64) | light | :425 |
| 86 | 0x56 | `kFlourescent` | 64 x 12 | (0,0,12,64) | light | :426 |
| 87 | 0x57 | `kTrackLight` | 64 x 24 | (0,0,24,64) | light | :427 |
| 88 | 0x58 | `kInvisLight` | 16 x 16 | (0,0,16,16) | light | :428 |
| 89 | 0x59 | *(unused code)* | - | - | - | - |
| 90 | 0x5A | *(unused code)* | - | - | - | - |
| 91 | 0x5B | *(unused code)* | - | - | - | - |
| 92 | 0x5C | *(unused code)* | - | - | - | - |
| 93 | 0x5D | *(unused code)* | - | - | - | - |
| 94 | 0x5E | *(unused code)* | - | - | - | - |
| 95 | 0x5F | *(unused code)* | - | - | - | - |
| 96 | 0x60 | *(unused code)* | - | - | - | - |
| 97 | 0x61 | `kShredder` | 73 x 22 | (0,0,22,73) | appliance | :430 |
| 98 | 0x62 | `kToaster` | 48 x 27 | (22,0,49,48) | appliance | :431 |
| 99 | 0x63 | `kMacPlus` | 48 x 58 | (49,0,107,48) | appliance | :433 |
| 100 | 0x64 | `kGuitar` | 64 x 172 | (0,0,172,64) | appliance | :435 |
| 101 | 0x65 | `kTV` | 92 x 77 | (0,0,77,92) | appliance | :436 |
| 102 | 0x66 | `kCoffee` | 43 x 64 | (107,0,171,43) | appliance | :437 |
| 103 | 0x67 | `kOutlet` | 16 x 24 | (22,64,46,80) | appliance | :439 |
| 104 | 0x68 | `kVCR` | 96 x 22 | (0,0,22,96) | appliance | :441 |
| 105 | 0x69 | `kStereo` | 128 x 53 | (0,0,53,128) | appliance | :442 |
| 106 | 0x6A | `kMicrowave` | 92 x 59 | (0,0,59,92) | appliance | :443 |
| 107 | 0x6B | `kCinderBlock` | 40 x 62 | (0,0,62,40) | appliance | :444 |
| 108 | 0x6C | `kFlowerBox` | 80 x 32 | (0,0,32,80) | appliance | :445 |
| 109 | 0x6D | `kCDs` | 16 x 30 | (22,48,52,64) | appliance | :446 |
| 110 | 0x6E | `kCustomPict` | 72 x 34 | (0,0,34,72) | appliance | :448 |
| 111 | 0x6F | *(unused code)* | - | - | - | - |
| 112 | 0x70 | *(unused code)* | - | - | - | - |
| 113 | 0x71 | `kBalloon` | 24 x 30 | (0,0,30,24) | enemy | :450 |
| 114 | 0x72 | `kCopterLf` | 32 x 30 | (0,0,30,32) | enemy | :451 |
| 115 | 0x73 | `kCopterRt` | 32 x 30 | (0,0,30,32) | enemy | :452 |
| 116 | 0x74 | `kDartLf` | 64 x 19 | (0,0,19,64) | enemy | :453 |
| 117 | 0x75 | `kDartRt` | 64 x 19 | (0,0,19,64) | enemy | :454 |
| 118 | 0x76 | `kBall` | 32 x 32 | (0,0,32,32) | enemy | :455 |
| 119 | 0x77 | `kDrip` | 16 x 12 | (0,0,12,16) | enemy | :456 |
| 120 | 0x78 | `kFish` | 36 x 33 | (0,0,33,36) | enemy | :457 |
| 121 | 0x79 | `kCobweb` | 54 x 45 | (0,0,45,54) | enemy | :458 |
| 122 | 0x7A | *(unused code)* | - | - | - | - |
| 123 | 0x7B | *(unused code)* | - | - | - | - |
| 124 | 0x7C | *(unused code)* | - | - | - | - |
| 125 | 0x7D | *(unused code)* | - | - | - | - |
| 126 | 0x7E | *(unused code)* | - | - | - | - |
| 127 | 0x7F | *(unused code)* | - | - | - | - |
| 128 | 0x80 | *(unused code)* | - | - | - | - |
| 129 | 0x81 | `kOzma` | 102 x 92 | (0,0,92,102) | clutter | :460 |
| 130 | 0x82 | `kMirror` | 64 x 64 | (0,0,64,64) | clutter | :461 |
| 131 | 0x83 | `kMousehole` | 10 x 11 | (0,0,11,10) | clutter | :462 |
| 132 | 0x84 | `kFireplace` | 180 x 142 | (0,0,142,180) | clutter | :463 |
| 133 | 0x85 | `kFlower` | **never initialised** | - | clutter | - |
| 134 | 0x86 | `kWallWindow` | 64 x 80 | (0,0,80,64) | clutter | :464 |
| 135 | 0x87 | `kBear` | 56 x 58 | (0,0,58,56) | clutter | :465 |
| 136 | 0x88 | `kCalendar` | 63 x 92 | (0,0,92,63) | clutter | :466 |
| 137 | 0x89 | `kVase1` | 36 x 45 | (0,0,45,36) | clutter | :467 |
| 138 | 0x8A | `kVase2` | 35 x 57 | (0,0,57,35) | clutter | :468 |
| 139 | 0x8B | `kBulletin` | 80 x 58 | (0,0,58,80) | clutter | :469 |
| 140 | 0x8C | `kCloud` | 128 x 30 | (0,0,30,128) | clutter | :470 |
| 141 | 0x8D | `kFaucet` | 56 x 18 | (51,0,69,56) | clutter | :471 |
| 142 | 0x8E | `kRug` | 144 x 18 | (0,0,18,144) | clutter | :473 |
| 143 | 0x8F | `kChimes` | 28 x 74 | (0,0,74,28) | clutter | :474 |

Summary of the array's occupancy:

| | Count |
|---|---|
| Slots in the array | 144 (`0x00`-`0x8F`) |
| Object `what` codes defined | 117 |
| Codes with no `#define` at all (gaps between families) | 27: `0x00`, `0x20`, `0x30`, `0x4A`-`0x50`, `0x59`-`0x60`, `0x6F`-`0x70`, `0x7A`-`0x80` |
| Defined codes with an initialised `srcRects` entry | 116 |
| Defined codes with **no** `srcRects` entry | 1: `kFlower` (0x85) |

`kFlower` is intentional: it is clutter, so `GetObjectRect` returns `who->data.i.bounds`
(`GliderPRO/Sources/ObjectRects.c:270`) and never touches `srcRects[kFlower]`. `DrawFlower`
(`GliderPRO/Sources/ObjectDraw2.c`) is passed both the rect and `data.i.pict`, so the flower's
size comes entirely from the house file. A Go port can leave the slot zero.

Two entries have a `QSetRect` but no `QOffsetRect`, so they sit at (0,0) and are **not** valid
sheet coordinates — they are size templates only: `kBBQ` (64x33, drawn from PICT 3958 via
`DrawPictSansWhiteObject`) and `kLiftArea` (64x32, an invisible object whose real size comes from
`data.a.distance`/`data.a.tall`). The same is true of all the other invisible objects
(`kInvisBlower` 24x24, `kInvisObstacle` 64x64, `kInvisBounce` 64x64, `kInvisBonus` 24x24,
`kInvisTrans` 64x32, `kDeluxeTrans` 64x64, `kInvisSwitch` 12x12, `kTrigger` 12x12,
`kLgTrigger` 48x48, `kSoundTrigger` 32x32, `kInvisLight` 16x16) and of the furniture family
(whose rects come from `data.b.bounds`) — for those, `srcRects` supplies only the default size the
editor uses when you create one.

Two entries use symbolic thicknesses: `kTable` is `64 x kTableThick` and `kShelf` is
`64 x kShelfThick` (`GliderPRO/Sources/StructuresInit2.c:338-339`), with `kTableThick` = **8** and
`kShelfThick` = **6** (`GliderPRO/Headers/GliderDefines.h`).

### 7.5 Which sheet each source rect indexes

`srcRects` entries are source coordinates in one of eight 8-bit sprite-sheet GWorlds, chosen by the
draw helper, not by the rect. Sheet sizes are from the `QSetRect` calls in `InitBlowers` /
`InitFurniture` / ... (`GliderPRO/Sources/StructuresInit.c`), PICT IDs from the `#define`s at
`GliderPRO/Sources/StructuresInit.c:19-39` and `GliderPRO/Sources/StructuresInit2.c:20-22`:

| Sheet GWorld (+ mask) | Size | Art PICT | Mask PICT | `QSetRect` line | Families that index it |
|---|---|---|---|---|---|
| `blowerSrcMap` / `blowerMaskMap` | 48 x 402 | `kBlowerPictID` 4000 | 5000 | :253 | blowers 0x01-0x0F, plus the two editor start-glider ghosts at y 358 / 374 |
| `furnitureSrcMap` / `furnitureMaskMap` | 64 x 278 | `kFurniturePictID` 4001 | 5001 | :301 | the sheet-drawn furniture (0x15 `kWasteBasket`, 0x16 `kMilkCrate`, 0x1A `kStool`) |
| `bonusSrcMap` / `bonusMaskMap` | 88 x 378 | `kBonusPictID` 4002 | 5002 | :350 | prizes 0x21-0x2F |
| `switchSrcMap` | 32 x 104 | `kSwitchPictID` 4003 | **none — PICT 5003 does not exist** | :455 | switches 0x41-0x45 |
| `lightSrcMap` / `lightMaskMap` | 72 x 126 | `kLightPictID` 4004 | 5004 | :501 | lights 0x51-0x53 |
| `applianceSrcMap` / `applianceMaskMap` | 80 x 269 | `kAppliancePictID` 4005 | 5005 | :538 | appliances 0x61-0x6D |
| `transSrcMap` / `transMaskMap` | 56 x 32 | `kTransportPictID` 4008 | 5008 | :431 | only 0x35 `kFloorTrans` (56x15 @ y 1) and 0x36 `kCeilingTrans` (56x15 @ y 16) |
| `clutterSrcMap` / `clutterMaskMap` | 128 x 69 | `kClutterPictID` 4018 | 5018 | `StructuresInit2.c:63` | 0x83 `kMousehole` (10x11), 0x8D `kFaucet` (56x18 @ y 51) |

Note two mismatches between sheet GWorld and art PICT (also flagged in 5.9): `furnitureSrcRect` is
64 x **278** but PICT 4001 measures 64 x **221**, leaving 57 unpainted rows at the bottom of the
sheet; and `applianceSrcRect` is 80 x **269** with PICT 4005 at 80 x 269 but mask PICT 5005 at
80 x **268**, one row short.

`kBBQ` (0x0C) has a `srcRects` entry (64x33) but its art comes from PICT 3958 via
`DrawPictSansWhiteObject`, not from `blowerSrcMap`; it is one of the "size template only" entries.

Sheets that exist for animated elements rather than for static objects, and so are read by the
per-frame renderer instead of by the object dispatcher:

| GWorld (+ mask) | Size | Art PICT | Mask PICT | Line |
|---|---|---|---|---|
| `glidSrcMap` | 48 x 668 | `kGliderPictID` 3999 | 4999 (`+1000`) | `StructuresInit.c:178-189` |
| `glid2SrcMap` | 48 x 668 | `kGlider2PictID` 3974 | *(shares `glidMaskMap`)* | :183-185 |
| `shadowSrcMap` / `shadowMaskMap` | 48 x (`kShadowHigh * kNumShadowSrcRects`) | `kShadowPictID` 3998 | 4998 | :207-214 |
| `bandsSrcMap` / `bandsMaskMap` | 16 x 18 | `kRubberBandsPictID` 4007 | 5007 | :222-229 |
| `pointsSrcMap` / `pointsMaskMap` | 24 x 120 | `kPointsPictID` 4006 | 5006 | :403-410 |
| `toastSrcMap` / `toastMaskMap` | 32 x 174 | `kToastPictID` 4009 | 5009 | :547-554 |
| `shredSrcMap` / `shredMaskMap` | 40 x 35 | `kShreddedPictID` 4010 | 5010 | :556-563 |
| `balloonSrcMap` / `balloonMaskMap` | 24 x 30·`kNumBalloonFrames` | `kBalloonPictID` 4011 | 5011 | :623-630 |
| `copterSrcMap` / `copterMaskMap` | 32 x 30·`kNumCopterFrames` | `kCopterPictID` 4012 | 5012 | :632-639 |
| `dartSrcMap` / `dartMaskMap` | 64 x 19·`kNumDartFrames` | `kDartPictID` 4013 | 5013 | :641-648 |
| `ballSrcMap` / `ballMaskMap` | 32 x 32·`kNumBallFrames` | `kBallPictID` 4014 | 5014 | :650-657 |
| `dripSrcMap` / `dripMaskMap` | 16 x 12·`kNumDripFrames` | `kDripPictID` 4015 | 5015 | :659-666 |
| `enemySrcMap` / `enemyMaskMap` | 36 x 33 | `kEnemyPictID` 4016 | 5016 | :668-675 |
| `fishSrcMap` / `fishMaskMap` | 16 x 16·`kNumFishFrames` | `kFishPictID` 4017 | 5017 | :677-684 |
| `suppSrcMap` | 512 x 44 | `kSupportPictID` 1999 | *(none — opaque)* | `StructuresInit2.c:106-109` |
| `angelSrcMap` / `angelMaskMap` | 96 x 44 | `kAngelPictID` 1019 | **1020 (`+1`, not `+1000`)** | `StructuresInit2.c:127-134` |
| `badgeSrcMap` | 32 x 66 | `kBadgePictID` 1996 | *(none)* | `StructuresInit.c:92-95` |

Everything else is drawn from a **per-object PICT** loaded on demand (`DrawPictObject`,
`DrawPictSansWhiteObject`, `DrawPictWithMaskObject`, `DrawCustPictSansWhite`) rather than from a
sheet; see section 8.

### 7.6 `DrawARoomsObjects` — the complete play-mode dispatcher

`GliderPRO/Sources/ObjectDrawAll.c:23-966`. Signature
`void DrawARoomsObjects (short neighbor, Boolean redraw)`.

Preamble (`:33-41`):

```
1  if localNumbers[neighbor] == kRoomIsEmpty: return            /* :33-34         */
2  testRect = houseRect;  ZeroRectCorner(&testRect)             /* :36-37         */
3  isLit = (numLights > 0)                                      /* :38            */
4  wasState = HGetState(thisHouse);  HLock(thisHouse)            /* :40-41         */
5  for i in 0 .. kMaxRoomObs-1:                                 /* :43, 24 slots  */
6      dynamicNum = -1;  legit = -1                             /* :45-46         */
7      if IsThisValid(localNumbers[neighbor], i):                /* :48            */
8          thisObject = (*thisHouse)->rooms[localNumbers[neighbor]].objects[i]
9          switch (thisObject.what):  ...                        /* :51-951        */
10     if !redraw:  <write dynamicNum back into masterObjects>   /* :953-965       */
```

The switch below was extracted mechanically from the source: 74 case groups covering all 118
codes (117 object types + `kObjectIsEmpty`). "gates" lists the conditions the C wraps around the
call; "central-only" means the branch is `if (neighbor == kCentralRoom)`.

| `what` codes | static draw call | animated-element registration | gates |
|---|---|---|---|
| `kObjectIsEmpty` 0x00 | - | - | - |
| `kFloorVent` 0x01, `kCeilingVent` 0x02, `kFloorBlower` 0x03, `kCeilingBlower` 0x04, `kSewerGrate` 0x05, `kLeftFan` 0x06, `kRightFan` 0x07, `kGrecoVent` 0x0E, `kSewerBlower` 0x0F | `DrawSimpleBlowers` :68 | - | SectRect(houseRect), isLit |
| `kTaper` 0x08 | `DrawSimpleBlowers` :77 | `ReBackUpFlames` :81, `AddCandleFlame` :83 | SectRect(houseRect), central-only, isLit |
| `kCandle` 0x09 | `DrawSimpleBlowers` :111 | `ReBackUpFlames` :115, `AddCandleFlame` :117 | SectRect(houseRect), central-only, isLit |
| `kStubby` 0x0A | `DrawSimpleBlowers` :145 | `ReBackUpFlames` :149, `AddCandleFlame` :151 | SectRect(houseRect), central-only, isLit |
| `kTiki` 0x0B | `DrawTiki` :177 | `ReBackUpTikiFlames` :179, `AddTikiFlame` :181 | isLit |
| `kBBQ` 0x0C | `DrawPictSansWhiteObject` :191 | `ReBackUpBBQCoals` :193, `AddBBQCoals` :195 | SectRect(houseRect), isLit |
| `kInvisBlower` 0x0D, `kLiftArea` 0x10 | - | - | - |
| `kTable` 0x11 | `DrawTable` :208 | - | isLit |
| `kShelf` 0x12 | `DrawShelf` :215 | - | isLit |
| `kCabinet` 0x13 | `DrawCabinet` :222 | - | SectRect(houseRect), isLit |
| `kFilingCabinet` 0x14, `kOzma` 0x81 | `DrawPictObject` :230 | - | SectRect(houseRect), isLit |
| `kWasteBasket` 0x15, `kMilkCrate` 0x16 | `DrawSimpleFurniture` :238 | - | SectRect(houseRect), isLit |
| `kCounter` 0x17 | `DrawCounter` :245 | - | SectRect(houseRect), isLit |
| `kDresser` 0x18 | `DrawDresser` :252 | - | isLit |
| `kDeckTable` 0x19 | `DrawDeckTable` :259 | - | isLit |
| `kStool` 0x1A | `DrawStool` :266 | - | isLit |
| `kInvisObstacle` 0x1C | - | - | - |
| `kManhole` 0x1D | `DrawPictSansWhiteObject` :279 | `AddTempManholeRect` :277 | SectRect(houseRect), isLit |
| `kInvisBounce` 0x1F | - | - | - |
| `kRedClock` 0x21 | `DrawRedClock` :296 | `ReBackUpSavedMap` :292, `BackUpToSavedMap` :294 | SectRect(houseRect) |
| `kBlueClock` 0x22 | `DrawBlueClock` :310 | `ReBackUpSavedMap` :306, `BackUpToSavedMap` :308 | SectRect(houseRect) |
| `kYellowClock` 0x23 | `DrawYellowClock` :324 | `ReBackUpSavedMap` :320, `BackUpToSavedMap` :322 | SectRect(houseRect) |
| `kCuckoo` 0x24 | `DrawCuckoo` :339 | `ReBackUpSavedMap` :334, `BackUpToSavedMap` :336, `ReBackUpPendulum` :341, `AddPendulum` :343 | SectRect(houseRect) |
| `kPaper` 0x25, `kBattery` 0x26, `kBands` 0x27, `kHelium` 0x2E | `DrawSimplePrizes` :362 | `ReBackUpSavedMap` :358, `BackUpToSavedMap` :360 | SectRect(houseRect) |
| `kGreaseRt` 0x28 | `DrawGreaseRt` :380 | `ReBackUpGrease` :374, `AddGrease` :376 | SectRect(houseRect) |
| `kGreaseLf` 0x29 | `DrawGreaseLf` :401 | `ReBackUpGrease` :395, `AddGrease` :397 | SectRect(houseRect) |
| `kFoil` 0x2A | `DrawFoil` :418 | `ReBackUpSavedMap` :414, `BackUpToSavedMap` :416 | SectRect(houseRect) |
| `kInvisBonus` 0x2B, `kSlider` 0x2F | - | - | - |
| `kStar` 0x2C | `DrawSimplePrizes` :442 | `ReBackUpSavedMap` :432, `BackUpToSavedMap` :434, `ReBackUpStar` :438, `AddStar` :440 | SectRect(houseRect) |
| `kSparkle` 0x2D | - | `AddDynamicObject` :456 | SectRect(houseRect), central-only |
| `kUpStairs` 0x31, `kDoorInLf` 0x37, `kDoorInRt` 0x38, `kWindowInLf` 0x3B, `kWindowInRt` 0x3C | `DrawPictSansWhiteObject` :470 | - | SectRect(houseRect) |
| `kDownStairs` 0x32, `kDoorExRt` 0x39, `kDoorExLf` 0x3A, `kWindowExRt` 0x3D, `kWindowExLf` 0x3E | `DrawPictObject` :481 | - | SectRect(houseRect) |
| `kMailboxLf` 0x33 | `DrawMailboxLeft` :487 | - | - |
| `kMailboxRt` 0x34 | `DrawMailboxRight` :493 | - | - |
| `kFloorTrans` 0x35, `kCeilingTrans` 0x36 | `DrawSimpleTransport` :501 | - | SectRect(houseRect) |
| `kInvisTrans` 0x3F, `kDeluxeTrans` 0x40 | - | - | - |
| `kLightSwitch` 0x41 | `DrawLightSwitch` :516 | - | SectRect(houseRect) |
| `kMachineSwitch` 0x42 | `DrawMachineSwitch` :529 | - | SectRect(houseRect) |
| `kThermostat` 0x43 | `DrawThermostat` :542 | - | SectRect(houseRect) |
| `kPowerSwitch` 0x44 | `DrawPowerSwitch` :555 | - | SectRect(houseRect) |
| `kKnifeSwitch` 0x45 | `DrawKnifeSwitch` :568 | - | SectRect(houseRect) |
| `kInvisSwitch` 0x46 | - | - | - |
| `kTrigger` 0x47, `kLgTrigger` 0x48, `kSoundTrigger` 0x49 | - | - | - |
| `kCeilingLight` 0x51, `kLightBulb` 0x52, `kTableLamp` 0x53 | `DrawSimpleLight` :588 | - | SectRect(houseRect), isLit |
| `kTrunk` 0x1B, `kBooks` 0x1E, `kHipLamp` 0x54, `kDecoLamp` 0x55, `kGuitar` 0x64, `kCinderBlock` 0x6B, `kFlowerBox` 0x6C, `kFireplace` 0x84, `kBear` 0x87, `kVase1` 0x89, `kVase2` 0x8A, `kRug` 0x8E, `kChimes` 0x8F | `DrawPictSansWhiteObject` :607 | - | SectRect(houseRect), isLit |
| `kCustomPict` 0x6E | `DrawCustPictSansWhite` :614 | - | SectRect(houseRect), isLit |
| `kFlourescent` 0x56 | `DrawFlourescent` :621 | - | SectRect(houseRect), isLit |
| `kTrackLight` 0x57 | `DrawTrackLight` :628 | - | SectRect(houseRect), isLit |
| `kInvisLight` 0x58 | - | - | - |
| `kShredder` 0x61, `kCDs` 0x6D | `DrawSimpleAppliance` :639 | - | SectRect(houseRect), isLit |
| `kToaster` 0x62 | `DrawSimpleAppliance` :647 | `AddDynamicObject` :652 | SectRect(houseRect), central-only |
| `kMacPlus` 0x63 | `DrawMacPlus` :663 | `AddDynamicObject` :668 | SectRect(houseRect), isLit |
| `kTV` 0x65 | `DrawTV` :696 | `AddDynamicObject` :701 | SectRect(houseRect), central-only, isLit |
| `kCoffee` 0x66 | `DrawCoffee` :720 | `AddDynamicObject` :725 | SectRect(houseRect), isLit |
| `kOutlet` 0x67 | `DrawOutlet` :737 | `AddDynamicObject` :742 | SectRect(houseRect), isLit |
| `kVCR` 0x68 | `DrawVCR` :753 | `AddDynamicObject` :758 | SectRect(houseRect), isLit |
| `kStereo` 0x69 | `DrawStereo` :769 | `AddDynamicObject` :774 | SectRect(houseRect), isLit |
| `kMicrowave` 0x6A | `DrawMicrowave` :785 | `AddDynamicObject` :790 | SectRect(houseRect), isLit |
| `kBalloon` 0x71 | - | `AddDynamicObject` :802 | central-only |
| `kCopterLf` 0x72 | - | `AddDynamicObject` :813 | central-only |
| `kCopterRt` 0x73 | - | `AddDynamicObject` :824 | central-only |
| `kDartLf` 0x74 | - | `AddDynamicObject` :835 | central-only |
| `kDartRt` 0x75 | - | `AddDynamicObject` :846 | central-only |
| `kBall` 0x76 | - | `AddDynamicObject` :857 | central-only |
| `kDrip` 0x77 | `DrawDrip` :867 | `AddDynamicObject` :872 | SectRect(houseRect), central-only |
| `kFish` 0x78 | `DrawFish` :883 | `AddDynamicObject` :888 | SectRect(houseRect), central-only |
| `kCobweb` 0x79, `kCloud` 0x8C | `DrawPictWithMaskObject` :899 | - | SectRect(houseRect), isLit |
| `kMirror` 0x82 | `DrawMirror` :906 | `AddToMirrorRegion` :910 | SectRect(houseRect), central-only, isLit |
| `kMousehole` 0x83, `kFaucet` 0x8D | `DrawSimpleClutter` :919 | - | SectRect(houseRect), isLit |
| `kFlower` 0x85 | `DrawFlower` :926 | - | SectRect(houseRect), isLit |
| `kWallWindow` 0x86 | `DrawWallWindow` :933 | - | SectRect(houseRect) |
| `kCalendar` 0x88 | `DrawCalendar` :940 | - | SectRect(houseRect), isLit |
| `kBulletin` 0x8B | `DrawBulletin` :947 | - | SectRect(houseRect), isLit |

Details the table cannot carry:

* **`redraw` selects `ReBackUp*` instead of `Add*`.** Every animated element has a pair: `AddX`
  allocates a new `savedMaps` slot and captures the background swatch; `ReBackUpX` re-captures the
  swatch for the element already registered for that (room, objectIndex). `RedrawRoomLighting`
  calls with `redraw = true`; `DrawLocale` with `redraw = false`.
* **Candle-family flames use a proximity test in neighbour rooms.** For `kTaper` (:88-100),
  `kCandle` (:122-134) and `kStubby` (:156-168), a flame in a *neighbour* room is registered only
  if a 16x15 rect placed at `(itsRect.left + dx - 8, itsRect.top + 7 - 15)` does **not** intersect
  `localRoomsDest[kCentralRoom]` grown by `kFloorSupportTall` (44) top and bottom. In other words:
  a neighbour flame is animated only when its flame rect is far enough from the playfield that it
  cannot be clobbered by the central room's per-frame erases. `dx` is +10 for `kTaper`, +14 for
  `kCandle`, +9 for `kStubby`; the flame's `y` anchor is `itsRect.top + 7` in all three cases.
* **The tiki flame anchor is different**: `AddTikiFlame(..., itsRect.left + 10, itsRect.top - 9)`
  (`:181-182`) — above the object, not inside it — and it has no proximity test.
* **BBQ coals**: `AddBBQCoals(..., itsRect.left + 16, itsRect.top + 9)` (`:195-196`).
* **`kSparkle`** offsets its rect *back* into room-local space before registering:
  `rectA = itsRect; QOffsetRect(&rectA, -playOriginH, -playOriginV)` (`:455`) then
  `AddDynamicObject(kSparkle, &rectA, ...)`. The dynamics system stores room-local coordinates and
  adds the origin per frame.
* **Prizes call `BackUpToSavedMap` before drawing** (e.g. `:294` for `kRedClock`) and record the
  returned slot in `legit`; the actual `DrawRedClock` at `:296` is skipped if `legit == -1` (out of
  savedMaps slots). This is the pre-composition described in section 12.
* **Switches resolve their room number the long way** (`:513-515`):
  `ExtractFloorSuite(localNumbers[neighbor], &floor, &suite); room = GetRoomNumber(floor, suite);
  obj = i;` then `DrawLightSwitch(&itsRect, GetObjectState(room, obj))` — the drawn state comes
  from the live object state table, not from `thisObject.data.e.initial`.
* **`dynamicNum = masterObjects[i].hotNum`** is set for all five real switches and for
  `kInvisSwitch` (`:518`, `:531`, `:544`, `:557`, `:570`, `:574`) — reusing the `dynamicNum`
  variable to carry a *hot-spot* number rather than a dynamic-object number.

### 7.7 `isLit` gating is **not** uniform

`isLit` appears 36 times in `DrawARoomsObjects` (lines 31, 38, 67, 76, 110, 144, 176, 190, 207,
214, 221, 229, 237, 244, 251, 258, 265, 278, 587, 606, 613, 620, 627, 638, 663, 696, 720, 736,
753, 769, 785, 898, 905, 918, 925, 939, 946). Grouping by family:

| Family | `what` range | Gated by `isLit`? |
|---|---|---|
| Blowers | 0x01-0x10 | **yes** (all drawn ones) |
| Furniture | 0x11-0x1F | **yes** |
| Prizes | 0x21-0x2F | **no** — clocks, paper, battery, bands, foil, star, grease all draw in the dark |
| Transport (stairs, doors, windows, mailboxes, floor/ceiling transports) | 0x31-0x40 | **no** |
| Switches | 0x41-0x49 | **no** |
| Lights | 0x51-0x58 | **yes** |
| Appliances | 0x61-0x6E | **yes** |
| Enemies | 0x71-0x79 | mixed: `kCobweb` **yes**; the rest only register dynamics (ungated) |
| Clutter | 0x81-0x8F | **yes**, except `kWallWindow` (0x86, `:933`) which is **ungated** |

So in a fully unlit room the player still sees: every uncollected prize, every stair/door/window,
both mailboxes, both simple transports, all five switches, and the wall window — floating on solid
black. This is a deliberate gameplay affordance (you must be able to find the light switch), and a
port that hoists the `isLit` test to the top of the loop will break it.

### 7.8 The `masterObjects` write-back

`GliderPRO/Sources/ObjectDrawAll.c:953-965`, executed once per object slot, only when
`redraw == false`:

```
if (!redraw)
    for n in 0 .. numMasterObjects-1:
        if (masterObjects[n].objectNum == i) &&
           (masterObjects[n].roomNum   == localNumbers[neighbor]):
            masterObjects[n].dynaNum = dynamicNum
```

`dynamicNum` is `-1` unless the case explicitly set it, so this both records newly created dynamic
objects/hot spots and *clears* stale indices for objects that no longer animate. It is a linear
scan inside a linear scan — O(kMaxRoomObs x numMasterObjects) per room, 9 rooms per room change.

### 7.9 Play-mode vs edit-mode object drawing

Both dispatchers `switch` on all 118 codes. Comparing the `Draw*` helper each one calls (generated
by extracting every `Draw*(` call per case from `GliderPRO/Sources/ObjectDrawAll.c:43-952` and
`GliderPRO/Sources/ObjectEdit.c:2370-2703`), **exactly 20 of 118 codes differ**, and in every one
of those 20 the play path draws *nothing* while the editor draws a visible stand-in:

| `what` | code | play-mode draw | edit-mode draw |
|---|---|---|---|
| `kInvisBlower` | 0x0D | *(none)* | `DrawInvisibleBlower` |
| `kLiftArea` | 0x10 | *(none)* | `DrawLiftArea` |
| `kInvisObstacle` | 0x1C | *(none)* | `DrawInvisObstacle` |
| `kInvisBounce` | 0x1F | *(none)* | `DrawInvisBounce` |
| `kInvisBonus` | 0x2B | *(none)* | `DrawInvisBonus` |
| `kSparkle` | 0x2D | *(none)* | `DrawSimplePrizes` |
| `kSlider` | 0x2F | *(none)* | `DrawSlider` |
| `kInvisTrans` | 0x3F | *(none)* | `DrawInvisTransport` |
| `kDeluxeTrans` | 0x40 | *(none)* | `DrawInvisTransport` |
| `kInvisSwitch` | 0x46 | *(none)* | `DrawInvisibleSwitch` |
| `kTrigger` | 0x47 | *(none)* | `DrawTrigger` |
| `kLgTrigger` | 0x48 | *(none)* | `DrawTrigger` |
| `kSoundTrigger` | 0x49 | *(none)* | `DrawSoundTrigger` |
| `kInvisLight` | 0x58 | *(none)* | `DrawInvisLight` |
| `kBalloon` | 0x71 | *(none)* | `DrawBalloon` |
| `kCopterLf` | 0x72 | *(none)* | `DrawCopter` |
| `kCopterRt` | 0x73 | *(none)* | `DrawCopter` |
| `kDartLf` | 0x74 | *(none)* | `DrawDart` |
| `kDartRt` | 0x75 | *(none)* | `DrawDart` |
| `kBall` | 0x76 | *(none)* | `DrawBall` |

The remaining 98 codes call the same helper in both modes; the differences there are in the
*arguments* (origin 0 vs `playOriginV`, `data.g.initial` vs `data.g.state`, `isLit` vs hard `true`)
as tabulated in 6.12.

The eleven invisible-object outline helpers and their definitions:

| Helper | Defined at |
|---|---|
| `DrawInvisibleBlower` | `GliderPRO/Sources/ObjectDraw.c` (blower group) |
| `DrawLiftArea` | `GliderPRO/Sources/ObjectDraw.c:134` |
| `DrawInvisObstacle` | `GliderPRO/Sources/ObjectDraw.c:933` |
| `DrawInvisBounce` | `GliderPRO/Sources/ObjectDraw.c:946` |
| `DrawInvisBonus` | `GliderPRO/Sources/ObjectDraw.c:1383` |
| `DrawSlider` | `GliderPRO/Sources/ObjectDraw.c:1396` |
| `DrawInvisTransport` | `GliderPRO/Sources/ObjectDraw2.c:286` |
| `DrawInvisibleSwitch` | `GliderPRO/Sources/ObjectDraw2.c:373` |
| `DrawTrigger` | `GliderPRO/Sources/ObjectDraw2.c:386` |
| `DrawSoundTrigger` | `GliderPRO/Sources/ObjectDraw2.c:399` |
| `DrawInvisLight` | `GliderPRO/Sources/ObjectDraw2.c:584` |

A Go port that ships only the game (no editor) can omit all 11 plus `DrawThisRoomsObjects`
entirely — roughly 400 lines of the original.

---

## 8. Sprite masking — the four strategies

Glider PRO has no alpha channel and no per-pixel blending. Everything is either an opaque
rectangle, a 1-bit-masked blit, or a colour-keyed blit. There are exactly four mechanisms.

### 8.1 Strategy A — pre-built 1-bit mask GWorld (`CopyMask`, no allocation)

Used for every sprite that lives in a permanently loaded sheet. The mask sheet is a second GWorld
of identical dimensions created at depth **1** and filled from PICT `artID + 1000`.

```c
CopyMask((BitMap *)*GetGWorldPixMap(applianceSrcMap),      /* colour source        */
         (BitMap *)*GetGWorldPixMap(applianceMaskMap),     /* 1-bit mask, same rect */
         (BitMap *)*GetGWorldPixMap(backSrcMap),           /* destination           */
         &srcRects[kOutlet], &srcRects[kOutlet], theRect); /* srcRect, maskRect, dst */
```
(`GliderPRO/Sources/ObjectDraw2.c:726-729`, `DrawOutlet` — the shortest example in the program.)

Note the middle rect is the **mask rect** and it is always identical to the source rect: the mask
sheet is a pixel-for-pixel parallel of the art sheet, so mask coordinates == art coordinates.
QuickDraw's `CopyMask` semantics: for each destination pixel, if the corresponding mask bit is 1
(black) copy the source pixel, else leave the destination alone. Mask PICTs are therefore
**black-where-opaque**, white-where-transparent.

Sheets with a permanent mask twin (18 pairs): `glidSrcMap`/`glidMaskMap`,
`shadowSrcMap`/`shadowMaskMap`, `bandsSrcMap`/`bandsMaskMap`,
`blowerSrcMap`/`blowerMaskMap`, `furnitureSrcMap`/`furnitureMaskMap`,
`bonusSrcMap`/`bonusMaskMap`, `pointsSrcMap`/`pointsMaskMap`, `transSrcMap`/`transMaskMap`,
`lightSrcMap`/`lightMaskMap`, `applianceSrcMap`/`applianceMaskMap`, `toastSrcMap`/`toastMaskMap`,
`shredSrcMap`/`shredMaskMap`, `balloonSrcMap`/`balloonMaskMap`, `copterSrcMap`/`copterMaskMap`,
`dartSrcMap`/`dartMaskMap`, `ballSrcMap`/`ballMaskMap`, `dripSrcMap`/`dripMaskMap`,
`enemySrcMap`/`enemyMaskMap`, `fishSrcMap`/`fishMaskMap`, `clutterSrcMap`/`clutterMaskMap`,
`angelSrcMap`/`angelMaskMap`.

Sheets deliberately **without** a mask: `switchSrcMap` (PICT 5003 does not exist — switches are
drawn as opaque rectangles), `suppSrcMap` (the floor slab is opaque by design), `badgeSrcMap`,
`boardSrcMap` and the three scoreboard sub-maps.

### 8.2 Strategy B — on-demand temp GWorld pair (`CreateOffScreenGWorld` x2, `CopyMask`, dispose)

Used for large, rarely drawn objects whose art is not worth keeping resident. The pattern, verbatim
from `DrawPictWithMaskObject` (`GliderPRO/Sources/ObjectDraw2.c:1253-1298`):

```
1  GetGWorld(&wasCPort, &wasWorld)
2  pictID, maskID = <switch on what>
3  bounds = srcRects[what]
4  CreateOffScreenGWorld(&tempMap,  &bounds, kPreferredDepth /* 8 */)
5  SetGWorld(tempMap);   LoadGraphic(pictID)
6  CreateOffScreenGWorld(&tempMask, &bounds, 1)
7  SetGWorld(tempMask);  LoadGraphic(maskID)
8  CopyMask(tempMap, tempMask, backSrcMap,
           &srcRects[what], &srcRects[what], theRect)
9  SetGWorld(wasCPort, wasWorld)
10 DisposeGWorld(tempMap);  DisposeGWorld(tempMask)
```

`DrawPictWithMaskObject` handles exactly two types (`GliderPRO/Sources/ObjectDraw2.c:1267-1275`):

| `what` | art PICT | mask PICT |
|---|---|---|
| `kCobweb` 0x79 | `kCobwebPictID` 3958 | `kCobwebMaskID` 3927 |
| `kCloud` 0x8C | `kCloudPictID` 3965 | `kCloudMaskID` 3921 |

Six more functions inline the same pattern with their own IDs:

| Function | Line | art PICT | mask PICT |
|---|---|---|---|
| `DrawMailboxLeft` | `ObjectDraw2.c:172-189` | `kMailboxLeftPictID` 3986 | `kMailboxLeftMaskID` 3904 |
| `DrawMailboxRight` | `ObjectDraw2.c` (mirror of the above) | `kMailboxRightPictID` 3985 | `kMailboxRightMaskID` 3903 |
| `DrawTV` | `ObjectDraw2.c:653-670` | `kTVPictID` 3992 | `kTVMaskID` 3912 |
| `DrawVCR` | `ObjectDraw2.c:747-760` | `kVCRPictID` 3990 | `kVCRMaskID` 3913 |
| `DrawStereo` | `ObjectDraw2.c` | `kStereoPictID` 3989 | `kStereoMaskID` 3914 |
| `DrawMicrowave` | `ObjectDraw2.c` | `kMicrowavePictID` 3971 | `kMicrowaveMaskID` 3915 |

**Empirically verified**: of the 28 `#define`d mask IDs 3900-3927
(`GliderPRO/Sources/ObjectDraw2.c:39-66`), **exactly 8 PICT resources exist** in the application's
resource fork, and they are exactly the 8 that the code actually references:

| Present (8) | Referenced by |
|---|---|
| 3903 `kMailboxRightMaskID` | `DrawMailboxRight` |
| 3904 `kMailboxLeftMaskID` | `DrawMailboxLeft` |
| 3912 `kTVMaskID` | `DrawTV` |
| 3913 `kVCRMaskID` | `DrawVCR` |
| 3914 `kStereoMaskID` | `DrawStereo` |
| 3915 `kMicrowaveMaskID` | `DrawMicrowave` |
| 3921 `kCloudMaskID` | `DrawPictWithMaskObject` |
| 3927 `kCobwebMaskID` | `DrawPictWithMaskObject` |

The other **20 mask IDs are dead code and the resources are absent**: 3900 `kBBQMaskID`, 3901
`kUpStairsMaskID`, 3902 `kTrunkMaskID`, 3905 `kDoorInLeftMaskID`, 3906 `kDoorInRightMaskID`, 3907
`kWindowInLeftMaskID`, 3908 `kWindowInRightMaskID`, 3909 `kHipLampMaskID`, 3910 `kDecoLampMaskID`,
3911 `kGuitarMaskID`, 3916 `kFireplaceMaskID`, 3917 `kBearMaskID`, 3918 `kVase1MaskID`, 3919
`kVase2MaskID`, 3920 `kManholeMaskID`, 3922 `kBooksMaskID`, 3923 `kRugMaskID`, 3924
`kChimesMaskID`, 3925 `kCinderMaskID`, 3926 `kFlowerBoxMaskID`. Those 20 map one-for-one onto the
20 object types that `DrawPictSansWhiteObject` handles instead (strategy C) — clear evidence that
those objects were converted from explicit masks to white-keying during development, and the masks
were dropped from the resource fork but not from the header.

All 8 present mask PICTs are v1 PICTs with a 1-bit PixMap and no ColorTable (see 5.2), consistent
with being loaded into a depth-1 GWorld.

### 8.3 Strategy C — white-keying with QuickDraw's `transparent` transfer mode

`GliderPRO/Sources/ObjectDraw2.c:1302-1409`, `DrawPictSansWhiteObject`:

```
1  bounds = srcRects[what]
2  CreateOffScreenGWorld(&tempMap, &bounds, kPreferredDepth)
3  SetGWorld(tempMap);  LoadGraphic(pictID)
4  CopyBits(tempMap -> backSrcMap, &srcRects[what], theRect, transparent, nil)
5  SetGWorld(wasCPort, wasWorld);  DisposeGWorld(tempMap)
```

`transparent` (QuickDraw transfer mode **36**) copies every source pixel whose value differs from
the *source port's background colour*. The offscreen GWorld is created with the default background
of **white**, which in the standard 256-colour CLUT is index **0**, so in practice: **colour index
0 is the transparent key**. A Go port must therefore treat palette index 0 as the chroma key for
these 20 object types (and only for these) — not "RGB white", because a sprite may legitimately
contain a *different* index that also decodes to #FFFFFF; empirically the standard CLUT has exactly
one white (index 0), so the distinction is moot for the shipped art but matters for user PICTs.

The 20 types and their PICT IDs (`GliderPRO/Sources/ObjectDraw2.c:1313-1394`, IDs from
`GliderPRO/Sources/ObjectDraw2.c:67-103`):

| `what` | code | PICT | measured size |
|---|---|---|---|
| `kBBQ` | 0x0C | `kBBQPictID` 3988 | 64 x 33 |
| `kTrunk` | 0x1B | `kTrunkPictID` 3987 | 144 x 80 |
| `kManhole` | 0x1D | `kManholePictID` 3967 | 123 x 22 |
| `kBooks` | 0x1E | `kBooksPictID` 3964 | 64 x 51 |
| `kUpStairs` | 0x31 | `kUpStairsPictID` 3997 | 160 x 267 |
| `kDoorInLf` | 0x37 | `kDoorInLeftPictID` 3984 | 144 x 322 |
| `kDoorInRt` | 0x38 | `kDoorInRightPictID` 3983 | 144 x 322 |
| `kWindowInLf` | 0x3B | `kWindowInLeftPictID` 3980 | 20 x 170 |
| `kWindowInRt` | 0x3C | `kWindowInRightPictID` 3979 | 20 x 170 |
| `kHipLamp` | 0x54 | `kHipLampPictID` 3994 | 72 x 276 |
| `kDecoLamp` | 0x55 | `kDecoLampPictID` 3993 | 64 x 212 |
| `kGuitar` | 0x64 | `kGuitarPictID` 3991 | 64 x 172 |
| `kCinderBlock` | 0x6B | `kCinderPictID` 3960 | 40 x 62 |
| `kFlowerBox` | 0x6C | `kFlowerBoxPictID` 3959 | 80 x 32 |
| `kFireplace` | 0x84 | `kFireplacePictID` 3973 | 180 x 142 |
| `kBear` | 0x87 | `kBearPictID` 3972 | 56 x 58 |
| `kVase1` | 0x89 | `kVase1PictID` 3969 | 36 x 45 |
| `kVase2` | 0x8A | `kVase2PictID` 3968 | 35 x 57 |
| `kRug` | 0x8E | `kRugPictID` 3962 | 144 x 18 |
| `kChimes` | 0x8F | `kChimesPictID` 3961 | 28 x 74 |

Every one of these 20 measured PICT sizes matches its `srcRects` entry exactly (compare the table
in 7.4) — a useful invariant for a port's asset-loading assertions.

`DrawCustPictSansWhite(pictID, theRect)` (`GliderPRO/Sources/ObjectDraw2.c:1412-1436`) is the same
thing for `kCustomPict` (0x6E), except that the temp GWorld is sized from `*theRect` normalised by
`ZeroRectCorner` rather than from `srcRects`, because the size came from the house file.

### 8.4 Strategy D — plain opaque `srcCopy`

`DrawPictObject` (`GliderPRO/Sources/ObjectDraw2.c:1197-1249`) does not even create a temp GWorld:
it sets the port to `backSrcMap` and calls `DrawPicture` directly with the PICT's frame offset to
the object's top-left. Seven types:

| `what` | code | PICT | measured size |
|---|---|---|---|
| `kFilingCabinet` | 0x14 | `kFilingCabinetPictID` 3995 | 74 x 107 |
| `kDownStairs` | 0x32 | `kDownStairsPictID` 3996 | 160 x 267 |
| `kDoorExRt` | 0x39 | `kDoorExRightPictID` 3982 | 16 x 322 |
| `kDoorExLf` | 0x3A | `kDoorExLeftPictID` 3981 | 16 x 322 |
| `kWindowExRt` | 0x3D | `kWindowExRightPictID` 3977 | 16 x 170 |
| `kWindowExLf` | 0x3E | `kWindowExLeftPictID` 3978 | 16 x 170 |
| `kOzma` | 0x81 | `kOzmaPictID` 3975 | 102 x 92 |

```c
bounds = srcRects[what];
QOffsetRect(&bounds, theRect->left, theRect->top);   /* ObjectDraw2.c:1243-1244   */
DrawPicture(thePicture, &bounds);
```

Note that this uses `srcRects[what]` for the *size*, not the PICT's own `picFrame` — so if a
user replaced one of these PICTs with a different size the art would be **scaled** by `DrawPicture`.
That is the only implicit scaling path in the object layer.

`DrawPictObject` is also the only one of the four that does **not** null-check gracefully: it calls
`RedAlert(kErrFailedGraphicLoad)` on a nil handle (`GliderPRO/Sources/ObjectDraw2.c:1240-1241`).
Note the switch at :1205-1234 has no `default:` case, so `pictID` is used uninitialised if a
caller passes an unexpected `what` — a latent bug that cannot fire given the dispatcher's case list.

### 8.5 Two-state appliance indicators

Several appliances draw a small "on"/"off" swatch over the base sprite with a plain `srcCopy` from
`applianceSrcMap`. All rects are in `applianceSrcMap` (80 x 269) coordinates
(`GliderPRO/Sources/StructuresInit.c:565-605`):

| Rect | t,l,b,r | size | Meaning |
|---|---|---|---|
| `plusScreen1` | (127,48,149,80) | 32 x 22 | Mac Plus screen, off |
| `plusScreen2` | (149,48,171,80) | 32 x 22 | Mac Plus screen, on |
| `tvScreen1` | (171,0,220,64) | 64 x 49 | TV screen, off |
| `tvScreen2` | (220,0,269,64) | 64 x 49 | TV screen, on |
| `coffeeLight1` | (171,72,175,80) | 8 x 4 | coffee maker light, off |
| `coffeeLight2` | (175,72,179,80) | 8 x 4 | coffee maker light, on |
| `vcrTime1` | (179,64,183,80) | 16 x 4 | VCR clock, off |
| `vcrTime2` | (183,64,187,80) | 16 x 4 | VCR clock, on |
| `stereoLight1` | (171,68,172,72) | 4 x 1 | stereo LED, off |
| `stereoLight2` | (172,68,173,72) | 4 x 1 | stereo LED, on |
| `microOff` | (187,64,222,80) | 16 x 35 | microwave panel, off |
| `microOn` | (222,64,257,80) | 16 x 35 | microwave panel, on |
| `outletSrc[0..3]` | (22+24i, 64, 46+24i, 80) | 16 x 24 | outlet, 4 animation frames (`kNumOutletPicts` = **4**) |

and in `toastSrcMap` (32 x 174): `breadSrc[0..5]` = 32 x 29 at y `29*i`, `kNumBreadPicts` = **6**
(`GliderPRO/Sources/StructuresInit.c:586-590`); 6 x 29 = 174 exactly.

The indicator swatch is drawn *after* the masked base sprite and at a hardcoded offset from the
object's rect, e.g. `DrawTV` (`GliderPRO/Sources/ObjectDraw2.c:674-688`):

```
bounds = tvScreen1; ZeroRectCorner(&bounds);
QOffsetRect(&bounds, theRect->left + 17, theRect->top + 10);
CopyBits(applianceSrcMap -> backSrcMap, isOn ? &tvScreen2 : &tvScreen1,
         &bounds, srcCopy, nil);
```

and `DrawCoffee` (`GliderPRO/Sources/ObjectDraw2.c:705-719`) at `(left + 32, top + 57)`.
Crucially the indicator is drawn **even when `isLit` is false** (the `isLit` test wraps only the
`CopyMask` of the base sprite, `:649-672`), so a switched-on TV or coffee-maker light is visible in
a dark room. Same in `DrawVCR`, `DrawStereo`, `DrawMicrowave`, `DrawMacPlus`.

### 8.6 Colour-depth degradation

Every draw helper that picks its own colours branches on `thisMac.isDepth == 4` and substitutes
4-bit palette indices, e.g. `DrawMailboxLeft`
(`GliderPRO/Sources/ObjectDraw2.c:125-136`):

```c
if (thisMac.isDepth == 4)
    { darkGrayC = 13;  lightWoodC = 9;   darkWoodC = 11; }
else
    { darkGrayC = k8DkGray2Color /*253*/; lightWoodC = k8PissYellowColor /*95*/;
      darkWoodC = k8BrownColor /*137*/; }
```

The 8-bit named colour indices are `#define`d twice, once per file — `GliderPRO/Sources/ObjectDraw.c:16-48` (33 names) and `GliderPRO/Sources/ObjectDraw2.c:19-37` (a 19-name subset). Resolved against the measured `clut 128` (5.6):

| Constant | Index (dec) | Index (hex) | CLUT RGB |
|---|---|---|---|
| `k8WhiteColor` | 0 | 0x00 | `#FFFFFF` |
| `kYellowColor / kIntenseYellowColor` | 5 | 0x05 | `#FFFF00` |
| `kGoldColor` | 11 | 0x0B | `#FFCC00` |
| `k8RedColor` | 35 | 0x23 | `#FF0000` |
| `kPaleVioletColor` | 42 | 0x2A | `#CCCCFF` |
| `k8LtstGray3Color` | 43 | 0x2B | `#CCCCCC` |
| `k8LtTanColor` | 52 | 0x34 | `#CC9933` |
| `k8BambooColor` | 53 | 0x35 | `#CC9900` |
| `kDarkFleshColor` | 58 | 0x3A | `#CC6633` |
| `k8OrangeColor` | 59 | 0x3B | `#CC6600` |
| `k8TanColor` | 94 | 0x5E | `#996633` |
| `k8PissYellowColor` | 95 | 0x5F | `#996600` |
| `k8PumpkinColor` | 101 | 0x65 | `#993300` |
| `k8BrownColor` | 137 | 0x89 | `#663300` |
| `k8Red4Color` | 143 | 0x8F | `#660000` |
| `k8SkyColor` | 150 | 0x96 | `#33CCFF` |
| `k8EarthBlueColor` | 170 | 0xAA | `#333399` |
| `k8DkGray3Color` | 172 | 0xAC | `#333333` |
| `k8DkRedColor` | 222 | 0xDE | `#440000` |
| `k8DkRed2Color` | 223 | 0xDF | `#220000` |
| `kIntenseGreenColor` | 225 | 0xE1 | `#00EE00` |
| `kIntenseBlueColor` | 235 | 0xEB | `#0000EE` |
| `k8LtstGrayColor` | 245 | 0xF5 | `#EEEEEE` |
| `k8LtstGray2Color` | 246 | 0xF6 | `#DDDDDD` |
| `k8LtstGray4Color` | 247 | 0xF7 | `#BBBBBB` |
| `k8LtstGray5Color` | 248 | 0xF8 | `#AAAAAA` |
| `k8LtGrayColor` | 249 | 0xF9 | `#888888` |
| `k8GrayColor` | 250 | 0xFA | `#777777` |
| `k8Gray2Color` | 251 | 0xFB | `#555555` |
| `k8DkGrayColor` | 252 | 0xFC | `#444444` |
| `k8DkGray2Color` | 253 | 0xFD | `#222222` |
| `k8DkstGrayColor` | 254 | 0xFE | `#111111` |
| `k8BlackColor` | 255 | 0xFF | `#000000` |

The only naming divergence between the two files is index 5: `ObjectDraw.c` calls it
`kYellowColor`, `ObjectDraw2.c` calls it `kIntenseYellowColor`. Note also that the gray ramp is
**not** linear — it is `EE DD BB AA 88 77 55 44 22 11` (indices 245-254), matching the ten values
the Mac standard CLUT uses for its gray ramp.

Because the shipped `clut 128` gray ramp lives at 245-254 and the 6x6x6 colour cube at 0-214, a Go
port that renders in RGBA can simply resolve these indices through the CLUT once at load time.

---

## 9. The dirty-rect double buffer

### 9.1 The two queues

`GliderPRO/Sources/Render.c:20` and `:35-41`:

```c
#define kMaxGarbageRects		48

Rect	work2MainRects[kMaxGarbageRects];    /* work -> screen, this frame       */
Rect	back2WorkRects[kMaxGarbageRects];    /* back -> work,   this frame       */
short	numWork2Main, numBack2Work;
```

Both are **fixed-size arrays of 48 rects, silently overflow-guarded, never coalesced and never
sorted.** The invariant that makes the whole scheme work: *source rect == destination rect in every
blit*, because `backSrcMap`, `workSrcMap` and the window's port all share one coordinate system
(see 1 and 2). So a "dirty rect" is a single rect, not a src/dst pair.

Semantics:

| Queue | Purpose |
|---|---|
| `work2MainRects` | regions of `workSrcMap` that changed this frame and must be pushed to the screen |
| `back2WorkRects` | regions of `workSrcMap` that were painted over by a moving sprite and must be restored from the clean `backSrcMap` **at the start of the next frame's copy phase** |

### 9.2 `AddRectToWorkRects` — clamps to `justRoomsRect`

`GliderPRO/Sources/Render.c:65-80`:

```
1  if numWork2Main >= kMaxGarbageRects - 1:  return            /* silently drop!  */
2  r = *theRect
3  if r.left   < justRoomsRect.left   : r.left   = justRoomsRect.left
4  elif r.right > justRoomsRect.right : r.right  = justRoomsRect.right
5  if r.top    < justRoomsRect.top    : r.top    = justRoomsRect.top
6  elif r.bottom > justRoomsRect.bottom: r.bottom = justRoomsRect.bottom
7  work2MainRects[numWork2Main++] = r
```

Three traps for a porter:

* **The guard is `< kMaxGarbageRects - 1`, i.e. at most 47 entries are ever stored.** Slot 47 is
  never used. This is a fencepost bug in the original that is harmless but must be replicated if
  you want bit-identical overflow behaviour.
* **The clamps are `if/else if`, not two independent `if`s.** A rect that is simultaneously too far
  left *and* too far right (i.e. wider than `justRoomsRect`) has only its left edge clamped. Same
  vertically. In practice no sprite is 1536 px wide, so it never fires — but a naive port using two
  independent clamps is *more* correct and therefore different.
* **Clamping is to `justRoomsRect`, not to the window.** `justRoomsRect` excludes the scoreboard
  band, so gameplay blits can never overwrite the score display. That is the sole mechanism
  protecting the scoreboard.

### 9.3 `AddRectToBackRects` — clamps to `workSrcRect`

`GliderPRO/Sources/Render.c:84-99`. Identical structure, but the clamp box is
`(0, 0, workSrcRect.right, workSrcRect.bottom)` — the whole work map, scoreboard included, because
this queue only ever *restores* clean background and can never leak sprite pixels.

### 9.4 `AddRectToWorkRectsWhole` — clamp with a real reject test

`GliderPRO/Sources/Render.c:103-132`. Used where the rect may be entirely outside the map:

```
1  if numWork2Main >= kMaxGarbageRects - 1: return
2  if (r.right <= workSrcRect.left) || (r.bottom <= workSrcRect.top) ||
     (r.left  >= workSrcRect.right) || (r.top    >= workSrcRect.bottom):  return   /* :107-111 */
3  <same if/else-if clamp as 9.2 but against workSrcRect>                          /* :115-122 */
4  if (r.right == r.left) || (r.top == r.bottom): return    /* degenerate, drop    */
5  work2MainRects[numWork2Main++] = r
```

Note that in step 4 the entry is **not** stored but `numWork2Main` is also not incremented, so
nothing leaks. Also note the asymmetry: this variant clamps to `workSrcRect` (whole map) while
`AddRectToWorkRects` clamps to `justRoomsRect` (rooms only), even though both push to the screen.
`AddRectToWorkRectsWhole` is therefore the function to use when you *do* want to update the
scoreboard band.

### 9.5 `CopyRectsQD` — the copy phase

`GliderPRO/Sources/Render.c:616-635`. Two loops, in this order:

```
1  for i in 0 .. numWork2Main-1:
2      CopyBits(workSrcMap -> GetPortBitMapForCopyBits(GetWindowPort(mainWindow)),
                &work2MainRects[i], &work2MainRects[i], srcCopy, nil)
3  for i in 0 .. numBack2Work-1:
4      CopyBits(backSrcMap -> workSrcMap,
                &back2WorkRects[i], &back2WorkRects[i], srcCopy, nil)
```

The order is essential: **show the frame first, then erase**. The erase-from-back pass prepares
`workSrcMap` for the *next* frame; it happens after the pixels have already been shown. This is why
`back2WorkRects` is filled during the same frame as `work2MainRects` and consumed at the end of it.

Rects are copied in insertion order with no overlap detection, so an overlapping region is blitted
multiple times per frame. With 47 max rects of typical sprite size this is cheap.

A sibling `CopyRectsAssm` is forward-declared at `GliderPRO/Sources/Render.c:32` and **never defined
or called** — the vestige of a hand-written 68k blitter.

### 9.6 Who fills the queues

| Producer | Adds to work2Main | Adds to back2Work | Line |
|---|---|---|---|
| `DrawReflection` | `thisGlider->whole` + origin, offset (−20,−16) | `dest` | :187-188 |
| `RenderFlames` (candles) | `flames[i].dest` | — | :216 |
| `RenderFlames` (tikis) | `tikiFlames[i].dest` | — | :235 |
| `RenderFlames` (coals) | `bbqCoals[i].dest` | — | :254 |
| `RenderPendulums` | `pendulums[i].dest` | — | :317 |
| `RenderFlyingPoints` (expiring) | `flyingPoints[i].dest` | — | :344 |
| `RenderFlyingPoints` (moving) | `flyingPoints[i].whole` | `flyingPoints[i].dest` | :373-374 |
| `RenderSparkles` (expiring) | `sparkles[i].bounds` | — | :397 |
| `RenderSparkles` (live) | `sparkles[i].bounds` | `sparkles[i].bounds` | :410-411 |
| `RenderStars` | `theStars[i].dest` | — | :445 |
| `RenderGlider` (shadow) | `wholeShadow` + origin | `dest` | :498-499 |
| `RenderGlider` (glider) | `whole` + origin | `dest` | :528-529 |
| `RenderBands` | `dest` | `dest` | :552-553 |
| `RenderShreds` (frame 0) | `dest` with `top--` | `dest` | :582-584 |
| `RenderShreds` (frames 1-19) | `dest` with `top -= 4` | `dest` | :606-608 |
| `RenderDynamics` | per dynamic object | per dynamic object | `Dynamics.c` |
| `HandleGrease` | grease rects | grease rects | `Grease.c` |
| `RedrawRoomLighting` | `localRoomsDest[kCentralRoom]` | — | `RoomGraphics.c:458` |

**Elements drawn from `savedMaps` add only to `work2Main`, never to `back2Work`.** That is the whole
point of pre-composition (section 12): the swatch already contains the background, so no erase is
needed. Elements drawn with `CopyMask` from a live sheet add to **both**.

**The "whole" rect trick.** Sprites that move (glider, flying points) keep two rects: `dest` (this
frame's position) and `whole` (the union of last frame's and this frame's position). `whole` goes to
the screen so the trailing edge is repainted; `dest` goes to the erase queue so only the current
footprint is restored. E.g. `RenderFlyingPoints` (`:350-375`):

```
dest.left += hVel;  dest.right += hVel
if hVel > 0:  whole.right  = dest.right      else: whole.left = dest.left
dest.top  += vVel;  dest.bottom += vVel
if vVel > 0:  whole.bottom = dest.bottom     else: whole.top  = dest.top
<CopyMask into workSrcMap at dest>
AddRectToWorkRects(&whole)
AddRectToBackRects(&dest)
whole = dest                                  /* reset for next frame           */
```

`RenderShreds` uses a cruder version of the same idea: it just extends `dest.top` upward by the
per-frame velocity (1 px in the growth phase, 4 px in the falling phase) before pushing to the
screen queue (`:583`, `:607`).

---

## 10. `RenderFrame` — the per-frame composition order

`GliderPRO/Sources/Render.c:639-671`. This is the entire per-frame renderer; it is called once per
game tick from `Play.c`.

```
 1  if hasMirror:                                  /* :641                        */
 2      DrawReflection(&theGlider,  true)
 3      if twoPlayerGame: DrawReflection(&theGlider2, false)
 4  HandleGrease()                                 /* :647  Grease.c              */
 5  RenderPendulums()                              /* :648                        */
 6  if evenFrame: RenderFlames()                   /* :649-650                    */
 7  else:         RenderStars()                    /* :651-652                    */
 8  RenderDynamics()                               /* :653  Dynamics.c            */
 9  RenderFlyingPoints()                           /* :654                        */
10  RenderSparkles()                               /* :655                        */
11  RenderGlider(&theGlider, true)                 /* :656                        */
12  if twoPlayerGame: RenderGlider(&theGlider2, false)  /* :657-658                */
13  RenderShreds()                                 /* :659                        */
14  RenderBands()                                  /* :660                        */
15  while (TickCount() < nextFrame) { }             /* :662-664  busy-wait!        */
16  nextFrame = TickCount() + kTicksPerFrame        /* :665                        */
17  CopyRectsQD()                                  /* :667                        */
18  numWork2Main = 0;  numBack2Work = 0            /* :669-670                    */
```

Observations a porter must preserve:

* **Z-order is exactly the call order.** Mirror reflections are bottom-most (they are drawn *into*
  the mirror region and then the mirror's own content is not re-drawn); rubber bands are top-most,
  above even the glider. There is no depth sort.
* **Flames and stars alternate on the `evenFrame` flag** (`GliderPRO/Sources/Render.c:649`), so each
  animates at ~15 fps while the frame rate is ~30 fps. `evenFrame` is toggled elsewhere in the play
  loop.
* **The busy-wait happens *before* the copy, not after.** So the pacing target is
  "start-of-composition to start-of-composition", and the blit-to-screen time is effectively free
  slack. On a fast machine the wait absorbs everything; on a slow one the game just runs slow with
  no frame dropping.
* `kTicksPerFrame` = **2** (`GliderPRO/Headers/GliderDefines.h:533`). A Mac tick is 1/60.15 s, so
  the target is 60.15/2 = **30.07 fps**, i.e. 33.25 ms per frame.
* **`TickCount()` is read twice** (:662 and :665), so `nextFrame` is computed from the *actual* wake
  time, not from `nextFrame + kTicksPerFrame`. The frame clock therefore drifts forward whenever a
  frame overruns and never tries to catch up. A Go port using
  `nextFrame = nextFrame.Add(period)` will be subtly different (it would try to catch up); to match
  the original use `nextFrame = time.Now().Add(period)` after the wait.
* Both queue counters are reset **after** the copy, so a producer that runs outside `RenderFrame`
  (e.g. `RedrawRoomLighting`) can enqueue and have it picked up by the next `RenderFrame`.

`InitGarbageRects` (`GliderPRO/Sources/Render.c:675-691`) resets both counters, clears all
`kMaxSparkles` = **3** sparkle slots and all `kMaxFlyingPts` = **3** flying-point slots to
`mode = -1`, and primes `nextFrame = TickCount() + kTicksPerFrame`.

### 10.1 The five one-shot copy helpers

`GliderPRO/Sources/Render.c:695-736` — plain immediate blits with src == dst, used outside the
per-frame path (transitions, editor, scoreboard, map):

| Function | Direction | Line |
|---|---|---|
| `CopyRectBackToWork` | `backSrcMap` -> `workSrcMap` | :695-700 |
| `CopyRectWorkToBack` | `workSrcMap` -> `backSrcMap` | :704-709 |
| `CopyRectWorkToMain` | `workSrcMap` -> window | :713-718 |
| `CopyRectMainToWork` | window -> `workSrcMap` | :722-727 |
| `CopyRectMainToBack` | window -> `backSrcMap` | :731-736 |

Two of them read *from the screen* — a Go port with a write-only framebuffer must keep a readable
copy of the window contents, or restructure the callers.

---

## 11. The animated-element renderers

`RenderFrame` (`GliderPRO/Sources/Render.c:639-671`) calls twelve sub-renderers. This section
documents each one at statement level. They all write into `workSrcMap` and register dirty rects;
none of them touch the screen.

### 11.1 The per-element state arrays

All of these are heap blocks allocated once in `CreatePointers`
(`GliderPRO/Sources/StructuresInit2.c:187-299`) and never resized. `savedMaps` is *also* declared
as a static array in `GliderPRO/Sources/Objects.c:73` (`savedType savedMaps[kMaxSavedMaps];`) —
the pointer version at `StructuresInit2.c:232-233` shadows it in that translation unit, an
inconsistency a Go port should collapse to one slice.

| Array | Element type | Count constant | Value | Allocation |
|---|---|---|---|---|
| `hotSpots` | `hotObject` | `kMaxHotSpots` | 56 | `StructuresInit2.c:198` |
| `sparkles` | `sparkleType` | `kMaxSparkles` | 3 | `StructuresInit2.c:203` |
| `flyingPoints` | `flyingPtType` | `kMaxFlyingPts` | 3 | `StructuresInit2.c:208` |
| `flames` | `flameType` | `kMaxCandles` | 20 | `StructuresInit2.c:213` |
| `tikiFlames` | `flameType` | `kMaxTikis` | 8 | `StructuresInit2.c:218` |
| `bbqCoals` | `flameType` | `kMaxCoals` | 8 | `StructuresInit2.c:223` |
| `pendulums` | `pendulumType` | `kMaxPendulums` | 8 | `StructuresInit2.c:228` |
| `savedMaps` | `savedType` | `kMaxSavedMaps` | 24 | `StructuresInit2.c:233` |
| `bands` | `bandType` | `kMaxRubberBands` | 2 | `StructuresInit2.c:241` |
| `grease` | `greaseType` | `kMaxGrease` | 16 | `StructuresInit2.c:246` |
| `theStars` | `starType` | `kMaxStars` | 4 | `StructuresInit2.c:251` |
| `shreds` | `shredType` | `kMaxShredded` | 4 | `StructuresInit2.c:256` |
| `dinahs` | `dynaType` | `kMaxDynamicObs` | 18 | `StructuresInit2.c:261` |
| `masterObjects` | `objDataType` | `kMaxMasterObjects` | 216 | `StructuresInit2.c:266` |
| `srcRects` | `Rect` | `kNumSrcRects` | 144 (0x90) | `StructuresInit2.c:271` |

Struct layouts (`GliderPRO/Sources/Headers/GliderStructs.h`; `Rect` = 8 bytes = four big-endian
`short`s in the order `top, left, bottom, right`; `short` = 2 bytes; `Boolean` = 1 byte):

| Type | Header line | Fields (in order) | Size |
|---|---|---|---|
| `savedType` | :227-233 | `Rect dest` (8); `GWorldPtr map` (4); `short where` (2); `short who` (2) | 16 |
| `sparkleType` | :235-239 | `Rect bounds` (8); `short mode` (2) | 10 |
| `flyingPtType` | :241-249 | `Rect dest` (8), `Rect whole` (8); `short start, stop, mode, loops, hVel, vVel` (12) | 28 |
| `flameType` | :251-256 | `Rect dest` (8), `Rect src` (8); `short mode, who` (4) | 20 |
| `pendulumType` | :258-264 | `Rect dest` (8), `Rect src` (8); `short mode, where, who, link` (8); `Boolean toOrFro, active` (2) | 26 |
| `bandType` | :274-279 | `Rect dest` (8); `short mode, count, hVel, vVel` (8) | 16 |
| `greaseType` | :287-295 | `Rect dest` (8); `short mapNum, mode, who, where, start, stop, frame, hotNum` (16); `Boolean isRight` (1) | 25→26 padded |
| `starType` | :297-302 | `Rect dest`, `Rect src` (16); `short mode, who, link, where` (8) | 24 |
| `shredType` | :304-308 | `Rect bounds` (8); `short frame` (2) | 10 |
| `dynaType` | :310-320 | `Rect dest` (8), `Rect whole` (8); `short hVel, vVel, type, count, frame, timer, position, room` (16); `Byte byte0, byte1` (2); `Boolean moving, active` (2) | 36 |

These structs are pure in-memory scratch; none of them is serialised to disk, so a Go port is free
to lay them out however it likes.

### 11.2 The `savedMaps` pre-composition trick

This is the single most important idea in the whole renderer and it is easy to miss.

For the *cycling* animated elements (candle flames, tiki flames, BBQ coals, cuckoo pendulums,
stars, grease jars) Glider PRO does **not** mask a sprite over the background every frame. Instead,
at room-setup time it allocates a tall private GWorld holding *N vertically stacked, fully
composited cells*: cell *i* is (background swatch from `backSrcMap`) with (frame *i* of the sprite
masked on top). Per frame the renderer then does one plain opaque `CopyBits(srcCopy)` of one cell
into `workSrcMap`. No mask, no erase-behind, no `back2WorkRects` entry.

`BackUpToSavedMap` (`GliderPRO/Sources/DynamicMaps.c:70-93`) is the allocator:

```
 1  if (numSavedMaps >= kMaxSavedMaps)          -> return -1        // 24 slots, hard cap
 2  mapRect = *theRect;  ZeroRectCorner(&mapRect)                    // move to (0,0)
 3  savedMaps[numSavedMaps].dest = *theRect
 4  CreateOffScreenGWorld(&savedMaps[n].map, &mapRect, kPreferredDepth)   // depth 8
 5  CopyBits(backSrcMap -> savedMaps[n].map, theRect, &mapRect, srcCopy)
 6  savedMaps[n].where = where;  savedMaps[n].who = who
 7  numSavedMaps++;  return numSavedMaps - 1
```

Two things to note: the `CreateOffScreenGWorld` return code is captured into `theErr` and **never
checked** (`DynamicMaps.c:82`), and step 5 copies the *background* even for the multi-cell cases
where the caller is about to overwrite every cell anyway.

`ReBackUpSavedMap` (`DynamicMaps.c:100-124`) is the "lights just changed" variant: it linearly
searches `savedMaps[0..numSavedMaps-1]` for the `(where, who)` pair, re-copies `backSrcMap` into
cell 0 and returns the index, or `-1` if not found.

`RestoreFromSavedMap(where, who, doSparkle)` (`DynamicMaps.c:131-163`) is the "player got the
prize, erase it" path. It copies the saved swatch **into both `backSrcMap` and `workSrcMap`**
(:144-149), pushes `savedMaps[i].dest` onto `work2MainRects` (:151), and when `doSparkle` is true
offsets the rect back to room space by `(-playOriginH, -playOriginV)`, calls `AddSparkle` and plays
`kFadeOutSound` at `kFadeOutPriority` (:153-159). Writing to `backSrcMap` is what makes the erase
permanent.

`NilSavedMaps` (`DynamicMaps.c:46-62`) disposes every non-nil GWorld, sets `where = who = -1` and
`numSavedMaps = 0`.

`ZeroFlamesAndTheLike` (`DynamicMaps.c:787-797`) resets, in this order, `numFlames`,
`numTikiFlames`, `numCoals`, `numPendulums`, `numGrease`, `numStars`, `numShredded`, `numChimes` —
called before a room is drawn. Note it does **not** reset `numSavedMaps`; that is `NilSavedMaps`'
job.

#### 11.2.1 The five multi-cell builders

Every builder has the same shape: a `dest` cursor rect starting at `(0,0,w,h)`, a loop over the
frame count, `CopyBits(backSrcMap -> savedMaps[index].map, src, dest)` then
`CopyMask(sheet, mask -> savedMaps[index].map, frameRect, frameRect, dest)`, then
`QOffsetRect(&dest, 0, h)`.

| Builder | Line | Cell size | Cells | Sheet / mask | Frame rect array | Total map size |
|---|---|---|---|---|---|---|
| `BackUpFlames` | `DynamicMaps.c:263-285` | 16 x 15 | `kNumCandleFlames` = 5 | `blowerSrcMap` / `blowerMaskMap` | `flame[i]` | 16 x 75 |
| `BackUpTikiFlames` | `DynamicMaps.c:349-370` | 8 x 10 | `kNumTikiFlames` = 5 | `blowerSrcMap` / `blowerMaskMap` | `tikiFlame[i]` | 8 x 50 |
| `BackUpBBQCoals` | `DynamicMaps.c:435-456` | 32 x 9 | `kNumBBQCoals` = 4 | `blowerSrcMap` / `blowerMaskMap` | `coals[i]` | 32 x 36 |
| `BackUpPendulum` | `DynamicMaps.c:521-540` | 32 x 28 | `kNumPendulums` = **3** | `bonusSrcMap` / `bonusMaskMap` | `pendulumSrc[i]` | 32 x 84 |
| `BackUpStar` | `DynamicMaps.c:612-632` | 32 x 31 | hard-coded **6** | `bonusSrcMap` / `bonusMaskMap` | `starSrc[i]` | 32 x 186 |
| `BackupGrease` | `GliderPRO/Sources/Grease.c:141-172` | 32 x 27 | hard-coded **4** | `bonusSrcMap` / `bonusMaskMap` | `greaseSrcRt[i]` or `greaseSrcLf[i]` | 32 x 108 |

Two constants that are easy to confuse: `kMaxPendulums` = 8 is the number of *cuckoo clocks* the
room may contain; `kNumPendulums` = 3 (`GliderPRO/Sources/Headers/GliderDefines.h:450`) is the
number of *animation frames* per clock. `BackUpStar` and `BackupGrease` use bare literals `6` and
`4` rather than `kNumStars`/`kNumGrease` — there is no such constant.

`BackupGrease` is the only builder whose loop also advances the *source* rect: it walks `src` by
`(±2, 0)` per cell (`Grease.c:159` / `:167`) so the four jar-tipping frames sample progressively
shifted background, matching the jar leaning over. `isRight` selects `greaseSrcRt` vs
`greaseSrcLf`.

#### 11.2.2 The 4-bit even-x alignment rule

At `thisMac.isDepth == 4` (16-colour screen) two pixels share a byte, and QuickDraw's blitter was
faster — and, in this code, only correct — when the source x is byte-aligned. Every `Add*` builder
therefore forces the destination x even:

| Function | Line | Rule |
|---|---|---|
| `AddCandleFlame` | `DynamicMaps.c:326-331` | after `QOffsetRect(&src, h-8, v-15)`: if `src.left` odd, `QOffsetRect(&src,-1,0)`; if that made `src.left < 0`, `QOffsetRect(&src,+2,0)` |
| `AddTikiFlame` | `DynamicMaps.c:409-414` | if `h` odd then `h--`; if `h < 0` then `h += 2` |
| `AddBBQCoals` | `DynamicMaps.c:495-500` | same as tiki |
| `AddPendulum` | `DynamicMaps.c:584-589` | same as tiki |
| `AddStar` | `DynamicMaps.c:671-676` | same as tiki |

`AddGrease` has **no** such adjustment (`Grease.c:206-251`), which is either a bug or an accepted
imperfection on 16-colour screens. A Go port on a 32-bit framebuffer should drop all five rules;
they change sprite position by one pixel, so reproducing them is only necessary for
bug-for-bug fidelity of a 4-bit mode a Go port will not have.

#### 11.2.3 The six registration functions

```
AddCandleFlame(where, who, h, v)                      DynamicMaps.c:316-344
 1  if numFlames >= kMaxCandles(20) or h < 16 or v < 15 -> return
 2  src = (0,0,16,15) offset by (h-8, v-15)            // caller passes the flame tip
 3  [4-bit even-x fixup]
 4  bounds = (0,0,16, 15*5)                            // 16 x 75
 5  savedNum = BackUpToSavedMap(&bounds, where, who);  if -1 -> return
 6  BackUpFlames(&src, savedNum)
 7  flames[n].dest = src
 8  flames[n].mode = RandomInt(5)                      // random starting phase
 9  flames[n].src  = (0,0,16,15) offset by (0, mode*15)
10  flames[n].who  = savedNum;  numFlames++
```

`AddTikiFlame` (`:400-429`) is identical with 8 x 10 cells, `kMaxTikis`(8), guards `h < 8 || v < 10`,
`bounds = (0,0,8,50)`, `RandomInt(5)`, and — unlike the candle — offsets `src` by `(h, v)`
directly, i.e. **the caller supplies the top-left, not the tip**.

`AddBBQCoals` (`:486-515`): 32 x 9, `kMaxCoals`(8), guards `h < 32 || v < 9`,
`bounds = (0,0,32,36)`, `RandomInt(4)`, `src` offset by `(h, v)`.

`AddPendulum` (`:570-606`): 32 x 28, `kMaxPendulums`(8), guards `h < 32 || v < 28`. Sets the global
`clockFrame = 10` (:578) — see 11.4. `bounds = (0,0,32,84)`. Start state is
`mode = 1`, `src = (0,0,32,28)` offset `(0,28)` i.e. **the middle cell**, `toOrFro = RandomInt(2)==0`,
`active = true`, and it records `where` and `link = who` so `StopPendulum` can find it.

`AddStar` (`:662-694`): 32 x 31, `kMaxStars`(4), **no h/v guard**, `bounds = (0,0,32,186)`,
`mode = RandomInt(6)`, `src` offset `(0, mode*31)`, records `link = who` and `where`.

`AddGrease(where, who, h, v, distance, isRight)` (`Grease.c:206-251`): `kMaxGrease`(16),
`src = (0,0,32,27)` offset `(h,v)`, `bounds = (0,0,32,108)`. After `BackupGrease` it *rewinds*
`src` by `(-8,0)` when `isRight` else `(+8,0)` (:223-226) — because `BackupGrease` advanced it four
times by ±2. Then `mode = kGreaseIdle`(0), `frame = -1`, and the spill extent:
`isRight` -> `start = src.right + 4`, `stop = src.right + distance`; else
`start = src.left - 4`, `stop = src.left - distance`.

The four `ReBackUp*` functions (`DynamicMaps.c:292-310`, `:376-394`, `:462-480`, `:546-564`;
`Grease.c:180-199`) all have the same double loop: outer over `savedMaps` matching `(where, who)`,
inner over the element array matching `element.who == i`, then re-run the corresponding
`BackUp*` builder and `return`. `ReBackUpGrease` additionally only rebuilds when
`mode == kGreaseIdle || mode == kGreaseFalling` (`Grease.c:189`) and returns the *grease* index,
not the savedMap index.

`StopPendulum(where, who)` (`DynamicMaps.c:700-709`) sets `active = false` on every pendulum whose
`link == who && where == where`. `StopStar(where, who)` (`:715-724`) sets `mode = -1`. Neither
frees the savedMap.

### 11.3 `RenderFlames` — candles, tikis and coals

`GliderPRO/Sources/Render.c:193-256`. One function, three identical loops.

```
 1  if numFlames == 0 and numTikiFlames == 0 and numCoals == 0 -> return
 2  for i in 0 .. numFlames-1:
 3      flames[i].mode++
 4      flames[i].src.top += 15;  flames[i].src.bottom += 15
 5      if flames[i].mode >= kNumCandleFlames(5):
 6          mode = 0;  src.top = 0;  src.bottom = 15
 7      CopyBits(savedMaps[flames[i].who].map -> workSrcMap,
 8               &flames[i].src, &flames[i].dest, srcCopy, nil)
 9      AddRectToWorkRects(&flames[i].dest)
10  for i in 0 .. numTikiFlames-1:   ... same, stride 10, kNumTikiFlames(5) ...
11  for i in 0 .. numCoals-1:        ... same, stride  9, kNumBBQCoals(4)  ...
```

Points a porter must not miss:

- The cell stride is baked into the loop as a literal (`15`, `10`, `9`) and the wrap resets
  `src.top = 0; src.bottom = <cellH>` — so `src` is always exactly one cell tall.
- `dest` is in **work-map coordinates already** (the `Add*` callers were passed screen-space
  h/v including `playOrigin`), so there is no `QOffsetRect` here.
- Only `AddRectToWorkRects` is called — never `AddRectToBackRects`. The pre-composited cell
  contains its own background, so the previous frame needs no erasing.
- Flames advance one frame per *even* game frame only, because `RenderFrame` gates
  `RenderFlames` behind `if (evenFrame)` (`Render.c:649-650`). Effective flame rate is
  ~15 Hz, not ~30 Hz.

### 11.4 `RenderPendulums`

`GliderPRO/Sources/Render.c:260-321`.

```
 1  playedTikTok = false
 2  if numPendulums == 0 -> return
 3  clockFrame++
 4  if clockFrame != 10 and clockFrame != 15 -> return          // only 2 of every 15 frames
 5  if clockFrame >= 15 -> clockFrame = 0
 6  for i in 0 .. numPendulums-1:
 7      if not pendulums[i].active -> continue
 8      if pendulums[i].toOrFro:
 9          mode++;  src.top += 28;  src.bottom += 28
10          if mode >= 2:  toOrFro = !toOrFro
11                         if !playedTikTok: PlayPrioritySound(kTikSound, kTikPriority)
12                                           playedTikTok = true
13      else:
14          mode--;  src.top -= 28;  src.bottom -= 28
15          if mode <= 0:  toOrFro = !toOrFro
16                         if !playedTikTok: PlayPrioritySound(kTokSound, kTokPriority)
17                                           playedTikTok = true
18      CopyBits(savedMaps[pendulums[i].who].map -> workSrcMap,
19               &pendulums[i].src, &pendulums[i].dest, srcCopy, nil)
20      AddRectToWorkRects(&pendulums[i].dest)
```

The timing is the subtle part. `clockFrame` counts *every* `RenderFrame` in which at least one
pendulum exists, and the body only runs when `clockFrame` is exactly 10 or exactly 15; when it is
15 it wraps to 0. So the swing pattern is: frames 10 and 15 of each 15-frame cycle, i.e. a 5-frame
gap then a 10-frame gap, forever. `AddPendulum` seeds `clockFrame = 10`
(`DynamicMaps.c:578`) so the first swing happens on the very next rendered frame. At ~30 fps
(`kTicksPerFrame` 2) that is roughly 2 ticks of the pendulum per second with deliberately uneven
spacing — the "tick... tock" cadence.

`mode` walks 1 -> 2 -> 1 -> 0 -> 1 ... so cells 0, 1, 2 are all used, but the reversal happens
*after* the increment/decrement, meaning cell 2 and cell 0 are each displayed once per half-cycle.
`playedTikTok` ensures only one sound plays even with eight clocks in the room.

### 11.5 `RenderFlyingPoints`

`GliderPRO/Sources/Render.c:325-380`. The floating score numerals. Unlike the flame family these
*are* masked per frame, out of `pointsSrcMap`/`pointsMaskMap`.

```
 1  if numFlyingPts == 0 -> return
 2  for i in 0 .. kMaxFlyingPts-1 (3):
 3      if flyingPoints[i].mode == -1 -> continue          // -1 == free slot
 4      if mode > stop:  mode = start;  loops++            // cycle the 3-frame wiggle
 5      if loops >= kMaxFlyingPointsLoop(24):
 6          AddRectToWorkRects(&dest);  mode = -1;  numFlyingPts--   // expire
 7      else:
 8          dest.left += hVel;  dest.right += hVel
 9          if hVel > 0: whole.right = dest.right   else: whole.left = dest.left
10          dest.top += vVel;  dest.bottom += vVel
11          if vVel > 0: whole.bottom = dest.bottom else: whole.top = dest.top
12          CopyMask(pointsSrcMap, pointsMaskMap -> workSrcMap,
13                   &pointsSrc[mode], &pointsSrc[mode], &dest)
14          AddRectToWorkRects(&whole)      // union of old and new position
15          AddRectToBackRects(&dest)       // erase-behind next frame
16          whole = dest
17          mode++
```

`whole` is the classic "swept rect" trick: steps 9 and 11 extend the *old* rect in the direction of
travel so that a single work->main blit covers both the vacated and the newly occupied pixels; then
step 16 collapses it back to `dest` for next time. This is why `AddRectToWorkRects` gets `whole`
but `AddRectToBackRects` gets `dest` — the erase only needs to cover where the sprite actually is.

`AddFlyingPoint(theRect, points, hVel, vVel)` (`DynamicMaps.c:199-254`):

```
 1  if numFlyingPts >= kMaxFlyingPts(3) -> return
 2  theRect is mutated in place: += (playOriginH, playOriginV) on all four edges
 3  centeredRect = pointsSrc[0];  CenterRectInRect(&centeredRect, theRect)
 4  find first i with flyingPoints[i].mode == -1
 5  dest = whole = centeredRect;  loops = 0;  hVel = hVel;  vVel = vVel
 6  switch (points): 100 -> start 12 stop 14
                     250 -> start  9 stop 11
                     300 -> start  6 stop  8
                     500 -> start  3 stop  5
                     default -> start 0 stop 2
 7  mode = start;  numFlyingPts++
```

So `pointsSrc[0..14]` (15 rects of 24 x 8 at y = 8*i, `GliderPRO/Sources/StructuresInit.c:412-416`)
is five groups of three animation frames, one group per point value. Group order top to bottom is
`default`(0-2), `500`(3-5), `300`(6-8), `250`(9-11), `100`(12-14). Note the caller's `Rect` is
modified in place — a Go port passing a struct by value silently changes behaviour for any caller
that reuses the rect afterwards.

### 11.6 `RenderSparkles`

`GliderPRO/Sources/Render.c:384-416`.

```
 1  if numSparkles == 0 -> return
 2  for i in 0 .. kMaxSparkles-1 (3):
 3      if sparkles[i].mode == -1 -> continue
 4      if sparkles[i].mode >= kNumSparkleModes(5):
 5          AddRectToWorkRects(&bounds);  mode = -1;  numSparkles--
 6      else:
 7          CopyMask(bonusSrcMap, bonusMaskMap -> workSrcMap,
 8                   &sparkleSrc[mode], &sparkleSrc[mode], &bounds)
 9          AddRectToWorkRects(&bounds);  AddRectToBackRects(&bounds);  mode++
```

A sparkle is a one-shot 5-frame flash that never moves, so `bounds` serves as both `dest` and
`whole`. `sparkleSrc` is a 5-entry ping-pong table built at `StructuresInit.c:395-401`: three
distinct 20 x 19 rects at `(0, 70+19i)` for i = 2, 3, 4, with `sparkleSrc[0] = sparkleSrc[4]` and
`sparkleSrc[1] = sparkleSrc[3]`. Playing indices 0..4 therefore shows frames 4, 3, 2, 3, 4 — the
sparkle grows and shrinks using only three bitmaps.

`AddSparkle(theRect)` (`DynamicMaps.c:169-193`) mirrors `AddFlyingPoint`: guard on
`kMaxSparkles`, mutate `theRect` by `+playOrigin` in place, `centeredRect = sparkleSrc[0]`
centred in it, first free slot, `mode = 0`, `numSparkles++`.

### 11.7 `RenderStars`

`GliderPRO/Sources/Render.c:420-448`. Structurally identical to a `RenderFlames` loop but with a
hard-coded frame count of 6 and a stride of 31, and with a `mode == -1` skip so `StopStar` can kill
a star in place:

```
 1  if numStars == 0 -> return
 2  for i in 0 .. numStars-1:
 3      if theStars[i].mode == -1 -> continue
 4      mode++;  src.top += 31;  src.bottom += 31
 5      if mode >= 6:  mode = 0;  src.top = 0;  src.bottom = 31
 6      CopyBits(savedMaps[theStars[i].who].map -> workSrcMap, &src, &dest, srcCopy, nil)
 7      AddRectToWorkRects(&dest)
```

Stars animate on **odd** frames only: `RenderFrame` runs `RenderFlames()` when `evenFrame` and
`RenderStars()` otherwise (`Render.c:649-652`). Note the consequence — in a room containing both a
candle and a star, the two never update on the same frame, and neither ever updates twice in a row.

### 11.8 `RenderBands`

`GliderPRO/Sources/Render.c:534-555`. Rubber bands in flight.

```
 1  if numBands == 0 -> return
 2  for i in 0 .. numBands-1:
 3      dest = bands[i].dest;  QOffsetRect(&dest, playOriginH, playOriginV)
 4      CopyMask(bandsSrcMap, bandsMaskMap -> workSrcMap,
 5               &bandRects[bands[i].mode], &bandRects[bands[i].mode], &dest)
 6      AddRectToWorkRects(&dest);  AddRectToBackRects(&dest)
```

`bands[i].dest` is stored in *room* coordinates, so this is one of the few renderers that applies
`playOrigin` itself. `bandRects[0..2]` are 16 x 6 at y = 6*i (`StructuresInit.c:231-235`) out of the
16 x 18 `bandsSrcMap` (PICT `kRubberBandsPictID` = 4007). There is no swept `whole` rect: a band
moves further than 16 x 6 per frame in general, so a fast-moving band relies on
`AddRectToBackRects(&dest)` from the *previous* frame having already restored the background before
this frame's composite — which it did, because `CopyRectsQD` processes `back2WorkRects` for frame
N-1 before `RenderFrame` for frame N composites.

### 11.9 `RenderShreds`

`GliderPRO/Sources/Render.c:559-612`. The paper-shredder death animation: the glider is drawn as a
40-pixel-wide strip that first *grows* downward out of the shredder slot, then falls.

```
 1  if numShredded <= 0 -> return
 2  for i in 0 .. numShredded-1:
 3      if shreds[i].frame == 0:                        // phase 1: emerging
 4          bounds.bottom += 1
 5          high = bounds.bottom - bounds.top
 6          if high >= 35: frame = 1                    // 35 == shredSrcRect height
 7          src = shredSrcRect;  src.top = src.bottom - high    // bottom-anchored reveal
 8          dest = bounds;  QOffsetRect(&dest, playOriginH, playOriginV)
 9          CopyMask(shredSrcMap, shredMaskMap -> workSrcMap, &src, &src, &dest)
10          AddRectToBackRects(&dest)
11          dest.top--;  AddRectToWorkRects(&dest)      // one extra row for the growth
12          PlayPrioritySound(kShredSound, kShredPriority)
13      else if shreds[i].frame < 20:                   // phase 2: falling
14          bounds.top += 4;  bounds.bottom += 4
15          dest = bounds;  QOffsetRect(&dest, playOriginH, playOriginV)
16          frame++
17          if frame < 20:
18              CopyMask(shredSrcMap, shredMaskMap -> workSrcMap,
19                       &shredSrcRect, &shredSrcRect, &dest)
20          else:
21              AddSparkle(&shreds[i].bounds)
22              PlayPrioritySound(kFadeOutSound, kFadeOutPriority)
23          AddRectToBackRects(&dest)
24          dest.top -= 4;  AddRectToWorkRects(&dest)   // swept rect: 4 rows of travel
```

- `shredSrcRect` is 40 x 35 (`StructuresInit.c:556`), sheet PICT `kShreddedPictID` = 4010.
- Phase 1 advances 1 pixel per frame for 35 frames, so `frame` stays 0 for 35 frames and only the
  `bounds.bottom` field encodes progress. `PlayPrioritySound(kShredSound, ...)` therefore fires on
  *every one of those 35 frames* — the priority mechanism is what stops it becoming a machine-gun.
- Phase 2 falls 4 px/frame for 19 frames (frame 1..19), i.e. 76 px, then on frame 20 spawns a
  sparkle instead of drawing and stops.
- Both phases enlarge the work rect by exactly the distance travelled (`dest.top--` / `dest.top -= 4`)
  so the vacated pixels get repainted. Both push the un-enlarged `dest` to `back2WorkRects`.

`AddAShreddedGlider(bounds)` (`DynamicMaps.c:730-742`):

```
 1  if numShredded > kMaxShredded -> return        // *** off-by-one, see below
 2  shreds[n].bounds.left   = bounds->left + 4
 3  shreds[n].bounds.right  = shreds[n].bounds.left + 40
 4  shreds[n].bounds.top    = bounds->top + 14
 5  shreds[n].bounds.bottom = shreds[n].bounds.top        // zero height: phase 1 grows it
 6  shreds[n].frame = 0;  numShredded++
```

The guard is `numShredded > kMaxShredded` where `kMaxShredded` = 4 and the array has exactly 4
elements (`StructuresInit2.c:256`). When `numShredded == 4` the test is `4 > 4` = false, so the
function writes `shreds[4]` — a 10-byte heap overrun past the end of the block, and leaves
`numShredded == 5` so `RenderShreds` then reads it back every frame. It should be `>=`. In practice
`RemoveShreds` keeps the count low so this is rarely hit, but a Go port must use `>=` (or it will
panic on the slice write, which is arguably the better outcome).

`RemoveShreds` (`DynamicMaps.c:748-778`) removes the shred with the *largest* `frame` (the one
furthest along) by swap-with-last: find `who` = argmax of `frame`; if it is already the last, just
`numShredded--` and zero its frame; otherwise copy the last element's `bounds` and `frame` over it,
decrement, and zero the vacated frame.

### 11.10 `RenderDynamics` and the seven dynamic-object renderers

`RenderDynamics` (`GliderPRO/Sources/Dynamics3.c:112-154`) walks `dinahs[0 .. numDynamics-1]` and
switches on `dinahs[i].type`. Only **seven** of the eighteen object types that can occupy a `dinahs`
slot have a renderer; the rest fall through `default: break;` and are invisible (they are the
appliance "indicator" objects, whose visuals are drawn by the scoreboard-style helpers in
`ObjectDraw2.c`, plus `kSparkle` and `kMacPlus` which are handled elsewhere).

| `dinahs[i].type` | code | renderer | line |
|---|---|---|---|
| `kToaster` | 0x62 | `RenderToast` | `Dynamics3.c:120-122` |
| `kBalloon` | 0x71 | `RenderBalloon` | `:124-126` |
| `kCopterLf` | 0x72 | `RenderCopter` | `:128-131` |
| `kCopterRt` | 0x73 | `RenderCopter` | `:128-131` |
| `kDartLf` | 0x74 | `RenderDart` | `:133-136` |
| `kDartRt` | 0x75 | `RenderDart` | `:133-136` |
| `kBall` | 0x76 | `RenderBall` | `:138-140` |
| `kDrip` | 0x77 | `RenderDrip` | `:142-144` |
| `kFish` | 0x78 | `RenderFish` | `:146-148` |

All seven live in `GliderPRO/Sources/Dynamics.c` and share one body shape:

```
 1  dest = dinahs[who].dest;  QOffsetRect(&dest, playOriginH, playOriginV)
 2  src  = <frameArray>[dinahs[who].frame]
 3  CopyMask(<sheet>SrcMap, <sheet>MaskMap -> workSrcMap, &src, &src, &dest)
 4  AddRectToBackRects(&dest)
 5  dest = dinahs[who].whole;  QOffsetRect(&dest, playOriginH, playOriginV)
 6  AddRectToWorkRects(&dest)
```

Here `dinahs[].whole` is maintained by the *physics* side (`HandleDynamics`, `Dynamics3.c:31-105`)
as the swept union, so the renderer just uses it. Per-function differences:

| Function | Line | Guarded by `moving`? | Frame array | Sheet / mask | Extra |
|---|---|---|---|---|---|
| `RenderToast` | `Dynamics.c:112-139` | yes (:117) | `breadSrc[frame]` | `toastSrcMap` / `toastMaskMap` | vertical clip, below |
| `RenderBalloon` | `:143-163` | yes (:147) | `balloonSrc[frame]` | `balloonSrcMap` / `balloonMaskMap` | - |
| `RenderCopter` | `:167-187` | yes (:171) | `copterSrc[frame]` | `copterSrcMap` / `copterMaskMap` | - |
| `RenderDart` | `:191-211` | yes (:195) | `dartSrc[frame]` | `dartSrcMap` / `dartMaskMap` | - |
| `RenderBall` | `:215-232` | **no** | `ballSrc[frame]` | `ballSrcMap` / `ballMaskMap` | always draws |
| `RenderDrip` | `:236-253` | **no** | `dripSrc[frame]` | `dripSrcMap` / `dripMaskMap` | always draws |
| `RenderFish` | `:257-285+` | branches | `fishSrc[frame]` | `fishSrcMap` / `fishMaskMap` | `else` branch uses plain `CopyBits(srcCopy)` |

`RenderToast`'s clip is the only genuinely different piece of logic (`Dynamics.c:122-127`):

```
vClip = dinahs[who].dest.bottom - dinahs[who].hVel;
if (vClip > 0) { src.bottom -= vClip;  dest.bottom -= vClip; }
```

`hVel` is being (ab)used to carry the toaster's slot height: while the toast is still emerging, the
bottom of both src and dest is pulled up by `vClip` so only the part above the slot is drawn. Note
this reads `dinahs[who].dest.bottom` — the *unoffset* room-space value — and subtracts a velocity
field, so a Go port must copy the expression literally rather than "fixing" it.

`RenderFish`'s `else` branch (`Dynamics.c:276-285`) draws the *not moving* fish with
`CopyBits(..., srcCopy, nil)` instead of `CopyMask`, i.e. the resting fish is drawn as an opaque
16 x 16 block including its background pixels from the sheet, while the swimming fish is masked.
That is visible behaviour, not an optimisation: a resting fish shows a rectangle of sheet
background. (`fishSrcMap` is 16 x 128 = 8 frames of 16 x 16, `StructuresInit.c:677`.)

Frame-array sizes, from `GliderPRO/Sources/Headers/GliderDefines.h:451-457` and the rect builders
in `StructuresInit.c:623-677`:

| Array | Frames | Constant | Cell | Sheet PICT |
|---|---|---|---|---|
| `breadSrc` | 6 | `kNumBreadPicts` = 6 | 32 x 29 | 4009 (`kToastPictID`) |
| `balloonSrc` | 8 | `kNumBalloonFrames` = 8 | 24 x 30 | 4011 (`kBalloonPictID`) |
| `copterSrc` | 10 | `kNumCopterFrames` = 10 | 32 x 30 | 4012 (`kCopterPictID`) |
| `dartSrc` | 4 | `kNumDartFrames` = 4 | 64 x 19 | 4013 (`kDartPictID`) |
| `ballSrc` | 2 | `kNumBallFrames` = 2 | 32 x 32 | 4014 (`kBallPictID`) |
| `dripSrc` | 6 | `kNumDripFrames` = 6 | 16 x 12 | 4015 (`kDripPictID`) |
| `fishSrc` | 8 | `kNumFishFrames` = 8 | 16 x 16 | 4017 (`kFishPictID`) |

### 11.11 `HandleGrease` — the only renderer that paints, not blits

`GliderPRO/Sources/Grease.c:43-133`. Called *first* in `RenderFrame` (`Render.c:647`), before any
sprite compositing. Grease has four modes:

| Constant | Value | Meaning | Definition |
|---|---|---|---|
| `kGreaseIdle` | 0 | jar upright, not yet knocked | `Grease.c:17` |
| `kGreaseFalling` | 1 | jar tipping over (4 cells) | `Grease.c:18` |
| `kGreaseSpreading` | 2 | black slick extending 2 px/frame | `Grease.c:19` |
| `kGreaseSpiltIdle` | 3 | done | `Grease.c:20` |

`kGreaseIdle` and `kGreaseSpiltIdle` do nothing per frame. The other two:

```
kGreaseFalling                                                Grease.c:53-87
 1  frame++
 2  if frame >= 3:                                    // fully tipped
 3      frame = 3;  mode = kGreaseSpreading
 4      hotSpots[hotNum].action = kSlideIt;  hotSpots[hotNum].isOn = true
 5      src = isRight ? (0,-2,2,0) : (-2,-2,0,0)      // 2x2 seed of slick
 6      QOffsetRect(&src, -playOriginH, -playOriginV) // hotSpots are room-space
 7      QOffsetRect(&src, start, dest.bottom)
 8      hotSpots[hotNum].bounds = src
 9  src = (0,0,32,27) offset (0, frame*27)            // select cell from savedMap
10  CopyBits(savedMaps[mapNum].map -> workSrcMap, &src, &dest, srcCopy, nil)
11  CopyBits(savedMaps[mapNum].map -> backSrcMap, &src, &dest, srcCopy, nil)
12  AddRectToWorkRects(&dest)
13  QOffsetRect(&dest, isRight ? +2 : -2, 0)          // jar leans as it falls

kGreaseSpreading                                              Grease.c:88-131
 1  src = isRight ? (0,-2,2,0) : (-2,-2,0,0)
 2  QOffsetRect(&src, start, dest.bottom)
 3  start += isRight ? +2 : -2
 4  hotSpots[hotNum].bounds.<right|left> += (isRight ? +2 : -2)
 5  save GWorld;  SetGWorld(backSrcMap); PaintRect(&src)
 6                SetGWorld(workSrcMap); PaintRect(&src); AddRectToWorkRects(&src)
 7                restore GWorld
 8  if isRight and start >= stop  -> mode = kGreaseSpiltIdle
 9  if !isRight and start <= stop -> mode = kGreaseSpiltIdle
```

The slick is literally a `PaintRect` of the current pen pattern (black) into **both** buffers, 2 px
per frame — the only place in the renderer where a game element is drawn with a fill rather than a
blit, and the only per-frame writer to `backSrcMap` besides `RestoreFromSavedMap`. Writing to
`backSrcMap` makes the slick permanent so it survives `back2WorkRects` erases.

Two coordinate-space traps: the `hotSpots[].bounds` written at step 6-8 of the falling phase is in
*room* space (hence the `-playOrigin`), whereas the `src`/`dest` rects painted in the spreading
phase are in *work map* space. And `dest.bottom` is used as the slick's y both times.

`RedrawAllGrease` (`Grease.c:270-301`) is the "lights changed / room repainted" fixup: for each
grease with `where == thisRoomNumber`, a 2-pixel-tall `hotSpots[].bounds`, and
`mode != kGreaseIdle`, it offsets that room-space rect by `+playOrigin` and `PaintRect`s it into
both `backSrcMap` and `workSrcMap`, queuing it to the screen. This is how already-spilt grease
reappears after a lighting change wipes the background.

---

## 12. The glider and its shadow

### 12.1 The glider sheets

Four PICTs, all 48 x 668, all loaded into 48 x 668 GWorlds
(`glidSrcRect` = `QSetRect(&glidSrcRect, 0, 0, 48, 668)`, `GliderPRO/Sources/StructuresInit.c:178`):

| PICT ID | Constant | Definition | Content |
|---|---|---|---|
| 3999 | `kGliderPictID` | `GliderDefines.h:568` | player 1, normal |
| 3974 | `kGlider2PictID` | `GliderDefines.h:566` | player 2, normal |
| 3976 | `kGliderFoilPictID` | `GliderDefines.h:567` | player 1, foil-wrapped |
| 3963 | `kGliderFoil2PictID` | `GliderDefines.h:565` | player 2, foil-wrapped |
| 4999 | (`kGliderPictID + 1000`) | `StructuresInit.c:188-189` | the 1-bit mask, shared by all four |

Measured: PICT 3999 is 48 x 668 v2, 18688 bytes; 3974 is 48 x 668 v2, 17604 bytes; 3976 is
48 x 668 v2, 14846 bytes; 3963 is 48 x 668 v2, 15258 bytes; 4999 is 48 x 668 **v1**, 4102 bytes
(`/tmp/wf-render/pictframes.txt`, rows for 3963/3974/3976/3999/4999). All four artwork sheets share
one mask because the silhouettes are identical.

Three GWorlds hold them (`StructuresInit.c:179-189`):

| GWorld | Depth | Holds |
|---|---|---|
| `glidSrcMap` | 8 | player 1's current sheet |
| `glid2SrcMap` | 8 | player 2's sheet, *or* in one-player mode the foil sheet |
| `glidMaskMap` | 1 | PICT 4999 |

The assignment of sheets to GWorlds is mode-dependent and is re-done by `LoadGraphic` at runtime,
which is unusual — the sheet contents change during play:

| Situation | `glidSrcMap` | `glid2SrcMap` | Line |
|---|---|---|---|
| two-player game start | 3999 | 3974 | `GliderPRO/Sources/Play.c:124-127` |
| one-player game start | 3999 | **3976** (foil) | `Play.c:132-135` |
| two-player, foil acquired (`DeckGliderInFoil`) | 3976 | 3963 | `GliderPRO/Sources/Player.c:1146-1152` |
| two-player, foil lost (`RemoveFoilFromGlider`) | 3999 | 3974 | `Player.c:1206-1212` |

So in a one-player game the foil is rendered by *switching source GWorld*, not by reloading:
`RenderGlider` picks `glid2SrcMap` when `(!twoPlayerGame) && showFoil` (`Render.c:507-511`). In a
two-player game `glid2SrcMap` is always player 2, so the foil has to be swapped in by reloading
both sheets. Note `Player.c:1148-1151` uses `SetPort((GrafPtr)glidSrcMap)` rather than
`SetGWorld` — the same "cast a GWorldPtr to GrafPtr" shortcut used in `Play.c:124`.

### 12.2 `gliderSrc[0..30]` — the 31 frame rects

`kNumGliderSrcRects` = 31 (`GliderDefines.h:558`). Built in `StructuresInit.c:191-205`:

```
for (i = 0; i < 21; i++)                                   // :191-195
    gliderSrc[i] = (0,0,48,20) offset (0, 20*i)            // y =   0 .. 419
for (i = 21; i < 29; i++)                                  // :196-200
    gliderSrc[i] = (0,0,48,26) offset (0, 420 + 26*(i-21))  // y = 420 .. 627
gliderSrc[29] = (0,0,48,20) offset (0, 628)                // :202-203
gliderSrc[30] = (0,0,48,20) offset (0, 648)                // :204-205
```

Total consumed height: 420 + 208 + 20 + 20 = 668 = the sheet height exactly. So:

| Index range | Cell | Sheet y | Meaning |
|---|---|---|---|
| 0 | 48 x 20 | 0 | facing right, level |
| 1 | 48 x 20 | 20 | facing right, tipped |
| 2 | 48 x 20 | 40 | facing left, level |
| 3 | 48 x 20 | 60 | facing left, tipped |
| 4..10 | 48 x 20 | 80..200 | facing right, 7 fade/dissolve stages |
| 11..17 | 48 x 20 | 220..340 | facing left, 7 fade/dissolve stages (`kLeftFadeOffset` = 7) |
| 18..20 | 48 x 20 | 360..400 | about-face turn-around, 3 frames (`kFirstAboutFaceFrame` = 18, `kLastAboutFaceFrame` = 20, `GliderDefines.h:560-561`) |
| 21..24 | 48 x **26** | 420..498 | facing right, burning (4 frames) |
| 25..28 | 48 x **26** | 524..602 | facing left, burning (4 frames) |
| 29 | 48 x 20 | 628 | facing right, sliding |
| 30 | 48 x 20 | 648 | facing left, sliding |

`kGliderWide` = 48, `kGliderHigh` = 20, `kGliderBurningHigh` = 26 (`GliderDefines.h:548-551`)
correspond exactly to those cell sizes. `kLeftFadeOffset` = 7 (`GliderDefines.h:563`) is the
constant every left-facing fade path adds.

The frame chosen per mode (all in `GliderPRO/Sources/Player.c` unless noted):

| Mode / function | Right-facing index | Left-facing index | Line |
|---|---|---|---|
| `MoveGliderNormal`, level | 0 | 2 | :192, :170 |
| `MoveGliderNormal`, tipped | 1 | 3 | :187, :165 |
| `MoveGliderNormal`, sliding | 29 | 30 | :179, :157 |
| `MoveGliderBurning` | `21 + frame` (frame 0..3) | `25 + frame` | :216, :211 |
| `FadeGliderIn` | `fadeInSequence[frame]` | `+ 7` | :253, :246 |
| `TransportGliderIn` | `fadeInSequence[frame]` | `+ 7` | :283, :276 |
| `FadeGliderOut` | `fadeInSequence[frame]` | `+ 7` | :307, :300 |
| `MoveGliderUpStairs` | 1 | 2 | :327, :322 |
| `FinishGliderUpStairs` | 2 (both facings) | 2 | :389 |
| `MoveGliderDownStairs` | 0 | 2 | :444, :439 |
| `DeckGliderInFoil` | `frame + 2` | `frame + 2 + 7` | :1163, :1156 |
| `MoveGliderFoilGoing`, frame < 5 | `10 - frame` | `10 - frame + 7` | :1190, :1183 |
| `RemoveFoilFromGlider` | `frame + 2` | `frame + 2 + 7` | :1223, :1216 |
| `MoveGliderFaceLeft` | `frame` counting **down** 20 -> 18 | same rects | :562 |
| `MoveGliderFaceRight` | `frame` counting **up** 18 -> 20 | same rects | :579 |
| `MoveGliderFoilLosing` (`Player.c:1230-1255`) | `10 - frame` | `10 - frame + 7` | :1248, :1241 |
| ducting / mail-in-out (`Modes.c:37-44`, `:62-69`, `:107-114`, `:321-328`) | `fadeInSequence[frame]` | `+ 7` | as cited |
| `Modes.c:419-425` (start burning) | 21 | 25 | :424, :419 |
| `Modes.c:441-453` (start about-face) | `kLastAboutFaceFrame` = 20 turning left / `kFirstAboutFaceFrame` = 18 turning right | - | :442, :452 |
| `Modes.c:593-599`, `:619-625` (foil in ducts) | `10 - frame` | `10 - frame + 7` | as cited |

`fadeInSequence[16]` is a lookup table (`kLastFadeSequence` = 16, `GliderDefines.h:564`;
declared `Player.c:51`, filled in `GliderPRO/Sources/InterfaceInit.c:167-182`):

| i | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `fadeInSequence[i]` | 4 | 5 | 6 | 7 | 5 | 6 | 7 | 8 | 6 | 7 | 8 | 9 | 7 | 8 | 9 | 10 |

i.e. four overlapping 4-step ramps 4-7, 5-8, 6-9, 7-10 — a dissolve that both advances and
"shimmers". `FadeGliderIn` counts frame up and terminates at `frame >= 16`; `FadeGliderOut` counts
down and terminates at `frame < 0`.

The glider mode enum (`GliderDefines.h:571-594`), needed because `RenderGlider` switches on three of
them:

| Constant | Value | Constant | Value |
|---|---|---|---|
| `kGliderNormal` | 0 | `kGliderDuctingIn` | 13 |
| `kGliderFadingIn` | 1 | `kGliderMailInLeft` | 14 |
| `kGliderFadingOut` | 2 | `kGliderMailOutLeft` | 15 |
| `kGliderGoingUp` | 3 | `kGliderMailInRight` | 16 |
| `kGliderComingUp` | 4 | `kGliderMailOutRight` | 17 |
| `kGliderGoingDown` | 5 | `kGliderGoingFoil` | 18 |
| `kGliderComingDown` | 6 | `kGliderLosingFoil` | 19 |
| `kGliderFaceLeft` | 7 | `kGliderShredding` | 20 |
| `kGliderFaceRight` | 8 | `kGliderInLimbo` | 21 |
| `kGliderBurning` | 9 | `kGliderIdle` | 22 |
| `kGliderTransporting` | 10 | `kGliderTransportingIn` | 23 |
| `kGliderDuctingDown` | 11 | `kFaceRight` | TRUE |
| `kGliderDuctingUp` | 12 | `kFaceLeft` | FALSE |

### 12.3 The shadow

`shadowSrcRect` is 48 x 18 (`StructuresInit.c:207`), PICT `kShadowPictID` = 3998
(48 x 18 v1, 167 bytes, mask 4998 identical size and byte count — an interesting pair, see 5.9),
split into two 48 x 9 cells (`kShadowHigh` = 9, `GliderDefines.h:552`):

```
for (i = 0; i < kNumShadowSrcRects(2); i++)                // StructuresInit.c:216-220
    shadowSrc[i] = (0,0,48,9) offset (0, 9*i)
```

`shadowSrc[0]` is the right-facing shadow, `shadowSrc[1]` the left-facing — selected by
`which = (facing == kFaceRight) ? 0 : 1` (`Render.c:460-463`).

The shadow's vertical position is fixed: `kShadowTop` = 306 (`GliderDefines.h:553`), so
`destShadow` is initialised to `(0,0,48,9)` offset by `(dest.left, 306)`
(`GliderPRO/Sources/Play.c:354-355`) — room-space y 306..315, sitting on the floor line of the
512 x 322 room interior. Every horizontal glider move updates `destShadow.left/right` in lockstep
with `dest.left/right` (`Player.c:106-109`, `:123-126`, `:402-405`, `:453-456`) and maintains a
swept `wholeShadow` the same way `whole` is maintained for the body.

The shadow is only drawn when the global `shadowVisible` is true, which is set once per room by
`shadowVisible = IsShadowVisible()` at the end of `DrawLocale`
(`GliderPRO/Sources/RoomGraphics.c:126`).

### 12.4 `RenderGlider`

`GliderPRO/Sources/Render.c:452-530`.

```
 1  if thisGlider->dontDraw -> return
 2  which = (facing == kFaceRight) ? 0 : 1
 3  if shadowVisible:
 4      dest = destShadow;  QOffsetRect(&dest, playOriginH, playOriginV)
 5      if mode == kGliderComingUp(4) or mode == kGliderGoingDown(5):
 6          src = shadowSrc[which];  src.right = src.left + (dest.right - dest.left)
 7          CopyMask(shadowSrcMap, shadowMaskMap -> workSrcMap, &src, &src, &dest)
 8      else if mode == kGliderComingDown(6):
 9          src = shadowSrc[which];  src.left = src.right - (dest.right - dest.left)
10          CopyMask(shadowSrcMap, shadowMaskMap -> workSrcMap, &src, &src, &dest)
11      else:
12          CopyMask(shadowSrcMap, shadowMaskMap -> workSrcMap,
13                   &shadowSrc[which], &shadowSrc[which], &dest)
14      src = wholeShadow;  QOffsetRect(&src, playOriginH, playOriginV)
15      AddRectToWorkRects(&src);  AddRectToBackRects(&dest)
16  dest = thisGlider->dest;  QOffsetRect(&dest, playOriginH, playOriginV)
17  if oneOrTwo:                                       // player 1
18      if (!twoPlayerGame) && showFoil:
19          CopyMask(glid2SrcMap, glidMaskMap -> workSrcMap, &src_, &mask_, &dest)
20      else:
21          CopyMask(glidSrcMap,  glidMaskMap -> workSrcMap, &src_, &mask_, &dest)
22  else:                                              // player 2
23      CopyMask(glid2SrcMap, glidMaskMap -> workSrcMap, &src_, &mask_, &dest)
24  src = thisGlider->whole;  QOffsetRect(&src, playOriginH, playOriginV)
25  AddRectToWorkRects(&src);  AddRectToBackRects(&dest)
```

(`src_` and `mask_` are `thisGlider->src` and `thisGlider->mask`; they are always equal in the
shipped code — every assignment site sets both to the same `gliderSrc[i]` — but `CopyMask` is given
them as separate arguments so a Go port should keep both fields.)

Points to get right:

- Steps 6 and 9 are the **stair clip**: the shadow is squeezed horizontally as the glider walks
  off the edge of a staircase, and the squeeze is anchored to the *left* edge for
  `kGliderComingUp`/`kGliderGoingDown` but to the *right* edge for `kGliderComingDown`. `dest` has
  already been narrowed by the mover functions; the shadow src is narrowed to match.
- The body's own clip is done the same way in the movers: whenever `dest` is narrowed, `src` and
  `mask` are narrowed by the identical amount and from the identical edge — e.g.
  `Player.c:374-376` (`top` clip going upstairs), `:410-412` (`right` clip), `:468-470`,
  `:1132-1135` (`left` clip). A Go port that clips only `dest` will stretch the sprite.
- `whole` (swept) goes to `work2MainRects`; the un-swept `dest` goes to `back2WorkRects`. Same
  rule as `RenderFlyingPoints`.
- `RenderGlider` is called from `RenderFrame` *after* `RenderSparkles` and *before* `RenderShreds`
  and `RenderBands` (`Render.c:656-660`), so the glider is behind flying shreds and rubber bands
  but in front of everything else.

`HideGlider` (`GliderPRO/Sources/Play.c:712-729`) is the inverse: it blits `whole` (offset by
`playOrigin`) from `workSrcMap` straight to the window with `CopyRectWorkToMain`, then — if
`hasMirror` — the same rect offset by `(-20,-16)`, then `wholeShadow`. It is used at game over and
before dialogs to make the glider vanish immediately without waiting for a frame.

### 12.5 Mirrors and reflections

A mirror is object type `kMirror` (0x82). Two things happen when one is drawn
(`GliderPRO/Sources/ObjectDrawAll.c:902-912`):

```
 1  GetObjectRect(&thisObject, &itsRect);  OffsetRectRoomRelative(&itsRect, neighbor)
 2  if SectRect(&itsRect, &testRect, &whoCares) and isLit:  DrawMirror(&itsRect)
 3  if neighbor == kCentralRoom and !redraw:
 4      InsetRect(&itsRect, 4, 4)
 5      AddToMirrorRegion(&itsRect)
```

The 4-pixel inset at step 4 is exactly the mirror's own frame thickness, so the reflection region
is the glass, not the frame.

`DrawMirror` (`GliderPRO/Sources/ObjectDraw2.c:984-1014`) paints the mirror into `backSrcMap` from
concentric rects — there is no PICT for a mirror:

```
 1  grayC = (thisMac.isDepth == 4) ? 13 : k8DkGray2Color
 2  SetGWorld(backSrcMap)
 3  ColorRect(mirror, k8WhiteColor)              // glass
 4  ColorFrameRect(mirror, grayC)                // outer frame
 5  InsetRect(1,1);  ColorFrameRect(k8EarthBlueColor)
 6  InsetRect(1,1);  ColorFrameRect(k8EarthBlueColor)
 7  InsetRect(1,1);  ColorFrameRect(grayC)       // inner frame
```

Four 1-pixel frames, hence the 4-pixel inset. The `isDepth == 4` branch substitutes raw palette
index 13 for `k8DkGray2Color` because the 16-colour CLUT has a different layout.

`AddToMirrorRegion` (`Render.c:740-761`) lazily creates `mirrorRgn` with `RectRgn` on the first
call, and `UnionRgn`s subsequent rects into it; either way it sets `hasMirror = true`. Multiple
mirrors in one room therefore share a single region. `ZeroMirrorRegion` (`:765-771`) disposes it and
clears `hasMirror`; it must be called on every room change or reflections will appear through walls.

`DrawReflection(thisGlider, oneOrTwo)` (`Render.c:136-189`) is the first thing `RenderFrame` does:

```
 1  if dontDraw -> return
 2  which = (facing == kFaceRight) ? 0 : 1      // computed, then never used (dead)
 3  dest = thisGlider->dest;  QOffsetRect(&dest, playOriginH - 20, playOriginV - 16)
 4  wasClip = NewRgn();  if nil -> return
 5  SetPort((GrafPtr)workSrcMap)
 6  GetClip(wasClip);  SetClip(mirrorRgn)
 7  if oneOrTwo:  CopyMask(showFoil ? glid2SrcMap : glidSrcMap, glidMaskMap -> workSrcMap,
 8                          &src, &mask, &dest)
 9  else:         CopyMask(glid2SrcMap, glidMaskMap -> workSrcMap, &src, &mask, &dest)
10  SetClip(wasClip);  DisposeRgn(wasClip)
11  src = thisGlider->whole;  QOffsetRect(&src, playOriginH - 20, playOriginV - 16)
12  AddRectToWorkRects(&src);  AddRectToBackRects(&dest)
```

The "reflection" is not a reflection at all: it is the *same*, un-mirrored glider sprite drawn at a
fixed offset of `(-20, -16)` from the glider, clipped to `mirrorRgn`. There is no horizontal flip
and no perspective. The offset is hard-coded in three places — `Render.c:151`, `Render.c:186` and
`Play.c:722` — so a Go port should hoist it to a named constant.

Two divergences from `RenderGlider` worth noting:

- The foil test here is just `showFoil` (`Render.c:163`), missing the `!twoPlayerGame` guard that
  `RenderGlider` has at `:507`. In a two-player game with foil, `glid2SrcMap` holds
  `kGliderFoil2PictID` (player 2's foil sheet), so **player 1's reflection is drawn with player 2's
  artwork**. This is a real bug; a faithful port reproduces it, a corrected port adds the
  `!twoPlayerGame`.
- `which` is computed at `:145-148` and never referenced — the reflection ignores facing entirely
  and relies on `thisGlider->src` already encoding it.

`DrawReflection` is the only place in the entire renderer that uses QuickDraw clipping regions for
game content (`GetClip`/`SetClip`, `Render.c:158-159`, `:182`). A Go port needs a
region/mask-clipped blit — a per-pixel "is this pixel inside the union of mirror rects" test is
sufficient, since `mirrorRgn` is always a union of axis-aligned rects.

---

## 13. The scoreboard

The scoreboard is the one part of the display that is **not** composited through `workSrcMap`. It is
a 20-pixel-tall strip built in its own offscreen GWorld and blitted **directly to the screen
bitmap**, outside the dirty-rect machinery, outside `RenderFrame`, and (in its default position)
outside `mainWindow` altogether. Everything about it lives in `GliderPRO/Sources/Scoreboard.c`
(457 lines) and `GliderPRO/Sources/StructuresInit.c:59-163` (`InitScoreboardMap`).

### 13.1 Constants

| Constant | Value | Definition | Meaning |
|---|---|---|---|
| `kScoreboardTall` | 20 | `GliderPRO/Headers/GliderDefines.h:515` | strip height, px |
| `kScoreboardHigh` | 0 | `GliderDefines.h:513` | strip sits above the window (9-neighbour mode) |
| `kScoreboardLow` | 1 | `GliderDefines.h:514` | strip sits inside the window, above the room (1/3-neighbour mode) |
| `kScoreboardPictID` | 1997 | `GliderDefines.h:623` | the 1536 x 20 background strip |
| `kBadgePictID` | 1996 | `GliderPRO/Sources/StructuresInit.c:39` | the 32 x 66 badge sheet |
| `kNormalTitleMode` | 0 | `GliderDefines.h:619` | title area shows `thisRoom->name` |
| `kEscapedTitleMode` | 1 | `GliderDefines.h:620` | title area shows `"Hit Delete key if unable to Follow"` |
| `kSavingTitleMode` | 2 | `GliderDefines.h:621` | title area shows `"Saving Game…"` |
| `kGrayBackgroundColor` | 251 | `GliderPRO/Sources/Scoreboard.c:15` | palette index of the strip body, = `#555555` (measured, §5.6) |
| `kGrayBackgroundColor4` | 10 | `Scoreboard.c:16` | same, in the 16-colour CLUT |
| `kFoilBadge` | 0 | `Scoreboard.c:17` | index into the three badge rect arrays |
| `kBandsBadge` | 1 | `Scoreboard.c:18` | " |
| `kBatteryBadge` | 2 | `Scoreboard.c:19` | " |
| `kHeliumBadge` | 3 | `Scoreboard.c:20` | " |
| `kScoreRollAmount` | 13 | `Scoreboard.c:21` | points added to `displayedScore` per frame while rolling |
| `kFoilLow` | 2 | `Scoreboard.c:74` | "25%" flash threshold |
| `kBatteryLow` | 17 | `Scoreboard.c:75` | " |
| `kHeliumLow` | −38 | `Scoreboard.c:76` | " (helium is stored as a negative battery) |
| `kBandsLow` | 2 | `Scoreboard.c:77` | " |
| `kScoreTikSound` | 17 | `GliderDefines.h:72` | sound played on every score-roll step |
| `kScoreTikPriority` | 101 | `GliderDefines.h:121` | its priority |

### 13.2 The five scoreboard GWorlds

All created by `InitScoreboardMap` at `kPreferredDepth` = 8
(`GliderPRO/Headers/Externs.h:15`). None has a mask; none is ever disposed except at quit.

| GWorld | Rect | Size | Created | Holds |
|---|---|---|---|---|
| `boardSrcMap` | `boardSrcRect` = `(0, 0, 20, houseWidth)` | screen-width x 20 | `StructuresInit.c:74` | the assembled strip: PICT 1997 plus the three text fields |
| `badgeSrcMap` | `badgeSrcRect` = `(0, 0, 32, 66)` | 32 x 66 | `:93` | PICT 1996, the four badges and their four blank swatches |
| `boardTSrcMap` | `boardTSrcRect` = `(0, 0, 256, 12)` | 256 x 12 | `:105` | room-title scratch |
| `boardGSrcMap` | `boardGSrcRect` = `(0, 0, 20, 10)` | 20 x 10 | `:114` | glider-count scratch |
| `boardPSrcMap` | `boardPSrcRect` = `(0, 0, 64, 10)` | 64 x 10 | `:123` | score scratch |

(`QSetRect(r, left, top, right, bottom)`, so `QSetRect(&boardTSrcRect, 0, 0, 256, 12)` is
256 wide x 12 tall.) The `// 2144 pixels` comment at `:92` and `// total = 6396 pixels` at `:126`
are the author's memory budget scratchings; 32 x 66 is 2112, not 2144.

Global rect inventory (`Scoreboard.c:29-42`):

| Global | Purpose |
|---|---|
| `boardSrcRect` | source rect of the whole strip in `boardSrcMap` |
| `boardDestRect` | destination of the whole strip, in **`mainWindow` local coordinates** |
| `badgeSrcRect` | bounds of `badgeSrcMap` |
| `boardTSrcRect` / `boardTDestRect` | title scratch bounds / its place inside `boardSrcMap` |
| `boardGSrcRect` / `boardGDestRect` | glider-count scratch / its place inside `boardSrcMap` |
| `boardPSrcRect` / `boardPDestRect` | score scratch / its place inside `boardSrcMap` |
| `boardGQDestRect` / `boardPQDestRect` | the same two fields' places **on screen**, for the "quick" partial refresh path |
| `badgesBlankRects[4]` | the four 16-wide blank swatches, at x 0..16 of `badgeSrcMap` |
| `badgesBadgesRects[4]` | the four 16-wide badge images, at x 16..32 of `badgeSrcMap` |
| `badgesDestRects[4]` | the four badges' places **on screen** |
| `displayedScore` (`long`) | the rolling score actually shown |
| `wasScoreboardMode` (`short`) | `kScoreboardHigh` / `kScoreboardLow` |
| `doRollScore` (`Boolean`) | if false the score snaps instead of rolling |

### 13.3 `InitScoreboardMap` — pseudocode

`GliderPRO/Sources/StructuresInit.c:59-163`.

```
 1  GetGWorld(&wasCPort, &wasWorld)
 2  wasScoreboardMode = kScoreboardHigh                                        // :70
 3  boardSrcRect = houseRect;  ZeroRectCorner(&boardSrcRect)
 4  boardSrcRect.bottom = kScoreboardTall                                      // (0,0,20,W)
 5  CreateOffScreenGWorld(&boardSrcMap, &boardSrcRect, kPreferredDepth)        // theErr ignored
 6  SetGWorld(boardSrcMap, nil)
 7  if boardSrcRect.right >= 640:  hOffset = (RectWide(&boardSrcRect) - kMaxViewWidth) / 2
 8  else:                          hOffset = -576                              // :77-80
 9  thePicture = GetPicture(kScoreboardPictID)                                  // 1997
10  if nil -> RedAlert(kErrFailedGraphicLoad)
11  bounds = (*thePicture)->picFrame                                            // (0,0,20,1536)
12  QOffsetRect(&bounds, -bounds.left, -bounds.top)                             // normalise
13  QOffsetRect(&bounds, hOffset, 0)
14  DrawPicture(thePicture, &bounds);  ReleaseResource(thePicture)
15  QSetRect(&badgeSrcRect, 0, 0, 32, 66)
16  CreateOffScreenGWorld(&badgeSrcMap, ...);  SetGWorld(badgeSrcMap, nil);  LoadGraphic(1996)
17  boardDestRect = boardSrcRect;  QOffsetRect(&boardDestRect, 0, -kScoreboardTall)
18  hOffset = (RectWide(&houseRect) - 640) / 2;   if hOffset < 0: hOffset = -128    // :100-102
19  QSetRect(&boardTSrcRect,0,0,256,12); create boardTSrcMap; SetGWorld
20  boardTDestRect = boardTSrcRect;  QOffsetRect(&boardTDestRect, 137 + hOffset, 5)
21  TextFont(applFont); TextSize(12); TextFace(bold)                            // :109-111
22  QSetRect(&boardGSrcRect,0,0,20,10); create boardGSrcMap; SetGWorld
23  boardGDestRect = boardGSrcRect;  QOffsetRect(&boardGDestRect, 526 + hOffset, 5)
24  TextFont(applFont); TextSize(12); TextFace(bold)                            // :118-120
25  QSetRect(&boardPSrcRect,0,0,64,10); create boardPSrcMap; SetGWorld
26  boardPDestRect = boardPSrcRect;  QOffsetRect(&boardPDestRect, 570 + hOffset, 5)
27  boardPQDestRect = boardPDestRect;  QOffsetRect(&boardPQDestRect, 0, -kScoreboardTall)
28  boardGQDestRect = boardGDestRect;  QOffsetRect(&boardGQDestRect, 0, -kScoreboardTall)
29  TextFont(applFont); TextSize(12); TextFace(bold)                            // :131-133
30  badgesBlankRects[0]  = (0,0,16,16)  offset (0,  0)      // foil
31  badgesBlankRects[1]  = (0,0,16,16)  offset (0, 16)      // rubber bands
32  badgesBlankRects[2]  = (0,0,16,17)  offset (0, 32)      // battery
33  badgesBlankRects[3]  = (0,0,16,17)  offset (0, 49)      // helium
34  badgesBadgesRects[i] = same sizes,  offset (16, same v)
35  badgesDestRects[0]   = (0,0,16,16)  offset (432 + hOffset, 2 - 20)
36  badgesDestRects[1]   = (0,0,16,16)  offset (449 + hOffset, 2 - 20)
37  badgesDestRects[2]   = (0,0,16,17)  offset (467 + hOffset, 1 - 20)
38  badgesDestRects[3]   = (0,0,16,17)  offset (467 + hOffset, 1 - 20)   // same slot as [2]
39  SetGWorld(wasCPort, wasWorld)
```

Three things to notice:

* **`TextFont`/`TextSize`/`TextFace` are set three times, once per scratch GWorld** (`:109-111`,
  `:118-120`, `:131-133`), because in classic QuickDraw the text state is per-`GrafPort`, and each
  `CreateOffScreenGWorld`+`SetGWorld` pair introduces a fresh port whose default is 12-pt system
  font, plain. A Go port needs one shared font, but must remember that the three fields all render
  in `applFont` (the *application* font, Geneva on a US system) at 12 pt **bold**.
* `badgesDestRects[2]` and `badgesDestRects[3]` are **identical** — battery and helium share one
  on-screen slot, because a house never grants both at once (`batteryTotal > 0` means battery,
  `< 0` means helium; `Scoreboard.c:340-355`).
* `boardDestRect` is offset by `-kScoreboardTall`, i.e. it lands at window-local y −20..0 — *above*
  the window's own content area. See 13.9.

### 13.4 Computed geometry for real screen sizes

All values are `(top, left, bottom, right)` in `mainWindow` **local** coordinates, computed from the
code above (verified with python3 against the formulae, screen origin assumed `(0,0)`):

| Quantity | 512 x 342 | 640 x 480 | 832 x 624 | 1024 x 768 | 1152 x 870 |
|---|---|---|---|---|---|
| `houseRect` | (0,0,322,512) | (0,0,460,640) | (0,0,604,832) | (0,0,748,1024) | (0,0,850,1152) |
| `boardSrcRect` | (0,0,20,512) | (0,0,20,640) | (0,0,20,832) | (0,0,20,1024) | (0,0,20,1152) |
| `hOffset` for PICT 1997 | **−576** | −448 | −352 | −256 | −192 |
| `hOffset` for the fields | **−128** | 0 | +96 | +192 | +256 |
| `boardDestRect` (High) | (−20,0,0,512) | (−20,0,0,640) | (−20,0,0,832) | (−20,0,0,1024) | (−20,0,0,1152) |
| `boardDestRect` (Low) | (−10,0,10,512) | (59,0,79,640) | (131,0,151,832) | (203,0,223,1024) | (254,0,274,1152) |
| `boardTDestRect` (in `boardSrcMap`) | (5,9,17,265) | (5,137,17,393) | (5,233,17,489) | (5,329,17,585) | (5,393,17,649) |
| `boardGDestRect` | (5,398,15,418) | (5,526,15,546) | (5,622,15,642) | (5,718,15,738) | (5,782,15,802) |
| `boardPDestRect` | (5,442,15,506) | (5,570,15,634) | (5,666,15,730) | (5,762,15,826) | (5,826,15,890) |
| `boardGQDestRect` | (−15,398,−5,418) | (−15,526,−5,546) | (−15,622,−5,642) | (−15,718,−5,738) | (−15,782,−5,802) |
| `boardPQDestRect` | (−15,442,−5,506) | (−15,570,−5,634) | (−15,666,−5,730) | (−15,762,−5,826) | (−15,826,−5,890) |
| `badgesDestRects[0]` foil | (−18,304,−2,320) | (−18,432,−2,448) | (−18,528,−2,544) | (−18,624,−2,640) | (−18,688,−2,704) |
| `badgesDestRects[1]` bands | (−18,321,−2,337) | (−18,449,−2,465) | (−18,545,−2,561) | (−18,641,−2,657) | (−18,705,−2,721) |
| `badgesDestRects[2..3]` batt/He | (−19,339,−2,355) | (−19,467,−2,483) | (−19,563,−2,579) | (−19,659,−2,675) | (−19,723,−2,739) |

The two `hOffset` computations differ by **exactly −448 in every case**, including the `< 640`
special case (−576 − (−128) = −448). That is the key to the whole layout: the 1536-wide strip
artwork is drawn so that **PICT x 448 lines up with field-layout x 0**, i.e.

```
pictX = fieldX + 448
```

and the "reference layout" — the one the hard-coded field offsets 137 / 526 / 570 / 432 / 449 / 467
were authored against — occupies exactly PICT x 448..1087, a 640-wide window (§13.5 confirms this
pixel-for-pixel). Wider screens reveal more of the strip symmetrically; the visible PICT range for a
screen of width `W` (with `W >= 640`) is

```
pictX in [ (1536 - W)/2 , (1536 + W)/2 - 1 ]
```

so W = 640 shows 448..1087, W = 832 shows 352..1183, W = 1152 shows 192..1343 and W = 1536 shows the
whole thing. For `W < 640` the special-case `-576` shifts everything a further 128 px left, which
crops the leftmost 128 px of the reference layout and moves the fields left by the same 128 px, so
the strip and the fields stay registered with each other.

### 13.5 Measured PICT 1997 — the strip artwork

Parsed from `GliderPRO/Glider PRO.r` with a from-scratch PICT decoder: **PICT 1997 is
`picFrame` (0,0,20,1536) = 1536 x 20, `picSize` 13296 = actual byte count, version 2, one
`PackBitsRect`-class image at `pixelSize` 8** (`/tmp/wf-render/pictframes.txt` row 1997).

Only 20 palette indices appear in the whole strip. Full histogram (index -> count, RGB from the
measured `clut 128`, §5.6):

| Index | RGB | Count | Role |
|---|---|---|---|
| 251 | `#555555` | 15972 | strip body — this is `kGrayBackgroundColor` |
| 172 | `#333333` | 3872 | lower bevel line, rule shading |
| 42 | `#CCCCFF` | 3462 | all lettering and ornament highlight |
| 254 | `#111111` | 1818 | rule cores, second-to-last bevel row |
| 248 | `#AAAAAA` | 1664 | top bevel highlight row |
| 250 | `#777777` | 1632 | second bevel row |
| 255 | `#000000` | 1536 | bottom row (all 1536 px) |
| 253 | `#222222` | 229 | shading |
| 11 | `#FFCC00` | 170 | icon colour |
| 0 | `#FFFFFF` | 78 | icon highlight |
| 2 | `#FFFF99` | 60 | icon colour |
| 4 | `#FFFF33` | 56 | icon colour |
| 17, 23, 43, 51, 78, 137, 249, 252 | `#FF9900`, `#FF6600`, `#CCCCCC`, `#CC9966`, `#99CCFF`, `#663300`, `#888888`, `#444444` | few | icon colours / anti-aliasing |

The **modal column** — the plain background, which occurs at 536 of the 1536 x-positions — is, top
to bottom:

| y | 0 | 1 | 2..16 | 17 | 18 | 19 |
|---|---|---|---|---|---|---|
| index | 248 | 250 | 251 (15 rows) | 172 | 254 | 255 |
| RGB | `#AAAAAA` | `#777777` | `#555555` | `#333333` | `#111111` | `#000000` |

i.e. a top-lit horizontal bevel: one light row, one mid row, a 15-row body at
`kGrayBackgroundColor`, then three progressively darker rows down to pure black at y 19. A Go port
can reproduce the plain background procedurally from those six values without decoding the PICT at
all.

Columns that differ from the modal column, as run-lengths (`pict x`, and the same range in
reference-layout / 640-window coordinates, `winX = pictX − 448`):

| PICT x | Window x (640) | Width | Content |
|---|---|---|---|
| 0..1 | −448..−447 | 2 | left edge bevel |
| 3..188 | −445..−260 | 186 | ornament panel |
| 190..193 | −258..−255 | 4 | vertical rule |
| 195..348 | −253..−100 | 154 | ornament panel |
| 350..353 | −98..−95 | 4 | vertical rule |
| 355..444 | −93..−4 | 90 | ornament panel |
| 446..449 | **−2..1** | 4 | vertical rule — the leftmost 2 px visible in a 640 window |
| 482..497 | 34..49 | 16 | letters `Gl` of the wordmark |
| 499..503 | 51..55 | 5 | letter `i` |
| 505..533 | 57..85 | 29 | letters `der` |
| 536..542 | 88..94 | 7 | 7 x 12 stacked-glyph logo block (the "PRO" mark) |
| 574..577 | 126..129 | 4 | vertical rule, immediately left of the title field at x 137 |
| 939..972 | 491..524 | 34 | 34 x 14 icon, immediately left of the glider-count field at x 526 |
| 1000..1016 | 552..568 | 17 | 17 x 16 icon, immediately left of the score field at x 570 |
| 1086..1089 | 638..641 | 4 | vertical rule — the rightmost 2 px visible in a 640 window |
| 1091..1180 | 643..732 | 90 | ornament panel (mirror of 355..444) |
| 1182..1185 | 734..737 | 4 | vertical rule |
| 1187..1340 | 739..892 | 154 | ornament panel |
| 1342..1345 | 894..897 | 4 | vertical rule |
| 1347..1532 | 899..1084 | 186 | ornament panel |
| 1534..1535 | 1086..1087 | 2 | right edge bevel |

The run structure is exactly symmetric about x = 767.5, but the pixels are not: 1074 of the 1536
columns differ from their mirror image, so the ornament was hand-touched. The ornament repeats with
a period of **18 px** (106 of 120 sampled columns match their neighbour 18 to the right).

The 4-px vertical rules are, column by column (values at PICT x 446..449, i.e. window x −2..1):

```
x=446 : 248, 251, 172 x16,      254, 255      (dark left shoulder)
x=447 : 251, 254 x18,           255           (near-black core)
x=448 : 248 x18,                251, 255      (light right shoulder)
x=449 : 248, 250 x16, 251,      254, 255      (fades back into the body)
```

so the rule is a 1-px dark / 1-px black / 1-px light incised groove.

The wordmark rendered as ASCII (`.` = background, `o` = index 42 `#CCCCFF`, `#` = a dark index),
PICT x 482..542, rows 2..17:

```
  ..oooooo#..oooo#  .....  .............................  .......
  .oooooooo#..oooo  .ooo#  ......ooo#...................  .oooo#.
  ooooo##ooo..oooo  .#ooo  ......oooo...................  o#oooo#
  oooo#.......oooo  .....  ......oooo...................  #oooo#o
  oooo........oooo  oooo#  ..oooooooo..ooooo#..ooooooo#.  .#oooo.
  oooooooooo#.oooo  #oooo  .ooooooooo.ooooooo#.oooooooo#  .oo.oo#
  oooo#oooooo.oooo  .oooo  ooooo#ooooooooo#ooo#ooooo#ooo  o#oo#oo
  oooo...oooo.oooo  .oooo  oooo#.oooooooooooooooooo.....  oooooo#
  oooo...oooo.oooo  .oooo  oooo..oooooooooooooooooo.....  #oooooo
  #ooo..ooooo.oooo  .oooo  oooo..oooooooo......oooo.....  .oo#...
  .#oooooooo#.oooo  .oooo  #ooo#ooooo#ooo#ooo#.oooo.....  o#oo#..
  ..#oooooo#..#ooo  .#ooo  .#oooooooo.#ooooooo.oooo.....  oooooo#
   G        l         i     d      e      r                 PRO
```

The badge zone (window x 432..483, PICT x 880..931) is **pure modal background** — no artwork at
all. That is direct empirical confirmation that the four badges are composited at runtime from
PICT 1996, not baked into the strip.

### 13.6 Measured PICT 1996 — the badge sheet

**PICT 1996 is `picFrame` (0,0,66,32) = 32 wide x 66 tall, `picSize` 3144 = actual, version 2**
(`/tmp/wf-render/pictframes.txt` row 1996). Measured layout, matching `badgesBlankRects` (x 0..15)
and `badgesBadgesRects` (x 16..31) exactly:

| Rows | Height | x 0..15 | x 16..31 | Badge index |
|---|---|---|---|---|
| 0..15 | 16 | blank (strip background) | aluminium-foil badge | `kFoilBadge` = 0 |
| 16..31 | 16 | blank | rubber-band badge | `kBandsBadge` = 1 |
| 32..48 | 17 | blank | battery badge | `kBatteryBadge` = 2 |
| 49..65 | 17 | blank | helium badge | `kHeliumBadge` = 3 |

The "blank" half is not white — it is a 16 x 66 swatch of the strip's own bevel/background, so
`CopyBits(badgeSrcMap, screen, badgesBlankRects[i], badgesDestRects[i], srcCopy)` erases a badge by
stamping matching background over it. This is the same pre-composition trick as `savedMaps`
(§11.2), applied to the scoreboard: no mask, no transparency, one opaque `srcCopy` either way.

### 13.7 The text idiom

All three text fields use one idiom (`Scoreboard.c:140-188`, `:198-227`, `:236-264`):

```
1  SetPort((GrafPtr)<scratch GWorld>)                     // NOT SetGWorld; see 13.9
2  GetForeColor(&wasColor)
3  Index2Color(thisMac.isDepth == 4 ? kGrayBackgroundColor4(10) : kGrayBackgroundColor(251), &rgb)
4  RGBForeColor(&rgb);  PaintRect(&<scratch rect>);  RGBForeColor(&wasColor)
5  MoveTo(1, 10);  ForeColor(blackColor);  DrawString(s)      // shadow, 1 px right and 1 px down
6  MoveTo(0,  9);  ForeColor(whiteColor);  DrawString(s)      // highlight
7  ForeColor(blackColor)
8  CopyBits(<scratch> -> <dest>, <scratchRect>, <destRect>, srcCopy, nil)
```

So every scoreboard string is drawn **twice**: black at baseline (1, 10), then white at baseline
(0, 9). The white copy is on top, so the visible glyphs are white with a 1-px black drop shadow down
and to the right. Field heights are 12 (title) and 10 (numbers) with baselines at y 9/10, meaning
descenders are clipped — acceptable for room names and digits.

Note step 3: the background is *not* painted with `PenPat`/`ForeColor(index)`; it round-trips the
palette index through `Index2Color` -> `RGBForeColor`, which asks the current GDevice's CLUT for
index 251's RGB and then lets QuickDraw re-match it. On an 8-bit device with `clut 128` loaded that
returns exactly `#555555` and re-matches to index 251. A Go port should just write index 251 (or
`#555555`) directly.

The three strings:

| Mode | String | Line |
|---|---|---|
| `kEscapedTitleMode` (1) | `"Hit Delete key if unable to Follow"` | `Scoreboard.c:156`, `:172` |
| `kSavingTitleMode` (2) | `"Saving Game…"` (Mac Roman 0xC9 ellipsis) | `:160`, `:176` |
| default / `kNormalTitleMode` (0) | `thisRoom->name` (a Pascal string in the room record) | `:164`, `:180` |

The 256-px title field at 12-pt bold Geneva holds roughly 42 characters; the escaped-mode string is
34 characters, so it fits. Long room names are silently clipped by the scratch GWorld's bounds.

`RefreshNumGliders` clamps a negative glider count to 0 (`Scoreboard.c:209-211`); the "quick" variant
`QuickGlidersRefresh` does **not** (`:284`), so `mortals` going negative briefly shows `-1`.
`NumToString` is used for both numbers (`:212`, `:247`, `:284`, `:319`).

### 13.8 The refresh entry points

| Function | Lines | Writes to | What it does |
|---|---|---|---|
| `RefreshRoomTitle(mode)` | `Scoreboard.c:136-188` | `boardSrcMap` | render title into `boardTSrcMap`, blit to `boardTDestRect` |
| `RefreshNumGliders()` | `:192-227` | `boardSrcMap` | render `mortals` (clamped >= 0), blit to `boardGDestRect` |
| `RefreshPoints()` | `:231-264` | `boardSrcMap` | render `theScore`, blit to `boardPDestRect`, set `displayedScore = theScore` |
| `RefreshScoreboard(mode)` | `:53-68` | **screen** | `doRollScore = true`; the three above; blit whole strip to `boardDestRect`; then `QuickBatteryRefresh(false)`, `QuickBandsRefresh(false)`, `QuickFoilRefresh(false)` |
| `QuickGlidersRefresh()` | `:268-299` | **screen** | re-render `mortals` (unclamped) and blit straight to `boardGQDestRect` |
| `QuickScoreRefresh()` | `:303-334` | **screen** | re-render `displayedScore` and blit straight to `boardPQDestRect` |
| `QuickBatteryRefresh(flash)` | `:338-364` | **screen** | battery badge if `batteryTotal > 0`, helium badge if `< 0`, blank if `flash` or 0 |
| `QuickBandsRefresh(flash)` | `:368-386` | **screen** | bands badge if `bandsTotal > 0 && !flash`, else blank |
| `QuickFoilRefresh(flash)` | `:390-408` | **screen** | foil badge if `foilTotal > 0 && !flash`, else blank |
| `HandleDynamicScoreboard()` | `:72-132` | **screen** | per-frame score roll + low-supply badge flashing |
| `AdjustScoreboardHeight()` | `:412-449` | (rects only) | move the strip between the High and Low positions |
| `BlackenScoreboard()` | `:453-456` | **screen** | calls `UpdateMenuBarWindow()` |

The `Refresh*` (slow) path rebuilds `boardSrcMap` and re-blits the *entire* strip; the `Quick*` path
touches only the 20 x 10, 64 x 10 or 16 x 17 rectangle that changed and never touches `boardSrcMap`.
Both end in a `CopyBits` whose destination is
`GetPortBitMapForCopyBits(GetWindowPort(mainWindow))`.

Callers of the slow path — 25 sites, all outside `Scoreboard.c`:

| Call | Sites |
|---|---|
| `RefreshScoreboard(kNormalTitleMode)` | `GliderPRO/Sources/Play.c:164` (game start), `:404` (update event), `:510` (game over), `:815` (`UpdateWholeScreen`), `GliderPRO/Sources/Input.c:71`, `GliderPRO/Sources/Transit.c:299`, `:336`, `:375`, `:414` (every room transition) |
| `RefreshScoreboard(kEscapedTitleMode)` | `Interactions.c:182`, `:216`, `:294`, `:328`, `:522`, `:553`, `:612`, `:643`, `:1267`, `:1302`; `Player.c:364`, `:492`, `:628`, `:735`, `:832`, `:1031`, `:1122` |
| `RefreshScoreboard(kSavingTitleMode)` | `Input.c:67` |
| `QuickGlidersRefresh()` | `Interactions.c:844`, `Player.c:1519` |
| `QuickBatteryRefresh(false)` | `Input.c:142`, `:167`, `Interactions.c:866`, `:972`, `:1182` |
| `QuickBandsRefresh(false)` | `Input.c:258`, `:357`, `Interactions.c:885`, `:1176` |
| `QuickFoilRefresh(false)` | `Modes.c:586`, `:611` |
| `AdjustScoreboardHeight()` | `Play.c:81` only (`NewGame`) |
| `BlackenScoreboard()` | `Play.c:255` only |
| `HandleDynamicScoreboard()` | `Play.c:470`, `:495` (once per game frame, immediately after `RenderFrame`) |

So a room transition costs a full strip rebuild + full-width blit, while picking up a battery costs
one 16 x 17 blit.

### 13.9 `HandleDynamicScoreboard` — the score roll and the badge blink

`GliderPRO/Sources/Scoreboard.c:72-132`, called once per game frame from `PlayGame` right after
`RenderFrame()` (`Play.c:470`, `:495`).

```
 1  if theScore > displayedScore:
 2      if doRollScore:  displayedScore += kScoreRollAmount(13)
 3                       if displayedScore > theScore: displayedScore = theScore
 4      else:            displayedScore = theScore
 5      PlayPrioritySound(kScoreTikSound(17), kScoreTikPriority(101))
 6      QuickScoreRefresh()
 7  whosTurn = gameFrame & 0x00000007                       // 0..7, period 8 frames
 8  switch whosTurn:
 9    case 0: if 0 < foilTotal    < kFoilLow(2)      : QuickFoilRefresh(false)     // show
10    case 1: if 0 < batteryTotal < kBatteryLow(17)  : QuickBatteryRefresh(true)   // hide
11            elif kHeliumLow(-38) < batteryTotal < 0: QuickBatteryRefresh(true)
12    case 2: if 0 < bandsTotal   < kBandsLow(2)     : QuickBandsRefresh(false)    // show
13    case 4: if 0 < batteryTotal < kBatteryLow(17)  : QuickBatteryRefresh(false)  // show
14            elif kHeliumLow(-38) < batteryTotal < 0: QuickBatteryRefresh(false)
15    case 6: (no case 6 — falls through to no-op)
16    case 5: if 0 < foilTotal    < kFoilLow(2)      : QuickFoilRefresh(true)      // hide
17    case 7: if 0 < bandsTotal   < kBandsLow(2)     : QuickBandsRefresh(true)     // hide
```

Details a port must match:

* **`kScoreRollAmount` = 13 points per frame**, and one `kScoreTikSound` per frame while rolling —
  so awarding 100 points produces 8 tick sounds over 8 frames (~0.27 s at 30 fps). `doRollScore` is
  forced `true` by every `RefreshScoreboard` (`:55`) and is never set false anywhere in the shipped
  source, so the `else` branch at `:88-89` is dead.
* The score display *only ever counts up*: if `theScore < displayedScore` (a penalty) nothing
  happens until the next full `RefreshScoreboard`.
* The blink is driven by `gameFrame & 7`, not by a timer, so its period is 8 game frames
  = 16 ticks ≈ 0.266 s at `kTicksPerFrame` = 2. Foil shows at phase 0 and hides at phase 5 (5 frames
  on, 3 off); bands show at 2 and hide at 7 (5 on, 3 off); battery **hides at 1 and shows at 4**
  (3 frames blank, 5 frames shown). Phases 3 and 6 have no case.
* The thresholds are exclusive on both ends, so a badge with exactly 0 or exactly the threshold value
  does not blink. Helium is stored as negative `batteryTotal`, hence the inverted comparison
  `batteryTotal > kHeliumLow(-38)`.

### 13.10 The two positions, `AdjustScoreboardHeight`, and `justRoomsRect`

`GliderPRO/Sources/Scoreboard.c:412-449`:

```
 1  newMode = (numNeighbors == 9) ? kScoreboardHigh(0) : kScoreboardLow(1)
 2  if wasScoreboardMode == newMode: return              // nothing to do
 3  switch newMode:
 4    case kScoreboardHigh:
 5      offset = -localRoomsDest[kCentralRoom].top
 6      justRoomsRect = workSrcRect                                    // the whole surface
 7      break
 8    case kScoreboardLow:
 9      offset = +localRoomsDest[kCentralRoom].top
10      justRoomsRect = workSrcRect
11      justRoomsRect.top    = localRoomsDest[kCentralRoom].top
12      justRoomsRect.bottom = localRoomsDest[kCentralRoom].bottom     // just the room band
13      break
14  QOffsetRect(&boardDestRect,     0, offset)
15  QOffsetRect(&boardGQDestRect,   0, offset)
16  QOffsetRect(&boardPQDestRect,   0, offset)
17  QOffsetRect(&badgesDestRects[kBatteryBadge], 0, offset)
18  QOffsetRect(&badgesDestRects[kBandsBadge],   0, offset)
19  QOffsetRect(&badgesDestRects[kFoilBadge],    0, offset)
20  QOffsetRect(&badgesDestRects[kHeliumBadge],  0, offset)
21  wasScoreboardMode = newMode
```

`wasScoreboardMode` is initialised to `kScoreboardHigh` (`StructuresInit.c:70`) and the only caller
is `NewGame` (`Play.c:81`), so:

| `numNeighbors` | Mode | `boardDestRect` (640 x 480) | Where that is on screen | `justRoomsRect` |
|---|---|---|---|---|
| 9 (default) | High | (−20, 0, 0, 640) | global y 0..20 — the menu-bar strip, **above** `mainWindow` | whole surface (0,0,460,640) |
| 3 | Low | (59, 0, 79, 640) | inside the window, in the 79-px gap above the central room | (79, 0, 401, 640) |
| 1 | Low | (59, 0, 79, 640) | same | (79, 0, 401, 640) |

`numNeighbors` defaults to **9** (`GliderPRO/Sources/Main.c:154`, and `Settings.c:888`), is restored
from prefs at `Main.c:111`, and is forced to 1 when `thisMac.screen.right <= 512`
(`Main.c:191-192`).

`justRoomsRect` is the whole point of the Low mode: it is the clamp rectangle used by
`AddRectToWorkRects` (`GliderPRO/Sources/Render.c:70-77`, §9.2). Narrowing it to the central room's
vertical band guarantees that no work->screen dirty rect can ever overlap the scoreboard strip that
now sits *inside* the window, so the renderer cannot erase the score. It is initialised to the full
`houseRect` (`GliderPRO/Sources/StructuresInit2.c:153-154`) and is also used as the transition
rectangle (`Transit.c:300`, `:338`, `:377`, `:416`; `Play.c:172`, `:177`, `:181`), the update-event
blit rectangle (`Play.c:403`) and the game-over region (`GameOver.c:307`).

Note the asymmetry: `AdjustScoreboardHeight` sets `justRoomsRect` in **both** branches even though
only the Low branch narrows it — the High branch resets it to `workSrcRect`, which is needed if the
player switches from 1/3 neighbours back to 9 between games.

### 13.11 Why the "High" position works: the leaked `GrafPort`

`boardDestRect` in High mode is `(-20, 0, 0, 640)` — entirely above `mainWindow`'s content area,
which starts at window-local y 0 (`MainWindow.c:222-232` sizes the window to
`thisMac.screen` minus 20 and moves it to `thisMac.screen.top + 20`). One would expect
`ClipRect(&mainWindowRect)` at `MainWindow.c:232` to throw the blit away. It does not, and the
reason is a QuickDraw subtlety that a Go port has to decide about explicitly:

1. Classic QuickDraw's `CopyBits` clips the destination to the **destination bitmap's `bounds`**,
   plus the current `GrafPort`'s `visRgn`/`clipRgn` **only when the destination bitmap is the current
   port's own bitmap**.
2. For an on-screen window, `GetPortBitMapForCopyBits(GetWindowPort(mainWindow))` hands back the
   *screen* bitmap with `bounds` expressed in window-local coordinates — `(-20, 0, 460, 640)` for a
   640 x 480 main monitor. `(-20, 0, 0, 640)` is entirely inside that.
3. The current port at the moment of the blit is **not** `mainWindow`. `RefreshScoreboard` calls
   `RefreshRoomTitle` -> `RefreshNumGliders` -> `RefreshPoints`, each of which does
   `SetPort((GrafPtr)<scratch GWorld>)` and **never restores it** (`Scoreboard.c:140`, `:198`,
   `:236`). By the time control returns to `Scoreboard.c:61` the current port is `boardPSrcMap`, a
   64 x 10 offscreen GWorld. Its clip is irrelevant because the destination is a different bitmap.

So the strip is written straight into screen memory in the 20 rows above the window — exactly where
`menuWindow` lives (a borderless `RectWide(&thisMac.screen)` x 20 window pinned at
`thisMac.screen.left, thisMac.screen.top`, `MainWindow.c:214-221`). The scoreboard is literally
painted on top of another window's content, which is why the whole thing has to be re-blitted on
every update event.

Two independent pieces of evidence that this is the real clipping rule in this codebase, not a
guess:

* `UpdateMainWindow` (`MainWindow.c:131-134`, `:144-147`) passes
  `GetPortVisibleRegion(GetWindowPort(mainWindow), dummyRgn)` as the **`maskRgn` argument** of
  `CopyBits`. If the current port's `visRgn` were applied automatically there would be no reason to
  pass it by hand — and at `:144` the current port is `workSrcMap` (set at `:139`), so it could not
  have been applied.
* `DrawReflection` (`Render.c:157-159`) explicitly does `SetPort((GrafPtr)workSrcMap)` *before*
  `SetClip(mirrorRgn)` precisely so that the clip belongs to the destination surface.

Under this rule the whole scoreboard design is coherent: `RefreshRoomTitle`'s blit into
`boardSrcMap` at `boardTDestRect` = (5, 137, 17, 393) is also unclipped even though the current port
is the 256 x 12 `boardTSrcMap`, and the `Quick*` blits land at negative y on the screen.

The one place this bites is `Play.c:395-407`, the `updateEvt` handler: the `CopyBits` at `:401` runs
while the current port *is* `mainWindow` and is inside `BeginUpdate`/`EndUpdate`, so it is correctly
clipped to the update region — but the `RefreshScoreboard` at `:404` is not clipped at all and
repaints the entire strip regardless of what was invalidated. Harmless, and it is what makes the
scoreboard survive being overwritten by other windows.

### 13.12 `BlackenScoreboard`, `UpdateMenuBarWindow`, and the second-monitor kludge

```
void BlackenScoreboard (void)  { UpdateMenuBarWindow(); }      // Scoreboard.c:453-456

void UpdateMenuBarWindow (void)                                // MainWindow.c:160-169
{
    Rect bounds;
    if (menuWindow == nil) return;
    GetLocalWindowRect(menuWindow, &bounds);
    PaintRect(&bounds);
}
```

`GetLocalWindowRect` (`GliderPRO/Sources/WindowUtils.c:65-72`) is misleadingly named: it does
`SetPortWindowPort(theWindow)` — **it changes the current port to `menuWindow`** — and then returns
`GetWindowBounds(theWindow, kWindowContentRgn, bounds)`, which is in **global** coordinates. On the
main monitor `thisMac.screen` is `(0, 0, H, W)`, so `menuWindow`'s local origin coincides with the
global origin and the returned `(0, 0, 20, W)` is simultaneously correct as local coordinates —
`PaintRect` blacks out the menu strip, erasing the High-position scoreboard. On a secondary monitor
`thisMac.screen.left` is non-zero (`GetDeviceRect` returns the raw `(*thisGDevice)->gdRect`,
`GliderPRO/Sources/Environ.c:334-342`), the global bounds are `(0, left, 20, left + W)` while
`menuWindow`'s local origin is that same `left`, so the painted rectangle is `left` pixels too far
right and mostly falls outside the 20 x W window — the strip is not erased. The comment at
`MainWindow.c:158` ("Ugly kludge to cover over the menu bar when playing game on 2nd monitor") shows
the author knew this area was fragile.

`menuWindow` is created only in play mode and destroyed on entering edit mode
(`MainWindow.c:187-189`). Its only other use is the `updateEvt` branch at
`GliderPRO/Sources/Events.c:378-384`, which sets the port to `menuWindow`, `BeginUpdate`s, calls
`UpdateMenuBarWindow` (painting the strip black) and `EndUpdate`s — i.e. an exposé of the menu strip
blanks the scoreboard until the next `RefreshScoreboard`.

### 13.13 `BUILD_ARCADE_VERSION` differences

Two conditional blocks blank the scoreboard and then draw something else over it. Both do
`SetGWorld(boardSrcMap, nil); PaintRect(&boardSrcRect);` (painting the strip in the port's current
fore colour) and then blit the now-blank strip to `boardDestRect`:

| Site | Context |
|---|---|
| `GliderPRO/Sources/Play.c:512-571` | at game over, "Need to paint over the scoreboard black", then re-derives `hOffset` with the same `boardSrcRect.right >= 640 ? (RectWide - kMaxViewWidth)/2 : -576` rule and draws a PICT into the strip |
| `Play.c:556-571` | the same idiom a second time |

The non-arcade build never blanks `boardSrcMap`; it relies on `BlackenScoreboard`.

### 13.14 What a Go port must do

* The scoreboard is a separate 20 x screenWidth surface. Build it once: fill the six bevel rows from
  the measured indices, or decode PICT 1997 and blit it at `x = hOffset`; then stamp the three text
  fields into it.
* Do not route it through the frame compositor. Blit it to the framebuffer whenever any of
  `thisRoom->name`, `mortals`, `theScore`, `foilTotal`, `bandsTotal`, `batteryTotal` changes, plus
  once per frame for the roll/blink logic.
* Reproduce the two positions. If the port composites a single logical framebuffer of
  `screenW x screenH` (rather than a 20-px-shorter window offset by 20), then the High position is
  simply "the top 20 rows of the screen" and the Low position is "the 20 rows immediately above the
  central room" — and the `-kScoreboardTall` offsets, the leaked `GrafPort`, `menuWindow` and
  `BlackenScoreboard` all disappear. That is the recommended simplification; it is behaviourally
  identical on the main monitor.
* Keep `justRoomsRect` exactly as the code has it: it is load-bearing for the dirty-rect clamp
  (§9.2) *and* for the transition rectangles (§14).
* Text: `applFont` 12 pt bold, white glyphs at baseline (0, 9) with a black shadow at (1, 10),
  clipped to 256 x 12 / 20 x 10 / 64 x 10.

---

## 14. Room transitions

`GliderPRO/Sources/Transitions.c` is only 145 lines and contains exactly three functions. There are
no dissolves, no fades and no cross-fades in the shipped 1.0.4 renderer — every one was commented
out. What remains is a hard cut and a 4-pixel-per-step wipe.

### 14.1 The three transitions

| Function | Lines | Called from | Effect |
|---|---|---|---|
| `PourScreenOn(Rect*)` | `Transitions.c:18-70` | **nowhere** — dead code | 16 x 20 "chips" rain down 96 independent columns in random order |
| `WipeScreenOn(short direction, Rect*)` | `:74-135` | `Transit.c:300`, `:338`, `:377`, `:416` | 4-px band sweeps across the rect |
| `DumpScreenOn(Rect*)` | `:139-144` | `Play.c:172`, `:177`, `:181` | one `CopyBits` of the whole rect — a hard cut |

All three read from `workSrcMap` and write to
`GetPortBitMapForCopyBits(GetWindowPort(mainWindow))`, with `srcRect == dstRect` — the same
"source rect equals dest rect" invariant as the dirty-rect blitter (§9). None of them touches
`backSrcMap`.

Direction constants (`GliderPRO/Headers/GliderDefines.h:210-213`):

| Constant | Value |
|---|---|
| `kAbove` | 1 |
| `kToRight` | 2 |
| `kBelow` | 3 |
| `kToLeft` | 4 |

### 14.2 `DumpScreenOn` — the hard cut

```c
void DumpScreenOn (Rect *theRect)                          // Transitions.c:139-144
{
    CopyBits((BitMap *)*GetGWorldPixMap(workSrcMap),
             GetPortBitMapForCopyBits(GetWindowPort(mainWindow)),
             theRect, theRect, srcCopy, nil);
}
```

`maskRgn` is `nil` and the current port at the call site is whatever the caller left, so — by the
rule established in 13.11 — this blit is **not** clipped to the window's visible region; it is
clipped only by the destination bitmap's bounds. Its three callers all pass `&justRoomsRect`
(`Play.c:172`, `:177`, `:181`), used for the initial reveal at the start of a game:

| Site | Condition |
|---|---|
| `Play.c:169-173` | `mode == kNewGameMode` -> `BringUpBanner(); DumpScreenOn(&justRoomsRect);` |
| `:174-178` | `mode == kResumeGameMode` -> `DisplayStarsRemaining(); DumpScreenOn(&justRoomsRect);` |
| `:179-182` | otherwise -> `DumpScreenOn(&justRoomsRect);` |

### 14.3 `WipeScreenOn` — the only transition the player ever sees

`GliderPRO/Sources/Transitions.c:74-135`. `kWipeRectThick` = **4** (`:76`).

```
 1  wipeRect = *theRect
 2  switch direction:
 3    case kAbove(1):                                        // glider went up
 4        wipeRect.bottom = wipeRect.top + 4
 5        hOffset = 0;  vOffset = +4
 6        count = ((theRect->bottom - theRect->top) / 4) + 1
 7    case kToRight(2):                                      // glider went right
 8        wipeRect.left = wipeRect.right - 4
 9        hOffset = -4; vOffset = 0
10        count = workSrcRect.right / 4                       // NOTE: not theRect's width
11    case kBelow(3):                                        // glider went down
12        wipeRect.top = wipeRect.bottom - 4
13        hOffset = 0;  vOffset = -4
14        count = ((theRect->bottom - theRect->top) / 4) + 1
15    case kToLeft(4):                                        // glider went left
16        wipeRect.right = wipeRect.left + 4
17        hOffset = +4; vOffset = 0
18        count = workSrcRect.right / 4
19  dummyRgn = NewRgn()
20  for i = 0 .. count-1:
21      CopyBits(workSrcMap -> mainWindow, &wipeRect, &wipeRect, srcCopy,
22               GetPortVisibleRegion(GetWindowPort(mainWindow), dummyRgn))
23      QOffsetRect(&wipeRect, hOffset, vOffset)
24      if wipeRect.top    < theRect->top:    wipeRect.top = theRect->top
25      elif wipeRect.top  > theRect->bottom: wipeRect.top = theRect->bottom
26      if wipeRect.bottom < theRect->top:    wipeRect.bottom = theRect->top
27      elif wipeRect.bottom > theRect->bottom: wipeRect.bottom = theRect->bottom
28  DisposeRgn(dummyRgn)
```

Everything a port needs to know about this, in order of how easy it is to get wrong:

* **The band always sweeps *away from* the edge the glider left through.** `kAbove` starts as the
  top 4 rows and marches down; `kToRight` starts as the rightmost 4 columns and marches *left*
  (`hOffset = -4`). So the new room appears to slide in from the direction of travel.
* **There is no delay and no frame pacing.** The loop is a tight `for` with a blit per iteration.
  On a 1994 68030 that is the transition's speed; on modern hardware it is instantaneous. A Go port
  that wants the original feel must add pacing — the original wipe on a 640-wide screen at
  `kAbove` is 116 blits of 640 x 4 = ~297 KB total, which on a period Mac took roughly a third of a
  second.
* **`maskRgn` is the window's visible region, passed explicitly** (`:120`) — the only transition that
  bothers. This is why the wipe never scribbles outside the window even though `DumpScreenOn` can.
* **The clamp at `:124-131` only clamps `top` and `bottom`, never `left`/`right`.** For the two
  horizontal directions the band therefore walks straight off the left or right edge and the last
  iterations blit nothing; correctness depends entirely on `CopyBits` clipping. A Go port must clip
  the rect (or skip empty rects) or it will index out of bounds.
* **`count` for horizontal wipes uses `workSrcRect.right / 4`, not the passed rect's width**
  (`:96`, `:110`). Because `justRoomsRect` is always full width, `workSrcRect.right` and
  `theRect->right - theRect->left` happen to agree in every shipped call; the discrepancy is latent,
  not live. Vertical wipes correctly use the rect's own height, `+1` so the last partial band is
  covered.
* Vertical `count` values for a 640 x 480 screen: High mode `justRoomsRect` = (0,0,460,640) ->
  `count` = 116; Low mode `justRoomsRect` = (79,0,401,640) -> `count` = 81. Horizontal `count` = 160
  in both.
* The band's *width* never changes, only its position, and `srcRect == dstRect`, so the wipe reveals
  the already-finished `workSrcMap` in place. It is a reveal, not a slide: nothing moves.

### 14.4 `PourScreenOn` — dead, but documented

`GliderPRO/Sources/Transitions.c:18-70`. Not referenced anywhere in the source
(`GliderProtos.h:495` declares it; nothing calls it). Kept here because it is the only other
transition style the author implemented and a port may want it.

```
 1  #define kMaxColumnsWide 96;  kChipHigh 20;  kChipWide 16
 2  Rect  columnRects[96];  short columnProgress[96]
 3  colWide = theRect->right / 16                  // NOTE: right, not width
 4  rowTall = (theRect->bottom / 20) + 1
 5  for i in 0..colWide-1:
 6      columnProgress[i] = 0
 7      columnRects[i] = (0,0,16,20) offset ((i*16) + theRect->left, theRect->top)
 8  while working:
 9      do i = RandomInt(colWide) while columnProgress[i] >= rowTall     // reject full columns
10      clamp columnRects[i] to theRect on all four sides
11      CopyBits(workSrcMap -> mainWindow, &columnRects[i], &columnRects[i], srcCopy, nil)
12      QOffsetRect(&columnRects[i], 0, 20)
13      columnProgress[i]++
14      if columnProgress[i] >= rowTall:
15          colsComplete++;  if colsComplete >= colWide: working = false
```

Notes: the array bound 96 is exactly `kMaxViewWidth`(1536) / `kChipWide`(16), so the clamp of
`houseRect.right` to 1536 (§2.3) is what keeps it in bounds. The `do/while` at step 9 is a rejection
sampler that gets progressively slower as columns fill — the expected number of `RandomInt` calls is
`colWide * rowTall * H(colWide)` -ish, i.e. ~5x the number of blits for 96 columns. Unlike
`WipeScreenOn` it clamps all four sides (`:48-55`) and passes `nil` for `maskRgn`.

### 14.5 The four commented-out dissolves

Two functions were removed from the build entirely — `DissBits(Rect*)` and
`DissBitsChunky(Rect*)`, both still declared (commented out) at `GliderProtos.h:498-499`. Every call
site survives as a comment, and the pattern is always the same:

```c
//  if (quickerTransitions)
//      DissBitsChunky(&someRect);
//  else
//      DissBits(&someRect);
```

| Site | Rect | Context |
|---|---|---|
| `GliderPRO/Sources/Play.c:156-159` | `&workSrcRect` | after blanking the screen at game start |
| `Play.c:165-168` | `&justRoomsRect` | after `DrawLocale` + `RefreshScoreboard` |
| `Play.c:816-819` | `&justRoomsRect` | `UpdateWholeScreen` |
| `GliderPRO/Sources/MainWindow.c:109-112` | `&workSrcRect` | `RedrawSplashScreen` |
| `GliderPRO/Sources/Banner.c:178-181`, `:194-197` | `&justRoomsRect` | banner up / banner down |
| `GliderPRO/Sources/HighScores.c:68-71`, `:76-79` | `&workSrcRect` | high-score screen in / out |

`quickerTransitions` is therefore a **live preference with no effect**: it is loaded from prefs
(`Main.c:82`), defaulted to `false` (`Main.c:153`, `Settings.c:1258`), saved back
(`Main.c:239`), and offered in the Preferences dialog (`Settings.c:241`, `:271`) — but the only code
that ever read it is commented out. A Go port can drop it or wire it to a real dissolve.

### 14.6 What a room change actually does, end to end

`MoveRoomToRoom(gliderPtr thisGlider, short where)`
(`GliderPRO/Sources/Transit.c:151-310`) is the canonical path; `TransportRoomToRoom`
(`:314-348`), `MoveDuctToDuct` (`:352-387`) and `MoveMailToMail` (`:391-426`) are the same tail with
different glider setup.

```
 1  HandleRoomVisitation()                                  // scoring / "visited" flag
 2  switch (where):
 3      kToRight: InsureGliderFacingRight(); ForceThisRoom(localNumbers[kEastRoom])
 4                OffsetGlider(g, kToLeft);  enteredRect = (0,0,48,20) offset
 5                   (0, kGliderStartsDown(32) + thisRoom->leftStart - 2)
 6      kToLeft:  InsureGliderFacingLeft();  ForceThisRoom(localNumbers[kWestRoom])
 7                OffsetGlider(g, kToRight); enteredRect = (0,0,48,20) offset
 8                   (kRoomWide(512) - 48, kGliderStartsDown + thisRoom->rightStart - 2)
 9      kAbove:   ForceThisRoom(localNumbers[kNorthRoom])
10                if !takingTheStairs: OffsetGlider(g, kBelow); enteredRect = g->dest
11                else:                ReadyGliderForTripUpStairs(g)
12      kBelow:   ForceThisRoom(localNumbers[kSouthRoom])
13                if !takingTheStairs: OffsetGlider(g, kAbove); enteredRect = g->dest
14                else:                ReadyGliderForTripDownStairs(g)
15  if twoPlayerGame && !onePlayerLeft:  TagGliderIdle(the other glider)
16  ReadyLevel()                                            // rebuild backSrcMap + workSrcMap
17  RefreshScoreboard(kNormalTitleMode)                      // full strip rebuild, straight to screen
18  WipeScreenOn(where, &justRoomsRect)                      // reveal
19  #ifdef COMPILEQT
20      RenderFrame()                                        // one frame, so sprites appear
21      if hasQT && hasMovie && tvInRoom && tvOn:  GoToBeginningOfMovie; StartMovie
22  #endif
```

`ReadyLevel` (`GliderPRO/Sources/RoomGraphics.c:402-418`):

```
1  NilSavedMaps()                                    // dispose all 24 savedMaps GWorlds
2  #ifdef COMPILEQT
3      if hasQT && hasMovie && tvInRoom:  tvInRoom = false; tvWithMovieNumber = -1; StopMovie
4  #endif
5  DetermineRoomOpenings()
6  DrawLocale()                                      // full 9-room repaint of backSrcMap, then
7                                                    //   RestoreWorkMap() copies it to workSrcMap
8  InitGarbageRects()
```

`InitGarbageRects` (`GliderPRO/Sources/Render.c:675-691`) is the frame-state reset:

```
1  numWork2Main = 0;  numBack2Work = 0                        // drop all pending dirty rects
2  numSparkles = 0;   for i in 0..kMaxSparkles-1:  sparkles[i].mode = -1
3  numFlyingPts = 0;  for i in 0..kMaxFlyingPts-1: flyingPoints[i].mode = -1
4  nextFrame = TickCount() + kTicksPerFrame                   // resync the frame clock
```

So the transition ordering is: **rebuild the offscreen world -> repaint the scoreboard directly to
the screen -> wipe the offscreen world onto the screen -> render exactly one frame of sprites.**
Step 20's `RenderFrame()` exists only inside `#ifdef COMPILEQT`; without QuickTime the first frame
of sprites is drawn by the normal game loop instead, one tick later.

Note the ordering hazard a port must preserve: `WipeScreenOn` reveals `workSrcMap`, which at that
moment contains only the static room (no glider, no flames — `RestoreWorkMap` has just overwritten
it from `backSrcMap`). The glider appears on the first `RenderFrame` *after* the wipe finishes.

---

## 15. Full-screen set pieces: splash, banner, stars-remaining, high scores, game over

Everything in this section draws into `workSrcMap` at *window* resolution (not room resolution), so
none of it uses `playOriginH/V`; it uses `splashOriginH/V` or `CenterRectInRect(&r, &thisMac.screen)`
instead. All of it is 1-shot rather than per-frame, and most of it is a victim of the
commented-out dissolve transitions (see 15.7).

### 15.1 `splashOriginH` / `splashOriginV`

Computed once in `OpenMainWindow` (`GliderPRO/Sources/MainWindow.c:238-243`):

```
splashOriginH = (RectWide(&thisMac.screen) - 640) / 2;  if (< 0) splashOriginH = 0;   // :238-240
splashOriginV = (RectTall(&thisMac.screen) - 480) / 2;  if (< 0) splashOriginV = 0;   // :241-243
```

The constant is **480** even though the splash artwork is 460 tall, because the window is 20 px
shorter than the screen: `(screenH - 480)/2 == (windowH + 20 - 480)/2 == (windowH - 460)/2`, i.e.
the 460-tall graphic ends up exactly vertically centred in the window. Worked values:

| Screen | Window (`mainWindowRect`) | `splashOriginH` | `splashOriginV` | splash occupies window y |
|---|---|---|---|---|
| 512 x 342 | 512 x 322 | 0 (clamped from -64) | 0 (clamped from -69) | 0..459 (199 rows clipped off the bottom) |
| 640 x 480 | 640 x 460 | 0 | 0 | 0..459 (exact fit) |
| 800 x 600 | 800 x 580 | 80 | 60 | 60..519 |
| 832 x 624 | 832 x 604 | 96 | 72 | 72..531 |
| 1024 x 768 | 1024 x 748 | 192 | 144 | 144..603 |
| 1152 x 870 | 1152 x 850 | 256 | 195 | 195..654 |

On a 512 x 342 Mac both origins clamp to 0 and the splash is simply cropped; the game does not
scale it down.

### 15.2 The splash screen

`kSplash8BitPICT` = 1000 (`GliderPRO/Headers/GliderDefines.h:524`). Measured: PICT 1000 is
640 x 460 v2, stored `picSize` = -22590 (the signed-16-bit overflow described in 5.3), actual
108482 bytes (`/tmp/wf-render/pictframes.txt` row 1000). There is no separate 4-bit splash resource
in the shipped file even though `kSplash4BitPICT` does not exist as a define; the depth-4 path only
changes the *text* colours (15.3).

Two different code paths draw it, and **they disagree about the origin**:

| Site | Call | Anchor |
|---|---|---|
| `MainWindow.c:245-247` (`OpenMainWindow`, first time) | `SetPort(workSrcMap); PaintRect(&workSrcRect); LoadGraphic(kSplash8BitPICT)` | PICT's own `picFrame`, i.e. window (0,0) — **not** centred |
| `MainWindow.c:102-107` (`RedrawSplashScreen`, every later time) | `PaintRect(&workSrcRect); QSetRect(&tempRect,0,0,640,460); QOffsetRect(&tempRect, splashOriginH, splashOriginV); LoadScaledGraphic(kSplash8BitPICT, &tempRect)` | centred via `splashOrigin` |

On a 640 x 480 screen both are identical because `splashOrigin` is (0,0). On anything bigger the
very first splash is drawn hard against the top-left corner and every subsequent one is centred. A
Go port should pick the centred behaviour and note the divergence.

### 15.3 `DrawOnSplash`

`GliderPRO/Sources/MainWindow.c:56-94`. Draws two pieces of text on top of the splash, in the
current port (the caller has set it to `workSrcMap`).

```
 1  PasStringCopy("\pHouse: ", houseLoadedStr)                       // :64
 2  PasStringConcat(houseLoadedStr, thisHouseName)                   // :65
 3  #ifdef COMPILEQT
 4     if (hasMovie) PasStringConcat(houseLoadedStr, "\p (QT)")      // :68-69
 5  TextSize(9); TextFace(1 /*bold*/); TextFont(applFont)            // :71-73
 6  MoveTo(splashOriginH + 436, splashOriginV + 314)                 // :74
 7  if (thisMac.isDepth == 4):                                       // :76
 8      if houseIsReadOnly: ForeColor(whiteColor) else ForeColor(blackColor)
 9      DrawString(houseLoadedStr)
10  else:
11      if houseIsReadOnly: ColorText(houseLoadedStr, 5L)            // :83
12      else:              ColorText(houseLoadedStr, 28L)            // :85
13  #if powerc                                                       // :88-93
14     PasStringCopy("\pPowerPC Native!", houseLoadedStr)
15     TextSize(12); TextFace(0); TextFont(systemFont)
16     ForeColor(blackColor); MoveTo(splashOriginH + 5, splashOriginV + 457); DrawString
17     ForeColor(whiteColor); MoveTo(splashOriginH + 4, splashOriginV + 456); DrawString
```

Notes for a port:

- Text baseline positions are **relative to `splashOrigin`**, so they track the centred artwork.
- `ColorText(str, color)` (`GliderPRO/Sources/ColorUtils.c:20`) does `Index2Color(color)` +
  `RGBForeColor` + `DrawString`, so the `5L` / `28L` are *palette indices*, not QuickDraw colour
  constants. Measured from the CLUT (5.4): index 5 = `#FFFF00` (pure yellow), index 28 = `#FF3333`
  (red). Read-only houses get their name in yellow, writable ones in red.
- The "PowerPC Native!" tag is compiled in only on PowerPC and is a 1-px drop shadow (black at
  `(+5,+457)`, white at `(+4,+456)`).
- Everything is drawn with the *application* font at 9 pt bold except the PowerPC tag, which is the
  *system* font at 12 pt plain.

### 15.4 `RedrawSplashScreen` — the backwards copy

`GliderPRO/Sources/MainWindow.c:98-114`:

```
 98  void RedrawSplashScreen (void)
102      SetPort((GrafPtr)workSrcMap);
103      PaintRect(&workSrcRect);
104      QSetRect(&tempRect, 0, 0, 640, 460);
105      QOffsetRect(&tempRect, splashOriginH, splashOriginV);
106      LoadScaledGraphic(kSplash8BitPICT, &tempRect);
107      DrawOnSplash();
108      SetPortWindowPort(mainWindow);
109  //   if (quickerTransitions)
110  //       DissBitsChunky(&workSrcRect);
111  //   else
112  //       DissBits(&workSrcRect);
113      CopyRectMainToWork(&workSrcRect);
114  }
```

Line 113 is backwards. `CopyRectMainToWork` copies **window -> workSrcMap** (`Render.c:723-729`);
the function has just spent lines 102-107 composing a fresh splash *into* `workSrcMap` and line 113
throws it away by overwriting it with whatever is currently on screen. The intended call is
`CopyRectWorkToMain` (`Render.c:713-719`). Consequence: `RedrawSplashScreen` never puts anything on
screen. It is called from `DoHighScores` (`HighScores.c:84`), `DoGameOver` (`GameOver.c:68`),
`DoDiedGameOver` (`GameOver.c:505`) and the mode-change paths, so all of those end with the screen
still showing the previous contents.

This is almost certainly collateral damage from commenting out `DissBits` at lines 109-112: with the
dissolve active, the dissolve was what published `workSrcMap` to the screen and line 113 was a
*resync* of `workSrcMap` from the screen afterwards (harmless, because they were then identical).
Removing the dissolve turned the resync into an erase. A faithful port reproduces the stale screen;
a corrected port replaces line 113 with a work->main blit. **Recommendation: implement the corrected
behaviour and document it**, because otherwise the splash, high-score and game-over screens are
literally never visible.

### 15.5 The banner (`Banner.c`)

Resource IDs (`GliderPRO/Sources/Banner.c:17-21`):

| Constant | Value | Measured | Purpose |
|---|---|---|---|
| `kBannerPageTopPICT` | 1993 | 330 x 190 v2, `picSize` -21600, 43936 actual bytes | top 190 rows of the notebook page |
| `kBannerPageBottomPICT` | 1992 | 330 x 30 v2, 9360 bytes | bottom 30 rows (torn edge) |
| `kBannerPageBottomMask` | 1991 | 330 x 30 **v1**, 477 bytes | 1-bit mask for the torn edge |
| `kStarsRemainingPICT` | 1017 | 256 x 64 v2, 9770 bytes | "N stars remaining" plaque (plural) |
| `kStarRemainingPICT` | 1018 | 256 x 64 v2, 9742 bytes | "1 star remaining" plaque (singular) |

(all measurements from `/tmp/wf-render/pictframes.txt`)

#### `DrawBanner(Point *topLeft)` — `Banner.c:41-84`

```
 1  QSetRect(&wholePage, 0, 0, 330, 220)                     // :52
 2  mapBounds = thisMac.screen;  ZeroRectCorner(&mapBounds)  // :53-54
 3  CenterRectInRect(&wholePage, &mapBounds)                 // :55
 4  topLeft->h = wholePage.left;  topLeft->v = wholePage.top // :56-57
 5  partPage = wholePage;  partPage.bottom = partPage.top + 190
 6  SetGWorld(workSrcMap, nil)                               // :61
 7  LoadScaledGraphic(kBannerPageTopPICT, &partPage)         // :62
 8  partPage = wholePage;  partPage.top = partPage.bottom - 30
 9  CreateOffScreenGWorld(&tempMap,  &(0,0,30,330), kPreferredDepth); LoadGraphic(1992)
10  CreateOffScreenGWorld(&tempMask, &(0,0,30,330), 1);       LoadGraphic(1991)
11  CopyMask(tempMap, tempMask -> workSrcMap, &mapBounds30, &mapBounds30, &partPage)
12  DisposeGWorld(tempMap);  DisposeGWorld(tempMask)
13  SetGWorld(wasCPort, wasWorld)
```

So the page is a 330 x 220 rect centred in the *screen* (with the screen's own origin subtracted, so
it is in window coordinates); rows 0..189 are an opaque PICT and rows 190..219 are a masked PICT so
the ragged torn bottom shows the room behind it. The two temp GWorlds are created and destroyed on
every call.

#### `DrawBannerMessage(Point topLeft)` — `Banner.c:117-166`

```
 1  TextFont(applFont); TextFace(bold); TextSize(12)          // :128-130
 2  count = 0
 3  do:
 4      GetLineOfText(bannerStr, count, subStr)               // :135
 5      MoveTo(topLeft.h + 16, topLeft.v + 32 + count*20)     // :136
 6      DrawString(subStr)
 7      count++
 8  while (subStr[0] > 0)
 9  if bannerStarCountOn:                                     // :143
10      ForeColor(redColor)
11      GetLocalizedString(1..5) assembled into two lines
12      MoveTo(topLeft.h + 16, topLeft.v + 164);  DrawString(line1)
13      MoveTo(topLeft.h + 16, topLeft.v + 180);  DrawString(line2)
14      ForeColor(blackColor)
```

Line spacing 20 px, first baseline at page-relative y 32, left margin 16 px. The star-count lines
are at fixed page-relative y 164 and 180 in red, independent of how many message lines there were —
a long banner will overlap them.

`bannerStr` comes from the house's `banner` field; `bannerStarCountOn` is set by
`CountStarsInHouse` (`Banner.c:89-111`), which walks every room and every object and counts
`what == kStar` (0x2C) plus `kInvisBonus`-family bonuses into `numStarsRemaining`.

#### `BringUpBanner` — `Banner.c:171-198` — the banner is never seen

```
171  void BringUpBanner (void)
176      DrawBanner(&topLeft);
177      DrawBannerMessage(topLeft);
178  //   if (quickerTransitions)
179  //       DissBitsChunky(&justRoomsRect);
180  //   else
181  //       DissBits(&justRoomsRect);
183      QSetRect(&wholePage, 0, 0, 330, 220);
184      QOffsetRect(&wholePage, topLeft.h, topLeft.v);
185      CopyBits(backSrcMap -> workSrcMap, &wholePage, &wholePage, srcCopy, nil);
190      WaitForInputEvent(demoGoing ? 4 : 15);
```

Line 185 restores the page area of `workSrcMap` from `backSrcMap` — i.e. it *erases* the banner —
and nothing between line 177 and line 185 copies `workSrcMap` to the screen. The caller
(`GliderPRO/Sources/Play.c:169-173`) is:

```
169  if (mode == kNewGameMode)
171      BringUpBanner();
172      DumpScreenOn(&justRoomsRect);
```

and `DumpScreenOn` runs *after* the erase, so it publishes a banner-free frame. The 15-tick
(0.25 s) wait at line 190 therefore just freezes the previous screen contents. With the two
`DissBits` calls live, the order was: draw banner into `workSrcMap` -> dissolve to screen (banner
visible) -> restore `workSrcMap` from `backSrcMap` -> wait -> (caller's dissolve) publish the
banner-free frame. Commenting out the dissolve removed both the reveal *and* nothing else, so the
banner is dead code in 1.0.4. A Go port that wants the banner must blit `workSrcMap`'s page rect to
the screen between steps 177 and 183.

#### `DisplayStarsRemaining` — `Banner.c:205-236`

This one *is* visible, because it draws straight into the window:

```
 1  SetPortWindowPort(mainWindow)                                  // :210
 2  QSetRect(&bounds, 0, 0, 256, 64)                               // :212
 3  CenterRectInRect(&bounds, &thisMac.screen)                     // :213
 4  src = bounds;  InsetRect(&src, 64, 32)                         // :214-215  (computed, never used)
 5  QOffsetRect(&bounds, -thisMac.screen.left, -thisMac.screen.top) // :217
 6  QOffsetRect(&bounds, 0, -20)                                   // :222
 7  if numStarsRemaining == 1: LoadScaledGraphic(1018, &bounds)
 8  else:                      LoadScaledGraphic(1017, &bounds)
 9  NumToString(numStarsRemaining, theStr) ... build message
10  MoveTo(bounds.left + 102 - StringWidth(theStr)/2, bounds.top + 23)
11  ColorText(theStr, 4L)
12  DelayTicks(60)
13  if (WaitForInputEvent(30)) RestoreEntireGameScreen()
14  CopyRectWorkToMain(&bounds)                                    // :235
```

The `-20` at step 6 is the same menu-bar compensation used by the scoreboard (13.3): the plaque is
centred on the *screen* but drawn in *window* coordinates, so it must move up by the menu-bar
height. Step 14 erases the plaque by copying the region back from `workSrcMap`, which is why the
plaque had to be drawn to the window rather than to `workSrcMap`. The count string is centred on
page-relative x 102 (not on the 128 midpoint) because the plaque artwork has the words to the right.
`ColorText(theStr, 4L)` = palette index 4 = `#FFFF33`.

### 15.6 The high-score screen

`kHighScoresPictID` = 1994 (332 x 30 v2, 4972 bytes), `kHighScoresMaskID` = 1998 (332 x 30 **v1**,
1049 bytes) — the "High Scores" title plaque and its mask (`HighScores.c:23-24`). The background is
`kStarPictID` = 1995 (`GliderDefines.h:534`), measured 640 x **460** v2, 20746 bytes.

`DoHighScores` — `GliderPRO/Sources/HighScores.c:58-85`:

```
 1  SpinCursor(3)
 2  SetPort((GrafPtr)workSrcMap);  PaintRect(&workSrcRect)      // :63-64
 3  QSetRect(&tempRect, 0, 0, 640, 480)                         // :65   <-- 480, not 460
 4  QOffsetRect(&tempRect, splashOriginH, splashOriginV)        // :66
 5  LoadScaledGraphic(kStarPictID, &tempRect)                   // :67
 6  // DissBits(&workSrcRect)                                   // :68-71 commented out
 7  SetPort((GrafPtr)workSrcMap);  DrawHighScores()             // :73-74
 8  // DissBits(&workSrcRect)                                   // :76-79 commented out
 9  InitCursor();  DelayTicks(60);  WaitForInputEvent(30)       // :80-82
10  RedrawSplashScreen()                                        // :84
```

Step 3/5 is a genuine **non-uniform scale**: the 640 x 460 star field is stretched into a 640 x 480
rect, a vertical factor of 480/460 = 1.04348. On a 640 x 480 screen (window 460 tall) the bottom 20
rows are then clipped by the destination bitmap, so the visible result is the top 460/480 = 95.8% of
the stretched image. `LoadScaledGraphic` is `DrawPicture` into an arbitrary rect (4.3), which is
QuickDraw's own scaler — nearest-neighbour row/column replication for an indexed-colour source.

As with the banner, both `DissBits` calls are commented out and `RedrawSplashScreen` ends with the
backwards copy, so nothing reaches the screen: **the high-score table is composed entirely
offscreen and discarded**.

#### `DrawHighScores` layout — `HighScores.c:94-276`

Constants (`:90-92`): `kScoreSpacing` = 18, `kScoreWide` = 352, `kKimsLifted` = 4.

```
scoreLeft = (RectWide(&thisMac.screen) - 352) / 2          // :106
dropIt    = 129 + splashOriginV                            // :107
```

`scoreLeft` is measured from the **screen** width but used as a window x-coordinate; since the
window is the full screen width horizontally that is correct. `dropIt` is the baseline of score
row 0 (well, of rows 1..9 — row 0 is lifted, see below).

The title plaque, drawn through its mask into `workSrcMap` (`:111-129`):

```
tempRect  = (0,0,30,332)
tempRect2 = tempRect offset by (scoreLeft + (352-332)/2, dropIt - 60)
          = (scoreLeft + 10, dropIt - 60, ...)
CopyMask(tempMap[PICT 1994], tempMask[PICT 1998] -> workSrcMap, tempRect, tempRect, tempRect2)
```

Both temp GWorlds are created (depth 8 and depth 1) and disposed inside the function.

Text, all as 1-px drop shadows (black at `+1` in x and y, colour at the nominal position):

| Item | Font | x (relative to `scoreLeft`) | y (relative to `dropIt`) | shadow colour | face colour | Lines |
|---|---|---|---|---|---|---|
| `"* " + thisHouseName + " *"` (bullet chars 0xC5) | applFont 14 bold | centred in 352: `(352 - StringWidth)/2`, shadow at `-1` | shadow `-66`, face `-65` | black | cyan | :133-146 |
| house `banner` (the message for score #1) | applFont 12 bold | centred in 352 | shadow `-4`, face `-5` | black | yellow | :148-163 |
| box around the banner | - | `-3 + (352-bannerWidth)/2`, width `bannerWidth+8`, height 18 | top `dropIt + 5 - 18 - 4`, second frame offset `(-1,-1)` | black | yellow | :165-172 |
| placing number `i+1` | applFont 12 bold | shadow `+1`, face `+0` | see row-y rule | black | cyan, or **white** if `i == lastHighScore` | :179-194 |
| player name | " | shadow `+31`, face `+30` | " | black | yellow, or white if `i == lastHighScore` | :196-211 |
| level (room) number | " | shadow `+161`, face `+160` | " | black | yellow / white | :213-228 |
| the word "room"/"rooms" (localized string 6 or 7) | " | shadow `+193`, face `+192` | " | black | cyan (never white) | :230-245 |
| score points | " | shadow `+291`, face `+290` | " | black | yellow / white | :247-262 |
| localized string 8 (the footer) | applFont **9** bold | `+80` | `dropIt - 1 + 10*18` = `dropIt + 179` | - | blue | :266-272 |

Row-y rule (repeated verbatim at every one of the five columns):

```
if (i == 0)  shadowY = dropIt - kScoreSpacing - kKimsLifted        // = dropIt - 22
             faceY   = dropIt - 1 - kScoreSpacing - kKimsLifted    // = dropIt - 23
else         shadowY = dropIt + i*kScoreSpacing                    // = dropIt + 18i
             faceY   = dropIt - 1 + i*kScoreSpacing
```

So row 0 sits 22 px *above* `dropIt` and rows 1..9 march downward from `dropIt` in 18-px steps —
i.e. there is a 40-px gap between row 0 and row 1, which is where the banner text and its box go.
`kMaxScores` = 10 (`GliderDefines.h:249`) and rows with `scores[i] <= 0` are skipped entirely
(`:176`).

`lastHighScore` is the index the player just achieved; that row is drawn in white instead of
cyan/yellow, except the "rooms" word which stays cyan.

For a 640 x 480 screen: `scoreLeft` = (640-352)/2 = **144**, `splashOriginV` = 0 so `dropIt` =
**129**. Row 0 baselines at y 106/107, row 1 at 128/129, row 9 at 290/291, footer at y 308, title
plaque at (154, 69)-(486, 99), house-name line at y 63/64.

### 15.7 Game over — house completed (`DoGameOver`)

`GliderPRO/Sources/GameOver.c:60-69`:

```
62  playing = false
63  SetUpFinalScreen()
64  SetPort((GrafPtr)mainWindow)
65  ColorRect(&mainWindowRect, 244)          // palette index 244 = #000011, near-black blue
66  DoGameOverStarAnimation()
67  if (!TestHighScore()) RedrawSplashScreen()
```

`SetUpFinalScreen` (`:76-129`):

```
 1  SetPort((GrafPtr)workSrcMap);  ColorRect(&workSrcRect, 244)      // :83-84
 2  QSetRect(&tempRect, 0, 0, 640, 460)                              // :85
 3  CenterRectInRect(&tempRect, &workSrcRect)                        // :86
 4  LoadScaledGraphic(kMilkywayPictID /*1021*/, &tempRect)           // :87
 5  textDown = tempRect.top;  if (textDown < 0) textDown = 0         // :88-90
 6  tempStr = (*thisHouse)->trailer                                  // :94
 7  count = 0
 8  do:
 9      GetLineOfText(tempStr, count, subStr)
10      offset = (RectWide(&thisMac.screen) - TextWidth(subStr,1,subStr[0])) / 2
11      TextFont(applFont); TextFace(bold); TextSize(12)
12      ForeColor(blackColor); MoveTo(offset+1, textDown+33+count*20); DrawString(subStr)
13      ForeColor(whiteColor); MoveTo(offset,   textDown+32+count*20); DrawString(subStr)
14      ForeColor(blackColor); count++
15  while (subStr[0] > 0)
16  CopyRectWorkToBack(&workSrcRect)                                 // :117
17  for i in 0..4:                                                   // :119-128
18      pages[i].dest = starSrc[0]
19      QOffsetRect(&pages[i].dest,
20                  workSrcRect.right + RandomInt(workSrcRect.right/5)
21                                    + (workSrcRect.right/4)*i,
22                  RandomInt(workSrcRect.bottom) - workSrcRect.bottom/2)
23      pages[i].was = pages[i].dest;  pages[i].frame = RandomInt(6)
```

`kMilkywayPictID` = 1021, measured 640 x 460 v2, stored `picSize` -14016, 51520 actual bytes.
Note this one is scaled into a 640 x 460 rect (no distortion), centred with `CenterRectInRect`
against `workSrcRect` rather than via `splashOrigin` — a third centring convention in the same
program. `textDown` is the top of that centred rect, clamped to >= 0, and the trailer text starts
33 px below it with 20-px line spacing, horizontally centred per line.

Step 16 is important: the finished background (star field + trailer text) is copied
`workSrcMap -> backSrcMap`, so that the dirty-rect machinery (section 9) can restore behind the
animating stars and angel exactly as it does during play.

#### `DoGameOverStarAnimation` — `GameOver.c:136-230`

`kStarFalls` = 8 (`:138`). Frame pacing is `nextLoop = TickCount() + 2` (`:149`, `:220`) — the same
2-tick / ~30 fps budget as the game loop, but implemented locally rather than via `nextFrame`.

```
 1  angelDest = angelSrcRect (96 x 44);  QOffsetRect(&angelDest, -96, 0)   // :146-147
 2  loop while noInteruption:
 3      if (angelDest.left % 32) == 0:                          // :156
 4          PlayPrioritySound(kMysticSound, kMysticPriority)
 5          which = (angelDest.left / 32) % 5                   // :159-160
 6          ZeroRectCorner(&pages[which].dest)
 7          QOffsetRect(&pages[which].dest, angelDest.left, angelDest.bottom)
 8          if (count < which+1) count = which+1
 9      for i in 0..count-1:                                    // :167
10          pages[i].frame = (pages[i].frame + 1) % 6           // :169-171
11          CopyMask(bonusSrcMap, bonusMaskMap -> workSrcMap,
12                   &starSrc[frame], &starSrc[frame], &pages[i].dest)   // :173-178
13          pages[i].was = pages[i].dest;  pages[i].was.top -= 8
14          AddRectToWorkRectsWhole(&pages[i].was)              // :183
15          AddRectToBackRects(&pages[i].dest)                  // :184
16          if (pages[i].dest.top < workSrcRect.bottom) QOffsetRect(&pages[i].dest, 0, 8)
17      if angelDest.left <= workSrcRect.right + 2:             // :190
18          CopyMask(angelSrcMap, angelMaskMap -> workSrcMap,
19                   &angelSrcRect, &angelSrcRect, &angelDest)  // :192-195
20          angelDest.left -= 2;  AddRectToWorkRectsWhole(&angelDest);  angelDest.left += 2
21          AddRectToBackRects(&angelDest)                      // :199
22          QOffsetRect(&angelDest, 2, 0);  pass = 0
23      CopyRectsQD()                                           // :204
24      numWork2Main = 0;  numBack2Work = 0                     // :206-207
25      busy-wait until TickCount() >= nextLoop, polling GetKeys / GetNextEvent
26      nextLoop = TickCount() + 2
27      if (pass < 80) pass++ else { WaitForInputEvent(5); noInteruption = false }
```

Points a port must get right:

- The angel flies **left to right** at 2 px/frame starting fully off the left edge
  (`angelDest = (-96, 0, 0, 44)`), and a new star is spawned at the angel's bottom-left every time
  its left edge is a multiple of 32 — i.e. every 16 frames. Only 5 star slots exist and they are
  reused round-robin (`which = (left/32) % 5`), so a still-falling star is teleported back up to
  the angel when its slot comes round again.
- Stars fall 8 px/frame and stop being advanced once `dest.top >= workSrcRect.bottom`, but they are
  still redrawn every frame at that final position (the `CopyMask` at step 12 is unconditional), so
  a partially-visible star remains stuck at the bottom edge.
- `pages[i].was` for a star is the *swept* rect (`dest` with `top` pulled up 8 px) and goes to
  `work2MainRects` via `AddRectToWorkRectsWhole`; the un-swept `dest` goes to `back2WorkRects`.
  Same discipline as `RenderFlyingPoints` (11.6).
- The angel's swept rect is `angelDest` with `left` pulled back 2 px.
- Steps 23-24 call `CopyRectsQD()` and then **manually zero the counters** instead of letting
  `RenderFrame` do it — this loop is outside the normal renderer.
- Termination: `pass` counts frames since the angel was last drawn; once the angel has been off the
  right edge for 80 consecutive frames (~2.7 s) the loop waits up to 5 ticks for input and exits.
  Any modifier key or mouse/key event also exits.
- The falling stars reuse the *bonus* sheet: `bonusSrcMap` / `bonusMaskMap` and `starSrc[0..5]`
  (6 frames), the same 88 x 378 sheet the in-room star prize uses (7.5, 11.5).
- `angelSrcMap` / `angelMaskMap` are 96 x 44 depth-8 / depth-1 GWorlds created once in
  `StructuresInit2.c:127-134` from `kAngelPictID` = 1019 (96 x 44 v2, 4064 bytes) and its mask
  1020 (96 x 44 **v1**, 457 bytes). They are never disposed.

### 15.8 Game over — died (`DoDiedGameOver`)

Resource IDs (`GameOver.c:17-22`):

| Constant | Value | Measured | Purpose |
|---|---|---|---|
| `kNumCountDownFrames` | 16 | - | delay in frames between death and the sequence |
| `kPageFrames` | 14 | - | animation frames per sheet of paper |
| `kPagesPictID` | 1990 | 32 x 448 v2, 10052 bytes | 14 vertically stacked 32 x 32 paper frames |
| `kPagesMaskID` | 1989 | 32 x 448 **v1**, 1851 bytes | its mask |
| `kLettersPictID` | 1988 | 25 x 256 v2, 5900 bytes | 8 vertically stacked 25 x 32 letters ("GAME OVER") |

`pageType` (`GameOver.c:25-30`), 8 instances in `pages[8]` (`:40`):

| Field | Type | Bytes |
|---|---|---|
| `dest` | `Rect` | 8 |
| `was` | `Rect` | 8 |
| `frame` | `short` | 2 |
| `counter` | `short` | 2 |
| `stuck` | `Boolean` | 1 (+1 pad) |

`InitDiedGameOver` (`:249-310`) — `kPageSpacing` = 40, `kPageRightOffset` = 128, `kPageBackUp` = 128
(the latter two are defined at `:252-253` and **never used**):

```
 1  pageSrcRect = (0,0,25,256);  gameOverSrcMap  = 8-bit GWorld;  LoadGraphic(1988)   // :261-264
 2  pageSrcRect = (0,0,32,448);  pageSrcMap      = 8-bit GWorld;  LoadGraphic(1990)   // :266-269
 3                               pageMaskMap     = 1-bit GWorld;  LoadGraphic(1989)   // :271-273
 4  for i in 0..13:  pageSrc[i] = (0,0,32,32) offset (0, 32i)                          // :275-279
 5  for i in 0..7:                                                                     // :281-298
 6      pages[i].dest = (0,0,32,32);  CenterRectInRect(&dest, &thisMac.screen)
 7      QOffsetRect(&dest, -screen.left, -screen.top)
 8      if (i < 4) QOffsetRect(&dest, -40*(4-i), 0) else QOffsetRect(&dest, 40*(i-3), 0)
 9      QOffsetRect(&dest, -RectWide(&screen)/2, -RectWide(&screen)/2)   // <-- width used twice
10      if (dest.left % 2 == 1) QOffsetRect(&dest, 1, 0)
11      pages[i].was = dest;  frame = 0;  counter = RandomInt(32);  stuck = false
12  for i in 0..7:  lettersSrc[i] = (0,0,25,32) offset (0, 32i)                        // :300-304
13  roomRgn = NewRgn();  RectRgn(roomRgn, &justRoomsRect)                              // :306-307
14  pagesStuck = 0
15  stopPages = RectTall(&screen)/2 - 16                                               // :309
```

Two things to preserve exactly:

- **Step 9 uses the screen *width* for both the horizontal and the vertical offset.** On a
  640 x 480 screen that is `(-320, -320)`, so the papers start 320 px left of and 320 px above the
  centre — off the top-left of the window. On a 1152 x 870 screen it is `(-576, -576)`, which is
  576 px above a centre that is only 425 px from the top, i.e. 151 px above the window. This is
  clearly a copy-paste bug (the vertical should use `RectTall`) but it is load-bearing for the
  animation: the papers blow *down and to the right* at 45 degrees, so they need to start up-left.
- Step 10 forces every `dest.left` even. `CopyBits`/`CopyMask` on 68k QuickDraw was substantially
  faster on even (word-aligned) boundaries at 8 bpp; the parity fix-up is a performance hack with a
  visible 1-px consequence, and it is applied only once at init, not maintained during the
  animation (all the offsets in `HandlePages` are even, so parity is preserved).

`stopPages` = half the *screen* height minus 16 is the y at which a paper sticks. `roomRgn` is a
region equal to `justRoomsRect` (2.3) and is used to clip the letters (see below).

`HandlePages` (`:316-394`) — pure state machine, no drawing:

```
for i in 0..7:
    if (pages[i].dest.bottom + RandomInt(8)) > stopPages:      // :322
        pages[i].frame = 0
        if !stuck:  dest.right = dest.left + 25;  stuck = true;  pagesStuck++    // :327-329
    else if frame == 0:
        counter--;  if (counter <= 0) frame = 1                 // :336-338
    else if frame == 7:
        counter--
        if (counter <= 0):  frame = 8;  play kPaper3Sound or kPaper4Sound
        else:               QOffsetRect(&dest, 10, 10)          // :352
    else:
        frame++
        switch (frame):
            5:  QOffsetRect(&dest, 6, 6)
            6:  QOffsetRect(&dest, 8, 8)
            7:  QOffsetRect(&dest, 8, 8);  counter = RandomInt(4) + 4
            8:  QOffsetRect(&dest, 8, 8)
            9:  QOffsetRect(&dest, 8, 8)
           10:  QOffsetRect(&dest, 6, 6)
           14 (kPageFrames):  QOffsetRect(&dest, 8, 0);  frame = 0;
                              counter = RandomInt(8) + 8;
                              play kPaper1Sound or kPaper2Sound
```

Frames 1-4 and 11-13 advance the frame counter without moving, so the sheet flutters in place;
frames 5-10 carry it 6-8 px diagonally per frame; frame 14 wraps to 0 with a purely horizontal 8-px
step. Note the randomised threshold at `:322` (`+ RandomInt(8)`) means the stick line is fuzzy by
up to 8 px per sheet per frame, and the `dest.right = dest.left + 25` at `:327` narrows the sheet
from 32 to 25 px wide — the letter glyph width.

`DrawPages` (`:401-435`):

```
for i in 0..7:
    if pages[i].stuck:
        CopyBits(gameOverSrcMap -> workSrcMap, &lettersSrc[i], &pages[i].dest,
                 srcCopy, roomRgn)                              // :409-412
    else:
        CopyMask(pageSrcMap, pageMaskMap -> workSrcMap,
                 &pageSrc[frame], &pageSrc[frame], &pages[i].dest)   // :416-421
    QUnionSimilarRect(&pages[i].dest, &pages[i].was, &pages[i].was)  // :424
    AddRectToWorkRects(&pages[i].was)
    AddRectToBackRects(&pages[i].dest)
    CopyRectsQD();  numWork2Main = 0;  numBack2Work = 0          // :428-431
    pages[i].was = pages[i].dest
```

- The stuck letters are blitted with `srcCopy` and **`roomRgn` as the mask region**, so a letter is
  clipped to `justRoomsRect` (the 3x3 viewport, or the central room when `numNeighbors == 1`).
  A letter that lands outside the room area is invisible. This is the only use of a region mask in
  `CopyBits` outside `UpdateMainWindow` (1.x) and `WipeScreenOn` (14.x).
- `srcCopy` (not `CopyMask`) means the letter's own white background is drawn opaque — the letters
  are stamped as 25 x 32 white-on-black tiles, not masked glyphs.
- `CopyRectsQD()` is called **inside** the per-sheet loop (`:428`), so the 8 sheets are published
  to the screen in 8 separate blit batches per frame. That is 8x the QuickDraw overhead of the game
  loop but it means the counters can never exceed 2 entries.
- `lettersSrc[i]` is indexed by the *sheet* index `i`, not by position, so the letters always spell
  the same string left to right only because `pages[i].dest` was laid out left to right in
  `InitDiedGameOver` step 8.

`DoDiedGameOver` (`:443-506`):

```
451  InitDiedGameOver()
452  CopyRectMainToWork(&workSrcRect)     // seed workSrcMap from the frozen screen
453  CopyRectMainToBack(&workSrcRect)     // and backSrcMap too
454  FlushEvents(everyEvent, 0)
456  nextLoop = TickCount() + 2
457  while (pagesStuck < 8):
459      HandlePages();  DrawPages()
461      busy-wait to nextLoop, polling GetKeys/GetNextEvent; any modifier or event
             sets pagesStuck = 8 and userAborted = true
478      nextLoop = TickCount() + 2
481  DisposeRgn(roomRgn)
484  DisposeGWorld(pageSrcMap);  pageSrcMap = nil
487  DisposeGWorld(pageMaskMap); pageMaskMap = nil
490  DisposeGWorld(gameOverSrcMap); gameOverSrcMap = nil
492  playing = false
494  if demoGoing:  if (!userAborted) WaitForInputEvent(1)
499  else:          if (!userAborted) WaitForInputEvent(10);  TestHighScore()
505  RedrawSplashScreen()
```

Lines 452-453 are the trick that makes this work over the live game screen: both offscreen buffers
are re-seeded *from the window*, so the frozen last frame of gameplay (including the scoreboard
strip, since `workSrcRect` covers the whole window) becomes the background the papers blow across
and the source the dirty-rect restore reads from. Unlike `DoGameOver` there is no new background
graphic at all.

The three GWorlds here are the only ones in the program that are created and disposed per use
rather than at startup (`angelSrcMap`/`angelMaskMap` are permanent; `pageSrcMap`/`pageMaskMap`/
`gameOverSrcMap` are transient).

### 15.9 Summary of full-screen graphic assets

| PICT | Size | Ver | Bytes | Used by | Scaled to |
|---|---|---|---|---|---|
| 1000 | 640 x 460 | v2 | 108482 | splash | 1:1, or 640x460 at `splashOrigin` |
| 1011 | 360 x 216 | v2 | 48728 | (see 5.x; not used by these paths) | - |
| 1017 | 256 x 64 | v2 | 9770 | `DisplayStarsRemaining`, plural | 1:1 |
| 1018 | 256 x 64 | v2 | 9742 | `DisplayStarsRemaining`, singular | 1:1 |
| 1019 / 1020 | 96 x 44 | v2 / v1 | 4064 / 457 | game-over angel + mask | 1:1 |
| 1021 | 640 x 460 | v2 | 51520 | `SetUpFinalScreen` milky way | 1:1, centred |
| 1988 | 25 x 256 | v2 | 5900 | "GAME OVER" letters, 8 x (25 x 32) | 1:1 |
| 1989 / 1990 | 32 x 448 | v1 / v2 | 1851 / 10052 | paper mask / paper, 14 x (32 x 32) | 1:1 |
| 1991 / 1992 | 330 x 30 | v1 / v2 | 477 / 9360 | banner page bottom mask / art | 1:1 |
| 1993 | 330 x 190 | v2 | 43936 | banner page top | 1:1 |
| 1994 / 1998 | 332 x 30 | v2 / v1 | 4972 / 1049 | "High Scores" plaque / mask | 1:1 |
| 1995 | 640 x **460** | v2 | 20746 | high-score star field | **stretched to 640 x 480** |

(all sizes and byte counts measured from the resource fork, `/tmp/wf-render/pictframes.txt`)

---

## 16. The marquee (editor selection feedback)

`GliderPRO/Sources/Marquee.c` (511 lines). This is the only part of the renderer that draws
*directly into the window with the QuickDraw pen* rather than blitting from an offscreen map, and the
only part that uses XOR modes to achieve reversible drawing. It runs in edit mode only; nothing in
`Play.c`'s loop touches it.

### 16.1 The `marquee` struct

`GliderPRO/Headers/Marquee.h:14-20`:

| Field | Type | Size | Notes |
|---|---|---|---|
| `pats[7]` | `Pattern[kNumMarqueePats]` | 7 x 8 = 56 bytes | `Pattern` is 8 bytes = an 8x8 1-bit tile |
| `bounds` | `Rect` | 8 | the framed selection rect, **window-local** coordinates |
| `handle` | `Rect` | 8 | the 9x9 drag handle |
| `index` | `short` | 2 | which of the 7 patterns is currently on screen |
| `direction` | `short` | 2 | `kAbove`/`kToRight`/`kBelow`/`kToLeft`/`kBottomCorner`/`kTopCorner` |
| `dist` | `short` | 2 | handle distance in pixels |
| `active` | `Boolean` | 1 | marquee exists |
| `paused` | `Boolean` | 1 | temporarily removed from screen |
| `handled` | `Boolean` | 1 (+1 pad) | draw the handle and its tether line |

Total 82 bytes (83 with trailing pad). `kNumMarqueePats` = 7 (`GliderDefines.h:460`),
`kHandleSideLong` = 9 (`Marquee.c:16`), `kMarqueePatListID` = 128 (`Marquee.c:15`).

Direction codes (`GliderDefines.h:210-215`): `kAbove` 1, `kToRight` 2, `kBelow` 3, `kToLeft` 4,
`kBottomCorner` 5, `kTopCorner` 6.

### 16.2 The 7 patterns — measured bytes

`InitMarquee` (`Marquee.c:499-510`) loads them with
`GetIndPattern(&theMarquee.pats[i], kMarqueePatListID, i + 1)` — note the resource index is 1-based
while the array index is 0-based.

Parsed straight out of the resource fork (`PAT#` 128, 58 bytes: a 2-byte count of 7 followed by
7 x 8 pattern bytes; `/tmp/wf-render/py/pats.py`):

| Resource index | Array index | Bytes (rows 0..7, MSB = leftmost pixel) |
|---|---|---|
| 1 | 0 | `F8 F1 E3 C7 8F 1F 3E 7C` |
| 2 | 1 | `3E 7C F8 F1 E3 C7 8F 1F` |
| 3 | 2 | `1F 3E 7C F8 F1 E3 C7 8F` |
| 4 | 3 | `8F 1F 3E 7C F8 F1 E3 C7` |
| 5 | 4 | `C7 8F 1F 3E 7C F8 F1 E3` |
| 6 | 5 | `E3 C7 8F 1F 3E 7C F8 F1` |
| 7 | 6 | `F1 E3 C7 8F 1F 3E 7C F8` |

Pattern 1 rendered as a bit grid (`#` = 1 = pen colour, `.` = 0 = background):

```
row 0  F8  #####...
row 1  F1  ####...#
row 2  E3  ###...##
row 3  C7  ##...###
row 4  8F  #...####
row 5  1F  ...#####
row 6  3E  ..#####.
row 7  7C  .#####..
```

Structure: a 5-on / 3-off diagonal stripe. Each row is the row above **rotated left by 1 bit**, so
the stripes run down-and-to-the-left ("/" diagonals) with a period of 8 px in both axes.

All seven patterns are vertical rotations of pattern 1. Expressing pattern *k* as
`pat_k[i] = pat_1[(i + 8 - r_k) mod 8]`, the measured rotation amounts `r_k` are:

| k | 1 | 2 | 3 | 4 | 5 | 6 | 7 |
|---|---|---|---|---|---|---|---|
| `r_k` (rows shifted down) | 0 | 2 | 3 | 4 | 5 | 6 | 7 |

**Rotation 1 is missing.** There are 8 distinct rotations of an 8-row pattern but only 7 slots in the
`PAT#` list, and the one that was dropped is the `r = 1` phase (`7C F8 F1 E3 C7 8F 1F 3E`, which
appears nowhere in the resource). Consequently the animation advances by 1 row per frame for six of
its seven steps and by **2 rows** for the step from index 0 to index 1 — the ants visibly stutter
once per 7-frame cycle. This is an artwork bug, empirically confirmed from the resource bytes, and a
Go port reproducing the resource faithfully will inherit it. (Generating all 8 rotations
procedurally is a 1-line fix if the port prefers smooth ants.)

QuickDraw aligns patterns to the *port* origin, not to the shape being filled, so the phase of the
stripe within `bounds` depends on `bounds`' absolute position. A Go port must index the pattern as
`pat[y mod 8]` bit `(7 - (x mod 8))` in window coordinates, not in rect-local coordinates.

### 16.3 The XOR animation cycle

`DoMarquee` (`Marquee.c:35-50`) is called once per event-loop pass in edit mode:

```
1  if (!theMarquee.active) or theMarquee.paused -> return          // :37-38
2  SetPortWindowPort(mainWindow)                                    // :40
3  PenMode(patXor)                                                  // :41
4  PenPat(&theMarquee.pats[theMarquee.index]);  DrawMarquee()       // :42-43   erase old
5  theMarquee.index++;  if (index >= 7) index = 0                   // :44-46
6  PenPat(&theMarquee.pats[theMarquee.index]);  DrawMarquee()       // :47-48   draw new
7  PenNormal()                                                      // :49
```

`patXor` inverts the destination pixel everywhere the pattern bit is 1, so drawing twice with the
*same* pattern is a no-op and drawing once with the old pattern then once with the new one leaves
exactly the new pattern's pixels inverted. There is no save/restore of the underlying image and no
dirty rect: the marquee lives only in the window's own pixels, and `workSrcMap` never contains it.

That has three consequences a port must handle:

1. Anything that repaints the window from `workSrcMap` (e.g. `UpdateMainWindow`) silently destroys
   the marquee's XOR state; the editor therefore calls `PauseMarquee` / `ResumeMarquee`
   (`Marquee.c:161-184`) around every such operation.
2. `StopMarquee` (`:139-157`) removes the marquee by drawing it one more time with the *current*
   index, relying on XOR self-inversion. If the index has changed since the last draw, the marquee
   is not erased but doubled.
3. On an 8-bit indexed display QuickDraw's XOR transfer modes operate on *pixel values* (palette
   indices), not on RGB. So the marquee colour at a given pixel is `CLUT[oldIndex XOR mask]` where
   `mask` is derived from the pen's foreground colour, and it therefore varies wildly with whatever
   the marquee happens to be drawn over. The exact `mask` value for a black foreground on an 8-bit
   port is not determinable from the Glider source alone (it is QuickDraw-internal) — see
   "Open questions". A Go port that XORs RGB components instead of indices will produce visibly
   different colours; the structurally faithful choice is to XOR the index and then look up the CLUT.

`InitMarquee` also sets `index = 0`, `active/paused/handled = false`, `gliderMarqueeUp = false`.

### 16.4 `DrawMarquee` — what actually gets stroked

`Marquee.c:455-495`:

```
 1  FrameRect(&theMarquee.bounds)                          // :457  1-px pen, current pattern
 2  if theMarquee.handled:
 3      PaintRect(&theMarquee.handle)                      // :460  solid 9x9 in the pattern
 4      switch (theMarquee.direction):
 5        kAbove:   MoveTo(handle.left + 4, handle.bottom)
 6                  LineTo(handle.left + 4, bounds.top - 1)          // :464-467
 7        kToRight: MoveTo(handle.left, handle.top + 4)
 8                  LineTo(bounds.right, handle.top + 4)             // :471-474
 9        kBelow:   MoveTo(handle.left + 4, handle.top - 1)
10                  LineTo(handle.left + 4, bounds.bottom)           // :478-481
11        kToLeft:  MoveTo(handle.right, handle.top + 4)
12                  LineTo(bounds.left, handle.top + 4)              // :485-488
13  if gliderMarqueeUp:  DrawGliderMarquee()               // :493-494
```

`kHandleSideLong / 2` = 4 (integer division of 9), so the tether line leaves the handle from
pixel 4 of 0..8 — one pixel left/above of true centre. `kBottomCorner` and `kTopCorner` draw the
handle but **no tether line** (they fall through the switch), which is correct: a corner handle is
attached to the rect corner already.

### 16.5 Handle placement

`StartMarqueeHandled(theRect, direction, dist)` — `Marquee.c:76-135`:

```
 1  handle = (0,0,9,9);  QOffsetRect(&handle, -4, -4)      // :89-90   centred on origin
 2  switch (direction):
 3    kAbove:        offset to (bounds.left, bounds.top),    then (+HalfRectWide, -dist)
 4    kToRight:      offset to (bounds.right, bounds.top),   then (+dist, +HalfRectTall)
 5    kBelow:        offset to (bounds.left, bounds.bottom), then (+HalfRectWide, +dist)
 6    kToLeft:       offset to (bounds.left, bounds.top),    then (-dist, +HalfRectTall)
 7    kBottomCorner: offset to (bounds.right, bounds.bottom)
 8    kTopCorner:    offset to (bounds.right, bounds.top)
 9  theMarquee.direction = direction;  theMarquee.dist = dist
10  PenMode(patXor);  PenPat(pats[index]);  DrawMarquee();  PenNormal()
11  SetCoordinateHVD(bounds.left, bounds.top, dist)
```

`kHandleSideLong / -2` is `-4` (C integer division truncates toward zero for 9/-2... in fact
`9 / -2 == -4` in C99 and `-5` under some pre-C89 compilers; MPW C truncates toward zero, giving
`-4`). The handle is therefore centred on the anchor point with 4 px above/left and 4 px below/right.

`HalfRectWide`/`HalfRectTall` are in `GliderPRO/Sources/RectUtils.c` and return `(right-left)/2` and
`(bottom-top)/2`.

### 16.6 The rubber-band drags

Four interactive drag loops, all of which follow the same "XOR-erase, move, XOR-draw" idiom inside a
`while (WaitMouseUp())` poll loop with `DeltaPoint` change detection:

| Function | Lines | Cursor | What moves | Coordinate feedback |
|---|---|---|---|---|
| `DragOutMarqueeRect(start, theRect)` | :188-214 | arrow (`InitCursor`) | a `FrameRect` from `start` to the mouse, `NormalizeRect`d each step | none |
| `DragMarqueeRect(start, theRect, lockH, lockV)` | :218-255 | `handCursor` | `bounds` translated; `lockV` zeroes `deltaH`, `lockH` zeroes `deltaV` | `SetCoordinateHVD(bounds.left, bounds.top, -2)` |
| `DragMarqueeHandle(start, dragged)` | :259-341 | `vertCursor` for kAbove/kBelow, else `horiCursor` | only `handle`; `*dragged` is clamped at 0 and the delta corrected so the handle cannot pass the anchor | `DeltaCoordinateD(*dragged)` |
| `DragMarqueeCorner(start, hDragged, vDragged, isTop)` | :345-403 | `diagCursor` | `handle` plus `bounds.right` and `bounds.top`-or-`bounds.bottom` | none |

Sign conventions in `DragMarqueeHandle` (`:280-329`) — note that "dragged" always means "bigger
distance", so two of the four cases negate the mouse delta:

| direction | `deltaH` | `deltaV` | `*dragged` update |
|---|---|---|---|
| `kAbove` | 0 | `newPt.v - wasPt.v` | `-= deltaV` |
| `kToRight` | `newPt.h - wasPt.h` | 0 | `+= deltaH` |
| `kBelow` | 0 | `newPt.v - wasPt.v` | `+= deltaV` |
| `kToLeft` | `newPt.h - wasPt.h` | 0 | `-= deltaH` |

and in every case `if (*dragged <= 0) { delta -/+= *dragged; *dragged = 0; }` — the correction is
applied with the *opposite* sign to the accumulate, so the handle stops dead at distance 0.

`DragMarqueeCorner` (`:363-393`) inverts the vertical delta when `isTop` (`deltaV = wasPt.v -
newPt.v`) and then applies `QOffsetRect(&handle, deltaH, -deltaV)` with `bounds.top -= deltaV`;
otherwise `QOffsetRect(&handle, deltaH, deltaV)` with `bounds.bottom += deltaV`. `bounds.right +=
deltaH` in both cases, so a corner drag always grows the rect to the right and never to the left.

Each loop finishes with one last `FrameRect`/`PaintRect` pair (`:212`, `:251`, `:337-338`,
`:399-400`) to XOR the outline away, then `PenNormal()` and `InitCursor()`.

### 16.7 The "glider marquee" — XOR sprite preview

`DrawGliderMarquee` (`Marquee.c:432-439`) is the one place in the program that does a **`srcXor`
`CopyBits` from a 1-bit map to the window**:

```
CopyBits((BitMap *)*GetGWorldPixMap(blowerMaskMap),
         GetPortBitMapForCopyBits(GetWindowPort(mainWindow)),
         &leftStartGliderSrc,      // src
         &marqueeGliderRect,       // dst
         srcXor, nil);
```

`leftStartGliderSrc` = `(0, 358, 48, 374)` — a 48 x 16 cell at y 358 in the blower sprite sheet
(`GliderPRO/Sources/StructuresInit.c:280-281`); its sibling `rightStartGliderSrc` =
`(0, 374, 48, 390)` (`:283-284`) is used by the editor's own start-point drawing
(`ObjectEdit.c:2719`) but not by the marquee. The source map is `blowerMaskMap`, the **1-bit mask**
GWorld, so what gets XORed is the glider *silhouette*.

`SetMarqueeGliderRect(h, v)` (`Marquee.c:443-451`):

```
marqueeGliderRect = leftStartGliderSrc;  ZeroRectCorner(&marqueeGliderRect);
QOffsetRect(&marqueeGliderRect, h - kHalfGliderWide, v - kGliderHigh);
DrawGliderMarquee();  gliderMarqueeUp = true;
```

`kHalfGliderWide` = 24, `kGliderHigh` = 20 (`GliderDefines.h:549-551`), but the cell is only **16**
px tall, so the rect spans `v-20 .. v-5` — the silhouette floats 4 px above the nominated point
rather than sitting on it. That is intentional (the point is the glider's feet and the artwork has
4 px of transparent margin at the bottom of the 20-px cell) but it means a port must use 20 for the
offset and 16 for the height.

`gliderMarqueeUp` is a separate flag from `theMarquee.active`: `DrawMarquee` re-XORs the glider on
every animation frame (`:493-494`), so the silhouette flickers in step with the ants but is *not*
patterned — it is a solid XOR of the mask. `StopMarquee` (`:141-145`) XORs it away first, before the
`active` check, so it is cleared even when no rect marquee is up.

`srcXor` from a 1-bit source to an 8-bit destination is a QuickDraw behaviour a Go port must emulate
explicitly: for each source bit that is 1, the destination *pixel value* (palette index) is XORed
with a mask derived from the port's foreground colour; source bits that are 0 leave the destination
alone. The important structural facts (which are all that the Glider source pins down) are that the
operation is an involution — calling `DrawGliderMarquee` twice restores the original pixels exactly,
which the code relies on at `Marquee.c:143` and `:494` — and that it reads the *mask* GWorld, so the
shape is a silhouette.

---
## 17. The coordinate windoid (`Coordinates.c`)

`GliderPRO/Sources/Coordinates.c`, 196 lines, read in full. A tiny always-on-top palette window that
shows the selected object's position and its handle distance while editing. Entirely wrapped in
`#ifndef COMPILEDEMO`, so it vanishes from the demo build. It is rendering-relevant only because it
is a second `GrafPort` that the editor draws text into, and because it demonstrates the
save/restore-port discipline that the scoreboard code (13.11) omits.

### 17.1 Globals and geometry

`Coordinates.c:16-20`:

| Global | Type | Meaning |
|---|---|---|
| `coordWindowRect` | `Rect` | always `(0, 0, 38, 50)` — 50 wide x 38 tall |
| `coordWindow` | `WindowPtr` | nil when closed |
| `isCoordH`, `isCoordV` | `short` | saved global top-left, persisted in prefs |
| `coordH`, `coordV`, `coordD` | `short` | the three displayed values; `-1` means "show a dash" |
| `isCoordOpen` | `Boolean` | menu-checkmark state |

`kWindoidWDEF` = 2048 (`GliderDefines.h:531`) is the window definition proc ID, i.e. resource
`WDEF` 128 with variation 0 (`2048 = 128 * 16`). The window is titled `"Tools"` (yes, the same title
as the real tools palette — `Coordinates.c:128`, `:131`) and is created with `NewCWindow` when
`thisMac.hasColor` and plain `NewWindow` otherwise (`:126-131`).

Default position when there is no prefs file: `isCoordH = 50`, `isCoordV = 204`
(`GliderPRO/Sources/Main.c:171-172`); otherwise `thePrefs.wasCoordH` / `wasCoordV`
(`Main.c:100-101`), written back at quit (`Main.c:259-260`). `Events.c:94` updates them with
`GetWindowLeftTop` whenever the user drags the windoid. The commented-out alternative at
`Coordinates.c:138-139` / `Main.c:170` would have put it at `screenBits.bounds.right - 55, 204`.

### 17.2 `UpdateCoordWindow` — the drawing

`Coordinates.c:61-111`:

```
 1  if (coordWindow == nil) return                          // :67-68
 2  GetPort(&wasPort);  SetPort((GrafPtr)coordWindow)        // :70-71
 3  EraseRect(&coordWindowRect)                             // :72
 4  tempStr = "h: " + (coordH != -1 ? NumToString(coordH) : "-")
 5  MoveTo(5, 12);   DrawString(tempStr)                    // :82-83
 6  tempStr = "v: " + (coordV != -1 ? NumToString(coordV) : "-")
 7  MoveTo(4, 22);   DrawString(tempStr)                    // :93-94
 8  ForeColor(blueColor)                                    // :96
 9  tempStr = "d: " + (coordD != -1 ? NumToString(coordD) : "-")
10  MoveTo(5, 32);   DrawString(tempStr)                    // :105-106
11  ForeColor(blackColor)                                   // :107
12  SetPort((GrafPtr)wasPort)                               // :109
```

Baselines at y 12, 22, 32 — 10 px line spacing in a 38-px-tall window. Left margins are 5, **4**,
5: the `v:` line is one pixel further left than the other two, which is either a typo or a manual
optical correction for the narrower `v` glyph. The `d:` line is drawn in `blueColor` (QuickDraw's
`blueColor` constant, not a palette index) and the foreground is restored to black afterwards.

Unlike the scoreboard helpers, this function *does* save and restore the port (`GetPort` at `:70`,
`SetPort` at `:109`), so it is safe to call from anywhere.

### 17.3 The font bug

`OpenCoordWindow` sets the text state once, at `Coordinates.c:154-155`:

```
154      TextFace(applFont);
155      TextSize(9);
```

`TextFace` takes a *style* bitmask, not a font number. `applFont` is 1
(`<Fonts.h>`: `applFont = 1`), and style bit 0 is `bold`, so line 154 is equivalent to
`TextFace(bold)` and the intended `TextFont(applFont)` never happens. The windoid therefore renders
in the **system font (Chicago), bold, 9 pt** rather than the application font (Geneva) at 9 pt.
Compare the correct idiom used everywhere else, e.g. `Scoreboard.c:140-142`,
`HighScores.c:133-135`, `Banner.c:128-130`, which all call `TextFont(applFont)` *and*
`TextFace(bold)` *and* `TextSize(n)`.

Also note the text state is set on the *coordinate window's* port at open time and never re-set;
because QuickDraw text state is per-`GrafPort` this survives, but any code that opens a new port with
the same `WindowPtr` would lose it.

### 17.4 Value plumbing

| Function | Lines | Behaviour |
|---|---|---|
| `SetCoordinateHVD(h, v, d)` | :29-40 | assigns each of `coordH`/`coordV`/`coordD` **unless the argument is `-2`**, then calls `UpdateCoordWindow` |
| `DeltaCoordinateD(d)` | :49-55 | sets `coordD` only |
| `OpenCoordWindow` | :116-167 | creates the window, `MoveWindow(isCoordH, isCoordV)`, `BringToFront`, `ShowHide(true)`, `HiliteAllWindows()`, initialises all three values to `-1`, then if an object is selected fills them from `theMarquee.bounds` and `ObjectHasHandle` |
| `CloseCoordWindow` | :172-176 | `CloseThisWindow(&coordWindow)` + uncheck menu |
| `ToggleCoordinateWindow` | :181-195 | open/close on `coordWindow == nil` |

Two magic values overlap in the same parameter: `-1` means "display a dash" and `-2` means "leave
unchanged". `DragMarqueeRect` uses `SetCoordinateHVD(left, top, -2)` (`Marquee.c:248`) to update
position without touching the distance; `StopMarquee` uses `SetCoordinateHVD(-1, -1, -1)`
(`Marquee.c:156`) to blank all three. So a real coordinate of `-1` or `-2` (possible in room space,
since object coordinates can be negative) would be misinterpreted — a latent bug, harmless because
the editor clamps object positions to the room.

`OpenCoordWindow` also builds two rects `src` and `dest` (`:143-145`) around the global mouse and the
window, presumably for a zoom-rects animation that was removed; they are never used. `MyGetGlobalMouse`
is called for the same dead purpose.

---

## 18. The map window and its room thumbnails (`Map.c`)

`GliderPRO/Sources/Map.c`, 796 lines. An editor-only floating window that draws the whole house as a
grid of 32 x 20 room thumbnails. Rendering-relevant because it is the only other consumer of the
background PICTs, it introduces a second offscreen GWorld (`nailSrcMap`), and it is the one place in
the program that calls `SetOrigin`.

### 18.1 Constants

| Constant | Value | Definition |
|---|---|---|
| `kMapRoomsHigh` | 9 (comment: "was 7") | `Map.c:17` |
| `kMapRoomsWide` | 9 (comment: "was 7") | `Map.c:18` |
| `kMapScrollBarWidth` | 16 | `Map.c:19` |
| `kHScrollRef` | 5L | `Map.c:20` |
| `kVScrollRef` | 27L | `Map.c:21` |
| `kMapGroundValue` | 56 | `Map.c:22` |
| `kNewRoomAlert` | 1004 | `Map.c:23` |
| `kYesDoNewRoom` | 1 | `Map.c:24` |
| `kThumbnailPictID` | 1010 | `Map.c:25` |
| `kMapRoomWidth` | 32 | `GliderDefines.h:247` |
| `kMapRoomHeight` | 20 | `GliderDefines.h:246` |
| `kMaxNumRoomsH` | 128 | `GliderDefines.h:543` |
| `kMaxNumRoomsV` | 64 | `GliderDefines.h:544` |
| `kNumBackgrounds` | 18 | `GliderDefines.h:521` |
| `kBaseBackgroundID` | 2000 | `GliderDefines.h:519` |
| `kWindoidGrowWDEF` | 2064 (= `WDEF` 129, variation 0) | `GliderDefines.h:532` |

`kMapRoomsHigh` / `kMapRoomsWide` are dead: the live values are the globals `mapRoomsHigh` /
`mapRoomsWide`, loaded from prefs (`Main.c:94-95`) or defaulted to **15 wide x 4 high**
(`Main.c:163-164`) — note the default is nothing like 9 x 9. `isMapH` / `isMapV` default to
`3, 100` (`Main.c:160-162`).

The thumbnail aspect ratio is 32 : 20 = 1.6, while a room is 512 x 322 = 1.5901 — close but not
exact, so thumbnails are very slightly horizontally stretched relative to the room.

### 18.2 Room-grid <-> map-cell coordinate transform

A room's identity in the house is `(suite, floor)`; the map's cell index is `(h, i)`:

```
h = suite - mapLeftRoom
i = (kMapGroundValue - floor) - mapTopRoom          // kMapGroundValue == 56
```

and the inverse used in both draw loops (`Map.c:138-139`, `:223-224`):

```
suite = h + mapLeftRoom
floor = kMapGroundValue - (i + mapTopRoom)
```

So **floor 56 is the top row of the scroll range** and floor numbers *decrease* as you go down the
map — i.e. `floor` is an altitude, and 56 is the ceiling of the addressable house. The ground line
within the visible map is at cell row `groundLevel = kMapGroundValue - mapTopRoom` (`Map.c:198`);
cells at `i >= groundLevel` are below ground (floor <= 0).

Cell rect in window coordinates (`Map.c:220-221`):

```
aRoom = (0, 0, kMapRoomWidth, kMapRoomHeight) offset by (32*h, 20*i)
```

### 18.3 `SetOrigin(1, 1)` — the 1-pixel port shift

`OpenMapWindow` does `SetPort((GrafPtr)mapWindow); SetOrigin(1, 1);` (`Map.c:397-398`). From then on
the map window's port has its origin at (1,1), i.e. **every drawing coordinate in the window is
offset by -1 in both axes** relative to the content rect. That is why:

- the scroll bar rects start at `-1` (`Map.c:399`, `:402`) — they need to overhang the top/left by
  one pixel to sit flush with the frame;
- `mapWindowRect` is `mapRoomsWide*32 + 16 - 2` wide (`Map.c:338-340`, `:371-373`) — the `-2`
  recovers the 1 px lost at each edge;
- the clip rect in `RedrawMapContents` is `mapWindowRect` with `right`/`bottom` **`+ 2 -
  kMapScrollBarWidth`** (`Map.c:200-203`), again compensating;
- `mapCenterRect` (the grow box) is `(-16,-16,0,0)` offset to `(mapWindowRect.right + 2,
  mapWindowRect.bottom + 2)` (`Map.c:415-417`).

For a 15 x 4 default map: `mapWindowRect` = `(0, 0, 15*32+16-2, 4*20+16-2)` = **(0, 0, 494, 94)**;
the thumbnail area is 480 x 80 at port coordinates (0,0)-(480,80); the h-scroll bar is at
`(-1, 80, 481, 96)`; the v-scroll bar at `(480, -1, 496, 81)`.

A Go port has no `SetOrigin`; the clean equivalent is to render the map into a surface whose (0,0) is
the thumbnail grid's top-left and to place the scroll bars by explicit arithmetic.

### 18.4 `nailSrcMap` — the thumbnail sheet

`CreateNailOffscreen` (`Map.c:733-750`):

```
if (nailSrcMap == nil):
    GetGWorld(&wasCPort, &wasWorld)
    QSetRect(&nailSrcRect, 0, 0, kMapRoomWidth, kMapRoomHeight * (kNumBackgrounds + 1))
    CreateOffScreenGWorld(&nailSrcMap, &nailSrcRect, kPreferredDepth)   // depth 8
    SetGWorld(nailSrcMap, nil)
    LoadGraphic(kThumbnailPictID)                                        // PICT 1010
    SetGWorld(wasCPort, wasWorld)
```

`nailSrcRect` = `(0, 0, 32, 20 * 19)` = **32 x 380**. Measured: PICT 1010 is 32 x 380 v2, 6850 bytes
(`/tmp/wf-render/pictframes.txt`) — an exact match, so the sheet is 19 vertically stacked 32 x 20
cells: indices 0..17 for the 18 real backgrounds and index 18 for the "?" placeholder.

`KillNailOffscreen` disposes it and nils the pointer. The GWorld exists only while the map window is
open.

Cell source rect (`Map.c:241-242`):

```
src = (0, 0, 32, 20) offset by (0, type * 20)
```

where `type = (*thisHouse)->rooms[whoCares].background - kBaseBackgroundID` (`Map.c:228`), i.e.
PICT 2000 -> type 0, ..., PICT 2017 -> type 17. `kNumBackgrounds` is 18, so any `type > 18` means a
*custom* background PICT (ID >= 2019) that has no thumbnail in the sheet.

### 18.5 `RedrawMapContents` — the draw order

`Map.c:185-302`.

```
 1  if (mapWindow == nil) return                                            // :194-195
 2  activeRoomVisible = false;  groundLevel = 56 - mapTopRoom               // :197-198
 3  newClip = mapWindowRect with right/bottom += 2 - 16                     // :200-203
 4  SetPort((GrafPtr)mapWindow)                                            // :205
 5  wasClip = NewRgn();  GetClip(wasClip);  ClipRect(&newClip)              // :206-211
 6  HLock(thisHouse)                                                       // :213-214
 7  for i in 0 .. mapRoomsHigh-1:                                          // :216
 8    for h in 0 .. mapRoomsWide-1:                                        // :218
 9      aRoom = (0,0,32,20) offset (32h, 20i)                              // :220-221
10      suite = h + mapLeftRoom;  floor = 56 - (i + mapTopRoom)            // :223-224
11      if RoomExists(suite, floor, &whoCares) and houseUnlocked:          // :225
12          PenNormal();  type = rooms[whoCares].background - 2000         // :227-228
13          if (type > 18) and !doPrettyMap:  type = 18   // the "?" cell  // :229-233
14          ForeColor(blackColor)                                          // :234
15          if (type > 18):  LoadGraphicPlus(type + 2000, &aRoom)          // :235-238
16          else:            CopyBits(nailSrcMap -> mapWindow port,
17                                    &src, &aRoom, srcCopy, nil)         // :241-246
18          if (whoCares == thisRoomNumber):
19              activeRoomRect = aRoom;  activeRoomVisible = true          // :248-252
20      else:                                                              // :254
21          PenPat(GetQDGlobalsGray(&dummyPat))                            // :258
22          ForeColor(i >= groundLevel ? greenColor : blueColor)           // :259-262
23          PaintRect(&aRoom)                                              // :263
24  HSetState(thisHouse)                                                   // :268
25  ForeColor(blackColor);  PenNormal()                                    // :270-271
26  for i in 1 .. mapRoomsWide-1:  MoveTo(32i, 0);  Line(0, 20*mapRoomsHigh)   // :273-277
27  for i in 1 .. mapRoomsHigh-1:  MoveTo(0, 20i);  Line(32*mapRoomsWide, 0)   // :279-283
28  if activeRoomVisible:                                                  // :285
29      ForeColor(redColor)
30      activeRoomRect.right++;  activeRoomRect.bottom++                   // :288-289
31      FrameRect(&activeRoomRect)                                         // :290
32      InsetRect(&activeRoomRect, 1, 1);  FrameRect(&activeRoomRect)      // :291-292
33      ForeColor(blackColor);  InsetRect(&activeRoomRect, -1, -1)         // :293-294
34  SetClip(wasClip);  DisposeRgn(wasClip)                                 // :297-301
```

Details worth transcribing exactly:

- **Empty cells are a 50% gray pattern in green or blue.** `GetQDGlobalsGray` returns the classic
  `AA 55 AA 55 AA 55 AA 55` checkerboard; `PaintRect` with that pen pattern and `greenColor` gives a
  green-on-white 50% dither below ground and blue-on-white above. There is no PICT involved.
- **The grid lines are drawn on top of every thumbnail**, 1 px black, at every internal cell
  boundary. They are drawn *after* all cells, so a thumbnail never covers a line.
- **The active-room highlight is a 2-px red double frame**, drawn as `FrameRect` then `InsetRect(1,1)`
  then `FrameRect` again on a rect whose `right`/`bottom` have been incremented by 1 first. So the
  outer frame is at `(32h-0, 20i-0)` .. `(32h+33, 20i+21)` in port coordinates: it overlaps the
  neighbouring cells by 1 px, which is deliberate — it draws *on* the grid lines.
- `activeRoomRect` is mutated in place by the highlight code (`right++`, `bottom++`,
  `InsetRect(1,1)`, `InsetRect(-1,-1)`), leaving it 1 px bigger than the cell on exit. That value is
  what `FlagMapRoomsForUpdate` (`Map.c:101-109`) later feeds to `InvalWindowRect`, so the
  invalidation is correctly 1 px oversized.
- `FindNewActiveRoomRect` (`Map.c:115-160`) is the same double loop with the drawing removed; it
  saves the old rect into `wasActiveRoomRect` and applies `right++`, `bottom++`, `InsetRect(-1,-1)`
  — i.e. a rect 2 px larger on the right/bottom and 1 px on the left/top, again for invalidation.
  Note the asymmetry between the two functions' final insets (`+1,+1` then `-1,-1` in one,
  `-1,-1` only in the other).
- `RoomExists(suite, floor, &whoCares)` returns the room index in `whoCares`; the whole grid is
  suppressed when `!houseUnlocked`, in which case every cell draws as an empty gray cell.

### 18.6 "Pretty map" mode and `LoadGraphicPlus`

`doPrettyMap` is a saved preference, default **false** (`Main.c:187`), settable in the Settings
dialog (`Settings.c:1230` sets it true; `:246`/`:276` save-and-restore it). Its only effect is at
`Map.c:229-238`:

| `type` | `doPrettyMap` | Result |
|---|---|---|
| 0..18 | either | `CopyBits` cell `type` from `nailSrcMap` |
| > 18 (custom PICT) | false | `type` forced to 18 -> the "?" cell |
| > 18 (custom PICT) | true | `LoadGraphicPlus(type + 2000, &aRoom)` — the **full-size custom room PICT scaled down into 32 x 20** |

Pretty mode is therefore extremely slow (it does a `GetPicture` + `DrawPicture` + `ReleaseResource`
per visible custom room per redraw, with QuickDraw scaling a 512 x 322 image down to 32 x 20) which
is presumably why it defaults off.

`LoadGraphicPlus` (`Map.c:165-180`) is a variant of `LoadScaledGraphic` (4.3) that falls back to the
`'Date'` resource type when `GetPicture` fails:

```
1  thePicture = GetPicture(resID)
2  if nil:  thePicture = (PicHandle)GetResource('Date', resID)
3  if still nil: return
4  DrawPicture(thePicture, theRect)
5  ReleaseResource((Handle)thePicture)
```

`'Date'` is Glider PRO's own private resource type for house-supplied artwork (see 4.2 and 5.x): a
house file stores its custom backgrounds as `'Date'` resources so they do not collide with the
application's `PICT`s, and every graphic loader in the program tries `PICT` first and `'Date'`
second. The bytes are in PICT format either way.

### 18.7 Window sizing

`ResizeMapWindow(newH, newV)` — `Map.c:325-358`:

```
 1  if (newH == 0 and newV == 0) return
 2  SetPortWindowPort(mapWindow)
 3  mapRoomsWide = newH / 32;  if (< 3) = 3                          // :332-334
 4  mapRoomsHigh = newV / 20;  if (< 3) = 3                          // :335-337
 5  mapWindowRect = (0, 0, 32*wide + 14, 20*high + 14)               // :338-340
 6  EraseRect(&mapWindowRect);  SizeWindow(...)                      // :341-342
 7  SetControlMaximum(mapHScroll, 128 - mapRoomsWide)                // :344
 8  MoveControl(mapHScroll, 0, mapWindowRect.bottom - 14)            // :345
 9  SizeControl(mapHScroll, mapWindowRect.right - 13, 16)            // :346-347
10  mapLeftRoom = GetControlValue(mapHScroll)                        // :348
11  SetControlMaximum(mapVScroll, 64 - mapRoomsHigh)                 // :350
12  MoveControl(mapVScroll, mapWindowRect.right - 14, 0)             // :351
13  SizeControl(mapVScroll, 16, mapWindowRect.bottom - 13)           // :352-353
14  mapTopRoom = GetControlValue(mapVScroll)                         // :354
15  InvalWindowRect(mapWindow, &mapWindowRect)                       // :356
```

The window is quantised to whole cells with a minimum of 3 x 3 cells (112 x 74 px). Scroll ranges are
`0 .. 128 - mapRoomsWide` horizontally and `0 .. 64 - mapRoomsHigh` vertically, matching
`kMaxNumRoomsH`/`kMaxNumRoomsV`.

`CenterMapOnRoom(h, v)` (`Map.c:72-95`) clamps `mapLeftRoom` to `[0, 128 - mapRoomsWide]` and
`mapTopRoom` to `[0, 64 - mapRoomsHigh]` and then pushes the values into the scroll bars.

`UpdateMapWindow` (`Map.c:307-321`) is the update-event handler: `SetControlValue` x2,
`SetPortWindowPort`, `DrawControls`, `DrawGrowIcon`, `RedrawMapContents`. Note it does **not**
`BeginUpdate`/`EndUpdate` — the caller does.

---

## 19. QuickTime movie compositing (the TV set)

`COMPILEQT` is defined unconditionally at `GliderPRO/Headers/GliderDefines.h:15`, so the shipped
1.0.4 build includes it. All movie code is additionally guarded at run time by `thisMac.hasQT`.

This is the only content in the entire game that is **not** drawn by the Glider renderer: QuickTime
draws it, directly into the on-screen window, behind the renderer's back.

### 19.1 The house movie

One movie per house, loaded by `OpenHouseMovie` (`GliderPRO/Sources/HouseIO.c:67-142`):

```
 1  theSpec = theHousesSpecs[thisHouseIndex];  theSpec.name += ".mov"      // :80-81
 2  if FSpGetFInfo fails -> return silently (no movie for this house)      // :83-85
 3  OpenMovieFile(&theSpec, &movieRefNum, fsCurPerm)                       // :87
 4  NewMovieFromFile(&theMovie, movieRefNum, nil, name, newMovieActive, &changed)  // :94-95
 5  CloseMovieFile(movieRefNum)                                           // :102
 6  spaceSaver = NewHandle(307200L)      // 300 KB reserve, immediately freed  // :104
 7  GoToBeginningOfMovie(theMovie)                                         // :112
 8  LoadMovieIntoRam(theMovie, 0, GetMovieDuration(theMovie), 0)           // :113-114
 9  DisposeHandle(spaceSaver)                                             // :122
10  PrerollMovie(theMovie, 0, 0x000F0000)   // rate 0x000F0000 = 15.0 fixed  // :124
11  theTime = GetMovieTimeBase(theMovie);  SetTimeBaseFlags(theTime, loopTimeBase)
12  SetMovieMasterTimeBase(theMovie, theTime, nil);  LoopMovie()           // :132-135
13  GetMovieBox(theMovie, &movieRect)                                      // :137
14  hasMovie = true                                                       // :139
```

Notable:

- The movie lives in the **data fork** of a file named `<house>.mov` next to the house file. This is
  why `GliderPRO/Houses/` contains both `*.binhex` (resource-fork house data) and `*.mov` files.
- `307200` = 640 x 480 bytes — a one-screen-at-8bpp scratch allocation made and freed purely to
  guarantee the memory is available *after* the movie is loaded into RAM.
- `PrerollMovie(theMovie, 0, 0x000F0000)`: the third argument is a `Fixed` rate, `0x000F0000` =
  15.0. That is a preroll hint, not the playback rate.
- `LoopMovie` (`HouseIO.c:48-63`) installs an empty `'LOOP'` user-data item (after removing any
  existing ones) which is QuickTime's flag for "loop forever"; combined with
  `SetTimeBaseFlags(theTime, loopTimeBase)` the movie loops seamlessly.
- `movieRect` is the movie's natural box, captured once and then repeatedly recentred (19.3).

`CloseHouseMovie` (`:146-159`) flushes it from RAM (`LoadMovieIntoRam(..., flushFromRam)`) and
`DisposeMovie`s it; `hasMovie = false` unconditionally.

### 19.2 State variables

| Variable | Declared | Meaning |
|---|---|---|
| `theMovie` | `Movie`, `HouseIO.c:31` | the one loaded movie |
| `movieRect` | `Rect`, `HouseIO.c:32` | current movie box, in **global screen** coordinates |
| `hasMovie` | `Boolean`, `HouseIO.c:36` | a `.mov` exists and loaded |
| `tvInRoom` | `Boolean`, `HouseIO.c:36` | the *current central room* contains the movie-bearing TV |
| `tvWithMovieNumber` | `short`, `Objects.c:77` | index into `dinahs[]` of that TV, or -1 |
| `tvOn` | `Boolean`, `Play.c:54` | the TV's power state |
| `thisMac.hasQT` | `Boolean` | QuickTime present |

`tvInRoom = false; tvWithMovieNumber = -1;` is the reset, done in `NilSavedMaps`
(`RoomGraphics.c:59-60`), `ReadyLevel`'s QT block (`RoomGraphics.c:409-410`) and
`HouseIO.c:197-198`.

Only the **first** TV in the central room gets the movie: the guard is
`(neighbor == kCentralRoom) && (!tvInRoom)` (`ObjectDrawAll.c:680-681`), and `tvInRoom` is set true
immediately afterwards (`:708`).

### 19.3 The screen rect and clipping

The TV sprite's two screen states live in the appliance sheet
(`GliderPRO/Sources/StructuresInit.c:570-573`):

| Rect | Value | Content |
|---|---|---|
| `tvScreen1` | `(0, 171, 64, 220)` — 64 x 49 | screen **off** |
| `tvScreen2` | `(0, 220, 64, 269)` — 64 x 49 | screen **on** (static/noise) |

The screen's position inside the TV object rect is a fixed `(+17, +10)` offset, hard-coded in three
places (`ObjectDraw2.c:676`, `ObjectDrawAll.c:685`, and via `dinahs[].dest` in `Dynamics3.c:265`):

```
bounds = tvScreen1;  ZeroRectCorner(&bounds);            // -> (0,0,64,49)
QOffsetRect(&bounds, theRect->left + 17, theRect->top + 10);
```

`ObjectDrawAll.c:679-694`, executed while `DrawLocale` is building the room, does the QuickTime
placement:

```
 1  whoCares = tvScreen1;  ZeroRectCorner(&whoCares)                       // :683-684
 2  OffsetRect(&whoCares, itsRect.left + 17, itsRect.top + 10)             // :685
 3  GetMovieBox(theMovie, &movieRect)                                      // :686
 4  CenterRectInRect(&movieRect, &whoCares)                                // :687
 5  SetMovieBox(theMovie, &movieRect)                                      // :688
 6  theRgn = NewRgn();  RectRgn(theRgn, &whoCares)                         // :689-690
 7  SetMovieDisplayClipRgn(theMovie, theRgn);  DisposeRgn(theRgn)          // :691-692
 8  tvOn = thisObject.data.g.state                                         // :693
```

- `itsRect` here has already been through `OffsetRectRoomRelative(&itsRect, neighbor)`
  (`ObjectDrawAll.c:676`), which converts room coordinates to `backSrcMap`/window coordinates by
  adding `localRoomsDest[neighbor]`'s origin — i.e. it already includes `playOriginH/V`. Since
  `SetMovieGWorld` binds the movie to `mainWindow` (19.4) and a `WindowPtr`'s port coordinates are
  window-local, the movie box is in **window** coordinates. This is the one place the QuickDraw
  window origin and the offscreen composition origin have to agree exactly.
- The movie is **centred in the 64 x 49 screen rect, not scaled to it** (`CenterRectInRect` at
  step 4, no `SetMovieBox` resize). A movie larger than 64 x 49 is centred and then clipped by the
  display clip region at step 7; a smaller one is centred with the TV's static image visible around
  it.
- The clip region is exactly the 64 x 49 screen rect, so the movie can never spill onto the room.

### 19.4 Binding the movie to the window

`Play.c:147-152`, in `PlayGame`'s setup, right after the black menu-bar strip is painted:

```
147  #ifdef COMPILEQT
148      if ((thisMac.hasQT) && (hasMovie))
150          SetMovieGWorld(theMovie, (CGrafPtr)mainWindow, nil);
152  #endif
```

The movie is drawn into the **on-screen window**, never into `workSrcMap`. Consequences that a Go
port must reproduce structurally even though it will not use QuickTime:

1. The movie's pixels are not in `workSrcMap`, so the dirty-rect machinery (section 9) does not know
   about them. Anything that blits `workSrcMap` over the TV screen rect erases the movie frame until
   the next `MoviesTask`.
2. Conversely the movie is drawn *after* the frame's `CopyRectsQD`, so it always appears on top of
   whatever the renderer just published — including the glider if the glider is in front of the TV.
   The glider therefore disappears behind a playing movie.
3. `HandleTV` (`Dynamics.c:428-478`) explicitly **skips** its own screen blits when this TV is the
   movie TV: both the `timer == 0` `AddRectToWorkRects` and the `timer == 1` `CopyBits(tvScreen2)`
   are inside empty `if` bodies with the real work in the `else`
   (`Dynamics.c:437-444`, `:449-460`). That is how the static "on" image is suppressed so the movie
   is not overdrawn. Note the asymmetry: turning the TV **off** always does the blit
   (`:465-475`, no QT guard), because the off image must replace the last movie frame.

### 19.5 Per-frame servicing

`MoviesTask(theMovie, 0)` is called once per game frame, immediately before `RenderFrame()`, in both
branches of the main loop:

| Site | Condition |
|---|---|
| `Play.c:465-468` (two-player branch) | `thisMac.hasQT && hasMovie && tvInRoom && tvOn` |
| `Play.c:490-493` (one-player branch) | same |

So the order within a frame is: dynamics -> input -> interaction -> **MoviesTask** -> `RenderFrame`
-> `HandleDynamicScoreboard`. Because `MoviesTask` draws into the window and `RenderFrame`'s
`CopyRectsQD` also draws into the window, the movie frame is drawn *before* the renderer's blits in
program order — so a dirty rect overlapping the TV screen will overwrite the movie for that frame.
In practice the TV screen rect is only dirty when something moves across it.

### 19.6 Start / stop points

| Event | Code | Action |
|---|---|---|
| game start | `Play.c:201-211` | `SetMovieActive(theMovie, true)`; if `tvOn` then `StartMovie` + `MoviesTask(theMovie, 0)` |
| game end | `Play.c:223-230` | `tvInRoom = false; StopMovie; SetMovieActive(false)` |
| room change (`ReadyLevel`) | `RoomGraphics.c:406-413` | `tvInRoom = false; tvWithMovieNumber = -1; StopMovie` |
| after a room transition | `Transit.c:302-308`, `:340-346`, `:379-385`, `:418-424` | if `tvInRoom && tvOn`: `GoToBeginningOfMovie` + `StartMovie`, then `RenderFrame()` |
| player flips the TV switch | `Trip.c:43-55` | on: `GoToBeginningOfMovie` + `StartMovie` + `tvOn = true`; off: `StopMovie` + `tvOn = false` |

`ReadyLevel` stops the movie *before* `DrawLocale` re-scans the new room, and `DrawLocale`'s TV case
re-arms it if the new central room has a TV. That is why `tvInRoom` must be cleared first: the
`!tvInRoom` guard at `ObjectDrawAll.c:681` would otherwise refuse to bind the new room's TV.

### 19.7 What a Go port must do instead

There is no QuickTime. The structurally equivalent design:

1. Decode the `.mov` (or a re-encoded equivalent) to a sequence of frames at load time, with the
   movie's natural size recorded as `movieRect`.
2. Keep a `tvInRoom` / `tvWithMovieNumber` / `tvOn` triple exactly as the C does, set from the same
   `DrawLocale` scan with the same `neighbor == kCentralRoom && !tvInRoom` first-only rule.
3. Compute the destination as `centre(movieRect within (objRect.left+17, objRect.top+10, +64, +49))`
   and clip to that 64 x 49 rect.
4. Draw the current movie frame **into `workSrcMap` after the object layer but before the glider**,
   and add the 64 x 49 rect to `work2MainRects` every frame the movie is running. This is *not* what
   the original does — the original draws to the screen after everything — but it is the only way to
   get correct layering without a second compositing pass, and it removes the "glider hidden behind
   the TV" artifact. If bug-for-bug fidelity is wanted, draw it to the final framebuffer after the
   dirty-rect blits instead.
5. Advance the movie clock from the frame counter, not from a wall clock, so the movie stays in sync
   with the 2-tick game frame; loop at the end.

---

## 20. Frame pacing and the game loop

The renderer has no concept of elapsed time, interpolation or variable time step. Everything —
sprite animation, physics, the scoreboard blink — is driven off an integer frame counter, and the
frame counter is throttled by a single busy-wait against the Mac `TickCount()` (1/60 s) clock.

### 20.1 The pacing constant

| Constant | Value | Definition | Meaning |
|---|---|---|---|
| `kTicksPerFrame` | 2 | `GliderPRO/Headers/GliderDefines.h:533` | ticks (1/60 s) per rendered frame |
| `kIdleSplashTicks` | 7200L | `GliderPRO/Headers/GliderDefines.h:197` | 2 minutes of idle before the auto-demo starts |

`TickCount()` on classic Mac OS increments 60 times per second (nominally 60.15 Hz), so
`kTicksPerFrame == 2` means the target frame rate is **30 frames per second** and the nominal frame
budget is **33.3 ms**.

Everything derived from that:

| Quantity | Value |
|---|---|
| Target frame rate | 60 / 2 = **30 fps** |
| Frame budget | 2 ticks ≈ 33.3 ms |
| Frames per second of game time | 30 |
| Frames per in-game "delay unit" | `(delay * 6) / kTicksPerFrame` = `delay * 3` (see 20.6) |

### 20.2 The two pacing variables

```c
long        gameFrame;        // GliderPRO/Sources/Play.c:51
Boolean     evenFrame;        // GliderPRO/Sources/Play.c:53
long        nextFrame;        // GliderPRO/Sources/Render.c:40
```

| Variable | Reset where | Meaning |
|---|---|---|
| `gameFrame` | `= 0L` in `NewGame` (`Play.c:112`) | monotonically increasing frame index for the current game |
| `evenFrame` | `= false` in `InterfaceInit` (`InterfaceInit.c:131`) | frame parity — but see 20.5, it gets clobbered |
| `nextFrame` | `= TickCount() + kTicksPerFrame` in `InitGarbageRects` (`Render.c:690`) | tick deadline for the frame currently being built |

### 20.3 The busy-wait, in `RenderFrame`

`GliderPRO/Sources/Render.c:639-671`. The wait is *between* the compositing work and the blit to
screen, which is unusual and deliberate:

```
 1  if (hasMirror):  DrawReflection(&theGlider, true)
 2                   if (twoPlayerGame) DrawReflection(&theGlider2, false)      // :641-646
 3  HandleGrease()                                                             // :647
 4  RenderPendulums()                                                          // :648
 5  if (evenFrame) RenderFlames() else RenderStars()                           // :649-652
 6  RenderDynamics()                                                           // :653
 7  RenderFlyingPoints()                                                       // :654
 8  RenderSparkles()                                                           // :655
 9  RenderGlider(&theGlider, true)                                             // :656
10  if (twoPlayerGame) RenderGlider(&theGlider2, false)                        // :657-658
11  RenderShreds()                                                             // :659
12  RenderBands()                                                              // :660
13  while (TickCount() < nextFrame) { }        // <-- spin                     // :662-664
14  nextFrame = TickCount() + kTicksPerFrame                                   // :665
15  CopyRectsQD()                                                              // :667
16  numWork2Main = 0;  numBack2Work = 0                                        // :669-670
```

Three properties of this arrangement matter for a port:

1. **The spin happens before the screen blit, not after.** All the offscreen compositing for frame
   *n* is done, then the process sleeps (spins) until the deadline, then the pixels are published.
   That minimises the delay between "spin finished" and "pixels visible", so the visible frame rate
   is as even as the tick clock allows even though the compositing time varies wildly with how much
   is on screen.
2. **`nextFrame` is recomputed from `TickCount()` *after* the wait, not by adding to the old
   `nextFrame`.** This is a *non-accumulating* frame limiter: a frame that overruns its budget is
   simply late, and the next deadline is set 2 ticks from *now*, not 2 ticks from the missed
   deadline. There is no catch-up and no frame dropping. Consequence: **on a slow machine the entire
   game runs in slow motion** — animation and physics stay in lockstep with rendering because they
   share the frame counter. On a fast machine it runs at exactly 30 fps.
3. **`CopyRectsQD` is the only thing after the wait**, and the dirty-rect queues are emptied
   immediately after it (`numWork2Main = 0; numBack2Work = 0`). Anything that adds a rect between
   the queue reset and the next `RenderFrame` is picked up by the next frame.

`CopyRectsQD` (`Render.c:616-635`) does the two queues in this order:

```
for i in 0 .. numWork2Main-1:  CopyBits(workSrcMap -> mainWindow port,
                                        &work2MainRects[i], &work2MainRects[i], srcCopy, nil)
for i in 0 .. numBack2Work-1:  CopyBits(backSrcMap  -> workSrcMap,
                                        &back2WorkRects[i], &back2WorkRects[i], srcCopy, nil)
```

so the *publish* pass runs first and the *erase-for-next-frame* pass second. See section 9 for why.

### 20.4 `PlayGame` — the whole loop

`GliderPRO/Sources/Play.c:430-599`. The two-player and one-player branches are identical except for
the second `GetInput`/`HandleGlider` and the `demoGoing` test, so this transcribes the one-player
branch (`:473-497`) and marks the two-player differences:

```
while (playing && !quitting):                                          // :432
   1  gameFrame++                                                      // :434
   2  evenFrame = !evenFrame                                           // :435
   3  if (doBackground):                                               // :437
          do { HandlePlayEvent(); } while (switchedOut);                // :439-442
   4  HandleTelephone()                                                // :445
   5  HandleDynamics()                                                 // :475     (2P: :449)
   6  if (!gameOver):
          if (demoGoing) GetDemoInput(&theGlider)                      // :478-479
          else           GetInput(&theGlider)                          // :481
          [2P: GetInput(&theGlider); GetInput(&theGlider2)              // :452-453 ]
          HandleInteraction()                                          // :482     (2P: :454)
   7  HandleTriggers()                                                 // :484     (2P: :456)
   8  HandleBands()                                                    // :485     (2P: :457)
   9  if (!gameOver):  HandleGlider(&theGlider)                        // :486-487
          [2P: HandleGlider(&theGlider); HandleGlider(&theGlider2)      // :460-461 ]
  10  if (playing):                                                    // :488     (2P: :463)
          #ifdef COMPILEQT
          if (hasQT && hasMovie && tvInRoom && tvOn) MoviesTask(theMovie, 0)   // :490-493
          #endif
          RenderFrame()                                                // :494     (2P: :469)
          HandleDynamicScoreboard()                                    // :495     (2P: :470)
  11  if (gameOver):  <game-over countdown, section 20.7>              // :499
```

Order facts a port must preserve exactly:

- **Dynamics run before input.** `HandleDynamics` moves every dynamic object (and, importantly,
  queues its dirty rects) *before* the player's velocity is read this frame, so a collision resolved
  in `HandleInteraction` sees the object at its new position.
- **`HandleTriggers` and `HandleBands` run even when `gameOver`.** Only input, interaction and glider
  physics are suppressed; the room keeps animating during the game-over countdown.
- **`RenderFrame` is the last thing in the frame** apart from the scoreboard, and it is skipped
  entirely if `playing` went false during this frame (a room transition or death sets it).
- **`HandleDynamicScoreboard` runs after the frame is published**, so a score change is visible one
  frame later than the event that caused it.

### 20.5 `evenFrame` is not reliably frame parity

`evenFrame` is toggled once per loop at `Play.c:435`, so nominally `evenFrame == (gameFrame odd)`.
But it is a **global that other code writes**:

| Site | Code | When |
|---|---|---|
| `Dynamics2.c:420` | `dinahs[who].moving = true; evenFrame = true;` | every time a **ball** starts moving |
| `Dynamics3.c:474` | `evenFrame = true;` | in `AddDynamicObject` `case kBall:` — dead assignment, `lilFrame` is used instead |
| `Dynamics3.c:524` | `evenFrame = true;` | in `AddDynamicObject` `case kFish:` — same |

The two `Dynamics3.c` writes are leftovers: the loop below them (`:476-483`, `:526-533`) reverse-
engineers the initial velocity from the object's `length`/`height` field using a *local* `lilFrame`
toggle, and `evenFrame` is never read. But they still clobber the global at room-load time. The
`Dynamics2.c:420` write clobbers it mid-frame during play.

Everything that reads `evenFrame` therefore has its phase reset whenever a ball starts moving:

| Reader | Effect of `evenFrame` |
|---|---|
| `Render.c:649-652` | `RenderFlames()` on even frames, `RenderStars()` on odd frames — the two most expensive renderers are *interleaved*, each running at 15 fps |
| `Dynamics.c:60-65` | tin-foil is consumed one unit per *two* frames while being shoved |
| `Dynamics.c:327-332` | toast/bread animation advances one frame per two game frames |
| `Dynamics2.c:38-43`, `:77-82` | balloon rise/fall animation advances every other frame (frames 0..5 rising, 6..7 falling) |
| `Dynamics2.c:409-410`, `:471-472`, `:551-552` | **gravity**: `vVel++` only on even frames, i.e. ball/drip/fish acceleration is 0.5 px/frame² |
| `Dynamics2.c:433-434` | drip animation `frame = 9 - frame` (a 2-frame flip) every other frame |
| `Interactions.c:1756-1761` | spider-web twang applies its impulse on even frames only |

The most visible consequence is the flames/stars interleave: **candle flames animate at 15 fps and
star sparkles at 15 fps, on alternating frames**, and if a ball starts rolling the two can end up on
the same parity for a while, so one of them stalls until the parity flips back. A Go port should keep
the interleave (it is a real performance decision, halving the per-frame cost of the two heaviest
renderers) but should use `gameFrame & 1` instead of a writable global, and drop the three stray
`evenFrame = true` assignments — with a note that this changes the flame/star phase.

### 20.6 Object delays are expressed in ticks and converted at spawn time

`AddDynamicObject` converts every author-specified delay into frames once, at room load:

```c
dinahs[numDynamics].count = ((short)who->data.h.delay * 6) / kTicksPerFrame;
```

at `Dynamics3.c:316` (`data.g.delay`, greasy/toast), `:401`, `:426`, `:456`, `:504` (`data.h.delay`)
and `Dynamics3.c:521` (fish `hVel`). With `kTicksPerFrame == 2` that is **`delay * 3` frames**, i.e.
one delay unit = 6 ticks = 1/10 second. A Go port that changes the frame rate must keep this formula
in terms of the frame rate, not hard-code `* 3`, or every timed object in every house drifts.

### 20.7 The scoreboard's 8-frame round robin

`HandleDynamicScoreboard` (`GliderPRO/Sources/Scoreboard.c:72-132`) uses the low three bits of
`gameFrame` as a time-division slot allocator so that at most one of the three "low warning" blinks
is redrawn per frame:

```c
whosTurn = gameFrame & 0x00000007;      // Scoreboard.c:95
```

| `whosTurn` | Action | Condition | Line |
|---|---|---|---|
| 0 | `QuickFoilRefresh(false)` — **show** foil | `0 < foilTotal < kFoilLow (2)` | :98-101 |
| 1 | `QuickBatteryRefresh(true)` — **hide** battery | `0 < batteryTotal < kBatteryLow (17)` or `kHeliumLow (-38) < batteryTotal < 0` | :103-108 |
| 2 | `QuickBandsRefresh(false)` — **show** bands | `0 < bandsTotal < kBandsLow (2)` | :110-113 |
| 3 | — (nothing) | | |
| 4 | `QuickBatteryRefresh(false)` — **show** battery | same as slot 1 | :115-120 |
| 5 | `QuickFoilRefresh(true)` — **hide** foil | same as slot 0 | :122-125 |
| 6 | — (nothing) | | |
| 7 | `QuickBandsRefresh(true)` — **hide** bands | same as slot 2 | :127-130 |

So the blink period is 8 game frames = 16 ticks ≈ 267 ms, and the duty cycles are deliberately
staggered and unequal:

| Meter | Shown at slot | Hidden at slot | Visible for | Hidden for |
|---|---|---|---|---|
| foil | 0 | 5 | 5 frames | 3 frames |
| battery | 4 | 1 | 5 frames | 3 frames (phase-shifted by 4) |
| bands | 2 | 7 | 5 frames | 3 frames (phase-shifted by 2) |

The three meters therefore never blink in unison — they are 4, 2 and 0 frames out of phase — which is
what makes a low-everything scoreboard look busy rather than strobing. The "low" thresholds are
local `#define`s inside the function (`Scoreboard.c:74-77`), each annotated `// 25%`:

| Threshold | Value | Comment |
|---|---|---|
| `kFoilLow` | 2 | 25 % |
| `kBatteryLow` | 17 | 25 % |
| `kHeliumLow` | -38 | 25 % (helium is negative battery) |
| `kBandsLow` | 2 | 25 % |

The score roll is also in this function and is *not* rate-limited (`Scoreboard.c:80-93`): while
`theScore > displayedScore`, add `kScoreRollAmount` every frame (or jump straight to the target if
`doRollScore` is false), play `kScoreTikSound`, and `QuickScoreRefresh()`.

### 20.8 Event handling during play, and `doBackground`

`HandlePlayEvent` (`GliderPRO/Sources/Play.c:387-426`):

```
 1  long sleep = 2                                                       // :391
 2  if WaitNextEvent(everyEvent, &theEvent, sleep, nil):                 // :393
 3      if (theEvent.what == updateEvt && message == mainWindow):        // :395-396
 4          GetPort(&wasPort);  SetPortWindowPort(mainWindow)             // :398-399
 5          BeginUpdate(mainWindow)                                      // :400
 6          CopyBits(workSrcMap -> mainWindow port,
                     &justRoomsRect, &justRoomsRect, srcCopy, nil)       // :401-403
 7          RefreshScoreboard(kNormalTitleMode)                           // :404
 8          EndUpdate(mainWindow);  SetPort(wasPort)                      // :405-406
 9      else if (osEvt && (message & 0x01000000)):                       // :408
10          if (message & 0x00000001):   // resume
11              switchedOut = false;  ToggleMusicWhilePlaying();  HideCursor()   // :412-414
12          else:                        // suspend
13              InitCursor();  switchedOut = true;  ToggleMusicWhilePlaying()    // :419-421
```

Key facts:

- **`doBackground` defaults to `false`** (`GliderPRO/Sources/Main.c:186`; the prefs value
  `thePrefs.wasDoBackground` overrides it at `Main.c:118`, and the Settings "defaults" button sets it
  false again at `Settings.c:1228`). So in the shipped configuration `HandlePlayEvent` is **never
  called during play** — the game does not yield to the OS at all inside the loop, does not process
  update events, and cannot be switched out. Input comes from `GetKeys` polling (`Input.c:190`), not
  from the event queue.
- With `doBackground` true, `sleep = 2` gives away exactly one frame's worth of ticks, and the
  `do { HandlePlayEvent(); } while (switchedOut);` loop parks the game entirely while the app is in
  the background.
- The update path is the **only** full-window repaint during play: `justRoomsRect` from `workSrcMap`,
  plus a `RefreshScoreboard`. It does *not* repaint the black menu-bar strip (that was painted once
  at `Play.c:142-145`) and it does not repaint the QuickTime movie (which reappears on the next
  `MoviesTask`).
- `0x01000000` is `osEvt`'s `suspendResumeMessage` tag in the high byte of `message`; bit 0 is the
  resume flag. A Go port maps this to whatever its windowing layer calls focus-lost/focus-gained.

Outside play the loop is `while (!quitting) HandleEvent();` (`Main.c:363-364`), a plain blocking
event loop; the splash-screen idle timer `incrementModeTime = TickCount() + kIdleSplashTicks`
(7200 ticks = 120 s) is reset at `InterfaceInit.c:163`, `Play.c:277`, `:302`, `Events.c:427`,
`Menu.c:336`, `:402`, `:419`, `:424`, `AppleEvents.c:111` and tested at `Events.c:538`.

### 20.9 Other busy-waits in the program

The same `while (TickCount() < deadline);` idiom appears in the two game-over animations, with their
own 2-tick cadence:

| Site | Code | Cadence |
|---|---|---|
| `GameOver.c:149`, `:219-220` | `nextLoop = TickCount() + 2` … `while (TickCount() < nextLoop); nextLoop = TickCount() + 2;` | 30 fps page-turn animation |
| `GameOver.c:456`, `:477-478` | same | 30 fps "died" animation |
| `Utilities.c:446`, `:474` | `timeToBail = TickCount() + 60L * seconds` | timeout, not pacing |

Note these recompute the deadline *after* the wait, exactly like `RenderFrame`, so they are also
non-accumulating.

### 20.10 What a Go port must do

1. Replace `TickCount()` with a monotonic clock, but keep the *integer tick* abstraction: define
   `tick = 1/60 s` and `frameBudget = 2 ticks`. Do not switch to floating-point delta time — every
   animation counter in the game is an integer frame count and the physics constants are tuned for
   exactly 30 fps.
2. Replace the spin loop with a sleep-until-deadline, keeping the **non-accumulating** semantics
   (`deadline = now() + frameBudget` *after* waking). Accumulating deadlines would make the game try
   to catch up after a hitch, which the original never does and which the physics cannot survive.
3. Keep the "composite, then wait, then publish" order. If the port publishes before waiting, input
   latency improves but frame-to-frame spacing gets noisier, which is visible at 30 fps.
4. Derive `evenFrame` from `gameFrame & 1`, and keep the flames/stars interleave.
5. Keep `(delay * 6) / kTicksPerFrame` as a *formula* so a future frame-rate change does not break
   every house's timing.
6. Keep the `gameFrame & 7` scoreboard round robin with its exact slot assignments — the phase
   offsets are what make the low-supply blink read correctly.

---

## 21. Build-time switches that change the rendering

`GliderPRO/Headers/GliderDefines.h:11-16` is the complete switch block, transcribed verbatim:

```c
//#define CREATEDEMODATA
//#define COMPILEDEMO
//#define CAREFULDEBUG
#define COMPILENOCP
#define COMPILEQT
#define BUILD_ARCADE_VERSION		1
```

So the shipped 1.0.4 build has:

| Switch | State | Effect |
|---|---|---|
| `CREATEDEMODATA` | **off** | would record the auto-demo keystroke stream (`Input.c:303`, `:320`, `:333`, `:347`; `StructuresInit2.c:280`) |
| `COMPILEDEMO` | **off** | the crippled demo build: strips the entire editor (see below) |
| `CAREFULDEBUG` | **off** | no references anywhere in `Sources/`; a dead switch |
| `COMPILENOCP` | **on** | "no copy protection": `copyGood = true` unconditionally (`Main.c:306-308`), skipping `ValidInstallation` |
| `COMPILEQT` | **on** | QuickTime TV support (section 19) |
| `BUILD_ARCADE_VERSION` | **1** | kiosk mode — this one *does* change what is drawn |

A Go port should treat `COMPILEDEMO` and `CREATEDEMODATA` as "not present", `COMPILEQT` as "present",
and must decide explicitly about `BUILD_ARCADE_VERSION`.

### 21.1 `BUILD_ARCADE_VERSION` — the five live sites

`grep -n 'BUILD_ARCADE_VERSION' Sources/*.c` gives exactly five conditionals:

| Site | Sense | What it does |
|---|---|---|
| `Play.c:138-140` | `#if !BUILD_ARCADE_VERSION` | body is a commented-out `HideMenuBarOld()`; **no code either way** |
| `Play.c:512-544` | `#if` | blank + rebuild the scoreboard before the game-over screen (21.2) |
| `Play.c:556-598` | `#if` | blank + rebuild the scoreboard when `PlayGame` returns (21.2) |
| `Play.c:806-808` | `#if !` | commented-out `HideMenuBarOld()`; no code either way |
| `Main.c:347-349` | `#if` | commented-out `HideMenuBarOld()`; no code either way |
| `Main.c:366-368` | inside a `/* */` block | commented-out `ShowMenuBarOld()`; dead |
| `Events.c:191-209` | `#if` | arrow keys become kiosk buttons (21.3) |
| `Input.c:192-203` | `#if` | any gameplay key aborts the auto-demo (21.4) |

Only three of these have any code in them at all. The three menu-bar ones are all commented out with
`// TEMP` notes, which is the same Carbon-port casualty as the dissolve functions (section 14.5): the
1.0.4 source has *no working menu-bar hiding at all*, in either build. The window is created 20 px
short and a 20 px black strip is painted at the bottom of the screen instead (`Play.c:142-145`),
so the real menu bar is simply left visible at the top under the `menuWindow` strip.

### 21.2 The arcade scoreboard blank-and-rebuild

Two nearly identical blocks. **First**, at game over, `Play.c:512-541`, inside the `if (gameOver)`
countdown-expired branch and after `HideGlider` + `RefreshScoreboard`:

```
 1  GetGWorld(&wasCPort, &wasWorld)                                       // :507
 2  HideGlider(&theGlider);  RefreshScoreboard(kNormalTitleMode)          // :509-510
 3  SetGWorld(boardSrcMap, nil)                                           // :515
 4  PaintRect(&boardSrcRect)              // blank the strip              // :516
 5  CopyBits(boardSrcMap -> mainWindow port,
              &boardSrcRect, &boardDestRect, srcCopy, 0L)                 // :518-520
 6  hOffset = (boardSrcRect.right >= 640)
                ? (RectWide(&boardSrcRect) - kMaxViewWidth) / 2
                : -576                                                    // :527-530
 7  thePicture = GetPicture(kScoreboardPictID /* 1997 */)                  // :531
 8  if (!thePicture) RedAlert(kErrFailedGraphicLoad)                       // :532-533
 9  bounds = (*thePicture)->picFrame                                       // :535
10  QOffsetRect(&bounds, -bounds.left, -bounds.top)                        // :537
11  QOffsetRect(&bounds, hOffset, 0)                                       // :538
12  DrawPicture(thePicture, &bounds)      // still in boardSrcMap          // :539
13  ReleaseResource((Handle)thePicture)                                    // :540
14  if (mortals < 0) DoDiedGameOver() else DoGameOver()                    // :546-549
15  SetGWorld(wasCPort, wasWorld)                                          // :551
```

This is exactly the `hOffset` computation from `InitScoreboardMap` (`StructuresInit.c:74-90`,
section 13.3): `kMaxViewWidth` = 1536 (`GliderDefines.h:267`), `kScoreboardPictID` = 1997
(`GliderDefines.h:623`). Worked values:

| `boardSrcRect.right` (= screen width) | branch | `hOffset` |
|---|---|---|
| 512 | `< 640` | **-576** |
| 640 | `>= 640` | (640 - 1536) / 2 = **-448** |
| 800 | `>= 640` | (800 - 1536) / 2 = **-368** |
| 832 | `>= 640` | (832 - 1536) / 2 = **-352** |
| 1024 | `>= 640` | (1024 - 1536) / 2 = **-256** |
| 1152 | `>= 640` | (1152 - 1536) / 2 = **-192** |
| 1280 | `>= 640` | (1280 - 1536) / 2 = **-128** |
| 1536 | `>= 640` | **0** |

PICT 1997 is 1536 px wide (measured, section 13.5), so the effect is "centre the 1536-wide strip
artwork on the screen width, clipped". The `-576` for 512-wide screens is a hand-tuned special case
(centring would give (512-1536)/2 = -512, so `-576` shifts 64 px further left to pick a different
region of the artwork).

**Second**, at `Play.c:556-592`, run unconditionally as `PlayGame` returns. Structurally the same but
there is a real difference:

```
 1  GetGWorld(&wasCPort, &wasWorld)                                       // :561
 2  SetGWorld(boardSrcMap, nil)                                           // :563
 3  PaintRect(&boardSrcRect)                                             // :564
 4  CopyBits(boardSrcMap -> mainWindow port, &boardSrcRect, &boardDestRect, srcCopy, 0L)  // :566-568
 5  SetGWorld(wasCPort, wasWorld)         // <-- restored HERE            // :570
 6  { ... hOffset ...; DrawPicture(thePicture, &bounds); ... }            // :573-592
```

The `SetGWorld(wasCPort, wasWorld)` at `:570` closes the first inner block **before** the second
inner block does its `DrawPicture`. So the second block's `DrawPicture` at `:590` draws PICT 1997
into **whatever port was current when `PlayGame` was entered**, not into `boardSrcMap`. In the first
block the restore is at `:551`, *after* the `DrawPicture`, so that one is correct.

That is a genuine bug: `boardSrcMap` is left blank (step 3 painted it, nothing repainted it), and the
1536-wide strip artwork is stamped into some unrelated port at `hOffset` — most likely the caller's
port, i.e. the main window or `workSrcMap`. It is not visible in practice because the very next thing
the caller does is `RestoreEntireGameScreen` (`Play.c:800-820`), which paints the whole screen black
and calls `DrawLocale()` + `RefreshScoreboard()`; and `RefreshScoreboard` rebuilds the strip contents
from the text fields. But a port that reproduces this literally will get a garbage flash.

**Recommendation for a Go port:** implement the *intent* — at game over, clear the scoreboard surface,
publish the cleared strip, then redraw the strip artwork into the scoreboard surface at `hOffset` —
and note the divergence.

Note also that `PaintRect(&boardSrcRect)` at `:516`/`:564` paints in `boardSrcMap`'s **current
foreground colour**, which is whatever was last set in that port. The comment says "paint over the
scoreboard black" and in practice the last `ForeColor` in that port is `blackColor` (from
`RefreshScoreboard`'s text drawing, `Scoreboard.c:152`), so it is black — but it is not explicit.

### 21.3 Arcade key remapping (`Events.c:191-207`)

In the arcade build the four arrow keys become kiosk buttons in the splash-screen key handler:

| Key | Arcade action | Non-arcade action |
|---|---|---|
| left arrow | `DoOptionsMenu(iHighScores)` — show the high-score screen | (the `#else` branch, `Events.c:209+`) |
| right arrow | `DoOptionsMenu(iHelp)` — show help | |
| up arrow | `DoGameMenu(iNewGame)` | |
| down arrow | `DoGameMenu(iNewGame)` — same as up | |

Rendering-relevant only because it means the high-score screen (15.6) and the help screen are
reachable without a menu bar.

### 21.4 Arcade demo abort (`Input.c:192-201`)

`GetDemoInput` normally checks for the command key to abort the auto-demo. In the arcade build it
instead aborts on **any** of the four gameplay keys:

```c
if ((BitTst(&theKeys, thisGlider->leftKey)) || (BitTst(&theKeys, thisGlider->rightKey)) ||
    (BitTst(&theKeys, thisGlider->battKey)) || (BitTst(&theKeys, thisGlider->bandKey)))
{ playing = false;  paused = false; }
```

so touching a control while the attract-mode demo is running drops straight out of it.

### 21.5 `COMPILEDEMO` — what would disappear

Not compiled in 1.0.4, but worth knowing because a large fraction of the rendering code is inside
`#ifndef COMPILEDEMO`. Counted from `grep -c`:

| File | `#ifndef COMPILEDEMO` blocks | Rendering content that would vanish |
|---|---|---|
| `ObjectEdit.c` | 16 | all edit-mode object drawing |
| `Map.c` | 16 | the entire map window (section 18) |
| `Tools.c` | 16 | the tools palette |
| `RoomInfo.c` | 9 | room-info dialog + its preview |
| `Room.c` | 7 | room creation/deletion |
| `Link.c` | 7 | link drawing |
| `Coordinates.c` | 5 | the coordinate windoid (section 17) |
| `MainWindow.c` | 2 (`:278`, `:305`) | the edit-mode window resize paths |
| `Menu.c` | 3 `#ifndef` + 3 `#ifdef` | the whole editor menu |
| `Link.c` 7, `Room.c` 7, `House.c` 5, `Scrap.c` 5, `SelectHouse.c` 5, `HouseLegal.c` 3, `ObjectAdd.c` 2, `Events.c` 2 | | (exact counts from `grep -c`) |

The pattern is uniform: **`COMPILEDEMO` removes the level editor**, and with it the marquee's reason
to exist, the map window, the coordinate windoid and the tools palette. Play mode is untouched.
`HouseIO.c:182`, `:348`, `:381`, `:403`, `:411`, `Main.c:58`, `:124`, `:214`, `Menu.c:328`, `:375`,
`:766`, `Environ.c:678` are the positive `#ifdef COMPILEDEMO` blocks that *add* demo restrictions
(a fixed house, a nag, a smaller memory requirement).

### 21.6 `powerc` — the one PowerPC-conditional drawing

`GliderPRO/Sources/MainWindow.c:82-93`, inside `DrawOnSplash`:

```c
#if defined(powerc) || defined(__powerc)
TextSize(12);  TextFace(0);  TextFont(systemFont);
ForeColor(blackColor);
MoveTo(splashOriginH + 5, splashOriginV + 457);   DrawString("\pPowerPC Native!");
ForeColor(whiteColor);
MoveTo(splashOriginH + 4, splashOriginV + 456);   DrawString("\pPowerPC Native!");
ForeColor(blackColor);
#endif
```

A drop-shadowed "PowerPC Native!" badge at the bottom-left of the splash screen: 12 pt `systemFont`
(Chicago), plain, black at `(splashOriginH+5, splashOriginV+457)` then white at `(+4, +456)` — a
1-pixel up-left offset, so the black copy is the shadow. `splashOriginV + 457` is 3 px above the
460-px splash's bottom edge.

The only other `powerc` reference is `Settings.c:109` (a preferences field) and `Externs.h:385`
(a calling-convention `#ifdef`); neither draws anything.

`Glider PRO 1.0.4` shipped as a fat binary, so on a Power Mac this string was visible. A Go port
should either drop it or make it a build-info string.

### 21.7 Switch summary for a port

| Behaviour | Recommendation |
|---|---|
| Copy protection (`COMPILENOCP`) | omit entirely |
| Demo build (`COMPILEDEMO`) | omit; build the full editor or none |
| Demo recording (`CREATEDEMODATA`) | omit; the recorded demo data is a resource, see `Input.c:224` `gameFrame == demoData[demoIndex].frame` |
| QuickTime (`COMPILEQT`) | keep as a runtime capability flag, section 19.7 |
| Arcade (`BUILD_ARCADE_VERSION`) | keep as a **runtime** flag: it only affects the arrow-key bindings, the demo abort condition, and the game-over scoreboard clear |
| Menu-bar hiding | there is none in 1.0.4; do not try to reconstruct it |
| `powerc` badge | drop |

---

## 22. Colour depth, the palette, and the disabled gray-to-colour fade

### 22.1 The game never installs a palette

This is the single most important colour fact for a port, and it is a negative: **there is no live
`SetEntries`, `NewPalette`, `SetPalette`, `ActivatePalette` or `PmForeColor` anywhere in
`GliderPRO/Sources/`.** The complete result of
`grep -n 'SetEntries\|NewPalette\|SetPalette\|ActivatePalette\|PmForeColor\|RestoreDeviceClut' Sources/*.c`
is:

| Hit | Status |
|---|---|
| `MainWindow.c:496` (`SetEntries` in `SetPaletteToGrays`) | inside `/* */` — dead |
| `MainWindow.c:586`, `:591` (`SetEntries` in `WashColorIn`) | inside `/* */` — dead |
| `ColorUtils.c:220` (`RestoreDeviceClut(nil)` in `RestoreColorsSlam`) | live function, but its only callers are the two dead ones |

So Glider PRO renders entirely against **whatever CLUT the device already has**, which on a Mac at
8 bits is the Apple standard 256-colour system palette. Every PICT in the resource fork is indexed
into that palette, and every hard-coded `k8...Color` constant in `ObjectDraw.c` / `ObjectDraw2.c` /
`Scoreboard.c` is an index into it.

A Go port must therefore **bake that CLUT in as data**. Fortunately the app carries a copy.

### 22.2 `clut` 128 and `clut` 129 are byte-identical

Measured from the resource fork:

| Resource | Size | `ctSeed` | `ctFlags` | `ctSize` | Entries |
|---|---|---|---|---|---|
| `clut` 128 | 2056 bytes | `0x00000000` | `0x0000` | 255 | 256 |
| `clut` 129 | 2056 bytes | `0x00000000` | `0x0000` | 255 | 256 |

Layout is the standard `ColorTable`: `long ctSeed; short ctFlags; short ctSize;` then `ctSize + 1`
`ColorSpec` records of `short value; RGBColor rgb` (three big-endian 16-bit channels) = 8 bytes each.
2056 = 8 + 256 × 8 ✓.

`clut` 128 and `clut` 129 differ in **zero** of their 256 entries (verified by direct comparison of
the parsed tables). Both are the Apple standard 8-bit system palette; the app simply ships two
copies. Neither is a grayscale table.

The first six entries and the last three, as observed:

| Index | 16-bit RGB | 8-bit `#RRGGBB` |
|---|---|---|
| 0 | `FFFF FFFF FFFF` | `#FFFFFF` |
| 1 | `FFFF FFFF CCCC` | `#FFFFCC` |
| 2 | `FFFF FFFF 9999` | `#FFFF99` |
| 3 | `FFFF FFFF 6666` | `#FFFF66` |
| 4 | `FFFF FFFF 3333` | `#FFFF33` |
| 5 | `FFFF FFFF 0000` | `#FFFF00` |
| … | | |
| 253 | `2222 2222 2222` | `#222222` |
| 254 | `1111 1111 1111` | `#111111` |
| 255 | `0000 0000 0000` | `#000000` |

Every channel byte is duplicated into both halves of the 16-bit word (`0xCC` → `0xCCCC`), so the
8-bit truncation `r >> 8` is exact and lossless. The full 256-entry table is in 5.6.

Structure recap (proved in 5.6): indices 0..214 are a 6 × 6 × 6 colour cube in the order
`red descending, green descending, blue descending` with levels `{FF, CC, 99, 66, 33, 00}`; 215..244
are ten 3-entry ramps of pure red, green, blue; 245..255 are an 11-entry gray ramp; index 255 is
black.

The **16 pure grays** inside the 256-entry table, measured:

| Index | 0 | 43 | 86 | 129 | 172 | 245 | 246 | 247 | 248 | 249 | 250 | 251 | 252 | 253 | 254 | 255 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Level | FF | CC | 99 | 66 | 33 | EE | DD | BB | AA | 88 | 77 | 55 | 44 | 22 | 11 | 00 |

Sorted descending those levels are `FF EE DD CC BB AA 99 88 77 66 55 44 33 22 11 00` — exactly the
16 multiples of `0x11`, with no gaps and no duplicates. That is not a coincidence: it is the same set
of 16 levels as the Apple 4-bit grayscale device CLUT, which is what makes the depth degradation in
22.4 work. Note they are *not* contiguous in index order: five of them (`FF CC 99 66 33` at indices
0, 43, 86, 129, 172) fall on the diagonal of the 6×6×6 colour cube, and the other eleven
(`EE DD BB AA 88 77 55 44 22 11 00` at indices 245..255) are the dedicated gray ramp at the end.

### 22.3 Depth negotiation

`thisMac` is a `macEnviron` (`GliderPRO/Headers/Environ.h:11-32`), 8 rect/long/short fields then 13
`Boolean`s:

| Field | Type | Meaning |
|---|---|---|
| `screen`, `gray` | `Rect` | main device bounds; `GetQDGlobalsScreenBits`-derived and gray-region bounds |
| `dirID` | `long` | app directory |
| `wasDepth`, `isDepth` | `short` | depth at launch; depth now |
| `thisResFile`, `numScreens`, `vRefNum` | `short` | |
| `can1Bit`, `can4Bit`, `can8Bit` | `Boolean` | device capability |
| `wasColorOrGray` | `Boolean` | **true = colour**, false = grayscale (see below) |
| `hasWNE`, `hasSystem7`, `hasColor`, `hasGestalt`, `canSwitch`, `canColor`, `hasSM3`, `hasQT`, `hasDrag` | `Boolean` | |

`WhatsOurDepth` (`GliderPRO/Sources/Environ.c:232-253`) reads
`(**(**thisGDevice).gdPMap).pixelSize`, or returns 1 if `!thisMac.hasColor`.

`AreWeColorOrGrayscale` (`Environ.c:348-368`) returns `(**thisGDevice).gdFlags & 0x0001`, which is
the `gdDevType` bit: **1 = colour, 0 = monochrome/grayscale**. The variable name
`wasColorOrGray` is therefore misleading — it means "was in *colour* (not gray)".

`isDepthPref` (a saved preference, default `kSwitchIfNeeded`, `Main.c:146`) selects one of three
policies (`GliderDefines.h:43-45`):

| Constant | Value |
|---|---|
| `kSwitchIfNeeded` | 0 |
| `kSwitchTo256Colors` | 1 |
| `kSwitchTo16Grays` | 2 |

`HandleDepthSwitching` (`Environ.c:505-543`), called once from `Main.c:320` before any GWorld exists:

```
if (thisMac.hasColor):
    switch (isDepthPref):
      kSwitchIfNeeded:                                                    // :511
        if (wasDepth != 8) && ((wasDepth != 4) || wasColorOrGray):
            SwitchDepthOrAbort()
      kSwitchTo256Colors:                                                 // :517
        if (wasDepth != 8):
            if (can8Bit) SwitchToDepth(8, true) else SwitchDepthOrAbort()
      kSwitchTo16Grays:                                                   // :527
        if (wasDepth != 4) || wasColorOrGray:
            if (can4Bit) SwitchToDepth(4, false) else SwitchDepthOrAbort()
thisMac.isDepth = WhatsOurDepth()                                         // :542
```

`SwitchToDepth(newDepth, doColor)` (`Environ.c:374-398`) is
`SetDepth(thisGDevice, newDepth, 1, doColor ? 1 : 0)`.

**Therefore the game supports exactly two display modes:**

| Mode | `isDepth` | `SetDepth` colour flag | Palette |
|---|---|---|---|
| 256 colours | 8 | 1 (colour) | Apple standard 8-bit CLUT = `clut` 128 |
| **16 grays** | 4 | 0 (grayscale) | Apple standard 4-bit **grayscale** CLUT |

Note the second one is 16 *grays*, not 16 colours — `kSwitchTo16Grays`, and `SwitchToDepth(4, false)`
at `Environ.c:531`. Every `thisMac.isDepth == 4` branch in the drawing code is a **grayscale**
fallback. (`kSwitchIfNeeded` also accepts an already-4-bit-grayscale device without asking.)

`SwitchDepthOrAbort` (`Environ.c:404-434`) puts up `ALRT` 130 (`kSwitchDepthAlert`,
`Environ.c:18`) with three buttons: 1 → `SwitchToDepth(8, true)`, 2 → `SwitchToDepth(4, false)`,
3 → `ExitToShell()`. If `!thisMac.canSwitch` it takes the `else` at `:430` instead.

`RestoreColorDepth` (`Environ.c:549-554`) puts the monitor back at quit if either the depth or the
colour/gray flag changed.

`Events.c:399-413` re-asserts the depth if the user changes it behind the game's back
(`if (WhatsOurDepth() != thisMac.isDepth) … SwitchToDepth(thisMac.isDepth, thisMac.wasColorOrGray)`).

### 22.4 The 16-gray degradation is a hand-authored lookup, and it is inconsistent

Every GWorld is created with `kPreferredDepth` = 8 (`GliderPRO/Headers/Externs.h:15`), **regardless
of `thisMac.isDepth`** — every one of the ~50 `CreateOffScreenGWorld` calls passes the constant. So
the offscreen surfaces are always 8-bit and the palette indices in them are always 8-bit CLUT
indices. What changes at depth 4 is only the **choice of index** in the ~30 code sites that draw
solid colours procedurally, and the `CopyBits` down-conversion to the screen.

The mechanism is a pair of `if (thisMac.isDepth == 4) {...} else {...}` blocks assigning `long`
colour variables which are then passed to `ColorLine` / `ColorRect` / `ColorOval` / `ColorRegion` /
`ColorFrameRect` / `HiliteRect` (section 4.5). The 4-bit constants are raw literals.

I extracted every (4-bit literal, 8-bit constant) pair from all 30 `isDepth == 4` sites and tested
them against the hypothesis "the 4-bit index is the standard Apple 16-gray ramp index whose level is
nearest to the 8-bit colour's luminance, using the game's own `(r*3 + g*6 + b*1) / 10` formula from
`SetPaletteToGrays` (`MainWindow.c:484-486`)". The 4-bit gray ramp is `level(i) = (15 - i) * 17`,
i.e. index 0 = `#FFFFFF`, index 15 = `#000000`.

| 4-bit idx | 8-bit constant | 8-bit idx | CLUT RGB | lum(3/6/1) | level of the 4-bit idx | nearest-lum 4-bit idx | match? |
|---|---|---|---|---|---|---|---|
| 14 | `k8DkstGrayColor` | 254 | `#111111` | 16 | 17 (`#111111`) | 14 | yes |
| **15** | `k8DkstGrayColor` | 254 | `#111111` | 16 | 0 (`#000000`) | 14 | **NO (+1)** |
| 6 | `k8BambooColor` | 53 | `#CC9900` | 152 | 153 (`#999999`) | 6 | yes |
| 9 | `k8PissYellowColor` | 95 | `#996600` | 106 | 102 (`#666666`) | 9 | yes |
| 11 | `k8BrownColor` | 137 | `#663300` | 60 | 68 (`#444444`) | 11 | yes |
| **9** | `k8TanColor` | 94 | `#996633` | 111 | 102 (`#666666`) | 8 | **NO (+1)** |
| 14 | `k8DkRed2Color` | 223 | `#220000` | 10 | 17 (`#111111`) | 14 | yes |
| **15** | `k8DkRed2Color` | 223 | `#220000` | 10 | 0 (`#000000`) | 14 | **NO (+1)** |
| 15 | `k8BlackColor` | 255 | `#000000` | 0 | 0 (`#000000`) | 15 | yes |
| **7** | `k8LtTanColor` | 52 | `#CC9933` | 157 | 136 (`#888888`) | 6 | **NO (+1)** |
| 13 | `k8DkGray2Color` | 253 | `#222222` | 33 | 34 (`#222222`) | 13 | yes |
| 7 | `k8LtGrayColor` | 249 | `#888888` | 134 | 136 (`#888888`) | 7 | yes |
| 5 | `k8LtstGray5Color` | 248 | `#AAAAAA` | 170 | 170 (`#AAAAAA`) | 5 | yes |
| 4 | `k8LtstGray4Color` | 247 | `#BBBBBB` | 186 | 187 (`#BBBBBB`) | 4 | yes |
| 1 | `k8LtstGrayColor` | 245 | `#EEEEEE` | 236 | 238 (`#EEEEEE`) | 1 | yes |
| 3 | `kPaleVioletColor` | 42 | `#CCCCFF` | 208 | 204 (`#CCCCCC`) | 3 | yes |
| **8** | `k8Gray2Color` | 251 | `#555555` | 84 | 119 (`#777777`) | 10 | **NO (-2)** |
| 10 | `kGrayBackgroundColor` (= 251) | 251 | `#555555` | 84 | 85 (`#555555`) | 10 | yes |
| 11 | `k8DkGrayColor` | 252 | `#444444` | 66 | 68 (`#444444`) | 11 | yes |
| 5 | `k8SkyColor` | 150 | `#33CCFF` | 162 | 170 (`#AAAAAA`) | 5 | yes |

**15 of 20 pairs land exactly on the nearest-luminance gray**, and all nine pure-gray sources are
exact. The five deviations are genuine inconsistencies in the source, not a different rule:

1. `k8DkstGrayColor` (254, `#111111`) maps to **14** when assigned to a named variable
   (`ObjectDraw.c:81`, `:372`, `:515`, `:661`, `:790`, `:899`) but to **15** in the five inline
   shadow calls `ColorOval(&tempRect, 15)` / `ColorRegion(shadowRgn, 15)` (`ObjectDraw.c:185`,
   `:310`, `:405`, `:547`, `:692`). So at depth 4 furniture drop-shadows are pure black instead of
   `#111111`.
2. `k8DkRed2Color` (223) maps to **14** at `ObjectDraw.c:163`, `:281`, `:375` and to **15** at
   `:518`, `:663` and `ObjectDraw2.c:1051`.
3. `k8Gray2Color` (251, `#555555`) maps to **8** (`#777777`) at `ObjectDraw2.c:512` but the same
   index 251 as `kGrayBackgroundColor` maps to **10** (`#555555`) at `Scoreboard.c:144` etc. The
   `ObjectDraw2.c` one is two steps too light.
4. `k8TanColor` (94) → 9 and `k8LtTanColor` (52) → 7 are each one step darker than nearest.

Caveat on evidence: the 16-level ramp `level(i) = (15 - i) * 17` is the *system* 4-bit grayscale
device CLUT; it is not one of the app's own resources, so I cannot read it out of the resource fork.
The inference is however strongly corroborated — nine of nine pure-gray sources map exactly, and the
game's own `SetPaletteToGrays` uses precisely the `3/6/1` luminance weights that make the other
eleven fit.

**Recommendation for a Go port:** implement depth 4 as a *derived* mode (render at 8 bits, then map
each index to `nearestGray4(lum(clut[i]))` at present time) rather than transcribing 30 hand-written
literal tables. That is behaviourally identical at 15 of 20 sites and strictly better at the other 5.
Or, much simpler, drop 4-bit support entirely: it is a 1994 concession to Mac SE/30-class hardware.

There is one more depth-4 behaviour that is *not* about colour:

```c
if ((thisMac.isDepth == 4) && ((src.left % 2) == 1))
{ QOffsetRect(&src, -1, 0);  if (src.left < 0) QOffsetRect(&src, 2, 0); }
```

at `GliderPRO/Sources/DynamicMaps.c:326-331` (`AddCandleFlame`), `:409-414`, `:495-500`, `:584-589`,
`:671-676`. At 4 bits two pixels share a byte, so an odd `src.left` forces QuickDraw's blitter into
a slow, nibble-shifting path (and on some 68k QuickDraw versions it corrupted). The fix nudges the
source rect one pixel left to make `left` even, unless that would go negative in which case it goes
one pixel right instead. **This shifts the flame sprite by one pixel at depth 4.** A Go port with a
byte-per-pixel surface has no reason to do this; drop the whole block.

### 22.5 The three dead palette-animation functions

All three are inside `/* */` blocks in `MainWindow.c` and all three call sites are commented out.
They are transcribed here because they document the *intended* launch presentation, which a port may
want to implement, and because two of them are the only direct-framebuffer code in the program.

**`SetPaletteToGrays`** — `MainWindow.c:454-498`. Snapshot the device CLUT into `wasColors`, build a
luminance-gray version in `newColors`, install the gray version:

```
 1  thePMap = (*thisGDevice)->gdPMap;  theCTab = (*thePMap)->pmTable        // :463-466
 2  wasColors = NewPtr(sizeof(ColorSpec) * 256)   // 256*8 = 2048 bytes      // :468
 3  newColors = NewPtr(sizeof(ColorSpec) * 256)                             // :473
 4  for i in 0..255:                                                        // :477
 5      wasColors[i] = newColors[i] = (*theCTab)->ctTable[i]                 // :479-480
 6      if (i != 5):                                                        // :482
 7          longGray = (red * 3)/10 + (green * 6)/10 + (blue * 1)/10        // :484-486
 8          newColors[i].rgb.red = green = blue = (unsigned short)longGray   // :488-490
 9  theDevice = GetGDevice();  SetGDevice(thisGDevice)                      // :494-495
10  SetEntries(0, 255, newColors)                                          // :496
11  SetGDevice(theDevice)                                                   // :497
```

Two things to note. The luminance weights are `0.3 R + 0.6 G + 0.1 B` computed on the **16-bit**
channel values with three separate integer divisions by 10 (so it truncates three times, losing up
to 2 units of 65535 — negligible). And **index 5 is skipped**: `clut` 128 index 5 is `#FFFF00`, pure
yellow, which is `kYellowColor` / `kIntenseYellowColor` — the colour used for the read-only house
name on the splash screen (`MainWindow.c:77`, section 15.3) and for `ColorText`. Leaving it in colour
means that one string stays yellow through the whole fade.

**`HardDrawMainWindow`** — `MainWindow.c:504-546`. Blits `workSrcMap` to the screen with a
hand-written 32-bit loop, bypassing QuickDraw entirely so that `CopyBits` cannot colour-match the
8-bit indices against the now-gray CLUT:

```
 1  pixMapH = (**thisGDevice).gdPMap                                        // :517
 2  srcRowBytes  = (*(workSrcMap->portPixMap))->rowBytes & 0x7FFF           // :520
 3  destRowBytes = (**pixMapH).rowBytes & 0x7FFF                            // :521
 4  src  = (*(workSrcMap->portPixMap))->baseAddr                            // :522
 5  dest = (**pixMapH).baseAddr + splashOriginH
                                + (splashOriginV + thisMac.menuHigh) * destRowBytes   // :523-524
 6  ShieldCursor(&mainWindowRect, (0,0))                                    // :526-528
 7  mode = true32b;  SwapMMUMode(&mode)                                     // :529-530
 8  for i in 0..459:                                                        // :531
 9      for w in 0..159:  *(long*)dest = *(long*)src;  dest += 4;  src += 4 // :533-538
10      src -= 640;  dest -= 640                                            // :539-540
11      src += srcRowBytes;  dest += destRowBytes                           // :541-542
12  SwapMMUMode(&mode)   // restore                                         // :544
13  ShowCursor()                                                            // :545
```

Every constant is hard-coded to the splash: **460 rows × 160 longs = 640 bytes = 640 pixels at
8 bpp**. `rowBytes & 0x7FFF` strips the high bit that marks a `PixMap` rather than a `BitMap`.
`SwapMMUMode(true32b)` switches the 68030/040 MMU into 32-bit addressing mode so that
`baseAddr` (a 32-bit pointer into NuBus video RAM) is valid — a pure 68k-era requirement.
`ShieldCursor` removes the software cursor from the rect so the loop does not copy over it.

The destination origin `splashOriginH, splashOriginV + thisMac.menuHigh` confirms that
`splashOriginH/V` are **window-local** and that the window's top is 20 px below the screen top.

`thisMac.menuHigh` is worth a note of its own: **there is no `menuHigh` field in `macEnviron`**
(`GliderPRO/Headers/Environ.h:11-32`, transcribed in 22.3 — the struct has `wasDepth`, `isDepth`,
`thisResFile`, `numScreens`, `vRefNum` and thirteen `Boolean`s, and nothing named `menuHigh`).
`grep -n 'menuHigh' Sources/*.c` returns seven hits and **every one of them is dead**:
`MainWindow.c:224` and `:229` are comments explaining a literal `20`, `Play.c:144` likewise, and
`MainWindow.c:378`, `:388`, `:432`, `:524` are all inside `/* */` blocks. So the field was deleted
during the Carbon port and the code that used it was commented out rather than fixed —
`HardDrawMainWindow` would not compile if it were uncommented. That is independent proof that the
menu-bar height is hard-wired to **20** everywhere in the shipped build.

A Go port has nothing to replace this loop with, and needs nothing: writing to its own framebuffer
never goes through a colour-matching layer.

**`WashColorIn`** — `MainWindow.c:552-600`. Interpolates the installed CLUT from gray back to colour
over 180 steps:

```
#define kGray2ColorSteps 180                                              // :554
for i in 0 .. 179:                                                        // :562
    for c in 0 .. 255:                                                    // :564
        if (c != 5):                                                      // :566
            new.red   += (was.red   - new.red)   / (kGray2ColorSteps - i) // :568-571
            new.green += (was.green - new.green) / (kGray2ColorSteps - i) // :573-577
            new.blue  += (was.blue  - new.blue)  / (kGray2ColorSteps - i) // :579-583
    SetEntries(0, 255, newColors)                                         // :586
    if (Button()) break                                                   // :587-588
SetEntries(0, 255, wasColors)                                             // :591
SetGDevice(theDevice);  RestoreColorsSlam()                               // :592-594
DisposePtr(wasColors);  DisposePtr(newColors)                             // :596-599
```

The step is `new += (was - new) / (steps - i)`, i.e. **each iteration closes `1/(180-i)` of the
remaining gap** — a geometric-to-linear schedule that reaches the target exactly at `i = 179` (where
the divisor is 1). It is *not* a constant-rate lerp. Because the arithmetic is integer, values within
`steps - i` of the target stop moving until the divisor shrinks, so dark colours converge in visible
jumps near the start. There is no delay in the loop: the pacing is entirely "one `SetEntries` per
iteration", i.e. as fast as the CLUT hardware will accept — on a 1994 Mac roughly one VBL per
`SetEntries`, so ≈ 180/60 = **3 seconds**. `Button()` lets the user skip it.

Note again that index 5 is skipped in the interpolation too, and that the final `SetEntries(0, 255,
wasColors)` restores the exact original table so no rounding error accumulates.

**`RestoreColorsSlam`** — `GliderPRO/Sources/ColorUtils.c:218-223`, the one live function of the
three-function set:

```c
void RestoreColorsSlam (void)
{
    RestoreDeviceClut(nil);          // :220   nil = all devices
    PaintBehind(nil, GetGrayRgn());  // :221   force every other window to redraw
    DrawMenuBar();                   // :222
}
```

`RestoreDeviceClut(nil)` reinstalls the standard CLUT on every device; `PaintBehind(nil,
GetGrayRgn())` makes the Window Manager repaint the whole desktop (because everything on screen is
now wrong); `DrawMenuBar()` repaints the menu bar. It is exported (`Externs.h:294`) but has no live
caller.

### 22.6 The dead call sites, and the preference that survives

| Site | Guard | Body |
|---|---|---|
| `Main.c:351-356` | `if ((isDoColorFade) && (thisMac.isDepth == 8))` | `wasSeed = ExtractCTSeed(mainWindow); WashColorIn(); ForceCTSeed(mainWindow, wasSeed);` |
| `MainWindow.c:249-257` | `if ((fadeGraysOut) && (isDoColorFade))` | `wasSeed = ExtractCTSeed(...); SetPaletteToGrays(); HardDrawMainWindow(); fadeGraysOut = false; ForceCTSeed(...);` |

Both are commented out. The intended sequence was:

1. `OpenMainWindow` → `UpdateMainWindow` composes the splash into `workSrcMap` (15.2).
2. `fadeGraysOut` is still true, so `SetPaletteToGrays()` turns the *whole screen* gray, then
   `HardDrawMainWindow()` writes the splash's raw 8-bit indices straight to video RAM so QuickDraw
   cannot remap them.
3. The rest of startup happens on a gray screen.
4. At the end of `Main`, `WashColorIn()` walks the CLUT back to full colour over ~3 s, so the splash
   appears to bloom from grayscale into colour.
5. `ExtractCTSeed` / `ForceCTSeed` (`Utilities.c:710`, `:722`) save and restore the port's colour
   table seed so QuickDraw does not notice the CLUT changed and invalidate its caches.
   `ForceCTSeed` is **commented out in the header** too (`Externs.h:368`), so it would not even link.

The state that survives in the shipped build:

| Variable | Default | Set where | Read where |
|---|---|---|---|
| `isDoColorFade` | **true** (`Main.c:155`); prefs override at `Main.c:83`; saved at `:240`; Settings "defaults" sets true at `Settings.c:1260` | user preference | only the two dead sites |
| `fadeGraysOut` | `true` if `thisMac.isDepth == 8`, else `false` (`InterfaceInit.c:132-135`) | `InterfaceInit` | only the dead site |

So the Settings dialog still offers a "color fade" checkbox (`Settings.c:1126`, `:1137`) that does
nothing. This is the same Carbon-port casualty as `DissBits` (section 14.5) and `HideMenuBarOld`
(section 21.1): 1.0.4 kept the preferences and deleted the implementations.

A Go port that wants the effect has an easy path that the original did not: keep the frame as
palette indices, and at present time apply
`out[i] = lerp(gray(clut[i]), clut[i], t)` with `t` ramping 0→1 over 180 frames, skipping index 5.

### 22.7 What a Go port must replace

| Mac mechanism | Go replacement |
|---|---|
| Device CLUT + 8-bit indexed pixels | a baked `[256]color.RGBA` table from `clut` 128, and `[]uint8` index surfaces (keep indices — the XOR modes and `ColorLine`/`ColorRect` all operate on indices) |
| `Index2Color(i, &rgb)` | `clut[i]` |
| `RGBForeColor` / `ForeColor(blackColor)` etc. | a current-index field on the surface state; QuickDraw's 8 basic `ForeColor` constants map to CLUT indices (`blackColor`→255, `whiteColor`→0, and the six others per 5.6) |
| `SetEntries` / `RestoreDeviceClut` | mutate the presentation LUT, not the surfaces |
| `SetDepth` / `SwitchToDepth` | nothing; render 8-bit always |
| `thisMac.isDepth == 4` branches | either derive from luminance (22.4) or drop |
| `SwapMMUMode(true32b)`, direct `baseAddr` writes | direct slice writes |
| `ShieldCursor` / `ShowCursor` | nothing (no hardware cursor overlay to worry about) |
| `ExtractCTSeed` / `ForceCTSeed` | nothing |

---

## 23. Mac Toolbox inventory and Go replacements

This is the complete list of Toolbox entry points the rendering subsystem uses, with what a Go port
must put in their place.

Call counts throughout this section are **measured**, over the twenty-one rendering-related sources
`Render.c RoomGraphics.c ObjectDraw.c ObjectDraw2.c ObjectDrawAll.c Coordinates.c Transitions.c
Marquee.c MainWindow.c Scoreboard.c Banner.c GameOver.c HighScores.c Map.c DynamicMaps.c Grease.c
RectUtils.c ColorUtils.c Utilities.c StructuresInit.c StructuresInit2.c`, with

```
cat <those 21 files> | grep -ao "\bNAME(" | wc -l
```

on the CR→LF converted copies. A count of 0 means the call exists elsewhere in the program but not in
the rendering subsystem. Counts include occurrences inside `/* */` and `//` comments, which matters for
`SetEntries` (all three dead, 22.1) and `SwapMMUMode` (both dead, 22.5).

### 23.1 The bit-blitters — this is the whole renderer

| Toolbox call | Count | Semantics as used here | Go replacement |
|---|---|---|---|
| `CopyBits(srcBits, dstBits, &srcRect, &dstRect, mode, maskRgn)` | **88** | rectangle blit between any two pixmaps with a transfer mode and an optional region mask. In Glider `srcRect` and `dstRect` are **always congruent**, so `CopyBits` never scales. | `Blit(dst, dstRect, src, srcPt, mode, clip)` over `[]uint8` index buffers |
| `CopyMask(srcBits, maskBits, dstBits, &srcRect, &maskRect, &dstRect)` | **76** | three-way blit: copy `src`→`dst` only where the 1-bit `mask` is 1. Glider always passes the same rect for `srcRect` and `maskRect`. | `BlitMask(dst, dstRect, src, mask, srcPt)` |
| `GetGWorldPixMap(gw)` | **366** | `PixMapHandle` for a GWorld, immediately dereferenced and cast `(BitMap *)*…` for `CopyBits`/`CopyMask` | nothing — pass the surface directly |
| `GetPortBitMapForCopyBits(GetWindowPort(w))` | **35** (`GetWindowPort` **24**) | the on-screen window's pixmap | the framebuffer surface |
| `DrawPicture(picHandle, &destRect)` | **8** | decode a PICT and draw it into the current port, **scaling** `picFrame` → `destRect` | decode PICTs at asset-build time; scale nearest-neighbour only if `destRect != picFrame` (4.3) |
| `LockPixels(GetGWorldPixMap(gw))` | **1** (`Utilities.c:275`) | pin the pixel buffer so `baseAddr` stays valid; **never unlocked anywhere in the program** | nothing (Go slices don't move) |

`GetGWorldPixMap` at 366 calls versus `CopyBits`+`CopyMask` at 164 is the giveaway: almost every blit
resolves 2–3 pixmaps inline, e.g.

```c
CopyMask((BitMap *)*GetGWorldPixMap(bonusSrcMap),
         (BitMap *)*GetGWorldPixMap(bonusMaskMap),
         (BitMap *)*GetGWorldPixMap(backSrcMap),
         &srcRects[kBlueClock], &srcRects[kBlueClock], theRect);
```

(`GliderPRO/Sources/ObjectDraw.c:1005-1008`). 366 = 164 blits × ~2.2 pixmaps each, plus the
`CreateOffScreenGWorld` and `HardDrawMainWindow` uses.

#### 23.1.1 Transfer modes — six of QuickDraw's sixteen, and every site

Measured over **all** of `Sources/*.c` (not just the rendering subset), because the mode constants are
rare enough to enumerate exhaustively:

| Mode | Numeric | Occurrences | Every site | Meaning on an 8-bit indexed port | Go |
|---|---|---|---|---|---|
| `srcCopy` | 0 | **119** | everywhere | opaque copy | `dst[i] = src[i]` |
| `transparent` | 36 | **2** | `ObjectDraw2.c:1403` (`DrawCustPict`), `:1430` (`DrawCustPictSansWhite`) | copy every source pixel whose value differs from the port's **background** colour (white = index 0 here, 8.3) | `if src[i] != 0 { dst[i] = src[i] }` |
| `srcXor` | 2 | **1** | `Marquee.c:438` (`DrawGliderMarquee`) | XOR the destination index where the 1-bit source is 1 | `dst[i] ^= mask` where source bit set |
| `patXor` | 10 | **15** | `Marquee.c:41, 67, 130, 151, 195, 225, 269, 352` (8), `RoomInfo.c:132, 172, 193, 213` (4), `DialogUtils.c:261, 315` (2), `ObjectEdit.c:2747` (1) | XOR the destination index where the 8×8 pen pattern bit is 1 | `if patBit(x,y) { dst[i] ^= mask }` |
| `patOr` | 9 | **6** | `ObjectDraw.c:183, 308, 403, 545, 690, 808` — **all six are furniture drop shadows** (23.4.1) | where the pattern bit is 1, `dst = fore`; where 0, leave `dst` unchanged | `if patBit(x,y) { dst[i] = fore }` |
| `srcOr` | 1 | **1** | `ObjectEdit.c:2364` — the edit-mode "darkness" stipple over the whole room | same as `patOr` but source-driven | `if src[i] != 0 { dst[i] = fore }` |

Two notes a porter needs:

- **`patCopy` (8) is never written literally.** It is the mode `PenNormal()` restores, and `PenNormal`
  is called **16 times** — always immediately after a `patXor` or `patOr` block, to put the port back.
  Every `PaintRect`/`ColorRect`/`ColorLine` in the program therefore runs in `patCopy` with the default
  1×1 black-pattern pen unless one of those 22 explicit blocks is active.
- On a *colour* port classic QuickDraw does **not** apply the boolean modes bitwise to the index
  value; `srcOr`/`patOr` degenerate to "1 bits get the foreground colour, 0 bits are left alone", which
  is why `PenPat(gray) + PenMode(patOr)` yields a 50 % dither rather than a bitwise OR of palette
  indices. The XOR modes *are* bitwise on the index, which is what makes the marquee reversible
  (16.4).

#### 23.1.2 The `CopyBits` clipping rule

Established in 1.2 and load-bearing for the mirror code: classic QuickDraw clips a `CopyBits` to the
destination pixmap's own `bounds`, **plus** the current `GrafPort`'s `visRgn` ∩ `clipRgn` *only when
the destination pixmap is the current port's own pixmap*. Two pieces of direct evidence in the source:

- `UpdateMainWindow` passes `GetPortVisibleRegion(GetWindowPort(mainWindow), …)` **explicitly** as the
  `maskRgn` argument (`GliderPRO/Sources/MainWindow.c:131-134`, `:144-147`) even though it is drawing
  into that very window — it is not relying on the implicit path.
- `DrawReflection` calls `SetPort((GrafPtr)workSrcMap)` **before** `SetClip(mirrorRgn)`
  (`GliderPRO/Sources/Render.c:157-159`): it makes `workSrcMap` the current port precisely so that the
  clip region will apply to a blit whose destination is `workSrcMap`.

A Go port should make this explicit — every blit takes a clip rect (or region) argument and there is no
hidden "current port" state — and must not silently clip blits whose destination is *not* the "current"
surface, because a dozen `SetGWorld(x); CopyBits(a → b)` sequences in Glider have `b != x`.

### 23.2 GWorlds and ports

| Toolbox call | Count | Go replacement |
|---|---|---|
| `NewGWorld(&gw, depth, &bounds, cTab, aGDevice, flags)` | **2** — both inside `CreateOffScreenGWorld` (`Utilities.c:270` with `useTempMem`, `:273` retry without) | `NewSurface(w, h)` → `{W, H, Stride, Pix []uint8}` |
| `CreateOffScreenGWorld(&gw, &r, depth)` (Glider's wrapper) | **77** | as above; the wrapper also does `LockPixels` and an `EraseRect` |
| `DisposeGWorld(gw)` | **25** | GC, or an explicit `Free()` for the temp-GWorld idiom (8.2) |
| `KillOffScreenPixMap` | **2** | nothing |
| `GetGWorld(&port, &dev)` / `SetGWorld(gw, nil)` | **62** / **178** | an explicit `dst *Surface` parameter on every draw call — **do not** reproduce the ambient current-port model |
| `SetPort((GrafPtr)gw)` | **33** | same. Note Glider frequently uses `SetPort` on a `GWorldPtr` rather than `SetGWorld`; this compiles only because a `GWorldPtr` *is* a `CGrafPtr`, and it differs from `SetGWorld` in that it does **not** set the current `GDevice` |
| `GetPort` / `SetPortWindowPort` | **3** / **19** | same |
| `OpenPort` / `ClosePort` / `CloseCPort` | **3** / **3** / **2** | nothing |
| `GetGDevice` / `SetGDevice` | **3** / **6** | nothing |

`SetGWorld` at **178** calls against `GetGWorld` at **62** is the second structural giveaway: the
save/restore pairing is 62 balanced sites plus 116 unbalanced `SetGWorld`s. Sections 13.11 and 17.2
document where that leaks visibly.

The ambient-current-port model is the single biggest structural difference from Go. Glider's draw
helpers (`ColorLine`, `ColorRect`, `ColorOval`, `ColorRegion`, `ColorText`, `DrawString`, `PaintRect`,
`FrameRect`, …) take **no destination argument at all** — they draw into whatever port was last
`SetPort`/`SetGWorld`'d. A port must thread the destination explicitly, and then decide deliberately
about the two places the original relies on the leak.

### 23.3 Rect and region arithmetic

Glider wraps most rect maths in `RectUtils.c`; its own helpers dwarf the Toolbox calls:

| Glider helper | Count | Definition |
|---|---|---|
| `QOffsetRect(r, dh, dv)` | **323** | `RectUtils.c` inline; no empty-rect check, unlike `OffsetRect` |
| `QSetRect(r, l, t, rt, b)` | **305** | inline `SetRect` |
| `LoadGraphic(pictID)` | **71** | `GetPicture` + `DrawPicture` into the current port (4.1) |
| `OffsetRectRoomRelative(r)` | **65** | room → screen translation (2.6) |
| `GetObjectRect(obj, r)` | **65** | object → room-local rect |
| `ZeroRectCorner(r)` | **35** | move top-left to (0,0) |
| `AddRectToWorkRects(r)` / `AddRectToBackRects(r)` | **21** / **11** | dirty-rect enqueue (9.3) |
| `RectWide` / `RectTall` / `HalfRectTall` / `HalfRectWide` | **20** / **8** / **11** / **7** | `right-left`, `bottom-top`, halves |
| `CenterRectInRect(r, inR)` | **7** | centring |
| `NormalizeRect(r)` | **2** | swap inverted edges |

The Toolbox calls that remain:

| Toolbox call | Count | Go replacement |
|---|---|---|
| `SectRect(&a, &b, &out)` | **60** | `image.Rectangle.Intersect` — **but the return value is used as a boolean "do they overlap"** at every object-culling site (`ObjectDrawAll.c:74, 93, 108, 127, 142, 161, 188, 275, 289, 303, 317, …`, always with a throwaway `whoCares` out-rect). In Go, `!a.Intersect(b).Empty()` |
| `UnionRect` | **2** | `image.Rectangle.Union` |
| `InsetRect(&r, dh, dv)` | **29** | QuickDraw insets h and v **independently**; `image.Rectangle.Inset(n)` takes one value, so write your own |
| `OffsetRect` | **9** | `r.Add(pt)` (this is the non-`Q` variant, which no-ops on empty rects) |
| `SetRect` | **2** | struct literal |
| `PtInRect` | **1** | `r.Contains(p)` |
| `NewRgn` / `DisposeRgn` | **16** / **16** | a region type — but see below, Glider's regions are shallow |
| `RectRgn(rgn, &r)` | **8** | region from one rect |
| `OpenRgn` / `CloseRgn` | **6** / **6** | region accumulated from the drawing calls made between them |
| `UnionRgn` / `SectRgn` / `DiffRgn` | **3** / **2** / **1** | region ops; the `DiffRgn` and 2 of the 3 `UnionRgn` are in the dead menu-bar code |
| `GetClip` / `SetClip` / `ClipRect` | **2** / **3** / **5** | an explicit clip field on the draw context |
| `GetPortVisibleRegion` | **3** | in a Go port the window is always fully visible, so pass `nil` |
| `GetQDGlobalsGray(&pat)` | **7** | the constant pattern `AA 55 AA 55 AA 55 AA 55` (50 % checkerboard) |
| `GetGrayRgn()` | **3** | the desktop region; only `RestoreColorsSlam` / `PaintBehind` use it — drop |
| `SetOrigin(1, 1)` | **1** (`Map.c:398`) | see 18.3 — replace with explicit coordinate arithmetic |

**Region usage is limited and shallow.** There are exactly four kinds:

1. `mirrorRgn` — a **union of axis-aligned rects**, built per room from the mirror objects, used as
   the clip for `DrawReflection` (12.5).
2. The four furniture drop-shadow regions, built with `OpenRgn` / diagonal `Line` / `CloseRgn` —
   **closed polygons with 45° edges** (23.4.1). These are the only non-rectilinear regions the
   renderer fills.
3. `menuBarRgn` in `HideMenuBarOld` (`MainWindow.c:389-397`) and `ShowMenuBarOld` (`:433-440`) —
   **both functions are entirely commented out** (21.1), so this is dead.
4. Single-rect regions from `RectRgn`, used only as arguments to `SetMovieDisplayClipRgn` (19.4) and
   the region ops above.

So a Go port needs: rect-union regions (a `[]image.Rectangle` with a scanline intersection test) plus
**one** polygon rasteriser for the four shadow shapes. It does not need a general region engine.

### 23.4 Pen, pattern and primitive drawing

| Toolbox call | Count | Go replacement |
|---|---|---|
| `PaintRect(&r)` | **32** | fill with the current **fore** index |
| `EraseRect(&r)` | **4** | fill with the current **back** index |
| `FrameRect(&r)` | **24** | 1-px outline drawn **inside** `r`: rows `top` and `bottom-1`, columns `left` and `right-1` |
| `FrameRoundRect(&r, ovalW, ovalH)` | **2** | rounded outline (`HighScores.c`) |
| `PaintOval` / `FrameOval` / `PaintRgn` | **1** / **1** / **1** | each appears exactly once, inside the `ColorOval` / `ColorFrameOval` / `ColorRegion` wrappers (`ColorUtils.c:59`, `:160`, `:75`) |
| `MoveTo` | **67** | set pen position |
| `Line(dh, dv)` / `LineTo(h, v)` | **68** / **13** | relative / absolute line. **Not all axis-aligned** — see 23.4.2 |
| `PenMode(mode)` | **14** | current transfer mode |
| `PenPat(&pat)` | **16** | current 8×8 pattern |
| `PenNormal()` | **16** | reset to `patCopy`, black pattern, 1×1 pen |
| `GetIndPattern(&pat, listID, index)` | **1** (`Marquee.c:508`) | index into the decoded `PAT#` 128 (5.7) |
| `ColorOval` / `ColorFrameOval` / `ColorRegion` | **3** / **2** / **8** | filled ellipse / outlined ellipse / filled region, each = `GetForeColor` + `Index2Color` + `RGBForeColor` + paint + restore (`ColorUtils.c:52-79`, `:153-162`) |

The pen is **always 1×1** — `PenSize` is never called anywhere in `Sources/*.c`. Patterns are
**always aligned to the port origin**, not to the shape being filled (16.2), which matters for the
marquee and for the shadow dither below.

#### 23.4.1 The furniture drop shadows — `patOr` + 50 % gray

All six `patOr` sites are the same idiom, and none of them is documented elsewhere in this file. The
generic shape is:

```
 1  SetGWorld(backSrcMap, nil)                 // shadows go in the *static* layer
 2  <build a shape: either a Rect for ColorOval, or OpenRgn..CloseRgn>
 3  PenPat(GetQDGlobalsGray(&dummyPattern))    // AA 55 AA 55 AA 55 AA 55
 4  PenMode(patOr)
 5  if (thisMac.isDepth == 4)  ColorXxx(shape, 15)          // 4-bit: index 15 = black
 6  else                       ColorXxx(shape, <dark gray>) // 8-bit: 8 or dkGrayC
 7  PenNormal()
```

Because the mode is `patOr` and the pattern is the 50 % checkerboard, **the shadow is a dither: half
the pixels become dark gray and half keep the floor/wall colour underneath**. It is not an alpha
blend and it is not a solid fill. The checkerboard phase is fixed to the `backSrcMap` origin, so a
shadow's dither pattern does not move when the object does.

The six sites in full:

| # | Function | Line | Shape | Path / rect | 8-bit index | 4-bit index |
|---|---|---|---|---|---|---|
| 1 | `DrawTable` | `ObjectDraw.c:177-188` | **oval** via `ColorOval` | `QSetRect(tableTop->left, 0, tableTop->right, RectWide(tableTop)/10)`, then `QOffsetRect(0, -HalfRectTall + kTableShadowTop + down)`, then `QOffsetRect(kTableShadowOffset, -kTableShadowOffset)` | `k8DkstGrayColor` | 15 |
| 2 | `DrawShelf` | `:296-314` | **region** | `MoveTo(shelfTop->left, shelfTop->bottom)`; `Line(12, 12)`; `Line(RectWide(shelfTop) - 4, 0)`; `Line(0, -6 + 1)`; `Line(-12, -12)`; `LineTo(shelfTop->left, shelfTop->bottom)` | `k8DkstGrayColor` | 15 |
| 3 | `DrawCabinet` | `:391-409` | **region** | `MoveTo(cabinet->left, cabinet->bottom)`; `Line(6, 6)`; `Line(RectWide(cabinet), 0)`; `Line(0, -RectTall(cabinet) + 4)`; `Line(-6, -6)`; `LineTo(cabinet->left, cabinet->bottom)` | `dkGrayC` | 15 |
| 4 | `DrawCounter` | `:532-551` | **region** | `MoveTo(counter->right - 2, counter->bottom)`; `Line(10, -10)`; `Line(0, -RectTall(counter) + 29)`; `Line(2, 0)`; `Line(0, -7)`; `Line(-12, -12)`; `LineTo(counter->right - 2, counter->bottom)` | `dkGrayC` | 15 |
| 5 | `DrawDresser` | `:677-696` | **region** | `MoveTo(dresser->left + 10, dresser->bottom + 9)`; `Line(RectWide(dresser) - 11, 0)`; `Line(9, -9)`; `Line(0, -RectTall(dresser) + 12)`; `Line(-9, -9)`; `Line(-RectWide(dresser) + 11, 0)`; `LineTo(dresser->left + 10, dresser->bottom + 9)` | `k8DkstGrayColor` | 15 |
| 6 | `DrawDeckTable` | `:802-810` | **oval** via `ColorOval` | identical to site 1 | `dkGrayC` — **no `isDepth == 4` branch at all** | (none) |

Constants (all function-local `#define`s except `kShelfThick`):

| Constant | Value | Definition |
|---|---|---|
| `kTableShadowTop` | 312 | `GliderPRO/Sources/ObjectDraw.c:150` (and re-defined identically at `:776`) |
| `kTableShadowOffset` | 12 | `GliderPRO/Sources/ObjectDraw.c:151` (and `:777`) |
| `kShelfDeep` | 4 | `GliderPRO/Sources/ObjectDraw.c:266` |
| `kShelfShadowOff` | 12 | `GliderPRO/Sources/ObjectDraw.c:268` |
| `kShelfThick` | 6 | `GliderPRO/Headers/GliderDefines.h:440` |
| `kCabinetDeep` | 4 | `GliderPRO/Sources/ObjectDraw.c:360` |
| `kCabinetShadowOff` | 6 | `GliderPRO/Sources/ObjectDraw.c:361` |

So the shadow shapes are **hexagons and pentagons with two 45° edges** (offsets of the form `(n, ±n)`),
projecting down-right for the shelf/cabinet/dresser and up-right for the counter. `DrawDeckTable`
(site 6) is the one that forgot the depth-4 branch, so on a 4-bit grayscale display a deck table's
shadow uses the 8-bit index `dkGrayC` — one of the five source inconsistencies enumerated in 22.4.

Note also that all six draw into `backSrcMap`, i.e. shadows are part of the static room background and
are composited once at room load, never per frame.

#### 23.4.2 Clock hands — the only diagonal lines in the renderer

The claim "every line in Glider is horizontal or vertical" is true for the 153 `ColorLine` calls
(verified by parsing all 153 argument lists: every one has either `h0 == h1` or `v0 == v1`), but
**false** for the raw `Line`/`LineTo` calls. The diagonal ones are:

| Where | Count | What |
|---|---|---|
| `ObjectDraw.c:1069-1171` (`DrawClockHands`) | 24 | small analog clock hands |
| `ObjectDraw.c:1187-1289` (`DrawLargeClockHands`) | 24 | large analog clock hands |
| `ObjectDraw.c:301, 304, 396, 399, 537, 541, 683, 685` | 8 | shadow-region path segments (23.4.1) — these bound a region, they are not rasterised as lines |
| `ObjectInfo.c:146-175` | 8 | edit-mode link arrowheads |
| `About.c:68-71` | 4 | the diamond in the About box |
| `DialogUtils.c:754-772`, `RectUtils.c:307-316` | 8 | `LineTo` calls that are actually axis-aligned in effect (they trace rect edges) |

So the renderer proper needs a real Bresenham rasteriser for exactly one thing: **clock hands**. All
four clock prize objects:

| Object | id | Draw fn | Line | Hand origin (relative to `theRect`) | Hands |
|---|---|---|---|---|---|
| `kRedClock` | 0x21 | `DrawRedClock` | `ObjectDraw.c:959-986` | — | **digital**: four 4×6 digit blits from `digits[]` |
| `kBlueClock` | 0x22 | `DrawBlueClock` | `:999-1016` | `(left + 13, top + 13)` | `DrawClockHands` (small) |
| `kYellowClock` | 0x23 | `DrawYellowClock` | `:1020-1037` | `(left + 13, top + 15)` | `DrawClockHands` (small) |
| `kCuckoo` | 0x24 | `DrawCuckoo` | `:1041-1058` | `(left + 19, top + 31)` | `DrawLargeClockHands` (large) |

All four read the real wall clock and quantise:

```c
GetTime(&timeRec);
hour    = timeRec.hour % 12;
minutes = ((timeRec.minute + 2) / 5) % 12;      // nearest 5-minute mark, 0..11
```

(`ObjectDraw.c:1012-1014`, `:1033-1035`, `:1054-1056`.) `DrawRedClock` is different — it shows real
minutes, and forces `hour == 0` to `12` (`:970-974`), then draws the digits with
`QSetRect(&dest, 0, 0, 4, 6)` at `(left + 5, top + 7)` and `QOffsetRect` steps of **+4, +6, +4**
(`:976-985`), with the tens-of-hours digit drawn only `if (hour > 9)`.

The hand vector tables are hard-coded 12-way switches, i.e. a pre-baked cosine table. Both hands start
at the same origin `where`:

| Position | small big hand (`:1073-1118`) | small little hand (`:1125-1170`) | large big hand (`:1191-1236`) | large little hand (`:1243-1288`) |
|---|---|---|---|---|
| 0 | `Line(0, -6)` | `Line(0, -4)` | `Line(0, -10)` | `Line(0, -6)` |
| 1 | `Line(3, -5)` | `Line(2, -3)` | `Line(5, -9)` | `Line(3, -5)` |
| 2 | `Line(5, -3)` | `Line(3, -2)` | `Line(9, -5)` | `Line(5, -3)` |
| 3 | `Line(6, 0)` | `Line(4, 0)` | `Line(10, 0)` | `Line(6, 0)` |
| 4 | `Line(5, 3)` | `Line(3, 2)` | `Line(9, 5)` | `Line(5, 3)` |
| 5 | `Line(3, 5)` | `Line(2, 3)` | `Line(5, 9)` | `Line(3, 5)` |
| 6 | `Line(0, 6)` | `Line(0, 4)` | `Line(0, 10)` | `Line(0, 6)` |
| 7 | `Line(-3, 5)` | `Line(-2, 3)` | `Line(-5, 9)` | `Line(-3, 5)` |
| 8 | `Line(-5, 3)` | `Line(-3, 2)` | `Line(-9, 5)` | `Line(-5, 3)` |
| 9 | `Line(-6, 0)` | `Line(-4, 0)` | `Line(-10, 0)` | `Line(-6, 0)` |
| 10 | `Line(-5, -3)` | `Line(-3, -2)` | `Line(-9, -5)` | `Line(-5, -3)` |
| 11 | `Line(-3, -5)` | `Line(-2, -3)` | `Line(-5, -9)` | `Line(-3, -5)` |

Every one of the 48 vectors above was extracted programmatically from the four `switch` bodies at
`ObjectDraw.c:1070-1119`, `:1122-1171`, `:1188-1237` and `:1240-1289`; all four tables are complete
(cases 0..11, no gaps, no fallthrough). The radii are:

| Table | Radius (magnitude at positions 0/3/6/9) | Intermediate magnitude |
|---|---|---|
| small big hand | **6** | (3,5) → √34 ≈ 5.83 |
| small little hand | **4** | (2,3) → √13 ≈ 3.61 |
| large big hand | **10** | (5,9) → √106 ≈ 10.30 |
| large little hand | **6** | (3,5) → √34 ≈ 5.83 |

Note that **the large clock's little-hand table is byte-identical to the small clock's big-hand
table** — the author reused the radius-6 ramp, so there are only **three** distinct hand tables in the
program (radius 4, 6 and 10).

All three are generated exactly by

```
dh = round(r * sin(30° * position))
dv = -round(r * cos(30° * position))
```

with round-half-away-from-zero. **Verified with python3: all 48 literal vectors reproduced, 12/12 for
each of the four tables, zero mismatches.** So a Go port can compute them rather than transcribing
them — but it must use this rounding rule; round-half-to-even would give the same answers here (no
value lands on an exact .5) but a truncating `int()` cast would not: `int(6·cos 30°) = 5` is right yet
`int(4·cos 30°) = 3` is right and `int(10·cos 30°) = 8` is **wrong** (the literal is 9).

Colour handling differs between the two, and is a genuine asymmetry:

| Function | Fore colour set? | On exit |
|---|---|---|
| `DrawClockHands` (`:1062-1174`) | **no `ForeColor` at all** — the hands are drawn in whatever colour `backSrcMap`'s port currently holds | just `SetGWorld(wasCPort, wasWorld)` (`:1173`) |
| `DrawLargeClockHands` (`:1178-1293`) | `ForeColor(whiteColor)` at `:1185` | `ForeColor(blackColor)` at `:1291`, then `SetGWorld` |

So the cuckoo clock's hands are **white** and the blue/yellow clocks' hands are the ambient colour —
which is black in practice because `DrawLargeClockHands` and the `Color*` wrappers all leave
`backSrcMap` at `blackColor`, but it is not stated anywhere. A Go port should pass the index
explicitly: white for `kCuckoo`, black for `kBlueClock` / `kYellowClock`.

Also note: these are the only four objects in the game whose appearance depends on **wall-clock time**,
so they are redrawn only when the room's background is rebuilt, not per frame — the hands are frozen at
the time the player entered the room.

### 23.5 Colour

| Toolbox call | Count | Go replacement |
|---|---|---|
| `ForeColor(n)` / `BackColor(n)` | **62** / **3** | set the current fore/back **index**; `n` is one of the eight legacy QuickDraw constants (22.7) |
| `GetForeColor(&rgb)` | **12** | read it back; every one is the save half of a `Color*` wrapper (`ColorUtils.c:24, 40, 56, 72, 88, 124, 157`) |
| `RGBForeColor(&rgb)` | **27** | set fore colour by RGB; QuickDraw finds the nearest CLUT index |
| `Index2Color(i, &rgb)` | **17** | `clut[i]` |
| `SetEntries(0, 255, specs)` | **3** — all inside comments (22.1) | mutate the presentation LUT |
| `RestoreDeviceClut(nil)` | **1** | nothing |
| `SwapMMUMode(&mode)` | **2** — both in the dead `HardDrawMainWindow` (22.5) | nothing |

The `Index2Color` + `RGBForeColor` pair is a **workaround, not a colour conversion**: `ForeColor` only
accepts the eight legacy constants, so to set an arbitrary palette index the code round-trips index →
RGB → nearest index. `ColorRect`, `ColorOval`, `ColorRegion`, `ColorLine`, `ColorText`,
`ColorFrameRect`, `ColorFrameOval` are all exactly this (`ColorUtils.c:20-162`):

```c
void ColorRect (Rect *theRect, long color)
{
    GetForeColor(&wasColor);          // ColorUtils.c:40
    Index2Color(color, &theRGBColor); // :41
    RGBForeColor(&theRGBColor);       // :42
    PaintRect(theRect);               // :43
    RGBForeColor(&wasColor);          // :44
}
```

The round trip is exact because the 256-entry CLUT has no duplicate RGB values. **Verified with
python3 against `clut` 128 parsed out of `Glider PRO.r`: 2056 bytes, `ctSeed` 0x00000000, `ctFlags`
0x0000, `ctSize` 255, `value` fields sequential 0..255, and 256 distinct RGB triples out of 256 — zero
duplicates, and still zero after collapsing each 16-bit channel to 8 bits.** So `nearestIndex(clut[i])`
== `i` for every `i`, and `RGBForeColor(Index2Color(i))` cannot land on a different index. A Go
port should collapse the whole family to "fill shape with index `color`" and skip the round trip
entirely — with one caveat: the *save/restore* of the previous fore colour is observable, because the
pen colour leaks between the `Color*` calls and the raw `PaintRect`/`Line`/`DrawString` calls
(23.4.2 is exactly this).

### 23.6 Text

| Toolbox call | Count | Go replacement |
|---|---|---|
| `TextFont(applFont \| systemFont)` | **12** | a bitmap font atlas. `systemFont` = 0 = **Chicago**, `applFont` = 1 = **Geneva** |
| `TextSize(n)` | **13** | 9, 10 and 12 pt are the only sizes used |
| `TextFace(bold)` | **13** | the bold variant; QuickDraw *synthesised* bold by OR-ing the glyph one pixel right, so a faithful port either bakes that in or emulates it |
| `DrawString(pStr)` | **41** | draw a Pascal string at the pen, advancing it |
| `StringWidth(pStr)` / `TextWidth(p, off, len)` | **5** / **1** | measure, for centring |
| `NumToString(long, str)` | **18** | integer → Pascal string |
| `GetIndString` / `GetLocalizedString` | **5** / **8** | `GetLocalizedString(i, s)` is Glider's own one-liner: `GetIndString(s, kLocalizedStringsID /* 150 */, i)` (`GliderPRO/Sources/StringUtils.c:321-327`) |
| `ColorText(str, index)` | **4** | `Index2Color` + `RGBForeColor` + `DrawString` + restore (`ColorUtils.c:20-30`) |
| `SetWTitle` | **1** | window title (edit mode, `MainWindow.c:306+`) |

Fonts are the one asset class the resource fork does **not** contain: Chicago and Geneva live in the
System file. A Go port must supply its own bitmap versions or accept a visual difference. Every text
position in the renderer is a **baseline** (`MoveTo(h, v)` then `DrawString`), so glyph ascent is
load-bearing: the measured scoreboard baselines (white at `(0, 9)` over black at `(1, 10)`, 13.7)
assume a 9-px ascent.

### 23.7 Windows and controls

| Toolbox call | Count | Go replacement |
|---|---|---|
| `GetNewCWindow(id, behind, front)` | **3** | `WIND` 128/129/130, measured in 5.8 |
| `NewCWindow` / `NewWindow` | **2** / **1** | programmatic window creation (the coordinate windoid, 17.1) |
| `SizeWindow` / `MoveWindow` | **4** / **5** | resize / move |
| `ShowWindow` / `ShowHide` / `DisposeWindow` | **3** / **2** / **2** | |
| `BeginUpdate` / `EndUpdate` | **2** / **2** | the update-event protocol: `BeginUpdate` narrows `visRgn` to the damaged region, `EndUpdate` clears it. A port that redraws every frame does not need this |
| `InvalWindowRect` | **3** | mark damaged |
| `GetWindowRect` | **2** | window bounds |
| `NewControl` / `MoveControl` / `SizeControl` / `SetControlValue` / `SetControlMaximum` / `GetControlValue` / `TrackControl` / `DrawControls` | **2** / **2** / **2** / **12** / **2** / **28** / **4** / **1** | the map window's two scroll bars, and only those (18.7) |
| `DrawGrowIcon` | **1** | the grow box |
| `HiliteAllWindows` / `BringToFront` / `FlagWindowFloating` | **2** / **2** / **2** | floating-palette behaviour for the two windoids |
| `LMGetMBarHeight` / `LMSetMBarHeight` / `DrawMenuBar` | **2** / **2** / **2** | all inside the commented-out `HideMenuBarOld` / `ShowMenuBarOld` (21.1). Menu-bar height is hard-wired to 20 (22.5); a port needs nothing |
| `ZoomRectToRect` | **2** | both inside the `/* */`-commented `ZoomBetweenWindows` (`MainWindow.c:277-299`) — dead, and the function has no definition anywhere in `Sources/`. Drop it |

### 23.8 Resources and memory

| Toolbox call | Count | Go replacement |
|---|---|---|
| `GetPicture(id)` | **9** | a decoded PICT from the asset bundle |
| `GetResource('Date', id)` | **3** | house-supplied art; same decoder, different container (4.2) |
| `ReleaseResource(h)` | **9** | nothing |
| `HLock` / `HUnlock` / `HGetState` / `HSetState` | **26** / **5** / **21** / **23** | nothing |
| `NewPtr` / `NewPtrClear` / `DisposePtr` | **25** / **2** / **7** | `make([]T, n)` |
| `MoreMasters` | **4** | nothing |

The `HGetState` / `HLock` / … / `HSetState` sandwich around every handle dereference is pure 68k Memory
Manager bookkeeping: relocatable blocks could move during any allocation, so anything holding a
`*(*h)` pointer had to lock first. All **75** of those calls plus the 4 `MoreMasters` vanish in Go —
79 lines of noise that make the C look much larger than the logic it expresses.

### 23.9 Timing, cursor, input, sound

| Toolbox call | Count | Go replacement |
|---|---|---|
| `TickCount()` | **11** | monotonic clock quantised to 1/60 s (20.10) |
| `GetDateTime` / `GetTime` | **3** / **5** | wall clock — `GetTime` is the clock prizes (23.4.2), `GetDateTime` the high-score date |
| `DelayTicks(n)` — Glider's wrapper on `Delay` (`Utilities.c`) | **2** | sleep |
| `InitCursor` / `SetCursor` / `HideCursor` / `ShowCursor` / `ShieldCursor` | **9** / **4** / **0** / **1** / **1** | a cursor image. `ShieldCursor` (1, in the dead `HardDrawMainWindow`) hid the cursor before direct framebuffer writes |
| `SpinCursor(n)` — Glider's own (`AnimCursor.c`) | **17** | animate the `acur` 128 progress cursor during load |
| `GetMouse` / `MyGetGlobalMouse` / `GlobalToLocal` / `LocalToGlobal` / `DeltaPoint` | **5** / **2** / **3** / **2** / **4** | pointer position. `DeltaPoint(a, b)` returns a packed `long` = `(dv << 16) \| (dh & 0xFFFF)` |
| `WaitMouseUp` / `Button()` | **4** / **1** | mouse state; the single `Button()` is `WashColorIn`'s skip test (22.5) |
| `GetKeys(theKeys)` / `BitTst` | **6** / **16** | `theKeys` is a 128-bit `KeyMap`; `BitTst(&map, n)` indexes it with **bit 0 = MSB of byte 0**, i.e. bit-reversed within each byte relative to a naive `map[n/8] >> (n%8)` |
| `WaitNextEvent` / `GetNextEvent` / `FlushEvents` | **0** / **3** / **8** | the event loop (20.8) — `WaitNextEvent` is in `Play.c`/`Events.c`, outside the rendering subset |
| `PlayPrioritySound` — Glider's wrapper | **18** | audio; out of scope |
| `RandomInt(range)` — Glider's own | **15** | `(\|Random()\| * range) / 32768` (`GliderPRO/Sources/Utilities.c:72-82`). **Edge case:** `Random()` can return `-32768`, whose absolute value is `32768`, so `RandomInt(range)` can return `range` itself — one in 65536 |
| `Random()` | **3** | the Toolbox 16-bit LCG. **For demo-playback determinism a port must reproduce its exact sequence**, or the recorded auto-demo (`demo` 128) desynchronises (20.4, 21.7) |
| `SysBeep` | **4** | |

### 23.10 QuickTime

Fully covered in section 19. The complete set of Movie Toolbox entry points used: `EnterMovies`,
`OpenMovieFile`, `NewMovieFromFile`, `CloseMovieFile`, `SetMovieGWorld`, `GetMovieBox`, `SetMovieBox`,
`SetMovieDisplayClipRgn`, `GoToBeginningOfMovie`, `StartMovie`, `StopMovie`, `SetMovieActive`,
`MoviesTask`, `PrerollMovie`, `LoadMovieIntoRam`, `GetMovieTimeBase`, `SetTimeBaseFlags`,
`SetMovieMasterTimeBase`, `GetMovieDuration`, `GetMovieUserData`, `RemoveUserDataItem`, `AddUserData`,
`DisposeMovie`. All of it is behind `#ifdef COMPILEQT` (21.1) and all of it is optional at run time
(`hasQT`), so a Go port can defer the whole subsystem behind a capability flag.

### 23.11 Byte order and on-disk layout

Everything in the resource fork and in house files is **big-endian**, and several Toolbox structs have
counter-intuitive layout:

| Type | Layout | Trap for a porter |
|---|---|---|
| `Point` | `short v; short h;` | **vertical first**. Every `Point` literal, every `GetMouse` result, every `DeltaPoint` |
| `Rect` | `short top, left, bottom, right;` | t-l-b-r, not x/y/w/h; `right`/`bottom` are exclusive |
| `RGBColor` | `unsigned short red, green, blue;` | 16 bits per channel. Glider's CLUT duplicates each byte (`0xCC` → `0xCCCC`), so `>>8` is lossless (22.2) |
| `Pattern` | `unsigned char[8]` | 8 rows, **MSB = leftmost pixel** |
| `ColorSpec` | `short value; RGBColor rgb;` | 8 bytes |
| `ColorTable` | `long ctSeed; short ctFlags; short ctSize;` then `ctSize + 1` `ColorSpec`s | **`ctSize` is the last index, not the count** — 255 means 256 entries, total 8 + 256×8 = 2056 bytes (measured, 22.2) |
| `Picture` | `short picSize; Rect picFrame;` then opcodes | `picSize` is a **signed** 16-bit byte count and overflows for pictures over 32767 bytes. PICT 1000 is 108482 bytes, so its `picSize` is garbage and must be ignored (5.2) |
| `PixMap.rowBytes` | high bit set distinguishes `PixMap` from `BitMap`; mask with `& 0x7FFF` | `MainWindow.c:520-521` |
| `KeyMap` | `unsigned long[4]`, 128 bits | `BitTst` bit numbering is MSB-first within each byte |
| house files | all `short`/`long` big-endian, `Str255` = length-prefixed Mac Roman | see the house-format analysis |

A Go port should decode all of this **once**, at asset-build time, into native little-endian structs,
and never parse big-endian data at run time.

### 23.12 68k / PowerPC specifics

| Thing | Where | Note |
|---|---|---|
| `SwapMMUMode(true32b)` | `MainWindow.c:519`, `:544` (dead) | 24-bit vs 32-bit addressing mode for direct `baseAddr` writes on 68k |
| Direct framebuffer writes | `HardDrawMainWindow`, `MainWindow.c:504-546` (dead) | 460 rows × 160 longs = 640 px per row, `destRowBytes = rowBytes & 0x7FFF` |
| `#if defined(powerc) \|\| defined(__powerc)` | `MainWindow.c:82-93` | the drop-shadowed "PowerPC Native!" badge on the splash (21.6) |
| `register long` | `Utilities.c:74` and ~40 other places | a 68k compiler hint; meaningless in Go |
| `#pragma parameter` / calling conventions | `Externs.h:385` | inline-trap glue |
| Fat binary | shipped as both 68k and PPC code resources | irrelevant |

There is **no** hand-written assembly and no dependence on 68k arithmetic quirks anywhere in the
rendering subsystem. The one place integer width matters is `short` overflow: `picSize` (above) and the
`long` accumulators in `CheckMemorySize` (3.14).

### 23.13 The minimum surface a Go port needs

Everything in sections 1–22 reduces to this API:

```go
type Surface struct { W, H, Stride int; Pix []uint8 }   // 8-bit palette indices
type Mask    struct { W, H, Stride int; Bits []uint8 }  // 1 bit per pixel, MSB leftmost
type Region  []image.Rectangle                          // rect union; plus one polygon path type

func Blit       (dst *Surface, dr image.Rectangle, src *Surface, sp image.Point, mode Mode, clip Region)
func BlitMask   (dst *Surface, dr image.Rectangle, src *Surface, msk *Mask, sp image.Point)
func FillRect   (dst *Surface, r image.Rectangle, idx uint8)
func FrameRect  (dst *Surface, r image.Rectangle, idx uint8)          // inside r, 1 px
func HLine      (dst *Surface, x0, x1, y int, idx uint8)
func VLine      (dst *Surface, x, y0, y1 int, idx uint8)
func DiagLine   (dst *Surface, p0, p1 image.Point, idx uint8)          // clock hands only
func FillOval   (dst *Surface, r image.Rectangle, idx uint8, pat *[8]byte)   // pat != nil => patOr dither
func FillPoly   (dst *Surface, pts []image.Point, idx uint8, pat *[8]byte)   // 4 furniture shadows
func XorPattern (dst *Surface, r image.Rectangle, pat [8]byte, mask uint8)   // marquee only
func DrawText   (dst *Surface, x, y int, s string, f *Font, idx uint8)       // y is the baseline
func Present    (fb *Surface, clut *[256]color.RGBA)                         // once per frame
```

Twelve functions. No ambient current-port state, no relocatable handles, no general region engine, no
palette animation, and no scaling outside the asset preprocessor.

---

## Open questions

These are the things this document could **not** settle from the source in front of it. Each entry says
what is known, what is not, why the source cannot answer it, and what a Go port should do in the
meantime. Nothing below is a guess presented as a fact.

### Q1. The exact result of `srcXor` / `patXor` on an 8-bit indexed destination

**Known.** The marquee (§16.3, §16.4) and the glider stencil (§16.7) rely on XOR being an involution:
`Marquee.c:35-51` draws the same shape twice with `PenMode(patXor)` to erase it, and
`Marquee.c:432-439` blits a 1-bit source with `srcXor`. The doc establishes that QuickDraw applies the
XOR modes to the **palette index**, not to RGB components — otherwise the double-draw would not restore
the original pixels, because the CLUT has no XOR-closed structure.

**Unknown.** What value is XOR'd in. For a 1-bit source or an 8×8 pattern, the source "colour" that
gets XOR'd into an 8-bit destination is not 1 — QuickDraw expands the 1-bit source through the port's
fore/back colours first, and the exact expansion for `srcXor` on a `CGrafPort` is a documented
Toolbox behaviour that is not derivable from Glider's own code. Both `0xFF` (invert every bit of the
index) and "the port's foreground index" are consistent with everything observable in the source.

**Why unresolvable here.** No Inside Macintosh, no QuickDraw source, no emulator on this airgapped
machine. The Glider source only ever draws-then-undraws, which any involution satisfies.

**What a port should do.** Pick one, keep it in one place (`XorPattern`'s `mask` parameter in §23.13),
and note that the *only* observable consequences are (a) the visual colour of the marching ants over
each background colour, and (b) the colour of the glider stencil in §16.7. Both are cosmetic editor
feedback; neither affects gameplay or the play-mode renderer. `mask = 0xFF` gives a per-index bitwise
inversion, which over `clut` 128 produces a high-contrast complement for most indices — visually
plausible marching ants.

### Q2. Whether the 4-bit grayscale device CLUT really is `level(i) = (15 - i) * 17`

**Known.** §22.4 shows that 15 of the 20 hand-authored `(4-bit index, 8-bit index)` pairs in
`ObjectDraw.c:16-48` / `ObjectDraw2.c:19-37` are reproduced exactly by
"nearest 4-bit gray to `lum(clut128[eightBitIndex])`" with `lum = (3r + 6g + b)/10` and
`gray4 level(i) = (15 - i) * 17`, and that all 9 pure-gray source colours match exactly. That is strong
evidence the ramp is the 16 multiples of 17 in **descending** order (index 0 = white `0xFF`, index 15 =
black `0x00`).

**Unknown.** Certainty. The 4-bit grayscale CLUT is a **system** resource installed by
`SetDepth(device, 4, 1, 0)`, not a resource in Glider's fork, so it cannot be parsed from
`Glider PRO.r`. The five mismatches (§22.4) are equally consistent with "the ramp is slightly different"
and with "the author eyeballed five of them".

**What a port should do.** Use `(15 - i) * 17` and the luminance formula to *generate* the 4-bit table
rather than transcribing 30 literals, but hard-code the five known divergences so the depth-4 rendering
is byte-identical to the original. Or: skip depth 4 entirely — it is a 1994 concession to grayscale
PowerBooks and nothing in the game logic depends on it.

### Q3. Which port is current when the misplaced `DrawPicture` at `Play.c:590` runs

**Known.** §21.2 proves the bug: the arcade-build block at `Play.c:556-592` restores the saved port with
`SetGWorld(wasCPort, wasWorld)` at `:570`, *before* the `DrawPicture(thePicture, &bounds)` at `:590`.
So PICT 1997 (1536 × 20, the scoreboard strip artwork) is stamped at `hOffset` into whatever port was
current when the block was entered, and `boardSrcMap` is left blank.

**Unknown.** Which port that is. `PlayGame` is called from exactly one place, `NewGame` at
`Play.c:214`, and the last port-setting call before it cannot be identified by inspection: neither
`DumpScreenOn` (`Transitions.c:139-144`, blits with no `SetPort`) nor `InitGarbageRects`
(`Render.c:675-691`, touches no port) sets one, and `StartGliderFadingIn`, `TagGliderIdle`,
`InitTelephone` and the QuickTime block would each have to be traced through every helper they call.
The likeliest candidates are the `mainWindow` port or `workSrcMap`.

**Why it does not matter much.** The caller's very next actions are
`RestoreEntireGameScreen` (`Play.c:800-820`), which paints the whole screen black and calls
`DrawLocale()` + `RefreshScoreboard()`. So any garbage is overwritten within one frame; the worst case
is a one-frame flash.

**What a port should do.** Implement the *intent* (clear the scoreboard surface, publish it, then redraw
the strip artwork into the scoreboard surface at `hOffset`) and record the divergence.

### Q4. Why `clut` 128 and `clut` 129 both exist

**Known.** §22.2: both are 2056 bytes, `ctSeed` 0x00000000, `ctFlags` 0x0000, `ctSize` 255, and
**byte-identical** — zero differing entries, verified with python3. Neither is referenced by any live
code: there is no `GetResource('clut', …)`, no `GetCTable`, no `NewPalette`, no `SetEntries` outside
comments (§22.1).

**Unknown.** Who was supposed to load them and why there are two. The most likely explanation is that
the dead `SetPaletteToGrays` / `WashColorIn` pair (§22.5) was meant to read one as the "colour" target
and the other as scratch, or that 129 is a leftover from an earlier build; but nothing in the source
names either id.

**What a port should do.** Bake `clut` 128 into the binary as the single 256-entry palette (its
structure is fully mechanical and verified entry-by-entry in §5.6) and ignore 129.

### Q5. The `PAT#` 128 double step

**Known.** §5.7, verified with python3: the seven marquee patterns form one 8-row diagonal barber pole,
with inter-pattern row shifts of **2, 1, 1, 1, 1, 1, 1** summing to 8 around the 7-step cycle. Each
pattern is internally a per-row 1-bit left rotation.

**Unknown.** Whether the double step at 1→2 is deliberate (7 frames is a slightly faster, less
regular-looking crawl) or a slip when the resource was authored (a missing 8th pattern would give a
perfectly even 8-step cycle). `Marquee.c:35-51` cycles `index` 0..6 unconditionally, so the source gives
no hint either way.

**What a port should do.** Reproduce the seven measured patterns verbatim. The hitch is one frame in
seven of a 1-pixel-wide dashed outline in the level editor; making it even would be a gratuitous
divergence.

### Q6. The `kMaxViewHeight` clamp breaks the window/offscreen height identity

**Known.** §2.2/§2.3: `houseRect` is clamped to `kMaxViewWidth` = 1536 × `kMaxViewHeight` = 1026
(`GliderDefines.h:267-268`), while `mainWindowRect` is sized to `screen height − 20`. On screens up to
1026 px tall the two agree by construction. On a taller screen the offscreen stays 1026 px while the
window grows, so the window becomes taller than `workSrcMap`.

**Unknown.** What the original actually did on such a display, because in 1994 no shipping Mac ran
taller than 1024 lines in a mode Glider supported, so the path was never exercised. `CopyRectsQD`
(§9.5) would blit `workSrcRect`-clipped rectangles into a taller window, leaving an unpainted band —
but §13.12's `BlackenScoreboard` / second-monitor kludge suggests the author expected *something*.

**What a port should do.** Either clamp the window to the offscreen size (letterbox) or drop the
`kMaxViewHeight` clamp and size the offscreen to the window. Do not leave the identity broken.

### Q7. Fonts

**Known.** §23.6: the renderer uses `systemFont` (0 = Chicago) and `applFont` (1 = Geneva) at sizes 9,
10 and 12, plain and bold, and QuickDraw synthesised bold by smearing the glyph one pixel right. All
text is positioned by **baseline**.

**Unknown.** The actual glyph bitmaps. Chicago and Geneva are `FOND`/`NFNT`/`sfnt` resources in the
System file, not in `Glider PRO.r`. There is no way to recover them from this repository, and the
measured baselines (e.g. the scoreboard's `(0, 9)` white over `(1, 10)` black, §13.7) only pin down the
ascent, not the glyph shapes or the advance widths.

**What a port should do.** Supply substitute bitmap fonts with a 9-px ascent for the 12-pt case and
accept that string widths — and therefore every `StringWidth`-based centring in §13.7, §15.5, §15.6 and
§17.2 — will differ by a few pixels.

### Q8. The exact `Random()` sequence

**Known.** §23.9: `RandomInt(range)` = `(|Random()| * range) / 32768` (`Utilities.c:72-82`), and it can
return `range` itself when `Random()` returns `-32768`. `RandomInt` is called 15 times in the rendering
subsystem alone (sparkles, flying points, shreds, star placement).

**Unknown.** The Toolbox `Random()` algorithm. It is a 16-bit LCG in ROM seeded from the low-memory
global `randSeed`; the constants are not in this repository.

**Why it matters.** The recorded auto-demo (`demo` 128, replayed by `GetDemoInput`, `Input.c:224`
matching `gameFrame == demoData[demoIndex].frame`) is a keystroke stream, not a state dump. If the port's
RNG diverges, any randomised object the demo interacts with desynchronises and the attract-mode demo
plays wrong.

**What a port should do.** If demo playback matters, implement the exact 68k `Random()`; otherwise use
any PRNG and disable the recorded demo. Note that only *visual* randomness (sparkles, shred
trajectories) is fed by `RandomInt` in the renderer — the gameplay-critical uses are elsewhere.

### Q9. `DrawClockHands`'s ambient foreground colour

**Known.** §23.4.2: `DrawLargeClockHands` explicitly sets `ForeColor(whiteColor)` (`ObjectDraw.c:1185`)
and restores `blackColor` (`:1291`). `DrawClockHands` — used by `kBlueClock` and `kYellowClock` — sets
**no** foreground colour at all, so it draws the hands in whatever colour `backSrcMap`'s port currently
holds.

**Unknown.** With certainty, what that colour is at each call site. In practice every `Color*` wrapper
restores the previous colour and `DrawLargeClockHands` leaves `blackColor`, and object drawing order
within a room is deterministic, so it is almost certainly black — but it depends on the full object
draw order for the room (§7.6) and on which prizes precede the clock.

**What a port should do.** Pass the index explicitly: white for `kCuckoo`, black for `kBlueClock` and
`kYellowClock`.

### Q10. House-movie codecs and frame rates

**Known.** §19: the TV set composites a QuickTime movie from the house's data fork, bound with
`SetMovieGWorld((CGrafPtr)mainWindow, nil)` (`GliderPRO/Sources/Play.c:150`, §19.4) and clipped with
`SetMovieDisplayClipRgn`, serviced once per game frame by `MoviesTask(theMovie, 0)`
(`Play.c:490-493`). `Houses/*.mov` are the data forks.

**Unknown.** The codecs, dimensions and frame rates of the shipped movies — the `.mov` files were not
decoded for this document, only the code path was.

**What a port should do.** Treat the TV as an optional capability (`hasQT`, §19.7), and drive playback
from the **frame counter**, not a wall clock, so the movie stays in lockstep with the 30 fps game
(§19.7 step 5). Decode the actual `.mov` files before committing to a video backend.

### Q11. Whether `transparent` mode keys on white or on the port's background colour

**Known.** §8.3 and §23.1.1: the two `transparent` blits (`ObjectDraw2.c:1403`, `:1430`) are documented
as "white-keying", and index 0 in `clut` 128 is `#FFFFFF` (§5.6), and `CreateOffScreenGWorld` erases the
temp GWorld with `EraseRect` — which fills with the port's **background** colour — before `LoadGraphic`
draws into it.

**Unknown.** Whether a fresh GWorld's background colour is guaranteed white. `CreateOffScreenGWorld`
never calls `BackColor`, so it inherits whatever `NewGWorld` initialises the port to. It is almost
certainly white (index 0), which makes "background" and "white" the same thing here, but the source does
not state it.

**What a port should do.** Key on index 0. If a house's custom PICT legitimately contains white pixels
that should be opaque, they will be dropped — which is exactly what the original does, and is why
`DrawCustPictSansWhite` is named that way.

### Q12. Two `evenFrame` writes with no reader

**Known.** §20.5: `Dynamics3.c:474` (`case kBall:`) and `:524` (`case kFish:`) assign `evenFrame = true`
inside `AddDynamicObject`, but the loops below them use a *local* `lilFrame` toggle and never read
`evenFrame`. `Dynamics2.c:420` also sets it whenever a ball starts moving, and that one *is* observable
— it resets the flames/stars interleave phase and the gravity parity.

**Unknown.** Whether the `Dynamics3.c` pair are vestigial (the loop was rewritten to use `lilFrame`) or
whether resetting the global at room-load time was intentional so that a room always starts on a known
parity.

**What a port should do.** Derive `evenFrame` from `gameFrame & 1` and drop all three writes, then
verify the flame/star phase and the ball gravity against a recording. §20.5 documents exactly what
changes.

---

## Porting notes

The actionable summary. Ordered by how much of the port depends on getting it right.

### P1. One 8-bit indexed framebuffer, one palette, one conversion at the end

The entire renderer works in **palette indices**, never in RGB. Every surface is `[]uint8`; the only
RGB in the pipeline is the final `Present(fb, clut)` that expands indices to RGBA once per frame
(§23.13). Bake `clut` 128 (§5.6, structure verified entry-by-entry, 256 distinct RGB triples with zero
duplicates) as a `[256]color.RGBA` constant.

Do **not** convert sprites to RGBA at load time. Three mechanisms depend on index identity:
white-keying on index 0 (§8.3), the `patXor` marquee (§16.3), and the depth-4 degradation tables
(§22.4). An RGBA pipeline breaks all three.

### P2. Three surfaces, one shared coordinate system

`backSrcMap` (static room), `workSrcMap` (composited frame) and the on-screen window all use the
**same** coordinate system (§1.2, §2.6) — the transform is a pure translation, and `CopyBits` is never
asked to scale. Keep it that way: `srcRect == dstRect` in every blit. This single invariant is what
makes the dirty-rect scheme (§9) a two-line loop.

There is no scaling anywhere in play mode. The only scaling in the program is `LoadScaledGraphic`
(§4.3), used for splash/high-score/banner backdrops, and it should be done in the asset preprocessor.

### P3. The dirty-rect double buffer, and its publish-then-erase order

`CopyRectsQD` (§9.5, §20.3) does the two queues in this order and it is not interchangeable:

1. `work2MainRects[0..numWork2Main-1]`: `workSrcMap` → window (**publish**)
2. `back2WorkRects[0..numBack2Work-1]`: `backSrcMap` → `workSrcMap` (**erase for next frame**)

Then both counters are zeroed. `kMaxGarbageRects` = 48 (`Render.c:20`) but the guard is
`< (kMaxGarbageRects - 1)` (`Render.c:67`, `:86`), so **at most 47 rects per queue are ever stored** and
slot 47 is dead (§9.4). Overflow past 47 is **silently dropped**: `AddRectToWorkRects` clamps the rect
to `justRoomsRect` and simply returns if the queue is full — no error, no fallback full-screen blit. A
Go port with a growable slice will render *more* than the original in heavy scenes; that is a
divergence, and probably a desirable one, but it must be a deliberate choice, because it changes which
pixels are stale in a room with many simultaneous dynamics.

### P4. Non-accumulating 30 fps, and integer frame counters everywhere

`kTicksPerFrame` = 2 (`GliderDefines.h:533`) against a 60 Hz tick → **30 fps**, 33.3 ms budget. The
limiter spins *before* publishing and then sets `nextFrame = TickCount() + kTicksPerFrame` from the
**current** time, not from the missed deadline (§20.3). So:

- a slow machine runs the whole game in slow motion, in lockstep — no frame dropping, no catch-up;
- every animation counter and every physics constant is an integer frame count tuned for exactly 30 fps.

Use a monotonic clock but keep the integer-tick abstraction. **Do not** switch to floating-point delta
time. Keep `(delay * 6) / kTicksPerFrame` as a formula (§20.6) so a future frame-rate change does not
silently retime every house.

### P5. Replace the ambient current port with an explicit destination

178 `SetGWorld` calls against 62 `GetGWorld` (§23.2). Every draw helper (`ColorLine`, `ColorRect`,
`DrawString`, …) takes no destination and writes to the last-set port. Thread the destination
explicitly — and then handle the two places the original *relies* on the leak: the scoreboard "High"
position (§13.11) and `RefreshRoomTitle` (§17.2).

Also: `CopyBits` clipping applies the port's `visRgn`/`clipRgn` only when the destination is the current
port's own pixmap (§23.1.2). Make clipping an explicit argument so this asymmetry disappears; the only
code that needs it is `DrawReflection`'s `mirrorRgn` (§12.5).

### P6. Preprocess every asset; parse no big-endian data at run time

Decode all 152 PICTs, `clut` 128, `PAT#` 128, the `WIND` bounds and the `STR#` tables once, offline,
into native structs (§5, §23.11). Specific traps:

| Trap | Fix |
|---|---|
| `picSize` is signed and wrong for 19 PICTs (§5.9 #1) | use the resource length, never `picSize` |
| PICTs 5006 and 5010 have non-zero `picFrame` origins (§5.9 #5) | replicate `LoadGraphic`'s `OffsetRect(&bounds, -bounds.left, -bounds.top)` |
| PICT 4001 is 57 rows shorter than its GWorld (§5.9 #3) | zero-fill the tail |
| PICT 1995 is 640×460 drawn into 640×480 (§5.9 #7) | stretch by 480/460 = 1.0435 |
| PICT 1020 breaks the `+1000` mask convention (§5.9 #10) | special-case the angel mask |
| PICT 5003 does not exist (§5.9 #2) | correct — switches need no mask |
| `Point` is `{v, h}`; `Rect` is `{t, l, b, r}`; `ctSize` is the last index (§23.11) | |

### P7. Only twelve drawing primitives are needed

§23.13 lists them. Notably **absent**: a general region engine, palette animation, alpha blending,
scaling, and rotation. Notably **present** and easy to overlook:

- a polygon filler for the **four furniture drop-shadow regions** (§23.4.1) — hexagons/pentagons with
  45° edges, filled through a 50 % checkerboard in `patOr` mode, so they are a **dither, not a blend**;
- a diagonal line rasteriser for the **clock hands** (§23.4.2) — the only diagonal drawn pixels in the
  renderer; all 153 `ColorLine` calls are axis-aligned (verified);
- a pattern XOR for the **marquee** (§16), pattern-aligned to the surface origin, not to the shape.

### P8. Reproduce the dead code's *absence*, not its intent

Glider PRO 1.0.4's Carbon port left a consistent trail: **the preference survives, the implementation
was deleted**. A port that "restores" any of these is not being faithful, it is inventing:

| Subsystem | Evidence | Status in 1.0.4 |
|---|---|---|
| Dissolve transitions | `DissBits` / `DissBitsChunky` (§14.5) | removed from the build; only wipe and hard cut are reachable |
| Gray-to-colour fade | `SetPaletteToGrays` / `WashColorIn` / `HardDrawMainWindow` (§22.5, §22.6) | dead; `HardDrawMainWindow` would not even compile — `thisMac.menuHigh` no longer exists |
| Menu-bar hiding | `HideMenuBarOld` / `ShowMenuBarOld` (§21.1) | every call site commented out with `// TEMP`; a 20-px black strip is painted instead |
| Window zoom | `ZoomBetweenWindows` (§23.7) | whole function inside `/* */`; `ZoomRectToRect` has no definition |
| `RedrawSplashScreen` | copies main→work where work→main was meant (§15.4) | ships broken |
| High-score screen | composes offscreen and discards (§15.6) | ships broken |

### P9. Things that look like bugs and must be preserved anyway

| Behaviour | Where | Why it must stay |
|---|---|---|
| `DrawLighting` is a no-op | §6.9 | rooms are lit or unlit, never gradated |
| `isLit` gating is not uniform — prizes draw in the dark | §7.7 | changing it changes which objects are visible in dark rooms |
| Flames and stars each animate at **15** fps on alternating frames | §20.3 step 5, §20.5 | a real performance decision; halves the cost of the two heaviest renderers |
| Gravity is `vVel++` only on even frames = 0.5 px/frame² | §20.5 | every jump arc and every ball roll in every house is tuned to it |
| Scoreboard meters blink 5-on/3-off, 4/2/0 frames out of phase | §20.7 | what stops a low-everything scoreboard strobing |
| Clock hands freeze at room-entry time | §23.4.2 | the hands are in `backSrcMap`, composited once |
| `DrawDeckTable`'s shadow has no depth-4 branch | §23.4.1 site 6 | one of the five 4-bit inconsistencies (§22.4) |

### P10. Suggested build order

1. `Surface`, `Blit`, `BlitMask`, `FillRect`, `Present` + the baked `clut` 128. Verify by decoding
   PICT 1000 (the splash) and putting it on screen.
2. The geometry constants (§2.1) and `localRoomsDest[9]` (§2.5). Verify against the three worked
   examples in §2.7–§2.9 — 640×480 must give `localRoomsDest[kCentralRoom].top == 79`.
3. `backSrcMap` composition: `DrawLocale` → `DrawRoomBackground` → tiles → `DrawFloorSupport` (§6).
   A static room should now be correct.
4. The object layer: `srcRects[144]` (§7.4), the sheet table (§7.5), the four masking strategies (§8),
   and `DrawARoomsObjects` (§7.6).
5. The dirty-rect queues (§9) and `RenderFrame`'s 16 steps (§10, §20.3). The game is now animating.
6. The scoreboard (§13), then transitions (§14), then the full-screen set pieces (§15).
7. The editor surfaces last: marquee (§16), coordinate windoid (§17), map window (§18).
8. QuickTime (§19) and depth-4 degradation (§22) only if you want them.

---

## Appendix Z. Erratum (appended 2026-09-10 while writing `graphics-assets.md` §6)

This section is appended by the sprite-atlas audit in
[`docs/analysis/graphics-assets.md`](graphics-assets.md) §6.8. It corrects one factual error above
and is deliberately kept separate from the body so the body's line numbering is stable for the
citations in the sibling documents.

### Z.1 `kBBQ`'s art is PICT **3988**, not 3958

Two sentences in §17.4 give `kBBQ` (object type `0x0C`) the wrong picture ID:

| location | says | should say |
|---|---|---|
| `docs/analysis/rendering.md:2400-2402` | "`kBBQ` (64x33, drawn from PICT **3958** via `DrawPictSansWhiteObject`)" | PICT **3988** |
| `docs/analysis/rendering.md:2438` | "its art comes from PICT **3958** via `DrawPictSansWhiteObject`" | PICT **3988** |

The two `#define`s live 27 lines apart in the same block of `ObjectDraw2.c`:

```c
#define kCobwebPictID           3958        /* GliderPRO/Sources/ObjectDraw2.c:67  */
#define kBBQPictID              3988        /* GliderPRO/Sources/ObjectDraw2.c:94  */
```

and are consumed in the same `switch`:

```c
    pictID = kCobwebPictID;                 /* GliderPRO/Sources/ObjectDraw2.c:1268 -> kCobweb  0x79 */
    pictID = kBBQPictID;                    /* GliderPRO/Sources/ObjectDraw2.c:1316 -> kBBQ     0x0C */
```

So **3958 is `kCobweb`'s art and 3988 is `kBBQ`'s.** Measured sizes agree with that assignment and
not with the erroneous one: PICT 3958 is 54 x 45, exactly `srcRects[kCobweb]`, while PICT 3988 is
64 x 33, exactly `srcRects[kBBQ]` (`GliderPRO/Sources/StructuresInit2.c:330`).

The error is localised to those two sentences. Everywhere else this document has 3958 right:
`:1344` measures PICT 3958 as 54 x 45 in the PICT inventory, and `:2756` maps `kCobweb` to it in the
masking table. The compositing path named in both sentences is also wrong as a consequence — `kBBQ`
does use `DrawPictSansWhiteObject` (white colour key), but `kCobweb` uses `DrawPictWithMaskObject`
with a real mask, PICT **3927**, so the two objects are not interchangeable in an extractor.

Nothing else in §17.4 is affected: the "size template only" classification of `kBBQ` is correct
(it has a `QSetRect` but no `QOffsetRect`, so its `srcRects` entry is a size, not a sheet
coordinate), and its four animated coal frames really do come from `blowerSrcMap` (PICT 4000) at
`coalsSrc[0..3]`, 32 x 9, stride 9, origin (0,304) — see `graphics-assets.md` §6.3 and §6.7.

### Z.2 Checked and found correct (no change needed)

For the record, the atlas audit re-verified two other claims in this document that looked like
candidate errors and found both correct, so they must **not** be "fixed":

- `:679-681` — "the lowest-reaching source rect is `srcRects[kStool]` (48 x 38 at y 183..221)". This
  is right. `deckSrc` (64 x 21 at y 162..183, `GliderPRO/Sources/StructuresInit.c:331-332`) is one row
  higher, and `kStool`'s rect at `:349-350` really is the bottom-most cell of `furnitureSrcMap`
  that any `srcRects[]` entry points at.
- `:2230` onwards — the `srcRects[]` walkthrough's group boundaries and its statement that the array
  is allocated with `NewPtr` rather than `NewPtrClear` (`GliderPRO/Sources/StructuresInit2.c:271`),
  so the 28 unassigned slots hold uninitialised heap bytes.

---

## Appendix ZB. Erratum + cross-reference (appended 2026-09-10 while writing `object-draw-all.md`)

Added by the `object-draw-all.md` pass, which read all 966 lines of
`GliderPRO/Sources/ObjectDrawAll.c`. Appended rather than edited in place so that nothing above moves.
Full detail, with the corpus measurements behind it, is in
`docs/analysis/object-draw-all.md` §13.

### ZB.1 `:2188-2190` — `kVertLocalOffset` is **322**, not 366

The text says `kVertLocalOffset` = `kTileHigh + kFloorSupportTall` = 322 + 44 = 366, and that
`VerticalRoomOffset` therefore returns ∓366. The header says otherwise:

```
#define kTileHigh              322      /* GliderPRO/Headers/GliderDefines.h:498 */
#define kRoomWide              512      /* :499  // kNumTiles * kTileWide       */
#define kFloorSupportTall       44      /* :500                                 */
#define kVertLocalOffset       322      /* :501  // kTileHigh - 39 (was 283, then 295) */
```

The value is **322**, numerically equal to `kTileHigh`; the trailing comment is stale history from when
the room was 283 and then 295 pixels tall. `VerticalRoomOffset`
(`GliderPRO/Sources/ObjectRects.c:1067-1088`) returns `-322` for `kNorthRoom`/`kNorthEastRoom`/
`kNorthWestRoom`, `+322` for `kSouthEastRoom`/`kSouthRoom`/`kSouthWestRoom`, `0` otherwise, and
`OffsetRectRoomRelative` (`:1093-1131`) shifts by the same `kVertLocalOffset`. Rooms therefore tile
**flush** with no gutter; with 366 every vertical seam in every `backSrcMap` composite would be 44 px
wrong. `kFloorSupportTall` = 44 is the *overhang* `DrawFloorSupport` paints **into** the neighbouring
room's band, not a gap between rooms.

### ZB.2 `:2192-2193` — two of the six listed helpers do **not** get `VerticalRoomOffset`

The list "`DrawTiki`, `DrawTable`, `DrawDeckTable`, `DrawStool`, `DrawMailboxLeft`, `DrawMailboxRight`
all take `playOriginV + VerticalRoomOffset(neighbor)`" is wrong for two entries. The six scalar-origin
call sites in `ObjectDrawAll.c` are exactly:

| Call site | Argument | Corrected? |
|---|---|---|
| `:177` `DrawTiki(&itsRect, playOriginV + VerticalRoomOffset(neighbor))` | corrected | yes |
| `:208` `DrawTable(&itsRect, playOriginV)` | **bare `playOriginV`** | **no** |
| `:259` `DrawDeckTable(&itsRect, playOriginV)` | **bare `playOriginV`** | **no** |
| `:266` `DrawStool(&itsRect, playOriginV + VerticalRoomOffset(neighbor))` | corrected | yes |
| `:487` `DrawMailboxLeft(…, playOriginV + VerticalRoomOffset(neighbor))` | corrected | yes |
| `:493` `DrawMailboxRight(…, playOriginV + VerticalRoomOffset(neighbor))` | corrected | yes |

`DrawShelf` (`ObjectDraw.c:263`), `DrawCabinet` (`:358`), `DrawCounter` (`:498`) and `DrawDresser`
(`:643`) take **no** origin argument at all — `grep -n playOrigin ObjectDraw.c` returns nothing — so
they derive everything from the already-room-corrected rect and are right in all nine slots.

Consequence, and it is a shipped-content-visible bug, not a nit: for a `kTable` or `kDeckTable` in the
N/NE/NW/S/SE/SW slots, `DrawTable`'s pedestal legs (`ObjectDraw.c:214-227`, guarded by
`tableTop->bottom < kTableBaseTop + down` with `kTableBaseTop` = 296), its 64 x 22 base
(`:253-258`) and its shadow ellipse (`kTableShadowTop` = 312, `:177-180`) are all computed against the
**central** room's floor, 322 px away. North-side rooms show a stray 5-line pedestal stripe; south-side
rooms show a table top with no pedestal, base or shadow at all. All **200** `kTable`+`kDeckTable`
objects in the 22-house shipped corpus live in vertically-connected rooms, so this affects 100 % of
them. A faithful port must reproduce the uncorrected calls. Worked geometry, per-slot symptom table and
per-house counts: `object-draw-all.md` §5.2.

### ZB.3 Cross-reference for §7 (line 2126)

§7 of this document is the summary treatment of `DrawARoomsObjects`; `object-draw-all.md` is the
exhaustive one. Map between them:

| §7 here | `object-draw-all.md` |
|---|---|
| 7.1 rect pipeline / room offsets | §5.1, §5.3 |
| 7.2 the clipping test | §5.4, §10.6 (23 of the 74 `case` groups never call `SectRect`; 14 of those do real work unguarded) |
| 7.3 `IsThisValid` | §3.2 |
| 7.6 dispatcher table (verified correct) | §10.2, all **74** `case` groups with line ranges, registrars and `dynamicNum` |
| 7.x `isLit` gating | §5.5, §10.6 |
| 7.x `masterObjects[]` write-back | §7.2 (`ObjectDrawAll.c:953-961`) |
| play-vs-edit split | §6.1 (49 helpers / 55 call sites) and §6.1's "no draw at all" table (16 of 74 groups) |

Three things §7 does not cover that a porter needs: the **inter-room** draw order
(NW, NE, N, SW, SE, S, W, E, Central — `RoomGraphics.c:80-120`, proved in §4), the fact that
`dinahs[]` indices and therefore simulation order are assigned during this draw and so depend on
`numNeighbors` and screen resolution (§7.4, §7.9), and what happens when the 18-slot
`kMaxDynamicObs` / 24-slot `kMaxSavedMaps` / 56-slot `kMaxHotSpots` budgets are exhausted mid-draw
(§9 — measured overflow in 46 rooms across 9 shipped houses at the maximum 1536 x 1026 view).

---

# Appendix Z. Open questions Q1, Q2, Q7 and Q8 closed

*Appended section. Added by the toolbox-primitives investigation; nothing above this line was
modified. Full evidence, tables and pseudocode live in
`docs/analysis/toolbox-primitives.md`. Section numbers below refer to that document.*

## Z.1 Q1 (`srcXor` / `patXor` on an 8-bit indexed destination) - CLOSED

Q1 offers two candidate semantics (XOR of RGB values vs XOR of palette indices) and cannot
choose between them. **They coincide, and the answer is `dest ^= 0xFF`.**

The tree contains **exactly one** `srcXor`, at `GliderPRO/Sources/Marquee.c:438`
(`DrawGliderMarquee`). Its source is `blowerMaskMap`, created at **depth 1**
(`GliderPRO/Sources/StructuresInit.c:258`, literal third argument `1` to
`CreateOffScreenGWorld`). Classic QuickDraw colourises a 1-bit source into a deeper destination
*before* the boolean op: set bits become the port foreground, clear bits the port background.
The boolean op then runs on the destination's **pixel values**, i.e. palette indices.

Foreground is invariantly black and background white at that call: `DrawGliderMarquee` and all
three of its callers set no colour, and every helper in `GliderPRO/Sources/ColorUtils.c`
brackets its draw with `GetForeColor` / `RGBForeColor(wasColor)`. On `clut` 128 (parsed:
index 0 = `FFFF FFFF FFFF`, index 255 = `0000 0000 0000`) that is index 255 and index 0.

Therefore: **set source bit -> `dest ^= 0xFF`; clear source bit -> `dest ^= 0x00` (no-op)**.
Verified for all 256 indices: `i ^ 255 == 255 - i` and `(i ^ 255) ^ 255 == i`.

Two facts §16.3/§16.4/§16.7 do not record:

* `DrawMarquee` itself re-XORs the stencil at `Marquee.c:493-494`, and `DrawMarquee` runs on both
  the erase and the draw pass of every ant-march tick, so **each blink applies an even number of
  XORs** to the glider stencil. Optimising the erase pass away makes it flicker.
* `StopMarquee` (`Marquee.c:141-144`) XORs the stencil *before* `SetPortWindowPort(mainWindow)`
  and before re-establishing `patXor`.

Full transfer-mode census (§2.1 of toolbox-primitives.md): srcCopy 119, patXor 15, patOr 6,
transparent 2, srcXor 1, srcOr 1 (the anomalous `PenMode(srcOr)` at `ObjectEdit.c:2364`, which
means `patOr`). `PAT#` 128 is 58 bytes / 7 patterns; the rotation order is +2 rows for 0 -> 1
then -1 row per step, not a uniform rotation.

## Z.2 Q2 (whether the 4-bit CLUT is `level(i) = (15 - i) * 17`) - CLOSED, and PROVED

**Yes.** `clut4[i].rgb = ((15-i)*0x1111)` on all three channels, for `i` in 0..15.

The proof does not depend on any assumption about the system's CLUT. The standard 8-bit palette
(`clut` 128, parsed: 2056 bytes, `ctSeed` 0, `ctFlags` 0x0000, `ctSize` 255) contains **exactly
one index per 4-bit gray level**:

```
gray8 = [0, 245, 246, 43, 247, 248, 86, 249, 250, 129, 251, 252, 172, 253, 254, 255]
        (levels 0xFF 0xEE 0xDD 0xCC 0xBB 0xAA 0x99 0x88 0x77 0x66 0x55 0x44 0x33 0x22 0x11 0x00)
all 16 distinct: True
```

Of the 13 authored `(i4, i8)` pairs whose `i8` is a pure gray, **11 satisfy
`gray8[i4] == i8` exactly** (`i4` in {1,2,3,4,5,7,10,11,13,14,15}). The two misses are
`(15, 254)` - `#111111`, which is `gray8[14]`, at 5 inline drop-shadow sites - and `(8, 251)` -
`#555555`, which is `gray8[10]`, at 1 site. Eleven exact identity hits admit no competing ramp
hypothesis. This also explains why `k8LtstGray3Color` = 43 and `k8DkGray3Color` = 172 come from
the colour cube rather than the contiguous 245-254 extra-gray run.

**§22.4's 20-row table is superseded** by a 71-site / 22-unique-combo table (§3.5, §3.6 of
toolbox-primitives.md) extracted from all 33 `thisMac.isDepth == 4` sites. The two combos §22.4
does not have are `(2, 246)` and `(3, 43)`.

**Two corrections to §22.4's luminance column** (using the game's own formula from the dead
`SetPaletteToGrays`, `MainWindow.c:454-498`, which is **three separate integer divisions**
`(r*3)/10 + (g*6)/10 + (b*1)/10`):

| RGB | §22.4 | correct |
|---|---|---|
| `#CC9933` | 158 | **157** |
| `#996633` | 112 | **111** |

**Nine candidate derivation rules were scored** against 22 combos / 71 sites / 13 pure grays:

| rule | combos | sites | pure grays |
|---|---|---|---|
| R1 nearest level, `lum = (r*3)/10 + (g*6)/10 + (b*1)/10` | **17** | **54** | **11** |
| R2 nearest level, NTSC 0.299/0.587/0.114 | 17 | 54 | 11 |
| R7 round, `15 - (lum+8)/17` | 17 | 54 | 11 |
| R9 nearest level, 16-bit `(3R+6G+B)/10` | 17 | 54 | 11 |
| R3 nearest level, `(r+g+b)/3` | 15 | 47 | 11 |
| R5 nearest level, `(max+min)/2` | 13 | 42 | 11 |
| R8 nearest level, `min(r,g,b)` | 13 | 38 | 11 |
| R4 nearest level, `max(r,g,b)` | 11 | 34 | 11 |
| **R6 truncate, `15 - lum/17`** | **7** | **26** | **3** |

R6 (truncation) is decisively wrong and is the most likely accidental error in a port. R1/R2/R7/R9
tie; R1 and R9 disagree on exactly 1 of 256 palette entries (index 84 `#9999FF`).

**No rule can reach 22/22, because the relation is not a function in either direction:**
`i8` 223 -> `i4` {14, 15}; `i8` 251 -> {8, 10}; `i8` 254 -> {14, 15}. The 33 conditional sites
must be carried literally.

**39 unguarded palette indices >= 16** exist and are undefined behaviour at depth 4 (8 literals:
`GameOver.c:65` and `:84` = 244, `ObjectDraw.c:128` and `:141` = 192, `ObjectDraw.c:1390` = 227,
`ObjectDraw2.c:295` = 32, `ObjectDraw2.c:593` = 17, `Tools.c:154` = 171; plus 31
`kRedOrangeColor8` = 23 sites). `kRedOrangeColor8` is 23 = `#FF6600`; the `// actually, 18`
comment at `GliderDefines.h:542` is wrong (index 18 is magenta `#FF66FF`).

Only two `clut` resources exist (128 and 129), they are byte-identical, and `GetCTable` is called
**zero** times, so neither is referenced by code. The 4-bit CLUT is not in the application at
all - it is a system artefact, hence the normative substitution.

## Z.3 Q7 (fonts) - CLOSED via option (b)

Option (a) is impossible: **zero** `FOND`, `NFNT`, `FONT`, `sfnt`, `fdsc` or `fmtx` resources
exist among the 538 resource records. Glyph bitmaps are unrecoverable from this release.

Option (b) is supplied by §4.5-4.10 of toolbox-primitives.md:

| key | substitute | ppem | asc | desc | lead | lineHeight | digit adv |
|---|---|---|---|---|---|---|---|
| CHI12 (Chicago 12 plain) | LiberationSans-Bold.ttf | 13 | 12 | 3 | 1 | 16 | 7 |
| GEN9 (Geneva 9 bold) | LiberationSans-Regular.ttf | 9 | 9 | 2 | 0 | 11 | 5 |
| GEN12 (Geneva 12 bold) | LiberationSans-Regular.ttf | 11 | 10 | 3 | 1 | 14 | 6 |
| GEN14 (Geneva 14 bold) | LiberationSans-Regular.ttf | 13 | 12 | 3 | 1 | 16 | 7 |

with full per-character advance tables for all four, and QuickDraw synthetic bold as **+1 px per
character**.

**Correction to §23.6:** the sizes used are **9, 12 and 14**, not "9, 10 and 12". Census of the
16 `TextSize` calls: 9 x7, 12 x8, 14 x1 (the single 14 is `HighScores.c:142`). There is no
`TextSize(10)` anywhere in the tree.

The decisive structural fact: **`GetFontInfo` and `FontMetrics` are called zero times**, so every
vertical position in the game is a hard-coded baseline. Only `TETextBox` (`DialogUtils.c:642`,
Geneva 9) consults font ascent. Also zero: `TextMode`, `CharExtra`, `SpaceExtra`, `GetFNum`,
`RealFont`, `SetFractEnable`, `TruncString`, `TruncText`, `StdTxMeas`, `MeasureText`. There are
11 `StringWidth` sites and 1 `TextWidth` site (`GameOver.c:102`).

Validation: **230** single-line DITL StaticText/Button/CheckBox/RadioButton items fit under CHI12
with **zero** raw-box overflows (tightest +2 px); under a conservative button/checkbox padding
assumption exactly one goes negative by 1 px (`Don't Lock`, DITL 1029). Of 40 hard-coded
game-site measurements, 11 are negative - **10 are genuine original overflows** and 1
(`SEPTEMBER`, 65 px in a 64-px calendar field) is substitute-induced.

Two anchors pin GEN12 to 11 ppem exactly: 9 digits at 6+1 px = **63 px in the 64-px
`boardPSrcRect`** score field, and ink descent 2 fitting the 12-px `boardTSrcRect` room-title
strip whose baseline is at v = 10.

Also relevant to §23.6: `ictb` 150 is the only `ictb` and is **36 bytes, all zero** (= no
override), and `DLGX` 150 is the only `DLGX` (190 bytes, Pascal string `08 "Charcoal"` then 63
shorts from offset 0x40). So **no dialog item in the game overrides the port default font**;
all 428 DITL items render in CHI12.

## Z.4 Q8 (the exact `Random()` sequence) - CLOSED, and de-risked

Specified: Lehmer / Park-Miller minimal standard on `qd.randSeed`, `a` = 16807 (0x41A7),
`m` = 2147483647 (0x7FFFFFFF), Schrage `q` = 127773 / `r` = 2836, return = low 16 bits of the new
state reinterpreted as `int16`. Initial seed **1**, because `GliderPRO/Prefix.h` sets
`TARGET_CARBON 1` and so compiles `GetDateTime((UInt32 *)&qd.randSeed)` (`Utilities.c:61`) out of
the shipped build.

`GliderPRO/CarbonLib` **exports** `Random` but contains **zero code sections** (all four PEF
containers have `sectionCount` 1 and that section is kind 4 = loader), so the 1994 algorithm is
not in the repository and a normative choice is unavoidable.

The seed-1 stream's first 24 `Random()` values, for CI assertion:

```
16807, 15089, -21287, 3114, -18558, -9528, -28968, 2558, 12099, 1101, -26472, 15445,
4748, -9246, -11085, 14151, 14615, 16657, -15464, 18772, 11823, 11025, -27091, -29947
```

**Two traps.** (1) `RandomInt(range)` returns `range` itself when the raw word is `0x8000`, so the
range is **inclusive**; `rand.Intn` is not equivalent. (2) A naive `int32`
`(state * 16807) % 2147483647` overflows for `state > 127773` and diverges from Schrage within two
draws (172226 of the 299999 seeds in [1, 299999] differ).

**Why Q8 matters less than §16-§22 imply.** The RNG feeds exactly five arrays - `flames[]`,
`tikiFlames[]`, `bbqCoals[]`, `pendulums[]`, `theStars[]` - written only in `DynamicMaps.c`
(:337-341, :421-426, :507-511, :592-603, :684-690) and read only in `Render.c`
(:202-216, :221-235, :240-254, :278-317, :429-445), where every read selects a `CopyBits` source
rect. `GliderPRO/Sources/Player.c` contains **no** RNG call at all. So a wrong `Random()` can
only change animation phase, the game-over page flutter, the flower species, the telephone
schedule and the splash variant - never a trajectory, a collision, a score or a star count.
