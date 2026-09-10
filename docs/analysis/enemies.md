# Glider PRO 1.0.4 — Enemies and Hazards

## Scope

This document specifies, exhaustively, every hostile or damaging entity in Glider PRO
1.0.4 and every code path by which such an entity can affect the player. It covers:

* The **nine "enemy" object classes** (`kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`,
  `kDartRt`, `kBall`, `kDrip`, `kFish`, `kCobweb`).
* The **appliance hazards** (`kShredder`, `kToaster`/flying toast, `kOutlet`/electric
  arc, `kMicrowave`, `kGuitar`, and the plain solid-body appliances that kill on touch).
* The **five open-flame blowers** (`kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ`) and
  the invisible blower `kInvisBlower`, plus the two **fans** (`kLeftFan`, `kRightFan`)
  whose bodies are lethal.
* The **dynamic-object engine** (`dinahs[]` / `dynaType`) that drives all moving
  hazards, the **hot-spot engine** (`hotSpots[]` / `hotObject`) that drives all static
  hazard volumes, and the exact per-frame ordering that couples them.
* Player-side consequences: instant death (`StartGliderFadingOut`), burning
  (`FlagGliderBurning`), shredding (`FlagGliderShredding`), webbing (`WebGlider`),
  foil absorption, shoving, pushing, and resource draining (microwave).
* The **rubber band** weapon, because it is the *only* mechanism in the game that
  destroys a hazard.
* Spawn rules, per-room instance limits, all animation frame counts and rates, all
  velocity/acceleration constants, all sounds and their priorities.

Out of scope (documented elsewhere): prizes/bonuses, transports (stairs, doors, ducts,
mailboxes), lights and switches *as such* (only their effect on hazards is covered
here), room/house geometry and scrolling, the scoreboard, and the QuickTime TV movie.

## Sources read

All paths are relative to the repository root. Line numbers throughout this
document refer to a **CR→LF converted** copy of each file (the shipped sources are
classic-Mac CR-only text); the conversion used was
`tr '\r' '\n' < GliderPRO/Sources/X.c > /tmp/wf-enemies/X.c`.

| File | Converted lines | Why it was read |
|---|---:|---|
| `GliderPRO/Headers/GliderDefines.h` | 625 | every constant, object code, sound ID, priority, limit |
| `GliderPRO/Headers/GliderStructs.h` | 347 | `objectType` union, `dynaType`, `hotObject`, `roomType`, `houseType` |
| `GliderPRO/Sources/Dynamics.c` | 776 | collision against dynamics, band-hit test, toast, outlet, sparkle object, all appliance dynamics, all `Render*` for dynamics |
| `GliderPRO/Sources/Dynamics2.c` | 589 | `HandleBalloon`, `HandleCopter`, `HandleDart`, `HandleBall`, `HandleDrip`, `HandleFish` |
| `GliderPRO/Sources/Dynamics3.c` | 555 | `HandleDynamics`, `RenderDynamics`, `ZeroDinahs`, `AddDynamicObject` (all spawn maths) |
| `GliderPRO/Sources/Interactions.c` | 1777 | `SectGlider`, `GliderInRect`, `GliderHitTop`, `BounceGlider`, `HandleSwitches`, `HandleMicrowaveAction`, `HandleHotSpotCollision`, `CheckForHotSpots`, `FlagStillOvers`, `WebGlider` |
| `GliderPRO/Sources/Objects.c` | 1001 | `IsThisValid`, `ListOneRoomsObjects`, `ListAllLocalObjects`, `SetObjectState`, `GetObjectState` |
| `GliderPRO/Sources/ObjectRects.c` | 1188 | `GetObjectRect`, `AddActiveRect`, `CreateActiveRects` (all hazard hot-spot geometry), `OffsetRectRoomRelative` |
| `GliderPRO/Sources/ObjectDrawAll.c` | 966 | `DrawARoomsObjects` — which hazards become dynamics, in which of the 9 neighbour rooms |
| `GliderPRO/Sources/ObjectDraw2.c` | 1437 | `DrawBalloon`, `DrawCopter`, `DrawDart`, `DrawBall`, `DrawDrip`, `DrawFish`, `DrawOutlet`, `DrawMicrowave` |
| `GliderPRO/Sources/ObjectAdd.c` | 1084 | editor defaults for every hazard, `HowManyDynamicObjects`, `HowManyShredderObjects` |
| `GliderPRO/Sources/ObjectInfo.c` | 2567 | which hazard fields are user-editable and their validated ranges |
| `GliderPRO/Sources/ObjectEdit.c` | — | which drag handle writes `data.g.height` vs `data.h.length` |
| `GliderPRO/Sources/HouseLegal.c` | 1219 | clamping of hazard `length`/`height`, even/odd `topLeft.h` forcing |
| `GliderPRO/Sources/Play.c` | 821 | `PlayGame` frame order, `SetObjectsToDefaults`, `StrikeChime` |
| `GliderPRO/Sources/Player.c` | 1605 | `MoveGliderBurning`, `MoveGliderShredding`, `OffAMortal`, gravity constants |
| `GliderPRO/Sources/Modes.c` | 640 | `FlagGliderShredding`, `FlagGliderBurning`, `StartGliderFadingOut`, `StartGliderFoilLosing` |
| `GliderPRO/Sources/Render.c` | 772 | `RenderFrame` ordering, `RenderFlames`, `RenderPendulums`, `RenderSparkles`, `RenderShreds`, `RenderBands` |
| `GliderPRO/Sources/DynamicMaps.c` | 798 | `AddSparkle`, `AddCandleFlame`, `AddTikiFlame`, `AddBBQCoals`, `BackUpFlames`, `AddAShreddedGlider`, `RemoveShreds` |
| `GliderPRO/Sources/RubberBands.c` | 318 | `AddBand`, `HandleBands`, `CheckBandCollision`, `KillBand` |
| `GliderPRO/Sources/Trip.c` | 245 | every `Toggle*` and `Trigger*` entry point for hazards |
| `GliderPRO/Sources/Triggers.c` | 205 | `ArmTrigger`, `HandleTriggers`, `FireTrigger` |
| `GliderPRO/Sources/StructuresInit.c` | 724 | `InitBlowers`, `InitAppliances`, `InitEnemies` — every sprite-sheet source rect |
| `GliderPRO/Sources/StructuresInit2.c` | 476 | `InitSrcRects` — the 144-entry `srcRects[]` table |
| `GliderPRO/Sources/Grease.c` | 302 | `SpillGrease` (reachable from a rubber band and from a trigger) |
| `GliderPRO/Sources/Sound.c` | 535 | `PlayPrioritySound` arbitration, `'snd '` resource numbering |
| `GliderPRO/Glider PRO.r` | 199843 | parsed with python3 for `'snd '` and `PICT` resource IDs and `picFrame` dimensions |
| `GliderPRO/Houses/*.binhex` | 22 files | BinHex-decoded and parsed with python3 for an empirical census of hazard field values |

---

## 1. Architecture: how a hazard gets from disk to the screen

Glider PRO keeps **four parallel representations** of the objects in and around the
room the player is currently in. Every hazard lives in one, two, or three of them.

### 1.1 The four tables

| Table | Type | Capacity | Coordinate space | Built by | Reset by |
|---|---|---:|---|---|---|
| `(*thisHouse)->rooms[r].objects[0..23]` | `objectType` (12 B) | 24 per room | room-local, unscrolled | loaded from disk | `SetObjectsToDefaults` (`GliderPRO/Sources/Play.c:603`) |
| `masterObjects[0..215]` | `objDataType` | `kMaxMasterObjects` = 216 = 24×9 | n/a (bookkeeping) | `ListAllLocalObjects` (`GliderPRO/Sources/Objects.c:300`) | same call (`numMasterObjects = 0`) |
| `hotSpots[0..55]` | `hotObject` | `kMaxHotSpots` = 56 | **room-local** (no `playOrigin`) | `CreateActiveRects` (`GliderPRO/Sources/ObjectRects.c:296`) via `AddActiveRect` (`:277`) | `nHotSpots = 0` in `ListAllLocalObjects` (`GliderPRO/Sources/Objects.c:307`) |
| `dinahs[0..17]` | `dynaType` | `kMaxDynamicObs` = 18 | **room-local** (`playOrigin` subtracted at add time) | `AddDynamicObject` (`GliderPRO/Sources/Dynamics3.c:187`) from `DrawARoomsObjects` | `ZeroDinahs` (`GliderPRO/Sources/Dynamics3.c:160`) |

`masterObjects` covers the **3×3 neighbourhood** of rooms (`kCentralRoom` 0,
`kNorthRoom` 1, `kNorthEastRoom` 2, `kEastRoom` 3, `kSouthEastRoom` 4, `kSouthRoom` 5,
`kSouthWestRoom` 6, `kWestRoom` 7, `kNorthWestRoom` 8 — `GliderPRO/Headers/GliderDefines.h:217-225`),
listed central-first then E/W then the remaining six
(`GliderPRO/Sources/Objects.c:312-328`).

`objDataType` (`GliderPRO/Headers/GliderStructs.h:322-332`) is the glue:

```c
typedef struct {
    short  roomNum;    // room # object in (real number)
    short  objectNum;  // obj. # in house (real number)
    short  roomLink;   // room # object linked to (if any)
    short  objectLink; // obj. # object linked to (if any)
    short  localLink;  // index in master list if exists
    short  hotNum;     // index into active rects (if any)
    short  dynaNum;    // index into dinahs (if any)
    objectType theObject;
} objDataType;
```

`hotNum` is assigned **only for the central room** and only when `IsThisValid()`
returns true (`GliderPRO/Sources/Objects.c:283-286`). `dynaNum` starts at `-1`
(`:287`) and is filled in afterwards by a linear back-search in `DrawARoomsObjects`
(`GliderPRO/Sources/ObjectDrawAll.c:953-960`).

`IsThisValid(where, who)` (`GliderPRO/Sources/Objects.c:89-122`) returns `false` only
for `kObjectIsEmpty` (-1) and for the 12 prize types whose `data.c.state` is false. **No
hazard type is ever suppressed by `IsThisValid`** — hazard hot spots are always created
for the central room.

> **Gotcha worth flagging for a port.** `ListOneRoomsObjects` calls
> `CreateActiveRects(n)` passing `n`, the object index *within the room*
> (`GliderPRO/Sources/Objects.c:284`), but `CreateActiveRects` immediately does
> `theObject = masterObjects[who].theObject;` (`GliderPRO/Sources/ObjectRects.c:304`),
> i.e. it treats its argument as a **masterObjects index**. This is only correct
> because `ListAllLocalObjects` lists `kCentralRoom` first
> (`GliderPRO/Sources/Objects.c:312`), so central-room object `n` is exactly
> `masterObjects[n]`. The same value is stored as `hotSpots[].who`, and
> `HandleSwitches`/`ArmTrigger` later index `masterObjects[who->who]`. A port must
> preserve the invariant "central room objects occupy masterObjects[0..23] in order" or
> re-plumb `who` explicitly.

### 1.2 Per-frame pipeline

`PlayGame` (`GliderPRO/Sources/Play.c:430`) drives the whole game. Every iteration:

```
 1. gameFrame++;                                     // Play.c:434
 2. evenFrame = !evenFrame;                           // Play.c:435
 3. (optionally pump events)
 4. HandleTelephone();                                // Play.c:445
 5. HandleDynamics();                                 // moves every dinah
 6. GetInput(&theGlider) [or GetDemoInput]
 7. HandleInteraction();                              // hot spots + room escape
 8. HandleTriggers();                                 // countdown timers
 9. HandleBands();                                    // moves/collides rubber bands
10. HandleGlider(&theGlider);                         // integrates player physics
11. (MoviesTask if a QT movie TV is on)
12. RenderFrame();                                    // composites and blits
13. HandleDynamicScoreboard();
```

(two-player branch `GliderPRO/Sources/Play.c:447-472`, one-player branch `:473-497`).

Ordering consequences a port must reproduce exactly:

* Hazards move **before** the player reads input, so a hazard's position when the
  player's collision is evaluated is this frame's position.
* **Player-vs-dynamic collision is detected inside `HandleDynamics`**, not in
  `HandleInteraction`. Each `Handle<Enemy>` calls `CheckDynamicCollision` itself.
* **Player-vs-static-hazard collision** happens in `HandleInteraction` →
  `CheckForHotSpots`.
* Rubber bands move after the hot spots are evaluated but the *hazard* checks
  `DidBandHitDynamic` at step 5, i.e. against the band position from the **previous**
  frame.

`RenderFrame` (`GliderPRO/Sources/Render.c:639-671`) draws in this fixed order:

```
 1. if (hasMirror) DrawReflection(&theGlider, true) [and theGlider2]
 2. HandleGrease();
 3. RenderPendulums();
 4. if (evenFrame) RenderFlames(); else RenderStars();   // Render.c:649-652
 5. RenderDynamics();          // toast, balloon, copter, dart, ball, drip, fish
 6. RenderFlyingPoints();
 7. RenderSparkles();
 8. RenderGlider(&theGlider, true) [and theGlider2]
 9. RenderShreds();
10. RenderBands();
11. while (TickCount() < nextFrame) { }                  // Render.c:662-664
12. nextFrame = TickCount() + kTicksPerFrame;            // Render.c:665
13. CopyRectsQD();
14. numWork2Main = 0; numBack2Work = 0;
```

`kTicksPerFrame` = **2** (`GliderPRO/Headers/GliderDefines.h:533`). A classic-Mac tick
is 1/60.15 s, so the target frame rate is **≈30.07 fps** and the frame budget is
≈33.26 ms. The wait is a **busy spin on `TickCount()`**.

### 1.3 `evenFrame` — the half-rate clock

`evenFrame` is a global `Boolean` flipped once per frame (`GliderPRO/Sources/Play.c:435`).
Hazards use it for two distinct purposes:

| Use | Effect |
|---|---|
| Animation gate | Balloon, toast: frame advances only on `evenFrame`, i.e. **15 fps animation**. |
| Gravity gate | Ball, drip, fish: `if (evenFrame) vVel++;` → **half gravity** compared with toast. |

**Three** sites **write** the global `evenFrame`, which is a genuine cross-system side
effect a port must replicate or consciously reject:

* `HandleBall` idle branch: `evenFrame = true;` (`GliderPRO/Sources/Dynamics2.c:420`).
* `AddDynamicObject` case `kBall`: `evenFrame = true;` (`GliderPRO/Sources/Dynamics3.c:474`)
  as part of the launch-velocity solve loop.
* `AddDynamicObject` case `kFish`: the same `evenFrame = true;`
  (`GliderPRO/Sources/Dynamics3.c:524`) before its identical solve loop.

In both `AddDynamicObject` cases the write is **vestigial** — the solve loop uses the
*local* `lilFrame` (declared at `GliderPRO/Sources/Dynamics3.c:191`), not `evenFrame`, so
the assignment has no effect on the solve and only leaks out to the renderer.

Because `PlayGame` flips `evenFrame` at the top of the next frame, a ball starting to
fall forces the *following* frame to be odd, which perturbs candle-flame vs star
rendering (`GliderPRO/Sources/Render.c:649-652`) and every other `evenFrame` consumer
for one frame.

---

## 2. On-disk data layout (empirically verified)

### 2.1 House file container

House files are the **data fork** of a Macintosh file of type `'gliH'`, creator
`'ozm5'`. The shipped houses in `GliderPRO/Houses/` are BinHex 4.0 (`*.binhex`); the
`*.mov` siblings are QuickTime movies used by `kTV` objects.

Verified by decoding all 22 shipped houses with a BinHex 4.0 decoder written for this
analysis and parsing the result with python3. Example (`Sampler.house`):

```
version   = 0x0200   (kHouseVersion)
timeStamp = ...
initial   = Point(v=85, h=384)
banner    = "Welcome to Omid's Happy Home."
trailer   = "Congratulations."
firstRoom = 1
nRooms    = 2
file size = 1564 = 866 + 348*2 + 2 trailing bytes
```

All multi-byte fields are **big-endian**; the C structs were read straight off disk on
a 68k/PPC Mac with no byte swapping.

### 2.2 `houseType` (`GliderPRO/Headers/GliderStructs.h:182-198`)

| Offset | Size | Field | Type |
|---:|---:|---|---|
| 0 | 2 | `version` | `short` (`0x0200` = `kHouseVersion`, `0x0300` = `kNewHouseVersion`) |
| 2 | 2 | `unusedShort` | `short` |
| 4 | 4 | `timeStamp` | `long` |
| 8 | 4 | `flags` | `long` (bit 0 = wardBit) |
| 12 | 4 | `initial` | `Point` (**v first, then h**) |
| 16 | 256 | `banner` | `Str255` |
| 272 | 256 | `trailer` | `Str255` |
| 528 | 292 | `highScores` | `scoresType` |
| 820 | 40 | `savedGame` | `gameType` |
| 860 | 1 | `hasGame` | `Boolean` |
| 861 | 1 | `unusedBoolean` | `Boolean` |
| 862 | 2 | `firstRoom` | `short` |
| 864 | 2 | `nRooms` | `short` |
| 866 | 348×`nRooms` | `rooms[]` | `roomType[]` |

### 2.3 `roomType` — 348 bytes (`GliderPRO/Headers/GliderStructs.h:166-180`)

| Offset | Size | Field |
|---:|---:|---|
| 0 | 28 | `Str27 name` |
| 28 | 2 | `short bounds` |
| 30 | 1 | `Byte leftStart` |
| 31 | 1 | `Byte rightStart` |
| 32 | 1 | `Byte unusedByte` |
| 33 | 1 | `Boolean visited` |
| 34 | 2 | `short background` |
| 36 | 16 | `short tiles[kNumTiles]` (`kNumTiles` = 8) |
| 52 | 2 | `short floor` |
| 54 | 2 | `short suite` |
| 56 | 2 | `short openings` |
| 58 | 2 | `short numObjects` |
| 60 | 288 | `objectType objects[24]` (`kMaxRoomObs` = 24) |

### 2.4 `objectType` — 12 bytes, and the union aliasing that matters for hazards

```c
typedef struct {
    short what;                    // 2   offset 0
    union {                        // 10  offset 2
        blowerType a; furnitureType b; bonusType c; transportType d;
        switchType e; lightType f; applianceType g; enemyType h; clutterType i;
    } data;
} objectType;                      // total 12
```
(`GliderPRO/Headers/GliderStructs.h:90-105`)

The three variants that hazards use:

```c
typedef struct { Point topLeft; short distance; Boolean initial;
                 Boolean state; Byte vector; Byte tall; } blowerType;   // 10
typedef struct { Point topLeft; short height; Byte byte0;
                 Byte delay; Boolean initial; Boolean state; } applianceType; // 10
typedef struct { Point topLeft; short length; Byte delay;
                 Byte byte0; Boolean initial; Boolean state; } enemyType;     // 10
```
(`GliderPRO/Headers/GliderStructs.h:11-19`, `:64-72`, `:74-82`)

Byte-exact overlay (offsets are **within the 12-byte `objectType`**):

| Off | Size | `blowerType a` | `applianceType g` | `enemyType h` |
|---:|---:|---|---|---|
| 0 | 2 | `what` | `what` | `what` |
| 2 | 2 | `topLeft.v` | `topLeft.v` | `topLeft.v` |
| 4 | 2 | `topLeft.h` | `topLeft.h` | `topLeft.h` |
| 6 | 2 | `distance` | `height` | `length` |
| 8 | 1 | `initial` | `byte0` | `delay` |
| 9 | 1 | `state` | `delay` | `byte0` |
| 10 | 1 | `vector` | `initial` | `initial` |
| 11 | 1 | `tall` | `state` | `state` |

Note `Point` is `{ short v; short h; }` — **v comes first** in QuickDraw. That is
verified empirically below.

**The critical aliasing facts for hazards:**

1. `data.g.height` and `data.h.length` are **the same two bytes** (payload offset 4,
   file offset 6). `AddDynamicObject` case `kFish` reads `who->data.g.height`
   (`GliderPRO/Sources/Dynamics3.c:531`) whereas the editor writes `data.h.length` for a
   fish (`GliderPRO/Sources/ObjectEdit.c:314-321`). Both refer to the same field; there
   is no bug, but a Go port with separate strongly-typed structs **must** unify them.
2. `data.g.byte0` (payload offset 6) and `data.h.delay` are the same byte. The
   microwave's kill-mask lives there (`GliderPRO/Sources/Interactions.c` `kills =
   (short)data.g.byte0`), so a microwave's "delay" in `enemyType` terms is really its
   kill mask.
3. `data.g.delay` (payload offset 7) and `data.h.byte0` are the same byte. The toaster
   reads `who->data.g.delay` for its cycle time; balloons/copters/darts/drips/fish read
   `who->data.h.delay`.

**Empirical confirmation.** A census of all 22 shipped houses (4070 rooms) shows
`kToaster` records always have payload offset 6 = 0 and offset 7 in 0..255, while
`kBalloon` records always have payload offset 6 in 0..255 and offset 7 = 0 — exactly
what the two struct layouts predict.

Two raw records, taken from `Demo House` (BinHex-decoded, data fork):

```
room  3 "Enemies"  obj 0  what=0x71 kBalloon  payload = 00 92 00 ce 00 00 03 00 01 01
                                             topLeft.v=0x0092=146  topLeft.h=0x00ce=206
                                             length=0  delay=3  byte0=0  initial=1  state=1
room  3 "Enemies"  obj 1  what=0x71 kBalloon  payload = 00 92 00 fe 00 00 07 00 01 01
                                             topLeft=(h=254,v=146) length=0 delay=7 ...
room  3 "Enemies"  obj 2  what=0x71 kBalloon  payload = 00 92 01 3e 00 00 00 00 01 01
                                             topLeft=(h=318,v=146) length=0 delay=0 ...
room 15 "Water"    obj 0  what=0x78 kFish     payload = 01 06 00 a8 00 fe 0a 00 01 01
                                             topLeft.v=0x0106=262 topLeft.h=0x00a8=168
                                             length=0x00fe=254  delay=10  byte0=0
                                             initial=1  state=1
