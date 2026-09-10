# Glider PRO 1.0.4 — Collision Detection and Player/Object Interaction Resolution

## Scope

This document reverse-specifies the *interaction layer* of Glider PRO 1.0.4: everything
that decides "did the paper glider touch something, and what happens as a result".
Concretely it covers:

* the rectangle lists that exist at runtime (`srcRects`, `hotSpots`, `masterObjects`,
  `dinahs`, `tempManholes`) and exactly how each is built per room;
* the overlap primitives (`SectGlider`, `GliderInRect`, `GliderHitTop`, `BounceGlider`,
  `IsRectLeftOfRect`) and how they differ from QuickDraw's `SectRect`;
* the per-frame test order, down to which function mutates which field and when;
* the 28 hot-spot action codes and the complete behaviour of each;
* how the glider is pushed out of solids (floor, ceiling, both walls, sloped roof);
* one-way surfaces (`kDirt` tunnel tiles, `kRoof` diagonals, `kManhole`, doors/windows);
* pickup logic for every bonus (points, rubber bands, batteries, helium, aluminium foil,
  stars, grease, paper/extra life);
* every damage and death condition;
* every room-transition trigger (walking off an edge, doors, windows, stairs, ducts,
  mailboxes, transporters) and its exact geometric precondition;
* the two-player "escape handshake" protocol that gates all of the above;
* the off-by-ones, integer-division truncations and genuine upstream bugs that a Go port
  will otherwise silently "fix" and thereby diverge.

**Not in scope** (documented elsewhere): rendering/`CopyBits` blitting, sound resource
loading, the house editor, scoreboard drawing, QuickTime movie playback.

A note on one file in the assigned list: **`Sources/Coordinates.c` is not collision code.**
It implements the level editor's floating "Coordinates" windoid (globals `coordWindowRect`,
`coordWindow`, `isCoordH`, `isCoordV`, `coordH`, `coordV`, `coordD`, `isCoordOpen` at
`GliderPRO/Sources/Coordinates.c:16`), and its two entry points `SetCoordinateHVD()`
(`GliderPRO/Sources/Coordinates.c:29`) and `DeltaCoordinateD()`
(`GliderPRO/Sources/Coordinates.c:49`) are both wrapped in `#ifndef COMPILEDEMO` and do
nothing but store three shorts and call `UpdateCoordWindow()`
(`GliderPRO/Sources/Coordinates.c:57`). The name is a false friend; the *game* coordinate
system is defined by constants in `Headers/GliderDefines.h` and by
`OffsetRectRoomRelative()` in `Sources/ObjectRects.c`. Section 1 documents the real
coordinate system; a Go port should not port `Coordinates.c` at all.

## Sources read

All source files under `GliderPRO/Sources/` and `GliderPRO/Headers/` are **classic Mac
text**: lines are terminated by a bare CR (`0x0D`), so `wc -l` reports 0 and naive readers
see one enormous line. Every file was converted with `tr '\r' '\n'` into a scratch
directory before reading, and **all line numbers cited in this document are line numbers in
the CR→LF converted copy** (which equals the 1-based line number any modern editor would
show). The files also contain Mac Roman high bytes (e.g. `0xA5` for the `÷` in
`// ÷ 64` comments), so GNU `grep` must be given `-a` or it classifies them as binary.

Read in full (converted line counts in parentheses):

| File | Lines | Why |
|---|---|---|
| `GliderPRO/Sources/Interactions.c` | 1777 | the core file: all overlap tests, all 28 actions, all escapes |
| `GliderPRO/Sources/ObjectRects.c` | 1188 | builds every hot-spot rect from every object type |
| `GliderPRO/Sources/Dynamics.c` | 776 | `CheckDynamicCollision`, toast, appliances, outlets |
| `GliderPRO/Sources/Dynamics2.c` | 589 | balloon, copter, dart, ball, drip, fish collisions |
| `GliderPRO/Sources/Dynamics3.c` | 555 | `HandleDynamics` dispatch, `RenderDynamics`, `ZeroDinahs` |
| `GliderPRO/Sources/Player.c` | 1605 | `MoveGlider` (the physics), `HandleGlider` mode dispatch, `OffsetGlider` |
| `GliderPRO/Sources/Play.c` | 821 | `PlayGame` — the authoritative per-frame order |
| `GliderPRO/Sources/Modes.c` | 640 | every `StartGlider*` / `FlagGlider*` mode transition |
| `GliderPRO/Sources/Transit.c` | 558 | `MoveRoomToRoom`, duct/mail/transport plumbing |
| `GliderPRO/Sources/Objects.c` | 1001 | `ListAllLocalObjects`, `IsThisValid`, `SetObjectState` |
| `GliderPRO/Sources/Room.c` | 1206 | `DetermineRoomOpenings`, `IsRoomAStructure`, `GetOriginalBounding` |
| `GliderPRO/Sources/RectUtils.c` | 318 | `QSetRect`, `QOffsetRect`, `ZeroRectCorner`, `IsRectLeftOfRect` |
| `GliderPRO/Sources/RubberBands.c` | 318 | band physics and band-vs-hot-spot collision |
| `GliderPRO/Sources/Grease.c` | 302 | grease spill, which *mutates hot spots at runtime* |
| `GliderPRO/Sources/Triggers.c` | 205 | delayed-action triggers |
| `GliderPRO/Sources/Input.c` | 398 | where `hDesiredVel`, `tipped`, band firing come from |
| `GliderPRO/Sources/Coordinates.c` | 196 | confirmed editor-only (see Scope) |
| `GliderPRO/Sources/StructuresInit2.c` | 476 | `CreatePointers` allocation sizes, `InitSrcRects` literal table |
| `GliderPRO/Sources/RoomGraphics.c` | 462 | `DrawLocale`, `ReadyLevel` — the room-setup pipeline |
| `GliderPRO/Sources/HouseIO.c` | 708 | how house bytes reach memory (raw `FSRead`, no swapping) |
| `GliderPRO/Sources/RoomInfo.c` | 907 | the *write* side of `roomType.bounds`, which decodes the bit layout |
| `GliderPRO/Headers/GliderDefines.h` | 625 | every constant cited here |
| `GliderPRO/Headers/GliderStructs.h` | 347 | every struct laid out here |

Read in part: `GliderPRO/Sources/Render.c` (frame composition, 620-680),
`GliderPRO/Sources/ObjectDrawAll.c` (grease creation and `dynaNum` assignment, 360-410 and
935-965), `GliderPRO/Sources/Trip.c` (all `Toggle*`/`Trigger*` entry points).

Binary evidence was produced by decoding `GliderPRO/Houses/*.binhex` (BinHex 4.0) with a
purpose-written decoder and parsing the resulting data forks with a struct parser; all
observed byte values quoted in section 20 are real output, not inference.

---

## 1. Coordinate systems, units and rect conventions

### 1.1 Two coordinate spaces

Glider PRO uses two 2-D integer spaces and converts between them by adding or subtracting
one global offset pair.

| Space | Origin | Used by |
|---|---|---|
| **Room space** ("local") | top-left of the current room's 512x322 play field | `glider.dest`, `hotSpots[].bounds`, `bands[].dest`, enemy `dinahs[].dest`, all limit constants |
| **Screen space** ("global"/work-map space) | top-left of the offscreen work map | blitting, `savedMaps`, appliance/outlet `dinahs[].dest`, grease art |

The conversion is `screen = room + (playOriginH, playOriginV)`. `QOffsetRect(&r,
playOriginH, playOriginV)` converts room→screen; the negated form converts back. Two
concrete examples of the round trip:

* `HandleSwitches()` takes the room-space hot rect and offsets it to screen space purely so
  it can draw: `newRect = who->bounds; QOffsetRect(&newRect, playOriginH, playOriginV);`
  (`GliderPRO/Sources/Interactions.c:1001`).
* `HandleGrease()` receives screen-space art coordinates and converts them *back* to room
  space before writing them into a hot spot:
  `QOffsetRect(&src, -playOriginH, -playOriginV);`
  (`GliderPRO/Sources/Grease.c:66`).

**Collision is always done in room space.** Every rect stored in `hotSpots[]` is room
space. The only reason `CheckDynamicCollision()` has a `doOffset` parameter
(`GliderPRO/Sources/Dynamics.c:34`) is that some `dinahs[]` entries are stored in screen
space and must be converted down before testing (section 17).

### 1.2 Room geometry constants

All from `GliderPRO/Headers/GliderDefines.h`:

| Constant | Value | Hex | Line | Meaning |
|---|---:|---|---|---|
| `kRoomWide` | 512 | 0x0200 | 499 | room width in pixels |
| `kTileWide` | 64 | 0x40 | 497 | width of one background tile |
| `kNumTiles` | 8 | 0x08 | 496 | tiles across a room (8 x 64 = 512) |
| `kTileHigh` | 322 | 0x142 | 498 | room height in pixels |
| `kVertLocalOffset` | 322 | 0x142 | 501 | vertical distance between vertically adjacent rooms (== `kTileHigh`) |
| `kFloorSupportTall` | 44 | 0x2C | 500 | height of the drawn floor-support band (graphics only) |

| Limit | Value | Line | Meaning |
|---|---:|---|---|
| `kCeilingLimit` | 8 | 503 | glider `dest.top` is clamped to this when the ceiling is solid |
| `kFloorLimit` | 312 | 504 | glider `dest.bottom` is clamped to this when the floor is solid |
| `kRoofLimit` | 122 | 505 | below this `dest.bottom`, `kRoof` rooms start testing the diagonal |
| `kLeftWallLimit` | 12 | 506 | glider `dest.left` clamp for a closed left wall |
| `kRightWallLimit` | 500 | 508 | glider `dest.right` clamp for a closed right wall |
| `kNoCeilingLimit` | -10 | 510 | `dest.top` must go *below* this to escape upward |
| `kNoFloorLimit` | 332 | 511 | `dest.bottom` must exceed this to escape downward |
| `kNoLeftWallLimit` | -24 | 507 | `dest.left` must go below this to escape left |
| `kNoRightWallLimit` | 536 | 509 | `dest.right` must exceed this to escape right |

Note the asymmetry that matters for a port: the *inner* limits are not centred in the room
(8 vs 322-312 = 10 at the bottom; 12 vs 512-500 = 12 at the sides), and the gap between each
inner limit and its *outer* "escape" limit is a different number on every side
(8 - (-10) = 18 at the top, 12 - (-24) = 36 at the left, 536 - 500 = 36 at the right,
332 - 312 = 20 at the bottom). They are simply four independent magic numbers; do not try
to derive them.

### 1.3 Glider geometry constants

| Constant | Value | Line |
|---|---:|---|
| `kGliderWide` | 48 | `GliderPRO/Headers/GliderDefines.h:548` |
| `kGliderHigh` | 20 | 549 |
| `kHalfGliderWide` | 24 | 550 |
| `kGliderBurningHigh` | 26 | 551 |
| `kShadowHigh` | 9 | 552 |
| `kShadowTop` | 306 | 553 |
| `kGliderStartsDown` | 32 | 569 |

The glider's `dest` is therefore normally 48 x 20. A burning glider is 48 x 26, anchored
at the *bottom*: `dest.top = dest.bottom - kGliderBurningHigh`
(`GliderPRO/Sources/Modes.c:413`). The shadow is a separate 48 x 9 rect pinned at
`top = kShadowTop = 306`, i.e. `(306, left, 315, left+48)`
(`GliderPRO/Sources/Play.c:354`).

### 1.4 Rect and Point conventions

`Rect` is the QuickDraw struct: **four big-endian `short`s in the order `top, left,
bottom, right`**. `Point` is **`v` then `h`** (vertical first). This is the single most
common porting mistake and it is *verified* against real house files in section 20.4.

The project's own rect helpers reverse the argument order relative to the struct:

```c
void QSetRect (Rect *theRect, short l, short t, short r, short b)   // RectUtils.c:210
```

so `QSetRect(&r, 0, -distance, kFloorColumnWide, 0)` means
`left=0, top=-distance, right=kFloorColumnWide, bottom=0`. Every `QSetRect` call quoted in
this document uses that `(l, t, r, b)` order. `QOffsetRect(theRect, h, v)`
(`GliderPRO/Sources/RectUtils.c:197`) takes horizontal first.

`ZeroRectCorner()` normalises a rect to the origin while preserving its size:

```c
theRect->right -= theRect->left;      // RectUtils.c:59
theRect->bottom -= theRect->top;      // RectUtils.c:60
theRect->left = 0;                    // RectUtils.c:61
theRect->top = 0;                     // RectUtils.c:62
```

This is used constantly: take the art's source rect from `srcRects[]`, zero its corner to
turn it into a size, then `QOffsetRect` by the object's stored `topLeft`.

Small helpers with truncating integer division (all matter for off-by-ones):

| Helper | Body | Line |
|---|---|---|
| `HalfRectWide` | `(right - left) / 2` | `GliderPRO/Sources/RectUtils.c:79` |
| `HalfRectTall` | `(bottom - top) / 2` | 87 |
| `RectWide` | `right - left` | 95 |
| `RectTall` | `bottom - top` | 103 |

### 1.5 `IsRectLeftOfRect` — a precedence bug that is load-bearing

```c
offset = (rect1->right - rect1->left) - (rect2->right - rect2->left) / 2;   // RectUtils.c:185
if ((rect1->left) < (rect2->left + offset))
    return (true);
else
    return (false);
```

C precedence makes this `width1 - (width2 / 2)`, **not** `(width1 - width2) / 2`. The only
caller is the foil-shove branch of `CheckDynamicCollision()`
(`GliderPRO/Sources/Dynamics.c:53`), where `rect1` is the enemy and `rect2` is the glider
(`width2 = 48`, so `offset = enemyWidth - 24`). For a 24-wide balloon the test becomes
`enemy.left < glider.left + 0`; for a 64-wide dart it becomes
`enemy.left < glider.left + 40`. **Port the expression verbatim**, comment and all, or
foil-armoured players will be shoved the wrong way past wide enemies.

---

## 2. The per-frame evaluation order

`PlayGame()` is the game loop (`GliderPRO/Sources/Play.c:430`). Reproduced in call order,
with the two near-identical bodies collapsed (`if (twoPlayerGame)` at
`GliderPRO/Sources/Play.c:447` selects between lines 448-472 and 474-497; the only
differences are the second `GetInput`/`HandleGlider` call and, in the one-player arm, the
`demoGoing ? GetDemoInput : GetInput` choice at `:478-481`):

1. `gameFrame++;` (`GliderPRO/Sources/Play.c:434`)
2. `evenFrame = !evenFrame;` (`GliderPRO/Sources/Play.c:435`) — a global boolean toggled
   every frame; several interactions only act on even frames.
3. event pump — only when `doBackground`, and as a `do { HandlePlayEvent(); } while
   (switchedOut);` so the loop parks the game while the app is suspended
   (`GliderPRO/Sources/Play.c:437-443`)
4. `HandleTelephone();` (`GliderPRO/Sources/Play.c:445`)
5. **`HandleDynamics();`** (`GliderPRO/Sources/Play.c:449`) — moves toast/enemies/appliances
   **and performs enemy-vs-glider collision inline** (section 17).
6. **`GetInput(&theGlider);`** (+ `GetInput(&theGlider2)` in two-player)
   (`GliderPRO/Sources/Play.c:452`) — sets `hDesiredVel`, `tipped`, `heldLeft`,
   `heldRight`, fires bands, spends battery/helium.
7. **`HandleInteraction();`** (`GliderPRO/Sources/Play.c:454`) — the subject of this
   document: hot spots first, then room boundaries.
8. `HandleTriggers();` (`GliderPRO/Sources/Play.c:456`) — decrements armed trigger timers.
9. `HandleBands();` (`GliderPRO/Sources/Play.c:457`) — moves rubber bands and runs
   band-vs-hot-spot and band-vs-glider collision.
10. **`HandleGlider(&theGlider);`** (+ `theGlider2`) (`GliderPRO/Sources/Play.c:460`) —
    dispatches on `mode`, ultimately calling `MoveGlider()` which *consumes* the desired
    velocities set in steps 5-7 and integrates position.
11. `RenderFrame();` (`GliderPRO/Sources/Play.c:469`)
12. `HandleDynamicScoreboard();` (`GliderPRO/Sources/Play.c:470`)

Two consequences that a naive port gets wrong:

**(a) Collision runs *before* movement, on last frame's position.** `HandleInteraction()`
sees `dest` as it was left by the *previous* frame's `MoveGlider()`. Interactions do not
resolve a penetration that is about to happen; they set `hDesiredVel`/`vDesiredVel`
(velocity *requests*) and occasionally write `hVel`/`vVel` directly. The clamp-out-of-solid
writes (`vVel = kFloorLimit - dest.bottom`) are therefore *exact* corrections applied to a
velocity that has not yet been used — `MoveGlider()` will then move `dest.bottom` to
precisely `kFloorLimit`. This is why they read as "assign the gap, not clamp the position".

**(b) `HandleGrease()` runs inside `RenderFrame()`, not in the interaction phase.**
`RenderFrame()` calls `HandleGrease()` at `GliderPRO/Sources/Render.c:647`, which is
*after* `HandleInteraction()`. Grease spreading mutates hot-spot `action`, `isOn` and
`bounds` (`GliderPRO/Sources/Grease.c:60`, `:61`, `:68`, `:95`, `:102`). So a grease slick
that grows on frame N is only collidable on frame N+1. Preserve that one-frame lag.

`RenderFrame()`'s full order, for completeness
(`GliderPRO/Sources/Render.c:638`-`:670`): `DrawReflection` (if `hasMirror`) →
`HandleGrease()` → `RenderPendulums()` → `RenderFlames()` on even frames else
`RenderStars()` → `RenderDynamics()` → `RenderFlyingPoints()` → `RenderSparkles()` →
`RenderGlider(&theGlider, true)` → `RenderGlider(&theGlider2, false)` →
`RenderShreds()` → `RenderBands()` → busy-wait `while (TickCount() < nextFrame) {}` →
`nextFrame = TickCount() + kTicksPerFrame` → `CopyRectsQD()` → reset work-rect counters.

### 2.1 Frame rate and the meaning of "velocity"

`kTicksPerFrame` is 2 (`GliderPRO/Headers/GliderDefines.h:533`). A Mac tick is 1/60 s, so
the target frame time is 2/60 s = 33.33 ms, i.e. **30 fps**. All velocities are integer
pixels *per frame*; there is no delta-time anywhere. The frame limiter is the spin loop at
`GliderPRO/Sources/Render.c:662`, so on a fast machine the simulation is exactly 30 Hz and
on a slow machine it simply runs slower (no catch-up, no substepping).

A Go port must run the simulation on a fixed 2-tick (33.333 ms) step and must **not**
scale any velocity by elapsed time; doing so changes every clamp arithmetic in this
document.

### 2.2 `HandleInteraction` itself

```c
void HandleInteraction (void)                       // Interactions.c:1691
{
    CheckForHotSpots();                             // Interactions.c:1693
    if (twoPlayerGame) { ... CheckGliderInRoom per live glider ... }
    else CheckGliderInRoom(&theGlider);
}
```

Hot spots are always evaluated **before** room boundaries
(`GliderPRO/Sources/Interactions.c:1693`-`:1710`). That ordering is what makes doors work:
a `kIgnoreLeftWall` hot spot sets `thisGlider->ignoreLeft = true` earlier in the same frame
than `CheckEscapeLeft()` reads it.

---

## 3. Data structures

Sizes below are 68k/PPC classic-Mac layout: `short` = 2 bytes big-endian, `long` = 4,
`Boolean`/`Byte` = 1, **no padding inserted by these compilers for the members used here**,
and `Str27`/`Str255` are Pascal strings (length byte + bytes). Offsets were confirmed
against real house files (section 20).

### 3.1 `Rect`, `Point`

| Offset | Type | Field |
|---:|---|---|
| 0 | short | `top` |
| 2 | short | `left` |
| 4 | short | `bottom` |
| 6 | short | `right` |

Total 8 bytes.

| Offset | Type | Field |
|---:|---|---|
| 0 | short | `v` |
| 2 | short | `h` |

Total 4 bytes.

### 3.2 `objectType` — 12 bytes (`GliderPRO/Headers/GliderStructs.h:90`)

| Offset | Size | Field |
|---:|---:|---|
| 0 | 2 | `short what` — object type code (section 6.1) |
| 2 | 10 | union `data` — one of nine variants |

The nine union variants and their internal layouts (every one is exactly 10 bytes, so the
union is 10 bytes and `objectType` is 12):

**`blowerType data.a`** (`GliderPRO/Headers/GliderStructs.h:11`) — vents, blowers, flames,
`kLiftArea`, `kInvisBlower`

| Offset (within data) | Size | Field |
|---:|---:|---|
| 0 | 4 | `Point topLeft` (v, h) |
| 4 | 2 | `short distance` |
| 6 | 1 | `Boolean initial` |
| 7 | 1 | `Boolean state` |
| 8 | 1 | `Byte vector` |
| 9 | 1 | `Byte tall` |

`vector` is a nibble bit field, documented in the header as
`| x | x | x | x | 8 | 4 | 2 | 1 |` = bit3 left, bit2 down, bit1 right, bit0 up.

**`furnitureType data.b`** (`:21`) — all solid furniture, `kManhole`, `kInvisObstacle`,
`kInvisBounce`, `kStool`, `kBooks`

| Offset | Size | Field |
|---:|---:|---|
| 0 | 8 | `Rect bounds` (top,left,bottom,right) |
| 8 | 2 | `short pict` |

**`bonusType data.c`** (`:27`) — all prizes, `kGreaseRt/Lf`, `kSlider`, `kSparkle`

| Offset | Size | Field |
|---:|---:|---|
| 0 | 4 | `Point topLeft` |
| 4 | 2 | `short length` |
| 6 | 2 | `short points` |
| 8 | 1 | `Boolean state` |
| 9 | 1 | `Boolean initial` |

**`transportType data.d`** (`:36`) — stairs, mailboxes, ducts, transporters, doors, windows

| Offset | Size | Field |
|---:|---:|---|
| 0 | 4 | `Point topLeft` |
| 4 | 2 | `short tall` |
| 6 | 2 | `short where` (destination room, -1 = unlinked) |
| 8 | 1 | `Byte who` (destination object slot, 255 = unlinked) |
| 9 | 1 | `Byte wide` |

**`switchType data.e`** (`:45`) — switches and triggers

| Offset | Size | Field |
|---:|---:|---|
| 0 | 4 | `Point topLeft` |
| 4 | 2 | `short delay` |
| 6 | 2 | `short where` (-1 = unlinked) |
| 8 | 1 | `Byte who` |
| 9 | 1 | `Byte type` |

**`lightType data.f`** (`:54`) — `Point topLeft` (4), `short length` (2), `Byte byte0` (1),
`Byte byte1` (1), `Boolean initial` (1), `Boolean state` (1). Note the two spare bytes come
*before* the two booleans, unlike `bonusType`.

**`applianceType data.g`** (`:64`) — shredder, appliances, microwave, guitar, outlet,
`kCustomPict`

| Offset | Size | Field |
|---:|---:|---|
| 0 | 4 | `Point topLeft` |
| 4 | 2 | `short height` |
| 6 | 1 | `Byte byte0` |
| 7 | 1 | `Byte delay` |
| 8 | 1 | `Boolean initial` |
| 9 | 1 | `Boolean state` |

**`enemyType data.h`** — `Point topLeft`, `short length`, `Byte delay`, `byte0`,
`Boolean initial`, `state`.

**`clutterType data.i`** — `Rect bounds`, `short pict`.

### 3.3 `roomType` — 348 bytes (`GliderPRO/Headers/GliderStructs.h:166`)

| Offset | Size | Field |
|---:|---:|---|
| 0 | 28 | `Str27 name` (1 length byte + 27) |
| 28 | 2 | `short bounds` (bit-packed openings, section 13) |
| 30 | 1 | `Byte leftStart` |
| 31 | 1 | `Byte rightStart` |
| 32 | 1 | `Byte unusedByte` |
| 33 | 1 | `Boolean visited` |
| 34 | 2 | `short background` |
| 36 | 16 | `short tiles[8]` |
| 52 | 2 | `short floor` |
| 54 | 2 | `short suite` |
| 56 | 2 | `short openings` |
| 58 | 2 | `short numObjects` |
| 60 | 288 | `objectType objects[24]` (24 x 12) |

`60 + 288 = 348`, verified: `866 + 348 * nRooms == fileLength` for 21 of 22 shipped houses
(section 20.1).

### 3.4 `houseType` header — 866 bytes before `rooms[]`

| Offset | Size | Field |
|---:|---:|---|
| 0 | 2 | `short version` (all shipped houses: 0x0200) |
| 2 | 2 | `short unusedShort` |
| 4 | 4 | `long timeStamp` |
| 8 | 4 | `long flags` |
| 12 | 4 | `Point initial` (v, h) |
| 16 | 256 | `Str255 banner` |
| 272 | 256 | `Str255 trailer` |
| 528 | 292 | `scoresType highScores` |
| 820 | 40 | `gameType savedGame` |
| 860 | 1 | `Boolean hasGame` |
| 861 | 1 | `Boolean unusedBoolean` |
| 862 | 2 | `short firstRoom` |
| 864 | 2 | `short nRooms` |
| 866 | … | `roomType rooms[]` |

### 3.5 `gliderType` (`GliderPRO/Headers/GliderStructs.h:200`)

Eight `Rect`s in this order, then four `long`s, then shorts, then Booleans:

| Field | Type | Purpose |
|---|---|---|
| `src` | Rect | source rect in the glider art map |
| `mask` | Rect | source rect in the mask map |
| `dest` | Rect | **the collision rect**, room space |
| `whole` | Rect | union of pre- and post-move `dest` (sweep rect, for dirty-rect blitting) |
| `destShadow` | Rect | 48 x 9 shadow rect at `top = 306` |
| `wholeShadow` | Rect | sweep union of the shadow |
| `clip` | Rect | clip rect used during stairs/duct/mail animations |
| `enteredRect` | Rect | rect the glider entered the room through |
| `leftKey`,`rightKey`,`battKey`,`bandKey` | long x4 | KeyMap bit indices |
| `hVel`,`vVel` | short | current velocity, px/frame |
| `wasHVel`,`wasVVel` | short | velocity actually applied this frame (post-clamp) |
| `vDesiredVel`,`hDesiredVel` | short | requested velocity, consumed and reset by `MoveGlider` |
| `mode` | short | one of 24 mode codes (section 5.2) |
| `frame` | short | animation counter, reused as a countdown by several modes |
| `wasMode` | short | saved mode for limbo; **reused as burn/web countdown** |
| `facing` | Boolean | `kFaceLeft`/`kFaceRight` |
| `tipped` | Boolean | glider is flying "backwards" (thrust opposite to facing) |
| `sliding` | Boolean | standing on grease |
| `ignoreLeft`,`ignoreRight` | Boolean | one-frame permission to leave through that wall |
| `fireHeld` | Boolean | band key debounce |
| `which` | Boolean | player index (0/1) |
| `heldLeft`,`heldRight` | Boolean | direction keys held this frame |
| `dontDraw` | Boolean | suppress rendering (limbo) |
| `ignoreGround` | Boolean | one-frame permission to fall through the floor |

### 3.6 `hotObject` — the hot spot (`GliderPRO/Headers/GliderStructs.h:218`)

| Field | Type | Meaning |
|---|---|---|
| `bounds` | Rect | **room-space** active rect |
| `action` | short | one of 28 action codes (section 9) |
| `who` | short | index into `masterObjects[]` of the owning object |
| `isOn` | Boolean | inactive hot spots are skipped entirely |
| `stillOver` | Boolean | latch: "the glider was already overlapping last frame" |
| `doScrutinize` | Boolean | if true, inset the glider rect by 5 px before testing |

### 3.7 `objDataType` — the master object (`GliderPRO/Headers/GliderStructs.h:322`)

| Field | Type | Meaning |
|---|---|---|
| `roomNum` | short | which room this object lives in (real house room number) |
| `objectNum` | short | slot 0..23 within that room |
| `roomLink` | short | destination room this object links to, or -1 |
| `objectLink` | short | destination object slot, or -1 |
| `localLink` | short | index in `masterObjects[]` of the linked object, or -1 |
| `hotNum` | short | index in `hotSpots[]` of this object's *last* hot spot, or -1 |
| `dynaNum` | short | index in `dinahs[]`, or -1 |
| `theObject` | objectType | a **copy** of the 12-byte object record |

Note `theObject` is a *copy*, so `SetObjectState()` must write both the copy and the
authoritative house record (`GliderPRO/Sources/Objects.c:366`).

### 3.8 `dynaType` — dynamic object (`GliderPRO/Headers/GliderStructs.h:310`)

| Field | Type |
|---|---|
| `dest` | Rect (screen space for appliances, **room space** for enemies/toast) |
| `whole` | Rect (sweep union) |
| `hVel`,`vVel` | short |
| `type` | short (object type code) |
| `count`,`frame`,`timer`,`position`,`room` | short |
| `byte0`,`byte1` | Byte |
| `moving`,`active` | Boolean |

### 3.9 `bandType`, `greaseType`

`bandType` (`:274`): `Rect dest`, `short mode, count, hVel, vVel`.
`greaseType` (`:287`): `Rect dest`, then `short mapNum, mode`, `short who, where`,
`short start, stop`, `short frame, hotNum`, `Boolean isRight` — in that declaration order
(there is no `savedNum` field). Full layout in section 19.1.

---

## 4. The five rectangle lists

`CreatePointers()` allocates all of them (`GliderPRO/Sources/StructuresInit2.c:187`):

| List | Element | Capacity | Constant | Cap line | Purpose |
|---|---|---:|---|---|---|
| `srcRects[]` | Rect | 144 | `kNumSrcRects` = 0x90 | `GliderPRO/Headers/GliderDefines.h:437` | art source rect *and canonical size* for every object type |
| `hotSpots[]` | hotObject | 56 | `kMaxHotSpots` | 259 | **the collision list** for the current room |
| `masterObjects[]` | objDataType | 216 | `kMaxMasterObjects` (24 x 9) | 266 | every object in the 3x3 room neighbourhood |
| `dinahs[]` | dynaType | 18 | `kMaxDynamicObs` | 265 | animated/moving objects |
| `bands[]` | bandType | 2 | `kMaxRubberBands` | — | rubber bands in flight |
| `grease[]` | greaseType | 16 | `kMaxGrease` | — | grease jars |
| `tempManholes[]` | Rect | 8 | `kMaxTempManholes` | `GliderPRO/Sources/Objects.c:12` | **graphics only** — floor-support drawing |

`tempManholes[]` is a trap: `AddTempManholeRect()` sets
`tempRect.bottom = tempRect.top + kFloorSupportTall;`
(`GliderPRO/Sources/Objects.c:358`) and the array is consumed only by the floor-support
drawing pass in `RoomGraphics.c`. **Manhole collision is not done through it**; a manhole's
collision behaviour comes from a `kIgnoreGround` hot spot (section 6.4).

Also note the two *counts* that bound iteration: `nHotSpots` (how many hot spots the
current room produced) and `numMasterObjects` / `numLocalMasterObjects`
(`GliderPRO/Sources/Objects.c:76`). Every sweep loops `0 .. count-1`, never the capacity.

### 4.1 `srcRects[]` — the canonical size table

`InitSrcRects()` (`GliderPRO/Sources/StructuresInit2.c:306`-`:475`) fills the array with
116 literal `QSetRect` calls (some followed by `QOffsetRect`). The array is sized 144
(`kNumSrcRects` = 0x90) but the `what` codes are sparse, so 28 of the 144 slots are never
written and stay zeroed — indexing one of those is a latent bug, not a size. These rects are positions
*inside the art sheets*, but because almost every hot-rect formula begins with
`bounds = srcRects[what]; ZeroRectCorner(&bounds);` the values that matter are the
**widths and heights**. The complete table, as `(left, top, right, bottom)` after any
offsetting:

| Type | `srcRects` value | W x H | Line |
|---|---|---|---|
| `kFloorVent` | (0, 0, 48, 11) | 48 x 11 | 308 |
| `kCeilingVent` | (0, 11, 48, 22) | 48 x 11 | 310 |
| `kFloorBlower` | (0, 22, 48, 37) | 48 x 15 | 312 |
| `kCeilingBlower` | (0, 37, 48, 52) | 48 x 15 | 314 |
| `kSewerGrate` | (0, 52, 48, 69) | 48 x 17 | 316 |
| `kLeftFan` | (0, 69, 40, 124) | 40 x 55 | 318 |
| `kRightFan` | (0, 124, 40, 179) | 40 x 55 | 320 |
| `kTaper` | (0, 209, 20, 268) | 20 x 59 | 322 |
| `kCandle` | (0, 179, 32, 209) | 32 x 30 | 324 |
| `kStubby` | (0, 268, 20, 304) | 20 x 36 | 326 |
| `kTiki` | (21, 268, 48, 296) | 27 x 28 | 328 |
| `kBBQ` | (0, 0, 64, 33) | 64 x 33 | 330 |
| `kInvisBlower` | (0, 0, 24, 24) | 24 x 24 | 331 |
| `kGrecoVent` | (0, 340, 48, 358) | 48 x 18 | 332 |
| `kSewerBlower` | (0, 390, 32, 402) | 32 x 12 | 334 |
| `kLiftArea` | (0, 0, 64, 32) | 64 x 32 | 336 |
| `kTable` | (0, 0, 64, 8) | 64 x 8 (`kTableThick`) | 338 |
| `kShelf` | (0, 0, 64, 6) | 64 x 6 (`kShelfThick`) | 339 |
| `kCabinet` | (0, 0, 64, 64) | 64 x 64 | 340 |
| `kFilingCabinet` | (0, 0, 74, 107) | 74 x 107 | 341 |
| `kWasteBasket` | (0, 43, 64, 104) | 64 x 61 | 342 |
| `kMilkCrate` | (0, 104, 64, 162) | 64 x 58 | 344 |
| `kCounter` | (0, 0, 128, 64) | 128 x 64 | 346 |
| `kDresser` | (0, 0, 128, 64) | 128 x 64 | 347 |
| `kDeckTable` | (0, 0, 64, 8) | 64 x 8 (`kTableThick`) | 348 |
| `kStool` | (0, 183, 48, 221) | 48 x 38 | 349 |
| `kTrunk` | (0, 0, 144, 80) | 144 x 80 | 351 |
| `kInvisObstacle` | (0, 0, 64, 64) | 64 x 64 | 352 |
| `kManhole` | (0, 0, 123, 22) | 123 x 22 | 353 |
| `kBooks` | (0, 0, 64, 51) | 64 x 51 | 354 |
| `kInvisBounce` | (0, 0, 64, 64) | 64 x 64 | 355 |
| `kRedClock` | (0, 0, 28, 17) | 28 x 17 | 357 |
| `kBlueClock` | (0, 17, 28, 42) | 28 x 25 | 358 |
| `kYellowClock` | (0, 42, 28, 70) | 28 x 28 | 360 |
| `kCuckoo` | (0, 148, 40, 228) | 40 x 80 | 362 |
| `kPaper` | (0, 127, 48, 148) | 48 x 21 | 364 |
| `kBattery` | (32, 0, 48, 25) | 16 x 25 | 366 |
| `kBands` | (20, 70, 48, 93) | 28 x 23 | 368 |
| `kGreaseRt` | (0, 243, 32, 270) | 32 x 27 | 370 |
| `kGreaseLf` | (0, 324, 32, 351) | 32 x 27 | 372 |
| `kFoil` | (0, 228, 55, 243) | 55 x 15 | 374 |
| `kInvisBonus` | (0, 0, 24, 24) | 24 x 24 | 376 |
| `kStar` | (48, 0, 80, 31) | 32 x 31 | 377 |
| `kSparkle` | (0, 70, 20, 89) | 20 x 19 | 379 |
| `kHelium` | (32, 270, 88, 286) | 56 x 16 | 381 |
| `kSlider` | (0, 0, 64, 16) | 64 x 16 | 383 |
| `kUpStairs`, `kDownStairs` | (0, 0, 160, 267) | 160 x 267 | 385 |
| `kMailboxLf`, `kMailboxRt` | (0, 0, 94, 80) | 94 x 80 | 387 |
| `kFloorTrans` | (0, 1, 56, 16) | 56 x 15 | 389 |
| `kCeilingTrans` | (0, 16, 56, 31) | 56 x 15 | 391 |
| `kDoorInLf`, `kDoorInRt` | (0, 0, 144, 322) | 144 x 322 | 393 |
| `kDoorExRt`, `kDoorExLf` | (0, 0, 16, 322) | 16 x 322 | 395 |
| `kWindowInLf`, `kWindowInRt` | (0, 0, 20, 170) | 20 x 170 | 397 |
| `kWindowExRt`, `kWindowExLf` | (0, 0, 16, 170) | 16 x 170 | 399 |
| `kInvisTrans` | (0, 0, 64, 32) | 64 x 32 | 401 |
| `kDeluxeTrans` | (0, 0, 64, 64) | 64 x 64 | 402 |
| `kLightSwitch` | (0, 0, 15, 24) | 15 x 24 | 404 |
| `kMachineSwitch` | (0, 48, 16, 72) | 16 x 24 | 405 |
| `kThermostat` | (0, 48, 15, 72) | 15 x 24 | 407 |
| `kPowerSwitch` | (0, 72, 8, 80) | 8 x 8 | 409 |
| `kKnifeSwitch` | (0, 80, 16, 104) | 16 x 24 | 411 |
| `kInvisSwitch` | (0, 0, 12, 12) | 12 x 12 | 413 |
| `kTrigger` | (0, 0, 12, 12) | 12 x 12 | 414 |
| `kLgTrigger` | (0, 0, 48, 48) | 48 x 48 | 415 |
| `kSoundTrigger` | (0, 0, 32, 32) | 32 x 32 | 416 |
| `kShredder` | (0, 0, 73, 22) | 73 x 22 | 430 |
| `kToaster` | (0, 22, 48, 49) | 48 x 27 | 431 |
| `kMacPlus` | (0, 49, 48, 107) | 48 x 58 | 433 |
| `kGuitar` | (0, 0, 64, 172) | 64 x 172 | 435 |
| `kTV` | (0, 0, 92, 77) | 92 x 77 | 436 |
| `kCoffee` | (0, 107, 43, 171) | 43 x 64 | 437 |
| `kOutlet` | (64, 22, 80, 46) | 16 x 24 | 439 |
| `kVCR` | (0, 0, 96, 22) | 96 x 22 | 441 |
| `kStereo` | (0, 0, 128, 53) | 128 x 53 | 442 |
| `kMicrowave` | (0, 0, 92, 59) | 92 x 59 | 443 |
| `kCinderBlock` | (0, 0, 40, 62) | 40 x 62 | 444 |
| `kFlowerBox` | (0, 0, 80, 32) | 80 x 32 | 445 |
| `kCDs` | (48, 22, 64, 52) | 16 x 30 | 446 |
| `kCustomPict` | (0, 0, 72, 34) | 72 x 34 | 448 |
| `kBalloon` | (0, 0, 24, 30) | 24 x 30 | 450 |
| `kCopterLf`, `kCopterRt` | (0, 0, 32, 30) | 32 x 30 | 451 |
| `kDartLf`, `kDartRt` | (0, 0, 64, 19) | 64 x 19 | 453 |
| `kBall` | (0, 0, 32, 32) | 32 x 32 | 455 |
| `kDrip` | (0, 0, 16, 12) | 16 x 12 | 456 |
| `kFish` | (0, 0, 36, 33) | 36 x 33 | 457 |
| `kCobweb` | (0, 0, 54, 45) | 54 x 45 | 458 |
| `kChimes` | (0, 0, 28, 74) | 28 x 74 | 474 |

