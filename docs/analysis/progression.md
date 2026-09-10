# Glider PRO 1.0.4 — Game Progression: Rooms, Transitions, Win/Lose Flow

## Scope

This document specifies, at implementation level, everything Glider PRO 1.0.4 does
between "the player picks New Game" and "the game-over screen is dismissed":

- The play-session lifecycle (`NewGame` → `PlayGame` → teardown).
- Where the glider starts, and how the house designates its start room / start point.
- Room addressing: the `(suite, floor)` grid, the 3×3 neighbourhood, room lookup.
- The exact mechanics of every room transition:
  - side exits (left/right walls, doors, windows),
  - up/down exits (ceiling/floor openings, dirt tunnels),
  - stairs (the "trip up/down stairs" mechanism),
  - transporters, mailboxes, and ducts (the "link" mechanism),
- What state persists and what resets when the room changes and when the player dies.
- Where per-room state (visited flag, lit lights, opened doors, taken bonuses) lives
  and how it is saved and restored.
- The "Glider PRO" end-of-house (win) condition and the win animation.
- The lose (out of gliders) condition and the "GAME OVER" paper animation.
- The saved-game data structures, their exact on-disk layout, and the fact that the
  save/resume path is entirely dead code in 1.0.4.
- The precise death/respawn rule set, including the two-player limbo/follow protocol.

Not in scope (covered by sibling documents): physics/velocity integration in
`Dynamics.c`/`Player.c` `MoveGliderNormal`, rendering/blitting internals,
object drawing, sound tables, the house editor, the map window, preferences.

Line citations of the form `GliderPRO/Sources/Play.c:74` refer to line numbers in the
CR→LF-converted copies of the shipped sources. The originals are classic Mac text
(bare `\r` line endings); the conversion is 1:1 on lines, so the numbers are the
numbers you get from any LF-based editor after conversion. Nothing under
`GliderPRO/` was modified; the conversions live in `/tmp/wf-prog/`.

All empirically observed byte values in this document were produced by parsing the
22 shipped houses in `GliderPRO/Houses/*.binhex` with purpose-written Python
(`/tmp/wf-prog/binhex_dec.py`, `parse_house.py`, `verify_links.py`, `survey.py`,
`audit_bad.py`). Where this document states a fact about on-disk layout, that fact
was checked against all 22 files unless stated otherwise.

## Sources read

Read in full (the assigned set):

| File | Lines | Role in progression |
|---|---|---|
| `GliderPRO/Sources/Play.c` | 821 | Session lifecycle, main loop, object-state reset, telephone |
| `GliderPRO/Sources/Trip.c` | 245 | *Not* room trips — object toggle/trigger action table |
| `GliderPRO/Sources/Transit.c` | 558 | All room-to-room movement, room visitation scoring, 2P limbo |
| `GliderPRO/Sources/GameOver.c` | 507 | Win animation, lose animation, `FlagGameOver` |
| `GliderPRO/Sources/Room.c` | 1206 | Room load/store, neighbour lookup, openings, wall/floor/ceiling predicates |
| `GliderPRO/Sources/DynamicMaps.c` | 798 | `savedMaps[]` background swatches (the "object disappears" mechanism), shreds |
| `GliderPRO/Sources/Link.c` | 396 | `MergeFloorSuite`/`ExtractFloorSuite` link codec, editor link/unlink |

Read in full or in the relevant part, following references:

| File | What was taken from it |
|---|---|
| `GliderPRO/Sources/Player.c` | 1605 lines: glider FSM dispatch, stairs, ducts, mail, shredder, `OffsetGlider`, `OffAMortal` |
| `GliderPRO/Sources/Modes.c` | 640 lines: every `StartGlider*`/`Flag*`/`Ready*` mode-entry routine |
| `GliderPRO/Sources/Interactions.c` | Escape detection (`CheckEscape*`, `CheckGliderInRoom`, `CheckRoofCollision`), `HandleRewards` (star → win), `HandleSwitches` |
| `GliderPRO/Sources/House.c` | `InitializeEmptyHouse`, `RealRoomNumberCount`, `GetFirstRoomNumber`, `WhereDoesGliderBegin`, `CountRoomsVisited`, `ConvertHouseVer1To2`, `ShiftWholeHouse` |
| `GliderPRO/Sources/HouseIO.c` | `ReadHouse`, `WriteHouse` — the on-disk format and the flag/timestamp decode |
| `GliderPRO/Sources/SavedGames.c` | 352 lines: `gameType`/`game2Type` handling; proof the resume path is dead |
| `GliderPRO/Sources/RoomGraphics.c` | `DrawLocale`, `ReadyLevel`, `RedrawRoomLighting`, `localNumbers[]` population |
| `GliderPRO/Sources/Objects.c` | `SetObjectState`, `GetObjectState`, `ListAllLocalObjects`, `GetRoomLinked`, `GetObjectLinked` |
| `GliderPRO/Sources/ObjectRects.c` | `CreateActiveRects` (the `who != 255` transport guard, all hot-spot rects), `GetUpStairsRightEdge`, `GetDownStairsLeftEdge` |
| `GliderPRO/Sources/ObjectDrawAll.c` | Which object types consume a `savedMaps` slot; the `isLit` draw-suppression pattern |
| `GliderPRO/Sources/Grease.c` | `AddGrease` (the 11th `savedMaps` consumer) |
| `GliderPRO/Sources/Render.c` | `RenderFrame`'s 2-tick busy-wait, `InitGarbageRects` |
| `GliderPRO/Sources/StructuresInit2.c`, `Environ.c` | `savedMaps[]` allocation and memory budget |
| `GliderPRO/Sources/Banner.c` | `CountStarsInHouse`, `BringUpBanner`, `DisplayStarsRemaining` |
| `GliderPRO/Sources/Transitions.c` | `WipeScreenOn`, `PourScreenOn`, `DumpScreenOn` |
| `GliderPRO/Sources/Scoreboard.c` | `RefreshScoreboard`, `AdjustScoreboardHeight`, room-title modes |
| `GliderPRO/Sources/Triggers.c` | `ArmTrigger`, `HandleTriggers`, `FireTrigger` |
| `GliderPRO/Sources/Input.c` | Cmd-Q / Cmd-S / Delete-key suicide |
| `GliderPRO/Sources/HighScores.c` | `TestHighScore` (the post-game-over hook) |
| `GliderPRO/Sources/Menu.c` | `DoGameMenu` (`iNewGame`/`iTwoPlayer`/`iOpenSavedGame`) |
| `GliderPRO/Sources/InterfaceInit.c` | `playOriginH/V`, `localRoomsDest[9]` geometry |
| `GliderPRO/Sources/Main.c`, `Settings.c` | `numNeighbors ∈ {1, 3, 9}` |
| `GliderPRO/Headers/GliderStructs.h` | Every struct layout quoted below |
| `GliderPRO/Headers/GliderDefines.h` | Every constant quoted below |

Empirical inputs: all 22 files in `GliderPRO/Houses/*.binhex`.

---

## 1. Vocabulary, coordinate systems, and global state

### 1.1 The three levels of "where"

Glider PRO has three nested notions of location, and the port must keep them
distinct because the code does:

1. **Room index** — `short`, an index into the flat `houseType.rooms[]` array. The
   authoritative runtime value is the global `thisRoomNumber`. `-1` means "no room".
2. **Room address** — the pair `(suite, floor)`. `suite` is the horizontal
   coordinate, `floor` the vertical. This is what the house *author* sees and what
   links are encoded in. There is **no index** from address to index: every lookup
   is a linear scan (`GliderPRO/Sources/Room.c:737`).
3. **Pixel position inside a room** — a QuickDraw `Rect` in room-local coordinates,
   origin at the room's top-left, `512 × 322` pixels.

### 1.2 Room pixel geometry

```
#define kNumTiles           8
#define kTileWide           64
#define kTileHigh           322
#define kRoomWide           512         // kNumTiles * kTileWide
#define kFloorSupportTall   44
#define kVertLocalOffset    322         // kTileHigh - 39 (was 283, then 295)
```
(`GliderPRO/Headers/GliderDefines.h:496-501`)

A room is exactly 8 tiles of 64px = 512px wide and 322px tall. The vertical stride
between vertically adjacent rooms on screen is `kVertLocalOffset` = 322, i.e. the
same as `kTileHigh`, so rooms tile the plane without overlap.

Collision/exit thresholds (`GliderPRO/Headers/GliderDefines.h:503-511`):

| Constant | Value | Meaning |
|---|---:|---|
| `kCeilingLimit` | 8 | glider `dest.top` above this ⇒ test ceiling |
| `kFloorLimit` | 312 | glider `dest.bottom` below this ⇒ test floor |
| `kRoofLimit` | 122 | on `kRoof` background, below this ⇒ roof slope test |
| `kLeftWallLimit` | 12 | closed-wall bounce plane on the left |
| `kNoLeftWallLimit` | -24 | open-wall exit plane on the left (`0 - kGliderWide/2`) |
| `kRightWallLimit` | 500 | closed-wall bounce plane on the right |
| `kNoRightWallLimit` | 536 | open-wall exit plane (`kRoomWide + kGliderWide/2`) |
| `kNoCeilingLimit` | -10 | open-ceiling exit plane |
| `kNoFloorLimit` | 332 | open-floor exit plane |

Glider geometry (`GliderPRO/Headers/GliderDefines.h:548-555`):

| Constant | Value |
|---|---:|
| `kGliderWide` | 48 |
| `kGliderHigh` | 20 |
| `kHalfGliderWide` | 24 |
| `kGliderBurningHigh` | 26 |
| `kShadowHigh` | 9 |
| `kShadowTop` | 306 |
| `kFaceRight` | `TRUE` (1) |
| `kFaceLeft` | `FALSE` (0) |
| `kPlayer1` | `TRUE` (1) |
| `kPlayer2` | `FALSE` (0) |
| `kGliderStartsDown` | 32 |

### 1.3 The 3×3 neighbourhood and `numNeighbors`

Glider PRO can draw the current room alone, the current room plus its left and right
neighbours, or the full 3×3 block. The selector is the global `numNeighbors`, which
takes exactly one of three values: 1, 3, or 9
(`GliderPRO/Sources/Settings.c:1153`, `:1164`, `:1176`; default 9 at
`GliderPRO/Sources/Main.c:154` and `GliderPRO/Sources/Settings.c:1257`). It is forced
to 1 on small screens: `if ((numNeighbors > 1) && (thisMac.screen.right <= 512)) numNeighbors = 1;`
(`GliderPRO/Sources/Main.c:191-192`).

Neighbour slot indices (`GliderPRO/Headers/GliderDefines.h:217-225`):

| Constant | Value | `(hDelta, vDelta)` applied to `(suite, floor)` |
|---|---:|---|
| `kCentralRoom` | 0 | (0, 0) |
| `kNorthRoom` | 1 | (0, +1) |
| `kNorthEastRoom` | 2 | (+1, +1) |
| `kEastRoom` | 3 | (+1, 0) |
| `kSouthEastRoom` | 4 | (+1, -1) |
| `kSouthRoom` | 5 | (0, -1) |
| `kSouthWestRoom` | 6 | (-1, -1) |
| `kWestRoom` | 7 | (-1, 0) |
| `kNorthWestRoom` | 8 | (-1, +1) |

Note the sign convention: **north = floor + 1**. Higher `floor` is physically higher.
Deltas are from `GliderPRO/Sources/Room.c:569-613`.

Screen placement of the nine slots (`GliderPRO/Sources/InterfaceInit.c:203-218`):

```
playOriginH = (RectWide(&thisMac.screen) - kRoomWide) / 2;    // (screenW - 512)/2
playOriginV = (RectTall(&thisMac.screen) - kTileHigh) / 2;    // (screenH - 322)/2
for (i = 0; i < 9; i++) {
    QSetRect(&localRoomsDest[i], 0, 0, kRoomWide, kTileHigh);
    QOffsetRect(&localRoomsDest[i], playOriginH, playOriginV); }
QOffsetRect(&localRoomsDest[kNorthRoom],      0,          -kVertLocalOffset);
QOffsetRect(&localRoomsDest[kNorthEastRoom],  kRoomWide,  -kVertLocalOffset);
QOffsetRect(&localRoomsDest[kEastRoom],       kRoomWide,   0);
QOffsetRect(&localRoomsDest[kSouthEastRoom],  kRoomWide,   kVertLocalOffset);
QOffsetRect(&localRoomsDest[kSouthRoom],      0,           kVertLocalOffset);
QOffsetRect(&localRoomsDest[kSouthWestRoom], -kRoomWide,   kVertLocalOffset);
QOffsetRect(&localRoomsDest[kWestRoom],      -kRoomWide,   0);
QOffsetRect(&localRoomsDest[kNorthWestRoom], -kRoomWide,  -kVertLocalOffset);
```

So `+kVertLocalOffset` in *screen* Y is the *south* room: screen Y grows downward
while `floor` grows upward.

`localNumbers[9]` holds the resolved room index for each slot; it is rebuilt at the
top of every `DrawLocale()`:

```
for (i = 0; i < 9; i++) {
    localNumbers[i] = GetNeighborRoomNumber(i);
    isStructure[i] = IsRoomAStructure(localNumbers[i]); }
```
(`GliderPRO/Sources/RoomGraphics.c:68-71`). Slots with no room hold `kRoomIsEmpty` = -1
(`GliderPRO/Headers/GliderDefines.h:525`).

**`localNumbers[kCentralRoom]` is the canonical "which house room am I in" for
writes** — `HandleRoomVisitation` uses it rather than `thisRoomNumber`
(`GliderPRO/Sources/Transit.c:440`). In practice the two are always equal, because
`GetNeighborRoomNumber(kCentralRoom)` searches for the room at delta (0,0) from
`thisRoomNumber`'s own address. They can differ only if two rooms share a
`(suite, floor)` address, in which case the linear scan returns the lower index.
(See §3.5 and the open question in §20.)

### 1.4 Global progression state

These globals are the whole of the progression state machine. Types are from the
`extern` declarations in the cited files.

| Global | Type | Defined | Meaning |
|---|---|---|---|
| `thisHouse` | `houseHand` (`houseType**`) | `House.c` | The whole house, as a relocatable block |
| `thisRoom` | `roomPtr` | `House.c` | Pointer to a **copy** of the current room |
| `thisRoomNumber` | `short` | `House.c` | Index of current room, `-1` = none |
| `previousRoom` | `short` | `House.c` | Set by `ForceThisRoom` to the outgoing index |
| `numberRooms` | `short` | `House.c` | Cached `(*thisHouse)->nRooms` |
| `noRoomAtAll` | `Boolean` | `House.c` | `RealRoomNumberCount() == 0` |
| `localNumbers[9]` | `short[9]` | `RoomGraphics.c:32` | Resolved 3×3 neighbourhood |
| `thisBackground` | `short` | `RoomGraphics.c:32` | Current room's background PICT ID |
| `thisTiles[8]` | `short[8]` | `RoomGraphics.c:31` | Current room's tile indices |
| `numNeighbors` | `short` | `RoomGraphics.c:31` | 1, 3, or 9 |
| `leftOpen`, `rightOpen`, `topOpen`, `bottomOpen` | `Boolean` | `Room.c` | Wall openness, recomputed per room |
| `leftThresh`, `rightThresh` | `short` | `Room.c` | Wall X planes, recomputed per room |
| `theGlider`, `theGlider2` | `gliderType` | `Player.c` | The two gliders |
| `mortals` | `short` | `Player.c` | Lives remaining (shared across both players) |
| `theScore` | `long` | `Player.c` | Score |
| `numStarsRemaining` | `short` | `Interactions.c` | Win counter |
| `gameOver` | `Boolean` | `GameOver.c:45` | Set by `FlagGameOver` |
| `countDown` | `short` | `GameOver.c:44` | Frames until the game-over screen |
| `playing` | `Boolean` | `Play.c` | Main-loop condition |
| `otherPlayerEscaped` | `short` | `Interactions.c` | Two-player handshake code |
| `onePlayerLeft` | `Boolean` | `Play.c` | One of the two players is permanently dead |
| `playerDead` | `Boolean` | `Play.c` | `which` of the dead player |
| `playerSuicide` | `Boolean` | `Play.c` | Delete-key suicide in progress |
| `firstPlayer` | `Boolean` | `Transit.c:18` | `which` of the player who escaped first |
| `saidFollow` | `short` | `Modes.c:13` | How many times the "follow me" sound played |
| `takingTheStairs` | `Boolean` | `Transit.c:18` | Selects the stairs entry animation |
| `linkedToWhat` | `short` | `Transit.c:17` | `kLinkedTo*` classification of the pending transit |
| `transRoom` | `short` | `Interactions.c` | Destination room index of the pending transit |
| `transRect` | `Rect` | `Interactions.c` | Destination object's rect |
| `gameFrame` | `long` | `Play.c` | Frame counter, incremented once per loop |
| `evenFrame` | `Boolean` | `Play.c` | Toggled every frame |
| `numSavedMaps` | `short` | `DynamicMaps.c:32` | Count of live background swatches |

---

## 2. On-disk data model

### 2.1 The house file *is* the `houseType` struct image

`ReadHouse()` does not parse anything. It gets the data-fork length, allocates one
handle of exactly that size, and does a single `FSRead` into it:

```
theErr = GetEOF(houseRefNum, &byteCount);
...
thisHouse = (houseHand)NewHandle(byteCount);
...
theErr = SetFPos(houseRefNum, fsFromStart, 0L);
...
theErr = FSRead(houseRefNum, &byteCount, *thisHouse);
```
(`GliderPRO/Sources/HouseIO.c:341-372`)

`WriteHouse()` is the mirror image, and the write length is simply the handle size:

```
byteCount = GetHandleSize((Handle)thisHouse);
...
theErr = FSWrite(houseRefNum, &byteCount, *thisHouse);
...
theErr = SetEOF(houseRefNum, byteCount);
```
(`GliderPRO/Sources/HouseIO.c:473`, `:491`, `:499`; `WriteHouse` spans `:448-519`)

Consequences a Go port must honour:

- The file is **big-endian**, with the exact field order and the exact 68k/PPC
  struct padding of the original. Every field in every struct in the chain is
  naturally aligned at its own size or better, and every struct's size is even, so
  **there is no padding anywhere** — the layout is the packed layout. This was
  verified empirically (§2.6).
- The file length is `866 + 348 * nRooms` **or larger**. It must not be validated for
  exact equality: one shipped house (`Sampler`) is 2 bytes longer (§2.6).
- `nRooms` at file offset 864 is the authority for the room count; there is no
  count field for objects-per-room (always 24 slots) and no per-room length field.
- Everything the game mutates during play that is not glider-local lives in this
  same image. There is no separate "save state" during a session.

The resource fork holds the house's custom `PICT` backgrounds, `'bnds'` bounding
codes, `'Date'`, and sounds; the QuickTime movie is a *sibling file* named
`<housename>.mov`. None of that participates in progression except `'bnds'`
(§4.3). File type is `'gliH'`, creator `'ozm5'` — verified for all 22 shipped houses
(§2.6).

### 2.2 `houseType` — 866 bytes + rooms

```c
typedef struct
{
    short       version;                    // 2
    short       unusedShort;                // 2
    long        timeStamp;                  // 4
    long        flags;                      // 4 (bit 0 = wardBit)
    Point       initial;                    // 4
    Str255      banner;                     // 256
    Str255      trailer;                    // 256
    scoresType  highScores;                 // 292
    gameType    savedGame;                  // 40
    Boolean     hasGame;                    // 1
    Boolean     unusedBoolean;              // 1
    short       firstRoom;                  // 2
    short       nRooms;                     // 2
    roomType    rooms[];                    // 348 * nRooms
} houseType, *housePtr, **houseHand;        // total = 866 +
```
(`GliderPRO/Headers/GliderStructs.h:182-198`)

Computed offsets (all verified by parsing, §2.6):

| Offset (dec) | Offset (hex) | Size | Field | Type |
|---:|---|---:|---|---|
| 0 | 0x000 | 2 | `version` | `short`, big-endian |
| 2 | 0x002 | 2 | `unusedShort` | `short` — **garbage on disk**, see §2.6 |
| 4 | 0x004 | 4 | `timeStamp` | `long`; **bit 0 is the lock bit** |
| 8 | 0x008 | 4 | `flags` | `long`; bit 0 ward, bit 1 phone, bit 2 star-count-off |
| 12 | 0x00C | 4 | `initial` | `Point` = `{short v; short h;}` — **v first!** |
| 16 | 0x010 | 256 | `banner` | `Str255`: length byte + 255 bytes MacRoman |
| 272 | 0x110 | 256 | `trailer` | `Str255` |
| 528 | 0x210 | 292 | `highScores` | `scoresType` |
| 820 | 0x334 | 40 | `savedGame` | `gameType` |
| 860 | 0x35C | 1 | `hasGame` | `Boolean` |
| 861 | 0x35D | 1 | `unusedBoolean` | `Boolean` |
| 862 | 0x35E | 2 | `firstRoom` | `short` — **room index, not address** |
| 864 | 0x360 | 2 | `nRooms` | `short` |
| 866 | 0x362 | 348·n | `rooms[]` | `roomType[]` |

`Point` in classic Mac OS is `struct { short v; short h; }` — **vertical first**.
Getting this backwards silently mirrors the start position of every house. The
`Sampler` house's `initial` bytes at 0x00C are `00 55 01 80` = `v=85, h=384`,
and `h=384` is inside a 512-wide room while `v=85` is inside a 322-tall room, so
`v`-first is confirmed by the data as well as by the header.

`kHouseVersion` = `0x0200`, `kNewHouseVersion` = `0x0300`
(`GliderPRO/Headers/GliderDefines.h:517-518`). `ReadHouse` rejects anything
`>= 0x0300`:

```
wasHouseVersion = (*thisHouse)->version;
if (wasHouseVersion >= kNewHouseVersion) { YellowAlert(kYellowNewerVersion, 0); ... return(false); }
```
(`GliderPRO/Sources/HouseIO.c:394-400`)

Flag and lock decode (`GliderPRO/Sources/HouseIO.c:402`, `:416-418` — the lock bit is
read several lines before the three flag bits, with the `firstRoom` fetch in between):

```
houseUnlocked      = (((*thisHouse)->timeStamp & 0x00000001) == 0);
wardBitSet         = (((*thisHouse)->flags & 0x00000001) == 0x00000001);
phoneBitSet        = (((*thisHouse)->flags & 0x00000002) == 0x00000002);
bannerStarCountOn  = (((*thisHouse)->flags & 0x00000004) == 0x00000000);
```

Note `bannerStarCountOn` is **inverted**: bit 2 *set* means "do not show the star
count in the banner". `phoneBitSet` *suppresses* the telephone (§6.7). The write side
forces the lock bit and clears the sign bit:

```
if (fileDirty)                              // <== the whole block is guarded
{
    GetDateTime(&timeStamp);
    timeStamp &= 0x7FFFFFFF;
    if (changeLockStateOfHouse) houseUnlocked = !saveHouseLocked;
    if (houseUnlocked) timeStamp &= 0x7FFFFFFE; else timeStamp |= 0x00000001;
    (*thisHouse)->timeStamp = (long)timeStamp;
    (*thisHouse)->version = wasHouseVersion;
}
```
(`GliderPRO/Sources/HouseIO.c:475-489`; the actual `FSWrite` of the whole handle is at
`:491`, and `WriteHouse` calls `CopyThisRoomToRoom()` at `:467` first — that is what
flushes the `thisRoom` working copy back into `rooms[thisRoomNumber]`.)

So the house timestamp is a Mac epoch seconds value with its LSB stolen for the lock
flag and its MSB cleared. Note also that `WriteHouse` writes back `wasHouseVersion`,
not `kHouseVersion` — a v1 house saved without an explicit conversion stays v1.

### 2.3 `roomType` — 348 bytes

```c
typedef struct
{
    Str27       name;                       // 28
    short       bounds;                     // 2
    Byte        leftStart;                  // 1
    Byte        rightStart;                 // 1
    Byte        unusedByte;                 // 1
    Boolean     visited;                    // 1
    short       background;                 // 2
    short       tiles[kNumTiles];           // 2 * 8
    short       floor, suite;               // 2 + 2
    short       openings;                   // 2
    short       numObjects;                 // 2
    objectType  objects[kMaxRoomObs];       // 24 * 12
} roomType, *roomPtr;                       // total = 348
```
(`GliderPRO/Headers/GliderStructs.h:166-180`)

Offsets **relative to the room's base** (`866 + 348*roomIndex`):

| Rel. offset | Size | Field | Notes |
|---:|---:|---|---|
| 0 | 28 | `name` | `Str27`: length byte + 27 bytes MacRoman |
| 28 | 2 | `bounds` | Packed bounding code; 0 = "use the `'bnds'` resource" |
| 30 | 1 | `leftStart` | Entry Y offset when entering from the left |
| 31 | 1 | `rightStart` | Entry Y offset when entering from the right |
| 32 | 1 | `unusedByte` | |
| 33 | 1 | `visited` | **The per-room progression flag; persisted to disk** |
| 34 | 2 | `background` | PICT ID; `>= 3000` = custom |
| 36 | 16 | `tiles[8]` | Tile index per 64px column |
| 52 | 2 | `floor` | Vertical address, signed |
| 54 | 2 | `suite` | Horizontal address, signed; `-1` = deleted room |
| 56 | 2 | `openings` | **Runtime-only; 0 on disk in every shipped house** |
| 58 | 2 | `numObjects` | Count of non-empty object slots |
| 60 | 288 | `objects[24]` | 24 × 12 bytes |

`suite == kRoomIsEmpty` (-1) is the tombstone for a deleted room:
`DeleteRoom` sets `thisRoom->suite = kRoomIsEmpty;` and
`(*thisHouse)->rooms[thisRoomNumber].suite = kRoomIsEmpty;`
(`GliderPRO/Sources/Room.c:463-464`) and never shrinks the array. New rooms reuse
the first tombstone if one exists, else `PtrAndHand` appends `sizeof(roomType)` and
increments `nRooms` (`GliderPRO/Sources/Room.c:194-215`). `RealRoomNumberCount()`
is therefore `nRooms` minus the tombstone count (`GliderPRO/Sources/House.c:170`).

`leftStart`/`rightStart` are `Byte` (unsigned 0..255) and are cast to `short` before
use: `kGliderStartsDown + (short)thisRoom->leftStart - 2`
(`GliderPRO/Sources/Transit.c:175-176`). With `kGliderStartsDown` = 32, the entry Y
of a side transition is `32 + leftStart - 2` = `30 + leftStart`.

### 2.4 `objectType` — 12 bytes, tagged union

```c
typedef struct
{
    short   what;                           // 2
    union { blowerType a; furnitureType b; bonusType c; transportType d;
            switchType e; lightType f; applianceType g; enemyType h;
            clutterType i; } data;          // 10
} objectType, *objectPtr;                   // total = 12
```
(`GliderPRO/Headers/GliderStructs.h:90-105`)

`what == kObjectIsEmpty` = -1 (`GliderPRO/Headers/GliderDefines.h:526`) marks an
unused slot. Every union arm is exactly 10 bytes, so the object record is exactly 12
bytes with no padding.

The union arms relevant to progression:

```c
typedef struct { Point topLeft; short distance; Boolean initial; Boolean state;
                 Byte vector; Byte tall; } blowerType;      // a, total 10
typedef struct { Rect bounds; short pict; } furnitureType;  // b
typedef struct { Point topLeft; short length; short points;
                 Boolean state; Boolean initial; } bonusType;    // c
typedef struct { Point topLeft; short tall; short where;
                 Byte who; Byte wide; } transportType;      // d
typedef struct { Point topLeft; short delay; short where;
                 Byte who; Byte type; } switchType;         // e
typedef struct { Point topLeft; short length; Byte byte0; Byte byte1;
                 Boolean initial; Boolean state; } lightType;     // f
typedef struct { Point topLeft; short height; Byte byte0; Byte delay;
                 Boolean initial; Boolean state; } applianceType; // g
typedef struct { Point topLeft; short length; Byte delay; Byte byte0;
                 Boolean initial; Boolean state; } enemyType;     // h
typedef struct { Rect bounds; short pict; } clutterType;    // i
```
(`GliderPRO/Headers/GliderStructs.h:11-88`)

Byte offsets within the 12-byte object record, for the arms progression cares about:

| Rel. | `d` transport | `e` switch | `c` bonus | `f` light | `g` appliance | `h` enemy | `a` blower |
|---:|---|---|---|---|---|---|---|
| 0 | `what` | `what` | `what` | `what` | `what` | `what` | `what` |
| 2 | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` |
| 4 | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` |
| 6 | `tall` | `delay` | `length` | `length` | `height` | `length` | `distance` |
| 8 | **`where`** | **`where`** | `points` | `byte0`,`byte1` | `byte0`,`delay` | `delay`,`byte0` | `initial`,`state` |
| 10 | **`who`**, `wide` | **`who`**, `type` | `state`,`initial` | `initial`,`state` | `initial`,`state` | `initial`,`state` | `vector`,`tall` |

**The `state`/`initial` byte order differs between `bonusType` (state at +10,
initial at +11) and `lightType`/`applianceType`/`enemyType` (initial at +10, state
at +11), and `blowerType` has them at +8/+9.** These are separate structs sharing a
union, and `SetObjectsToDefaults()` reads/writes each through the correct arm
(§6.5), so a port must not unify them.

The critical fields for progression are `data.d.where` / `data.d.who` for transports
and `data.e.where` / `data.e.who` for switches and triggers: `where` is a
floor/suite combo (§5), `who` is a **1-byte object index** with sentinel 255.

### 2.5 `scoresType`, `gameType`, `savedRoom`, `game2Type`

```c
typedef struct
{
    Str31           banner;                 // 32       = 32
    Str15           names[kMaxScores];      // 16 * 10  = 160
    long            scores[kMaxScores];     // 4 * 10   = 40
    unsigned long   timeStamps[kMaxScores]; // 4 * 10   = 40
    short           levels[kMaxScores];     // 2 * 10   = 20
} scoresType;                               // total    = 292
```
(`GliderPRO/Headers/GliderStructs.h:107-114`; `kMaxScores` = 10,
`GliderPRO/Headers/GliderDefines.h:249`)

Offsets relative to the house's `highScores` at 528: `banner` 528, `names[i]`
560+16i, `scores[i]` 720+4i, `timeStamps[i]` 760+4i, `levels[i]` 800+2i.
`levels[i]` is *rooms visited*, not a level number (§13).

```c
typedef struct
{
    short       version;        // 2      offset  0  (house-relative 820)
    short       wasStarsLeft;   // 2               2             822
    long        timeStamp;      // 4               4             824
    Point       where;          // 4               8             828  (v then h)
    long        score;          // 4              12             832
    long        unusedLong;     // 4              16             836
    long        unusedLong2;    // 4              20             840
    short       energy;         // 2              24             844
    short       bands;          // 2              26             846
    short       roomNumber;     // 2              28             848
    short       gliderState;    // 2              30             850
    short       numGliders;     // 2              32             852
    short       foil;           // 2              34             854
    short       unusedShort;    // 2              36             856
    Boolean     facing;         // 1              38             858
    Boolean     showFoil;       // 1              39             859
} gameType;                     // total = 40
```
(`GliderPRO/Headers/GliderStructs.h:116-134`)

`gameType` is embedded in the house at offset 820. This is the **in-house saved
game**: a single savegame slot carried inside the house file itself, valid only when
`hasGame` (offset 860) is nonzero.

```c
typedef struct
{
    short       unusedShort;                // 2
    Byte        unusedByte;                 // 1
    Boolean     visited;                    // 1
    objectType  objects[kMaxRoomObs];       // 24 * 12
} savedRoom, *saveRoomPtr;                  // total = 292
```
(`GliderPRO/Headers/GliderStructs.h:136-142`)

`savedRoom` is precisely the subset of `roomType` that changes during play: the
`visited` flag and all 24 object records (whose `state` bytes carry lights, doors,
and taken bonuses). Note the layout is *not* a prefix of `roomType`: `roomType` has
`visited` at +33 and `objects` at +60, whereas `savedRoom` has `visited` at +3 and
`objects` at +4. It exists only for the *external* savegame file.

```c
typedef struct
{
    FSSpec      house;          // 70
    short       version;        // 2
    short       wasStarsLeft;   // 2
    long        timeStamp;      // 4
    Point       where;          // 4
    long        score;          // 4
    long        unusedLong;     // 4
    long        unusedLong2;    // 4
    short       energy;         // 2
    short       bands;          // 2
    short       roomNumber;     // 2
    short       gliderState;    // 2
    short       numGliders;     // 2
    short       foil;           // 2
    short       nRooms;         // 2
    Boolean     facing;         // 1
    Boolean     showFoil;       // 1
    savedRoom   savedData[];    // 4
} game2Type, *gamePtr;          // total = 114
```
(`GliderPRO/Headers/GliderStructs.h:144-163`)

`game2Type` is the *external* `'gliG'` savegame format: an `FSSpec` naming the house,
the same scalar fields as `gameType` plus `nRooms`, then `nRooms` × 292-byte
`savedRoom` records. The author's `// total = 114` comment is wrong: 70 + 2 + 2 + 4
+ 4 + 4 + 4 + 4 + 2·6 + 2 + 1 + 1 = 110, and the flexible array contributes 0. It
does not matter, because **nothing in 1.0.4 ever writes or reads this format**
(§15). Note `FSSpec` is `{ short vRefNum; long parID; Str63 name; }` = 2 + 4 + 64 =
70 bytes, and it embeds a volume reference number and directory ID — machine-local
values that a Go port must replace with a path or house name (see §15 and Porting
notes).
### 2.6 Empirical verification of the on-disk layout

Everything in §2.1–2.5 was checked by decoding all 22 shipped houses out of BinHex
and parsing the data forks at the offsets given above. The decoder and parsers are
`/tmp/wf-prog/binhex_dec.py`, `parse_house.py`, `verify_links.py`, `survey.py`,
`audit_bad.py`.

**Test 1 — file type, creator, and the `866 + 348·n` size law.**
All 22 files decode as type `gliH`, creator `ozm5`. Observed:

| House | `version` | `nRooms` | data fork | `866 + 348·n` | delta | `firstRoom` | resource fork |
|---|---|---:|---:|---:|---:|---:|---:|
| Art Museum | 0x0200 | 109 | 38798 | 38798 | 0 | 91 | 7476159 |
| CD Demo House | 0x0200 | 206 | 72554 | 72554 | 0 | 70 | 1612342 |
| California or Bust! | 0x0200 | 16 | 6434 | 6434 | 0 | 14 | 259542 |
| Castle o' the Air | 0x0200 | 85 | 30446 | 30446 | 0 | 33 | 306364 |
| Davis Station | 0x0200 | 65 | 23486 | 23486 | 0 | 4 | 1871320 |
| Demo House | 0x0200 | 45 | 16526 | 16526 | 0 | 0 | 491757 |
| Empty House | 0x0200 | 35 | 13046 | 13046 | 0 | 0 | 2670 |
| Fun House | 0x0200 | 43 | 15830 | 15830 | 0 | 29 | 662443 |
| Grand Prix | 0x0200 | 175 | 61766 | 61766 | 0 | 127 | 1347764 |
| ImagineHouse PRO II | 0x0200 | 279 | 97958 | 97958 | 0 | 1 | 677770 |
| In The Mirror | 0x0200 | 97 | 34622 | 34622 | 0 | 6 | 151870 |
| Land of Illusion | 0x0200 | 303 | 106310 | 106310 | 0 | 43 | 401793 |
| Leviathan | 0x0200 | 472 | 165122 | 165122 | 0 | 39 | 1901282 |
| Metropolis | 0x0200 | 127 | 45062 | 45062 | 0 | 8 | 946907 |
| Nemo's Market | 0x0200 | 124 | 44018 | 44018 | 0 | 0 | 590602 |
| Rainbow's End | 0x0200 | 223 | 78470 | 78470 | 0 | 30 | 316065 |
| **Sampler** | 0x0200 | 2 | **1564** | 1562 | **+2** | 1 | 286 |
| Slumberland | 0x0200 | 383 | 134150 | 134150 | 0 | 126 | 1041112 |
| SpacePods | 0x0200 | 402 | 140762 | 140762 | 0 | 259 | 636844 |
| Teddy World | 0x0200 | 531 | 185654 | 185654 | 0 | 0 | 3107348 |
| The Asylum Pro | 0x0200 | 140 | 49586 | 49586 | 0 | 20 | 494107 |
| Titanic | 0x0200 | 208 | 73250 | 73250 | 0 | 92 | 845727 |

21 of 22 are *exactly* `866 + 348·nRooms` bytes, which confirms every offset in
§2.2/§2.3/§2.4 simultaneously (any padding anywhere in the chain would break the
arithmetic). All 22 are `version == 0x0200`; **no v1 (`< 0x0200`) house ships with
1.0.4**, so the v1 link decode path (§5.2) is exercised only by third-party houses.

`Sampler` is 2 bytes *longer* than the struct image. The trailing bytes are observed as:

```
offset 1556: 00 20 00 00 01 01 01 01
                          ^^^^^ last 2 bytes of room 1 (objects[23] tail)
offset 1562: 01 01        <- 2 extra bytes past the end of the struct
```

This is explained by `WriteHouse` writing `GetHandleSize((Handle)thisHouse)`
(`GliderPRO/Sources/HouseIO.c:473`) rather than a computed struct size. Some editing
sequence left the handle 2 bytes larger than `866 + 348·2`, and the extra bytes were
written verbatim. A Go loader must therefore accept
`len(data) >= 866 + 348*nRooms` and ignore trailing slop. It must **not** derive
`nRooms` from the file length.

**Test 2 — header field survey across all 22 houses.**