```

The three balloons share `topLeft.v` = 146 and differ only in `topLeft.h` (206, 254,
318) and `delay` (3, 7, 0) — consistent with `HandleBalloon` ignoring `topLeft.v`
entirely (it always spawns at `kBalloonStart` = 310) and using `delay` as the respawn
period.

### 2.5 `dynaType` — the runtime record for every moving hazard

```c
typedef struct {
    Rect    dest;        // current on-screen rect, room-local
    Rect    whole;       // dirty union of previous+current, for the blitter
    short   hVel, vVel;
    short   type, count;
    short   frame, timer;
    short   position, room;
    Byte    byte0, byte1;
    Boolean moving, active;
} dynaType, *dynaPtr;
```
(`GliderPRO/Headers/GliderStructs.h:310-320`)

The fields are **heavily overloaded per `type`**. This table is the single most
important thing in this document for a porter:

| type | `hVel` | `vVel` | `count` | `frame` | `timer` | `position` | `byte0` | `byte1` | `moving` | `active` |
|---|---|---|---|---|---|---|---|---|---|---|
| `kSparkle` | — | — | — | sparkle-suppress counter | countdown to next sparkle | — | object index | 0 | — | on/off |
| `kToaster` | **clip line** = spawn `top+2` | current vertical velocity | launch speed (peak) | bread animation frame 0..5 / idle countdown | idle reload period (frames) | 0 | object index | 0 | in flight | on/off |
| `kMacPlus` | — | — | — | — | countdown | — | idx | 0 | — | on/off |
| `kTV` | — | — | — | — | countdown | — | idx | 0 | — | on/off |
| `kCoffee` | — | — | — | — | countdown | — | idx | 0 | — | on/off |
| `kOutlet` | **numLights** in room | — | zap period (frames) | arc frame 0..3 | countdown | **0 = idle, 1 = zapping** | idx | 0 | — | on/off |
| `kVCR` | — | — | — | 0/1 blink phase | countdown | — | idx | 0 | — | on/off |
| `kStereo` | — | — | — | — | countdown | — | idx | 0 | — | on/off |
| `kMicrowave` | — | — | — | — | countdown | — | idx | 0 | — | on/off |
| `kBalloon` | 0 | −2 rising / +8 popped | respawn period (frames) | 0..5 rising, 6..7 popped | respawn countdown | 0 | idx | 0 | in flight | on/off |
| `kCopterLf/Rt` | −1 / +1 | +2 descending / +8 shot | respawn period | 0..7 flying, 8..9 shot | respawn countdown | **spawn `dest.left`** | idx | 0 | in flight | on/off |
| `kDartLf/Rt` | −6 / +6 | +2 / +8 shot | respawn period | 0 or 2 flying, 1 or 3 shot | respawn countdown | **spawn `dest.top`** | idx | 0 | in flight | on/off |
| `kBall` | 0 | current velocity | **bounce impulse, negative** = `-velocity` | 0 airborne, 1 bouncing | 0 | **floor line** (`dest.bottom` at spawn) | idx | 0 | in flight | on/off |
| `kDrip` | **spawn `dest.top`** | current velocity | reload period | 0,1,2 forming; 3 idle; 4/5 falling | countdown | **splash line** = spawn `top + length` | idx | 0 | in flight | on/off |
| `kFish` | **reload period** | current velocity | **jump impulse, negative** = `-velocity` | 0..3 idle bob, 4..7 in air | countdown | **water line** (`dest.bottom` at spawn) | idx | 0 | in flight | on/off |

Sign conventions for `count`, which differ between hazards and are a classic porting
trap:

* **Toast**: `count = velocity` — **positive** (`GliderPRO/Sources/Dynamics3.c:233`).
  Launch does `vVel = -count` (`GliderPRO/Sources/Dynamics.c:375`,
  `GliderPRO/Sources/Trip.c:159`); landing test is `if (vVel > count)`
  (`GliderPRO/Sources/Dynamics.c:357`).
* **Ball**: `count = -velocity` — **negative** (`GliderPRO/Sources/Dynamics3.c:486`).
  Bounce does `vVel = count` directly (`GliderPRO/Sources/Dynamics2.c:389`).
* **Fish**: `count = -velocity` — **negative** (`GliderPRO/Sources/Dynamics3.c:535`).
  Re-entry does `vVel = count` (`GliderPRO/Sources/Dynamics2.c:538`).
* **Balloon / copter / dart / drip / outlet**: `count` is a *frame count*, always
  `(delay * 6) / kTicksPerFrame` = `delay * 3`, and is copied into `timer` on respawn.

`room` holds the real room number the object belongs to and is used only by
`UpdateOutletsLighting` (`GliderPRO/Sources/Trip.c:239-243`). **`AddDynamicObject`'s
`kDartLf`/`kDartRt` case never assigns `room`** (`GliderPRO/Sources/Dynamics3.c:437-463`),
so a dart's `room` is always **0**: `ZeroDinahs` explicitly sets `room = 0`
(`GliderPRO/Sources/Dynamics3.c:175`) and it runs at the top of every `DrawLocale`
(`GliderPRO/Sources/RoomGraphics.c:52`), before any `AddDynamicObject` call, and each
slot is written at most once per locale draw. This is harmless in practice because only
outlets consult `room`, and `UpdateOutletsLighting` also filters on `type == kOutlet`.

`ZeroDinahs` (`GliderPRO/Sources/Dynamics3.c:160-180`) sets, for all 18 slots:
`type = kObjectIsEmpty` (−1), `dest` and `whole` to (0,0,0,0), `hVel = vVel = 0`,
`count = 0`, `frame = 0`, `timer = 0`, `position = 0`, `room = 0`, `byte0 = 0`,
`active = false`, and finally `numDynamics = 0`. It does **not** clear `moving` or
`byte1`. Every `AddDynamicObject` case explicitly writes `byte1 = 0`, and **all** the
hazard cases write `moving = false` — balloon `:408`, copter `:433` (shared by
`kCopterLf`/`kCopterRt`), dart `:461`, ball `:485`, drip `:512`, fish `:542` — so no case
relies on `moving` already being false. See §16.

### 2.6 `hotObject` — the runtime record for every static hazard volume

```c
typedef struct {
    Rect    bounds;
    short   action;        // one of the 28 k*It codes
    short   who;           // index into masterObjects
    Boolean isOn, stillOver;
    Boolean doScrutinize;
} hotObject, *hotPtr;
```
(`GliderPRO/Headers/GliderStructs.h:218-225`)

`AddActiveRect` (`GliderPRO/Sources/ObjectRects.c:277-292`):

```
1. if (nHotSpots >= kMaxHotSpots)   return (-1);       // 56 max
2. hotSpots[nHotSpots].bounds       = *bounds;
3. hotSpots[nHotSpots].action       = action;
4. hotSpots[nHotSpots].who          = who;
5. hotSpots[nHotSpots].isOn         = isOn;
6. hotSpots[nHotSpots].stillOver    = false;
7. hotSpots[nHotSpots].doScrutinize = doScrutinize;
8. nHotSpots++;
9. return (nHotSpots - 1);
```

`stillOver` is the universal "already handled this contact, do not repeat" latch. It is
set by `HandleSwitches`, `HandleMicrowaveAction`, `ArmTrigger`, `kStrumIt`, `kChimeIt`,
`kSoundIt`; it is cleared in `CheckForHotSpots` when the glider is no longer inside
(`GliderPRO/Sources/Interactions.c:1683-1684`) and re-derived wholesale by
`FlagStillOvers` (`:1715-1732`) on room entry.

`doScrutinize` selects the **5-pixel inset** collision test in `SectGlider` (see §9.2).

---

## 3. Object codes

### 3.1 Enemy class (`GliderPRO/Headers/GliderDefines.h:411-419`)

| Symbol | Hex | Dec | Editor palette | Becomes a dinah? | Static hot spot |
|---|---:|---:|---|---|---|
| `kBalloon` | 0x71 | 113 | `kEnemyMode` (8) | yes, central room only | `kIgnoreIt` |
| `kCopterLf` | 0x72 | 114 | 8 | yes, central room only | `kIgnoreIt` |
| `kCopterRt` | 0x73 | 115 | 8 | yes, central room only | `kIgnoreIt` |
| `kDartLf` | 0x74 | 116 | 8 | yes, central room only | `kIgnoreIt` |
| `kDartRt` | 0x75 | 117 | 8 | yes, central room only | `kIgnoreIt` |
| `kBall` | 0x76 | 118 | 8 | yes, central room only | `kIgnoreIt` |
| `kDrip` | 0x77 | 119 | 8 | yes, central room only | `kIgnoreIt` |
| `kFish` | 0x78 | 120 | 8 | yes, central room only | **`kDissolveIt`** |
| `kCobweb` | 0x79 | 121 | 8 | **no** | **`kWebIt`** |

### 3.2 Appliance class, hazard-relevant members (`GliderPRO/Headers/GliderDefines.h:396-409`)

| Symbol | Hex | Dec | Hot spot(s) | Dinah? |
|---|---:|---:|---|---|
| `kShredder` | 0x61 | 97 | `kShredIt` (state-gated) | no |
| `kToaster` | 0x62 | 98 | `kDissolveIt` | yes (any visible neighbour) |
| `kMacPlus` | 0x63 | 99 | `kDissolveIt` | yes |
| `kGuitar` | 0x64 | 100 | `kStrumIt` | no |
| `kTV` | 0x65 | 101 | `kDissolveIt` | yes |
| `kCoffee` | 0x66 | 102 | `kDissolveIt` | yes |
| `kOutlet` | 0x67 | 103 | `kIgnoreIt` (state-gated) | yes |
| `kVCR` | 0x68 | 104 | `kDissolveIt` | yes |
| `kStereo` | 0x69 | 105 | `kDissolveIt` | yes |
| `kMicrowave` | 0x6A | 106 | `kDissolveIt` + `kMicrowaveIt` column | yes |
| `kCinderBlock` | 0x6B | 107 | `kDissolveIt` | no |
| `kFlowerBox` | 0x6C | 108 | `kDissolveIt` | no |
| `kCDs` | 0x6D | 109 | `kDissolveIt` | no |
| `kCustomPict` | 0x6E | 110 | none | no |

### 3.3 Blower class, hazard-relevant members (`GliderPRO/Headers/GliderDefines.h:311-326`)

| Symbol | Hex | Dec | Hot spot(s) |
|---|---:|---:|---|
| `kFloorVent` | 0x01 | 1 | `kLiftIt` (state-gated) |
| `kCeilingVent` | 0x02 | 2 | `kDropIt` (state-gated) |
| `kFloorBlower` | 0x03 | 3 | `kLiftIt` (state-gated) |
| `kCeilingBlower` | 0x04 | 4 | `kDropIt` (state-gated) |
| `kSewerGrate` | 0x05 | 5 | `kLiftIt` (state-gated) |
| `kLeftFan` | 0x06 | 6 | `kDissolveIt` body + `kPushItLeft` column |
| `kRightFan` | 0x07 | 7 | `kDissolveIt` body + `kPushItRight` column |
| `kTaper` | 0x08 | 8 | `kLiftIt` (upper column) + `kBurnIt` (lower 24 px) + `kDissolveIt` body |
| `kCandle` | 0x09 | 9 | same three |
| `kStubby` | 0x0A | 10 | same three |
| `kTiki` | 0x0B | 11 | same three |
| `kBBQ` | 0x0C | 12 | same three |
| `kInvisBlower` | 0x0D | 13 | one of `kLiftIt`/`kDropIt`/`kPushItLeft`/`kPushItRight`, chosen by `vector` |
| `kGrecoVent` | 0x0E | 14 | `kLiftIt` |
| `kSewerBlower` | 0x0F | 15 | `kLiftIt` |
| `kLiftArea` | 0x10 | 16 | `kLiftIt` |

`kSparkle` = 0x2D = 45 (`GliderPRO/Headers/GliderDefines.h:356`) is a *prize*-class code
but becomes a dinah; it is cosmetic and never harms the player.

---

## 4. Sprite sheets, PICT resources and frame rectangles

### 4.1 PICT resource IDs (`GliderPRO/Sources/StructuresInit.c:19-38`)

Every sprite sheet is a `PICT` resource in the application's resource fork; its 1-bit
transparency mask is always at **PICT ID + 1000**.

| Constant | PICT ID | Mask ID |
|---|---:|---:|
| `kShadowPictID` | 3998 | 4998 |
| `kGliderPictID` | 3999 | 4999 |
| `kBlowerPictID` | 4000 | 5000 |
| `kFurniturePictID` | 4001 | 5001 |
| `kBonusPictID` | 4002 | 5002 |
| `kSwitchPictID` | 4003 | 5003 (**does not exist**) |
| `kLightPictID` | 4004 | 5004 |
| `kAppliancePictID` | 4005 | 5005 |
| `kPointsPictID` | 4006 | 5006 |
| `kRubberBandsPictID` | 4007 | 5007 |
| `kTransportPictID` | 4008 | 5008 |
| `kToastPictID` | 4009 | 5009 |
| `kShreddedPictID` | 4010 | 5010 |
| `kBalloonPictID` | 4011 | 5011 |
| `kCopterPictID` | 4012 | 5012 |
| `kDartPictID` | 4013 | 5013 |
| `kBallPictID` | 4014 | 5014 |
| `kDripPictID` | 4015 | 5015 |
| `kEnemyPictID` | 4016 | 5016 |
| `kFishPictID` | 4017 | 5017 |

### 4.2 Verified `picFrame` geometry

Parsed directly out of `GliderPRO/Glider PRO.r` with python3: each `data 'PICT' (id…)`
block's first 12 bytes were unpacked as `>h` size, `>4h` picFrame (t,l,b,r), `>h`
version opcode.

| PICT | size (bytes) | picFrame (t,l,b,r) | w×h | opcode |
|---:|---:|---|---|---|
| 3998 | 167 | (0,0,18,48) | 48×18 | 0x1101 |
| 3999 | 18688 | (0,0,668,48) | 48×668 | 0x0011 |
| 4000 | 11800 | (0,0,402,48) | 48×402 | 0x0011 |
| 4001 | 9416 | (0,0,221,64) | 64×221 | 0x0011 |
| 4002 | 15432 | (0,0,378,88) | 88×378 | 0x0011 |
| 4003 | 4650 | (0,0,104,32) | 32×104 | 0x0011 |
| 4004 | 5284 | (0,0,126,72) | 72×126 | 0x0011 |
| 4005 | 12578 | (0,0,269,80) | 80×269 | 0x0011 |
| 4006 | 4682 | (0,0,120,24) | 24×120 | 0x0011 |
| 4007 | 2384 | (0,0,18,16) | 16×18 | 0x0011 |
| 4008 | 2740 | (0,0,32,56) | 56×32 | 0x0011 |
| 4009 | 4768 | (0,0,174,32) | 32×174 | 0x0011 |
| 4010 | 3308 | (0,0,35,40) | 40×35 | 0x0011 |
| 4011 | 5618 | (0,0,240,24) | 24×240 | 0x0011 |
| 4012 | 5770 | (0,0,300,32) | 32×300 | 0x0011 |
| 4013 | 4434 | (0,0,76,64) | 64×76 | 0x0011 |
| 4014 | 3526 | (0,0,64,32) | 32×64 | 0x0011 |
| 4015 | 2822 | (0,0,72,16) | 16×72 | 0x0011 |
| 4016 | 2752 | (0,0,33,36) | 36×33 | 0x0011 |
| 4017 | 3902 | (0,0,128,16) | 16×128 | 0x0011 |
| 4998 | 167 | (0,0,18,48) | 48×18 | 0x1101 |
| 4999 | 4102 | (0,0,668,48) | 48×668 | 0x1101 |
| 5000 | 2471 | (0,0,402,48) | 48×402 | 0x1101 |
| 5001 | 1908 | (0,0,221,64) | 64×221 | 0x1101 |
| 5002 | 4500 | (0,0,378,88) | 88×378 | 0x1101 |
| 5004 | 1334 | (0,0,126,72) | 72×126 | 0x1101 |
| 5005 | 2124 | (0,0,**268**,80) | 80×268 | 0x1101 |
| 5006 | 539 | (0,**91**,120,**115**) | 24×120 | 0x1101 |
| 5007 | 95 | (0,0,18,16) | 16×18 | 0x1101 |
| 5008 | 313 | (0,0,32,56) | 56×32 | 0x1101 |
| 5009 | 755 | (0,0,174,32) | 32×174 | 0x1101 |
| 5010 | 269 | (**195**,0,**230**,40) | 40×35 | 0x1101 |
| 5011 | 1019 | (0,0,240,24) | 24×240 | 0x1101 |
| 5012 | 1259 | (0,0,300,32) | 32×300 | 0x1101 |
| 5013 | 702 | (0,0,76,64) | 64×76 | 0x1101 |
| 5014 | 315 | (0,0,64,32) | 32×64 | 0x1101 |
| 5015 | 203 | (0,0,72,16) | 16×72 | 0x1101 |
| 5016 | 257 | (0,0,33,36) | 36×33 | 0x1101 |
| 5017 | 315 | (0,0,128,16) | 16×128 | 0x1101 |

Observations relevant to fidelity:

* **PICT 5003 is absent.** There is no mask for the switch sheet; switches are drawn
  with `CopyBits`, not `CopyMask`. Confirmed by a full enumeration of `PICT` IDs in
  `Glider PRO.r`: 152 resources in runs
  `(150–151) (153) (1000–1023) (1202) (1211) (1216–1217) (1988–2017) (3903–3904)
  (3912–3915) (3921) (3927) (3957–4018) (4998–5002) (5004–5018) (10000)`.
* **Mask 5005 is 80×268, one row shorter than PICT 4005 (80×269).** `microOn` is
  defined at y = 222..257 (`GliderPRO/Sources/StructuresInit.c:602-603`) and `microOff` at
  y = 187..222 (`:604-605`), both well inside 268, so nothing breaks, but a port that
  asserts "mask and image are the same size" will fail on this pair.
* Masks 5006 and 5010 have **non-zero picFrame origins** (`(0,91)` and `(195,0)`).
  The Mac Picture Utilities draw a PICT into a destination rect, so the origin is
  ignored at draw time — `LoadGraphic` maps `picFrame` onto the GWorld's bounds. A port
  that naively decodes a PICT into an image at its picFrame origin will place these two
  masks in the wrong place.
* All colour sheets carry version opcode `0x0011` (PICT v2, `$0011` = `versionOp`
  followed by `$02FF`), while all masks carry `0x1101`, i.e. they are **PICT v1**
  (opcode stream begins immediately). A port needs both decoders.

### 4.3 Sprite-sheet GWorlds and per-frame source rects

`InitEnemies` (`GliderPRO/Sources/StructuresInit.c:614-723`) creates, for each of the
seven enemy sheets, an 8-bit indexed-colour source GWorld (`kPreferredDepth`) plus a
**1-bit** mask GWorld, then slices the sheet into frame rects.

| Sheet GWorld | Sheet rect | Frames | Frame size | Frame `i` rect | Constant |
|---|---|---:|---|---|---|
| `balloonSrcMap` / `balloonMaskMap` | 24×240 (`:623`) | 8 | 24×30 | `(0, 30i, 24, 30i+30)` (`:687-689`) | `kNumBalloonFrames` = 8 |
| `copterSrcMap` / `copterMaskMap` | 32×300 (`:632`) | 10 | 32×30 | `(0, 30i, 32, 30i+30)` (`:692-694`) | `kNumCopterFrames` = 10 |
| `dartSrcMap` / `dartMaskMap` | 64×76 (`:641`) | 4 | 64×19 | `(0, 19i, 64, 19i+19)` (`:697-699`) | `kNumDartFrames` = 4 |
| `ballSrcMap` / `ballMaskMap` | 32×64 (`:650`) | 2 | 32×32 | `(0, 32i, 32, 32i+32)` (`:702-704`) | `kNumBallFrames` = 2 |
| `dripSrcMap` / `dripMaskMap` | 16×72 (`:659`) | 6 | 16×12 | `(0, 12i, 16, 12i+12)` (`:707-709`) | `kNumDripFrames` = 6 |
| `enemySrcMap` / `enemyMaskMap` | 36×33 (`:668`) | 1 | 36×33 | (the static fish) | — |
| `fishSrcMap` / `fishMaskMap` | 16×128 (`:677`) | 8 | 16×16 | `(0, 16i, 16, 16i+16)` (`:717-719`) | `kNumFishFrames` = 8 |

Frame counts from `GliderPRO/Headers/GliderDefines.h:452-457`.

`InitAppliances` (`GliderPRO/Sources/StructuresInit.c:529-608`) — hazard-relevant slices
out of the 80×269 appliance sheet (`applianceSrcRect`), the 32×174 toast sheet
(`toastSrcRect`) and the 40×35 shredded-glider sheet (`shredSrcRect`):

| Rect | Size | Position in sheet | Cite |
|---|---|---|---|
| `breadSrc[i]`, i = 0..5 | 32×29 | `(0, 29i)` in `toastSrcMap` | `:586-590` |
| `shredSrcRect` | 40×35 | whole `shredSrcMap` | `:556` |
| `outletSrc[i]`, i = 0..3 | 16×24 | `(64, 22+24i)` | `:580-584` |
| `plusScreen1` | 32×22 | `(48, 127)` | `:565-566` |
| `plusScreen2` | 32×22 | `(48, 149)` | `:567-568` |
| `tvScreen1` | 64×49 | `(0, 171)` | `:570-571` |
| `tvScreen2` | 64×49 | `(0, 220)` | `:572-573` |
| `coffeeLight1` | 8×4 | `(72, 171)` | `:575-576` |
| `coffeeLight2` | 8×4 | `(72, 175)` | `:577-578` |
| `vcrTime1` | 16×4 | `(64, 179)` | `:592-593` |
| `vcrTime2` | 16×4 | `(64, 183)` | `:594-595` |
| `stereoLight1` | 4×1 | `(68, 171)` | `:597-598` |
| `stereoLight2` | 4×1 | `(68, 172)` | `:599-600` |
| `microOn` | 16×35 | `(64, 222)` | `:602-603` |
| `microOff` | 16×35 | `(64, 187)` | `:604-605` |

(The GWorld is named `shredSrcMap`, not `shreddedSrcMap`; `kShreddedPictID` is the PICT
resource that fills it — `GliderPRO/Sources/StructuresInit.c:556-563`.)

`InitBlowers` (`GliderPRO/Sources/StructuresInit.c:244-287`) — flame frame slices out of
the 48×402 blower sheet (`blowerSrcRect`):

| Rect | Size | Position | Count | Constant | Cite |
|---|---|---|---:|---|---|
| `flame[i]` | 16×15 | `(32, 179 + 15i)` | 5 | `kNumCandleFlames` = 5 | `:262-266` |
| `tikiFlame[i]` | 8×10 | `(40, 69 + 10i)` | 5 | `kNumTikiFlames` = 5 | `:268-272` |
| `coals[i]` | 32×9 | `(0, 304 + 9i)` | 4 | `kNumBBQCoals` = 4 | `:274-278` |
| `leftStartGliderSrc` | 48×16 | `(0, 358)` | 1 | — | `:280-281` |
| `rightStartGliderSrc` | 48×16 | `(0, 374)` | 1 | — | `:283-284` |

Sparkle frames come out of the bonus sheet in `InitPrizes`
(`GliderPRO/Sources/StructuresInit.c:341`): `sparkleSrc[i+2]` = 20×19 at `(0, 70+19i)`
for i = 0..2, then the deliberately non-monotonic assignment
`sparkleSrc[0] = sparkleSrc[4]; sparkleSrc[1] = sparkleSrc[3];`. The resulting
5-mode sequence walks y = **108, 89, 70, 89, 108** — the sparkle grows then shrinks.
`kNumSparkleModes` = 5 (`GliderPRO/Headers/GliderDefines.h:252`).

### 4.4 `srcRects[]` — the 144-entry static geometry table

`srcRects` is **not a resource**; it is `NewPtr(sizeof(Rect) * kNumSrcRects)` with
`kNumSrcRects` = `0x90` = **144** (`GliderPRO/Headers/GliderDefines.h:437`), filled
entirely by hardcoded `QSetRect`/`QOffsetRect` calls in `InitSrcRects`
(`GliderPRO/Sources/StructuresInit2.c:306-475`). It is indexed by the object code
itself, so `srcRects[kBalloon]` == `srcRects[0x71]`.

Hazard-relevant entries:

| Index | Symbol | w×h | Offset in sheet | Cite |
|---:|---|---|---|---|
| 0x01 | `kFloorVent` | 48×11 | (0,0) | `:308` |
| 0x02 | `kCeilingVent` | 48×11 | (0,11) | `:310` |
| 0x03 | `kFloorBlower` | 48×15 | (0,22) | `:312` |
| 0x04 | `kCeilingBlower` | 48×15 | (0,37) | `:314` |
| 0x05 | `kSewerGrate` | 48×17 | (0,52) | `:316` |
| 0x06 | `kLeftFan` | 40×55 | (0,69) | `:318` |
| 0x07 | `kRightFan` | 40×55 | (0,124) | `:320` |
| 0x08 | `kTaper` | 20×59 | (0,209) | `:322` |
| 0x09 | `kCandle` | 32×30 | (0,179) | `:324` |
| 0x0A | `kStubby` | 20×36 | (0,268) | `:326` |
| 0x0B | `kTiki` | 27×28 | (21,268) | `:328` |
| 0x0C | `kBBQ` | 64×33 | (0,0) | `:330` |
| 0x0D | `kInvisBlower` | 24×24 | — | `:331` |
| 0x0E | `kGrecoVent` | 48×18 | (0,340) | `:332` |
| 0x0F | `kSewerBlower` | 32×12 | (0,390) | `:334` |
| 0x10 | `kLiftArea` | 64×32 | — | `:336` |
| 0x61 | `kShredder` | 73×22 | — | `:430` |
| 0x62 | `kToaster` | 48×27 | (0,22) | `:431` |
| 0x63 | `kMacPlus` | 48×58 | (0,49) | `:433` |
| 0x64 | `kGuitar` | 64×172 | — | `:435` |
| 0x65 | `kTV` | 92×77 | — | `:436` |
| 0x66 | `kCoffee` | 43×64 | (0,107) | `:437` |
| 0x67 | `kOutlet` | 16×24 | (64,22) | `:439` |
| 0x68 | `kVCR` | 96×22 | — | `:441` |
| 0x69 | `kStereo` | 128×53 | — | `:442` |
| 0x6A | `kMicrowave` | 92×59 | — | `:443` |
| 0x6B | `kCinderBlock` | 40×62 | — | `:444` |
| 0x6C | `kFlowerBox` | 80×32 | — | `:445` |
| 0x6D | `kCDs` | 16×30 | (48,22) | `:446` |
| 0x6E | `kCustomPict` | 72×34 | — | `:448` |
| 0x71 | `kBalloon` | 24×30 | — | `:450` |
| 0x72 | `kCopterLf` | 32×30 | — | `:451` |
| 0x73 | `kCopterRt` | 32×30 | — | `:452` |
| 0x74 | `kDartLf` | 64×19 | — | `:453` |
| 0x75 | `kDartRt` | 64×19 | — | `:454` |
| 0x76 | `kBall` | 32×32 | — | `:455` |
| 0x77 | `kDrip` | 16×12 | — | `:456` |
| 0x78 | `kFish` | 36×33 | — | `:457` |
| 0x79 | `kCobweb` | 54×45 | — | `:458` |
| 0x8F | `kChimes` | 28×74 | — | `:474` |

`srcRects[kFlower]` (0x85) is **never assigned** in `InitSrcRects`; the flower's rect
comes from `data.i.bounds` instead. (Not a hazard, but a porter enumerating the table
will notice the hole.)

### 4.5 Room geometry constants (`GliderPRO/Headers/GliderDefines.h:496-511`)

| Constant | Value | Meaning |
|---|---:|---|
| `kNumTiles` | 8 | background tiles per room |
| `kTileWide` | 64 | tile width in px |
| `kTileHigh` | 322 | room height in px |
| `kRoomWide` | 512 | `kNumTiles * kTileWide` |
| `kFloorSupportTall` | 44 | height of the floor-support strip below a room |
| `kVertLocalOffset` | 322 | vertical stride between vertically adjacent rooms |
| `kCeilingLimit` | 8 | player ceiling |
| `kFloorLimit` | 312 | player floor |
| `kRoofLimit` | 122 | outdoor roof line |
| `kLeftWallLimit` | 12 | player left wall |
| `kNoLeftWallLimit` | −24 | = `0 - kGliderWide/2` |
| `kRightWallLimit` | 500 | player right wall |
| `kNoRightWallLimit` | 536 | = `kRoomWide + kGliderWide/2` |
| `kNoCeilingLimit` | −10 | — |
| `kNoFloorLimit` | 332 | — |

Hazard-local bounds (defined per-file, not in the header):

| Constant | Value | File | Used by |
|---|---:|---|---|
| `kBalloonStop` | 8 | `GliderPRO/Sources/Dynamics2.c:13` | balloon top respawn line |
| `kBalloonStart` | 310 | `GliderPRO/Sources/Dynamics2.c:14` and `Dynamics3.c:13` | balloon spawn `dest.bottom` and bottom respawn line |
| `kCopterStart` | 8 | `GliderPRO/Sources/Dynamics2.c:15`, `Dynamics3.c:14` | copter spawn `dest.top` |
| `kCopterStop` | 310 | `GliderPRO/Sources/Dynamics2.c:16` | copter bottom respawn line; `HandleCopter` compares against the named constants `kCopterStart`/`kCopterStop` (`Dynamics2.c:191-192`) |
| `kDartVelocity` | 6 | `GliderPRO/Sources/Dynamics2.c:17`, `Dynamics3.c:15` | dart `|hVel|` |
| `kDartStop` | 310 | `GliderPRO/Sources/Dynamics2.c:18` | dart bottom respawn line |
| `kEnemyDropSpeed` | 8 | `GliderPRO/Sources/Dynamics2.c:19` | `vVel` after a rubber band hit |
| `kShoveVelocity` | 8 | `GliderPRO/Sources/Dynamics.c:17` | foil shove impulse |
| `kFloorVentLift` | −6 | `GliderPRO/Sources/Interactions.c:13` | `kLiftIt` |
| `kCeilingVentDrop` | 8 | `GliderPRO/Sources/Interactions.c:14` | `kDropIt` |
| `kFanStrength` | 12 | `GliderPRO/Sources/Interactions.c:15` | `kPushItLeft`/`kPushItRight` |
| `kDeadlyFlameHeight` | 24 | `GliderPRO/Sources/ObjectRects.c` (local) | lower 24 px of a flame column burn |
| `kFloorColumnWide` | 4 | `GliderPRO/Sources/ObjectRects.c` (local) | flame/floor-vent column width |
| `kCeilingColumnWide` | 24 | `GliderPRO/Sources/ObjectRects.c` (local) | ceiling-vent column width |
| `kFanColumnThick` | 16 | `GliderPRO/Sources/ObjectRects.c` (local) | fan push column height |
| `kFanColumnDown` | 20 | `GliderPRO/Sources/ObjectRects.c` (local) | fan push column vertical offset |
| `kShredderActiveHigh` | 40 | `GliderPRO/Sources/ObjectRects.c` (local) | shredder kill-zone height |
| `kKillWebbedGlider` | 150 | `GliderPRO/Sources/Interactions.c:1738` | frames of struggle before a web kills |
| `kFramesToBurn` | 60 | `GliderPRO/Sources/Modes.c:408` | frames of burning before death |
| `kShredderCountdown` | −68 | `GliderPRO/Sources/Player.c:17` | shred post-death dwell |
| `kRubberBandVelocity` | 20 | `GliderPRO/Sources/RubberBands.c:12` | band launch `|hVel|` |
| `kBandFallCount` | 4 | `GliderPRO/Sources/RubberBands.c:13` | frames per band gravity tick |
| `kKillBandMode` | −1 | `GliderPRO/Sources/RubberBands.c:14` | band "delete me" sentinel |

Player physics constants (`GliderPRO/Sources/Player.c:13-17`): `kGravity` = 3,
`kHImpulse` = 2, `kVImpulse` = 2, `kMaxHVel` = 16, `kShredderCountdown` = −68.

Player size (`GliderPRO/Headers/GliderDefines.h:548-553`): `kGliderWide` = 48,
`kGliderHigh` = 20, `kHalfGliderWide` = 24, `kGliderBurningHigh` = 26,
`kShadowHigh` = 9, `kShadowTop` = 306.

---

## 5. Sounds

Sound index N corresponds to `'snd '` resource **1000 + N**
(`kBaseBufferSoundID` = 1000, `GliderPRO/Sources/Sound.c:14`). `LoadBufferSounds`
(`GliderPRO/Sources/Sound.c:316-346`) loads `i + kBaseBufferSoundID` for
i = 0..`kMaxSounds - 2` = 0..62, and each buffer pointer is advanced by
**20 bytes** past the resource start (`*theSound + 20L`) — the classic `'snd '` format 1
header for a single sampled-sound command. Slot 63 (`kMaxSounds - 1`) is reserved: it is
loaded from the *house* file by `LoadTriggerSound` (`GliderPRO/Sources/Sound.c:265-303`),
also skipping 20 bytes, and is played as `kTriggerSound`.

**Empirically verified** by enumerating `data 'snd ' (id…)` blocks in
`GliderPRO/Glider PRO.r`: **70** `'snd '` resources, IDs in two runs —
**1000–1062** (63 contiguous, exactly indices 0..62) and **2000–2006** (7 music
resources). There is no `'snd '` 1063, confirming that index 63 is house-supplied.

### 5.1 Full sound table (`GliderPRO/Headers/GliderDefines.h:55-180`)

| Idx | `'snd '` | Symbol | Priority symbol | Priority | Used by (hazard-relevant) |
|---:|---:|---|---|---:|---|
| 0 | 1000 | `kHitWallSound` | `kHitWallPriority` | 100 | `BounceGlider` w/o foil; band-hits-glider |
| 1 | 1001 | `kFadeInSound` | `kFadeInPriority` | 900 | respawn |
| 2 | 1002 | `kFadeOutSound` | `kFadeOutPriority` | 901 | **every instant death**; shred sparkle |
| 3 | 1003 | `kBeepsSound` | `kBeepsPriority` | 800 | — |
| 4 | 1004 | `kBuzzerSound` | `kBuzzerPriority` | 801 | — |
| 5 | 1005 | `kDingSound` | `kDingPriority` | 802 | — |
| 6 | 1006 | `kEnergizeSound` | `kEnergizePriority` | 803 | — |
| 7 | 1007 | `kFollowSound` | `kFollowPriority` | 904 | — |
| 8 | 1008 | `kMicrowavedSound` | `kMicrowavedPriority` | 811 | **microwave drain** |
| 9 | 1009 | `kSwitchSound` | `kSwitchPriority` | 700 | switch flip |
| 10 | 1010 | `kBirdSound` | `kBirdPriority` | 804 | — |
| 11 | 1011 | `kCuckooSound` | `kCuckooPriority` | 805 | — |
| 12 | 1012 | `kTikSound` | `kTikPriority` | 200 | pendulum far swing |
| 13 | 1013 | `kTokSound` | `kTokPriority` | 201 | pendulum near swing |
| 14 | 1014 | `kBlowerOn` | `kBlowerOnPriority` | 701 | blower switched on |
| 15 | 1015 | `kBlowerOff` | `kBlowerOffPriority` | 702 | blower switched off |
| 16 | 1016 | `kCaughtFireSound` | `kCaughtFirePriority` | 902 | **`FlagGliderBurning` and `FlagGliderShredding`** |
| 17 | 1017 | `kScoreTikSound` | `kScoreTikPriority` | 101 | — |
| 18 | 1018 | `kThrustSound` | `kThrustPriority` | 300 | — |
| 19 | 1019 | `kFizzleSound` | `kFizzlePriority` | 703 | — |
| 20 | 1020 | `kFireBandSound` | `kFireBandPriority` | 301 | **band fired** |
| 21 | 1021 | `kBandReboundSound` | `kBandReboundPriority` | 102 | **band bounces off wall/obstacle** |
| 22 | 1022 | `kGreaseSpillSound` | `kGreaseSpillPriority` | 806 | grease knocked over |
| 23 | 1023 | `kChordSound` | `kChordPriority` | 302 | **guitar strum**; `kSoundTrigger` fired |
| 24 | 1024 | `kVCRSound` | `kVCRPriority` | 303 | VCR |
| 25 | 1025 | `kFoilHitSound` | `kFoilHitPriority` | 400 | **foil absorbs a hit** (dynamic, shredder, wall) |
| 26 | 1026 | `kShredSound` | `kShredPriority` | 903 | **shredding, per frame** |
| 27 | 1027 | `kToastLaunchSound` | `kToastLaunchPriority` | 304 | **toast launches** |
| 28 | 1028 | `kToastLandSound` | `kToastLandPriority` | 305 | **toast lands** |
| 29 | 1029 | `kMacOnSound` | `kMacOnPriority` | 401 | appliance turns on |
| 30 | 1030 | `kMacBeepSound` | `kMacBeepPriority` | 403 | Mac Plus boot chime |
| 31 | 1031 | `kMacOffSound` | `kMacOffPriority` | 402 | appliance turns off |
| 32 | 1032 | `kTVOnSound` | `kTVOnPriority` | 404 | TV on |
| 33 | 1033 | `kTVOffSound` | `kTVOffPriority` | 405 | TV off |
| 34 | 1034 | `kCoffeeSound` | `kCoffeePriority` | 306 | coffee percolate; `kCoffee` trigger |
| 35 | 1035 | `kMysticSound` | `kMysticPriority` | 202 | `kSparkle` object twinkle |
| 36 | 1036 | `kZapSound` | `kZapPriority` | 406 | **outlet arc** |
| 37 | 1037 | `kPopSound` | `kPopPriority` | 407 | **balloon popped by a band** |
| 38 | 1038 | `kEnemyInSound` | `kEnemyInPriority` | 408 | **balloon/copter/dart materialises** |
| 39 | 1039 | `kEnemyOutSound` | `kEnemyOutPriority` | 409 | **balloon/copter/dart despawns** |
| 40 | 1040 | `kPaperCrunchSound` | `kPaperCrunchPriority` | 410 | **copter or dart shot down** |
| 41 | 1041 | `kBounceSound` | `kBouncePriority` | 307 | **ball hits its floor** |
| 42 | 1042 | `kDripSound` | `kDripPriority` | 308 | **drip detaches** |
| 43 | 1043 | `kDropSound` | `kDropPriority` | 309 | **drip splashes; fish re-enters water** |
| 44 | 1044 | `kFishOutSound` | `kFishOutPriority` | 411 | **fish jumps** |
| 45 | 1045 | `kFishInSound` | `kFishInPriority` | 412 | **fish lands** |
| 46 | 1046 | `kDontExitSound` | `kDontExitPriority` | 103 | — |
| 47 | 1047 | `kSizzleSound` | `kSizzlePriority` | 413 | **foil sizzles in a flame** |
| 48 | 1048 | `kPaper1Sound` | `kPapersPriority` | 807 | — |
| 49 | 1049 | `kPaper2Sound` | `kPapersPriority` | 807 | — |
| 50 | 1050 | `kPaper3Sound` | `kPapersPriority` | 807 | — |
| 51 | 1051 | `kPaper4Sound` | `kPapersPriority` | 807 | — |
| 52 | 1052 | `kTypingSound` | `kTypingPriority` | 808 | — |
| 53 | 1053 | `kCarriageSound` | `kCarriagePriority` | 809 | — |
| 54 | 1054 | `kChord2Sound` | `kChord2Priority` | 810 | — |
| 55 | 1055 | `kPhoneRingSound` | `kPhoneRingPriority` | 500 | — |
| 56 | 1056 | `kChime1Sound` | `kChime1Priority` | 203 | `kChimeIt` |
| 57 | 1057 | `kChime2Sound` | `kChime2Priority` | 204 | `kChimeIt` |
| 58 | 1058 | `kWebTwangSound` | `kWebTwangPriority` | 310 | **cobweb struggle** |
| 59 | 1059 | `kTransOutSound` | `kTransOutPriority` | 906 | — |
| 60 | 1060 | `kTransInSound` | `kTransInPriority` | 905 | — |
| 61 | 1061 | `kBonusSound` | `kBonusPriority` | 812 | — |
| 62 | 1062 | `kHissSound` | `kHissPriority` | 311 | — |
| 63 | *house* | `kTriggerSound` | `kTriggerPriority` | 999 | `kSoundTrigger` hot spot (`kSoundIt`) |

### 5.2 Priority arbitration

`PlayPrioritySound(which, priority)` (`GliderPRO/Sources/Sound.c:40-85`):

```
1. if (failedSound || dontLoadSounds) return;
2. if (priority == kTriggerPriority) and any of the 3 channels currently has
       priority == kTriggerPriority, return;              // one trigger sound at a time