Light types (lines 418-428) and clutter types (460-473) also have entries but produce no
hot spots.

**Porting note:** these are *not* derived from the PICT resources at runtime for most
types — they are hard-coded, so a Go port can hard-code the same table and does not need
the original art to get collision right. The two exceptions that *do* consult art are
`kCustomPict`, whose height comes from `GetPicture(data.g.height)` with a 10000 fallback
(`GliderPRO/Sources/ObjectRects.c:220`), and the light `length` variants
(`GliderPRO/Sources/ObjectRects.c:190`).

---

## 5. Room setup: how `hotSpots[]` comes to exist

### 5.1 The pipeline

`ReadyLevel()` (`GliderPRO/Sources/RoomGraphics.c:402`) is called on every room change:

1. `NilSavedMaps();`
2. … background/art setup …
3. **`DetermineRoomOpenings();`** — computes `leftOpen`, `rightOpen`, `topOpen`,
   `bottomOpen`, `leftThresh`, `rightThresh` (section 13).
4. **`DrawLocale();`**
5. `InitGarbageRects();`

`DrawLocale()` (`GliderPRO/Sources/RoomGraphics.c:44`) then:

1. Resets all per-room state:
   `ZeroFlamesAndTheLike(); ZeroDinahs(); KillAllBands(); ZeroMirrorRegion();
   ZeroTriggers(); numTempManholes = 0; FlushAnyTriggerPlaying(); DumpTriggerSound();
   tvInRoom = false; tvWithMovieNumber = -1;`
   (`GliderPRO/Sources/RoomGraphics.c:51`-`:60`).
2. `roomV = rooms[thisRoomNumber].floor;` (`GliderPRO/Sources/RoomGraphics.c:64`)
3. For `i = 0 .. 8`: `localNumbers[i] = GetNeighborRoomNumber(i);
   isStructure[i] = IsRoomAStructure(localNumbers[i]);`
   (`GliderPRO/Sources/RoomGraphics.c:67`-`:71`)
4. **`ListAllLocalObjects();`** (`GliderPRO/Sources/RoomGraphics.c:72`) — builds
   `masterObjects[]` and, as a side effect, `hotSpots[]`.
5. Draws the eight neighbour rooms then the central room (78-121).
6. `DrawFloorSupport()` if `numNeighbors > 3` (123).
7. `RestoreWorkMap();` (125), `shadowVisible = IsShadowVisible();` (126),
   `takingTheStairs = false;` (127).

The nine neighbour slots are indexed by the `kCentralRoom` … `kNorthWestRoom` enumeration;
`localNumbers[i]` holds the real house room number of each, or `kRoomIsEmpty` if absent.

### 5.2 `ListAllLocalObjects` (`GliderPRO/Sources/Objects.c:300`)

```
 1. numMasterObjects = 0; numLocalMasterObjects = 0; nHotSpots = 0;   // Objects.c:305-307
 2. ListOneRoomsObjects(kCentralRoom);                                // Objects.c:312
 3. if (numNeighbors > 1)  { ListOneRoomsObjects(kEastRoom);
                             ListOneRoomsObjects(kWestRoom); }        // Objects.c:314-318
 4. if (numNeighbors > 3)  { North, NorthEast, SouthEast,
                             South, SouthWest, NorthWest }            // Objects.c:320-328
 5. O(n^2) link-resolution pass: for each master object with roomLink/objectLink set,
    find the master object whose (roomNum, objectNum) matches and store its index in
    localLink; otherwise leave localLink = -1.                        // Objects.c:332-346
```

`numNeighbors` is 1 (central only), 3 (central + E + W) or 9 depending on the house's
"neighbourhood size" setting. That means **objects in vertically adjacent rooms are only
loaded in the 9-room mode**, which affects cross-room links.

### 5.3 `ListOneRoomsObjects` (`GliderPRO/Sources/Objects.c:254`)

```
 1. roomNum = localNumbers[where];
 2. if (roomNum == kRoomIsEmpty) return;                              // Objects.c:261
 3. for (n = 0; n < kMaxRoomObs (24); n++)                            // Objects.c:266
 4.     if (numMasterObjects >= kMaxMasterObjects (216)) break;       // Objects.c:268
 5.     masterObjects[numMasterObjects].roomNum    = roomNum;         // Objects.c:272
 6.     masterObjects[numMasterObjects].objectNum  = n;               // Objects.c:273
 7.     masterObjects[numMasterObjects].roomLink   = GetRoomLinked(roomNum, n);
 8.     masterObjects[numMasterObjects].objectLink = GetObjectLinked(roomNum, n);
 9.     masterObjects[numMasterObjects].localLink  = -1;              // Objects.c:278
10.     masterObjects[numMasterObjects].theObject  = (copy of the 12-byte record);
11.     if ((where == kCentralRoom) && IsThisValid(roomNum, n))
12.         masterObjects[...].hotNum = CreateActiveRects(numMasterObjects);
13.     else
14.         masterObjects[...].hotNum = -1;                           // Objects.c:283-286
15.     masterObjects[...].dynaNum = -1;                              // Objects.c:287
16.     numMasterObjects++;                                           // Objects.c:289
17.     if (where == kCentralRoom) numLocalMasterObjects++;            // Objects.c:291
```

Three critical facts fall out of this:

* **Empty slots still consume a `masterObjects[]` entry.** The loop runs all 24 slots
  unconditionally, so master-object indices are `roomOrder * 24 + slot` for as long as no
  room fills the array. A port must keep the same indexing because `hotSpots[].who`,
  `masterObjects[].localLink`, and `triggers[].index` are all indices into this array.
* **Only the central room produces hot spots** (`where == kCentralRoom`,
  `GliderPRO/Sources/Objects.c:283`). Objects in neighbouring rooms are drawn but never
  collidable. There is therefore no "collide with the room next door".
* **`hotNum` records only the *last* hot spot an object created.** `CreateActiveRects()`
  returns a single index (`GliderPRO/Sources/ObjectRects.c:1062`) even for objects that add
  two or three rects (flames, fans, microwave). `SetObjectState()` therefore only ever
  toggles the last of them (section 11.4) — for a fan that is the push column, and for a
  flame it is the `kDissolveIt` body rect, not the lift column.

`dynaNum` stays -1 until the *drawing* pass fills it in
(`GliderPRO/Sources/ObjectDrawAll.c:952`-`:960`):

```c
if (!redraw)
    for (n = 0; n < numMasterObjects; n++)
        if ((masterObjects[n].objectNum == i) &&
            (masterObjects[n].roomNum == localNumbers[neighbor]))
            masterObjects[n].dynaNum = dynamicNum;
```

So anything that dereferences `dynaNum` (switch actions, triggers) is only valid after
`DrawLocale()` completes.

### 5.4 `IsThisValid` (`GliderPRO/Sources/Objects.c:89`)

Returns:

* `false` if `what == kObjectIsEmpty` (0);
* `data.c.state` for `kRedClock`, `kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`,
  `kBattery`, `kBands`, `kFoil`, `kInvisBonus`, `kStar`, `kSparkle`, `kHelium` — i.e. an
  already-collected prize produces no hot spot at all;
* `true` otherwise.

Note `kGreaseRt`/`kGreaseLf` are *not* in that list, so a spilled grease jar still produces
a hot spot (as a `kSlideIt` rect — see `GliderPRO/Sources/ObjectRects.c:681`-`:717`).

### 5.5 `AddActiveRect` (`GliderPRO/Sources/ObjectRects.c:277`)

```c
if (nHotSpots >= kMaxHotSpots)   return (-1);          // ObjectRects.c:280
hotSpots[nHotSpots].bounds       = *theRect;           // ObjectRects.c:283
hotSpots[nHotSpots].action       = action;             // ObjectRects.c:284
hotSpots[nHotSpots].who          = who;                // ObjectRects.c:285
hotSpots[nHotSpots].isOn         = isOn;               // ObjectRects.c:286
hotSpots[nHotSpots].stillOver    = false;              // ObjectRects.c:287
hotSpots[nHotSpots].doScrutinize = scrutinize;         // ObjectRects.c:288
nHotSpots++;                                           // ObjectRects.c:289
return (nHotSpots - 1);                                // ObjectRects.c:291
```

**Silent overflow.** Past 56 hot spots the object simply becomes non-interactive and
`hotNum` becomes -1; there is no error. A 24-object room can legitimately want more than 56
rects (24 BBQs would want 72), so the cap is reachable. Replicate the cap: dropping it
changes behaviour in dense rooms.

`hotSpots[]` is **ordered by (object slot, then order of `AddActiveRect` calls within
`CreateActiveRects`)**. That ordering is observable: `activeRectEscaped` stores a hot-spot
*index* and compares it across two players' frames (section 9.16), and `kMicrowaveIt` sits
at `hotNum` for the microwave while its `kDissolveIt` sibling sits at `hotNum - 1`.

---

## 6. Per-object hot-spot geometry (the complete table)

`CreateActiveRects(short who)` (`GliderPRO/Sources/ObjectRects.c:296`) is one giant switch
on `masterObjects[who].theObject.what`. Its preamble:

```c
hotSpotNumber = -1;                                    // ObjectRects.c:303
theObject = masterObjects[who].theObject;              // ObjectRects.c:304
```

and it returns `hotSpotNumber` (`GliderPRO/Sources/ObjectRects.c:1062`), which is the index
of the *last* `AddActiveRect` performed.

### 6.1 Object type codes

The complete `what` enumeration from `GliderPRO/Headers/GliderDefines.h:311`-`:435`. Values
are the on-disk codes; note the deliberate gaps at 0x20, 0x30, 0x4A-0x50, 0x59-0x60,
0x6F-0x70, 0x7A-0x80, which group the types into families.

| Code | Name | Family | Hot-spot action(s) |
|---|---|---|---|
| 0x00 | `kObjectIsEmpty` | — | none |
| 0x01 | `kFloorVent` | blower | `kLiftIt` |
| 0x02 | `kCeilingVent` | blower | `kDropIt` |
| 0x03 | `kFloorBlower` | blower | `kLiftIt` |
| 0x04 | `kCeilingBlower` | blower | `kDropIt` |
| 0x05 | `kSewerGrate` | blower | `kLiftIt` |
| 0x06 | `kLeftFan` | blower | `kDissolveIt` + `kPushItLeft` |
| 0x07 | `kRightFan` | blower | `kDissolveIt` + `kPushItRight` |
| 0x08 | `kTaper` | flame | (`kLiftIt` +) `kBurnIt` + `kDissolveIt` |
| 0x09 | `kCandle` | flame | (`kLiftIt` +) `kBurnIt` + `kDissolveIt` |
| 0x0A | `kStubby` | flame | (`kLiftIt` +) `kBurnIt` + `kDissolveIt` |
| 0x0B | `kTiki` | flame | (`kLiftIt` +) `kBurnIt` + `kDissolveIt` |
| 0x0C | `kBBQ` | flame | (`kLiftIt` +) `kBurnIt` + `kDissolveIt` |
| 0x0D | `kInvisBlower` | blower | one of `kLiftIt`/`kPushItRight`/`kDropIt`/`kPushItLeft` |
| 0x0E | `kGrecoVent` | blower | `kLiftIt` |
| 0x0F | `kSewerBlower` | blower | `kLiftIt` |
| 0x10 | `kLiftArea` | blower | one of `kLiftIt`/`kPushItRight`/`kDropIt`/`kPushItLeft` |
| 0x11 | `kTable` | furniture | `kDissolveIt` |
| 0x12 | `kShelf` | furniture | `kDissolveIt` |
| 0x13 | `kCabinet` | furniture | `kDissolveIt` |
| 0x14 | `kFilingCabinet` | furniture | `kDissolveIt` |
| 0x15 | `kWasteBasket` | furniture | `kDissolveIt` |
| 0x16 | `kMilkCrate` | furniture | `kDissolveIt` |
| 0x17 | `kCounter` | furniture | `kDissolveIt` |
| 0x18 | `kDresser` | furniture | `kDissolveIt` |
| 0x19 | `kDeckTable` | furniture | `kDissolveIt` |
| 0x1A | `kStool` | furniture | `kDissolveIt` (top 25 px only) |
| 0x1B | `kTrunk` | furniture | `kDissolveIt` |
| 0x1C | `kInvisObstacle` | furniture | `kDissolveIt` |
| 0x1D | `kManhole` | furniture | `kIgnoreGround` |
| 0x1E | `kBooks` | furniture | `kDissolveIt` (width - 2) |
| 0x1F | `kInvisBounce` | furniture | `kBounceIt` |
| 0x21 | `kRedClock` | prize | `kRewardIt` |
| 0x22 | `kBlueClock` | prize | `kRewardIt` |
| 0x23 | `kYellowClock` | prize | `kRewardIt` |
| 0x24 | `kCuckoo` | prize | `kRewardIt` |
| 0x25 | `kPaper` | prize | `kRewardIt` |
| 0x26 | `kBattery` | prize | `kRewardIt` |
| 0x27 | `kBands` | prize | `kRewardIt` |
| 0x28 | `kGreaseRt` | prize | `kRewardIt` (unspilled) / `kSlideIt` (spilled) |
| 0x29 | `kGreaseLf` | prize | `kRewardIt` (unspilled) / `kSlideIt` (spilled) |
| 0x2A | `kFoil` | prize | `kRewardIt` |
| 0x2B | `kInvisBonus` | prize | `kRewardIt` |
| 0x2C | `kStar` | prize | `kRewardIt` |
| 0x2D | `kSparkle` | prize | **none** (bounds computed, never added) |
| 0x2E | `kHelium` | prize | `kRewardIt` |
| 0x2F | `kSlider` | prize | `kSlideIt` |
| 0x31 | `kUpStairs` | transport | `kMoveItUp` |
| 0x32 | `kDownStairs` | transport | `kMoveItDown` |
| 0x33 | `kMailboxLf` | transport | `kMailItLeft` (if linked) |
| 0x34 | `kMailboxRt` | transport | `kMailItRight` (if linked) |
| 0x35 | `kFloorTrans` | transport | `kDuctItDown` (if linked) |
| 0x36 | `kCeilingTrans` | transport | `kDuctItUp` (if linked) |
| 0x37 | `kDoorInLf` | transport | `kIgnoreLeftWall` |
| 0x38 | `kDoorInRt` | transport | `kIgnoreRightWall` |
| 0x39 | `kDoorExRt` | transport | `kIgnoreRightWall` |
| 0x3A | `kDoorExLf` | transport | `kIgnoreLeftWall` |
| 0x3B | `kWindowInLf` | transport | `kIgnoreLeftWall` |
| 0x3C | `kWindowInRt` | transport | `kIgnoreRightWall` |
| 0x3D | `kWindowExRt` | transport | `kIgnoreRightWall` |
| 0x3E | `kWindowExLf` | transport | `kIgnoreLeftWall` |
| 0x3F | `kInvisTrans` | transport | `kTransportIt` (if linked) |
| 0x40 | `kDeluxeTrans` | transport | `kTransportIt` (if linked) |
| 0x41 | `kLightSwitch` | switch | `kSwitchIt` (if `where != -1`) |
| 0x42 | `kMachineSwitch` | switch | `kSwitchIt` (if `where != -1`) |
| 0x43 | `kThermostat` | switch | `kSwitchIt` (if `where != -1`) |
| 0x44 | `kPowerSwitch` | switch | `kSwitchIt` (if `where != -1`) |
| 0x45 | `kKnifeSwitch` | switch | `kSwitchIt` (if `where != -1`) |
| 0x46 | `kInvisSwitch` | switch | `kSwitchIt` (if `where != -1`) |
| 0x47 | `kTrigger` | switch | `kTriggerIt` (if `where != -1`) |
| 0x48 | `kLgTrigger` | switch | `kTriggerIt` (if `where != -1`) |
| 0x49 | `kSoundTrigger` | switch | `kSoundIt` (if the sound loads) |
| 0x51 | `kCeilingLight` | light | none |
| 0x52 | `kLightBulb` | light | none |
| 0x53 | `kTableLamp` | light | none |
| 0x54 | `kHipLamp` | light | none |
| 0x55 | `kDecoLamp` | light | none |
| 0x56 | `kFlourescent` | light | none |
| 0x57 | `kTrackLight` | light | none |
| 0x58 | `kInvisLight` | light | none |
| 0x61 | `kShredder` | appliance | `kShredIt` |
| 0x62 | `kToaster` | appliance | `kDissolveIt` |
| 0x63 | `kMacPlus` | appliance | `kDissolveIt` |
| 0x64 | `kGuitar` | appliance | `kStrumIt` |
| 0x65 | `kTV` | appliance | `kDissolveIt` |
| 0x66 | `kCoffee` | appliance | `kDissolveIt` |
| 0x67 | `kOutlet` | appliance | `kIgnoreIt` |
| 0x68 | `kVCR` | appliance | `kDissolveIt` |
| 0x69 | `kStereo` | appliance | `kDissolveIt` |
| 0x6A | `kMicrowave` | appliance | `kDissolveIt` **+** `kMicrowaveIt` |
| 0x6B | `kCinderBlock` | appliance | `kDissolveIt` |
| 0x6C | `kFlowerBox` | appliance | `kDissolveIt` |
| 0x6D | `kCDs` | appliance | `kDissolveIt` |
| 0x6E | `kCustomPict` | appliance | none |
| 0x71 | `kBalloon` | enemy | `kIgnoreIt` |
| 0x72 | `kCopterLf` | enemy | `kIgnoreIt` |
| 0x73 | `kCopterRt` | enemy | `kIgnoreIt` |
| 0x74 | `kDartLf` | enemy | `kIgnoreIt` |
| 0x75 | `kDartRt` | enemy | `kIgnoreIt` |
| 0x76 | `kBall` | enemy | `kIgnoreIt` |
| 0x77 | `kDrip` | enemy | `kIgnoreIt` |
| 0x78 | `kFish` | enemy | `kDissolveIt` |
| 0x79 | `kCobweb` | enemy | `kWebIt` |
| 0x81 | `kOzma` | clutter | none |
| 0x82 | `kMirror` | clutter | none |
| 0x83 | `kMousehole` | clutter | none |
| 0x84 | `kFireplace` | clutter | none |
| 0x85 | `kFlower` | clutter | none |
| 0x86 | `kWallWindow` | clutter | none |
| 0x87 | `kBear` | clutter | none |
| 0x88 | `kCalendar` | clutter | none |
| 0x89 | `kVase1` | clutter | none |
| 0x8A | `kVase2` | clutter | none |
| 0x8B | `kBulletin` | clutter | none |
| 0x8C | `kCloud` | clutter | none |
| 0x8D | `kFaucet` | clutter | none |
| 0x8E | `kRug` | clutter | none |
| 0x8F | `kChimes` | clutter | `kChimeIt` |

Enemies get `kIgnoreIt` hot spots: the rect exists (so that rubber bands and the rendering
system can see the object) but the interaction switch does nothing for action 0. Enemy
damage is done separately, from `HandleDynamics()` via `CheckDynamicCollision()`
(section 17). `kFish` is the exception — it has a real `kDissolveIt` rect as well.

### 6.2 Geometry constants used by `CreateActiveRects`

From `GliderPRO/Sources/ObjectRects.c:13`-`:19`:

| Constant | Value |
|---|---:|
| `kFloorColumnWide` | 4 |
| `kCeilingColumnWide` | 24 |
| `kFanColumnThick` | 16 |
| `kFanColumnDown` | 20 |
| `kDeadlyFlameHeight` | 24 |
| `kStoolThick` | 25 |
| `kShredderActiveHigh` | 40 |

### 6.3 Blowers and vents

Every upward blower uses the same pattern. `kFloorVent`
(`GliderPRO/Sources/ObjectRects.c:311`):

```c
QSetRect(&bounds, 0, -theObject.data.a.distance, kFloorColumnWide, 0);      // :312
QOffsetRect(&bounds, HalfRectWide(&srcRects[kFloorVent]) -
                     (kFloorColumnWide / 2), 0);                           // :313
QOffsetRect(&bounds, theObject.data.a.topLeft.h,
                     theObject.data.a.topLeft.v);                          // :315
hotSpotNumber = AddActiveRect(&bounds, kLiftIt, who,
                              theObject.data.a.state, false);              // :317
```

So the lift column is **4 px wide**, `distance` px tall, sitting *above* the object's
`topLeft`, horizontally centred on the art: for `kFloorVent`, `HalfRectWide` = 48/2 = 24,
minus 4/2 = 2, so offset +22. Both divisions truncate.

Complete blower table (all are `AddActiveRect(..., action, who, data.a.state, false)`):

| Type | `QSetRect(l, t, r, b)` | Centring `QOffsetRect(h, v)` | Action | Lines |
|---|---|---|---|---|
| `kFloorVent` | (0, -distance, 4, 0) | (24 - 2, 0) = (22, 0) | `kLiftIt` | 312-317 |
| `kCeilingVent` | (0, 0, 24, distance) | (24 - 12, 0) = (12, 0) | `kDropIt` | 322-329 |
| `kFloorBlower` | (0, -distance, 4, 0) | (24 - 2, 0) = (22, 0) | `kLiftIt` | 334-341 |
| `kCeilingBlower` | (0, 0, 24, distance) | (12, 0) | `kDropIt` | 346-353 |
| `kSewerGrate` | (0, -distance, 4, 0) | (22, 0) | `kLiftIt` | 358-365 |
| `kGrecoVent` | (0, -distance, 4, 0) | (22, 0) | `kLiftIt` | 567-574 |
| `kSewerBlower` | (0, -distance, 4, 0) | (16 - 2, 0) = (14, 0) | `kLiftIt` | 579-586 |

(`kSewerBlower`'s art is 32 wide, hence 16 - 2 = 14.)

`kCeilingVent`/`kCeilingBlower` use `kCeilingColumnWide` = 24 for the drop column, i.e.
**the drop column is six times wider than the lift column**. That asymmetry is intentional
and very noticeable in play.

### 6.4 Fans — two rects each

`kLeftFan` (`GliderPRO/Sources/ObjectRects.c:369`):

```c
QSetRect(&bounds, 0, 0, 13, 43);                                   // :370  the blade
QOffsetRect(&bounds, theObject.data.a.topLeft.h + 16,
                     theObject.data.a.topLeft.v + 12);             // :371
AddActiveRect(&bounds, kDissolveIt, who, true, true);              // :374  always on, scrutinized

QSetRect(&bounds, 0, 0, theObject.data.a.distance, kFanColumnThick); // :375  the push column
QOffsetRect(&bounds, -(theObject.data.a.distance), kFanColumnDown);  // :376
QOffsetRect(&bounds, topLeft.h, topLeft.v);                          // :378
hotSpotNumber = AddActiveRect(&bounds, kPushItLeft, who,
                              theObject.data.a.state, false);        // :380
```

`kRightFan` (`GliderPRO/Sources/ObjectRects.c:384`) is the mirror: blade at
`(topLeft.h + 6, topLeft.v + 12)` size 13 x 43 (`:386`), push column offset
`(RectWide(&srcRects[kRightFan]) /* = 40 */, kFanColumnDown /* = 20 */)` (`:391`) then
`topLeft`, action `kPushItRight` (`:395`).

So a fan's blade is a *permanent* solid (isOn hard-coded `true`) 13 x 43 rect, while only
the wind column responds to the on/off state. `hotNum` points at the **wind column**, so
`SetObjectState()` toggling the fan toggles the wind, not the blade.

### 6.5 Flames — up to three rects each

The five flame types share one shape. Using `kTaper`
(`GliderPRO/Sources/ObjectRects.c:399`):

```c
QSetRect(&bounds, 0, -distance, kFloorColumnWide, 0);           // :400
QOffsetRect(&bounds, HalfRectWide(&srcRects[kTaper]) -
                     (kFloorColumnWide / 2), 0);                // :401
QOffsetRect(&bounds, topLeft.h, topLeft.v);                     // :403
if ((bounds.bottom - bounds.top) > kDeadlyFlameHeight)          // :407  i.e. distance > 24
{
    bounds.bottom -= kDeadlyFlameHeight;                        // :409
    AddActiveRect(&bounds, kLiftIt, who, true, false);          // :410
    bounds.bottom += kDeadlyFlameHeight;
    bounds.top = bounds.bottom - kDeadlyFlameHeight + 2;        // :412
    AddActiveRect(&bounds, kBurnIt, who, true, false);          // :413
}
else
    AddActiveRect(&bounds, kBurnIt, who, true, false);          // :416
QSetRect(&bounds, 0, 0, 7, 48);                                 // :417  the body
QOffsetRect(&bounds, topLeft.h + 6, topLeft.v + 11);            // :418
hotSpotNumber = AddActiveRect(&bounds, kDissolveIt, who, true, true);  // :421
```

Read carefully: the *hot* part of a tall flame is the **bottom 22 px** of the column
(`kDeadlyFlameHeight - 2` = 24 - 2 = 22, because `top = bottom - 24 + 2`), and the part
above it is a plain lift column. A short flame (`distance <= 24`) is entirely deadly. All
three rects are hard-coded `isOn = true` — flames cannot be switched off.

Per-type variations:

| Type | Column `QSetRect` | Column centring | Body `QSetRect` | Body offset | Lines |
|---|---|---|---|---|---|
| `kTaper` | (0, -distance, 4, 0) | 10 - 2 = 8 | (0, 0, 7, 48) | (+6, +11) | 400-421 |
| `kCandle` | (0, -distance, 4, 0) | 16 - 2 - 2 = 12 (`topLeft.h - 2`) | (0, 0, 8, 20) | (+9, +11) | 425-446 |
| `kStubby` | (0, -distance, 4, 0) | 10 - 2 - 1 = 7 | (0, 0, 15, 26) | (+1, +11) | 450-471 |
| `kTiki` | (0, -distance, 4, 0) | 13 - 2 = 11 | (0, 0, 15, 14) | (+6, +6) | 475-496 |
| `kBBQ` | **(0, -distance, 4, 8)** | 32 - 2 = 30 | (0, 0, 52, 17) | (+6, +8) | 500-519 |

Two exact quirks to preserve:

* `kCandle` applies an extra `- 2` to the horizontal centring
  (`GliderPRO/Sources/ObjectRects.c:430` offsets by `topLeft.h - 2`).
* `kStubby` applies an extra `- 1` (`GliderPRO/Sources/ObjectRects.c:452`).
* `kBBQ`'s column has `bottom = 8` rather than 0
  (`GliderPRO/Sources/ObjectRects.c:500`), so its heat column extends 8 px *below* the
  object's `topLeft.v`.

### 6.6 Directional blowers: `kInvisBlower` and `kLiftArea`

`kInvisBlower` (`GliderPRO/Sources/ObjectRects.c:522`) switches on
`theObject.data.a.vector & 0x0F` and adds exactly one rect. The art is 24 x 24, so the
centring constants are literals `12`:

| `vector & 0x0F` | Direction | `QSetRect(l, t, r, b)` | `QOffsetRect(h, v)` | Action | Lines |
|---:|---|---|---|---|---|
| 1 | up | (0, -distance - 24, 4, 0) | (12 - 2, 24) | `kLiftIt` | 525-533 |
| 2 | right | (0, 0, distance + 24, 16) | (0, 12 - 8) = (0, 4) | `kPushItRight` | 535-543 |
| 4 | down | (0, 0, 4, distance + 24) | (12 - 2, 0) = (10, 0) | `kDropIt` | 545-553 |
| 8 | left | (0, 0, distance + 24, 16) | (-(distance), 4) | `kPushItLeft` | 555-562 |

Note the `+ 24` on the length in every case (the blower's own 24 px is included in the
reach), and that the *up* case is additionally shifted down by 24 so the column starts at
the blower's bottom edge.

`kLiftArea` (`GliderPRO/Sources/ObjectRects.c:590`) is a pure region with no art:

```c
QSetRect(&bounds, 0, 0, theObject.data.a.distance, theObject.data.a.tall * 2);  // :591
QOffsetRect(&bounds, topLeft.h, topLeft.v);                                     // :592
switch (theObject.data.a.vector & 0x0F) {                                       // :595
    case 1: AddActiveRect(..., kLiftIt, ...);       break;                      // :597
    case 2: AddActiveRect(..., kPushItRight, ...);  break;
    case 4: AddActiveRect(..., kDropIt, ...);       break;
    case 8: AddActiveRect(..., kPushItLeft, ...);   break;                       // :615
}
```

`tall * 2` is not a typo — `data.a.tall` is a single byte so it is doubled to reach a useful
pixel height. `distance` is used directly as the *width*. An unrecognised vector produces
no hot spot at all.

`GetObjectRect()` uses the same doubling for `kLiftArea`:
`QSetRect(itsRect, 0, 0, data.a.distance, data.a.tall * 2)`
(`GliderPRO/Sources/ObjectRects.c:64`).

### 6.7 Furniture and solids

```c
case kTable: case kShelf: case kCabinet: case kFilingCabinet:
case kWasteBasket: case kMilkCrate: case kCounter: case kDresser:
case kDeckTable: case kTrunk: case kInvisObstacle:                  // :619-629
bounds = theObject.data.b.bounds;                                   // :630
hotSpotNumber = AddActiveRect(&bounds, kDissolveIt, who, true, true);  // :631
```

The rect is stored **absolutely** in the object record (a `Rect`, not a `Point` + size), so
furniture geometry comes straight off disk. All are scrutinized (glider inset by 5).

Variants:

| Type | Modification | Action | Lines |
|---|---|---|---|
| `kBooks` | `bounds.right -= 2;` | `kDissolveIt` | 634-637 |
| `kStool` | `InsetRect(&bounds, 1, 1); bounds.bottom = bounds.top + kStoolThick (25);` | `kDissolveIt` | 654-658 |
| `kInvisBounce` | none | `kBounceIt` (scrutinize true) | 649-651 |
| `kManhole` | see below | `kIgnoreGround` (scrutinize **false**) | 640-647 |

`kManhole` (`GliderPRO/Sources/ObjectRects.c:640`) is the one-way floor hole:

```c
bounds = theObject.data.b.bounds;
bounds.left  += kGliderWide + 3;      // +51        // :642
bounds.right -= kGliderWide + 3;      // -51        // :643
bounds.top    = kFloorLimit - 1;      // = 311      // :644
bounds.bottom = kTileHigh;            // = 322      // :645
hotSpotNumber = AddActiveRect(&bounds, kIgnoreGround, who, true, false);   // :646
```

The rect is only **11 px tall** (311..322) and is inset by 51 px on each side. For the
123-wide manhole art that leaves `123 - 102 = 21` px of usable width. A verified real
example (Davis Station room 15, section 20.5): stored bounds
`(top=300, left=67, bottom=322, right=190)` → hot rect
`(top=311, left=118, bottom=322, right=139)`.

**Consequence:** to fall through a manhole the glider's `dest` must overlap a 21 x 11 rect
whose top is at 311, i.e. it must already be at the floor. Since the hot spot is *not*
scrutinized, the raw 48-wide `dest` is used, so the effective horizontal window is
`[118 - 48, 139] = [70, 139]` for `dest.left`. The flag it sets, `ignoreGround`, is then
consumed by `CheckEscapeDown` in the same frame.

### 6.8 Prizes

```c
case kRedClock: case kBlueClock: case kYellowClock: case kCuckoo:
case kPaper: case kBattery: case kBands: case kFoil:
case kInvisBonus: case kStar: case kHelium:                           // :661-671
bounds = srcRects[theObject.what];                                    // :672
ZeroRectCorner(&bounds);                                              // :673
QOffsetRect(&bounds, theObject.data.c.topLeft.h,
                     theObject.data.c.topLeft.v);                     // :674
hotSpotNumber = AddActiveRect(&bounds, kRewardIt, who,
                              theObject.data.c.state, false);         // :677
```

Size comes from `srcRects[]` (section 4.1), position from `data.c.topLeft`, `isOn` from
`data.c.state`, **not scrutinized** (raw 48 x 20 glider rect). Note `IsThisValid()` already
excluded collected prizes, so `isOn` here is redundant on room entry but becomes meaningful
when `HandleRewards` sets `who->isOn = false` mid-room.

### 6.9 Grease jars

`kGreaseRt` (`GliderPRO/Sources/ObjectRects.c:681`):

```c
if (theObject.data.c.state)                    // unspilled: it is a prize   // :682
{
    bounds = srcRects[theObject.what]; ZeroRectCorner(&bounds);
    QOffsetRect(&bounds, topLeft.h, topLeft.v);
    hotSpotNumber = AddActiveRect(&bounds, kRewardIt, who, true, false);     // :688
}
else                                           // already spilled            // :691
{
    QSetRect(&bounds, 0, -2, theObject.data.c.length - 5, 0);                // :692
    QOffsetRect(&bounds, 32 - 1, 27);                                        // :693
    QOffsetRect(&bounds, topLeft.h, topLeft.v);                              // :694
    hotSpotNumber = AddActiveRect(&bounds, kSlideIt, who, true, false);      // :696
}
```

`kGreaseLf` (`GliderPRO/Sources/ObjectRects.c:700`) mirrors it:
`QSetRect(&bounds, -theObject.data.c.length + 5, -2, 0, 0)` (`:711`) then
`QOffsetRect(&bounds, 1, 27)` (`:712`).

The spilled slick is **2 px tall** (`top = -2`, `bottom = 0`, offset by 27) and
`length - 5` px long. The `32 - 1` and `1` horizontal offsets are the right/left edges of
the 32-wide jar art, minus/plus one. `RedrawAllGrease()` relies on exactly this: it
repaints any hot spot whose height is exactly 2 (`GliderPRO/Sources/Grease.c:283`).

### 6.10 `kSlider`

```c
QSetRect(&bounds, 0, 0, theObject.data.c.length, 16);                 // :728
QOffsetRect(&bounds, topLeft.h, topLeft.v);
hotSpotNumber = AddActiveRect(&bounds, kSlideIt, who, true, false);   // :731
```

A `length` x 16 permanent slide surface.

### 6.11 `kSparkle` — bounds computed and thrown away

`GliderPRO/Sources/ObjectRects.c:719`-`:725` builds `bounds` from `srcRects[kSparkle]` and
then `break`s **without** calling `AddActiveRect`. Sparkles are decorative; the
`kSparkle` case in `HandleRewards` (`GliderPRO/Sources/Interactions.c:953`) is therefore
dead code reachable only if some other object were mis-linked. Keep both no-ops.

### 6.12 Stairs

`kUpStairs` (`GliderPRO/Sources/ObjectRects.c:734`):

```c
QSetRect(&bounds, 0, 0, 112, 32);                                     // :735
QOffsetRect(&bounds, theObject.data.d.topLeft.h,
                     theObject.data.d.topLeft.v);                     // :736
hotSpotNumber = AddActiveRect(&bounds, kMoveItUp, who, true, false);  // :739
```

`kDownStairs` (`GliderPRO/Sources/ObjectRects.c:742`):

```c
QSetRect(&bounds, -80, -56, 0, 0);                                    // :743
QOffsetRect(&bounds, srcRects[kDownStairs].right, 170);               // :744  = (160, 170)
QOffsetRect(&bounds, theObject.data.d.topLeft.h,
                     theObject.data.d.topLeft.v);                     // :745
hotSpotNumber = AddActiveRect(&bounds, kMoveItDown, who, true, false);  // :748
```

So the up-stairs trigger is a 112 x 32 rect at the object's top-left, and the down-stairs
trigger is an 80 x 56 rect whose **bottom-right** corner is at `topLeft + (160, 170)`, i.e.
spanning `topLeft + (80..160, 114..170)`. Neither is scrutinized. Both actions additionally
require `GliderInRect` (full containment) at interaction time, which is the real gate.

`srcRects[kDownStairs].right` is 160 (section 4.1) — hard-code it, do not recompute.

Two helper functions clip the stair animations:

* `GetUpStairsRightEdge()` (`GliderPRO/Sources/ObjectRects.c:1135`): default
  `rightEdge = kRoomWide` (512) (`:1141`); scans all 24 slots of the current room for a
  `kDownStairs` (`:1149`) and if found sets
  `rightEdge = thisObject.data.d.topLeft.h + srcRects[kDownStairs].right - 1;` (`:1151`),
  i.e. `topLeft.h + 159`.
* `GetDownStairsLeftEdge()` (`GliderPRO/Sources/ObjectRects.c:1163`): default
  `leftEdge = 0` (`:1169`); scans for a `kUpStairs` (`:1177`) and sets
  `leftEdge = thisObject.data.d.topLeft.h + 1;` (`:1179`).

These feed `rightClip`/`leftClip` in `StartGliderGoingDownStairs`
(`GliderPRO/Sources/Modes.c:150`) — cosmetic, but a port that clips differently will show
the glider drawn over the wrong stair banister.

### 6.13 Mailboxes

| Type | Guard | `QSetRect(l, t, r, b)` | `QOffsetRect(h, v)` | then | Action | Lines |
|---|---|---|---|---|---|---|
| `kMailboxLf` | `data.d.who != 255` | (-72, 0, 0, 40) | (30, 16) | `+ topLeft` | `kMailItLeft` | 751-759 |
| `kMailboxRt` | `data.d.who != 255` | (0, 0, 72, 40) | (79, 16) | `+ topLeft` | `kMailItRight` | 763-771 |

So a left mailbox's catch rect is 72 x 40 spanning `topLeft + (-42..30, 16..56)`, and a
right mailbox's is `topLeft + (79..151, 16..56)`. The art is 94 wide, so the left box's
rect extends 42 px to the *left* of the art and the right box's starts 79 px in (15 px
before the art's right edge) and extends 57 px beyond it. Neither is scrutinized.

`who == 255` means "not linked to anything"; such a mailbox is pure scenery.

### 6.14 Ducts (floor/ceiling transports)

| Type | Guard | `QSetRect(l, t, r, b)` | `QOffsetRect(h, v)` | Action | Lines |
|---|---|---|---|---|---|
| `kFloorTrans` | `data.d.who != 255` | (0, -48, 76, 0) | (-8, `RectTall(&srcRects[kFloorTrans])` = 15) | `kDuctItDown` | 775-783 |
| `kCeilingTrans` | `data.d.who != 255` | (0, 0, 76, 48) | (-8, 0) | `kDuctItUp` | 787-795 |

Both are 76 x 48 and shifted 8 px left of the object's `topLeft` (the art is 56 wide, so the
rect overhangs 8 px each side plus 12 more on the right). The floor duct's rect sits
*above* its `topLeft` but is then pushed down by the art height (15), so it spans
`topLeft.v - 33 .. topLeft.v + 15`.