| House | `timeStamp&1` (locked) | `flags` | ward b0 | phone b1 | starOff b2 | `hasGame` | rooms w/ `visited` | rooms w/ `openings != 0` | `numObjects` mismatches | suite range | floor range |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| Art Museum | 1 | 6 | 0 | 1 | **1** | 0 | 22 | 0 | 0 | 54..71 | -1..9 |
| CD Demo House | 1 | 2 | 0 | 1 | 0 | 0 | 28 | 0 | 0 | 2..53 | -2..14 |
| California or Bust! | 0 | 2 | 0 | 1 | 0 | 0 | 13 | 0 | 0 | 55..70 | 1..1 |
| Castle o' the Air | 0 | 0 | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 63..75 | -1..8 |
| Davis Station | 1 | 2 | 0 | 1 | 0 | 0 | 38 | 0 | 0 | 58..77 | -6..8 |
| Demo House | 1 | 0 | 0 | 0 | 0 | 0 | 12 | 0 | 0 | 62..71 | -1..5 |
| Empty House | 0 | 0 | 0 | 0 | 0 | 0 | 16 | 0 | 0 | 62..70 | -1..4 |
| Fun House | 0 | 0 | 0 | 0 | 0 | 0 | 2 | 0 | 0 | 56..71 | 1..5 |
| Grand Prix | 1 | 0 | 0 | 0 | 0 | 0 | 33 | 0 | 0 | 53..79 | -2..9 |
| ImagineHouse PRO II | 1 | 0 | 0 | 0 | 0 | **1** | 108 | 0 | 0 | 6..95 | -3..11 |
| In The Mirror | 1 | 0 | 0 | 0 | 0 | 0 | 3 | 0 | 0 | 55..71 | -2..9 |
| Land of Illusion | 0 | 2 | 0 | 1 | 0 | 0 | 9 | 0 | 0 | 49..77 | -2..17 |
| Leviathan | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0..94 | -6..19 |
| Metropolis | 1 | 0 | 0 | 0 | 0 | 0 | 17 | 0 | 0 | 54..68 | -1..12 |
| Nemo's Market | 1 | 2 | 0 | 1 | 0 | 0 | 23 | 0 | 0 | 54..95 | 0..2 |
| Rainbow's End | 1 | 2 | 0 | 1 | 0 | 0 | 8 | 0 | 0 | 52..87 | -7..6 |
| Sampler | 0 | 0 | 0 | 0 | 0 | 0 | 2 | 0 | 0 | 66..67 | 1..1 |
| Slumberland | 1 | 0 | 0 | 0 | 0 | 0 | 25 | 0 | 0 | 0..86 | -7..10 |
| SpacePods | 1 | 2 | 0 | 1 | 0 | 0 | 43 | 0 | 0 | 110..127 | 4..39 |
| Teddy World | 1 | 0 | 0 | 0 | 0 | 0 | 16 | 0 | 0 | 1..113 | -5..26 |
| The Asylum Pro | 1 | 0 | 0 | 0 | 0 | 0 | 16 | 0 | 0 | 63..72 | 0..13 |
| Titanic | 0 | 0 | 0 | 0 | 0 | **1** | 1 | 0 | 0 | 46..73 | -7..5 |

Facts established:

1. **`openings` is a runtime-only field.** It is 0 in every room of every one of the
   22 houses (`survey.py` "open!=0" column is 0 everywhere). It is written by nothing
   in the shipped code — `CreateNewRoom` sets it to 0 (`GliderPRO/Sources/Room.c:186`)
   and no other assignment exists. `DetermineRoomOpenings` writes the *globals*
   `leftOpen`/`rightOpen`/`topOpen`/`bottomOpen`, not the field. A Go port should
   preserve the field for round-trip fidelity and otherwise ignore it.
2. **`numObjects` always equals the count of slots with `what != -1`** — 0 mismatches
   in 22 houses, 4 070 rooms (re-derived: that is the exact sum of `nRooms` over the
   22 shipped houses). So `numObjects` is a trustworthy loop bound, but the
   game itself never trusts it: every object walk in the shipped code loops
   `i < kMaxRoomObs` (24) and skips `kObjectIsEmpty`
   (e.g. `GliderPRO/Sources/Play.c:611`, `GliderPRO/Sources/Objects.c:262`).
3. **`wardBit` (flags bit 0) is 0 in all 22 houses.** It is used only by
   `GliderPRO/Sources/Interactions.c` gating an easter-egg-ish behaviour, not by
   progression.
4. **`phoneBit` (flags bit 1) is set in exactly 8 houses**: Art Museum, CD Demo
   House, California or Bust!, Davis Station, Land of Illusion, Nemo's Market,
   Rainbow's End, SpacePods. In those, the random telephone ring is *suppressed*
   (`GliderPRO/Sources/Play.c:746`).
5. **`starCountOff` (flags bit 2) is set in exactly one house**: Art Museum
   (`flags == 6`). There, the opening banner does not print "N stars remaining".
6. **`hasGame` is 1 in exactly two houses**: ImagineHouse PRO II and Titanic.
7. **15 of 22 houses are locked** (`timeStamp & 1 == 1`): Art Museum, CD Demo House,
   Davis Station, Demo House, Grand Prix, ImagineHouse PRO II, In The Mirror,
   Leviathan, Metropolis, Nemo's Market, Rainbow's End, Slumberland, SpacePods,
   Teddy World, The Asylum Pro. Locked means `houseUnlocked = false`, which disables
   the editor for that house but has no effect on play.
8. **`visited` flags persist on disk and are stale author playtesting residue.**
   Observed counts range from 0 (Leviathan) to 108 of 279 (ImagineHouse PRO II).
   A new game clears every one of them (`SetObjectsToDefaults`, §6.5). They matter
   because `CountRoomsVisited()` is the high-score "level" (§13), and because a
   *resumed* game would not clear them.
9. **`suite` ranges 0..127 and `floor` ranges -7..39 across the whole corpus.** Both
   fit comfortably in `short`. Note SpacePods has `suite` up to 127 and `floor` up to
   39 — a Go port must not assume the 0..99 / -8..91 range implied by the link codec
   (§5.4).

**Test 3 — `houseType.unusedShort` is uninitialized garbage.** Observed values at
offset 2:

| House | `unusedShort` |
|---|---:|
| Art Museum | -30082 |
| California or Bust! | 13107 |
| Castle o' the Air | 259 |
| Davis Station | 222 |
| Fun House | 26228 |
| Grand Prix | 60 |
| In The Mirror | 147 |
| Land of Illusion | 259 |
| Metropolis | 196 |
| Titanic | 2074 |
| *(the other 12 houses)* | 0 |

`InitializeEmptyHouse` never assigns it (`GliderPRO/Sources/House.c:108-168`), and
`NewHandle` does not zero memory, so whatever was on the Mac's heap got written to
disk. **A Go loader must not validate this field, and a Go writer should emit 0.**

**Test 4 — a full header dump.** `Sampler` (the smallest house, 2 rooms):

```
file size            = 1564 bytes (0x61C)
version              = 0x0200
unusedShort          = 0
timeStamp            = 0x3540BFE8 (893435880)  low bit=0 -> houseUnlocked=True
flags                = 0x00000000  wardBit(b0)=0 phoneBit(b1)=0 starCountOff(b2)=0
initial (h,v)        = (384, 85)
banner               = "Welcome to Omid's Happy Home."
trailer              = 'Congratulations.'
highScores.banner    = 'Your Message Here'
highScores.names     = ['Your Name', 'Your Name', '--------------' x8]
highScores.scores    = (5200, 5100, 0, 0, 0, 0, 0, 0, 0, 0)
highScores.timeStamps= (3040919567, 3040919419, 0, 0, 0, 0, 0, 0, 0, 0)
highScores.levels    = (2, 1, 0, 0, 0, 0, 0, 0, 0, 0)
savedGame raw 40 bytes @820:
  00 01 00 08 00 00 00 00 00 00 00 00 00 00 00 00
  00 47 3e 04 00 00 00 ff 00 00 ff ff ff ff ff ff
  00 01 ff ff ff ff cc cc
savedGame.version    = 0x0001          <- garbage, hasGame == 0
savedGame.wasStarsLeft= 8
savedGame.timeStamp  = 0x00000000
savedGame.where(v,h) = (0, 0)
savedGame.score      = 0
savedGame.unusedLong = 0x00473E04   unusedLong2 = 255
savedGame.energy     = 0   bands = -1
savedGame.roomNumber = -1  gliderState = -1  numGliders = 1  foil = -1
savedGame.unusedShort= -1
savedGame.facing     = 204  showFoil = 204
hasGame              = 0   unusedBoolean = 2
firstRoom            = 1
nRooms               = 2
expected size        = 866 + 348*2 = 1562   (delta +2)

--- room 0 @0x362 ---
  name='Entrance' bounds=0 leftStart=32 rightStart=27 unusedByte=0 visited=1
  background=2003 tiles=(0, 1, 4, 1, 2, 1, 1, 3) floor=1 suite=67 openings=0x0000 numObjects=7
   obj[ 4] what=0x35 (53=kFloorTrans)   raw=01 2e 01 31 00 00 1a 35 05 00
   obj[ 5] what=0x36 (54=kCeilingTrans) raw=00 06 01 35 00 00 ff ff ff 00
   obj[ 6] what=0x33 (51=kMailboxLf)    raw=00 2f 01 61 00 00 19 d1 02 00

--- room 1 @0x4BE ---
  name='Welcome' bounds=0 leftStart=32 rightStart=32 unusedByte=0 visited=1
  background=2012 tiles=(1, 1, 1, 2, 3, 1, 0, 1) floor=1 suite=66 numObjects=4
   obj[ 3] what=0x2C (44=kStar)         raw=00 c9 00 36 00 00 00 00 00 01
```

Reading obj[5] of room 0 by the `transportType` field map: `topLeft = (v=6, h=0x135=309)`,
`tall = 0`, `where = 0xFFFF = -1`, `who = 0xFF = 255`, `wide = 0` — i.e. an
**unlinked** ceiling transport, exactly the `DoUnlink` signature
(`GliderPRO/Sources/Link.c:333-340`). obj[4] `kFloorTrans`: `topLeft = (302, 305)`,
`tall = 0`, `where = 0x1A35 = 6709`, `who = 5`. Decoding `where` by the v2 rule
(§5.2): `suite = 6709/100 = 67`, `floor = 6709%100 - 8 = 9 - 8 = 1` → room with
`(floor 1, suite 67)` = **room 0, itself**, object 5. So the Sampler's floor
transporter goes to the ceiling transporter in the same room — a vertical
teleport within one room, which is the `sameRoom` case at
`GliderPRO/Sources/Transit.c:322`. obj[6] `kMailboxLf`: `where = 0x19D1 = 6609` →
`suite = 66, floor = 9-8 = 1` → room 1; `who = 2`.

obj[3] of room 1 by the `bonusType` field map: `topLeft = (201, 54)`, `length = 0`,
`points = 0`, `state = 0`, `initial = 1`. **A star with `initial = 1`.** This is why
`Sampler`'s star count is 1 (Test 5).

**Test 5 — `Demo House` validates all four `COMPILEDEMO` assertions at once.** Observed:

```
version = 0x0200   unusedShort = 0
timeStamp = 0x2C53B041  low bit=1 -> houseUnlocked=False
flags = 0x00000000
initial (h,v) = (49, 107)
banner  = 'Welcome to the Demo House!\rThis is a small beginner house that\racts
           as a sort of tutorial.\r(house by Kim Money)'
trailer = "Excellent!\rThat's the extent of the Demo House though.  The house\r
           \"Slumberland\" has over 400 rooms!  Good luck."
highScores.banner = 'The Return of Ozma!'  names[0] = 'Ozma'  scores[0] = 7400  levels[0] = 13
hasGame = 0  firstRoom = 0  nRooms = 45   file size = 16526
room 0: name='Air Vents' background=3000 tiles=(0,1,2,3,4,5,6,7) floor=1 suite=63 numObjects=10
```

The four `#ifdef COMPILEDEMO` gates in `ReadHouse` are:

```
if (byteCount != 16526L) return(false);           // HouseIO.c:349
if (numberRooms != 45)   return(false);           // HouseIO.c:382
if (houseUnlocked)       return(false);           // HouseIO.c:404
if (whichRoom != 0)      return(false);           // HouseIO.c:412
```

Observed: 16526 bytes ✓, 45 rooms ✓, locked ✓, `firstRoom == 0` ✓. All four match
`Demo House` exactly. This is independent confirmation that the offsets at 862 and
864 are `firstRoom` and `nRooms`, and that the file length is the whole struct image.

Note the banner and trailer contain embedded `\r` (0x0D) as a **line separator inside
a Pascal string**; `GetLineOfText` splits on it (`GliderPRO/Sources/Utilities.c`, used
at `GliderPRO/Sources/Banner.c:130` and `GliderPRO/Sources/GameOver.c:105`).

**Test 6 — the two embedded saved games, byte for byte.** Raw 40 bytes at offset 820:

```
ImagineHouse PRO II:
  01 00 00 03 ab 09 1a fd 00 3d ff f6 00 00 17 0c
  00 00 00 00 00 00 00 00 ff 6a 00 17 00 2d 00 00
  00 05 00 08 00 00 01 01
Titanic:
  01 00 00 01 ab ad e4 37 00 f0 00 55 00 00 12 5c
  00 00 00 00 00 00 00 00 00 00 00 00 00 68 00 00
  00 02 00 00 00 00 01 00
```

Decoded by the `gameType` field map:

| Field | ImagineHouse PRO II | Titanic |
|---|---:|---:|
| `version` | 0x0100 | 0x0100 |
| `wasStarsLeft` | 3 | 1 |
| `timeStamp` | 0xAB091AFD | 0xABADE437 |
| `where` (v, h) | (61, -10) | (240, 85) |
| `score` | 5900 | 4700 |
| `unusedLong`, `unusedLong2` | 0, 0 | 0, 0 |
| `energy` | **-150** | 0 |
| `bands` | 23 | 0 |
| `roomNumber` | 45 | 104 |
| `gliderState` | 0 (`kGliderNormal`) | 0 |
| `numGliders` | 5 | 2 |
| `foil` | 8 | 0 |
| `unusedShort` | 0 | 0 |
| `facing` | 1 (`kFaceRight`) | 1 |
| `showFoil` | 1 | 0 |
| — house `timeStamp` | 0x2C1DC93D | 0x2D0246FE |
| — `rooms[roomNumber].name` | `'Grin and Bear it'` (floor 1, suite 70) | `'Oceanic Depths'` (floor -1, suite 72) |

Two independent observations:

1. `savedGame.version` is **0x0100**, but `kSavedGameVersion` is **0x0200**
   (`GliderPRO/Sources/SavedGames.c:14`). The validation
   `if (savedGame->version != kSavedGameVersion)` would reject both
   (`GliderPRO/Sources/SavedGames.c:243-250`).
2. `savedGame.timeStamp` does not equal the house's `timeStamp` in either case
   (0xAB091AFD vs 0x2C1DC93D; 0xABADE437 vs 0x2D0246FE), which the validation at
   `GliderPRO/Sources/SavedGames.c:235-242` also rejects. (Order of the three
   validations in `OpenSavedGame`: house *name* `:227`, then `timeStamp` `:235`, then
   `version` `:243`.)

So even if the resume path were live, neither shipped saved game would load. Combined
with §15, the resume path is unreachable *and* the data is invalid. **A Go port should
load and preserve these bytes but must not attempt to resume from them.**

`energy = -150` is worth noting: `energy` is `batteryTotal`, and *negative* battery
is helium (the glider rises). See `kBatteryLow 17` / `kHeliumLow -38`
(`GliderPRO/Sources/Scoreboard.c:75-78`). The saved game therefore carries helium,
not battery.

**Test 7 — the star count.** `verify_links.py` counts `what == 0x2C` (`kStar` = 44)
objects across all live rooms. This is the same walk `CountStarsInHouse()` performs
(`GliderPRO/Sources/Banner.c:89-111`):

| House | stars | House | stars |
|---|---:|---|---:|
| Art Museum | 6 | Metropolis | 4 |
| CD Demo House | 9 | Nemo's Market | 5 |
| California or Bust! | 1 | Rainbow's End | 5 |
| Castle o' the Air | 4 | Sampler | 1 |
| Davis Station | 4 | Slumberland | 6 |
| Demo House | 1 | SpacePods | 1 |
| Empty House | 1 | Teddy World | 1 |
| **Fun House** | **0** | The Asylum Pro | 1 |
| Grand Prix | 3 | Titanic | 1 |
| ImagineHouse PRO II | 3 | | |
| In The Mirror | 1 | | |
| Land of Illusion | 5 | | |
| Leviathan | 6 | | |

**`Fun House` has zero stars.** `CountStarsInHouse` returns 0, so
`numStarsRemaining = 0` at the start of the game — and the win check
`if (numStarsRemaining <= 0) FlagGameOver();` is only evaluated *when a star is
collected* (`GliderPRO/Sources/Interactions.c:943`). Since there are no stars, it is
never evaluated, so `Fun House` is unwinnable: the only exit is running out of
gliders. **This is not a bug in the loader and a Go port must reproduce it** — do
*not* add a "0 stars ⇒ instant win" check at game start.

Note `CountStarsInHouse` counts stars whose *slot is a `kStar`*, regardless of
`data.c.initial`. It does **not** filter on `initial`
(`GliderPRO/Sources/Banner.c:100-107`). So a star authored with `initial = 0` (already
taken) still increments the required count — making such a house unwinnable too.

---

## 3. Room addressing

### 3.1 `thisRoom` is a copy, not a pointer into the house

The single most important structural fact about room handling:

```c
void ForceThisRoom (short roomNumber)                    // Room.c:369
{
    char        tagByte;
    if (roomNumber == -1) return;
    tagByte = HGetState((Handle)thisHouse);
    HLock((Handle)thisHouse);
    if (roomNumber < (*thisHouse)->nRooms)
        *thisRoom = (*thisHouse)->rooms[roomNumber];     // 348-byte struct COPY
    else
        YellowAlert(kYellowIllegalRoomNum, 0);
    HSetState((Handle)thisHouse, tagByte);
    previousRoom = thisRoomNumber;
    thisRoomNumber = roomNumber;
}
```

`thisRoom` is a `roomPtr` to a standalone 348-byte buffer. `ForceThisRoom` **does not
write the outgoing room back to the house.** Only `CopyThisRoomToRoom` does:

```c
void CopyThisRoomToRoom (void)                           // Room.c:354
{
    if ((noRoomAtAll) || (thisRoomNumber == -1)) return;
    ...HLock...
    (*thisHouse)->rooms[thisRoomNumber] = *thisRoom;
    ...HSetState...
}

void CopyRoomToThisRoom (short roomNumber)               // Room.c:343
{
    if (roomNumber == -1) return;
    CopyThisRoomToRoom();
    ForceThisRoom(roomNumber);
}
```

`CopyRoomToThisRoom` is the *save-then-load* pair; `ForceThisRoom` is *load only,
discarding changes to the copy*. Which one gets used matters enormously:

| Caller | Uses | Effect on the outgoing room |
|---|---|---|
| `NewGame` → `SetHouseToFirstRoom` (`Play.c:373`) | `ForceThisRoom` | discarded (nothing to save) |
| `NewGame` → `SetHouseToSavedRoom` (`Play.c:382`) | `ForceThisRoom` | discarded |
| `MoveRoomToRoom` (`Transit.c:165` etc.) | `ForceThisRoom` | **discarded** |
| `TransportRoomToRoom` (`Transit.c:325`) | `ForceThisRoom` | **discarded** |
| `MoveDuctToDuct` (`Transit.c:364`) | `ForceThisRoom` | **discarded** |
| `MoveMailToMail` (`Transit.c:403`) | `ForceThisRoom` | **discarded** |
| `ReadHouse` (`HouseIO.c:425`) | `CopyRoomToThisRoom` | saved |
| Editor room navigation (`Room.c`, `Map.c`) | `CopyRoomToThisRoom` | saved |
| `ConvertHouseVer1To2` (`House.c:766`, `:815`) | `ForceThisRoom` + explicit `CopyThisRoomToRoom` | saved explicitly |

**Therefore, during gameplay, every mutation to persistent room state must be written
directly into `(*thisHouse)->rooms[...]`, not into `thisRoom`** — because `thisRoom`
is thrown away at the next transition. The shipped code does exactly this:

- `HandleRoomVisitation` writes `thisHousePtr->rooms[localNumbers[kCentralRoom]].visited = true;`
  *and* `thisRoom->visited = true;` (`GliderPRO/Sources/Transit.c:440-444`) — the
  first for persistence, the second so the in-flight copy agrees.
- `SetObjectState(room, object, ...)` indexes `(*thisHouse)->rooms[room].objects[object]`
  (`GliderPRO/Sources/Objects.c:366+`), never `thisRoom`.

A Go port that models the current room as a *pointer into* the house room slice would
be *more* correct in the "obvious" sense but would change behaviour: any transient
edit to `thisRoom` (there are none during play, but there are during editing) would
become persistent. Model it as a copy.

### 3.2 Room lookup: linear scans everywhere

```c
Boolean RoomExists (short suite, short floor, short *roomNum)     // Room.c:390
{
    if (suite < 0) return(false);
    for (i = 0; i < numberRooms; i++)
        if (((*thisHouse)->rooms[i].floor == floor) &&
            ((*thisHouse)->rooms[i].suite == suite))
        { *roomNum = i; return(true); }
    return(false);
}

short GetRoomNumber (short floor, short suite)                    // Room.c:737
{
    for (i = 0; i < numberRooms; i++)
        if (((*thisHouse)->rooms[i].floor == floor) &&
            ((*thisHouse)->rooms[i].suite == suite))
            return(i);
    return(kRoomIsEmpty);
}
```

Two things a port must copy:

1. **First match wins.** If two live rooms share `(floor, suite)`, only the
   lower-indexed one is ever reachable by address. No shipped house has a duplicate
   (verified: `verify_links.py` builds the index with `setdefault`, and no house's
   link audit changed when duplicates were checked for).
2. **`RoomExists` rejects `suite < 0` up front but `GetRoomNumber` does not.**
   `GetRoomNumber(-8, -1)` would happily match a tombstoned room (`suite == -1`) if
   its `floor` happened to be -8. This is the mechanism by which `where == -100`
   links resolve to nothing (§5.5): the extracted `suite` is -1 and no live room has
   `suite == -1`... unless a *deleted* room does. Since deleted rooms keep their old
   `floor`, a `where == -100` link in a house that has a tombstone at
   `floor == -8` would resolve to that tombstone. No shipped house has a tombstone at
   all (`empty=0` in every row of Test 1), so this cannot bite in practice, but it is
   a real difference between `RoomExists` and `GetRoomNumber` that a Go port should
   preserve rather than "fix".

`GetRoomFloorSuite` is the inverse and is tombstone-aware:

```c
Boolean GetRoomFloorSuite (short room, short *floor, short *suite)   // Room.c:711
{
    if ((*thisHouse)->rooms[room].suite == kRoomIsEmpty)
    { *floor = 0; *suite = kRoomIsEmpty; return(false); }
    *floor = (*thisHouse)->rooms[room].floor;
    *suite = (*thisHouse)->rooms[room].suite;
    return(true);
}
```

`RoomNumExists(short roomNum)` composes the two (`GliderPRO/Sources/Room.c:425-435`).

### 3.3 `GetNeighborRoomNumber` and the 3×3 grid

```
1.  GetNeighborRoomNumber(which):
2.    switch (which):
3.      kCentralRoom:    hDelta =  0, vDelta =  0
4.      kNorthRoom:      hDelta =  0, vDelta = +1
5.      kNorthEastRoom:  hDelta = +1, vDelta = +1
6.      kEastRoom:       hDelta = +1, vDelta =  0
7.      kSouthEastRoom:  hDelta = +1, vDelta = -1
8.      kSouthRoom:      hDelta =  0, vDelta = -1
9.      kSouthWestRoom:  hDelta = -1, vDelta = -1
10.     kWestRoom:       hDelta = -1, vDelta =  0
11.     kNorthWestRoom:  hDelta = -1, vDelta = +1
12.     default:         return kRoomIsEmpty
13.   lock thisHouse
14.   wantSuite = rooms[thisRoomNumber].suite + hDelta
15.   wantFloor = rooms[thisRoomNumber].floor + vDelta
16.   result = kRoomIsEmpty
17.   for i in 0 .. numberRooms-1:
18.     if rooms[i].suite == wantSuite and rooms[i].floor == wantFloor:
19.        result = i; break
20.   unlock thisHouse
21.   return result
```
(`GliderPRO/Sources/Room.c:562-637`)

Note line 14/15: the base address is read from **`(*thisHouse)->rooms[thisRoomNumber]`**,
not from `thisRoom`. During play these are identical. Note also that this scan does
*not* skip tombstones — a deleted room at the target address would be returned, and
then `IsRoomAStructure(kRoomIsEmpty)` returns false while `DrawRoomBackground` would
try to draw it. Again, no shipped house has tombstones.

`SetToNearestNeighborRoom(wasFloor, wasSuite)` (`GliderPRO/Sources/Room.c:639-709`) is
an editor-only clockwise spiral search used after a room deletion:

```
distance = 1; h = -1; v = 0; hStep = 0; vStep = -1
loop:  if RoomExists(wasSuite + h, wasFloor + v, &n): select n; done
       advance (h,v) by (hStep,vStep); on reaching a corner rotate (hStep,vStep)
       when hStep == -1 and vStep == 0: distance++, restart the ring
```

### 3.4 `IsRoomAStructure`

Used to decide whether to draw floor support beams between vertically adjacent rooms:

```c
Boolean IsRoomAStructure (short roomNum)                 // Room.c:763
{
    if (roomNum == kRoomIsEmpty) return(false);
    theBack = (*thisHouse)->rooms[roomNum].background;
    if (theBack >= kUserBackground)          // 3000
    {   if ((*thisHouse)->rooms[roomNum].bounds != 0)
            return (((*thisHouse)->rooms[roomNum].bounds & 32) == 32);
        else
            return (theBack < kUserStructureRange);       // 3300
    }
    switch (theBack) {
        case kPaneledRoom: case kSimpleRoom: case kChildsRoom: case kAsianRoom:
        case kUnfinishedRoom: case kSwingersRoom: case kBathroom: case kLibrary:
        case kSkywalk: case kRoof:
            return(true);
        default:
            return(false); }
}
```

So for a **user background** (`>= 3000`): if `bounds != 0`, bit 5 (value 32) of
`bounds` is the "is a structure" flag; otherwise the PICT ID range decides
(3000..3299 = structure, 3300+ = not). This is the only use of `bounds` bit 5.

Built-in background IDs (`GliderPRO/Headers/GliderDefines.h:227-244`):

| Constant | Value | Structure? | Has floor? | Has ceiling? | Shadow visible? |
|---|---:|:-:|:-:|:-:|:-:|
| `kSimpleRoom` | 2000 | yes | yes | yes | yes |
| `kPaneledRoom` | 2001 | yes | yes | yes | yes |
| `kBasement` | 2002 | no | yes | yes | yes |
| `kChildsRoom` | 2003 | yes | yes | yes | yes |
| `kAsianRoom` | 2004 | yes | yes | yes | yes |
| `kUnfinishedRoom` | 2005 | yes | yes | yes | yes |
| `kSwingersRoom` | 2006 | yes | yes | yes | yes |
| `kBathroom` | 2007 | yes | yes | yes | yes |
| `kLibrary` | 2008 | yes | yes | yes | yes |
| `kGarden` | 2009 | no | yes | **no** | yes |
| `kSkywalk` | 2010 | yes | yes | yes | yes |
| `kDirt` | 2011 | no | yes | yes | yes |
| `kMeadow` | 2012 | no | yes | **no** | yes |
| `kField` | 2013 | no | yes | **no** | yes |
| `kRoof` | 2014 | yes | **yes** | **no** | **no** |
| `kSky` | 2015 | no | **no** | **no** | **no** |
| `kStratosphere` | 2016 | no | **no** | **no** | **no** |
| `kStars` | 2017 | no | **no** | **no** | **no** |
| `kUserBackground` | 3000 | via `bounds`/range | via `bounds` bit 3 | via `bounds` bit 1 | via `bounds` bit 3 |
| `kUserStructureRange` | 3300 | boundary | | | |

Sources: structure `Room.c:763-812`, floor `Room.c:1138-1168`, ceiling
`Room.c:1172-1205`, shadow `Room.c:1103-1134`.

**`kRoof` is the odd one out: `DoesRoomHaveFloor()` returns true for it (so a glider
does not fall out the bottom) but `IsShadowVisible()` returns false (so no shadow is
drawn) and `DoesRoomHaveCeiling()` returns false (so you can fly off the top).** The
roof "floor" is not a plane at all — it is the sloped-tile test in
`CheckRoofCollision` (§8.2.4).

### 3.5 `firstRoom` and `GetFirstRoomNumber`

```c
short GetFirstRoomNumber (void)                          // House.c:196
{
    if ((*thisHouse)->nRooms <= 0)
    { firstRoom = -1; noRoomAtAll = true; }
    else
    {   firstRoom = (*thisHouse)->firstRoom;
        if ((firstRoom >= (*thisHouse)->nRooms) || (firstRoom < 0))
            firstRoom = 0; }
    return (firstRoom);
}
```

`firstRoom` in the file is a **room index**, not an address. It is clamped to
`[0, nRooms)` on load, falling back to 0. It is *not* checked for being a tombstone,
so a house whose `firstRoom` points at a deleted room would start the player in an
un-drawable room. Observed `firstRoom` values across the corpus are all in range
(Test 1); the largest is 259 for SpacePods (of 402 rooms).

`firstRoom` is (re)assigned by the editor in two places:
`CreateNewRoom` sets `(*thisHouse)->firstRoom = thisRoomNumber;` if the house had no
rooms at all (`GliderPRO/Sources/Room.c:230-231`), and `DeleteRoom` reassigns it if
the deleted room was the first (`GliderPRO/Sources/Room.c:475-483`).

---

## 4. Room openness: `DetermineRoomOpenings` and the wall predicates

Every time a room becomes current, four booleans and two X thresholds are recomputed.
These *are* the room-transition rules for the four sides.

### 4.1 `DetermineRoomOpenings` in full

```
1.  DetermineRoomOpenings():
2.    whichBack = thisRoom->background
3.    leftTile  = thisRoom->tiles[0]
4.    rightTile = thisRoom->tiles[kNumTiles - 1]        // tiles[7]
5.    if whichBack >= kUserBackground:                  // >= 3000
6.       if thisRoom->bounds != 0:
7.          boundsCode = thisRoom->bounds >> 1
8.       else:
9.          boundsCode = GetOriginalBounding(whichBack) // the 'bnds' resource
10.      leftOpen  = ((boundsCode & 0x0001) == 0x0001)
11.      rightOpen = ((boundsCode & 0x0004) == 0x0004)
12.      leftThresh  = leftOpen  ? kNoLeftWallLimit  : kLeftWallLimit
13.      rightThresh = rightOpen ? kNoRightWallLimit : kRightWallLimit
14.    else switch whichBack:
15.      kSimpleRoom, kPaneledRoom, kBasement, kChildsRoom, kAsianRoom,
16.      kUnfinishedRoom, kSwingersRoom, kBathroom, kLibrary, kSky, default:
17.         leftThresh  = (leftTile  == 0)              ? kLeftWallLimit  : kNoLeftWallLimit
18.         rightThresh = (rightTile == kNumTiles - 1)  ? kRightWallLimit : kNoRightWallLimit
19.         leftOpen    = (leftTile  != 0)
20.         rightOpen   = (rightTile != kNumTiles - 1)
21.      kDirt:
22.         leftThresh  = (leftTile  == 1)              ? kLeftWallLimit  : kNoLeftWallLimit
23.         rightThresh = (rightTile == kNumTiles - 1)  ? kRightWallLimit : kNoRightWallLimit
24.         leftOpen    = (leftTile  != 0)              // NOTE: 0, not 1 !!
25.         rightOpen   = (rightTile != kNumTiles - 1)
26.      kMeadow:
27.         leftThresh  = (leftTile  == 6)              ? kLeftWallLimit  : kNoLeftWallLimit
28.         rightThresh = (rightTile == 7)              ? kRightWallLimit : kNoRightWallLimit
29.         leftOpen    = (leftTile  != 6)
30.         rightOpen   = (rightTile != 7)
31.      kGarden, kSkywalk, kField, kStratosphere, kStars:
32.         leftThresh  = kNoLeftWallLimit
33.         rightThresh = kNoRightWallLimit
34.         leftOpen    = true
35.         rightOpen   = true
36.    bottomOpen = !DoesRoomHaveFloor()
37.    topOpen    = !DoesRoomHaveCeiling()
```
(`GliderPRO/Sources/Room.c:816-935`)

**Line 22 vs line 24 is a genuine inconsistency in the original and must be
reproduced.** For `kDirt`, the *threshold* is the closed-wall plane when
`tiles[0] == 1`, but the *openness* flag is false only when `tiles[0] == 0`. So a
`kDirt` room with `tiles[0] == 1` gets `leftThresh = kLeftWallLimit` (12, a wall) but
`leftOpen = true`. Reading `CheckEscapeLeft` (§8.2.3), `leftThresh == kLeftWallLimit`
is precisely the test that makes the wall solid, so such a room has a solid left wall
that reports itself as open. `leftOpen`/`rightOpen` are consumed only by
`Interactions.c` for `ignoreLeft`/`ignoreRight` door logic and by the editor's
`bounds` display, so the practical effect is narrow — but it is observable.

Conversely, a `kDirt` room with `tiles[0] == 0` gets `leftThresh = kNoLeftWallLimit`
(-24, open) and `leftOpen = false`. That combination lets the glider *leave* through
the left (because the threshold is the open plane) while `leftOpen` says otherwise.

The `bounds >> 1` at line 7 is the important detail for user backgrounds: `bounds`
packs one extra bit below the four bounding bits (bit 0 is unused-or-a-flag, bit 5 is
"is structure", see §3.4), so the four bounding bits sit at `bounds` bits 1..4, and
after the shift they are at bits 0..3.

### 4.2 The four bounding bits

`GetOriginalBounding` builds the same nibble from the `'bnds'` resource:

```c
short GetOriginalBounding (short theID)                  // Room.c:937
{
    theBounds = (Rect **)GetResource('bnds', theID);
    if (theBounds == nil)
    {   if (PictIDExists(theID)) YellowAlert(kYellowNoBoundsRes, 0);
        boundCode = 0; }
    else
    {   boundCode = 0;
        if ((*theBounds)->left  != 0) boundCode += 1;
        if ((*theBounds)->top   != 0) boundCode += 2;
        if ((*theBounds)->right != 0) boundCode += 4;
        if ((*theBounds)->bottom!= 0) boundCode += 8;
        ReleaseResource((Handle)theBounds); }
    return (boundCode);
}
```

| Bit | Value | `'bnds'` field nonzero | Meaning |
|---:|---:|---|---|
| 0 | 1 | `left` | left wall is **open** |
| 1 | 2 | `top` | ceiling is **absent**? — no: see below |
| 2 | 4 | `right` | right wall is **open** |
| 3 | 8 | `bottom` | floor is **absent**? — no: see below |

The polarity of bits 1 and 3 is *inverted* relative to bits 0 and 2, because the
predicates test for the bit being **clear**:

```c
// Room.c:1172 DoesRoomHaveCeiling
hasCeiling = ((boundsCode & 0x0002) != 0x0002);
// Room.c:1138 DoesRoomHaveFloor      and  Room.c:1103 IsShadowVisible
hasFloor   = ((boundsCode & 0x0008) != 0x0008);
```

So: **bit 0 set = left open; bit 2 set = right open; bit 1 set = NO ceiling;
bit 3 set = NO floor.** A Go port must not "normalise" these to a uniform polarity.

The complete `bounds` bit map for a user background (before the `>> 1`):

| `bounds` bit | Value | Meaning |
|---:|---:|---|
| 0 | 1 | (unused by any read; `bounds != 0` is the "explicit bounds present" test) |
| 1 | 2 | left wall open |
| 2 | 4 | no ceiling |
| 3 | 8 | right wall open |
| 4 | 16 | no floor |
| 5 | 32 | room is a structure (`IsRoomAStructure`) |

**Observed:** every room of every shipped house has `bounds == 0` except where a user
background is in use; the Sampler and Demo House rooms shown in §2.6 all have
`bounds == 0` and use built-in backgrounds 2003/2012/3000. Because `bounds == 0` is
also the sentinel for "no explicit bounds", **a room cannot express "left closed,
ceiling present, right closed, floor present, not a structure" via `bounds`** — that
combination is bit-pattern 0, which routes to the `'bnds'` resource instead. That is
why `bounds` bit 0 exists as a value-1 "always set me" marker in editor-written
rooms.

### 4.3 `numLights` and `DrawLighting` (why lit lights barely matter)

```c
short GetNumberOfLights (short where)                    // Room.c:970
{
    ...
    for (i = 0; i < kMaxRoomObs; i++)
      switch ((*thisHouse)->rooms[where].objects[i].what) {
        case kCeilingLight: case kLightBulb: case kTableLamp: case kHipLamp:
        case kDecoLamp: case kFlourescent: case kTrackLight: case kInvisLight:
            if ((*thisHouse)->rooms[where].objects[i].data.f.state) numLights++;
            break; }
    return numLights;
}
```

The count of **currently-on** lights in a room. It feeds two things:

1. `DrawRoomBackground(..., numLights)` — which picks a *dark* or *lit* variant of the
   background art.
2. `UpdateOutletsLighting(room, nLights)` — sets `dinahs[i].hVel = nLights` for every
   `kOutlet` in the room (`GliderPRO/Sources/Trip.c:235-244`).

There is no third consumer, because:

```c
void DrawLighting (void)                                 // RoomGraphics.c:422
{
    if (numLights == 0) return;
    else { /* for future construction */ }
}
```

**`DrawLighting` is an empty stub.** Turning lights on or off changes which
background bitmap is composited and animates outlets; it never affects gameplay
physics, visibility, or collision. `RedrawRoomLighting` short-circuits when the
lit/unlit state has not flipped:

