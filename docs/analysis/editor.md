# Glider PRO 1.0.4 — The Built-in House Editor

## Scope

This document specifies the **house editor** that ships inside Glider PRO 1.0.4 — the
mode the application enters via *Options → Room Editor* (Cmd-E), in which the user
authors "houses" (level files) in place: creating and deleting rooms on a map, placing
and dragging objects inside a room, resizing them by handle, editing their per-class
properties in modal dialogs, linking switches/triggers/transports to remote objects,
and saving the result back to the `'gliH'` data fork.

Covered here:

* entering and leaving edit mode, and everything that gates it (`houseUnlocked`,
  `noRoomAtAll`, `theMode`);
* the floating **Tools** palette — every tool, every class, the PICT tile layout,
  the icon-index remapping quirks;
* **selection**: hit testing, the three glider pseudo-selections, the XOR marquee,
  handles, and the coordinate readout;
* **adding** objects: the palette→`what` arithmetic, the full per-class default table,
  every per-room object cap and its alert, and the automatic door/window/stair pairing;
* **object info dialogs**: all 13 of them, with every DITL item number and every
  field written;
* **Room Info** (including the tile editor, background popup and "Original Artwork"
  bounds editor) and **House Info**;
* the **Map** window: layout, thumbnails, scrolling, room creation/deletion;
* **linking**: the Link windoid, the `where`/`who` encoding, retro-links;
* **copy/paste**, **undo**, **arrow-key nudging** and **room relocation** — all four of
  which are *absent or disabled* in 1.0.4, and exactly how;
* **validation**: `KeepObjectLegal` and the twelve-stage `CheckHouseForProblems`;
* **saving**: `WriteHouse`, the lock bit, version conversion;
* a **bug and quirk catalogue** (§21) of every out-of-range index, silent no-op, wrong-union
  write, validation-ordering bug, UI bug, and resource-level inconsistency in the editor path,
  each cross-checked against the 22 shipped houses to say whether real data triggers it.

Not covered: gameplay simulation, rendering of the room during play, sound, scoring.
Those live in sibling documents (`docs/analysis/rendering.md`, `object-dynamics.md`,
`house-format.md`, etc.). Where this document restates a house-format fact it does so
because the editor *writes* it.

Everything below is derived from the 1994 GPLv2 source drop in
`GliderPRO` plus the 22 shipped `.binhex` houses in
`GliderPRO/Houses/`. Binary claims were verified by parsing the actual bytes with
python3; observed values are reproduced inline.

**Citation convention.** `GliderPRO/Sources/ObjectEdit.c:143` means line 143 of that
file *after* converting the classic-Mac CR-only line endings to LF
(`tr '\r' '\n'`). The original files in the repo are CR-only; any editor that shows
them as one long line will not agree with these numbers.

## Sources read

Primary (assigned):

| File | Lines | Role |
|---|---|---|
| `GliderPRO/Sources/ObjectEdit.c` | 2810 | selection, drag, handles, delete, duplicate, rect derivation, edit-mode drawing |
| `GliderPRO/Sources/ObjectAdd.c` | 1084 | `AddNewObject` and all per-class defaults + caps |
| `GliderPRO/Sources/Tools.c` | 544 | the Tools palette windoid |
| `GliderPRO/Sources/Marquee.c` | 511 | animated XOR selection marquee + all drag primitives |
| `GliderPRO/Sources/Scrap.c` | 517 | **entirely commented out** — clipboard |
| `GliderPRO/Sources/Map.c` | 796 | the Map windoid |
| `GliderPRO/Sources/DynamicMaps.c` | 798 | animated-object backing stores (gameplay; editor draws none of them) |
| `GliderPRO/Sources/RoomInfo.c` | 907 | Room Info dialog, tile editor, Original Artwork |
| `GliderPRO/Sources/HouseInfo.c` | 343 | House Info dialog |
| `GliderPRO/Sources/ObjectInfo.c` | 2567 | 13 per-class object info dialogs |
| `GliderPRO/Sources/Link.c` | 396 | Link windoid + floor/suite encoding |
| `GliderPRO/Sources/Validate.c` | 398 | copy protection — **stubbed to `return true`** |
| `GliderPRO/Headers/ObjectEdit.h` | 15 | |
| `GliderPRO/Headers/Tools.h` | 12 | |
| `GliderPRO/Headers/Map.h` | 12 | |
| `GliderPRO/Headers/Marquee.h` | 24 | the `marquee` struct |

Followed into (definitions live elsewhere):

`GliderPRO/Headers/GliderDefines.h` (625), `GliderPRO/Headers/GliderStructs.h` (347),
`GliderPRO/Headers/Externs.h` (393), `GliderPRO/Headers/GliderProtos.h` (530),
`GliderPRO/Sources/HouseLegal.c` (1219),
`GliderPRO/Sources/Room.c` (1206), `GliderPRO/Sources/House.c` (860),
`GliderPRO/Sources/HouseIO.c` (708), `GliderPRO/Sources/Objects.c` (1001),
`GliderPRO/Sources/ObjectRects.c` (1188), `GliderPRO/Sources/ObjectDrawAll.c` (966),
`GliderPRO/Sources/StructuresInit2.c` (476),
`GliderPRO/Sources/Menu.c` (813), `GliderPRO/Sources/MainWindow.c` (601),
`GliderPRO/Sources/Coordinates.c` (196), `GliderPRO/Sources/RectUtils.c` (318),
`GliderPRO/Sources/DialogUtils.c` (806), `GliderPRO/Sources/StringUtils.c` (329),
`GliderPRO/Sources/Events.c` (573), `GliderPRO/Sources/Modes.c` (640),
`GliderPRO/Sources/SelectHouse.c` (676), `GliderPRO/Sources/HighScores.c` (856),
`GliderPRO/Sources/RoomGraphics.c` (462), `GliderPRO/Sources/Banner.c` (237),
`GliderPRO/Sources/Utilities.c` (788), `GliderPRO/Sources/InterfaceInit.c` (220),
`GliderPRO/Sources/Main.c` (386), `GliderPRO/Sources/Settings.c` (1482).

Resources: `GliderPRO/Glider PRO.r` (15 MB derez dump, 199 843 LF lines) — DLOG/DITL
geometry, ALRT text, MENU contents, STR# tables, CNTL/WDEF/CDEF definitions, PICT
dimensions, PAT# marquee patterns.

Binary evidence: all 22 `GliderPRO/Houses/*.binhex` files, decoded with a hand-written
BinHex 4.0 decoder and parsed with a hand-written `houseType`/`roomType`/`objectType`
reader and a Mac resource-fork reader.

---

# 1. Application modes and how the editor is entered

## 1.1 The three modes

`theMode` is a global `short` (`GliderPRO/Headers/GliderVars.h`) taking exactly three
values, defined in `GliderPRO/Headers/GliderDefines.h`:

| Constant | Value | Meaning |
|---|---:|---|
| `kSplashMode` | 0 | title screen / idle attract |
| `kEditMode` | 1 | **the editor** |
| `kPlayMode` | 2 | a game is running |

The editor is reachable only from splash mode and returns only to splash mode. There
is no way to edit while playing.

## 1.2 Entering / leaving (`GliderPRO/Sources/Menu.c:374`)

`DoOptionsMenu(iEditor)` (menu *Options* item 1, "Room Editor", Cmd-E — `MENU 130`)
toggles:

```
 1. if theMode == kEditMode:                       // leaving the editor
 2.     if fileDirty:  SortHouseObjects()           // compact object arrays, fix links
 3.     if not QuerySaveChanges(): break            // user hit Cancel -> stay in editor
 4.     theMode = kSplashMode
 5.     CloseMapWindow(); CloseToolsWindow(); CloseCoordWindow(); CloseLinkWindow()
 6.     DeselectObject(); StopMarquee()
 7.     if isPlayMusicIdle: StartMusic()   (YellowAlert kYellowNoMusic on error)
 8.     CloseMainWindow(); OpenMainWindow()
 9.     incrementModeTime = TickCount() + kIdleSplashTicks
10. else if theMode == kSplashMode:                 // entering the editor
11.     theMode = kEditMode
12.     StopTheMusic()
13.     CloseMainWindow(); OpenMainWindow()
14.     OpenCloseEditWindows()
15. InitCursor(); UpdateMenus(true)
```

Note step 3: `QuerySaveChanges()` can abort the transition. Note also that the *main
window is destroyed and recreated* on every mode change, because its size differs
(§4.1).

`OpenCloseEditWindows()` (`GliderPRO/Sources/Menu.c:792`) re-opens the three floating
windoids according to sticky preference flags, but **only if the house is unlocked**:

```
1. if theMode != kEditMode: return
2. if houseUnlocked:
3.     if isMapOpen:   OpenMapWindow()
4.     if isToolsOpen: OpenToolsWindow()
5.     if isCoordOpen: OpenCoordWindow()
6. else:
7.     CloseMapWindow(); CloseToolsWindow(); CloseCoordWindow()
```

It is also called after *Game → Open House* (`GliderPRO/Sources/Menu.c:334`), after
*House → New House* (`GliderPRO/Sources/Menu.c:448`) and from the Apple Event
open-document handler (`GliderPRO/Sources/AppleEvents.c:110`).

## 1.3 The three gates

Almost every editor entry point is guarded by some subset of:

| Gate | Where set | Meaning |
|---|---|---|
| `theMode == kEditMode` | `GliderPRO/Sources/Menu.c:384`/`406` | editor active |
| `houseUnlocked` | `GliderPRO/Sources/HouseIO.c` (`ReadHouse`) | `((*thisHouse)->timeStamp & 0x00000001) == 0` |
| `noRoomAtAll` | `GliderPRO/Sources/HouseIO.c` (`ReadHouse`) | `RealRoomNumberCount() == 0` |

`houseUnlocked` is the *only* write-protection mechanism: bit 0 of the house's
`timeStamp` long. Setting it makes the house read-only in the editor forever (there is
no unlock UI — see §16.4). `IsFileReadOnly()` (`GliderPRO/Sources/HouseIO.c:659`) has
its body commented out and always returns `false`, so a house on a locked volume or a
CD is *not* detected; the editor will happily let you edit it and then fail on write.

## 1.4 Menu state (`GliderPRO/Sources/Menu.c:98`)

`UpdateMenusHouseOpen()`:

| Item | Enabled when |
|---|---|
| `iSave` (House 2) | `fileDirty && houseUnlocked` |
| `iHouse` (House 4, House Info) | `houseUnlocked` |
| `iRoom` (House 5, Room Info) | `!noRoomAtAll && houseUnlocked` |
| `iObject` (House 6, Object Info) | `objActive != kNoObjectSelected && houseUnlocked` |
| `iBringForward` (House 14) | as `iObject`, **and** `objActive` is not one of the three glider pseudo-selections |
| `iSendBack` (House 15) | ditto |

`iSaveAs` is commented out in both `UpdateMenusHouseOpen` and
`UpdateMenusHouseClosed` (`GliderPRO/Sources/Menu.c:107`, `:112`, `:150`) — there is no
Save As in 1.0.4.

The full House menu (`MENU 131`, enableFlags `0xFFFADF77`):

| # | Title | Key | Handler |
|--:|---|---|---|
| 1 | New House… | N | `CreateNewHouse()` + `InitializeEmptyHouse()` |
| 2 | Save House | S | `WriteHouse(true)` |
| 3 | *(divider)* | | |
| 4 | House Info… | | `DoHouseInfo()` |
| 5 | Room Info… | R | `DoRoomInfo()` |
| 6 | Object Info… | I | `DoObjectInfo()` |
| 7 | *(divider)* | | |
| 8 | Cut Room | X | `DeleteObject()`/`DeleteRoom(false)` — *no scrap* |
| 9 | Copy Room | C | **no-op** |
| 10 | Paste Room | V | **no-op** |
| 11 | Delete Room | | `DeleteObject()`/`DeleteRoom(false)` |
| 12 | Duplicate Object | D | `DuplicateObject()` |
| 13 | *(divider)* | | |
| 14 | Bring To Front | = | `BringSendFrontBack(true)` |
| 15 | Send To Back | - | `BringSendFrontBack(false)` |
| 16 | *(divider)* | | |
| 17 | Go To Room… | G | `DoGoToDialog()` |
| 18 | *(divider)* | | |
| 19 | Map Window | M | `ToggleMapWindow()` |
| 20 | Tools Window | T | `ToggleToolsWindow()` |
| 21 | Coordinate Window | K | `ToggleCoordinateWindow()` |

Items 8-11 have their titles swapped at runtime by `UpdateClipboardMenus()` between
"Cut/Copy/Clear Object" (`STR# 150` indices 36, 37, 38) and "Cut/Copy/Clear Room"
(39, 40, 41) depending on whether an object is selected; Paste is *always* disabled and
retitled "Nothing To Paste" (`STR# 150` index 44).

## 1.5 Editor keyboard (`GliderPRO/Sources/Events.c:180`)

`GliderDefines.h:16` contains `#define BUILD_ARCADE_VERSION 1`. In the shipped
configuration the four arrow keys are reassigned to arcade shortcuts:

| Key | ASCII | With `BUILD_ARCADE_VERSION` | In the `#else` branch (dead) |
|---|---|---|---|
| Left | 0x1C | `DoOptionsMenu(iHighScores)` | `SelectNeighborRoom(kRoomToLeft)` / `MoveObject(kBumpLeft, shift)` |
| Right | 0x1D | `DoOptionsMenu(iHelp)` | `SelectNeighborRoom(kRoomToRight)` / `MoveObject(kBumpRight, shift)` |
| Up | 0x1E | `DoGameMenu(iNewGame)` | `SelectNeighborRoom(kRoomAbove)` / `MoveObject(kBumpUp, shift)` |
| Down | 0x1F | `DoGameMenu(iNewGame)` | `SelectNeighborRoom(kRoomBelow)` / `MoveObject(kBumpDown, shift)` |

Live editor keys:

| Key | ASCII / keymap | Action |
|---|---|---|
| Page Up | `kPageUpKeyASCII` 0x0B | `PrevToolMode()` |
| Page Down | `kPageDownKeyASCII` 0x0C | `NextToolMode()` |
| Delete | `kDeleteKeyASCII` 0x08 | `DeleteObject()` if something selected, else `DeleteRoom(true)` |
| Cmd+Option held | `kCommandKeyMap` 48 / `kOptionKeyMap` 61 | `HiliteAllObjects()` — flashes all 24 object rects |

So **arrow-key nudging of objects does not exist in the shipped build** (§15.3).

---

# 2. Editor global state

Every piece of editor state is a global. A Go port will want these in one struct.

## 2.1 Selection (`GliderPRO/Sources/ObjectEdit.c:28`)

```c
Rect     roomObjectRects[kMaxRoomObs];    // 24 window-local rects, one per slot
Rect     initialGliderRect;               // where the game's very first glider appears
Rect     leftStartGliderSrc, rightStartGliderSrc;   // sprite source rects
Rect     leftStartGliderDest, rightStartGliderDest; // the two "new glider" markers
short    objActive;                       // the selection
Boolean  isFirstRoom;                     // this room == GetFirstRoomNumber()
```

`objActive` is an object index 0..23, or one of four sentinels
(`GliderPRO/Headers/GliderDefines.h`):

| Constant | Value | Meaning |
|---|---:|---|
| `kNoObjectSelected` | −1 | nothing selected |
| `kInitialGliderSelected` | −2 | the house-wide starting position `(*thisHouse)->initial` |
| `kLeftGliderSelected` | −3 | this room's `leftStart` respawn height |
| `kRightGliderSelected` | −4 | this room's `rightStart` respawn height |

Note `kNoObjectSelected` and `kObjectIsEmpty` are both −1 but mean different things
(`kObjectIsEmpty` is an object's `what` code for "slot unused").

## 2.2 Tools (`GliderPRO/Sources/Tools.c:57`)

```c
Rect          toolsWindowRect, toolSrcRect, toolTextRect;
Rect          toolRects[kTotalTools];     // 16
ControlHandle classPopUp;
GWorldPtr     toolSrcMap;
WindowPtr     toolsWindow;
short         isToolsH, isToolsV;         // sticky window position
short         toolSelected, toolMode;
short         firstTool, lastTool, objectBase;
Boolean       isToolsOpen;
```

`toolSelected` is 0 (`kSelectTool`) or 1..15; `toolMode` is 1..9. `objectBase` and
`firstTool` are assigned in `SwitchToolModes` and then **never read** — dead state
(`GliderPRO/Sources/Tools.c:379`-`429`). Only `lastTool` is used (hit-test clamp).

## 2.3 Marquee (`GliderPRO/Headers/Marquee.h:14`)

```c
typedef struct {
    Pattern pats[kNumMarqueePats];   // 7 patterns, 8 bytes each
    Rect    bounds;
    Rect    handle;
    short   index;                   // animation phase 0..6
    short   direction;               // kAbove(1)..kTopCorner(6); never set to -1
    short   dist;
    Boolean active, paused, handled;
} marquee;
extern marquee theMarquee;
```

## 2.4 Link (`GliderPRO/Sources/Link.c:26`)

```c
Rect          linkWindowRect;
ControlHandle linkControl, unlinkControl;
WindowPtr     linkWindow;
short         isLinkH, isLinkV;    // sticky position
short         linkRoom, linkType;
Byte          linkObject;
Boolean       isLinkOpen, linkerIsSwitch;
```

## 2.5 Map (`GliderPRO/Sources/Map.c`)

```c
Rect          mapWindowRect, mapCenterRect;
Rect          activeRoomRect, wasActiveRoomRect;
ControlHandle mapHScroll, mapVScroll;
GWorldPtr     nailSrcMap;
WindowPtr     mapWindow;
short         isMapH, isMapV, isMapWide, isMapHigh;
short         mapRoomsWide, mapRoomsHigh, mapLeftRoom, mapTopRoom;
Boolean       isMapOpen;
```

`mapLeftRoom` / `mapTopRoom` are persisted in the preferences (initialized to 60 / 50
by `InitializeEmptyHouse`).

## 2.6 House / room (`GliderPRO/Sources/House.c`, `Room.c`)

| Global | Type | Meaning |
|---|---|---|
| `thisHouse` | `houseHand` (`houseType**`) | the *whole file* as one relocatable block |
| `thisRoom` | `roomPtr` | a **detached working copy** of the current room |
| `thisRoomNumber` | `short` | index of `thisRoom` inside `(*thisHouse)->rooms[]` |
| `previousRoom` | `short` | for map redraw |
| `numberRooms` | `short` | mirror of `(*thisHouse)->nRooms` |
| `houseUnlocked` | `Boolean` | see §1.3 |
| `noRoomAtAll` | `Boolean` | house has zero real rooms |
| `fileDirty` | `Boolean` | needs saving |
| `gameDirty` | `Boolean` | saved-game data dirty |
| `wasHouseVersion` | `short` | version as read from disk |
| `lastBackground` | `short` | background ID for the next new room |
| `autoRoomEdit` / `newRoomNow` | `Boolean` | pop Room Info immediately after creating a room |
| `doBitchDialogs` | `Boolean` | show confirmation alerts |
| `doPrettyMap` | `Boolean` | draw real background PICTs on the map instead of thumbnails |
| `retroLinkList[kMaxRoomObs]` | `retroLink[24]` | "who links *to* slot i of this room" |
| `wasFlower` | `short` | last flower variety used |
| `changeLockStateOfHouse`, `saveHouseLocked` | `Boolean` | pending lock request |

**`thisRoom` is a detached copy.** `CopyRoomToThisRoom(n)`
(`GliderPRO/Sources/Room.c:343`) `BlockMove`s `(*thisHouse)->rooms[n]` into the
`thisRoom` pointer; `CopyThisRoomToRoom()` (`:354`) writes it back. Every code path
that navigates away from a room, saves, or reorders objects must call
`CopyThisRoomToRoom()` first — and several bugs in the editor are missing writebacks.
`ForceThisRoom(n)` (`:369`) is `CopyRoomToThisRoom` plus `thisRoomNumber = n` plus
`DetermineRoomOpenings()`.

---

# 3. The data the editor writes

The editor's whole job is to produce a byte-exact `houseType` image. Because Go must
reproduce it, here is the layout, **verified against real files**.

## 3.1 Alignment and endianness

`GliderPRO/Headers/Externs.h:231` contains `#pragma options align=mac68k`: 2-byte
struct alignment, `Boolean`/`Byte` are 1 byte and are *not* padded to even offsets
inside a struct unless a `short` follows an odd number of bytes. All multi-byte
integers are **big-endian**. QuickDraw `Point` is `{short v; short h;}` — **v first**.
QuickDraw `Rect` is `{short top, left, bottom, right;}` — **top-left-bottom-right**.
Both orderings are a classic porting trap.

## 3.2 `objectType` — 12 bytes (`GliderPRO/Headers/GliderStructs.h:90`)

```c
typedef struct {
    short what;                  // offset 0
    union { blowerType a; furnitureType b; bonusType c; transportType d;
            switchType e; lightType f; applianceType g; enemyType h;
            clutterType i; } data;   // offset 2, 10 bytes
} objectType;                    // total = 12
```

The nine variants, all exactly 10 bytes (`GliderPRO/Headers/GliderStructs.h:11`-`88`).
Offsets below are *relative to the start of the 12-byte object record*:

| Variant | Field | Type | Off | Size |
|---|---|---|--:|--:|
| **a** `blowerType` | `topLeft` | `Point` (v,h) | 2 | 4 |
| | `distance` | `short` | 6 | 2 |
| | `initial` | `Boolean` | 8 | 1 |
| | `state` | `Boolean` | 9 | 1 |
| | `vector` | `Byte` | 10 | 1 |
| | `tall` | `Byte` | 11 | 1 |
| **b** `furnitureType` | `bounds` | `Rect` (t,l,b,r) | 2 | 8 |
| | `pict` | `short` | 10 | 2 |
| **c** `bonusType` | `topLeft` | `Point` | 2 | 4 |
| | `length` | `short` | 6 | 2 |
| | `points` | `short` | 8 | 2 |
| | `state` | `Boolean` | 10 | 1 |
| | `initial` | `Boolean` | 11 | 1 |
| **d** `transportType` | `topLeft` | `Point` | 2 | 4 |
| | `tall` | `short` | 6 | 2 |
| | `where` | `short` | 8 | 2 |
| | `who` | `Byte` | 10 | 1 |
| | `wide` | `Byte` | 11 | 1 |
| **e** `switchType` | `topLeft` | `Point` | 2 | 4 |
| | `delay` | `short` | 6 | 2 |
| | `where` | `short` | 8 | 2 |
| | `who` | `Byte` | 10 | 1 |
| | `type` | `Byte` | 11 | 1 |
| **f** `lightType` | `topLeft` | `Point` | 2 | 4 |
| | `length` | `short` | 6 | 2 |
| | `byte0` | `Byte` | 8 | 1 |
| | `byte1` | `Byte` | 9 | 1 |
| | `initial` | `Boolean` | 10 | 1 |
| | `state` | `Boolean` | 11 | 1 |
| **g** `applianceType` | `topLeft` | `Point` | 2 | 4 |
| | `height` | `short` | 6 | 2 |
| | `byte0` | `Byte` | 8 | 1 |
| | `delay` | `Byte` | 9 | 1 |
| | `initial` | `Boolean` | 10 | 1 |
| | `state` | `Boolean` | 11 | 1 |
| **h** `enemyType` | `topLeft` | `Point` | 2 | 4 |
| | `length` | `short` | 6 | 2 |
| | `delay` | `Byte` | 8 | 1 |
| | `byte0` | `Byte` | 9 | 1 |
| | `initial` | `Boolean` | 10 | 1 |
| | `state` | `Boolean` | 11 | 1 |
| **i** `clutterType` | `bounds` | `Rect` | 2 | 8 |
| | `pict` | `short` | 10 | 2 |

Critically, **`data.d.where` and `data.e.where` are at the same offset (8), and
`data.d.who` and `data.e.who` are both at offset 10**. Several places in the source
use the "wrong" variant name to reach the link fields; because the union overlays them
identically this is harmless. Do not "fix" it in a port by giving transports and
switches different offsets.

The `vector` byte carries an undocumented fifth bit
(`GliderPRO/Headers/GliderStructs.h:16`-`17`):

```
// Boolean state;   //  1               F. lf. dn. rt. up
// Byte    vector;  //  1   | x | x | x | x | 8 | 4 | 2 | 1 |
```

so 0x01 = up, 0x02 = right, 0x04 = down, 0x08 = left, **0x10 = "F" (extra forceful)**.
Every read in the program masks `& 0x0F` (`GliderPRO/Sources/ObjectRects.c:523`, `:595`;
`GliderPRO/Sources/ObjectEdit.c:190`, `:1834`; `GliderPRO/Sources/ObjectInfo.c:955`), and
nothing anywhere tests 0x10 — the DITL checkbox for it exists but is positioned
off-dialog (§12.2). Four shipped objects still have it set (observed `vector` values
across all 22 houses: `{1: 4900, 2: 412, 4: 326, 8: 402, 17: 3, 20: 1}` — 0x11 and
0x14). **Opening Blower Info and clicking OK destroys the bit**, because
`GliderPRO/Sources/ObjectInfo.c:955` reads `vector & 0x0F` and `:1008` writes the
result straight back.

## 3.3 `roomType` — 348 bytes (`GliderPRO/Headers/GliderStructs.h:166`)

| Field | Type | Offset | Size | Notes |
|---|---|--:|--:|---|
| `name` | `Str27` | 0 | 28 | length byte + up to 27 chars |
| `bounds` | `short` | 28 | 2 | the "original artwork" bit field (§13.5) |
| `leftStart` | `Byte` | 30 | 1 | left respawn offset, added to `kGliderStartsDown` |
| `rightStart` | `Byte` | 31 | 1 | right respawn offset |
| `unusedByte` | `Byte` | 32 | 1 | |
| `visited` | `Boolean` | 33 | 1 | gameplay |
| `background` | `short` | 34 | 2 | PICT ID |
| `tiles[8]` | `short[8]` | 36 | 16 | tile index 0..7 into the background strip |
| `floor` | `short` | 52 | 2 | map row (0 = ground) |
| `suite` | `short` | 54 | 2 | map column; `kRoomIsEmpty` (−1) = deleted slot |
| `openings` | `short` | 56 | 2 | **always 0 in every shipped house** |
| `numObjects` | `short` | 58 | 2 | count of non-empty slots |
| `objects[24]` | `objectType[24]` | 60 | 288 | |

Observed across the 4070 rooms in all 22 houses:

* `tiles[]` values range exactly `[0, 7]` (`kNumTiles` = 8) — no out-of-range tiles;
* `leftStart` / `rightStart` range `[0, 255]` — the full `Byte`;
* `floor` range `[-7, 39]`, `suite` range `[0, 127]`;
* `openings` is `0` in **4070 / 4070** rooms;
* max room-name length is exactly **27** (the `CheckRoomNameLength` cap);
* `numObjects` matched the live-slot count in **4070 / 4070** rooms;
* max live objects in a single room is exactly **24** (`kMaxRoomObs`) — the cap is
  reached in practice;
* zero rooms have `suite == kRoomIsEmpty`, and zero duplicate `(floor, suite)` pairs
  exist — i.e. every shipped house is already "compressed" and validated.

## 3.4 `houseType` — 866-byte prefix (`GliderPRO/Headers/GliderStructs.h:182`)

The source comment literally reads `// total = 866 +`.

| Field | Type | Offset | Size |
|---|---|--:|--:|
| `version` | `short` | 0 | 2 |
| `unusedShort` | `short` | 2 | 2 |
| `timeStamp` | `long` | 4 | 4 |
| `flags` | `long` | 8 | 4 |
| `initial` | `Point` (v,h) | 12 | 4 |
| `banner` | `Str255` | 16 | 256 |
| `trailer` | `Str255` | 272 | 256 |
| `highScores` | `scoresType` | 528 | 292 |
| `savedGame` | `gameType` | 820 | 40 |
| `hasGame` | `Boolean` | 860 | 1 |
| `unusedBoolean` | `Boolean` | 861 | 1 |
| `firstRoom` | `short` | 862 | 2 |
| `nRooms` | `short` | 864 | 2 |
| `rooms[]` | `roomType[nRooms]` | 866 | 348·n |

`scoresType` = 292 (`Str31 banner` 32 + `Str15 names[10]` 160 + `long scores[10]` 40 +
`unsigned long timeStamps[10]` 40 + `short levels[10]` 20). `gameType` = 40.

### Verification: `866 + 348·nRooms == data-fork length`

Observed, decoding all 22 BinHex files (`type='gliH'`, `creator='ozm5'` in every case):

| House | data fork | rsrc fork | nRooms | 866+348·n | version | timeStamp | firstRoom | unlocked | flags |
|---|--:|--:|--:|--:|---|---|--:|---|---|
| Art Museum | 38798 | 7476159 | 109 | 38798 ✓ | 0x0200 | 0x2C808C35 | 91 | false | 0x00000006 |
| CD Demo House | 72554 | 1612342 | 206 | 72554 ✓ | 0x0200 | 0x2C3E8559 | 70 | false | 0x00000002 |
| California or Bust! | 6434 | 259542 | 16 | 6434 ✓ | 0x0200 | 0x2C7542B8 | 14 | true | 0x00000002 |
| Castle o' the Air | 30446 | 306364 | 85 | 30446 ✓ | 0x0200 | 0x2C32F464 | 33 | true | 0x00000000 |
| Davis Station | 23486 | 1871320 | 65 | 23486 ✓ | 0x0200 | 0x2C53AEFD | 4 | false | 0x00000002 |
| Demo House | 16526 | 491757 | 45 | 16526 ✓ | 0x0200 | 0x2C53B041 | 0 | false | 0x00000000 |
| Empty House | 13046 | 2670 | 35 | 13046 ✓ | 0x0200 | 0x2C3130EC | 0 | true | 0x00000000 |
| Fun House | 15830 | 662443 | 43 | 15830 ✓ | 0x0200 | 0x2C446C10 | 29 | true | 0x00000000 |
| Grand Prix | 61766 | 1347764 | 175 | 61766 ✓ | 0x0200 | 0x2C318A3D | 127 | false | 0x00000000 |
| ImagineHouse PRO II | 97958 | 677770 | 279 | 97958 ✓ | 0x0200 | 0x2C1DC93D | 1 | false | 0x00000000 |
| In The Mirror | 34622 | 151870 | 97 | 34622 ✓ | 0x0200 | 0x2C329401 | 6 | false | 0x00000000 |
| Land of Illusion | 106310 | 401793 | 303 | 106310 ✓ | 0x0200 | 0x2C6BE354 | 43 | true | 0x00000002 |
| Leviathan | 165122 | 1901282 | 472 | 165122 ✓ | 0x0200 | 0x2C329E33 | 39 | false | 0x00000000 |
| Metropolis | 45062 | 946907 | 127 | 45062 ✓ | 0x0200 | 0x2C34904B | 8 | false | 0x00000000 |
| Nemo's Market | 44018 | 590602 | 124 | 44018 ✓ | 0x0200 | 0x2C2C9CD1 | 0 | false | 0x00000002 |
| Rainbow's End | 78470 | 316065 | 223 | 78470 ✓ | 0x0200 | 0x2C2C9DD1 | 30 | false | 0x00000002 |
| **Sampler** | **1564** | 286 | 2 | **1562 ✗ (+2)** | 0x0200 | 0x3540BFE8 | 1 | true | 0x00000000 |
| Slumberland | 134150 | 1041112 | 383 | 134150 ✓ | 0x0200 | 0x2C53B149 | 126 | false | 0x00000000 |
| SpacePods | 140762 | 636844 | 402 | 140762 ✓ | 0x0200 | 0x2C2CA019 | 259 | false | 0x00000002 |
| Teddy World | 185654 | 3107348 | 531 | 185654 ✓ | 0x0200 | 0x2C5EAB21 | 0 | false | 0x00000000 |
| The Asylum Pro | 49586 | 494107 | 140 | 49586 ✓ | 0x0200 | 0x2BFD3EA1 | 20 | false | 0x00000000 |
| Titanic | 73250 | 845727 | 208 | 73250 ✓ | 0x0200 | 0x2D0246FE | 92 | true | 0x00000000 |

21 of 22 are byte-exact. **"Sampler" carries 2 trailing bytes of slack.** The original
tolerates this because `ValidateNumberOfRooms` (`GliderPRO/Sources/HouseLegal.c:621`)
computes `countedRooms = (GetHandleSize(thisHouse) - sizeof(houseType)) / sizeof(roomType)`
with integer division: `(1564 − 866) / 348 = 2 == nRooms`. **A Go loader must use
truncating division on the file length, not an exact-size assertion.**

`flags` bit meanings (`GliderPRO/Sources/HouseIO.c`, `ReadHouse`):

| Bit | Mask | Name | Meaning |
|--:|---|---|---|
| 0 | 0x00000001 | `wardBitSet` | Easter-egg |
| 1 | 0x00000002 | `phoneBitSet` | "no phone number" checkbox in House Info |
| 2 | 0x00000004 | | `bannerStarCountOn = ((flags & 4) == 0)` |

## 3.5 The 117 object types, in nine classes, over a 144-entry index space

`what` is a direct index into `STR# 1007` ("Object Names"): index 1 = "Floor Vent" =
`what` 0x01.

**144 is an array size, not a type count.** `GliderPRO/Headers/GliderDefines.h:311-435`
declares exactly **117** object-type `#define`s with 117 distinct codes, running `0x01`
(`kFloorVent`) to `0x8F` (`kChimes`) with the six inter-class gap runs listed below
(`0x20`, `0x30`, `0x4A`-`0x50`, `0x59`-`0x60`, `0x6F`-`0x70`, `0x7A`-`0x80` = 26 codes);
`kNumSrcRects` (`:437`) = `0x90` = **144** is only the length of `srcRects[]` and of the
`STR# 1007` name list, with index 0 unused (`kObjectIsEmpty` is −1, `:526`) and 26 of the
remaining indices unassigned: `144 − 1 − 26 = 117`. `InitSrcRects` (`GliderPRO/Sources/StructuresInit2.c:306-475`)
writes only **116** of the 144 slots — `kFlower` (`0x85`) is a real type but never gets a
default rect — so 28 slots are never written at all. See object-taxonomy.md §2.2/§3.
A Go port sizes its dispatch table `[144]` but must populate only these 117 codes.

| Class | Range | Members (value: name) |
|---|---|---|
| Blowers | 0x01-0x10 | 01 kFloorVent, 02 kCeilingVent, 03 kFloorBlower, 04 kCeilingBlower, 05 kSewerGrate, 06 kLeftFan, 07 kRightFan, 08 kTaper, 09 kCandle, 0A kStubby, 0B kTiki, 0C kBBQ, 0D kInvisBlower, 0E kGrecoVent, 0F kSewerBlower, 10 kLiftArea |
| Furniture | 0x11-0x1F | 11 kTable, 12 kShelf, 13 kCabinet, 14 kFilingCabinet, 15 kWasteBasket, 16 kMilkCrate, 17 kCounter, 18 kDresser, 19 kDeckTable, 1A kStool, 1B kTrunk, 1C kInvisObstacle, 1D kManhole, 1E kBooks, 1F kInvisBounce |
| Prizes | 0x21-0x2F | 21 kRedClock, 22 kBlueClock, 23 kYellowClock, 24 kCuckoo, 25 kPaper, 26 kBattery, 27 kBands, 28 kGreaseRt, 29 kGreaseLf, 2A kFoil, 2B kInvisBonus, 2C kStar, 2D kSparkle, 2E kHelium, 2F kSlider |
| Transport | 0x31-0x40 | 31 kUpStairs, 32 kDownStairs, 33 kMailboxLf, 34 kMailboxRt, 35 kFloorTrans, 36 kCeilingTrans, 37 kDoorInLf, 38 kDoorInRt, **39 kDoorExRt, 3A kDoorExLf**, 3B kWindowInLf, 3C kWindowInRt, **3D kWindowExRt, 3E kWindowExLf**, 3F kInvisTrans, 40 kDeluxeTrans |
| Switches | 0x41-0x49 | 41 kLightSwitch, 42 kMachineSwitch, 43 kThermostat, 44 kPowerSwitch, 45 kKnifeSwitch, 46 kInvisSwitch, 47 kTrigger, 48 kLgTrigger, 49 kSoundTrigger |
| Lights | 0x51-0x58 | 51 kCeilingLight, 52 kLightBulb, 53 kTableLamp, 54 kHipLamp, 55 kDecoLamp, 56 kFlourescent, 57 kTrackLight, 58 kInvisLight |
| Appliances | 0x61-0x6E | 61 kShredder, 62 kToaster, 63 kMacPlus, 64 kGuitar, 65 kTV, 66 kCoffee, 67 kOutlet, 68 kVCR, 69 kStereo, 6A kMicrowave, 6B kCinderBlock, 6C kFlowerBox, 6D kCDs, 6E kCustomPict |
| Enemies | 0x71-0x79 | 71 kBalloon, 72 kCopterLf, 73 kCopterRt, 74 kDartLf, 75 kDartRt, 76 kBall, 77 kDrip, 78 kFish, 79 kCobweb |
| Clutter | 0x81-0x8F | 81 kOzma, 82 kMirror, 83 kMousehole, 84 kFireplace, 85 kFlower, 86 kWallWindow, 87 kBear, 88 kCalendar, 89 kVase1, 8A kVase2, 8B kBulletin, 8C kCloud, 8D kFaucet, 8E kRug, 8F kChimes |

**Note the exterior door/window pairs are Rt-then-Lf, reversed relative to the
interior pairs.** This is a real asymmetry in the enum, not a typo.

Empirical confirmation of the class boundaries: scanning every object in all 22
houses, the set of `what` codes actually used is 117 distinct values, and the unused
codes are *exactly* the inter-class gaps:

```
0x20, 0x30, 0x4A 0x4B 0x4C 0x4D 0x4E 0x4F 0x50,
0x59 0x5A 0x5B 0x5C 0x5D 0x5E 0x5F 0x60, 0x6F 0x70,
0x7A 0x7B 0x7C 0x7D 0x7E 0x7F 0x80
```

Which variant each class uses (this mapping is repeated by hand in ten different
`switch` statements — `GetObjectRect`, `GetThisRoomsObjRects`, `KeepObjectLegal`,
`DragObject`, `DragHandle`, `MoveObject`, `DuplicateObject`, `AddNewObject`,
`CreateActiveRects`, `DrawThisRoomsObjects`):

| Variant | Used by |
|---|---|
| **a** blower | 0x01-0x10 (all blowers, flames, fans, `kLiftArea`) |
| **b** furniture | 0x11-0x1F (all furniture) |
| **c** bonus | 0x21-0x2F (all prizes) |
| **d** transport | 0x31-0x40 (all transport) |
| **e** switch | 0x41-0x49 (all switches/triggers) |
| **f** light | 0x51-0x58 (all lights) |
| **g** appliance | 0x61-0x6E (all appliances, incl. `kCustomPict`) |
| **h** enemy | 0x71-0x79 (all enemies) |
| **i** clutter | 0x81-0x8F (all clutter) |

## 3.6 `srcRects[]` — the canonical size table

`srcRects` is `NewPtr(sizeof(Rect) * kNumSrcRects)` at
`GliderPRO/Sources/StructuresInit2.c:270`-`273` — **not zeroed** — and filled by
`InitSrcRects()` at `GliderPRO/Sources/StructuresInit2.c:306`-`475`. Each entry is
both the sprite's source rect inside its class GWorld *and* the object's default size.
Since `QSetRect(r, left, top, right, bottom)`, the calls below give width×height at the
listed sprite-sheet offset.

| `what` | Name | W×H | Sheet offset (h,v) |
|---|---|---|---|
| 0x01 | kFloorVent | 48×11 | (0,0) |
| 0x02 | kCeilingVent | 48×11 | (0,11) |
| 0x03 | kFloorBlower | 48×15 | (0,22) |
| 0x04 | kCeilingBlower | 48×15 | (0,37) |
| 0x05 | kSewerGrate | 48×17 | (0,52) |
| 0x06 | kLeftFan | 40×55 | (0,69) |
| 0x07 | kRightFan | 40×55 | (0,124) |
| 0x08 | kTaper | 20×59 | (0,209) |
| 0x09 | kCandle | 32×30 | (0,179) |
| 0x0A | kStubby | 20×36 | (0,268) |
| 0x0B | kTiki | 27×28 | (21,268) |
| 0x0C | kBBQ | 64×33 | (0,0) |
| 0x0D | kInvisBlower | 24×24 | (0,0) |
| 0x0E | kGrecoVent | 48×18 | (0,340) |
| 0x0F | kSewerBlower | 32×12 | (0,390) |
| 0x10 | kLiftArea | 64×32 | (0,0) |
| 0x11 | kTable | 64×8 (`kTableThick`) | (0,0) |
| 0x12 | kShelf | 64×6 (`kShelfThick`) | (0,0) |
| 0x13 | kCabinet | 64×64 | (0,0) |
| 0x14 | kFilingCabinet | 74×107 | (0,0) |
| 0x15 | kWasteBasket | 64×61 | (0,43) |
| 0x16 | kMilkCrate | 64×58 | (0,104) |
| 0x17 | kCounter | 128×64 | (0,0) |
| 0x18 | kDresser | 128×64 | (0,0) |
| 0x19 | kDeckTable | 64×8 | (0,0) |
| 0x1A | kStool | 48×38 | (0,183) |
| 0x1B | kTrunk | 144×80 | (0,0) |
| 0x1C | kInvisObstacle | 64×64 | (0,0) |
| 0x1D | kManhole | 123×22 | (0,0) |
| 0x1E | kBooks | 64×51 | (0,0) |
| 0x1F | kInvisBounce | 64×64 | (0,0) |
| 0x21 | kRedClock | 28×17 | (0,0) |
| 0x22 | kBlueClock | 28×25 | (0,17) |
| 0x23 | kYellowClock | 28×28 | (0,42) |
| 0x24 | kCuckoo | 40×80 | (0,148) |
| 0x25 | kPaper | 48×21 | (0,127) |
| 0x26 | kBattery | 16×25 | (32,0) |
| 0x27 | kBands | 28×23 | (20,70) |
| 0x28 | kGreaseRt | 32×27 | (0,243) |
| 0x29 | kGreaseLf | 32×27 | (0,324) |
| 0x2A | kFoil | 55×15 | (0,228) |
| 0x2B | kInvisBonus | 24×24 | (0,0) |
| 0x2C | kStar | 32×31 | (48,0) |
| 0x2D | kSparkle | 20×19 | (0,70) |
| 0x2E | kHelium | 56×16 | (32,270) |
| 0x2F | kSlider | 64×16 | (0,0) |
| 0x31 | kUpStairs | 160×267 | (0,0) |
| 0x32 | kDownStairs | 160×267 | (0,0) |
| 0x33 | kMailboxLf | 94×80 | (0,0) |
| 0x34 | kMailboxRt | 94×80 | (0,0) |
| 0x35 | kFloorTrans | 56×15 | (0,1) |
| 0x36 | kCeilingTrans | 56×15 | (0,16) |
| 0x37 | kDoorInLf | 144×322 | (0,0) |
| 0x38 | kDoorInRt | 144×322 | (0,0) |
| 0x39 | kDoorExRt | 16×322 | (0,0) |
| 0x3A | kDoorExLf | 16×322 | (0,0) |
| 0x3B | kWindowInLf | 20×170 | (0,0) |
| 0x3C | kWindowInRt | 20×170 | (0,0) |
| 0x3D | kWindowExRt | 16×170 | (0,0) |
| 0x3E | kWindowExLf | 16×170 | (0,0) |
| 0x3F | kInvisTrans | 64×32 | (0,0) |
| 0x40 | kDeluxeTrans | 64×64 | (0,0) |
| 0x41 | kLightSwitch | 15×24 | (0,0) |
| 0x42 | kMachineSwitch | 16×24 | (0,48) |
| 0x43 | kThermostat | 15×24 | (0,48) |
| 0x44 | kPowerSwitch | 8×8 | (0,72) |
| 0x45 | kKnifeSwitch | 16×24 | (0,80) |
| 0x46 | kInvisSwitch | 12×12 | (0,0) |
| 0x47 | kTrigger | 12×12 | (0,0) |
| 0x48 | kLgTrigger | 48×48 | (0,0) |
| 0x49 | kSoundTrigger | 32×32 | (0,0) |
| 0x51 | kCeilingLight | 64×20 | (0,0) |
| 0x52 | kLightBulb | 16×28 | (0,20) |
| 0x53 | kTableLamp | 48×70 | (16,20) |
| 0x54 | kHipLamp | 72×276 | (0,0) |
| 0x55 | kDecoLamp | 64×212 | (0,0) |
| 0x56 | kFlourescent | 64×12 | (0,0) |
| 0x57 | kTrackLight | 64×24 | (0,0) |
| 0x58 | kInvisLight | 16×16 | (0,0) |
| 0x61 | kShredder | 73×22 | (0,0) |
| 0x62 | kToaster | 48×27 | (0,22) |
| 0x63 | kMacPlus | 48×58 | (0,49) |
| 0x64 | kGuitar | 64×172 | (0,0) |
| 0x65 | kTV | 92×77 | (0,0) |
| 0x66 | kCoffee | 43×64 | (0,107) |
| 0x67 | kOutlet | 16×24 | (64,22) |
| 0x68 | kVCR | 96×22 | (0,0) |
| 0x69 | kStereo | 128×53 | (0,0) |
| 0x6A | kMicrowave | 92×59 | (0,0) |
| 0x6B | kCinderBlock | 40×62 | (0,0) |
| 0x6C | kFlowerBox | 80×32 | (0,0) |
| 0x6D | kCDs | 16×30 | (48,22) |
| 0x6E | kCustomPict | 72×34 | (0,0) |
| 0x71 | kBalloon | 24×30 | (0,0) |
| 0x72 | kCopterLf | 32×30 | (0,0) |
| 0x73 | kCopterRt | 32×30 | (0,0) |
| 0x74 | kDartLf | 64×19 | (0,0) |
| 0x75 | kDartRt | 64×19 | (0,0) |
| 0x76 | kBall | 32×32 | (0,0) |
| 0x77 | kDrip | 16×12 | (0,0) |
| 0x78 | kFish | 36×33 | (0,0) |
| 0x79 | kCobweb | 54×45 | (0,0) |
| 0x81 | kOzma | 102×92 | (0,0) |
| 0x82 | kMirror | 64×64 | (0,0) |
| 0x83 | kMousehole | 10×11 | (0,0) |
| 0x84 | kFireplace | 180×142 | (0,0) |
| **0x85** | **kFlower** | **NEVER SET** | — |
| 0x86 | kWallWindow | 64×80 | (0,0) |
| 0x87 | kBear | 56×58 | (0,0) |
| 0x88 | kCalendar | 63×92 | (0,0) |
| 0x89 | kVase1 | 36×45 | (0,0) |
| 0x8A | kVase2 | 35×57 | (0,0) |
| 0x8B | kBulletin | 80×58 | (0,0) |
| 0x8C | kCloud | 128×30 | (0,0) |
| 0x8D | kFaucet | 56×18 | (0,51) |
| 0x8E | kRug | 144×18 | (0,0) |
| 0x8F | kChimes | 28×74 | (0,0) |

**`srcRects[kFlower]` (0x85) is never initialized** and the pointer is not cleared, so
it contains whatever was in the heap. Nothing reads it: `kFlower` always goes through
`flowerSrc[data.i.pict]` instead (`GliderPRO/Sources/StructuresInit2.c:72`-`88`):

| `data.i.pict` | Flower | W×H | Offset in PICT 4018 |
|--:|---|---|---|
| 0 | Dandelion | 10×28 | (0,23) |
| 1 | Tulip | 24×35 | (10,16) |
| 2 | Orchid | 34×35 | (34,16) |
| 3 | Violets | 27×23 | (68,14) |
| 4 | Daisies | 27×14 | (68,37) |
| 5 | Sunflower | 32×51 | (95,0) |

Observed `data.i.pict` distribution **of the `kFlower` objects** across all houses:
`{0:84, 1:83, 2:56, 3:81, 4:117, 5:126}` (547 flowers) — all six varieties in use, none
out of range. (Counted over kFlower only; every other clutter type leaves `pict` at 0, so
the distribution over the whole clutter class is dominated by a `0` bucket of 3676.)

## 3.7 Placement constants (`GliderPRO/Headers/GliderDefines.h:467`-`494`)

| Constant | Value | Constant | Value |
|---|--:|---|--:|
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
| `kDecoLampTop` | 91 | `kGliderStartsDown` | 32 |
| `kFlourescentTop` | 12 | `kRoomWide` | 512 |
| `kTrackLightTop` | 5 | `kTileHigh` | 322 |

Plus, local `#define`s at the top of `GliderPRO/Sources/ObjectAdd.c:19`-`23`:
`kMouseholeBottom` 295, `kFireplaceBottom` 297, `kManholeSits` 322, `kGrecoVentTop` 303,
`kSewerBlowerTop` 292. (The same file also defines `kNoMoreObjectsAlert` 1008 and
`kNoMoreSpecialAlert` 1028 at `:15`-`:16`, and `kMaxSoundTriggers` 1 / `kMaxStairs` 1 at
`:17`-`:18` — those two caps are *not* in `GliderDefines.h`.)

### Verified against real data

Scanning every object in all 22 houses:

| Rule | Source | Observed |
|---|---|---|
| `kCounter.bounds.bottom == 304` | `kCounterBottom` | 287/287 ✓ |
| `kDresser.bounds.bottom == 293` | `kDresserBottom` | 122/122 ✓ |
| `kManhole.bounds.left ≡ 3 (mod 64)` | `HouseLegal.c` | 40/40 ✓ (`left % 64 == 3` in every case) |
| `kMirror.bounds.left` and `.right` both even | `HouseLegal.c` | 667/667 ✓ |
| `kTV.bounds.left` odd | `HouseLegal.c` | 80/80 ✓ |
| `kStubby.topLeft.h` odd | `HouseLegal.c:103` | 127/127 ✓ |
| `kFloorVent/kFloorBlower/kSewerGrate/kCeilingVent/…` fixed tops | `HouseLegal.c:115`-`131` | no violations |
| Doors/windows at their `k*Left` and `k*Top` | `HouseLegal.c` | no violations |
| `kFloorTrans.topLeft.v == 302` | `HouseLegal.c:132` | **201 at 302, 43 at 300** |

The `kFloorTrans` exception is genuine and instructive: `KeepObjectLegal` only forces
`v = kFloorTransTop` when the object is *touched by the editor*, and legacy 1.0 houses
carry `v = 300`. Do not normalize on load; normalize on edit, exactly as the original.

---

# 4. The main edit window

## 4.1 Geometry (`GliderPRO/Sources/MainWindow.c:174`)

`OpenMainWindow()` has exactly **two** branches — `kEditMode`, and one shared branch for
splash *and* play:

| Branch | `mainWindowRect` (l,t,r,b) | W×H | Window resource |
|---|---|---|---|
| `theMode == kEditMode` (`:185`) | `QSetRect(&mainWindowRect, 0, 0, 512, 322)` | **512×322** | `kEditWindowID` **129** |
| anything else — splash and play (`:212`) | `thisMac.screen`, corner zeroed, `bottom -= 20` | screen minus the 20-pixel menu bar | `kMainWindowID` **128** |

> [verified: the `WIND` resource IDs are `#define`d in
> `GliderPRO/Sources/MainWindow.c:16-18` as `kMainWindowID` 128, `kEditWindowID` 129,
> `kMenuWindowID` 130. The names "kSplashWindowID"/"kPlayWindowID" do not exist in the
> source. `kMenuWindowID` 130 is **not** the play-mode main window: it is a separate
> screen-wide, 20-pixel-tall window created in the non-edit branch (`:216`) to cover the
> real menu bar, and it is disposed on entry to edit mode (`:187-189`).]

The play/splash branch is *not* sized from `houseRect` and is not 640×460: it is always the
full screen below the menu bar. The 640×480 figure that shows up around splash mode is only
the *centring* of the splash PICT inside that window
(`splashOriginH = (screenWidth - 640) / 2`, `splashOriginV = (screenHeight - 480) / 2`,
both clamped at 0, `:238-243`).

So in edit mode the window is *exactly one room*: `kRoomWide` 512 × `kTileHigh` 322.
The room's coordinate system therefore **is** the window's local coordinate system —
`(0,0)` is the room's top-left corner. Objects' `topLeft`/`bounds` are stored directly
in these room coordinates. No scrolling, no zoom.

The title bar shows the current room (`UpdateEditWindowTitle`,
`GliderPRO/Sources/MainWindow.c:306`):

```
1. if mainWindow == nil: return
2. PasStringCopy(thisHouseName, newTitle)
3. PasStringConcat(newTitle, "\p - ")
4. if noRoomAtAll:
5.     PasStringConcat(newTitle, "\pNo rooms")
6. else if houseUnlocked:
7.     PasStringConcat(newTitle, thisRoom->name)
8.     PasStringConcat(newTitle, "\p (")
9.     NumToString(thisRoom->floor, tempStr);  PasStringConcat(newTitle, tempStr)
10.    PasStringConcat(newTitle, "\p, ")
11.    NumToString(thisRoom->suite, tempStr);  PasStringConcat(newTitle, tempStr)
12.    PasStringConcat(newTitle, "\p)")
13. else:
14.    PasStringConcat(newTitle, "\pHouse Locked")
15. SetWTitle(mainWindow, newTitle)
```

The three literal title strings are **hard-coded Pascal strings in the C, not `STR#`
resources** — `" - "`, `"No rooms"`, `"House Locked"`. So the edit-mode window title is
exactly `<house file name> - <room name> (<floor>, <suite>)`, e.g.
`Demo House - Foyer (0, 0)`. A locked house shows `Demo House - House Locked` and never
reveals the room name, which is the visible symptom of `houseUnlocked == false` (§1.3).
Note this uses `thisHouseName` (the *file* name), not the house banner.

## 4.2 Update (`GliderPRO/Sources/MainWindow.c:120`)

`UpdateMainWindow()` in edit mode:

```
1. BeginUpdate(mainWindow)
2. if theMode == kEditMode:
3.     if noRoomAtAll:
4.         PaintRect(&mainWindowRect)  in gray  (GetQDGlobalsGray)
5.     else:
6.         CopyBits(backSrcMap -> window, &backSrcRect, &mainWindowRect, srcCopy)
7.         // marquee and object handles are XOR'd on top by the idle loop
8. EndUpdate(mainWindow)
```

`backSrcMap` is the composited room: `ReadyBackground(background, tiles)` paints the
eight 64-px background tiles plus the floor support, and `DrawThisRoomsObjects()` then
blits every object's sprite into it. So the whole visible room is a single offscreen
GWorld that gets `CopyBits`'d in one shot.

## 4.3 Click routing (`GliderPRO/Sources/MainWindow.c:338`)

`HandleMainClick(wherePt, isDoubleClick)`:

```
1. if theMode != kEditMode: return
2. SetPortWindowPort(mainWindow); GlobalToLocal(&wherePt)
3. if !houseUnlocked:
4.     SysBeep(1); return                     // locked house: clicks do nothing
5. if toolSelected == kSelectTool:
6.     DoSelectionClick(wherePt, isDoubleClick)
7. else:
8.     DoNewObjectClick(wherePt)
```

So the palette's current tool decides between "select/drag" and "create". There is no
modifier-key override.

## 4.4 The XOR overlay: what edit mode draws that play mode does not

| Overlay | Source | Drawn when |
|---|---|---|
| the animated marquee | `DrawMarquee` (`GliderPRO/Sources/Marquee.c:455`) | `theMarquee.active && !theMarquee.paused` |
| the 9×9 handle (filled with the current marquee pattern in `patXor`, not black) plus its connector line | `DrawMarquee` | `theMarquee.handled` |
| the glider start marker (left) | `DrawThisRoomsObjects` | `thisRoom` has a left opening |
| the glider start marker (right) | ditto | right opening |
| the initial-glider marker | ditto | `isFirstRoom` |
| a dark-gray `srcOr` wash | `DrawThisRoomsObjects` | `GetNumberOfLights(thisRoomNumber) <= 0` |

The dark wash is how the editor tells you a room has no working light source —
`GliderPRO/Sources/ObjectEdit.c:2347` (`DrawThisRoomsObjects`). Objects themselves are
drawn in their **initial** state (`data.*.initial`), never animated: the editor renders
no flames, no pendulums, no balloons in motion.

---

# 5. The Tools palette

## 5.1 Window (`GliderPRO/Sources/Tools.c:277`)

```
QSetRect(&toolsWindowRect, 0, 0, 116, 152);        // 116 wide x 152 tall
QSetRect(&toolTextRect,    0, 0, 116, 12);
InsetRect(&toolTextRect, -1, -1);                  // -> -1,-1,117,13
QOffsetRect(&toolTextRect, 0, 157 - 15);           // -> -1,141,117,155
```

Title `"Tools"`, `procID = kWindoidWDEF` = 2048 (= `WDEF` resource 128, variant 0 — a
floating "windoid" with a narrow drag bar). Colour build uses `NewCWindow`, mono
`NewWindow`; `visible = false` then `MoveWindow(isToolsH, isToolsV)` then
`BringToFront` + `ShowHide(true)` + `HiliteAllWindows()`. `isToolsH`/`isToolsV` persist
in the preferences file. The commented-out block at `GliderPRO/Sources/Tools.c:300`-`304`
would have snapped it to the top-right of the screen when the Option key was held.

The class popup is `GetNewControl(kPopUpControl = 129, toolsWindow)`. `CNTL 129` in
`GliderPRO/Glider PRO.r` is a custom popup: `procID` 2048 + variant, i.e. `CDEF` 128,
bounds inside the windoid's top strip, `refCon` = the `MENU 141` id. Its value 1..9 is
`toolMode`.

## 5.2 Modes and their tool counts (`GliderPRO/Sources/Tools.c:19`-`45`)

| `toolMode` | Name | `MENU 141` item | `kFirst*` | `kLast*` | `k*Base` | `what` range |
|--:|---|---|--:|--:|--:|---|
| 1 | `kBlowerMode` | Blowers | 1 | **15** | 1 | 0x01-0x10 |
| 2 | `kFurnitureMode` | Furniture | 1 | **15** | 21 | 0x11-0x1F |
| 3 | `kBonusMode` | Prizes | 1 | **15** | 41 | 0x21-0x2F |
| 4 | `kTransportMode` | Transport | 1 | **12** | 61 | 0x31-0x40 |
| 5 | `kSwitchMode` | Switches | 1 | **9** | 81 | 0x41-0x49 |
| 6 | `kLightMode` | Lights | 1 | **8** | 101 | 0x51-0x58 |
| 7 | `kApplianceMode` | Appliances | 1 | **14** | 121 | 0x61-0x6E |
| 8 | `kEnemyMode` | Enemies | 1 | **9** | 141 | 0x71-0x79 |
| 9 | `kClutterMode` | Clutter | 1 | **15** | 161 | 0x81-0x8F |

`k*Base` and `firstTool` are assigned by `SwitchToolModes`
(`GliderPRO/Sources/Tools.c:370`) into `objectBase` / `firstTool` and then **never read
anywhere in the program** — they are vestigial. Only `lastTool` matters, as the
hit-test clamp. A Go port can drop them; but note that the base values (1, 21, 41, 61,
81, 101, 121, 141, 161) are *decimal* and do **not** equal the hex class bases used by
the real arithmetic (see §5.5).

## 5.3 The tool grid (`GliderPRO/Sources/Tools.c:321`)

```
for v in 0..3:
  for h in 0..3:
    QSetRect(&toolRects[v*4 + h], 2, 29, 30, 57)     // 28x28 at (2,29)
    QOffsetRect(&toolRects[v*4 + h], h*28, v*28)
```

A 4×4 grid of 28×28 cells, top-left cell at (left 2, top 29), so the grid spans
h = 2..114 and v = 29..141 — exactly filling the 116×152 windoid below the popup and
above `toolTextRect`.

`kToolsHigh` 4, `kToolsWide` 4, `kTotalTools` 16. Cell 0 is always the **selection
tool**, drawn with `DrawCIcon(2000, toolRects[0].left, toolRects[0].top)`. Cells 1..15
are 24×24 blits from the sprite sheet, inset 2 px:

```
srcRect  = (0,0,24,24) offset by (i*24, (toolMode-1)*24)
destRect = (0,0,24,24) offset by (toolRects[i+1].left + 2, toolRects[i+1].top + 2)
CopyBits(toolSrcMap, toolsWindow, srcRect, destRect, srcCopy, nil)
```

`toolSrcRect` = `QSetRect(0,0,360,216)` → **360×216**, loaded from `PICT`
`kToolsPictID = 1011`. 360/24 = 15 columns, 216/24 = 9 rows: **one row per `toolMode`,
15 icons per row** (the `Glider PRO.r` dump confirms `PICT 1011` is 360×216 at 8-bit).

The selected cell is framed with `PenSize(2,2)` in `redColor` (`FrameSelectedTool`,
`GliderPRO/Sources/Tools.c:109`) and un-framed by re-framing in `whiteColor`
(`EraseSelectedTool`, `:185`). Note `EraseSelectedTool` never calls `PenNormal()`,
leaving a 2×2 pen for whatever draws next.

A `DkGrayForeColor()` divider is drawn under the popup: `MoveTo(4,25); Line(108,0)`
(`GliderPRO/Sources/Tools.c:264`).

## 5.4 The two icon-index remaps

`toolSelected` runs 1..15 and maps to a *grid cell* that is **not** `toolSelected` for
two modes, because two rows of the sprite sheet are sparse. Forward (tool → icon cell),
used identically in `FrameSelectedTool` (`:114`), `EraseSelectedTool` (`:196`) and
`SelectTool` (`:229`):

```c
toolIcon = toolSelected;
if ((toolMode == kBlowerMode) && (toolIcon >= 7))
    toolIcon--;
else if ((toolMode == kTransportMode) && (toolIcon >= 7)) {
    if (toolIcon >= 15) toolIcon -= 4;
    else                toolIcon = ((toolIcon - 7) / 2) + 7;
}
```

Inverse (icon cell → tool), in `HandleToolsClick` (`GliderPRO/Sources/Tools.c:473`):

```c
if ((toolMode == kBlowerMode)    && (toolIcon >= 7)) toolIcon++;
if ((toolMode == kTransportMode) && (toolIcon >= 7)) {
    if (toolIcon >= 11) toolIcon += 4;
    else                toolIcon = ((toolIcon - 7) * 2) + 7;
}
```

Resolved tables:

**Blower mode** — on the way *out*, a tool ≥ 7 is drawn one cell to the *left*
(`toolIcon--`), so tool 7 and tool 8 both want cell 6/7 respectively and tool 15 lands in
cell 14. On the way *in*, clicking cell *i* ≥ 7 yields tool *i* + 1:

| grid cell | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 |
|---|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|
| tool (click) | 0 | 1 | 2 | 3 | 4 | 5 | 6 | **8** | 9 | 10 | 11 | 12 | 13 | 14 | 15 | 16 |
| `what` | — | 01 | 02 | 03 | 04 | 05 | 06 | **08** | 09 | 0A | 0B | 0C | 0D | 0E | 0F | (10) |

Consequence: **`kRightFan` (tool 7, `what` 0x07) is not reachable from the palette in
blower mode** — the inverse remap jumps from tool 6 straight to tool 8. Cell 15 *is*
live: `lastTool = kLastBlower = 15` (`GliderPRO/Sources/Tools.c:20`) and the hit test is
`(i < kTotalTools) && PtInRect(...) && (i <= lastTool)` (`:468-469`) with `kTotalTools`
16, so cells 0..15 all pass and cell 15 selects tool 16 = `kLiftArea` (0x10). Blower mode
therefore reaches 15 of its 16 types through a 16-cell grid: one cell is spent on the
select arrow, `kRightFan` is skipped, and cell 15 picks up the 16th type.

**Transport mode** — `lastTool` is 12, so cells 0..12 are live:

| grid cell | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 |
|---|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|
| tool (click) | 0 | 1 | 2 | 3 | 4 | 5 | 6 | **7** | **9** | **11** | **13** | **15** | **16** |
| `what` | — | 31 | 32 | 33 | 34 | 35 | 36 | 37 | 39 | 3B | 3D | 3F | 40 |
| object | sel | UpStairs | DownStairs | MailboxLf | MailboxRt | FloorTrans | CeilingTrans | DoorInLf | **DoorExRt** | WindowInLf | **WindowExRt** | InvisTrans | DeluxeTrans |

Worked through: cells 8/9/10 take the `((i-7)*2)+7` branch → tools 9, 11, 13; cells 11 and
12 take the `i >= 11` branch (`toolIcon += 4`) → tools 15 (`kInvisTrans`) and 16
(`kDeluxeTrans`). `lastTool = kLastTransport = 12`, so cells 13-15 are dead in this mode.

Consequence: **the four "left/interior-pair partner" transports — `kDoorInRt` (0x38),
`kDoorExLf` (0x3A), `kWindowInRt` (0x3C), `kWindowExLf` (0x3E) — are unreachable from
the palette.** They are created only by `AddObjectPairing()` (§10), which automatically
adds the partner on the opposite wall. That is the whole point of the remap: the sheet
shows one icon per door/window *pair*.

Note the forward map is not the exact inverse (forward divides by 2 with truncation at
`>= 15`, inverse multiplies at `>= 11`), so `FrameSelectedTool` frames the wrong cell
for the auto-created partner objects — but since the palette never selects those, it
does not show.

## 5.5 The `what` computation

`DoNewObjectClick` (`GliderPRO/Sources/ObjectEdit.c:773`):

```c
whatObject = toolSelected + ((toolMode - 1) * 0x0010);
```

That is `toolSelected + (toolMode-1)*16`. With `toolMode` 1..9 and `toolSelected` 1..16
this produces exactly the nine 16-wide class blocks of §3.5: mode 1 → 0x01-0x10,
mode 2 → 0x11-0x20, mode 3 → 0x21-0x30, …, mode 9 → 0x81-0x90. The gaps (0x20, 0x30,
0x4A-0x50, …) are unreachable because `lastTool` stops short.

The tool *name* uses the same index into `STR# 1007`
(`GliderPRO/Sources/Tools.c:143`):

```c
if (toolSelected == 0) PasStringCopy("\pSelection Tool", theString);
else GetIndString(theString, kObjectNameStrings, toolSelected + ((toolMode-1) * 0x0010));
EraseRect(&toolTextRect);
MoveTo(toolTextRect.left + 3, toolTextRect.bottom - 6);
TextFont(applFont); TextSize(9); TextFace(bold);
ColorText(theString, 171L);
```

`kObjectNameStrings` = `STR# 1007`, 144 strings, index == `what`. Colour 171 is an
index into the 8-bit CLUT (`ColorText` in `GliderPRO/Sources/Utilities.c`).

## 5.6 Palette navigation

| Entry point | Effect |
|---|---|
| `HandleToolsClick` popup part | `TrackControl`; if value changed: `EraseSelectedTool(); SwitchToolModes(newMode)` — which calls `SelectTool(kSelectTool)`, so **changing class always resets to the selection tool** |
| `HandleToolsClick` grid | first `i` in 0..15 with `PtInRect(wherePt,&toolRects[i]) && i <= lastTool` → inverse remap → `SelectTool(toolIcon)` |
| `NextToolMode` (Page Down) | `if theMode==kEditMode && toolMode < kClutterMode`: `toolMode++`, `SetControlValue(classPopUp, toolMode)`, `SwitchToolModes`, `toolSelected = kSelectTool` |
| `PrevToolMode` (Page Up) | mirror, guard `toolMode > kBlowerMode` |
| `SetSpecificToolMode(m)` | same but **no clamping** — caller must pass 1..9 |
| `ToggleToolsWindow` (Cmd-T) | open or close |

`SetSpecificToolMode` is called from the *Tools* menu (`MENU 141` picked directly from
the menu bar) — `GliderPRO/Sources/Menu.c` `DoToolsMenu`.

**Bug**: `ToggleToolsWindow` (`GliderPRO/Sources/Tools.c:351`) sets `isToolsOpen = true`
in *both* branches:

```c
if (toolsWindow == nil) { OpenToolsWindow();  isToolsOpen = true; }
else                    { CloseToolsWindow(); isToolsOpen = true; }   // <-- should be false
```

So once the Tools window has ever been open, the preference sticks and
`OpenCloseEditWindows()` re-opens it on every subsequent entry to edit mode. `Map` and
`Coord` do this correctly (`GliderPRO/Sources/Map.c:438`,
`GliderPRO/Sources/Coordinates.c:181`).

---

# 6. Selection, the marquee, and handles

## 6.1 Hit testing (`GliderPRO/Sources/ObjectEdit.c:46`)

```
FindObjectSelected(where):
 1. if PtInRect(where, &initialGliderRect):    return kInitialGliderSelected   (-2)
 2. if PtInRect(where, &leftStartGliderDest):  return kLeftGliderSelected      (-3)
 3. if PtInRect(where, &rightStartGliderDest): return kRightGliderSelected     (-4)
 4. for i = kMaxRoomObs-1 down to 0:
 5.     if PtInRect(where, &roomObjectRects[i]): return i
 6. return kNoObjectSelected  (-1)
```

The loop runs **downward**, so when rects overlap the *highest slot index* wins. Slot
index is also the draw order (`DrawThisRoomsObjects` iterates 0→23), so "highest index"
== "drawn last" == "topmost". That is why *Bring To Front* / *Send To Back* (§11.3)
manipulate slot order rather than a separate z field.

Empty slots have `roomObjectRects[i]` set to `(0,0,0,0)` by `GetThisRoomsObjRects`, and
`PtInRect` on an empty rect is always false, so they never match.

## 6.2 `DoSelectionClick` (`GliderPRO/Sources/ObjectEdit.c:73`)

```
 1. StopMarquee()
 2. if PtInMarqueeHandle(where) and objActive != kNoObjectSelected:
 3.     if StillDown(): DragHandle(where)                      // resize
 4.     if ObjectHasHandle(&direction,&dist):
 5.         StartMarqueeHandled(&roomObjectRects[objActive], direction, dist)
 6.         HandleBlowerGlider()
 7.     else StartMarquee(&roomObjectRects[objActive])
 8. else:
 9.     objActive = FindObjectSelected(where)
10.     if objActive == kNoObjectSelected:
11.         if isDoubleClick: DoRoomInfo()                      // dbl-click empty space
12.     else if isDoubleClick:
13.         DoObjectInfo()                                      // dbl-click object
14.         re-arm marquee (handled or plain)
15.     else:
16.         if StillDown(): DragObject(where)                   // move
17.         re-arm marquee; for the three glider pseudo-selections
18.            use initialGliderRect / leftStartGliderDest / rightStartGliderDest
19. UpdateMenus(false)
```

Note the handle test comes **first** and uses the *previous* `objActive`, so a click in
the handle of the currently selected object resizes it even if another object's rect
also covers that point.

`StillDown()` distinguishes drag from click: if the button was already released by the
time we get here, no drag happens and we just re-arm the marquee.

## 6.3 The marquee (`GliderPRO/Sources/Marquee.c`)

`kNumMarqueePats` = 7, from `PAT#` `kMarqueePatListID = 128`
(`GliderPRO/Sources/Marquee.c:499`):

```
InitMarquee():
 1. for i = 0..6: GetIndPattern(&theMarquee.pats[i], kMarqueePatListID, i+1)
 2. theMarquee.index = 0
 3. theMarquee.active = false; paused = false; handled = false
 4. gliderMarqueeUp = false
```

`InitMarquee` deliberately leaves `bounds`, `handle`, `direction` and `dist`
**uninitialised** — there is no `direction = -1` anywhere. Nothing reads them before
`StartMarquee`/`StartMarqueeHandled` writes them (both of those are gated on
`handled`/`active`), but a Go port should zero them anyway; `direction = 0` is the value
`MarqueeHasHandles` hands out for "no handle" and is not a legal direction (§6.4).

The seven patterns in `Glider PRO.r` are the classic marching-ants diagonals, each a
1-pixel phase shift of the previous.

`DrawMarquee` (`GliderPRO/Sources/Marquee.c:455`):

```
1. FrameRect(&theMarquee.bounds)
2. if theMarquee.handled:
3.     PaintRect(&theMarquee.handle)
4.     switch direction:                 // connector line, edge directions only
5.       kAbove:  MoveTo(handle.left+4, handle.bottom); LineTo(handle.left+4, bounds.top-1)
6.       kToRight: MoveTo(handle.left, handle.top+4);   LineTo(bounds.right, handle.top+4)
7.       kBelow:  MoveTo(handle.left+4, handle.top-1);  LineTo(handle.left+4, bounds.bottom)
8.       kToLeft: MoveTo(handle.right, handle.top+4);   LineTo(bounds.left, handle.top+4)
9. if gliderMarqueeUp: DrawGliderMarquee()
```

`DrawMarquee` itself does **no** pen or port setup and no `PenNormal()` — every caller
(`DoMarquee`, `StartMarquee`, `StartMarqueeHandled`, `StopMarquee`, the drag loops) is
responsible for `SetPortWindowPort(mainWindow)`, `PenMode(patXor)`,
`PenPat(&theMarquee.pats[theMarquee.index])` before the call and `PenNormal()` after. It
also re-draws the ghost glider when `gliderMarqueeUp`, so the blower preview stays in sync
with the ants. `patXor` means the frame is drawn by inverting; drawing it a second time
with the same pattern erases it exactly. `DoMarquee` (`:35`) is the idle-time animator:

```
1. if !theMarquee.active or theMarquee.paused: return
2. DrawMarquee()                     // erase (XOR)
3. theMarquee.index++; if index >= kNumMarqueePats: index = 0
4. DrawMarquee()                     // draw with next phase
```

so the ants march one phase per idle tick.

| Function | Line | Behaviour |
|---|--:|---|
| `StartMarquee(Rect*)` | `Marquee.c:54` | if active, `StopMarquee()`; **return immediately if `objActive == kNoObjectSelected`**; `bounds = *theRect`; `active = true`; `paused = false`; `handled = false`; draw with the *current* `index`; `SetCoordinateHVD(bounds.left, bounds.top, -1)` |
| `StartMarqueeHandled(Rect*, dir, dist)` | `:76` | same prologue and same `objActive` early return, plus `handled = true`, the `handle` rect (§6.4), then `direction = dir`, `dist = dist`; ends with `SetCoordinateHVD(bounds.left, bounds.top, dist)` |
| `StopMarquee()` | `:139` | if `gliderMarqueeUp`, erase the ghost glider and clear the flag; then return if `!active`; erase; `active = false`; `SetCoordinateHVD(-1, -1, -1)` |
| `PauseMarquee()` | `:161` | return if `!active`; `paused = true`; then `StopMarquee()` (which erases) |
| `ResumeMarquee()` | `:172` | return if `!paused`; if `handled`, `StartMarqueeHandled(&bounds, direction, dist)` + `HandleBlowerGlider()`, else `StartMarquee(&bounds)` |

`PauseMarquee`/`ResumeMarquee` bracket anything that draws into the window (dialogs,
room redraws) so the XOR overlay does not get baked in.

Three field-lifetime details a port must copy exactly:

* **`index` is never reset.** Neither `StartMarquee` nor `StartMarqueeHandled` zeroes it, so
  a new marquee starts at whatever ant phase the previous one ended on. This is harmless
  but visible.
* **`StopMarquee` does not clear `handled`.** Only `StartMarquee` clears it. So after a
  `StopMarquee()` the struct still reports `handled == true`, and `MarqueeHasHandles` /
  `PtInMarqueeHandle` keep answering about a marquee that is no longer on screen.
* **`paused` is cleared only by `StartMarquee`/`StartMarqueeHandled`.** Because both of
  those return early when `objActive == kNoObjectSelected`, a `ResumeMarquee()` issued
  with nothing selected leaves `paused == true` forever, and `DoMarquee` then never
  animates again until something is selected and a fresh `StartMarquee` runs.

## 6.4 Handle geometry

`kHandleSideLong` = 9. `StartMarqueeHandled` builds the handle as a 9×9 square that is
**centred on an anchor point**, not aligned to an edge:

```
QSetRect(&handle, 0, 0, 9, 9);
QOffsetRect(&handle, 9 / -2, 9 / -2);      // C truncation: -4, -4
// then one QOffsetRect to the anchor corner, then one for the direction/dist
```

so the handle always spans `anchor-4 … anchor+5` on both axes. The anchors, with
`HalfRectWide = (right-left)/2` and `HalfRectTall = (bottom-top)/2`
(`GliderPRO/Sources/RectUtils.c:79`, `:87`):

| `direction` | Constant (`GliderDefines.h:210-215`) | Handle centre |
|---|--:|---|
| `kAbove` | 1 | `(bounds.left + HalfRectWide, bounds.top - dist)` |
| `kToRight` | 2 | `(bounds.right + dist, bounds.top + HalfRectTall)` |
| `kBelow` | 3 | `(bounds.left + HalfRectWide, bounds.bottom + dist)` |
| `kToLeft` | 4 | `(bounds.left - dist, bounds.top + HalfRectTall)` |
| `kBottomCorner` | 5 | `(bounds.right, bounds.bottom)` — the SE corner |
| `kTopCorner` | 6 | `(bounds.right, bounds.top)` — the NE corner |

> [verified against `GliderPRO/Headers/GliderDefines.h:210-215`: the six constants are
> `kAbove` 1, `kToRight` 2, `kBelow` 3, `kToLeft` 4, `kBottomCorner` 5, `kTopCorner` 6 —
> they are **1-based**, so 0 is not a valid direction. §6.5 and §20 use these values.]

The handle is *not* drawn black: `DrawMarquee` (`:455`) `PaintRect`s it with the pen
already set to `patXor` + the current marquee pattern, so it flickers through the same
seven patterns as the surrounding frame. For the four edge directions `DrawMarquee` also
draws a 1-pixel connector line from the handle's centre line to the corresponding edge of
`bounds`; the two corner directions get no connector.

`PtInMarqueeHandle(Point)` (`GliderPRO/Sources/Marquee.c:425`) is just
`return (PtInRect(where, &theMarquee.handle));` — it does **not** test
`theMarquee.handled`, so callers must check `MarqueeHasHandles()` (`:407`, which returns
`theMarquee.handled` and also copies out `direction`/`dist` through its two `short*`
parameters) or a stale handle rect from a previous selection will answer `true`.

## 6.5 Which objects have a handle (`GliderPRO/Sources/ObjectEdit.c:1791`)

`ObjectHasHandle(short *direction, short *dist)` returns `false` (no handle) for
`kInitialGliderSelected` and `kNoObjectSelected` — note it does **not** special-case
`kLeftGliderSelected`/`kRightGliderSelected`, which fall through to
`thisRoom->objects[objActive]` with a negative index; in practice `objActive` −3/−4
reads garbage before the array and almost always hits `default: return false`, but this
is an out-of-bounds read a Go port must not reproduce literally.

| `direction` | `dist` source | Object types |
|---|---|---|
| `kAbove` | `data.a.distance` | kFloorVent, kFloorBlower, kSewerGrate, kTaper, kCandle, kStubby, kTiki, kBBQ, kGrecoVent, kSewerBlower |
| `kBelow` | `data.a.distance` | kCeilingVent, kCeilingBlower |
| `kToLeft` | `data.a.distance` | kLeftFan |
| `kToRight` | `data.a.distance` | kRightFan |
| per `vector & 0x0F` | `data.a.distance` | kInvisBlower: 1→`kAbove`, 2→`kToRight`, 4→`kBelow`, 8→`kToLeft` |
| `kToRight` | 0 | kTable, kShelf, kDeckTable |
| `kBottomCorner` | 0 | kLiftArea, kCabinet, kInvisObstacle, kInvisBounce, kMirror, kWallWindow |
| `kTopCorner` | 0 | kCounter, kDresser |
| `kToRight` | `data.c.length` | kGreaseRt |
| `kToLeft` | `data.c.length` | kGreaseLf |
| `kToRight` | 0 | kSlider |
| `kBottomCorner` | 0 | kInvisTrans, kDeluxeTrans |
| `kToRight` | 0 | kFlourescent, kTrackLight |
| `kAbove` | `data.g.height` | kToaster |
| `kAbove` | `data.h.length` | kBall, kFish |
| `kBelow` | `data.h.length` | kDrip |
| *(none)* | | everything else |

Note `kLiftArea` appears here with `kBottomCorner` even though it is a blower-variant
object — it is resized as a rectangle (`distance` = width, `tall`·2 = height).

`HandleBlowerGlider()` (`GliderPRO/Sources/ObjectEdit.c:1963`):

```
1. if ObjectIsUpBlower(&thisRoom->objects[objActive]):
2.     SetMarqueeGliderRect((bounds.right + bounds.left)/2, bounds.top - dist)
```

which draws a ghost glider at the height the blower will lift it to.
`ObjectIsUpBlower` (`:1942`) is true for kFloorVent, kFloorBlower, kSewerGrate, kTaper,
kCandle, kStubby, kTiki, kBBQ, kGrecoVent, kSewerBlower unconditionally, and for
kInvisBlower / kLiftArea when `(data.a.vector & 0x01) == 0x01`.

## 6.6 Drag primitives (`GliderPRO/Sources/Marquee.c`)

All three are XOR "rubber-band" loops that track the mouse until button-up, then apply
the delta once. They never touch the document.

**`DragMarqueeRect(Point start, Rect *theRect, Boolean lockH, Boolean lockV)`**
(`GliderPRO/Sources/Marquee.c:218`) — note the argument order: **`lockH` is the third
parameter, `lockV` the fourth**.

```
 1. SetCursor(&handCursor); StopMarquee()
 2. PenMode(patXor); PenPat(&theMarquee.pats[theMarquee.index])
 3. theMarquee.bounds = *theRect;  FrameRect(&theMarquee.bounds)
 4. wasPt = start
 5. while WaitMouseUp():
 6.     GetMouse(&newPt)
 7.     if DeltaPoint(wasPt, newPt):                 // only on actual movement
 8.         deltaH = lockV ? 0 : newPt.h - wasPt.h   // incremental, not from `start`
 9.         deltaV = lockH ? 0 : newPt.v - wasPt.v
10.         FrameRect(&theMarquee.bounds)            // erase
11.         QOffsetRect(&theMarquee.bounds, deltaH, deltaV)
12.         FrameRect(&theMarquee.bounds)            // draw
13.         wasPt = newPt
14.         SetCoordinateHVD(theMarquee.bounds.left, theMarquee.bounds.top, -2)
15. FrameRect(&theMarquee.bounds)                    // final erase
16. *theRect = theMarquee.bounds
17. PenNormal(); InitCursor()
```

The flag names mean "lock motion **to** this axis", not "freeze this axis":
`lockH = true` zeroes `deltaV` (motion constrained to horizontal), `lockV = true`
zeroes `deltaH` (motion constrained to vertical). With the real parameter order this is
self-consistent with what `DragObject` commits: wall/floor/ceiling-mounted objects such
as `kFloorVent` are dragged with `DragMarqueeRect(where, &newRect, true, false)`
(`GliderPRO/Sources/ObjectEdit.c:411`) → `lockH = true` → the rubber band slides
horizontally only, and the commit switch stores only `deltaH`
(`data.a.topLeft.h += deltaH`). Darts and the two glider start markers use
`(false, true)` → `lockV = true` → vertical only, and only `deltaV` is committed. There
is no preview/commit mismatch here.

Note also that the rubber band reuses the global `theMarquee.bounds` and the current
marquee pattern (`theMarquee.pats[theMarquee.index]`) rather than a private rect and
`qd.gray`, that it calls `StopMarquee()` first (so the marching ants stop while
dragging), and that the deltas are accumulated *incrementally* from the previous mouse
sample. The incremental form matters when a lock flag is set: absolute
`mouse - start` arithmetic would behave differently after the pointer wanders off-axis
and comes back.

**`DragMarqueeHandle(Point start, short *dragged)`** (`:259`) — drags the single handle
along its axis and updates `*dragged` (the distance). Cursor is `vertCursor` for
`kAbove`/`kBelow`, else `horiCursor`. Direction comes from `theMarquee.direction`; the
sign is inverted for `kAbove` and `kToLeft` (`*dragged -= delta`) versus `kBelow` and
`kToRight` (`*dragged += delta`). The clamp is at `*dragged <= 0`: the excess is folded
back into the delta so the handle stops moving exactly at 0, and `*dragged` is set to 0.
Each iteration calls `DeltaCoordinateD(*dragged)`, and the handle is drawn with
`PaintRect` (filled) while the bounds are drawn with `FrameRect`.

**`DragMarqueeCorner(Point start, short *hDragged, short *vDragged, Boolean isTop)`**
(`:345`) — drags the SE corner (or NE when `isTop`, which inverts `deltaV` and moves
`bounds.top` instead of `bounds.bottom`) and updates width/height, each clamped at 0 the
same way.

## 6.7 The coordinate windoid (`GliderPRO/Sources/Coordinates.c`)

A 50×38 windoid (`kWindoidWDEF`) titled from `STR# 150`, showing three numbers.

```
OpenCoordWindow  (:116)  QSetRect(&coordWindowRect, 0, 0, 50, 38)
UpdateCoordWindow (:61)  MoveTo(5,12) "H:" value; MoveTo(4,22) "V:"; MoveTo(5,32) "D:"
SetCoordinateHVD(h, v, d)  (:29)   -2 in any argument means "leave unchanged"
DeltaCoordinateD(d)        (:49)   adds to the D readout
CloseCoordWindow (:172), ToggleCoordinateWindow (:181)
```

`SetCoordinateHVD` is called from the drag loops so the user sees live coordinates; `D`
is the handle distance. The sentinel −2 ("don't change this field") is worth
reproducing literally because callers rely on it.

---

# 7. Moving an object — `DragObject` (`GliderPRO/Sources/ObjectEdit.c:351`)

```
 1. if objActive == kInitialGliderSelected:
 2.     wasRect = newRect = initialGliderRect
 3.     DragMarqueeRect(where, &newRect, false, false)      // free 2-D drag
 4. else if objActive == kLeftGliderSelected:
 5.     wasRect = newRect = leftStartGliderDest
 6.     DragMarqueeRect(where, &newRect, false, true)       // lockV -> vertical only
 7. else if objActive == kRightGliderSelected:  (same as left, on rightStartGliderDest)
 8. else:
 9.     wasRect = newRect = roomObjectRects[objActive]
10.     switch on thisRoom->objects[objActive].what -> pick (lockH, lockV, invalAll)
11.
12. deltaH = newRect.left - wasRect.left
13. deltaV = newRect.top  - wasRect.top
14. if deltaH != 0 or deltaV != 0: fileDirty = true; UpdateMenus(false)
15.
16. if objActive == kInitialGliderSelected:
17.     HGetState/HLock(thisHouse)
18.     (*thisHouse)->initial.h += deltaH; (*thisHouse)->initial.v += deltaV
19.     HSetState(thisHouse)
20. else if objActive == kLeftGliderSelected:
21.     increment = thisRoom->leftStart + deltaV
22.     clamp to [0, 255];  thisRoom->leftStart = (Byte)increment
23.     QSetRect(&leftStartGliderDest, 0,0,48,16)
24.     QOffsetRect(&leftStartGliderDest, 0, kGliderStartsDown + thisRoom->leftStart)
25. else if objActive == kRightGliderSelected:  (same, rightStart / rightStartGliderDest)
26. else:
27.     switch on what -> add deltaH and/or deltaV into the right union fields
28.
29. KeepObjectLegal()                 // result deliberately ignored
30. GetThisRoomsObjRects()
31. if invalAll: InvalWindowRect(mainWindow, &mainWindowRect)
32. else:        InvalWindowRect(mainWindow, &wasRect)
33.              InvalWindowRect(mainWindow, &<the new rect for objActive>)
34. ReadyBackground(thisRoom->background, thisRoom->tiles)
35. DrawThisRoomsObjects()
```

Note step 23-24: the glider start markers are always 48×16 and their vertical position
is `kGliderStartsDown (32) + leftStart`. `leftStart` is therefore an unsigned offset
from y = 32, range 0..255 → y = 32..287.

## 7.1 Rubber-band constraint and repaint scope, per class

| `(lockH, lockV)` | `invalAll` | Types |
|---|---|---|
| `(true, false)` — horizontal rubber band | `false` | kFloorVent, kCeilingVent, kFloorBlower, kCeilingBlower, kSewerGrate, kGrecoVent, kSewerBlower, kManhole, kUpStairs, kDownStairs, kCeilingLight, kHipLamp, kDecoLamp, kFlourescent, kFloorTrans, kCeilingTrans, kDoorInLf, kDoorInRt, kDoorExRt, kDoorExLf, kWindowInLf, kWindowInRt, kWindowExRt, kWindowExLf, kBalloon, kCopterLf, kCopterRt, kMousehole, kFireplace |
| `(false, true)` — vertical rubber band | `false` | kDartLf, kDartRt |
| `(false, false)` — free | `true` | kTiki, kTable, kShelf, kCabinet, kDeckTable, kStool, kInvisObstacle, kInvisBounce, kGreaseRt, kGreaseLf, kSlider, kMailboxLf, kMailboxRt, kInvisTrans, kDeluxeTrans, kMirror, kWallWindow |
| `(true, false)` | `true` | kCounter, kDresser, kTrackLight |
| `(false, false)` | `false` | all 57 remaining types (clocks, prizes, fans, flames, furniture boxes, switches, bulbs, appliances, enemies, clutter) |

`invalAll = true` means the whole 512×322 room is invalidated, because these objects can
be *behind* other things or are large enough that partial invalidation misses.

**Bug**: `invalAll` is a local `Boolean` that is **never initialized**, and the three
glider pseudo-selection branches (steps 1-7) skip the switch entirely, so step 31 tests
an uninitialized value. Likewise any `what` not in the switch (an empty slot, or a
corrupt code) leaves it uninitialized. In a Go port, default it to `true` (safe: full
redraw).

## 7.2 Which field(s) receive the delta

| Fields updated | Types |
|---|---|
| `data.a.topLeft.h` only | kFloorVent, kCeilingVent, kFloorBlower, kCeilingBlower, kSewerGrate, kGrecoVent, kSewerBlower |
| `data.a.topLeft.h` + `.v` | kLeftFan, kRightFan, kTaper, kCandle, kStubby, kTiki, kBBQ, kInvisBlower, kLiftArea |
| `data.b.bounds` all four | kTable, kShelf, kCabinet, kFilingCabinet, kWasteBasket, kMilkCrate, kDeckTable, kStool, kTrunk, kInvisObstacle, kBooks, kInvisBounce |
| `data.b.bounds.left`/`.right` only | kCounter, kDresser, kManhole |
| `data.c.topLeft.h` + `.v` | kRedClock, kBlueClock, kYellowClock, kCuckoo, kPaper, kBattery, kBands, kGreaseRt, kGreaseLf, kFoil, kInvisBonus, kStar, kSparkle, kHelium, kSlider |
| `data.d.topLeft.h` only | kUpStairs, kDownStairs, kFloorTrans, kCeilingTrans, kDoorInLf, kDoorInRt, kDoorExRt, kDoorExLf, kWindowInLf, kWindowInRt, kWindowExRt, kWindowExLf |
| `data.d.topLeft.h` + `.v` | kMailboxLf, kMailboxRt, kInvisTrans, kDeluxeTrans |
| `data.e.topLeft.h` + `.v` | kLightSwitch, kMachineSwitch, kThermostat, kPowerSwitch, kKnifeSwitch, kInvisSwitch, kTrigger, kLgTrigger, kSoundTrigger |
| `data.f.topLeft.h` only | kCeilingLight, kHipLamp, kDecoLamp, kFlourescent, kTrackLight |
| `data.f.topLeft.h` + `.v` | kLightBulb, kTableLamp, kInvisLight |
| `data.g.topLeft.h` + `.v` | kShredder, kToaster, kMacPlus, kGuitar, kTV, kCoffee, kOutlet, kVCR, kStereo, kMicrowave, kCinderBlock, kFlowerBox, kCDs, kCustomPict |
| `data.h.topLeft.h` only | kBalloon, kCopterLf, kCopterRt |
| `data.h.topLeft.v` only | kDartLf, kDartRt |
| `data.h.topLeft.h` + `.v` | kBall, kDrip, kFish, kCobweb |
| `data.i.bounds` all four | kOzma, kMirror, kFlower, kWallWindow, kBear, kCalendar, kVase1, kVase2, kBulletin, kCloud, kFaucet, kRug, kChimes |
| `data.i.bounds.left`/`.right` only | kMousehole, kFireplace |

Types constrained to `h`-only or `.left/.right`-only are exactly those whose vertical
position is pinned by `KeepObjectLegal` (floor/ceiling-mounted). Types constrained to
`v`-only (the two darts) slide along their launch wall.

---

# 8. Resizing an object — `DragHandle` (`GliderPRO/Sources/ObjectEdit.c:143`)

Every branch does: read the current dimension out of the union, call a marquee drag
primitive, write it back, call `KeepObjectLegal()`, and (for types whose size affects
the composited room) invalidate + recomposite. The tail is always
`fileDirty = true; UpdateMenus(false);`.

| Types | Read → drag → write | Redraw? |
|---|---|---|
| kFloorVent, kCeilingVent, kFloorBlower, kCeilingBlower, kSewerGrate, kTaper, kCandle, kStubby, kTiki, kBBQ, kGrecoVent, kSewerBlower | `vDelta = data.a.distance`; `DragMarqueeHandle`; `data.a.distance = vDelta` | no |
| kLiftArea | `hDelta = data.a.distance`, `vDelta = data.a.tall * 2`; `DragMarqueeCorner(...,false)`; `data.a.distance = hDelta`, `data.a.tall = vDelta / 2` | **yes** |
| kLeftFan, kRightFan | `hDelta = data.a.distance` (horizontal) | no |
| kInvisBlower | if `(vector & 0x0F) == 1 or == 4` drag vertically else horizontally, both into `data.a.distance` | no |
| kTable, kShelf, kDeckTable | `hDelta = RectWide(data.b.bounds)`; `bounds.right = bounds.left + hDelta` | yes |
| kCabinet, kInvisObstacle, kInvisBounce | `RectWide`/`RectTall`; `DragMarqueeCorner(...,false)`; `right = left + h`, `bottom = top + v` | yes |
| kCounter, kDresser | `RectWide`/`RectTall`; `DragMarqueeCorner(..., true)` (NE corner); `right = left + h`, **`top = bottom - v`** | yes |
| kGreaseRt, kGreaseLf, kSlider | `hDelta = data.c.length` | yes |
| kInvisTrans | `hDelta = data.d.wide`, `vDelta = data.d.tall`; corner drag; **`if (hDelta > 127) hDelta = 127;`** then `data.d.wide = (Byte)hDelta`, `data.d.tall = vDelta` | yes |
| kDeluxeTrans | `hDelta = ((data.d.tall & 0xFF00) >> 8) * 4`, `vDelta = (data.d.tall & 0x00FF) * 4`; corner drag; `if (hDelta < 64) hDelta = 64; if (vDelta < 32) vDelta = 32;` then `data.d.tall = ((hDelta/4) << 8) + (vDelta/4)` | yes |
| kFlourescent, kTrackLight | `hDelta = data.f.length` | yes |
| kToaster | `vDelta = data.g.height` | no |
| kBall, kDrip, kFish | `vDelta = data.h.length` | no |
| kMirror, kWallWindow | `RectWide`/`RectTall` on `data.i.bounds`; corner drag; `right = left + h`, `bottom = top + v` | yes |

Two encodings deserve emphasis because they are the only *packed* geometry in the file:

**`kInvisTrans`** — `data.d.wide` is a **`Byte` width in pixels** (hence the 127 clamp,
which is not 255 — the author clamped to `char` range) and `data.d.tall` is a **`short`
height in pixels**. Observed across all 22 houses: `(tall, wide)` = `(32, 0)` appears
204 times (the `AddNewObject` default leaves `wide` at 0!), `wide` never exceeds 127
✓, `tall` up to 322 (a full-height room).

**`kDeluxeTrans`** — `data.d.tall` packs **both** dimensions in units of 4 px:
`(width/4) << 8 | (height/4)`, minimum 64×32 px = `0x1008`. Observed: 45 distinct
`(hi, lo)` pairs, e.g. `(81, 56)` → 324×224 px. `data.d.wide` is *not* a dimension for
this type — its high nibble is the initial on/off state (§12.9).

Neither type has a redraw-suppression case: both recomposite the room, because they are
invisible in play but drawn as outlines in the editor.

---

# 9. Adding an object — `AddNewObject`

`GliderPRO/Sources/ObjectAdd.c:48`, signature:

```c
Boolean AddNewObject (Point where, short what, Boolean showItNow)
```

`where` is in **room-local coordinates** (which in edit mode are also main-window local
coordinates, §4.1). It is called from exactly three places:

| Caller | Site | `where` | `what` | `showItNow` |
|---|---|---|---|---|
| `DoNewObjectClick` (palette tool click in the room) | `GliderPRO/Sources/ObjectEdit.c:773` | mouse-down point | `toolSelected + ((toolMode - 1) * 0x0010)` | `true` |
| `AddObjectPairing` (auto-add of the partner object in the *neighbour* room) | `GliderPRO/Sources/ObjectEdit.c:792` | computed partner point | partner `what` | `false` |
| paste-object (dead — `Scrap.c` is commented out, §21.1) | — | — | — | — |

## 9.1 Control flow

```
 1.  objActive = FindEmptyObjectSlot()            // first i in 0..23 with what == kObjectIsEmpty
 2.  if objActive == -1:
 3.      ShoutNoMoreObjects()                     // Alert 1008 "No More Objects"
 4.      return false
 5.  drawWholeRoom = false
 6.  switch (what):   <one arm per class; see §9.3-§9.13>
 7.      each arm may first run a per-type census and, if at the cap:
 8.          ShoutNoMoreSpecialObjects()          // Alert 1028 "Too Many Objects"
 9.          return false                         // NOTE: objActive is left pointing at the
10.                                               //       empty slot, and what is NOT written
11.      each arm sets the union payload and computes newRect (the object's room rect)
12.      default: return false                    // unknown what code -> silent no-op
13.  thisRoom->objects[objActive].what = what
14.  thisRoom->numObjects++
15.  if (KeepObjectLegal()) { }                   // return value deliberately discarded
16.  fileDirty = true
17.  UpdateMenus(false)
18.  handled = ObjectHasHandle(&direction, &dist)
19.  if showItNow:
20.      if drawWholeRoom: ReadyBackground(thisRoom->background, thisRoom->tiles)
21.      GetThisRoomsObjRects()
22.      DrawThisRoomsObjects()
23.      InvalWindowRect(mainWindow, &mainWindowRect)
24.      if handled:
25.          StartMarqueeHandled(&roomObjectRects[objActive], direction, dist)
26.          HandleBlowerGlider()
27.      else:
28.          StartMarquee(&roomObjectRects[objActive])
29.  return true
```

Notes for a porter:

* **`newRect` is computed but never stored.** Every arm builds it, and nothing after the
  switch reads it. The authoritative geometry is always the union payload; `newRect` is
  vestigial (it was presumably once passed to the marquee, which now re-derives the rect
  from `roomObjectRects[objActive]` filled in by `GetThisRoomsObjRects`).
* `objActive` is set (line 1) *before* validation, so a rejected add (lines 8-9, 12)
  leaves `objActive` pointing at an empty slot. The subsequent `DrawThisRoomsObjects`
  from the caller's redraw path skips empty slots, so this is benign but it means
  `objActive` is not a reliable "current selection" after a failed add.
* `drawWholeRoom` is set `true` only by the door/window arm (`ObjectAdd.c:450`) and the
  lights arm (`ObjectAdd.c:586`), because those two change the *background* composite
  (doors/windows cut holes in the wall art; lights change room brightness).
* `KeepObjectLegal()` (`GliderPRO/Sources/HouseLegal.c:42`) runs on every add and can
  silently move the object (odd/even x snapping, floor-object v pinning, minimum
  handle extents). See §9.14.

## 9.2 Per-room object caps

Two ceilings exist: the hard 24-object room limit and per-type "special" caps. All
`HowMany*Objects` counters scan the **24 slots of `thisRoom` only** (not the house), so
these are per-room caps.

| Type(s) counted | Counter | Cap constant | Value | Alert on hit | Source |
|---|---|---|---|---|---|
| any (`what == kObjectIsEmpty` search) | `FindEmptyObjectSlot` | `kMaxRoomObs` | **24** | 1008 | `GliderPRO/Sources/ObjectAdd.c:806` |
| kTaper + kCandle + kStubby (combined) | `HowManyCandleObjects` | `kMaxCandles` | **20** | 1028 | `GliderPRO/Sources/ObjectAdd.c:888` |
| kTiki | `HowManyTikiObjects` | `kMaxTikis` | **8** | 1028 | `GliderPRO/Sources/ObjectAdd.c:904` |
| kBBQ | `HowManyBBQObjects` | `kMaxCoals` | **8** | 1028 | `GliderPRO/Sources/ObjectAdd.c:918` |
| kCuckoo | `HowManyCuckooObjects` | `kMaxPendulums` | **8** | 1028 | `GliderPRO/Sources/ObjectAdd.c:932` |
| kBands | `HowManyBandsObjects` | `kMaxRubberBands` | **2** | 1028 | `GliderPRO/Sources/ObjectAdd.c:946` |
| kGreaseRt + kGreaseLf (combined) | `HowManyGreaseObjects` | `kMaxGrease` | **16** | 1028 | `GliderPRO/Sources/ObjectAdd.c:960` |
| kStar | `HowManyStarsObjects` | `kMaxStars` | **4** | 1028 | `GliderPRO/Sources/ObjectAdd.c:975` |
| kSoundTrigger | `HowManySoundObjects` | `kMaxSoundTriggers` | **1** | 1028 | `GliderPRO/Sources/ObjectAdd.c:989` |
| kUpStairs | `HowManyUpStairsObjects` | `kMaxStairs` | **1** | 1028 | `GliderPRO/Sources/ObjectAdd.c:1003` |
| kDownStairs | `HowManyDownStairsObjects` | `kMaxStairs` | **1** | 1028 | `GliderPRO/Sources/ObjectAdd.c:1017` |
| kShredder | `HowManyShredderObjects` | `kMaxShredded` | **4** | 1028 | `GliderPRO/Sources/ObjectAdd.c:1031` |
| 17 "dynamic" types (below) | `HowManyDynamicObjects` | `kMaxDynamicObs` | **18** | 1028 | `GliderPRO/Sources/ObjectAdd.c:1045` |

Most `kMax*` values are in `GliderPRO/Headers/GliderDefines.h:250-266` (`kMaxRoomObs` 24 at
`:250`, `kMaxCandles` 20 … `kMaxDynamicObs` 18 at `:255-265`), but **`kMaxSoundTriggers` and
`kMaxStairs` are local `#define`s in `GliderPRO/Sources/ObjectAdd.c:17-18`**, both 1.

**The 17 dynamic types** (`GliderPRO/Sources/ObjectAdd.c:1051-1067`), in source order:
`kSparkle`, `kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`,
`kMicrowave`, `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`,
`kDrip`, `kFish`.

**Which arms actually consult `HowManyDynamicObjects`** is not symmetric with that list:

| Arm | Guard |
|---|---|
| prizes arm | only for `what == kSparkle` (`ObjectAdd.c:230`) |
| appliances arm | **`HowManyShredderObjects() >= kMaxShredded`**, and only when `what` is *not* kGuitar/kCinderBlock/kFlowerBox/kCDs/kCustomPict (`ObjectAdd.c:603-605`) |
| enemies arm 1 (kBalloon, kCopterLf, kCopterRt, kDartLf, kDartRt, kCobweb) | `HowManyDynamicObjects() >= kMaxDynamicObs` when `what != kCobweb` (`ObjectAdd.c:652`) |
| enemies arm 2 (kBall, kDrip, kFish) | `HowManyDynamicObjects() >= kMaxDynamicObs` unconditionally (`ObjectAdd.c:698`) |

So **the appliances arm applies the `kShredder` cap of 4 to nine unrelated types**
(kShredder, kToaster, kMacPlus, kTV, kCoffee, kOutlet, kVCR, kStereo, kMicrowave):
adding a fifth *appliance of any of those kinds* is refused once the room already holds
four kShredders — but conversely you may add unlimited toasters, because
`HowManyShredderObjects` counts only `kShredder`. This is almost certainly a bug (the
intent was clearly `HowManyDynamicObjects`), and the dynamic cap of 18 is therefore
**not enforced for the nine appliance dynamic types**. A room can consequently be
authored with more than `kMaxDynamicObs` dynamic objects, which overruns the `dinahs`
array of `kMaxDynamicObs` = 18 entries at play time
(`GliderPRO/Sources/StructuresInit2.c:261`).

Empirical check over all 22 shipped houses (4070 rooms): maximum dynamic-object count
per room observed = **11**, so no shipped house trips the latent overflow.

## 9.3 Blowers arm 1 — floor-mounted and free-floating (`ObjectAdd.c:68-121`)

Types: kFloorVent, kFloorBlower, kSewerGrate, kTaper, kCandle, kStubby, kTiki, kBBQ,
kInvisBlower, kGrecoVent, kSewerBlower, kLiftArea. Variant **`data.a`**.

Horizontal: `topLeft.h = where.h - HalfRectWide(&srcRects[what])` for all twelve.

Vertical, by type:

| Type | `topLeft.v` | Constant value |
|---|---|---|
| kFloorVent | `kFloorVentTop` | **305** |
| kFloorBlower | `kFloorBlowerTop` | **304** |
| kSewerGrate | *(none of the branches match!)* | see below |
| kGrecoVent | `kGrecoVentTop` | **303** (local `#define`, `ObjectAdd.c:22`) |
| kSewerBlower | `kSewerBlowerTop` | **292** (local `#define`, `ObjectAdd.c:23`) |
| kTaper, kCandle, kStubby, kTiki, kBBQ, kInvisBlower, kLiftArea | `where.v - HalfRectTall(&srcRects[what])` | centred on click |

**kSewerGrate falls through every `else if`**, so `data.a.topLeft.v` is left holding
whatever was in the recycled slot's union bytes — uninitialized. `KeepObjectLegal`
rescues it: `GliderPRO/Sources/HouseLegal.c` pins `kSewerGrate` to
`kSewerGrateTop` = **303**. A Go port must reproduce the *final* value (303), not the
intermediate garbage.

Common payload for all twelve:

| Field | Value | Note |
|---|---|---|
| `data.a.distance` | **64** | |
| `data.a.initial` | `true` (1) | |
| `data.a.state` | `true` (1) | |
| `data.a.vector` | **0x01** (up) | see §12.1 for the bit meanings |
| `data.a.tall` | **0x10** for kLiftArea, else **0x00** | kLiftArea packs 64×(0x10*4=64) |

## 9.4 Blowers arm 2 — ceiling-mounted (`ObjectAdd.c:123-138`)

Types: kCeilingVent, kCeilingBlower. Variant `data.a`.

| Field | Value |
|---|---|
| `topLeft.h` | `where.h - HalfRectWide(&srcRects[what])` |
| `topLeft.v` | `kCeilingVentTop` = **8** / `kCeilingBlowerTop` = **5** |
| `distance` | **32** |
| `initial`, `state` | `true`, `true` |
| `vector` | **0x04** (down) |
| `tall` | *not written* (left as slot garbage) |

## 9.5 Blowers arm 3/4 — side fans (`ObjectAdd.c:140-167`)

| Type | `topLeft` | `distance` | `vector` |
|---|---|---|---|
| kLeftFan | centred on `where` both axes | **32** | **0x08** (left) |
| kRightFan | centred on `where` both axes | **32** | **0x02** (right) |

Both set `initial = state = true`; neither writes `tall`.

**Palette reachability:** as established in §5.5, blower mode's tool-icon remap cannot
produce `kRightFan`. The only ways to create a kRightFan are `DuplicateObject` on an
existing one, or authoring the byte directly. The 22 shipped houses nevertheless contain
**54 `kRightFan` objects** (re-counted from the decoded data forks), so they do exist in
shipped content — presumably created before the remap was introduced, or by duplication.
Their `vector` bytes are *not* uniformly `0x02`: 33 are `0x02` (right) and 21 are `0x08`
(left), i.e. `what` and `vector` are independent and a kRightFan can blow left. (The
figure "412 objects with `vector == 0x02`" quoted in §3.2 is the count over *all*
blower-class types, not kRightFan alone.)

## 9.6 Furniture arm (`ObjectAdd.c:169-202`)

Types: kTable, kShelf, kCabinet, kFilingCabinet, kWasteBasket, kMilkCrate, kCounter,
kDresser, kDeckTable, kStool, kTrunk, kInvisObstacle, kBooks, kInvisBounce. Variant
**`data.b`** (a `Rect bounds` + `short pict`).

```
1. newRect = srcRects[what]
2. CenterRectOnPoint(&newRect, where)
3. if what == kCounter: newRect.bottom = kCounterBottom        // 304
4. if what == kDresser: newRect.bottom = kDresserBottom        // 293
5. data.b.bounds = newRect
6. data.b.pict   = 0
```

Note steps 3-4 move `bottom` **without** moving `top`, so a counter/dresser dropped near
the ceiling is *stretched*, not translated, at add time. (Both have a `srcRects` height
of 64, so the stretch is invisible only if the click was already at the right height.)
`KeepObjectLegal` does not re-normalise these two.

Empirically: all 287 kCounter objects in the shipped houses have `bounds.bottom == 304`,
and all 122 kDresser objects have `bounds.bottom == 293` ✓ — nothing ever moves them
vertically afterwards, confirming the editor has no vertical drag for these types
(§7.1's `(lockH, lockV)` table: both types drag with `lockH = true`, i.e. horizontally
only).

`kManhole` is its own arm (`ObjectAdd.c:193`):

```
1. newRect = srcRects[kManhole]                 // 123 x 22
2. CenterRectOnPoint(&newRect, where)
3. newRect.left  = (((newRect.left - 3) / 64) * 64) + 3    // snap to 64-px grid, +3
4. newRect.right = newRect.left + RectWide(&srcRects[kManhole])
5. newRect.bottom = kManholeSits                // 322  (local #define, ObjectAdd.c:21)
6. newRect.top    = newRect.bottom - RectTall(&srcRects[kManhole])   // 322 - 22 = 300
7. data.b.bounds = newRect;  data.b.pict = 0
```

Step 3 uses C integer division, which **truncates toward zero**, so for
`newRect.left - 3` in `-63..-1` the result is `+3`, not `-61`. Verified empirically: all
40 kManhole objects across the shipped houses satisfy `left % 64 == 3`, and
`bounds.top == 300`, `bounds.bottom == 322` in every case.

## 9.7 Prizes arm (`ObjectAdd.c:204-295`)

Variant **`data.c`** for all of them. Positioning is uniform — centred on `where` in both
axes — and only the four scalars differ:

| Sub-arm | Types | `length` | `points` | `state` | `initial` | Cap check |
|---|---|---|---|---|---|---|
| `ObjectAdd.c:204` | kRedClock, kBlueClock, kYellowClock, kCuckoo, kPaper, kBattery, kBands, kFoil, kStar, kSparkle, kHelium | **0** | **0** | true | true | kCuckoo→8, kBands→2, kStar→4, kSparkle→18 |
| `ObjectAdd.c:249` | kGreaseRt, kGreaseLf | **64** | 0 | true | true | kMaxGrease 16 |
| `ObjectAdd.c:269` | kInvisBonus | 0 | **100** | true | true | none |
| `ObjectAdd.c:283` | kSlider | **64** | 0 | true | true | none |

`data.c.length` is the *slide/spill distance* for grease and sliders; it is 0 (unused)
for the ordinary prizes. `data.c.points` is only meaningful for `kInvisBonus`
(100/300/500, §12.7); for real prizes the point value is a hard-coded table in the play
engine, not stored per object. Empirically the shipped houses contain kInvisBonus
`points` values of exactly {100: 136, 300: 27, 500: 164} — no other value ✓, and all
non-kInvisBonus `data.c.points` are 0.

## 9.8 Transport arms (`ObjectAdd.c:297-473`)

All use variant **`data.d`** = `{Point topLeft; short tall; short where; Byte who; Byte wide;}`.
Every arm ends with the same three "no link yet" writes:

```
data.d.where = -1        // no destination room
data.d.who   = 255       // no destination object
data.d.wide  = 0         // (except kDeluxeTrans, which uses 0x10)
```

| Sub-arm | Types | `topLeft.h` | `topLeft.v` | `tall` | `wide` | Cap | `drawWholeRoom` |
|---|---|---|---|---|---|---|---|
| `:297` | kUpStairs, kDownStairs | centred | `kStairsTop` = **28** | 0 | 0 | 1 each | no |
| `:322` | kMailboxLf, kMailboxRt | centred | centred | 0 | 0 | — | no |
| `:338` | kFloorTrans | centred | `kFloorTransTop` = **302** | 0 | 0 | — | no |
| `:352` | kCeilingTrans | centred | `kCeilingTransTop` = **6** | 0 | 0 | — | no |
| `:366` | doors/windows (8 types) | snapped, see below | fixed, see below | 0 | 0 | — | **yes** |
| `:453` | kInvisTrans | `newRect.left` (centred) | `newRect.top` (centred) | `bottom - top` = **32** | **0** | — | no |
| `:464` | kDeluxeTrans | `newRect.left` (centred) | `newRect.top` (centred) | **0x1010** (64×64) | **0x10** ("initially on") | — | no |

Note the kInvisTrans default leaves `wide` = 0, i.e. a **zero-width** invisible
transport, which is why the shipped houses contain 204 kInvisTrans objects with
`(tall, wide) == (32, 0)` — the author had to drag a handle to make them usable.

### 9.8.1 Door/window side snapping (`ObjectAdd.c:366-451`)

The eight door/window codes come in Lf/Rt pairs. The arm **rewrites `what`** based on
which half of the room was clicked, then writes a fixed `topLeft` — these objects are
never freely positioned:

| Clicked pair | `where.h > kRoomWide/2` (i.e. > 256) | resulting `what` | `topLeft.h` | constant | `topLeft.v` | constant |
|---|---|---|---|---|---|---|
| kDoorInLf / kDoorInRt | true | kDoorInRt | `kDoorInRtLeft` | **368** | `kDoorInTop` | **0** |
| kDoorInLf / kDoorInRt | false | kDoorInLf | `kDoorInLfLeft` | **0** | `kDoorInTop` | **0** |
| kDoorExRt / kDoorExLf | true | kDoorExRt | `kDoorExRtLeft` | **496** | `kDoorExTop` | **0** |
| kDoorExRt / kDoorExLf | false | kDoorExLf | `kDoorExLfLeft` | **0** | `kDoorExTop` | **0** |
| kWindowInLf / kWindowInRt | true | kWindowInRt | `kWindowInRtLeft` | **492** | `kWindowInTop` | **64** |
| kWindowInLf / kWindowInRt | false | kWindowInLf | `kWindowInLfLeft` | **0** | `kWindowInTop` | **64** |
| kWindowExRt / kWindowExLf | true | kWindowExRt | `kWindowExRtLeft` | **496** | `kWindowExTop` | **64** |
| kWindowExRt / kWindowExLf | false | kWindowExLf | `kWindowExLfLeft` | **0** | `kWindowExTop` | **64** |

(constants from `GliderPRO/Headers/GliderDefines.h:483-496`)

Both `what` *and* `thisRoom->objects[objActive].what` are assigned inside each branch —
redundantly, since line 13 of §9.1 assigns `what` again — but this matters because the
`srcRects[what]` lookup at `ObjectAdd.c:442` must use the *rewritten* code to get the
right size (a kDoorIn is 144×322 while a kDoorEx is 16×322).

Because the palette can only reach kDoorInLf, kDoorExRt, kWindowInLf and kWindowExRt
(§5.5), the *only* way to author the other four is to click in the appropriate half of
the room — which the snapping code handles. So the palette gap is harmless here: clicking
the "interior door" tool on the right side of the room produces kDoorInRt regardless.

## 9.9 Switches arm (`ObjectAdd.c:475-507`)

Types: kLightSwitch, kMachineSwitch, kThermostat, kPowerSwitch, kKnifeSwitch,
kInvisSwitch, kTrigger, kLgTrigger, kSoundTrigger. Variant **`data.e`** =
`{Point topLeft; short delay; short where; Byte who; Byte type;}`.

| Field | Value |
|---|---|
| `topLeft` | centred on `where`, both axes |
| `delay` | **0** |
| `where` | **3000** for kSoundTrigger, else **−1** |
| `who` | **255** |
| `type` | `kOneShot` = **3** for kTrigger and kLgTrigger, else `kToggle` = **0** |

The kSoundTrigger `where = 3000` is *not* a room link — for that one type the field is
reused as a **`snd ` resource ID** in the house's resource fork (base 3000). This is the
single most confusing field overload in the format: the same 2 bytes are a room-link
value (`MergeFloorSuite` output) for the other eight switch types.

`kToggle` 0, `kForceOn` 1, `kForceOff` 2, `kOneShot` 3 are at
`GliderPRO/Headers/GliderDefines.h:441-444`. Verified: `data.e.type` across all shipped
houses is always in {0, 1, 2, 3} ✓.

## 9.10 Lights arm (`ObjectAdd.c:509-587`)

Types: kCeilingLight, kLightBulb, kTableLamp, kHipLamp, kDecoLamp, kFlourescent,
kTrackLight, kInvisLight. Variant **`data.f`** =
`{Point topLeft; short length; Byte byte0; Byte byte1; Byte initial; Byte state;}`.

| Type | `topLeft.h` | `topLeft.v` | constant | `length` | `newRect` built from |
|---|---|---|---|---|---|
| kCeilingLight | centred | `kCeilingLightTop` | **4** | **64** | `QSetRect(0,0,W,H)` + offset |
| kHipLamp | centred | `kHipLampTop` | **23** | **0** | `QSetRect(0,0,W,H)` + offset |
| kDecoLamp | centred | `kDecoLampTop` | **91** | **0** | `QSetRect(0,0,W,H)` + offset |
| kFlourescent | centred | `kFlourescentTop` | **12** | **64** | **`srcRects[what]` copied directly** then offset |
| kTrackLight | centred | `kTrackLightTop` | **5** | **64** | **`srcRects[what]` copied directly** then offset |
| kLightBulb, kTableLamp, kInvisLight (the `else`) | centred | centred on `where.v` | — | **0** | `QSetRect(0,0,W,H)` + offset |

All eight then get `initial = true`, `state = true`, `byte0 = 0`, `byte1 = 0`, and
`drawWholeRoom = true` (because adding a light changes room brightness — §4.3's
dark-room wash disappears once `GetNumberOfLights(thisRoomNumber) > 0`).

`data.f.length` is a **width in pixels** for kCeilingLight, kFlourescent and kTrackLight
(the light fixture is tiled horizontally to that width) and unused (0) for the others.
Confirmed against `GliderPRO/Sources/HouseLegal.c` (`length = bounds.right - topLeft.h`)
and the drawing code. Observed ranges across the shipped houses: kFlourescent `length`
24..512, kTrackLight `length` 64..512, kCeilingLight `length` 64..512 — always a
multiple of the base 64 except for a handful of hand-tweaked kFlourescents.

The kFlourescent/kTrackLight arms copy `srcRects[what]` *including its offset* into
`newRect` and then offset it again, so `newRect` is doubly displaced. Since `newRect` is
discarded (§9.1), this has no effect.

## 9.11 Appliances arm (`ObjectAdd.c:589-644`)

Types: kShredder, kToaster, kMacPlus, kGuitar, kTV, kCoffee, kOutlet, kVCR, kStereo,
kMicrowave, kCinderBlock, kFlowerBox, kCDs, kCustomPict. Variant **`data.g`** =
`{Point topLeft; short height; Byte byte0; Byte delay; Boolean initial; Boolean state;}`
(`GliderPRO/Headers/GliderStructs.h:64-72`). Note the byte order and widths: **`delay` is a
single `Byte` and it comes *after* `byte0`**, so on disk `data.g` is
`v(2) h(2) height(2) byte0(1) delay(1) initial(1) state(1)`. `delay` can therefore only
hold 0..255.

`topLeft` is centred on `where` in both axes for all fourteen. Then:

| Type | `data.g.height` | `data.g.delay` | `data.g.byte0` |
|---|---|---|---|
| kToaster | **64** | **10 + RandomInt(10)** → 10..19 | 0 |
| kOutlet | 0 | **10 + RandomInt(10)** → 10..19 | 0 |
| kCustomPict | **10000** | 0 | 0 |
| kMicrowave | 0 | 0 | **7** |
| all others | 0 | 0 | 0 |

Plus `initial = true`, `state = true` for all.

Field overloads: `height` is the **toaster's rise height in pixels** for kToaster, the
**PICT resource ID** for kCustomPict (base **10000**, which is why user pictures live at
PICT ≥ 10000 in the house resource fork — verified: every kCustomPict `height` in the
shipped houses resolves to a PICT in its own house's resource fork, 0 missing), and
unused elsewhere. `delay` is in **tenths of a second**. `byte0` on kMicrowave is the
"what it destroys" bit field: **1 = bands, 2 = battery/helium, 4 = foil**, so the default
7 destroys everything (§12.10).

`RandomInt(n)` returns 0..n−1 and consumes the global `theSeed` PRNG, which means adding
a toaster **advances the same random stream the game uses**. A Go port that wants
byte-identical authoring must reproduce `RandomInt` exactly; otherwise it should treat
the initial delay as "any value in 10..19".

## 9.12 Enemies arms (`ObjectAdd.c:646-719`)

Variant **`data.h`** =
`{Point topLeft; short length; Byte delay; Byte byte0; Boolean initial; Boolean state;}`
(`GliderPRO/Headers/GliderStructs.h:74-82`). Both bytes are single `Byte`s and — unlike
`data.g` — **`delay` comes *before* `byte0`** here, so on disk `data.h` is
`v(2) h(2) length(2) delay(1) byte0(1) initial(1) state(1)`. Getting this pair the wrong
way round silently swaps an enemy's delay with its flags.

Arm 1 (`ObjectAdd.c:646`) — kBalloon, kCopterLf, kCopterRt, kDartLf, kDartRt, kCobweb:

| Type | `topLeft.h` | `topLeft.v` |
|---|---|---|
| kDartLf | **`kRoomWide - RectWide(&srcRects[kDartLf])`** = 512 − 64 = **448** | `where.v - HalfRectTall` |
| kDartRt | **0** | `where.v - HalfRectTall` |
| kCobweb | centred on `where.h` | `where.v - HalfRectTall` |
| kBalloon, kCopterLf, kCopterRt | centred on `where.h` | **`(kTileHigh / 2) - HalfRectTall`** = 161 − H/2 |

Note the naming inversion: **kDartLf spawns at the *right* edge** and kDartRt at the
left, because the suffix names the *direction of travel*, not the spawn side.

Then `length = 0`; `delay = 0` for kCobweb else `10 + RandomInt(10)`; `byte0 = 0`;
`initial = state = true`.

Arm 2 (`ObjectAdd.c:695`) — kBall, kDrip, kFish: `topLeft` centred on `where` both axes,
`length = 64`, `delay = 0` for kBall else `10 + RandomInt(10)`, `byte0 = 0`,
`initial = state = true`.

`data.h.length` is the **travel extent** — the vertical bounce height for kBall, the
fall distance for kDrip, the swim range for kFish (§8's `vDelta = data.h.length` handle).

## 9.13 Clutter arms (`ObjectAdd.c:721-765`)

Variant **`data.i`** = `{Rect bounds; short pict;}`.

| Sub-arm | Types | Geometry |
|---|---|---|
| `:721` kMousehole | — | `srcRects` centred, then `bottom = kMouseholeBottom` = **295** (local `#define`, `ObjectAdd.c:19`), `top = bottom - RectTall` = 295 − 11 = **284** |
| `:730` kFireplace | — | `srcRects` centred, then `bottom = kFireplaceBottom` = **297** (local `#define`, `ObjectAdd.c:20`), `top = 297 - 142 = 155` |
| `:739` kFlower | — | see below |
| `:749` | kOzma, kMirror, kWallWindow, kBear, kCalendar, kVase1, kVase2, kBulletin, kCloud, kFaucet, kRug, kChimes | `srcRects[what]` centred on `where`; `pict = 0` |

Unlike §9.6's counter/dresser, the mousehole and fireplace arms move **both** `top` and
`bottom`, so they translate rather than stretch.

**kFlower** (`ObjectAdd.c:739-747`) is the only add path that reads the keyboard:

```
1. GetKeys(theseKeys)
2. if !BitTst(&theseKeys, kShiftKeyMap):        // kShiftKeyMap = 63  (Externs.h:163)
3.     wasFlower = RandomInt(kNumFlowers)       // kNumFlowers = 6 -> 0..5
4. newRect = flowerSrc[wasFlower]
5. CenterRectOnPoint(&newRect, where)
6. data.i.bounds = newRect
7. data.i.pict   = wasFlower
```

i.e. **hold Shift to repeat the previous flower species**, otherwise a random one is
chosen. `wasFlower` is a file-scope global (`GliderPRO/Sources/ObjectAdd.c:42`)
initialised to 0 by BSS, so the first shift-click plants a dandelion.

`flowerSrc[0..5]` (`GliderPRO/Sources/StructuresInit2.c:72-88`) are sub-rects of the
128×69 clutter sheet (PICT 4018 + 1-bit mask PICT 5018), so `data.i.bounds` for a flower
is sized by species, not by `srcRects` — note `srcRects[kFlower]` is **never
initialised at all** (§3.9):

| `pict` | species | W×H | offset in sheet (h, v) |
|---|---|---|---|
| 0 | Dandelion | 10×28 | (0, 23) |
| 1 | Tulip | 24×35 | (10, 16) |
| 2 | Orchid | 34×35 | (34, 16) |
| 3 | Violets | 27×23 | (68, 14) |
| 4 | Daisies | 27×14 | (68, 37) |
| 5 | Sunflower | 32×51 | (95, 0) |

(Species names from DITL 1033, §12.11.) Observed `pict` distribution across the shipped
houses: {0: 84, 1: 83, 2: 56, 3: 81, 4: 117, 5: 126} — consistent with a uniform
`RandomInt(6)` plus authorial shift-repeat.

## 9.14 What `KeepObjectLegal` changes after every add

`GliderPRO/Sources/HouseLegal.c:42`, called at `ObjectAdd.c:774` and again from
`DragObject`/`DragHandle`. It is a per-type normaliser; the parts that matter for adds:

| Rule | Types | Effect |
|---|---|---|
| force **odd** x | kStubby | `if ((topLeft.h % 2) == 0) topLeft.h++` |
| force **even** x | kTaper, kCandle, kTiki, kBBQ | `if ((topLeft.h % 2) != 0) topLeft.h--` |
| pin v and bump distance | kFloorVent → **305**, kFloorBlower → **304**, kSewerGrate → **303**, kFloorTrans → **302** | sets `topLeft.v` to the constant and does `distance += 2` |
| minimum blast height 36 | kFloorVent, kFloorBlower, kTaper, kCandle, kStubby | `if (distance < 36) distance = 36` for the `kAbove` handle |
| clamp fixture width | kFlourescent, kTrackLight | clamps `bounds.right` into the room and recomputes `length = bounds.right - topLeft.h` |

Empirically over all 22 houses: kStubby x is odd in 127/127 cases ✓; kTaper/kCandle/kTiki
x is even in 667/667 cases ✓; kTV x is odd in 80/80 cases (a rule enforced elsewhere).
kFloorTrans `topLeft.v` is **302 in 201 objects but 300 in 43 objects** — the 43 come
from houses saved by an earlier version, and the editor silently rewrites them to 302 the
first time the room is edited. A Go port must not treat 300 as invalid on load.

---

# 10. Automatic partner objects — `AddObjectPairing`

`GliderPRO/Sources/ObjectEdit.c:792`. After a successful `AddNewObject` in the *current*
room, `DoNewObjectClick` calls this to insert the matching object in the **neighbouring
room**, so that doors, windows and stairs always come in pairs. It is the only editor
code that writes into a room other than `thisRoom`.

## 10.1 Control flow

It is a flat chain of ten `if / else if` tests on
`thisRoom->objects[objActive].what` (not a `switch`), and every arm has the identical
body:

```
 1.  roomNum = DoesNeighborRoomExist(dir)     // dir = kRoomAbove/kRoomBelow/kRoomToRight/kRoomToLeft
 2.  if roomNum == -1: nothing happens        // no such room (or not kEditMode) -> silent
 3.  emptySlot = FindObjectSlotInRoom(roomNum)
 4.  if emptySlot == -1 or DoesRoomNumHaveObject(roomNum, otherWhat): nothing happens
 5.  HGetState/HLock(thisHouse); testRoomPtr = &(*thisHouse)->rooms[roomNum]
 6.  testRoomPtr->objects[emptySlot].what          = otherWhat
 7.  testRoomPtr->objects[emptySlot].data.d.topLeft = otherPt
 8.  ...data.d.tall = 0; .where = -1; .who = 255; .wide = 0
 9.  testRoomPtr->numObjects++
10.  HSetState(thisHouse)
11.  GetLocalizedString(45, message); OpenMessageWindow(message)
12.  ForeColor(blueColor)
13.  GetLocalizedString(strIndex, message); SetMessageWindowMessage(message)
14.  ForeColor(blackColor); DelayTicks(60); CloseMessageWindow()
```

Three consequences worth carrying into a port:

* **The neighbour lookup is `DoesNeighborRoomExist` (`GliderPRO/Sources/Room.c:505`), not
  `GetNeighborRoomNumber`.** It returns `-1` outright when `theMode != kEditMode`, and it
  takes the four `kRoomAbove`/`kRoomBelow`/`kRoomToRight`/`kRoomToLeft` codes.
* **`fileDirty` is never set here.** The neighbouring room really is modified in memory,
  but the dirty flag is set by `AddNewObject`'s own tail (`GliderPRO/Sources/ObjectAdd.c:777`)
  for the object the user actually placed; the pairing rides along on that. If a port ever
  calls pairing without the preceding add, it must set `fileDirty` itself.
* **The message is modal for one second.** Step 11 opens the message windoid with string
  45 ("Object Pair Added"), step 13 replaces the text with the specific message drawn in
  **blue**, then `DelayTicks(60)` blocks and the windoid is closed. It is not a transient
  status line, and string 45 *is* used — by every one of the ten arms.

The write at steps 6-9 goes through the locked `thisHouse` handle
(`HGetState`/`HLock`/`HSetState`), *not* through `thisRoom`, so it is immediately part of
the in-memory house and needs no `CopyThisRoomToRoom`.

## 10.2 The ten pairings

**Interior objects pair with *exterior* partners and vice versa** — the pairing crosses the
interior/exterior distinction rather than mirroring within it. (That is the whole point: an
interior door on your right wall means the room to your right is the outside of the
building, so it gets an *exterior* door on its left.) Order below follows the source
(`GliderPRO/Sources/ObjectEdit.c`).

| Trigger `what` | Line | Neighbour | Partner `what` | Partner position | STR# 150 index | Message |
|---|--:|---|---|---|--:|---|
| kDoorInRt | `:799` | right (`kRoomToRight`) | **kDoorExLf** | `(kDoorExLfLeft, kDoorExTop)` = (0, 0) | 46 | "Exterior Door Added in Next Room" |
| kDoorInLf | `:833` | left (`kRoomToLeft`) | **kDoorExRt** | `(kDoorExRtLeft, kDoorExTop)` = (496, 0) | 46 | "Exterior Door Added in Next Room" |
| kDoorExRt | `:868` | right | **kDoorInLf** | `(kDoorInLfLeft, kDoorInTop)` = (0, 0) | 47 | "Interior Door Added in Next Room" |
| kDoorExLf | `:903` | left | **kDoorInRt** | `(kDoorInRtLeft, kDoorInTop)` = (368, 0) | 47 | "Interior Door Added in Next Room" |
| kWindowInLf | `:938` | left | **kWindowExRt** | `(kWindowExRtLeft, kWindowExTop)` = (496, 64) | 48 | "Ext. Window Added in Next Room" |
| kWindowInRt | `:973` | right | **kWindowExLf** | `(kWindowExLfLeft, kWindowExTop)` = (0, 64) | 48 | "Ext. Window Added in Next Room" |
| kWindowExRt | `:1008` | right | **kWindowInLf** | `(kWindowInLfLeft, kWindowInTop)` = (0, 64) | 49 | "Int. Window Added in Next Room" |
| kWindowExLf | `:1043` | left | **kWindowInRt** | `(kWindowInRtLeft, kWindowInTop)` = (492, 64) | 49 | "Int. Window Added in Next Room" |
| kUpStairs | `:1078` | above (`kRoomAbove`) | kDownStairs | same `topLeft.h`, `v = kStairsTop` = 28 | 50 | "Down Stairs Added in Room Above" |
| kDownStairs | `:1114` | below (`kRoomBelow`) | kUpStairs | same `topLeft.h`, `v = kStairsTop` = 28 | 51 | "Up Stairs Added in Room Below" |

Note the direction asymmetry that follows from this: a right-wall trigger
(`kDoorInRt`, `kDoorExRt`, `kWindowInRt`, `kWindowExRt`) always looks to the room on the
**right** and drops a left-edge partner there (`left = 0`), and a left-wall trigger always
looks **left** and drops a right-edge partner (`left` = 368/496/492).

All ten partners are `data.d` objects and get the standard `tall = 0`, `where = -1`,
`who = 255`, `wide = 0` (no link) — the *pairing* is positional, not a link. The play
engine matches a door on the right edge of room N with a door on the left edge of room
N+1 purely by geometry.

STR# 150 indices 45-51 (verified by parsing the resource, §0.3) are:

| Index | Text |
|---|---|
| 45 | `Object Pair Added` |
| 46 | `Exterior Door Added in Next Room` |
| 47 | `Interior Door Added in Next Room` |
| 48 | `Ext. Window Added in Next Room` |
| 49 | `Int. Window Added in Next Room` |
| 50 | `Down Stairs Added in Room Above` |
| 51 | `Up Stairs Added in Room Below` |

Index 45 ("Object Pair Added") is **used by all ten arms** as the initial text of the
message windoid (`GetLocalizedString(45, …); OpenMessageWindow(…)`); the specific message
46-51 then replaces it in blue for 60 ticks.

## 10.3 The two coordinate/`what` confusions

`GliderPRO/Sources/ObjectEdit.c:874` and `:909` pass **`kDoorInLfLeft`** and
**`kDoorInRtLeft`** — which are *x coordinates* 0 and 368 — into positions where an
object `what` code is expected. `kDoorInLfLeft` = 0 = `kObjectIsEmpty`-adjacent
(actually `what` 0 is unassigned; `kFloorVent` is 1), and `kDoorInRtLeft` = 368 = 0x170,
far outside the 0..0x8F `what` range. In practice these feed
`DoesRoomNumHaveObject(neighbor, 0)` / `(neighbor, 368)`, which can never match a real
object, so the duplicate-suppression check at step 9 is defeated for those two arms and
a duplicate partner *can* be created. A Go port should use the correct `what` codes
(`kDoorInRt` / `kDoorInLf`) — the intent is unambiguous from the surrounding arms.

## 10.4 Two different neighbour lookups

There are **two** neighbour helpers with different direction vocabularies, and mixing them
up is easy:

**`DoesNeighborRoomExist(short whichNeighbor)`** (`GliderPRO/Sources/Room.c:505`) — the one
`AddObjectPairing` and `SelectNeighborRoom` use. Takes the four
`kRoomAbove` (1) / `kRoomBelow` (2) / `kRoomToRight` (3) / `kRoomToLeft` (4) codes.

```
if theMode != kEditMode: return -1        // hard gate, editor-only
newH = thisRoom->suite;  newV = thisRoom->floor
kRoomAbove:  newV++     kRoomBelow: newV--
kRoomToRight:newH++     kRoomToLeft:newH--
return RoomExists(newH, newV, &n) ? n : -1        // note: (suite, floor) order
```

Any other direction code falls through the `switch` and returns the *current* room.

**`GetNeighborRoomNumber(short which)`** (`GliderPRO/Sources/Room.c:562`) — used by the map
window and by the eight-way navigation, and it takes the **nine compass codes**
(`kCentralRoom` 0, `kNorthRoom` 1, `kNorthEastRoom`, `kEastRoom`, `kSouthEastRoom`,
`kSouthRoom`, `kSouthWestRoom`, `kWestRoom`, `kNorthWestRoom`), not the `kRoom*` codes:

| `which` | suite delta (`hDelta`) | floor delta (`vDelta`) |
|---|--:|--:|
| `kCentralRoom` | 0 | 0 |
| `kNorthRoom` | 0 | +1 |
| `kNorthEastRoom` | +1 | +1 |
| `kEastRoom` | +1 | 0 |
| `kSouthEastRoom` | +1 | −1 |
| `kSouthRoom` | 0 | −1 |
| `kSouthWestRoom` | −1 | −1 |
| `kWestRoom` | −1 | 0 |
| `kNorthWestRoom` | −1 | +1 |

It reads the base coordinates from `(*thisHouse)->rooms[thisRoomNumber]` (not `thisRoom`),
linear-scans `i = 0 .. numberRooms-1` — the *global* room count, not the locally reloaded
`nRooms` — and returns the first index whose `(suite, floor)` matches, else
`kRoomIsEmpty` (−1). `hDelta`/`vDelta` are uninitialised if `which` is out of range.

Neither helper has a spatial index — both are O(nRooms) scans performed on every pairing
and every room navigation. With `nRooms` up to 531 in the largest shipped house this is
still trivial, but a Go port will naturally want a `map[[2]int16]int` — just remember that
duplicates are *possible* in a corrupt house (§19.3 validates against them) and the C
returns the **first** match.

---

# 11. Duplicate, delete, and depth ordering

## 11.1 `DuplicateObject` (`GliderPRO/Sources/ObjectEdit.c:1191`)

Bound to House ▸ Duplicate (⌘=), `MenuBar.c`'s `iDuplicate`.

```
 1.  if objActive == kNoObjectSelected: return
 2.  newSlot = FindEmptyObjectSlot()
 3.  if newSlot == -1: ShoutNoMoreObjects(); return
 4.  rect = roomObjectRects[objActive]
 5.  placePt.h = rect.left + HalfRectWide(&rect) + 64        // 64 px to the right
 6.  placePt.v = rect.top  + HalfRectTall(&rect)
 7.  thisRoom->objects[newSlot] = thisRoom->objects[objActive]   // full 12-byte copy
 8.  switch on what:  reposition the copy
 9.      point-based variants (a,c,d,e,f,g,h): topLeft.h += 64
10.      rect-based variants  (b,i):            QOffsetRect(&bounds, 64, 0)
11.  thisRoom->numObjects++
12.  objActive = newSlot
13.  KeepObjectLegal()
14.  fileDirty = true
15.  GetThisRoomsObjRects(); DrawThisRoomsObjects(); InvalWindowRect(...)
16.  handled = ObjectHasHandle(&direction, &dist)
17.  if handled: StartMarqueeHandled(...); HandleBlowerGlider()
18.  else:       StartMarquee(...)
```

Key points for a port:

* The duplicate is offset **+64 px horizontally only** — one tile width. It is not
  clamped into the room here; `KeepObjectLegal` does whatever clamping applies to the
  type, which for most types is nothing horizontal. So duplicating an object near the
  right edge repeatedly walks it out of the room, and `topLeft.h` can exceed 512.
* No per-type cap check! `DuplicateObject` does **not** call `HowMany*Objects`, so it is
  the documented way to exceed `kMaxStars`, `kMaxStairs`, `kMaxSoundTriggers`, etc. This
  is how a room can end up with two kUpStairs or five kStars.
* The link fields are copied verbatim, so a duplicated switch/transport **shares the
  original's destination** (`where`/`who`). The destination object's `retroLinkList`
  entry, however, still names only one source (§17.5), so the "Linked From?" reverse
  lookup will find only one of the two.
* Duplicating a `kDeluxeTrans` copies its packed size (`data.d.tall`) exactly.

## 11.2 `DeleteObject` (`GliderPRO/Sources/ObjectEdit.c:1154`)

Bound to House ▸ Clear (no key equivalent) and to the Delete/Backspace key in edit mode.

```
1.  if objActive == kNoObjectSelected: return
2.  for i in 0..kMaxRoomObs-1:
3.      if retroLinkList[i].room   == thisRoomNumber and
4.         retroLinkList[i].object == objActive:
5.              retroLinkList[i].room = -1                 // drop reverse links to me
6.  StopMarquee()
7.  thisRoom->objects[objActive].what = kObjectIsEmpty     // -1
8.  thisRoom->numObjects--
9.  QSetRect(&roomObjectRects[objActive], -1, -1, 0, 0)    // "nowhere" rect
10. objActive = kNoObjectSelected
11. fileDirty = true
12. UpdateMenus(false)
13. GetThisRoomsObjRects(); DrawThisRoomsObjects(); InvalWindowRect(...)
```

What it **does not** do:

* It does not clear *forward* links pointing **at** the deleted object from other rooms.
  A switch in room 7 linked to object 3 of room 12 keeps `where`/`who` after object 3 of
  room 12 is deleted. This is precisely how the shipped houses accumulated **195 dangling
  links** (§3.13). Play-time code tolerates them by ignoring links whose target `what` is
  `kObjectIsEmpty`.
* It does not compact the slot array. The slot becomes a hole with `what == -1`; slot
  indices of all other objects are preserved, which is essential because links address
  objects **by slot index**. Compaction happens only in `SortRoomsObjects`
  (`GliderPRO/Sources/House.c:393`), which fixes links as it goes.
* `numObjects` is decremented even though the array is not compacted, so `numObjects` is
  a *count of live objects*, not a high-water mark. `MakeSureNumObjectsJives`
  (`GliderPRO/Sources/HouseLegal.c:885`) recomputes it at validation time. Verified: 0
  mismatches across 4070 shipped rooms ✓.
* The `kObjectIsEmpty` sentinel is **−1** (`GliderPRO/Headers/GliderDefines.h:526`),
  stored big-endian as `0xFFFF` in the `what` field.

## 11.3 `BringSendFrontBack` (`GliderPRO/Sources/Objects.c:876`)

Bound to House ▸ Bring Forward / Send Backward. Draw order is slot order, so "front" is
the *last* slot — but the implementation rotates the whole 24-slot array rather than
swapping neighbours:

```
 1.  if bringFront and objActive == kMaxRoomObs - 1: return    // already frontmost
 2.  if !bringFront and objActive == 0: return                 // already backmost
 3.  CopyThisRoomToRoom()
 4.  numLinks = CountHouseLinks()
 5.  if numLinks != 0:                         // no allocation at all when 0
 6.      linksList = (linksPtr)NewPtr(sizeof(linksType) * numLinks)   // global, a Ptr
 7.      if linksList == nil: YellowAlert(kYellowCantOrderLinks, MemError()); return
 8.      GenerateLinksList()
 9.  HGetState/HLock(thisHouse)
10.  build sorting[0..23]: the identity permutation, then shift the objects in
11.      thisHouse->rooms[thisRoomNumber] and sorting[] together so objActive ends up in
12.      slot 23 (bringFront) or slot 0 (!bringFront); SpinCursor(2) each iteration
13.  build the inverse:  sorted[sorting[i]] = i
14.  for each link i in linksList with destRoom == thisRoomNumber:
15.      srcObj = (srcRoom == thisRoomNumber) ? sorted[srcObj] : srcObj   // local links moved too
16.      rooms[srcRoom].objects[srcObj].<who> = sorted[destObj]
17.  HSetState(thisHouse); if linksList != nil: DisposePtr(linksList)
18.  ForceThisRoom(thisRoomNumber)
19.  fileDirty = true;  UpdateMenus(false)
20.  InvalWindowRect(mainWindow, &mainWindowRect)
21.  DeselectObject()
22.  GetThisRoomsObjRects(); ReadyBackground(...); DrawThisRoomsObjects()
23.  GenerateRetroLinks()
24.  InitCursor()
```

Step 15 matters: a link *inside* the reordered room has had its own source slot moved as
well, so the source index has to be pushed through `sorted` before it is used to address
the object being patched.

Step 16 uses a `switch` on the source object's `what` whose two arms are **swapped**
relative to the union variants: the eight switch types write `data.d.who` (`:971`) and the
`default:` arm — reached by the transport types — writes `data.e.who` (`:976`). This is
harmless in C only because `who` sits at the same offset (payload byte 8) in both
`switchType` and `transportType`. A Go port with distinct structs per variant must write
the *correct* variant's `who`, not transcribe the swap.

The link renumbering (steps 4-13) is the reason this is a heavyweight operation: because
links address objects by slot index, **any reordering must rewrite every inbound link in
the entire house**. `linksType` is 8 bytes (`GliderPRO/Headers/GliderStructs.h:281`):

| Offset | Field | Type | Meaning |
|---|---|---|---|
| 0 | `srcRoom` | short | room index of the linking object |
| 2 | `srcObj` | short | slot index of the linking object |
| 4 | `destRoom` | short | room index of the target |
| 6 | `destObj` | short | slot index of the target |

Note step 17: the selection is **dropped** rather than followed to its new slot, so the
user must re-click the object to reorder it again.

Two quirks:

* `DeselectObject()` at step 17 happens *after* `ForceThisRoom()`, so the marquee is
  stopped correctly, but `objActive` is invalidated before `GenerateRetroLinks()`, which
  is fine since that function does not read it.
* The switch at `GliderPRO/Sources/Objects.c:971-978` that decides whether to write
  `data.d.who` or `data.e.who` has the switch/default arms transposed relative to the
  rest of the file. Because `data.d.who` and `data.e.who` occupy the **same byte offset
  (10)** in the union (§3.2), the transposition is *type-notional only* and produces
  correct bytes. Do not "fix" it in a way that changes the offset.

---

# 12. Object info dialogs

`GliderPRO/Sources/ObjectInfo.c` (2567 lines) implements thirteen modal dialogs, one per
object family. Every object type in the game is routed to exactly one of them by
`DoObjectInfo` (§12.15).

## 12.1 Entry points

| Trigger | Site | Guard |
|---|---|---|
| Double-click an object in the room | `DoSelectionClick`, `GliderPRO/Sources/ObjectEdit.c:104` | selection tool active, an object was hit |
| House ▸ Object Info (⌘I) | `GliderPRO/Sources/Menu.c:487` | `if (houseUnlocked)` |
| Double-click **empty** room space | `GliderPRO/Sources/ObjectEdit.c:98` | opens **Room** Info instead (§13) |
| Auto-open on new room | `HandleIdleTask`, `GliderPRO/Sources/Events.c:469` | `autoRoomEdit && newRoomNow` → Room Info |

Both object-info entry points re-start the marquee afterwards, because the dialog
disposal repaints over it:

```
DoObjectInfo();
if (ObjectHasHandle(&direction, &dist)) { StartMarqueeHandled(...); HandleBlowerGlider(); }
else                                      StartMarquee(&roomObjectRects[objActive]);
```

## 12.2 The common shape

Twelve of the thirteen follow an identical skeleton. Written once here; the per-dialog
sections below list only the deltas.

```
 1. xFilterUPP = NewModalFilterUPP(XFilter)
 2. NumToString(objActive + 1, numberStr)                  // 1-based for display
 3. GetIndString(kindStr, kObjectNameStrings /*1007*/, thisRoom->objects[objActive].what)
 4. ParamText(numberStr, kindStr, <^2>, <^3>)              // fills the ^0..^3 in the DITL
 5. BringUpDialog(&infoDial, kXInfoDialogID)
 6.    == GetNewDialog(id, nil, kPutInFront); SetPort; ShowWindow; DrawDefaultButton
 7. <seed control values from the object's union payload>
 8. if (retroLinkList[objActive].room == -1) HideDialogItem(infoDial, <linkedFromItem>)
 9. leaving = false; (doLink = doGoTo = doReturn = false)
10. while (!leaving):
11.     ModalDialog(xFilterUPP, &item)
12.     dispatch on item; OK/Cancel/other buttons set leaving
13. DisposeDialog(infoDial); DisposeModalFilterUPP(xFilterUPP)
14. if (doLink)   { linkType = ...; linkerIsSwitch = ...; OpenLinkWindow();
15.                 linkRoom = thisRoomNumber; linkObject = objActive; DeselectObject(); }
16. else if (doGoTo)  GoToObjectInRoom(who, floor, suite)
17. else if (doReturn) GoToObjectInRoomNum(retroLinkList[objActive].object,
18.                                        retroLinkList[objActive].room)
```

Key structural points a Go port must reproduce:

* **Deferred navigation.** `doLink` / `doGoTo` / `doReturn` are set inside the modal loop
  but acted on only *after* `DisposeDialog`. This is deliberate: `OpenLinkWindow` and
  `GoToObjectInRoom*` change the current room and would corrupt a live modal dialog.
* **"Linked From?" is hidden, not disabled**, when nothing links to this object
  (`retroLinkList[objActive].room == -1`). The DITL always contains the button.
* `kOkayButton` = **1**, `kCancelButton` = **2** (`GliderPRO/Headers/Externs.h:22-23`) —
  by convention DITL item 1 is Okay and item 2 is Cancel in every one of these dialogs.
* Every `Update*Info` routine ends with `DrawDialog` + `DrawDefaultButton` +
  `FrameDialogItemC(dial, <divider item>, kRedOrangeColor8)`. `kRedOrangeColor8` = **23**
  (`GliderPRO/Headers/GliderDefines.h:542`, with the comment "actually, 18"), resolved
  through `Index2Color` against the current 8-bit CLUT. A Go port must hard-code the RGB
  it wants; index 23 of Glider's clut is a red-orange.
* Filter key handling (all thirteen `XFilter` functions): `kReturnKeyASCII` **0x0D** and
  `kEnterKeyASCII` **0x03** → flash and return `kOkayButton`; `kEscapeKeyASCII` **0x1B**
  → flash and return `kCancelButton` (only in the filters that have a Cancel);
  `kTabKeyASCII` **0x09** → `SelectDialogItemText` on the dialog's edit field (or a no-op
  `return true` where the field is commented out); `updateEvt` → `BeginUpdate` /
  `Update*Info` / `EndUpdate` then rewrite `event->what = nullEvent` and return `false`.
  (`GliderPRO/Headers/Externs.h:30-40`)

## 12.3 Blower info — DLOG/DITL **1007**, `DoBlowerObjectInfo(short what)`

`GliderPRO/Sources/ObjectInfo.c:934`. Dialog size **167×256** (rect t/l/b/r as stored;
see §0.3 for how the resource was parsed).

| DITL item | Type | Content / role |
|---|---|---|
| 1 | btnCtrl | `Okay` (default) |
| 2 | btnCtrl | `Cancel` |
| 3 | statText | `Object Number: ^0` |
| 4 | statText | `Object Kind: ^1` |
| 5 | userItem | divider box, framed `kRedOrangeColor8` |
| 6 | chkCtrl | `Initially On` (`kInitialStateCheckbox`) |
| 7 | chkCtrl | `Extra Forceful` (`kForceCheckbox`) at **t100 l400 b118 r512 — outside the 256-px-wide dialog, therefore invisible and unclickable** |
| 8 | userItem | direction-arrow canvas |
| 9 | statText | `Direction:` (`kDirectionText`) |
| 10 | picItem | blower illustration |
| 11 | userItem | up hot-spot, t69 l218 b?? |
| 12 | userItem | right hot-spot, t90 l239 |
| 13 | userItem | down hot-spot, t111 l218 |
| 14 | userItem | left hot-spot, t90 l197 |
| 15 | btnCtrl | `Linked From?` |
| 16 | radCtrl | `Left Facing` (`kLeftFacingRadio`) |
| 17 | radCtrl | `Right Facing` (`kRightFacingRadio`) |

Seeding (`ObjectInfo.c:944-990`):

```
1. ParamText(numberStr, kindStr, distStr, "\p")     // ^2 = data.a.distance as decimal
2. newDirection = data.a.vector & 0x0F              // low nibble only!
3. SetDialogItemValue(item 6, data.a.initial ? 1 : 0)
4. if what in {kTaper, kCandle, kStubby, kTiki, kBBQ}: HideDialogItem(item 6)
5. if what in {kLeftFan, kRightFan}:
6.     SelectFromRadioGroup(16 or 17, 16, 17);  leftFacing = (what == kLeftFan)
7.     HideDialogItem(item 9)                   // "Direction:"
8. else:
9.     HideDialogItem(16); HideDialogItem(17)
10. if retroLinkList[objActive].room == -1: HideDialogItem(item 15)
11. ShowWindow(...)
```

Item dispatch:

| Item | Action |
|---|---|
| 1 Okay | write `initial` from checkbox; **`data.a.vector = (Byte)newDirection`**; if fan, set `what` = kLeftFan/kRightFan, `KeepObjectLegal()`, full room redraw; `fileDirty = true`; leave |
| 2 Cancel | leave, no writes |
| 6 | `ToggleDialogItemValue(6)` |
| 15 | same writes as Okay, then `doReturn = true` |
| 16 | `leftFacing = true`, reselect radio |
| 17 | `leftFacing = false`, reselect radio |
| 11/12/13/14 | **only if `what` is kInvisBlower or kLiftArea**: `newDirection` = 0x01 / 0x02 / 0x04 / 0x08, then `UpdateBlowerInfo(infoDial)` |

### 12.3.1 The `vector` field and the lost "forceful" bit

`data.a.vector` is a bitfield (`GliderPRO/Headers/GliderStructs.h:16-17` documents it):

| Bit | Value | Meaning |
|---|---|---|
| 0 | 0x01 | blows **up** |
| 1 | 0x02 | blows **right** |
| 2 | 0x04 | blows **down** |
| 3 | 0x08 | blows **left** |
| 4 | 0x10 | **"F." — extra forceful** |

Because line 2 of the seeding masks with `0x0F` and the Okay handler writes
`vector = (Byte)newDirection` unconditionally, **opening this dialog and clicking Okay
clears the 0x10 forceful bit**. The `Extra Forceful` checkbox that was supposed to
preserve it is positioned off the edge of the dialog (item 7 above) and is never read.
Observed `vector` values across all 22 shipped houses:

| `vector` | count | interpretation |
|---|---|---|
| 0x01 | 4900 | up |
| 0x02 | 412 | right |
| 0x04 | 326 | down |
| 0x08 | 402 | left |
| 0x11 | 3 | up + forceful |
| 0x14 | 1 | down + forceful |

So only **4 objects in all of the shipped content** carry the forceful bit, all of which
would be silently downgraded by one visit to their info dialog. A Go port should keep the
high nibble (`vector = (vector & 0xF0) | newDirection`) and expose the checkbox, and must
still *load* 0x11/0x14 correctly.

Note also that only kInvisBlower and kLiftArea can have their direction changed at all
(the four hot-spots are inert for every other type), and that the direction is not
reflected in the `what` code — a kFloorVent with `vector = 0x04` blows downward through
the floor.

### 12.3.2 The arrow drawing (`UpdateBlowerInfo`, `ObjectInfo.c:120`)

For non-fan blowers it erases item 8's rect (grown by 2 px right and bottom), sets
`PenSize(2,2)`, and draws a shaft plus a 4-px (`kArrowheadLength`) arrowhead:

| `newDirection` | shaft | arrowhead |
|---|---|---|
| 1 (up) | from top-centre, `Line(0, RectTall)` | two 4-px diagonals from the top-centre |
| 2 (right) | from right-middle, `Line(-RectWide, 0)` | two 4-px diagonals from the right-middle |
| 4 (down) | from top-centre, `Line(0, RectTall)` | two 4-px diagonals from the bottom-centre |
| 8 (left) | from left-middle, `Line(RectWide, 0)` | two 4-px diagonals from the left-middle |

Note case 1 and case 4 draw the *identical shaft* (both start at the top and draw
downward); only the arrowhead end differs. For kInvisBlower/kLiftArea it additionally
draws ovals (`FrameOvalDialogItem`) around the three *non-selected* hot-spots and
`EraseDialogItem`s the selected one — an inverted-looking affordance where the *unringed*
arrow is the current direction.

## 12.4 Furniture info — DLOG/DITL **1010**, `DoFurnitureObjectInfo(void)`

`GliderPRO/Sources/ObjectInfo.c:1107`. Dialog **120×256**. This is the "no editable
properties" dialog and serves **57 object types** (§12.15) plus the three glider
pseudo-objects.

| DITL item | Type | Content |
|---|---|---|
| 1 | btnCtrl | `Okay` |
| 2 | statText | (label) |
| 3 | statText | `Object Number: ^0` |
| 4 | userItem | divider, framed `kRedOrangeColor8` |
| 5 | statText | `Object Kind: ^1` |
| 6 | btnCtrl | `Linked From?` |

There is **no Cancel** — the only exits are Okay (item 1) and Linked From? (item 6).
`FurnitureFilter` accordingly handles only Return/Enter (no Escape).

The glider pseudo-objects get literal names instead of a `GetIndString` lookup
(`ObjectInfo.c:1117-1131`):

| `objActive` | value | `^0` | `^1` |
|---|---|---|---|
| `kInitialGliderSelected` | **−2** | `-` | `Glider Begins` |
| `kLeftGliderSelected` | **−3** | `-` | `New Glider (left)` |
| `kRightGliderSelected` | **−4** | `-` | `New Glider (right)` |

**Latent out-of-bounds read:** the guard at `ObjectInfo.c:1141` is
`if ((objActive < 0) || (retroLinkList[objActive].room == -1)) HideDialogItem(infoDial, 6)`
— C's `||` short-circuits, so the negative index is not dereferenced. But the *post*-loop
`doReturn` path at `ObjectInfo.c:1165` indexes `retroLinkList[objActive]` with no such
guard. It is unreachable only because item 6 was hidden (hidden items cannot be hit by
`ModalDialog`), which is a fragile invariant. Every other dialog in the file omits the
`objActive < 0` half of the test entirely — safe because only this dialog can be reached
with a negative `objActive`.

## 12.5 Custom picture / sound trigger info — DLOG/DITL **1045**, `DoCustPictObjectInfo(void)`

`GliderPRO/Sources/ObjectInfo.c:1172`. Dialog **141×256**. One dialog, two completely
different fields, chosen by `what`.

| DITL item | Type | Content |
|---|---|---|
| 1 | btnCtrl | `Okay` |
| 2 | btnCtrl | `Cancel` |
| 3 | statText | `Object Number: ^0` |
| 4 | statText | `Object Kind: ^1` |
| 5 | userItem | divider, framed `kRedOrangeColor8` |
| 6 | statText | (label) |
| 7 | editText | the ID (`kCustPictIDItem`) |
| 8 | statText | `^2 ID:` |
| 9 | statText | `(I.D. > ^3)` |

| `what` | field edited | `ParamText ^2` | `^3` | accepted range | on failure |
|---|---|---|---|---|---|
| `kCustomPict` (0x6E) | `data.g.height` | `PICT` | `10000` | **10000 ≤ v ≤ 32767** | `SysBeep(1)`, restore old value, reselect text, stay in loop |
| `kSoundTrigger` (0x49) | `data.e.where` | `Sound` | `3000` | **3000 ≤ v ≤ 32767** | ditto |

On success both paths do `fileDirty = true; UpdateMenus(false); InvalWindowRect;
GetThisRoomsObjRects(); ReadyBackground(...); DrawThisRoomsObjects(); leaving = true`.
The kCustomPict path additionally calls `KeepObjectLegal()` first (because changing the
PICT changes the object's size), the kSoundTrigger path does not.

This dialog is the *only* place a `snd ` resource ID is authored, and it confirms the
field overload documented in §9.9: for kSoundTrigger, `data.e.where` is a resource ID,
not a room link. Note the validation only enforces a lower bound and 32767 — it does
**not** verify the resource exists, so a house can reference a missing PICT or sound.

## 12.6 Switch info — DLOG/DITL **1011**, `DoSwitchObjectInfo(void)`

`GliderPRO/Sources/ObjectInfo.c:1269`. Dialog **187×256**. Serves the six switch types
(kLightSwitch, kMachineSwitch, kThermostat, kPowerSwitch, kKnifeSwitch, kInvisSwitch).

| DITL item | Type | Content |
|---|---|---|
| 1 | btnCtrl | `Okay` |
| 2 | btnCtrl | `Cancel` |
| 3 | statText | `Object Number: ^0` |
| 4 | userItem | divider, framed |
| 5 | statText | `Object Kind: ^1` |
| 6 | radCtrl | `Toggle` (`kToggleRadio`) |
| 7 | radCtrl | `Force On` (`kForceOnRadio`) |
| 8 | radCtrl | `Force Off` (`kForceOffRadio`) |
| 9 | btnCtrl | `Link` |
| 10 | picItem | illustration |
| 11 | statText | `Room Link: ^2` |
| 12 | statText | `Object Link: ^3` |
| 13 | userItem | second divider, framed |
| 14 | btnCtrl | `Go To` (`kGotoButton2`) |
| 15 | btnCtrl | `Linked From?` |

Seeding:

```
1. if data.e.where == -1:  roomStr = "none"
2. else: ExtractFloorSuite(data.e.where, &floor, &suite);  roomStr = "<floor> / <suite>"
3. if data.e.who == 255:   objStr = "none"    else objStr = <who + 1>
4. ParamText(numberStr, kindStr, roomStr, objStr)
5. newType = data.e.type
6. if data.e.who == 255: MyDisableControl(item 14)          // Go To greyed
7. if retroLinkList[objActive].room == -1: HideDialogItem(item 15)
```

`UpdateSwitchInfo` calls `SelectFromRadioGroup(dial, newType + kToggleRadio, 6, 8)`, i.e.
the radio index **is** `data.e.type + 6`, so:

| `data.e.type` | constant | value | radio |
|---|---|---|---|
| `kToggle` | 0 | item 6 | Toggle |
| `kForceOn` | 1 | item 7 | Force On |
| `kForceOff` | 2 | item 8 | Force Off |
| `kOneShot` | 3 | *item 9* — **the Link button!** | — |

A switch with `type == kOneShot` (only triggers get that by default, but nothing prevents
it) makes `UpdateSwitchInfo` "select" item 9, a push button, on every update event.
`SelectFromRadioGroup` sets item 9's control value to 1 and 6..8 to 0, which for a
`btnCtrl` is a harmless no-op visually but means **no radio appears selected**.

Item dispatch: items 1, 9, 14, 15 all first commit `data.e.type = newType` and set
`fileDirty`; then item 9 sets `doLink`, 14 sets `doGoTo`, 15 sets `doReturn`. Items 6/7/8
set `newType` to 0/1/2. Item 2 leaves without writing.

The link-out path (`ObjectInfo.c:1369`):

```
linkType = kSwitchLinkOnly      // 3
linkerIsSwitch = true
OpenLinkWindow()
linkRoom = thisRoomNumber
linkObject = (Byte)objActive
DeselectObject()
```

**Bug:** the `doGoTo` path at `ObjectInfo.c:1380` passes `floor` and `suite`, which are
only assigned inside the `else` at step 2. If `data.e.where == -1` they are
**uninitialized stack values** — but the Go To button is disabled when `who == 255`, and a
`where` of −1 with a valid `who` is possible (it is exactly the "1 room-only link"
observed in the shipped houses, §17.6). So Go To with garbage coordinates is reachable in
principle.

## 12.7 Trigger info — DLOG/DITL **1034**, `DoTriggerObjectInfo(void)`

`GliderPRO/Sources/ObjectInfo.c:1391`. Dialog **187×256**. Serves kTrigger and
kLgTrigger. Identical to Switch info except the three radios are replaced by a delay
field and there is no `type` editing at all.

| DITL item | Type | Content |
|---|---|---|
| 1/2 | btnCtrl | `Okay` / `Cancel` |
| 3 | statText | `Object Number: ^0` |
| 4 | userItem | divider, framed |
| 5 | statText | `Object Kind: ^1` |
| 6 | editText | delay (`kDelay3Item`) |
| 7 | statText | `Delay:` |
| 8 | statText | `(1/10 secs)` |
| 9 | btnCtrl | `Link` |
| 10 | picItem | illustration |
| 11 | statText | `Room Link: ^2` |
| 12 | statText | `Object Link: ^3` |
| 13 | userItem | second divider, framed |
| 14 | btnCtrl | `Go To` |
| 15 | btnCtrl | `Linked From?` |

`data.e.delay` is validated as **0 ≤ delayIs ≤ 32767** (a `short`, unlike the appliance
and enemy delays which are clamped to 255 — see §12.9/§12.13). On out-of-range: `SysBeep(1)`,
restore, reselect, stay. The identical validate-then-commit block is **duplicated four
times** (items 1, 9, 14, 15) at `ObjectInfo.c:1445`, `:1465`, `:1484`, `:1503`.

`newType` is read at `ObjectInfo.c:1421` but never written back — so `data.e.type`
(kOneShot for triggers by default) is immutable through this dialog.

Link-out uses `linkType = kTriggerLinkOnly` (**4**) and `linkerIsSwitch = true`.

## 12.8 Light info — DLOG/DITL **1013**, `DoLightObjectInfo(void)`

`GliderPRO/Sources/ObjectInfo.c:1547`. Dialog **143×256**. Note this one does **not** use
`BringUpDialog`; it inlines `GetNewDialog` / `SetPort` / seeding / `ShowWindow` so that
the checkbox is set before the window becomes visible (`ObjectInfo.c:1562-1575`).

| DITL item | Type | Content |
|---|---|---|
| 1/2 | btnCtrl | `Okay` / `Cancel` |
| 3 | statText | `Object Number: ^0` |
| 4 | statText | `Object Kind: ^1` |
| 5 | userItem | divider, framed |
| 6 | chkCtrl | `Initially On` (`kInitialStateCheckbox`) |
| 7 | picItem | illustration |
| 8 | btnCtrl | `Linked From?` |

Only `data.f.initial` is editable. Both commit paths (items 1 and 8) do
`ReadyBackground(...)`, `DrawThisRoomsObjects()`, `InvalWindowRect(...)` — a full
recomposite, because a light's initial state changes whether the room renders lit or with
the dark-gray wash (§4.3).

`data.f.state` is *not* touched by the dialog; it is the runtime state and is
re-initialised from `initial` at play time.

## 12.9 Appliance info — DLOG/DITL **1014**, `DoApplianceObjectInfo(short what)`

`GliderPRO/Sources/ObjectInfo.c:1633`. Dialog **153×256**. Serves kShredder, kToaster,
kMacPlus, kTV, kCoffee, kOutlet, kVCR.

| DITL item | Type | Content |
|---|---|---|
| 1/2 | btnCtrl | `Okay` / `Cancel` |
| 3 | statText | `Object Number: ^0` |
| 4 | statText | `Object Kind: ^1` |
| 5 | userItem | divider, framed |
| 6 | chkCtrl | `Initially On` (`kInitialStateCheckbox`) |
| 7 | picItem | illustration |
| 8 | editText | interval (`kDelayItem`) |
| 9 | statText | `Interval:\r(1/10 sec)` (`kDelayLabelItem`) |
| 10 | btnCtrl | `Linked From?` |

Items 8 and 9 are **hidden** when `what` ∈ {kShredder, kMacPlus, kTV, kCoffee, kVCR,
kMicrowave} (`ObjectInfo.c:1658-1663`) — i.e. only kToaster and kOutlet have a settable
interval. (kMicrowave is in the list even though it never reaches this function; it has
its own dialog, §12.10.)

`data.g.delay` is validated as **0 ≤ delay ≤ 255** and stored with `(Byte)` truncation —
note the *struct* field is a `short`, so the dialog is narrower than the format. On
failure: `SysBeep(0)` (note: 0 here, 1 in the trigger dialog), restore, reselect, stay.

The two commit blocks (items 1 and 10) are byte-identical at `ObjectInfo.c:1678` and
`:1709`.

## 12.10 Microwave info — DLOG/DITL **1035**, `DoMicrowaveObjectInfo(void)`

`GliderPRO/Sources/ObjectInfo.c:1749`. Dialog **160×278** — the only object dialog that
is *not* 256 px wide.

| DITL item | Type | Content |
|---|---|---|
| 1/2 | btnCtrl | `Okay` / `Cancel` |
| 3 | statText | `Object Number: ^0` |
| 4 | statText | `Object Kind: ^1` |
| 5 | userItem | divider, framed |
| 6 | chkCtrl | `Initially On` (`kInitialStateCheckbox`) |
| 7 | picItem | illustration |
| 8 | chkCtrl | `Zero Bands` (`kKillBandsCheckbox`) |
| 9 | chkCtrl | `Zero Battery/He` (`kKillBatteryCheckbox`) |
| 10 | chkCtrl | `Zero Foil` (`kKillFoilCheckbox`) |
| 11 | btnCtrl | `Linked From?` |

`data.g.byte0` is the "kills" bit field, read at `ObjectInfo.c:1773` and rebuilt at
`:1801-1811`:

| Bit | Value | Checkbox | Destroys |
|---|---|---|---|
| 0 | **1** | item 8 | rubber bands |
| 1 | **2** | item 9 | battery **and** helium |
| 2 | **4** | item 10 | aluminium foil |

`AddNewObject` defaults it to **7** (all three). Bits 3-7 are unused and are **destroyed
on Okay** because `kills` is rebuilt from zero rather than masked.

## 12.11 Grease info — DLOG/DITL **1019**, `DoGreaseObjectInfo(void)`

`GliderPRO/Sources/ObjectInfo.c:1873`. Dialog **146×256**.

| DITL item | Type | Content |
|---|---|---|
| 1/2 | btnCtrl | `Okay` / `Cancel` |
| 3 | statText | `Object Number: ^0` |
| 4 | statText | `Object Kind: ^1` |
| 5 | userItem | divider, framed |
| 6 | chkCtrl | `Grease initially tipped` (`kGreaseItem`) |
| 7 | picItem | illustration |
| 8 | btnCtrl | `Linked From?` |

The checkbox is the **logical negation** of the stored field
(`ObjectInfo.c:1892`, `:1903`):

```
wasSpilled = !(data.c.initial)         // seeding
...
data.c.initial = !wasSpilled           // commit
```

So `data.c.initial == true` means "grease is still in the tin" (not tipped). The dialog
tracks `wasSpilled` in a local and drives the checkbox with `SetDialogItemValue` directly
rather than `ToggleDialogItemValue`, so the control value and the local can never
disagree.

## 12.12 Invisible bonus info — DLOG/DITL **1015**, `DoInvisBonusObjectInfo(void)`

`GliderPRO/Sources/ObjectInfo.c:1947`. Dialog **155×256**.

| DITL item | Type | Content |
|---|---|---|
| 1/2 | btnCtrl | `Okay` / `Cancel` |
| 3 | statText | `Object Number: ^0` |
| 4 | userItem | divider, framed |
| 5 | statText | `Object Kind: ^1` |
| 6 | radCtrl | `100 Points` (`k100PtRadio`) |
| 7 | radCtrl | `300 Points` (`k300PtRadio`) |
| 8 | radCtrl | `500 Points` (`k500PtRadio`) |
| 9 | btnCtrl | `Linked From?` |

`newPoint` is a 0/1/2 index, mapped both ways with explicit switches — there is no
arithmetic relationship, so **only 100, 300 and 500 are representable**, and any other
stored value silently becomes 100 on the next Okay:

| stored `data.c.points` | `newPoint` | radio | committed value |
|---|---|---|---|
| 300 | 1 | item 7 | 300 |
| 500 | 2 | item 8 | 500 |
| anything else (incl. 100) | 0 | item 6 | **100** |

`UpdateInvisBonusInfo` uses `SelectFromRadioGroup(dial, newPoint + k100PtRadio, 6, 8)`.

This is the **only** object dialog whose commit path does *not* redraw the room (there is
nothing visible to redraw — the object is invisible) — it does only
`fileDirty = true; UpdateMenus(false)`.

## 12.13 Transport info — DLOG/DITL **1022**, `DoTransObjectInfo(short what)`

`GliderPRO/Sources/ObjectInfo.c:2060`. Dialog **185×256**. Serves kMailboxLf,
kMailboxRt, kFloorTrans, kCeilingTrans, kInvisTrans, kDeluxeTrans.

| DITL item | Type | Content |
|---|---|---|
| 1/2 | btnCtrl | `Okay` / `Cancel` |
| 3 | statText | `Object Number: ^0` |
| 4 | userItem | divider, framed |
| 5 | statText | `Object Kind: ^1` |
| 6 | btnCtrl | `Link` (`kLinkTransButton`) |
| 7 | picItem | illustration |
| 8 | statText | `Room Link: ^2` (`kTransRoomText`) |
| 9 | statText | `Object Link: ^3` (`kTransObjectText`) |
| 10 | userItem | second divider, framed |
| 11 | btnCtrl | `Go To` (`kGotoButton1`) |
| 12 | btnCtrl | `Linked From?` |
| 13 | chkCtrl | `Initially On` (`kInitialStateCheckbox3`) |

Item 13 is hidden unless `what == kDeluxeTrans` (`ObjectInfo.c:2094`). For kDeluxeTrans:

```
seed:   wasState = (data.d.wide & 0xF0) >> 4
commit: data.d.wide = wasState << 4
```

so the "initially on" flag lives in the **high nibble of `data.d.wide`**, and the commit
**zeroes the low nibble** (which is unused for kDeluxeTrans, so this is safe here — but
it would destroy the width of a kInvisTrans, which is why item 13 is hidden for that
type). `wasState` is a `Boolean`, so only values 0x00 and 0x10 are ever written.

Room/object link display and the disabled Go To follow §12.6 exactly, using `data.d`
instead of `data.e`. Link-out uses `linkType = kTransportLinkOnly` (**5**) and
`linkerIsSwitch = false`.

**Same uninitialized-`floor`/`suite` hazard** as §12.6, at `ObjectInfo.c:2172`.

## 12.14 Enemy info — DLOG/DITL **1027**, `DoEnemyObjectInfo(short what)`

`GliderPRO/Sources/ObjectInfo.c:2183`. Dialog **160×256**. Serves kBalloon, kCopterLf,
kCopterRt, kDartLf, kDartRt, kBall, kDrip, kFish.

| DITL item | Type | Content |
|---|---|---|
| 1/2 | btnCtrl | `Okay` / `Cancel` |
| 3 | statText | `Object Number: ^0` |
| 4 | userItem | divider, framed |
| 5 | statText | `Object Kind: ^1` |
| 6 | picItem | illustration |
| 7 | editText | delay (`kDelay2Item`) |
| 8 | statText | `Idle Delay:` (`kDelay2LabelItem`) |
| 9 | statText | `(1/10 secs.)` (`kDelay2LabelItem2`) |
| 10 | chkCtrl | `Initially "On"` (`kInitialStateCheckbox2`) |
| 11 | btnCtrl | `Linked From?` |

Items 7, 8, 9 are hidden when `what == kBall` (`ObjectInfo.c:2212`) — a bouncing ball has
no idle delay. Correspondingly the validation and the store are both guarded:

```
if (((delay < 0L) || (delay > 255L)) && (what != kBall))  -> reject
...
if (what != kBall) data.h.delay = (Byte)delay;
```

Range **0..255** with `(Byte)` truncation into a `short` field, same as §12.9.
`SysBeep(0)` on failure. `data.h.initial` is the checkbox. Note this dialog does **not**
redraw the room on commit (only `fileDirty` + `UpdateMenus`).

## 12.15 Flower info — DLOG/DITL **1033**, `DoFlowerObjectInfo(void)`

`GliderPRO/Sources/ObjectInfo.c:2293`. Dialog **175×256**.

| DITL item | Type | Content |
|---|---|---|
| 1 | btnCtrl | `Okay` |
| 2 | statText | (label) |
| 3 | statText | `Object Number: ^0` |
| 4 | userItem | divider, framed |
| 5 | statText | `Object Kind: ^1` |
| 6 | radCtrl | `Dandelion` (`kRadioFlower1`) |
| 7 | radCtrl | `Tulip` |
| 8 | radCtrl | `Orchid` |
| 9 | radCtrl | `Violets` |
| 10 | radCtrl | `Daisies` |
| 11 | radCtrl | `Sunflower` (`kRadioFlower6`) |
| 12 | btnCtrl | `Cancel` (`kFlowerCancel`) — **note: item 12, not item 2** |
| 13 | btnCtrl | `Linked From?` |

The species index is `data.i.pict`, mapped directly: `flower = data.i.pict + kRadioFlower1`
(so `pict` 0..5 ↔ items 6..11). The names confirm the `flowerSrc[]` table in §9.13.

Commit is conditional on an actual change (`ObjectInfo.c:2325`) and **resizes the
object's bounds anchored at bottom-left**:

```
1. InvalWindowRect(mainWindow, &data.i.bounds)                  // erase old
2. data.i.bounds.right = data.i.bounds.left + RectWide(&flowerSrc[flower])
3. data.i.bounds.top   = data.i.bounds.bottom - RectTall(&flowerSrc[flower])
4. data.i.pict = flower
5. InvalWindowRect(mainWindow, &data.i.bounds)                  // draw new
6. GetThisRoomsObjRects(); ReadyBackground(...); DrawThisRoomsObjects()
7. fileDirty = true; UpdateMenus(false)
8. wasFlower = flower                    // becomes the shift-click default (§9.13)
```

So a flower keeps its **bottom-left corner** when the species changes — correct, since
flowers grow from the ground. Note the `Cancel` at item 12 means this dialog's Escape key
is *not* handled by `FlowerFilter` (which only maps Return/Enter to item 1), so Escape
does nothing here.

## 12.16 The dispatch table — `DoObjectInfo`

`GliderPRO/Sources/ObjectInfo.c:2392`. First, the three glider pseudo-selections route to
the furniture dialog:

```
if (objActive is kInitialGliderSelected or kLeftGliderSelected or kRightGliderSelected)
    { DoFurnitureObjectInfo(); return; }
```

Then a 144-case switch on `thisRoom->objects[objActive].what`:

| Handler | Dialog | `what` codes served | count |
|---|---|---|---|
| `DoBlowerObjectInfo(what)` | 1007 | kFloorVent, kCeilingVent, kFloorBlower, kCeilingBlower, kSewerGrate, kLeftFan, kRightFan, kTaper, kCandle, kStubby, kTiki, kBBQ, kInvisBlower, kGrecoVent, kSewerBlower, kLiftArea | 16 |
| `DoFurnitureObjectInfo()` | 1010 | kTable, kShelf, kCabinet, kFilingCabinet, kWasteBasket, kMilkCrate, kCounter, kDresser, kDeckTable, kStool, kTrunk, kInvisObstacle, kManhole, kBooks, kInvisBounce, kRedClock, kBlueClock, kYellowClock, kCuckoo, kPaper, kBattery, kBands, kFoil, kStar, kSparkle, kHelium, kSlider, kUpStairs, kDownStairs, kDoorInLf, kDoorInRt, kDoorExRt, kDoorExLf, kWindowInLf, kWindowInRt, kWindowExRt, kWindowExLf, kCinderBlock, kFlowerBox, kCDs, kGuitar, kStereo, kCobweb, kOzma, kMirror, kMousehole, kFireplace, kWallWindow, kBear, kCalendar, kVase1, kVase2, kBulletin, kCloud, kFaucet, kRug, kChimes | **57** |
| `DoGreaseObjectInfo()` | 1019 | kGreaseRt, kGreaseLf | 2 |
| `DoInvisBonusObjectInfo()` | 1015 | kInvisBonus | 1 |
| `DoTransObjectInfo(what)` | 1022 | kMailboxLf, kMailboxRt, kFloorTrans, kCeilingTrans, kInvisTrans, kDeluxeTrans | 6 |
| `DoSwitchObjectInfo()` | 1011 | kLightSwitch, kMachineSwitch, kThermostat, kPowerSwitch, kKnifeSwitch, kInvisSwitch | 6 |
| `DoTriggerObjectInfo()` | 1034 | kTrigger, kLgTrigger | 2 |
| `DoLightObjectInfo()` | 1013 | kCeilingLight, kLightBulb, kTableLamp, kHipLamp, kDecoLamp, kFlourescent, kTrackLight, kInvisLight | 8 |
| `DoApplianceObjectInfo(what)` | 1014 | kShredder, kToaster, kMacPlus, kTV, kCoffee, kOutlet, kVCR | 7 |
| `DoMicrowaveObjectInfo()` | 1035 | kMicrowave | 1 |
| `DoEnemyObjectInfo(what)` | 1027 | kBalloon, kCopterLf, kCopterRt, kDartLf, kDartRt, kBall, kDrip, kFish | 8 |
| `DoFlowerObjectInfo()` | 1033 | kFlower | 1 |
| `DoCustPictObjectInfo()` | 1045 | kSoundTrigger, kCustomPict | 2 |
| `default:` | — | `SysBeep(1)` | — |

Total: 16 + 57 + 2 + 1 + 6 + 6 + 2 + 8 + 7 + 1 + 8 + 1 + 2 = **117** types dispatched.
The `what` code space is 0x01..0x8F = 143 codes, of which 26 are the unused class gaps
(§3.12), and 143 − 26 = **117** ✓. So every live `what` code has exactly one dialog, and
only the 26 gap codes can reach the `default: SysBeep(1)` arm.

Notable routings a porter would not guess:

* **kStar, kSparkle, kBattery, kFoil, kBands, kHelium, kPaper** and the three clocks all
  go to the *furniture* (no-properties) dialog — prizes have no editable properties.
* **kUpStairs / kDownStairs** and all eight **doors/windows** also go to the furniture
  dialog: they are position-locked and unlinked (their `data.d.where`/`who` exist but are
  never authored through a dialog — pairing is positional, §10.2).
* **kGuitar, kStereo, kCinderBlock, kFlowerBox, kCDs** are appliance-class `what` codes
  that route to the *furniture* dialog, because they have no interval or initial state.
* **kCobweb** is an enemy-class code that routes to the furniture dialog.
* **kSoundTrigger** shares a dialog with **kCustomPict** even though it is a switch-class
  code, because both edit a resource ID.

## 12.17 `STR# 1007` "Object Names" — all 144 strings

The dialogs' `Object Kind: ^1` and the tool palette's name strip both index this list by
`what` directly (1-based, so index N is `what` N). Parsed from
`GliderPRO/Glider PRO.r`:

| `what` | hex | Name | | `what` | hex | Name |
|---|---|---|---|---|---|---|
| 1 | 0x01 | Floor Vent | | 73 | 0x49 | Sound Trigger |
| 2 | 0x02 | Ceiling Vent | | 74 | 0x4A | `9` *(gap)* |
| 3 | 0x03 | Floor Duct | | 75 | 0x4B | `a` *(gap)* |
| 4 | 0x04 | Ceiling Duct | | 76 | 0x4C | `b` *(gap)* |
| 5 | 0x05 | Sewer Grate | | 77 | 0x4D | `c` *(gap)* |
| 6 | 0x06 | Table Fan | | 78 | 0x4E | `d` *(gap)* |
| 7 | 0x07 | Table Fan | | 79 | 0x4F | `e` *(gap)* |
| 8 | 0x08 | Taper | | 80 | 0x50 | `f` *(gap)* |
| 9 | 0x09 | Simple Candle | | 81 | 0x51 | Ceiling Light |
| 10 | 0x0A | Stubby Candle | | 82 | 0x52 | Simple Bulb |
| 11 | 0x0B | Tiki Torch | | 83 | 0x53 | Table Lamp |
| 12 | 0x0C | Barbecue Grill | | 84 | 0x54 | Hip Pole Lamp |
| 13 | 0x0D | Invisible Blower | | 85 | 0x55 | Deco Lamp |
| 14 | 0x0E | Greco-Roman Vent | | 86 | 0x56 | Flourescent Light |
| 15 | 0x0F | Sewer Blower | | 87 | 0x57 | Track Lighting |
| 16 | 0x10 | Lift Area | | 88 | 0x58 | Invisible Light |
| 17 | 0x11 | Table | | 89 | 0x59 | `8` *(gap)* |
| 18 | 0x12 | Shelf | | 90 | 0x5A | `9` *(gap)* |
| 19 | 0x13 | Cabinet | | 91 | 0x5B | `a` *(gap)* |
| 20 | 0x14 | Filing Cabinet | | 92 | 0x5C | `b` *(gap)* |
| 21 | 0x15 | Wastebasket | | 93 | 0x5D | `c` *(gap)* |
| 22 | 0x16 | Milk Crate | | 94 | 0x5E | `d` *(gap)* |
| 23 | 0x17 | Counter | | 95 | 0x5F | `e` *(gap)* |
| 24 | 0x18 | Dresser | | 96 | 0x60 | `f` *(gap)* |
| 25 | 0x19 | Deck Table | | 97 | 0x61 | Paper Shredder |
| 26 | 0x1A | Bar Stool | | 98 | 0x62 | Toaster |
| 27 | 0x1B | Steamer Trunk | | 99 | 0x63 | Mac Plus |
| 28 | 0x1C | Invisible Obstacle | | 100 | 0x64 | Guitar |
| 29 | 0x1D | Manhole | | 101 | 0x65 | T.V. |
| 30 | 0x1E | Books | | 102 | 0x66 | Coffee Machine |
| 31 | 0x1F | Invisible Rebounder | | 103 | 0x67 | Electrical Outlet |
| 32 | 0x20 | `f` *(gap)* | | 104 | 0x68 | VCR |
| 33 | 0x21 | Digital Clock | | 105 | 0x69 | Stereo System |
| 34 | 0x22 | Wall Clock | | 106 | 0x6A | Microwave Oven |
| 35 | 0x23 | Alarm Clock | | 107 | 0x6B | Cinder Block |
| 36 | 0x24 | Cuckoo Clock | | 108 | 0x6C | Flower Box |
| 37 | 0x25 | **Extra Glider** | | 109 | 0x6D | Compact Discs |
| 38 | 0x26 | Battery | | 110 | 0x6E | Custom Picture |
| 39 | 0x27 | Rubber Bands (8) | | 111 | 0x6F | `e` *(gap)* |
| 40 | 0x28 | Grease (spills rt.) | | 112 | 0x70 | `f` *(gap)* |
| 41 | 0x29 | Grease (spills lf.) | | 113 | 0x71 | Balloon |
| 42 | 0x2A | Aluminum Foil | | 114 | 0x72 | 'Copter (lf. drift) |
| 43 | 0x2B | Invisible Bonus | | 115 | 0x73 | 'Copter (rt. drift) |
| 44 | 0x2C | Magic Star | | 116 | 0x74 | Dart (lf. moving) |
| 45 | 0x2D | Sparkle | | 117 | 0x75 | Dart (rt. moving) |
| 46 | 0x2E | Helium (He) | | 118 | 0x76 | Bouncing Ball |
| 47 | 0x2F | Slide Rect | | 119 | 0x77 | Water Drip |
| 48 | 0x30 | `f` *(gap)* | | 120 | 0x78 | Fish Bowl & Fish |
| 49 | 0x31 | Up Stairs | | 121 | 0x79 | Cobweb |
| 50 | 0x32 | Down Stairs | | 122 | 0x7A | `9` *(gap)* |
| 51 | 0x33 | Mailbox (faces lf.) | | 123 | 0x7B | `a` *(gap)* |
| 52 | 0x34 | Mailbox (faces rt.) | | 124 | 0x7C | `b` *(gap)* |
| 53 | 0x35 | Floor Trans. Duct | | 125 | 0x7D | `c` *(gap)* |
| 54 | 0x36 | Ceiling Trans. Duct | | 126 | 0x7E | `d` *(gap)* |
| 55 | 0x37 | Door (interior) | | 127 | 0x7F | `e` *(gap)* |
| 56 | 0x38 | Door (interior) | | 128 | 0x80 | `f` *(gap)* |
| 57 | 0x39 | Door (exterior) | | 129 | 0x81 | Ozma |
| 58 | 0x3A | Door (exterior) | | 130 | 0x82 | Mirror |
| 59 | 0x3B | Window (interior) | | 131 | 0x83 | Mouse Hole |
| 60 | 0x3C | Window (interior) | | 132 | 0x84 | Fireplace |
| 61 | 0x3D | Window (exterior) | | 133 | 0x85 | Flower |
| 62 | 0x3E | Window (exterior) | | 134 | 0x86 | Window (closed) |
| 63 | 0x3F | Invisible Transport | | 135 | 0x87 | Teddy Bear |
| 64 | 0x40 | Deluxe Transport | | 136 | 0x88 | Calendar |
| 65 | 0x41 | Light Switch | | 137 | 0x89 | Broad Vase |
| 66 | 0x42 | Machine Switch | | 138 | 0x8A | Narrow Vase |
| 67 | 0x43 | Thermostat | | 139 | 0x8B | Bulletin Board |
| 68 | 0x44 | Digital Switch | | 140 | 0x8C | Cloud |
| 69 | 0x45 | Knife Switch | | 141 | 0x8D | Faucet |
| 70 | 0x46 | Invisible Switch | | 142 | 0x8E | Throw Rug |
| 71 | 0x47 | Trigger | | 143 | 0x8F | Wind Chimes |
| 72 | 0x48 | Large Trigger | | **144** | **0x90** | **Mermaid** *(no such `what`)* |

Observations:

* **`what` 0x25 is named "Extra Glider"**, not "Paper" — `kPaper` is the paper-airplane
  prize that grants a spare glider. The C constant name is misleading.
* The 26 **gap codes** carry single-character placeholder names (`8`, `9`, `a`..`f`) that
  are literally the hex digit of the low nibble, left over from the author's numbering
  scheme. They are the codes at 0x20, 0x30, 0x4A-0x50, 0x59-0x60, 0x6F, 0x70, 0x7A-0x80.
* **Index 144 (0x90) is "Mermaid"**, one past `kChimes` (0x8F) and equal to
  `kNumSrcRects` (0x90 = 144). There is no `what` code 0x90, no `srcRects` slot for it
  (the array is 0..0x8F), and no dispatch arm. It is either a cut feature or an off-by-one
  guard string. A Go port should size its name table at 144 entries but never reach 144.
* Duplicated names are intentional: 6/7 "Table Fan" (left/right), 55/56 "Door
  (interior)", 57/58 "Door (exterior)", 59/60 "Window (interior)", 61/62 "Window
  (exterior)" — the Lf/Rt distinction is not surfaced to the user because §9.8.1 chooses
  the side automatically.
* `kObjectNameStrings` = **1007** (`GliderPRO/Headers/GliderDefines.h:461`) — the same
  numeric value as the blower **DLOG** id 1007. They are different resource *types*
  (`STR#` vs `DLOG`), so this is legal on the Mac but is a trap for a port that flattens
  resources into one namespace.

---

# 13. Room Info — DLOG/DITL **1003** and "Select Original Art" DLOG/DITL **1016**

`DoRoomInfo()` (`GliderPRO/Sources/RoomInfo.c:376`) is the per-room property sheet. It is
reached three ways:

| Trigger | Site |
|---|---|
| House menu → `iRoom` ("Room Info…", cmd-R) | `GliderPRO/Sources/Menu.c:479` — guarded by `if (houseUnlocked)` |
| Double-click on empty floor in the edit window | `GliderPRO/Sources/ObjectEdit.c:73` (`DoSelectionClick`, see §6.2) |
| Automatically after `CreateNewRoom` when `autoRoomEdit` is on | `GliderPRO/Sources/Events.c:458` idle task tests `autoRoomEdit && newRoomNow` |

## 13.1 Constants (`GliderPRO/Sources/RoomInfo.c:18`-`33`)

| Name | Value | Meaning |
|---|---|---|
| `kRoomInfoDialogID` | 1003 | DLOG/DITL for the room sheet |
| `kOriginalArtDialogID` | 1016 | DLOG/DITL for "Select Original Art" |
| `kNoPICTFoundAlert` | 1036 | ALRT shown when a room's PICT is missing |
| `kRoomNameItem` | 3 | editText, the room name |
| `kRoomLocationBox` | 6 | userItem framed red-orange around Floor/Suite |
| `kRoomTilesBox` | 10 | userItem, the **source** tile strip |
| `kRoomPopupItem` | 11 | resCtrl, the background popup (CNTL 128) |
| `kRoomDividerLine` | 12 | userItem, 1-pixel-high red-orange divider |
| `kRoomTilesBox2` | 15 | userItem, the **destination** tile strip |
| `kRoomFirstCheck` | 17 | chkCtrl "First Room" |
| `kLitUnlitText` | 18 | statText "(Room Is Dark)"/"(Room Is Lit)" |
| `kMiniTileWide` | 16 | width in pixels of one mini-tile |
| `kBoundsButton` | 19 | btnCtrl "Bounds" **in DITL 1003** |
| `kOriginalArtworkItem` | 19 | item index **in MENU 140** — same number, different namespace |
| `kPICTIDItem` | 5 | editText **in DITL 1016** |
| `kFloorSupportCheck` | 12 | chkCtrl **in DITL 1016** — collides numerically with `kRoomDividerLine` |

Two deliberate-looking collisions to be careful about when porting: `kBoundsButton` and
`kOriginalArtworkItem` are both 19 but one is a DITL item and the other a MENU item index;
`kRoomDividerLine` and `kFloorSupportCheck` are both 12 but live in DITL 1003 and DITL 1016
respectively.

## 13.2 File-scope state (`GliderPRO/Sources/RoomInfo.c:48`-`54`)

| Declaration | Purpose |
|---|---|
| `Rect tileSrc` | screen rect of DITL item 10 (source strip) |
| `Rect tileDest` | screen rect of DITL item 15 (destination strip) |
| `Rect tileSrcRect` | `{0,0,80,128}` — the offscreen thumbnail bounds, set once in `CreateOffscreens` (`GliderPRO/Sources/StructuresInit2.c:178`) |
| `Rect editTETextBox` | screen rect of DITL item 3, used only for I-beam cursor tracking |
| `Rect leftBound, topBound, rightBound, bottomBound` | screen rects of DITL 1016 items 7,8,9,10 |
| `CGrafPtr tileSrcMap` | 8-bit GWorld holding the 128×80 scaled background |
| `short tempTiles[kNumTiles]` | working copy of `thisRoom->tiles[8]`; committed only on OK |
| `short tileOver` | index 0..7 of the source column under the mouse, or −1 |
| `short tempBack` | working copy of `thisRoom->background`; committed only on OK |
| `short cursorIs` | `kArrowCursor` 0 / `kBeamCursor` 1 / `kHandCursor` 2 (`GliderPRO/Headers/GliderDefines.h:182`-`184`) |
| `Boolean originalLeftOpen, originalTopOpen, originalRightOpen, originalBottomOpen` | working copies of the four "wall open" bits |
| `Boolean originalFloor` | working copy of the "floor support" bit |

`tileSrcMap` is created on entry to `DoRoomInfo` and disposed on exit
(`GliderPRO/Sources/RoomInfo.c:407` and `:567`); it is `nil` at all other times
(`GliderPRO/Sources/StructuresInit2.c:179`).

## 13.3 DITL 1003, verified from `Glider PRO.r`

`DLOG 1003 "Room Info"` — `rect(t0,l0,b267,r384)` = **267 high × 384 wide**, `procID` 1
(`dBoxProc` + 1 = modal, no grow), initially invisible, no go-away box, DITL 1003, and a
`dctb 1003 "Room Info"` colour table.

| # | Type | Rect (t,l,b,r) | Text / resource |
|---|---|---|---|
| 1 | btnCtrl | 239,318,259,376 | `Okay` |
| 2 | btnCtrl | 239,250,259,308 | `Cancel` |
| 3 | editText | 59,11,75,196 | (room name) |
| 4 | statText (disabled) | 59,228,75,292 | `Floor: ^0` |
| 5 | statText (disabled) | 59,293,75,373 | `Suite: ^1` |
| 6 | userItem (disabled) | 56,209,78,376 | framed red-orange |
| 7 | statText (disabled) | 40,209,56,301 | `Location:` |
| 8 | statText (disabled) | 40,8,56,100 | `Room Name:` |
| 9 | statText (disabled) | 81,209,97,376 | `Number of Objects: ^2` |
| 10 | userItem (disabled) | 142,24,222,152 | source tile strip (**128×80**) |
| 11 | resCtrl | 115,8,135,168 | CNTL **128** |
| 12 | userItem (disabled) | 228,8,229,376 | 1-pixel divider, red-orange |
| 13 | picItem (disabled) | 0,0,32,384 | PICT **1007** — the title-bar art (`picFrame` 385×32) |
| 14 | statText (disabled) | 99,8,115,100 | `Tiles:` |
| 15 | userItem (disabled) | 142,232,222,360 | destination tile strip (**128×80**) |
| 16 | picItem (disabled) | 150,154,178,230 | PICT **1009** — the "drag →" arrow art (`picFrame` 76×28) |
| 17 | chkCtrl | 117,209,135,376 | `First Room` |
| 18 | statText (disabled) | 99,209,115,376 | (empty; filled at runtime) |
| 19 | btnCtrl | 194,160,214,224 | `Bounds` |

`ParamText(floorStr, suiteStr, objectsStr, "\p")` (`GliderPRO/Sources/RoomInfo.c:405`) fills
`^0`/`^1`/`^2` from `thisRoom->floor`, `thisRoom->suite`, `thisRoom->numObjects`. Note
`^0` is the **raw** floor, i.e. the signed value with 0 = ground and negatives underground,
*not* the `+kNumUndergroundFloors` biased form used in links (§17).

Both tile strips are exactly 128 × 80, matching `tileSrcRect` and
`kNumTiles (8) × kMiniTileWide (16) = 128`.

Item 13's `PICT 1007` is a **third** distinct resource with numeric id 1007, alongside
`STR# 1007` (the 144 object names, §12.17) and `DLOG 1007` (Blower Info, §12.3). Three
resource types, one number — legal on the Mac, fatal for a port that keys assets by number
alone.

## 13.4 CNTL 128 — the background popup, and why it does nothing in 1.0.4

`CNTL 128` raw bytes (23) are `0000 0000 0014 00BE 0001 0100 0012 0001 0802 0000 0000 00`,
which decode as:

| Field | Bytes | Value |
|---|---|---|
| `contrlRect` | `0000 0000 0014 00BE` | t0 l0 b20 r190 |
| `contrlValue` | `0001` | 1 |
| visible / fill | `01 00` | visible |
| `contrlMax` | `0012` | **18** |
| `contrlMin` | `0001` | 1 |
| `procID` | `0802` | **2050** = CDEF 128 × 16 + variant 2 |
| `contrlRfCon` | `0000 0000` | **0** |
| title | `00` | empty |

Compare the Tools-palette popup `CNTL 129` (`0002 0004 0016 0070 0001 0100 0003 0001 0802
0000 008D 00`): same `procID` 2050 but `contrlRfCon` = `0x0000008D` = **141**, which is the
resource ID of `MENU 141 "Tools"`.

`CDEF 128` is a 5 978-byte compiled 68k resource with no source in the tree. Disassembling
its `initCntl` handler shows the byte sequence

```
48 79 4D 45 4E 55      PEA     #'MENU'
22 51                  MOVEA.L (A1),A1          ; A1 = *ControlHandle
3F 29 00 26            MOVE.W  $0026(A1),-(SP)  ; low word of contrlRfCon (offset 0x24+2)
2D 50 FF F8            MOVE.L  A0,-8(A6)
A9 A0                  _GetResource
```

at file offset 3252, immediately before the MacsBug symbol `INITIALI` at offset 3444. So the
CDEF obtains its menu by `GetResource('MENU', LoWord(contrlRfCon))`. That is consistent with
`CNTL 129` → `MENU 141`, and means **`CNTL 128`, whose refCon is 0, has no menu**.

The only code that could give item 11 a menu is `AddMenuToPopUp`, and it is commented out in
*both* places:

* the call site, `//	AddMenuToPopUp(roomInfoDialog, kRoomPopupItem, backgroundsMenu);`
  (`GliderPRO/Sources/RoomInfo.c:434`), preceded by the author's own note
  `// Fix this later.  TEMP` at `:433`;
* the function body itself, wrapped in `/* … */` (`GliderPRO/Sources/DialogUtils.c:547`-`557`).
  Had it run it would have written a **`MenuHandle`** (a pointer) into `contrlRfCon`, not a
  resource ID — a different convention from the one the resource uses.

Consequence for a port: the Room Info background popup is drawn and can be clicked, but in
1.0.4 it cannot present a menu, so `item == kRoomPopupItem` is (almost certainly) never
returned by `ModalDialog` and the whole `else if (item == kRoomPopupItem)` arm at
`GliderPRO/Sources/RoomInfo.c:497`-`547` is dead. `DoRoomInfo` still calls
`GetMenu(kBackgroundsMenuID)` at `:397` and `EnableMenuItem(backgroundsMenu, 19)` at `:400`,
and never disposes the `MenuHandle` — a small leak. A Go port that wants a usable editor
should implement the popup properly (the arm's logic is documented in §13.8 below) rather
than reproduce the dead end. This is recorded as an open question because the CDEF's
fallback path cannot be fully verified from the binary.

`MENU 140` — the menu that *would* have been attached — parses as `menuID` **133** (not 140),
`enableFlags` `0xFFF7FFFF` (bit 19 clear ⇒ item 19 disabled at build time, re-enabled by
`EnableMenuItem` only when `HouseHasOriginalPicts()`), title `Rooms`, 19 items:

| # | Item | Background PICT |
|---:|---|---:|
| 1 | Simple Room | 2000 (`kSimpleRoom`) |
| 2 | Paneled Room | 2001 |
| 3 | Basement | 2002 |
| 4 | Child's Room | 2003 |
| 5 | Asian Room | 2004 |
| 6 | Unfinished Room | 2005 |
| 7 | Swinger's Room | 2006 |
| 8 | Bathroom | 2007 |
| 9 | Library | 2008 |
| 10 | Garden | 2009 (`kFirstOutdoorBack`) |
| 11 | Skywalk | 2010 |
| 12 | Dirt | 2011 |
| 13 | Meadow | 2012 |
| 14 | Field | 2013 |
| 15 | Roof | 2014 |
| 16 | Sky | 2015 |
| 17 | Stratosphere | 2016 |
| 18 | Stars | 2017 (`kStars`) |
| 19 | Original Artwork | (opens DLOG 1016) |

Names from `MENU 140`; IDs from `GliderPRO/Headers/GliderDefines.h:227`-`244`. The mapping is
`PICT = (item − 1) + kBaseBackgroundID` where `kBaseBackgroundID` = **2000**
(`GliderPRO/Headers/GliderDefines.h:519`), and `kNumBackgrounds` = **18**
(`GliderPRO/Headers/GliderDefines.h:521`).

## 13.5 The 128×80 thumbnail, and the four backgrounds that have a dedicated small PICT

`DoRoomInfo` builds `tileSrcMap` and fills it (`GliderPRO/Sources/RoomInfo.c:407`-`420`):

```
theErr = CreateOffScreenGWorld(&tileSrcMap, &tileSrcRect, kPreferredDepth);   // 128x80, 8-bit
SetGWorld(tileSrcMap, nil);
if ((tempBack > kStars) && (!PictIDExists(tempBack))) { BitchAboutPICTNotFound(); tempBack = kSimpleRoom; }
if ((tempBack == 2002) || (tempBack == 2011) || (tempBack == 2016) || (tempBack == 2017))
    LoadScaledGraphic(tempBack - 800, &tileSrcRect);
else
    LoadScaledGraphic(tempBack, &tileSrcRect);
```

`LoadScaledGraphic` is just `GetPicture(resID)` + `DrawPicture(pic, theRect)` + `ReleaseResource`
(`GliderPRO/Sources/Utilities.c:340`-`349`), so QuickDraw does the scaling.

The `− 800` special case is **not** an offset into a different family; it selects a
hand-drawn 128×80 replacement. Verified by parsing the `picFrame` of each PICT out of
`Glider PRO.r`:

| PICT | `picFrame` | Size | Data bytes |
|---:|---|---|---:|
| 2000 | t0 l0 b322 r512 | 512×322 | 31 838 |
| 2002 | t0 l0 b322 r512 | 512×322 | 52 328 |
| 2011 | t0 l0 b322 r512 | 512×322 | 81 036 |
| 2016 | t0 l0 b322 r512 | 512×322 | 76 176 |
| 2017 | t0 l0 b322 r512 | 512×322 | 12 624 |
| **1202** | t0 l0 b80 r128 | **128×80** | 5 826 |
| **1211** | t0 l0 b80 r128 | **128×80** | 3 328 → (5 272) |
| **1216** | t0 l0 b80 r128 | **128×80** | 3 328 |
| **1217** | t0 l0 b80 r128 | **128×80** | 2 884 |

(2002−800 = 1202, 2011−800 = 1211, 2016−800 = 1216, 2017−800 = 1217; all four exist and are
exactly 128×80, so no scaling occurs for them.) The four are Basement, Dirt, Stratosphere and
Stars — the backgrounds whose fine dither or starfield turns to mush under QuickDraw's
nearest-neighbour `DrawPicture` scale. A Go port needs the same four-way special case, or a
better downscaler plus a decision about visual fidelity.

Note the asymmetry: the same `− 800` rule is applied again at `GliderPRO/Sources/RoomInfo.c:539`
inside the popup arm, but **not** at `:514` (the Original-Artwork reload) nor at `:555` (the
Bounds-button reload). Those two paths only ever see IDs ≥ `kUserBackground` (3000), where the
rule does not apply, so it is harmless.

## 13.6 `UpdateRoomInfoDialog` (`GliderPRO/Sources/RoomInfo.c:64`)

1. `DrawDialog(theDialog)`.
2. Reassert the popup value: if `tempBack >= kUserBackground` (3000) set it to
   `kOriginalArtworkItem` (19), else to `(tempBack - kBaseBackgroundID) + 1`. So background
   2000 → item 1, 2017 → item 18, anything ≥3000 → item 19.
3. `CopyBits` the whole of `tileSrcRect` (128×80) from `tileSrcMap` into `tileSrc`
   (DITL item 10) with `srcCopy`. This is the un-sliced preview of the background.
4. `dest = tileDest; dest.right = dest.left + kMiniTileWide;` then for `i` = 0..7:
   * `QSetRect(&src, 0, 0, kMiniTileWide, 80)` — i.e. `{t0,l0,b80,r16}`;
   * `QOffsetRect(&src, tempTiles[i] * kMiniTileWide, 0)` — pick column `tempTiles[i]`;
   * `CopyBits(tileSrcMap → dialog port, &src, &dest, srcCopy, nil)`;
   * `QOffsetRect(&dest, kMiniTileWide, 0)`.

   So the destination strip is the room as actually composited: eight 16-px columns, column
   `i` showing source column `tempTiles[i]`. The identity permutation `0,1,2,3,4,5,6,7`
   reproduces the background unchanged.
5. `SetDialogString(theDialog, kLitUnlitText, "\p(Room Is Dark)")` if
   `GetNumberOfLights(thisRoomNumber) == 0`, else `"\p(Room Is Lit)"`.
   `GetNumberOfLights` is `GliderPRO/Sources/Room.c:970`.
6. Four decorations:
   * `FrameDialogItemC(theDialog, kRoomLocationBox /*6*/, kRedOrangeColor8 /*23*/)`
   * `FrameDialogItem(theDialog, kRoomTilesBox /*10*/)` — plain black
   * `FrameDialogItemC(theDialog, kRoomDividerLine /*12*/, kRedOrangeColor8)`
   * `FrameDialogItem(theDialog, kRoomTilesBox2 /*15*/)` — plain black

`FrameDialogItemC` resolves the colour with `Index2Color` on the current `GDevice` CLUT
(`GliderPRO/Sources/DialogUtils.c:706`); `kRedOrangeColor8` = **23**
(`GliderPRO/Headers/GliderDefines.h:542`), with the author's own comment
`// actually, 18`. In a Go port this must become a fixed RGB taken from the 8-bit CLUT
(`clut` resources are present: two of them), not an index.

## 13.7 Tile drag-and-drop

The tile editor is a hand-rolled drag: pick a column from the **source** strip (item 10),
drop it into a slot of the **destination** strip (item 15). There is no way to drag within
the destination strip, and no undo.

### `HiliteTileOver(Point mouseIs)` (`GliderPRO/Sources/RoomInfo.c:225`)

Called from `RoomFilter`'s `default:` arm on every null event, with `GetMouse` coordinates.

1. If `mouseIs` is inside `tileSrc`:
   1. If `cursorIs != kHandCursor`, `SetCursor(&handCursor)`; `cursorIs = kHandCursor`.
   2. `newTileOver = (mouseIs.h - tileSrc.left) / kMiniTileWide`.
   3. If `newTileOver != tileOver`: with `PenSize(1,2)` and `ForeColor(redColor)`, draw a
      16-px horizontal bar at `tileSrc.top - 3` and another at `tileSrc.bottom + 1`, both
      starting at `tileSrc.left + newTileOver*16`; then, if `tileOver != -1`, erase the old
      pair by redrawing them in `whiteColor`; `ForeColor(blackColor)`, `PenNormal()`,
      `tileOver = newTileOver`.
2. Else:
   1. If `tileOver != -1`, erase its red bars in `whiteColor` and set `tileOver = -1`.
   2. If `mouseIs` is inside `editTETextBox`, switch to `beamCursor`; otherwise `InitCursor()`
      (arrow). `cursorIs` caches the state so the trap is only called on change.

`handCursor` and `beamCursor` are `Cursor` globals loaded elsewhere; the resource fork carries
16 `CURS` and 12 `crsr` resources.

### `DragMiniTile(Point mouseIs, short *newTileOver)` (`GliderPRO/Sources/RoomInfo.c:119`)

Entered from `RoomFilter` on `mouseDown` inside `tileSrc` when `StillDown()`.

1. `tileOver = (mouseIs.h - tileSrc.left) / kMiniTileWide` — the column being dragged.
2. `wasTileOver = -1`.
3. `dragRect = {0,0,80,16}` offset to `(tileSrc.left + tileOver*16, tileSrc.top)`.
4. `PenMode(patXor)`, `PenPat(gray)`, `FrameRect(&dragRect)` — draw the XOR ghost.
5. `mouseWas = mouseIs`.
6. `while (WaitMouseUp())`:
   1. `GetMouse(&mouseIs)`.
   2. If `DeltaPoint(mouseWas, mouseIs) != 0`:
      1. `FrameRect(&dragRect)` (erase), `QOffsetRect(&dragRect, mouseIs.h - mouseWas.h, 0)`
         — **horizontal only**, `FrameRect(&dragRect)` (redraw).
      2. If `mouseIs` is in `tileDest`:
         `*newTileOver = (mouseIs.h - tileDest.left) / kMiniTileWide`; if it changed, draw a
         blue `PenSize(1,2)` 16-px bar at `tileDest.top - 3` and at `tileDest.bottom + 1`,
         erase the previous pair in white, restore `patXor`/gray, `wasTileOver = *newTileOver`.
      3. Else `*newTileOver = -1` and erase any previous blue pair.
      4. `mouseWas = mouseIs`.
7. On mouse-up, erase the last blue pair if any, then `FrameRect(&dragRect)` to remove the
   ghost, `PenNormal()`.

`RoomFilter` then commits: `if ((newTileOver >= 0) && (newTileOver < kNumTiles))
{ tempTiles[newTileOver] = tileOver; UpdateRoomInfoDialog(dial); }`
(`GliderPRO/Sources/RoomInfo.c:340`-`344`). Note the assignment direction: the *destination*
slot receives the *source* column index. `tileOver` is left pointing at the dragged source
column, which is also what `HiliteTileOver` uses as its cached state, so the red bars stay on
that column after the drop.

### `RoomFilter` (`GliderPRO/Sources/RoomInfo.c:299`)

| Event | Behaviour |
|---|---|
| `keyDown` with `kReturnKeyASCII` or `kEnterKeyASCII` | `FlashDialogButton(dial, kOkayButton)`, return item 1 |
| `keyDown` with `kEscapeKeyASCII` | `FlashDialogButton(dial, kCancelButton)`, return item 2 |
| `keyDown` with `kTabKeyASCII` | `SelectDialogItemText(dial, kRoomNameItem, 0, 1024)` and swallow |
| other `keyDown` | `return false` (Dialog Manager handles it) |
| `mouseDown` inside `tileSrc` | `if (StillDown()) DragMiniTile(...)`; always `return true` |
| `mouseDown` elsewhere | `return false` |
| `mouseUp` | `return false` |
| `updateEvt` | `BeginUpdate` / `UpdateRoomInfoDialog` / `EndUpdate`, rewrite `event->what = nullEvent`, `return false` |
| anything else (incl. null) | `GetMouse(&mouseIs); HiliteTileOver(mouseIs); return false` |

Because `mouseDown` inside `tileSrc` returns `true` with `*item` **never assigned**, the
`short item` in `DoRoomInfo` keeps whatever value it had. On the first such click `item` is
uninitialised stack; on later clicks it is the previous item hit. If that stale value happens
to be 1 or 2 the dialog will close. This is a real latent bug — a Go port should return a
sentinel (e.g. 0) instead.

## 13.8 `DoRoomInfo` control flow (`GliderPRO/Sources/RoomInfo.c:376`)

1. `GetGWorld(&wasCPort, &wasWorld)`; `roomFilterUPP = NewModalFilterUPP(RoomFilter)`.
2. `tileOver = -1`; `cursorIs = kArrowCursor`; `tempBack = thisRoom->background`.
3. `backgroundsMenu = GetMenu(kBackgroundsMenuID /*140*/)`;
   `if (HouseHasOriginalPicts()) EnableMenuItem(backgroundsMenu, kOriginalArtworkItem /*19*/)`.
   `HouseHasOriginalPicts()` is `Count1Resources('PICT') > 0` on the *house's* resource fork
   (`GliderPRO/Sources/House.c:246`).
4. `NumToString` floor, suite, `numObjects`; `ParamText(floorStr, suiteStr, objectsStr, "\p")`.
5. `CreateOffScreenGWorld(&tileSrcMap, &tileSrcRect, kPreferredDepth)`; `SetGWorld(tileSrcMap, nil)`.
6. If `tempBack > kStars` (2017) and `!PictIDExists(tempBack)`: `BitchAboutPICTNotFound()`
   (ALRT 1036) and `tempBack = kSimpleRoom` (2000). Note the test is `> kStars`, so a bogus
   *built-in* ID in 2000..2017 is not caught here.
7. `LoadScaledGraphic` with the `−800` rule of §13.5.
8. `SetGWorld(wasCPort, wasWorld)`.
9. `for i in 0..kNumTiles-1: tempTiles[i] = thisRoom->tiles[i]`.
10. `roomInfoDialog = GetNewDialog(kRoomInfoDialogID, nil, kPutInFront)`;
    `RedAlert(kErrDialogDidntLoad)` on failure; `SetPort`.
11. (`AddMenuToPopUp` would go here — commented out, see §13.4.)
12. Set the popup value from `tempBack` exactly as in step 2 of §13.6.
13. `SetDialogString(roomInfoDialog, kRoomNameItem, thisRoom->name)`.
14. `GetDialogItemRect` for items 10 → `tileSrc`, 15 → `tileDest`, 3 → `editTETextBox`.
15. `SelectDialogItemText(roomInfoDialog, kRoomNameItem, 0, 1024)` — select all of the name.
16. `ShowWindow`; `DrawDefaultButton`.
17. `wasFirstRoom = ((*thisHouse)->firstRoom == thisRoomNumber)` (under `HLock`);
    `SetDialogItemValue(roomInfoDialog, kRoomFirstCheck, (short)wasFirstRoom)`.
18. `if (tempBack >= kUserBackground) MyEnableControl(…, kBoundsButton) else MyDisableControl(…)`.
19. `leaving = false`; loop `ModalDialog(roomFilterUPP, &item)`:
    * **`item == kOkayButton` (1)**
      1. `for i in 0..7: thisRoom->tiles[i] = tempTiles[i]`.
      2. `GetDialogString(…, kRoomNameItem, tempStr)`;
         `PasStringCopyNum(tempStr, thisRoom->name, 27)` — hard truncation at **27**
         characters, matching `CheckRoomNameLength` (§18) and the 28-byte `Str27` field.
      3. `if (wasFirstRoom) { (*thisHouse)->firstRoom = thisRoomNumber; }` — note this is
         **one-way**: unchecking "First Room" never clears `firstRoom`, so the only way to
         move it is to check the box in some other room.
      4. `thisRoom->background = tempBack`.
      5. `if (tempBack < kUserBackground) lastBackground = tempBack` — remembers the last
         *built-in* background for `CreateNewRoom` (`GliderPRO/Sources/Room.c:175`).
      6. `CopyThisRoomToRoom()`; `ReflectCurrentRoom(false)`; `fileDirty = true`;
         `UpdateMenus(false)`; `leaving = true`.
    * **`item == kCancelButton` (2)** — `leaving = true`. All of `tempTiles`, `tempBack` and
      the name edit are simply discarded. Note `thisRoom->bounds` is **not** part of the
      temp set: `ChooseOriginalArt` writes it directly (see §13.9), so Cancel here does
      **not** undo a bounds change.
    * **`item == kRoomFirstCheck` (17)** — toggle `wasFirstRoom` and re-set the checkbox value.
    * **`item == kRoomPopupItem` (11)** — dead in 1.0.4 (§13.4); logic preserved for a port:
      1. `GetPopUpMenuValue(…, &newBack)`.
      2. If `newBack == kOriginalArtworkItem` (19):
         * if `tempBack < kUserBackground`, `tempBack = GetFirstPICT()` and `forceDraw = true`,
           else `forceDraw = false`;
         * `newBack = ChooseOriginalArt(tempBack)`;
         * if `(tempBack != newBack) || forceDraw`, set `tempBack = newBack`,
           `SetPort((GrafPtr)tileSrcMap)`, `LoadScaledGraphic(tempBack, &tileSrcRect)`, and
           invalidate both `tileSrc` and `tileDest`.
      3. Else: `newBack += (kBaseBackgroundID - 1)` (i.e. `+1999`), and
         `if (newBack != tempBack) SetInitialTiles(newBack, false)`.
      4. `if (newBack >= kUserBackground) { MyEnableControl(kBoundsButton);
         if (newBack != tempBack) SetInitialTiles(newBack, false); }`
         `else MyDisableControl(kBoundsButton)`.
      5. `if (newBack != tempBack) { tempBack = newBack; SetPort(tileSrcMap);` reload with the
         `−800` rule`; invalidate tileSrc and tileDest; }`.
      Because step 2 already assigned `tempBack = newBack`, steps 4 and 5 are no-ops on the
      Original-Artwork path; on the built-in path steps 3 and 5 both fire, giving one
      `SetInitialTiles` and one reload.
    * **`item == kBoundsButton` (19)**
      1. `newBack = ChooseOriginalArt(tempBack)`.
      2. If it changed: `tempBack = newBack`, `SetPort(tileSrcMap)`,
         `LoadScaledGraphic(tempBack, &tileSrcRect)`, invalidate both strips.
      This button is the **only** live route to DLOG 1016 in 1.0.4, and it is enabled only
      when the room's background is already ≥ 3000 (step 18). Rooms with a built-in
      background therefore cannot get custom bounds through the shipped UI.
20. `InitCursor()`; `DisposeDialog`; `DisposeModalFilterUPP`; `DisposeGWorld(tileSrcMap)`;
    `tileSrcMap = nil`. `backgroundsMenu` is never released.

### `SetInitialTiles(short background, Boolean doRoom)` (`GliderPRO/Sources/Room.c:40`)

Writes either `thisRoom->tiles[]` (`doRoom == true`) or `tempTiles[]` (`false`).

| Condition | tiles[0..7] |
|---|---|
| `background >= kUserBackground` (≥3000) | identity `0,1,2,3,4,5,6,7` |
| 2000 `kSimpleRoom`, 2001 `kPaneledRoom`, 2002 `kBasement`, 2003 `kChildsRoom`, 2004 `kAsianRoom`, 2005 `kUnfinishedRoom`, 2006 `kSwingersRoom`, 2007 `kBathroom`, 2008 `kLibrary` | all `1`, then `[0] = 0` and `[7] = 7` ⇒ `0,1,1,1,1,1,1,7` |
| 2010 `kSkywalk` | identity |
| 2013 `kField`, 2009 `kGarden`, 2011 `kDirt` | all `0` |
| 2012 `kMeadow` | all `1` |
| 2014 `kRoof` | all `3` |
| 2015 `kSky` | all `2` |
| 2016 `kStratosphere`, 2017 `kStars` | identity |
| anything else | **unchanged** (`default: break`) |

Empirically confirmed over all 4 070 rooms of the 22 shipped houses (decoded from the
BinHex files, tiles at room offset 36, eight big-endian `short`s):

* 1 262 distinct tile permutations are in use; the ten most common are
  `0,1,2,3,4,5,6,7` (591), all-0 (396), all-2 (295), all-4 (185), `0,1,1,1,1,1,1,7` (168),
  all-1 (79), all-7 (76), `0,2,2,2,2,2,2,2` (67), `1,1,1,1,1,1,1,7` (65),
  `2,2,2,2,2,2,2,7` (60).
* Every value is in 0..7 — never out of range.
* Per background, the most common pattern matches `SetInitialTiles`' default:
  2000 → `0,1,1,1,1,1,1,7` (53 of 171); 2002 → same (23 of 103); 2006 → same (14 of 49);
  2011 → all-0 (240 of 421); 2012 → all-1 (16 of 97); 2014 → all-3 (38 of 181);
  2015 → all-2 (264 of 595); **2016 → identity in 62 of 62 (100 %)**; 2017 → identity in
  243 of 246. User backgrounds (≥3000) are identity in only 285 of 1 797, so authors do
  re-arrange tiles for custom art.

## 13.9 "Select Original Art" — DLOG/DITL 1016

`DLOG 1016 "Select Original Art"` — `rect(t0,l0,b144,r256)` = **144 × 256**, `procID` 1,
**initially visible**, **has a go-away box** (unusual for a modal), DITL 1016. Brought up by
`BringUpDialog(&theDialog, kOriginalArtDialogID)` (`GliderPRO/Sources/DialogUtils.c:24`), which
centres it, calls `GetNewDialog`, `SetPort`, `ShowWindow` and `DrawDialog`.

| # | Type | Rect (t,l,b,r) | Text |
|---|---|---|---|
| 1 | btnCtrl | 116,190,136,248 | `Okay` |
| 2 | btnCtrl | 116,124,136,182 | `Cancel` |
| 3 | statText (dis) | 8,8,40,248 | `Enter the ID number of the PICT in your house you would like to use.` |
| 4 | statText (dis) | 51,8,67,62 | `PICT ID:` |
| 5 | editText | 51,67,67,102 | default `3000` — this is `kPICTIDItem` |
| 6 | statText (dis) | 70,10,86,102 | `(3000 - 3499)` |
| 7 | userItem (dis) | 54,139,94,149 | left edge bar → `leftBound` |
| 8 | userItem (dis) | 44,149,54,213 | top edge bar → `topBound` |
| 9 | userItem (dis) | 54,213,94,223 | right edge bar → `rightBound` |
| 10 | userItem (dis) | 94,149,104,213 | bottom edge bar → `bottomBound` |
| 11 | statText (dis) | 66,152,82,210 | `Bounded` |
| 12 | chkCtrl | 90,8,108,114 | `Floor Support` — this is `kFloorSupportCheck` |

Items 7-10 form a picture-frame: a 10-px-thick border around the word "Bounded", each side
individually clickable. Note the static text says `(3000 - 3499)` but the OK validator accepts
**3000..3799** (§13.11 step 3) and `kUserStructureRange` is **3300**
(`GliderPRO/Headers/GliderDefines.h:523`) — three different numbers for what is nominally the
same range. A port must pick the code's 3000..3799.

## 13.10 `UpdateOriginalArt` (`GliderPRO/Sources/RoomInfo.c:575`)

1. `DrawDialog(theDialog)`; `DrawDefaultButton(theDialog)`.
2. `PenSize(2, 1)`; if `!originalLeftOpen` `BorderDialogItem(theDialog, 7, 8)` (solid black),
   else set `PenPat(gray)`, `BorderDialogItem(theDialog, 7, 8)`, restore black pattern.
3. `PenSize(1, 2)`; same for item **8** with edge code **4** and `originalTopOpen`.
4. `PenSize(2, 1)`; same for item **9** with edge code **1** and `originalRightOpen`.
5. `PenSize(1, 2)`; same for item **10** with edge code **2** and `originalBottomOpen`.
6. `PenSize(1, 1)`.

`BorderDialogItem`'s edge codes are a bit mask: **1 = right, 2 = bottom, 4 = top,
8 = left**. So a *closed* wall draws a solid 2-px line on that side and an *open* wall draws
the same line in the QuickDraw gray pattern. The pen sizes are anisotropic — `(2,1)` for the
vertical left/right bars and `(1,2)` for the horizontal top/bottom bars — so all four read as
2 pixels thick.

## 13.11 `ChooseOriginalArt(short was)` (`GliderPRO/Sources/RoomInfo.c:711`)

Returns the chosen PICT ID (or `was` on Cancel). Writes `thisRoom->bounds` **directly** on OK.

1. `originalArtFilterUPP = NewModalFilterUPP(OriginalArtFilter)`.
2. `if (was < kUserBackground) was = kUserBackground;` — clamp up to 3000.
3. `InitCursor()`; `BringUpDialog(&theDialog, kOriginalArtDialogID)`.
4. `if (was >= kOriginalArtworkItem /*19*/) { newPictID = was; wasPictID = was; }
   else { newPictID = kUserBackground; wasPictID = 0; }` — after step 2 `was` is always
   ≥ 3000 > 19, so the `else` is unreachable. (The comparison against `kOriginalArtworkItem`,
   a *menu item index*, is almost certainly a copy-paste of the wrong constant.)
5. `SetDialogNumToStr(theDialog, kPICTIDItem, (long)newPictID)`;
   `SelectDialogItemText(theDialog, kPICTIDItem, 0, 16)`.
6. `GetDialogItemRect` for items 7,8,9,10 → `leftBound`, `topBound`, `rightBound`, `bottomBound`.
7. Decode the existing bounds word:
   ```
   tempShort = thisRoom->bounds >> 1;        // "version 2.0 house"
   originalLeftOpen   = ((tempShort & 1)  == 1);
   originalTopOpen    = ((tempShort & 2)  == 2);
   originalRightOpen  = ((tempShort & 4)  == 4);
   originalBottomOpen = ((tempShort & 8)  == 8);
   originalFloor      = ((tempShort & 16) == 16);
   ```
8. `SetDialogItemValue(theDialog, kFloorSupportCheck, (short)originalFloor)`.
9. Loop `ModalDialog(originalArtFilterUPP, &item)`:
   * **item 1 `kOkayButton`**
     1. `GetDialogNumFromStr(theDialog, kPICTIDItem, &longID)`.
     2. If `longID >= 3000 && longID < 3800 && PictIDExists((short)longID)`:
        * `newPictID = (short)longID`;
        * `if (newPictID != wasPictID) SetInitialTiles(tempBack, false)` — **note it passes
          `tempBack`, the file-scope global, not `newPictID`**. Both are ≥3000 in every
          reachable case so the identity branch is taken either way, but the intent is
          clearly `newPictID`;
        * recompose the bounds word:
          ```
          tempShort = 0;
          if (originalLeftOpen)   tempShort += 1;
          if (originalTopOpen)    tempShort += 2;
          if (originalRightOpen)  tempShort += 4;
          if (originalBottomOpen) tempShort += 8;
          if (originalFloor)      tempShort += 16;
          tempShort = tempShort << 1;   // shift left 1 bit
          tempShort += 1;               // flag that says original bounds used
          thisRoom->bounds = tempShort;
          ```
        * `leaving = true`.
     3. Else `SysBeep(1)` and restore the text field to `newPictID`.
   * **item 2 `kCancelButton`** — `newPictID = was; leaving = true`. The four edge toggles and
     the floor-support toggle are **not** reverted, but since they are only written into
     `thisRoom->bounds` on the OK path, Cancel does leave the room untouched.
   * **items 7 / 8 / 9 / 10** — flip `originalLeftOpen` / `originalTopOpen` /
     `originalRightOpen` / `originalBottomOpen` and call `UpdateOriginalArt(theDialog)`.
   * **item 12 `kFloorSupportCheck`** — flip `originalFloor` and
     `ToggleDialogItemValue(theDialog, kFloorSupportCheck)`.
10. `DisposeDialog`; `DisposeModalFilterUPP`; `return newPictID`.

### The `bounds` word, on disk

| Bit | Mask | Meaning |
|---:|---:|---|
| 0 | `0x0001` | 1 = "original bounds are in use"; 0 = derive openings from the background |
| 1 | `0x0002` | left wall **open** |
| 2 | `0x0004` | top **open** |
| 3 | `0x0008` | right wall **open** |
| 4 | `0x0010` | bottom **open** |
| 5 | `0x0020` | has a floor support beam |

Empirically verified across all 4 070 shipped rooms (`bounds` is the big-endian `short` at
room offset 28): **2 401 rooms have `bounds == 0`**; of the rest, **every non-zero value is
odd** (31 distinct non-zero values, range 1..63; 32 distinct values counting 0) — exactly
as the `<< 1` + `+1` recomposition
requires. No shipped room has bit 0 clear together with any higher bit set, so the encoding
is self-consistent.

`GetOriginalBounding` (`GliderPRO/Sources/Room.c:937`) and `DetermineRoomOpenings`
(`GliderPRO/Sources/Room.c:816`) are the consumers.

### `OriginalArtFilter` (`GliderPRO/Sources/RoomInfo.c:629`)

| Event | Behaviour |
|---|---|
| `keyDown` Return/Enter | flash and return item 1 |
| `keyDown` Escape | flash and return item 2 |
| `keyDown` Tab | `SelectDialogItemText(dial, kPICTIDItem, 0, 1024)`, swallow |
| other `keyDown` | `return false` |
| `mouseDown` | `GlobalToLocal`, then `PtInRect` against `leftBound`/`topBound`/`rightBound`/`bottomBound` in that order, returning item 7/8/9/10; else `return false` |
| `mouseUp` | `return false` |
| `updateEvt` | `BeginUpdate` / `UpdateOriginalArt` / `EndUpdate`, `event->what = nullEvent`, `return false` |
| default | `return false` |

Because the edge bars are `userItem`s the Dialog Manager would never report them; the filter
is what makes them clickable. This is the same trick used for the tile strips.

## 13.12 PICT existence helpers

### `PictIDExists(short theID)` (`GliderPRO/Sources/RoomInfo.c:830`)

```
thePicture = GetPicture(theID);
if (thePicture == nil) {
    thePicture = (PicHandle)GetResource('Date', theID);
    if (thePicture == nil) foundIt = false;
    else ReleaseResource((Handle)thePicture);
} else ReleaseResource((Handle)thePicture);
```

The `'Date'` fallback is Glider PRO's convention for a PICT that has been retyped so the
Finder will not show it as a picture; the same fallback appears in `LoadGraphicPlus`
(`GliderPRO/Sources/Map.c:165`) and `ReadyBackground` (`GliderPRO/Sources/Room.c:272`). A Go
port must treat resource types `'PICT'` and `'Date'` as interchangeable sources of picture
data. An older `Count1Resources('PICT')` + `Get1IndResource` scan is present but commented
out at `:856`-`:871`.

Empirically: across the 9 shipped houses that have resource forks, **every** room background
ID ≥ 3000 and **every** `kCustomPict` PICT ID ≥ 10000 resolves to a resource in that house's
own fork — zero missing. So the 1036 alert path is never exercised by the shipped content.

### `GetFirstPICT(void)` (`GliderPRO/Sources/RoomInfo.c:878`)

`Get1IndResource('PICT', 1)` then `GetResInfo` → returns that resource's ID, or **−1** if the
fork has no PICTs. `Get1IndResource` searches only the *current* resource file, which is the
house (`OpenHouseResFork` at `GliderPRO/Sources/HouseIO.c:567`). Returning −1 into `tempBack`
would make the subsequent `LoadScaledGraphic(-1, …)` call `RedAlert(kErrFailedGraphicLoad)`
and abort the program; the guard is that MENU 140 item 19 is only enabled when
`HouseHasOriginalPicts()`. With the popup inert (§13.4) this path is unreachable in 1.0.4,
but a Go port that re-enables the popup must check for −1.

### `BitchAboutPICTNotFound(void)` (`GliderPRO/Sources/RoomInfo.c:899`)

`Alert(kNoPICTFoundAlert /*1036*/, nil)`. ALRT 1036 "PICT Gone" is `rect(t40,l40,b148,r314)`
= 108 × 274, DITL 1036, stages `0x4444`. DITL 1036 = { 1: btnCtrl `Okay` (80,208,100,266);
2: statText (8,8,72,225) "Where is your PICT?  I'm going to drop back to the Simple room
background and see if that comes up."; 3: iconItem (8,234,40,266) icon **1071** }.

---

# 14. House Info — DLOG/DITL **1001**

`DoHouseInfo()` (`GliderPRO/Sources/HouseInfo.c:206`) edits the four house-wide properties
that the editor exposes: the opening banner, the finished-house message, the "No Phone" flag,
and two destructive one-way operations (lock the house, clear the high scores). Reached only
from House menu → `iHouse` ("House Info…", cmd-I) at `GliderPRO/Sources/Menu.c:474`, inside
`if (houseUnlocked)`.

The whole file is wrapped in `#ifndef COMPILEDEMO` (`GliderPRO/Sources/HouseInfo.c:42`,
`:344`), so the demo build has no House Info at all.

## 14.1 Constants (`GliderPRO/Sources/HouseInfo.c:13`-`23`)

| Name | Value |
|---|---|
| `kHouseInfoDialogID` | 1001 |
| `kBannerTextItem` | 4 |
| `kLockHouseButton` | 6 |
| `kClearScoresButton` | 9 |
| `kTrailerTextItem` | 11 |
| `kNoPhoneCheck` | 14 |
| `kBannerNCharsItem` | 15 |
| `kTrailerNCharsItem` | 16 |
| `kHouseSizeItem` | 18 |
| `kLockHouseAlert` | 1029 |
| `kZeroScoresAlert` | 1032 |

File-scope state (`GliderPRO/Sources/HouseInfo.c:32`-`35`):

| Declaration | Purpose |
|---|---|
| `Str255 banner, trailer` | working copies; **also the globals `Play.c` and `GameOver.c` read at run time** |
| `Rect houseEditText1, houseEditText2` | screen rects of items 4 and 11, for I-beam tracking |
| `short houseCursorIs` | `kArrowCursor` / `kBeamCursor` cache |
| `Boolean keyHit` | set by the filter on any ordinary keystroke; consumed on the next filter call |
| `Boolean tempPhoneBit` | working copy of `phoneBitSet` |

Note that `banner`/`trailer` are not local: they are the same `Str255` globals that
`WrapBannerAndTrailer` (`GliderPRO/Sources/HouseLegal.c:604`) word-wraps for display. Cancel
still leaves them holding the house's on-disk values because they were loaded from
`(*thisHouse)->banner`/`->trailer` on entry and are only written back on OK.

## 14.2 DITL 1001, verified from `Glider PRO.r`

`DLOG 1001 "House Info"` — `rect(t32,l68,b357,r380)` = **325 high × 312 wide**, `procID` 1,
initially invisible, no go-away box, DITL 1001, `dctb 1001 "House Info"`. `CenterDialog` is
commented out at `GliderPRO/Sources/HouseInfo.c:238`, so the dialog appears at the DLOG's
fixed screen position (32, 68) — a problem on a large display and something a Go port should
centre instead.

| # | Type | Rect (t,l,b,r) | Text |
|---|---|---|---|
| 1 | btnCtrl | 297,246,317,304 | `Okay` |
| 2 | btnCtrl | 297,175,317,233 | `Cancel` |
| 3 | statText (dis) | 240,8,256,68 | `v. ^0.^1` |
| 4 | editText | 59,11,123,301 | (banner) |
| 5 | statText (dis) | 224,8,240,180 | `Number of rooms: ^2` |
| 6 | btnCtrl | 264,8,280,100 | `Lock House` |
| 7 | userItem (dis) | 300,9,316,100 | (unused — never drawn or hit-tested) |
| 8 | picItem (dis) | 0,0,32,312 | PICT **1005** title art (`picFrame` 313×32) |
| 9 | btnCtrl | 264,108,280,200 | `Clear Scores` |
| 10 | userItem (dis) | 288,8,289,304 | 1-px divider, framed red-orange |
| 11 | editText | 149,11,213,301 | (trailer) |
| 12 | statText (dis) | 40,8,56,248 | `Opening Message:` |
| 13 | statText (dis) | 130,8,146,248 | `Finished House Message:` |
| 14 | chkCtrl | 224,188,240,304 | `No Phone` |
| 15 | statText (dis) | 40,268,56,304 | `()` — live banner char count |
| 16 | statText (dis) | 130,268,146,304 | `()` — live trailer char count |
| 17 | statText (dis) | 248,208,264,304 | `Highest Score:` |
| 18 | statText (dis) | 264,208,280,304 | (empty; total house points) |

Note the `picItem` occupies the whole 32-pixel-tall band at the top of the dialog — every
editor dialog in Glider PRO uses the same trick for its title bar (Room Info uses PICT 1007,
Go To Room uses PICT 1023). Item 8's declared rect is 312 wide but PICT 1005's `picFrame` is
**313** wide, so QuickDraw scales it down by one pixel horizontally.

`ParamText(versStr, loVers, nRoomsStr, "\p")` (`GliderPRO/Sources/HouseInfo.c:236`) fills
`^0`, `^1`, `^2`.

Item 17's label reads "Highest Score:" but item 18 is filled with
`CountTotalHousePoints()` — the *theoretical maximum*, not any recorded score. Verified: the
`SetDialogNumToStr(theDialog, kHouseSizeItem, CountTotalHousePoints())` at
`GliderPRO/Sources/HouseInfo.c:118` is the only writer of item 18. A Go port should relabel it
("Maximum Score") rather than reproduce the mislabel.

Item 7 is a `userItem` that nothing ever draws into or hit-tests; it is dead DITL.

## 14.3 Version display

```
NumToString((long)version >> 8, versStr);        // the "1's" part
NumToString((long)version % 0x0100, loVers);     // the "1/10th's" part
```
(`GliderPRO/Sources/HouseInfo.c:232`-`233`.) With `kHouseVersion` = `0x0200`
(`GliderPRO/Headers/GliderDefines.h:517`) this renders as "v. 2.0". A version of `0x0201`
would render as "v. 2.1" but `0x020A` would render "v. 2.10" — the low byte is printed in
decimal, so it is not a real fixed-point fraction. Verified empirically: **all 22 shipped
houses have `version == 0x0200`** (the big-endian `short` at file offset 0), so this is
always "v. 2.0" in practice.

## 14.4 `CountTotalHousePoints` (`GliderPRO/Sources/HouseInfo.c:51`)

```
pointTotal = (long)RealRoomNumberCount() * (long)kRoomVisitScore;      // 100 per real room
HLock(thisHouse);
for i in 0 .. (*thisHouse)->nRooms - 1:
    if rooms[i].suite != kRoomIsEmpty:
        for h in 0 .. kMaxRoomObs-1 (24):
            switch (rooms[i].objects[h].what):
                kRedClock    (0x21): += kRedClockPoints     = 100
                kBlueClock   (0x22): += kBlueClockPoints    = 300
                kYellowClock (0x23): += kYellowClockPoints  = 500
                kCuckoo      (0x24): += kCuckooClockPoints  = 1000
                kStar        (0x2C): += kStarPoints         = 5000
                kInvisBonus  (0x2B): += objects[h].data.c.points
HSetState(...)
```

Constants at `GliderPRO/Headers/GliderDefines.h:536`-`541`. Note the object scan covers **all
24 slots** including empty ones (`what == kObjectIsEmpty` = −1), which the `default:` arm
ignores, so `numObjects` is not consulted.

`data.c.points` is `bonusType.points`, at **payload offset 6** (`bonusType` =
`Point topLeft` 4 + `short length` 2 + `short points` 2 + `Boolean state` 1 +
`Boolean initial` 1; `GliderPRO/Headers/GliderStructs.h:27`-`33`), i.e. absolute object offset
`2 + 6 = 8`. Getting this wrong (using offset 4, which is `length`) yields zero for every
house — a trap I hit while verifying.

Empirically computed over all 22 shipped houses by re-implementing the routine in Python:

| House | real rooms | total points | of which `kInvisBonus` | # kInvisBonus |
|---|---:|---:|---:|---:|
| Art Museum | 109 | 58 900 | 9 000 | 18 |
| CD Demo House | 206 | 77 600 | 4 000 | 8 |
| California or Bust! | 16 | 7 900 | 400 | 4 |
| Castle o' the Air | 85 | 45 500 | 12 000 | 24 |
| Davis Station | 65 | 31 800 | 0 | 0 |
| Demo House | 45 | 12 500 | 800 | 2 |
| Empty House | 35 | 8 500 | 0 | 0 |
| Fun House | 43 | 7 900 | 0 | 0 |
| Grand Prix | 175 | 44 600 | 4 800 | 10 |
| ImagineHouse PRO II | 279 | 68 500 | 6 000 | 16 |
| In The Mirror | 97 | 30 700 | 3 600 | 10 |
| Land of Illusion | 303 | 77 700 | 3 200 | 16 |
| Leviathan | 472 | 118 100 | 10 700 | 45 |
| Metropolis | 127 | 39 300 | 2 000 | 4 |
| Nemo's Market | 124 | 52 200 | 2 100 | 13 |
| Rainbow's End | 223 | 77 500 | 2 500 | 25 |
| Sampler | 2 | 5 200 | 0 | 0 |
| Slumberland | 383 | 141 400 | 15 000 | 48 |
| SpacePods | 402 | 73 000 | 3 300 | 29 |
| Teddy World | 531 | 80 500 | 3 900 | 11 |
| The Asylum Pro | 140 | 38 000 | 6 100 | 13 |
| Titanic | 208 | 86 300 | 14 300 | 31 |

Across all houses `kInvisBonus.points` takes exactly three values: **100 (×136), 300 (×27),
500 (×164)** — always one of the three clock values, never anything else. And
`nRooms == RealRoomNumberCount()` in every shipped house (no room has
`suite == kRoomIsEmpty`), which is what `CompressHouse` + `LopOffExtraRooms` guarantee on
every save (§18).

## 14.5 `UpdateHouseInfoDialog` (`GliderPRO/Sources/HouseInfo.c:109`)

1. `DrawDialog(theDialog)`.
2. `SetDialogNumToStr(item 15, GetDialogStringLen(item 4))` — banner length.
3. `SetDialogNumToStr(item 16, GetDialogStringLen(item 11))` — trailer length.
4. `SetDialogNumToStr(item 18, CountTotalHousePoints())`.
5. `FrameDialogItemC(theDialog, 10, kRedOrangeColor8)` — the divider.
6. `SetDialogItemValue(theDialog, kNoPhoneCheck, (short)tempPhoneBit)`.

`GetDialogStringLen` returns the current TE length of an editText item
(`GliderPRO/Sources/DialogUtils.c:373`). Because the item is a plain DITL `editText` with no
length filter, nothing stops the user typing more than 255 characters; the truncation happens
only at commit time via `PasStringCopyNum(..., 255)`.

## 14.6 `HouseFilter` (`GliderPRO/Sources/HouseInfo.c:125`)

The filter is unusual in doing its work *before* the event switch:

```
if (keyHit) {
    recompute item 15 from item 4's length;
    recompute item 16 from item 11's length;
    keyHit = false;
}
```

so the character counters update one event late (the count is refreshed on the event *after*
the keystroke, which in practice is the next null event and therefore looks live).

| Event | Behaviour |
|---|---|
| `keyDown` `kEnterKeyASCII` | `FlashDialogButton(kOkayButton)`, `*item = 1`, return `true` |
| `keyDown` `kEscapeKeyASCII` | `FlashDialogButton(kCancelButton)`, `*item = 2`, return `true` |
| any other `keyDown` | `keyHit = true`, return `false` |
| `mouseDown`, `mouseUp` | `return false` |
| `updateEvt` | `SetPort`, `BeginUpdate`, `UpdateHouseInfoDialog`, `EndUpdate`, `event->what = nullEvent`, `return false` |
| default (incl. null) | `GlobalToLocal(event->where)`; I-beam inside `houseEditText1` or `houseEditText2`, arrow otherwise; `return false` |

**Return is *not* an accelerator here** — only Enter is. That is deliberate: both editText
items are multi-line message boxes (64 pixels tall, `rect` 59..123 and 149..213), so Return
must insert a line break. Compare `RoomFilter` and `OriginalArtFilter`, which accept both.
The `kOkayButton` DITL item is still the "default" button (a heavy border is drawn), so a Go
port has to reproduce this asymmetry or users will find Return does nothing.

## 14.7 `DoHouseInfo` control flow (`GliderPRO/Sources/HouseInfo.c:206`)

1. `houseFilterUPP = NewModalFilterUPP(HouseFilter)`; `tempPhoneBit = phoneBitSet`.
2. `numRooms = RealRoomNumberCount()`; `HLock(thisHouse)`;
   `PasStringCopy((*thisHouse)->banner, banner)`; same for `trailer`;
   `version = (*thisHouse)->version`.
3. `if (!noRoomAtAll) { h = rooms[firstRoom].suite; v = rooms[firstRoom].floor; }` — `h` and
   `v` are **never used afterwards**; dead code (`GliderPRO/Sources/HouseInfo.c:225`-`229`).
4. `NumToString` version hi/lo and `numRooms`; `ParamText`.
5. `GetNewDialog(kHouseInfoDialogID, nil, kPutInFront)`; `RedAlert(kErrDialogDidntLoad)` on
   failure; `SetPort`; `ShowWindow`.
6. `SetDialogString(item 4, banner)`; `SetDialogString(item 11, trailer)`;
   `SelectDialogItemText(item 4, 0, 1024)`.
7. `GetDialogItemRect(item 4) → houseEditText1`; `(item 11) → houseEditText2`;
   `houseCursorIs = kArrowCursor`.
8. Loop `ModalDialog(houseFilterUPP, &item)`:
   * **item 1 `kOkayButton`**
     1. `GetDialogString(item 4, banner)`; `GetDialogString(item 11, trailer)`.
     2. `HLock(thisHouse)`; `PasStringCopyNum(banner, (*thisHouse)->banner, 255)`;
        `PasStringCopyNum(trailer, (*thisHouse)->trailer, 255)`.
     3. If `tempPhoneBit != phoneBitSet`:
        ```
        phoneBitSet = tempPhoneBit;
        if (phoneBitSet) (*thisHouse)->flags = (*thisHouse)->flags | 0x00000002;
        else             (*thisHouse)->flags = (*thisHouse)->flags & 0xFFFFDFFD;
        ```
     4. `HSetState`; `fileDirty = true`; `UpdateMenus(false)`; `leaving = true`.
   * **item 2 `kCancelButton`** — `leaving = true`. Nothing is written.
   * **item 6 `kLockHouseButton`** — `if (WarnLockingHouse()) { changeLockStateOfHouse = true;
     saveHouseLocked = true; fileDirty = true; UpdateMenus(false); }`. The dialog stays open;
     this is **not** undone by Cancel, because it sets globals rather than dialog state.
   * **item 9 `kClearScoresButton`** — `HowToZeroScores()`. Also not undone by Cancel: it
     mutates `(*thisHouse)->highScores` immediately.
   * **item 14 `kNoPhoneCheck`** — `tempPhoneBit = !tempPhoneBit`;
     `SetDialogItemValue(item 14, (short)tempPhoneBit)`.
9. `InitCursor()`; `DisposeDialog`; `DisposeModalFilterUPP`.

### The `0xFFFFDFFD` mask

`0xFFFFDFFD` = `~0x00002002`, i.e. it clears bit 1 (the phone bit, correct) **and bit 13**
(`0x2000`, not intended). Bit 13 of `houseType.flags` is unused in 1.0.4, so the bug is
currently harmless, but a Go port must use `flags &^ 0x00000002` — reproducing the literal
would silently clear a future flag.

### `houseType.flags` bit assignments

| Bit | Mask | Global | Set at | Meaning |
|---:|---:|---|---|---|
| 0 | `0x00000001` | `wardBitSet` | `GliderPRO/Sources/HouseIO.c:416` | non-existent neighbour rooms are painted solid instead of showing through (`GliderPRO/Sources/RoomGraphics.c:195`) |
| 1 | `0x00000002` | `phoneBitSet` | `GliderPRO/Sources/HouseIO.c:417` | **set = no telephone**; `HandleTelephone` runs only `if (!phoneBitSet)` (`GliderPRO/Sources/Play.c:748`) |
| 2 | `0x00000004` | `bannerStarCountOn` | `GliderPRO/Sources/HouseIO.c:418` | `bannerStarCountOn = ((flags & 4) == 0)` — **set = star count suppressed** from the banner |

Only bit 1 has any editor UI. Bits 0 and 2 can be set on disk but no dialog reaches them; a
house author had to patch the file with ResEdit or a hex editor. Verified empirically across
all 22 shipped houses (the big-endian `long` at file offset **8**, per §3.4):

| `flags` | Count | Houses |
|---:|---:|---|
| `0x00000000` | 14 | Castle o' the Air, Demo House, Empty House, Fun House, Grand Prix, ImagineHouse PRO II, In The Mirror, Leviathan, Metropolis, Sampler, Slumberland, Teddy World, The Asylum Pro, Titanic |
| `0x00000002` | 7 | CD Demo House, California or Bust!, Davis Station, Land of Illusion, Nemo's Market, Rainbow's End, SpacePods |
| `0x00000006` | 1 | Art Museum |

So bit 0 (`wardBit`) is **never** set in any shipped house, and bit 2 is set only in Art
Museum. No shipped house has `flags` bit 13 set, which is why the mask bug was never noticed.

Also verified: `houseType.initial` (the `Point` at offset **12**, stored `v` then `h` because
QuickDraw `Point` is `{v,h}`) is in range h ∈ [30, 424], v ∈ [7, 200] across the 22 houses;
`hasGame` (offset 860) is `1` in exactly two — ImagineHouse PRO II and Titanic — meaning they
ship with a saved game in the file.

## 14.8 `WarnLockingHouse` (`GliderPRO/Sources/HouseInfo.c:307`) and ALRT 1029

`hitWhat = Alert(kLockHouseAlert /*1029*/, nil); return (hitWhat == 1);`

`ALRT 1029 "Lock House"` — `rect(t40,l40,b164,r338)` = 124 × 298, DITL 1029, stages `0xFFFF`
(every stage: beep, draw, default item 1). DITL 1029 "Lock House":

| # | Type | Rect (t,l,b,r) | Text |
|---|---|---|---|
| 1 | btnCtrl | 96,232,116,290 | `Lock It!` |
| 2 | btnCtrl | 96,150,116,224 | `Don't Lock` |
| 3 | statText (dis) | 8,8,88,234 | `Wait!!!  If you lock this house, you will not be able to edit it in the future!  You should keep an unlocked copy.  This is your only warning!` |
| 4 | iconItem (dis) | 8,258,40,290 | icon **1060** — both `ICON 1060` and `cicn 1060` exist, so the colour version is used on a colour Mac |

Locking is not applied immediately. It sets `changeLockStateOfHouse = true` and
`saveHouseLocked = true`; `WriteHouse` then does
`if (changeLockStateOfHouse) houseUnlocked = !saveHouseLocked;` and encodes the state in the
low bit of `timeStamp` (§19). Because there is no UI to *unlock*, this is a genuinely
irreversible operation from inside the program.

Verified empirically — `timeStamp & 1` over the 22 shipped houses (the big-endian signed
`long` at offset 2, always positive because `WriteHouse` masks with `0x7FFFFFFF`):

| Locked (`timeStamp` odd) | Unlocked (even) |
|---|---|
| Art Museum, CD Demo House, Davis Station, Demo House, Grand Prix, ImagineHouse PRO II, In The Mirror, Leviathan, Metropolis, Nemo's Market, Rainbow's End, Slumberland, SpacePods, Teddy World, The Asylum Pro (15) | California or Bust!, Castle o' the Air, Empty House, Fun House, Land of Illusion, Sampler, Titanic (7) |

## 14.9 `HowToZeroScores` (`GliderPRO/Sources/HouseInfo.c:319`) and ALRT 1032

`ALRT 1032 "Clear Scores"` — `rect(t40,l40,b129,r320)` = 89 × 280, DITL 1032, stages `0x4444`
(no beep, draw, default item **4** — which is a `statText`, so the Alert's "default item"
is not a button at all and Return/Enter does nothing useful). DITL 1032 is named
"High Scores":

| # | Type | Rect (t,l,b,r) | Text |
|---|---|---|---|
| 1 | btnCtrl | 61,214,81,272 | `Cancel` |
| 2 | btnCtrl | 61,134,81,206 | `Clear All` |
| 3 | btnCtrl | 61,54,81,126 | `All But #1` |
| 4 | statText (dis) | 8,8,56,232 | `Do what?  You can clear all but the highest score, all the scores, or cancel this operation.` |
| 5 | iconItem (dis) | 8,240,40,272 | icon **910** (`ICON 910` + `cicn 910`) |

Dispatch:

| `Alert` result | Action |
|---:|---|
| 1 | (Cancel) nothing |
| 2 | `ZeroHighScores(); fileDirty = true; UpdateMenus(false);` |
| 3 | `ZeroAllButHighestScore(); fileDirty = true; UpdateMenus(false);` |

`ZeroHighScores` (`GliderPRO/Sources/HighScores.c:302`):
`PasStringCopy(thisHouseName, highScores.banner)` — **it overwrites the high-score banner with
the house's file name** — then for `i` = 0..`kMaxScores-1` (9):
`names[i] = "\p--------------"` (14 hyphens), `scores[i] = 0`, `timeStamps[i] = 0`,
`levels[i] = 0`.

`ZeroAllButHighestScore` (`GliderPRO/Sources/HighScores.c:347`): identical but the loop starts
at `i = 1` and it does not touch the banner. It assumes slot 0 holds the highest score, which
is only true after `SortHighScores` (`GliderPRO/Sources/HighScores.c:280`) has run — and
nothing calls `SortHighScores` from this path. If the on-disk list is unsorted, "All But #1"
preserves whatever is in slot 0, not the actual maximum.

`scoresType` layout (`GliderPRO/Headers/GliderStructs.h:107`-`114`), at house offset **528**,
total **292** bytes:

| Field | Type | Size | Offset in `scoresType` |
|---|---|---:|---:|
| `banner` | `Str31` | 32 | 0 |
| `names[10]` | `Str15` | 160 | 32 |
| `scores[10]` | `long` | 40 | 192 |
| `timeStamps[10]` | `unsigned long` | 40 | 232 |
| `levels[10]` | `short` | 20 | 272 |

`kMaxScores` = **10** (`GliderPRO/Headers/GliderDefines.h:249`).

---

# 15. The Map window

`GliderPRO/Sources/Map.c` (796 lines) is the whole map window. It is a resizable "windoid"
showing a scrollable grid of room thumbnails over a fixed 128 × 64 virtual sheet, with the
current room outlined in red. Clicking an existing room navigates to it; clicking an empty
cell offers to create a room there. Every function in the file is inside
`#ifndef COMPILEDEMO` except `LoadGraphicPlus`, `UpdateMapWindow`, `ResizeMapWindow`,
`OpenMapWindow`, `CloseMapWindow`, `ToggleMapWindow`, `HandleMapClick` and `MoveRoom`, which
are `#ifndef`-guarded internally instead.

## 15.1 Constants

| Name | Value | Where |
|---|---:|---|
| `kMapRoomsHigh` | 9 (`// was 7`) | `GliderPRO/Sources/Map.c:17` |
| `kMapRoomsWide` | 9 (`// was 7`) | `GliderPRO/Sources/Map.c:18` |
| `kMapScrollBarWidth` | 16 | `GliderPRO/Sources/Map.c:19` |
| `kHScrollRef` | `5L` | `GliderPRO/Sources/Map.c:20` |
| `kVScrollRef` | `27L` | `GliderPRO/Sources/Map.c:21` |
| `kMapGroundValue` | **56** | `GliderPRO/Sources/Map.c:22` |
| `kNewRoomAlert` | 1004 | `GliderPRO/Sources/Map.c:23` |
| `kYesDoNewRoom` | 1 | `GliderPRO/Sources/Map.c:24` |
| `kThumbnailPictID` | 1010 | `GliderPRO/Sources/Map.c:25` |
| `kMapRoomHeight` | **20** | `GliderPRO/Headers/GliderDefines.h:246` |
| `kMapRoomWidth` | **32** | `GliderPRO/Headers/GliderDefines.h:247` |
| `kMaxNumRoomsH` | **128** | `GliderPRO/Headers/GliderDefines.h:543` |
| `kMaxNumRoomsV` | **64** | `GliderPRO/Headers/GliderDefines.h:544` |
| `kNumBackgrounds` | 18 | `GliderPRO/Headers/GliderDefines.h:521` |
| `kWindoidGrowWDEF` | 2064 = WDEF **129** variant 0 | `GliderPRO/Headers/GliderDefines.h:532` |
| `scrollBarProc` | 16 (Toolbox) | — |

`kHScrollRef` = 5 and `kVScrollRef` = 27 are arbitrary tags stored in each scroll bar's
`contrlRfCon` so `HandleMapClick` can tell them apart
(`GliderPRO/Sources/Map.c:657`, `:679`). Any two distinct values would do.

## 15.2 Globals (`GliderPRO/Sources/Map.c:36`-`43`)

| Declaration | Purpose |
|---|---|
| `Rect nailSrcRect` | `{0,0,380,32}` — the thumbnail strip's bounds |
| `Rect activeRoomRect` | window-local rect of the current room's cell (already outset by 1) |
| `Rect wasActiveRoomRect` | the previous `activeRoomRect`, so both can be invalidated |
| `Rect mapHScrollRect, mapVScrollRect` | the two scroll bars' rects |
| `Rect mapCenterRect` | 16 × 16 at the bottom-right — **written once and never read** |
| `Rect mapWindowRect` | `{0,0,H,W}` content rect |
| `GWorldPtr nailSrcMap` | 8-bit GWorld holding PICT 1010 |
| `WindowPtr mapWindow` | `nil` when closed; doubles as the "is open" test |
| `ControlHandle mapHScroll, mapVScroll` | the two scroll bars |
| `short isMapH, isMapV` | saved window position (prefs) |
| `short mapRoomsHigh, mapRoomsWide` | current grid size in cells |
| `short mapLeftRoom, mapTopRoom` | scroll position, in cells, into the 128 × 64 sheet |
| `Boolean isMapOpen` | prefs flag |
| `Boolean doPrettyMap` | prefs flag — draw real custom art in thumbnails |

`mapCenterRect` is set at `GliderPRO/Sources/Map.c:415` to a 16 × 16 rect at
`(mapWindowRect.right + 2, mapWindowRect.bottom + 2)` — evidently intended as a
"centre on current room" hot spot in the grow-box corner. Nothing hit-tests it; grepping the
whole tree finds no other reference. Dead code.

## 15.3 The coordinate transform

The virtual sheet is **128 suites wide × 64 map rows tall**. A room's cell is

```
h (column) = thisRoom->suite
v (row)    = kMapGroundValue - thisRoom->floor        // 56 - floor
```

(`GliderPRO/Sources/Map.c:57`-`58`). The *inverse* transform, `floor = kMapGroundValue - row`,
appears at `:78` (`CenterMapOnRoom`), `:139` and `:224` (the two thumbnail loops), `:605`
(`HandleMapClick`) and `:781` (`MoveRoom`); `:198` uses it to place the ground line
(`groundLevel = kMapGroundValue - mapTopRoom`). Inverting, `floor = 56 - v`, so:

| `floor` | map row `v` |
|---:|---:|
| 56 | 0 (top of sheet) |
| 1 | 55 |
| 0 (ground) | 56 |
| −7 (deepest legal) | 63 (bottom of sheet) |

That is exactly why `kMapGroundValue` is 56 and `kMaxNumRoomsV` is 64: the sheet holds floors
+56 down to −7, matching `ValidateRoomNumbers`' legal window of `floor ∈ [−7, 56]`
(`GliderPRO/Sources/HouseLegal.c:785`, the test itself at `:804`-`:805`; §18.10) and
`kNumUndergroundFloors` = 8
(`GliderPRO/Headers/GliderDefines.h:535` — floors 0 through −7 inclusive).
`kRoomsTimesSuites` = **8192** = 64 × 128 in `CheckDuplicateFloorSuite`
(`GliderPRO/Sources/HouseLegal.c:648`) is the same sheet as a bitmap.

Empirically verified over all 4 070 rooms of the 22 shipped houses: `floor` ranges
**−7 .. 39** (⇒ map row 17 .. 63) and `suite` ranges **0 .. 127**. Nothing in the shipped
content approaches floor 56, and nothing is out of range.

### `ThisRoomVisibleOnMap` (`GliderPRO/Sources/Map.c:53`)

```
h = thisRoom->suite;  v = kMapGroundValue - thisRoom->floor;
return !(h < mapLeftRoom || v < mapTopRoom ||
         h >= mapLeftRoom + mapRoomsWide || v >= mapTopRoom + mapRoomsHigh);
```

### `CenterMapOnRoom(short h, short v)` (`GliderPRO/Sources/Map.c:72`)

Takes `h` = suite and `v` = **floor** (not row).

1. Return immediately if `mapWindow == nil`.
2. `mapLeftRoom = h - (mapRoomsWide / 2)`.
3. `mapTopRoom = (kMapGroundValue - v) - (mapRoomsHigh / 2)`.
4. Clamp `mapLeftRoom` to `[0, kMaxNumRoomsH - mapRoomsWide]` = `[0, 128 - cols]`.
5. Clamp `mapTopRoom` to `[0, kMaxNumRoomsV - mapRoomsHigh]` = `[0, 64 - rows]`.
6. `SetControlValue(mapHScroll, mapLeftRoom)`; `SetControlValue(mapVScroll, mapTopRoom)`.

With the default 9 × 9 grid, `mapRoomsWide / 2` = 4, so the target room lands in column 4 and
row 4 — the exact centre of a 9 × 9 grid. With an even grid size it is one cell left/up of
centre.

Callers: `ReflectCurrentRoom` (`GliderPRO/Sources/Room.c:316` with the hard-coded
`CenterMapOnRoom(64, 1)` when `noRoomAtAll || !houseUnlocked`, and `:323` with the real room)
and `OpenMapWindow` (`GliderPRO/Sources/Map.c:417`). The `(64, 1)` fallback centres the sheet
horizontally (suite 64 of 128) at floor 1, which is why an empty/locked house shows the map
scrolled to the middle-ish with the ground line just below centre.

`InitializeEmptyHouse` sets `mapLeftRoom = 60; mapTopRoom = 50;`
(`GliderPRO/Sources/House.c:148`-`149`) for a brand-new house, i.e. suites 60..68 and rows
50..58 = floors 6 down to −2.

## 15.4 The thumbnail strip — PICT 1010

`CreateNailOffscreen` (`GliderPRO/Sources/Map.c:733`):

```
QSetRect(&nailSrcRect, 0, 0, kMapRoomWidth, kMapRoomHeight * (kNumBackgrounds + 1));
                          // = {top 0, left 0, bottom 380, right 32}
CreateOffScreenGWorld(&nailSrcMap, &nailSrcRect, kPreferredDepth);   // 8-bit
SetGWorld(nailSrcMap, nil);
LoadGraphic(kThumbnailPictID /*1010*/);
```

Note the `QSetRect(r, left, top, right, bottom)` argument order
(`GliderPRO/Sources/RectUtils.c`), so this is a **32 wide × 380 tall** strip of
`kNumBackgrounds + 1 = 19` stacked 32 × 20 cells.

Verified against the resource: `PICT 1010`'s `picFrame` is `(t0,l0,b380,r32)` — 32 × 380,
6 850 bytes — exactly 19 rows of 32 × 20. Rows 0..17 are the 18 built-in backgrounds
2000..2017 in order; **row 18 is the "?" thumbnail** used for custom artwork when
`doPrettyMap` is off.

`KillNailOffscreen` (`GliderPRO/Sources/Map.c:756`) exists but **nothing calls it** —
grepping the whole tree finds only its prototype at `:34` and its definition. So the 32 × 380
8-bit GWorld leaks the first time the map is opened and is never freed, not even by
`CloseMapWindow` (which only calls `CloseThisWindow(&mapWindow)`). Because
`CreateNailOffscreen` is guarded by `if (nailSrcMap == nil)` the leak is bounded at one
GWorld for the life of the process, so it is a wart rather than a bug — but a Go port should
still free it on close.

## 15.5 `RedrawMapContents` (`GliderPRO/Sources/Map.c:185`)

1. Return if `mapWindow == nil`. `activeRoomVisible = false`.
2. `groundLevel = kMapGroundValue - mapTopRoom` — the map row at which floor 0 sits.
3. Build the clip:
   ```
   newClip.left   = mapWindowRect.left;                                  // 0
   newClip.top    = mapWindowRect.top;                                   // 0
   newClip.right  = mapWindowRect.right  + 2 - kMapScrollBarWidth;       // cols*32
   newClip.bottom = mapWindowRect.bottom + 2 - kMapScrollBarWidth;       // rows*20
   ```
   `SetPort(mapWindow)`; `wasClip = NewRgn(); GetClip(wasClip); ClipRect(&newClip)`.
   (`mapWindowRect.right` is `cols*32 + 16 - 2`, so `+2 - 16` cancels back to `cols*32`.)
4. `HLock(thisHouse)`. For `i` = 0..`mapRoomsHigh-1`, `h` = 0..`mapRoomsWide-1`:
   1. `QSetRect(&aRoom, 0, 0, kMapRoomWidth, kMapRoomHeight)`;
      `QOffsetRect(&aRoom, kMapRoomWidth * h, kMapRoomHeight * i)`.
   2. `suite = h + mapLeftRoom`; `floor = kMapGroundValue - (i + mapTopRoom)`.
   3. If `RoomExists(suite, floor, &whoCares) && houseUnlocked`:
      1. `PenNormal()`; `type = (*thisHouse)->rooms[whoCares].background - kBaseBackgroundID`.
      2. `if (type > kNumBackgrounds) { if (!doPrettyMap) type = kNumBackgrounds; }`
         — i.e. custom art collapses to row 18 unless "pretty map" is on.
      3. `ForeColor(blackColor)`.
      4. If `type > kNumBackgrounds` (still, so `doPrettyMap` is on and the background is
         custom): `LoadGraphicPlus(type + kBaseBackgroundID, &aRoom)` — draw the *full-size*
         512 × 322 house PICT scaled down into a 32 × 20 cell, live, every redraw.
      5. Else: `QSetRect(&src, 0, 0, kMapRoomWidth, kMapRoomHeight)`;
         `QOffsetRect(&src, 0, type * kMapRoomHeight)`;
         `CopyBits(nailSrcMap → mapWindow, &src, &aRoom, srcCopy, nil)`.
      6. If `whoCares == thisRoomNumber`: `activeRoomRect = aRoom; activeRoomVisible = true`.
   4. Else (no room, or house locked): `PenPat(GetQDGlobalsGray(&dummyPat))`;
      `ForeColor(i >= groundLevel ? greenColor : blueColor)`; `PaintRect(&aRoom)`.
5. `HSetState(thisHouse)`; `ForeColor(blackColor)`; `PenNormal()`.
6. Grid lines: for `i` = 1..`mapRoomsWide-1`, `MoveTo(i*32, 0); Line(0, mapRoomsHigh*20)`;
   for `i` = 1..`mapRoomsHigh-1`, `MoveTo(0, i*20); Line(mapRoomsWide*32, 0)`.
7. If `activeRoomVisible`: `ForeColor(redColor)`; `activeRoomRect.right++`;
   `activeRoomRect.bottom++`; `FrameRect`; `InsetRect(±1)`; `FrameRect` again;
   `ForeColor(blackColor)`; `InsetRect(-1,-1)`. Two nested 1-px red rectangles = a 2-px red
   outline, and `activeRoomRect` is left outset by 1 for `FlagMapRoomsForUpdate`.
8. `SetClip(wasClip)`; `DisposeRgn(wasClip)`.

### The green / blue "no room" wash

`i >= groundLevel` ⟺ `i + mapTopRoom >= 56` ⟺ `floor <= 0`. So **floors 0 and below are
gray-dithered green (earth) and floors 1 and up are gray-dithered blue (sky)**. The
`PenPat(gray)` plus `PaintRect` gives a 50 % checkerboard in the current foreground colour on
white — an 8-bit-indexed-colour trick. In a Go port this is two solid colours at 50 % alpha,
or an actual 2×2 dither if you want the original look.

### `LoadGraphicPlus(short resID, Rect *theRect)` (`GliderPRO/Sources/Map.c:165`)

```
thePicture = GetPicture(resID);
if (thePicture == nil) {
    thePicture = (PicHandle)GetResource('Date', resID);
    if (thePicture == nil) return;          // silently draw nothing
}
DrawPicture(thePicture, theRect);
ReleaseResource((Handle)thePicture);
```

Same `'PICT'`-then-`'Date'` fallback as `PictIDExists` (§13.12), but unlike `LoadGraphic` it
returns quietly instead of calling `RedAlert` — which is what makes `doPrettyMap` safe on a
house with dangling PICT IDs.

`doPrettyMap` is a preference: default **false** on first run
(`GliderPRO/Sources/Main.c:187`) but **true** in `SetAllDefaults`
(`GliderPRO/Sources/Settings.c:1230`), and editable in the Display Preferences dialog
(`GliderPRO/Sources/Settings.c:246`, `:276`). With it on, each visible custom-art cell decodes
and scales a full 512 × 322 PackBits PICT on **every** redraw — that is up to 81 cells in the
default grid, which is why it defaults off. A Go port should cache scaled thumbnails.

## 15.6 `FindNewActiveRoomRect` and `FlagMapRoomsForUpdate`

`FindNewActiveRoomRect` (`GliderPRO/Sources/Map.c:115`) is `RedrawMapContents`' cell loop with
all drawing removed: it walks the same `mapRoomsHigh × mapRoomsWide` grid, and when
`whoCares == thisRoomNumber` it does

```
wasActiveRoomRect = activeRoomRect;
activeRoomRect    = aRoom;
activeRoomVisible = true;
```

then afterwards, if visible, `activeRoomRect.right++; activeRoomRect.bottom++;
InsetRect(&activeRoomRect, -1, -1);` — i.e. the same 1-px outset that `RedrawMapContents`
leaves behind, so the invalidated rect covers the 2-px red frame.

`FlagMapRoomsForUpdate` (`GliderPRO/Sources/Map.c:99`) simply
`InvalWindowRect(mapWindow, &wasActiveRoomRect)` then
`InvalWindowRect(mapWindow, &activeRoomRect)`.

`ReflectCurrentRoom(Boolean forceMapRedraw)` (`GliderPRO/Sources/Room.c:308`) decides between
the cheap and expensive paths:

```
if (theMode != kEditMode) return;
if (noRoomAtAll || !houseUnlocked) { CenterMapOnRoom(64, 1); UpdateMapWindow(); }
else if (!ThisRoomVisibleOnMap() || forceMapRedraw)
     { CenterMapOnRoom(thisRoom->suite, thisRoom->floor); UpdateMapWindow(); }
else { FindNewActiveRoomRect(); FlagMapRoomsForUpdate(); }
GenerateRetroLinks();
UpdateEditWindowTitle();
ReadyBackground(thisRoom->background, thisRoom->tiles);
GetThisRoomsObjRects();
DrawThisRoomsObjects();
InvalWindowRect(mainWindow, &mainWindowRect);
```

This is the single funnel every editor navigation goes through, so a Go port can treat
`ReflectCurrentRoom` as "the room changed; refresh everything".

`UpdateMapWindow` (`GliderPRO/Sources/Map.c:302`): `SetControlValue` both scroll bars from
`mapLeftRoom`/`mapTopRoom`, `SetPortWindowPort`, `DrawControls`, `DrawGrowIcon`,
`RedrawMapContents`.

## 15.7 Window geometry and resizing

`OpenMapWindow` (`GliderPRO/Sources/Map.c:362`), when `mapWindow == nil`:

1. `CreateNailOffscreen()`.
2. ```
   QSetRect(&mapWindowRect, 0, 0,
            mapRoomsWide * kMapRoomWidth  + kMapScrollBarWidth - 2,
            mapRoomsHigh * kMapRoomHeight + kMapScrollBarWidth - 2);
   ```
   i.e. content size = `cols*32 + 14` by `rows*20 + 14`. With the default 9 × 9 that is
   **302 × 194**.
3. `mapWindow = NewCWindow(nil, &mapWindowRect, "\pMap", false, kWindoidGrowWDEF /*2064*/,
   kPutInFront, true, 0L)`; `RedAlert(kErrNoMemory)` on failure. `NewCWindow` unconditionally
   — unlike the Tools palette, there is no `thisMac.hasColor` fallback to `NewWindow`.
4. `MoveWindow(mapWindow, isMapH, isMapV, true)`. First-run defaults: `isMapH = 3`,
   `isMapV = 100` (`GliderPRO/Sources/Main.c:160`-`162`, with the intended
   `qd.screenBits.bounds.bottom - 100` commented out).
5. `QSetRect(&wasActiveRoomRect, 0, 0, 1, 1)`; same for `activeRoomRect`.
6. `BringToFront`; `ShowHide(mapWindow, true)`; `HiliteAllWindows()`. (The
   `FlagWindowFloating(mapWindow)` call is commented out with the note
   `TEMP - use flaoting windows`, so the map is a normal window that can be brought in front
   of the edit window.)
7. `SetPort((GrafPtr)mapWindow)`; **`SetOrigin(1, 1)`**.
8. ```
   QSetRect(&mapHScrollRect, -1, rows*20, cols*32 + 1, rows*20 + 16);
   QSetRect(&mapVScrollRect, cols*32, -1, cols*32 + 16, rows*20 + 1);
   ```
   (again `QSetRect(r, left, top, right, bottom)`).
9. `mapHScroll = NewControl(mapWindow, &mapHScrollRect, "\p", true, mapLeftRoom, 0,
   kMaxNumRoomsH - mapRoomsWide, scrollBarProc, kHScrollRef)`;
   `mapVScroll = NewControl(..., mapTopRoom, 0, kMaxNumRoomsV - mapRoomsHigh, scrollBarProc,
   kVScrollRef)`. So the scroll ranges are `[0, 128-cols]` and `[0, 64-rows]` — the control
   *is* the scroll state; `mapLeftRoom`/`mapTopRoom` are mirrors.
10. `mapCenterRect` (dead, §15.2).
11. `CenterMapOnRoom(thisRoom->suite, thisRoom->floor)`.
12. `UpdateMapCheckmark(true)`.

`SetOrigin(1, 1)` shifts the port so that window pixel (0,0) has port coordinate (1,1). That
is what makes the top and left grid lines visible: cell rects start at port x = 0, which is
one pixel outside the window, so the window's leftmost visible column is port x = 1 — the
first pixel of cell 0's interior. `HandleMapClick` undoes it with `wherePt.h -= 1;
wherePt.v -= 1;` after `GlobalToLocal`.

`ResizeMapWindow(short newH, short newV)` (`GliderPRO/Sources/Map.c:325`), called from
`Events.c:116` after `GrowWindow`:

1. `if ((newH == 0) && (newV == 0)) return;` — user cancelled the drag.
2. `mapRoomsWide = newH / kMapRoomWidth`; clamp to a minimum of **3**.
3. `mapRoomsHigh = newV / kMapRoomHeight`; clamp to a minimum of **3**.
4. Recompute `mapWindowRect` with the same `+16-2` formula; `EraseRect(&mapWindowRect)`;
   `SizeWindow(mapWindow, mapWindowRect.right, mapWindowRect.bottom, true)`.
5. `SetControlMaximum(mapHScroll, kMaxNumRoomsH - mapRoomsWide)`;
   `MoveControl(mapHScroll, 0, mapWindowRect.bottom - 16 + 2)`;
   `SizeControl(mapHScroll, mapWindowRect.right - 16 + 3, 16)`;
   `mapLeftRoom = GetControlValue(mapHScroll)` — reading it back picks up any clamping the
   Control Manager did when the maximum shrank.
6. Same for the vertical bar: `SetControlMaximum(64 - mapRoomsHigh)`;
   `MoveControl(mapWindowRect.right - 16 + 2, 0)`;
   `SizeControl(16, mapWindowRect.bottom - 16 + 3)`; `mapTopRoom = GetControlValue(...)`.
7. `InvalWindowRect(mapWindow, &mapWindowRect)`.

The window snaps to whole cells (the size is always `cols*32+14` × `rows*20+14`), and the
minimum is 3 × 3 = 110 × 74. There is **no maximum**: growing past 128 columns would make
`SetControlMaximum` negative. In practice the screen is the limit — 128 × 32 = 4096 px wide.

`ToggleMapWindow` (`GliderPRO/Sources/Map.c:438`) sets `isMapOpen = true` when opening and
`false` when closing — correct, unlike the Tools palette's `ToggleToolsWindow`, which sets
`isToolsOpen = true` in **both** branches (`GliderPRO/Sources/Tools.c:357`, `:362`; see §5.1).

## 15.8 Live scrolling — `LiveHScrollAction` / `LiveVScrollAction`

`GliderPRO/Sources/Map.c:457` and `:514`. Installed as `ControlActionUPP`s so the map redraws
continuously while an arrow or page region is held.

| `thePart` | Delta (H) | Delta (V) |
|---|---|---|
| `kControlUpButtonPart` | −1 | −1 |
| `kControlDownButtonPart` | +1 | +1 |
| `kControlPageUpPart` | `-(mapRoomsWide / 2)` | `-(mapRoomsHigh / 2)` |
| `kControlPageDownPart` | `+(mapRoomsWide / 2)` | `+(mapRoomsHigh / 2)` |
| `kControlIndicatorPart` | (nothing — handled by `TrackControl` with a `nil` action) | same |

Each arm is the identical five-line idiom:

```
wasValue = GetControlValue(theControl);
SetControlValue(theControl, wasValue ± delta);      // Control Mgr clamps to [min,max]
if (GetControlValue(theControl) != wasValue) {
    mapLeftRoom /* or mapTopRoom */ = GetControlValue(theControl);
    RedrawMapContents();
}
```

Page scroll is **half a screen**, not a full screen — 4 cells for the default 9-cell grid.

## 15.9 `HandleMapClick(EventRecord *theEvent)` (`GliderPRO/Sources/Map.c:570`)

1. `wherePt = theEvent->where`.
2. `scrollHActionUPP = NewControlActionUPP(LiveHScrollAction)`;
   `scrollVActionUPP = NewControlActionUPP(LiveVScrollAction)`.
3. `if (mapWindow == nil) return;` — **this leaks both UPPs**, because the disposal is at the
   end of the function. Minor, and unreachable in practice (`Events.c` only dispatches here
   when `whichWindow == mapWindow`).
4. `SetPortWindowPort(mapWindow)`; `globalWhere = wherePt`; `GlobalToLocal(&wherePt)`;
   `wherePt.h -= 1; wherePt.v -= 1;` (undo `SetOrigin(1,1)`).
5. `whichPart = FindControl(wherePt, mapWindow, &whichControl)`.
6. **`whichPart == 0` — content click:**
   1. `localH = wherePt.h / kMapRoomWidth`; `localV = wherePt.v / kMapRoomHeight`.
   2. `if ((localH >= mapRoomsWide) || (localV >= mapRoomsHigh)) return;` — again leaking the
      two UPPs. Note there is **no negative check**: C integer division truncates toward
      zero, so a click at port x = −1 (possible in the 1-px margin left by `SetOrigin`) gives
      `localH == 0`, not −1. Harmless.
   3. `roomH = localH + mapLeftRoom`; `roomV = kMapGroundValue - (localV + mapTopRoom)`.
   4. If `RoomExists(roomH, roomV, &itsNumber)`:
      * `CopyRoomToThisRoom(itsNumber)`; `DeselectObject()`; `ReflectCurrentRoom(false)`.
      * Then, `if (thisMac.hasDrag)`: `SetPortWindowPort(mainWindow)`;
        `QSetRect(&aRoom, 0, 0, kMapRoomWidth, kMapRoomHeight)`;
        `CenterRectOnPoint(&aRoom, globalWhere)`; and the `DragRoom` call is **commented out**
        with `// TEMP disabled.` (`GliderPRO/Sources/Map.c:618`-`620`). So the room-dragging
        feature is dead but its setup code still runs.
   5. Else (empty cell):
      * If `doBitchDialogs`: `if (QueryNewRoom())` → `CreateNewRoom(roomH, roomV)`;
        on failure `YellowAlert(kYellowUnaccounted, 11)` and return; on success
        `DeselectObject(); ReflectCurrentRoom(false)`. If the user cancels, `return`.
      * Else (`doBitchDialogs` off): the same `CreateNewRoom` block with no confirmation.
7. **`whichPart != 0` — scroll bar click:** `controlRef = GetControlReference(whichControl)`.
   * `controlRef == kHScrollRef` (5): for `kControlUpButtonPart`, `kControlDownButtonPart`,
     `kControlPageUpPart`, `kControlPageDownPart` → `TrackControl(whichControl, wherePt,
     scrollHActionUPP)`; for `kControlIndicatorPart` → `TrackControl(..., nil)` then
     `mapLeftRoom = GetControlValue(whichControl); RedrawMapContents();`.
   * `controlRef == kVScrollRef` (27): identical with `scrollVActionUPP` and `mapTopRoom`.
8. `DisposeControlActionUPP` on both.

Note the map has **no double-click handling** and no context menu: a single click both
navigates and (via the dead `DragRoom`) would have started a drag. There is also no way to
delete a room from the map — that is House menu → Clear / Cut (§16.4).

### `QueryNewRoom` (`GliderPRO/Sources/Map.c:711`) and ALRT 1004

`hitWhat = Alert(kNewRoomAlert /*1004*/, nil); return (hitWhat == kYesDoNewRoom /*1*/);`

`ALRT 1004 "New Room?"` — `rect(t40,l40,b136,r328)` = 96 × 288, DITL 1004, stages `0x5555`
(no beep in any stage, draw, default item 1). DITL 1004 "New Room?":

| # | Type | Rect (t,l,b,r) | Text |
|---|---|---|---|
| 1 | btnCtrl | 68,222,88,280 | `Create` |
| 2 | btnCtrl | 68,156,88,214 | `Cancel` |
| 3 | statText (dis) | 13,14,50,212 | `Do you wish to create a New Room at this location?` |
| 4 | iconItem (dis) | 13,248,45,280 | icon **1004** (raw bytes `03 EC`; both `ICON 1004` and `cicn 1004` exist) |

`CenterAlert(kNewRoomAlert)` is commented out at `GliderPRO/Sources/Map.c:715`, so the alert
appears at the ALRT's fixed (40, 40).

## 15.10 `MoveRoom(Point wherePt)` — dead code (`GliderPRO/Sources/Map.c:769`)

```
localH = wherePt.h / kMapRoomWidth;
localV = wherePt.v / kMapRoomHeight;
if ((localH >= mapRoomsWide) || (localV >= mapRoomsHigh)) return;
roomH = localH + mapLeftRoom;
roomV = kMapGroundValue - (localV + mapTopRoom);
if (RoomExists(roomH, roomV, &itsNumber)) { }        // empty: refuse to overwrite
else {
    thisRoom->floor = roomV;
    thisRoom->suite = roomH;
    fileDirty = true;
    UpdateMenus(false);
    RedrawMapContents();
}
```

The only reference anywhere in the tree is the commented-out `// MoveRoom(dragPoint);` at
`GliderPRO/Sources/Scrap.c:463`, inside the block-commented `DragRoom`. So **1.0.4 has no way
to relocate a room** once created; the author's intended drag-and-drop on the map never
shipped.

Note also what it *doesn't* do: it writes only `thisRoom` (the in-memory working copy), never
`CopyThisRoomToRoom()`, so even if it were called the move would be lost on the next
navigation. And it does not fix up any link `where` fields, which encode
`MergeFloorSuite(floor + 8, suite)` (§17) — moving a room would silently break every link
pointing into it. A Go port that adds room relocation must (a) commit the room, and (b) rewrite
every `data.d.where`/`data.e.where` in the house that referenced the old location.

---

## 16. Room lifecycle: creating, deleting, navigating

Everything in this section is reachable only in `kEditMode` (`theMode == 1`), and almost
everything is additionally gated on `houseUnlocked`.

### 16.1 The House menu (`MENU`/`MENU` resource 131)

The House menu is *not* present in the menu bar outside edit mode. `UpdateMenus(true)`
inserts and removes it:

```
if (newMode)
{
    if (theMode == kEditMode)
        InsertMenu(houseMenu, 0);
    else
        DeleteMenu(kHouseMenuID);
}
```
(`GliderPRO/Sources/Menu.c:248-254`; `kHouseMenuID` = 131, `GliderPRO/Headers/GliderDefines.h:189`.)

`MENU` 131 parsed out of `Glider PRO.r` (menuID 131, `enableFlags` = `0xFFFADF77`,
`menuProc` 0, title `House`):

| # | Item text | Cmd key | `Externs.h` constant | Enabled by default |
|--:|---|---|---|:--:|
| 1 | `New House…` | ⌘N | `iNewHouse` = 1 (`GliderPRO/Headers/Externs.h:207`) | yes |
| 2 | `Save House` | ⌘S | `iSave` = 2 (`:208`) | yes |
| 3 | `-` (separator) | — | — | no |
| 4 | `House Info…` | — | `iHouse` = 4 (`:209`) | yes |
| 5 | `Room Info…` | ⌘R | `iRoom` = 5 (`:210`) | yes |
| 6 | `Object Info…` | ⌘I | `iObject` = 6 (`:211`) | yes |
| 7 | `-` | — | — | no |
| 8 | `Cut Room` | ⌘X | `iCut` = 8 (`:212`) | yes |
| 9 | `Copy Room` | ⌘C | `iCopy` = 9 (`:213`) | yes |
| 10 | `Paste Room` | ⌘V | `iPaste` = 10 (`:214`) | yes |
| 11 | `Delete Room` | — | `iClear` = 11 (`:215`) | yes |
| 12 | `Duplicate Object` | ⌘D | `iDuplicate` = 12 (`:216`) | yes |
| 13 | `-` | — | — | no |
| 14 | `Bring To Front` | ⌘= | `iBringForward` = 14 (`:217`) | yes |
| 15 | `Send To Back` | ⌘- | `iSendBack` = 15 (`:218`) | yes |
| 16 | `-` | — | — | no |
| 17 | `Go To Room…` | ⌘G | `iGoToRoom` = 17 (`:219`) | yes |
| 18 | `-` | — | — | no |
| 19 | `Map Window` | ⌘M | `iMapWindow` = 19 (`:220`) | yes |
| 20 | `Tools Window` | ⌘T | `iObjectWindow` = 20 (`:221`) | yes |
| 21 | `Coordinate Window` | ⌘K | `iCoordinateWindow` = 21 (`:222`) | yes |

`enableFlags` semantics: bit *n* enables item *n*, bit 0 enables the menu title.
`0xFFFADF77` = `1111 1111 1111 1010 1101 1111 0111 0111`. The clear bits are
**3, 7, 13, 16, 18** — exactly the five separator items, and nothing else. So every real
command starts life enabled and is enabled/disabled at runtime.

There is no `Save As` item. `iSaveAs` does not exist in `Externs.h`, the menu resource has
no such item, and every reference to it in `Menu.c` is commented out
(`GliderPRO/Sources/Menu.c:107`, `:112`, `:150`, `:470-472`).

Note also the *text* of items 8/9/10/11 is rewritten at runtime by `UpdateClipboardMenus`
(§16.2) — the authored strings say "Room" but they become "Object" whenever an object is
selected.

### 16.2 Menu enable/disable state machine

`UpdateMenus(Boolean newMode)` (`GliderPRO/Sources/Menu.c:243-272`) is the single entry
point; it is called from ~40 places after any state change. In edit mode it runs
`UpdateMenusEditMode()`, then either `UpdateMenusHouseOpen()` + `UpdateClipboardMenus()`
or `UpdateMenusHouseClosed()`, then `UpdateLinkControl()`, then `DrawMenuBar()`.
It short-circuits on `if (!menusUp) return;`.

**`UpdateMenusHouseOpen`** (`GliderPRO/Sources/Menu.c:98-141`):

| Item | Enabled iff |
|---|---|
| `iLoadHouse` (Game 5) | always |
| `iSave` (2) | `fileDirty && houseUnlocked` |
| `iHouse` (4) | `houseUnlocked` |
| `iRoom` (5) | `!noRoomAtAll && houseUnlocked` |
| `iObject` (6) | `objActive != kNoObjectSelected && houseUnlocked` |
| `iBringForward` (14), `iSendBack` (15) | `objActive != kNoObjectSelected && houseUnlocked` **and** `objActive` is not one of `kInitialGliderSelected` (−2), `kLeftGliderSelected` (−3), `kRightGliderSelected` (−4) |

Note the sign test: `objActive != kNoObjectSelected` (−1) is true for the three glider
pseudo-selections, so Object Info *is* enabled for them (they have their own dialogs,
§12.16) while depth ordering is not.

**`UpdateMenusHouseClosed`** (`:146-159`) disables `iSave`, `iHouse`, `iRoom`, `iObject`,
`iCut`, `iCopy`, `iPaste`, `iClear`, `iDuplicate` and `iLoadHouse`.

**`UpdateClipboardMenus`** (`:165-235`) returns immediately if `!houseOpen`. If
`houseUnlocked`:

1. If `objActive > kNoObjectSelected` (i.e. a real object 0..23 is selected) the item
   titles become `GetLocalizedString(36/37/38)` = `Cut Object` / `Copy Object` /
   `Clear Object`, and `iDuplicate` is enabled.
2. Otherwise they become `GetLocalizedString(39/40/41)` = `Cut Room` / `Copy Room` /
   `Clear Room`, and `iDuplicate` is disabled.
3. `iCut`, `iCopy`, `iClear`, `iGoToRoom`, `iMapWindow`, `iObjectWindow`,
   `iCoordinateWindow` are enabled.
4. `iPaste` is **always disabled** and always retitled `GetLocalizedString(44)` =
   `Nothing To Paste`; the `hasScrap` / `scrapIsARoom` branch that would have enabled it
   is commented out (`GliderPRO/Sources/Menu.c:197-216`). See §20.

If `!houseUnlocked` all nine of those items are disabled.

`GetLocalizedString(short index, StringPtr theString)` is
`GetIndString(theString, 150, index)` (`GliderPRO/Sources/StringUtils.c:321-325`,
`kLocalizedStringsID` = 150). The full `STR#` 150 table is in §18.6.

**`DoHouseMenu`** (`GliderPRO/Sources/Menu.c:436-585`) dispatches; every case except
`iNewHouse` and `iSave` is wrapped in `if (houseUnlocked)`.

### 16.3 `CreateNewRoom(h, v)`

`GliderPRO/Sources/Room.c:159-239`. `h` is the **suite**, `v` is the **floor** (in
"user" coordinates, i.e. floor 0 = ground, negative = underground). Returns `Boolean`.

```
 1  CopyThisRoomToRoom()                    // flush the room currently being edited
 2  thisRoom->name       = "Untitled Room"  // Pascal string, 13 chars
 3  thisRoom->leftStart  = 32               // Byte
 4  thisRoom->rightStart = 32               // Byte
 5  thisRoom->bounds     = 0
 6  thisRoom->unusedByte = 0
 7  thisRoom->visited    = false
 8  thisRoom->background = lastBackground   // sticky global: last background chosen
 9  SetInitialTiles(thisRoom->background, true)
10  thisRoom->floor      = v
11  thisRoom->suite      = h
12  thisRoom->openings   = 0
13  thisRoom->numObjects = 0
14  for i in 0..kMaxRoomObs-1:  thisRoom->objects[i].what = kObjectIsEmpty  // −1
15  wasState = HGetState(thisHouse); MoveHHi(thisHouse); HLock(thisHouse)
16  availableRoom = −1
17  if (*thisHouse)->nRooms > 0:
18      for i in 0..nRooms-1:
19          if rooms[i].suite == kRoomIsEmpty: availableRoom = i; break
20  if availableRoom == −1:                       // no recycled slot
21      HUnlock(thisHouse)
22      theErr = PtrAndHand(thisRoom, thisHouse, sizeof(roomType))   // append 348 bytes
23      if theErr != noErr:
24          YellowAlert(kYellowUnaccounted, theErr); MoveHHi; HLock; return false
25      MoveHHi(thisHouse); HLock(thisHouse)
26      (*thisHouse)->nRooms++
27      numberRooms   = (*thisHouse)->nRooms
28      previousRoom  = thisRoomNumber
29      thisRoomNumber = numberRooms − 1
30  else:                                          // reuse the deleted slot
31      previousRoom  = thisRoomNumber
32      thisRoomNumber = availableRoom
33  if noRoomAtAll: (*thisHouse)->firstRoom = thisRoomNumber
34  HSetState(thisHouse, wasState)
35  CopyThisRoomToRoom()                    // commit the new room into the handle
36  UpdateEditWindowTitle()
37  noRoomAtAll = false
38  fileDirty   = true
39  UpdateMenus(false)
40  GetKeys(theKeys)
41  newRoomNow = BitTst(theKeys, kShiftKeyMap) ? false : autoRoomEdit
42  return true
```

Points a porter must get exactly right:

* Step 22 grows the *whole house handle* by exactly `sizeof(roomType)` = **348** bytes and
  the bytes copied are the *staging* `thisRoom` — i.e. the room record is appended already
  filled in. Step 26 then bumps `nRooms`. If `PtrAndHand` fails the function re-locks and
  returns `false` **without** having incremented `nRooms`, but `thisRoom` has already been
  overwritten with the new empty room — the previously-edited room's contents are lost.
* The empty-slot search (steps 17-19) takes the **first** `suite == kRoomIsEmpty` slot, so
  deleted rooms are recycled in file order, not LRU.
* `lastBackground` is a sticky global (`GliderPRO/Sources/Room.c:28`), so a new room
  inherits the background of whatever room was last given one, not a fixed default.
* Step 41: holding **Shift** while creating a room suppresses the automatic Room Info
  dialog. `newRoomNow` is polled by the main event loop, which then calls `DoRoomInfo()`.
  `autoRoomEdit` is a saved preference.
* `CreateNewRoom` never validates `h`/`v`; the caller (`HandleMapClick`, §15.9) is the
  only place that computes them, from the map cell that was double-clicked.

Callers: `HandleMapClick` (`GliderPRO/Sources/Map.c`, after the ALRT 1004 confirmation),
and nothing else. There is no keyboard or menu command to create a room; the map window is
the only room-creation UI.

### 16.4 `SetInitialTiles(background, doRoom)`

`GliderPRO/Sources/Room.c:40-153`. Fills either `thisRoom->tiles[0..7]` (when
`doRoom` is true) or the global staging array `tempTiles[0..7]` (when false — used by the
Room Info dialog before the user commits). `kNumTiles` = 8.

| `background` | tiles written |
|---|---|
| `>= kUserBackground` (3000) | `tiles[i] = i` for i in 0..7 |
| `kSimpleRoom` 2000, `kPaneledRoom` 2001, `kBasement` 2002, `kChildsRoom` 2003, `kAsianRoom` 2004, `kUnfinishedRoom` 2005, `kSwingersRoom` 2006, `kBathroom` 2007, `kLibrary` 2008 | all 8 = 1, then `tiles[0] = 0` and `tiles[7] = 7` (only when `doRoom`; the `tempTiles` branch does the same) |
| `kSkywalk` 2010 | `tiles[i] = i` |
| `kField` 2013, `kGarden` 2009, `kDirt` 2011 | all 8 = 0 |
| `kMeadow` 2012 | all 8 = 1 |
| `kRoof` 2014 | all 8 = 3 |
| `kSky` 2015 | all 8 = 2 |
| `kStratosphere` 2016, `kStars` 2017 | `tiles[i] = i` |
| anything else | nothing (tiles left as they were) |

The nine "indoor" backgrounds get end-cap tiles: column 0 uses tile 0 (left wall), columns
1-6 use tile 1 (plain wall), column 7 uses tile 7 (right wall). Each tile is
`kTileWide` = 64 px wide out of `kRoomWide` = 512.

### 16.5 Deleting a room

`DeleteRoom(Boolean doWarn)` — `GliderPRO/Sources/Room.c:439-485`.

```
 1  if theMode != kEditMode or noRoomAtAll: return
 2  if doWarn and !QueryDeleteRoom(): return
 3  DeselectObject()
 4  HLock(thisHouse)
 5  wasFloor     = rooms[thisRoomNumber].floor
 6  wasSuite     = rooms[thisRoomNumber].suite
 7  firstDeleted = ((*thisHouse)->firstRoom == thisRoomNumber)
 8  thisRoom->suite                        = kRoomIsEmpty   // −1
 9  rooms[thisRoomNumber].suite            = kRoomIsEmpty
10  HSetState(...)
11  noRoomAtAll = (RealRoomNumberCount() == 0)
12  if noRoomAtAll: thisRoomNumber = kRoomIsEmpty
13  else:           SetToNearestNeighborRoom(wasFloor, wasSuite)
14  if firstDeleted: (*thisHouse)->firstRoom = thisRoomNumber
15  newRoomNow = false
16  fileDirty  = true
17  UpdateMenus(false)
18  ReflectCurrentRoom(false)
```

Deletion is a **tombstone**: the 348-byte record stays in the file, only `suite` is set to
−1. `RealRoomNumberCount()` (`GliderPRO/Sources/House.c:170-189`) returns
`nRooms` minus the number of tombstones. Tombstones are physically removed only by
`CompressHouse()` + `LopOffExtraRooms()` during `CheckHouseForProblems()` on save (§18.4).

`DeleteRoom` does **not** rewrite any link that pointed at the deleted room, and does not
touch the deleted room's own objects. §17.10 shows the observable consequence in the
shipped houses.

`QueryDeleteRoom()` (`GliderPRO/Sources/Room.c:490-500`) is
`Alert(kDeleteRoomAlert = 1005, nil)` and returns true iff the result is
`kYesDoDeleteRoom` = 1.

**ALRT 1005** header (verified from the resource bytes): rect `(t0,l0,b96,r288)`,
`itemsID` 1005, stages `0x5555`. **DITL 1005** (verified numerically):

| # | Type | Rect (t,l,b,r) | Contents |
|--:|---|---|---|
| 1 | Button | 68,222,88,280 | `Delete` |
| 2 | Button | 68,156,88,214 | `Cancel` |
| 3 | StaticText (disabled) | 12,12,52,225 | `Do you really want to delete this room?` |
| 4 | Icon (disabled) | 12,248,44,280 | `cicn`/`ICON` id **1005** (bytes `03 ED`) |

Item 1 (`Delete`) is the default button, so Return deletes. There is **no undo**.

Callers (all four, grepped across the tree):

| Call site | `doWarn` | Gesture |
|---|:-:|---|
| `GliderPRO/Sources/Events.c:257` | `true` | **Delete/Backspace key** in edit mode with no object selected (`if (objActive == kNoObjectSelected) DeleteRoom(true); else DeleteObject();`) |
| `GliderPRO/Sources/Scrap.c:455` | `true` | dropping the room's map thumbnail on the **Finder Trash** (drag left the app, option not held, `DropLocationIsTrash`) |
| `GliderPRO/Sources/Menu.c:509` | `false` | House ▸ **Cut** (`iCut`) with no object selected |
| `GliderPRO/Sources/Menu.c:544` | `false` | House ▸ **Clear** (`iClear`) with no object selected |

So the two **menu** paths delete with **no confirmation alert at all**, while the keyboard
and drag-to-Trash paths do confirm. That asymmetry is almost certainly a bug (§21).
`HandleMapClick` never calls `DeleteRoom` — the map window has no delete gesture of its own;
its only room-destroying involvement is as the *source* of the Trash drag in `Scrap.c`.

### 16.6 `SetToNearestNeighborRoom(wasFloor, wasSuite)` — the spiral walk

`GliderPRO/Sources/Room.c:639-707`. After a deletion this picks the nearest surviving
room and makes it current. It walks a clockwise square spiral outward from
`(wasSuite, wasFloor)`:

```
 1  distance = 1;  h = −1;  v = 0            // start at the room to the left
 2  hStep = 0;  vStep = −1                   // initial travel direction: up
 3  loop:
 4      testH = wasSuite + h;  testV = wasFloor + v
 5      if RoomExists(testH, testV, &testRoomNum):
 6          CopyRoomToThisRoom(testRoomNum);  finished = true
 7      else:
 8          h += hStep;  v += vStep
 9          if |h| > distance or |v| > distance:      // stepped outside the ring
10              if hStep == −1 and vStep == 0:       // finished the ring
11                  distance++;  hStep = 0;  vStep = −1        // start next ring going up
12              else:
13                  h −= hStep;  v −= vStep          // back up one
14                  if hStep == 0:                   // was moving vertically
15                      hStep = (vStep == −1) ? 1 : −1;  vStep = 0
16                  else:                            // was moving horizontally
17                      hStep = 0;  vStep = 1
18                  h += hStep;  v += vStep
19  while !finished
```

`v` here is a **floor delta** (`testV = wasFloor + v`), so `vStep = −1` moves to a *lower*
floor even though the source comment at `GliderPRO/Sources/Room.c:657` calls it "we 'walk'
up". The comment — and the word "clockwise" — are written in map-row terms (§15.3, where
`row = 56 − floor`, so decreasing floor is downward on screen and the ring is traversed
clockwise there); in floor terms the ring is traversed counter-clockwise. The spiral is
symmetric, so this only affects the tie-break order among equidistant rooms: ring 1 is
visited as `(−1,0) (−1,−1) (0,−1) (1,−1) (1,0) (1,1) (0,1) (−1,1)`, i.e. left, then
below-left, below, below-right, right, above-right, above, above-left. A port that wants
identical room selection after a delete must reproduce that order exactly.

Note also that when a ring *completes* (step 11) the code does **not** back up: `h`/`v` are
left at the position that overflowed the old ring, which is already a valid cell of the new
ring, and that cell is tested first.

**This loop has no termination guard.** If `RoomExists` never succeeds it spins forever,
`h`/`v` growing without bound until `short` overflow. `DeleteRoom` only calls it when
`RealRoomNumberCount() != 0`, so in practice a room always exists — but `RoomExists`
returns false immediately for `suite < 0` (`GliderPRO/Sources/Room.c:400`-`401`), so the
half-plane of negative suites is skipped rather than terminating the search, and a house
whose only surviving rooms are at negative suites (impossible via the editor, possible in
a hand-edited file) hangs the application. A Go port must bound the search
(e.g. `distance <= kMaxNumRoomsH` = 128) and fall back to a linear scan.

### 16.7 Neighbour navigation

Three different neighbour lookups exist, with different conventions. A porter must not
conflate them.

**(a) `DoesNeighborRoomExist(short whichNeighbor)`** —
`GliderPRO/Sources/Room.c:505-540`. Uses the 4-way constants
`kRoomAbove` 1, `kRoomBelow` 2, `kRoomToRight` 3, `kRoomToLeft` 4
(`GliderPRO/Headers/GliderDefines.h:200-203`). Starts from **`thisRoom`** (the staging
copy) and returns the room number or −1. Returns −1 immediately if
`theMode != kEditMode`.

| `whichNeighbor` | suite delta | floor delta |
|---|--:|--:|
| `kRoomAbove` (1) | 0 | +1 |
| `kRoomBelow` (2) | 0 | −1 |
| `kRoomToRight` (3) | +1 | 0 |
| `kRoomToLeft` (4) | −1 | 0 |

Note the call `RoomExists(newH, newV, &newRoomNumber)` — arguments are
`(suite, floor)`, matching `RoomExists`'s signature.

Falling off the end of the `#ifndef COMPILEDEMO` block, this function has **no `return`
statement** on the demo build path; in the shipping build the `#endif` sits after the
final `return`, so it is well-formed.

**(b) `SelectNeighborRoom(short whichNeighbor)`** —
`GliderPRO/Sources/Room.c:544-558`:

```
newRoomNumber = DoesNeighborRoomExist(whichNeighbor)
if newRoomNumber != −1:
    DeselectObject()
    CopyRoomToThisRoom(newRoomNumber)
    ReflectCurrentRoom(false)
```

This is what the arrow keys do in edit mode (when no object is selected).

**(c) `GetNeighborRoomNumber(short which)`** —
`GliderPRO/Sources/Room.c:562-632`. Uses the *9-way* constants
`kCentralRoom` 0, `kNorthRoom` 1, `kNorthEastRoom` 2, `kEastRoom` 3, `kSouthEastRoom` 4,
`kSouthRoom` 5, `kSouthWestRoom` 6, `kWestRoom` 7, `kNorthWestRoom` 8
(`GliderPRO/Headers/GliderDefines.h:217-225`), reads floor/suite from
**`(*thisHouse)->rooms[thisRoomNumber]`** (not `thisRoom`), and returns `kRoomIsEmpty`
(−1) when not found.

| `which` | name | hDelta | vDelta |
|--:|---|--:|--:|
| 0 | `kCentralRoom` | 0 | 0 |
| 1 | `kNorthRoom` | 0 | +1 |
| 2 | `kNorthEastRoom` | +1 | +1 |
| 3 | `kEastRoom` | +1 | 0 |
| 4 | `kSouthEastRoom` | +1 | −1 |
| 5 | `kSouthRoom` | 0 | −1 |
| 6 | `kSouthWestRoom` | −1 | −1 |
| 7 | `kWestRoom` | −1 | 0 |
| 8 | `kNorthWestRoom` | −1 | +1 |

`hDelta`/`vDelta` are **uninitialised** if `which` is outside 0..8 — the `switch` has no
`default`. Only `CheckForStaircasePairs` (§18.4) and the 9-room play-mode renderer call
this, always with literal constants.

### 16.8 Room lookup helpers

| Function | Location | Signature / behaviour |
|---|---|---|
| `RoomExists` | `GliderPRO/Sources/Room.c:390-421` | `(short suite, short floor, short *roomNum) -> Boolean`. Returns false at once if `suite < 0`. Linear scan of `0..numberRooms-1` on the *first* match of `floor` **and** `suite`. Does **not** skip tombstones explicitly — but a tombstone has `suite == −1` which can never match a `suite >= 0` query. |
| `RoomNumExists` | `:425-435` | `(short roomNum) -> Boolean`. `GetRoomFloorSuite(roomNum,…)` then `RoomExists(...)`. |
| `GetRoomNumber` | `:737-759` | `(short floor, short suite) -> short`. **Arguments are in the opposite order to `RoomExists`.** Returns `kRoomIsEmpty` (−1) when absent. No `suite < 0` early-out, so `GetRoomNumber(f, −1)` will happily return the index of the first tombstone whose floor matches. |
| `GetRoomFloorSuite` | `:711-733` | `(short room, short *floor, short *suite) -> Boolean`. Returns false and writes `*floor = 0, *suite = kRoomIsEmpty` for a tombstone. **No bounds check on `room`** — `GetRoomFloorSuite(-1, …)` or a value `>= nRooms` reads out of bounds. |
| `RealRoomNumberCount` | `GliderPRO/Sources/House.c:170-189` | `nRooms` minus tombstones. |
| `GetFirstRoomNumber` | `:196-216` | Returns `(*thisHouse)->firstRoom`, clamped: if `nRooms <= 0` returns −1 and sets `noRoomAtAll = true`; if `firstRoom >= nRooms` or `< 0`, returns 0. Note it can return the index of a *tombstone*. |
| `IsRoomAStructure` | `GliderPRO/Sources/Room.c:763-…` | See §13 (Room Info); for `background >= kUserBackground` it consults `bounds & 32` when `bounds != 0`, else `background < kUserStructureRange` (3300). |

`numberRooms` is a *cached copy* of `(*thisHouse)->nRooms` (set in `ReadHouse`,
`CreateNewRoom`, `ValidateNumberOfRooms`, `LopOffExtraRooms`). `RoomExists` and
`GetRoomNumber` iterate `numberRooms`, while `CountHouseLinks`, `GenerateLinksList`,
`GenerateRetroLinks`, and every `HouseLegal.c` pass iterate `(*thisHouse)->nRooms`
directly. Any divergence between the two is a latent out-of-bounds read.

### 16.9 Room staging: `thisRoom` vs the house handle

There are two copies of the room being edited and the editor moves data between them
constantly. `thisRoom` is a `roomPtr` allocated once by
`CreatePointers` (`GliderPRO/Sources/StructuresInit2.c:192-195`,
`NewPtr(sizeof(roomType))` = 348 bytes).

| Function | Location | Effect |
|---|---|---|
| `CopyThisRoomToRoom()` | `GliderPRO/Sources/Room.c:354-365` | `(*thisHouse)->rooms[thisRoomNumber] = *thisRoom`. No-op if `noRoomAtAll` or `thisRoomNumber == -1`. |
| `ForceThisRoom(n)` | `:369-386` | `*thisRoom = (*thisHouse)->rooms[n]` (with `YellowAlert(kYellowIllegalRoomNum, 0)` if `n >= nRooms`), then `previousRoom = thisRoomNumber; thisRoomNumber = n`. No-op if `n == −1`. **Discards** unsaved changes in `thisRoom`. |
| `CopyRoomToThisRoom(n)` | `:343-350` | `CopyThisRoomToRoom(); ForceThisRoom(n);` — the safe "switch rooms" primitive. No-op if `n == −1`. |
| `ReflectCurrentRoom(force)` | `:308-338` | Re-derives all derived state after a room switch: map recenter/redraw, `GenerateRetroLinks()`, `UpdateEditWindowTitle()`, `ReadyBackground(thisRoom->background, thisRoom->tiles)`, `GetThisRoomsObjRects()`, `DrawThisRoomsObjects()`, `InvalWindowRect(mainWindow, &mainWindowRect)`. Returns immediately if `theMode != kEditMode`. When `noRoomAtAll || !houseUnlocked` it centres the map on `(suite 64, floor 1)` instead of the current room. |

A Go port that keeps a single mutable room object and an index would eliminate this whole
class of bug, but must then audit every original call site: several functions rely on
`thisRoom` being *stale* relative to the handle (e.g. `KeepAllObjectsLegal` uses
`ForceThisRoom` + `CopyThisRoomToRoom` as an explicit load/store pair).

### 16.10 `Go To Room…` (`DLOG`/`DITL` 1043)

`DoGoToDialog()` — `GliderPRO/Sources/House.c:665-737`. `kGoToDialogID` = 1043
(`GliderPRO/Sources/House.c:18`).

**DLOG 1043**, raw resource bytes
`00 28 00 28 00 c9 01 40 | 00 01 | 01 00 | 01 00 | 00 00 00 00 | 04 13 | 00`:
rect `(t40,l40,b201,r320)` = 161 × 280, `procID` = **1** (`dBoxProc` — modal, no title
bar), `visible` = 1, `goAwayFlag` = 1 (meaningless for `dBoxProc`), `refCon` = 0,
`itemsID` = **1043**, and the window **title is the empty string** (length byte `00`).
The human-readable `Go To Room…` is the *resource name*, not the title.

**DITL 1043**, verified numerically:

| # | Type | Rect (t,l,b,r) | Contents / constant |
|--:|---|---|---|
| 1 | Button | 133,214,153,272 | `Cancel` — but the code treats it as `kOkayButton` |
| 2 | Button | 40,8,60,168 | `Go to First Room` (`kGoToFirstButt` = 2) |
| 3 | Button | 68,8,88,168 | `Go to Previous Room` (`kGoToPrevButt` = 3) |
| 4 | Button | 96,8,116,128 | `Go to Room…` (`kGoToFSButt` = 4) |
| 5 | EditText | 98,174,114,194 | floor field (`kFloorEditText` = 5) |
| 6 | EditText | 98,241,114,269 | suite field (`kSuiteEditText` = 6) |
| 7 | StaticText (disabled) | 98,132,114,171 | `floor:` |
| 8 | StaticText (disabled) | 99,199,115,238 | `suite:` |
| 9 | Picture (disabled) | 0,0,32,280 | `PICT` id **1023** (bytes `03 FF`), 280 × 32 banner |
| 10 | UserItem (disabled) | 124,8,125,272 | 1-pixel-tall divider, framed by `UpdateGoToDialog` |

Flow:

```
 1  goToFilterUPP = NewModalFilterUPP(GoToFilter)
 2  BringUpDialog(&theDialog, 1043)
 3  if GetFirstRoomNumber() == thisRoomNumber:  MyDisableControl(dial, 2)
 4  if !RoomNumExists(previousRoom) or previousRoom == thisRoomNumber:
 5      MyDisableControl(dial, 3)
 6  SetDialogNumToStr(dial, 5, wasFloor)      // globals persist across invocations
 7  SetDialogNumToStr(dial, 6, wasSuite)
 8  SelectDialogItemText(dial, 5, 0, 1024)
 9  loop ModalDialog(goToFilterUPP, &item):
10      item == 1 (kOkayButton):  roomToGoTo = −1; canceled = true; leaving = true
11      item == 2:                roomToGoTo = GetFirstRoomNumber(); leaving = true
12      item == 3:                roomToGoTo = previousRoom;         leaving = true
13      item == 4:                wasFloor = atoi(item 5); wasSuite = atoi(item 6)
14                                roomToGoTo = GetRoomNumber(wasFloor, wasSuite)
15                                leaving = true
16  DisposeDialog; DisposeModalFilterUPP
17  if !canceled:
18      if RoomNumExists(roomToGoTo):
19          DeselectObject(); CopyRoomToThisRoom(roomToGoTo); ReflectCurrentRoom(false)
20      else: SysBeep(1)
```

`wasFloor`/`wasSuite` are file-scope globals in `House.c` (`GliderPRO/Sources/House.c:30`)
shared with nothing else, so the dialog remembers the last typed coordinates for the whole
session.

`GoToFilter` (`GliderPRO/Sources/House.c:628-659`) maps **both** `kReturnKeyASCII` and
`kEnterKeyASCII` to `kOkayButton` (item 1) after `FlashDialogButton(dial, kOkayButton)`,
and repaints on `updateEvt` via `UpdateGoToDialog`. Since item 1 is labelled **Cancel**,
pressing Return *cancels* the dialog. `UpdateGoToDialog`
(`GliderPRO/Sources/House.c:618-623`) does `DrawDialog`, `DrawDefaultButton`, and
`FrameDialogItemC(theDialog, 10, kRedOrangeColor8)` — `kRedOrangeColor8` = 23
(`GliderPRO/Headers/GliderDefines.h:542`, whose comment says "actually, 18").

Note the floor typed here is the **user-visible** floor (may be negative); `GetRoomNumber`
compares it directly to `rooms[i].floor`, with no `kNumUndergroundFloors` bias. The bias
appears only in link encoding (§17.1).

### 16.11 Creating a whole new house

**`CreateNewHouse()`** — `GliderPRO/Sources/House.c:45-100`. Navigation Services:

```
 1  NavGetDefaultDialogOptions(&dialogOptions)
 2  NavPutFile(nil, &theReply, &dialogOptions, nil, 'gliH', 'ozm5', nil)
 3  if theErr == userCanceledErr or !theReply.validRecord: return false
 4  AEGetNthPtr(&theReply.selection, 1, typeFSS, …, &theSpec, sizeof(FSSpec), …)
 5  if theReply.replacing:
 6      FSMakeFSSpec(...) ; FSpDelete(&tempSpec)        // both via CheckFileError
 7  if houseOpen and !CloseHouse(): return false
 8  FSpCreate(&theSpec, 'ozm5', 'gliH', theReply.keyScript)
 9  HCreateResFile(theSpec.vRefNum, theSpec.parID, theSpec.name)
10  if ResError() != noErr: YellowAlert(kYellowFailedResCreate, ResError())
11  PasStringCopy(theSpec.name, thisHouseName)
12  AddExtraHouse(&theSpec);  BuildHouseList();  InitCursor()
13  return OpenHouse()
```

The file type is `'gliH'` and the creator is `'ozm5'` ("Ozma" — Calhoun's signature).
Verified empirically: all 22 shipped `.binhex` houses decode to type `gliH`, creator
`ozm5`. Both a **data fork** (the house itself) and an **empty resource fork** are
created; the resource fork later holds user `PICT`s, `'bnds'`, and `'snd '` resources
(§13.4).

**`InitializeEmptyHouse()`** — `GliderPRO/Sources/House.c:108-159`. Allocates a handle of
exactly `sizeof(houseType)` = **866** bytes (no rooms) and fills it:

| Field | Value |
|---|---|
| `version` | `kHouseVersion` = `0x0200` |
| `unusedShort` | not written (garbage from `NewHandle`) |
| `firstRoom` | −1 |
| `timeStamp` | `0L` |
| `flags` | `0L` |
| `initial` | `.h = 32`, `.v = 32` |
| `highScores` | `ZeroHighScores()` |
| `banner` | `GetLocalizedString(11)` = `Enter New-Game Message here (max. 255 characters)` |
| `trailer` | `GetLocalizedString(12)` = `Enter Finished-House Message here (max. 255 characters)` |
| `hasGame` | `false` |
| `unusedBoolean` | not written |
| `nRooms` | 0 |

then the globals: `wardBitSet = false`, `phoneBitSet = false`, `numberRooms = 0`,
`mapLeftRoom = 60`, `mapTopRoom = 50`, `thisRoomNumber = kRoomIsEmpty`,
`previousRoom = −1`, `houseUnlocked = true`, `OpenMapWindow()`, `UpdateMapWindow()`,
`noRoomAtAll = true`, `fileDirty = true`, `UpdateMenus(false)`,
`ReflectCurrentRoom(true)`.

`mapLeftRoom = 60` / `mapTopRoom = 50` puts the map viewport at suite 60, map row 50
(= floor `kMapGroundValue − 50` = 6) so that the first room the author creates lands
near the middle of the 128 × 64 map grid rather than at a corner
(`GliderPRO/Sources/House.c:148-149`; see §15.3 for the transform).

`InitializeEmptyHouse` uses `NewHandle`, not `NewHandleClear`, so `unusedShort` and
`unusedBoolean` in a brand-new house contain uninitialised memory. Empirically the shipped
houses all have `unusedShort == 0`, but a Go port should write zeros deliberately.

`DoHouseMenu` case `iNewHouse` (`GliderPRO/Sources/Menu.c:444-450`):
`if (CreateNewHouse()) { InitializeEmptyHouse(); OpenCloseEditWindows(); }`. Note
`CreateNewHouse` ends by calling `OpenHouse()`, which reads the (zero-length) data fork;
`InitializeEmptyHouse` then discards whatever `OpenHouse`/`ReadHouse` produced and builds
the 866-byte handle from scratch.

---

## 17. Linking objects across rooms

Links are the editor's most intricate feature: a switch/trigger/transport object in one
room stores a reference to a *specific object slot in a specific room* elsewhere in the
house. All of it lives in `GliderPRO/Sources/Link.c` (396 lines) plus three call sites in
`ObjectInfo.c`.

### 17.1 The on-disk encoding: `MergeFloorSuite` / `ExtractFloorSuite`

A link is two fields inside the 10-byte object payload:

| Field | Payload offset | Type | Meaning |
|---|--:|---|---|
| `where` | **6-7** | `short` (big-endian) | encoded destination room, or **−1** = no link |
| `who` | **8** | `Byte` | destination object slot 0..23, or **255** = no object |

This holds for both union variants that can be a link source — `transportType`
(`data.d`, `GliderPRO/Headers/GliderStructs.h:44-52`) and `switchType`
(`data.e`, `:54-62`) — and the two fields are at the *same* byte offsets in both, which is
why the code can be sloppy about which variant it writes through (§17.7).

```
short MergeFloorSuite (short floor, short suite)
{
    return ((suite * 100) + floor);
}
```
(`GliderPRO/Sources/Link.c:34-37`)

```
void ExtractFloorSuite (short combo, short *floor, short *suite)
{
    if ((*thisHouse)->version < 0x0200)      // old floor/suite combo
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
(`GliderPRO/Sources/Link.c:41-53`; `kNumUndergroundFloors` = **8**,
`GliderPRO/Headers/GliderDefines.h:535`.)

So in house version **0x0200** (the only version any shipped house uses — verified: all 22
`.binhex` houses report `version == 0x0200`):

```
where = suite * 100 + (floor + 8)
suite = where / 100                  (C truncating division)
floor = where % 100 - 8              (C truncating remainder)
```

In version **0x0100** the two halves were swapped: `where = floor_biased * 100 + suite`.
`ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746-822`) rewrites every link when a v1
house is saved: it calls `ExtractFloorSuite` (which takes the v1 branch because `version`
is still `< 0x0200` at that point), adds `kNumUndergroundFloors` to the floor, and calls
`MergeFloorSuite` — then finally sets `version = kHouseVersion`.

**The `+ 8` bias exists only inside `where`.** The Go To Room dialog (§16.10), the map
window (§15), `RoomExists`, `GetRoomNumber` and `roomType.floor` all use *unbiased* floor
numbers. `DoLink` adds the bias on the way in and `ExtractFloorSuite` removes it on the
way out; nowhere else.

**C division semantics are load-bearing.** `ExtractFloorSuite` relies on `/` and `%`
truncating toward zero. For a negative `where` this differs from floor-division languages:
`where = −50` gives C `suite = 0, floor = −58`, but Python gives `suite = −1, floor = 42`.
Go's `/` and `%` truncate exactly like C, so a straight transcription is correct — but any
port through Python/Ruby/Lua must be explicit.

Empirically (all 22 houses, `where != −1`): **2474 links**, `where` range
**[−100, 12531]**.

| Verdict | Count |
|---|--:|
| fully valid (room exists, `who` ≤ 23, target slot non-empty) | **2267** |
| `where` decodes to a floor/suite with no room | **189** |
| room exists but `who` > 23 | **13** |
| room and `who` fine, but target slot is `kObjectIsEmpty` | **5** |

### 17.2 Which object types are link *sources*

Three whitelists, all identical, appear in `CountHouseLinks`
(`GliderPRO/Sources/House.c:277-300`), `GenerateLinksList` (`:337-380`) and
`GenerateRetroLinks` (`:559-608`):

**Switch-style sources (`data.e.where`):**

| Constant | Value |
|---|--:|
| `kLightSwitch` | `0x41` |
| `kMachineSwitch` | `0x42` |
| `kThermostat` | `0x43` |
| `kPowerSwitch` | `0x44` |
| `kKnifeSwitch` | `0x45` |
| `kInvisSwitch` | `0x46` |
| `kTrigger` | `0x47` |
| `kLgTrigger` | `0x48` |

**Transport-style sources (`data.d.where`):**

| Constant | Value |
|---|--:|
| `kMailboxLf` | `0x33` |
| `kMailboxRt` | `0x34` |
| `kFloorTrans` | `0x35` |
| `kCeilingTrans` | `0x36` |
| `kInvisTrans` | `0x3F` |
| `kDeluxeTrans` | `0x40` |

**`kSoundTrigger` (`0x49`) is deliberately absent from all three lists.** Its `where` field
is not a link at all — it is a **sound resource ID**. `AddNewObject` seeds it with
`data.e.where = 3000` (`GliderPRO/Sources/ObjectAdd.c:498-501`) and
`DoCustPictObjectInfo` writes a picked resource ID straight into it
(`thisRoom->objects[objActive].data.e.where = (short)wasPict;`,
`GliderPRO/Sources/ObjectInfo.c:1246`). `DoObjectInfo` routes both `kSoundTrigger` and
`kCustomPict` to `DoCustPictObjectInfo` (`GliderPRO/Sources/ObjectInfo.c:2555-2557`).

Verified empirically: the 22 houses contain **122** `kSoundTrigger` objects and their
`where` values are all plausible resource IDs, never link codes —
3000 (×35), 3001 (×18), 3002 (×14), 3003 (×6), 3004 (×6), 3005 (×5), 3006 (×13),
3007 (×2), 3008 (×2), 3009, 3010, 3011 (×2), 3012, 3037, 3042 (×6), 3043, 3044 (×3),
3045, 3046, 3058, and **10000 (×2)** (21 distinct IDs, 122 objects). A Go port must therefore **never** run link validation or
link fix-up over `kSoundTrigger`.

`kSoundTrigger` is additionally rationed: `kMaxSoundTriggers` = **1**
(`GliderPRO/Sources/ObjectAdd.c:17`) and `HowManySoundObjects()` counts only within
`thisRoom` (`GliderPRO/Sources/ObjectAdd.c:989-999`), so **at most one sound trigger per
room**. Verified: the maximum `kSoundTrigger` count in any of the 4070 shipped rooms is
exactly **1**.

Observed distribution of the 2474 real links by source type:

| Source type | Links |
|---|--:|
| `kInvisSwitch` `0x46` | 634 |
| `kCeilingTrans` `0x36` | 280 |
| `kInvisTrans` `0x3F` | 277 |
| `kFloorTrans` `0x35` | 241 |
| `kTrigger` `0x47` | 239 |
| `kKnifeSwitch` `0x45` | 226 |
| `kLightSwitch` `0x41` | 113 |
| `kThermostat` `0x43` | 108 |
| `kLgTrigger` `0x48` | 81 |
| `kMachineSwitch` `0x42` | 78 |
| `kPowerSwitch` `0x44` | 77 |
| `kDeluxeTrans` `0x40` | 59 |
| `kMailboxLf` `0x33` | 39 |
| `kMailboxRt` `0x34` | 22 |

### 17.3 The Link windoid

`OpenLinkWindow()` — `GliderPRO/Sources/Link.c:212-252`:

```
if (linkWindow == nil)
{
    QSetRect(&linkWindowRect, 0, 0, 129, 30)            // left,top,right,bottom
    linkWindow = NewCWindow(nil, &linkWindowRect, "\pLink", false,
                            kWindoidWDEF, kPutInFront, true, 0L)
        // (NewWindow if !thisMac.hasColor)
    MoveWindow(linkWindow, isLinkH, isLinkV, true)      // saved position
    BringToFront(linkWindow);  ShowHide(linkWindow, true);  HiliteAllWindows()
    linkControl   = GetNewControl(kLinkControlID   = 130, linkWindow)
    unlinkControl = GetNewControl(kUnlinkControlID = 131, linkWindow)
    linkRoom   = −1
    linkObject = 255
    isLinkOpen = true
}
```

So the window is **129 × 30 px**, titled `Link`, drawn with `kWindoidWDEF` = 2048
(WDEF resource 128, variant 0 — a small "windoid" / floating palette frame) and a
**close box** (`goAwayFlag` = `true`).

`CNTL` resources, verified from the resource bytes:

| ID | Rect (t,l,b,r) | value | max | min | procID | refCon | Title |
|--:|---|--:|--:|--:|--:|--:|---|
| **130** | 5,70,25,124 | 0 | 0 | 0 | 0 (`pushButProc`) | 0 | `Link` |
| **131** | 5,5,25,59 | 0 | 0 | 0 | 0 (`pushButProc`) | 0 | `Unlink` |

Both are 54 × 20 push buttons. Note the layout: **`Unlink` is on the left** (left 5) and
**`Link` is on the right** (left 70). There is no other content — no text, no indication of
what is being linked.

`CloseLinkWindow()` (`:256-265`) is `DisposeWindow` + `linkWindow = nil` +
`isLinkOpen = false`. `UpdateLinkWindow()` (`:198-208`) is
`SetPortWindowPort` + `DrawControls` + `UpdateLinkControl`.

Event routing (`GliderPRO/Sources/Events.c`):

* `inDrag` → `DragWindow` then `GetWindowLeftTop(whichWindow, &isLinkH, &isLinkV)`
  (`:91-92`) — the position is remembered in the prefs globals.
* `inGoAway` → `TrackGoAway` then **`CloseLinkWindow()`** (`:105-106`), which silently
  abandons the pending link. Note this is *not* `ToggleLinkWindow` — there is no such
  function, and no menu item for the Link window, unlike Map/Tools/Coordinate.
* `updateEvt` → `BeginUpdate` / `UpdateLinkWindow` / `EndUpdate` (`:364-369`).
* `inContent` → `HandleLinkClick(theEvent->where)` (`:150-151`).
* On resume/activate, `HiliteWindow(linkWindow, true)` (`:558-559`).

### 17.4 The three entry points

There is no "Link" menu command. The Link windoid is only opened from an Object Info
dialog, by clicking that dialog's link button. All three sites look the same:

| Caller | `linkType` | `linkerIsSwitch` | Applies to `what` | Link item | Go To item | Linked From? item |
|---|---|:--:|---|--:|--:|--:|
| `DoSwitchObjectInfo` (`GliderPRO/Sources/ObjectInfo.c:1369-1377`) | `kSwitchLinkOnly` = **3** | `true` | `kLightSwitch`…`kInvisSwitch` (`0x41`–`0x46`) | **9** (`:1340`) | 14 = `kGotoButton2` (`:1348`) | 15 (`:1356`) |
| `DoTriggerObjectInfo` (`:1525-1531`) | `kTriggerLinkOnly` = **4** | `true` | `kTrigger` `0x47`, `kLgTrigger` `0x48` | **9** (`:1463`) | 14 = `kGotoButton2` (`:1482`) | 15 (`:1501`) |
| `DoTransObjectInfo` (`:2161-2167`) | `kTransportLinkOnly` = **5** | `false` | `kMailboxLf`…`kDeluxeTrans` (`0x33`–`0x36`, `0x3F`, `0x40`) | **6** = `kLinkTransButton` (`:2124`) | 11 = `kGotoButton1` (`:2133`) | 12 (`:2142`) |

(`kGotoButton1` = 11, `kGotoButton2` = 14, `kLinkTransButton` = 6 —
`GliderPRO/Sources/ObjectInfo.c:62`, `:63`, `:46`. The link and "Linked From?" items are
hard-coded integer literals with no named constants, except in the transport dialog.)

(`kSwitchLinkOnly`/`kTriggerLinkOnly`/`kTransportLinkOnly` = 3/4/5,
`GliderPRO/Headers/GliderDefines.h:463-465`. The dispatch table for `what` → dialog is
`DoObjectInfo`, `GliderPRO/Sources/ObjectInfo.c:2392-2564`.)

The exact code, from `DoSwitchObjectInfo`:

```
if (doLink)
{
    linkType = kSwitchLinkOnly;
    linkerIsSwitch = true;
    OpenLinkWindow();
    linkRoom   = thisRoomNumber;
    linkObject = (Byte)objActive;
    DeselectObject();
}
```
(`GliderPRO/Sources/ObjectInfo.c:1369-1377`)

Order matters: `OpenLinkWindow()` clobbers `linkRoom`/`linkObject` with −1/255, so the
caller must assign *after* the call. It works, but only because the reset lives inside the
`if (linkWindow == nil)` guard — if the windoid is already open the reset is skipped and
the caller's assignment is the only write. Both orderings happen to produce the right
answer, so this is a latent trap rather than a live bug.

`DeselectObject()` (`GliderPRO/Sources/ObjectEdit.c:1776-1786`) then sets
`objActive = kNoObjectSelected`, `StopMarquee()`, `UpdateMenus(false)` — which reaches
`UpdateLinkControl()` (`GliderPRO/Sources/Menu.c:266`) and therefore renders the `Link`
button **inactive** until the user picks a target.

### 17.5 The user-facing gesture

1. Select the switch / trigger / transport, choose **House ▸ Object Info…** (⌘I).
2. Click the dialog's link button (item 9, or 6 for a transport). The dialog closes; the
   Link windoid appears;
   the source is remembered in `linkRoom` / `linkObject` / `linkType` / `linkerIsSwitch`;
   the current selection is dropped.
3. Navigate anywhere in the house (map window, arrow keys, Go To Room…) and click the
   object that should be the *destination*. `objActive` changes, `UpdateMenus(false)` runs,
   `UpdateLinkControl()` decides whether the `Link` button lights up.
4. Click **Link** to write the link, or **Unlink** to clear it. Either action closes the
   windoid.

`Unlink` is **never disabled**, so it can be clicked at any moment after step 2 — including
with nothing selected — and will clear the source's link.

### 17.6 `UpdateLinkControl` — the destination whitelists

`GliderPRO/Sources/Link.c:57-194`. Returns immediately if `linkWindow == nil`. Then a
`switch (linkType)` with **no `default`** — `linkType` values other than 3/4/5 do nothing
at all. Within each arm, `objActive == kNoObjectSelected` forces
`HiliteControl(linkControl, kControlInactive)`; otherwise it switches on
`thisRoom->objects[objActive].what`.

**`kSwitchLinkOnly` (3) — 44 legal destinations** (`:71-124`), i.e. everything a switch
can operate:

| Group | Types |
|---|---|
| Blowers (11 of 16) | `kFloorVent` `0x01`, `kCeilingVent` `0x02`, `kFloorBlower` `0x03`, `kCeilingBlower` `0x04`, `kSewerGrate` `0x05`, `kLeftFan` `0x06`, `kRightFan` `0x07`, `kInvisBlower` `0x0D`, `kGrecoVent` `0x0E`, `kSewerBlower` `0x0F`, `kLiftArea` `0x10` |
| Prizes (10 of 15) | `kRedClock` `0x21`, `kBlueClock` `0x22`, `kYellowClock` `0x23`, `kCuckoo` `0x24`, `kPaper` `0x25`, `kBattery` `0x26`, `kBands` `0x27`, `kFoil` `0x2A`, `kInvisBonus` `0x2B`, `kHelium` `0x2E` |
| Transport (1) | `kDeluxeTrans` `0x40` |
| Lights (all 8) | `kCeilingLight` `0x51`, `kLightBulb` `0x52`, `kTableLamp` `0x53`, `kHipLamp` `0x54`, `kDecoLamp` `0x55`, `kFlourescent` `0x56`, `kTrackLight` `0x57`, `kInvisLight` `0x58` |
| Appliances (9 of 14) | `kShredder` `0x61`, `kToaster` `0x62`, `kMacPlus` `0x63`, `kTV` `0x65`, `kCoffee` `0x66`, `kOutlet` `0x67`, `kVCR` `0x68`, `kStereo` `0x69`, `kMicrowave` `0x6A` |
| Enemies (8 of 9) | `kBalloon` `0x71`, `kCopterLf` `0x72`, `kCopterRt` `0x73`, `kDartLf` `0x74`, `kDartRt` `0x75`, `kBall` `0x76`, `kDrip` `0x77`, `kFish` `0x78` |

Excluded on purpose: the flame blowers `kTaper` `0x08`, `kCandle` `0x09`, `kStubby` `0x0A`,
`kTiki` `0x0B`, `kBBQ` `0x0C`; the prizes `kGreaseRt` `0x28`, `kGreaseLf` `0x29`,
`kStar` `0x2C`, `kSparkle` `0x2D`, `kSlider` `0x2F`; the appliances `kGuitar` `0x64`,
`kCinderBlock` `0x6B`, `kFlowerBox` `0x6C`, `kCDs` `0x6D`, `kCustomPict` `0x6E`; and
`kCobweb` `0x79`.

**`kTriggerLinkOnly` (4)** (`:133-161`) — 13 unconditional destinations:
`kGreaseRt` `0x28`, `kGreaseLf` `0x29`, `kToaster` `0x62`, `kGuitar` `0x64`,
`kCoffee` `0x66`, `kOutlet` `0x67`, `kBalloon` `0x71`, `kCopterLf` `0x72`,
`kCopterRt` `0x73`, `kDartLf` `0x74`, `kDartRt` `0x75`, `kDrip` `0x77`, `kFish` `0x78`.

Plus a **conditional** sub-case for the six switches
(`kLightSwitch` `0x41` … `kInvisSwitch` `0x46`):

```
case kLightSwitch: … case kInvisSwitch:
if (linkRoom == thisRoomNumber)
    HiliteControl(linkControl, kControlActive);
break;
```
(`GliderPRO/Sources/Link.c:149-157`)

i.e. a trigger may fire a switch **only if the switch is in the same room as the trigger**.
Note the missing `else` — when `linkRoom != thisRoomNumber` the control is left in
**whatever state it was already in**, which may be *active* from a previous selection. A Go
port should write `inactive` explicitly.

**`kTransportLinkOnly` (5)** (`:171-189`) — 13 destinations:
`kMailboxLf` `0x33`, `kMailboxRt` `0x34`, `kCeilingTrans` `0x36`, `kInvisTrans` `0x3F`,
`kDeluxeTrans` `0x40`, `kInvisLight` `0x58`, `kOzma` `0x81`, `kMirror` `0x82`,
`kFireplace` `0x84`, `kWallWindow` `0x86`, `kCalendar` `0x88`, `kBulletin` `0x8B`,
`kCloud` `0x8C`.

`kFloorTrans` `0x35` is a legal *source* but **not** a legal destination — a floor duct
always sends the glider to a ceiling duct, never to another floor duct. The clutter items
(`kOzma`, `kMirror`, `kFireplace`, `kWallWindow`, `kCalendar`, `kBulletin`, `kCloud`) and
`kInvisLight` appear here as pure "arrival markers": they carry no link fields of their own
(they are `clutterType` / `lightType`), they only supply a landing coordinate.

### 17.7 `DoLink`

`GliderPRO/Sources/Link.c:270-320`. Not conditional on `objActive` at all — that gating is
entirely `UpdateLinkControl`'s job.

```
 1  if (!GetRoomFloorSuite(thisRoomNumber, &floor, &suite))  return   // silently
 2  floor += kNumUndergroundFloors                                    // +8
 3  if (thisRoomNumber == linkRoom):
 4      target = thisRoom->objects[linkObject]                        // staging copy
 5  else:
 6      HLock(thisHouse)
 7      target = (*thisHouse)->rooms[linkRoom].objects[linkObject]    // in the handle
 8  if (linkerIsSwitch):
 9      target.data.e.where = MergeFloorSuite(floor, suite)
10      target.data.e.who   = objActive
11  else:
12      target.data.d.where = MergeFloorSuite(floor, suite)
13      target.data.d.who   = objActive
14  fileDirty = true
15  UpdateMenus(false)
16  CloseLinkWindow()
```

The `.e` vs `.d` distinction is **cosmetic**: `where` sits at payload offset 6-7 and `who`
at offset 8 in *both* `switchType` and `transportType`
(`GliderPRO/Headers/GliderStructs.h:44-62`), so the two branches emit identical machine
code. A Go port can collapse them.

Failure behaviour worth noting:

* Step 1: if the current room is a tombstone (`suite == kRoomIsEmpty`),
  `GetRoomFloorSuite` returns false and the entire body — **including
  `CloseLinkWindow()`** — is skipped. Clicking `Link` does nothing whatsoever: no beep, no
  message, and the windoid stays up.
* Step 10/13: `who` is assigned from `objActive`, a `short`, into a `Byte`. If `objActive`
  were a glider pseudo-selection (−2/−3/−4) the stored `who` would be 254/253/252; if
  `kNoObjectSelected` (−1), **255**. `UpdateLinkControl` guards only against −1, and for
  −2/−3/−4 it does `thisRoom->objects[objActive].what`, reading 24/36/48 bytes *before*
  `objects[0]` (room offset 60): `objects[-2].what` aliases `tiles[0]`, and
  `objects[-3]`/`objects[-4]` alias bytes inside `name`. Those reads stay inside the
  348-byte `roomType` so they will not fault, but they are garbage; if the garbage happens
  to match a whitelisted type, `Link` activates and a nonsense `who` is written.
* `linkObject` is a `Byte` initialised to 255 by `OpenLinkWindow`. If `DoLink` ran with
  `linkObject` still 255 it would write 12 × 255 = 3060 bytes past `objects[0]`, i.e. far
  outside the room and into the next rooms of the house handle. In practice all three
  callers assign it immediately after `OpenLinkWindow`, so this cannot happen — but
  `Unlink` has the identical exposure and *is* clickable at any time (§17.8).

### 17.8 `DoUnlink`

`GliderPRO/Sources/Link.c:325-361`. Same structure, writing the sentinels:

```
where = −1
who   = 255
fileDirty = true
UpdateMenus(false)
CloseLinkWindow()
```

There is no `GetRoomFloorSuite` guard here, and no check that `linkRoom`/`linkObject` are
valid. `unlinkControl` is never hilited inactive anywhere in the program.

### 17.9 `HandleLinkClick` and the commit

```
void HandleLinkClick (Point wherePt)
{
    if (linkWindow == nil) return;
    SetPortWindowPort(linkWindow);
    GlobalToLocal(&wherePt);
    part = FindControl(wherePt, linkWindow, &theControl);
    if ((theControl != nil) && (part != 0))
    {
        part = TrackControl(theControl, wherePt, nil);
        if (part != 0)
        {
            if      (theControl == linkControl)   DoLink();
            else if (theControl == unlinkControl) DoUnlink();

            if (thisRoomNumber == linkRoom)
                CopyThisRoomToRoom();
            GenerateRetroLinks();
        }
    }
}
```
(`GliderPRO/Sources/Link.c:366-395`)

`FindControl` returns 0 for an inactive control, so a click on a greyed-out `Link` button
falls through harmlessly. The `CopyThisRoomToRoom()` is what flushes the same-room case
from `thisRoom` into the house handle; the cross-room case wrote straight into the handle
and needs no flush. Note that `linkRoom` is compared *after* `DoLink`/`DoUnlink` already
called `CloseLinkWindow()` — `linkRoom` is not reset by `CloseLinkWindow`, so the
comparison is still meaningful.

### 17.10 `GenerateRetroLinks` — the reverse index

`GliderPRO/Sources/House.c:538-613`. Builds a per-slot answer to "what links *into* this
object?" for the current room, so the Object Info dialogs can offer a **Linked From?**
button (DITL item 15).

```
retroLink retroLinkList[kMaxRoomObs];       // 24 entries × 4 bytes
typedef struct { short room; short object; } retroLink;
```
(`GliderPRO/Headers/GliderStructs.h:341-345`; the array is declared in
`GliderPRO/Sources/House.c:31`.)

```
 1  for i in 0..23: retroLinkList[i].room = −1
 2  for r in 0..nRooms-1:
 3      for i in 0..23:
 4          what = rooms[r].objects[i].what
 5          if what is a switch-style source and data.e.where != −1:
 6              ExtractFloorSuite(data.e.where, &floor, &suite)
 7              if GetRoomNumber(floor, suite) == thisRoomNumber:
 8                  objectLinked = (short)data.e.who
 9                  if retroLinkList[objectLinked].room == −1:
10                      retroLinkList[objectLinked] = { r, i }
11          if what is a transport-style source and data.d.where != −1:
12              ...identical with data.d...
```

Two things to carry into a port:

* Only the **first** in-bound link per slot is recorded (the `== −1` test at step 9), so
  the dialog's "Linked From?" shows one arbitrary predecessor even if several exist.
* **Step 8/9 is an out-of-bounds array access.** `objectLinked` is a `Byte` widened to
  `short`, so it ranges 0..255, and there is no bounds check before
  `retroLinkList[objectLinked]`. With `who == 255` the code reads and writes
  `retroLinkList[255]`, i.e. bytes 1020..1023 of a 96-byte array — 928 bytes past the end,
  straight into whatever globals follow it.

  This is reachable in the **shipped data**. Scanning all 22 houses for links whose target
  room resolves (so the `roomLinked == thisRoomNumber` test can fire) *and* whose `who`
  exceeds 23 yields exactly **13 hits, all in "CD Demo House"**, and every one of them is a
  self-link (the encoded room is the object's own room):

  | room | slot | `what` | `where` | decoded (floor, suite) | `who` |
  |--:|--:|---|--:|---|--:|
  | 33 | 8 | `kInvisTrans` `0x3F` | 1410 | (2, 14) | 255 |
  | 72 | 22 | `kMailboxRt` `0x34` | 2109 | (1, 21) | **35** |
  | 90 | 3 | `kCeilingTrans` `0x36` | 3210 | (2, 32) | 255 |
  | 123 | 7 | `kInvisTrans` `0x3F` | 4509 | (1, 45) | 255 |
  | 139 | 2 | `kLightSwitch` `0x41` | 4809 | (1, 48) | 255 |
  | 142 | 1 | `kMailboxLf` `0x33` | 5109 | (1, 51) | 255 |
  | 143 | 19 | `kCeilingTrans` `0x36` | 5209 | (1, 52) | 255 |
  | 148 | 3 | `kInvisTrans` `0x3F` | 5010 | (2, 50) | 255 |
  | 151 | 2 | `kInvisTrans` `0x3F` | 5107 | (−1, 51) | 255 |
  | 192 | 4 | `kCeilingTrans` `0x36` | 720 | (12, 7) | 255 |
  | 197 | 5 | `kCeilingTrans` `0x36` | 1121 | (13, 11) | 255 |
  | 200 | 5 | `kCeilingTrans` `0x36` | 1120 | (12, 11) | 255 |
  | 202 | 13 | `kCeilingTrans` `0x36` | 920 | (12, 9) | 255 |

  Because `GenerateRetroLinks` is called from `ReflectCurrentRoom`
  (`GliderPRO/Sources/Room.c:329`) on **every room change**, simply opening CD Demo House
  in the editor and navigating to any of those 13 rooms performs the stray write. A Go port
  must clamp: `if objectLinked > 23 { continue }`.

### 17.11 Link preservation during object compaction

`SortHouseObjects` (`GliderPRO/Sources/House.c:448-503`) is run before every save
(`GliderPRO/Sources/Menu.c:454`, `:378`) and closes the holes in every room's 24-slot
object array while keeping links pointing at the right objects.

```
 1  SpinCursor(3)
 2  CopyThisRoomToRoom()
 3  numLinks = CountHouseLinks()
 4  if numLinks == 0:  return                          // <-- see below
 5  linksList = NewPtr(sizeof(linksType) * numLinks)    // 8 bytes each
 6  if linksList == nil: RedAlert(kErrNoMemory)
 7  GenerateLinksList()
 8  HLock(thisHouse)
 9  for r in 0..nRooms-1:
10      for i in 0..23: srcLocations[i] = −1; destLocations[i] = −1
11      for i in 0..23:
12          for l in 0..numLinks-1:
13              if linksList[l].srcRoom  == r and linksList[l].srcObj  == i: srcLocations[i]  = l
14              if linksList[l].destRoom == r and linksList[l].destObj == i: destLocations[i] = l
15      SortRoomsObjects(r)
16      if (r & 0x0007) == 0x0007: IncrementCursor()
17  SpinCursor(3);  HSetState(thisHouse)
18  DisposePtr(linksList)
19  ForceThisRoom(thisRoomNumber)
```

`linksType` is `{ short srcRoom, srcObj, destRoom, destObj; }` = **8 bytes**
(`GliderPRO/Headers/GliderStructs.h:281-285`). `srcLocations[24]` and
`destLocations[24]` are `short` arrays declared in `GliderPRO/Sources/House.c:29`.

`GenerateLinksList` (`:317-385`) fills `linksList` with, for each link,
`srcRoom = r`, `srcObj = i`, `destRoom = GetRoomNumber(ExtractFloorSuite(where))`,
`destObj = (short)who`. Note `destRoom` may be **−1** for the 189 unresolvable links and
`destObj` may be **up to 255** — both are then simply never matched at steps 13-14, so
those links are left alone (their slot indices are *not* fixed up). That is a real data-loss
path: compacting a room that contains a broken link silently mis-points it.

`SortRoomsObjects(which)` (`:393-440`) is the actual bubble-down:

```
 1  probe = 0;  busy = true
 2  do:
 3      if rooms[which].objects[probe].what == kObjectIsEmpty:
 4          probe2 = probe + 1;  looking = true
 5          do:
 6              if rooms[which].objects[probe2].what != kObjectIsEmpty:
 7                  rooms[which].objects[probe] = rooms[which].objects[probe2]
 8                  rooms[which].objects[probe2].what = kObjectIsEmpty
 9                  if srcLocations[probe2]  != −1: linksList[srcLocations[probe2]].srcObj  = probe
10                  if destLocations[probe2] != −1:
11                      linksList[destLocations[probe2]].destObj = probe
12                      room = linksList[destLocations[probe2]].srcRoom
13                      obj  = linksList[destLocations[probe2]].srcObj
14                      rooms[room].objects[obj].data.e.who = probe      // patch the source
15                  fileDirty = true;  looking = false
16              probe2++
17              if probe2 >= kMaxRoomObs and looking: looking = false; busy = false
18          while looking
19      probe++
20      if probe >= (kMaxRoomObs − 1): busy = false
21  while busy
```

Step 14 writes through `data.e` even when the source is a transport, which is again
harmless because `who` is at payload offset 8 in both variants.

Step 20 stops at `probe == 23`, so the **last slot is never itself probed as a hole** — a
room whose only gap is at index 23 is already compact, so this is correct.

Steps 9-14 patch only `who`, never `where`. That is fine because compaction never moves
objects between rooms.

**The `numLinks == 0` early return (step 4) is a real bug**, and it is observable in the
shipped data. It aborts before the compaction loop *and* before the trailing
`ForceThisRoom(thisRoomNumber)` at step 19 — so for a link-free house neither the
compaction nor the room reload happens.

Empirical proof: scanning all 4070 real rooms in all 22 houses for "an empty slot that
precedes a non-empty slot" finds **exactly 6 rooms, all in "California or Bust!"**:

| room | first empty slot | occupied slots after it | `numObjects` field |
|--:|--:|---|--:|
| 3 | 18 | 23 | 19 |
| 4 | 1 | 2…22 (21 objects) | 22 |
| 5 | 15 | 23 | 16 |
| 7 | 20 | 23 | 21 |
| 9 | 20 | 23 | 21 |
| 13 | 12 | 23 | 13 |

"California or Bust!" is one of only two houses with **zero** links (the other is "Empty
House", which happens to already be compact). Every one of the other 20 houses — all of
which have links, and therefore ran the full `SortHouseObjects` — is perfectly compact.
Note that `numObjects` is nonetheless correct in all six rooms (it is a *count*, not a
high-water mark, and `MakeSureNumObjectsJives` recomputes it — §18.4), so the gap is
invisible to every consumer except code that stops at the first empty slot.

### 17.12 Following a link in the editor

Two navigation helpers in `GliderPRO/Sources/ObjectEdit.c`:

```
void GoToObjectInRoom (short object, short floor, short suite)   // :2769
{
    if (RoomExists(suite, floor, &itsNumber))
    {
        if (itsNumber != thisRoomNumber)
        {
            DeselectObject();
            CopyRoomToThisRoom(itsNumber);
            ReflectCurrentRoom(false);
        }
        else
            DeselectObject();

        if (thisRoom->objects[object].what != kObjectIsEmpty)
        {
            objActive = object;
            // start a handled or plain marquee around roomObjectRects[objActive]
        }
        UpdateMenus(false);
    }
}

void GoToObjectInRoomNum (short object, short roomNum)           // :2803
{
    if (GetRoomFloorSuite(roomNum, &floor, &suite))
        GoToObjectInRoom(object, floor, suite);
}
```

`GoToObjectInRoom` is what the switch/trigger/transport dialogs' **Go To** button
(`kGotoButton2`) calls, with `(who, floor, suite)` from the object's own link;
`GoToObjectInRoomNum` is what the **Linked From?** button calls, with
`(retroLinkList[objActive].object, retroLinkList[objActive].room)`.

Neither bounds-checks `object`, so a `who` of 255 in the data would index
`thisRoom->objects[255]`. The dialogs mitigate this by calling
`MyDisableControl(infoDial, kGotoButton2)` when `who == 255`
(`GliderPRO/Sources/ObjectInfo.c:1306-1307`, `:1432-1433`, and the transport equivalent)
— but only for exactly 255,
not for 24..254. The single `who == 35` link in CD Demo House (room 72, slot 22) therefore
has a live **Go To** button that would read 12 × 35 = 420 bytes past `objects[0]`.

### 17.13 The `where == −100` anomaly

**165 of the 2474 links carry `where == −100` and, in every single case, `who == 255`.**
Distribution:

| House | count |
|---|--:|
| Slumberland | 69 |
| Land of Illusion | 48 |
| Rainbow's End | 36 |
| Castle o' the Air | 12 |

By source type: `kCeilingTrans` 110, `kInvisTrans` 38, `kFloorTrans` 12,
`kMailboxRt` 2, `kThermostat` 1, `kKnifeSwitch` 1, `kLightSwitch` 1.

`ExtractFloorSuite(−100)` with C truncating division gives `suite = −1`,
`floor = 0 − 8 = −8`. Both values are one step outside the ranges `ValidateRoomNumbers`
enforces, and `suite = −1` is `kRoomIsEmpty`, so `GetRoomNumber(−8, −1)` cannot match any
real room. Effectively these are "no link", but they are *not* the `−1` sentinel, so
`CountHouseLinks` counts them, `GenerateLinksList` allocates entries for them,
`GenerateRetroLinks` walks them, and `ExtractFloorSuite` is invoked on them.

`DoLink` cannot produce this value: `MergeFloorSuite` is only ever called with
`floor = realFloor + 8` and `suite` from a room that `GetRoomFloorSuite` reported as valid,
and `GetRoomFloorSuite` fails for `suite == kRoomIsEmpty`. Running the v1→v2 conversion
formula backwards does not yield −100 either. The provenance is therefore undetermined
from this source tree (see `## Open questions`).

**Porting requirement:** treat *any* `where` that does not resolve to an existing room as
"no link", and `who > 23` as "no object" — both silently, without an error, or 189 links in
the shipped houses will trip the loader.

### 17.14 Constant summary for §17

| Constant | Value | Location |
|---|--:|---|
| `kLinkControlID` | 130 | `GliderPRO/Sources/Link.c:15` |
| `kUnlinkControlID` | 131 | `GliderPRO/Sources/Link.c:16` |
| `kSwitchLinkOnly` | 3 | `GliderPRO/Headers/GliderDefines.h:463` |
| `kTriggerLinkOnly` | 4 | `:464` |
| `kTransportLinkOnly` | 5 | `:465` |
| `kNumUndergroundFloors` | 8 | `:535` |
| `kMaxRoomObs` | 24 | `:250` |
| `kRoomIsEmpty` / `kObjectIsEmpty` / `kNoObjectSelected` | −1 | `:525-527` |
| `kWindoidWDEF` | 2048 | `:531` |
| `kHouseVersion` | `0x0200` | `:517` |
| link window rect | 0,0,129,30 (`QSetRect` l,t,r,b) | `GliderPRO/Sources/Link.c:220` |
| "no link" `where` | −1 | `GliderPRO/Sources/Link.c:333` etc. |
| "no object" `who` | 255 | `GliderPRO/Sources/Link.c:334` etc. |
| `kSoundTrigger` default `where` | 3000 | `GliderPRO/Sources/ObjectAdd.c:499` |
| `kMaxSoundTriggers` (per **room**) | 1 | `GliderPRO/Sources/ObjectAdd.c:17` |
| `kGotoButton1` / `kGotoButton2` / `kLinkTransButton` | 11 / 14 / 6 | `GliderPRO/Sources/ObjectInfo.c:61`, `:62`, `:47` |

---

## 18. Validation: `CheckHouseForProblems` and `KeepObjectLegal`

Everything in this section lives in one file, `GliderPRO/Sources/HouseLegal.c` (1219 lines). The
file contains exactly twelve functions: one public per-object legalizer (`KeepObjectLegal`), one
public whole-house driver (`CheckHouseForProblems`), and ten file-static helpers that the driver
calls in a fixed order. The ten helpers are forward-declared at the top of the file
(`GliderPRO/Sources/HouseLegal.c:16-26`) in exactly the order in which the driver invokes them —
except that `WrapBannerAndTrailer` is first in the declaration list and also first in the driver.

The whole file except `KeepObjectLegal`'s `unchanged = true; ... return (unchanged);` skeleton and
`CheckHouseForProblems`'s empty shell is wrapped in `#ifndef COMPILEDEMO`
(`GliderPRO/Sources/HouseLegal.c:51`/`:594`, `:603`/`:1046`, `:1054`/`:1217`). `COMPILEDEMO` is
commented out in the shipping build (`GliderPRO/Headers/GliderDefines.h:12`), so all of it is live.

### 18.1 Where validation runs

There are two entirely separate validation paths, and it is important not to confuse them.

| Path | Trigger | Scope |
|---|---|---|
| `KeepObjectLegal()` | after **every** object edit — drag, handle drag, add, duplicate, info-dialog OK | the one object `thisRoom->objects[objActive]` |
| `CheckHouseForProblems()` | only from `WriteHouse(true)` (`GliderPRO/Sources/HouseIO.c:469-470`) | the whole `thisHouse` handle |

`CheckHouseForProblems` has exactly one call site
(`GliderPRO/Sources/HouseIO.c:470`, inside `WriteHouse`, gated on the `checkIt` parameter):

```
	CopyThisRoomToRoom();

	if (checkIt)
		CheckHouseForProblems();
```

`WriteHouse` has six call sites, and only two of them pass `true` unconditionally:

| Call site | `checkIt` | Meaning |
|---|---|---|
| `GliderPRO/Sources/Menu.c:462` — `iSave` menu item | `true` | House ▸ Save House (⌘S) |
| `GliderPRO/Sources/HouseIO.c:613` — `QuerySaveChanges`, user hit **Save** | `true` | the "Save Changes?" alert |
| `GliderPRO/Sources/HouseIO.c:538` — `CloseHouse`, `gameDirty` | `theMode == kEditMode` | validate only if editing |
| `GliderPRO/Sources/SavedGames.c:348` — `SaveGame(doSave)` | `theMode == kEditMode` | validate only if editing |
| `GliderPRO/Sources/HouseIO.c:337` — `ReadHouse` flushing a dirty house before re-read | `false` | never validate |
| `GliderPRO/Sources/HouseIO.c:282` — inside `SaveHouseAs` | `false` | dead: body commented out (`:244-306`) |

So: **the only way to run the house validator is to save a dirty house.** There is no
"Check House" menu item. `iSave` first calls `DeselectObject()`, then `SortHouseObjects()` if
`fileDirty`, then only writes if `fileDirty && houseUnlocked`
(`GliderPRO/Sources/Menu.c:452-467`):

```
	case iSave:
	DeselectObject();
	if (fileDirty)
		SortHouseObjects();
	if ((fileDirty) && (houseUnlocked))
	{
//		SaveGame(false);
		if (wasHouseVersion < kHouseVersion)
			ConvertHouseVer1To2();
		wasHouseVersion = kHouseVersion;
		whoCares = WriteHouse(true);
		ForceThisRoom(thisRoomNumber);
		ReadyBackground(thisRoom->background, thisRoom->tiles);
		GetThisRoomsObjRects();
		DrawThisRoomsObjects();
	}
	break;
```

Note the ordering consequence: link compaction (`SortHouseObjects`, §17.9) runs **before**
`CompressHouse` (§18.8) renumbers rooms, and `CompressHouse` does *not* fix up `linksList` —
see §18.20.

`KeepObjectLegal()` has 22 call sites:

| File / line | Context |
|---|---|
| `GliderPRO/Sources/ObjectEdit.c:165`, `:174`, `:186`, `:203`, `:213`, `:230`, `:246`, `:259`, `:274`, `:290`, `:302`, `:313`, `:322`, `:334` | `DragHandle()`, one per handle-bearing object family (§8) |
| `GliderPRO/Sources/ObjectEdit.c:747` | `DragObject()` (§7) |
| `GliderPRO/Sources/ObjectEdit.c:1352` | `DuplicateObject()` (§11) |
| `GliderPRO/Sources/ObjectEdit.c:1700` | `MoveObject()` — dead, `BUILD_ARCADE_VERSION` |
| `GliderPRO/Sources/ObjectAdd.c:774` | `AddNewObject()` (§9) |
| `GliderPRO/Sources/ObjectInfo.c:1015`, `:1045`, `:1223` | three object-info dialogs that edit geometry |
| `GliderPRO/Sources/HouseLegal.c:939` | `KeepAllObjectsLegal()` |

At 14 of those sites the return value is thrown away into a variable literally named
`whoCares`; at three more (`ObjectEdit.c:747`, `:1352`, `:1700`) the code is
`if (KeepObjectLegal()) { }` — an empty `if` body. Only `KeepAllObjectsLegal` actually consumes
the result.

### 18.2 Module state

```c
short		houseErrors, wasRoom;			// GliderPRO/Sources/HouseLegal.c:29
Boolean		isHouseChecks;					// GliderPRO/Sources/HouseLegal.c:30

extern	short		numberRooms;			// GliderPRO/Sources/HouseLegal.c:32
```

| Global | Type | Meaning |
|---|---|---|
| `houseErrors` | `short` | running error counter; reset to 0 by the driver at various points, never at others (see §18.4) |
| `wasRoom` | `short` | the room number the editor was parked in; `CompressHouse` rewrites it if it moves that room |
| `isHouseChecks` | `Boolean` | user preference — "Error check house" |
| `numberRooms` | `short` (in `GliderPRO/Sources/Room.c`) | mirror of `(*thisHouse)->nRooms`; kept in sync by `ValidateNumberOfRooms` and `LopOffExtraRooms` |

`isHouseChecks` is a saved preference, restored at launch and defaulted to `true`:

| Site | Code |
|---|---|
| `GliderPRO/Sources/Main.c:86` | `isHouseChecks = thePrefs.wasHouseChecks;` |
| `GliderPRO/Sources/Main.c:151` | `isHouseChecks = true;` (no-prefs default) |
| `GliderPRO/Sources/Main.c:243` | `thePrefs.wasHouseChecks = isHouseChecks;` (save) |
| `GliderPRO/Sources/Settings.c:245` | `wasErrorCheck = isHouseChecks;` (Preferences dialog snapshot) |
| `GliderPRO/Sources/Settings.c:275` | `isHouseChecks = wasErrorCheck;` (Cancel restores) |
| `GliderPRO/Sources/Settings.c:1229` | `isHouseChecks = true;` (Use Defaults) |

So the default for a new user is **checks on**.

### 18.3 The message window

Validation reports progress through a bare 256 × 48 pixel document window with no scroll bars and
no close box, created and destroyed on the fly. Three functions, all in
`GliderPRO/Sources/WindowUtils.c`:

```c
#define kMessageWindowTall	48				// GliderPRO/Sources/WindowUtils.c:15

void OpenMessageWindow (StringPtr title)	// GliderPRO/Sources/WindowUtils.c:107
{
	Rect		mssgWindowRect;

	SetRect(&mssgWindowRect, 0, 0, 256, kMessageWindowTall);
	if (thisMac.hasColor)
		mssgWindow = NewCWindow(nil, &mssgWindowRect,
				title, false, noGrowDocProc, kPutInFront, false, 0L);
	else
		mssgWindow = NewWindow(nil, &mssgWindowRect,
				title, false, noGrowDocProc, kPutInFront, false, 0L);

	if (mssgWindow != nil)
	{
		ShowWindow(mssgWindow);
		SetPort((GrafPtr)mssgWindow);
		ClipRect(&mssgWindowRect);
		ForeColor(blackColor);
		BackColor(whiteColor);
		TextFont(systemFont);
	}
}

void SetMessageWindowMessage (StringPtr message)	// GliderPRO/Sources/WindowUtils.c:135
{
	Rect		mssgWindowRect;

	if (mssgWindow != nil)
	{
		SetPort((GrafPtr)mssgWindow);
		SetRect(&mssgWindowRect, 0, 0, 256, kMessageWindowTall);
		InsetRect(&mssgWindowRect, 16, 16);
		EraseRect(&mssgWindowRect);
		MoveTo(mssgWindowRect.left, mssgWindowRect.bottom - 6);
		DrawString(message);
	}
}

void CloseMessageWindow (void)				// GliderPRO/Sources/WindowUtils.c:154
{
	if (mssgWindow != nil)
		DisposeWindow(mssgWindow);
	mssgWindow = nil;
}
```

Facts a port must reproduce:

| Fact | Value | Citation |
|---|---|---|
| window content rect | `(0, 0, 256, 48)` — top, left, bottom, right | `GliderPRO/Sources/WindowUtils.c:111` |
| window definition | `noGrowDocProc` (procID 4) | `:112`, `:115` |
| `behind` | `kPutInFront` | `:113`, `:116` |
| `goAwayFlag` | `false` | `:113`, `:116` |
| `visible` | `false`, then explicit `ShowWindow` | `:112`/`:120` |
| window is **never positioned** | left/top stay at global (0, 0) → under the menu bar, top-left of the main screen | — |
| font | `systemFont` (Chicago 12) | `:126` |
| text area | window rect inset by (16, 16) → `(16, 16, 32, 240)` | `:142-143` |
| baseline | `MoveTo(16, 32 - 6)` = `(16, 26)` | `:144` |
| draw | `EraseRect` then `DrawString`; no clipping of over-long strings beyond the port `ClipRect` | `:143`, `:145` |
| colour | caller sets `ForeColor(redColor)` / `ForeColor(blueColor)` *before* calling and restores `blackColor` after | e.g. `GliderPRO/Sources/HouseLegal.c:1095-1097` |

`mssgWindow` is a global `WindowPtr`. The same trio is used by the copy/paste stubs
(ten sites, `GliderPRO/Sources/ObjectEdit.c:823` through `:1140`) and by the v1→v2 house
conversion (`GliderPRO/Sources/House.c:756`).

Toolbox notes for a Go port: `noGrowDocProc` is a classic titled document window without a size
box; `ForeColor(redColor)` selects an entry from the 8-colour QuickDraw-1 palette, not an RGB
value — the 8-bit CLUT index actually used depends on the current `GDevice`'s colour table. The
window is drawn to synchronously from inside a loop with no event pump at all, so on the original
it is not redrawn if obscured; a Go port with a compositing UI toolkit should simply render a
modal progress panel.

### 18.4 The driver: `CheckHouseForProblems`

`GliderPRO/Sources/HouseLegal.c:1052-1218`. Straight-line code, no loops, no error returns.

```
 1  houseErrors = 0                                            :1058
 2  CopyThisRoomToRoom()          // flush the live room back   :1059
 3  wasRoom  = thisRoomNumber                                   :1060
 4  wasActive = objActive                                       :1061
 5  OpenMessageWindow(localized 24)   // "Checking House File"  :1062-1063
 6  SpinCursor(3)                                               :1065
 7  SetMessageWindowMessage(localized 25)  // "Checking House…"  :1066-1067
 8  WrapBannerAndTrailer()                                      :1068

 9  if isHouseChecks:                                           :1070
10      SpinCursor(3)
11      msg(localized 26)             // "Checking Rooms…"
12      ValidateNumberOfRooms()
13      if houseErrors != 0:
14          msg(localized 27)         // "Room Number Errors"
15          DelayTicks(60); houseErrors = 0

16  if isHouseChecks:                                           :1085
17      SpinCursor(3); houseErrors = 0
18      CheckDuplicateFloorSuite()
19      if houseErrors != 0:
20          msg(NumToString(houseErrors) + localized 28) in RED  // " Duplicate Floor/Suites"
21          DelayTicks(45)

22  SpinCursor(3); CompressHouse()                              :1102-1103
23  SpinCursor(3); LopOffExtraRooms()                           :1104-1105

24  if isHouseChecks:                                           :1107
25      SpinCursor(3)                 // NOTE: houseErrors NOT reset here
26      ValidateRoomNumbers()
27      if houseErrors != 0:
28          msg(NumToString(houseErrors) + localized 29) in RED  // " Room Errors"
29          DelayTicks(60)

30  if isHouseChecks:                                           :1123
31      SpinCursor(3); houseErrors = 0
32      CountUntitledRooms()
33      if houseErrors != 0:
34          msg(NumToString(houseErrors) + localized 30) in BLUE // " 'Untitled' Rooms"
35          DelayTicks(45)

36  if isHouseChecks:                                           :1140
37      SpinCursor(3); houseErrors = 0
38      CheckRoomNameLength()
39      if houseErrors != 0:
40          msg(NumToString(houseErrors) + localized 31) in BLUE // " Room Names Too Long"
41          DelayTicks(45)

42  if isHouseChecks:                                           :1157
43      SpinCursor(3); houseErrors = 0
44      MakeSureNumObjectsJives()
45      if houseErrors != 0:
46          msg(NumToString(houseErrors) + localized 32) in RED  // " Room's # of Objects Wrong"
47          DelayTicks(60)

48  if isHouseChecks:                                           :1174
49      SpinCursor(3); houseErrors = 0
50      msg(localized 33)             // "Checking Objects…"
51      KeepAllObjectsLegal()
52      if houseErrors != 0:
53          msg(NumToString(houseErrors) + localized 34) in RED  // " Object Errors"
54          DelayTicks(60)

55  if isHouseChecks:                                           :1193
56      SpinCursor(3); houseErrors = 0
57      CheckForStaircasePairs()      // reports its own messages, never touches houseErrors

58  if isHouseChecks:                                           :1200
59      SpinCursor(3)
60      if CountStarsInHouse() < 1:
61          msg(localized 35) in RED  // "You have no stars in the house!"
62          DelayTicks(60)

63  InitCursor()                                                :1213
64  CloseMessageWindow()                                        :1214
65  ForceThisRoom(wasRoom)                                      :1215
66  objActive = wasActive                                        :1216
```

Message construction for the counted checks is always the same three lines (shown here from
step 20, `GliderPRO/Sources/HouseLegal.c:1092-1094`):

```c
	NumToString((long)houseErrors, message);
	GetLocalizedString(28, message2);
	PasStringConcat(message, message2);
```

so the displayed text is the decimal count immediately followed by the localized string, which is
why strings 28-32 and 17/18 all begin with a leading space (`" Duplicate Floor/Suites"`).
`GetLocalizedString(index, buf)` is `GetIndString(buf, 150, index)`
(`GliderPRO/Sources/StringUtils.c:321-327`, `#define kLocalizedStringsID 150`).

Summary of the nine `isHouseChecks` blocks:

| # | Line | Helper | Resets `houseErrors` first? | Localized string on failure | Colour | Delay (ticks) |
|---|---|---|---|---|---|---|
| 1 | `:1070` | `ValidateNumberOfRooms` | no (driver zeroed it at `:1058`) | 27 `Room Number Errors` | black | 60 |
| 2 | `:1085` | `CheckDuplicateFloorSuite` | **yes** (`:1088`) | 28 ` Duplicate Floor/Suites` | red | 45 |
| — | `:1102` | `CompressHouse` + `LopOffExtraRooms` | **runs unconditionally** | — | — | — |
| 3 | `:1107` | `ValidateRoomNumbers` | **NO** — inherits block 2's count | 29 ` Room Errors` | red | 60 |
| 4 | `:1123` | `CountUntitledRooms` | yes (`:1126`) | 30 ` 'Untitled' Rooms` | blue | 45 |
| 5 | `:1140` | `CheckRoomNameLength` | yes (`:1143`) | 31 ` Room Names Too Long` | blue | 45 |
| 6 | `:1157` | `MakeSureNumObjectsJives` | yes (`:1160`) | 32 ` Room's # of Objects Wrong` | red | 60 |
| 7 | `:1174` | `KeepAllObjectsLegal` | yes (`:1177`) | 34 ` Object Errors` | red | 60 |
| 8 | `:1193` | `CheckForStaircasePairs` | yes (`:1196`), but the helper never increments it | 20/21/22/23, per occurrence | red | 60 each |
| 9 | `:1200` | `CountStarsInHouse() < 1` | n/a | 35 `You have no stars in the house!` | red | 60 |

Note that step 1 reports its message in whatever colour is current (black) and *without* a count
prefix — it is the only counted check whose message is a bare string.

Two structural consequences a port must preserve:

* `CompressHouse` and `LopOffExtraRooms` run **whether or not** `isHouseChecks` is set. Turning
  error checking off does not turn off room compaction. Every save renumbers rooms.
* Block 3 (`ValidateRoomNumbers`) does not zero `houseErrors`, so if block 2 found *n* duplicate
  floor/suites and block 3 finds none, the user still sees "*n* Room Errors" in red for 60 ticks.
  If block 2 is skipped because `isHouseChecks` is false, block 3 is skipped too, so the bug is
  only reachable with checks on — i.e. always, by default.

`SpinCursor(3)` (`GliderPRO/Sources/AnimCursor.c:236-246`) advances an `acur`-driven animated
cursor three frames, `Delay(1, …)` between each — so each `SpinCursor(3)` costs at least 3 ticks
(50 ms). There are 13 `SpinCursor(3)` calls, so the validator burns ~39 ticks of pure cursor
animation on a clean house. `InitCursor()` at `:1213` restores the arrow.

### 18.5 `WrapBannerAndTrailer` and `WrapText`

```c
void WrapBannerAndTrailer (void)			// GliderPRO/Sources/HouseLegal.c:604
{
	char		wasState;

	wasState = HGetState((Handle)thisHouse);
	HLock((Handle)thisHouse);

	WrapText((*thisHouse)->banner, 40);
	WrapText((*thisHouse)->trailer, 64);

	HSetState((Handle)thisHouse, wasState);
}
```

| Field | Max chars per line | Citation |
|---|---|---|
| `(*thisHouse)->banner` (`Str255` at house offset 16) | **40** | `GliderPRO/Sources/HouseLegal.c:611` |
| `(*thisHouse)->trailer` (`Str255` at house offset 272) | **64** | `GliderPRO/Sources/HouseLegal.c:612` |

`WrapText` (`GliderPRO/Sources/StringUtils.c:216-250`) rewrites the Pascal string **in place**,
converting spaces to carriage returns (`kReturnKeyASCII` = 0x0D). It never changes the length byte
and never inserts characters.

```
 1  lastChar = theText[0]                    // Pascal length byte
 2  count = 0
 3  repeat
 4      chars = 0; foundEdge = false; foundSpace = false
 5      repeat
 6          count++; chars++
 7          if theText[count] == CR  then foundEdge  = true
 8          elif theText[count] == ' ' then { foundSpace = true; spaceIs = count }
 9      until count >= lastChar or chars >= maxChars or foundEdge
10      if (not foundEdge) and (count < lastChar) and foundSpace then
11          theText[spaceIs] = CR
12          count = spaceIs + 1
13  until count >= lastChar
```

Behaviour notes:

* The inner loop examines at most `maxChars` characters (`chars` is incremented before the test),
  so an emitted line is never longer than `maxChars`.
* If the window contained a CR already (`foundEdge`), the pass ends there and the next pass
  measures from the character after it — pre-existing hard breaks are respected and reset the
  column counter.
* If the window contains no space at all — a single word longer than `maxChars` — nothing is
  changed and the scan simply continues into the next window. **Long words overflow silently.**
* The break is inserted at the **last** space in the window, not the first, which is what makes
  it a greedy word wrap.
* Because the space is *overwritten*, wrapping is destructive and idempotent-ish but lossy: run
  it once with `maxChars = 40` and the spaces are gone; you cannot re-flow to a wider column.

Empirically, all 22 shipped houses are already wrapped within their limits — the longest banner
line is 39 characters (California or Bust!, and Titanic) and the longest trailer line is 63
(The Asylum Pro), both under the 40/64 caps. Observed values (python3 over the 22 data forks,
splitting the Pascal strings on `\r`):

| House | banner len | longest banner line | trailer len | longest trailer line |
|---|---|---|---|---|
| Art Museum | 0 | 0 | 67 | 34 |
| CD Demo House | 138 | 38 | 58 | 58 |
| California or Bust! | 166 | **39** | 41 | 39 |
| Castle o' the Air | 79 | 30 | 66 | 33 |
| Davis Station | 0 | 0 | 50 | 50 |
| Demo House | 111 | 35 | 111 | 54 |
| Empty House | 141 | 37 | 55 | 55 |
| Fun House | 0 | 0 | 12 | 6 |
| Grand Prix | 0 | 0 | 99 | 59 |
| ImagineHouse PRO II | 0 | 0 | 77 | 56 |
| In The Mirror | 0 | 0 | 54 | 54 |
| Land of Illusion | 0 | 0 | 96 | 47 |
| Leviathan | 0 | 0 | 105 | 58 |
| Metropolis | 0 | 0 | 93 | 49 |
| Nemo's Market | 0 | 0 | 47 | 45 |
| Rainbow's End | 0 | 0 | 51 | 49 |
| Sampler | 29 | 29 | 16 | 16 |
| Slumberland | 94 | 34 | 99 | 42 |
| SpacePods | 0 | 0 | 153 | 61 |
| Teddy World | 0 | 0 | 112 | 43 |
| The Asylum Pro | 159 | 36 | 78 | **63** |
| Titanic | 153 | **39** | 80 | 56 |

Fourteen of the 22 houses ship with an **empty banner** (length byte 0) and a non-empty trailer.

### 18.6 `ValidateNumberOfRooms`

`GliderPRO/Sources/HouseLegal.c:621-640`.

```
 1  reportsRooms = (*thisHouse)->nRooms
 2  countedRooms = (GetHandleSize(thisHouse) - sizeof(houseType)) / sizeof(roomType)
 3  if reportsRooms != countedRooms:
 4      (*thisHouse)->nRooms = (short)countedRooms
 5      numberRooms = (*thisHouse)->nRooms
 6      houseErrors++
```

with `sizeof(houseType)` = **866** and `sizeof(roomType)` = **348** under
`#pragma options align=mac68k` (`GliderPRO/Headers/Externs.h:231`; layouts in §3). The handle is
sized by `ReadHouse` as `NewHandle(byteCount)` where `byteCount` came from `GetEOF`
(`GliderPRO/Sources/HouseIO.c:341`, `:356`) — i.e. it is exactly the file length, not a multiple
of 348.

The division truncates, so up to 347 stray trailing bytes are absorbed without complaint. This is
not hypothetical: **Sampler**'s data fork is 1564 bytes, two more than `866 + 348 × 2 = 1562`, and
the two extra bytes are `01 01`. `(1564 - 866) / 348 = 698 / 348 = 2`, which equals its `nRooms`,
so no error is reported, `LopOffExtraRooms` finds no trailing tombstones and never shrinks the
handle, and `WriteHouse` writes `GetHandleSize` = 1564 bytes back out
(`GliderPRO/Sources/HouseIO.c:473`, `:491`). The two junk bytes survive every save/load cycle
forever. Verified: all other 21 houses have `(len - 866) % 348 == 0`.

Note also that `nRooms` is a signed `short` and the assignment `(short)countedRooms` truncates a
`long`; a >11 MB house (`866 + 348 × 32768`) would wrap negative. `ValidateRoomNumbers` (§18.10)
catches `nRooms < 0` afterwards, but only in the same `isHouseChecks` run.

### 18.7 `CheckDuplicateFloorSuite`

`GliderPRO/Sources/HouseLegal.c:646-682`. Detects two rooms occupying the same map cell.

```c
	#define		kRoomsTimesSuites	8192
```

```
 1  pidgeonHoles = NewPtrClear(sizeof(char) * 8192)   // 8192 bytes, zeroed
 2  if pidgeonHoles == nil: return                    // silent no-op on low memory
 3  for i in 0 .. nRooms-1:
 4      if rooms[i].suite == kRoomIsEmpty: continue
 5      bitPlace = ((rooms[i].floor + 7) * 128) + rooms[i].suite
 6      if bitPlace < 0 or bitPlace >= 8192: DebugStr("\pBlew array")
 7      if pidgeonHoles[bitPlace] != 0:
 8          houseErrors++
 9          rooms[i].suite = kRoomIsEmpty            // DESTROY the duplicate
10      else:
11          pidgeonHoles[bitPlace]++
```

| Fact | Value | Citation |
|---|---|---|
| table size | 8192 bytes (`kRoomsTimesSuites`), one `char` per cell | `GliderPRO/Sources/HouseLegal.c:648`, `:653` |
| index formula | `((floor + 7) * 128) + suite` | `:665-666` |
| implied legal floor range | `-7 … 56` (7 + 56 = 63; 63 × 128 + 127 = 8191) | derived |
| implied legal suite range | `0 … 127` | derived |
| bounds violation | `DebugStr("\pBlew array")` **and then indexes anyway** | `:667-668` |
| duplicate action | second and later occurrences become tombstones (`suite = -1`) | `:672` |

Three things to carry into a port:

1. The bounds check calls `DebugStr` (which drops into MacsBug in a debugging build and is a
   no-op otherwise) and then **falls through and writes out of bounds regardless** — there is no
   `continue`/`return`. A room with `floor = 100` gives `bitPlace = 13696`, a 5.5 KB heap
   overwrite. `ValidateRoomNumbers`, which would have caught the bad floor, runs *after* this
   (driver step 3 vs step 2). A Go port must range-check before indexing.
2. `NewPtrClear` failing is treated as "no duplicates", not as an error.
3. The "winner" of a duplicate pair is the room with the lower index; all higher-indexed
   duplicates are silently deleted, taking their objects and links with them. There is no
   confirmation dialog.

Empirically: **0 duplicate floor/suite pairs across all 4070 rooms of the 22 shipped houses**, and
every floor is inside `-7 … 56` and every suite inside `0 … 127`, so `bitPlace` never leaves the
table for shipped data.

### 18.8 `CompressHouse`

`GliderPRO/Sources/HouseLegal.c:688-734`. Moves live rooms down into tombstone slots so that the
room array is dense. This is the function that makes room numbers unstable across saves.

```
 1  wasFirstRoom = (*thisHouse)->firstRoom     // snapshot, because we rewrite firstRoom
 2  compressing = true
 3  roomNumber = nRooms - 1                    // start at the LAST room
 4  do
 5      if rooms[roomNumber].suite != kRoomIsEmpty:      // a live room
 6          probe = 0
 7          probing = true
 8          do
 9              if rooms[probe].suite == kRoomIsEmpty:   // found a hole
10                  rooms[probe] = rooms[roomNumber]     // 348-byte struct copy
11                  rooms[roomNumber].suite = kRoomIsEmpty
12                  if roomNumber == wasFirstRoom:  (*thisHouse)->firstRoom = probe
13                  if roomNumber == wasRoom:       wasRoom = probe
14                  probing = false
15              probe++
16              if probing and probe >= roomNumber:
17                  probing    = false               // no hole below us
18                  compressing = false              // and therefore none anywhere
19          while probing
20      roomNumber--
21      if roomNumber <= 0: compressing = false
22  while compressing
```

Properties:

* It is a **reverse-and-fill** compaction, not a stable shift. The last live room is moved into
  the first hole, then the second-to-last into the next hole, and so on. **Relative room order is
  reversed for the moved rooms.** A house with holes at 0 and 1 and live rooms at 2, 3 ends up as
  `[old3, old2, tombstone, tombstone]`.
* `probe++` at `:718` runs *after* the copy, so `probe` is left one past the hole; harmless
  because `probing` is already false.
* Termination: the outer loop stops when `probe >= roomNumber` (no hole strictly below the
  current room) or when `roomNumber <= 0`. Note `roomNumber <= 0`, so room 0 is never itself
  scanned as a source — correct, since there is nowhere below it.
* `firstRoom` is fixed up via the `wasFirstRoom` snapshot, and the editor's parked room via the
  global `wasRoom` (which the driver set from `thisRoomNumber` at `:1060` and restores through
  `ForceThisRoom(wasRoom)` at `:1215`).
* **Nothing else is fixed up.** Every cross-room link (`data.d.where` / `data.e.where`) encodes a
  floor/suite pair, not a room index (§17.4), so links survive; but `linksList`, built by
  `GenerateLinksList` in terms of room *indices* (`GliderPRO/Sources/House.c:317-385`), is stale
  the moment `CompressHouse` runs. In the `iSave` path `SortHouseObjects()` has already run and
  finished before `WriteHouse`, so this is latent rather than live — see §18.20.
* Cost is O(nRooms²) struct copies in the worst case, each 348 bytes, with `thisHouse` locked.

Empirically: **0 tombstones in any of the 22 shipped houses** (all 4070 rooms are live), so
`CompressHouse` is a no-op on all shipped data. It has visible effect only on a house the user has
deleted rooms from during the current session.

### 18.9 `LopOffExtraRooms`

`GliderPRO/Sources/HouseLegal.c:740-779`. Shrinks the handle to drop trailing tombstones.

```
 1  count = 0
 2  r = (*thisHouse)->nRooms
 3  do
 4      r--
 5      if rooms[r].suite == kRoomIsEmpty: count++
 6      else: r = 0                                  // break out of the loop
 7  while r > 0
 8  if count > 0:
 9      r = nRooms - count
10      newSize = sizeof(houseType) + (sizeof(roomType) * (long)r)     // 866 + 348*r
11      HUnlock(thisHouse)
12      SetHandleSize(thisHouse, newSize)
13      if MemError() != noErr:
14          ForeColor(redColor); SetMessageWindowMessage(localized 16)   // "Failed House Shrinkage"
15      HLock(thisHouse)
16      (*thisHouse)->nRooms -= count
17      numberRooms = (*thisHouse)->nRooms
```

| Fact | Value | Citation |
|---|---|---|
| new handle size | `866 + 348 × (nRooms − count)` | `GliderPRO/Sources/HouseLegal.c:765` |
| failure message | localized string 16, `Failed House Shrinkage`, red | `:770-772` |
| `houseErrors` | **never incremented**, even on failure | — |

Quirks:

* The `else r = 0;` at `:757-758` is a `break` written as an assignment; it works because the
  `while (r > 0)` test follows. But note the loop body decrements `r` *first*, so if
  `nRooms == 0` the first iteration reads `rooms[-1]` — 348 bytes before the room array, i.e. the
  tail of the house header (`hasGame`, `firstRoom`, `nRooms`, and 342 bytes of `savedGame` /
  `highScores`). With `nRooms == 0` the `while (r > 0)` test then fails immediately, but the
  out-of-bounds read has already happened and may have set `count = 1`, in which case
  `newSize = 866 + 348 × (0 − 1) = 518` — `SetHandleSize` to **less than the header**. This is
  reachable only if `nRooms == 0`, which `ReadHouse` rejects (`numberRooms < 1` →
  `YellowAlert(kYellowNoRooms, 0)`, `GliderPRO/Sources/HouseIO.c:385-392`) and
  `ValidateRoomNumbers` can produce (`nRooms < 0 → 0`, `:795-799`) — but that runs *after*
  `LopOffExtraRooms` in the driver. A Go port should guard `nRooms >= 1`.
* On `SetHandleSize` failure it prints the message but **still** subtracts `count` from `nRooms`,
  so the in-memory room count now claims fewer rooms than the handle holds. Benign (the extra
  bytes are just ignored and re-written), but a port that stores rooms in a slice must not shrink
  the slice on the error path only.
* `HUnlock` / `SetHandleSize` / `HLock` around the resize is mandatory on a Mac heap; a Go port
  reslicing `[]Room` needs none of it.

Empirically: **0 trailing tombstones in all 22 shipped houses**, so `count == 0` and this function
never fires on shipped data.

### 18.10 `ValidateRoomNumbers`

`GliderPRO/Sources/HouseLegal.c:785-828`.

```
 1  numRooms = (*thisHouse)->nRooms
 2  if numRooms < 0:  (*thisHouse)->nRooms = 0;  numRooms = 0
 3  for i in 0 .. numRooms-1:
 4      if rooms[i].suite == kRoomIsEmpty: continue
 5      if rooms[i].floor > 56 or rooms[i].floor < -7:
 6          rooms[i].suite = kRoomIsEmpty
 7          RED; msg(localized 17)  // " Floor Number Bad"
 8          houseErrors++; BLACK
 9      if rooms[i].suite >= 128 or rooms[i].suite < 0:      // NOT else-if
10          rooms[i].suite = kRoomIsEmpty
11          RED; msg(localized 18)  // " Suite Number Bad"
12          houseErrors++; BLACK
```

| Field | Legal range | Action on violation | Citation |
|---|---|---|---|
| `floor` | `-7 … 56` inclusive | room becomes a tombstone, `houseErrors++` | `GliderPRO/Sources/HouseLegal.c:804-813` |
| `suite` | `0 … 127` inclusive | room becomes a tombstone, `houseErrors++` | `:814-823` |

These are exactly the ranges implied by `CheckDuplicateFloorSuite`'s
`((floor + 7) * 128) + suite < 8192`, and by `kMaxNumRoomsH 128` / `kMaxNumRoomsV 64`
(`GliderPRO/Headers/GliderDefines.h:543-544`). Note the asymmetry: 64 vertical map slots
(`kMaxNumRoomsV`) but a legal floor range spanning 64 values `-7 … 56`, of which `-7 … -1` are the
"underground" floors — `kNumUndergroundFloors` is **8**
(`GliderPRO/Headers/GliderDefines.h:535`), and the link encoding adds 8 to the floor
(§17.4), giving `1 … 64` for `where`'s floor component. `kMapGroundValue` is 56, so
`v = 56 - floor` maps floor 56 → row 0 and floor −7 → row 63.

Quirks:

1. Because the two tests are separate `if`s rather than `else if`, a room with **both** a bad
   floor and a bad suite is counted twice and the message flashes twice. (The second test reads
   the suite *after* the first test may have set it to −1, and −1 < 0, so a bad-floor room always
   also trips the bad-suite test: `houseErrors` is incremented **twice** for a single bad-floor
   room, and localized string 18 is what the user actually sees.)
2. The messages are drawn and immediately overwritten by the next iteration with no delay — the
   driver only delays once, after the whole loop. On a house with many bad rooms the user sees a
   flicker, not a list.
3. `houseErrors` is *not* zeroed by the driver before this block (§18.4 step 3), so the reported
   count is `duplicates + floor-errors + suite-errors`.

Empirically: all 4070 shipped rooms pass — floors observed span `-7 … 39` and every suite is
in `0 … 127`, so no violation is ever printed.

### 18.11 `CountUntitledRooms`

`GliderPRO/Sources/HouseLegal.c:834-851`.

```
 1  for i in 0 .. nRooms-1:
 2      if rooms[i].suite != kRoomIsEmpty and
 3         EqualString(rooms[i].name, "\pUntitled Room", false, true):
 4          houseErrors++
```

`EqualString(a, b, caseSensitive, diacriticalSensitive)` with `caseSensitive = false` and
`diacSensitive = true` — so the comparison is **case-insensitive** but diacritical-sensitive
(`GliderPRO/Sources/HouseLegal.c:846`). "untitled room" and "UNTITLED ROOM" both count.

This check is purely advisory: it counts, reports in **blue** for 45 ticks, and changes nothing.
The default name a new room gets is exactly `"\pUntitled Room"`
(see §16 and `GliderPRO/Sources/Room.c`), 13 characters.

Empirically: **53 rooms are still named "Untitled Room"** across the 22 shipped houses —
Empty House 34, CD Demo House 15, California or Bust! 2, Fun House 2. (Re-derived by
decoding the data forks and comparing the `Str27` room name against `"Untitled Room"`;
case-insensitive and exact comparison give the same 53, so no house uses a different
casing.) So this advisory check *does* fire on shipped content, and a Go port that
reports it will show a non-zero count on those four houses.

### 18.12 `CheckRoomNameLength`

`GliderPRO/Sources/HouseLegal.c:857-879`.

```
 1  for i in 0 .. nRooms-1:
 2      rooms[i].unusedByte = 0                       // UNCONDITIONAL, tombstones too
 3      if rooms[i].suite != kRoomIsEmpty and rooms[i].name[0] > 27:
 4          rooms[i].name[0] = 27                     // truncate in place
 5          houseErrors++
```

| Fact | Value | Citation |
|---|---|---|
| max room-name length | **27** characters | `GliderPRO/Sources/HouseLegal.c:871`, `:873` |
| `name` field | `Str27` at room offset 0, 28 bytes (1 length + 27) | §3, `GliderPRO/Headers/GliderStructs.h` |
| `unusedByte` | room offset **32**, forced to 0 for **every** room including tombstones | `:868` |
| report colour | blue, 45 ticks, string 31 | `GliderPRO/Sources/HouseLegal.c:1150-1153` |

Truncation is done by lowering the Pascal length byte only; the dropped characters remain in the
file as trailing garbage inside the 28-byte field. A Go port that stores names as `string` and
re-serialises with zero padding will produce byte-different (but semantically identical) files.

The unconditional `unusedByte = 0` at `:868` is the only place in the program that writes that
field, and it is why the field is reliably zero in practice. Empirically: `unusedByte == 0` in all
4070 shipped rooms, and **0 rooms with `name[0] > 27`**.

### 18.13 `MakeSureNumObjectsJives`

`GliderPRO/Sources/HouseLegal.c:885-913`.

```
 1  for i in 0 .. nRooms-1:
 2      if rooms[i].suite == kRoomIsEmpty: continue
 3      count = 0
 4      for h in 0 .. kMaxRoomObs-1:                       // 0..23
 5          if rooms[i].objects[h].what != kObjectIsEmpty: count++
 6      if count != rooms[i].numObjects:
 7          houseErrors++
 8          rooms[i].numObjects = count                    // trust the objects, not the count
```

`kMaxRoomObs` = 24 (`GliderPRO/Headers/GliderDefines.h:250`); `kObjectIsEmpty` = −1
(`:526`). `numObjects` lives at room offset 58 (§3).

Important: the count is a **population count over all 24 slots**, not a high-water mark. It is
therefore *not* an index bound — a room can legitimately have `numObjects = 19` with objects in
slots 0-17 and 23 (a "gapped" list). Code that treats `numObjects` as "objects live in slots
`0 … numObjects-1`" is wrong; `GliderPRO/Sources/ObjectAdd.c`'s `AddNewObject` and
`GliderPRO/Sources/House.c`'s `SortRoomsObjects` both scan all 24 slots.

Empirically: `numObjects` is correct in **all 4070** shipped rooms, including the six known gapped
rooms (all in *California or Bust!*):

| room index | first empty slot | occupied slots after it | `numObjects` |
|---|---|---|---|
| 3 | 18 | 23 | 19 |
| 4 | 1 | 2 … 22 | 22 |
| 5 | 15 | 23 | 16 |
| 7 | 20 | 23 | 21 |
| 9 | 20 | 23 | 21 |
| 13 | 12 | 23 | 13 |

Those gaps exist because *California or Bust!* has **zero links**, and `SortHouseObjects` returns
early when `numLinks == 0` (`GliderPRO/Sources/House.c:459-460`), skipping the slot compaction
entirely — see §17.9 and §21.

### 18.14 `KeepAllObjectsLegal`

`GliderPRO/Sources/HouseLegal.c:919-955`. Runs the per-object legalizer over every object in the
house, using the live-room globals as scratch space.

```
 1  numRooms = (*thisHouse)->nRooms
 2  for i in 0 .. numRooms-1:
 3      if rooms[i].suite == kRoomIsEmpty: continue
 4      ForceThisRoom(i)                       // *thisRoom = (*thisHouse)->rooms[i]
 5      for h in 0 .. kMaxRoomObs-1:           // 0..23, INCLUDING empty slots
 6          objActive = h
 7          if thisRoom->objects[objActive].what != kObjectIsEmpty:
 8              if !KeepObjectLegal():         // false == "I changed something"
 9                  RED; msg(localized 19)     // "Object Bad"
10                  houseErrors++; BLACK
11                  DelayTicks(60)             // one full second PER bad object
12      CopyThisRoomToRoom()                   // (*thisHouse)->rooms[i] = *thisRoom
```

Notes:

* `ForceThisRoom` (`GliderPRO/Sources/Room.c:369-385`) copies the 348-byte room out of the handle
  into the global `*thisRoom`, sets `previousRoom = thisRoomNumber` and `thisRoomNumber = i`, and
  `YellowAlert(kYellowIllegalRoomNum, 0)` if `roomNumber >= nRooms`.
  `CopyThisRoomToRoom` (`:354-365`) copies it back, but returns early if
  `noRoomAtAll || thisRoomNumber == -1`.
* `objActive` is assigned for all 24 slots even when the slot is empty, so after the function
  `objActive == 23` and `thisRoomNumber == ` the last live room. The driver restores both
  (`:1215-1216`).
* `KeepObjectLegal` returns `unchanged`, so the test `if (!KeepObjectLegal())` fires when the
  object **was** modified. The function's own comment ("returns true if any changes were made",
  `GliderPRO/Sources/HouseLegal.c:39-40`) is the opposite of what the code does; the code is
  authoritative — see §18.15.
* `DelayTicks(60)` per bad object means a house with 100 out-of-spec objects stalls for 100
  seconds with no way to cancel.

### 18.15 `CheckForStaircasePairs`

`GliderPRO/Sources/HouseLegal.c:961-1045`. The only check that inspects *relationships* between
rooms.

```
 1  for i in 0 .. nRooms-1:
 2      if rooms[i].suite == kRoomIsEmpty: continue
 3      for h in 0 .. 23:
 4          if rooms[i].objects[h].what == kUpStairs:              // 0x31
 5              thisRoomNumber = i                                // GLOBAL, never restored
 6              neighbor = GetNeighborRoomNumber(kNorthRoom)      // floor+1, same suite
 7              if neighbor == kRoomIsEmpty:
 8                  RED; msg(localized 20) "No room upstairs!";      BLACK; DelayTicks(60)
 9              else:
10                  hasStairs = any(rooms[neighbor].objects[g].what == kDownStairs for g in 0..23)
11                  if not hasStairs:
12                      RED; msg(localized 21) "No downstairs to match!"; BLACK; DelayTicks(60)
13          elif rooms[i].objects[h].what == kDownStairs:          // 0x32
14              thisRoomNumber = i
15              neighbor = GetNeighborRoomNumber(kSouthRoom)      // floor-1, same suite
16              if neighbor == kRoomIsEmpty:
17                  RED; msg(localized 22) "No room downstairs!";    BLACK; DelayTicks(60)
18              else:
19                  hasStairs = any(rooms[neighbor].objects[g].what == kUpStairs for g in 0..23)
20                  if not hasStairs:
21                      RED; msg(localized 23) "No upstairs to match!";  BLACK; DelayTicks(60)
```

`GetNeighborRoomNumber(which)` (`GliderPRO/Sources/Room.c:562-635`) is a linear search over
`numberRooms` for a room whose `(suite, floor)` equals the current room's plus a delta:

| `which` | constant | hDelta (suite) | vDelta (floor) |
|---|---|---|---|
| `kCentralRoom` | 0 | 0 | 0 |
| `kNorthRoom` | 1 | 0 | **+1** |
| `kNorthEastRoom` | 2 | +1 | +1 |
| `kEastRoom` | 3 | +1 | 0 |
| `kSouthEastRoom` | 4 | +1 | −1 |
| `kSouthRoom` | 5 | 0 | **−1** |
| `kSouthWestRoom` | 6 | −1 | −1 |
| `kWestRoom` | 7 | −1 | 0 |
| `kNorthWestRoom` | 8 | −1 | +1 |

(`GliderPRO/Headers/GliderDefines.h:217-225` for the constants.) Returns `kRoomIsEmpty` (−1) if
nothing matches. It does **not** skip tombstones, so a tombstone (`suite == -1`) can be returned
as a "neighbour" if the sought suite is −1; and it iterates `numberRooms`, the global mirror, not
`(*thisHouse)->nRooms`.

Quirks a port must know about:

* This check **never increments `houseErrors`.** The driver zeroes the counter at `:1196` and
  never looks at it afterwards, so staircase problems are reported one-at-a-time as flashing
  messages and never summarised. There is no aggregate "N staircase errors" line.
* It **clobbers the global `thisRoomNumber`** at `:980` and `:1011` and never restores it. The
  only reason the editor recovers is the driver's `ForceThisRoom(wasRoom)` at `:1215`.
* Only *existence* is checked, not position: an up-staircase at x = 0 pairs happily with a
  down-staircase at x = 448 in the room above. There is no coordinate check anywhere.
* Multiple up-staircases in one room each produce their own message; one down-staircase in the
  room above satisfies all of them.
* The lattice is per-suite: staircases only ever connect `(floor, suite)` to
  `(floor ± 1, suite)`.

Empirically: the 22 shipped houses contain **163 `kUpStairs` and 163 `kDownStairs`** and produce
**zero** messages — every up-staircase has a room above it containing a down-staircase, and vice
versa.

### 18.16 `CountStarsInHouse`

Lives in `GliderPRO/Sources/Banner.c:89-111`, not `HouseLegal.c`.

```c
short CountStarsInHouse (void)
{
	short		i, h, numRooms, numStars;
	char		wasState;

	numStars = 0;

	wasState = HGetState((Handle)thisHouse);
	HLock((Handle)thisHouse);
	numRooms = (*thisHouse)->nRooms;
	for (i = 0; i < numRooms; i++)
	{
		if ((*thisHouse)->rooms[i].suite != kRoomIsEmpty)
			for (h = 0; h < kMaxRoomObs; h++)
			{
				if ((*thisHouse)->rooms[i].objects[h].what == kStar)
					numStars++;
			}
	}
	HSetState((Handle)thisHouse, wasState);

	return (numStars);
}
```

`kStar` = `0x2C` (`GliderPRO/Headers/GliderDefines.h:355`), a prize-class object
(variant `c`, `bonusType`). Tombstones are skipped; all 24 slots are scanned.

The validator only tests `< 1` and only warns (`GliderPRO/Sources/HouseLegal.c:1203-1210`); it
never blocks the save. The same function is the game's win condition source:
`numStarsRemaining = CountStarsInHouse()` at `GliderPRO/Sources/Play.c:314`, and `kStarPoints`
is **5000** (`GliderPRO/Headers/GliderDefines.h:541`).

Observed star counts for the 22 shipped houses (python3 over the data forks, counting
`objects[h].what == 0x2C` in live rooms):

| stars | house |
|---|---|
| 6 | Art Museum |
| 9 | CD Demo House |
| 1 | California or Bust! |
| 4 | Castle o' the Air |
| 4 | Davis Station |
| 1 | Demo House |
| 1 | Empty House |
| **0** | **Fun House** |
| 3 | Grand Prix |
| 3 | ImagineHouse PRO II |
| 1 | In The Mirror |
| 5 | Land of Illusion |
| 6 | Leviathan |
| 4 | Metropolis |
| 5 | Nemo's Market |
| 5 | Rainbow's End |
| 1 | Sampler |
| 6 | Slumberland |
| 1 | SpacePods |
| 1 | Teddy World |
| 1 | The Asylum Pro |
| 1 | Titanic |

Total 69. **Fun House is the only shipped house that would trip localized string 35**
("You have no stars in the house!") — and it is also the only shipped house that would produce
*any* validator message at all.

### 18.17 `KeepObjectLegal` — the per-object legalizer

`GliderPRO/Sources/HouseLegal.c:42-597`. 556 lines, a single `switch (theObject->what)` with nine
arms (one per object class) plus a pre-switch special case for the initial glider. This is the
single most porting-sensitive function in the editor: it encodes every geometric invariant the
renderer and the physics engine rely on.

```c
Boolean KeepObjectLegal (void)
{
	objectType	*theObject;
	Rect		bounds, roomRect;
	short		direction, dist;
	char		wasState;
	Boolean		unchanged;

	unchanged = true;
	theObject = &thisRoom->objects[objActive];
	...
	return (unchanged);
}
```

**The return value is `unchanged`, i.e. `true` means "I did not touch it".** The doc comment at
`:38-40` says the opposite; the code wins. `KeepAllObjectsLegal` correctly tests `!KeepObjectLegal()`.

#### 18.17.1 The initial-glider arm (`:55-69`)

```
 1  if objActive == kInitialGliderSelected:            // -2
 2      HLock(thisHouse)
 3      if (*thisHouse)->initial.h < 0:  initial.h = 0
 4      if (*thisHouse)->initial.v < 0:  initial.v = 0
 5      if initial.h > (kRoomWide - kGliderWide):  initial.h = kRoomWide - kGliderWide
 6      if initial.v > (kTileHigh - kGliderHigh):  initial.v = kTileHigh - kGliderHigh
 7      HSetState(...)
 8      return (true)                                  // "unchanged" EVEN IF CLAMPED
```

| Constant | Value | Citation |
|---|---|---|
| `kRoomWide` | 512 | `GliderPRO/Headers/GliderDefines.h:499` |
| `kGliderWide` | 48 | `:548` |
| `kTileHigh` | 322 | `:498` |
| `kGliderHigh` | 20 | `:549` |
| ⇒ legal `initial.h` | **0 … 464** | derived |
| ⇒ legal `initial.v` | **0 … 302** | derived |

`initial` is a `Point` at house offset 12 (`v` at 12:14, `h` at 14:16 — **v first**, §3). The arm
returns unconditionally, so an initial glider position is never reported as an error even when
clamped. Note `objActive == kLeftGliderSelected` (−3) and `kRightGliderSelected` (−4) fall through
to `theObject = &thisRoom->objects[-3 / -4]` and then into the `switch` on whatever garbage `what`
is there — 36 or 48 bytes before the object array, i.e. inside `numObjects`/`openings`/`suite`.
In practice `DeselectObject`/`ObjectHasHandle` keep those two selections out of the drag paths
that call `KeepObjectLegal`, but a Go port should reject any negative `objActive` other than −2.

Empirically all 22 shipped houses have `initial` inside the legal box:

```
Art Museum          (h=231, v= 56)      Land of Illusion    (h=245, v=163)
CD Demo House       (h=223, v=105)      Leviathan           (h=229, v= 44)
California or Bust! (h=308, v= 62)      Metropolis          (h=245, v= 24)
Castle o' the Air   (h=239, v=124)      Nemo's Market       (h=342, v= 51)
Davis Station       (h=189, v= 95)      Rainbow's End       (h=237, v=200)
Demo House          (h= 49, v=107)      Sampler             (h=384, v= 85)
Empty House         (h= 83, v= 64)      Slumberland         (h=361, v=  7)
Fun House           (h= 42, v= 78)      SpacePods           (h= 30, v= 72)
Grand Prix          (h=268, v= 58)      Teddy World         (h=362, v= 41)
ImagineHouse PRO II (h=211, v= 35)      The Asylum Pro      (h= 78, v= 27)
In The Mirror       (h=424, v= 50)      Titanic             (h= 39, v= 97)
```

The maximum observed `h` is 424 (In The Mirror) — well under 464 — and the maximum `v` is 200
(Rainbow's End), under 302.

#### 18.17.2 The room rectangle and the geometry helpers

```c
	QSetRect(&roomRect, 0, 0, kRoomWide, kTileHigh);		// :71
```

`QSetRect(r, left, top, right, bottom)` (`GliderPRO/Sources/RectUtils.c`), so `roomRect` is
`{top = 0, left = 0, bottom = 322, right = 512}` — the full room interior, 512 × 322.

Every arm begins with

```c
	GetObjectRect(&thisRoom->objects[objActive], &bounds);
	if (ForceRectInRect(&bounds, &roomRect)) { ...write back... unchanged = false; }
```

`GetObjectRect` (`GliderPRO/Sources/ObjectRects.c:32`) turns the 10-byte payload into a screen
`Rect`. For the classes that matter here it is uniformly

```
	*itsRect = srcRects[who->what];
	ZeroRectCorner(itsRect);                 // slide to (0,0): RectUtils.c
	QOffsetRect(itsRect, topLeft.h, topLeft.v);
```

(blowers `GliderPRO/Sources/ObjectRects.c:43-61`, prizes `:91-110`, transports `:121-140`,
switches `:161-175`, non-strip lights `:177-188`, appliances `:200-218`, enemies `:239-253`), with
several per-type overrides. Because `KeepObjectLegal` writes the *result* of `ForceRectInRect` back
through the same encoding, a port must get every one of these right. The complete dispatch
(`GliderPRO/Sources/ObjectRects.c:37-272`), with `ZeroRectCorner` at
`GliderPRO/Sources/RectUtils.c:57`:

| `what` | lines | how the `Rect` is produced |
|---|---|---|
| `kObjectIsEmpty` (−1) | `:39-41` | `QSetRect(itsRect, 0, 0, 0, 0)` |
| `kFloorVent` … `kSewerBlower` (0x01-0x0F) | `:43-61` | `srcRects[what]`, zero-cornered, offset by `data.a.topLeft` |
| **`kLiftArea`** (0x10) | `:63-66` | `QSetRect(itsRect, 0, 0, data.a.distance, data.a.tall * 2)` then offset by `data.a.topLeft` — **`distance` is the width and `tall` is HALF the height** |
| — | `:68-71` | **dead code**: three statements after the `break`, never reachable |
| `kTable` … `kInvisBounce` (0x11-0x1F) | `:73-89` | `*itsRect = data.b.bounds` — verbatim, no `srcRects` |
| `kRedClock` … `kHelium` (0x21-0x2E, incl. `kStar`) | `:91-110` | `srcRects[what]`, zero-cornered, offset by `data.c.topLeft` |
| `kSlider` (0x2F) | `:112-119` | as above, then `itsRect->right = itsRect->left + data.c.length` |
| `kUpStairs` … `kWindowExLf` (0x31-0x3E) | `:121-140` | `srcRects[what]`, zero-cornered, offset by `data.d.topLeft` |
| **`kInvisTrans`** (0x3F) | `:142-150` | as above, then `bottom = top + data.d.tall` **and** `right += (short)data.d.wide` — so the width is `64 + wide` |
| **`kDeluxeTrans`** (0x40) | `:152-159` | `wide = (data.d.tall & 0xFF00) >> 8; tall = data.d.tall & 0x00FF; QSetRect(0, 0, wide * 4, tall * 4)` then offset |
| `kLightSwitch` … `kSoundTrigger` (0x41-0x49) | `:161-175` | `srcRects[what]`, zero-cornered, offset by `data.e.topLeft` |
| `kCeilingLight`, `kLightBulb`, `kTableLamp`, `kHipLamp`, `kDecoLamp`, `kInvisLight` | `:177-188` | `srcRects[what]`, zero-cornered, offset by `data.f.topLeft` |
| **`kFlourescent`, `kTrackLight`** | `:190-198` | `srcRects[what]`, zero-cornered, **`itsRect->right = data.f.length`**, *then* offset by `data.f.topLeft` — so final width **is** `length` |
| `kShredder` … `kCDs` (0x61-0x6D) | `:200-218` | `srcRects[what]`, zero-cornered, offset by `data.g.topLeft` |
| **`kCustomPict`** (0x6E) | `:220-237` | `GetPicture(data.g.height)` — **`height` is a PICT resource ID here**; on `nil` it writes `data.g.height = 10000` and falls back to `srcRects[kCustomPict]`; else uses `(*thePict)->picFrame` |
| `kBalloon` … `kCobweb` (0x71-0x79) | `:239-253` | `srcRects[what]`, zero-cornered, offset by `data.h.topLeft` |
| `kOzma` … `kChimes` (0x81-0x8F) | `:255-271` | `*itsRect = data.i.bounds` — verbatim |
| anything else | — | **no `default:` arm — `*itsRect` is left uninitialised** |

Two consequences worth flagging now:

* `kCustomPict` makes `GetObjectRect` a **mutating** call with a Resource Manager dependency. A Go
  port must keep the 10000 sentinel, because it is written into the house file and will be saved.
* `data.f.length` for strip lights is a **width**, and the assignment happens *before* the offset,
  which is why `KeepObjectLegal`'s `topLeft.h + length > bounds.right` test is self-consistent.

`ForceRectInRect(small, large)` (`GliderPRO/Sources/RectUtils.c:223-270`) returns `true` if it
changed anything:

```
 1  changed = false
 2  NormalizeRect(small)                       // swap l/r and t/b if inverted
 3  if height(small) > height(large): small->bottom = small->top + height(large); changed = true
 4  if width(small)  > width(large):  small->right  = small->left + width(large); changed = true
 5  hOff = large->left  - small->left;   if hOff > 0: OffsetRect(small, hOff, 0); changed = true
 6  hOff = large->right - small->right;  if hOff < 0: OffsetRect(small, hOff, 0); changed = true
 7  vOff = large->top    - small->top;   if vOff > 0: OffsetRect(small, 0, vOff); changed = true
 8  vOff = large->bottom - small->bottom;if vOff < 0: OffsetRect(small, 0, vOff); changed = true
 9  return changed
```

Semantics: **shrink to fit, then translate to be inside** — the object keeps its size wherever
possible and is *slid* back into the room rather than clipped. `NormalizeRect`
(`GliderPRO/Sources/RectUtils.c:34-51`) swaps `left`/`right` and `top`/`bottom` if inverted, so a
right-to-left drag is silently corrected. Note that `NormalizeRect` alone does not set `changed`,
so an inverted-but-in-bounds rect is normalised *silently*.

Also relevant: `RectWide(r)` = `right - left`, `RectTall(r)` = `bottom - top`,
`HalfRectWide(r)` = `(right - left) / 2` (`GliderPRO/Sources/RectUtils.c:79-82`).

#### 18.17.3 Arm 1 — Blowers (`:75-194`), variant `a` (`blowerType`)

Cases: `kFloorVent` 0x01, `kCeilingVent` 0x02, `kFloorBlower` 0x03, `kCeilingBlower` 0x04,
`kSewerGrate` 0x05, `kLeftFan` 0x06, `kRightFan` 0x07, `kTaper` 0x08, `kCandle` 0x09,
`kStubby` 0x0A, `kTiki` 0x0B, `kBBQ` 0x0C, `kInvisBlower` 0x0D, `kGrecoVent` 0x0E,
`kSewerBlower` 0x0F, `kLiftArea` 0x10.

```
 1  GetObjectRect(obj, &bounds)
 2  if ForceRectInRect(&bounds, &roomRect):
 3      data.a.topLeft.h = bounds.left ; data.a.topLeft.v = bounds.top ; unchanged = false
 4      if what == kLiftArea:
 5          data.a.distance = RectWide(&bounds)          // lift area encodes its width here
 6          data.a.tall     = RectTall(&bounds) / 2      // and HALF its height in the byte
                                                        // NOTE: tall is a Byte -> truncated mod 256
 7  if what == kStubby   and topLeft.h % 2 == 0:  topLeft.h-- ; unchanged = false   // force ODD
 8  if what in {kTaper,kCandle,kTiki,kBBQ} and topLeft.h % 2 != 0:
 9                                              topLeft.h-- ; unchanged = false   // force EVEN
10  if what == kFloorVent    and topLeft.v != 305: topLeft.v = 305; data.a.distance += 2   // SILENT
11  if what == kFloorBlower  and topLeft.v != 304: topLeft.v = 304; data.a.distance += 2   // SILENT
12  if what == kSewerGrate   and topLeft.v != 303: topLeft.v = 303; data.a.distance += 2   // SILENT
13  if what == kFloorTrans   and topLeft.v != 302: topLeft.v = 302; data.a.distance += 2   // UNREACHABLE
14  if ObjectHasHandle(&direction, &dist):
15      switch direction:
16        kAbove:  dist = bounds.top - dist
17                 if what in {kFloorVent,kFloorBlower,kTaper,kCandle,kStubby}:
18                     if dist < 36:  data.a.distance += dist - 36 ; unchanged = false
19                 else:
20                     if dist < 0:   data.a.distance += dist      ; unchanged = false
21        kToRight: dist = bounds.right + dist
22                 if dist > kRoomWide:  data.a.distance += (kRoomWide - dist) ; unchanged = false
23        kBelow:  dist = bounds.bottom + dist
24                 if dist > kTileHigh:  data.a.distance += (kTileHigh - dist) ; unchanged = false
25        kToLeft: dist = bounds.left - dist
26                 if dist < 0:  data.a.distance += dist ; unchanged = false
```

Constants:

| Constant | Value | Citation |
|---|---|---|
| `kFloorVentTop` | **305** | `GliderPRO/Headers/GliderDefines.h:467` |
| `kFloorBlowerTop` | **304** | `:469` |
| `kSewerGrateTop` | **303** | `:471` |
| `kFloorTransTop` | **302** | `:473` |
| minimum blast height for floor-standing blowers | **36** px | `GliderPRO/Sources/HouseLegal.c:150-152` |

`ObjectHasHandle(&direction, &dist)` (`GliderPRO/Sources/ObjectEdit.c:1791`, §8) returns, for
the blower family:

| Object(s) | `*direction` | `*dist` |
|---|---|---|
| `kFloorVent`, `kFloorBlower`, `kSewerGrate`, `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ`, `kGrecoVent`, `kSewerBlower` | `kAbove` (1) | `data.a.distance` |
| `kCeilingVent`, `kCeilingBlower` | `kBelow` (3) | `data.a.distance` |
| `kLeftFan` | `kToLeft` (4) | `data.a.distance` |
| `kRightFan` | `kToRight` (2) | `data.a.distance` |
| `kInvisBlower` | from `data.a.vector & 0x0F`: 1→`kAbove`, 2→`kToRight`, 4→`kBelow`, 8→`kToLeft` | `data.a.distance` |
| `kLiftArea` | `kBottomCorner` (5) | 0 |

Because `kLiftArea` reports `kBottomCorner`, none of the four `switch (direction)` cases match and
its `distance`/`tall` are left alone by step 14 onwards — the geometry was already recomputed at
step 5.

`kInvisBlower` with a `vector` nibble that is 0, 3, 5-7, or 9-15 leaves `direction`
**uninitialised** (the inner `switch` at `GliderPRO/Sources/ObjectEdit.c:1834-1851` has no `default`), so step 15
branches on a stack garbage value. A Go port must default it (`kAbove` is the safe choice, matching
nibble 1).

The 36-pixel minimum at step 18 is asymmetric: it applies to `kFloorVent`, `kFloorBlower`,
`kTaper`, `kCandle`, `kStubby` but **not** to `kSewerGrate`, `kTiki`, `kBBQ`, `kGrecoVent`,
`kSewerBlower`, which only get the `dist < 0` clamp. `data.a.distance += dist - 36` is a *relative*
adjustment, so it lands the blast top exactly 36 px below the room top.

#### 18.17.4 Arm 2 — Furniture (`:196-227`), variant `b` (`furnitureType`)

Cases: `kTable` 0x11 … `kInvisBounce` 0x1F (15 types; note 0x20 is unused).

```
 1  GetObjectRect(obj, &bounds)
 2  if ForceRectInRect(&bounds, &roomRect):  data.b.bounds = bounds ; unchanged = false
 3  if what == kManhole and ((bounds.left - 3) % 64) != 0:
 4      data.b.bounds.left  = (((bounds.left + 29) / 64) * 64) + 3
 5      data.b.bounds.right = data.b.bounds.left + RectWide(&srcRects[kManhole])
 6      unchanged = false
```

| Constant | Value | Citation |
|---|---|---|
| `kManhole` | 0x1D | `GliderPRO/Headers/GliderDefines.h:340` |
| `srcRects[kManhole]` | `QSetRect(…, 0, 0, 123, 22)` → 123 wide × 22 tall | `GliderPRO/Sources/StructuresInit2.c:353` |
| tile width `kTileWide` | 64 | `GliderPRO/Headers/GliderDefines.h:497` |

The manhole snaps to `left ≡ 3 (mod 64)`, i.e. left ∈ {3, 67, 131, 195, 259, 323, 387, 451}. The
rounding `((left + 29) / 64) * 64 + 3` is C integer division (truncating toward zero), so for
negative `left` it rounds the *wrong* way; `ForceRectInRect` has already forced `left >= 0` at that
point unless the manhole is wider than the room (123 < 512, so it cannot be). +29 is +32−3, i.e.
round-to-nearest of `(left − 3) / 64`.

Note the snap reads `bounds.left` (the local, post-`ForceRectInRect` copy) but writes
`data.b.bounds.left`/`.right`; `bounds` itself is not updated, which is harmless because nothing
reads it afterwards in this arm.

#### 18.17.5 Arm 3 — Prizes (`:229-280`), variant `c` (`bonusType`)

Cases: `kRedClock` 0x21 … `kSlider` 0x2F (15 types; 0x30 unused).

```
 1  GetObjectRect(obj, &bounds)
 2  if ForceRectInRect(&bounds, &roomRect):
 3      data.c.topLeft.h = bounds.left ; data.c.topLeft.v = bounds.top ; unchanged = false
 4  if      what == kGreaseRt and bounds.right + data.c.length > kRoomWide:
 5          data.c.length = kRoomWide - bounds.right ; unchanged = false
 6  else if what == kGreaseLf and bounds.left  - data.c.length < 0:
 7          data.c.length = bounds.left              ; unchanged = false
 8  else if what == kSlider   and bounds.left  + data.c.length > kRoomWide:
 9          data.c.length = kRoomWide - bounds.left  ; unchanged = false
10  if data.c.topLeft.h % 2 != 0:  topLeft.h-- ; unchanged = false            // ALL prizes: EVEN h
11  if what != kStar and data.c.length % 2 != 0:  length-- ; unchanged = false // EVEN length
```

Steps 4-9 are a single if/else-if chain, so at most one fires — irrelevant in practice because the
three conditions are mutually exclusive by `what`. Steps 10-11 are **unconditional** for the whole
class: every prize is snapped to an even x, and every prize except `kStar` has an even `length`.
`kStar` (0x2C) is exempted from the even-length rule because for a star `length` is not a length
at all — the star's payload uses `length` as an animation/spin field in some houses.

| Field | Payload offset | Note |
|---|---|---|
| `data.c.topLeft` | 0-3 (`v` 0:2, `h` 2:4) | |
| `data.c.length` | 4-5 | grease slick run, slider travel |
| `data.c.points` | 6-7 | prize score |
| `data.c.state` | 8 | |
| `data.c.initial` | 9 | |

`srcRects` used by the three length clamps
(`GliderPRO/Sources/StructuresInit2.c:370-383`): `kGreaseRt` 32 × 27 offset (0, 243),
`kGreaseLf` 32 × 27 offset (0, 324), `kSlider` 64 × 16, `kStar` 32 × 31 offset (48, 0).

#### 18.17.6 Arm 4 — Transports (`:282-385`), variant `d` (`transportType`)

Cases: `kUpStairs` 0x31 … `kDeluxeTrans` 0x40.

```
 1  GetObjectRect(obj, &bounds)
 2  if ForceRectInRect(&bounds, &roomRect):
 3      data.d.topLeft.h = bounds.left ; data.d.topLeft.v = bounds.top ; unchanged = false
 4      if what == kDeluxeTrans:
 5          data.d.tall = ((RectWide(&bounds) / 4) << 8) + (RectTall(&bounds) / 4)
 6  // --- four handedness auto-flips, ALL SILENT (never set unchanged = false) ---
 7  if what in {kDoorInLf, kDoorInRt}:
 8      if topLeft.h + HalfRectWide(&srcRects[kDoorInLf]) > kRoomWide/2:
 9          topLeft.h = kDoorInRtLeft (368) ; what = kDoorInRt
10      else: topLeft.h = kDoorInLfLeft (0) ; what = kDoorInLf
11  if what in {kDoorExRt, kDoorExLf}:
12      if topLeft.h + HalfRectWide(&srcRects[kDoorExRt]) > 256:
13          topLeft.h = kDoorExRtLeft (496) ; what = kDoorExRt
14      else: topLeft.h = kDoorExLfLeft (0) ; what = kDoorExLf
15  if what in {kWindowInLf, kWindowInRt}:
16      if topLeft.h + HalfRectWide(&srcRects[kWindowInLf]) > 256:
17          topLeft.h = kWindowInRtLeft (492) ; what = kWindowInRt
18      else: topLeft.h = kWindowInLfLeft (0) ; what = kWindowInLf
19  if what in {kWindowExRt, kWindowExLf}:
20      if topLeft.h + HalfRectWide(&srcRects[kWindowExRt]) > 256:
21          topLeft.h = kWindowExRtLeft (496) ; what = kWindowExRt
22      else: topLeft.h = kWindowExLfLeft (0) ; what = kWindowExLf
23  if what == kInvisTrans and topLeft.v + data.d.tall > kTileHigh:
24      data.d.tall = kTileHigh - topLeft.v ; unchanged = false
25  if what == kInvisTrans and data.d.wide < 0:
26      data.d.wide = 0 ; unchanged = false
```

`data.d.wide` is a `Byte`, so `data.d.wide < 0` at step 25 is **always false** on a Mac 68k/PPC
compiler where `Byte` is `unsigned char` — dead code. (`GliderStructs.h` declares
`Byte who, wide;` — `Byte` is `UInt8`.)

The four flip thresholds, with the widths read from `GliderPRO/Sources/StructuresInit2.c:393-400`:

| Pair | `srcRects` width | `HalfRectWide` | flip when `topLeft.h >` | right-hand `topLeft.h` | left-hand `topLeft.h` |
|---|---|---|---|---|---|
| `kDoorInLf` 0x37 / `kDoorInRt` 0x38 | 144 (both) | 72 | **184** | `kDoorInRtLeft` = **368** | `kDoorInLfLeft` = **0** |
| `kDoorExLf` 0x3A / `kDoorExRt` 0x39 | 16 (both) | 8 | **248** | `kDoorExRtLeft` = **496** | `kDoorExLfLeft` = **0** |
| `kWindowInLf` 0x3B / `kWindowInRt` 0x3C | 20 (both) | 10 | **246** | `kWindowInRtLeft` = **492** | `kWindowInLfLeft` = **0** |
| `kWindowExLf` 0x3E / `kWindowExRt` 0x3D | 16 (both) | 8 | **248** | `kWindowExRtLeft` = **496** | `kWindowExLfLeft` = **0** |

(Constants: `GliderPRO/Headers/GliderDefines.h:483-494`; `srcRects` heights are 322, 322, 170,
170 respectively and are not used by these tests. Note the code reads the `Lf` srcRect for the
interior door and interior window but the `Rt` srcRect for the exterior door and exterior window
— immaterial because each pair has identical widths, but a porter diffing the C should not
"fix" it.)

Behavioural consequences: doors and windows have **exactly two legal x positions each** and
`KeepObjectLegal` snaps them there and rewrites `what` to match the side. You cannot place an
interior door in the middle of a room; dragging it past x = 184 turns it into a right-hand door.
Because these blocks never set `unchanged = false`, the change is invisible to
`KeepAllObjectsLegal` and never counts as an "Object Error". They also run *every* time — even a
door already at 0 is "re-snapped" to 0 — so the branch is not conditional on being wrong.

`kDeluxeTrans`'s `tall` field is a **packed pair**: high byte = width/4, low byte = height/4
(step 5). With `srcRects[kDeluxeTrans]` = 64 × 64 (`StructuresInit2.c:402`), the default is
`(16 << 8) + 16` = 0x1010 = 4112.

`kInvisTrans`'s `tall` is a plain height in pixels, clamped so the object cannot extend past
`kTileHigh` = 322 (step 23-24). `srcRects[kInvisTrans]` is 64 × 32
(`GliderPRO/Sources/StructuresInit2.c:401`), but `GetObjectRect` also does
`itsRect->right += (short)data.d.wide` (`GliderPRO/Sources/ObjectRects.c:149`), so the actual width
is `64 + wide` and the actual height is `tall` (not 32). `data.d.wide` is a `Byte`
(`GliderPRO/Headers/GliderStructs.h`), i.e. 0-255, so an invisible transport spans 64-319 px
horizontally.

#### 18.17.7 Arm 5 — Switches (`:387-408`), variant `e` (`switchType`)

Cases: `kLightSwitch` 0x41, `kMachineSwitch` 0x42, `kThermostat` 0x43, `kPowerSwitch` 0x44,
`kKnifeSwitch` 0x45, `kInvisSwitch` 0x46, `kTrigger` 0x47, `kLgTrigger` 0x48,
`kSoundTrigger` 0x49.

```
 1  GetObjectRect(obj, &bounds)
 2  if ForceRectInRect(&bounds, &roomRect):
 3      data.e.topLeft.h = bounds.left ; data.e.topLeft.v = bounds.top ; unchanged = false
 4  if data.e.topLeft.h % 2 != 0:  topLeft.h-- ; unchanged = false      // force EVEN x
```

Nothing about links is validated here. `data.e.where` (payload 6:8) and `data.e.who` (payload 8)
are never range-checked by any function in `HouseLegal.c` — which is why the 189 dangling links
and the 180 out-of-range `who` values documented in §17.13 survive in shipped houses, and why
`GenerateRetroLinks`' `retroLinkList[objectLinked]` out-of-bounds write (§17.10) is reachable.
**A Go port that adds link validation here would be changing behaviour**; if you do add it, add it
as a separate opt-in check so that loading a shipped house does not mutate it.

#### 18.17.8 Arm 6 — Lights (`:410-465`), variant `f` (`lightType`)

Cases: `kCeilingLight` 0x51, `kLightBulb` 0x52, `kTableLamp` 0x53, `kHipLamp` 0x54,
`kDecoLamp` 0x55, `kFlourescent` 0x56, `kTrackLight` 0x57, `kInvisLight` 0x58.

```
 1  GetObjectRect(obj, &bounds)
 2  if ForceRectInRect(&bounds, &roomRect):
 3      if what in {kFlourescent, kTrackLight}:               // horizontal strip lights
 4          if data.f.topLeft.h < bounds.left: topLeft.h = bounds.left
 5          if data.f.topLeft.v < bounds.top:  topLeft.v = bounds.top
 6          if topLeft.h + data.f.length > bounds.right:
 7              data.f.length = bounds.right - topLeft.h      // SHRINK instead of slide
 8      else:
 9          data.f.topLeft.h = bounds.left ; data.f.topLeft.v = bounds.top
10      unchanged = false
11  if what in {kFlourescent, kTrackLight} and (bounds.right > kRoomWide or bounds.left < 0):
12      if data.f.topLeft.h < 0:        topLeft.h = 0        ; unchanged = false
13      if bounds.left < 0:             bounds.left = 0      ; unchanged = false
14      if data.f.topLeft.h > kRoomWide: topLeft.h = kRoomWide; unchanged = false
15      if bounds.right > kRoomWide:    bounds.right = kRoomWide ; unchanged = false
16      data.f.length = kRoomWide - bounds.left               // UNCONDITIONAL, SILENT
```

`lightType` payload: `topLeft` 0-3, `length` 4-5, `byte0` 6, `byte1` 7, `initial` 8, `state` 9.
`length` is a **width in pixels**, not an absolute right edge (see §21 — this was a documented
mis-reading; `ObjectEdit.c` treats it as a width at both of its sites).

The asymmetry at step 3-9 is the important part: strip lights (`kFlourescent`, `kTrackLight`) are
*shrunk* to fit, all other lights are *slid*. Constants for those two:
`kFlourescentTop` = 12, `kTrackLightTop` = 5
(`GliderPRO/Headers/GliderDefines.h:480-481`) — used by `ObjectAdd.c`, not here.

Step 11-16 is a second, partly redundant clamp block that runs whenever the *pre-clamp* bounds
stuck out of the room. Its last line is unconditional and does not set `unchanged = false`, so a
strip light that overhung the right wall by 1 px silently has its `length` recomputed from
`bounds.left`. Because step 13/15 have already clamped the local `bounds`, the result is
`length = 512 - max(bounds.left, 0)` — i.e. **the light is stretched to the right wall**, not
merely trimmed. That is almost certainly not what the author intended, but it is what a port must
reproduce to round-trip a house byte-identically.

#### 18.17.9 Arm 7 — Appliances (`:467-512`), variant `g` (`applianceType`)

Cases: `kShredder` 0x61, `kToaster` 0x62, `kMacPlus` 0x63, `kGuitar` 0x64, `kTV` 0x65,
`kCoffee` 0x66, `kOutlet` 0x67, `kVCR` 0x68, `kStereo` 0x69, `kMicrowave` 0x6A,
`kCinderBlock` 0x6B, `kFlowerBox` 0x6C, `kCDs` 0x6D, `kCustomPict` 0x6E.

```
 1  GetObjectRect(obj, &bounds)
 2  if ForceRectInRect(&bounds, &roomRect):
 3      data.g.topLeft.h = bounds.left ; data.g.topLeft.v = bounds.top ; unchanged = false
 4  if what == kToaster and bounds.top - data.g.height < 0:
 5      data.g.height = bounds.top ; unchanged = false        // toast cannot fly above the ceiling
 6  if what == kTV and data.g.topLeft.h % 2 == 0:
 7      topLeft.h-- ; unchanged = false                        // TV forced ODD
 8  if what in {kToaster,kMacPlus,kCoffee,kOutlet,kVCR,kStereo,kMicrowave}
 9        and data.g.topLeft.h % 2 != 0:
10      topLeft.h-- ; unchanged = false                        // these forced EVEN
```

`applianceType` payload: `topLeft` 0-3, `height` 4-5, `byte0` 6, `delay` 7, `initial` 8,
`state` 9. `height` is the toaster's launch height. `srcRects[kToaster]` is 48 × 27 offset
(0, 22) (`GliderPRO/Sources/StructuresInit2.c:431-432`).

Note `kTV` is the **only** object in the whole program forced to an odd x besides `kStubby`.
`kShredder`, `kGuitar`, `kCinderBlock`, `kFlowerBox`, `kCDs`, `kCustomPict` have **no** parity
constraint at all.

#### 18.17.10 Arm 8 — Enemies (`:514-554`), variant `h` (`enemyType`)

Cases: `kBalloon` 0x71, `kCopterLf` 0x72, `kCopterRt` 0x73, `kDartLf` 0x74, `kDartRt` 0x75,
`kBall` 0x76, `kDrip` 0x77, `kFish` 0x78, `kCobweb` 0x79.

```
 1  GetObjectRect(obj, &bounds)
 2  if ForceRectInRect(&bounds, &roomRect):
 3      data.h.topLeft.h = bounds.left ; data.h.topLeft.v = bounds.top ; unchanged = false
 4  if what in {kBall, kFish} and bounds.top - data.h.length < 0:
 5      data.h.length = bounds.top ; unchanged = false          // bounce/jump height
 6  if what == kDrip and bounds.bottom + data.h.length > kTileHigh:
 7      data.h.length = kTileHigh - bounds.bottom ; unchanged = false   // fall distance
 8  if what in {kBalloon,kCopterLf,kCopterRt,kBall,kDrip,kFish} and data.h.topLeft.h % 2 != 0:
 9      topLeft.h-- ; unchanged = false                          // force EVEN x
```

`enemyType` payload: `topLeft` 0-3, `length` 4-5, `delay` 6, `byte0` 7, `initial` 8, `state` 9.
`length` means "how far up" for `kBall`/`kFish` and "how far down" for `kDrip`.
`kDartLf`, `kDartRt`, `kCobweb` have no parity or length constraint.

`srcRects` (`GliderPRO/Sources/StructuresInit2.c:455-457`): `kBall` 32 × 32, `kDrip` 16 × 12,
`kFish` 36 × 33.

#### 18.17.11 Arm 9 — Clutter (`:556-590`), variant `i` (`clutterType`)

Cases: `kOzma` 0x81, `kMirror` 0x82, `kMousehole` 0x83, `kFireplace` 0x84, `kFlower` 0x85,
`kWallWindow` 0x86, `kBear` 0x87, `kCalendar` 0x88, `kVase1` 0x89, `kVase2` 0x8A,
`kBulletin` 0x8B, `kCloud` 0x8C, `kFaucet` 0x8D, `kRug` 0x8E, `kChimes` 0x8F.

```
 1  GetObjectRect(obj, &bounds)
 2  if ForceRectInRect(&bounds, &roomRect):  data.i.bounds = bounds ; unchanged = false
 3  if what == kMirror:
 4      if data.i.bounds.left  % 2 != 0:  bounds.left--  ; unchanged = false
 5      if data.i.bounds.right % 2 != 0:  bounds.right-- ; unchanged = false
```

`clutterType` payload: `bounds` 0-7 (`top` 0:2, `left` 2:4, `bottom` 4:6, `right` 6:8),
`pict` 8-9. The mirror is the only clutter object with a constraint, and it is the only object in
the program whose *right* edge is parity-constrained (because the mirror's reflection blit needs an
even byte alignment in the 8-bit pixmap).

Nothing after the `switch` — the function falls straight to `return (unchanged);` at `:596`. There
is **no `default:` arm**, so an object with an unrecognised `what` (0x00, 0x20, 0x30, 0x40, 0x4A-0x50,
0x59-0x60, 0x6F-0x70, 0x7A-0x80, 0x90+) is left completely alone and reports "unchanged".

#### 18.17.12 Complete table of every snap rule

| Object(s) | Rule | Sets `unchanged = false`? | Line |
|---|---|---|---|
| initial glider | `h` ∈ [0, 464], `v` ∈ [0, 302] | no (returns `true`) | `:59-68` |
| all 9 classes | `ForceRectInRect(bounds, {0,0,322,512})` | yes | per arm |
| `kLiftArea` | `distance = RectWide`, `tall = RectTall / 2` | yes (via the enclosing `if`) | `:99-100` |
| `kStubby` | `topLeft.h` **odd** | yes | `:103-107` |
| `kTaper`, `kCandle`, `kTiki`, `kBBQ` | `topLeft.h` **even** | yes | `:108-114` |
| `kFloorVent` | `topLeft.v = 305`, `distance += 2` | **no** | `:115-119` |
| `kFloorBlower` | `topLeft.v = 304`, `distance += 2` | **no** | `:120-125` |
| `kSewerGrate` | `topLeft.v = 303`, `distance += 2` | **no** | `:126-131` |
| `kFloorTrans` | `topLeft.v = 302`, `distance += 2` | **no** — and **unreachable** | `:132-137` |
| `kFloorVent`, `kFloorBlower`, `kTaper`, `kCandle`, `kStubby` | blast top ≥ 36 px from room top | yes | `:150-154` |
| other `kAbove` blowers | blast top ≥ 0 | yes | `:158-162` |
| `kRightFan`, `kInvisBlower`(→right) | blast right ≤ 512 | yes | `:166-173` |
| `kCeilingVent`, `kCeilingBlower`, `kInvisBlower`(→down) | blast bottom ≤ 322 | yes | `:175-182` |
| `kLeftFan`, `kInvisBlower`(→left) | blast left ≥ 0 | yes | `:184-191` |
| `kManhole` | `left ≡ 3 (mod 64)`, `right = left + 123` | yes | `:217-226` |
| `kGreaseRt` | `right + length ≤ 512` | yes | `:251-256` |
| `kGreaseLf` | `left - length ≥ 0` | yes | `:257-262` |
| `kSlider` | `left + length ≤ 512` | yes | `:263-268` |
| **all prizes** | `topLeft.h` **even** | yes | `:269-273` |
| all prizes except `kStar` | `length` **even** | yes | `:274-279` |
| `kDeluxeTrans` | `tall = (wide/4 << 8) + (tall/4)` | yes (via the enclosing `if`) | `:306-307` |
| `kDoorInLf`/`kDoorInRt` | `h` = 0 or 368, `what` matched to side; threshold `h > 184` | **no** | `:310-324` |
| `kDoorExLf`/`kDoorExRt` | `h` = 0 or 496; threshold `h > 248` | **no** | `:325-339` |
| `kWindowInLf`/`kWindowInRt` | `h` = 0 or 492; threshold `h > 246` | **no** | `:340-354` |
| `kWindowExLf`/`kWindowExRt` | `h` = 0 or 496; threshold `h > 248` | **no** | `:355-369` |
| `kInvisTrans` | `topLeft.v + tall ≤ 322` | yes | `:371-378` |
| `kInvisTrans` | `wide ≥ 0` — dead, `wide` is unsigned | yes | `:379-384` |
| **all switches/triggers** | `topLeft.h` **even** | yes | `:403-407` |
| `kFlourescent`, `kTrackLight` | clamp `topLeft` up to `bounds` topLeft, shrink `length` to `bounds.right` | yes | `:421-431` |
| all other lights | `topLeft = bounds.topLeft` | yes | `:432-436` |
| `kFlourescent`, `kTrackLight` (overhang) | `topLeft.h` ∈ [0, 512]; `length = 512 - bounds.left` | yes for the clamps, **no** for the `length` line | `:439-464` |
| `kToaster` | `bounds.top - height ≥ 0` | yes | `:488-493` |
| `kTV` | `topLeft.h` **odd** | yes | `:494-499` |
| `kToaster`, `kMacPlus`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`, `kMicrowave` | `topLeft.h` **even** | yes | `:500-511` |
| `kBall`, `kFish` | `bounds.top - length ≥ 0` | yes | `:530-536` |
| `kDrip` | `bounds.bottom + length ≤ 322` | yes | `:537-542` |
| `kBalloon`, `kCopterLf`, `kCopterRt`, `kBall`, `kDrip`, `kFish` | `topLeft.h` **even** | yes | `:543-553` |
| `kMirror` | `bounds.left` **even**, `bounds.right` **even** | yes | `:577-589` |

#### 18.17.13 The silent mutations

Nine places mutate the object and do **not** set `unchanged = false`, so `KeepAllObjectsLegal`
never counts them and the user is never told:

1. the initial-glider clamp (`:59-68`, returns `true`);
2. `kFloorVent` v-snap + `distance += 2` (`:115-119`);
3. `kFloorBlower` v-snap + `distance += 2` (`:120-125`);
4. `kSewerGrate` v-snap + `distance += 2` (`:126-131`);
5. `kFloorTrans` v-snap (`:132-137`) — unreachable anyway, see below;
6. the interior-door handedness flip (`:310-324`);
7. the exterior-door handedness flip (`:325-339`);
8. the interior/exterior window handedness flips (`:340-369`);
9. the strip-light `length = kRoomWide - bounds.left` line (`:463`).

Additionally `NormalizeRect` inside `ForceRectInRect` silently un-inverts rects without setting
`changed` (`GliderPRO/Sources/RectUtils.c:230`).

**The unreachable `kFloorTrans` case.** `HouseLegal.c:132-137` tests
`theObject->what == kFloorTrans` (0x35) from *inside the blower arm*, whose `case` labels are
0x01-0x10. `kFloorTrans` is handled by the transport arm (`:286`). The block is therefore dead
code — and it would be wrong if it ran, because it writes `data.a.topLeft`/`data.a.distance`
whereas a floor transport is a variant `d`. (`data.a.topLeft` and `data.d.topLeft` do overlap at
payload 0-3, but `data.a.distance` at 4-5 is `data.d.tall`, so the `+= 2` would corrupt the
transport's height.) A Go port must **not** "fix" this by moving the case into the transport arm;
doing so would change `kFloorTrans` geometry in every existing house.

#### 18.17.14 Empirical verification of `KeepObjectLegal` against the shipped houses

I re-implemented in python3 every snap rule from §18.17.12 that can be evaluated from the raw
payload plus the `srcRects` table (i.e. everything except the general `ForceRectInRect`
containment, which needs the full 144-entry `srcRects` array), and ran it over all 4070 live rooms
of all 22 shipped houses.

Rules checked: `kStubby` odd-h; `kTaper`/`kCandle`/`kTiki`/`kBBQ` even-h; `kFloorVent` v == 305;
`kFloorBlower` v == 304; `kSewerGrate` v == 303; `kManhole` `(left-3) % 64 == 0`; all-prize
even-h; non-star even-length; `kGreaseRt` `right+length ≤ 512`; `kGreaseLf` `left-length ≥ 0`;
`kSlider` `left+length ≤ 512`; all four door/window handedness pairs (`what` **and** x);
`kInvisTrans` `v + tall ≤ 322`; all-switch even-h; `kTV` odd-h; the seven even-h appliances;
`kToaster` `top - height ≥ 0`; the six even-h enemies; `kBall`/`kFish` `top - length ≥ 0`;
`kDrip` `bottom + length ≤ 322`; `kMirror` even left and right.

**Result: 0 violations, for every rule, in every house.** (692 grease/slider objects examined for
the length rules alone.) Every shipped house is already a fixed point of `KeepObjectLegal` for all
the locally-checkable rules — which is exactly what one expects, since every house was authored in
this editor and every save runs `KeepAllObjectsLegal`.

That result is also a strong independent confirmation of the `Point` field order. A first pass that
read `topLeft` as `(h, v)` reported 3421 "loud" and 2175 "silent" violations, including 1453
`kFloorVent`s allegedly not at v = 305 — nonsense. Re-reading `Point` as
**`{short v; short h;}`** (v at payload 0:2, h at 2:4, per
`GliderPRO/Headers/GliderStructs.h` and `MacTypes.h`) took every count to zero. If a Go port ever
sees `KeepObjectLegal` reporting mass violations on a shipped house, the field order is the first
thing to check.

### 18.18 Localized strings used by the validator

All from `STR# 150` ("Localized Strings", 51 entries), fetched via
`GetLocalizedString(index, buf)` → `GetIndString(buf, 150, index)`
(`GliderPRO/Sources/StringUtils.c:321-327`). Verified by parsing the `STR# 150` resource data out
of `GliderPRO/Glider PRO.r`:

| # | String (exact, leading spaces significant) | Used by |
|---|---|---|
| 16 | `Failed House Shrinkage` | `LopOffExtraRooms` `:771` |
| 17 | ` Floor Number Bad` | `ValidateRoomNumbers` `:809` |
| 18 | ` Suite Number Bad` | `ValidateRoomNumbers` `:819` |
| 19 | `Object Bad` | `KeepAllObjectsLegal` `:942` |
| 20 | `No room upstairs!` | `CheckForStaircasePairs` `:985` |
| 21 | `No downstairs to match!` | `CheckForStaircasePairs` `:1002` |
| 22 | `No room downstairs!` | `CheckForStaircasePairs` `:1016` |
| 23 | `No upstairs to match!` | `CheckForStaircasePairs` `:1033` |
| 24 | `Checking House File` | driver, **window title** `:1062-1063` |
| 25 | `Checking House…` | driver `:1066` |
| 26 | `Checking Rooms…` | driver `:1073` |
| 27 | `Room Number Errors` | driver `:1078` |
| 28 | ` Duplicate Floor/Suites` | driver `:1093` |
| 29 | ` Room Errors` | driver `:1114` |
| 30 | ` 'Untitled' Rooms` | driver `:1131` |
| 31 | ` Room Names Too Long` | driver `:1148` |
| 32 | ` Room's # of Objects Wrong` | driver `:1165` |
| 33 | `Checking Objects…` | driver `:1178` |
| 34 | ` Object Errors` | driver `:1184` |
| 35 | `You have no stars in the house!` | driver `:1206` |

The `…` characters are single-byte Mac Roman 0xC9, not three ASCII periods.

### 18.19 What validation does *not* check

Enumerated, because a porter will be tempted to add these:

* **Nothing about links.** No `where`/`who` range check, no dangling-link detection, no
  self-link detection, no check that a switch's target is a switchable object. §17.13 documents
  2474 links in the shipped houses of which 189 point at a nonexistent room, 180 have
  `who > 23`, and 5 point at an empty slot — all of which pass validation untouched.
* **Nothing about `background` or `tiles`.** A room may name a PICT that does not exist;
  `ReadyBackground` handles that at draw time.
* **Nothing about `openings`, `leftStart`, `rightStart`, `visited`.**
* **Nothing about `numObjects` being a *usable* count** — only that it equals the population
  count (§18.13).
* **No object-overlap check.** Two objects may occupy the same pixels.
* **No check that `firstRoom` names a live room.** `CompressHouse` maintains it across moves but
  never validates it. `GetFirstRoomNumber` (`GliderPRO/Sources/House.c:196-216`) is the function
  that copes at load time.
* **No check that the house has a reachable room graph**, or that the initial glider position is
  inside `firstRoom`'s walls.
* **No duplicate-object-in-slot check, no `what == 0` check.** Object type 0 is not a valid type
  (`kObjectIsEmpty` is −1) but nothing rejects it; `KeepObjectLegal`'s `switch` has no `default`
  and `GetObjectRect`'s does (`QSetRect(0,0,0,0)` only for `kObjectIsEmpty`).
* **No `kMaxSoundTriggers` enforcement.** `ObjectAdd.c` refuses to *add* a second
  `kSoundTrigger` to a room (`GliderPRO/Sources/ObjectAdd.c:17`, `:484-488`, helper `HowManySoundObjects` at `:989`) but the validator
  never re-checks. (Empirically the maximum observed per room across all 4070 rooms is indeed 1.)

### 18.20 Interaction with `SortHouseObjects` and room renumbering

The save path is, in order (`GliderPRO/Sources/Menu.c:452-467` then
`GliderPRO/Sources/HouseIO.c:448-520`):

```
 1  DeselectObject()
 2  if fileDirty: SortHouseObjects()          // compacts object slots, rewrites links   (§17.9)
 3  if fileDirty && houseUnlocked:
 4      if wasHouseVersion < kHouseVersion: ConvertHouseVer1To2()
 5      wasHouseVersion = kHouseVersion
 6      WriteHouse(true):
 7          SetFPos(houseRefNum, fsFromStart, 0)
 8          CopyThisRoomToRoom()
 9          CheckHouseForProblems()           // §18.4 — INCLUDING CompressHouse()
10          HLock(thisHouse); byteCount = GetHandleSize(thisHouse)
11          if fileDirty:
12              GetDateTime(&timeStamp); timeStamp &= 0x7FFFFFFF
13              if changeLockStateOfHouse: houseUnlocked = !saveHouseLocked
14              if houseUnlocked: timeStamp &= 0x7FFFFFFE  else: timeStamp |= 0x00000001
15              (*thisHouse)->timeStamp = timeStamp
16              (*thisHouse)->version   = wasHouseVersion
17          FSWrite(houseRefNum, &byteCount, *thisHouse)
18          SetEOF(houseRefNum, byteCount)
19  ForceThisRoom(thisRoomNumber); ReadyBackground(...); GetThisRoomsObjRects(); DrawThisRoomsObjects()
```

The ordering hazard: step 2 builds and consumes `linksList` in terms of **room indices**, and step
9's `CompressHouse` can **change those indices**. Because `SortHouseObjects` has fully finished
before `WriteHouse` is entered, and `linksList` is regenerated from scratch on the next
`SortHouseObjects` call (`GliderPRO/Sources/House.c:448-503`), no live corruption results — but a
Go port that keeps a persistent link index must invalidate it whenever rooms are renumbered.

Also note step 14: the house **lock bit is the low bit of `timeStamp`**. `houseUnlocked` clears
it; locked sets it. `ReadHouse` reads it back as
`houseUnlocked = (((*thisHouse)->timeStamp & 0x00000001) == 0)`
(`GliderPRO/Sources/HouseIO.c:402`). And `timeStamp &= 0x7FFFFFFF` at step 12 forces the sign bit
off so the `long` stays positive.

### 18.21 Empirical bottom line for the shipped houses

Running the complete `CheckHouseForProblems` simulation over all 22 houses (4070 live rooms,
0 tombstones):

| Check | Violations found |
|---|---|
| `ValidateNumberOfRooms` (`(len − 866) / 348 != nRooms`) | **0** (Sampler's 2 stray bytes absorbed by truncation) |
| `CheckDuplicateFloorSuite` | **0** |
| `CompressHouse` / `LopOffExtraRooms` (tombstones) | **0** total, **0** trailing |
| `ValidateRoomNumbers` (floor ∉ [−7, 56] or suite ∉ [0, 127]) | **0** |
| `CountUntitledRooms` | **0** |
| `CheckRoomNameLength` (`name[0] > 27`) | **0** |
| `MakeSureNumObjectsJives` | **0** |
| `KeepAllObjectsLegal` (all locally-checkable snap rules) | **0** |
| `CheckForStaircasePairs` (163 up / 163 down) | **0** |
| `CountStarsInHouse() < 1` | **1 — Fun House** |
| `WrapBannerAndTrailer` (line > 40 / > 64) | **0** |
| initial-glider clamp | **0** |

So on shipped data the validator is a 39-tick no-op with exactly one red message, for exactly one
house. A Go port can use "load all 22 houses, run the validator, expect byte-identical output
except for a red message on Fun House" as its regression test for this entire section.

---

## 19. Saving and loading the house

Everything the editor changes lives in one relocatable block, `thisHouse`, and is written back
to disk by exactly one function, `WriteHouse`. This section documents the file format on disk,
the discovery of house files, the four entry points that touch the disk
(`OpenHouse` / `ReadHouse` / `WriteHouse` / `CloseHouse`), the dirty-flag protocol that decides
*when* a write happens, the lock bit that decides *whether* the editor is allowed to write at
all, and the two side-car files (high scores, QuickTime movie).

All of `HouseIO.c` (709 lines) was read in full; every line number below was re-confirmed by
`grep` against the converted copy.

---

### 19.1 The house file on disk

A Glider PRO house is a classic-Mac two-forked file:

| Fork | Contents | Written by the editor? |
|---|---|---|
| **Data fork** | one contiguous big-endian `houseType` blob: an 866-byte header followed by `nRooms` × 348-byte `roomType` records | **Yes** — whole-file rewrite on every save |
| **Resource fork** | user artwork (`PICT`), room-opening descriptors (`bnds`), house sounds (`snd `), a custom Finder icon family, sometimes `vers` | **No** — opened read-only-in-practice; the app never calls `AddResource` |
| side-car `<name>.mov` | optional QuickTime movie shown on in-room TVs | No |
| side-car in Prefs folder | high scores for read-only houses | Yes, via `WriteScoresToDisk` |

Finder type/creator are **`'gliH'` / `'ozm5'`**, set at creation
(`FSpCreate(&theSpec, 'ozm5', 'gliH', theReply.keyScript)`, `GliderPRO/Sources/House.c:85`) and
matched during the disk scan
(`fdType == 'gliH' && fdCreator == 'ozm5'`, `GliderPRO/Sources/SelectHouse.c:592-593`).
`NavPutFile(nil, &theReply, &dialogOptions, nil, 'gliH', 'ozm5', nil)` at
`GliderPRO/Sources/House.c:58` also passes them to Navigation Services.

There is **no magic number and no checksum**. The only integrity signals are:

1. `version` at offset 0 (must be `< kNewHouseVersion` = `0x0300`), and
2. `nRooms` at offset 864 (must be `>= 1`).

#### 19.1.1 The data-fork size identity

`ReadHouse` sizes its Handle from `GetEOF` and never checks it against `nRooms`
(`GliderPRO/Sources/HouseIO.c:341`, `:356`). `WriteHouse` writes
`GetHandleSize((Handle)thisHouse)` bytes (`:473`) and then `SetEOF`s to that same count (`:499`).
So the file length is whatever the Handle length is, and *should* be

```
fileLength = sizeof(houseType prefix) + nRooms * sizeof(roomType)
           = 866 + 348 * nRooms
```

Verified with python3 over all 22 shipped houses (data forks extracted from the BinHex files in
`GliderPRO/Houses/`):

| house | data fork bytes | `nRooms` | 866 + 348·nRooms | delta |
|---|---:|---:|---:|---:|
| Art Museum | 38798 | 109 | 38798 | 0 |
| CD Demo House | 72554 | 206 | 72554 | 0 |
| California or Bust! | 6434 | 16 | 6434 | 0 |
| Castle o' the Air | 30446 | 85 | 30446 | 0 |
| Davis Station | 23486 | 65 | 23486 | 0 |
| Demo House | 16526 | 45 | 16526 | 0 |
| Empty House | 13046 | 35 | 13046 | 0 |
| Fun House | 15830 | 43 | 15830 | 0 |
| Grand Prix | 61766 | 175 | 61766 | 0 |
| ImagineHouse PRO II | 97958 | 279 | 97958 | 0 |
| In The Mirror | 34622 | 97 | 34622 | 0 |
| Land of Illusion | 106310 | 303 | 106310 | 0 |
| Leviathan | 165122 | 472 | 165122 | 0 |
| Metropolis | 45062 | 127 | 45062 | 0 |
| Nemo's Market | 44018 | 124 | 44018 | 0 |
| Rainbow's End | 78470 | 223 | 78470 | 0 |
| **Sampler** | **1564** | **2** | **1562** | **+2** |
| Slumberland | 134150 | 383 | 134150 | 0 |
| SpacePods | 140762 | 402 | 140762 | 0 |
| Teddy World | 185654 | 531 | 185654 | 0 |
| The Asylum Pro | 49586 | 140 | 49586 | 0 |
| Titanic | 73250 | 208 | 73250 | 0 |

21 of 22 match exactly. **Sampler carries two extra trailing bytes, `01 01`.** Those bytes
survive a load/save round-trip because `ReadHouse` sizes the Handle from `GetEOF` and
`WriteHouse` writes `GetHandleSize` bytes — nothing ever truncates them, and
`ValidateNumberOfRooms` (§18.6) computes `nRooms` from a *truncating* division so it absorbs
them silently. A Go port that reconstructs the file length from `nRooms` instead of preserving
the input length will produce a 1562-byte Sampler; that is arguably better behaviour but it is
not byte-identical.

#### 19.1.2 Annotated header dump

Empirical field-by-field dump of the first 866 bytes of **Demo House**'s data fork
(struct layout from `GliderPRO/Headers/GliderStructs.h:182-198`, 2-byte 68k alignment from
`#pragma options align=mac68k` at `GliderPRO/Headers/Externs.h:231`):

| offset (dec / hex) | field | C type | size | observed bytes | value |
|---:|---|---|---:|---|---|
| 0 / 0x000 | `version` | `short` | 2 | `02 00` | `0x0200` = `kHouseVersion` |
| 2 / 0x002 | `unusedShort` | `short` | 2 | `00 00` | 0 |
| 4 / 0x004 | `timeStamp` | `long` | 4 | `2C 53 B0 41` | 743682113; bit 0 = 1 → **locked** |
| 8 / 0x008 | `flags` | `long` | 4 | `00 00 00 00` | wardBit 0, phoneBit 0, bit 2 = 0 |
| 12 / 0x00C | `initial` | `Point` | 4 | `00 6B 00 31` | **v = 107, h = 49** (v first!) |
| 16 / 0x010 | `banner` | `Str255` | 256 | len `0x6F` = 111 | `"Welcome to the Demo House!\rThis is a small beginner house that\racts as a sort of tutorial.\r(house by Kim Money)"` |
| 272 / 0x110 | `trailer` | `Str255` | 256 | len `0x6F` = 111 | `"Excellent!\rThat's the extent of the Demo House though.  The house\r\"Slumberland\" has over 400 rooms!  Good luck."` |
| 528 / 0x210 | `highScores` | `scoresType` | 292 | banner len 19 | `"The Return of Ozma!"`; `names[0]` = `"Ozma"`; `scores[0..2]` = 7400, 0, 0 |
| 820 / 0x334 | `savedGame` | `gameType` | 40 | `01 00 00 01 AA BB 15 14 …` | stale, ignored because… |
| 860 / 0x35C | `hasGame` | `Boolean` | 1 | `00` | false |
| 861 / 0x35D | `unusedBoolean` | `Boolean` | 1 | `00` | 0 |
| 862 / 0x35E | `firstRoom` | `short` | 2 | `00 00` | 0 |
| 864 / 0x360 | `nRooms` | `short` | 2 | `00 2D` | 45 |
| 866 / 0x362 | `rooms[0]` | `roomType` | 348 | — | first room record |

Note the `savedGame` bytes are non-zero even though `hasGame` is false — the field is never
cleared, only flagged (`SaveGame(false)` sets only `hasGame = false`,
`GliderPRO/Sources/SavedGames.c:339-342`). A Go port must not treat a non-zero `savedGame` as
meaningful without checking `hasGame`.

`scoresType` (`GliderPRO/Headers/GliderStructs.h:107-114`), 292 bytes, relative to 0x210:

| rel. offset | field | type | size |
|---:|---|---|---:|
| 0 | `banner` | `Str31` | 32 |
| 32 | `names[10]` | `Str15` | 160 |
| 192 | `scores[10]` | `long` | 40 |
| 232 | `timeStamps[10]` | `unsigned long` | 40 |
| 272 | `levels[10]` | `short` | 20 |

`gameType` (`GliderPRO/Headers/GliderStructs.h:116-134`), 40 bytes, relative to 0x334:

| rel. offset | field | type | size |
|---:|---|---|---:|
| 0 | `version` | `short` | 2 |
| 2 | `wasStarsLeft` | `short` | 2 |
| 4 | `timeStamp` | `long` | 4 |
| 8 | `where` | `Point` | 4 |
| 12 | `score` | `long` | 4 |
| 16 | `unusedLong` | `long` | 4 |
| 20 | `unusedLong2` | `long` | 4 |
| 24 | `energy` | `short` | 2 |
| 26 | `bands` | `short` | 2 |
| 28 | `roomNumber` | `short` | 2 |
| 30 | `gliderState` | `short` | 2 |
| 32 | `numGliders` | `short` | 2 |
| 34 | `foil` | `short` | 2 |
| 36 | `unusedShort` | `short` | 2 |
| 38 | `facing` | `Boolean` | 1 |
| 39 | `showFoil` | `Boolean` | 1 |

#### 19.1.3 The COMPILEDEMO fingerprint — an independent confirmation of Demo House

Under `#ifdef COMPILEDEMO` (not defined in the shipped build — `GliderDefines.h:12` has it
commented out) `ReadHouse` hard-codes a fingerprint of the one house the demo may open:

| check | line | value | matches my dump? |
|---|---|---|---|
| `byteCount != 16526L` → fail | `GliderPRO/Sources/HouseIO.c:349` | 16526 bytes | yes — Demo House is 16526 |
| `numberRooms != 45` → fail | `:382` | 45 rooms | yes |
| `houseUnlocked` → fail | `:404` | must be **locked** | yes — `timeStamp` bit 0 = 1 |
| `whichRoom != 0` → fail | `:412` | `firstRoom` 0 | yes |
| name != `"Demo House"` → fail | `:183` | — | — |

Four independent constants baked into 1994 source agree with four fields I parsed out of the
BinHex'd file in 2026. That is the strongest available confirmation that the layout above is
correct.

---

### 19.2 The resource fork

`OpenHouseResFork` (`GliderPRO/Sources/HouseIO.c:567-577`) opens it and makes it the *current*
resource file, so every subsequent unqualified `GetResource` / `GetPicture` searches the house
first and falls back to the application:

```c
if (houseResFork == -1)
{
    houseResFork = FSpOpenResFile(&theHousesSpecs[thisHouseIndex], fsCurPerm);
    if (houseResFork == -1)
        YellowAlert(kYellowFailedResOpen, ResError());
    else
        UseResFile(houseResFork);
}
```

`CloseHouseResFork` (`:582-589`) closes it and resets the sentinel to `-1`.
`houseResFork` is initialised to `-1` at `GliderPRO/Sources/InterfaceInit.c:158`.

#### 19.2.1 Observed resource types across all 22 shipped houses

Parsed the resource forks out of the BinHex files with python3:

| type | total count | purpose |
|---|---:|---|
| `PICT` | 919 | user backgrounds, user structures, custom object pictures, dialog art |
| `bnds` | 70 | 4-byte room-opening descriptor for a user background |
| `snd ` | 63 | sounds played by `kSoundTrigger` objects |
| `ICN#` | 25 | custom Finder icon (b&w) |
| `icl8` | 25 | custom Finder icon (8-bit) |
| `icl4` | 23 | custom Finder icon (4-bit) |
| `ics#` / `ics4` / `ics8` | 22 each | small Finder icons |
| `vers` | 12 | author/version strings (Finder "Get Info") |

**"Empty House" and "Sampler" are the only two houses with no `PICT`s**; Sampler's resource fork
is entirely empty.

The Finder icon family is almost always ID **-16455** (the standard custom-icon ID);
`UpdateLoadDialog` looks for exactly that (`Get1Resource('icl8', -16455)`,
`GliderPRO/Sources/SelectHouse.c:106`) and falls back to `PICT` 1004 (`kDefaultHousePict8`) if
absent. Four houses (In The Mirror, Leviathan, Teddy World, Titanic) also carry an `ICN#` 128.

#### 19.2.2 `PICT` ID conventions

| range | constant | count observed | meaning |
|---|---|---:|---|
| 1000–1999 | — | 49 | overrides of built-in app art (e.g. 1017/1018 room-info thumbnails, 1991–1993) |
| 2000–2017 | `kBaseBackgroundID` … | 2 | overrides of built-in backgrounds |
| **3000–3299** | `kUserBackground` = 3000 (`GliderDefines.h:522`) | 201 | user room backgrounds (512×322) |
| **3300–3999** | `kUserStructureRange` = 3300 (`GliderDefines.h:523`) | 121 | user backgrounds *with* wall/floor structure |
| 4000–9999 | — | 1 | stray |
| **≥ 10000** | — | 545 | `kCustomPict` object pictures, referenced by `data.g.height` |

The 3000/3300 split matters to `IsRoomAStructure` (`GliderPRO/Sources/Room.c:762-811`), which
decides whether a room is an indoor "structure" (walls, floor, ceiling) or an outdoor one:

```c
if ((*thisHouse)->rooms[roomNum].background >= kUserBackground)   // :773
{
    if ((*thisHouse)->rooms[roomNum].bounds != 0)                 // :775
        isStructure = (((*thisHouse)->rooms[roomNum].bounds & 32) == 32);   // :777
    else
    {
        if ((*thisHouse)->rooms[roomNum].background < kUserStructureRange)  // :781
            isStructure = true;
        else
            isStructure = false;
    }
}
else  { switch (background) { kPaneledRoom … kRoof: isStructure = true; default: false; } }  // :788-806
```

So for a user background with `bounds == 0` (no in-house opening data): **3000–3299 →
`isStructure = true`, 3300+ → `isStructure = false`.** Despite its name,
`kUserStructureRange` is the ID *above which* user backgrounds are **not** structures.
When `bounds != 0` the room's own bit 5 wins and the ID range is ignored entirely.

`GetObjectRect` reaches for `GetPicture(who->data.g.height)` for `kCustomPict` and, if the PICT
is missing, **rewrites the object** to `data.g.height = 10000`
(`GliderPRO/Sources/ObjectRects.c:224`) and falls back to `srcRects[kCustomPict]`
(the whole case is `ObjectRects.c:220-237`). That is the only place the Resource Manager can
mutate house data.

The "Select Original Art" dialog accepts only `3000 <= id < 3800` and only if
`PictIDExists(id)` (`GliderPRO/Sources/RoomInfo.c:762`) — so **3800–9999 is unusable as a
background** even though the drawing code would accept it, and the 545 observed PICTs at
ID ≥ 10000 can only be `kCustomPict` object art.

#### 19.2.3 `bnds` — 4 bytes, `boundsType`

`boundsType` is four `Boolean`s (`GliderPRO/Headers/GliderStructs.h:266-272`):

```c
typedef struct { Boolean left; Boolean top; Boolean right; Boolean bottom; } boundsType;
```

Loaded by `GetOriginalBounding(short theID)` (`GliderPRO/Sources/Room.c:937-964`) via
`GetResource('bnds', theID)` at `:942`; a missing resource raises
`YellowAlert(kYellowNoBoundsRes, 0)` at `:946` **only if `PictIDExists(theID)`**, and returns
`boundCode = 0`. The four Booleans are folded into a bit code (`Room.c:951-958`):

| bit | value | field |
|---:|---:|---|
| 0 | 1 | `left` |
| 1 | 2 | `top` |
| 2 | 4 | `right` |
| 3 | 8 | `bottom` |

The resource is only consulted when the room's own `bounds` field (offset 28) is 0. `room.bounds`
is a packed short, written by the Select Original Art dialog
(`GliderPRO/Sources/RoomInfo.c:766-780`):

```c
tempShort = 0;
if (originalLeftOpen)   tempShort += 1;
if (originalTopOpen)    tempShort += 2;
if (originalRightOpen)  tempShort += 4;
if (originalBottomOpen) tempShort += 8;
if (originalFloor)      tempShort += 16;
tempShort = tempShort << 1;   // shift left 1 bit
tempShort += 1;               // flag that says orginal bounds used
thisRoom->bounds = tempShort;
```

| `room.bounds` bit | value | meaning |
|---:|---:|---|
| 0 | 1 | "in-house bounds are present" flag; `bounds == 0` → fall back to the `'bnds'` resource |
| 1 | 2 | left open |
| 2 | 4 | top open |
| 3 | 8 | right open |
| 4 | 16 | bottom open |
| 5 | 32 | floor support / room is a structure |

Readers use `bounds >> 1` to recover the bit code
(`Room.c:828`, `:1111`, `:1146`, `:1180`, `RoomInfo.c:744`) and `bounds & 32` for the structure
bit (`Room.c:777`). `CreateNewRoom` initialises `thisRoom->bounds = 0`
(`GliderPRO/Sources/Room.c:172`), so a freshly created room always uses the resource.

All 70 observed `bnds` resources are exactly **4 bytes**. Sample values:

| house | ID | bytes | left / top / right / bottom |
|---|---:|---|---|
| Demo House | 3000 | `00 00 01 00` | 0 / 0 / 1 / 0 |
| Demo House | 3005 | `00 00 00 00` | 0 / 0 / 0 / 0 |
| Demo House | 3300 | `00 01 01 01` | 0 / 1 / 1 / 1 |
| Castle o' the Air | 3301 | `01 01 01 01` | 1 / 1 / 1 / 1 |
| Castle o' the Air | 3000 | `01 00 01 00` | 1 / 0 / 1 / 0 |

#### 19.2.4 `snd ` — house sounds for `kSoundTrigger`

A `kSoundTrigger` object's `data.e.where` is **not a link** — it is a `'snd '` resource ID.
`GliderPRO/Sources/ObjectRects.c:923-928`:

```c
case kSoundTrigger:
QSetRect(&bounds, 0, 0, 48, 48);
QOffsetRect(&bounds, theObject.data.e.topLeft.h, theObject.data.e.topLeft.v);
if (LoadTriggerSound(theObject.data.e.where) == noErr)
    hotSpotNumber = AddActiveRect(&bounds, kSoundIt, who, true, false);
```

`LoadTriggerSound` (`GliderPRO/Sources/Sound.c:265-303`) does
`GetResource('snd ', soundID)` at `:279`, then `soundDataSize = GetHandleSize(theSound) - 20L`
(`:286`) and `BlockMove((Ptr)(*theSound + 20L), theSoundData[kMaxSounds - 1], soundDataSize)`
(`:296`) — i.e. it **skips a fixed 20-byte `'snd '` header** and treats the remainder as raw
sampled data. If the resource is missing it returns `-1` (`:282`) and the object simply gets no
hot rect (silently inactive).

Observed `snd ` IDs per house:

| house | count | IDs |
|---|---:|---|
| Art Museum | 10 | 3000-3004, 3042-3046 |
| CD Demo House | 10 | 3000-3008, 3042 |
| California or Bust! | 3 | 3001-3003 |
| Davis Station | 5 | 3000-3004 |
| Demo House | 1 | 3011 |
| Grand Prix | 3 | 3000-3002 |
| ImagineHouse PRO II | 2 | 3001, 3003 |
| In The Mirror | 2 | 3001, 3002 |
| Leviathan | 12 | 3000, 3001, 3003-3012 |
| Nemo's Market | 5 | 3001-3005 |
| Rainbow's End | 1 | 3000 |
| SpacePods | 3 | 3000-3002 |
| Titanic | 6 | 3000, 3006, 3032, 3037, 3058, 3061 |

There are exactly **122 `kSoundTrigger` objects** (`what == 0x49`,
`GliderPRO/Headers/GliderDefines.h:385`) across the 22 houses. Their `where` histogram
(payload bytes 6:8, big-endian `short`):

| `where` | count | | `where` | count | | `where` | count |
|---:|---:|---|---:|---:|---|---:|---:|
| 3000 | 35 | | 3005 | 5 | | 3010 | 1 |
| 3001 | 18 | | 3044 | 3 | | 3012 | 1 |
| 3002 | 14 | | 3007 | 2 | | 3037 | 1 |
| 3006 | 13 | | 3008 | 2 | | 3043 | 1 |
| 3003 | 6 | | 3011 | 2 | | 3045 | 1 |
| 3004 | 6 | | **10000** | **2** | | 3046 | 1 |
| 3042 | 6 | | 3009 | 1 | | 3058 | 1 |

Cross-referencing each object's `where` against the `snd ` IDs actually present in *its own*
house exposes two houses that ship broken sound triggers:

| house | `kSoundTrigger` objects | `snd ` resources | unresolvable |
|---|---:|---:|---|
| **Teddy World** | 11 | **0** | all 11 (`where == 3000`) |
| **In The Mirror** | 7 | 2 | 2 (`where == 10000`) |
| the other 12 houses with triggers | 96 | 1–12 | none |

So all 122 minus 13 = 109 triggers resolve. The 13 that do not are silently inert:
`LoadTriggerSound` returns `-1`, `AddActiveRect` is never called, and the 48×48 hot spot simply
does not exist. No alert, no log. (`where == 10000` is also the sentinel `GetObjectRect` writes
into `data.g.height` for a missing `kCustomPict`; whether that is coincidence or a shared
"broken" marker is unresolved — see Open questions.)

#### 19.2.5 The editor never writes the resource fork

Grepping the whole source for resource-writing calls:

| call | occurrences | where |
|---|---:|---|
| `AddResource` | 1 | `GliderPRO/Sources/DebugUtilities.c:352` (writes a `'demo'` resource; debug-only) |
| `ChangedResource` | 2 | `DebugUtilities.c:353`, `GliderPRO/Sources/Environ.c:725` (prefs) |
| `WriteResource` | 1 | `Environ.c:726` (prefs) |
| `RemoveResource` | 1 | `Environ.c:713` (prefs) |
| `UpdateResFile` | 1 | `Environ.c:730` (the *application's* res file) |

None of them touch `houseResFork`. Consequently:

* **`kYellowFailedResAdd` (yellow-alert string 3, "I failed to add a resource to the house's
  resource fork") is defined at `GliderPRO/Headers/GliderDefines.h:20` and never called from
  anywhere.** Verified by grepping all 92 `.c` files.
* Adding a custom background, a custom object picture, a `bnds` record or a house sound is an
  **out-of-band operation**: the author must paste the resource into the house file with ResEdit
  (or similar). The in-app "Select Original Art" dialog (§13.9) only *chooses among* PICT IDs
  that already exist; `ChooseOriginalArt` walks IDs with `GetPicture`/`Count1Resources`, it never
  creates one.
* A Go port therefore needs an *import* path that does not exist in the original: some way to
  attach a PNG/whatever as "user background 3000" and a 4-byte opening descriptor beside it.
  §19.20 discusses this.

`HCreateResFile` is called exactly twice, both at house *creation* time, to make an **empty**
resource fork: `GliderPRO/Sources/House.c:88` (live) and `GliderPRO/Sources/HouseIO.c:270`
(inside the commented-out `SaveHouseAs`). Failure raises
`YellowAlert(kYellowFailedResCreate, ResError())` (`House.c:89-90`, `HouseIO.c:272-273`).

---

### 19.3 Module state (`GliderPRO/Sources/HouseIO.c:31-42`)

| variable | type | line | meaning |
|---|---|---|---|
| `theMovie` | `Movie` | `:31` | QuickTime movie handle for the side-car `.mov` |
| `movieRect` | `Rect` | `:32` | `GetMovieBox` result |
| `houseRefNum` | `short` | `:33` | data-fork file refNum |
| `houseResFork` | `short` | `:33` | resource-fork refNum, `-1` = closed |
| `wasHouseVersion` | `short` | `:33` | `version` as read from disk; written back verbatim by `WriteHouse` |
| `houseOpen` | `Boolean` | `:34` | data fork is open |
| **`fileDirty`** | `Boolean` | `:34` | house *structure* changed → needs `WriteHouse` |
| **`gameDirty`** | `Boolean` | `:34` | only high scores changed → may need only `WriteScoresToDisk` |
| `changeLockStateOfHouse` | `Boolean` | `:35` | House Info asked to flip the lock bit |
| `saveHouseLocked` | `Boolean` | `:35` | the *requested* new lock state (true = locked) |
| `houseIsReadOnly` | `Boolean` | `:35` | file/volume is write-protected — always **false** in 1.0.4 (§19.17.2) |
| `hasMovie` | `Boolean` | `:36` | a movie loaded successfully |
| `tvInRoom` | `Boolean` | `:36` | current room contains a `kTV` |

Externally owned but central to this section:

| variable | defined at | meaning |
|---|---|---|
| `thisHouse` | `GliderPRO/Sources/House.c:25` (`houseHand`) | the whole file, in one Handle |
| `thisHouseName` | `House.c:27` (`Str32`) | current house file name |
| `houseUnlocked` | `House.c:32` (`Boolean`) | decoded from `timeStamp` bit 0 |
| `numberRooms` | — | mirror of `(*thisHouse)->nRooms` |
| `noRoomAtAll` | — | `RealRoomNumberCount() == 0` |
| `theHousesSpecs` | `GliderPRO/Sources/SelectHouse.c:47` (`FSSpecPtr`) | array of found houses |
| `thisHouseIndex` | `SelectHouse.c:51` | index into `theHousesSpecs` of the open house |
| `housesFound` | `SelectHouse.c:51` | number of valid entries |
| `maxFiles` | `SelectHouse.c:51` | capacity of `theHousesSpecs` |

---

### 19.4 Finding house files

#### 19.4.1 `theHousesSpecs` allocation

`GliderPRO/Sources/StructuresInit2.c:275-276`:

```c
theHousesSpecs = nil;
theHousesSpecs = (FSSpecPtr)NewPtr(sizeof(FSSpec) * maxFiles);
```

`maxFiles` comes from prefs, clamped to [12, 500], default 48
(`GliderPRO/Sources/Main.c:87-89`, `:156-157`; the Preferences dialog edits `willMaxFiles`,
`GliderPRO/Sources/Settings.c:268-282`, default reset to 48 at `Settings.c:1225`).
`FSSpec` is 70 bytes, so the default array is 3360 bytes.

#### 19.4.2 `BuildHouseList` (`GliderPRO/Sources/SelectHouse.c:650-664`)

1. If `!thisMac.hasSystem7`, **do nothing at all** (leaves `housesFound` at whatever it was).
2. `housesFound = 0`.
3. Copy the `numExtraHouses` entries of `extraHouseSpecs[]` (capacity `kMaxExtraHouses` = 8,
   `SelectHouse.c:35`) into the front of `theHousesSpecs`.
4. `DoDirSearch()`.

`AddExtraHouse(FSSpec *newHouse)` (`:668-675`) appends to `extraHouseSpecs`, silently dropping
the 9th and later. Callers: `GliderPRO/Sources/House.c:93` (newly created house) and
`GliderPRO/Sources/AppleEvents.c:92` (house dropped on the app).

#### 19.4.3 `DoDirSearch` (`GliderPRO/Sources/SelectHouse.c:557-646`)

A breadth-ish `PBGetCatInfo` walk starting at the application's own directory:

1. `theDirs[0] = thisMac.dirID`, `numDirs = 1`, `currentDir = 0`;
   `kMaxDirectories` = 32 (`:559`).
2. While `currentDir < numDirs && currentDir < 32`: enumerate entries `count = 1, 2, …` of
   `theDirs[currentDir]` until `PBGetCatInfo` errors.
   * file (`ioFlAttrib & 0x10 == 0`) and `fdType == 'gliH'` and `fdCreator == 'ozm5'` and
     `housesFound < maxFiles` → `FSMakeFSSpec` into `theHousesSpecs[housesFound++]`.
   * directory (`ioFlAttrib & 0x10 == 0x10`) → push `ioDirID` onto `theDirs` if room.
3. If `housesFound < 1`: `thisHouseIndex = -1` and `YellowAlert(kYellowNoHouses, 0)` (`:620`).
4. Else `SortHouseList()`, then set `thisHouseIndex` to the entry whose name equals
   `thisHouseName` (default 0), copy that name back into `thisHouseName`, and finally look up
   `demoHouseIndex` — pre-set to `-1` at `:636` and assigned at `:641` on
   `EqualString(theHousesSpecs[i].name, "\pDemo House", false, true)` (`:639`). If the house is
   absent, `demoHouseIndex` stays `-1`, which disables Options ▸ item `iHelp` (= 5,
   `GliderPRO/Headers/Externs.h:206`) at `GliderPRO/Sources/Menu.c:88-91`; that item's handler is
   `DoDemoGame()` (`Menu.c:427-428`), i.e. the built-in tutorial plays Demo House.

`SortHouseList` (`:516-553`) de-duplicates then bubble-sorts by `WhichStringFirst`. **The
de-duplication test is buggy**: it compares `theHousesSpecs[i].vRefNum == theHousesSpecs[i].vRefNum`
and `theHousesSpecs[i].parID == theHousesSpecs[i].parID` (both `i`, should be `i` vs `h`,
`:528-529`), so any two houses with the same *name* anywhere on the volume collapse to one, and
the survivor is moved by swapping in `theHousesSpecs[housesFound - 1]` while the outer `while`
does not re-test index `h` — entries can be skipped.

---

### 19.5 `OpenHouse` (`GliderPRO/Sources/HouseIO.c:164-202`)

Opens the *currently selected* house — the selection is `thisHouseIndex`, not an argument.

```
 1. if (houseOpen) { if (!CloseHouse()) return false; }        // :169-173
 2. if ((housesFound < 1) || (thisHouseIndex == -1)) return false;   // :174-175
 3. theErr = ResolveAliasFile(&theHousesSpecs[thisHouseIndex], true,
                              &targetIsFolder, &wasAliased);   // :177-178
    if (!CheckFileError(theErr, thisHouseName)) return false;
 4. houseIsReadOnly = IsFileReadOnly(&theHousesSpecs[thisHouseIndex]);  // :187  (always false)
 5. theErr = FSpOpenDF(&theHousesSpecs[thisHouseIndex], fsCurPerm, &houseRefNum);  // :189
    if (!CheckFileError(theErr, thisHouseName)) return false;
 6. houseOpen = true;                                          // :193
 7. OpenHouseResFork();                                        // :194
 8. hasMovie = false; tvInRoom = false; tvWithMovieNumber = -1; // :196-198
 9. OpenHouseMovie();                                          // :199
10. return true;
```

`fsCurPerm` means "whatever permission is available" — read/write if possible, read-only
otherwise. Note that `OpenHouse` does **not** read any data; `ReadHouse` is always a separate
call and every caller pairs them:

| caller | line |
|---|---|
| `Main.c` startup | `GliderPRO/Sources/Main.c:340-341` |
| `DoLoadHouse` (OK button) | `GliderPRO/Sources/SelectHouse.c:419-420` |
| `DoLoadHouse` (double-click on name) | `SelectHouse.c:458-459` |
| `DoLoadHouse` (double-click on icon) | `SelectHouse.c:493-494` |
| `OpenSpecificHouse` | `GliderPRO/Sources/HouseIO.c:226-227` |
| `QuerySaveChanges` "Discard" | `HouseIO.c:624-625` |
| `CreateNewHouse` | `GliderPRO/Sources/House.c:96` (no `ReadHouse` — `InitializeEmptyHouse` supplies the data) |

`OpenSpecificHouse(FSSpec *specs)` (`:208-235`) linearly searches `theHousesSpecs` for a
matching (vRefNum, parID, name) triple, sets `thisHouseIndex`, then `OpenHouse()` +
`ReadHouse()`. Its only caller is inside the commented-out `SaveHouseAs`, so it is **dead code**
in 1.0.4.

---

### 19.6 `ReadHouse` (`GliderPRO/Sources/HouseIO.c:315-443`)

The whole file in one gulp. Numbered against the C:

```
 1. if (!houseOpen) { YellowAlert(kYellowUnaccounted, 2); return false; }        // :321-325
 2. if (gameDirty || fileDirty)                                                  // :327
    2a.   if (houseIsReadOnly)
              if (!WriteScoresToDisk()) { YellowAlert(kYellowFailedWrite,0); return false; }  // :331-335
    2b.   else if (!WriteHouse(false)) return false;                             // :337-338
 3. theErr = GetEOF(houseRefNum, &byteCount);                                    // :341
    on error: CheckFileError(theErr, thisHouseName); return false;
 4. [COMPILEDEMO only] if (byteCount != 16526L) return false;                    // :349
 5. if (thisHouse != nil) DisposeHandle((Handle)thisHouse);                      // :353-354
 6. thisHouse = (houseHand)NewHandle(byteCount);                                 // :356
    if (thisHouse == nil) { YellowAlert(kYellowNoMemory, 10); return false; }    // :357-361
 7. MoveHHi((Handle)thisHouse);                                                  // :362
 8. theErr = SetFPos(houseRefNum, fsFromStart, 0L);                              // :364
 9. HLock((Handle)thisHouse);                                                    // :371
10. theErr = FSRead(houseRefNum, &byteCount, *thisHouse);                        // :372
    on error: CheckFileError; HUnlock; return false;
11. numberRooms = (*thisHouse)->nRooms;                                          // :380
12. [COMPILEDEMO only] if (numberRooms != 45) return false;                      // :382
13. if ((numberRooms < 1) || (byteCount == 0L))                                  // :385
    { numberRooms = 0; noRoomAtAll = true;
      YellowAlert(kYellowNoRooms, 0);           // :389
      HUnlock; return false; }
14. wasHouseVersion = (*thisHouse)->version;                                     // :394
15. if (wasHouseVersion >= kNewHouseVersion)   // 0x0300                         // :395
    { YellowAlert(kYellowNewerVersion, 0); HUnlock; return false; }              // :397-399
16. houseUnlocked = (((*thisHouse)->timeStamp & 0x00000001) == 0);               // :402
17. [COMPILEDEMO only] if (houseUnlocked) return false;                          // :404
18. changeLockStateOfHouse = false; saveHouseLocked = false;                     // :407-408
19. whichRoom = (*thisHouse)->firstRoom;                                         // :410
20. [COMPILEDEMO only] if (whichRoom != 0) return false;                         // :412
21. wardBitSet        = ((flags & 0x00000001) == 0x00000001);                    // :416
    phoneBitSet       = ((flags & 0x00000002) == 0x00000002);                    // :417
    bannerStarCountOn = ((flags & 0x00000004) == 0x00000000);   // note: inverted // :418
22. HUnlock((Handle)thisHouse);                                                  // :420
23. noRoomAtAll = (RealRoomNumberCount() == 0);                                  // :422
24. thisRoomNumber = -1; previousRoom = -1;                                      // :423-424
25. if (!noRoomAtAll) CopyRoomToThisRoom(whichRoom);                             // :425-426
26. if (houseIsReadOnly) { houseUnlocked = false; ReadScoresFromDisk(); }        // :428-434
27. objActive = kNoObjectSelected;                                               // :436
28. ReflectCurrentRoom(true);                                                    // :437
29. gameDirty = false; fileDirty = false; UpdateMenus(false);                    // :438-440
30. return true;
```

Points a porter must not miss:

* **Step 2 writes to disk during a *load*.** Switching houses with unsaved changes silently
  saves the old one via `WriteHouse(false)` — note `checkIt == false`, so the validator does
  *not* run. `DoLoadHouse` calls `CloseHouse()` first (`SelectHouse.c:416`), which normally
  consumes the dirty flag via `QuerySaveChanges`, so in practice step 2 fires only on paths
  that skip `CloseHouse`.
* Step 6 allocates a Handle of *exactly* the file length. Nothing pads to
  `866 + 348·nRooms`.
* Steps 13/15 are the only validation. A file whose `nRooms` claims more rooms than the byte
  count can hold is accepted and every later loop reads past the end of the Handle.
* Step 16 is the whole locking mechanism: **`houseUnlocked = (timeStamp & 1) == 0`**.
* Step 21's `bannerStarCountOn` is the **inverse** of bit 2 — flag bit set means star counting
  *off*. Exactly one shipped house sets it (Art Museum, `flags == 0x00000006`).
* Step 25 is guarded — `CopyRoomToThisRoom` is skipped for an all-tombstone house, leaving
  `thisRoomNumber == -1`.
* Step 29 clears both dirty flags, so a failed-and-recovered load cannot leave a stale dirty
  state.
* On the error paths of steps 13/15 the function returns false with `thisHouse` allocated and
  populated, `houseOpen` still true, and `numberRooms` possibly bogus. Callers ignore the
  return value (`whoCares = ReadHouse();`).

#### 19.6.1 Empirical: the header fields of all 22 houses

| house | `timeStamp` | bit 0 | lock state | `flags` | `hasGame` | `firstRoom` | `nRooms` |
|---|---:|---:|---|---|---:|---:|---:|
| Art Museum | 746622005 | 1 | locked | `0x00000006` | 0 | 91 | 109 |
| CD Demo House | 742294873 | 1 | locked | `0x00000002` | 0 | 70 | 206 |
| California or Bust! | 745882296 | 0 | unlocked | `0x00000002` | 0 | 14 | 16 |
| Castle o' the Air | 741536868 | 0 | unlocked | `0x00000000` | 0 | 33 | 85 |
| Davis Station | 743681789 | 1 | locked | `0x00000002` | 0 | 4 | 65 |
| Demo House | 743682113 | 1 | locked | `0x00000000` | 0 | 0 | 45 |
| Empty House | 741421292 | 0 | unlocked | `0x00000000` | 0 | 0 | 35 |
| Fun House | 742681616 | 0 | unlocked | `0x00000000` | 0 | 29 | 43 |
| Grand Prix | 741444157 | 1 | locked | `0x00000000` | 0 | 127 | 175 |
| ImagineHouse PRO II | 740149565 | 1 | locked | `0x00000000` | **1** | 1 | 279 |
| In The Mirror | 741512193 | 1 | locked | `0x00000000` | 0 | 6 | 97 |
| Land of Illusion | 745268052 | 0 | unlocked | `0x00000002` | 0 | 43 | 303 |
| Leviathan | 741514803 | 1 | locked | `0x00000000` | 0 | 39 | 472 |
| Metropolis | 741642315 | 1 | locked | `0x00000000` | 0 | 8 | 127 |
| Nemo's Market | 741121233 | 1 | locked | `0x00000002` | 0 | 0 | 124 |
| Rainbow's End | 741121489 | 1 | locked | `0x00000002` | 0 | 30 | 223 |
| Sampler | 893435880 | 0 | unlocked | `0x00000000` | 0 | 1 | 2 |
| Slumberland | 743682377 | 1 | locked | `0x00000000` | 0 | 126 | 383 |
| SpacePods | 741122073 | 1 | locked | `0x00000002` | 0 | 259 | 402 |
| Teddy World | 744401697 | 1 | locked | `0x00000000` | 0 | 0 | 531 |
| The Asylum Pro | 738016929 | 1 | locked | `0x00000000` | 0 | 20 | 140 |
| Titanic | 755123966 | 0 | unlocked | `0x00000000` | **1** | 92 | 208 |

Totals: **15 locked, 7 unlocked**; `flags` = `0x00000000` ×14, `0x00000002` ×7 (phone bit),
`0x00000006` ×1 (phone bit + no-star-count); `hasGame` true in 2 houses
(ImagineHouse PRO II, Titanic) — those two ship with a stale saved game that the Game menu will
offer to resume.

Note `SpacePods` has `firstRoom == 259` with `nRooms == 402`; that is in range, but
`GetFirstRoomNumber` (`GliderPRO/Sources/House.c:196-217`) clamps `firstRoom` to 0 if it is
`>= nRooms` or `< 0`, so an out-of-range value degrades to room 0 rather than crashing.

---

### 19.7 `WriteHouse(Boolean checkIt)` (`GliderPRO/Sources/HouseIO.c:448-519`)

The *only* function that writes house data. Numbered against the C:

```
 1. if (!houseOpen) { YellowAlert(kYellowUnaccounted, 4); return false; }     // :454-458
 2. theErr = SetFPos(houseRefNum, fsFromStart, 0L);                           // :460
    on error: CheckFileError(theErr, thisHouseName); return false;
 3. CopyThisRoomToRoom();          // flush the scratch room back into thisHouse   // :467
 4. if (checkIt) CheckHouseForProblems();          // the validator, §18       // :469-470
 5. HLock((Handle)thisHouse);                                                  // :472
 6. byteCount = GetHandleSize((Handle)thisHouse);                              // :473
 7. if (fileDirty) {                                                          // :475
 7a.   GetDateTime(&timeStamp);                                               // :477
 7b.   timeStamp &= 0x7FFFFFFF;                    // clear the sign bit      // :478
 7c.   if (changeLockStateOfHouse) houseUnlocked = !saveHouseLocked;          // :480-481
 7d.   if (houseUnlocked) timeStamp &= 0x7FFFFFFE;   // clear bit 0           // :484
       else               timeStamp |= 0x00000001;   // set bit 0             // :486
 7e.   (*thisHouse)->timeStamp = (long)timeStamp;                             // :487
 7f.   (*thisHouse)->version   = wasHouseVersion;                             // :488
    }
 8. theErr = FSWrite(houseRefNum, &byteCount, *thisHouse);                    // :491
    on error: CheckFileError; HUnlock; return false;
 9. theErr = SetEOF(houseRefNum, byteCount);                                  // :499
    on error: CheckFileError; HUnlock; return false;
10. HUnlock((Handle)thisHouse);                                               // :507
11. if (changeLockStateOfHouse) { changeLockStateOfHouse = false;
                                  ReflectCurrentRoom(true); }                // :509-513
12. gameDirty = false; fileDirty = false; UpdateMenus(false);                 // :515-517
13. return true;
```

Observations:

* **The whole file is rewritten every time.** There is no partial/incremental write. That is
  fine — the largest shipped house is 185654 bytes.
* Step 3 is essential: the editor works on a *copy* of the current room (`thisRoom`, a
  `roomType*` into a separate buffer) and `CopyThisRoomToRoom`
  (`GliderPRO/Sources/Room.c:354-365`) is what commits it — the whole 348-byte struct, in one
  assignment: `(*thisHouse)->rooms[thisRoomNumber] = *thisRoom;` (`:363`). It early-returns on
  `(noRoomAtAll) || (thisRoomNumber == -1)` (`:358-359`), so saving an all-tombstone house does
  not scribble on `rooms[-1]`. A Go port that edits rooms in place can drop this, but must not
  drop the equivalent flush if it keeps a scratch room.
* Step 4 is the only call site of `CheckHouseForProblems` in the entire program (grep-verified).
  `checkIt` is true from `Menu.c:462` (House ▸ Save House), `HouseIO.c:613`
  (`QuerySaveChanges` → Save), `HouseIO.c:538` when `theMode == kEditMode`, and
  `SavedGames.c:348` when `theMode == kEditMode`. It is false from `HouseIO.c:337`
  (the flush inside `ReadHouse`) and `HouseIO.c:282` (dead `SaveHouseAs`).
* Step 6 uses `GetHandleSize`, not `866 + 348·nRooms`. If the validator's `CompressHouse` /
  `LopOffExtraRooms` shrank the Handle, the file shrinks with it (step 9's `SetEOF`).
* Step 7 runs **only if `fileDirty`**. Saving a house whose only change was a high score
  (`gameDirty` alone) therefore does **not** touch `timeStamp`, `version` or the lock bit — the
  house keeps its original modification stamp. This is deliberate: it lets a locked house record
  scores without becoming "modified".
* Step 7f writes back `wasHouseVersion`, *not* `kHouseVersion`. Version promotion happens in the
  callers (`Menu.c:459-461`, `HouseIO.c:610-612`), which run `ConvertHouseVer1To2()` and then set
  `wasHouseVersion = kHouseVersion` **before** calling `WriteHouse`.

#### 19.7.1 The `timeStamp` encoding, verified empirically

`GetDateTime` returns seconds since **1904-01-01 00:00:00 local time** as a `UInt32`. In 1995
that value is ≈ 2.89 × 10⁹, which has bit 31 set — and step 7b clears bit 31. So the stored
value is `realMacSeconds & 0x7FFFFFFF`, i.e. the true date minus 2³¹ seconds, with bit 0
overwritten by the lock flag.

Decoding all 22 shipped `timeStamp`s both ways with python3:

| house | stored | naïve 1904 + t | **1904 + (t \| 0x80000000)** |
|---|---:|---|---|
| Art Museum | 746622005 | 1927-08-29 11:00:05 | **1995-09-16 14:14:13** |
| CD Demo House | 742294873 | 1927-07-10 09:01:13 | **1995-07-28 12:15:21** |
| California or Bust! | 745882296 | 1927-08-20 21:31:36 | **1995-09-08 00:45:44** |
| Castle o' the Air | 741536868 | 1927-07-01 14:27:48 | **1995-07-19 17:41:56** |
| Davis Station | 743681789 | 1927-07-26 10:16:29 | **1995-08-13 13:30:37** |
| Demo House | 743682113 | 1927-07-26 10:21:53 | **1995-08-13 13:36:01** |
| Empty House | 741421292 | 1927-06-30 06:21:32 | **1995-07-18 09:35:40** |
| Fun House | 742681616 | 1927-07-14 20:26:56 | **1995-08-01 23:41:04** |
| Grand Prix | 741444157 | 1927-06-30 12:42:37 | **1995-07-18 15:56:45** |
| ImagineHouse PRO II | 740149565 | 1927-06-15 13:06:05 | **1995-07-03 16:20:13** |
| In The Mirror | 741512193 | 1927-07-01 07:36:33 | **1995-07-19 10:50:41** |
| Land of Illusion | 745268052 | 1927-08-13 18:54:12 | **1995-08-31 22:08:20** |
| Leviathan | 741514803 | 1927-07-01 08:20:03 | **1995-07-19 11:34:11** |
| Metropolis | 741642315 | 1927-07-02 19:45:15 | **1995-07-20 22:59:23** |
| Nemo's Market | 741121233 | 1927-06-26 19:00:33 | **1995-07-14 22:14:41** |
| Rainbow's End | 741121489 | 1927-06-26 19:04:49 | **1995-07-14 22:18:57** |
| Sampler | 893435880 | 1932-04-23 16:38:00 | **2000-05-11 19:52:08** |
| Slumberland | 743682377 | 1927-07-26 10:26:17 | **1995-08-13 13:40:25** |
| SpacePods | 741122073 | 1927-06-26 19:14:33 | **1995-07-14 22:28:41** |
| Teddy World | 744401697 | 1927-08-03 18:14:57 | **1995-08-21 21:29:05** |
| The Asylum Pro | 738016929 | 1927-05-21 20:42:09 | **1995-06-08 23:56:17** |
| Titanic | 755123966 | 1927-12-05 20:39:26 | **1995-12-23 23:53:34** |

21 of 22 land in **Jun–Dec 1995** with bit 31 restored, and in a nonsensical 1927 without. That
proves the `& 0x7FFFFFFF` masking is real and that the field is a Mac 1904-epoch timestamp with
bit 31 dropped and bit 0 stolen. (Sampler stamps 2000-05-11 — it was clearly re-saved years
after the rest.)

A Go port that wants to reproduce the encoding:

```go
// mac epoch, local time
secs := uint32(t.Sub(time.Date(1904,1,1,0,0,0,0,time.Local)) / time.Second)
secs &= 0x7FFFFFFF
if unlocked { secs &= 0xFFFFFFFE } else { secs |= 1 }
```

Note the loss of information: bit 0 is destroyed, so the recoverable resolution is 2 seconds,
and dates after 1972-01-… + 2³¹ s wrap. Since nothing in the program *reads* the timestamp as a
date (only bit 0 is ever tested), a port could store anything here, but preserving the encoding
keeps files interchangeable with the original.

#### 19.7.2 Lock state and the House Info dialog

`houseUnlocked` gates almost every editing command:

| effect | citation |
|---|---|
| House ▸ Save House enabled only if `fileDirty && houseUnlocked` | `GliderPRO/Sources/Menu.c:101-104` |
| House ▸ House Info… enabled only if `houseUnlocked` | `Menu.c:105-113` |
| House ▸ Room Info… disabled if `noRoomAtAll \|\| !houseUnlocked` | `Menu.c:115-118` |
| House ▸ Object Info… / Bring To Front / Send To Back disabled if `!houseUnlocked` | `Menu.c:119-124` |
| every `DoHouseMenu` action re-tests `if (houseUnlocked)` | `Menu.c:474-496` etc. |
| the main window title shows `"House Locked"` instead of the room name and (floor, suite) | `GliderPRO/Sources/MainWindow.c:329` (function `UpdateEditWindowTitle`, `:306-331`) |

`changeLockStateOfHouse` / `saveHouseLocked` are set by the House Info dialog's lock checkbox
(§14.7) and consumed by `WriteHouse` step 7c. Because step 7 is inside `if (fileDirty)`, and
because the checkbox itself sets `fileDirty = true` (`GliderPRO/Sources/HouseInfo.c:276`,
`:288`), the flip always takes effect on the next save.

---

### 19.8 The dirty flags

Two Booleans, both in `HouseIO.c:34`:

* **`fileDirty`** — the house structure changed. Requires a full `WriteHouse`.
* **`gameDirty`** — only `highScores` changed. May be satisfied by `WriteScoresToDisk` when the
  file is read-only.

They are cleared in exactly **three** places, all in `HouseIO.c`: `:439` (end of `ReadHouse`),
`:516` (end of `WriteHouse`), `:620` (`QuerySaveChanges` → Discard). There is no
"mark clean" anywhere else.

`gameDirty = true` occurs at exactly **one** site: `GliderPRO/Sources/HighScores.c:414`, after a
new high score is recorded.

`fileDirty = true` occurs at **49** sites across 11 files. Complete enumeration
(grep-verified, function names resolved from the banner comments):

| file:line | enclosing function | note |
|---|---|---|
| `House.c:156` | `InitializeEmptyHouse` | a brand-new house starts dirty |
| `House.c:423` | `SortRoomsObjects` | set when an object is slid down to fill a gap |
| `Link.c:315` | `DoLink` | |
| `Link.c:358` | `DoUnlink` | |
| `Map.c:791` | `MoveRoom` | **dead** — no live caller (§15.10) |
| `Room.c:229` | `CreateNewRoom` | |
| `Room.c:481` | `DeleteRoom` | |
| `RoomInfo.c:484` | `DoRoomInfo` | |
| `Scrap.c:161` | `GetRoomScrap` | **dead** — whole file commented out |
| `ObjectEdit.c:343` | `DragHandle` | |
| `ObjectEdit.c:525` | `DragObject` | |
| `ObjectEdit.c:1179` | `DeleteObject` | |
| `ObjectEdit.c:1703` | `MoveObject` | **dead** — compiled out by `BUILD_ARCADE_VERSION 1` |
| `ObjectAdd.c:777` | `AddNewObject` | |
| `HouseInfo.c:276` | `DoHouseInfo` | |
| `HouseInfo.c:288` | `DoHouseInfo` | |
| `HouseInfo.c:330` | `HowToZeroScores` | |
| `HouseInfo.c:336` | `HowToZeroScores` | |
| `Objects.c:989` | `SendObjectToBack` | (`BringObjectForward` shares the tail) |
| `ObjectInfo.c:1023`, `:1053` | `DoBlowerObjectInfo` | |
| `ObjectInfo.c:1226`, `:1247` | `DoCustPictObjectInfo` | |
| `ObjectInfo.c:1319`, `:1343`, `:1351`, `:1359` | `DoSwitchObjectInfo` | |
| `ObjectInfo.c:1456`, `:1476`, `:1495`, `:1514` | `DoTriggerObjectInfo` | |
| `ObjectInfo.c:1595`, `:1614` | `DoLightObjectInfo` | |
| `ObjectInfo.c:1694`, `:1725` | `DoApplianceObjectInfo` | |
| `ObjectInfo.c:1813`, `:1850` | `DoMicrowaveObjectInfo` | |
| `ObjectInfo.c:1904`, `:1924` | `DoGreaseObjectInfo` | |
| `ObjectInfo.c:2004`, `:2041` | `DoInvisBonusObjectInfo` | |
| `ObjectInfo.c:2118`, `:2128`, `:2137`, `:2146` | `DoTransObjectInfo` | |
| `ObjectInfo.c:2245`, `:2273` | `DoEnemyObjectInfo` | |
| `ObjectInfo.c:2339`, `:2371` | `DoFlowerObjectInfo` | |

46 of the 49 are live. Things that **do not** set `fileDirty`, and are therefore lost if the
user quits without another change:

* Selecting a different object or room (correct — nothing changed).
* `KeepObjectLegal`'s silent mutations (§18.17.13) — a clamp applied while drawing does not mark
  the file dirty, so an illegal object can be re-legalised on screen and then not saved.
* `GetObjectRect`'s `kCustomPict` rewrite to `data.g.height = 10000`
  (`GliderPRO/Sources/ObjectRects.c:224`) — the object is silently altered in memory and the
  change is only persisted if something *else* dirties the file.
* `ZeroHighScores` called from `InitializeEmptyHouse` (`House.c:133`) — but `:156` sets the flag
  anyway.

---

### 19.9 `CloseHouse` (`GliderPRO/Sources/HouseIO.c:524-562`)

```
 1. if (!houseOpen) return true;                                              // :528-529
 2. if (gameDirty)                                                            // :531
      if (houseIsReadOnly) { if (!WriteScoresToDisk()) YellowAlert(kYellowFailedWrite,0); } // :533-537
      else if (!WriteHouse(theMode == kEditMode)) YellowAlert(kYellowFailedWrite, 0);       // :538-539
    else if (fileDirty)                                                       // :541
      if (!QuerySaveChanges()) return false;   // user cancelled              // :544-545
 3. CloseHouseResFork();                                                      // :549
 4. CloseHouseMovie();                                                        // :550
 5. theErr = FSClose(houseRefNum);                                            // :552
    on error: CheckFileError(theErr, thisHouseName); return false;
 6. houseOpen = false;                                                        // :559
 7. return true;
```

The `else if` at step 2 is significant: **if `gameDirty` is set, `fileDirty` is never
consulted.** A high score earned during a session where the house was also edited causes an
*unprompted* full save (`WriteHouse(theMode == kEditMode)`) with no "Save changes?" dialog. In
practice `gameDirty` is only set during play, and the editor is not reachable while playing, so
the combination is rare but reachable (edit → save-less switch to play mode → score → quit).

Step 5's failure path returns false with the resource fork already closed and the movie already
disposed, but `houseOpen` still true — the caller in `Main.c:372-379` handles exactly this by
force-closing:

```c
if (houseOpen)
{
    if (!CloseHouse())
    {
        CloseHouseResFork();
        fileErr = FSClose(houseRefNum);
        houseOpen = false;
    }
}
```

---

### 19.10 `QuerySaveChanges` (`GliderPRO/Sources/HouseIO.c:596-632`) and ALRT 1002

```
 1. if (!fileDirty) return true;                                    // :601-602
 2. InitCursor();                                                   // :604
 3. ParamText(thisHouseName, "\p", "\p", "\p");                     // :606
 4. hitWhat = Alert(kSaveChangesAlert /* 1002 */, nil);              // :607
 5. if (hitWhat == kSaveChanges /* 1 */) {                           // :608
      if (wasHouseVersion < kHouseVersion) ConvertHouseVer1To2();    // :610-611
      wasHouseVersion = kHouseVersion;                               // :612
      return WriteHouse(true);                                       // :613-616
    }
 6. else if (hitWhat == kDiscardChanges /* 2 */) {                   // :618
      fileDirty = false;                                             // :620
      if (!quitting) { CloseHouse(); if (OpenHouse()) ReadHouse(); } // :621-626
      UpdateMenus(false);                                            // :627
      return true;                                                   // :628
    }
 7. else return false;      // Cancel (item 3)                       // :631
```

Constants: `kSaveChangesAlert` = **1002**, `kSaveChanges` = **1**, `kDiscardChanges` = **2**
(`GliderPRO/Sources/HouseIO.c:20-22`).

Note step 6's re-load: "Discard" does not just drop the flag, it **closes and re-opens the house
from disk**, so the in-memory model is guaranteed to match the file afterwards — unless we are
`quitting`, in which case there is no point. `CloseHouse()` is re-entered here with `fileDirty`
already false, so it will not recurse into `QuerySaveChanges`.

**ALRT 1002**, verified from `Glider PRO.r`: rect `(top 0, left 0, bottom 128, right 320)`,
DITL 1002, stages `0x5555`. The zero origin means the alert would draw at the top-left of the
screen; the commented-out `CenterAlert(kSaveChangesAlert)` at `:605` was meant to fix that.

**DITL 1002** ("Save Changes?"), 5 items:

| # | type | rect (t,l,b,r) | text / resource |
|---|---|---|---|
| 1 | btnCtrl | 99, 253, 119, 311 | `Save` |
| 2 | btnCtrl | 71, 253, 91, 311 | `Discard` |
| 3 | btnCtrl | 99, 184, 119, 242 | `Cancel` |
| 4 | statText (disabled) | 6, 7, 80, 244 | `You have made changes to ^0 and haven't saved these changes.  Save the house before proceeding?` |
| 5 | iconItem (disabled) | 7, 278, 39, 310 | icon **1000** (raw bytes `03 e8`) |

`^0` is `thisHouseName`. Item order matters: `Alert` returns the item number, and the code maps
1 → Save, 2 → Discard, anything else → Cancel. **Item 1 is the default button** (Return/Enter
activates it) because it is item 1 and stage words `0x5555` select default-button 1 for all four
stages.

---

### 19.11 The House ▸ Save House menu path (`GliderPRO/Sources/Menu.c:452-468`)

```c
case iSave:
DeselectObject();                                   // :453
if (fileDirty)
    SortHouseObjects();                             // :455
if ((fileDirty) && (houseUnlocked))
{
//  SaveGame(false);
    if (wasHouseVersion < kHouseVersion)            // :459
        ConvertHouseVer1To2();                      // :460
    wasHouseVersion = kHouseVersion;                // :461
    whoCares = WriteHouse(true);                    // :462
    ForceThisRoom(thisRoomNumber);                  // :463
    ReadyBackground(thisRoom->background, thisRoom->tiles);  // :464
    GetThisRoomsObjRects();                         // :465
    DrawThisRoomsObjects();                         // :466
}
break;
```

Ordering notes for a port:

1. `DeselectObject()` first, so no marquee is left pointing at an object index that
   `SortHouseObjects` is about to move.
2. `SortHouseObjects()` compacts every room's object list and rewrites the links to match
   (§18.20). Its `numLinks == 0` early return (`GliderPRO/Sources/House.c:459-460`) means
   link-free houses are **not** compacted — proven by "California or Bust!", which ships with 6
   rooms containing gaps in their object arrays.
3. `WriteHouse(true)` then runs the validator, which may itself renumber/compress rooms.
4. Steps `:463-466` rebuild the on-screen room from scratch because both
   `SortHouseObjects` and `CheckHouseForProblems` can have moved the current room's contents out
   from under the display.
5. There is no user feedback on success — no dialog, no sound. The only visible change is that
   House ▸ Save House greys out (`UpdateMenus(false)` → `Menu.c:101-104`).

`iSaveAs` is commented out of the switch (`Menu.c:470-472`), out of both menu-enable helpers
(`:107`, `:112`, `:150`), and no `iSaveAs` constant exists — **there is no Save As in 1.0.4**.

---

### 19.12 Creating a new house

#### 19.12.1 `CreateNewHouse` (`GliderPRO/Sources/House.c:46-100`)

```
 1. NavGetDefaultDialogOptions(&dialogOptions);                                 // :57
 2. NavPutFile(nil, &theReply, &dialogOptions, nil, 'gliH', 'ozm5', nil);       // :58
 3. if (theErr == userCanceledErr) return false;                                // :59-60
 4. if (!theReply.validRecord) return false;                                    // :61-62
 5. AEGetNthPtr(&(theReply.selection), 1, typeFSS, …, &theSpec, sizeof(FSSpec), …) // :64-65
 6. if (theReply.replacing) { FSMakeFSSpec(...,&tempSpec); FSpDelete(&tempSpec); } // :67-77
 7. if (houseOpen) { if (!CloseHouse()) return false; }                         // :79-83
 8. FSpCreate(&theSpec, 'ozm5', 'gliH', theReply.keyScript);                    // :85
    if (!CheckFileError(theErr, "\pNew House")) return false;                   // :86-87
 9. HCreateResFile(theSpec.vRefNum, theSpec.parID, theSpec.name);               // :88
    if (ResError() != noErr) YellowAlert(kYellowFailedResCreate, ResError());   // :89-90
10. PasStringCopy(theSpec.name, thisHouseName);                                 // :92
11. AddExtraHouse(&theSpec);                                                    // :93
12. BuildHouseList();                                                           // :94
13. InitCursor();                                                               // :95
14. if (!OpenHouse()) return false;                                             // :96
15. return true;                                                                // :99
```

Note step 14 does **not** call `ReadHouse` — the file is zero bytes long. The caller supplies
the content:

```c
case iNewHouse:                        // GliderPRO/Sources/Menu.c:444
if (CreateNewHouse())
{
    whoCares = InitializeEmptyHouse();
    OpenCloseEditWindows();
}
break;
```

Step 11 is what makes the just-created file findable: `DoDirSearch` scans the *application's*
directory tree, and the new house may be anywhere the user chose, so it is pushed onto
`extraHouseSpecs` (max 8) and re-inserted at the front of the list by every subsequent
`BuildHouseList`.

The file is created with a resource fork that exists but is empty.

`GetLocalizedString(short index, StringPtr theString)`
(`GliderPRO/Sources/StringUtils.c:321-327`) is just
`GetIndString(theString, kLocalizedStringsID /* 150 */, index)`. The entries used by the
save/load path, dumped verbatim from **STR# 150 "Localized Strings"** (51 strings total):

| index | string | used by |
|---:|---|---|
| 9 | `Name for New House:` | vestigial — Navigation Services supplies its own prompt |
| 10 | `Untitled House` | vestigial |
| 11 | `Enter New-Game Message here (max. 255 characters)` | `InitializeEmptyHouse` → `banner` (`House.c:135-136`) |
| 12 | `Enter Finished-House Message here (max. 255 characters)` | `InitializeEmptyHouse` → `trailer` (`House.c:137-138`) |
| 13 | `Converting 1.0 House to 2.0` | `ConvertHouseVer1To2` message window (`House.c:755-756`) |
| 14 | `Converting Room ` (note trailing space) | `ConvertHouseVer1To2` per-room message (`House.c:768-771`) |
| 15 | `Save copy of house as:` | dead `SaveHouseAs` (`HouseIO.c:253-254`) |
| 16 | `Failed House Shrinkage` | validator (§18) |

#### 19.12.2 `InitializeEmptyHouse` (`GliderPRO/Sources/House.c:108-161`)

```
 1. if (thisHouse != nil) DisposeHandle((Handle)thisHouse);                  // :113-114
 2. thisHouse = (houseHand)NewHandle(sizeof(houseType));                     // :116
    if (thisHouse == nil) { YellowAlert(kYellowUnaccounted, 1); return false; }  // :118-122
 3. HLock; thisHousePtr = *thisHouse;                                        // :124-125
 4. version   = kHouseVersion;   // 0x0200                                   // :127
 5. firstRoom = -1;                                                          // :128
 6. timeStamp = 0L;                                                          // :129
 7. flags     = 0L;                                                          // :130
 8. initial.h = 32; initial.v = 32;                                          // :131-132
 9. ZeroHighScores();                                                        // :133
10. banner  = GetLocalizedString(11);   // "Enter New-Game Message here (max. 255 characters)"      // :135-136
11. trailer = GetLocalizedString(12);   // "Enter Finished-House Message here (max. 255 characters)" // :137-138
12. hasGame = false; nRooms = 0;                                             // :139-140
13. wardBitSet = false; phoneBitSet = false;                                 // :142-143
14. HUnlock;                                                                 // :145
15. numberRooms = 0;                                                         // :147
16. mapLeftRoom = 60; mapTopRoom = 50;                                       // :148-149
17. thisRoomNumber = kRoomIsEmpty /* -1 */; previousRoom = -1;               // :150-151
18. houseUnlocked = true;                                                    // :152
19. OpenMapWindow(); UpdateMapWindow();                                      // :153-154
20. noRoomAtAll = true;                                                      // :155
21. fileDirty = true;                                                        // :156
22. UpdateMenus(false); ReflectCurrentRoom(true);                            // :157-158
23. return true;
```

**Bug:** step 2 uses `NewHandle`, not `NewHandleClear`. Fields that steps 4–13 do not assign are
whatever the Memory Manager handed back: `unusedShort`, `unusedBoolean`, and the entire 40-byte
`savedGame` block. `hasGame = false` makes `savedGame` harmless, but the two "unused" fields
carry heap garbage into the file on the first save. This also means **a fresh house is not
byte-reproducible**. A Go port that zero-fills will differ from the original in exactly those
bytes; that is a strict improvement.

Note step 6: `timeStamp = 0` means bit 0 == 0 means *unlocked*, consistent with step 18.
Note also that `houseUnlocked = true` is set directly rather than derived, and `WriteHouse`
step 7d will therefore clear bit 0 of the real date on the first save.

`ZeroHighScores` (`GliderPRO/Sources/HighScores.c:323-343`) sets
`highScores.banner = thisHouseName` and, for all `kMaxScores` = 10 slots,
`names[i] = "--------------"` (14 hyphens), `scores[i] = 0L`, `timeStamps[i] = 0L`,
`levels[i] = 0`.

---

### 19.13 `ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746-820`)

Version 1 houses stored link destinations without the underground-floor bias. The conversion
re-encodes every link's `where`:

```
 1. CopyThisRoomToRoom(); wasRoom = thisRoomNumber;                       // :753-754
 2. OpenMessageWindow(GetLocalizedString(13));   // "Converting 1.0 House to 2.0"  // :755-756
 3. SpinCursor(3);                                                        // :758
 4. HLock(thisHouse); numRooms = nRooms;                                  // :760-763
 5. for i in 0 .. numRooms-1:                                             // :764
      if (rooms[i].suite == kRoomIsEmpty) continue;                       // :766
      SetMessageWindowMessage(GetLocalizedString(14) + itoa(i));  // "Converting Room N"  // :768-771
      SpinCursor(1);                                                      // :772
      ForceThisRoom(i);                                                   // :774
      for h in 0 .. kMaxRoomObs-1:                                        // :775
        switch (thisRoom->objects[h].what):
          kMailboxLf, kMailboxRt, kFloorTrans, kCeilingTrans,
          kInvisTrans, kDeluxeTrans:                                      // :779-784
            if (data.d.where != -1) {
                ExtractFloorSuite(data.d.where, &floor, &suite);          // :787
                floor += kNumUndergroundFloors;   // +8                   // :788
                data.d.where = MergeFloorSuite(floor, suite);             // :789
            }
          kLightSwitch, kMachineSwitch, kThermostat, kPowerSwitch,
          kKnifeSwitch, kInvisSwitch, kTrigger, kLgTrigger:               // :793-800
            same, through data.e.where                                    // :801-806
      CopyThisRoomToRoom();                                               // :810
 6. (*thisHouse)->version = kHouseVersion;   // 0x0200                    // :814
 7. HSetState; InitCursor(); CloseMessageWindow(); ForceThisRoom(wasRoom); // :815-819
```

`kNumUndergroundFloors` = **8** (`GliderPRO/Headers/GliderDefines.h:535`).
Note `kSoundTrigger` (0x49) is **not** in either case list, so its `where` (a `'snd '` ID) is
correctly left alone.

Both callers gate on `wasHouseVersion < kHouseVersion` (`Menu.c:459`, `HouseIO.c:610`), and
since **all 22 shipped houses report `version == 0x0200`**, this function never runs on shipped
data. A Go port still needs it to accept third-party 1.0 houses. Note step 6 sets `version`
directly *and* the callers set `wasHouseVersion = kHouseVersion` — both are needed because
`WriteHouse` step 7f overwrites `version` from `wasHouseVersion`.

---

### 19.14 `YellowAlert` — the non-fatal error channel

```c
void YellowAlert (short whichAlert, short identifier)      // GliderPRO/Sources/HouseIO.c:640
{
    #define     kYellowAlert    1006                       // :642
    Str255      errStr, errNumStr;
    short       whoCares;

    InitCursor();                                          // :646
    GetIndString(errStr, kYellowAlert, whichAlert);        // :648
    NumToString((long)identifier, errNumStr);              // :649
//  CenterAlert(kYellowAlert);
    ParamText(errStr, errNumStr, "\p", "\p");              // :652
    whoCares = Alert(kYellowAlert, nil);                   // :654
}
```

`kYellowAlert` = **1006** is used as *both* the ALRT resource ID and the STR# resource ID.

**ALRT 1006** (verified from `Glider PRO.r`): rect `(40, 40, 168, 340)` = 128 × 300,
DITL 1006, stages `0x5555`.

**DITL 1006** ("Yellow Alert"), 5 items:

| # | type | rect (t,l,b,r) | text / resource |
|---|---|---|---|
| 1 | btnCtrl | 100, 234, 120, 292 | `Okay` |
| 2 | statText (disabled) | 25, 9, 89, 253 | `^0` — the message |
| 3 | statText (disabled) | 9, 9, 25, 195 | `A problem came up:` |
| 4 | statText (disabled) | 102, 9, 118, 148 | `Error #: ^1` |
| 5 | iconItem (disabled) | 9, 260, 41, 292 | icon **1006** (raw bytes `03 ee`) |

**STR# 1006 "Yellow Alerts"** — all 24 strings, transcribed verbatim, with the constant from
`GliderPRO/Headers/GliderDefines.h:18-41` and every call site (grep-verified across all 92
`.c` files):

| # | constant | string | call sites |
|---:|---|---|---|
| 1 | `kYellowUnaccounted` | `A never-before-seen error has arisen.  Proceed with caution!  (Save and Quit immediately.)` | `House.c:120` (id 1), `HouseIO.c:323` (2), `HouseIO.c:456` (4), `MainWindow.c:181` (6), `Map.c:631` (11), `Map.c:647` (11), `Room.c:203` (theErr) |
| 2 | `kYellowFailedResOpen` | `I failed to open the house's resource fork.  Any unique room backgrounds are not accessible.` | `HouseIO.c:573` |
| 3 | `kYellowFailedResAdd` | `I failed to add a resource to the house's resource fork.  See error number.` | **none — dead string** |
| 4 | `kYellowFailedResCreate` | `I failed to create a new resource fork for the house.  See error number for problem.` | `House.c:90`, `HouseIO.c:273` (dead) |
| 5 | `kYellowNoHouses` | `There are no houses on this drive!  About your only option is to create your own new house with the Editor.` | `SelectHouse.c:620` |
| 6 | `kYellowNewerVersion` | `This house is incompatible with us!  You'll need to upgrade Glider PRO to use this house.  Do not attempt to play/edit this house!` | `HouseIO.c:397` |
| 7 | `kYellowNoBackground` | `The background specified by this room was not found!  Try re-selecting a new background (the Room Info menu).` | `Room.c:275` |
| 8 | `kYellowIllegalRoomNum` | `The room number is out of bounds.  I suspect the house file is corrupt.  Try deleting this "illegal" room though.` | `Room.c:381` |
| 9 | `kYellowNoBoundsRes` | `The data is missing that specifies where the openings in this room are.  The house may be damaged.  Try selecting a new background though.` | `Room.c:946` |
| 10 | `kYellowScrapError` | `There was a problem with the clipboard (Cut, Copy and Paste commands).  I couldn't guess why.` | `Scrap.c:58, 63, 94, 97, 121, 188` — **all dead** |
| 11 | `kYellowNoMemory` | `I think we just ran out of memory.  Quit now and give Glider PRO™ more memory.` | `HouseIO.c:359` (id 10); `Scrap.c:115, 182` dead |
| 12 | `kYellowFailedWrite` | `We failed to write the house to disk.  (That shouldn't have happened.)` | `HouseIO.c:333, 536, 539`, `SavedGames.c:349` |
| 13 | `kYellowNoMusic` | `Well, the music didn't load.  Glider PRO™ will still run, you'll just be musically challenged.` | 10 sites (`Events.c:423`, `Music.c:326, 365`, `Menu.c:396`, `Play.c:91, 242`, `Settings.c:648, 805, 860, 1252`) |
| 14 | `kYellowFailedSound` | `Wow, there was a problem bringing sounds up.  You might try giving Glider PRO™ more memory - otherwise ... silence.` | `Sound.c:461, 470` |
| 15 | `kYellowAppleEventErr` | `Some kind of strange Apple Event error.  I think I would just ignore it.  Or call Casady & Greene with the error number.` | `AppleEvents.c:65, 187, 192, 197, 202, 206`, `Events.c:453` |
| 16 | `kYellowOpenedOldHouse` | `Did you save the house on the same volume Glider PRO is on?  I saved the house but had to re-open the old house because I couldn't find the new one.` | `HouseIO.c:294` — **dead** (inside `SaveHouseAs`) |
| 17 | `kYellowLostAllHouses` | `Wow, I couldn't find the old or new house.  Go to the Select House menu item and see if it's there.  If not, make sure they're on the same volume as Glider PRO.` | `HouseIO.c:298` — **dead** |
| 18 | `kYellowFailedSaveGame` | `Couldn't create a saved game structure.  Memory is probably too low.` | `SavedGames.c:60, 202` |
| 19 | `kYellowSavedTimeWrong` | `The saved game doesn't match the house.  Either this game was saved for a different house or the house was modified recently.` | `SavedGames.c:237` |
| 20 | `kYellowSavedVersWrong` | `This saved game is an old version.  We cannot use this game with this house.` | `SavedGames.c:245` |
| 21 | `kYellowSavedRoomsWrong` | `The number of rooms saved doesn't match the number of house rooms.  We cannot use this game with this house.` | `SavedGames.c:253` |
| 22 | `kYellowQTMovieNotLoaded` | `The QuickTime™ movie that goes with this house will not be used.  Glider PRO™ must have enough memory to easily load the entire movie into RAM.` | `HouseIO.c:90, 98, 107, 117, 127` |
| 23 | `kYellowNoRooms` | `This house has no rooms!  Do not attempt to play this house!  Select a new house to play.` | `HouseIO.c:389` |
| 24 | `kYellowCantOrderLinks` | `There was an error generating or parsing a links list.  Memory may be tight.` | `Objects.c:906` |

Three strings are wholly unreachable in 1.0.4 (3, 16, 17) and one more (10) only from the
commented-out `Scrap.c`. The `™` characters are single-byte Mac Roman `0xAA`.

---

### 19.15 `CheckFileError` — the File Manager error channel

```c
Boolean CheckFileError (short resultCode, StringPtr fileName)   // GliderPRO/Sources/FileError.c:28
```

Returns `true` for `noErr` (`:33-34`), otherwise maps the OSErr to a STR# index, shows ALRT
**140**, and returns `false`. Constants at `FileError.c:14-15`:
`rFileErrorAlert` = 140, `rFileErrorStrings` = 140.

Mapping (`FileError.c:36-89`):

| index | OSErr constant | numeric value | string (STR# 140) |
|---:|---|---:|---|
| 1 | `default` | — | `A miscellaneous input/output error occurred (see error code below).` |
| 2 | `dirFulErr` | −33 | `An error occurred because the directory is full.` |
| 3 | `dskFulErr` | −34 | `An error occurred because the disk is full.  Try another disk.` |
| 4 | `ioErr` | −36 | `An unspecified input output error occurred.` |
| 5 | `bdNamErr` | −37 | `An error occurred because the name you chose is unacceptable.  Try a different name.` |
| 6 | `fnOpnErr` | −38 | `An error occurred because the file is not open.  Call tech support.` |
| 7 | `mFulErr` | −41 | `An error occurred because memory was too full.  Close windows, save, and bail out!` |
| 8 | `tmfoErr` | −42 | `An error occurred because there are too many files open (no more than 12 at once allowed).` |
| 9 | `wPrErr` | −44 | `An error occurred because the disk is write protected.  Save onto an unlocked disk.` |
| 10 | `fLckdErr` | −45 | `An error occurred because the file is locked.  Save under a different name.` |
| 11 | `vLckdErr` | −46 | `An error occurred because the volume you selected is locked.  Save onto an unlocked disk.` |
| 12 | `fBsyErr` | −47 | `An error occurred because the file is busy (perhaps another application has opened it).` |
| 13 | `dupFNErr` | −48 | `An error occurred because the name you chose has already been used.  Save with another name.` |
| 14 | `opWrErr` | −49 | `An error occurred because the file is already open for writing.  Call tech support.` |
| 15 | `volOffLinErr` | −53 | `An error occurred because that volume is off-line.  Save onto another disk.` |
| 16 | `permErr` | −54 | `An error occurred because of a permission violation.  Save onto an unlocked disk.` |
| 17 | `wrPermErr` | −61 | `An error occurred because write permission was denied.  Save onto an unlocked disk.` |

`ParamText(errMessage, errNumString, fileName, "\p")` at `FileError.c:94` fills `^0` (message),
`^1` (the numeric OSErr) and `^2` (the file name).

**ALRT 140**: rect `(92, 60, 220, 446)` = 128 × 386, DITL 140, stages `0x5555`.

**DITL 140** ("File Error"), 5 items:

| # | type | rect (t,l,b,r) | text / resource |
|---|---|---|---|
| 1 | btnCtrl | 92, 313, 112, 371 | `Okay` |
| 2 | statText (disabled) | 29, 14, 89, 295 | `^0` |
| 3 | statText (disabled) | 93, 14, 111, 252 | `(error = ^1)` |
| 4 | iconItem (disabled) | 13, 341, 45, 373 | icon **140** (raw bytes `00 8c`) |
| 5 | statText (disabled) | 6, 14, 23, 332 | `A File Error Loading/Saving ^2` |

The house-file paths that funnel through `CheckFileError` are
`HouseIO.c:179, 190, 263, 268, 277, 344, 367, 375, 463, 494, 502, 555, 678, 693` and
`House.c:71, 75, 86`.

---

### 19.16 High scores as a side-car file (read-only houses)

When `houseIsReadOnly` is true the house cannot be rewritten, so scores are kept in the
Preferences folder instead:

* **Folder**: `Preferences ▸ "G-PRO Scores Ä"` — created by `CreateScoresFolder`
  (`GliderPRO/Sources/HighScores.c:642-661`) via
  `FindFolder(kOnSystemDisk, kPreferencesFolderType, kCreateFolder, …)` then
  `FSpDirCreate(&scoresSpec, smSystemScript, scoresDirID)`. The `Ä` is Mac Roman `0x80`
  (capital A-umlaut) — one byte, not two.
* **File**: named exactly `thisHouseName`, type `'gliS'`, creator `'ozm5'`
  (`OpenHighScoresFile`, `HighScores.c:720-738`: `FSpOpenDF(fsCurPerm)`, and on `fnfErr`
  `FSpCreate(scoreSpec, 'ozm5', 'gliS', smSystemScript)` then re-open).
* **Contents**: exactly `sizeof(scoresType)` = **292 bytes**, the raw `highScores` sub-struct,
  written from `&((*thisHouse)->highScores)` (`WriteScoresToDisk`, `HighScores.c:742-797`:
  `SetFPos` 0, `FSWrite` 292, `SetEOF` 292, `FSClose`).
* **Read back** by `ReadScoresFromDisk` (`:801-855`), which uses `GetEOF` as the byte count —
  so a short or long file is read verbatim into the 292-byte struct. A file longer than 292
  bytes **overruns the struct** and corrupts the `savedGame` field that follows it in
  `houseType`. Not reachable from the shipped code (the writer always writes 292), but a
  hostile file triggers it.

Since `IsFileReadOnly` always returns `false` in 1.0.4 (§19.17.2), the score side-car path is
**never taken** in practice: `ReadHouse` step 26, `ReadHouse` step 2a and `CloseHouse` step 2's
read-only branch are all dead. A Go port that implements real read-only detection will
resurrect them.

---

### 19.17 Dead and disabled code in the save/load path

#### 19.17.1 `SaveHouseAs` (`GliderPRO/Sources/HouseIO.c:241-308`)

The entire body is commented out (`:244` opens the comment, `:306` closes it) behind the marker
`// TEMP - fix this later -- use NavServices (see House.c)` at `:243`. The live body is
`return false;` at `:307`. The commented-out implementation would have:

1. `StandardPutFile(GetLocalizedString(15) /* "Save copy of house as:" */, thisHouseName, &theReply)` (`:253-254`)
2. saved `oldHouse = theHousesSpecs[thisHouseIndex]` (`:257`)
3. `CloseHouseResFork()` + `FSClose(houseRefNum)` (`:259-260`)
4. `FSpCreate(&theReply.sfFile, 'ozm5', 'gliH', theReply.sfScript)` + `HCreateResFile` (`:267-271`)
5. `FSpOpenDF(&theReply.sfFile, fsRdWrPerm, &houseRefNum)` (`:276`)
6. `houseOpen = true; WriteHouse(false)` (`:280-282`)
7. `BuildHouseList()` then `OpenSpecificHouse(&theReply.sfFile)`, falling back to
   `OpenSpecificHouse(&oldHouse)` with `YellowAlert(kYellowOpenedOldHouse)` or, if that also
   fails, `YellowAlert(kYellowLostAllHouses)` (`:286-301`)

Note step 6 would `WriteHouse(false)` — no validation — and would **not** copy the resource
fork, so a "Save As" would have silently dropped every custom background. That is very likely
why it was disabled.

Consequences for the editor: **the only way to duplicate a house is at the Finder level.** The
menu item, its constant, and its handler are all gone (`Menu.c:107, 112, 150, 470-472`).

#### 19.17.2 `IsFileReadOnly` (`GliderPRO/Sources/HouseIO.c:659-707`)

```c
Boolean IsFileReadOnly (FSSpec *theSpec)
{
#pragma unused (theSpec)                      // :661
    return false;                             // :663
/*  … 43 lines commented out (:664-706) …  */
}
```

The commented-out body would have called `PBHGetVInfo` and tested `ioVAtrb & 0x0080`
(software-locked volume) and `0x8000` (hardware-locked volume), then `PBHGetFInfo` and tested
`ioFlAttrib & 0x0001` (locked file). Because it returns false unconditionally,
`houseIsReadOnly` is always false and:

* `ReadHouse` step 26 never forces `houseUnlocked = false`;
* the `WriteScoresToDisk` branches in `ReadHouse` (`:329-336`) and `CloseHouse` (`:533-537`)
  never run;
* opening a house on a locked disk will fail later, at `FSWrite`, surfacing as
  `CheckFileError(wPrErr)`.

#### 19.17.3 `ShiftWholeHouse` (`GliderPRO/Sources/House.c:824-859`)

`#pragma unused (howFar)` at `:826`; opens a message window
`"Shifting Whole House…"` (`:831`), walks every non-tombstone room calling `ForceThisRoom(i)`
and `CopyThisRoomToRoom()`, but the inner per-object loop at `:847-849` is **empty**. Running it
would mark nothing dirty and change nothing, but it would rewrite `thisRoom` for every room.
No live caller.

---

### 19.18 The QuickTime movie side-car

`OpenHouseMovie` (`GliderPRO/Sources/HouseIO.c:67-142`), guarded by `#ifdef COMPILEQT`
(`GliderDefines.h:15`, defined) *and* `thisMac.hasQT` at runtime:

```
 1. theSpec = theHousesSpecs[thisHouseIndex];
    PasStringConcat(theSpec.name, "\p.mov");           // :80-81
 2. FSpGetFInfo(&theSpec, &finderInfo); if error return;  // :83-85  (silent — no movie is normal)
 3. OpenMovieFile(&theSpec, &movieRefNum, fsCurPerm);   // :87
    on error: YellowAlert(kYellowQTMovieNotLoaded, theErr); return;
 4. NewMovieFromFile(&theMovie, movieRefNum, nil, theSpec.name, newMovieActive, &changed);  // :94-95
    on error: YellowAlert; CloseMovieFile; return;
 5. CloseMovieFile(movieRefNum);                        // :102
 6. spaceSaver = NewHandle(307200L);   // 300 KB memory reservation   // :104
    if nil: YellowAlert(kYellowQTMovieNotLoaded, 749); CloseHouseMovie(); return;  // :105-110
 7. GoToBeginningOfMovie(theMovie);                     // :112
 8. LoadMovieIntoRam(theMovie, GetMovieTime(theMovie,0), GetMovieDuration(theMovie), 0);  // :113-114
    on error: YellowAlert; DisposeHandle(spaceSaver); CloseHouseMovie(); return;
 9. DisposeHandle(spaceSaver);                          // :122
10. PrerollMovie(theMovie, 0, 0x000F0000);              // :124   (rate 15.0 in Fixed)
    on error: YellowAlert; CloseHouseMovie(); return;
11. theTime = GetMovieTimeBase(theMovie);               // :132
    SetTimeBaseFlags(theTime, loopTimeBase);            // :133
    SetMovieMasterTimeBase(theMovie, theTime, nil);     // :134
    LoopMovie();                                        // :135
12. GetMovieBox(theMovie, &movieRect);                  // :137
13. hasMovie = true;                                    // :139
```

The magic `749` at `:107` is a made-up identifier for "couldn't reserve 300 KB", displayed as
`Error #: 749`.

`LoopMovie` (`:48-63`) installs a `'LOOP'` user-data item: allocate a 4-byte Handle containing
0, remove every existing `'LOOP'` item (`CountUserDataType` then `RemoveUserData` in a loop),
then `AddUserData(theUserData, theLoop, 'LOOP')`.

`CloseHouseMovie` (`:146-159`) calls `LoadMovieIntoRam(..., flushFromRam)` then `DisposeMovie`,
and always clears `hasMovie`.

The editor does not interact with the movie beyond `OpenHouse`/`CloseHouse`. `tvInRoom` and
`tvWithMovieNumber` are the play-mode hooks. `GliderPRO/Houses/*.mov` are the shipped movies.

---

### 19.19 Every disk-touching call, summarised

| Toolbox call | site(s) | purpose |
|---|---|---|
| `PBGetCatInfo` | `SelectHouse.c:586`, `HighScores.c:691` | directory scan |
| `FSMakeFSSpec` | `SelectHouse.c:596`, `House.c:69`, `HighScores.c:654, 757, 816` | build specs |
| `ResolveAliasFile` | `HouseIO.c:177` | follow an alias to the real house |
| `FSpGetFInfo` | `HouseIO.c:83` | probe for `<name>.mov` |
| `FSpCreate` | `House.c:85`, `HouseIO.c:267` (dead), `HighScores.c:727` | create house / score file |
| `FSpDelete` | `House.c:74` | replace-on-create |
| `HCreateResFile` | `House.c:88`, `HouseIO.c:270` (dead) | empty resource fork |
| `FSpOpenDF` | `HouseIO.c:189`, `:276` (dead), `HighScores.c:724, 730` | open data fork |
| `FSpOpenResFile` | `HouseIO.c:571` | open house resource fork |
| `HOpenResFile` | `SelectHouse.c:102` | peek at a house's icon in the Load dialog |
| `CloseResFile` | `HouseIO.c:586`, `SelectHouse.c:113` | |
| `UseResFile` | `HouseIO.c:575`, `SelectHouse.c:131` | make the house the current res file |
| `GetEOF` | `HouseIO.c:341`, `HighScores.c:823` | size the read |
| `SetFPos` | `HouseIO.c:364, 460`, `HighScores.c:764, 830` | rewind |
| `FSRead` | `HouseIO.c:372`, `HighScores.c:841` | whole-file read |
| `FSWrite` | `HouseIO.c:491`, `HighScores.c:776` | whole-file write |
| `SetEOF` | `HouseIO.c:499`, `HighScores.c:785` | truncate to the written length |
| `FSClose` | `HouseIO.c:552`, `:260` (dead), `Main.c:377`, `HighScores.c:767, 780, 788, 792, 826, 833, 845, 850` | |
| `FindFolder` | `HighScores.c:649, 674` | locate Preferences |
| `FSpDirCreate` | `HighScores.c:656` | create the scores folder |
| `NavPutFile` | `House.c:58` | new-house Save dialog |
| `StandardPutFile` | `HouseIO.c:254` | **dead** |
| `OpenMovieFile` / `NewMovieFromFile` / `CloseMovieFile` | `HouseIO.c:87, 94, 99, 102` | movie side-car |

---

### 19.20 What a Go port must replace

| original mechanism | Go replacement |
|---|---|
| `Handle thisHouse` holding the whole file, `HLock`/`HUnlock`/`MoveHHi`/`HGetState`/`HSetState` | one `[]byte`, or a decoded `type House struct { … Rooms []Room }`. There is no relocation, so all the locking disappears. Keep the original byte length if you want byte-identical round-trips (see Sampler's 2 stray bytes). |
| big-endian `short`/`long`, 2-byte 68k struct alignment | `encoding/binary` with `binary.BigEndian`. **Do not** rely on Go struct layout: `Str27`/`Str255` are length-prefixed byte arrays, `Point` is `{v, h}` with **v first**, and `objectType` is a 2-byte tag plus a 10-byte untyped payload that different `what` values interpret differently. Write explicit encode/decode per struct. |
| `GetHandleSize` as the authoritative byte count | store the original file length alongside the decoded model, or recompute `866 + 348*len(rooms)` and accept the (better) truncation |
| `GetDateTime` 1904 epoch, `& 0x7FFFFFFF`, bit 0 = lock | see the snippet in §19.7.1 |
| Resource fork: `PICT` 3000+/3300+, `bnds`, `snd `, custom Finder icons | a sibling directory or a zip/container. **The original has no way to author these; a port needs an import path.** Suggested mapping: `<house>.d/backgrounds/3000.png` + `3000.bnds` (4 bytes, or JSON `{left,top,right,bottom}`), `<house>.d/sounds/3011.wav`, `<house>.d/picts/10000.png`. Keep the numeric IDs — they are stored *inside* room and object records (`room.background`, `object.data.g.height`, `object.data.e.where` for `kSoundTrigger`) and renumbering means rewriting house data. |
| `'snd '` resource minus a fixed 20-byte header (`Sound.c:286-296`) | if importing real `'snd '` resources, keep the skip; if importing WAV, skip nothing |
| `FSpOpenDF(fsCurPerm)` + `FSWrite` + `SetEOF` whole-file rewrite | `os.WriteFile` to a temp file then `os.Rename` — atomic, which the original is not. The original can leave a truncated house if it crashes between `FSWrite` and `SetEOF`. |
| `ResolveAliasFile` | `filepath.EvalSymlinks` |
| `DoDirSearch` matching `fdType`/`fdCreator` | walk for `*.glh` (or read the 866-byte header and check `version < 0x0300` && `nRooms >= 1`) |
| `IsFileReadOnly` (stubbed) | a real `unix.Access(path, W_OK)` / open-for-write probe. Implementing it correctly **re-enables** three dead code paths (§19.17.2); test them. |
| ALRT 1002 / 140 / 1006 with `ParamText` `^0…^2` | modal dialogs; keep the exact strings from §19.14/§19.15 if you want parity |
| Navigation Services `NavPutFile` | native file dialog |
| QuickTime side-car | optional; the file is `<houseName>.mov` |
| Preferences-folder score side-car (`G-PRO Scores Ä/<houseName>`, 292 raw bytes, type `'gliS'`) | per-user config dir; the 292-byte blob is trivially portable but `ReadScoresFromDisk` trusting `GetEOF` is an overflow — bound the read |

Behavioural details worth preserving deliberately (or deliberately fixing):

1. `WriteHouse` only stamps `timeStamp`/`version` when `fileDirty` — a score-only save leaves the
   modification stamp alone.
2. A `checkIt == false` write happens on the load path (`ReadHouse` step 2b) — the validator is
   skipped there on purpose.
3. `InitializeEmptyHouse` uses `NewHandle`, so a fresh house contains uninitialised
   `unusedShort`, `unusedBoolean` and `savedGame` bytes. Zero-filling is a divergence, and a good
   one.
4. There is **no Save As**, **no copy/paste**, and **no undo** anywhere in the editor. If the
   port adds them, they are new features, not restorations.
5. The only "are you sure?" on quit/switch is ALRT 1002, and it is bypassed whenever `gameDirty`
   is set (§19.9).

---

## 20. Copy/paste, undo, and the other disabled subsystems

Glider PRO 1.0.4 shipped with several editor features present in source but switched off. A porter
reading the source cold will otherwise waste time implementing behaviour that the shipped
application never had, or — worse — will assume a feature exists because its prototype does.
This section enumerates every one of them, states precisely *how* it is disabled, and says what
the observable behaviour actually is.

| subsystem | mechanism | net effect in 1.0.4 |
|---|---|---|
| Clipboard (Cut/Copy/Paste of rooms and objects) | whole of `Scrap.c` wrapped in one `/* … */` (`GliderPRO/Sources/Scrap.c:8` → `:516`); every call site commented out | Cut and Clear delete without copying; Copy does nothing at all; Paste is permanently greyed out and retitled |
| Drag Manager (drag a room in the map window; drop on the Trash to delete it) | same comment block (`DragRoom`, `GliderPRO/Sources/Scrap.c:337-471`); call site `GliderPRO/Sources/Map.c:618` commented | no drag-and-drop anywhere; the *setup* code still runs (§20.2) |
| `MoveRoom` (relocate a room by dropping it on an empty map cell) | live code at `GliderPRO/Sources/Map.c:769-795`, but its only reference is the commented `GliderPRO/Sources/Scrap.c:463` | dead; rooms can only be relocated by retyping floor/suite in Room Info |
| Arrow-key object nudging (`MoveObject`) | `#if BUILD_ARCADE_VERSION` / `#else` / `#endif` at `GliderPRO/Sources/Events.c:191` / `:209` / `:251`, with `#define BUILD_ARCADE_VERSION 1` at `GliderPRO/Headers/GliderDefines.h:16` | arrow keys instead show High Scores / play the demo / start a new game; objects can only be moved with the mouse |
| Arrow-key room stepping (`SelectNeighborRoom`) | same `#if` | dead |
| Undo | never existed | **no undo anywhere in the program** |
| Save As | `SaveHouseAs` body commented (`GliderPRO/Sources/HouseIO.c:244-306`), no `iSaveAs` constant, no such item in MENU 131 | duplicate the house in the Finder instead |
| Copy protection (`ValidInstallation`) | `#define COMPILENOCP` (`GliderPRO/Headers/GliderDefines.h:14`) short-circuits the caller; the surviving function body is `return true;` anyway | no key-disk check |
| `AddMenuToPopUp` | definition commented (`GliderPRO/Sources/DialogUtils.c:548-558`), prototype commented (`GliderPRO/Headers/DialogUtils.h:34`), call commented (`GliderPRO/Sources/RoomInfo.c:434`) | the Room Info background popup has no menu attached and pops up nothing |
| `IsFileReadOnly` | body commented (`GliderPRO/Sources/HouseIO.c:664-706`), `return false;` at `:663` | read-only houses are never detected (§19.17.2) |
| `ShiftWholeHouse` | empty inner loop (`GliderPRO/Sources/House.c:847-849`), no caller | dead |
| `OpenSpecificHouse` | its only caller is inside the dead `SaveHouseAs` | dead |

---

### 20.1 `Scrap.c` — how it is disabled

`GliderPRO/Sources/Scrap.c` is 517 lines. Line 8 is a bare `/*` and line 516 is the matching
`*/`. **Everything between them — including the `#include` directives, the file-scope globals and
all seven functions — is one comment.** The compiled object file is empty.

Structure of the commented-out file (all line numbers verified against the CR→LF conversion):

| lines | contents |
|---|---|
| `:8` | `/*` — opens the comment that swallows the file |
| `:9-11` | `#include "Externs.h"`, `#include "Environ.h"`, `#include <Drag.h>` |
| `:14` | `Boolean DropLocationIsTrash (AEDesc *);` |
| `:17` | `Boolean hasScrap, scrapIsARoom;` |
| `:19-21` | `extern WindowPtr mapWindow; extern Rect roomObjectRects[]; extern short objActive;` |
| `:28-64` | `PutRoomScrap` — additionally inside `#ifndef COMPILEDEMO` (`:27` … `:65`) |
| `:70-98` | `PutObjectScrap` — `#ifndef COMPILEDEMO` (`:69` … `:99`) |
| `:104-164` | `GetRoomScrap` — `#ifndef COMPILEDEMO` (`:103` … `:165`) |
| `:170-216` | `GetObjectScrap` — `#ifndef COMPILEDEMO` (`:169` … `:217`) |
| `:222-255` | `SeeIfValidScrapAvailable` — `#ifndef COMPILEDEMO` (`:221` … `:256`) |
| `:260-300` | `DropLocationIsTrash` — *not* inside any `#ifndef` |
| `:306-332` | `DragTrackingFunc` — additionally inside `#if 0` (`:305` … `:333`); its three case bodies are literally `xxx;` at `:319`, `:323`, `:327` |
| `:337-471` | `DragRoom` |
| `:476-496` | `InitDragInfo` — additionally `#if 0` (`:475` … `:497`); ends `return err;` with no `err` declared |
| `:502-513` | `KillDragInfo` — additionally `#if 0` (`:501` … `:514`); declared `void` yet contains `return (noErr);` |
| `:516` | `*/` |

The last three would not even compile (`xxx;`, an undeclared `err`, a value returned from a
`void`). That is strong evidence the Drag Manager work was abandoned mid-stream and the whole file
commented out so the project would still link.

Prototypes in `GliderPRO/Headers/GliderProtos.h` are commented out to match — **except two**:

```
:433  //void PutRoomScrap (void);              // --- Scrap.c
:434  //void PutObjectScrap (void);
:435  void GetRoomScrap (void);                // <-- still declared!
:436  void GetObjectScrap (void);              // <-- still declared!
:437  //void SeeIfValidScrapAvailable (Boolean);
:439  //Boolean DragRoom (EventRecord *, Rect *, SInt16);
```

`GetRoomScrap` and `GetObjectScrap` remain declared with no definition anywhere in the program.
That still links only because their sole call sites (`GliderPRO/Sources/Menu.c:530` and `:532`)
are themselves inside a `/* … */` block (`Menu.c:529-534`). A porter grepping the headers will
conclude that paste exists. It does not.

#### 20.1.1 What the clipboard *would* have done

Documented because it is the only surviving specification of Glider PRO's clipboard format, and
because a Go port that adds copy/paste should probably follow it.

**`PutRoomScrap` (`GliderPRO/Sources/Scrap.c:28-64`)** — copy the current room:

```
1. theErr = ZeroScrap();                                              // :35
   if (theErr != noErr) { YellowAlert(kYellowScrapError, theErr); return; }   // :62-63
2. SetRect(&largeBounds, 0, 0, kRoomWide, kTileHigh);                 // :38   -> (0,0,512,322)
   SetRect(&smallBounds, 0, 0, kRoomWide / 4, kTileHigh / 4);         // :39   -> (0,0,128,80)
3. smallPict = OpenPicture(&smallBounds);                             // :40
   CopyBits(mainWindow->portBits -> mainWindow->portBits,
            &largeBounds, &smallBounds, srcCopy, nil);                // :41-42
   ClosePicture();                                                    // :43
4. HLock((Handle)smallPict);                                          // :45
   theErr = PutScrap(GetHandleSize((Handle)smallPict), 'PICT', *smallPict);   // :46
5. theErr = PutScrap(sizeof(roomType) /* 348 */, 'Room', (Ptr)thisRoom);      // :47
6. if (theErr == noErr)                                               // :48
     if (!hasScrap) { hasScrap = true; UpdateMenus(false); }          // :50-54
     scrapIsARoom = true;                                             // :55
   else YellowAlert(kYellowScrapError, theErr);                       // :58
7. KillPicture(smallPict);                                            // :60
```

So a copied room is **the raw 348-byte `roomType` under scrap type `'Room'`**, plus a
**128 x 80** quarter-scale PICT preview for interoperability with other applications.
`kRoomWide` = 512 and `kTileHigh` = 322 (`GliderPRO/Headers/GliderDefines.h:499`, `:498`), so
`kTileHigh / 4` truncates to 80. Note step 6: `UpdateMenus` is called only on the
false-to-true *transition* of `hasScrap`. `kYellowScrapError` = **10**
(`GliderPRO/Headers/GliderDefines.h:27`).

**`PutObjectScrap` (`GliderPRO/Sources/Scrap.c:70-98`)** — copy the selected object:

```
1. theErr = ZeroScrap();                                                          // :77
2. GetIndString(kindStr, kObjectNameStrings /* STR# 1007 */,
                thisRoom->objects[objActive].what);                               // :80
3. theErr = PutScrap(kindStr[0], 'TEXT', (Ptr)(kindStr + 1));                      // :81
4. scrapObjPtr = &(thisRoom->objects[objActive]);                                  // :82
   theErr = PutScrap(sizeof(objectType) /* 12 */, 'Obj.', (Ptr)scrapObjPtr);       // :83
5. if (theErr == noErr) { if (!hasScrap) { hasScrap = true; UpdateMenus(false); }
                          scrapIsARoom = false; }                                  // :84-92
   else YellowAlert(kYellowScrapError, theErr);                                    // :94
```

A copied object is **the raw 12-byte `objectType` under scrap type `'Obj.'`** (note the trailing
period — the four-char code is `'O','b','j','.'`), plus its human-readable name as `'TEXT'`
drawn from `STR# kObjectNameStrings` = **1007** (`GliderPRO/Headers/GliderDefines.h:461`), the
144-entry object-name table catalogued in §12. The `what` value indexes STR# 1007 directly with
no bounds check; `what` is always 1..144 for a real object, so this is safe in practice.

**`GetRoomScrap` (`GliderPRO/Sources/Scrap.c:104-164`)** — paste a room over the current one. The
interesting part is the link fix-up:

```
 1. tempRoom = NewHandle(0L);                                          // :112
    if (tempRoom == nil) { YellowAlert(kYellowNoMemory /* 11 */, 0); return; }  // :113-117
 2. theErr = GetScrap(tempRoom, 'Room', &scrapOffset);                 // :119
    if (theErr < 0) { YellowAlert(kYellowScrapError, theErr); }        // :120-121
    else {
 3.   DeselectObject();                                                // :124
 4.   wasFloor = thisRoom->floor; wasSuite = thisRoom->suite;           // :126-127
      destRoomNumber = GetRoomNumber(thisRoom->floor, thisRoom->suite); // :128  <- never used
 5.   HLock(tempRoom);                                                 // :129
      BlockMove(*tempRoom, (Ptr)thisRoom, sizeof(roomType));           // :130  <- whole room replaced
      HUnlock(tempRoom); DisposeHandle(tempRoom);                      // :131-132
 6.   srcRoomNumber = GetRoomNumber(thisRoom->floor, thisRoom->suite);  // :133  (the ORIGIN's number)
 7.   thisRoom->floor = wasFloor; thisRoom->suite = wasSuite;           // :134-135  restore our address
 8.   for (i = 0; i < kMaxRoomObs /* 24 */; i++)                        // :137
        if (ObjectIsLinkTransport(&thisRoom->objects[i]) ||
            ObjectIsLinkSwitch(&thisRoom->objects[i]))                 // :139-140
          linkRoomNumber = GetRoomLinked(&thisRoom->objects[i]);       // :142
          if (linkRoomNumber == srcRoomNumber)   // a link internal to the copied room  // :143
            if (ObjectIsLinkSwitch(&thisRoom->objects[i]))             // :145
              thisRoom->objects[i].data.d.where =
                  (wasSuite * 100) + wasFloor + kNumUndergroundFloors; // :147-148
            else
              thisRoom->objects[i].data.e.where =
                  (wasSuite * 100) + wasFloor + kNumUndergroundFloors; // :152-153
 9.   CopyThisRoomToRoom();                                            // :159
      ReflectCurrentRoom(false);                                       // :160
      fileDirty = true;                                                // :161
      UpdateMenus(false);                                              // :162
    }
```

Four things to note.

1. `(wasSuite * 100) + wasFloor + kNumUndergroundFloors` (with `kNumUndergroundFloors` = **8**,
   `GliderPRO/Headers/GliderDefines.h:535`) is exactly `MergeFloorSuite(wasFloor + 8, wasSuite)`
   written out longhand — the same link encoding documented in §17.

2. **The switch and transport arms are swapped.** At `:145` `ObjectIsLinkSwitch` is true, yet
   `:147` writes `data.d.where` — the *transport* variant — and the `else` (transport) branch at
   `:152` writes `data.e.where`, the *switch* variant. Because `where` sits at payload offset
   6:8 in **both** variants d and e (§3), this is harmless in C — the same two bytes are written
   either way — but it is the third instance of this exact slip in the codebase (cf.
   `GliderPRO/Sources/House.c:421` and `GliderPRO/Sources/Objects.c:971-978`). A Go port with a
   properly discriminated union will get a type error here and must resolve it by writing
   `where` in both arms.

3. Only links *internal to the copied room* are repaired. Links from the copied room out to other
   rooms are carried over verbatim (correct), and links from other rooms *into* the room being
   overwritten are silently orphaned — not repaired, not reported, not even counted in
   `houseErrors`.

4. `destRoomNumber` (`:128`) is computed and never read. Dead local.

**`GetObjectScrap` (`GliderPRO/Sources/Scrap.c:170-216`)** — paste an object:

```
1. tempObjectHand = NewHandle(0L);                                          // :179
   if (tempObjectHand == nil) { YellowAlert(kYellowNoMemory, 0); return; }  // :180-184
2. theErr = GetScrap(tempObjectHand, 'Obj.', &scrapOffset);                 // :186
   if (theErr < 0) YellowAlert(kYellowScrapError, theErr);                  // :187-188
   else {
3.   DeselectObject();                                                      // :191
4.   HLock(tempObjectHand);                                                 // :193
     noPoint.h = 100; noPoint.v = 100;                                      // :194-195
5.   BlockMove(*tempObjectHand, (Ptr)(&tempObject), sizeof(objectType));    // :196
6.   if (AddNewObject(noPoint, tempObject.what, false))                     // :197
     {
       thisRoom->objects[objActive] = tempObject;                           // :199  <- full 12 bytes
       ReadyBackground(thisRoom->background, thisRoom->tiles);              // :200
       GetThisRoomsObjRects();                                              // :201
       DrawThisRoomsObjects();                                              // :202
       SetPort((GrafPtr)mainWindow); InvalRect(&mainWindowRect);            // :203-204
       if (ObjectHasHandle(&direction, &dist))                             // :205
         { StartMarqueeHandled(&roomObjectRects[objActive], direction, dist);  // :207
           HandleBlowerGlider(); }                                          // :208
       else
         StartMarquee(&roomObjectRects[objActive]);                         // :211
     }
     HUnlock(tempObjectHand); DisposeHandle(tempObjectHand);                // :213-214
   }
```

The trick at step 6 is worth copying: rather than find a free slot itself, it calls `AddNewObject`
(§9) to allocate the slot, run `KeepObjectLegal`, and set `objActive`, and then stomps the whole
12 bytes with the clipboard copy. Note `noPoint` is `{v = 100, h = 100}` — a pasted object always
lands at the *same* point, not under the mouse and not offset from the original, so pasting twice
stacks two objects exactly on top of each other. Note also that `fileDirty` is **not** set here;
it is set inside `AddNewObject` (`GliderPRO/Sources/ObjectAdd.c:777`).

**`SeeIfValidScrapAvailable(Boolean updateMenus)` (`GliderPRO/Sources/Scrap.c:222-255`)**:

```
1. hasScrap = false;                                                  // :227
2. tempRoom = NewHandle(0L);
   if (tempRoom != nil) {
     theErr = GetScrap(tempRoom, 'Room', &scrapOffset);               // :232
     if (theErr >= 0) { hasScrap = true; scrapIsARoom = true; }       // :233-237
     DisposeHandle(tempRoom); }                                       // :238
3. tempObject = NewHandle(0L);
   if (tempObject != nil) {
     theErr = GetScrap(tempObject, 'Obj.', &scrapOffset);             // :244
     if (theErr >= 0) { hasScrap = true; scrapIsARoom = false; }      // :245-249
     DisposeHandle(tempObject); }                                     // :250
4. if (updateMenus) UpdateClipboardMenus();                           // :253-254
```

Because the `'Obj.'` probe is second and unconditional, **a scrap holding both flavors always
reports "object"** — `scrapIsARoom` is overwritten. Its two intended call sites are
`GliderPRO/Sources/InterfaceInit.c:144` (startup) and `GliderPRO/Sources/Events.c:431`
(resume/suspend); both are commented out.

#### 20.1.2 What Cut / Copy / Paste / Clear actually do

`DoHouseMenu` (`GliderPRO/Sources/Menu.c:498-547`), verbatim with real line numbers:

```c
		case iCut:                                    // :498
		if (houseUnlocked)
		{
			if (objActive > kNoObjectSelected)        // :501
			{
//				PutObjectScrap();                    // :503
				DeleteObject();                      // :504
			}
			else
			{
//				PutRoomScrap();                      // :508
				DeleteRoom(false);                   // :509
			}
			UpdateClipboardMenus();                  // :511
		}
		break;                                       // :513

		case iCopy:                                  // :515
		if (houseUnlocked)
		{
//			if (objActive > kNoObjectSelected)       // :518
//				PutObjectScrap();                    // :519
//			else
//				PutRoomScrap();                      // :521
			UpdateClipboardMenus();                  // :522
		}
		break;                                       // :524

		case iPaste:                                 // :526
		if (houseUnlocked)
		{
/*			if (scrapIsARoom)                        // :529
				GetRoomScrap();                      // :530
			else
				GetObjectScrap();                    // :532
			UpdateClipboardMenus();                  // :533
*/                                                   // :534
		}
		break;                                       // :536

		case iClear:                                 // :538
		if (houseUnlocked)
		{
			if (objActive > kNoObjectSelected)       // :541
				DeleteObject();                      // :542
			else
				DeleteRoom(false);                   // :544
			UpdateClipboardMenus();                  // :545
		}
		break;                                       // :547
```

Observable behaviour in 1.0.4:

* **Cut is a synonym for Clear**: it destroys with no copy. There is no undo, so Command-X on a
  room is an unrecoverable 348-byte deletion.
* **Copy does nothing at all** except recompute the menu item titles. The menu item stays
  **enabled** (`Menu.c:196`), so it looks live and silently isn't.
* **Paste does nothing** and is always greyed out.
* Because `DeleteRoom(false)` is passed `false`, **Cut Room and Clear Room skip the
  confirmation alert.** `DeleteRoom(Boolean doWarn)` (`GliderPRO/Sources/Room.c:439-485`) only
  calls `QueryDeleteRoom()` (`:451`) when `doWarn` is true, and `QueryDeleteRoom`
  (`GliderPRO/Sources/Room.c:490-500`) is what puts up `Alert(kDeleteRoomAlert /* 1005 */, nil)`
  at `:495` and returns true on item `kYesDoDeleteRoom` = **1** (`Room.c:16-17`). The Delete-key
  path passes `true` (`GliderPRO/Sources/Events.c:257`). **The menu is therefore more destructive
  than the keyboard.**

#### 20.1.3 The House menu, verbatim

`MENU` resource 131 (`kHouseMenuID` = 131, `GliderPRO/Headers/GliderDefines.h:189`), parsed out of
`GliderPRO/Glider PRO.r` with python3: title `House`, `menuID` **131**, `menuProc` **0**
(standard MDEF), `enableFlags` **`0xFFFADF77`**. There are **21 items, and no Undo and no
Save As**. The "enable bit" column is bit *n* of `enableFlags` (bit 0 is the menu title itself,
which is set):

| item | `Externs.h` constant | title | Cmd key | enable bit |
|---:|---|---|:---:|:---:|
| 1 | `iNewHouse` = 1 (`Externs.h:207`) | `New House…` | N | 1 |
| 2 | `iSave` = 2 (`:208`) | `Save House` | S | 1 |
| 3 | — | `-` (separator) | | 0 |
| 4 | `iHouse` = 4 (`:209`) | `House Info…` | | 1 |
| 5 | `iRoom` = 5 (`:210`) | `Room Info…` | R | 1 |
| 6 | `iObject` = 6 (`:211`) | `Object Info…` | I | 1 |
| 7 | — | `-` | | 0 |
| 8 | `iCut` = 8 (`:212`) | `Cut Room` | X | 1 |
| 9 | `iCopy` = 9 (`:213`) | `Copy Room` | C | 1 |
| 10 | `iPaste` = 10 (`:214`) | `Paste Room` | V | 1 |
| 11 | `iClear` = 11 (`:215`) | `Delete Room` | | 1 |
| 12 | `iDuplicate` = 12 (`:216`) | `Duplicate Object` | D | 1 |
| 13 | — | `-` | | 0 |
| 14 | `iBringForward` = 14 (`:217`) | `Bring To Front` | `=` | 1 |
| 15 | `iSendBack` = 15 (`:218`) | `Send To Back` | `-` | 1 |
| 16 | — | `-` | | 0 |
| 17 | `iGoToRoom` = 17 (`:219`) | `Go To Room…` | G | 1 |
| 18 | — | `-` | | 0 |
| 19 | `iMapWindow` = 19 (`:220`) | `Map Window` | M | 1 |
| 20 | `iObjectWindow` = 20 (`:221`) | `Tools Window` | T | 1 |
| 21 | `iCoordinateWindow` = 21 (`:222`) | `Coordinate Window` | K | 1 |

Two observations. First, the resource ships items 8/9/10/11 titled for a **room** selection; the
object titles are substituted at runtime (§20.1.4). Second, item 11's *resource* title is
`Delete Room` but its runtime title is `Clear Room` / `Clear Object` — the resource text is
never seen once a house is open, because `UpdateClipboardMenus` overwrites it.

#### 20.1.4 `UpdateClipboardMenus` (`GliderPRO/Sources/Menu.c:165-235`)

The one piece of the clipboard machinery that *is* live. It retitles Cut/Copy/Clear according to
whether an object or a room is selected, using STR# 150 entries 36–44.

```
 1. if (!houseOpen) return;                                             // :169-170
 2. if (houseUnlocked)                                                  // :172
 2a.  if (objActive > kNoObjectSelected)                                // :174
        iCut   <- GetLocalizedString(36) = "Cut Object"                 // :176-177
        iCopy  <- GetLocalizedString(37) = "Copy Object"                // :178-179
        iClear <- GetLocalizedString(38) = "Clear Object"               // :180-181
        EnableMenuItem(houseMenu, iDuplicate);                          // :182
 2b.  else
        iCut   <- GetLocalizedString(39) = "Cut Room"                   // :186-187
        iCopy  <- GetLocalizedString(40) = "Copy Room"                  // :188-189
        iClear <- GetLocalizedString(41) = "Clear Room"                 // :190-191
        DisableMenuItem(houseMenu, iDuplicate);                         // :192
 2c.  EnableMenuItem(houseMenu, iCut);                                  // :195
      EnableMenuItem(houseMenu, iCopy);                                 // :196
 2d.  //  if (hasScrap) { EnableMenuItem(iPaste);
      //     iPaste <- GetLocalizedString(scrapIsARoom ? 42 : 43) }
      //  else                                                          // :197-211  ALL COMMENTED
      {
        DisableMenuItem(houseMenu, iPaste);                             // :213
        iPaste <- GetLocalizedString(44) = "Nothing To Paste"           // :214-215
      }
 2e.  EnableMenuItem(houseMenu, iClear);                                // :217
      EnableMenuItem(houseMenu, iGoToRoom);                             // :218
      EnableMenuItem(houseMenu, iMapWindow);                            // :219
      EnableMenuItem(houseMenu, iObjectWindow);                         // :220
      EnableMenuItem(houseMenu, iCoordinateWindow);                     // :221
 3. else   // house locked                                              // :223
      DisableMenuItem(houseMenu, …) for iCut, iCopy, iPaste, iClear,
        iDuplicate, iGoToRoom, iMapWindow, iObjectWindow,
        iCoordinateWindow                                               // :225-233
```

Because the `if (hasScrap)` test at `:197` and its whole then-branch (`:198-210`) plus the bare
`else` at `:211` are commented out, the block at `:212-216` becomes **unconditional**: the Paste
item is always disabled and always reads **`Nothing To Paste`**. Strings 42 (`Paste Room`) and 43
(`Paste Object`) are present in the resource but unreachable.

Note step 3: locking a house disables the Map, Tools and Coordinate window items too, so a locked
house cannot even be *browsed* with the palettes.

The clipboard block of `STR# 150` ("Localized Strings", 51 entries), transcribed byte-for-byte
from `GliderPRO/Glider PRO.r`:

| index | string | reachable in 1.0.4? |
|---:|---|---|
| 36 | `Cut Object` | yes |
| 37 | `Copy Object` | yes |
| 38 | `Clear Object` | yes |
| 39 | `Cut Room` | yes |
| 40 | `Copy Room` | yes |
| 41 | `Clear Room` | yes |
| 42 | `Paste Room` | **no** |
| 43 | `Paste Object` | **no** |
| 44 | `Nothing To Paste` | yes — always |

`GetLocalizedString(short index, StringPtr theString)` is
`GetIndString(theString, kLocalizedStringsID /* 150 */, index)`
(`GliderPRO/Sources/StringUtils.c:321-327`; the ID is `#define`d inside the function body at
`:323`).

---

### 20.2 The Drag Manager

`DragRoom(EventRecord *theEvent, Rect *roomSrc, short roomNumber)`
(`GliderPRO/Sources/Scrap.c:337-471`) was the intended way to move and delete rooms in the map
window. Its only call site is commented out:

```c
			if (thisMac.hasDrag)                                        // Map.c:613
			{
				SetPortWindowPort(mainWindow);                          // :615
				QSetRect(&aRoom, 0, 0, kMapRoomWidth, kMapRoomHeight);   // :616
				CenterRectOnPoint(&aRoom, globalWhere);                  // :617
//				if (DragRoom(theEvent, &aRoom, itsNumber))               // :618
//				{		// TEMP disabled.                              // :619
//				}                                                        // :620
			}
```

Note that `Map.c:615-617` are **live**: on every map-window click, if the Drag Manager is present,
`HandleMapClick` still sets the port, builds `aRoom`, and centres it on the click — then throws
the result away. Harmless, but a Go port should not reproduce it. The prototype is commented at
`GliderPRO/Headers/GliderProtos.h:439`.

`DragRoom` itself is documented here because it reveals the intended map-window UX:

```
 1. if (thisMac.hasDrag) { … } else fall straight through to `return (true);`  // :351, :470
 2. if (!WaitMouseMoved(theEvent->where)) return (false);               // :353-354  -> a click, not a drag
 3. SetPort(mainWindow); BeginUpdate; UpdateMainWindow(); EndUpdate;    // :356-359
 4. theErr = NewDrag(&theDrag); if (theErr) return (false);             // :361-363
 5. wasState = HGetState(thisHouse); HLock(thisHouse);                  // :365-366
    theRoom = &((*thisHouse)->rooms[roomNumber]);                       // :367
    AddDragItemFlavor(theDrag, (ItemReference)roomNumber, 'Room',
                      (Ptr)theRoom, sizeof(roomType), 0);               // :369-371
    on failure: HSetState; DisposeDrag; return (false);                 // :372-377
 6. SetRect(&largeBounds, 0,0, kRoomWide, kTileHigh);                   // :379
    SetRect(&smallBounds, 0,0, kRoomWide/4, kTileHigh/4);               // :380  -> 128 x 80
    smallPict = OpenPicture(&smallBounds); CopyBits(…); ClosePicture(); // :381-384
    HLock(smallPict);
    AddDragItemFlavor(theDrag, roomNumber, 'PICT', *smallPict,
                      GetHandleSize((Handle)smallPict), 0);             // :386-388
    HUnlock(smallPict); KillPicture(smallPict);                         // :389-390
    HSetState(thisHouse, wasState);                                     // :392
 7. SetDragItemBounds(theDrag, roomNumber, roomSrc);                    // :399
 8. boundsRgn = NewRgn(); RectRgn(boundsRgn, roomSrc);                  // :406-407
    tempRgn = NewRgn(); CopyRgn(boundsRgn, tempRgn);
    InsetRgn(tempRgn, 1, 1); DiffRgn(boundsRgn, tempRgn, boundsRgn);
    DisposeRgn(tempRgn);                                                // :409-413   -> a 1-px outline
 9. theErr = TrackDrag(theDrag, theEvent, boundsRgn);                   // :415
    if (theErr && theErr != userCanceledErr) -> clean up, return (true)  // :417-422
10. GetDragAttributes(theDrag, &attributes);                            // :424
    GetDropLocation(theDrag, &dropLocation);                            // :432
    GetDragModifiers(theDrag, 0L, &mouseDnMods, &mouseUpMods);          // :440
    (each with its own clean-up-and-return-true failure arm)            // :425-446
11. copyRoom = (mouseDnMods | mouseUpMods) & optionKey;                 // :448   <- Option = copy
12. if (!(attributes & kDragInsideSenderApplication))                   // :450   dropped in ANOTHER app
      if ((!copyRoom) && (DropLocationIsTrash(&dropLocation)))          // :452
      { DeselectObject();                                               // :454
        DeleteRoom(true); }                                             // :455   <- WITH confirmation
    else if (attributes & kDragInsideSenderWindow)                      // :458
    {
//    SetPort(mapWindow);                                               // :460
//    GetDragMouse(theDrag, &dragPoint, 0L);                            // :461
//    GlobalToLocal(&dragPoint);                                        // :462
//    MoveRoom(dragPoint);                                              // :463   <- never finished
    }
13. DisposeRgn(boundsRgn); DisposeDrag(theDrag);                        // :466-467
14. return (true);                                                      // :470
```

So: drag a room onto the Finder's Trash to delete it (with the ALRT 1005 confirmation, since
`DeleteRoom(true)`); hold Option to suppress the delete; drag it inside the map window to
relocate it. The second half — the four lines that would have called `MoveRoom` — was never
uncommented, which is precisely *why* `MoveRoom` has no live caller.

`DropLocationIsTrash(AEDesc *dropLocation)` (`GliderPRO/Sources/Scrap.c:260-300`) is the
supporting helper:

```
 1. if (dropLocation->descriptorType != typeNull &&
        AECoerceDesc(dropLocation, typeFSS, &dropSpec) == noErr)         // :269-270
 2.   HLock(dropSpec.dataHandle); theSpec = (FSSpec *)*dropSpec.dataHandle;  // :272-273
 3.   thePB.dirInfo = { ioCompletion 0, ioNamePtr &theSpec->name,
         ioVRefNum theSpec->vRefNum, ioFDirIndex 0, ioDrDirID theSpec->parID }  // :275-279
 4.   theErr = PBGetCatInfo(&thePB, false);                              // :281
 5.   HUnlock(dropSpec.dataHandle); AEDisposeDesc(&dropSpec);            // :283-284
 6.   if (theErr != noErr) return (false);                               // :286-287
 7.   if (!(thePB.dirInfo.ioFlAttrib & (1 << 4))) return (false);         // :289-290  must be a DIRECTORY
 8.   FindFolder(theSpec->vRefNum, kTrashFolderType, kCreateFolder,
                 &trashVRefNum, &trashDirID);                            // :292-293
 9.   if (thePB.dirInfo.ioDrDirID == trashDirID) return (true);           // :295-296
10. return (false);                                                      // :299
```

`DragTrackingFunc` (`:306-332`), `InitDragInfo` (`:476-496`) and `KillDragInfo` (`:502-513`) are
additionally wrapped in `#if 0` and are non-compiling stubs. Nothing else in the program consults
`thisMac.hasDrag` for editing purposes.

---

### 20.3 `MoveRoom` (`GliderPRO/Sources/Map.c:769-795`)

Live code, correct-looking, and unreachable. Verbatim:

```c
void MoveRoom (Point wherePt)                                    // :769
{
	short		localH, localV;
	short		roomH, roomV, itsNumber;

	localH = wherePt.h / kMapRoomWidth;                          // :774
	localV = wherePt.v / kMapRoomHeight;                         // :775

	if ((localH >= mapRoomsWide) || (localV >= mapRoomsHigh))     // :777
		return;                                                  // :778

	roomH = localH + mapLeftRoom;                                // :780
	roomV = kMapGroundValue - (localV + mapTopRoom);              // :781

	if (RoomExists(roomH, roomV, &itsNumber))                    // :783
	{
		                                                         // :785   EMPTY BLOCK
	}
	else
	{
		thisRoom->floor = roomV;                                 // :789
		thisRoom->suite = roomH;                                 // :790
		fileDirty = true;                                        // :791
		UpdateMenus(false);                                      // :792
		RedrawMapContents();                                     // :793
	}
}
```

Five notes for anyone reviving it.

1. The occupied-cell branch (`:784-786`) is an **empty block** — dropping onto an occupied cell is
   a silent no-op. That is the intended behaviour, just expressed oddly.
2. It changes only `thisRoom->floor` / `thisRoom->suite` and does **not** call
   `CopyThisRoomToRoom()`, relying on a later flush (§16) to push the change into
   `(*thisHouse)->rooms[thisRoomNumber]`.
3. It does not fix up any link that points *at* the room by its old (floor, suite). Reviving
   `MoveRoom` without also rewriting inbound links would break every mailbox, switch and
   staircase aimed at the moved room.
4. `RoomExists(roomH, roomV, …)` is called with **(suite, floor)** in that order — the opposite
   of `GetRoomNumber(floor, suite)`. See §21 for that trap.
5. The map's vertical transform is `roomV = kMapGroundValue - (localV + mapTopRoom)`, the inverse
   of the `v = kMapGroundValue - floor` used everywhere else (§15). `kMapGroundValue` = **56**.

---

### 20.4 Arrow-key nudging: `MoveObject` and `SelectNeighborRoom`

`GliderPRO/Sources/Events.c` compiles exactly one of two arrow-key blocks:

```c
#if BUILD_ARCADE_VERSION                                      // :191

			case kLeftArrowKeyASCII:
			DoOptionsMenu(iHighScores);                       // :194
			break;

			case kRightArrowKeyASCII:
			DoOptionsMenu(iHelp);                             // :198
			break;

			case kUpArrowKeyASCII:
			DoGameMenu(iNewGame);                             // :202
			break;

			case kDownArrowKeyASCII:
			DoGameMenu(iNewGame);                             // :206
			break;

#else                                                         // :209

			case kLeftArrowKeyASCII:                          // :211
			if (houseUnlocked)
			{
				if (objActive == kNoObjectSelected)
					SelectNeighborRoom(kRoomToLeft);          // :215
				else
					MoveObject(kBumpLeft, shiftDown);         // :217
			}
			break;
			… kRightArrowKeyASCII :221-229 (:225 / :227) …
			… kUpArrowKeyASCII    :231-239 (:235 / :237) …
			… kDownArrowKeyASCII  :241-249 (:245 / :247) …

#endif                                                        // :251
```

`BUILD_ARCADE_VERSION` is **1** (`GliderPRO/Headers/GliderDefines.h:16`), so the **arcade** block
wins. In the shipped 1.0.4, pressing an arrow key **while editing a house**:

| key | handler | what it actually does |
|---|---|---|
| left | `DoOptionsMenu(iHighScores)` (`iHighScores` = 3, `Externs.h:204`) | `DoHighScores()` (`Menu.c:417-420`) — shows the high-score display |
| right | `DoOptionsMenu(iHelp)` (`iHelp` = 5, `Externs.h:206`) | `DoDemoGame()` (`Menu.c:427-429`) — **starts playing Demo House** |
| up | `DoGameMenu(iNewGame)` (`iNewGame` = 1, `Externs.h:198`) | `twoPlayerGame = false; resumedSavedGame = false; NewGame(kNewGameMode);` (`Menu.c:305-309`) — **starts a new game** |
| down | `DoGameMenu(iNewGame)` | identical to up |

That is almost certainly not intended for the editor: an accidental arrow key abandons editing and
drops into play mode. `NewGame` goes through the normal new-game path, which will handle the
dirty house per §19, but the editing session's selection and marquee are gone.

The `kDeleteKeyASCII` case immediately after the `#endif` (`GliderPRO/Sources/Events.c:253-261`)
is **not** inside the conditional, so Delete still works: `DeleteRoom(true)` at `:257` when
nothing is selected (with the ALRT 1005 confirmation) and `DeleteObject()` at `:259` otherwise.
Also live and outside the conditional: `kHelpKeyASCII` is a no-op (`:178-179`),
`kPageUpKeyASCII` -> `PrevToolMode()` (`:181-184`), `kPageDownKeyASCII` -> `NextToolMode()`
(`:186-189`), and `if ((commandDown) && (!optionDown)) DoMenuChoice(MenuKey(theChar));`
(`:172-173`).

`MoveObject(short whichWay, Boolean shiftDown)` (`GliderPRO/Sources/ObjectEdit.c:1374-1772`) is
therefore dead — 399 lines of per-object-class nudging logic whose only references are
`Events.c:217`, `:227`, `:237`, `:247`. It is worth reading if a port wants keyboard nudging:

* `#ifndef COMPILEDEMO` at `:1376`; `if (theMode != kEditMode) return;` at `:1382-1383`;
  `StopMarquee();` at `:1385`.
* `increment` = **10** when `shiftDown` (`:1388`); otherwise **1** for
  `kInitialGliderSelected` (`:1391-1394`); otherwise a per-object-class value chosen by a
  `switch (thisRoom->objects[objActive].what)` beginning at `:1397-1400`, so that e.g. a floor
  vent steps by its own grid.
* A second giant `switch` applies `deltaH` / `deltaV` to the correct payload variant — e.g.
  `data.i.bounds.left += deltaH; data.i.bounds.right += deltaH;` for `kMousehole` / `kFireplace`
  at `:1691-1695`.
* `if (KeepObjectLegal()) { }` — an **empty** `if` body at `:1700-1702`; the return value is
  computed and discarded.
* `fileDirty = true; UpdateMenus(false); GetThisRoomsObjRects();` at `:1703-1705`.
* Then class-specific invalidation: whole-window `InvalWindowRect(mainWindow, &mainWindowRect)`
  for the 16 object classes listed at `:1726-1742` (`kTiki`, `kTable`, `kShelf`, `kCabinet`,
  `kDeckTable`, `kStool`, `kCounter`, `kDresser`, `kGreaseRt`, `kGreaseLf`, `kSlider`,
  `kMailboxLf`, `kMailboxRt`, `kTrackLight`, `kMirror`, `kWallWindow`), and the union of
  `wasRect` + the new rect for everything else (`:1745-1748`).
* `ReadyBackground(); DrawThisRoomsObjects();` at `:1752-1753`, then the marquee is restarted
  at `:1755-1770` (with the three glider pseudo-object special cases at `:1762-1767`).

`SelectNeighborRoom(short whichNeighbor)` (`GliderPRO/Sources/Room.c:544`) is likewise dead — its
only references are the four compiled-out lines `Events.c:215`, `:225`, `:235`, `:245`.

---

### 20.5 There is no undo

Grepping every `.c` and `.h` in the tree for `Undo` / `undo` yields exactly one symbol:
**`UndoGliderLimbo`** (`GliderPRO/Sources/Modes.c:472`, declared at
`GliderPRO/Headers/GliderProtos.h:216`, called only from `GliderPRO/Sources/Transit.c:162`,
`:163`, `:194`, `:195`, `:229`, `:230`, `:261`, `:262`) — a **play-mode** function that clears a
glider's limbo flag after a room transit. It has nothing to do with editing.

Concretely, there is:

* no `iUndo` constant in `GliderPRO/Headers/Externs.h` (the House-menu block is `:207-222`, and
  it jumps straight from `iObject` = 6 to `iCut` = 8);
* no Undo item in `MENU` 131 (§20.1.3) nor in any other `MENU` resource in `Glider PRO.r`;
* no snapshot, journal, or before-image buffer anywhere in the program;
* no Command-Z handler (`MenuKey('z')` resolves to nothing).

Every destructive operation is immediate and permanent in memory. The **only** recovery is to
close the house without saving — `QuerySaveChanges` -> **Discard**
(`GliderPRO/Sources/HouseIO.c:618-629`), which sets `fileDirty = false` at `:620` and then
re-opens and re-reads the file from disk. That is an all-or-nothing revert of the entire editing
session.

Operations that are destructive with no undo, tabulated with their confirmation status:

| operation | citation | confirmation? |
|---|---|---|
| `DeleteObject()` — Delete key, `iCut`, `iClear` | `GliderPRO/Sources/ObjectEdit.c:1154`; `fileDirty` at `:1179` | **none** |
| `DeleteRoom(false)` — menu Cut Room / Clear Room | `GliderPRO/Sources/Menu.c:509`, `:544` -> `Room.c:439` | **none** |
| `DeleteRoom(true)` — Delete key with no object selected | `GliderPRO/Sources/Events.c:257` -> `Room.c:451` -> `QueryDeleteRoom` `Room.c:490-500` | ALRT **1005**, OK = item 1 |
| `ZeroHighScores()` / `ZeroAllButHighestScore()` | `GliderPRO/Sources/HouseInfo.c:319-340`; `Alert(kZeroScoresAlert /* 1032 */, nil)` at `:324`; item 2 = zero all (`:329`), item 3 = keep highest (`:335`) | its own 3-button alert |
| Room Info changing floor/suite (relocates the room) | `GliderPRO/Sources/RoomInfo.c:484` sets `fileDirty` | **none** |
| `SortHouseObjects` renumbering objects on save | `GliderPRO/Sources/House.c:448-503` | **none** |
| `CheckHouseForProblems` compressing rooms / lopping trailing tombstones | §18 | messages only, no veto |

A Go port that adds undo should record before-images at the `fileDirty = true` sites enumerated in
§19.8 — the 46 live sites out of 49 are exactly the mutation points, and a whole-house
before-image is cheap: the largest shipped house, **Teddy World**, is
**185 654 bytes** of data fork (866 + 348 x 531 rooms), verified by parsing the BinHex in
`GliderPRO/Houses/`. A naive full-state undo stack costing ~180 KB per step is entirely
affordable on modern hardware; it was not on a 4 MB 1994 Macintosh, which is very likely why undo
was never attempted.

---

### 20.6 Copy protection (`GliderPRO/Sources/Validate.c`)

The whole file sits inside `#ifndef COMPILEDEMO` (`:13` … `:397`) and `COMPILEDEMO` is commented
out (`GliderPRO/Headers/GliderDefines.h:12`), so the file *is* nominally compiled. Constants that
survive (`:16-23`):

| constant | value | line |
|---|---|---|
| `kEncryptMask` | `0x05218947` | `:16` |
| `kLegalVolumeCreation` | `0xAA2D3E41` | `:17` |
| `kMasterDialogID` | 1026 | `:18` |
| `kMasterFinderButton` | 1 | `:19` |
| `kMasterNetOnlyButton` | 2 | `:20` |
| `kMasterUserBalloon` | 3 | `:21` |
| `kMasterTitleLeft` | 6 | `:22` |
| `kMasterTitleTop` | 16 | `:23` |

File-scope globals: `long encryptedNumber;` (`:37`), `short theSystemVol;` (`:38`),
`Boolean legitMasterDisk, bailOut, didValidation;` (`:39`).

Lines `:41` (`/*`) through `:365` (`*/`) are one comment containing nine functions —
`GetSystemVolume` `:47`, `VolumeCreated` `:63`, `VolumeMatchesPrefs` `:96`, `NoFloppyException`
`:125`, `GetIndVolumeDate` `:144`, `LoopThruMountedVolumes` `:165`, `SpecificVolumeCreated`
`:191`, `MasterFilter` `:244`, `GetMasterDisk` `:326` — implementing a key-disk scheme that read
each mounted volume's creation date, encrypted it with `kEncryptMask`, and compared the result
against a value stored in the preferences.

The surviving entry point:

```c
Boolean ValidInstallation (Boolean returnToFinder)     // :370
{
#pragma unused (returnToFinder)                        // :372

	return true;                                       // :374
	/*                                                 // :375
	long		actualEncrypted;
	Boolean		isValid;

	theSystemVol = GetSystemVolume();                  // :379
	isValid = VolumeMatchesPrefs(encryptedNumber, &actualEncrypted);   // :380
	if (!isValid) isValid = NoFloppyException();       // :381-382
	if (!isValid) isValid = LoopThruMountedVolumes();  // :383-384
//		isValid = SpecificVolumeCreated();             // :385
	if (!isValid) isValid = GetMasterDisk();           // :386-387
	if (bailOut && returnToFinder) ExitToShell();      // :388-389
	if (isValid && !bailOut) encryptedNumber = actualEncrypted;   // :390-391

	return (isValid);                                  // :393
	*/                                                 // :394
}
```

And it is not even called. `GliderPRO/Sources/Main.c:304-316`, verbatim:

```c
#if defined COMPILEDEMO
	copyGood = true;                                   // :305
#elif defined COMPILENOCP
//	didValidation = false;                             // :307
	copyGood = true;                                   // :308
#else
	didValidation = false;                             // :310
	copyGood = ValidInstallation(true);                // :311
	if (!copyGood)
		encryptedNumber = 0L;                          // :313
	else if (didValidation)
		WriteOutPrefs();				SpinCursor(3);  // :315
#endif                                                 // :316
```

`COMPILENOCP` is defined (`GliderPRO/Headers/GliderDefines.h:14`), so the second arm compiles and
`copyGood` is unconditionally `true`. A Go port should delete this concept entirely;
`encryptedNumber` survives only as a field in the on-disk preferences record.

---

### 20.7 `AddMenuToPopUp` and the orphaned Room Info background popup

The Room Info dialog's background chooser is a custom popup control (CDEF 128, `procID` **2050**;
see §13.4). A popup CDEF finds its `MenuHandle` in the control's `contrlRfCon`. The one function
that would install it is commented out at **both** ends:

```c
//--------------------------------------------------------------  AddMenuToPopUp   // DialogUtils.c:545
// Assigns a menu handle to a pop-up dialog item - thus, giving that…
// pop-up item something to pop up.
/*                                                                                 // :548
void AddMenuToPopUp (DialogPtr theDialog, short whichItem, MenuHandle theMenu)      // :549
{
	Rect		iRect;
	Handle		iHandle;
	short		iType;

	GetDialogItem(theDialog, whichItem, &iType, &iHandle, &iRect);                   // :555
	(**(ControlHandle)iHandle).contrlRfCon = (long)theMenu;                          // :556
}
*/                                                                                 // :558
```

The prototype is commented at `GliderPRO/Headers/DialogUtils.h:34`. The call site:

```c
	// Fix this later.  TEMP                                              // RoomInfo.c:433
//	AddMenuToPopUp(roomInfoDialog, kRoomPopupItem, backgroundsMenu);       // :434
```

Meanwhile `DoRoomInfo` (`GliderPRO/Sources/RoomInfo.c:376`) still does all the setup:

* `#define kBackgroundsMenuID 140` at `:379`;
* `MenuHandle backgroundsMenu;` at `:381`;
* `backgroundsMenu = GetMenu(kBackgroundsMenuID);` at `:397`;
* `// SetMenuItemTextStyle(backgroundsMenu, kOriginalArtworkItem, italic);` at `:398`
  (also commented);
* `if (HouseHasOriginalPicts()) EnableMenuItem(backgroundsMenu, kOriginalArtworkItem);`
  at `:399-400`;
* `kRoomPopupItem` = **11** (`:24`), `kOriginalArtworkItem` = **19** (`:31`);
* and then, at `:435-439`, it still sets the control's *value*:
  `SetPopUpMenuValue(roomInfoDialog, kRoomPopupItem, kOriginalArtworkItem)` if
  `tempBack >= kUserBackground` (3000), else
  `SetPopUpMenuValue(roomInfoDialog, kRoomPopupItem, (tempBack - kBaseBackgroundID) + 1)`.

It never attaches the menu and never releases it. Three consequences, each verified:

1. The popup's `contrlRfCon` stays whatever the `CNTL` resource says. Parsing `CNTL` 128 out of
   `GliderPRO/Glider PRO.r` gives the 23 bytes
   `00000000 0014 00be 0001 01 00 0012 0001 0802 00000000 00`, i.e. `boundsRect` = (0, 0, 20, 190),
   `value` = 1, `visible` = 1, `max` = **18**, `min` = **1**, `procID` = **2050**,
   `refCon` = **0**, empty title. So `contrlRfCon` is **0** — a NULL `MenuHandle`. Clicking the
   popup pops up nothing. (`max` = 18 matches `kNumBackgrounds` = 18,
   `GliderPRO/Headers/GliderDefines.h:521`; the 19th item, "Original Artwork", is outside the
   control's declared range as well.)
2. `backgroundsMenu` is a leaked `GetMenu` handle. `RoomInfo.c` contains no `DisposeMenu` and no
   `ReleaseResource((Handle)backgroundsMenu)`; grep shows `backgroundsMenu` at only `:381`,
   `:397`, `:398` (comment), `:400`, `:434` (comment), and the file's four `ReleaseResource`
   calls (`:851`, `:854`, `:864` — itself commented out — and `:889`) all take PICT and
   `resHandle` variables. **One MENU handle leaks per Room Info invocation.**
3. The current background is still *displayed* only insofar as the CDEF can draw a value it has
   no menu for. In practice the popup renders as an effectively empty box; the user must reach
   backgrounds through the "Select Original Art" sub-dialog (DLOG 1016, §13.9) or by typing a
   PICT ID.

The menu that *would* have been attached is `MENU` 140, and it is worth recording verbatim
because a Go port has to build this chooser from scratch. Parsed from `Glider PRO.r`: title
`Rooms`, `menuID` **133** (**not** 140 — `MENU` 141 "Tools" *also* declares `menuID` 133, a
separate bug), `menuProc` 0, `enableFlags` **`0xFFF7FFFF`** — i.e. every item enabled *except*
bit 19, so item 19 "Original Artwork" ships disabled and is switched on at run time only when
`HouseHasOriginalPicts()` is true (`RoomInfo.c:399-400`). 19 items, no command keys, no marks,
no styles:

| item | title | | item | title |
|---:|---|---|---:|---|
| 1 | `Simple Room` | | 11 | `Skywalk` |
| 2 | `Paneled Room` | | 12 | `Dirt` |
| 3 | `Basement` | | 13 | `Meadow` |
| 4 | `Child's Room` | | 14 | `Field` |
| 5 | `Asian Room` | | 15 | `Roof` |
| 6 | `Unfinished Room` | | 16 | `Sky` |
| 7 | `Swinger's Room` | | 17 | `Stratosphere` |
| 8 | `Bathroom` | | 18 | `Stars` |
| 9 | `Library` | | 19 | `Original Artwork` (disabled by default) |
| 10 | `Garden` | | | |

Items 1–18 map to PICT IDs `kBaseBackgroundID + (item - 1)` = 2000…2017, matching the
`SetPopUpMenuValue` arithmetic `(tempBack - kBaseBackgroundID) + 1` at `RoomInfo.c:438-439`
(`kBaseBackgroundID` = 2000, `GliderPRO/Headers/GliderDefines.h:519`; `kFirstOutdoorBack` = 2009
at `:520`, which is item 10 `Garden` — so items 1–9 are indoor and 10–18 outdoor). Item 19 is the
escape hatch into user artwork (PICT ID >= `kUserBackground` = 3000, `:522`).

This is the single most visible unfinished feature in the editor.

---

### 20.8 What a Go port should conclude

1. **Do not port `Scrap.c` as-is.** Port the *format* — `'Room'` = raw 348-byte `roomType`,
   `'Obj.'` = raw 12-byte `objectType`, `'PICT'` = 128 x 80 preview, `'TEXT'` = the STR# 1007
   name — if interoperability with the original matters. Otherwise implement the clipboard as an
   internal typed value; a Go port has no reason to round-trip through an OS scrap.
2. **Fix the swapped variant arms in `GetRoomScrap`** (`GliderPRO/Sources/Scrap.c:145-154`) rather
   than reproducing them; with a properly discriminated union the original code does not
   typecheck.
3. **Repair inbound links on paste, not just self-links.** The original leaves every link that
   pointed *into* the overwritten room dangling and does not even increment `houseErrors`.
4. **Cut must copy.** The shipped behaviour (Cut is identical to Clear) is a bug, not a design
   decision.
5. **Add the confirmation to menu-driven room deletion**, or at least make `iCut` / `iClear` and
   the Delete key behave identically. Today the keyboard is safer than the menu.
6. **Do not leave Copy enabled if it does nothing.** Either implement it or disable the item.
7. **Implement undo.** The 46 live `fileDirty = true` sites (§19.8) are a ready-made list of
   mutation points to snapshot, and 185 654 bytes is the worst-case whole-house before-image.
8. **Restore arrow-key nudging** (`MoveObject`, the `#else` arm at
   `GliderPRO/Sources/Events.c:211-249`) and arrow-key room stepping (`SelectNeighborRoom`). The
   arcade key bindings are actively harmful in edit mode: right-arrow plays the demo and
   up/down-arrow start a new game, discarding the editing context.
9. **Build the background chooser.** `AddMenuToPopUp` never shipped, `CNTL` 128 has `refCon` 0,
   and `MENU` 140 leaks. Use the 19-item table above and add a searchable list of user PICT
   IDs in 3000…3799 (§19.2.2).
10. **Delete the copy-protection concept entirely**, keeping `encryptedNumber` in the preferences
    record only if you need to read 1994-era prefs files at all.
11. `MoveRoom` and `DragRoom` together describe a map-window drag-to-move / drag-to-Trash UX that
    was designed but never wired up. If you implement it, remember point 3: rewriting a room's
    (floor, suite) invalidates every link that addresses it as
    `(suite * 100) + floor + kNumUndergroundFloors`.

---

## 21. Bug and quirk catalogue

Sections 1–20 describe what the editor is *supposed* to do. This section is the counter-list:
every defect, dead branch, silent no-op, and resource-level inconsistency in the editor code
path that a Go port must make a conscious decision about. It exists because several of these
defects are load-bearing — the shipped houses contain data that only exists *because* the
1994 editor was buggy, and a Go port that "does the right thing" will read those houses
differently from Glider PRO 1.0.4.

### 21.1 How to read this section

Every entry has an ID, a one-line symptom, the citation, and — where it matters — an
**empirically measured** answer to "does any shipped house actually trigger this?".

| ID prefix | Class | Subsection |
|-----------|-------|------------|
| `B-n` | Out-of-range array indexing (memory safety) | §21.2 |
| `S-n` | Silent no-op / missing `else` / missing `default:` | §21.3 |
| `T-n` | Wrong union member written — **type-notional only** | §21.4 |
| `V-n` | Validation-pass ordering and error accounting | §21.5 |
| `U-n` | UI, menu, and event handling | §21.6 |
| `R-n` | Resource-level (`MENU`, `CNTL`, `CDEF`, `WDEF`) | §21.7 |
| `D-n` | Dead code, dead constants, dead prototypes | §21.8 |

Three severities are used:

* **Corrupting** — can write outside an allocated block, or writes wrong data to the house
  file. A Go port must not reproduce it.
* **Behavioural** — no memory damage, but the observable behaviour differs from what the
  code obviously intended. A Go port has to choose: reproduce (bug-for-bug fidelity when
  loading old houses) or fix (better authoring experience).
* **Cosmetic** — dead code, unreachable constants, leaks that the 1994 Memory Manager
  tolerated. Delete in the port.

A note on why so many of these never crashed in 1994: the house is one relocatable
`Handle` (`GliderPRO/Sources/House.c:25`, `houseHand thisHouse`) holding an 866-byte header
followed by `nRooms` × 348-byte rooms (§3.4). Reads a few kilobytes past a room record
usually land inside the *same* handle, in another room, and the Memory Manager did not
trap. §21.2.9 gives the exact arithmetic that makes that happen.

---

### 21.2 Out-of-range array indexing

#### 21.2.1 B-1 — `GenerateRetroLinks` writes `retroLinkList[who]` with `who` up to 255

`retroLinkList` is 24 entries of `retroLink { short room; short object; }` — 96 bytes:

```
short		srcLocations[kMaxRoomObs];                      // House.c:28
short		destLocations[kMaxRoomObs];                     // House.c:29
short		wasFloor, wasSuite;                             // House.c:30
retroLink	retroLinkList[kMaxRoomObs];                     // House.c:31   96 bytes
Boolean		houseUnlocked;                                  // House.c:32
```

The loop body (`GliderPRO/Sources/House.c:576-581` for switches, `:599-604` for transports):

```c
objectLinked = (short)thisObject.data.e.who;        // :576   0..255, unsigned Byte
if (retroLinkList[objectLinked].room == -1)         // :577   OOB READ
{
    retroLinkList[objectLinked].room = r;           // :579   OOB WRITE
    retroLinkList[objectLinked].object = i;         // :580   OOB WRITE
}
```

`who` is a `Byte` (`GliderStructs.h` variant `d`/`e`, payload offset 8), cast to `short`, so
its range is **0..255**. The only guard in the whole function is `where != -1` (`:570`,
`:593`) and `roomLinked == thisRoomNumber` (`:574`, `:597`). Index 255 writes to byte offset
`255 × 4 = 1020`, i.e. **924 bytes past the end of a 96-byte array**. In declaration order
the next global is `houseUnlocked` (`House.c:32`) — but declaration order is not a layout
guarantee, so the real target depends on how the MPW/CodeWarrior linker laid out `House.c`'s
globals. Treat the corrupted region as "unknown editor state".

`kMaxRoomObs` is **24** (`GliderPRO/Headers/GliderDefines.h:250`); `255 = kNoObjectSelected`
in the `Byte` domain is spelled **255** everywhere, never as a named constant.

**Reachability.** `GenerateRetroLinks()` is called from `ReflectCurrentRoom`
(`GliderPRO/Sources/Room.c:332`) — i.e. **every single time the editor navigates to a
room** — and from `BringSendFrontBack` (`GliderPRO/Sources/Objects.c:996`). Because the
predicate is `roomLinked == thisRoomNumber`, the OOB write happens when you navigate *into*
a room that is the destination of a link whose `who` is > 23.

**Measured.** Scanning all 22 shipped houses for link-source objects (transports
`0x31`–`0x40`, switches `0x41`–`0x48`, excluding `kSoundTrigger 0x49` whose `where` is a
sound ID, not a room — §19.2.4) gives 2474 live links. Of those, 189 name a destination
`(floor, suite)` with no room, and 180 carry `who > 23`. The intersection — destination room
exists *and* `who > 23`, i.e. exactly the set that reaches the OOB write — is **13 objects,
all in CD Demo House, and every one of them self-referential** (destination room == the room
the object lives in):

| House | Room # | Room name | Obj | `what` | `who` | `where` | → floor / suite | destRoom |
|-------|--------|-----------|-----|--------|-------|---------|-----------------|----------|
| CD Demo House | 33 | `The Other Side` | 8 | 0x3F `kInvisTrans` | 255 | 1410 | 2 / 14 | 33 |
| CD Demo House | 72 | `Let's Roll` | 22 | 0x34 `kMailboxRt` | **35** | 2109 | 1 / 21 | 72 |
| CD Demo House | 90 | `Welcome to the Museum` | 3 | 0x36 `kCeilingTrans` | 255 | 3210 | 2 / 32 | 90 |
| CD Demo House | 123 | `On the Poop Deck!` | 7 | 0x3F `kInvisTrans` | 255 | 4509 | 1 / 45 | 123 |
| CD Demo House | 139 | `N` | 2 | 0x41 `kLightSwitch` | 255 | 4809 | 1 / 48 | 139 |
| CD Demo House | 142 | `Welcome to Teddy World!` | 1 | 0x33 `kMailboxLf` | 255 | 5109 | 1 / 51 | 142 |
| CD Demo House | 143 | `Every Teddy's a Transport!` | 19 | 0x36 `kCeilingTrans` | 255 | 5209 | 1 / 52 | 143 |
| CD Demo House | 148 | `Santa World!` | 3 | 0x3F `kInvisTrans` | 255 | 5010 | 2 / 50 | 148 |
| CD Demo House | 151 | `Sewerland USA` | 2 | 0x3F `kInvisTrans` | 255 | 5107 | −1 / 51 | 151 |
| CD Demo House | 192 | `(Emergency Stairway)` | 4 | 0x36 `kCeilingTrans` | 255 | 720 | 12 / 7 | 192 |
| CD Demo House | 197 | `(Apartment 4A)` | 5 | 0x36 `kCeilingTrans` | 255 | 1121 | 13 / 11 | 197 |
| CD Demo House | 200 | `Apartment 3A` | 5 | 0x36 `kCeilingTrans` | 255 | 1120 | 12 / 11 | 200 |
| CD Demo House | 202 | `What's Zig without Zag?` | 13 | 0x36 `kCeilingTrans` | 255 | 920 | 12 / 9 | 202 |

All thirteen `what` values are members of `GenerateRetroLinks`' own case lists
(`House.c:561-568` switches: `kLightSwitch`…`kLgTrigger`; `House.c:586-591` transports:
`kMailboxLf`, `kMailboxRt`, `kFloorTrans`, `kCeilingTrans`, `kInvisTrans`, `kDeluxeTrans`),
so all thirteen are genuinely reachable. Twelve of them write `retroLinkList[255]`
(offset +1020); one writes `retroLinkList[35]` (offset +140, 44 bytes past the array).

Note that `GenerateRetroLinks` does **not** consider `kUpStairs` (0x31), `kDownStairs`
(0x32), or any of the eight door/window transports (0x37–0x3E) to be link sources, even
though `CountHouseLinks`/`GenerateLinksList` do (§17.2, §17.11). That inconsistency is
harmless here — it only means the "Linked From?" button is unavailable for those classes.

**Go port:** bound-check. `if who >= kMaxRoomObs { continue }`.

#### 21.2.2 B-2 — `GetObjectState(room, object)` validates neither argument

```c
Boolean GetObjectState (short room, short object)                     // Objects.c:703
{
    ...
    HLock((Handle)thisHouse);
    switch ((*thisHouse)->rooms[room].objects[object].what)           // Objects.c:712
```

There is no check on `room` (which can be −1) and none on `object` (0..255). The function
runs to `:871` and returns a default of `true` (`:708`) for any `what` it does not
recognise, so a garbage read silently yields "switch is on".

The five call sites are all in the **play-mode** drawing pass, one per drawn switch class
(`GliderPRO/Sources/ObjectDrawAll.c:516, 529, 542, 555, 568`). Each does:

```c
ExtractFloorSuite(thisObject.data.e.where, &floor, &suite);    // e.g. :513
room = GetRoomNumber(floor, suite);                            //      :514   can be -1
obj  = (short)thisObject.data.e.who;                           //      :515   0..255
DrawLightSwitch(&itsRect, GetObjectState(room, obj));          //      :516
```

`where == -1` ("no link") is **not** filtered before `ExtractFloorSuite`. For
`version >= 0x0200` (`GliderPRO/Sources/Link.c:41-53`) `ExtractFloorSuite(-1)` gives
`suite = -1/100 = 0` (C truncation toward zero) and `floor = (-1 % 100) - 8 = -9`, and
`GetRoomNumber(-9, 0)` returns −1 — so *every unlinked drawn switch* calls
`GetObjectState(-1, 255)`.

**Measured.** 608 objects of the five drawn-switch classes exist in the shipped houses. Of
those, **5** call `GetObjectState` with `room == -1` or `object > 23`:

| House | Room # | Room name | Obj | `what` | `who` | `where` | resolved room | byte offset read | in handle? | observed `what` |
|-------|--------|-----------|-----|--------|-------|---------|---------------|------------------|-----------|-----------------|
| CD Demo House | 139 | `N` | 2 | 0x41 `kLightSwitch` | 255 | 4809 | 139 | 52358 | yes (size 72554) | `0xFFFF` = `kObjectIsEmpty` |
| Land of Illusion | 69 | `Stickybear's Math Town` | 19 | 0x43 `kThermostat` | 255 | −100 | −1 | 3638 | yes (size 106310) | `0x0025` = 37 = `kPaper` |
| Land of Illusion | 74 | `Window Illusion` | 1 | 0x45 `kKnifeSwitch` | 255 | −100 | −1 | 3638 | yes | `0x0025` = `kPaper` |
| Leviathan | 29 | `The Atrium Returns!` | 7 | 0x41 `kLightSwitch` | **6** | 953 | −1 | 650 | yes (size 165122) | `0x2D2D` = 11565 (`"--"`) |
| Slumberland | 268 | `Down But Not Out?` | 15 | 0x41 `kLightSwitch` | 255 | −100 | −1 | 3638 | yes (size 134150) | `0xFFFF` |

Byte offsets are computed as `866 + 348·room + 60 + 12·object` (§3.3, §3.4) and read directly
out of the shipped data forks with python3. All five land *inside* the handle:

* `room == -1` reads 348 bytes before `rooms[0]`, i.e. into the header at offsets 518..865 —
  `highScores` (528..819), `savedGame` (820..859), `hasGame` (860), `firstRoom` (862),
  `nRooms` (864). Offset 650 is inside `highScores`, which is why Leviathan's case observes
  `0x2D2D` — two ASCII hyphens from a high-score name string.
* Offset 3638 (`room = -1`, `object = 255`) is inside `rooms[7]`'s object array, which is why
  Land of Illusion observes `kPaper` (a prize → `GetObjectState` returns `data.c.state`,
  payload offset 8).

The worst case is not exercised by the shipped houses but is trivially reachable: a
`who == 255` link whose destination is the **last** room of the house reads
`866 + 348(n−1) + 60 + 3060 + 12` bytes, which for a house of size `866 + 348n` is exactly
**+2784 bytes past the end of the handle** — the same overrun for every house (measured
+2784 for all 21 houses whose data fork is exactly `866 + 348n`, and +2782 for Sampler,
whose fork carries 2 extra trailing bytes; see §19.1.1).

**Go port:** validate both indices; return a defined default (`true`, matching `:708`).

#### 21.2.3 B-3 — `GoToObjectInRoom` trusts `who`, then trusts it again as `objActive`

```
void GoToObjectInRoom (short object, short floor, short suite)   // ObjectEdit.c:2769
```

Body highlights (`GliderPRO/Sources/ObjectEdit.c:2785, 2787, 2790, 2794`):

| Line | Statement | Risk |
|------|-----------|------|
| `:2785` | `if (thisRoom->objects[object].what == kObjectIsEmpty)` | reads `thisRoom->objects[0..255]` |
| `:2787` | `objActive = object;` | `objActive` escapes as a global with a value > 23 |
| `:2790` | `theMarquee.bounds = roomObjectRects[objActive];` | reads past `roomObjectRects[24]` |
| `:2794` | `StartMarquee(&roomObjectRects[objActive]);` | ditto |

`roomObjectRects` is declared `Rect roomObjectRects[kMaxRoomObs];`
(`GliderPRO/Sources/ObjectEdit.c:28`) — 24 × 8 = **192 bytes**. Index 255 reads offset 2040.

`thisRoom` is **not** a pointer into the house handle: `StructuresInit2.c:192-193` allocates
it standalone,

```c
thisRoom = (roomPtr)NewPtr(sizeof(roomType));      // StructuresInit2.c:193
```

so `thisRoom->objects[35]` reads 348 + 60 + 420 − 348 … i.e. plain heap past a 348-byte
`NewPtr` block, **not** a neighbouring room. The §21.2.9 aliasing argument does *not* apply
here. (This is worth stating explicitly because it is the natural wrong inference.)

**The guard is at the call site, not in the function.** All three dialogs that offer a
"Go To" button disable it when `who == 255`:

| Dialog | Guard | Disabled item |
|--------|-------|---------------|
| Switch info (DITL 1011) | `ObjectInfo.c:1306-1307` | `kGotoButton2` = **14** (`ObjectInfo.c:62`) |
| Trigger info (DITL 1034) | `ObjectInfo.c:1432-1433` | `kGotoButton2` = 14 |
| Transport info (DITL 1022) | `ObjectInfo.c:2107-2108` | `kGotoButton1` = **11** (`ObjectInfo.c:61`) |

Each guard is exactly `if (… .data.{d,e}.who == 255) MyDisableControl(infoDial, …)`. So `who`
values in **24..254** pass the guard and reach `GoToObjectInRoom` unchecked
(`ObjectInfo.c:1380`, `:1536`, and the transport epilogue).

**Measured.** Across all 22 shipped houses exactly **one** object anywhere has `who` in the
24..254 window: CD Demo House room 72 `Let's Roll`, object 22, `what = 0x34 kMailboxRt`,
payload `00 42 00 6f 00 00 08 3d 23 00` → `topLeft = (v 66, h 111)`, `where = 0x083D = 2109`
(suite 21 / floor 1 = its own room), **`who = 0x23 = 35`**. Opening that object's Transport
Info dialog and clicking "Go To" is the single reproducible path to B-3 in the shipped data.

**Go port:** clamp `object` to `0..kMaxRoomObs-1` inside the navigation function and treat
out-of-range as "no target".

#### 21.2.4 B-4 — `BringSendFrontBack` indexes `sorted[destObj]` with `destObj` up to 255

```c
short	sorting[kMaxRoomObs];      // Objects.c:882
short	sorted[kMaxRoomObs];       // Objects.c:883
```

Both are 24 `short` = 48 bytes, on the stack. The link-fixup pass writes

```c
(*thisHouse)->rooms[…].objects[…].data.e.who = (Byte)sorted[destObj];    // Objects.c:972
…
(*thisHouse)->rooms[…].objects[…].data.d.who = (Byte)sorted[destObj];    // Objects.c:977
```

where `destObj` comes from `linksList[i].destObj`, built by `GenerateLinksList` from the raw
`who` byte (`GliderPRO/Sources/House.c:356`, `:376`), hence 0..255. `sorted[255]` reads 510
bytes past a 48-byte stack array — a **stack** overread, on the 68k/PPC stack, which is much
less forgiving than the house handle. Note the read result is then *written into the house*,
so this is corrupting in both directions.

Reachability: `BringSendFrontBack(true/false)` is *House ▸ Bring To Front* / *Send To Back*
(`GliderPRO/Sources/Menu.c:554-561`), on the current room only. Any of the 180 measured
`who > 23` links whose destination is the current room triggers it.

**Go port:** bound-check `destObj`; treat > 23 as "unlinked".

#### 21.2.5 B-5 — `LopOffExtraRooms` reads `rooms[-1]` when `nRooms == 0`

```
void LopOffExtraRooms (void)          // HouseLegal.c:740-779
```

The loop that finds the last non-tombstone room (`HouseLegal.c:751-760`) starts at
`i = (*thisHouse)->nRooms - 1` and reads `(*thisHouse)->rooms[i].suite`. With `nRooms == 0`
that is `rooms[-1].suite` — house-header byte offset `866 − 348 + 54 = 572`, inside
`highScores`. If that garbage `short` happens not to equal `kRoomIsEmpty (-1)`, the function
concludes the house has one live room and calls `SetHandleSize` for `866 + 348` bytes;
if it does equal −1 the loop walks *further* negative.

Reachability in the editor: `InitializeEmptyHouse` sets `nRooms` to 0
(`GliderPRO/Sources/House.c:130` region; `numberRooms = 0` at `:147`), and
`CheckHouseForProblems` calls `LopOffExtraRooms` at `HouseLegal.c:1103`. In practice
`CreateNewHouse` immediately creates the first room, so a 0-room house only exists
transiently — but a hand-edited or truncated file with `nRooms == 0` reaches it. `Sampler`
ships with `nRooms = 2`, the minimum observed.

**Go port:** `if nRooms == 0 { return }` first.

#### 21.2.6 B-6 — `CheckDuplicateFloorSuite`: `DebugStr` and then carry on regardless

```c
#define kRoomsTimesSuites   8192                          // HouseLegal.c:648
…
bitPlace = ((floor + kNumUndergroundFloors - 1) * 128) + suite;   // HouseLegal.c:665-666
if ((bitPlace < 0) || (bitPlace >= kRoomsTimesSuites))
    DebugStr("\pBlew array");                             // HouseLegal.c:667-668
if (pidgeonHoles[bitPlace])                               // HouseLegal.c:669 — runs anyway
    …
    houseErrors++;                                        // HouseLegal.c:671
pidgeonHoles[bitPlace] = true;
```

There is **no `continue`** after the `DebugStr`, so an out-of-range `bitPlace` falls straight
through to the read and write of `pidgeonHoles[bitPlace]`. `pidgeonHoles` is 8192 `Boolean`
= 8192 bytes. `DebugStr` on a non-debugger machine (no MacsBug) either drops into the
Debugger trap or, on a shipping system, is a no-op — so on a user's machine the *only* effect
of the check is nothing.

Range arithmetic: legal floor is −7..56 and legal suite 0..127 (`HouseLegal.c:804-805`,
`:814-815`), and with `kNumUndergroundFloors = 8` (`GliderDefines.h:535`),
`bitPlace = (floor + 7) × 128 + suite` spans exactly 0..8191. Anything outside the legal box
overflows.

**Ordering makes it reachable.** In `CheckHouseForProblems` the step order is
`CheckDuplicateFloorSuite` at `HouseLegal.c:1089`, `ValidateRoomNumbers` at `:1110` — the
*duplicate* check runs **before** the *range* check, so an illegal floor/suite is guaranteed
to hit the pigeon-hole array first. See V-1.

**Measured.** All 4070 live rooms in all 22 shipped houses are inside the legal box
(floor −7..39, suite 0..127), giving observed `bitPlace` 0..6015. No shipped house triggers
B-6; only a corrupted or hand-authored file does.

**Go port:** validate ranges before indexing, or use a map keyed on `(floor, suite)`.

#### 21.2.7 B-7 — the unbounded room-lookup helpers

Three helpers index `rooms[]` without a bound check, and two of them lack the negative-suite
guard that their sibling has:

| Function | Cite | Missing guard |
|----------|------|---------------|
| `GetRoomFloorSuite(short room, short *floor, short *suite)` | `GliderPRO/Sources/Room.c:711-733`; the read is `:718` | no check that `0 <= room < nRooms` |
| `GetRoomNumber(short floor, short suite)` | `GliderPRO/Sources/Room.c:737-759` | no `suite < 0` early-out |
| `GetNeighborRoomNumber(short where)` | `GliderPRO/Sources/Room.c:562-635` | no `default:` in the direction `switch`, so `hDelta`/`vDelta` stay uninitialised for an unknown `where`; also no `suite < 0` guard |

Compare `RoomExists(short suite, short floor, short *roomNum)`
(`GliderPRO/Sources/Room.c:390-421`), which *does* guard:

```c
if (suite < 0)
    return (foundIt);              // Room.c:400-401
```

Two traps follow from this:

1. **Argument order.** `RoomExists(suite, floor, …)` takes **suite first**;
   `GetRoomNumber(floor, suite)` and `GetRoomFloorSuite(room, *floor, *suite)` take
   **floor first**. Any port that unifies these must not silently swap them.
2. **Tombstones.** A deleted room is marked by `suite = kRoomIsEmpty = -1`
   (`GliderDefines.h:525`) with `floor` left alone (`GliderPRO/Sources/Room.c:462-463` writes
   both). Only `RoomExists` filters it.

`GetNeighborRoomNumber`'s uninitialised deltas are only unreachable because all six call
sites pass one of the nine named constants `kCentralRoom`…`kNorthWestRoom`
(`GliderDefines.h:217-225`) or `kRoomAbove`…`kRoomToLeft` (`:200-203`).

#### 21.2.8 B-8 — `ReadScoresFromDisk` reads the whole file into a fixed struct

```
OSErr ReadScoresFromDisk (…)             // HighScores.c:801-855
```

`GetEOF` is used to size the read (`GliderPRO/Sources/HighScores.c:823`) and then the
`FSRead` at `:841` copies that many bytes into a `scoresType` — **292 bytes**
(`GliderPRO/Headers/GliderStructs.h:107-114`; verified in §3.4's offset table:
`highScores` occupies house-header bytes 528..819). A side-car scores file longer than
292 bytes overruns. This is the read-only-house high-score path (§19.16), not the editor
proper, but it is in the same save/load module family and a port will touch it.

#### 21.2.9 Why B-1…B-6 rarely crashed: the room-record aliasing identity

`sizeof(roomType)` is **348** and `objects[]` starts at room-relative offset **60**
(§3.3). Both are exact multiples of `sizeof(objectType) = 12`:

```
348 = 29 × 12
 60 =  5 × 12
```

Therefore, for an access through the house handle,

```
addr(rooms[i].objects[n])  =  866 + 348·i + 60 + 12·n
                           =  866 + 348·(i + k) + 60 + 12·(n − 29k)
```

for any integer `k`. In words: **`rooms[i].objects[n]` for `n ≥ 24` is bit-for-bit
`rooms[i+k].objects[n − 29k]`**, choosing `k = (n + 5) / 29`:

| `n` | `k` | Aliases |
|-----|-----|---------|
| 24 | 1 | `rooms[i+1]` header bytes 0..11 (`name[0..11]`) |
| 28 | 1 | `rooms[i+1]` header bytes 48..59 (`floor`, `suite`, `openings`, `numObjects`, +4) |
| 29 | 1 | `rooms[i+1].objects[0]` |
| 35 | 1 | `rooms[i+1].objects[6]` |
| 255 | 8 | `rooms[i+8].objects[23]` |

**Verified byte-for-byte** against CD Demo House's data fork: `rooms[72].objects[35]` and
`rooms[73].objects[6]` are both at file offset **26402** and read
`00 6e 01 11 00 00 27 24 00 00 01 01` — i.e. `what = 0x6E kCustomPict`
(`GliderDefines.h:409`), `data.g.topLeft = (v 273, h 0)`,
`data.g.height = 0x2724 = 10020` (the user PICT ID; §19.2.2), `initial = 1`, `state = 1`.
Room 72 is `Let's Roll` (floor 1, suite 21) and room 73 is `Flatland` (floor 1, suite 22).

This identity is the *entire* reason B-1's twelve `who == 255` cases and B-2's five cases
never produced a visible crash on a real Mac: the reads and writes stayed inside the same
relocatable block. It is also why a Go port that stores rooms as a `[]Room` of Go structs
will behave **differently** — a slice index of 255 panics. That divergence is desirable, but
it means the port cannot claim byte-identical behaviour on the shipped CD Demo House.

---

### 21.3 Silent no-ops and missing branches

#### 21.3.1 S-1 — `DoLink` does nothing, silently, if the source room can't be resolved

The entire body of `DoLink` is wrapped in one condition:

```c
void DoLink (void)                                                    // Link.c:270
{
    …
    if (GetRoomFloorSuite(linkRoom, &floor, &suite))                  // Link.c:275
    {
        …  // the whole link commit
    }
}
```

There is no `else`, no `SysBeep`, no alert. If `linkRoom` does not resolve, the user's Link
click appears to succeed (the windoid's *Link* button tracks, the dialog closes) and nothing
is written. Contrast `DoUnlink` (`GliderPRO/Sources/Link.c:325-361`), which has **no guard at
all** and unconditionally writes `where = -1` / `who = 255`.

Compounding this: `GetRoomFloorSuite` is B-7 — it doesn't bound-check `room` — so with
`linkRoom` out of range it returns `true` on garbage rather than `false`.

#### 21.3.2 S-2 — `UpdateLinkControl`'s else-less sub-case and missing `default:`

`UpdateLinkControl` (`GliderPRO/Sources/Link.c:57-194`) decides whether the *Link* button is
active by switching on `linkType` and then on the candidate destination's `what`. Two
structural gaps:

* The outer `switch (linkType)` has **no `default:`**. `linkType` values are
  `kSwitchLinkOnly` = 3, `kTriggerLinkOnly` = 4, `kTransportLinkOnly` = 5
  (`GliderDefines.h:463-465`); any other value leaves the control in whatever state it had.
* One inner sub-case (`GliderPRO/Sources/Link.c:149-157`) has no `else`, so when its
  predicate fails the button is left in its previous hilite state rather than being
  explicitly deactivated.

Additionally, `unlinkControl` (`CNTL` 131, `GetNewControl(kUnlinkControlID, linkWindow)` at
`GliderPRO/Sources/Link.c:242`) is **never** passed to `HiliteControl` anywhere in the
program. It is always active, even when there is nothing to unlink.

#### 21.3.3 S-3 — `MoveRoom`'s occupied-cell branch is empty

```
void MoveRoom (Point wherePt)          // Map.c:769-795
```

The branch that should handle "you dropped a room on a cell that already has a room"
(`GliderPRO/Sources/Map.c:784-786`) is `{ }`. The function is dead in 1.0.4 anyway (§20.3) —
nothing calls it — but the empty block is the reason it was abandoned: moving a room
rewrites its `(floor, suite)`, which silently invalidates every link that addresses it as
`(suite × 100) + floor + kNumUndergroundFloors` (§17.1).

#### 21.3.4 S-4 — `ShiftWholeHouse`'s inner loop is empty and its argument is unused

```c
void ShiftWholeHouse (short howFar)        // House.c:824
{
    #pragma unused (howFar)                // House.c:826
    …
    for (…)
    {
                                           // House.c:847-849  — empty
    }
}
```

Dead (§19.17.3), never called, argument explicitly declared unused.

#### 21.3.5 S-5 — seven `if (KeepObjectLegal()) { }` with empty bodies

`KeepObjectLegal` returns a `Boolean` that 21 of its 22 call sites throw away. Fourteen do it
honestly, via a variable named `whoCares`:

`GliderPRO/Sources/ObjectEdit.c:165, 174, 186, 203, 213, 230, 246, 259, 274, 290, 302, 313,
322, 334` — all `whoCares = KeepObjectLegal();`, all inside `DragHandle`
(`ObjectEdit.c:143`…).

Seven do it as an `if` with an empty compound statement:

| Site | Enclosing function |
|------|--------------------|
| `GliderPRO/Sources/ObjectAdd.c:774-776` | `AddNewObject` tail (`what` just assigned, `numObjects++`) |
| `GliderPRO/Sources/ObjectInfo.c:1015-1017` | `DoBlowerObjectInfo`, left/right-fan handedness flip |
| `GliderPRO/Sources/ObjectInfo.c:1045-1047` | `DoBlowerObjectInfo`, the second handedness flip |
| `GliderPRO/Sources/ObjectInfo.c:1223-1225` | `DoCustPictObjectInfo`, after writing `data.g.height = wasPict` |
| `GliderPRO/Sources/ObjectEdit.c:747-749` | `DragObject` tail |
| `GliderPRO/Sources/ObjectEdit.c:1352-1354` | `DuplicateObject` (`:1191-1371`) |
| `GliderPRO/Sources/ObjectEdit.c:1700-1702` | `MoveObject` (`:1374-1772`, dead — §20.4) |

Only **one** site uses the result:

```c
if (!KeepObjectLegal())          // HouseLegal.c:939
    …
    houseErrors++;               // HouseLegal.c:944
```

inside `KeepAllObjectsLegal` (`HouseLegal.c:919-955`).

The empty bodies are not merely stylistic: because the return value is discarded at 21 sites,
the *editor* never tells the user that it silently moved their object, and `fileDirty` is set
by the surrounding code regardless. See V-6 for what `KeepObjectLegal` actually changes
without flagging it.

#### 21.3.6 S-6 — `SortHouseObjects` bails out before compacting when there are no links

```c
void SortHouseObjects (void)                    // House.c:448
{
    …
    numLinks = CountHouseLinks();
    if (numLinks == 0)
        return;                                 // House.c:459-460
    …
    // per-room compaction loop
    …
    ForceThisRoom(thisRoomNumber);              // House.c:502
}
```

The early return skips **the object compaction pass itself**, not just the link fixup. A
house with zero links therefore never has its `objects[]` arrays compacted (empty slots
pulled forward) and never gets the closing `ForceThisRoom` re-stage. `Empty House` (35 rooms)
is the obvious shipped example of a house with few or no links.

`MakeSureNumObjectsJives` (`HouseLegal.c:885-913`) still fixes `numObjects`, so the house
stays *consistent* — it just stays fragmented.

#### 21.3.7 S-7 — two `TrackControl` results discarded into empty blocks

```c
case kControlPageDownPart:
if (TrackControl(whichControl, wherePt, scrollHActionUPP))     // Map.c:669
{
                                                               // :670-672 empty
}
break;
```

and the vertical twin at `GliderPRO/Sources/Map.c:692-695`. Harmless (the live-scroll action
proc does the work) but it means the arrow/page parts never trigger the final
`RedrawMapContents()` that the thumb case does (`Map.c:676-680`, `:698-702`), so the map can
be left one action-proc tick stale.

#### 21.3.8 S-8 — `SetToNearestNeighborRoom`'s spiral has no bound

`SetToNearestNeighborRoom(short wasFloor, short wasSuite)`
(`GliderPRO/Sources/Room.c:639-707`) walks outward from `(wasFloor, wasSuite)` looking for
any live room. `finished` is set to `true` in exactly one place — `GliderPRO/Sources/Room.c:667`
— when a room is found. The spiral itself (`:659-704`) has no radius cap and no
"searched the whole legal box" exit. Called from `DeleteRoom` (`Room.c:470`), which has
already handled the "no rooms left" case separately (`:472-478`, which sets
`(*thisHouse)->firstRoom = -1`), so the loop is only entered when at least one room exists —
but any bug that lets it be entered with zero live rooms hangs the app.

---

### 21.4 Wrong-union-member writes — type-notional only

These three sites write through the *wrong* `objectType.data` union arm for the object class
in hand. **All three are harmless**, and it is important that a Go port understands *why*,
because a port with real (non-overlapping) types will break where the C did not.

The reason is the union layout (§3.2). Variants `d` (`transportType`) and `e` (`switchType`)
place their link fields at **identical payload offsets**:

| Variant | offset 0..3 | 4..5 | **6..7** | **8** | 9 |
|---------|-------------|------|----------|-------|---|
| `d transportType` | `Point topLeft` | `short tall` | **`short where`** | **`Byte who`** | `Byte wide` |
| `e switchType` | `Point topLeft` | `short delay` | **`short where`** | **`Byte who`** | `Byte type` |

So `data.d.where` and `data.e.where` are the same two bytes, and `data.d.who` and
`data.e.who` are the same byte.

| ID | Site | What it writes | Why it's fine |
|----|------|----------------|---------------|
| T-1 | `GliderPRO/Sources/House.c:421` in `SortRoomsObjects` | `…objects[i].data.e.who = …` for **every** link source, including transports | `d.who` ≡ `e.who` at payload offset 8 |
| T-2 | `GliderPRO/Sources/Objects.c:961-978` in `BringSendFrontBack` | the `switch`/`default:` arms are swapped — the switch-class arm runs `data.d.who` and the default arm runs `data.e.who` | same identity, both directions |
| T-3 | `GliderPRO/Sources/Scrap.c:147`, `:152` | swapped `d`/`e` in the (entirely commented-out) clipboard code | dead anyway (§20.1) |

A **fourth**, genuinely different case is the unreachable `kFloorTrans` snap in
`KeepObjectLegal` (`GliderPRO/Sources/HouseLegal.c:132-137`), which writes
`theObject->data.a.distance` — payload 4:6 — where a transport's field at 4:6 is
`data.d.tall`. That *would* be a real cross-type write, but the case label is absent from the
governing case list at `HouseLegal.c:75-90`, so the code is unreachable. See V-6.

**Go port:** implement `where`/`who` as fields on a single link-carrying struct (or an
interface method), not as separate per-class fields, and the whole class of confusion goes
away. Do **not** "fix" T-1/T-2 by writing to a different field — they are correct as
compiled.

---

### 21.5 Validation-pass ordering and error accounting

`CheckHouseForProblems` (`GliderPRO/Sources/HouseLegal.c:1052-1218`) runs eleven steps, in
this order (§18.4):

| # | Step | Call site | `houseErrors = 0` immediately before? |
|---|------|-----------|----------------------------------------|
| 0 | (function entry) | `houseErrors = 0` at `:1058` | — |
| 1 | `WrapBannerAndTrailer` | `:1068` | n/a — counts nothing |
| 2 | `ValidateNumberOfRooms` | `:1075` | no, but inherits the `:1058` entry reset; resets to 0 *after* reporting at `:1081` |
| 3 | `CheckDuplicateFloorSuite` | `:1089` | yes — `:1088` |
| 4 | `CompressHouse` | `:1103` | n/a |
| 5 | `LopOffExtraRooms` | `:1105` | n/a |
| 6 | `ValidateRoomNumbers` | `:1110` | **NO** — and step 3 does not reset after reporting |
| 7 | `CountUntitledRooms` | `:1127` | yes — `:1126` |
| 8 | `CheckRoomNameLength` | `:1144` | yes — `:1143` |
| 9 | `MakeSureNumObjectsJives` | `:1161` | yes — `:1160` |
| 10 | `KeepAllObjectsLegal` | `:1180` | yes — `:1177` |
| 11 | `CheckForStaircasePairs` | `:1197` | yes — `:1196`, but the function never increments it |
| 12 | `CountStarsInHouse` | `:1203` | n/a — tested by return value, not `houseErrors` |

Epilogue: `InitCursor()` `:1213`, `CloseMessageWindow()` `:1214`,
`ForceThisRoom(wasRoom)` `:1215`, `objActive = wasActive` `:1216`. Steps 2–3 and 6–12 are
each individually gated on `isHouseChecks`; steps 4 and 5 are **not** — `CompressHouse` and
`LopOffExtraRooms` run unconditionally.

#### 21.5.1 V-1 — the range check runs *after* the duplicate check

`CheckDuplicateFloorSuite` (step 3, `:1089`) indexes an 8192-entry array with
`bitPlace = (floor + 7) × 128 + suite`; `ValidateRoomNumbers` (step 6, `:1110`) is what
enforces floor ∈ −7..56 (`HouseLegal.c:804-805`) and suite ∈ 0..127 (`:814-815`). So on the
first validation pass over a house with an illegal `(floor, suite)`, the pigeon-hole array is
indexed out of range *before* anything clamps the values. Combined with B-6's missing
`continue`, that is an unguarded OOB read+write on the very first pass.

#### 21.5.2 V-2 — `ValidateRoomNumbers` never resets `houseErrors`

Steps 3, 7, 8, 9, 10, and 11 are each preceded by `houseErrors = 0;` (`:1088`, `:1126`,
`:1143`, `:1160`, `:1177`, `:1196`). Step 6 is not —
`GliderPRO/Sources/HouseLegal.c:1107-1121` contains no reset. Its own `houseErrors++` sites
are `:811` (illegal floor) and `:821` (illegal suite).

The predecessor that leaks into it is step 3: `CheckDuplicateFloorSuite`'s reporting block
(`:1090-1099`) prints the count but **does not zero it afterwards**, unlike step 2, which
does (`:1081`). So the "*N* Room Errors" message shown for step 6 is the *sum* of the
duplicate-floor/suite count and the illegal-floor/suite count, and step 6's message fires
even when `ValidateRoomNumbers` itself found nothing.

#### 21.5.3 V-3 — `CheckForStaircasePairs` counts nothing and clobbers `thisRoomNumber`

```
void CheckForStaircasePairs (void)         // HouseLegal.c:961-1045
```

* It never touches `houseErrors`, so unpaired staircases are silently repaired (or silently
  left) with no report.
* It writes the global `thisRoomNumber` twice — `GliderPRO/Sources/HouseLegal.c:980` and
  `:1011` — in order to make `GetNeighborRoomNumber(kNorthRoom)` (`:981`) and
  `GetNeighborRoomNumber(kSouthRoom)` (`:1012`) look at the room it is inspecting, and never
  restores it. The only reason this is survivable is that its caller re-stages afterwards
  (`ForceThisRoom(wasRoom)`, `HouseLegal.c:1215`).

Any Go port that calls a staircase-pair check outside `CheckHouseForProblems` must save and
restore the "current room" itself.

#### 21.5.4 V-4 — `CheckRoomNameLength` mutates tombstones

```c
(*thisHouse)->rooms[i].unusedByte = 0;      // HouseLegal.c:868
```

is executed for **every** room in the loop, including rooms whose `suite == kRoomIsEmpty`.
Zeroing a byte inside a tombstone is harmless in 1.0.4 (nothing reads `unusedByte`), but it
means a validation pass **modifies the house even when it reports zero errors** — so the file
is not byte-stable across an open/validate/save cycle unless every room already had
`unusedByte == 0`. `houseErrors++` for an over-long name is at `:874`.

#### 21.5.5 V-5 — `ValidateNumberOfRooms` uses truncating division, hiding trailing garbage

```
void ValidateNumberOfRooms (void)          // HouseLegal.c:621-640
```

It derives the expected room count from `GetHandleSize` arithmetic
(`GliderPRO/Sources/HouseLegal.c:630-631`) — integer division of
`(handleSize − sizeof(houseType-prefix))` by `sizeof(roomType)`. Any remainder is
**discarded**, so a data fork with trailing bytes validates clean.

**Measured.** 21 of the 22 shipped houses satisfy `len(dataFork) == 866 + 348·nRooms`
exactly. **`Sampler` does not**: 1564 bytes with `nRooms = 2` gives
`866 + 696 = 1562`, leaving **2 stray trailing bytes**. `ValidateNumberOfRooms` reports no
error for it. (`houseErrors++` for a genuine mismatch is at `:636`.)

#### 21.5.6 V-6 — `KeepObjectLegal`'s return value is the inverse of its own comment, and eight mutations are unflagged

The header comment, verbatim from `GliderPRO/Sources/HouseLegal.c:38-40` (the `…` are
Mac-Roman `0xC9` ellipses; the `erros` typo is the author's):

```
// Does a test of the current object active for any illegal bounds…
// or values.  It corrects the erros and returns true if any changes…
// were made.
```

The implementation initialises `unchanged = true` (`GliderPRO/Sources/HouseLegal.c:50`) and
ends with

```c
return (unchanged);                               // HouseLegal.c:596
```

— i.e. it returns **true when nothing changed**, the exact inverse of the comment. The one
consumer, `KeepAllObjectsLegal`, uses `if (!KeepObjectLegal())` (`HouseLegal.c:939`), which is
correct for the *implementation*. So the comment is wrong and the code is right; a porter
reading only the comment will invert the sense.

The `kInitialGliderSelected` arm is a second inversion: it clamps the glider's start point
(`GliderPRO/Sources/HouseLegal.c:60, 62, 64, 66`) and then does `return (true)`
(`:68`) — reporting "unchanged" *after* changing something.

**Eight mutations never set `unchanged = false`:**

| Mutation | Cite | Notes |
|----------|------|-------|
| `kFloorVent` v-snap to the floor | `HouseLegal.c:115-119` | silent |
| `kFloorBlower` v-snap | `:120-125` | silent |
| `kSewerGrate` v-snap | `:126-131` | silent |
| `kFloorTrans` v-snap | `:132-137` | silent **and unreachable** — `kFloorTrans` is absent from the governing case list at `:75-90`; also writes `data.a.distance` (payload 4:6) where a transport keeps `data.d.tall` (T-3 discussion) |
| `kDoorInLf` ⇄ `kDoorInRt` auto-flip | `:310-324` | **mutates `theObject->what`** |
| `kDoorExRt` ⇄ `kDoorExLf` auto-flip | `:325-339` | mutates `what` |
| `kWindowInLf` ⇄ `kWindowInRt` auto-flip | `:340-354` | mutates `what` |
| `kWindowExRt` ⇄ `kWindowExLf` auto-flip | `:355-369` | mutates `what` |

The four handedness flips are the significant ones: opening a door on the wrong side of a
room silently changes the object's **class code**, and because `unchanged` stays `true`,
`KeepAllObjectsLegal` does not count it and the validator reports "no problems".

For the avoidance of doubt: the light-geometry recompute at `HouseLegal.c:429-430` **is**
flagged — `unchanged = false` at `:437`, with the second light block flagging at `:446`,
`:451`, and `:456`. It is not part of the silent set.

#### 21.5.7 V-7 — `CompressHouse` fixes two room references and misses a third

`CompressHouse` (`GliderPRO/Sources/HouseLegal.c:688-734`) removes tombstones and renumbers,
then repairs the stored room indices:

| Global / field | Repaired? | Cite |
|----------------|-----------|------|
| `(*thisHouse)->firstRoom` | yes | `HouseLegal.c:712-713` |
| `wasRoom` (the validator's saved current room) | yes | `:714-715` |
| `previousRoom` | **no** | — |

`previousRoom` is a real global (`GliderPRO/Sources/HouseIO.c:424` sets it to −1 on load) used
by the editor's room navigation. After a compaction it can point at a different room than it
did before, or past the end.

The concrete symptom: *Go To Room… ▸ Go to Previous Room* is
`roomToGoTo = previousRoom;` (`GliderPRO/Sources/House.c:711`), guarded only by
`RoomNumExists(roomToGoTo)` (`:730`). So after any `CheckHouseForProblems` pass that removed
tombstones, "Previous Room" either beeps (`SysBeep(1)`, `:737`) or silently navigates to
**a different room than the one you were last in**.

#### 21.5.8 V-8 — `WrapText` leaves over-long lines rather than breaking words

`WrapBannerAndTrailer` (`GliderPRO/Sources/HouseLegal.c:604-613`) calls

```c
WrapText(…banner…,  40);        // HouseLegal.c:611
WrapText(…trailer…, 64);        // HouseLegal.c:612
```

`WrapText` (`GliderPRO/Sources/StringUtils.c:216-250`) inserts carriage returns at spaces. A
single "word" longer than `maxChars` gets **no** return inserted; the line simply stays
long. It does **not** overflow the buffer. The banner and trailer are `Str255`
(house-header offsets 16..271 and 272..527, §3.4), so length is capped at 255 regardless.

---

### 21.6 UI, menu, and event bugs

#### 21.6.1 U-1 — `ToggleToolsWindow` sets `isToolsOpen = true` in both branches

```c
void ToggleToolsWindow (void)          // Tools.c:351-365
{
    if (isToolsOpen)
    {
        CloseToolsWindow();
        isToolsOpen = true;            // Tools.c:357   <-- should be false
    }
    else
    {
        OpenToolsWindow();
        isToolsOpen = true;            // Tools.c:362
    }
}
```

Once the Tools windoid has been opened, `isToolsOpen` can never return to `false` via the
menu item. The *Options ▸ Tools Window* checkmark and the reopen path both consult it.

#### 21.6.2 U-2 — `HandleMapClick` leaks two `ControlActionUPP`s on five paths

`HandleMapClick` (`GliderPRO/Sources/Map.c:570-712`) allocates
`scrollHActionUPP` and `scrollVActionUPP` at the top and disposes them at
`GliderPRO/Sources/Map.c:709-710`. Five `return` statements sit **before** that disposal:

`GliderPRO/Sources/Map.c:587`, `:602`, `:632`, `:641`, `:648`.

Each early return leaks two UPPs (68k: two 6-byte routine descriptors; PPC: two
`RoutineDescriptor`s in the system heap). Map clicks are frequent, so this leaks steadily
during a long editing session.

#### 21.6.3 U-3 — the *Go To Room…* dialog makes **Cancel** the default button

`DITL` 1043, verified from `Glider PRO.r` (10 items):

| # | Type | Rect (t,l,b,r) | Text / data |
|---|------|----------------|-------------|
| 1 | Button | (133, 214, 153, 272) | `Cancel` |
| 2 | Button | (40, 8, 60, 168) | `Go to First Room` |
| 3 | Button | (68, 8, 88, 168) | `Go to Previous Room` |
| 4 | Button | (96, 8, 116, 128) | `Go to Room…` |
| 5 | editText | (98, 174, 114, 194) | — (floor) |
| 6 | editText | (98, 241, 114, 269) | — (suite) |
| 7 | statText | (98, 132, 114, 171) | `floor:` |
| 8 | statText | (99, 199, 115, 238) | `suite:` |
| 9 | picture (typ 0xC0) | (0, 0, 32, 280) | resID `0x03FF` = 1023 |
| 10 | userItem | (124, 8, 125, 272) | — |

`UpdateGoToDialog` (`GliderPRO/Sources/House.c:618-623`) calls
`DrawDefaultButton(theDialog)` at `:621`, and `DrawDefaultButton`
(`GliderPRO/Sources/DialogUtils.c:352-363`) is hard-wired to draw the heavy default ring
around **item 1** — it takes no item number. `GoToFilter`
(`GliderPRO/Sources/House.c:628-660`) maps both Return and Enter to item 1:

```c
case kReturnKeyASCII:
case kEnterKeyASCII:
FlashDialogButton(dial, kOkayButton);      // House.c:637
*item = kOkayButton;                       // House.c:638
return(true);
```

with `#define kOkayButton 1` (`GliderPRO/Headers/Externs.h:22`). And `DoGoToDialog`
(`GliderPRO/Sources/House.c:666-739`) handles item 1 as the *cancel* action:

```c
if (item == kOkayButton)
{
    roomToGoTo = -1;                       // House.c:700
    canceled = true;                       // House.c:701
    leaving = true;
}
```

So the visually-emphasised, keyboard-default action in the room-navigation dialog is to
dismiss it — and the constant that names it is called `kOkayButton`. The three real actions
are items 2, 3, and 4 (`kGoToFirstButt`, `kGoToPrevButt`, `kGoToFSButt` —
`House.c:704`, `:709`, `:714`), none of which can be reached from the keyboard.

`UpdateGoToDialog` also frames item **10** — the 1-pixel-high userItem at (124, 8, 125, 272),
a horizontal divider — in `kRedOrangeColor8` (`House.c:622`).

#### 21.6.4 U-4 — Cut / Clear delete a room with no confirmation

`QueryDeleteRoom` (`GliderPRO/Sources/Room.c:490-500`) exists and puts up the confirmation
alert; `DeleteRoom(Boolean doAsk)` (`Room.c:439-485`) calls it only when `doAsk` is true.
The two menu paths pass **`false`**:

```c
case iCut:
if (houseUnlocked)
{
    if (objActive > kNoObjectSelected)
    {
//      PutObjectScrap();
        DeleteObject();
    }
    else
    {
//      PutRoomScrap();
        DeleteRoom(false);          // Menu.c:509
    }
    UpdateClipboardMenus();
}
break;
…
case iClear:
if (houseUnlocked)
{
    if (objActive > kNoObjectSelected)
        DeleteObject();
    else
        DeleteRoom(false);          // Menu.c:544
    UpdateClipboardMenus();
}
break;
```

Because `PutRoomScrap()` is commented out (`Menu.c:508`, `:521`) there is also **no
clipboard copy** — and no undo (§20.5). *Cut Room* with nothing selected is therefore an
unconfirmed, unrecoverable delete of the current room and all its objects.

The Delete-key path is the opposite: `Events.c:253-261` calls `DeleteRoom(true)` at
`GliderPRO/Sources/Events.c:257`, which *does* ask.

#### 21.6.5 U-5 — *Copy Room* / *Copy Object* is a fully-commented no-op that stays enabled

```c
case iCopy:
if (houseUnlocked)
{
//  if (objActive > kNoObjectSelected)
//      PutObjectScrap();
//  else
//      PutRoomScrap();
    UpdateClipboardMenus();         // Menu.c:522
}
break;
```

(`GliderPRO/Sources/Menu.c:515-524`.) The item is never disabled, and
`UpdateClipboardMenus` (`GliderPRO/Sources/Menu.c:165-235`) still *relabels* it
("Copy Room" ⇄ "Copy Object") based on selection. So the menu advertises a working Copy that
does nothing at all. `iPaste` is likewise a commented block (`Menu.c:526-536`, the body
`/* … */` at `:529-534`). See §20.1.2 for the full matrix.

#### 21.6.6 U-6 — `backgroundsMenu` is fetched, mutated, never attached, never released

In `DoRoomInfo`:

```c
#define kBackgroundsMenuID  140                                    // RoomInfo.c:379
MenuHandle backgroundsMenu;                                        // RoomInfo.c:381
…
backgroundsMenu = GetMenu(kBackgroundsMenuID);                      // RoomInfo.c:397
//  SetMenuItemTextStyle(backgroundsMenu, kOriginalArtworkItem, italic);   // :398
if (HouseHasOriginalPicts())
    EnableMenuItem(backgroundsMenu, kOriginalArtworkItem);          // RoomInfo.c:400
…
//  AddMenuToPopUp(roomInfoDialog, kRoomPopupItem, backgroundsMenu); // RoomInfo.c:434
```

There is no `ReleaseResource` / `DisposeMenu` for it anywhere in `RoomInfo.c` (the four
`ReleaseResource` calls at `:851`, `:854`, `:864` (commented), and `:889` are all for
`PICT` handles). `GetMenu` returns a **resource** handle, so the second and subsequent
invocations get the same already-loaded handle — meaning the `EnableMenuItem` mutation at
`:400` persists in the loaded resource for the rest of the session even if
`HouseHasOriginalPicts()` later becomes false.

And because `AddMenuToPopUp` is commented out at `:434`, the menu is never attached to
anything. See R-2 and §13.4 / §20.7 for the consequence.

#### 21.6.7 U-7 — `CreateNewRoom` can lose the staged room on allocation failure

`CreateNewRoom(short h, short v)` (`GliderPRO/Sources/Room.c:159-239`) fills `*thisRoom` with
defaults (`:169-182`), reuses a tombstone if one exists (`:188-194`), and otherwise grows the
house handle with

```c
PtrAndHand((Ptr)thisRoom, (Handle)thisHouse, sizeof(roomType));     // Room.c:200
```

On failure it takes the error path at `:201-207` — but `*thisRoom` has already been
overwritten with the blank new room, so the previously-staged room's edits are gone and the
house handle is unchanged. There is no restore.

A related quirk in the same function: holding **Shift** while creating suppresses part of the
initialisation (`GliderPRO/Sources/Room.c:232-236`) — an undocumented modifier.

#### 21.6.8 U-8 — `DeleteRoom` can write `firstRoom = -1`

`DeleteRoom(Boolean doAsk)` (`GliderPRO/Sources/Room.c:439-485`), verbatim on the relevant
lines:

```c
wasFloor = (*thisHouse)->rooms[thisRoomNumber].floor;             // Room.c:459
wasSuite = (*thisHouse)->rooms[thisRoomNumber].suite;             // Room.c:460
firstDeleted = ((*thisHouse)->firstRoom == thisRoomNumber);       // Room.c:461
thisRoom->suite = kRoomIsEmpty;                                   // Room.c:462
(*thisHouse)->rooms[thisRoomNumber].suite = kRoomIsEmpty;         // Room.c:463
HSetState((Handle)thisHouse, wasState);                           // Room.c:464

noRoomAtAll = (RealRoomNumberCount() == 0);                       // Room.c:466
if (noRoomAtAll)
    thisRoomNumber = kRoomIsEmpty;                                // Room.c:468   == -1
else
    SetToNearestNeighborRoom(wasFloor, wasSuite);                 // Room.c:470   (S-8)

if (firstDeleted)
{
    …
    (*thisHouse)->firstRoom = thisRoomNumber;                      // Room.c:476
    …
}
```

Points to note:

* The tombstone is written **twice** — once into the staged `*thisRoom` (`:462`) and once
  into the house handle (`:463`). Only `suite` is set to `kRoomIsEmpty` (−1); `floor` is left
  at its old value, which is why every tombstone test in the program is on `suite`.
* `RealRoomNumberCount()` is `GliderPRO/Sources/House.c:170-189`.
* `firstRoom` (a `short` at house-header offset **862..863**, §3.4 — the room the game starts
  in) is rewritten from `thisRoomNumber` at `:476` whenever the deleted room *was* the first
  room. If that was also the **last live room**, `thisRoomNumber` is −1 from `:468`, so
  `firstRoom` becomes **−1**, which is not a valid room number. `CompressHouse` will not
  repair it — `HouseLegal.c:712-713` only clamps `firstRoom` *down* when it is ≥ `nRooms`. A
  house saved in this state carries an invalid start room on disk.
* If rooms *do* remain, `SetToNearestNeighborRoom` has already put a valid room number in
  `thisRoomNumber`, so `firstRoom` correctly moves to a surviving room.

#### 21.6.9 U-9 — `ReflectCurrentRoom` hard-codes the map centring offsets

```c
CenterMapOnRoom(64, 1);            // Room.c:316
```

inside `ReflectCurrentRoom` (`GliderPRO/Sources/Room.c:308-339`). `CenterMapOnRoom(h, v)`
(`GliderPRO/Sources/Map.c:72`) expects map cell coordinates; the literals `64` and `1` are
the middle of the 128-suite axis and one below the top, not a function of the room that is
actually being reflected. The very next thing the function does is
`GenerateRetroLinks()` (`Room.c:332`) — the B-1 trigger.

#### 21.6.10 U-10 — `BUILD_ARCADE_VERSION` hijacks all four arrow keys in edit mode

The edit-mode key switch is `GliderPRO/Sources/Events.c:176`. It is split by
`#if BUILD_ARCADE_VERSION` at `:191`, `#else` at `:209`, `#endif` at `:251`. The
**compiled-out** block (`:211-249`) is the real editor key map — the four arrow keys doing
`SelectNeighborRoom` / `MoveObject` nudging (§20.4). The block that **is** compiled is the
arcade one:

| Lines | Key | Action in edit mode |
|-------|-----|---------------------|
| `:178-179` | Help | nothing (empty `case`) |
| `:181-184` | Page Up | `PrevToolMode()` if `houseUnlocked` — real editor behaviour |
| `:186-189` | Page Down | `NextToolMode()` if `houseUnlocked` — real editor behaviour |
| `:193-195` | Left arrow | `DoOptionsMenu(iHighScores)` — opens the High Scores dialog |
| `:197-199` | Right arrow | `DoOptionsMenu(iHelp)` — runs `DoDemoGame()` (§19.4.3) |
| `:201-203` | Up arrow | `DoGameMenu(iNewGame)` — **starts a new game** |
| `:205-207` | Down arrow | `DoGameMenu(iNewGame)` — **starts a new game** |
| `:253-261` | Delete | `DeleteRoom(true)` at `:257` / `DeleteObject()` at `:259`, gated on `houseUnlocked` |

None of the four arcade arms is gated on `houseUnlocked` or on `theMode`, and none of them
asks for confirmation. Pressing Up or Down while editing abandons the edit session for a
game. §1.5 documents the key map that §20.4's dead code was supposed to provide.

#### 21.6.11 U-11 — `DoesNeighborRoomExist` has its `#endif` inside the function body

`GliderPRO/Sources/Room.c:505-540`, with the `#endif` at `:539` — inside the braces rather
than after the closing `}`. It compiles, but the conditional-compilation intent is unclear
and a naive reformat breaks it.

#### 21.6.12 U-12 — `SortHouseList`'s duplicate removal compares two fields against themselves and skips an element

```c
if ((EqualString(theHousesSpecs[i].name, theHousesSpecs[h].name, true, true)) &&
        (theHousesSpecs[i].vRefNum == theHousesSpecs[i].vRefNum) &&      // SelectHouse.c:528
        (theHousesSpecs[i].parID == theHousesSpecs[i].parID))            // SelectHouse.c:529
{
    theHousesSpecs[h] = theHousesSpecs[housesFound - 1];                 // :531
    housesFound--;                                                       // :532
}
h++;                                                                     // :534
```

Two bugs in five lines (`GliderPRO/Sources/SelectHouse.c:516-553`):

1. `:528` and `:529` compare `[i]` against `[i]` — both are tautologies. The intent was
   `[h]`. So the dedupe treats two *different* houses with the same file name on different
   volumes or in different folders as duplicates and discards one.
2. `:531-534` overwrites slot `h` with the last element and then unconditionally does
   `h++`, so the element just moved into slot `h` is never examined.

---

### 21.7 Resource-level bugs and orphans

All numbers in this subsection were re-parsed out of
`GliderPRO/Glider PRO.r` for this section.

#### 21.7.1 R-1 — `MENU` 140 and `MENU` 141 both declare `menuID` 133

There are exactly **six** `MENU` resources in the application:

| resID | `menuID` | `enableFlags` | Title | Items |
|-------|----------|---------------|-------|-------|
| 128 | 128 | `0xFFFFFFFB` | `\x14` (Apple) | 2 |
| 129 | 129 | `0xFFFFFFAF` | `Game` | 7 |
| 130 | 130 | `0xFFFFFFFB` | `Options` | 5 |
| 131 | 131 | `0xFFFADF77` | `House` | 21 |
| **140** | **133** | `0xFFF7FFFF` | `Rooms` | **19** |
| **141** | **133** | `0xFFFFFFFF` | `Tools` | **9** |

Only 128–131 are installed in the menu bar. There are exactly **five** `GetMenu` call sites
in the whole tree: `GliderPRO/Sources/InterfaceInit.c:49, 55, 60, 68` (the four menu-bar
menus) and `GliderPRO/Sources/RoomInfo.c:397` (`GetMenu(140)`, U-6). Nothing calls
`GetMenu(141)`.

`MENU` 140's 19 items are the room backgrounds:
`Simple Room`, `Paneled Room`, `Basement`, `Child's Room`, `Asian Room`,
`Unfinished Room`, `Swinger's Room`, `Bathroom`, `Library`, `Garden`, `Skywalk`, `Dirt`,
`Meadow`, `Field`, `Roof`, `Sky`, `Stratosphere`, `Stars`, `Original Artwork`.

`MENU` 141's 9 items are the tool classes:
`Blowers`, `Furniture`, `Prizes`, `Transport`, `Switches`, `Lighting`,
`Appliances Etc.`, `Enemies`, `Clutter` — matching `kBlowerMode` = 1 …
`kClutterMode` = 9 (`GliderPRO/Headers/GliderDefines.h:272-280`).

Sharing `menuID` 133 means that if both were ever inserted into the menu list
simultaneously, `GetMenuHandle(133)` would be ambiguous. In 1.0.4 neither is inserted, so it
is latent. (The one menu the code *does* insert and remove dynamically is `houseMenu`:
`InsertMenu(houseMenu, 0)` at `GliderPRO/Sources/Menu.c:251` and
`DeleteMenu(kHouseMenuID)` at `:253`, with menu IDs `GliderDefines.h:186-189`.)

#### 21.7.2 R-2 — `CNTL` 128's `max` is 18 and its `refCon` is 0

All five `CNTL` resources, parsed from the `.r` dump:

| resID | bounds (t,l,b,r) | `value` | `vis` | `max` | `min` | `procID` | `refCon` | Title |
|-------|------------------|---------|-------|-------|-------|----------|----------|-------|
| **128** | (0, 0, 20, 190) | 1 | 1 | **18** | 1 | **2050** | **0** | `''` |
| **129** | (2, 4, 22, 112) | 1 | 1 | **3** | 1 | **2050** | **141** | `''` |
| 130 | (5, 70, 25, 124) | 0 | 1 | 0 | 0 | 0 (`pushButProc`) | 0 | `Link` |
| 131 | (5, 5, 25, 59) | 0 | 1 | 0 | 0 | 0 | 0 | `Unlink` |
| 132 | (132, 184, 195, 247) | 150 | 1 | 151 | 3 | 176 | 0 | `''` |

`procID` 2050 = `128 × 16 + 2` → **`CDEF` 128**, variant 2. `CDEF` 128 is the only `CDEF` in
the file: a 5978-byte compiled 68k resource whose first bytes are
`600e 0000 'CDEF' 0002 0000 …`. **There is no source for it** anywhere in the release.

Two problems:

* **`max = 18` vs 19 menu items.** `CNTL` 128 is the Room Info background popup — `DITL`
  1003 item 11 has type `0x07` (`ctrlItem + resCtrl`) and data `\x00\x80` = **128**. `MENU`
  140 has **19** items, the 19th being `Original Artwork` — which is exactly the item the
  code selectively enables (`kOriginalArtworkItem` = 19, `RoomInfo.c:400`) and selectively
  sets (`SetPopUpMenuValue(theDialog, kRoomPopupItem, kOriginalArtworkItem)` at
  `RoomInfo.c:71` and `:436`). With `max` stuck at 18 the popup cannot represent it, unless
  the undocumented `CDEF` overwrites `max` from the menu's item count.
* **`refCon = 0`.** The convention this program uses is *`refCon` holds the menu's resource
  ID* — which is why `CNTL` 129's `refCon` is **141**, the Tools menu. `CNTL` 128's is 0.
  The only code that could have supplied the menu instead is `AddMenuToPopUp`, whose body
  would have done `(*theControl)->contrlRfCon = (long)theMenu`
  (`GliderPRO/Sources/DialogUtils.c:556`) — and the whole function
  (`DialogUtils.c:545-558`), its prototype (`GliderPRO/Headers/DialogUtils.h:34`), and its
  single call site (`GliderPRO/Sources/RoomInfo.c:434`) are all commented out. See §13.4 and
  §20.7 for the full analysis.

#### 21.7.3 R-3 — `CNTL` 129's `max` is 3 but there are 9 tool modes

`Tools.c:315` creates the class popup with

```c
classPopUp = GetNewControl(kPopUpControl, toolsWindow);      // Tools.c:315, kPopUpControl = 129 (Tools.c:18)
```

and there is **no** `SetControlMaximum` call on it. The only two `SetControlMaximum` calls in
the entire program are the map scroll bars (`GliderPRO/Sources/Map.c:344`, `:350`). Yet
`toolMode` legally runs 1..9 (`GliderDefines.h:272-280`), `NextToolMode`
(`GliderPRO/Sources/Tools.c:493-508`) and `PrevToolMode` (`:512-527`) clamp against 9, and
`SetPopUpMenuValue` (`GliderPRO/Sources/DialogUtils.c:575-583`) just calls `SetControlValue`.

If `CDEF` 128 enforces `max`, modes 4..9 (Transport, Switches, Lighting, Appliances,
Enemies, Clutter) are unreachable *from the popup* — though still reachable from the
*Tools* menu path (`SetSpecificToolMode`, §5.6) and the keyboard. Since the shipped houses
use objects from all nine classes, either the `CDEF` overrides `max` from the attached
menu's item count, or every author used the menu. **This cannot be resolved from the source
alone** — see Open questions.

#### 21.7.4 R-4 — the House Info flags mask clears bit 13 as well as bit 1

`DoHouseInfo` writes the house flags with

```c
(*thisHouse)->flags = (*thisHouse)->flags & 0xFFFFDFFD;      // HouseInfo.c:272
```

`0xFFFFDFFD` = `~0x00002002` — it clears **bit 1** (`0x2`, `phoneBitSet`) *and* **bit 13**
(`0x2000`). Only bit 1 is then conditionally re-set. The decoder is

```c
wardBitSet         = (((*thisHouse)->flags & 0x00000001) == 0x00000001);   // HouseIO.c:416
phoneBitSet        = (((*thisHouse)->flags & 0x00000002) == 0x00000002);   // HouseIO.c:417
bannerStarCountOn  = (((*thisHouse)->flags & 0x00000004) != 0x00000004);   // HouseIO.c:418
```

so bit 13 has no reader. Whatever it once meant, the House Info dialog destroys it on every
OK. (`0xFFFFFFFD` — clearing only bit 1 — is the value the code appears to have intended.)

Note also that `bannerStarCountOn` at `:418` is **inverted**: the flag bit *suppresses* the
banner star count.

**Measured `flags` across the 22 shipped houses:**

| `flags` | Count | Meaning | Houses |
|---------|-------|---------|--------|
| `0x0` | 14 | no ward, no phone, banner star count **on** | (the remainder) |
| `0x2` | 7 | phone bit set | CD Demo House, California or Bust!, Davis Station, Land of Illusion, Nemo's Market, Rainbow's End, SpacePods |
| `0x6` | **1** | phone bit set **and** banner star count suppressed | **Art Museum** |

Art Museum is the only house in the corpus with bit 2 set. No house has bit 0
(`wardBitSet`) or bit 13.

#### 21.7.5 R-5 — `CDEF` 128 and `WDEF` 128/129 ship without source

| Type | IDs | Name | Size |
|------|-----|------|------|
| `CDEF` | 128 | (none) | 5978 bytes |
| `WDEF` | 128 | `Infinity Windoid 2.6` | 5452 bytes |
| `WDEF` | 129 | `Infinity Windoid 2.6 (grow)` | 6476 bytes |
| `MDEF` | — | none | — |

`kWindoidWDEF` is **2048** (`GliderPRO/Headers/GliderDefines.h:531`) = `128 × 16 + 0`, i.e.
`WDEF` 128 variant 0. These three compiled 68k resources are third-party and are the only
part of the editor's look-and-feel with no readable definition. Everything a Go port needs
about them is *behavioural*: `WDEF` 128 draws a small floating "windoid" title strip
(the Tools, Coordinates, and Link palettes), `WDEF` 129 the same with a grow box, and
`CDEF` 128 draws a rectangular popup-menu button that displays the current item's text.

#### 21.7.6 R-6 — `kYellowFailedResAdd` is defined and never used

`#define kYellowFailedResAdd 3` (`GliderPRO/Headers/GliderDefines.h:20`) has no reader; the
only resource-related yellow alert actually raised is
`YellowAlert(kYellowFailedResOpen, …)` (`GliderPRO/Sources/HouseIO.c:573`). Consistent with
§19.2.5: the editor never writes the resource fork, so there is nothing to fail at adding.

---

### 21.8 Dead code, dead constants, dead prototypes

§20 catalogues the deliberately-disabled subsystems (`Scrap.c`, the Drag Manager,
`MoveRoom`, `MoveObject`/`SelectNeighborRoom`, `SaveHouseAs`, `Validate.c`,
`AddMenuToPopUp`, `IsFileReadOnly`, `ShiftWholeHouse`, `OpenSpecificHouse`). This
subsection adds the smaller residue.

#### 21.8.1 D-1 — two prototypes with no definitions anywhere

`GliderPRO/Headers/GliderProtos.h`:

```
433://void PutRoomScrap (void);                    // --- Scrap.c
434://void PutObjectScrap (void);
435:void GetRoomScrap (void);                      // NOT commented
436:void GetObjectScrap (void);                    // NOT commented
437://void SeeIfValidScrapAvailable (Boolean);
```

`GliderPRO/Sources/Scrap.c` is 517 lines of which **lines 8 through 516 are one block
comment** (`/*` at `:8`, `*/` at `:516`), so the definitions at `Scrap.c:104`
(`GetRoomScrap`) and `Scrap.c:170` (`GetObjectScrap`) do not exist in the object file.
`GliderProtos.h:435-436` therefore declare two functions that would be link errors if
called. Nothing calls them — the only call sites are the commented `Menu.c:529-534` — so the
linker never notices.

#### 21.8.2 D-2 — `GetObjectRect` has unreachable code, no `default:`, and mutates its input

`GetObjectRect(objectType *who, Rect *theRect)` (`GliderPRO/Sources/ObjectRects.c:32-273`):

* `GliderPRO/Sources/ObjectRects.c:68-71` is unreachable.
* There is **no `default:`**, so an unrecognised `what` leaves `*theRect` untouched
  (whatever the caller had in it).
* For `kCustomPict` it **writes to the object**:

```c
who->data.g.height = 10000;          // ObjectRects.c:224
```

A geometry query mutating the house data is surprising in itself; here it is also the
mechanism by which a custom-picture object whose PICT is missing gets a fallback ID. Note
`data.g.height` doubles as the user PICT ID for `kCustomPict` (§3.2 variant `g`), and 10000
is the "not found" sentinel. Empirically **2** `kSoundTrigger` objects in the shipped houses
carry `where = 10000` (see §21.9), and `In The Mirror` is where they live.

#### 21.8.3 D-3 — `kSoundTrigger`'s `where` field is write-only

The editor lets you set a sound-trigger's sound (`DoCustPictObjectInfo`,
`GliderPRO/Sources/ObjectInfo.c:1172-1265`, shared between `kCustomPict` and
`kSoundTrigger`; the two `ParamText` variants at `:1186` and `:1188`; the two accepted
numeric ranges at `:1213` and `:1237`), stores it in `data.e.where`, and the *game never
reads it*:

```c
case kSoundTrigger:
PlayPrioritySound(kChordSound, kChordPriority);      // Triggers.c:132
// Change me
```

(`GliderPRO/Sources/Triggers.c:131-133`; the second path is
`PlayPrioritySound(kTriggerSound, kTriggerPriority)` at
`GliderPRO/Sources/Interactions.c:1071-1073`.) Neither consults `where`. The author's own
`// Change me` comment says it was unfinished.

This is confirmed by the data: **Teddy World ships 11 `kSoundTrigger` objects and zero
`snd ` resources**, and two triggers carry `where = 10000`, which is not a plausible
resource ID. See §21.9 for the full histogram.

#### 21.8.4 D-4 — dead navigation code

| Function | Cite | Status |
|----------|------|--------|
| `MoveObject(short whichWay, Boolean shiftDown)` | `GliderPRO/Sources/ObjectEdit.c:1374-1772` (399 lines) | never called — arrow-key nudging was never wired up (§20.4). Contains one of the seven empty `KeepObjectLegal` ifs (`:1700-1702`) and a `kMousehole`/`kFireplace` special case at `:1691-1695`. |
| `SelectNeighborRoom(short)` | `GliderPRO/Sources/Room.c:544` | never called |
| `MoveRoom(Point)` | `GliderPRO/Sources/Map.c:769-795` | never called (§20.3, S-3) |
| `ShiftWholeHouse(short)` | `GliderPRO/Sources/House.c:824-859` | never called (S-4) |

#### 21.8.5 D-5 — `OpenLinkWindow`'s dead reset

```c
linkRoom = -1;                     // Link.c:246
linkObject = 255;                  // Link.c:247
```

inside `OpenLinkWindow` (`GliderPRO/Sources/Link.c:212-252`). These are immediately
overwritten by whichever of the three entry points (§17.4) opened the window, so the reset
never takes effect. The two controls are created just above:
`linkControl = GetNewControl(kLinkControlID, linkWindow)` at `:238` (`CNTL` 130) and
`unlinkControl = GetNewControl(kUnlinkControlID, linkWindow)` at `:242` (`CNTL` 131).

#### 21.8.6 D-6 — `DoObjectInfo`'s `default:` is a beep

```c
case kSoundTrigger:
case kCustomPict:
DoCustPictObjectInfo();                     // ObjectInfo.c:2555-2557
break;
…
default:
SysBeep(1);                                 // ObjectInfo.c:2560-2562
break;
```

Any object class with no info dialog produces an unexplained system beep rather than a
message. §12.16 lists which classes fall through.

---

### 21.9 Empirical: what the 22 shipped houses actually contain

Every number here was computed by parsing the 22 BinHex'd house files in
`GliderPRO/Houses/` with python3.

#### 21.9.1 Corpus summary

| House | Data-fork bytes | `nRooms` | `866 + 348·n` | Δ |
|-------|-----------------|----------|----------------|---|
| Teddy World | 185654 | 531 | 185654 | 0 |
| Leviathan | 165122 | 472 | 165122 | 0 |
| SpacePods | 140762 | 402 | 140762 | 0 |
| Slumberland | 134150 | 383 | 134150 | 0 |
| Land of Illusion | 106310 | 303 | 106310 | 0 |
| ImagineHouse PRO II | 97958 | 279 | 97958 | 0 |
| Rainbow's End | 78470 | 223 | 78470 | 0 |
| Titanic | 73250 | 208 | 73250 | 0 |
| CD Demo House | 72554 | 206 | 72554 | 0 |
| Grand Prix | 61766 | 175 | 61766 | 0 |
| The Asylum Pro | 49586 | 140 | 49586 | 0 |
| Metropolis | 45062 | 127 | 45062 | 0 |
| Nemo's Market | 44018 | 124 | 44018 | 0 |
| Art Museum | 38798 | 109 | 38798 | 0 |
| In The Mirror | 34622 | 97 | 34622 | 0 |
| Castle o' the Air | 30446 | 85 | 30446 | 0 |
| Davis Station | 23486 | 65 | 23486 | 0 |
| Demo House | 16526 | 45 | 16526 | 0 |
| Fun House | 15830 | 43 | 15830 | 0 |
| Empty House | 13046 | 35 | 13046 | 0 |
| California or Bust! | 6434 | 16 | 6434 | 0 |
| **Sampler** | **1564** | **2** | **1562** | **+2** |

* `version` is **`0x0200`** in all 22 — no house in the corpus uses `0x0300`
  (`kNewHouseVersion`, `GliderDefines.h:518`) or the pre-2.0 format that
  `ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746-820`) exists to upgrade.
* **4070 live rooms; ZERO tombstones** (`suite == kRoomIsEmpty == -1`). Every shipped house
  was saved after a `CompressHouse` pass.
* Floor range **−7 .. 39**; suite range **0 .. 127**. The validator's legal box is floor
  −7..56 (`HouseLegal.c:804-805`) and suite 0..127 (`:814-815`), so no shipped room is
  outside it and no shipped house triggers B-6. Observed `bitPlace` = 0..6015 of 8191.
* Sampler's resource fork is **entirely empty** — no `PICT`, no `bnds`, no `snd `, no
  `vers`, no icons.

#### 21.9.2 Links

Counting transports (`0x31`–`0x40`) and switches (`0x41`–`0x48`), excluding
`kSoundTrigger` (`0x49`):

| Quantity | Count |
|----------|-------|
| Objects with `where != -1` (live links) | **2474** |
| …whose destination `(floor, suite)` has **no room** | **189** |
| …with `who > 23` (invalid object index) | **180** |
| …with destination room present **and** `who > 23` (B-1 trigger) | **13** (all CD Demo House) |
| Objects anywhere with `who` in **24..254** | **1** (CD Demo House room 72, `who = 35`) |

Note the corollary: 179 of the 180 bad-`who` objects use exactly **255**, the "no object"
sentinel, so they are *intentionally* room-only links; only one is a genuine stale index.

#### 21.9.3 The `where == -100` anomaly

**165** link-source objects across **4** houses carry `where == -100`, which
`ExtractFloorSuite` for `version >= 0x0200` (`GliderPRO/Sources/Link.c:41-53`) decodes as
`suite = -100/100 = -1`, `floor = (-100 % 100) - 8 = -8`:

| House | Count |
|-------|-------|
| Slumberland | 69 |
| Land of Illusion | 48 |
| Rainbow's End | 36 |
| Castle o' the Air | 12 |

By class:

| `what` | Name | Count |
|--------|------|-------|
| 0x36 | `kCeilingTrans` | 110 |
| 0x3F | `kInvisTrans` | 38 |
| 0x35 | `kFloorTrans` | 12 |
| 0x34 | `kMailboxRt` | 2 |
| 0x41 | `kLightSwitch` | 1 |
| 0x43 | `kThermostat` | 1 |
| 0x45 | `kKnifeSwitch` | 1 |

**No code path in 1.0.4 can produce −100.** There are exactly six `MergeFloorSuite` write
sites and every one pre-adds `kNumUndergroundFloors` (8) to the floor
(`MergeFloorSuite(floor, suite) = (suite × 100) + floor`,
`GliderPRO/Sources/Link.c:34-37`), so the smallest producible value for a legal floor of −7
is `(0 × 100) + (−7 + 8) = 1`. And `ConvertHouseVer1To2`'s digit-swap
(`GliderPRO/Sources/House.c:746-820`) is inside a `where != -1` guard, so it cannot manufacture
−100 either. Whatever wrote these bytes was a pre-1.0.4 build or an external tool. Three of
the five B-2 cases are `where == -100` objects.

#### 21.9.4 `kSoundTrigger`

**122** `kSoundTrigger` (`0x49`) objects live in **14** of the 22 houses. `who` is **255** in
all 122 — sound triggers never name an object.

Per house:

| House | Count | House | Count |
|-------|-------|-------|-------|
| SpacePods | 24 | Nemo's Market | 11 |
| Titanic | 16 | CD Demo House | 10 |
| Art Museum | 15 | In The Mirror | 7 |
| Leviathan | 12 | Davis Station | 5 |
| Teddy World | 11 | California or Bust! | 3 |
| | | Grand Prix | 3 |
| | | ImagineHouse PRO II | 2 |
| | | Rainbow's End | 2 |
| | | Demo House | 1 |

`where` histogram (the intended `snd ` resource ID):

| `where` | n | `where` | n | `where` | n |
|---------|---|---------|---|---------|---|
| 3000 | 35 | 3007 | 2 | 3037 | 1 |
| 3001 | 18 | 3008 | 2 | 3042 | 6 |
| 3002 | 14 | 3009 | 1 | 3043 | 1 |
| 3003 | 6 | 3010 | 1 | 3044 | 3 |
| 3004 | 6 | 3011 | 2 | 3045 | 1 |
| 3005 | 5 | 3012 | 1 | 3046 | 1 |
| 3006 | 13 | | | 3058 | 1 |
| | | | | **10000** | **2** |

The two `10000`s are `GetObjectRect`'s "PICT not found" sentinel (D-2,
`ObjectRects.c:224`) leaking into a field it has no business touching — further evidence
that `where` was never validated for this class. Combined with D-3 (nothing reads it) and
Teddy World's 11 triggers against 0 `snd ` resources, the conclusion is firm: for
`kSoundTrigger`, **`where` is editor-only bookkeeping with no runtime effect in 1.0.4**.

#### 21.9.5 Drawn switches

**608** objects of the five drawn-switch classes (`kLightSwitch` 0x41, `kMachineSwitch` 0x42,
`kThermostat` 0x43, `kPowerSwitch` 0x44, `kKnifeSwitch` 0x45) exist in the corpus. **5** of
them make an out-of-range `GetObjectState` call during normal play-mode drawing — see the
B-2 table in §21.2.2 for the full list with observed byte offsets and values.

---

### 21.10 Consolidated table

Severity: **C** = corrupting, **B** = behavioural, **X** = cosmetic.

| ID | Sev | Symptom | Primary cite | Triggered by shipped data? |
|----|-----|---------|--------------|----------------------------|
| B-1 | C | `retroLinkList[who]`, `who` ≤ 255, in a 24-entry array | `GliderPRO/Sources/House.c:577-580`, `:600-603` | **Yes** — 13 objects, CD Demo House |
| B-2 | C | `GetObjectState(room, object)` validates neither arg | `GliderPRO/Sources/Objects.c:712` | **Yes** — 5 objects, 4 houses |
| B-3 | C | `GoToObjectInRoom` trusts `who`, then `objActive` | `GliderPRO/Sources/ObjectEdit.c:2785, 2787, 2790, 2794` | **Yes** — 1 object (`who = 35`) |
| B-4 | C | `sorted[destObj]`, `destObj` ≤ 255, 24-entry stack array | `GliderPRO/Sources/Objects.c:972, 977` | Yes, if you Bring/Send in an affected room |
| B-5 | C | `rooms[-1]` when `nRooms == 0`, then `SetHandleSize` | `GliderPRO/Sources/HouseLegal.c:751-760` | No (min shipped `nRooms` = 2) |
| B-6 | C | `pidgeonHoles[bitPlace]` after `DebugStr` with no `continue` | `GliderPRO/Sources/HouseLegal.c:665-675` | No (all rooms in the legal box) |
| B-7 | C | unbounded `rooms[room]`; missing `suite < 0` guard; missing `default:` | `GliderPRO/Sources/Room.c:718, 737-759, 562-635` | Indirectly (via B-2's `room = -1`) |
| B-8 | C | `FSRead` of `GetEOF` bytes into a 292-byte struct | `GliderPRO/Sources/HighScores.c:823, 841` | No |
| S-1 | B | `DoLink` silently no-ops if the source room won't resolve; `DoUnlink` has no guard at all | `GliderPRO/Sources/Link.c:275` vs `:325-361` | — |
| S-2 | B | `UpdateLinkControl`: no top-level `default:`, one else-less sub-case; `unlinkControl` never hilited | `GliderPRO/Sources/Link.c:57-194`, esp. `:149-157` | — |
| S-3 | X | `MoveRoom`'s occupied-cell branch is `{ }` | `GliderPRO/Sources/Map.c:784-786` | dead code |
| S-4 | X | `ShiftWholeHouse`: empty inner loop, `#pragma unused` argument | `GliderPRO/Sources/House.c:826, 847-849` | dead code |
| S-5 | B | 7 × `if (KeepObjectLegal()) { }`; 14 × `whoCares =`; 1 real consumer | see §21.3.5 | — |
| S-6 | B | `SortHouseObjects` skips compaction *and* `ForceThisRoom` when `numLinks == 0` | `GliderPRO/Sources/House.c:459-460` | Yes for link-free houses |
| S-7 | X | two `TrackControl` results discarded into empty blocks | `GliderPRO/Sources/Map.c:669-672, 692-695` | — |
| S-8 | B | `SetToNearestNeighborRoom`'s spiral has one exit and no radius cap | `GliderPRO/Sources/Room.c:659-704`, `finished` only at `:667` | — |
| T-1 | X | `data.e.who` written for transports too | `GliderPRO/Sources/House.c:421` | harmless: same offset |
| T-2 | X | `BringSendFrontBack`'s `switch`/`default:` arms swapped | `GliderPRO/Sources/Objects.c:961-978` | harmless: same offset |
| T-3 | X | `Scrap.c` `d`/`e` swap | `GliderPRO/Sources/Scrap.c:147, 152` | dead code |
| V-1 | C | duplicate check runs before the range check | `GliderPRO/Sources/HouseLegal.c:1089` vs `:1110` | enables B-6 |
| V-2 | B | `ValidateRoomNumbers` never resets `houseErrors` | `GliderPRO/Sources/HouseLegal.c:1107-1121` | Yes — inflated error counts |
| V-3 | B | `CheckForStaircasePairs` counts no errors, clobbers `thisRoomNumber` | `GliderPRO/Sources/HouseLegal.c:961-1045`, `:980`, `:1011` | — |
| V-4 | B | `CheckRoomNameLength` zeroes `unusedByte` on tombstones too | `GliderPRO/Sources/HouseLegal.c:868` | validation mutates the file |
| V-5 | B | `ValidateNumberOfRooms` truncating division hides trailing bytes | `GliderPRO/Sources/HouseLegal.c:630-631` | **Yes** — Sampler, +2 bytes |
| V-6 | B | `KeepObjectLegal`'s comment is inverted; 8 mutations unflagged, 4 of them change `what` | `GliderPRO/Sources/HouseLegal.c:38-40` vs `:596`; `:115-137`, `:310-369` | — |
| V-7 | B | `CompressHouse` repairs `firstRoom` and `wasRoom` but not `previousRoom` | `GliderPRO/Sources/HouseLegal.c:712-715` | — |
| V-8 | X | `WrapText` leaves an over-long line for a word longer than `maxChars` | `GliderPRO/Sources/StringUtils.c:216-250` | — |
| U-1 | B | `ToggleToolsWindow` sets `isToolsOpen = true` in both branches | `GliderPRO/Sources/Tools.c:357, 362` | Yes |
| U-2 | X | `HandleMapClick` leaks 2 UPPs on 5 early returns | `GliderPRO/Sources/Map.c:587, 602, 632, 641, 648` | Yes, every map click |
| U-3 | B | *Go To Room…*'s default and Return/Enter target is **Cancel** | `DITL` 1043 item 1; `GliderPRO/Sources/DialogUtils.c:352-363`; `House.c:628-660` | Yes |
| U-4 | B | *Cut Room* / *Clear* delete without confirmation, without clipboard, without undo | `GliderPRO/Sources/Menu.c:509, 544` | Yes |
| U-5 | B | *Copy* is fully commented out yet enabled and relabelled | `GliderPRO/Sources/Menu.c:515-524`; `:165-235` | Yes |
| U-6 | X | `backgroundsMenu` fetched, mutated, never attached, never released | `GliderPRO/Sources/RoomInfo.c:397, 400, 434` | Yes, every Room Info |
| U-7 | C | `CreateNewRoom` loses the staged room if `PtrAndHand` fails | `GliderPRO/Sources/Room.c:200-207` | only under memory pressure |
| U-8 | B | `DeleteRoom` writes `firstRoom = -1` when the deleted room was both first and last | `GliderPRO/Sources/Room.c:461, 468, 476` | — |
| U-9 | B | `ReflectCurrentRoom` hard-codes `CenterMapOnRoom(64, 1)` | `GliderPRO/Sources/Room.c:316` | Yes, every navigation |
| U-10 | B | all four arrow keys hijacked in edit mode; Up/Down start a new game | `GliderPRO/Sources/Events.c:193-207` (arcade arms live; `#if` `:191`, `#else` `:209`, `#endif` `:251`) | Yes |
| U-11 | X | `DoesNeighborRoomExist`'s `#endif` is inside the function body | `GliderPRO/Sources/Room.c:539` | — |
| U-12 | B | `SortHouseList`: two self-comparisons, and skips the swapped element | `GliderPRO/Sources/SelectHouse.c:528-534` | Yes with same-named houses |
| R-1 | B | `MENU` 140 and 141 both declare `menuID` 133 | `Glider PRO.r` | latent |
| R-2 | B | `CNTL` 128: `max = 18` for a 19-item menu, `refCon = 0` (no menu) | `Glider PRO.r`; `GliderPRO/Sources/RoomInfo.c:434` | Yes — popup is empty |
| R-3 | B | `CNTL` 129: `max = 3` for 9 tool modes, no `SetControlMaximum` | `Glider PRO.r`; `GliderPRO/Sources/Tools.c:315` | unknown (`CDEF` has no source) |
| R-4 | B | flags mask `0xFFFFDFFD` clears bit 13 as well as bit 1 | `GliderPRO/Sources/HouseInfo.c:272` | Yes (no house sets bit 13) |
| R-5 | X | `CDEF` 128, `WDEF` 128/129 ship as compiled 68k with no source | `Glider PRO.r` | — |
| R-6 | X | `kYellowFailedResAdd` has no reader | `GliderPRO/Headers/GliderDefines.h:20` | — |
| D-1 | X | `GetRoomScrap` / `GetObjectScrap` declared, never defined | `GliderPRO/Headers/GliderProtos.h:435-436`; `Scrap.c:8`/`:516` | — |
| D-2 | B | `GetObjectRect`: unreachable code, no `default:`, writes `data.g.height = 10000` | `GliderPRO/Sources/ObjectRects.c:68-71`, `:224` | Yes — 2 sound triggers hold 10000 |
| D-3 | B | `kSoundTrigger`'s `where` is never read at runtime | `GliderPRO/Sources/Triggers.c:131-133`; `Interactions.c:1071-1073` | Yes — 122 objects |
| D-4 | X | `MoveObject`, `SelectNeighborRoom`, `MoveRoom`, `ShiftWholeHouse` all dead | see §21.8.4 | — |
| D-5 | X | `OpenLinkWindow`'s `linkRoom = -1; linkObject = 255;` is immediately overwritten | `GliderPRO/Sources/Link.c:246-247` | — |
| D-6 | B | `DoObjectInfo`'s `default:` is `SysBeep(1)` | `GliderPRO/Sources/ObjectInfo.c:2560-2562` | — |

---

## Open questions

Everything below is something I could not settle from the source and data in front of me.
Each entry states what is known, what is missing, and — where it matters for a port — what
I would do in the absence of an answer.

### Q1 — Where did the 165 `where == -100` links come from?

**Known.** 165 link-source objects in 4 houses (Slumberland 69, Land of Illusion 48,
Rainbow's End 36, Castle o' the Air 12) store `where = -100`, which `ExtractFloorSuite` for
`version >= 0x0200` decodes as suite −1 / floor −8 (`GliderPRO/Sources/Link.c:41-53`). Full
per-class breakdown in §21.9.3.

**Missing.** No code path in 1.0.4 can produce it. There are exactly six `MergeFloorSuite`
write sites and every one pre-adds `kNumUndergroundFloors` (8) to the floor before merging
(`MergeFloorSuite(floor, suite) = (suite * 100) + floor`,
`GliderPRO/Sources/Link.c:34-37`), so the minimum producible value for the minimum legal
floor (−7) is `(0 * 100) + 1 = 1`. `ConvertHouseVer1To2`'s digit swap
(`GliderPRO/Sources/House.c:746-820`) is inside a `where != -1` guard and cannot manufacture
−100 either.

**Hypotheses I can neither confirm nor refute:** (a) an earlier Glider PRO beta whose
`MergeFloorSuite` did not pre-add the underground offset, writing `(−1 * 100) + 0 = -100` for
"suite −1, floor 0"; (b) a third-party house editor; (c) `-100` as a deliberate hand-poked
"deleted link" sentinel that predates `-1`.

**What a port should do.** Treat any `where` that decodes to `suite < 0` as "unlinked",
exactly as `RoomExists`' `if (suite < 0) return (foundIt);` guard
(`GliderPRO/Sources/Room.c:400-401`) already does for room lookup. Do **not** special-case
the literal −100.

### Q2 — Is `bannerStarCountOn`'s inversion intentional?

**Known.** `wardBitSet` and `phoneBitSet` decode as "bit set means feature on"; the third
flag does the opposite:

```c
wardBitSet        = (((*thisHouse)->flags & 0x00000001) == 0x00000001);   // HouseIO.c:416
phoneBitSet       = (((*thisHouse)->flags & 0x00000002) == 0x00000002);   // HouseIO.c:417
bannerStarCountOn = (((*thisHouse)->flags & 0x00000004) != 0x00000004);   // HouseIO.c:418
```

**Missing.** Whether `!=` at `:418` is a deliberate "suppress" bit or a typo for `==`.

**Evidence.** Exactly **one** shipped house sets bit 2 — Art Museum (`flags = 0x6`). If the
inversion were a bug, Art Museum would be the only house *with* a banner star count and the
other 21 would be without; the shipped screenshots and the fact that the star count is a
headline feature make the "bit 2 = suppress" reading far more plausible. I have documented it
as intentional but flagged it here.

**What a port should do.** Reproduce `!=`. Round-tripping matters more than the semantics.

### Q3 — Is `0xFFFFDFFD` a typo, and what was bit 13?

**Known.** `DoHouseInfo` writes
`(*thisHouse)->flags = (*thisHouse)->flags & 0xFFFFDFFD;`
(`GliderPRO/Sources/HouseInfo.c:272`). `0xFFFFDFFD` = `~0x00002002`, so it clears bit 1
(`phoneBitSet`, which is then conditionally re-set) **and** bit 13 (`0x2000`), which has no
reader anywhere in the program.

**Missing.** Whether the intended constant was `0xFFFFFFFD` (clear bit 1 only) and, if bit 13
was real, what it meant.

**Evidence.** No shipped house has bit 13 set (all 22 have `flags` ∈ {0x0, 0x2, 0x6}), so
nothing observable is lost. The `D` nibble is one keystroke from `F`, which suggests a typo.

**What a port should do.** Clear only the bits it re-writes. Preserve unknown flag bits on
save — that is strictly better than 1.0.4 and cannot break any existing house.

### Q4 — Why does `Sampler` have 2 trailing bytes and an empty resource fork?

**Known.** 21 houses satisfy `len(dataFork) == 866 + 348 * nRooms` exactly. Sampler is 1564
bytes with `nRooms = 2`, i.e. **1562 + 2**. Its resource fork contains no `PICT`, no `bnds`,
no `snd `, no `vers`, no icons — it is the only house in the corpus with an empty resource
fork. Its `timeStamp` decodes to **2000-05-11**, five years after the other 21 (§19.7.1).

**Missing.** What the 2 bytes are and what wrote them. `ValidateNumberOfRooms`' truncating
division (`GliderPRO/Sources/HouseLegal.c:630-631`, V-5) means the original never noticed.

**What a port should do.** Ignore trailing bytes beyond `866 + 348 * nRooms` on load; do not
write them back. Log a warning rather than refusing the file.

### Q5 — What is `CDEF` 128's `max` / `refCon` contract, and can the Tools popup reach modes 4–9?

**Known.** `CNTL` 129 (the Tools class popup) has `max = 3`, `refCon = 141`, `procID = 2050`
(= `CDEF` 128 variant 2). `MENU` 141 has **9** items; `toolMode` legally runs 1..9
(`GliderPRO/Headers/GliderDefines.h:272-280`); `NextToolMode`/`PrevToolMode`
(`GliderPRO/Sources/Tools.c:493-527`) clamp against 9; there is **no** `SetControlMaximum`
call on this control anywhere (the only two in the program are the map scroll bars,
`GliderPRO/Sources/Map.c:344`, `:350`).

**Missing.** `CDEF` 128 is a 5978-byte compiled 68k resource with **no source in the
release**. I cannot tell whether it (a) overwrites `contrlMax` from the attached menu's item
count at `initCntl` time, or (b) honours the resource's `max = 3` and clamps
`SetControlValue`.

**Evidence for (a).** The shipped houses use objects from all nine classes, and the CNTL's
`max = 18` for the Room Info popup (whose menu has 19 items, R-2) is *also* one short — a
systematic off-by-one that a `CDEF` recomputing from the menu would make invisible.

**What a port should do.** Ignore `max` entirely; drive the popup from the mode list.

### Q6 — Which `doPrettyMap` default is right?

`Main.c:187` sets `doPrettyMap = false` in the "no preferences file" arm; the settings
defaults routine sets `doPrettyMap = true` (`GliderPRO/Sources/Settings.c:1230`). The flag
selects the map window's detailed room rendering (`GliderPRO/Sources/Map.c:231`). Which the
author intended for a first launch is unknowable; the practical answer is `true`, since the
pretty map is the documented feature.

### Q7 — Are user `PICT` IDs 3800–9999 reachable as room backgrounds?

**Known.** `kUserBackground` = 3000 and `kUserStructureRange` = 3300
(`GliderPRO/Headers/GliderDefines.h:522-523`), and `IsRoomAStructure`
(`GliderPRO/Sources/Room.c:763`…) treats a background ID ≥ 3300 as a structure. §19.2.2
documents the observed convention: 3000–3299 = ordinary user backgrounds, 3300–3799 =
user structures.

**Missing.** Nothing enforces an upper bound. `ChooseOriginalArt`
(`GliderPRO/Sources/RoomInfo.c:711`) enumerates whatever `PICT`s the house's resource fork
contains, and `PictIDExists` (`:830`) just probes. So an ID of 5000 *would* load. Whether the
author intended 3800–9999 to be usable, and what `10000` (the `GetObjectRect` "not found"
sentinel, `ObjectRects.c:224`) collides with, is undecided.

**What a port should do.** Accept any ID present in the house's picture set; classify
`>= 3300` as a structure; reserve 10000 as the not-found sentinel for compatibility.

### Q8 — Why do `MENU` 140 and `MENU` 141 share `menuID` 133?

Both declare `menuID` 133 with different resource IDs and different contents (19 room
backgrounds vs 9 tool classes). Neither is inserted into the menu bar, so it is latent
(R-1). Most likely one was cloned from the other and the `menuID` field was never edited.
Unknowable from the source.

### Q9 — What was `kNewHouseVersion` (`0x0300`) for?

`HouseIO.c:394-400` reads the version and refuses to open anything
`>= kNewHouseVersion` with `YellowAlert(kYellowNewerVersion, 0)`. Nothing in 1.0.4 ever
*writes* `0x0300`, and no shipped house uses it — all 22 are `0x0200`. It is a forward
compatibility guard for a format that never shipped. What the 3.0 format would have added
(more than 24 objects per room? more than 255 rooms? a real link table?) is unknown.

### Q10 — What did the reserved fields hold?

Three fields are named "unused" and are never read:

| Field | Location | Size |
|-------|----------|------|
| `houseType.unusedShort` | house-header offset 2..3 | 2 |
| `houseType.unusedBoolean` | house-header offset 861 | 1 |
| `roomType.unusedByte` | room offset 32 | 1 |

`roomType.unusedByte` is *written* — `CheckRoomNameLength` zeroes it for every room including
tombstones (`GliderPRO/Sources/HouseLegal.c:868`, V-4). A survey of the shipped houses would
tell you whether any of them are non-zero before that pass; I did not run it, and it would
not tell you what the bytes meant.

### Q11 — What is `wardBitSet` (flags bit 0)?

`wardBitSet` is decoded from flags bit 0 (`GliderPRO/Sources/HouseIO.c:416`), initialised
`false` (`GliderPRO/Sources/House.c:142`), and read in exactly one place:

```c
if (who == kRoomIsEmpty)        // This call should be smarter than this   // RoomGraphics.c:193
{
    if (wardBitSet)                                                        // RoomGraphics.c:195
    {
        …  // draws something into backSrcMap
```

There is **no editor UI for it** — House Info edits only the phone bit (§14.7) — and **no
shipped house sets it** (all 22 have `flags` bit 0 clear). So the feature is unreachable in
1.0.4. The author's own comment at `:193` says the surrounding dispatch is provisional.

### Q12 — The blower `vector` field's lost "forceful" bit

§12.3.1 documents that `blowerType.vector` (payload offset 8, a `Byte`) encodes a direction in
its low bits and that the info dialog once offered a "forceful" toggle in a higher bit which
the shipped dialog no longer exposes. Which bit, and whether the physics still honours it, I
could not pin down from the drawing and interaction code I read; a full audit of
`Interactions.c` / `Environs.c` would be needed.

### Q13 — Undo: was it ever attempted?

There is **no** undo of any kind (§20.5): no undo buffer, no `iUndo` menu item in `MENU` 131,
no saved-state snapshot, and the Cut/Clear paths do not even put the deleted object or room on
the (disabled) clipboard. Whether an undo design existed and was cut, or was never started, is
not recoverable from the source. The absence is total and deliberate enough that the port
should treat undo as a **new feature**, not a restoration.

### Q14 — What was the `room.bounds` structure bit (0x20) for?

§13.11 and §19.2.3 document the `bounds` word: bit 0 = "in-house bounds present",
`bounds >> 1` gives 1 = left open, 2 = top open, 4 = right open, 8 = bottom open, 16 = floor
support, and `bounds & 32` is a further "structure" bit. The relationship between that bit,
the 4-byte `bnds` resource, and `IsRoomAStructure`'s ID-range test
(`background >= kUserStructureRange`) is not fully consistent in the code, and I could not
determine which is authoritative when they disagree.

---

## Porting notes

This section is the "what to actually build" summary. It assumes the reader has §§1–21 to
hand for detail and will not repeat citations already given there.

### P1 — The on-disk format is the contract; everything else is negotiable

The editor's entire purpose is to produce a byte layout that the game reads. That layout is
fixed and small:

| Structure | Size | Cite |
|-----------|------|------|
| `objectType` | **12** bytes | `GliderPRO/Headers/GliderStructs.h:11-105` |
| `roomType` | **348** bytes | `GliderPRO/Headers/GliderStructs.h:166-180` |
| `houseType` prefix | **866** bytes | `GliderPRO/Headers/GliderStructs.h:182-198` |
| `scoresType` | 292 bytes | `GliderPRO/Headers/GliderStructs.h:107-114` |
| `gameType` | 40 bytes | `GliderPRO/Headers/GliderStructs.h:116-134` |
| `boundsType` | 4 bytes (4 × `Boolean`) | `GliderPRO/Headers/GliderStructs.h:266-272` |
| `linksType` | 8 bytes | `GliderPRO/Headers/GliderStructs.h:281-285` |
| `retroLink` | 4 bytes | (`{short room; short object;}`) |

File length is exactly `866 + 348 * nRooms` (verified for 21 of 22 shipped houses; Sampler is
+2, Q4).

Non-negotiables for a Go port:

1. **Big-endian.** Use `encoding/binary` with `binary.BigEndian`, never
   `binary.NativeEndian`.
2. **2-byte 68k alignment.** `#pragma options align=mac68k` (`GliderPRO/Headers/Externs.h:231`,
   reset `:269`). Do **not** rely on Go struct layout; marshal field-by-field at explicit
   offsets. The 866/348/12 sizes above are the *packed-at-2* sizes and there are no interior
   pad bytes to guess at, but writing an explicit codec removes the whole question.
3. **`Rect` is `{top, left, bottom, right}`** — in that order, 4 × `int16`.
4. **`Point` is `{v, h}` — vertical FIRST.** This is the single most common porting error in
   this codebase (it produced a false result in my own snap-rule survey before I caught it).
   Every `Point topLeft` in the object payload is `(v, h)`.
5. **Pascal strings.** `Str27` (`roomType.name`, offsets 0..27) is a length byte followed by
   up to 27 bytes; `Str255` (banner 16..271, trailer 272..527) is a length byte followed by up
   to 255. The text is **Mac Roman**, not UTF-8 — `É` (0xC9) is the ellipsis the sources use
   in comments and `…` in resource strings. Decode on load, re-encode on save, and never let a
   round-trip lengthen a string past its field.
6. **The object payload is a 10-byte untyped blob with nine overlapping interpretations.**
   Model it as `[10]byte` plus typed accessors, or as a single struct whose `Where`/`Who`
   fields are shared. See P4.

### P2 — What to replace, Toolbox call by Toolbox call

| Mac Toolbox | Used for | Go replacement |
|-------------|----------|----------------|
| QuickDraw `Rect`/`Point`/`SetRect`/`OffsetRect`/`InsetRect`/`SectRect`/`PtInRect` | all editor geometry | `image.Rectangle`/`image.Point` — but keep your own type so `{v,h}` order is explicit and so you can port `ZeroRectCorner` (`GliderPRO/Sources/RectUtils.c:57`) and `ForceRectInRect` verbatim |
| `PenMode(patXor)` + `FrameRect` | the marquee and every selection handle | draw an overlay layer and composite; XOR-on-screen has no modern analogue and is not needed |
| `GWorld` / `NewGWorld(..., useTempMem)` / `CopyBits` / `CopyMask` | offscreen room composition | `*image.RGBA` (or a GPU texture) + `draw.Draw`/`draw.DrawMask`. Note `CreateOffScreenGWorld` (`GliderPRO/Sources/Utilities.c:266-278`) **ignores its error return** — do not copy that |
| 8-bit indexed colour, `kPreferredDepth = 8`, `ColorText(str, 171L)` | all art | decode the `PICT`s to RGBA once at load; keep the CLUT only if you want pixel-exact output |
| Resource Manager (`GetPicture`, `GetResource('bnds')`, `GetResource('snd ')`, `Count1Resources('PICT')`, `GetIndString`) | house-supplied art, sounds, bounds, and the 144-entry `STR#` 1007 name table | a resource-fork reader; §19.2 documents every type the houses actually use. `STR#` 1007's 144 strings are in §12.17 and can be a Go slice literal |
| Dialog Manager (`GetNewDialog`, `ModalDialog`, `ParamText` `^0`..`^3`, DITL item numbers) | every info dialog | your UI toolkit. §§12–14 give every DITL's item numbers and rects; the item *numbers* are the load-bearing part, the rects only matter for visual fidelity |
| `CDEF` 128 popup (procID 2050) | Tools class popup, Room Info background popup | a normal combo box (see Q5 — ignore `max`) |
| `WDEF` 128/129 "Infinity Windoid 2.6" (`kWindoidWDEF` = **2048** = 128 × 16, `kWindoidGrowWDEF` = **2064** = 129 × 16, `GliderPRO/Headers/GliderDefines.h:531-532`) | Tools, Coordinates, Link palettes | floating tool windows |
| Memory Manager `Handle`, `HLock`/`HUnlock`/`HGetState`/`HSetState`, `PtrAndHand`, `SetHandleSize` | the house lives in one relocatable block | a Go `[]Room` slice. **This is where the bug classes in §21.2 disappear** — and where behaviour legitimately diverges from 1.0.4 (§21.2.9) |
| File Manager (`FSpCreate`, `FSpOpenDF`, `FSRead`, `FSWrite`, `SetEOF`, `GetEOF`) | house I/O | `os`/`io`. §19.19 tabulates every disk-touching call |
| Navigation Services (`NavPutFile(..., 'gliH', 'ozm5', ...)`) | Save As (dead — §19.17.1) | a normal file dialog; the type/creator codes `'gliH'`/`'ozm5'` are worth preserving as a magic marker |
| Scrap Manager, Drag Manager | **both entirely disabled** (§20.1, §20.2) | implement from scratch; there is nothing to port |
| Sound Manager `PlayPrioritySound` | `kSoundTrigger` (which ignores its own `where` — D-3) | any audio backend |
| `GetKeys`/`BitTst`, `StillDown`, `WaitMouseUp`, `GetMouse` | drag loops (`DragObject`, `DragHandle`, `DragMiniTile`) | your event loop. The drag loops in §§7–8 are written as poll-until-mouse-up; restructure as event-driven and the pseudocode still maps 1:1 |
| `SetOrigin(1, 1)` on the map window | map coordinate offset | do the offset in your transform, not in the drawing context — this one is easy to lose |
| `SpinCursor`/`IncrementCursor`/`InitCursor` | validation progress | a progress indicator |
| `DebugStr` | one range check (`GliderPRO/Sources/HouseLegal.c:668`, B-6) | a real error return |

### P3 — Build the room store as a slice, and accept the divergence

Almost every corrupting bug in §21.2 exists because the house is one flat block and
`rooms[i].objects[n]` is pointer arithmetic that silently wraps into the next room
(§21.2.9's `348 = 29 × 12`, `60 = 5 × 12` identity, verified byte-for-byte against CD Demo
House). A Go `[]Room` with `[24]Object` panics instead.

Therefore:

* **Bound-check on load, once, and record what you fixed.** Specifically: `who > 23` → treat
  as 255 ("no object"); `where` decoding to `suite < 0` → treat as −1 ("no link"); a
  destination `(floor, suite)` with no room → keep the bytes but mark the link dangling.
  The measured scale of this in the shipped corpus is 2474 links, 189 dangling destinations,
  180 bad `who`, of which only 1 is a *plausible* stale index rather than the 255 sentinel
  (§21.9.2).
* **Round-trip the bytes you did not understand.** If you normalise `who` from 255 to some Go
  sentinel, write 255 back out. Otherwise re-saving CD Demo House changes 180 objects.
* **Expect not to be byte-identical after an edit.** Even 1.0.4 is not: `CheckRoomNameLength`
  zeroes `unusedByte` on every room including tombstones (V-4), so validation mutates a file
  that has zero errors.

### P4 — Model `where` / `who` once, not per class

Union variants `d` (`transportType`) and `e` (`switchType`) put `where` at payload 6:8 and
`who` at payload 8 — **identical offsets**. That is the only reason the three "wrong union
member" writes in the original are harmless (T-1, T-2, T-3). Do not port them as-is into a
world with real types; give the object a single link pair:

```go
type Object struct {
    What    ObjClass
    Payload [10]byte   // authoritative bytes, always round-tripped
}

func (o *Object) Where() int16 { return int16(binary.BigEndian.Uint16(o.Payload[6:8])) }
func (o *Object) Who()   uint8 { return o.Payload[8] }
```

and derive the class-specific views (`topLeft`, `distance`, `length`, `points`, `pict`,
`height`, `delay`, `initial`, `state`, `vector`, `tall`, `wide`, `type`) from the same
`Payload`. §3.2 gives all nine variants' field offsets.

The link encoding itself is two functions and must be copied exactly:

```
MergeFloorSuite(floor, suite) = (suite * 100) + floor          // Link.c:34-37
   — every caller passes floor + kNumUndergroundFloors (8)

ExtractFloorSuite(combo) —                                     // Link.c:41-53
   version <  0x0200:  floor = combo/100 - 8;  suite = combo % 100
   version >= 0x0200:  suite = combo/100;      floor = combo % 100 - 8
```

Both use **C truncating division toward zero**, which Go's `/` and `%` also do for `int`, so
the port is literal. `where == -1` means no link; `who == 255` means no object;
`suite == -1` (`kRoomIsEmpty`) marks a deleted room.

### P5 — The five sentinel values you must not confuse

| Value | Name | Domain | Cite |
|-------|------|--------|------|
| `-1` | `kObjectIsEmpty` | `objectType.what` | `GliderPRO/Headers/GliderDefines.h:526` |
| `-1` | `kRoomIsEmpty` | `roomType.suite` (tombstone) | `GliderPRO/Headers/GliderDefines.h:525` |
| `-1` | `kNoObjectSelected` | `objActive` | `GliderPRO/Headers/GliderDefines.h:527` |
| `-1` | (no link) | `data.{d,e}.where` | `GliderPRO/Sources/Link.c` |
| `255` | (no object) | `data.{d,e}.who` | — never named |

Plus the three negative `objActive` values for the gliders: `kInitialGliderSelected` = **−2**,
`kLeftGliderSelected` = **−3**, `kRightGliderSelected` = **−4**
(`GliderPRO/Headers/GliderDefines.h:528-530`). Any port that uses `-1` as a generic "none"
will silently conflate a tombstoned room with an unselected object.

Two more that are easy to miss: `10000` is `GetObjectRect`'s "PICT not found" sentinel written
into `data.g.height` (`GliderPRO/Sources/ObjectRects.c:224`, D-2), and it has leaked into two
shipped `kSoundTrigger` `where` fields (§21.9.4).

### P6 — Coordinate systems, in the order you will need them

1. **Room-local pixels.** `kRoomWide` = **512**, `kTileHigh` = **322**
   (`GliderPRO/Headers/GliderDefines.h:498-499`). All object `topLeft`/`bounds` values are in
   this space. Snapping rules in §18.17 are all expressed here.
2. **Room grid.** `(floor, suite)`. Legal floor **−7..56**, legal suite **0..127**
   (`GliderPRO/Sources/HouseLegal.c:804-805`, `:814-815`) — a 64 × 128 = **8192** cell space,
   exactly `kRoomsTimesSuites` (`GliderPRO/Sources/HouseLegal.c:648`), addressed as
   `bitPlace = (floor + 7) * 128 + suite` (`:665-666`).
3. **Map cells.** `v = kMapGroundValue - floor` with `kMapGroundValue` = **56**, defined in
   `GliderPRO/Sources/Map.c:22` — **not** in any header, which is easy to miss. Cell size is
   `kMapRoomWidth` = **32** × `kMapRoomHeight` = **20**
   (`GliderPRO/Headers/GliderDefines.h:246-247`).
4. **Link encoding.** `(suite * 100) + floor + 8` (P4).

Observed ranges in the shipped corpus: floor **−7..39**, suite **0..127**, `bitPlace`
**0..6015** (§21.9.1). Nothing needs a 56-floor ceiling in practice, but the validator
enforces one.

### P7 — Keep `KeepObjectLegal`, fix its reporting

`KeepObjectLegal` (`GliderPRO/Sources/HouseLegal.c:42-597`) is the single most important
function to port faithfully: it is what makes the editor produce *playable* geometry, and
§18.17.12's table enumerates every snap rule. Port the rules verbatim.

But fix three things:

1. **Return a "changed" flag with the sense the name implies.** The original returns
   `unchanged` (`:596`) while its comment (`:38-40`) promises the opposite, and 21 of 22 call
   sites discard it anyway (S-5).
2. **Flag the eight currently-silent mutations** (V-6): the three floor-object v-snaps
   (`:115-119`, `:120-125`, `:126-131`), the unreachable `kFloorTrans` snap (`:132-137`), and
   above all the **four handedness auto-flips that rewrite `theObject->what`** (`:310-324`,
   `:325-339`, `:340-354`, `:355-369`). Silently changing an object's class when the user
   drags a door to the wrong wall is exactly the kind of thing a modern editor should say out
   loud.
3. **Make the `kFloorTrans` case reachable or delete it.** It is absent from the governing
   case list at `:75-90` and, if reached, would write `data.a.distance` (payload 4:6) where a
   transport keeps `data.d.tall`.

Empirically, **zero** objects in the 22 shipped houses violate the snap rules — the shipped
houses were all saved through this function.

### P8 — Reorder and re-scope validation

`CheckHouseForProblems` (§18.4, §21.5) needs three changes beyond a literal port:

1. **Range-check before pigeon-holing.** `ValidateRoomNumbers` (step 6, `:1110`) must run
   before `CheckDuplicateFloorSuite` (step 3, `:1089`), or the 8192-entry array is indexed
   out of range on the first pass over a bad file (V-1 + B-6). Better: replace the array with
   a `map[[2]int16]int`.
2. **Reset the error count per step.** Step 6 is the only one without a preceding reset
   (V-2), so its message double-counts step 3.
3. **Make `CheckForStaircasePairs` report.** It repairs silently and counts nothing (V-3).
   And it must not clobber the global current-room number the way the original does
   (`:980`, `:1011`).

Also: `LopOffExtraRooms` must early-out on `nRooms == 0` (B-5), and validation should not
mutate tombstones (V-4).

### P9 — Build the features 1.0.4 shipped without

The following are *absent*, not broken. All of them are cheap in Go and all of them are
things a level author will immediately want:

| Feature | Status in 1.0.4 | Cite |
|---------|-----------------|------|
| Undo / redo | never existed | §20.5, Q13 |
| Clipboard (cut/copy/paste of objects and rooms) | `Scrap.c` is one 509-line block comment; menu items still enabled | §20.1, U-4, U-5, D-1 |
| Drag-and-drop | Drag Manager code present but disabled | §20.2 |
| Move a room to a new `(floor, suite)` | `MoveRoom` dead, with an empty occupied-cell branch | §20.3, S-3 |
| Arrow-key nudging of objects | `MoveObject` (399 lines) never called; arrow keys hijacked by the arcade build | §20.4, U-10 |
| Save As | `SaveHouseAs` dead | §19.17.1 |
| Room background chooser | the popup has no menu attached | §13.4, §20.7, R-2, U-6 |
| A searchable list of user `PICT` IDs | only a 19-item fixed menu + "Original Artwork" | §19.2.2, §13.11 |
| Any feedback when a link fails | `DoLink` silently no-ops | S-1 |
| Confirmation on destructive menu actions | `DeleteRoom(false)` from Cut and Clear | U-4 |

When you implement Move Room, remember what killed it: rewriting `(floor, suite)`
invalidates every link that addresses the room as `(suite * 100) + floor + 8`. Either rewrite
all inbound links atomically, or store links as room **indices** internally and only encode
`(floor, suite)` at save time.

### P10 — The bits of the original UI worth keeping verbatim

Not everything here is a mistake. These are good and should survive the port:

* **The nine-mode tool palette** with per-mode object ranges and the
  `what = toolSelected + ((toolMode - 1) * 0x0010)` encoding (§5.5) — a compact,
  discoverable design. `MENU` 141's nine titles are the labels.
* **Page Up / Page Down to cycle tool modes** (`GliderPRO/Sources/Events.c:181-189`) — the
  only two arrow-adjacent keys the arcade build did not steal.
* **The live map window** with click-to-navigate, click-empty-to-create
  (`QueryNewRoom`, `GliderPRO/Sources/Map.c:711`), and the room thumbnail strip (§15).
* **The "Linked From?" reverse-link button** (the `retroLinkList` idea, §17.10) — bound-check
  it (B-1) but keep it; navigating a link backwards is genuinely useful.
* **Per-class info dialogs** rather than a generic property inspector: 15 dialogs, each with
  only the fields that class has (§12).
* **The validation pass on save** with a progress window and coloured severity
  (`ForeColor(redColor)` for structural problems, `blueColor` for cosmetic ones —
  `GliderPRO/Sources/HouseLegal.c:1095`, `:1133`) (§18.3).
* **`AddObjectPairing`** — placing an up-staircase automatically creates the matching
  down-staircase in the room above (§10, ten pairings). This is real authoring leverage.

### P11 — Test corpus

The 22 BinHex'd houses in `GliderPRO/Houses/` are the regression
suite. Concretely:

| Test | House | Why |
|------|-------|-----|
| Round-trip byte-identity | all 22 | load → save with no edits must reproduce the input (modulo `unusedByte`, V-4) |
| Size identity `866 + 348n` | all 22 | 21 exact, Sampler +2 (Q4) |
| Version gate | all 22 are `0x0200` | nothing exercises `ConvertHouseVer1To2` or the `0x0300` refusal — write synthetic files for both |
| Tombstone handling | **none** — 0 tombstones in 4070 rooms | must be tested synthetically |
| Out-of-range `who` | CD Demo House (13 cases), + 179 `who == 255` across the corpus | §21.9.2 |
| `where == -100` | Slumberland 69, Land of Illusion 48, Rainbow's End 36, Castle o' the Air 12 | §21.9.3 |
| Dangling link destinations | 189 across the corpus | §21.9.2 |
| Sound triggers with no `snd ` | Teddy World (11 triggers, 0 `snd ` resources) | §21.9.4, D-3 |
| `where == 10000` sentinel leak | In The Mirror | §21.9.4 |
| Empty resource fork | Sampler | Q4 |
| Banner-star suppression flag | Art Museum (`flags = 0x6`, the only one) | R-4 |
| Phone flag | 7 houses (`flags = 0x2`) | R-4 |
| Largest house | Teddy World (531 rooms, 185654 bytes) | scaling |
| Smallest house | Sampler (2 rooms, 1564 bytes) | edge cases |
| Object-index aliasing | CD Demo House rooms 72/73 at offset 26402 | §21.2.9 — a Go port must **not** reproduce this |

Two assertions worth hard-coding: **all 4070 live rooms are inside the validator's legal box**
(floor −7..56, suite 0..127), and **zero objects violate `KeepObjectLegal`'s snap rules**. If
your port's validator disagrees with either, the port is wrong, not the data.

### P12 — Things I would deliberately *not* port

* `Validate.c` — the copy-protection / installation check (§20.6). Delete the concept.
* `IsFileReadOnly` (§19.17.2) and the high-scores side-car for locked houses (§19.16) — use
  ordinary filesystem permissions.
* `ShiftWholeHouse` (S-4), `MoveObject` (D-4), `SelectNeighborRoom` (D-4), `MoveRoom` (S-3),
  `SaveHouseAs` (§19.17.1) — dead; rewrite from the spec, do not resurrect the code.
* `AddMenuToPopUp` (§20.7) — replaced by a real combo box.
* The `BUILD_ARCADE_VERSION` key map (U-10).
* `GetObjectRect`'s mutation of its own input (D-2). Return the fallback, do not write it.
* The `unusedShort` / `unusedByte` / `unusedBoolean` semantics (Q10) — preserve the bytes,
  give them no meaning.