### 6.15 Doors and windows — the wall-permission rects

All eight are `AddActiveRect(..., kIgnore*Wall, who, true, false)`:

| Type | `QSetRect(l, t, r, b)` | `QOffsetRect(h, v)` | Action | Lines |
|---|---|---|---|---|
| `kDoorInLf` | (0, 0, 16, 240) | (0, 52) | `kIgnoreLeftWall` | 799-805 |
| `kDoorInRt` | (0, 0, 16, 240) | (128, 52) | `kIgnoreRightWall` | 808-814 |
| `kDoorExRt` | (0, 0, 16, 240) | (0, 52) | `kIgnoreRightWall` | 817-823 |
| `kDoorExLf` | (0, 0, 16, 240) | (0, 52) | `kIgnoreLeftWall` | 826-832 |
| `kWindowInLf` | (0, 0, 16, 44) | (0, 96) | `kIgnoreLeftWall` | 835-841 |
| `kWindowInRt` | (0, 0, 16, 44) | (4, 96) | `kIgnoreRightWall` | 844-850 |
| `kWindowExRt` | (0, 0, 16, 44) | (0, 96) | `kIgnoreRightWall` | 853-859 |
| `kWindowExLf` | (0, 0, 16, 44) | (0, 96) | `kIgnoreLeftWall` | 862-868 |

All then add `data.d.topLeft`. Doors give a 16 x 240 vertical band starting 52 px down;
windows a 16 x 44 band starting 96 px down. Note `kDoorInRt` is additionally offset +128
horizontally (the interior door art is 144 wide, so the permission band sits at its right
edge) and `kWindowInRt` +4.

These rects are what make a door passable at all: they set `ignoreRight`/`ignoreLeft` for
one frame, which `CheckEscapeRight`/`CheckEscapeLeft` then require before letting the glider
cross `kNoRightWallLimit`/`kNoLeftWallLimit` (section 12.4).

Because the band is only 16 px wide and the glider is 48 wide, the glider overlaps it well
before reaching the wall, which is exactly why the design works: the flag is set every
frame while the glider is anywhere near the doorway.

### 6.16 Transporters

`kInvisTrans` (`GliderPRO/Sources/ObjectRects.c:871`):

```c
if (theObject.data.d.who != 255)                                     // :872
{
    QSetRect(&bounds, 0, 0, 64, 32);                                 // :874
    QOffsetRect(&bounds, topLeft.h, topLeft.v);                      // :875
    bounds.bottom = bounds.top + theObject.data.d.tall;              // :878
    bounds.right += (short)theObject.data.d.wide;                     // :879
    hotSpotNumber = AddActiveRect(&bounds, kTransportIt, who, true, false);  // :880
}
```

Note the asymmetry: `tall` **replaces** the height (so the 32 in the `QSetRect` is only a
default that is always overwritten), while `wide` is **added** to the 64. Verified against
a real object (section 20.6): raw `00 7A 00 E3 00 43 05 82 08 02` → `topLeft = (v=122,
h=227)`, `tall = 67`, `where = 1410`, `who = 8`, `wide = 2` → hot rect
`(left=227, top=122, right=227+64+2=293, bottom=122+67=189)`.

`kDeluxeTrans` (`GliderPRO/Sources/ObjectRects.c:884`) packs its size differently:

```c
if (theObject.data.d.who != 255)                                     // :885
{
    wide = (theObject.data.d.tall & 0xFF00) >> 8;                    // :887
    tall =  theObject.data.d.tall & 0x00FF;                          // :888
    QSetRect(&bounds, 0, 0, wide * 4, tall * 4);                     // :889
    QOffsetRect(&bounds, topLeft.h, topLeft.v);                      // :890
    isOn = theObject.data.d.wide & 0x0F;                             // :893
    hotSpotNumber = AddActiveRect(&bounds, kTransportIt, who, isOn, false);  // :894
}
```

So for `kDeluxeTrans` the *same* `short tall` field carries two bytes: high byte = width in
4-px units, low byte = height in 4-px units. And `data.d.wide` is a nibble pair: low nibble
= current on/off state, high nibble = initial state. `SetObjectsToDefaults()` confirms the
nibble split (`GliderPRO/Sources/Play.c:657`-`:661`):

```c
initState = (thisObject->data.d.wide & 0xF0) >> 4;
thisObject->data.d.wide &= 0xF0;
thisObject->data.d.wide += initState;
```

Verified: an observed `data.d.tall` of `0x5138` = (0x51, 0x38) = (81, 56) → 324 x 224 px,
and `data.d.wide` = 0x11 → initial 1, state 1 (section 20.6).

`GetObjectRect()` repeats the same decode at `GliderPRO/Sources/ObjectRects.c:152`.

### 6.17 Switches, triggers, sound triggers

```c
case kLightSwitch: case kMachineSwitch: case kThermostat:
case kPowerSwitch: case kKnifeSwitch: case kInvisSwitch:
case kTrigger:     case kLgTrigger:                                  // :898-905
bounds = srcRects[theObject.what];                                   // :906
ZeroRectCorner(&bounds);                                             // :907
QOffsetRect(&bounds, theObject.data.e.topLeft.h,
                     theObject.data.e.topLeft.v);                    // :908
if ((theObject.what == kTrigger) || (theObject.what == kLgTrigger))   // :911
{
    if (theObject.data.e.where != -1)
        hotSpotNumber = AddActiveRect(&bounds, kTriggerIt, who, true, false);  // :914
}
else if (theObject.data.e.where != -1)
    hotSpotNumber = AddActiveRect(&bounds, kSwitchIt, who, true, false);        // :919
```

An unlinked switch (`where == -1`) is scenery. Sizes come from `srcRects[]`:
`kLightSwitch` 15 x 24, `kMachineSwitch` 16 x 24, `kThermostat` 15 x 24, `kPowerSwitch`
8 x 8, `kKnifeSwitch` 16 x 24, `kInvisSwitch` 12 x 12, `kTrigger` 12 x 12, `kLgTrigger`
48 x 48.

`kSoundTrigger` (`GliderPRO/Sources/ObjectRects.c:923`) is different — its rect is a
literal 48 x 48 and the guard is that the *sound resource loads*:

```c
QSetRect(&bounds, 0, 0, 48, 48);                                     // :924
QOffsetRect(&bounds, data.e.topLeft.h, data.e.topLeft.v);            // :925
if (LoadTriggerSound(theObject.data.e.where) == noErr)               // :926
    hotSpotNumber = AddActiveRect(&bounds, kSoundIt, who, true, false);  // :927
```

(`srcRects[kSoundTrigger]` is 32 x 32 and is *not* used here — the hot rect is 48 x 48.)

### 6.18 Shredder, guitar, outlet, microwave, appliances

`kShredder` (`GliderPRO/Sources/ObjectRects.c:940`):

```c
bounds = srcRects[theObject.what];                    // 73 x 22          // :941
bounds.bottom = bounds.top + kShredderActiveHigh;     // height := 40     // :942
bounds.right += 48;                                   // width  := 121    // :943
ZeroRectCorner(&bounds);                                                  // :944
QOffsetRect(&bounds, data.g.topLeft.h, data.g.topLeft.v);                 // :945
QOffsetRect(&bounds, -24, -36);                                           // :947
hotSpotNumber = AddActiveRect(&bounds, kShredIt, who,
                              theObject.data.g.state, true);              // :948
```

Final rect: 121 x 40 at `topLeft + (-24, -36)`. Note the order — `bounds.right += 48`
happens *before* `ZeroRectCorner`, and since `srcRects[kShredder].left` is 0 that is
equivalent, but the `bounds.bottom = bounds.top + 40` also precedes it, so the 40 survives
zeroing. Scrutinized.

`kGuitar` (`GliderPRO/Sources/ObjectRects.c:952`):

```c
QSetRect(&bounds, 0, 0, 8, 96);                                          // :953
QOffsetRect(&bounds, data.g.topLeft.h + 34, data.g.topLeft.v + 32);      // :954
hotSpotNumber = AddActiveRect(&bounds, kStrumIt, who, true, false);      // :956
```

An 8 x 96 strum strip 34 px in and 32 px down (the strings).

`kOutlet` (`GliderPRO/Sources/ObjectRects.c:959`): `srcRects[kOutlet]` (16 x 24) zeroed and
offset by `data.g.topLeft`, action `kIgnoreIt`, `isOn = data.g.state` (`:965`). The rect
exists only so bands and `SetObjectState` have something to address; the zap damage comes
from `HandleOutlet` via `CheckDynamicCollision`.

`kMicrowave` (`GliderPRO/Sources/ObjectRects.c:969`) adds **two** rects:

```c
bounds = srcRects[theObject.what];  ZeroRectCorner(&bounds);              // :970
QOffsetRect(&bounds, data.g.topLeft.h, data.g.topLeft.v);                 // :972
AddActiveRect(&bounds, kDissolveIt, who, true, true);                     // :975  solid body
bounds.bottom = bounds.top;                                              // :976
bounds.top    = 0;                                                       // :977
hotSpotNumber = AddActiveRect(&bounds, kMicrowaveIt, who, true, true);   // :978
```

The second rect is the **entire column of room above the microwave**: zero height collapsed
to the body's top, then `top` set to 0 — i.e. `(top=0, left=topLeft.h,
bottom=topLeft.v, right=topLeft.h + 92)`. Standing anywhere in that 92-px-wide column above
the microwave triggers it. Both are scrutinized. `hotNum` points at the
`kMicrowaveIt` rect, so `SetObjectState` toggling the microwave toggles the *zapper*, and
its body stays solid.

Ordinary appliances (`GliderPRO/Sources/ObjectRects.c:981`-`:996`) — `kToaster`,
`kMacPlus`, `kTV`, `kCoffee`, `kVCR`, `kStereo`, `kCinderBlock`, `kFlowerBox`, `kCDs` — are
`srcRects[what]` zeroed, offset by `data.g.topLeft`, `kDissolveIt`, `isOn = true`,
scrutinize **true** (`:995`).

`kCustomPict` (`GliderPRO/Sources/ObjectRects.c:998`) adds nothing — decorative only, even
though `GetObjectRect` will size it from its PICT.

### 6.19 Enemies, fish, cobweb

```c
case kBalloon: case kCopterLf: case kCopterRt:
case kDartLf: case kDartRt: case kBall: case kDrip:                      // :1001-1012
bounds = srcRects[what]; ZeroRectCorner(&bounds);
QOffsetRect(&bounds, data.h.topLeft.h, data.h.topLeft.v);
hotSpotNumber = AddActiveRect(&bounds, kIgnoreIt, who, true, false);     // :1013
```

`kFish` (`GliderPRO/Sources/ObjectRects.c:1016`) is identical but `kDissolveIt`,
scrutinize true (`:1022`) — a fish is a solid you can crash into as well as an enemy.

`kCobweb` (`GliderPRO/Sources/ObjectRects.c:1025`):

```c
bounds = srcRects[kCobweb];  ZeroRectCorner(&bounds);   // 54 x 45
QOffsetRect(&bounds, data.h.topLeft.h, data.h.topLeft.v);
InsetRect(&bounds, -24, -10);                                            // :1031
hotSpotNumber = AddActiveRect(&bounds, kWebIt, who, true, true);          // :1032
```

`InsetRect` with negative values **grows** the rect: final size 54 + 48 = 102 wide by
45 + 20 = 65 tall, centred on the art. Scrutinized.

### 6.20 Chimes

```c
case kChimes:                                                            // :1051
numChimes++;                                                             // :1052
bounds = srcRects[kChimes];   // 28 x 74                                 // :1053
ZeroRectCorner(&bounds);                                                 // :1054
QOffsetRect(&bounds, theObject.data.i.bounds.left,
                     theObject.data.i.bounds.top);                       // :1055
hotSpotNumber = AddActiveRect(&bounds, kChimeIt, who, true, false);      // :1058
```

`kChimes` is a `clutterType`, so its position comes from `data.i.bounds.left/.top` rather
than a `Point`. `numChimes` is a global counter used by `StrikeChime()`.

All other clutter types (`GliderPRO/Sources/ObjectRects.c:1035`-`:1049`) and all light
types (`:930`-`:938`) `break` without adding anything.

### 6.21 `GetObjectRect` vs `CreateActiveRects`

`GetObjectRect()` (`GliderPRO/Sources/ObjectRects.c:32`) returns an object's *drawn* rect,
not its hot rect. It is used by the editor, by `StartGliderMailingIn`
(`GliderPRO/Sources/Modes.c:170`) to find the destination mailbox, and by the grease setup.
The two functions disagree for most types, so never substitute one for the other. Notable
`GetObjectRect` behaviours:

* blowers: `*itsRect = srcRects[who->what]; ZeroRectCorner; QOffsetRect(topLeft)`
  (`:43`-`:61`);
* `kLiftArea`: `QSetRect(itsRect, 0, 0, data.a.distance, data.a.tall * 2)` (`:64`) —
  followed by **unreachable dead code** at `:68`-`:71` (statements after the `break`);
* furniture: `*itsRect = who->data.b.bounds` (`:73`-`:89`);
* `kSlider`: `right = left + data.c.length` (`:112`-`:119`);
* `kInvisTrans`: `bottom = top + data.d.tall; right += (short)data.d.wide;` (`:142`-`:149`);
* `kDeluxeTrans`: the 0xFF00/0x00FF split, x4 (`:152`);
* `kFlourescent`/`kTrackLight`: `right = data.f.length` (`:190`);
* `kCustomPict`: `GetPicture(data.g.height)`, and if the handle is nil it substitutes
  `height = 10000` (`:220`) — a deliberate "you will notice this" sentinel;
* clutter: `data.i.bounds` (`:270`).

### 6.22 Room-relative offsetting for neighbour rooms

`OffsetRectRoomRelative(Rect *theRect, short neighbor)`
(`GliderPRO/Sources/ObjectRects.c:1093`):

```c
QOffsetRect(theRect, playOriginH, playOriginV);              // :1095  room -> screen
switch (neighbor) {                                          // :1097
    kCentralRoom:   nothing
    kEastRoom:      QOffsetRect(theRect,  kRoomWide, 0)
    kWestRoom:      QOffsetRect(theRect, -kRoomWide, 0)
    kNorthRoom:     QOffsetRect(theRect, 0, -kVertLocalOffset)
    kNorthEastRoom: QOffsetRect(theRect,  kRoomWide, -kVertLocalOffset)
    kSouthEastRoom: QOffsetRect(theRect,  kRoomWide,  kVertLocalOffset)
    kSouthRoom:     QOffsetRect(theRect, 0,  kVertLocalOffset)
    kSouthWestRoom: QOffsetRect(theRect, -kRoomWide,  kVertLocalOffset)
    kNorthWestRoom: QOffsetRect(theRect, -kRoomWide, -kVertLocalOffset)
}                                                            // :1130
```

`VerticalRoomOffset()` (`GliderPRO/Sources/ObjectRects.c:1067`) is the same thing for the
vertical component only: `-kVertLocalOffset` for N/NE/NW (`:1075`-`:1079`),
`+kVertLocalOffset` for SE/S/SW (`:1081`-`:1085`), 0 otherwise.

This is drawing-side machinery, but it explains why `AddGrease` receives screen
coordinates: `ObjectDrawAll.c:376` calls `GetObjectRect` then
`OffsetRectRoomRelative(&itsRect, neighbor)` and passes `itsRect.left/.top` straight into
`AddGrease` (`GliderPRO/Sources/ObjectDrawAll.c:376`-`:378`, and the mirror at `:397`-`:399`
for `kGreaseLf`).

---

## 7. Overlap primitives

There are five, and they are *not* interchangeable. All operate on the glider's `dest`
rect in room space.

### 7.1 `SectGlider` — the general "did it touch" test

```c
Boolean SectGlider (gliderPtr thisGlider, Rect *theRect, Boolean scrutinize)   // :101
{
    glideBounds = thisGlider->dest;                                    // :106
    if (thisGlider->mode == kGliderBurning)
        glideBounds.top += 6;                                          // :108
    if (scrutinize) {
        glideBounds.left   += 5;                                       // :112
        glideBounds.top    += 5;                                       // :113
        glideBounds.right  -= 5;                                       // :114
        glideBounds.bottom -= 5;                                       // :115
    }
    if      (theRect->bottom < glideBounds.top)    itHit = false;       // :118
    else if (theRect->top    > glideBounds.bottom) itHit = false;       // :120
    else if (theRect->right  < glideBounds.left)   itHit = false;       // :122
    else if (theRect->left   > glideBounds.right)  itHit = false;       // :124
    else                                           itHit = true;       // :127
    return (itHit);
}
```

**This is an inclusive test.** `theRect->bottom == glideBounds.top` returns *hit*.
QuickDraw's `SectRect` treats rects as half-open and would return *no intersection* for
touching edges. A Go port that writes `a.Max.Y > b.Min.Y && ...` (Go's `image.Rectangle`
`Overlaps`, which is exclusive) will produce a **1-pixel-narrower** hitbox on all four
sides. Write the four comparisons exactly as above.

Two modifiers:

* **Burning glider**: `top += 6` unconditionally. A burning glider is 26 tall (vs 20), so
  this shaves the extra 6 px off the top, leaving the same 20 px collision height while the
  art is taller.
* **Scrutinize**: shrinks the glider by 5 px on *every* side, giving a 38 x 10 core (or
  38 x 16 for a burning glider before the `+6`, i.e. 38 x 10 after). Used for solids
  (`kDissolveIt`), fan blades, shredders, microwaves, cobwebs and fish — i.e. things that
  should not kill you on a graze. **Note the ordering**: the burning adjustment is applied
  first, then the inset, so a burning glider tested with `scrutinize` has
  `top = dest.top + 11`.

Callers of `SectGlider`: `CheckForHotSpots` (`GliderPRO/Sources/Interactions.c:1639`,
`:1657`, `:1679`), `FlagStillOvers` (`:1723`), `CheckDynamicCollision`
(`GliderPRO/Sources/Dynamics.c:42`, always with `scrutinize = true`).

### 7.2 `GliderInRect` — strict containment

```c
Boolean GliderInRect (gliderPtr thisGlider, Rect *theRect)             // :134
{
    glideBounds = thisGlider->dest;                                    // :138
    if      (glideBounds.top    < theRect->top)    return false;       // :140
    else if (glideBounds.bottom > theRect->bottom) return false;       // :142
    else if (glideBounds.left   < theRect->left)   return false;       // :144
    else if (glideBounds.right  > theRect->right)  return false;       // :146
    else                                           return true;
}
```

The **entire** 48 x 20 glider must be inside the rect. No burning adjustment, no
scrutinize. This is the gate for every *transition* action — stairs, mailboxes, ducts,
transporters, microwave, web — because you should not be teleported while half-way out of
a doorway. It is also why the mailbox catch rect must be at least 72 x 40 and the duct rect
76 x 48: they need slack around a 48 x 20 glider.

Because both edges are compared with strict `<`/`>`, a rect exactly 48 x 20 works
(equality passes).

### 7.3 `GliderHitTop` — the foil "did I land on top of it" test

```c
Boolean GliderHitTop (gliderPtr thisGlider, Rect *theRect)             // :54
{
    glideBounds.left   = thisGlider->dest.left   + 5;                  // :60
    glideBounds.top    = thisGlider->dest.top    + 5;                  // :61
    glideBounds.right  = thisGlider->dest.right  - 5;                   // :62
    glideBounds.bottom = thisGlider->dest.bottom - 5;                   // :63

    glideBounds.left  -= thisGlider->wasHVel;                          // :65
    glideBounds.right -= thisGlider->wasHVel;                          // :66

    ... same four inclusive comparisons as SectGlider ...              // :68-:77

    if (!hitTop)                                                        // :79
    {
        PlayPrioritySound(kFoilHitSound, kFoilHitPriority);            // :81
        foilTotal--;                                                   // :82
        if (foilTotal <= 0) StartGliderFoilLosing(thisGlider);          // :84
        glideBounds.left  += thisGlider->wasHVel;                      // :86
        glideBounds.right += thisGlider->wasHVel;                      // :87
        if (thisGlider->hVel > 0)
            offset = 2 + glideBounds.right - theRect->left;             // :89
        else
            offset = 2 + glideBounds.left  - theRect->right;            // :91
        thisGlider->hVel = -thisGlider->hVel - offset;                  // :93
    }
    return (hitTop);
}
```

The semantics are counter-intuitive and worth spelling out:

* The glider rect is inset by 5 (same as `scrutinize`) and then **rewound horizontally** by
  `wasHVel` — i.e. moved back to where it was *before* this frame's horizontal step.
* If the rewound rect still overlaps the obstacle, the collision must have come from
  *vertical* motion → `hitTop = true` → the caller (`kDissolveIt`) treats it as a fatal
  landing-on-top, killing even a foil-armoured glider
  (`GliderPRO/Sources/Interactions.c:1223`-`:1227`).
* If the rewound rect does *not* overlap, the collision came from horizontal motion → foil
  absorbs it. One foil unit is spent, and the glider is bounced: `hVel` is reversed *and*
  pushed out by `offset`, where `offset` is the penetration depth plus 2.

`offset` uses the *un*-rewound (`+= wasHVel`) inset rect and the **current** `hVel` sign,
while the reversal uses `hVel` (not `wasHVel`). If `hVel > 0` (moving right),
`offset = 2 + gliderRight - rectLeft` (positive penetration) and
`hVel = -hVel - offset` — so a rightward-moving glider ends up with a *more negative*
velocity than a pure reflection, guaranteeing separation. Symmetrically for leftward.

The `+5` insets mean `offset` is computed from the *inset* edge, so the glider ends the next
frame with a 5-px gap plus 2, i.e. it visually separates by 7 px. Preserve the constants.

### 7.4 `BounceGlider` — `kInvisBounce`

```c
void BounceGlider (gliderPtr thisGlider, Rect *theRect)                // :154
{
    glideBounds = thisGlider->dest;                                    // :158
    if ((theRect->right - glideBounds.left) < (glideBounds.right - theRect->left))
        thisGlider->hVel = theRect->right - glideBounds.left;          // :160
    else
        thisGlider->hVel = theRect->left  - glideBounds.right;         // :162
    if (foilTotal > 0) PlayPrioritySound(kFoilHitSound, ...);           // :164
    else               PlayPrioritySound(kHitWallSound, ...);           // :166
}
```

This does **not** reflect velocity — it *assigns* the exact gap needed to push the glider
completely out of the rect on the nearer side, so next frame's `MoveGlider` teleports it
clear in one step. The choice of side is "whichever escape distance is smaller". Note both
expressions use the raw `dest` (no inset, no rewind).

Because `hVel` is later clamped to ±`kMaxHVel` (16) in `MoveGlider`
(`GliderPRO/Sources/Player.c:96`, `:113`), a bounce demanding more than 16 px of separation
is silently truncated and the glider can remain embedded — and will bounce again next
frame. That is original behaviour.

### 7.5 `IsRectLeftOfRect`

See section 1.5. Used only by `CheckDynamicCollision`.

### 7.6 The `stillOver` latch

`hotObject.stillOver` implements "fire once per entry". Its lifecycle:

| Event | Effect |
|---|---|
| hot spot created | `stillOver = false` (`GliderPRO/Sources/ObjectRects.c:287`) |
| `CheckForHotSpots` finds **no** overlap | `stillOver = false` (`GliderPRO/Sources/Interactions.c:1675`, `:1683`) |
| `kStrumIt`, `kChimeIt`, `kSoundIt` fire | set `stillOver = true` themselves (`:1347`, `:1598`, `:1619`) |
| `HandleSwitches` runs | early-returns if `stillOver`, sets it `true` at the end (`:990`, `:1154`) |
| `ArmTrigger` runs | early-returns if `stillOver`, sets it `true` (`GliderPRO/Sources/Triggers.c:38`, `:54`) |
| `HandleMicrowaveAction` runs | early-returns if `stillOver` (`:1164`) but **never sets it** |
| `kDuctItUp` in one-player | sets `stillOver = true` (`:1576`) |
| glider enters the room | `FlagStillOvers(thisGlider)` pre-latches every overlapping hot spot (`:1715`) |

`FlagStillOvers` exists so that a glider that materialises *inside* a switch or trigger does
not instantly fire it. It is the only place `stillOver` is set for a rect the glider has
not "entered".

**Two-player subtlety** (`GliderPRO/Sources/Interactions.c:1674`): in two-player mode
`stillOver` is only cleared when *neither* glider overlaps the rect. So one player standing
on a switch prevents the other from re-triggering it.

**Bug to preserve or fix consciously:** `HandleMicrowaveAction` checks `stillOver` but never
sets it (`GliderPRO/Sources/Interactions.c:1164`-`:1194`). The only thing that sets it for
a microwave rect is `FlagStillOvers` on room entry, so a microwave zaps **every frame** the
glider is fully inside its column, not once. Given `kills` clears the affected supply to 0
the repetition is invisible except for the repeated `kMicrowavedSound`.

---

## 8. The hot-spot sweep

`CheckForHotSpots()` (`GliderPRO/Sources/Interactions.c:1627`):

```
for (i = 0; i < nHotSpots; i++)                        // :1632   ascending index order
    if (!hotSpots[i].isOn) continue;                   // :1634   inactive rects are invisible

    if (twoPlayerGame) {
        hitObject = false;                             // :1638
        if (SectGlider(&theGlider,  &hotSpots[i].bounds, hotSpots[i].doScrutinize)) {
            if (onePlayerLeft) { if (playerDead == kPlayer2) { HandleHotSpotCollision(&theGlider,  &hotSpots[i], i); hitObject = true; } }
            else               {                              HandleHotSpotCollision(&theGlider,  &hotSpots[i], i); hitObject = true;   }
        }                                              // :1639-:1655
        if (SectGlider(&theGlider2, &hotSpots[i].bounds, hotSpots[i].doScrutinize)) {
            if (onePlayerLeft) { if (playerDead == kPlayer1) { HandleHotSpotCollision(&theGlider2, &hotSpots[i], i); hitObject = true; } }
            else               {                              HandleHotSpotCollision(&theGlider2, &hotSpots[i], i); hitObject = true;   }
        }                                              // :1657-:1673
        if (!hitObject) hotSpots[i].stillOver = false;  // :1674
    } else {
        if (SectGlider(&theGlider, &hotSpots[i].bounds, hotSpots[i].doScrutinize))
            HandleHotSpotCollision(&theGlider, &hotSpots[i], i);        // :1681
        else
            hotSpots[i].stillOver = false;                              // :1683
    }