```
1.  RedrawRoomLighting():
2.    wasLit = (numLights > 0)
3.    numLights = GetNumberOfLights(localNumbers[kCentralRoom])
4.    isLit = (numLights > 0)
5.    if wasLit == isLit: return                         // nothing to do
6.    SetGWorld(backSrcMap, nil)
7.    DrawRoomBackground(thisRoom->background, localNumbers[kCentralRoom], numLights)
8.    DrawARoomsObjects(kCentralRoom, true)
9.    DrawLighting()                                     // no-op
10.   UpdateOutletsLighting(localNumbers[kCentralRoom], numLights)
11.   if numNeighbors > 3: DrawFloorSupport()
12.   RestoreWorkMap()
13.   AddRectToWorkRects(&localRoomsDest[kCentralRoom])
14.   shadowVisible = IsShadowVisible()
```
(`GliderPRO/Sources/RoomGraphics.c:434-...`)

Note line 5: it is the *boolean* lit-ness that matters, not the count. Going from 3
lights to 1 redraws nothing.

---

## 5. The link encoding

### 5.1 What a link is

Transporters, mailboxes, ducts, switches, and triggers all carry a two-field
destination: a `short where` and a `Byte who`. `where` is a packed
`(floor, suite)` address of the destination *room*; `who` is the destination
*object index* within that room (0..23, sentinel 255 = unlinked).

Object types that carry `data.d.where` / `data.d.who` (the **transports**,
`transportType`):

| Constant | Value (dec) | Value (hex) |
|---|---:|---|
| `kMailboxLf` | 51 | 0x33 |
| `kMailboxRt` | 52 | 0x34 |
| `kFloorTrans` | 53 | 0x35 |
| `kCeilingTrans` | 54 | 0x36 |
| `kInvisTrans` | 63 | 0x3F |
| `kDeluxeTrans` | 64 | 0x40 |

Object types that carry `data.e.where` / `data.e.who` (the **switches/triggers**,
`switchType`):

| Constant | Value (dec) | Value (hex) |
|---|---:|---|
| `kLightSwitch` | 65 | 0x41 |
| `kMachineSwitch` | 66 | 0x42 |
| `kThermostat` | 67 | 0x43 |
| `kPowerSwitch` | 68 | 0x44 |
| `kKnifeSwitch` | 69 | 0x45 |
| `kInvisSwitch` | 70 | 0x46 |
| `kTrigger` | 71 | 0x47 |
| `kLgTrigger` | 72 | 0x48 |

(`GliderPRO/Headers/GliderDefines.h:362-384`; the classification predicates are
`ObjectIsLinkTransport` at `GliderPRO/Sources/Objects.c:219` and
`ObjectIsLinkSwitch` at `GliderPRO/Sources/Objects.c:236`.)

### 5.2 The codec

```c
short MergeFloorSuite (short floor, short suite)                  // Link.c:34
{
    return ((suite * 100) + floor);
}

void ExtractFloorSuite (short combo, short *floor, short *suite)  // Link.c:41
{
    if ((*thisHouse)->version < 0x0200)     // old floor/suite combo
    {   *floor = (combo / 100) - kNumUndergroundFloors;
        *suite = combo % 100; }
    else
    {   *suite = combo / 100;
        *floor = (combo % 100) - kNumUndergroundFloors; }
}
```

`kNumUndergroundFloors` = 8 (`GliderPRO/Headers/GliderDefines.h:535`).

Four things to get exactly right:

1. **`MergeFloorSuite` takes a *biased* floor.** Every caller adds
   `kNumUndergroundFloors` first:
   ```c
   if (GetRoomFloorSuite(thisRoomNumber, &floor, &suite))   // Link.c:275
   {   floor += kNumUndergroundFloors;
       ... data.e.where = MergeFloorSuite(floor, suite); ... }
   ```
   (`GliderPRO/Sources/Link.c:275-319`; identically in `ConvertHouseVer1To2` at
   `GliderPRO/Sources/House.c:790` and `:805`.) `ExtractFloorSuite` subtracts it back.
   So the stored value is `suite*100 + (floor + 8)`.
2. **`ExtractFloorSuite` is version-sensitive and the two arms swap the roles of the
   quotient and the remainder.** v1 (`version < 0x0200`): quotient is the biased
   floor, remainder is the suite. v2: quotient is the suite, remainder is the biased
   floor. **`MergeFloorSuite` has no v1 arm** — it always writes the v2 layout. That
   asymmetry is what `ConvertHouseVer1To2` fixes up (§5.3).
3. **The version test reads the *live* `(*thisHouse)->version`.** So the same `where`
   value decodes differently depending on the house header, and if a house is
   converted mid-session (`ConvertHouseVer1To2` sets
   `(*thisHouse)->version = kHouseVersion;` at `GliderPRO/Sources/House.c:813`), every
   subsequent decode flips arm.
4. **C division and modulo truncate toward zero and the remainder takes the sign of
   the dividend.** For `combo = -100`: C gives `-100/100 = -1` and `-100%100 = 0`, so
   v2 yields `suite = -1`, `floor = 0 - 8 = -8`. Go's `/` and `%` match C exactly.
   **Python's `//` and `%` do not** (`-100//100 == -1` by luck, but `-100 % 100 == 0`
   also by luck; `-150//100 == -2` where C gives `-1`). The verification scripts here
   use explicit truncating helpers where negatives can appear.

Range implied by the codec: with `suite ∈ [0, 99]` and biased floor `∈ [0, 99]`, the
encodable space is `suite ∈ [0,99]`, `floor ∈ [-8, 91]`. Observed `suite` goes up to
127 (SpacePods) and up to 113 (Teddy World) — **outside the encodable range.**
`MergeFloorSuite(9, 110) = 11009`, and `ExtractFloorSuite(11009)` v2 gives
`suite = 110, floor = 9-8 = 1`, which round-trips fine, so suites above 99 are
actually safe as long as the biased floor stays in `[0, 99]`. What breaks is a
*negative* biased floor (floor < -8) or a biased floor ≥ 100 (floor ≥ 92):
`MergeFloorSuite(-1, 5) = 499`, and v2 extract gives `suite = 4, floor = 99-8 = 91`.
Observed floors span -7..39, i.e. biased 1..47 — always in range. So no shipped house
trips this. **A Go port should still implement the arithmetic literally rather than
"fixing" it, and should treat any link whose round-trip
`MergeFloorSuite(floor + 8, suite) != where` as unlinked**, which is what the audit
below does.

### 5.3 `ConvertHouseVer1To2`

```
1.  ConvertHouseVer1To2():
2.    CopyThisRoomToRoom(); wasRoom = thisRoomNumber
3.    put up a modal "converting…" message window; SpinCursor
4.    for i in 0 .. numberRooms-1:
5.      if (*thisHouse)->rooms[i].suite == kRoomIsEmpty: continue
6.      ForceThisRoom(i)
7.      for j in 0 .. kMaxRoomObs-1:
8.        switch thisRoom->objects[j].what:
9.          kMailboxLf, kMailboxRt, kFloorTrans, kCeilingTrans, kInvisTrans, kDeluxeTrans:
10.            if data.d.where != -1:
11.               ExtractFloorSuite(data.d.where, &floor, &suite)    // v1 arm, version still < 0x200
12.               floor += kNumUndergroundFloors
13.               data.d.where = MergeFloorSuite(floor, suite)       // writes the v2 layout
14.         kLightSwitch, kMachineSwitch, kThermostat, kPowerSwitch,
15.         kKnifeSwitch, kInvisSwitch, kTrigger, kLgTrigger:
16.            same, on data.e.where
17.      CopyThisRoomToRoom()
18.    (*thisHouse)->version = kHouseVersion            // 0x0200
19.    ForceThisRoom(wasRoom)
```
(`GliderPRO/Sources/House.c:746-822`)

It is invoked from exactly one place, the editor's Save command:

```c
case iSave:
    if (wasHouseVersion < kHouseVersion) ConvertHouseVer1To2();
    // SaveGame(false);
    if (!WriteHouse(theMode == kEditMode)) YellowAlert(kYellowFailedWrite, 0);
```
(`GliderPRO/Sources/Menu.c:452-461`)

**A v1 house is therefore played with v1 link decoding and never converted unless the
user opens it in the editor and saves.** A Go port must implement both arms of
`ExtractFloorSuite`.

`GenerateRetroLinks` (`GliderPRO/Sources/House.c:538-616`) is the reverse-index
builder used by the editor's link display: it walks the whole house looking for
objects that link *into* the current room. It is not used during play.

### 5.4 Runtime link resolution

At play time, links are read through the `masterObjects[]` table built by
`ListAllLocalObjects()` (`GliderPRO/Sources/Objects.c:300-349`), which flattens the
current 3×3 neighbourhood into one array with pre-resolved link fields:

```c
masterObjects[i].roomLink   = GetRoomLinked(objectPtr);     // Objects.c:126
masterObjects[i].objectLink = GetObjectLinked(objectPtr);   // Objects.c:177
```

`GetRoomLinked` is the important one:

```
1.  GetRoomLinked(theObject):
2.    if ObjectIsLinkTransport(theObject->what):
3.       if theObject->data.d.where == -1: return -1
4.       ExtractFloorSuite(theObject->data.d.where, &floor, &suite)
5.       return GetRoomNumber(floor, suite)
6.    else if ObjectIsLinkSwitch(theObject->what):
7.       if theObject->data.e.where == -1: return -1
8.       ExtractFloorSuite(theObject->data.e.where, &floor, &suite)
9.       return GetRoomNumber(floor, suite)
10.   else return -1
```
(`GliderPRO/Sources/Objects.c:126-175`)

`GetObjectLinked` mirrors it and maps the 255 sentinel to -1
(`GliderPRO/Sources/Objects.c:177-217`).

`WhatAreWeLinkedTo` then classifies the *destination object type*, which is what
selects the arrival animation:

```c
short WhatAreWeLinkedTo (short where, Byte who)          // Transit.c:31
{
    switch ((*thisHouse)->rooms[where].objects[who].what)
    {
        case kMailboxLf:    return kLinkedToLeftMailbox;    // 1
        case kMailboxRt:    return kLinkedToRightMailbox;   // 2
        case kCeilingTrans: return kLinkedToCeilingDuct;    // 3
        default:            return kLinkedToOther;          // 0
    }
}
```

`kLinkedTo*` values (`GliderPRO/Headers/GliderDefines.h:610-614`):

| Constant | Value |
|---|---:|
| `kLinkedToOther` | 0 |
| `kLinkedToLeftMailbox` | 1 |
| `kLinkedToRightMailbox` | 2 |
| `kLinkedToCeilingDuct` | 3 |
| `kLinkedToFloorDuct` | 4 |

**`kLinkedToFloorDuct` (4) is never produced by `WhatAreWeLinkedTo`** — a
`kFloorTrans` destination falls into `default` and yields `kLinkedToOther`. And in
`ReadyGliderFromTransit`, `case kLinkedToFloorDuct: break;` is an empty case
(`GliderPRO/Sources/Transit.c:138`). So the constant is vestigial: a link *to* a
floor transporter arrives as a plain fade-in, not as a "rise out of the floor duct"
animation. A Go port must not implement the missing animation.

### 5.5 Empirical link audit: all links in all 22 houses

`verify_links.py` walks every non-tombstone room of every house, reads
`where` (int16 at objBase+8) and `who` (uint8 at objBase+10) for all 14 link-bearing
object types, applies `ExtractFloorSuite` for the house's version, resolves with
`GetRoomNumber`, and round-trips through `MergeFloorSuite`.

| House | link-bearing objects | resolved | `where == -1` (unlinked) | did not resolve |
|---|---:|---:|---:|---:|
| Art Museum | 61 | 53 | 8 | 0 |
| CD Demo House | 101 | 94 | 7 | 0 |
| California or Bust! | 0 | 0 | 0 | 0 |
| Castle o' the Air | 35 | 23 | 0 | **12** |
| Davis Station | 24 | 17 | 7 | 0 |
| Demo House | 2 | 2 | 0 | 0 |
| Empty House | 0 | 0 | 0 | 0 |
| Fun House | 91 | 83 | 8 | 0 |
| Grand Prix | 152 | 127 | 25 | 0 |
| ImagineHouse PRO II | 137 | 99 | 37 | **1** |
| In The Mirror | 75 | 68 | 7 | 0 |
| Land of Illusion | 359 | 310 | 1 | **48** |
| Leviathan | 254 | 168 | 63 | **23** |
| Metropolis | 92 | 74 | 18 | 0 |
| Nemo's Market | 36 | 27 | 9 | 0 |
| Rainbow's End | 174 | 138 | 0 | **36** |
| Sampler | 4 | 2 | 2 | 0 |
| Slumberland | 302 | 233 | 0 | **69** |
| SpacePods | 282 | 279 | 3 | 0 |
| Teddy World | 377 | 297 | 80 | 0 |
| The Asylum Pro | 141 | 120 | 21 | 0 |
| Titanic | 97 | 71 | 26 | 0 |
| **Total** | **2796** | **2285** | **322** | **189** |

`audit_bad.py` classifies the 189 non-resolving links using C truncating
division/modulo:

| Class | Count | Houses |
|---|---:|---|
| `where == -100`, `who == 255` | **165** | Castle o' the Air 12, Land of Illusion 48, Rainbow's End 36, Slumberland 69 |
| resolves only under the **v1** rule | **24** | ImagineHouse PRO II 1, Leviathan 23 |
| **truly dangling** | **0** | — |

**Class 1: `where == -100` is a second spelling of "unlinked".** Under the v2 arm with
C semantics: `suite = -100/100 = -1` (which is exactly `kRoomIsEmpty`) and
`floor = (-100 % 100) - 8 = 0 - 8 = -8`. `GetRoomNumber(-8, -1)` finds no live room
(verified: **zero** rooms with `suite == kRoomIsEmpty` exist anywhere in the 22-house
corpus, so there is nothing for it to match), so `GetRoomLinked`
(`GliderPRO/Sources/Objects.c:126-173`) returns `kRoomIsEmpty`. `who == 255` also maps
to -1 through `GetObjectLinked` (`GliderPRO/Sources/Objects.c:177-215`). So these
behave identically to `where == -1`.

Arithmetically the value is exactly `MergeFloorSuite(floor = 0, suite = -1)`
= `(-1)*100 + 0` = `-100`, and `(0, kRoomIsEmpty)` is precisely what
`GetRoomFloorSuite` writes on its failure path (`GliderPRO/Sources/Room.c:718-722`).
But the shipped `DoLink` *does* guard on that return value
(`if (GetRoomFloorSuite(thisRoomNumber, &floor, &suite))`,
`GliderPRO/Sources/Link.c:275`), so the shipped editor cannot produce `-100`; some
earlier editor build did. **(the provenance is unverified — only the value and its
runtime effect are confirmed)** Regardless, **a Go port must treat `where == -100` as
unlinked**, which it will do automatically if it implements the codec and
`GetRoomNumber` faithfully.

**Class 2: 24 links are stale v1 encodings in v2 houses** — links that were authored
before the format change and never passed through `ConvertHouseVer1To2` (because the
house header already said 0x0200). Concrete observed examples:

```
ImagineHouse PRO II   room  48 obj  9  kMailboxRt  where= 971  who=  1
    v2 -> (floor 63, suite  9) = MISSING      v1 -> (floor 1, suite 71) = room 26
Leviathan             room   9 obj  6  kTrigger    where= 960  who= 11
    v2 -> (floor 52, suite  9) = MISSING      v1 -> (floor 1, suite 60) = room 9
Leviathan             room   9 obj  7  kTrigger    where= 960  who= 11   (same)
Leviathan             room   9 obj  8  kTrigger    where= 960  who= 11   (same)
Leviathan             room   9 obj  9  kTrigger    where= 960  who= 11   (same)
Leviathan             room   9 obj 12  kTrigger    where= 960  who= 11   (same)
Leviathan             room   9 obj 13  kTrigger    where= 960  who= 11   (same)
Leviathan             room   9 obj 14  kTrigger    where= 960  who= 11   (same)
    ... 23 in Leviathan in total: where=960 x15 (all in room 9), 879 x4 (room 425),
        and 953, 554, 1479, 1375 once each
```

Since the house version is 0x0200, the shipped game decodes these with the v2 arm and
gets nothing — so **in the original game these links are dead**, and the intended
targets are never reached. A Go port must reproduce the dead behaviour, i.e. decode
strictly by the house's declared version. Adding a v1 fallback would make these
objects *work*, changing the game.

**Class 3 is empty: no shipped house has a link that resolves to a nonexistent room
under its own version's rules other than the two classes above.** That is strong
evidence the codec as documented is exactly right.

### 5.6 What may be a link *target* (the editor's rules)

`UpdateLinkControl` in the editor enumerates precisely which object types may be
selected as the destination of each link kind. This is a useful cross-check on the
runtime, and a Go port's editor (if any) must reproduce it.

`linkType` values (`GliderPRO/Headers/GliderDefines.h:463-465`): `kSwitchLinkOnly` 3,
`kTriggerLinkOnly` 4, `kTransportLinkOnly` 5. Those three are the only `*LinkOnly`
constants that exist; the header defines no `kNoLinkSelected`/`kNewLinkSelected` and no
values 0-2 in this family, so 0 is simply "no link mode".

