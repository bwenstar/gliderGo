# Glider PRO 1.0.4 — `ObjectDrawAll.c` / `DrawARoomsObjects`: object draw order, dynamic-object registration, and budget exhaustion

## Scope

This document is a complete reverse-specification of one function:

```c
void DrawARoomsObjects (short neighbor, Boolean redraw);   /* GliderPRO/Sources/ObjectDrawAll.c:23-965 */
```

`ObjectDrawAll.c` is 966 lines and contains nothing else — no other function, no file-scope
variables, only four `extern` declarations. Everything below is about that one function, its
callers, its callees, and the invariants it establishes for the rest of the engine.

It is the single most load-bearing function for a faithful port, because it is simultaneously:

1. the **static object compositor** (it paints every non-animated object into `backSrcMap`),
2. the **animation registrar** (it creates every entry in `dinahs[]`, `flames[]`, `tikiFlames[]`,
   `bbqCoals[]`, `pendulums[]`, `theStars[]`, `grease[]`, `savedMaps[]`, `tempManholes[]`, and the
   mirror region),
3. the **index authority** — the order in which it registers those entries *is* the order in which
   `HandleDynamics` simulates them and `RenderDynamics` draws them, and it is the mapping that
   `masterObjects[].dynaNum` records for triggers and switch links, and
4. the **QuickTime movie binder** (it decides which TV in the house gets the movie).

The six questions this document is required to answer are answered in dedicated sections:

| Question | Section |
|---|---|
| (a) exact draw order within `DrawARoomsObjects`, object index 0..23 | [§3](#3-question-a--the-draw-order-within-a-room-object-index-023) |
| (b) draw order across the nine rooms, **proved** | [§4](#4-question-b--the-draw-order-across-the-nine-rooms-proved) |
| (c) drawn into `backSrcMap` once at room load vs re-drawn every frame | [§6](#6-question-c--drawn-once-into-backsrcmap-versus-re-drawn-every-frame) |
| (d) exact point `dinahs[]` entries are registered, and why draw order ⇒ simulation order | [§7](#7-question-d--where-dinahs-entries-are-registered-and-why-draw-order-fixes-simulation-order) |
| (e) the `(neighbor == kCentralRoom) && (!tvInRoom)` first-TV rule, and how `tvWithMovieNumber` is chosen | [§8](#8-question-e--the-first-tv-rule-and-tvwithmovienumber) |
| (f) what happens when the 18-slot `kMaxDynamicObs` or the 56-slot `kMaxHotSpots` budget runs out mid-draw | [§9](#9-question-f--budget-exhaustion-mid-draw) |

Cross-references: `docs/analysis/rendering.md` §7 ("The object layer") tabulates the dispatcher
case-by-case and is the companion to this document; §7.6 there is the *table*, this document is the
*semantics*. `docs/analysis/determinism.md` §3.3 asserts the draw-order ⇒ simulation-order coupling;
§7 below is the proof of it. `docs/analysis/object-dynamics.md` covers what each `dinahs[]` entry
does once created; this document covers only how it gets created. Two factual corrections to
existing docs are recorded in [§13](#13-corrections-to-existing-documents-in-docsanalysis).

All line numbers are for the CR→LF-converted copies of `GliderPRO/Sources/*.c` and
`GliderPRO/Headers/*.h` (the originals are classic-Mac CR-only text; `tr '\r' '\n'` reproduces the
numbering used here). Note that `/usr/bin/grep` on this machine is ugrep, which treats the
ISO-8859/MacRoman headers as binary and silently skips them — `grep -a` is required.

## Sources read

Fully read:

| File | Lines | Why |
|---|---|---|
| `GliderPRO/Sources/ObjectDrawAll.c` | 966 | the subject |
| `GliderPRO/Sources/RoomGraphics.c` | 462 | `DrawLocale` (the caller), `DrawRoomBackground`, `DrawFloorSupport`, `RestoreWorkMap`, `ReadyLevel`, `RedrawRoomLighting` |
| `GliderPRO/Sources/Dynamics3.c` | 555 | `AddDynamicObject`, `ZeroDinahs`, `HandleDynamics`, `RenderDynamics` |
| `GliderPRO/Sources/Trip.c` | 246 | every `Toggle*`/`Trigger*` consumer of `dinahs[]` indices |
| `GliderPRO/Sources/Triggers.c` | 206 | `FireTrigger` — the other `dynaNum` consumer |
| `GliderPRO/Sources/Render.c` | 773 | `RenderFrame` order, `AddToMirrorRegion`, the dirty-rect queues |
| `GliderPRO/Sources/RectUtils.c` | 319 | `QSetRect`, `QOffsetRect`, `ZeroRectCorner`, `CenterRectInRect` |
| `GliderPRO/Sources/Play.c` | 822 | `PlayGame` frame order, QuickTime lifecycle, `SetObjectsToDefaults` |

Read in part:

| File | Lines read | Why |
|---|---|---|
| `GliderPRO/Sources/ObjectRects.c` | 1-1135 | `GetObjectRect`, `AddActiveRect`, `CreateActiveRects`, `VerticalRoomOffset`, `OffsetRectRoomRelative` |
| `GliderPRO/Sources/DynamicMaps.c` | 46-800 | `BackUpToSavedMap`, `ReBackUpSavedMap`, `AddCandleFlame`, `AddTikiFlame`, `AddBBQCoals`, `AddPendulum`, `AddStar`, `ZeroFlamesAndTheLike` |
| `GliderPRO/Sources/Objects.c` | 83-350 | `IsThisValid`, `ListOneRoomsObjects`, `ListAllLocalObjects`, `AddTempManholeRect` |
| `GliderPRO/Sources/ObjectAdd.c` | 1003-1090 | `HowManyDynamicObjects` and the sibling editor budget checks |
| `GliderPRO/Sources/Grease.c` | 1-300 | `AddGrease`, `ReBackUpGrease`, `SpillGrease` |
| `GliderPRO/Sources/Room.c` | 555-1140 | `GetNeighborRoomNumber`, `GetRoomNumber`, `GetNumberOfLights`, `IsShadowVisible` |
| `GliderPRO/Sources/Interactions.c` | 1030-1155 | `HandleRewards` link dispatch — every `masterObjects[].dynaNum` consumer |
| `GliderPRO/Sources/Dynamics.c` | 428-478 | `HandleTV` (the `tvWithMovieNumber` consumer) |
| `GliderPRO/Sources/StructuresInit.c` | 300-400, 565-605 | `tableSrc`, `deckSrc`, `stoolSrc`; appliance indicator source rects |
| `GliderPRO/Sources/StructuresInit2.c` | 260-520 | the complete `srcRects[144]` table |
| `GliderPRO/Sources/InterfaceInit.c` | 180-220 | `houseRect`, `playOriginH/V`, `localRoomsDest[9]` |
| `GliderPRO/Sources/Settings.c` | 1120-1185 | `numNeighbors` ∈ {1,3,9} |
| `GliderPRO/Sources/Main.c` | 100-200 | `numNeighbors` forced to 1 on ≤512-wide screens |
| `GliderPRO/Sources/ObjectDraw.c` | 60-1000 (`DrawTiki`, `DrawTable`, `DrawShelf`, `DrawCabinet`, `DrawCounter`, `DrawDresser`, `DrawDeckTable`, `DrawStool`) + scanned | helper signatures, which of them take a vertical origin, every `Draw*` helper's target GWorld, the `thisMac.isDepth == 4` palette branches |
| `GliderPRO/Sources/ObjectDraw2.c` | 900-1000, 1190-1440 + scanned | `DrawBalloon`/`DrawBall` (editor-only), `DrawPictObject`, `DrawPictSansWhiteObject`, `DrawCustPictSansWhite` |
| `GliderPRO/Sources/ObjectEdit.c` | 2620-2670 | proves `DrawBalloon`/`DrawBall` have editor-only callers |
| `GliderPRO/Sources/HouseIO.c` | 320-510 | `ReadHouse`/`WriteHouse`, the `COMPILEDEMO` 16526-byte / 45-room assertions |
| `GliderPRO/Sources/Environ.c` | 220-260, 380-400, 530-550 | `WhatsOurDepth`, `thisMac.isDepth` |
| `GliderPRO/Headers/GliderDefines.h` | all | every constant quoted below |
| `GliderPRO/Headers/GliderStructs.h` | all | `objectType`, `dynaType`, `objDataType`, `hotType` |

Byte-level verification scripts written for this document (throwaway, under `/tmp/wf-objdraw/`,
built on the repo's existing `tools/probe_house.py` BinHex+house parser):

* `verify_draw.py` — replays the budget consumption of `DrawARoomsObjects` +
  `ListAllLocalObjects` + `CreateActiveRects` over every 3×3 neighbourhood of every shipped house.
* `dump_room.py` — replays the `dinahs[]` registration order for one neighbourhood.
* `res_sweep.py` — the same, swept across screen sizes and `numNeighbors` values.
* `budget_sweep.py` (165 lines) — models `DrawLocale` → `DrawARoomsObjects` budget consumption
  *including* the `SectRect(itsRect, testRect)` gate, for a given screen size and `numNeighbors`;
  produced the eight-row resolution sweep in §9.9.5.
* `over.py` — lists every individual room that reaches or exceeds a cap, per house.
* `tabcount_final.py` — counts `kTable`/`kDeckTable` objects in rooms that are ever drawn in a
  vertically-offset slot; produced the §5.2 table.

Their observed output is quoted inline (§3.7, §5.2, §7.8, §7.9, §9.9). Everything in this document
that is labelled "measured" came from one of these scripts run against
`GliderPRO/Houses/*.binhex` (22 houses).

---

## 0. Executive summary

`DrawARoomsObjects(neighbor, redraw)` is called nine times per room change, once per slot of the
3×3 local neighbourhood, in the order **NW, NE, N, SW, SE, S, W, E, Central**
(`GliderPRO/Sources/RoomGraphics.c:80-120`). For each of those rooms it walks the room's
`objects[24]` array in **ascending slot index 0..23** (`GliderPRO/Sources/ObjectDrawAll.c:43`) —
i.e. exactly the order the objects are stored in the house file — and `switch`es on
`thisObject.what`.

Each case does up to three things, in this fixed order:

1. compute the object's rect (`GetObjectRect` → `OffsetRectRoomRelative`),
2. paint the object's static art into `backSrcMap` (most cases gated on
   `SectRect(&itsRect, &testRect, &whoCares)` and many additionally on `isLit`),
3. register an animation record — a `dinahs[]` entry via `AddDynamicObject`, or a `savedMaps[]`
   swatch via `BackUpToSavedMap`, or a `flames[]`/`tikiFlames[]`/`bbqCoals[]`/`pendulums[]`/
   `theStars[]`/`grease[]` entry, or a mirror-region union, or a `tempManholes[]` rect — gated on
   `!redraw` and, for most types, on `neighbor == kCentralRoom`.

Every registration function is an **append-to-array-with-hard-cap** that returns the new index or
`-1`. Because the arrays are appended in draw order, and because `HandleDynamics` and
`RenderDynamics` both iterate `for (i = 0; i < numDynamics; i++)`
(`GliderPRO/Sources/Dynamics3.c:35`, `:116`), **the draw order fixes the simulation order, the
dynamic-object z-order, and the numeric identity of every dynamic object** — which is in turn what
`masterObjects[].dynaNum` records at `GliderPRO/Sources/ObjectDrawAll.c:953-961` so that switches
and triggers can address them.

There is no depth sort anywhere. Painter's order *is* slot order within a room, and room order
across rooms, with the central room painted last so it overpaints its neighbours.

Budget exhaustion is silent everywhere. `AddDynamicObject` returns `-1` past 18 entries but the
object's static art is still drawn, so it becomes a permanently frozen prop; `AddActiveRect` returns
`-1` past 56 entries so the object becomes non-interactive; only `BackUpToSavedMap` (cap 24)
propagates its `-1` back into a guard that *skips the draw*, making over-budget prizes invisible but
still collectable.

---

## 1. Where `DrawARoomsObjects` sits

### 1.1 Call graph

```
NewGame                       Play.c:74
  └─ DrawLocale               RoomGraphics.c:44        ← the whole-neighbourhood compositor
       ├─ ZeroFlamesAndTheLike, ZeroDinahs, KillAllBands, ZeroMirrorRegion,
       │  ZeroTriggers, numTempManholes = 0, tvInRoom = false, tvWithMovieNumber = -1
       ├─ GetNeighborRoomNumber × 9  → localNumbers[9]
       ├─ IsRoomAStructure    × 9  → isStructure[9]
       ├─ ListAllLocalObjects  Objects.c:300           ← builds masterObjects[] + hotSpots[]
       ├─ SetGWorld(backSrcMap); PaintRect(&backSrcRect)
       ├─ 9 × { GetNumberOfLights ; DrawRoomBackground ; DrawARoomsObjects(slot, false) }
       ├─ DrawFloorSupport    RoomGraphics.c:257       (only if numNeighbors > 3)
       ├─ RestoreWorkMap      RoomGraphics.c:389       ← backSrcMap → workSrcMap, whole rect
       └─ shadowVisible = IsShadowVisible(); takingTheStairs = false

ReadyLevel                    RoomGraphics.c:402
  ├─ NilSavedMaps
  ├─ (QuickTime teardown)
  ├─ DetermineRoomOpenings
  ├─ DrawLocale
  └─ InitGarbageRects

RedrawRoomLighting            RoomGraphics.c:434       ← the ONLY redraw == true caller
  ├─ numLights = GetNumberOfLights(thisRoomNumber)
  ├─ DrawRoomBackground(thisRoomNumber, kCentralRoom, roomV)
  ├─ DrawARoomsObjects(kCentralRoom, true)             RoomGraphics.c:451
  ├─ UpdateOutletsLighting(thisRoomNumber, numLights)
  ├─ (DrawFloorSupport if numNeighbors > 3)
  ├─ CopyRectBackToWork / RestoreWorkMap
  └─ AddRectToWorkRects(&localRoomsDest[kCentralRoom])
```

`RestoreEntireGameScreen` (`GliderPRO/Sources/Play.c:800`) also calls `DrawLocale` directly, after
repainting the screen black — same path.

### 1.2 The two calling modes

`redraw` is `false` for all nine calls from `DrawLocale`, and `true` for the single call from
`RedrawRoomLighting`. Everything that differs between them:

| Behaviour | `redraw == false` (room load) | `redraw == true` (lighting change) |
|---|---|---|
| animation registration | `Add*` — **create** a new record | `ReBackUp*` — **re-capture** the background swatch of an existing record |
| `savedMaps[]` | `BackUpToSavedMap` (allocates a GWorld, appends) | `ReBackUpSavedMap` (linear search, re-copies, no allocation) |
| `dinahs[]` | `AddDynamicObject` called | **never called** — all `AddDynamicObject` sites are inside `if (!redraw)` |
| `masterObjects[].dynaNum` write-back | yes, `:953-961` | **no** — the whole write-back block is inside `if (!redraw)` |
| mirror region | `AddToMirrorRegion` (`:910`) | not called (`if ((neighbor == kCentralRoom) && (!redraw))`, `:907`) |
| `tempManholes[]` | `AddTempManholeRect` (`:277`) | **also called** — `:275-280` has no `redraw` guard, so a lighting change appends duplicate manhole rects until the 8-slot cap |
| `tvWithMovieNumber` / `tvInRoom` | may be set (`:707-708`) | not set (inside `if (!redraw)`, `:697`) |
| the QuickTime movie-box wiring at `:679-695` | runs | **also runs** — it is *outside* the `if (!redraw)` block, so a lighting change re-centres and re-clips the movie |
| which rooms | all 9 (subject to `numNeighbors`) | central only |

The `AddTempManholeRect` asymmetry is a real (if benign) defect: `numTempManholes` is only reset in
`DrawLocale` (`GliderPRO/Sources/RoomGraphics.c:56`), so every light switch flip in a room with a
manhole appends another copy of the same rect. `kMaxTempManholes` is 8
(`GliderPRO/Sources/Objects.c:12`), and `AddTempManholeRect` silently drops past that, so the
observable effect is bounded: after four flips in a two-manhole room the array is full and
`DrawFloorSupport` draws the same manhole cut-outs repeatedly. A Go port should reset the slice at
the top of the redraw path too, or guard the append on `!redraw`.

### 1.3 Preconditions on entry

`DrawARoomsObjects` reads a lot of global state and establishes none of it. Everything in this table
must already be correct or the function silently misbehaves:

| Global | Type | Set by | Meaning |
|---|---|---|---|
| `localNumbers[9]` | `short[9]` | `DrawLocale` `RoomGraphics.c:69` via `GetNeighborRoomNumber` | house-file room index for each of the 9 slots, or `kRoomIsEmpty` (−1) |
| `numLights` | `short` | `DrawLocale` immediately before each call, `RoomGraphics.c:80,84,88,92,96,100,107,112,117` | light count **for the room about to be drawn** |
| `masterObjects` | `objDataPtr` | `ListAllLocalObjects` `Objects.c:300` | 24 entries per non-empty local room |
| `numMasterObjects` | `short` | same | 24 × (number of non-empty local rooms) |
| `hotSpots` / `nHotSpots` | `hotPtr` / `short` | `CreateActiveRects` from `ListOneRoomsObjects` `Objects.c:284` | central-room interaction rects |
| `thisHouse` | `houseHand` | `ReadHouse` | the house handle; locked for the duration (`:40-41`, `:964`) |
| `houseRect` | `Rect` | `InterfaceInit.c:196-201` | screen rect minus the scoreboard, clamped to 1536×1026 |
| `playOriginH/V` | `short` | `InterfaceInit.c:203-204` | centring offsets |
| `localRoomsDest[9]` | `Rect[9]` | `InterfaceInit.c:206-218` | the nine 512×322 viewports |
| `srcRects[144]` | `Rect*` | `StructuresInit2.c:271` (alloc), `:306-475` (`InitSrcRects`) | per-type sprite-sheet source rects, consulted by `GetObjectRect` |
| `tvInRoom` | `Boolean` | cleared by `DrawLocale` `RoomGraphics.c:59` | "the movie has already been bound to some TV" |
| `tvWithMovieNumber` | `short` | reset to −1 by `DrawLocale` `RoomGraphics.c:60` | `dinahs[]` index of the movie TV |
| `hasMovie`, `theMovie`, `thisMac.hasQT` | `Boolean`, `Movie`, `Boolean` | `OpenHouseMovie` / `CheckOurEnvirons` | QuickTime availability |
| `isPlayMusicGame` | `Boolean` | prefs | passed to `DrawStereo` as the stereo's lit state (`:769`) |
| `thisMac.isDepth` | `short` | `CheckOurEnvirons` | 4 ⇒ the `Add*` helpers force even x-coordinates |
| the ambient `GrafPort` | — | `DrawRoomBackground` leaves it as `workSrcMap` (`RoomGraphics.c:238`) | see §5.6 |
| the ambient `GWorld` | — | `DrawLocale` set it to `backSrcMap` (`RoomGraphics.c:75`) | see §5.6 |

### 1.4 Postconditions

After the ninth call returns, the following are true and the rest of the engine depends on them:

1. `backSrcMap` holds the fully composited static picture of all nine rooms (backgrounds from
   `DrawRoomBackground`, objects from `DrawARoomsObjects`), *except* the between-floor support bands
   which `DrawFloorSupport` adds afterwards.
2. `numDynamics` ∈ [0, 18] and `dinahs[0..numDynamics-1]` are populated in draw order.
3. `numSavedMaps` ∈ [0, 24]; each `savedMaps[i]` is a GWorld holding either a background swatch or a
   pre-rendered animation strip, tagged with `(where, who)` = (room index, object slot).
4. `numFlames`, `numTikiFlames`, `numCoals`, `numPendulums`, `numStars`, `numGrease` are set, each
   entry pointing at a `savedMaps[]` index via its `.who` field.
5. `numTempManholes` ∈ [0, 8], consumed by `DrawFloorSupport`.
6. `mirrorRgn` / `hasMirror` describe the union of all central-room mirror interiors.
7. `masterObjects[n].dynaNum` is `-1` or a valid index into `dinahs[]` **or** `hotSpots[]` **or**
   `grease[]` depending on `masterObjects[n].theObject.what` (see §7.5).
8. `tvInRoom` and `tvWithMovieNumber` describe the movie binding; `tvOn` mirrors that TV's state.
9. `numLights` holds the light count of the **central** room (the last one assigned), which is what
   `RedrawRoomLighting` and `UpdateOutletsLighting` later assume.

Note (9): `numLights` is a single global reused as a per-room parameter. It is left holding the
central room's value because the central room is drawn last. Anything that reads `numLights` after
`DrawLocale` is reading the central room's count. A Go port that parallelises the nine rooms, or
that reorders them, breaks this.

---

## 2. Constants, types and globals

### 2.1 Every constant `DrawARoomsObjects` depends on

From `GliderPRO/Headers/GliderDefines.h` unless noted. Hex given where the source writes hex.

| Constant | Value | Hex | Defined at | Role in this function |
|---|---|---|---|---|
| `kCentralRoom` | 0 | 0x00 | `:217` | the `neighbor` value that unlocks central-only behaviour |
| `kNorthRoom` | 1 | 0x01 | `:218` | slot index |
| `kNorthEastRoom` | 2 | 0x02 | `:219` | slot index |
| `kEastRoom` | 3 | 0x03 | `:220` | slot index |
| `kSouthEastRoom` | 4 | 0x04 | `:221` | slot index |
| `kSouthRoom` | 5 | 0x05 | `:222` | slot index |
| `kSouthWestRoom` | 6 | 0x06 | `:223` | slot index |
| `kWestRoom` | 7 | 0x07 | `:224` | slot index |
| `kNorthWestRoom` | 8 | 0x08 | `:225` | slot index |
| `kMaxRoomObs` | 24 | 0x18 | `:250` | the loop bound at `:43` |
| `kMaxSparkles` | 3 | | `:251` | *not* used here (that is `sparkles[]`, a different array from `kSparkle` objects) |
| `kMaxFlyingPts` | 3 | | `:253` | not used here |
| `kMaxCandles` | 20 | 0x14 | `:255` | cap on `flames[]`, enforced inside `AddCandleFlame` |
| `kMaxTikis` | 8 | | `:256` | cap on `tikiFlames[]` |
| `kMaxCoals` | 8 | | `:257` | cap on `bbqCoals[]` |
| `kMaxPendulums` | 8 | | `:258` | cap on `pendulums[]` |
| `kMaxHotSpots` | **56** | 0x38 | `:259` | cap on `hotSpots[]`, enforced in `AddActiveRect` |
| `kMaxSavedMaps` | **24** | 0x18 | `:260` | cap on `savedMaps[]`, enforced in `BackUpToSavedMap` |
| `kMaxRubberBands` | 2 | | `:261` | not used here |
| `kMaxGrease` | 16 | 0x10 | `:262` | cap on `grease[]`, enforced in `AddGrease` |
| `kMaxStars` | 4 | | `:263` | cap on `theStars[]` |
| `kMaxShredded` | 4 | | `:264` | not used here |
| `kMaxDynamicObs` | **18** | 0x12 | `:265` | cap on `dinahs[]`, enforced in `AddDynamicObject` |
| `kMaxMasterObjects` | **216** | 0xD8 | `:266` | cap on `masterObjects[]` = exactly 9 × 24 |
| `kMaxViewWidth` | 1536 | 0x600 | `:267` | `houseRect` horizontal clamp |
| `kMaxViewHeight` | 1026 | 0x402 | `:268` | `houseRect` vertical clamp |
| `kMaxTempManholes` | 8 | | `Objects.c:12` | cap on `tempManholes[]` |
| `kMaxTriggers` | 16 | 0x10 | `Triggers.c:12` | cap on armed triggers (downstream) |
| `kMaxGarbageRects` | 48 | 0x30 | `Render.c:20` | cap on each dirty-rect queue (downstream) |
| `kNumTiles` | 8 | | `:496` | background tile count |
| `kTileWide` | 64 | 0x40 | `:497` | tile width |
| `kTileHigh` | 322 | 0x142 | `:498` | room height in pixels |
| `kRoomWide` | 512 | 0x200 | `:499` | room width = `kNumTiles * kTileWide` |
| `kFloorSupportTall` | **44** | 0x2C | `:500` | used at `:91-92` to fatten the central-room band for the candle-proximity test, and by `DrawFloorSupport` |
| `kVertLocalOffset` | **322** | 0x142 | `:501` | vertical stride between stacked rooms; source comment reads `// kTileHigh - 39 (was 283, then 295)` but the *value is 322*, equal to `kTileHigh` |
| `kRoomIsEmpty` | −1 | 0xFFFF | `:525` | `localNumbers[]` sentinel; the early return at `:33-34` |
| `kObjectIsEmpty` | −1 | 0xFFFF | `:526` | `objects[i].what` sentinel; case at `:53-54` |
| `kTicksPerFrame` | 2 | | `:533` | frame pacing (downstream, `Render.c:665`) |
| `kNumCandleFlames` | 5 | | `:447` | frames in a candle-flame strip |
| `kNumTikiFlames` | 5 | | `:448` | frames in a tiki strip |
| `kNumBBQCoals` | 4 | | `:449` | frames in a coal strip |
| `kNumPendulums` | 3 | | `:450` | frames in a pendulum strip |
| `kScoreboardTall` | 20 | 0x14 | `:515` | subtracted from the screen to make `houseRect` |
| `kManholeThruFloor` | 3957 | | `RoomGraphics.c:16` | PICT used by `DrawFloorSupport` for each `tempManholes[]` rect |
| `kNumSrcRects` | 144 | **0x90** | `:437` | size of `srcRects[]`; also 1 + the largest `what` code |
| `kFloorLimit` | 312 | | `:504` | used by `CreateActiveRects` for manholes |
| `kGliderWide` | 48 | | `:548` | used by `CreateActiveRects` for manholes |

The nine room slots are numbered so that the compass order is **not** the drawing order and **not**
the listing order; see §4.

### 2.2 The 118 `what` codes

`what` is a `short` and the codes are sparse: nine families, each starting on a 16-boundary
(`0x01`, `0x11`, `0x21`, `0x31`, `0x41`, `0x51`, `0x61`, `0x71`, `0x81`), with gaps at `0x20`,
`0x30`, `0x4A`-`0x50`, `0x59`-`0x60`, `0x6F`-`0x70`, `0x7A`-`0x80`. `GliderDefines.h:311-435`.

| Family | Range | Union arm | Codes |
|---|---|---|---|
| Blowers | `0x01`-`0x10` | `data.a` `blowerType` | `kFloorVent` 0x01, `kCeilingVent` 0x02, `kFloorBlower` 0x03, `kCeilingBlower` 0x04, `kSewerGrate` 0x05, `kLeftFan` 0x06, `kRightFan` 0x07, `kTaper` 0x08, `kCandle` 0x09, `kStubby` 0x0A, `kTiki` 0x0B, `kBBQ` 0x0C, `kInvisBlower` 0x0D, `kGrecoVent` 0x0E, `kSewerBlower` 0x0F, `kLiftArea` 0x10 |
| Furniture | `0x11`-`0x1F` | `data.b` `furnitureType` | `kTable` 0x11, `kShelf` 0x12, `kCabinet` 0x13, `kFilingCabinet` 0x14, `kWasteBasket` 0x15, `kMilkCrate` 0x16, `kCounter` 0x17, `kDresser` 0x18, `kDeckTable` 0x19, `kStool` 0x1A, `kTrunk` 0x1B, `kInvisObstacle` 0x1C, `kManhole` 0x1D, `kBooks` 0x1E, `kInvisBounce` 0x1F |
| Prizes | `0x21`-`0x2F` | `data.c` `bonusType` | `kRedClock` 0x21, `kBlueClock` 0x22, `kYellowClock` 0x23, `kCuckoo` 0x24, `kPaper` 0x25, `kBattery` 0x26, `kBands` 0x27, `kGreaseRt` 0x28, `kGreaseLf` 0x29, `kFoil` 0x2A, `kInvisBonus` 0x2B, `kStar` 0x2C, `kSparkle` 0x2D, `kHelium` 0x2E, `kSlider` 0x2F |
| Transport | `0x31`-`0x40` | `data.d` `transportType` | `kUpStairs` 0x31, `kDownStairs` 0x32, `kMailboxLf` 0x33, `kMailboxRt` 0x34, `kFloorTrans` 0x35, `kCeilingTrans` 0x36, `kDoorInLf` 0x37, `kDoorInRt` 0x38, `kDoorExRt` 0x39, `kDoorExLf` 0x3A, `kWindowInLf` 0x3B, `kWindowInRt` 0x3C, `kWindowExRt` 0x3D, `kWindowExLf` 0x3E, `kInvisTrans` 0x3F, `kDeluxeTrans` 0x40 |
| Switches | `0x41`-`0x49` | `data.e` `switchType` | `kLightSwitch` 0x41, `kMachineSwitch` 0x42, `kThermostat` 0x43, `kPowerSwitch` 0x44, `kKnifeSwitch` 0x45, `kInvisSwitch` 0x46, `kTrigger` 0x47, `kLgTrigger` 0x48, `kSoundTrigger` 0x49 |
| Lights | `0x51`-`0x58` | `data.f` `lightType` | `kCeilingLight` 0x51, `kLightBulb` 0x52, `kTableLamp` 0x53, `kHipLamp` 0x54, `kDecoLamp` 0x55, `kFlourescent` 0x56, `kTrackLight` 0x57, `kInvisLight` 0x58 |
| Appliances | `0x61`-`0x6E` | `data.g` `applianceType` | `kShredder` 0x61, `kToaster` 0x62, `kMacPlus` 0x63, `kGuitar` 0x64, `kTV` 0x65, `kCoffee` 0x66, `kOutlet` 0x67, `kVCR` 0x68, `kStereo` 0x69, `kMicrowave` 0x6A, `kCinderBlock` 0x6B, `kFlowerBox` 0x6C, `kCDs` 0x6D, `kCustomPict` 0x6E |
| Enemies | `0x71`-`0x79` | `data.h` `enemyType` | `kBalloon` 0x71, `kCopterLf` 0x72, `kCopterRt` 0x73, `kDartLf` 0x74, `kDartRt` 0x75, `kBall` 0x76, `kDrip` 0x77, `kFish` 0x78, `kCobweb` 0x79 |
| Clutter | `0x81`-`0x8F` | `data.i` `clutterType` | `kOzma` 0x81, `kMirror` 0x82, `kMousehole` 0x83, `kFireplace` 0x84, `kFlower` 0x85, `kWallWindow` 0x86, `kBear` 0x87, `kCalendar` 0x88, `kVase1` 0x89, `kVase2` 0x8A, `kBulletin` 0x8B, `kCloud` 0x8C, `kFaucet` 0x8D, `kRug` 0x8E, `kChimes` 0x8F |

The `switch` in `DrawARoomsObjects` has **no `default:` arm**. Any `what` value outside the table —
including the family gaps, and including anything a corrupt or newer house file might contain —
falls through the `switch` with no draw and no registration, but the `masterObjects` write-back at
`:953-961` still runs and clears `dynaNum` to −1. A Go port must reproduce the "unknown type is a
silent no-op" behaviour rather than erroring.

### 2.3 `objectType` on disk — verified byte-for-byte

`GliderPRO/Headers/GliderStructs.h:90-105`:

```c
typedef struct
{
    short   what;                 /* offset 0, 2 bytes, big-endian */
    union
    {
        blowerType      a;
        furnitureType   b;
        bonusType       c;
        transportType   d;
        switchType      e;
        lightType       f;
        applianceType   g;
        enemyType       h;
        clutterType     i;
    } data;                       /* offset 2, 10 bytes */
} objectType, *objectPtr;         /* total 12 bytes, no padding */
```

Each union arm is exactly 10 bytes (`GliderStructs.h:11-88`). The arms `DrawARoomsObjects` actually
reads:

| Arm | Fields (offset within the 12-byte object, size) |
|---|---|
| `blowerType a` | `Point topLeft` (2: `v` s16, 4: `h` s16), `short distance` (6), `Boolean initial` (8, u8), `Boolean state` (9, u8), `Byte vector` (10), `Byte tall` (11) |
| `furnitureType b` | `Rect bounds` (2: `top`, 4: `left`, 6: `bottom`, 8: `right`, all s16), `short pict` (10) |
| `bonusType c` | `Point topLeft` (2,4), `short length` (6), `short points` (8), `Boolean state` (10), `Boolean initial` (11) |
| `transportType d` | `Point topLeft` (2,4), `short tall` (6), `short where` (8), `Byte who` (10), `Byte wide` (11) |
| `switchType e` | `Point topLeft` (2,4), `short delay` (6), `short where` (8), `Byte who` (10), `Byte type` (11) |
| `lightType f` | `Point topLeft` (2,4), `short length` (6), `Byte byte0` (8), `Byte byte1` (9), `Boolean initial` (10), `Boolean state` (11) |
| `applianceType g` | `Point topLeft` (2,4), `short height` (6), `Byte byte0` (8), `Byte delay` (9), `Boolean initial` (10), `Boolean state` (11) |
| `enemyType h` | `Point topLeft` (2,4), `short length` (6), `Byte delay` (8), `Byte byte0` (9), `Boolean initial` (10), `Boolean state` (11) |
| `clutterType i` | `Rect bounds` (2,4,6,8), `short pict` (10) |

`Point` is Mac-order `{short v; short h;}` — **v first**. Getting that backwards silently mirrors
every object's position across the diagonal.

Verified against real bytes. `tools/probe_house.py` reports
`SIZEOF_OBJECT=12`, `SIZEOF_ROOM=348`, `SIZEOF_HOUSE_HEADER=866`, and
`room[N].objects[]` begins at room offset 60, so `room[0].objects[i]` lives at file offset
`866 + 60 + 12*i`. Observed, "Demo House":

```
  version 0x0200  nRooms=51  fileSize=18614 expected=18614
  room[0] objects[] base = houseHdr(866) + 60 = 926
    room[0].objects[ 0] @0x00039E = 00 01 01 31 00 ab 01 0d 01 01 01 01  what=0x01 (kFloorVent)
    room[0].objects[ 1] @0x0003AA = 00 18 00 9a 00 d9 01 25 01 55 00 00  what=0x18 (kDresser)
    room[0].objects[ 2] @0x0003B6 = 00 21 00 89 01 36 00 00 00 00 00 01  what=0x21 (kRedClock)
```

Decoding slot 0 by hand: `what = 0x0001 = kFloorVent`; `data.a.topLeft.v = 0x0131 = 305`
(= `kFloorVentTop`, `GliderDefines.h:467`), `data.a.topLeft.h = 0x00AB = 171`,
`data.a.distance = 0x010D = 269`, `initial = 0x01`, `state = 0x01`, `vector = 0x01`, `tall = 0x01`.
Slot 2: `what = 0x0021 = kRedClock`; `topLeft.v = 0x0089 = 137`, `topLeft.h = 0x0136 = 310`,
`length = 0`, `points = 0`, `state = 0x00`, `initial = 0x01` — a red clock that has already been
collected in this saved state, which `IsThisValid` will therefore skip (§3.4).

Slot 1: `what = 0x0018 = kDresser`; `data.b.bounds = {top=0x009A=154, left=0x00D9=217,
bottom=0x0125=293, right=0x0155=341}`, `pict = 0`. Note `bottom == 293 == kDresserBottom`
(`GliderDefines.h:476`).

`kMaxRoomObs` slots are always physically present in the file whether used or not — the array is
fixed-size, 24 × 12 = 288 bytes, and unused slots hold `what = -1` (`0xFFFF`). `numObjects` at room
offset 58 is a *hint* the editor maintains, not a bound: `DrawARoomsObjects` ignores it entirely and
always scans all 24 slots.

### 2.4 `dynaType` — the thing `AddDynamicObject` appends

`GliderPRO/Headers/GliderStructs.h:310-320`, with byte offsets computed for 68k/PPC alignment (all
members are 2-byte or byte-pair, so there is no padding):

```c
typedef struct
{
    Rect        dest;             /* offset  0, 8 bytes: top, left, bottom, right (s16 each) */
    Rect        whole;            /* offset  8, 8 bytes: the dirty-rect union                */
    short       hVel, vVel;       /* offset 16, 18                                           */
    short       type, count;      /* offset 20, 22   type == the object's `what` code         */
    short       frame, timer;     /* offset 24, 26                                           */
    short       position, room;   /* offset 28, 30   room == house-file room index            */
    Byte        byte0, byte1;     /* offset 32, 33   byte0 == the object's slot 0..23          */
    Boolean     moving, active;   /* offset 34, 35                                           */
} dynaType, *dynaPtr;             /* total 36 bytes */
```

`dinahs` is `dynaPtr dinahs;` (`Dynamics3.c:18`) — a pointer to a heap array allocated as
`NewPtr(sizeof(dynaType) * kMaxDynamicObs)` = 36 × 18 = 648 bytes
(`GliderPRO/Sources/StructuresInit2.c:261`). `numDynamics` is `short numDynamics;`
(`Dynamics3.c:19`).

Fields whose meaning is type-dependent and therefore easy to get wrong when porting:

| Field | Normal meaning | Overloaded meanings |
|---|---|---|
| `dest` | the rect actually blitted | for appliances it is the **indicator** rect (e.g. `tvScreen1` at `itsRect.left+17, itsRect.top+10`), not the whole object |
| `whole` | dirty-rect union for the blit | for `kFish` it is repurposed as the saved home rect (`Trip.c:200`) |
| `hVel` | horizontal velocity | toaster: the *clip top*, `where->top + 2` (`Dynamics3.c:223`); outlet: the **light count** `numLights` (`:314`), later refreshed by `UpdateOutletsLighting` (`Trip.c:242`) |
| `count` | frame count | toaster: the *initial launch velocity* (`Dynamics3.c:233`); outlet: the idle timer reload (`:316`) |
| `position` | a `short`, generally a pixel coordinate for movers | outlet/toaster: a 0/1 launch-vs-idle **state flag** (`Dynamics3.c:236`, `:319`; `Trip.c:173-177`) |
| `byte0` | the object's slot index 0..23, cast from `index` | — (this is how `Handle*` finds its way back to the house object) |
| `byte1`, `moving` | per-type animation state | **deliberately not reset** by `ZeroDinahs` — see §9.2 |

Note that `position` is a `short`, not a `long`. The four writes of `dinahs[].position` in
`AddDynamicObject` (`:209`, `:236`, `:256`, `:276`, …) all store small values, but the moving-enemy
handlers in `Dynamics2.c` treat it as a sub-pixel accumulator, so a Go port must use `int16` and
**must** let it wrap at ±32767 if it wants bit-identical behaviour in pathological houses.

### 2.5 `objDataType` — the local object list `DrawARoomsObjects` writes back into

`GliderPRO/Headers/GliderStructs.h:322-332`:

```c
typedef struct
{
    short       roomNum;          /* house-file room index                     */
    short       objectNum;        /* slot 0..23 within that room               */
    short       roomLink;         /* room this object links to, or -1          */
    short       objectLink;       /* object slot linked to, or -1              */
    short       localLink;        /* index into masterObjects[] of the linkee  */
    short       hotNum;           /* index into hotSpots[], or -1              */
    short       dynaNum;          /* index into dinahs[] / hotSpots[] / grease[], or -1 */
    objectType  theObject;        /* 12-byte copy of the house-file object     */
} objDataType, *objDataPtr;       /* 26 bytes */
```

`masterObjects` is `objDataPtr masterObjects;` with `short numMasterObjects, numLocalMasterObjects;`.
It is rebuilt from scratch by `ListAllLocalObjects` on every room change
(`Objects.c:305-307` zero `numMasterObjects`, `numLocalMasterObjects`, `nHotSpots`).

### 2.6 Locals of `DrawARoomsObjects`

`ObjectDrawAll.c:25-31`:

| Local | Type | Purpose |
|---|---|---|
| `thisObject` | `objectType` | a **copy** of the house-file object (`:50`), so mutations do not persist |
| `whoCares` | `Rect` | scratch output of `SectRect` — and then **reused as the TV screen rect** at `:683-690` |
| `itsRect` | `Rect` | the object's screen rect |
| `rectA` | `Rect` | room-relative copy of `itsRect` for `AddDynamicObject`; also the candle-proximity probe (`:88`) |
| `rectB` | `Rect` | the fattened central-room band for the candle-proximity test (`:90-92`) |
| `testRect` | `Rect` | `houseRect` zeroed to (0,0) — the clip rect |
| `theRgn` | `RgnHandle` | the QuickTime display clip region (`:689-692`) |
| `i` | `short` | the object slot, 0..23 |
| `legit` | `short` | `savedMaps[]` index, or −1 |
| `dynamicNum` | `short` | the value written into `masterObjects[].dynaNum` |
| `n` | `short` | write-back loop counter |
| `floor`, `suite`, `room`, `obj` | `short` | switch-target decoding (`:513-515` etc.) |
| `wasState` | `char` | saved handle state for `HGetState`/`HSetState` |
| `isLit` | `Boolean` | `numLights > 0` (`:38`), constant for the whole room |

`whoCares` being reused for the TV screen rect is not a bug (the `SectRect` result is dead by then)
but it is a readability trap: the same variable is an output parameter on `:677` and a geometry
value on `:683-690`.

---
## 3. Question (a) — the draw order **within** a room, object index 0..23

### 3.1 The loop

`GliderPRO/Sources/ObjectDrawAll.c:23-53` and `:950-965`, verbatim, with nothing elided from the
control flow (only the 900-line `switch` body replaced by a comment):

```c
void DrawARoomsObjects (short neighbor, Boolean redraw)          /* :23 */
{
	objectType	thisObject;                                       /* :25 */
	Rect		whoCares, itsRect, rectA, rectB, testRect;
	RgnHandle	theRgn;
	short		i, legit, dynamicNum, n;
	short		floor, suite, room, obj;
	char		wasState;
	Boolean		isLit;                                            /* :31 */

	if (localNumbers[neighbor] == kRoomIsEmpty)                   /* :33 */
		return;                                                    /* :34 */

	testRect = houseRect;                                         /* :36 */
	ZeroRectCorner(&testRect);                                    /* :37 */
	isLit = (numLights > 0);                                      /* :38 */

	wasState = HGetState((Handle)thisHouse);                      /* :40 */
	HLock((Handle)thisHouse);                                     /* :41 */

	for (i = 0; i < kMaxRoomObs; i++)                             /* :43 */
	{
		dynamicNum = -1;                                           /* :45 */
		legit = -1;                                                /* :46 */

		if (IsThisValid(localNumbers[neighbor], i))                /* :48 */
		{
			thisObject =                                           /* :50 */
				(*thisHouse)->rooms[localNumbers[neighbor]].objects[i];

			switch (thisObject.what)                               /* :51 */
			{
				/* ... 74 case groups, :53-949 ... */
			}
		}

		if (!redraw)                                               /* :953 */
		{
			for (n = 0; n < numMasterObjects; n++)                  /* :955 */
			{
				if ((masterObjects[n].objectNum == i) &&             /* :956-957 */
						(masterObjects[n].roomNum == localNumbers[neighbor]))
					masterObjects[n].dynaNum = dynamicNum;           /* :959 */
			}
		}
	}

	HSetState((Handle)thisHouse, wasState);                       /* :964 */
}
```

**The answer to (a):** the draw order within a room is `i = 0, 1, 2, ... 23` — a plain ascending
scan of `objects[kMaxRoomObs]` — with no sorting, no filtering by type, no depth key, and no second
pass. `kMaxRoomObs` is 24 (`GliderPRO/Headers/GliderDefines.h:250`).

### 3.2 Proof that slot order is house-file storage order

Three independent arguments:

1. **The array is read in place.** `:50` indexes `(*thisHouse)->rooms[R].objects[i]` directly with
   the loop variable. `roomType.objects` is `objectType objects[kMaxRoomObs];`
   (`GliderPRO/Headers/GliderStructs.h:126`) — a fixed-size inline array, not a pointer, not a list.
   There is no indirection layer between file order and draw order.
2. **The file layout is flat and fixed-stride.** Verified by parsing: `sizeof(objectType)` is 12 with
   no padding, `objects[]` sits at offset 60 within the 348-byte `roomType`, and the house header is
   866 bytes, so `room[r].objects[i]` is at file offset `866 + 348*r + 60 + 12*i`. For every shipped
   house, `866 + 348*nRooms == fileSize` exactly (see §3.7), which can only hold if the stride
   assumption is right.
3. **`numObjects` is not consulted.** The room header carries `short numObjects` at room offset 58,
   but `DrawARoomsObjects` never reads it — the loop bound is the compile-time constant
   `kMaxRoomObs`. So even a house whose `numObjects` is stale or wrong draws exactly the slots whose
   `what != -1`. A Go port must likewise ignore `numObjects` for drawing purposes.

### 3.3 What slot order *means* — z-order, and how the editor assigns it

Because the draw is a straight painter's blit into `backSrcMap` with no depth buffer, **slot index is
z-order**: object 23 paints over object 0. Nothing else influences it — not the object's `top`/`left`,
not its type, not its family.

The editor assigns slots with `FindEmptyObjectSlot` (`GliderPRO/Sources/ObjectAdd.c:806-819`):

```c
emptySlot = -1;
for (i = 0; i < kMaxRoomObs; i++)
	if (thisRoom->objects[i].what == kObjectIsEmpty)
	{ emptySlot = i; break; }
return (emptySlot);
```

— **the first hole**, not the end of the array. Consequences a porter must be aware of because they
are observable in shipped houses:

* Adding objects to a fresh room yields slots 0, 1, 2, … in creation order, so z-order matches
  creation order.
* Deleting object *k* and then adding a new one puts the new object in slot *k*, which places it
  *behind* everything created after the deleted object. This is why some rooms in the shipped houses
  have visually "wrong" overlaps.
* There is no "bring to front" operation in the editor. The only way to change z-order is delete and
  re-add.

An importer/exporter in Go must therefore preserve slot indices byte-for-byte on round-trip. Compacting
the array (removing holes) would silently re-order the artwork *and* re-number every `dinahs[]` index
(§7), which would change gameplay.

### 3.4 `IsThisValid` — the only per-slot filter

`GliderPRO/Sources/Objects.c:89-122`:

```c
Boolean IsThisValid (short where, short who)
{
	itsGood = true;
	wasState = HGetState((Handle)thisHouse);  HLock((Handle)thisHouse);
	switch ((*thisHouse)->rooms[where].objects[who].what)
	{
		case kObjectIsEmpty:                    /* -1 */
			itsGood = false;  break;
		case kRedClock: case kBlueClock: case kYellowClock: case kCuckoo:
		case kPaper: case kBattery: case kBands: case kFoil:
		case kInvisBonus: case kStar: case kSparkle: case kHelium:
			itsGood = (*thisHouse)->rooms[where].objects[who].data.c.state;
			break;
	}
	HSetState((Handle)thisHouse, wasState);
	return (itsGood);
}
```

Exactly two reasons a slot is skipped:

| Reason | Test | Which types |
|---|---|---|
| slot unused | `what == kObjectIsEmpty` (−1) | any |
| prize already collected | `data.c.state == false` | the 12 collectable prize types listed above |

Note the omissions from that prize list: **`kGreaseRt` (0x28) and `kGreaseLf` (0x29) are absent**,
and `kSlider` (0x2F) is absent. Grease is therefore always "valid" and always reaches the switch —
which is correct, because a spilt grease bucket is still a visible object (it draws in its
knocked-over pose, `:384`/`:405`), unlike a collected clock which vanishes. `kSlider` has no case in
the draw switch at all (`:422-424` groups it with `kInvisBonus` as an explicit no-op), so its
validity is irrelevant.

Also note `IsThisValid` locks and unlocks `thisHouse` on **every call** — 24 calls per room, 216 per
`DrawLocale` — even though the caller already holds the lock from `:41`. Nested `HLock`/`HSetState`
on a locked handle is a no-op-then-restore, so it is correct but wasteful. Irrelevant to a Go port
except as an explanation for why the code looks redundant.

`IsThisValid(where, who)` reads `rooms[where]` with **no bounds check**. It is called at `:48` with
`localNumbers[neighbor]`, which the `:33-34` early return has already proved is not −1, so this
particular call site is safe. Other call sites in the codebase are not (see §4.7).

### 3.5 The `masterObjects` write-back runs for skipped slots too

`dynamicNum` is reset to −1 at the **top** of every iteration (`:45`), before `IsThisValid`. The
write-back at `:953-961` is outside the `if (IsThisValid(...))` block. So:

* a slot that is empty, or a prize that has been collected, still gets
  `masterObjects[n].dynaNum = -1` written for its `(roomNum, objectNum)`;
* a slot whose type has no registration also gets −1;
* only a slot that actually registered something gets a non-negative value.

This is load-bearing: it clears stale `dynaNum` values from the *previous* room's `DrawLocale`. If a
Go port only writes back on success, a `dynaNum` left over from a previous neighbourhood would alias
a live `dinahs[]` entry and a switch would toggle the wrong appliance.

The write-back is an **O(24 × numMasterObjects)** linear scan — up to 24 × 216 = 5184 comparisons per
room, 46 656 per `DrawLocale`. It matches on the pair `(objectNum == i, roomNum == localNumbers[neighbor])`.
Because `ListOneRoomsObjects` writes exactly one entry per `(room, slot)` pair
(`GliderPRO/Sources/Objects.c:270-292`), the match is unique when the room appears once in the
neighbourhood. **It is not unique when the same house room appears in two slots of the 3×3
neighbourhood** — which happens in one-room-wide houses where, e.g., the east and west neighbours are
both the same room, or where a room is its own neighbour. In that case both master entries for that
room get the same `dynaNum`, and the later-drawn slot's value wins. A Go port can either replicate
the linear scan (safest) or build a `map[roomSlot]int` — but only if it replicates the
last-write-wins behaviour.

### 3.6 Consequences of painter's-order-is-slot-order

| Consequence | Why | Port implication |
|---|---|---|
| static art overlaps resolve by slot index | single-pass painter's blit into `backSrcMap` | do not sort by y, do not use a z-buffer |
| `savedMaps[]` background swatches capture whatever has been drawn *so far* | `BackUpToSavedMap` reads `backSrcMap` at the time of the call (`DynamicMaps.c:70-93`) | a flame in slot 3 captures a background that does **not** include the table in slot 7; move the flame to slot 9 and the swatch changes |
| animated elements always appear *above* all static art | they are re-blitted into `workSrcMap` each frame by `RenderFrame`, after `RestoreWorkMap` has copied `backSrcMap` in | correct z-order for animation is: all static, then the `RenderFrame` order in §6.5 |
| `dinahs[]` indices follow slot order | `AddDynamicObject` appends | §7 |

The middle row is the subtle one and it is a genuine authored-content dependency: the "background"
behind an animated element is frozen at the moment its slot is drawn, so a later-slot object that
overlaps an earlier-slot flame will be *erased* every time the flame animates. This is visible in
shipped houses as flames "punching holes" in furniture. A Go port that instead re-composites the
true background per frame will look *better* and be *wrong*.

### 3.7 Byte-verified: real slot orders from the shipped houses

Running the verification script over **all 22** houses shipped in `GliderPRO/Houses/`. `firstRoom` and
`nRooms` are the two `short`s at file offset **862** and **864** (the last four bytes of the 866-byte
header); `fileSize` is the length of the BinHex data fork:

```
house                   @862..865   firstRoom  nRooms  fileSize  866+348*n  delta  version
Art Museum              ...              --       109     38798      38798      0   0x0200
CD Demo House           00 46 00 CE      70       206     72554      72554      0   0x0200
California or Bust!     ...              --        16      6434       6434      0   0x0200
Castle o' the Air       ...              --        85     30446      30446      0   0x0200
Davis Station           ...              --        65     23486      23486      0   0x0200
Demo House              00 00 00 2D       0        45     16526      16526      0   0x0200
Empty House             ...              --        35     13046      13046      0   0x0200
Fun House               00 1D 00 2B      29        43     15830      15830      0   0x0200
Grand Prix              ...              --       175     61766      61766      0   0x0200
ImagineHouse PRO II     00 01 01 17       1       279     97958      97958      0   0x0200
In The Mirror           ...              --        97     34622      34622      0   0x0200
Land of Illusion        ...              --       303    106310     106310      0   0x0200
Leviathan               ...              --       472    165122     165122      0   0x0200
Metropolis              ...              --       127     45062      45062      0   0x0200
Nemo's Market           ...              --       124     44018      44018      0   0x0200
Rainbow's End           ...              --       223     78470      78470      0   0x0200
Sampler                 00 01 00 02       1         2      1564       1562     +2   0x0200
Slumberland             ...              --       383    134150     134150      0   0x0200
SpacePods               ...              --       402    140762     140762      0   0x0200
Teddy World             ...              --       531    185654     185654      0   0x0200
The Asylum Pro          ...              --       140     49586      49586      0   0x0200
Titanic                 ...              --       208     73250      73250      0   0x0200
```

All 22 report `version` = `0x0200` (`kNewHouseVersion` is checked at `HouseIO.c:395`; a house whose
version is `>= kNewHouseVersion` is refused with `kYellowNewerVersion`). The stride arithmetic
`866 + 348 × nRooms` is exact in **21 of 22** houses, confirming §3.2 argument 2.

**`Sampler` is 2 bytes longer than the formula predicts** (1564 vs 1562), and the trailing two bytes are
`0x01 0x01`. The engine does not care: `HouseIO.c:341-372` reads the file with
`GetEOF` → `NewHandle(byteCount)` → `FSRead`, then takes the room count from the header
(`numberRooms = (*thisHouse)->nRooms`, `:380`) and never validates the length against the formula.
A Go port must do the same — **derive room count from the header field at offset 862/864, never from
`(fileSize − 866) / 348`** — or it will read one extra garbage room from `Sampler` and mis-size any
third-party house with a trailer. Conversely `WriteHouse` writes `GetHandleSize((Handle)thisHouse)`
bytes and then `SetEOF(houseRefNum, byteCount)` (`:473, :491, :499`), so a trailer survives a
load/save round trip.

Independent confirmation of the `Demo House` figures, from the demo build's hard-coded sanity checks:

```c
#ifdef COMPILEDEMO
	if (byteCount != 16526L)                 /* HouseIO.c:349 */
		return (false);
#endif
	...
	numberRooms = (*thisHouse)->nRooms;      /* HouseIO.c:380 */
#ifdef COMPILEDEMO
	if (numberRooms != 45)                   /* HouseIO.c:382 */
		return (false);
#endif
```

`16526` and `45` are exactly the measured values, so the parse is right: `866 + 348 × 45 = 16526`.

"Demo House" room 0 — `floor` = 1, `suite` = 63, header `numObjects` = 10 — all 24 slots as stored
(`--` = `what == 0xFFFF` = `kObjectIsEmpty` = −1):

```
slot  what   name             state   note
   0  0x01   kFloorVent         1     drawn first, so everything else can overlap it
   1  0x18   kDresser           -     furnitureType: no state field (data.b)
   2  0x21   kRedClock          0     data.c.state == 0 -> IsThisValid false -> SKIPPED
   3  0x01   kFloorVent         1
   4  0x86   kWallWindow        -
   5  0x01   kFloorVent         1
   6  0x8A   kVase2             -
   7  0x85   kFlower            -
   8  0x51   kCeilingLight      1     contributes to numLights for this room
   9  0x6E   kCustomPict        1
  10.. --                             empty
```

Ten occupied slots, and the header's `numObjects` field is exactly 10 — the field is *accurate* here,
which is worth stating because §3.3 proves the draw loop never reads it (it always iterates
`i = 0 .. kMaxRoomObs-1` = 0..23). A port must not "optimise" the loop to `numObjects`, because
nothing in the file format or the editor guarantees the field stays correct after edits, and
`IsThisValid` is the only skip rule the engine honours.

Note slot 2: a `kRedClock` with `data.c.state == 0` (already collected in a saved game, or authored
off). It occupies slot 2 for ever, so it still *reserves* its position in the z-order even though
nothing is drawn — removing it in a port would not change the picture, but it *would* change the
`savedMaps[]`/`dinahs[]` index assignment of every later slot (§7.4).

"ImagineHouse PRO II" room 120 ("All TVs on = Safe Passage") — `floor` = 1, `suite` = 81, header
`numObjects` = 16 — the room used as the worked example in §7.8 and §8.8:

```
slot  what   name             state   note
   0  0x58   kInvisLight        1     draws nothing; numLights++ so isLit is true
   1  0x65   kTV                1     <-- lowest-index TV: gets the QuickTime movie (§8)
   2  0x65   kTV                1
   3  0x65   kTV                1
   4  0x65   kTV                1
   5  0x12   kShelf             -
   6  0x35   kFloorTrans        -
   7  0x35   kFloorTrans        -
   8  0x35   kFloorTrans        -
   9  0x35   kFloorTrans        -
  10  0x01   kFloorVent         1
  11  0x01   kFloorVent         1
  12  0x01   kFloorVent         1
  13  0x01   kFloorVent         1
  14  0x1E   kBooks             -
  15  0x6D   kCDs               1
  16.. --                             empty
```

The four TVs occupy consecutive slots 1..4, so they are drawn in slot order and registered into
`dinahs[]` in slot order — which is what makes the "first TV" rule deterministic (§8.3). With the room
as central, the `dinahs[]` assignment for this room alone is exactly four entries: slot 1 → index 0
(and `tvWithMovieNumber = 0`), slot 2 → 1, slot 3 → 2, slot 4 → 3. Slot 15's `kCDs` registers
**nothing** — it is not in `HowManyDynamicObjects`'s list (`ObjectAdd.c:1051-1067`) and its case
(`ObjectDrawAll.c:635-640`) only calls `DrawSimpleAppliance`, so a CD rack is static art despite living
in the appliance numeric range. Slot 0
(`kInvisLight`) draws nothing but sets `numLights`, so `isLit` is already true before the first TV is
reached — which matters because the `kTV` case is one of the ones that *does* gate its art on `isLit`
(§10.2 group 54).

---

## 4. Question (b) — the draw order **across** the nine rooms, proved

### 4.1 The code

`GliderPRO/Sources/RoomGraphics.c:74-124`, verbatim:

```c
	GetGWorld(&wasCPort, &wasWorld);                          /* :74 */
	SetGWorld(backSrcMap, nil);                               /* :75 */
	PaintRect(&backSrcRect);                                  /* :76 */

	if (numNeighbors > 3)                                     /* :78 */
	{
		numLights = GetNumberOfLights(localNumbers[kNorthWestRoom]);        /* :80 */
		DrawRoomBackground(localNumbers[kNorthWestRoom], kNorthWestRoom, roomV + 1);
		DrawARoomsObjects(kNorthWestRoom, false);                            /* :82 */

		numLights = GetNumberOfLights(localNumbers[kNorthEastRoom]);        /* :84 */
		DrawRoomBackground(localNumbers[kNorthEastRoom], kNorthEastRoom, roomV + 1);
		DrawARoomsObjects(kNorthEastRoom, false);                            /* :86 */

		numLights = GetNumberOfLights(localNumbers[kNorthRoom]);            /* :88 */
		DrawRoomBackground(localNumbers[kNorthRoom], kNorthRoom, roomV + 1);
		DrawARoomsObjects(kNorthRoom, false);                                /* :90 */

		numLights = GetNumberOfLights(localNumbers[kSouthWestRoom]);        /* :92 */
		DrawRoomBackground(localNumbers[kSouthWestRoom], kSouthWestRoom, roomV - 1);
		DrawARoomsObjects(kSouthWestRoom, false);                            /* :94 */

		numLights = GetNumberOfLights(localNumbers[kSouthEastRoom]);        /* :96 */
		DrawRoomBackground(localNumbers[kSouthEastRoom], kSouthEastRoom, roomV - 1);
		DrawARoomsObjects(kSouthEastRoom, false);                            /* :98 */

		numLights = GetNumberOfLights(localNumbers[kSouthRoom]);            /* :100 */
		DrawRoomBackground(localNumbers[kSouthRoom], kSouthRoom, roomV - 1);
		DrawARoomsObjects(kSouthRoom, false);                                /* :102 */
	}

	if (numNeighbors > 1)                                     /* :105 */
	{
		numLights = GetNumberOfLights(localNumbers[kWestRoom]);             /* :107 */
		DrawRoomBackground(localNumbers[kWestRoom], kWestRoom, roomV);
		DrawARoomsObjects(kWestRoom, false);                                 /* :109 */
		DrawLighting();                                                      /* :110 */

		numLights = GetNumberOfLights(localNumbers[kEastRoom]);             /* :112 */
		DrawRoomBackground(localNumbers[kEastRoom], kEastRoom, roomV);
		DrawARoomsObjects(kEastRoom, false);                                 /* :114 */
		DrawLighting();                                                      /* :115 */
	}

	numLights = GetNumberOfLights(localNumbers[kCentralRoom]);             /* :118 */
	DrawRoomBackground(localNumbers[kCentralRoom], kCentralRoom, roomV);
	DrawARoomsObjects(kCentralRoom, false);                                 /* :120 */
	DrawLighting();                                                         /* :121 */

	if (numNeighbors > 3)                                     /* :123 */
		DrawFloorSupport();                                    /* :124 */
```

### 4.2 The order, stated

| Draw # | Slot constant | Value | Compass | Gate | `roomV` passed to background |
|---|---|---|---|---|---|
| 1 | `kNorthWestRoom` | 8 | NW | `numNeighbors > 3` | `roomV + 1` |
| 2 | `kNorthEastRoom` | 2 | NE | `numNeighbors > 3` | `roomV + 1` |
| 3 | `kNorthRoom` | 1 | N | `numNeighbors > 3` | `roomV + 1` |
| 4 | `kSouthWestRoom` | 6 | SW | `numNeighbors > 3` | `roomV - 1` |
| 5 | `kSouthEastRoom` | 4 | SE | `numNeighbors > 3` | `roomV - 1` |
| 6 | `kSouthRoom` | 5 | S | `numNeighbors > 3` | `roomV - 1` |
| 7 | `kWestRoom` | 7 | W | `numNeighbors > 1` | `roomV` |
| 8 | `kEastRoom` | 3 | E | `numNeighbors > 1` | `roomV` |
| 9 | `kCentralRoom` | 0 | centre | none — always | `roomV` |

**NW, NE, N, SW, SE, S, W, E, Central.**

Note that this is neither numeric slot order (which would be 0,1,2,3,4,5,6,7,8) nor compass order
(which would be N,NE,E,SE,S,SW,W,NW) nor the listing order used by `ListAllLocalObjects` (§4.6). It
is: **corners-then-edge of the top row, corners-then-edge of the bottom row, then the two side
edges, then the centre.** Within each of the first two groups the *corners come before the edge*
(NW, NE, then N), which only matters for overlapping art at room seams.

### 4.3 The proof

Four independent lines of evidence, since the task requires this be proved and not merely asserted.

**Proof 1 — textual order of the nine call sites.** The nine `DrawARoomsObjects` calls in
`DrawLocale` appear at source lines 82, 86, 90, 94, 98, 102, 109, 114, 120 with the slot arguments
`kNorthWestRoom, kNorthEastRoom, kNorthRoom, kSouthWestRoom, kSouthEastRoom, kSouthRoom, kWestRoom,
kEastRoom, kCentralRoom` respectively. C evaluates statements in order; there are no loops, no
function-pointer tables, no `goto`s, and the only branches are the two `if (numNeighbors > N)` gates,
which are monotone (they include prefixes of the list, never reorder it). Therefore the dynamic call
order equals the textual order. Grep confirms these are the *only* nine call sites in `DrawLocale`
and that the whole program has exactly ten call sites total (the tenth is `RoomGraphics.c:451`, the
`redraw = true` one).

**Proof 2 — the `numLights` interleave forces sequential execution.** Each call is preceded by
`numLights = GetNumberOfLights(localNumbers[X])` and `DrawARoomsObjects` reads the global `numLights`
at its line 38. The nine calls therefore *cannot* be reordered or parallelised without changing
behaviour, and the compiler cannot reorder them because `GetNumberOfLights` and
`DrawARoomsObjects` both touch globals (no `const`, no `pure`). The observable order is pinned.

**Proof 3 — the overpaint evidence.** All nine rooms composite into the same `backSrcMap`, and their
destination rects overlap: `localRoomsDest[]` places them on a 512-wide × 322-tall grid
(`InterfaceInit.c:206-218`), and objects routinely stick out past their room's rect (a stool at the
bottom of the north room hangs down into the central room's band; `VerticalRoomOffset` is exactly
`kVertLocalOffset` = 322 = the room height, so there is no gutter). Central being drawn ninth is what
makes the *playable* room's art win every seam conflict. If the order were reversed you would see
neighbour objects drawn on top of central-room objects, which does not happen.

**Proof 4 — the `tvInRoom` mechanism only works in this order.** `DrawLocale` sets
`tvInRoom = false` at `:59` and `tvWithMovieNumber = -1` at `:60` before any room is drawn. The TV
case (`ObjectDrawAll.c:680`, `:705`) requires `(neighbor == kCentralRoom) && (!tvInRoom)`, and
`tvInRoom` is only ever set true inside that same central-only branch (`:708`). So the guard's
`!tvInRoom` term can only ever be false if the central room contains *two or more* TVs — it is a
within-room "first one wins" latch, not a cross-room one. This is consistent with, and only sensible
under, an order in which the central room is a single contiguous pass. It is also *independent* of
where in the nine the central room falls — which is the point of the next observation.

**Corollary the porter needs:** the fact that the central room is drawn **last** is what makes
central-room dynamic objects occupy the **highest** `dinahs[]` indices. Everything in §7.8 and §7.9
follows from that.

### 4.4 `numNeighbors` gating: 1, 3, or 9 rooms

`numNeighbors` is a user preference with exactly three legal values:

| Value | Rooms drawn | Set at |
|---|---|---|
| 1 | Central only | `Settings.c:1153`; also forced by `Main.c:191-192` |
| 3 | West, East, Central | `Settings.c:1164` |
| 9 | all nine | `Settings.c:1176`, `Settings.c:888`, `Settings.c:1257`, `Main.c:154` (default) |

`Main.c:191-192`:

```c
if ((numNeighbors > 1) && (thisMac.screen.right <= 512))
	numNeighbors = 1;
```

and `Settings.c:1164`/`:1176` refuse to select 3 or 9 unless `thisMac.screen.right > 512`. So on a
512-wide screen the value is pinned to 1.

`DrawFloorSupport()` is also gated on `numNeighbors > 3` (`RoomGraphics.c:123-124`), i.e. it only
runs in the 9-room mode, because the floor-support bands only exist between vertically stacked rooms.

**This is a determinism hazard.** With `numNeighbors == 1`, rooms 1..8 are never drawn, so:
* no neighbour-room `dinahs[]` entries are created (see the list of types that register from any room
  in §7.3), so `numDynamics` and every index shift;
* no neighbour-room `savedMaps[]`, so `numSavedMaps` shifts;
* no `RandomInt` calls from neighbour-room flames/tikis/coals/pendulums/stars, so the RNG stream
  diverges (§7.10);
* `numLights` ends up holding the central room's count either way, so lighting is unaffected.

A Go port that wants bit-identical replay across machines must therefore treat `numNeighbors` as part
of the replay seed, exactly like the screen size (§7.9).

### 4.5 `numLights` and `isLit` are per-room

`numLights = GetNumberOfLights(localNumbers[X])` immediately before each pair of draw calls means
`isLit = (numLights > 0)` at `ObjectDrawAll.c:38` is the *drawn room's* lit state. So a dark north
room and a lit central room render differently in the same pass. `GetNumberOfLights`
(`GliderPRO/Sources/Room.c:970-1095`) counts, in play mode (`thisHousePtr->rooms[where]`, not the
editor's `thisRoom`):

1. background whitelist → returns 1 immediately: `kGarden`, `kSkywalk`, `kMeadow`, `kField`, `kRoof`,
   `kSky`, `kStratosphere`, `kStars`;
2. `kDirt` → returns 1 only if all 8 `tiles[]` are 0;
3. otherwise: +1 for each `kDoorInLf`, `kDoorInRt`, `kWindowInLf`, `kWindowInRt`, `kWallWindow`
   **unconditionally** (they are holes to the outside, always lit), plus +1 for each of the eight
   light types (`kCeilingLight`..`kInvisLight`) whose `data.f.state` is true.

Three details of `GetNumberOfLights` that are easy to get wrong in a port:

* **The object loop is gated on `if (count == 0)`** (`Room.c:1068` in the play branch, `:1004` in the editor branch).
  So the background check *short-circuits*: an outdoor room reports `numLights == 1` exactly, no matter
  how many lamps it contains. `numLights` is not a count of lights, it is a tri-state
  "0 = dark / 1 = outdoors or one indoor source / n = n indoor sources".
* **The function has two whole bodies, and they read different struct fields.** `theMode == kEditMode`
  (`Room.c:976`) walks `thisRoom->objects[i]` and tests `data.f.initial` (`:1026`); the play-mode branch
  walks `thisHousePtr->rooms[where].objects[i]` and tests `data.f.state` (`:1090`). In `lightType` those
  are **different bytes**: `initial` is at object offset 10, `state` at offset 11 (§2.4). A Go port with
  one flattened light struct and one `Lit bool` field will make the editor and the engine agree where
  the original deliberately disagreed — the editor previews the *authored* state, the engine renders
  the *current* state.
* **The play-mode branch locks the house handle** (`HGetState`/`HLock` at `:1035-1036`,
  `HSetState` at `:1096`) and `DrawARoomsObjects` does the same around its own loop
  (`ObjectDrawAll.c:40-41, :964`). The lock/unlock pairs nest; `HGetState`/`HSetState` (rather than
  `HUnlock`) is what makes nesting safe on the classic Mac Memory Manager. In Go there is nothing to
  replace this with — delete it.

The 22 object cases in `DrawARoomsObjects` that consult `isLit` are enumerated in §10; the notable
asymmetry is that `kWallWindow` (`:933`) is drawn **without** an `isLit` test even though it *counts*
as a light, which is self-consistent (a window is visible in the dark) but easy to get wrong.

### 4.6 Contrast: `ListAllLocalObjects` uses a *different* order

`GliderPRO/Sources/Objects.c:300-347`:

```c
	numMasterObjects = 0;                        /* :305 */
	numLocalMasterObjects = 0;                   /* :306 */
	nHotSpots = 0;                               /* :307 */

	ListOneRoomsObjects(kCentralRoom);           /* :312  <-- FIRST */

	if (numNeighbors > 1)                        /* :314 */
	{
		ListOneRoomsObjects(kEastRoom);           /* :316 */
		ListOneRoomsObjects(kWestRoom);           /* :317 */
	}
	if (numNeighbors > 3)                        /* :320 */
	{
		ListOneRoomsObjects(kNorthRoom);          /* :322 */
		ListOneRoomsObjects(kNorthEastRoom);      /* :323 */
		ListOneRoomsObjects(kSouthEastRoom);      /* :324 */
		ListOneRoomsObjects(kSouthRoom);          /* :325 */
		ListOneRoomsObjects(kSouthWestRoom);      /* :326 */
		ListOneRoomsObjects(kNorthWestRoom);      /* :327 */
	}
	/* then the O(n^2) localLink correlation, :332-346 */
```

| Order | Sequence |
|---|---|
| `ListAllLocalObjects` (builds `masterObjects[]`, `hotSpots[]`) | Central, E, W, N, NE, SE, S, SW, NW |
| `DrawLocale` (builds `dinahs[]`, `savedMaps[]`, …) | NW, NE, N, SW, SE, S, W, E, Central |

These are **near-exact reverses of each other**, but not exactly (E/W and NE/SE/SW/NW orders differ).
Therefore:

* `masterObjects[]` index order ≠ `dinahs[]` index order. The only mapping between them is the
  `dynaNum` field written back at `ObjectDrawAll.c:959`.
* `masterObjects[0..23]` is **always** the central room, because Central is listed first and
  `ListOneRoomsObjects` always fills all 24 slots when the room exists. This accidental invariant is
  what makes the `masterObjects[i].hotNum` bug in the switch cases (§7.7) *almost* work.
* `hotSpots[]` is central-only (`Objects.c:283-286`), so hot-spot indices are unaffected by the
  neighbour ordering.

A Go port must implement **both** orders and must not "tidy" them into one.

### 4.7 `GetNumberOfLights(kRoomIsEmpty)` — an out-of-bounds read on every absent neighbour

`DrawLocale` calls `GetNumberOfLights(localNumbers[X])` at `:80, 84, 88, 92, 96, 100, 107, 112, 117`
with **no `kRoomIsEmpty` check**, and `GetNumberOfLights` (`Room.c:970-1095`) does not check either:
its play-mode branch dereferences `thisHousePtr->rooms[where]` directly. With `where == -1` this reads
348 bytes *before* `rooms[0]` — which in the house handle is the tail of the 866-byte house header
(the `Str255 banner`/`trailer` region), i.e. in-bounds of the handle but garbage data.

Why it is harmless in the original: the resulting garbage `numLights` is only consumed by
`DrawRoomBackground` and `DrawARoomsObjects`, and **both return immediately** for an empty room
(`RoomGraphics.c:179-191` and `ObjectDrawAll.c:33-34`). The garbage never reaches the screen.

Why it matters in Go: `rooms[-1]` panics. A port must add the `kRoomIsEmpty` guard, and must be aware
that this changes nothing observable — do not go hunting for a behaviour to reproduce.

The same missing guard exists in the `IsRoomAStructure(localNumbers[i])` call at `RoomGraphics.c:70`.

### 4.8 What runs after the nine rooms

`RoomGraphics.c:123-129`:

```c
	if (numNeighbors > 3)
		DrawFloorSupport();          /* :124  the between-floor bands + manhole cut-outs */
	RestoreWorkMap();                /* :125  backSrcMap -> workSrcMap, whole rect */
	shadowVisible = IsShadowVisible();  /* :126 */
	takingTheStairs = false;         /* :127 */
	SetGWorld(wasCPort, wasWorld);   /* :129 */
```

`DrawFloorSupport` (`RoomGraphics.c:257-376`) paints up to six horizontal support bands, each
`kFloorSupportTall` = 44 px, chosen by the six `isStructure[]` flags, and for each band it rescans
`tempManholes[0..numTempManholes-1]` calling `LoadScaledGraphic(kManholeThruFloor /* 3957 */, &tempManholes[i])`
to punch the manhole holes back through. `tempManholes[]` was populated during the object draw by
`AddTempManholeRect(&itsRect)` at `ObjectDrawAll.c:277` — i.e. **the manhole case deliberately defers
part of its rendering until after all nine rooms are composited**, because the support band that would
cover it does not exist yet when the object is drawn. This is the only such deferral in the whole
object pass and a Go port must keep the two-phase structure.

`RestoreWorkMap` (`RoomGraphics.c:389-398`) does a single whole-rect `CopyBits` from `backSrcMap` to
`workSrcMap` and then `AddRectToWorkRects(&workSrcRect)` so the first frame pushes everything to the
screen.

---

## 5. The rect pipeline, the clip test, and the ambient port

The three steps every drawing case performs before it can draw are the same, and they are worth
pinning down precisely because they are where a port most easily drifts by a few pixels.

### 5.1 Step 1 — `GetObjectRect`

`GliderPRO/Sources/ObjectRects.c:31-268`. Signature `void GetObjectRect (objectType *who, Rect *theRect)`.
It builds the object's rect in **room-local** coordinates (origin = top-left of the room's 512×322
play field) by one of five strategies depending on `what`:

| Strategy | Types | Formula |
|---|---|---|
| sprite-sheet size at `topLeft` | most blowers, prizes, transports, switches, appliances, enemies | `*theRect = srcRects[what]; ZeroRectCorner(theRect); QOffsetRect(theRect, topLeft.h, topLeft.v)` |
| authored bounds | furniture (`data.b.bounds`), clutter (`data.i.bounds`) | copied verbatim |
| derived thickness | `kTable` (`kTableThick` 8), `kShelf` (`kShelfThick` 6), `kStool` (`kStoolThick` 25) | bounds with a computed height |
| derived from `distance`/`tall`/`length` | `kFloorVent`-family columns, `kInvisBlower`, `kLiftArea`, `kInvisObstacle`, `kInvisBounce`, `kInvisTrans`, `kDeluxeTrans`, mailboxes, ducts | width/height from the data fields; `kFloorColumnWide` 4, `kCeilingColumnWide` 24, `kFanColumnThick` 16, `kFanColumnDown` 20 (`ObjectRects.c:13-16`) |
| fixed size, ignores sheet | lights, some clutter | literal constants |

**Dead code in the switch.** `ObjectRects.c:68-71` is four statements sitting inside the `switch` with
**no `case` label**, immediately after the `break;` that ends `kLiftArea` at `:66`:

```c
    case kLiftArea:                                                  /* :63 */
    QSetRect(itsRect, 0, 0, who->data.a.distance, who->data.a.tall * 2);
    QOffsetRect(itsRect, who->data.a.topLeft.h, who->data.a.topLeft.v);
    break;                                                           /* :66 */

    *itsRect = srcRects[who->what];                                  /* :68  UNREACHABLE */
    ZeroRectCorner(itsRect);                                         /* :69 */
    QOffsetRect(itsRect, who->data.a.topLeft.h, who->data.a.topLeft.v);
    break;                                                           /* :71 */
```

They are unreachable — a leftover from a case group that was merged into the `:43-61` block. A port
should simply not transcribe them; noted here so a future reader does not mistake them for a missing
`case` that changes behaviour.

The sizes that matter for reasoning about the `SectRect` test, from `srcRects[]`
(`GliderPRO/Sources/StructuresInit2.c:306-475`, `InitSrcRects`):

| Type | `what` | width × height | sheet position (l, t) |
|---|---|---|---|
| `kTV` | 0x65 | 92 × 77 | (0, 0) |
| `kMacPlus` | 0x63 | 48 × 58 | (0, 49) |
| `kToaster` | 0x62 | 48 × 27 | (0, 22) |
| `kCoffee` | 0x66 | 43 × 64 | (0, 107) |
| `kOutlet` | 0x67 | 16 × 24 | (64, 22) |
| `kVCR` | 0x68 | 96 × 22 | (0, 0) |
| `kStereo` | 0x69 | 128 × 53 | (0, 0) |
| `kMicrowave` | 0x6A | 92 × 59 | (0, 0) |
| `kShredder` | 0x61 | 73 × 22 | (0, 0) |
| `kSparkle` | 0x2D | 20 × 19 | (0, 70) |
| `kBalloon` | 0x71 | 24 × 30 | (0, 0) |
| `kCopterLf`/`kCopterRt` | 0x72/0x73 | 32 × 30 | (0, 0) |
| `kDartLf`/`kDartRt` | 0x74/0x75 | 64 × 19 | (0, 0) |
| `kBall` | 0x76 | 32 × 32 | (0, 0) |
| `kDrip` | 0x77 | 16 × 12 | (0, 0) |
| `kFish` | 0x78 | 36 × 33 | (0, 0) |
| `kCobweb` | 0x79 | 54 × 45 | (0, 0) |
| `kMirror` | 0x82 | 64 × 64 | (0, 0) |
| `kStar` | 0x2C | 32 × 31 | (48, 0) |
| `kGreaseRt` | 0x28 | 32 × 27 | (0, 243) |
| `kGreaseLf` | 0x29 | 32 × 27 | (0, 324) |
| `kCuckoo` | 0x24 | 40 × 80 | (0, 148) |
| `kTaper` | 0x08 | 20 × 59 | (0, 209) |
| `kCandle` | 0x09 | 32 × 30 | (0, 179) |
| `kStubby` | 0x0A | 20 × 36 | (0, 268) |
| `kTiki` | 0x0B | 27 × 28 | (21, 268) |
| `kBBQ` | 0x0C | 64 × 33 | (0, 0) |
| `kManhole` | 0x1D | 123 × 22 | (0, 0) |

`srcRects` is `Rect *` allocated as `NewPtr(sizeof(Rect) * kNumSrcRects)` with `kNumSrcRects` = 0x90
= 144 (`StructuresInit2.c:271`, `GliderDefines.h:437`) — indexed directly by `what`, so the sparse
`what` numbering leaves holes in the table that are never read.

### 5.2 Step 2 — `OffsetRectRoomRelative`

`GliderPRO/Sources/ObjectRects.c:1093-1131`:

```c
void OffsetRectRoomRelative (Rect *theRect, short neighbor)
{
	QOffsetRect(theRect, playOriginH, playOriginV);       /* to central-room screen coords */
	switch (neighbor)
	{
		case kCentralRoom:   /* nothing */                        break;
		case kNorthRoom:     QOffsetRect(theRect, 0, -kVertLocalOffset);            break;
		case kNorthEastRoom: QOffsetRect(theRect, kRoomWide, -kVertLocalOffset);    break;
		case kEastRoom:      QOffsetRect(theRect, kRoomWide, 0);                    break;
		case kSouthEastRoom: QOffsetRect(theRect, kRoomWide, kVertLocalOffset);     break;
		case kSouthRoom:     QOffsetRect(theRect, 0, kVertLocalOffset);             break;
		case kSouthWestRoom: QOffsetRect(theRect, -kRoomWide, kVertLocalOffset);    break;
		case kWestRoom:      QOffsetRect(theRect, -kRoomWide, 0);                   break;
		case kNorthWestRoom: QOffsetRect(theRect, -kRoomWide, -kVertLocalOffset);   break;
	}
}
```

with `kRoomWide` = 512 = 0x200 and `kVertLocalOffset` = **322** = 0x142 (`GliderDefines.h:499`,
`:501`). `playOriginH`/`playOriginV` come from `InterfaceInit.c:203-204`:

```c
playOriginH = (RectWide(&thisMac.screen) - kRoomWide) / 2;    /* (screenW - 512) / 2 */
playOriginV = (RectTall(&thisMac.screen) - kTileHigh) / 2;    /* (screenH - 322) / 2 */
```

`VerticalRoomOffset(neighbor)` (`ObjectRects.c:1067-1089`) is the vertical half of the same table,
returned as a value rather than applied: −`kVertLocalOffset` for N/NE/NW, +`kVertLocalOffset` for
S/SE/SW, 0 for Central/E/W. It exists because several `Draw*` helpers need the *vertical* origin
separately (they draw a shadow or a column that must be clipped against the room band), and they are
called as e.g. `DrawStool(&itsRect, playOriginV + VerticalRoomOffset(neighbor))`
(`ObjectDrawAll.c:266`).

**Exhaustive list.** `grep -n 'playOriginV\|VerticalRoomOffset' ObjectDrawAll.c` finds exactly **six**
calls that pass a vertical origin to a `Draw*` helper. Every other appearance of `playOriginV` in the
file is one of the sixteen `QOffsetRect(&rectA, -playOriginH, -playOriginV)` / `QOffsetRect(&itsRect,
-playOriginH, -playOriginV)` un-offsettings used before `AddDynamicObject` (§7.3), not a helper
argument:

| Call site | Call | Second argument | Helper signature |
|---|---|---|---|
| `ObjectDrawAll.c:177` | `DrawTiki(&itsRect, playOriginV + VerticalRoomOffset(neighbor))` | **corrected** | `void DrawTiki (Rect *theRect, short down)` `ObjectDraw.c:69` |
| `:208` | `DrawTable(&itsRect, playOriginV)` | **plain** | `void DrawTable (Rect *tableTop, short down)` `ObjectDraw.c:147` |
| `:259` | `DrawDeckTable(&itsRect, playOriginV)` | **plain** | `void DrawDeckTable (Rect *tableTop, short down)` `ObjectDraw.c:773` |
| `:266` | `DrawStool(&itsRect, playOriginV + VerticalRoomOffset(neighbor))` | **corrected** | `void DrawStool (Rect *theRect, short down)` `ObjectDraw.c:886` |
| `:487` | `DrawMailboxLeft(&itsRect, playOriginV + VerticalRoomOffset(neighbor))` | **corrected** | — |
| `:493` | `DrawMailboxRight(&itsRect, playOriginV + VerticalRoomOffset(neighbor))` | **corrected** | — |

The other furniture helpers take **no** origin argument at all — they are not part of this
inconsistency and an earlier draft of this document was wrong to include them:

| Helper | Signature | Call site |
|---|---|---|
| `DrawShelf` | `void DrawShelf (Rect *shelfTop)` `ObjectDraw.c:263` | `ObjectDrawAll.c:215` |
| `DrawCabinet` | `void DrawCabinet (Rect *cabinet)` `ObjectDraw.c:358` | `:222` |
| `DrawCounter` | `void DrawCounter (Rect *counter)` `ObjectDraw.c:498` | `:245` |
| `DrawDresser` | `void DrawDresser (Rect *dresser)` `ObjectDraw.c:643` | `:252` |

**So the defect is precisely two calls: `DrawTable` at `:208` and `DrawDeckTable` at `:259`.** They
receive `playOriginV` with **no** `VerticalRoomOffset(neighbor)`, so everything the helper positions
from `down` lands at the *central* room's vertical origin even when drawing a north or south room —
an error of exactly `kVertLocalOffset` = 322 pixels.

This is not just a shadow. `DrawTable` (`GliderPRO/Sources/ObjectDraw.c:147-259`) uses `down` three
times, with `#define kTableBaseTop 296`, `#define kTableShadowTop 312`, `#define kTableShadowOffset 12`
at the top of the function:

1. the drop-shadow ellipse, painted at `kTableShadowTop + down`;
2. the pedestal legs, drawn from `tableTop->bottom` down to `kTableBaseTop + down`, guarded by
   `if (tableTop->bottom < kTableBaseTop + down)` (`ObjectDraw.c:214`);
3. the table-base sprite, positioned by `QOffsetRect(&tempRect, …, kTableBaseTop + down)` at
   `ObjectDraw.c:253-254` and `CopyMask`ed at `:255-258` — **outside** that guard.

Because (3) is unguarded, for a table in a north or south room the entire base graphic is emitted 322
pixels away from its top, i.e. inside the *central* room's band, where the player sees a stray table
pedestal with no table on it. (1) and (2) are misplaced too.

Three further `DrawTable` details a porter needs:

* it does `GetGWorld(&wasCPort, &wasWorld)` at `:174` and `SetGWorld(backSrcMap, nil)` at `:175`,
  restoring with `SetGWorld(wasCPort, wasWorld)` at `:250` — i.e. the port is switched *inside* the
  helper, so `DrawARoomsObjects` never has to set it (see §11.3);
* the final `CopyMask` at `:255-258` names `backSrcMap` explicitly as its destination `BitMap`, which is
  why it still works after the port has been restored at `:250`;
* `InsetRect(tableTop, 0, 1)` at `:190` and `InsetRect(tableTop, 0, -1)` at `:192` mutate and then
  restore the caller's `itsRect` **in place**. `itsRect` is a local of `DrawARoomsObjects` and is
  re-derived from `GetObjectRect` on the next iteration, so the mutation is invisible — but a Go port
  that passes a `Rect` by value here will silently change nothing, while one that passes a pointer must
  keep both `InsetRect`s or the table top will be drawn 2 px too tall;
* `if (thisMac.isDepth == 4)` at `:159-165` swaps the four palette indices (`brownC` 11, `tanC` 9,
  `dkRedC` 14, `blackC` 15) for the 8-bit constants `k8BrownColor`, `k8TanColor`, `k8DkRed2Color`,
  `k8BlackColor` at `:167-172` — another 4-bit-depth special case (cf. §9.5).

**This is reachable from shipped content, on a large scale.** The defect fires whenever a table's room
is drawn in one of the six non-central-row slots (N, NE, NW, S, SE, SW), i.e. whenever the table's room
has any *non-empty* neighbour at `floor ± 1`, `suite − 1 … +1`. Counting `kTable` (`0x11`) and
`kDeckTable` (`0x19`) — **and only those two**, because they are the only two types whose helper is
called with an uncorrected `playOriginV` (see the six-call table above; `kCounter` `0x17`,
`kShelf` `0x12`, `kDresser` `0x18` and `kStool` `0x1A` are *not* affected in this way) — over the whole
corpus:

| House | `kTable` 0x11 | `kDeckTable` 0x19 | Exposed tables | In how many rooms |
|---|---|---|---|---|
| `Leviathan` | 54 | 7 | 61 | 52 |
| `ImagineHouse PRO II` | 25 | 1 | 26 | 25 |
| `Slumberland` | 16 | 4 | 20 | 19 |
| `Titanic` | 16 | 2 | 18 | 17 |
| `Metropolis` | 14 | 0 | 14 | 11 |
| `CD Demo House` | 11 | 2 | 13 | 10 |
| `Land of Illusion` | 9 | 0 | 9 | 7 |
| `Grand Prix` | 7 | 0 | 7 | 5 |
| `In The Mirror` | 6 | 1 | 7 | 6 |
| `Teddy World` | 6 | 1 | 7 | 7 |
| `Nemo's Market` | 0 | 5 | 5 | 4 |
| `Rainbow's End` | 3 | 2 | 5 | 5 |
| `The Asylum Pro` | 0 | 5 | 5 | 3 |
| `Davis Station` | 2 | 0 | 2 | 2 |
| `Art Museum` | 1 | 0 | 1 | 1 |
| **corpus** | **170** | **30** | **200** | **174** |

(Seven houses contain no exposed tables at all: `California or Bust!`, `Castle o' the Air`,
`Demo House`, `Empty House`, `Fun House`, `Sampler`, `SpacePods`.) In other words **every single table
in the shipped corpus that lives in a vertically-connected room** exhibits the defect at
`numNeighbors = 9` — and the corpus contains **exactly 200** `kTable` plus `kDeckTable` objects in
non-empty rooms, so that is **100 % of every table in every shipped house**. It is
not a corner case; it is what
Glider PRO looks like. A faithful port must reproduce the un-corrected calls, mis-placed bases and all.
See §10 for the per-case table and "Porting notes" item 9.

**What the player actually sees**, worked out from the geometry. `tableSrc` is 64 × 22 at sheet
`(0, 0)` (`StructuresInit.c:310-311`), so the misplaced base occupies
v ∈ [`playOriginV` + 296, `playOriginV` + 318] and the shadow ellipse (height
`RectWide(tableTop) / 10` = 64/10 = 6, centred on `kTableShadowTop` = 312) occupies
v ∈ [`playOriginV` + 309, `playOriginV` + 315). **Both are entirely inside the central room's
512 × 322 band** [`playOriginV`, `playOriginV` + 322). Because the central room is drawn **last**
(§4) and `DrawRoomBackground` fully repaints `localRoomsDest[where]`
(`RoomGraphics.c:187` for a dark/empty room, tiles otherwise), the central room's background at
`RoomGraphics.c:119` **overpaints the stray base and shadow completely**. So they are invisible. Which
leaves the pedestal legs — and there the two vertical directions behave oppositely:

| Table's room slot | `tableTop->bottom` vs `kTableBaseTop + playOriginV` | Legs drawn? | Visible result |
|---|---|---|---|
| N, NE, NW | `bottom` ≈ `playOriginV` − 322 + y, so **less than** `playOriginV` + 296 → guard at `ObjectDraw.c:214` is **true** | yes, 5 `ColorLine`s (`:218-227`) from `bottom` down to `playOriginV` + 296 | a 5-px-wide grey/black pedestal stripe runs from the table straight down through the rest of the north room's viewport. The part inside the north band is drawn *after* the north background (`:89` then `:90`) and *before* the W/E/central backgrounds, so **it is visible**. |
| S, SE, SW | `bottom` ≈ `playOriginV` + 322 + y, so **greater than** `playOriginV` + 296 → guard is **false** | no | the table top floats with **no pedestal, no base and no shadow at all**. |
| Central, E, W | `VerticalRoomOffset` returns 0 for these slots (`ObjectRects.c:1067-1089`), so plain `playOriginV` **is** the correct value | yes | correct |

So the defect has two distinct visible signatures, and both are common: 200 tables corpus-wide.
`vShadow = tableTop->bottom + RectWide(tableTop) / 4 - 2` (`ObjectDraw.c:229`) picks between a
full-height and a truncated inner highlight at `:230-247`; for a north-room table `vShadow` is far above
`kTableBaseTop + down`, so the `else` branch at `:239-247` runs and the highlight stops at `vShadow`
rather than at the base — which is why the stripe has a lighter top and darker bottom.

Two reasons it was never fixed: (a) at `numNeighbors` 1 or 3 the N/S/NE/NW/SE/SW slots are never drawn
at all (`RoomGraphics.c:78`), so the defect cannot appear in the small-screen configuration the author
used most; and (b) the most obviously wrong artefacts (base, shadow) are exactly the ones the
central-room overpaint hides. **A port that draws the rooms in any other order, or that composites
rooms into separate buffers, will expose the base and shadow too and will not look like Glider PRO.**

**Second, separate inconsistency in the same block of cases:** `kTable` (`:204-209`), `kShelf`
(`:211-216`), `kDresser` (`:248-253`), `kDeckTable` (`:255-260`) and `kStool` (`:262-267`) test **only**
`isLit` and have **no** `SectRect` clip test, while their immediate neighbours `kCabinet` (`:221`),
`kFilingCabinet`/`kOzma` (`:229`), `kWasteBasket`/`kMilkCrate` (`:237`) and `kCounter` (`:244`) all test
`(SectRect(&itsRect, &testRect, &whoCares)) && isLit`. The un-clipped five rely entirely on QuickDraw's
own clipping to the destination `GWorld`; a Go port whose blitter does not clip will write out of
bounds. See §5.4.

### 5.3 Step 3 — `testRect`, the clip rect

`ObjectDrawAll.c:36-37`:

```c
testRect = houseRect;
ZeroRectCorner(&testRect);
```

`houseRect` is built once at startup (`InterfaceInit.c:196-201`):

```c
houseRect = thisMac.screen;
houseRect.bottom -= kScoreboardTall;                                  /* 20 */
if (houseRect.right  > kMaxViewWidth)  houseRect.right  = kMaxViewWidth;    /* 1536 */
if (houseRect.bottom > kMaxViewHeight) houseRect.bottom = kMaxViewHeight;   /* 1026 */
```

and `ZeroRectCorner` (`RectUtils.c:57-63`) slides it so `left = top = 0` while preserving width and
height. So `testRect` is `(0, 0, min(screenW, 1536), min(screenH - 20, 1026))`.

Observed values (from the resolution sweep, §7.9):

| screen | `houseRect` | `testRect` | `playOriginH` | `playOriginV` |
|---|---|---|---|---|
| 512 × 384 | (0,0,512,364) | (0,0,512,364) | 0 | 31 |
| 640 × 480 | (0,0,640,460) | (0,0,640,460) | 64 | 79 |
| 800 × 600 | (0,0,800,580) | (0,0,800,580) | 144 | 139 |
| 832 × 624 | (0,0,832,604) | (0,0,832,604) | 160 | 151 |
| 1024 × 768 | (0,0,1024,748) | (0,0,1024,748) | 256 | 223 |
| 1152 × 870 | (0,0,1152,850) | (0,0,1152,850) | 320 | 274 |
| 1280 × 1024 | (0,0,1280,1004) | (0,0,1280,1004) | 384 | 351 |
| 1600 × 1200 | (0,0,1536,1026) | (0,0,1536,1026) | 544 | 439 |

(The last row shows both clamps engaging: 1600 → 1536 and 1180 → 1026.)

### 5.4 `SectRect` semantics and the empty-rect trap

`if (SectRect(&itsRect, &testRect, &whoCares))` is the standard guard. QuickDraw's `SectRect`
(Toolbox, `<QuickDraw.h>`) computes the intersection into the third argument and returns `true`
**iff the intersection is non-empty**. Critically:

* it treats rects as half-open — `right`/`bottom` are exclusive — so a rect touching the edge
  (`itsRect.left == testRect.right`) does **not** intersect;
* it returns `false` and sets the output to `(0,0,0,0)` for an empty intersection;
* it does **not** normalise: an inverted rect (`right < left`) is treated as empty.

A Go port needs exactly:

```go
func sectRect(a, b Rect) (Rect, bool) {
    r := Rect{max(a.Left, b.Left), max(a.Top, b.Top), min(a.Right, b.Right), min(a.Bottom, b.Bottom)}
    if r.Left >= r.Right || r.Top >= r.Bottom { return Rect{}, false }
    return r, true
}
```

The guard is a **whole-object visibility test, not a clip**. An object that straddles the screen edge
passes the test and is then drawn in full; the actual clipping is done by QuickDraw against the
destination `PixMap` bounds. So a Go port must clip in the blitter, not in the dispatcher.

### 5.5 Cases with **no** `SectRect` guard — enumerated

Most cases are of the form `GetObjectRect; OffsetRectRoomRelative; if (SectRect(...)) { draw; register; }`.
The exceptions are exhaustive and each one matters:

| Case | Lines | What is unguarded | Consequence |
|---|---|---|---|
| `kTiki` (0x0B) | `:173-183` | **everything** — draw *and* `AddTikiFlame` | a tiki torch in a far neighbour room is always drawn and always registers a flame; the only limiter is `AddTikiFlame`'s own `(h < 8) \|\| (v < 10)` check and `kMaxTikis` = 8 |
| `kMailboxLf` (0x33) | `:484-489` | draw | mailbox art always painted |
| `kMailboxRt` (0x34) | `:490-494` | draw | ditto |
| `kGreaseRt` fallen branch | `:382-385` | draw | a spilt grease puddle is always drawn |
| `kGreaseLf` fallen branch | `:403-406` | draw | ditto |
| all six switch types | `:518`, `:531`, `:544`, `:557`, `:570`, `:574` | `dynamicNum = masterObjects[i].hotNum` | the `dynaNum` write-back happens even for off-screen switches — which is *necessary*, since a switch can be triggered remotely, but see §7.7 for the aliasing bug |
| `kManhole` (0x1D) | `:272-281` | nothing (guarded), but `AddTempManholeRect` is called *before* the draw | ordering matters only if the draw could fail; it cannot |

The `kTiki` omission is the interesting one. Compare it to its three siblings `kTaper`/`kCandle`/
`kStubby`, which all use an elaborate proximity test (§5.5.1) instead. There is no comment; the most
likely reading is that it was overlooked. Reproduce it as-is.

#### 5.5.1 The candle-family proximity test (the most intricate geometry in the file)

`kTaper` at `:71-103` verbatim (`kCandle` `:105-137` and `kStubby` `:139-171` are byte-for-byte the
same shape with different hot-spot offsets):

```c
case kTaper:                                                     /* :71 */
GetObjectRect(&thisObject, &itsRect);                            /* :72 */
OffsetRectRoomRelative(&itsRect, neighbor);                      /* :73 */
if (SectRect(&itsRect, &testRect, &whoCares))                    /* :74 */
{
	if (isLit)                                                    /* :76 */
		DrawSimpleBlowers(thisObject.what, &itsRect);              /* :77 */
	if (neighbor == kCentralRoom)                                 /* :78 */
	{
		if (redraw)                                                /* :80 */
			ReBackUpFlames(localNumbers[neighbor], i);              /* :81 */
		else
			AddCandleFlame(localNumbers[neighbor], i,               /* :83 */
					itsRect.left + 10, itsRect.top + 7);            /* :84 */
	}
	else                                                          /* :86 */
	{
		QSetRect(&rectA, 0, 0, 16, 15);                            /* :88 */
		QOffsetRect(&rectA, itsRect.left + 10 - 8,                 /* :89 */
				itsRect.top + 7 - 15);
		rectB = localRoomsDest[kCentralRoom];                      /* :90 */
		rectB.top -= kFloorSupportTall;                            /* :91   -44 */
		rectB.bottom += kFloorSupportTall;                         /* :92   +44 */
		if (!SectRect(&rectA, &rectB, &whoCares))                  /* :93 */
		{
			if (redraw)                                             /* :95 */
				ReBackUpFlames(localNumbers[neighbor], i);           /* :96 */
			else
				AddCandleFlame(localNumbers[neighbor], i,            /* :98 */
						itsRect.left + 10, itsRect.top + 7);         /* :99 */
		}
	}
}
break;                                                           /* :103 */
```

Numbered semantics, close to the control flow:

1. Compute the candle's screen rect. If it does not intersect `testRect`, do nothing at all — no
   draw, no flame. (So an off-screen candle in *any* room, including the central one, is inert.)
2. If the room is lit, paint the candle body from the blower sheet.
3. If this is the **central** room, register (or, on redraw, re-capture) a flame unconditionally, at
   hot spot `(itsRect.left + 10, itsRect.top + 7)`.
4. Otherwise (a **neighbour** room), build `rectA` — a 16 × 15 rect positioned so that its
   *bottom-right* corner sits on the hot spot, hence the `- 8` horizontally (16/2) and `- 15`
   vertically (the full height); build `rectB` — the central room's 512 × 322 viewport
   (`localRoomsDest[kCentralRoom]`) fattened by `kFloorSupportTall` = 44 px above and below, i.e.
   512 × 410; and register the flame **only if `rectA` and `rectB` do NOT intersect**.

The purpose of step 4 is now clear: the flame renderer `srcCopy`s a 16 × 15 patch from a
`savedMaps[]` sheet straight into `workSrcMap` every even frame (`Render.c:212-214`). If a
neighbour-room candle's flame patch landed inside the central room's visible band, that blit would
stamp neighbour-room pixels over playfield pixels every frame. The 44-px fattening covers the floor
support bands drawn immediately above and below the central room by `DrawFloorSupport`. So the rule
is: *a neighbour room's candle animates only if its flame is clear of the central band and its
margins.*

The gap this leaves is a neighbour-room candle right at a seam: it is drawn (step 2) but never
animates, showing a static unlit wick. That is the shipped behaviour.

Note also that the flame is registered even when `isLit` is false — steps 3 and 4 are **outside** the
`if (isLit)`. An unlit room therefore has invisible candles with live, animating flames. That is also
the shipped behaviour, and it is used deliberately in the shipped houses: a dark room lit only by
candle flames.

Per-family offsets and gating, exhaustive:

| Type | `what` | flame hot spot | draw gated on | registration gated on | probe rect | lines |
|---|---|---|---|---|---|---|
| `kTaper` | 0x08 | `(left + 10, top + 7)` | `SectRect && isLit` | `SectRect`, then central ⇒ always / neighbour ⇒ `!SectRect(rectA, rectB)` | 16 × 15 at `(-8, -15)` from hot spot | `:71-103` |
| `kCandle` | 0x09 | `(left + 14, top + 7)` | same | same | 16 × 15 at `(-8, -15)` | `:105-137` |
| `kStubby` | 0x0A | `(left + 9, top + 7)` | same | same | 16 × 15 at `(-8, -15)` | `:139-171` |
| `kTiki` | 0x0B | `(left + 10, top **−** 9)` | `isLit` **only — no `SectRect` at all** | **nothing** — always registers | none | `:173-183` |
| `kBBQ` | 0x0C | `(left + 16, top + 9)` | `SectRect && isLit` | `SectRect` only — **not** central-gated, **not** proximity-tested | none | `:185-198` |

Two omissions to reproduce faithfully:

* **`kTiki` has no `SectRect` guard whatsoever** (`:173-183`). Its draw is gated only on `isLit`, and
  `AddTikiFlame` is called unconditionally for every tiki in every one of the nine rooms. The only
  limiters are `AddTikiFlame`'s own `(h < 8) || (v < 10)` coordinate check and `kMaxTikis` = 8. Note
  the hot spot is `top - 9` — *above* the object rect — which means a tiki near the top of a north
  room produces `v < 10` and is silently dropped.
* **`kBBQ` is not central-gated and not proximity-tested** (`:185-198`), so a BBQ in a neighbour room
  animates its coals into the playfield if the geometry lines up, which is exactly what the candle
  family's `rectA`/`rectB` test exists to prevent.

`AddCandleFlame` itself (`GliderPRO/Sources/DynamicMaps.c:316-344`) then applies its own guards:

```c
if ((numFlames >= kMaxCandles) || (h < 16) || (v < 15))     /* :321-322 */
	return;
if (thisMac.isDepth == 4)                                    /* :326 */
{	/* force h even */ }
QSetRect(&bounds, 0, 0, 16, 15 * kNumCandleFlames);          /* 16 x 75 */
... flames[numFlames].mode = RandomInt(kNumCandleFlames);    /* :338  consumes RNG */
```

`(h < 16) || (v < 15)` rejects flames whose 16 × 15 patch would start at a negative coordinate — a
crude off-left/off-top clip. `RandomInt(kNumCandleFlames)` (5) is the RNG consumption referenced in
§7.10. `thisMac.isDepth == 4` (16-colour mode) forces `h` even because the 4-bit blitter cannot
handle odd byte offsets; a Go port with a 32-bit framebuffer should still reproduce the coordinate
adjustment if it wants pixel-identical output at that depth, but can skip it for 8-bit and deeper.

The sibling registrars and their guards:

| Function | Line | Cap check | Coordinate check | Sheet size | RNG |
|---|---|---|---|---|---|
| `AddCandleFlame` | `DynamicMaps.c:316` | `numFlames >= kMaxCandles` (20) | `h < 16 \|\| v < 15` | 16 × (15 × 5) | `RandomInt(5)` `:338` |
| `AddTikiFlame` | `DynamicMaps.c:400` | `numTikiFlames >= kMaxTikis` (8) | `h < 8 \|\| v < 10` | 8 × (10 × 5) | `RandomInt(5)` `:422` |
| `AddBBQCoals` | `DynamicMaps.c:486` | `numCoals >= kMaxCoals` (8) | `h < 32 \|\| v < 9` | 32 × (9 × 4) | `RandomInt(4)` `:508` |
| `AddPendulum` | `DynamicMaps.c:570` | `numPendulums >= kMaxPendulums` (8) | `h < 32 \|\| v < 28` | 32 × (28 × 3) | `RandomInt(2)` `:594` |
| `AddStar` | `DynamicMaps.c:662` | `numStars >= kMaxStars` (4) | **none** | 32 × (31 × 6) | `RandomInt(6)` `:685` |

`AddPendulum` also sets the global `clockFrame = 10` (`DynamicMaps.c:578`), which is what makes
`RenderPendulums` fire on its very first frame (`Render.c:271` tests `clockFrame == 10 || == 15`).
Registering a second pendulum re-sets `clockFrame` to 10, so **all pendulums in a neighbourhood tick
in lockstep, phase-reset by the last one registered** — and since the last one registered is the
highest-slot pendulum in the central room, the phase depends on draw order.

### 5.6 The ambient GrafPort is `workSrcMap`, but every `Draw*` targets `backSrcMap`

This is the single most confusing thing about the object pass and the easiest to get wrong.

`DrawLocale` sets the ambient GWorld to `backSrcMap` at `RoomGraphics.c:75`. But
`DrawRoomBackground`, which runs immediately before each `DrawARoomsObjects` call, does
`SetPort((GrafPtr)workSrcMap)` at `RoomGraphics.c:238` in order to `LoadGraphicSpecial(pictID)` into
`workSrcMap` as a scratch buffer, then `CopyBits` the eight 64-px tiles from `workSrcMap` into
`backSrcMap` (`:241-252`, `src.left = tiles[i] * kTileWide`). **It never restores the port.** So when
`DrawARoomsObjects` runs, `thePort` is `workSrcMap` and the current GWorld is `backSrcMap`.

`DrawARoomsObjects` itself never calls `SetPort` or `SetGWorld`. It relies on every `Draw*` helper
naming its destination explicitly. Spot-checking `GliderPRO/Sources/ObjectDraw.c` and
`ObjectDraw2.c`, every helper is of one of these forms:

```c
CopyBits((BitMap *)*GetGWorldPixMap(<sheet>SrcMap),
         (BitMap *)*GetGWorldPixMap(backSrcMap),      /* explicit destination */
         &srcRect, destRect, srcCopy, nil);
```
```c
CopyMask((BitMap *)*GetGWorldPixMap(<sheet>SrcMap),
         (BitMap *)*GetGWorldPixMap(<sheet>MaskMap),
         (BitMap *)*GetGWorldPixMap(backSrcMap),      /* explicit destination */
         &srcRect, &maskRect, destRect);
```

and the few that use pen-drawing calls (`PaintRect`, `FrameRect`, `MoveTo`/`LineTo` — e.g.
`DrawTable`'s shadow, the invisible-object outlines in edit mode) do their own `SetPort` first.

Consequences:

1. The static object layer lands in `backSrcMap`. Nothing in `DrawARoomsObjects` writes
   `workSrcMap`. `RestoreWorkMap` (`RoomGraphics.c:389`) then copies the whole thing across.
2. `BackUpToSavedMap` reads from `backSrcMap` (`DynamicMaps.c:70-93`), which is why the swatch
   captures the objects drawn *so far* (§3.6).
3. Any Go port that models "current render target" as ambient state inherits a latent bug the
   original avoided by never using ambient state here. Make the destination an explicit parameter.

The three surfaces, for reference:

| GWorld | Contents | Written by |
|---|---|---|
| `backSrcMap` | clean composited static scene (backgrounds + objects + floor supports) | `DrawRoomBackground`, `DrawARoomsObjects`, `DrawFloorSupport`, appliance state-change re-blits |
| `workSrcMap` | per-frame scratch: `backSrcMap` plus this frame's animated elements | `RestoreWorkMap`, `CopyRectsQD`'s back→work pass, all `Render*` functions |
| `mainWindow` port | what the user sees | `CopyRectsQD`'s work→main pass; **and QuickTime, directly** (§8.4) |

---
## 6. Question (c) — drawn once into `backSrcMap` versus re-drawn every frame

There are **three** tiers, not two, and mixing them up is the most likely source of visual drift in a
port.

| Tier | Surface | When | Mechanism |
|---|---|---|---|
| **1. Static** | `backSrcMap` | once, during `DrawARoomsObjects` | direct `CopyBits`/`CopyMask` from a sprite sheet |
| **2. Animated** | `workSrcMap`, every frame | every frame from `RenderFrame` | `CopyBits` from a `savedMaps[]` strip (pre-composed during `DrawARoomsObjects`) |
| **3. State-latched** | `backSrcMap`, on change | when a `Handle*` sees a state transition | `CopyBits` of one indicator sub-rect + `AddRectToBackRects` |

### 6.1 Tier 1 — static art painted once into `backSrcMap`

Every `Draw*` call reachable from `DrawARoomsObjects` writes `backSrcMap` and is never repeated for
the life of the room (the sole exception being the `RedrawRoomLighting` path, which re-runs the whole
central-room pass). Complete list, grouped by helper:

| Helper | `what` codes it serves | Call site(s) in `ObjectDrawAll.c` |
|---|---|---|
| `DrawSimpleBlowers` | `kFloorVent` 0x01, `kCeilingVent` 0x02, `kFloorBlower` 0x03, `kCeilingBlower` 0x04, `kSewerGrate` 0x05, `kLeftFan` 0x06, `kRightFan` 0x07, `kGrecoVent` 0x0E, `kSewerBlower` 0x0F, `kTaper` 0x08, `kCandle` 0x09, `kStubby` 0x0A | `:68`, `:77`, `:111`, `:145` |
| `DrawTiki` | `kTiki` 0x0B | `:177` |
| `DrawPictSansWhiteObject` | `kBBQ` 0x0C, `kManhole` 0x1D, `kUpStairs` 0x31, `kDoorInLf` 0x37, `kDoorInRt` 0x38, `kWindowInLf` 0x3B, `kWindowInRt` 0x3C, `kTrunk` 0x1B, `kBooks` 0x1E, `kHipLamp` 0x54, `kDecoLamp` 0x55, `kGuitar` 0x64, `kCinderBlock` 0x6B, `kFlowerBox` 0x6C, `kFireplace` 0x84, `kBear` 0x87, `kVase1` 0x89, `kVase2` 0x8A, `kRug` 0x8E, `kChimes` 0x8F | `:191`, `:279`, `:470`, `:607` |
| `DrawTable` | `kTable` 0x11 | `:208` |
| `DrawShelf` | `kShelf` 0x12 | `:215` |
| `DrawCabinet` | `kCabinet` 0x13 | `:222` |
| `DrawPictObject` | `kFilingCabinet` 0x14, `kOzma` 0x81, `kDownStairs` 0x32, `kDoorExRt` 0x39, `kDoorExLf` 0x3A, `kWindowExRt` 0x3D, `kWindowExLf` 0x3E | `:230`, `:481` |
| `DrawSimpleFurniture` | `kWasteBasket` 0x15, `kMilkCrate` 0x16 | `:238` |
| `DrawCounter` | `kCounter` 0x17 | `:245` |
| `DrawDresser` | `kDresser` 0x18 | `:252` |
| `DrawDeckTable` | `kDeckTable` 0x19 | `:259` |
| `DrawStool` | `kStool` 0x1A | `:266` |
| `DrawRedClock` | `kRedClock` 0x21 | `:296` |
| `DrawBlueClock` | `kBlueClock` 0x22 | `:310` |
| `DrawYellowClock` | `kYellowClock` 0x23 | `:324` |
| `DrawCuckoo` | `kCuckoo` 0x24 | `:339` |
| `DrawSimplePrizes` | `kPaper` 0x25, `kBattery` 0x26, `kBands` 0x27, `kHelium` 0x2E, `kStar` 0x2C | `:362`, `:442` |
| `DrawGreaseRt` | `kGreaseRt` 0x28 | `:380` (standing), `:384` (fallen) |
| `DrawGreaseLf` | `kGreaseLf` 0x29 | `:401` (standing), `:405` (fallen) |
| `DrawFoil` | `kFoil` 0x2A | `:418` |
| `DrawMailboxLeft` | `kMailboxLf` 0x33 | `:487` |
| `DrawMailboxRight` | `kMailboxRt` 0x34 | `:493` |
| `DrawSimpleTransport` | `kFloorTrans` 0x35, `kCeilingTrans` 0x36 | `:501` |
| `DrawLightSwitch` | `kLightSwitch` 0x41 | `:516` |
| `DrawMachineSwitch` | `kMachineSwitch` 0x42 | `:529` |
| `DrawThermostat` | `kThermostat` 0x43 | `:542` |
| `DrawPowerSwitch` | `kPowerSwitch` 0x44 | `:555` |
| `DrawKnifeSwitch` | `kKnifeSwitch` 0x45 | `:568` |
| `DrawSimpleLight` | `kCeilingLight` 0x51, `kLightBulb` 0x52, `kTableLamp` 0x53 | `:588` |
| `DrawCustPictSansWhite` | `kCustomPict` 0x6E | `:614` |
| `DrawFlourescent` | `kFlourescent` 0x56 | `:621` |
| `DrawTrackLight` | `kTrackLight` 0x57 | `:628` |
| `DrawSimpleAppliance` | `kShredder` 0x61, `kCDs` 0x6D, `kToaster` 0x62 | `:639`, `:647` |
| `DrawMacPlus` | `kMacPlus` 0x63 | `:663` |
| `DrawTV` | `kTV` 0x65 | `:696` |
| `DrawCoffee` | `kCoffee` 0x66 | `:720` |
| `DrawOutlet` | `kOutlet` 0x67 | `:737` |
| `DrawVCR` | `kVCR` 0x68 | `:753` |
| `DrawStereo` | `kStereo` 0x69 | `:769` |
| `DrawMicrowave` | `kMicrowave` 0x6A | `:785` |
| `DrawDrip` | `kDrip` 0x77 | `:867` |
| `DrawFish` | `kFish` 0x78 | `:883` |
| `DrawPictWithMaskObject` | `kCobweb` 0x79, `kCloud` 0x8C | `:899` |
| `DrawMirror` | `kMirror` 0x82 | `:906` |
| `DrawSimpleClutter` | `kMousehole` 0x83, `kFaucet` 0x8D | `:919` |
| `DrawFlower` | `kFlower` 0x85 | `:926` |
| `DrawWallWindow` | `kWallWindow` 0x86 | `:933` |
| `DrawCalendar` | `kCalendar` 0x88 | `:940` |
| `DrawBulletin` | `kBulletin` 0x8B | `:947` |

That is **49 distinct helpers at 55 call sites**, mechanically extracted (regex `\bDraw[A-Za-z]+\s*\(`
over `:51-950`, attributing each call to the `case` labels that fall through into it). Three warnings
for a porter reading the C by eye:

* `DrawPictSansWhiteObject` is the workhorse — **20** `what` codes at **four** call sites (`:191`,
  `:279`, `:470`, `:607`). It is easy to mistake the `:607` group for `DrawPictObject`; it is not.
  `DrawPictObject` serves only 7 codes at 2 sites (`:230`, `:481`).
* There is no `DrawManhole`, no `DrawToaster` and no `DrawCustPict` in this file: `kManhole` uses
  `DrawPictSansWhiteObject` (`:279`), `kToaster` uses `DrawSimpleAppliance` (`:647`), `kCustomPict`
  uses `DrawCustPictSansWhite` (`:614`).
* `DrawBalloon` (`ObjectDraw2.c:914`) and `DrawBall` (`ObjectDraw2.c:954`) exist but are **editor-only**
  — their only callers are `ObjectEdit.c:2645` and `ObjectEdit.c:2659`. In play mode the six enemy
  types draw nothing here at all; see the next table.

Types with **no draw at all**, i.e. pure logic objects:

| `what` | Name | Lines | Note |
|---|---|---|---|
| −1 | `kObjectIsEmpty` | `:53-54` | explicit empty case |
| 0x0D | `kInvisBlower` | `:200-202` | invisible air source |
| 0x10 | `kLiftArea` | `:200-202` | invisible lift zone |
| 0x1C | `kInvisObstacle` | `:269-270` | invisible wall |
| 0x1F | `kInvisBounce` | `:283-284` | invisible bouncer |
| 0x2B | `kInvisBonus` | `:422-424` | invisible points |
| 0x2F | `kSlider` | `:422-424` | invisible slide-through |
| 0x2D | `kSparkle` | `:447-460` | registers a `dinahs[]` entry, draws nothing — the sparkle is entirely animation |
| 0x3F | `kInvisTrans` | `:504-506` | invisible transport |
| 0x40 | `kDeluxeTrans` | `:504-506` | invisible transport |
| 0x46 | `kInvisSwitch` | `:573-575` | writes `dynamicNum` only |
| 0x47 | `kTrigger` | `:577-580` | invisible trigger |
| 0x48 | `kLgTrigger` | `:577-580` | invisible trigger |
| 0x49 | `kSoundTrigger` | `:577-580` | invisible trigger |
| 0x58 | `kInvisLight` | `:631-632` | contributes to `numLights` only |
| 0x71 | `kBalloon` | `:796-805` | `AddDynamicObject` only — no static art (see below) |
| 0x72 | `kCopterLf` | `:807-816` | `AddDynamicObject` only |
| 0x73 | `kCopterRt` | `:818-827` | `AddDynamicObject` only |
| 0x74 | `kDartLf` | `:829-838` | `AddDynamicObject` only |
| 0x75 | `kDartRt` | `:840-849` | `AddDynamicObject` only |
| 0x76 | `kBall` | `:851-860` | `AddDynamicObject` only |

**16 of the 74 case groups draw nothing** (mechanically counted: groups whose body contains no
`Draw*(` call). Six of those 16 are the enemy types `kBalloon` … `kBall`: they are pure
`AddDynamicObject` registrations, so an enemy is invisible until the first `RenderFrame` composites it
from `dinahs[]`. That is why the enemy cases have **no** `SectRect` guard and **no** `isLit` gate — see
§9.2 and Porting notes item 9. Compare `kDrip` `0x77` (`:862-876`) and `kFish` `0x78` (`:878-892`),
which *are* enemy-adjacent dynamics but *do* also draw static art, behind a `SectRect` guard.

`kInvisLight` draws nothing but is still counted by `GetNumberOfLights` (`Room.c:1082-1091`, the
`case kInvisLight:` falling into `if (…data.f.state) count++;`), so it is the mechanism for "this room
is lit but has no visible light fixture".

### 6.2 Tier 2 — the `savedMaps[]` pre-composition trick, and what re-draws every frame

The engine has no sprite compositor for animation. Instead, at room-load time, for each animated
element it:

1. captures the background rect behind the element from `backSrcMap` into a fresh offscreen GWorld
   (`BackUpToSavedMap`), **or** allocates a *tall* GWorld and pre-renders every animation frame into
   it as a vertical strip (`AddCandleFlame` and friends);
2. stores the GWorld pointer plus its `(where, who)` tag in `savedMaps[numSavedMaps]`;
3. records the `savedMaps[]` index in the element's own record (`flames[i].who`,
   `pendulums[i].who`, `theStars[i].who`, `grease[i].mapNum`).

Then every frame `RenderFrame` `CopyBits` **one strip row** from that GWorld into `workSrcMap` and
queues the destination rect for the work→main blit.

`BackUpToSavedMap` (`GliderPRO/Sources/DynamicMaps.c:70-93`) verbatim:

```c
short BackUpToSavedMap (Rect *theRect, short where, short who)
{
	if (numSavedMaps >= kMaxSavedMaps)                 /* :75   24 */
		return(-1);                                     /* :76 */

	mapRect = *theRect;
	ZeroRectCorner(&mapRect);
	savedMaps[numSavedMaps].dest = *theRect;
	theErr = CreateOffScreenGWorld(&savedMaps[numSavedMaps].map,
			&mapRect, kPreferredDepth);                  /* :82   theErr IGNORED */
	CopyBits((BitMap *)*GetGWorldPixMap(backSrcMap),
			GetPortBitMapForCopyBits(savedMaps[numSavedMaps].map),
			theRect, &mapRect, srcCopy, nil);            /* :84-86 */
	savedMaps[numSavedMaps].where = where;
	savedMaps[numSavedMaps].who = who;
	numSavedMaps++;                                     /* :90 */
	return (numSavedMaps - 1);                          /* :92 */
}
```

`savedType` (`GliderPRO/Headers/GliderStructs.h:227-233`):

```c
typedef struct
{
	Rect       dest;    /* the screen rect this swatch belongs to */
	GWorldPtr  map;     /* the offscreen buffer                   */
	short      where;   /* house-file room index                   */
	short      who;     /* object slot 0..23                       */
} savedType, *savedPtr;
```

`savedMaps` is a **statically sized array** `savedType savedMaps[kMaxSavedMaps];` and
`NilSavedMaps` (`DynamicMaps.c:46-62`) disposes every GWorld and resets `where`/`who` to −1 and
`numSavedMaps` to 0. `NilSavedMaps` is called from `ReadyLevel` (`RoomGraphics.c:405`) and
`Play.c:110`, i.e. **before** `DrawLocale`, not inside it — so the GWorlds are recycled per room
change, and a `DrawLocale` that runs without a preceding `NilSavedMaps` (there is none in the shipped
code, but a port might introduce one) would leak 24 GWorlds per room.

Note `theErr` from `CreateOffScreenGWorld` is discarded at `:82`. On failure `savedMaps[].map` is
`nil`, the following `CopyBits` dereferences `nil` (a crash on a real Mac), and `numSavedMaps` is
still incremented so a valid-looking index is returned. A Go port should return −1 on allocation
failure — the callers already handle −1 correctly, so this is a safe divergence.

Which `savedMaps[]` entries `DrawARoomsObjects` creates:

| Case | `what` | Registration call | Purpose of the GWorld | Sheet dimensions | Line |
|---|---|---|---|---|---|
| `kRedClock` | 0x21 | `BackUpToSavedMap(&itsRect, …)` | background swatch, for "collected" erase | = `itsRect` | `:294` |
| `kBlueClock` | 0x22 | same | same | = `itsRect` | `:308` |
| `kYellowClock` | 0x23 | same | same | = `itsRect` | `:322` |
| `kCuckoo` | 0x24 | same, **plus** `AddPendulum` | swatch **and** a 32 × 84 pendulum strip | `itsRect`; 32 × (28 × 3) | `:336`, `:343` |
| `kPaper`, `kBattery`, `kBands`, `kHelium` | 0x25, 0x26, 0x27, 0x2E | `BackUpToSavedMap` | swatch | = `itsRect` | `:360` |
| `kFoil` | 0x2A | `BackUpToSavedMap` | swatch | = `itsRect` | `:416` |
| `kStar` | 0x2C | `BackUpToSavedMap`, **plus** `AddStar` | swatch **and** a 32 × 186 star strip | `itsRect`; 32 × (31 × 6) | `:434`, `:440` |
| `kTaper`, `kCandle`, `kStubby` | 0x08-0x0A | `AddCandleFlame` (internally calls `BackUpToSavedMap`) | 16 × 75 flame strip | 16 × (15 × 5) | `:83`, `:117`, `:151` |
| `kTiki` | 0x0B | `AddTikiFlame` | 8 × 50 flame strip | 8 × (10 × 5) | `:181` |
| `kBBQ` | 0x0C | `AddBBQCoals` | 32 × 36 coal strip | 32 × (9 × 4) | `:195` |
| `kGreaseRt`, `kGreaseLf` | 0x28, 0x29 | `AddGrease` (internally calls `BackUpToSavedMap`) | 32 × 108 strip | 32 × (27 × 4) | `:376`, `:397` |

So one object can consume **two** `savedMaps[]` slots (`kCuckoo`, `kStar`), which matters for the
24-slot budget (§9.4).

**Which of these re-draw every frame**, from `RenderFrame` (`GliderPRO/Sources/Render.c:639-671`):

| Renderer | Fires | Source | Destination | Line |
|---|---|---|---|---|
| `DrawReflection` ×1 or ×2 | if `hasMirror` | glider sheets, clipped to `mirrorRgn` | `workSrcMap` | `:641-646` |
| `HandleGrease` | always | `savedMaps[grease[i].mapNum]` | `workSrcMap` | `:647` |
| `RenderPendulums` | always (internally only on `clockFrame == 10 \|\| 15`) | `savedMaps[pendulums[i].who]` | `workSrcMap` | `:648` |
| `RenderFlames` | **even frames only** | `savedMaps[flames/tikiFlames/bbqCoals[i].who]` | `workSrcMap` | `:649-650` |
| `RenderStars` | **odd frames only** | `savedMaps[theStars[i].who]` | `workSrcMap` | `:651-652` |
| `RenderDynamics` | always | appliance/enemy sheets | `workSrcMap` | `:653` |
| `RenderFlyingPoints` | always | `pointsSrcMap` + mask | `workSrcMap` | `:654` |
| `RenderSparkles` | always | `bonusSrcMap` + `bonusMaskMap` | `workSrcMap` | `:655` |
| `RenderGlider` ×1 or ×2 | always | glider sheets + shadow | `workSrcMap` | `:656-658` |
| `RenderShreds` | always | `shredSrcMap` + mask | `workSrcMap` | `:659` |
| `RenderBands` | always | `bandsSrcMap` + mask | `workSrcMap` | `:660` |

`evenFrame` is toggled once per frame at `GliderPRO/Sources/Play.c:435`, so flames animate at
30 Hz and stars at 30 Hz on alternate frames, both on a 60 Hz-ish base
(`kTicksPerFrame` = 2 ticks = 2/60 s, so the base is nominally 30 fps and flames/stars run at
15 fps).

**This `RenderFrame` order is the animated z-order.** It is fixed and unrelated to object slot index:
grease is always under pendulums, which are always under flames/stars, which are under dynamics,
which are under points, sparkles, gliders, shreds and bands. A Go port must keep the sequence
literally.

The dirty-rect protocol every renderer follows:

* `AddRectToWorkRects(&rect)` — queue for the `workSrcMap` → `mainWindow` blit this frame
  (`Render.c:65-80`, clamped to `justRoomsRect`, cap `kMaxGarbageRects - 1` = 47);
* `AddRectToBackRects(&rect)` — queue for the `backSrcMap` → `workSrcMap` **restore** at the end of
  this frame (`Render.c:84-99`, clamped to `workSrcRect`, same cap), which is how the animated pixel
  is erased before the next frame draws it somewhere else;
* `CopyRectsQD` (`Render.c:616-635`) performs work→main first, then back→work, then
  `numWork2Main = numBack2Work = 0`.

Renderers that only queue a work rect and no back rect (flames, tikis, coals, pendulums, stars) do so
because they always redraw the *same* rect, so the previous frame's pixels are always fully covered.
Renderers of moving objects queue both.

### 6.3 Tier 3 — appliance indicators re-blitted into `backSrcMap` on state change

`DrawARoomsObjects` paints the appliance's **whole body including its current indicator state** once
(`DrawTV(&itsRect, thisObject.data.g.state, isLit)` at `:696`, and the same shape for
`DrawMacPlus` `:663`, `DrawCoffee` `:720`, `DrawVCR` `:753`, `DrawMicrowave` `:785`, `DrawOutlet`
`:737`, and `DrawStereo(&itsRect, isPlayMusicGame, isLit)` `:769`). Thereafter the indicator is
updated **not** every frame but on transition, by the corresponding `Handle*` function writing a
small sub-rect straight into `backSrcMap` and queueing it for the back→work restore.

`HandleTV` (`GliderPRO/Sources/Dynamics.c:428-478`) is the canonical example:

```c
void HandleTV (short who)
{
	if (dinahs[who].timer > 0)
	{
		dinahs[who].timer--;
		if (dinahs[who].active)
		{
			if (dinahs[who].timer == 0)
			{
				if ((thisMac.hasQT) && (hasMovie) && (tvInRoom) &&
						(who == tvWithMovieNumber))
				{ }                                                 /* :439-440  deliberately empty */
				else
					AddRectToWorkRects(&dinahs[who].dest);           /* :443 */
			}
			else if (dinahs[who].timer == 1)
			{
				PlayPrioritySound(kTVOnSound, kTVOnPriority);
				if ((thisMac.hasQT) && (hasMovie) && (tvInRoom) &&
						(who == tvWithMovieNumber))
				{ }                                                 /* :451-452  deliberately empty */
				else
				{
					CopyBits((BitMap *)*GetGWorldPixMap(applianceSrcMap),
							(BitMap *)*GetGWorldPixMap(backSrcMap),   /* :455-456 */
							&tvScreen2, &dinahs[who].dest, srcCopy, nil);
					AddRectToBackRects(&dinahs[who].dest);
				}
			}
		}
		else                                                       /* turning OFF */
		{
			if (dinahs[who].timer == 0)
				AddRectToWorkRects(&dinahs[who].dest);              /* :466 */
			else if (dinahs[who].timer == 1)
			{
				PlayPrioritySound(kTVOffSound, kTVOffPriority);
				CopyBits((BitMap *)*GetGWorldPixMap(applianceSrcMap),
						(BitMap *)*GetGWorldPixMap(backSrcMap),      /* :470-471 */
						&tvScreen1, &dinahs[who].dest, srcCopy, nil);
				AddRectToBackRects(&dinahs[who].dest);
			}
		}
	}
}
```

Three things to extract:

1. **The write target is `backSrcMap`, not `workSrcMap`.** Tier-3 updates mutate the "clean"
   background so that all subsequent frames inherit the new indicator without redrawing it. This is
   the only place outside `DrawLocale` that writes `backSrcMap` during play.
2. `dinahs[who].dest` for an appliance is the **indicator** rect, not the object rect. For `kTV` it is
   `tvScreen1` zeroed and offset to `(where->left + playOriginH + 17, where->top + playOriginV + 10)`
   (`Dynamics3.c:265-269`) — which, because `where` = `itsRect` shifted by `(-playOriginH, -playOriginV)`,
   is exactly `(itsRect.left + 17, itsRect.top + 10)`, i.e. **the same rect the QuickTime movie box is
   built from** at `ObjectDrawAll.c:683-685`. That is not a coincidence; it is how the movie replaces
   the static CRT image pixel-for-pixel.
3. The two empty `{ }` blocks are the movie-TV special case: for `tvWithMovieNumber` the static CRT
   image is **not** blitted and the rect is **not** queued, because QuickTime is drawing there
   directly. See §8.5.

The tier-3 indicator offsets and source rects, from `AddDynamicObject` (`Dynamics3.c`) and
`StructuresInit.c:565-605`:

| Type | `dest` source rect | size | position on `applianceSrcMap` | offset from `itsRect` top-left | `AddDynamicObject` line |
|---|---|---|---|---|---|
| `kMacPlus` 0x63 | `plusScreen1` / `plusScreen2` | 32 × 22 | `(48, 127)` / `(48, 149)` | `(+10, +7)` | `:245-249` |
| `kTV` 0x65 | `tvScreen1` / `tvScreen2` | 64 × 49 | `(0, 171)` / `(0, 220)` | `(+17, +10)` | `:265-269` |
| `kCoffee` 0x66 | `coffeeLight1` / `coffeeLight2` | 8 × 4 | `(72, 171)` / `(72, 175)` | `(+32, +57)` | `:285-289` |
| `kOutlet` 0x67 | `outletSrc[i]`, i = 0..3 (`kNumOutletPicts` = 4, `GliderDefines.h:446`) | 16 × 24 | `(64, 22 + 24·i)` → `(64,22)`, `(64,46)`, `(64,70)`, `(64,94)` | `(0, 0)` | `:308-312` |
| `kVCR` 0x68 | `vcrTime1` / `vcrTime2` | 16 × 4 | `(64, 179)` / `(64, 183)` | see `Dynamics3.c:327-349` | `:327` |
| `kStereo` 0x69 | `stereoLight1` / `stereoLight2` | 4 × 1 | `(68, 171)` / `(68, 172)` | see `:350-369` | `:350` |
| `kMicrowave` 0x6A | `microOff` / `microOn` | 16 × 35 | `(64, 187)` / `(64, 222)` | see `:370-390` | `:370` |
| `kToaster` 0x62 | `breadSrc[i]`, i = 0..5 (`kNumBreadPicts` = 6, `GliderDefines.h:451`) | 32 × 29 | `(0, 29·i)` | `CenterRectInRect` then `VOffsetRect` | `:218-222` |

Sheet positions verified verbatim at `GliderPRO/Sources/StructuresInit.c:565-583`:

```c
QSetRect(&plusScreen1,  0, 0, 32, 22);  QOffsetRect(&plusScreen1,  48, 127);
QSetRect(&plusScreen2,  0, 0, 32, 22);  QOffsetRect(&plusScreen2,  48, 149);
QSetRect(&tvScreen1,    0, 0, 64, 49);  QOffsetRect(&tvScreen1,     0, 171);
QSetRect(&tvScreen2,    0, 0, 64, 49);  QOffsetRect(&tvScreen2,     0, 220);
QSetRect(&coffeeLight1, 0, 0,  8,  4);  QOffsetRect(&coffeeLight1, 72, 171);
QSetRect(&coffeeLight2, 0, 0,  8,  4);  QOffsetRect(&coffeeLight2, 72, 175);
for (i = 0; i < kNumOutletPicts; i++) {          /* :580  kNumOutletPicts = 4 */
    QSetRect(&outletSrc[i], 0, 0, 16, 24);  QOffsetRect(&outletSrc[i], 64, 22 + (i * 24));
}
for (i = 0; i < kNumBreadPicts; i++) {           /* :586  kNumBreadPicts = 6 */
    QSetRect(&breadSrc[i], 0, 0, 32, 29);   QOffsetRect(&breadSrc[i], 0, i * 29);
}
QSetRect(&vcrTime1,     0, 0, 16,  4);  QOffsetRect(&vcrTime1,     64, 179);
QSetRect(&vcrTime2,     0, 0, 16,  4);  QOffsetRect(&vcrTime2,     64, 183);
QSetRect(&stereoLight1, 0, 0,  4,  1);  QOffsetRect(&stereoLight1, 68, 171);
QSetRect(&stereoLight2, 0, 0,  4,  1);  QOffsetRect(&stereoLight2, 68, 172);
QSetRect(&microOn,      0, 0, 16, 35);  QOffsetRect(&microOn,      64, 222);
QSetRect(&microOff,     0, 0, 16, 35);  QOffsetRect(&microOff,     64, 187);
```

Note the `1`/`2` convention: **`…1` is the OFF image and `…2` is the ON image**, and `…2` sits directly
below `…1` on the sheet (171 + 49 = 220 for the TV; 127 + 22 = 149 for the Mac; 171 + 4 = 175 for the
coffee light; 179 + 4 = 183 for the VCR; 171 + 1 = 172 for the stereo). `AddDynamicObject` always
initialises `dest` from the `…1` (off) rect and lets `HandleTV`/`HandleMacPlus`/`HandleCoffee`/
`HandleVCR`/`HandleStereo` swap in `…2` when the appliance turns on. The microwave breaks the
convention: it uses named `microOff` (64, 187) and `microOn` (64, 222) — and note **`microOff` is
*above* `microOn`** on the sheet, i.e. the pair is declared in the opposite order from the way it is
laid out, at `StructuresInit.c:602-605`. Do not infer sheet positions from declaration order.

### 6.4 The `redraw == true` path: `RedrawRoomLighting`

`GliderPRO/Sources/RoomGraphics.c:434-461`, the only place tier-1 art is ever repainted mid-game:

```c
void RedrawRoomLighting (void)
{
	GetGWorld(&wasCPort, &wasWorld);
	SetGWorld(backSrcMap, nil);
	numLights = GetNumberOfLights(thisRoomNumber);
	roomV = (*thisHouse)->rooms[thisRoomNumber].floor;
	DrawRoomBackground(thisRoomNumber, kCentralRoom, roomV);
	DrawARoomsObjects(kCentralRoom, true);                  /* :451 */
	UpdateOutletsLighting(thisRoomNumber, numLights);       /* :453 */
	if (numNeighbors > 3)
		DrawFloorSupport();
	RestoreWorkMap();                                        /* or CopyRectBackToWork */
	SetGWorld(wasCPort, wasWorld);
	AddRectToWorkRects(&localRoomsDest[kCentralRoom]);       /* :458 */
}
```

It is reached from `Interactions.c:1083` when a light-type object is toggled. Because `redraw` is
`true`:

* no `dinahs[]` entries are created — the existing ones survive with their existing indices, which is
  the whole point (a light switch must not renumber the appliances);
* `ReBackUpSavedMap`/`ReBackUpFlames`/`ReBackUpTikiFlames`/`ReBackUpBBQCoals`/`ReBackUpPendulum`/
  `ReBackUpStar`/`ReBackUpGrease` re-capture the swatches against the newly lit/unlit background,
  which is essential or every animated element would keep punching the *old* lighting through;
* `masterObjects[].dynaNum` is not rewritten;
* `UpdateOutletsLighting` (`Trip.c:235-244`) then pushes the new `numLights` into every
  `dinahs[i].hVel` where `type == kOutlet && room == thisRoomNumber`, because the outlet's zap
  animation frame depends on the light count.

`ReBackUpSavedMap` (`DynamicMaps.c:100-124`) is a **linear search** over `savedMaps[0..numSavedMaps-1]`
matching `(where, who)`, returning the found index or −1. If the original `BackUpToSavedMap` had
failed (budget exhausted), the re-back-up also returns −1 and the prize stays invisible — consistent.

### 6.5 Summary answer to (c)

| Category | Once into `backSrcMap` at room load | Every frame into `workSrcMap` | On state change into `backSrcMap` |
|---|---|---|---|
| Blowers (vents, fans) | yes | — | — |
| Candle/tiki/BBQ **bodies** | yes | — | — |
| Candle/tiki/BBQ **flames** | pre-composed strip only | **yes** (even frames) | — |
| Furniture, clutter, custom PICTs | yes | — | — |
| Stairs, doors, windows, mailboxes, transports | yes | — | — |
| Switches (5 visible kinds) | yes, with linked object's state | — | — |
| Lights (3 visible kinds + flourescent + track) | yes | — | — |
| Prizes (clocks, paper, battery, bands, foil, helium) | yes, plus a background swatch | — | — |
| `kStar` | yes, plus swatch **and** 6-frame strip | **yes** (odd frames) | — |
| `kCuckoo` | yes, plus swatch **and** 3-frame pendulum strip | **yes** (on `clockFrame` 10/15) | — |
| Grease (standing) | yes, plus 4-frame strip | **yes** (`HandleGrease`, every frame) | — |
| `kSparkle` | **nothing drawn** | **yes** (`RenderDynamics` → sparkle handler) | — |
| Appliance **bodies** | yes | — | — |
| Appliance **indicators** | yes (initial state) | — | **yes** |
| Toaster **bread** | body only | **yes** | — |
| Enemies (balloon, copter, dart, ball, drip, fish) | yes (their resting art) | **yes** | — |
| `kTV` with the movie | body yes, CRT yes | QuickTime draws **direct to `mainWindow`** | suppressed for the movie TV |
| `kMirror` | yes | reflections drawn into `workSrcMap` via `mirrorRgn` clip | — |
| Floor supports + manhole cut-outs | yes, **after** all nine rooms | — | — |
| Glider, shadow, bands, shreds, flying points | — | **yes** | — |

---

## 7. Question (d) — where `dinahs[]` entries are registered, and why draw order fixes simulation order

### 7.1 `AddDynamicObject` — the append

`GliderPRO/Sources/Dynamics3.c:187-554`. Signature:

```c
short AddDynamicObject (short what, Rect *where, objectType *who,
		short room, short index, Boolean isOn);
```

| Parameter | Supplied by `DrawARoomsObjects` as | Meaning |
|---|---|---|
| `what` | a literal type constant (`kTV`, `kToaster`, …) — **not** `thisObject.what` | selects the per-type init block |
| `where` | `&rectA` (= `itsRect` shifted by `(-playOriginH, -playOriginV)`) for 10 of the 17 types; `&itsRect` **unshifted** for the six enemies plus `kBall` | the reference rect |
| `who` | `&thisObject` | read for `data.g.height`, `data.g.delay`, `data.h.length`, `data.h.delay` |
| `room` | `localNumbers[neighbor]` | house-file room index, stored in `dinahs[].room` |
| `index` | `i` | slot 0..23, stored in `dinahs[].byte0` |
| `isOn` | `thisObject.data.g.state` (appliances) or `data.h.state` (enemies) or `data.c.state` (`kSparkle`) | stored in `dinahs[].active` |

Structure:

```c
	if (numDynamics >= kMaxDynamicObs)      /* :193   18 */
		return (-1);                         /* :194 */

	dinahs[numDynamics].type = what;        /* :196 */
	switch (what)
	{
		case kSparkle:     /* :199 */ ... break;
		case kToaster:     /* :217 */ ... break;
		case kMacPlus:     /* :244 */ ... break;
		case kTV:          /* :264 */ ... break;
		case kCoffee:      /* :284 */ ... break;
		case kOutlet:      /* :307 */ ... break;
		case kVCR:         /* :327 */ ... break;
		case kStereo:      /* :350 */ ... break;
		case kMicrowave:   /* :370 */ ... break;
		case kBalloon:     /* :391 */ ... break;
		case kCopterLf:
		case kCopterRt:    /* :412 */ ... break;
		case kDartLf:
		case kDartRt:      /* :437 */ ... break;
		case kBall:        /* :465 */ ... break;
		case kDrip:        /* :496 */ ... break;
		case kFish:        /* :516 */ ... break;
		default:                             /* :546 */
		return (-1);                         /* :548   nothing appended */
	}
	numDynamics++;                          /* :551 */
	return (numDynamics - 1);               /* :553 */
```

Two early-out paths, both returning −1: the cap check at `:193` (nothing written) and the `default`
at `:546` (`dinahs[numDynamics].type` **has already been written** at `:196`, but `numDynamics` is not
incremented, so the write is to a scratch slot that the next successful call overwrites — harmless,
but a port must not treat it as a partial commit).

`kSparkle` consumes RNG at `:208`: `dinahs[numDynamics].timer = RandomInt(60) + 15;` — the only
`RandomInt` inside `AddDynamicObject`.

### 7.2 The 17 registration call sites in `ObjectDrawAll.c`

Exhaustive, in slot-scan order as they appear in the `switch`:

| # | Case | `what` | `AddDynamicObject` at | `where` argument | `isOn` argument | Gate |
|---|---|---|---|---|---|---|
| 1 | `kSparkle` | 0x2D | `:456-457` | `&rectA` | `data.c.state` | `SectRect` ∧ `!redraw` ∧ `neighbor == kCentralRoom` |
| 2 | `kToaster` | 0x62 | `:652-653` | `&rectA` | `data.g.state` | `SectRect` ∧ `!redraw` ∧ `neighbor == kCentralRoom` |
| 3 | `kMacPlus` | 0x63 | `:668-669` | `&rectA` | `data.g.state` | `SectRect` ∧ `!redraw` — **any room** |
| 4 | `kTV` | 0x65 | `:701-702` | `&rectA` | `data.g.state` | `SectRect` ∧ `!redraw` — **any room** |
| 5 | `kCoffee` | 0x66 | `:725-726` | `&rectA` | `data.g.state` | `SectRect` ∧ `!redraw` — **any room** |
| 6 | `kOutlet` | 0x67 | `:742-743` | `&rectA` | `data.g.state` | `SectRect` ∧ `!redraw` — **any room** |
| 7 | `kVCR` | 0x68 | `:758-759` | `&rectA` | `data.g.state` | `SectRect` ∧ `!redraw` — **any room** |
| 8 | `kStereo` | 0x69 | `:774-775` | `&rectA` | `data.g.state` | `SectRect` ∧ `!redraw` — **any room** |
| 9 | `kMicrowave` | 0x6A | `:790-791` | `&rectA` | `data.g.state` | `SectRect` ∧ `!redraw` — **any room** |
| 10 | `kBalloon` | 0x71 | `:802-803` | `&itsRect` | `data.h.state` | `neighbor == kCentralRoom` ∧ `!redraw` — **no `SectRect`** |
| 11 | `kCopterLf` | 0x72 | `:813-814` | `&itsRect` | `data.h.state` | same |
| 12 | `kCopterRt` | 0x73 | `:824-825` | `&itsRect` | `data.h.state` | same |
| 13 | `kDartLf` | 0x74 | `:835-836` | `&itsRect` | `data.h.state` | same |
| 14 | `kDartRt` | 0x75 | `:846-847` | `&itsRect` | `data.h.state` | same |
| 15 | `kBall` | 0x76 | `:857-858` | `&itsRect` | `data.h.state` | same |
| 16 | `kDrip` | 0x77 | `:872-873` | `&rectA` | `data.h.state` | `SectRect` ∧ `!redraw` ∧ `neighbor == kCentralRoom` |
| 17 | `kFish` | 0x78 | `:888-889` | `&rectA` | `data.h.state` | `SectRect` ∧ `!redraw` ∧ `neighbor == kCentralRoom` |

The canonical shape (`kTV`, `:697-703`) is:

```c
	if (!redraw)                                            /* :697 */
	{
		rectA = itsRect;                                     /* :699 */
		QOffsetRect(&rectA, -playOriginH, -playOriginV);     /* :700 */
		dynamicNum = AddDynamicObject(kTV, &rectA, &thisObject,
				localNumbers[neighbor], i, thisObject.data.g.state);  /* :701-702 */
		...
	}
```

Note the `-playOriginH, -playOriginV` shift: `rectA` is the object's rect in **room-local** coords
again, undoing the first half of `OffsetRectRoomRelative` but **keeping** the per-neighbour
`±kRoomWide`/`±kVertLocalOffset` shift. `AddDynamicObject` then re-adds `playOriginH`/`playOriginV`
for the appliance types (`Dynamics3.c:247-249`, `:267-269`, `:287-289`, `:310-312`) — a round trip
whose only effect is to strip and re-add the same constants. The six enemies plus `kBall` skip the
round trip entirely and pass `&itsRect` directly, which is why their `dest` rects are computed
differently inside `AddDynamicObject` (`:391-544` use `where` as-is, in screen coords).

**A porter must not "simplify" the round trip away** without checking each `AddDynamicObject` arm,
because the two conventions are not interchangeable: for the appliance types `where` is room-local,
for the enemy types it is already screen-absolute.

### 7.3 The three registration gates, and the room-scope split

| Gate | Types affected |
|---|---|
| `!redraw` | all 17 |
| `SectRect(&itsRect, &testRect, …)` | 11: `kSparkle`, `kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`, `kMicrowave`, `kDrip`, `kFish` |
| `neighbor == kCentralRoom` | 10: `kSparkle`, `kToaster`, `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish` |

So there are exactly **two** disjoint families, and the split is semantically meaningful:

* **Appliances that register from *any* of the nine rooms** — `kMacPlus`, `kTV`, `kCoffee`,
  `kOutlet`, `kVCR`, `kStereo`, `kMicrowave` (7 types). They are *scenery with an indicator*: you can
  see a neighbour room's TV flicker through a doorway, so it must be simulated. Their registration is
  gated on `SectRect` only.
* **Things that interact with the glider, which only exists in the central room** — `kSparkle`,
  `kToaster`, `kBalloon`, `kCopterLf/Rt`, `kDartLf/Rt`, `kBall`, `kDrip`, `kFish` (10 types). They
  register only for `kCentralRoom`.

`kBalloon`, `kCopterLf/Rt`, `kDartLf/Rt` and `kBall` (6 types) have **no `SectRect` guard at all** on
either the draw or the registration — because a flying enemy legitimately starts off-screen
(`kBalloonStart` = 310, `kCopterStart` = 8, `Dynamics3.c:13-14`) and must be simulated from before it
becomes visible.

This is the load-bearing asymmetry for §7.9: the seven "any room" appliances make `numDynamics`
depend on the **screen size**, because `SectRect` compares against `testRect` which is derived from
the screen.

### 7.4 Why draw order fixes simulation order — the proof

Four links in the chain, each individually verifiable:

**Link 1 — `AddDynamicObject` is a monotone append.** It writes `dinahs[numDynamics]`, then
`numDynamics++`, then returns `numDynamics - 1` (`Dynamics3.c:196`, `:551`, `:553`). There is no free
list, no reuse, no reordering, no sort. Therefore the *k*-th successful call owns index *k−1*.

**Link 2 — `ZeroDinahs` is the only reset and it runs before all nine rooms.**
`GliderPRO/Sources/Dynamics3.c:160-180`:

```c
void ZeroDinahs (void)
{
	for (i = 0; i < kMaxDynamicObs; i++)     /* :164 */
	{
		dinahs[i].type = kObjectIsEmpty;
		QSetRect(&dinahs[i].dest, 0, 0, 0, 0);
		dinahs[i].whole = dinahs[i].dest;
		dinahs[i].hVel = 0;   dinahs[i].vVel = 0;
		dinahs[i].timer = 0;  dinahs[i].frame = 0;
		dinahs[i].count = 0;  dinahs[i].position = 0;
		dinahs[i].room = -1;  dinahs[i].byte0 = 0;
		dinahs[i].active = false;
	}
	numDynamics = 0;                          /* :179 */
}
```

Called once, from `DrawLocale` at `RoomGraphics.c:52`, before any room is drawn. Note it does **not**
reset `byte1` or `moving` — a real, if subtle, cross-room state leak: a fish that was mid-jump
(`moving == true`) in the previous room leaves `moving` set in the slot a new dynamic object lands in.
`AddDynamicObject` happens to set `moving = false` in all 15 of its per-type arms, so the leak is
masked for every currently-defined type; a port that adds a type and forgets to initialise `moving`
would inherit the bug. Same for `byte1`, which every arm also sets to 0.

**Link 3 — both simulation and render iterate `0 .. numDynamics-1` in order.**
`HandleDynamics` (`Dynamics3.c:31-105`):

```c
void HandleDynamics (void)
{
	for (i = 0; i < numDynamics; i++)          /* :35 */
	{
		switch (dinahs[i].type)
		{
			case kSparkle:    HandleSparkleObject(i);  break;
			case kToaster:    HandleToast(i);          break;
			case kMacPlus:    HandleMacPlus(i);        break;
			case kTV:         HandleTV(i);             break;
			case kCoffee:     HandleCoffee(i);         break;
			case kOutlet:     HandleOutlet(i);         break;
			case kVCR:        HandleVCR(i);            break;
			case kStereo:     HandleStereo(i);         break;
			case kMicrowave:  HandleMicrowave(i);      break;
			case kBalloon:    HandleBalloon(i);        break;
			case kCopterLf:
			case kCopterRt:   HandleCopter(i);         break;
			case kDartLf:
			case kDartRt:     HandleDart(i);           break;
			case kBall:       HandleBall(i);           break;
			case kDrip:       HandleDrip(i);           break;
			case kFish:       HandleFish(i);           break;
		}
	}
}
```

`RenderDynamics` (`Dynamics3.c:112-154`) is the same loop with seven renderers
(`RenderToast`, `RenderBalloon`, `RenderCopter`, `RenderDart`, `RenderBall`, `RenderDrip`,
`RenderFish` — the nine appliance types have no per-frame renderer; they are tier-3).

**Link 4 — the simulation is order-sensitive.** Three concrete couplings:

* Sounds. `HandleToast`, `HandleMacPlus`, `HandleTV`, `HandleCoffee`, `HandleOutlet`, `HandleVCR`,
  `HandleStereo`, `HandleMicrowave` all call `PlayPrioritySound`, and the priority mixer keeps only
  the highest-priority sound per channel; two appliances transitioning on the same frame resolve by
  loop index.
* Collision. `HandleBall`, `HandleFish`, `HandleDart`, `HandleCopter` all read and can modify the
  glider's state within the loop (via `Interactions.c` helpers). Two enemies hitting the glider on
  the same frame resolve by loop index.
* Dirty rects. Every renderer appends to `work2MainRects[]` / `back2WorkRects[]`, which are processed
  in append order by `CopyRectsQD` and are hard-capped at 47. When the cap is hit, the *later*
  entries are silently dropped — so a high-index dynamic object is the one that loses its blit.

Therefore: **draw order → `AddDynamicObject` call order → `dinahs[]` index order → `HandleDynamics`
and `RenderDynamics` iteration order → observable behaviour.** ∎

And since draw order = (room order NW,NE,N,SW,SE,S,W,E,Central) × (slot order 0..23), the full
ordering rule is:

> `dinahs[]` is sorted by (room-draw-rank, object-slot-index), where room-draw-rank is
> NW=0, NE=1, N=2, SW=3, SE=4, S=5, W=6, E=7, Central=8, restricted to the types and gates in §7.3.

Central-room dynamics therefore always occupy the **highest** indices.

### 7.5 `dynaNum` is overloaded three ways — the single biggest porting trap

`masterObjects[].dynaNum` is documented in the header as "index into dinahs (if any)"
(`GliderPRO/Headers/GliderStructs.h:330`). It is not. It holds an index into one of **three
unrelated arrays**, selected by the object's type:

| Meaning | Types | Where `dynamicNum` is assigned | Consumers |
|---|---|---|---|
| index into `dinahs[]` (cap 18) | the 17 types in §7.2 | `AddDynamicObject`'s return | `Interactions.c:1090-1148` (the 18 `Toggle*` calls), `Triggers.c:136-170` (the 8 `Trigger*` calls) |
| index into `hotSpots[]` (cap 56) | `kLightSwitch` 0x41, `kMachineSwitch` 0x42, `kThermostat` 0x43, `kPowerSwitch` 0x44, `kKnifeSwitch` 0x45, `kInvisSwitch` 0x46 | `masterObjects[i].hotNum` at `ObjectDrawAll.c:518, 531, 544, 557, 570, 574` | `Triggers.c:128` → `TriggerSwitch(who)` → `HandleSwitches(&hotSpots[who])` (`Trip.c:146-149`) |
| index into `grease[]` (cap 16) | `kGreaseRt` 0x28, `kGreaseLf` 0x29 | `AddGrease` / `ReBackUpGrease` return at `ObjectDrawAll.c:374, 376, 395, 397` | `Interactions.c:1060-1061` and `Triggers.c:117-118` → `SpillGrease(who, index)` (`Grease.c:257-265`) |

`AddGrease` (`GliderPRO/Sources/Grease.c:206-249`):

```c
short AddGrease (short where, short who, short h, short v, short distance, Boolean isRight)
{
	if (numGrease >= kMaxGrease)                   /* :212   16 */
		return (-1);                                /* :213 */
	QSetRect(&src, 0, 0, 32, 27);
	QOffsetRect(&src, h, v);
	QSetRect(&bounds, 0, 0, 32, 27 * 4);
	savedNum = BackUpToSavedMap(&bounds, where, who);
	if (savedNum != -1)
	{	/* … fill grease[numGrease] … */
		numGrease++;
		return (numGrease - 1);
	}
	else
		return (-1);
}
```

`ReBackUpGrease` (`Grease.c:180-199`) is the redraw-path analogue: a linear search over
`grease[0..numGrease-1]` matching `(where, who)`, re-capturing only when
`mode == kGreaseIdle` (0) or `kGreaseFalling` (1), returning `i` or −1. Grease modes are
`kGreaseIdle` 0, `kGreaseFalling` 1, `kGreaseSpreading` 2, `kGreaseSpiltIdle` 3 (`Grease.c:16-19`).

`SpillGrease` (`Grease.c:257-265`):

```c
void SpillGrease (short who, short index)
{
	grease[who].mode = kGreaseFalling;
	grease[who].hotNum = index;
	...
}
```

So `masterObjects[].dynaNum` for a grease object is `who` (a `grease[]` index) and
`masterObjects[].hotNum` is `index` (a `hotSpots[]` index) — two different arrays, passed as two
arguments, from the same master entry.

**Go port guidance:** do not model `dynaNum` as a single `int`. Model it as a tagged union, or keep
three separate fields, or (cleanest) keep an opaque handle whose type is derived from
`theObject.what`. And note that a value of −1 is the universal "no entry", which every consumer
**fails to check** (§9.10).

### 7.6 The `masterObjects` write-back, in detail

`ObjectDrawAll.c:953-961`:

```
1.  if redraw: skip entirely.
2.  for n in 0 .. numMasterObjects-1:
3.      if masterObjects[n].objectNum == i AND masterObjects[n].roomNum == localNumbers[neighbor]:
4.          masterObjects[n].dynaNum = dynamicNum
```

Facts:

* It runs **once per slot per room**, i.e. 24 times per `DrawARoomsObjects` call, even for empty slots.
* `dynamicNum` is whatever the switch left in it — −1 by default (`:45`).
* It does not `break` out of the loop after a match, so a duplicated `(room, slot)` pair gets the same
  value written twice (see §3.5).
* `numMasterObjects` is 24 × (non-empty local rooms), so this is up to 216 iterations. Total cost per
  `DrawLocale`: 9 rooms × 24 slots × up to 216 = up to 46 656 comparisons. Negligible now; it was not
  negligible on a 68030, which is presumably why `DrawLocale` is only called on room change.

### 7.7 The `masterObjects[i].hotNum` aliasing bug in the switch cases

`ObjectDrawAll.c:518` (and `:531`, `:544`, `:557`, `:570`, `:574`):

```c
dynamicNum = masterObjects[i].hotNum;
```

`i` is the **room slot index** (0..23), used here as a **`masterObjects[]` index**. Those two index
spaces only coincide for one room. Specifically:

* `ListAllLocalObjects` calls `ListOneRoomsObjects(kCentralRoom)` **first** (`Objects.c:312`), and
  `ListOneRoomsObjects` unconditionally fills all `kMaxRoomObs` = 24 entries for a non-empty room
  (`Objects.c:266-295`). So `masterObjects[0..23]` is always the central room, and
  `masterObjects[i].objectNum == i` for `i` in 0..23.
* For any other room, `masterObjects[i]` is still a *central-room* entry. So a switch in the north
  room at slot 3 gets `masterObjects[3].hotNum` — the hot-spot index of the **central room's** object
  3.
* `hotNum` is only ever non-−1 for central-room objects anyway (`Objects.c:283-286` gates
  `CreateActiveRects` on `where == kCentralRoom`), so the aliased value is at least always a *valid*
  hot-spot index or −1 — it is just the wrong one.

Consequences:

1. A switch in a neighbour room gets a `dynaNum` pointing at an unrelated central-room hot spot.
   `FireTrigger` → `TriggerSwitch(dynaNum)` → `HandleSwitches(&hotSpots[dynaNum])` then operates on
   the wrong object.
2. If the central room's object `i` had no hot spot, `dynaNum` is −1 and `TriggerSwitch(-1)` reads
   `hotSpots[-1]`.
3. The line is **outside** the `SectRect` guard, so it runs for off-screen switches too — which is
   deliberate and correct (a remote trigger must work on a switch you cannot see); the aliasing is the
   bug, not the ungating.

The presumably intended code is the same `(roomNum, objectNum)` search the write-back at `:955-959`
performs. A faithful Go port must reproduce the aliasing (it is observable in houses that put
switches in neighbour rooms) but should log it, and a "fixed" mode is a reasonable option flag.

`kInvisSwitch` at `:573-575` is the purest form: the entire case body is
`dynamicNum = masterObjects[i].hotNum;` with no rect computation at all.

### 7.8 Byte-verified worked example: two rooms of "ImagineHouse PRO II"

Replaying the exact draw order over the real house bytes. **Both examples below are shown at
1280 × 1024 with `numNeighbors = 9`**, because that is the only configuration in which the neighbour
rooms contribute anything at all — see §7.9 for the resolution sweep and for the much smaller
640 × 480 results.

**Room 120, "All TVs on = Safe Passage"** (floor 1, suite 81, background PICT 2001). The
neighbourhood and the dynamic-registering objects in each room, in **draw order**:

```
draw#  slot          room   name                          registering objects (slot: type)
  1    NW  (8)        156   'Exeunt'                      4: kOutlet   6: kTV   7: kVCR
  2    NE  (2)        118   'Powerful Motions'            3: kOutlet   4: kOutlet
  3    N   (1)        119   'Whole lotta toastin''        -   (its 4 kToasters are kCentralRoom-only)
  4    SW  (6)        124   'I'm sick of the Basement!'   -
  5    SE  (4)        122   '…and you're homefree'        -
  6    S   (5)        123   'It was all worth it'         -
  7    W   (7)        157   'Emergence'                   -
  8    E   (3)        121   'The Shredder Bank'           -
  9    Central (0)    120   'All TVs on = Safe Passage'   1: kTV  2: kTV  3: kTV  4: kTV
```

resulting `dinahs[]` **at 1280 × 1024**:

```
 idx  type       room  slot   source
   0  kOutlet     156     4   NW room, first registering slot
   1  kTV         156     6   NW room
   2  kVCR        156     7   NW room
   3  kOutlet     118     3   NE room
   4  kOutlet     118     4   NE room
   5  kTV         120     1   CENTRAL room  <-- gets the QuickTime movie
   6  kTV         120     2   CENTRAL room
   7  kTV         120     3   CENTRAL room
   8  kTV         120     4   CENTRAL room
numDynamics = 9      tvWithMovieNumber = 5      tvInRoom = true
```

Everything the six questions ask about is visible in that one table:

* (a) within the NW room the order is slot 4, 6, 7 — ascending;
* (b) NW is drawn first so it takes indices 0-2, Central is drawn last so it takes 5-8;
* (d) the four central TVs are simulated in slot order 1,2,3,4 at indices 5,6,7,8;
* (e) `tvWithMovieNumber` = 5 = the *lowest-index* TV **in the central room** — note that
  `dinahs[1]` is also a TV, in the NW room, and it does **not** get the movie because
  `neighbor != kCentralRoom`;
* (f) 9 of 18 slots used, so no exhaustion here.

At **640 × 480** the same room yields only the four central TVs, so `dinahs[0..3]` are those TVs and
`tvWithMovieNumber = 0`: every one of the indices above, and the identity of the movie-playing TV,
changes with the display. That is the point of §7.9.

**Room 119, "Whole lotta toastin'"** — the worst case found in any shipped house, 14 of 18 slots at
1280 × 1024 (10 of 18 at 640 × 480, where only the South room reaches inside `testRect`). Its
neighbourhood is *not* room 120's: N 115, NE 117, E 118, SE 121, S 120, SW 157, W 156, NW 155.

```
 idx  type       room  slot   source room
   0  kTV         120     1   South
   1  kTV         120     2   South
   2  kTV         120     3   South
   3  kTV         120     4   South
   4  kOutlet     156     4   West
   5  kTV         156     6   West
   6  kVCR        156     7   West
   7  kOutlet     118     3   East
   8  kOutlet     118     4   East
   9  kToaster    119     3   CENTRAL
  10  kToaster    119     4   CENTRAL
  11  kToaster    119     5   CENTRAL
  12  kToaster    119     6   CENTRAL
  13  kDrip       119    19   CENTRAL
numDynamics = 14      four slots of headroom      tvWithMovieNumber = -1 (no TV in the central room)
```

Note the room ordering is visible: South (draw #6) before West (#7) before East (#8) before Central
(#9) — which is exactly the `RoomGraphics.c:80-120` order, and *not* compass order, and *not*
`ListAllLocalObjects` order. Note also that although `dinahs[0]`, `[1]` and `[5]` are TVs,
`tvWithMovieNumber` stays −1 here, because none of them is in the central room (§8).

### 7.9 Byte-verified: `numDynamics` and `tvWithMovieNumber` depend on **screen resolution**

Sweeping the same room 120 across screen sizes:

```
screen      nn  playOrigin   testRect              numDynamics  tvWithMovieNumber
 512x384     9  (  0,  31)   (0,0, 512, 364)             4              0
 640x480     9  ( 64,  79)   (0,0, 640, 460)             4              0
 640x480     3  ( 64,  79)   (0,0, 640, 460)             4              0
 640x480     1  ( 64,  79)   (0,0, 640, 460)             4              0
 800x600     9  (144, 139)   (0,0, 800, 580)             4              0
 832x624     9  (160, 151)   (0,0, 832, 604)             4              0
1024x768     9  (256, 223)   (0,0,1024, 748)             5              1
1152x870     9  (320, 274)   (0,0,1152, 850)             8              4
1280x1024    9  (384, 351)   (0,0,1280,1004)             9              5
1600x1200    9  (544, 439)   (0,0,1536,1026)             9              5
```

(The 640×480 rows for `numNeighbors` 3 and 1 give the same numbers here only because the neighbour
appliances are already outside `testRect` at that size; on a 1280×1024 screen the `nn = 1` row drops
to `numDynamics = 4, tvWithMovieNumber = 0`.)

The mechanism, precisely:

1. `playOriginH = (screenW - 512) / 2` and `playOriginV = (screenH - 322) / 2`
   (`InterfaceInit.c:203-204`), so a bigger screen pushes the central room further right and down.
2. `testRect` = `(0, 0, min(screenW, 1536), min(screenH - 20, 1026))` (§5.3), so a bigger screen has a
   bigger clip rect.
3. Neighbour-room appliance rects are `playOrigin ± {512, 322}` (§5.2). On a small screen they fall
   outside `testRect`; on a large screen they fall inside.
4. The seven "any room" appliance types (§7.3) register **only if** `SectRect` passes.

⇒ **`numDynamics`, every `dinahs[]` index, `tvWithMovieNumber`, and hence which TV plays the movie,
are all functions of the display resolution and of the `numNeighbors` preference.**

The same mechanism affects `savedMaps[]` (whose `BackUpToSavedMap` calls are all inside `SectRect`
guards), `flames[]`/`tikiFlames[]`/`bbqCoals[]`/`pendulums[]`/`theStars[]` (same), and therefore the
RNG (§7.10).

### 7.10 RNG coupling: `DrawLocale` consumes random numbers, and how many depends on the display

Every animated-element registrar seeds its starting frame from `RandomInt`:

| Call site | Expression | Range | Line |
|---|---|---|---|
| `AddCandleFlame` | `flames[numFlames].mode = RandomInt(kNumCandleFlames)` | 0..4 | `DynamicMaps.c:338` |
| `AddTikiFlame` | `tikiFlames[numTikiFlames].mode = RandomInt(kNumTikiFlames)` | 0..4 | `DynamicMaps.c:422` |
| `AddBBQCoals` | `bbqCoals[numCoals].mode = RandomInt(kNumBBQCoals)` | 0..3 | `DynamicMaps.c:508` |
| `AddPendulum` | `pendulums[numPendulums].toOrFro = (RandomInt(2) == 0)` | 0..1 | `DynamicMaps.c:594` — note the draw picks the **swing direction**, not the frame: `mode` is hard-set to the constant `1` at `:593` |
| `AddStar` | `theStars[numStars].mode = RandomInt(6)` | 0..5 | `DynamicMaps.c:685` |
| `AddDynamicObject` (`kSparkle` arm) | `dinahs[numDynamics].timer = RandomInt(60) + 15` | 15..74 | `Dynamics3.c:208` |

Each of those is reached only through the gates in §5.5.1 / §7.3, all of which involve `SectRect`
against the resolution-derived `testRect`, or the `(h < …) || (v < …)` coordinate checks, or a cap.
Therefore **the number of `RandomInt` calls made during `DrawLocale` is a function of screen size and
`numNeighbors`**, and after a room change two machines with different displays have RNG streams offset
by a different number of draws.

Consequences for a Go port:

* Demo playback (`GetDemoInput`, `Play.c`) and any replay/determinism feature must pin screen size and
  `numNeighbors`, not just the seed.
* If the port wants cross-resolution determinism it must decouple the animation-phase RNG from the
  gameplay RNG — two separate streams. That is a **divergence from the original** and must be a
  documented, opt-in choice.
* `RandomInt` itself must be reimplemented bit-exactly (see `docs/analysis/determinism.md` for the
  LCG); this document only establishes *how many* times it is called and *in what order*.

Order of RNG consumption within `DrawLocale` is fully determined: room draw order × slot order ×
per-case position of the `Add*` call. E.g. a room with a candle at slot 2 and a star at slot 5
consumes `RandomInt(5)` then `RandomInt(6)`.

---
## 8. Question (e) — the first-TV rule and `tvWithMovieNumber`

### 8.1 The code, verbatim

`GliderPRO/Sources/ObjectDrawAll.c:674-713`:

```c
case kTV:                                                            /* :674 */
GetObjectRect(&thisObject, &itsRect);                                /* :675 */
OffsetRectRoomRelative(&itsRect, neighbor);                          /* :676 */
if (SectRect(&itsRect, &testRect, &whoCares))                        /* :677 */
{
#ifdef COMPILEQT
	if ((thisMac.hasQT) && (hasMovie) && (neighbor == kCentralRoom) &&  /* :680 */
			(!tvInRoom))                                                /* :681 */
	{
		whoCares = tvScreen1;                                           /* :683 */
		ZeroRectCorner(&whoCares);                                      /* :684 */
		OffsetRect(&whoCares, itsRect.left + 17, itsRect.top + 10);      /* :685 */
		GetMovieBox(theMovie, &movieRect);                              /* :686 */
		CenterRectInRect(&movieRect, &whoCares);                        /* :687 */
		SetMovieBox(theMovie, &movieRect);                              /* :688 */
		theRgn = NewRgn();                                              /* :689 */
		RectRgn(theRgn, &whoCares);                                     /* :690 */
		SetMovieDisplayClipRgn(theMovie, theRgn);                        /* :691 */
		DisposeRgn(theRgn);                                             /* :692 */
		tvOn = thisObject.data.g.state;                                 /* :693 */
	}
#endif
	DrawTV(&itsRect, thisObject.data.g.state, isLit);                  /* :696 */
	if (!redraw)                                                       /* :697 */
	{
		rectA = itsRect;                                                /* :699 */
		QOffsetRect(&rectA, -playOriginH, -playOriginV);                /* :700 */
		dynamicNum = AddDynamicObject(kTV, &rectA, &thisObject,         /* :701 */
				localNumbers[neighbor], i, thisObject.data.g.state);     /* :702 */
#ifdef COMPILEQT
		if ((thisMac.hasQT) && (hasMovie) && (neighbor == kCentralRoom) && /* :704 */
				(!tvInRoom))                                             /* :705 */
		{
			tvWithMovieNumber = dynamicNum;                              /* :707 */
			tvInRoom = true;                                            /* :708 */
		}
#endif
	}
}
break;                                                               /* :713 */
```

### 8.2 The four conditions, dissected

Both QuickTime blocks test the identical four-term conjunction:

| Term | Source | Meaning | Constant? |
|---|---|---|---|
| `thisMac.hasQT` | `CheckOurEnvirons` (`Environ.c`) | QuickTime is installed | yes, for the process lifetime |
| `hasMovie` | `HouseIO.c:139` on successful `NewMovieFromFile`, `:158`/`:196` on failure/teardown | this house has a loadable `.mov` | yes, for the house lifetime |
| `neighbor == kCentralRoom` | the function parameter | we are drawing the playable room | varies per call |
| `!tvInRoom` | `RoomGraphics.c:59` (`false` at the top of every `DrawLocale`), `ObjectDrawAll.c:708` (`true`), `RoomGraphics.c:409` and `Play.c:226` (`false`) | no TV has claimed the movie yet in this neighbourhood | varies within a call |

`COMPILEQT` is a compile-time flag; when undefined the whole feature vanishes and the static CRT image
is used for every TV. A Go port should treat "no QuickTime" as the `#ifndef COMPILEQT` path, which is
strictly simpler and is also what a house with no movie file gets.

### 8.3 Why the rule selects the **lowest-slot TV in the central room**

Chain of reasoning:

1. `tvInRoom` is cleared to `false` and `tvWithMovieNumber` to −1 at `RoomGraphics.c:59-60`, at the top
   of `DrawLocale`, **before** any room is drawn.
2. The only place `tvInRoom` is ever set `true` is `ObjectDrawAll.c:708`, inside the `kTV` case, inside
   `if (!redraw)`, inside the same four-term guard.
3. Therefore for the first eight `DrawARoomsObjects` calls (NW, NE, N, SW, SE, S, W, E) the
   `neighbor == kCentralRoom` term is false and neither block runs. A TV in a neighbour room gets a
   `dinahs[]` entry and a static CRT image but never the movie.
4. On the ninth call (`kCentralRoom`), the slot scan runs `i = 0 .. 23`. The first `i` whose object is
   a valid on-screen `kTV`:
   * runs the geometry block at `:683-693` (movie box + clip region + `tvOn`),
   * draws its static CRT image at `:696` (yes — even the movie TV gets the static image painted once
     into `backSrcMap`; the movie then covers it),
   * registers into `dinahs[]` at `:701`,
   * sets `tvWithMovieNumber = dynamicNum` and `tvInRoom = true` at `:707-708`.
5. Every subsequent central-room TV finds `tvInRoom == true`, so both blocks are skipped: it gets the
   static CRT image and a `dinahs[]` entry but no movie.

**So: the movie goes to the lowest-numbered object slot in the central room that holds a `kTV` whose
screen rect intersects `testRect`.** Because the central room's own rect is always fully on screen
(it is centred by construction), the `SectRect` term is effectively always true for a central TV, so
in practice it is simply *the lowest slot index*.

Verified in §7.8: room 120 of "ImagineHouse PRO II" has four central TVs at slots 1, 2, 3, 4; slot 1
always wins. Its `tvWithMovieNumber` is 5 at 1280 × 1024 but 0 at 640 × 480 — the value is the winning
TV's `dinahs[]` index, not its slot index, and that index depends on how many neighbour appliances
registered ahead of it (§7.9).

### 8.4 The movie box geometry, exactly

```
whoCares = tvScreen1                       /* (0, 171, 64, 220) — a 64 x 49 sheet rect */
ZeroRectCorner(&whoCares)                  /* -> (0, 0, 64, 49) */
OffsetRect(&whoCares, itsRect.left + 17, itsRect.top + 10)
                                           /* -> (L+17, T+10, L+81, T+59) */
GetMovieBox(theMovie, &movieRect)          /* the movie's natural size */
CenterRectInRect(&movieRect, &whoCares)    /* centre the movie inside the 64x49 CRT hole */
SetMovieBox(theMovie, &movieRect)
theRgn = NewRgn(); RectRgn(theRgn, &whoCares)
SetMovieDisplayClipRgn(theMovie, theRgn)   /* clip to the CRT hole, not the movie */
DisposeRgn(theRgn)
```

Constants involved:

| Constant | Value | Source |
|---|---|---|
| `tvScreen1` | `(0, 171, 64, 220)` — 64 wide × 49 tall at sheet position (0, 171) | `StructuresInit.c:570-571` |
| `tvScreen2` | `(0, 220, 64, 269)` — 64 × 49 at (0, 220) — the "on" CRT image | `StructuresInit.c:572-573` |
| CRT hole offset | `(+17, +10)` from the TV's top-left | `ObjectDrawAll.c:685` |
| `kTV` object size | 92 × 77 | `StructuresInit2.c` `srcRects[kTV]` |

Sanity check: the 64 × 49 hole at `(+17, +10)` inside a 92 × 77 body leaves an 11-px right bezel
(92 − 17 − 64) and an 18-px bottom bezel (77 − 10 − 49). Plausible for a TV sprite.

`CenterRectInRect(rectA, rectB)` (`GliderPRO/Sources/RectUtils.c:142-154`) preserves `rectA`'s size
and centres it in `rectB` using integer division:

```c
rectA->left  = rectB->left + (RectWide(rectB) - widthA) / 2;
rectA->right = rectA->left + widthA;
rectA->top   = rectB->top  + (RectTall(rectB) - tallA) / 2;
rectA->bottom= rectA->top  + tallA;
```

Note: **it does not scale.** If the movie is larger than 64 × 49 the movie box hangs outside the hole
and is then cropped by the display clip region. If it is smaller it is letterboxed inside the hole,
and the *uncovered* part of the hole shows whatever static CRT image `DrawTV` painted. The integer
division truncates toward zero for positive values and rounds *away from zero* for negative ones in
C89 — but both operands here are non-negative in practice, so `/2` truncates. A Go port must use the
same truncating division, not rounding.

**Coordinate space:** these are `mainWindow` port coordinates, because `Play.c:150` does
`SetMovieGWorld(theMovie, (CGrafPtr)mainWindow, nil)` — the movie draws **straight to the window**,
not into `workSrcMap`. It therefore bypasses `backSrcMap`, `workSrcMap`, the dirty-rect queues, and
`CopyRectsQD` entirely. This is the only element of the scene rendered outside the three-surface
pipeline, and it is why `HandleTV` has to *suppress* the static blits for `tvWithMovieNumber` (§6.3):
otherwise the CRT image would be blitted into `backSrcMap`, then `CopyRectsQD`'s work→main pass would
push it over the movie every frame.

### 8.5 Every consumer of `tvWithMovieNumber` and `tvInRoom`

| Site | Code | Purpose |
|---|---|---|
| `ObjectDrawAll.c:707-708` | `tvWithMovieNumber = dynamicNum; tvInRoom = true;` | the write |
| `RoomGraphics.c:59-60` | `tvInRoom = false; tvWithMovieNumber = -1;` | reset per `DrawLocale` |
| `RoomGraphics.c:407-412` | in `ReadyLevel`: if a movie was bound, clear both and `StopMovie` | reset per level load |
| `HouseIO.c:196-198` | `hasMovie = false; tvInRoom = false; tvWithMovieNumber = -1;` | reset on house close |
| `Dynamics.c:437-438` | in `HandleTV`, "turning on, timer hits 0": if this is the movie TV, do **not** `AddRectToWorkRects` | suppress the work→main push over the movie |
| `Dynamics.c:449-452` | in `HandleTV`, "turning on, timer hits 1": if this is the movie TV, do **not** blit `tvScreen2` into `backSrcMap` | suppress the static "on" CRT image |
| `Trip.c:43-56` | in `ToggleTV`: if this is the movie TV, `GoToBeginningOfMovie` + `StartMovie` + `tvOn = true` on activate, `StopMovie` + `tvOn = false` on deactivate | the actual play/stop |
| `Play.c:202-210` | in `NewGame`: `SetMovieActive(theMovie, true)`, and if `tvOn` then `StartMovie` + `MoviesTask` | arm the movie at game start |
| `Play.c:224-229` | in `NewGame`'s teardown: `tvInRoom = false; StopMovie; SetMovieActive(false)` | disarm |
| `Play.c:466-467`, `:491-492` | in `PlayGame`'s two loop variants: `if ((thisMac.hasQT) && (hasMovie) && (tvInRoom) && (tvOn)) MoviesTask(theMovie, 0);` | pump QuickTime once per frame, **before** `RenderFrame` |
| `Transit.c:304-307`, `:342-345`, `:381-384`, `:420-423` | on each of the four room-transition kinds: if `tvInRoom && tvOn`, `GoToBeginningOfMovie` + `StartMovie` | restart the movie when you re-enter a room whose TV is on |
| `MainWindow.c:62` | `if ((thisMac.hasQT) && (hasMovie))` — window update handling | redraw |

Note the asymmetry in `HandleTV`: the **turning-off** branch (`Dynamics.c:463-476`) has **no**
`tvWithMovieNumber` check, so the static "off" CRT image (`tvScreen1`) **is** blitted into
`backSrcMap` even for the movie TV. That is correct and necessary — `ToggleTV` has already called
`StopMovie`, so something must repaint the dark screen.

### 8.6 `tvOn` — a global mirroring one TV's state

`tvOn` is `Boolean tvOn;` defined at `Play.c:54`. It is:

* **initialised** at `ObjectDrawAll.c:693` to `thisObject.data.g.state` — the *authored* state of the
  first central TV, read at room-load time, and set **outside** the `if (!redraw)` block so a lighting
  redraw re-reads it (harmlessly, since the house object's `state` is the live state);
* **read** at `Play.c:205` (should the movie start immediately on entering the room?),
  `Play.c:466`/`:491` (should `MoviesTask` be pumped?), and `Transit.c:304`/`:342`/`:381`/`:420`
  (should the movie restart on transition?);
* **written** at `Trip.c:49`/`:54` when the player toggles the movie TV.

It is *not* generally "is any TV on" — it tracks only the movie TV. A house with the movie TV off and
three other TVs on has `tvOn == false`.

### 8.7 The `-1` defect: `tvWithMovieNumber` can be −1 while `tvInRoom` is `true`

`ObjectDrawAll.c:707` assigns `dynamicNum` unconditionally. `dynamicNum` at that point is whatever
`AddDynamicObject` returned at `:701`, which is **−1 if `numDynamics >= kMaxDynamicObs`**
(`Dynamics3.c:193-194`). `tvInRoom` is nevertheless set `true` at `:708`.

Downstream effects of the inconsistent state `tvInRoom == true && tvWithMovieNumber == -1`:

| Site | Behaviour with −1 |
|---|---|
| `Dynamics.c:437-438`, `:449-450` | `who == tvWithMovieNumber` is never true for a real `who ≥ 0`, so **all** TVs (including the intended movie TV, which has no `dinahs[]` entry at all) get their static CRT blits — visually correct |
| `Trip.c:43` | `tvWithMovieNumber == index` never true, so `ToggleTV` never starts or stops the movie |
| `Play.c:202-210` | `tvInRoom` is true, so `SetMovieActive(theMovie, true)` runs, and if `tvOn` (set from the authored state at `:693`) then `StartMovie` runs — **the movie plays, un-stoppable, clipped to the CRT hole of a TV that has no dinahs entry** |
| `Play.c:466`/`:491` | `MoviesTask` is pumped every frame while `tvOn` |
| `Transit.c:304` etc. | the movie restarts on every room transition |

So the failure mode is a permanently-playing movie that the player cannot turn off. Reaching it
requires 18 dynamic objects to be registered *before* the first central TV — which, because the
central room is drawn last, means 18 dynamics in the eight neighbour rooms. Given the seven "any room"
appliance types, that needs 18 visible appliances across the neighbours, which is achievable on a
large screen in a dense house (the shipped maximum is 14 neighbour-room dynamics, in `Land of Illusion`
room 247 `Black On Black` at the 1536 × 1046 view — §9.9). A Go port should either reproduce the
bug behind a compatibility flag or guard `:707-708` with `if (dynamicNum != -1)` and document the
divergence.

A second, milder oddity: the geometry block at `:679-695` runs **before** `AddDynamicObject`, and it is
outside `if (!redraw)`. So on a `RedrawRoomLighting` pass the movie box is re-centred and re-clipped
against the same rect (no-op) and `tvOn` is re-read from the house object (also effectively a no-op,
since `Trip.c` keeps `data.g.state` and `dinahs[].active` in sync via `SetObjectState`). Harmless, but
a port that changes the redraw path must keep the geometry idempotent.

### 8.8 Byte-verified example, restated as a decision table

"ImagineHouse PRO II" room 120, at **1280 × 1024**, `numNeighbors = 9` (§7.8 — this is the only
resolution at which the neighbour rooms contribute at all):

| `dinahs[]` idx | type | room | slot | `neighbor` when registered | `!tvInRoom` at that moment | gets movie? |
|---|---|---|---|---|---|---|
| 0 | `kOutlet` | 156 | 4 | NW | true | n/a (not a TV) |
| 1 | `kTV` | 156 | 6 | NW | true | **no** — `neighbor != kCentralRoom` |
| 2 | `kVCR` | 156 | 7 | NW | true | n/a |
| 3 | `kOutlet` | 118 | 3 | NE | true | n/a |
| 4 | `kOutlet` | 118 | 4 | NE | true | n/a |
| 5 | `kTV` | 120 | 1 | **Central** | **true** | **YES** → `tvWithMovieNumber = 5`, `tvInRoom = true` |
| 6 | `kTV` | 120 | 2 | Central | false | no |
| 7 | `kTV` | 120 | 3 | Central | false | no |
| 8 | `kTV` | 120 | 4 | Central | false | no |

At 640 × 480 (and at every size up to 832 × 624) the five neighbour appliances are all outside
`testRect`, so the table collapses to its last four rows renumbered 0-3: `numDynamics = 4` and
`tvWithMovieNumber = 0` — **the same physical TV object, a different numeric identity.** At
1024 × 768 the split falls in between: `numDynamics = 5`, `tvWithMovieNumber = 1` (§7.9). Anything that persists
`tvWithMovieNumber` (a save file, a replay, a network packet) is therefore resolution-bound.

### 8.9 Mac/QuickTime specifics and their Go replacements

| Toolbox call | Line | What it does | Go replacement |
|---|---|---|---|
| `GetMovieBox(theMovie, &movieRect)` | `:686` | read the movie's current display rect | ask the video decoder for its natural size |
| `SetMovieBox(theMovie, &movieRect)` | `:688` | set the movie's display rect in its GWorld's coords | set the destination rect for the video blit |
| `NewRgn()` / `RectRgn()` / `DisposeRgn()` | `:689-692` | build a one-rect QuickDraw region | a `Rect` clip; regions are only needed here because `SetMovieDisplayClipRgn` demands one |
| `SetMovieDisplayClipRgn(theMovie, theRgn)` | `:691` | clip all movie output to that region | scissor rect on the video blit |
| `SetMovieGWorld(theMovie, (CGrafPtr)mainWindow, nil)` | `Play.c:150` | render the movie into the window, bypassing offscreen buffers | render the video frame into the final framebuffer **after** the dirty-rect blits, clipped to the CRT hole |
| `MoviesTask(theMovie, 0)` | `Play.c:467`, `:492` | give QuickTime CPU time; it decodes and draws as needed | decode-and-present one video frame, called once per game frame at the same point in the loop |
| `StartMovie` / `StopMovie` / `GoToBeginningOfMovie` | `Trip.c:47-53`, `Transit.c` | transport control | play / pause / seek-to-zero |
| `SetMovieActive(theMovie, bool)` | `Play.c:204`, `:228` | attach/detach the movie from the idle-time task chain | enable/disable the decoder |
| `LoadMovieIntoRam`, `PrerollMovie`, `SetMovieMasterTimeBase` | `HouseIO.c:113-134` | preload and slave the movie clock | preload the file; drive presentation from the game clock, not a wall clock, if determinism matters |

The single most important porting decision here: `MoviesTask` is called **before** `RenderFrame`
(`Play.c:466-469`, `:491-494`) and the movie draws directly to the window, while `RenderFrame`'s
`CopyRectsQD` also draws to the window afterwards. Because `HandleTV` suppresses the CRT rect for the
movie TV, `CopyRectsQD` normally has nothing queued that overlaps the movie — but *other* renderers
can queue overlapping rects (a glider flying past the TV queues its own rect, which comes from
`workSrcMap` and therefore contains the stale static CRT pixels). On a real Mac this produces a
visible one-frame flicker over the TV whenever the glider overlaps it. Reproduce or fix, but know
that it is there.

---

## 9. Question (f) — budget exhaustion mid-draw

### 9.1 Every budget the object pass can exhaust

| Array | Cap constant | Value | Enforced in | Return on full | Draw suppressed? | Downstream index consumer |
|---|---|---|---|---|---|---|
| `dinahs[]` | `kMaxDynamicObs` | **18** | `AddDynamicObject` `Dynamics3.c:193` | −1 | **no** | `Toggle*`/`Trigger*` — **unchecked** |
| `hotSpots[]` | `kMaxHotSpots` | **56** | `AddActiveRect` `ObjectRects.c:280` | −1 | **no** (drawing is separate) | `TriggerSwitch` → `hotSpots[who]` — **unchecked** |
| `savedMaps[]` | `kMaxSavedMaps` | **24** | `BackUpToSavedMap` `DynamicMaps.c:75` | −1 | **YES** | `RestoreFromSavedMap`, `Render*` |
| `grease[]` | `kMaxGrease` | 16 | `AddGrease` `Grease.c:212` | −1 | **YES** (`if (dynamicNum != -1) DrawGrease*`) | `SpillGrease` — **unchecked** |
| `flames[]` | `kMaxCandles` | 20 | `AddCandleFlame` `DynamicMaps.c:321` | `void` | no | — |
| `tikiFlames[]` | `kMaxTikis` | 8 | `AddTikiFlame` `DynamicMaps.c:405` | `void` | no | — |
| `bbqCoals[]` | `kMaxCoals` | 8 | `AddBBQCoals` `DynamicMaps.c:491` | `void` | no | — |
| `pendulums[]` | `kMaxPendulums` | 8 | `AddPendulum` `DynamicMaps.c:575` | `void` | no | — |
| `theStars[]` | `kMaxStars` | 4 | `AddStar` `DynamicMaps.c:667` | `void` | no | — |
| `tempManholes[]` | `kMaxTempManholes` | 8 | `AddTempManholeRect` `Objects.c` | `void` | no (only the floor cut-out is lost) | `DrawFloorSupport` |
| `masterObjects[]` | `kMaxMasterObjects` | 216 | `ListOneRoomsObjects` `Objects.c:266` | loop stops | n/a | everything |
| `work2MainRects[]`, `back2WorkRects[]` | `kMaxGarbageRects` | 48 (effective 47) | `AddRectToWorkRects` `Render.c:67`, `AddRectToBackRects` `Render.c:86` | silently dropped | n/a (per-frame) | `CopyRectsQD` |
| `triggers[]` | `kMaxTriggers` | 16 | `FindEmptyTriggerSlot` `Triggers.c:59` | −1, trigger not armed | n/a | `HandleTriggers` |

Three distinct failure disciplines are in play, and the difference is the whole answer to (f):

* **Silent drop, art still drawn** — `dinahs[]`, `hotSpots[]`, and all the per-element caps. The object
  appears but is inert. This is the majority.
* **Silent drop, art suppressed** — `savedMaps[]` and `grease[]`. The object becomes *invisible* but
  remains logically present.
* **Silent truncation of the list** — `masterObjects[]`.

### 9.2 `kMaxDynamicObs` = 18 exhaustion, step by step

```
1. DrawARoomsObjects reaches, say, case kTV at slot i of some room.
2. SectRect passes; DrawTV paints the TV body + CRT into backSrcMap.        <-- art IS drawn
3. !redraw, so rectA is computed and AddDynamicObject(kTV, ...) is called.
4. AddDynamicObject sees numDynamics == 18 and returns -1 at Dynamics3.c:194.
   Nothing is written to dinahs[]; numDynamics stays 18.
5. dynamicNum == -1.
6. (kTV only) tvWithMovieNumber = -1 and tvInRoom = true, if the QT guard passes.   <-- §8.7
7. The write-back at :953-961 stores masterObjects[n].dynaNum = -1.
8. HandleDynamics never iterates past index 17, so the object never animates:
   - a TV/Mac/coffee/VCR/stereo/microwave never changes its indicator,
   - an outlet never zaps,
   - a toaster never launches toast,
   - a balloon/copter/dart/ball/drip/fish never moves (it sits as static art),
   - a sparkle never sparkles (and, having no art at all, is simply absent).
9. Any switch or trigger linked to it calls Toggle*(-1) or Trigger*(-1).       <-- §9.10
```

The object's **hot spot still exists** if it is in the central room (`CreateActiveRects` is called from
`ListAllLocalObjects`, which runs *before* any drawing and has its own independent 56-slot budget), so
the glider still collides with it, still gets shredded by an over-budget shredder, still bounces off an
over-budget ball's rect. Only the animation and the toggle path are lost.

### 9.3 `kMaxHotSpots` = 56 exhaustion

`hotSpots[]` is filled by `CreateActiveRects`, called from `ListOneRoomsObjects`
(`GliderPRO/Sources/Objects.c:283-286`) **only for the central room**:

```c
	if ((where == kCentralRoom) && (IsThisValid(roomNum, n)))
		masterObjects[numMasterObjects].hotNum = CreateActiveRects(n);
	else
		masterObjects[numMasterObjects].hotNum = -1;
```

`AddActiveRect` (`GliderPRO/Sources/ObjectRects.c:277-292`):

```c
short AddActiveRect (Rect *theRect, short action, short who)
{
	if (nHotSpots >= kMaxHotSpots)          /* :280   56 */
		return (-1);                         /* :281 */
	hotSpots[nHotSpots].bounds = *theRect;
	hotSpots[nHotSpots].action = action;
	hotSpots[nHotSpots].who = who;
	hotSpots[nHotSpots].isOn = ...;
	hotSpots[nHotSpots].stillOver = false;
	hotSpots[nHotSpots].doScrutinize = ...;
	nHotSpots++;                             /* :290 */
	return (nHotSpots - 1);                  /* :292 */
}
```

`hotObject` (`GliderPRO/Headers/GliderStructs.h:218-225`):

```c
typedef struct
{
	Rect     bounds;
	short    action;                 /* kDissolveIt, kPushItLeft, kBurnIt, kLiftIt, ... */
	short    who;                    /* masterObjects[] index                          */
	Boolean  isOn, stillOver;
	Boolean  doScrutinize;
} hotObject, *hotPtr;
```

**One object can consume more than one hot spot.** From `CreateActiveRects`
(`ObjectRects.c:296-1060`):

| Type | Hot spots created | Actions |
|---|---|---|
| `kLeftFan` 0x06, `kRightFan` 0x07 | **2** | `kDissolveIt` + `kPushItLeft` / `kPushItRight` |
| `kTaper` 0x08, `kCandle` 0x09, `kStubby` 0x0A, `kTiki` 0x0B, `kBBQ` 0x0C | **2 or 3** | `kLiftIt` (only if the column height exceeds `kDeadlyFlameHeight` = 24, `ObjectRects.c:17`), then `kBurnIt`, then `kDissolveIt` |
| `kMicrowave` 0x6A | **2** | `kDissolveIt` + `kMicrowaveIt` |
| all 8 light types, `kCustomPict` 0x6E, `kSparkle` 0x2D | **0** | — |
| 14 of the 15 clutter types | **0** | the exception is `kChimes` 0x8F, which creates one **and** increments `numChimes` |
| mailboxes 0x33/0x34, ducts, `kInvisTrans` 0x3F, `kDeluxeTrans` 0x40 | 1, **only if** `data.d.who != 255` | transport |
| the 6 switch types 0x41-0x46 | 1, **only if** `data.e.where != -1` | switch |
| `kTrigger` 0x47, `kLgTrigger` 0x48 | 1, **only if** `data.e.where != -1` | trigger |
| `kSoundTrigger` 0x49 | 1, **only if** `LoadTriggerSound() == noErr` | trigger |
| `kInvisBlower` 0x0D, `kLiftArea` 0x10 | 1, **only if** the `data.a.vector` nibble ∈ {1, 2, 4, 8} | push/lift |
| everything else | 1 | per-type |

`CreateActiveRects` returns only the **last** `hotSpotNumber` it assigned, so for a multi-rect object
`masterObjects[].hotNum` points at the last rect, not the first. Anything that walks an object's rects
must scan `hotSpots[]` for matching `who`, not index off `hotNum`.

On exhaustion:

* `AddActiveRect` returns −1 and `hotSpotNumber` stays at the previous value (or −1 if this was the
  first). For a multi-rect object this means a *partial* rect set: e.g. a fan whose `kDissolveIt`
  rect made it in but whose `kPushItLeft` rect did not — the glider dissolves on contact but is never
  pushed.
* The object is still **drawn** (drawing is `DrawARoomsObjects`' job and is entirely independent of
  hot spots).
* `masterObjects[].hotNum` becomes −1, and for the six switch types
  `ObjectDrawAll.c:518/531/544/557/570/574` copies that −1 into `dynaNum`, so
  `TriggerSwitch(-1)` → `HandleSwitches(&hotSpots[-1])`.

### 9.4 `kMaxSavedMaps` = 24 exhaustion — the only one that hides the object

The prize cases are all of this shape (`kRedClock`, `:286-298`):

```c
case kRedClock:
GetObjectRect(&thisObject, &itsRect);
OffsetRectRoomRelative(&itsRect, neighbor);
if (SectRect(&itsRect, &testRect, &whoCares))
{
	if (redraw)
		legit = ReBackUpSavedMap(&itsRect, localNumbers[neighbor], i);
	else
		legit = BackUpToSavedMap(&itsRect, localNumbers[neighbor], i);
	if (legit != -1)                       /* <-- the draw is GATED on the budget */
		DrawRedClock(&itsRect);
}
break;
```

So an over-budget clock is **invisible**. But:

* `IsThisValid` still returns true for it, so it is still scanned;
* `CreateActiveRects` still gave it a hot spot (independent 56-slot budget), so the glider still
  collects it and still scores its points;
* `RestoreFromSavedMap` (called from `Interactions.c:1049` when it is collected) will index
  `savedMaps[...]` for a slot that was never created for this object — `RestoreFromSavedMap`
  (`DynamicMaps.c:131+`) does its own `(where, who)` linear search and returns without drawing if it
  finds nothing, so this part is safe.

Net effect: an invisible-but-collectable prize. This is a real authoring hazard because
`kMaxSavedMaps` = 24 is consumed by *nine rooms' worth* of prizes, flames, tikis, coals, pendulums,
stars and grease, and the editor's pre-flight check (§9.7) does not count `savedMaps[]` at all.

Note also that `kCuckoo` and `kStar` each consume **two** `savedMaps[]` slots (a background swatch
plus an animation strip), so 12 stars saturate the array.

For `kStar` the gating is nested one deeper (`:429-444`): the `AddStar` call *and* the
`DrawSimplePrizes` call are both inside `if (legit != -1)`, so an over-budget star neither draws nor
animates. `kCuckoo` (`:331-347`) likewise gates both `DrawCuckoo` and `AddPendulum`.

### 9.5 The per-element caps

None of these suppress the draw; they only drop the animation:

| Array | Cap | Effect when full |
|---|---|---|
| `flames[]` | 20 | the candle body is drawn, no flame animates |
| `tikiFlames[]` | 8 | the tiki body is drawn (if `isLit`), no flame |
| `bbqCoals[]` | 8 | the BBQ is drawn, no coals |
| `pendulums[]` | 8 | the cuckoo clock is drawn, no pendulum swings; `clockFrame` is **not** re-set to 10 |
| `theStars[]` | 4 | — but see §9.4: `AddStar` is inside the `legit != -1` guard, and `AddStar` itself calls `BackUpToSavedMap`, so a full `theStars[]` leaves the star drawn but static |
| `tempManholes[]` | 8 | the manhole art is drawn but `DrawFloorSupport` paints the support band straight over it, so the hole is sealed |

`AddCandleFlame`/`AddTikiFlame`/`AddBBQCoals`/`AddPendulum`/`AddStar` return `void`, so
`DrawARoomsObjects` cannot even observe the failure.

**All five of them are, however, internally correct about the nested `savedMaps[]` failure.** Each
calls `BackUpToSavedMap` and tests the result before touching its own array:

| Function | own-cap test | `savedNum = BackUpToSavedMap(...)` | `if (savedNum != -1)` guard | `numXxx++` inside the guard? |
|---|---|---|---|---|
| `AddCandleFlame` | `:321` | `:333` | `:334` | yes, `:342` |
| `AddTikiFlame` | `:405` | `:417` | `:418` | yes, `:427` |
| `AddBBQCoals` | `:491` | `:503` | `:504` | yes, `:513` |
| `AddPendulum` | `:575` | `:580` | `:581` | yes, `:604` |
| `AddStar` | `:667` | `:680` | `:681` | yes, `:692` |

(All line numbers in `GliderPRO/Sources/DynamicMaps.c`.) So a `savedMaps[]` overflow that happens
*inside* one of these five leaves `numFlames`/`numTikiFlames`/`numCoals`/`numPendulums`/`numStars`
unchanged and the animator array untouched — no `savedMaps[-1]` read is possible from this path. The
`.who` field always holds a valid index. This is the **one** place in the object pass where the −1
sentinel is handled correctly everywhere; contrast §9.10.

Two consequences of the exact ordering that a port must copy:

1. **`AddPendulum` writes the global `clockFrame = 10` at `:578` — before `BackUpToSavedMap` at
   `:580`.** So a cuckoo dropped for lack of a *savedMaps* slot still phase-resets every already-
   registered pendulum, while one dropped for lack of a *pendulums* slot (early `return` at `:576`)
   does not. Two different failure modes with two different visible outcomes.
2. **The own-cap test comes first in all five**, so an over-budget flame/tiki/coal/pendulum/star does
   **not** consume a `savedMaps[]` slot. The two budgets interact in one direction only.
3. **`kCuckoo` and `kStar` each consume *two* `savedMaps[]` slots per object**, not one. The case body
   takes one for the object swatch itself (`ObjectDrawAll.c:336` for `kCuckoo`, `:434` for `kStar`) and
   then the registrar it calls inside the `legit != -1` guard takes a second one for the animation
   strip (`AddPendulum` → `DynamicMaps.c:580`; `AddStar` → `:680`). Eight cuckoo clocks therefore cost
   16 of the 24 slots. By contrast `kTaper`/`kCandle`/`kStubby`, `kTiki`, `kBBQ` and the grease jars
   cost exactly **one** each, because the only `BackUpToSavedMap` call on those paths is the one inside
   `AddCandleFlame` / `AddTikiFlame` / `AddBBQCoals` / `AddGrease` (`Grease.c:219`) — the case bodies
   themselves do not back anything up.

`AddCandleFlame`'s extra guards `h < 16` and `v < 15` (`:321`) are **position** tests, not size tests:
`h`/`v` are the flame's anchor in room-relative screen space (`itsRect.left + 10`, `itsRect.top + 7`
from `ObjectDrawAll.c:83-84`) and `src` is built as `(0,0,16,15)` offset by `(h − 8, v − 15)`, so the
guard exists purely to keep `src` from going negative. Same for `AddTikiFlame` (`h < 8 || v < 10`),
`AddBBQCoals` (`h < 32 || v < 9`) and `AddPendulum` (`h < 32 || v < 28`); `AddStar` has no such test.

**Mac 4-bit-depth quirk, present in all five.** Each nudges the strip to an even horizontal pixel when
the screen is 4 bits deep, because a 4-bit `PixMap` packs two pixels per byte and QuickDraw's
`CopyBits` fast path needed byte alignment:

```c
/* AddCandleFlame, DynamicMaps.c:326-331 — tests src.left AFTER offsetting */
if ((thisMac.isDepth == 4) && ((src.left % 2) == 1))
{ QOffsetRect(&src, -1, 0); if (src.left < 0) QOffsetRect(&src, 2, 0); }

/* AddTikiFlame :409, AddBBQCoals :495, AddPendulum :584, AddStar :671 — test h BEFORE offsetting */
if ((thisMac.isDepth == 4) && ((h % 2) == 1))
{ h--; if (h < 0) h += 2; }
```

The two spellings are equivalent for candles because `src.left == h − 8` and 8 is even. A Go port
rendering into a 32-bit RGBA buffer must decide whether to keep the nudge: keeping it means flames sit
one pixel left of where they do at 8-bit depth (a 1-pixel visual difference on 4-bit screens only);
dropping it is the simpler and almost certainly correct choice, but it is a *behavioural* divergence, so
record it. `thisMac.isDepth` is set at `Environ.c:542` from `WhatsOurDepth()`
(`Environ.c:234-252`), which reads `(**(**thisGDevice).gdPMap).pixelSize` at `:243` and falls back to
`1` at `:250` on a non-colour machine; it is also re-set at `Environ.c:394` after a depth switch.

### 9.6 `masterObjects[]` saturates at exactly 216

`kMaxMasterObjects` = 216 = 9 × 24 exactly (`GliderDefines.h:266`). `ListOneRoomsObjects`
(`Objects.c:266`) loops `while (numMasterObjects < kMaxMasterObjects)` — so with all nine local rooms
present the array is **exactly full**, with zero slack. Verified against the whole corpus: every one of
the 22 shipped houses hits `numMasterObjects == 216` for every neighbourhood in which all nine local
rooms exist (§9.9.3), and 24 × (number of existing local rooms) otherwise — 48 for `Sampler`, 72 for
`California or Bust!`, whose floors are too sparse to ever fill a 3×3 block.

That means:

* the array size is not a "budget" that can be exceeded by content — it is dimensioned to the maximum
  possible;
* but it also means any port that adds a tenth pseudo-room, or that keeps stale entries, immediately
  truncates.

### 9.7 The editor's only pre-flight guard: `HowManyDynamicObjects`

`GliderPRO/Sources/ObjectAdd.c:1045-1071`:

```c
short HowManyDynamicObjects (void)
{
	aDinah = 0;
	for (i = 0; i < kMaxRoomObs; i++)
		if ((thisRoom->objects[i].what == kSparkle) ||
				(thisRoom->objects[i].what == kToaster) ||
				(thisRoom->objects[i].what == kMacPlus) ||
				(thisRoom->objects[i].what == kTV) ||
				(thisRoom->objects[i].what == kCoffee) ||
				(thisRoom->objects[i].what == kOutlet) ||
				(thisRoom->objects[i].what == kVCR) ||
				(thisRoom->objects[i].what == kStereo) ||
				(thisRoom->objects[i].what == kMicrowave) ||
				(thisRoom->objects[i].what == kBalloon) ||
				(thisRoom->objects[i].what == kCopterLf) ||
				(thisRoom->objects[i].what == kCopterRt) ||
				(thisRoom->objects[i].what == kDartLf) ||
				(thisRoom->objects[i].what == kDartRt) ||
				(thisRoom->objects[i].what == kBall) ||
				(thisRoom->objects[i].what == kDrip) ||
				(thisRoom->objects[i].what == kFish))
			aDinah++;
	return (aDinah);
}
```

Exactly the 17 types of §7.2 — the list is consistent, which is worth noting because it is the only
place in the codebase the set is written out. Call sites:

| Site | Condition | Applies to |
|---|---|---|
| `ObjectAdd.c:230` | `(what == kSparkle) && (HowManyDynamicObjects() >= kMaxDynamicObs)` | adding a sparkle |
| `ObjectAdd.c:652` | `(what != kCobweb) && (HowManyDynamicObjects() >= kMaxDynamicObs)` | adding an enemy (all but the cobweb) |
| `ObjectAdd.c:698` | `HowManyDynamicObjects() >= kMaxDynamicObs` | adding an appliance |

All three then call `ShoutNoMoreSpecialObjects()` (`ObjectAdd.c:1075-1081`, an `Alert`) and
`return (false)`.

Sibling per-type guards in the same function. This is the **complete** set — twelve `HowMany*`
counters declared at `ObjectAdd.c:27-38`, each with exactly the call site(s) listed:

| Guard call site | Condition | Counter body | Cap macro | Value | Where the value is defined |
|---|---|---|---|---|---|
| `ObjectAdd.c:80-85` | `(kTaper \| kCandle \| kStubby) && HowManyCandleObjects() >= kMaxCandles` | `:888-900` | `kMaxCandles` | 20 | `GliderDefines.h:255` |
| `ObjectAdd.c:86-90` | `kTiki && HowManyTikiObjects() >= kMaxTikis` | `:904-914` | `kMaxTikis` | 8 | `GliderDefines.h:256` |
| `ObjectAdd.c:91-95` | `kBBQ && HowManyBBQObjects() >= kMaxCoals` | `:918-928` | `kMaxCoals` | 8 | `GliderDefines.h:257` |
| `ObjectAdd.c:215-219` | `kCuckoo && HowManyCuckooObjects() >= kMaxPendulums` | `:932-942` | `kMaxPendulums` | 8 | `GliderDefines.h:258` |
| `ObjectAdd.c:220-224` | `kBands && HowManyBandsObjects() >= kMaxRubberBands` | `:946-956` | `kMaxRubberBands` | 2 | `GliderDefines.h:261` |
| `ObjectAdd.c:225-229` | `kStar && HowManyStarsObjects() >= kMaxStars` | `:975-985` | `kMaxStars` | 4 | `GliderDefines.h:263` |
| `ObjectAdd.c:230-234` | `kSparkle && HowManyDynamicObjects() >= kMaxDynamicObs` | `:1045-1071` | `kMaxDynamicObs` | 18 | `GliderDefines.h:265` |
| `ObjectAdd.c:251-255` | `HowManyGreaseObjects() >= kMaxGrease` (grease arm, unconditional) | `:960-971` | `kMaxGrease` | 16 | `GliderDefines.h:262` |
| `ObjectAdd.c:299-303` | `kUpStairs && HowManyUpStairsObjects() >= kMaxStairs` | `:1003-1013` | `kMaxStairs` | **1** | `ObjectAdd.c:18` (file-local `#define`) |
| `ObjectAdd.c:304-308` | `kDownStairs && HowManyDownStairsObjects() >= kMaxStairs` | `:1017-1027` | `kMaxStairs` | **1** | `ObjectAdd.c:18` |
| `ObjectAdd.c:484-488` | `kSoundTrigger && HowManySoundObjects() >= kMaxSoundTriggers` | `:989-999` | `kMaxSoundTriggers` | **1** | `ObjectAdd.c:17` (file-local) |
| `ObjectAdd.c:605-609` | `kShredder && HowManyShredderObjects() >= kMaxShredded` | `:1031-1041` | `kMaxShredded` | 4 | `GliderDefines.h:264` |
| `ObjectAdd.c:652-656` | `(what != kCobweb) && HowManyDynamicObjects() >= kMaxDynamicObs` | `:1045-1071` | `kMaxDynamicObs` | 18 | `GliderDefines.h:265` |
| `ObjectAdd.c:698-702` | `HowManyDynamicObjects() >= kMaxDynamicObs` (kBall/kDrip/kFish) | `:1045-1071` | `kMaxDynamicObs` | 18 | `GliderDefines.h:265` |

Two of these caps are **1**, and both are declared file-locally in `ObjectAdd.c` rather than in
`GliderDefines.h` (`kMaxSoundTriggers` at `:17`, `kMaxStairs` at `:18`): a room may hold at most one
`kUpStairs`, one `kDownStairs` and one `kSoundTrigger`. A Go editor must reproduce those or it will
author houses the engine mishandles (`Objects.c`'s stair search takes the first match only).

**The appliances have no `kMaxDynamicObs` guard at all, and the guard they do have is wrong.**
`ObjectAdd.c:589-609` verbatim:

```c
case kShredder:  case kToaster:  case kMacPlus:  case kGuitar:      /* :589-592 */
case kTV:        case kCoffee:   case kOutlet:   case kVCR:         /* :593-596 */
case kStereo:    case kMicrowave: case kCinderBlock: case kFlowerBox: /* :597-600 */
case kCDs:       case kCustomPict:                                  /* :601-602 */
	if ((what != kGuitar) && (what != kCinderBlock) && (what != kFlowerBox) &&
			(what != kCDs) && (what != kCustomPict) &&
			(HowManyShredderObjects() >= kMaxShredded))                /* :603-605 */
	{
		ShoutNoMoreSpecialObjects();                                  /* :607 */
		return (false);                                               /* :608 */
	}
```

`HowManyShredderObjects` (`:1031-1041`) counts **only** `what == kShredder`. So:

* adding a `kTV` to a room that already contains four `kShredder`s is **refused** (the shredder count
  gates eight unrelated appliance types);
* adding twenty-four `kTV`s to a shredder-free room is **allowed**, and every one of them will try to
  claim a `dinahs[]` slot at run time — 24 > `kMaxDynamicObs` = 18 from a single room, before the eight
  neighbours contribute anything.

`HowManyDynamicObjects` is therefore consulted for exactly three of the seventeen type families it
counts: `kSparkle` (`:230`), the five enemies excluding `kCobweb` (`:652`), and `kBall`/`kDrip`/`kFish`
(`:698`). The eight appliance types it lists are never checked against it.

### 9.8 Five things the editor does **not** check

**This is the single most important structural fact about all twelve editor guards: every
`HowMany*Objects` function loops `for (i = 0; i < kMaxRoomObs; i++)` over `thisRoom->objects[i]` — one
room — while every runtime budget is consumed by the whole 3×3 neighbourhood.** The editor's
invariant is therefore a factor-of-nine weaker than the engine's requirement, and §9.9 shows that
shipped houses really do exceed the engine's caps as a result.

1. **The caps are per-room, the budgets are per-neighbourhood.** `HowManyCandleObjects` allows 20
   candles *per room*; `flames` is dimensioned `sizeof(flameType) * kMaxCandles` = 20 entries *per
   neighbourhood* (`StructuresInit2.c:213`). Nine legal rooms can therefore demand 180 flame slots.
   The same 9× gap applies to `kMaxTikis`, `kMaxCoals`, `kMaxPendulums`, `kMaxStars`, `kMaxGrease` and
   `kMaxDynamicObs`.
2. **`kMaxDynamicObs` is not checked for appliances at all** (§9.7), so even the per-room invariant
   does not hold for the eight appliance types.
3. **It does not count `savedMaps[]`.** Nothing anywhere checks the 24-slot limit. Prizes, flames,
   tikis, coals, pendulums, stars and grease all draw from it, `kCuckoo` and `kStar` take two each,
   and exceeding it makes prizes silently invisible. Because `savedMaps[]` is the *union* of six
   per-type budgets that are individually checked, it is the cap that shipped content breaks most
   often (§9.9).
4. **It does not count `hotSpots[]`.** Nothing checks the 56-slot limit, and the multi-rect types
   (fans 2, candle family 2-3, microwave 2) make the mapping from object count to hot-spot count
   non-obvious. This one is *not* aggravated by the 9× gap, because `ListAllLocalObjects` fills
   `hotSpots[]` from the central room only (§9.3).
5. **It ignores resolution.** As §7.9 shows, the actual consumption depends on the player's screen
   size, because most registrations are gated on `SectRect(&itsRect, &testRect, &whoCares)`. A house
   authored and tested at 640 × 480 can overflow at 1024 × 768 — and §9.9 measures exactly that
   happening.

### 9.9 Measured budget usage across the whole shipped house corpus

#### 9.9.1 The corpus

`GliderPRO/Houses/` contains **22** houses (`*.binhex`, BinHex 4.0 wrappers around the classic-Mac
data/resource fork pair) and **15** QuickTime movies (`*.mov`) for the `kTV` case of §8:

| # | House | Movie shipped? |
|---|---|---|
| 1 | `Art Museum` | yes |
| 2 | `California or Bust!` | no |
| 3 | `Castle o' the Air` | yes |
| 4 | `CD Demo House` | yes |
| 5 | `Davis Station` | yes |
| 6 | `Demo House` | yes |
| 7 | `Empty House` | no |
| 8 | `Fun House` | no |
| 9 | `Grand Prix` | yes |
| 10 | `ImagineHouse PRO II` | yes |
| 11 | `In The Mirror` | no |
| 12 | `Land of Illusion` | yes |
| 13 | `Leviathan` | yes |
| 14 | `Metropolis` | no |
| 15 | `Nemo's Market` | yes |
| 16 | `Rainbow's End` | yes |
| 17 | `Sampler` | no |
| 18 | `Slumberland` | yes |
| 19 | `SpacePods` | yes |
| 20 | `Teddy World` | yes |
| 21 | `The Asylum Pro` | no |
| 22 | `Titanic` | yes |

#### 9.9.2 Method

A replay harness (written for this document; reuses `tools/probe_house.py` for the BinHex and house
parsing) walks, for every non-empty room of every house, the budget-consuming parts of
`DrawLocale` → `DrawARoomsObjects`. The tables in §9.9.3-§9.9.5 below were regenerated during this
document's verification pass by the corrected model described in the "known limitations" list; the
first draft's model had the vertical neighbour sign wrong and did not implement the candle proximity
test, and its numbers have been replaced. The steps are:

1. build the nine `localNumbers[]` by looking up `(floor + Δv, suite + Δh)` for each of the nine
   `kXxxRoom` slots (Δ table in §4.2);
2. visit them in the **draw order** `NW, NE, N, SW, SE, S, W, E, Central` of §4, honouring the
   `numNeighbors > 3` / `numNeighbors > 1` prefix gates;
3. skip objects rejected by `IsThisValid` (§3.4);
4. model `SectRect(&itsRect, &testRect, &whoCares)` exactly as the C does — compute
   `itsRect` as `srcRects[what]` (§2.7) placed at the object's `topLeft`, offset by
   `(playOriginH, playOriginV)` and then by `(Δh × kRoomWide, Δv × kVertLocalOffset)` =
   `(Δh × 512, Δv × 322)`, and intersect against `testRect` = `houseRect` zeroed =
   `(0, 0, min(screenW, 1536), min(screenH − 20, 1026))`;
5. reproduce the per-case registration rules of §7.2 — the seven `DYN_ANY_ROOM` appliance types
   register from **any** room, the ten glider-interacting types only from Central, and the six enemy
   types register with **no** `SectRect` gate at all;
6. count `hotSpots[]` from the central room only (§9.3) and `masterObjects[]` as
   24 × (number of listed non-empty rooms), with `ListAllLocalObjects`'s own `numNeighbors` gates
   (`Objects.c:314, 320`).

**Known limitations of the model**, so the numbers are read correctly:

* `kTiki` genuinely has **no** `SectRect` guard (§10, group 6), and the model reproduces that; but the
  model assumes visibility (`return True`) for any `what` whose `srcRects[]` size is not in its table,
  which slightly over-counts for the furniture/clutter types that use `data.b.bounds` rather than
  `topLeft`. None of those consume the budgets measured here, so the effect is nil for these columns.
* `AddCandleFlame`'s secondary size gates (`h < 16 || v < 15`, `DynamicMaps.c:321`), and the analogous
  `h < 8 || v < 10` for tikis (`:405`), `h < 32 || v < 9` for coals (`:491`) and `h < 32 || v < 28` for
  pendulums (`:575`), are **not** modelled. Those only ever *reduce* consumption, so the flame/tiki/
  coal/pendulum columns are upper bounds.
* Consumption is counted as *demand*, not as *what the C actually stored*. Where the demand exceeds
  the cap, the C silently drops the excess — that is the point of the exercise.
* The vertical neighbour offset must use the **screen** delta, not the floor delta. `localNumbers[]` is
  built with North at `vDelta = +1` (`Room.c:576-579`, one floor *up*), but `OffsetRectRoomRelative`
  shifts North by `-kVertLocalOffset` (`ObjectRects.c:1099-1101`). Using `+1 × 322` as the pixel offset
  mirrors all six N/S rooms and changes which of them `SectRect` accepts. (An earlier revision of the
  tables in §9.9.3-§9.9.5 had this sign wrong; they have been regenerated.)
* The candle-family **proximity** test is modelled: a non-central `kTaper`/`kCandle`/`kStubby`
  registers a flame only if its 16 × 15 flame rect does **not** intersect
  `localRoomsDest[kCentralRoom]` fattened by ±`kFloorSupportTall` (44) — `ObjectDrawAll.c:86-101`,
  `:120-135`, `:154-169`. Omitting it over-counts `flames[]` and `savedMaps[]`.
* `kManhole`'s rect comes from `data.b.bounds` (`ObjectRects.c:88`), not from `srcRects[]` at
  `topLeft`, and its `AddTempManholeRect` call sits **inside** the `SectRect` block but is *not* gated
  on `redraw` (`ObjectDrawAll.c:275-277`).
* The `hotSpots[]` column is produced by a hand-written per-type hot-spot count, **not** re-derived
  from `CreateActiveRects` (unverified: the heuristic in `budget_sweep.py`/`sweep2.py` was not checked
  against `Objects.c`'s real `AddActiveRect` call pattern, so treat that one column as indicative
  only). Every other column is derived from the C control flow directly.

#### 9.9.3 Peak demand at 1024 × 768, `numNeighbors = 9`

`playOriginH` = (1024 − 512) / 2 = 256, `playOriginV` = (768 − 322) / 2 = 223,
`testRect` = (0, 0, 1024, 748). Bold = **over cap**.

| House | `dinahs`/18 | `savedMaps`/24 | `hotSpots`/56 | `masterObjects`/216 | `flames`/20 | `tikiFlames`/8 | `bbqCoals`/8 | `pendulums`/8 | `theStars`/4 | `grease`/16 | `tempManholes`/8 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Art Museum | 7 | 10 | 27 | 216 | 7 | 0 | 0 | 0 | 1 | 2 | 0 |
| California or Bust! | 17 | 13 | 40 | 72 | 10 | 0 | 3 | 0 | 0 | 0 | 0 |
| Castle o' the Air | 16 | 14 | 30 | 216 | 3 | 2 | 6 | 2 | 2 | 2 | 0 |
| CD Demo House | 9 | 9 | 21 | 216 | 4 | 1 | 1 | 1 | 1 | 2 | 0 |
| Davis Station | 6 | 5 | 18 | 216 | 2 | 1 | 1 | 0 | 1 | 0 | 2 |
| Demo House | 5 | 3 | 12 | 216 | 2 | 0 | 0 | 0 | 1 | 1 | 2 |
| Empty House | 0 | 0 | 7 | 216 | 0 | 0 | 0 | 0 | 0 | 0 | 2 |
| Fun House | 7 | 7 | 22 | 216 | 2 | 0 | 0 | 1 | 0 | 2 | 0 |
| Grand Prix | 10 | 10 | 23 | 216 | 7 | 0 | 0 | 0 | 1 | 1 | 2 |
| ImagineHouse PRO II | 12 | 8 | 24 | 216 | 4 | 6 | 0 | 1 | 0 | 1 | 2 |
| In The Mirror | 9 | **32** | 22 | 216 | 2 | 5 | 0 | 4 | 1 | 1 | 1 |
| Land of Illusion | 12 | 20 | 26 | 216 | 8 | 5 | 0 | 5 | 1 | 2 | 2 |
| Leviathan | 16 | 19 | 29 | 216 | 9 | 1 | 1 | 1 | 1 | 5 | 3 |
| Metropolis | 10 | 15 | 23 | 216 | 4 | 0 | 0 | 1 | 1 | 3 | 0 |
| Nemo's Market | 12 | 11 | 24 | 216 | 1 | 0 | 0 | 1 | 1 | 0 | 0 |
| Rainbow's End | 10 | 23 | 28 | 216 | 8 | 3 | 4 | 2 | 1 | 3 | 2 |
| Sampler | 0 | 1 | 5 | 48 | 0 | 0 | 1 | 0 | 0 | 0 | 0 |
| Slumberland | 8 | **25** | 52 | 216 | 16 | 1 | 2 | 2 | 1 | 7 | 1 |
| SpacePods | 16 | 18 | 23 | 216 | 4 | 0 | 0 | **9** | 1 | 0 | 0 |
| Teddy World | 18 | **27** | 40 | 216 | 14 | 6 | 4 | 1 | 1 | **21** | 1 |
| The Asylum Pro | 6 | 21 | 24 | 216 | 6 | 7 | 1 | 1 | 1 | 3 | 1 |
| Titanic | 11 | **25** | 39 | 216 | 2 | 7 | 5 | 6 | 1 | 2 | 2 |
| **corpus max** | 18 | **32** | 52 | 216 | 16 | 7 | 6 | **9** | 2 | **21** | 3 |

The rooms that are at or over a cap at this resolution, with the room index as stored in the house
file and the room's own name string:

| House | Room | Name | Result |
|---|---|---|---|
| In The Mirror | 76 | `No Mirror Here` | `savedMaps` 32 / 24 — **8 dropped** |
| In The Mirror | 85 | `Too Many Mirrors!` | `savedMaps` 27 / 24 — 3 dropped |
| Teddy World | 321 | `Where is the grease from…` | `savedMaps` 27 / 24 and `grease` 19 / 16 |
| Teddy World | 322 | `That's a lot of grease!` | `savedMaps` 27 / 24 and `grease` 21 / 16 — **5 grease dropped** |
| Slumberland | 177 | `A Pair Of Drawers` | `savedMaps` 25 / 24 — 1 dropped |
| Titanic | 5 | `Cabin 2A` | `savedMaps` 25 / 24 — 1 dropped |
| Slumberland | 256 | `Flaming Pathway` | `savedMaps` 24 / 24 — exactly full |
| Teddy World | 244 | `Moving walls` | `dinahs` 18 / 18 — exactly full |
| Teddy World | 379 | `Get out of here!` | `dinahs` 18 / 18 — exactly full |
| Teddy World | 250 | `Does your head hurt now?` | `grease` 16 / 16 — exactly full |
| SpacePods | 40 | `Pod Side` | `pendulums` **9 / 8** — 1 dropped |
| SpacePods | 41 | `Pod Side` | `pendulums` **9 / 8** — 1 dropped |
| SpacePods | 48 | `Pod Standard Time` | `pendulums` **9 / 8** — 1 dropped |

That is **13 rooms** at or over a cap at 1024 × 768, of which **9** are strictly over.

#### 9.9.4 Peak demand at the maximum view, 1536 × 1046, `numNeighbors = 9`

At this size all nine 512 × 322 room viewports are fully on screen (`playOrigin` = (512, 362),
`testRect` = (0, 0, 1536, 1026), so the three column bands are `[0,512)`, `[512,1024)`, `[1024,1536)`
and the three row bands are `[40,362)`, `[362,684)`, `[684,1006)`), so `SectRect` rejects nothing and
this table is the **absolute upper bound** on demand for each house:

| House | `dinahs`/18 | `savedMaps`/24 | `hotSpots`/56 | `masterObjects`/216 | `flames`/20 | `tikiFlames`/8 | `bbqCoals`/8 | `pendulums`/8 | `theStars`/4 | `grease`/16 | `tempManholes`/8 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Art Museum | 7 | 13 | 27 | 216 | 8 | 0 | 0 | 0 | 1 | 2 | 0 |
| California or Bust! | 18 | 14 | 40 | 72 | 10 | 0 | 3 | 0 | 0 | 0 | 0 |
| Castle o' the Air | 16 | 19 | 30 | 216 | 4 | 2 | 6 | 2 | 2 | 2 | 0 |
| CD Demo House | 9 | 11 | 21 | 216 | 4 | 1 | 1 | 1 | 1 | 3 | 0 |
| Davis Station | 6 | 5 | 18 | 216 | 2 | 1 | 1 | 0 | 1 | 0 | 3 |
| Demo House | 5 | 4 | 12 | 216 | 3 | 0 | 0 | 0 | 1 | 1 | 2 |
| Empty House | 0 | 0 | 7 | 216 | 0 | 0 | 0 | 0 | 0 | 0 | 2 |
| Fun House | 8 | 11 | 22 | 216 | 2 | 0 | 0 | 1 | 0 | 3 | 0 |
| Grand Prix | 10 | 12 | 23 | 216 | 8 | 0 | 0 | 0 | 1 | 1 | 2 |
| ImagineHouse PRO II | 14 | 9 | 24 | 216 | 4 | 6 | 0 | 1 | 0 | 1 | 2 |
| In The Mirror | 9 | **40** | 22 | 216 | 3 | 5 | 0 | 4 | 1 | 1 | 2 |
| Land of Illusion | 14 | **30** | 26 | 216 | 8 | 5 | 0 | 5 | 1 | 2 | 2 |
| Leviathan | 16 | **27** | 29 | 216 | 11 | 1 | 1 | 2 | 1 | 8 | 4 |
| Metropolis | 11 | 17 | 23 | 216 | 4 | 0 | 0 | 1 | 2 | 4 | 0 |
| Nemo's Market | 16 | 15 | 24 | 216 | 1 | 0 | 0 | 1 | 1 | 0 | 0 |
| Rainbow's End | 12 | **32** | 28 | 216 | 10 | 3 | 5 | 2 | 1 | 3 | 3 |
| Sampler | 0 | 1 | 5 | 48 | 0 | 0 | 1 | 0 | 0 | 0 | 0 |
| Slumberland | 10 | **47** | 52 | 216 | **29** | 1 | 2 | 2 | 1 | 8 | 2 |
| SpacePods | 17 | 18 | 23 | 216 | 4 | 0 | 0 | **9** | 1 | 0 | 0 |
| Teddy World | 18 | **35** | 40 | 216 | 15 | 6 | 4 | 1 | 1 | **25** | 1 |
| The Asylum Pro | 7 | 23 | 24 | 216 | 7 | 7 | 1 | 1 | 1 | 4 | 1 |
| Titanic | 11 | **33** | 39 | 216 | 3 | 7 | 5 | 8 | 1 | 2 | 2 |
| **corpus max** | 18 | **47** | 52 | 216 | **29** | 7 | 6 | **9** | 2 | **25** | 4 |

At this resolution **57 rooms** across **nine** of the 22 houses are at or over a cap: **46** rooms are
strictly over at least one cap, 42 of them on `savedMaps[]`. The nine houses are
`California or Bust!`, `In The Mirror`, `Land of Illusion`, `Leviathan`, `Rainbow's End`,
`Slumberland`, `SpacePods`, `Teddy World`, `Titanic`. The worst cases:

| House | Room | Name | Result |
|---|---|---|---|
| Slumberland | 257 | `Speed Slick` | `savedMaps` **47 / 24** (23 dropped), `flames` **29 / 20** (9 dropped) |
| In The Mirror | 76 | `No Mirror Here` | `savedMaps` **40 / 24** (16 dropped) |
| Teddy World | 321 | `Where is the grease from…` | `savedMaps` **34 / 24**, `grease` **25 / 16** |
| Slumberland | 254 | `Mazeway 2000` | `savedMaps` **34 / 24**, `flames` 21 / 20 |
| Slumberland | 255 | `Pedestal Problems` | `savedMaps` **34 / 24**, `flames` 21 / 20 |
| Teddy World | 322 | `That's a lot of grease!` | `savedMaps` **34 / 24**, `grease` 22 / 16 |
| In The Mirror | 85 | `Too Many Mirrors!` | `savedMaps` **33 / 24** |
| Titanic | 5 | `Cabin 2A` | `savedMaps` **33 / 24**, `pendulums` 8 / 8 |
| Rainbow's End | 72 | `Drip Buckets` | `savedMaps` **32 / 24** |
| Slumberland | 177 | `A Pair Of Drawers` | `savedMaps` **32 / 24** |
| Slumberland | 256 | `Flaming Pathway` | `savedMaps` **32 / 24** |
| Teddy World | 326 | `Flaming guardpost` | `savedMaps` **35 / 24**, `grease` 22 / 16 |
| SpacePods | 40, 41, 48 | `Pod Side`, `Pod Side`, `Pod Standard Time` | `pendulums` **9 / 8** each |
| Teddy World | 249 | `Zippy's magic walls!` | `savedMaps` **28 / 24**, `grease` **24 / 16** |
| California or Bust! | 10 | `And the Pets, Too` | `dinahs` 18 / 18 — exactly full |

#### 9.9.5 The full resolution sweep — corpus maxima at eight screen sizes

Same model, same 22 houses, `numNeighbors = 9` except where noted. Bold = over cap. The last two
columns count rooms that are **at or over** some cap and rooms that are **strictly over** some cap
(the second is the number of rooms whose rendering actually differs from a hypothetical uncapped
build).

| Config | `dinahs`/18 | `savedMaps`/24 | `hotSpots`/56 | `masterObjects`/216 | `flames`/20 | `tikiFlames`/8 | `bbqCoals`/8 | `pendulums`/8 | `theStars`/4 | `grease`/16 | `tempManholes`/8 | at-or-over rooms | strictly-over rooms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 512 × 342, `numNeighbors = 1` (forced by `Main.c:191-192`) | 18 | 16 | 52 | 24 | 16 | 7 | 6 | 5 | 1 | 15 | 2 | 2 | 0 |
| 640 × 480 | 18 | 18 | 52 | 216 | 16 | 7 | 6 | 6 | 1 | 15 | 2 | 2 | 0 |
| 800 × 600 | 18 | 20 | 52 | 216 | 16 | 7 | 6 | 8 | 1 | 16 | 2 | 5 | 0 |
| 832 × 624 (Mac 16″) | 18 | 20 | 52 | 216 | 16 | 7 | 6 | 8 | 1 | 16 | 2 | 5 | 0 |
| 1024 × 768 | 18 | **32** | 52 | 216 | 16 | 7 | 6 | **9** | 2 | **21** | 3 | 13 | 9 |
| 1152 × 870 (Mac 21″) | 18 | **39** | 52 | 216 | **23** | 7 | 6 | **9** | 2 | **24** | 4 | 31 | 25 |
| 1280 × 1024 | 18 | **45** | 52 | 216 | **28** | 7 | 6 | **9** | 2 | **25** | 4 | 46 | 33 |
| 1536 × 1046 (`kMaxViewWidth`/`kMaxViewHeight` clamp) | 18 | **47** | 52 | 216 | **29** | 7 | 6 | **9** | 2 | **25** | 4 | 57 | 46 |

**No cap is *exceeded* by any shipped house at 832 × 624 or below.** That is the smoking gun: Glider PRO
was authored on Macs whose screens were 512 × 342 (Plus/SE/Classic), 640 × 480 (LC/II 13″) or
832 × 624 (16″), and every one of these houses is *exactly* within budget at those sizes — several
rooms sit precisely **at** a cap (`dinahs` 18/18, `grease` 16/16, `pendulums` 8/8), which is what a
content author hitting the editor's own guards looks like, but nothing spills. The caps were never
wrong for the content; the content was tuned to the caps at the resolution the author was using. The
9× per-room / per-neighbourhood gap of §9.8 was masked by `SectRect` clipping until 17″ and 21″
displays became common.

The progression is monotone and steep. Rooms at or over some cap: 2 at 640 × 480, 5 at 800 × 600 and
832 × 624, 13 at 1024 × 768, 31 at 1152 × 870, 46 at 1280 × 1024, 57 at the maximum view. Rooms
*strictly* over: 0, 0, 0, 9, 25, 33, 46. The first cap to actually break is a tie at **1024 × 768**,
where `savedMaps[]` (`In The Mirror` room 76, 32/24), `grease[]` (`Teddy World` room 322, 21/16) and
`pendulums[]` (`SpacePods` rooms 40, 41, 48, 9/8) all overflow together; `flames[]` holds until
1152 × 870. **`dinahs[]`, `hotSpots[]`, `tikiFlames[]`, `bbqCoals[]`, `theStars[]` and `tempManholes[]`
never break at any resolution** — `dinahs[]` and `grease[]` do reach exactly 18/18 and 16/16.

Note that `hotSpots[]` and `masterObjects[]` peaks are **resolution-independent** except through the
`numNeighbors` gate — 52 and 216/24 in every row — which is exactly what the code predicts, since
`ListAllLocalObjects` runs before any `SectRect` test exists.

#### 9.9.6 Conclusions for a port

1. **The exhaustion paths are reachable from shipped content**, not dead code. Any port that runs at a
   modern resolution with `numNeighbors = 9` will hit them on `Slumberland`, `In The Mirror`,
   `Teddy World`, `Rainbow's End`, `Titanic`, `Land of Illusion`, `Leviathan` and `SpacePods`.
2. **`savedMaps[]` (24) is by far the most-broken budget** — 42 distinct rooms over cap at the maximum
   view, peaking at 47. Because the prize cases gate their `Draw*` call on `legit != -1` (§9.4), the
   visible symptom is *missing prize art with working collision* — a prize you cannot see but can
   still collect. A faithful port must reproduce that, or those rooms will look different from the
   original.
3. **`grease[]` (16) is second** and behaves the same way (art suppressed).
4. **`flames[]` (20) overflows in five Slumberland rooms** at the maximum view — rooms 241, 246, 254,
   255 (all 21) and 257 (29). The excess candles keep their body art (drawn by `DrawSimpleBlowers`
   outside the flame registration, `ObjectDrawAll.c:76-77`) but never flicker. Rooms 254, 255 and 257
   overflow `savedMaps[]` **and** `flames[]` simultaneously; because `AddCandleFlame` tests
   `numFlames >= kMaxCandles` *before* calling `BackUpToSavedMap` (§9.5), the flame cap wins and the
   over-budget candles do not even consume a `savedMaps` slot — which is why the measured
   `savedMaps` demand in those rooms is an over-estimate.
5. **`pendulums[]` (8) overflows in three SpacePods rooms** (rooms 40 and 41 `Pod Side`, room 48
   `Pod Standard Time`, all 9) — and unlike the other overflows this one already happens at
   **1024 × 768**, not only at the maximum view.
   The dropped ninth cuckoo clock loses its pendulum but keeps its static art — and because
   `AddPendulum` sets the global `clockFrame = 10`, whether that ninth call happens changes the phase
   of **all eight** surviving pendulums (§9.5). This is a visible, resolution-dependent difference.
6. **`dinahs[]` (18) is reached exactly but never exceeded** — three rooms sit at 18/18
   (`California or Bust!` room 10, `Teddy World` rooms 244 and 379) and none goes over. So the editor's
   `HowManyDynamicObjects` guard, weak as it is, did its job for this corpus.
7. **`masterObjects[]` is exactly saturated** at 216 in every fully-populated neighbourhood,
   confirming the 9 × 24 dimensioning; houses with sparse floors show 24, 48 or 72 instead
   (`Sampler` 48, `California or Bust!` 72).
8. **`hotSpots[]` peaks at 52 of 56** (`Slumberland`, a single room's own objects) — 93 % full, never
   over (unverified: this column comes from a per-type hot-spot heuristic, not from re-running
   `CreateActiveRects`; see §9.9.2). `bbqCoals[]` peaks at 6/8, `tikiFlames[]` at 7/8, `theStars[]` at
   2/4, `tempManholes[]` at 4/8.
9. Therefore the **safe** port strategy is: keep the caps and the silent-drop semantics exactly (so
   the rendering matches), but bounds-check every −1 consumer of §9.10 (so nothing corrupts memory).
   Do **not** raise the caps: raising `savedMaps` to 48 would make prizes appear in `Slumberland` room
   257 that the original never showed.

### 9.10 The unchecked −1 consumers — the actual crash surface

`Interactions.c:1036-1151` is the link dispatch reached when the glider touches an object with a
link. Every one of these lines passes `masterObjects[linkIndex].dynaNum` with **no `!= -1` test**:

| Line | Call |
|---|---|
| `Interactions.c:1090` | `ToggleToaster(masterObjects[linkIndex].dynaNum)` |
| `:1094` | `ToggleMacPlus(...)` |
| `:1102` | `ToggleTV(...)` |
| `:1106` | `ToggleCoffee(...)` |
| `:1110` | `ToggleOutlet(...)` |
| `:1114` | `ToggleVCR(...)` |
| `:1118` | `ToggleStereos(...)` |
| `:1122` | `ToggleMicrowave(...)` |
| `:1126` | `ToggleBalloon(...)` |
| `:1131` | `ToggleCopter(...)` |
| `:1136` | `ToggleDart(...)` |
| `:1140` | `ToggleBall(...)` |
| `:1144` | `ToggleDrip(...)` |
| `:1148` | `ToggleFish(...)` |
| `:1060-1061` | `SpillGrease(masterObjects[linkIndex].dynaNum, masterObjects[linkIndex].hotNum)` |

and in `Triggers.c`'s `FireTrigger`:

| Line | Call |
|---|---|
| `Triggers.c:117-118` | `SpillGrease(masterObjects[triggeredIs].dynaNum, masterObjects[triggeredIs].hotNum)` |
| `:128` | `TriggerSwitch(masterObjects[triggeredIs].dynaNum)` → `HandleSwitches(&hotSpots[who])` |
| `:136` | `TriggerToast(...)` |
| `:148` | `TriggerOutlet(...)` |
| `:152` | `TriggerBalloon(...)` |
| `:157` | `TriggerCopter(...)` |
| `:162` | `TriggerDart(...)` |
| `:166` | `TriggerDrip(...)` |
| `:170` | `TriggerFish(...)` |
| `:187-188` | `SpillGrease(...)` (the `localLink == -1` fallback branch) |

Each of those bodies dereferences `dinahs[index]`, `hotSpots[who]` or `grease[who]` immediately
(e.g. `Trip.c:24`: `dinahs[index].active = !dinahs[index].active;`). With `index == -1` that is
`*(dinahs - 36 bytes)` — a write **before** the array. On a Mac with `NewPtr`-allocated blocks this
scribbles on the Memory Manager's block header, typically corrupting the heap and crashing later at an
unrelated place. In Go it is an immediate panic (or, with unsafe pointer arithmetic, the same
corruption).

**Minimum safe Go port:** bounds-check every one of these 25 call sites, returning silently on −1.
That is the behaviour the author clearly intended (the −1 sentinel exists precisely to mean "no
entry") and it cannot change behaviour for any house that does not overflow.

### 9.11 Summary answer to (f)

> When `kMaxDynamicObs` (18) is exhausted mid-draw, `AddDynamicObject` returns −1 without writing
> anything; the object's static art is still painted into `backSrcMap`, its hot spot (if central)
> still exists so collisions still work, and `masterObjects[].dynaNum` records −1. The object simply
> never animates and never responds to a switch or trigger — except that any switch or trigger linked
> to it will call `Toggle*(-1)`/`Trigger*(-1)` and index `dinahs[-1]`, corrupting memory. In the `kTV`
> case the exhaustion additionally leaves `tvWithMovieNumber == -1` with `tvInRoom == true`, which
> starts an un-stoppable QuickTime movie.
>
> When `kMaxHotSpots` (56) is exhausted, `AddActiveRect` returns −1; the object is still drawn but has
> no interaction rect, and multi-rect objects can end up with a *partial* rect set. The six switch
> types then copy that −1 into `dynaNum`, so `TriggerSwitch(-1)` indexes `hotSpots[-1]`.
>
> The two budgets are independent and are consumed at different times: `hotSpots[]` during
> `ListAllLocalObjects` (before any drawing, central room only), `dinahs[]` during the nine
> `DrawARoomsObjects` calls (all nine rooms for appliances, central only for glider-interacting
> types).
>
> The only budgets whose exhaustion suppresses drawing are `savedMaps[]` (24) and `grease[]` (16),
> because the prize and grease cases gate their `Draw*` call on `legit != -1`; over-budget prizes
> become invisible but stay collectable, and over-budget grease patches become invisible but stay
> slippery.
>
> None of this is hypothetical. Measured over all 22 shipped houses (§9.9): nothing overflows at
> 832 × 624 or below; at 1024 × 768 thirteen rooms are at or over a cap and nine are strictly over
> (`savedMaps[]`, `grease[]` and `pendulums[]` all break there); at the maximum 1536 × 1046 view
> fifty-seven rooms are at or over and forty-six strictly over, with `savedMaps[]` peaking at 47
> against a cap of 24 in `Slumberland` room 257 (`Speed Slick`). The editor's twelve `HowMany*Objects` guards (§9.7) count
> a **single room** while every runtime budget is consumed by the whole **3 × 3 neighbourhood**, and
> `SectRect` clipping hid that factor-of-nine gap at the resolutions the houses were authored on.
> `dinahs[]` and `hotSpots[]` are the two budgets the corpus never breaks (18/18 and 52/56 peaks).

---
## 10. Case-by-case reference for all 74 case groups

### 10.1 Coverage

`GliderPRO/Sources/ObjectDrawAll.c:51-950` is one `switch (thisObject.what)` containing **118 `case`
labels** grouped into **74 bodies** (a body ends at its `break;`). Mechanically extracted counts:

| Quantity | Value | How derived |
|---|---|---|
| `case` labels | 118 | `grep -c 'case k'` over `:51-950` |
| distinct bodies (groups) | 74 | labels followed by shared code up to `break;` |
| object types defined in `GliderDefines.h:311-435` | 117 (`0x01`–`0x8F`, with gaps) | enumerated below |
| plus `kObjectIsEmpty` = −1 (`GliderDefines.h:526`) | 1 | |
| types with **no** case label | **0** | every defined type is handled |
| numeric values in `0x01..0x8F` with no `#define` | **26**: `0x20`, `0x30`, `0x4A`–`0x50`, `0x59`–`0x5F`, `0x60`, `0x6F`, `0x70`, `0x7A`–`0x80` | family-boundary padding |

There is **no `default:` arm**. A `what` value in one of those 26 gaps (a corrupt or
forward-version house file) falls straight through the `switch` and is a silent no-op — but note it
*does* still reach the write-back at `:953-961`, so `masterObjects[].dynaNum` is set to −1 for it.
`IsThisValid` (`GliderPRO/Sources/Objects.c:89-122`) also has no default, and its `itsGood` starts
`true`, so a garbage type is treated as valid and scanned.

### 10.2 The table

Column meanings:

* **Lines** — first `case` label line through the `break;` line, in the CR→LF converted copy.
* **Draw call** — the `Draw*` helper(s) invoked; `—` means the case paints nothing at all.
* **`isLit`?** — whether the case consults `isLit` (`= numLights > 0`, `:38`).
* **`SectRect`?** — whether the case guards on `SectRect(&itsRect, &testRect, &whoCares)`. `no` here is
  significant: those cases draw or register unconditionally, off-screen or not.
* **Registrar** — the side-effect calls that add to `flames[]`/`tikiFlames[]`/`bbqCoals[]`/
  `pendulums[]`/`theStars[]`/`savedMaps[]`/`grease[]`/`dinahs[]`/`tempManholes[]`/`mirrorRgn`.
* **`dynamicNum`** — `set` if the case assigns `dynamicNum`, which is then written into
  `masterObjects[].dynaNum` at `:953-961`; `−` means it stays at the `:45` initial value of −1.

| # | `case` label(s) with hex value | Lines | Draw call | `isLit`? | `SectRect`? | Registrar | `dynamicNum` |
|---|---|---|---|---|---|---|---|
| 1 | `kObjectIsEmpty` -1 | 53–54 | — | no | no | — | − |
| 2 | `kFloorVent` 0x01, `kCeilingVent` 0x02, `kFloorBlower` 0x03, `kCeilingBlower` 0x04, `kSewerGrate` 0x05, `kLeftFan` 0x06, `kRightFan` 0x07, `kGrecoVent` 0x0E, `kSewerBlower` 0x0F | 56–69 | DrawSimpleBlowers | yes | yes | — | − |
| 3 | `kTaper` 0x08 | 71–103 | DrawSimpleBlowers | yes | yes | AddCandleFlame, ReBackUpFlames | − |
| 4 | `kCandle` 0x09 | 105–137 | DrawSimpleBlowers | yes | yes | AddCandleFlame, ReBackUpFlames | − |
| 5 | `kStubby` 0x0A | 139–171 | DrawSimpleBlowers | yes | yes | AddCandleFlame, ReBackUpFlames | − |
| 6 | `kTiki` 0x0B | 173–183 | DrawTiki | yes | no | AddTikiFlame, ReBackUpTikiFlames | − |
| 7 | `kBBQ` 0x0C | 185–198 | DrawPictSansWhiteObject | yes | yes | AddBBQCoals, ReBackUpBBQCoals | − |
| 8 | `kInvisBlower` 0x0D, `kLiftArea` 0x10 | 200–202 | — | no | no | — | − |
| 9 | `kTable` 0x11 | 204–209 | DrawTable | yes | no | — | − |
| 10 | `kShelf` 0x12 | 211–216 | DrawShelf | yes | no | — | − |
| 11 | `kCabinet` 0x13 | 218–223 | DrawCabinet | yes | yes | — | − |
| 12 | `kFilingCabinet` 0x14, `kOzma` 0x81 | 225–231 | DrawPictObject | yes | yes | — | − |
| 13 | `kWasteBasket` 0x15, `kMilkCrate` 0x16 | 233–239 | DrawSimpleFurniture | yes | yes | — | − |
| 14 | `kCounter` 0x17 | 241–246 | DrawCounter | yes | yes | — | − |
| 15 | `kDresser` 0x18 | 248–253 | DrawDresser | yes | no | — | − |
| 16 | `kDeckTable` 0x19 | 255–260 | DrawDeckTable | yes | no | — | − |
| 17 | `kStool` 0x1A | 262–267 | DrawStool | yes | no | — | − |
| 18 | `kInvisObstacle` 0x1C | 269–270 | — | no | no | — | − |
| 19 | `kManhole` 0x1D | 272–281 | DrawPictSansWhiteObject | yes | yes | AddTempManholeRect | − |
| 20 | `kInvisBounce` 0x1F | 283–284 | — | no | no | — | − |
| 21 | `kRedClock` 0x21 | 286–298 | DrawRedClock | no | yes | BackUpToSavedMap, ReBackUpSavedMap | − |
| 22 | `kBlueClock` 0x22 | 300–312 | DrawBlueClock | no | yes | BackUpToSavedMap, ReBackUpSavedMap | − |
| 23 | `kYellowClock` 0x23 | 314–326 | DrawYellowClock | no | yes | BackUpToSavedMap, ReBackUpSavedMap | − |
| 24 | `kCuckoo` 0x24 | 328–347 | DrawCuckoo | no | yes | AddPendulum, BackUpToSavedMap, ReBackUpPendulum, ReBackUpSavedMap | − |
| 25 | `kPaper` 0x25, `kBattery` 0x26, `kBands` 0x27, `kHelium` 0x2E | 349–364 | DrawSimplePrizes | no | yes | BackUpToSavedMap, ReBackUpSavedMap | − |
| 26 | `kGreaseRt` 0x28 | 366–385 | DrawGreaseRt | no | yes | AddGrease, ReBackUpGrease | set |
| 27 | `kGreaseLf` 0x29 | 387–406 | DrawGreaseLf | no | yes | AddGrease, ReBackUpGrease | set |
| 28 | `kFoil` 0x2A | 408–420 | DrawFoil | no | yes | BackUpToSavedMap, ReBackUpSavedMap | − |
| 29 | `kInvisBonus` 0x2B, `kSlider` 0x2F | 422–424 | — | no | no | — | − |
| 30 | `kStar` 0x2C | 426–445 | DrawSimplePrizes | no | yes | AddStar, BackUpToSavedMap, ReBackUpSavedMap, ReBackUpStar | − |
| 31 | `kSparkle` 0x2D | 447–460 | — | no | yes | AddDynamicObject | set |
| 32 | `kUpStairs` 0x31, `kDoorInLf` 0x37, `kDoorInRt` 0x38, `kWindowInLf` 0x3B, `kWindowInRt` 0x3C | 462–471 | DrawPictSansWhiteObject | no | yes | — | − |
| 33 | `kDownStairs` 0x32, `kDoorExRt` 0x39, `kDoorExLf` 0x3A, `kWindowExRt` 0x3D, `kWindowExLf` 0x3E | 473–482 | DrawPictObject | no | yes | — | − |
| 34 | `kMailboxLf` 0x33 | 484–488 | DrawMailboxLeft | no | no | — | − |
| 35 | `kMailboxRt` 0x34 | 490–494 | DrawMailboxRight | no | no | — | − |
| 36 | `kFloorTrans` 0x35, `kCeilingTrans` 0x36 | 496–502 | DrawSimpleTransport | no | yes | — | − |
| 37 | `kInvisTrans` 0x3F, `kDeluxeTrans` 0x40 | 504–506 | — | no | no | — | − |
| 38 | `kLightSwitch` 0x41 | 508–519 | DrawLightSwitch | no | yes | — | set |
| 39 | `kMachineSwitch` 0x42 | 521–532 | DrawMachineSwitch | no | yes | — | set |
| 40 | `kThermostat` 0x43 | 534–545 | DrawThermostat | no | yes | — | set |
| 41 | `kPowerSwitch` 0x44 | 547–558 | DrawPowerSwitch | no | yes | — | set |
| 42 | `kKnifeSwitch` 0x45 | 560–571 | DrawKnifeSwitch | no | yes | — | set |
| 43 | `kInvisSwitch` 0x46 | 573–575 | — | no | no | — | set |
| 44 | `kTrigger` 0x47, `kLgTrigger` 0x48, `kSoundTrigger` 0x49 | 577–580 | — | no | no | — | − |
| 45 | `kCeilingLight` 0x51, `kLightBulb` 0x52, `kTableLamp` 0x53 | 582–589 | DrawSimpleLight | yes | yes | — | − |
| 46 | `kTrunk` 0x1B, `kBooks` 0x1E, `kHipLamp` 0x54, `kDecoLamp` 0x55, `kGuitar` 0x64, `kCinderBlock` 0x6B, `kFlowerBox` 0x6C, `kFireplace` 0x84, `kBear` 0x87, `kVase1` 0x89, `kVase2` 0x8A, `kRug` 0x8E, `kChimes` 0x8F | 591–608 | DrawPictSansWhiteObject | yes | yes | — | − |
| 47 | `kCustomPict` 0x6E | 610–615 | DrawCustPictSansWhite | yes | yes | — | − |
| 48 | `kFlourescent` 0x56 | 617–622 | DrawFlourescent | yes | yes | — | − |
| 49 | `kTrackLight` 0x57 | 624–629 | DrawTrackLight | yes | yes | — | − |
| 50 | `kInvisLight` 0x58 | 631–632 | — | no | no | — | − |
| 51 | `kShredder` 0x61, `kCDs` 0x6D | 634–640 | DrawSimpleAppliance | yes | yes | — | − |
| 52 | `kToaster` 0x62 | 642–656 | DrawSimpleAppliance | no | yes | AddDynamicObject | set |
| 53 | `kMacPlus` 0x63 | 658–672 | DrawMacPlus | yes | yes | AddDynamicObject | set |
| 54 | `kTV` 0x65 | 674–713 | DrawTV | yes | yes | AddDynamicObject | set |
| 55 | `kCoffee` 0x66 | 715–729 | DrawCoffee | yes | yes | AddDynamicObject | set |
| 56 | `kOutlet` 0x67 | 731–746 | DrawOutlet | yes | yes | AddDynamicObject | set |
| 57 | `kVCR` 0x68 | 748–762 | DrawVCR | yes | yes | AddDynamicObject | set |
| 58 | `kStereo` 0x69 | 764–778 | DrawStereo | yes | yes | AddDynamicObject | set |
| 59 | `kMicrowave` 0x6A | 780–794 | DrawMicrowave | yes | yes | AddDynamicObject | set |
| 60 | `kBalloon` 0x71 | 796–805 | — | no | no | AddDynamicObject | set |
| 61 | `kCopterLf` 0x72 | 807–816 | — | no | no | AddDynamicObject | set |
| 62 | `kCopterRt` 0x73 | 818–827 | — | no | no | AddDynamicObject | set |
| 63 | `kDartLf` 0x74 | 829–838 | — | no | no | AddDynamicObject | set |
| 64 | `kDartRt` 0x75 | 840–849 | — | no | no | AddDynamicObject | set |
| 65 | `kBall` 0x76 | 851–860 | — | no | no | AddDynamicObject | set |
| 66 | `kDrip` 0x77 | 862–876 | DrawDrip | no | yes | AddDynamicObject | set |
| 67 | `kFish` 0x78 | 878–892 | DrawFish | no | yes | AddDynamicObject | set |
| 68 | `kCobweb` 0x79, `kCloud` 0x8C | 894–900 | DrawPictWithMaskObject | yes | yes | — | − |
| 69 | `kMirror` 0x82 | 902–912 | DrawMirror | yes | yes | AddToMirrorRegion | − |
| 70 | `kMousehole` 0x83, `kFaucet` 0x8D | 914–920 | DrawSimpleClutter | yes | yes | — | − |
| 71 | `kFlower` 0x85 | 922–927 | DrawFlower | yes | yes | — | − |
| 72 | `kWallWindow` 0x86 | 929–934 | DrawWallWindow | no | yes | — | − |
| 73 | `kCalendar` 0x88 | 936–941 | DrawCalendar | yes | yes | — | − |
| 74 | `kBulletin` 0x8B | 943–948 | DrawBulletin | yes | yes | — | − |

### 10.3 Reading the table — the seven non-obvious groups

**Group 2 (blowers, 9 labels).** One helper, `DrawSimpleBlowers(thisObject.what, &itsRect)`, dispatches
internally on `what`. The `kLeftFan`/`kRightFan` blades are static art; the *push* is a hot spot, not a
dynamic object.

**Group 6 (`kTiki`).** The only case that **registers an animated element with no `SectRect` guard
whatsoever** — five other groups (`kTable`, `kShelf`, `kDresser`, `kDeckTable`, `kStool`) and two more
(`kMailboxLf`, `kMailboxRt`) draw unguarded, but none of them registers anything; see §10.6 for the
complete list. `DrawTiki` and `AddTikiFlame`/`ReBackUpTikiFlames` run for every valid tiki in every one
of the nine rooms, on-screen or not. `AddTikiFlame`'s own coordinate guard (`h < 8 || v < 10`,
`GliderPRO/Sources/DynamicMaps.c:405`) is the only thing that stops it registering a flame at a
negative sheet position. Contrast group 3/4/5 (`kTaper`/`kCandle`/`kStubby`) which *are* guarded and
which additionally run a proximity test for non-central rooms (§5.5.1).

**Group 7 (`kBBQ`).** `SectRect`-guarded, `isLit`-gated for the draw, but the coal registration is
**not** central-gated and **not** proximity-tested — unlike the three candle types. A BBQ in a
neighbour room whose coals overlap the central viewport will animate its coals over the neighbour's
art. This asymmetry looks like an oversight: `kTaper`/`kCandle`/`kStubby` got the proximity fix and
`kTiki`/`kBBQ` did not.

**Groups 26/27 (grease).** The only cases with a **two-armed** structure keyed on object state
(`thisObject.data.c.state`, "standing"). Standing grease is `SectRect`-guarded and its draw is gated on
`dynamicNum != -1`; fallen grease is drawn with **no `SectRect` and no gate at all**. `dynamicNum` here
is a `grease[]` index, not a `dinahs[]` index (§7.5). Note also that `IsThisValid` does **not** list
`kGreaseRt`/`kGreaseLf`, so a spilled grease is still scanned and still drawn in its fallen form.

**Group 31 (`kSparkle`).** Draws **nothing**. `GetObjectRect` and `OffsetRectRoomRelative` run purely to
compute the registration rect. Central-room-only, `!redraw`-only, `SectRect`-guarded. This is the only
group whose entire purpose is registration.

**Groups 38–43 (the six switch types).** `dynamicNum = masterObjects[i].hotNum;` sits **outside** the
`SectRect` block (lines `:518`, `:531`, `:544`, `:557`, `:570`, `:574`) so an off-screen switch still
gets its `dynaNum` link — necessary, because a remote trigger must be able to flip a switch you cannot
see. But `masterObjects[i]` is indexed by the **room slot** `i`, which only names the right master entry
for the central room (§7.7). Note `kInvisSwitch` (group 43) has a two-line body: no rect, no draw,
just the `dynaNum` copy.

**Groups 60–65 (`kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`).** Inverted
structure: the room/redraw gate is the **outermost** test, and `GetObjectRect` is inside it. No
`SectRect`, no draw. `itsRect` itself (not a copy in `rectA`) is offset by
`-playOriginH, -playOriginV` and passed to `AddDynamicObject`, because nothing downstream in the case
needs the screen rect. All six are pure registrations: the sprite is drawn entirely by
`RenderDynamics` (`GliderPRO/Sources/Dynamics3.c:112-154`).

### 10.4 Draw-helper frequency

| Helper | Groups using it | Notes |
|---|---|---|
| `DrawPictSansWhiteObject` | 4 (7, 19, 32, 46) | white-keyed blit; group 46 alone covers 13 object types |
| `DrawSimpleBlowers` | 4 (2, 3, 4, 5) | internal dispatch on `what` |
| `DrawPictObject` | 2 (12, 33) | opaque blit |
| `DrawSimplePrizes` | 2 (25, 30) | internal dispatch on `what` |
| `DrawSimpleAppliance` | 2 (51, 52) | internal dispatch on `what` |
| `DrawGreaseRt` / `DrawGreaseLf` | 1 each (26, 27) | called twice per group (standing / fallen) |
| everything else | 1 group each | 33 further helpers |
| **no helper** | 10 groups (1, 8, 18, 20, 29, 31, 37, 44, 50, 60–65) | see §6.1 |

Counting object *types* rather than groups: **15 types draw nothing**:
`kObjectIsEmpty` (−1), `kInvisBlower` (0x0D), `kLiftArea` (0x10), `kInvisObstacle` (0x1C),
`kInvisBounce` (0x1F), `kInvisBonus` (0x2B), `kSlider` (0x2F), `kSparkle` (0x2D),
`kInvisTrans` (0x3F), `kDeluxeTrans` (0x40), `kInvisSwitch` (0x46), `kTrigger` (0x47),
`kLgTrigger` (0x48), `kSoundTrigger` (0x49), `kInvisLight` (0x58) — plus the six enemy types
(0x71–0x76) which draw nothing *here* but are drawn every frame by `RenderDynamics`.

### 10.5 The `isLit` gate, exhaustively

`isLit = (numLights > 0);` at `ObjectDrawAll.c:38`, from the global `numLights` that `DrawLocale`
re-computes before each of the nine calls (`GliderPRO/Sources/RoomGraphics.c:81`, `:85`, `:89`, `:93`,
`:97`, `:101`, `:108`, `:113`, `:119`).

Types whose art is **suppressed in a dark room** (39 groups, marked `yes` above): all blowers, the
three candles, tiki, BBQ, all furniture, the manhole art, all lights, `kCustomPict`,
`kShredder`/`kCDs`, `kMacPlus`/`kTV`/`kCoffee`/`kOutlet`/`kVCR`/`kStereo`/`kMicrowave` bodies,
`kCobweb`/`kCloud`, `kMirror`, and 5 of the 6 remaining clutter groups.

Types whose art is **always drawn even in the dark** (notable, because it looks like a bug list):

| Type | Line | Why it matters |
|---|---|---|
| all 12 prize types (groups 21–25, 28, 30) | `:296`, `:310`, `:324`, `:339`, `:362`, `:418`, `:442` | prizes glow in the dark; probably deliberate (they must be findable) |
| `kGreaseRt`/`kGreaseLf` | `:380`, `:384`, `:401`, `:405` | grease visible in the dark |
| stairs/doors/windows (groups 32, 33) | `:470`, `:481` | room exits must stay visible |
| `kMailboxLf`/`kMailboxRt` | `:487`, `:493` | and unguarded by `SectRect` too |
| `kFloorTrans`/`kCeilingTrans` | `:501` | transport pads |
| the five visible switch types | `:516`, `:529`, `:542`, `:555`, `:568` | you must be able to find the light switch **in a dark room** — clearly deliberate |
| `kToaster` | `:647` | inconsistent with `kShredder`/`kCDs`, which use the same helper and *are* gated |
| `kDrip`, `kFish` | `:867`, `:883` | |
| **`kWallWindow`** | `:932` | inconsistent with every other clutter type, all of which are gated. `GetNumberOfLights` (`GliderPRO/Sources/Room.c:970-1095`) counts `kWallWindow` as a *light source*, so a room containing one can never be dark — the missing gate is therefore unreachable in practice, which is probably why it was never noticed. |

`kToaster`'s missing `isLit` gate **is** reachable and is a genuine visual inconsistency: a dark room
containing a toaster shows the toaster.

### 10.6 The `SectRect` gate, exhaustively

Cases with **no** `SectRect(&itsRect, &testRect, ...)` guard anywhere in the body:

| Group | Types | What runs unguarded |
|---|---|---|
| 6 | `kTiki` | `DrawTiki` + `AddTikiFlame`/`ReBackUpTikiFlames` |
| 9, 10, 15, 16, 17 | `kTable`, `kShelf`, `kDresser`, `kDeckTable`, `kStool` | the draw (all `isLit`-gated) |
| 26, 27 (partial) | `kGreaseRt`, `kGreaseLf` | the **fallen** arm's draw |
| 34, 35 | `kMailboxLf`, `kMailboxRt` | the draw |
| 38–43 (partial) | the six switch types | `dynamicNum = masterObjects[i].hotNum` |
| 60–65 | `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall` | `AddDynamicObject` |
| 69 (partial) | `kMirror` | `InsetRect` + `AddToMirrorRegion` |

Mechanically (`SectRect` absent from the whole group body over `:51-950`), **23 of the 74 groups**
never call `SectRect`; the other 9 of those 23 are the pure no-op groups `kObjectIsEmpty` (`:53-54`),
`kInvisBlower`/`kLiftArea` (`:200-202`), `kInvisObstacle` (`:269-270`), `kInvisBounce` (`:283-284`),
`kInvisBonus`/`kSlider` (`:422-424`), `kInvisTrans`/`kDeluxeTrans` (`:504-506`), `kInvisSwitch`
(`:573-575`), `kTrigger`/`kLgTrigger`/`kSoundTrigger` (`:577-580`) and `kInvisLight` (`:631-632`), which
have nothing to guard. So **14 groups do real work unguarded**, and everything else is guarded.

The unguarded draws are safe only because QuickDraw clips `CopyBits` to the
destination `PixMap` bounds — a Go port that indexes a pixel slice directly **must** clip explicitly
(§11).

The five unguarded furniture types (`kTable` `:204-209`, `kShelf` `:211-216`, `kDresser` `:248-253`,
`kDeckTable` `:255-260`, `kStool` `:262-267`) are the five whose helper draws geometry **outside**
`itsRect` — a drop shadow, a pedestal, or a support column reaching down to a fixed floor line; the
guard was presumably dropped because a `SectRect` on `itsRect` alone would clip an object whose visible
extent is much larger than `itsRect`. Their direct neighbours in the same `switch` — `kCabinet` `:221`,
`kFilingCabinet`/`kOzma` `:229`, `kWasteBasket`/`kMilkCrate` `:237`, `kCounter` `:244` — all *do* test
`(SectRect(&itsRect, &testRect, &whoCares)) && isLit`, so the difference is deliberate, not an
oversight.

The related origin defect is narrower than an earlier draft of this document claimed. Only **two**
call sites pass an un-corrected origin — `DrawTable(&itsRect, playOriginV)` at `:208` and
`DrawDeckTable(&itsRect, playOriginV)` at `:259` — against **four** that pass the corrected
`playOriginV + VerticalRoomOffset(neighbor)`: `DrawTiki` `:177`, `DrawStool` `:266`,
`DrawMailboxLeft` `:487`, `DrawMailboxRight` `:493`. `DrawShelf` (`:215`), `DrawCabinet` (`:222`),
`DrawCounter` (`:245`) and `DrawDresser` (`:252`) take **no** origin parameter at all — see the
signature table in §5.2. So a `kTable` or `kDeckTable` in the north/south/diagonal rooms computes its
pedestal, base and shadow against the *central* room's floor, 322 px away. §5.2 works through exactly
what is visible (a stray pedestal stripe for north-side rooms, a missing pedestal entirely for
south-side rooms) and counts 200 affected tables in the shipped corpus — every `kTable` and
`kDeckTable` there is. Reproduce it: houses were
authored against this behaviour.

---

## 11. Mac Toolbox inventory and Go replacements

### 11.1 Called directly from `ObjectDrawAll.c`

| Toolbox call | Line(s) | Semantics | Go replacement |
|---|---|---|---|
| `HGetState((Handle)thisHouse)` | `:40` | read the Memory Manager flags byte of a relocatable block | nothing — Go objects do not move |
| `HLock((Handle)thisHouse)` | `:41` | pin the block so `(*thisHouse)->rooms[...]` stays valid across 900 lines of Toolbox calls | nothing |
| `HSetState((Handle)thisHouse, wasState)` | `:964` | restore the flags byte | nothing |
| `SectRect(&a, &b, &out)` | **54** sites (51 of the 74 case groups call it once; `kTaper`, `kCandle` and `kStubby` each call it a second time for the proximity test at `:93`, `:127`, `:161`) | `out = a ∩ b`; returns `false` and **zeroes `out`** if empty | `func sectRect(a, b Rect) (Rect, bool)`; must zero the output on the empty case because callers reuse `whoCares` |
| `OffsetRect(&r, dh, dv)` | `:685` | translate | trivial; note the Toolbox version, unlike `QOffsetRect`, is the real trap |
| `InsetRect(&r, 4, 4)` | `:909` | shrink by 4 on all sides | trivial; can invert the rect if it is smaller than 8×8 — QuickDraw tolerates inverted rects, Go code must not assume `left <= right` |
| `NewRgn()` / `RectRgn()` / `DisposeRgn()` | `:689`, `:690`, `:692` | allocate/populate/free a QuickDraw region | a `Rect` clip; the region only exists to satisfy `SetMovieDisplayClipRgn` |
| `GetMovieBox` / `SetMovieBox` / `SetMovieDisplayClipRgn` | `:686`, `:688`, `:691` | QuickTime geometry | see §8.9 |

`RgnHandle theRgn;` is declared at `:27` and is **only** used inside the `#ifdef COMPILEQT` block. In a
non-QuickTime build it is an unused local — harmless, but it means a Go port with no video support
needs neither the variable nor the region type.

### 11.2 Reached one level down (the `Draw*` helpers, `GliderPRO/Sources/ObjectsDraw*.c`)

| Toolbox facility | Typical use | Go replacement |
|---|---|---|
| `CopyBits(srcBits, dstBits, &srcRect, &dstRect, srcCopy, nil)` | every opaque blit | `draw.Draw` equivalent over an 8-bit indexed buffer, or a hand-written `copy` loop per scanline |
| `CopyBits(..., transparent, nil)` | the "sans white" blits | per-pixel test against the white index; **note this is index 0 in the game palette, not RGB white** |
| `CopyMask(srcBits, maskBits, dstBits, &src, &mask, &dst)` | `DrawPictWithMaskObject` (`kCobweb`, `kCloud`) | 1-bit mask test per pixel |
| `GWorldPtr` + `GetGWorldPixMap` + `SetGWorld`/`GetGWorld` | `backSrcMap`, `workSrcMap`, `savedMaps[].map`, the ~30 art sheets | plain `[]uint8` buffers plus explicit width/stride; **there is no ambient "current port" in Go, so every draw must take its destination explicitly** |
| `SetPort((GrafPtr)workSrcMap)` | `DrawRoomBackground` at `RoomGraphics.c:238`, never restored | the ambient-port leak must be *modelled*: see §11.3 |
| `LoadGraphic` / `LoadGraphicSpecial` / `LoadScaledGraphic` (Resource Manager `GetPicture`) | fetching `PICT` art by resource ID | decode the `PICT` resources ahead of time into indexed bitmaps keyed by ID |
| `PaintRect` | `DrawLocale`'s black fill at `RoomGraphics.c:76` | `memset` the buffer to the black index |
| 8-bit indexed colour, `kPreferredDepth` | every surface | keep the indexed model; converting to RGBA early breaks the "transparent = white index" blits and the `thisMac.isDepth == 4` fixups |

### 11.3 The ambient-port hazard, stated precisely

`DrawRoomBackground` does `SetPort((GrafPtr)workSrcMap)` at
`GliderPRO/Sources/RoomGraphics.c:238` and **never restores it**. `DrawLocale` had earlier done
`SetGWorld(backSrcMap, nil)` at `:75`. So when `DrawARoomsObjects` runs, the *current GWorld* is
`backSrcMap` but the *current port* may be `workSrcMap`.

This does not corrupt anything, because **every** `Draw*` helper names its destination explicitly in the
`CopyBits` call (`GetPortBitMapForCopyBits(backSrcMap)` or equivalent). The only Toolbox calls in
`ObjectDrawAll.c` that would consult the ambient port are `SectRect`, `OffsetRect` and `InsetRect`,
which are pure rect arithmetic and consult nothing.

**Go port rule:** make the destination surface an explicit parameter of every draw function. Do not
model a current-port global; if you do, you must reproduce the leak, and the leak is only benign by
accident.

### 11.4 Endianness and the on-disk house

Every field the dispatcher reads comes out of the big-endian house file:

| Field | Offset in `objectType` | Type | Size |
|---|---|---|---|
| `what` | 0 | `short` (BE) | 2 |
| `data.a.topLeft.v` | 2 | `short` (BE) — **v before h** | 2 |
| `data.a.topLeft.h` | 4 | `short` (BE) | 2 |
| `data.a.distance` | 6 | `short` (BE) | 2 |
| `data.a.initial` | 8 | `Boolean` | 1 |
| `data.a.state` | 9 | `Boolean` | 1 |
| `data.a.vector` | 10 | `Byte` | 1 |
| `data.a.tall` | 11 | `Byte` | 1 |
| `data.b.bounds` (`Rect`) | 2 | 4 × `short` BE, order **top, left, bottom, right** | 8 |
| `data.b.pict` | 10 | `short` (BE) | 2 |
| `data.c.length` | 6 | `short` (BE) | 2 |
| `data.c.points` | 8 | `short` (BE) | 2 |
| `data.c.state` | 10 | `Boolean` | 1 |
| `data.c.initial` | 11 | `Boolean` | 1 |
| `data.d.tall` | 6 | `short` (BE) | 2 |
| `data.d.where` | 8 | `short` (BE) | 2 |
| `data.d.who` | 10 | `Byte` | 1 |
| `data.d.wide` | 11 | `Byte` | 1 |
| `data.e.delay` | 6 | `short` (BE) | 2 |
| `data.e.where` | 8 | `short` (BE) | 2 |
| `data.e.who` | 10 | `Byte` | 1 |
| `data.e.type` | 11 | `Byte` | 1 |
| `data.f.length` | 6 | `short` (BE) | 2 |
| `data.f.byte0`, `data.f.byte1` | 8, 9 | `Byte` | 1 each |
| `data.f.initial`, `data.f.state` | 10, 11 | `Boolean` | 1 each |
| `data.g.height` | 6 | `short` (BE) — also used as a `PICT` ID for `kCustomPict` | 2 |
| `data.g.byte0`, `data.g.delay` | 8, 9 | `Byte` | 1 each |
| `data.g.initial`, `data.g.state` | 10, 11 | `Boolean` | 1 each |
| `data.h.length` | 6 | `short` (BE) | 2 |
| `data.h.delay`, `data.h.byte0` | 8, 9 | `Byte` | 1 each |
| `data.h.initial`, `data.h.state` | 10, 11 | `Boolean` | 1 each |
| `data.i.bounds` (`Rect`) | 2 | 4 × `short` BE | 8 |
| `data.i.pict` | 10 | `short` (BE) | 2 |

(All nine union arms verified as exactly 10 bytes at `GliderPRO/Headers/GliderStructs.h:11-105`;
`objectType` = 12 bytes at `:105`.) `Boolean` is a Mac `unsigned char` where any non-zero is true; the
house files use `0x00`/`0x01`. `Byte` is `unsigned char`. `Point` is `{short v; short h;}` — **v
first** — which is the single most common porting error in Mac code.

Note the `union`: `data.c.state` (offset 10) and `data.g.state` (offset 11) are **different bytes**.
The dispatcher picks the arm per case, and a Go port must too — a single flat struct with one `state`
field will silently mis-read half the object types. Specifically: prizes and grease read `state` at
**offset 10**; lights, appliances and enemies read `state` at **offset 11**.

---

## 12. A faithful Go port of `DrawARoomsObjects`

Numbered to follow the C control flow exactly. Original C identifiers in parentheses.

```
DrawARoomsObjects(neighbor int16, redraw bool):

 1. if localNumbers[neighbor] == kRoomIsEmpty (-1):            /* :33-34 */
        return

 2. testRect (testRect) = zeroCorner(houseRect)                /* :36-37 */
        /* houseRect is set once in InterfaceInit.c:196-201:
             houseRect = screen; houseRect.bottom -= kScoreboardTall (20)
             clamp right  to kMaxViewWidth  (1536)
             clamp bottom to kMaxViewHeight (1026)
           zeroCorner leaves (0, 0, w, h). */

 3. isLit (isLit) = numLights > 0                               /* :38 */
        /* numLights is a GLOBAL that DrawLocale re-assigns before each of the
           nine calls; see RoomGraphics.c:81,85,89,93,97,101,108,113,119. */

 4. /* :40-41 HLock -- no Go equivalent */

 5. for i := 0; i < kMaxRoomObs (24); i++ {                      /* :43 */

 6.     dynamicNum := int16(-1)                                  /* :45 */
 7.     legit      := int16(-1)                                  /* :46 */

 8.     if IsThisValid(localNumbers[neighbor], i) {               /* :48 */

 9.         thisObject := house.rooms[localNumbers[neighbor]].objects[i]  /* :50 (a COPY) */

10.         switch thisObject.what {                              /* :51 */
               /* 74 bodies; see section 10. The canonical shapes are:

                  (A) simple static, guarded and lit  -- 39 groups
                        itsRect = GetObjectRect(thisObject)
                        itsRect = OffsetRectRoomRelative(itsRect, neighbor)
                        if sectRect(itsRect, testRect) && isLit {
                            DrawXxx(itsRect)     /* into backSrcMap */
                        }

                  (B) prize, savedMap-backed -- 7 groups
                        itsRect = ...
                        if sectRect(itsRect, testRect) {
                            if redraw { legit = ReBackUpSavedMap(itsRect, room, i) }
                            else      { legit = BackUpToSavedMap(itsRect, room, i) }
                            if legit != -1 { DrawXxx(itsRect) }
                        }

                  (C) appliance, dinahs-backed -- 7 groups
                        itsRect = ...
                        if sectRect(itsRect, testRect) {
                            DrawXxx(itsRect, thisObject.data.g.state, isLit)
                            if !redraw {
                                rectA = itsRect
                                rectA.offset(-playOriginH, -playOriginV)
                                dynamicNum = AddDynamicObject(what, rectA, thisObject,
                                                localNumbers[neighbor], i,
                                                thisObject.data.g.state)
                            }
                        }

                  (D) enemy, central-only, no draw -- 6 groups
                        if neighbor == kCentralRoom && !redraw {
                            itsRect = GetObjectRect(thisObject)
                            itsRect = OffsetRectRoomRelative(itsRect, neighbor)
                            itsRect.offset(-playOriginH, -playOriginV)
                            dynamicNum = AddDynamicObject(what, itsRect, thisObject,
                                            localNumbers[neighbor], i,
                                            thisObject.data.h.state)
                        }

                  (E) switch -- 6 groups
                        itsRect = ...
                        if sectRect(itsRect, testRect) {
                            floor, suite = ExtractFloorSuite(thisObject.data.e.where)
                            room = GetRoomNumber(floor, suite)
                            obj  = int16(thisObject.data.e.who)
                            DrawXxxSwitch(itsRect, GetObjectState(room, obj))
                        }
                        dynamicNum = masterObjects[i].hotNum   /* OUTSIDE the guard */

                  and the 15 special shapes documented case-by-case in section 10. */
            }
        }

11.     if !redraw {                                             /* :953 */
12.         for n := 0; n < numMasterObjects; n++ {               /* :955 */
13.             if masterObjects[n].objectNum == i &&
                   masterObjects[n].roomNum == localNumbers[neighbor] {  /* :957-958 */
14.                 masterObjects[n].dynaNum = dynamicNum        /* :959 */
                }
            }
        }
    }

15. /* :964 HSetState -- no Go equivalent */
```

Points a porter will get wrong if not warned:

* **Step 6/7 run before step 8.** `dynamicNum` and `legit` are reset for *every* slot, including invalid
  ones. That is why the write-back (step 11) clears a stale `dynaNum` for a slot whose prize has just
  been collected: `IsThisValid` returns false, the switch is skipped, and `dynaNum` is set to −1.
* **Step 11 is outside the `IsThisValid` block** but **inside** the `for i` loop. It runs 24 times per
  room, i.e. 216 times per `DrawLocale`, each time scanning up to 216 master objects — an O(216 × 216)
  = 46 656-iteration inner loop per `DrawLocale`. Do not "optimise" it into a map without checking
  behaviour: the loop writes to **every** match, and there can be more than one when two local rooms
  are the same house room (possible if the house geometry double-references a room; not seen in
  shipped houses but not prevented).
* **Step 11's match is on `(objectNum, roomNum)`**, so it is *not* the aliasing bug of §7.7. The
  aliasing bug is only in the `masterObjects[i].hotNum` read at step 10-(E).
* **`thisObject` is a copy** (step 9), so all the `data.*` reads in the switch see the state as of the
  start of this slot's processing. Nothing in the switch writes back to the house.
* `legit` is used by exactly the 7 savedMap-backed groups and is otherwise dead.
* The C `switch` falls out of `case kObjectIsEmpty:` immediately (`:53-54`) even though
  `IsThisValid` has already filtered `what == -1`. Dead code; keep or drop.

### 12.1 The whole-`DrawLocale` sequence a port must reproduce

```
DrawLocale():                                            /* RoomGraphics.c:44-130 */
 1. ZeroFlamesAndTheLike()          /* :51  flames, tikiFlames, bbqCoals, pendulums, theStars */
 2. ZeroDinahs()                    /* :52  numDynamics = 0 */
 3. KillAllBands()                  /* :53 */
 4. ZeroMirrorRegion()              /* :54 */
 5. ZeroTriggers()                  /* :55 */
 6. numTempManholes = 0             /* :56 */
 7. FlushAnyTriggerPlaying()        /* :57 */
 8. DumpTriggerSound()              /* :58 */
 9. tvInRoom = false                /* :59 */
10. tvWithMovieNumber = -1          /* :60 */
11. roomV = <floor of thisRoom>     /* :64 */
12. for i := 0..8:                                            /* :67-71 */
        localNumbers[i]  = GetNeighborRoomNumber(i)
        isStructure[i]   = IsRoomAStructure(localNumbers[i])   /* OOB when -1 */
13. ListAllLocalObjects()           /* :72  fills masterObjects[], hotSpots[] */
14. save GWorld; SetGWorld(backSrcMap); PaintRect(backSrcRect)  /* :74-76 */
15. if numNeighbors > 3 {                                      /* :79 */
        numLights = GetNumberOfLights(localNumbers[kNorthWestRoom]);  DrawRoomBackground(NW); DrawARoomsObjects(NW, false)
        numLights = GetNumberOfLights(localNumbers[kNorthEastRoom]);  DrawRoomBackground(NE); DrawARoomsObjects(NE, false)
        numLights = GetNumberOfLights(localNumbers[kNorthRoom]);      DrawRoomBackground(N);  DrawARoomsObjects(N,  false)
        numLights = GetNumberOfLights(localNumbers[kSouthWestRoom]);  DrawRoomBackground(SW); DrawARoomsObjects(SW, false)
        numLights = GetNumberOfLights(localNumbers[kSouthEastRoom]);  DrawRoomBackground(SE); DrawARoomsObjects(SE, false)
        numLights = GetNumberOfLights(localNumbers[kSouthRoom]);      DrawRoomBackground(S);  DrawARoomsObjects(S,  false)
    }
16. if numNeighbors > 1 {                                      /* :106 */
        numLights = GetNumberOfLights(localNumbers[kWestRoom]);  DrawRoomBackground(W); DrawARoomsObjects(W, false); DrawLighting()
        numLights = GetNumberOfLights(localNumbers[kEastRoom]);  DrawRoomBackground(E); DrawARoomsObjects(E, false); DrawLighting()
    }
17. numLights = GetNumberOfLights(localNumbers[kCentralRoom])   /* :118 */
    DrawRoomBackground(kCentralRoom)                           /* :118-119 */
    DrawARoomsObjects(kCentralRoom, false)                     /* :120 */
    DrawLighting()                                             /* :121  no-op */
18. if numNeighbors > 3 { DrawFloorSupport() }                 /* :123-124 */
19. RestoreWorkMap()                /* :125  backSrcMap -> workSrcMap, whole rect */
20. shadowVisible = IsShadowVisible()                          /* :126 */
21. takingTheStairs = false                                    /* :127 */
22. restore GWorld                                             /* :129 */
```

Steps 15–17 are the answer to question (b). Note that `DrawLighting()` is called after W, E and Central
but *not* after the six diagonal/vertical rooms, and that it is a **no-op**
(`GliderPRO/Sources/RoomGraphics.c:422-430`):

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

The asymmetry is therefore invisible; a port should keep the call sites as comments so the structure
stays diffable. Similarly `RestoreWorkMap` (`RoomGraphics.c:389-398`) declares `Rect dest;` and assigns
`dest = backSrcRect;` but never uses it — the `CopyBits` at `:395-397` passes `&backSrcRect` for both
source and destination. Dead local; do not port it.

---

## 13. Corrections to existing documents in `docs/analysis/`

Flagged here rather than edited, per the "append only" rule for other documents.

### 13.1 `docs/analysis/rendering.md:2188` — `kVertLocalOffset` is 322, not 366

That line states `kVertLocalOffset` = `kTileHigh + kFloorSupportTall` = 322 + 44 = 366. The header
says:

```
#define kVertLocalOffset        322     // kTileHigh - 39 (was 283, then 295)
```

(`GliderPRO/Headers/GliderDefines.h:501`, with `kTileHigh` = 322 at `:498` and `kFloorSupportTall` = 44
at `:500`.) The value is **322**, numerically equal to `kTileHigh`, and the trailing comment is stale
history from when the room was 283 then 295 pixels tall.

This matters: `OffsetRectRoomRelative` (`GliderPRO/Sources/ObjectRects.c:1093-1131`) shifts the
north/south rooms by exactly `kVertLocalOffset`, so adjacent rooms tile **flush** with no gutter. With
366 the rooms would be 44 px apart and every seam in every `backSrcMap` composite would be wrong. The
44 px of `kFloorSupportTall` is the *overhang* that `DrawFloorSupport` paints **into** the neighbouring
room's band, not a gap between rooms.

### 13.2 `docs/analysis/rendering.md:2192-2193` — `DrawTable` and `DrawDeckTable` do **not** get `VerticalRoomOffset`

Those lines state that "`DrawTiki`, `DrawTable`, `DrawDeckTable`, `DrawStool`, `DrawMailboxLeft`,
`DrawMailboxRight` all take `playOriginV + VerticalRoomOffset(neighbor)`". Two of the six do not:

| Helper | Actual argument | Line |
|---|---|---|
| `DrawTiki` | `playOriginV + VerticalRoomOffset(neighbor)` | `ObjectDrawAll.c:177` |
| **`DrawTable`** | **`playOriginV`** | `ObjectDrawAll.c:208` |
| **`DrawDeckTable`** | **`playOriginV`** | `ObjectDrawAll.c:259` |
| `DrawStool` | `playOriginV + VerticalRoomOffset(neighbor)` | `ObjectDrawAll.c:266` |
| `DrawMailboxLeft` | `playOriginV + VerticalRoomOffset(neighbor)` | `ObjectDrawAll.c:487` |
| `DrawMailboxRight` | `playOriginV + VerticalRoomOffset(neighbor)` | `ObjectDrawAll.c:493` |

The four other furniture helpers that also paint drop shadows — `DrawShelf` (called at `:215`),
`DrawCabinet` (`:222`), `DrawCounter` (`:245`) and `DrawDresser` (`:252`) — take **no origin argument
at all**: their signatures are `void DrawShelf (Rect *shelfTop)` (`ObjectDraw.c:263`),
`void DrawCabinet (Rect *cabinet)` (`:358`), `void DrawCounter (Rect *counter)` (`:498`) and
`void DrawDresser (Rect *dresser)` (`:643`), and `grep -n 'playOrigin' ObjectDraw.c` returns **nothing**,
so they derive every coordinate from the rect they are handed. Because
`OffsetRectRoomRelative` (`ObjectRects.c:1093-1131`) has already applied
`playOrigin + VerticalRoomOffset` to that rect, those four are correct in all nine room slots.

The correct statement is therefore: **exactly four helpers receive the room-corrected vertical origin
as a scalar** (`DrawTiki` `:177`, `DrawStool` `:266`, `DrawMailboxLeft` `:487`, `DrawMailboxRight`
`:493`); `DrawTable` `:208` and `DrawDeckTable` `:259` receive the uncorrected `playOriginV`, which is
the drop-shadow/pedestal defect documented in §5.2 and §10.6 of this document; and the remaining
furniture helpers receive no origin at all and are unaffected. Reproduce the uncorrected calls.

### 13.3 `docs/analysis/determinism.md` — `ObjectDrawAll.c` line numbers are off by 1–3

`determinism.md` cites `ObjectDrawAll.c` line numbers that do not match a `tr '\r' '\n'` conversion of
the released source. Mapping:

| Cited in determinism.md | Actual (CR→LF copy) | What is there |
|---|---|---|
| `:22-966` | `:23-965` | the function |
| `:30-31` | `:33-34` | `if (localNumbers[neighbor] == kRoomIsEmpty) return;` |
| `:33-34` | `:36-37` | `testRect = houseRect; ZeroRectCorner(&testRect);` |
| `:35` | `:38` | `isLit = (numLights > 0);` |
| `:41` | `:43` | `for (i = 0; i < kMaxRoomObs; i++)` |
| `:43-44` | `:45-46` | `dynamicNum = -1; legit = -1;` |
| `:46` | `:48` | `if (IsThisValid(...))` |
| `:944` | `:953` | `if (!redraw)` (write-back) |
| `:945` | `:955` | `for (n = 0; n < numMasterObjects; n++)` |
| `:963` | `:964` | `HSetState` |
| `AddCandleFlame` `:80`, `:91` | `:83`, `:98` | `kTaper` central / proximity registration |
| `kCandle` `:112`, `:123` | `:117`, `:132` | |
| `kStubby` `:143`, `:154` | `:151`, `:166` | |
| `kTiki` `:168` | `:181` | |
| `kBBQ` `:181` | `:195` | |

`rendering.md`'s `ObjectDrawAll.c` citations match the CR→LF conversion exactly, so the conversion used
in this document is the correct baseline and `determinism.md` is the outlier. Likely cause: an earlier
pass read the file through a converter that collapsed the blank line after each `break;`.

### 13.4 `docs/analysis/determinism.md` §3.3 — the coupling claim is right but incomplete

§3.3 asserts that draw order determines simulation order. Confirmed (§7.4). What it omits, and what a
Go port must know:

1. The order is **(room-draw-rank, slot index)** where room-draw-rank is NW=0, NE=1, N=2, SW=3, SE=4,
   S=5, W=6, E=7, Central=8 — *not* the `masterObjects[]` order, which is Central, E, W, N, NE, SE, S,
   SW, NW (`GliderPRO/Sources/Objects.c:312-327`).
2. The *set* of registered objects, and therefore every `dinahs[]` index, depends on
   `numNeighbors` ∈ {1, 3, 9} and on screen resolution, because the 7 appliance types register from
   any room subject only to `SectRect(itsRect, testRect)` (§7.9). A replay is invalid across a
   resolution change.
3. Consequently the count of `RandomInt` calls made during `DrawLocale` is also resolution-dependent
   (`Dynamics3.c:208` for `kSparkle`; plus `DynamicMaps.c:338`, `:422`, `:508`, `:594`, `:685` for
   flames, tikis, coals, pendulums and stars).

### 13.5 Suggested cross-reference for `rendering.md` §7

`rendering.md` §7 (line 2126) covers the rect pipeline, the clipping test, `IsThisValid`, the `srcRects`
table, sheet mapping, the dispatcher table, the `isLit` gating, the `masterObjects` write-back and the
play-vs-edit split. This document is the exhaustive treatment of the same function; §7.6's dispatcher
table is a summary of §10.2 here. A one-line pointer appended as a new `### 7.10` before `## 8`
(line 2695) would close the loop without restructuring that document.

---

## Open questions

1. **Why is `kBBQ`'s coal registration not central-gated or proximity-tested when
   `kTaper`/`kCandle`/`kStubby` are?** (`ObjectDrawAll.c:192-196` vs `:78-100`.) The three candle
   types received an explicit "is this flame going to appear inside the central viewport?" test;
   `kTiki` and `kBBQ` did not. Nothing in the source explains whether this is deliberate (coals are
   short and rarely near a seam) or an unfinished edit. **Recommendation:** reproduce as-is.
2. **Why does `kTiki` have no `SectRect` guard at all?** It is the only group that *registers* an
   animated element unguarded (§10.6 lists the other 13 unguarded groups, none of which registers).
   Because
   `DrawTiki` also takes `playOriginV + VerticalRoomOffset(neighbor)` — the corrected origin — it looks
   like the tiki case was written or revised separately from its neighbours.
3. **Is the `masterObjects[i].hotNum` read at `:518`/`:531`/`:544`/`:557`/`:570`/`:574` a bug or an
   exploited invariant?** It is correct for the central room and aliases the central room's slot `i`
   for the other eight. Since switches in neighbour rooms are drawn but not interactive (their hot
   spots are never created — `Objects.c:283`), the wrong `dynaNum` on a neighbour switch may be
   harmless in every reachable case. Not proven.
4. **What is `dynaType.position` for?** It is a `short` at offset 28
   (`GliderPRO/Headers/GliderStructs.h:317`). `AddDynamicObject` sets it for some arms; `HandleDynamics`
   reads it for the flying enemies. Whether any type uses it as a *sub-pixel* accumulator or purely as
   a frame counter was not established in this pass.
5. **Answered for `kMaxDynamicObs`, still open for third-party content.** All 22 shipped houses were
   swept at eight screen sizes from 512 × 342 to the maximum 1536 × 1046 view (§9.9.5): `dinahs[]`
   reaches exactly 18/18 in three rooms (`California or Bust!` room 10, `Teddy World` rooms 244 and
   379) and never exceeds it, while `savedMaps[]`, `grease[]`, `flames[]` and `pendulums[]` **do**
   overflow — in 46 rooms across nine houses at the maximum view, and in 9 rooms as early as
   1024 × 768. What remains open is (a) whether any third-party house in circulation overflows
   `dinahs[]` — the format is open, the editor ships, and the editor's guard is per-room rather than
   per-neighbourhood (§9.8) — and (b) whether the sweep's remaining over-counts (the un-modelled
   `AddCandleFlame`/`AddTikiFlame`/`AddBBQCoals`/`AddPendulum` position gates, the nested
   `legit != -1` conditions, and the unverified `hotSpots[]` heuristic listed in §9.9.2) hide any
   *further* overflow that the model reports as merely at-cap.
6. **`ZeroDinahs` does not reset `byte1` or `moving`** (`Dynamics3.c:160-180`). Today every
   `AddDynamicObject` arm assigns both, so the omission is masked. Was that verified by the author or
   is it luck? A port that adds a dynamic type must set both explicitly.
7. **Why is `DrawLighting()` called after W, E and Central but not after the six other rooms?** The
   function is a no-op in the shipped source (`RoomGraphics.c:422-430`), so the pattern is a fossil of
   an abandoned lighting pass. What it used to do is unrecoverable from this source drop.
8. **Is the one-frame flicker over a movie TV (§8.9) actually observable?** The reasoning is sound —
   `workSrcMap` holds the stale static CRT pixels and any renderer overlapping the TV queues a
   work→main rect — but it was not confirmed against a running binary.
9. **`kWallWindow` is not `isLit`-gated** (`:929-933` — only `SectRect` guards the draw). Because `GetNumberOfLights` counts `kWallWindow` as
   a light source (`Room.c:970-1095`), the ungated path may be unreachable. Not proven for the case
   where the room's background is on the "always lit" whitelist and `numLights` is computed by a
   different branch.

## Porting notes

Ordered by the amount of damage getting it wrong does.

1. **Draw order is load-bearing, twice over.** Within a room it is ascending slot index 0..23
   (`ObjectDrawAll.c:43`), which is simultaneously the file storage order and the painter's-algorithm
   z-order. Across rooms it is NW, NE, N, SW, SE, S, W, E, **Central last**
   (`RoomGraphics.c:80-120`). Getting either wrong changes both the visible overlaps and the
   `dinahs[]` index assignment, and the `dinahs[]` index is the identity used by every switch, trigger
   and save. Do not sort, do not parallelise, do not use a map anywhere in this path.
2. **`dinahs[]` indices are display-dependent.** `numDynamics`, every `dinahs[]` index, and
   `tvWithMovieNumber` depend on the screen size (via `houseRect` → `testRect` → the appliance
   `SectRect` tests) and on `numNeighbors` (via the `> 3` and `> 1` gates). Anything persisted or
   transmitted must carry the resolution and neighbour count, or be re-derived. A replay recorded at
   640×480 desyncs at 1024×768.
3. **`Point` is `{short v; short h;}`.** v first. Every `topLeft` in the house file. `Rect` is
   `{top, left, bottom, right}`. Big-endian on disk.
4. **The `objectType` union means `state` lives at two different offsets** — byte 10 for prizes and
   grease (`bonusType`), byte 11 for lights, appliances and enemies (`lightType`, `applianceType`,
   `enemyType`). A flattened struct with a single `state` field will silently read the wrong byte for
   half the types. Model the union, or generate per-arm accessors.
5. **`SectRect` zeroes its output rect on the empty case**, and `whoCares` is reused as a scratch
   variable throughout — including at `:683` where it is *assigned* `tvScreen1`. A Go
   `func sectRect(a, b Rect) (Rect, bool)` must return the zero `Rect` when `ok == false`.
6. **Bounds-check every −1.** The 25 call sites listed in §9.10 pass `masterObjects[].dynaNum` or
   `.hotNum` unchecked into `dinahs[]`, `hotSpots[]` and `grease[]`. In C these are silent heap
   corruption; in Go they are panics. Add the checks — and note that shipped houses **do** reach the
   producing overflows (§9.9), so this is not defensive programming, it is required. The five
   `DynamicMaps.c` `AddXxx` helpers are the one family that already checks correctly (§9.5); everything
   in `Interactions.c` and `Triggers.c` does not.
7. **Reproduce the budget-exhaustion *semantics*, not the crashes.** `AddDynamicObject` → −1 with the
   art still drawn; `AddActiveRect` → −1 with the art still drawn; `BackUpToSavedMap` → −1 **with the
   art suppressed**; `AddGrease` → −1 with the art suppressed. The third one is the surprising one, and
   shipped houses **do** rely on it: at 1024 × 768 with nine neighbours, `savedMaps[]` overflows in
   `In The Mirror` rooms 76 and 85, `Slumberland` room 177, `Teddy World` rooms 321 and 322 and
   `Titanic` room 5, and `grease[]` overflows in `Teddy World` rooms 321 and 322 — so those rooms
   have invisible-but-collectable prizes and invisible-but-slippery grease patches in the original
   (§9.9.3). **Do not raise the caps** to "fix" this; it changes the picture.
8. **Do not model a current-port global.** Pass the destination surface explicitly (§11.3). The
   original leaks `workSrcMap` as the current port out of `DrawRoomBackground`
   (`RoomGraphics.c:238`) and gets away with it only because every helper names its destination.
9. **Reproduce the four known geometry defects**, because houses were authored against them:
   * exactly two calls pass an un-corrected vertical origin — `DrawTable(&itsRect, playOriginV)` at
     `:208` and `DrawDeckTable(&itsRect, playOriginV)` at `:259` — instead of
     `playOriginV + VerticalRoomOffset(neighbor)`, which `DrawTiki` `:177`, `DrawStool` `:266`,
     `DrawMailboxLeft` `:487` and `DrawMailboxRight` `:493` all use. (`DrawShelf`, `DrawCabinet`,
     `DrawCounter` and `DrawDresser` take no origin argument at all — an earlier draft of this document
     wrongly listed them here.) All 200 `kTable`/`kDeckTable` objects in the shipped corpus are
     affected — 100 % of them, because every one lives in a vertically-connected room; §5.2 gives the
     exact visible symptom per room slot;
   * `kTiki` (`:173-183`) has no `SectRect` guard on either its draw or its `AddTikiFlame` registration;
   * `kMailboxLf` (`:484-488`) and `kMailboxRt` (`:490-494`) have no `SectRect` guard;
   * `kToaster` has no `isLit` gate (`:647`).
   And five furniture types have no `SectRect` guard for their *draw*: `kTable` `:207`, `kShelf` `:214`,
   `kDresser` `:251`, `kDeckTable` `:258`, `kStool` `:265` test only `isLit` (§10.6). In total **23 of
   the 74 `case` groups never call `SectRect`**, of which **14 do real work** unguarded.
10. **`AddTempManholeRect` at `:277` has no `redraw` guard** while `numTempManholes` is only reset in
    `DrawLocale` (`RoomGraphics.c:56`). Every `RedrawRoomLighting` in a manhole room therefore appends
    duplicate rects until the 8-slot cap. Harmless visually (the same rect is painted twice) but it
    silently consumes the budget; a room with 3 manholes seals its holes after the second light-switch
    flip. Decide and document.
11. **`masterObjects[]` is exactly 216 = 9 × 24 with zero slack.** Do not add pseudo-rooms.
12. **The write-back loop is O(24 × numMasterObjects) per room**, 46 656 iterations per `DrawLocale`
    at full occupancy. It is not a hot path (once per room entry) but do not "optimise" it into a
    single-match map lookup: it writes to *every* `(objectNum, roomNum)` match.
13. **`numLights` is a global mutated between calls.** `isLit` is captured once per call at `:38`.
    Pass it in, or reproduce the global; do not recompute it per object.
14. **`GetNumberOfLights(-1)` and `IsRoomAStructure(-1)` are reachable** whenever a neighbour is
    absent (`IsRoomAStructure` at `RoomGraphics.c:70`, `GetNumberOfLights` at `:80`, `:84`, `:88` … `:118`). In C they read `rooms[-1]` and return garbage that is
    harmlessly discarded; in Go they panic. Guard them and return 0 / false, which matches the
    effective behaviour.
15. **QuickTime is optional and separable.** Everything QuickTime does lives in two `#ifdef COMPILEQT`
    blocks in this function (`:679-695`, `:703-710`) plus the sites in §8.5. A first Go port can omit
    it entirely and get the correct static CRT image for every TV — provided it also omits the
    `tvInRoom` latch, or the latch will suppress a static blit for a TV that has no movie.
16. **Keep the case grouping.** The 74 bodies map to 118 types; several helpers
    (`DrawSimpleBlowers`, `DrawSimplePrizes`, `DrawSimpleAppliance`, `DrawSimpleFurniture`,
    `DrawSimpleLight`, `DrawSimpleClutter`, `DrawSimpleTransport`) dispatch internally on `what` and
    index sheet rects by type. Splitting the groups per type without also splitting those helpers
    duplicates the sheet-offset tables and invites drift.