```

Properties a port must reproduce:

1. **Ascending index order, single pass, no early exit.** Every active overlapping hot spot
   is processed every frame. There is no "first hit wins" and no sorting by distance.
2. **Effects accumulate within a frame.** `kPushItLeft`/`kPushItRight` use `+=`
   (`GliderPRO/Sources/Interactions.c:1211`, `:1215`), so two fans blowing into each other
   cancel. `kLiftIt`/`kDropIt` use `=` (`:1203`, `:1207`), so the *last* vent in index order
   wins outright, and a `kDropIt` after a `kLiftIt` completely overrides it.
3. `index` (`i`) is passed to `HandleHotSpotCollision` and stored in `activeRectEscaped` for
   the two-player transport/mail/duct handshake, so hot-spot indices are part of the
   protocol.
4. In two-player mode **both gliders are tested against the same rect in the same
   iteration**, glider 1 first. A rect can therefore fire twice in one frame.
5. When `onePlayerLeft` is set, the dead player is skipped — note the inverted-looking
   guards (`playerDead == kPlayer2` guards *glider 1*'s handling).

---

## 9. The 28 hot-spot actions

Action codes from `GliderPRO/Headers/GliderDefines.h:282`-`:309`:

| Value | Name | Value | Name |
|---:|---|---:|---|
| 0 | `kIgnoreIt` | 14 | `kSlideIt` |
| 1 | `kLiftIt` | 15 | `kTransportIt` |
| 2 | `kDropIt` | 16 | `kIgnoreLeftWall` |
| 3 | `kPushItLeft` | 17 | `kIgnoreRightWall` |
| 4 | `kPushItRight` | 18 | `kMailItLeft` |
| 5 | `kDissolveIt` | 19 | `kMailItRight` |
| 6 | `kRewardIt` | 20 | `kDuctItDown` |
| 7 | `kMoveItUp` | 21 | `kDuctItUp` |
| 8 | `kMoveItDown` | 22 | `kMicrowaveIt` |
| 9 | `kSwitchIt` | 23 | `kIgnoreGround` |
| 10 | `kShredIt` | 24 | `kBounceIt` |
| 11 | `kStrumIt` | 25 | `kChimeIt` |
| 12 | `kTriggerIt` | 26 | `kWebIt` |
| 13 | `kBurnIt` | 27 | `kSoundIt` |

`HandleHotSpotCollision(gliderPtr thisGlider, hotPtr who, short index)` is a single switch
on `who->action` (`GliderPRO/Sources/Interactions.c:1200`). Action 0 (`kIgnoreIt`) has no
case at all — it falls out of the switch doing nothing.

Interaction-layer constants (`GliderPRO/Sources/Interactions.c:13`-`:19`):

| Constant | Value |
|---|---:|
| `kFloorVentLift` | -6 |
| `kCeilingVentDrop` | 8 |
| `kFanStrength` | 12 |
| `kBatterySupply` | 50 |
| `kHeliumSupply` | 150 |
| `kBandsSupply` | 8 |
| `kFoilSupply` | 8 |

### 9.1 `kLiftIt` (1) / `kDropIt` (2)

```c
case kLiftIt:  thisGlider->vDesiredVel = kFloorVentLift;   /* -6 */   break;  // :1203
case kDropIt:  thisGlider->vDesiredVel = kCeilingVentDrop; /*  8 */   break;  // :1207
```

Plain assignment, not accumulation. `vDesiredVel` is then consumed by `MoveGlider`, which
ramps `vVel` toward it by ±`kVImpulse` (2) and afterwards resets `vDesiredVel = kGravity`
(3) (`GliderPRO/Sources/Player.c:92`). So a floor vent does not teleport the glider up; it
pulls `vVel` toward -6 at 2 px/frame², and the instant the glider leaves the column gravity
resumes pulling toward +3.

`kCeilingVentDrop` (8) is stronger than gravity (3), so a ceiling vent actively pushes down
faster than free fall.

### 9.2 `kPushItLeft` (3) / `kPushItRight` (4)

```c
case kPushItLeft:  thisGlider->hDesiredVel += -kFanStrength;  break;   // :1211
case kPushItRight: thisGlider->hDesiredVel +=  kFanStrength;  break;   // :1215
```

**Accumulating**, and accumulating *on top of* whatever `GetInput` already put in
`hDesiredVel` this frame (thrust is ±`kNormalThrust` = 5, or ±`kHyperThrust` = 8 with a
battery — `GliderPRO/Sources/Input.c:14`-`:15`). So flying right (+5) into a leftward fan
(-12) yields `hDesiredVel = -7`, and `MoveGlider` ramps `hVel` toward -7 at 2/frame.

Two overlapping fans of the same direction give ±24, which after `MoveGlider`'s clamp to
±`kMaxHVel` (16) caps out at 16 px/frame.

### 9.3 `kDissolveIt` (5) — the generic solid

```c
case kDissolveIt:                                                     // :1218
if (thisGlider->mode != kGliderFadingOut)                             // :1219
{
    if ((foilTotal > 0) || (thisGlider->mode == kGliderLosingFoil))    // :1221
    {
        if (GliderHitTop(thisGlider, &(who->bounds)))                 // :1223
        {
            StartGliderFadingOut(thisGlider);                          // :1225
            PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
        }
        else
        {
            if (foilTotal > 0) {                                       // :1230
                foilTotal--;
                if (foilTotal <= 0) StartGliderFoilLosing(thisGlider);
            }
        }
    }
    else
    {
        StartGliderFadingOut(thisGlider);                              // :1240
        PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
    }
}
break;
```

Semantics:

* **Without foil: instant death.** Touching *any* solid — table, shelf, cabinet, appliance,
  fan blade, flame body, fish, microwave body — kills. This is the core of the game.
* **With foil**: the collision is survivable **only if it was a horizontal hit** (see 7.3);
  a vertical hit (landing on top, or rising into the underside) is fatal even with foil.
  Note `GliderHitTop` *also* decrements `foilTotal` and applies the bounce internally, so
  the `foilTotal--` at `:1232` is a **second** decrement in the horizontal case: a
  foil-absorbed side impact costs **two** foil units, not one. (`GliderHitTop` decrements at
  `:82`, and the caller decrements again at `:1232`.) This is almost certainly unintended
  but it is the shipped behaviour and it halves the effective value of `kFoilSupply` = 8 to
  four side impacts.
* `mode == kGliderLosingFoil` counts as armoured even with `foilTotal == 0`, so the
  foil-removal animation grants brief invulnerability; during it no decrement occurs
  (guarded by `if (foilTotal > 0)` at `:1230`).
* Already-dying gliders (`kGliderFadingOut`) are skipped entirely.

Because `kDissolveIt` rects are all created with `scrutinize = true`, the effective glider
box is 38 x 10 — a 5-px graze on any side is free.

### 9.4 `kRewardIt` (6)

`HandleRewards(thisGlider, who)` — section 10.

### 9.5 `kMoveItUp` (7) — up stairs

```c
case kMoveItUp:                                                        // :1250
if (!thisGlider->heldRight && GliderInRect(thisGlider, &who->bounds))   // :1251
{
    if (thisGlider->mode == kGliderBurning) { wasMode = 0; StartGliderFadingOut; sound; }  // :1253
    else if ((twoPlayerGame) && (!onePlayerLeft))
    {
        if (otherPlayerEscaped == kNoOneEscaped) {                      // :1261
            if ((mode != kGliderGoingUp) && (mode != kGliderInLimbo)) {
                otherPlayerEscaped = kPlayerEscapingUpStairs;           // :1266
                RefreshScoreboard(kEscapedTitleMode);
                StartGliderGoingUpStairs(thisGlider);                   // :1268
            }
        }
        else if (otherPlayerEscaped == kPlayerEscapedUpStairs) {        // :1271
            if ((mode != kGliderGoingUp) && (mode != kGliderInLimbo))
                StartGliderGoingUpStairs(thisGlider);                   // :1276
        }
    }
    else StartGliderGoingUpStairs(thisGlider);                          // :1281
}
break;
```

Geometric precondition: the **whole** glider inside the 112 x 32 stair rect **and**
`heldRight == false`. `heldRight` is set by `GetInput` whenever the right key is down
(`GliderPRO/Sources/Input.c`), so *holding right cancels the stairs* — that is how the
player walks past an up-staircase instead of climbing it.

Symmetrically `kMoveItDown` requires `!heldLeft` (`GliderPRO/Sources/Interactions.c:1286`)
and full containment of the 80 x 56 rect.

A burning glider that reaches a staircase dies instead (`wasMode = 0` first so the burn
countdown does not also fire).

### 9.6 `kMoveItDown` (8) — down stairs

Identical structure with `kPlayerEscapingDownStairs` / `kPlayerEscapedDownStairs`,
`kGliderGoingDown`, `StartGliderGoingDownStairs`
(`GliderPRO/Sources/Interactions.c:1285`-`:1318`).

`StartGliderGoingDownStairs` additionally sets `rightClip = GetUpStairsRightEdge();`
(`GliderPRO/Sources/Modes.c:150`).

### 9.7 `kSwitchIt` (9)

`HandleSwitches(who)` — section 11.

### 9.8 `kShredIt` (10)

```c
case kShredIt:                                                         // :1324
if ((thisGlider->mode != kGliderShredding) &&
        (GliderInRect(thisGlider, &who->bounds)))                      // :1325
{
    if ((foilTotal > 0) || (thisGlider->mode == kGliderLosingFoil))     // :1328
    {
        PlayPrioritySound(kFoilHitSound, kFoilHitPriority);
        if (foilTotal > 0) { foilTotal--; if (foilTotal <= 0) StartGliderFoilLosing(...); }
    }
    else
        FlagGliderShredding(thisGlider, &who->bounds);                  // :1339
}
break;
```

Requires **full containment** in the 121 x 40 shredder rect. Foil grants complete immunity
(costing 1 unit per frame while inside). Without foil, `FlagGliderShredding`
(`GliderPRO/Sources/Modes.c:366`) begins the shred animation:

```c
PlayPrioritySound(kCaughtFireSound, ...);                              // Modes.c:368
thisGlider->dest.left  = bounds->left + 36;                            // Modes.c:369
thisGlider->dest.right = thisGlider->dest.left + kGliderWide;          // Modes.c:370
thisGlider->dest.bottom = thisGlider->dest.top + kGliderHigh;          // Modes.c:371
... whole-rect union ...
thisGlider->mode = kGliderShredding;                                   // Modes.c:385
hVel = vVel = hDesiredVel = vDesiredVel = 0;                           // Modes.c:396-:399
thisGlider->frame = bounds->bottom - 3;                                // Modes.c:400
thisGlider->tipped = false;                                            // Modes.c:401
```

Note the two magic numbers: the glider is **snapped to `shredderRect.left + 36`**
horizontally, and `frame` is initialised to `shredderRect.bottom - 3` (used as the target Y
for the descent animation). `kShredderCountdown` is -68
(`GliderPRO/Sources/Player.c:17`).

### 9.9 `kStrumIt` (11)

```c
case kStrumIt:
if (!who->stillOver) { PlayPrioritySound(kChordSound, kChordPriority); who->stillOver = true; }
break;                                                                 // :1343-:1349
```

Pure sound, one-shot per entry. Guitar strings are not solid.

### 9.10 `kTriggerIt` (12) / `kLgTrigger`

```c
case kTriggerIt:
case kLgTrigger:                                                       // :1352
ArmTrigger(who);
break;
```

**Note the bug-shaped constant reuse**: the second `case` label is `kLgTrigger`, which is
the *object type code* 0x48 = 72, not an action code. Because no hot spot is ever created
with `action = 72`, that label is unreachable and harmless — but a Go port using a typed
enum for actions will not even compile it. Drop the second label; the `kLgTrigger`
*object* correctly produces `action = kTriggerIt` at
`GliderPRO/Sources/ObjectRects.c:914`.

`ArmTrigger` (`GliderPRO/Sources/Triggers.c:34`):

```
 1. if (who->stillOver) return;                                        // Triggers.c:38
 2. where = FindEmptyTriggerSlot();     // first i in 0..15 with !armed // Triggers.c:41, :59
 3. if (where != -1) {
 4.     whoLinked = who->who;                                          // Triggers.c:45
 5.     triggers[where].room   = masterObjects[whoLinked].roomLink;     // Triggers.c:46
 6.     triggers[where].object = masterObjects[whoLinked].objectLink;   // Triggers.c:47
 7.     triggers[where].index  = whoLinked;                            // Triggers.c:48
 8.     triggers[where].timer  = masterObjects[whoLinked].theObject.data.e.delay * 3;  // Triggers.c:49
 9.     triggers[where].what   = masterObjects[triggers[where].object].theObject.what; // Triggers.c:50
10.     triggers[where].armed  = true;
11. }
12. who->stillOver = true;                                             // Triggers.c:54
```

`kMaxTriggers` = 16 (`GliderPRO/Sources/Triggers.c:12`). **Timer is `delay * 3` frames**,
i.e. `delay` is in tenths of a second at 30 fps. A `delay` of 0 gives `timer = 0`, which
fires on the very next `HandleTriggers` call (the check is `timer--` then `<= 0`, so it
fires one frame later).

Line 9 is dangerous: `triggers[where].object` is an *object slot number* (0..23) but it is
used to index `masterObjects[]`. Since `masterObjects[0..23]` are the central room's
objects, this happens to read the central room's slot-`objectLink` object rather than the
linked room's. `what` is only used for informational purposes in this version, so the
mis-index is latent.

`HandleTriggers()` (`GliderPRO/Sources/Triggers.c:79`) decrements every armed timer once per
frame and calls `FireTrigger(i)` when it reaches <= 0, first clamping `timer = 0` and
clearing `armed`.

`FireTrigger` (`GliderPRO/Sources/Triggers.c:100`) dispatches on
`masterObjects[localLink].theObject.what` when `localLink != -1`
(`GliderPRO/Sources/Triggers.c:107`):

| Target type | Action | Line |
|---|---|---|
| `kGreaseRt`, `kGreaseLf` | `SetObjectState(..., kForceOn, ...)` then `SpillGrease(dynaNum, hotNum)` | 114-119 |
| six switch types | `TriggerSwitch(dynaNum)` → `HandleSwitches(&hotSpots[who])` | 128 |
| `kSoundTrigger` | `PlayPrioritySound(kChordSound, ...)` (the source comments "Change me") | 132 |
| `kToaster` | `TriggerToast(dynaNum)` | 136 |
| `kGuitar` | `PlayPrioritySound(kChordSound, ...)` | 140 |
| `kCoffee` | `PlayPrioritySound(kCoffeeSound, ...)` | 144 |
| `kOutlet` | `TriggerOutlet(dynaNum)` | 148 |
| `kBalloon` | `TriggerBalloon(dynaNum)` | 152 |
| `kCopterLf`, `kCopterRt` | `TriggerCopter(dynaNum)` | 157 |
| `kDartLf`, `kDartRt` | `TriggerDart(dynaNum)` | 162 |
| `kDrip` | `TriggerDrip(dynaNum)` | 166 |
| `kFish` | `TriggerFish(dynaNum)` | 170 |

When `localLink == -1` (the target is in a room that is not currently loaded) only grease is
handled, via the house record directly (`GliderPRO/Sources/Triggers.c:174`-`:193`) — and
note that branch dereferences `masterObjects[triggeredIs]` with `triggeredIs = -1`
(`GliderPRO/Sources/Triggers.c:178`), reading out of bounds. A Go port must guard this
(section 22).

### 9.11 `kBurnIt` (13)

```c
case kBurnIt:                                                          // :1356
if ((mode != kGliderBurning) && (mode != kGliderFadingOut))            // :1357
{
    if ((foilTotal > 0) || (mode == kGliderLosingFoil))                // :1360
    {
        thisGlider->vDesiredVel = kFloorVentLift;   /* -6 */            // :1362
        if (foilTotal > 0) {
            PlayPrioritySound(kSizzleSound, kSizzlePriority);            // :1365
            foilTotal--;
            if (foilTotal <= 0) StartGliderFoilLosing(thisGlider);
        }
    }
    else
        FlagGliderBurning(thisGlider);                                  // :1372
}
break;
```

Foil turns a flame into a **lift** source (with a sizzle and 1 foil/frame). Without foil,
`FlagGliderBurning` (`GliderPRO/Sources/Modes.c:406`):

```c
#define kFramesToBurn 60                                               // Modes.c:408
thisGlider->dest.right = thisGlider->dest.left + kGliderWide;          // Modes.c:412
thisGlider->dest.top   = thisGlider->dest.bottom - kGliderBurningHigh; // Modes.c:413  (26)
thisGlider->mode = kGliderBurning;                                     // Modes.c:416
src/mask = gliderSrc[25] if facing left else gliderSrc[21];            // Modes.c:417-:426
hVel = vVel = hDesiredVel = vDesiredVel = 0;                           // Modes.c:427-:430
thisGlider->frame = 0;                                                 // Modes.c:431
thisGlider->wasMode = kFramesToBurn;   /* = 60 */                      // Modes.c:432
```

So a burning glider has **60 frames = 2 seconds** to reach a room exit before
`MoveGliderBurning` (`GliderPRO/Sources/Player.c:203`) expires it. While burning:

* the collision box grows to 48 x 26 but `SectGlider` compensates with `top += 6`;
* `GetInput` gives it *only* horizontal thrust — no battery, no bands, no helium, and
  `tipped` is not touched (`GliderPRO/Sources/Input.c:281`);
* every transition action (`kMoveItUp/Down`, `kTransportIt`, `kMailIt*`, `kDuctIt*`,
  `kWebIt`) and every room edge kills it instead of transitioning
  (`GliderPRO/Sources/Interactions.c:698`, `:711`, `:727`, `:740`, `:1253`, `:1288`,
  `:1382`, `:1425`, `:1468`, `:1511`, `:1544`, `:1606`), each first setting
  `wasMode = 0`.

### 9.12 `kSlideIt` (14)

```c
case kSlideIt:
thisGlider->sliding = true;                                            // :1377
thisGlider->vVel = who->bounds.top - thisGlider->dest.bottom;          // :1378
break;
```

Two effects. First, `vVel` is assigned the exact gap to stand the glider's bottom edge on
the slick's top edge — a hard snap, not a spring. Second, `sliding = true` which:

* suppresses `CheckRoofCollision` entirely (`GliderPRO/Sources/Interactions.c:455`), so you
  can slide off a roof;
* is consumed and cleared by `MoveGliderNormal` (`GliderPRO/Sources/Player.c:155`-`:159`
  and `:177`-`:181`), where it disables the normal ground handling for one frame.

`sliding` is set **only** here. It is cleared in `MoveGliderNormal` and in `InitGlider`
(`GliderPRO/Sources/Play.c:364`).

### 9.13 `kTransportIt` (15)

Full quote at `GliderPRO/Sources/Interactions.c:1381`-`:1414`. Preconditions:

* not burning (burning → death, `:1382`);
* `GliderInRect` — the whole glider inside the transporter rect (`:1388`);
* `mode != kGliderTransporting` and `mode != kGliderFadingOut` (`:1389`).

One player: `StartGliderTransporting(thisGlider, who)` (`:1412`).

Two players: the handshake. If nobody has escaped yet, the first glider to enter stores
`activeRectEscaped = index` (`:1398`) and transports. The second glider may only follow if
`otherPlayerEscaped == kPlayerTransportedOut` **and** `activeRectEscaped == index`
(`:1404`-`:1405`) — i.e. it must use the *same* transporter. That index is a `hotSpots[]`
index, which is why hot-spot ordering is protocol.

`StartGliderTransporting` (`GliderPRO/Sources/Modes.c:288`) normalises the glider to
48 x 20 at the destination (`:311`-`:312`), rebuilds the shadow (`:313`-`:314`), sets
`mode = kGliderTransporting`, `whole = dest`, `frame = kLastFadeSequence - 1` = 15
(`:315`-`:317`) and picks art from `gliderSrc[fadeInSequence[frame]]`, plus
`kLeftFadeOffset` (7) if facing left (`:318`-`:329`).

### 9.14 `kIgnoreLeftWall` (16) / `kIgnoreRightWall` (17)

```c
case kIgnoreLeftWall:  thisGlider->ignoreLeft  = true;  break;         // :1417
case kIgnoreRightWall: thisGlider->ignoreRight = true;  break;         // :1421
```

Set every frame the glider overlaps a door/window band; cleared at the very end of
`HandleGlider` (section 15).

### 9.15 `kMailItLeft` (18) / `kMailItRight` (19)

Preconditions for `kMailItLeft` (a mailbox whose slot **opens to the right**,
`GliderPRO/Sources/Interactions.c:1424`):

1. not burning (`:1425`);
2. `GliderInRect(thisGlider, &who->bounds)` (`:1431`);
3. `mode != kGliderMailOutRight` (`:1432`);
4. `mode != kGliderMailInLeft` (`:1433`);
5. `mode != kGliderFadingOut` (`:1434`);
6. **facing test**:
   `((facing == kFaceRight) && !tipped) || ((facing == kFaceLeft) && tipped)` (`:1435`).

Condition 6 is "the glider is *travelling* rightward". `tipped` means the glider's thrust
is opposite its facing (`GliderPRO/Sources/Input.c`: `tipped = (facing == kFaceLeft)` when
the right key is pressed, and `tipped = (facing == kFaceRight)` when the left key is
pressed). So `facing XOR tipped` gives the direction of travel. `kMailItRight` inverts the
test (`:1478`-`:1479`).

**You cannot enter a mailbox from behind.** This is the geometric/directional condition most
likely to be missed in a port.

Both then run the two-player handshake with `kPlayerMailedOut` and `activeRectEscaped`
(`:1440`-`:1457`) and finish with `StartGliderMailingIn(thisGlider, &who->bounds, who)`
followed by an explicit `thisGlider->mode = kGliderMailInLeft/Right`
(`:1446`, `:1462`, `:1489`, `:1505`).

`StartGliderMailingIn` (`GliderPRO/Sources/Modes.c:155`):

```c
transRoom   = masterObjects[whoLinked].roomLink;                        // Modes.c:164
objLinked   = masterObjects[whoLinked].objectLink;                      // Modes.c:165
linkedToWhat = WhatAreWeLinkedTo(transRoom, objLinked);                 // Modes.c:166
GetObjectRect(&(*thisHouse)->rooms[transRoom].objects[objLinked], &transRect);  // Modes.c:170
clip     = *bounds;                                                     // Modes.c:174
topSought = bounds->bottom - RectTall(&thisGlider->dest);               // Modes.c:175
clip.top  = topSought;                                                  // Modes.c:176
```

so the entry animation clips the glider to the bottom `kGliderHigh` (20) pixels of the
mailbox rect.

### 9.16 `kDuctItDown` (20) / `kDuctItUp` (21)

`kDuctItDown` (`GliderPRO/Sources/Interactions.c:1510`) requires: not burning,
`GliderInRect`, `mode != kGliderDuctingDown`, `mode != kGliderFadingOut` (`:1517`-`:1519`),
then the `kPlayerDuckedOut` handshake (`:1531`), then
`StartGliderDuctingDown(thisGlider, &who->bounds, who)`.

`kDuctItUp` (`:1543`) additionally requires **`!who->stillOver`** (`:1554`) and, in the
one-player path only, sets `who->stillOver = true` (`:1576`). So a ceiling duct fires once
per entry in one-player mode but (because the two-player path never sets `stillOver`) can
re-fire in two-player mode. Preserve the asymmetry or document the divergence.

`StartGliderDuctingDown` (`GliderPRO/Sources/Modes.c:213`) and `StartGliderDuctingUp`
(`:246`) both centre the glider horizontally in the duct:

```c
clip = *bounds;
leftSought = bounds->left + ((RectWide(bounds) - kGliderWide) / 2);      // Modes.c:238/:271
clip.left  = leftSought;
```

With a 76-wide duct rect: `(76 - 48) / 2 = 14`, so the glider is centred 14 px in. The
division truncates (irrelevant here since 28 is even, but matters if a port changes the
duct width).

### 9.17 `kMicrowaveIt` (22)

```c
case kMicrowaveIt:
if (GliderInRect(thisGlider, &who->bounds))                            // :1582
    HandleMicrowaveAction(who, thisGlider);
break;
```

`HandleMicrowaveAction` (`GliderPRO/Sources/Interactions.c:1159`):

```
 1. if (who->stillOver) return;                                        // :1164
 2. whoLinked = who->who;
 3. if (masterObjects[whoLinked].theObject.data.g.state)                // :1169
 4.     kills = masterObjects[whoLinked].theObject.data.g.byte0;        // :1171
 5.     if (kills & 0x0001) { if (bandsTotal   != 0) { bandsTotal   = 0; killed = true; } }  // :1172
 6.     if (kills & 0x0002) { if (batteryTotal != 0) { batteryTotal = 0; killed = true; } }  // :1178
 7.     if (kills & 0x0004) { if (foilTotal    != 0) { foilTotal    = 0; killed = true;
                                                       StartGliderFoilLosing(thisGlider); } } // :1184
 8.     if (killed) PlayPrioritySound(kMicrowavedSound, kMicrowavedPriority);                 // :1192
```

So `data.g.byte0` is a 3-bit mask: **bit0 = destroy rubber bands, bit1 = destroy battery
(and helium — it is the same counter), bit2 = destroy foil**. Verified real values: 0x07
(all three) in Art Museum room 47 and 0x06 (battery + foil) in CD Demo House room 51
(section 20.7).

Because the microwave's active rect is the whole 92-px column above it, a glider *anywhere*
above the microwave and fully inside the column is zapped. The microwave must be switched
on (`data.g.state`).

### 9.18 `kIgnoreGround` (23)

```c
case kIgnoreGround: thisGlider->ignoreGround = true; break;            // :1587
```

### 9.19 `kBounceIt` (24)

`BounceGlider(thisGlider, &who->bounds)` — section 7.4. Note this fires *every frame* while
overlapping, with no `stillOver` guard and no foil interaction.

### 9.20 `kChimeIt` (25)

```c
case kChimeIt:
if (!who->stillOver) { StrikeChime(); who->stillOver = true; }          // :1594-:1599
break;
```

`StrikeChime()` is at `GliderPRO/Sources/Play.c:791`.

### 9.21 `kWebIt` (26)

```c
case kWebIt:                                                            // :1602
if ((GliderInRect(thisGlider, &who->bounds)) && (mode != kGliderBurning))
    WebGlider(thisGlider, &who->bounds);                                // :1605
else if ((mode == kGliderBurning) && (GliderInRect(thisGlider, &who->bounds)))
{ wasMode = 0; StartGliderFadingOut(thisGlider); sound; }               // :1609-:1611
break;
```

`WebGlider` (`GliderPRO/Sources/Interactions.c:1736`):

```c
#define kKillWebbedGlider 150                                           // :1738
if ((mode == kGliderBurning) && GliderInRect(...)) { death; return; }    // :1741  (unreachable
                                                                        //         from kWebIt)
hDist = ((webBounds->right  - dest.right ) + (webBounds->left - dest.left)) >> 3;   // :1749
vDist = ((webBounds->bottom - dest.bottom) + (webBounds->top  - dest.top )) >> 3;   // :1751
if (thisGlider->hDesiredVel != 0) {
    if (evenFrame) {                                                    // :1756
        thisGlider->hVel = hDist;                                       // :1758
        thisGlider->vVel = vDist;                                       // :1759
        PlayPrioritySound(kWebTwangSound, kWebTwangPriority);
    }
} else {
    thisGlider->hDesiredVel = 0;                                        // :1765
    thisGlider->vDesiredVel = 0;                                        // :1766
}
thisGlider->wasMode++;                                                  // :1769
if (thisGlider->wasMode >= kKillWebbedGlider) { wasMode = 0; StartGliderFadingOut; sound; }  // :1770
```

Mechanics:

* `hDist`/`vDist` are the average of the two edge deltas, **arithmetic-shifted right by 3**.
  `(a + b) >> 3` on negative values rounds *toward negative infinity* in C for
  arithmetic shift (and in Go for signed `>>`), which is **not** the same as `/8`. Use
  `>>` in Go, not division.
* Struggling (`hDesiredVel != 0`, i.e. a direction key held) yanks the glider toward the
  web centre — but only on even frames. On odd frames nothing happens and gravity applies
  normally.
* *Not* struggling zeroes both desired velocities, which makes `MoveGlider` ramp `vVel`
  toward 0 instead of `kGravity` — the glider hangs still in the web.
* `wasMode` is reused as the web countdown and increments every frame regardless: **150
  frames = 5 seconds** in a web is fatal.
* The `kGliderBurning` early-out at `:1741` is dead code because the only caller already
  excludes burning gliders.

`WebGlider` does *not* consult foil at all — foil gives no protection from cobwebs.

### 9.22 `kSoundIt` (27)

```c
case kSoundIt:
if (!who->stillOver) { PlayPrioritySound(kTriggerSound, kTriggerPriority); who->stillOver = true; }
break;                                                                  // :1615-:1621
```

---

## 10. `HandleRewards` — every pickup

`HandleRewards(gliderPtr thisGlider, hotPtr who)`
(`GliderPRO/Sources/Interactions.c:756`). Preamble:

```c
whoLinked = who->who;                                                  // :761
bounds    = who->bounds;                                               // :762
switch (masterObjects[whoLinked].theObject.what)                        // :764
```

Every case is gated on

```c
if (SetObjectState(thisRoomNumber, masterObjects[whoLinked].objectNum, 0, whoLinked))
```

where the `0` is `kToggle` (`GliderPRO/Headers/GliderDefines.h:441`) but the prize branch of
`SetObjectState` ignores the action and unconditionally forces `state = false`, returning
`changed = (previous state == true)` (`GliderPRO/Sources/Objects.c:460`-`:462`). So the guard
means "this prize had not already been taken". Every case then sets `who->isOn = false`
**outside** the guard, so the hot spot is disabled either way.

Score constants (`GliderPRO/Headers/GliderDefines.h:536`-`:541`):

| Constant | Value |
|---|---:|
| `kRoomVisitScore` | 100 |
| `kRedClockPoints` | 100 |
| `kBlueClockPoints` | 300 |
| `kYellowClockPoints` | 500 |
| `kCuckooClockPoints` | 1000 |
| `kStarPoints` | 5000 |

### 10.1 The complete reward table

| `what` | Sound | Flying-point label | Velocity damping | Supply change | Score | Extra | Lines |
|---|---|---:|---|---|---:|---|---|
| `kRedClock` | `kBeepsSound` | 100 | `hVel /= 4; vVel /= 4` | — | +100 | — | 766-780 |
| `kBlueClock` | `kBuzzerSound` | 300 | `/= 4` | — | +300 | — | 782-796 |
| `kYellowClock` | `kDingSound` | 500 | `/= 4` | — | +500 | — | 798-812 |
| `kCuckoo` | `kCuckooSound` | 1000 | `/= 4` | — | +1000 | `StopPendulum` | 814-829 |
| `kPaper` | `kEnergizeSound` | (sparkle) | `/= 2` | `mortals++` (twice in 2P) | — | `QuickGlidersRefresh` | 831-848 |
| `kBattery` | `kEnergizeSound` | (sparkle) | `/= 2` | see 10.3 | — | `QuickBatteryRefresh` | 850-870 |
| `kBands` | `kEnergizeSound` | (sparkle) | `/= 2` | `bandsTotal += 8` (twice in 2P) | — | `QuickBandsRefresh` | 872-889 |
| `kGreaseRt`/`kGreaseLf` | (none) | — | none | — | — | `SpillGrease(dynaNum, hotNum)` | 891-898 |
| `kFoil` | `kEnergizeSound` | (sparkle) | `/= 2` | `foilTotal += 8` (twice in 2P) | — | `StartGliderFoilGoing` | 900-917 |
| `kInvisBonus` | `kBonusSound` | `data.c.points` | `/= 4` | — | `+= points` | **no** `RestoreFromSavedMap` | 919-931 |
| `kStar` | `kEnergizeSound` | (sparkle) | none | `numStarsRemaining--` | +5000 | `StopStar`; `FlagGameOver()` at <= 0 | 933-951 |
| `kSparkle` | — | — | — | — | — | no-op | 953-954 |
| `kHelium` | `kEnergizeSound` | (sparkle) | `/= 2` | see 10.3 | — | `QuickBatteryRefresh` | 956-976 |
| `kSlider` | — | — | — | — | — | no-op | 978-979 |

Notes:

* The four clocks and `kInvisBonus` use `AddFlyingPoint(&bounds, N, hVel / 2, vVel / 2)` and
  then damp the glider to a **quarter** speed; the supply pickups use `AddSparkle(&bounds)`
  and damp to a **half**. `kStar` does not damp at all.
* `hVel /= 4` in C truncates **toward zero** for negative values (C89 was
  implementation-defined but every Mac compiler and the C99 standard truncate toward zero).
  Go's `/` also truncates toward zero, so `hVel /= 4` ports directly. Do **not** use `>> 2`
  here — that would round toward negative infinity and change the result for negative
  velocities.
* `RestoreFromSavedMap(thisRoomNumber, objectNum, false)` erases the prize art. `kInvisBonus`
  skips it (there is nothing drawn) and `kGreaseRt/Lf` skip it (the jar stays visible).
* In two-player mode with both players alive, **every supply pickup is doubled**
  (`GliderPRO/Sources/Interactions.c:842`, `:864`, `:883`, `:911`, `:970`) — the pickup
  benefits both gliders from a single shared counter.

### 10.2 `kStar` and game completion

```c
StopStar(thisRoomNumber, masterObjects[whoLinked].objectNum);           // :941
numStarsRemaining--;                                                   // :942
if (numStarsRemaining <= 0) FlagGameOver();                            // :943
else                        DisplayStarsRemaining();                    // :946
theScore += kStarPoints;   /* 5000 */                                  // :948
```

Collecting the last star ends the game. `numStarsRemaining` is a global counted at house
load.

### 10.3 The battery / helium shared counter

`batteryTotal` is a **signed** counter that encodes two different resources:

* `batteryTotal > 0` — battery charge (used by `DoBatteryEngaged`,
  `GliderPRO/Sources/Input.c:121`, which applies ±`kHyperThrust` = 8 and does
  `batteryTotal--`).
* `batteryTotal < 0` — helium (used by `DoHeliumEngaged`,
  `GliderPRO/Sources/Input.c:160`, which sets `vDesiredVel = -kHeliumLift` = -4 and does
  `batteryTotal++`, i.e. counts *toward* zero).

Pickup arithmetic:

```c
case kBattery:                                                         // :850
if (batteryTotal > 0) batteryTotal += kBatterySupply;   /* +50 */       // :861
else                  batteryTotal  = kBatterySupply;   /* := 50 */     // :863
if ((twoPlayerGame) && (!onePlayerLeft)) batteryTotal += kBatterySupply; // :865

case kHelium:                                                          // :956
if (batteryTotal < 0) batteryTotal -= kHeliumSupply;    /* -150 */      // :967
else                  batteryTotal  = -kHeliumSupply;   /* := -150 */   // :969
if ((twoPlayerGame) && (!onePlayerLeft)) batteryTotal -= kHeliumSupply; // :971
```

So picking up a battery while holding helium **discards the helium** and vice versa, and the
two-player doubling is applied *after* the replace-vs-add decision (so a two-player battery
pickup while holding helium yields exactly 100, not 50).

The microwave zeroes this single counter for both resources
(`GliderPRO/Sources/Interactions.c:1180`).

### 10.4 `kPaper` grants extra lives

```c
mortals++;                                                             // :841
if ((twoPlayerGame) && (!onePlayerLeft)) mortals++;                     // :843
```

`mortals` is the shared life counter, initialised to `kInitialGliders` (doubled in
two-player) in `InitGlider` (`GliderPRO/Sources/Play.c:343`).

---

## 11. Switches, the microwave, and object state

### 11.1 `HandleSwitches` (`GliderPRO/Sources/Interactions.c:985`)

```
 1. if (who->stillOver) return;                                        // :990
 2. whoLinked    = who->who;                                           // :993
 3. roomLinked   = masterObjects[whoLinked].roomLink;                   // :994
 4. objectLinked = masterObjects[whoLinked].objectLink;                 // :995
 5. linkIndex    = masterObjects[whoLinked].localLink;                  // :996
 6. if (SetObjectState(roomLinked, objectLinked,
                       masterObjects[whoLinked].theObject.data.e.type,
                       linkIndex))                                     // :998
 7. {
 8.     newRect = who->bounds;                                         // :1001
 9.     QOffsetRect(&newRect, playOriginH, playOriginV);               // :1002
10.     switch (masterObjects[whoLinked].theObject.what)               // :1003
11.         kLightSwitch   -> kSwitchSound + DrawLightSwitch(&newRect, newState)     // :1007
12.         kMachineSwitch -> kSwitchSound + DrawMachineSwitch(...)                   // :1012
13.         kThermostat    -> kSwitchSound + DrawThermostat(...)                      // :1017
14.         kPowerSwitch   -> kSwitchSound + DrawPowerSwitch(...)                     // :1022
15.         kKnifeSwitch   -> kSwitchSound + DrawKnifeSwitch(...)                     // :1027
16.         kInvisSwitch   -> (nothing, no sound)                                     // :1031
17.     CopyRectBackToWork(&newRect);  AddRectToWorkRects(&newRect);    // :1033
18.     if (linkIndex != -1) switch (masterObjects[linkIndex].theObject.what) { ... }  // :1036
19. }
20. who->stillOver = true;                                             // :1154
```

**`data.e.type` is the switch's *action*, not its art.** It is passed straight to
`SetObjectState` as the `action` argument, so its legal values are `kToggle` (0),
`kForceOn` (1), `kForceOff` (2) (`GliderPRO/Headers/GliderDefines.h:441`-`:443`). A verified
real trigger object has `type = 3` (section 20.8), which is out of range and falls through
every inner `switch (action)` leaving `changed` **uninitialized** — see section 22.

The `linkIndex` dispatch (`GliderPRO/Sources/Interactions.c:1038`-`:1150`):

| Linked object | Effect | Line |
|---|---|---|
| `kRedClock`, `kBlueClock`, `kYellowClock`, `kPaper`, `kBattery`, `kBands`, `kFoil`, `kStar`, `kHelium` | `RestoreFromSavedMap(roomLinked, objectLinked, true); AddSparkle(&bounds);` | 1049-1050 |
| `kCuckoo` | `RestoreFromSavedMap(...); StopPendulum(...)` | 1054-1055 |
| `kGreaseRt`, `kGreaseLf` | `SpillGrease(masterObjects[linkIndex].dynaNum, masterObjects[linkIndex].hotNum)` | 1060 |
| `kInvisBonus`, `kSlider`, `kDeluxeTrans`, `kShredder` | nothing | 1064-1069, 1086 |
| `kSoundTrigger` | `PlayPrioritySound(kTriggerSound, ...)` | 1072 |
| all eight light types | `RedrawRoomLighting()` | 1083 |
| `kToaster` | `ToggleToaster(dynaNum)` | 1090 |
| `kMacPlus` | `ToggleMacPlus(dynaNum)` | 1094 |
| `kGuitar` | `PlayPrioritySound(kChordSound, ...)` | 1098 |
| `kTV` | `ToggleTV(dynaNum)` | 1102 |
| `kCoffee` | `ToggleCoffee(dynaNum)` | 1106 |
| `kOutlet` | `ToggleOutlet(dynaNum)` | 1110 |
| `kVCR` | `ToggleVCR(dynaNum)` | 1114 |
| `kStereo` | `ToggleStereos(dynaNum)` | 1118 |
| `kMicrowave` | `ToggleMicrowave(dynaNum)` | 1122 |
| `kBalloon` | `ToggleBalloon(dynaNum)` | 1126 |
| `kCopterLf`, `kCopterRt` | `ToggleCopter(dynaNum)` | 1131 |
| `kDartLf`, `kDartRt` | `ToggleDart(dynaNum)` | 1136 |
| `kBall` | `ToggleBall(dynaNum)` | 1140 |
| `kDrip` | `ToggleDrip(dynaNum)` | 1144 |
| `kFish` | `ToggleFish(dynaNum)` | 1148 |

**Genuine bug at line 1050:** `AddSparkle(&bounds)` where `bounds` is the *uninitialized*
local `Rect bounds` declared at `GliderPRO/Sources/Interactions.c:987`. It is never assigned
anywhere in `HandleSwitches`. The sparkle therefore appears at a garbage location (in
practice whatever was on the stack). A Go port will get zero-valued `bounds` and draw the
sparkle at (0,0,0,0); the original drew it somewhere random. Neither is "correct"; document
the divergence.

The `Toggle*` functions (`GliderPRO/Sources/Trip.c`) all flip `dinahs[index].active`, with
per-type extras: `ToggleMacPlus` sets `timer = 10`, `ToggleToaster` sets `timer = 40`,
`ToggleStereos` acts only when `timer == 0`. `TriggerSwitch(who)`
(`GliderPRO/Sources/Trip.c:146`) is literally `HandleSwitches(&hotSpots[who])`.
`TriggerOutlet` (`:171`) sets `position = 1; timer = kLengthOfZap (30)`. `TriggerDrip`
(`:188`) clamps `timer` to 7. `TriggerFish` (`:196`) sets `frame = 4`.
`TriggerBalloon`/`TriggerCopter`/`TriggerDart` set `timer = kStartSparkle + 1` = 5.
`UpdateOutletsLighting` (`:235`) stores `nLights` in `dinahs[i].hVel` — a field reused as a
non-velocity.

### 11.2 `SetObjectState` (`GliderPRO/Sources/Objects.c:366`)

Signature `Boolean SetObjectState(short room, short object, short action, short local)`.
Returns "something actually changed". The global `newState` is set as a side effect and read
by `HandleSwitches`' drawing code.

Behaviour by object family:

| Family | `kToggle` | `kForceOn` | `kForceOff` | Extra |
|---|---|---|---|---|
| blowers/vents/fans/`kLiftArea` (`:376`-`:405`) | flip `data.a.state`, `changed = true` | `changed = (state == false)`, set true | `changed = (state == true)`, set false | plays `kBlowerOn`/`kBlowerOff`; syncs `hotSpots[hotNum].isOn = newState` (`:416`) |
| flames (`:420`-`:425`) | — | — | — | `changed = false` — flames cannot be switched |
| furniture, `kManhole`, `kBooks`, `kInvisBounce` (`:428`-`:443`) | — | — | — | `changed = false` |
| prizes + grease + `kSparkle` (`:446`-`:473`) | ignores `action`; always forces `state = false`; `changed = (was true)` | | | syncs `hotSpots[hotNum].isOn = false` only when `room == thisRoomNumber` (`:470`) |
| `kSlider` (`:475`) | — | — | — | falls through with `changed` **uninitialized** |
| stairs/mail/duct/doors/windows/`kInvisTrans` (`:478`-`:493`) | — | — | — | `changed = false` |
| `kDeluxeTrans` (`:496`+) | flip low nibble of `data.d.wide` | force low nibble 1 | force low nibble 0 | `changed` accordingly |
| lights, appliances, enemies | per-family `state` byte, `hotSpots[hotNum].isOn` sync at `:529`, `:625` | | | |

**Bug at `GliderPRO/Sources/Objects.c:465`:** in the prize branch the master-object copy is
updated as

```c
masterObjects[local].theObject.data.a.state = false;
```

but a prize is a `bonusType`, whose `state` lives at **data offset 8**
(`GliderPRO/Headers/GliderStructs.h:28`), while `blowerType.state` is at **offset 7** — which
for a `bonusType` is the *low byte of `short points`*. So collecting an object with this code
path zeroes the low byte of `points` in the master copy instead of clearing `state`.
Consequences: (a) the master copy's `data.c.state` stays `true`; (b) a `kInvisBonus`'s
`points` is corrupted in the master copy. Neither is observable in normal play because
`HandleRewards` reads `points` *before* calling `SetObjectState`
(`GliderPRO/Sources/Interactions.c:920` then `:921`) and `hotSpots[].isOn` is cleared
independently. A Go port should write `data.c.state` (the obvious intent) and note the
divergence.

### 11.3 `SetObjectsToDefaults` (`GliderPRO/Sources/Play.c:603`)

Called on a new game. Resets `visited = false` (`:619`) and per-family `state = initial`:
blowers (`:635`), prizes (`:653`), lights (`:671`), appliances (`:688`), enemies (`:700`).
Two special cases:

* `kDeluxeTrans` (`:657`-`:661`) — the nibble dance quoted in section 6.16.
* `kStereo` (`:676`) — `state = isPlayMusicGame`, i.e. stereos start on only if music is
  enabled.

### 11.4 The `hotNum` / `isOn` coupling

`SetObjectState` writes `hotSpots[masterObjects[local].hotNum].isOn`. Because `hotNum` is
only the *last* rect an object created (section 5.3), toggling:

* a **fan** toggles its wind column, leaving the blade permanently solid;
* a **microwave** toggles its `kMicrowaveIt` column, leaving its body permanently solid;
* a **flame** cannot be toggled at all (`changed = false`).

Any port that stores a *list* of hot-spot indices per object and toggles them all will make
fan blades and microwave bodies vanish. Store only the last index.

---

## 12. Room-boundary resolution

`CheckGliderInRoom(gliderPtr thisGlider)` (`GliderPRO/Sources/Interactions.c:689`).

### 12.1 The mode gate

```c
if ((mode == kGliderNormal) || (mode == kGliderFaceLeft) ||
    (mode == kGliderFaceRight) || (mode == kGliderBurning))            // :691-:694