3. lowestPriority = min(priority0, priority1, priority2);
   whichChannel   = argmin, tie-broken toward channel 0, then 1, then 2;
4. if (priority >= lowestPriority)  PlaySound<whichChannel>(which, priority);
```

`which` is passed through **unchanged** — `PlaySound0/1/2` use it directly as the index
into the pre-loaded `theSoundData[]` array (`GliderPRO/Sources/Sound.c:133-157`), not as a
resource ID. `kBaseBufferSoundID` is added only once, at load time, in
`LoadBufferSounds`.

There are exactly **three** sound channels. **Higher number = higher priority.**
`priority` for an idle channel is `kNoSoundPlaying` = −1 (`GliderPRO/Sources/Sound.c:16`),
so an idle channel always wins the argmin. A sound is dropped only when *all three*
channels are busy with sounds of strictly higher priority.

Practical consequences for hazards: `kFadeOutSound` (901) and `kCaughtFireSound` (902)
and `kShredSound` (903) essentially always play; `kHitWallSound` (100) and
`kBandReboundSound` (102) are the first things dropped in a busy room.

---

## 6. Hot-spot action codes

All 28 actions (`GliderPRO/Headers/GliderDefines.h:282-309`), with the handler line in
`HandleHotSpotCollision` (`GliderPRO/Sources/Interactions.c:1198-1623`):

| Code | Value | Handler line | Effect on the player |
|---|---:|---:|---|
| `kIgnoreIt` | 0 | (no case) | nothing — placeholder so `hotNum` exists |
| `kLiftIt` | 1 | 1202 | `vDesiredVel = kFloorVentLift` = **−6** |
| `kDropIt` | 2 | 1206 | `vDesiredVel = kCeilingVentDrop` = **+8** |
| `kPushItLeft` | 3 | 1210 | `hDesiredVel += -kFanStrength` = **−12** |
| `kPushItRight` | 4 | 1214 | `hDesiredVel += kFanStrength` = **+12** |
| `kDissolveIt` | 5 | 1218 | **death**, or one foil charge + shove |
| `kRewardIt` | 6 | 1246 | prize |
| `kMoveItUp` | 7 | 1250 | stairs up |
| `kMoveItDown` | 8 | 1285 | stairs down |
| `kSwitchIt` | 9 | 1320 | `HandleSwitches` |
| `kShredIt` | 10 | 1324 | **shredder death**, or one foil charge |
| `kStrumIt` | 11 | 1343 | guitar chord, once per contact |
| `kTriggerIt` | 12 | 1351 | `ArmTrigger` |
| `kBurnIt` | 13 | 1356 | **burning death**, or lift + sizzle + one foil charge |
| `kSlideIt` | 14 | 1376 | `sliding = true`, snap to surface |
| `kTransportIt` | 15 | 1381 | teleport |
| `kIgnoreLeftWall` | 16 | 1416 | `ignoreLeft = true` |
| `kIgnoreRightWall` | 17 | 1420 | `ignoreRight = true` |
| `kMailItLeft` | 18 | 1424 | mailbox |
| `kMailItRight` | 19 | 1467 | mailbox |
| `kDuctItDown` | 20 | 1510 | duct |
| `kDuctItUp` | 21 | 1543 | duct |
| `kMicrowaveIt` | 22 | 1581 | **drains bands/battery/foil** |
| `kIgnoreGround` | 23 | 1586 | `ignoreGround = true` |
| `kBounceIt` | 24 | 1590 | `BounceGlider` |
| `kChimeIt` | 25 | 1594 | `StrikeChime()`, once per contact |
| `kWebIt` | 26 | 1602 | **cobweb capture, then death after 150 frames** |
| `kSoundIt` | 27 | 1615 | `kTriggerSound`, once per contact |

`case kLgTrigger:` appears as a *label* alongside `case kTriggerIt:`
(`GliderPRO/Sources/Interactions.c:1351-1354`). `kLgTrigger` = `0x48` = **72**, which is
an *object* code, not an action code. Since no `AddActiveRect` call ever stores action
72, this label is dead. A Go port should simply drop it (see §16).

---

## 7. Hazard → effect summary

The table the assignment asks for. "Drains" means it removes a player resource without
killing. Foil (`foilTotal`) is a damage-absorbing counter granted by the `kFoil` prize
(supply `kFoilSupply` = 8, `GliderPRO/Sources/Interactions.c:19`).

| Hazard | Without foil | With foil (`foilTotal > 0`) | Destructible | Sound on contact |
|---|---|---|---|---|
| `kBalloon` (moving) | **kills instantly** | pushes (shove ±8) + inherits balloon `vVel` if rising; costs 1 foil on `evenFrame` | **yes** — rubber band | `kFoilHitSound` (foil) / `kFadeOutSound` (death) |
| `kCopterLf`/`kCopterRt` (moving) | **kills instantly** | same shove | **yes** — rubber band | as above |
| `kDartLf`/`kDartRt` (moving) | **kills instantly** | same shove | **yes** — rubber band | as above |
| `kBall` (always, even idle) | **kills instantly** | same shove | no | as above |
| `kDrip` (always, even idle) | **kills instantly** | same shove | no | as above |
| `kFish` (moving) | **kills instantly** | same shove | no | as above |
| `kFish` (static hot spot, 36×33) | **kills instantly** (`kDissolveIt`) | 1 foil + horizontal bounce | no | `kFoilHitSound` |
| `kCobweb` | **captures, then kills after 150 frames** of struggle | **no protection at all** — foil is not consulted | no | `kWebTwangSound` |
| `kShredder` (73+48 × 40 zone) | **kills** via `kGliderShredding` | 1 foil per frame of contact, no shred | no | `kFoilHitSound` / `kCaughtFireSound` |
| Flying toast (`kToaster` dinah) | **kills instantly** | pushes | no | `kFoilHitSound` / `kFadeOutSound` |
| `kToaster` body | **kills instantly** (`kDissolveIt`) | **2** foil + bounce (see below) | no | `kFoilHitSound` |
| `kOutlet` arc (`position != 0`) | **kills instantly** | pushes | no | `kZapSound` continuously |
| `kMicrowave` body | **kills instantly** (`kDissolveIt`) | **2** foil + bounce (see below) | no | `kFoilHitSound` |
| `kMicrowave` column above it | **drains** bands and/or battery and/or foil per `byte0` mask | drains foil to 0 if bit 2 set | no | `kMicrowavedSound` |
| Flame column, lower 24 px (`kBurnIt`) | **sets the glider on fire**, dies after 60 frames | `vDesiredVel = −6` (lifted), `kSizzleSound`, 1 foil | no | `kCaughtFireSound` / `kSizzleSound` |
| Flame column, above the lower 24 px (`kLiftIt`) | **pushes** up (`vDesiredVel = −6`) | same | no | — |
| Flame body (`kDissolveIt` rect) | **kills instantly** | **2** foil + bounce (see below) | no | `kFoilHitSound` |
| `kLeftFan` / `kRightFan` column | **pushes** ±12 | same | no | — |
| `kLeftFan` / `kRightFan` body (13×43) | **kills instantly** | **2** foil + bounce (see below) | no | `kFoilHitSound` |
| `kMacPlus`, `kTV`, `kCoffee`, `kVCR`, `kStereo`, `kCinderBlock`, `kFlowerBox`, `kCDs` bodies | **kills instantly** | **2** foil + bounce (see below) | no | `kFoilHitSound` |
| `kGuitar` strum column (8×96) | plays `kChordSound` — **harmless** | same | no | `kChordSound` |
| `kFloorVent`, `kFloorBlower`, `kSewerGrate`, `kGrecoVent`, `kSewerBlower`, `kLiftArea` | **pushes** up −6 | same | no | — |
| `kCeilingVent`, `kCeilingBlower` | **pushes** down +8 | same | no | — |
| `kInvisBlower` | pushes per `vector` bit | same | no | — |
| Rubber band (own or other player's) | `theGlider.hVel += band.hVel / 2` — **pushes** | same | (the band dies) | `kHitWallSound` |

Explicit note on the phrase "bounces": there are two distinct bounce mechanisms.

* **`kBounceIt`** (`kInvisBounce` object) calls `BounceGlider`
  (`GliderPRO/Sources/Interactions.c:154-167`), which sets `hVel` to the *penetration
  depth* on the nearer side. This is not a hazard.
* **Foil-absorbed `kDissolveIt`** calls `GliderHitTop`
  (`GliderPRO/Sources/Interactions.c:54-97`) which, when the contact was *not* a top
  hit, reverses `hVel` with an extra separation offset. That is the "bounce off a solid
  appliance while wearing foil" behaviour.

**`kDissolveIt` costs TWO foil charges per frame of side contact, not one.**
`GliderHitTop` decrements `foilTotal` itself in its `!hitTop` branch
(`GliderPRO/Sources/Interactions.c:82`), and then `case kDissolveIt:` decrements it
*again* on return (`:1230-1235`). So `kFoilSupply` = 8 buys only **4 frames** of
grinding against a solid appliance body, versus 8 frames against a shredder or a flame
(`kShredIt` `:1331-1336` and `kBurnIt` `:1363-1369` each decrement once) and 16 frames
against a moving dinah (`CheckDynamicCollision`, `evenFrame`-gated). `kFoilHitSound` is
likewise played twice — once inside `GliderHitTop` (`:81`) — though the second play is
suppressed by the one-per-priority mixer rule. If `foilTotal` was exactly 1 on entry,
`GliderHitTop` drops it to 0 and calls `StartGliderFoilLosing`, and the `:1230`
`foilTotal > 0` guard then skips the second decrement — so the cost is
`min(2, foilTotal)`.

---

## 8. Spawning: `AddDynamicObject` and the neighbour-room rule

`AddDynamicObject(short what, Rect *where, objectType *who, short room, short index,
Boolean isOn)` (`GliderPRO/Sources/Dynamics3.c:187-554`).

```
 1. if (numDynamics >= kMaxDynamicObs)  return (-1);         // 18 slots, Dynamics3.c:193
 2. dinahs[numDynamics].type = what;
 3. switch (what) { ...per-type initialisation... default: return (-1); }
 4. numDynamics++;
 5. return (numDynamics - 1);
```

`where` is **room-local** (the caller has already subtracted `playOrigin`), `who` is the
`objectType` record, `room` is the real room number, `index` is the object's slot 0..23
in that room, `isOn` is the object's `state`.

Only 17 `what` values are accepted; anything else falls into `default: return (-1)`
(`GliderPRO/Sources/Dynamics3.c:546-548`). That list is mirrored exactly in the editor's
`HowManyDynamicObjects` (`GliderPRO/Sources/ObjectAdd.c:1045-1070`): `kSparkle`,
`kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`, `kMicrowave`,
`kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish`.

### 8.1 Which neighbours produce dynamics

`DrawARoomsObjects(neighbor, redraw)` (`GliderPRO/Sources/ObjectDrawAll.c:23-965`) is
called once per visible neighbour room. Hazard-relevant behaviour:

| Object | Statically drawn? | `AddDynamicObject` gate | Cite |
|---|---|---|---|
| `kSparkle` | no | `(!redraw) && (neighbor == kCentralRoom)` | `:447-460` |
| `kToaster` | yes, in any visible neighbour | `(!redraw) && (neighbor == kCentralRoom)` | `:642-656` |
| `kMacPlus` | yes | `(!redraw)` — **any** neighbour | `:658-672` |
| `kTV` | yes | `(!redraw)` — any neighbour | `:674-713` |
| `kCoffee` | yes | `(!redraw)` — any neighbour | `:715-729` |
| `kOutlet` | yes if `isLit` | `(!redraw)` — any neighbour | `:731-746` |
| `kVCR` | yes | `(!redraw)` — any neighbour | `:748-762` |
| `kStereo` | yes | `(!redraw)` — any neighbour | `:764-778` |
| `kMicrowave` | yes | `(!redraw)` — any neighbour | `:780-794` |
| `kBalloon` | **never drawn statically** | `(neighbor == kCentralRoom) && (!redraw)` | `:796-805` |
| `kCopterLf` | never | same | `:807-816` |
| `kCopterRt` | never | same | `:818-827` |
| `kDartLf` | never | same | `:829-838` |
| `kDartRt` | never | same | `:840-849` |
| `kBall` | never | same | `:851-860` |
| `kDrip` | yes (`DrawDrip`) | `(!redraw) && (neighbor == kCentralRoom)` | `:862-876` |
| `kFish` | yes (`DrawFish`, 36×33 static art) | `(!redraw) && (neighbor == kCentralRoom)` | `:878-892` |
| `kShredder` | yes if `isLit`, no dynamic | — | `:634-640` |
| `kCDs` | yes if `isLit`, no dynamic | — | `:634-640` |
| `kCobweb` | yes if `isLit`, no dynamic | — | `:894-900` |

Consequences:

1. **Balloon, both copters, both darts and the ball are the only six object types that
   are never statically drawn and are never `SectRect`-tested against the room.** The
   `if ((neighbor == kCentralRoom) && (!redraw))` test comes *before* `GetObjectRect`,
   so nothing at all happens for them in a non-central room. They exist only as
   dinahs.
2. Everything else with a dynamic is `SectRect`-tested against `testRect = houseRect`
   with corner zeroed (`GliderPRO/Sources/ObjectDrawAll.c:36-37`), so an off-screen
   appliance gets no dynamic even in the central room.
3. `redraw == true` (a room repaint, e.g. after a lighting change) suppresses **all**
   dynamic creation, on the assumption that the existing dinahs are still live.
4. The 18-slot budget is consumed by appliances in the whole 3×3 neighbourhood plus
   hazards from the central room only. A room surrounded by TVs can therefore starve
   its own balloons.

### 8.2 The `dynaNum` back-link

After the object loop, `DrawARoomsObjects` writes the dynamic index back into
`masterObjects` (`GliderPRO/Sources/ObjectDrawAll.c:953-960`):

```
if (!redraw)
    for (n = 0; n < numMasterObjects; n++)
        if ((masterObjects[n].objectNum == i) &&
            (masterObjects[n].roomNum   == localNumbers[neighbor]))
            masterObjects[n].dynaNum = dynamicNum;