- **`kSwitchLinkOnly` (3)** — a switch may target: the 11 blowers (`kFloorVent`,
  `kCeilingVent`, `kFloorBlower`, `kCeilingBlower`, `kLeftFan`, `kRightFan`,
  `kSewerGrate`, `kInvisBlower`, `kGrecoVent`, `kSewerBlower`, `kLiftArea`); 10
  bonuses (`kRedClock`, `kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`,
  `kBattery`, `kBands`, `kFoil`, `kInvisBonus`, `kHelium`); `kDeluxeTrans`; the 8
  lights (`kCeilingLight`, `kLightBulb`, `kTableLamp`, `kHipLamp`, `kDecoLamp`,
  `kFlourescent`, `kTrackLight`, `kInvisLight`); the 9 appliances (`kShredder`,
  `kToaster`, `kMacPlus`, `kGuitar`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`,
  `kMicrowave`); `kStereo`; the 8 enemies (`kBalloon`, `kCopterLf`, `kCopterRt`,
  `kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish`).
- **`kTriggerLinkOnly` (4)** — a trigger may target: `kGreaseRt`, `kGreaseLf`,
  `kToaster`, `kGuitar`, `kCoffee`, `kOutlet`, `kBalloon`, `kCopterLf`, `kCopterRt`,
  `kDartLf`, `kDartRt`, `kDrip`, `kFish`, plus the 6 switches (`kLightSwitch`,
  `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch`, `kInvisSwitch`)
  **only when `linkRoom == thisRoomNumber`** — a trigger can chain to a switch only
  in its own room.
- **`kTransportLinkOnly` (5)** — a transport may target: `kMailboxLf`, `kMailboxRt`,
  `kCeilingTrans`, `kInvisTrans`, `kDeluxeTrans`, `kInvisLight`, `kOzma`, `kMirror`,
  `kFireplace`, `kWallWindow`, `kCalendar`, `kBulletin`, `kCloud`.

(`GliderPRO/Sources/Link.c:57-210`)

Note the transport target list includes `kInvisLight`, `kOzma`, `kMirror`,
`kFireplace`, `kWallWindow`, `kCalendar`, `kBulletin`, `kCloud` — objects that are not
themselves transports. Arriving at one of those goes through `WhatAreWeLinkedTo`'s
`default:` arm and produces `kLinkedToOther`, i.e. a plain fade-in at the destination
object's rect. **`kFloorTrans` is not in the transport target list**, which is
consistent with `kLinkedToFloorDuct` being unreachable (§5.4) — and yet the Sampler
ships a `kFloorTrans` whose target is a `kCeilingTrans` (§2.6), the legal direction.

`DoLink`/`DoUnlink`:

```c
// Link.c:270  DoLink (abridged)
if (GetRoomFloorSuite(thisRoomNumber, &floor, &suite))
{   floor += kNumUndergroundFloors;
    if (linkerIsSwitch)  { ...objects[linkObject].data.e.where = MergeFloorSuite(floor, suite);
                              ...data.e.who = objActive; }
    else                 { ...objects[linkObject].data.d.where = MergeFloorSuite(floor, suite);
                              ...data.d.who = objActive; } }
// Link.c:325  DoUnlink (abridged)
if (linkerIsSwitch) { ...data.e.where = -1; ...data.e.who = 255; }
else                { ...data.d.where = -1; ...data.d.who = 255; }
// Link.c:366  HandleLinkClick tail
if (thisRoomNumber == linkRoom) CopyThisRoomToRoom();
GenerateRetroLinks();
```

Note the direction: linking is performed *from the destination*. `linkRoom`/
`linkObject` identify the object being *edited* (the transport/switch), and
`thisRoomNumber`/`objActive` identify the object currently selected — which becomes
the target. `OpenLinkWindow` initialises `linkRoom = -1; linkObject = 255;`
(`GliderPRO/Sources/Link.c:250-251`). The link windoid is 129×30 with controls
`kLinkControlID` 130 and `kUnlinkControlID` 131
(`GliderPRO/Sources/Link.c:19-20`, `:212`).
---

## 6. The play-session lifecycle

### 6.1 Entry points

There are exactly four ways a game starts, all funnelling into `NewGame(short mode)`:

| Menu item | Sets | Calls |
|---|---|---|
| `iNewGame` | `twoPlayerGame = false; resumedSavedGame = false;` | `NewGame(kNewGameMode)` |
| `iTwoPlayer` | `twoPlayerGame = true; resumedSavedGame = false;` | `NewGame(kNewGameMode)` |
| `iOpenSavedGame` | `resumedSavedGame = true; HeyYourPissingAHighScore();` | `if (OpenSavedGame()) { twoPlayerGame = false; NewGame(kResumeGameMode); }` |
| idle-splash timeout | `demoHouseIndex` swap, `demoGoing = true` | `NewGame(kNewGameMode)` via `DoDemoGame()` |

(`GliderPRO/Sources/Menu.c:302-340`, `GliderPRO/Sources/Play.c:282-305`)

`kResumeGameMode` = **0**, `kNewGameMode` = **1**
(`GliderPRO/Headers/GliderDefines.h:616-617`). Note the ordering: the *resume* mode is
the zero value, so a Go port must not treat a zero-valued mode parameter as "new game".
`OpenSavedGame()` returns `false` unconditionally in 1.0.4 (§15), so
**`kResumeGameMode` is unreachable in the shipped build**.

All three game-menu items are enabled only when
`(!noRoomAtAll) && (houseOpen) && (numberRooms > 0)` (`GliderPRO/Sources/Menu.c`
`UpdateMenus`).

`DoDemoGame` (`GliderPRO/Sources/Play.c:282-305`) saves `thisHouseIndex`, opens the
demo house, sets `demoGoing = true`, calls `NewGame(kNewGameMode)`, then reopens the
previous house. Input during a demo comes from `GetDemoInput` replaying `demoData[]`
indexed by `demoIndex` instead of `GetKeys` (`GliderPRO/Sources/Play.c:472-479`).

### 6.2 `NewGame` — the exact setup order

```
1.  NewGame(mode):
2.    AdjustScoreboardHeight()
3.    gameOver = false
4.    theMode = kPlayMode
5.    if isPlayMusicGame:
6.       if !isMusicOn: err = StartMusic(); on error YellowAlert(kYellowNoMusic), failedMusic = true
7.       SetMusicalMode(kPlayGameScoreMode)
8.    else if isMusicOn: StopTheMusic()
9.    if mode != kResumeGameMode: SetObjectsToDefaults()        // <== the state reset
10.   HideCursor()
11.   if mode == kResumeGameMode: SetHouseToSavedRoom()
12.   else if mode == kNewGameMode: SetHouseToFirstRoom()
13.   DetermineRoomOpenings()
14.   NilSavedMaps()
15.   gameFrame = 0; numBands = 0; demoIndex = 0; saidFollow = 0
16.   otherPlayerEscaped = kNoOneEscaped; onePlayerLeft = false; playerSuicide = false
17.   if twoPlayerGame:
18.      InitGlider(&theGlider,  kNewGameMode)                  // NOTE: forced to kNewGameMode
19.      InitGlider(&theGlider2, kNewGameMode)
20.      load kGliderPictID  into glidSrcMap
21.      load kGlider2PictID into glid2SrcMap
22.   else:
23.      InitGlider(&theGlider, mode)
24.      load kGliderPictID     into glidSrcMap
25.      load kGliderFoilPictID into glid2SrcMap                // foil sprite reuses glid2SrcMap
26.   SetPort(mainWindow); paint the bottom 20 pixels of thisMac.screen black
27.   [COMPILEQT] if hasQT and hasMovie: SetMovieGWorld(theMovie, mainWindow, nil)
28.   SetPort(workSrcMap); PaintRect(&workSrcRect)              // clear the offscreen to black
29.   DrawLocale()                                              // draw the whole 3x3 neighbourhood
30.   RefreshScoreboard(kNormalTitleMode)
31.   if mode == kNewGameMode:   BringUpBanner();       DumpScreenOn(&justRoomsRect)
32.   elif mode == kResumeGameMode: DisplayStarsRemaining(); DumpScreenOn(&justRoomsRect)
33.   else:                                            DumpScreenOn(&justRoomsRect)
34.   InitGarbageRects()
35.   StartGliderFadingIn(&theGlider)
36.   if twoPlayerGame:
37.      StartGliderFadingIn(&theGlider2)
38.      TagGliderIdle(&theGlider2)                             // player 2 waits 30 frames
39.      theGlider2.dontDraw = true
40.   InitTelephone()
41.   wasPlayMusicPref = isPlayMusicGame
42.   freeBytes = MaxMem(&growBytes)
43.   [COMPILEQT] if hasQT and hasMovie and tvInRoom: SetMovieActive(theMovie, true);
44.                if tvOn: StartMovie(theMovie); MoviesTask(theMovie, 0)
45.   playing = true
46.   PlayGame()                                                // <=== blocks until the game ends
47.   isPlayMusicGame = wasPlayMusicPref
48.   ZeroMirrorRegion()
49.   [COMPILEQT] if hasQT and hasMovie and tvInRoom: tvInRoom = false; StopMovie; SetMovieActive(false)
50.   twoPlayerGame = false
51.   theMode = kSplashMode
52.   InitCursor()
53.   restore idle music (StartMusic + SetMusicalMode(kPlayWholeScoreMode) or StopTheMusic)
54.   NilSavedMaps()
55.   SetPortWindowPort(mainWindow)
56.   BlackenScoreboard()
57.   UpdateMenus(false)
58.   if !gameOver:                                             // i.e. the player quit, not lost/won
59.      InvalWindowRect(mainWindow, &mainWindowRect)
60.      SetGWorld(workSrcMap, nil); PaintRect(&workSrcRect)
61.      tempRect = 640x460 offset by (splashOriginH, splashOriginV)
62.      LoadScaledGraphic(kSplash8BitPICT, &tempRect)
63.   WaitCommandQReleased()
64.   demoGoing = false
65.   incrementModeTime = TickCount() + kIdleSplashTicks
```
(`GliderPRO/Sources/Play.c:74-278`)

Load-bearing details:

- **Line 9**: `SetObjectsToDefaults()` runs for `kNewGameMode` *and for any mode that
  is not `kResumeGameMode`*. Since only 0 and 1 are ever passed, this means "new game
  resets, resume does not".
- **Lines 11-12**: for any mode other than 0 or 1, **neither** branch runs, so
  `thisRoomNumber` keeps whatever value it had from house loading. The same
  fall-through exists in `WhereDoesGliderBegin` (§7.1). Unreachable in the shipped
  build, but a Go port that adds a third mode must handle it.
- **Line 18**: in a two-player game, `InitGlider` is called with `kNewGameMode` for
  *both* gliders regardless of `mode`. That is why `iOpenSavedGame` forces
  `twoPlayerGame = false` before calling `NewGame(kResumeGameMode)`.
- **Line 25**: the foil sprite sheet is loaded into `glid2SrcMap`, the same GWorld used
  for player 2's glider. **A single-player game cannot have a second glider and a
  two-player game cannot have foil sprites.** That is a hard architectural constraint,
  not an accident, and it is why `HandleRewards` doubles the foil *count* in two-player
  mode instead of drawing foil (see §11.3).
- **Line 29**: `DrawLocale()` renders all nine rooms into `backSrcMap` and then copies
  to `workSrcMap`; see §8.6.
- **Line 31 vs 32**: a new game shows the house banner; a resumed game shows only the
  star count. Both then blit.
- **Line 58**: `gameOver` distinguishes "the player quit (Cmd-Q, or Esc→quit)" from
  "the game ended". Only in the quit case does `NewGame` restore the splash screen
  itself; in the game-over cases `DoGameOver`/`DoDiedGameOver` already did
  `RedrawSplashScreen()`.

### 6.3 `InitGlider`

```
1.  InitGlider(thisGlider, mode):
2.    WhereDoesGliderBegin(&thisGlider->dest, mode)
3.    if mode == kResumeGameMode: numStarsRemaining = smallGame.wasStarsLeft
4.    elif mode == kNewGameMode:  numStarsRemaining = CountStarsInHouse()
5.    if mode == kResumeGameMode:
6.       theScore     = smallGame.score
7.       mortals      = smallGame.numGliders
8.       batteryTotal = smallGame.energy
9.       bandsTotal   = smallGame.bands
10.      foilTotal    = smallGame.foil
11.      thisGlider->mode   = smallGame.gliderState
12.      thisGlider->facing = smallGame.facing
13.      showFoil           = smallGame.showFoil
14.      switch thisGlider->mode:
15.         kGliderBurning: FlagGliderBurning(thisGlider)
16.         default:        FlagGliderNormal(thisGlider)
17.    else:                                  // kNewGameMode (and anything else)
18.      theScore = 0
19.      mortals  = kInitialGliders                  // 2
20.      if twoPlayerGame: mortals += kInitialGliders   // => 4
21.      batteryTotal = 0; bandsTotal = 0; foilTotal = 0
22.      thisGlider->mode   = kGliderNormal
23.      thisGlider->facing = kFaceRight
24.      thisGlider->src = thisGlider->mask = gliderSrc[0]
25.      showFoil = false
26.    QSetRect(&thisGlider->destShadow, 0, 0, kGliderWide, kShadowHigh)   // 48 x 9
27.    QOffsetRect(&thisGlider->destShadow, thisGlider->dest.left, kShadowTop)  // y = 306
28.    thisGlider->wholeShadow = thisGlider->destShadow
29.    thisGlider->hVel = 0; thisGlider->vVel = 0
30.    thisGlider->hDesiredVel = 0; thisGlider->vDesiredVel = 0
31.    thisGlider->tipped = false; thisGlider->sliding = false
32.    thisGlider->dontDraw = false
```
(`GliderPRO/Sources/Play.c:307-368`)

- `kInitialGliders` = 2 (`GliderPRO/Sources/Play.c:19`). A one-player game starts with
  `mortals = 2`; a two-player game with `mortals = 4`. **`mortals` is a single shared
  pool** — both players draw from the same counter (§10).
- `mortals` counts *spare* gliders, not total lives: the glider currently in play is
  not counted, so the actual number of deaths tolerated is `mortals + 1`. The loss
  test is `mortals < 0` after a decrement (§10.1), so a one-player game ends on the
  **third** death.
- **`InitGlider` is called twice in a two-player game (lines 18-19 of `NewGame`), so
  `mortals` is set to 4 twice, not to 8.** It is an assignment, not an accumulation.
- Note lines 26-28 use `thisGlider->dest.left` for the shadow's X but a hard-coded
  `kShadowTop` = 306 for its Y — the shadow is always on the floor plane, regardless
  of where the glider starts.
- Line 24 sets `src`/`mask` for the new-game path but not for the resume path (the
  `FlagGliderNormal`/`FlagGliderBurning` calls do it there).

### 6.4 `PlayGame` — the main loop

```
1.  PlayGame():
2.    while playing and not quitting:
3.      gameFrame++
4.      evenFrame = !evenFrame
5.      if doBackground:                        // "run in background" preference
6.         do { HandlePlayEvent(); } while (switchedOut)
7.      HandleTelephone()
8.      if twoPlayerGame:
9.         HandleDynamics()
10.        if !gameOver:
11.           GetInput(&theGlider); GetInput2(&theGlider2)      (or GetDemoInput)
12.           HandleInteraction()                               // both gliders
13.        HandleTriggers()
14.        HandleBands()
15.        if !gameOver: HandleGlider(&theGlider); HandleGlider(&theGlider2)
16.        if playing: [MoviesTask] RenderFrame(); HandleDynamicScoreboard()
17.      else:
18.         HandleDynamics()
19.        if !gameOver:
20.           GetInput(&theGlider) (or GetDemoInput)
21.           HandleInteraction()
22.        HandleTriggers()
23.        HandleBands()
24.        if !gameOver: HandleGlider(&theGlider)
25.        if playing: [MoviesTask] RenderFrame(); HandleDynamicScoreboard()
26.      if gameOver:
27.         countDown--
28.         if countDown <= 0:
29.            GetGWorld(&wasCPort, &wasWorld)
30.            HideGlider(&theGlider)
31.            RefreshScoreboard(kNormalTitleMode)
32.            [BUILD_ARCADE_VERSION: black out boardSrcMap, blit to mainWindow,
33.                redraw kScoreboardPictID at hOffset =
34.                (boardSrcRect.right >= 640) ? (RectWide(&boardSrcRect) - kMaxViewWidth)/2 : -576]
35.            if mortals < 0: DoDiedGameOver()        // LOSE
36.            else:           DoGameOver()            // WIN
37.            SetGWorld(wasCPort, wasWorld)
38.    [BUILD_ARCADE_VERSION: repeat the board blackout + scoreboard redraw]
```
(`GliderPRO/Sources/Play.c:430-600`)

Key observations:

- **Once `gameOver` is set, input, interaction, and glider FSM stop** (lines 10, 15,
  19, 24) but `HandleDynamics`, `HandleTriggers`, `HandleBands`, and rendering keep
  running. So the world keeps animating for `countDown` frames while the glider is
  frozen. `countDown` = `kNumCountDownFrames` = 16
  (`GliderPRO/Sources/GameOver.c:17`, set by `FlagGameOver`, §11.1).
- **The win/lose choice at lines 35-36 is made by inspecting `mortals`, not by a
  separate flag.** Both the star-collected win and the out-of-gliders loss set the
  same `gameOver` boolean; `mortals < 0` is the only discriminator. Consequence: if
  the player collects the last star **on the same frame** that would take `mortals`
  below zero, the loss animation wins. (In practice `OffAMortal` returns immediately
  when `gameOver` is already set — `GliderPRO/Sources/Player.c:1486` — so the ordering
  is: whichever sets `gameOver` first determines the outcome, and `mortals` is not
  decremented afterwards.)
- **Frame pacing lives inside `RenderFrame()`, not in `PlayGame`.** The tail of
  `RenderFrame` is:
  ```c
  while (TickCount() < nextFrame) { }          // Render.c:662-664 — busy-wait
  nextFrame = TickCount() + kTicksPerFrame;    // Render.c:665
  CopyRectsQD();
  numWork2Main = 0; numBack2Work = 0;
  ```
  with `kTicksPerFrame` = 2 (`GliderPRO/Headers/GliderDefines.h:533`) and a Mac tick
  of 1/60.15 s. **The target frame rate is therefore ~30 fps** (2 ticks). `nextFrame`
  is seeded in `InitGarbageRects` (`GliderPRO/Sources/Render.c:690`).
  Two consequences: (a) the deadline is recomputed *after* the wait, so a frame that
  overruns does not try to catch up — the clock drifts rather than accumulating debt;
  (b) since `RenderFrame` is only called when `playing` (line 16/25), the game-over
  countdown frames are also paced at 30 fps, so `countDown` = 16 is about 0.53 s.
  All the animation constants (`kFramesToBurn` 60 = 2 s, `kShredderCountdown` -68 ≈
  2.3 s, `TagGliderIdle`'s `hVel = 30` = 1 s, `kRingDelay` 90 = 3 s, `kRingSpread`
  25000 ≈ 14 minutes) should be read as multiples of 1/30 s.

`HandlePlayEvent` handles only two event classes (`GliderPRO/Sources/Play.c:387-426`):

```
updateEvt:  BeginUpdate; CopyBits(workSrcMap -> mainWindow, justRoomsRect, justRoomsRect,
                                  srcCopy, nil); RefreshScoreboard(kNormalTitleMode); EndUpdate
osEvt:      if ((theEvent.message & 0x01000000) != 0)     // resume
               switchedOut = false; ToggleMusicWhilePlaying(); HideCursor()
            else                                          // suspend
               InitCursor(); switchedOut = true; ToggleMusicWhilePlaying()
```

with `long sleep = 2;` passed to `WaitNextEvent`. The `0x01000000` bit is bit 24 of the
`osEvt` message = the resume flag.

### 6.5 `SetObjectsToDefaults` — the state reset

This is the *only* thing that resets per-room state, and it runs exactly once per new
game (`NewGame` line 9).

```
1.  SetObjectsToDefaults():
2.    lock thisHouse
3.    for i in 0 .. (*thisHouse)->nRooms - 1:
4.      (*thisHouse)->rooms[i].visited = false
5.      for j in 0 .. kMaxRoomObs-1:                       // 24
6.        switch rooms[i].objects[j].what:
7.          // ---- blowers (11 types) ----
8.          kFloorVent, kCeilingVent, kFloorBlower, kCeilingBlower, kLeftFan,
9.          kRightFan, kSewerGrate, kInvisBlower, kGrecoVent, kSewerBlower, kLiftArea:
10.              objects[j].data.a.state = objects[j].data.a.initial
11.         // ---- bonuses (14 types) ----
12.         kRedClock, kBlueClock, kYellowClock, kCuckoo, kPaper, kBattery, kBands,
13.         kGreaseRt, kGreaseLf, kFoil, kInvisBonus, kStar, kSparkle, kHelium:
14.             objects[j].data.c.state = objects[j].data.c.initial
15.         // ---- the deluxe transporter, which packs its state in a nibble ----
16.         kDeluxeTrans:
17.             initState = (objects[j].data.d.wide & 0xF0) >> 4
18.             objects[j].data.d.wide &= 0xF0
19.             objects[j].data.d.wide += initState
20.         // ---- lights (8 types) ----
21.         kCeilingLight, kLightBulb, kTableLamp, kHipLamp, kDecoLamp,
22.         kFlourescent, kTrackLight, kInvisLight:
23.             objects[j].data.f.state = objects[j].data.f.initial
24.         // ---- the stereo, whose state is the music preference ----
25.         kStereo:
26.             objects[j].data.g.state = isPlayMusicGame
27.         // ---- appliances (9 types) ----
28.         kShredder, kToaster, kMacPlus, kGuitar, kTV, kCoffee, kOutlet, kVCR, kMicrowave:
29.             objects[j].data.g.state = objects[j].data.g.initial
30.         // ---- enemies (8 types) ----
31.         kBalloon, kCopterLf, kCopterRt, kDartLf, kDartRt, kBall, kDrip, kFish:
32.             objects[j].data.h.state = objects[j].data.h.initial
33.   unlock thisHouse
```
(`GliderPRO/Sources/Play.c:603-706`)

- **Line 4 clears every `visited` flag** — this is what makes room-visit scoring
  (§8.5) start from zero and what makes `CountRoomsVisited()` (the high-score
  "level") meaningful.
- **The `kDeluxeTrans` case is the only non-uniform one.** `data.d.wide` is a `Byte`
  whose high nibble holds the *initial* state and low nibble the *current* state. Line
  18 clears the low nibble; line 19 copies the high nibble's value into it. So after
  the reset, `wide = (initial << 4) | initial`. `SetObjectState` toggles the low
  nibble (`GliderPRO/Sources/Objects.c:496`). Note the arithmetic is `+=`, not `|=` —
  equivalent here because the low nibble was just cleared.
- **`kStereo` ignores its `initial` and takes the *global music preference*** (line
  26). So the same house plays differently depending on a user setting.
- Note that `data.a.state`, `data.c.state`, `data.f.state`, `data.g.state`, and
  `data.h.state` are at *different byte offsets* within the object (§2.4): `a` at +9,
  `c` at +10, `f`/`g`/`h` at +11. Reading them all through one union arm would corrupt
  data.
- Object types not listed are left alone: all furniture, clutter, the 8 switches, the
  6 non-deluxe transports, `kInvisTrans`, `kMailboxLf/Rt`, `kFloorTrans`,
  `kCeilingTrans`, all the stairs/doors/windows, `kOzma`, `kMirror`, etc. Those have
  no mutable state.

### 6.6 What `SetObjectsToDefaults` implies for "opened doors" and "lit lights"

The question "where is room state saved?" has a single answer: **in the in-memory
house image, in the `visited` byte and the per-object `state` bytes.** Specifically:

| Player-visible state | Storage | Reset by |
|---|---|---|
| Room has been entered before | `rooms[r].visited` (offset +33) | `SetObjectsToDefaults` line 4 |
| A light is on/off | `objects[j].data.f.state` | line 23 (from `data.f.initial`) |
| A blower is on/off | `objects[j].data.a.state` | line 10 |
| A bonus has been taken | `objects[j].data.c.state` (false = taken) | line 14 |
| A star has been taken | `objects[j].data.c.state` | line 14 |
| An appliance (TV, toaster, shredder…) is on/off | `objects[j].data.g.state` | line 29 |
| An enemy is active | `objects[j].data.h.state` | line 32 |
| A deluxe transporter is enabled | low nibble of `objects[j].data.d.wide` | lines 17-19 |
| The stereo is on | `objects[j].data.g.state` | line 26 (from the music pref) |
| The *visual hole* left by a taken bonus | `savedMaps[]` GWorld swatch (§17) | `NilSavedMaps()` |

**There are no "opened doors" as persistent state.** Doors and windows in Glider PRO
are not objects with state; they are `kDoorInLf`/`kDoorInRt`/`kDoorExRt`/`kDoorExLf`
and `kWindowInLf`/etc. whose only role is to set `ignoreLeft`/`ignoreRight` on the
glider when it overlaps them, permitting a side exit through an otherwise closed wall
(`GliderPRO/Sources/Interactions.c` `HandleHotSpotCollision`). They open and close
purely as an animation driven by the glider's proximity; nothing is remembered.

And critically: `SetObjectState` writes into `(*thisHouse)->rooms[...]`, so **object
state changes are written into the house image and therefore survive room
transitions** — the whole point. They do *not* survive a new game (line 9 of
`NewGame`), and they are *not* written to disk unless the house is saved
(`WriteHouse` writes the whole image, including whatever the player did — which is
why `gameDirty` exists and why the editor prompts).

### 6.7 The telephone and chimes

Purely cosmetic, but it is part of the session lifecycle and it is gated on a house
flag.

```
#define kRingDelay      90
#define kRingSpread     25000
#define kRingBaseDelay  5000
#define kChimeDelay     180
typedef struct { short nextRing; short rings; short delay; } phoneType;
phoneType thePhone, theChimes;
```
(`GliderPRO/Sources/Play.c:20-31`, `:47`)

```
1.  InitTelephone():
2.    thePhone.nextRing = RandomInt(kRingSpread) + kRingBaseDelay   // 5000..29999 frames
3.    thePhone.rings    = RandomInt(3) + 3                          // 3..5 rings
4.    thePhone.delay    = kRingDelay                                // 90
5.    theChimes.nextRing = RandomInt(kChimeDelay) + 1               // 1..180
```
(`GliderPRO/Sources/Play.c:733-740`)

Both countdowns are **test-for-zero-then-decrement-in-the-else**, not
decrement-then-test; port them literally or the ring cadence drifts by a frame:

```
1.  HandleTelephone():
2.    if !phoneBitSet:                                    // house flags bit 1 CLEAR
3.       if thePhone.nextRing == 0:
4.          if thePhone.delay == 0:
5.             thePhone.delay = kRingDelay                 // re-arm BEFORE playing
6.             PlayPrioritySound(kPhoneRingSound, kPhoneRingPriority)
7.             thePhone.rings--
8.             if thePhone.rings == 0:                     // ring burst finished
9.                thePhone.nextRing = RandomInt(kRingSpread) + kRingBaseDelay
10.               thePhone.rings    = RandomInt(3) + 3     // NOTE: thePhone only --
11.                                                        // theChimes is NOT touched
12.         else: thePhone.delay--
13.      else: thePhone.nextRing--
14.   if numChimes > 0:                                    // chimes are independent of
15.      if theChimes.nextRing == 0:                       // phoneBitSet
16.         if RandomInt(2) == 0: PlayPrioritySound(kChime1Sound, kChime1Priority)
17.         else:                 PlayPrioritySound(kChime2Sound, kChime2Priority)
18.         delayTime = kChimeDelay / numChimes
19.         if delayTime < 2: delayTime = 2
20.         theChimes.nextRing = RandomInt(delayTime) + 1
21.      else: theChimes.nextRing--
```
(`GliderPRO/Sources/Play.c:744-789`)

Note that the ring-burst rearm at lines 9-10 is **inline, not a call to
`InitTelephone`** — an implementation that calls the initialiser would also reseed
`theChimes.nextRing`, desynchronising the chimes. Also note `theChimes.rings` and
`theChimes.delay` are never used: the chimes only ever use `nextRing`, and each strike
reseeds it, so more chime objects (`numChimes`) means proportionally faster chiming.

`StrikeChime()` (`GliderPRO/Sources/Play.c:793-796`) is the whole function body
`theChimes.nextRing = 0;` — the next frame therefore plays a chime immediately. It is
called by the clock objects.

Note **line 2's polarity**: `phoneBitSet` *suppresses* the phone. The 8 houses with
flags bit 1 set (§2.6 Test 2) are the *quiet* ones.

---

## 7. Where the player starts

### 7.1 The start room and the start point are independent

Two separate fields decide where the glider begins:

- `houseType.firstRoom` (file offset 862) — a **room index**.
- `houseType.initial` (file offset 12) — a `Point` in **room-local pixel
  coordinates**.

```c
void SetHouseToFirstRoom (void)                          // Play.c:370
{
    short  firstRoom;
    firstRoom = GetFirstRoomNumber();
    ForceThisRoom(firstRoom);
}

void SetHouseToSavedRoom (void)                          // Play.c:380
{
    ForceThisRoom(smallGame.roomNumber);
}
```

```c
void WhereDoesGliderBegin (Rect *theRect, short mode)    // House.c:224
{
    Point  initialPt;
    if (mode == kResumeGameMode)
        initialPt = smallGame.where;
    else if (mode == kNewGameMode)
        initialPt = (*thisHouse)->initial;
    QSetRect(theRect, 0, 0, kGliderWide, kGliderHigh);   // 48 x 20
    QOffsetRect(theRect, initialPt.h, initialPt.v);
}
```

**`initialPt` is left uninitialized if `mode` is neither 0 nor 1.** In the shipped
build no such call happens, but a Go port must not accidentally introduce a third
mode. Go would zero-initialise, giving `(0,0)`; the original would give stack
garbage.

The resulting rect is `48 × 20` with its **top-left** at `(initial.h, initial.v)`.
`initial` is `{v, h}` on disk (§2.2). Observed values: Sampler `(h=384, v=85)`,
Demo House `(h=49, v=107)`, ImagineHouse PRO II `(h=211, v=35)`, Titanic
`(h=39, v=97)`.

Note `WhereDoesGliderBegin` does **not** clamp the point to the room, and does not
check that the resulting rect is inside `[0,512) × [0,322)`. A house authored with
`initial.h = 500` starts the glider straddling the right wall.

`InitializeEmptyHouse` defaults `initial.h = 32; initial.v = 32;`
(`GliderPRO/Sources/House.c:126-127`) and `firstRoom = -1`
(`GliderPRO/Sources/House.c:120`).

### 7.2 The shadow's start position

`InitGlider` lines 26-28 set the shadow to `48 × 9` at
`(dest.left, kShadowTop = 306)`, i.e. always on the floor plane, independent of
`initial.v`. So a glider that starts in mid-air has its shadow directly below it on
the floor — correct behaviour, achieved by hard-coding rather than by projection.

`shadowVisible` is not set by `InitGlider`; it is set by `FlagGliderNormal`
(`shadowVisible = IsShadowVisible();`, `GliderPRO/Sources/Modes.c:363`) and by
`DrawLocale` (`GliderPRO/Sources/RoomGraphics.c:126`). Since `NewGame` calls
`DrawLocale()` at line 29 before `StartGliderFadingIn` at line 35, `shadowVisible` is
correct for the start room.

### 7.3 The opening banner

`BringUpBanner()` (`GliderPRO/Sources/Banner.c:171-203`) is called only for
`kNewGameMode`:

```
1.  BringUpBanner():
2.    DrawBanner()                               // the "Glider PRO" title art + house banner text
3.    if bannerStarCountOn: draw "N stars remaining"
4.    if demoGoing: WaitForInputEvent(4)         // 4 ticks in a demo
5.    else:         WaitForInputEvent(15)        // 15 ticks normally
```

`DrawBanner` composites PICTs 1993 / 1992 / 1991 / 1017 / 1018 and then
`DrawBannerMessage` renders `(*thisHouse)->banner` split on `\r`
(`GliderPRO/Sources/Banner.c:41-168`). `bannerStarCountOn` is the inverse of house
flags bit 2 (§2.2), so only Art Museum suppresses the star count.

`DisplayStarsRemaining()` (`GliderPRO/Sources/Banner.c:205-...`) draws just the star
count with `DelayTicks(60)` then `WaitForInputEvent(30)`. It is used for
`kResumeGameMode` at `NewGame` line 32 and after every star pickup
(`GliderPRO/Sources/Interactions.c:945`).

### 7.4 `CountStarsInHouse`

```
1.  CountStarsInHouse():
2.    count = 0
3.    lock thisHouse
4.    for i in 0 .. nRooms-1:
5.      if rooms[i].suite != kRoomIsEmpty:
6.         for j in 0 .. kMaxRoomObs-1:
7.            if rooms[i].objects[j].what == kStar: count++
8.    unlock thisHouse
9.    return count
```
(`GliderPRO/Sources/Banner.c:89-111`)

It counts *slots*, not *available* stars — `data.c.state` and `data.c.initial` are
ignored. Verified against the corpus in §2.6 Test 7. `Fun House` returns 0, making it
unwinnable (see the discussion there).

---

## 8. Room transitions

There are exactly **seven** ways the current room changes:

| # | Trigger | Function called | Wipe direction |
|---|---|---|---|
| 1 | Glider exits right | `MoveRoomToRoom(g, kToRight)` | `kToRight` |
| 2 | Glider exits left | `MoveRoomToRoom(g, kToLeft)` | `kToLeft` |
| 3 | Glider exits top (or finishes up-stairs) | `MoveRoomToRoom(g, kAbove)` | `kAbove` |
| 4 | Glider exits bottom (or finishes down-stairs) | `MoveRoomToRoom(g, kBelow)` | `kBelow` |
| 5 | Transporter fade-out completes | `TransportRoomToRoom(g)` | `kAbove` |
| 6 | Duct traversal completes | `MoveDuctToDuct(g)` | `kAbove` |
| 7 | Mailbox traversal completes | `MoveMailToMail(g)` | `kAbove` |

Direction constants (`GliderPRO/Headers/GliderDefines.h:210-215`) — note they are
**1-based**, so 0 is not a valid direction:

| Constant | Value |
|---|---:|
| `kAbove` | 1 |
| `kToRight` | 2 |
| `kBelow` | 3 |
| `kToLeft` | 4 |
| `kBottomCorner` | 5 |
| `kTopCorner` | 6 |

### 8.1 Where transitions are detected

`HandleInteraction()` (`GliderPRO/Sources/Interactions.c:1691`) calls
`CheckGliderInRoom(thisGlider)` for each glider every frame. That is the only place
edge-of-room transitions originate.

```
1.  CheckGliderInRoom(thisGlider):
2.    if thisGlider->mode not in {kGliderNormal, kGliderFaceLeft,
3.                                kGliderFaceRight, kGliderBurning}: return
4.    if thisGlider->dest.top < kCeilingLimit:                       // < 8
5.       if thisGlider->mode == kGliderBurning:
6.          thisGlider->wasMode = 0
7.          StartGliderFadingOut(thisGlider)
8.          PlayPrioritySound(kFadeOutSound, kFadeOutPriority)
9.       elif twoPlayerGame: CheckEscapeUpTwo(thisGlider)
10.      else:               CheckEscapeUp(thisGlider)
11.   elif thisGlider->dest.bottom > kFloorLimit:                    // > 312
12.      if thisGlider->mode == kGliderBurning: (same fade-out)
13.      elif twoPlayerGame: CheckEscapeDownTwo(thisGlider)
14.      else:               CheckEscapeDown(thisGlider)
15.   elif thisBackground == kRoof and thisGlider->dest.bottom > kRoofLimit:   // > 122
16.      CheckRoofCollision(thisGlider)
17.   // --- INDEPENDENT of the above chain ---
18.   if thisGlider->dest.left < leftThresh:
19.      if twoPlayerGame: CheckEscapeLeftTwo(thisGlider)
20.      else:             CheckEscapeLeft(thisGlider)
21.   elif thisGlider->dest.right > rightThresh:
22.      if twoPlayerGame: CheckEscapeRightTwo(thisGlider)
23.      else:             CheckEscapeRight(thisGlider)
```
(`GliderPRO/Sources/Interactions.c:689-752`)

**Line 17 starts a new `if`, not an `else if`.** So a glider in a corner can be
processed vertically *and* horizontally in the same frame. If the vertical test
already called `MoveRoomToRoom`, the room has already changed and `leftThresh` /
`rightThresh` have already been recomputed by `ReadyLevel` — so the horizontal test
runs against the **new** room's thresholds using the glider's **new** position (which
`OffsetGlider` has moved by ±322 vertically, leaving X unchanged). This can chain two
transitions in one frame. A Go port must keep the two tests independent and must
re-read the globals, not cache them.

**A burning glider cannot change rooms.** Lines 5-8 and 12 intercept it and kill it
instead. This is deliberate: you cannot carry fire between rooms.

### 8.2 The four escape checks (one-player)

#### 8.2.1 `CheckEscapeUp`

```
1.  CheckEscapeUp(thisGlider):
2.    if topOpen and thisGlider->dest.top < kNoCeilingLimit:        // < -10
3.       MoveRoomToRoom(thisGlider, kAbove); return
4.    if thisBackground == kDirt:
5.       leftIdx  = thisGlider->dest.left  >> 6                     // /64
6.       rightIdx = thisGlider->dest.right >> 6
7.       if leftIdx  < 0 or leftIdx  > 7: bail to the wall bounce
8.       if rightIdx < 0 or rightIdx > 7: bail to the wall bounce
9.       if thisTiles[leftIdx] in {5,6} and thisTiles[rightIdx] in {5,6}:
10.         if thisGlider->dest.top < kNoCeilingLimit:
11.            MoveRoomToRoom(thisGlider, kAbove); return
12.         // else keep rising through the tunnel
13.         return
14.   thisGlider->vVel = kCeilingLimit - thisGlider->dest.top       // bounce off the ceiling
```
(`GliderPRO/Sources/Interactions.c:244-281`)

The `kDirt` special case is the underground tunnel: tiles 5 and 6 of the dirt tile set
are the vertical shaft graphics, and the glider may pass up through the ceiling only
if **both** of its horizontal extremes are over a shaft tile. Tile index is
`x >> 6` (divide by `kTileWide` = 64), bounds-checked to `[0, 7]` on both ends. The
same structure appears in `CheckEscapeDown` with tiles `{2, 3}`.

The final line is the bounce: `vVel` is set to exactly the displacement needed to put
`dest.top` back at `kCeilingLimit`. It is an *absolute* correction, not a reflection.

#### 8.2.2 `CheckEscapeDown`

```
1.  CheckEscapeDown(thisGlider):
2.    if bottomOpen and thisGlider->dest.bottom > kNoFloorLimit:    // > 332
3.       MoveRoomToRoom(thisGlider, kBelow); return
4.    if thisBackground == kDirt:
5.       leftIdx, rightIdx as above, bounds-checked
6.       if thisTiles[leftIdx] in {2,3} and thisTiles[rightIdx] in {2,3}:
7.          if thisGlider->dest.bottom > kNoFloorLimit:
8.             MoveRoomToRoom(thisGlider, kBelow); return
9.          return
10.   if thisGlider->ignoreGround:                                  // a manhole / open grate
11.      if thisGlider->dest.bottom > kNoFloorLimit:
12.         MoveRoomToRoom(thisGlider, kBelow); return
13.      return
14.   thisGlider->vVel = kFloorLimit - thisGlider->dest.bottom
15.   StartGliderFadingOut(thisGlider)
16.   PlayPrioritySound(kFadeOutSound, kFadeOutPriority)
```
(`GliderPRO/Sources/Interactions.c:378-448`)

**Falling through a closed floor kills you** (lines 14-16), unlike hitting a closed
ceiling which merely bounces. `ignoreGround` (line 10) is set by manholes and open
grates via `HandleHotSpotCollision`, and is cleared unconditionally at the end of
every `HandleGlider` (`GliderPRO/Sources/Player.c:1436`), so it is a per-frame
permission.

#### 8.2.3 `CheckEscapeLeft` / `CheckEscapeRight`

```
1.  CheckEscapeLeft(thisGlider):
2.    if leftThresh == kLeftWallLimit:                    // the wall is CLOSED
3.       if thisGlider->ignoreLeft and thisGlider->dest.left < kNoLeftWallLimit:
4.          MoveRoomToRoom(thisGlider, kToLeft); return   // through a door/window
5.       if foilTotal > 0: PlayPrioritySound(kFoilHitSound, kFoilHitPriority)
6.       else:             PlayPrioritySound(kHitWallSound, kHitWallPriority)
7.       offset = kLeftWallLimit - thisGlider->dest.left
8.       thisGlider->hVel = -thisGlider->hVel + offset    // reflect + push out
9.       QOffsetRect(&thisGlider->dest, offset, 0)  (and destShadow)
10.   else:                                               // the wall is OPEN
11.      MoveRoomToRoom(thisGlider, kToLeft)
```
(`GliderPRO/Sources/Interactions.c:572-597`; the right-hand mirror at `:662-687`
uses `kRightWallLimit`, `kNoRightWallLimit`, `dest.right`, and
`offset = kRightWallLimit - thisGlider->dest.right`.)

Note the discriminator on line 2 is **`leftThresh == kLeftWallLimit`, not
`!leftOpen`.** That is why the `kDirt` inconsistency in §4.1 has narrow effect: the
transition logic reads the *threshold*, and only the door/window `ignoreLeft` path
and the editor read `leftOpen`. Line 8's reflection is `-hVel + offset` — a bounce
*plus* a positional correction folded into the velocity.

#### 8.2.4 `CheckRoofCollision` — the sloped roof

```
1.  CheckRoofCollision(thisGlider):
2.    offset = (thisGlider->dest.left + kHalfGliderWide) >> 6        // tile under the centre
3.    if offset < 0 or offset > 7 or thisGlider->sliding: return
4.    localX = (thisGlider->dest.left + 24) - (offset << 6)          // 0..63 within the tile
5.    switch thisTiles[offset]:
6.      1: died = (localX          > (250 - thisGlider->dest.bottom))
7.      2: died = (localX          > (186 - thisGlider->dest.bottom))
8.      5: died = ((64 - localX)   > (186 - thisGlider->dest.bottom))
9.      6: died = ((64 - localX)   > (250 - thisGlider->dest.bottom))
10.     default: died = true
11.   if died:
12.      thisGlider->vVel = kFloorLimit - thisGlider->dest.bottom
13.      StartGliderFadingOut(thisGlider)
14.      PlayPrioritySound(kFadeOutSound, kFadeOutPriority)
```
(`GliderPRO/Sources/Interactions.c:450-507`)

Tiles 1 and 6 are the shallow slopes (constant 250), tiles 2 and 5 the steep ones
(186); 1 and 2 face one way, 5 and 6 the other (note the `64 - localX` mirror). Any
other tile is fatal. `sliding` (line 3) is set when the glider is already sliding
along a roof, suppressing the check. `kRoofLimit` = 122 gates entry to this function
from `CheckGliderInRoom` line 15.

**This is not a room transition** — it is a death. It is documented here because it is
the only thing that happens at a `kRoof` room's "bottom", `kRoof` having
`DoesRoomHaveFloor() == true` (§3.4).

### 8.3 `MoveRoomToRoom` — the side/vertical transition

```
1.  MoveRoomToRoom(thisGlider, where):
2.    HandleRoomVisitation()                        // score + mark the OUTGOING room
3.    switch where:
4.      case kToRight:
5.        SetMusicalMode(kProdGameScoreMode)
6.        if twoPlayerGame:
7.           UndoGliderLimbo(&theGlider); UndoGliderLimbo(&theGlider2)
8.           InsureGliderFacingRight(&theGlider); InsureGliderFacingRight(&theGlider2)
9.        else:
10.          InsureGliderFacingRight(thisGlider)
11.       ForceThisRoom(localNumbers[kEastRoom])
12.       if twoPlayerGame:
13.          OffsetGlider(&theGlider,  kToLeft); OffsetGlider(&theGlider2, kToLeft)
14.       else:
15.          OffsetGlider(thisGlider, kToLeft)
16.       QSetRect(&enterRect, 0, 0, 48, 20)
17.       QOffsetRect(&enterRect, 0, kGliderStartsDown + (short)thisRoom->leftStart - 2)
18.       theGlider.enteredRect = enterRect  (and theGlider2 in 2P)
19.     case kToLeft:
20.       SetMusicalMode(kProdGameScoreMode)
21.       ... InsureGliderFacingLeft ...
22.       ForceThisRoom(localNumbers[kWestRoom])
23.       OffsetGlider(..., kToRight)
24.       QSetRect(&enterRect, 0, 0, 48, 20)
25.       QOffsetRect(&enterRect, kRoomWide - 48,
26.                              kGliderStartsDown + (short)thisRoom->rightStart - 2)
27.       ... enteredRect = enterRect ...
28.     case kAbove:
29.       SetMusicalMode(kKickGameScoreMode)
30.       ForceThisRoom(localNumbers[kNorthRoom])
31.       if !takingTheStairs:
32.          if twoPlayerGame:
33.             UndoGliderLimbo(&theGlider); UndoGliderLimbo(&theGlider2)
34.             OffsetGlider(&theGlider, kBelow);  theGlider.enteredRect  = theGlider.dest
35.             OffsetGlider(&theGlider2, kBelow); theGlider2.enteredRect = theGlider2.dest
36.          else:
37.             OffsetGlider(thisGlider, kBelow);  thisGlider->enteredRect = thisGlider->dest
38.       else:
39.          if twoPlayerGame:
40.             ReadyGliderForTripUpStairs(&theGlider)
41.             ReadyGliderForTripUpStairs(&theGlider2)
42.          else:
43.             ReadyGliderForTripUpStairs(thisGlider)
44.     case kBelow:
45.       SetMusicalMode(kKickGameScoreMode)
46.       ForceThisRoom(localNumbers[kSouthRoom])
47.       if !takingTheStairs: ... OffsetGlider(kAbove) ... enteredRect = dest ...
48.       else:                ... ReadyGliderForTripDownStairs(...) ...
49.   if twoPlayerGame and !onePlayerLeft:
50.      if firstPlayer == kPlayer1: TagGliderIdle(&theGlider2)
51.      else:                       TagGliderIdle(&theGlider)
52.   ReadyLevel()
53.   RefreshScoreboard(kNormalTitleMode)
54.   WipeScreenOn(where, &justRoomsRect)
55.   [COMPILEQT] RenderFrame()
56.               if hasQT and hasMovie and tvInRoom and tvOn:
57.                  GoToBeginningOfMovie(theMovie); StartMovie(theMovie)
```
(`GliderPRO/Sources/Transit.c:151-310`)

Every detail here matters:

- **Line 2 runs *before* `ForceThisRoom`**, so `HandleRoomVisitation` marks the room
  the glider is *leaving*. See §8.5.
- **Line 11 uses `localNumbers[kEastRoom]`, the pre-resolved neighbour index.** If
  there is no east room, that is `kRoomIsEmpty` = -1 and `ForceThisRoom(-1)` returns
  immediately without changing anything (`GliderPRO/Sources/Room.c:373`). The glider
  is then `OffsetGlider`ed by -512 anyway (line 15), teleporting it to the *left* edge
  of the *same* room, and `ReadyLevel` redraws. **This is the observable behaviour when
  a house has an open wall with nothing beyond it**: the glider wraps to the other
  side of the same room. A Go port must reproduce this rather than guarding against
  it. (Whether any shipped house exhibits it depends on authoring; the mechanism is
  unconditional.)
- **Lines 16-18 read `thisRoom->leftStart` *after* `ForceThisRoom`**, so the entry Y
  comes from the **destination** room's `leftStart`. The `enteredRect` is where the
  glider will respawn if it dies in this room (§10.2). For a right-exit, the entry
  point is `(0, 30 + newRoom.leftStart)`; for a left-exit,
  `(464, 30 + newRoom.rightStart)`. Note the *glider itself* is not moved to
  `enterRect` — `OffsetGlider` preserves its Y and shifts X by exactly one room width,
  so the glider slides in at whatever height it left at. `enteredRect` is only the
  respawn point.
- **Lines 34-37: for vertical transitions `enteredRect` is set to the glider's actual
  post-offset `dest`,** not to a `leftStart`-derived point. Vertical entry has no
  authored entry height.
- **Lines 38-43: the stairs path never calls `OffsetGlider`.** `ReadyGliderForTripUpStairs`
  positions the glider itself (§8.4).
- **Lines 49-51: after any two-player transition, the *other* player is put in the
  idle state for 30 frames**, giving the follower time to appear.
- **Line 52 `ReadyLevel()` is the full room-change teardown/setup:**
  ```c
  void ReadyLevel (void)                            // RoomGraphics.c:402
  {   NilSavedMaps();
  #ifdef COMPILEQT
      if ((thisMac.hasQT) && (hasMovie) && (tvInRoom))
      {   tvInRoom = false; tvWithMovieNumber = -1; StopMovie(theMovie); }
  #endif
      DetermineRoomOpenings();
      DrawLocale();
      InitGarbageRects(); }
  ```
  `NilSavedMaps()` discards all 24 background swatches — which is why a bonus you took
  in room A does not leave a hole when you come back (the room is redrawn from scratch
  with the object's `state` now false, so the object simply is not drawn). See §17.
- **Line 54 `WipeScreenOn(where, ...)` uses the *same* direction constant** as the
  transition, so the wipe animates in the direction of travel (§8.7).

### 8.4 Stairs — the "trip" mechanism

Stairs are the only transition with a bespoke entry animation, and `takingTheStairs`
is the flag that selects it.

```c
Boolean takingTheStairs;      // Transit.c:18  (declared alongside firstPlayer)
```

Lifecycle:

| Event | Where |
|---|---|
| Set `true` when the glider finishes climbing off the top of the stairs | `GliderPRO/Sources/Player.c:344` |
| Set `true` when the glider finishes descending off the stairs | `GliderPRO/Sources/Player.c:472` |
| Read by `MoveRoomToRoom` to choose the entry animation | `GliderPRO/Sources/Transit.c:225`, `:257` |
| Cleared at the end of `DrawLocale` | `GliderPRO/Sources/RoomGraphics.c:127` |

Since `MoveRoomToRoom` calls `ReadyLevel()` → `DrawLocale()` at line 52, and reads
`takingTheStairs` at lines 31/47 *before* that, the flag is consumed exactly once per
transition and self-clears. **A Go port must clear it in the equivalent of
`DrawLocale`, not in `MoveRoomToRoom`, or a subsequent non-stairs transition into a
room with stairs will animate wrongly.**

#### 8.4.1 Going up

```
1.  MoveGliderUpStairs(thisGlider):                     // Player.c:315
2.    #define kClimbStairsSpeed  -4
3.    thisGlider->src = thisGlider->mask =
4.        (thisGlider->facing == kFaceLeft) ? gliderSrc[2] : gliderSrc[1]
5.    whole.bottom = dest.bottom
6.    dest.top += kClimbStairsSpeed; dest.bottom += kClimbStairsSpeed   // rise 4 px
7.    whole.top = dest.top
8.    vNotClipped = thisGlider->dest.bottom - 29
9.    if vNotClipped < kGliderHigh:                                    // < 20; else do nothing
10.      if vNotClipped <= 0:
11.         thisGlider->dest.top  = thisGlider->dest.bottom           // collapse to nothing
12.         thisGlider->src.top   = thisGlider->src.bottom
13.         thisGlider->mask.top  = thisGlider->mask.bottom
14.         takingTheStairs = true
15.         if twoPlayerGame:
16.            if onePlayerLeft:
17.               MoveRoomToRoom(playerDead == kPlayer1 ? &theGlider2 : &theGlider, kAbove)
18.            elif otherPlayerEscaped == kPlayerEscapedUpStairs:
19.               otherPlayerEscaped = kNoOneEscaped
20.               MoveRoomToRoom(thisGlider, kAbove)
21.            else:
22.               otherPlayerEscaped = kPlayerEscapedUpStairs
23.               RefreshScoreboard(kEscapedTitleMode)
24.               FlagGliderInLimbo(thisGlider, true)
25.         else:
26.            MoveRoomToRoom(thisGlider, kAbove)
27.      else:                                                          // 0 < vNotClipped < 20
28.         dest.top = dest.bottom - vNotClipped
29.         src.top  = src.bottom  - vNotClipped
30.         mask.top = mask.bottom - vNotClipped
```
(`GliderPRO/Sources/Player.c:315-379`)

The magic number 29 on line 8 is the Y above which the glider is considered to have
gone through the ceiling on the stairs. `kClimbStairsSpeed` = -4 means 4 px per frame,
so the climb takes roughly `(startBottom - 29) / 4` frames.

**Note the two-level test on lines 9-10.** Clipping only happens once the glider is
within `kGliderHigh` (20) px of the cut-off; while `vNotClipped >= 20` the function
falls straight through and nothing but the 4 px rise happens. A port that clips
unconditionally will squash the sprite for the whole climb.

#### 8.4.2 Arriving from below

```
1.  ReadyGliderForTripUpStairs(thisGlider):             // Modes.c:519
2.    #define kVGliderAppearsComingUp  100
3.    if twoPlayerGame and onePlayerLeft and thisGlider->which == playerDead: return
4.    thisGlider->facing = kFaceLeft
5.    thisGlider->mode   = kGliderComingUp
6.    thisGlider->src = thisGlider->mask = gliderSrc[2]
7.    thisGlider->hVel = 0; thisGlider->vVel = 0
8.    thisGlider->hDesiredVel = 0; thisGlider->vDesiredVel = 0
9.    thisGlider->tipped = false
10.   rightClip = GetUpStairsRightEdge()                 // GLOBAL, not a glider field
11.   thisGlider->dest = thisGlider->src
12.   ZeroRectCorner(&thisGlider->dest)                  // move top-left to (0,0)
13.   QOffsetRect(&thisGlider->dest, rightClip, kVGliderAppearsComingUp)
14.   thisGlider->whole = thisGlider->dest
15.   thisGlider->destShadow.left  = thisGlider->dest.left
16.   thisGlider->destShadow.right = thisGlider->dest.right
17.   thisGlider->wholeShadow = thisGlider->destShadow
18.   FinishGliderUpStairs(thisGlider)                   // run one frame immediately
```
(`GliderPRO/Sources/Modes.c:519-546`)

`rightClip` and `leftClip` are **file-scope globals shared by both players**, not
per-glider fields; `FinishGliderUpStairs` reads the global `rightClip`
(`GliderPRO/Sources/Player.c:407`). In a two-player game
`ReadyGliderForTripUpStairs` runs for both gliders in the same room, so both compute
the same value and the sharing is harmless — but a Go port that makes it a glider
field is *not* a faithful translation and will diverge if the two gliders are ever in
different rooms.

`ReadyGliderForTripDownStairs` (`GliderPRO/Sources/Modes.c:550-577`) is the mirror:
`facing = kFaceRight`, `mode = kGliderComingDown`, `src = gliderSrc[0]`,
`leftClip = GetDownStairsLeftEdge()`, and
`QOffsetRect(&dest, leftClip - kGliderWide, kVGliderAppearsComingDown)` with
`kVGliderAppearsComingDown` = 100, then `FinishGliderDownStairs`.

The clip edges come from the destination room's *other* staircase:

```c
short GetUpStairsRightEdge (void)                        // ObjectRects.c:1135
{
    short  edge = kRoomWide;                             // 512 default
    for i in 0..kMaxRoomObs-1:                           // (*thisHouse)->rooms[thisRoomNumber]
        if what == kDownStairs:
            edge = data.d.topLeft.h + srcRects[kDownStairs].right - 1
            break                                        // FIRST match wins
    return edge;
}
short GetDownStairsLeftEdge (void)                       // ObjectRects.c:1163
{
    short  edge = 0;                                     // 0 default
    for i in 0..kMaxRoomObs-1:
        if what == kUpStairs:
            edge = data.d.topLeft.h + 1
            break                                        // FIRST match wins
    return edge;
}
```

So the glider walking *up* out of the floor emerges at the right edge of the
destination room's *down*-staircase graphic; if the destination has no down-staircase,
it emerges at x = 512 (the right wall) and walks left. Symmetrically, a glider
descending emerges at the left edge of the destination's up-staircase, or x = 0.
Both loops `break` on the first match, so the **first** matching object in the room's
`objects[]` array wins — a room with two down-staircases uses the lower-indexed one.

#### 8.4.3 The arrival walk

```
1.  FinishGliderUpStairs(thisGlider):                    // Player.c:383
2.    #define kVClimbStairsSpeed  -4
3.    #define kHClimbStairsSpeed  -4
4.    QOffsetRect(&thisGlider->dest, kHClimbStairsSpeed, kVClimbStairsSpeed)   // up-left 4,4
5.    (same for destShadow horizontally)
6.    hNotClipped = rightClip - thisGlider->dest.left        // GLOBAL rightClip
7.    if hNotClipped < kGliderWide:                      // still partly behind the stairs
8.       clip src/mask/dest right edges to hNotClipped
9.    else:
10.      if thisGlider->frame == kWasBurning: FlagGliderBurning(thisGlider)
11.      else:                                FlagGliderNormal(thisGlider)
12.      thisGlider->hVel = kHClimbStairsSpeed; thisGlider->hDesiredVel = kHClimbStairsSpeed
13.      thisGlider->vVel = kVClimbStairsSpeed; thisGlider->vDesiredVel = kVClimbStairsSpeed
14.      thisGlider->enteredRect = thisGlider->dest
```
(`GliderPRO/Sources/Player.c:383-427`; `FinishGliderDownStairs` at `:512-556` uses
`hNotClipped = thisGlider->dest.right - leftClip` — again the global — with
`kHDropStairsSpeed` = `kVDropStairsSpeed` = 4, so `hVel = vVel = +4`.)

The glider emerges diagonally at 4 px/frame in both axes, clipped by the staircase
graphic until fully clear, then hands off to the normal FSM with that diagonal
velocity intact. `enteredRect` (line 14) is set here, so a death in the new room
respawns at the top of the stairs.

`kWasBurning` = 2 (`GliderPRO/Headers/GliderDefines.h`); `StartGliderGoingUpStairs`
seeds `frame = kWasBurning` if the glider was on fire (`GliderPRO/Sources/Modes.c:127`),
so **burning survives a stairs transition** even though it cannot survive a
ceiling/floor exit (§8.1). That is a real asymmetry.

`StartGliderGoingDownStairs` additionally sets the global
`rightClip = GetUpStairsRightEdge();`
(`GliderPRO/Sources/Modes.c:150`) — used by
`MoveGliderDownStairs`'s `hNotClipped = rightClip - dest.left`
(`GliderPRO/Sources/Player.c:463`), i.e. the *departure* clip on the way down is the
current room's own up-stairs edge.

### 8.5 `HandleRoomVisitation` — room-visit scoring and the `visited` flag

```c
void HandleRoomVisitation (void)                         // Transit.c:430
{
    houseType   *thisHousePtr;
    char        wasState;

    if (!thisRoom->visited)
    {
        wasState = HGetState((Handle)thisHouse);
        HLock((Handle)thisHouse);
        thisHousePtr = *thisHouse;
        thisHousePtr->rooms[localNumbers[kCentralRoom]].visited = true;
        HSetState((Handle)thisHouse, wasState);
        theScore += kRoomVisitScore;
        thisRoom->visited = true;
    }
}
```

`kRoomVisitScore` = 100 (`GliderPRO/Headers/GliderDefines.h`).

- Called at the top of **all four** transition entry points: `MoveRoomToRoom`
  (`GliderPRO/Sources/Transit.c:155`), `TransportRoomToRoom` (`:319`),
  `MoveDuctToDuct` (`:357`), `MoveMailToMail` (`:396`).
- It marks the room being **left**, not the room being entered. The very first room a
  new game starts in is therefore marked (and scored) when the player *leaves* it, and
  a player who never leaves the start room scores no visit points at all.
- It writes into `(*thisHouse)->rooms[localNumbers[kCentralRoom]]` — the *house*, so it
  persists — *and* into `thisRoom->visited`, so the local copy agrees for the rest of
  this frame. (The local copy is about to be overwritten by `ForceThisRoom` anyway.)
- 100 points per room, once per room per game. Combined with `CountRoomsVisited()`
  driving the high-score "level" (§13), exploration is the main scoring channel:
  observed high scores in the shipped houses are 47000 with level 108 for
  ImagineHouse PRO II, 7400 with level 13 for Demo House (§2.6).

### 8.6 `DrawLocale` — rebuilding the 3×3 view

```
1.  DrawLocale():
2.    ZeroFlamesAndTheLike()             // numFlames/tikis/coals/pendulums/grease/stars/shreds/chimes = 0
3.    ZeroDinahs()                       // clear the dynamic-object list
4.    KillAllBands()
5.    ZeroMirrorRegion()
6.    ZeroTriggers()
7.    numTempManholes = 0
8.    FlushAnyTriggerPlaying(); DumpTriggerSound()
9.    tvInRoom = false; tvWithMovieNumber = -1
10.   roomV = (*thisHouse)->rooms[thisRoomNumber].floor
11.   for i in 0..8:
12.      localNumbers[i] = GetNeighborRoomNumber(i)
13.      isStructure[i]  = IsRoomAStructure(localNumbers[i])
14.   ListAllLocalObjects()              // build masterObjects[] and the hot-spot list
15.   GetGWorld(...); SetGWorld(backSrcMap, nil); PaintRect(&backSrcRect)
      // DrawRoomBackground(short who, short where, short elevation)
      //   who = absolute room number, where = local 0..8 index, elevation = floor
16.   if numNeighbors > 3:
17.      for w in {kNorthWestRoom, kNorthEastRoom, kNorthRoom}:      // in that order
18.         numLights = GetNumberOfLights(localNumbers[w])
19.         DrawRoomBackground(localNumbers[w], w, roomV + 1)
20.         DrawARoomsObjects(w, false)                              // NO DrawLighting here
21.      for w in {kSouthWestRoom, kSouthEastRoom, kSouthRoom}:  ... roomV - 1 ...
22.   if numNeighbors > 1:
23.      for w in {kWestRoom, kEastRoom}:
24.         numLights = GetNumberOfLights(localNumbers[w])
25.         DrawRoomBackground(localNumbers[w], w, roomV)
            DrawARoomsObjects(w, false); DrawLighting()
26.   // always:
27.   numLights = GetNumberOfLights(localNumbers[kCentralRoom])
28.   DrawRoomBackground(localNumbers[kCentralRoom], kCentralRoom, roomV)
29.   DrawARoomsObjects(kCentralRoom, false); DrawLighting()
30.   if numNeighbors > 3: DrawFloorSupport()
31.   RestoreWorkMap()                   // CopyBits backSrcMap -> workSrcMap over backSrcRect
32.   shadowVisible = IsShadowVisible()
33.   takingTheStairs = false            // <== the stairs flag self-clears here
```
(`GliderPRO/Sources/RoomGraphics.c:44-130`)

Notes:

- **Lines 2-9 are the "forget everything transient" step.** Every dynamic object,
  band, trigger, flame, pendulum, star animation, and shred is destroyed on every room
  change. Nothing animated survives a transition. This is the definitive answer to
  "what resets on room change".
- **Line 14 `ListAllLocalObjects()`** rebuilds `masterObjects[]` from the local rooms
  in a fixed order that depends on `numNeighbors`
  (`GliderPRO/Sources/Objects.c:300-347`): always `kCentralRoom`; then, if
  `numNeighbors > 1`, `kEastRoom` then `kWestRoom`; then, if `numNeighbors > 3`,
  `kNorthRoom`, `kNorthEastRoom`, `kSouthEastRoom`, `kSouthRoom`, `kSouthWestRoom`,
  `kNorthWestRoom` — **note this is not the `kCentralRoom..kNorthWestRoom` numeric
  order, and `masterObjects[]` indices are what `localLink` stores, so a port must
  reproduce this exact order.** It then runs an O(n²) pass resolving `localLink`: for
  each object whose `roomLink` *and* `objectLink` are both != -1, scan the whole list
  for a matching `(roomNum, objectNum)` and record its index
  (`GliderPRO/Sources/Objects.c:332-346`). The scan does not break, so with duplicates
  the *last* match wins. `localLink` is initialised to -1 when the entry is created
  (`GliderPRO/Sources/Objects.c:278`), **so a link whose target is outside the visible
  3×3 keeps `localLink = -1`,** which `FireTrigger` handles specially (§16.2).
- **`numLights` is a global that is reused as a loop temporary** (lines 18, 24, 27).
  After `DrawLocale` returns it holds the *central* room's light count, which is what
  `RedrawRoomLighting` expects. But during lines 16-25 it holds neighbours' counts.
  A Go port that makes it a local would need to set the global explicitly at the end.
- Line 32 sets `shadowVisible` from the central room.

### 8.7 `WipeScreenOn` — the transition animation

```
1.  WipeScreenOn(direction, theRect):
2.    #define kWipeRectThick  4
3.    wipeRect = *theRect
4.    switch direction:
5.      kAbove:   wipeRect.bottom = wipeRect.top + 4;   hOffset = 0;  vOffset = +4
6.                count = ((theRect->bottom - theRect->top) / 4) + 1
7.      kToRight: wipeRect.left = wipeRect.right - 4;   hOffset = -4; vOffset = 0
8.                count = workSrcRect.right / 4
9.      kBelow:   wipeRect.top = wipeRect.bottom - 4;   hOffset = 0;  vOffset = -4
10.               count = ((theRect->bottom - theRect->top) / 4) + 1
11.     kToLeft:  wipeRect.right = wipeRect.left + 4;   hOffset = +4; vOffset = 0
12.               count = workSrcRect.right / 4
13.   for i in 0 .. count-1:
14.     CopyBits(workSrcMap -> mainWindow, &wipeRect, &wipeRect, srcCopy,
15.              GetPortVisibleRegion(GetWindowPort(mainWindow), tempRgn))
16.     QOffsetRect(&wipeRect, hOffset, vOffset)
17.     clamp wipeRect.top/bottom into theRect
```
(`GliderPRO/Sources/Transitions.c:74-135`)

A 4-pixel-thick band sweeps across, copying the already-rendered new room from
`workSrcMap` to the screen. **The offsets are the opposite sign to the direction
name**: `kAbove` (moving to the room above) wipes downward (`vOffset = +4`), because
the new room slides in from the top. `kToRight` wipes leftward.

`PourScreenOn` (`GliderPRO/Sources/Transitions.c:18-70`) is the alternative
"chunky dissolve" (`kMaxColumnsWide` 96, `kChipHigh` 20, `kChipWide` 16,
`colWide = theRect->right / 16`, `rowTall = theRect->bottom / 20 + 1`, random column
order) — used for the initial screen-on, not for room changes.
`DumpScreenOn` (`GliderPRO/Sources/Transitions.c:139-144`) is a single `CopyBits` with
no animation, used by `NewGame` lines 31-33.

### 8.8 `OffsetGlider`

```c
void OffsetGlider (gliderPtr thisGlider, short where)    // Player.c:1443
{
    if ((twoPlayerGame) && (onePlayerLeft) && (thisGlider->which == playerDead))
        return;
    switch (where)
    {
        case kToRight:                                  // dest.left/right += kRoomWide
            dest += (+512, 0);  destShadow += (+512, 0);
            whole = dest;       wholeShadow = destShadow;
            break;
        case kToLeft:
            dest += (-512, 0);  destShadow += (-512, 0);
            whole = dest;       wholeShadow = destShadow;
            break;
        case kAbove:                                    // dest.top/bottom -= kTileHigh
            dest += (0, -322);                          // shadow untouched
            whole = dest;                               // wholeShadow NOT touched
            break;
        case kBelow:
            dest += (0, +322);
            whole = dest;                               // wholeShadow NOT touched
            break;
    }
}
```

- The four cases each write their own `whole`/`wholeShadow` inside the `switch`; there
  is no shared epilogue. In particular the vertical cases leave `wholeShadow` alone.
- **The horizontal cases offset the shadow; the vertical cases do not.** That is
  correct: the shadow lives on the floor plane at `kShadowTop` = 306 and must stay
  there when the room changes vertically, but must follow horizontally.
- `MoveRoomToRoom` calls it with the **opposite** direction to the travel direction
  (line 13/15: exiting right calls `OffsetGlider(kToLeft)`), because the glider is
  being wrapped from the right edge of the old room to the left edge of the new one.
- The dead-player guard on line 1 stops the permanently-dead player's corpse from
  being dragged along.

### 8.9 Transport, mailbox, and duct transitions

The three link-based transitions are implemented by **three byte-for-byte identical
functions** differing only in name:

```
1.  TransportRoomToRoom(thisGlider) / MoveDuctToDuct(thisGlider) / MoveMailToMail(thisGlider):
2.    SetMusicalMode(kKickGameScoreMode)
3.    HandleRoomVisitation()
4.    sameRoom = (transRoom == thisRoomNumber)
5.    if !sameRoom: ForceThisRoom(transRoom)
6.    if twoPlayerGame:
7.       UndoGliderLimbo(&theGlider); UndoGliderLimbo(&theGlider2)
8.       ReadyGliderFromTransit(&theGlider,  linkedToWhat)
9.       ReadyGliderFromTransit(&theGlider2, linkedToWhat)
10.   else:
11.      ReadyGliderFromTransit(thisGlider, linkedToWhat)
12.   if !sameRoom: ReadyLevel()
13.   RefreshScoreboard(kNormalTitleMode)
14.   if !sameRoom: WipeScreenOn(kAbove, &justRoomsRect)
15.   [COMPILEQT] RenderFrame()
16.               if hasQT and hasMovie and tvInRoom and tvOn:
17.                  GoToBeginningOfMovie(theMovie); StartMovie(theMovie)
```
(`GliderPRO/Sources/Transit.c:314-348`, `:352-387`, `:391-426`)

- **`sameRoom` (line 4) is the intra-room teleport case** — e.g. the Sampler's floor
  transporter linking to the ceiling transporter in the same room (§2.6). When
  `sameRoom`, no room load, no `ReadyLevel`, and **no wipe**: the glider just appears
  elsewhere. Note `HandleRoomVisitation` still runs (line 3), harmlessly (the room is
  already visited by then in the same-room case... unless it is not, in which case a
  same-room teleport scores 100 points, once).
- **The wipe direction is always `kAbove`** (line 14) for link transitions, regardless
  of geometry.
- `transRoom` and `linkedToWhat` were set when the transit *started*, by the
  `StartGlider*` routine (§8.10).

### 8.10 Starting a link transit

`StartGliderTransporting`, `StartGliderMailingIn`, `StartGliderDuctingDown`, and
`StartGliderDuctingUp` all share this prologue:

```
1.    PlayPrioritySound(kTransOutSound, kTransOutPriority)
2.    whoLinked = who->who                                    // hot-spot's master index
3.    transRoom  = masterObjects[whoLinked].roomLink
4.    objLinked  = masterObjects[whoLinked].objectLink
5.    linkedToWhat = WhatAreWeLinkedTo(transRoom, objLinked)
6.    GetObjectRect(&(*thisHouse)->rooms[transRoom].objects[objLinked], &transRect)
```
(`GliderPRO/Sources/Modes.c:155-177` mailing-in, `:213-242` ducting down,
`:246-275` ducting up, `:288-330` transporting. The three duct/transport routines also
run the `kGliderGoingFoil`/`kGliderLosingFoil` foil fix-up before reading the link;
`StartGliderMailingIn` does **not**.)

Then each sets its own sprite and clip:

| Routine | `mode` set | Extra setup |
|---|---|---|
| `StartGliderMailingIn` (`Modes.c:155`) | **none** — the caller sets `kGliderMailInLeft`/`Right` (`GliderPRO/Sources/Interactions.c:1446`, `:1489`) | `frame = 0; clip = *bounds; clip.top = bounds->bottom - RectTall(&dest);` |
| `StartGliderDuctingDown` (`Modes.c:213`) | `kGliderDuctingDown` | `frame = 0; clip.left = bounds->left + ((RectWide(bounds) - kGliderWide) / 2);` |
| `StartGliderDuctingUp` (`Modes.c:246`) | `kGliderDuctingUp` | same centring |
| `StartGliderTransporting` (`Modes.c:288`) | `kGliderTransporting` | resizes `dest`/`destShadow` to `kGliderWide`×`kGliderHigh`/`kShadowHigh`, then `frame = kLastFadeSequence - 1` = 15 and picks `gliderSrc[fadeInSequence[frame] (+ kLeftFadeOffset if facing left)]` |

**If `transRoom` is -1 (an unlinked or dangling transport), line 6 indexes
`(*thisHouse)->rooms[-1]`.** In the original that reads 348 bytes *before* the rooms
array — i.e. into the house header (specifically the tail of `highScores`, the
`savedGame` block, `hasGame`, `firstRoom`, `nRooms`). It does not crash on a Mac; it
produces a garbage `transRect`. Then `WhatAreWeLinkedTo(-1, who)` likewise reads
garbage and almost certainly returns `kLinkedToOther`. Finally
`TransportRoomToRoom` does `ForceThisRoom(-1)` which returns immediately, so the room
does not change, and `ReadyGliderFromTransit` centres the glider in the garbage
`transRect`.

**The actual guard is weaker than you would hope, and it is on `who`, not on
`where`.** `CreateActiveRects` creates the hot spot only when
`theObject.data.d.who != 255`:

```c
case kMailboxLf:
if (theObject.data.d.who != 255)
{   QSetRect(&bounds, -72, 0, 0, 40); QOffsetRect(&bounds, 30, 16);
    QOffsetRect(&bounds, theObject.data.d.topLeft.h, theObject.data.d.topLeft.v);
    hotSpotNumber = AddActiveRect(&bounds, kMailItLeft, who, true, false); }
break;
```
(`GliderPRO/Sources/ObjectRects.c:751-761`; identically for `kMailboxRt` at `:763`,
`kFloorTrans` at `:775`, `kCeilingTrans` at `:787`, `kInvisTrans` at `:871`, and
`kDeluxeTrans` at `:884`. Note `kFloorTrans`/`kCeilingTrans` create *duct* hot spots
(`kDuctItDown`/`kDuctItUp`); the two `kTransportIt` hot spots come from
`kInvisTrans`/`kDeluxeTrans`. All six link-transit hot spots are guarded.)

So `who == 255` ⇒ no hot spot ⇒ unenterable. That covers `where == -1` and the 165
`where == -100` links (all of which have `who == 255`, §5.5). It does **not** cover
the 24 stale-v1 links, which have real `who` values (1 and 11). For those:

- ImagineHouse PRO II room 48 object 9 is a `kMailboxRt` with `where = 971`,
  `who = 1`. A hot spot **is** created. `GetRoomLinked` returns `kRoomIsEmpty` = -1.
  When the glider enters, `StartGliderMailingIn` executes
  `GetObjectRect(&(*thisHouse)->rooms[-1].objects[1], &transRect)` — **a read 348
  bytes before the rooms array**, i.e. into `highScores.levels[]` /
  `savedGame` / `hasGame` / `firstRoom` / `nRooms`. On a Mac this silently returns a
  garbage rect. Then `WhatAreWeLinkedTo(-1, 1)` reads the same garbage and returns
  whatever `what` value happens to be there (almost certainly `kLinkedToOther`), and
  `MoveMailToMail` does `ForceThisRoom(-1)` which is a no-op, so the glider stays in
  room 48 and reappears somewhere arbitrary inside it.
- The Leviathan `kTrigger` cases (`where = 960`, `who = 11`) go through `ArmTrigger`,
  which stores `room = masterObjects[whoLinked].roomLink` = -1 and
  `object = masterObjects[whoLinked].objectLink` = -1, then reads
  `masterObjects[triggers[where].object].theObject.what` = `masterObjects[-1]`
  (`GliderPRO/Sources/Triggers.c:50`). If the trigger fires, `FireTrigger`'s
  `localLink == -1` branch switches on
  `(*thisHouse)->rooms[-1].objects[-1].what` (`GliderPRO/Sources/Triggers.c:179-180`).
  Both are out-of-bounds reads. Because the only case handled in that branch is
  grease, the practical effect is "nothing happens" unless the garbage byte pair
  happens to equal `kGreaseRt`/`kGreaseLf`.

**No code path in the shipped game checks `roomLink != -1` before using it as a room
index.** Verified by grep: `roomLink` appears at `GliderPRO/Sources/Interactions.c:994`
(`HandleSwitches`, feeding `SetObjectState(roomLinked, ...)` which indexes
`(*thisHouse)->rooms[room]` with no guard, `GliderPRO/Sources/Objects.c:373`),
`GliderPRO/Sources/Triggers.c:46`, and the four `StartGlider*` prologues.

**A Go port cannot reproduce this literally** — Go panics on a negative slice index.
The port must decide on a defined behaviour and must document it. The
lowest-risk choice: treat `roomLink == -1` at *use* time as "do nothing" (skip the
transit / skip the trigger fire / skip the `SetObjectState`), and additionally suppress
hot-spot creation when `roomLink == -1`, which makes the 24 stale links behave like
the 165 `where == -100` ones — i.e. inert, which is the closest observable match to
"garbage rect inside the same room, no room change".

### 8.11 `ReadyGliderFromTransit` — arrival positioning

```
1.  ReadyGliderFromTransit(thisGlider, toWhat):
2.    if twoPlayerGame and onePlayerLeft and thisGlider->which == playerDead: return
3.    FlagGliderNormal(thisGlider)          // FIRST, before any tempRect snapshot:
4.                                          // normalises dest/destShadow to 48x20 / 48x9
5.    switch toWhat:
6.      case kLinkedToOther:                            // 0 - a plain transporter
7.        StartGliderTransportingIn(thisGlider)
8.        tempRect = thisGlider->dest                   // ALREADY-normalised rect
9.        CenterRectInRect(&tempRect, &transRect)
10.       thisGlider->dest.left   = tempRect.left
11.       thisGlider->dest.right  = tempRect.right
12.       thisGlider->dest.top    = tempRect.top
13.       thisGlider->dest.bottom = tempRect.bottom
14.       thisGlider->destShadow.left  = tempRect.left
15.       thisGlider->destShadow.right = tempRect.right
16.       thisGlider->whole       = thisGlider->dest
17.       thisGlider->wholeShadow = thisGlider->destShadow
18.       thisGlider->enteredRect = thisGlider->dest        // <== set HERE
19.     case kLinkedToLeftMailbox:                      // 1
20.       StartGliderMailingOut(thisGlider)             // ONE argument; it reads the
21.                                                     // global linkedToWhat itself
22.       thisGlider->clip = transRect
23.       thisGlider->clip.right  -= 64
24.       thisGlider->clip.bottom -= 25
25.       tempRect = thisGlider->dest
26.       thisGlider->dest.left   = thisGlider->clip.right
27.       thisGlider->dest.right  = thisGlider->dest.left        // zero width
28.       thisGlider->dest.bottom = thisGlider->clip.bottom - 4
29.       thisGlider->dest.top    = thisGlider->dest.bottom - RectTall(&tempRect)
30.       thisGlider->destShadow.left  = thisGlider->dest.left
31.       thisGlider->destShadow.right = thisGlider->dest.right
32.       thisGlider->whole = dest; thisGlider->wholeShadow = destShadow
33.       // NOTE: enteredRect NOT set here
34.     case kLinkedToRightMailbox:                     // 2
35.       StartGliderMailingOut(thisGlider)
36.       thisGlider->clip = transRect
37.       thisGlider->clip.left   += 79
38.       thisGlider->clip.bottom -= 25
39.       tempRect = thisGlider->dest
40.       thisGlider->dest.right  = thisGlider->clip.left
41.       thisGlider->dest.left   = thisGlider->dest.right
42.       thisGlider->dest.bottom = thisGlider->clip.bottom - 4
43.       thisGlider->dest.top    = thisGlider->dest.bottom - RectTall(&tempRect)
44.       ... same destShadow / whole / wholeShadow tail as lines 30-32 ...
45.     case kLinkedToCeilingDuct:                      // 3
46.       StartGliderDuctingIn(thisGlider)              // ONE argument
47.       tempRect = thisGlider->dest
48.       CenterRectInRect(&tempRect, &transRect)
49.       dest.left = tempRect.left; dest.right = tempRect.right; dest.top = tempRect.top
50.       thisGlider->dest.bottom = thisGlider->dest.top
51.       QOffsetRect(&thisGlider->dest, 0, -RectTall(&tempRect))   // start above the duct
52.       destShadow.left = tempRect.left; destShadow.right = tempRect.right
53.       thisGlider->whole = dest; thisGlider->wholeShadow = destShadow
54.     case kLinkedToFloorDuct:                        // 4
55.       break;                                        // <== EMPTY
56.     default:
57.       break;
58.   if twoPlayerGame and thisGlider->which != firstPlayer:
59.      TagGliderIdle(thisGlider)
```
(`GliderPRO/Sources/Transit.c:65-147`)

- **`enteredRect` is set only in the `kLinkedToOther` arm** (line 18). The mailbox and
  duct arms set it later, in their `Finish*` routines: `FinishGliderMailingLeft`
  (`GliderPRO/Sources/Player.c:851-887`), `FinishGliderMailingRight` (`:889-925`), and
  `FinishGliderDuctingIn` (`:927-956`) each set `enteredRect = dest` (`:883`, `:921`,
  `:953`) once the emergence completes. So a death
  *during* a mailbox emergence respawns at the *previous* room's `enteredRect`. That is
  the original behaviour.
- **`FinishGliderDuctingIn` is the only transit finisher that calls
  `FlagStillOvers(thisGlider)`** (`GliderPRO/Sources/Player.c:954`), which pre-marks
  every hot spot the glider currently overlaps as `stillOver` so it does not
  immediately re-trigger. A glider that arrives in a duct sitting on a switch will not
  fire it; a glider that arrives from a mailbox will. Reproduce this asymmetry.
- The magic numbers 64 / 25 / 4 (lines 23-28) and 79 / 25 / 4 (lines 37-42) are the
  mailbox art's slot geometry.
- Line 58: in two-player mode, the glider that did *not* initiate the transit is put
  to sleep for 30 frames.
- `StartGliderMailingOut` and `StartGliderDuctingIn` each take **only the glider
  pointer**; `StartGliderMailingOut` picks left vs. right by testing the global
  `linkedToWhat == kLinkedToLeftMailbox` itself
  (`GliderPRO/Sources/Modes.c:181-209`, `:279-284`).
---

## 9. The two-player "escape / limbo / follow the leader" protocol

Two-player Glider PRO is not split-screen and not independent: **both gliders must be
in the same room at all times.** The protocol that enforces this is the single most
intricate piece of progression logic in the game.

### 9.1 The state variables

| Variable | Type | Values |
|---|---|---|
| `otherPlayerEscaped` | `short` | one of the `kPlayer*` codes below, or `kNoOneEscaped` |
| `firstPlayer` | `Boolean` | `which` of the glider that escaped first (`kPlayer1` = 1 / `kPlayer2` = 0) |
| `onePlayerLeft` | `Boolean` | one player is permanently dead; the other plays alone |
| `playerDead` | `Boolean` | `which` of the permanently-dead player |
| `playerSuicide` | `Boolean` | a Delete-key forced kill is in progress |
| `saidFollow` | `short` | how many times the "follow me" voice has played (max 3) |
| `activeRectEscaped` | `short` | the hot-spot index the first player used (transport/mail/duct) |

Escape codes (`GliderPRO/Headers/GliderDefines.h:596-608`), all negative:

| Constant | Value | Meaning |
|---|---:|---|
| `kNoOneEscaped` | -1 | both gliders in the room, nobody waiting |
| `kPlayerEscapedRight` | -2 | first player left through the right wall |
| `kPlayerEscapedLeft` | -3 | first player left through the left wall |
| `kPlayerEscapedUp` | -4 | first player left through the ceiling |
| `kPlayerEscapedDown` | -5 | first player left through the floor |
| `kPlayerEscapedUpStairs` | -6 | first player finished climbing the up-stairs |
| `kPlayerEscapedDownStairs` | -7 | first player finished the down-stairs |
| `kPlayerEscapingUpStairs` | -8 | first player has *started* climbing the up-stairs (climb in progress) |
| `kPlayerEscapingDownStairs` | -9 | first player has *started* descending (descent in progress) |
| `kPlayerTransportedOut` | -10 | first player faded out of a transporter |
| `kPlayerDuckedOut` | -11 | first player went into a duct |
| `kPlayerMailedOut` | -12 | first player went into a mailbox |
| `kPlayerIsDeadForever` | -69 | terminal marker set by `OffAMortal` |

`kPlayerEscapingUpStairs` (-8) and `kPlayerEscapingDownStairs` (-9) are **live, not
vestigial**, and the distinction between "-ing" and "-ed" is load-bearing. The
stairs are the only escape route with a multi-frame animation, so they need two
states:

- **`-ing` is set at the hot spot, when the first player *begins* the climb/descent**
  (`GliderPRO/Sources/Interactions.c:1266` for `kMoveItUp`, `:1301` for
  `kMoveItDown`), together with `RefreshScoreboard(kEscapedTitleMode)` and
  `StartGliderGoingUpStairs`/`Down`.
- **`-ed` is set when the animation *completes*** and the climber goes into limbo
  (`GliderPRO/Sources/Player.c:363` / `:491`).
- The second player's hot-spot test at `Interactions.c:1271` / `:1306` matches only the
  **`-ed`** code, so player 2 cannot start climbing until player 1 has fully
  disappeared. Note the asymmetric consequence: while `otherPlayerEscaped` is `-8`,
  player 2 hitting the *same* staircase falls through both branches and nothing
  happens.
- `FollowTheLeader` (`GliderPRO/Sources/Transit.c:499-556`) groups `-8` with
  `kPlayerEscapedUp`/`kPlayerEscapedUpStairs` (`:501-503`) and `-9` with
  `kPlayerEscapedDown`/`kPlayerEscapedDownStairs` (`:510-512`), so a forced follow
  mid-climb still moves the pair the right way. `Player.c:1546` / `:1555` handle the
  same two codes on respawn (§10.3).

### 9.2 The three-way handshake

Every escape site in `Interactions.c` implements the same three-way test. Taking
`CheckEscapeRightTwo` as the archetype
(`GliderPRO/Sources/Interactions.c:599-660`):

```
1.  CheckEscapeRightTwo(thisGlider):
2.    if rightThresh == kRightWallLimit:                    // wall closed
3.       if thisGlider->ignoreRight:                        // a door/gap in a closed wall
4.          if thisGlider->dest.right > kNoRightWallLimit:  // > 536
5.             HANDSHAKE (see below)
6.          // else: nothing at all happens - no bounce, no sound
7.       else:
8.          PlayPrioritySound(foilTotal > 0 ? kFoilHitSound : kHitWallSound, ...)
9.          offset = kRightWallLimit - thisGlider->dest.right     // 500 - right
10.         thisGlider->hVel = -thisGlider->hVel + offset
11.   else:                                                 // wall open
12.      HANDSHAKE
13.
14.   HANDSHAKE:
15.     if otherPlayerEscaped == kNoOneEscaped:
16.        // ---- FIRST player to leave ----
17.        otherPlayerEscaped = kPlayerEscapedRight
18.        RefreshScoreboard(kEscapedTitleMode)
19.        FlagGliderInLimbo(thisGlider, true)
20.     elif otherPlayerEscaped == kPlayerEscapedRight:
21.        // ---- SECOND player agreed; both go ----
22.        otherPlayerEscaped = kNoOneEscaped
23.        MoveRoomToRoom(thisGlider, kToRight)
24.     else:
25.        // ---- SECOND player tried to leave a DIFFERENT way: refuse ----
26.        PlayPrioritySound(kDontExitSound, kDontExitPriority)
27.        offset = kNoRightWallLimit - thisGlider->dest.right   // ALWAYS 536 - right
28.        thisGlider->hVel = -thisGlider->hVel + offset
```

Two details a port is likely to get wrong:

- **The refusal branch (lines 24-28) does not move the glider.** It only adjusts
  `hVel`; there is no `QOffsetRect`. The `offset` term is folded into the velocity and
  the position correction happens on the next `MoveGlider` step.
- **The refusal branch always uses `kNoRightWallLimit` (536)**, in both the
  `ignoreRight` and the open-wall paths — not `kRightWallLimit`. Only the genuine
  wall-bounce at line 9 uses `kRightWallLimit` (500).
- The `ignoreRight`-but-not-past-536 case (line 6) is a silent no-op: the glider keeps
  flying and neither bounces nor escapes.

The same pattern appears in `CheckEscapeUpTwo` (`:171-242`, code
`kPlayerEscapedUp`), `CheckEscapeDownTwo` (`:283-376`, `kPlayerEscapedDown`),
`CheckEscapeLeftTwo` (`:509-570`, `kPlayerEscapedLeft`), and in
`MoveGliderUpStairs` (`GliderPRO/Sources/Player.c:345-368`,
`kPlayerEscapedUpStairs`) / `MoveGliderDownStairs`
(`GliderPRO/Sources/Player.c:473-496`, `kPlayerEscapedDownStairs`) — note those two are
in `Player.c`, not `Interactions.c`.

For the three link transitions the handshake also matches on the **hot-spot index**,
so both gliders must use the *same* transporter/mailbox/duct, not merely the same
kind:

```c
if (otherPlayerEscaped == kNoOneEscaped)
{   if (thisGlider->mode != kGliderInLimbo)
    {   activeRectEscaped = index;
        StartGliderTransporting(thisGlider, who); } }
else if (otherPlayerEscaped == kPlayerTransportedOut)
{   if ((thisGlider->mode != kGliderInLimbo) && (activeRectEscaped == index))
    {   StartGliderTransporting(thisGlider, who); } }
```
(`GliderPRO/Sources/Interactions.c:1391-1410`; `kMailItLeft` at `:1424`,
`kMailItRight` at `:1467`, `kDuctItDown` at `:1510`, `kDuctItUp` at `:1543` are the
same shape.)

Note the escape code for these is not set by the hot-spot handler — it is set by the
`MoveGlider*` routine when the animation *completes*
(`TransportGliderOut` at `GliderPRO/Sources/Player.c:594-655`,
`MoveGliderDownDuct` at `:657-752`, `MoveGliderUpDuct` at `:754-849`,
`MoveGliderInMailLeft` at `:960-1049`, `MoveGliderInMailRight` at `:1051-1140`).

### 9.3 Limbo

```c
void FlagGliderInLimbo (gliderPtr thisGlider, Boolean sayIt)     // Modes.c:458
{
    thisGlider->wasMode = thisGlider->mode;
    thisGlider->mode = kGliderInLimbo;
    if ((sayIt) && (saidFollow < 3))
    {   PlayPrioritySound(kFollowSound, kFollowPriority);
        saidFollow++; }
    firstPlayer = thisGlider->which;
}

void UndoGliderLimbo (gliderPtr thisGlider)                      // Modes.c:472
{
    if ((twoPlayerGame) && (onePlayerLeft) && (thisGlider->which == playerDead))
        return;
    if (thisGlider->mode == kGliderInLimbo)
        thisGlider->mode = thisGlider->wasMode;
    thisGlider->dontDraw = false;
}
```

- `kGliderInLimbo` = 21. In `HandleGlider`'s dispatch it is
  **`case kGliderInLimbo: break;`** — a no-op (`GliderPRO/Sources/Player.c:1423-1424`).
  A glider in limbo does not move, does not animate, and is not affected by physics.
  It is still *drawn* unless `dontDraw` is set.
- `saidFollow` is reset to 0 at the start of every game (`GliderPRO/Sources/Play.c:115`)
  and is capped at 3, so the "follow me!" voice plays at most three times per game.
- `firstPlayer` is set on *every* `FlagGliderInLimbo`, so it always identifies the most
  recent escaper.

The various `Insure*`/`Toggle*` mode helpers all refuse to act on a limbo or dead
glider:

```c
void InsureGliderFacingRight (gliderPtr thisGlider)              // Modes.c:497
{   if ((twoPlayerGame) && (onePlayerLeft) && (thisGlider->which == playerDead)) return;
    if (thisGlider->mode == kGliderBurning) return;
    ... }
void StartGliderFoilGoing (gliderPtr thisGlider)                 // Modes.c:581
{   if ((thisGlider->mode == kGliderGoingFoil) || (thisGlider->mode == kGliderInLimbo)) return;
    ... }
```
(`GliderPRO/Sources/Modes.c:484-629`)

### 9.4 The Delete-key forced kill

If one player refuses (or is unable) to follow, Player 1 can kill the *waiting*
glider to unstick the game.

```c
// Input.c:367 — inside GetInput / GetInput2
if ((otherPlayerEscaped != kNoOneEscaped) &&
    (BitTst(&theKeys, kDeleteKeyMap)) &&
    (thisGlider->which) &&                  // <== kPlayer1 == TRUE == 1
    (!onePlayerLeft))
    ForceKillGlider();
```

`(thisGlider->which)` is truthy only for `kPlayer1` (= `TRUE` = 1), so **only Player 1's
Delete key works** (`kPlayer2` = `FALSE` = 0). The scoreboard advertises this with the
room-title override `"Hit Delete key if unable to Follow"` in `kEscapedTitleMode`
(drawn by `RefreshRoomTitle`, which `RefreshScoreboard` calls —
`GliderPRO/Sources/Scoreboard.c:136-188`, strings at `:156` and `:172`).

```c
void ForceKillGlider (void)                                      // Transit.c:449
{
    if (theGlider.mode == kGliderInLimbo)
    {   if (theGlider2.mode != kGliderFadingOut)
        {   StartGliderFadingOut(&theGlider2);
            PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
            playerSuicide = true; } }
    else if (theGlider2.mode == kGliderInLimbo)
    {   if (theGlider.mode != kGliderFadingOut)
        {   StartGliderFadingOut(&theGlider);
            PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
            playerSuicide = true; } }
}
```

**It kills the glider that is *not* in limbo** — the one that refused to follow. The
limbo glider is the one that already left, and it is the survivor. `playerSuicide`
routes the subsequent `OffAMortal` into `FollowTheLeader()` instead of a normal
respawn (§10.2).

### 9.5 `FollowTheLeader`

```
1.  FollowTheLeader():
2.    playerSuicide = false
3.    wasEscaped = otherPlayerEscaped
4.    otherPlayerEscaped = kNoOneEscaped
5.    Boolean oneOrTwo;                             // NOTE: not a pointer, and NOT
6.                                                  // initialised on the fall-through
7.    if theGlider.mode == kGliderInLimbo:
8.       oneOrTwo = true                            // glider 1 is the leader/in limbo
9.       theGlider2.dest       = theGlider.dest
10.      theGlider2.destShadow = theGlider.destShadow
11.      theGlider2.whole      = theGlider2.dest
12.      theGlider2.wholeShadow= theGlider2.destShadow
13.   elif theGlider2.mode == kGliderInLimbo:        // NOT a plain `else`
14.      oneOrTwo = false
15.      theGlider.dest       = theGlider2.dest
16.      theGlider.destShadow = theGlider2.destShadow
17.      theGlider.whole      = theGlider.dest
18.      theGlider.wholeShadow= theGlider.destShadow
19.   // if NEITHER is in limbo, oneOrTwo is uninitialised garbage
20.   pick = oneOrTwo ? &theGlider2 : &theGlider     // the FOLLOWER is passed below
21.   switch wasEscaped:
22.     kPlayerEscapedUp, kPlayerEscapingUpStairs, kPlayerEscapedUpStairs:
23.        MoveRoomToRoom(pick, kAbove)
24.     kPlayerEscapedDown, kPlayerEscapingDownStairs, kPlayerEscapedDownStairs:
25.        MoveRoomToRoom(pick, kBelow)
26.     kPlayerEscapedLeft:   MoveRoomToRoom(pick, kToLeft)
27.     kPlayerEscapedRight:  MoveRoomToRoom(pick, kToRight)
28.     kPlayerTransportedOut: TransportRoomToRoom(pick)
29.     kPlayerMailedOut:      MoveMailToMail(pick)
30.     kPlayerDuckedOut:      MoveDuctToDuct(pick)
31.     default: break
```
(`GliderPRO/Sources/Transit.c:473-557`)

Three things to get right:

- **`oneOrTwo` is a `Boolean`, not a pointer, and the glider actually passed to the
  `Move*` call is the *non*-limbo one.** `oneOrTwo == true` means `theGlider` is the
  limbo leader, and the code calls `MoveRoomToRoom(&theGlider2, ...)`. This matters
  only for the single-glider `MoveRoomToRoom` path, which in a two-player game
  immediately re-branches on `twoPlayerGame` and moves both anyway — but it is what the
  original does.
- **The second branch is `else if (theGlider2.mode == kGliderInLimbo)`, not `else`.**
  If neither glider is in limbo, `oneOrTwo` is read uninitialised. A Go port cannot
  reproduce that; pick a deterministic value and document it. (In practice
  `FollowTheLeader` is only reached with `otherPlayerEscaped != kNoOneEscaped`, which
  implies one glider is in limbo.)

The revived glider is teleported *on top of* the leader (lines 9-12 / 15-18) and then
both are moved into the destination room. Note lines 22-25 use the **same** target for
the plain-escape and the stairs codes: a `kPlayerEscapedUpStairs` follow does
`MoveRoomToRoom(kAbove)`, and since `takingTheStairs` is false at this point (it was
cleared by the leader's own `DrawLocale`), the arrival uses the plain `OffsetGlider`
path, not the stairs animation. That is a visible difference from a normal
both-players-took-the-stairs transition.

**`FollowTheLeader` is called from exactly one place**: `OffAMortal`, when
`playerSuicide` is set (`GliderPRO/Sources/Player.c:1530`).

### 9.6 The scoreboard during an escape

`RefreshScoreboard(kEscapedTitleMode)` replaces the room name with
`"Hit Delete key if unable to Follow"`; `kSavingTitleMode` shows `"Saving Game…"`;
`kNormalTitleMode` shows `thisRoom->name`. The text is drawn twice — black at
`(1, 10)` then white at `(0, 9)` — a 1-pixel drop shadow
(`GliderPRO/Sources/Scoreboard.c:136-190`).

---

## 10. Death and respawn

### 10.1 `OffAMortal` — the complete rule set

Every death funnels through this one function.

```
1.  OffAMortal(thisGlider):
2.    if gameOver: return                                    // deaths after game over are ignored
3.    if numShredded > 0: RemoveShreds()
4.    mortals--
5.    if mortals < 0:
6.       HideGlider(thisGlider)
7.       if twoPlayerGame:
8.          if mortals < -1:                                 // second player also out
9.             FlagGameOver()
10.            thisGlider->dontDraw = true
11.         else:                                            // mortals == -1
12.            FlagGliderInLimbo(thisGlider, false)           // sayIt = false: no voice
13.            thisGlider->dontDraw = true
14.            onePlayerLeft = true
15.            playerDead = thisGlider->which
16.      else:
17.         FlagGameOver()
18.         thisGlider->dontDraw = true
19.   else:
20.      QuickGlidersRefresh()
21.      HideGlider(thisGlider)
22.   if mortals >= 0:
23.      if thisGlider->mode == kGliderGoingFoil: DeckGliderInFoil(thisGlider)
24.      FlagGliderNormal(thisGlider)
25.      if playerSuicide:
26.         FollowTheLeader()
27.      else:
28.         StartGliderFadingIn(thisGlider)
29.         thisGlider->dest              = thisGlider->enteredRect     // <== RESPAWN POINT
30.         thisGlider->whole             = thisGlider->dest
31.         thisGlider->destShadow.left   = thisGlider->dest.left
32.         thisGlider->destShadow.right  = thisGlider->dest.right
33.         thisGlider->wholeShadow       = thisGlider->destShadow
34.   elif mortals == -1 and onePlayerLeft and !gameOver:
35.      switch otherPlayerEscaped:                 // same 7 code groups as
36.         // FollowTheLeader lines 21-31, but the glider is chosen by
37.         // `playerDead == kPlayer1 ? &theGlider2 : &theGlider`
38.         // (i.e. the SURVIVOR follows the room the escaper left for)
39.      otherPlayerEscaped = kPlayerIsDeadForever
```
(`GliderPRO/Sources/Player.c:1484-1604`)

Rules stated plainly:

1. **`mortals` is a shared pool.** One player dying decrements the same counter the
   other draws from. A two-player game starts at 4 and ends when it goes below -1.
2. **A one-player game ends when `mortals` goes below 0**, i.e. on the third death
   (2 → 1 → 0 → -1).
3. **A two-player game gives the first player to run out a "limbo forever" state
   rather than ending** (lines 11-15). `onePlayerLeft = true` and
   `playerDead = which`. From then on every routine that touches a glider checks
   `(twoPlayerGame) && (onePlayerLeft) && (which == playerDead)` and bails —
   `UndoGliderLimbo` (`Modes.c:474`), `InsureGliderFacingRight/Left`
   (`Modes.c:499`/`:510`), `TagGliderIdle` (`Modes.c:633`),
   `ReadyGliderForTripUpStairs/DownStairs` (`Modes.c:523`/`:554` — note those two spell
   the conjuncts in a different order, `which == playerDead` before `onePlayerLeft`),
   `ReadyGliderFromTransit` (`Transit.c:69`), `OffsetGlider` (`Player.c:1445`). That
   is the complete set of eight; `grep -n "which == playerDead"` finds no others.
   The game ends on the *next* death (line 8, `mortals < -1`).
4. **The respawn point is `thisGlider->enteredRect`** (line 29) — the rect recorded
   when the glider *entered the current room*. So you always respawn where you came in,
   not where you died and not at the house's `initial` point. The exhaustive list of
   places `enteredRect` is written:

| Site | Value written |
|---|---|
| `MoveRoomToRoom` kToRight (`Transit.c:177`, `:178`, `:186`) | `(0, 30 + newRoom.leftStart)`, 48×20 |
| `MoveRoomToRoom` kToLeft (`Transit.c:209`, `:210`, `:218`) | `(464, 30 + newRoom.rightStart)`, 48×20 |
| `MoveRoomToRoom` kAbove/kBelow non-stairs (`Transit.c:233`, `:234`, `:239`, `:265`, `:266`, `:271`) | the glider's post-`OffsetGlider` `dest` |
| `FinishGliderUpStairs` (`Player.c:425`) | `dest` at the moment the glider clears the stairs |
| `FinishGliderDownStairs` (`Player.c:554`) | same |
| `ReadyGliderFromTransit` kLinkedToOther (`Transit.c:88`) | the transporter-centred `dest` |
| `FinishGliderMailingLeft` (`Player.c:883`) | `dest` when fully out of the mailbox |
| `FinishGliderMailingRight` (`Player.c:921`) | same |
| `FinishGliderDuctingIn` (`Player.c:953`) | `dest` when fully out of the duct |
| **`FadeGliderIn`** (`Player.c:240`) | `dest`, on the frame the fade-in completes |
| **`TransportGliderIn`** (`Player.c:270`) | `dest`, on the frame the transport-in completes |

  Those are all 12 writes in the whole program; `grep -n enteredRect` finds nothing
  outside `Player.c` and `Transit.c`, so **`enteredRect` is never touched by
  `InitGlider` or `NewGame`.** In the original the `gliderType` lives in a static
  global, so at the instant `PlayGame` starts it holds whatever the previous game left
  (or zeros on the first game). In practice this is almost never observable, because
  `NewGame` puts the glider into `kGliderFadingIn`
  (`GliderPRO/Sources/Play.c:185`, `:188`) and `FadeGliderIn` writes
  `enteredRect = dest` as soon as that 16-frame fade completes
  (`GliderPRO/Sources/Player.c:237-240`) — i.e. the start position becomes the respawn
  point before normal play begins. The stale value could only be used by a death inside
  the opening fade. **A Go port should still set `enteredRect = dest` in `InitGlider`**
  so the behaviour is defined rather than dependent on struct-zeroing.

5. **`playerSuicide` diverts the respawn into `FollowTheLeader()`** (lines 25-26),
   which is why the Delete-key kill costs a life *and* moves both gliders.
6. **Line 23**: a glider dying while in `kGliderGoingFoil` (mid-foil-donning animation)
   gets `DeckGliderInFoil` called so the foil sprite state is consistent before
   `FlagGliderNormal` resets it.
7. **Line 20 `QuickGlidersRefresh()` is called only in the non-fatal branch** (i.e. only
   when `mortals >= 0`). That function does **not** clamp negative `mortals` when
   drawing the count, whereas `RefreshNumGliders` does:
   ```c
   // Scoreboard.c:209-211  RefreshNumGliders
   displayMortals = mortals;
   if (displayMortals < 0) displayMortals = 0;
   // Scoreboard.c:284  QuickGlidersRefresh  — NumToString((long)mortals, ...), no clamp
   ```
   In practice the missing clamp is unreachable from `OffAMortal`, because the only two
   call sites are this non-fatal branch and the bonus-glider award
   (`GliderPRO/Sources/Interactions.c:844`), both of which run with `mortals >= 0`. The
   asymmetry is worth knowing about but does not produce a visible negative count in the
   shipped game.
8. **Line 2**: once `gameOver` is set, further deaths are no-ops. Combined with
   `PlayGame` line 35 choosing the animation by `mortals < 0`, this makes the outcome
   sticky: whichever condition fires first wins.

### 10.2 The complete list of things that kill the glider

Everything that ends a glider's life calls either `StartGliderFadingOut` (a delayed
death: the fade-out animation runs, then `FadeGliderOut` calls `OffAMortal`) or
`FlagGliderShredding` (the shredder, with its own 68-frame delay).

| Cause | Site | Path |
|---|---|---|
| Falling through a closed floor | `Interactions.c:441-447` | `vVel = kFloorLimit - dest.bottom`; `StartGliderFadingOut`; `kFadeOutSound` |
| Hitting a roof at a fatal angle | `Interactions.c:496-505` | same |
| Flying into the ceiling while burning | `Interactions.c:698-704` | `wasMode = 0`; `StartGliderFadingOut`; `kFadeOutSound` |
| Flying into the floor while burning | `Interactions.c:717-723` | same |
| Entering a transporter/mailbox/duct while burning | `Interactions.c:1381-1387`, `:1424-1430`, `:1467-…`, `:1510-…`, `:1543-…` | same |
| Burning out (`wasMode` countdown from `kFramesToBurn` 60) | `Player.c` `MoveGliderBurning` (`:203`) | `StartGliderFadingOut` |
| The paper shredder | `Interactions.c` → `FlagGliderShredding` (`Modes.c:366`) | see below |
| Enemy/hazard collision (dart, ball, drip, fish, copter, outlet zap, toast) | `Dynamics*.c` → `StartGliderFadingOut` | same |
| Delete-key forced kill (2P only) | `Transit.c:449` `ForceKillGlider` | `StartGliderFadingOut` + `playerSuicide = true` |

`StartGliderFadingOut` (`GliderPRO/Sources/Modes.c:75-116`) is idempotent (early return
if already `kGliderFadingOut`), removes foil, and shrinks the glider's `dest` to a
48×20 rect anchored at the bottom-left if it was taller (the burning glider is 26
tall), adding both the old rect and — when `hasMirror` — a mirrored copy offset by
`(-20, -16)` to the work rects. It sets `frame = kLastFadeSequence - 1` = 15.
`FadeGliderOut` (`GliderPRO/Sources/Player.c:291-311`) decrements `frame` and calls
`OffAMortal` as soon as it goes negative (`:293-295`), i.e. after 16 frames.

The shredder path is the longest:

```
1.  FlagGliderShredding(thisGlider, bounds):                  // Modes.c:366-402
2.    PlayPrioritySound(kCaughtFireSound, ...)                // FIRST; yes, the fire sound
3.    dest.left = bounds->left + 36; dest normalised to 48x20
4.    whole/wholeShadow/destShadow fixed up; src/mask = gliderSrc[facing ? 2 : 0]
5.    hVel = vVel = hDesiredVel = vDesiredVel = 0
6.    mode = kGliderShredding                                 // Modes.c:385
7.    frame = bounds->bottom - 3                              // Modes.c:400 - target Y,
8.                                                            // NOT a frame index
--
6.  MoveGliderShredding(thisGlider):                          // Player.c:1260
7.    #define kDropShredSlow  1
8.    #define kDropShredFast  4
9.    if thisGlider->frame > 0:
10.      src = mask = gliderSrc[facing == kFaceLeft ? 2 : 0]
11.      vNotClipped = thisGlider->frame - thisGlider->dest.top
12.      if vNotClipped < kGliderHigh:                         // already in the slot
13.         dest.top += kDropShredSlow; dest.bottom += kDropShredSlow   // 1 px/frame
14.         shadowVisible = false
15.         PlayPrioritySound(kShredSound, kShredPriority)
16.      else:
17.         dest.top += kDropShredFast; dest.bottom += kDropShredFast   // 4 px/frame
18.      vNotClipped = thisGlider->frame - thisGlider->dest.top   // RECOMPUTED
19.      if vNotClipped < kGliderHigh:
20.         if vNotClipped <= 0:
21.            AddAShreddedGlider(&thisGlider->dest)
22.            thisGlider->frame = kShredderCountdown          // -68 (Player.c:17)
23.         else:                                              // partially eaten
24.            dest.bottom = dest.top + vNotClipped
25.            src.bottom  = src.top  + vNotClipped
26.            mask.bottom = mask.top + vNotClipped
27.   else:
28.      thisGlider->frame++
29.      if thisGlider->frame >= 0: OffAMortal(thisGlider)
```
(`GliderPRO/Sources/Player.c:1260-1319`)

Two easy things to get wrong: `vNotClipped` is **recomputed** at line 18 after the
drop, and the `<= 0` test at line 20 is **nested inside** a second
`< kGliderHigh` test, whose `else` clips the sprite's *bottom* (not its top, unlike the
stairs code).

So `frame` is overloaded: while positive it is the **target Y coordinate**; once the
glider is fully shredded it becomes a **negative countdown** of 68 frames (≈2.3 s at
30 fps) during which the paper shreds fall, and only then is the life lost.

### 10.3 What resets on death vs on room change

This table is the answer to "what state persists".

| State | Room change | Death (non-fatal) | New game |
|---|---|---|---|
| `thisRoomNumber`, `thisRoom` | **changes** | unchanged | set to `firstRoom` |
| `previousRoom` | set to the old room | unchanged | set by `ForceThisRoom` |
| `localNumbers[9]`, `isStructure[9]` | rebuilt (`DrawLocale`) | unchanged | rebuilt |
| `thisTiles[8]`, `thisBackground` | reloaded | unchanged | reloaded |
| `leftOpen/rightOpen/topOpen/bottomOpen`, `leftThresh/rightThresh` | recomputed (`DetermineRoomOpenings`) | unchanged | recomputed |
| `numLights`, `shadowVisible` | recomputed | unchanged | recomputed |
| `masterObjects[]`, `hotSpots[]` | rebuilt (`ListAllLocalObjects`) | unchanged | rebuilt |
| `dinahs[]` (dynamic objects) | **destroyed** (`ZeroDinahs`) | unchanged | destroyed |
| bands in flight | **destroyed** (`KillAllBands`) | unchanged | destroyed |
| `triggers[16]` | **destroyed** (`ZeroTriggers`) | unchanged | destroyed |
| flames / tikis / coals / pendulums / stars / grease / shreds / chimes | **destroyed** (`ZeroFlamesAndTheLike`) | shreds removed (`RemoveShreds`) | destroyed |
| `savedMaps[24]` background swatches | **destroyed** (`NilSavedMaps` in `ReadyLevel`) | unchanged | destroyed |
| `numTempManholes` | zeroed | unchanged | zeroed |
| `tvInRoom`, `tvWithMovieNumber`, movie playback | reset / stopped | unchanged | reset |
| `takingTheStairs` | consumed then cleared | unchanged | n/a |
| `theScore` | +100 if the room was unvisited | unchanged | 0 |
| `mortals` | unchanged | **-1** | 2 (or 4 in 2P) |
| `numStarsRemaining` | unchanged | unchanged | `CountStarsInHouse()` |
| `batteryTotal`, `bandsTotal`, `foilTotal` | unchanged | unchanged | 0 |
| `showFoil` | unchanged | unchanged | false |
| glider `mode` | set by the transition | `kGliderFadingIn` | `kGliderFadingIn` |
| glider `dest` | offset / repositioned | `= enteredRect` | `= initial` |
| glider `hVel/vVel/hDesiredVel/vDesiredVel` | zeroed by the `Ready*`/`Flag*` routines | zeroed (`FlagGliderNormal`) | zeroed |
| glider `facing` | forced by `Insure*` on side exits | unchanged (`FlagGliderNormal` keeps it) | `kFaceRight` |
| glider `enteredRect` | **set** | read, not written | **not written** (see §10.1 note 4) |
| `otherPlayerEscaped` | cleared to `kNoOneEscaped` on a successful move | set to `kPlayerIsDeadForever` on the 2P permanent death | `kNoOneEscaped` |
| `onePlayerLeft`, `playerDead` | unchanged | set on the 2P permanent death | false |
| `saidFollow` | unchanged | unchanged | 0 |
| `gameFrame`, `evenFrame` | unchanged | unchanged | 0 |
| `numBands` | unchanged | unchanged | 0 |
| room `visited` flags | **the outgoing room is marked** | unchanged | **all cleared** |
| object `state` bytes | unchanged (they live in the house) | unchanged | reset from `initial` |
| the telephone/chime timers | unchanged | unchanged | reinitialised |

The single most important row: **object `state` bytes survive room changes and
deaths.** A bonus you took stays taken; a light you turned on stays on; a star you
collected stays collected. Only a new game resets them.

---

## 11. Winning: the end-of-house condition

### 11.1 The condition

There is exactly one win condition: collect every star in the house.

```c
case kStar:
if (SetObjectState(thisRoomNumber, masterObjects[whoLinked].objectNum, 0, whoLinked))
{
    PlayPrioritySound(kEnergizeSound, kEnergizePriority);
    RestoreFromSavedMap(thisRoomNumber, masterObjects[whoLinked].objectNum, false);
    AddSparkle(&bounds);
    StopStar(thisRoomNumber, masterObjects[whoLinked].objectNum);
    numStarsRemaining--;
    if (numStarsRemaining <= 0)
        FlagGameOver();
    else
        DisplayStarsRemaining();
    RedrawAllGrease();
    theScore += kStarPoints;                          // 5000
}
who->isOn = false;
```
(`GliderPRO/Sources/Interactions.c:933-951`)

- `numStarsRemaining` is seeded by `CountStarsInHouse()` for a new game or
  `smallGame.wasStarsLeft` for a resume (`GliderPRO/Sources/Play.c:312-314`).
- `SetObjectState(..., 0, ...)` — action 0 is `kToggle`
  (`GliderPRO/Headers/GliderDefines.h:441`), but the bonus arm of `SetObjectState`
  (`GliderPRO/Sources/Objects.c:446-462`, which is the arm `kStar` falls into)
  **ignores `action` entirely**: it unconditionally does
  `changed = (data.c.state == true); newState = false; data.c.state = false;`. So it is
  a force-off, not a toggle, and it returns `true` only if the state actually changed —
  which is why a star cannot be collected twice.
- `kStarPoints` = 5000 (`GliderPRO/Headers/GliderDefines.h:541`).
- **The win test is inside the star-pickup handler.** It is never evaluated anywhere
  else. See §2.6 Test 7 for the consequence in `Fun House` (0 stars ⇒ unwinnable).

```c
void FlagGameOver (void)                                 // GameOver.c:237
{
    gameOver = true;
    countDown = kNumCountDownFrames;                     // 16
    SetMusicalMode(kPlayWholeScoreMode);
}
```

`FlagGameOver` is called from exactly two functions — the star pickup above (win,
`GliderPRO/Sources/Interactions.c:944`) and `OffAMortal` (loss) — but from three call
sites, because `OffAMortal` has one for the two-player both-dead case
(`GliderPRO/Sources/Player.c:1500`) and one for single player
(`GliderPRO/Sources/Player.c:1513`). It does *not* record which. `PlayGame` disambiguates by
`mortals < 0` (§6.4 line 35).

### 11.2 `DoGameOver` — the win sequence

```c
void DoGameOver (void)                                   // GameOver.c:60
{
    playing = false;
    SetUpFinalScreen();
    SetPort((GrafPtr)mainWindow);
    ColorRect(&mainWindowRect, 244);
    DoGameOverStarAnimation();
    if (!TestHighScore())
        RedrawSplashScreen();
}
```

Colour index 244 is the 8-bit-palette "background" colour used throughout for
`ColorRect` fills.

```
1.  SetUpFinalScreen():
2.    SetPort(workSrcMap); ColorRect(&workSrcRect, 244)
3.    tempRect = 640 x 460, centred in workSrcRect
4.    LoadScaledGraphic(kMilkywayPictID, &tempRect)         // PICT 1021
5.    textDown = tempRect.top;  if textDown < 0: textDown = 0
6.    tempStr = (*thisHouse)->trailer
7.    count = 0
8.    do:
9.      GetLineOfText(tempStr, count, subStr)               // split on \r
10.     offset = ((thisMac.screen.right - thisMac.screen.left) - TextWidth(subStr,1,subStr[0])) / 2
11.     TextFont(applFont); TextFace(bold); TextSize(12)
12.     ForeColor(blackColor); MoveTo(offset + 1, textDown + 33 + (count * 20)); DrawString(subStr)
13.     ForeColor(whiteColor); MoveTo(offset,     textDown + 32 + (count * 20)); DrawString(subStr)
14.     ForeColor(blackColor)
15.     count++
16.   while subStr[0] > 0
17.   CopyRectWorkToBack(&workSrcRect)
18.   for i in 0..4:                                       // seed 5 falling stars
19.     pages[i].dest = starSrc[0]
20.     QOffsetRect(&pages[i].dest,
21.        workSrcRect.right + RandomInt(workSrcRect.right / 5) + (workSrcRect.right / 4) * i,
22.        RandomInt(workSrcRect.bottom) - workSrcRect.bottom / 2)
23.     pages[i].was = pages[i].dest
24.     pages[i].frame = RandomInt(6)
```
(`GliderPRO/Sources/GameOver.c:76-129`)

The house's `trailer` string is the win message, drawn centred with a 1-pixel black
drop shadow at 20-pixel line spacing starting at `textDown + 32`. Observed trailers:
Sampler `'Congratulations.'`, Demo House `"Excellent!\rThat's the extent of the Demo
House though.  The house\r\"Slumberland\" has over 400 rooms!  Good luck."`,
ImagineHouse PRO II `'Congratulations on beating ImagineHouse PRO II! Hope you\renjoyed
your stay! \r'`.

Note the loop terminates when `GetLineOfText` returns an empty line, so the final
iteration draws nothing and `count` ends one past the last line.

**The `pages[8]` array is shared between the win animation (falling stars) and the
lose animation (falling paper).** Only the first 5 slots are used for the win.

```
1.  DoGameOverStarAnimation():
2.    #define kStarFalls  8
3.    angelDest = angelSrcRect;  QOffsetRect(&angelDest, -96, 0)
4.    noInteruption = true
5.    nextLoop = TickCount() + 2
6.    count = 0; pass = 0
7.    FlushEvents(everyEvent, 0)
8.    while noInteruption:
9.      if (angelDest.left % 32) == 0:
10.        PlayPrioritySound(kMysticSound, kMysticPriority)
11.        which = (angelDest.left / 32) % 5
12.        ZeroRectCorner(&pages[which].dest)
13.        QOffsetRect(&pages[which].dest, angelDest.left, angelDest.bottom)
14.        if count < which + 1: count = which + 1
15.      for i in 0 .. count-1:
16.        pages[i].frame++;  if pages[i].frame >= 6: pages[i].frame = 0
17.        CopyMask(bonusSrcMap, bonusMaskMap, workSrcMap,
18.                 &starSrc[frame], &starSrc[frame], &pages[i].dest)
19.        pages[i].was = pages[i].dest;  pages[i].was.top -= kStarFalls
20.        AddRectToWorkRectsWhole(&pages[i].was)
21.        AddRectToBackRects(&pages[i].dest)
22.        if pages[i].dest.top < workSrcRect.bottom:
23.           QOffsetRect(&pages[i].dest, 0, kStarFalls)          // fall 8 px
24.      if angelDest.left <= workSrcRect.right + 2:
25.         CopyMask(angelSrcMap, angelMaskMap, workSrcMap, &angelSrcRect, &angelSrcRect, &angelDest)
26.         ... add rects ...
27.         QOffsetRect(&angelDest, 2, 0)                         // angel drifts right 2 px
28.         pass = 0
29.      CopyRectsQD();  numWork2Main = 0;  numBack2Work = 0
30.      do { if cmd/opt/shift/ctrl held, or a mouseDown/keyDown arrives:
31.              noInteruption = false } while TickCount() < nextLoop
32.      nextLoop = TickCount() + 2
33.      if pass < 80: pass++                                  // unconditional
34.      else: WaitForInputEvent(5); noInteruption = false
```
(`GliderPRO/Sources/GameOver.c:136-230`)

An angel sprite drifts right across the screen at 2 px/frame, dropping a spinning star
every 32 pixels (5 stars cycling), and each dropped star falls at 8 px/frame. Once the
angel is off the right edge, 80 more frames run, then a 5-tick input wait, then exit.
Paced at 2 ticks/frame (~30 fps), same as gameplay.

`pass++` is *unconditional* (line 33); the "80 frames after the angel leaves" behaviour
comes from `pass = 0` being executed inside the angel-drawing branch (line 28,
`GliderPRO/Sources/GameOver.c:201`) every frame the angel is still on screen. A port
that guards the increment on "angel gone" instead is behaviourally equivalent, but the
source's shape is the one above.

### 11.3 What the win does *not* do

- It does **not** reset the house's object states. The mutated house image stays in
  memory; `gameDirty` remains set from whatever `SetObjectState` calls happened.
- It does **not** write anything to disk (except via `TestHighScore` → `gameDirty`,
  and only if the user later saves).
- It does **not** advance to another house. Glider PRO has no house progression: one
  house per session.

For completeness, the two-player bonus doubling that shows up next to the star code:

```c
case kFoil:                                              // Interactions.c:900-917
    ...
    foilTotal += kFoilSupply;                            // kFoilSupply == 8
    if ((twoPlayerGame) && (!onePlayerLeft))
        foilTotal += kFoilSupply;                        // i.e. doubled, not "* 2"

case kBattery:                                           // Interactions.c:850-870
    ...                                                  // kBatterySupply == 50
    if (batteryTotal > 0) batteryTotal += kBatterySupply; // already battery: add
    else                  batteryTotal  = kBatterySupply; // was helium (or 0): replace
    if ((twoPlayerGame) && (!onePlayerLeft))
        batteryTotal += kBatterySupply;

case kHelium:                                            // Interactions.c:956-976
    ...                                                  // kHeliumSupply == 150
    if (batteryTotal < 0) batteryTotal -= kHeliumSupply; // already helium: add more
    else                  batteryTotal  = -kHeliumSupply;// was battery: replace it
    if ((twoPlayerGame) && (!onePlayerLeft))
        batteryTotal -= kHeliumSupply;
```
(`kBatterySupply`/`kHeliumSupply`/`kFoilSupply` are `#define`d at
`GliderPRO/Sources/Interactions.c:16-19`, not in `GliderDefines.h`. Note that
`batteryTotal` is a *signed* pool: positive = battery thrust, negative = helium, and
picking up the opposite kind **replaces** rather than offsets the current charge.)

---

## 12. Losing: `mortals < 0`

### 12.1 The condition

`OffAMortal` line 5/9/17 (§10.1): `mortals--` then `mortals < 0` (single player) or
`mortals < -1` (two player) ⇒ `FlagGameOver()`. `PlayGame` then sees
`mortals < 0` and calls `DoDiedGameOver()` instead of `DoGameOver()`.

Note that in a two-player game `mortals` is `-1` for the entire "one player left"
phase, so `PlayGame`'s `mortals < 0` test is already true — but `gameOver` is false
during that phase, so the countdown never runs. Only `FlagGameOver` starts it.

### 12.2 `InitDiedGameOver` — setting up the paper animation

```
#define kPageSpacing      40
#define kPageRightOffset  128
#define kPageBackUp       128
#define kNumCountDownFrames 16
#define kPageFrames       14
#define kPagesPictID      1990
#define kPagesMaskID      1989
#define kLettersPictID    1988
#define kMilkywayPictID   1021
typedef struct { Rect dest, was; short frame, counter; Boolean stuck; } pageType;
pageType pages[8];
Rect     pageSrcRect, pageSrc[kPageFrames], lettersSrc[8], angelSrcRect;
RgnHandle roomRgn;
GWorldPtr pageSrcMap, gameOverSrcMap, angelSrcMap, pageMaskMap, angelMaskMap;
short    countDown, stopPages, pagesStuck;
Boolean  gameOver;
```
(`GliderPRO/Sources/GameOver.c:17-22` for the picture/frame IDs, `:25-30` for
`pageType`, `:40-46` for the file-scope state. `kPageSpacing`, `kPageRightOffset` and
`kPageBackUp` are *function*-scope `#define`s inside `InitDiedGameOver`,
`GliderPRO/Sources/GameOver.c:251-253`.)

```
1.  InitDiedGameOver():
2.    create gameOverSrcMap: 25 wide x (32*8 = 256) tall, from kLettersPictID (1988)
3.    create pageSrcMap:     32 wide x (32*14 = 448) tall, from kPagesPictID  (1990)
4.    create pageMaskMap:    same rect, DEPTH 1, from kPagesMaskID (1989)
5.    for i in 0..kPageFrames-1: pageSrc[i] = 32x32 rect at y = 32*i
6.    for i in 0..7:
7.      pages[i].dest = 32x32 centred in thisMac.screen
8.      QOffsetRect(&pages[i].dest, -thisMac.screen.left, -thisMac.screen.top)
9.      if i < 4: QOffsetRect(&pages[i].dest, -kPageSpacing * (4 - i), 0)
10.     else:     QOffsetRect(&pages[i].dest, +kPageSpacing * (i - 3), 0)
11.     QOffsetRect(&pages[i].dest, (thisMac.screen.right - thisMac.screen.left) / -2,
12.                                 (thisMac.screen.right - thisMac.screen.left) / -2)
13.     if (pages[i].dest.left % 2) == 1: QOffsetRect(&pages[i].dest, 1, 0)
14.     pages[i].was = pages[i].dest
15.     pages[i].frame = 0
16.     pages[i].counter = RandomInt(32)
17.     pages[i].stuck = false
18.   for i in 0..7: lettersSrc[i] = 25x32 rect at y = 32*i
19.   roomRgn = NewRgn();  RectRgn(roomRgn, &justRoomsRect)
20.   pagesStuck = 0
21.   stopPages = ((thisMac.screen.bottom - thisMac.screen.top) / 2) - 16
```
(`GliderPRO/Sources/GameOver.c:249-310`)

**Lines 11-12 use the screen *width* for both the horizontal and the vertical
offset.** On a 640×480 screen that is `-320` in both axes, which happens to look
plausible; on a 1024×768 screen it is `-512` vertically, putting the pages well above
the visible area at the start. This is a bug in the original and it is visible on
non-4:3 or large screens. A Go port should reproduce it (or fix it and say so).

Line 13's `% 2` fixup keeps each page's left edge even — a 4-bit-depth `CopyBits`
alignment concern (`thisMac.isDepth == 4` paths elsewhere do the same). In Go with a
32-bit framebuffer it is unnecessary but harmless, and dropping it changes the pages'
X by 1 px.

`stopPages` (line 21) is the Y at which a page stops falling and turns into a letter:
half the screen height minus 16.

### 12.3 `HandlePages` — the per-frame page state machine

```
1.  HandlePages():
2.    for i in 0..7:
3.      if (pages[i].dest.bottom + RandomInt(8)) > stopPages:
4.         pages[i].frame = 0
5.         if !pages[i].stuck:
6.            pages[i].dest.right = pages[i].dest.left + 25    // letters are 25 wide
7.            pages[i].stuck = true
8.            pagesStuck++
9.      else:
10.        if pages[i].frame == 0:
11.           pages[i].counter--
12.           if pages[i].counter <= 0: pages[i].frame = 1
13.        elif pages[i].frame == 7:
14.           pages[i].counter--
15.           if pages[i].counter <= 0:
16.              pages[i].frame = 8
17.              PlayPrioritySound(RandomInt(2) == 0 ? kPaper3Sound : kPaper4Sound, ...)
18.           else:
19.              QOffsetRect(&pages[i].dest, 10, 10)
20.        else:
21.           pages[i].frame++
22.           switch pages[i].frame:
23.             case 5:            QOffsetRect(&dest, 6, 6)
24.             case 6:            QOffsetRect(&dest, 8, 8)
25.             case 7:            QOffsetRect(&dest, 8, 8); counter = RandomInt(4) + 4
26.             case 8:            QOffsetRect(&dest, 8, 8)
27.             case 9:            QOffsetRect(&dest, 8, 8)
28.             case 10:           QOffsetRect(&dest, 6, 6)
29.             case kPageFrames:  QOffsetRect(&dest, 8, 0); frame = 0
30.                                counter = RandomInt(8) + 8
31.                                PlayPrioritySound(RandomInt(2) == 0 ? kPaper1Sound : kPaper2Sound, ...)
```
(`GliderPRO/Sources/GameOver.c:316-394`)

Note `RandomInt(8)` on line 3 makes the stop threshold jittery, so the eight letters
of "GAME OVER" do not all land on the same frame.

### 12.4 `DrawPages`

```
1.  DrawPages():
2.    for i in 0..7:
3.      if pages[i].stuck:
4.         CopyBits(gameOverSrcMap -> workSrcMap, &lettersSrc[i], &pages[i].dest,
5.                  srcCopy, roomRgn)
6.      else:
7.         CopyMask(pageSrcMap, pageMaskMap, workSrcMap,
8.                  &pageSrc[pages[i].frame], &pageSrc[pages[i].frame], &pages[i].dest)
9.      QUnionSimilarRect(&pages[i].dest, &pages[i].was, &pages[i].was)
10.     AddRectToWorkRects(&pages[i].was)
11.     AddRectToBackRects(&pages[i].dest)
12.     CopyRectsQD()
13.     numWork2Main = 0
14.     numBack2Work = 0
15.     pages[i].was = pages[i].dest
```
(`GliderPRO/Sources/GameOver.c:401-435`)

**Lines 12-14 are inside the loop**, so `CopyRectsQD()` runs eight times per frame,
once per page, each time flushing a one-or-two-rect list. That is deliberate (or at
least harmless) but it means the eight letters are composited to the screen serially,
not as one batch. `roomRgn` clips the letters to `justRoomsRect` so they cannot
overwrite the scoreboard.

### 12.5 `DoDiedGameOver`

```
1.  DoDiedGameOver():
2.    userAborted = false
3.    InitDiedGameOver()
4.    CopyRectMainToWork(&workSrcRect)
5.    CopyRectMainToBack(&workSrcRect)
6.    FlushEvents(everyEvent, 0)
7.    nextLoop = TickCount() + 2
8.    while pagesStuck < 8:
9.      HandlePages()
10.     DrawPages()
11.     do:
12.        GetKeys(theKeys)
13.        if cmd/opt/shift/ctrl or a keyDown/mouseDown:
14.           pagesStuck = 8;  userAborted = true
15.     while TickCount() < nextLoop
16.     nextLoop = TickCount() + 2
17.   if roomRgn != nil: DisposeRgn(roomRgn)
18.   DisposeGWorld(pageSrcMap);      pageSrcMap = nil
19.   DisposeGWorld(pageMaskMap);     pageMaskMap = nil
20.   DisposeGWorld(gameOverSrcMap);  gameOverSrcMap = nil
21.   playing = false
22.   if demoGoing:
23.      if !userAborted: WaitForInputEvent(1)
24.   else:
25.      if !userAborted: WaitForInputEvent(10)
26.      TestHighScore()
27.   RedrawSplashScreen()
```
(`GliderPRO/Sources/GameOver.c:443-506`)

- **The screen content at the start is whatever was on screen when the player died**
  (lines 4-5 copy `mainWindow` into both offscreen buffers), so the paper falls over
  the frozen game room. Contrast with the win, which paints a fresh Milky Way
  background.
- **`TestHighScore()` is skipped during a demo** (lines 22-26) but `RedrawSplashScreen()`
  always runs. Compare the win path (`DoGameOver`), which calls `TestHighScore()`
  unconditionally and only calls `RedrawSplashScreen()` if the high-score dialog did
  *not* appear.
- `playing = false` at line 21 is what ends `PlayGame`'s loop. Note it is set *after*
  the animation, whereas `DoGameOver` sets it *first* (`GliderPRO/Sources/GameOver.c:62`).
  Both work because `PlayGame` only checks `playing` at the top of the loop and the
  game-over block is the last thing in the body.

---

## 13. High scores — the post-game hook

```
1.  TestHighScore():
2.    if resumedSavedGame: return false                     // resumed games cannot score
3.    lastHighScore = -1;  placing = -1
4.    for i in 0 .. kMaxScores-1:                           // 10
5.      if theScore > (*thisHouse)->highScores.scores[i]: placing = i; lastHighScore = i; break
6.    if placing == -1: return false
7.    GetHighScoreName(placing + 1) -> names[kMaxScores - 1]
8.    if placing == 0: GetHighScoreBanner() -> highScores.banner
9.    (*thisHouse)->highScores.scores[kMaxScores - 1] = theScore
10.   GetDateTime(&(*thisHouse)->highScores.timeStamps[kMaxScores - 1])
11.   (*thisHouse)->highScores.levels[kMaxScores - 1] = CountRoomsVisited()
12.   SortHighScores()
13.   gameDirty = true
14.   DoHighScores()
15.   return true
```
(`GliderPRO/Sources/HighScores.c:374-426`)

- The new entry is written into **slot 9** (the last), then `SortHighScores()` moves it
  into place. That is why the on-disk arrays are always sorted descending.
- **`levels[i]` is `CountRoomsVisited()`**, i.e. the number of rooms with `visited`
  set:
  ```c
  short CountRoomsVisited (void)                        // House.c:511
  {   count = 0;
      for (i = 0; i < (*thisHouse)->nRooms; i++)
          if ((*thisHouse)->rooms[i].visited) count++;
      return count; }
  ```
  So the high-score "level" column is an exploration metric, and it depends on the
  `visited` flags that `SetObjectsToDefaults` cleared at the start of the game.
  Observed: ImagineHouse PRO II `levels[0] = 108` with 108 rooms currently flagged
  `visited` on disk and `scores[0] = 47000` for `'Paul'`; Demo House
  `levels[0] = 13`, `scores[0] = 7400`, `'Ozma'`; Sampler `levels = (2, 1, 0, …)`,
  `scores = (5200, 5100, 0, …)`, both named `'Your Name'`.
- `resumedSavedGame` (line 2) is set by the `iOpenSavedGame` menu item
  (`GliderPRO/Sources/Menu.c:333`) and cleared by `iNewGame`/`iTwoPlayer`. Since
  `OpenSavedGame()` always fails (§15), it is only ever true briefly and never during
  a game. `HeyYourPissingAHighScore()` (`GliderPRO/Sources/Menu.c:779`, alert
  `kNoHighScoreAlert` 1046) warns the user about this before they resume.
- `gameDirty = true` (line 13) marks the house as needing a save; `QuerySaveChanges`
  will prompt on close (`GliderPRO/Sources/HouseIO.c:596`).
- `ZeroAllButHighestScore()` (`GliderPRO/Sources/HighScores.c:348-367`) resets slots 1..9
  to `"--------------"` / 0 / 0 / 0; `ZeroHighScores()`
  (`GliderPRO/Sources/HighScores.c:323-343`) does all 10 slots *and* copies
  `thisHouseName` into `highScores.banner`. **The "empty" high-score name on disk is the
  literal 14-dash Pascal string, not a zero-length string** — verified in the shipped
  houses (e.g. `Sampler` slots 2..9). A Go writer must emit the dashes to round-trip.
---

## 14. `Trip.c` and `Triggers.c` — the object toggle table and the delayed-action queue

### 14.1 What `Trip.c` actually is

Despite the assignment's expectation, **`Trip.c` has nothing to do with room "trips" or
transitions.** It is a 245-line dispatch table of one-liners that toggle or kick
*dynamic objects* (`dinahs[]` entries) when a switch or a trigger fires. The name
appears to be short for "trip" as in "tripping a switch".

All 14 `Toggle*` functions are called from exactly one place —
`HandleSwitches` in `GliderPRO/Sources/Interactions.c:1090-1148` — and all 8
`Trigger*` functions from exactly one place, `FireTrigger` in
`GliderPRO/Sources/Triggers.c:128-171`. `UpdateOutletsLighting` is called from
`RedrawRoomLighting` (`GliderPRO/Sources/RoomGraphics.c:453`).

Complete table:

| Function | Line | Body |
|---|---:|---|
| `ToggleToaster` | 22 | `active = !active` |
| `ToggleMacPlus` | 29 | `active = !active`; `timer = active ? 40 : 10` |
| `ToggleTV` | 40 | `active = !active`; if `hasQT && hasMovie && tvInRoom && tvWithMovieNumber == index`: `active` ⇒ `GoToBeginningOfMovie`+`StartMovie`+`tvOn = true`, else `StopMovie`+`tvOn = false`; then `timer = 4` unconditionally |
| `ToggleCoffee` | 62 | `active = !active`; `timer = 4` |
| `ToggleOutlet` | 70 | `active = !active` |
| `ToggleVCR` | 77 | `active = !active`; `timer = 4` |
| `ToggleStereos` | 85 | `if (dinahs[index].timer == 0) { active = !active; timer = 4; }` — **only when the timer has expired** |
| `ToggleMicrowave` | 96 | `active = !active`; `timer = 4` |
| `ToggleBalloon` | 104 | `active = !active` |
| `ToggleCopter` | 111 | `active = !active` |
| `ToggleDart` | 118 | `active = !active` |
| `ToggleBall` | 125 | `active = !active` |
| `ToggleDrip` | 132 | `active = !active` |
| `ToggleFish` | 139 | `active = !active` |
| `TriggerSwitch` | 146 | `HandleSwitches(&hotSpots[who])` — a trigger that fires a switch recurses into the switch handler |
| `TriggerToast` | 153 | `if (!moving)`: if `active` then `vVel = -count; frame = 0; moving = true;` + `kToastLaunchSound`, else `frame = timer` |
| `TriggerOutlet` | 171 | `if (position == 0)`: if `active` then `position = 1; timer = kLengthOfZap` (30) + `kZapSound`, else `timer = count` |
| `TriggerDrip` | 188 | `if (!moving && timer > 7) timer = 7` |
| `TriggerFish` | 196 | `if (active && !moving)`: `whole = dest; moving = true; frame = 4;` + `kFishOutSound` |
| `TriggerBalloon` | 209 | `if (!moving) timer = kStartSparkle + 1` = 5 |
| `TriggerCopter` | 218 | `if (!moving) timer = kStartSparkle + 1` = 5 |
| `TriggerDart` | 227 | `if (!moving) timer = kStartSparkle + 1` = 5 |
| `UpdateOutletsLighting` | 235 | `for i < numDynamics: if (dinahs[i].type == kOutlet && dinahs[i].room == room) dinahs[i].hVel = nLights` — the outlet's `hVel` field is reused as its light count |

`kLengthOfZap` = 30 and `kStartSparkle` = 4 (`GliderPRO/Headers/GliderDefines.h:546`,
`:545`).

Every one of these calls passes `masterObjects[...].dynaNum`, and **`dynaNum` is
overloaded**: for a genuine dynamic object it is the `dinahs[]` index, but for the six
switch types `DrawAllObjects` deliberately stores the *`hotSpots[]`* index into it
(`dynamicNum = masterObjects[i].hotNum`, `GliderPRO/Sources/ObjectDrawAll.c:518`, `:531`,
`:544`, `:557`, `:570`, `:574`), which is why `TriggerSwitch`'s `hotSpots[who]` is
correct rather than an out-of-bounds read. A Go port should keep the two indices in
separate fields and pick per call site.

**None of this state is persistent.** `dinahs[]` is rebuilt from scratch on every room
change, so every toggled appliance reverts to its `initial` value when you leave the
room and come back. Contrast the *switch itself*, whose `state` byte lives in the house
and does persist. That asymmetry is intentional-looking but produces the observable
oddity that a light switch stays flipped while the machine it controls resets.

### 14.2 The trigger queue

```c
#define kMaxTriggers  16
typedef struct {
    short   object, room;      // house-absolute address of the TARGET
    short   index, timer;      // masterObjects index of the TRIGGER; countdown in frames
    short   what;              // cached `what` of the target
    Boolean armed;
} trigType, *trigPtr;
trigType triggers[kMaxTriggers];
```
(`GliderPRO/Sources/Triggers.c:11-28`)

```
1.  ArmTrigger(who):                                   // who is a hotSpot
2.    if who->stillOver: return                         // debounce: already standing on it
3.    where = FindEmptyTriggerSlot()                    // first slot with !armed, else -1
4.    if where != -1:
5.       whoLinked = who->who                           // masterObjects index of the trigger
6.       triggers[where].room   = masterObjects[whoLinked].roomLink
7.       triggers[where].object = masterObjects[whoLinked].objectLink
8.       triggers[where].index  = whoLinked
9.       triggers[where].timer  = masterObjects[whoLinked].theObject.data.e.delay * 3
10.      triggers[where].what   = masterObjects[triggers[where].object].theObject.what
11.      triggers[where].armed  = true
12.   who->stillOver = true
```
(`GliderPRO/Sources/Triggers.c:34-55`)

**Line 10 is a bug.** `triggers[where].object` is an *object number within a room*
(0..23), but it is used to index `masterObjects[]`, which is indexed by
*master-object number* (0..`numMasterObjects-1` across the whole 3×3 neighbourhood).
The resulting `what` is garbage. Fortunately `triggers[].what` is written and never
read anywhere — grep confirms `\.what` on a `trigType` appears only at this one
assignment — so the bug is inert. A Go port should simply not have the field.

Line 9: the delay is `data.e.delay * 3` frames, i.e. `delay` × 0.1 s at 30 fps.
`data.e` is the **`switchType`** arm (`GliderPRO/Headers/GliderStructs.h:45-52`), whose
`delay` is a **`short` at object offset `+6`** (after the 2-byte `what` and the 4-byte
`topLeft`) — *not* the 1-byte `applianceType.delay`. So the trigger delay is a
big-endian 16-bit field on disk and the timer can legitimately exceed 255 frames.

```
1.  HandleTriggers():                                   // once per frame from PlayGame
2.    for i in 0..15:
3.      if triggers[i].armed:
4.         triggers[i].timer--
5.         if triggers[i].timer <= 0:
6.            triggers[i].timer = 0
7.            triggers[i].armed = false
8.            FireTrigger(i)
```
(`GliderPRO/Sources/Triggers.c:79-96`)

Note a trigger with `delay == 0` gets `timer = 0`, is decremented to -1 on the very next
frame, and fires immediately.

`FireTrigger` (`GliderPRO/Sources/Triggers.c:100-194`) has two branches:

- **`masterObjects[triggerIs].localLink != -1`** — the target is in the current 3×3
  neighbourhood, so it has a live `masterObjects` entry. Dispatch on
  `masterObjects[triggeredIs].theObject.what` across 14 cases: grease (`kGreaseRt`,
  `kGreaseLf` → `SetObjectState(..., kForceOn, ...)` then `SpillGrease`), the six
  switches (→ `TriggerSwitch`), `kSoundTrigger` and `kGuitar` (→ `kChordSound`),
  `kToaster` (→ `TriggerToast`), `kCoffee` (→ `kCoffeeSound`), `kOutlet`, `kBalloon`,
  `kCopterLf`/`kCopterRt`, `kDartLf`/`kDartRt`, `kDrip`, `kFish`.
- **`localLink == -1`** — the target is in a room *outside* the neighbourhood. This
  branch locks the house handle, reads
  `(*thisHouse)->rooms[triggers[i].room].objects[triggers[i].object].what`, and handles
  **only grease**. Everything else is silently dropped. So a trigger wired to a distant
  room does nothing unless the target is a grease jar.

  **And in that branch there is a second live bug:**
  ```c
  triggeredIs = masterObjects[triggerIs].localLink;   // == -1 by construction
  ...
  SpillGrease(masterObjects[triggeredIs].dynaNum,     // masterObjects[-1]
              masterObjects[triggeredIs].hotNum);
  ```
  (`GliderPRO/Sources/Triggers.c:178` for the `localLink` read, `:187-188` for the use) —
  a guaranteed out-of-bounds read whenever a
  remote grease trigger fires. In Go this is a panic. Guard it.

`ZeroTriggers()` (`GliderPRO/Sources/Triggers.c:198-204`) clears all 16 `armed` flags. It
has exactly one caller: `DrawLocale` (`GliderPRO/Sources/RoomGraphics.c:55`, in the
transient-clear block §8.6), which every room transition goes through, so **a pending
trigger is cancelled by leaving the room**. It also covers the start of a game, because
`NewGame` itself calls `DrawLocale()` (`GliderPRO/Sources/Play.c:163`) before the gliders
start fading in — so there is no cross-game leak of armed triggers, and a Go port gets
the same behaviour for free by clearing the slice inside its `DrawLocale` equivalent.

---

## 15. `DynamicMaps.c` — `savedMaps` and the dynamic layer

### 15.1 What `savedMaps` is for

Bonus objects (clocks, batteries, bands, helium, foil, stars, grease) can be *removed*
mid-game. Rather than redrawing the whole room background when one is taken, the game
pre-captures the patch of background that each bonus covers into its own tiny GWorld;
"erasing" the bonus is then a single `CopyBits` of that patch back over it.

```c
typedef struct { Rect dest; GWorldPtr map; short where; short who; } savedType, *savedPtr;
savedType savedMaps[kMaxSavedMaps];          // Objects.c:73, kMaxSavedMaps == 24
short     numSavedMaps;                      // DynamicMaps.c:32
```
(`savedType` at `GliderPRO/Headers/GliderStructs.h:227-233`, `kMaxSavedMaps` at
`GliderPRO/Headers/GliderDefines.h:260`, the array at `GliderPRO/Sources/Objects.c:73`)

`where` is the **house-absolute room number** (`localNumbers[neighbor]`), `who` is the
**object number within that room** (0..23). Together they are the key used by
`ReBackUpSavedMap` and `RestoreFromSavedMap` to find the right slot.

Note `savedMaps` is *also* allocated dynamically at
`GliderPRO/Sources/StructuresInit2.c:233` (`NewPtr(sizeof(savedType) * kMaxSavedMaps)`),
and its memory budget is accounted at `GliderPRO/Sources/Environ.c:650`. **The two
declarations disagree on the type.** `Objects.c:73` *defines* an array
(`savedType savedMaps[kMaxSavedMaps]`), and `DynamicMaps.c:36`, `Grease.c:30` and
`Render.c:54` all re-declare it correctly as `extern savedType savedMaps[]` — but
`StructuresInit2.c:40` declares `extern savedPtr savedMaps;` (a *pointer*). The
un-typechecked cross-TU link means `savedMaps = NewPtr(...)` at `:233` writes the heap
block's address into the first four bytes of the real array, i.e. over
`savedMaps[0].dest.top` and `.left`; the subsequent `savedMaps[i].map = nil` loop at
`:237-238` then nils the *heap* block, and that allocation (24 × 16 = 384 bytes) is never
used again.
The corruption is benign only because `NilSavedMaps()` re-nils `map` and
`BackUpToSavedMap` always writes `dest` before anything reads it. A Go port has one
slice and none of this.

### 15.2 The four operations

```c
void NilSavedMaps (void)                                  // DynamicMaps.c:46
{   for (i = 0; i < kMaxSavedMaps; i++)
    {   if (savedMaps[i].map != nil) { DisposeGWorld(savedMaps[i].map); savedMaps[i].map = nil; }
        savedMaps[i].where = -1;
        savedMaps[i].who   = -1; }
    numSavedMaps = 0; }

short BackUpToSavedMap (Rect *theRect, short where, short who)   // DynamicMaps.c:70
{   if (numSavedMaps >= kMaxSavedMaps) return (-1);              // <== EXHAUSTION
    mapRect = *theRect; ZeroRectCorner(&mapRect);
    savedMaps[numSavedMaps].dest = *theRect;
    CreateOffScreenGWorld(&savedMaps[numSavedMaps].map, &mapRect, kPreferredDepth);
    CopyBits(backSrcMap, savedMaps[numSavedMaps].map, theRect, &mapRect, srcCopy, nil);
    savedMaps[numSavedMaps].where = where;
    savedMaps[numSavedMaps].who   = who;
    numSavedMaps++;
    return (numSavedMaps - 1); }

short ReBackUpSavedMap (Rect *theRect, short where, short who)    // DynamicMaps.c:100
{   for (i = 0; i < numSavedMaps; i++)
        if ((savedMaps[i].where == where) && (savedMaps[i].who == who))
        {   mapRect = *theRect; ZeroRectCorner(&mapRect);
            CopyBits(backSrcMap, savedMaps[i].map, theRect, &mapRect, srcCopy, nil);
            return (i); }
    return (-1); }

void RestoreFromSavedMap (short where, short who, Boolean doSparkle)   // DynamicMaps.c:131
{   for (i = 0; i < numSavedMaps; i++)
        if ((savedMaps[i].where == where) && (savedMaps[i].who == who) && (savedMaps[i].map != nil))
        {   mapRect = savedMaps[i].dest; ZeroRectCorner(&mapRect);
            CopyBits(savedMaps[i].map, backSrcMap, &mapRect, &savedMaps[i].dest, srcCopy, nil);
            CopyBits(savedMaps[i].map, workSrcMap, &mapRect, &savedMaps[i].dest, srcCopy, nil);
            AddRectToWorkRects(&savedMaps[i].dest);
            if (doSparkle)
            {   bounds = savedMaps[i].dest;
                QOffsetRect(&bounds, -playOriginH, -playOriginV);
                AddSparkle(&bounds);
                PlayPrioritySound(kFadeOutSound, kFadeOutPriority); }
            break; } }
```

The `where`/`who` linear search is O(24) and unindexed. There is no removal: a slot
stays occupied for the lifetime of the room.

### 15.3 The exhaustion behaviour is observable

`BackUpToSavedMap` returns -1 when all 24 slots are used, and **every caller treats -1
as "don't draw the object at all"**:

```c
case kRedClock:
GetObjectRect(&thisObject, &itsRect);
OffsetRectRoomRelative(&itsRect, neighbor);
if (SectRect(&itsRect, &testRect, &whoCares))
{   if (redraw) legit = ReBackUpSavedMap(&itsRect, localNumbers[neighbor], i);
    else        legit = BackUpToSavedMap(&itsRect, localNumbers[neighbor], i);
    if (legit != -1)
        DrawRedClock(&itsRect); }
break;
```
(`GliderPRO/Sources/ObjectDrawAll.c:286-298`)

So **the 25th bonus visible in the current 3×3 neighbourhood is invisible** — and,
because the hot spot is created independently by `CreateActiveRects`, it is still
collectible. A Go port that uses a growable slice will make invisible bonuses visible
and change the look of dense rooms in `Land of Illusion` / `Slumberland` / `Teddy World`.
**Keep the 24 limit if you want fidelity.**

Complete list of object types that consume a `savedMaps` slot:

| Type | Value | Site |
|---|---:|---|
| `kRedClock` | 33 / 0x21 | `ObjectDrawAll.c:286` |
| `kBlueClock` | 34 / 0x22 | `ObjectDrawAll.c:300` |
| `kYellowClock` | 35 / 0x23 | `ObjectDrawAll.c:314` |
| `kCuckoo` | 36 / 0x24 | `ObjectDrawAll.c:328` |
| `kPaper` | 37 / 0x25 | `ObjectDrawAll.c:349` |
| `kBattery` | 38 / 0x26 | `ObjectDrawAll.c:350` |
| `kBands` | 39 / 0x27 | `ObjectDrawAll.c:351` |
| `kHelium` | 46 / 0x2E | `ObjectDrawAll.c:352` |
| `kFoil` | 42 / 0x2A | `ObjectDrawAll.c:408` |
| `kStar` | 44 / 0x2C | `ObjectDrawAll.c:426` |
| `kGreaseRt` / `kGreaseLf` | 40 / 0x28, 41 / 0x29 | `ObjectDrawAll.c:366`, `:387`; `Grease.c:219` (via `AddGrease`, 32 × 108 rect) |

`kInvisBonus` and `kSlider` are explicitly `break;` with no drawing and no saved map
(`GliderPRO/Sources/ObjectDrawAll.c:422-424`), which is why an invisible bonus does not
sparkle when taken.

### 15.4 The rest of the dynamic layer, with its hard limits

Every one of these arrays is fixed-size, zeroed on room entry by
`ZeroFlamesAndTheLike()` (`GliderPRO/Sources/DynamicMaps.c:787-797`), and silently
drops overflow.

| Array | Limit | Constant | Swatch size × frames | `Add*` | `BackUp*` |
|---|---:|---|---|---:|---:|
| `sparkles` | 3 | `kMaxSparkles` (`:251`) | — | `:169` | — |
| `flyingPoints` | — | — | — | `:199` | — |
| `flames` (candles/tapers) | 20 | `kMaxCandles` (`:255`) | 16 × 15, 5 frames (`kNumCandleFlames`) | `:316` | `:263` |
| `tikiFlames` | 8 | `kMaxTikis` (`:256`) | 8 × 10, 5 frames | `:400` | `:349` |
| `bbqCoals` | 8 | `kMaxCoals` (`:257`) | 32 × 9, 4 frames (`kNumBBQCoals`) | `:486` | `:435` |
| `pendulums` | 8 | `kMaxPendulums` (`:258`) | 32 × 28, 3 frames (`kNumPendulums`) | `:570` | `:521` |
| `theStars` | 4 | `kMaxStars` (`:263`) | 32 × 31, 6 frames | `:662` | `:612` |
| `shreds` | 4 | `kMaxShredded` (`:264`) | — | `:730` | — |
| `grease` | 16 | `kMaxGrease` (`:262`) | 32 × 27 (×4 for the map) | `Grease.c:206` | `Grease.c:219` |

`AddPendulum` seeds the shared `clockFrame = 10` (`GliderPRO/Sources/DynamicMaps.c:570+`).
`AddStar` seeds each star's `mode = RandomInt(6)` so the six-frame spin is
desynchronised.

**`AddAShreddedGlider` has an off-by-one:**
```c
void AddAShreddedGlider (Rect *theRect)               // DynamicMaps.c:730
{
    if (numShredded > kMaxShredded)      // <== should be >=
        return;
    ...
    numShredded++;
}
```
(`GliderPRO/Sources/DynamicMaps.c:730-742`) With `kMaxShredded` = 4 and
`shredType shreds[kMaxShredded]`, the guard admits `numShredded == 4` and writes
`shreds[4]` — one element past the end. In the original this scribbles on whatever
follows in the heap block; in Go it panics. Since only one glider shreds at a time and
`RemoveShreds()` runs from `OffAMortal` (`GliderPRO/Sources/Player.c:1490`), reaching 5
requires shredding both gliders repeatedly without an intervening death, which is hard
but not obviously impossible. Use `>=`.

`StopPendulum(where, who)` (`:700`) and `StopStar(where, who)` (`:715`) deactivate a
single entry by `(where, who)` key — used when a clock or star is collected so the
animation stops even though the array slot stays allocated.

### 15.5 `DrawLighting` is a stub

```c
void DrawLighting (void)                              // RoomGraphics.c:422
{
    if (numLights == 0)
        return;
    else
    {
        // for future construction
    }
}
```
(`GliderPRO/Sources/RoomGraphics.c:422-430`)

**Darkness is not rendered by dimming anything.** A dark room is dark because
`DrawARoomsObjects` / `DrawRoomBackground` consult `isLit` and simply *skip drawing*
most objects (see the `if (isLit)` guards throughout `ObjectDrawAll.c`, e.g.
`:266`, `:279`). A Go port must implement darkness the same way — per-object
suppression, not a post-process — or the rooms will not match.

`RedrawRoomLighting()` (`GliderPRO/Sources/RoomGraphics.c:434-461`) is the transition
between lit and unlit and runs only when the lit-ness actually flipped:

```
1.  RedrawRoomLighting():
2.    roomV   = (*thisHouse)->rooms[thisRoomNumber].floor
3.    wasLit  = numLights > 0
4.    numLights = GetNumberOfLights(localNumbers[kCentralRoom])
5.    isLit   = numLights > 0
6.    if wasLit != isLit:
7.       DrawRoomBackground(localNumbers[kCentralRoom], kCentralRoom, roomV)
8.       DrawARoomsObjects(kCentralRoom, true)          // redraw = true -> ReBackUpSavedMap
9.       DrawLighting()                                 // no-op
10.      UpdateOutletsLighting(localNumbers[kCentralRoom], numLights)
11.      if numNeighbors > 3: DrawFloorSupport()
12.      RestoreWorkMap()
13.      AddRectToWorkRects(&localRoomsDest[kCentralRoom])
14.      shadowVisible = IsShadowVisible()
```

Line 8's `redraw = true` is what routes the bonus cases through `ReBackUpSavedMap`
instead of `BackUpToSavedMap`, re-capturing the *new* (lit or unlit) background into the
existing slots without allocating more. **That is the whole reason
`ReBackUpSavedMap` exists.** Only the central room is redrawn — neighbour rooms keep
their old lighting until you walk into them.

---

## 16. Saved games: a complete, and completely dead, subsystem

### 16.1 The two formats

There are two saved-game representations in the sources.

**(a) Embedded in the house file** — `houseType.savedGame`, a 40-byte `gameType` at
offset 820, plus the `hasGame` Boolean at 860:

| Offset (in `gameType`) | House offset | Field | Type | Size |
|---:|---:|---|---|---:|
| 0 | 820 | `version` | `short` | 2 |
| 2 | 822 | `wasStarsLeft` | `short` | 2 |
| 4 | 824 | `timeStamp` | `long` | 4 |
| 8 | 828 | `where` | `Point` (v then h) | 4 |
| 12 | 832 | `score` | `long` | 4 |
| 16 | 836 | `unusedLong` | `long` | 4 |
| 20 | 840 | `unusedLong2` | `long` | 4 |
| 24 | 844 | `energy` | `short` | 2 |
| 26 | 846 | `bands` | `short` | 2 |
| 28 | 848 | `roomNumber` | `short` | 2 |
| 30 | 850 | `gliderState` | `short` | 2 |
| 32 | 852 | `numGliders` | `short` | 2 |
| 34 | 854 | `foil` | `short` | 2 |
| 36 | 856 | `unusedShort` | `short` | 2 |
| 38 | 858 | `facing` | `Boolean` | 1 |
| 39 | 859 | `showFoil` | `Boolean` | 1 |
(`GliderPRO/Headers/GliderStructs.h:116-134`, comment `// total = 40`)

`SaveGame(Boolean doSave)` (`GliderPRO/Sources/SavedGames.c:303-351`) is the only writer.
The field-by-field mapping, verbatim:

| `savedGame` field | Source expression |
|---|---|
| `version` | `kSavedGameVersion` (0x0200, `#define`d at `SavedGames.c:14`) |
| `wasStarsLeft` | `numStarsRemaining` |
| `timeStamp` | `GetDateTime(&stamp)` — **the current wall clock, NOT the house's `timeStamp`** |
| `where.h` | `theGlider.dest.left` |
| `where.v` | `theGlider.dest.top` |
| `score` | `theScore` |
| `unusedLong`, `unusedLong2` | `0L` |
| `energy` | `batteryTotal` |
| `bands` | `bandsTotal` |
| `roomNumber` | `thisRoomNumber` |
| `gliderState` | `theGlider.mode` |
| `numGliders` | `mortals` |
| `foil` | `foilTotal` |
| `unusedShort` | `0` |
| `facing` | `theGlider.facing` |
| `showFoil` | `showFoil` |
| (`hasGame`) | `true` |

It early-returns `if (twoPlayerGame)`, and it calls
`WriteHouse(theMode == kEditMode)` to flush the whole house image to disk.

**Critically, `SaveGame` saves no per-room state.** No `visited` flags, no object
`state` bytes. Since it writes the entire house image, the *current* mutated
`visited`/`state` bytes ride along in the same file by accident — which actually makes
the embedded format work, but only because the resume path re-reads the same file.
It also means a saved game permanently overwrites the house's pristine object states on
disk. Note `SaveGame`'s `timeStamp` is the wall clock at save time, whereas
`OpenSavedGame` validates the saved timestamp against `(*thisHouse)->timeStamp`
(the house's creation/modification stamp) — **these can never match**, so even a
successfully-written embedded save would be rejected on load. See §16.3.

**(b) An external `'gliG'` file** — `game2Type` followed by `nRooms` × `savedRoom`:

```c
typedef struct { FSSpec house; short version; short wasStarsLeft; long timeStamp;
                 Point where; long score; long unusedLong; long unusedLong2;
                 short energy; short bands; short roomNumber; short gliderState;
                 short numGliders; short foil; short nRooms;
                 Boolean facing; Boolean showFoil;
                 savedRoom savedData[]; } game2Type, *gamePtr;   // comment: total = 114
typedef struct { short unusedShort; Byte unusedByte; Boolean visited;
                 objectType objects[kMaxRoomObs]; } savedRoom;   // total = 292
```
(`GliderPRO/Headers/GliderStructs.h:136-163`)

`FSSpec` is `{short vRefNum; long parID; Str63 name;}` = 70 bytes under
`align=mac68k`. The fixed part therefore measures
70+2+2+4+4+4+4+4+2+2+2+2+2+2+2+1+1 = **110 bytes**, not the 114 the comment claims
(the author counted 4 bytes for the flexible array member). The file size is
`sizeof(game2Type) + sizeof(savedRoom) * nRooms`
(`GliderPRO/Sources/SavedGames.c:55`), so the exact value of `sizeof(game2Type)`
under the shipping compiler's alignment determines the layout. **This is unresolvable
from source alone and irrelevant in practice**, because no such file exists and the
code that writes it is commented out. A Go port should invent its own format.

`savedRoom` carries exactly what the embedded format omits: `visited` and all 24
`objectType`s (so all `state` bytes). Type/creator `'gliG'` / `'ozm5'`
(`GliderPRO/Sources/SavedGames.c:123`).

### 16.2 Every save/resume code path is dead

| Function | State | Evidence |
|---|---|---|
| `SaveGame2()` | **empty stub** — entire body inside `/* ... */` (`:33`-`:147`) | `GliderPRO/Sources/SavedGames.c:30-148` |
| `OpenSavedGame()` | **`return false;` on the first line**, everything after is unreachable | `GliderPRO/Sources/SavedGames.c:167-169` (`return false; // TEMP fix this iwth NavServices`) |
| `SaveGame(Boolean)` | live code, **never called** — the only call site is commented out | `GliderPRO/Sources/Menu.c:458` `//    SaveGame(false);` |
| `NewGame(kResumeGameMode)` | unreachable: the only caller passing `kResumeGameMode` is guarded by `OpenSavedGame()` succeeding | `GliderPRO/Sources/Menu.c:317-325` (call at `:323`) |
| `smallGame` (`gameType`) | never populated at runtime | populated only in `OpenSavedGame`'s dead tail, `SavedGames.c:261-275` |
| `houseType.hasGame` | **written in 3 places, read in 0** | written `House.c:139` (`false`), `SavedGames.c:337` (`true`), `SavedGames.c:341` (`false`); no reads |
| Cmd-S / Cmd-Q "save game?" | calls `SaveGame2()`, i.e. nothing | `GliderPRO/Sources/Input.c:62` (Q, behind `QuerySaveGame()`) and `:68` (S) |

So in Glider PRO 1.0.4 as shipped: **you cannot save a game, and you cannot resume
one.** Cmd-S flashes `"Saving Game…"` on the scoreboard
(`RefreshScoreboard(kSavingTitleMode)`, `GliderPRO/Sources/Input.c:67`) and then does
nothing. Quitting a game offers to save (`QuerySaveGame`, alert `kSaveGameAlert` 1041,
`GliderPRO/Sources/Input.c:383`) and then does nothing.

### 16.3 The intended validation order (from the dead code)

Preserved for a Go port that wants to implement resume properly:

```
1.  OpenSavedGame():
2.    return false                                    // <== 1.0.4 stops here
3.    StandardGetFile(nil, 1, {'gliG'}, &reply)
4.    read the whole file into a gamePtr
5.    if !EqualString(savedGame->house.name, thisHouseName, true, true):
6.       SavedGameMismatchError(savedGame->house.name)   // alert kSavedGameErrorAlert 1044
7.       return false
8.    else if (*thisHouse)->timeStamp != savedGame->timeStamp:
9.       YellowAlert(kYellowSavedTimeWrong, 0); return false
10.   else if savedGame->version != kSavedGameVersion:   // 0x0200, SavedGames.c:14
11.      YellowAlert(kYellowSavedVersWrong, kSavedGameVersion); return false
12.   else if savedGame->nRooms != (*thisHouse)->nRooms:
13.      YellowAlert(kYellowSavedRoomsWrong, savedGame->nRooms - (*thisHouse)->nRooms)
14.      return false
15.   smallGame.wasStarsLeft = savedGame->wasStarsLeft
16.   smallGame.where.h      = savedGame->where.h
17.   smallGame.where.v      = savedGame->where.v
18.   smallGame.score        = savedGame->score
19.   smallGame.unusedLong   = savedGame->unusedLong
20.   smallGame.unusedLong2  = savedGame->unusedLong2
21.   smallGame.energy       = savedGame->energy
22.   smallGame.bands        = savedGame->bands
23.   smallGame.roomNumber   = savedGame->roomNumber
24.   smallGame.gliderState  = savedGame->gliderState
25.   smallGame.numGliders   = savedGame->numGliders
26.   smallGame.foil         = savedGame->foil
27.   smallGame.unusedShort  = 0
28.   smallGame.facing       = savedGame->facing
29.   smallGame.showFoil     = savedGame->showFoil
30.   for r in 0 .. savedGame->nRooms-1:
31.      (*thisHouse)->rooms[r].visited = savedGame->savedData[r].visited
32.      for i in 0..23:
33.         (*thisHouse)->rooms[r].objects[i] = savedGame->savedData[r].objects[i]
34.   return true
```
(`GliderPRO/Sources/SavedGames.c:167-296`; the validation chain is `:227-258`, the
`smallGame` copy `:261-275`, the per-room copy `:277-284`.)

Note **`smallGame.version` and `smallGame.timeStamp` are never assigned** by this path —
the two fields are validated off the file image and then dropped, so `smallGame.version`
stays whatever it was (zero, from BSS). Nothing reads them, but a port should not assume
`smallGame` is a faithful copy of the file header.

Line 8 compares against the *house's* timestamp, confirming that a resume is only
valid for the exact house build it was made from. Combined with §16.1, note that
`SaveGame` (format (a)) stores the wall clock instead, which is why the two shipped
embedded saves fail this test — see below.

### 16.4 Empirical: the two shipped embedded saved games

`hasGame == 1` in exactly 2 of the 22 shipped houses. Raw 40 bytes at offset 820:

**ImagineHouse PRO II**
```
01 00 00 03 ab 09 1a fd 00 3d ff f6 00 00 17 0c
00 00 00 00 00 00 00 00 ff 6a 00 17 00 2d 00 00
00 05 00 08 00 00 01 01
```
| Field | Value |
|---|---|
| `version` | `0x0100` |
| `wasStarsLeft` | 3 |
| `timeStamp` | `0xAB091AFD` = 2869370621 |
| `where` | v = 61, h = -10 |
| `score` | 5900 |
| `energy` | -150 (helium: `batteryTotal` is negative for helium) |
| `bands` | 23 |
| `roomNumber` | 45 → room name `'Grin and Bear it'`, floor 1, suite 70 |
| `gliderState` | 0 (`kGliderNormal`) |
| `numGliders` | 5 |
| `foil` | 8 |
| `unusedShort` | 0 |
| `facing` | 1 (`kFaceRight`) |
| `showFoil` | 1 |

House `timeStamp` = `0x2C1DC93D`. **Mismatch** with the save's `0xAB091AFD`.

**Titanic**
```
01 00 00 01 ab ad e4 37 00 f0 00 55 00 00 12 5c
00 00 00 00 00 00 00 00 00 00 00 00 00 68 00 00
00 02 00 00 00 00 01 00
```
| Field | Value |
|---|---|
| `version` | `0x0100` |
| `wasStarsLeft` | 1 |
| `timeStamp` | `0xABADE437` |
| `where` | v = 240, h = 85 |
| `score` | 4700 |
| `energy` | 0 |
| `bands` | 0 |
| `roomNumber` | 104 → room name `'Oceanic Depths'`, floor -1, suite 72 |
| `gliderState` | 0 |
| `numGliders` | 2 |
| `foil` | 0 |
| `facing` | 1 |
| `showFoil` | 0 |

House `timeStamp` = `0x2D0246FE`. **Mismatch.**

**Both would be rejected twice over** by the §16.3 validation: `version` is `0x0100`,
not `kSavedGameVersion` = `0x0200`, and both timestamps differ from their house's. They
are relics of Glider 4.0 / an earlier Glider PRO beta. A Go port should ignore
`houseType.savedGame` entirely when loading a shipped house, and must not treat
`hasGame == 1` as "there is a resumable game here".

### 16.5 What `NewGame(kResumeGameMode)` *would* do

For completeness, the branches in `NewGame` that key off `mode`
(`GliderPRO/Sources/Play.c:74-278`):

| Statement | `kNewGameMode` (1) | `kResumeGameMode` (0) |
|---|---|---|
| room selection | `SetHouseToFirstRoom()` → `ForceThisRoom(GetFirstRoomNumber())` | `SetHouseToSavedRoom()` → `ForceThisRoom(smallGame.roomNumber)` |
| `SetObjectsToDefaults()` | **called** (clears `visited`, resets every `state` from `initial`) | **not called** |
| `numStarsRemaining` | `CountStarsInHouse()` | `smallGame.wasStarsLeft` |
| `theScore` | `0` | `smallGame.score` |
| `mortals` | `kInitialGliders` (2), doubled to 4 if `twoPlayerGame` | `smallGame.numGliders` |
| `batteryTotal` | `0` | `smallGame.energy` |
| `bandsTotal` | `0` | `smallGame.bands` |
| `foilTotal` | `0` | `smallGame.foil` |
| `showFoil` | `false` | `smallGame.showFoil` |
| glider start point | `WhereDoesGliderBegin(&initialPt, mode)` → `(*thisHouse)->initial` | → `smallGame.where` |
| glider `facing` | `kFaceRight` | `smallGame.facing` |

```c
void WhereDoesGliderBegin (Rect *theRect, short mode)     // House.c:224
{
    Point initialPt;
    if (mode == kResumeGameMode)
        initialPt = smallGame.where;
    else if (mode == kNewGameMode)
        initialPt = (*thisHouse)->initial;
    QSetRect(theRect, 0, 0, kGliderWide, kGliderHigh);     // 48 x 20
    QOffsetRect(theRect, initialPt.h, initialPt.v);
}
```
**`initialPt` is uninitialised if `mode` is neither 0 nor 1** — no `else`, no default.
Harmless today (only 0 and 1 are ever passed) but a Go port must pick a defined
behaviour.

---

## 17. The authoritative persistence map

Where every piece of progression state lives, and what its lifetime is.

### 17.1 In the house image (survives room change, death, and — if the house is saved — process exit)

| State | Location | Written by | Reset by |
|---|---|---|---|
| room `visited` flag | `rooms[n]` offset +33, 1 byte | `HandleRoomVisitation` — the house at `Transit.c:440`, the `thisRoom` copy at `:443` | `SetObjectsToDefaults` (`Play.c:619`) |
| blower on/off (`data.a.state`) | object +9 | `SetObjectState` | `SetObjectsToDefaults` → `= data.a.initial` |
| bonus taken (`data.c.state`) | object +10 | `SetObjectState` | `SetObjectsToDefaults` → `= data.c.initial` |
| light on/off (`data.f.state`) | object +11 | `SetObjectState` | `SetObjectsToDefaults` → `= data.f.initial` |
| appliance on/off (`data.g.state`) | object +11 | `SetObjectState` | `SetObjectsToDefaults` → `= data.g.initial` |
| enemy active (`data.h.state`) | object +11 | `SetObjectState` | `SetObjectsToDefaults` → `= data.h.initial` |
| deluxe-transport enabled nibble | `kDeluxeTrans` `data.d.wide` low nibble | `SetObjectState` (`Objects.c:496`) | `SetObjectsToDefaults` |
| high scores | `highScores` at house offset 528, 292 bytes | `TestHighScore` (`HighScores.c:374`) | `ZeroHighScores` / `ZeroAllButHighestScore` |
| `firstRoom` | house offset 862 | editor (`CreateNewRoom`, `DeleteRoom`) | — |
| embedded saved game | house offsets 820-860 | `SaveGame` (never called) | — |

`SetObjectsToDefaults()` (`GliderPRO/Sources/Play.c:603-708`) is the *only* thing that
resets object state, and it is called from exactly one place:
`NewGame(kNewGameMode)`. Its shape:

```
1.  SetObjectsToDefaults():
2.    wasState = HGetState(thisHouse); HLock(thisHouse)
3.    for r in 0 .. nRooms-1:
4.      rooms[r].visited = false
5.      for i in 0..23:
6.        switch rooms[r].objects[i].what:
7.          (all blower types)      : data.a.state = data.a.initial
8.          (all bonus/prize types) : data.c.state = data.c.initial
9.          (all light types)       : data.f.state = data.f.initial
10.         kStereo                 : data.g.state = isPlayMusicGame   // <== NOT .initial
11.         (other appliance types)  : data.g.state = data.g.initial
12.         (all enemy types)       : data.h.state = data.h.initial
13.         kDeluxeTrans            : initState = (data.d.wide & 0xF0) >> 4
14.                                   data.d.wide &= 0xF0
15.                                   data.d.wide += initState        // high nibble -> low
16.   HSetState(thisHouse, wasState)
```
(`GliderPRO/Sources/Play.c:603-708`; the `kStereo` special case is `:675-677`, the
`kDeluxeTrans` nibble copy `:657-661`.)

Two details a port must not smooth over: **a stereo's initial state is the player's
music preference, not the object's `initial` bit**, and `kDeluxeTrans` keeps its initial
state in the *high* nibble of `data.d.wide` and its live state in the low nibble (the
same packing `SetObjectState` maintains at `GliderPRO/Sources/Objects.c:496-531`).

**Note the house handle is mutated but `fileDirty`/`gameDirty` is not set here**, so a
new game does not by itself make the house need saving; only `TestHighScore` sets
`gameDirty`. But the in-memory house *has* diverged from disk. If the player then
edits and saves the house, the reset states are what get written.

### 17.2 In globals, rebuilt on every room change

Cleared/rebuilt by `MoveRoomToRoom` → `ReadyLevel` → `DetermineRoomOpenings` +
`DrawLocale` (see §8):

`localNumbers[9]`, `isStructure[9]`, `thisTiles[8]`, `thisBackground`,
`numNeighbors`-dependent draw set, `leftOpen`/`rightOpen`/`topOpen`/`bottomOpen`,
`leftThresh`/`rightThresh`, `numLights`, `shadowVisible`, `masterObjects[]`,
`numMasterObjects`, `hotSpots[]`, `numHotSpots`, `dinahs[]`, `numDynamics`,
`savedMaps[24]`, `numSavedMaps`, `sparkles`, `flyingPoints`, `flames`, `tikiFlames`,
`bbqCoals`, `pendulums`, `theStars`, `shreds`, `grease`, `triggers[16]`, bands in
flight, `numTempManholes`, `tvInRoom`, `tvWithMovieNumber`, `takingTheStairs`.

### 17.3 In globals, persisting across rooms for the whole game

`theScore`, `mortals`, `numStarsRemaining`, `batteryTotal`, `bandsTotal`, `foilTotal`,
`showFoil`, `gameFrame`, `evenFrame`, `numBands`, `otherPlayerEscaped`, `firstPlayer`,
`onePlayerLeft`, `playerDead`, `playerSuicide`, `saidFollow`, `activeRectEscaped`,
`thisRoomNumber`, `previousRoom`, `gameOver`, `countDown`, `playing`, `twoPlayerGame`,
`demoGoing`, `thePhone`, `theChimes`, `phoneBitSet`, `tvOn`, `resumedSavedGame`, and
the two `gliderType` structs.

### 17.4 The one-line summary

> **Progression state = the house's `visited` bits + the objects' `state` bytes +
> `{theScore, mortals, numStarsRemaining, batteryTotal, bandsTotal, foilTotal,
> showFoil, thisRoomNumber, glider.enteredRect}`.** Everything else is derived and
> rebuilt on every room entry.

A Go port that snapshots exactly that set can implement working save/resume, which the
original never shipped.

---

## 18. Timing and the frame budget

### 18.1 The clock

- A classic Mac tick is 1/60.15 s (`TickCount()` increments ~60.15 times/second).
- `kTicksPerFrame` = 2 (`GliderPRO/Headers/GliderDefines.h:533`).
- Pacing is a busy-wait at the tail of `RenderFrame`:
  ```c
  while (TickCount() < nextFrame) { }            // Render.c:662-664
  nextFrame = TickCount() + kTicksPerFrame;      // Render.c:665
  CopyRectsQD();
  numWork2Main = 0;
  numBack2Work = 0;
  ```
  `nextFrame` is seeded in `InitGarbageRects` (`GliderPRO/Sources/Render.c:690`), which
  `ReadyLevel` calls on every room entry.
- **Target rate ≈ 30.07 fps.** The deadline is recomputed *after* the wait from the
  *current* tick, not from the previous deadline, so a slow frame permanently shifts the
  schedule: there is **no catch-up and no frame skipping**. On a machine too slow to
  render in 2 ticks the whole game simply runs slower, and all the frame-counted
  animations slow down with it. That is why the original feels consistent rather than
  jerky on slow hardware, and it is the behaviour a Go port should emulate if it wants
  the original's feel: **advance the simulation exactly one step per rendered frame and
  cap at 30 fps**, rather than using a wall-clock delta.

### 18.2 Everything measured in frames

| Constant / variable | Value | Seconds at 30 fps | Site |
|---|---:|---:|---|
| `kTicksPerFrame` | 2 | 1/30 | `GliderDefines.h:533` |
| `kNumCountDownFrames` (`countDown`) | 16 | 0.53 | `GameOver.c:17` |
| `kFramesToBurn` | 60 | 2.0 | `Modes.c:408` |
| `kShredderCountdown` | -68 | 2.27 | `Player.c:17` |
| `kLastFadeSequence` | 16 | 0.53 | `GliderDefines.h:564` |
| `TagGliderIdle` `hVel` delay | 30 | 1.0 | `Modes.c:638` |
| `kLengthOfZap` | 30 | 1.0 | `GliderDefines.h:546` |
| `kStartSparkle` | 4 | 0.13 | `GliderDefines.h:545` |
| trigger delay | `data.e.delay * 3` | `delay` × 0.1 | `Triggers.c:47` |
| `ToggleMacPlus` timer | 40 on / 10 off | 1.33 / 0.33 | `Trip.c:31-34` |
| `ToggleTV`/`Coffee`/`VCR`/`Stereos`/`Microwave` timer | 4 | 0.13 | `Trip.c:56, 64, 80, 91, 100` |
| `TriggerDrip` timer cap | 7 | 0.23 | `Trip.c:191` |
| `kRingDelay` | 90 | 3.0 | `Play.c:20` |
| `kRingBaseDelay` | 5000 | 166 (2 min 46 s) | `Play.c:22` |
| `kRingSpread` | 25000 | up to 832 (13 min 52 s) | `Play.c:21` |
| `kChimeDelay` | 180 | 6.0 | `Play.c:23` |
| game-over star fall | 8 px/frame | 240 px/s | `GameOver.c:138` |
| game-over angel drift | 2 px/frame | 60 px/s | `GameOver.c:200` |
| died-game-over pace | `TickCount() + 2` | 30 fps | `GameOver.c:456`, `:478` |
| `pass` after angel exits | 80 frames | 2.7 | `GameOver.c:222` |
| `WaitForInputEvent` after loss | 10 ticks | 0.17 | `GameOver.c:502` |
| `WaitForInputEvent` after demo loss | 1 tick | 0.017 | `GameOver.c:497` |
| `WaitForInputEvent` after the win animation | 5 ticks | 0.083 | `GameOver.c:226` |
| `HandlePlayEvent` sleep | `long sleep = 2;` ticks | 0.033 | `Play.c:391` |

Every random draw in the tables above goes through one function:

```c
short RandomInt (short range)                          // Utilities.c:72-82
{
    register long rawResult;
    rawResult = Random();                              // Toolbox: -32768..32767
    if (rawResult < 0L) rawResult *= -1L;
    rawResult = (rawResult * (long)range) / 32768L;
    return ((short)rawResult);
}
```

So `RandomInt(n)` is *almost* `[0, n)`: because `Random()` can return `-32768`, whose
absolute value is `32768`, the result is `n` itself with probability 1/65536. Every
`RandomInt(k)`-indexes-a-`k`-element-array site in the game is therefore a one-in-65536
out-of-bounds read (e.g. `pages[i].frame = RandomInt(6)` into the 6-entry `starSrc`,
`RandomInt(2) == 0` picking a third case that no `if` covers). A Go port should use
`rand.Intn(n)` — strictly `[0, n)` — and note that this *fixes* rather than reproduces
the original. Ranges quoted elsewhere in this document as `a..b` mean "0..range-1 plus
the base", i.e. they ignore that 1/65536 tail.

### 18.3 The per-frame call order in `PlayGame`

Exactly this, every frame (`GliderPRO/Sources/Play.c:430-599`; the loop body is
`:432-554`, and it is written out twice — `:447-472` for two players, `:473-497` for
one):

```
 1.  gameFrame++
 2.  evenFrame = !evenFrame
 3.  if doBackground:  do { HandlePlayEvent(); } while (switchedOut)
 4.  HandleTelephone()
 5.  HandleDynamics()
 6.  if !gameOver:
 7.     GetInput(&theGlider)                     [demo: GetDemoInput; 2P: also theGlider2]
 8.     HandleInteraction()
 9.  HandleTriggers()
10.  HandleBands()
11.  if !gameOver:
12.     HandleGlider(&theGlider)                 [2P: also HandleGlider(&theGlider2)]
13.  if playing:
14.     [COMPILEQT] if hasQT && hasMovie && tvInRoom && tvOn: MoviesTask(theMovie, 0)
15.     RenderFrame()                            // <== the 2-tick busy-wait lives here
16.     HandleDynamicScoreboard()
17.  if gameOver:
18.     countDown--
19.     if countDown <= 0:
20.        save GWorld; HideGlider(&theGlider); RefreshScoreboard(kNormalTitleMode)
21.        [BUILD_ARCADE_VERSION] paint the scoreboard black, redraw kScoreboardPictID
22.        if mortals < 0: DoDiedGameOver()  else: DoGameOver()
23.        restore GWorld
```

Load-bearing ordering facts:

- **`HandleDynamics()` runs before input and before `gameOver` is checked** (line 5), so
  the room keeps animating during the 16-frame game-over countdown while the glider is
  frozen.
- **`HandleTriggers()` and `HandleBands()` run even when `gameOver`** (lines 9-10).
- **`HandleGlider` runs after `HandleInteraction`** (line 12 after line 8), so
  collisions are resolved against the *previous* frame's position, and the glider's
  velocity is applied afterwards. A Go port that reorders these will change the physics.
- **The room transition happens inside `HandleInteraction` → `CheckGliderInRoom` →
  `MoveRoomToRoom`** (line 8), i.e. *mid-frame*. By the time `HandleGlider` runs on
  line 12, `thisRoom`, `localNumbers`, `masterObjects`, `hotSpots` and `dinahs` have all
  been replaced, and `RenderFrame` on line 15 draws the new room. A transition therefore
  costs no extra frame — but everything after line 8 in that frame operates on the new
  room's data with the old frame's glider velocity.
- `HandleDynamicScoreboard` (line 16) updates one badge per frame, chosen by
  `whosTurn = gameFrame & 0x00000007` (`GliderPRO/Sources/Scoreboard.c:95`), so the four
  badges (`kFoilBadge` 0, `kBandsBadge` 1, `kBatteryBadge` 2, `kHeliumBadge` 3) each
  refresh at most every 8 frames ≈ 3.7 Hz.
- `evenFrame` (line 2) is the parity flag used by half-rate animations.

### 18.4 The telephone

The only wall-clock-independent random event in progression.

```c
#define kRingDelay      90
#define kRingSpread     25000
#define kRingBaseDelay  5000
#define kChimeDelay     180
typedef struct { short nextRing; short rings; short delay; } phoneType;
phoneType thePhone, theChimes;
```
(`GliderPRO/Sources/Play.c:20-31`, `:47`)

The exact bodies of `InitTelephone` / `HandleTelephone` / `StrikeChime` are transcribed
in **§6.7**; do not duplicate them here. The timing-relevant facts:

- `InitTelephone` (`Play.c:733-740`) seeds `thePhone.nextRing = RandomInt(kRingSpread)
  + kRingBaseDelay` (5000..29999 frames), `thePhone.rings = RandomInt(3) + 3` (**3..5**
  rings), `thePhone.delay = kRingDelay` (90), and `theChimes.nextRing =
  RandomInt(kChimeDelay) + 1` (1..180). `theChimes.rings` and `theChimes.delay` are
  never touched by anything.
- `HandleTelephone` (`Play.c:744-789`) runs the phone only when **`phoneBitSet` is
  false**; the chimes half runs unconditionally whenever `numChimes > 0`.
- Both countdowns test for `== 0` and decrement in the `else`, so a burst is
  `nextRing` frames of silence, then one ring every `kRingDelay` = 90 frames
  (≈3 s), `rings` times, then a fresh random `nextRing`/`rings`.
- `StrikeChime` (`Play.c:793-796`) sets `theChimes.nextRing = 0` only.

`phoneBitSet` comes from house flags bit 1 (`phoneBit`), decoded in `ReadHouse`
(`GliderPRO/Sources/HouseIO.c:417`). Empirically set in 8 of the 22 shipped houses:
Art Museum, CD Demo House, California or Bust!, Davis Station, Land of Illusion,
Nemo's Market, Rainbow's End, SpacePods — those are the houses whose phone is
*silenced*.

So in a house with the phone enabled the first ring lands between frame 5000 and frame
30000 — **≈2 min 46 s to ≈16 min 38 s** into the game — and then 3 to 5 rings ≈3 s
apart, then a fresh random wait. It has no gameplay effect; it is atmosphere.
`StrikeChime` is called from the chimes object (`kChimes`) when the glider bumps it.

---

## Open questions

1. **`sizeof(game2Type)` under the shipping compiler.** The struct comment says 114
   (`GliderPRO/Headers/GliderStructs.h:163`) but the fields sum to 110 under
   `align=mac68k`. Since the external `'gliG'` writer and reader are both commented out
   and no `'gliG'` file exists in the release, the on-disk offset of `savedData[0]`
   cannot be determined from the sources. Irrelevant unless someone finds a real
   Glider PRO saved game to read.
2. **Which build was actually shipped as 1.0.4?** `BUILD_ARCADE_VERSION` is `1` in the
   headers, which paints the scoreboard black and redraws `kScoreboardPictID` at
   game over (`GliderPRO/Sources/Play.c:512-541`) and unconditionally at the end of
   `PlayGame` (`:556-592`). Whether the retail 1.0.4 binary had this on cannot be
   confirmed without the binary. The non-arcade `#else` branches contain only
   `// ShowMenuBarOld(); // TEMP`, so the arcade path is the only one that does
   anything.
3. ~~**`kPlayerEscapingUpStairs` (-8) / `kPlayerEscapingDownStairs` (-9) are never
   assigned.**~~ **Resolved — they are.** `HandleHotSpotCollision` sets them the moment a
   glider steps onto a staircase hot spot (`GliderPRO/Sources/Interactions.c:1266`,
   `:1301`) and `MoveGliderUpStairs`/`MoveGliderDownStairs` promote them to the `-ed`
   codes when the climb completes (`GliderPRO/Sources/Player.c:363`, `:491`). What is
   *not* determinable from the sources is whether the asymmetry in the second player's
   test — it matches only the `-ed` codes (`Interactions.c:1271`, `:1306`), so a second
   player who reaches the same staircase while the first is still mid-climb is silently
   ignored — was intended or is a bug. See §9.1.
4. **`houseType.unusedShort` (offset 2) is uninitialised garbage** in 10 of 22 shipped
   houses (observed: -30082, 13107, 259, 222, 26228, 60, 147, 259, 196, 2074). It is
   never written by `InitializeEmptyHouse` (`GliderPRO/Sources/House.c:108-168`). Whether
   an ancestor of the format used it (a Glider 4.0 field?) is unknown. **A loader must
   not validate it.**
5. **`roomType.unusedByte` (offset +32) and `savedRoom.unusedShort`/`unusedByte`**: same
   situation, purpose unknown, values unverified per-room across the corpus.
6. **Why does `ReadyGliderFromTransit` have an empty `case kLinkedToFloorDuct: break;`?**
   (`GliderPRO/Sources/Transit.c:138`.) Arriving via a floor duct therefore leaves the
   glider wherever `MoveDuctToDuct` left it, and does not set `enteredRect`. Whether
   this is a deliberate "the duct animation already positioned it" or an omission is
   not determinable; the `kLinkedToCeilingDuct` case immediately above it *does* do
   work.
7. **Is the 24-slot `savedMaps` exhaustion ever hit in the shipped houses?** Determining
   this requires simulating each room's 3×3 neighbourhood and counting visible
   bonus objects, which needs the full `GetObjectRect` / `SectRect` logic (out of scope
   for this document). The dense houses (`Land of Illusion`, `Slumberland`,
   `Teddy World`) are the candidates.
8. **`ShiftWholeHouse`** (`GliderPRO/Sources/House.c:824`) is dead code: `#pragma unused
   (howFar)` and an empty inner loop. What it was meant to do (renumber all suites?) is
   unknown.
9. **`Fun House` has 0 stars and is therefore unwinnable** (verified: §2.6 Test 7).
   Whether the shipped house is simply broken, or whether a `'bnds'`/PICT-supplied
   variant adds stars, could not be determined — `CountStarsInHouse` only walks
   `rooms[].objects[]`, so no resource can add one.
10. **`Castle o' the Air`, `Land of Illusion`, `Rainbow's End`, `Slumberland`** together
    contain 165 links encoded as `where == -100`, `who == 255`. `-100` decodes under v2
    to `suite = -1` (`kRoomIsEmpty`), `floor = -8`, and `MergeFloorSuite(0, -1)` is
    exactly `-100`. This looks like the editor's "unlinked" sentinel from a build that
    used `(floor 0, suite kRoomIsEmpty)` instead of `where = -1`. Whether 1.0.4's editor
    ever writes `-100` (as opposed to only reading it) was not traced.

---

## Porting notes

Ordered roughly by how likely each is to bite.

### Data and layout

1. **`Point` is `{short v; short h;}` — vertical first.** Every `Point` in the house
   file (`houseType.initial`, `gameType.where`, every object's `topLeft`) is
   `v` then `h`. Getting this backwards yields plausible-looking but wrong coordinates.
2. **All on-disk integers are big-endian.** Use `encoding/binary.BigEndian`, never
   `unsafe` struct casts.
3. **Read the house as `866 + 348 * nRooms` bytes, but accept a longer file.** Verified:
   21 of 22 shipped houses are exactly that size; `Sampler` is 2 bytes longer because
   `WriteHouse` writes `GetHandleSize()` (`GliderPRO/Sources/HouseIO.c:473`) and
   `ReadHouse` allocates `NewHandle(GetEOF)` (`:341-356`). Reject only `size < 866 + 348*nRooms`.
4. **Pascal strings.** `Str27` is 28 bytes (`length` byte + 27), `Str31` 32, `Str15` 16,
   `Str255` 256. Bytes past `length` are garbage, not NUL. Decode as **MacRoman**, not
   UTF-8 or Latin-1 — house banners contain `…` (0xC9) and curly quotes
   (observed: ImagineHouse PRO II's high-score banner `'Thankyuh… Thankyuh verra much…'`).
5. **`state` and `initial` are at *different* offsets in different union arms.**
   `data.a` (blower): `initial` at +8, `state` at +9. `data.c` (bonus): `state` at +10,
   `initial` at +11 — **swapped relative to the others**. `data.f`/`data.g`/`data.h`:
   `initial` at +10, `state` at +11. Get this wrong and every bonus in the game is
   pre-taken or every light is inverted.
6. **`houseType.unusedShort` at offset 2 is garbage.** Do not validate it. Do not
   round-trip it as zero if you want byte-identical saves — preserve it.
7. **`roomType.openings` (offset +56) is always 0 on disk.** Verified across all 22
   houses, all 4 070 rooms: zero mismatches. It is a runtime cache recomputed by
   `DetermineRoomOpenings`. Do not read it; do not trust it.
8. **`roomType.numObjects` (offset +58) is reliable** — verified to equal the count of
   objects with `what != kObjectIsEmpty` (`kObjectIsEmpty` is **-1**, not 0,
   `GliderPRO/Headers/GliderDefines.h:526`) in all 4 070 rooms of all 22 shipped
   houses: zero mismatches. But the
   original mostly ignores it and loops to 24 anyway, so a port should also tolerate a
   wrong value by looping over all 24 slots and skipping empties.
9. **The 40-byte `savedGame` block and `hasGame` are useless.** Both shipped saves are
   `version 0x0100` (vs `kSavedGameVersion` 0x0200) with mismatched timestamps. Ignore
   them.

### Room addressing and links

10. **`ExtractFloorSuite` needs C truncating division, not floor division.**
    ```
    v2:  suite = combo / 100;        floor = (combo % 100) - kNumUndergroundFloors
    v1:  floor = (combo / 100) - kNumUndergroundFloors;  suite = combo % 100
    ```
    (`GliderPRO/Sources/Link.c:41-53`, `kNumUndergroundFloors` = 8.) Go's `/` and `%`
    truncate toward zero exactly like C's, so a direct transcription is correct —
    but do **not** verify it against a Python prototype, whose `//`/`%` floor and give
    different answers for negative `combo`. Verified: `-100 / 100 == -1` and
    `-100 % 100 == 0` in both C and Go, so `where == -100` decodes to
    `suite = -1` (`kRoomIsEmpty`), `floor = -8`.
11. **Rooms are addressed by `(suite, floor)` and resolved by linear search**
    (`GetRoomNumber`, `GliderPRO/Sources/Room.c:737-759`). `floor` on disk is *not*
    biased; the +8 bias exists only inside the link encoding. North is
    `floor + 1`, south is `floor - 1` (`GetNeighborRoomNumber`,
    `GliderPRO/Sources/Room.c:562-637`). Observed corpus ranges: suite 0..127,
    floor -7..39 (limits `kMaxNumRoomsH` 128, `kMaxNumRoomsV` 64).
12. **`ForceThisRoom(-1)` is a no-op**, so walking through an open wall with no room
    beyond re-enters the *same* room and the glider is wrapped to the opposite side.
    Reproduce this; it is load-bearing level-design behaviour in several houses.
13. **`ForceThisRoom` does not write the outgoing room back.** `thisRoom` is a 348-byte
    *copy*; only `CopyThisRoomToRoom()` stores it, and that is called only from the
    editor. Runtime object-state changes therefore go through `SetObjectState`, which
    writes the house **directly**, bypassing `thisRoom`. In Go, the cleanest model is to
    make the current room a pointer/index into the house slice and drop the copy — but
    then you must audit every place the original relies on `thisRoom` being stale.
14. **No code path checks `roomLink != -1` before using it as a room index.** Verified at
    `GliderPRO/Sources/Interactions.c:994` (reads `roomLink`) and `:998` (passes it to
    `SetObjectState`), `GliderPRO/Sources/Objects.c:373` (`SetObjectState`'s own
    `(*thisHouse)->rooms[room]`), and in `Triggers.c`: `ArmTrigger` stores the raw
    `roomLink` at `:46`, and `FireTrigger` uses it as a room index at `:179-180` (the
    `switch` on `rooms[triggers[index].room].objects[...].what`) and again at `:184`.
    The only guard anywhere is `if (theObject.data.d.who != 255)` inside
    `CreateActiveRects`, and it exists for exactly six of the sixteen transport types —
    `kMailboxLf` (`GliderPRO/Sources/ObjectRects.c:752`), `kMailboxRt` (`:764`),
    `kFloorTrans` (`:776`), `kCeilingTrans` (`:788`), `kInvisTrans` (`:872`) and
    `kDeluxeTrans` (`:885`); the `case` label is the line above each guard. It tests `who`,
    not `where`, so it screens out only stale links that also happen to have
    `who == 255`. Of the 24 stale-v1 links in the shipped corpus (§5.5 Class 2), 22 have
    a real `who` (re-derived: `who` ∈ {0, 1, 6, 8, 11}) and sail straight through.
    Concretely, `ImagineHouse PRO II` room 48 object 9 (`kMailboxRt`, `where = 971`,
    `who = 1`) and the 15 `kTrigger`s in `Leviathan` room 9 (objects 6-9, 12-21, 23;
    `where = 960`, `who = 11`), plus `Leviathan` room 29 object 7 (`kLightSwitch`,
    `where = 953`), room 376 object 7 (`where = 1479`) and room 425 objects 12/14/17/18
    (`where = 879`), all resolve to `kRoomIsEmpty` (-1) and are then used as a room
    index. **In Go these panic.** Treat `roomLink == -1` (and any out-of-range room) as
    "do nothing" at every use site, and additionally suppress hot-spot creation for such
    objects.
15. **`FireTrigger`'s remote-grease arm reads `masterObjects[-1]`.** `triggeredIs` is
    read from `masterObjects[triggerIs].localLink` at `GliderPRO/Sources/Triggers.c:178`
    — that is -1 for any object whose link did not resolve locally — and is then used as
    an index at `:187-188` (`SpillGrease(masterObjects[triggeredIs].dynaNum,
    masterObjects[triggeredIs].hotNum)`). Guard it.
16. **`ArmTrigger` indexes `masterObjects[]` with a room-relative object number**
    (`GliderPRO/Sources/Triggers.c:50`). The value is never read; just omit the field.

### Progression semantics

17. **Room-visit scoring: `theScore += kRoomVisitScore` (100) only when the room was
    not already `visited`**, and the flag is written to the *house*, not to `thisRoom`
    alone (`HandleRoomVisitation`, `GliderPRO/Sources/Transit.c:430-447` writes
    `thisHousePtr->rooms[localNumbers[kCentralRoom]].visited = true;` **and**
    `thisRoom->visited = true;`).
18. **The win condition is evaluated only inside the star-pickup handler**
    (`GliderPRO/Sources/Interactions.c:943-944`). A house with 0 stars can never be won
    (`Fun House`). A house whose stars are pre-taken (`data.c.initial == 0`) is also
    unwinnable, because `CountStarsInHouse` counts `what == kStar` regardless of
    `initial`, skipping only rooms whose `suite == kRoomIsEmpty`
    (`GliderPRO/Sources/Banner.c:89-111`) while a pre-taken star cannot be
    collected. Do not "fix" this silently.
19. **The respawn point is `glider.enteredRect`, which `InitGlider`/`NewGame` never
    initialise.** Set it to the initial `dest` in a Go port and document the divergence.
20. **`mortals` is a shared pool.** Single player: game over when `mortals < 0` (third
    death). Two player: the first player to exhaust it goes into permanent limbo
    (`onePlayerLeft = true`, `playerDead = which`, `mortals == -1`) and the game ends on
    the next death (`mortals < -1`). **Eight** call sites test the full conjunction
    `(twoPlayerGame) && (onePlayerLeft) && (thisGlider->which == playerDead)` —
    `GliderPRO/Sources/Modes.c:474`, `:499`, `:510`, `:523`, `:554`, `:633` (the last two
    of those spell the operands in the order `twoPlayerGame && which == playerDead &&
    onePlayerLeft`), `GliderPRO/Sources/Player.c:1445`, and
    `GliderPRO/Sources/Transit.c:69`. `playerDead` is read at **35** sites in total: the
    other 27 are `playerDead == kPlayer1` / `== kPlayer2` / `== theGlider.which` tests
    scattered through `Dynamics.c`, `Dynamics2.c`, `Interactions.c`, `RubberBands.c` and
    `Player.c` (it is written in exactly one place, `Player.c:1508`). Enumerate them all
    or the dead player will move.
21. **`FlagGameOver()` does not record win vs lose.** `PlayGame` decides by
    `mortals < 0` sixteen frames later (`GliderPRO/Sources/Play.c:546`). Keep that, or
    add an explicit outcome field and make sure `mortals < 0` can no longer be true at
    a win (it cannot today, but a refactor could break it).
22. **Only Player 1's Delete key force-kills**, because the test is the truthiness of
    `thisGlider->which` and `kPlayer1` is `TRUE`
    (`GliderPRO/Headers/GliderDefines.h:556-557`: `kPlayer1 TRUE`, `kPlayer2 FALSE`; the
    three-term test is `GliderPRO/Sources/Input.c:366-368`, with the `BitTst` on
    `kDeleteKeyMap` at `:367` and the bare `(thisGlider->which)` term at `:368`).
23. **`ForceKillGlider` kills the glider that is *not* in limbo** — the one that
    refused to follow (`GliderPRO/Sources/Transit.c:449-471`).
24. **`FollowTheLeader` maps `kPlayerEscaped*Stairs` to a plain `MoveRoomToRoom(kAbove/
    kBelow)`**, not to the stairs animation, because `takingTheStairs` is already false
    (the `switch (wasEscaped)` is `GliderPRO/Sources/Transit.c:499-556`; the up arm —
    `kPlayerEscapedUp` / `kPlayerEscapingUpStairs` / `kPlayerEscapedUpStairs` all falling
    into one `MoveRoomToRoom(..., kAbove)` — is `:501-508`, the down arm `:510-517`).
25. **Object `state` persists across room changes and deaths; `dinahs[]` state does
    not.** A flipped switch stays flipped; the appliance it controls resets. Preserve
    the asymmetry.
26. **`ZeroTriggers()` has exactly one caller, `DrawLocale`**
    (`GliderPRO/Sources/RoomGraphics.c:55`), and that is enough: every room change and
    every game start goes through `DrawLocale` (`NewGame` calls it at
    `GliderPRO/Sources/Play.c:163`), so the 16-slot `triggers[]` array is cleared on
    entry to every room and armed triggers cannot leak across rooms or across games. In
    Go, keep the clear attached to room entry rather than to game start — a trigger armed
    in room A must not fire after the glider has left.
27. **`CheckGliderInRoom`'s horizontal and vertical tests are independent `if`s, not
    `else if`s** (`GliderPRO/Sources/Interactions.c:689-752`), so a glider in a corner
    can trigger two transitions in one frame. Reproduce or explicitly diverge.
28. **`MoveRoomToRoom` reads `thisRoom->leftStart` / `rightStart` *after* the new room
    has been loaded** (`leftStart` is read at `GliderPRO/Sources/Transit.c:176` in the
    two-player arm and `:185` in the one-player arm; `rightStart` at `:208` and `:217`),
    i.e. the entry Y is the **destination** room's start value, not the source's.
    `leftStart`/`rightStart` are `Byte`s at room offsets +30 and +31; the entry rect is
    `QSetRect(&enterRect, 0, 0, 48, 20)` offset by
    `(0, kGliderStartsDown + leftStart - 2)` for a rightward move and by
    `(kRoomWide - 48, kGliderStartsDown + rightStart - 2)` for a leftward one — with
    `kGliderStartsDown` 32 and `kRoomWide` 512 (`GliderPRO/Headers/GliderDefines.h:569`,
    `:499`) that is `(0, 30 + leftStart)` and `(464, 30 + rightStart)`.

### Rendering and Mac Toolbox replacements

29. **`DrawLighting()` is an empty stub.** Darkness = per-object draw suppression via
    `isLit`, not a post-process (`GliderPRO/Sources/RoomGraphics.c:422-430`).
    `RedrawRoomLighting` redraws **only the central room** and only when lit-ness
    flipped (`:434-461`).
30. **The 24-slot `savedMaps` cap makes the 25th visible bonus invisible but still
    collectible** (`BackUpToSavedMap` returns -1 → callers skip drawing,
    `GliderPRO/Sources/DynamicMaps.c:75`, `GliderPRO/Sources/ObjectDrawAll.c:293-297`).
    Keep the cap for fidelity. In Go, `savedMaps[i].map` becomes an `*image.RGBA` (or a
    texture handle); `CreateOffScreenGWorld(..., kPreferredDepth)` becomes an
    allocation, `DisposeGWorld` becomes dropping the reference.
31. **`CopyBits`/`CopyMask` → blit with and without a 1-bit mask.** `CopyMask`'s mask is
    a separate depth-1 GWorld built from a companion PICT (e.g. `kPagesPictID` 1990 with
    `kPagesMaskID` 1989). In Go, pre-multiply the mask into an alpha channel at load
    time.
32. **8-bit indexed colour.** `ColorRect(&r, 244)` and `kRedOrangeColor8` = 23 are
    palette indices into the game's `'clut'`. `kGrayBackgroundColor` = 251 and
    `kGrayBackgroundColor4` = 10 (`GliderPRO/Sources/Scoreboard.c:15-16`). A Go port needs the palette from the resource fork to
    reproduce these exactly.
33. **`thisMac.isDepth == 4` even-pixel fixups** (e.g. `if ((pages[i].dest.left % 2) == 1)
    QOffsetRect(&pages[i].dest, 1, 0);`, `GliderPRO/Sources/GameOver.c:292-293`) are 4-bit
    packing artefacts. Dropping them shifts sprites by 1 px.
34. **`RgnHandle` clipping** (`roomRgn = NewRgn(); RectRgn(roomRgn, &justRoomsRect);`,
    `GliderPRO/Sources/GameOver.c:306-307`) becomes a rectangular clip in Go. Only the
    rectangular case is used in the progression code.
35. **`InitDiedGameOver` uses the screen *width* for both the H and V offsets**
    (`GliderPRO/Sources/GameOver.c:290-291` — both `QOffsetRect` arguments are
    `(thisMac.screen.right - thisMac.screen.left) / -2`). On non-4:3 screens the
    "GAME OVER" pages
    start far above the visible area. Decide and document.
36. **`DrawPages` calls `CopyRectsQD()` inside the per-page loop**
    (the loop is `GliderPRO/Sources/GameOver.c:405-434`, the flush at `:428`, and
    `numWork2Main`/`numBack2Work` are reset to 0 immediately after it at `:430-431`),
    flushing 8 times per frame.
37. **The three-buffer model.** `backSrcMap` (clean background) → `workSrcMap` (composed
    frame) → `mainWindow` (screen), with dirty-rect lists `numBack2Work` /
    `numWork2Main` flushed by `CopyRectsQD()`. `RestoreFromSavedMap` writes to **both**
    `backSrcMap` and `workSrcMap` (two `CopyBits` calls, `GliderPRO/Sources/DynamicMaps.c:144-146`
    and `:147-149`) — that is
    how a taken bonus stays gone. A Go port can render everything every frame and skip
    the dirty-rect machinery, but must then re-derive which objects are "erased" from
    `state` rather than relying on the background buffer having been overwritten.
38. **Frame pacing: advance exactly one simulation step per frame, capped at 30 fps
    (`kTicksPerFrame` = 2 ticks of 1/60.15 s), with the deadline recomputed after the
    wait.** No catch-up, no frame skipping, no delta-time integration
    (`GliderPRO/Sources/Render.c:662-670`, `:690`). Using wall-clock deltas will change
    every animation and every physics constant in the game.
39. **`PlayPrioritySound(id, priority)`** is a fixed-channel priority mixer: a higher
    priority preempts a lower one. Progression uses `kFadeOutSound` (death fade),
    `kFollowSound` (the "follow me" voice, capped at 3 per game by
    `if ((sayIt) && (saidFollow < 3))`, `GliderPRO/Sources/Modes.c:462-465`, with
    `saidFollow = 0` in `NewGame`, `GliderPRO/Sources/Play.c:115`), `kDontExitSound`
    (refused escape), `kEnergizeSound` (star), `kMysticSound` (win animation),
    `kPaper1..kPaper4Sound` (lose animation), `kPhoneRingSound`, the grandfather clock's
    `kChime1Sound` (56) / `kChime2Sound` (57) with priorities `kChime1Priority` 203 /
    `kChime2Priority` 204 — there is **no** `kChimeSound` —
    `kShredSound`, `kCaughtFireSound`, `kChordSound`, `kCoffeeSound`.
40. **`WaitNextEvent`/`GetKeys`/`KeyMap`/`BitTst`** → your framework's polled keyboard
    state. `GetKeys` reads the *current physical* key state, not buffered events, so
    key-repeat and focus changes behave differently from an event queue. `switchedOut`
    (the `HandlePlayEvent` suspend/resume loop, `GliderPRO/Sources/Play.c:387-426`;
    `0x01000000` at `:408` is the `osEvt` suspend/resume *class* mask and the resume bit
    proper is `theEvent.message & 0x00000001` at `:410`, with the suspend branch at
    `:417-423`) becomes window-focus handling; the
    original spins in `do { HandlePlayEvent(); } while (switchedOut)` and thereby
    freezes the game while backgrounded.
41. **`GetIndString` / `DrawString` / `TextFont(applFont)` / `TextWidth`** →
    embedded strings and a font metric. The win-screen text is centred with
    `offset = (screenWidth - TextWidth(subStr)) / 2` and drawn twice, black at
    `(offset+1, y+1)` then white at `(offset, y)`, 20 px line spacing from `y = textDown + 32`
    (the `do`/`while (subStr[0] > 0)` loop is `GliderPRO/Sources/GameOver.c:97-115`: the
    centring at `:101-102`, the black pass at `:107-108` using `textDown + 33`, the white
    pass at `:110-111` using `textDown + 32`). Line breaks are `\r`.
42. **QuickTime.** The house movie is a sibling file `<housename>.mov`; the TV object
    plays it via `SetMovieGWorld` into `workSrcMap`, with `MoviesTask(theMovie, 0)`
    pumped once per frame from `PlayGame`. Room changes call `StopMovie` and clear
    `tvInRoom`/`tvWithMovieNumber` (`GliderPRO/Sources/RoomGraphics.c` `ReadyLevel`,
    `:402-418`, with the `StopMovie` block at `:407-412`). Everything QuickTime is inside
    `#ifdef COMPILEQT`; a Go port can stub
    it and lose only the animated TV.
43. **Memory Manager handle locking** (`HLock`/`HUnlock`/`HGetState`/`HSetState` around
    every `(*thisHouse)->` dereference) has no Go analogue. Delete it. But note that
    `HLock` regions mark exactly the spans where the original assumed the house pointer
    was stable — useful when deciding where a Go port can hold a `*Room` pointer.