```

**Only four of the 24 modes are subject to room boundaries.** During stairs, ducts, mail,
transport, fading, foil animations and limbo the glider is free to be outside the room
rectangle — those movers handle their own geometry.

### 12.2 Vertical: one `if / else if / else if` chain

```c
if      (dest.top    < kCeilingLimit /* 8 */)                          // :696
        ... burning ? die : (2P ? CheckEscapeUpTwo : CheckEscapeUp)
else if (dest.bottom > kFloorLimit /* 312 */)                          // :709
        ... burning ? die : (2P ? CheckEscapeDownTwo : CheckEscapeDown)
else if ((thisBackground == kRoof) && (dest.bottom > kRoofLimit /* 122 */))  // :722
        CheckRoofCollision(thisGlider);
```

Note `kRoofLimit` (122) is far above `kFloorLimit` (312), and the roof test is in the
**`else if`** chain, so a glider whose bottom is past 312 gets the *floor* handling, not the
roof handling, even in a `kRoof` room.

### 12.3 Horizontal: a *separate* `if / else if`

```c
if      (dest.left  < leftThresh)                                      // :725
        ... burning ? die : (2P ? CheckEscapeLeftTwo  : CheckEscapeLeft)
else if (dest.right > rightThresh)                                     // :738
        ... burning ? die : (2P ? CheckEscapeRightTwo : CheckEscapeRight)
```

This is *not* chained onto the vertical tests, so **both a vertical and a horizontal
resolution can happen in the same frame**. A glider in a corner gets both `vVel` and `hVel`
rewritten. But left and right are mutually exclusive, and up/down/roof are mutually
exclusive.

Also note the horizontal test compares against `leftThresh`/`rightThresh` (which vary with
room openness — section 13), while the vertical test uses the *fixed* constants
`kCeilingLimit`/`kFloorLimit` and defers the openness decision to the escape functions.
This asymmetry is important: for an open-sided room `leftThresh` is `kNoLeftWallLimit`
(-24), so `CheckEscapeLeft` is not even called until the glider is 24 px outside the room.

### 12.4 Burning gliders die at every boundary

Each of the four branches begins with

```c
if (thisGlider->mode == kGliderBurning)
{
    thisGlider->wasMode = 0;                                           // :700/:713/:729/:742
    StartGliderFadingOut(thisGlider);
    PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
}
```

`wasMode = 0` is essential: `wasMode` is the burn countdown, and zeroing it stops
`MoveGliderBurning` from also expiring the glider.

### 12.5 `CheckEscapeUp` / `CheckEscapeUpTwo`

One-player (`GliderPRO/Sources/Interactions.c:244`):

```
 1. if (topOpen)                                                       // :248
 2.     if (dest.top < kNoCeilingLimit /* -10 */)                      // :250
 3.         MoveRoomToRoom(thisGlider, kAbove);                         // :252
 4.     (else: nothing — the glider keeps rising into the gap)
 5. else if (thisBackground == kDirt)                                   // :255
 6.     leftTile  = dest.left  >> 6;                                    // :257
 7.     rightTile = dest.right >> 6;                                    // :258
 8.     if (leftTile in [0,8) and rightTile in [0,8))                    // :260
 9.         if (thisTiles[leftTile] in {5,6} and thisTiles[rightTile] in {5,6})  // :263
10.             if (dest.top < kNoCeilingLimit) MoveRoomToRoom(kAbove);  // :268
11.         else  vVel = kCeilingLimit - dest.top;                       // :272
12.     else      vVel = kCeilingLimit - dest.top;                       // :275
13. else          vVel = kCeilingLimit - dest.top;                       // :278
```

Two-player (`:171`) is identical except that step 3/10 becomes the handshake:

```
if (otherPlayerEscaped == kNoOneEscaped) {
    otherPlayerEscaped = kPlayerEscapedUp;                              // :181
    RefreshScoreboard(kEscapedTitleMode);
    FlagGliderInLimbo(thisGlider, true);                                // :183
}
else if (otherPlayerEscaped == kPlayerEscapedUp) {
    otherPlayerEscaped = kNoOneEscaped;                                 // :187
    MoveRoomToRoom(thisGlider, kAbove);                                 // :188
}
else {                    // the other player left by a *different* exit
    PlayPrioritySound(kDontExitSound, kDontExitPriority);               // :192
    offset = kNoCeilingLimit - dest.top;                                // :193
    vVel = -vVel + offset;                                              // :194
}
```

### 12.6 `CheckEscapeDown` / `CheckEscapeDownTwo`

One-player (`GliderPRO/Sources/Interactions.c:378`):

```
 1. if (bottomOpen)                                                    // :382
 2.     if (dest.bottom > kNoFloorLimit /* 332 */) MoveRoomToRoom(kBelow);  // :384
 3. else if (thisBackground == kDirt)                                    // :389
 4.     leftTile = dest.left >> 6; rightTile = dest.right >> 6;          // :391
 5.     if (both tiles in [0,8))                                         // :394
 6.         if (thisTiles[leftTile] in {2,3} and thisTiles[rightTile] in {2,3})  // :396
 7.             if (dest.bottom > kNoFloorLimit) MoveRoomToRoom(kBelow);  // :399
 8.         else if (ignoreGround) { if (dest.bottom > kNoFloorLimit) MoveRoomToRoom(kBelow); }  // :404
 9.              else { vVel = kFloorLimit - dest.bottom;
                        StartGliderFadingOut; kFadeOutSound; }           // :411
10.     else (tiles out of range) same as 8/9                            // :417-:430
11. else (not kDirt) same as 8/9                                         // :432-:445
```

So **landing on a solid floor is fatal** (`StartGliderFadingOut`) unless `ignoreGround` was
set this frame by a `kManhole` hot spot, in which case the glider must additionally be past
`kNoFloorLimit` (332) to actually transition. Note step 9 both clamps `vVel` *and* kills —
the clamp is cosmetic (it stops the corpse sinking below the floor).

The two-player version (`:283`) is the same with the `kPlayerEscapedDown` handshake, **except
that the `kDirt` tiles-out-of-range case has no `else` at all** (the `if` at
`GliderPRO/Sources/Interactions.c:315` closes at `:358` and the enclosing `else if` closes at
`:359`). In two-player mode, a glider in a `kDirt` room whose `dest` straddles tile index
< 0 or >= 8 falls through the floor with no clamp and no death. See section 22.

### 12.7 `CheckEscapeLeft` / `CheckEscapeLeftTwo`

One-player (`GliderPRO/Sources/Interactions.c:572`):

```
 1. if (leftThresh == kLeftWallLimit /* 12 */)     // there IS a left wall              // :576
 2.     if (thisGlider->ignoreLeft)                 // a door/window granted permission  // :578
 3.         if (dest.left < kNoLeftWallLimit /* -24 */) MoveRoomToRoom(kToLeft);         // :580
 4.     else {
 5.         PlayPrioritySound(foilTotal > 0 ? kFoilHitSound : kHitWallSound);            // :585
 6.         offset = kLeftWallLimit - dest.left;                                         // :589
 7.         hVel   = -hVel + offset;                                                     // :590
 8.     }
 9. else                                            // no wall at all
10.     MoveRoomToRoom(thisGlider, kToLeft);                                             // :594
```

`CheckEscapeRight` (`:662`) mirrors with `rightThresh == kRightWallLimit` (500),
`ignoreRight`, `dest.right > kNoRightWallLimit` (536) and
`offset = kRightWallLimit - dest.right` (`:679`).

Read step 6/7 carefully. `offset` is **positive** when the glider has penetrated the wall
(`dest.left < 12`), and `hVel = -hVel + offset` therefore both reflects and pushes right.
For example `dest.left = 4`, `hVel = -8`: `offset = 8`, `hVel = 16` — the reflection is
*amplified* by exactly the penetration depth, so the glider is guaranteed to be clear of the
wall after the next `MoveGlider`. This is the same "assign the gap" philosophy as elsewhere,
layered on a reflection.

**Crucially there is no wall clamp when `ignoreLeft` is set but the glider has not yet
reached -24.** Steps 2-3 have no `else`, so while standing in a doorway the glider passes
freely through the region `-24 <= dest.left < 12` with *no* correction. That is the whole
mechanism of walking through a door.

Two-player (`:509`, `:599`) inserts the handshake, and note the two different `offset`s in
the refusal path: it uses `kNoLeftWallLimit - dest.left` (`:533`, `:564`) — the *outer*
limit — rather than `kLeftWallLimit`, because the glider is already outside the room.

### 12.8 The two-player escape protocol

`otherPlayerEscaped` and `activeRectEscaped` are globals
(`GliderPRO/Sources/Interactions.c:42`). The escape codes
(`GliderPRO/Headers/GliderDefines.h:596`-`:608`):

| Constant | Value |
|---|---:|
| `kPlayerIsDeadForever` | -69 |
| `kPlayerMailedOut` | -12 |
| `kPlayerDuckedOut` | -11 |
| `kPlayerTransportedOut` | -10 |
| `kPlayerEscapingDownStairs` | -9 |
| `kPlayerEscapingUpStairs` | -8 |
| `kPlayerEscapedDownStairs` | -7 |
| `kPlayerEscapedUpStairs` | -6 |
| `kPlayerEscapedDown` | -5 |
| `kPlayerEscapedUp` | -4 |
| `kPlayerEscapedLeft` | -3 |
| `kPlayerEscapedRight` | -2 |
| `kNoOneEscaped` | -1 |

The rule is: **both gliders must leave a room the same way before the room changes.** The
first to reach an exit sets `otherPlayerEscaped` to the matching code and goes into limbo
(`FlagGliderInLimbo`, `GliderPRO/Sources/Modes.c:458`, which saves `wasMode = mode`, sets
`mode = kGliderInLimbo`, plays `kFollowSound` if `saidFollow < 3`, and records
`firstPlayer = thisGlider->which`). The second glider may only follow through the *same kind*
of exit, at which point `otherPlayerEscaped` is reset to `kNoOneEscaped` and
`MoveRoomToRoom` runs. A glider that reaches a *different* exit gets `kDontExitSound` and is
bounced back.

For exits identified by a specific object — transporters, mailboxes, ducts — the additional
`activeRectEscaped == index` test (`GliderPRO/Sources/Interactions.c:1405`, `:1452`,
`:1495`, `:1534`, `:1569`) requires the same **hot-spot index**, i.e. the same physical
device.

Note the stairs use two codes each: `kPlayerEscapingUpStairs` (set when the *first* player
starts climbing) and `kPlayerEscapedUpStairs` (set later, presumably when the climb
animation completes, in `Player.c`'s `FinishGliderUpStairs`). `kMoveItUp` tests for
`kPlayerEscapedUpStairs` (`:1271`), not the "escaping" value.

`UndoGliderLimbo` (`GliderPRO/Sources/Modes.c:472`) restores `mode = wasMode` and
`dontDraw = false`, early-returning for the dead player in two-player mode (`:474`).

### 12.9 `MoveRoomToRoom` (`GliderPRO/Sources/Transit.c:151`)

```
 1. HandleRoomVisitation();                                            // Transit.c:155  (score + visited flag)
 2. switch (where)
 3. case kToRight:                                                     // Transit.c:158
 4.     SetMusicalMode(kProdGameScoreMode);                            // Transit.c:159
 5.     2P: UndoGliderLimbo both, InsureGliderFacingRight both         // Transit.c:160-:166
 6.     1P: InsureGliderFacingRight(thisGlider)                        // Transit.c:168
 7.     ForceThisRoom(localNumbers[kEastRoom]);                        // Transit.c:169
 8.     OffsetGlider(glider, kToLeft);      // shift left by kRoomWide  // Transit.c:172
 9.     QSetRect(&enterRect, 0, 0, 48, 20);                            // Transit.c:174
10.     QOffsetRect(&enterRect, 0, kGliderStartsDown +
                                  (short)thisRoom->leftStart - 2);      // Transit.c:175
11.     glider->enteredRect = enterRect;
12. case kToLeft:  mirror, kWestRoom, OffsetGlider(kToRight),
        QOffsetRect(&enterRect, kRoomWide - 48,
                    kGliderStartsDown + (short)thisRoom->rightStart - 2);  // Transit.c:207
13. case kAbove:                                                        // Transit.c:222
14.     SetMusicalMode(kKickGameScoreMode);                            // Transit.c:223
15.     ForceThisRoom(localNumbers[kNorthRoom]);                       // Transit.c:224
16.     if (!takingTheStairs) { OffsetGlider(glider, kBelow); enteredRect = dest; }  // :231-:239
17.     else                    ReadyGliderForTripUpStairs(glider);     // Transit.c:246
18. case kBelow: mirror, kSouthRoom, OffsetGlider(kAbove),
        ReadyGliderForTripDownStairs                                    // Transit.c:254-:282
19. 2P: TagGliderIdle on the non-first player                           // Transit.c:290
20. ReadyLevel();                                                      // Transit.c:298
21. RefreshScoreboard(kNormalTitleMode);
22. WipeScreenOn(where, &justRoomsRect);                               // Transit.c:300
```

`OffsetGlider` (`GliderPRO/Sources/Player.c:1443`) shifts every glider rect:

| `where` | Delta |
|---|---|
| `kToRight` | `+kRoomWide` (+512) |
| `kToLeft` | `-kRoomWide` (-512) |
| `kAbove` | `-kTileHigh` (-322) |
| `kBelow` | `+kTileHigh` (+322) |

with an early return for the dead player in two-player mode
(`GliderPRO/Sources/Player.c:1445`). Note the direction naming inverts: leaving *to the
right* offsets the glider *to the left* by a room width, because the glider's coordinates
are relative to the new room.

The entry Y position is `kGliderStartsDown (32) + roomType.leftStart - 2` (or
`rightStart`), where `leftStart`/`rightStart` are the per-room `Byte`s at room offsets 30
and 31 (section 3.3). Note `(short)` cast of an unsigned `Byte`, so the range is 0..255,
and the `- 2`.

---

## 13. Where `leftThresh`, `rightThresh`, `topOpen`, `bottomOpen` come from

These four globals (`GliderPRO/Sources/Room.c:28`, `:30`) are recomputed **once per room
entry** by `DetermineRoomOpenings()` (`GliderPRO/Sources/Room.c:816`), called from
`InitializeRoom`-time code at `GliderPRO/Sources/Play.c:109` and from
`GliderPRO/Sources/RoomGraphics.c:415` (`DrawLocale`). They are *not* recomputed per frame,
so a Go port can compute them at room load.

### 13.1 The two thresholds and their four possible values

| Global | "wall present" | "no wall" |
|---|---:|---:|
| `leftThresh` | `kLeftWallLimit` = 12 | `kNoLeftWallLimit` = -24 |
| `rightThresh` | `kRightWallLimit` = 500 | `kNoRightWallLimit` = 536 |

`CheckEscapeLeft` and `CheckEscapeLeftTwo` decide which regime they are in by the test
`leftThresh == kLeftWallLimit` (`GliderPRO/Sources/Interactions.c:513`, `:576`) — i.e. the
threshold value itself is the flag. `leftOpen`/`rightOpen` are set alongside but are only
consumed by drawing code; the *collision* path reads `leftThresh`/`rightThresh`.

### 13.2 `DetermineRoomOpenings` decision table

Inputs: `whichBack = thisRoom->background`, `leftTile = thisRoom->tiles[0]`,
`rightTile = thisRoom->tiles[7]` (`GliderPRO/Sources/Room.c:821`-`:823`).

**Case A — user background (`whichBack >= kUserBackground` = 3000)**
(`GliderPRO/Sources/Room.c:825`-`:843`):

```
boundsCode = (thisRoom->bounds != 0) ? (thisRoom->bounds >> 1)
                                     : GetOriginalBounding(whichBack);
leftOpen  = ((boundsCode & 0x0001) == 0x0001);
rightOpen = ((boundsCode & 0x0004) == 0x0004);
```

**Case B — built-in backgrounds** (`GliderPRO/Sources/Room.c:846`-`:920`):

| Backgrounds | `leftThresh = kLeftWallLimit` iff | `rightThresh = kRightWallLimit` iff | Line |
|---|---|---|---|
| `kSimpleRoom` 2000, `kPaneledRoom` 2001, `kBasement` 2002, `kChildsRoom` 2003, `kAsianRoom` 2004, `kUnfinishedRoom` 2005, `kSwingersRoom` 2006, `kBathroom` 2007, `kLibrary` 2008, `kSky` 2015 | `leftTile == 0` | `rightTile == 7` | 848-867 |
| `kDirt` 2011 | `leftTile == 1` | `rightTile == 7` | 870-880 |
| `kMeadow` 2012 | `leftTile == 6` | `rightTile == 7` | 883-893 |
| `kGarden` 2009, `kSkywalk` 2010, `kField` 2013, `kStratosphere` 2016, `kStars` 2017 | never (always open) | never (always open) | 896-904 |
| default (incl. `kRoof` 2014) | `leftTile == 0` | `rightTile == 7` | 907-919 |

Three subtleties:

1. The `kDirt` **left** test is `leftTile == 1`, not `== 0`. Tile code 1 is the left wall cap
   art. But `leftOpen` in the same branch is computed as `(leftTile != 0)`
   (`GliderPRO/Sources/Room.c:879`) — **inconsistent with `leftThresh`**. A `kDirt` room with
   `tiles[0] == 4` (plain tunnel) gets `leftThresh = kNoLeftWallLimit` (open, correct) *and*
   `leftOpen = true` (also correct), but a `kDirt` room with `tiles[0] == 1` gets
   `leftThresh = kLeftWallLimit` (wall) and `leftOpen = true` (open) — contradictory. Only
   `leftThresh` affects collision.
2. `rightThresh == kRightWallLimit` requires `rightTile == kNumTiles - 1 == 7`. In every
   built-in background *except* `kMeadow`, tile code 7 happens to be the right-wall cap. This
   is a coincidence of the tile art, not a rule; `rightTile` is a tile *code*, and the
   comparison against `kNumTiles - 1` is numerology.
3. `kRoof` falls through to `default`, so a roof room walls off at `tiles[0]==0` /
   `tiles[7]==7`. Observed roof tile arrays (section 20.6) commonly end with 7 — e.g. CD Demo
   House room 184 `[3,3,3,3,3,3,3,7]` and Demo House room 6 `[3,3,3,3,3,3,3,7]` — so those
   rooms *do* have a right wall.

### 13.3 `topOpen` / `bottomOpen`

```c
bottomOpen = !DoesRoomHaveFloor();     // GliderPRO/Sources/Room.c:924
topOpen    = !DoesRoomHaveCeiling();   // GliderPRO/Sources/Room.c:929
```

`DoesRoomHaveFloor` (`GliderPRO/Sources/Room.c:1138`):

* user background: `hasFloor = ((boundsCode & 0x0008) != 0x0008)` — bit 3 set means **no**
  floor. Note the double negative: the `bnds` resource's `bottom` byte being non-zero means
  "bottom is open".
* built-in: `false` for `kSky` (2015), `kStratosphere` (2016), `kStars` (2017); `true` for
  everything else — **including `kRoof`**.

`DoesRoomHaveCeiling` (`GliderPRO/Sources/Room.c:1172`):

* user background: `hasCeiling = ((boundsCode & 0x0002) != 0x0002)`.
* built-in: `false` for `kGarden` (2009), `kMeadow` (2012), `kField` (2013), `kRoof` (2014),
  `kSky` (2015), `kStratosphere` (2016), `kStars` (2017); `true` for everything else.

`IsShadowVisible` (`GliderPRO/Sources/Room.c:1103`) is a near-duplicate of
`DoesRoomHaveFloor` whose built-in list *additionally* includes `kRoof`
(`GliderPRO/Sources/Room.c:1120`). So a `kRoof` room draws no glider shadow but does have a
collidable floor. Do not merge the two functions.

### 13.4 `IsRoomAStructure` (`GliderPRO/Sources/Room.c:763`)

Not part of collision, but it shares the `bounds` word and is needed to reproduce the room
graphics that the tile-code collision reads:

```
if (roomNum == kRoomIsEmpty /* -2 */) return false;                     // Room.c:768
if (background >= kUserBackground /* 3000 */)
    if (bounds != 0) isStructure = ((bounds & 32) == 32);               // Room.c:777
    else             isStructure = (background < kUserStructureRange /* 3300 */);  // Room.c:781
else isStructure = background in { kPaneledRoom, kSimpleRoom, kChildsRoom,
                                   kAsianRoom, kUnfinishedRoom, kSwingersRoom,
                                   kBathroom, kLibrary, kSkywalk, kRoof };  // Room.c:789-:800
```

Note bit 5 (value 32) is tested **without** the `>> 1`, unlike `boundsCode`.

### 13.5 `GetOriginalBounding` (`GliderPRO/Sources/Room.c:937`)

```c
boundsRes = (boundsHand)GetResource('bnds', theID);
if (boundsRes == nil) { if (PictIDExists(theID)) YellowAlert(kYellowNoBoundsRes, 0);
                        boundCode = 0; }
else {
    boundCode = 0;
    if ((*boundsRes)->left)   boundCode += 1;                           // Room.c:954
    if ((*boundsRes)->top)    boundCode += 2;                           // Room.c:956
    if ((*boundsRes)->right)  boundCode += 4;                           // Room.c:958
    if ((*boundsRes)->bottom) boundCode += 8;                           // Room.c:960
}
```

`boundsType` is four `Boolean` (one byte each) in the order left, top, right, bottom
(`GliderPRO/Headers/GliderStructs.h:266`-`:272`), so a `'bnds'` resource is exactly **4
bytes**. Non-zero means *open*. Verified against real houses in section 20.5.

Note the bit assignment differs from the room `bounds` field's raw layout: in the room word,
after `>> 1`, bit 0 = left, bit 1 = top, bit 2 = right, bit 3 = bottom, and bit 4 (pre-shift
bit 5, value 32) = isStructure. `RoomInfo.c:767`-`:780` is the write side and confirms the
`<< 1` then `+ 1` encoding, where the low bit is a "this is a 2.0-format room, use my bounds"
marker.

### 13.6 The full `bounds` word encoding (write side)

`GliderPRO/Sources/RoomInfo.c:767`-`:780`:

| Bit (raw word) | Value | Meaning |
|---:|---:|---|
| 0 | 1 | always set — "explicit bounds present" marker; `bounds != 0` tests for it |
| 1 | 2 | left open |
| 2 | 4 | top open |
| 3 | 8 | right open |
| 4 | 16 | bottom open |
| 5 | 32 | written from `originalFloor` — the editor's "Floor support" checkbox (`RoomInfo.c:751`, `:776`-`:777`); read back only by `IsRoomAStructure` as "room is a structure" (`Room.c:777`) |

Read side: `boundsCode = bounds >> 1`, then test `& 1` left, `& 2` top, `& 4` right,
`& 8` bottom; `isStructure` tests the *unshifted* `& 32`. Observed values from real houses
are tabulated in section 20.4.

---

## 14. One-way surfaces and pass-through geometry

Glider PRO has five distinct one-way mechanisms. None of them is a generic "one-way
platform"; each is special-cased.

### 14.1 `kDirt` vertical tunnels

Only in rooms whose `background == kDirt` (2011). The room's eight `tiles[]` codes decide
which columns are open.

| Tile code | Art | Escape up? | Escape down? |
|---:|---|---|---|
| 0 | solid / absent | no | no |
| 1 | left wall cap | no | no |
| 2 | down-opening, left half | no | **yes** |
| 3 | down-opening, right half | no | **yes** |
| 4 | plain horizontal tunnel | no | no |
| 5 | up-opening, left half | **yes** | no |
| 6 | up-opening, right half | **yes** | no |
| 7 | right wall cap | no | no |

The test requires **both** the tile under `dest.left` and the tile under `dest.right` to be
in the permitted pair (`GliderPRO/Sources/Interactions.c:206`-`:209` up,
`:318`-`:321` down):

```c
leftTile  = dest.left  >> 6;      // divide by kTileWide (64)
rightTile = dest.right >> 6;
if ((leftTile >= 0) && (leftTile < 8) &&      // literal 8, not kNumTiles
    (rightTile >= 0) && (rightTile < 8))
```

(The bound is spelled as the bare literal `8` in both places —
`GliderPRO/Sources/Interactions.c:203`-`:204` and `:315`-`:316` — even though `kNumTiles` is
8; the tile-code comparisons at `:206`-`:209` / `:318`-`:321` are likewise bare literals.)

`>> 6` on a negative `dest.left` is an **arithmetic** shift in every classic Mac compiler and
in Go, so `dest.left = -1` gives `leftTile = -1` and the range test correctly rejects it. A
Go port that writes `dest.left / 64` instead gets `0` for `-1` and would wrongly permit the
escape. **Use `>> 6`, not `/ 64`.**

The pairs are adjacent tile codes so that a 48-px-wide glider straddling a tile boundary
still passes: codes 5,6 form one 128-px-wide up shaft and codes 2,3 form one 128-px-wide down
shaft. Observed real tile arrays confirming this are in section 20.6, e.g. Demo House room 38
`[1,5,6,4,4,5,6,7]` — two up shafts at tile pairs (1,2) and (5,6).

### 14.2 `kRoof` diagonal roofs

`CheckRoofCollision` (`GliderPRO/Sources/Interactions.c:450`) runs when
`thisBackground == kRoof` and `dest.bottom > kRoofLimit` (122). It samples **one** tile, the
one under the glider's horizontal centre:

```c
offset = (dest.left + kHalfGliderWide /* 24 */) >> 6;                   // :454
if ((offset >= 0) && (offset <= 7) && (!thisGlider->sliding))            // :455
    tileOver = thisTiles[offset];                                       // :457
```

`x = (dest.left + 24) - (offset << 6)` is the glider centre's offset **within** the tile,
0..63.

| Tile code | Art | Collision test → death | Line |
|---:|---|---|---|
| 1 | left slope, upper half | `x > (250 - dest.bottom)` | 460 |
| 2 | left slope, lower half | `x > (186 - dest.bottom)` | 470 |
| 5 | right slope, upper half | `(64 - x) > (186 - dest.bottom)` | 480 |
| 6 | right slope, lower half | `(64 - x) > (250 - dest.bottom)` | 490 |
| anything else (0, 3, 4, 7) | flat / absent | **always** death | 498 |

Death action, identical in all five branches:

```c
thisGlider->vVel = kFloorLimit /* 312 */ - dest.bottom;                 // :464 etc.
StartGliderFadingOut(thisGlider);
PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
```

Interpretation: the roof surface is a line. For a left-facing slope, the surface height at
in-tile x is `250 - x` (upper half) or `186 - x` (lower half); the glider is inside the roof
when `dest.bottom > surface`, which rearranges to `x > (250 - dest.bottom)`. The two constants
250 and 186 differ by 64 — exactly one tile — because codes 1/2 (and 5/6) are the two vertical
halves of a 128-px-tall slope drawn across two tile rows.

Note the `!thisGlider->sliding` guard: a glider sliding on grease is immune to roof
collision for that frame.

**Tile codes 3, 4, 7 and 0 mean instant death** if the glider's bottom is below 122 in a roof
room. Tile 3 is the *flat* roof art, so a glider standing on a flat roof section that somehow
gets `dest.bottom > 122` dies. In practice a flat roof room has other geometry keeping the
glider up; but this is the single most counter-intuitive branch in the whole file, and a Go
port must reproduce the `default: die` behaviour verbatim.

### 14.3 `kManhole` — one-way floor via `kIgnoreGround`

`kManhole` produces **no** solid rect. Instead `GetHotRect` emits a `kIgnoreGround` hot spot
(`GliderPRO/Sources/ObjectRects.c:640`-`:647`) inset inside the manhole art. When the glider
overlaps it, `HandleHotSpotCollision` sets `thisGlider->ignoreGround = true`
(`GliderPRO/Sources/Interactions.c:1310`). `CheckEscapeDown` then takes its
`if (ignoreGround)` branch, requiring `dest.bottom > kNoFloorLimit` (332) to transition.

So the manhole is a *permission window*, not a hole: the glider must be horizontally over the
manhole **and** 20 px below the nominal floor before the room changes. The floor stays lethal
everywhere else. Verified geometry (Davis Station room 15, section 20.7): manhole art rect
(left 118, top 300, right 139, bottom 322) → derived `kIgnoreGround` rect left 118, right 139,
top 311, bottom 322.

### 14.4 Doors and windows — one-way walls via `ignoreLeft` / `ignoreRight`

`kCeilingTrans`-style side exits use the same permission pattern. `kDoorInLf`, `kDoorExLf`,
`kWindowInLf`, `kWindowExLf` produce a `kIgnoreLeft` hot spot; the `*Rt` variants produce
`kIgnoreRight`. `HandleHotSpotCollision` sets the flag
(`GliderPRO/Sources/Interactions.c:1302`, `:1306`), and `CheckEscapeLeft` /
`CheckEscapeRight` then skip the bounce and wait for `dest.left < -24` /
`dest.right > 536`.

Because the escape functions' `if (ignoreLeft)` branch has **no else**, the region between
the wall limit (12) and the escape limit (-24) is entirely uncollided while the flag is set —
that is the door's "depth".

### 14.5 Grease — one-way *ceiling* for the glider's feet

`kSlideIt` (`GliderPRO/Sources/Interactions.c:1376`-`:1378`):

```c
thisGlider->sliding = true;
thisGlider->vVel = who->bounds.top - thisGlider->dest.bottom;
```

The grease trail's rect is 2 px tall (`GliderPRO/Sources/Grease.c:64`-`:67`). Overlapping it
snaps the glider's bottom exactly to the trail's top by *assigning* the required velocity, and
sets `sliding`, which (a) selects the sliding sprite and is cleared in `MoveGliderNormal`
(`GliderPRO/Sources/Player.c:159`, `:181`) and (b) suppresses roof collision for the frame.
There is no test of approach direction, so the glider is snapped up onto the grease even when
rising through it from below.

---

## 15. The `ignore*` and `stillOver` flag lifecycles

### 15.1 The three one-frame permission flags

`gliderType` carries `Boolean sliding, ignoreLeft, ignoreRight;`
(`GliderPRO/Headers/GliderStructs.h:212`) plus `ignoreGround`.

| Flag | Set by | Read by | Cleared by |
|---|---|---|---|
| `ignoreLeft` | `kIgnoreLeft` hot spot, `GliderPRO/Sources/Interactions.c:1302` | `CheckEscapeLeft(Two)`, `:515`/`:578` | end of `HandleGlider`, `GliderPRO/Sources/Player.c:1436` |
| `ignoreRight` | `kIgnoreRight` hot spot, `:1306` | `CheckEscapeRight(Two)`, `:605`/`:668` | `GliderPRO/Sources/Player.c:1437` |
| `ignoreGround` | `kIgnoreGround` hot spot, `:1310` | `CheckEscapeDown(Two)`, `:344`/`:404` | `GliderPRO/Sources/Player.c:1438` |
| `sliding` | `kSlideIt` hot spot, `:1377` | `CheckRoofCollision`, `:455` | `MoveGliderNormal`, `GliderPRO/Sources/Player.c:159`/`:181` |

The ordering constraint is exact and load-bearing:

```
CheckForHotSpots()   sets the flags        (HandleInteraction, Interactions.c:1693)
CheckGliderInRoom()  reads the flags       (HandleInteraction, Interactions.c:1694-:1710)
HandleGlider()       clears the flags      (Play.c:460 -> Player.c:1436-:1438)
```

`CheckForHotSpots` runs **before** `CheckGliderInRoom` inside the same
`HandleInteraction` call (`GliderPRO/Sources/Interactions.c:1691`-`:1711`), so the flag set
this frame is consumed this frame. A Go port that clears the flags at the *top* of the frame
instead of after the movement pass will still work; one that clears them before
`CheckGliderInRoom` will break every door.

`sliding` is different — it is cleared inside the *mover*, and only in `MoveGliderNormal`. If
the glider is in any other mode when the grease is touched, `sliding` stays set indefinitely
and permanently disables roof collision. This is a real (harmless in practice) latch.

### 15.2 `stillOver`

`hotSpots[i].stillOver` (`GliderPRO/Headers/GliderStructs.h`, `hotType`) is the debounce for
edge-triggered actions. Three consumers:

* `HandleSwitches` — `if (who->stillOver) return;` at
  `GliderPRO/Sources/Interactions.c:990`, sets `stillOver = true` at `:1154`.
* `ArmTrigger` — `if (who->stillOver) return;` at `GliderPRO/Sources/Triggers.c:38`, sets
  `stillOver = true` at `:54`.
* `HandleMicrowaveAction` — `if (who->stillOver) return;` at
  `GliderPRO/Sources/Interactions.c:1164` but **never sets it** (see section 22).

Clearing happens in three places, and the two-player and one-player rules differ:

1. `CheckForHotSpots`, **two-player** path: `if (!hitObject) hotSpots[i].stillOver = false;`
   (`GliderPRO/Sources/Interactions.c:1673`-`:1674`). `hitObject` is true if *either* live
   glider both intersected the rect **and** was allowed to act on it. Note the asymmetry: in
   `onePlayerLeft` mode, an intersection by the *dead* player's glider satisfies `SectGlider`
   but not the `playerDead` guard, so `hitObject` stays false and `stillOver` is cleared even
   though a glider rect is sitting on the switch — the dead glider's stale `dest` cannot latch
   a switch, but neither does it block the debounce clearing.
2. `CheckForHotSpots`, **one-player** path: a plain
   `else hotSpots[i].stillOver = false;` (`GliderPRO/Sources/Interactions.c:1683`-`:1684`) on
   the `SectGlider` test.
3. `FlagStillOvers()` (`GliderPRO/Sources/Interactions.c:1715`), which recomputes
   `stillOver` for **every** hot spot from a fresh `SectGlider` test, and additionally forces
   `false` for hot spots with `isOn == false` (`:1730`) — which the per-frame sweep never
   touches, because the sweep's outer `if (hotSpots[i].isOn)` skips them entirely.

```c
void FlagStillOvers (gliderPtr thisGlider)
{
    for (i = 0; i < nHotSpots; i++)
        if (hotSpots[i].isOn)                                           // :1721
        {
            if (SectGlider(thisGlider, &hotSpots[i].bounds,
                           hotSpots[i].doScrutinize))                   // :1723-:1724
                hotSpots[i].stillOver = true;                           // :1725
            else
                hotSpots[i].stillOver = false;                          // :1727
        }
        else
            hotSpots[i].stillOver = false;                              // :1730
}
```

All four `SectGlider` call sites in this file pass the per-hot-spot
`hotSpots[i].doScrutinize` flag (`GliderPRO/Sources/Interactions.c:1640`, `:1658`, `:1680`,
`:1724`), which was recorded when the rect was added
(`GliderPRO/Sources/ObjectRects.c:288`). `FlagStillOvers` is therefore consistent with the
sweep. It is called from room-entry code so that a glider materialising on top of a switch
does not immediately flip it, and it takes a **single** glider argument — in two-player mode
only one glider's position seeds the debounce.

`hotObject` layout (`GliderPRO/Headers/GliderStructs.h:218`-`:225`):

| Offset | Type | Field | Size |
|---:|---|---|---:|
| 0 | `Rect` | `bounds` | 8 |
| 8 | `short` | `action` | 2 |
| 10 | `short` | `who` | 2 |
| 12 | `Boolean` | `isOn` | 1 |
| 13 | `Boolean` | `stillOver` | 1 |
| 14 | `Boolean` | `doScrutinize` | 1 |
| 15 | — | pad to even | 1 |

`sizeof(hotObject) == 16`. This is an in-memory structure only; it is never written to disk.

---

## 16. `MoveGlider` — the velocity model the collision code manipulates

`MoveGlider(gliderPtr thisGlider)` (`GliderPRO/Sources/Player.c:64`) is the single place
`dest` changes. Constants (`GliderPRO/Sources/Player.c:13`-`:17`):

| Constant | Value |
|---|---:|
| `kGravity` | 3 |
| `kHImpulse` | 2 |
| `kVImpulse` | 2 |
| `kMaxHVel` | 16 |
| `kShredderCountdown` | -68 |

### 16.1 Algorithm

```
 1. if (hVel > hDesiredVel)                                             // Player.c:66
 2.     hVel -= kHImpulse;  if (hVel < hDesiredVel) hVel = hDesiredVel;  // :68-:70
 3. else if (hVel < hDesiredVel)                                        // :72
 4.     hVel += kHImpulse;  if (hVel > hDesiredVel) hVel = hDesiredVel;  // :74-:76
 5. hDesiredVel = 0;                                                    // :78
 6. if (vVel > vDesiredVel)                                             // :80
 7.     vVel -= kVImpulse;  if (vVel < vDesiredVel) vVel = vDesiredVel;  // :82-:84
 8. else if (vVel < vDesiredVel)                                        // :86
 9.     vVel += kVImpulse;  if (vVel > vDesiredVel) vVel = vDesiredVel;  // :88-:90