```

`dynamicNum` is reset to −1 at the top of each object iteration
(`GliderPRO/Sources/ObjectDrawAll.c:45`), so objects that produced no dynamic get −1.
This is the link that lets a switch or a trigger reach a live hazard: switches go
`hotSpots[i].who` → `masterObjects[...].localLink` → `masterObjects[link].dynaNum` →
`dinahs[dynaNum]`.

### 8.3 The reverse-engineered launch-velocity solvers

Three hazards let the level designer specify a **pixel height** and the engine solves
for the integer launch velocity that reaches it. There are two variants.

**Full-rate (toast only)** — `GliderPRO/Sources/Dynamics3.c:224-233`:

```
position = who->data.g.height;      // pixels of desired rise
velocity = 0;
do {
    velocity++;
    position -= velocity;
} while (position > 0);
vVel  = -velocity;
count =  velocity;
```

This is the smallest `n` with `n(n+1)/2 >= height`. Table of solutions:

| `height` range | `velocity` | rise actually achieved (`n(n+1)/2`) |
|---|---:|---:|
| 1 | 1 | 1 |
| 2–3 | 2 | 3 |
| 4–6 | 3 | 6 |
| 7–10 | 4 | 10 |
| 11–15 | 5 | 15 |
| 16–21 | 6 | 21 |
| 22–28 | 7 | 28 |
| 29–36 | 8 | 36 |
| 37–45 | 9 | 45 |
| 46–55 | 10 | 55 |
| 56–66 | 11 | 66 |
| 67–78 | 12 | 78 |
| 79–91 | 13 | 91 |
| 92–105 | 14 | 105 |
| 106–120 | 15 | 120 |
| 121–136 | 16 | 136 |
| 137–153 | 17 | 153 |
| 154–171 | 18 | 171 |
| 172–190 | 19 | 190 |
| 191–210 | 20 | 210 |
| 211–231 | 21 | 231 |
| 232–253 | 22 | 253 |
| 254–276 | 23 | 276 |
| 277–300 | 24 | 300 |

Observed shipped-house range for `kToaster.height` is **9..277** (81 distinct values),
so velocities 4..24 all occur.

**Half-rate (ball and fish)** — `GliderPRO/Sources/Dynamics3.c:472-484` and `:522-534`:

```
position = who->data.h.length;      // (kFish reads the same bytes via data.g.height)
velocity = 0;
evenFrame = true;                   // NOTE: writes the global
lilFrame  = true;
do {
    if (lilFrame) velocity++;       // gravity applies only every other iteration
    lilFrame = !lilFrame;
    position -= velocity;
} while (position > 0);
vVel  = -velocity;
count = -velocity;
```

The iteration sequence of `velocity` is 1, 1, 2, 2, 3, 3, … and the accumulated
`position` decrement after `k` iterations is 1, 2, 4, 6, 9, 12, 16, 20, 25, 30, 36, 42,
49, 56, 64, … (i.e. `floor((k+1)^2/4)` — a "quarter-squares" sequence). Solutions:

| `length` range | `velocity` |
|---|---:|
| 1 | 1 |
| 2 | 1 |
| 3–4 | 2 |
| 5–6 | 2 |
| 7–9 | 3 |
| 10–12 | 3 |
| 13–16 | 4 |
| 17–20 | 4 |
| 21–25 | 5 |
| 26–30 | 5 |
| 31–36 | 6 |
| 37–42 | 6 |
| 43–49 | 7 |
| 50–56 | 7 |
| 57–64 | 8 |
| 65–72 | 8 |
| 73–81 | 9 |
| 82–90 | 9 |
| 91–100 | 10 |
| 101–110 | 10 |
| 111–121 | 11 |
| 122–132 | 11 |
| 133–144 | 12 |
| 145–156 | 12 |
| 157–169 | 13 |
| 170–182 | 13 |
| 183–196 | 14 |
| 197–210 | 14 |
| 211–225 | 15 |
| 226–240 | 15 |
| 241–256 | 16 |
| 257–272 | 16 |
| 273–289 | 17 |
| 290–306 | 17 |

Observed shipped range for `kBall.length` is **0..281** and for `kFish.length`
**0..261**, i.e. velocities 1..17.

**`length == 0` is a degenerate case that a port must handle identically.** With
`position = 0` the `do…while` still runs its body once (`do…while (position > 0)`), so
`velocity` becomes 1 and `position` becomes −1, terminating. Result: `vVel = -1`,
`count = -1`. A `while` loop instead of a `do…while` would give `velocity = 0` and
produce a permanently stationary ball. Observed shipped data does contain
`kBall.length = 0` and `kFish.length = 0`.

### 8.4 Delay→frames conversion

Every periodic hazard converts its byte `delay` to frames the same way:

```
count = ((short)delay * 6) / kTicksPerFrame;      // kTicksPerFrame == 2  →  delay * 3
```

Sites: balloon `GliderPRO/Sources/Dynamics3.c:401`, copter `:426`, dart `:456`, drip
`:504`, fish `:521` (into `hVel`), outlet `:316`. The toaster is the exception — it uses
`frame = (short)who->data.g.delay * 3` directly (`:234`), which is numerically the same
but bypasses `kTicksPerFrame`.

`delay` is a `Byte`, validated 0..255 in the editor
(`GliderPRO/Sources/ObjectInfo.c:2182-2285` for enemies, `~1630-1740` for appliances),
so `count` ranges **0..765 frames** ≈ 0..25.4 s.

Trigger delay uses a different constant: `triggers[where].timer =
masterObjects[whoLinked].theObject.data.e.delay * 3` (`GliderPRO/Sources/Triggers.c:49`),
where `data.e` is `switchType` and `delay` there is a `short` at payload offset 4.

---

## 9. Player collision primitives

### 9.1 `CheckDynamicCollision` — player vs a moving hazard

`CheckDynamicCollision(short who, gliderPtr thisGlider, Boolean doOffset)`
(`GliderPRO/Sources/Dynamics.c:34-74`):

```
 1. dinahRect = dinahs[who].dest;
 2. if (doOffset) QOffsetRect(&dinahRect, -playOriginH, -playOriginV);
 3. if (!SectGlider(thisGlider, &dinahRect, true))  return;      // scrutinize = TRUE
 4. if (thisGlider->mode is NOT one of
        kGliderNormal(0), kGliderFaceLeft(7), kGliderFaceRight(8),
        kGliderBurning(9), kGliderGoingFoil(18), kGliderLosingFoil(19))
        return;                                                  // Dynamics.c:44-49
 5. if ((foilTotal > 0) || (thisGlider->mode == kGliderLosingFoil))
 6.     thisGlider->hDesiredVel = IsRectLeftOfRect(&dinahRect, &thisGlider->dest)
                                  ?  kShoveVelocity      //  +8
                                  : -kShoveVelocity;     //  -8
 7.     if (dinahs[who].vVel < 0)  thisGlider->vDesiredVel = dinahs[who].vVel;
 8.     PlayPrioritySound(kFoilHitSound, kFoilHitPriority);
 9.     if ((evenFrame) && (foilTotal > 0)) {
10.         foilTotal--;
11.         if (foilTotal <= 0)  StartGliderFoilLosing(thisGlider);
12.     }
13. else {
14.     StartGliderFadingOut(thisGlider);
15.     PlayPrioritySound(kFadeOutSound, kFadeOutPriority);        // DEATH
16. }
```

Notes:

* `doOffset` is `false` for every hazard except `kOutlet`, whose `dest` was stored in
  work-map (playOrigin-included) coordinates by `AddDynamicObject`
  (`GliderPRO/Sources/Dynamics3.c:310-312`); `HandleOutlet` passes `true`
  (`GliderPRO/Sources/Dynamics.c:545-550`).
* Step 4's mode filter means a hazard **cannot** hurt a glider that is fading in/out,
  going up/down stairs, transporting, ducting, being mailed, shredding, in limbo or
  idle. A **burning** glider *can* still be hit and killed.
* Step 7 makes a *rising* hazard transfer its upward velocity to a foil-clad glider
  (balloon rising at −2, ball/fish/toast on the way up). A descending hazard does not
  push the glider down.
* Step 9's `evenFrame` gate halves the foil-drain rate to one charge per two frames
  while in continuous contact. `kFoilSupply` = 8 → 16 frames ≈ 0.53 s of grinding
  against a hazard.
* `IsRectLeftOfRect(a, b)` decides which way to shove: if the *hazard* is to the left,
  the glider is shoved **right** (+8).

### 9.2 `SectGlider` — the scrutinized overlap test

`SectGlider(gliderPtr thisGlider, Rect *theRect, Boolean scrutinize)`
(`GliderPRO/Sources/Interactions.c:101-130`):

```
1. glideBounds = thisGlider->dest;
2. if (thisGlider->mode == kGliderBurning)  glideBounds.top += 6;
3. if (scrutinize)  InsetRect(&glideBounds, 5, 5);       // 5 px on all four sides
4. return plain rect overlap of glideBounds and *theRect;
```

A normal glider is 48×20; scrutinized it is **38×10**. A burning glider is 48×26 with
`top += 6` → 48×20 effective, scrutinized 38×10. The inset makes contact more forgiving:
the player must be 5 px *inside* the hazard before it registers.

`doScrutinize` per hazard hot spot (from `CreateActiveRects`):

| Hot spot | `doScrutinize` | Cite |
|---|---|---|
| flame column (`kLiftIt`, `kBurnIt`) | **false** | `GliderPRO/Sources/ObjectRects.c:410,413,416` |
| flame body (`kDissolveIt`) | true | `:421` |
| fan body (`kDissolveIt`) | true | `:374` (left), `:389` (right) |
| fan push column | false | `:381` |
| `kShredder` (`kShredIt`) | true | `:948` |
| `kGuitar` (`kStrumIt`) | false | `:956` |
| `kOutlet` (`kIgnoreIt`) | false | `:965` |
| `kMicrowave` body + column | true | `:975, :978` |
| appliance bodies (`kDissolveIt`) | true | `:995` |
| enemy placeholders (`kIgnoreIt`) | false | `:1013` |
| static `kFish` (`kDissolveIt`) | true | `:1022` |
| `kCobweb` (`kWebIt`) | true | `:1032` |
| `kChimes` (`kChimeIt`) | false | `:1058` |
| dynamic hazards via `CheckDynamicCollision` | **always true** | `GliderPRO/Sources/Dynamics.c:42` |

### 9.3 `GliderInRect` — full containment

`GliderInRect(gliderPtr thisGlider, Rect *theRect)`
(`GliderPRO/Sources/Interactions.c:134-150`) requires the glider rect to be **entirely
inside** `theRect`. It uses `thisGlider->dest` **raw** — no 5-px inset and, unlike
`SectGlider`, **no `kGliderBurning` `top += 6` adjustment**. Among the hazards it is used
by `kShredIt` (`:1326`), `kMicrowaveIt` (`:1582`) and `kWebIt` (`:1603` and again at
`:1607` for the burning-glider arm) — you must be fully inside a shredder / microwave
column / cobweb to trigger it. It is also used by the non-hazard transport/mailbox cases
(`:1251`, `:1286`, `:1388`, `:1431`, `:1474`, `:1517`, `:1550`) and by `WebGlider` itself
(`:1741`).

### 9.4 `GliderHitTop` — the foil bounce discriminator

`GliderHitTop(gliderPtr thisGlider, Rect *theRect)`
(`GliderPRO/Sources/Interactions.c:54-97`):

```
 1. glideBounds = thisGlider->dest;  InsetRect(&glideBounds, 5, 5);
 2. glideBounds.left -= thisGlider->wasHVel;   // rewind horizontally one frame
    glideBounds.right -= thisGlider->wasHVel;
 3. hitTop = (overlap tests conclude the contact was with theRect's TOP edge)
 4. if (!hitTop) {
 5.     PlayPrioritySound(kFoilHitSound, kFoilHitPriority);
 6.     foilTotal--;
 7.     if (foilTotal <= 0) StartGliderFoilLosing(thisGlider);
 8.     glideBounds.left += wasHVel; glideBounds.right += wasHVel;   // un-rewind
 9.     if (thisGlider->hVel > 0) offset = 2 + glideBounds.right - theRect->left;
    else                     offset = 2 + glideBounds.left  - theRect->right;
10.     thisGlider->hVel = -thisGlider->hVel - offset;
11. }
12. return hitTop;
```

Step 10 reverses the horizontal velocity **and** adds a separation term so the glider is
pushed clear of the obstacle. This is the "clang off a toaster while wearing foil"
behaviour. `GliderHitTop` returning `true` means the glider landed on top of the
hazard — and for `kDissolveIt` that is treated as **fatal even with foil**
(`GliderPRO/Sources/Interactions.c:1218-1244`), because you cannot stand on a lethal
appliance.

Note that steps 5-7 make `GliderHitTop` a **mutating** predicate: it plays a sound and
spends a foil charge as a side effect of returning `false`. Its only caller,
`case kDissolveIt:`, then spends a *second* charge (`:1230-1235`), so a foil-clad side
impact on a solid body costs 2 charges per frame. See §7.

### 9.5 `BounceGlider`

`BounceGlider(gliderPtr thisGlider, Rect *theRect)`
(`GliderPRO/Sources/Interactions.c:154-167`):

```
1. glideBounds = thisGlider->dest;      // NO inset - the raw glider rect
2. if (glider is nearer theRect's left edge)
       thisGlider->hVel = theRect->right - glideBounds.left;
   else
       thisGlider->hVel = theRect->left  - glideBounds.right;
3. PlayPrioritySound(foilTotal > 0 ? kFoilHitSound : kHitWallSound, ...);
```

Used only by `kBounceIt` (object `kInvisBounce` = 0x1F) and by
`CheckBandCollision`'s obstacle rebound.

### 9.6 `CheckForHotSpots` — the driver

`CheckForHotSpots()` (`GliderPRO/Sources/Interactions.c:1627-1687`):

```
for (i = 0; i < nHotSpots; i++)
{
    if (!hotSpots[i].isOn)  continue;
    if (twoPlayerGame) {
        ... test theGlider and/or theGlider2 depending on onePlayerLeft/playerDead;
        call HandleHotSpotCollision for each that overlaps;
        clear hotSpots[i].stillOver only if NEITHER call was made;
        // precise: the local `hitObject` is set only when HandleHotSpotCollision
        // actually ran, so an overlap by the *dead* player (skipped by the
        // onePlayerLeft/playerDead guards at :1642-1649, :1660-1667) still leaves
        // hitObject false and therefore clears stillOver.
    } else {
        if (SectGlider(&theGlider, &hotSpots[i].bounds, hotSpots[i].doScrutinize))
            HandleHotSpotCollision(&theGlider, &hotSpots[i], i);
        else
            hotSpots[i].stillOver = false;
    }
}
```

`HandleInteraction()` (`GliderPRO/Sources/Interactions.c:1691-1711`) is
`CheckForHotSpots()` followed by `CheckGliderInRoom()` per live glider.

`FlagStillOvers(gliderPtr)` (`GliderPRO/Sources/Interactions.c:1715-1732`) is called on
room entry and sets `hotSpots[i].stillOver = isOn && SectGlider(...)`, so a glider that
materialises already inside a switch or trigger does not fire it.

---

## 10. The dispatch tables

### 10.1 `HandleDynamics` (`GliderPRO/Sources/Dynamics3.c:31-105`)

```
for (i = 0; i < numDynamics; i++)
    switch (dinahs[i].type) {
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
```

### 10.2 `RenderDynamics` (`GliderPRO/Sources/Dynamics3.c:112-154`)

Only **seven** types render:

```
for (i = 0; i < numDynamics; i++)
    switch (dinahs[i].type) {
        case kToaster:   RenderToast(i);    break;
        case kBalloon:   RenderBalloon(i);  break;
        case kCopterLf:
        case kCopterRt:  RenderCopter(i);   break;
        case kDartLf:
        case kDartRt:    RenderDart(i);     break;
        case kBall:      RenderBall(i);     break;
        case kDrip:      RenderDrip(i);     break;
        case kFish:      RenderFish(i);     break;
    }
```

The appliance dynamics (`kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`,
`kMicrowave`) draw themselves *inside their `Handle*` function*, straight into
`backSrcMap` and/or `workSrcMap`, because they are state changes on a static background
rather than moving sprites.

---

## 11. Hazard reference: the six mobile enemies

### 11.1 `kBalloon` (0x71) — rising balloon

**Art.** PICT 4011 / mask 5011, 24×240, 8 frames of 24×30 (`balloonSrc[0..7]`).
Frames **0..5** are the intact rising balloon; frames **6..7** are the popped/falling
balloon. `DrawBalloon` (the editor/static drawing path) uses `balloonSrc[1]`
(`GliderPRO/Sources/ObjectDraw2.c:914-920`) — but in play the balloon is never statically
drawn.

**Fields.** `data.h.topLeft.h` = spawn column (`topLeft.v` is **ignored**);
`data.h.length` unused (observed always 0); `data.h.delay` = respawn period byte
(observed 0..255, 40 distinct values); `data.h.byte0` unused (always 0);
`initial`/`state` = on/off.

**Spawn** (`GliderPRO/Sources/Dynamics3.c:391-410`):

```
dest       = balloonSrc[0]; ZeroRectCorner(dest);      // 24x30 at (0,0)
dest      += (where->left, 0)                          // horizontal only
dest.bottom = kBalloonStart  = 310
dest.top    = 310 - 30 = 280
whole  = dest
hVel   = 0
vVel   = -2
count  = (delay * 6) / 2 = delay * 3
frame  = 0
timer  = count
position = 0        (unused)
moving = false      (starts idle)
active = state
```

**`HandleBalloon`** (`GliderPRO/Sources/Dynamics2.c:30-130`), numbered against the C:

```
IF moving:
  A. IF vVel < 0  (still rising, i.e. not popped):
     A1. if (evenFrame) { frame++; if (frame >= 6) frame = 0; }        // 15 fps, 6 frames
     A2. CheckDynamicCollision(who, <each live glider>, false);
     A3. if ((numBands > 0) && DidBandHitDynamic(who)):
              frame = 6;  vVel = kEnemyDropSpeed (8);
              PlayPrioritySound(kPopSound, kPopPriority);
         else:
              VOffsetRect(dest, vVel);            // rise 2 px per frame
              whole = dest;  whole.bottom -= vVel;   // dirty rect grows downward
  B. ELSE (vVel >= 0, popped and falling):
     B1. if (evenFrame) { frame++; if (frame >= 8) frame = 6; }        // frames 6,7 loop
     B2. VOffsetRect(dest, vVel);                 // fall 8 px per frame
     B3. whole = dest;  whole.top -= vVel;
  C. IF (dest.top <= kBalloonStop (8)) OR (dest.bottom >= kBalloonStart (310)):
     C1. AddRectToWorkRects(whole + playOrigin);
     C2. AddSparkle(&dest);
     C3. PlayPrioritySound(kEnemyOutSound, kEnemyOutPriority);
     C4. moving = false;  vVel = -2;  timer = count;
     C5. dest.bottom = 310;  dest.top = 310 - RectTall(balloonSrc[0]);
     C6. whole = dest;
ELSE (idle):
  D. IF active:
     D1. timer--;
     D2. if (timer <= 0) { moving = true;
             if (count < kStartSparkle (4)) { AddSparkle(&dest);
                 PlayPrioritySound(kEnemyInSound, kEnemyInPriority); } }
     D3. else if (timer == kStartSparkle (4)) { AddSparkle(&dest);
             PlayPrioritySound(kEnemyInSound, kEnemyInPriority); }
```

**Timings.** Rise speed 2 px/frame; the trip from `dest.top` = 280 to `dest.top` = 8 is
272 px = **136 frames ≈ 4.52 s**. After a pop, `vVel = 8` and the balloon needs
`ceil((310 - dest.bottom) / 8)` frames to clear the bottom line. Note the pop **does not
reset the horizontal position** — a balloon always occupies the same column.

**Popping.** A rubber band hit sets `frame = 6` and `vVel = +8` immediately, so the same
frame's `whole` update happens through branch B on the *next* frame. This is the only
enemy where the pop sound (`kPopSound`) differs from the paper-crunch used by
copters/darts.

**Materialisation sparkle logic (D2/D3) is subtle and easy to get wrong.** The
`kEnemyInSound` + sparkle plays *4 frames before* the balloon appears when
`count >= 4`, and *at the moment of appearance* when `count < 4`. `count < 4` means
`delay * 3 < 4`, i.e. `delay <= 1`. With `delay == 0` the balloon respawns
instantaneously every frame it is off-screen (`timer = 0`, then `timer--` → −1 → `<= 0`
→ `moving = true`), so a `delay == 0` balloon is effectively continuous — and the census
shows shipped houses do use `delay = 0`.

### 11.2 `kCopterLf` (0x72) / `kCopterRt` (0x73) — descending helicopter

**Art.** PICT 4012 / mask 5012, 32×300, 10 frames of 32×30 (`copterSrc[0..9]`).
Frames **0..7** flying, frames **8..9** shot/falling. Static/editor draw uses
`copterSrc[1]` (`GliderPRO/Sources/ObjectDraw2.c:924-930`).

**Fields.** `topLeft.h` = spawn column (`topLeft.v` **ignored**); `delay` = respawn
period byte (observed kCopterLf 0..255 / 37 distinct, kCopterRt 0..255 / 29 distinct);
`length`, `byte0` unused.

**Spawn** (`GliderPRO/Sources/Dynamics3.c:412-435`):

```
dest       = copterSrc[0]; ZeroRectCorner;   dest += (where->left, 0)
dest.top    = kCopterStart = 8
dest.bottom = 8 + 30 = 38
hVel   = (what == kCopterLf) ? -1 : +1
vVel   = 2
count  = delay * 3
frame  = 0
timer  = count
position = dest.left            // remembered spawn column, used on respawn
moving = false
active = state
```

**`HandleCopter`** (`GliderPRO/Sources/Dynamics2.c:134-238`):

```
IF moving:
  A. IF hVel != 0 (not shot):
     A1. frame++; if (frame >= 8) frame = 0;      // EVERY frame -> 30 fps, 8 frames
     A2. CheckDynamicCollision(who, <each live glider>, false);
     A3. if ((numBands > 0) && DidBandHitDynamic(who)):
              frame = 8;  hVel = 0;  vVel = kEnemyDropSpeed (8);
              PlayPrioritySound(kPaperCrunchSound, kPaperCrunchPriority);
         else:
              HOffsetRect(dest, hVel);  VOffsetRect(dest, vVel);
              whole = dest;  whole.top -= vVel;
              if (hVel < 0) whole.right -= hVel;  else whole.left -= hVel;
  B. ELSE (shot, falling straight down):
     B1. frame++; if (frame >= 10) frame = 8;     // frames 8,9 loop at 30 fps
     B2. VOffsetRect(dest, vVel);  whole = dest;  whole.top -= vVel;
  C. IF (dest.top <= kCopterStart (8)) OR (dest.bottom >= kCopterStop (310)):
     C1. AddRectToWorkRects(whole + playOrigin);
     C2. AddSparkle(&dest);
     C3. PlayPrioritySound(kEnemyOutSound, kEnemyOutPriority);
     C4. moving = false;  vVel = 2;
         hVel = (type == kCopterLf) ? -1 : +1;
         timer = count;
     C5. dest.top = 8;  dest.bottom = 8 + RectTall(copterSrc[0]);
         dest.left = position;  dest.right = dest.left + 32;      // HARDCODED 32
     C6. whole = dest;
ELSE (idle): identical to the balloon's branch D.
```

**Trajectory.** The copter drifts diagonally: 1 px horizontally per frame in its facing
direction, 2 px down per frame. From `dest.top` = 8 to `dest.bottom` = 310 is
`310 - 38` = 272 px of descent = **136 frames ≈ 4.52 s**, during which it drifts 136 px
horizontally. There is **no wall check** — the copter is allowed to walk off the side of
the room and keeps going until its bottom reaches 310. It is only clipped by the
blitter.

**Animation is full-rate**, unlike the balloon, so 8 frames cycle in 8 frames
(≈0.27 s).

**`dest.right = dest.left + 32`** at C5 is a hardcoded 32 rather than
`RectWide(&copterSrc[0])`, which is also 32 — noted so a port does not "fix" it into a
behavioural difference if the art is ever resized.

### 11.3 `kDartLf` (0x74) / `kDartRt` (0x75) — horizontal dart

**Art.** PICT 4013 / mask 5013, 64×76, 4 frames of 64×19 (`dartSrc[0..3]`).
Frame **0** = intact left-flying, frame **1** = shot left-flying, frame **2** = intact
right-flying, frame **3** = shot right-flying. There is **no animation at all** — the
frame index never changes except when the dart is shot. Static/editor draw uses
`dartSrc[0]` for `kDartLf` and `dartSrc[2]` otherwise
(`GliderPRO/Sources/ObjectDraw2.c:934-950`).

**Fields.** `topLeft.v` = spawn row (`topLeft.h` **ignored** at runtime; the editor
forces it to `kRoomWide - RectWide(srcRects[what])` = 448 for `kDartLf` and 0 for
`kDartRt`, `GliderPRO/Sources/ObjectAdd.c:646-693`); `delay` = respawn period byte
(observed kDartLf 0..53 / 28 distinct, kDartRt 2..255 / 25 distinct); `length`, `byte0`
unused.

**Spawn** (`GliderPRO/Sources/Dynamics3.c:437-463`):

```
dest = dartSrc[0]; ZeroRectCorner;                    // 64x19 at (0,0)
IF what == kDartLf:
    QOffsetRect(dest, kRoomWide - RectWide(dartSrc[0]), where->top)   // (512-64, top) = (448, top)
    hVel  = -kDartVelocity = -6
    frame = 0
ELSE:
    QOffsetRect(dest, 0, where->top)
    hVel  = +6
    frame = 2
vVel     = 2
count    = delay * 3
timer    = count
position = dest.top          // remembered spawn row
moving   = false
active   = state
// NOTE: this case never assigns dinahs[].room
```

**`HandleDart`** (`GliderPRO/Sources/Dynamics2.c:242-354`):

```
IF moving:
  A. IF hVel != 0 (not shot):
     A1. (no frame animation)
     A2. CheckDynamicCollision(who, <each live glider>, false);
     A3. if ((numBands > 0) && DidBandHitDynamic(who)):
              frame = (type == kDartLf) ? 1 : 3;
              hVel = 0;  vVel = kEnemyDropSpeed (8);
              PlayPrioritySound(kPaperCrunchSound, kPaperCrunchPriority);
         else:
              HOffsetRect(dest, hVel);  VOffsetRect(dest, vVel);
              whole = dest;  whole.top -= vVel;
              if (hVel < 0) whole.right -= hVel;  else whole.left -= hVel;
  B. ELSE (shot): VOffsetRect(dest, vVel); whole = dest; whole.top -= vVel;
  C. IF (dest.left <= 0) OR (dest.right >= kRoomWide (512)) OR
        (dest.bottom >= kDartStop (310)):
     C1. AddRectToWorkRects(whole + playOrigin);
     C2. AddSparkle(&dest);
     C3. PlayPrioritySound(kEnemyOutSound, kEnemyOutPriority);
     C4. moving = false;  vVel = 2;
     C5. IF type == kDartLf:  frame = 0; hVel = -6;
                              dest.right = 512; dest.left = 512 - RectWide(dartSrc[0]);
         ELSE:                frame = 2; hVel = +6;
                              dest.left = 0;   dest.right = 0 + RectWide(dartSrc[0]);
     C6. timer = count;  dest.top = position;  dest.bottom = dest.top + RectTall(dartSrc[0]);
     C7. whole = dest;
ELSE (idle): identical to the balloon's branch D.
```

**Trajectory.** The dart moves **6 px horizontally and 2 px down** per frame — it is a
shallow glide, not a straight line. Crossing the room takes
`(512 - 64) / 6` = 74.67 → **75 frames ≈ 2.49 s**, during which it also drops 150 px.
Because `dest.bottom >= 310` is also a respawn trigger, a dart spawned low enough exits
through the floor line before reaching the far wall. Simulating the exact loop
(move first, then test) the crossover is at **spawn row ≥ 143**: rows ≤ 140 despawn on
the wall at frame 75, rows 141-142 satisfy the wall *and* floor tests on the same frame
75, and rows ≥ 143 despawn on the floor at frame 74 or earlier.

The respawn condition `dest.left <= 0` fires for a *left*-flying dart, and
`dest.right >= 512` for a right-flying one — but both tests are evaluated for both
directions, which is harmless because a left-flying dart starts at `right == 512` and…
**actually it is not harmless: a `kDartLf` spawns with `dest.right = 512` exactly**, and
`512 >= kRoomWide` is true. The saving grace is that condition C is only evaluated
*after* branch A/B has already moved the dart by `hVel = -6`, so on the first moving
frame `dest.right` is 506 and the test fails. A port that evaluates the boundary test
before the move will make every left dart despawn immediately.

Symmetrically `kDartRt` spawns at `dest.left = 0` and `0 <= 0` is true — the same
ordering argument applies (first move makes `left = 6`).

### 11.4 `kBall` (0x76) — bouncing ball

**Art.** PICT 4014 / mask 5014, 32×64, 2 frames of 32×32 (`ballSrc[0..1]`). Frame
**0** = airborne, frame **1** = squashed/bouncing. Static/editor draw uses
`ballSrcMap` with `srcRects[what]` (`GliderPRO/Sources/ObjectDraw2.c:954-960`).

**Fields.** `topLeft` = resting position (the ball's floor is `topLeft.v + 32`);
`data.h.length` = bounce **height** in pixels, solved into a velocity by the half-rate
loop (observed 0..281, 115 distinct values); **`delay` is always 0** — the editor hides
the delay field entirely for `kBall` (`GliderPRO/Sources/ObjectInfo.c:2212-2217`) and the
census confirms `delay == 0` in every shipped ball. The editor default is
`length = 64` (`GliderPRO/Sources/ObjectAdd.c:695-720`).

**Spawn** (`GliderPRO/Sources/Dynamics3.c:465-494`):

```
dest = ballSrc[0]; ZeroRectCorner;  dest += (where->left, where->top)
whole = dest
hVel = 0
<half-rate solver on data.h.length>  ->  velocity
vVel     = -velocity
moving   = false
count    = -velocity            // NEGATIVE: the bounce impulse
frame    = 0
timer    = 0
position = dest.bottom          // the floor line
active   = state
```

**`HandleBall`** (`GliderPRO/Sources/Dynamics2.c:358-423`):

```
 1. CheckDynamicCollision(who, <each live glider>, false);   // ALWAYS, even when idle!
 2. IF moving:
 3.    VOffsetRect(dest, vVel);
 4.    IF dest.bottom >= position:                            // hit the floor line
 5.       whole = dest;  whole.top -= vVel;  whole.bottom = position;
 6.       dest.bottom = position;  dest.top = dest.bottom - 32;
 7.       IF active:  vVel = count;                           // full-energy re-launch
 8.       ELSE:       vVel = -((vVel * 3) / 4);               // 3/4 damping
 9.                   if (vVel == 0)  moving = false;
10.       if (whole.bottom < dest.bottom)  whole.bottom = dest.bottom;
11.       PlayPrioritySound(kBounceSound, kBouncePriority);
12.       if (moving)  frame = 1;
13.    ELSE (airborne):
14.       whole = dest;
15.       if (vVel > 0)  whole.top    -= vVel;
          else           whole.bottom -= vVel;
16.       if (evenFrame)  vVel++;                             // HALF gravity
17.       frame = 0;
18. ELSE (idle):
19.    IF active:  vVel = count;  moving = true;  evenFrame = true;
```

Key behaviours:

* **A ball is lethal even while sitting still**, because step 1 runs unconditionally.
  It is the only hazard with this property, and it is why balls make good static
  obstacles.
* An **active** ball bounces forever at constant height (step 7). Deactivating it
  (via a switch → `ToggleBall`) makes it **decay by ×3/4 each bounce** (step 8) until
  `vVel` truncates to 0 and it stops. Re-activating restarts it at full height
  (step 19).
* Step 8's `-((vVel * 3) / 4)` is **C integer division truncating toward zero** on a
  *positive* `vVel` (the ball is moving down when it hits), so `vVel = 5` → `-(15/4)` =
  `-3`. A Go port using `/` on ints matches; a port using floats and `math.Floor` does
  not.
* Step 16 makes ball gravity **half** that of toast: +1 to `vVel` every other frame.
* Step 19 writes the **global** `evenFrame = true` (`GliderPRO/Sources/Dynamics2.c:420`).
* `RenderBall` has **no `moving` gate** (`GliderPRO/Sources/Dynamics.c:215-232`) — the
  ball is always composited.

**Bounce period.** With half-rate gravity, an initial `vVel = -v` reaches apex in `2v`
frames and returns in another `2v`, so the period is ≈`4v` frames. For the default
`length = 64` → `v = 8` → ≈32 frames ≈ 1.06 s.

### 11.5 `kDrip` (0x77) — dripping water

**Art.** PICT 4015 / mask 5015, 16×72, 6 frames of 16×12 (`dripSrc[0..5]`). Frames
**0, 1, 2** are the swelling droplet on the ceiling, frame **3** is the idle/empty
state, frames **4** and **5** are the falling droplet (alternating). Static/editor draw
uses `dripSrc[3]` (`GliderPRO/Sources/ObjectDraw2.c:974-980`), i.e. the idle art.

**Fields.** `topLeft` = the ceiling attachment point; `data.h.length` = **fall
distance** in pixels (observed 1..310, 178 distinct values); `delay` = reload period byte
(observed 0..255, 40 distinct). The editor defaults are `length = 64` and
`delay = 10 + RandomInt(10)` (`GliderPRO/Sources/ObjectAdd.c:695-720`);
`HouseLegal.c` clamps `length` so the splash line stays inside the room:
`if ((what == kDrip) && (bounds.bottom + data.h.length > kTileHigh)) data.h.length =
kTileHigh - bounds.bottom;` with `kTileHigh` = 322.

**Spawn** (`GliderPRO/Sources/Dynamics3.c:496-514`):

```
dest = dripSrc[0];  CenterRectInRect(dest, where);
VOffsetRect(dest, where->top - dest.top);      // align tops
whole    = dest
hVel     = dest.top                            // remembered ceiling row
vVel     = 0
count    = delay * 3
frame    = 3                                   // idle art
timer    = count
position = dest.top + data.h.length            // the splash line
moving   = false
active   = state
```

**`HandleDrip`** (`GliderPRO/Sources/Dynamics2.c:427-497`):

```
IF moving:
  A1. if (evenFrame)  frame = 9 - frame;        // toggles 4 <-> 5 at 15 fps
  A2. CheckDynamicCollision(who, <each live glider>, false);
  A3. VOffsetRect(dest, vVel);
  A4. IF dest.bottom >= position:               // splash
        A4a. AddRectToWorkRects(whole + playOrigin);
        A4b. dest.top = hVel;  dest.bottom = dest.top + 12;    // teleport back up
        A4c. PlayPrioritySound(kDropSound, kDropPriority);
        A4d. vVel = 0;  timer = count;  frame = 3;  moving = false;
      ELSE:
        A4e. whole = dest;  whole.top -= vVel;
        A4f. if (evenFrame)  vVel++;            // HALF gravity
ELSE (idle):
  B1. IF active:
        B2. timer--;
        B3. if (timer == 6)  frame = 0;
        B4. if (timer == 4)  frame = 1;
        B5. if (timer == 2)  frame = 2;
        B6. if (timer <= 0) {
                VOffsetRect(dest, 3);           // detach: nudge down 3 px
                whole = dest;
                moving = true;  frame = 4;
                PlayPrioritySound(kDripSound, kDripPriority);
            }
```

Key behaviours:

* The 3-frame swell animation is driven **by absolute `timer` values 6, 4, 2**, not by a
  counter. If `count < 7` the swell frames are skipped (or partially skipped), and if
  `count == 0` the drip detaches every frame. `TriggerDrip`
  (`GliderPRO/Sources/Trip.c:188-192`) exploits this: it sets `timer = 7` if `timer > 7`,
  which forces the full 3-frame swell then a detach — i.e. a trigger cannot "instantly"
  drop a drip, it always plays a **7-frame** windup (`timer` = 7 → 6 → 5 → 4 → 3 → 2 → 1
  → 0, with the detach happening on the frame that decrements 1 → 0, so 7 frames after
  the trigger).
* `vVel = 0` at detach, so the first falling frame moves 0 px; the drip is nudged
  3 px on detach (B6) so it visibly separates.
* `RenderDrip` has **no `moving` gate** (`GliderPRO/Sources/Dynamics.c:236-253`) — the
  drip is always composited, so its idle frame 3 art is what you see on the ceiling.
* A drip is **lethal only while `moving`** in the sense that `CheckDynamicCollision`
  is called only in the moving branch — unlike the ball.
* Fall time for distance `d` at half gravity is ≈`2*sqrt(d)` frames: default
  `length = 64` → ≈16 frames ≈ 0.53 s.

### 11.6 `kFish` (0x78) — leaping fish

The fish is unusual: it has **both** a static 36×33 hot spot that kills on touch **and**
a 16×16 animated dynamic that leaps out of the water.

**Art.**
* Static: PICT 4016 (`kEnemyPictID`) / mask 5016, 36×33, drawn via
  `enemySrcMap`/`enemyMaskMap` with `srcRects[kFish]`
  (`GliderPRO/Sources/ObjectDraw2.c:964-970`). This is the "fish in the water" picture.
* Dynamic: PICT 4017 (`kFishPictID`) / mask 5017, 16×128, 8 frames of 16×16
  (`fishSrc[0..7]`). Frames **0..3** are the idle bob cycle, frames **4..7** are the
  airborne leap.

**Fields.** `topLeft` = the static art's position; `data.h.length` (== `data.g.height`)
= leap height in pixels (observed 0..261, 74 distinct); `delay` = reload period byte
(observed 0..255, 37 distinct). `HouseLegal.c` clamps
`if ((what == kBall || what == kFish) && (bounds.top - data.h.length < 0))
data.h.length = bounds.top;`.

**Spawn** (`GliderPRO/Sources/Dynamics3.c:516-544`):

```
dest = fishSrc[0];                              // already at origin, no ZeroRectCorner
QOffsetRect(dest, where->left + 10, where->top + 8);   // 16x16 inset into the 36x33 art
whole = dest
hVel  = (delay * 6) / 2 = delay * 3             // hVel holds the RELOAD PERIOD
<half-rate solver on data.g.height>  ->  velocity
vVel     = -velocity
count    = -velocity                            // NEGATIVE jump impulse
frame    = 0
timer    = hVel                                 // first countdown
position = dest.bottom                          // the water line
moving   = false
active   = state
```

Note `hVel` is repurposed as storage for the reload period because `timer` is consumed
by the idle bob animation.

**`HandleFish`** (`GliderPRO/Sources/Dynamics2.c:501-588`):

```
IF moving:
  A1. if ((vVel >= 0) && (frame < 7))  frame++;   // advance 4->7 only while DESCENDING
  A2. CheckDynamicCollision(who, <each live glider>, false);
  A3. VOffsetRect(dest, vVel);
  A4. IF dest.bottom >= position:                 // back in the water
        A4a. AddRectToWorkRects(whole + playOrigin);
        A4b. dest.bottom = position;  dest.top = dest.bottom - 16;
        A4c. whole = dest;  whole.top -= 2;
        A4d. PlayPrioritySound(kDropSound, kDropPriority);
        A4e. vVel = count;   // NEGATIVE: pre-load the next jump
             timer = hVel;   // reload the countdown
             frame = 0;  moving = false;
        A4f. PlayPrioritySound(kFishInSound, kFishInPriority);   // SECOND sound
      ELSE:
        A4g. whole = dest;
             if (vVel > 0)  whole.top    -= vVel;
             else           whole.bottom -= vVel;
        A4h. if (evenFrame)  vVel++;              // HALF gravity
ELSE (idle):
  B1. whole = dest;
  B2. IF (timer & 0x0003) == 0x0003:              // every 4th frame
        B3. frame++;  if (frame > 3)  frame = 0;
        B4. IF (frame == 1) OR (frame == 2):  dest.top++; dest.bottom++; whole.bottom++;
            ELSE:                             dest.top--; dest.bottom--; whole.top--;
  B5. IF active:
        B6. timer--;
        B7. if (timer <= 0) { whole = dest;  moving = true;  frame = 4;
                PlayPrioritySound(kFishOutSound, kFishOutPriority); }
```

Key behaviours:

* **Two sounds fire on re-entry**: `kDropSound` (priority 309) at A4d *and*
  `kFishInSound` (priority 412) at A4f. Both are emitted, on different channels; a port
  must play both.
* **Airborne animation only advances while falling** (A1's `vVel >= 0` test). The fish
  is frozen on frame 4 for the entire ascent, then walks 4 → 5 → 6 → 7 as it comes down
  and holds on 7.
* The **idle bob** (B2–B4) moves the fish 1 px down, 1 px down, 1 px up, 1 px up in a
  4-step cycle keyed off `timer & 3 == 3`, so it bobs once per 16 frames while
  `timer` is counting. Because `timer` also gates the leap, the bob phase depends on the
  reload period.
* The bob runs **whether or not the fish is active** (B2 is outside the `if (active)`),
  so a switched-off fish still bobs. Only the countdown at B6 is gated.
* `RenderFish` (`GliderPRO/Sources/Dynamics.c:257-286`) uses `CopyMask` while `moving`
  but a **plain opaque `CopyBits` with `srcCopy`** while idle (`:278-280`) — the idle
  fish's 16×16 rect is stamped over the background including its white pixels. A port
  must not use alpha blending for the idle fish, or the water will show through where
  the original showed opaque white.

---

## 12. Hazard reference: appliances

### 12.1 `kToaster` (0x62) — the toast launcher

Two separate hazards: the toaster *body* (a `kDissolveIt` hot spot on its 48×27 rect,
`GliderPRO/Sources/ObjectRects.c:981-996`) and the *flying toast* dynamic.

**Art.** PICT 4009 / mask 5009, 32×174, 6 frames of 32×29 (`breadSrc[0..5]`),
`kNumBreadPicts` = 6. The toaster itself comes from the 80×269 appliance sheet
(`srcRects[kToaster]` = 48×27 at (0,22)).

**Fields.** `data.g.height` = launch height in pixels (observed 9..277, 81 distinct);
`data.g.delay` = idle reload in units of 3 frames (observed 0..255, 27 distinct);
`data.g.byte0` unused (always 0). Editor defaults `height = 64`,
`delay = 10 + RandomInt(10)` (`GliderPRO/Sources/ObjectAdd.c:589-644`).
`HouseLegal.c` clamps: `if ((what == kToaster) && (bounds.top - data.g.height < 0))
data.g.height = bounds.top;`. `topLeft.h` is forced **even**.

**Spawn** (`GliderPRO/Sources/Dynamics3.c:217-242`):

```
dest = breadSrc[0];  CenterRectInRect(dest, where);   // centre 32x29 in the 48x27 toaster
VOffsetRect(dest, where->top - dest.top);             // align tops
whole    = dest
hVel     = where->top + 2         // CLIP LINE: toast is hidden below this
<full-rate solver on data.g.height>  ->  velocity
vVel     = -velocity
count    =  velocity              // POSITIVE
frame    = data.g.delay * 3       // doubles as the idle countdown
timer    = frame                  // remembered reload value
position = 0                      // unused
moving   = false
active   = state
```

**`HandleToast`** (`GliderPRO/Sources/Dynamics.c:321-384`):

```
IF moving:
  A1. if (evenFrame) { frame++; if (frame >= kNumBreadPicts (6)) frame = 0; }   // 15 fps
  A2. CheckDynamicCollision(who, <each live glider>, false);
  A3. VOffsetRect(dest, vVel);
  A4. whole = dest;
      if (vVel > 0)  whole.top    -= vVel;
      else           whole.bottom -= vVel;
  A5. vVel++;                                   // FULL gravity - not evenFrame gated!
  A6. IF vVel > count:                          // fallen faster than it launched
        A6a. AddRectToWorkRects(whole + playOrigin);
        A6b. moving = false;  frame = timer;     // reload the idle countdown
        A6c. PlayPrioritySound(kToastLandSound, kToastLandPriority);
ELSE (idle):
  B1. if (active)  frame--;
  B2. IF frame <= 0:
        B3. IF active: vVel = -count;  frame = 0;  moving = true;
                       PlayPrioritySound(kToastLaunchSound, kToastLaunchPriority);
        B4. ELSE:      frame = timer;
```

Key behaviours:

* **Toast gravity is full-rate** (A5, one `vVel++` per frame) whereas ball, drip and
  fish use half-rate. This is the single largest behavioural difference among the
  ballistic hazards and it is very easy to get wrong.
* The landing test is on **velocity**, not position (A6): the toast lands when its
  downward speed exceeds the launch speed. Because gravity is `+1`/frame and the launch
  is `-velocity`, the flight lasts exactly `2*velocity + 1` frames and the toast's net
  displacement over the whole flight is **exactly 0** — it lands back on the same
  `dest.top` it launched from. (Verified by simulating the A3/A5/A6 order for
  `velocity` = 1, 2, 4, 8, 17, 24: frames = `2v+1`, net = 0 in every case.) The clipping
  in `RenderToast` is unrelated to the landing test: it exists because the toast *spawns*
  with `dest.top == where->top` while the clip line is `where->top + 2`, so a resting
  toast is almost entirely hidden inside the toaster slot and becomes visible only as it
  rises.
* `RenderToast` (`GliderPRO/Sources/Dynamics.c:112-139`):
  ```
  src   = breadSrc[frame];
  vClip = dinahs[who].dest.bottom - dinahs[who].hVel;   // hVel = clip line
  if (vClip > 0) { src.bottom -= vClip; dest.bottom -= vClip; }
  CopyMask(toastSrcMap, toastMaskMap, workSrcMap, &src, &src, &dest);
  AddRectToBackRects(&dest);
  AddRectToWorkRects(whole + playOrigin);
  ```
  So the toast is progressively hidden as it sinks below `topLeft.v + 2`, giving the
  illusion of popping out of the slot. `RenderToast` is gated on `moving`, so idle toast
  is invisible.
* `frame` is doing double duty: 0..5 while flying (animation) and a countdown while
  idle. On landing it is reset to `timer` (the stored `delay * 3`).
* An **inactive** toaster still counts down harmlessly (B1 is gated on `active`, so
  actually it does not count down at all; B4 just re-arms `frame = timer` each frame).
  Combined with `TriggerToast` (`GliderPRO/Sources/Trip.c:153-167`), which also checks
  `active`, a switched-off toaster is completely inert.
* Editor limit: an appliance add is rejected with `ShoutNoMoreSpecialObjects()` if
  `HowManyShredderObjects() >= kMaxShredded` (4) and the type is not one of
  `kGuitar`/`kCinderBlock`/`kFlowerBox`/`kCDs`/`kCustomPict`
  (`GliderPRO/Sources/ObjectAdd.c:589-644`). `HowManyShredderObjects`
  (`:1031-1041`) counts only `kShredder`, so this limits shredders to 4 per room but
  gates *all* hazardous appliance additions behind that count.

### 12.2 `kOutlet` (0x67) — electric arc

**Art.** `outletSrc[0..3]` = 16×24 at `(64, 22 + 24i)` in the 80×269 appliance sheet
(`GliderPRO/Sources/StructuresInit.c:591-593`), `kNumOutletPicts` = 4. Frame **0** is the
quiescent outlet; frames **1..3** are the arc animation.

**Fields.** `data.g.height` = 0 (editor default, `GliderPRO/Sources/ObjectAdd.c:589-644`);
`data.g.delay` = idle period byte (observed 0..240, 27 distinct); `byte0` unused.
`topLeft.h` forced **even**.

**Hot spot.** `kIgnoreIt` on the 16×24 sprite rect, gated on `data.g.state`
(`GliderPRO/Sources/ObjectRects.c:959-967`). The outlet does **not** kill via a hot
spot — the kill happens inside `HandleOutlet` via `CheckDynamicCollision`.

**Spawn** (`GliderPRO/Sources/Dynamics3.c:307-325`) — note the `dest` includes
`playOrigin`. This is true of **all seven appliance dinahs** — `kMacPlus` (`:247-249`),
`kTV` (`:267-269`), `kCoffee` (`:287-289`), `kOutlet` (`:310-312`), `kVCR` (`:330-332`),
`kStereo` (`:353-355`), `kMicrowave` (`:373-375`) — and of **none** of the other dinah
types (`kSparkle` `:202`, `kToaster` `:219-221`, and the six mobile enemies all offset by
`where` alone). The outlet is only the one that *matters* for collision, because it is
the only playOrigin-inclusive dinah that calls `CheckDynamicCollision`:

```
dest = outletSrc[0]; ZeroRectCorner;
QOffsetRect(dest, where->left + playOriginH, where->top + playOriginV);
hVel     = numLights          // number of lights currently on in the room
vVel     = 0
count    = (data.g.delay * 6) / 2
frame    = 0
timer    = count
position = 0                  // 0 = idle, 1 = zapping
active   = state
```

**`HandleOutlet`** (`GliderPRO/Sources/Dynamics.c:528-599`):

```
IF position != 0 (zapping):
  A1. timer--;
  A2. IF twoPlayerGame AND onePlayerLeft:
          CheckDynamicCollision(who, <the surviving glider>, FALSE);  // doOffset = FALSE — see below
      ELSE IF twoPlayerGame:
          CheckDynamicCollision(who, &theGlider,  TRUE);
          CheckDynamicCollision(who, &theGlider2, TRUE);
      ELSE:
          CheckDynamicCollision(who, &theGlider,  TRUE);
  A3. IF timer <= 0:  frame = 0;  position = 0;  timer = count;
      ELSE:
        A3a. if ((timer % 5) == 0)  PlayPrioritySound(kZapSound, kZapPriority);
        A3b. frame++;  if (frame >= kNumOutletPicts (4))  frame = 1;
  A4. IF (position != 0) OR (hVel > 0):
          CopyBits(applianceSrcMap -> workSrcMap, &outletSrc[frame], &dest, srcCopy, nil);
      ELSE
          PaintRect(&dest);          // dark room: paint black (see caveat below)
  A5. AddRectToWorkRects(&dest);
ELSE (idle):
  B1. if (active)  timer--;
  B2. IF timer <= 0:
        B3. IF active: position = 1;  timer = kLengthOfZap (30);
                       PlayPrioritySound(kZapSound, kZapPriority);
        B4. ELSE:      timer = count;
```

Key behaviours:

* An arc lasts exactly **`kLengthOfZap` = 30 frames ≈ 1 s**, during which the
  4-frame arc animation cycles at full rate (frames 1,2,3,1,2,3,…) and `kZapSound`
  re-fires every 5 frames (`timer % 5 == 0`, so at `timer` = 25, 20, 15, 10, 5 — five
  extra plays on top of the initial one at B3).
* Collision uses `doOffset = TRUE` because of the playOrigin-inclusive `dest` — **except**
  in the two-player `onePlayerLeft` branch (`GliderPRO/Sources/Dynamics.c:536-542`), which
  passes `false`. That looks like a **bug**: the surviving glider's rect is then compared
  against a `dest` that is `playOrigin` too far right/down, so outlets stop killing (or
  kill at the wrong place) once one of two players is dead. Reproduce the original
  behaviour exactly rather than "fixing" it if you want bit-identical play.
* The outlet blits with plain **`CopyBits` … `srcCopy`** (no mask), not `CopyMask` — the
  outlet sprites are opaque rectangles.
* The `SetPort((GrafPtr)workSrcMap)` before `PaintRect` is **commented out** in the
  shipped source (`GliderPRO/Sources/Dynamics.c:577`), so the dark-room `PaintRect` hits
  whatever port is current at that moment. In practice the render loop leaves `workSrcMap`
  current, so it works; do not rely on it as documented intent.
* The outlet draws itself; it is not in `RenderDynamics`.
* `hVel` tracks the room's light count. `UpdateOutletsLighting(room, nLights)`
  (`GliderPRO/Sources/Trip.c:235-244`) walks all dinahs and refreshes `hVel` for outlets
  in that room. In a dark room (`hVel == 0`) a quiescent outlet is painted **black**
  (A4's `PaintRect`), but a *zapping* outlet is still drawn — the arc is visible in the
  dark.
* `TriggerOutlet` (`GliderPRO/Sources/Trip.c:171-184`) starts an arc immediately if
  `position == 0` and `active`.

### 12.3 `kShredder` (0x61) — the paper shredder

**Art.** `srcRects[kShredder]` = 73×22 (`GliderPRO/Sources/StructuresInit2.c:430`), drawn
with `DrawSimpleAppliance` from the appliance sheet, only when `isLit`
(`GliderPRO/Sources/ObjectDrawAll.c:634-640`). No dynamic, no animation.

**Fields.** Every payload field except `initial`/`state` is **0** in all 50 shipped
shredders (verified by census). `topLeft.h` may be either parity.

**Hot spot** (`GliderPRO/Sources/ObjectRects.c:940-950`):

```
1. bounds        = srcRects[kShredder];              // 73 x 22
2. bounds.bottom = bounds.top + kShredderActiveHigh; // -> 73 x 40
3. bounds.right += 48;                               // -> 121 x 40
4. ZeroRectCorner(&bounds);
5. QOffsetRect(&bounds, topLeft.h, topLeft.v);
6. QOffsetRect(&bounds, -24, -36);                   // shift LEFT 24, UP 36
7. AddActiveRect(&bounds, kShredIt, who, data.g.state, /*scrutinize*/ true);
```

The resulting kill zone is **121×40 px positioned 24 px left of and 36 px above the
sprite's top-left**, i.e. it covers the air above and around the shredder mouth, not the
shredder body. With scrutinize on, the glider's effective 38×10 box must be **fully
inside** it (`kShredIt` uses `GliderInRect`).

**`kShredIt` handler** (`GliderPRO/Sources/Interactions.c:1324-1341`):

```
1. if (thisGlider->mode == kGliderShredding)  break;      // already dying
2. if (!GliderInRect(thisGlider, &who->bounds))  break;   // must be FULLY inside
3. if (foilTotal > 0) {
       PlayPrioritySound(kFoilHitSound, kFoilHitPriority);
       foilTotal--;
       if (foilTotal <= 0) StartGliderFoilLosing(thisGlider);
   } else {
       FlagGliderShredding(thisGlider, &who->bounds);
   }
```

Unlike `CheckDynamicCollision`, the foil drain here is **not** `evenFrame`-gated, so a
foil-clad glider inside a shredder loses one charge **per frame** — 8 frames ≈ 0.27 s to
strip a full foil, then the next frame shreds.

**`FlagGliderShredding`** (`GliderPRO/Sources/Modes.c:366-402`):

```
 1. PlayPrioritySound(kCaughtFireSound, kCaughtFirePriority);
 2. dest.left   = bounds->left + 36;
    dest.right  = dest.left + kGliderWide (48);
    dest.bottom = dest.top + kGliderHigh (20);
 3. widen `whole` and `wholeShadow` to cover both the old and new positions;
 4. destShadow = 48 x kShadowHigh (9);
 5. mode  = kGliderShredding (20);
 6. src = mask = gliderSrc[thisGlider->facing ? 0 : 2];
 7. hVel = vVel = hDesiredVel = vDesiredVel = 0;
 8. frame = bounds->bottom - 3;         // TARGET Y for the descent
 9. tipped = false;
```

The glider is **teleported** to `bounds->left + 36`, i.e. 36 px right of the hot spot's
left edge = 12 px right of the sprite's left edge (because the hot spot was shifted 24
left). `frame` becomes an absolute target `y`, not a frame index.

**`MoveGliderShredding`** (`GliderPRO/Sources/Player.c:1260-1319`), with
`kDropShredSlow` = 1 and `kDropShredFast` = 4:

```
IF frame > 0:
  1. src = mask = gliderSrc[facing ? 0 : 2];
  2. vNotClipped = frame - dest.top;
  3. IF vNotClipped < kGliderHigh (20):        // being consumed
        whole.top = dest.top;  dest.top += 1;  dest.bottom += 1;
        whole.bottom = dest.bottom;  shadowVisible = false;
        PlayPrioritySound(kShredSound, kShredPriority);     // EVERY frame
     ELSE:                                      // still falling toward the mouth
        drop by kDropShredFast (4), no sound
  4. recompute vNotClipped = frame - dest.top;
  5. IF vNotClipped < 20:
        IF vNotClipped <= 0:  AddAShreddedGlider(&thisGlider->dest);
                              frame = kShredderCountdown (-68);
        ELSE:                 clip dest.bottom, src.bottom, mask.bottom
                              to dest.top + vNotClipped;
ELSE:
  6. frame++;  if (frame >= 0)  OffAMortal(thisGlider);
```

So the sequence is: fall at 4 px/frame until within 20 px of `bounds->bottom - 3`, then
descend at 1 px/frame while being progressively clipped (and screaming `kShredSound`
every frame), then spawn the shredded-paper effect and wait **68 frames ≈ 2.26 s**
before the life is actually lost.

**The shredded-paper effect.** `AddAShreddedGlider(Rect *bounds)`
(`GliderPRO/Sources/DynamicMaps.c:730-742`):

```
1. if (numShredded > kMaxShredded)  return;      // NOTE: '>' not '>=' -- see §17
2. shreds[numShredded].bounds.left   = bounds->left + 4;
3. shreds[numShredded].bounds.right  = shreds[numShredded].bounds.left + 40;
4. shreds[numShredded].bounds.top    = bounds->top + 14;
5. shreds[numShredded].bounds.bottom = shreds[numShredded].bounds.top;   // zero height
6. shreds[numShredded].frame = 0;
7. numShredded++;
```

`RenderShreds` (`GliderPRO/Sources/Render.c:559-612`):

```
for each shred i:
  IF frame == 0 (revealing):
     bounds.bottom += 1;  high = bounds.bottom - bounds.top;
     if (high >= 35)  frame = 1;
     src = shredSrcRect;  src.top = src.bottom - high;      // reveal from the bottom up
     CopyMask(...);  AddRectToBackRects(&dest);
     dest.top--;  AddRectToWorkRects(&dest);
     PlayPrioritySound(kShredSound, kShredPriority);        // EVERY frame of the reveal
  ELSE IF frame < 20 (drifting down):
     bounds.top += 4;  bounds.bottom += 4;  frame++;
     if (frame < 20)  CopyMask(full shredSrcRect);
     else { AddSparkle(&shreds[i].bounds);
            PlayPrioritySound(kFadeOutSound, kFadeOutPriority); }
     AddRectToBackRects(&dest);  dest.top -= 4;  AddRectToWorkRects(&dest);
```

`shredSrcRect` is 40×35 (PICT 4010 / mask 5010). The reveal takes 35 frames, the drift
19 more, then a sparkle and `kFadeOutSound`. `RemoveShreds`
(`GliderPRO/Sources/DynamicMaps.c:748-778`) removes the slot with the largest `frame`,
compacting from the end; it is called from `OffAMortal`
(`GliderPRO/Sources/Player.c:1486-…`) when `numShredded > 0`.

**Shredders and switches.** `SetObjectState` handles `kShredder` in the appliance block
(`GliderPRO/Sources/Objects.c:590-628`) and, uniquely among appliances, also updates the
hot spot: `if (...what == kShredder) hotSpots[masterObjects[local].hotNum].isOn =
newState;` (`:625-626`). But `HandleSwitches`'s per-type dispatch has
`case kShredder: break;` with **no action** (`GliderPRO/Sources/Interactions.c:1086-1087`)
— there is no `ToggleShredder`. The state change alone (mirrored into `hotSpots[].isOn`)
is what turns the shredder off.

### 12.4 `kMicrowave` (0x6A) — the resource drainer

**Art.** `srcRects[kMicrowave]` = 92×59. `DrawMicrowave`
(`GliderPRO/Sources/ObjectDraw2.c:843-910`) creates a **temporary GWorld + mask on the
fly** (`CreateOffScreenGWorld` + `LoadGraphic(kMicrowavePictID / kMicrowaveMaskID)`)
when `isLit`, `CopyMask`s the body, then `DisposeGWorld`s them; then it stamps three
16-px-wide `microOn`/`microOff` columns starting at `(theRect->left + 14,
theRect->top + 13)`.

**Fields.** `data.g.byte0` = **kill mask** (observed 0..7, 7 distinct values in shipped
data); `data.g.delay` = 0 in all shipped microwaves; `topLeft.h` forced **even**. Editor
default `byte0 = 7` (kill everything, `GliderPRO/Sources/ObjectAdd.c:589-644`).
`DoMicrowaveObjectInfo` (`GliderPRO/Sources/ObjectInfo.c:~1750-1865`, dialog ID
`kMicrowaveInfoDialogID` = 1035) exposes three checkboxes.

**Kill mask bits:**

| Bit | Value | Resource drained |
|---:|---:|---|
| 0 | `0x0001` = 1 | rubber bands (`bandsTotal`) |
| 1 | `0x0002` = 2 | battery / helium (`batteryTotal`) |
| 2 | `0x0004` = 4 | foil (`foilTotal`) |

**Hot spots** (`GliderPRO/Sources/ObjectRects.c:969-979`): the 92×59 body becomes
`kDissolveIt` (scrutinized), then `bounds.bottom = bounds.top; bounds.top = 0;` makes a
**full-height column from the room ceiling (y = 0) down to the microwave's top**, 92 px
wide, registered as `kMicrowaveIt` (scrutinized). So you get zapped by flying *above*
the microwave, and killed by touching it.

**`kMicrowaveIt` handler** (`GliderPRO/Sources/Interactions.c:1581-1584`):

```
if (GliderInRect(thisGlider, &who->bounds))
    HandleMicrowaveAction(who, thisGlider);
```

**`HandleMicrowaveAction`** (`GliderPRO/Sources/Interactions.c:1159-1194`):

```
 1. if (who->stillOver)  return;                        // once per entry
 2. whoLinked = who->who;
 3. if (!masterObjects[whoLinked].theObject.data.g.state)  return;   // must be ON
 4. kills = (short)masterObjects[whoLinked].theObject.data.g.byte0;
 5. didSomething = false;
 6. if ((kills & 0x0001) && (bandsTotal > 0)) {
        bandsTotal = 0;  QuickBandsRefresh(false);  didSomething = true; }
 7. if ((kills & 0x0002) && (batteryTotal != 0)) {
        batteryTotal = 0;  QuickBatteryRefresh(false);  didSomething = true; }
 8. if ((kills & 0x0004) && (foilTotal > 0)) {
        foilTotal = 0;  StartGliderFoilLosing(thisGlider);  didSomething = true; }
 9. if (didSomething)  PlayPrioritySound(kMicrowavedSound, kMicrowavedPriority);
```

Notes:

* Resources are **zeroed**, not decremented. `batteryTotal != 0` (not `> 0`) because a
  negative `batteryTotal` encodes **helium** while positive encodes battery — the
  microwave destroys either.
* `who->stillOver` is set by the caller chain (`HandleMicrowaveAction` does **not** set
  it — it returns early on it, and it is `HandleHotSpotCollision`'s general contract via
  `CheckForHotSpots` clearing it on exit… but in fact **nothing sets `stillOver` for
  `kMicrowaveIt`**). Re-read: `HandleMicrowaveAction` checks `who->stillOver` at entry
  but never assigns it, and `case kMicrowaveIt:` does not assign it either. So the
  guard at step 1 is never true and the microwave fires **every frame** the glider is
  fully inside the column. Since it zeroes rather than decrements, the observable
  difference is only the repeated `kMicrowavedSound`. See §17.
* The microwave is **not** protected against by foil; foil is one of the things it
  destroys.
* Supplies for reference (`GliderPRO/Sources/Interactions.c:16-19`):
  `kBatterySupply` = 50, `kHeliumSupply` = 150, `kBandsSupply` = 8, `kFoilSupply` = 8.

### 12.5 `kGuitar` (0x64) — harmless but hot

**Art.** `srcRects[kGuitar]` = 64×172. Hot spot (`GliderPRO/Sources/ObjectRects.c:952-957`):
`QSetRect(0, 0, 8, 96)` offset by `(topLeft.h + 34, topLeft.v + 32)`, action
`kStrumIt`, `isOn` = literal `true`, scrutinize `false`. So the strum zone is an 8×96
vertical strip across the strings.

**`kStrumIt`** (`GliderPRO/Sources/Interactions.c:1343-1349`):

```
if (!who->stillOver) {
    PlayPrioritySound(kChordSound, kChordPriority);
    who->stillOver = true;
}
```

Purely cosmetic. `SetObjectState` explicitly refuses to change a guitar's state
(`case kGuitar: changed = false;`, `GliderPRO/Sources/Objects.c:580-582`), and
`HandleSwitches` and `FireTrigger` both just play `kChordSound` for it
(`GliderPRO/Sources/Interactions.c` guitar case, `GliderPRO/Sources/Triggers.c:139-141`).
Also note `isOn` is hardcoded `true`, so a guitar can never be switched off.

### 12.6 The plain lethal appliance bodies

`kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kVCR`, `kStereo`, `kCinderBlock`,
`kFlowerBox`, `kCDs` all get exactly one hot spot: their sprite rect, `kDissolveIt`,
`isOn = true` (**hardcoded**, so unswitchable as an obstacle), scrutinize `true`
(`GliderPRO/Sources/ObjectRects.c:981-996`). `kCustomPict` gets nothing at all
(`:998-999`) — custom pictures are pure decoration.

**`kDissolveIt` handler** (`GliderPRO/Sources/Interactions.c:1218-1244`):

```
1. if (thisGlider->mode == kGliderFadingOut)  break;          // already dying
2. IF (foilTotal > 0) OR (thisGlider->mode == kGliderLosingFoil):
     3. IF GliderHitTop(thisGlider, &who->bounds):            // landed on top
            StartGliderFadingOut(thisGlider);
            PlayPrioritySound(kFadeOutSound, kFadeOutPriority);   // DEATH ANYWAY
        ELSE:
            // GliderHitTop already played kFoilHitSound, decremented foilTotal
            // (Interactions.c:81-84), reversed hVel and separated the glider
            IF foilTotal > 0:                                 // SECOND decrement!
                foilTotal--;
                if (foilTotal <= 0)  StartGliderFoilLosing(thisGlider);
   ELSE:
     4. StartGliderFadingOut(thisGlider);
        PlayPrioritySound(kFadeOutSound, kFadeOutPriority);        // DEATH
```

The crucial asymmetry: **foil protects you from side impacts but not from landing on
top of a lethal object.**

Also note the state-machine gate: `kGliderFadingOut` is skipped, but unlike
`CheckDynamicCollision` there is **no whitelist of safe modes** — a glider that is
transporting, ducting, mailing or shredding *can* be killed by a `kDissolveIt` hot spot
if its `dest` still overlaps one.

### 12.7 Appliance state animations (non-lethal, listed for completeness)

These consume dinah slots and therefore compete with hazards for the 18-slot budget.

| Handler | Cite | Behaviour |
|---|---|---|
| `HandleMacPlus` | `GliderPRO/Sources/Dynamics.c:388-424` | `timer--`; active: `timer == 30` → `kMacOnSound`; `timer == 1` → `kMacBeepSound` + `plusScreen2`; inactive: `timer == 1` → `kMacOffSound` + `plusScreen1`. `ToggleMacPlus` sets `timer = 40` (on) / `10` (off). |
| `HandleTV` | `:428-478` | active `timer == 1` → `kTVOnSound` (+`tvScreen2` unless a QuickTime movie owns the screen); inactive `timer == 1` → `kTVOffSound` + `tvScreen1`. |
| `HandleCoffee` | `:482-524` | active `timer == 0` → `timer = 200 + RandomInt(200)`; `== 1` → `kMacOnSound` + `coffeeLight2`; `== 100` → `kCoffeeSound` and `timer = 200 + RandomInt(200)`; inactive `== 1` → `kMacOffSound` + `coffeeLight1`. Initial `timer` = 200 if on else 0. |
| `HandleVCR` | `:603-667` | active reset to 115; `== 5` → `kMacOnSound`; `== 1` → `kVCRSound` + `vcrTime2`; `== 100` → `timer = 115`, `frame = 1 - frame`; `== 101` blinks `vcrTime2`/`vcrTime1`; inactive `== 1` → `kMacOffSound` + `vcrTime1`. Initial `timer` = 115 if on else 0. |
| `HandleStereo` | `:671-711` | `timer == 0` → `ToggleMusicWhilePlaying()`; `== 1` → `kMacOnSound`/`kMacOffSound` + `stereoLight2`/`stereoLight1`. `ToggleStereos` only acts `if (timer == 0)`. |
| `HandleMicrowave` | `:715-775` | `timer == 1` → `kMacOnSound`/`kMacOffSound` and three 16-px `microOn`/`microOff` columns blitted into `backSrcMap`. |

### 12.8 `kSparkle` (0x2D) — the ambient twinkle

Not a hazard, but it occupies a dinah slot. `HandleSparkleObject`
(`GliderPRO/Sources/Dynamics.c:293-317`):

```
IF active:
    IF frame <= 0:                        // idle
        timer--;
        IF timer <= 0:
            timer = RandomInt(240) + 60;  // 60..299 frames = 2..10 s
            frame = kNumSparkleModes (5);
            AddSparkle(&tempRect);
            PlayPrioritySound(kMysticSound, kMysticPriority);
    ELSE:                                 // sparkling
        frame--;
ELSE:
    /* nothing at all -- the else body is literally empty (:314-316) */
```

Note the nesting: `frame--` is the `else` of the **inner** `frame <= 0` test, not of
`active`, so an inactive sparkle is frozen mid-animation rather than winding down
(`GliderPRO/Sources/Dynamics.c:297-316`).

Spawn `timer = RandomInt(60) + 15` (`GliderPRO/Sources/Dynamics3.c:208`), i.e. 15..74
frames for the first twinkle. `frame` is used as a lockout so a sparkle cannot restart
until the previous 5-mode animation has finished.

---

## 13. Hazard reference: flames and blowers

### 13.1 The five open flames

`kTaper` (0x08), `kCandle` (0x09), `kStubby` (0x0A), `kTiki` (0x0B), `kBBQ` (0x0C).

Each produces **three** hot spots from one object, in this order
(`GliderPRO/Sources/ObjectRects.c:399-520`):

```
1. QSetRect(&bounds, 0, -theObject.data.a.distance, kFloorColumnWide (4), 0);
     // BBQ uses bottom = 8 instead of 0 (ObjectRects.c:500)
2. QOffsetRect(&bounds, HalfRectWide(&srcRects[what]) - kFloorColumnWide/2, 0);
     // kStubby subtracts an extra 1  (ObjectRects.c:452)
3. QOffsetRect(&bounds, topLeft.h, topLeft.v);
     // kCandle uses topLeft.h - 2   (ObjectRects.c:430)
4. IF (bounds.bottom - bounds.top) > kDeadlyFlameHeight (24):
       bounds.bottom -= 24;
       AddActiveRect(&bounds, kLiftIt,  who, /*isOn*/ TRUE, /*scrutinize*/ false);
       bounds.bottom += 24;
       bounds.top = bounds.bottom - 24 + 2;
       AddActiveRect(&bounds, kBurnIt,  who, TRUE, false);
   ELSE
       AddActiveRect(&bounds, kBurnIt,  who, TRUE, false);
5. QSetRect(&bounds, 0, 0, <bodyW>, <bodyH>);
   QOffsetRect(&bounds, topLeft.h + <dx>, topLeft.v + <dy>);
   AddActiveRect(&bounds, kDissolveIt, who, TRUE, TRUE);
```

Per-type numbers:

| Flame | `srcRects[]` | Half-width term | Column x-offset | Body rect | Body offset | Cites |
|---|---|---:|---|---|---|---|
| `kTaper` | 20×59 | `10 - 2 = 8` | `topLeft.h + 8` | 7×48 | `(h+6, v+11)` | `:400-421` |
| `kCandle` | 32×30 | `16 - 2 = 14` | `topLeft.h - 2 + 14 = h+12` | 8×20 | `(h+9, v+11)` | `:425-446` |
| `kStubby` | 20×36 | `10 - 2 - 1 = 7` | `topLeft.h + 7` | 15×26 | `(h+1, v+11)` | `:450-471` |
| `kTiki` | 27×28 | `13 - 2 = 11` | `topLeft.h + 11` | 15×14 | `(h+6, v+6)` | `:475-496` |
| `kBBQ` | 64×33 | `32 - 2 = 30` | `topLeft.h + 30` | 52×17 | `(h+6, v+8)` | `:500-519` |

(`HalfRectWide` returns `(right - left) / 2` on the zeroed-corner sprite rect.)

**The `isOn` argument for all three flame hot spots is the literal `TRUE`.** A flame's
`data.a.state` is never consulted, so **a flame column can never be switched off**. The
census confirms shipped flames always have `initial = state = 1` anyway. Additionally
`SetObjectsToDefaults` (`GliderPRO/Sources/Play.c:624-637`) resets
`data.a.state = data.a.initial` for `kFloorVent`, `kCeilingVent`, `kFloorBlower`,
`kCeilingBlower`, `kLeftFan`, `kRightFan`, `kSewerGrate`, `kInvisBlower`, `kGrecoVent`,
`kSewerBlower`, `kLiftArea` — and **omits kTaper/kCandle/kStubby/kTiki/kBBQ entirely**,
which is consistent: their state is irrelevant.

**The column geometry.** `bounds` is 4 px wide (`kFloorColumnWide`) and
`data.a.distance` tall, extending **upward** from `topLeft.v` (because `top` is
`-distance` and `bottom` is 0 before the offset). If the column is taller than 24 px, the
top part becomes a `kLiftIt` updraft and only the **bottom 22 px** (`bottom - 24 + 2` to
`bottom`) burn. If the column is 24 px or shorter, the whole thing burns.

Observed shipped `distance` for flames goes up to **240** (`kTaper`/`kCandle`) and
**276** (some), so tall updraft columns are common: a 240-px candle gives a 216-px
`kLiftIt` column above a 22-px `kBurnIt` zone.

**`kBurnIt` handler** (`GliderPRO/Sources/Interactions.c:1356-1374`):

```
1. if (thisGlider->mode is kGliderBurning or kGliderFadingOut)  break;
2. IF (foilTotal > 0) OR (mode == kGliderLosingFoil):
      thisGlider->vDesiredVel = kFloorVentLift (-6);        // lifted, not burned
      PlayPrioritySound(kSizzleSound, kSizzlePriority);
      foilTotal--;
      if (foilTotal <= 0)  StartGliderFoilLosing(thisGlider);
   ELSE:
      FlagGliderBurning(thisGlider);
```

Note the foil drain is **not** `evenFrame`-gated here either: one charge per frame.

**`FlagGliderBurning`** (`GliderPRO/Sources/Modes.c:406-434`), `kFramesToBurn` = 60
(`:408`):

```
1. PlayPrioritySound(kCaughtFireSound, kCaughtFirePriority);
2. dest.right = dest.left + kGliderWide (48);
   dest.top   = dest.bottom - kGliderBurningHigh (26);     // grows UPWARD to 26 px
3. destShadow = 48 x kShadowHigh (9);
4. mode = kGliderBurning (9);
5. src = mask = gliderSrc[facing ? 21 : 25];
6. hVel = vVel = hDesiredVel = vDesiredVel = 0;
7. frame = 0;
8. wasMode = kFramesToBurn (60);        // countdown to death
9. tipped = false;
```

**`MoveGliderBurning`** (`GliderPRO/Sources/Player.c:203-227`):

```
1. frame++;  if (frame > 3)  frame = 0;                       // 4-frame flame cycle, 30 fps
2. src = mask = gliderSrc[facing ? 21 + frame : 25 + frame];
3. wasMode--;
4. if (wasMode <= 0) { StartGliderFadingOut(thisGlider);
                       PlayPrioritySound(kFadeOutSound, kFadeOutPriority); }
5. MoveGlider(thisGlider);                                    // normal physics continues
```

So a burning glider still flies (with player control) for **60 frames ≈ 2 s** and then
dies. Its collision box is `top += 6`-adjusted in `SectGlider`
(`GliderPRO/Sources/Interactions.c:103`) so the flames on top do not count as body.

**`kLiftIt` handler** (`GliderPRO/Sources/Interactions.c:1202-1204`): simply
`thisGlider->vDesiredVel = kFloorVentLift` = **−6**.

**Flame rendering.** Flames are **not** dinahs. They are `flameType` records with a
per-flame pre-composited backing store:

```c
typedef struct { Rect dest, src; short mode; short who; } flameType;
```
(`GliderPRO/Headers/GliderStructs.h:251-256`)

`AddCandleFlame(where, who, h, v)` (`GliderPRO/Sources/DynamicMaps.c:316-344`):

```
1. if ((numFlames >= kMaxCandles (20)) || (h < 16) || (v < 15))  return;
2. QSetRect(&src, 0, 0, 16, 15);  QOffsetRect(&src, h - 8, v - 15);
3. (4-bit-depth even-x nudge, DynamicMaps.c:326-331)
4. QSetRect(&bounds, 0, 0, 16, 15 * kNumCandleFlames (5));    // 16 x 75
5. savedNum = BackUpToSavedMap(&bounds, where, who);
6. if (savedNum != -1) {
       BackUpFlames(&src, savedNum);
       flames[numFlames].dest = src;
       flames[numFlames].mode = RandomInt(kNumCandleFlames);   // random start phase
       flames[numFlames].src  = 16 x 15 at (0, mode * 15);
       flames[numFlames].who  = savedNum;
       numFlames++;
   }
```

`BackUpFlames(Rect *src, short index)` (`GliderPRO/Sources/DynamicMaps.c:263-285`)
pre-composites **all five animation frames over the room background** into a 16×75
`savedMap`:

```
QSetRect(&dest, 0, 0, 16, 15);
for (i = 0; i < kNumCandleFlames; i++) {
    CopyBits(backSrcMap -> savedMaps[index].map, src, dest, srcCopy);   // background
    CopyMask(blowerSrcMap, blowerMaskMap -> savedMaps[index].map,
             &flame[i], &flame[i], &dest);                              // flame over it
    QOffsetRect(&dest, 0, 15);
}
```

`RenderFlames` (`GliderPRO/Sources/Render.c:193-256`) then only has to do a plain
`CopyBits` of the right 16×15 strip:

```
if ((numFlames == 0) && (numTikiFlames == 0) && (numCoals == 0))  return;
for each candle flame i:
    mode++;  src.top += 15;  src.bottom += 15;
    if (mode >= kNumCandleFlames) { mode = 0; src.top = 0; src.bottom = 15; }
    CopyBits(savedMaps[flames[i].who].map -> workSrcMap, &src, &dest, srcCopy, nil);
    AddRectToWorkRects(&flames[i].dest);
// tikis: stride 10, wrap at kNumTikiFlames (5)
// coals:  stride  9, wrap at kNumBBQCoals  (4)
```

`RenderFlames` is called **only on `evenFrame`** (`GliderPRO/Sources/Render.c:649-650`),
so flames animate at **15 fps**: 5 frames in 10 game frames = 3 cycles/second.

Flame attach points, from `DrawARoomsObjects`
(`GliderPRO/Sources/ObjectDrawAll.c:83-100` and the per-type cases):

| Flame | Call | Attach point |
|---|---|---|
| `kTaper` | `AddCandleFlame(room, i, itsRect.left + 10, itsRect.top + 7)` | 16×15 flame |
| `kCandle` | `AddCandleFlame(room, i, left + 14, top + 7)` | 16×15 |
| `kStubby` | `AddCandleFlame(room, i, left + 9, top + 7)` | 16×15 |
| `kTiki` | `AddTikiFlame(room, i, left + 10, top - 9)` | 8×10 |
| `kBBQ` | `AddBBQCoals(room, i, left + 16, top + 9)` | 32×9 |

For **non-central** neighbours the flame is only added if a 16×15 rect at
`(left + 10 - 8, top + 7 - 15)` does **not** intersect `localRoomsDest[kCentralRoom]`
grown by `kFloorSupportTall` (44) top and bottom
(`GliderPRO/Sources/ObjectDrawAll.c:88-100`) — a de-duplication guard so a flame near a
room boundary is not drawn twice.

`AddTikiFlame` (`GliderPRO/Sources/DynamicMaps.c:400-429`): guard
`numTikiFlames >= kMaxTikis (8) || h < 8 || v < 10`; src 8×10 at `(h, v)` with **no**
`-8/-10` re-centring (unlike candles); bounds 8×50; `mode = RandomInt(5)`.

`AddBBQCoals` (`GliderPRO/Sources/DynamicMaps.c:486-515`): guard
`numCoals >= kMaxCoals (8) || h < 32 || v < 9`; src 32×9 at `(h, v)`; bounds 32×36;
`mode = RandomInt(4)`.

`ZeroFlamesAndTheLike` (`GliderPRO/Sources/DynamicMaps.c:787-797`) zeroes `numFlames`,
`numTikiFlames`, `numCoals`, `numPendulums`, `numGrease`, `numStars`, `numShredded`,
`numChimes`.

### 13.2 `kLeftFan` (0x06) / `kRightFan` (0x07)

Two hot spots each.

`kLeftFan` (`GliderPRO/Sources/ObjectRects.c:369-382`):

```
1. QSetRect(&bounds, 0, 0, 13, 43);
   QOffsetRect(&bounds, topLeft.h + 16, topLeft.v + 12);
   AddActiveRect(&bounds, kDissolveIt, who, TRUE, TRUE);              // lethal blades
2. QSetRect(&bounds, 0, 0, theObject.data.a.distance, kFanColumnThick (16));
   QOffsetRect(&bounds, -theObject.data.a.distance, kFanColumnDown (20));
   QOffsetRect(&bounds, topLeft.h, topLeft.v);
   AddActiveRect(&bounds, kPushItLeft, who, theObject.data.a.state, false);
```

`kRightFan` (`GliderPRO/Sources/ObjectRects.c:384-397`): body 13×43 at
`(topLeft.h + 6, topLeft.v + 12)` → `kDissolveIt`; push column offset by
`(RectWide(&srcRects[kRightFan]) = 40, 20)` → `kPushItRight`, `isOn = data.a.state`.

So the wind column is `distance` px long and 16 px tall, positioned 20 px below the
sprite top, blowing away from the fan. The **blades are lethal** and **cannot be
switched off** (`isOn` is the literal `TRUE`); only the wind is switchable.

Observed shipped values: `kLeftFan` always `vector = 8`, `distance` 3..300;
`kRightFan` `vector` ∈ {2, 8}, `distance` 0..368. (`vector` is unused by fans; only
`kInvisBlower` reads it.)

**`kPushItLeft` / `kPushItRight`** (`GliderPRO/Sources/Interactions.c:1210-1216`):

```
case kPushItLeft:  thisGlider->hDesiredVel += -kFanStrength;   // -12
case kPushItRight: thisGlider->hDesiredVel +=  kFanStrength;   // +12
```

Note `+=`, not `=` — **overlapping fan columns stack**. Compare `kLiftIt`/`kDropIt`,
which assign.

### 13.3 `kInvisBlower` (0x0D)

`srcRects[kInvisBlower]` = 24×24. `CreateActiveRects` switches on
`theObject.data.a.vector & 0x0F` (`GliderPRO/Sources/ObjectRects.c:522…`). The `vector`
bit meaning is documented in the struct comment
(`GliderPRO/Headers/GliderStructs.h:16-17`):

```
Boolean state;   //  F. lf. dn. rt. up
Byte    vector;  // | x | x | x | x | 8 | 4 | 2 | 1 |
```

| Bit | Value | Direction | Resulting action |
|---:|---:|---|---|
| 0 | 1 | up | `kLiftIt` |
| 1 | 2 | right | `kPushItRight` |
| 2 | 4 | down | `kDropIt` |
| 3 | 8 | left | `kPushItLeft` |

The `case 1` (up) geometry is
`QSetRect(0, -distance - 24, kFloorColumnWide (4), 0)` then
`QOffsetRect(12 - kFloorColumnWide/2, 24)` then by `topLeft` — i.e. a 4-px column
`distance + 24` tall starting 24 px below the object origin, centred in the 24-px-wide
invisible sprite.

`kInvisBlower` is by far the most common blower in shipped content: **2336 instances**
across 22 houses. Observed `vector` values are 1, 2, 4, 8, **17** and **20** — the
latter two have high bits set (17 = 0x11, 20 = 0x14), which the `& 0x0F` masks down to
1 and 4. Observed `distance` 0..488. **Observed `initial`/`state` bytes include the
non-boolean value 23** in shipped data; every consumer treats them as truthy, so this is
benign, but a Go port that unmarshals them as `bool` must accept any non-zero byte.

### 13.4 The non-lethal blowers

`kFloorVent` (0x01, `:311-319`), `kFloorBlower` (0x03, `:333-343`), `kSewerGrate` (0x05,
`:357-367`), `kGrecoVent` (0x0E, `:566-576`) and `kSewerBlower` (0x0F, `:578-588`) each
produce a single `kLiftIt` column of width `kFloorColumnWide` = 4 and height
`data.a.distance`, gated on `data.a.state`. `kCeilingVent` (0x02) and `kCeilingBlower`
(0x04) produce a `kDropIt` column of width `kCeilingColumnWide` = 24 and height
`data.a.distance` (`GliderPRO/Sources/ObjectRects.c:321-331` and `:345-355`).

`kLiftArea` (0x10) is **not** a plain lift column: like `kInvisBlower` it switches on
`data.a.vector & 0x0F` and emits `kLiftIt` (1), `kPushItRight` (2), `kDropIt` (4) or
`kPushItLeft` (8) over a rect of `distance` × `tall * 2`
(`GliderPRO/Sources/ObjectRects.c:590-617`).

None of these has a lethal body.

---

## 14. Hazard reference: the cobweb

`kCobweb` (0x79) is the strangest hazard in the game: it is an *enemy*-class object with
no dynamic, no animation, no state, and no foil protection.

**Art.** `srcRects[kCobweb]` = 54×45; drawn with `DrawPictWithMaskObject` and only when
`isLit` (`GliderPRO/Sources/ObjectDrawAll.c:894-900`).

**Hot spot** (`GliderPRO/Sources/ObjectRects.c:1025-1033`):

```
bounds = srcRects[kCobweb];   // 54 x 45
ZeroRectCorner(&bounds);
QOffsetRect(&bounds, data.h.topLeft.h, data.h.topLeft.v);
InsetRect(&bounds, -24, -10);                  // GROWS to 102 x 65
AddActiveRect(&bounds, kWebIt, who, TRUE, TRUE);
```

Negative `InsetRect` grows the rect: 54 + 48 = 102 wide, 45 + 20 = 65 tall. `isOn` is
the literal `TRUE`; scrutinize is `TRUE`.

**Fields.** Observed shipped values: `length = 0`, `delay = 0`, `initial = state = 1`
always; `topLeft.h` may be either parity. 86 cobwebs across the 22 shipped houses.

**`kWebIt` handler** (`GliderPRO/Sources/Interactions.c:1602-1613`):

```
IF GliderInRect(thisGlider, &who->bounds) AND mode != kGliderBurning:
    WebGlider(thisGlider, &who->bounds);
ELSE IF mode == kGliderBurning AND GliderInRect(...):
    thisGlider->wasMode = 0;
    StartGliderFadingOut(thisGlider);
    PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
```

A **burning** glider that flies into a cobweb dies immediately (the web catches fire).

**`WebGlider`** (`GliderPRO/Sources/Interactions.c:1736-1776`), `kKillWebbedGlider` = 150
(`:1738`):

```
 1. IF mode == kGliderBurning AND GliderInRect(...):
        wasMode = 0;  StartGliderFadingOut;  kFadeOutSound;  return;
 2. hDist = ((webBounds->right  - dest.right)  + (webBounds->left - dest.left))  >> 3;
 3. vDist = ((webBounds->bottom - dest.bottom) + (webBounds->top  - dest.top))   >> 3;
 4. IF thisGlider->hDesiredVel != 0:                  // player is struggling
       IF evenFrame:
           thisGlider->hVel = hDist;
           thisGlider->vVel = vDist;
           PlayPrioritySound(kWebTwangSound, kWebTwangPriority);
    ELSE:
       thisGlider->hDesiredVel = 0;
       thisGlider->vDesiredVel = 0;
 5. thisGlider->wasMode++;
 6. IF thisGlider->wasMode >= kKillWebbedGlider (150):
       thisGlider->wasMode = 0;
       StartGliderFadingOut(thisGlider);
       PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
```

Reading of the physics: `hDist`/`vDist` are `(2 * centre offset) >> 3` = **one quarter of
the distance from the glider's centre to the web's centre** (arithmetic shift, so it
rounds toward negative infinity for negative values). While the player holds a direction
key (`hDesiredVel != 0`), the glider is yanked back toward the web centre at a quarter
of the offset, once every two frames, with a `kWebTwangSound`. Releasing the keys
freezes the glider (`hDesiredVel = vDesiredVel = 0`).

**The player can never escape a cobweb.** `wasMode` counts up unconditionally at step 5.
After **150 frames ≈ 5 s** the glider dies. Foil is never consulted — the cobweb is the
only hazard that ignores foil entirely.

`wasMode` is normally the *saved glider mode* field; the cobweb repurposes it as a
counter, the same way `FlagGliderBurning` does (`wasMode = 60`).

`kCobweb` is absent from `SetObjectsToDefaults` (`GliderPRO/Sources/Play.c:692-702`
lists only kBalloon, kCopterLf, kCopterRt, kDartLf, kDartRt, kBall, kDrip, kFish) **and**
from `SetObjectState`'s enemy block, where it gets `case kCobweb: changed = false;`
(`GliderPRO/Sources/Objects.c:673-675`). It is completely unswitchable and its `state`
byte is never reset — consistent with `isOn` being hardcoded `TRUE`.

---

## 15. The rubber band: the only way to destroy a hazard

### 15.1 Data

```c
typedef struct { Rect dest; short mode, count; short hVel, vVel; } bandType;
```
(`GliderPRO/Headers/GliderStructs.h:274-279`)

Globals (`GliderPRO/Sources/RubberBands.c:21-26`): `bandPtr bands;`,
`Rect bandsSrcRect;`, `Rect bandRects[3];`, `GWorldPtr bandsSrcMap, bandsMaskMap;`,
`short numBands, bandHitLast;`.

Constants: `kRubberBandVelocity` = 20 (`:12`), `kBandFallCount` = 4 (`:13`),
`kKillBandMode` = −1 (`:14`), `kMaxRubberBands` = 2
(`GliderPRO/Headers/GliderDefines.h:261`). Art: PICT 4007 / mask 5007, 16×18, three
6-px frames.

### 15.2 Firing

`AddBand(gliderPtr thisGlider, short h, short v, Boolean direction)`
(`GliderPRO/Sources/RubberBands.c:256-290`):

```
1. if (numBands >= kMaxRubberBands (2))  return (false);
2. mode = 0;  count = 0;  vVel = thisGlider->tipped ? -2 : 0;
3. dest.left = h - 8;  dest.right = h + 8;  dest.top = v - 3;  dest.bottom = v + 3;
4. IF direction (right):  QOffsetRect(&dest,  32, 0);  hVel =  kRubberBandVelocity (20);
   ELSE:                  QOffsetRect(&dest, -32, 0);  hVel = -20;
5. thisGlider->hVel -= (bands[numBands].hVel / 2);     // RECOIL: -10 or +10
6. numBands++;
7. PlayPrioritySound(kFireBandSound, kFireBandPriority);
8. return (true);
```

The band starts as a 16×6 rect 32 px in front of the glider. **Firing recoils the glider
by 10 px/frame in the opposite direction** (step 5) — a real gameplay mechanic often
overlooked.

### 15.3 Per-frame motion

`HandleBands` (`GliderPRO/Sources/RubberBands.c:208-252`):

```
1. if (numBands == 0)  return;
2. for (i = 0; i < numBands; i++) {
     mode++;  if (mode > 2)  mode = 0;                 // 3-frame spin, full rate
     count++;  if (count >= kBandFallCount (4)) { vVel++; count = 0; }   // quarter gravity
     AddRectToWorkRects(bands[i].dest + playOrigin);   // erase old
     QOffsetRect(&bands[i].dest, bands[i].hVel, bands[i].vVel);
     CheckBandCollision(i);
   }
3. count = 0;
   do {
       while (bands[count].mode == kKillBandMode) { bands[count].mode = 0; KillBand(count); }
       count++;
   } while (count < numBands);
```

Band gravity is `+1` to `vVel` every **4 frames** — the weakest gravity in the game
(toast +1/frame, ball/drip/fish +1/2 frames).

`KillBand(which)` (`GliderPRO/Sources/RubberBands.c:294-303`) swaps the last band into
the dead slot and decrements `numBands`. `KillAllBands`
(`:307-317`) sets all `kMaxRubberBands` modes to 0 and `numBands = 0`.

### 15.4 Band collision

`CheckBandCollision(short who)` (`GliderPRO/Sources/RubberBands.c:37-204`):

```
 1. nothingCollided = true;
 2. IF (leftThresh == kLeftWallLimit) AND (dest.left < kLeftWallLimit (12)):
        if (hVel < 0) hVel = -hVel;
        dest.left = 12;  dest.right = dest.left + 16;
        PlayPrioritySound(kBandReboundSound, kBandReboundPriority);
 3. IF (rightThresh == kRightWallLimit) AND (dest.right > kRightWallLimit (500)):
        mirrored
 4. FOR i in 0..nHotSpots-1 with hotSpots[i].isOn AND action in
        { kDissolveIt, kRewardIt, kSwitchIt, kTriggerIt, kBounceIt }:
      4a. if no plain rect overlap, continue;
      4b. nothingCollided = false;
      4c. if (bandHitLast == i) continue;      // already handled this spot last frame:
                                              // skip the effect but KEEP SCANNING
      4d. bandHitLast = i;
      4e. switch (action):
            kDissolveIt / kBounceIt:
                if the band was OUTSIDE the spot last frame:
                    reverse hVel and snap the band flush to the spot's edge
                else
                    mode = kKillBandMode;      // stuck inside: die
                PlayPrioritySound(kBandReboundSound, kBandReboundPriority);
                break;
            kRewardIt:
                only for kGreaseRt/kGreaseLf: SetObjectState(...); SpillGrease(...);
                hotSpots[i].isOn = false;
                break;
            kSwitchIt:  HandleSwitches(&hotSpots[i]);  break;
            kTriggerIt: ArmTrigger(&hotSpots[i]);      break;
 5. if (nothingCollided)  bandHitLast = -1;
 6. IF bands[who].hVel != 0 AND the band overlaps a live glider's dest:
        theGlider.hVel += (bands[who].hVel / 2);       // PUSH the player
        bands[who].hVel = 0;
        PlayPrioritySound(kHitWallSound, kHitWallPriority);
 7. IF (dest.left < kLeftWallLimit) OR (dest.right > kRightWallLimit) OR
       (dest.bottom > kFloorLimit (312)):
        mode = kKillBandMode;
```

Points to note:

* `bandHitLast` is a **single global**, not per-band, so with two bands in flight the
  "already handled" latch is shared and can suppress a legitimate second hit.
* The only `break` out of the hot-spot loop is inside the `kDissolveIt`/`kBounceIt` arm
  (`GliderPRO/Sources/RubberBands.c:116`). `kRewardIt`, `kSwitchIt` and `kTriggerIt` do
  **not** break, so one band can flip several switches / arm several triggers in a single
  frame if their rects overlap it — but only the *last* one scanned survives in
  `bandHitLast`.
* The overlap tests here are strict (`<` / `>`) just like `DidBandHitDynamic`, i.e.
  inclusive on the shared edge.
* A band can **flip switches and arm triggers remotely** (steps 4e `kSwitchIt` /
  `kTriggerIt`) and can **knock over grease jars** (`kRewardIt`).
* A band **pushes the glider** (step 6): `theGlider.hVel += hVel/2` = ±10. This works on
  your own band too — you can shoot yourself.
* A band **cannot hit a hazard hot spot at all** in the sense of destroying it; the
  hazard destruction path is entirely separate (§15.5).

### 15.5 `DidBandHitDynamic` — destroying an enemy

`DidBandHitDynamic(short who)` (`GliderPRO/Sources/Dynamics.c:80-106`):

```
for (i = 0; i < numBands; i++) {
    collided = !(band.bottom < dinah.top  ||  band.top   > dinah.bottom ||
                 band.right  < dinah.left ||  band.left  > dinah.right);
    if (collided) break;
}
return (collided);
```

The comparisons are **strict** (`<` / `>`, `GliderPRO/Sources/Dynamics.c:90-99`), so this
test is *inclusive* on the shared edge: a band whose `bottom` exactly equals the hazard's
`top` counts as a hit. That is one pixel more permissive than QuickDraw's `SectRect` (and
than Go's `image.Rectangle.Overlaps`), which are exclusive on bottom/right.

**`collided` is uninitialised** if `numBands == 0`. Every one of the three call sites
guards with `(numBands > 0) &&` (`GliderPRO/Sources/Dynamics2.c:62`, `:162`, `:267`), so
the bug is unreachable — but a Go port must initialise it to `false` and must **not**
remove the `numBands > 0` guard on the assumption it is redundant, since the guard is
what makes the C correct.

Note the comparison is between **`bands[i].dest`** (room-local? see below) and
**`dinahs[who].dest`** (room-local for hazards). `AddBand` builds `dest` from the
glider's `h`/`v` which are room-local, and `HandleBands` adds `playOrigin` only when
handing rects to the blitter (`AddRectToWorkRects`) and in `RenderBands`
(`GliderPRO/Sources/Render.c:534-556`). So both are room-local and the comparison is
correct.

**Only three hazards call `DidBandHitDynamic`:**

| Hazard | Result of a hit | Sound |
|---|---|---|
| `kBalloon` | `frame = 6`, `vVel = 8`; falls off the bottom and respawns | `kPopSound` (407) |
| `kCopterLf`/`kCopterRt` | `frame = 8`, `hVel = 0`, `vVel = 8` | `kPaperCrunchSound` (410) |
| `kDartLf`/`kDartRt` | `frame = 1` or `3`, `hVel = 0`, `vVel = 8` | `kPaperCrunchSound` (410) |

"Destroyed" means "knocked into a falling state that terminates in the normal respawn
path". A shot enemy still respawns after `count` frames — you cannot permanently remove
one. Ball, drip, fish, toast, outlet arc, flames, shredder, cobweb and microwave are
completely immune.

---

## 16. Switching and triggering hazards

### 16.1 `SetObjectState`

`SetObjectState(short room, short object, short action, short local)`
(`GliderPRO/Sources/Objects.c:366-699`). `action` is one of `kToggle` = 0,
`kForceOn` = 1, `kForceOff` = 2, `kOneShot` = 3
(`GliderPRO/Headers/GliderDefines.h:441-444`).

**`kOneShot` never actually reaches `SetObjectState`.** It is assigned only to
`kTrigger`/`kLgTrigger` objects' `data.e.type` (`GliderPRO/Sources/ObjectAdd.c:503-504`),
and triggers call `SetObjectState(..., kForceOn, ...)` explicitly
(`GliderPRO/Sources/Triggers.c:114-115`, `:184-185`); the only other caller,
`HandleSwitches`, passes a switch's `data.e.type`, which the editor restricts to
kToggle/kForceOn/kForceOff (`GliderPRO/Sources/ObjectInfo.c:1325-1338`). This matters
because every inner `switch (action)` in `SetObjectState` has no `default:` and the local
`Boolean changed` is left **uninitialised** (`GliderPRO/Sources/Objects.c:369`), so an
`action` of 3 would branch on and return garbage. A port should initialise
`changed = false` and treat `kOneShot` as unreachable.

The **appliance** block (`GliderPRO/Sources/Objects.c:590-628`) handles `kShredder`,
`kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kMicrowave` on
`data.g.state`, and then:

```
if ((changed) && (local != -1)) {
    masterObjects[local].theObject.data.g.state = newState;
    if (room == thisRoomNumber) {
        thisRoom->objects[object].data.g.state = newState;
        // note: the type test reads the HOUSE handle, not masterObjects
        if ((*thisHouse)->rooms[room].objects[object].what == kShredder)
            hotSpots[masterObjects[local].hotNum].isOn = newState;
    }
}
```

**Only `kShredder` propagates to `hotSpots[].isOn`.** Every other appliance's
`kDissolveIt` hot spot is hardcoded `isOn = true`, so switching a TV off does not make
it safe to touch.

The **enemy** block (`GliderPRO/Sources/Objects.c:637-671`) handles `kBalloon`,
`kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish` on
`data.h.state`, mirroring into `masterObjects` and `thisRoom` but **never touching
`hotSpots`** — correct, because their hot spots are `kIgnoreIt` placeholders (except the
static fish, whose lethal 36×33 rect stays live regardless).

Types explicitly declared unswitchable (`changed = false`): `kGuitar` (`:580-582`),
`kCinderBlock`/`kFlowerBox`/`kCDs`/`kCustomPict` (`:630-635`), `kCobweb` (`:673-675`),
clutter and `kChimes` (`:677-693`). `kStereo` is special: it is **not** in the appliance
block at all. Its case ignores `action`, does `newState = !isPlayMusicGame;
isPlayMusicGame = newState; changed = true;` (`:584-588`), and **never writes any
object's `data.g.state`** — so a stereo's stored state is inert and a switch wired to a
stereo just toggles the global background-music flag.

### 16.2 `HandleSwitches`

`HandleSwitches(hotPtr who)` (`GliderPRO/Sources/Interactions.c:985-1155`):

```
 1. if (who->stillOver)  return;
 2. resolve roomLinked/objectLinked/linkIndex from masterObjects[who->who];
 3. SetObjectState(roomLinked, objectLinked, data.e.type, linkIndex);
 4. draw the switch in its new position and play kSwitchSound/kSwitchPriority
      (for kLightSwitch, kMachineSwitch, kThermostat, kPowerSwitch, kKnifeSwitch);
 5. switch (masterObjects[linkIndex].theObject.what) {          // lines 1038-1150
      prizes:            RestoreFromSavedMap(...); AddSparkle(&bounds);   // see §17
      kCuckoo:           StopPendulum(...);
      kGreaseRt/kGreaseLf: SpillGrease(...);
      kSoundTrigger:     PlayPrioritySound(kTriggerSound, kTriggerPriority);
      lights:            RedrawRoomLighting();
      kShredder:         break;                                 // NOTHING (1086-1087)
      kToaster:          ToggleToaster(dynaNum);
      kMacPlus:          ToggleMacPlus(dynaNum);
      kGuitar:           PlayPrioritySound(kChordSound, kChordPriority);
      kTV:               ToggleTV(dynaNum);
      kCoffee:           ToggleCoffee(dynaNum);
      kOutlet:           ToggleOutlet(dynaNum);
      kVCR:              ToggleVCR(dynaNum);
      kStereo:           ToggleStereos(dynaNum);
      kMicrowave:        ToggleMicrowave(dynaNum);
      kBalloon:          ToggleBalloon(dynaNum);
      kCopterLf/Rt:      ToggleCopter(dynaNum);
      kDartLf/Rt:        ToggleDart(dynaNum);
      kBall:             ToggleBall(dynaNum);
      kDrip:             ToggleDrip(dynaNum);
      kFish:             ToggleFish(dynaNum);
   }
 6. who->stillOver = true;                                      // line 1154
```

Every hazard `Toggle*` in `GliderPRO/Sources/Trip.c` is literally
`dinahs[index].active = !dinahs[index].active;`:
`ToggleToaster` (`:22-25`), `ToggleOutlet` (`:70-73`), `ToggleBalloon` (`:104-107`),
`ToggleCopter` (`:111-114`), `ToggleDart` (`:118-121`), `ToggleBall` (`:125-128`),
`ToggleDrip` (`:132-135`), `ToggleFish` (`:139-142`).

Non-hazard toggles that also set `timer`: `ToggleMacPlus` sets `timer = 40`/`10`
(`:29-36`); `ToggleTV` starts/stops the QuickTime movie and sets `timer = 4` (`:40-58`);
`ToggleCoffee` (`:62-66`), `ToggleVCR` (`:77-81`), `ToggleMicrowave` (`:96-100`) set
`timer = 4`; `ToggleStereos` only acts `if (timer == 0)` (`:85-92`).

**The `dynaNum` may be −1** if the hazard did not become a dynamic (off-screen appliance,
non-central-room enemy, or the 18-slot budget exhausted). The `Toggle*` functions index
`dinahs[-1]` unchecked. In C this reads/writes memory before the array; in Go it panics.
A port must guard.

### 16.3 Triggers

```c
typedef struct { short object, room; short index, timer; short what; Boolean armed; } trigType;
trigType triggers[kMaxTriggers];     // kMaxTriggers == 16
```
(`GliderPRO/Sources/Triggers.c:12-28`)

`ArmTrigger(hotPtr who)` (`GliderPRO/Sources/Triggers.c:34-55`):

```
1. if (who->stillOver)  return;
2. where = FindEmptyTriggerSlot();                   // first !armed slot, else -1
3. if (where != -1) {
       whoLinked = who->who;
       triggers[where].room   = masterObjects[whoLinked].roomLink;
       triggers[where].object = masterObjects[whoLinked].objectLink;
       triggers[where].index  = whoLinked;
       triggers[where].timer  = masterObjects[whoLinked].theObject.data.e.delay * 3;
       triggers[where].what   = masterObjects[triggers[where].object].theObject.what;
       triggers[where].armed  = true;
   }
4. who->stillOver = true;
```

Note step 3's `.what` indexes `masterObjects[triggers[where].object]` where `object` is a
**room-relative object number**, not a masterObjects index. This only lines up for
central-room targets. `.what` is in fact never read anywhere — `FireTrigger` re-derives
the type — so it is dead.

`HandleTriggers` (`GliderPRO/Sources/Triggers.c:79-96`): for all 16 slots, if `armed`
then `timer--`, and at `<= 0` set `timer = 0`, `armed = false`, `FireTrigger(i)`.

`FireTrigger(short index)` (`GliderPRO/Sources/Triggers.c:100-194`): if
`masterObjects[triggerIs].localLink != -1`, dispatch on
`masterObjects[triggeredIs].theObject.what`:

| Target | Action | Cite |
|---|---|---|
| `kGreaseRt`, `kGreaseLf` | `SetObjectState(..., kForceOn, ...)` then `SpillGrease` | `:112-120` |
| `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch`, `kInvisSwitch` | `TriggerSwitch(dynaNum)` → `HandleSwitches(&hotSpots[dynaNum])` | `:122-129` |
| `kSoundTrigger` | `PlayPrioritySound(kChordSound, kChordPriority)` — with the source comment `// Change me` | `:131-133` |
| `kToaster` | `TriggerToast(dynaNum)` | `:135-137` |
| `kGuitar` | `kChordSound` | `:139-141` |
| `kCoffee` | `kCoffeeSound`/`kCoffeePriority` | `:143-145` |
| `kOutlet` | `TriggerOutlet(dynaNum)` | `:147-149` |
| `kBalloon` | `TriggerBalloon(dynaNum)` | `:151-153` |
| `kCopterLf`, `kCopterRt` | `TriggerCopter(dynaNum)` | `:155-158` |
| `kDartLf`, `kDartRt` | `TriggerDart(dynaNum)` | `:160-163` |
| `kDrip` | `TriggerDrip(dynaNum)` | `:165-167` |
| `kFish` | `TriggerFish(dynaNum)` | `:169-171` |

**`kBall` and `kMicrowave` have no trigger case** — a trigger cannot start a ball or fire
a microwave. Also note `TriggerSwitch(masterObjects[...].dynaNum)` indexes
`hotSpots[]` with a **dynaNum**, which is a bug in the original: switches never become
dynamics, so their `dynaNum` is −1. `TriggerSwitch(-1)` calls
`HandleSwitches(&hotSpots[-1])`. See §17.

The `else` branch (`GliderPRO/Sources/Triggers.c:174-193`) — used when the trigger's
target is *outside* the loaded 3×3 neighbourhood — handles only grease.

`ZeroTriggers` (`GliderPRO/Sources/Triggers.c:198-204`) clears all 16 `armed` flags.

**`Trigger*` semantics** (`GliderPRO/Sources/Trip.c`):

| Function | Cite | Behaviour |
|---|---|---|
| `TriggerToast` | `:153-167` | `if (!moving)`: if `active` → `vVel = -count; frame = 0; moving = true; kToastLaunchSound`; else `frame = timer` |
| `TriggerOutlet` | `:171-184` | `if (position == 0)`: if `active` → `position = 1; timer = kLengthOfZap (30); kZapSound`; else `timer = count` |
| `TriggerDrip` | `:188-192` | `if ((!moving) && (timer > 7))  timer = 7;` — forces the swell sequence |
| `TriggerFish` | `:196-205` | `if (active && !moving)`: `whole = dest; moving = true; frame = 4; kFishOutSound` |
| `TriggerBalloon` | `:209-213` | `if (!moving)  timer = kStartSparkle + 1;` = **5** |
| `TriggerCopter` | `:218-222` | same |
| `TriggerDart` | `:227-231` | same |

`TriggerBalloon`/`Copter`/`Dart` set `timer = 5`, which means the *next* frame's
`timer--` produces 4 == `kStartSparkle`, firing the materialisation sparkle
(`kEnemyInSound`), and the frame after that produces 3, then 2, 1, 0 → appear. So a
triggered enemy appears **5 frames** later, with a proper sparkle warning. Note these
three ignore `active` entirely — a switched-off balloon can still be triggered.

### 16.4 `SetObjectsToDefaults` — what gets reset on a new game

`SetObjectsToDefaults` (`GliderPRO/Sources/Play.c:603-708`) walks every room in the
house and:

| Lines | Types | Action |
|---|---|---|
| `:612-620` | all rooms | `visited = false` |
| `:624-637` | `kFloorVent`, `kCeilingVent`, `kFloorBlower`, `kCeilingBlower`, `kLeftFan`, `kRightFan`, `kSewerGrate`, `kInvisBlower`, `kGrecoVent`, `kSewerBlower`, `kLiftArea` | `data.a.state = data.a.initial` |
| `:639-655` | bonuses/prizes | `data.c.state = data.c.initial` |
| `:657-661` | `kDeluxeTrans` | nibble copy |
| `:663-673` | lights | `data.f.state = data.f.initial` |
| `:675-677` | `kStereo` | `data.g.state = isPlayMusicGame` |
| `:679-690` | `kShredder`, `kToaster`, `kMacPlus`, `kGuitar`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kMicrowave` | `data.g.state = data.g.initial` |
| `:692-702` | `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish` | `data.h.state = data.h.initial` |

**Absent from all of the above:** `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ` (their
state is never consulted), `kCobweb` (unswitchable), `kCinderBlock`, `kFlowerBox`,
`kCDs`, `kCustomPict` (unswitchable).

---

## 17. Instance limits

| Limit | Constant | Value | Enforced where |
|---|---|---:|---|
| Objects per room | `kMaxRoomObs` | 24 | file format; `FindEmptyObjectSlot` |
| Master objects (3×3 rooms) | `kMaxMasterObjects` | 216 | `GliderPRO/Sources/Objects.c:268` |
| Hot spots | `kMaxHotSpots` | 56 | `GliderPRO/Sources/ObjectRects.c:280` (in `AddActiveRect`) |
| Dynamic objects (dinahs) | `kMaxDynamicObs` | 18 | `GliderPRO/Sources/Dynamics3.c:193`; editor `GliderPRO/Sources/ObjectAdd.c:646-720` |
| Saved maps | `kMaxSavedMaps` | 24 | `BackUpToSavedMap` |
| Rubber bands in flight | `kMaxRubberBands` | 2 | `GliderPRO/Sources/RubberBands.c:258` |
| Candle-class flames | `kMaxCandles` | 20 | `GliderPRO/Sources/DynamicMaps.c:321` |
| Tiki flames | `kMaxTikis` | 8 | `GliderPRO/Sources/DynamicMaps.c:405` |
| BBQ coal sets | `kMaxCoals` | 8 | `GliderPRO/Sources/DynamicMaps.c:491` |
| Pendulums | `kMaxPendulums` | 8 | `AddPendulum`, `GliderPRO/Sources/DynamicMaps.c:575` |
| Stars | `kMaxStars` | 4 | `AddStar`, `GliderPRO/Sources/DynamicMaps.c:667` |
| Sparkles on screen | `kMaxSparkles` | 3 | `GliderPRO/Sources/DynamicMaps.c:174` |
| Sparkle animation modes | `kNumSparkleModes` | 5 | `RenderSparkles` |
| Flying score popups | `kMaxFlyingPts` | 3 | `AddFlyingPoint`, `GliderPRO/Sources/DynamicMaps.c:204` |
| Shredded-glider effects | `kMaxShredded` | 4 | `GliderPRO/Sources/DynamicMaps.c:732` (**with `>` not `>=`** — see below) |
| Grease spills | `kMaxGrease` | 16 | `AddGrease`, `GliderPRO/Sources/Grease.c:212` |
| Armed triggers | `kMaxTriggers` | 16 | `GliderPRO/Sources/Triggers.c:12` |
| Shredders per room (editor) | `kMaxShredded` | 4 | `GliderPRO/Sources/ObjectAdd.c:589-644` via `HowManyShredderObjects` |

**`kMaxShredded` is an off-by-one buffer overflow.** The `shreds` array is allocated as
exactly `kMaxShredded` = 4 elements (`shreds = NewPtr(sizeof(shredType) * kMaxShredded)`,
`GliderPRO/Sources/StructuresInit2.c:256`), so the valid indices are 0..3. But
`AddAShreddedGlider` guards with `if (numShredded > kMaxShredded) return;`
(`GliderPRO/Sources/DynamicMaps.c:732`) and then writes `shreds[numShredded]` before
incrementing. When `numShredded == 4` the test `4 > 4` is false, so the function writes
`shreds[4]` — one element past the end of the block. On classic Mac this scribbles on
whatever the Memory Manager put next; a Go port using a fixed-size array or slice will
panic instead. **Port the guard as `>=`, not `>`** (i.e. cap at 4 live shreds); the
overflowing 5th write is a bug, not behaviour any content depends on.

**Editor-side dynamic budget.** `AddNewObject` rejects a new enemy or hazard with
`ShoutNoMoreSpecialObjects()` when `HowManyDynamicObjects() >= kMaxDynamicObs`
(`GliderPRO/Sources/ObjectAdd.c:646-720`). `kCobweb` is exempt (it is not a dynamic).
So a room can never be authored with more than 18 dynamic-capable objects — but the
runtime budget is shared with appliances from all nine neighbour rooms, so it can still
be exhausted at play time.

**Empirically verified:** across all 22 shipped houses (4070 rooms), the maximum number
of dynamic-capable objects in a single room is exactly **18** — reached in `Teddy World`
room 244, "Moving walls". The authoring limit is actually hit in shipped content.

**Editor defaults for hazards** (`GliderPRO/Sources/ObjectAdd.c`):

| Type | Defaults | Cite |
|---|---|---|
| `kToaster` | `height = 64`, `delay = 10 + RandomInt(10)`, `initial = state = true` | `:589-644` |
| `kOutlet` | `height = 0`, `delay = 10 + RandomInt(10)` | `:589-644` |
| `kMicrowave` | `byte0 = 7` (kill bands + battery + foil) | `:589-644` |
| `kCustomPict` | `height = 10000` (PICT ID base) | `:589-644` |
| `kBalloon`, `kCopterLf/Rt` | `length = 0`, `delay = 10 + RandomInt(10)`, `byte0 = 0`, `topLeft.v = (kTileHigh/2) - HalfRectTall(...)` | `:646-693` |
| `kDartLf` | `topLeft.h` forced to `kRoomWide - RectWide(&srcRects[kDartLf])` = 448; `topLeft.v` = click v | `:646-693` |
| `kDartRt` | `topLeft.h` forced to 0 | `:646-693` |
| `kCobweb` | `length = 0`, `delay = 0`, `topLeft.v` = click v | `:646-693` |
| `kBall` | `length = 64`, `delay = 0`, centred on the click | `:695-720` |
| `kDrip`, `kFish` | `length = 64`, `delay = 10 + RandomInt(10)`, centred | `:695-720` |

**Editable fields.** `DoEnemyObjectInfo(what)` (`GliderPRO/Sources/ObjectInfo.c:2182-2285`)
exposes only `delay`, validated 0..255, and **hides the delay field entirely for
`kBall`** (`:2212-2217`). `DoApplianceObjectInfo` (`~1630-1740`) hides delay for
`kShredder`, `kMacPlus`, `kTV`, `kCoffee`, `kVCR`, `kMicrowave`. The
`length`/`height` field is **not editable in any dialog** — it is set by dragging a
marquee handle (`GliderPRO/Sources/ObjectEdit.c:307-321`, direction table `:1914-1932`:
`kToaster` → `kAbove` with `data.g.height`; `kBall`/`kFish` → `kAbove` with
`data.h.length`; `kDrip` → `kBelow` with `data.h.length`).

**Position parity rules** (`GliderPRO/Sources/HouseLegal.c`): `kTV` is forced to an
**odd** `topLeft.h`; `kToaster`, `kMacPlus`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`,
`kMicrowave`, `kBalloon`, `kCopterLf`, `kCopterRt`, `kBall`, `kDrip`, `kFish` are forced
to **even** `topLeft.h` (`GliderPRO/Sources/HouseLegal.c:494-511` for appliances,
`:543-553` for enemies). Darts, cobwebs and shredders are exempt. The census confirms
all of this: every `kTV` has odd `h`, every `kBalloon`/`kBall`/`kToaster`/`kMicrowave`
has even `h`, and `kShredder`/`kCobweb` have both parities. (Darts are exempt but happen
to be even in all 268 shipped instances, so do not infer a rule from the data alone.)
The reason is 8-bit
`CopyBits` alignment on 68k Macs; a Go port can ignore parity but must not "correct" the
stored data.

---

## 18. Empirical census of shipped houses

All 22 `GliderPRO/Houses/*.binhex` files were BinHex-4.0 decoded with a decoder written
for this analysis and parsed with python3 against the `houseType`/`roomType`/`objectType`
layout in §2. Total: **4070 rooms**.

### 18.1 Object counts

| Object | Count |
|---|---:|
| `kCustomPict` (0x6E) | 4782 |
| `kInvisBlower` (0x0D) | 2336 |
| `kBalloon` (0x71) | 500 |
| `kSparkle` (0x2D) | 486 |
| `kDrip` (0x77) | 477 |
| `kCopterLf` (0x72) | 259 |
| `kBall` (0x76) | 212 |
| `kCandle` (0x09) | 180 |
| `kCopterRt` (0x73) | 155 |
| `kDartLf` (0x74) | 151 |
| `kToaster` (0x62) | 140 |
| `kStubby` (0x0A) | 127 |
| `kFish` (0x78) | 120 |
| `kDartRt` (0x75) | 117 |
| `kOutlet` (0x67) | 100 |
| `kTaper` (0x08) | 90 |
| `kCobweb` (0x79) | 86 |
| `kTV` (0x65) | 80 |
| `kMacPlus` (0x63) | 74 |
| `kCDs` (0x6D) | 63 |
| `kTiki` (0x0B) | 58 |
| `kMicrowave` (0x6A) | 56 |
| `kChimes` (0x8F) | 54 |
| `kRightFan` (0x07) | 54 |
| `kShredder` (0x61) | 50 |
| `kLeftFan` (0x06) | 45 |
| `kBBQ` (0x0C) | 43 |
| `kCinderBlock` (0x6B) | 43 |
| `kCoffee` (0x66) | 38 |
| `kStereo` (0x69) | 36 |
| `kFlowerBox` (0x6C) | 35 |
| `kGuitar` (0x64) | 29 |
| `kVCR` (0x68) | 26 |

### 18.2 Observed field ranges

| Object | `length`/`height` | `delay` (off 6) | `byte0` (off 7) | `initial`/`state` | `topLeft.h` parity |
|---|---|---|---|---|---|
| `kBalloon` | always 0 | 0..255 (40 distinct) | always 0 | {0,1} | always even |
| `kCopterLf` | always 0 | 0..255 (37 distinct) | always 0 | {0,1} | always even |
| `kCopterRt` | always 0 | 0..255 (29 distinct) | always 0 | {0,1} | always even |
| `kDartLf` | always 0 | 0..53 (28 distinct) | always 0 | `initial` {0,1}, `state` always 1 | always even |
| `kDartRt` | always 0 | 2..255 (25 distinct) | always 0 | {0,1} | always even |
| `kBall` | 0..281 (115 distinct) | **always 0** | always 0 | {0,1} | always even |
| `kDrip` | 1..310 (178 distinct) | 0..255 (40 distinct) | always 0 | {0,1} | always even |
| `kFish` | 0..261 (74 distinct) | 0..255 (37 distinct) | always 0 | {0,1} | always even |
| `kCobweb` | always 0 | always 0 | always 0 | always 1/1 | either |
| `kToaster` | 9..277 (81 distinct) | **always 0** at off 6 | 0..255 (27 distinct) at off 7 | {0,1} | always even |
| `kOutlet` | 0 | 0 | 0..240 (27 distinct) | {0,1} | always even |
| `kMicrowave` | 0 | 0..7 (7 distinct) — the **kill mask** | 0 | {0,1} | always even |
| `kShredder` | 0 | 0 | 0 | {0,1} | either |
| `kGuitar` | 0 | 0 | 0 | always 1/1 | either |
| `kTV` | 0 | 0 | always 0 | {0,1} | always **odd** |
| `kCustomPict` | 10000..12351 (PICT ID) | — | — | — | — |

The toaster/microwave rows are the empirical proof of the union aliasing described in
§2.4: for `applianceType` the byte at payload offset 6 is `byte0` and at offset 7 is
`delay`, so a toaster shows its cycle time at offset 7 while a balloon
(`enemyType`: offset 6 = `delay`) shows it at offset 6.

### 18.3 Blower/flame field ranges

For `blowerType` the payload offsets are `topLeft` 0-3, `distance` 4-5, `initial` 6,
`state` 7, `vector` 8, `tall` 9.

| Object | `distance` (off 4) | `vector` (off 8) | `initial`/`state` (off 6/7) |
|---|---|---|---|
| `kTaper` | 0..206 (53 distinct) | always 1 | always 1/1 |
| `kCandle` | 0..240 (96 distinct) | always 1 | always 1/1 |
| `kStubby` | 0..237 (68 distinct) | always 1 | always 1/1 |
| `kTiki` | 0..213 (19 distinct) | always 1 | always 1/1 |
| `kBBQ` | 32..276 (29 distinct) | always 1 | always 1/1 |
| `kLeftFan` | 3..300 (39 distinct) | always 8 | {0,1} |
| `kRightFan` | 0..368 (44 distinct) | {2, 8} | {0,1} |
| `kInvisBlower` | 0..488 (319 distinct) | {1, 2, 4, 8, 17, 20} | {0, 1, **23**} |

(`tall`, at payload offset 9 for `blowerType`, is observed ∈ {0, 1, 3, 4, 5} for the
five flames — per flame: taper {0,1,4,5}, candle {0,1,3,4,5}, stubby {0,1,3,4,5},
tiki {0,1,4}, BBQ {0,1,4,5}. The fans and `kInvisBlower` use a much wider spread —
left fan {0,1,22,41}, right fan {0,1,13,16,85}, invisible blower
{0,1,2,3,4,5,11,13,16,22,41,47} — so `tall` is not a boolean for blowers.)

---

## 19. Mac Toolbox dependencies and Go replacements

| Toolbox facility | Where it is used for hazards | Go replacement |
|---|---|---|
| `Rect` = `{short top, left, bottom, right}` | everywhere; `QSetRect`, `QOffsetRect`, `HOffsetRect`, `VOffsetRect`, `InsetRect`, `SectRect`, `ZeroRectCorner`, `CenterRectInRect`, `RectWide`, `RectTall`, `HalfRectWide`, `IsRectLeftOfRect` | a `Rect{T,L,B,R int16}` value type with the identical helper set. **Do not switch to `image.Rectangle`** without care: QuickDraw's `SectRect` is exclusive on bottom/right like Go's, but `InsetRect` with a negative amount **grows** the rect (used for the cobweb) and QuickDraw does not normalise. |
| `Point` = `{short v; short h}` | `topLeft` in every object variant, `houseType.initial` | a struct with **v first** if you decode by field order, or explicit big-endian offsets. Getting the order wrong silently transposes every hazard. |
| 8-bit indexed-colour GWorlds (`CreateOffScreenGWorld` with `kPreferredDepth`) | all sprite sheets, `backSrcMap`, `workSrcMap`, `savedMaps[]` | RGBA or paletted `image.Paletted` offscreen buffers. The palette matters for `kRedOrangeColor8` = 23 and for `PaintRect` (which paints in the *current fore colour*, normally black). |
| 1-bit mask GWorlds + `CopyMask` | every hazard sprite | premultiplied alpha or an explicit 1-bit stencil blit. `CopyMask(src, mask, dst, srcRect, maskRect, dstRect)` copies `src` where `mask` is **black** (1 bits in a 1-bit map). |
| `CopyBits(..., srcCopy, nil)` | flame strips, idle fish, appliance screens | plain opaque copy — **no transparency**. The idle fish specifically relies on this (`GliderPRO/Sources/Dynamics.c:278-280`). |
| `GetPortBitMapForCopyBits` / `GetGWorldPixMap` | `BackUpFlames`, all `Render*` | not needed; just address the buffer. |
| `SetGWorld` / `GetGWorld` | every draw helper saves and restores the current port | an explicit destination argument. |
| Resource Manager (`GetPicture`, `GetResource('snd ')`, `HLock`, `HGetState`, `HSetState`) | all PICTs, all sounds, `thisHouse` handle | embed decoded assets; the `HLock`/`HSetState` dances around `thisHouse` are pure Memory-Manager bookkeeping and vanish. |
| PICT decoding (v1 opcode stream and v2 `$0011`+`$02FF`) | all 40 hazard-relevant PICTs | a PICT decoder handling both versions, plus the `picFrame`-origin quirk on masks 5006 and 5010. |
| Sound Manager, 3 `SndChannelPtr`s, `bufferCmd` | `PlayPrioritySound` | three mixer voices with the exact argmin-priority arbitration of §5.2. The 20-byte skip into each `'snd '` resource must be replicated or replaced by pre-extracted PCM. |
| `TickCount()` busy-wait | `RenderFrame` step 11 | a fixed-timestep loop at 2/60.15 s ≈ 33.26 ms. Do **not** decouple hazard updates from the 30 fps tick — every velocity constant in this document is px/frame. |
| QuickTime `Movie` | `kTV` only | out of scope for hazards; `ToggleTV` must still flip `active` and set `timer = 4` even with no movie. |
| `RandomInt(n)` | sparkle timers, flame start phase, coffee timer | any PRNG returning 0..n−1. Flame/coal/tiki start phases are cosmetic; sparkle timers are cosmetic; **no hazard's lethality depends on randomness**. |
| Big-endian struct reads from the data fork | `houseType`, `roomType`, `objectType` | explicit `binary.BigEndian` decoding at the offsets in §2. |
| 68k/PPC `short` arithmetic | every velocity | `int16` semantics. `-((vVel * 3) / 4)` in `HandleBall` and `>> 3` in `WebGlider` are **integer** operations with C truncation / arithmetic shift respectively. |

---

## 20. Bugs and quirks in the original that a port must decide about

1. **`DidBandHitDynamic` reads an uninitialised `collided`** when `numBands == 0`
   (`GliderPRO/Sources/Dynamics.c:80-106`). Unreachable because all three call sites
   guard with `numBands > 0`. Initialise to `false` in the port and keep the guard.
2. **`HandleSwitches` calls `AddSparkle(&bounds)` with an uninitialised local `bounds`**
   for the prize cases (`GliderPRO/Sources/Interactions.c:1050`). The sparkle appears at
   a garbage location. Genuine bug; a port should either use the prize's real rect or
   suppress the sparkle. This is not a hazard path but shares the sparkle pool with
   hazards.
3. **`AddAShreddedGlider` uses `if (numShredded > kMaxShredded) return;`**
   (`GliderPRO/Sources/DynamicMaps.c:732`) — `>` rather than `>=`, so a **fifth** entry
   can be written into a 4-element array. Off-by-one buffer overrun. Use `>=`.
4. **`HandleBands`'s kill loop is a `do…while`** that executes at least once even when
   `numBands` has just become 0 (`GliderPRO/Sources/RubberBands.c:241-251`), reading
   `bands[0].mode` when no band is live. If slot 0 still holds `kKillBandMode` from a
   previous frame, the inner `while` then calls `KillBand(0)` with `numBands == 0`, and
   `KillBand` computes `lastBand = numBands - 1` = -1, does `bands[0] = bands[-1]`
   (a read *before* the block) and leaves `numBands == -1`
   (`GliderPRO/Sources/RubberBands.c:294-303`). In practice `KillBand` also zeroes
   `mode` first (`:246`) so the `while` cannot spin, and the game clears bands between
   rooms, which is why this never showed up. Use a `for` loop with an explicit bound and
   guard `KillBand` on `numBands > 0`.
5. **`case kLgTrigger:` is used as a hot-spot action label**
   (`GliderPRO/Sources/Interactions.c:1352`). `kLgTrigger` = 0x48 = 72 is an *object*
   code; no `AddActiveRect` ever stores 72 as an action, so the label is dead. Drop it.
6. **`HandleBall` and `AddDynamicObject`(kBall, kFish) write the global `evenFrame`**
   (`GliderPRO/Sources/Dynamics2.c:420`, `GliderPRO/Sources/Dynamics3.c:474`,
   `:524`; the two `AddDynamicObject` writes are vestigial — the solve loops use the
   local `lilFrame`). This perturbs flame-vs-star rendering and every other `evenFrame` consumer
   for one frame. Reproduce it if you want frame-exact fidelity; use a local in the
   solver otherwise.
7. **`AddDynamicObject`'s dart case never assigns `dinahs[].room`**
   (`GliderPRO/Sources/Dynamics3.c:437-463`). Harmless (only outlets read `room`), but
   Go's zero value differs from the C leftover.
8. **`ZeroDinahs` does not clear `moving` or `byte1`**
   (`GliderPRO/Sources/Dynamics3.c:160-180`). Every add case sets both, so harmless.
9. **`Toggle*` and `Trigger*` are called with `dynaNum`, which may be −1.** In C this
   is an out-of-bounds access; in Go it panics. Guard every call site.
10. **`TriggerSwitch(masterObjects[triggeredIs].dynaNum)`**
    (`GliderPRO/Sources/Triggers.c:128`) passes a *dynamic* index into
    `hotSpots[]`. Switches never become dynamics, so `dynaNum` is −1 and this always
    indexes `hotSpots[-1]`. The intent was clearly `hotNum`. A trigger wired to a
    switch is therefore broken in the original. A port must choose: replicate
    (trigger-to-switch does nothing / crashes) or fix (use `hotNum`). Fixing changes
    shipped-level behaviour.
11. **`HandleMicrowaveAction` never sets `who->stillOver`**
    (`GliderPRO/Sources/Interactions.c:1159-1194`), so its "once per contact" guard
    never engages and `kMicrowavedSound` re-fires every frame the glider is fully inside
    the column. Because the drains zero rather than decrement, gameplay is unaffected;
    only the sound repeats.
12. **Flame (both the `kLiftIt`/`kBurnIt` columns and the `kDissolveIt` body), fan-blade,
    guitar, `kDissolveIt` appliance body, `kMicrowave` body *and* its `kMicrowaveIt`
    column, `kFish` static and `kCobweb` hot spots all pass the literal `true` for
    `isOn`**, so none of them can be switched off at the hot-spot level. The
    state-gated hazard hot spots are exactly: `kShredder`
    (`GliderPRO/Sources/ObjectRects.c:948-949`), `kOutlet`'s `kIgnoreIt` (`:965-966`),
    the fan *push* columns (`:380-381`, `:395-396`) and the `kInvisBlower` /
    `kGrecoVent` / `kSewerBlower` / `kLiftArea` push columns (`:531-561`, `:574`, `:586`,
    `:598-613`). The microwave *does* still honour its `state` — but inside
    `HandleMicrowaveAction` (`GliderPRO/Sources/Interactions.c:1169`), not via `isOn`.
13. **`RenderShreds` plays `kShredSound` every frame** during the 35-frame reveal
    (`GliderPRO/Sources/Render.c:559-612`) *and* `MoveGliderShredding` plays it every
    frame during the 20-frame consumption (`GliderPRO/Sources/Player.c:1290`). With
    priority 903 this dominates the mixer for over a second.
14. **`srcRects[kFlower]` is never initialised** in `InitSrcRects`
    (`GliderPRO/Sources/StructuresInit2.c:306-475`). Not a hazard, but the table has a
    hole; `NewPtr` memory is uninitialised on classic Mac OS.
15. **The copter respawn uses a hardcoded `dest.right = dest.left + 32`**
    (`GliderPRO/Sources/Dynamics2.c:211`) rather than `RectWide(&copterSrc[0])`. Line 211
    is in the shared `HandleCopter` respawn block, so this applies to **both**
    `kCopterLf` and `kCopterRt`. Same value today.
16. **Dart boundary tests would fire on the spawn frame** if evaluated before the move
    (`GliderPRO/Sources/Dynamics2.c:296-298` vs the spawn at `:312-322` /
    `GliderPRO/Sources/Dynamics3.c:443-452`). Preserve the move-then-test order.
17. **Shipped `kInvisBlower` records contain `initial`/`state` = 23**, a non-boolean
    value in a `Boolean` field. Decode as "non-zero is true".
18. **PICT 5003 (the switch mask) does not exist**, and mask 5005 is one scanline
    shorter than its image. Do not assert image/mask congruence.
19. **`CreateActiveRects(n)` is called with a room-object index but indexes
    `masterObjects[n]`** (`GliderPRO/Sources/Objects.c:284` vs
    `GliderPRO/Sources/ObjectRects.c:304`). Correct only because the central room is
    listed first. Make the invariant explicit or pass the master index.
20. **`ArmTrigger` writes `triggers[].what` from
    `masterObjects[triggers[where].object]`**, mixing a room-object number with a master
    index (`GliderPRO/Sources/Triggers.c:50`). The field is never read, so it is dead
    code — do not port it.

---

## Open questions

1. **What sets `hotSpots[].stillOver` for `kMicrowaveIt`?** Nothing in
   `HandleHotSpotCollision`'s `kMicrowaveIt` case
   (`GliderPRO/Sources/Interactions.c:1581-1584`) nor in `HandleMicrowaveAction`
   (`:1159-1194`) assigns it, yet `HandleMicrowaveAction` guards on it. I could not find
   any other writer for that specific hot spot, so I believe the guard is simply dead —
   but I cannot rule out an assignment in a path I did not read (e.g. a `FlagStillOvers`
   interaction on room entry, which does set it, but only while the glider is inside).
   **Consequence if I am wrong:** the microwave would drain once per entry rather than
   once per frame; since it zeroes rather than decrements, only the repeated
   `kMicrowavedSound` distinguishes the two.
2. **Was `TriggerSwitch(dynaNum)` (`GliderPRO/Sources/Triggers.c:128`) intended to be
   `hotNum`?** I am confident it is a bug, but I did not find a shipped house that
   actually wires a trigger to a switch, so I cannot say what the observable behaviour
   was on a real Mac (it would read `hotSpots[-1]`, i.e. whatever precedes the
   allocation). A census of trigger→switch links across the 22 houses would settle
   whether any shipped level depends on it.
3. **Exact QuickDraw palette entries.** I verified `kRedOrangeColor8` = 23
   (`GliderPRO/Headers/GliderDefines.h:542`) with the source comment "actually, 18", but
   I did not decode the `clut` resource, so I cannot give RGB values for the 8-bit
   palette used by the sprite sheets. This affects colour fidelity, not behaviour.
4. **`RectWide(&dartSrc[0])` vs `srcRects[kDartLf]`.** Both are 64, but the respawn code
   uses the former (`GliderPRO/Sources/Dynamics2.c:313`) and the editor's position
   clamp uses the latter (`GliderPRO/Sources/ObjectAdd.c:646-693`). If a house was
   authored with a custom dart PICT of a different width the two would disagree. I found
   no mechanism for custom enemy art, so I believe this cannot happen, but I did not
   exhaustively check the `kUserStructureRange` (3300) custom-graphics path.
5. **Whether `kFish.delay == 0` is reachable in play.** The census shows shipped fish
   with `delay` down to 0, giving `hVel = 0` and hence `timer = 0` at spawn, so the fish
   leaps on the very first frame and then every frame after re-entry. I traced no guard
   against this and believe it is intended ("continuous fish"), but I did not observe it
   running.
6. **The precise semantics of `blowerType.tall`.** The struct comments it as a plain
   `Byte` (`GliderPRO/Headers/GliderStructs.h:18`) and the census shows values
   {0,1,3,4,5} on flames, but I found no reader for it in any hazard path. It may be
   used only by the editor's drawing code, which I did not read exhaustively.
7. **What `dinahs[].byte1` is for.** It is written as 0 by every `AddDynamicObject` case
   and never read anywhere I found. It may be vestigial.
8. **Two-player `stillOver` semantics.** `CheckForHotSpots`'s two-player branch
   (`GliderPRO/Sources/Interactions.c:1636-1676`) clears `stillOver` only when *neither*
   glider overlaps. That means player 2 standing in a switch prevents player 1 from
   re-triggering it. I read the code but did not verify against a two-player recording
   that this is the shipped behaviour rather than a compile-time-disabled path
   (`BUILD_ARCADE_VERSION` = 1 is defined at `GliderPRO/Headers/GliderDefines.h:16` and I
   did not trace every `#ifdef`).

---

## Porting notes

### Get these right first — they are the highest-leverage fidelity risks

1. **Everything is px/frame at exactly 30.07 fps.** `kTicksPerFrame` = 2 ticks of
   1/60.15 s. Do not normalise velocities to px/second and do not use a variable
   timestep. `HandleBalloon`'s `vVel = -2` means "two pixels per frame", full stop.
2. **`evenFrame` is a single global boolean flipped once per frame, and two hazard
   functions write it.** Model it as engine state, not as `frameCount % 2`, or you will
   not reproduce the perturbation from `HandleBall`.
3. **Three different gravity rates.** Toast `vVel++` every frame; ball/drip/fish
   `vVel++` on `evenFrame` only; rubber bands `vVel++` every 4th frame
   (`kBandFallCount`). Getting toast on half-gravity is the single most likely
   behavioural bug.
4. **Three different animation rates.** Balloon and toast advance on `evenFrame`
   (15 fps); copter advances every frame (30 fps); dart never animates; fish only
   advances while descending; drip toggles 4↔5 on `evenFrame`; flames advance on
   `evenFrame` only because `RenderFlames` itself is called on `evenFrame`.
5. **`count`'s sign differs between toast (positive) and ball/fish (negative).** See
   §2.5.
6. **The launch-velocity solvers are `do…while`, so `length == 0` yields
   `velocity == 1`, not 0.** Shipped data contains `length == 0`.
7. **`Point` is `{v, h}`.** Decode by explicit offset.
8. **The `objectType.data` union aliases `height`/`length` at payload offset 4,
   `byte0`/`delay` at offset 6, and `delay`/`byte0` at offset 7.** In Go, decode the
   10-byte payload once into a raw `[10]byte` (or a single struct with both names as
   accessors) rather than into per-variant structs with independent fields. The fish
   reading `data.g.height` while the editor writes `data.h.length` is the canonical trap.
9. **Only balloon, copter and dart are destructible, and only by a rubber band.** They
   are not removed; they enter a falling state and respawn after `count` frames.
10. **The seven appliance dinahs (`kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`,
    `kStereo`, `kMicrowave`) keep `dest` in playOrigin-inclusive coordinates; every
    other dinah — `kSparkle`, `kToaster` and the six mobile enemies — is room-local**
    (`GliderPRO/Sources/Dynamics3.c:199-390`). The outlet is the only one of the seven
    that collides with the glider, so `CheckDynamicCollision`'s `doOffset` flag is
    passed `TRUE` there and `FALSE` everywhere else. If you unify the coordinate space,
    remove the flag; if you keep the split, keep it — and remember the split also
    affects the `AddRectToWorkRects` calls, which add `playOrigin` for the room-local
    dinahs but not for the appliances.
11. **Foil drains at different rates in different handlers.**
    `CheckDynamicCollision` gates on `evenFrame` (one charge per 2 frames);
    `kShredIt` (`GliderPRO/Sources/Interactions.c:1331-1336`) and `kBurnIt` (`:1363-1369`)
    take one per frame; and `kDissolveIt` takes **two** per frame — once inside
    `GliderHitTop` (`:82`) and once again on return (`:1230-1235`). So the same
    `kFoilSupply` = 8 lasts 16 frames against a balloon, 8 against a shredder or flame,
    and only 4 against a toaster body.
12. **Foil does not protect against landing on top of a `kDissolveIt` object, the
    microwave column, or a cobweb.**
13. **A ball is lethal while stationary** (its `CheckDynamicCollision` call is outside
    the `moving` test) — the only hazard with that property.
14. **`RenderBall` and `RenderDrip` have no `moving` gate; `RenderToast`, `RenderBalloon`,
    `RenderCopter`, `RenderDart` do; `RenderFish` switches from `CopyMask` to opaque
    `CopyBits` when idle.**
15. **Move first, then test bounds.** Dart, balloon and copter all spawn exactly on a
    despawn boundary; the original survives only because branch A/B moves before
    condition C is evaluated.

### Structural advice

16. **Keep the four-table architecture.** It is tempting to collapse
    `objects`/`masterObjects`/`hotSpots`/`dinahs` into one entity list, but the
    *lifetimes* differ (objects persist, masterObjects and hotSpots are rebuilt on room
    change, dinahs are rebuilt on room change but only for the central room for
    hazards), and switches/triggers navigate between them by index. A single ECS with
    stable IDs works, but you must reproduce "hazards from the central room only,
    appliances from all nine neighbours" and the shared 18-slot budget.
17. **Make `dynaNum == -1` and `hotNum == -1` explicit `Option`-style values** and
    check them before every `dinahs[...]`/`hotSpots[...]` access. The original does not,
    and Go will panic where C silently corrupted memory.
18. **Represent the `dynaType` field overloading with per-type views.** A
    `type BalloonState struct{...}` layered over a common record keeps the pseudocode in
    §11 readable without losing the exact semantics. Do **not** rename the fields in the
    canonical record — a porter diffing against `Dynamics2.c` needs `hVel`, `vVel`,
    `count`, `frame`, `timer`, `position` to mean what the C says.
19. **Integer arithmetic.** `-((vVel * 3) / 4)` (ball damping) is C truncation toward
    zero on a positive operand; `>> 3` (web pull) is an arithmetic shift, which floors
    for negative values. Use `int16`/`int` division and `>>` respectively, never floats.
20. **Sound arbitration is observable.** Three voices, argmin priority with ties toward
    channel 0, drop only if all three hold strictly higher priorities, plus the special
    "at most one `kTriggerPriority` sound" rule. Hazard sounds cluster in the 300–413
    band and are routinely dropped by 900-band death sounds; a naive "play everything"
    mixer sounds noticeably different.
21. **Flames are pre-composited, not alpha-blended.** `BackUpFlames` bakes each of the 5
    frames over the room background into a 16×75 buffer and `RenderFlames` does a plain
    strip copy. You can implement flames as ordinary alpha sprites *provided* you also
    handle the dirty-rect bookkeeping (`AddRectToWorkRects(&flames[i].dest)`), but if you
    want pixel-exact output you must reproduce the bake, because the bake captures the
    background *as it was when the room was entered*.
22. **Dirty-rectangle bookkeeping is load-bearing for correctness, not just speed.**
    `dinahs[].whole` is the union of the previous and current positions, computed
    differently per hazard (see the `whole.top -= vVel` / `whole.bottom -= vVel` /
    `whole.right -= hVel` / `whole.left -= hVel` variations). If you render with a full
    per-frame clear you can ignore `whole` entirely — but then `AddRectToWorkRects`
    calls at despawn time (balloon C1, copter C1, dart C1, drip A4a, fish A4a, toast
    A6a) become no-ops and you must make sure nothing else depended on them.
23. **`kMaxGarbageRects` = 48** (`GliderPRO/Sources/Render.c:20`) sizes the dirty-rect
    arrays, but the guards are `if (num… < (kMaxGarbageRects - 1))`
    (`GliderPRO/Sources/Render.c:67`, `:86`, `:105`), so the **effective cap is 47
    entries** (indices 0..46) and slot 47 is never used. The original silently drops
    rects past that. A full-redraw port sidesteps it.
24. **Two-player mode multiplies every collision site.** Each of the six mobile-hazard
    handlers has the same 16-line `twoPlayerGame` / `onePlayerLeft` / `playerDead`
    dispatch (e.g. `GliderPRO/Sources/Dynamics2.c:44-60`). Factor it into one
    `forEachLiveGlider(func(*Glider))` helper rather than copying it seven times.
25. **Do not "fix" the shipped data.** Even `topLeft.h` for appliances, odd for TVs,
    `state` bytes of 23, `delay == 0` — all of it is in the shipped houses and all of it
    must round-trip.
26. **Test vectors worth building first:** (a) a balloon with `delay = 0` (continuous
    respawn); (b) a ball with `length = 0` (`velocity = 1`); (c) a toaster with
    `height = 277` (`velocity = 24`, 49-frame flight); (d) a `kDartLf` at
    `topLeft.v = 200` (exits through the floor line, not the wall); (e) a drip with
    `delay = 1` (`count = 3`, swell frames partly skipped); (f) a foil-clad glider
    grinding on a shredder (8 frames to strip) versus on a moving balloon (16 frames);
    (g) a cobweb with the player holding a direction key (quarter-pull every 2 frames,
    death at frame 150).

---

## Appendix A (appended 2026-09-10 by the object-dynamics pass) — retraction of the `TriggerSwitch(dynaNum)` claim, and closure of Open Question 2

This appendix corrects a claim made twice above and answers this document's own Open Question 2. It
is additive: nothing else in this document has been altered. The full switch/trigger model now lives
in `docs/analysis/object-dynamics.md` §8.

### A.1 What §16.3 and hazard-list item 10 assert

* §16.3 (`| kLightSwitch … kInvisSwitch | TriggerSwitch(dynaNum) → HandleSwitches(&hotSpots[dynaNum]) |`)
  states: "`TriggerSwitch(masterObjects[...].dynaNum)` indexes `hotSpots[]` with a **dynaNum**, which
  is a bug in the original: switches never become dynamics, so their `dynaNum` is −1.
  `TriggerSwitch(-1)` calls `HandleSwitches(&hotSpots[-1])`."
* Hazard-list item 10 states: "Switches never become dynamics, so `dynaNum` is −1 and this **always**
  indexes `hotSpots[-1]`. The intent was clearly `hotNum`. A trigger wired to a switch is therefore
  broken in the original."
* Open Question 2 asks whether `TriggerSwitch(dynaNum)` was intended to be `hotNum`, and asks for a
  census of trigger→switch links.

### A.2 The premise is wrong: `dynaNum` deliberately holds `hotNum` for switches

`DrawARoomsObjects` (`GliderPRO/Sources/ObjectDrawAll.c:23-965`) uses a local `dynamicNum`,
initialised to −1 for every room-object slot (`:45`), and writes it into the master list at the end of
each iteration:

```c
if (!redraw)                                        // ObjectDrawAll.c:953   "set up links"
{
    for (n = 0; n < numMasterObjects; n++)
    {
        if ((masterObjects[n].objectNum == i) &&
                (masterObjects[n].roomNum == localNumbers[neighbor]))
            masterObjects[n].dynaNum = dynamicNum;   // :959
    }
}
```

All **six** real switch types set `dynamicNum` from `hotNum`, not from an `AddDynamicObject` return
value:

| Case | Line | Statement |
|---|---|---|
| `kLightSwitch` | `ObjectDrawAll.c:518` | `dynamicNum = masterObjects[i].hotNum;` |
| `kMachineSwitch` | `:531` | `dynamicNum = masterObjects[i].hotNum;` |
| `kThermostat` | `:544` | `dynamicNum = masterObjects[i].hotNum;` |
| `kPowerSwitch` | `:557` | `dynamicNum = masterObjects[i].hotNum;` |
| `kKnifeSwitch` | `:570` | `dynamicNum = masterObjects[i].hotNum;` |
| `kInvisSwitch` | `:573-575` | the entire case body is `dynamicNum = masterObjects[i].hotNum; break;` |

`kInvisSwitch` is decisive: its case does **nothing else at all** — it draws nothing, tests nothing,
and exists solely to perform that assignment. So `dynaNum` is not −1 for switches; it is an
intentional alias for `hotNum`, and `TriggerSwitch(short who) { HandleSwitches(&hotSpots[who]); }`
(`GliderPRO/Sources/Trip.c:146-149`) is receiving exactly the index it wants.

Contrast the three non-switch members of the family, which leave `dynamicNum` at −1:

```c
case kTrigger:
case kLgTrigger:
case kSoundTrigger:
break;                                              // ObjectDrawAll.c:577-580
```

**Retraction:** §16.3's "which is a bug in the original" and hazard-list item 10's "The intent was
clearly `hotNum`. A trigger wired to a switch is therefore broken in the original" are both
incorrect. Do **not** "fix" `Triggers.c:128` to read `hotNum`; it already effectively does.

### A.3 The residual, narrower defect

There *is* a real mis-index, but it is not the one described. In the assignment
`dynamicNum = masterObjects[i].hotNum`, the loop variable `i` is the **room-object slot** (0..23),
while `masterObjects[]` is indexed over the whole nine-room neighbourhood (0..215). Because
`ListAllLocalObjects` enrols the central room first, central-room object `i` lands at
`masterObjects[i]` — so the expression is correct **only for the central room**. For a switch in one
of the eight neighbour rooms it reads the *central* room's object `i`, whose `hotNum` is some
unrelated hot spot or −1.

Compounding it, `CreateActiveRects` is called only for the central room
(`GliderPRO/Sources/Objects.c:283-286`), so every neighbour-room object has `hotNum == -1` anyway.
The net effect for a neighbour-room switch is `dynaNum` = the central room's object-`i` `hotNum`,
i.e. either a wrong-switch throw or −1.

### A.4 Census, answering Open Question 2

Parsed from all 22 shipped `*.binhex` houses in `GliderPRO/Houses/` (4 070 non-empty rooms, 1 685
switch/trigger objects, 1 552 of them linked):

| Question | Answer |
|---|---|
| Do any shipped houses wire a trigger to one of the six real switches? | **Yes — 171 links:** `kTrigger` → `kInvisSwitch` ×99 and `kLgTrigger` → `kInvisSwitch` ×72. No shipped trigger targets `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch` or `kKnifeSwitch` |
| Trigger → trigger links | 3 (`kTrigger` → `kTrigger`), all no-ops: `FireTrigger` has no case for `kTrigger`/`kLgTrigger` |
| Are any of them **cross-room**? | **No.** All 219 linked `kTrigger`s except one are same-room; all 81 `kLgTrigger`s are same-room |
| The single cross-room trigger link | Land of Illusion, room 1 "Steamy Star", object 16 → room 174 "Edge Of Light", object 15, whose `what` is **−1 (an empty slot)**. ΔSuite = −5, ΔFloor = +1, i.e. **outside** the 3×3 neighbourhood |
| So is the neighbour-room mis-index (§A.3) reachable in shipped data? | **No.** Every trigger→switch link in the corpus targets a switch in the **central** room, where `masterObjects[i]` is the correct entry. The one cross-room trigger takes `FireTrigger`'s `else` branch (its `localLink` is −1) and matches nothing |

**Open Question 2 is therefore closed:** the observable behaviour on a real Mac was *correct* — 171
shipped trigger→switch links all throw the right switch. A Go port should implement
`TriggerSwitch(masterObjects[triggeredIs].dynaNum)` as "look up the target's hot-spot index and call
`HandleSwitches` on it", and should additionally assert `dynaNum != -1` (which never fires for the
shipped corpus).

### A.5 Two further corrections to §16.3

1. **`FireTrigger`'s `else` branch handles more than grease.** §16.3 says "The `else` branch
   (`GliderPRO/Sources/Triggers.c:174-193`) — used when the trigger's target is *outside* the loaded
   3×3 neighbourhood — handles only grease." That is true of what it *attempts*, but the branch
   computes `triggeredIs` from a `localLink` that is by construction −1 and then reads
   `masterObjects[-1].theObject.what`, so it is a wild read, not merely a grease-only path. A Go port
   must early-return here.
2. **`ZeroTriggers` clears only the `armed` flag**, not `object`, `room`, `index`, `timer` or `what`
   (`GliderPRO/Sources/Triggers.c:198-204`). §16.3's "clears all 16 `armed` flags" is accurate;
   noted here only because a port that zeroes the whole struct is equivalent but not identical, and
   `FindEmptyTriggerSlot` (`:59-75`) relies solely on `armed`.
3. **Markdown rendering defect, not a content error:** the `kDartVelocity` row of the
   "Hazard-local bounds" table in §8 contains an unescaped pipe inside a code span (`` `|hVel|` ``),
   which GFM parses as two extra cell boundaries, so that one row renders with seven columns instead
   of five. The intended text is "dart |hVel|", i.e. the absolute value of the dart's horizontal
   velocity, which is **6**. Escape the inner pipes as `` `\|hVel\|` `` when next editing that
   section. I have not modified it here because this appendix is additive only.

### A.6 Answers to three more of this document's open questions

* **Open Question 7, "What `dinahs[].byte1` is for":** confirmed vestigial. Written as 0 by all 18
  `AddDynamicObject` cases, **not** cleared by `ZeroDinahs` (`GliderPRO/Sources/Dynamics3.c:160-180`
  resets 12 of `dynaType`'s 14 members, omitting `byte1` and `moving`), and read nowhere. `byte0`, by
  contrast, holds the room-object slot index 0..23 and *is* read.
* **Open Question 1, "What sets `hotSpots[].stillOver` for `kMicrowaveIt`":** no new evidence; the
  guard still appears dead. Recorded here only so the two documents do not disagree.
* **`kBall` and `kMicrowave` have no trigger case** — confirmed. Neither does any of the eight light
  types, `kShredder`, `kMacPlus`, `kTV`, `kVCR`, `kStereo`, or `kSlider`. The complete `FireTrigger`
  dispatch table is in `docs/analysis/object-dynamics.md` §8.6.
