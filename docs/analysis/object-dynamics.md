# Object Dynamics: Blowers, Transport, Switches, Lights, Appliances

## Scope

This document specifies the per-frame runtime behaviour of every **non-enemy dynamic object** in
Glider PRO 1.0.4: air-flow sources (vents, blowers, fans, flames, lift areas), transport objects
(mailboxes, doors, windows, stairs, ducts, teleporters), switches and triggers, lights, appliances,
rubber bands, grease, and animated clutter. For each object it gives the on-disk record layout, the
hot-spot geometry that makes it interactive, the trigger condition, the effect on the glider and/or
on other objects, the exact timing constants, and the sound played.

Explicitly **out of scope** (covered by sibling documents in this directory):

* Enemy AI and enemy-specific collision — `docs/analysis/enemies.md`.
* Glider physics integration, hot-spot dispatch internals, prize/reward scoring —
  `docs/analysis/interactions.md`.
* The full object-code taxonomy and sprite atlas — `docs/analysis/object-taxonomy.md`.
* Frame loop, GWorld/dirty-rect architecture, house file container —
  `docs/analysis/architecture.md`.

Where those documents and this one overlap (hot-spot actions, the link mechanism, `SetObjectState`)
this document repeats the material rather than cross-referencing, because the object dynamics cannot
be ported without it. The one deliberate exception is [Section 10](#10-appliance-dynamics): the
general dynamic-object model (`dynaType`, the `dinahs[18]` budget, `AddDynamicObject`,
`CheckDynamicCollision`) and the *lethal* behaviour of appliance bodies are specified in
`docs/analysis/enemies.md` §12, so §10 here confines itself to the timing, the RNG consumption and
the spawn rules that §12 does not give. `docs/analysis/enemies.md` also now carries an
**Appendix A**, appended by this pass, retracting its claim that `TriggerSwitch(dynaNum)` is a bug
and closing its Open Question 2 with a census — see [Section 8.4.1](#84-the-instantaneous-path-handleswitches).

All source line numbers are cited against **LF-converted copies** of the original CR-only Classic Mac
text files. The conversion is byte-for-byte lossless (`tr '\r' '\n'`), so line *numbers* are
identical to the originals. Paths are given relative to the repository root
(the repository root).

## Sources read

Read in full:

| File | Lines | What was taken from it |
|---|---|---|
| `GliderPRO/Sources/Dynamics.c` | 776 | `CheckDynamicCollision`, `DidBandHitDynamic`, the 7 `Render*` functions, `HandleSparkleObject`, `HandleToast`, `HandleMacPlus`, `HandleTV`, `HandleCoffee`, `HandleOutlet`, `HandleVCR`, `HandleStereo`, `HandleMicrowave` |
| `GliderPRO/Sources/Dynamics2.c` | 589 | `HandleBalloon`, `HandleCopter`, `HandleDart`, `HandleBall`, `HandleDrip`, `HandleFish` |
| `GliderPRO/Sources/Dynamics3.c` | 555 | `HandleDynamics`, `RenderDynamics`, `ZeroDinahs`, `AddDynamicObject` (18 per-type cases) |
| `GliderPRO/Sources/Triggers.c` | — | `trigType`, `ArmTrigger`, `HandleTriggers`, `FireTrigger`, `ZeroTriggers` |
| `GliderPRO/Sources/RubberBands.c` | — | `kRubberBandVelocity`, `CheckBandCollision`, `HandleBands`, `AddBand`, `KillAllBands` |
| `GliderPRO/Sources/Grease.c` | — | grease modes 0-3, `HandleGrease`, `BackupGrease`, `AddGrease`, `SpillGrease`, `RedrawAllGrease` |
| `GliderPRO/Sources/Objects.c` | — | `SetObjectState` (the whole 335-line switch), `GetObjectState`, `ListAllLocalObjects`, `ObjectIsLinkTransport/Switch`, `BringSendFrontBack` |
| `GliderPRO/Sources/ObjectRects.c` | 300+ | `GetObjectRect`, `AddActiveRect`, `CreateActiveRects` — every hot-spot rectangle in the game |
| `GliderPRO/Sources/StructuresInit2.c` | 306-475 | the complete `srcRects[0x90]` sprite-geometry table |
| `GliderPRO/Sources/RoomGraphics.c` | 463 | `DrawLocale`, `DrawRoomBackground`, `DrawFloorSupport`, `DrawLighting`, `RedrawRoomLighting` |
| `GliderPRO/Sources/Modes.c` | 641 | the four transit entry points, `FlagGliderShredding`, `FlagGliderBurning` |
| `GliderPRO/Sources/Room.c` | 970-1134 | `GetNumberOfLights` (both mirrored branches), `IsShadowVisible`, `GetNeighborRoomNumber`, `GetRoomNumber` |
| `GliderPRO/Sources/ObjectDraw2.c` | 1436 | `DrawLightSwitch`/`DrawMachineSwitch`/`DrawThermostat`/`DrawPowerSwitch`/`DrawKnifeSwitch`/`DrawInvisibleSwitch`/`DrawTrigger`/`DrawSoundTrigger`, `DrawSimpleLight`, `DrawFlourescent`, `DrawTrackLight`, `DrawInvisLight`, `DrawSimpleAppliance`, `DrawMacPlus`, `DrawTV`, `DrawCoffee`, `DrawOutlet`, `DrawVCR`, `DrawStereo`, `DrawMicrowave`, `DrawPictSansWhiteObject`, `DrawCustPictSansWhite` — including both depth-4 and 8-bit palettes |
| `GliderPRO/Sources/StructuresInit.c` | — | `InitSwitches`, `InitLights`, `InitAppliances` — every sprite-sheet GWorld size, PICT ID and sub-rect |
| `GliderPRO/Sources/Sound.c` | 536 | `PlayPrioritySound` (3-channel priority arbitration), `FlushAnyTriggerPlaying`, `LoadTriggerSound`, `DumpTriggerSound`, `LoadBufferSounds` |
| `GliderPRO/Sources/Render.c` | 20-131 | `AddRectToWorkRects`, `AddRectToBackRects`, `AddRectToWorkRectsWhole`, `kMaxGarbageRects` |
| `GliderPRO/Sources/Link.c` | 32-53 | `MergeFloorSuite`, `ExtractFloorSuite`, `UpdateLinkControl` |
| `GliderPRO/Sources/Play.c` | 603-703 | `SetObjectsToDefaults` |
| `GliderPRO/Sources/Player.c` | 64-146 | `MoveGlider` (why vents must re-push every frame) |
| `GliderPRO/Sources/ObjectAdd.c` | — | editor placement defaults for every object family |
| `GliderPRO/Sources/Interactions.c` | 750-1777 | `HandleRewards`, `HandleSwitches`, `HandleMicrowaveAction`, `HandleHotSpotCollision`, `CheckForHotSpots`, `FlagStillOvers`, `WebGlider` |
| `GliderPRO/Sources/Transit.c` | — | `WhatAreWeLinkedTo`, `ReadyGliderFromTransit`, `Move*RoomToRoom`, `HandleRoomVisitation` |
| `GliderPRO/Sources/Trip.c` | — | `Toggle*`/`Trigger*` helpers, `UpdateOutletsLighting` |
| `GliderPRO/Sources/DynamicMaps.c` | — | `savedMaps`, `AddSparkle`, `AddFlyingPoint`, the flame/tiki/BBQ/pendulum/star strip backups |
| `GliderPRO/Sources/ObjectDrawAll.c` | — | `DrawARoomsObjects`, flame anchors, the `dynaNum`/`hotNum` correlation loop |
| `GliderPRO/Headers/GliderDefines.h` | 640 | every constant quoted in this document |
| `GliderPRO/Headers/GliderStructs.h` | 340 | every struct and union variant |

Also consulted: `Sources/HouseLegal.c`, `Sources/ObjectInfo.c`, `Sources/ObjectEdit.c`
(`DrawThisRoomsObjects`, the edit-mode dark-room gray overlay), `Sources/RoomInfo.c`
(the "(Room Is Dark)" / "(Room Is Lit)" dialog strings), `Sources/Music.c`,
`Sources/Utilities.c`, `Sources/Coordinates.c`, `Sources/StructuresInit1.c`.

Binary data verified by parsing with python3: `GliderPRO/Houses/Empty House.binhex`,
`Demo House.binhex`, `Art Museum.binhex`, `CD Demo House.binhex`, `California or Bust!.binhex`,
`Castle o' the Air.binhex`, `Grand Prix.binhex`, `ImagineHouse PRO II.binhex`, plus a type histogram
over all 22 shipped houses (every `*.binhex` in `GliderPRO/Houses/`). Observed bytes are quoted in
[Section 3](#3-on-disk-record-layout-verified-against-real-house-bytes).

---

## 1. The frame model everything hangs off

Glider PRO runs a fixed-step loop. One *frame* is `kTicksPerFrame = 2` ticks = 2/60 s, so the
nominal rate is **30 frames per second**.

| Constant | Value | Cite |
|---|---|---|
| `kTicksPerFrame` | `2` | `GliderPRO/Headers/GliderDefines.h:533` |

Two globals sequence everything:

* `gameFrame` — monotonically increasing frame counter.
* `evenFrame` — a `Boolean` toggled once per frame. Many animations advance only on `evenFrame`,
  giving an effective 15 fps for those sprites.

The order of operations within one frame that matters for object dynamics is fixed by `PlayGame`
(`GliderPRO/Sources/Play.c:430-497`; the one-player arm is `:475-496`, the two-player arm `:449-471`
and it has the same order). Full 12-step listing with line cites in §8.8:

1. `HandleTelephone` (`Play.c:445`).
2. **`HandleDynamics` ticks the 18 dynamic-object slots first** (`Play.c:475`), before any input is
   read. Appliance handlers also self-render here (§10.3).
3. Glider input is sampled and `hDesiredVel` / `vDesiredVel` are set from the keyboard (`:481`).
4. `HandleInteraction` → `CheckForHotSpots` → `HandleHotSpotCollision` (`:482`). **This is where
   every vent, blower, switch, trigger, transport and appliance-contact effect is applied.**
5. `HandleTriggers` decrements armed trigger timers and fires the expired ones (`:484`).
6. `HandleBands` (`:485`).
7. `HandleGlider` → `MoveGliderNormal`/… → `MoveGlider`, which integrates velocity and then
   **resets the desired velocities** (`:487`).
8. `RenderFrame` (`:494`): `HandleGrease`, `RenderPendulums`, flames on `evenFrame` else stars,
   `RenderDynamics`, flying points, sparkles, gliders, shreds, bands, then the dirty rects are
   flushed to the screen (`Render.c:639-671`).
9. `HandleDynamicScoreboard` (`:495`).

Note that dynamics tick *before* input and interaction, and that triggers and bands are serviced
*before* the glider integrates — an earlier revision of this section had `MoveGlider` third and
`HandleDynamics`/`HandleTriggers` after it, which is wrong in both respects.

### 1.1 Why air flow must be re-applied every single frame

`MoveGlider` clears the desired velocities at the end of every call:

| Line | Effect |
|---|---|
| `GliderPRO/Sources/Player.c:78` | `thisGlider->hDesiredVel = 0;` |
| `GliderPRO/Sources/Player.c:92` | `thisGlider->vDesiredVel = kGravity;` |

with

| Constant | Value | Cite |
|---|---|---|
| `kGravity` | `3` | `GliderPRO/Sources/Player.c:13` |
| `kHImpulse` | `2` | `GliderPRO/Sources/Player.c:14` |
| `kVImpulse` | `2` | `GliderPRO/Sources/Player.c:15` |
| `kMaxHVel` | `16` | `GliderPRO/Sources/Player.c:16` |
| `kShredderCountdown` | `-68` | `GliderPRO/Sources/Player.c:17` |

`MoveGlider` (`GliderPRO/Sources/Player.c:64-147`) is a *desired-velocity servo*, not a force
integrator:

```
 1. if hVel > hDesiredVel: hVel -= kHImpulse; clamp so hVel does not undershoot   // :66-71
    else if hVel < hDesiredVel: hVel += kHImpulse; clamp so it does not overshoot  // :72-77
 2. hDesiredVel = 0                                                               // :78
 3. if vVel > vDesiredVel: vVel -= kVImpulse; clamp                               // :80-85
    else if vVel < vDesiredVel: vVel += kVImpulse; clamp                          // :86-91
 4. vDesiredVel = kGravity                                                        // :92
 5. clamp hVel to [-kMaxHVel, +kMaxHVel]                                          // :96-97, :113-114
 6. wasHVel = hVel   (AFTER the clamp)                                            // :99 / :116
 7. offset dest.left/right and destShadow.left/right by hVel; set whole.left/right
    from the pre- and post-move edges                                             // :101-109, :118-126
 8. wasVVel = vVel   (no vertical clamp exists)                                   // :131 / :140
 9. offset dest.top/bottom by vVel; set whole.top/bottom from the pre- and post-move edges
```

So `whole` ends up as the union of the pre- and post-move `dest` (the swept dirty rect), and
`wasHVel`/`wasVVel` are snapshots of the *clamped, post-servo* velocities — not of the values on
entry. `kMaxHVel` bounds only the horizontal axis.

**Consequence for the port**: a blower does not impart an impulse that persists. It writes
`vDesiredVel = -distance-ish value` (see Section 6) *this frame only*, and `MoveGlider` moves the
glider 2 px/frame closer to that target. Next frame the target is gravity again unless the hot spot
is still overlapped. This is why a glider hovering in a vent column oscillates rather than
accelerating away, and why leaving a column makes the glider fall immediately with no residual lift.
Any Go port that models blowers as accelerations applied to a persistent velocity will get visibly
different flight behaviour.

---

## 2. The three runtime tables

An object exists in up to four places at once. Getting these four representations and their
index spaces right is the single largest porting hazard in this subsystem.

### 2.1 `thisHouse` — the authored data

`(*thisHouse)->rooms[roomNum].objects[objNum]` is the persistent authored record, `objectType`,
12 bytes (Section 3). `objNum` ranges 0..23 (`kMaxRoomObs`).

| Constant | Value | Cite |
|---|---|---|
| `kMaxRoomObs` | `24` | `GliderPRO/Headers/GliderDefines.h:250` |

### 2.2 `masterObjects[]` — the nine-room neighbourhood

When the player enters a room, `ListAllLocalObjects` (`GliderPRO/Sources/Objects.c`) flattens the
**central room plus its 8 neighbours** into one array:

The nine neighbourhood slot indices are **not** in reading order — they walk clockwise starting
North, with the centre first. This is the authoritative table, transcribed verbatim from the header:

| Constant | Value | Direction | Cite |
|---|---|---|---|
| `kCentralRoom` | `0` | the room the glider is in | `GliderPRO/Headers/GliderDefines.h:217` |
| `kNorthRoom` | `1` | one floor **up**, same suite | `:218` |
| `kNorthEastRoom` | `2` | up + right | `:219` |
| `kEastRoom` | `3` | same floor, suite + 1 | `:220` |
| `kSouthEastRoom` | `4` | down + right | `:221` |
| `kSouthRoom` | `5` | one floor **down**, same suite | `:222` |
| `kSouthWestRoom` | `6` | down + left | `:223` |
| `kWestRoom` | `7` | same floor, suite − 1 | `:224` |
| `kNorthWestRoom` | `8` | up + left | `:225` |
| `kMaxMasterObjects` | `216` (= 24 × 9) | — | `GliderPRO/Headers/GliderDefines.h:266` |

> **Correction.** An earlier revision of this section published a shuffled table
> (`kSouthRoom 2`, `kEastRoom 3`, `kWestRoom 4`, `kNorthEastRoom 5`, `kNorthWestRoom 6`,
> `kSouthEastRoom 7`) and omitted `kSouthWestRoom` entirely. That was wrong. Any Go port that
> copied it would mis-place six of the eight neighbour rooms on screen (only `kCentralRoom 0`,
> `kNorthRoom 1` and `kEastRoom 3` happened to coincide), mis-resolve `localLink`, and mis-order the
> `DrawLocale` painting sequence. The table above is correct: the nine `#define`s occupy nine
> consecutive lines of `GliderPRO/Headers/GliderDefines.h`, `:217`-`:225`, in exactly the
> centre-then-clockwise-from-North order tabulated above, with no second definition of any name —
> there is no `kNorthWestRoom` duplicate. All line numbers in this section are in the CR-to-LF
> converted copies (`tr '\r' '\n'`); the on-disk originals are CR-only single-line-looking files.

Three independent places in the source confirm the ordering, and they agree:

1. **`GetNeighborRoomNumber`** (`GliderPRO/Sources/Room.c:562-635`) maps slot → `(hDelta, vDelta)`
   applied to `(suite, floor)`. Note `floor` **increases upward** (`+1` is North):

   | slot | constant | `hDelta` (suite) | `vDelta` (floor) | cite |
   |---|---|---|---|---|
   | 0 | `kCentralRoom` | `0` | `0` | `Room.c:571-574` |
   | 1 | `kNorthRoom` | `0` | `+1` | `:576-579` |
   | 2 | `kNorthEastRoom` | `+1` | `+1` | `:581-584` |
   | 3 | `kEastRoom` | `+1` | `0` | `:586-589` |
   | 4 | `kSouthEastRoom` | `+1` | `−1` | `:591-594` |
   | 5 | `kSouthRoom` | `0` | `−1` | `:596-599` |
   | 6 | `kSouthWestRoom` | `−1` | `−1` | `:601-604` |
   | 7 | `kWestRoom` | `−1` | `0` | `:606-609` |
   | 8 | `kNorthWestRoom` | `−1` | `+1` | `:611-614` |

   The lookup is a linear scan of all `numberRooms` rooms for a `(suite, floor)` match, returning
   `kRoomIsEmpty` (−1) when there is none (`Room.c:617`, `:623-631`;
   `#define kRoomIsEmpty -1` at `GliderPRO/Headers/GliderDefines.h:525`).

2. **`localRoomsDest[]`** (`GliderPRO/Sources/InterfaceInit.c:206-218`) offsets the nine screen
   rects by exactly the same deltas, with `kRoomWide = 512` horizontally
   (`GliderPRO/Headers/GliderDefines.h:499`) and `kVertLocalOffset = 322` vertically
   (`:501`) — negative = up, because screen `v` grows downward:

   | slot | `h` offset | `v` offset | cite |
   |---|---|---|---|
   | `kNorthRoom` | `0` | `−322` | `InterfaceInit.c:211` |
   | `kNorthEastRoom` | `+512` | `−322` | `:212` |
   | `kEastRoom` | `+512` | `0` | `:213` |
   | `kSouthEastRoom` | `+512` | `+322` | `:214` |
   | `kSouthRoom` | `0` | `+322` | `:215` |
   | `kSouthWestRoom` | `−512` | `+322` | `:216` |
   | `kWestRoom` | `−512` | `0` | `:217` |
   | `kNorthWestRoom` | `−512` | `−322` | `:218` |

3. **`DrawLocale`** (`GliderPRO/Sources/RoomGraphics.c:78-121`) paints in the order NW, NE, N, SW,
   SE, S, W, E, Central — see §9.3. It uses the symbolic names, so it is order-agnostic, but it is a
   fourth textual occurrence of all nine constants including `kSouthWestRoom` (`:92-94`).

`docs/analysis/editor.md:4900-4920` already publishes this same correct table; the two documents now
agree.

Each entry is an `objDataType` (`GliderPRO/Headers/GliderStructs.h:322-332`), **26 bytes** in memory
(seven `short`s = 14 bytes, then a 12-byte `objectType`; every member is naturally 2-byte aligned so
there is no padding on either 68k or PPC):

| Field | Type | Size | Offset | Meaning |
|---|---|---|---|---|
| `roomNum` | `short` | 2 | 0 | house room number this object lives in |
| `objectNum` | `short` | 2 | 2 | index 0..23 within that room |
| `roomLink` | `short` | 2 | 4 | resolved destination room number (transport/switch target), −1 if unlinked |
| `objectLink` | `short` | 2 | 6 | resolved destination object index 0..23, −1 if unlinked |
| `localLink` | `short` | 2 | 8 | index into `masterObjects[]` of the target, **if the target is also in the current 9-room neighbourhood**, else −1 |
| `hotNum` | `short` | 2 | 10 | index into `hotSpots[]`, or −1 |
| `dynaNum` | `short` | 2 | 12 | index into `dinahs[]`, or −1 — but see §2.5 |
| `theObject` | `objectType` | 12 | 14 | a **copy** of the authored record |

`objectLink` is a **`short`, not a `Byte`** (`GliderPRO/Headers/GliderStructs.h:327`). This matters:
`GetObjectLinked` returns `-1` for "unlinked" (`GliderPRO/Sources/Objects.c:192`, `:206`, `:210`), and
that −1 has to survive round-tripping. A Go port that declares it `uint8` would store `255` and then
fail the `objectLink != -1` test in the `localLink` correlation loop
(`GliderPRO/Sources/Objects.c:335`), silently linking every unlinked object to room-object 255.

`localLink` is computed by an O(n²) loop in `ListAllLocalObjects` (`GliderPRO/Sources/Objects.c:332-346`)
that, for every object with `roomLink != -1 && objectLink != -1`, scans all master objects for one
whose `roomNum`/`objectNum` match. Note the loop does **not** `break` on a match, so if two master
entries described the same room-object the *last* would win; in practice `ListOneRoomsObjects` visits
each room at most once, so matches are unique. This is what lets a switch in room A reach the *live*
state of a light in adjacent room B.

`ListOneRoomsObjects` (`GliderPRO/Sources/Objects.c:254-296`) fills all **24** slots per room
unconditionally — empty room-object slots become master entries with `theObject.what == -1`. So
`numMasterObjects` is `24 × (number of non-empty rooms in the neighbourhood)`, not the number of real
objects. `ListAllLocalObjects` (`:300-347`) enrols the rooms in this order, which is *not* the
`DrawLocale` order:

```
1. ListOneRoomsObjects(kCentralRoom)                       // always
2. if numNeighbors > 1:  kEastRoom, kWestRoom
3. if numNeighbors > 3:  kNorthRoom, kNorthEastRoom, kSouthEastRoom,
                         kSouthRoom, kSouthWestRoom, kNorthWestRoom
```

Because the central room is enrolled first, **central-room object `i` always lands at
`masterObjects[i]`** for `i` in 0..23. Two places in the renderer rely on that coincidence
(§2.5): `CreateActiveRects` is called with the *master* index while `GetObjectRect` is handed the
*room-object* record (`GliderPRO/Sources/ObjectRects.c:304`), and `DrawARoomsObjects`' correlation
loop matches on `(objectNum, roomNum)` rather than trusting the index
(`GliderPRO/Sources/ObjectDrawAll.c:953-961`).

Hot spots are created **only** for the central room: `masterObjects[n].hotNum = CreateActiveRects(n)`
is guarded by `(where == kCentralRoom) && IsThisValid(roomNum, n)`, otherwise `hotNum = -1`
(`GliderPRO/Sources/Objects.c:283-286`). `dynaNum` is initialised to −1 for every entry (`:287`) and
filled in later by `DrawARoomsObjects`.

### 2.3 `hotSpots[]` — interaction rectangles

| Constant | Value | Cite |
|---|---|---|
| `kMaxHotSpots` | `56` | `GliderPRO/Headers/GliderDefines.h:259` |

`hotObject` (`GliderPRO/Headers/GliderStructs.h:218-225`):

| Field | Type | Meaning |
|---|---|---|
| `bounds` | `Rect` | screen-space rect (already offset into the local 9-room canvas) |
| `action` | `short` | one of the 28 `kIgnoreIt`..`kSoundIt` codes |
| `who` | `short` | index into `masterObjects[]` of the owning object |
| `isOn` | `Boolean` | live enable flag; a hot spot with `isOn == false` is skipped |
| `stillOver` | `Boolean` | debounce latch; prevents a switch/trigger re-firing while the glider stays on it. **Set** by `HandleSwitches` (`GliderPRO/Sources/Interactions.c:1154`), `ArmTrigger` (`GliderPRO/Sources/Triggers.c:54`), `kStrumIt` (`Interactions.c:1347`) and `kSoundIt` (`:1619`). **Cleared** every frame by `CheckForHotSpots` for any hot spot the glider is *not* intersecting (`Interactions.c:1683` in the one-player arm, `:1675` in the two-player arm). Re-evaluated for **every** hot spot by `FlagStillOvers` (`Interactions.c:1715-1732`: `true` if the glider intersects it, `false` otherwise, `false` for every `isOn == false` spot), which has exactly one caller — `FinishGliderDuctingIn` (`GliderPRO/Sources/Player.c:927-956`, the call at `:954`; there is no function named `MoveGliderDuctingDown` — the duct movers are `MoveGliderDownDuct` `:657` and `MoveGliderUpDuct` `:754`) — so that a glider emerging from a duct does not instantly trip whatever it lands on |
| `doScrutinize` | `Boolean` | true ⇒ do a precise sub-rect test (`SectGlider` with the mask), false ⇒ plain rect intersection |

Built by `AddActiveRect` (`GliderPRO/Sources/ObjectRects.c:277`) from `CreateActiveRects`
(`:296`). `AddActiveRect` returns −1 when `nHotSpots >= kMaxHotSpots`; **`CreateActiveRects` returns
only the LAST hot-spot index it created**, so for a multi-rect object (flames, fans, microwave) the
`masterObjects[].hotNum` records the *second* rect, not the first. This matters because
`SetObjectState` flips `hotSpots[masterObjects[local].hotNum].isOn` — see Section 8.4.

### 2.4 `dinahs[]` — the animated dynamic objects

| Constant | Value | Cite |
|---|---|---|
| `kMaxDynamicObs` | `18` | `GliderPRO/Headers/GliderDefines.h:265` |

`dynaType` (`GliderPRO/Headers/GliderStructs.h:310-320`), globals `dynaPtr dinahs; short numDynamics;`
(`GliderPRO/Sources/Dynamics3.c:18-19`):

| Field | Type | Nominal meaning | Overloads |
|---|---|---|---|
| `dest` | `Rect` | current sprite destination | — |
| `whole` | `Rect` | dirty-rect union | — |
| `hVel` | `short` | horizontal velocity | toast: clip line; drip: origin `top`; fish: idle delay; **outlet: `numLights`** |
| `vVel` | `short` | vertical velocity | — |
| `type` | `short` | object code (`kToaster`, `kOutlet`, …) | — |
| `count` | `short` | reload/period value | toast/ball/fish: launch velocity magnitude |
| `frame` | `short` | animation frame | toast idle: countdown |
| `timer` | `short` | countdown | — |
| `position` | `short` | resting/limit coordinate | outlet: 0=idle, 1=zapping |
| `room` | `short` | house room number | **never set for darts** |
| `byte0` | `Byte` | room object index | — |
| `byte1` | `Byte` | *never read or written anywhere* | — |
| `moving` | `Boolean` | in flight | — |
| `active` | `Boolean` | the object's live on/off state | — |

`ZeroDinahs` (`GliderPRO/Sources/Dynamics3.c:160-180`) clears `type, dest, whole, hVel, vVel, count,
frame, timer, position, room, byte0, active` for all 18 slots and sets `numDynamics = 0`. It
**does not clear `byte1` or `moving`** — a latent bug; `moving` happens to be re-initialised by every
`AddDynamicObject` case that uses it, so it is harmless in practice, but a Go port should zero the
whole struct.

### 2.5 The `dynaNum` / `hotNum` overloading (critical)

Two independent things are stored in `masterObjects[].dynaNum`:

* For an object that owns a dynamic slot (appliance, enemy, sparkle), `dynaNum` is its index into
  `dinahs[]`. Written by the correlation loop at the end of `DrawARoomsObjects`
  (`GliderPRO/Sources/ObjectDrawAll.c:953-961`).
* For the **six switch types** (`kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`,
  `kKnifeSwitch`, `kInvisSwitch`) `DrawARoomsObjects` instead assigns
  `dynamicNum = masterObjects[i].hotNum` at `GliderPRO/Sources/ObjectDrawAll.c:518`, `:531`, `:544`,
  `:557`, `:570`, `:574`. So a switch's `dynaNum` holds its own **hot-spot** index. `TriggerSwitch`
  (`GliderPRO/Sources/Trip.c:146-149`) consumes this — `TriggerSwitch(short who)` is literally
  `HandleSwitches(&hotSpots[who])` — so `FireTrigger`'s call
  `TriggerSwitch(masterObjects[triggeredIs].dynaNum)` (`GliderPRO/Sources/Triggers.c:128`) is
  *deliberate*, not a `dynaNum`/`hotNum` typo. See §8.6.

Additionally, `DrawARoomsObjects` reads `masterObjects[i].hotNum` using `i` = the *room* object
number rather than the master index (`GliderPRO/Sources/ObjectDrawAll.c:518`). This is correct only
for the central room, whose objects occupy `masterObjects[0..23]` in order because
`ListAllLocalObjects` enrols the central room first (`ListOneRoomsObjects(kCentralRoom)` at
`GliderPRO/Sources/Objects.c:312`). For the
eight neighbour rooms it reads the *central* room's slot `i`.

An earlier revision of this document claimed the mis-index "is not observable in 1.0.4" because
"neighbour rooms are drawn with `doDraw == false` for the switch cases". **That justification is
wrong**, twice over:

* The parameter is `redraw`, not `doDraw`, and `DrawLocale` passes `redraw == false` for **all nine**
  rooms (`GliderPRO/Sources/RoomGraphics.c:82`, `:86`, `:90`, `:94`, `:98`, `:102`, `:109`, `:114`,
  `:120`). `redraw == false` is precisely the condition that *enables* the correlation loop
  (`ObjectDrawAll.c:953`: `if (!redraw)`), so the bogus value is **committed** to
  `masterObjects[n].dynaNum` for every neighbour-room switch.
* The switch cases run for neighbour rooms too — the geometry/state block is unconditional; only the
  actual `CopyBits` is skipped when the room is unlit.

What *does* make it unobservable is the shipped data, not the code. A census of all 22 shipped houses
(§8.9) finds 171 trigger→switch links (`kTrigger`→`kInvisSwitch` 99, `kLgTrigger`→`kInvisSwitch` 72)
and **every one of them is same-room**, i.e. the target switch is in the central room, where the index
is accidentally correct. The single cross-room trigger link in the entire shipped corpus (Land of
Illusion room 1 "Steamy Star" object 16 → room 174 "Edge Of Light" object 15) points at an *empty*
slot outside the 3×3 neighbourhood, so it never reaches this code either. A Go port that fixes the
index changes nothing about the shipped houses but does change third-party houses that wire a trigger
to a neighbour-room switch.

---

## 3. On-disk record layout, verified against real house bytes

### 3.1 Container

Shipped houses are BinHex 4.0 files. Verified with a from-scratch decoder (Python's `binhex` module
was removed in 3.11, so one was written for this analysis):

```
$ python3 binhex.py "GliderPRO/Houses/Empty House.binhex"
name    : Empty House
type    : b'gliH' creator: b'ozm5'
datafork: 13046 rsrcfork: 2670
first 32 data bytes: 02 00 00 00 2c 31 30 ec 00 00 00 00 00 40 00 53 8d 54 68 69 73 20 69 73 20 61 6e 20 65 6d 70 74
```

Decoding those 32 bytes against `houseType` (`GliderPRO/Headers/GliderStructs.h:182-198`):

| Bytes | Field | Value |
|---|---|---|
| `02 00` | `short version` | `0x0200` — equals `kHouseVersion` (`GliderDefines.h:517`) |
| `00 00` | `short unusedShort` | 0 |
| `2c 31 30 ec` | `long timeStamp` | `0x2C3130EC` |
| `00 00 00 00` | `long flags` | 0 — bit 0 is the `wardBit` |
| `00 40 00 53` | `Point initial` | `{v = 0x0040 = 64, h = 0x0053 = 83}` |
| `8d 54 68 …` | `Str255 banner` | Pascal length `0x8D` = 141, then `"This is an empt…"` |

This confirms three things a Go port must honour: **big-endian** scalars, `Point` is `{v, h}` (v
first), and strings are Pascal (length-prefixed, fixed-size storage).

### 3.2 The 866 / 348 byte strides — proved arithmetically

| House | version | nRooms | firstRoom | initial (h,v) | flags | data fork | `866 + 348*nRooms` |
|---|---|---|---|---|---|---|---|
| Empty House | 0x0200 | 35 | 0 | (83, 64) | 0x00000000 | 13046 | **13046** |
| Demo House | 0x0200 | 45 | 0 | (49, 107) | 0x00000000 | 16526 | **16526** |

Exact for both. So `sizeof(houseType_header) == 866` and `sizeof(roomType) == 348`, with no
padding, and `rooms[]` starts at data-fork offset 866.

`houseType` header (`GliderPRO/Headers/GliderStructs.h:182-198`):

| Offset | Field | Type | Size |
|---|---|---|---|
| 0 | `version` | `short` | 2 |
| 2 | `unusedShort` | `short` | 2 |
| 4 | `timeStamp` | `long` | 4 |
| 8 | `flags` | `long` | 4 (bit 0 = wardBit) |
| 12 | `initial` | `Point` | 4 |
| 16 | `banner` | `Str255` | 256 |
| 272 | `trailer` | `Str255` | 256 |
| 528 | `highScores` | `scoresType` | 292 |
| 820 | `savedGame` | `gameType` | 40 |
| 860 | `hasGame` | `Boolean` | 1 |
| 861 | `unusedBoolean` | `Boolean` | 1 |
| 862 | `firstRoom` | `short` | 2 |
| 864 | `nRooms` | `short` | 2 |
| 866 | `rooms[]` | `roomType[]` | 348 each |

`roomType` (`GliderPRO/Headers/GliderStructs.h:166-180`):

| Offset | Field | Type | Size |
|---|---|---|---|
| 0 | `name` | `Str27` | 28 |
| 28 | `bounds` | `short` | 2 |
| 30 | `leftStart` | `Byte` | 1 |
| 31 | `rightStart` | `Byte` | 1 |
| 32 | `unusedByte` | `Byte` | 1 |
| 33 | `visited` | `Boolean` | 1 |
| 34 | `background` | `short` | 2 |
| 36 | `tiles[8]` | `short[8]` | 16 |
| 52 | `floor` | `short` | 2 |
| 54 | `suite` | `short` | 2 |
| 56 | `openings` | `short` | 2 |
| 58 | `numObjects` | `short` | 2 |
| 60 | `objects[24]` | `objectType[24]` | 288 |

60 + 288 = 348. ✔

### 3.3 `objectType` and its 10-byte union — every variant, every offset

`objectType` = `short what` at offset 0, then a 10-byte union at offset 2. Total **12 bytes**
(`GliderPRO/Headers/GliderStructs.h:90-105`).

| Variant | Field | Offset | Size | Cite |
|---|---|---|---|---|
| **a** `blowerType` | `topLeft` (`Point{v,h}`) | 2 | 4 | `GliderStructs.h:11-19` |
| | `distance` (`short`) | 6 | 2 | |
| | `initial` (`Boolean`) | 8 | 1 | |
| | `state` (`Boolean`) | 9 | 1 | |
| | `vector` (`Byte`) | 10 | 1 | `// \|x\|x\|x\|x\|8=lf\|4=dn\|2=rt\|1=up\|` |
| | `tall` (`Byte`) | 11 | 1 | |
| **b** `furnitureType` | `bounds` (`Rect` t,l,b,r) | 2 | 8 | `GliderStructs.h:21-25` |
| | `pict` (`short`) | 10 | 2 | |
| **c** `bonusType` | `topLeft` | 2 | 4 | `GliderStructs.h:27-34` |
| | `length` (`short`) | 6 | 2 | |
| | `points` (`short`) | 8 | 2 | |
| | `state` (`Boolean`) | 10 | 1 | |
| | `initial` (`Boolean`) | 11 | 1 | |
| **d** `transportType` | `topLeft` | 2 | 4 | `GliderStructs.h:36-43` |
| | `tall` (`short`) | 6 | 2 | |
| | `where` (`short`) | 8 | 2 | |
| | `who` (`Byte`) | 10 | 1 | |
| | `wide` (`Byte`) | 11 | 1 | |
| **e** `switchType` | `topLeft` | 2 | 4 | `GliderStructs.h:45-52` |
| | `delay` (`short`) | 6 | 2 | |
| | `where` (`short`) | 8 | 2 | |
| | `who` (`Byte`) | 10 | 1 | |
| | `type` (`Byte`) | 11 | 1 | |
| **f** `lightType` | `topLeft` | 2 | 4 | `GliderStructs.h:54-62` |
| | `length` (`short`) | 6 | 2 | |
| | `byte0` (`Byte`) | 8 | 1 | |
| | `byte1` (`Byte`) | 9 | 1 | |
| | `initial` (`Boolean`) | 10 | 1 | |
| | `state` (`Boolean`) | 11 | 1 | |
| **g** `applianceType` | `topLeft` | 2 | 4 | `GliderStructs.h:64-72` |
| | `height` (`short`) | 6 | 2 | |
| | `byte0` (`Byte`) | 8 | 1 | |
| | `delay` (`Byte`) | 9 | 1 | |
| | `initial` (`Boolean`) | 10 | 1 | |
| | `state` (`Boolean`) | 11 | 1 | |
| **h** `enemyType` | `topLeft` | 2 | 4 | `GliderStructs.h:74-82` |
| | `length` (`short`) | 6 | 2 | |
| | `delay` (`Byte`) | 8 | 1 | |
| | `byte0` (`Byte`) | 9 | 1 | |
| | `initial` (`Boolean`) | 10 | 1 | |
| | `state` (`Boolean`) | 11 | 1 | |
| **i** `clutterType` | `bounds` (`Rect`) | 2 | 8 | `GliderStructs.h:84-88` |
| | `pict` (`short`) | 10 | 2 | |

Note the **`f`/`g` vs `h` asymmetry**: in `lightType` and `applianceType` the byte at union offset 8
is `byte0` and offset 9 is `delay`; in `enemyType` offset 8 is `delay` and offset 9 is `byte0`. Aliasing
one for the other silently swaps two fields. `AddDynamicObject`'s kFish case does exactly this (see
`docs/analysis/enemies.md` §12; this document has no §12).

### 3.4 Observed raw union payloads, decoded field by field

Every row below is a real 10-byte payload read out of a shipped house and decoded against the table
above. The hex is the union bytes at record offsets 2..11.

| Where | raw union bytes | decode |
|---|---|---|
| Art Museum r0 obj[4] `kFloorTrans` | `01 2e 00 e9 00 00 18 a6 04 00` | topLeft (h=233, v=302); tall=0; where=0x18A6=6310; who=4; wide=0 → floor 2, suite 63 |
| Art Museum r8 obj[2] `kCeilingTrans` | `00 06 00 52 00 00 ff ff ff 00` | topLeft (82, 6); **where=−1, who=255 ⇒ unlinked**, no hot spot created |
| Art Museum r18 obj[4] `kLiftArea` | `00 00 00 25 00 1a 01 01 01 12` | topLeft (37, 0); distance=26; initial=1; state=1; vector=0x01 (up); tall=0x12=18 → box 26 × 36 |
| Art Museum r19 obj[1] `kLiftArea` | `00 00 01 c3 00 17 01 01 01 20` | distance=23; tall=0x20=32 → box 23 × 64 |
| Art Museum r19 obj[6] `kDeluxeTrans` | `00 2b 00 5c 51 38 18 a4 0d 11` | topLeft (92, 43); tall=0x5138 ⇒ wide-nibble 0x51=81 → 324 px, tall-byte 0x38=56 → 224 px; where=6308 → floor 0 suite 63; who=13; wide=0x11 ⇒ initial nibble 1, live nibble 1 (on) |
| Art Museum r20 obj[16] `kInvisSwitch` | `00 4e 01 76 00 00 19 07 0e 02` | delay=0; where=0x1907=6407 → floor −1 suite 64; who=14; **type=2 = kForceOff** |
| Art Museum r20 obj[17] `kInvisSwitch` | `00 4e 01 76 00 00 19 07 0f 02` | same rect, who=15, type=kForceOff — two switches stacked on one spot |
| Art Museum r22 obj[12] `kToaster` | `00 b0 00 90 00 a5 00 0b 01 01` | height=0x00A5=165; byte0=0; delay=11; initial=1; state=1 |
| Art Museum r22 obj[13] `kToaster` | `00 b5 00 fe 00 9e 00 10 01 01` | height=158; delay=16 |
| Art Museum r26 obj[19] `kDeluxeTrans` | `00 05 00 0a 13 11 15 8c 03 00` | 0x1311 → 19×4=76 wide, 17×4=68 tall; where=5516 → floor 8 suite 55; who=3; **wide=0x00 ⇒ initially OFF** |
| Art Museum r26 obj[20] `kLgTrigger` | `00 0c 00 12 00 28 18 40 15 03` | delay=0x28=40; where=6208 → floor 0 suite 62; who=21; **type=3 = kOneShot** |
| Art Museum r36 obj[4] `kShredder` | `00 a0 00 50 00 00 00 00 01 01` | initial=1, state=1 |
| Art Museum r36 obj[5] `kShredder` | `00 a0 00 99 00 00 00 00 00 00` | initial=0, state=0 (room is literally named "Only One is Off!") |
| Art Museum r47 obj[7] `kMicrowave` | `00 e7 00 e8 00 00 07 00 01 01` | **byte0=7** = all three kill bits |
| Art Museum r53 obj[7] `kLgTrigger` | `01 07 01 4a 00 0c 18 45 0b 03` | delay=12; where=6213 → floor 5 suite 62; who=11; kOneShot |
| CD Demo r5 obj[9] `kRightFan` | `00 67 00 5d 00 b6 01 01 08 55` | distance=182; state=1; **vector=0x08 (LEFT) on a right-fan**; tall=0x55=85 junk |
| CD Demo r30 obj[13] `kTrigger` | `01 22 01 36 00 00 05 e5 0a 03` | delay=0; where=1509 → floor 1 suite 15; who=10; kOneShot |
| CD Demo r30 obj[14] `kTrigger` | `01 22 01 36 00 32 05 e5 0a 03` | delay=0x32=50; identical target — two triggers on one target, 0 and 50 delay |
| CD Demo r32 obj[21] `kInvisTrans` | `00 7a 00 e3 00 43 05 82 08 02` | topLeft (227, 122); tall=0x43=67; where=1410 → floor 2 suite 14; who=8; wide=2 |
| CD Demo r33 obj[8] `kInvisTrans` | `00 8d 00 ec 00 20 05 82 ff 00` | tall=32; where=1410 but **who=255 ⇒ unlinked**, no hot spot |
| CD Demo r49 obj[17] `kLeftFan` | `00 87 01 34 00 5f 00 00 08 01` | distance=95; **initial=0, state=0** (starts off); vector=0x08 |
| CD Demo r50 obj[2] `kSewerBlower` | `01 24 00 38 00 f0 01 01 01 00` | topLeft.v = 0x0124 = **292 = `kSewerBlowerTop`**; distance=240 |
| CD Demo r50 obj[5] `kSewerBlower` | `01 24 01 5d 01 0d 01 01 01 00` | distance=0x010D=269 |
| CD Demo r51 obj[16] `kMicrowave` | `00 f2 01 5a 00 00 06 00 01 01` | **byte0=6** = battery + foil, bands survive |
| CD Demo r69 obj[0] `kMailboxLf` | `00 1b 01 a2 00 00 02 c5 06 00` | where=0x02C5=709 → floor 1 suite 7; who=6 |
| CD Demo r69 obj[2] `kMailboxRt` | `00 1c 00 00 00 00 0c 8a 03 00` | where=0x0C8A=3210 → floor 2 suite 32; who=3 |
| CD Demo r69 obj[6] `kMailboxLf` | `00 56 01 8c 00 00 02 c8 12 00` | where=712 → floor 4 suite 7; who=0x12=18 |
| CD Demo r76 obj[5] `kTiki` | `00 74 00 ac 00 4a 01 01 01 00` | distance=0x4A=74 |
| CD Demo r76 obj[15] `kBBQ` | `01 1e 01 5e 00 c0 01 01 01 00` | distance=0xC0=192 |
| CD Demo r201 obj[11] `kOutlet` | `00 70 00 38 00 00 00 0b 01 01` | height=0; delay=11 |
| Grand Prix r53 obj[7] `kOutlet` | `00 8c 01 0c 00 00 00 0b 01 01` | height=0; delay=11 |
| California or Bust r11 obj[12] `kStereo` | `00 8c 00 fa 00 00 00 00 01 00` | **initial=1 but state=0** — the `isPlayMusicGame` override (Section 5.2) |
| Grand Prix r53 obj[9] `kStereo` | `01 02 00 f2 00 00 00 00 01 00` | same |
| California or Bust r11 obj[4] `kVCR` | `00 e6 00 14 00 00 00 00 00 00` | initial=0, state=0 |
| Demo r0 obj[0] `kFloorVent` | — | topLeft (171, 305 = `kFloorVentTop`); distance=269; vector=0x01; **tall=1 (junk)** |
| Demo r2 obj[8] `kMachineSwitch` | — | where=6509 → floor 1 suite 65 (its own room); who=6; type=0 = kToggle. Room object 6 is the `kMacPlus`. |
| Demo r4 obj[11] `kKnifeSwitch` | — | where=6709 → floor 1 suite 67; who=10 = the `kTV`; type=kToggle |
| Demo r20 obj[4] `kSoundTrigger` | — | where=3011, who=255, type=0 — here **`where` is a `'snd '` resource ID**, not a room link |
| Demo r20 obj[2]/[3] `kInvisBonus` | — | points=300 (state=0) and points=500 (state=1) |
| Demo r3 obj[5..7] `kBalloon` | — | delay = 3, 7, 0 |
| Demo r15 obj[0] `kFish` | — | length=254; delay=10 |
| Demo r38 obj[5]/[6] `kGreaseRt` | — | state=0 initial=0 (authored pre-spilled) and state=0 initial=1 |
| Demo r14/21/22/23 `kInvisBlower` | — | distance ∈ {257, 261, 294, 230, 32}; tall ∈ {0,1,3,4,5} (junk); all vector=0x01 |
| Demo rooms `kCustomPict` | — | `height` = PICT resource ID: 10012, 10014, 10017, 10018, 10036, 10049, 10065 |

Three authoring facts fall out of this table and are load-bearing for a port:

1. **`data.a.tall` is only meaningful for `kLiftArea`.** Every other blower type carries junk there
   (observed 0, 1, 3, 4, 5, 85). Do not read it.
2. **`data.a.vector` can contradict `what`.** CD Demo r5 obj[9] is a `kRightFan` with `vector = 0x08`
   (left). `CreateActiveRects` and `DrawSimpleBlowers` branch on `what` for the two fan types, so
   `vector` is read only for `kInvisBlower` and `kLiftArea`. A port that trusts `vector` for fans
   will blow gliders the wrong way in at least one shipped house.
3. **`data.g.height` is a resource ID for `kCustomPict`, a launch height for `kToaster`, and unused
   for every other appliance.**

### 3.5 Object-type population across all 22 shipped houses

Counts from a full scan (only types relevant to this document plus the top of the tail shown; the
full whole-corpus histogram is in `docs/analysis/enemies.md` §18 — this document has no appendix):

| Type | Count | Type | Count | Type | Count |
|---|---|---|---|---|---|
| `kCustomPict` | 4782 | `kInvisBlower` | 2336 | `kInvisLight` | 1764 |
| `kFloorVent` | 1458 | `kLiftArea` | 716 | `kInvisSwitch` | 635 |
| `kSewerGrate` | 507 | `kSparkle` | 486 | `kCeilingTrans` | 465 |
| `kInvisTrans` | 385 | `kSlider` | 354 | `kFloorTrans` | 244 |
| `kTrigger` | 239 | `kKnifeSwitch` | 230 | `kGreaseRt` | 195 |
| `kSewerBlower` | 191 | `kCandle` | 180 | `kUpStairs` | 163 |
| `kDownStairs` | 163 | `kCeilingLight` | 160 | `kBands` | 150 |
| `kGreaseLf` | 143 | `kToaster` | 140 | `kStubby` | 127 |
| `kSoundTrigger` | 122 | `kGrecoVent` | 116 | `kFlourescent` | 115 |
| `kLightSwitch` | 113 | `kThermostat` | 108 | `kCuckoo` | 100 |
| `kOutlet` | 100 | `kTaper` | 90 | `kFloorBlower` | 83 |
| `kTrackLight` | 81 | `kLgTrigger` | 81 | `kTV` | 80 |
| `kMachineSwitch` | 79 | `kPowerSwitch` | 78 | `kMacPlus` | 74 |
| `kTableLamp` | 70 | `kCDs` | 63 | `kDecoLamp` | 62 |
| `kDeluxeTrans` | 61 | `kTiki` | 58 | `kMicrowave` | 56 |
| `kChimes` | 54 | `kRightFan` | 54 | `kShredder` | 50 |
| `kLeftFan` | 45 | `kMailboxLf` | 44 | `kBBQ` | 43 |
| `kCinderBlock` | 43 | `kCoffee` | 38 | `kStereo` | 36 |
| `kFlowerBox` | 35 | `kMailboxRt` | 34 | `kWindowInRt` | 32 |
| `kGuitar` | 29 | `kWindowExLf` | 29 | `kCeilingVent` | 28 |
| `kVCR` | 26 | `kHipLamp` | 26 | `kDoorInLf` | 23 |
| `kWindowInLf` | 21 | `kDoorExRt` | 21 | `kWindowExRt` | 19 |
| `kCeilingBlower` | 12 | `kDoorExLf` | 11 | `kDoorInRt` | 11 |

Practical implication: the invisible objects dominate. `kInvisBlower` (2336) and `kLiftArea` (716)
together outnumber all visible blowers. Getting the invisible-object geometry right (Section 6.7,
6.8) matters far more than getting the fan sprites right.

---

## 4. `srcRects[]` — the sprite geometry table

Every object's rectangle is derived from a shared table of source rectangles built once at startup by
`InitSrcRects` (`GliderPRO/Sources/StructuresInit2.c:306-475`), allocated at
`GliderPRO/Sources/StructuresInit2.c:271`:

```c
srcRects = (Rect *)NewPtr(sizeof(Rect) * kNumSrcRects);
```

| Constant | Value | Cite |
|---|---|---|
| `kNumSrcRects` | `0x90` (144) | `GliderPRO/Headers/GliderDefines.h:437` |

The table is indexed by object code, so `srcRects[kFloorVent]` etc. Each entry is set with
`QSetRect(&srcRects[X], 0, 0, w, h)` and then, for objects that live inside a packed sprite atlas,
`QOffsetRect(&srcRects[X], dh, dv)` to point at the sprite's location in its source GWorld. Objects
with no offset are drawn from a PICT loaded on demand, so their `srcRects` entry carries size only.

The **size** columns are what a port needs for collision and hot-spot math; the **src offset**
columns are only needed if you replicate the atlas packing.

### 4.1 Blowers / air sources

| Object | w × h | src offset (h, v) | line |
|---|---|---|---|
| `kFloorVent` | 48 × 11 | (0, 0) | `StructuresInit2.c:308` |
| `kCeilingVent` | 48 × 11 | (0, 11) | `:310` |
| `kFloorBlower` | 48 × 15 | (0, 22) | `:312` |
| `kCeilingBlower` | 48 × 15 | (0, 37) | `:314` |
| `kSewerGrate` | 48 × 17 | (0, 52) | `:316` |
| `kLeftFan` | 40 × 55 | (0, 69) | `:318` |
| `kRightFan` | 40 × 55 | (0, 124) | `:320` |
| `kTaper` | 20 × 59 | (0, 209) | `:322` |
| `kCandle` | 32 × 30 | (0, 179) | `:324` |
| `kStubby` | 20 × 36 | (0, 268) | `:326` |
| `kTiki` | 27 × 28 | (21, 268) | `:328` |
| `kBBQ` | 64 × 33 | (0, 0) | `:330` |
| `kInvisBlower` | 24 × 24 | (0, 0) | `:331` |
| `kGrecoVent` | 48 × 18 | (0, 340) | `:332` |
| `kSewerBlower` | 32 × 12 | (0, 390) | `:334` |
| `kLiftArea` | 64 × 32 | (0, 0) | `:336` |

`kLiftArea`'s 64 × 32 is a placeholder; its real rect comes from `distance` and `tall` (Section 6.8).
`kInvisBlower`'s 24 × 24 *is* used, as the centring reference for its four directional columns.

### 4.2 Furniture (obstacles — included because `kTable`, `kShelf`, `kCabinet`, `kFilingCabinet`, `kCounter`, `kDresser`, `kDeckTable`, `kTrunk`, `kInvisObstacle` and `kBooks` block air flow)

| Object | w × h | src offset | line |
|---|---|---|---|
| `kTable` | 64 × 8 (`kTableThick`) | — | `:338` |
| `kShelf` | 64 × 6 (`kShelfThick`) | — | `:339` |
| `kCabinet` | 64 × 64 | — | `:340` |
| `kFilingCabinet` | 74 × 107 | — | `:341` |
| `kWasteBasket` | 64 × 61 | (0, 43) | `:342` |
| `kMilkCrate` | 64 × 58 | (0, 104) | `:344` |
| `kCounter` | 128 × 64 | — | `:346` |
| `kDresser` | 128 × 64 | — | `:347` |
| `kDeckTable` | 64 × 8 | — | `:348` |
| `kStool` | 48 × 38 | (0, 183) | `:349` |
| `kTrunk` | 144 × 80 | — | `:351` |
| `kInvisObstacle` | 64 × 64 | — | `:352` |
| `kManhole` | 123 × 22 | — | `:353` |
| `kBooks` | 64 × 51 | — | `:354` |
| `kInvisBounce` | 64 × 64 | — | `:355` |

| Constant | Value | Cite |
|---|---|---|
| `kTableThick` | `8` | `GliderPRO/Headers/GliderDefines.h:439` |
| `kShelfThick` | `6` | `:440` |

### 4.3 Prizes (grease and bands are here; both have dynamics)

| Object | w × h | src offset | line |
|---|---|---|---|
| `kRedClock` | 28 × 17 | (0, 0) | `:357` |
| `kBlueClock` | 28 × 25 | (0, 17) | `:358` |
| `kYellowClock` | 28 × 28 | (0, 42) | `:360` |
| `kCuckoo` | 40 × 80 | (0, 148) | `:362` |
| `kPaper` | 48 × 21 | (0, 127) | `:364` |
| `kBattery` | 16 × 25 | (32, 0) | `:366` |
| `kBands` | 28 × 23 | (20, 70) | `:368` |
| `kGreaseRt` | 32 × 27 | (0, 243) | `:370` |
| `kGreaseLf` | 32 × 27 | (0, 324) | `:372` |
| `kFoil` | 55 × 15 | (0, 228) | `:374` |
| `kInvisBonus` | 24 × 24 | — | `:376` |
| `kStar` | 32 × 31 | (48, 0) | `:377` |
| `kSparkle` | 20 × 19 | (0, 70) | `:379` |
| `kHelium` | 56 × 16 | (32, 270) | `:381` |
| `kSlider` | 64 × 16 | — | `:383` |

### 4.4 Transport

| Object | w × h | src offset | line |
|---|---|---|---|
| `kUpStairs` | 160 × 267 | — | `:385` |
| `kDownStairs` | 160 × 267 | — | `:386` |
| `kMailboxLf` | 94 × 80 | — | `:387` |
| `kMailboxRt` | 94 × 80 | — | `:388` |
| `kFloorTrans` | 56 × 15 | (0, 1) | `:389` |
| `kCeilingTrans` | 56 × 15 | (0, 16) | `:391` |
| `kDoorInLf` | 144 × 322 | — | `:393` |
| `kDoorInRt` | 144 × 322 | — | `:394` |
| `kDoorExRt` | 16 × 322 | — | `:395` |
| `kDoorExLf` | 16 × 322 | — | `:396` |
| `kWindowInLf` | 20 × 170 | — | `:397` |
| `kWindowInRt` | 20 × 170 | — | `:398` |
| `kWindowExRt` | 16 × 170 | — | `:399` |
| `kWindowExLf` | 16 × 170 | — | `:400` |
| `kInvisTrans` | 64 × 32 | — | `:401` |
| `kDeluxeTrans` | 64 × 64 | — | `:402` |

`kInvisTrans` and `kDeluxeTrans` sizes are placeholders overridden from `data.d` (Sections 7.6, 7.7).

### 4.5 Switches and triggers

| Object | w × h | src offset | line |
|---|---|---|---|
| `kLightSwitch` | 15 × 24 | (0, 0) | `:404` |
| `kMachineSwitch` | 16 × 24 | (0, 48) | `:405` |
| `kThermostat` | 15 × 24 | (0, 48) | `:407` |
| `kPowerSwitch` | 8 × 8 | (0, 72) | `:409` |
| `kKnifeSwitch` | 16 × 24 | (0, 80) | `:411` |
| `kInvisSwitch` | 12 × 12 | — | `:413` |
| `kTrigger` | 12 × 12 | — | `:414` |
| `kLgTrigger` | 48 × 48 | — | `:415` |
| `kSoundTrigger` | 32 × 32 | — | `:416` |

Note `kSoundTrigger`'s `srcRects` entry is 32 × 32 but its **hot spot is 48 × 48** (Section 8.9) — the
sprite rect and the trip rect disagree deliberately.

### 4.6 Lights

| Object | w × h | src offset | line |
|---|---|---|---|
| `kCeilingLight` | 64 × 20 | (0, 0) | `:418` |
| `kLightBulb` | 16 × 28 | (0, 20) | `:420` |
| `kTableLamp` | 48 × 70 | (16, 20) | `:422` |
| `kHipLamp` | 72 × 276 | — | `:424` |
| `kDecoLamp` | 64 × 212 | — | `:425` |
| `kFlourescent` | 64 × 12 | — | `:426` |
| `kTrackLight` | 64 × 24 | — | `:427` |
| `kInvisLight` | 16 × 16 | — | `:428` |

### 4.7 Appliances

| Object | w × h | src offset | line |
|---|---|---|---|
| `kShredder` | 73 × 22 | — | `:430` |
| `kToaster` | 48 × 27 | (0, 22) | `:431` |
| `kMacPlus` | 48 × 58 | (0, 49) | `:433` |
| `kGuitar` | 64 × 172 | — | `:435` |
| `kTV` | 92 × 77 | — | `:436` |
| `kCoffee` | 43 × 64 | (0, 107) | `:437` |
| `kOutlet` | 16 × 24 | (64, 22) | `:439` |
| `kVCR` | 96 × 22 | — | `:441` |
| `kStereo` | 128 × 53 | — | `:442` |
| `kMicrowave` | 92 × 59 | — | `:443` |
| `kCinderBlock` | 40 × 62 | — | `:444` |
| `kFlowerBox` | 80 × 32 | — | `:445` |
| `kCDs` | 16 × 30 | (48, 22) | `:446` |
| `kCustomPict` | 72 × 34 | — | `:448` |

### 4.8 Enemies (sizes needed by the shared dynamic renderers)

| Object | w × h | line |
|---|---|---|
| `kBalloon` | 24 × 30 | `:450` |
| `kCopterLf` | 32 × 30 | `:451` |
| `kCopterRt` | 32 × 30 | `:452` |
| `kDartLf` | 64 × 19 | `:453` |
| `kDartRt` | 64 × 19 | `:454` |
| `kBall` | 32 × 32 | `:455` |
| `kDrip` | 16 × 12 | `:456` |
| `kFish` | 36 × 33 | `:457` |
| `kCobweb` | 54 × 45 | `:458` |

### 4.9 Clutter

| Object | w × h | src offset | line |
|---|---|---|---|
| `kOzma` | 102 × 92 | — | `:460` |
| `kMirror` | 64 × 64 | — | `:461` |
| `kMousehole` | 10 × 11 | — | `:462` |
| `kFireplace` | 180 × 142 | — | `:463` |
| `kWallWindow` | 64 × 80 | — | `:464` |
| `kBear` | 56 × 58 | — | `:465` |
| `kCalendar` | 63 × 92 | — | `:466` |
| `kVase1` | 36 × 45 | — | `:467` |
| `kVase2` | 35 × 57 | — | `:468` |
| `kBulletin` | 80 × 58 | — | `:469` |
| `kCloud` | 128 × 30 | — | `:470` |
| `kFaucet` | 56 × 18 | (0, 51) | `:471` |
| `kRug` | 144 × 18 | — | `:473` |
| `kChimes` | 28 × 74 | — | `:474` |

### 4.10 Room geometry constants used by all the rect math

| Constant | Value | Cite | Meaning |
|---|---|---|---|
| `kNumTiles` | `8` | `GliderDefines.h:496` | background tiles per room |
| `kTileWide` | `64` | `:497` | tile width |
| `kTileHigh` | `322` | `:498` | tile height = room height |
| `kRoomWide` | `512` | `:499` | 8 × 64 |
| `kFloorSupportTall` | `44` | `:500` | floor/ceiling slab between stacked rooms |
| `kVertLocalOffset` | `322` | `:501` | vertical spacing of the 9-room canvas |
| `kCeilingLimit` | `8` | `:503` | glider cannot go above this |
| `kFloorLimit` | `312` | `:504` | glider rests here |
| `kRoofLimit` | `122` | `:505` | |
| `kLeftWallLimit` | `12` | `:506` | |
| `kNoLeftWallLimit` | `-24` | `:507` | used when a door/window opens the wall |
| `kRightWallLimit` | `500` | `:508` | |
| `kNoRightWallLimit` | `536` | `:509` | |
| `kNoCeilingLimit` | `-10` | `:510` | |
| `kNoFloorLimit` | `332` | `:511` | |
| `kGliderWide` | `48` | `:548` | |
| `kGliderHigh` | `20` | `:549` | |
| `kHalfGliderWide` | `24` | `:550` | |
| `kGliderBurningHigh` | `26` | `:551` | |
| `kShadowHigh` | `9` | `:552` | |
| `kShadowTop` | `306` | `:553` | |

---

## 5. `initial` vs `state`: authored default vs live value

Every switchable object carries two flags in its union: an **authored default** (`initial`) and a
**live runtime value** (`state`). The house file stores both, but only `initial` is authoritative —
`state` as read from disk is discarded at game start.

### 5.1 `SetObjectsToDefaults` — the reset pass

`GliderPRO/Sources/Play.c:603-708`. Called when a game starts. For every room `r < nRooms`, it sets
`rooms[r].visited = false`, then for every object:

```
 1. for r = 0 to nRooms-1:
 2.     rooms[r].visited = false
 3.     for o = 0 to kMaxRoomObs-1:
 4.         switch (objects[o].what):
 5.           case kFloorVent, kCeilingVent, kFloorBlower, kCeilingBlower, kSewerGrate,
 6.                kLeftFan, kRightFan, kInvisBlower, kGrecoVent, kSewerBlower, kLiftArea:
 7.               data.a.state = data.a.initial
 8.           case kRedClock, kBlueClock, kYellowClock, kCuckoo, kPaper, kBattery, kBands,
 9.                kGreaseRt, kGreaseLf, kFoil, kInvisBonus, kStar, kSparkle, kHelium:
10.               data.c.state = data.c.initial
11.           case kDeluxeTrans:
12.               initState = (data.d.wide & 0xF0) >> 4     // high nibble = authored default
13.               data.d.wide &= 0xF0                        // clear the live nibble
14.               data.d.wide += initState                   // low nibble := high nibble
15.           case kCeilingLight, kLightBulb, kTableLamp, kHipLamp, kDecoLamp,
16.                kFlourescent, kTrackLight, kInvisLight:
17.               data.f.state = data.f.initial
18.           case kStereo:
19.               data.g.state = isPlayMusicGame            // GLOBAL preference, not initial!
20.           case kShredder, kToaster, kMacPlus, kGuitar, kTV, kCoffee, kOutlet,
21.                kVCR, kMicrowave:
22.               data.g.state = data.g.initial
23.           case kBalloon, kCopterLf, kCopterRt, kDartLf, kDartRt, kBall, kDrip, kFish:
24.               data.h.state = data.h.initial
```

Note the families that are **absent** from this switch and therefore never reset: furniture,
transport other than `kDeluxeTrans`, switches, triggers, `kCinderBlock`, `kFlowerBox`, `kCDs`,
`kCustomPict`, `kCobweb`, clutter. Those either have no state or their state is not runtime-mutable.

### 5.2 The two exceptions worth memorising

* **`kStereo`** takes its live state from the global `isPlayMusicGame` preference, not from `initial`.
  That is why shipped houses contain stereos with `initial = 1, state = 0` (observed in
  California or Bust! r11 obj[12] and Grand Prix r53 obj[9]).
* **`kDeluxeTrans`** packs both flags into the single byte `data.d.wide`: high nibble = authored
  default, low nibble = live. Observed `wide = 0x11` (default on, live on) and `wide = 0x00`
  (default off, live off).

### 5.3 `GetObjectState`

`Sources/Objects.c` provides a read accessor that returns the live flag from whichever union variant
applies (`data.a.state`, `data.c.state`, `data.d.wide & 0x0F` for `kDeluxeTrans`, `data.f.state`,
`data.g.state`, `data.h.state`) and `false` for anything stateless. The editor's info dialogs use it
to pre-check the "initially on" box.

An editor bug worth noting for round-trip fidelity: `DoTransObjectInfo` writes
`data.d.wide = wasState << 4`, which sets the authored nibble but **zeroes the live nibble** of a
`kDeluxeTrans`. Harmless in the editor (a game start re-derives it) but a Go editor that reuses the
same code path in-game would switch the transporter off.

---

## 6. Air flow: vents, blowers, fans, flames, lift areas

### 6.0 The 28 hot-spot actions

Every interactive effect in the game is one of 28 action codes
(`GliderPRO/Headers/GliderDefines.h:282-309`):

| Code | Value | Effect summary |
|---|---|---|
| `kIgnoreIt` | 0 | no-op; exists only so an object can own a toggleable hot spot |
| `kLiftIt` | 1 | `vDesiredVel = kFloorVentLift` |
| `kDropIt` | 2 | `vDesiredVel = kCeilingVentDrop` |
| `kPushItLeft` | 3 | `hDesiredVel += -kFanStrength` |
| `kPushItRight` | 4 | `hDesiredVel += kFanStrength` |
| `kDissolveIt` | 5 | glider fades out (or spends one foil) |
| `kRewardIt` | 6 | collect prize |
| `kMoveItUp` | 7 | up the stairs |
| `kMoveItDown` | 8 | down the stairs |
| `kSwitchIt` | 9 | operate a switch |
| `kShredIt` | 10 | shredder |
| `kStrumIt` | 11 | play `kChordSound` once |
| `kTriggerIt` | 12 | arm a trigger |
| `kBurnIt` | 13 | catch fire (or spend one foil) |
| `kSlideIt` | 14 | slide along a surface |
| `kTransportIt` | 15 | teleport |
| `kIgnoreLeftWall` | 16 | disable the left wall limit |
| `kIgnoreRightWall` | 17 | disable the right wall limit |
| `kMailItLeft` | 18 | enter a left-facing mailbox |
| `kMailItRight` | 19 | enter a right-facing mailbox |
| `kDuctItDown` | 20 | enter a floor duct |
| `kDuctItUp` | 21 | enter a ceiling duct |
| `kMicrowaveIt` | 22 | destroy carried items |
| `kIgnoreGround` | 23 | disable the floor limit (manhole) |
| `kBounceIt` | 24 | bounce off an invisible bumper |
| `kChimeIt` | 25 | strike a wind chime |
| `kWebIt` | 26 | stick to a cobweb |
| `kSoundIt` | 27 | play a custom `'snd '` resource |

### 6.1 The three force magnitudes — the entire air-flow model

`GliderPRO/Sources/Interactions.c:13-15`:

| Constant | Value | Applied as |
|---|---|---|
| `kFloorVentLift` | `-6` | `thisGlider->vDesiredVel = kFloorVentLift;` (`Interactions.c:1203`) |
| `kCeilingVentDrop` | `8` | `thisGlider->vDesiredVel = kCeilingVentDrop;` (`Interactions.c:1207`) |
| `kFanStrength` | `12` | `thisGlider->hDesiredVel += ±kFanStrength;` (`Interactions.c:1211`, `:1215`) |

Three things to note, all of them behaviourally significant:

1. **Vertical flow *assigns*; horizontal flow *accumulates*.** `kLiftIt` and `kDropIt` use `=`, so
   two overlapping vents give exactly the same lift as one, and a vent overlapping a ceiling blower
   gives whichever hot spot is tested *last* (hot spots are scanned in `hotSpots[]` index order, i.e.
   creation order). `kPushItLeft`/`kPushItRight` use `+=`, so two right-fans give `+24` and
   opposing fans cancel to `0`.
2. **There is no distance falloff and no strength authoring.** `data.a.distance` sets the *reach* of
   the column, never its force. A 32-px vent and a 301-px vent push identically hard; the tall one
   just reaches further. This is the single most counter-intuitive fact about Glider PRO air flow.
3. **`vDesiredVel = -6` is only 2 px/frame stronger than gravity's `+3`** in servo terms: `MoveGlider`
   walks `vVel` toward `-6` at `kVImpulse = 2` per frame, so a glider entering a vent from rest takes
   3 frames to reach full lift and rises at 6 px/frame thereafter, i.e. 180 px/s.

Effective terminal speeds, given `kVImpulse = 2`, `kHImpulse = 2`, `kMaxHVel = 16`:

| Situation | terminal velocity | frames to reach it from rest |
|---|---|---|
| free fall | `vVel = +3` | 2 |
| in a lift column | `vVel = -6` | 3 (from `+3`: +3→+1→−1→−3→−5→−6, 5 frames) |
| in a drop column | `vVel = +8` | 3 (from `+3`) |
| in one fan column | `hVel = ±12` | 6 |
| in two same-direction fans | `hVel = ±16` (clamped by `kMaxHVel`) | 8 |

### 6.2 Which objects produce air flow

Eleven object codes are in the "blower" family, i.e. they use union variant `a` and are the ones
`SetObjectsToDefaults` resets and `SetObjectState` can switch:

`kFloorVent`, `kCeilingVent`, `kFloorBlower`, `kCeilingBlower`, `kSewerGrate`, `kLeftFan`,
`kRightFan`, `kInvisBlower`, `kGrecoVent`, `kSewerBlower`, `kLiftArea`.

Five more use variant `a` and produce lift, but are **not switchable** — `SetObjectState` explicitly
sets `changed = false` for them with the comment "Cannot switch on/off these"
(`GliderPRO/Sources/Objects.c:420`): `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ`. Those five are
also the only ones whose lift column doubles as a **burn** column.

### 6.3 Simple lift columns: `kFloorVent`, `kFloorBlower`, `kSewerGrate`, `kGrecoVent`, `kSewerBlower`

All five share identical geometry code in `CreateActiveRects`
(`GliderPRO/Sources/ObjectRects.c:311`, `:333`, `:357`, `:566`, `:578`):

```
 1. QSetRect(&bounds, 0, -distance, kFloorColumnWide, 0)
 2. QOffsetRect(&bounds, HalfRectWide(&srcRects[what]) - 2, 0)
 3. QOffsetRect(&bounds, topLeft.h, topLeft.v)
 4. hotNum = AddActiveRect(&bounds, kLiftIt, who, data.a.state, false)
```

with

| Constant | Value | Cite |
|---|---|---|
| `kFloorColumnWide` | `4` | `GliderPRO/Sources/ObjectRects.c:13` |

So the column is **4 pixels wide**, centred on the sprite (`HalfRectWide - 2`), extends
`data.a.distance` pixels *up* from the sprite's top-left `v`, and its bottom edge is exactly at
`topLeft.v`. Worked example using observed bytes (Demo House room 0 object 0, `kFloorVent`,
topLeft = (h 171, v 305), distance = 269, `srcRects[kFloorVent]` = 48 × 11):

* `HalfRectWide(48) - 2 = 24 - 2 = 22`
* column left = `171 + 22 = 193`, right = `193 + 4 = 197`
* column top = `305 - 269 = 36`, bottom = `305`

i.e. a 4 × 269 strip from y=36 to y=305 at x=193..197. Since `kCeilingLimit = 8` and
`kFloorLimit = 312`, that column spans nearly the full room height.

**The column is only 4 px wide but the glider is 48 px wide.** Any overlap counts (these hot spots
have `doScrutinize = false`, so it is a plain `SectRect` against the glider's `dest`), so the
*effective* catch width is 4 + 48 = 52 px. This is why vents feel much wider than they look.

`isOn` is initialised from `data.a.state`, and a hot spot with `isOn == false` is skipped by
`CheckForHotSpots`, so switching a vent off is a pure hot-spot-enable operation.

### 6.4 Ceiling columns: `kCeilingVent`, `kCeilingBlower`

`GliderPRO/Sources/ObjectRects.c:321`, `:345`:

```
 1. QSetRect(&bounds, 0, 0, kCeilingColumnWide, distance)
 2. QOffsetRect(&bounds, HalfRectWide(&srcRects[what]) - 12, 0)
 3. QOffsetRect(&bounds, topLeft.h, topLeft.v)
 4. hotNum = AddActiveRect(&bounds, kDropIt, who, data.a.state, false)
```

| Constant | Value | Cite |
|---|---|---|
| `kCeilingColumnWide` | `24` | `GliderPRO/Sources/ObjectRects.c:14` |

Downdrafts are **24 px wide, six times wider than updrafts**, centred (`HalfRectWide - 12`), and
extend `distance` px *down* from `topLeft.v`. `kCeilingVent` and `kCeilingBlower` are both 48 px
wide sprites, so the offset is `24 - 12 = +12` and the column is centred exactly.

Editor default `distance` for ceiling types is **32**, versus **64** for floor types (Section 6.10),
so authored downdrafts are short and wide, updrafts tall and narrow.

### 6.5 Fans: `kLeftFan`, `kRightFan`

Each fan creates **two** hot spots (`GliderPRO/Sources/ObjectRects.c:369` and `:384`).

| Constant | Value | Cite |
|---|---|---|
| `kFanColumnThick` | `16` | `GliderPRO/Sources/ObjectRects.c:15` |
| `kFanColumnDown` | `20` | `GliderPRO/Sources/ObjectRects.c:16` |

`kLeftFan` (blows left):

```
 1. // hot spot 1: the spinning blade, lethal on contact
 2. QSetRect(&bounds, 0, 0, 13, 43)
 3. QOffsetRect(&bounds, 16, 12)
 4. QOffsetRect(&bounds, topLeft.h, topLeft.v)
 5. AddActiveRect(&bounds, kDissolveIt, who, true, true)   // always on, scrutinized
 6. // hot spot 2: the air column
 7. QSetRect(&bounds, 0, 0, distance, kFanColumnThick)     // distance wide, 16 tall
 8. QOffsetRect(&bounds, -distance, kFanColumnDown)        // extends LEFT of the fan
 9. QOffsetRect(&bounds, topLeft.h, topLeft.v)
10. hotNum = AddActiveRect(&bounds, kPushItLeft, who, data.a.state, false)
```

`kRightFan` is mirrored:

```
 1. QSetRect(&bounds, 0, 0, 13, 43); QOffsetRect(&bounds, 6, 12)   // blade at +6, not +16
 2. AddActiveRect(&bounds, kDissolveIt, who, true, true)
 3. QSetRect(&bounds, 0, 0, distance, kFanColumnThick)
 4. QOffsetRect(&bounds, RectWide(&srcRects[kRightFan]), kFanColumnDown)   // = +40
 5. QOffsetRect(&bounds, topLeft.h, topLeft.v)
 6. hotNum = AddActiveRect(&bounds, kPushItRight, who, data.a.state, false)
```

Facts a port must not get wrong:

* The air column is **16 px tall and `distance` px long**, i.e. the axes swap relative to vents.
* The column starts at `topLeft.v + 20` and so covers y = `topLeft.v+20 .. topLeft.v+36`, in the
  middle of the 55-px-tall fan sprite.
* **The blade `kDissolveIt` rect is created with `isOn = true` unconditionally.** Turning a fan off
  stops the wind but the blade still kills you. Observed: CD Demo r49 obj[17] is a `kLeftFan` with
  `initial = 0, state = 0` in a room named "Prepare To Ascend" — an off fan you must fly past.
* `masterObjects[].hotNum` ends up pointing at the **air column** (the second/last rect), which is
  correct: `SetObjectState` flips the wind, not the blade.
* The blade rect is 13 × 43 for both fans, but sits at h-offset **+16 for the left fan and +6 for the
  right fan** within the same 40 × 55 sprite box. That is the sprite's own asymmetry, not a bug.
* `data.a.vector` is **not consulted** for fans. `CreateActiveRects` branches on `what`. Observed
  contradictory data (CD Demo r5 obj[9]: a `kRightFan` with `vector = 0x08` = left) confirms authoring
  tools did not keep them in sync.

### 6.6 Flames: `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ`

These are the only objects with a **split column** — lift above, fire below.

| Constant | Value | Cite |
|---|---|---|
| `kDeadlyFlameHeight` | `24` | `GliderPRO/Sources/ObjectRects.c:17` |

Shared algorithm (`GliderPRO/Sources/ObjectRects.c:399` taper, `:424` candle, `:449` stubby, `:474`
tiki, `:499` BBQ):

```
 1. QSetRect(&bounds, 0, -distance, kFloorColumnWide, 0)      // BBQ: bottom is +8, not 0
 2. QOffsetRect(&bounds, <per-type h centring>, 0)
 3. QOffsetRect(&bounds, topLeft.h, topLeft.v)
 4. if ((bounds.bottom - bounds.top) > kDeadlyFlameHeight):          // ObjectRects.c:407
 5.     lift = bounds; lift.bottom = bounds.bottom - kDeadlyFlameHeight        // :409
 6.     AddActiveRect(&lift, kLiftIt, who, true, false)        // ALWAYS ON    // :410
 7.     burn = bounds; burn.top = bounds.bottom - kDeadlyFlameHeight + 2       // :412
 8.     hotNum = AddActiveRect(&burn, kBurnIt, who, true, false)               // :413
 9. else:
10.     hotNum = AddActiveRect(&bounds, kBurnIt, who, true, false)
11. // then the body: a lethal-on-contact sprite rect
12. QSetRect(&body, 0, 0, <w>, <h>); QOffsetRect(&body, <dh>, <dv>)
13. QOffsetRect(&body, topLeft.h, topLeft.v)
14. hotNum = AddActiveRect(&body, kDissolveIt, who, true, true)
```

Per-type parameters:

| Object | column h centring | column bottom | body w × h | body offset (h, v) |
|---|---|---|---|---|
| `kTaper` | `HalfRectWide(20) - 2 = 8` | `topLeft.v` | 7 × 48 | (+6, +11) |
| `kCandle` | `HalfRectWide(32) - 2 = 14`, **then −2** | `topLeft.v` | 8 × 20 | (+9, +11) |
| `kStubby` | `HalfRectWide(20) - 2 = 8`, **then −1** | `topLeft.v` | 15 × 26 | (+1, +11) |
| `kTiki` | `HalfRectWide(27) - 2 = 11` | `topLeft.v` | 15 × 14 | (+6, +6) |
| `kBBQ` | `HalfRectWide(64) - 2 = 30` | `topLeft.v + 8` | 52 × 17 | (+6, +8) |

The `−2` on the candle and `−1` on the stubby are hard-coded pixel nudges in the original, not
derivable from the sprite sizes; they must be copied literally.

Behaviour:

* The **lowest 22 px** of the column is `kBurnIt` and everything above `bounds.bottom - 24` is
  `kLiftIt`, so the two rects do **not** meet: the lift rect ends at `bounds.bottom - 24` and the burn
  rect starts at `bounds.bottom - 22`, leaving a **2-pixel dead band** (rows `bottom-24` and
  `bottom-23`) covered by neither hot spot. Lift height is `distance - 24`, burn height is always 22.
  So a flame with `distance = 32` gives **8** px of lift, 2 px of nothing and 22 px of fire; a flame
  with `distance = 167` (observed: Demo House r2 obj[3] `kCandle`) gives **143** px of lift, 2 px of
  nothing and 22 px of fire. Reproduce the gap literally — a glider whose rect falls entirely inside
  it is neither lifted nor burnt.
* If `distance <= kDeadlyFlameHeight` the whole column is fire with no lift at all. A `kTaper` with
  the observed `distance = 32` (Demo House r5 obj[4]/[5]) is 32 > 24, so it does split.
* Both hot spots are created with `isOn = true` **unconditionally**, matching `SetObjectState`'s
  refusal to switch these types.
* `kBurnIt` (`GliderPRO/Sources/Interactions.c:1356-1373`):

```
 1. if mode is not kGliderBurning and not kGliderFadingOut:
 2.     if (foilTotal > 0) or (mode == kGliderLosingFoil):
 3.         vDesiredVel = kFloorVentLift            // foil-protected: the fire LIFTS you
 4.         if foilTotal > 0:
 5.             PlayPrioritySound(kSizzleSound, kSizzlePriority)
 6.             foilTotal--
 7.             if foilTotal <= 0: StartGliderFoilLosing(thisGlider)
 8.     else:
 9.         FlagGliderBurning(thisGlider)
```

  So with aluminium foil, fire behaves as extra lift and costs one foil charge per frame with
  `kSizzleSound` each frame. Without foil, `FlagGliderBurning` (`GliderPRO/Sources/Modes.c:406`)
  plays `kCaughtFireSound` at `kCaughtFirePriority`, snaps `dest.top = dest.bottom -
  kGliderBurningHigh (26)`, selects `gliderSrc[25]` (facing left) or `gliderSrc[21]` (facing right),
  zeroes all four velocity fields, and sets `wasMode = kFramesToBurn = 60`
  (`GliderPRO/Sources/Modes.c:408`) as a burn-down counter.

| Constant | Value | Cite |
|---|---|---|
| `kFramesToBurn` | `60` | `GliderPRO/Sources/Modes.c:408` |
| `kGliderBurningHigh` | `26` | `GliderPRO/Headers/GliderDefines.h:551` |
| `kSizzleSound` / `kSizzlePriority` | `47` / `413` | `GliderDefines.h:102`, `:163` |
| `kCaughtFireSound` / `kCaughtFirePriority` | `16` / `902` | `GliderDefines.h:71`, `:174` |

* The **body** rect is `kDissolveIt` with `doScrutinize = true`, i.e. touching the candle itself (as
  opposed to its flame column) makes the glider fade out, or spends one foil, exactly like touching
  furniture.

### 6.7 `kInvisBlower` — the four-direction invisible blower

The most numerous air source in the game (2336 instances across the 22 shipped houses). It is the
only blower besides `kLiftArea` that reads `data.a.vector`
(`GliderPRO/Sources/ObjectRects.c:522`):

```
 1. switch (data.a.vector & 0x0F):
 2.   case 0x01:  // up
 3.       QSetRect(&bounds, 0, -distance - 24, kFloorColumnWide, 0)
 4.       QOffsetRect(&bounds, 12 - 2, 24)
 5.       action = kLiftIt
 6.   case 0x02:  // right
 7.       QSetRect(&bounds, 0, 0, distance + 24, kFanColumnThick)
 8.       QOffsetRect(&bounds, 0, 12 - 8)
 9.       action = kPushItRight
10.   case 0x04:  // down
11.       QSetRect(&bounds, 0, 0, kFloorColumnWide, distance + 24)
12.       QOffsetRect(&bounds, 12 - 2, 0)
13.       action = kDropIt
14.   case 0x08:  // left
15.       QSetRect(&bounds, 0, 0, distance + 24, kFanColumnThick)
16.       QOffsetRect(&bounds, -distance, 12 - 8)
17.       action = kPushItLeft
18. QOffsetRect(&bounds, topLeft.h, topLeft.v)
19. hotNum = AddActiveRect(&bounds, action, who, data.a.state, false)
```

Key differences from the visible blowers:

* The `srcRects[kInvisBlower]` box is **24 × 24**, so the centring constant is the literal `12`
  (half of 24) rather than a `HalfRectWide` call. `12 - 2 = 10` for vertical columns (4 px wide) and
  `12 - 8 = 4` for horizontal columns (16 px tall).
* Every direction adds **`+ 24`** to `distance` — the reach is `distance + 24`, not `distance`,
  because the column also spans the blower's own 24-px box.
* The up-column's rect is `(0, -distance-24) .. (4, 0)` then offset by `(10, 24)`, so relative to
  `topLeft` it spans y = `topLeft.v - distance` to `topLeft.v + 24`. The down-column spans
  `topLeft.v` to `topLeft.v + distance + 24`.
* The left-column is offset by `-distance` (not `-distance-24`), so it spans
  x = `topLeft.h - distance` .. `topLeft.h + 24`. The right-column spans
  `topLeft.h` .. `topLeft.h + distance + 24`.
* `isOn = data.a.state`, so invisible blowers *are* switchable, and 2336 of them exist — this is the
  standard mechanism for authoring switch-controlled updrafts with no visible sprite.
* Every `kInvisBlower` observed in Demo House has `vector = 0x01`. Editor default is `0x01`
  (Section 6.10). Down/left/right invisible blowers exist but are rare.

### 6.8 `kLiftArea` — a rectangular directional flow region

`kLiftArea` is the only blower whose rect is fully authored: `data.a.distance` is the **width** and
`data.a.tall` is **half the height**.

`GetObjectRect` (`GliderPRO/Sources/ObjectRects.c:63`):

```
QSetRect(itsRect, 0, 0, who->data.a.distance, who->data.a.tall * 2);
QOffsetRect(itsRect, who->data.a.topLeft.h, who->data.a.topLeft.v);
```

`CreateActiveRects` (`GliderPRO/Sources/ObjectRects.c:590`):

```
 1. QSetRect(&bounds, 0, 0, data.a.distance, data.a.tall * 2)
 2. QOffsetRect(&bounds, topLeft.h, topLeft.v)
 3. switch (data.a.vector & 0x0F):
 4.   case 0x01: action = kLiftIt
 5.   case 0x02: action = kPushItRight
 6.   case 0x04: action = kDropIt
 7.   case 0x08: action = kPushItLeft
 8. hotNum = AddActiveRect(&bounds, action, who, data.a.state, false)
```

Verified against observed bytes:

| House / room / obj | distance | tall | vector | resulting box |
|---|---|---|---|---|
| Art Museum r18 obj[4] | 26 | 0x12 = 18 | 0x01 | 26 wide × 36 tall at (h 37, v 0), `kLiftIt` |
| Art Museum r19 obj[1] | 23 | 0x20 = 32 | 0x01 | 23 wide × 64 tall at (h 451, v 0), `kLiftIt` |

The `tall * 2` is why the editor's default is `tall = 0x10` (16) giving a 32-px-tall region — see
Section 6.10. A `tall` byte is one byte, so the maximum region height is `255 * 2 = 510` px, just
under the 512-px room width and comfortably more than the 322-px room height.

`kLiftArea` is the second most common air source (716 instances). Because it produces a *rectangular*
flow region rather than a thin column, and because its `vector` **is** honoured, it is the general
tool authors used for wide updrafts and horizontal wind tunnels.

### 6.9 Air flow summary table

| Object | reach source | column geometry | action | `isOn` from | switchable |
|---|---|---|---|---|---|
| `kFloorVent` | `distance` up | 4 × `distance`, centred, bottom at `topLeft.v` | `kLiftIt` | `state` | yes |
| `kFloorBlower` | `distance` up | 4 × `distance`, centred | `kLiftIt` | `state` | yes |
| `kSewerGrate` | `distance` up | 4 × `distance`, centred | `kLiftIt` | `state` | yes |
| `kGrecoVent` | `distance` up | 4 × `distance`, centred | `kLiftIt` | `state` | yes |
| `kSewerBlower` | `distance` up | 4 × `distance`, centred | `kLiftIt` | `state` | yes |
| `kCeilingVent` | `distance` down | 24 × `distance`, centred, top at `topLeft.v` | `kDropIt` | `state` | yes |
| `kCeilingBlower` | `distance` down | 24 × `distance`, centred | `kDropIt` | `state` | yes |
| `kLeftFan` | `distance` left | `distance` × 16 at `(-distance, +20)`; plus 13 × 43 blade `kDissolveIt` always on | `kPushItLeft` | `state` (column only) | yes (column only) |
| `kRightFan` | `distance` right | `distance` × 16 at `(+40, +20)`; plus blade | `kPushItRight` | `state` (column only) | yes (column only) |
| `kTaper` | `distance` up | split: upper `distance − 24` px `kLiftIt`, 2 px gap, lower 22 px `kBurnIt`; plus 7 × 48 body | `kLiftIt`+`kBurnIt` | always `true` | **no** |
| `kCandle` | `distance` up | same, 8 × 20 body | `kLiftIt`+`kBurnIt` | always `true` | **no** |
| `kStubby` | `distance` up | same, 15 × 26 body | `kLiftIt`+`kBurnIt` | always `true` | **no** |
| `kTiki` | `distance` up | same, 15 × 14 body | `kLiftIt`+`kBurnIt` | always `true` | **no** |
| `kBBQ` | `distance` up | same but column bottom `+8`, 52 × 17 body | `kLiftIt`+`kBurnIt` | always `true` | **no** |
| `kInvisBlower` | `distance + 24` per `vector` | 4 wide vertical / 16 tall horizontal | per `vector` | `state` | yes |
| `kLiftArea` | `distance` × `tall*2` box | authored rectangle | per `vector` | `state` | yes |

### 6.10 Editor placement defaults (what authored data normally looks like)

From `GliderPRO/Sources/ObjectAdd.c`:

| Constant | Value | Cite |
|---|---|---|
| `kFloorVentTop` | `305` | `GliderPRO/Headers/GliderDefines.h:467` |
| `kCeilingVentTop` | `8` | `:468` |
| `kFloorBlowerTop` | `304` | `:469` |
| `kCeilingBlowerTop` | `5` | `:470` |
| `kSewerGrateTop` | `303` | `:471` |
| `kGrecoVentTop` | `303` | `GliderPRO/Sources/ObjectAdd.c:22` |
| `kSewerBlowerTop` | `292` | `GliderPRO/Sources/ObjectAdd.c:23` |
| `kMaxCandles` | `20` | `GliderPRO/Headers/GliderDefines.h:255` (enforced at `ObjectAdd.c:80-81`) |
| `kMaxTikis` | `8` | `GliderPRO/Headers/GliderDefines.h:256` (enforced at `ObjectAdd.c:86`) |
| `kMaxCoals` | `8` | `GliderPRO/Headers/GliderDefines.h:257` (enforced at `ObjectAdd.c:91`) |

Defaults applied when the author drops a new object:

| Family | `distance` | `initial`/`state` | `vector` | `tall` | `topLeft.v` |
|---|---|---|---|---|---|
| `kFloorVent` | 64 | true | 0x01 | 0x00 | `kFloorVentTop` = 305 |
| `kFloorBlower` | 64 | true | 0x01 | 0x00 | `kFloorBlowerTop` = 304 |
| `kSewerGrate` | 64 | true | 0x01 | 0x00 | `kSewerGrateTop` = 303 |
| `kGrecoVent` | 64 | true | 0x01 | 0x00 | 303 |
| `kSewerBlower` | 64 | true | 0x01 | 0x00 | 292 |
| `kTaper`/`kCandle`/`kStubby`/`kTiki`/`kBBQ`/`kInvisBlower` | 64 | true | 0x01 | 0x00 | centred on the click |
| `kLiftArea` | 64 | true | 0x01 | **0x10** | centred on the click |
| `kCeilingVent` | 32 | true | **0x04** | 0x00 | `kCeilingVentTop` = 8 |
| `kCeilingBlower` | 32 | true | **0x04** | 0x00 | `kCeilingBlowerTop` = 5 |
| `kLeftFan` | 32 | true | 0x08 | 0x00 | click |
| `kRightFan` | 32 | true | **0x02** | 0x00 | click |

Observed `topLeft.v` values confirm the defaults survive authoring: Demo House `kFloorVent`s all sit
at v = 305, `kGrecoVent`s at v = 303, and CD Demo `kSewerBlower`s at v = 0x0124 = 292.

Per-type placement caps are enforced by `ObjectAdd.c` because each of these families shares a
pre-composited animation strip in a `savedMaps` slot:

| Family | cap | constant |
|---|---|---|
| tapers + candles + stubbies (combined) | 20 | `kMaxCandles` |
| tikis | 8 | `kMaxTikis` |
| BBQs | 8 | `kMaxCoals` |
| shredders + toasters + macs + TVs + coffees + outlets + VCRs + microwaves + stereos (combined) | 4 | `kMaxShredded` |
| up-stairs | 1 | `kMaxStairs` |
| down-stairs | 1 | `kMaxStairs` |
| sound triggers | 1 | `kMaxSoundTriggers` |

| Constant | Value | Cite |
|---|---|---|
| `kMaxSoundTriggers` | `1` | `GliderPRO/Sources/ObjectAdd.c:17` |
| `kMaxStairs` | `1` | `GliderPRO/Sources/ObjectAdd.c:18` |
| `kMouseholeBottom` | `295` | `GliderPRO/Sources/ObjectAdd.c:19` |
| `kFireplaceBottom` | `297` | `GliderPRO/Sources/ObjectAdd.c:20` |
| `kManholeSits` | `322` | `GliderPRO/Sources/ObjectAdd.c:21` |

### 6.11 Flame animation (visual only, no gameplay effect)

Flames, tikis, coals, pendulums and stars animate by blitting from pre-composited strips saved at
room-load time by `BackUpFlames`, `BackUpTiki`, `BackUpBBQ`, `BackUpPendulum`, `BackUpStar`
(`GliderPRO/Sources/DynamicMaps.c`), each into a slot of the shared `savedMaps[]` array.

| Constant | Value | Cite |
|---|---|---|
| `kNumCandleFlames` | `5` | `GliderPRO/Headers/GliderDefines.h:447` |
| `kNumTikiFlames` | `5` | `:448` |
| `kNumBBQCoals` | `4` | `:449` |
| `kNumPendulums` | `3` | `:450` |
| `kNumTrackLights` | `3` | `:445` |
| `kNumFlowers` | `6` | `:458` |

`flameType` (`GliderPRO/Headers/GliderStructs.h:251-256`) and `pendulumType` (`:258-264`) hold the
per-instance rect and frame counter. Frames advance once per frame (not gated on `evenFrame`) and
wrap at the counts above. There is **no interaction between flame frame and the burn column** — the
`kBurnIt` rect is static regardless of which flame frame is showing.

**4-bit-depth quirk**: when `thisMac.isDepth == 4`, the flame/tiki/coal/pendulum/star draw code forces
the destination x-coordinate to be **even**, because the 4-bit pixel packing puts two pixels per byte
and the blitter cannot handle odd offsets. A Go port that always renders 32-bit RGBA should simply
ignore this, but should be aware that shipped houses may have been authored on 4-bit displays and so
show a 1-px difference from the 8-bit rendering.

---

## 7. Transport: the exact link and destination mechanism

Nine object codes are transport: `kUpStairs`, `kDownStairs`, `kMailboxLf`, `kMailboxRt`,
`kFloorTrans`, `kCeilingTrans`, `kDoorInLf`/`kDoorInRt`/`kDoorExRt`/`kDoorExLf`,
`kWindowInLf`/`kWindowInRt`/`kWindowExRt`/`kWindowExLf`, `kInvisTrans`, `kDeluxeTrans`. Of these
**only six carry a link**: `kMailboxLf`, `kMailboxRt`, `kFloorTrans`, `kCeilingTrans`, `kInvisTrans`,
`kDeluxeTrans` (`ObjectIsLinkTransport`, `GliderPRO/Sources/Objects.c:219-232`). Doors, windows and
stairs are *implicit* transport — they move the glider to a geometrically adjacent room and need no
authored destination.

### 7.1 `where` encoding: `MergeFloorSuite` / `ExtractFloorSuite`

A room is identified in authored data not by its index in `rooms[]` (which depends on authoring order)
but by a **floor/suite pair** packed into one `short`.

`GliderPRO/Sources/Link.c:32-53`:

```c
short MergeFloorSuite (short floor, short suite)
{
    return ((suite * 100) + floor);
}

void ExtractFloorSuite (short combo, short *floor, short *suite)
{
    if ((*thisHouse)->version < 0x0200)               // literal in the source, not kHouseVersion
    {
        *floor = (combo / 100) - kNumUndergroundFloors;
        *suite = combo % 100;
    }
    else
    {
        *suite = combo / 100;
        *floor = (combo % 100) - kNumUndergroundFloors;
    }
}
```

| Constant | Value | Cite |
|---|---|---|
| `kNumUndergroundFloors` | `8` | `GliderPRO/Headers/GliderDefines.h:535` |
| `kHouseVersion` | `0x0200` | `:517` |
| `kNewHouseVersion` | `0x0300` | `:518` |

**The two fields swapped position between house-file versions.** In pre-0x0200 houses
`combo = floor_biased * 100 + suite`; in 0x0200 and later `combo = suite * 100 + floor_biased`.
`MergeFloorSuite` only ever writes the *new* order, so a pre-0x0200 house that is opened and re-saved
by 1.0.4 has all its links silently rewritten into the new order. A Go port must branch on
`version < 0x0200` when reading, and always write the new order.

**`/` and `%` here are C integer division, which truncates toward zero.** This only matters for the
negative `where` values that do occur in shipped data (`-1` and `-100`): C gives `-1 / 100 == 0` and
`-1 % 100 == -1` (→ suite 0, floor −9), whereas a language with floored division (Python, and Go's
`math.Floor` idioms — Go's own `/` and `%` truncate like C, so plain Go is fine) gives suite −1,
floor 83. Neither resolves to a real room in any shipped house, so no census below is affected, but a
port that reimplements this with floored arithmetic will produce different garbage room numbers on
malformed links.

The floor is stored **biased by +8** so that basements (floor −1 … −8) fit in an unsigned-looking
range. Verified against every observed link in the shipped houses:

| observed `where` | `suite = where/100` | `floor = where%100 - 8` | resolves to |
|---|---|---|---|
| 6310 | 63 | 10 − 8 = **2** | Art Museum floor 2 suite 63 ✔ exists |
| 6407 | 64 | 7 − 8 = **−1** | floor −1 suite 64 ✔ (a basement room) |
| 5516 | 55 | 16 − 8 = **8** | floor 8 suite 55 ✔ |
| 1410 | 14 | 10 − 8 = **2** | floor 2 suite 14 ✔ |
| 709 | 7 | 9 − 8 = **1** | floor 1 suite 7 ✔ |
| 3210 | 32 | 10 − 8 = **2** | floor 2 suite 32 ✔ |
| 712 | 7 | 12 − 8 = **4** | floor 4 suite 7 ✔ |
| 6208 | 62 | 8 − 8 = **0** | floor 0 suite 62 ✔ |
| 6213 | 62 | 13 − 8 = **5** | floor 5 suite 62 ✔ |
| 6509 | 65 | 9 − 8 = **1** | floor 1 suite 65 ✔ (the switch's own room) |
| 6709 | 67 | 9 − 8 = **1** | floor 1 suite 67 ✔ (the switch's own room) |
| 1509 | 15 | 9 − 8 = **1** | floor 1 suite 15 ✔ |
| 6308 | 63 | 8 − 8 = **0** | floor 0 suite 63 ✔ |

All 13 resolve to a floor/suite pair that exists in the same house — the encoding is confirmed.

`GetRoomNumber(floor, suite)` (in `Sources/Room.c`) does the final linear search of `rooms[]` for the
matching pair and returns the array index, or `kRoomIsEmpty` (−1) if no such room exists.

| Constant | Value | Cite |
|---|---|---|
| `kRoomIsEmpty` | `-1` | `GliderPRO/Headers/GliderDefines.h:525` |
| `kObjectIsEmpty` | `-1` | `:526` |

### 7.2 `who` encoding and the "unlinked" sentinel

`data.d.who` (transport) and `data.e.who` (switch/trigger) are **`Byte`** object indices 0..23 within
the destination room, with **255 meaning unlinked**
(`GliderPRO/Sources/Objects.c:177-210`, `GetObjectLinked`):

```
if (who->data.d.who != 255)  whoLinked = (short)who->data.d.who;
else                          whoLinked = -1;
```

There are therefore **two independent unlinked sentinels**: `where == -1` and `who == 255`. Observed
data uses both, sometimes together:

* Art Museum r8 obj[2] `kCeilingTrans`: `where = -1`, `who = 255` — fully unlinked.
* CD Demo r33 obj[8] `kInvisTrans`: `where = 1410` (a valid room!) but `who = 255` — half-linked.

`CreateActiveRects` tests only `data.d.who != 255` for mailboxes, ducts and `kInvisTrans`, so a
half-linked object like the second example gets **no hot spot at all** and is inert. A Go port must
check *both* sentinels the way the original does per call site, or some shipped houses will grow
transporters that crash on a −1 room index.

### 7.3 Link resolution at room-entry time

`ListAllLocalObjects` (`GliderPRO/Sources/Objects.c`) resolves each authored link into
`masterObjects[]`:

```
 1. nMasterObjects = 0
 2. for where = 0 to 8:                          // the 9 neighbourhood slots
 3.     ListOneRoomsObjects(where)                // appends this room's objects
 4. // per object appended:
 5.     masterObjects[n].roomNum   = localNumbers[where]
 6.     masterObjects[n].objectNum = objIndex
 7.     masterObjects[n].theObject = <copy of the authored record>
 8.     masterObjects[n].roomLink   = GetRoomLinked(&thisObject)     // -1 if unlinked
 9.     masterObjects[n].objectLink = GetObjectLinked(&thisObject)   // -1 if unlinked
10.     masterObjects[n].localLink  = -1
11.     masterObjects[n].hotNum     = -1
12.     masterObjects[n].dynaNum    = -1
13. // then the O(n^2) correlation pass:
14. for i = 0 to nMasterObjects-1:
15.     if masterObjects[i].roomLink != -1:
16.         for j = 0 to nMasterObjects-1:
17.             if (masterObjects[j].roomNum   == masterObjects[i].roomLink) and
18.                (masterObjects[j].objectNum == masterObjects[i].objectLink):
19.                 masterObjects[i].localLink = j
20.                 break
```

So `localLink` is non-−1 **only when the target is one of the ≤216 objects in the current 9-room
neighbourhood**. This is the mechanism by which a switch reaches the live state of a light in an
adjacent room. When the target is farther away, `localLink == -1` and `SetObjectState` is called with
`local = -1`, which updates the authored copy in `thisHouse` but skips the `masterObjects` /
`hotSpots` / sound side effects (Section 8.4).

### 7.4 The complete transit call chain

Take a glider flying into a `kFloorTrans` (floor duct). The chain is:

```
 1. CheckForHotSpots() finds a hot spot with action == kDuctItDown whose bounds
    intersect the glider (and isOn == true).
 2. HandleHotSpotCollision(thisGlider, who, index) dispatches case kDuctItDown
    -> StartGliderDuctingDown(thisGlider, &who->bounds, who).
 3. StartGliderDuctingDown  (GliderPRO/Sources/Modes.c:213):
      a. PlayPrioritySound(kTransOutSound, kTransOutPriority)     // 59 / 906
      b. if mode == kGliderGoingFoil:  DeckGliderInFoil()
         else if mode == kGliderLosingFoil: RemoveFoilFromGlider()
      c. whoLinked  = who->who                                   // master-object index
      d. transRoom  = masterObjects[whoLinked].roomLink          // GLOBAL
      e. objLinked  = masterObjects[whoLinked].objectLink
      f. linkedToWhat = WhatAreWeLinkedTo(transRoom, objLinked)   // GLOBAL
      g. GetObjectRect(&(*thisHouse)->rooms[transRoom].objects[objLinked], &transRect)
                                                                 // GLOBAL destination rect
      h. thisGlider->frame = 0
      i. thisGlider->clip  = *bounds
         thisGlider->clip.left = bounds->left + ((RectWide(bounds) - kGliderWide) / 2)
      j. thisGlider->mode = kGliderDuctingDown
 4. Over the following frames the glider animates sinking into the duct, clipped
    to thisGlider->clip.
 5. When the animation finishes, MoveDuctToDuct(thisGlider) (Transit.c:352) runs:
      a. SetMusicalMode(kKickGameScoreMode)
      b. HandleRoomVisitation()          // award kRoomVisitScore if first visit
      c. sameRoom = (transRoom == thisRoomNumber)
      d. if !sameRoom: ForceThisRoom(transRoom)
      e. ReadyGliderFromTransit(thisGlider, linkedToWhat)   (both gliders if 2-player)
      f. if !sameRoom: ReadyLevel()      // rebuild the whole 9-room canvas
      g. RefreshScoreboard(kNormalTitleMode)
      h. if !sameRoom: WipeScreenOn(kAbove, &justRoomsRect)
      i. #ifdef COMPILEQT: RenderFrame(); if QT && hasMovie && tvInRoom && tvOn:
             GoToBeginningOfMovie(theMovie); StartMovie(theMovie)
 6. ReadyGliderFromTransit(thisGlider, toWhat)  (Transit.c:64) positions the glider
    at the destination -- see 7.5.
```

`TransportRoomToRoom` (`Transit.c:314`) and `MoveMailToMail` (`Transit.c:391`) are **byte-for-byte
identical** to `MoveDuctToDuct` apart from their names. Three copies of the same function.

Three globals carry state across the multi-frame transit: `transRoom` (`short`),
`transRect` (`Rect`), `linkedToWhat` (`short`, defined at `GliderPRO/Sources/Transit.c:16`). A Go port
should carry these in a struct on the glider or on a transit-in-progress value; note they are **shared
between both players** in a two-player game, which is why `ReadyGliderFromTransit` is called for both
gliders with the same `linkedToWhat`.

### 7.5 Arrival style: `WhatAreWeLinkedTo` and the four cases

`WhatAreWeLinkedTo(where, who)` (`GliderPRO/Sources/Transit.c:30-59`) inspects the **destination
object's** `what` and returns an arrival style:

| Destination `what` | returns | value | Cite |
|---|---|---|---|
| `kMailboxLf` | `kLinkedToLeftMailbox` | 1 | `GliderDefines.h:611` |
| `kMailboxRt` | `kLinkedToRightMailbox` | 2 | `:612` |
| `kCeilingTrans` | `kLinkedToCeilingDuct` | 3 | `:613` |
| anything else | `kLinkedToOther` | 0 | `:610` |

| Constant | Value | Cite |
|---|---|---|
| `kLinkedToOther` | `0` | `GliderPRO/Headers/GliderDefines.h:610` |
| `kLinkedToLeftMailbox` | `1` | `:611` |
| `kLinkedToRightMailbox` | `2` | `:612` |
| `kLinkedToCeilingDuct` | `3` | `:613` |
| `kLinkedToFloorDuct` | `4` | `:614` |

**`kLinkedToFloorDuct` (4) is never returned by `WhatAreWeLinkedTo`** — a `kFloorTrans` destination
falls into `default` and returns `kLinkedToOther`. Correspondingly the `case kLinkedToFloorDuct:` in
`ReadyGliderFromTransit` (`GliderPRO/Sources/Transit.c:138-139`) is an empty `break;` and is dead
code. A port can omit constant 4 entirely, but should know that a duct linked *down* into a
`kFloorTrans` therefore arrives with the *generic* teleport-in animation, not a duct animation. This
is observable in shipped houses.

`ReadyGliderFromTransit(thisGlider, toWhat)` (`GliderPRO/Sources/Transit.c:64-142`):

```
 1. if twoPlayerGame and onePlayerLeft and thisGlider->which == playerDead: return
 2. FlagGliderNormal(thisGlider)      // resets vel, size 48x20, shadow, frame 0
 3. switch (toWhat):
 4.   case kLinkedToOther:
 5.       StartGliderTransportingIn(thisGlider)     // fade-in animation
 6.       tempRect = dest; CenterRectInRect(&tempRect, &transRect)
 7.       dest = tempRect (all four edges)
 8.       destShadow.left/right = tempRect.left/right
 9.       whole = dest; wholeShadow = destShadow; enteredRect = dest
10.   case kLinkedToLeftMailbox:
11.       StartGliderMailingOut(thisGlider)
12.       clip = transRect; clip.right -= 64; clip.bottom -= 25
13.       dest.left = clip.right; dest.right = dest.left      // ZERO WIDTH to start
14.       dest.bottom = clip.bottom - 4
15.       dest.top = dest.bottom - RectTall(&tempRect)        // tempRect = old dest
16.       destShadow.left/right = dest.left/right
17.       whole = dest; wholeShadow = destShadow
18.   case kLinkedToRightMailbox:
19.       StartGliderMailingOut(thisGlider)
20.       clip = transRect; clip.left += 79; clip.bottom -= 25
21.       dest.right = clip.left; dest.left = dest.right      // ZERO WIDTH to start
22.       dest.bottom = clip.bottom - 4
23.       dest.top = dest.bottom - RectTall(&tempRect)
24.       destShadow.left/right = dest.left/right
25.       whole = dest; wholeShadow = destShadow
26.   case kLinkedToCeilingDuct:
27.       StartGliderDuctingIn(thisGlider)
28.       tempRect = dest; CenterRectInRect(&tempRect, &transRect)
29.       dest.left/right = tempRect.left/right
30.       dest.top = tempRect.top; dest.bottom = dest.top     // ZERO HEIGHT to start
31.       QOffsetRect(&dest, 0, -RectTall(&tempRect))          // start fully above the duct
32.       destShadow.left/right = tempRect.left/right
33.       whole = dest; wholeShadow = destShadow
34.   case kLinkedToFloorDuct: break                           // DEAD
35. if twoPlayerGame and thisGlider->which != firstPlayer: TagGliderIdle(thisGlider)
```

The zero-width / zero-height starting rects are the emergence animation: the glider grows out of the
mailbox slot or drops out of the ceiling duct over subsequent frames, clipped to `thisGlider->clip`.

`StartGliderMailingOut` (`GliderPRO/Sources/Modes.c:181`) picks the facing:

```
if (linkedToWhat == kLinkedToLeftMailbox):
    facing = kFaceLeft;  mode = kGliderMailOutLeft;  src = mask = gliderSrc[2]
else:
    facing = kFaceRight; mode = kGliderMailOutRight; src = mask = gliderSrc[0]
then hVel = vVel = hDesiredVel = vDesiredVel = 0; tipped = false; dontDraw = false
```

`TagGliderIdle` (`GliderPRO/Sources/Modes.c:631`) sets `wasMode = mode; mode = kGliderIdle;
hVel = 30` with the comment `// used for 30 frame delay` — `hVel` is repurposed as a countdown — one of
seven field overloads catalogued in this document (glider `hVel` as an idle countdown here;
`dinahs[].hVel` as a toaster clip line and as an outlet light-count cache, `count` as a toaster
launch velocity, `frame` as an idle-toaster countdown and as a sparkle lockout, `position` as an
outlet phase flag — §10.5; and `objDataType.dynaNum` as a switch's `hotNum` — §8.4.1).

### 7.6 Per-object transport geometry

All from `CreateActiveRects` (`GliderPRO/Sources/ObjectRects.c`). "linked" below means
`data.d.who != 255`.

| Object | condition | hot-spot rect | offset | action | `isOn` | line |
|---|---|---|---|---|---|---|
| `kUpStairs` | always | `QSetRect(0,0,112,32)` | `+topLeft` | `kMoveItUp` | true | `:734` |
| `kDownStairs` | always | `QSetRect(-80,-56,0,0)` | `+(160, 170)` then `+topLeft` | `kMoveItDown` | true | `:742` |
| `kMailboxLf` | **linked** | `QSetRect(-72,0,0,40)` | `+(30, 16)` then `+topLeft` | `kMailItLeft` | true | `:751` |
| `kMailboxRt` | **linked** | `QSetRect(0,0,72,40)` | `+(79, 16)` then `+topLeft` | `kMailItRight` | true | `:763` |
| `kFloorTrans` | **linked** | `QSetRect(0,-48,76,0)` | `+(-8, 15)` then `+topLeft` | `kDuctItDown` | true | `:775` |
| `kCeilingTrans` | **linked** | `QSetRect(0,0,76,48)` | `+(-8, 0)` then `+topLeft` | `kDuctItUp` | true | `:787` |
| `kDoorInLf` | always | `16 × 240` | `+(0, 52)` then `+topLeft` | `kIgnoreLeftWall` | true | `:799` |
| `kDoorInRt` | always | `16 × 240` | `+(128, 52)` then `+topLeft` | `kIgnoreRightWall` | true | `:808` |
| `kDoorExRt` | always | `16 × 240` | `+(0, 52)` then `+topLeft` | `kIgnoreRightWall` | true | `:817` |
| `kDoorExLf` | always | `16 × 240` | `+(0, 52)` then `+topLeft` | `kIgnoreLeftWall` | true | `:826` |
| `kWindowInLf` | always | `16 × 44` | `+(0, 96)` then `+topLeft` | `kIgnoreLeftWall` | true | `:835` |
| `kWindowInRt` | always | `16 × 44` | `+(4, 96)` then `+topLeft` | `kIgnoreRightWall` | true | `:844` |
| `kWindowExRt` | always | `16 × 44` | `+(0, 96)` then `+topLeft` | `kIgnoreRightWall` | true | `:853` |
| `kWindowExLf` | always | `16 × 44` | `+(0, 96)` then `+topLeft` | `kIgnoreLeftWall` | true | `:862` |
| `kInvisTrans` | **linked** | `QSetRect(0,0,64,32)` `+topLeft`, then `bottom = top + data.d.tall`, `right += (short)data.d.wide` | — | `kTransportIt` | **always true** | `:871` |
| `kDeluxeTrans` | **linked** | `wide=(tall&0xFF00)>>8; tall=tall&0x00FF; QSetRect(0,0,wide*4,tall*4)` `+topLeft` | — | `kTransportIt` | `data.d.wide & 0x0F` | `:884` |

Notes:

* The `kFloorTrans` offset `(-8, 15)` uses `RectTall(&srcRects[kFloorTrans])` = 15, i.e. the duct's
  trip rect starts at the duct's *bottom* and extends 48 px **upward** (`QSetRect(0,-48,76,0)`). It is
  76 px wide and hangs 8 px left of the 56-px-wide sprite, so it is 20 px wider than the duct itself.
* `kCeilingTrans` mirrors that: 76 × 48 hanging from the sprite top, extending downward.
* Doors are `144 × 322` sprites (interior) or `16 × 322` (exterior); the trip rect is only
  `16 × 240` at `+52` vertically. Doors do not teleport — they set `ignoreLeft`/`ignoreRight` so the
  glider can walk out through the wall, and the actual room change is handled by
  `CheckEscapeLeft`/`CheckEscapeRight` in `Interactions.c` using `kNoLeftWallLimit = -24` /
  `kNoRightWallLimit = 536`.
* Windows behave identically to doors but with a `16 × 44` rect at `+96` vertically.
* **`kInvisTrans`'s hot spot is created with `isOn = true` unconditionally** — it cannot be switched
  off. Only `kDeluxeTrans` is switchable, and that is exactly what its low nibble is for.
* Stairs are capped at one up and one down per room by the editor (`kMaxStairs = 1`), because
  `GetUpStairsRightEdge` / `GetDownStairsLeftEdge` scan for *the* staircase.

### 7.7 `kInvisTrans` geometry, verified

`GetObjectRect` (`GliderPRO/Sources/ObjectRects.c:142`):

```
*itsRect = srcRects[kInvisTrans];                 // 64 x 32
QOffsetRect(itsRect, data.d.topLeft.h, data.d.topLeft.v);
itsRect->bottom = itsRect->top + who->data.d.tall;
itsRect->right += (short)who->data.d.wide;
```

So `tall` is an absolute height (a `short`) while `wide` is an **increment** added to the fixed 64-px
width (a `Byte`, so 0..255 → widths 64..319).

Verified against CD Demo r32 obj[21]: `topLeft = (h 122, v 227)`, `tall = 67`, `wide = 2` →
rect = (left 122, top 227, right 122+64+2 = 188, bottom 227+67 = 294). A 66 × 67 teleport pad.

Verified against CD Demo r33 obj[8]: `tall = 32`, `wide = 0` → 64 × 32, the editor default
(`ObjectAdd.c` sets `tall = newRect.bottom - newRect.top` = 32 but leaves `wide = 0` — an inconsistency
in the original, which is why almost all authored `kInvisTrans` are exactly 64 wide).

### 7.8 `kDeluxeTrans` geometry and state, verified

The `data.d.tall` field is a **packed pair**: high byte = width in 4-px units, low byte = height in
4-px units (`GliderPRO/Sources/ObjectRects.c:152`):

```
wide = (who->data.d.tall & 0xFF00) >> 8;
tall =  who->data.d.tall & 0x00FF;
QSetRect(itsRect, 0, 0, wide * 4, tall * 4);
QOffsetRect(itsRect, data.d.topLeft.h, data.d.topLeft.v);
```

And `data.d.wide` is a **packed nibble pair**: high nibble = authored default state, low nibble = live
state.

Verified against observed bytes:

| Where | `tall` raw | wide units | tall units | pixels | `wide` raw | default | live |
|---|---|---|---|---|---|---|---|
| Art Museum r19 obj[6] | `0x5138` | 0x51 = 81 | 0x38 = 56 | 324 × 224 | `0x11` | 1 | 1 (on) |
| Art Museum r26 obj[19] | `0x1311` | 0x13 = 19 | 0x11 = 17 | 76 × 68 | `0x00` | 0 | 0 (off) |

The 324 × 224 example is a huge pad covering most of a room; the 76 × 68 one is a small doorway-sized
pad. Both are legal. Editor default is `tall = 0x1010` (16 × 16 units = 64 × 64 px) and
`wide = 0x10` (default on, live off — the live nibble is filled in by `SetObjectsToDefaults`).

The max size is `0xFF * 4 = 1020` px in each axis, well beyond a 512 × 322 room.

### 7.9 Transport is not switchable (except `kDeluxeTrans`)

`SetObjectState` (`GliderPRO/Sources/Objects.c:478-494`) sets `changed = false` for `kUpStairs`,
`kDownStairs`, `kMailboxLf`, `kMailboxRt`, `kFloorTrans`, `kCeilingTrans`, all four door types, all
four window types and `kInvisTrans`. Only `kDeluxeTrans` (`:496`) is switchable:

```
 1. switch (action):
 2.   case kToggle:
 3.       newState = !(data.d.wide & 0x0F)
 4.       data.d.wide &= 0xF0
 5.       data.d.wide += newState
 6.       changed = true
 7.   case kForceOn:
 8.       changed = ((data.d.wide & 0x0F) == 0x00)
 9.       ... set low nibble to 1
10.   case kForceOff:
11.       changed = ((data.d.wide & 0x0F) != 0x00)
12.       ... set low nibble to 0
13. if changed and local != -1:
14.     masterObjects[local].theObject.data.d.wide = data.d.wide   // whole byte
15.     if room == thisRoomNumber: thisRoom->objects[object].data.d.wide = data.d.wide
16.     if masterObjects[local].hotNum != -1:
17.         hotSpots[masterObjects[local].hotNum].isOn = newState
```

Note it copies the **whole byte** (both nibbles), preserving the authored default. And unlike the
blower family, no sound is played.

### 7.10 The stairs

`kUpStairs` / `kDownStairs` are 160 × 267 sprites drawn behind everything. Their hot spots are
`kMoveItUp` (112 × 32 at `topLeft`) and `kMoveItDown` (80 × 56 whose bottom-right corner is at
`topLeft + (160, 170)`). `HandleHotSpotCollision` calls `StartGliderGoingUpStairs` /
`StartGliderGoingDownStairs` (`GliderPRO/Sources/Modes.c:120`, `:137`), which:

```
 1. if mode == kGliderGoingFoil:  DeckGliderInFoil()
 2. else if mode == kGliderLosingFoil: RemoveFoilFromGlider()
 3. frame = (mode == kGliderBurning) ? kWasBurning : 0
 4. mode = kGliderGoingUp   (or kGliderGoingDown)
 5. // down only: rightClip = GetUpStairsRightEdge()
```

| Constant | Value | Cite |
|---|---|---|
| `kWasBurning` | `2` | `GliderPRO/Headers/GliderDefines.h:562` |
| `kStairsTop` | `28` | `:474` |
| `kGliderStartsDown` | `32` | `:569` |

Arrival is `ReadyGliderForTripUpStairs` / `ReadyGliderForTripDownStairs`
(`GliderPRO/Sources/Modes.c:519`, `:550`) with

| Constant | Value | Cite |
|---|---|---|
| `kVGliderAppearsComingUp` | `100` | `GliderPRO/Sources/Modes.c:521` |
| `kVGliderAppearsComingDown` | `100` | `GliderPRO/Sources/Modes.c:552` |

Coming up: facing left, `mode = kGliderComingUp`, `src = mask = gliderSrc[2]`, dest placed at
`(rightClip, 100)`. Coming down: facing right, `mode = kGliderComingDown`, `src = gliderSrc[0]`, dest
placed at `(leftClip - kGliderWide, 100)`. `takingTheStairs` (a global, cleared by `DrawLocale` at
`GliderPRO/Sources/RoomGraphics.c:127`) suppresses the normal room-entry animation.

---

## 8. Switches and triggers: the complete cause-and-effect model

This section answers, for each of the nine switch-family object codes: which field holds the live
state, what exactly one throw mutates, whether the effect is local or cross-room, whether it is
instantaneous or queued through the delayed-action queue, and what the frame-order guarantee is
relative to `HandleInteraction`.

### 8.1 The nine object codes

| Constant | Value | Hex | Visible? | Hot-spot action | Cite |
|---|---|---|---|---|---|
| `kLightSwitch` | 65 | `0x41` | yes, 15 × 24 sprite | `kSwitchIt` | `GliderPRO/Headers/GliderDefines.h:377` |
| `kMachineSwitch` | 66 | `0x42` | yes, 16 × 24 | `kSwitchIt` | `:378` |
| `kThermostat` | 67 | `0x43` | yes, 15 × 24 | `kSwitchIt` | `:379` |
| `kPowerSwitch` | 68 | `0x44` | yes, 8 × 8 | `kSwitchIt` | `:380` |
| `kKnifeSwitch` | 69 | `0x45` | yes, 16 × 24 | `kSwitchIt` | `:381` |
| `kInvisSwitch` | 70 | `0x46` | no (editor outline only) | `kSwitchIt` | `:382` |
| `kTrigger` | 71 | `0x47` | no (editor outline only) | `kTriggerIt` | `:383` |
| `kLgTrigger` | 72 | `0x48` | no (editor outline only) | `kTriggerIt` | `:384` |
| `kSoundTrigger` | 73 | `0x49` | no (editor outline only) | `kSoundIt` | `:385` |

**Exactly eight of the nine are "link switches".** `ObjectIsLinkSwitch`
(`GliderPRO/Sources/Objects.c:236-250`) returns true for `kLightSwitch`..`kLgTrigger` and **false for
`kSoundTrigger`**. `GetRoomLinked` (`:149-165`) and `GetObjectLinked` (`:195-207`) have the same
eight-case list. Consequence: a `kSoundTrigger` never gets a `roomLink`, `objectLink` or `localLink` —
all three stay −1 for its whole life, because `ListOneRoomsObjects` calls exactly those two functions
to fill them (`Objects.c:266-282`). This is confirmed empirically: `data.e.who == 255` in **all 122**
`kSoundTrigger` instances across the 22 shipped houses (§8.12).

Three action codes are involved (`GliderPRO/Headers/GliderDefines.h`):

| Action code | Value | Handler | Cite |
|---|---|---|---|
| `kSwitchIt` | `9` (`GliderDefines.h:291`) | `HandleSwitches(who)` | `GliderPRO/Sources/Interactions.c:1320-1322` |
| `kTriggerIt` | `12` (`:294`) | `ArmTrigger(who)` | `Interactions.c:1351-1354` |
| `kSoundIt` | `27` (`:309`) | inline: `PlayPrioritySound(kTriggerSound, kTriggerPriority)` | `Interactions.c:1615-1621` |

Note the dispatch reads

```c
case kTriggerIt:
case kLgTrigger:            // <-- object code 0x48 = 72 used as an ACTION code
ArmTrigger(who);
break;
```

(`Interactions.c:1351-1353`). `kLgTrigger` is 72 and the action-code enum only runs to `kSoundIt = 27`,
so the second label is unreachable dead code. No hot spot is ever created with `action == 72`
(`AddActiveRect` is only ever called with the 28 real action codes). A Go port should drop the label,
not replicate it.

### 8.2 On-disk record: `switchType` (`data.e`)

`GliderPRO/Headers/GliderStructs.h:45-52`, 10 bytes, big-endian, unpadded:

| Field | C type | Size | Union offset | Meaning |
|---|---|---|---|---|
| `topLeft` | `Point` | 4 | 0 | `{v, h}` — **`v` first on disk**; room-relative pixel position of the sprite's top-left |
| `delay` | `short` | 2 | 4 | trigger delay in **tenths of a second**; `× 3` gives frames (§8.6) |
| `where` | `short` | 2 | 6 | target room as `MergeFloorSuite` combo, or −1 = unlinked. For `kSoundTrigger` this is a `'snd '` **resource ID** instead |
| `who` | `Byte` | 1 | 8 | target object index 0..23, or `255` = unlinked |
| `type` | `Byte` | 1 | 9 | one of `kToggle 0`, `kForceOn 1`, `kForceOff 2`, `kOneShot 3` (`GliderDefines.h:441-444`) |

Because `who` is a **`Byte`** on disk but `objectLink` is a **`short`** in memory, the "unlinked"
sentinel changes representation: 255 on disk, −1 in `masterObjects[]`
(`GliderPRO/Sources/Objects.c:203-206`). A Go port must keep the two distinct or it will treat every
unlinked switch as pointing at room-object 255.

Observed bytes (parsed with python3 from `GliderPRO/Houses/Demo House.binhex`):

| Object | Raw 12 bytes (`what` + union) | Decode |
|---|---|---|
| room 2 object 8 `kMachineSwitch` | `00 42` `01 2c 01 c2` `00 00 19 6d 06 00` | topLeft.v=300 topLeft.h=450, delay=0, where=6509 → floor 1 suite 65 (its own room), who=6, type=0 (`kToggle`). Room-object 6 is the `kMacPlus` |
| room 4 object 11 `kKnifeSwitch` | `00 45` `…` | delay=0, where=6709 → floor 1 suite 67, who=10 (the `kTV`), type=`kToggle` |
| room 20 object 4 `kSoundTrigger` | `00 49` `…` | delay=0, **where=3011 = a `'snd '` resource ID**, who=255, type=0 |

`ExtractFloorSuite` (`GliderPRO/Sources/Link.c:41-53`) decodes `where`; it is **version-dependent**:

```
if ((*thisHouse)->version < 0x0200)   { floor = combo / 100 - kNumUndergroundFloors;  suite = combo % 100; }
else                                  { suite = combo / 100;  floor = combo % 100 - kNumUndergroundFloors; }
```

with `kNumUndergroundFloors = 8` (`GliderDefines.h:535`). `MergeFloorSuite(floor, suite) = suite * 100 + floor`
(`Link.c:34-39`) is the *modern* encoding only; there is no legacy encoder.

### 8.3 Hot-spot creation and geometry

`CreateActiveRects` (`GliderPRO/Sources/ObjectRects.c:296`) is called only for **central-room**
objects (`Objects.c:283-286`), so **only the central room's switches are ever throwable**. The
switch/trigger cases:

```c
case kLightSwitch: case kMachineSwitch: case kThermostat: case kPowerSwitch:
case kKnifeSwitch: case kInvisSwitch:  case kTrigger:     case kLgTrigger:
    bounds = srcRects[theObject.what];              // ObjectRects.c:906
    ZeroRectCorner(&bounds);
    QOffsetRect(&bounds, data.e.topLeft.h, data.e.topLeft.v);
    if (what == kTrigger || what == kLgTrigger) {
        if (data.e.where != -1)                     // :913
            hotSpotNumber = AddActiveRect(&bounds, kTriggerIt, who, true, false);
    } else {
        if (data.e.where != -1)                     // :918
            hotSpotNumber = AddActiveRect(&bounds, kSwitchIt, who, true, false);
    }
    break;

case kSoundTrigger:
    QSetRect(&bounds, 0, 0, 48, 48);                // :924  fixed 48x48, NOT srcRects[]
    QOffsetRect(&bounds, data.e.topLeft.h, data.e.topLeft.v);
    if (LoadTriggerSound(data.e.where) == noErr)    // :926
        hotSpotNumber = AddActiveRect(&bounds, kSoundIt, who, true, false);
    break;
```

Three facts a port must reproduce exactly:

1. **`where == -1` suppresses the hot spot entirely.** An unlinked switch is decoration; it cannot be
   thrown, so it never reaches `SetObjectState(-1, …)`. The census finds **7** such objects (§8.12:
   `kKnifeSwitch` 4, `kPowerSwitch` 1, `kInvisSwitch` 1, `kMachineSwitch` 1).
   `who == 255` is **not** checked here, so a switch with `where` valid but `who == 255` *does* get a
   `kSwitchIt` hot spot and *does* call `SetObjectState(room, -1, …)` — an out-of-bounds `objects[-1]`
   read. The census finds **4** such objects in shipped data (`kLightSwitch` 2, `kThermostat` 1,
   `kKnifeSwitch` 1); 7 + 4 = the 11 "unlinked" objects of §8.12. So this path **is** reachable in
   shipped houses and a port must bounds-check it.
2. **The `kSoundTrigger` hot spot is a hard-coded 48 × 48 square**, not `srcRects[kSoundTrigger]`
   (which is 32 × 32, `GliderPRO/Sources/StructuresInit2.c:416`). The drawn editor outline and the
   interactive area therefore differ in size.
3. **A `kSoundTrigger` hot spot exists only if its sound loaded.** `LoadTriggerSound`
   (`GliderPRO/Sources/Sound.c:265-303`) fails when `dontLoadSounds` is set or when
   `theSoundData[kMaxSounds - 1] != nil` — i.e. **only one trigger sound can be resident at a time**
   (`kMaxSounds = 64`, so slot 63, and `kTriggerSound = 63` — the trigger sound *is* the last buffer
   slot). This is why `kMaxSoundTriggers = 1` per room (`GliderPRO/Sources/ObjectAdd.c:17`) and why
   `DrawLocale` calls `DumpTriggerSound()` on every room change
   (`GliderPRO/Sources/RoomGraphics.c:58`).

All hot spots are created with `isOn = true` and `doScrutinize = false`, so a plain rectangle
intersection with the glider's bounding box triggers them (`Interactions.c:1679`), and nothing ever
turns a switch's hot spot off.

Geometry of the eight hot-spot rects, all `srcRects[what]` zeroed and offset by
`data.e.topLeft` (`GliderPRO/Sources/StructuresInit2.c:404-416`):

| Type | Hot-spot size | Sprite source in `switchSrcMap` | Cite |
|---|---|---|---|
| `kLightSwitch` | 15 × 24 | `lightSwitchSrc[0]` @h=0 on, `[1]` @h=16 off, both at v=0 | `StructuresInit.c:460-463` |
| `kMachineSwitch` | 16 × 24 | `machineSwitchSrc[0]` @(h=0,v=24), `[1]` @(h=16,v=24) | `:465-468` |
| `kThermostat` | 15 × 24 | `thermostatSrc[0]` @(h=0,v=48), `[1]` @(h=16,v=48) | `:470-473` |
| `kPowerSwitch` | 8 × 8 | `powerSrc[0]` @(h=0,v=72), `[1]` @(h=8,v=72) | `:475-478` |
| `kKnifeSwitch` | 16 × 24 | `knifeSwitchSrc[0]` @(h=0,v=80), `[1]` @(h=16,v=80) | `:480-483` |
| `kInvisSwitch` | 12 × 12 | none | `StructuresInit2.c:413` |
| `kTrigger` | 12 × 12 | none | `:414` |
| `kLgTrigger` | 48 × 48 | none | `:415` |
| `kSoundTrigger` | **48 × 48 hard-coded** (`srcRects` says 32 × 32) | none | `ObjectRects.c:924` |

`switchSrcRect` is `QSetRect(&switchSrcRect, 0, 0, 32, 104)` — `QSetRect`'s argument order is
`(left, top, right, bottom)`, so the GWorld is **32 px wide × 104 px tall** = 3 328 pixels. (The
source comment says "3360 pixels", which is arithmetically wrong; `StructuresInit.c:455`.) PICT
`kSwitchPictID` = **4003** (`StructuresInit.c:23`), loaded at `kPreferredDepth` with **no mask**, so
switches blit `srcCopy` and are opaque rectangles (`StructuresInit.c:456-458`). The two frames of
each switch sit side by side at a 16-px horizontal pitch; index `[0]` is **on**, `[1]` is **off**
(`DrawLightSwitch`, `ObjectDraw2.c:301-315`; `DrawMachineSwitch` `:319-329`; `DrawThermostat`
`:333-343`; `DrawPowerSwitch` `:347-357`; `DrawKnifeSwitch` `:361-371` — all five are literally
`if (state) CopyBits(…Src[0]…) else CopyBits(…Src[1]…)` from `switchSrcMap` into **`backSrcMap`**,
not `workSrcMap`).

### 8.4 The instantaneous path: `HandleSwitches`

`GliderPRO/Sources/Interactions.c:985-1155`. Numbered pseudocode, original variable names in
parentheses:

```
HandleSwitches(hotPtr who):
 1. if (who->stillOver) return                                       // :990  debounce
 2. whoLinked    = who->who                                          // :993  index into masterObjects[]
 3. roomLinked   = masterObjects[whoLinked].roomLink                  // :994
 4. objectLinked = masterObjects[whoLinked].objectLink                // :995
 5. linkIndex    = masterObjects[whoLinked].localLink                 // :996
 6. if SetObjectState(roomLinked, objectLinked,
                      masterObjects[whoLinked].theObject.data.e.type,
                      linkIndex):                                    // :998
 7.     newRect = who->bounds;  QOffsetRect(&newRect, playOriginH, playOriginV)
 8.     switch (masterObjects[whoLinked].theObject.what):             // :1003
          kLightSwitch   -> PlayPrioritySound(kSwitchSound, kSwitchPriority); DrawLightSwitch(&newRect, newState)
          kMachineSwitch -> ditto + DrawMachineSwitch
          kThermostat    -> ditto + DrawThermostat
          kPowerSwitch   -> ditto + DrawPowerSwitch
          kKnifeSwitch   -> ditto + DrawKnifeSwitch
          kInvisSwitch   -> nothing                                   // :1030-1031
 9.     CopyRectBackToWork(&newRect); AddRectToWorkRects(&newRect)    // :1033-1034
10.     if (linkIndex != -1):                                         // :1036
11.         switch (masterObjects[linkIndex].theObject.what):  ...     // :1038-1150, see below
12. who->stillOver = true                                             // :1154
```

Everything in this routine is **synchronous**: by the time `HandleSwitches` returns, the house record,
the master copy, the room copy, the affected hot spot, the affected `dinahs[]` slot, the back GWorld
and the dirty-rect list have all been updated. There is no queue.

`kSwitchSound = 9`, `kSwitchPriority = 700` (`GliderDefines.h:64`, `:156`) — one of the highest
priorities in the game, above every appliance and enemy sound, below only `kFadeIn/Out`, transit,
shred, fire and `kTriggerPriority = 999`.

The **inner switch (step 11) is keyed on the TARGET's object code**, and this is where all the
cross-object side effects live:

| Target `what` | Effect | Cite |
|---|---|---|
| `kRedClock`, `kBlueClock`, `kYellowClock`, `kPaper`, `kBattery`, `kBands`, `kFoil`, `kStar`, `kHelium` | `RestoreFromSavedMap(roomLinked, objectLinked, true)` then `AddSparkle(&bounds)` — **`bounds` is an uninitialised local** | `Interactions.c:1049-1050` |
| `kCuckoo` | `RestoreFromSavedMap` + `StopPendulum(roomLinked, objectLinked)` | `:1054-1055` |
| `kGreaseRt`, `kGreaseLf` | `SpillGrease(masterObjects[linkIndex].dynaNum, masterObjects[linkIndex].hotNum)` | `:1060-1061` |
| `kInvisBonus`, `kSlider` | nothing | `:1064-1066` |
| `kDeluxeTrans` | nothing (state already changed by `SetObjectState`; the hot spot's `isOn` was flipped there) | `:1068-1069` |
| `kSoundTrigger` | `PlayPrioritySound(kTriggerSound, kTriggerPriority)` | `:1071-1073` |
| **all 8 light types** | `RedrawRoomLighting()` | `:1075-1084` |
| `kShredder` | nothing extra (its `hotSpots[].isOn` was flipped inside `SetObjectState`) | `:1086-1087` |
| `kToaster` | `ToggleToaster(dynaNum)` | `:1089-1091` |
| `kMacPlus` | `ToggleMacPlus(dynaNum)` | `:1093-1095` |
| `kGuitar` | `PlayPrioritySound(kChordSound, kChordPriority)` (23 / 302) | `:1097-1099` |
| `kTV` | `ToggleTV(dynaNum)` | `:1101-1103` |
| `kCoffee` | `ToggleCoffee(dynaNum)` | `:1105-1107` |
| `kOutlet` | `ToggleOutlet(dynaNum)` | `:1109-1111` |
| `kVCR` | `ToggleVCR(dynaNum)` | `:1113-1115` |
| `kStereo` | `ToggleStereos(dynaNum)` | `:1117-1119` |
| `kMicrowave` | `ToggleMicrowave(dynaNum)` | `:1121-1123` |
| `kBalloon` | `ToggleBalloon(dynaNum)` | `:1125-1127` |
| `kCopterLf`, `kCopterRt` | `ToggleCopter(dynaNum)` | `:1129-1132` |
| `kDartLf`, `kDartRt` | `ToggleDart(dynaNum)` | `:1134-1137` |
| `kBall` | `ToggleBall(dynaNum)` | `:1139-1141` |
| `kDrip` | `ToggleDrip(dynaNum)` | `:1143-1145` |
| `kFish` | `ToggleFish(dynaNum)` | `:1147-1149` |

**There is no case for any of the nine switch/trigger types.** A switch wired to another switch
therefore does nothing beyond `SetObjectState`, and `SetObjectState`'s own switch case sets
`changed = false` (`Objects.c:533-542`), so `HandleSwitches` never even enters step 7 — no sound, no
sprite redraw, nothing. 18 such links exist in the shipped houses (§8.12) and every one of them is a
no-op. Also absent: the **air-flow types** (`kFloorVent`, `kInvisBlower`, `kLiftArea`, …). Those need
no side effect because `SetObjectState` itself flips `hotSpots[hotNum].isOn`, plays
`kBlowerOn`/`kBlowerOff`, and the vent sprite is not redrawn (vent sprites look identical on and off;
only the invisible lift column changes).

#### 8.4.1 `newState` is a global

`Boolean newState;` is declared at file scope in `GliderPRO/Sources/Objects.c:78` and is the only
channel by which `SetObjectState` tells `HandleSwitches` what the new state *is*. `HandleSwitches`
reads it at `Interactions.c:1007`, `:1012`, `:1017`, `:1022`, `:1027`. In a Go port this must become a
second return value (or an out-parameter); leaving it as a package-level variable will break as soon
as anything is made concurrent, and it is genuinely load-bearing — it decides which of the two switch
sprite frames is drawn.

`SetObjectState` writes `newState` in every branch that can return `true`, so the coupling is sound
for the *reachable* paths. It does **not** write it for `kSlider`, `kKnifeSwitch` (see §8.7) or an
empty target, but those branches also do not return `true` reliably (they leave `changed`
uninitialised), so a port that returns `(changed, newState)` should return `(false, false)` there.

### 8.5 `SetObjectState` — the state mutation table

`GliderPRO/Sources/Objects.c:366-699`. Signature
`Boolean SetObjectState (short room, short object, short action, short local)`.

The routine is one 320-line `switch` on `(*thisHouse)->rooms[room].objects[object].what`. Every branch
that supports switching has the identical three-way shape:

```
kToggle  (action 0): newState = !current;                       current = newState;  changed = true
kForceOn (action 1): changed  = (current == false); newState = true;   current = newState
kForceOff(action 2): changed  = (current == true);  newState = false;  current = newState
```

and then, gated on `if ((changed) && (local != -1))`, propagates to the **three other copies**:

```
masterObjects[local].theObject.data.X.state = newState        // the live 9-room copy
if (room == thisRoomNumber)
    thisRoom->objects[object].data.X.state  = newState        // the editor/serialisation copy
```

`kOneShot` (action 3) is **not handled by any branch** — it falls through the inner `switch (action)`
with no case, leaving `changed` at whatever the enclosing branch initialised it to (nothing). This is
harmless only because the two object types that use `kOneShot` (`kTrigger`, `kLgTrigger`, 100 % of
shipped instances, §8.12) never *receive* a `SetObjectState` call with their own `type`; their `type`
is passed as the `action` for **their target**. So `kOneShot` reaching `SetObjectState` means a
`kTrigger` whose target is a switchable object — and then that target's `switch (action)` matches
nothing, `changed` is garbage, and no state changes. In the shipped corpus, `kTrigger`/`kLgTrigger`
targets are `kInvisSwitch` (171), `kGuitar` (42), appliances and enemies (§8.12) — for the
`kInvisSwitch` majority this lands in the `changed = false` switch branch, so it is inert; for
`kToaster`, `kOutlet`, `kDrip`, `kFish`, `kBalloon`, `kCopter*`, `kDart*` it lands in the
`switch (action)` with no matching case, leaving the appliance/enemy state **unchanged** and `changed`
**uninitialised**. That is deliberate: `FireTrigger` ignores `SetObjectState` entirely for those types
and calls `Trigger*()` directly (§8.6).

Complete branch table:

| Branch (target `what`) | Field mutated | `changed` | Extra side effects inside `SetObjectState` | Cite |
|---|---|---|---|---|
| `kFloorVent`, `kCeilingVent`, `kFloorBlower`, `kCeilingBlower`, `kLeftFan`, `kRightFan`, `kSewerGrate`, `kInvisBlower`, `kGrecoVent`, `kSewerBlower`, `kLiftArea` | `data.a.state` (union offset 7) | 3-way | `PlayPrioritySound(kBlowerOn 14 / kBlowerOff 15, 701 / 702)`; `if (hotNum != -1) hotSpots[hotNum].isOn = newState` | `Objects.c:375-418` |
| `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ` | — | `false` | — | `:420-426` |
| 15 furniture types | — | `false` | — | `:428-444` |
| 14 prize types (`kRedClock`..`kHelium`) | `data.c.state` (offset 8) — **always forced to `false`, `action` ignored** | `(was true)` | `masterObjects[local].theObject.data.a.state = false` ← **wrong union arm**; `if (room == thisRoomNumber) { thisRoom->…data.c.state = false; if (hotNum != -1) hotSpots[hotNum].isOn = false; }` | `:446-473` |
| `kSlider` | — | **UNINITIALISED** (`case kSlider: break;`) | — | `:475-476` |
| 15 transport types (`kUpStairs`..`kInvisTrans`) | — | `false` | — | `:478-494` |
| `kDeluxeTrans` | low nibble of `data.d.wide` | 3-way | `masterObjects[local].theObject.data.d.wide = …`; `if (hotNum != -1) hotSpots[hotNum].isOn = newState` | `:496-531` |
| `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kInvisSwitch`, `kTrigger`, `kLgTrigger`, `kSoundTrigger` — **`kKnifeSwitch` IS MISSING** | — | `false` | — | `:533-542` |
| 8 light types | `data.f.state` (offset 9) | 3-way | **no sound, no hot-spot touch** (lights have no hot spot) | `:544-578` |
| `kGuitar` | — | `false` | — | `:580-582` |
| `kStereo` | **global `isPlayMusicGame`** — not any house field, `action` ignored, always toggles | `true` | — | `:584-588` |
| `kShredder`, `kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kMicrowave` | `data.g.state` (offset 9) | 3-way | `if (room == thisRoomNumber && what == kShredder) hotSpots[masterObjects[local].hotNum].isOn = newState` — **not guarded by `hotNum != -1`** | `:590-628` |
| `kCinderBlock`, `kFlowerBox`, `kCDs`, `kCustomPict` | — | `false` | — | `:630-635` |
| `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish` | `data.h.state` (offset 9) | 3-way | — | `:637-671` |
| `kCobweb` | — | `false` | — | `:673-675` |
| 15 clutter types | — | `false` | — | `:677-693` |
| **no `default:` label** | — | **UNINITIALISED** | — | `:695` |

Four defects fall out of this table and all four are reachable in principle:

1. **`changed` is never initialised** (`Objects.c:369`). Three inputs leave it garbage:
   `what == kSlider` (0x2F), `what == kKnifeSwitch` (0x45), and any `what` not in the list —
   including `kObjectIsEmpty` (−1), which is what an empty target slot looks like. The shipped corpus
   has **5 links to empty slots** and **3 links to a `kKnifeSwitch`** (§8.12). On 68k/PPC the garbage
   is whatever was on the stack; if it happens to be non-zero, `HandleSwitches` proceeds to play the
   switch sound, draw the switch with the *stale* global `newState`, and dirty the rect — a visible
   glitch that depends on stack contents. A Go port gets `false` for free and will simply never
   glitch; that is the correct behaviour to pick, but it *is* a behavioural divergence.
2. **The prize branch writes the wrong union arm** (`Objects.c:465`:
   `masterObjects[local].theObject.data.a.state = false`). `blowerType.state` is at union offset 7;
   `bonusType.state` is at union offset **8** (`GliderStructs.h:16` vs `:32`). Union offset 7 is the
   **low byte of `bonusType.points`** (a big-endian `short` at offsets 6-7). So force-offing a prize
   zeroes the low byte of the master copy's point value: 100 → 0, 300 → 256, 500 → 256. The master
   copy's `data.c.state` is left `true`. This is latent, not observable: the only reader is
   `HandleRewards` (`Interactions.c:920`) which reads `points` *before* calling `SetObjectState`, and
   the same-room path disables the hot spot so the prize can never be collected afterwards. **Do not
   replicate; write `data.c.state`.**
3. **`kKnifeSwitch` is absent from the switch-family case list** at `:533-542` even though it is
   present in the identical list in `GetObjectState` (`:793-800`)… where it is *also* absent. In
   `GetObjectState` the omission is harmless (`theState` is pre-initialised to `true` at `:708` and
   the listed switches also just `break`), but in `SetObjectState` it means "throw a switch that
   targets a knife switch" is undefined behaviour.
4. **The `kShredder` hot-spot write is unguarded.** Every other branch tests
   `masterObjects[local].hotNum != -1` first (`:415`, `:469`, `:528`); `:624-625` does not. A
   `kShredder` in the central room always has a hot spot (`ObjectRects.c:940`), and the write is
   already inside `if (room == thisRoomNumber)`, so `hotNum` is in fact never −1 there — but the
   invariant is accidental, not enforced.

### 8.6 The delayed path: the trigger queue

`GliderPRO/Sources/Triggers.c`, 205 lines, entirely dedicated to this.

```c
#define kMaxTriggers  16                        // :12
typedef struct {
    short   object, room;                       // :17  target room-object and room number
    short   index, timer;                       // :18  index = masterObjects[] index of the TRIGGER
    short   what;                               // :19  written once, NEVER READ
    Boolean armed;                              // :20
} trigType;
trigType triggers[kMaxTriggers];                // :28
```

`triggers[]` is a plain 16-slot array of C structs — 12 bytes each on 68k/PPC, 192 bytes total. It is
**not** a priority queue and **not** ordered; `FindEmptyTriggerSlot` (`:59-75`) linear-scans for the
first `!armed` slot and returns −1 if all 16 are busy, in which case `ArmTrigger` silently drops the
trigger (but still sets `stillOver`, so the player must step off and back on to retry).

#### Arming

```
ArmTrigger(hotPtr who):                                        // Triggers.c:34-55
 1. if (who->stillOver) return                                 // :38  debounce
 2. where = FindEmptyTriggerSlot()                             // :41
 3. if (where != -1):
 4.     whoLinked = who->who                                   // :45  masterObjects[] index of the trigger
 5.     triggers[where].room   = masterObjects[whoLinked].roomLink        // :46
 6.     triggers[where].object = masterObjects[whoLinked].objectLink      // :47
 7.     triggers[where].index  = whoLinked                                // :48
 8.     triggers[where].timer  = masterObjects[whoLinked].theObject.data.e.delay * 3   // :49
 9.     triggers[where].what   = masterObjects[triggers[where].object].theObject.what  // :50  <-- BUG
10.     triggers[where].armed  = true                                     // :51
11. who->stillOver = true                                                 // :54
```

Line 9 is doubly wrong and doubly harmless. It indexes `masterObjects[]` with `objectLink`, which is a
**room-object index 0..23**, not a master index — so for anything other than a central-room target it
reads the wrong entry, and when `objectLink == -1` it reads `masterObjects[-1]`. And the value it
stores, `triggers[].what`, is **never read anywhere in the program** (grep confirms `.what` on a
`trigType` appears only at `Triggers.c:50`). A Go port should delete the field; keeping it means
reproducing an out-of-bounds read for no observable gain.

#### The delay formula

`timer = data.e.delay * 3` frames. `HandleTriggers` decrements **then** tests `<= 0`:

```
HandleTriggers():                                              // Triggers.c:79-96
 for i in 0..15:
     if triggers[i].armed:
         triggers[i].timer--                                   // :87
         if triggers[i].timer <= 0:                            // :88
             triggers[i].timer = 0
             triggers[i].armed = false
             FireTrigger(i)                                    // :92
```

Because `HandleTriggers` runs **after** `HandleInteraction` in the same frame
(`GliderPRO/Sources/Play.c:482-484`), the arming frame is also a decrement frame. Exact latency:

| `data.e.delay` | `timer` at arm | Frames until `FireTrigger` | Wall-clock at 30 fps |
|---|---|---|---|
| 0 | 0 | **0** — fires in the same frame the glider touches it (0 → −1 ≤ 0) | 0 ms |
| 1 | 3 | 3 | 100 ms |
| 2 | 6 | 6 | 200 ms |
| `d` | `3d` | `3d` | `100 d` ms |

So `delay` is in **tenths of a second** (3 frames at 30 fps = 1/10 s), and the effective latency is
exactly `3 × delay` frames — the arming frame's decrement is one of the three. The largest shipped
value is `delay = 120` → 360 frames = 12 seconds (`kTrigger` in 1 house, `kLgTrigger` in 2, §8.12).

`ZeroTriggers` (`:198-204`) clears **only `armed`**; `room`, `object`, `index`, `timer` and `what`
keep their previous values. It is called from `DrawLocale` (`RoomGraphics.c:55`), so every room change
cancels all pending triggers. A Go port zeroing the whole slot is behaviourally identical, since
`armed == false` gates every read.

#### Firing

```
FireTrigger(short index):                                      // Triggers.c:100-194
 1. triggerIs = triggers[index].index                          // :105  master index of the TRIGGER
 2. if masterObjects[triggerIs].localLink != -1:               // :107
 3.     triggeredIs = masterObjects[triggerIs].localLink       // :109  master index of the TARGET
 4.     switch (masterObjects[triggeredIs].theObject.what):    // :110
          kGreaseRt / kGreaseLf:
              if SetObjectState(triggers[index].room, triggers[index].object, kForceOn, triggeredIs)
                  SpillGrease(masterObjects[triggeredIs].dynaNum,
                              masterObjects[triggeredIs].hotNum)          // :114-119
          kLightSwitch / kMachineSwitch / kThermostat / kPowerSwitch / kKnifeSwitch / kInvisSwitch:
              TriggerSwitch(masterObjects[triggeredIs].dynaNum)           // :122-129  see below
          kSoundTrigger:
              PlayPrioritySound(kChordSound, kChordPriority)  // "// Change me"  :131-133
          kToaster:   TriggerToast  (dynaNum)                             // :135-137
          kGuitar:    PlayPrioritySound(kChordSound, kChordPriority)      // :139-141
          kCoffee:    PlayPrioritySound(kCoffeeSound 34, kCoffeePriority 306)  // :143-145
          kOutlet:    TriggerOutlet (dynaNum)                             // :147-149
          kBalloon:   TriggerBalloon(dynaNum)                             // :151-153
          kCopterLf/Rt: TriggerCopter(dynaNum)                            // :155-158
          kDartLf/Rt:   TriggerDart  (dynaNum)                            // :160-163
          kDrip:      TriggerDrip   (dynaNum)                             // :165-167
          kFish:      TriggerFish   (dynaNum)                             // :169-171
 5. else:                                                                 // :174-193  the DEAD branch
 6.     triggeredIs = masterObjects[triggerIs].localLink        // == -1 by construction
 7.     switch ((*thisHouse)->rooms[triggers[index].room].objects[triggers[index].object].what):
          kGreaseRt / kGreaseLf:
              if SetObjectState(…, kForceOn, -1)  SpillGrease(masterObjects[-1].dynaNum,
                                                              masterObjects[-1].hotNum)
```

Notes:

* **12 target families are reachable; 12 more are silently ignored.** A trigger wired to a light, a
  prize, a `kShredder`, a `kMacPlus`, a `kTV`, a `kVCR`, a `kStereo`, a `kMicrowave`, a `kBall`, a
  transport, a blower or furniture does **nothing at all** — there is no case for them. Compare
  `HandleSwitches`, which handles all of those. This is the single biggest semantic difference
  between a switch and a trigger.
* **`FireTrigger` does not call `SetObjectState` except for grease.** So a trigger fired at a
  `kToaster` launches the toast without changing `data.g.state`; the appliance's persistent on/off
  flag is untouched. Triggers are *momentary pulses*, switches are *state changes*.
* **The `else` branch (step 5) is dead and would be catastrophic if reached.** `triggeredIs` is
  assigned `localLink` which the `if` already proved to be −1, then used to index
  `masterObjects[-1]`. It can only be entered when the trigger's target is outside the loaded 3 × 3
  neighbourhood, and it does something only if the target is grease. The shipped corpus has exactly
  **one** cross-room trigger (Land of Illusion room 1 "Steamy Star" object 16 → room 174 "Edge Of
  Light" object 15, ΔSuite = −5, ΔFloor = +1) and its target slot is **empty**, so `what` is −1 and
  even the grease case does not match. The branch is unreachable in 1.0.4 with shipped data.
* **`TriggerSwitch(dynaNum)` is deliberate, not a typo.** `TriggerSwitch`
  (`GliderPRO/Sources/Trip.c:146-149`) is literally `HandleSwitches(&hotSpots[who])`, and
  `DrawARoomsObjects` stores a switch's own `hotNum` into its `dynaNum`
  (`GliderPRO/Sources/ObjectDrawAll.c:518`, `:531`, `:544`, `:557`, `:570`, `:574`). So
  trigger → switch → switch's own target is a real, working two-stage chain, and it is the single most
  common trigger wiring in the shipped houses (171 of 320 triggers, §8.12). The index is nonetheless
  mis-computed for **neighbour-room** switches (§2.5) and would index `hotSpots[-1]` for a switch with
  no hot spot; neither case occurs in shipped data.

`kChordSound = 23` / `kChordPriority = 302`; `kCoffeeSound = 34` / `kCoffeePriority = 306`;
`kTriggerSound = 63` / `kTriggerPriority = 999` (`GliderDefines.h:78`, `:89`, `:118`, `:131`, `:135`,
`:180`).

#### The `Trigger*` / `Toggle*` primitive table

All of `GliderPRO/Sources/Trip.c`. `Toggle*` is what a **switch** calls; `Trigger*` is what a
**trigger** calls. Note that they are *different operations* for the same object.

| Routine | Body | Cite |
|---|---|---|
| `ToggleToaster(i)` | `active = !active` | `Trip.c:22-25` |
| `ToggleMacPlus(i)` | `active = !active; timer = active ? 40 : 10` | `:29-36` |
| `ToggleTV(i)` | `active = !active`; if QuickTime movie owns this TV (`thisMac.hasQT && hasMovie && tvInRoom && tvWithMovieNumber == i`) then `GoToBeginningOfMovie`+`StartMovie`+`tvOn=true` / `StopMovie`+`tvOn=false`; then `timer = 4` | `:40-58` |
| `ToggleCoffee(i)` | `active = !active; timer = 4` | `:62-66` |
| `ToggleOutlet(i)` | `active = !active` | `:70-73` |
| `ToggleVCR(i)` | `active = !active; timer = 4` | `:77-81` |
| `ToggleStereos(i)` | **`if (timer == 0)`** `{ active = !active; timer = 4; }` — the only guarded toggle | `:85-92` |
| `ToggleMicrowave(i)` | `active = !active; timer = 4` | `:96-100` |
| `ToggleBalloon`, `ToggleCopter`, `ToggleDart`, `ToggleBall`, `ToggleDrip`, `ToggleFish` | `active = !active` | `:104-142` |
| `TriggerSwitch(who)` | `HandleSwitches(&hotSpots[who])` | `:146-149` |
| `TriggerToast(who)` | `if (!moving) { if (active) { vVel = -count; frame = 0; moving = true; kToastLaunchSound } else frame = timer; }` | `:153-167` |
| `TriggerOutlet(who)` | `if (position == 0) { if (active) { position = 1; timer = kLengthOfZap (30); kZapSound } else timer = count; }` | `:171-184` |
| `TriggerDrip(who)` | `if (!moving && timer > 7) timer = 7` | `:188-192` |
| `TriggerFish(who)` | `if (active && !moving) { whole = dest; moving = true; frame = 4; kFishOutSound }` | `:196-205` |
| `TriggerBalloon`, `TriggerCopter`, `TriggerDart` | `if (!moving) timer = kStartSparkle + 1` (= 5) | `:209-231` |
| `UpdateOutletsLighting(room, nLights)` | for every `dinahs[i]` with `type == kOutlet && room == room`: `hVel = nLights` | `:235-244` |

Two asymmetries worth internalising:

* A **switch** on a `kToaster` flips `active` (so it stops/starts the periodic launcher); a **trigger**
  on the same toaster launches one slice immediately and leaves `active` alone.
* A **switch** on an `kOutlet` flips `active`; a **trigger** starts a 30-frame zap immediately.

`ToggleStereos`'s `timer == 0` guard exists because `HandleStereo` calls
`ToggleMusicWhilePlaying()` when its timer hits 0 (`GliderPRO/Sources/Dynamics.c:681`, `:698`), and
re-toggling mid-countdown would desynchronise the music state from `isPlayMusicGame`. Note the
mismatch: `SetObjectState`'s `kStereo` branch (`Objects.c:584-588`) toggles `isPlayMusicGame`
**unconditionally** and returns `changed = true`, but `ToggleStereos` may then refuse to act. Throwing
a stereo switch twice within 4 frames therefore leaves `isPlayMusicGame` flipped while the stereo's
`active` and the music playback are not — a real, reproducible desync.

### 8.7 `GetObjectState` — the read side

`GliderPRO/Sources/Objects.c:703-871`. Pre-initialises `theState = true` (`:708`) and reads:

| Target family | Returns | Cite |
|---|---|---|
| 11 air-flow types | `data.a.state` | `Objects.c:714-726` |
| 14 prize types incl. `kGreaseRt/Lf` | `data.c.state` | `:752-767` |
| `kDeluxeTrans` | `data.d.wide & 0x0F` | `:789-791` |
| 8 light types | `data.f.state` | `:803-812` |
| `kStereo` | the **global `isPlayMusicGame`** | `:814-816` |
| `kShredder`, `kToaster`, `kMacPlus`, **`kGuitar`**, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kMicrowave` | `data.g.state` | `:818-828` |
| 8 enemy types | `data.h.state` | `:836-845` |
| flames, furniture, `kSlider`, 15 transports, the 8 switch/trigger types (`kKnifeSwitch` again absent), `kCinderBlock`/`kFlowerBox`/`kCDs`/`kCustomPict`, `kCobweb`, 15 clutter types, and **anything unmatched** | **`true`** | `:728-750`, `:769-787`, `:793-801`, `:830-834`, `:847-865` |

`kGuitar` is in the `data.g.state` group here but has its own `changed = false` branch in
`SetObjectState` (`:580-582`) — asymmetric but consistent in effect (readable, not writable).

This function is called from exactly one place at runtime: the five visible switch cases in
`DrawARoomsObjects` (`ObjectDrawAll.c:516`, `:529`, `:542`, `:555`, `:568`), which resolve the
target's room from `data.e.where` and ask for its state so the switch sprite can be drawn in the
matching position:

```
ExtractFloorSuite(thisObject.data.e.where, &floor, &suite);   // ObjectDrawAll.c:513
room = GetRoomNumber(floor, suite);                           // :514
obj  = (short)thisObject.data.e.who;                          // :515
DrawLightSwitch(&itsRect, GetObjectState(room, obj));         // :516
```

Note the **unguarded** `room` and `obj`. `GetRoomNumber` returns `kRoomIsEmpty` (−1) for a target room
that does not exist (`GliderPRO/Sources/Room.c:737-759`) and `obj` is 255 for an unlinked switch, so
this is `(*thisHouse)->rooms[-1].objects[255]` in the bad cases. The shipped corpus contains **1
`kLightSwitch` with a nonexistent target room** and **11 unlinked switches** (§8.12). Ten of those
eleven hit this path every time their room is drawn — the exception is the single unlinked
`kInvisSwitch`, because `DrawARoomsObjects` has no `GetObjectState` call for `kInvisSwitch` (it is
never drawn in play mode) and `kTrigger`/`kLgTrigger` likewise have none. The read is harmless on 68k/PPC (it just yields a
meaningless `Boolean` and picks a sprite frame); a Go port must bounds-check and choose a default —
`true` matches `GetObjectState`'s pre-initialised value and is the safer choice.

### 8.8 Frame-order guarantees

The fixed frame order (`GliderPRO/Sources/Play.c:430-497`; the two-player and one-player arms are
identical in ordering):

```
 1. gameFrame++;  evenFrame = !evenFrame;                     // Play.c:434-435
 2. if (doBackground) do { HandlePlayEvent(); } while (switchedOut);
 3. HandleTelephone();                                        // :445
 4. HandleDynamics();                  // all dinahs[] tick + appliances self-render   :449 / :475
 5. GetInput(&theGlider[, &theGlider2]);                       // :452 / :481
 6. HandleInteraction();               // CheckForHotSpots -> HandleSwitches / ArmTrigger  :454 / :482
 7. HandleTriggers();                  // drain the 16-slot queue -> FireTrigger      :456 / :484
 8. HandleBands();                                            // :457 / :485
 9. HandleGlider(&theGlider[, &theGlider2]);                   // :460 / :487
10. [MoviesTask(theMovie, 0) if QuickTime TV on]               // :467 / :492
11. RenderFrame();                                            // :469 / :494
12. HandleDynamicScoreboard();                                 // :470 / :495
```

The guarantees that follow, in the form a port must preserve:

| Guarantee | Why |
|---|---|
| A switch throw is visible in the **same** frame it is triggered | step 6 mutates state and blits into `backSrcMap`; step 11 composites |
| A `delay == 0` trigger fires in the **same** frame it is armed | step 7 runs after step 6, and `timer = 0` then `timer--` → −1 ≤ 0 |
| A `delay == d ≥ 1` trigger fires exactly `3d` frames after arming | one of the `3d` decrements happens on the arming frame |
| Appliance timers advance **before** the player can interact | step 4 precedes step 6, so a `Toggle*` issued in step 6 first ticks on the *next* frame |
| A trigger that fires at step 7 and calls `Toggle*` also first ticks on the next frame | step 4 has already run |
| A trigger that fires at step 7 and calls `TriggerSwitch` runs the **whole** of `HandleSwitches` re-entrantly, including `RedrawRoomLighting()` and a full room re-composite, *after* `HandleInteraction` has finished | `TriggerSwitch` = `HandleSwitches` |
| `stillOver` is cleared for non-intersecting hot spots at step 6, in the same loop that fires them | `CheckForHotSpots` `else hotSpots[i].stillOver = false` (`Interactions.c:1683`) |
| A room change (`DrawLocale`) discards every armed trigger and every `dinahs[]` slot | `ZeroTriggers` + `ZeroDinahs` at `RoomGraphics.c:52`, `:55` |

The **debounce rule** is uniform: a switch or trigger fires on the frame the glider's rect *first*
intersects it and not again until a frame in which it does not intersect. `stillOver` is set by
`HandleSwitches` (`:1154`) and `ArmTrigger` (`Triggers.c:54`) whether or not anything actually
happened — including the `where == -1` early return path and the "all 16 trigger slots full" path.
`kStrumIt` and `kSoundIt` implement the same latch inline (`Interactions.c:1344-1348`, `:1616-1620`).

### 8.9 `kSoundTrigger` in full

`kSoundTrigger` is the odd one out in every respect:

| Property | Value |
|---|---|
| Is it a "link switch"? | **No** — excluded from `ObjectIsLinkSwitch`, `GetRoomLinked`, `GetObjectLinked` (`Objects.c:236-250`, `:149-165`, `:195-207`) |
| `roomLink` / `objectLink` / `localLink` | always −1 |
| `data.e.where` | a **`'snd '` resource ID**, not a room combo |
| `data.e.who` | always 255 in shipped data (122/122) |
| `data.e.type` | always `kToggle` in shipped data (122/122) — and never read |
| `data.e.delay` | always 0 in shipped data (122/122) — and never read |
| Hot-spot rect | hard-coded 48 × 48 at `topLeft` (`ObjectRects.c:924`) |
| Hot-spot action | `kSoundIt` = 27 |
| Effect | `PlayPrioritySound(kTriggerSound 63, kTriggerPriority 999)` (`Interactions.c:1618`) |
| Queued? | No. Instantaneous, inside `HandleInteraction`. |
| Per-room limit | 1 (`kMaxSoundTriggers` = 1, `ObjectAdd.c:17`, enforced by `HowManySoundObjects` at `:989-999`) |
| Resident sound slot | `theSoundData[kMaxSounds - 1]` = `theSoundData[63]`, loaded by `LoadTriggerSound` on room entry, freed by `DumpTriggerSound` on room exit (`Sound.c:265-312`, `RoomGraphics.c:57-58`) |

`kTriggerPriority = 999` is the highest priority in the game, and `PlayPrioritySound` has a special
case for it: if **any** of the three channels is already playing at `kTriggerPriority` the new request
is dropped (`Sound.c:47-51`), so a trigger sound never overlaps itself.
`FlushAnyTriggerPlaying` (`:89-129`) issues `quietCmd` + `flushCmd` on every channel at that priority
and is called from `DrawLocale` before `DumpTriggerSound` — otherwise the Sound Manager would be
playing out of a `DisposePtr`'d buffer.

`LoadTriggerSound` calls `HLock(theSound)` (`Sound.c:288`) and **never `HUnlock`s**, unlike
`LoadBufferSounds` which balances them (`:331`/`:333`). It does `ReleaseResource` afterwards
(`:291`, `:297`), which makes the lock moot, but a Resource-Manager-faithful port should not copy the
imbalance.

A Go port replaces this whole mechanism with "decode PCM for sound resource ID `where`, keep one
slot": there is no Resource Manager, no handle locking, and no reason for the single-slot limit other
than fidelity to `kMaxSoundTriggers`.

### 8.10 Per-type answer table

The five questions, answered for all nine types.

| Type | Live state lives in | One throw mutates | Local or cross-room | Instantaneous or queued | Frame-order guarantee |
|---|---|---|---|---|---|
| `kLightSwitch` | nothing of its own; rendered from `GetObjectState(target)`. Debounce latch in `hotSpots[hotNum].stillOver`. The global `newState` carries the result to the sprite | target's `data.?.state` in `thisHouse`, in `masterObjects[localLink]`, and in `thisRoom` if same room; possibly `hotSpots[].isOn`; possibly a `dinahs[]` slot via `Toggle*`; possibly `RedrawRoomLighting` | authored record: **any room in the house**. Live tables + sound + sprite: **only if `localLink != -1`**, i.e. target in the loaded 3 × 3 | **instantaneous**, inside `HandleInteraction` | complete before step 7; visible in the same frame's `RenderFrame` |
| `kMachineSwitch` | ditto | ditto | ditto | instantaneous | ditto |
| `kThermostat` | ditto | ditto | ditto | instantaneous | ditto |
| `kPowerSwitch` | ditto | ditto | ditto | instantaneous | ditto |
| `kKnifeSwitch` | ditto | ditto | ditto | instantaneous | ditto |
| `kInvisSwitch` | ditto, but **no sprite** — `HandleSwitches` case is `break` (`Interactions.c:1030-1031`), so no `kSwitchSound` either | same as above minus the sprite | ditto | instantaneous | ditto |
| `kTrigger` | a slot in `triggers[16]` (`room`, `object`, `index`, `timer`, `armed`) plus `hotSpots[hotNum].stillOver` | on touch: **only** the queue slot. On fire (`3 × delay` frames later): one of 12 `Trigger*`/`Toggle*` pulses, or nothing if the target family is unhandled | queue slot records the absolute `roomLink`/`objectLink`, but `FireTrigger` requires `localLink != -1` to do anything useful, so **effectively local to the 3 × 3** | **queued**, `delay × 3` frames | armed at step 6, drained at step 7 of the same or a later frame; `delay == 0` ⇒ same frame |
| `kLgTrigger` | identical to `kTrigger`; the only difference is a 48 × 48 hot spot instead of 12 × 12 | ditto | ditto | queued | ditto |
| `kSoundTrigger` | `hotSpots[hotNum].stillOver` only | nothing at all — plays `kTriggerSound` | neither; it has no link | **instantaneous** | inside step 6 |

### 8.11 Editor placement defaults

`AddNewObject` (`GliderPRO/Sources/ObjectAdd.c:475-507`) — the values a freshly placed switch has
before the author edits it:

| Field | Value | Note |
|---|---|---|
| `data.e.topLeft` | click point minus half the sprite size | centred on the click |
| `data.e.delay` | `0` | |
| `data.e.where` | `3000` for `kSoundTrigger`, else `-1` | 3000 is the first trigger `'snd '` ID |
| `data.e.who` | `255` | unlinked |
| `data.e.type` | `kOneShot` (3) for `kTrigger`/`kLgTrigger`, else `kToggle` (0) | |

`kSoundTrigger` placement is refused when `HowManySoundObjects() >= kMaxSoundTriggers`
(`ObjectAdd.c:17`, `:989-999`), i.e. one per room. `UpdateLinkControl` (`GliderPRO/Sources/Link.c`) and
the link modes `kSwitchLinkOnly = 3`, `kTriggerLinkOnly = 4`, `kTransportLinkOnly = 5`
(`GliderDefines.h:463-465`) drive the editor's "click the target" interaction.

The editor draws the three invisible types as coloured outlines (all `ObjectDraw2.c`, none reachable
in play mode):

| Routine | Drawing | Cite |
|---|---|---|
| `DrawInvisibleSwitch` | `ColorFrameRect(theRect, kIntenseGreenColor)`, colour index **225** | `ObjectDraw2.c:382`, constant at `:29` |
| `DrawTrigger` | `ColorFrameRect(theRect, kIntenseBlueColor)`, colour index **235** | `:395`, constant at `:30` |
| `DrawSoundTrigger` | `ColorFrameRect(theRect, kIntenseYellowColor)`, colour index **5** | `:408`, constant at `:20` |

### 8.12 Census over all 22 shipped houses

Every claim below was produced by decoding all 22 `GliderPRO/Houses/*.binhex` files with
`tools/probe_house.py` and walking every one of the 4 070 non-empty rooms' 24 object slots. Method:
`fk = probe_house.binhex_decode(path); h = probe_house.parse_house(fk['data'])`, then
`probe_house.extract_floor_suite(where, h['version'])` to resolve each `data.e.where`.

**Population and `data.e.type`:**

| Type | Count | `kToggle` | `kForceOn` | `kForceOff` | `kOneShot` |
|---|---|---|---|---|---|
| `kInvisSwitch` | 635 | 167 | 259 | 209 | 0 |
| `kTrigger` | 239 | 0 | 0 | 0 | **239** |
| `kKnifeSwitch` | 230 | 110 | 43 | 77 | 0 |
| `kSoundTrigger` | 122 | **122** | 0 | 0 | 0 |
| `kLightSwitch` | 113 | 61 | 30 | 22 | 0 |
| `kThermostat` | 108 | 55 | 33 | 20 | 0 |
| `kLgTrigger` | 81 | 0 | 0 | 0 | **81** |
| `kMachineSwitch` | 79 | 32 | 22 | 25 | 0 |
| `kPowerSwitch` | 78 | 62 | 6 | 10 | 0 |
| **total** | **1 685** | 609 | 393 | 363 | 320 |

`kOneShot` is used by, and only by, the two trigger types; the two trigger types use nothing else.
Since `SetObjectState` has no `case kOneShot:`, **`kOneShot` is a pure marker meaning "this is a
trigger"** — it never changes any object's state. Do not implement it as an action.

**`data.e.delay`:** zero for **every** instance of all six real switches (635 + 230 + 113 + 108 + 79 +
78 = 1 243) and for every `kSoundTrigger` (122). Only triggers use it:

| Type | `delay == 0` | nonzero | max | full nonzero histogram |
|---|---|---|---|---|
| `kTrigger` | 101 | 138 | 120 | `{2:16, 3:2, 4:2, 5:8, 8:4, 10:9, 11:2, 12:4, 13:2, 14:2, 15:4, 16:6, 18:2, 19:1, 20:13, 21:2, 22:2, 24:3, 25:4, 26:1, 28:4, 29:1, 30:12, 31:1, 33:1, 35:2, 40:2, 43:1, 45:1, 50:3, 51:1, 52:2, 54:2, 55:1, 56:2, 58:1, 59:2, 60:3, 62:1, 65:1, 75:1, 80:1, 85:1, 100:1, 120:1}` |
| `kLgTrigger` | 52 | 29 | 120 | `{2:1, 3:1, 4:1, 5:1, 6:1, 8:1, 9:1, 12:2, 15:1, 30:8, 40:3, 42:1, 50:1, 70:2, 80:1, 90:1, 120:2}` |

Latency range in frames: 0 to 360 (0 to 12 s). Modal nonzero delays are 2 (200 ms), 20 (2 s) and
30 (3 s).

**Unlinked** (`where == -1` **or** `who == 255`): `kKnifeSwitch` 5, `kLightSwitch` 2,
`kInvisSwitch` 1, `kMachineSwitch` 1, `kPowerSwitch` 1, `kThermostat` 1 = **11 total**. **Zero**
unlinked `kTrigger` or `kLgTrigger`. (`kSoundTrigger` is excluded: all 122 have `who == 255` by
design, because they use `data.e.where` as a `'snd '` ID and never link to an object.)

The two halves of that "or" behave **differently**, and the doc's earlier claim that all 11 get no
hot spot is wrong:

* **7 have `where == -1`** (`kKnifeSwitch` 4, `kPowerSwitch` 1, `kInvisSwitch` 1, `kMachineSwitch` 1)
  and therefore get **no hot spot at all** — the guard in `ObjectRects.c` is exactly
  `if (theObject.data.e.where != -1)` (`:913` for the two trigger types, `:918` for the six real
  switches). These objects are inert.
* **4 have a valid `where` but `who == 255`** (`kLightSwitch` 2, `kThermostat` 1, `kKnifeSwitch` 1)
  and therefore **do** get a `kSwitchIt` hot spot: the guard never looks at `who`. Touching one runs
  `HandleSwitches`, which reads `objectLinked = masterObjects[whoLinked].objectLink`
  (`Interactions.c:995`). That field came from `GetObjectLinked`, which returns **−1** when
  `data.e.who == 255` (`Objects.c:203-206`), so the call is
  `SetObjectState(roomLinked, -1, type, localLink)` and `SetObjectState` switches on
  `(*thisHouse)->rooms[room].objects[-1].what` with **no bounds guard** (`Objects.c:366-373`) — an
  out-of-bounds read 12 bytes before the room's object array, and a potential out-of-bounds *write*
  if that garbage `what` happens to match a switchable case. Three of the four also have
  `where == -100`, which `ExtractFloorSuite` turns into suite −1 / floor −8 — no such room, so
  `roomLinked` is **also** −1 and the expression degenerates to `rooms[-1].objects[-1]`. The fourth
  (a `kLightSwitch` in *CD Demo House*, `where == 4809` → suite 48, floor 1) *does* resolve to a real
  room, so only the object index is out of bounds there. A Go port must bounds-check **both**
  `roomLinked` and `objectLinked` before touching `rooms[]`/`objects[]`; the safe reproduction is
  "do nothing, return `changed = false`".

**Link locality** — of the 1 552 linked (non-`kSoundTrigger`) switch/trigger objects:

| Bucket | Count |
|---|---|
| target inside the loaded 3 × 3 neighbourhood | **1 384** |
| target outside the 3 × 3 (so `localLink == -1`) | **147** |
| target room does not exist (`GetRoomNumber` → −1) | **21** (`kTrigger` 20, `kLightSwitch` 1) |
| target slot exists but is empty (`what == -1`) | **5** (`kTrigger` 4, `kLgTrigger` 1) |

Same-room vs other-room, per type:

| Type | same room | other room |
|---|---|---|
| `kTrigger` | 218 | **1** |
| `kLgTrigger` | 81 | **0** |
| `kInvisSwitch` | 538 | 96 |
| `kKnifeSwitch` | 147 | 78 |
| `kLightSwitch` | 99 | 11 |
| `kMachineSwitch` | 56 | 22 |
| `kPowerSwitch` | 40 | 37 |
| `kThermostat` | 74 | 33 |

Target offset `(ΔSuite, ΔFloor)` histogram, top 12: `(0,0)` 1 253, `(1,0)` 39, `(0,1)` 35,
`(0,−1)` 29, `(−1,0)` 14, `(2,0)` 14, `(−2,0)` 11, `(5,0)` 9, `(−1,1)` 6, `(1,1)` 4, `(2,−2)` 4,
`(1,−1)` 4. **81 % of all links are same-room**, and cross-room links are overwhelmingly one step.

**Target object types** (top entries per switch type; hex is the target's `what`):

| Switch | Most common targets |
|---|---|
| `kInvisSwitch` (635) | `kInvisBlower` 0x0D ×151, `kBall` 0x76 ×71, `kLiftArea` 0x10 ×45, `kPaper` 0x25 ×42, `kFoil` 0x2A ×39, `kToaster` 0x62 ×37, `kDeluxeTrans` 0x40 ×30, `kYellowClock` 0x23 ×23, `kBalloon` 0x71 ×14, `kHelium` 0x2E ×12, `kInvisLight` 0x58 ×12, `kStereo` 0x69 ×11, `kMicrowave` 0x6A ×10 |
| `kKnifeSwitch` (230) | `kInvisBlower` ×53, `kToaster` ×34, `kShredder` 0x61 ×14, `kDeluxeTrans` ×13, `kLiftArea` ×12, `kDrip` 0x77 ×11, `kOutlet` 0x67 ×10, `kTV` 0x65 ×9, `kMicrowave` ×9, `kBalloon` ×7 |
| `kThermostat` (108) | `kInvisBlower` ×51, **`kFloorVent` 0x01 ×41**, `kLiftArea` ×7, `kSewerGrate` 0x05 ×4 |
| `kLightSwitch` (113) | **`kLightBulb` 0x52 ×18**, `kOutlet` ×13, `kInvisBlower` ×11, `kInvisLight` ×11, `kTrackLight` 0x57 ×9, `kMicrowave` ×5, `kTV` ×5, `kDecoLamp` 0x55 ×5, `kFlourescent` 0x56 ×4, `kTableLamp` 0x53 ×4, `kMacPlus` 0x63 ×4 |
| `kMachineSwitch` (79) | `kInvisBlower` ×18, `kBall` ×17, `kToaster` ×8, `kDeluxeTrans` ×6, `kFloorVent` ×5, `kRedClock` 0x21 ×4, `kCoffee` 0x66 ×3, `kShredder` ×3, `kMacPlus`/`kMicrowave`/`kLeftFan` 0x06/`kRightFan` 0x07/`kTV` ×2 each, `kCeilingVent` 0x02/`kOutlet`/`kStereo`/`kVCR` 0x68 ×1 each (78 linked; **no `kFloorBlower` target occurs anywhere in the corpus**) |
| `kPowerSwitch` (78) | `kInvisBlower` ×36, `kDrip` ×18, `kBall` ×10, **`kPowerSwitch` 0x44 ×8 (no-op)**, `kInvisLight` ×3 |
| `kTrigger` (239) | **`kInvisSwitch` 0x46 ×99**, `kGuitar` 0x64 ×41, *nonexistent room* ×20, `kFish` 0x78 ×15, `kGreaseRt` 0x28 ×13, `kToaster` ×9, `kOutlet` ×9, `kDrip` ×7, `kGreaseLf` 0x29 ×6, `kDartRt` 0x75 ×6, *empty slot* ×4, `kTrigger` 0x47 ×3, `kCopterLf` 0x72 ×2, `kCopterRt` 0x73 ×2, `kTaper` 0x08 ×1, `kShelf` 0x12 ×1, `kBalloon` ×1 |
| `kLgTrigger` (81) | **`kInvisSwitch` ×72**, `kDrip` ×3, `kDartRt` ×2, `kBalloon` ×1, `kToaster` ×1, `kGuitar` ×1, *empty slot* ×1 |

Consequences that matter for a port:

* **The dominant trigger idiom is `kTrigger`/`kLgTrigger` → `kInvisSwitch` → real target** (171 of 320
  triggers, 53 %). `TriggerSwitch` therefore must work, and it must work through the
  `dynaNum`-holds-`hotNum` overload. All 171 are same-room, so §2.5's mis-index never bites.
* **`kTrigger` → `kGuitar`** (41) and `kTrigger` → `kSoundTrigger` (0 observed) are pure sound cues.
* **Switch → switch links exist and are all dead**: `kInvisSwitch`→`kInvisSwitch` 7,
  `kPowerSwitch`→`kPowerSwitch` 8, `kInvisSwitch`→`kKnifeSwitch` 2, `kKnifeSwitch`→`kInvisSwitch` 1 =
  18. `SetObjectState` returns `changed = false` for six of them and **uninitialised** for the three
  whose target is a `kKnifeSwitch`. A Go port returning `false` reproduces the almost-certainly
  intended no-op.
* **20 `kTrigger`s and 1 `kLightSwitch` point at a room that does not exist.** Their hot spots *are*
  created (the guard is `where != -1`, not "room resolves"), so they *are* thrown. For the
  `kLightSwitch`, `SetObjectState(-1, obj, …)` reads and potentially **writes**
  `(*thisHouse)->rooms[-1]`; for the 20 triggers, `ArmTrigger` stores `room = -1` and `FireTrigger`
  takes the dead `else` branch (because `localLink == -1`) and matches nothing. A Go port must
  bounds-check `roomLinked` before touching `rooms[]`.
* **`kSoundTrigger` `'snd '` IDs** actually used: `{3000:35, 3001:18, 3002:14, 3003:6, 3004:6, 3005:5,
  3006:13, 3007:2, 3008:2, 3009:1, 3010:1, 3011:2, 3012:1, 3037:1, 3042:6, 3043:1, 3044:3, 3045:1,
  3046:1, 3058:1, 10000:2}` — i.e. mostly the 3000-3012 block, a few 3037-3058, and two houses using
  10000. Per house: Art Museum 15, CD Demo House 10, California or Bust! 3, Davis Station 5,
  Demo House 1, Grand Prix 3, ImagineHouse PRO II 2, In The Mirror 7, Leviathan 12, Nemo's Market 11,
  Rainbow's End 2, SpacePods 24, Teddy World 11, Titanic 16. Exactly one per room in all 122 cases,
  consistent with `kMaxSoundTriggers = 1`.

---

## 9. Lights and room lighting

Room lighting in Glider PRO is **binary**. A room is either lit or dark; there is no gradient, no
falloff, no per-light radius and no shadow casting. The entire model is one function that returns an
integer, `GetNumberOfLights`, and one predicate derived from it, `numLights > 0`.

### 9.1 The eight light object types

| Constant | Value | Hex | `srcRects` size | Rect derivation | Draw routine |
|---|---|---|---|---|---|
| `kCeilingLight` | 81 | `0x51` | 64 × 20 @(h=0,v=0) | `srcRects[what]`, offset by `data.f.topLeft` | `DrawSimpleLight` |
| `kLightBulb` | 82 | `0x52` | 16 × 28 @(h=0,v=20) | ditto | `DrawSimpleLight` |
| `kTableLamp` | 83 | `0x53` | 48 × 70 @(h=16,v=20) | ditto | `DrawSimpleLight` |
| `kHipLamp` | 84 | `0x54` | 72 × 276 | ditto | `DrawPictSansWhiteObject` (PICT 3994) |
| `kDecoLamp` | 85 | `0x55` | 64 × 212 | ditto | `DrawPictSansWhiteObject` (PICT 3993) |
| `kFlourescent` | 86 | `0x56` | 64 × 12 | `srcRects[what]` then **`right = data.f.length`** | `DrawFlourescent` (procedural) |
| `kTrackLight` | 87 | `0x57` | 64 × 24 | `srcRects[what]` then **`right = data.f.length`** | `DrawTrackLight` (procedural) |
| `kInvisLight` | 88 | `0x58` | 16 × 16 | `srcRects[what]`, offset by `data.f.topLeft` | nothing in play mode |

Constants at `GliderPRO/Headers/GliderDefines.h:387-394`; `srcRects` at
`GliderPRO/Sources/StructuresInit2.c:418-428`; rect derivation at
`GliderPRO/Sources/ObjectRects.c:177-198`; draw dispatch at
`GliderPRO/Sources/ObjectDrawAll.c:582-632`.

The `right = data.f.length` override for `kFlourescent` and `kTrackLight` (`ObjectRects.c:194`)
happens **before** the `QOffsetRect`, so `length` is the object's **width in pixels**, not an absolute
right edge:

```c
case kFlourescent:
case kTrackLight:
*itsRect = srcRects[who->what];          // 64 x 12 or 64 x 24
ZeroRectCorner(itsRect);                 // -> (0, 0, 64, 12)
itsRect->right = who->data.f.length;     // -> (0, 0, length, 12)
QOffsetRect(itsRect, who->data.f.topLeft.h, who->data.f.topLeft.v);
break;
```

`length` is meaningful only for those two types (and is authored 64 for `kCeilingLight` in all 160
shipped instances, which happens to equal its sprite width — see the census in §9.8).

### 9.2 On-disk record: `lightType` (`data.f`)

`GliderPRO/Headers/GliderStructs.h:54-62`, 10 bytes:

| Field | C type | Size | Union offset | Meaning |
|---|---|---|---|---|
| `topLeft` | `Point` | 4 | 0 | `{v, h}` — v first on disk |
| `length` | `short` | 2 | 4 | width in px, `kFlourescent` / `kTrackLight` only |
| `byte0` | `Byte` | 1 | 6 | **unused** — 0 in every shipped instance |
| `byte1` | `Byte` | 1 | 7 | **unused** — 0 in every shipped instance |
| `initial` | `Boolean` | 1 | 8 | authored on/off, used in **edit mode only** |
| `state` | `Boolean` | 1 | 9 | live on/off, used in **play mode only** |

Note `initial` at 8 and `state` at 9. Three arms agree on that — `lightType`, `applianceType` and
`enemyType` all put `initial` at 8 and `state` at 9 — but `blowerType` puts them at **6 and 7**, and
`bonusType` **reverses** them (`state` at 8, `initial` at 9, `GliderStructs.h:27-34`). Every union arm
has to be laid out independently; there is no shared "state byte" offset. (`furnitureType` and
`clutterType` have neither field.)

`initial` and `state` **genuinely differ** in shipped data — this is not a copy. Four of the eight
light types have different true/false counts for the two fields (§9.8), proving `state` is the
**saved live** value and `initial` the authored value. A Go port that loads a house and copies
`initial` into `state` will change gameplay for those houses.

### 9.3 `GetNumberOfLights` — the definitive algorithm

`GliderPRO/Sources/Room.c:970-1099`. The function is two nearly identical mirrored branches; the only
differences are the record it reads and which Boolean field it consults.

```
GetNumberOfLights(short where) -> short:

 if theMode == kEditMode:                                       # Room.c:976
     ROOM  = thisRoom          # <-- the ARGUMENT `where` IS IGNORED ENTIRELY
     FIELD = data.f.initial                                     # :1026
 else:                                                          # :1033
     HLock(thisHouse)                                           # :1035-1037
     ROOM  = (*thisHouse)->rooms[where]
     FIELD = data.f.state                                       # :1090

 # Step 1 -- background contribution
 switch ROOM.background:
     kGarden (2009), kSkywalk (2010), kMeadow (2012), kField (2013),
     kRoof (2014), kSky (2015), kStratosphere (2016), kStars (2017):
         count = 1                                              # :988 / :1048
     kDirt (2011):
         count = 0
         if ROOM.tiles[0..7] are ALL == 0:  count = 1           # :991-998 / :1051-1062
     default:
         count = 0                                              # :1000-1002 / :1064-1066

 # Step 2 -- object contribution, ONLY IF the background contributed nothing
 if count == 0:                                                 # :1004 / :1068
     for i in 0 .. kMaxRoomObs-1 (0..23):
         switch ROOM.objects[i].what:
             kDoorInLf (0x37), kDoorInRt (0x38),
             kWindowInLf (0x3B), kWindowInRt (0x3C),
             kWallWindow (0x86):
                 count++                       # UNCONDITIONAL  # :1010-1016 / :1074-1080
             kCeilingLight .. kInvisLight (0x51..0x58):
                 if FIELD: count++                              # :1018-1028 / :1082-1092

 if theMode != kEditMode:  HSetState(thisHouse, wasState)       # :1096
 return count                                                   # :1098
```

Five non-obvious rules a port must get exactly right:

1. **Eight background IDs are self-lighting**, and only those eight:
   `kGarden` 2009, `kSkywalk` 2010, `kMeadow` 2012, `kField` 2013, `kRoof` 2014, `kSky` 2015,
   `kStratosphere` 2016, `kStars` 2017 (`GliderPRO/Headers/GliderDefines.h:236-244`). **`kDirt` 2011
   is conditional** and every ID below 2009 or at/above `kUserBackground` is not self-lighting at all.
2. **`kDirt` is lit only when all eight tile indices are zero.** Any non-zero tile — i.e. any
   authored variation of the dirt strip — makes the room dark unless it also contains a light. That
   is an odd rule and looks like "an untouched underground room is the open-air outdoors".
3. **The object loop runs only if the background contributed 0.** A `kMeadow` room with 24 lights
   returns 1, not 25. So the *count* is not a physical quantity; only `count > 0` and `count == 0` are
   distinguishable — except by `UpdateOutletsLighting`, which stores the raw count into every outlet's
   `hVel` and then only tests `> 0` (`GliderPRO/Sources/Trip.c:235-244`, `Dynamics.c:567`). So in
   practice the value is a Boolean throughout.
4. **Doors and windows count unconditionally, and count as *lights*.** `kDoorInLf`, `kDoorInRt`,
   `kWindowInLf`, `kWindowInRt` are transport objects; `kWallWindow` is clutter. None of them has a
   `data.f` arm, none is switchable, and none appears in `SetObjectState`'s light branch. They are
   pure daylight sources: a room with an interior door or window is always lit.
5. **In edit mode the `where` argument is dead.** The edit branch reads the globally loaded
   `thisRoom` regardless. Both call sites pass `thisRoomNumber`
   (`GliderPRO/Sources/ObjectEdit.c:2362`, `GliderPRO/Sources/RoomInfo.c:104`), so this is
   decorative — but a Go port that implements one function for both modes must not accidentally make
   the argument live, or the editor's dark-room preview will differ.

Note the edit branch **does not lock `thisHouse`** because it does not touch it; the play branch
brackets the whole body in `HGetState` / `HLock` / `HSetState` (`Room.c:1035-1036`, `Room.c:1096`;
the two `ObjectEdit.c`/`RoomInfo.c` cites just above are the *call sites*, not this code) because
`thisHouse` is a relocatable Memory Manager handle. A Go port drops all of that.

#### 9.3.1 The `rooms[-1]` out-of-bounds read, verified against real bytes

`DrawLocale` calls `GetNumberOfLights(localNumbers[i])` **unconditionally** for all nine slots
(`GliderPRO/Sources/RoomGraphics.c:80`, `:84`, `:88`, `:92`, `:96`, `:100`, `:107`, `:112`, `:118`),
and `localNumbers[i]` is `kRoomIsEmpty` = **−1** for any neighbour that does not exist
(`GliderPRO/Sources/Room.c:617`, `:623-631`). So the play-mode branch reads
`thisHousePtr->rooms[-1]`, which is the 348 bytes immediately **before** `rooms[0]` — i.e. file bytes
`866 - 348 = 518` through `865`, the tail of the 866-byte house header.

Overlaying `roomType` on header bytes 518..865 gives:

`roomType` is 348 unpadded bytes (`GliderStructs.h:166-180`):

| `roomType` field | C type | Size | Offset in room | File offset for `rooms[-1]` | Real header field it lands in |
|---|---|---|---|---|---|
| `name` | `Str27` | 28 | 0 | 518-545 | tail of `trailer` (272-527) + start of `highScores.banner` (528-559) |
| `bounds` | `short` | 2 | 28 | 546-547 | `highScores.banner[18..19]` |
| `leftStart` | `Byte` | 1 | 30 | 548 | `highScores.banner[20]` |
| `rightStart` | `Byte` | 1 | 31 | 549 | `highScores.banner[21]` |
| `unusedByte` | `Byte` | 1 | 32 | 550 | `highScores.banner[22]` |
| `visited` | `Boolean` | 1 | 33 | 551 | `highScores.banner[23]` |
| **`background`** | `short` | 2 | 34 | **552-553** | `highScores.banner[24..25]` |
| **`tiles[0..7]`** | `short[8]` | 16 | 36 | **554-569** | `banner[26..31]` + `highScores.names[0][0..9]` |
| `floor` | `short` | 2 | 52 | 570-571 | `highScores.names[0]` |
| `suite` | `short` | 2 | 54 | 572-573 | `highScores.names[0]` |
| `openings` | `short` | 2 | 56 | 574-575 | `highScores.names[0]` |
| `numObjects` | `short` | 2 | 58 | 576-577 | `highScores.names[1]` |
| **`objects[i]`** | `objectType[24]` | 288 | 60 + 12·i | **578 + 12·i … 865** | `highScores` (to 819), `savedGame` (820-859), `hasGame` (860), `unusedBoolean` (861), `firstRoom` (862-863) |

`highScores` is `scoresType`: `banner` Str31 at 528-559, `names[10]` Str15 at 560-719,
`scores[10]` long at 720-759, `timeStamps[10]` at 760-799, `levels[10]` short at 800-819
(`GliderStructs.h:107-114`). `savedGame` is `gameType` at 820: `version` 820, `wasStarsLeft` 822,
`timeStamp` 824, `where` (Point) 828-831, `score` 832, … `showFoil` 859
(`GliderStructs.h:116-134`).

Measured over all 22 shipped houses (parsed with python3 from the BinHex payloads):

| Result | Houses |
|---|---|
| `GetNumberOfLights(-1)` returns **0** | 21 of 22 |
| returns **1** | 1 — **Demo House** |

Demo House returns 1 because the pseudo-room's `objects[21].what` (file offset
`578 + 12 × 21 = 830`) decodes to `0x3B` = `kWindowInLf` — and offset 830 is really
`savedGame.where.h`, the horizontal component of the saved glider position. No shipped house
produces a `background` in the 2009-2017 range at pseudo-offset 552, and no house has a stray light
type in the 24 pseudo-object slots.

This is benign in 1.0.4 for two independent reasons: `DrawRoomBackground`'s dark path is gated on
`who != kRoomIsEmpty` (`RoomGraphics.c:179`), and `DrawARoomsObjects` early-returns for
`kRoomIsEmpty` (`ObjectDrawAll.c:33-34`). So the garbage count is computed, assigned to the global
`numLights`, and then never consulted for that slot. **A bounds-checked Go port will panic here** and
must instead special-case `roomNum < 0` to return 0 — which reproduces 21 of 22 houses exactly and
diverges from Demo House only in a value that is discarded.

### 9.4 What makes a room dark, and what a dark room looks like

`numLights` is a **global** `short` (`RoomGraphics.c:32`). `DrawLocale` sets it immediately before
drawing each room and `DrawRoomBackground` / `DrawARoomsObjects` read it as ambient state — there is
no parameter. `DrawARoomsObjects` snapshots it once per call:
`isLit = (numLights > 0);` (`ObjectDrawAll.c:38`).

**Crucially, `DrawLocale` draws the central room LAST** (`RoomGraphics.c:118-121`), after the six
diagonal/vertical neighbours and after West and East. That is not cosmetic: it means that when
`DrawLocale` returns, the global `numLights` describes the **central** room, which is exactly the
invariant `RedrawRoomLighting` depends on for its `wasLit` (`:445`). A port that reorders the nine
draws breaks light toggling.

#### The dark background path

```c
if ((numLights == 0) && (who != kRoomIsEmpty))     // RoomGraphics.c:179
{
    GetGWorld(&wasCPort, &wasWorld);
    SetGWorld(backSrcMap, nil);
    PaintRect(&localRoomsDest[where]);             // :187
    SetGWorld(wasCPort, wasWorld);
    return;                                        // :190  -- NO TILES DRAWN
}
```

`PaintRect` with the GWorld's default pen (black, `patCopy`) fills the room's 512 × 322 slot in
`backSrcMap` with **solid black**. No background PICT is loaded, no tile `CopyBits` happens. Since
`DrawLocale` has already blanked the whole `backSrcMap` at `:76`, the effect is simply "leave this
room black".

#### Everything the darkness then suppresses

`DrawARoomsObjects` gates individual objects on `isLit`. The gating is **not uniform** — this table is
the authoritative list of what does and does not survive a dark room:

| Object group | Gated on `isLit`? | Cite |
|---|---|---|
| `kCeilingLight`, `kLightBulb`, `kTableLamp` | **yes** — `DrawSimpleLight` skipped | `ObjectDrawAll.c:587-588` |
| `kTrunk`, `kBooks`, `kHipLamp`, `kDecoLamp`, `kGuitar`, `kCinderBlock`, `kFlowerBox`, `kFireplace`, `kBear`, `kVase1`, `kVase2`, `kRug`, `kChimes` | **yes** | `:606-607` |
| `kCustomPict` | **yes** | `:613-614` |
| `kFlourescent` | **yes** | `:620-621` |
| `kTrackLight` | **yes** | `:627-628` |
| `kInvisLight` | n/a — draws nothing ever | `:631-632` |
| `kShredder`, `kCDs` | **yes** | `:638-639` |
| **`kToaster`** | **NO** — `DrawSimpleAppliance` runs unconditionally | `:645-647` |
| `kMacPlus` body | yes (inside `DrawMacPlus`); **screen blits unconditionally** | `ObjectDraw2.c:613-635` |
| `kTV` body | yes; **screen blits unconditionally** | `:649-688` |
| `kCoffee` body | yes; **indicator light blits unconditionally** | `:697-719` |
| `kOutlet` | **yes** — but the outlet has no overlay, so it vanishes entirely | `ObjectDrawAll.c:736-737` |
| `kVCR` body | yes; **time display blits unconditionally** | `ObjectDraw2.c:743-783` |
| `kStereo` body | yes; **power light blits unconditionally** | `:798-838` |
| `kMicrowave` body | yes; **three 16-px indicator blits unconditionally** | `:853-894` |

So a dark room containing appliances shows **floating screens, clock displays and pilot lights in the
void** — the Mac Plus's 32 × 22 screen, the TV's 64 × 49 screen, the coffee maker's 8 × 4 light, the
VCR's 16 × 4 clock, the stereo's 4 × 1 LED, three 16 × 35 microwave segments — plus a fully drawn
toaster. This is deliberate (a dark room is meant to be navigable by the glow of its appliances) and
must be reproduced blit-for-blit.

**Air-flow objects, prizes, transports, enemies, furniture and most clutter are not gated on `isLit`
at all** and draw normally in a dark room; blowers and vents are the important case, because they are
solid sprites over a black field.

#### Edit mode is different

`DrawThisRoomsObjects` (`GliderPRO/Sources/ObjectEdit.c:2347`) overlays a **50 % QuickDraw gray
pattern in `srcOr` mode** instead of a black fill:

```c
if (GetNumberOfLights(thisRoomNumber) <= 0)     // ObjectEdit.c:2362
{
    PenMode(srcOr);
    PenPat(GetQDGlobalsGray(&dummyPattern));    // :2365
    PaintRect(&backSrcRect);                    // :2366  -- WHOLE map, not one room slot
    PenNormal();
}
```

Three differences from play mode: the test is `<= 0` not `== 0`; the fill is the whole `backSrcRect`
rather than `localRoomsDest[where]`; and it is drawn **before** the objects, in `srcOr`, so the room
appears as a 50 %-stippled dark version of its normal self with all objects visible on top. The
editor's Room Info dialog also reports the state literally:

```c
if (GetNumberOfLights(thisRoomNumber) == 0)                                  // RoomInfo.c:104
    SetDialogString(theDialog, kLitUnlitText, "\p(Room Is Dark)");
else
    SetDialogString(theDialog, kLitUnlitText, "\p(Room Is Lit)");
```

A Go editor should reproduce the strings verbatim (Pascal-string literals in the original).

#### `DrawLighting` is an empty stub

```c
void DrawLighting (void)          // RoomGraphics.c:422-430
{
    if (numLights == 0)
        return;
    else
    {
        // for future construction
    }
}
```

It is called four times — after West, after East, after Central in `DrawLocale`
(`RoomGraphics.c:110`, `:115`, `:121`) and once in `RedrawRoomLighting` (`:452`) — and does nothing in
all four. There is **no** partial lighting, no light cone, no per-light illumination in shipped
Glider PRO 1.0.4. A Go port should keep the call sites (they document intent) but must not invent a
lighting model; doing so would change every room's appearance.

### 9.5 Toggling a light mid-play

Lights are the **only** object family in `SetObjectState` that changes state without touching a hot
spot or playing a sound (`GliderPRO/Sources/Objects.c:544-578`) — because a light **has no hot
spot at all**:

```c
case kCeilingLight: case kLightBulb: case kTableLamp: case kHipLamp:
case kDecoLamp: case kFlourescent: case kTrackLight: case kInvisLight:
break;                              // ObjectRects.c:930-938 -- CreateActiveRects does nothing
```

A light can therefore only be changed by something else acting on it: a switch
(`HandleSwitches`) or, in principle, a trigger — except `FireTrigger` has **no case for any light
type**, so a trigger wired to a light does nothing (§8.6). Confirmed by the census in §8.12: light
types appear as switch targets (`kLightSwitch` → `kLightBulb` ×18, `kInvisLight` ×11, `kTrackLight`
×9, `kDecoLamp` ×5, `kFlourescent` ×4, `kTableLamp` ×4; `kInvisSwitch` → `kInvisLight` ×12;
`kPowerSwitch` → `kInvisLight` ×3) but **never** as `kTrigger` or `kLgTrigger` targets in any of the
22 shipped houses.

`HandleSwitches`'s inner switch routes all eight light types to one call:

```c
case kCeilingLight: case kLightBulb: case kTableLamp: case kHipLamp:
case kDecoLamp: case kFlourescent: case kTrackLight: case kInvisLight:
RedrawRoomLighting();               // Interactions.c:1075-1084
break;
```

```
RedrawRoomLighting():                                     # RoomGraphics.c:434-461
 1. roomV  = (*thisHouse)->rooms[thisRoomNumber].floor     # :442
 2. wasLit = (numLights > 0)                               # :445  global from the LAST DrawLocale
 3. numLights = GetNumberOfLights(localNumbers[kCentralRoom])   # :446
 4. isLit  = (numLights > 0)                               # :447
 5. if wasLit == isLit:  RETURN, doing nothing             # :448
 6. DrawRoomBackground(localNumbers[kCentralRoom], kCentralRoom, roomV)   # :450
 7. DrawARoomsObjects(kCentralRoom, /*redraw=*/true)       # :451
 8. DrawLighting()                                         # :452  (no-op)
 9. UpdateOutletsLighting(localNumbers[kCentralRoom], numLights)          # :453
10. if numNeighbors > 3:  DrawFloorSupport()               # :455-456
11. RestoreWorkMap()                                       # :457  backSrcMap -> workSrcMap, full 3x3
12. AddRectToWorkRects(&localRoomsDest[kCentralRoom])      # :458  dirty only the central slot
13. shadowVisible = IsShadowVisible()                      # :459
```

Behavioural consequences, all of which a port must match:

* **Step 5 is the whole optimisation, and it is observable.** Switching off one of two lit lamps
  costs **nothing at all** — the lamp sprite is *not* erased, because nothing is redrawn. A room with
  two `kLightBulb`s where a switch turns one off looks completely unchanged. Only the transition
  0 ⇄ ≥1 re-composites.
* **`redraw = true` at step 7** is what suppresses the side effects of a re-draw: the
  `dynaNum` correlation loop is skipped (`ObjectDrawAll.c:953`), sparkles are not re-added
  (`:452`), and the toaster is not re-registered as a dinah (`:648`). But note the **seven appliance
  types whose `AddDynamicObject` is guarded only by `!redraw`** (`kMacPlus`, `kTV`, `kCoffee`,
  `kOutlet`, `kVCR`, `kStereo`, `kMicrowave` — `:664`, `:697`, `:721`, `:738`, `:754`, `:770`, `:786`)
  are likewise skipped, so existing dinah slots survive a lighting change intact. Their `dest` rects
  were computed in back-map coordinates at room-entry time and remain valid.
* **Only the central room is redrawn.** Turning on a light in the *neighbour* room you can see
  through a doorway does not update it — but `SetObjectState` did write the neighbour's authored
  `state`, so walking into that room later shows the new value. Lighting is visually stale across
  room boundaries until the next `DrawLocale`.
* **Step 9 refreshes every outlet's cached light count.** `UpdateOutletsLighting(room, nLights)`
  walks `dinahs[0..numDynamics-1]` and sets `hVel = nLights` for every `type == kOutlet` in the
  matching room (`Trip.c:235-244`). `HandleOutlet` uses `hVel > 0` to decide whether the final frame
  of a zap restores the outlet sprite or paints the rect black (`Dynamics.c:567-579`), so without
  step 9 an outlet in a newly darkened room would keep drawing its lit sprite forever.
* **Step 11/12 asymmetry.** `RestoreWorkMap` copies the *entire* `backSrcRect` into `workSrcMap`, but
  only the central room's rect is added to the dirty list. Neighbour rooms are therefore refreshed in
  `workSrcMap` yet not scheduled for blitting to the screen — correct, because nothing about them
  changed.
* **QuickDraw port leakage.** `RedrawRoomLighting` never brackets its work in
  `GetGWorld` / `SetGWorld` the way `DrawLocale` does (`RoomGraphics.c:74`, `:129`), and
  `DrawRoomBackground` does `SetPort((GrafPtr)workSrcMap)` at `:238` without restoring. So after a
  lit-path `RedrawRoomLighting` the current QuickDraw port is left as `workSrcMap`. Anything drawn
  next that relies on the ambient port (e.g. `HandleOutlet`'s bare `PaintRect` at `Dynamics.c:578`)
  is affected. A Go port with explicit render targets sidesteps this, but must pick `workSrcMap` as
  the target for those bare calls to match.

Because `GetObjectState` is only ever called for the five *visible switch* sprites
(`ObjectDrawAll.c:516`, `:529`, `:542`, `:555`, `:568`), a light's `state` has exactly two runtime
readers: `GetNumberOfLights` and — indirectly, through the switch that targets it — the switch sprite
frame. **Light sprites themselves are drawn from `isLit`, never from the light's own `state`.** So an
"off" `kLightBulb` in a room that is lit by something else is still **drawn in its lit form**. There
is no "dark lamp" sprite anywhere in the game.

### 9.6 The procedural light sprites

Six of the eight types are ordinary blits; two are drawn algorithmically because they stretch to an
arbitrary authored width.

`DrawSimpleLight` (`ObjectDraw2.c:414-420`) — one masked blit, used for `kCeilingLight`,
`kLightBulb`, `kTableLamp`:

```c
CopyMask(lightSrcMap, lightMaskMap, backSrcMap,
         &srcRects[what], &srcRects[what], theRect);
```

Source GWorld: `lightSrcRect` = `QSetRect(&lightSrcRect, 0, 0, 72, 126)` → **72 px wide × 126 px
tall** (`StructuresInit.c:501`; the source comment "9144 pixels" should be 9072), PICT
`kLightPictID` = **4004** at `kPreferredDepth`, plus a 1-bit mask from PICT **5004**
(`kLightPictID + 1000`) (`:502-508`). Sub-rects: `flourescentSrc1` 16 × 12 @(h=0,v=78),
`flourescentSrc2` 16 × 12 @(h=0,v=90), `trackLightSrc[i]` 24 × 24 @(h=24·i, v=102) for i ∈ {0,1,2}
(`:510-520`, `kNumTrackLights` = 3, `GliderDefines.h:445`).

`kHipLamp` and `kDecoLamp` go through `DrawPictSansWhiteObject` (`ObjectDraw2.c:1302-1409`), which
creates a temporary GWorld the size of `srcRects[what]`, loads PICT 3994 / 3993 into it, blits with
transfer mode **`transparent`** (white becomes see-through — no mask needed), then disposes the
GWorld. That is a `CreateOffScreenGWorld` + `DrawPicture` + `DisposeGWorld` **per draw call**. Note
`pictID` is uninitialised if `what` is not in the routine's 20-case list.

#### `DrawFlourescent` (`ObjectDraw2.c:424-496`)

Twelve horizontal `ColorLine`s spanning `left + 16` to `right − 17` (i.e. inset 16 px on the left and
17 px on the right, leaving room for the two 16 px end caps), at `top + 0` through `top + 11`, then
the two end-cap sprites. The palette is **depth-dependent**:

| Row (`top +`) | Colour name | index at `thisMac.isDepth == 4` | index at 8-bit |
|---|---|---|---|
| 0 | `grayC` | 7 | `k8LtGrayColor` = **249** |
| 1 | `gray2C` | 5 | `k8LtstGray5Color` = **248** |
| 2 | `gray2C` | 5 | 248 |
| 3 | `gray3C` | 4 | `k8LtstGray4Color` = **247** |
| 4 | `gray4C` | 1 | `k8LtstGrayColor` = **245** |
| 5 | `violetC` | 3 | `kPaleVioletColor` = **42** |
| 6 | `k8WhiteColor` | **0 (same in both)** | 0 |
| 7 | `k8WhiteColor` | 0 | 0 |
| 8 | `k8WhiteColor` | 0 | 0 |
| 9 | `k8WhiteColor` | 0 | 0 |
| 10 | `k8WhiteColor` | 0 | 0 |
| 11 | `violetC` | 3 | 42 |

Constants: depth-4 values at `ObjectDraw2.c:433-437`, 8-bit at `:441-445`, the definitions at `:19`
(`k8WhiteColor` 0), `:21` (`kPaleVioletColor` 42), `:31-34` (`k8LtstGrayColor` 245,
`k8LtstGray4Color` 247, `k8LtstGray5Color` 248, `k8LtGrayColor` 249). **`k8WhiteColor` is used
literally in both branches**, i.e. the depth-4 case still uses palette index 0 for the five white
rows — the only colour not remapped.

End caps: `flourescentSrc1` masked-blitted at `(theRect->left, theRect->top)` (`:478-485`);
`flourescentSrc2` **right-aligned** — `partRect` is zeroed, offset by `-partRect.right`, then offset
by `(theRect->right, theRect->top)`, so its right edge lands exactly on `theRect->right`
(`:487-495`).

The `ColorLine` block is bracketed by `GetGWorld` / `SetGWorld(backSrcMap, nil)` /
`SetGWorld(was…)` (`:448-449`, `:476`) but the two `CopyMask`s target `backSrcMap` explicitly and sit
**outside** that bracket.

#### `DrawTrackLight` (`ObjectDraw2.c:500-582`)

`#define kTrackLightSpacing 64` (`:502`). Six horizontal `ColorLine`s spanning `left` to
`right − 1`, at `top − 3` through `top + 2` — **note three rows are drawn above `theRect->top`**, so a
track light bleeds 3 px outside its own rect:

| Row (`top +`) | Colour name | depth-4 index | 8-bit index |
|---|---|---|---|
| −3 | `gray2C` | 8 | `k8Gray2Color` = **251** |
| −2 | `grayC` | 7 | `k8LtGrayColor` = **249** |
| −1 | `grayC` | 7 | 249 |
| 0 | `gray3C` | 4 | `k8LtstGray4Color` = **247** |
| +1 | `gray4C` | 11 | `k8DkGrayColor` = **252** |
| +2 | `gray3C` | 4 | 247 |

(depth-4 at `:511-514`, 8-bit at `:518-521`, definitions at `:32`, `:34`, `:35`, `:36`.)

Then the lamps:

```
 1. leftmost:  partRect = trackLightSrc[0] zeroed, offset to (theRect->left, theRect->top)
               which = 0;  CopyMask trackLightSrc[0]                      # :542-549
 2. rightmost: partRect = trackLightSrc[0] zeroed, offset by -right, then to
               (theRect->right, theRect->top)     # right-aligned
               which = 2;  CopyMask trackLightSrc[2]                      # :551-559
 3. howMany = ((RectWide(theRect) - RectWide(&trackLightSrc[0])) / 64) - 1 # :561-562
 4. if howMany > 0:
 5.     which = 0
 6.     spread = (RectWide(theRect) - RectWide(&trackLightSrc[0])) / (howMany + 1)   # :566
 7.     for i in 0 .. howMany-1:
 8.         partRect = trackLightSrc[0] zeroed, offset to (left, top),
                       then offset by (spread * (i+1), 0)                  # :569-572
 9.         which++;  if which >= kNumTrackLights (3): which = 0           # :573-575
10.         CopyMask trackLightSrc[which]                                  # :576-579
```

`RectWide(&trackLightSrc[0])` is 24. So for a `length` of L:
`howMany = ((L − 24) / 64) − 1` (integer division), `spread = (L − 24) / (howMany + 1)`.
Because `which++` happens **before** the blit (step 9 precedes step 10), the filler lamps cycle
`trackLightSrc[1], [2], [0], [1], [2], …` — **not** starting at `[0]`. Worked examples over the
observed `length` range (§9.8):

| `length` | `howMany` | `spread` | filler x-offsets from `left` | filler sprite indices |
|---|---|---|---|---|
| 64 | −1 | n/a | none | — (2 lamps: `[0]` left, `[2]` right) |
| 88 | 0 | n/a | none | — |
| 112 | 0 | n/a | none | — |
| 185 | 1 | 161 | 161 | `[1]` |
| 459 | 5 | 72 | 72, 144, 216, 288, 360 | `[1] [2] [0] [1] [2]` |
| 484 | 6 | 65 | 65, 130, 195, 260, 325, 390 | `[1] [2] [0] [1] [2] [0]` |
| 512 | 6 | 69 | 69, 138, 207, 276, 345, 414 | `[1] [2] [0] [1] [2] [0]` |

`DrawInvisLight` (`ObjectDraw2.c:586-595`) is `ColorFrameOval(theRect, 17)` on `backSrcMap` — an
editor-only outline in palette index 17. It is never reached in play mode because
`ObjectDrawAll.c:631-632` is a bare `break`.

### 9.7 `numLights`, `numNeighbors`, and who else reads them

| Global | Declared | Written | Read |
|---|---|---|---|
| `numLights` (`short`) | `RoomGraphics.c:32` | `DrawLocale` ×9 (`:80`-`:118`), `RedrawRoomLighting` (`:446`) | `DrawRoomBackground` (`:179`), `DrawARoomsObjects` (`:38`), `DrawLighting` (`:424`), `AddDynamicObject` for `kOutlet` (`Dynamics3.c:314`), `RedrawRoomLighting` (`:445`) |
| `numNeighbors` (`short`) | `RoomGraphics.c:32` | set from the 1-room / 3-room / 9-room display preference | `DrawLocale` (`:78`, `:105`, `:123`), `RedrawRoomLighting` (`:455`) |

`numLights` being global is the only reason `AddDynamicObject`'s outlet case can seed
`hVel = numLights` (`Dynamics3.c:314`) — it relies on being called from inside
`DrawARoomsObjects` for the room whose count is currently loaded. A Go port must thread the count
explicitly through the room-draw call, not hold it in package state, or outlets in neighbour rooms
will get the wrong cached value.

### 9.8 Census over all 22 shipped houses

Population, and `initial` vs `state`:

| Type | Count | `initial` true | `initial` false | `state` true | `state` false | Differ? |
|---|---|---|---|---|---|---|
| `kInvisLight` | 1 764 | 1 740 | 24 | 1 745 | 19 | **yes** |
| `kLightBulb` | 252 | 212 | 40 | 214 | 38 | **yes** |
| `kCeilingLight` | 160 | 157 | 3 | 158 | 2 | **yes** |
| `kFlourescent` | 115 | 108 | 7 | 110 | 5 | **yes** |
| `kTrackLight` | 81 | 73 | 8 | 73 | 8 | no |
| `kTableLamp` | 70 | 64 | 6 | 64 | 6 | no |
| `kDecoLamp` | 62 | 55 | 7 | 55 | 7 | no |
| `kHipLamp` | 26 | 22 | 4 | 22 | 4 | no |
| **total** | **2 530** | 2 431 | 99 | 2 441 | 89 | |

`kInvisLight` is **70 % of all lights** — the overwhelmingly common idiom is "make this room lit
without drawing anything", which makes sense given that no light type has an off-sprite. Ten objects
across the corpus have `state != initial`, which is the direct proof that `state` is persisted
separately (a house saved mid-game keeps the player's switch throws).

`length`:

| Type | Range | Distinct values | Modes |
|---|---|---|---|
| `kCeilingLight` | 64 in **all 160** | 1 | 64 |
| `kFlourescent` | 24 … 487 | 83 | 199 ×6, 480 ×5, 198 ×4, 200 ×3 |
| `kTrackLight` | 64 … 512 | 50 | 459 ×14, 185 ×4, 512 ×4, 484 ×3 |
| all others | 0 | 1 | 0 |

`byte0` and `byte1` are **0 in every one of the 2 530 instances** — genuinely unused fields.

Doors and windows (the unconditional light sources) and the automatic backgrounds mean many rooms are
lit with no light object at all. Distribution over the 4 070 non-empty rooms is dominated by the
`kInvisLight` idiom; only rooms whose background is not one of the nine self-lighting cases and which
contain no interior door, interior window, wall window or lit light are dark.

---

## 10. Appliance dynamics

Nine object codes become `dinahs[]` entries with per-frame behaviour. This section tabulates every
timer/frame state machine with its period and its RNG consumption. **The general dynamic-object
model — `dynaType`, the `dinahs[18]` budget, `AddDynamicObject`, `HandleDynamics`,
`RenderDynamics`, `CheckDynamicCollision`, and the enemy state machines — is documented in
`docs/analysis/enemies.md` §12 (appliance state animations at §12.7, triggers at §16.3, the
`delay → frames` conversion at §8.4).** This section does not repeat it; it is the timing and RNG
reference that §12.7 does not provide.

### 10.1 The nine appliance codes and their dinah spawn rules

`GliderDefines.h:396-409`. `AddDynamicObject` is reached from `DrawARoomsObjects`; the guard on each
call site determines whether an appliance in a *neighbour* room gets a live slot.

| Constant | Value | Hex | Gets a `dinahs[]` slot? | Guard | Cite |
|---|---|---|---|---|---|
| `kShredder` | 97 | `0x61` | **no** — static sprite + hot spot only | — | `ObjectDrawAll.c:634-640` |
| `kToaster` | 98 | `0x62` | yes | `(!redraw) && (neighbor == kCentralRoom)` | `:648` |
| `kMacPlus` | 99 | `0x63` | yes | `!redraw` — **all 9 rooms** | `:664` |
| `kGuitar` | 100 | `0x64` | **no** — sound-only object | — | `:591-608` (drawn as clutter) |
| `kTV` | 101 | `0x65` | yes | `!redraw` — **all 9 rooms** | `:697` |
| `kCoffee` | 102 | `0x66` | yes | `!redraw` — **all 9 rooms** | `:721` |
| `kOutlet` | 103 | `0x67` | yes | `!redraw` — **all 9 rooms** | `:738` |
| `kVCR` | 104 | `0x68` | yes | `!redraw` — **all 9 rooms** | `:754` |
| `kStereo` | 105 | `0x69` | yes | `!redraw` — **all 9 rooms** | `:770` |
| `kMicrowave` | 106 | `0x6A` | yes | `!redraw` — **all 9 rooms** | `:786` |
| `kCinderBlock` | 107 | `0x6B` | no | — | `:591-608` |
| `kFlowerBox` | 108 | `0x6C` | no | — | `:591-608` |
| `kCDs` | 109 | `0x6D` | no | — | `:634-640` |
| `kCustomPict` | 110 | `0x6E` | no | — | `:610-615` |

Plus `kSparkle` (45, `0x2D`), which is a *prize* code but is handled by the same dispatcher and is
guarded `(!redraw) && (neighbor == kCentralRoom)` (`:452`).

**Seven of the eight dynamic appliance types spawn for all nine rooms.** That is deliberate — their
`dest` rects are computed in back/work-map coordinates so their blits land correctly in a neighbour
room's slot, and `HandleDynamics` iterates every slot with no room filter
(`Dynamics3.c:35-104`), so a neighbour-room VCR really does blink and a neighbour-room coffee maker
really does gurgle audibly. Only `kToaster` (and `kSparkle`, and every enemy) is restricted to the
central room. With `kMaxDynamicObs = 18` (`GliderDefines.h:265`) and
`if (numDynamics >= kMaxDynamicObs) return (-1);` (`Dynamics3.c:193-194`), a 9-room view crowded with
appliances can genuinely exhaust the table and silently drop later objects — enrolment order is the
`DrawLocale` room order (NW, NE, N, SW, SE, S, W, E, Central) then slot index 0..23 within each room.

#### Coordinate spaces — a portability trap

| Dinah type | `dest` coordinate space | Why |
|---|---|---|
| `kSparkle`, `kToaster`, and every enemy | **room-relative** | `AddDynamicObject` offsets by `where->left/top` only (`Dynamics3.c:202`, `:219-221`); `RenderToast` re-adds `playOriginH/V` before blitting (`Dynamics.c:120`) |
| `kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`, `kMicrowave` | **back/work-map (screen) coordinates** | `AddDynamicObject` offsets by `where->left + playOriginH + dx`, `where->top + playOriginV + dy` (`Dynamics3.c:247-249` etc.); the handlers blit `dest` with no further offset |

The caller passes `rectA = itsRect` already offset by `(-playOriginH, -playOriginV)`
(`ObjectDrawAll.c:651`, `:667`, …), so the appliance cases simply undo that. Getting this backwards
puts every appliance indicator 256 px left and 32 px up (or down) of where it belongs.

### 10.2 Per-type `AddDynamicObject` initialisation

All from `GliderPRO/Sources/Dynamics3.c:187-389`. Every case ends with
`byte0 = (Byte)index; byte1 = 0; moving = false; active = isOn;` where `index` is the room-object slot
0..23 and `isOn` is the authored `data.g.state`.

| Type | `dest` seed | Offset from object rect | `count` | `frame` | `timer` | `hVel` | Cite |
|---|---|---|---|---|---|---|---|
| `kSparkle` | `sparkleSrc[0]` | `(left, top)` room-rel | 0 | 0 | **`RandomInt(60) + 15`** | 0 | `:199-215` |
| `kToaster` | `breadSrc[0]` centred in the toaster rect, then top-aligned to it | room-rel | initial launch velocity `v` | `data.g.delay * 3` | = `frame` | `where->top + 2` (**clip line**) | `:217-242` |
| `kMacPlus` | `plusScreen1` (32 × 22) | `+playOrigin + (10, 7)` | 0 | 0 | 0 | 0 | `:244-262` |
| `kTV` | `tvScreen1` (64 × 49) | `+playOrigin + (17, 10)` | 0 | 0 | 0 | 0 | `:264-282` |
| `kCoffee` | `coffeeLight1` (8 × 4) | `+playOrigin + (32, 57)` | 0 | 0 | `isOn ? 200 : 0` | 0 | `:284-305` |
| `kOutlet` | `outletSrc[0]` (16 × 24) | `+playOrigin + (0, 0)` | `(data.g.delay * 6) / kTicksPerFrame` = `delay * 3` | 0 | = `count` | **`numLights`** | `:307-325` |
| `kVCR` | `vcrTime1` (16 × 4) | `+playOrigin + (64, 6)` | 0 | 0 | `isOn ? 115 : 0` | 0 | `:327-348` |
| `kStereo` | `stereoLight1` (4 × 1) | `+playOrigin + (56, 20)` | 0 | 0 | 0 | 0 | `:350-368` |
| `kMicrowave` | `microOn` (16 × 35), then **`dest.right = dest.left + 48`** | `+playOrigin + (14, 13)` | 0 | 0 | 0 | 0 | `:370-389` |

The toaster's launch velocity is solved numerically at spawn time from the authored throw height
`data.g.height`:

```c
position = who->data.g.height;      // Dynamics3.c:224
velocity = 0;
do { velocity++;  position -= velocity; } while (position > 0);
dinahs[n].vVel  = -velocity;        // :232
dinahs[n].count =  velocity;        // :233
```

i.e. the smallest `v` with `v(v+1)/2 >= height`. For the shipped modal heights: 64 → v = 11
(11·12/2 = 66 ≥ 64), 18 → v = 6 (21 ≥ 18), 60 → v = 11 (66 ≥ 60), 9 → v = 4 (10 ≥ 9). The `do/while`
runs at least once, so `height <= 0` still yields `v = 1`. Sprite geometry: `breadSrc[i]` is 32 × 29
at `(h=0, v=29·i)` for i ∈ 0..5, `kNumBreadPicts = 6` (`StructuresInit.c`, `GliderDefines.h:451`).

### 10.3 The universal appliance handler shape

Six of the eight (`kMacPlus`, `kTV`, `kCoffee`, `kVCR`, `kStereo`, `kMicrowave`) share one skeleton:

```
Handle<X>(who):
    if dinahs[who].timer <= 0:  RETURN                # the gate: timer == 0 means "quiescent"
    dinahs[who].timer--
    if dinahs[who].active:                            # turning/being ON
        timer == 0   -> AddRectToWorkRects(dest)      # dirty: let RenderFrame push it to screen
        timer == 1   -> play the ON sound; CopyBits the ON overlay into backSrcMap;
                        AddRectToBackRects(dest)
        (type-specific extra thresholds)
    else:                                             # turning/being OFF
        timer == 0   -> AddRectToWorkRects(dest)
        timer == 1   -> play the OFF sound; CopyBits the OFF overlay into backSrcMap;
                        AddRectToBackRects(dest)
```

Two things make this work. First, the `timer > 0` gate means an appliance at rest costs one
comparison per frame. Second, the `timer == 1` / `timer == 0` pair is a **deliberate two-stage
pipeline** through the two dirty-rect lists (`GliderPRO/Sources/Render.c:65-99`,
`kMaxGarbageRects = 48` at `:20`, both guarded `< 47`):

| List | Array | Copies | Added by |
|---|---|---|---|
| back-to-work | `back2WorkRects[48]`, `numBack2Work` | `backSrcMap` → `workSrcMap`, clipped to `workSrcRect` | `AddRectToBackRects` (`:84`) |
| work-to-main | `work2MainRects[48]`, `numWork2Main` | `workSrcMap` → the game window, clipped to `justRoomsRect` | `AddRectToWorkRects` (`:65`) |

So on the `timer == 1` frame the new overlay is `CopyBits`ed into `backSrcMap` and queued for
propagation into `workSrcMap`; on the `timer == 0` frame the same rect is queued for the final blit
to the screen. Skipping either stage leaves the appliance's indicator either stale on screen or
correct for one frame and then overwritten by the next background restore.

**These handlers render themselves.** `RenderDynamics` (`Dynamics3.c:112-154`) has cases only for
`kToaster`, `kBalloon`, `kCopterLf/Rt`, `kDartLf/Rt`, `kBall`, `kDrip`, `kFish` — no appliance. The
appliance `CopyBits` calls live inside the `Handle*` routines, which run at **step 4 of the 12-step
frame order listed in §8.8** (`HandleDynamics`, `Play.c:475`), i.e. *before* input and interaction,
not at render time.

### 10.4 Per-appliance state machine, period and RNG

`kTicksPerFrame = 2` at 60 ticks/s ⇒ **30 fps**, so 1 frame = 33.3 ms
(`GliderDefines.h:533`). All timings below are in frames.

| Type | Reachable `timer` values | Periodic? | Period | Sounds (ID / priority) | RNG per frame |
|---|---|---|---|---|---|
| `kSparkle` | `frame`/`timer` pair, see below | **yes** | `RandomInt(240)+60` = **60…299** frames (2.0…9.97 s) | `kMysticSound` 35 / `kMysticPriority` 202 | **1 `RandomInt(240)` per sparkle** |
| `kToaster` | `frame` counts down `timer → 0`; flight length = `2·count + 1` | **yes** | idle `data.g.delay * 3` frames + flight | `kToastLaunchSound` 27 / 304, `kToastLandSound` 28 / 305 | none |
| `kMacPlus` | 0…40 | **no — one-shot** | n/a | `kMacOnSound` 29 / 401 at t=30, `kMacBeepSound` 30 / 403 at t=1, `kMacOffSound` 31 / 402 at t=1 | none |
| `kTV` | 0…4 | **no — one-shot** | n/a | `kTVOnSound` 32 / 404, `kTVOffSound` 33 / 405, both at t=1 | none |
| `kCoffee` | 0…4 and 100…399 | **yes** | `T − 100` where `T = 200 + RandomInt(200)` ⇒ **100…299** frames (3.3…10.0 s) | `kMacOnSound` 29 / 401 at t=1, `kCoffeeSound` 34 / `kCoffeePriority` 306 at t=100, `kMacOffSound` 31 / 402 | **2 sites, 1 `RandomInt(200)` per gurgle** |
| `kOutlet` | 0…`count` idle, 30→0 zapping | **yes** | `count + 30` = `3·delay + 30` frames | `kZapSound` 36 / `kZapPriority` 406 at zap start and every `timer % 5 == 0` | none |
| `kVCR` | 0…4 and 100…115 | **yes** | **15 frames** (0.5 s) blink | `kMacOnSound` at t=5 (**dead code**), `kVCRSound` 24 / `kVCRPriority` 303 at t=1, `kMacOffSound` at t=1 | none |
| `kStereo` | 0…4 | **no — one-shot** | n/a | `kMacOnSound` / `kMacOffSound` at t=1; calls `ToggleMusicWhilePlaying()` at t=0 | none |
| `kMicrowave` | 0…4 | **no — one-shot** | n/a | `kMacOnSound` / `kMacOffSound` at t=1 | none |

Sound IDs from `GliderDefines.h:69-99`; priorities from `:126-158`.

**Total RNG consumption of the whole appliance subsystem is four call sites**
(`Dynamics.c:304`, `:492`, `:506`, `Dynamics3.c:208`) — one at sparkle spawn, one per sparkle fire,
two in the coffee maker. Everything else is deterministic. A Go port sharing one PRNG stream with the
enemies must therefore consume `RandomInt` at exactly those four points, in the `HandleDynamics`
iteration order (`i` ascending over `dinahs[]`, which is the `DrawLocale` enrolment order), or every
enemy trajectory in the room will diverge.

`RandomInt` itself is `rawResult = Random(); if (rawResult < 0) rawResult *= -1;
rawResult = rawResult * range / 32768` (`GliderPRO/Sources/Utilities.c:72-82`). Its range is
therefore **0 … range** inclusive, *not* 0 … range−1: the Toolbox `Random()` returns
−32768 … 32767, and the single value −32768 negates to 32768, yielding exactly `range`. All the
"…299 frames" upper bounds quoted in this section are therefore one frame short in the
1-in-65 536 case (`RandomInt(240)` can return 240, `RandomInt(200)` can return 200). A port should
reproduce the formula rather than a `rand.Intn(range)` that can never hit `range`.

#### Detailed traces

**`kSparkle`** (`Dynamics.c:293-317`) — the only appliance-family handler with no `timer > 0` gate:

```
if active:
    if frame <= 0:                       # idle
        timer--
        if timer <= 0:
            timer = RandomInt(240) + 60          # :304
            frame = kNumSparkleModes  (= 5)      # :305   GliderDefines.h:252
            tempRect = dest;  AddSparkle(&tempRect)   # :306-307
            PlayPrioritySound(kMysticSound, kMysticPriority)
    else:                                # sparkling
        frame--                                  # :312
else:
    { }                                  # :314-316  EMPTY BLOCK -- an inactive sparkle is frozen
```

`AddSparkle` (`GliderPRO/Sources/DynamicMaps.c:169-193`) mutates the rect it is handed by
`playOriginH/V`, centres `sparkleSrc[0]` in it, and fills the first `sparkles[i].mode == -1` slot of
`kMaxSparkles = 3` (`GliderDefines.h:251`) — hence the `tempRect` copy at `:306`, which protects
`dinahs[who].dest` from being offset. **The animation itself lives in the `sparkles[]` array, not in
the dinah**; `frame` counting 5 → 0 is only a re-fire lockout of 5 frames. `kStartSparkle = 4`,
`kNumSparkleModes = 5` (`GliderDefines.h:545`, `:252`).

**`kToaster`** (`Dynamics.c:321-384`) — the only appliance with a moving sprite and a collision test:

```
if moving:                                                   # :325
    if evenFrame:  frame++;  if frame >= kNumBreadPicts (6): frame = 0    # :327-332  half-rate anim
    CheckDynamicCollision(who, &theGlider, /*deadly=*/false)              # :333-349
    VOffsetRect(&dest, vVel)                                              # :350
    whole = dest
    if vVel > 0:  whole.top    -= vVel   else:  whole.bottom -= vVel      # :352-355
    vVel++                                    # gravity, EVERY frame      # :356
    if vVel > count:                                                      # :357
        AddRectToWorkRects(whole + playOrigin);  moving = false
        frame = timer;   PlayPrioritySound(kToastLandSound, ...)          # :359-364
else:
    if active:  frame--                                                   # :369-370
    if frame <= 0:
        if active:  vVel = -count;  frame = 0;  moving = true;
                    PlayPrioritySound(kToastLaunchSound, ...)             # :373-379
        else:       frame = timer                                         # :380-381
```

Flight length: the offset applied on successive frames is `vVel` = `−count, −count+1, …, −1, 0, +1,
…, +count`, because `vVel++` happens *after* the offset and the landing test `vVel > count` first
succeeds on the frame after `vVel == count` was used. So a launch lasts exactly **`2·count + 1`
frames** and lands back at the launch height (the sum `−c … +c` is 0). Idle interval is
`timer = data.g.delay * 3` frames — the **same
tenths-of-a-second scale as the trigger delay** (§8.6). Note the inactive branch keeps resetting
`frame = timer` every frame, so a toaster switched on mid-idle launches after a full `3·delay` wait.
`hVel` is repurposed as a **vertical clip line** (`where->top + 2`), used by `RenderToast` to hide
the slice while it is still inside the slot (`Dynamics.c:122-127`).

**`kMacPlus`** (`Dynamics.c:388-424`) — pure one-shot, driven entirely by `ToggleMacPlus`:

```
ToggleMacPlus:  active = !active;  timer = active ? 40 : 10      # Trip.c:29-36
ON  trace: 40 → … → 30 (kMacOnSound) → … → 1 (kMacBeepSound + plusScreen2 blit) → 0 (dirty), stop
OFF trace: 10 → …             → 1 (kMacOffSound + plusScreen1 blit) → 0 (dirty), stop
```

40 frames = 1.33 s from switch-throw to beep; the `kMacOnSound` fires 30 frames (1 s) into it. There
is **no self-restart**, so an authored-on Mac Plus (`timer = 0` at spawn) shows `plusScreen2` from
`DrawMacPlus` and never makes a sound until switched off and on.

**`kTV`** (`Dynamics.c:428-478`) — same shape, `ToggleTV` sets `timer = 4` always (`Trip.c:57`). Both
the `timer == 0` dirty and the `timer == 1` `tvScreen2` blit are **skipped entirely** for the one TV
that owns the QuickTime movie: `if (thisMac.hasQT && hasMovie && tvInRoom && who == tvWithMovieNumber)`
(`Dynamics.c:437-438`, `Dynamics.c:449-450`) leaves an empty block, because `MoviesTask` owns that
rectangle. The `kTVOnSound` at `Dynamics.c:448` is **outside** the guard, so even the movie TV still
makes the sound.
`tvWithMovieNumber` is set to the dinah index in `DrawARoomsObjects` (`ObjectDrawAll.c:707-708`)
and reset by `DrawLocale` (`RoomGraphics.c:60`).

**`kCoffee`** (`Dynamics.c:482-524`) — the only self-restarting appliance:

```
spawn on:  timer = 200
ON periodic loop:      T → … → 100:  kCoffeeSound;  timer = 200 + RandomInt(200)   # :503-507
switched ON:           4 → 3 → 2 → 1 (kMacOnSound + coffeeLight2 blit)
                         → 0 (dirty; timer = 200 + RandomInt(200))                 # :489-493
switched OFF:          4 → … → 1 (kMacOffSound + coffeeLight1) → 0 (dirty), stop
```

Because the reset at `timer == 100` jumps back to 200…399, the gurgle period is `T − 100`
∈ **[100, 299] frames = [3.33 s, 9.97 s]** and `timer` never descends below 100 while running — so
the `timer == 1` / `timer == 0` branches are reachable **only** via `ToggleCoffee`'s `timer = 4`
(`Trip.c:62-66`). An authored-on coffee maker gurgles first at 200 − 100 = 100 frames (3.33 s) after
room entry, deterministically.

**`kOutlet`** (`Dynamics.c:528-599`) — two-phase, keyed on `position` not `timer`:

```
if position != 0:                     # ZAPPING
    timer--                                                        # :532
    CheckDynamicCollision(who, &theGlider, /*deadly=*/true)        # :534-550   LETHAL
    if timer <= 0:  frame = 0;  position = 0;  timer = count       # :552-557
    else:           if (timer % 5) == 0: kZapSound                 # :560-561
                    frame++;  if frame >= kNumOutletPicts (4): frame = 1   # :562-564
    if (position != 0) || (hVel > 0):
        CopyBits applianceSrcMap outletSrc[frame] -> workSrcMap dest       # :569-573
    else:
        PaintRect(&dest)               # bare PaintRect -- AMBIENT PORT     # :578
    AddRectToWorkRects(&dest)                                      # :580
else:                                 # IDLE
    if active:  timer--                                            # :584-585
    if timer <= 0:
        if active:  position = 1;  timer = kLengthOfZap (30);  kZapSound    # :589-594
        else:       timer = count                                  # :595-596
```

`kLengthOfZap = 30` (`GliderDefines.h:546`), `kNumOutletPicts = 4` (`:446`), `outletSrc[i]` is
16 × 24 at `(h=64, v=22 + 24·i)` for i ∈ 0..3. Cycle: `count = 3·delay` idle frames, then exactly 30
zap frames, i.e. period `3·delay + 30`. `kZapSound` plays once at zap start and again at
timer = 25, 20, 15, 10, 5 — **six times per zap**. `frame` cycles 1, 2, 3, 1, 2, 3, … (never 0
during a zap) at full rate.

The `hVel > 0` test only matters on the **last** frame of a zap, when `position` has just been set to
0: in a lit room (`hVel = numLights > 0`) `outletSrc[0]` — the plain, un-zapping outlet — is blitted
back; in a dark room `PaintRect` blanks the rect. That is the entire purpose of caching `numLights`
into `hVel`, and it is why `RedrawRoomLighting` must call `UpdateOutletsLighting` (§9.5). Note the
`PaintRect` at `:578` uses whatever the ambient QuickDraw port is (the `SetPort` is commented out at
`:577`); in practice that is `workSrcMap`, matching the `CopyBits` destination.

**`kVCR`** (`Dynamics.c:603-667`) — a 15-frame blink loop plus a one-shot switch response:

```
spawn on: timer = 115
loop:  115 → … → 101:  blit vcrTime2 if frame==0 else vcrTime1;  AddRectToBackRects   # :632-650
           → 100:  AddRectToWorkRects;  timer = 115;  frame = 1 - frame               # :626-631
switched ON:  4 → 3 → 2 → 1 (kVCRSound + vcrTime2 blit) → 0 (dirty; timer = 115)      # :610-625
switched OFF: 4 → … → 1 (kMacOffSound + vcrTime1) → 0 (dirty), stop                   # :654-664
```

Reachable `timer` values are 0…4 and 100…115 only. **`timer == 5` → `kMacOnSound` (`:615-616`) is
therefore unreachable dead code**: `ToggleVCR` sets 4 and the loop never descends below 100. Blink
period = 115 − 100 = **15 frames = 0.5 s**, alternating `vcrTime2` (the "12:00" display) and
`vcrTime1` (blank) via `frame = 1 - frame`. `vcrTime1` / `vcrTime2` are 16 × 4 at `(h=64, v=179)` and
`(h=64, v=183)`.

**`kStereo`** (`Dynamics.c:671-711`) — one-shot, but with a global side effect at `timer == 0`:

```
ON:  4 → 3 → 2 → 1 (kMacOnSound + stereoLight2) → 0 (dirty + ToggleMusicWhilePlaying())   # :678-691
OFF: 4 → …       → 1 (kMacOffSound + stereoLight1) → 0 (dirty + ToggleMusicWhilePlaying()) # :695-708
```

`ToggleMusicWhilePlaying()` is called in **both** arms, so the music actually toggles on each
completed transition. This is the third of three places the stereo's state lives:
`SetObjectState` toggles the global `isPlayMusicGame` directly (`Objects.c:584-588`),
`ToggleStereos` flips `dinahs[].active` but **only if `timer == 0`** (`Trip.c:85-92`), and
`HandleStereo` calls `ToggleMusicWhilePlaying` 4 frames later. Consequence: **throwing a stereo
switch twice within 4 frames desynchronises `isPlayMusicGame` from the stereo's `active` and from
the actual music playback.** `DrawStereo` compounds it by drawing the **global**, not the object's
state: `DrawStereo(&itsRect, isPlayMusicGame, isLit)` (`ObjectDrawAll.c:769`) while
`AddDynamicObject` is seeded from `thisObject.data.g.state` (`:775`). Also note
`data.g.state` is authored `false` in 31 of 36 shipped stereos (`(initial, state) == (1, 0)`), i.e.
the corpus expects the global to win.

**`kMicrowave`** (`Dynamics.c:715-775`) — one-shot; the only appliance whose overlay is **three**
blits:

```
ON:  4 → … → 1:  kMacOnSound;  dest' = dest with right = left+16;
                 CopyBits microOn -> dest';  QOffsetRect(+16,0);  CopyBits;  +16;  CopyBits;
                 AddRectToBackRects(dest)                                    # :726-746
     → 0: dirty
OFF: same with microOff                                                      # :752-772
```

`dest` was widened to 48 px at spawn (`Dynamics3.c:376`) precisely so three 16-px-wide copies of the
35-px-tall `microOn` / `microOff` sprites tile it. `microOn` is 16 × 35 at `(h=64, v=222)`,
`microOff` 16 × 35 at `(h=64, v=187)`. `data.g.byte0` is the only appliance byte0 in use (values
0…7 across 56 shipped microwaves) — and **nothing in the source reads it**; grep finds no
`data.g.byte0` reader. It is an authored-but-ignored field.

### 10.5 `Toggle*` and `Trigger*` entry points, and `ZeroDinahs`

See §8.6 for the full table. The appliance-relevant asymmetries:

| Appliance | Switch (`Toggle*`) does | Trigger (`FireTrigger`) does |
|---|---|---|
| `kShredder` | flips `data.g.state` and `hotSpots[hotNum].isOn` inside `SetObjectState` (`Objects.c:624-625`) | **nothing** — no case in `FireTrigger` |
| `kToaster` | `ToggleToaster`: flips `active` only | `TriggerToast`: launches one slice now if `active`, else `frame = timer` |
| `kMacPlus` | `ToggleMacPlus`: flips `active`, `timer = 40 / 10` | **nothing** |
| `kGuitar` | plays `kChordSound` (`Interactions.c:1097-1099`) | plays `kChordSound` (`Triggers.c:139-141`) |
| `kTV` | `ToggleTV`: flips `active`, drives QuickTime, `timer = 4` | **nothing** |
| `kCoffee` | `ToggleCoffee`: flips `active`, `timer = 4` | plays `kCoffeeSound` only — does **not** touch the dinah |
| `kOutlet` | `ToggleOutlet`: flips `active` only | `TriggerOutlet`: starts a 30-frame zap now if `active`, else `timer = count` |
| `kVCR` | `ToggleVCR`: flips `active`, `timer = 4` | **nothing** |
| `kStereo` | `ToggleStereos`: flips `active` + `timer = 4`, **only if `timer == 0`** | **nothing** |
| `kMicrowave` | `ToggleMicrowave`: flips `active`, `timer = 4` | **nothing** |

`ZeroDinahs` (`Dynamics3.c:160-180`) is called by `DrawLocale` (`RoomGraphics.c:52`) and clears
`type` (to `kObjectIsEmpty` = −1), `dest`, `whole`, `hVel`, `vVel`, `count`, `frame`, `timer`,
`position`, `room`, `byte0`, `active`, and `numDynamics = 0`. It **does not clear `byte1` or
`moving`** (`dynaType` has 14 members; only 12 are reset). Every `AddDynamicObject` case assigns both
explicitly, so no stale value is ever read — but a Go port zeroing the whole struct is both safer and
behaviourally identical.

`dynaType` layout (`GliderStructs.h:310-320`), 36 bytes on 68k/PPC, in-memory only (never
serialised):

| Field | Type | Size | Offset |
|---|---|---|---|
| `dest` | `Rect` | 8 | 0 |
| `whole` | `Rect` | 8 | 8 |
| `hVel` | `short` | 2 | 16 |
| `vVel` | `short` | 2 | 18 |
| `type` | `short` | 2 | 20 |
| `count` | `short` | 2 | 22 |
| `frame` | `short` | 2 | 24 |
| `timer` | `short` | 2 | 26 |
| `position` | `short` | 2 | 28 |
| `room` | `short` | 2 | 30 |
| `byte0` | `Byte` | 1 | 32 |
| `byte1` | `Byte` | 1 | 33 |
| `moving` | `Boolean` | 1 | 34 |
| `active` | `Boolean` | 1 | 35 |

Field overloading, per type (this is the single most confusing thing about `dinahs[]`):

| Field | Normal meaning | Overloaded as |
|---|---|---|
| `hVel` | horizontal velocity | `kToaster`: vertical clip line. `kOutlet`: cached `numLights` |
| `count` | period / limit | `kToaster`: launch speed. `kOutlet`: idle period in frames. `kBalloon`/enemies: `delay * 3` |
| `frame` | animation frame | `kToaster` **while idle**: countdown to launch. `kSparkle`: 5-frame re-fire lockout. `kVCR`: which of two blink phases |
| `timer` | countdown | `kToaster`: the idle period to reload `frame` from |
| `position` | — | `kOutlet`: 0 = idle, ≠ 0 = zapping |
| `byte0` | — | the room-object slot index 0..23 |
| `byte1` | — | **never read for any appliance** |

### 10.6 Appliance sprite sheet geometry

`applianceSrcRect` = `QSetRect(&applianceSrcRect, 0, 0, 80, 269)` → **80 px wide × 269 px tall**
(`StructuresInit.c:538`; source comment "21600 pixels" should be 21 520), PICT
`kAppliancePictID` = **4005** plus a 1-bit mask from PICT **5005** (`:539-545`).
`toastSrcRect` = 32 × 174, PICT `kToastPictID` = **4009** + mask 5009 (`:547-554`).
`shredSrcRect` = 40 × 35, PICT `kShreddedPictID` = **4010** (`:556-559`).

Overlay sub-rects (all `StructuresInit.c`, all `(width × height) @ (h, v)` within
`applianceSrcMap`):

| Sub-rect | Size | Position | Used by |
|---|---|---|---|
| `plusScreen1` | 32 × 22 | (48, 127) | Mac Plus off |
| `plusScreen2` | 32 × 22 | (48, 149) | Mac Plus on |
| `tvScreen1` | 64 × 49 | (0, 171) | TV off |
| `tvScreen2` | 64 × 49 | (0, 220) | TV on |
| `coffeeLight1` | 8 × 4 | (72, 171) | coffee off |
| `coffeeLight2` | 8 × 4 | (72, 175) | coffee on |
| `outletSrc[0..3]` | 16 × 24 | (64, 22 + 24·i) | outlet idle / 3 zap frames |
| `vcrTime1` | 16 × 4 | (64, 179) | VCR display blank |
| `vcrTime2` | 16 × 4 | (64, 183) | VCR display "12:00" |
| `stereoLight1` | 4 × 1 | (68, 171) | stereo LED off |
| `stereoLight2` | 4 × 1 | (68, 172) | stereo LED on |
| `microOff` | 16 × 35 | (64, 187) | microwave window dark |
| `microOn` | 16 × 35 | (64, 222) | microwave window lit |
| `breadSrc[0..5]` | 32 × 29 | (0, 29·i) | toast flight animation, in `toastSrcMap` |

Four appliance **bodies** are not in `applianceSrcMap` at all — they are loaded from individual PICTs
into a temporary GWorld on **every single draw call** and disposed immediately:

| Appliance | Body PICT | Mask PICT | Routine |
|---|---|---|---|
| `kTV` | `kTVPictID` **3992** | `kTVMaskID` **3912** | `DrawTV` (`ObjectDraw2.c:649-672`) |
| `kVCR` | `kVCRPictID` **3990** | `kVCRMaskID` **3913** | `DrawVCR` (`:743-767`) |
| `kStereo` | `kStereoPictID` **3989** | `kStereoMaskID` **3914** | `DrawStereo` (`:798-822`) |
| `kMicrowave` | `kMicrowavePictID` **3971** | `kMicrowaveMaskID` **3915** | `DrawMicrowave` (`:853-877`) |

and three more via `DrawPictSansWhiteObject` with `transparent` mode and no mask:
`kGuitar` PICT **3991**, `kCinderBlock` PICT **3960**, `kFlowerBox` PICT **3959**
(`ObjectDraw2.c:1359-1369`, `:1396-1407`). A Go port should decode all of these **once** at
start-up into textures; the per-draw create/load/dispose is pure Resource-Manager overhead with no
semantic content.

`kCustomPict` is special: its PICT ID is stored in **`data.g.height`**, and `GetObjectRect` uses the
PICT's own `picFrame` as the object's bounds, **rewriting `data.g.height` to 10000 in memory if the
PICT is missing**:

```c
case kCustomPict:                                     // ObjectRects.c:220
thePict = GetPicture(who->data.g.height);
if (thePict == nil)
{
    who->data.g.height = 10000;                       // :224  MUTATES the object
    *itsRect = srcRects[who->what];                   // fallback 72 x 34
}
else
{
    HLock((Handle)thePict);
    *itsRect = (*thePict)->picFrame;                  // :230
    HUnlock((Handle)thePict);
}
ZeroRectCorner(itsRect);
QOffsetRect(itsRect, who->data.g.topLeft.h, who->data.g.topLeft.v);
```

There is **no `ReleaseResource`**, so every `GetObjectRect` on a `kCustomPict` leaks a resource
handle. `DrawCustPictSansWhite(pictID, theRect)` (`ObjectDraw2.c:1412-1436`) then loads it again into
a temporary GWorld and blits `transparent`.

### 10.7 Authored-field census over all 22 shipped houses

Population:

| Type | Count |
|---|---|
| `kCustomPict` | 4 782 |
| `kToaster` | 140 |
| `kOutlet` | 100 |
| `kTV` | 80 |
| `kMacPlus` | 74 |
| `kCDs` | 63 |
| `kMicrowave` | 56 |
| `kShredder` | 50 |
| `kCinderBlock` | 43 |
| `kCoffee` | 38 |
| `kStereo` | 36 |
| `kFlowerBox` | 35 |
| `kGuitar` | 29 |
| `kVCR` | 26 |
| **total** | **5 552** |

`data.g.delay` — **zero for every appliance type except two**:

| Type | Distinct values | Range | Top values |
|---|---|---|---|
| `kToaster` | 27 | 0 … 255 | 0 ×26, 10 ×17, 11 ×8, 15 ×8, 8 ×8, 19 ×7, 16 ×6, 1 ×6 |
| `kOutlet` | 27 | 0 … 240 | 10 ×22, 240 ×9, 15 ×8, 12 ×6, 19 ×5, 13 ×5, 20 ×5, 8 ×5 |
| all others | 1 | 0 | 0 |

So the reachable frame periods in shipped data are: toaster idle `3 × delay` ∈ {0, 3, …, 765} frames
(0 … 25.5 s), outlet cycle `3 × delay + 30` ∈ {30, …, 750} frames (1 … 25 s). `delay = 0` means a
toaster that launches every frame it is not in flight, and an outlet that zaps continuously with a
0-frame gap — both occur in shipped houses.

`data.g.height` — zero for every appliance except two:

| Type | Distinct values | Range | Top values |
|---|---|---|---|
| `kToaster` (throw height, px) | 81 | 9 … 277 | 64 ×19, 18 ×10, 60 ×6 |
| `kCustomPict` (PICT resource ID) | 149 | 10 000 … 12 351 | 10000 ×2 179, 10002 ×319, 10001 ×296, 11003 ×126, 10020 ×100 |
| all others | 1 | 0 | 0 |

`data.g.byte0` — **zero everywhere except `kMicrowave`**: `{7: 22, 6: 11, 2: 9, 0: 6, 4: 3, 1: 3,
3: 2}`. As noted in §10.4, no code reads it. The editor seeds it to 7 for a new microwave and 0
otherwise (`GliderPRO/Sources/ObjectAdd.c:589-644`).

`(initial, state)` pairs:

| Type | `(1,1)` | `(1,0)` | `(0,1)` | `(0,0)` |
|---|---|---|---|---|
| `kGuitar` | 29 (100 %) | 0 | 0 | 0 |
| `kCinderBlock` | 43 (100 %) | 0 | 0 | 0 |
| `kFlowerBox` | 35 (100 %) | 0 | 0 | 0 |
| `kCDs` | 63 (100 %) | 0 | 0 | 0 |
| `kCustomPict` | 4 782 (100 %) | 0 | 0 | 0 |
| `kStereo` (36) | 1 | **31** | 3 | 1 |
| `kToaster` (140) | 118 | 4 | 0 | 18 |
| `kOutlet` (100) | 77 | 1 | 0 | 22 |
| `kTV` (80) | 44 | 0 | 5 | 31 |
| `kMacPlus` (74) | 36 | 0 | 5 | 33 |
| `kMicrowave` (56) | 43 | 0 | 2 | 11 |
| `kShredder` (50) | 32 | 5 | 1 | 12 |
| `kCoffee` (38) | 25 | 0 | 0 | 13 |
| `kVCR` (26) | 12 | 0 | 3 | 11 |

The five types that are `(1,1)` in 100 % of instances are exactly the five that are **not
switchable** in `SetObjectState` (`kGuitar` at `Objects.c:580-582`, `kCinderBlock`/`kFlowerBox`/
`kCDs`/`kCustomPict` at `:630-635`, all `changed = false`) — their state bytes are inert. `kStereo`'s
overwhelming `(1, 0)` reflects that its live state is really the global `isPlayMusicGame`, which
`DrawStereo` reads instead of `data.g.state` (`ObjectDrawAll.c:769`).

Editor defaults (`GliderPRO/Sources/ObjectAdd.c:589-644`): `kToaster` gets
`height = 64, delay = 10 + RandomInt(10)`; `kOutlet` gets `height = 0, delay = 10 + RandomInt(10)`;
`kCustomPict` gets `height = 10000, delay = 0`; every other appliance gets `height = 0, delay = 0`;
`kMicrowave` gets `byte0 = 7`, all others `byte0 = 0`; all get `initial = state = true`.

---

## Open questions

1. **Is `SetObjectState`'s missing `case kKnifeSwitch:` deliberate?** The switch-type branch
   (`GliderPRO/Sources/Objects.c:533-542`) lists `kLightSwitch`, `kMachineSwitch`, `kThermostat`,
   `kPowerSwitch`, `kInvisSwitch`, `kTrigger`, `kLgTrigger`, `kSoundTrigger` — but **not**
   `kKnifeSwitch` (0x45). There is also no `default:` (`:695`), and `changed` is uninitialised
   (`:369`), so a switch pointed at a knife switch returns whatever was on the stack. **Exactly two
   shipped links do that** (§8.12: `kInvisSwitch` → `kKnifeSwitch` ×2; the reverse,
   `kKnifeSwitch` → `kInvisSwitch` ×1, *is* covered by the `kInvisSwitch` label). I cannot tell
   whether the author intended knife switches to be un-switchable (plausible — they are the "master
   power" idiom) or simply forgot the label; both readings produce identical behaviour for the other
   1 683 switch/trigger objects.
   **Consequence if I am wrong:** if the intent was `changed = true`, a switch-throws-a-switch chain
   would repaint the target knife switch's sprite; if `false`, it would not. Since the target's
   `data.e` has no state field either way, no gameplay state changes — only whether
   `CopyRectBackToWork` / `AddRectToWorkRects` run for the *thrower's* rect, which they do
   unconditionally at `Interactions.c:1033-1034`. So I believe this is unobservable, but I did not
   build the game to confirm.

2. **What was `triggers[].what` for?** `ArmTrigger` writes it from
   `masterObjects[triggers[where].object].theObject.what` (`GliderPRO/Sources/Triggers.c:50`), mixing
   a *room-object* number (`object`, 0..23) with a *master-list* index (0..215). Nothing ever reads
   the field. My reading is that an earlier design dispatched on `what` inside `HandleTriggers`
   instead of re-deriving it in `FireTrigger` (`:109`), and the write survived the refactor.
   **Consequence if I am wrong:** none for behaviour — but a Go port that faithfully reproduces the
   write will read out of bounds for any `object > numMasterObjects`, so it must be dropped, and I am
   asserting that dropping it is safe on the strength of a whole-source grep for `.what` on a
   `trigType`.

3. **Is `FireTrigger`'s `else` branch (`Triggers.c:174-193`) reachable at all?** It is guarded by
   `if (masterObjects[triggerIs].localLink != -1)` at `:107` — the `else` therefore runs when the
   trigger's *own* target is not in the 3×3 neighbourhood, and it then indexes
   `masterObjects[triggeredIs]` where `triggeredIs` was computed from that same
   `localLink == -1`, i.e. `masterObjects[-1]`. In the 22 shipped houses every armed trigger's target
   is local (§8.12: 218/219 `kTrigger` and 81/81 `kLgTrigger` links are same-room; the single
   cross-room trigger link points at an empty slot in a room outside the 3×3, so it *would* take this
   branch — but only if the player is standing in room 1 of Land of Illusion and touches object 16).
   I could not run the game to see what `masterObjects[-1]` contains on a real Mac.
   **Consequence if I am wrong:** one shipped house has a reachable wild read. A Go port must
   early-return when `localLink == -1`, which is the only safe interpretation, and which changes
   behaviour for at most that one trigger.

4. **Does any shipped house rely on `GetNumberOfLights(-1)` returning 1?** §9.3.1 shows Demo House's
   pseudo-room decodes `kWindowInLf` from `savedGame.where.h`, so `numLights` is briefly 1 for a
   nonexistent neighbour. I traced both consumers (`DrawRoomBackground`'s `who != kRoomIsEmpty`
   guard, `DrawARoomsObjects`'s early return) and both discard it, so I am confident it is inert. But
   I did not audit every reader of the global `numLights` for a path that runs *between* the
   `GetNumberOfLights` call and the next assignment.
   **Consequence if I am wrong:** a Go port returning 0 for `rooms[-1]` would render Demo House
   differently in 9-room mode. I judge this a non-risk, but it is the only place where a house file's
   *header* bytes influence rendering.

5. **Why does `kDirt` (2011) require all eight tiles to be zero to be self-lighting?**
   (`GliderPRO/Sources/Room.c:991-998`, `:1051-1062`.) Every other outdoor background is
   unconditional. My reading is "an unmodified dirt strip is open sky; a decorated one is a tunnel",
   but the tile indices for `kDirt` are not documented anywhere in the source and I did not decode
   PICT 2011 to see what tile 0 looks like versus tiles 1..7.
   **Consequence if I am wrong:** underground rooms would be lit or dark contrary to the original.
   The rule itself is unambiguous in code, so a port can reproduce it without understanding it — the
   open question is only *why*.

6. **Is `data.f.byte0` / `data.f.byte1` truly unused?** Both are 0 in all 2 530 shipped lights
   (§9.8) and I found no reader. `lightType` is the only union arm with two spare bytes, which
   suggests they were reserved for a per-light radius/intensity that `DrawLighting`
   (`GliderPRO/Sources/RoomGraphics.c:422-430`, an empty stub) would have consumed.
   **Consequence if I am wrong:** none — 0 in every shipped house means any port that ignores them
   round-trips perfectly.

7. **Is `data.g.byte0` on `kMicrowave` read anywhere?** It is the only non-zero `byte0` in the whole
   appliance corpus (values 0…7 across 56 microwaves, §10.7) and the editor seeds it to 7
   (`GliderPRO/Sources/ObjectAdd.c:589-644`), which strongly implies it was meaningful — a duration,
   a power level, or the number of lit segments. I grepped the whole source for `data.g.byte0` and
   found only writers. `HandleMicrowave` hardcodes three 16-px segments (`Dynamics.c:730-744`) and
   `HandleMicrowaveAction` (`Interactions.c:1159-1194`) does not consult it.
   **Consequence if I am wrong:** microwave animation width or drain rate would be authored per
   object rather than fixed. Since the shipped modal value is 7 and the code draws 3 segments
   regardless, no port can be wrong by ignoring it — but the *editor* should preserve it verbatim.

8. **What is the intended relationship between `isPlayMusicGame`, `dinahs[].active` and
   `data.g.state` for `kStereo`?** Three separate places hold "is the stereo on":
   `SetObjectState` toggles the global unconditionally (`Objects.c:584-588`), `ToggleStereos` flips
   `active` **only if `timer == 0`** (`Trip.c:85-92`), and `HandleStereo` calls
   `ToggleMusicWhilePlaying()` four frames later (`Dynamics.c:680`, `:697`). `DrawStereo` draws the
   *global* (`ObjectDrawAll.c:769`) while `AddDynamicObject` is seeded from `data.g.state` (`:775`),
   and 31 of 36 shipped stereos are authored `(initial, state) = (1, 0)`. I believe the global is
   canonical and the dinah is a 4-frame animation trigger, but I could not determine whether the
   `timer == 0` guard is a debounce (intentional) or an oversight.
   **Consequence if I am wrong:** throwing a stereo switch twice within 4 frames desynchronises the
   music from the sprite. A port that removes the guard, or that makes `SetObjectState` conditional,
   changes audible behaviour in a way a player could notice.

9. **Is the `kMacOnSound` at `HandleVCR`'s `timer == 5` (`Dynamics.c:615-616`) truly dead?** The
   reachable `timer` values are 0…4 (from `ToggleVCR`'s `timer = 4`) and 100…115 (from the blink
   reload), so 5 is never hit. I enumerated every writer of a `kVCR` dinah's `timer`
   (`Dynamics3.c:338-341`, `Trip.c:77-81`, `Dynamics.c:613`, `:629`) and found no path to 5.
   **Consequence if I am wrong:** a port that faithfully includes the branch plays one extra
   `kMacOnSound` per switch-on, which would be audible.

10. **Does `numLights` describe the central room at every point where something reads it?**
    `DrawLocale` draws the central room last precisely so this holds (`RoomGraphics.c:118-121`), and
    `RedrawRoomLighting`'s `wasLit` depends on it (`:445`). But `AddDynamicObject`'s `kOutlet` case
    also reads it (`Dynamics3.c:314`) while enrolling objects for *neighbour* rooms, so a neighbour
    room's outlets cache the *neighbour's* count — which is correct, because that call happens inside
    that neighbour's `DrawARoomsObjects`. I believe the invariant is "`numLights` is always the count
    for the room currently being drawn, and after `DrawLocale` returns that is the central room", but
    the source states this nowhere and it is an easy thing to break.
    **Consequence if I am wrong:** an outlet in a dark neighbour room would paint its rect instead of
    restoring its sprite after a zap, or vice versa.

11. **Are there houses with more than 18 dynamic objects visible in 9-room mode?** Seven of the eight
    dynamic appliance types spawn for all nine rooms (§10.1) and the budget is
    `kMaxDynamicObs = 18` with a silent `return (-1)` (`Dynamics3.c:193-194`). The census gives
    totals per house but I did not compute, for each room of each house, the sum over the 3×3
    neighbourhood of (appliance dinahs) + (central-room hazards + toasters + sparkles).
    **Consequence if I am wrong:** if the budget *is* exceeded anywhere, a faithful Go port must
    reproduce the *enrolment order* (NW, NE, N, SW, SE, S, W, E, Central, then slot 0..23) and the
    silent drop, or different objects will go missing.

12. **What did `kLgTrigger`'s dead `case kLgTrigger:` action label mean?**
    `HandleHotSpotCollision` has `case kTriggerIt: case kLgTrigger: ArmTrigger(who);`
    (`Interactions.c:1351-1354`), i.e. an *object code* (0x48 = 72) used where an *action code* is
    expected. `kTriggerIt` is 12 and no action code is 72, so the label is unreachable — but it means
    a hot spot with `action == 72` would arm a trigger. `AddActiveRect` is only ever called with
    named `k*It` actions, so nothing produces 72.
    **Consequence if I am wrong:** none observable; a port should simply drop the label. I include it
    because it is the sort of thing a mechanical translation reproduces and then trips over.

---

## Porting notes

### Get these right first — they are the highest-leverage fidelity risks

1. **Switches are synchronous; triggers are queued — and both resolve within the same frame.**
   `HandleInteraction` (`GliderPRO/Sources/Play.c:482`) calls `HandleSwitches` immediately for a
   `kSwitchIt` hot spot, but only *arms* a `triggers[]` slot for `kTriggerIt`. `HandleTriggers` runs
   two calls later at `Play.c:484`. Therefore a trigger with `delay == 0` fires **in the same frame it
   was touched, but after all switch effects**, and a trigger with `delay == d` fires exactly `3·d`
   frames later (`timer = data.e.delay * 3`, `Triggers.c:49`). 101 of 239 shipped `kTrigger`s and 52
   of 81 `kLgTrigger`s have `delay == 0` (§8.12), so this ordering is load-bearing, not academic.

2. **The frame order is fixed and observable.** `HandleDynamics` → input → `HandleInteraction` →
   `HandleTriggers` → `HandleBands` → `HandleGlider` → `RenderFrame` → scoreboard
   (`Play.c:475-495`). Appliance animations advance *before* the player moves; trigger effects land
   *after* interaction but *before* the glider integrates. Reordering any of these changes what a
   trigger-launched toaster slice can hit on its first frame.

3. **`newState` is a global, not a return value.** `SetObjectState` writes
   `newState` (`Objects.c:78`) and `HandleSwitches` reads it back to pick the switch sprite frame
   (`Interactions.c:1005-1031`). Threading it as a return value is fine, but do **not** assume
   `SetObjectState`'s `Boolean` return (`changed`) carries the state — it carries "did anything
   change", and it is uninitialised on four paths (§8.5, note 1 above).

4. **A room is lit iff `GetNumberOfLights(room) > 0`, and that function is not "count the lamps".**
   Eight background IDs self-light unconditionally; `kDirt` self-lights only with all eight tiles
   zero; **five transport/clutter codes (`kDoorInLf` 0x37, `kDoorInRt` 0x38, `kWindowInLf` 0x3B,
   `kWindowInRt` 0x3C, `kWallWindow` 0x86) count unconditionally as lights**; and the 24-slot object
   loop runs **only if the background contributed nothing** (`Room.c:1004`, `:1068`). Get any of
   those four rules wrong and whole rooms flip between lit and pitch black.

5. **`GetNumberOfLights` reads `data.f.initial` in edit mode and `data.f.state` in play mode**, and
   the two genuinely differ in shipped data for `kInvisLight`, `kLightBulb`, `kCeilingLight` and
   `kFlourescent` (§9.8). Never copy `initial` into `state` on load; never read `initial` at runtime.

6. **A dark room is a flat black `PaintRect`, not a dimmed room** (`RoomGraphics.c:187-190`), and
   `DrawLighting` is an empty stub (`:422-430`). Do not invent a lighting model. But note the dark
   room is **not** empty: appliance *overlays* (Mac Plus screen, TV screen, coffee light, VCR clock,
   stereo LED, three microwave segments) blit unconditionally, and the **toaster body draws even when
   dark** (`ObjectDrawAll.c:645-647`). Blowers, prizes, transports, enemies and furniture are not
   gated on `isLit` at all. Table in §9.4 is the authoritative list.

7. **`RedrawRoomLighting` short-circuits unless the room crosses the lit/dark boundary**
   (`RoomGraphics.c:448`). Turning off one of two lit lamps produces **zero** visual change — the
   lamp sprite is not erased. And light sprites are chosen from the *room's* `isLit`, never from the
   individual light's `state` (`ObjectDrawAll.c:582-632`), so there is no "off lamp" sprite anywhere.

8. **Lights have no hot spot** (`ObjectRects.c:930-938`) and `FireTrigger` has **no case for any
   light type**. A light can only be changed by a switch. Wiring a `kTrigger` to a light is a no-op,
   and no shipped house does it.

9. **`timer` is a countdown with a `> 0` gate, and `timer == 0` means "quiescent", not "fire now".**
   Every appliance handler is `if (timer <= 0) return; timer--; …`. The `timer == 1` frame commits the
   overlay to `backSrcMap` and queues a back-to-work copy; the `timer == 0` frame queues the
   work-to-main blit (§10.3). Collapse the two stages and indicators will flicker or lag one frame.

10. **Three appliances are periodic and five are one-shot, and only two consume RNG.** Periodic:
    `kCoffee` (gurgle every `T − 100` frames where `T = 200 + RandomInt(200)`, i.e. **100…299
    frames**), `kOutlet` (cycle `3·delay + 30` frames exactly), `kVCR` (blink every **15** frames),
    plus `kToaster` (idle `3·delay` + flight `2·count + 1`) and `kSparkle` (`RandomInt(240) + 60`).
    One-shot: `kMacPlus` (40 on / 10 off), `kTV`, `kStereo`, `kMicrowave` (all 4). **Exactly four
    `RandomInt` call sites exist in the whole dynamic subsystem** — `Dynamics.c:304`, `:492`, `:506`,
    `Dynamics3.c:208` — so a shared PRNG stream must be consumed at precisely those points, in
    `dinahs[]` index order, or enemy trajectories diverge.

11. **Seven appliance types spawn dinahs for all nine rooms; only `kToaster`, `kSparkle` and the
    enemies are central-room-only.** The guard is literally `!redraw` versus
    `(!redraw) && (neighbor == kCentralRoom)` (`ObjectDrawAll.c:664` vs `:648`). Combined with
    `kMaxDynamicObs = 18` and a silent `return (-1)` on overflow, this is a real budget you must
    model, and neighbour-room appliances really do animate and **make sound**
    (`HandleDynamics` has no room filter, `Dynamics3.c:35-104`).

12. **Two coordinate spaces in `dinahs[].dest`.** The seven all-rooms appliances store
    **back/work-map (screen) coordinates** — `AddDynamicObject` adds `playOriginH/V` at spawn. The
    toaster, the sparkle and every enemy store **room-relative** coordinates and their renderers add
    `playOriginH/V` per frame (`Dynamics.c:119-120`). Mixing these up displaces every appliance
    indicator by the play origin.

13. **`dinahs[]` field overloading is not cosmetic.** `hVel` is a *clip line* for the toaster and a
    *cached light count* for the outlet; `count` is a launch velocity for the toaster and an idle
    period for the outlet; `frame` is a countdown-to-launch for an idle toaster and a re-fire lockout
    for a sparkle; `position` is the outlet's phase flag. Table in §10.5. Keep the canonical field
    names so a porter can diff against `Dynamics.c`.

14. **`UpdateOutletsLighting` is mandatory.** After any lit⇄dark transition, every `kOutlet` dinah in
    the room must have `hVel = numLights` refreshed (`Trip.c:235-244`, called from
    `RoomGraphics.c:453`). `HandleOutlet` uses `hVel > 0` on the final frame of a zap to decide
    between restoring `outletSrc[0]` and painting the rect black (`Dynamics.c:567-579`). Skip it and
    an outlet in a newly darkened room glows forever.

15. **`kZapSound` plays six times per zap**: once at launch and once each at `timer` = 25, 20, 15,
    10, 5 (`Dynamics.c:560-561`). The zap sprite cycles `outletSrc[1] → [2] → [3] → [1] → …`, never
    `[0]`, at full frame rate.

16. **`data.g.delay` is in tenths of a second → `× 3` frames, and `data.g.height` means two different
    things.** For `kToaster` `height` is a throw height in pixels, converted at spawn to a launch
    velocity by the smallest `v` with `v(v+1)/2 ≥ height` (`Dynamics3.c:224-231`). For `kCustomPict`
    `height` is a **PICT resource ID** (10 000…12 351 in shipped data) — and `GetObjectRect`
    **mutates it to 10000** if the resource is missing (`ObjectRects.c:224`) and leaks the handle
    every call.

17. **`kFlourescent` and `kTrackLight` take their width from `data.f.length`, applied to
    `right` before the offset** (`ObjectRects.c:194`). `kTrackLight`'s filler lamps cycle
    `trackLightSrc[1], [2], [0], [1], …` because `which++` precedes the blit
    (`ObjectDraw2.c:573-579`), and it draws three of its six pinstripe rows **above** its own
    `top` (`:527-538`). Worked `howMany`/`spread` examples for the observed `length` range are in
    §9.6.

18. **Both procedural light routines have a depth-4 palette and an 8-bit palette, and `k8WhiteColor`
    (0) is used literally in both.** Full index tables in §9.6. Getting the 8-bit indices wrong (249,
    248, 247, 245, 42 for the fluorescent; 251, 249, 247, 252 for the track light) changes the
    on-screen colour of every fluorescent and track light in the game.

19. **Guard every `-1` index.** The original dereferences `rooms[-1]` (via
    `GetNumberOfLights(localNumbers[i])` for missing neighbours, and via `SetObjectState` when
    `roomLinked == -1` — 21 shipped links), **`objects[-1]`** (via `SetObjectState` when
    `objectLinked == -1`, which happens for the **4 shipped switches that have a valid `where` but
    `who == 255`** — they still get a `kSwitchIt` hot spot because the guard at `ObjectRects.c:918`
    only tests `where != -1`; see §8.12), `hotSpots[-1]` (via `TriggerSwitch(dynaNum)` when
    `dynaNum == -1`), and `masterObjects[-1]` (`Triggers.c:174-193`, `:50`). C read garbage; Go will
    panic. §8.5, §8.12 and §9.3.1 give the safe substitutions and what each one costs in fidelity.

20. **Uninitialised locals that reach real code.** `changed` in `SetObjectState` (`Objects.c:369`);
    `bounds` in `HandleSwitches` (`Interactions.c:987`) which is then passed to `AddSparkle` at
    `:1050`; `pictID` in `DrawPictSansWhiteObject` (`ObjectDraw2.c:1302-1394`, no `default:`).
    Initialise them to `false` / an empty rect / a sentinel and document the divergence — do not
    silently pick a value that happens to look right.

21. **The `data.a.state` / `data.c.points` union collision at `Objects.c:465` is real but latent.**
    The prize force-off path writes union byte **7**, which is the low byte of a big-endian `points`
    `short`, not `bonusType.state` at byte 8. Shipped `points` values 100/300/500 would become
    0/256/256. It is unobservable in 1.0.4 only because the sole reader
    (`Interactions.c:920`) reads `points` *before* calling `SetObjectState` and the inner switch has
    `case kInvisBonus: break;`. A port with a tagged union cannot reproduce the bug at all, which is
    fine — but do not "fix" it into `bonusType.state`, or invisible bonuses will stop respawning.

22. **`data.e.delay` is 0 in every one of the 1 365 shipped real switches and sound triggers**; only
    `kTrigger` and `kLgTrigger` use it. So the delayed-action queue exists solely for triggers, and
    `ArmTrigger`'s `stillOver` check (`Triggers.c:38-39`) is the only debounce.

23. **`kSoundTrigger` (0x49) is a fourth thing.** It is excluded from `ObjectIsLinkSwitch`
    (`Objects.c:236-250`), from `GetRoomLinked` and from `GetObjectLinked`; its `data.e.where` is a
    `'snd '` resource ID (3000…3012, 3037…3058, 10000 in shipped data — §8.12), its `who` is 255 in
    all 122 instances, its hot spot is a **hardcoded 48 × 48** rect rather than
    `srcRects[kSoundTrigger]`'s 32 × 32 (`ObjectRects.c:924`), and it is only created if
    `LoadTriggerSound` succeeds (`:926`). `kTriggerSound` = 63 = `kMaxSounds - 1`, i.e. the trigger
    sound occupies the last of the 64 sound slots, and at most one `kTriggerPriority` sound may be
    playing at a time (`Sound.c:47-51`).

24. **Do not reorder `DrawLocale`'s nine room draws.** Central must be last, because the global
    `numLights` is left describing it and `RedrawRoomLighting`'s `wasLit` depends on that
    (`RoomGraphics.c:118-121`, `:445`).

### Structural advice

25. **Replace the Toolbox GWorld churn with textures loaded once.** `kTV`, `kVCR`, `kStereo`,
    `kMicrowave`, `kHipLamp`, `kDecoLamp`, `kCustomPict` and the whole `DrawPictSansWhiteObject`
    family create an offscreen GWorld, load a PICT, `CopyMask`/`CopyBits`, and `DisposeGWorld` **on
    every draw call** (`ObjectDraw2.c:649-672`, `:743-767`, `:798-822`, `:853-877`, `:1397-1407`,
    `:1424-1433`). That is Resource-Manager overhead with no semantic content. Decode every PICT once
    at start-up. The mask PICTs are `body + 1000` for the sprite sheets (4004 → 5004, 4005 → 5005)
    but individually numbered for the per-object ones (`kTVPictID` 3992 / `kTVMaskID` 3912,
    `kVCRPictID` 3990 / `kVCRMaskID` 3913, `kStereoPictID` 3989 / `kStereoMaskID` 3914,
    `kMicrowavePictID` 3971 / `kMicrowaveMaskID` 3915) — and `DrawPictSansWhiteObject` uses **no mask
    at all**, relying on QuickDraw's `transparent` transfer mode to knock out white. In a Go port
    that means "treat pure white as the colour key" for exactly those PICTs.

26. **Switches are drawn `srcCopy` and are therefore opaque.** `InitSwitches` loads
    `kSwitchPictID` = 4003 into a 32 × 104 GWorld with **no mask**
    (`StructuresInit.c:455-458`; the source comment "3360 pixels" is arithmetically wrong — 32 × 104
    = 3 328), and the five `Draw*Switch` routines blit `srcCopy` into `backSrcMap`
    (`ObjectDraw2.c:301-371`). Lights, by contrast, use `CopyMask` with the 1-bit PICT 5004.

27. **Model `hotNum`, `dynaNum`, `localLink`, `roomLink` and `objectLink` as explicit optionals.**
    They are `-1`-sentinel `short`s that index four different tables, and `dynaNum` is **deliberately
    overloaded** for switches: `DrawARoomsObjects` stores a switch's `hotNum` into its `dynaNum`
    (`ObjectDrawAll.c:518`, `:531`, `:544`, `:557`, `:570`, and — crucially, since it is the dominant
    trigger target — `:574` for `kInvisSwitch`) so that `TriggerSwitch(dynaNum)` can do
    `HandleSwitches(&hotSpots[who])` (`Trip.c:146-149`). That is **not** the bug it looks like — see
    the retraction in §8.4.1. It *is* mis-indexed for switches in neighbour rooms, but no shipped
    house reaches that path.

28. **`objectLink` must be a signed 16-bit type.** On disk `data.e.who` / `data.d.who` is a `Byte`
    with 255 meaning "unlinked"; `GetObjectLinked` converts it to `-1` in memory
    (`Objects.c:189`, `:203`). A `uint8` field will silently break the "unlinked" test; a `byte` that
    round-trips 255 but is compared against `-1` will break it differently.

29. **`ZeroDinahs` does not clear `byte1` or `moving`** (`Dynamics3.c:160-180`, 12 of `dynaType`'s 14
    members). Every `AddDynamicObject` case assigns both, so nothing reads a stale value — zeroing
    the whole struct in Go is both safer and behaviourally identical.

30. **QuickDraw port/GWorld leakage is pervasive and occasionally load-bearing.**
    `DrawRoomBackground` does `SetPort((GrafPtr)workSrcMap)` at `RoomGraphics.c:238` and never
    restores it; `RedrawRoomLighting` has no `GetGWorld`/`SetGWorld` bracket at all;
    `DrawThisRoomsObjects` returns early at `ObjectEdit.c:2358-2359` after
    `SetGWorld(backSrcMap, nil)`; `HandleOutlet`'s `PaintRect` at `Dynamics.c:578` targets the
    *ambient* port because the `SetPort` is commented out at `:577`. With explicit render targets a Go
    port sidesteps all of this, but for `HandleOutlet` it must pick `workSrcMap` — the same
    destination as the `CopyBits` two lines above — to match.

31. **`LoadTriggerSound` `HLock`s the `'snd '` handle and never `HUnlock`s it**
    (`Sound.c:288`), and `GetObjectRect`'s `kCustomPict` case never `ReleaseResource`s the PICT
    (`ObjectRects.c:220-237`). Both are leaks a Go port simply does not have; note them so nobody
    "reproduces" them.

32. **`CreateActiveRects` returns the *last* hot spot it created** (`ObjectRects.c:296-303` onward,
    `hotSpotNumber` reassigned per `AddActiveRect`), so a multi-rect object records the wrong
    `hotNum`. Correct for every switch (one rect each) but a trap if you add objects.

33. **Do not "fix" the shipped data.** `data.e.delay == 0` on triggers, `kToaster`/`kOutlet`
    `delay == 0` (a continuously launching toaster, a continuously zapping outlet — both occur),
    `state != initial` on ten lights, 21 switch links pointing at nonexistent rooms, 5 pointing at
    empty slots, 18 switch→switch links that are no-ops, `kStereo` authored `(1, 0)` in 31 of 36
    cases, `kMicrowave.byte0` values nothing reads — all of it is in the 22 shipped houses and all of
    it must round-trip byte-for-byte.

34. **Test vectors worth building first:** (a) a room whose only light source is a `kWindowInRt`
    (must be lit with zero light objects); (b) a `kDirt` room with `tiles = {0,0,0,0,0,0,0,1}` (must
    be **dark**) versus all-zero (lit); (c) two `kLightBulb`s in one room, a `kLightSwitch` on one —
    throwing it must produce **no visible change**; (d) the same with one bulb — must produce a full
    black⇄lit re-composite plus an outlet `hVel` refresh; (e) a `kTrigger` with `delay = 0` wired to a
    `kToaster` (slice must launch in the touch frame, after switch effects); (f) the same with
    `delay = 20` (must launch exactly 60 frames later); (g) a `kFlourescent` with `length = 487` and a
    `kTrackLight` with `length = 459` (5 fillers at 72 px spacing, sprites `[1] [2] [0] [1] [2]`);
    (h) a `kToaster` with `height = 277` (velocity 24, 49-frame flight — matches
    `docs/analysis/enemies.md` §12.1); (i) a `kCoffee` authored on (first gurgle at exactly frame
    100, then 100…299-frame intervals); (j) a `kOutlet` with `delay = 240` (750-frame cycle, 30 zap
    frames, six `kZapSound`s); (k) a 9-room view with 20 appliances (must drop the last two silently);
    (l) `Demo House` in 9-room mode on a room with fewer than four neighbours (exercises
    `GetNumberOfLights(-1)`).

35. **Cross-references.** The general dynamic-object model, the enemy state machines, the lethal
    appliance bodies and the `kMicrowaveIt` drain are in `docs/analysis/enemies.md` — §12 for
    appliances, §12.7 for state animations, §16.1-16.3 for `SetObjectState` / `HandleSwitches` /
    triggers from the hazard side, §18 for the object census. Room adjacency, `GetNeighborRoomNumber`,
    `MergeFloorSuite`/`ExtractFloorSuite` and the house file format are in
    `docs/analysis/house-format.md`. The nine-room composite and the dirty-rect pipeline are in
    `docs/analysis/rendering.md` and `docs/analysis/object-draw-all.md`. Hot-spot dispatch is in
    `docs/analysis/interactions.md`. The editor's placement defaults and dialogs are in
    `docs/analysis/editor.md` and `docs/analysis/editor-object-manipulation.md`. Sprite-sheet and
    PICT inventories are in `docs/analysis/graphics-assets.md`; the sound tables are in
    `docs/analysis/audio.md`; RNG determinism is in `docs/analysis/determinism.md`.