10. vDesiredVel = kGravity;   /* 3 */                                   // :92
11. if (hVel < 0)
12.     if (hVel < -kMaxHVel) hVel = -kMaxHVel;                          // :96
13.     wasHVel = hVel;                                                 // :99
14.     whole.right = dest.right;  dest.left += hVel; dest.right += hVel;
        whole.left = dest.left;                                         // :101-:104
15.     (same four lines for shadowRect)                                // :106-:109
16. else
17.     if (hVel > kMaxHVel) hVel = kMaxHVel;                            // :113
18.     wasHVel = hVel;                                                 // :116
19.     whole.left = dest.left;  dest.left += hVel; dest.right += hVel;
        whole.right = dest.right;                                       // :118-:121
20.     (same for shadowRect)                                          // :123-:126
21. if (vVel < 0)
22.     wasVVel = vVel;                                                // :131
23.     whole.bottom = dest.bottom; dest.top += vVel; dest.bottom += vVel;
        whole.top = dest.top;                                          // :133-:136
24. else
25.     wasVVel = vVel;                                                // :140
26.     whole.top = dest.top; dest.top += vVel; dest.bottom += vVel;
        whole.bottom = dest.bottom;                                    // :142-:145
```

Facts a port must not lose:

* **`hDesiredVel` and `vDesiredVel` are consumed and reset every frame.** `vDesiredVel` is
  reset to `kGravity` (3), not 0. Anything that wants to change vertical motion must set
  `vDesiredVel` *this* frame. That is why fans/vents/flames set `hDesiredVel`/`vDesiredVel`
  rather than the velocities.
* **`hVel` is clamped to ±16; `vVel` is NEVER clamped.** A glider falling for many frames
  reaches unbounded `vVel`. This is why `CheckEscapeDown` clamps by *assignment*
  (`vVel = kFloorLimit - dest.bottom`) rather than by reflection — a reflection of a huge
  `vVel` would be catastrophic.
* The clamp happens **after** the impulse ramp but **before** `wasHVel` is recorded, so
  `wasHVel` is the clamped value. `GliderHitTop` relies on `wasHVel`
  (`GliderPRO/Sources/Interactions.c:65`-`:66`).
* `whole` is the **swept union** of the old and new `dest` (for dirty-rect blitting). The
  order of the four assignments in each branch is what produces the union: the leading edge is
  saved *before* the move and the trailing edge *after*. A Go port computing
  `whole = union(oldDest, newDest)` gets the same rect but must apply the horizontal sweep
  first and the vertical sweep second, because the vertical block overwrites
  `whole.top`/`whole.bottom` that the horizontal block just set from the *pre-vertical* dest.
* `hDesiredVel` is set by `GetInput` to ±`kNormalThrust` (**5**, `Input.c:14`) — there is no
  `kHVel` constant. `kFanStrength` (12) is *added* to it by fans
  (`GliderPRO/Sources/Interactions.c:1211`, `:1215`), so a fan plus keyboard can request
  **17** in one frame, which the ±`kMaxHVel` clamp then truncates to 16.
* The battery is a separate mechanism: `DoBatteryEngaged` (`GliderPRO/Sources/Input.c:121`)
  adds ±`kHyperThrust` (8) directly to **`hVel`**, not to `hDesiredVel`, so it bypasses the
  `kHImpulse` ramp entirely (but is still subject to the ±16 clamp later in the same frame).
  Helium is likewise `vDesiredVel = -kHeliumLift` (4) at `Input.c:162`.

### 16.2 How collision "pushes out" — three distinct idioms

Glider PRO never does a swept collision or an iterative separation. It uses three patterns:

**(a) Velocity assignment (exact snap).** Used for floors, ceilings and grease:

```c
vVel = kFloorLimit - dest.bottom;        // Interactions.c:349, :411, :438, :464, ...
vVel = kCeilingLimit - dest.top;         // Interactions.c:233, :236, :239, :272, ...
vVel = who->bounds.top - dest.bottom;    // Interactions.c:1378  (grease)
```

The rect is *not* moved. `MoveGlider` next frame applies `vVel`, landing `dest.bottom`
exactly on the limit. **The glider is therefore inside the solid for one frame.** Anything
that renders between `HandleInteraction` and the next `HandleGlider` shows the penetration —
which is exactly what happens, since `RenderFrame` runs after `HandleGlider`
(`GliderPRO/Sources/Play.c:460`, `:469`), so the penetrated frame is never drawn. A Go port
that renders from a different point in the loop will show one frame of clipping.

The impulse ramp interacts with this: `MoveGlider` step 6-9 ramps `vVel` toward
`vDesiredVel` by only ±2 before applying it, so the "exact snap" is *not* exact if the
required correction differs from the current velocity by more than 2 and `vDesiredVel` pulls
the other way. Concretely: `vVel` is set to `312 - dest.bottom`, then next frame
`vDesiredVel == kGravity == 3` so `vVel` moves 2 toward 3 before being applied. For a glider
resting on the floor with `dest.bottom == 312`, `vVel` is set to 0, then ramped to 2, so the
glider sinks 2 px, gets `vVel = 312 - 314 = -2`, ramps to 0, rises 0... the steady state is a
2-px jitter, not a rest. This is the source of Glider PRO's characteristic "bobbing" contact.

**(b) Reflection plus penetration bonus.** Used for walls and `kBounceIt`:

```c
offset = kLeftWallLimit - dest.left;     // positive when penetrating
hVel = -hVel + offset;                   // Interactions.c:590
```

and for the right wall `offset = kRightWallLimit - dest.right` (negative when penetrating),
`hVel = -hVel + offset` (`GliderPRO/Sources/Interactions.c:679`). Same expression, opposite
sign convention, because `offset` carries the direction.

**(c) Whole-gap assignment.** `BounceGlider` (`GliderPRO/Sources/Interactions.c:154`) picks
the *shorter* escape and assigns the whole gap as velocity:

```c
if ((theRect->right - glideBounds.left) < (glideBounds.right - theRect->left))
    hVel = theRect->right - glideBounds.left;    // push right                  // :160
else
    hVel = theRect->left - glideBounds.right;    // push left                   // :162
```

`glideBounds` here is the **raw** `dest` (`:157`), *not* inset. So the escape distance is
measured on the full 48x20 body even though the hit test that got us here used the inset
rect. Off by exactly the inset (5 px per side, or 6 px on top when burning).

**(d) Foil ricochet.** `GliderHitTop` (`GliderPRO/Sources/Interactions.c:54`) is a
combination: a horizontal-approach test on an inset-and-sweep-expanded rect, then

```c
if (thisGlider->hVel > 0) offset = 2 + glideBounds.right - theRect->left;   // :89
else                      offset = 2 + glideBounds.left  - theRect->right;  // :91
thisGlider->hVel = -thisGlider->hVel - offset;                              // :93
```

Note the magic `+ 2` and that here the sign convention makes `-hVel - offset` the correct
combination (contrast pattern (b), which uses `+ offset`).

---

## 17. Dynamic objects (`dinahs[]`) and the coordinate-space split

### 17.1 `dynaType` layout

`GliderPRO/Headers/GliderStructs.h:310`-`:320`. In-memory only.

| Offset | Type | Field | Notes |
|---:|---|---|---|
| 0 | `Rect` | `dest` | the collidable rect; **space depends on type** (17.2) |
| 8 | `Rect` | `whole` | swept union for dirty-rect blitting |
| 16 | `short` | `hVel` | overloaded: velocity, clip line, or `numLights` |
| 18 | `short` | `vVel` | vertical velocity |
| 20 | `short` | `type` | the object's `what` code |
| 22 | `short` | `count` | per-type period / initial velocity |
| 24 | `short` | `frame` | animation frame |
| 26 | `short` | `timer` | countdown |
| 28 | `short` | `position` | per-type: launch state, floor line, or top line |
| 30 | `short` | `room` | real room number |
| 32 | `Byte` | `byte0` | object slot index within the room |
| 33 | `Byte` | `byte1` | unused (always 0) |
| 34 | `Boolean` | `moving` | in flight / animating |
| 35 | `Boolean` | `active` | switched on |

`sizeof(dynaType) == 36`. Capacity `kMaxDynamicObs` = 18
(`GliderPRO/Headers/GliderDefines.h:265`); `numDynamics` is the live count and
`AddDynamicObject` returns -1 when full (`GliderPRO/Sources/Dynamics3.c:193`-`:194`).

Field overloads worth calling out, because a Go port that names them semantically will get
them wrong:

| Type | `hVel` means | `count` means | `position` means |
|---|---|---|---|
| `kToaster` | `where->top + 2`, a **clip line** (`GliderPRO/Sources/Dynamics3.c:223`) | initial launch velocity (`:233`) | launch/idle state 0/1 (`:236`) |
| `kOutlet` | `numLights` at build time (`GliderPRO/Sources/Dynamics3.c:314`) | `(delay * 6) / kTicksPerFrame` (`:316`) | zap state 0/1 (`:318`) |
| `kBall` | — | rebound velocity | floor line for the bounce |
| `kDrip` | the drip's start `top` | respawn period | floor line |
| `kFish` | idle timer reload | launch velocity | water-surface line |
| `kDart` | horizontal velocity (0 = falling) | respawn period | respawn `top` |

### 17.2 The coordinate-space split — **the single most porting-hostile detail here**

`AddDynamicObject` (`GliderPRO/Sources/Dynamics3.c:187`) builds `dest` differently per type:

**Screen-relative (`playOriginH`/`playOriginV` added).** All seven appliances:

| Type | `dest` origin | Line |
|---|---|---|
| `kMacPlus` | `where->left + playOriginH + 10`, `where->top + playOriginV + 7` | 247-249 |
| `kTV` | `+ playOriginH + 17`, `+ playOriginV + 10` | 266-268 |
| `kCoffee` | `+ playOriginH + 32`, `+ playOriginV + 57` | 285-287 |
| `kOutlet` | `+ playOriginH`, `+ playOriginV` | 310-312 |
| `kVCR` | `+ playOriginH + 64`, `+ playOriginV + 6` | 331-332 |
| `kStereo` | `+ playOriginH + 56`, `+ playOriginV + 20` | 354-355 |
| `kMicrowave` | `+ playOriginH + 14`, `+ playOriginV + 13` | 374-375 |

**Room-relative (no offset).** Everything else: `kSparkle` (`:202`), `kToaster` (`:219`),
`kBalloon` (`:394`), `kCopterLf`/`kCopterRt` (`:416`), `kDartLf`/`kDartRt` (`:444`),
`kBall` (`:469`), `kDrip` (`:499`), `kFish` (`:519`).

The `where` rect handed to `AddDynamicObject` is always `itsRect` after
`OffsetRectRoomRelative` (which **adds** `playOrigin`,
`GliderPRO/Sources/ObjectRects.c:1095`) and then `QOffsetRect(&rectA, -playOriginH,
-playOriginV)` (which removes it again) — e.g. `GliderPRO/Sources/ObjectDrawAll.c:653`-`:655`.
So `where` is always **room-relative**, and the appliances re-add `playOrigin` inside
`AddDynamicObject`.

That is why `CheckDynamicCollision` has a `doOffset` parameter
(`GliderPRO/Sources/Dynamics.c:34`):

```c
dinahRect = dinahs[who].dest;
if (doOffset)
    QOffsetRect(&dinahRect, -playOriginH, -playOriginV);                // Dynamics.c:40
```

`playOriginH = (screenWidth - kRoomWide) / 2`,
`playOriginV = (screenHeight - kTileHigh) / 2` (`GliderPRO/Sources/InterfaceInit.c:203`-
`:204`). On a 640x480 screen that is `playOriginH = (640 - 512) / 2 = 64` and
`playOriginV = (480 - 322) / 2 = 79`.

Of the fifteen dynamic types, only **eight** ever call `CheckDynamicCollision`:

| Handler | Type(s) | `dest` space | `doOffset` passed | Correct? |
|---|---|---|---|---|
| `HandleToast` (`GliderPRO/Sources/Dynamics.c:321`) | `kToaster` | room | `false` (`:338`, `:340`, `:344`, `:345`, `:349`) | yes |
| `HandleOutlet` (`GliderPRO/Sources/Dynamics.c:528`) | `kOutlet` | **screen** | `false` at `:539`, `:541`; `true` at `:545`, `:546`, `:550` | **no** — see 22 |
| `HandleBalloon` (`GliderPRO/Sources/Dynamics2.c:30`) | `kBalloon` | room | `false` (`:49`-`:60`) | yes |
| `HandleCopter` (`GliderPRO/Sources/Dynamics2.c:134`) | `kCopterLf/Rt` | room | `false` (`:150`-`:161`) | yes |
| `HandleDart` (`GliderPRO/Sources/Dynamics2.c:242`) | `kDartLf/Rt` | room | `false` (`:255`-`:266`) | yes |
| `HandleBall` (`GliderPRO/Sources/Dynamics2.c:358`) | `kBall` | room | `false` (`:365`-`:376`) | yes |
| `HandleDrip` (`GliderPRO/Sources/Dynamics2.c:427`) | `kDrip` | room | `false` (`:440`-`:451`) | yes |
| `HandleFish` (`GliderPRO/Sources/Dynamics2.c:501`) | `kFish` | room | `false` (`:514`-`:525`) | yes |

`kSparkle`, `kMacPlus`, `kTV`, `kCoffee`, `kVCR`, `kStereo` and `kMicrowave` never test the
glider from `HandleDynamics`; the microwave's lethality comes from its `kMicrowaveIt` hot spot
instead.

**A Go port should just keep every dynamic in room coordinates and drop `doOffset`
entirely** — but then the outlet's zap will collide 64/79 px away from where the original
collided it in the `onePlayerLeft` path, so if bug-for-bug fidelity is wanted, reproduce the
split.

### 17.3 `CheckDynamicCollision` (`GliderPRO/Sources/Dynamics.c:34`)

```
 1. dinahRect = dinahs[who].dest;
 2. if (doOffset) QOffsetRect(&dinahRect, -playOriginH, -playOriginV);
 3. if (!SectGlider(thisGlider, &dinahRect, true)) return;   // scrutinize = TRUE, always
 4. if (mode not in {kGliderNormal, kGliderFaceLeft, kGliderFaceRight,
                     kGliderBurning, kGliderGoingFoil, kGliderLosingFoil}) return;  // :44-:49
 5. if ((foilTotal > 0) || (mode == kGliderLosingFoil))                 // :51
 6. {
 7.     if (IsRectLeftOfRect(&dinahRect, &thisGlider->dest))            // :53
 8.         hDesiredVel =  kShoveVelocity;   /*  8 */                    // :54
 9.     else
10.         hDesiredVel = -kShoveVelocity;   /* -8 */                    // :56
11.     if (dinahs[who].vVel < 0)  vDesiredVel = dinahs[who].vVel;       // :57-:58
12.     PlayPrioritySound(kFoilHitSound, kFoilHitPriority);              // :59
13.     if ((evenFrame) && (foilTotal > 0))                             // :60
14.     {
15.         foilTotal--;
16.         if (foilTotal <= 0) StartGliderFoilLosing(thisGlider);       // :63-:64
17.     }
18. }
19. else
20. {
21.     StartGliderFadingOut(thisGlider);                               // :69
22.     PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
23. }
```

Facts:

* `kShoveVelocity` = 8 (`GliderPRO/Sources/Dynamics.c:17`).
* `scrutinize` is hard-coded `true`, so the glider is inset by 5 on every side (6 more on top
  when burning) — see section 7. Dynamic hazards are therefore **10 px narrower and 10 px
  shorter** in effect than their art.
* The mode list is the *six* modes that can be hit. `kGliderGoingFoil` and
  `kGliderLosingFoil` are included, so an enemy can hit a glider mid-foil-animation.
* `foilTotal` is decremented **only on even frames** — i.e. once every 4 ticks / 15 Hz — so a
  glider parked inside an enemy loses foil at half the frame rate.
* The push is `hDesiredVel = ±8` (an *assignment*, unlike the fans' `+=`), so an enemy hit
  overrides keyboard input for that frame.
* `vDesiredVel` is only overridden when the dynamic is moving **up** (`vVel < 0`), and it is
  set to the dynamic's own `vVel` — so a rising balloon carries the glider at the balloon's
  speed, but a falling dart does not push down.
* `IsRectLeftOfRect` (`GliderPRO/Sources/RectUtils.c:185`) decides the push direction; see
  section 7.5 for its precedence quirk.
* With no foil, contact is instant death regardless of relative velocity.

### 17.4 `DidBandHitDynamic` (`GliderPRO/Sources/Dynamics.c:80`)

The reverse test: a rubber band vs a dynamic. Same inclusive-overlap idiom against the raw
`dinahs[who].dest` and each `bands[i].dest`, breaking on the first hit
(`GliderPRO/Sources/Dynamics.c:84`-`:102`). **No coordinate offset is applied**, so this is
only correct for room-relative dynamics — which is fine, because only the four enemy types
call it (`GliderPRO/Sources/Dynamics2.c:59`, `:165`, `:267`, and none for ball/drip/fish).

`collided` is **uninitialized** when `numBands == 0`; the four call sites all guard with
`(numBands > 0) && DidBandHitDynamic(who)`, so the bug is unreachable. A Go port returns
`false`.

### 17.5 Enemy lifecycle constants

`GliderPRO/Sources/Dynamics2.c:13`-`:19`:

| Constant | Value | Meaning |
|---|---:|---|
| `kBalloonStop` | 8 | balloon despawns when `dest.top <= 8` |
| `kBalloonStart` | 310 | balloon respawns with `dest.bottom = 310` |
| `kCopterStart` | 8 | copter respawns with `dest.top = 8` |
| `kCopterStop` | 310 | copter despawns when `dest.bottom >= 310` |
| `kDartStop` | 310 | dart despawns when `dest.bottom >= 310` |
| `kEnemyDropSpeed` | 8 | vertical velocity after a band pops the enemy |
| `kStartSparkle` | 4 (`GliderPRO/Headers/GliderDefines.h:545`) | frames before respawn that the sparkle plays |

Dart also despawns on `dest.left <= 0` or `dest.right >= kRoomWide` (512)
(`GliderPRO/Sources/Dynamics2.c:296`-`:298`), and respawns pinned to whichever wall it flies
from (`:308`-`:322`).

Band hits: balloon → `frame = 6`, `vVel = 8`, `kPopSound`
(`GliderPRO/Sources/Dynamics2.c:63`-`:66`); dart → `frame = 1` (`kDartLf`) or `3`
(`kDartRt`), `hVel = 0`, `vVel = 8`, `kPaperCrunchSound` (`:267`-`:275`). A popped enemy has
`hVel == 0`, and `HandleDart`'s outer `if (dinahs[who].hVel != 0)` then skips
`CheckDynamicCollision` — **a falling popped dart is harmless**
(`GliderPRO/Sources/Dynamics2.c:248`). The balloon's equivalent guard is
`if (dinahs[who].vVel < 0)` (`:34`), so a popped falling balloon is also harmless.

Gravity for dynamics is applied on even frames only: `if (evenFrame) dinahs[who].vVel++`
(`GliderPRO/Sources/Dynamics2.c:409` ball, `:459` drip, `:534` fish). Toast is the exception —
`dinahs[who].vVel++` runs every frame (`GliderPRO/Sources/Dynamics.c:356`).

`HandleBall` clobbers the global `evenFrame`: `evenFrame = true;`
(`GliderPRO/Sources/Dynamics2.c:421`) when a ball starts bouncing. See section 22.

---

## 18. Rubber bands

### 18.1 `bandType` and constants

`GliderPRO/Headers/GliderStructs.h:274`-`:279`:

| Offset | Type | Field |
|---:|---|---|
| 0 | `Rect` | `dest` |
| 8 | `short` | `mode` — animation frame 0..2, or `kKillBandMode` (-1) |
| 10 | `short` | `count` — gravity sub-counter |
| 12 | `short` | `hVel` |
| 14 | `short` | `vVel` |

`sizeof(bandType) == 16`. `GliderPRO/Sources/RubberBands.c:12`-`:14`:

| Constant | Value |
|---|---:|
| `kRubberBandVelocity` | 20 |
| `kBandFallCount` | 4 |
| `kKillBandMode` | -1 |
| `kMaxRubberBands` (`GliderPRO/Headers/GliderDefines.h:261`) | 2 |

Globals: `bandPtr bands; short numBands, bandHitLast;`
(`GliderPRO/Sources/RubberBands.c:21`, `:26`).

### 18.2 `AddBand` (`GliderPRO/Sources/RubberBands.c:256`)

```
 1. if (numBands >= kMaxRubberBands /* 2 */) return false;              // :258
 2. mode = 0; count = 0;                                                 // :261-:262
 3. vVel = thisGlider->tipped ? -2 : 0;                                  // :263-:266
 4. dest = { left: h - 8, right: h + 8, top: v - 3, bottom: v + 3 };     // :267-:270
 5. if (direction == kFaceLeft) { dest.left -= 32; dest.right -= 32;
                                  hVel = -kRubberBandVelocity; }         // :272-:276
 6. else                        { dest.left += 32; dest.right += 32;
                                  hVel =  kRubberBandVelocity; }         // :278-:282
 7. thisGlider->hVel -= (bands[numBands].hVel / 2);                       // :285
 8. numBands++;
 9. PlayPrioritySound(kFireBandSound, kFireBandPriority);
```

So a band is **16 x 6** px, spawned 32 px to the side of `(h, v)`, and firing gives the
glider a **recoil of 10 px/frame** in the opposite direction (`20 / 2`). Firing while
`tipped` gives the band an initial upward `vVel` of -2.

### 18.3 `HandleBands` (`GliderPRO/Sources/RubberBands.c:208`)

```
 1. if (numBands == 0) return;                                          // :213-:214
 2. for (i = 0; i < numBands; i++)                                      // :216
 3.     mode++; if (mode > 2) mode = 0;                                 // :218-:220
 4.     count++; if (count >= kBandFallCount /* 4 */) { vVel++; count = 0; }  // :222-:227
 5.     AddRectToWorkRects(dest + playOrigin);                          // :229-:231
 6.     dest.left += hVel; dest.right += hVel;
        dest.top  += vVel; dest.bottom += vVel;                        // :233-:236
 7.     CheckBandCollision(i);                                         // :238
 8. count = 0;                                                         // :241
 9. do { while (bands[count].mode == kKillBandMode) { bands[count].mode = 0;
                                                      KillBand(count); }
        count++; } while (count < numBands);                            // :242-:251
```

Gravity is **one unit every four frames** (0.25 px/frame², i.e. much lighter than the
glider's `kGravity` = 3 desired velocity). The band is moved **before** the collision test, so
`CheckBandCollision` sees the post-move rect and reconstructs the pre-move edge by subtracting
`hVel`.

The compaction loop at step 8-9 is a `do/while` over a shrinking `numBands`; `KillBand`
(`GliderPRO/Sources/RubberBands.c:294`) swaps the last live band into the dead slot, so the
inner `while` re-tests the same index. If `numBands` becomes 0 the `do` body still runs once
with `count = 0` and reads `bands[0].mode` — which `KillBand` did not clear, but step "mode =
0" did. Safe, but only by accident. A Go port should iterate backwards or filter.

### 18.4 `CheckBandCollision` (`GliderPRO/Sources/RubberBands.c:37`)

Five phases, in order:

**Phase 1 — walls** (`GliderPRO/Sources/RubberBands.c:44`-`:60`):

```c
if ((leftThresh == kLeftWallLimit) && (bands[who].dest.left < kLeftWallLimit)) {
    if (bands[who].hVel < 0) bands[who].hVel = -bands[who].hVel;
    bands[who].dest.left  = kLeftWallLimit;      // snap
    bands[who].dest.right = bands[who].dest.left + 16;
    PlayPrioritySound(kBandReboundSound, ...);
    collided = true;                             // dead store
}
else if ((rightThresh == kRightWallLimit) && (bands[who].dest.right > kRightWallLimit)) {
    if (bands[who].hVel > 0) bands[who].hVel = -bands[who].hVel;
    bands[who].dest.right = kRightWallLimit;
    bands[who].dest.left  = bands[who].dest.right - 16;
    ...
}
```

Bands **do** snap their rect (unlike the glider, which only sets velocity). Note the width is
re-derived as a hard-coded 16, matching `AddBand`. `collided = true` here is a dead store —
the hot-spot loop overwrites `collided` unconditionally on its first iteration.

**Phase 2 — hot spots** (`GliderPRO/Sources/RubberBands.c:62`-`:143`). For each
`hotSpots[i]` with `isOn`, only these five actions are considered
(`GliderPRO/Sources/RubberBands.c:67`-`:69`):

| Action | Effect on band |
|---|---|
| `kDissolveIt` | bounce or die |
| `kBounceIt` | bounce or die |
| `kRewardIt` | grease jars only (see below) |
| `kSwitchIt` | `HandleSwitches(&hotSpots[i])` |
| `kTriggerIt` | `ArmTrigger(&hotSpots[i])` |

Overlap is the same inclusive four-way test on the raw rects
(`GliderPRO/Sources/RubberBands.c:70`-`:79`) — **no inset**, so bands hit hot spots on their
full 16x6 body.

Debounce: the global `bandHitLast` holds the hot-spot index hit on the previous frame. The
action only fires if `bandHitLast != i` (`GliderPRO/Sources/RubberBands.c:83`), and
`bandHitLast = -1` when no hot spot was hit at all this frame
(`:145`-`:146`). This is a *single* global, shared across both bands and all hot spots — so
two bands hitting two different switches on the same frame will fire both (the second
overwrites `bandHitLast`), but one band grazing two hot spots in successive frames will
suppress the second.

`kDissolveIt` / `kBounceIt` (`GliderPRO/Sources/RubberBands.c:87`-`:115`):

```c
if (bands[who].hVel > 0) {
    if ((bands[who].dest.right - bands[who].hVel) < hotSpots[i].bounds.left) {
        bands[who].hVel = -bands[who].hVel;                             // reflect
        bands[who].dest.right = hotSpots[i].bounds.left;                // snap
        bands[who].dest.left  = bands[who].dest.right - 16;
    }
    else bands[who].mode = kKillBandMode;                              // absorbed
}
else { /* mirror against bounds.right */ }
PlayPrioritySound(kBandReboundSound, ...);
break;                                                                 // leaves the hot-spot loop
```

The test `(dest.right - hVel) < bounds.left` asks "was the band's leading edge outside the
obstacle *before* this frame's move?" If yes it is a genuine face-on hit and the band
rebounds; if no (the band was already inside, i.e. it came from above/below/behind) it is
destroyed. Note the `break` — a bouncing band stops scanning further hot spots this frame.

`kRewardIt` (`GliderPRO/Sources/RubberBands.c:116`-`:129`) only reacts to grease jars:

```c
whoLinked = hotSpots[i].who;
if ((masterObjects[whoLinked].theObject.what == kGreaseRt) ||
    (masterObjects[whoLinked].theObject.what == kGreaseLf)) {
    if (SetObjectState(thisRoomNumber, masterObjects[whoLinked].objectNum, 0, whoLinked))
        SpillGrease(masterObjects[whoLinked].dynaNum, masterObjects[whoLinked].hotNum);
    hotSpots[i].isOn = false;
}
```

**A band cannot collect any other prize.** Clocks, batteries, bands, foil, helium, stars and
paper are all `kRewardIt` and are simply ignored (no `else`), and the band is not consumed.

**Phase 3 — the gliders** (`GliderPRO/Sources/RubberBands.c:148`-`:194`), only if
`bands[who].hVel != 0`:

```c
/* inclusive overlap of band dest vs theGlider.dest, no inset */
if (collided) {
    if ((!twoPlayerGame) || (!onePlayerLeft) || (playerDead == kPlayer2)) {
        theGlider.hVel += (bands[who].hVel / 2);                        // :167
        bands[who].hVel = 0;                                            // :168
        PlayPrioritySound(kHitWallSound, kHitWallPriority);
    }
}
if (twoPlayerGame) { /* same against theGlider2, guard playerDead == kPlayer1 */ }
```

So a band **pushes a glider by half its speed (10 px/frame) and then stops dead
horizontally** — but keeps falling, since only `hVel` is zeroed. Friendly fire is real: your
own band, fired and then flown into, will shove you. Note this test uses the **raw**
`theGlider.dest`, never the inset rect, and does not check the glider's mode — a fading,
transporting or limbo glider is still shoved.

**Phase 4 — out-of-bounds kill** (`GliderPRO/Sources/RubberBands.c:195`-`:203`):

```c
if ((bands[who].dest.left < kLeftWallLimit) ||
    (bands[who].dest.right > kRightWallLimit))  bands[who].mode = kKillBandMode;
else if (bands[who].dest.bottom > kFloorLimit)  bands[who].mode = kKillBandMode;
```

Note this uses the **fixed** `kLeftWallLimit` (12) / `kRightWallLimit` (500) /
`kFloorLimit` (312), **not** `leftThresh`/`rightThresh`. So in an open-sided room, phase 1 does
nothing (the `leftThresh == kLeftWallLimit` guard fails) and phase 4 kills the band the moment
it crosses 12 or 500. **Bands never leave the room.**

`KillAllBands` (`GliderPRO/Sources/RubberBands.c:305`) zeroes every `mode` and sets
`numBands = 0`; it is called on room transition, so bands do not survive a room change.

---

## 19. Grease

### 19.1 `greaseType` and modes

`GliderPRO/Headers/GliderStructs.h:287`-`:295`:

| Offset | Type | Field | Notes |
|---:|---|---|---|
| 0 | `Rect` | `dest` | the jar's 32x27 art rect, **screen-relative** |
| 8 | `short` | `mapNum` | index into `savedMaps[]` |
| 10 | `short` | `mode` | 0..3, see below |
| 12 | `short` | `who` | object slot index |
| 14 | `short` | `where` | real room number |
| 16 | `short` | `start` | current leading edge of the trail (screen H) |
| 18 | `short` | `stop` | final leading edge (screen H) |
| 20 | `short` | `frame` | tip animation frame, -1 when upright |
| 22 | `short` | `hotNum` | index into `hotSpots[]` |
| 24 | `Boolean` | `isRight` | spill direction |
| 25 | — | pad | |

`sizeof(greaseType) == 26`. Capacity `kMaxGrease` = 16
(`GliderPRO/Headers/GliderDefines.h:262`); `numGrease` is the live count.

Modes (`GliderPRO/Sources/Grease.c:17`-`:20`):

| Constant | Value | Meaning |
|---|---:|---|
| `kGreaseIdle` | 0 | jar upright |
| `kGreaseFalling` | 1 | jar tipping over (3-frame animation) |
| `kGreaseSpreading` | 2 | trail growing 2 px/frame |
| `kGreaseSpiltIdle` | 3 | trail complete |

### 19.2 `AddGrease` (`GliderPRO/Sources/Grease.c:206`)

Called from the drawing pass with **screen** coordinates
(`GliderPRO/Sources/ObjectDrawAll.c:376`-`:378` for `kGreaseRt`, `:397`-`:399` for
`kGreaseLf`).

```
 1. if (numGrease >= kMaxGrease /* 16 */) return -1;                    // :212-:213
 2. src = {0,0,32,27} offset to (h, v);                                 // :215-:216
 3. bounds = {0,0,32,27*4};  savedNum = BackUpToSavedMap(&bounds, where, who);  // :218-:219
 4. if (savedNum == -1) return -1;   /* written as `if (savedNum != -1) { ... } else return -1;` */
 5. BackupGrease(&src, savedNum, isRight);   // renders 4 tip frames into the saved map  // :222
 6. QOffsetRect(&src, isRight ? -8 : +8, 0);                            // :223-:226
 7. grease[n] = { who, where, dest: src, mapNum: savedNum,
                  mode: kGreaseIdle, frame: -1 };                       // :227-:232
 8. if (isRight) { isRight = true;  start = src.right + 4;
                                    stop  = src.right + distance; }     // :235-:237
 9. else         { isRight = false; start = src.left  - 4;
                                    stop  = src.left  - distance; }     // :241-:243
10. numGrease++;  return numGrease - 1;                                 // :245-:247
```

`distance` is `masterObjects[...].theObject.data.a.distance` — the jar's *spill length* in
pixels. The trail is 4 px clear of the jar art at either end (the `± 4`), and the jar art is
shifted 8 px *away* from the spill direction (the `± 8` at step 6) so the tipped jar's
silhouette lines up.

The 108-px-tall backup rect is `27 * 4` — the four tip frames stacked vertically in the saved
GWorld (`BackupGrease`, `GliderPRO/Sources/Grease.c:141`).

### 19.3 `SpillGrease` (`GliderPRO/Sources/Grease.c:257`)

```c
void SpillGrease (short who, short index)
{
    if (grease[who].mode == kGreaseIdle) {
        grease[who].mode   = kGreaseFalling;
        grease[who].hotNum = index;                                     // :262
        PlayPrioritySound(kGreaseSpillSound, kGreaseSpillPriority);
    }
}
```

`who` is `masterObjects[...].dynaNum` and `index` is `masterObjects[...].hotNum` at both call
sites (`GliderPRO/Sources/Interactions.c:895`, `:1060`;
`GliderPRO/Sources/RubberBands.c:124`; `GliderPRO/Sources/Triggers.c:117`). **`dynaNum` is
reused as the grease index**, not a `dinahs[]` index — `AddGrease` returns
`numGrease - 1` and that is what gets stored (`GliderPRO/Sources/ObjectDrawAll.c:379`,
`:400`). A Go port must keep grease and dynamic indices in the same field or accept the
overload deliberately.

The guard `mode == kGreaseIdle` makes spilling idempotent: hitting the jar twice does
nothing.

### 19.4 `HandleGrease` (`GliderPRO/Sources/Grease.c:43`) — how the slide rect appears

Called from `RenderFrame` (`GliderPRO/Sources/Render.c:647`), i.e. **after**
`HandleInteraction` in the frame (`GliderPRO/Sources/Play.c:454` then `:469`). So the hot rect
created here is not tested until the *next* frame.

`kGreaseFalling` branch (`GliderPRO/Sources/Grease.c:51`-`:85`):

```
 1. frame++;
 2. if (frame >= 3) {
 3.     frame = 3;  mode = kGreaseSpreading;                            // :55-:57
 4.     hotSpots[hotNum].action = kSlideIt;                             // :58
 5.     hotSpots[hotNum].isOn   = true;                                 // :59
 6.     if (isRight) QSetRect(&src, 0, -2, 2, 0);                       // :61
        else         QSetRect(&src, -2, -2, 0, 0);                      // :63
 7.     QOffsetRect(&src, -playOriginH, -playOriginV);                  // :64
 8.     QOffsetRect(&src, grease[i].start, grease[i].dest.bottom);      // :65
 9.     hotSpots[hotNum].bounds = src;                                  // :66
10. }
11. blit tip frame  = {0, frame*27, 32, frame*27+27} to dest in both work and back maps  // :68-:78
12. AddRectToWorkRects(&dest);
13. QOffsetRect(&dest, isRight ? +2 : -2, 0);                           // :81-:84
```

Step 6-9 is the load-bearing geometry. The seed rect is 2 x 2:

* `isRight`: `{left 0, top -2, right 2, bottom 0}` then offset by `(start, dest.bottom)` →
  `{left start, top dest.bottom - 2, right start + 2, bottom dest.bottom}`.
* `isLeft`: `{left -2, top -2, right 0, bottom 0}` → `{left start - 2, top dest.bottom - 2,
  right start, bottom dest.bottom}`.

Because `grease[].dest` and `grease[].start` are **screen** coordinates, step 7 subtracts
`playOrigin` to convert to the **room** coordinates that `hotSpots[].bounds` uses. Note the
order: the `playOrigin` subtraction happens *before* the `(start, dest.bottom)` offset, so it
is only applied once even though both terms are screen values — which is wrong by
`-playOrigin` and right by accident? No: `src` is a *relative* offset rect at step 6, so
subtracting `playOrigin` then adding the screen-space anchor yields
`screenAnchor - playOrigin` = room space. Correct.

`kGreaseSpreading` branch (`GliderPRO/Sources/Grease.c:86`-`:129`):

```
 1. if (isRight) { src = {0,-2,2,0} offset by (start, dest.bottom);
                   start += 2;  hotSpots[hotNum].bounds.right += 2; }   // :90-:96
 2. else         { src = {-2,-2,0,0} offset by (start, dest.bottom);
                   start -= 2;  hotSpots[hotNum].bounds.left  -= 2; }   // :99-:104
 3. PaintRect(&src) into backSrcMap and workSrcMap; AddRectToWorkRects(&src);  // :111-:120
 4. if (isRight) { if (start >= stop) mode = kGreaseSpiltIdle; }         // :124-:125
    else         { if (start <= stop) mode = kGreaseSpiltIdle; }         // :128-:129
```

The trail grows **2 px per frame** and the hot rect grows with it. Note that in this branch
`src` is *not* `playOrigin`-corrected — it is used only for drawing into the screen-space
GWorlds, which is correct. The hot rect is grown by direct arithmetic on
`hotSpots[hotNum].bounds`, so no conversion is needed.

The hot rect stays exactly **2 px tall** forever
(`top = dest.bottom - 2`, `bottom = dest.bottom`), and grows only horizontally.
`RedrawAllGrease` (`GliderPRO/Sources/Grease.c:265`) relies on that:

```c
src = hotSpots[grease[i].hotNum].bounds;
if ((grease[i].where == thisRoomNumber) &&
    ((src.bottom - src.top) == 2) &&                                    // :283
    (grease[i].mode != kGreaseIdle))
{
    QOffsetRect(&src, playOriginH, playOriginV);   // room -> screen     // :286
    PaintRect into backSrcMap and workSrcMap; AddRectToWorkRects(&src);
}
```

The `== 2` height test is how it distinguishes a real grease trail from a recycled hot spot.
`RedrawAllGrease` is called after `RestoreFromSavedMap` in the clock-pickup paths
(`GliderPRO/Sources/Interactions.c:777`, `:793`, `:809`, `:826`) because restoring the saved
background erases any trail drawn over it.

### 19.5 `ReBackUpGrease` (`GliderPRO/Sources/Grease.c:180`)

Re-renders the four tip frames when room lighting changes, but **only** for
`kGreaseIdle`/`kGreaseFalling` (`GliderPRO/Sources/Grease.c:189`) — a fully spilt jar is not
refreshed. Returns the grease index for `(where, who)` or -1. A Go port that re-renders on
lighting change must reproduce the same exclusion or the jar will pop back upright.

---

## 20. On-disk verification — observed bytes

Everything in this section is real output from parsing the shipped house files in
`GliderPRO/Houses/*.binhex` with python3. The scratch scripts live in
`/tmp/wf-interactions/` (`bhx.py` BinHex 4.0 decoder, `house.py` house parser, `rsrc.py`
resource-fork parser, `verify.py` the evidence generator). Houses are BinHex 4.0 containers;
the level data is in the **data** fork with type `'gliH'` and creator `'ozm5'`, read by a raw
`FSRead` with **no byte swapping** (`GliderPRO/Sources/HouseIO.c:372`), so every multi-byte
field is **big-endian**.

### 20.1 House file size formula

`866 + 348 * nRooms == len(dataFork)`, from
`sizeof(houseType) == 866` (`GliderPRO/Headers/GliderStructs.h:184`-`:198`) and
`sizeof(roomType) == 348` (`:180`).

```
Art Museum.binhex                nRooms= 109 ver=0x0200 expect=38798 actual=38798 OK
CD Demo House.binhex             nRooms= 206 ver=0x0200 expect=72554 actual=72554 OK
California or Bust!.binhex       nRooms=  16 ver=0x0200 expect=6434 actual=6434 OK
Castle o' the Air.binhex         nRooms=  85 ver=0x0200 expect=30446 actual=30446 OK
Davis Station.binhex             nRooms=  65 ver=0x0200 expect=23486 actual=23486 OK
Demo House.binhex                nRooms=  45 ver=0x0200 expect=16526 actual=16526 OK
Empty House.binhex               nRooms=  35 ver=0x0200 expect=13046 actual=13046 OK
Fun House.binhex                 nRooms=  43 ver=0x0200 expect=15830 actual=15830 OK
Grand Prix.binhex                nRooms= 175 ver=0x0200 expect=61766 actual=61766 OK
ImagineHouse PRO II.binhex       nRooms= 279 ver=0x0200 expect=97958 actual=97958 OK
In The Mirror.binhex             nRooms=  97 ver=0x0200 expect=34622 actual=34622 OK
Land of Illusion.binhex          nRooms= 303 ver=0x0200 expect=106310 actual=106310 OK
Leviathan.binhex                 nRooms= 472 ver=0x0200 expect=165122 actual=165122 OK
Metropolis.binhex                nRooms= 127 ver=0x0200 expect=45062 actual=45062 OK
Nemo's Market.binhex             nRooms= 124 ver=0x0200 expect=44018 actual=44018 OK
Rainbow's End.binhex             nRooms= 223 ver=0x0200 expect=78470 actual=78470 OK
Sampler.binhex                   nRooms=   2 ver=0x0200 expect=1562 actual=1564 MISMATCH
Slumberland.binhex               nRooms= 383 ver=0x0200 expect=134150 actual=134150 OK
SpacePods.binhex                 nRooms= 402 ver=0x0200 expect=140762 actual=140762 OK
Teddy World.binhex               nRooms= 531 ver=0x0200 expect=185654 actual=185654 OK
The Asylum Pro.binhex            nRooms= 140 ver=0x0200 expect=49586 actual=49586 OK
Titanic.binhex                   nRooms= 208 ver=0x0200 expect=73250 actual=73250 OK
```

21 of 22 match exactly. `Sampler.binhex` has **2 extra trailing bytes**
(`00 20 00 00 01 01 01 01` are its last eight); the room count in the header is authoritative,
so a Go loader must size its room slice from `nRooms` and **tolerate trailing slack** rather
than dividing the file length.

Every shipped house is `version == 0x0200`. `Demo House.binhex` is exactly 45 rooms /
16526 bytes, which matches the hard-coded `COMPILEDEMO` assertions at
`GliderPRO/Sources/HouseIO.c:349` (`byteCount != 16526L`) and `:382`.

### 20.2 `Point` is `(v, h)` — verified

QuickDraw `Point` is `{short v; short h;}`, so the **vertical** coordinate comes first on
disk. Every object whose `topLeft` is forced to a fixed row by the source confirms it:

```
kFloorVent     0x01 Art Museum     r0   o1  raw[0:4]=01 31 00 73 -> v=305 h=115  (expected v==305: MATCH)
kFloorTrans    0x35 Art Museum     r0   o4  raw[0:4]=01 2E 00 E9 -> v=302 h=233  (expected v==302: MATCH)
kCeilingTrans  0x36 Art Museum     r8   o2  raw[0:4]=00 06 00 52 -> v=6   h=82   (expected v==6:   MATCH)
kSewerGrate    0x05 CD Demo House  r0   o2  raw[0:4]=01 2F 01 78 -> v=303 h=376  (expected v==303: MATCH)
kWindowExRt    0x3D CD Demo House  r33  o9  raw[0:4]=00 40 01 F0 -> v=64  h=496  (expected v==64:  MATCH)
kDoorExRt      0x39 CD Demo House  r142 o5  raw[0:4]=00 00 01 F0 -> v=0   h=496  (expected h==kDoorExRtLeft(496): MATCH)
kFloorBlower   0x03 CD Demo House  r172 o2  raw[0:4]=01 30 00 82 -> v=304 h=130  (expected v==304: MATCH)
kCeilingBlower 0x04 CD Demo House  r173 o5  raw[0:4]=00 05 00 14 -> v=5   h=20   (expected v==5:   MATCH)
kCeilingVent   0x02 CD Demo House  r181 o5  raw[0:4]=00 08 01 40 -> v=8   h=320  (expected v==8:   MATCH)
kUpStairs      0x31 CD Demo House  r192 o1  raw[0:4]=00 1C 01 33 -> v=28  h=307  (expected v==28:  MATCH)
```

Matching constants: `kFloorVentTop` 305, `kCeilingVentTop` 8, `kFloorBlowerTop` 304,
`kCeilingBlowerTop` 5, `kSewerGrateTop` 303, `kCeilingTransTop` 6, `kFloorTransTop` 302,
`kStairsTop` 28, `kDoorExRtLeft` 496, `kWindowExTop` 64
(`GliderPRO/Headers/GliderDefines.h:467`-`:493`).

`Rect` is `{short top; short left; short bottom; short right;}` — also verified below by the
manhole records.

### 20.3 `furnitureType` rect order and the `kManhole` hot rect

`furnitureType` is `Rect bounds; short pict;` = 10 bytes
(`GliderPRO/Headers/GliderStructs.h:21`-`:24`). Decoding real `kManhole` (0x1D) records as
`(top, left, bottom, right, pict)`:

```
Davis Station  r15 o4  raw=01 2C 00 43 01 42 00 BE 00 00 -> rect(l=67,t=300,r=190,b=322)  123x22 pict=0
Davis Station  r15 o5  raw=01 2C 01 43 01 42 01 BE 00 00 -> rect(l=323,t=300,r=446,b=322) 123x22 pict=0
Davis Station  r16 o1  raw=01 2C 01 43 01 42 01 BE 00 00 -> rect(l=323,t=300,r=446,b=322) 123x22 pict=0
Davis Station  r32 o3  raw=01 2C 01 43 01 42 01 BE 00 00 -> rect(l=323,t=300,r=446,b=322) 123x22 pict=0
Demo House     r14 o0  raw=01 2C 00 43 01 42 00 BE 00 00 -> rect(l=67,t=300,r=190,b=322)  123x22 pict=0
Demo House     r14 o1  raw=01 2C 01 43 01 42 01 BE 00 00 -> rect(l=323,t=300,r=446,b=322) 123x22 pict=0
Empty House    r4  o1  raw=01 2C 01 43 01 42 01 BE 00 00 -> rect(l=323,t=300,r=446,b=322) 123x22 pict=0
Empty House    r4  o2  raw=01 2C 00 43 01 42 00 BE 00 00 -> rect(l=67,t=300,r=190,b=322)  123x22 pict=0
```

Every manhole is exactly **123 x 22**, matching the `srcRects[kManhole]` entry. Interpreting
the four shorts in any other order gives absurd rects (e.g. right < left), so the
`(top, left, bottom, right)` order is confirmed.

Applying `GetHotRect`'s `kManhole` transform (`GliderPRO/Sources/ObjectRects.c:640`-`:647`):

```c
bounds.left  += kGliderWide + 3;   /* +51 */
bounds.right -= kGliderWide + 3;   /* -51 */
bounds.top    = kFloorLimit - 1;   /* 311 */
bounds.bottom = kTileHigh;         /* 322 */
AddActiveRect(&bounds, kIgnoreGround, who, true, false);   /* doScrutinize = false */
```

| Art rect | Derived `kIgnoreGround` rect | Size |
|---|---|---|
| (l 67, t 300, r 190, b 322) | (l **118**, t **311**, r **139**, b **322**) | 21 x 11 |
| (l 323, t 300, r 446, b 322) | (l **374**, t **311**, r **395**, b **322**) | 21 x 11 |

So the manhole's permission window is **21 px wide** even though the art is 123 px wide, and
`doScrutinize` is **false**, so the raw (un-inset) 48-px-wide glider rect is tested against it.
Net effect: the glider's 48-px body must overlap a 21-px window whose top is 311 — i.e. the
glider must be almost exactly centred on the manhole and already at floor level.

### 20.4 Room `bounds` word — observed values

`roomType.bounds` is the `short` at room offset **28**. Decoded per section 13.6
(`boundsCode = bounds >> 1`; bits 1/2/4/8 = left/top/right/bottom open; unshifted bit 32 =
isStructure):

```
Art Museum r0    bg=3012 0x002F(47) >>1=0x0017 leftOpen=True  topOpen=True  rightOpen=True  bottomOpen=False isStructure=True
Art Museum r1    bg=3008 0x002F(47) >>1=0x0017 leftOpen=True  topOpen=True  rightOpen=True  bottomOpen=False isStructure=True
Art Museum r2    bg=3009 0x002F(47) >>1=0x0017 leftOpen=True  topOpen=True  rightOpen=True  bottomOpen=False isStructure=True
Art Museum r3    bg=3010 0x002F(47) >>1=0x0017 leftOpen=True  topOpen=True  rightOpen=True  bottomOpen=False isStructure=True
Art Museum r4    bg=3005 0x0027(39) >>1=0x0013 leftOpen=True  topOpen=True  rightOpen=False bottomOpen=False isStructure=True
Art Museum r5    bg=3004 0x002F(47) >>1=0x0017 leftOpen=True  topOpen=True  rightOpen=True  bottomOpen=False isStructure=True
Art Museum r6    bg=3007 0x002F(47) >>1=0x0017 leftOpen=True  topOpen=True  rightOpen=True  bottomOpen=False isStructure=True
Art Museum r7    bg=3003 0x002D(45) >>1=0x0016 leftOpen=False topOpen=True  rightOpen=True  bottomOpen=False isStructure=True
Art Museum r8    bg=3000 0x0019(25) >>1=0x000C leftOpen=False topOpen=False rightOpen=True  bottomOpen=True  isStructure=False
Art Museum r9    bg=3006 0x001B(27) >>1=0x000D leftOpen=True  topOpen=False rightOpen=True  bottomOpen=True  isStructure=False
Art Museum r10   bg=3001 0x001B(27) >>1=0x000D leftOpen=True  topOpen=False rightOpen=True  bottomOpen=True  isStructure=False
Art Museum r11   bg=3011 0x001B(27) >>1=0x000D leftOpen=True  topOpen=False rightOpen=True  bottomOpen=True  isStructure=False
Art Museum r15   bg=3002 0x0013(19) >>1=0x0009 leftOpen=True  topOpen=False rightOpen=False bottomOpen=True  isStructure=False
```

Note bit 0 (value 1) is set in every single one, confirming it is the "explicit bounds
present" marker that `bounds != 0` tests for. All the values listed above have
`background >= 3000` (user backgrounds), which is exactly the case in which
`DetermineRoomOpenings` consults the word (`GliderPRO/Sources/Room.c:825`).

For built-in backgrounds the word is *ignored* — the tile-code path is used instead — but it
is **not** reliably zero on disk. Re-parsing all 22 shipped `.binhex` houses gives 2273 rooms
with `background < 3000`, of which **27 have a non-zero `bounds`**: CD Demo House 23,
Grand Prix 2, Land of Illusion 1, Teddy World 1, with the raw values
`{1: 4, 3: 2, 9: 1, 15: 11, 23: 8, 35: 1}`. These are leftover editor state (a room whose
background was later changed from a user PICT back to a built-in one). A converter must
therefore drop `bounds` when `background < 3000` rather than assert it is zero.

Resulting collision parameters for these rooms:

| Room | `leftThresh` | `rightThresh` | `topOpen` | `bottomOpen` |
|---|---:|---:|---|---|
| Art Museum r0-r3, r5, r6 | -24 | 536 | true | false |
| Art Museum r4 | -24 | 500 | true | false |
| Art Museum r7 | 12 | 536 | true | false |
| Art Museum r8 | 12 | 536 | false | true |
| Art Museum r9-r14 | -24 | 536 | false | true |
| Art Museum r15 | -24 | 500 | false | true |

### 20.5 `'bnds'` resources — the 4-byte fallback

Houses with user backgrounds carry a `'bnds'` resource per background PICT in their
**resource** fork. `boundsType` is four one-byte `Boolean`s in the order left, top, right,
bottom (`GliderPRO/Headers/GliderStructs.h:266`-`:272`), so each resource is exactly 4 bytes.
Observed in `Castle o' the Air.binhex` (resource fork also contains
`'ICN#'`x1, `'PICT'`x12, `'icl4'`x1, `'icl8'`x1, `'ics#'`x1, `'ics4'`x1, `'ics8'`x1):

```
'bnds' 3000 len=4 bytes=01 00 01 00 -> boundCode= 5 leftOpen=T topOpen=F rightOpen=T bottomOpen=F | leftThresh=-24 rightThresh=536 hasFloor=T hasCeiling=T
'bnds' 3001 len=4 bytes=01 01 01 00 -> boundCode= 7 leftOpen=T topOpen=T rightOpen=T bottomOpen=F | leftThresh=-24 rightThresh=536 hasFloor=T hasCeiling=F
'bnds' 3002 len=4 bytes=01 00 01 00 -> boundCode= 5 leftOpen=T topOpen=F rightOpen=T bottomOpen=F | leftThresh=-24 rightThresh=536 hasFloor=T hasCeiling=T
'bnds' 3003 len=4 bytes=01 01 01 01 -> boundCode=15 leftOpen=T topOpen=T rightOpen=T bottomOpen=T | leftThresh=-24 rightThresh=536 hasFloor=F hasCeiling=F
'bnds' 3300 len=4 bytes=01 01 01 00 -> boundCode= 7 leftOpen=T topOpen=T rightOpen=T bottomOpen=F | leftThresh=-24 rightThresh=536 hasFloor=T hasCeiling=F
'bnds' 3301 len=4 bytes=01 01 01 01 -> boundCode=15 leftOpen=T topOpen=T rightOpen=T bottomOpen=T | leftThresh=-24 rightThresh=536 hasFloor=F hasCeiling=F
'bnds' 3302 len=4 bytes=01 01 01 00 -> boundCode= 7 leftOpen=T topOpen=T rightOpen=T bottomOpen=F | leftThresh=-24 rightThresh=536 hasFloor=T hasCeiling=F
'bnds' 3303 len=4 bytes=01 01 01 01 -> boundCode=15 leftOpen=T topOpen=T rightOpen=T bottomOpen=T | leftThresh=-24 rightThresh=536 hasFloor=F hasCeiling=F
'bnds' 3304 len=4 bytes=01 00 01 00 -> boundCode= 5 leftOpen=T topOpen=F rightOpen=T bottomOpen=F | leftThresh=-24 rightThresh=536 hasFloor=T hasCeiling=T
```

Note `boundCode` here is the *already-shifted* value (there is no marker bit in a `'bnds'`
resource), so it is directly comparable to `bounds >> 1`. IDs 3300+ are the "structure"
range (`kUserStructureRange` 3300) and get `isStructure = true` from the ID alone when
`bounds == 0` (`GliderPRO/Sources/Room.c:779`-`:784`). A Go port must ship a table of these
4-byte records extracted from each house's resource fork, or bake them into the house
converter.

### 20.6 `kDirt` tile arrays — the one-way vertical tunnels

`roomType.tiles[8]` are eight `short`s at room offset **36**. Real `kDirt` (background 2011)
rooms, with the `DetermineRoomOpenings` result derived from `tiles[0]`/`tiles[7]`:

```
CD Demo House  r150  tiles=[1, 4, 4, 4, 4, 5, 6, 7] floor=-2  leftThresh=WALL(12)  rightThresh=WALL(500)
CD Demo House  r151  tiles=[1, 4, 4, 4, 4, 2, 3, 7] floor=-1  leftThresh=WALL(12)  rightThresh=WALL(500)
Davis Station  r18   tiles=[1, 2, 3, 4, 4, 5, 6, 4] floor=0   leftThresh=WALL(12)  rightThresh=OPEN(536)
Davis Station  r19   tiles=[4, 5, 6, 4, 7, 0, 0, 0] floor=-1  leftThresh=OPEN(-24) rightThresh=OPEN(536)
Davis Station  r20   tiles=[4, 4, 4, 4, 4, 4, 4, 4] floor=-1  leftThresh=OPEN(-24) rightThresh=OPEN(536)
Davis Station  r45   tiles=[1, 5, 6, 4, 4, 5, 6, 7] floor=0   leftThresh=WALL(12)  rightThresh=WALL(500)
Davis Station  r46   tiles=[4, 4, 4, 4, 4, 4, 4, 4] floor=0   leftThresh=OPEN(-24) rightThresh=OPEN(536)
Davis Station  r47   tiles=[4, 4, 4, 4, 4, 5, 6, 7] floor=0   leftThresh=OPEN(-24) rightThresh=WALL(500)
Demo House     r38   tiles=[1, 5, 6, 4, 4, 5, 6, 7] floor=0   leftThresh=WALL(12)  rightThresh=WALL(500)
Empty House    r28   tiles=[1, 5, 6, 4, 4, 4, 4, 7] floor=-1  leftThresh=WALL(12)  rightThresh=WALL(500)
Empty House    r29   tiles=[1, 2, 3, 4, 4, 5, 6, 7] floor=0   leftThresh=WALL(12)  rightThresh=WALL(500)
Empty House    r32   tiles=[1, 5, 6, 4, 4, 5, 6, 4] floor=0   leftThresh=WALL(12)  rightThresh=OPEN(536)
Empty House    r33   tiles=[4, 4, 4, 4, 4, 4, 4, 4] floor=0   leftThresh=OPEN(-24) rightThresh=OPEN(536)
Empty House    r34   tiles=[4, 4, 4, 4, 4, 4, 4, 7] floor=0   leftThresh=OPEN(-24) rightThresh=WALL(500)
Grand Prix     r87   tiles=[0, 1, 5, 6, 2, 3, 7, 0] floor=0   leftThresh=OPEN(-24) rightThresh=OPEN(536)
Grand Prix     r90   tiles=[4, 4, 4, 4, 5, 6, 4, 7] floor=-1  leftThresh=OPEN(-24) rightThresh=WALL(500)
Grand Prix     r91   tiles=[4, 4, 4, 4, 4, 4, 4, 4] floor=-1  leftThresh=OPEN(-24) rightThresh=OPEN(536)
Grand Prix     r92   tiles=[0, 1, 4, 5, 6, 4, 4, 4] floor=-1  leftThresh=OPEN(-24) rightThresh=OPEN(536)
```

Observations that confirm the tile-code semantics of section 14.1:

* Codes **5 and 6 always appear as an adjacent pair** in that order (Davis Station r45 has two
  such pairs, at indices 1-2 and 5-6). This is the up-shaft.
* Codes **2 and 3 always appear as an adjacent pair** in that order (Empty House r29 indices
  1-2; Grand Prix r87 indices 4-5). This is the down-shaft.
* Code 1 only ever appears at index 0 (Davis Station r45, Demo House r38, Empty House r28/29/32,
  CD Demo r150/151) or immediately after a 0 (Grand Prix r87 index 1, r92 index 1) — it is the
  left wall cap. Code 7 similarly appears only as the last non-zero tile.
* Code 4 is the plain tunnel — no vertical escape.
* Code 0 is "no tile"; Davis Station r19 `[4,5,6,4,7,0,0,0]` and Grand Prix r87
  `[0,1,5,6,2,3,7,0]` show 0 used to blank out columns beyond the tunnel.
* Grand Prix r87 has **both** an up-pair (indices 2-3) and a down-pair (indices 4-5) in the
  same room.
* Note Grand Prix r87 gets `leftThresh = OPEN(-24)` because `tiles[0] == 0`, not 1 — the
  `kDirt` left test is `leftTile == 1` — yet the tile at index 0 is blank, so the glider can
  fly off the left edge into a room that visually has no opening. This is a real content
  quirk, not a parsing error.

### 20.7 `kRoof` tile arrays — the diagonal roofs

Real `kRoof` (background 2014) rooms:

```
CD Demo House  r36   tiles=[1, 6, 1, 2, 5, 6, 1, 6] floor=3
CD Demo House  r138  tiles=[1, 2, 3, 3, 3, 4, 3, 3] floor=2
CD Demo House  r146  tiles=[1, 2, 3, 3, 3, 4, 5, 6] floor=2
CD Demo House  r148  tiles=[4, 3, 4, 3, 3, 3, 5, 6] floor=2
CD Demo House  r149  tiles=[3, 4, 3, 3, 4, 3, 3, 4] floor=2
CD Demo House  r165  tiles=[3, 3, 3, 3, 3, 3, 5, 6] floor=8
CD Demo House  r166  tiles=[3, 3, 3, 3, 3, 3, 3, 3] floor=8
CD Demo House  r167  tiles=[3, 3, 3, 3, 3, 3, 3, 3] floor=8
CD Demo House  r168  tiles=[3, 3, 3, 3, 3, 3, 3, 3] floor=8
CD Demo House  r169  tiles=[1, 2, 3, 3, 3, 3, 3, 3] floor=8
CD Demo House  r177  tiles=[0, 3, 3, 3, 3, 3, 5, 6] floor=7
CD Demo House  r184  tiles=[3, 3, 3, 3, 3, 3, 3, 7] floor=6
Demo House     r6    tiles=[3, 3, 3, 3, 3, 3, 3, 7] floor=2
Demo House     r8    tiles=[3, 3, 3, 3, 3, 3, 3, 3] floor=2
Demo House     r9    tiles=[3, 3, 3, 3, 3, 3, 3, 3] floor=2
Demo House     r10   tiles=[1, 2, 3, 3, 3, 3, 3, 3] floor=2
```

* Code **3 is the flat roof** (dominant everywhere).
* Codes **1, 2 form the left-slope pair**, always in that order and always at the left end
  (CD Demo r138/r146/r169, Demo House r10). `CheckRoofCollision` tests them against 250 and
  186 respectively (`GliderPRO/Sources/Interactions.c:460`, `:470`).
* Codes **5, 6 form the right-slope pair**, always in that order and always at the right end
  (CD Demo r146/r148/r165/r177). Tested against 186 and 250 (`:480`, `:490`).
* Code **7 is the right wall cap** (CD Demo r184, Demo House r6) — and per section 13.2 it is
  also what makes `rightThresh = kRightWallLimit`.
* Code **4 appears interspersed** (CD Demo r138 index 5, r148 indices 0/2, r149 indices 1/4/7).
  `CheckRoofCollision` has **no case for 4**, so it falls to `default:` and **kills the glider
  outright** whenever `dest.bottom > 122` over such a tile. Code 4 must be a chimney or vent
  art that is meant to be solid.
* CD Demo r36 `[1,6,1,2,5,6,1,6]` is a saw-tooth roof mixing left and right slope halves out
  of pair order — the per-tile independence of `CheckRoofCollision` handles it.
* Code **0** (CD Demo r177 index 0) also hits `default:` and kills.

### 20.8 Object `data` union interpretations — observed records

The 12-byte `objectType` is `short what` + a 10-byte union
(`GliderPRO/Headers/GliderStructs.h:87`-`:101`). Real records for each variant used by the
collision code:

**`blowerType`** — `Point topLeft; short distance; Boolean initial, state; Byte vector, tall`
(`GliderPRO/Headers/GliderStructs.h:11`-`:18`). `kGreaseRt` (0x28) and `kLeftFan` (0x06):

```
kGreaseRt Art Museum   r3  o2  raw=01 1B 00 8C 01 54 00 00 00 01 | v=283 h=140 distance=340 initial=0 state=0 vector=0x00 tall=1
kGreaseRt Art Museum   r19 o5  raw=01 19 00 14 01 7E 00 00 01 01 | v=281 h=20  distance=382 initial=0 state=0 vector=0x01 tall=1
kLeftFan  CD Demo      r49 o17 raw=00 87 01 34 00 5F 00 00 08 01 | v=135 h=308 distance=95  initial=0 state=0 vector=0x08 tall=1
kLeftFan  CD Demo      r187 o3 raw=00 DB 01 00 00 8E 01 01 08 01 | v=219 h=256 distance=142 initial=1 state=1 vector=0x08 tall=1
kLeftFan  Grand Prix   r43 o6  raw=00 6F 01 83 00 A0 01 01 08 00 | v=111 h=387 distance=160 initial=1 state=1 vector=0x08 tall=0
```

`distance` is the grease **spill length** for grease jars (340, 382 px) and the wind-column
**length** for fans (95, 142, 160 px). `vector` is the bit field documented as
`| x | x | x | x | 8 | 4 | 2 | 1 |` = "F. lf. dn. rt. up"
(`GliderPRO/Headers/GliderStructs.h:16`) — `0x08` on every fan (left), `0x00`/`0x01` on grease
jars where the field is unused by the collision code.

**`bonusType`** — `Point topLeft; short length; short points; Boolean state, initial`
(`:27`-`:33`). `kInvisBonus` (0x2B) and `kStar` (0x2C):

```
kInvisBonus Art Museum r0  o3  raw=00 E0 00 F6 00 00 01 F4 00 01 | v=224 h=246 length=0 points=500 state=0 initial=1
kInvisBonus Art Museum r3  o3  raw=00 E7 00 32 00 00 01 F4 00 01 | v=231 h=50  length=0 points=500 state=0 initial=1
kStar       Art Museum r6  o14 raw=00 4A 00 BA 00 00 00 00 00 01 | v=74  h=186 length=0 points=0   state=0 initial=1
kStar       Art Museum r27 o2  raw=00 8E 00 F2 00 00 00 00 01 01 | v=142 h=242 length=0 points=0   state=1 initial=1
```

`points` = 500 (0x01F4) is what `HandleRewards` reads at
`GliderPRO/Sources/Interactions.c:920`. Confirms `state` is at data offset **8** and
`initial` at **9** — which is why writing `data.a.state` (offset **7**) in the prize branch of
`SetObjectState` (`GliderPRO/Sources/Objects.c:465`) corrupts the **low byte of `points`**
instead. For the Art Museum r0 record that would turn `points` from 0x01F4 (500) into 0x0100
(256) in the master copy.

**`transportType`** — `Point topLeft; short tall; short where; Byte who, wide`
(`:36`-`:42`). `kInvisTrans` (0x3F), `kDeluxeTrans` (0x40), `kUpStairs` (0x31):

```
kInvisTrans  CD Demo r32  o21 raw=00 7A 00 E3 00 43 05 82 08 02 | v=122 h=227 tall=67 where=1410 who=8   wide=2
kInvisTrans  CD Demo r33  o8  raw=00 8D 00 EC 00 20 05 82 FF 00 | v=141 h=236 tall=32 where=1410 who=255 wide=0
kDeluxeTrans Art Museum r19 o6 raw=00 2B 00 5C 51 38 18 A4 0D 11 | v=43  h=92  tall=0x5138 where=6308 who=13 wide=0x11
kDeluxeTrans Art Museum r26 o19 raw=00 05 00 0A 13 11 15 8C 03 00 | v=5  h=10  tall=0x1311 where=5516 who=3  wide=0x00
kDeluxeTrans Art Museum r36 o11 raw=00 B5 00 9B 11 10 1A 37 09 11 | v=181 h=155 tall=0x1110 where=6711 who=9 wide=0x11
kUpStairs    CD Demo r192 o1  raw=00 1C 01 33 00 00 FF FF FF 00 | v=28 h=307 tall=0 where=-1 who=255 wide=0
```

* `kUpStairs` uses `where == -1` (0xFFFF) and `who == 255` (0xFF) as the **unlinked**
  sentinels. `tall` and `wide` are 0 because the stairs rect is derived from `srcRects`, not
  from the record.
* `kInvisTrans` uses `tall` and `wide` literally: `where = 1410` is a packed link
  (room 1410 / object index in the high nibble scheme), `who = 8`, `wide = 2`. The derived hot
  rect for the first record is (l 227, t 122, r 293, b 189) — `wide = 2` giving 66 px of width
  and `tall = 67`.
* `kDeluxeTrans` **re-uses `tall` as two packed bytes** (art size / 4) and `wide` as two packed
  nibbles (`state` low, `initial` high): `0x5138` -> 0x51 * 4 = 324 wide by 0x38 * 4 = 224
  tall; `wide = 0x11` -> `state = 1`, `initial = 1`; `wide = 0x00` -> both 0. `0x1311` ->
  76 x 68; `0x1110` -> 68 x 64. `SetObjectState`'s nibble dance
  (`GliderPRO/Sources/Objects.c:499`-`:519`) operates only on the low nibble.

**`switchType`** — `Point topLeft; short delay; short where; Byte who, type`
(`:45`-`:51`). `kTrigger` (0x47) and `kLightSwitch` (0x41):

```
kTrigger     CD Demo r30 o13 raw=01 22 01 36 00 00 05 E5 0A 03 | v=290 h=310 delay= 0 where=1509 who=10 type=3 -> timer= 0
kTrigger     CD Demo r30 o14 raw=01 22 01 36 00 32 05 E5 0A 03 | v=290 h=310 delay=50 where=1509 who=10 type=3 -> timer=150
kTrigger     CD Demo r30 o15 raw=01 22 01 36 00 00 05 E5 09 03 | v=290 h=310 delay= 0 where=1509 who= 9 type=3 -> timer= 0
kTrigger     CD Demo r30 o16 raw=01 22 01 36 00 34 05 E5 09 03 | v=290 h=310 delay=52 where=1509 who= 9 type=3 -> timer=156
kTrigger     CD Demo r30 o17 raw=01 22 01 36 00 00 05 E5 08 03 | v=290 h=310 delay= 0 where=1509 who= 8 type=3 -> timer= 0
kTrigger     CD Demo r30 o18 raw=01 22 01 36 00 36 05 E5 08 03 | v=290 h=310 delay=54 where=1509 who= 8 type=3 -> timer=162
kLightSwitch CD Demo r5   o11 raw=00 97 01 B4 00 00 04 55 09 02 | v=151 h=436 delay=0 where=1109 who= 9 type=2
kLightSwitch CD Demo r139 o2  raw=00 8C 01 0E 00 00 12 C9 FF 00 | v=140 h=270 delay=0 where=4809 who=255 type=0
kLightSwitch CD Demo r172 o4  raw=00 8C 00 CE 00 00 05 EB 03 01 | v=140 h=206 delay=0 where=1515 who= 3 type=1
kLightSwitch CD Demo r195 o11 raw=00 63 00 82 00 00 03 99 0E 00 | v=99  h=130 delay=0 where=921  who=14 type=0
```

Two important confirmations:

1. `ArmTrigger` computes `timer = data.e.delay * 3` (`GliderPRO/Sources/Triggers.c:49`). The
   observed delays 50/52/54 give 150/156/162 frames. Six triggers in one room at the same
   `(v, h)` = 290, 310 — a stack of coincident trigger rects with staggered delays, so a single
   glider pass fires a cascade.
2. **`kLightSwitch` records with `type = 0`, `1` and `2` all exist** — matching
   `kToggle` 0 / `kForceOn` 1 / `kForceOff` 2. But **`kTrigger` records carry `type = 3`**,
   which is not a valid `SetObjectState` action. `HandleSwitches` passes `data.e.type`
   straight through (`GliderPRO/Sources/Interactions.c:998`), so a trigger routed through
   `HandleSwitches` would hit no `case` and leave `changed` uninitialized. Triggers are not
   routed through `HandleSwitches` (they use `kTriggerIt` -> `ArmTrigger`), so this is latent —
   but it means a Go port **must not** assume `data.e.type` is always a valid action code. The
   safe reading is `type == 3` means "trigger link" (compare
   `kTriggerLinkOnly` = 4, `kSwitchLinkOnly` = 3, `kTransportLinkOnly` = 5,
   `GliderPRO/Headers/GliderDefines.h:463`-`:465`).

**`applianceType`** — `Point topLeft; short height; Byte byte0, delay; Boolean initial, state`
(`:64`-`:71`). Note the field order: `byte0` comes **before** `delay`.

```
kMicrowave Art Museum       r47 o7  raw=00 E7 00 E8 00 00 07 00 01 01 | v=231 h=232 height=0 byte0=0x07 delay=0 initial=1 state=1
kMicrowave CD Demo House    r51 o16 raw=00 F2 01 5A 00 00 06 00 01 01 | v=242 h=346 height=0 byte0=0x06 delay=0 initial=1 state=1
kMicrowave CD Demo House    r195 o15 raw=00 84 00 8C 00 00 06 00 01 01| v=132 h=140 height=0 byte0=0x06 delay=0 initial=1 state=1
kMicrowave California/Bust! r11 o3  raw=00 C1 01 1A 00 00 07 00 00 00 | v=193 h=282 height=0 byte0=0x07 delay=0 initial=0 state=0
kMicrowave Fun House        r10 o4  raw=00 F5 00 0E 00 00 04 00 00 00 | v=245 h=14  height=0 byte0=0x04 delay=0 initial=0 state=0
kToaster   Art Museum       r22 o12 raw=00 B0 00 90 00 A5 00 0B 01 01 | v=176 h=144 height=165 byte0=0x00 delay=11 initial=1 state=1
kToaster   Art Museum       r22 o13 raw=00 B5 00 FE 00 9E 00 10 01 01 | v=181 h=254 height=158 byte0=0x00 delay=16 initial=1 state=1
kToaster   Art Museum       r56 o4  raw=01 1D 00 7A 01 04 00 14 01 01 | v=285 h=122 height=260 byte0=0x00 delay=20 initial=1 state=1
kOutlet    CD Demo House    r201 o11 raw=00 70 00 38 00 00 00 0B 01 01| v=112 h=56  height=0   byte0=0x00 delay=11 initial=1 state=1
kOutlet    Grand Prix       r86 o3  raw=00 2D 00 F4 00 00 00 13 01 01 | v=45  h=244 height=0   byte0=0x00 delay=19 initial=1 state=1
```

* `kMicrowave.byte0` is the **kill mask** read by `HandleMicrowaveAction`
  (`GliderPRO/Sources/Interactions.c:1171`): bit `0x0001` kills `bandsTotal`, `0x0002` kills
  `batteryTotal`, `0x0004` kills `foilTotal`. Observed values: **0x07** (all three, Art
  Museum r47, California or Bust! r11), **0x06** (battery + foil, CD Demo r51/r195),
  **0x04** (foil only, Fun House r10/r11). No shipped microwave uses 0x00, 0x01, 0x02, 0x03 or
  0x05.
* `kToaster.height` is the toast's launch height in pixels (165, 158, 260), from which
  `AddDynamicObject` reverse-engineers the initial velocity by summing 1+2+3+...
  (`GliderPRO/Sources/Dynamics3.c:224`-`:233`). For height 165 that loop gives velocity 18
  (since 1+..+18 = 171 >= 165), so toast launches at `vVel = -18`.
* `kToaster.delay` (11, 16, 20) is multiplied by 3 for the idle frame count
  (`GliderPRO/Sources/Dynamics3.c:234`).
* `kOutlet.delay` (11, 19) becomes `count = (delay * 6) / kTicksPerFrame` = `delay * 3`
  (`GliderPRO/Sources/Dynamics3.c:316`) — 33 and 57 frames between zaps.

**`enemyType`** — `Point topLeft; short length; Byte delay, byte0; Boolean initial, state`
(`:74`-`:81`). Note `delay` and `byte0` are in the **opposite order** from `applianceType`.

```
kBalloon Art Museum r27 o4 raw=00 92 01 58 00 00 11 00 00 00 | v=146 h=344 length=0   delay=17 byte0=0x00 initial=0 state=0
kBalloon Art Museum r27 o5 raw=00 92 01 2A 00 00 0B 00 00 00 | v=146 h=298 length=0   delay=11 byte0=0x00 initial=0 state=0
kBalloon Art Museum r27 o6 raw=00 92 00 C0 00 00 07 00 00 00 | v=146 h=192 length=0   delay= 7 byte0=0x00 initial=0 state=0
kDartLf  Art Museum r26 o4 raw=00 26 01 C0 00 00 03 00 01 01 | v=38  h=448 length=0   delay= 3 byte0=0x00 initial=1 state=1
kDartLf  Art Museum r26 o5 raw=00 43 01 C0 00 00 0D 00 01 01 | v=67  h=448 length=0   delay=13 byte0=0x00 initial=1 state=1
kDrip    Art Museum r23 o1 raw=00 1A 00 92 00 F5 02 00 01 01 | v=26  h=146 length=245 delay= 2 byte0=0x00 initial=1 state=1
kDrip    Art Museum r23 o2 raw=00 14 00 DA 00 FB 07 00 01 01 | v=20  h=218 length=251 delay= 7 byte0=0x00 initial=1 state=1
kDrip    Art Museum r23 o3 raw=00 16 01 42 00 F9 0F 00 01 01 | v=22  h=322 length=249 delay=15 byte0=0x00 initial=1 state=1
```

`kDrip.length` (245, 251, 249) is the **floor line** stored into `dinahs[].position`; the drip
falls from `topLeft.v` (26, 20, 22) to that line. `delay` sets the respawn period.

### 20.9 `kMaxRoomObs` and the object slot loop

`roomType.objects[24]` at room offset **60**, 12 bytes each, filling the room record to
`60 + 288 = 348`. `numObjects` is the `short` at offset 58, but the hot-rect builder loops the
**full 24 slots** and tests `IsThisValid` rather than trusting `numObjects`
(`GliderPRO/Sources/Objects.c:266`, `:283`-`:286`). Objects with `what == kObjectIsEmpty`
(-1) or `0` are skipped. A Go port must iterate all 24 and skip invalid — using `numObjects`
as a loop bound would drop objects in houses where the editor left gaps.

---

## 21. Off-by-one catalogue

Every arithmetic detail in the collision path where a Go port can silently diverge.

| # | Location | The trap | Correct behaviour |
|---:|---|---|---|
| 1 | `SectGlider` (`GliderPRO/Sources/Interactions.c:118`-`:127`) | overlap uses `<`/`>` on opposite edges, so **touching counts as a hit** | `a.bottom < b.top` -> miss. Equality (`a.bottom == b.top`) is a **HIT**. Go's `image.Rectangle.Overlaps` and QuickDraw's `SectRect` both treat touching as a miss — do not use either |
| 2 | `SectGlider` (`:110`-`:116`) | inset is `left += 5; top += 5; right -= 5; bottom -= 5` — an `InsetRect(&r, 5, 5)`, applied to a 48x20 glider giving 38x10 | do not inset by 5 on only one axis, and do not use `image.Rectangle.Inset` on the *raw* rect if the mode is burning (see #3) |
| 3 | `SectGlider` (`:107`-`:108`) | burning adds `glideBounds.top += 6` **before** the inset, so a burning glider's effective top is `dest.top + 11` | apply the burn offset first, then the inset |
| 4 | `GliderInRect` (`:140`-`:149`) | **strict containment**, and against the **raw** `dest` with no inset and no burn offset | `dest.left < r.left` -> fail; equality passes |
| 5 | `GliderHitTop` (`:60`-`:66`) | insets by 5 **then** expands horizontally by `wasHVel`: `left -= wasHVel; right -= wasHVel`. Both edges move the **same** direction, it is a shift not an expansion. Unlike `SectGlider` it does **not** apply the `kGliderBurning` `top += 6` offset, so a burning glider's foil hit box differs between the two primitives | reproduce literally; the sign of `wasHVel` decides which side the sweep covers |
| 6 | `CheckEscapeUp*` tile index (`:200`-`:204`) | `dest.left >> 6` is an **arithmetic** right shift; `-1 >> 6 == -1`, but `-1 / 64 == 0` | use `>>`, not `/` |
| 7 | `CheckRoofCollision` (`:454`-`:455`) | the range test is `(offset >= 0) && (offset <= 7)`, inclusive of 7 — but the up/down tunnel tests use `< 8` (`:203`, `:315`; a bare literal, not `kNumTiles`). Same set, two spellings | either is fine, but `<= 8` / `<= kNumTiles` would be wrong |
| 8 | `CheckRoofCollision` (`:460`) | in-tile x is `(dest.left + 24) - (offset << 6)`, using the glider's **centre** `dest.left + kHalfGliderWide`, while `offset` was computed from the **same** centre — so x is always in 0..63 | do not mix `dest.left` and the centre |
| 9 | `CheckRoofCollision` right slopes (`:480`, `:490`) | `(64 - x)` mirrors the in-tile x; the constants swap (186 for code 5, 250 for code 6) relative to the left slopes | 1->250, 2->186, 5->186, 6->250 |
| 10 | `CheckEscapeLeft` (`:589`-`:590`) | `offset = kLeftWallLimit - dest.left` then `hVel = -hVel + offset` — a **plus** | the right-wall mirror also uses `+ offset` with a negative offset (`:679`) |
| 11 | `GliderHitTop` (`:89`-`:93`) | `hVel = -hVel - offset` — a **minus**, and `offset` has a magic `+ 2` | do not unify with #10 |
| 12 | `BounceGlider` (`:157`, `:159`-`:162`) | uses the **raw** `dest` for the escape-distance comparison even though the hit test that led here used the inset rect | do not inset in `BounceGlider` |
| 13 | `CheckEscapeDown` death (`:411`) | `vVel = kFloorLimit - dest.bottom` **and** `StartGliderFadingOut` — the clamp still runs on a dying glider | keep both |
| 14 | `kManhole` hot rect (`GliderPRO/Sources/ObjectRects.c:642`-`:645`) | `bounds.left += kGliderWide + 3` = **+51**, `bounds.top = kFloorLimit - 1` = **311**, `bounds.bottom = kTileHigh` = **322** | 51, not 48; 311, not 312 |
| 15 | `kBooks` hot rect (`GliderPRO/Sources/ObjectRects.c:636`) | `bounds.right -= 2` only, no other adjustment | asymmetric on purpose |
| 16 | `MoveGlider` clamp order (`GliderPRO/Sources/Player.c:96`-`:99` negative branch, `:113`-`:116` positive branch) | `hVel` is clamped **before** `wasHVel` is recorded | `wasHVel` is the clamped value |
| 17 | `MoveGlider` (`:92`) | `vDesiredVel` resets to `kGravity` (3), **not** 0 | forgetting this makes the glider float |
| 18 | `MoveGlider` (`:78`) | `hDesiredVel` resets to **0** | asymmetric with #17 |
| 19 | `MoveGlider` (`:101`-`:104` vs `:118`-`:121`) | the two horizontal branches assign `whole.left`/`whole.right` in **opposite order**, so `whole` is the swept union | do not simplify to a single branch without recomputing the union |
| 20 | `HandleRewards` velocity damping (`GliderPRO/Sources/Interactions.c:774`-`:775` vs `:839`-`:840`) | clocks/`kInvisBonus` use `/= 4`; `kPaper`/`kBattery`/`kBands`/`kFoil`/`kHelium` use `/= 2`; `kStar` uses neither | integer division truncating **toward zero**, not `>>` |
| 21 | `AddFlyingPoint` velocity (`:773`) | passes `hVel / 2, vVel / 2` — the **pre-damping** velocities, evaluated before the `/= 4` on the next lines | order matters |
| 22 | `kInvisBonus` (`:920`-`:921`) | `points` is read **before** `SetObjectState` is called | reversing these reads the corrupted value (see section 22) |
| 23 | Battery/helium (`:861`-`:865`, `:967`-`:971`) | replace-vs-add is decided by the **sign** of `batteryTotal`, and the two-player doubling is applied **after** | a two-player battery grab while holding helium yields exactly +100, not +50 |
| 24 | `CheckDynamicCollision` foil loss (`GliderPRO/Sources/Dynamics.c:58`) | `foilTotal--` only on `evenFrame` | half-rate |
| 25 | `kDissolveIt` foil loss (`GliderPRO/Sources/Interactions.c:1232`) | decrements **every** frame, and `GliderHitTop` may have already decremented at `:82` — a double decrement in one frame | see section 22 |
| 26 | `AddBand` (`GliderPRO/Sources/RubberBands.c:264`-`:267`) | band rect is `{h-8, v-3, h+8, v+3}` = **16 x 6**, then shifted ±32 | not centred on the glider |
| 27 | `AddBand` recoil (`:282`) | `thisGlider->hVel -= (hVel / 2)` = ∓10, applied to `hVel` directly (not `hDesiredVel`) | bypasses the impulse ramp |
| 28 | `HandleBands` gravity (`:219`-`:224`) | `vVel++` only when `count >= kBandFallCount` (4), and `count` resets to 0 | 1 unit per 4 frames |
| 29 | `CheckBandCollision` wall snap (`:47`-`:48`) | re-derives width as a literal **16** | must match `AddBand` |
| 30 | `CheckBandCollision` phase 4 (`:195`-`:203`) | uses **fixed** `kLeftWallLimit`/`kRightWallLimit`/`kFloorLimit`, not the room's thresholds | bands die at 12/500/312 even in open rooms |
| 31 | `CheckBandCollision` glider push (`:167`) | `theGlider.hVel += (bands[who].hVel / 2)`, then `bands[who].hVel = 0` — `vVel` untouched | band keeps falling |
| 32 | `HandleGrease` seed rect (`GliderPRO/Sources/Grease.c:61`-`:66`) | 2x2 seed, `playOrigin` subtracted **before** the anchor offset | net result is room coordinates |
| 33 | `HandleGrease` spread (`:95`, `:103`) | trail grows **2 px/frame**, and `hotSpots[].bounds.right += 2` / `.left -= 2` directly | the rect stays exactly 2 px tall |
| 34 | `RedrawAllGrease` (`GliderPRO/Sources/Grease.c:283`) | tests `(src.bottom - src.top) == 2` to identify a grease trail | exact equality |
| 35 | `AddGrease` (`:221`-`:224`, `:234`-`:241`) | jar art shifted ∓8, trail start ±4 from the art edge, stop = `start_edge ± distance` | note `stop` is measured from the **art** edge, not from `start` |
| 36 | `HandleToast` gravity (`GliderPRO/Sources/Dynamics.c:356`) | `vVel++` **every** frame, unlike every other dynamic which is `if (evenFrame) vVel++` | toast falls twice as fast |
| 37 | `kFloorVentLift` / `kCeilingVentDrop` (`GliderPRO/Sources/Interactions.c:13`-`:14`) | -6 and **+8** — not symmetric | |
| 38 | `kFanStrength` (`:15`) | fans use `hDesiredVel += ±12` (accumulating), while enemies use `hDesiredVel = ±8` (assigning) | two fans stack to 24, then `MoveGlider` clamps `hVel` to 16 |
| 39 | Room entry Y (`GliderPRO/Sources/Transit.c:176`, `:185` left edge; `:208`, `:217` right edge) | `kGliderStartsDown + (short)thisRoom->leftStart - 2` — the `- 2` and the `(short)` cast of an unsigned `Byte` | range 0..255 |
| 40 | `OffsetGlider` (`GliderPRO/Sources/Player.c:1450` `kToRight`, `:1459` `kToLeft`, `:1468` `kAbove`) | `kToRight` offsets by **+kRoomWide**; the direction naming is inverted relative to travel. `dest`, `destShadow`, `whole` and `wholeShadow` are all shifted, and `whole` is reassigned from `dest` (collapsing the swept union) | |
| 41 | `WebGlider` (`GliderPRO/Sources/Interactions.c:1749`-`:1752`) | `hDist = ((webBounds->right - dest.right) + (webBounds->left - dest.left)) >> 3` — a **signed** `>> 3`, i.e. floor division by 8 | `>> 3` on a negative sum rounds toward -inf; `/ 8` would round toward 0 |
| 42 | `HandleTriggers` (`GliderPRO/Sources/Triggers.c:87`-`:92`) | `timer--` then fire at `<= 0`; `timer = data.e.delay * 3` | a delay of 0 fires on the **first** frame after arming |

---

## 22. Known upstream bugs

These are defects in the original 1.0.4 source. A Go port must decide, per item, whether to
reproduce or fix; the recommendation is given.

### 22.1 Uninitialized `bounds` passed to `AddSparkle`

`GliderPRO/Sources/Interactions.c:1050`. `HandleSwitches` declares `Rect bounds;` at `:987`,
never assigns it, and passes `&bounds` to `AddSparkle` in the prize-restore case. The sparkle
lands at whatever was on the 68k stack. **Fix**: use `who->bounds` offset into the linked
object's rect, or skip the sparkle. Divergence is cosmetic.

### 22.2 `SetObjectState` prize branch writes the wrong union member

`GliderPRO/Sources/Objects.c:465` writes `masterObjects[local].theObject.data.a.state`
(`blowerType.state`, data offset 7) instead of `.data.c.state` (`bonusType.state`, data
offset 8). Effects: (a) the master copy's `state` is never cleared; (b) the **low byte of
`bonusType.points`** is zeroed. Verified against a real record: Art Museum r0 o3 has
`points = 0x01F4` (500), which the write would turn into `0x0100` (256). Unobservable in
normal play only because `HandleRewards` reads `points` at
`GliderPRO/Sources/Interactions.c:920` **before** calling `SetObjectState` at `:921`, and
because `hotSpots[].isOn` is cleared through a separate path (`GliderPRO/Sources/Objects.c:470`
and `GliderPRO/Sources/Interactions.c:779` etc.). **Fix**: write `data.c.state`.

### 22.3 `FireTrigger`'s else branch dereferences `masterObjects[-1]`

`GliderPRO/Sources/Triggers.c:174`-`:193`. The `else` of
`if (masterObjects[triggerIs].localLink != -1)` sets
`triggeredIs = masterObjects[triggerIs].localLink` at `:178` — which is **-1** by construction
— and then indexes `masterObjects[triggeredIs]` in the grease case at `:182`-`:190`.
**Fix**: skip the branch entirely (it can only be reached for a trigger whose target is
outside the 3x3 neighborhood, in which case there is nothing local to act on).

### 22.4 `ArmTrigger` indexes `masterObjects[]` with an object slot number

`GliderPRO/Sources/Triggers.c:50`:

```c
triggers[where].what = masterObjects[triggers[where].object].theObject.what;
```

`triggers[where].object` was just set to `masterObjects[whoLinked].objectLink` at `:47` — a
**room object slot** (0..23), not a master-object index (0..215). So `.what` is read from an
unrelated master object. `.what` is never actually consumed (`FireTrigger` switches on
`masterObjects[triggeredIs].theObject.what`, recomputed at `:110`), so the bug is inert.
**Fix**: drop the field.

### 22.5 Double foil decrement in `kDissolveIt`

`GliderPRO/Sources/Interactions.c:1224`-`:1240`. `GliderHitTop` already does
`foilTotal--` at `:82` when the glider hit the obstacle from the side; the caller then does
`foilTotal--` again at `:1232` in the `else` branch. But look at the control flow: the caller's
decrement is in the `else` of `if (GliderHitTop(...))`, so if `GliderHitTop` returned **true**
(a top hit) the glider dies and no second decrement happens; if it returned **false**
`GliderHitTop` **already decremented** at `:82` and then the caller decrements again. So a
side-on foil hit costs **2 foil**, not 1. **Fix**: remove one. Reproducing it changes how many
hits a foil survives, which is gameplay-visible.

### 22.6 `HandleMicrowaveAction` never sets `stillOver`

`GliderPRO/Sources/Interactions.c:1159`-`:1195` early-returns on `who->stillOver`
(`:1164`) but never assigns it. Since the sweep clears `stillOver` whenever the glider is not
overlapping (`:1683` one-player, `:1674` two-player), the microwave fires **every frame** the
glider is inside it. With `byte0 = 0x07` that means bands/battery/foil are re-zeroed each
frame and `kMicrowavedSound` plays every frame — but only the **first** frame does anything,
because the code checks `if (bandsTotal > 0)` etc. before zeroing (`:1173`, `:1179`, `:1185`).
So the practical effect is only the repeated sound. **Fix**: set `who->stillOver = true` at the
end. Note doing so also stops the sound repeat, an audible divergence.

### 22.7 `CheckEscapeDownTwo` is missing the `kDirt` out-of-range `else`

Compare `CheckEscapeDown` (`GliderPRO/Sources/Interactions.c:394`-`:430`), which has an
`else` at `:417` for `leftTile`/`rightTile` out of `[0, 8)`, against
`CheckEscapeDownTwo` (`:315`-`:358`), which does **not**. In two-player mode, a glider in a
`kDirt` room whose `dest` spans a tile index < 0 or >= 8 falls through the floor with **no
clamp, no death and no room transition** — it just keeps accelerating downward. Reachable when
the room has `leftThresh == kNoLeftWallLimit` (-24), letting `dest.left` go to -24 and
`dest.left >> 6 == -1`. **Fix**: add the missing `else` mirroring `:417`-`:430`.

### 22.8 `HandleOutlet` passes the wrong `doOffset` in the `onePlayerLeft` branch

`GliderPRO/Sources/Dynamics.c:539`, `:541` pass `false` while `:545`, `:546`, `:550` pass
`true`. The outlet's `dest` is **screen**-relative (`GliderPRO/Sources/Dynamics3.c:310`-
`:312`), so `true` is correct and the `onePlayerLeft` path tests the zap 64/79 px off (on a
640x480 screen). **Fix**: pass `true` in all five. This is the only screen-relative dynamic
that collides.

### 22.9 `kLgTrigger` used as an *action* case label

`GliderPRO/Sources/Interactions.c:1352`. `HandleHotSpotCollision` switches on
`who->action`, whose legal values are the 28 action codes 0..27
(`GliderPRO/Headers/GliderDefines.h:282`-`:309`). `kLgTrigger` is an **object** code
(0x48 = 72, `GliderPRO/Headers/GliderDefines.h:384`), so this case is unreachable. **Fix**:
delete; the intended label was presumably `kTriggerIt`.

### 22.10 `WebGlider`'s burning early-out is dead code

`GliderPRO/Sources/Interactions.c:1741`-`:1747` opens `WebGlider` with
`if ((mode == kGliderBurning) && GliderInRect(thisGlider, webBounds))` → fade out and
`return`. That guard can never fire. `WebGlider` has exactly one call site, the `kWebIt` case
at `GliderPRO/Sources/Interactions.c:1602`-`:1612`, and it is guarded by

```c
if ((GliderInRect(thisGlider, &who->bounds)) && (thisGlider->mode != kGliderBurning))
	WebGlider(thisGlider, &who->bounds);            // :1605
else if ((thisGlider->mode == kGliderBurning) && (GliderInRect(...)))
	{ wasMode = 0; StartGliderFadingOut(...); PlayPrioritySound(kFadeOutSound, ...); }
```

so a burning glider is killed by the caller's `else if` and never enters the function; a
non-burning one fails the internal test. The two blocks are byte-identical duplicates.

Note also that this path uses `GliderInRect` — **strict containment** (`:134`-`:150`: any
glider edge outside the rect returns `false`) — on *both* the caller test and the dead
internal test. It is not the inclusive `SectGlider` overlap used elsewhere, so a glider only
partially over a web is neither webbed nor killed. **Fix**: none needed; a port may drop the
dead branch, but must keep the `GliderInRect` containment semantics.

### 22.11 `DidBandHitDynamic` returns an uninitialized value when `numBands == 0`

`GliderPRO/Sources/Dynamics.c:80`-`:105`. All four call sites guard with `numBands > 0`, so
unreachable. **Fix**: initialise to `false`.

### 22.12 `HandleBall` clobbers the global `evenFrame`

`GliderPRO/Sources/Dynamics2.c:420` (in `HandleBall`, `:358`): `evenFrame = true;` when a
ball starts bouncing.
`evenFrame` is the global frame parity toggled in `Play.c:435` and read by **everything** —
toast animation, foil decrement rate, dynamic gravity, `WebGlider`'s pull. A ball starting a
bounce therefore perturbs global timing for one frame. **Fix**: remove; use a per-dynamic
counter.

### 22.13 `IsRectLeftOfRect` compares the wrong quantities

`GliderPRO/Sources/RectUtils.c:181`-`:190`. The whole body is:

```c
offset = (rect1->right - rect1->left) - (rect2->right - rect2->left) / 2;
if ((rect1->left) < (rect2->left + offset))
	return (true);
```

There are **no midpoints**: `offset` mixes `rect1`'s *full* width with *half* of `rect2`'s
width (C precedence binds `/ 2` to the second width only, not to the difference), and the
test then compares `rect1->left` against `rect2->left + offset`. So the function does not
answer "is rect1 left of rect2" in any geometric sense — it is biased by both widths, and for
a 48-px-wide glider against a narrow enemy it returns `true` even when the glider is well to
the *right* of it. See section 1.5 for the same expression written out.

The consequence is in `CheckDynamicCollision` (`GliderPRO/Sources/Dynamics.c:53`-`:56`):
`true` sets `hDesiredVel = kShoveVelocity` (**+8**, shove right, `:54`) and `false` sets
`-kShoveVelocity` (**-8**, shove left, `:56`), with `kShoveVelocity` = 8 at
`GliderPRO/Sources/Dynamics.c:17`. Because the predicate is skewed, a glider can be shoved
*into* the enemy rather than away from it. **Fix**: none; reproduce the exact expression for
fidelity — a "correct" midpoint comparison changes observable gameplay.

### 22.14 Unreachable dead code in `GetObjectRect`

`GliderPRO/Sources/ObjectRects.c:68`-`:71`. A `case` after an unconditional
`return`/`break` chain. Inert.

### 22.15 Dead stores `collided = true` in `CheckBandCollision`

`GliderPRO/Sources/RubberBands.c:51`, `:60`. The hot-spot loop immediately below overwrites
`collided` on its first iteration. Inert, but a Go port that hoists `collided` out of the loop
would change behaviour when `nHotSpots == 0`.

### 22.16 `GetInput` does not update `tipped` when both direction keys are held

`GliderPRO/Sources/Input.c:306`-`:309`. With right down and left also down, the code calls
`ToggleGliderFacing(thisGlider)` (`:308`) and sets `heldLeft = true` (`:309`) but **never
assigns `tipped`**, whereas the right-only `else` branch at `:314`-`:316` does
(`tipped = (facing == kFaceLeft)`). So `hDesiredVel` is left at 0 and `tipped` keeps its
previous value, making the glider's sprite and `AddBand`'s initial `vVel`
(`GliderPRO/Sources/RubberBands.c:260`) stale. Cosmetic plus a 2 px/frame band velocity
difference.

### 22.17 `kSparkle` computes an unused rect in `GetHotRect`

`GliderPRO/Sources/ObjectRects.c:719`-`:725`. The rect is built and then not passed to
`AddActiveRect`. Inert.

### 22.18 Summary — which to fix in a Go port

| Bug | Reproduce | Fix |
|---|---|---|
| 22.1 uninitialized sparkle `bounds` | | fix (draw nothing) |
| 22.2 wrong union member | | fix |
| 22.3 `masterObjects[-1]` | | fix (skip branch) |
| 22.4 `ArmTrigger` bad index | | fix (drop field) |
| 22.5 double foil decrement | **reproduce** (gameplay-visible foil durability) | |
| 22.6 microwave `stillOver` | **reproduce** (audible) | |
| 22.7 missing `kDirt` else in 2P | | fix (otherwise a fall-through-the-world) |
| 22.8 `HandleOutlet` `doOffset` | | fix |
| 22.9 `kLgTrigger` case | | delete |
| 22.10 `WebGlider` early-out | | may delete (provably dead) |
| 22.11 `DidBandHitDynamic` | | fix (return false) |
| 22.12 `evenFrame` clobber | **reproduce** (timing-visible) | |
| 22.13 `IsRectLeftOfRect` skew | **reproduce** | |
| 22.14-22.15, 22.17 dead code | | omit |
| 22.16 `tipped` staleness | **reproduce** | |

---

## Open questions

1. **Are `hotSpots[]` indices stable across a `DrawLocale` redraw?** `activeRectEscaped`
   (`GliderPRO/Sources/Interactions.c:42`) saves a hot-spot index in one frame and compares it
   in a later frame (`:1405`, `:1452`, `:1495`, `:1534`, `:1569`). `nHotSpots` is reset to 0
   and the list rebuilt whenever `DrawLocale` runs (`GliderPRO/Sources/Objects.c:307`). If a
   light is switched off mid-transport, `RedrawRoomLighting` may re-run the drawing pass and
   the index could shift. I have not traced whether `RedrawRoomLighting` rebuilds hot spots.
   A port should probably key the transporter handshake on `(roomNum, objectNum)` instead.
2. **What is the authoritative meaning of `roomType.floor` and `roomType.suite`?** They appear
   in the observed data (floor values -2, -1, 0, 2, 3, 6, 7, 8; suite values like 51) and are
   used for map-mode room placement, but I found no collision consumer. Confirm they are
   purely navigational.
3. **`kRoof` tile code 4.** Observed in real roof rooms (CD Demo House r138 index 5, r148
   indices 0 and 2, r149 indices 1, 4, 7) yet `CheckRoofCollision` has no case for it and its
   `default:` kills the glider. Either the art at code 4 is a solid chimney the player is
   expected to avoid, or those rooms have a `kRoofLimit`-guarding object. Worth checking
   against the `'PICT'` tile art.
4. **Is `kSlider` ever a live hot spot?** `HandleRewards` has a bare `case kSlider: break;`
   (`GliderPRO/Sources/Interactions.c:978`) and `SetObjectState` falls through it with
   `changed` uninitialized (`GliderPRO/Sources/Objects.c:475`). No shipped house I sampled has
   a `kSlider` (0x2F). Confirm by exhaustive scan before implementing.
5. **`Sampler.binhex`'s 2 extra bytes.** `866 + 348 * 2 = 1562` but the data fork is 1564. The
   trailing bytes are `... 00 20 00 00 01 01 01 01`. Is this a truncated third room record, or
   editor slack? A Go loader that sizes by `nRooms` is safe either way, but it would be good to
   know whether other third-party houses do this systematically.
6. **`Coordinates.c` really is editor-only.** It is entirely wrapped in
   `#ifndef COMPILEDEMO` and implements the "Coordinates" windoid that shows the mouse position
   while editing a room. Nothing in it is referenced from the gameplay collision path. It was
   in the assigned file list; I read all 196 lines and confirmed it contributes nothing to
   interactions. Flagging in case the assignment intended a different file.
7. **`kChimeIt` (action 25).** I found the action constant and its `HandleHotSpotCollision`
   case, but no `AddActiveRect` call site emits it. Is it dead, or emitted from a path I have
   not traced (`ObjectDraw2.c`?).
8. **Two-player stairs codes.** `kPlayerEscapingUpStairs` (-8) is set by `kMoveItUp`
   (`GliderPRO/Sources/Interactions.c:1266`) but `kMoveItUp` tests for
   `kPlayerEscapedUpStairs` (-6) at `:1271`. The transition -8 -> -6 must happen in
   `Player.c`'s stairs mover; I did not locate the exact line.
9. **Exact frame at which `masterObjects[].dynaNum` becomes valid.** It is filled during the
   drawing pass (`GliderPRO/Sources/ObjectDrawAll.c:952`-`:960`), so it is -1 until
   `DrawLocale` completes. `SpillGrease(dynaNum, hotNum)` on the very first frame of a room
   would index `grease[-1]`. Is `DrawLocale` guaranteed to have run before the first
   `HandleInteraction`?

## Porting notes

### Toolbox replacements

| Mac Toolbox thing | What it does here | Go replacement |
|---|---|---|
| `Rect` (`{short top, left, bottom, right}`) | every collision rect | a struct with **int16-sized semantics**; do **not** use `image.Rectangle` for collision because its `Overlaps` treats touching as a miss (trap #1 in section 21) |
| `Point` (`{short v, h}`) | object `topLeft` on disk | note **v comes first** on disk (verified, section 20.2) |
| `SectRect` | used only in the **drawing** pass to cull off-screen objects (`GliderPRO/Sources/ObjectDrawAll.c` throughout) | `image.Rectangle.Overlaps` is correct **here** — same exclusive semantics |
| `QSetRect`, `QOffsetRect`, `HOffsetRect`, `VOffsetRect`, `InsetRect`, `ZeroRectCorner`, `CenterRectInRect` | rect arithmetic (`GliderPRO/Sources/RectUtils.c`) | trivial helpers; `QOffsetRect` is the non-Toolbox fast version and behaves identically |
| `GWorldPtr` / `CopyBits` / `CopyMask` / `PaintRect` | 8-bit indexed offscreen buffers | any 2D renderer; the **collision code never reads pixels**, so the port's collision layer needs no graphics at all — with two exceptions: grease creates its hot rect from screen-space geometry (section 19.4), and `dinahs[].dest` for appliances is screen-space (section 17.2) |
| `GetResource('bnds', id)` | 4-byte room-openness fallback | extract to a table at house-conversion time (section 20.5) |
| `GetPicture` / `PICT` | tile and object art | irrelevant to collision except that tile **codes** (not art) drive `kDirt`/`kRoof` |
| `HGetState`/`HLock`/`HSetState` | Memory Manager handle locking around `thisHouse` accesses | delete; Go slices do not move |
| `PlayPrioritySound(id, priority)` | priority-based sound cutting | any mixer; the **priority** argument matters for fidelity (a higher-priority sound preempts) |
| `GetKeys` / `BitTst` | KeyMap polling in `GetInput` | poll-based input; **not** event-based, because the physics is a fixed step |
| `TickCount()` | 60.15 Hz tick counter | `kTicksPerFrame` = 2, so the game runs at **30.075 fps**; use a fixed 1/30.075 s step or a tick-accumulator |
| big-endian on-disk data | every house field | `encoding/binary.BigEndian` |
| 68k/PPC `short` = int16 | all velocities and coordinates | use `int16` **or** `int` with explicit wrapping awareness; nothing in the collision path overflows int16 in practice except an unbounded falling `vVel`, which the original also would have wrapped after ~10900 frames |

### Ordering contract a Go port must honour

```
for each frame:
    gameFrame++                                  // Play.c:434
    evenFrame = !evenFrame                       // Play.c:435
    HandleTelephone()                            // Play.c:445
    HandleDynamics()                             // Play.c:449   -- moves dinahs, tests glider
    GetInput()                                   // Play.c:452   -- sets hDesiredVel, tipped, fires bands
    HandleInteraction()                          // Play.c:454
        CheckForHotSpots()                       //   sets ignore*/sliding, fires switches, awards prizes
        CheckGliderInRoom(each live glider)      //   reads ignore*, clamps to room, triggers transitions
    HandleTriggers()                             // Play.c:456   -- countdown, FireTrigger
    HandleBands()                                // Play.c:457   -- moves bands, CheckBandCollision
    HandleGlider()                               // Play.c:460   -- MoveGlider* per mode, then clears ignore*
    RenderFrame()                                // Play.c:469
        HandleGrease()                           //   Render.c:647 -- creates/grows the kSlideIt rect
    HandleDynamicScoreboard()                    // Play.c:470
```

Consequences of this order that are easy to get wrong:

* **Hot-spot actions run before movement.** A prize is awarded from the *previous* frame's
  glider position.
* **`ignore*` flags are set and consumed within `HandleInteraction`, then cleared by
  `HandleGlider`.** Clearing them anywhere before `CheckGliderInRoom` breaks doors and
  manholes.
* **Grease hot rects appear one frame late**, because `HandleGrease` runs after
  `HandleInteraction`.
* **Bands move after the glider's hot-spot pass but before the glider moves**, so a band fired
  this frame can already hit a switch this frame.
* **`HandleDynamics` runs before `GetInput`**, so an enemy's `hDesiredVel = ±8` is overwritten
  by keyboard input in the same frame if the player is holding a direction. The fans, which run
  in `CheckForHotSpots` *after* `GetInput`, use `+=` and therefore stack with input instead.

### Data-structure recommendations

1. Keep the **hot-spot list rebuild** exactly as-is: clear to zero length, then iterate the
   central room's 24 object slots in order 0..23, appending 0..3 rects each. Hot-spot **index
   is protocol** for the two-player transporter handshake, so the order must match.
2. Store `hotNum` as the index of the **last** rect an object appended, matching
   `GetHotRect`'s return value. Toggling by object touches only that one rect (section 11.4).
3. Keep `masterObjects[]` at capacity 216 = 24 * 9 and preserve `localLink`, `roomLink`,
   `objectLink`, `hotNum`, `dynaNum`, `objectNum` as **separate** fields with -1 sentinels.
   `dynaNum` is overloaded: it indexes `dinahs[]` for dynamic objects and `grease[]` for grease
   jars.
4. Do **not** unify `dinahs[].dest` coordinate spaces unless you accept the 22.8 divergence.
5. Model `batteryTotal` as a single signed counter (positive = battery, negative = helium), not
   two fields — the replace-vs-add semantics of section 10.3 depend on it.
6. `vVel` must be unclamped. `hVel` must be clamped to ±16 **after** the impulse ramp and
   **before** `wasHVel` is recorded.
7. Represent room openness as the four resolved values `leftThresh`, `rightThresh`,
   `topOpen`, `bottomOpen`, computed once per room entry. The escape functions test
   `leftThresh == 12` as a boolean, so keep the threshold as the source of truth rather than
   deriving it from `leftOpen` (which is inconsistent for `kDirt`, section 13.2).
8. Implement one inclusive-overlap helper and use it everywhere; implement one exclusive
   (containment) helper for `GliderInRect`. Never mix them.

### Testing strategy

* Golden-test the five rect-derivation paths (`GetObjectRect`, `GetHotRect`, the grease seed,
  the manhole transform, the `srcRects` table) against the observed byte tables in section 20 —
  those are exact and derived from shipped content.
* Golden-test `DetermineRoomOpenings` against the Art Museum `bounds` table (section 20.4) and
  the `kDirt`/`kRoof` tile tables (sections 20.6, 20.7).
* Property-test the overlap primitive: for every pair of rects with a shared edge, the
  inclusive helper must return true and the exclusive helper false.
* Regression-test the floor-contact jitter described in section 16.2(a): a glider resting on a
  floor at `dest.bottom == 312` must oscillate with a 2 px amplitude, not settle. If your port
  settles, you have replaced the "assign the velocity" idiom with a position clamp and the game
  will feel wrong.






