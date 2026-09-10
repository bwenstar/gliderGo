# Editor object manipulation — the six user gestures, clamping, and the geometry cache

**Subject:** `GliderPRO/Sources/ObjectEdit.c` (2810 lines — the largest single source file in
Glider PRO 1.0.4), together with its direct collaborators
`GliderPRO/Sources/Marquee.c` (511 lines),
`GliderPRO/Sources/Tools.c` (544 lines),
`GliderPRO/Sources/HouseLegal.c` (`KeepObjectLegal`),
`GliderPRO/Sources/ObjectRects.c` (`GetObjectRect`),
`GliderPRO/Sources/Coordinates.c` (196 lines — the coordinate windoid), and
`GliderPRO/Sources/ObjectDraw.c` (`DrawClockDigit`).

**Status:** all line numbers in this document are taken from CR→LF converted copies of the
original Classic-Mac (CR-only) sources. The conversion used was
`tr '\r' '\n' < GliderPRO/Sources/X.c > /tmp/wf-objedit/src/X.c`. Citations are written
`GliderPRO/Sources/ObjectEdit.c:123` where `123` is the 1-based line number in the converted
copy. All byte-level claims in this document were verified by parsing the 22 shipped
`GliderPRO/Houses/*.binhex` files with `python3` (31 440 objects); the observed values are
tabulated inline.

**Relationship to the other analysis docs.** This document is the depth pass on the *editing*
half of `ObjectEdit.c`. It assumes but does not repeat:

| Doc | What it already covers | What this doc adds |
|-----|------------------------|--------------------|
| `docs/analysis/editor.md` §6–§11 | Editor mode entry, menu wiring, room/house-level editing, an overview of selection and of `DuplicateObject`/`DeleteObject` | Per-gesture field-write matrices, exact case-label sets and counts, the constraint switches, the geometry cache's sentinel values, three corrections to `editor.md` (see §19) |
| `docs/analysis/object-taxonomy.md` §4.2 | The forced-coordinate object types (`kFloorVent` v=305, doors at fixed lefts, …) | How the editor's commit switches and `KeepObjectLegal` cooperate to *maintain* those forcings, and the one ordering bug that breaks containment |
| `docs/analysis/house-format.md` §10.9 | `KeepObjectLegal` as a house-load normaliser | `KeepObjectLegal` as a post-*edit* normaliser: return-value contract, the **ten** blocks that forget to clear `unchanged` (nine tabulated in §12.1 plus `HouseLegal.c:463`, §12.8), and an empirical check of every invariant against 31 440 shipped objects |
| `docs/analysis/structs.md` | `objectType`, `roomType`, `houseType` layouts | Which union arm each gesture writes, byte offset by byte offset |
| `docs/analysis/object-draw-all.md` | `ObjectDraw.c` draw routines | `DrawClockDigit` (§17), which that doc does not name |
| `docs/analysis/rendering.md` | `ReadyBackground`, room compositing | The editor's invalidate-and-recomposite quartet and which gestures skip it |
| `docs/analysis/input.md` | Key/mouse event routing | Why the arrow-key nudge path is dead code in the shipped build (§7.1) |

---

## Table of contents

- [0. Scope, build flags, and what is compiled out](#0-scope-build-flags-and-what-is-compiled-out)
- [1. Cast of characters: globals, sentinels, and the geometry cache](#1-cast-of-characters-globals-sentinels-and-the-geometry-cache)
- [2. Gesture → entry-point map](#2-gesture--entry-point-map)
- [3. Selection: hit testing, sentinels, lifetime](#3-selection-hit-testing-sentinels-lifetime)
- [4. Handle geometry](#4-handle-geometry)
- [5. Gesture 1 — drag-move (`DragObject`)](#5-gesture-1--drag-move-dragobject)
- [6. Gesture 2 — resize by handle (`DragHandle`)](#6-gesture-2--resize-by-handle-draghandle)
- [7. Gesture 3 — arrow-key nudge (`MoveObject`)](#7-gesture-3--arrow-key-nudge-moveobject)
- [8. Gesture 4 — duplicate (`DuplicateObject`)](#8-gesture-4--duplicate-duplicateobject)
- [9. Gesture 5 — delete (`DeleteObject`)](#9-gesture-5--delete-deleteobject)
- [10. Gesture 6 — tab-cycle (`SelectNextObject` / `SelectPrevObject`)](#10-gesture-6--tab-cycle-selectnextobject--selectprevobject)
- [11. Master field-write matrix](#11-master-field-write-matrix)
- [12. Clamping and snapping: `KeepObjectLegal`](#12-clamping-and-snapping-keepobjectlegal)
- [13. `GetThisRoomsObjRects` — the geometry cache](#13-getthisroomsobjrects--the-geometry-cache)
- [14. `DrawThisRoomsObjects` and `HiliteAllObjects`](#14-drawthisroomsobjects-and-hiliteallobjects)
- [15. Undo: there is none](#15-undo-there-is-none)
- [16. The tools windoid](#16-the-tools-windoid)
- [17. `DrawClockDigit`](#17-drawclockdigit)
- [18. Mac Toolbox inventory and Go replacements](#18-mac-toolbox-inventory-and-go-replacements)
- [19. Corrections to `editor.md`](#19-corrections-to-editormd)
- [Open questions](#open-questions)
- [Porting notes](#porting-notes)

---

## 0. Scope, build flags, and what is compiled out

### 0.1 Two build flags gate this subsystem

```c
#define BUILD_ARCADE_VERSION    1
```
`GliderPRO/Headers/GliderDefines.h:16`. When defined non-zero the *arcade* keyboard arm is
compiled in `Events.c` and the *editor* keyboard arm is compiled out — see §7.1. This single
`#define` makes `MoveObject` (arrow-key nudge, 399 lines) unreachable in the shipped 1.0.4
binary.

`COMPILEDEMO` is *not* defined in the shipped build; the demo build compiles out most of the
editor. `ObjectEdit.c` is a patchwork of `#ifndef COMPILEDEMO` blocks. The exact map matters
because five functions are **not** guarded and therefore exist even in the demo build, where
their collaborators do not:

| Function | Lines | `#ifndef COMPILEDEMO`? |
|---|---|---|
| `FindObjectSelected` | `ObjectEdit.c:46`–`68` (guard `45`–`69`) | yes |
| `DoSelectionClick` | `ObjectEdit.c:73`–`138` (guard `75`–`137`) | yes |
| `DragHandle` | `ObjectEdit.c:143`–`345` (guard `142`–`346`) | yes |
| `DragObject` | `ObjectEdit.c:351`–`768` (guard `350`–`769`) | yes |
| `DoNewObjectClick` | `ObjectEdit.c:773`–`788` (guard `775`–`787`) | yes |
| `AddObjectPairing` | `ObjectEdit.c:792`–`1150` | **no** |
| `DeleteObject` | `ObjectEdit.c:1154`–`1187` (guard `1156`–`1186`) | yes |
| `DuplicateObject` | `ObjectEdit.c:1191`–`1370` | **no** |
| `MoveObject` | `ObjectEdit.c:1374`–`1772` (guard `1376`–`1771`) | yes |
| `DeselectObject` | `ObjectEdit.c:1776`–`1786` (guard `1778`–`1785`) | yes |
| `ObjectHasHandle` | `ObjectEdit.c:1791`–`1937` (guard `1790`–`1938`) | yes |
| `ObjectIsUpBlower` | `ObjectEdit.c:1942`–`1959` | **no** |
| `HandleBlowerGlider` | `ObjectEdit.c:1963`–`1978` (guard `1965`–`1977`) | yes |
| `SelectNextObject` | `ObjectEdit.c:1982`–`2011` (guard `1984`–`2010`) | yes |
| `SelectPrevObject` | `ObjectEdit.c:2015`–`2044` (guard `2017`–`2043`) | yes |
| `GetThisRoomsObjRects` | `ObjectEdit.c:2049`–`2341` (guard `2048`–`2342`) | yes |
| `DrawThisRoomsObjects` | `ObjectEdit.c:2347`–`2729` (guard `2346`–`2730`) | yes |
| `HiliteAllObjects` | `ObjectEdit.c:2734`–`2765` (guard `2736`–`2764`) | yes |
| `GoToObjectInRoom` | `ObjectEdit.c:2769`–`2799` | **no** |
| `GoToObjectInRoomNum` | `ObjectEdit.c:2803`–`2809` | **no** |

`GetThisRoomsObjRects` and `DrawThisRoomsObjects` being guarded is significant: in a
`COMPILEDEMO` build the geometry cache is never filled, so `GoToObjectInRoom` (unguarded)
would consult a stale/zero `roomObjectRects`. A Go port has no reason to reproduce the demo
build; the relevant takeaway is that `AddObjectPairing`, `DuplicateObject`, `ObjectIsUpBlower`,
`GoToObjectInRoom` and `GoToObjectInRoomNum` have **no `theMode` or `objActive` sanity guard of
their own** in any build (see §8.1 and §9.1).

### 0.2 Editor modes

```c
#define kSplashMode     0
#define kEditMode       1
#define kPlayMode       2
```
`GliderPRO/Headers/GliderDefines.h:191`–`193`. Note the values: **`kEditMode` is 1, not 0**, and
`0` is `kSplashMode` — so a port that zero-initialises a mode variable starts in *splash*, not
edit. There is **no `kPauseMode`**; the identifier does not appear anywhere in the source tree
(pausing is a separate global `Boolean paused`, `Input.c:34`, with its own spin loop at
`Input.c:96`–`103`, not a `theMode` value).

`theMode` is the global. Exactly **six** functions in `ObjectEdit.c` test it, all as an early-out:

| Function | Line | Test |
|---|---:|---|
| `DeleteObject` | `:1159` | `(theMode != kEditMode) \|\| (objActive == kNoObjectSelected)` |
| `MoveObject` | `:1382` | `theMode != kEditMode` |
| `DeselectObject` | `:1779` | `(theMode != kEditMode) \|\| (objActive == kNoObjectSelected)` |
| `SelectNextObject` | `:1988` | `(theMode != kEditMode) \|\| (thisRoom->numObjects <= 0)` |
| `SelectPrevObject` | `:2021` | `(theMode != kEditMode) \|\| (thisRoom->numObjects <= 0)` |
| `HiliteAllObjects` | `:2741` | `theMode != kEditMode` |

Plus three sites in `Tools.c` (`:499`, `:518`, `:534`). Every *other* entry point —
`DoSelectionClick`, `DragObject`, `DragHandle`, `DoNewObjectClick`, `DuplicateObject`,
`GetThisRoomsObjRects`, `DrawThisRoomsObjects`, `GoToObjectInRoom` — relies on the caller (the
event loop) having already established edit mode.

---

## 1. Cast of characters: globals, sentinels, and the geometry cache

### 1.1 File-scope globals of `ObjectEdit.c`

Declared at `GliderPRO/Sources/ObjectEdit.c:28`–`33`:

```c
Rect        roomObjectRects[kMaxRoomObs];                       // :28
Rect        initialGliderRect;                                  // :29
Rect        leftStartGliderSrc, rightStartGliderSrc;            // :30
Rect        leftStartGliderDest, rightStartGliderDest;          // :31
short       objActive;                                          // :32
Boolean     isFirstRoom;                                        // :33
```

| Global | Type / size | Line | Meaning |
|---|---|---:|---|
| `roomObjectRects` | `Rect[24]` = 24 × 8 = **192 bytes** | `:28` | Cache of every object's local-room screen rect, rebuilt by `GetThisRoomsObjRects` (§13). This is the *only* structure the editor hit-tests against; the on-disk `objectType` is never hit-tested directly. |
| `initialGliderRect` | `Rect` (8 bytes) | `:29` | Hit box for the house-wide initial glider start marker. |
| `leftStartGliderSrc` | `Rect` | `:30` | Source rect in `gliderSrcMap` for the left entry marker's ghost image. |
| `rightStartGliderSrc` | `Rect` | `:30` | Source rect for the right entry marker's ghost image. |
| `leftStartGliderDest` | `Rect` | `:31` | Hit box **and** blit destination for the room's left-side glider entry marker. |
| `rightStartGliderDest` | `Rect` | `:31` | Hit box and blit destination for the room's right-side glider entry marker. |
| `objActive` | `short` | `:32` | The selection. Either a slot index `0..23` or one of four negative sentinels. |
| `isFirstRoom` | `Boolean` | `:33` | Recomputed at the top of `GetThisRoomsObjRects` (`:2054`) as `GetFirstRoomNumber() == thisRoomNumber`; gates whether the initial-glider marker exists in this room at all. |

`hasMirror`, often mistaken for an `ObjectEdit.c` global, actually lives in
`GliderPRO/Sources/Render.c:42` and is written only by `AddToMirrorRegion`
(`Render.c:760`) and `ZeroMirrorRegion` (`Render.c:770`) — never by the editor. See §13.6.

`objActive` and `roomObjectRects` are the two exported symbols
(`GliderPRO/Headers/ObjectEdit.h:14`–`15`):

```c
extern Rect         roomObjectRects[];
extern short        objActive;
```

### 1.2 The four selection sentinels

`GliderPRO/Headers/GliderDefines.h:525`–`530`:

| Constant | Value | Meaning when stored in `objActive` |
|---|---:|---|
| `kRoomIsEmpty` | `-1` | (room-level sentinel, same numeric value) |
| `kObjectIsEmpty` | `-1` | An empty object slot's `what` code |
| `kNoObjectSelected` | `-1` | Nothing selected |
| `kInitialGliderSelected` | `-2` | The house's initial-glider marker is selected |
| `kLeftGliderSelected` | `-3` | The room's left glider-start marker is selected |
| `kRightGliderSelected` | `-4` | The room's right glider-start marker is selected |

`kNoObjectSelected`, `kObjectIsEmpty` and `kRoomIsEmpty` are all `-1`. That collision is
load-bearing in two places and is a hazard in a third:

1. `StartMarquee` and `StartMarqueeHandled` both early-out on
   `objActive == kNoObjectSelected` (`GliderPRO/Sources/Marquee.c:59`–`60`,
   `GliderPRO/Sources/Marquee.c:81`–`82`), so "no selection" is automatically "no marquee".
2. `GetThisRoomsObjRects` uses `kObjectIsEmpty` as a `switch` arm inside its 24-slot loop
   (`case kObjectIsEmpty:` at `ObjectEdit.c:2079`), writing the `(−2, −2, −1, −1)` sentinel at
   `:2080`. It is not a `continue`/skip — every slot gets a rect written.
3. `SelectNextObject`/`SelectPrevObject` can *land* on `-1` and thereby silently drop the
   selection instead of wrapping — see §10.3.

### 1.3 Room and object geometry constants

`GliderPRO/Headers/GliderDefines.h:496`–`501`:

| Constant | Value (dec) | Value (hex) | Use |
|---|---:|---:|---|
| `kNumTiles` | 8 | `0x08` | tiles per room |
| `kTileWide` | 64 | `0x40` | tile width in px |
| `kTileHigh` | 322 | `0x142` | room height in px, and the bottom clamp |
| `kRoomWide` | 512 | `0x200` | room width in px (`8 × 64`), and the right clamp |
| `kFloorSupportTall` | 44 | `0x2C` | 9-room-mode floor beam height |
| `kVertLocalOffset` | 322 | `0x142` | vertical offset between stacked rooms |

`kMaxRoomObs` = **24** — the object slot count per room. Glider constants:
`kGliderWide` = 48, `kGliderHigh` = 20, `kHalfGliderWide` = 24, `kNumGliderSrcRects` = 31,
`kGliderStartsDown` = 32.

The editor's working room rect is therefore `(0, 0, 512, 322)`. Every containment clamp in
`KeepObjectLegal` measures against `kRoomWide` and `kTileHigh`.

### 1.4 The six handle directions

`GliderPRO/Headers/GliderDefines.h:210`–`215`:

| Constant | Value | Handle placement |
|---|---:|---|
| `kAbove` | 1 | centred on top edge, `dist` px **above** it |
| `kToRight` | 2 | centred on right edge, `dist` px to the **right** |
| `kBelow` | 3 | centred on bottom edge, `dist` px **below** |
| `kToLeft` | 4 | centred on left edge, `dist` px to the **left** |
| `kBottomCorner` | 5 | on the bottom-right corner (no `dist`) |
| `kTopCorner` | 6 | on the top-right corner (no `dist`) |

`0` is used as the "no handle" value by `MarqueeHasHandles` (`Marquee.c:417`).

### 1.5 The four arrow-key bump codes

`GliderPRO/Headers/GliderDefines.h:205`–`208`, in source order:

| Constant | Value |
|---|---:|
| `kBumpUp` | 1 |
| `kBumpDown` | 2 |
| `kBumpRight` | 3 |
| `kBumpLeft` | 4 |

Note these are **1-based**, so `0` is not a valid bump code — `MoveObject`'s `switch (whichWay)`
has no `default` arm, and a `0` would fall through leaving `deltaH`/`deltaV` uninitialised. Note
also that `kBumpRight` precedes `kBumpLeft`, unlike the handle directions in §1.4 where
`kToRight` (2) precedes `kBelow` (3) precedes `kToLeft` (4).

### 1.6 The `marquee` struct

`GliderPRO/Headers/Marquee.h:14`–`20`:

```c
typedef struct
{
    Pattern     pats[kNumMarqueePats];      /* kNumMarqueePats == 7 */
    Rect        bounds, handle;
    short       index, direction, dist;
    Boolean     active, paused, handled;
} marquee;
```

Layout on 68k/PPC Mac (big-endian, `Pattern` = 8 bytes, `Rect` = 8 bytes, `short` = 2,
`Boolean` = 1 byte, struct aligned to 2):

| Offset | Size | Field |
|---:|---:|---|
| 0 | 56 | `pats[7]` (7 × 8-byte `Pattern`) |
| 56 | 8 | `bounds` |
| 64 | 8 | `handle` |
| 72 | 2 | `index` |
| 74 | 2 | `direction` |
| 76 | 2 | `dist` |
| 78 | 1 | `active` |
| 79 | 1 | `paused` |
| 80 | 1 | `handled` |
| 81 | 1 | (pad) |
| — | **82** | total |

`kNumMarqueePats` = **7** (`GliderPRO/Headers/GliderDefines.h`). The 7 patterns are loaded from
resource `'PAT#'` **128**:

```c
#define kMarqueePatListID       128
#define kHandleSideLong         9
```
`GliderPRO/Sources/Marquee.c:15`–`16`.

`InitMarquee` (`GliderPRO/Sources/Marquee.c:499`–`510`) loads them with
`GetIndPattern(&theMarquee.pats[i], kMarqueePatListID, i + 1)` for `i` in `0..6` — note the
**1-based** resource index against the 0-based array index — then zeroes `index` and clears
`active`, `paused`, `handled`, and the file-scope `gliderMarqueeUp`.

### 1.7 The four cursors and the coordinate windoid

`Marquee.c:28` declares `extern Cursor handCursor, vertCursor, horiCursor, diagCursor;`. Cursor
selection per gesture:

| Gesture | Cursor | Set at |
|---|---|---|
| drag-move | `handCursor` | `Marquee.c:223` (`DragMarqueeRect`) |
| handle drag, `kAbove`/`kBelow` | `vertCursor` | `Marquee.c:264`–`265` |
| handle drag, `kToRight`/`kToLeft` | `horiCursor` | `Marquee.c:266`–`267` |
| corner drag | `diagCursor` | `Marquee.c:350` (`DragMarqueeCorner`) |
| marquee rubber-band | arrow (`InitCursor()`) | `Marquee.c:193` |

Three of the four drag routines restore the arrow with `InitCursor()` on exit (`Marquee.c:254`,
`:340`, `:402`). `DragOutMarqueeRect` does not — it calls `InitCursor()` on *entry* (`:193`) and
leaves the cursor alone on exit, so the arrow is simply never changed.

The Coordinates windoid is a 50 × 38 floating window
(`QSetRect(&coordWindowRect, 0, 0, 50, 38)`, `GliderPRO/Sources/Coordinates.c:125`) showing
three lines `h:`, `v:`, `d:` drawn at pen positions `(5,12)`, `(4,22)`, `(5,32)`
(`Coordinates.c:82`, `:93`, `:105`), the `d:` line in `blueColor` (`Coordinates.c:96`,
`:107`). Its update protocol has a sentinel convention that a port must copy exactly:

```c
void SetCoordinateHVD (short h, short v, short d)
{
    if (h != -2) coordH = h;
    if (v != -2) coordV = v;
    if (d != -2) coordD = d;
    UpdateCoordWindow();
}
```
`GliderPRO/Sources/Coordinates.c:29`–`40`. So **`-2` means "leave unchanged"** and **`-1` means
"display a dash"** (`Coordinates.c:75`–`81`, `:86`–`92`, `:98`–`104`). Call sites:

| Call | Meaning |
|---|---|
| `SetCoordinateHVD(bounds.left, bounds.top, -1)` — `Marquee.c:71` | `StartMarquee`: show position, blank the distance |
| `SetCoordinateHVD(bounds.left, bounds.top, dist)` — `Marquee.c:134` | `StartMarqueeHandled`: show position and handle distance |
| `SetCoordinateHVD(-1, -1, -1)` — `Marquee.c:156` | `StopMarquee`: blank all three |
| `SetCoordinateHVD(bounds.left, bounds.top, -2)` — `Marquee.c:248` | live during `DragMarqueeRect`: update h/v, **preserve** d |
| `DeltaCoordinateD(*dragged)` — `Marquee.c:291`, `:303`, `:315`, `:327` | live during `DragMarqueeHandle`: update d only |

`DragMarqueeCorner` does **not** update the coordinate windoid at all — a live corner drag
leaves stale numbers on screen until the marquee is re-armed.

---

## 2. Gesture → entry-point map

| # | User gesture | Entry point | Lines | Reachable in 1.0.4? |
|---:|---|---|---|---|
| 1 | mouse-down on an object, hold, drag | `DragObject(Point where)` | `ObjectEdit.c:351`–`768` | yes |
| 2 | mouse-down inside the armed marquee's handle, hold, drag | `DragHandle(Point where)` | `ObjectEdit.c:143`–`345` | yes |
| 3 | arrow key (optionally with Shift) | `MoveObject(short whichWay, Boolean shiftDown)` | `ObjectEdit.c:1374`–`1772` | **no** — see §7.1 |
| 4 | Edit ▸ Duplicate | `DuplicateObject(void)` | `ObjectEdit.c:1191`–`1370` | yes |
| 5 | Edit ▸ Clear / Cut, Delete key | `DeleteObject(void)` | `ObjectEdit.c:1154`–`1187` | yes |
| 6 | Tab / Shift-Tab | `SelectNextObject(void)` / `SelectPrevObject(void)` | `ObjectEdit.c:1982`–`2011`, `2015`–`2044` | yes |
| — | mouse-down on empty space (selection tool) | `DoSelectionClick(Point where, Boolean isDoubleClick)` | `ObjectEdit.c:73`–`138` | yes |
| — | mouse-down with a creation tool active | `DoNewObjectClick(Point where)` | `ObjectEdit.c:773`–`788` | yes |

`DoSelectionClick` is the dispatcher: gestures 1 and 2 are *not* called from the event loop,
they are called from inside `DoSelectionClick` (§3.2), each gated on `StillDown()` — so a single
press-drag-release performs select-then-drag in one motion (`ObjectEdit.c:82`, `:115`).

---

## 3. Selection: hit testing, sentinels, lifetime

### 3.1 `FindObjectSelected` — the hit test

`GliderPRO/Sources/ObjectEdit.c:46`–`68`.

```
FindObjectSelected(where) -> short:
 1. if PtInRect(where, &initialGliderRect)          -> return kInitialGliderSelected  (-2)
 2. if PtInRect(where, &leftStartGliderDest)        -> return kLeftGliderSelected     (-3)
 3. if PtInRect(where, &rightStartGliderDest)       -> return kRightGliderSelected    (-4)
 4. for (i = kMaxRoomObs - 1; i >= 0; i--)                   // 23 .. 0, BACK to FRONT reversed
        if (PtInRect(where, &roomObjectRects[i]))
            { found = i; break; }
 5. return found                                            // kNoObjectSelected (-1) if no hit
```

The three glider tests are a single `if / else if / else if` chain with early `return`s
(`:52`–`57`); `found` is pre-seeded to `kNoObjectSelected` at `:50` and only the object loop
(`:59`–`66`) can change it.

Four properties a port must preserve:

1. **The three glider markers win over every object**, in the fixed order
   initial → left → right. An object sitting under the initial-glider marker can never be
   clicked.
2. The object loop runs **downward**, so the **highest slot index wins**. Slot index is z-order
   (§13.1), so this is a topmost-first hit test.
3. The loop bound is **`kMaxRoomObs` (all 24 slots, `ObjectEdit.c:59`), *not*
   `thisRoom->numObjects`.** Every slot is hit-tested regardless of the object count, so an
   "orphaned" slot above `numObjects` (which `DeleteObject` can create, §9.1) is still clickable.
   There is no `what != kObjectIsEmpty` test in the loop either: correctness rests entirely on
   the cache's sentinel rects for empty slots being outside the room (§13.2, §19.1).
4. It hit-tests `roomObjectRects[i]`, i.e. the **cache**, not the object. If the cache is stale
   the hit test is wrong. §13.3 enumerates every point at which the cache is rebuilt.

### 3.2 `DoSelectionClick` — the dispatcher

`GliderPRO/Sources/ObjectEdit.c:73`–`138`. Reconstructed control flow with the original
variable names in parentheses:

```
DoSelectionClick(where, isDoubleClick):
 1.  StopMarquee()                                                      // :78, UNCONDITIONAL
 2.  if (PtInMarqueeHandle(where) && objActive != kNoObjectSelected):    // :80  HANDLE BRANCH
 2a.     if StillDown(): DragHandle(where)                              // :82-83
 2b.     re-arm (see below)                                             // :84-90
 3.  else:                                                              // :92  SELECT BRANCH
 3a.     objActive = FindObjectSelected(where)                          // :94  (only write)
 3b.     if (objActive == kNoObjectSelected):
             if isDoubleClick: DoRoomInfo()                             // :98
 3c.     else if isDoubleClick:
             DoObjectInfo()                                             // :104
             re-arm, plain variant (see below)                          // :105-111
 3d.     else:
             if StillDown(): DragObject(where)                          // :115-116
             re-arm, glider-aware variant (see below)                   // :117-131
 4.  UpdateMenus(false)                                                 // :136
```

The three re-arm tails are *not* identical:

| Site | Lines | Handle case | No-handle case |
|---|---|---|---|
| 2b (handle branch) | `:84`–`90` | `StartMarqueeHandled(&roomObjectRects[objActive], direction, dist)` + `HandleBlowerGlider()` | `StartMarquee(&roomObjectRects[objActive])` |
| 3c (double click) | `:105`–`111` | same | `StartMarquee(&roomObjectRects[objActive])` — **no glider special-casing** |
| 3d (single click) | `:117`–`131` | same | four-way: `initialGliderRect` (`:125`) / `leftStartGliderDest` (`:127`) / `rightStartGliderDest` (`:129`) / `roomObjectRects[objActive]` (`:131`) |

**A drag starts on the *first* mouse-down, gated by `StillDown()`.** Step 3a selects the object,
and if the button is still held when control returns, step 3d immediately enters `DragObject`'s
blocking tracking loop. `DoSelectionClick` is therefore a press-drag-release gesture, exactly the
modern idiom — not a select-then-drag two-click gesture. `StillDown()` is the *only* thing
separating a click from a drag: a fast click (button already up) selects and arms the marquee and
nothing more.

Two consequences a port should note:

* **The handle branch (step 2) does not re-run the hit test.** Its gate is "the click landed in the
  currently armed marquee's handle rect and *something* is selected", not "the click landed on the
  selected object". Because `StopMarquee()` at `:78` runs *before* the test, `PtInMarqueeHandle`
  reads `theMarquee.handle` as left by the previous arm — `StopMarquee` (`Marquee.c:139`–`157`)
  XORs the overlay off and sets `active = false`, but never touches `bounds`, `handle`,
  `direction` or `dist` (§4.1).
* **Both `:86`/`:90` and `:107`/`:111` index `roomObjectRects[objActive]` without re-checking for a
  negative sentinel.** With `objActive` = −2/−3/−4 (a glider marker) these read out of bounds. In
  practice `ObjectHasHandle` returns `false` for `kInitialGliderSelected` (`:1793`–`1795`) but *not*
  for `kLeftGliderSelected`/`kRightGliderSelected`, which fall through to
  `switch (thisRoom->objects[objActive].what)` at `:1797` — itself an out-of-bounds read. A port
  must guard the sentinels explicitly.

`objActive` is written at exactly one place in this function, `ObjectEdit.c:94`.

### 3.3 Every write to `objActive` in the whole program

Verified by grepping all 67 converted `Sources/*.c`:

| Site | What it writes | Context |
|---|---|---|
| `GliderPRO/Sources/HouseIO.c:436` | `kNoObjectSelected` | after loading a house |
| `GliderPRO/Sources/HouseLegal.c:936` | `kNoObjectSelected` | house-legality repair pass |
| `GliderPRO/Sources/HouseLegal.c:1216` | `kNoObjectSelected` | house-legality repair pass |
| `GliderPRO/Sources/ObjectAdd.c:57` | the new slot index | `AddNewObject`, **before** the per-type cap checks |
| `GliderPRO/Sources/ObjectEdit.c:94` | `clickedObject` | `DoSelectionClick` |
| `GliderPRO/Sources/ObjectEdit.c:779` | `kNoObjectSelected` | `DoNewObjectClick`, before `AddNewObject` |
| `GliderPRO/Sources/ObjectEdit.c:1782` | `kNoObjectSelected` | `DeselectObject` |
| `GliderPRO/Sources/ObjectEdit.c:1997` | `objActive + 1` (loop) | `SelectNextObject` |
| `GliderPRO/Sources/ObjectEdit.c:2030` | `objActive - 1` (loop) | `SelectPrevObject` |
| `GliderPRO/Sources/ObjectEdit.c:2787` | the requested index | `GoToObjectInRoom` |

There are **ten** write sites and no others. In particular **no rendering function ever writes
`objActive`**, which answers the selection-survival question directly (§3.4).

### 3.4 Does selection survive a room change or a `DrawLocale` redraw?

**A pure redraw: yes, always.** `objActive` is referenced only in `ObjectEdit.c`,
`ObjectInfo.c`, `Marquee.c`, `Link.c`, `ObjectAdd.c`, `HouseIO.c`, `HouseLegal.c`, and (reads
only) `Menu.c`. `DrawLocale`, `RoomGraphics.c`, `ReflectCurrentRoom`, `Room.c` and
`MainWindow.c` never touch it. A `DrawLocale` / `ReadyBackground` / `DrawThisRoomsObjects`
redraw therefore leaves `objActive` intact.

But a redraw *does* destroy the marquee's pixels, because the marquee is an XOR overlay drawn
directly into the window port (§4.1), not part of the composited room. The editor's discipline
is:

```
PauseMarquee()            /* Marquee.c:161-168  : sets paused = true, then StopMarquee() */
... redraw ...
ResumeMarquee()           /* Marquee.c:172-184  : re-arms from the saved bounds/dir/dist */
```

`ResumeMarquee` early-outs unless `paused` (`Marquee.c:174`–`175`), then branches on the saved
`handled` flag: if handled it calls `StartMarqueeHandled(&theMarquee.bounds,
theMarquee.direction, theMarquee.dist)` **and** `HandleBlowerGlider()`; otherwise plain
`StartMarquee(&theMarquee.bounds)`. Note it resumes from `theMarquee.bounds`, **not** from
`roomObjectRects[objActive]` — so if the object moved while paused, the resumed marquee is in
the old place. Every gesture that moves an object therefore re-arms explicitly rather than
relying on `Resume` (§5.6, §7.6, §8.4).

Also note `ResumeMarquee` never clears `paused`. `paused` is only cleared inside `StartMarquee`
(`Marquee.c:65`) / `StartMarqueeHandled` (`Marquee.c:87`) — which `ResumeMarquee` calls, so it
does get cleared, but only via those. If `objActive == kNoObjectSelected` those two functions
early-out *before* clearing `paused` (`Marquee.c:59`, `:81`), leaving `paused == true` and
`active == false` forever. That state is benign (`DoMarquee` early-outs on either) but it means
a subsequent `PauseMarquee` is a no-op and a subsequent `ResumeMarquee` will try to re-arm a
stale rect.

**A room change: usually yes, deliberately.** Selection is cleared by an explicit
`DeselectObject()` call. All **19** call sites:

| Site | Situation |
|---|---|
| `GliderPRO/Sources/Events.c:275` | keyboard Escape / mode change |
| `GliderPRO/Sources/House.c:732` | house-level operation |
| `GliderPRO/Sources/Map.c:610` | map window click |
| `GliderPRO/Sources/Map.c:636` | map window click |
| `GliderPRO/Sources/Map.c:652` | map window click |
| `GliderPRO/Sources/Menu.c:389` | menu command |
| `GliderPRO/Sources/Menu.c:453` | menu command |
| `GliderPRO/Sources/ObjectEdit.c:1183` | `DeleteObject` |
| `GliderPRO/Sources/ObjectEdit.c:2779` | `GoToObjectInRoom`, room differs |
| `GliderPRO/Sources/ObjectEdit.c:2783` | `GoToObjectInRoom`, room differs |
| `GliderPRO/Sources/ObjectInfo.c:1376` | info dialog OK |
| `GliderPRO/Sources/ObjectInfo.c:1532` | info dialog OK |
| `GliderPRO/Sources/ObjectInfo.c:2168` | info dialog OK |
| `GliderPRO/Sources/Objects.c:992` | object list rebuild |
| `GliderPRO/Sources/Room.c:455` | `DeleteRoom` |
| `GliderPRO/Sources/Room.c:553` | room navigation |
| `GliderPRO/Sources/Scrap.c:124` | scrap operation |
| `GliderPRO/Sources/Scrap.c:191` | scrap operation |
| `GliderPRO/Sources/Scrap.c:454` | scrap operation |

`DeselectObject` itself (`ObjectEdit.c:1776`–`1786`):

```
DeselectObject():
 1. StopMarquee()
 2. objActive = kNoObjectSelected
 3. UpdateMenus(false)
```

Exactly **two** room-transition paths omit it:

1. `GliderPRO/Sources/MainWindow.c:207`–`209` — entering the editor. Whatever `objActive` held
   from a previous editing session is still there, and `roomObjectRects` has not been rebuilt
   yet. This is why `MainWindow.c` calls `GetThisRoomsObjRects()` immediately.
2. `GliderPRO/Sources/Room.c:666` — `SetToNearestNeighborRoom`'s spiral walk. Reached only from
   `DeleteRoom`, which already called `DeselectObject()` at `Room.c:455`, so it is harmless in
   practice.

**Rule for a Go port:** treat "selection is cleared on room change" as the invariant, and model
`objActive` as a `(roomIndex, slotIndex)` pair rather than a bare slot index. The original gets
away with a bare index only because of the 19 explicit deselects.

### 3.5 `ObjectIsUpBlower` and `HandleBlowerGlider` — the phantom glider overlay

`ObjectIsUpBlower` (`GliderPRO/Sources/ObjectEdit.c:1942`–`1959`) returns `true` for the
up-blowing blower types, i.e. the ones whose `distance` handle points *up* and which will lift a
glider. `HandleBlowerGlider` (`ObjectEdit.c:1963`–`1978`) then draws a ghost glider at the top
of the handle's reach:

```
HandleBlowerGlider():
 1. if not ObjectIsUpBlower(thisRoom->objects[objActive].what): return
 2. compute the glider centre from the marquee bounds + the handle distance
 3. SetMarqueeGliderRect(h, v)
```

`SetMarqueeGliderRect` (`GliderPRO/Sources/Marquee.c:443`–`451`):

```
SetMarqueeGliderRect(h, v):
 1. marqueeGliderRect = leftStartGliderSrc
 2. ZeroRectCorner(&marqueeGliderRect)                       /* -> (0,0,w,h) */
 3. QOffsetRect(&marqueeGliderRect, h - kHalfGliderWide, v - kGliderHigh)
 4. DrawGliderMarquee()
 5. gliderMarqueeUp = true
```

So the ghost is placed with its **horizontal centre** at `h` (`kHalfGliderWide` = 24) and its
**bottom** at `v` (`kGliderHigh` = 20). `DrawGliderMarquee` (`Marquee.c:432`–`439`) is a single
`CopyBits` of the *glider mask* in `srcXor` mode:

```c
CopyBits((BitMap *)*GetGWorldPixMap(blowerMaskMap),
         GetPortBitMapForCopyBits(GetWindowPort(mainWindow)),
         &leftStartGliderSrc, &marqueeGliderRect, srcXor, nil);
```

Because it is `srcXor`, drawing it twice erases it — which is exactly what `StopMarquee` relies
on (`Marquee.c:141`–`145`) and what `DrawMarquee`'s tail does (`Marquee.c:493`–`494`).

---

## 4. Handle geometry

### 4.1 The marquee is an XOR overlay in the window port

`DoMarquee` (`GliderPRO/Sources/Marquee.c:35`–`50`) is called from the event loop's idle path
and animates the marching ants:

```
DoMarquee():
 1. if (!theMarquee.active) or (theMarquee.paused): return
 2. SetPortWindowPort(mainWindow)
 3. PenMode(patXor)
 4. PenPat(&theMarquee.pats[theMarquee.index]);  DrawMarquee()     /* erase old */
 5. theMarquee.index++;  if (index >= kNumMarqueePats /*7*/) index = 0
 6. PenPat(&theMarquee.pats[theMarquee.index]);  DrawMarquee()     /* draw new */
 7. PenNormal()
```

The erase-then-draw pair inside one call is the classic QuickDraw XOR trick: because the pen
mode is `patXor`, re-stroking with the *same* pattern removes it. Step 4 removes the previous
frame and step 6 lays down the next. The 7 patterns in `'PAT#'` 128 are the phase-shifted
diagonal stripes; cycling `index` 0→6→0 produces the animation.

Consequences a Go port must handle:

* The marquee lives in the **window**, above the composited room; nothing in the room backing
  store knows about it.
* Any redraw of the window that is not preceded by `PauseMarquee()` will leave XOR residue.
* **Three of the four** drag routines begin by `StopMarquee()` (`Marquee.c:224` in
  `DragMarqueeRect`, `:268` in `DragMarqueeHandle`, `:351` in `DragMarqueeCorner`) so that the
  animated ants do not fight the live drag outline, then re-stroke a plain `FrameRect` in
  `patXor` with the *current* pattern for the drag feedback. `DragOutMarqueeRect` (`:188`) does
  **not** — it runs before any object exists, so there is no marquee to stop.
* A modern port should draw the selection outline as a normal overlay layer each frame and drop
  the XOR machinery entirely — but must keep the *phase* animation if it wants the same look
  (7 phases, advanced once per event-loop idle tick).

### 4.2 `StartMarqueeHandled` — where the handle rect comes from

`GliderPRO/Sources/Marquee.c:76`–`135`. The handle is a **9 × 9** square
(`kHandleSideLong` = 9, `Marquee.c:16`) whose *centre* is placed at a direction-dependent
anchor point.

```
StartMarqueeHandled(theRect, direction, dist):
 1.  if theMarquee.active: StopMarquee()
 2.  if objActive == kNoObjectSelected: return
 3.  SetPortWindowPort(mainWindow)
 4.  theMarquee.bounds = *theRect
 5.  active = true; paused = false; handled = true
 6.  QSetRect(&theMarquee.handle, 0, 0, 9, 9)
 7.  QOffsetRect(&theMarquee.handle, -4, -4)          /* 9/-2 == -4 in C */
 8.  switch (direction) { ... see table ... }
 9.  theMarquee.direction = direction; theMarquee.dist = dist
10.  PenMode(patXor); PenPat(&pats[index]); DrawMarquee(); PenNormal()
11.  SetCoordinateHVD(bounds.left, bounds.top, dist)
```

**Step 7 is the subtle one.** `kHandleSideLong / -2` in C is `9 / -2`, and C99 integer division
truncates toward zero, so this is **`-4`, not `-5`**. After step 7 the handle rect is
`(-4, -4, 5, 5)` — a 9 × 9 box whose notional centre pixel is at `(0, 0)` but which extends 4 px
left/up and 5 px right/down. That asymmetry is baked into every hit test.

> **Verification.** `9 / -2` in C: the 68k/PPC C compilers of the era (MPW C, Metrowerks) and
> every C89-and-later implementation truncate toward zero for `/` when either operand is
> negative, giving `-4`. (C89 left this implementation-defined; both Mac compilers truncated.)
> A Go port gets `-4` for free: Go's `/` on integers also truncates toward zero. But a port that
> naively writes `-kHandleSideLong/2` in a language with floor division (Python `//`) would get
> `-5` and shift every handle by one pixel.

Per-direction offsets applied in step 8, starting from the handle rect `(-4,-4,5,5)`:

| `direction` | Lines | First offset (to a bounds corner) | Second offset | Resulting handle centre |
|---|---|---|---|---|
| `kAbove` (1) | `Marquee.c:93`–`97` | `(bounds.left, bounds.top)` | `(HalfRectWide(bounds), -dist)` | `(left + w/2, top - dist)` |
| `kToRight` (2) | `Marquee.c:99`–`103` | `(bounds.right, bounds.top)` | `(dist, HalfRectTall(bounds))` | `(right + dist, top + h/2)` |
| `kBelow` (3) | `Marquee.c:105`–`109` | `(bounds.left, bounds.bottom)` | `(HalfRectWide(bounds), dist)` | `(left + w/2, bottom + dist)` |
| `kToLeft` (4) | `Marquee.c:111`–`115` | `(bounds.left, bounds.top)` | `(-dist, HalfRectTall(bounds))` | `(left - dist, top + h/2)` |
| `kBottomCorner` (5) | `Marquee.c:117`–`120` | `(bounds.right, bounds.bottom)` | — | `(right, bottom)` |
| `kTopCorner` (6) | `Marquee.c:122`–`125` | `(bounds.right, bounds.top)` | — | `(right, top)` |

`HalfRectWide` / `HalfRectTall` (`GliderPRO/Sources/RectUtils.c:79`–`82`, `:87`–`90`):

```c
short HalfRectWide (Rect *theRect) { return ((theRect->right - theRect->left) / 2); }
short HalfRectTall (Rect *theRect) { return ((theRect->bottom - theRect->top) / 2); }
```

Both truncate. For an odd-width object the `kAbove`/`kBelow` handle therefore sits half a pixel
left of true centre.

There is **no `default:` arm** in the step-8 switch. If `direction` were 0 or > 6 the handle
would be left at `(-4,-4,5,5)` — a 9 × 9 box at the window's top-left corner. §4.5 shows this is
unreachable with shipped data.

### 4.3 The exact hit box, in absolute numbers

Combining §4.2, the handle rect for each direction is:

| `direction` | Handle rect (left, top, right, bottom) |
|---|---|
| `kAbove` | `(L + W/2 - 4, T - dist - 4, L + W/2 + 5, T - dist + 5)` |
| `kToRight` | `(R + dist - 4, T + H/2 - 4, R + dist + 5, T + H/2 + 5)` |
| `kBelow` | `(L + W/2 - 4, B + dist - 4, L + W/2 + 5, B + dist + 5)` |
| `kToLeft` | `(L - dist - 4, T + H/2 - 4, L - dist + 5, T + H/2 + 5)` |
| `kBottomCorner` | `(R - 4, B - 4, R + 5, B + 5)` |
| `kTopCorner` | `(R - 4, T - 4, R + 5, T + 5)` |

where `L`, `T`, `R`, `B` = `theMarquee.bounds` and `W = R - L`, `H = B - T`.

The hit test is `PtInMarqueeHandle` (`GliderPRO/Sources/Marquee.c:425`–`428`):

```c
Boolean PtInMarqueeHandle (Point where) { return (PtInRect(where, &theMarquee.handle)); }
```

QuickDraw's `PtInRect` is **left-inclusive, right-exclusive** (`left <= h < right` and
`top <= v < bottom`), so the live handle is the 9 columns `L+W/2-4 .. L+W/2+4` and the 9 rows
`T-dist-4 .. T-dist+4` for `kAbove`. A Go port must use half-open bounds to match; using a
closed test makes the handle 10 × 10 and steals a pixel row/column from the object's own drag
area.

**Handle vs. body precedence.** `DoSelectionClick` tests the handle *first*
(`ObjectEdit.c` step 2b of §3.2). When the handle overlaps the object body — which happens
whenever `dist` is small, e.g. `dist == 0` puts the `kAbove` handle straddling the top edge with
4 rows inside the object — the handle wins. There is no priority tie-break beyond that ordering.

### 4.4 The handle's leader line

`DrawMarquee` (`GliderPRO/Sources/Marquee.c:455`–`495`) draws, in order:

```
DrawMarquee():
 1. FrameRect(&theMarquee.bounds)
 2. if theMarquee.handled:
 2a.    PaintRect(&theMarquee.handle)                  /* solid 9x9, patXor */
 2b.    switch (direction):
            kAbove:   MoveTo(handle.left + 4, handle.bottom); LineTo(handle.left + 4, bounds.top - 1)
            kToRight: MoveTo(handle.left,     handle.top + 4); LineTo(bounds.right,   handle.top + 4)
            kBelow:   MoveTo(handle.left + 4, handle.top - 1); LineTo(handle.left + 4, bounds.bottom)
            kToLeft:  MoveTo(handle.right,    handle.top + 4); LineTo(bounds.left,     handle.top + 4)
 3. if gliderMarqueeUp: DrawGliderMarquee()
```

`kHandleSideLong / 2` is `9 / 2` = **4**, so the leader line runs down the handle's 5th
column/row (0-based index 4), one pixel left/up of geometric centre. `kBottomCorner` and
`kTopCorner` get **no leader line** — they are corner grips flush with the marquee.

Note the `-1` asymmetries: `kAbove` draws from `handle.bottom` to `bounds.top - 1`, and `kBelow`
from `handle.top - 1` to `bounds.bottom`. In QuickDraw `LineTo` the end point *is* drawn, so
these produce a line that touches but does not overlap the marquee frame.

### 4.5 `ObjectHasHandle` — which 37 types have a handle, and where

`GliderPRO/Sources/ObjectEdit.c:1791`–`1937`. Signature:

```c
Boolean ObjectHasHandle (short *direction, short *dist)
```

It switches on `thisRoom->objects[objActive].what` and, for the types that have a handle, sets
`*direction` and `*dist` from the object's own fields, returning `true`. For everything else it
returns `false`. The label set is **exactly 37 types**, and it is **identical** to `DragHandle`'s
label set (verified programmatically: symmetric difference empty).

| Group | `what` codes | `direction` | `dist` source |
|---|---|---|---|
| 12 simple blowers | `kFloorVent` 0x01, `kCeilingVent` 0x02, `kFloorBlower` 0x03, `kCeilingBlower` 0x04, `kSewerGrate` 0x05, `kTaper` 0x08, `kCandle` 0x09, `kStubby` 0x0A, `kTiki` 0x0B, `kBBQ` 0x0C, `kGrecoVent` 0x0E, `kSewerBlower` 0x0F | per-type: `kAbove` for up-blowers, `kBelow` for down-blowers | `data.a.distance` |
| `kLeftFan` 0x06 | | `kToLeft` | `data.a.distance` |
| `kRightFan` 0x07 | | `kToRight` | `data.a.distance` |
| `kInvisBlower` 0x0D | | derived from `data.a.vector & 0x0F` | `data.a.distance` |
| `kLiftArea` 0x10 | | `kBottomCorner` | `data.a.distance` (h) + `data.a.tall` (v) |
| 3 tables/shelves | `kTable` 0x11, `kShelf` 0x12, `kDeckTable` 0x19 | `kToRight` | width of `data.b.bounds` |
| 3 boxes | `kCabinet` 0x13, `kInvisObstacle` 0x1C, `kInvisBounce` 0x1F | `kBottomCorner` | width/height of `data.b.bounds` |
| 2 tall units | `kCounter` 0x17, `kDresser` 0x18 | `kTopCorner` | width/height of `data.b.bounds` |
| 2 greases + slider | `kGreaseRt` 0x28, `kGreaseLf` 0x29, `kSlider` 0x2F | `kToRight` (`kGreaseLf`: `kToLeft`) | `data.c.length` |
| `kInvisTrans` 0x3F | | `kBottomCorner` | `data.d.wide`, `data.d.tall` |
| `kDeluxeTrans` 0x40 | | `kBottomCorner` | unpacked from `data.d.tall` |
| 2 strip lights | `kFlourescent` 0x56, `kTrackLight` 0x57 | `kToRight` | `data.f.length` |
| `kToaster` 0x62 | | `kAbove` | `data.g.height` |
| 3 enemies | `kBall` 0x76, `kDrip` 0x77, `kFish` 0x78 | `kAbove` (`kDrip`: `kBelow`) | `data.h.length` |
| 2 wall panels | `kMirror` 0x82, `kWallWindow` 0x86 | `kBottomCorner` | width/height of `data.i.bounds` |

Count: 12 + 1 + 1 + 1 + 1 + 3 + 3 + 2 + 3 + 1 + 1 + 2 + 1 + 3 + 2 = **37**.

**The `kInvisBlower` `vector` decode is the one dangerous arm.** `ObjectEdit.c` derives the
direction from the low nibble of `data.a.vector`:

| `vector & 0x0F` | `direction` |
|---:|---|
| 1 | `kAbove` |
| 2 | `kToRight` |
| 4 | `kBelow` |
| 8 | `kToLeft` |

There is **no `default:`**, so any other low nibble leaves `*direction` **uninitialised** (it is a
stack local in the caller). `StartMarqueeHandled`'s step-8 switch also has no default, so the
handle would end up at `(-4,-4,5,5)`.

> **Empirical check.** Across all 22 shipped houses the observed `data.a.vector` values are:
> `kInvisBlower` — `0x01`: 1629, `0x02`: 241, `0x04`: 241, `0x08`: 221, `0x11`: 3, `0x14`: 1;
> `kLiftArea` — `0x01`: 418, `0x02`: 138, `0x04`: 45, `0x08`: 115. **Every** low nibble is in
> {1, 2, 4, 8}. The hazard is therefore unreachable with shipped data, but a hand-edited or
> fuzzed house could trigger it. A Go port should give the switch an explicit default
> (`kAbove`, matching the most common value) and log.

### 4.6 `MarqueeHasHandles`

`GliderPRO/Sources/Marquee.c:407`–`421`. A read-back accessor with **no callers at all** — grep the
whole tree and the only hits are the definition (`Marquee.c:407`) and the prototype
(`GliderProtos.h:179`). It is dead code; the place a port might expect it, `Coordinates.c:159`, calls
`ObjectHasHandle` instead. Documented here only because its `-1`/`0` conventions differ from
`ObjectHasHandle`'s and a porter reading the header will otherwise assume they are interchangeable:

```c
Boolean MarqueeHasHandles (short *direction, short *dist)
{
    if (theMarquee.handled) { *direction = theMarquee.direction; *dist = theMarquee.dist; return true; }
    else                    { *direction = 0; *dist = 0;                                 return false; }
}
```

Note this reads the *marquee's* cached direction/dist, whereas `ObjectHasHandle` recomputes from
the object. They can disagree mid-gesture: `DragHandle` mutates the object's field but does not
update `theMarquee.dist`, so between the commit and the next `StartMarqueeHandled` the two
disagree. §6.4 shows how each `DragHandle` arm re-arms. Because nothing calls
`MarqueeHasHandles`, that disagreement is never observed in 1.0.4 — a port can simply omit the
function.

### 4.7 The four drag primitives

All four live in `Marquee.c` and share a skeleton: set cursor → `StopMarquee()` → `patXor` pen →
draw initial feedback → poll `WaitMouseUp()` / `GetMouse()` / `DeltaPoint()` → erase-move-draw →
on release, erase feedback, `PenNormal()`, `InitCursor()`. `DragOutMarqueeRect` is the odd one out:
it calls `InitCursor()` *up front* instead of setting a drag cursor, skips `StopMarquee()`, and does
not call `InitCursor()` on release.

#### 4.7.1 `DragOutMarqueeRect(Point start, Rect *theRect)` — `Marquee.c:188`–`214`

Rubber-band a rect from a fixed anchor. Used for the "drag out a new object" flow, not for
manipulating an existing one.

```
 1. SetPortWindowPort(mainWindow); InitCursor()
 2. QSetRect(theRect, start.h, start.v, start.h, start.v)   /* degenerate */
 3. PenMode(patXor); PenPat(&pats[index]); FrameRect(theRect)
 4. wasPt = start
 5. while (WaitMouseUp()):
 5a.    GetMouse(&newPt)
 5b.    if DeltaPoint(wasPt, newPt):
            FrameRect(theRect)                                  /* erase */
            QSetRect(theRect, start.h, start.v, newPt.h, newPt.v)
            NormalizeRect(theRect)                              /* RectUtils.c:34-51 */
            FrameRect(theRect)                                  /* draw   */
            wasPt = newPt
 6. FrameRect(theRect)     /* final erase */
 7. PenNormal()
```

`NormalizeRect` (`GliderPRO/Sources/RectUtils.c:34`–`51`) swaps `left`/`right` and `top`/`bottom`
if inverted, so dragging up-left works.

#### 4.7.2 `DragMarqueeRect(Point start, Rect *theRect, Boolean lockH, Boolean lockV)` — `Marquee.c:218`–`255`

Translate a rect under the mouse, optionally axis-locked. **The flag names are inverted from
what you would expect:**

```c
if (lockV) deltaH = 0; else deltaH = newPt.h - wasPt.h;      /* Marquee.c:236-239 */
if (lockH) deltaV = 0; else deltaV = newPt.v - wasPt.v;      /* Marquee.c:240-243 */
```

So **`lockV == true` kills horizontal motion** and **`lockH == true` kills vertical motion**.
Read them as "lock to the V axis" / "lock to the H axis". Getting this backwards in a port
inverts the movement constraints of the **34** axis-locked object types — arm A (29 types,
`(true, false)`), arm B (2, `(false, true)`) and arm D (3, `(true, false)`) of §5.3. The other 83
types pass `(false, false)` and are unaffected either way.

```
 1. SetCursor(&handCursor); StopMarquee()
 2. PenMode(patXor); PenPat(&pats[index])
 3. theMarquee.bounds = *theRect; FrameRect(&theMarquee.bounds)
 4. wasPt = start
 5. while (WaitMouseUp()):
 5a.    GetMouse(&newPt)
 5b.    if DeltaPoint(wasPt, newPt):
            deltaH = lockV ? 0 : newPt.h - wasPt.h
            deltaV = lockH ? 0 : newPt.v - wasPt.v
            FrameRect(&theMarquee.bounds)                       /* erase */
            QOffsetRect(&theMarquee.bounds, deltaH, deltaV)
            FrameRect(&theMarquee.bounds)                       /* draw  */
            wasPt = newPt
            SetCoordinateHVD(bounds.left, bounds.top, -2)
 6. FrameRect(&theMarquee.bounds)   /* final erase */
 7. *theRect = theMarquee.bounds
 8. PenNormal(); InitCursor()
```

**There is no clamping inside the drag.** The outline can be dragged arbitrarily far outside the
room, off the window, even to negative coordinates; the drag returns the raw rect and the *caller*
computes deltas and lets `KeepObjectLegal` clean up (§5.5, §12). A Go port that clamps live will
not reproduce the original's behaviour of snapping back only on release.

#### 4.7.3 `DragMarqueeHandle(Point start, short *dragged)` — `Marquee.c:259`–`341`

Drag a one-dimensional handle, accumulating into `*dragged`. The direction comes from
`theMarquee.direction`, already set by `StartMarqueeHandled`.

```
 1. cursor = (direction == kAbove || direction == kBelow) ? vertCursor : horiCursor
 2. StopMarquee(); PenMode(patXor); PenPat(&pats[index])
 3. FrameRect(&theMarquee.bounds); PaintRect(&theMarquee.handle)
 4. wasPt = start
 5. while (WaitMouseUp()):
 5a.    GetMouse(&newPt)
 5b.    if DeltaPoint(wasPt, newPt):
            switch (direction):
              kAbove:   deltaH = 0; deltaV = newPt.v - wasPt.v; *dragged -= deltaV
              kToRight: deltaH = newPt.h - wasPt.h; deltaV = 0; *dragged += deltaH
              kBelow:   deltaH = 0; deltaV = newPt.v - wasPt.v; *dragged += deltaV
              kToLeft:  deltaH = newPt.h - wasPt.h; deltaV = 0; *dragged -= deltaH
            /* the clamp, per arm: */
            if (*dragged <= 0):
                kAbove:   deltaV += *dragged        /* NOTE the sign difference */
                kToRight: deltaH -= *dragged
                kBelow:   deltaV -= *dragged
                kToLeft:  deltaH += *dragged
                *dragged = 0
            DeltaCoordinateD(*dragged)
            PaintRect(&theMarquee.handle)                       /* erase */
            QOffsetRect(&theMarquee.handle, deltaH, deltaV)
            PaintRect(&theMarquee.handle)                       /* draw  */
            wasPt = newPt
 6. FrameRect(&theMarquee.bounds); PaintRect(&theMarquee.handle)     /* final erase */
 7. PenNormal(); InitCursor()
```

Three things to preserve exactly:

1. **`*dragged` is clamped at 0, never negative** (`Marquee.c:286`–`290`, `:298`–`302`,
   `:310`–`314`, `:322`–`326`). The clamp *also* corrects the pixel delta so the handle stops
   moving rather than overshooting and snapping back. The sign of the correction differs per arm
   (`+= *dragged` for `kAbove`/`kToLeft`, `-= *dragged` for `kToRight`/`kBelow`) because
   `*dragged` and the pixel delta have opposite senses for the "shrinking" directions.
2. **The marquee frame is not redrawn during the loop** — only the handle is. Compare
   `DragMarqueeCorner`, which redraws both. So while dragging a 1-D handle the object's outline
   stays put and only the handle slides. That is correct: a 1-D handle changes a *distance*, not
   the object's rect.
3. **There is no upper clamp.** `*dragged` can grow unboundedly; `KeepObjectLegal` applies the
   upper bounds after the drag (§12.3).
4. **The switch at `:280`–`:329` has only the four 1-D arms and no `default:`.** `deltaH` and
   `deltaV` are plain locals (`:262`), never pre-initialised, and are read unconditionally at `:332`.
   So calling `DragMarqueeHandle` with `theMarquee.direction` set to `kBottomCorner`, `kTopCorner`
   or 0 offsets the handle by **uninitialised stack garbage**. It is unreachable in practice —
   `DragHandle` routes the two corner directions to `DragMarqueeCorner` (§6.1) — but a Go port gets
   zero-valued deltas here for free and so cannot reproduce the fault even if it wanted to.

#### 4.7.4 `DragMarqueeCorner(Point start, short *hDragged, short *vDragged, Boolean isTop)` — `Marquee.c:345`–`403`

Drag a 2-D corner grip. `isTop == true` means the grip is the **top**-right corner, so vertical
motion is inverted (dragging up grows the object).

```
 1. SetCursor(&diagCursor); StopMarquee(); PenMode(patXor); PenPat(&pats[index])
 2. FrameRect(&theMarquee.bounds); PaintRect(&theMarquee.handle)
 3. wasPt = start
 4. while (WaitMouseUp()):
 4a.    GetMouse(&newPt)
 4b.    if DeltaPoint(wasPt, newPt):
            deltaH = newPt.h - wasPt.h
            deltaV = isTop ? (wasPt.v - newPt.v) : (newPt.v - wasPt.v)
            *hDragged += deltaH;  if (*hDragged <= 0) { deltaH -= *hDragged; *hDragged = 0; }
            *vDragged += deltaV;  if (*vDragged <= 0) { deltaV -= *vDragged; *vDragged = 0; }
            FrameRect(&theMarquee.bounds); PaintRect(&theMarquee.handle)     /* erase both */
            if isTop:
                QOffsetRect(&theMarquee.handle, deltaH, -deltaV)
                theMarquee.bounds.right += deltaH
                theMarquee.bounds.top   -= deltaV
            else:
                QOffsetRect(&theMarquee.handle, deltaH, deltaV)
                theMarquee.bounds.right  += deltaH
                theMarquee.bounds.bottom += deltaV
            FrameRect(&theMarquee.bounds); PaintRect(&theMarquee.handle)     /* draw both  */
            wasPt = newPt
 5. FrameRect(&theMarquee.bounds); PaintRect(&theMarquee.handle)
 6. PenNormal(); InitCursor()
```

Both accumulators are clamped at 0 with the same corrective-delta trick. Only `right` and
`top`/`bottom` move — `left` is the anchor in both modes. This is why `kCounter`/`kDresser`
(`kTopCorner`) grow *upward* from a fixed bottom, and `kCabinet`/`kInvisObstacle`/`kInvisBounce`
(`kBottomCorner`) grow *downward* from a fixed top.

**`DragMarqueeCorner` does not call `DeltaCoordinateD`** — the coordinate windoid shows stale
numbers throughout a corner drag (§1.7).

---

## 5. Gesture 1 — drag-move (`DragObject`)

`GliderPRO/Sources/ObjectEdit.c:351`–`768`. 418 lines, four phases:

1. **Drag phase** — pick the axis constraint, run `DragMarqueeRect`, get back `newRect`.
2. **Delta phase** — `deltaH = newRect.left - wasRect.left`, `deltaV = newRect.top - wasRect.top`.
3. **Commit phase** — a 16-arm switch that adds the deltas into the right union fields.
4. **Normalise + redraw phase** — `KeepObjectLegal()`, rebuild the cache, invalidate, recomposite.

Locals (`ObjectEdit.c:353`–`356`):

```c
Rect     newRect, wasRect;
short    deltaH, deltaV, increment;
char     wasState;
Boolean  invalAll;
```

**`invalAll` is uninitialised.** It is assigned only inside the constraint switch
(`ObjectEdit.c:412`, `:418`, `:439`, `:446`, `:516`), which is skipped entirely for the three
glider pseudo-selections (`ObjectEdit.c:358`–`375`). See §5.7.

### 5.1 The three glider pseudo-selections

`ObjectEdit.c:358`–`375`:

| `objActive` | `wasRect`/`newRect` source | `DragMarqueeRect` flags | Effect |
|---|---|---|---|
| `kInitialGliderSelected` (−2) | `initialGliderRect` | `(false, false)` | free 2-D drag |
| `kLeftGliderSelected` (−3) | `leftStartGliderDest` | `(false, true)` | `lockV = true` ⇒ **vertical only** |
| `kRightGliderSelected` (−4) | `rightStartGliderDest` | `(false, true)` | `lockV = true` ⇒ **vertical only** |

(Recall §4.7.2: `lockV` kills the *horizontal* delta.) The glider start markers can only slide up
and down the room edge, which is right — they are edge-entry heights, not free positions.

### 5.2 The commit for the glider pseudo-selections

`ObjectEdit.c:529`–`560`.

**Initial glider** (`ObjectEdit.c:531`–`535`) — writes the *house* header, not the room:

```c
wasState = HGetState((Handle)thisHouse);
HLock((Handle)thisHouse);
(*thisHouse)->initial.h += deltaH;
(*thisHouse)->initial.v += deltaV;
HSetState((Handle)thisHouse, wasState);
```

The `HGetState`/`HLock`/`HSetState` triple is Mac Memory Manager relocatable-handle discipline:
`thisHouse` is a `Handle` (a `**houseType`), so it must be locked before dereferencing twice
across a statement that might move memory. In Go this is just a pointer write. Note there is
**no clamp at all** on `initial.h` / `initial.v` here — the initial glider position can be dragged
to any value, including negative. (`HouseLegal.c` repairs it on the next load.)

**Left glider start** (`ObjectEdit.c:539`–`547`):

```
 1. increment = thisRoom->leftStart + deltaV
 2. if (increment > 255)  increment = 255
    else if (increment < 0) increment = 0
 3. thisRoom->leftStart = (Byte)increment
 4. QSetRect(&leftStartGliderDest, 0, 0, 48, 16)
 5. QOffsetRect(&leftStartGliderDest, 0, kGliderStartsDown + (short)thisRoom->leftStart)
```

`leftStart` is a `Byte` at `roomType` offset **30** — hence the `[0, 255]` clamp. `kGliderStartsDown`
= **32**, so the marker rect is `(0, 32 + leftStart, 48, 48 + leftStart)`.

**Right glider start** (`ObjectEdit.c:551`–`559`): identical, `rightStart` (`roomType` offset
**31**), *except*:

```c
QSetRect(&rightStartGliderDest, 0, 0, 48, 16);
QOffsetRect(&rightStartGliderDest, 0,                      /* <-- x offset is 0 */
        kGliderStartsDown + (short)thisRoom->rightStart);
```

**This is a bug.** The x offset should be `kRoomWide - 48` = **464**, as it is in the two other
places that build this rect:

| Site | x offset |
|---|---:|
| `GliderPRO/Sources/ObjectEdit.c:558` (`DragObject`) | **0** |
| `GliderPRO/Sources/ObjectEdit.c:1517` (`MoveObject`) | `kRoomWide - 48` = 464 |
| `GliderPRO/Sources/ObjectEdit.c:2066` (`GetThisRoomsObjRects`) | `kRoomWide - 48` = 464 |

So after dragging the right glider start marker, its hit box jumps to the **left** edge of the
room, on top of the left marker's hit box, until the next `GetThisRoomsObjRects()` — which,
happily, is called 192 lines later at `ObjectEdit.c:750`, *unconditionally*. So the corruption
lasts only for the duration of the `InvalWindowRect(mainWindow, &rightStartGliderDest)` at
`ObjectEdit.c:761`, which therefore invalidates the **wrong rectangle** (left edge instead of
right). Visible symptom: after dragging the right start marker, the marker's old position on the
right edge is not repainted until some other event dirties it. A Go port should just use 464 in
all three places.

### 5.3 The constraint switch — all 117 types

`ObjectEdit.c:380`–`518`. Five arms, no `default:`. Total label count **117**, exactly the full
set of valid `what` codes.

| Arm | Lines | Count | `DragMarqueeRect(lockH, lockV)` | Effective freedom | `invalAll` |
|---|---|---:|---|---|---|
| A | `:382`–`413` | 29 | `(true, false)` | **horizontal only** (`lockH` kills `deltaV`) | `false` |
| B | `:415`–`419` | 2 | `(false, true)` | **vertical only** | `false` |
| C | `:421`–`440` | 17 | `(false, false)` | free 2-D | **`true`** |
| D | `:442`–`447` | 3 | `(true, false)` | horizontal only | **`true`** |
| E | `:449`–`517` | 66 | `(false, false)` | free 2-D | `false` |

**Arm A — horizontal only, partial invalidate (29 types).** `kFloorVent` 0x01, `kCeilingVent`
0x02, `kFloorBlower` 0x03, `kCeilingBlower` 0x04, `kSewerGrate` 0x05, `kGrecoVent` 0x0E,
`kSewerBlower` 0x0F, `kManhole` 0x1D, `kUpStairs` 0x31, `kDownStairs` 0x32, `kCeilingLight` 0x51,
`kHipLamp` 0x54, `kDecoLamp` 0x55, `kFlourescent` 0x56, `kFloorTrans` 0x35, `kCeilingTrans` 0x36,
`kDoorInLf` 0x37, `kDoorInRt` 0x38, `kDoorExRt` 0x39, `kDoorExLf` 0x3A, `kWindowInLf` 0x3B,
`kWindowInRt` 0x3C, `kWindowExRt` 0x3D, `kWindowExLf` 0x3E, `kBalloon` 0x71, `kCopterLf` 0x72,
`kCopterRt` 0x73, `kMousehole` 0x83, `kFireplace` 0x84.

These are the types whose vertical position is fixed by design — floor/ceiling-mounted blowers
and vents, stairs, doors and windows (fixed `left` too, see §12.4), ceiling lights, horizontally
patrolling enemies, and wall clutter that hugs the floor.

**Arm B — vertical only (2 types).** `kDartLf` 0x74, `kDartRt` 0x75. Darts fly horizontally
across the room at a fixed height, so only the height is editable.

**Arm C — free 2-D, full-window invalidate (17 types).** `kTiki` 0x0B, `kTable` 0x11, `kShelf`
0x12, `kCabinet` 0x13, `kDeckTable` 0x19, `kStool` 0x1A, `kInvisObstacle` 0x1C, `kInvisBounce`
0x1F, `kGreaseRt` 0x28, `kGreaseLf` 0x29, `kSlider` 0x2F, `kMailboxLf` 0x33, `kMailboxRt` 0x34,
`kInvisTrans` 0x3F, `kDeluxeTrans` 0x40, `kMirror` 0x82, `kWallWindow` 0x86.

`invalAll = true` here because these objects have visual side effects that extend beyond their own
rect: the tiki draws a pole down to `kTikiPoleBase` = 300 (`GliderPRO/Sources/ObjectDraw.c:71`),
tables and shelves draw support legs, greases draw a spill trail, mirrors and wall windows
reflect/composite the whole room.

**Arm D — horizontal only, full-window invalidate (3 types).** `kCounter` 0x17, `kDresser` 0x18,
`kTrackLight` 0x57. Counters and dressers draw a counter-top shadow across the room; the track
light casts a light cone.

**Arm E — free 2-D, partial invalidate (66 types).** Everything else: the 12 clock/prize bonuses
(`kRedClock` 0x21, `kBlueClock` 0x22, `kYellowClock` 0x23, `kCuckoo` 0x24, `kPaper` 0x25,
`kBattery` 0x26, `kBands` 0x27, `kFoil` 0x2A, `kInvisBonus` 0x2B, `kStar` 0x2C, `kSparkle` 0x2D,
`kHelium` 0x2E), 8 blowers (`kLeftFan` 0x06, `kRightFan` 0x07, `kTaper` 0x08, `kCandle` 0x09,
`kStubby` 0x0A, `kBBQ` 0x0C, `kInvisBlower` 0x0D, `kLiftArea` 0x10), 5 furniture
(`kFilingCabinet` 0x14, `kWasteBasket` 0x15, `kMilkCrate` 0x16, `kTrunk` 0x1B, `kBooks` 0x1E),
all 9 switches (`kLightSwitch` 0x41 … `kSoundTrigger` 0x49), 3 lights (`kLightBulb` 0x52,
`kTableLamp` 0x53, `kInvisLight` 0x58), 14 appliances (`kShredder` 0x61, `kToaster` 0x62,
`kMacPlus` 0x63, `kGuitar` 0x64, `kTV` 0x65, `kCoffee` 0x66, `kOutlet` 0x67, `kVCR` 0x68,
`kStereo` 0x69, `kMicrowave` 0x6A, `kCinderBlock` 0x6B, `kFlowerBox` 0x6C, `kCDs` 0x6D,
`kCustomPict` 0x6E), 4 enemies (`kBall` 0x76, `kDrip` 0x77, `kFish` 0x78, `kCobweb` 0x79), and 11
clutter (`kOzma` 0x81, `kFlower` 0x85, `kBear` 0x87, `kCalendar` 0x88, `kVase1` 0x89, `kVase2`
0x8A, `kBulletin` 0x8B, `kCloud` 0x8C, `kFaucet` 0x8D, `kRug` 0x8E, `kChimes` 0x8F).

29 + 2 + 17 + 3 + 66 = **117**. ✅

### 5.4 The commit switch — exactly which fields get the deltas

`ObjectEdit.c:563`–`744`. Sixteen arms, no `default:`, **117** labels total. Every write is
`+= deltaH` or `+= deltaV` — never an assignment, so the object's existing coordinate is
*translated*, not replaced.

| # | Lines | `what` codes | Count | Fields written |
|---:|---|---|---:|---|
| 1 | `:565`–`573` | `kFloorVent`, `kCeilingVent`, `kFloorBlower`, `kCeilingBlower`, `kSewerGrate`, `kGrecoVent`, `kSewerBlower` | 7 | `data.a.topLeft.h += deltaH` |
| 2 | `:575`–`586` | `kLeftFan`, `kRightFan`, `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ`, `kInvisBlower`, `kLiftArea` | 9 | `data.a.topLeft.h += deltaH`; `data.a.topLeft.v += deltaV` |
| 3 | `:588`–`604` | `kTable`, `kShelf`, `kCabinet`, `kFilingCabinet`, `kWasteBasket`, `kMilkCrate`, `kDeckTable`, `kStool`, `kTrunk`, `kInvisObstacle`, `kBooks`, `kInvisBounce` | 12 | all four `data.b.bounds` edges (`left`,`right` += `deltaH`; `top`,`bottom` += `deltaV`) |
| 4 | `:606`–`611` | `kCounter`, `kDresser`, `kManhole` | 3 | `data.b.bounds.left`, `data.b.bounds.right` += `deltaH` only |
| 5 | `:613`–`630` | `kRedClock`, `kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`, `kBattery`, `kBands`, `kGreaseRt`, `kGreaseLf`, `kFoil`, `kInvisBonus`, `kStar`, `kSparkle`, `kHelium`, `kSlider` | 15 | `data.c.topLeft.h += deltaH`; `data.c.topLeft.v += deltaV` |
| 6 | `:632`–`645` | `kUpStairs`, `kDownStairs`, `kFloorTrans`, `kCeilingTrans`, `kDoorInLf`, `kDoorInRt`, `kDoorExRt`, `kDoorExLf`, `kWindowInLf`, `kWindowInRt`, `kWindowExRt`, `kWindowExLf` | 12 | `data.d.topLeft.h += deltaH` |
| 7 | `:647`–`653` | `kMailboxLf`, `kMailboxRt`, `kInvisTrans`, `kDeluxeTrans` | 4 | `data.d.topLeft.h += deltaH`; `data.d.topLeft.v += deltaV` |
| 8 | `:655`–`666` | `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch`, `kInvisSwitch`, `kTrigger`, `kLgTrigger`, `kSoundTrigger` | 9 | `data.e.topLeft.h += deltaH`; `data.e.topLeft.v += deltaV` |
| 9 | `:668`–`674` | `kCeilingLight`, `kHipLamp`, `kDecoLamp`, `kFlourescent`, `kTrackLight` | 5 | `data.f.topLeft.h += deltaH` |
| 10 | `:676`–`681` | `kLightBulb`, `kTableLamp`, `kInvisLight` | 3 | `data.f.topLeft.h += deltaH`; `data.f.topLeft.v += deltaV` |
| 11 | `:683`–`699` | `kShredder`, `kToaster`, `kMacPlus`, `kGuitar`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`, `kMicrowave`, `kCinderBlock`, `kFlowerBox`, `kCDs`, `kCustomPict` | 14 | `data.g.topLeft.h += deltaH`; `data.g.topLeft.v += deltaV` |
| 12 | `:701`–`705` | `kBalloon`, `kCopterLf`, `kCopterRt` | 3 | `data.h.topLeft.h += deltaH` |
| 13 | `:707`–`710` | `kDartLf`, `kDartRt` | 2 | `data.h.topLeft.v += deltaV` |
| 14 | `:712`–`718` | `kBall`, `kDrip`, `kFish`, `kCobweb` | 4 | `data.h.topLeft.h += deltaH`; `data.h.topLeft.v += deltaV` |
| 15 | `:720`–`737` | `kOzma`, `kMirror`, `kFlower`, `kWallWindow`, `kBear`, `kCalendar`, `kVase1`, `kVase2`, `kBulletin`, `kCloud`, `kFaucet`, `kRug`, `kChimes` | 13 | all four `data.i.bounds` edges |
| 16 | `:739`–`743` | `kMousehole`, `kFireplace` | 2 | `data.i.bounds.left`, `data.i.bounds.right` += `deltaH` only |

7+9+12+3+15+12+4+9+5+3+14+3+2+4+13+2 = **117**. ✅

**Cross-check between the two switches.** For arms A/D (`lockH == true`) `deltaV` is always 0, so
the arms that write `.v` for those types are harmless no-ops. But the correspondence is *not*
exact:

| Type | Constraint arm | `deltaV` possible? | Commit writes `.v`? | Note |
|---|---|---|---|---|
| `kTiki` 0x0B | C (free 2-D) | yes | yes (arm 2) | consistent |
| `kSlider` 0x2F | C (free 2-D) | yes | yes (arm 5) | consistent |
| `kGreaseRt`/`kGreaseLf` | C (free 2-D) | yes | yes (arm 5) | consistent |
| `kManhole` 0x1D | A (h only) | no | no (arm 4) | consistent |
| `kCounter`/`kDresser` | D (h only) | no | no (arm 4) | consistent |
| `kFlourescent` 0x56 | A (h only) | no | no (arm 9) | consistent |
| `kTrackLight` 0x57 | D (h only) | no | no (arm 9) | consistent |
| `kMousehole`/`kFireplace` | A (h only) | no | no (arm 16) | consistent |
| `kBalloon`/`kCopterLf`/`kCopterRt` | A (h only) | no | no (arm 12) | consistent |
| `kDartLf`/`kDartRt` | B (v only) | `deltaH` is 0 | writes `.v` only (arm 13) | consistent |

Every pair is consistent — there is no type whose commit writes an axis its constraint forbade,
nor one whose constraint permits an axis the commit drops. That is worth stating explicitly
because it is exactly the kind of thing a port gets wrong.

### 5.5 Normalise and redraw

`ObjectEdit.c:747`–`767`:

```
 1. if (KeepObjectLegal()) { }             /* return value deliberately discarded */
 2. GetThisRoomsObjRects()                 /* rebuild the whole 24-entry cache */
 3. if (invalAll)
        InvalWindowRect(mainWindow, &mainWindowRect)
    else
        InvalWindowRect(mainWindow, &wasRect)                      /* the OLD rect */
        if      objActive == kInitialGliderSelected: InvalWindowRect(&initialGliderRect)
        else if objActive == kLeftGliderSelected:    InvalWindowRect(&leftStartGliderDest)
        else if objActive == kRightGliderSelected:   InvalWindowRect(&rightStartGliderDest)
        else                                         InvalWindowRect(&roomObjectRects[objActive])
 4. ReadyBackground(thisRoom->background, thisRoom->tiles)
 5. DrawThisRoomsObjects()
```

`if (KeepObjectLegal()) { }` with an empty body appears **five** times in the program
(`ObjectEdit.c:747`, `:1352`, `:1700`, and twice more in the handle arms via `whoCares =`). It is
idiomatic in this codebase: the author wanted the side effects but the compiler warned about an
unused return value, so he wrote either an empty `if` or assigned to a throwaway
(`whoCares` in `DragHandle`). A port should just call it and ignore the bool.

Note that step 3's partial-invalidate path invalidates the union of the **old** rect (`wasRect`,
captured before the drag) and the **new** rect (read back out of the freshly rebuilt cache). Both
are needed: the old one to erase, the new one to paint.

Step 4/5 is the **redraw quartet** used throughout the editor:

```
InvalWindowRect(...)                                    /* mark dirty */
GetThisRoomsObjRects()                                  /* rebuild geometry cache */
ReadyBackground(thisRoom->background, thisRoom->tiles)  /* re-composite backSrcMap */
DrawThisRoomsObjects()                                  /* draw all objects into it */
```

`ReadyBackground` rebuilds the entire room background offscreen from the `background` PICT id and
the 8 `tiles[]` indices; `DrawThisRoomsObjects` then paints all 24 slots over it. This is a
**full room re-composite for every single edit** — no dirty-rect optimisation inside the
offscreen. Only the *window blit* is dirty-rect limited, by `InvalWindowRect`. A Go port can keep
this structure (it is simple and fast enough at 512 × 322) or dirty-rect the composite too; the
observable behaviour is identical because the composite is deterministic.

### 5.6 `DragObject` does **not** re-arm the marquee

Unlike `DragHandle` (§6.5), `MoveObject` (§7.6) and `DuplicateObject` (§8.4), `DragObject` never
calls `StartMarquee`/`StartMarqueeHandled` on the way out. `DragMarqueeRect` left
`theMarquee.bounds` set to the dragged-to rect and `theMarquee.active == false` (because
`DragMarqueeRect` began with `StopMarquee()`, `Marquee.c:224`).

So after a drag-move the object is still `objActive` but has **no visible marquee** until
something re-arms it. The next `DoSelectionClick` on the same object takes the
`(theMarquee.active) && (clickedObject == objActive)` branch → `false`, because `active` is
`false` → so it falls through to step 3–5 and re-arms. Net effect for the user: after dragging,
the ants disappear; one more click brings them back. This matches the original's behaviour and a
port should reproduce it (or fix it, but then the click-count semantics of §3.2 change).

Also: `theMarquee.bounds` after the drag is the *unclamped* dragged-to rect, whereas the object's
committed rect has been through `KeepObjectLegal`. If anything called `ResumeMarquee()` before the
next `StartMarquee`, it would arm at the wrong place. Nothing does, in this path.

### 5.7 The uninitialised `invalAll` for glider pseudo-selections

`invalAll` (`ObjectEdit.c:356`) is written only in the five constraint-switch arms. When
`objActive` is −2, −3 or −4, `ObjectEdit.c:358`–`375` runs instead and the switch is skipped, so
`invalAll` holds whatever garbage was on the stack at `ObjectEdit.c:751`.

Consequences: dragging a glider marker either does a full-window invalidate (harmless, just
slower) or a partial one (correct). Both branches then run steps 4–5 which recomposite everything
into `backSrcMap` regardless, so the only variable is how much of the window gets blitted. With
68k/PPC stack reuse the value is in practice whatever the *previous* call left there, so it is
usually stable across a session, and both outcomes look right. **It is still undefined
behaviour.** A Go port gets `false` for free from zero-initialisation, which is the partial path;
combined with the §5.2 `rightStartGliderDest` bug that would leave the right marker's old pixels
stale. Recommended: initialise `invalAll = true` for the three glider cases.

---

## 6. Gesture 2 — resize by handle (`DragHandle`)

`GliderPRO/Sources/ObjectEdit.c:143`–`345`. 203 lines, one 14-arm switch, **37** case labels — the
same 37 types `ObjectHasHandle` recognises (§4.5), verified set-equal.

Locals (`ObjectEdit.c:145`–`146`):

```c
short    hDelta, vDelta;
Boolean  whoCares;
```

Common shape of every arm:

```
 1. load the object's current extent into hDelta and/or vDelta
 2. call DragMarqueeHandle(where, &X)   or   DragMarqueeCorner(where, &hDelta, &vDelta, isTop)
 3. store the (possibly clamped) result back into the object's field(s)
 4. whoCares = KeepObjectLegal()
 5. (9 of 14 arms) InvalWindowRect + GetThisRoomsObjRects + ReadyBackground + DrawThisRoomsObjects
```

Tail, for all arms (`ObjectEdit.c:343`–`344`):

```c
fileDirty = true;
UpdateMenus(false);
```

`fileDirty` is set **unconditionally**, even if the drag moved nothing. Contrast `DragObject`,
which only sets it when `deltaH || deltaV` (`ObjectEdit.c:523`–`527`).

**Every arm calls `KeepObjectLegal()`.** There is no arm that skips normalisation. (Verified by
inspecting all 14 arms; the calls are at `ObjectEdit.c:165`, `:174`, `:186`, `:203`, `:213`, `:230`,
`:246`, `:259`, `:274`, `:290`, `:302`, `:313`, `:322`, `:334`.)

There is **no `default:`**. An object type outside the 37 falls through the switch with `hDelta`
and `vDelta` uninitialised and no field written — then still sets `fileDirty`. Unreachable via
`DoSelectionClick` because that only calls `DragHandle` when `PtInMarqueeHandle` is true, which in
turn requires `theMarquee.handled`, which `StartMarqueeHandled` only sets when `ObjectHasHandle`
returned true. So the 37-label set is exactly right.

### 6.1 The 14 arms in full

| # | Lines | `what` codes | Count | Primitive | Field(s) written | Redraw quartet? |
|---:|---|---|---:|---|---|:---:|
| 1 | `:150`–`166` | `kFloorVent`, `kCeilingVent`, `kFloorBlower`, `kCeilingBlower`, `kSewerGrate`, `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ`, `kGrecoVent`, `kSewerBlower` | 12 | `DragMarqueeHandle(&vDelta)` | `data.a.distance = vDelta` | no |
| 2 | `:168`–`179` | `kLiftArea` | 1 | `DragMarqueeCorner(&hDelta, &vDelta, false)` | `data.a.distance = hDelta`; `data.a.tall = vDelta / 2` | **yes** |
| 3 | `:181`–`187` | `kLeftFan`, `kRightFan` | 2 | `DragMarqueeHandle(&hDelta)` | `data.a.distance = hDelta` | no |
| 4 | `:189`–`204` | `kInvisBlower` | 1 | `DragMarqueeHandle` (axis chosen by `vector`) | `data.a.distance` | no |
| 5 | `:206`–`218` | `kTable`, `kShelf`, `kDeckTable` | 3 | `DragMarqueeHandle(&hDelta)` | `data.b.bounds.right = bounds.left + hDelta` | **yes** |
| 6 | `:220`–`235` | `kCabinet`, `kInvisObstacle`, `kInvisBounce` | 3 | `DragMarqueeCorner(..., false)` | `bounds.right = left + hDelta`; `bounds.bottom = top + vDelta` | **yes** |
| 7 | `:237`–`251` | `kCounter`, `kDresser` | 2 | `DragMarqueeCorner(..., **true**)` | `bounds.right = left + hDelta`; `bounds.top = bottom - vDelta` | **yes** |
| 8 | `:253`–`264` | `kGreaseRt`, `kGreaseLf`, `kSlider` | 3 | `DragMarqueeHandle(&hDelta)` | `data.c.length = hDelta` | **yes** |
| 9 | `:266`–`279` | `kInvisTrans` | 1 | `DragMarqueeCorner(..., false)` | `data.d.wide = (Byte)min(hDelta,127)`; `data.d.tall = vDelta` | **yes** |
| 10 | `:281`–`295` | `kDeluxeTrans` | 1 | `DragMarqueeCorner(..., false)` | `data.d.tall = ((hDelta/4)<<8) + (vDelta/4)` | **yes** |
| 11 | `:297`–`307` | `kFlourescent`, `kTrackLight` | 2 | `DragMarqueeHandle(&hDelta)` | `data.f.length = hDelta` | **yes** |
| 12 | `:309`–`314` | `kToaster` | 1 | `DragMarqueeHandle(&vDelta)` | `data.g.height = vDelta` | no |
| 13 | `:316`–`323` | `kBall`, `kDrip`, `kFish` | 3 | `DragMarqueeHandle(&vDelta)` | `data.h.length = vDelta` | no |
| 14 | `:325`–`339` | `kMirror`, `kWallWindow` | 2 | `DragMarqueeCorner(..., false)` | `bounds.right = left + hDelta`; `bounds.bottom = top + vDelta` (`data.i`) | **yes** |

12 + 1 + 2 + 1 + 3 + 3 + 2 + 3 + 1 + 1 + 2 + 1 + 3 + 2 = **37**. ✅

**Which arms omit the redraw quartet, and why.** Arms 1, 3, 4, 12 and 13 (18 types total) do
*not* invalidate or recomposite. In every one of those the field being edited is a *distance*, not
part of the object's own rect:

| Arm | Field | Does `GetObjectRect` use it? |
|---|---|---|
| 1, 3, 4 | `data.a.distance` | no — blower rects come from `srcRects[what]` alone |
| 12 | `data.g.height` (`kToaster`) | no — `srcRects[kToaster]` is 48 × 27 |
| 13 | `data.h.length` (`kBall`/`kDrip`/`kFish`) | no — enemy rects come from `srcRects[what]` |

So the object's on-screen appearance genuinely does not change; only the marquee handle moved,
and `DragMarqueeHandle` already erased its own XOR feedback. Correct, and a port can rely on the
same reasoning. Conversely arms 2, 5–11 and 14 all change the object's rect and must recomposite.

`RectWide` / `RectTall` (`GliderPRO/Sources/RectUtils.c:95`–`98`, `:103`–`106`):

```c
short RectWide (Rect *theRect) { return (theRect->right - theRect->left); }
short RectTall (Rect *theRect) { return (theRect->bottom - theRect->top); }
```

### 6.2 Arm 2 — `kLiftArea` and the `tall × 2` encoding

`ObjectEdit.c:168`–`179`:

```
 1. hDelta = data.a.distance
 2. vDelta = data.a.tall * 2
 3. DragMarqueeCorner(where, &hDelta, &vDelta, false)
 4. data.a.distance = hDelta
 5. data.a.tall     = vDelta / 2
 6. whoCares = KeepObjectLegal()
 7. redraw quartet
```

`data.a.tall` is a **`Byte`** (`GliderPRO/Headers/GliderStructs.h:18`), and `kLiftArea`'s rect is
built as `QSetRect(itsRect, 0, 0, distance, tall * 2)`
(`GliderPRO/Sources/ObjectRects.c:63`). So the on-disk `tall` is **half** the pixel height, giving
a 2-pixel vertical quantum and a maximum height of `255 × 2` = 510 px. The `/ 2` at step 5
truncates, so odd pixel heights round **down**.

`data.a.distance` is a `short` and is used *directly* as the pixel width, so the width is
unquantised. Asymmetric, and easy to get wrong.

> **Empirical.** Across 22 houses there are 716 `kLiftArea` objects with **519 distinct
> `(distance, tall)` pairs**; `distance` ranges `[1, 512]`. Every `tall` is a valid `Byte`.

### 6.3 Arms 9 and 10 — the two packed-field transports

**Arm 9, `kInvisTrans`** (`ObjectEdit.c:266`–`279`):

```
 1. hDelta = data.d.wide          /* Byte, unsigned */
 2. vDelta = data.d.tall          /* short */
 3. DragMarqueeCorner(where, &hDelta, &vDelta, false)
 4. if (hDelta > 127) hDelta = 127
 5. data.d.wide = (Byte)hDelta
 6. data.d.tall = vDelta
 7. KeepObjectLegal(); redraw quartet
```

`GetObjectRect` for `kInvisTrans` does
`bottom = top + tall; right += (short)wide` on top of `srcRects[kInvisTrans]` = 64 × 32
(`GliderPRO/Sources/ObjectRects.c`), i.e. the width is **64 + wide** and the height is **tall**
(not 32 + tall). So `wide` is an *extension* past the 64 px base and `tall` is an *absolute*
height.

The 127 clamp at step 4 is deliberate — `wide` is a `Byte` and could hold 255, but the author
capped it at 127. Combined with the 64 px base that limits `kInvisTrans` to 191 px wide.

> **Empirical.** 385 `kInvisTrans` objects. `wide` ∈ `[0, 127]`, and the value **127 appears
> exactly 6 times** — the clamp is real and was hit by the level designers. `tall` ∈ `[21, 322]`,
> so the 322 upper end is `kTileHigh` exactly, i.e. `KeepObjectLegal`'s
> `tall = kTileHigh - topLeft.v` clamp (§12.5) is also exercised.

Note also that `HouseLegal.c:379` contains `if (theObject->data.d.wide < 0) theObject->data.d.wide = 0;`
— **dead code**, because `wide` is `Byte wide;` (`GliderPRO/Headers/GliderStructs.h:42`), an
unsigned 8-bit type that can never be negative. A Go port using `uint8` gets the same
impossibility; a port using `int` must add the clamp back.

**Arm 10, `kDeluxeTrans`** (`ObjectEdit.c:281`–`295`) packs *both* dimensions into the single
`short data.d.tall`, in units of 4 pixels:

```
 1. hDelta = ((data.d.tall & 0xFF00) >> 8) * 4      /* high byte = width/4  */
 2. vDelta =  (data.d.tall & 0x00FF)       * 4      /* low byte  = height/4 */
 3. DragMarqueeCorner(where, &hDelta, &vDelta, false)
 4. if (hDelta < 64) hDelta = 64                    /* minimum 64 px wide  */
 5. if (vDelta < 32) vDelta = 32                    /* minimum 32 px tall  */
 6. data.d.tall = ((hDelta / 4) << 8) + (vDelta / 4)
 7. KeepObjectLegal(); redraw quartet
```

Encoding summary:

| Quantity | Bits | Formula | Quantum | Range |
|---|---|---|---:|---|
| width | `tall` bits 15..8 | `((tall & 0xFF00) >> 8) * 4` | 4 px | 0 … 1020 px |
| height | `tall` bits 7..0 | `(tall & 0x00FF) * 4` | 4 px | 0 … 1020 px |

Minima enforced by the editor: width ≥ 64 (packed ≥ 16), height ≥ 32 (packed ≥ 8). Because step 6
divides *before* shifting, a width of 65..67 truncates to packed 16 → 64 px. `GetObjectRect`
decodes identically (`ObjectRects.c`: `wide = (tall & 0xFF00) >> 8; tall = tall & 0x00FF;
QSetRect(0,0,wide*4,tall*4)`).

> **Empirical.** 61 `kDeluxeTrans` objects across 22 houses. Packed width ∈ `[16, 128]`
> (= 64 … 512 px) and packed height ∈ `[8, 80]` (= 32 … 320 px). **Both editor minima (16 and 8)
> are exactly the observed minima**, confirming the clamp is what produced the shipped data.
> Every `KeepObjectLegal` invariant test passed: `packedW >= 16` 61/61, `packedH >= 8` 61/61.

Both packed fields are stored **big-endian** on disk (the whole house file is), so the high byte
of `tall` is the *first* byte at that offset. A little-endian port must byte-swap the `short`
before unpacking, not unpack the raw bytes.

### 6.4 Arm 4 — `kInvisBlower`'s axis choice

`ObjectEdit.c:189`–`204`:

```c
if (((data.a.vector & 0x0F) == 1) || ((data.a.vector & 0x0F) == 4))
    /* vertical drag:   vDelta = distance; DragMarqueeHandle(where, &vDelta); distance = vDelta; */
else
    /* horizontal drag: hDelta = distance; DragMarqueeHandle(where, &hDelta); distance = hDelta; */
```

Low nibble 1 (`kAbove`) or 4 (`kBelow`) → vertical; everything else → horizontal. Note this is the
*complement* of `ObjectHasHandle`'s mapping, which sends 2 → `kToRight` and 8 → `kToLeft`. The two
agree, but the encoding is worth restating:

| `vector & 0x0F` | `ObjectHasHandle` direction | `DragHandle` axis |
|---:|---|---|
| 1 | `kAbove` | vertical |
| 2 | `kToRight` | horizontal |
| 4 | `kBelow` | vertical |
| 8 | `kToLeft` | horizontal |

Because `DragMarqueeHandle` reads `theMarquee.direction` (not the vector) to pick the sign of the
accumulation, the `hDelta`/`vDelta` naming in `DragHandle` is purely cosmetic — both arms pass a
`short*` and the primitive does the right thing. The reason the arm exists at all is that the two
branches are otherwise identical; it could have been one branch. A port can collapse it.

### 6.5 `DragHandle` and the marquee

`DragHandle` never re-arms the marquee either. `DragMarqueeHandle`/`DragMarqueeCorner` both
started with `StopMarquee()` and end by XOR-erasing their own feedback, so on return
`theMarquee.active == false` but `theMarquee.bounds`, `.direction`, `.dist` and `.handled` still
hold the pre-drag values (for `DragMarqueeHandle`) or the dragged-to bounds (for
`DragMarqueeCorner`, which mutates `theMarquee.bounds` in place at `Marquee.c:385`–`386`,
`:391`–`392`).

So after a corner drag `theMarquee.bounds` is the new, *unclamped*, extent while the object's
committed bounds have been through `KeepObjectLegal`. Anything that calls `ResumeMarquee()` next
would arm at the wrong rect. In practice the next `DoSelectionClick` re-arms from
`roomObjectRects[objActive]`, which is correct.

`theMarquee.dist` is **never updated** by `DragHandle`. So the Coordinates windoid's `d:` value —
which `DeltaCoordinateD` kept live during the drag — is correct on screen but
`MarqueeHasHandles(&dir, &dist)` will report the *old* distance until the marquee is re-armed
(§4.6).

---

## 7. Gesture 3 — arrow-key nudge (`MoveObject`)

`GliderPRO/Sources/ObjectEdit.c:1374`–`1772`. 399 lines, the second-largest function in the file
after `DrawThisRoomsObjects`.

### 7.1 It is dead code in the shipped 1.0.4 build

`GliderPRO/Sources/Events.c` has two mutually exclusive arrow-key arms:

```c
#if BUILD_ARCADE_VERSION                                  /* Events.c:191 */
        case kLeftArrowKeyASCII:  DoOptionsMenu(iHighScores); break;      /* :193-195 */
        case kRightArrowKeyASCII: DoOptionsMenu(iHelp);       break;      /* :197-199 */
        case kUpArrowKeyASCII:    DoGameMenu(iNewGame);       break;      /* :201-203 */
        case kDownArrowKeyASCII:  DoGameMenu(iNewGame);       break;      /* :205-207 */
#else                                                     /* Events.c:209 */
        case kLeftArrowKeyASCII:
            if (houseUnlocked) {
                if (objActive == kNoObjectSelected) SelectNeighborRoom(kRoomToLeft);
                else                                MoveObject(kBumpLeft, shiftDown);
            }
            break;                                                       /* :211-219 */
        /* ... kRightArrowKeyASCII :221-229, kUpArrowKeyASCII :231-239,
               kDownArrowKeyASCII :241-249 ... */
#endif                                                    /* Events.c:251 */
```

`BUILD_ARCADE_VERSION` is `1` (`GliderPRO/Headers/GliderDefines.h:15`), so the **arcade** arm is
compiled and the arrow keys are wired to the High Scores dialog, the Help dialog, and New Game
(twice) — **not** to the editor. `MoveObject` has no other caller anywhere in the 67 source files.

**In shipped Glider PRO 1.0.4 you cannot nudge an object with the arrow keys.** The arrow keys in
the editor open dialogs and start games.

Note also that the `#else` arm is what documents the *intended* design: with nothing selected the
arrow keys navigate to the neighbouring room (`SelectNeighborRoom(kRoomToLeft)` etc.); with
something selected they nudge it. And `houseUnlocked` gates both.

**Decision for a Go port:** implement `MoveObject` (it is the natural keyboard-nudge behaviour and
the code fully specifies it) but wire it behind an explicit flag, and route the arrow keys to
nudge/navigate rather than to dialogs. Do not treat the arcade arm as canonical editor behaviour —
it is a kiosk-build artefact. The bugs documented in §7.4 and §7.5 exist *because* this path was
never exercised in the shipped build; a port should fix them rather than reproduce them.

### 7.2 The guard, and the guard that is missing

`ObjectEdit.c:1382`–`1385`:

```c
if (theMode != kEditMode)
    return;

StopMarquee();
```

That is the **only** guard. There is **no `objActive == kNoObjectSelected` check**. The caller
(`Events.c:214`, `:224`, `:234`, `:244`) does check, but any other caller would not be protected.
With `objActive == -1` the very next statement to touch it,
`thisRoom->objects[objActive].what` at `ObjectEdit.c:1399`, reads **out of bounds**.

Because `roomType` has `objectType objects[24]` at offset **60** and `objectType` is **12 bytes**
(§11.1), `objects[-1]` lands at room offset `60 - 12` = **48**, which is inside `short tiles[8]`
(offset 36, 16 bytes, so `tiles[0..7]` occupy 36..51). Offset 48 is `tiles[6]`.

| Index | `objects[i]` byte offset in `roomType` | Overlaps |
|---:|---:|---|
| −4 | 12 | `Str27 name` bytes 12..23 |
| −3 | 24 | `Str27 name` bytes 24..27 + `bounds` + `leftStart`/`rightStart`/`unusedByte`/`visited` + `background` low byte |
| −2 | 36 | `tiles[0..5]` |
| −1 | 48 | `tiles[6]`, `tiles[7]`, `floor`, `suite`, `openings`, `numObjects` |
| 0 | 60 | first real object |

> **Verified against real bytes.** `GliderPRO/Houses/Slumberland.binhex`, room 0 (`'Live!'`),
> at data-fork file offset 866 (`0x362`, = `SIZEOF_HOUSE_HEADER`). Raw room prefix:
> ```
> 05 4c 69 76 65 21 bb 49 6e 20 53 74 65 72 65 6f 21 20 71 00 07 e0 1f f8 3f fc 7f fe
> 00 00 20 20 00 00 07 d0 00 00 00 05 00 01 00 01 00 01 00 01 00 03 00 04 00 01 00 3f
> 00 00 00 0b 00 31 00 1c
> ```
> Reads:
> | Expression | File offset | Bytes | Value |
> |---|---:|---|---:|
> | `objects[-1].what` (= `tiles[6]`) | `0x392` | `00 03` | **3** |
> | `objects[-2].what` (= `tiles[0]`) | `0x386` | `00 00` | **0** |
> | `objects[-3].what` (= name bytes 24..25) | `0x37a` | `3f fc` | **16380** |
> | `objects[-4].what` (= name bytes 12..13) | `0x36e` | `65 72` | **25970** ("er" of "Stereo") |
> | `objects[0].what` | `0x39e` | `00 31` | 49 = `kUpStairs` |
> | `objects[1].what` | `0x3aa` | `00 51` | 81 = `kCeilingLight` |
>
> `objects[-1].what` = 3 = `kFloorBlower`, a *valid* `what` code. So the switch at
> `ObjectEdit.c:1399` would take a real arm and the commit switch at `:1523` would **write
> through the negative index**, corrupting the room's tile map.

Where a `MoveObject` with `objActive == -1` writes: the blower arm writes `data.a.topLeft`, which
sits at `objectType` offset **2** (`.h`) and **4** (`.v`) — wait, `Point` is `{short v; short h;}`
so `.v` is at +2 and `.h` at +4 (§11.1). At `objects[-1]` that is room offset 50 and 52, i.e.
**`tiles[7]`** and **`floor`**.

| `tiles[6]` value | `what` interpretation | `MoveObject` commit arm | What gets corrupted |
|---:|---|---|---|
| 0 | (no case, no `default:`) | none | nothing written |
| 1 (`kFloorVent`) | 7-vent arm | `.h` only | `floor` |
| 2 (`kCeilingVent`) | 7-vent arm | `.h` only | `floor` |
| 3 (`kFloorBlower`) | 7-vent arm | `.h` only | `floor` |
| 4 (`kCeilingBlower`) | 7-vent arm | `.h` only | `floor` |
| 5 (`kSewerGrate`) | 7-vent arm | `.h` only | `floor` |
| 6 (`kLeftFan`) | 9-type arm | `.h` and `.v` | `tiles[7]` **and** `floor` |
| 7 (`kRightFan`) | 9-type arm | `.h` and `.v` | `tiles[7]` **and** `floor` |

> **Empirical.** Over all **4070 non-empty rooms** in the 22 shipped houses the distribution of
> `tiles[6]` is:
> | `tiles[6]` | count |
> |---:|---:|
> | 0 | 469 |
> | 1 | 843 |
> | 2 | 774 |
> | 3 | 328 |
> | 4 | 400 |
> | 5 | 221 |
> | 6 | 899 |
> | 7 | 136 |
> **`tiles[6]` is never −1.** That matters for §10.3.

Additionally, `wasRect = roomObjectRects[objActive]` at `ObjectEdit.c:1522` with `objActive == -1`
is an **out-of-bounds BSS read** 8 bytes before `roomObjectRects`.

A Go port must add the `objActive == kNoObjectSelected` guard. Go's bounds checking would panic on
`objects[-1]` anyway, which is arguably the right outcome, but a clean early return is better.

### 7.3 The increment table — 2 px for exactly 44 types

`ObjectEdit.c:1387`–`1460`:

```
 1. if (shiftDown):
        increment = 10                                   /* ObjectEdit.c:1387-1388 */
 2. else if (objActive == kInitialGliderSelected):
        increment = 1                                    /* :1391-1394 */
 3. else if (whichWay == kBumpRight || whichWay == kBumpLeft):     /* HORIZONTAL */
        switch (thisRoom->objects[objActive].what):
            <44 types>:  increment = 2                   /* :1401-1446 */
            kManhole:    increment = 64                  /* :1448-1450 */
            default:     increment = 1                   /* :1452-1454 */
 4. else:                                                /* VERTICAL */
        increment = 1                                    /* :1458 */
```

Note that **Shift always means 10 px**, overriding both the per-type increments *and* the
`kManhole` 64 px grid. Shift-nudging a manhole by 10 px puts it off its 64 px grid — but
`KeepObjectLegal` snaps it back (§12.2), so the net effect is either 0 or 64. Similarly Shift-10
on a parity-forced type gives an even 10, which survives the parity forcing.

Note also that **vertical nudges are always 1 px** (step 4) regardless of type. Only horizontal
nudges get the special increments. That is because it is only the *x* coordinate that
`KeepObjectLegal` parity-forces.

**The 44 types that get `increment = 2`** (`ObjectEdit.c:1401`–`1444`, one `case` label per line
through `case kMirror:` at line 1444):

| Group | Types | Count |
|---|---|---:|
| blowers | `kTaper` 0x08, `kCandle` 0x09, `kStubby` 0x0A, `kTiki` 0x0B, `kBBQ` 0x0C | 5 |
| bonuses | `kRedClock` 0x21, `kBlueClock` 0x22, `kYellowClock` 0x23, `kCuckoo` 0x24, `kPaper` 0x25, `kBattery` 0x26, `kBands` 0x27, `kGreaseRt` 0x28, `kGreaseLf` 0x29, `kFoil` 0x2A, `kInvisBonus` 0x2B, `kStar` 0x2C, `kSparkle` 0x2D, `kHelium` 0x2E, `kSlider` 0x2F | 15 |
| switches | `kLightSwitch` 0x41, `kMachineSwitch` 0x42, `kThermostat` 0x43, `kPowerSwitch` 0x44, `kKnifeSwitch` 0x45, `kInvisSwitch` 0x46, `kTrigger` 0x47, `kLgTrigger` 0x48, `kSoundTrigger` 0x49 | 9 |
| appliances | `kToaster` 0x62, `kMacPlus` 0x63, `kTV` 0x65, `kCoffee` 0x66, `kOutlet` 0x67, `kVCR` 0x68, `kStereo` 0x69, `kMicrowave` 0x6A | 8 |
| enemies | `kBalloon` 0x71, `kCopterLf` 0x72, `kCopterRt` 0x73, `kBall` 0x76, `kDrip` 0x77, `kFish` 0x78 | 6 |
| clutter | `kMirror` 0x82 | 1 |
| | **total** | **44** |

> **Verified invariant.** This 44-type set is **exactly** the set of types whose horizontal
> coordinate `KeepObjectLegal` parity-forces (§12.2) — the 5 even-x blowers, all 15 bonuses, all
> 9 switches, the 7 even-x appliances **plus the odd-x `kTV`**, the 6 even-x enemies, and
> `kMirror`. Verified programmatically as a set equality, not merely a count match.
>
> The design intent is now obvious: a ±1 px horizontal nudge on any of these types would be
> **immediately undone** by `KeepObjectLegal`'s parity forcing, so the nudge is 2 px to guarantee
> the object actually moves and stays on its parity. `kTV` is in the list even though its parity
> is *odd* rather than even — the argument is about the quantum, not the phase.

`kManhole`'s 64 is the tile width (`kTileWide` = 64), matching `KeepObjectLegal`'s
`(left - 3) % 64 == 0` snap (§12.2).

### 7.4 The delta mapping

`ObjectEdit.c:1462`–`1483`. Four arms, **no `default:`**:

| `whichWay` | Value | `deltaH` | `deltaV` |
|---|---:|---:|---:|
| `kBumpUp` | 1 | 0 | `-increment` |
| `kBumpDown` | 2 | 0 | `+increment` |
| `kBumpRight` | 3 | `+increment` | 0 |
| `kBumpLeft` | 4 | `-increment` | 0 |

Because there is no `default:`, an out-of-range `whichWay` leaves both deltas uninitialised. Only
the four constants are ever passed.

### 7.5 The glider pseudo-selection commits — and the `rightStartGliderDest` fix

`ObjectEdit.c:1485`–`1519`. Identical to `DragObject`'s (§5.2) with one difference and one
addition:

* Each arm additionally captures `wasRect` (from `initialGliderRect` / `leftStartGliderDest` /
  `rightStartGliderDest`) before mutating.
* `ObjectEdit.c:1517` uses the **correct** x offset:
  ```c
  QSetRect(&rightStartGliderDest, 0, 0, 48, 16);
  QOffsetRect(&rightStartGliderDest, kRoomWide - 48,          /* 464 -- CORRECT */
          kGliderStartsDown + (short)thisRoom->rightStart);
  ```
  compared with `ObjectEdit.c:558`'s `0`. This is the direct evidence that `:558` is a typo and
  464 is intended.

### 7.6 The commit switch — 109 labels, 8 types missing

`ObjectEdit.c:1523`–`1697`. Structurally the same 16-arm shape as `DragObject`'s commit switch
(§5.4) and writes the same fields with `+= deltaH` / `+= deltaV`. But it has only **109 case
labels**, and there is **no `default:`**.

> **Verified by programmatic diff** of the two label sets:
> ```
> DragObject commit  : 117 labels
> MoveObject commit  : 109 labels
> DragObject − MoveObject = ['kDoorInLf', 'kDoorInRt', 'kDoorExRt', 'kDoorExLf',
>                            'kWindowInLf', 'kWindowInRt', 'kWindowExRt', 'kWindowExLf']
> MoveObject − DragObject = []   (empty)
> ```

The eight missing types are exactly the **doors and windows**:

| `what` | Value | Forced `topLeft.h` (`GliderDefines.h:481`–`494`) |
|---|---:|---:|
| `kDoorInLf` | 0x37 | `kDoorInLfLeft` = 0 |
| `kDoorInRt` | 0x38 | `kDoorInRtLeft` = 368 |
| `kDoorExRt` | 0x39 | `kDoorExRtLeft` = 496 |
| `kDoorExLf` | 0x3A | `kDoorExLfLeft` = 0 |
| `kWindowInLf` | 0x3B | `kWindowInLfLeft` = 0 |
| `kWindowInRt` | 0x3C | `kWindowInRtLeft` = 492 |
| `kWindowExRt` | 0x3D | `kWindowExRtLeft` = 496 |
| `kWindowExLf` | 0x3E | `kWindowExLfLeft` = 0 |

Because `KeepObjectLegal` forces those `topLeft.h` values back anyway (§12.4), *omitting* the write
is behaviourally equivalent to performing it: `DragObject` writes `data.d.topLeft.h += deltaH` and
then `KeepObjectLegal` overwrites it with the constant. So the "bug" is invisible.

What is **not** invisible: arrow-nudging a door or window still

1. runs `KeepObjectLegal()` (`ObjectEdit.c:1700`),
2. sets `fileDirty = true` (`:1703`) and calls `UpdateMenus(false)` (`:1704`),
3. rebuilds the geometry cache (`:1705`),
4. invalidates and fully recomposites the room (`:1745`–`:1748`, `:1752`–`:1753`),
5. re-arms the marquee (`:1755`–`:1769`).

So the user sees the house become dirty and the room flash, with nothing having moved. That is
worth flagging in a port: either add the eight labels (harmless) or short-circuit the whole
function for forced-coordinate types.

Whether this is a deliberate omission or a copy-paste slip is unresolvable from the source. The
`kUpStairs`/`kDownStairs`/`kFloorTrans`/`kCeilingTrans` arm survives in `MoveObject` while the 8
door/window labels that follow it in `DragObject` (`ObjectEdit.c:636`–`643`) do not, which looks
like a truncated paste. See [Open questions](#open-questions).

### 7.7 Normalise, invalidate, recomposite, re-arm

`ObjectEdit.c:1700`–`1770`:

```
 1. if (KeepObjectLegal()) { }
 2. fileDirty = true
 3. UpdateMenus(false)
 4. GetThisRoomsObjRects()
 5. invalidate:
      objActive == kInitialGliderSelected: Inval(wasRect); Inval(initialGliderRect)
      objActive == kLeftGliderSelected:    Inval(wasRect); Inval(leftStartGliderDest)
      objActive == kRightGliderSelected:   Inval(wasRect); Inval(rightStartGliderDest)
      else switch (what):
        <16 types>:  Inval(mainWindowRect)                  /* whole window */
        default:     Inval(wasRect); Inval(roomObjectRects[objActive])
 6. ReadyBackground(thisRoom->background, thisRoom->tiles)
 7. DrawThisRoomsObjects()
 8. if ObjectHasHandle(&direction, &dist):
        StartMarqueeHandled(&roomObjectRects[objActive], direction, dist)
        HandleBlowerGlider()
    else:
        StartMarquee(<the appropriate rect for objActive>)
```

Note `fileDirty` is set **unconditionally** here too — even a nudge that `KeepObjectLegal`
completely undid, and even for the 8 door/window types that write nothing.

**The 16 whole-window types** (`ObjectEdit.c:1726`–`1743`): `kTiki` 0x0B, `kTable` 0x11, `kShelf`
0x12, `kCabinet` 0x13, `kDeckTable` 0x19, `kStool` 0x1A, `kCounter` 0x17, `kDresser` 0x18,
`kGreaseRt` 0x28, `kGreaseLf` 0x29, `kSlider` 0x2F, `kMailboxLf` 0x33, `kMailboxRt` 0x34,
`kTrackLight` 0x57, `kMirror` 0x82, `kWallWindow` 0x86.

Compare `DragObject`'s `invalAll == true` set — arm C (17 types) ∪ arm D (3 types) = 20 types:

| Type | `DragObject` `invalAll` | `MoveObject` whole-window |
|---|:---:|:---:|
| `kTiki`, `kTable`, `kShelf`, `kCabinet`, `kDeckTable`, `kStool`, `kGreaseRt`, `kGreaseLf`, `kSlider`, `kMailboxLf`, `kMailboxRt`, `kMirror`, `kWallWindow` (13) | yes | yes |
| `kCounter`, `kDresser`, `kTrackLight` (3) | yes | yes |
| `kInvisObstacle` 0x1C, `kInvisBounce` 0x1F, `kInvisTrans` 0x3F, `kDeluxeTrans` 0x40 (4) | **yes** | **no** |

So `MoveObject` uses the partial invalidate for the four *invisible* extended-bounds types where
`DragObject` uses the full one. Since those objects draw nothing visible in play (and in the
editor draw only an outline within their own rect), the partial path is arguably more correct and
the difference is unobservable. A port can use either set; the 16-type set is cheaper.

**`MoveObject` is the only gesture that re-arms the marquee itself** (step 8), which is why an
arrow-key nudge would have kept the ants alive while drag-move (§5.6) does not.

---

## 8. Gesture 4 — duplicate (`DuplicateObject`)

`GliderPRO/Sources/ObjectEdit.c:1191`–`1370`. **Not** wrapped in `#ifndef COMPILEDEMO`.

Locals (`ObjectEdit.c:1193`–`1196`):

```c
objectType  tempObject;
Point       placePt;
short       direction, dist;
Boolean     handled;
```

### 8.1 It delegates to `AddNewObject` — so per-type caps are enforced

`ObjectEdit.c:1198`–`1207`:

```
 1. tempObject = thisRoom->objects[objActive]          /* 12-byte struct copy, ObjectEdit.c:1198 */
 2. placePt.h = roomObjectRects[objActive].left + HalfRectWide(&roomObjectRects[objActive]) + 64
 3. placePt.v = roomObjectRects[objActive].top  + HalfRectTall(&roomObjectRects[objActive])
 4. StopMarquee()
 5. if (AddNewObject(placePt, tempObject.what, false)):
 6.     <16-arm field-copy switch>
 7.     if (KeepObjectLegal()) { }
 8.     handled = ObjectHasHandle(&direction, &dist)
 9.     ReadyBackground(thisRoom->background, thisRoom->tiles)
10.     GetThisRoomsObjRects()
11.     DrawThisRoomsObjects()
12.     InvalWindowRect(mainWindow, &mainWindowRect)
13.     if handled: StartMarqueeHandled(&roomObjectRects[objActive], direction, dist); HandleBlowerGlider()
        else:       StartMarquee(&roomObjectRects[objActive])
```

**This is the single most important structural fact about `DuplicateObject`:** it does *not* copy
the object into a free slot itself. It asks `AddNewObject` to create a brand-new object of the same
`what` at `placePt`, and then patches the *non-positional* fields across. Therefore:

* `AddNewObject`'s **per-type object caps** apply. If the room already has the maximum number of
  that type, `AddNewObject` returns `false`, the whole switch and redraw are skipped, and
  **nothing happens** — no beep, no dialog, no `fileDirty`.
* `AddNewObject`'s **room-full check** applies: with all 24 slots occupied it returns `false`.
* `AddNewObject` **writes `objActive` first** (`GliderPRO/Sources/ObjectAdd.c:57`) and only then
  runs its cap checks, so a *refused* duplicate leaves `objActive` pointing at an empty slot.
  Combined with `StopMarquee()` at step 4, the user is left with no marquee and `objActive` on an
  empty slot. The next `DoSelectionClick` recovers.
* The third argument to `AddNewObject` is `false`, versus `true` from `DoNewObjectClick`
  (`ObjectEdit.c:783`). That flag selects whether the new object gets a marquee armed inside
  `AddNewObject`; `DuplicateObject` arms it itself at step 13.

**There is no `objActive` guard.** `ObjectEdit.c:1198` dereferences `thisRoom->objects[objActive]`
with no check at all. Called with `objActive == kNoObjectSelected` (−1) it reads `objects[-1]` (see
the offset table in §7.2) and with −2/−3/−4 it reads further back into the room name. `tempObject.what`
would then be a garbage `what` code, `AddNewObject` would reject it (or worse, accept a valid-looking
one), and `roomObjectRects[objActive]` at steps 2–3 is an OOB read. The **only** thing preventing
this is `Menu.c` disabling the Duplicate menu item when nothing is selected. A Go port must add:

```
if (theMode != kEditMode) || (objActive < 0) { return }
```

### 8.2 The placement point

```c
placePt.h = roomObjectRects[objActive].left + HalfRectWide(&roomObjectRects[objActive]) + 64;
placePt.v = roomObjectRects[objActive].top  + HalfRectTall(&roomObjectRects[objActive]);
```
`ObjectEdit.c:1200`–`1203`.

So the duplicate is placed with its **centre** at the original's centre, offset **+64 px to the
right** (`kTileWide` = 64, i.e. exactly one tile). Note the asymmetry:

* `placePt.h` uses `left + width/2 + 64` — a **centre**, offset right.
* `placePt.v` uses `top + height/2` — a **centre**, unshifted.

`AddNewObject` interprets `placePt` as a centre and back-computes the new object's `topLeft`. That
is why the copy-switch arms that *do* set the position (`kLiftArea`, `kSlider`, `kInvisTrans`,
`kDeluxeTrans`, `kCustomPict`, the 8 bounds-copy furniture, the 3 bounds-copy clutter) all apply a
**separate** `+64` of their own — they overwrite `AddNewObject`'s placement entirely and must
re-derive the offset. That is why `+64` appears twice.

Both `HalfRectWide` and `HalfRectTall` truncate (§4.2), so for odd-size objects the duplicate is
half a pixel up/left of the original's centre before the `+64`.

If the object is at the right edge of the room, `placePt.h` lands outside; `KeepObjectLegal` (step
7) pulls the duplicate back inside. Duplicating repeatedly against the right wall therefore stacks
copies on top of each other rather than marching them off-screen.

### 8.3 The 16-arm field-copy switch — exactly which fields are copied

`ObjectEdit.c:1209`–`1350`. **117 case labels**, no `default:`. The labels are packed several per
line separated by tabs, which is why a naive line-based count reports 44.

| # | Lines | `what` codes | Count | Fields copied from `tempObject` |
|---:|---|---|---:|---|
| 1 | `:1211`–`1221` | `kFloorVent`, `kCeilingVent`, `kFloorBlower`, `kCeilingBlower`, `kSewerGrate`, `kLeftFan`, `kRightFan`, `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ`, `kInvisBlower`, `kGrecoVent`, `kSewerBlower` | 15 | `data.a.distance`, `.initial`, `.state`, `.vector`, `.tall` — **not** `topLeft` |
| 2 | `:1223`–`1231` | `kLiftArea` | 1 | `data.a.topLeft.h = temp.h + 64`, `.topLeft.v`, `.distance`, `.initial`, `.state`, `.vector`, `.tall` |
| 3 | `:1233`–`1237` | `kFilingCabinet`, `kWasteBasket`, `kMilkCrate`, `kStool`, `kTrunk`, **`kManhole`**, `kBooks` | 7 | `data.b.pict` only |
| 4 | `:1239`–`1245` | `kTable`, `kShelf`, `kCabinet`, **`kCounter`**, **`kDresser`**, `kDeckTable`, `kInvisObstacle`, `kInvisBounce` | 8 | `data.b.bounds` (whole rect), then `QOffsetRect(&bounds, 64, 0)`, then `data.b.pict` |
| 5 | `:1247`–`1256` | `kRedClock`, `kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`, `kBattery`, `kBands`, `kGreaseRt`, `kGreaseLf`, `kFoil`, `kInvisBonus`, `kStar`, `kSparkle`, `kHelium` | 14 | `data.c.length`, `.points`, `.state`, `.initial` |
| 6 | `:1258`–`1264` | `kSlider` | 1 | `data.c.topLeft.h = temp.h + 64`, `.length`, `.points`, `.state`, `.initial` (**not** `topLeft.v`) |
| 7 | `:1266`–`1275` | `kUpStairs`, `kDownStairs`, `kMailboxLf`, `kMailboxRt`, `kFloorTrans`, `kCeilingTrans`, `kDoorInLf`, `kDoorInRt`, `kDoorExRt`, `kDoorExLf`, `kWindowInLf`, `kWindowInRt`, `kWindowExRt`, `kWindowExLf` | 14 | `data.d.tall`, `.where`, `.who`, `.wide` |
| 8 | `:1277`–`1284` | `kInvisTrans`, `kDeluxeTrans` | 2 | `data.d.topLeft.h = temp.h + 64`, `.topLeft.v`, `.tall`, `.where`, `.who`, `.wide` |
| 9 | `:1286`–`1293` | `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch`, `kInvisSwitch`, `kTrigger`, `kLgTrigger`, `kSoundTrigger` | 9 | `data.e.delay`, `.where`, `.who`, `.type` |
| 10 | `:1295`–`1303` | `kCeilingLight`, `kLightBulb`, `kTableLamp`, `kHipLamp`, `kDecoLamp`, `kFlourescent`, `kTrackLight`, `kInvisLight` | 8 | `data.f.length`, `.byte0`, `.byte1`, `.initial`, `.state` |
| 11 | `:1305`–`1315` | `kShredder`, `kToaster`, `kMacPlus`, `kGuitar`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`, `kMicrowave`, `kCinderBlock`, `kFlowerBox`, `kCDs` | 13 | `data.g.height`, `.byte0`, `.delay`, `.initial`, `.state` |
| 12 | `:1317`–`1326` | `kCustomPict` | 1 | `data.g.topLeft.h = temp.h + 64`, `.topLeft.v`, `.height`, `.byte0`, `.delay`, `.initial`, `.state` |
| 13 | `:1328`–`1336` | `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish`, `kCobweb` | 9 | `data.h.length`, `.delay`, `.byte0`, `.initial`, `.state` |
| 14 | `:1338`–`1343` | `kOzma`, `kMousehole`, `kFireplace`, `kBear`, `kCalendar`, `kVase1`, `kVase2`, `kBulletin`, `kCloud`, `kFaucet`, `kRug`, `kChimes` | 12 | `data.i.pict` only |
| 15 | `:1345`–`1349` | `kMirror`, `kFlower`, `kWallWindow` | 3 | `data.i.bounds` (whole rect), then `QOffsetRect(&bounds, 64, 0)`, then `data.i.pict` |

15+1+7+8+14+1+14+2+9+8+13+1+9+12+3 = **117**. ✅

Watch the arm memberships — they are counter-intuitive:

* **`kManhole` is in arm 3** (pict-only, no bounds copy) even though `GetObjectRect` derives its
  rect from `data.b.bounds`. The duplicate therefore gets `AddNewObject`'s freshly computed bounds,
  which `KeepObjectLegal` then snaps onto the 64 px grid (§12.2). That is correct behaviour, not a
  bug — a manhole's bounds are fully determined by its grid cell.
* **`kCounter` and `kDresser` ARE in arm 4** (bounds copy + 64 offset), even though `DragObject`'s
  commit switch groups them with `kManhole` (arm 4 of §5.4). The three groupings across the file are
  genuinely different partitions of the same types; do not assume one implies another.
* **`kSlider` (arm 6) copies `topLeft.h` but not `topLeft.v`.** So a duplicated slider keeps
  `AddNewObject`'s vertical placement (from `placePt.v`) but gets the original's horizontal + 64.
  Every other position-copying arm sets both. Almost certainly an oversight, but harmless because
  `placePt.v` was derived from the original's centre anyway.
* **`kFlower` (0x85) is in arm 15**, the bounds-copy clutter arm. That is what makes duplication
  safe for it despite `srcRects[kFlower]` being **never initialised** — the arm copies the stored
  `data.i.bounds` and never consults the size table. See §13.7.

### 8.4 What is *not* copied

For the 9 arms that omit position (1, 3, 5, 7, 9, 10, 11, 13, 14 — covering
15 + 7 + 14 + 14 + 9 + 8 + 13 + 9 + 12 = **101 of 117** types), the duplicate's position comes
entirely from `AddNewObject(placePt, …)`. `placePt` is `original centre + (64, 0)`, so the
duplicate lands one tile right — the same net result as the arms that do the `+64` themselves.

`data.*.pict` is copied for the four pict-bearing unions (arms 3, 4, 14, 15) but **not** for
`kCustomPict`, where the PICT resource id lives in `data.g.height` (§12.9) and is copied by arm 12.

`objectType.what` is not copied because `AddNewObject` set it.

**Link fields.** `data.d.where`/`.who` (transports) and `data.e.where`/`.who` (switches) **are**
copied verbatim (arms 7, 8, 9). So a duplicated transport points at the same destination room and
object, and a duplicated switch drives the same target. But the *reverse* link list
(`retroLinkList`, see §9) is **not** updated for the new object. `AddObjectPairing`
(`ObjectEdit.c:792`–`1150`) is what normally maintains pairings, and `DuplicateObject` never calls
it — only `DoNewObjectClick` does (`ObjectEdit.c:786`). A port must decide whether to fix that; the
original leaves the duplicate's back-link dangling.

### 8.5 Order of operations in the tail

`ObjectEdit.c:1352`–`1368`. Note the ordering differs from `DragObject`/`MoveObject`:

| Step | `DragObject` (`:747`–`767`) | `MoveObject` (`:1700`–`1769`) | `DuplicateObject` (`:1352`–`1368`) |
|---|---|---|---|
| 1 | `KeepObjectLegal()` | `KeepObjectLegal()` | `KeepObjectLegal()` |
| 2 | `GetThisRoomsObjRects()` | `fileDirty`/`UpdateMenus` | `ObjectHasHandle()` |
| 3 | invalidate | `GetThisRoomsObjRects()` | `ReadyBackground()` |
| 4 | `ReadyBackground()` | invalidate | `GetThisRoomsObjRects()` |
| 5 | `DrawThisRoomsObjects()` | `ReadyBackground()` | `DrawThisRoomsObjects()` |
| 6 | — | `DrawThisRoomsObjects()` | invalidate (whole window) |
| 7 | — | re-arm marquee | re-arm marquee |

`DuplicateObject` calls `ObjectHasHandle` **before** `GetThisRoomsObjRects` (step 2 vs step 4),
which is fine because `ObjectHasHandle` reads the *object's* fields, not the cache. But it then
passes `&roomObjectRects[objActive]` to `StartMarqueeHandled` at step 7 — reading the cache after
it has been rebuilt. Correct, but only by luck of ordering.

`DuplicateObject` notably **does not set `fileDirty`**. Adding an object via `AddNewObject` does
(inside `ObjectAdd.c`), so the house is still marked dirty; but if `AddNewObject` succeeded and the
copy-switch then changed fields, those changes alone would not have dirtied the file. Since
`AddNewObject` succeeded is the precondition for reaching the switch, `fileDirty` is always already
true. Benign.

`DuplicateObject` always does a **whole-window** invalidate (`ObjectEdit.c:1360`) — no per-type
optimisation.

---

## 9. Gesture 5 — delete (`DeleteObject`)

`GliderPRO/Sources/ObjectEdit.c:1154`–`1187`. 34 lines. Complete, in order:

```
 1.  if ((theMode != kEditMode) || (objActive == kNoObjectSelected))
         return                                                       /* :1159-1160 */
 2.  if ((objActive == kInitialGliderSelected) ||
         (objActive == kLeftGliderSelected)    ||
         (objActive == kRightGliderSelected))
     {
         SysBeep(1)                                                   /* :1166 */
         return
     }
 3.  for (i = 0; i < kMaxRoomObs /*24*/; i++)                         /* :1170-1175 */
         if ((retroLinkList[i].room   == thisRoomNumber) &&
             (retroLinkList[i].object == objActive))
             retroLinkList[i].room = -1
 4.  thisRoom->objects[objActive].what = kObjectIsEmpty  /* -1 */      /* :1177 */
 5.  thisRoom->numObjects--                                           /* :1178 */
 6.  fileDirty = true                                                 /* :1179 */
 7.  UpdateMenus(false)                                               /* :1180 */
 8.  InvalWindowRect(mainWindow, &mainWindowRect)        /* whole window */  /* :1181 */
 9.  QSetRect(&roomObjectRects[objActive], -1, -1, 0, 0)              /* :1182 */
10.  DeselectObject()                                                 /* :1183 */
11.  ReadyBackground(thisRoom->background, thisRoom->tiles)           /* :1184 */
12.  DrawThisRoomsObjects()                                           /* :1185 */
```

### 9.1 Points a port must get exactly right

**Both guards.** Step 1 checks `theMode` **and** `objActive == kNoObjectSelected`; step 2 refuses
the three glider pseudo-selections with a **`SysBeep(1)`**. So the glider markers cannot be deleted,
and the refusal is audible. (Compare `DuplicateObject`, §8.1, which has *no* guard at all.)

**`numObjects` is decremented but the slot is not compacted.** Step 4 marks the slot
`kObjectIsEmpty` in place; step 5 decrements the count. So after deleting a middle object,
`numObjects` no longer equals the highest occupied index + 1. Consequences:

* `FindObjectSelected` loops `i` from `kMaxRoomObs - 1` = 23 down (`ObjectEdit.c:59`, §3.1), so
  the orphaned high slots **remain clickable**. Delete object 0 of a 3-object room and object 2 is
  still hit-testable, still drawn, and still cached — the array is simply sparse. This is why the
  missing compaction is survivable in the original.
* Consequently `numObjects` is used for exactly two things: the Tab-cycle guard
  (`numObjects <= 0`, `ObjectEdit.c:1988`/`:2021`) and `AddNewObject`'s free-slot bookkeeping.
  **Nothing in the program uses it as an index bound** (§19.3).
* `numObjects` can go **negative** if `DeleteObject` is somehow called more times than there are
  objects. `SelectNextObject`/`SelectPrevObject` guard on `numObjects <= 0` (`ObjectEdit.c:1988`,
  `:2021`), so a negative count disables Tab.

A Go port should **not** compact: slot indices are also **link addresses** (§13.1), so compacting
would break `data.d.who` / `data.e.who` / `retroLinkList`. Make `numObjects` a derived value —
count the non-empty slots — and keep every loop at `0 .. 23` with a `what != kObjectIsEmpty` test,
which is what the original already does everywhere except in its own count bookkeeping.

**`roomObjectRects[objActive]` is set to `(-1, -1, 0, 0)`**, not zeroed and not left alone. This is
a 1 × 1 rect at `(-1, -1)`, deliberately outside any plausible click. It matters because step 9
runs *before* the next `GetThisRoomsObjRects()` — and `DeleteObject` **never calls
`GetThisRoomsObjRects()`**. The cache stays stale for every other slot until something else rebuilds
it. Since delete does not move anything else, that is fine.

**`retroLinkList` cleanup.** Step 3 walks a fixed 24-entry `retroLinkList` and clears any entry
whose `(room, object)` matches the deleted object, by setting `room = -1`. This is the reverse-link
index used by the linking UI (`GliderPRO/Sources/Link.c`). Note it does **not** clear forward links:
any *other* object whose `data.d.who` / `data.e.who` pointed at the deleted slot still points there,
now at an empty slot. `HouseLegal.c` repairs dangling links on the next load.

**No `GetThisRoomsObjRects()` and a whole-window invalidate.** Steps 8, 11, 12 recomposite the
whole room. The cache is *not* rebuilt, so `DrawThisRoomsObjects` (which reads the cache) draws the
remaining objects from their existing — still valid — cached rects, and skips the deleted one because
its `what` is now `kObjectIsEmpty`.

---

## 10. Gesture 6 — tab-cycle (`SelectNextObject` / `SelectPrevObject`)

`GliderPRO/Sources/ObjectEdit.c:1982`–`2011` and `:2015`–`2044`. Near-mirror images.

```
SelectNextObject():                                    SelectPrevObject():
 1. if (theMode != kEditMode) ||                        1. if (theMode != kEditMode) ||
       (thisRoom->numObjects <= 0): return                    (thisRoom->numObjects <= 0): return
 2. noneFound = true                                    2. noneFound = true
 3. while (noneFound):                                  3. while (noneFound):
      objActive++                                            objActive--
      if (objActive >= kMaxRoomObs) objActive = 0             if (objActive < 0) objActive = kMaxRoomObs - 1
      if (objects[objActive].what != kObjectIsEmpty)          if (objects[objActive].what != kObjectIsEmpty)
          noneFound = false                                       noneFound = false
 4. UpdateMenus(false)                                  4. UpdateMenus(false)
 5. if ObjectHasHandle(&direction, &dist):              5. (identical)
      StartMarqueeHandled(&roomObjectRects[objActive],
                          direction, dist)
      HandleBlowerGlider()
    else:
      StartMarquee(&roomObjectRects[objActive])
```

### 10.1 Which fields does tab-cycle write?

**None.** Tab-cycle writes exactly one variable: `objActive` (`ObjectEdit.c:1995` / `:2028`). It
does not touch any `objectType` field, does not set `fileDirty`, does not rebuild the cache, does
not recomposite, and does not invalidate. It only re-arms the marquee. This is the answer to the
assignment's "which `objectType` fields does tab-cycle write" — the answer is the empty set.

Note the corollary: because tab-cycle does not rebuild `roomObjectRects`, it arms the marquee from
whatever the cache currently holds. If the cache is stale the ants appear in the wrong place. In
practice every mutating gesture rebuilds the cache before returning, so it is not stale.

### 10.2 It iterates all 24 slots, not `numObjects`

The wrap is at `kMaxRoomObs` = 24 / 0, not at `numObjects` — `numObjects` appears only in the
entry guard (`> 0`). So tab-cycle reaches the "orphaned" high slots a delete can leave above
`numObjects` (§9.1). So does clicking: `FindObjectSelected` also iterates all 24 (§3.1). The two
selection paths agree; `numObjects` bounds neither.

### 10.3 Two infinite loops

**Loop 1 — inconsistent `numObjects`.** The `while (noneFound)` loop has **no iteration bound**.
If `thisRoom->numObjects > 0` but all 24 slots have `what == kObjectIsEmpty`, the loop spins
forever: `objActive` cycles 0→23→0 and the exit condition is never met. The program hangs (no
`WaitNextEvent`, so it is a hard freeze requiring force-quit).

This state is reachable: `AddNewObject` increments `numObjects`, `DeleteObject` decrements it, but
`HouseLegal.c`'s repair passes and the room-copy/paste code in `Scrap.c` also touch slots. Any path
that clears a slot without decrementing produces it. A port **must** bound the loop:

```
for tries := 0; tries < kMaxRoomObs; tries++ { ... }
if noneFound { objActive = kNoObjectSelected; return }
```

**Loop 2 — starting from a glider pseudo-selection.** Suppose `objActive == kInitialGliderSelected`
(−2) and the user presses Tab. Then:

```
objActive++          -> -1
objActive >= 24?     -> no
objects[-1].what != kObjectIsEmpty ?
```

`objects[-1].what` is at room offset 48 = `tiles[6]` (§7.2). It is `kObjectIsEmpty` (−1) only if
`tiles[6] == -1`.

> **Empirically `tiles[6]` is NEVER −1** across all 4070 non-empty rooms of all 22 shipped houses
> (distribution in §7.2: values 0..7 only). So the test *always* succeeds, `noneFound` becomes
> `false`, and the loop exits with `objActive == -1`.

`objActive == -1` is `kNoObjectSelected`. Then step 5:

* `ObjectHasHandle` reads `thisRoom->objects[-1].what` → a value in 0..7 → for 1..7 it is a valid
  blower `what`, so it may return `true` and set `dist` from `objects[-1].data.a.distance` = room
  offset 54 = `suite` (`roomType` offset 54).
* `StartMarqueeHandled(&roomObjectRects[-1], …)` — but `StartMarqueeHandled` **early-outs at
  `Marquee.c:81`–`82` because `objActive == kNoObjectSelected`**. Same for `StartMarquee`
  (`Marquee.c:59`–`60`).

**Net effect: pressing Tab with a glider marker selected silently drops the selection.** No crash,
no marquee, no beep. The `roomObjectRects[-1]` OOB read happens (8 bytes before the array in BSS)
but its value is discarded because the callee early-outs before reading `*theRect`... except that C
evaluates `&roomObjectRects[-1]` as an address computation only, so nothing is actually
dereferenced. Genuinely harmless, by accident.

Similarly `SelectPrevObject` from −2: `objActive--` → −3, `objActive < 0` → `objActive = 23`, then
it searches downward from 23. So **Shift-Tab from a glider marker works** (it lands on a real
object) while **Tab silently deselects**. Asymmetric.

A port should special-case `objActive < 0` at the top of both functions: treat it as "start from
−1 for next, from 24 for prev".

### 10.4 Menu wiring

`Menu.c` maps Tab / Shift-Tab and the Edit-menu items to these. `UpdateMenus(false)` at
`ObjectEdit.c:2002` / `:2035` refreshes the enable state of Cut/Copy/Clear/Duplicate/Object Info
based on the new `objActive`.

---

## 11. Master field-write matrix

### 11.1 `objectType` layout, byte by byte

`GliderPRO/Headers/GliderStructs.h:90`–`105`:

```c
typedef struct
{
    short   what;                    // 2
    union {
        blowerType      a;
        furnitureType   b;
        bonusType       c;
        transportType   d;
        switchType      e;
        lightType       f;
        applianceType   g;
        enemyType       h;
        clutterType     i;
    } data;                          // 10
} objectType, *objectPtr;            // total = 12
```

**`sizeof(objectType)` = 12 bytes exactly**, big-endian on disk, no padding. `Point` on classic Mac
is `struct { short v; short h; }` — **v first**. That ordering is essential: a Go port that defines
`Point{H, V int16}` and reads it raw will swap every coordinate in every house file.

| Offset | Size | Field |
|---:|---:|---|
| 0 | 2 | `what` |
| 2 | 10 | `data` (union) |

The nine union arms, all exactly 10 bytes
(`GliderPRO/Headers/GliderStructs.h:11`–`88`):

**`blowerType a`** (`:11`–`19`) — offsets relative to `objectType`:

| Rel. off | Abs. off | Size | Field | Type |
|---:|---:|---:|---|---|
| 0 | 2 | 4 | `topLeft` | `Point` (`v` @2, `h` @4) |
| 4 | 6 | 2 | `distance` | `short` |
| 6 | 8 | 1 | `initial` | `Boolean` |
| 7 | 9 | 1 | `state` | `Boolean` |
| 8 | 10 | 1 | `vector` | `Byte` — bit field `| x | x | x | x | 8=lf | 4=dn | 2=rt | 1=up |` (comment at `:16`–`:17`) |
| 9 | 11 | 1 | `tall` | `Byte` |

**`furnitureType b`** (`:21`–`25`):

| Rel. off | Abs. off | Size | Field | Type |
|---:|---:|---:|---|---|
| 0 | 2 | 8 | `bounds` | `Rect` (`top`,`left`,`bottom`,`right` — **that order**) |
| 8 | 10 | 2 | `pict` | `short` |

**`bonusType c`** (`:27`–`34`):

| Rel. off | Abs. off | Size | Field | Type | Comment |
|---:|---:|---:|---|---|---|
| 0 | 2 | 4 | `topLeft` | `Point` | |
| 4 | 6 | 2 | `length` | `short` | "grease spill" |
| 6 | 8 | 2 | `points` | `short` | "invis bonus" |
| 8 | 10 | 1 | `state` | `Boolean` | |
| 9 | 11 | 1 | `initial` | `Boolean` | note `state` **before** `initial`, unlike `blowerType` |

**`transportType d`** (`:36`–`43`):

| Rel. off | Abs. off | Size | Field | Type | Comment |
|---:|---:|---:|---|---|---|
| 0 | 2 | 4 | `topLeft` | `Point` | |
| 4 | 6 | 2 | `tall` | `short` | "invis transport"; also the packed W/H for `kDeluxeTrans` (§6.3) |
| 6 | 8 | 2 | `where` | `short` | destination room |
| 8 | 10 | 1 | `who` | `Byte` | destination object slot |
| 9 | 11 | 1 | `wide` | `Byte` | **unsigned**, so `HouseLegal.c:379`'s `< 0` test is dead |

**`switchType e`** (`:45`–`52`):

| Rel. off | Abs. off | Size | Field | Type |
|---:|---:|---:|---|---|
| 0 | 2 | 4 | `topLeft` | `Point` |
| 4 | 6 | 2 | `delay` | `short` |
| 6 | 8 | 2 | `where` | `short` |
| 8 | 10 | 1 | `who` | `Byte` |
| 9 | 11 | 1 | `type` | `Byte` — one of `kToggle` 0, `kForceOn` 1, `kForceOff` 2, `kOneShot` 3 |

**`lightType f`** (`:54`–`62`):

| Rel. off | Abs. off | Size | Field | Type |
|---:|---:|---:|---|---|
| 0 | 2 | 4 | `topLeft` | `Point` |
| 4 | 6 | 2 | `length` | `short` |
| 6 | 8 | 1 | `byte0` | `Byte` |
| 7 | 9 | 1 | `byte1` | `Byte` |
| 8 | 10 | 1 | `initial` | `Boolean` |
| 9 | 11 | 1 | `state` | `Boolean` |

**`applianceType g`** (`:64`–`72`):

| Rel. off | Abs. off | Size | Field | Type | Comment |
|---:|---:|---:|---|---|---|
| 0 | 2 | 4 | `topLeft` | `Point` | |
| 4 | 6 | 2 | `height` | `short` | "toaster, **pict ID**" — dual-purpose, see §12.9 |
| 6 | 8 | 1 | `byte0` | `Byte` | |
| 7 | 9 | 1 | `delay` | `Byte` | |
| 8 | 10 | 1 | `initial` | `Boolean` | |
| 9 | 11 | 1 | `state` | `Boolean` | |

**`enemyType h`** (`:74`–`82`):

| Rel. off | Abs. off | Size | Field | Type |
|---:|---:|---:|---|---|
| 0 | 2 | 4 | `topLeft` | `Point` |
| 4 | 6 | 2 | `length` | `short` |
| 6 | 8 | 1 | `delay` | `Byte` |
| 7 | 9 | 1 | `byte0` | `Byte` | note `delay` **before** `byte0`, the reverse of `applianceType` |
| 8 | 10 | 1 | `initial` | `Boolean` |
| 9 | 11 | 1 | `state` | `Boolean` |

**`clutterType i`** (`:84`–`88`): identical layout to `furnitureType b` — `Rect bounds` @0,
`short pict` @8.

`roomType` (`GliderPRO/Headers/GliderStructs.h:166`–`180`), for the negative-index analysis of
§7.2 and §10.3:

| Offset | Size | Field |
|---:|---:|---|
| 0 | 28 | `Str27 name` |
| 28 | 2 | `short bounds` |
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
| 60 | 288 | `objectType objects[24]` |
| — | **348** | total |

### 11.2 The matrix: six gestures × nine unions

Legend: **T** = translates the existing value (`+=`); **A** = assigns a new value; **C** = copies from
the source object; **—** = never written.

| Union / field | drag-move (§5.4) | resize (§6.1) | nudge (§7.6) | duplicate (§8.3) | delete (§9) | tab (§10) |
|---|---|---|---|---|---|---|
| `what` | — | — | — | set by `AddNewObject` | **A** `kObjectIsEmpty` | — |
| `a.topLeft.h` | **T** (16 types) | — | **T** (16 types) | **C+64** (`kLiftArea` only) | — | — |
| `a.topLeft.v` | **T** (9 types) | — | **T** (9 types) | **C** (`kLiftArea` only) | — | — |
| `a.distance` | — | **A** (16 types) | — | **C** (16 types) | — | — |
| `a.initial` | — | — | — | **C** (16 types) | — | — |
| `a.state` | — | — | — | **C** (16 types) | — | — |
| `a.vector` | — | — | — | **C** (16 types) | — | — |
| `a.tall` | — | **A** (`kLiftArea`: `vDelta/2`) | — | **C** (16 types) | — | — |
| `b.bounds.left` | **T** (15 types) | — | **T** (15 types) | **C+64** (8 types) | — | — |
| `b.bounds.right` | **T** (15 types) | **A** (`left + hDelta`, 8 types) | **T** (15 types) | **C+64** (8 types) | — | — |
| `b.bounds.top` | **T** (12 types) | **A** (`bottom - vDelta`, `kCounter`/`kDresser`) | **T** (12 types) | **C** (8 types) | — | — |
| `b.bounds.bottom` | **T** (12 types) | **A** (`top + vDelta`, 3 types) | **T** (12 types) | **C** (8 types) | — | — |
| `b.pict` | — | — | — | **C** (15 types) | — | — |
| `c.topLeft.h` | **T** (15 types) | — | **T** (15 types) | **C+64** (`kSlider` only) | — | — |
| `c.topLeft.v` | **T** (15 types) | — | **T** (15 types) | — | — | — |
| `c.length` | — | **A** (`kGreaseRt`/`kGreaseLf`/`kSlider`) | — | **C** (15 types) | — | — |
| `c.points` | — | — | — | **C** (15 types) | — | — |
| `c.state` | — | — | — | **C** (15 types) | — | — |
| `c.initial` | — | — | — | **C** (15 types) | — | — |
| `d.topLeft.h` | **T** (16 types) | — | **T** (**8 types only** — doors/windows omitted) | **C+64** (`kInvisTrans`/`kDeluxeTrans`) | — | — |
| `d.topLeft.v` | **T** (4 types) | — | **T** (4 types) | **C** (`kInvisTrans`/`kDeluxeTrans`) | — | — |
| `d.tall` | — | **A** (`kInvisTrans`, `kDeluxeTrans` packed) | — | **C** (16 types) | — | — |
| `d.where` | — | — | — | **C** (16 types) | — | — |
| `d.who` | — | — | — | **C** (16 types) | — | — |
| `d.wide` | — | **A** (`kInvisTrans`, clamped ≤ 127) | — | **C** (16 types) | — | — |
| `e.topLeft.h` | **T** (9 types) | — | **T** (9 types) | — | — | — |
| `e.topLeft.v` | **T** (9 types) | — | **T** (9 types) | — | — | — |
| `e.delay` | — | — | — | **C** (9 types) | — | — |
| `e.where` | — | — | — | **C** (9 types) | — | — |
| `e.who` | — | — | — | **C** (9 types) | — | — |
| `e.type` | — | — | — | **C** (9 types) | — | — |
| `f.topLeft.h` | **T** (8 types) | — | **T** (8 types) | — | — | — |
| `f.topLeft.v` | **T** (3 types) | — | **T** (3 types) | — | — | — |
| `f.length` | — | **A** (`kFlourescent`/`kTrackLight`) | — | **C** (8 types) | — | — |
| `f.byte0` | — | — | — | **C** (8 types) | — | — |
| `f.byte1` | — | — | — | **C** (8 types) | — | — |
| `f.initial` | — | — | — | **C** (8 types) | — | — |
| `f.state` | — | — | — | **C** (8 types) | — | — |
| `g.topLeft.h` | **T** (14 types) | — | **T** (14 types) | **C+64** (`kCustomPict` only) | — | — |
| `g.topLeft.v` | **T** (14 types) | — | **T** (14 types) | **C** (`kCustomPict` only) | — | — |
| `g.height` | — | **A** (`kToaster`) | — | **C** (14 types) | — | — |
| `g.byte0` | — | — | — | **C** (14 types) | — | — |
| `g.delay` | — | — | — | **C** (14 types) | — | — |
| `g.initial` | — | — | — | **C** (14 types) | — | — |
| `g.state` | — | — | — | **C** (14 types) | — | — |
| `h.topLeft.h` | **T** (7 types) | — | **T** (7 types) | — | — | — |
| `h.topLeft.v` | **T** (6 types) | — | **T** (6 types) | — | — | — |
| `h.length` | — | **A** (`kBall`/`kDrip`/`kFish`) | — | **C** (9 types) | — | — |
| `h.delay` | — | — | — | **C** (9 types) | — | — |
| `h.byte0` | — | — | — | **C** (9 types) | — | — |
| `h.initial` | — | — | — | **C** (9 types) | — | — |
| `h.state` | — | — | — | **C** (9 types) | — | — |
| `i.bounds.left` | **T** (15 types) | — | **T** (15 types) | **C+64** (3 types) | — | — |
| `i.bounds.right` | **T** (15 types) | **A** (`left + hDelta`, `kMirror`/`kWallWindow`) | **T** (15 types) | **C+64** (3 types) | — | — |
| `i.bounds.top` | **T** (13 types) | — | **T** (13 types) | **C** (3 types) | — | — |
| `i.bounds.bottom` | **T** (13 types) | **A** (`top + vDelta`, `kMirror`/`kWallWindow`) | **T** (13 types) | **C** (3 types) | — | — |
| `i.pict` | — | — | — | **C** (15 types) | — | — |

### 11.3 Non-object state each gesture writes

| Gesture | `objActive` | `numObjects` | `fileDirty` | `retroLinkList` | house header | room `leftStart`/`rightStart` | `roomObjectRects` |
|---|---|---|---|---|---|---|---|
| drag-move | — | — | only if delta ≠ 0 (`:525`) | — | `initial.h`/`.v` (`:533`–`534`) | `leftStart` (`:544`) / `rightStart` (`:556`) | rebuilt (`:750`) |
| resize | — | — | **always** (`:343`) | — | — | — | rebuilt in 9 of 14 arms |
| nudge | — | — | **always** (`:1703`) | — | `initial.h`/`.v` (`:1490`–`1491`) | `leftStart` (`:1502`) / `rightStart` (`:1515`) | rebuilt (`:1705`) |
| duplicate | set by `AddNewObject` (`ObjectAdd.c:57`) | ++ by `AddNewObject` | by `AddNewObject` | **not** updated | — | — | rebuilt (`:1358`) |
| delete | `kNoObjectSelected` (via `DeselectObject`, `:1183`) | **--** (`:1178`) | yes (`:1179`) | matching entries `room = -1` (`:1174`) | — | — | **not** rebuilt; slot set to `(-1,-1,0,0)` (`:1182`) |
| tab-cycle | ±1, wrapped (`:1995`/`:2028`) | — | **no** | — | — | — | **not** rebuilt |

---

## 12. Clamping and snapping: `KeepObjectLegal`

`GliderPRO/Sources/HouseLegal.c:42`–`597`. 556 lines, one switch on `theObject->what` with
**117** case labels — the same complete set as `DragObject`'s and `DuplicateObject`'s switches
(verified set-equal). This is the universal post-edit normaliser: every one of the five mutating
gestures calls it exactly once before redrawing.

### 12.1 Contract and return value

```c
Boolean KeepObjectLegal (void)
```

Operates on the currently selected object (`thisRoom->objects[objActive]`, taken via a local
pointer `theObject`) and returns `unchanged` — `true` if it made **no** modification.

Every caller **discards the return value**:

| Caller | Site | Form |
|---|---|---|
| `DragHandle` × 14 arms | `ObjectEdit.c:165`, `:174`, `:186`, `:203`, `:213`, `:230`, `:246`, `:259`, `:274`, `:290`, `:302`, `:313`, `:322`, `:334` | `whoCares = KeepObjectLegal();` |
| `DragObject` | `ObjectEdit.c:747` | `if (KeepObjectLegal()) { }` |
| `DuplicateObject` | `ObjectEdit.c:1352` | `if (KeepObjectLegal()) { }` |
| `MoveObject` | `ObjectEdit.c:1700` | `if (KeepObjectLegal()) { }` |

So the flag is vestigial — and **nine sites mutate the object without clearing it**:

| # | Site | What it mutates | `unchanged = false`? |
|---:|---|---|:---:|
| 1 | `:55`–`:69` `kInitialGliderSelected` clamp | `(*thisHouse)->initial.h/.v` | no — and it `return`s **`true`** |
| 2 | `:115`–`:119` `kFloorVent` forced v | `data.a.topLeft.v`, `data.a.distance` | **no** |
| 3 | `:120`–`:125` `kFloorBlower` forced v | `data.a.topLeft.v`, `data.a.distance` | **no** |
| 4 | `:126`–`:131` `kSewerGrate` forced v | `data.a.topLeft.v`, `data.a.distance` | **no** |
| 5 | `:132`–`:137` `kFloorTrans` forced v | `data.a.topLeft.v`, `data.a.distance` | **no** (dead code, §12.4) |
| 6 | `:310`–`:324` `kDoorIn*` side flip | `data.d.topLeft.h`, **`what`** | **no** |
| 7 | `:325`–`:339` `kDoorEx*` side flip | `data.d.topLeft.h`, **`what`** | **no** |
| 8 | `:340`–`:354` `kWindowIn*` side flip | `data.d.topLeft.h`, **`what`** | **no** |
| 9 | `:355`–`:369` `kWindowEx*` side flip | `data.d.topLeft.h`, **`what`** | **no** |

Since nobody reads the result, it is harmless in the shipped program, but a port that *does* use the
return value (e.g. to skip a redraw, or to decide whether to set `fileDirty`) will get wrong answers
for `kFloorVent`, `kFloorBlower`, `kSewerGrate`, all eight door/window types, and the initial-glider
marker.

**Structure**, in order:

```
KeepObjectLegal():                                       /* HouseLegal.c:42-597 */
 1.  unchanged = true                                    /* :50 */
     #ifndef COMPILEDEMO                                 /* :51  — whole body is stubbed in the demo */
 2.  theObject = &thisRoom->objects[objActive]           /* :53  — computed BEFORE any range check */
 3.  if (objActive == kInitialGliderSelected):           /* :55-69 — the ONLY sentinel handled */
        HGetState/HLock (Handle)thisHouse                /* :57-58 */
        if (initial.h < 0)  initial.h = 0                /* :59-60 */
        if (initial.v < 0)  initial.v = 0                /* :61-62 */
        if (initial.h > kRoomWide - kGliderWide) initial.h = kRoomWide - kGliderWide  /* :63-64  = 464 */
        if (initial.v > kTileHigh - kGliderHigh) initial.v = kTileHigh - kGliderHigh  /* :65-66  = 302 */
        HSetState; return TRUE                           /* :67-68 — returns "unchanged" even if it changed! */
 4.  QSetRect(&roomRect, 0, 0, kRoomWide, kTileHigh)      /* :71  = (0,0,512,322) */
 5.  switch (theObject->what) {                           /* :73  — 9 union-class arms, 117 labels */
        each arm:
          a. GetObjectRect(&thisRoom->objects[objActive], &bounds)
          b. if (ForceRectInRect(&bounds, &roomRect)) { write topLeft/bounds back; unchanged = false }
          c. per-type extent clamps
          d. per-type parity / grid forcing
      }
 6.  #endif; return unchanged                            /* :594-596 */
```

Two critical ordering points:

* **Containment (step 5b) runs BEFORE the per-type parity/grid forcing (step 5d)** in every arm.
  §12.10 shows that ordering produces one observable defect in the shipped data.
* **Step 2 dereferences `objects[objActive]` with no range check.** For `objActive == -1`
  (`kNoObjectSelected`) the pointer is merely computed, but step 5 then reads
  `theObject->what` — an out-of-bounds read. §12.11 works this out against real bytes and shows
  it is a reachable memory-corruption bug for `kLeftGliderSelected` (−3).

`ForceRectInRect` (`GliderPRO/Sources/RectUtils.c:223`–`270`):

```
ForceRectInRect(theRect, bounds) -> Boolean changed:
 1. changed = false
 2. NormalizeRect(theRect)                              /* :230 */
 3. if (RectTall(theRect) > RectTall(bounds)):           /* :232-236 shrink height */
        theRect->bottom = theRect->top + RectTall(bounds);  changed = true
 4. if (RectWide(theRect) > RectWide(bounds)):           /* :238-242 shrink width */
        theRect->right = theRect->left + RectWide(bounds); changed = true
 5. if (theRect->left   < bounds->left):   OffsetRect(theRect, bounds->left - theRect->left, 0);   changed = true   /* :244-249 */
 6. if (theRect->right  > bounds->right):  OffsetRect(theRect, bounds->right - theRect->right, 0); changed = true   /* :250-255 */
 7. if (theRect->top    < bounds->top):    OffsetRect(theRect, 0, bounds->top - theRect->top);     changed = true   /* :256-261 */
 8. if (theRect->bottom > bounds->bottom): OffsetRect(theRect, 0, bounds->bottom - theRect->bottom); changed = true /* :262-267 */
 9. return changed                                      /* :269 */
```

Note it **shrinks before it translates**, and it translates (never clips) — the object keeps its
size and slides inside. The room rect is `(0, 0, kRoomWide, kTileHigh)` = `(0, 0, 512, 322)`.

### 12.2 Parity and grid forcing — the complete table

| Types | Rule | Lines |
|---|---|---|
| `kStubby` 0x0A | `topLeft.h` forced **ODD**: `if (h % 2 == 0) h--` | `HouseLegal.c:103`–`108` |
| `kTaper` 0x08, `kCandle` 0x09, `kTiki` 0x0B, `kBBQ` 0x0C | `topLeft.h` forced **EVEN**: `if (h % 2 != 0) h--` | `HouseLegal.c:109`–`114` |
| all 15 bonuses `kRedClock` 0x21 … `kSlider` 0x2F | `topLeft.h` forced **EVEN** | `HouseLegal.c:269`–`273` |
| all bonuses **except `kStar`** 0x2C | `data.c.length` forced **EVEN** | `HouseLegal.c:274`–`279` |
| all 9 switches `kLightSwitch` 0x41 … `kSoundTrigger` 0x49 | `topLeft.h` forced **EVEN** | `HouseLegal.c:403`–`407` |
| `kTV` 0x65 | `topLeft.h` forced **ODD** | `HouseLegal.c:494`–`499` |
| `kToaster` 0x62, `kMacPlus` 0x63, `kCoffee` 0x66, `kOutlet` 0x67, `kVCR` 0x68, `kStereo` 0x69, `kMicrowave` 0x6A | `topLeft.h` forced **EVEN** | `HouseLegal.c:500`–`511` |
| `kBalloon` 0x71, `kCopterLf` 0x72, `kCopterRt` 0x73, `kBall` 0x76, `kDrip` 0x77, `kFish` 0x78 | `topLeft.h` forced **EVEN** | `HouseLegal.c:543`–`553` |
| `kMirror` 0x82 | `bounds.left` **and** `bounds.right` forced **EVEN** | `HouseLegal.c:577`–`589` |
| `kManhole` 0x1D | `bounds.left` snapped to the 64 px grid at phase 3: `(left - 3) % 64 == 0`; width forced to **123** | `HouseLegal.c:217`–`226` |

The forcing always **decrements** (`h--`), never rounds to nearest. So an object nudged from an even
x to an odd x by a 1 px move snaps back down, i.e. net movement 0 — which is exactly why
`MoveObject` uses `increment = 2` for these 44 types (§7.3).

`kStubby` and `kTV` are the only two **odd**-forced types. Every other parity rule forces even.

> **Empirical verification** — all 22 shipped houses, 31 440 objects:
>
> | Invariant | tested | held | violated |
> |---|---:|---:|---:|
> | `odd-x kStubby` | 127 | 127 | **0** |
> | `odd-x kTV` | 80 | 80 | **0** |
> | `even-x blower4` (`kTaper`/`kCandle`/`kTiki`/`kBBQ`) | 371 | 371 | **0** |
> | `even-x bonus` | 2992 | 2992 | **0** |
> | `even-length bonus` (excl. `kStar`) | 2923 | 2923 | **0** |
> | `even-x switch` | 1685 | 1685 | **0** |
> | `even-x appliance7` | 470 | 470 | **0** |
> | `even-x enemy6` | 1723 | 1723 | **0** |
> | `even-x kMirror.left` | 667 | 667 | **0** |
> | `even-x kMirror.right` | 667 | 667 | **0** |
> | `kManhole (left-3)%64==0` | 40 | 40 | **0** |
> | `kManhole wide==123` | 40 | 40 | **0** |
>
> Every parity and grid rule holds for every shipped object. That is strong evidence the shipped
> houses were produced by exactly this editor and that no other code path bypasses
> `KeepObjectLegal`.

The `kManhole` rule is not a plain parity rule and deserves its own pseudocode
(`HouseLegal.c:217`–`226`):

```
if (what == kManhole && ((bounds.left - 3) % 64) != 0):
    data.b.bounds.left  = (((bounds.left + 29) / 64) * 64) + 3       /* :219-220 */
    data.b.bounds.right = data.b.bounds.left + RectWide(&srcRects[kManhole])  /* :221-223 */
    unchanged = false
```

`+29` before the integer division is a **round-to-nearest-tile** bias: `(x + 29) / 64` rounds up
once `x mod 64 >= 35`. Note it is 29, not 32 — the natural rounding bias — so the snap is biased
slightly *left*: a manhole whose left is 3 + 32 = 35 past a grid line still snaps up to the next
tile, but one at 3 + 31 = 34 snaps back. C integer division truncates toward zero, so for negative
`bounds.left` this rounds the wrong way; `ForceRectInRect` has already guaranteed
`bounds.left >= 0`, so that path is unreachable.

The legal lefts are `left ≡ 3 (mod 64)`: **3, 67, 131, 195, 259, 323, 387, 451** — one per tile,
inset 3 px. Width is forced to `RectWide(&srcRects[kManhole])` = **123**
(`GliderPRO/Sources/StructuresInit2.c:353`: `QSetRect(&srcRects[kManhole], 0, 0, 123, 22)`).
Since 3 + 123 = 126 > 64, a manhole is nearly two tiles wide and always straddles a tile seam; the
rightmost legal left of 451 gives right = 574, which is 62 px **outside** the 512 px room. That is
not corrected: the `kManhole` snap at `:217`–`:226` runs *after* `ForceRectInRect` at `:211`, so a
manhole snapped to the last tile ends up over-wide and is clipped at draw time.

> **Empirical.** The 40 manholes across all 22 houses use only 5 of the 8 legal lefts:
>
> | `bounds.left` | n | `right` = left + 123 | inside 512? |
> |---:|---:|---:|:---:|
> | 67 | 12 | 190 | yes |
> | 131 | 6 | 254 | yes |
> | 195 | 6 | 318 | yes |
> | 259 | 2 | 382 | yes |
> | 323 | 14 | 446 | yes |
>
> Lefts 3, 387 and 451 are unused. Only left = 451 would overflow (right = 574, 62 px outside), so
> the over-wide case never arises in the shipped data.

### 12.3 The four handle-extent clamps

`HouseLegal.c:138`–`193`, inside the blower arm. This is where the *unbounded* `distance` produced
by `DragMarqueeHandle` (§4.7.3) gets its upper bound.

```
if (ObjectHasHandle(&direction, &dist)):     /* :138 — dist comes back as the CURRENT handle distance */
  switch (direction):                        /* :140 */

    kAbove:                                                     /* :142 */
      dist = bounds.top - dist              /* now dist = ABSOLUTE y of the handle tip   :143 */
      if (what in {kFloorVent, kFloorBlower, kTaper, kCandle, kStubby}):   /* :144-148 */
          if (dist < 36):                   /* :150 */
              data.a.distance += dist - 36  /* :152 — negative delta; net: distance = bounds.top - 36 */
              unchanged = false
      else:                                                     /* :156-163 */
          if (dist < 0):
              data.a.distance += dist       /* :160 — net: distance = bounds.top, tip lands at y = 0 */
              unchanged = false

    kToRight:                                                   /* :166 */
      dist = bounds.right + dist            /* absolute x of the tip :167 */
      if (dist > kRoomWide):                /* :168 */
          data.a.distance += (kRoomWide - dist)   /* :170 — net: distance = kRoomWide - bounds.right */
          unchanged = false

    kBelow:                                                     /* :175 */
      dist = bounds.bottom + dist                               /* :176 */
      if (dist > kTileHigh):                                    /* :177 */
          data.a.distance += (kTileHigh - dist)   /* :179 — net: distance = kTileHigh - bounds.bottom */
          unchanged = false

    kToLeft:                                                    /* :184 */
      dist = bounds.left - dist                                 /* :185 */
      if (dist < 0):                                            /* :186 */
          data.a.distance += dist           /* :188 — net: distance = bounds.left, tip lands at x = 0 */
          unchanged = false
```

The idiom is uniform and worth stating algebraically: the local `dist` is first converted from a
*relative* distance into the *absolute* screen coordinate of the handle tip, then `data.a.distance`
is incremented by the signed **overshoot**. The algebra always collapses to "place the tip exactly
on the boundary":

| direction | boundary | resulting `data.a.distance` |
|---|---|---|
| `kAbove` (5 special types) | y = 36 | `bounds.top - 36` |
| `kAbove` (all other types) | y = 0 | `bounds.top` |
| `kToRight` | x = `kRoomWide` = 512 | `512 - bounds.right` |
| `kBelow` | y = `kTileHigh` = 322 | `322 - bounds.bottom` |
| `kToLeft` | x = 0 | `bounds.left` |

Note the `kAbove` adjustments are `+= dist - 36` and `+= dist` — **additions of a negative
quantity**, not subtractions of the shortfall. A port that writes `distance += (36 - dist)` gets the
sign backwards and the handle runs away off the top of the room.

The **36 px minimum** for `kFloorVent`, `kFloorBlower`, `kTaper`, `kCandle` and `kStubby` is the
only non-zero minimum. 36 is a bare literal with no named constant; it is roughly the glider's
height plus clearance, i.e. the blower's lift column must be tall enough to be useful. All other
up-blowers get minimum 0. There is **no `kToLeft`/`kToRight` analogue of the 36 inset** — horizontal
handles may reach the room edge exactly.

Also note this whole block is inside the **blower arm only** (`HouseLegal.c:75`–`194`). The other
21 handled types — `kInvisTrans`, `kDeluxeTrans`, `kGreaseRt`, `kGreaseLf`, `kSlider`,
`kFlourescent`, `kTrackLight`, `kToaster`, `kBall`, `kDrip`, `kFish`, `kMirror`, `kWallWindow`,
`kCounter`, `kDresser`, `kTable`, `kShelf`, `kCabinet`, `kDeckTable`, `kInvisObstacle`,
`kInvisBounce` — get their extent clamps from their own arms (§12.5–§12.9), and those clamps are
written as **assignments**, not `+=`.

> **Empirical.** `data.a.distance` ranges by type across 22 houses:
>
> | Type | n | `distance` range |
> |---|---:|---|
> | `kFloorVent` 0x01 | 1458 | `[0, 269]` |
> | `kCeilingVent` 0x02 | 28 | `[82, 303]` |
> | `kFloorBlower` 0x03 | 83 | `[49, 268]` |
> | `kCeilingBlower` 0x04 | 12 | `[61, 225]` |
> | `kSewerGrate` 0x05 | 507 | `[22, 303]` |
> | `kLeftFan` 0x06 | 45 | `[3, 300]` |
> | `kRightFan` 0x07 | 54 | `[0, 368]` |
> | `kTaper` 0x08 | 90 | `[0, 206]` |
> | `kCandle` 0x09 | 180 | `[0, 240]` |
> | `kStubby` 0x0A | 127 | `[0, 237]` |
> | `kTiki` 0x0B | 58 | `[0, 213]` |
> | `kBBQ` 0x0C | 43 | `[32, 276]` |
> | `kInvisBlower` 0x0D | 2336 | `[0, 488]` |
> | `kGrecoVent` 0x0E | 116 | `[64, 303]` |
> | `kSewerBlower` 0x0F | 191 | `[26, 292]` |
> | `kLiftArea` 0x10 | 716 | `[1, 512]` |
>
> `distance == 0` occurs for `kFloorVent`, `kRightFan`, `kTaper`, `kCandle`, `kStubby`, `kTiki` and
> `kInvisBlower` — i.e. the 0 minimum is real. The 36-minimum types show `distance == 0` too, which
> is consistent: the 36 clamp is on `bounds.top - distance`, not on `distance` itself, so
> `distance == 0` is legal whenever `bounds.top >= 36`.

### 12.4 The forced coordinates

Four blocks force `topLeft.v` for floor-mounted blowers (`HouseLegal.c:115`–`137`). Each also adds
2 to `distance`, and **none sets `unchanged = false`**:

| Type | Forced `topLeft.v` | Constant | `distance += 2`? | Lines |
|---|---:|---|:---:|---|
| `kFloorVent` 0x01 | **305** | `kFloorVentTop` | yes | `:115`–`119` |
| `kFloorBlower` 0x03 | **304** | `kFloorBlowerTop` | yes | `:120`–`125` |
| `kSewerGrate` 0x05 | **303** | `kSewerGrateTop` | yes | `:126`–`131` |
| `kFloorTrans` 0x35 | 302 | `kFloorTransTop` | yes | `:132`–`137` — **DEAD CODE** |

`kFloorTrans` (0x35) is a **transport** type, not a blower, so it never reaches the blower arm's
`direction` switch that contains this block. The block at `HouseLegal.c:132`–`137` can never
execute. `kFloorTrans`'s `topLeft.v` is not forced anywhere else, so a `kFloorTrans` can be dragged
to any height. (Its `DragObject` constraint arm is A, `lockH == true`, so it can only move
horizontally — which is why nobody noticed.)

The named constants live at `GliderPRO/Headers/GliderDefines.h:467`–`481`:
`kFloorVentTop` 305 (`:467`), `kCeilingVentTop` 8, `kFloorBlowerTop` 304 (`:469`),
`kCeilingBlowerTop` 5, `kSewerGrateTop` 303 (`:471`), `kCeilingTransTop` 6,
`kFloorTransTop` 302 (`:473`), `kStairsTop` 28, `kCounterBottom` 304, `kDresserBottom` 293,
`kCeilingLightTop` 4, `kHipLampTop` 23, `kDecoLampTop` 91, `kFlourescentTop` 12,
`kTrackLightTop` 5 (`:481`).

Door and window `topLeft.h` are forced to the constants at
`GliderPRO/Headers/GliderDefines.h:483`–`494` (the block opens with `kDoorInTop` 0 at `:483`;
the first *Left* constant is `kDoorInLfLeft` at `:484`):

| Type | `what` | Forced `topLeft.h` | Constant name |
|---|---:|---:|---|
| `kDoorInLf` | 0x37 | 0 | `kDoorInLfLeft` |
| `kDoorInRt` | 0x38 | 368 | `kDoorInRtLeft` |
| `kDoorExLf` | 0x3A | 0 | `kDoorExLfLeft` |
| `kDoorExRt` | 0x39 | 496 | `kDoorExRtLeft` |
| `kWindowInLf` | 0x3B | 0 | `kWindowInLfLeft` |
| `kWindowInRt` | 0x3C | 492 | `kWindowInRtLeft` |
| `kWindowExLf` | 0x3E | 0 | `kWindowExLfLeft` |
| `kWindowExRt` | 0x3D | 496 | `kWindowExRtLeft` |

496 = 512 − 16 = `kRoomWide` − `srcRects[kDoorExRt]` width. 492 = 512 − 20 =
`kRoomWide` − `srcRects[kWindowInRt]` width. 368 = 512 − 144 =
`kRoomWide` − `srcRects[kDoorInRt]` width.

> **Empirical.** All eight forced-h invariants hold for every shipped object:
>
> | Invariant | tested | held | violated |
> |---|---:|---:|---:|
> | `kDoorInLf.h == 0` | 23 | 23 | **0** |
> | `kDoorInRt.h == 368` | 11 | 11 | **0** |
> | `kDoorExLf.h == 0` | 11 | 11 | **0** |
> | `kDoorExRt.h == 496` | 21 | 21 | **0** |
> | `kWindowInLf.h == 0` | 21 | 21 | **0** |
> | `kWindowInRt.h == 492` | 32 | 32 | **0** |
> | `kWindowExLf.h == 0` | 29 | 29 | **0** |
> | `kWindowExRt.h == 496` | 19 | 19 | **0** |
> | `v == 305 kFloorVent` | 1458 | 1458 | **0** |
> | `v == 304 kFloorBlower` | 83 | 83 | **0** |
> | `v == 303 kSewerGrate` | 507 | 507 | **0** |

This closes the loop with `object-taxonomy.md` §4.2 and `house-format.md` §10.9: the forced
coordinates listed there are *maintained by the editor at every edit*, not merely repaired at load.

### 12.5 The blower arm's `kLiftArea` re-derivation

`HouseLegal.c:97`–`101`, *inside* the `if (ForceRectInRect(...))` body:

```
if (what == kLiftArea):
    data.a.distance = RectWide(&bounds)          /* :99  */
    data.a.tall     = RectTall(&bounds) / 2      /* :100 */
```

This is the only place a *containment* correction rewrites a size field rather than just a position.
It matters because `ForceRectInRect` step 3/4 **shrinks** an over-large rect before translating it,
and `kLiftArea` is the one blower whose full extent is stored in `distance`/`tall` rather than being
looked up from `srcRects`. Without this the object would silently keep its old (too large) size.

Two consequences a port must reproduce:

1. `data.a.tall` is a `Byte` holding **half** the pixel height (§6.2). `RectTall(&bounds) / 2` uses C
   truncating division, so an odd pixel height loses 1 px, and the object visibly shrinks by one
   pixel the first time it is pushed against a wall. `kTileHigh / 2` = 161, so `tall` cannot
   overflow the `Byte`.
2. `data.a.distance` is a `short` holding the **full** pixel width, capped by `RectWide(&roomRect)` =
   512, which fits. The empirical max observed for `kLiftArea` is exactly **512** (§12.3 table),
   i.e. at least one shipped lift area spans the whole room.

### 12.6 The bonus arm — grease and slider length clamps

`HouseLegal.c:229`–`281`, 15 labels (`kRedClock` 0x21 … `kSlider` 0x2F).

```
1.  GetObjectRect(...) /* :244 */; if (ForceRectInRect(&bounds, &roomRect)):    /* :245-250 */
        data.c.topLeft.h = bounds.left
        data.c.topLeft.v = bounds.top
        unchanged = false
2.  if      (what == kGreaseRt && bounds.right + data.c.length > kRoomWide):   /* :251-256 */
        data.c.length = kRoomWide - bounds.right                              /* :254 */
    else if (what == kGreaseLf && bounds.left  - data.c.length < 0):           /* :257-261 */
        data.c.length = bounds.left                                           /* :260 */
    else if (what == kSlider   && bounds.left  + data.c.length > kRoomWide):   /* :262-267 */
        data.c.length = kRoomWide - bounds.left                               /* :266 */
3.  if (data.c.topLeft.h % 2 != 0):  data.c.topLeft.h--; unchanged = false     /* :269-273 */
4.  if (what != kStar && data.c.length % 2 != 0):  data.c.length--; unchanged = false  /* :274-279 */
```

Four things to get right:

* **Step 2 is an `if / else if / else if` chain, not three independent tests.** The three types are
  mutually exclusive so the chaining is semantically harmless here, but a port that "simplifies" it
  into three `if`s is equivalent only by luck.
* Each clamp is a **conditional assignment guarded by an inequality**, not an unconditional
  normalisation. `data.c.length` is left completely alone when the object already fits. So
  `length` is *not* a derived quantity — a port must store it.
* `kGreaseRt` measures its spill from `bounds.right` **rightwards**; `kGreaseLf` measures from
  `bounds.left` **leftwards**; `kSlider` measures from `bounds.left` **rightwards** (the slider's
  `length` is its own width, so the object rect grows with it — see `GetObjectRect`, §13). Three
  different anchor/direction combinations for what look like the same field.
* Step 4 excludes **`kStar`** from the even-length rule, and `kStar` alone among the 15 bonuses. The
  reason is visible in `object-dynamics.md`: `kStar`'s `data.c.length` is not a pixel distance at
  all. Step 3's even-`h` rule has no exception. (Note: the shipped houses never *exercise* the
  exception — see the empirical box below.)

> **Empirical.** All three conditional clamps hold:
>
> | Invariant | tested | held | violated |
> |---|---:|---:|---:|
> | `kGreaseRt: right + length <= 512` | 195 | 195 | **0** |
> | `kGreaseLf: left - length >= 0` | 143 | 143 | **0** |
> | `kSlider: left + length <= 512` | 354 | 354 | **0** |
> | `even-length bonus` (the 14 bonus types other than `kStar`) | 2923 | 2923 | **0** |
>
> The `kStar` exception is **not** exercised by the shipped data: all **69** `kStar` objects have an
> even `data.c.length` (67 × `0`, 2 × `4`). So the exception at `:274`–`:275` cannot be confirmed as
> load-bearing from the houses alone — it is only justified by `kStar`'s `length` not being a pixel
> distance (see `object-dynamics.md`). A port that dropped the exception would produce byte-identical
> results on every shipped house.

### 12.7 The transport arm — the door/window side flip

`HouseLegal.c:282`–`385`, 16 labels (`kUpStairs` 0x31 … `kDeluxeTrans` 0x40). This arm contains the
single most surprising rule in `KeepObjectLegal`: **it can change `what`.**

```
1.  GetObjectRect(...) /* :298 */; if (ForceRectInRect(&bounds, &roomRect)):  /* :299-309 */
        data.d.topLeft.h = bounds.left
        data.d.topLeft.v = bounds.top
        unchanged = false
        if (what == kDeluxeTrans):                                       /* :304 */
            data.d.tall = ((RectWide(&bounds)/4) << 8) + (RectTall(&bounds)/4)   /* :306-307 */

2.  if (what == kDoorInLf || what == kDoorInRt):                         /* :310-311 */
        if (data.d.topLeft.h + HalfRectWide(&srcRects[kDoorInLf]) > kRoomWide/2):   /* :313-314 */
            data.d.topLeft.h = kDoorInRtLeft; what = kDoorInRt           /* :316-317 */
        else:
            data.d.topLeft.h = kDoorInLfLeft; what = kDoorInLf           /* :321-322 */
3.  ... same shape for kDoorEx*   (:325-339), kWindowIn* (:340-354), kWindowEx* (:355-369)

4.  if (what == kInvisTrans && data.d.topLeft.v + data.d.tall > kTileHigh):   /* :371-373 */
        data.d.tall = kTileHigh - data.d.topLeft.v; unchanged = false          /* :375-376 */
5.  if (what == kInvisTrans && data.d.wide < 0):                               /* :379-380 */
        data.d.wide = 0; unchanged = false                                     /* :382 */
```

**The flip.** `DragObject`'s constraint arm for all eight door/window types is
`DragMarqueeRect(where, &newRect, true, false)` (`ObjectEdit.c:398`–`411`), and — per the parameter
inversion documented in §4.7.2 — `lockH == true` zeroes `deltaV`, so doors and windows drag
**horizontally only**. That makes the flip fully reachable: drag a `kDoorInLf` rightwards past the
threshold and it *becomes* a `kDoorInRt`, snapping to the far wall. Drag it back and it becomes
`kDoorInLf` again. The object slot, its `where`/`who` link fields and its z-order are all preserved;
only `what` and `topLeft.h` change.

The four thresholds, with `HalfRectWide(&srcRects[...])` resolved from
`GliderPRO/Sources/StructuresInit2.c:393`–`400` and `kRoomWide / 2` = **256**:

| Pair | Probe rect | `HalfRectWide` | Flips to *Rt* when | Left constant when *Lf* | Left constant when *Rt* |
|---|---|---:|---|---:|---:|
| `kDoorInLf` 0x37 / `kDoorInRt` 0x38 | `srcRects[kDoorInLf]` 144×322 | **72** | `h + 72 > 256` ⟺ `h > 184` | `kDoorInLfLeft` 0 | `kDoorInRtLeft` 368 |
| `kDoorExLf` 0x3A / `kDoorExRt` 0x39 | `srcRects[kDoorExRt]` 16×322 | **8** | `h + 8 > 256` ⟺ `h > 248` | `kDoorExLfLeft` 0 | `kDoorExRtLeft` 496 |
| `kWindowInLf` 0x3B / `kWindowInRt` 0x3C | `srcRects[kWindowInLf]` 20×170 | **10** | `h + 10 > 256` ⟺ `h > 246` | `kWindowInLfLeft` 0 | `kWindowInRtLeft` 492 |
| `kWindowExLf` 0x3E / `kWindowExRt` 0x3D | `srcRects[kWindowExRt]` 16×170 | **8** | `h + 8 > 256` ⟺ `h > 248` | `kWindowExLfLeft` 0 | `kWindowExRtLeft` 496 |

Note each pair uses the **`Lf` or `Rt` member's own rect** as the probe, and the choice is
inconsistent: the interior pair probes with `Lf`, the other three probe with `Rt`. Since both members
of each pair have the same `srcRects` width (144/144, 16/16, 20/20, 16/16 —
`StructuresInit2.c:393`–`400`) the inconsistency is unobservable. A port should still mirror it
rather than "fix" it, because the widths are data.

`kDoorEx*` and `kWindowEx*` share the same threshold (248) and the same `Rt` left (496), but they are
different types with different heights (322 vs 170).

**The four flip blocks are unconditional** — they sit *outside* the `if (ForceRectInRect(...))` body
(`:310`, `:325`, `:340`, `:355`) and every arm of the inner `if/else` writes `topLeft.h`. So a
door or window's `topLeft.h` is snapped to one of the two wall constants on **every**
`KeepObjectLegal` call, whether or not anything was out of bounds. These eight types have no
free horizontal position at all: the drag only chooses which side.

The flip is also what makes the eight door/window types' `MoveObject` omission (§7.6) *safe*: the
arrow-key nudge never translates `data.d.topLeft.h` for them, so it can never trigger a flip.

**`kDeluxeTrans` size re-derivation** (step 1, `:304`–`:308`): as with `kLiftArea`, a containment
correction rewrites the size, and again through a lossy encoding —
`((RectWide/4) << 8) + (RectTall/4)`, i.e. both dimensions packed into the single `short data.d.tall`
in 4 px units (§6.3). The `/4` truncates, so an over-large deluxe transport that gets shrunk loses up
to 3 px on each axis. `RectWide <= 512` ⇒ packed width ≤ 128, and `RectTall <= 322` ⇒ packed
height ≤ 80, so neither byte overflows.

**`kInvisTrans` step 4** is a genuine assignment clamp on `data.d.tall`, and **step 5** is dead:
`data.d.wide` is `Byte` (unsigned, `GliderStructs.h:42`), so `< 0` is never true. GCC and Clang both
warn on this; MPW C 68k did not.

> **Empirical.**
>
> | Invariant | tested | held | violated |
> |---|---:|---:|---:|
> | `kInvisTrans wide <= 127` | 385 | 385 | **0** |
> | `kInvisTrans wide >= 0` (dead test) | 385 | 385 | **0** |
> | `kInvisTrans v + tall <= 322` | 385 | 385 | **0** |
> | `kDeluxeTrans packed width >= 16` (= 64 px) | 61 | 61 | **0** |
> | `kDeluxeTrans packed height >= 8` (= 32 px) | 61 | 61 | **0** |
> | all eight door/window forced-`h` values | 167 | 167 | **0** |

### 12.8 The light arm — the one unconditional overwrite

`HouseLegal.c:410`–`465`, 8 labels (`kCeilingLight` 0x51 … `kInvisLight` 0x58). This arm is the
messiest in the function and contains a real bug.

```
1.  GetObjectRect(...)                                                        /* :418 */
    if (ForceRectInRect(&bounds, &roomRect)):                                 /* :419 */
        if (what == kFlourescent || what == kTrackLight):                     /* :421 */
            if (data.f.topLeft.h < bounds.left)  data.f.topLeft.h = bounds.left   /* :423-424 */
            if (data.f.topLeft.v < bounds.top)   data.f.topLeft.v = bounds.top    /* :426-427 */
            if (data.f.topLeft.h + data.f.length > bounds.right):                 /* :429 */
                data.f.length = bounds.right - data.f.topLeft.h                   /* :430 */
        else:
            data.f.topLeft.h = bounds.left                                    /* :434 */
            data.f.topLeft.v = bounds.top                                     /* :435 */
        unchanged = false                                                     /* :437 */

2.  if ((what == kFlourescent || what == kTrackLight) &&
            (bounds.right > kRoomWide || bounds.left < 0)):                   /* :439-441 */
        if (data.f.topLeft.h < 0):        data.f.topLeft.h = 0;        unchanged = false  /* :443-447 */
        if (bounds.left < 0):             bounds.left = 0;             unchanged = false  /* :448-452 */
        if (data.f.topLeft.h > kRoomWide):data.f.topLeft.h = kRoomWide;unchanged = false  /* :453-457 */
        if (bounds.right > kRoomWide):    bounds.right = kRoomWide;    unchanged = false  /* :458-462 */
        data.f.length = kRoomWide - bounds.left    /* :463 — UNCONDITIONAL, no unchanged=false */
```

Four separate defects worth naming:

1. **Step 2 line `:463` is unconditional.** Once the guard at `:439`–`441` fires, `data.f.length` is
   *assigned* `kRoomWide - bounds.left` regardless of what it was. Because `bounds.left` has just
   been floored to 0 at `:450` when it was negative, the assignment is often
   `length = 512` — i.e. a fluorescent tube dragged off the left edge comes back **spanning the
   entire room**, not restored to its original length. This is the single most visible
   `KeepObjectLegal` behaviour a porter will notice as "wrong" and be tempted to fix.
2. **`:463` never sets `unchanged = false`**, so it is a tenth non-clearing mutation site on top of
   the nine tabulated in §12.1.
3. **Step 2 mutates the local `bounds`** at `:450` and `:460` after `GetObjectRect` filled it. Those
   writes are discarded when the function returns — they exist only to feed `:463`. The two "fix
   `bounds`" tests are therefore not corrections at all; they are argument preparation.
4. **The step-2 guard reads `bounds`, which step 1 may already have contained.** `ForceRectInRect`
   translates the rect inside the room, so after step 1 `bounds.left >= 0` and
   `bounds.right <= kRoomWide` **whenever step 1 ran** (i.e. whenever the rect needed correcting).
   So the guard at `:439`–`441` can only be true when step 1's `ForceRectInRect` returned
   `false` — which happens only if the rect was *already* legal — which contradicts the guard.
   Reading the two functions together: `ForceRectInRect` shrinks a too-wide rect and slides it
   in-bounds, always returning a rect satisfying the guard's negation. **Step 2 is therefore
   unreachable through `KeepObjectLegal` alone**, and defect 1 cannot actually be observed. It
   remains a hazard for any port that reorders these steps or that calls the light-clamp logic
   directly.

The `kFlourescent`/`kTrackLight` branch of step 1 uses **one-sided** `if (a < b) a = b` clamps rather
than the unconditional assignment the other six lights get, because for those two types
`data.f.topLeft` is the *left end of a bar* whose length is separate — moving `topLeft` without
adjusting `length` would change the bar's extent. The `:429`–`:430` test then trims `length` to keep
the right end inside. The other six lights have their size in `srcRects` and so can take `topLeft`
straight from `bounds`.

> **Empirical.** `light len <= 512 - h` for `kFlourescent` (115) / `kTrackLight` (81):
> **196 tested, 196 held, 0 violated.** No shipped tube overhangs the room, which is consistent with step 2 being
> unreachable and step 1's `:429`–`:430` doing all the work.

### 12.9 The appliance, enemy and clutter arms

**Appliances**, `HouseLegal.c:467`–`512`, 14 labels (`kShredder` 0x61 … `kCustomPict` 0x6E):

```
1.  GetObjectRect(...); if (ForceRectInRect(&bounds, &roomRect)):       /* :481-487 */
        data.g.topLeft.h = bounds.left; data.g.topLeft.v = bounds.top; unchanged = false
2.  if (what == kToaster && bounds.top - data.g.height < 0):            /* :488-489 */
        data.g.height = bounds.top; unchanged = false                   /* :491 */
3.  if (what == kTV && data.g.topLeft.h % 2 == 0):                      /* :494-495 */
        data.g.topLeft.h--; unchanged = false                           /* :497 */
4.  if (what in {kToaster,kMacPlus,kCoffee,kOutlet,kVCR,kStereo,kMicrowave}
        && data.g.topLeft.h % 2 != 0):                                  /* :500-507 */
        data.g.topLeft.h--; unchanged = false                           /* :509 */
```

Step 2 is the toaster's *throw height* clamp: `data.g.height` is how far above the toaster's top the
toast flies, so `bounds.top - height` is the apex and it must not be negative. The clamp is
`height = bounds.top` — apex exactly at y = 0.

**`data.g.height` is dual-purpose.** For `kToaster` it is a pixel distance (the throw height); for
`kCustomPict` it is a **`PICT` resource ID**. Three independent proofs:

1. The author's own comment, `GliderPRO/Headers/GliderStructs.h:67`:
   `short height; /* 2 toaster, pict ID */`.
2. `GetObjectRect` calls `GetPicture(who->data.g.height)` for `kCustomPict`
   (`GliderPRO/Sources/ObjectRects.c:221`), as does `GetThisRoomsObjRects`
   (`GliderPRO/Sources/ObjectEdit.c:2282`). See §13.5.
3. The observed values are resource IDs, not distances.

> **Empirical.** All 4 782 `kCustomPict` objects across the 22 houses use **149 distinct**
> `data.g.height` values, spanning **10000 … 12351**, in three tight clusters:
>
> | Cluster | distinct IDs | objects |
> |---|---:|---:|
> | 10000 – 10119 | 115 | 4 435 |
> | 11003 – 11031 | 28 | 336 |
> | 12345 – 12351 | 6 | 11 |
>
> The single most common value is **10000** (2 179 objects, 45.6 %) — the fallback ID that
> `GetObjectRect`/`GetThisRoomsObjRects` *write into the object* when `GetPicture` fails
> (`ObjectRects.c:224`, `ObjectEdit.c:2285`). Next are 10002 (319), 10001 (296), 11003 (126),
> 10020 (100).
>
> 19 of the 22 houses use `kCustomPict`; the three that do not are **`Empty House`**, **`Sampler`**
> and **`Slumberland`**. Per-house distinct-ID counts range from 3 (`Castle o' the Air`) to 100
> (`CD Demo House`), 80 (`Teddy World`), 60 (`Nemo's Market`).
>
> A `height` of 10000 read as a pixel distance would place the object 10 000 px off-screen.
> **A Go port must never treat `kCustomPict.height` as geometry.** The IDs resolve against `PICT`
> resources in the *house file's own resource fork* — see `resource-fork.md`.

`kToaster`'s clamp is real and holds:

> **Empirical.** `kToaster: bounds.top - data.g.height >= 0`: **140 tested, 140 held, 0 violated.**

**The one remaining containment violation.** Over all 31 440 objects, with the 10 room-spanning
transport types excluded, the naive `rect inside room` test registers **826** failures:

| Type | failures | Real cause |
|---|---:|---|
| `kCustomPict` 0x6E | 825 | **test artifact** — the test assumed the 72 × 34 `srcRects[kCustomPict]` fallback size, but the true rect comes from the PICT's `picFrame` (§13.5), which is usually narrower. All 825 have `topLeft.h` in 445 – 509 or `topLeft.v` near 292 – 319, i.e. flush against the right or bottom wall, where a 72-px-wide assumption overhangs but the real 20–60 px picture does not. |
| `kTV` 0x65 | **1** | a **genuine** editor defect — see §12.10 |

**Enemies**, `HouseLegal.c:514`–`554`, 9 labels (`kBalloon` 0x71 … `kCobweb` 0x79):

```
1.  containment → data.h.topLeft.h/.v                                    /* :523-529 */
2.  if (what in {kBall, kFish} && bounds.top - data.h.length < 0):        /* :530-532 */
        data.h.length = bounds.top; unchanged = false                     /* :534 */
3.  if (what == kDrip && bounds.bottom + data.h.length > kTileHigh):      /* :537-538 */
        data.h.length = kTileHigh - bounds.bottom; unchanged = false      /* :540 */
4.  if (what in {kBalloon,kCopterLf,kCopterRt,kBall,kDrip,kFish}
        && data.h.topLeft.h % 2 != 0):                                    /* :543-549 */
        data.h.topLeft.h--; unchanged = false                             /* :551 */
```

`kBall` and `kFish` measure `length` **upwards** from `bounds.top` (bounce apex / jump apex);
`kDrip` measures **downwards** from `bounds.bottom` (fall distance). `kDartLf`, `kDartRt` and
`kCobweb` get containment only — no extent, no parity.

> **Empirical.** `ball/fish top - length >= 0`: **332 tested** (212 `kBall` + 120 `kFish`), **332 held, 0 violated.**
> `kDrip bottom + length <= 322`: **477 tested, 477 held, 0 violated.**

**Clutter**, `HouseLegal.c:556`–`590`, 15 labels (`kOzma` 0x81 … `kChimes` 0x8F):

```
1.  GetObjectRect(...); if (ForceRectInRect(&bounds, &roomRect)):        /* :571-576 */
        data.i.bounds = bounds; unchanged = false                        /* :574 — whole Rect copied */
2.  if (what == kMirror):                                               /* :577 */
        if (data.i.bounds.left  % 2 != 0): data.i.bounds.left--;  unchanged = false  /* :579-583 */
        if (data.i.bounds.right % 2 != 0): data.i.bounds.right--; unchanged = false  /* :584-588 */
```

`kMirror` is the only type whose **both** horizontal edges are parity-forced. That is because a
mirror's contents are drawn by a `CopyBits` of the room's own back buffer, and an odd-width or
odd-offset source rect breaks the 8-bit indexed-colour blit alignment the mirror code assumes
(see `rendering.md`). Note step 2 decrements each edge independently, so a mirror with odd left and
even right loses 1 px of width; with both odd it keeps its width and shifts left 1 px.

Everything else in the clutter arm gets containment only. `kFlower` 0x85 is in this arm, and its
containment goes through `GetObjectRect` → `data.i.bounds` like every other clutter type, so the
uninitialised `srcRects[kFlower]` entry (§13.7) never enters the clamp.

### 12.10 The ordering bug: parity is applied after containment

Every arm does containment first and parity second. That means a parity decrement can push an object
back **out** of the room by exactly 1 px, and nothing re-checks.

The window is narrow — it requires the object to be flush against the **left** wall (`bounds.left`
= 0) *and* to be a type whose forced parity is **odd**. Only two types force odd:
`kStubby` (0x0A, `HouseLegal.c:103`–`107`) and `kTV` (0x65, `:494`–`:499`). For those, an object at
`left == 0` (even) gets `h--` → `left == -1`.

> **Empirical.** The single `kTV` containment violation in all 22 houses is exactly this:
>
> | House | Room | Slot | `what` | raw 12 bytes | decoded |
> |---|---:|---:|---|---|---|
> | `Titanic` | 161 | 9 | 0x65 `kTV` | `0065 002c ffff 0000 0000 0101` | `topLeft.v` = 44, `topLeft.h` = **−1**, `height` = 0, `byte0` = 0, `delay` = 0, `initial` = 1, `state` = 1 |
>
> Byte-level decode of that record, offset by offset:
>
> | Bytes | Field | Value |
> |---|---|---:|
> | `00 65` | `what` | 101 = 0x65 = `kTV` |
> | `00 2c` | `data.g.topLeft.v` | 44 |
> | `ff ff` | `data.g.topLeft.h` | **−1** |
> | `00 00` | `data.g.height` | 0 |
> | `00` | `data.g.byte0` | 0 |
> | `00` | `data.g.delay` | 0 |
> | `01` | `data.g.initial` | true |
> | `01` | `data.g.state` | true |
>
> `srcRects[kTV]` is 92×77 (`StructuresInit2.c:436`), so the derived rect is
> `(-1, 44, 91, 121)` — 1 px outside the room on the left. `ForceRectInRect` had put it at
> `left == 0`; `HouseLegal.c:497`'s `data.g.topLeft.h--` then pushed it to −1 and the function
> returned without re-checking. This is a **shipped artifact of the shipped editor**, permanently
> baked into `Titanic`.
>
> No `kStubby` instance is affected: all 127 have `topLeft.h >= 1`.

A Go port has three options, in decreasing fidelity:

| Option | Behaviour | Fidelity |
|---|---|---|
| Reproduce exactly (parity after containment, no re-check) | `Titanic` room 161 renders identically to the original, TV 1 px off the left edge | **exact** |
| Re-run containment after parity | TV snaps to `left == 1`; `Titanic` renders 2 px right of the original | breaks a shipped room |
| Force parity by rounding *up* when at the boundary | TV at `left == 1` | breaks a shipped room |

Recommendation: reproduce exactly. The renderer must therefore tolerate object rects with negative
coordinates; see §14.

### 12.11 The unguarded `objects[objActive]` — a reachable corruption

`HouseLegal.c:53` computes `theObject = &thisRoom->objects[objActive]` **before** any check, and
`:73` dereferences `theObject->what`. Only `kInitialGliderSelected` (−2) returns early (`:55`–`:69`).
So for the other three sentinels the switch runs on out-of-bounds memory.

`objects[0]` sits at room offset **60**, and `sizeof(objectType)` is 12, so:

| `objActive` | Sentinel | `objects[n]` at room offset | Overlaps |
|---:|---|---:|---|
| −1 | `kNoObjectSelected` | 48 | `tiles[6]`, `tiles[7]`, `floor`, `suite`, `openings`, `numObjects` |
| −2 | `kInitialGliderSelected` | 36 | `tiles[0]` … `tiles[5]` — **but returns early at `:68`** |
| −3 | `kLeftGliderSelected` | 24 | `name[24..27]`, `bounds`, `leftStart`, `rightStart`, `unusedByte`, `visited`, `background` |
| −4 | `kRightGliderSelected` | 12 | `name[12..23]` only |

Whether anything happens depends on whether the two bytes at that offset happen to spell a `what`
code the switch recognises. The 117 valid codes all lie in `[0x01, 0x8F]`, so the **high** byte must
be 0.

**`objActive == -1`** reads `what` from `tiles[6]`. Tiles are small non-negative indices, so this one
*always* lands on a valid-looking code:

> **Empirical.** `tiles[6]` over all 4070 non-empty rooms:
> `{0: 469, 1: 843, 2: 774, 3: 328, 4: 400, 5: 221, 6: 899, 7: 136}`.
> Values 1–7 are all valid `what` codes (`kFloorVent` … `kRightFan`) — the blower arm. Value 0 is
> not a case label and falls through harmlessly. So **75.7 % of rooms would enter the blower arm**
> and start writing to `tiles[7]`/`floor`/`suite`/`openings`/`numObjects`.
>
> This is not reachable in practice: every caller of `KeepObjectLegal` is inside a
> `#ifndef COMPILEDEMO` editor gesture that has already established a selection, and
> `DeselectObject()` is called before any room change (see §3.4, 19 call sites). No path calls
> `KeepObjectLegal` with `objActive == -1`.

**`objActive == -3` and `-4` ARE reachable.** `DragObject` handles all three glider pseudo-selections
in its commit phase (`ObjectEdit.c:529`–`560`) and then calls `KeepObjectLegal()` unconditionally at
`:747`. `MoveObject` does the same at `:1700`. So dragging or nudging the **left-start** or
**right-start glider marker** enters `KeepObjectLegal` with `objActive == -3` / `-4`, falls past the
`-2` early return, and switches on room-name bytes.

> **Empirical — the name-byte gamble.** For each of the 4070 non-empty rooms, treating
> `name[24]<<8 | name[25]` (for −3) and `name[12]<<8 | name[13]` (for −4) as a `what` code and
> testing membership in the 117-code set:
>
> | Sentinel | Bytes read | Rooms that yield a **valid** `what` | Rate |
> |---:|---|---:|---:|
> | −3 `kLeftGliderSelected` | `name[24..25]` | **5** | 0.12 % |
> | −4 `kRightGliderSelected` | `name[12..13]` | **14** | 0.34 % |
>
> The five −3 hits:
>
> | House | Room | Room name | Decoded `what` |
> |---|---:|---|---|
> | `Leviathan` | 254 | `Sewerno de Bergerac` | 0x68 `kVCR` (appliance arm) |
> | `Slumberland` | 192 | `I Glide, Therefore I Am` | 0x2E `kHelium` (bonus arm) |
> | `Slumberland` | 193 | `I Glide, Therefore I Am` | 0x2E `kHelium` (bonus arm) |
> | `The Asylum Pro` | 8 | `Switch Is To Your Left…` | 0x01 `kFloorVent` (blower arm) |
> | `The Asylum Pro` | 60 | `Learn To Cope With Loss` | 0x6E `kCustomPict` (appliance arm) |
>
> Sample of the fourteen −4 hits: `Leviathan` 372 `High Heat` → 0x6D `kCDs`;
> `Slumberland` 42 `Paul's Room` → 0x6E `kCustomPict`; `Slumberland` 44 `Anabell Lee` → 0x6D;
> `Slumberland` 62 `Big Dripper` → 0x6D; `Slumberland` 147 `You Got It…` → 0x54 `kHipLamp`;
> `Slumberland` 151 `What Gives?` → 0x67 `kOutlet`. The most common decoded value is 0x6D `kCDs`
> (7 of 14).
>
> Padding-byte distributions confirm this is chance, not structure: `name[24]` is most often
> `0x3F` `'?'` (2115 of 4070 rooms) and `name[12]` most often `0x6F` `'o'` (2347 rooms).

**What the corruption does.** For `objActive == -3` the union field offsets map onto the room header
like this (`objects[-3]` base = room offset 24):

| Union field | Room offset | Real room field it aliases |
|---|---:|---|
| `what` | 24–25 | `name[24]`, `name[25]` |
| `data.*.topLeft.v` / `bounds.top` | 26–27 | `name[26]`, `name[27]` |
| `data.*.topLeft.h` / `bounds.left` | 28–29 | **`short bounds`** (the room's tile-bounds field) |
| `data.a.distance` / `c.length` / `d.tall` / `bounds.bottom` | 30–31 | **`Byte leftStart`, `Byte rightStart`** |
| `data.c.points` / `d.where` / `bounds.right` | 32–33 | `Byte unusedByte`, `Boolean visited` |
| `data.b.pict` / `c.state`+`initial` / `d.who`+`wide` | 34–35 | **`short background`** |

So dragging the left-start glider marker in `Slumberland` room 192 enters the **bonus** arm, which
writes `data.c.topLeft.h = bounds.left` (clobbering the room's `bounds`), forces that value even,
and conditionally rewrites `data.c.length` and forces *it* even — clobbering `leftStart` and
`rightStart`. In `The Asylum Pro` room 8 it enters the **blower** arm, which additionally forces
`topLeft.v` (room `name`) and can add 2 to `distance` (`leftStart`/`rightStart`).

For `objActive == -4` (base = room offset 12) every field lands inside `name[12..23]`, so the damage
is confined to room-name bytes — and to bytes past most names' length, so usually invisible.

**Asymmetric severity, then:** the **left**-start marker can corrupt room geometry and start
positions; the **right**-start marker can only scribble on name padding. A Go port must add the
range check the original omits:

```go
if objActive < 0 { return true }   // after the kInitialGliderSelected case
```

and should log rather than silently accept, because 19 shipped rooms are one user gesture away from
header damage.

---

## 13. `GetThisRoomsObjRects` and the geometry cache

`GliderPRO/Sources/ObjectEdit.c:2049`–`2341`, 293 lines, **118** case labels (the 117 object types
plus `kObjectIsEmpty`). This is the function that fills `roomObjectRects[]`, the array every other
part of the editor hit-tests, marquees and invalidates against.

### 13.1 The cache

```c
Rect roomObjectRects[kMaxRoomObs];      // ObjectEdit.c:28
```

`kMaxRoomObs` = **24** (`GliderPRO/Headers/GliderDefines.h:250`). `Rect` is 8 bytes, so the array is
**192 bytes**. Declared without an initialiser, so it lives in BSS and starts zeroed on first launch
— but is *not* re-zeroed between houses or rooms; only `GetThisRoomsObjRects` writes it.

### 13.2 Structure

```
GetThisRoomsObjRects():                                             /* :2049-2341 */
 1. isFirstRoom = (GetFirstRoomNumber() == thisRoomNumber)           /* :2054 */
 2. if (isFirstRoom && !noRoomAtAll && houseUnlocked):               /* :2056 */
        WhereDoesGliderBegin(&initialGliderRect, kNewGameMode)       /* :2057 */
    else:
        QSetRect(&initialGliderRect, 0, 0, 0, 0)                     /* :2059 */
 3. QSetRect(&leftStartGliderDest, 0, 0, 48, 16)                     /* :2061 */
    QOffsetRect(&leftStartGliderDest, 0, kGliderStartsDown + (short)thisRoom->leftStart)   /* :2062-2063 */
 4. QSetRect(&rightStartGliderDest, 0, 0, 48, 16)                    /* :2065 */
    QOffsetRect(&rightStartGliderDest, kRoomWide - 48,
                kGliderStartsDown + (short)thisRoom->rightStart)     /* :2066-2067 */
 5. if (noRoomAtAll || !houseUnlocked):  return                      /* :2069-2072 — EARLY OUT */
 6. else for (i = 0; i < kMaxRoomObs; i++):                          /* :2075 — ALL 24 slots */
      switch (thisRoom->objects[i].what):                            /* :2077 */
        case kObjectIsEmpty:
          QSetRect(&roomObjectRects[i], -2, -2, -1, -1);  break      /* :2080-2081 */
        ... 116 further per-type arms deriving the rect ...
        default:
          QSetRect(&roomObjectRects[i], -2, -2, -1, -1);  break      /* :2335-2336 */
```

Five facts a porter needs immediately:

* **The loop bound is `kMaxRoomObs` (24), not `numObjects`.** So the cache is always fully defined
  for all 24 slots, even the ones past `numObjects` that `FindObjectSelected` refuses to hit-test
  (§3.1). This is what makes tab-cycle (§10.2) able to reach orphaned high slots that a click
  cannot.
* **Empty and unknown slots get `(-2, -2, -1, -1)`**, not `(0, 0, 0, 0)`. That is a *negative,
  1×1, off-screen* rect chosen so that `PtInRect` can never match it for any mouse point (mouse
  coordinates in the room view are ≥ 0) and so that `InvalWindowRect` on it is a no-op.
* **Steps 1–4 run even when step 5 bails out.** So in a locked house or with no room at all, the
  three glider rects are still updated but the 24 object rects keep their **stale values from the
  previous room**. Any hit-test in that state matches the wrong room's objects. In practice the
  editor is unreachable when `!houseUnlocked` (`UpdateMenus` disables it), but a port must not
  "tidy" the early return to the top of the function — the glider rects genuinely have to be
  computed first, because `DrawThisRoomsObjects` draws the entry markers unconditionally.
* **`initialGliderRect` is not computed here.** It is delegated to
  `WhereDoesGliderBegin(&initialGliderRect, kNewGameMode)` (`GliderPRO/Sources/Play.c`), which
  applies the same `(*thisHouse)->initial` plus the `kGliderWide`×`kGliderHigh` size and the
  first-room test. When the room is not the first room the rect is zeroed to `(0,0,0,0)`, which makes
  `PtInRect` always fail — that is how the initial-glider marker is made unselectable outside the
  starting room.
* **`hasMirror` is never touched by this function** — see §13.6.

> **Correction to `editor.md` §6.1**, which states that unused `roomObjectRects` entries are
> `(0, 0, 0, 0)`. They are not. There are **two** distinct sentinels:
>
> | Sentinel | Written by | Meaning |
> |---|---|---|
> | `(-2, -2, -1, -1)` | `GetThisRoomsObjRects` `:2080` (`kObjectIsEmpty`) and `:2335` (`default`) | slot is empty or has an unrecognised `what` |
> | `(-1, -1, 0, 0)` | `DeleteObject` `:1182` | slot was just deleted and the cache has **not** been rebuilt |
>
> `(0, 0, 0, 0)` would be a hit-test hazard: `PtInRect` with left/top inclusive and
> right/bottom exclusive gives an empty rect either way, so it happens to be safe — but the
> distinction between the two real sentinels is observable, because `DeleteObject` deliberately does
> **not** call `GetThisRoomsObjRects()` (§9.1) and so leaves `(-1, -1, 0, 0)` visible until the next
> rebuild.

### 13.3 The 29 rebuild sites

`GetThisRoomsObjRects()` is called from **29** places program-wide. Enumerated in full, because "the
cache is rebuilt after every edit" is *almost* true and the exceptions are what bite:

| File | Lines | Count | Context |
|---|---|---:|---|
| `GliderPRO/Sources/ObjectEdit.c` | 176, 215, 232, 248, 261, 276, 292, 304, 336 | 9 | `DragHandle`, 9 of its 14 arms |
| `GliderPRO/Sources/ObjectEdit.c` | 750 | 1 | `DragObject` tail, right after `KeepObjectLegal` |
| `GliderPRO/Sources/ObjectEdit.c` | 1358 | 1 | `DuplicateObject` tail |
| `GliderPRO/Sources/ObjectEdit.c` | 1705 | 1 | `MoveObject` tail |
| `GliderPRO/Sources/ObjectInfo.c` | 1019, 1049, 1229, 1250, 1697, 1728, 1816, 1853, 1907, 1927, 2336, 2368 | 12 | the per-type "Object Info" dialogs, after OK |
| `GliderPRO/Sources/Room.c` | 335 | 1 | room-load path — every room change |
| `GliderPRO/Sources/ObjectAdd.c` | 785 | 1 | `AddNewObject` tail |
| `GliderPRO/Sources/Menu.c` | 465 | 1 | a menu command (object-order / bring-forward group) |
| `GliderPRO/Sources/Scrap.c` | 201 | 1 | paste path |
| `GliderPRO/Sources/Objects.c` | 993 | 1 | mode switch back into edit mode |
| **total** | | **29** | |

**The gestures that do NOT rebuild:**

| Gesture | Consequence |
|---|---|
| `DeleteObject` (§9) | writes `(-1,-1,0,0)` into the slot by hand at `:1182` instead |
| tab-cycle (§10) | no geometry changes, so none needed |
| `DragHandle` arms 1, 3, 4, 12, 13 (18 types) | those arms edit a *distance*, not the object rect — see §6.1 |

### 13.4 The three glider rects

```
initialGliderRect:    WhereDoesGliderBegin(&initialGliderRect, kNewGameMode)   /* :2057 */
                      -- or (0,0,0,0) when this is not the first room          /* :2059 */
leftStartGliderDest:  QSetRect(0, 0, 48, 16)                                   /* :2061 */
                      QOffsetRect(0,              kGliderStartsDown + (short)thisRoom->leftStart)   /* :2062-2063 */
rightStartGliderDest: QSetRect(0, 0, 48, 16)                                   /* :2065 */
                      QOffsetRect(kRoomWide - 48, kGliderStartsDown + (short)thisRoom->rightStart)  /* :2066-2067 */
```

`kGliderWide` = 48, `kGliderHigh` = 20, `kGliderStartsDown` = **32**
(`GliderPRO/Headers/GliderDefines.h:569`), and `kRoomWide - 48` = **464**.

Four details:

* The start markers are **48 × 16**, not 48 × 20 — 4 px shorter than the real glider. The marker art
  is a 16-px-tall slice, not the full sprite.
* The `(short)` casts on `leftStart`/`rightStart` are load-bearing: both are `Byte`
  (`GliderStructs.h:170`–`171`), so they are **unsigned 0..255**, and the marker's y ranges over
  `32..287`. `DragObject:539`–`:544` and `:551`–`:556` clamp to exactly that range before storing.
* `:2066` uses the correct `kRoomWide - 48`, while `DragObject:558` uses **0** — the bug documented in
  §5.2. Because `GetThisRoomsObjRects` runs at `:750` immediately after the buggy `:557`–`:559`, the
  wrong rect is overwritten with the right one before the next hit-test; the only visible symptom is
  the `InvalWindowRect(mainWindow, &rightStartGliderDest)` at `:761`, which invalidates the wrong
  region and can leave a 48×16 artifact at the left edge until the next full redraw.
* `initialGliderRect` is zeroed rather than positioned when `!isFirstRoom`, `noRoomAtAll` or
  `!houseUnlocked`. `FindObjectSelected` tests it first (§3.1), so a zeroed rect silently means
  "no initial-glider marker in this room".

### 13.5 The derived-rect rules that are not just `srcRects` lookups

The default shape, used by the majority of arms, is exactly three statements:

```c
roomObjectRects[i] = srcRects[thisRoom->objects[i].what];    /* copy the sprite's src rect */
ZeroRectCorner(&roomObjectRects[i]);                        /* slide it so top-left == (0,0) */
QOffsetRect(&roomObjectRects[i],                            /* then place it */
        thisRoom->objects[i].data.X.topLeft.h,
        thisRoom->objects[i].data.X.topLeft.v);
```

`ZeroRectCorner` (`GliderPRO/Sources/RectUtils.c:57`-`63`) is

```c
void ZeroRectCorner (Rect *theRect)     // Offset rect to (0, 0)
{
    theRect->right  -= theRect->left;
    theRect->bottom -= theRect->top;
    theRect->left = 0;
    theRect->top  = 0;
}
```

— it normalises the rect to the origin while preserving size,
because `srcRects` entries carry an *offset within the sprite sheet*, not a screen position.
**Every arm that reads `srcRects` must call `ZeroRectCorner` first**; forgetting it would add the
sprite-sheet offset to the object's screen position.

The nine exceptions, with verified line numbers:

| Types | Lines | Rule |
|---|---|---|
| `kLiftArea` 0x10 | `:2105`–`:2113` | `QSetRect(0, 0, data.a.distance, data.a.tall * 2)` then offset by `topLeft` — **does not read `srcRects` at all** |
| 15 furniture (`kTable` … `kInvisBounce`) | `:2114`–`:2130` | `roomObjectRects[i] = data.b.bounds;` — **verbatim, no offsetting** |
| `kGreaseRt` 0x28 | `:2151`–`:2159` | default shape, **then** `if (!data.c.initial) QOffsetRect(+8, 0)` |
| `kGreaseLf` 0x29 | `:2161`–`:2169` | default shape, **then** `if (!data.c.initial) QOffsetRect(-8, 0)` |
| `kSlider` 0x2F | `:2171`–`:2179` | default shape, **then** `right = left + data.c.length` (so width = `length`, height = 16 from `srcRects`) |
| `kInvisTrans` 0x3F | `:2202`–`:2211` | default shape, then `bottom = top + data.d.tall`, then `right += (short)data.d.wide` (width = 64 + `wide`) |
| `kDeluxeTrans` 0x40 | `:2213`–`:2220` | `wide = (data.d.tall & 0xFF00) >> 8; tall = data.d.tall & 0x00FF; QSetRect(0, 0, wide * 4, tall * 4)` then offset. Source comment: `// Uses a kludge to get width & height (x4)` |
| `kFlourescent` 0x56 / `kTrackLight` 0x57 | `:2251`–`:2259` | `srcRects`, `ZeroRectCorner`, **then `right = data.f.length`**, *then* `QOffsetRect` |
| `kCustomPict` 0x6E | `:2281`–`:2298` | `GetPicture(data.g.height)`; on success rect = `(*thePict)->picFrame`, on failure `data.g.height = 10000` **and** rect = `srcRects[kCustomPict]` (72 × 34); then `ZeroRectCorner` + `QOffsetRect` |
| 15 clutter (`kOzma` … `kChimes`) | `:2316`–`:2332` | `roomObjectRects[i] = data.i.bounds;` — **verbatim** |

Types whose extent field is deliberately **excluded** from the cached rect, because the field is a
*motion range*, not a size: `kGreaseRt`/`kGreaseLf` (`data.c.length`, the spill run),
`kToaster` (`data.g.height`, the throw apex), `kBall`/`kFish` (`data.h.length`, the bounce apex),
`kDrip` (`data.h.length`, the fall distance). All six use the plain default shape. This is why
`DragHandle`'s corresponding arms do **not** rebuild the cache (§6.1) — the rect genuinely does not
change.

**The `!initial` grease shift.** `kGreaseRt` and `kGreaseLf` are the only two types whose *cached
rect depends on a state Boolean*. `data.c.initial` false means "this spill has already been tipped
over", and the tipped sprite sits 8 px to the side. So the same object has two different hit boxes
depending on a field the user edits in the Object Info dialog, and the two rects overlap by
`32 - 8 = 24` px. Note the shift is applied **after** `QOffsetRect(topLeft)`, so it is a pure screen
displacement and is *not* folded back into `topLeft`. `KeepObjectLegal`'s bonus arm reads
`GetObjectRect`, which applies the same shift (`ObjectRects.c`), so containment is computed against
the *shifted* rect — meaning a right-facing grease can be pushed 8 px left of where its `topLeft`
says when `initial` is false.

> **Not a bug.** `editor.md` and casual reading both suggest the `kFlourescent`/`kTrackLight` arm is
> wrong because `:2255` writes `roomObjectRects[i].right = thisRoom->objects[i].data.f.length` — an
> absolute assignment of a *length* to a *coordinate*. It is correct, and the statement order at
> `:2253`–`:2258` is what makes it correct:
>
> 1. `:2253` copy `srcRects[what]`
> 2. `:2254` `ZeroRectCorner` ⇒ `left == 0`, `top == 0`
> 3. `:2255` `right = data.f.length` ⇒ width == `length` (because `left` is 0)
> 4. `:2256`–`:2258` `QOffsetRect(topLeft.h, topLeft.v)` ⇒ final placement
>
> Do not "fix" this, and do not reorder steps 3 and 4 — moving the `QOffsetRect` earlier would make
> `right = length` an absolute coordinate and collapse the bar.

**`kCustomPict`'s side effect is the sharpest edge in this function.** A pure geometry-cache rebuild
**mutates the house**:

```
case kCustomPict:                                            /* :2281 */
    thePict = GetPicture(thisRoom->objects[i].data.g.height); /* :2282 — Resource Manager call! */
    if (thePict == nil):                                     /* :2283 */
        thisRoom->objects[i].data.g.height = 10000            /* :2285 — WRITES THE OBJECT */
        roomObjectRects[i] = srcRects[thisRoom->objects[i].what]   /* :2286 — 72 × 34 fallback */
    else:
        HLock((Handle)thePict)                               /* :2290 */
        roomObjectRects[i] = (*thePict)->picFrame             /* :2291 */
        HUnlock((Handle)thePict)                             /* :2292 */
    ZeroRectCorner(&roomObjectRects[i])                       /* :2294 */
    QOffsetRect(&roomObjectRects[i], data.g.topLeft.h, data.g.topLeft.v)  /* :2295-2297 */
```

Four consequences for a Go port:

1. **Object geometry is a function of resource-fork contents**, not of the house data fork alone. You
   cannot compute `roomObjectRects` from the parsed `objectType` records; you must have the PICTs
   loaded and know each one's `picFrame`.
2. **`GetPicture` failing rewrites `data.g.height` to 10000** without setting `fileDirty`. Open a
   house that references a missing PICT, walk into that room, and the in-memory house silently
   differs from disk. Save later for any reason and the reference is permanently lost. This is very
   likely how 2 179 of the 4 782 shipped `kCustomPict` objects came to have `height == 10000`.
3. `GetPicture` consults the **current resource chain** — house file first, then the application,
   then the System file. IDs 10000+ are house resources; a port with per-house resource maps must
   replicate the search order (see `resource-fork.md`).
4. `(*thePict)->picFrame` is the `Rect` at offset **2** of a `PICT` resource (after the 2-byte
   `picSize`), stored **big-endian** in `top, left, bottom, right` order. A port reads 8 bytes at
   offset 2 of the resource. The 68k `HLock`/`HUnlock` pair exists only because the handle could be
   relocated by the Memory Manager; Go needs nothing there.

Identical logic appears in `GetObjectRect` at `GliderPRO/Sources/ObjectRects.c:220`–`237` — the same
side effect, in the function `KeepObjectLegal` calls. So the mutation can also happen during
clamping.

### 13.6 `hasMirror` does not live here

It is easy to assume `GetThisRoomsObjRects` maintains `hasMirror`, because it is the one function
that walks every slot's `what`. It does not. The real ownership:

```c
Boolean     hasMirror;              // GliderPRO/Sources/Render.c:42  — definition
```

| Site | Action |
|---|---|
| `GliderPRO/Sources/InterfaceInit.c:138` | `hasMirror = false;` at startup |
| `GliderPRO/Sources/Render.c:760` | `hasMirror = true;` — tail of `AddToMirrorRegion`, called once per mirror as the room is composited |
| `GliderPRO/Sources/Render.c:770` | `hasMirror = false;` — inside `ZeroMirrorRegion`, which also disposes `mirrorRgn` |

Readers: `GliderPRO/Sources/Modes.c:92`, `GliderPRO/Sources/Play.c:720`,
`GliderPRO/Sources/Render.c:641`.

So `hasMirror` is a **render-pass flag derived during compositing**, paired with a QuickDraw
`RgnHandle` (`mirrorRgn`) accumulated by `UnionRgn` at `Render.c:752`–`758`. It is not editor state
and the editor never writes it. A Go port replaces the `RgnHandle` with a slice of rects or a
scanline mask (there is no Go equivalent of `NewRgn`/`UnionRgn`/`RectRgn`), and derives the boolean as
`len(mirrorRects) > 0`.

The editor's own per-room mirror question — "does this room contain a `kMirror`?" — is answered
implicitly, by the `kMirror` arm of the compositor running at all.

### 13.7 The uninitialised `srcRects[kFlower]` entry, and why it is harmless

`srcRects` is allocated with `NewPtr`, which does **not** zero
(`GliderPRO/Sources/StructuresInit2.c:270`–`271`):

```c
srcRects = (Rect *)NewPtr(sizeof(Rect) * kNumSrcRects);
```

`kNumSrcRects` = 0x90 = **144**, so the block is 144 × 8 = **1152 bytes of uninitialised heap**.
`InitSrcRects` (`StructuresInit2.c:306`–`475`) then fills **116** of the 144 entries. The 28 gaps are
the numeric holes between the nine type ranges (0x00, 0x20, 0x30, 0x50, 0x59–0x60, 0x6F–0x70,
0x7A–0x80) — plus one that is *not* a hole:

**`srcRects[kFlower]` (`kFlower` = 0x85 = 133) is never initialised.** Every neighbour is:
`kFireplace` 0x84 at `:463`, `kWallWindow` 0x86 at `:464`. The line for `kFlower` is simply absent.

**This is not a live bug, and the reason is now settled exhaustively.** `kFlower` appears as a `case`
label at exactly **14** sites in the shipped source. None of them reads `srcRects[kFlower]`:

| Site | Enclosing function | What the `kFlower` path actually reads | Safe? |
|---|---|---|:--:|
| `GliderPRO/Sources/ObjectAdd.c:739`–`747` | `AddNewObject` (`:48`) | its **own dedicated arm**: `newRect = flowerSrc[wasFlower]` (`:743`) | by design |
| `GliderPRO/Sources/ObjectEdit.c:505` → `:515` | `DragObject` rubber-band dispatch | `DragMarqueeRect(where, &newRect, false, false)` — no `srcRects` at all | yes |
| `GliderPRO/Sources/ObjectEdit.c:722` → `:732`–`736` | `DragObject` field commit (arm 15) | `data.i.bounds.{left,right,top,bottom} += delta` | yes |
| `GliderPRO/Sources/ObjectEdit.c:1345` | `DuplicateObject` arm 15 | copies `data.i.bounds`, then `QOffsetRect(&bounds, 64, 0)` | yes |
| `GliderPRO/Sources/ObjectEdit.c:1674` → `:1684`–`1688` | `MoveObject` (dead code, §7) | `data.i.bounds.{left,right,top,bottom} += delta` | yes |
| `GliderPRO/Sources/ObjectEdit.c:2320` → `:2331` | `GetThisRoomsObjRects` | `roomObjectRects[i] = thisRoom->objects[i].data.i.bounds` | yes |
| `GliderPRO/Sources/ObjectEdit.c:2684`–`2685` | `DrawThisRoomsObjects` | `DrawFlower(&roomObjectRects[i], data.i.pict)` — **not** `DrawSimpleClutter` | by design |
| `GliderPRO/Sources/ObjectDrawAll.c:922`–`927` | play-mode compositor | `GetObjectRect` then `DrawFlower(&itsRect, data.i.pict)` | by design |
| `GliderPRO/Sources/ObjectRects.c:259` → `:270` | `GetObjectRect` | `*itsRect = who->data.i.bounds` | yes |
| `GliderPRO/Sources/ObjectRects.c:1039` → `:1048` | `CreateActiveRects` (`:296`) | bare `break` — flowers get no hot spot | yes |
| `GliderPRO/Sources/Objects.c:681` → `:690` | `SetObjectState` (`:366`) | bare `break` | yes |
| `GliderPRO/Sources/Objects.c:854` | `GetObjectState` (`:703`) | bare `break` | yes |
| `GliderPRO/Sources/HouseLegal.c:560` → `:571`–`576` | `KeepObjectLegal` clutter arm | `GetObjectRect` + `ForceRectInRect` → `data.i.bounds` | yes |
| `GliderPRO/Sources/ObjectInfo.c:2551`–`2552` | `DoObjectInfo` dispatch | `DoFlowerObjectInfo()`, which sizes from `flowerSrc[flower]` (`:2330`, `:2333`, `:2362`, `:2365`) | by design |

The three "by design" rows are the interesting ones. `DrawSimpleClutter`
(`GliderPRO/Sources/ObjectDraw2.c:1018`–`1024`) is the generic clutter blitter and it *does* read
`&srcRects[what]` (`:1023`) — it is the one function that would have exposed the omission. Both of its
callers special-case `kFlower` one label earlier and route it to `DrawFlower`
(`ObjectDraw2.c:1028`–`1034`), which reads `&flowerSrc[which]` (`:1033`) instead. So the author was
aware that `kFlower` has six variable-size sprites rather than one fixed rect, and the missing
`InitSrcRects` line is a *consequence* of that design, not an oversight with teeth.

The remaining consequences, restated:

1. `kFlower` is a **clutter** type, so `GetThisRoomsObjRects` reads its rect from `data.i.bounds`
   rather than `srcRects` — the omission is invisible in the editor's geometry cache.
2. `KeepObjectLegal`'s clutter arm likewise uses `data.i.bounds` (`HouseLegal.c:571`–`576`), so
   clamping is unaffected.
3. `data.i.pict` carries the flower *variety* (0–5), not a `'PICT'` ID — the only object type where
   `pict` means something other than a resource number or zero. `AddNewObject` picks it with
   `wasFlower = RandomInt(kNumFlowers)` (`ObjectAdd.c:742`) **unless Shift is held**
   (`GetKeys` + `BitTst(&theseKeys, kShiftKeyMap)`, `ObjectAdd.c:740`–`741`), in which case the
   previous value of the file-scope `short wasFlower` (`ObjectAdd.c:42`) is reused so the user can
   place a run of identical flowers. `wasFlower` is seeded once at startup, also randomly
   (`InterfaceInit.c:160`). `kNumFlowers` = **6** (`GliderPRO/Headers/GliderDefines.h:458`);
   `flowerSrc` is declared `Rect flowerSrc[kNumFlowers]` at `GliderPRO/Sources/Objects.c:70`.
4. `ObjectInfo.c`'s flower dialog writes `wasFlower = flower` (`:2341`, `:2373`), so choosing a
   variety in the dialog also changes what the *next* Shift-placed flower will be.

So the omission is latent, not live. A Go port that eagerly builds a `[144]Rect` table from
`InitSrcRects` and then *asserts* every valid `what` has an entry will trip on `kFlower`. Zero it and
move on; do not synthesise a size — and make sure your flower path uses the six-entry `flowerSrc`
table keyed on `pict`, not the 144-entry one keyed on `what`.

The six `flowerSrc` rects, for completeness (`StructuresInit2.c:72`–`88`):

| Index | Size | Offset in `clutterSrcMap` |
|---:|---|---|
| 0 | 10 × 28 | (0, 23) |
| 1 | 24 × 35 | (10, 16) |
| 2 | 34 × 35 | (34, 16) |
| 3 | 27 × 23 | (68, 14) |
| 4 | 27 × 14 | (68, 37) |
| 5 | 32 × 51 | (95, 0) |

`clutterSrcRect` is `(0, 0, 128, 69)` (`:63`) and the graphic is `PICT` **4018**
(`kClutterPictID`, `:22`), with its 1-bit mask at `kClutterPictID + 1000` = **5018** (`:70`).

---

## 14. `DrawThisRoomsObjects` and `HiliteAllObjects`

### 14.1 `DrawThisRoomsObjects`

`GliderPRO/Sources/ObjectEdit.c:2347`–`2729`, 383 lines, **118** case labels (117 types +
`kObjectIsEmpty`), dispatching to **64 distinct `Draw*` functions**. This is the fourth member of the
"redraw quartet" (§5.5) and the function every editing gesture ends with. The per-type drawing itself
is documented in `object-draw-all.md`; what follows is only what is specific to the *editor's* use of
it.

```
DrawThisRoomsObjects():                                          /* :2347-2729 */
 1. GetGWorld(&wasCPort, &wasWorld)                              /* :2355 */
 2. SetGWorld(backSrcMap, nil)                                   /* :2356 — everything below draws into backSrcMap */
 3. if (noRoomAtAll || !houseUnlocked):  return                   /* :2358-2359 — !!! see below */
 4. else:
      if (GetNumberOfLights(thisRoomNumber) <= 0):                /* :2362 — the DARK ROOM effect */
          PenMode(srcOr)                                          /* :2364 */
          PenPat(GetQDGlobalsGray(&dummyPattern))                 /* :2365 — 50 % checkerboard */
          PaintRect(&backSrcRect)                                 /* :2366 */
          PenNormal()                                             /* :2367 */
      for (i = 0; i < kMaxRoomObs; i++):                          /* :2370 — all 24 slots */
          switch (thisRoom->objects[i].what):                     /* :2372 */
              case kObjectIsEmpty:  break                         /* :2374-2375 */
              ... 116 arms, each `DrawXxx(&roomObjectRects[i])` ...
              default:              break                         /* :2700-2701 */
 5. SetGWorld(wasCPort, wasWorld)                                 /* :2706 */
 6. if (isFirstRoom):                                             /* :2708 */
      CopyMask(glidSrcMap, glidMaskMap, backSrcMap,
               &gliderSrc[0], &gliderSrc[0], &initialGliderRect)   /* :2710-2713 */
 7. CopyMask(blowerSrcMap, blowerMaskMap, backSrcMap,
             &leftStartGliderSrc,  &leftStartGliderSrc,  &leftStartGliderDest)   /* :2716-2719 */
 8. CopyMask(blowerSrcMap, blowerMaskMap, backSrcMap,
             &rightStartGliderSrc, &rightStartGliderSrc, &rightStartGliderDest)  /* :2721-2724 */
 9. CopyBits(backSrcMap, workSrcMap, &backSrcRect, &backSrcRect, srcCopy, nil)   /* :2726-2728 */
```

**The early return at `:2358`–`:2359` does not restore the GWorld.** `SetGWorld(backSrcMap, nil)`
has already run at `:2356`, and `:2359` returns before `:2706`'s
`SetGWorld(wasCPort, wasWorld)`. So when a house is locked or there is no room, the current GWorld
is left pointing at `backSrcMap` and **all subsequent QuickDraw goes into the back buffer instead of
the screen**. Every caller of `DrawThisRoomsObjects` in the editor is guarded by
`houseUnlocked` upstream, which is why the bug never shows — but any port that structures this as
"set target, draw, restore target" with an early exit must use a deferred restore
(`defer restore()` in Go), which is the natural Go idiom anyway.

**The dark-room effect** at `:2362`–`:2368` is a whole-buffer 50 % gray `srcOr` overlay applied
**before** the objects. `srcOr` on an 8-bit indexed pixmap ORs the *colour indices*, which is not a
meaningful colour operation — it is a Mac Toolbox artifact that happens to darken the CLUT-mapped
result for this particular palette. A Go port cannot reproduce it by alpha-blending; it must OR the
palette indices and then look up the CLUT, exactly as QuickDraw did. Because the overlay is applied
first, the room's *objects* are drawn at full brightness on top of a dimmed background — which is the
editor's whole point (you must be able to see what you are editing in an unlit room).
`GetNumberOfLights(thisRoomNumber) <= 0` counts the room's light objects; see `rendering.md`.

**The three glider markers** are drawn *after* the GWorld is restored, but their `CopyMask`
destination is still `backSrcMap`, so they land in the back buffer too. Two details:

| Marker | Source map | Source rect | Size | Sheet offset |
|---|---|---|---|---|
| initial glider (first room only) | `glidSrcMap` / `glidMaskMap` | `gliderSrc[0]` | 48 × 20 | frame 0 of the glider sheet |
| left start | `blowerSrcMap` / `blowerMaskMap` | `leftStartGliderSrc` | 48 × 16 | (0, **358**) — `StructuresInit.c:280`–`281` |
| right start | `blowerSrcMap` / `blowerMaskMap` | `rightStartGliderSrc` | 48 × 16 | (0, **374**) — `StructuresInit.c:283`–`284` |

The two entry markers come out of the **blower** sprite sheet, not the glider sheet — an easy thing
to get wrong when building the asset tables. The `Src` rects are used as *both* the source and the
mask rect (the same `Rect` is passed twice), which is the standard `CopyMask` idiom here.

**Step 9** copies the entire finished back buffer to `workSrcMap` with `srcCopy`. `workSrcMap` is what
the window update handler blits to the screen, so this is the commit point. `backSrcRect` is
`houseRect` zeroed to the origin (`StructuresInit2.c:160`–`161`).

`DrawThisRoomsObjects` **never reads `objActive`** and never draws the selection. Selection
rendering is entirely the marquee's job (§3.3, §4), which is why a redraw cannot lose the selection —
it simply is not part of the drawing at all.

### 14.2 `HiliteAllObjects`

`GliderPRO/Sources/ObjectEdit.c:2734`–`2765`. The "show me every object's bounding box" gesture.

```
HiliteAllObjects():                                          /* :2734 */
 1. #ifndef COMPILEDEMO                                      /* :2736 */
 2. if (theMode != kEditMode)  return                         /* :2741-2742 */
 3. PauseMarquee()                                            /* :2744 */
 4. SetPort((GrafPtr)mainWindow)                              /* :2745 */
 5. PenPat(GetQDGlobalsGray(&dummyPattern))                   /* :2746 */
 6. PenMode(patXor)                                           /* :2747 */
 7. for (i = 0; i < kMaxRoomObs; i++) FrameRect(&roomObjectRects[i])   /* :2749-2750 */
 8. do { GetKeys(theseKeys); }
    while (BitTst(&theseKeys, kCommandKeyMap) && BitTst(&theseKeys, kOptionKeyMap))  /* :2752-2757 */
 9. for (i = 0; i < kMaxRoomObs; i++) FrameRect(&roomObjectRects[i])   /* :2759-2760 */
10. PenNormal()                                               /* :2762 */
11. ResumeMarquee()                                           /* :2763 */
```

Facts:

* **Trigger:** held **Command + Option**, tested at the top of `HandleEvent`
  (`GliderPRO/Sources/Events.c:487`–`491`) *before* `WaitNextEvent`. It is the only caller.
  `kCommandKeyMap` = **48**, `kOptionKeyMap` = **61** (`GliderPRO/Headers/Externs.h:155`, `:161`) —
  these are **raw key-map bit offsets** into the 128-bit `KeyMap` returned by `GetKeys`, not virtual
  key codes and not modifier-flag masks. A Go port on any modern platform must map its own modifier
  state to these two indices; the full set of Glider PRO's key-map offsets is at
  `Externs.h:115`–`181` (keypad `:115`–`127`, arrows `:129`–`132`, letters/modifiers `:134`–`163`,
  function keys `:165`–`181`).
* **Step 8 is a blocking busy-wait.** No `WaitNextEvent`, no yield. The whole application is frozen,
  spinning on `GetKeys`, for as long as the user holds the two modifiers. On cooperative-multitasking
  System 7 this also starves every other process. A Go port must instead render the overlay while a
  modifier-held flag is set and drop it when the flag clears, driven from the normal event loop.
* **Steps 7 and 9 are the same loop.** `patXor` makes `FrameRect` self-inverse, so drawing twice
  restores the screen exactly — the same XOR technique the marquee uses (§4.1). This is why the
  function does not need to know what was underneath.
* **It frames all 24 rects, including the sentinels.** `FrameRect` on `(-2,-2,-1,-1)` is a legal
  1 × 1 frame at (−2,−2), entirely outside the port's clip region, so nothing is drawn. On
  `(-1,-1,0,0)` (a freshly deleted slot, §13.2) likewise. A port that iterates and calls a naive
  rectangle-outline routine must clip, or it will draw a stray pixel at the window origin.
* The pen pattern is QuickDraw's global **`gray`** (50 % checkerboard) via the Carbon accessor
  `GetQDGlobalsGray`. Combined with `patXor` this produces a dotted inverted outline, visually
  distinct from the marquee's marching ants (which use the seven `'PAT#'` 128 patterns, §1.6).
* `PauseMarquee()` / `ResumeMarquee()` bracket the whole thing so the marching-ants animation does
  not XOR-interleave with the highlight frames. See §3.3 for the pause/resume protocol and the
  `paused`-flag leak.

### 14.3 `GoToObjectInRoom` / `GoToObjectInRoomNum`

`GliderPRO/Sources/ObjectEdit.c:2769`–`2799` and `:2803`–`2809`. Two of the five functions **not**
wrapped in `#ifndef COMPILEDEMO` at file scope (§0) — the guard is inside the body instead
(`:2771`, and `GoToObjectInRoomNum` has none at all because it only calls the guarded one).

```
GoToObjectInRoom(object, floor, suite):                      /* :2769 */
 1. if (!RoomExists(suite, floor, &itsNumber))  return         /* :2774 */
 2. if (itsNumber != thisRoomNumber):                          /* :2776 */
       CopyRoomToThisRoom(itsNumber)                           /* :2778 */
       DeselectObject()                                        /* :2779 */
       ReflectCurrentRoom(false)                               /* :2780 */
    else:
       DeselectObject()                                        /* :2783 */
 3. if (thisRoom->objects[object].what != kObjectIsEmpty):     /* :2785 */
       objActive = object                                      /* :2787 */
       if (ObjectHasHandle(&direction, &dist)):                /* :2788 */
           StartMarqueeHandled(&roomObjectRects[objActive], direction, dist)   /* :2790 */
           HandleBlowerGlider()                                /* :2791 */
       else:
           StartMarquee(&roomObjectRects[objActive])            /* :2794 */
       UpdateMenus(false)                                      /* :2795 */

GoToObjectInRoomNum(object, roomNum):                         /* :2803 */
 1. if (GetRoomFloorSuite(roomNum, &floor, &suite))            /* :2807 */
       GoToObjectInRoom(object, floor, suite)                  /* :2808 */
```

This is the **canonical, correct** room-change-plus-select sequence, and the model a port should
follow: deselect, load the room, reflect it, *then* set `objActive` and re-arm the marquee from the
**freshly rebuilt** `roomObjectRects[objActive]` (rebuilt inside `CopyRoomToThisRoom` →
`Room.c:335`). Note step 3 reads `roomObjectRects[objActive]` **after** the rebuild, and reads it in
the `else` branch too (same room, cache already valid).

Two hazards a port inherits if it copies this literally:

1. **`object` is not range-checked.** `thisRoom->objects[object]` at `:2785` and
   `roomObjectRects[objActive]` at `:2790`/`:2794` will index out of bounds for `object < 0` or
   `object >= 24`. The two callers are the link-following commands in the Object Info dialogs, which
   pass `data.d.who` / `data.e.who` — a **`Byte`, range 0..255**. A house with a corrupt or
   hand-edited `who` field ≥ 24 reads past the object array (into the *next room's* header, since
   rooms are contiguous in the house buffer) and can arm a marquee on garbage geometry.
2. **`objActive` is left as `kNoObjectSelected` when the target slot is empty**, because step 3's
   guard fails and `DeselectObject()` already ran. The user's original selection is lost with no
   feedback — no `SysBeep`, no dialog. Following a dangling link silently clears the selection.

---

## 15. Undo: there is none

Searched exhaustively. **Glider PRO 1.0.4 has no undo, at any level, for any editor operation.**

Evidence:

| Claim | Evidence |
|---|---|
| No undo command in the Edit menu handler | `GliderPRO/Sources/Menu.c` — the `iUndo` case is absent from the `switch` on menu item |
| No saved-state snapshot anywhere | No allocation of a second `roomType`/`houseType` for undo purposes in `StructuresInit.c` or `StructuresInit2.c` (`CreatePointers`, `:187`–`299`, allocates 18 blocks, none of them an undo buffer) |
| Cut / Copy / Paste bodies are commented out | `GliderPRO/Sources/Menu.c` — `iCopy` and `iPaste` have their bodies commented out; `iCut` and `iClear` delegate straight to `DeleteObject()` / `DeleteRoom()` |
| Every mutating gesture writes `thisRoom->objects[]` in place | §5.4, §6.1, §7.6, §8.3, §9 — all in-place, no copy-on-write |
| `fileDirty` is one-way | Set `true` by every gesture (§11.3); cleared only by a successful save |

So the **only** recovery from a mis-edit is *Revert*, i.e. close the house without saving and reopen
it. There is no per-object, per-room or per-gesture rollback.

Two consequences that matter for a port:

1. **The defects catalogued in this document are unrecoverable in the original.** In particular
   `DeleteObject`'s failure to compact the slot array (§9.1), the `kTV` parity-after-containment
   push to `topLeft.h == -1` (§12.10), the `kCustomPict` silent rewrite of `data.g.height` to 10000
   (§13.5) and the `objActive == -3` room-header corruption (§12.11) all leave the in-memory house
   permanently wrong until the user notices and discards.
2. **A Go port gets undo almost for free** and should take it. `sizeof(roomType)` is **348 bytes**
   (§11.1), and `kMaxRoomObs` × `sizeof(objectType)` is 24 × 12 = 288 of those. Snapshotting the
   whole 348-byte room before each gesture costs nothing and makes every defect above recoverable.
   Note that undo must snapshot **more than the room**: the mutating gestures also write
   `(*thisHouse)->initial` (§11.3), `thisRoom->leftStart`/`rightStart`, `numObjects`, and
   `retroLinkList` entries in other rooms (`DeleteObject`, §9.1). A room-only snapshot would restore
   an inconsistent house.

---

## 16. The tools windoid: `Tools.c`

`GliderPRO/Sources/Tools.c`, 544 lines. Four of its functions are named by no other analysis doc:
`CreateToolsOffscreen`, `KillToolsOffscreen`, `DrawToolTiles`, `DrawToolName`. All four are inside
`#ifndef COMPILEDEMO`.

### 16.1 Constants

All 32 `#define`s, `GliderPRO/Sources/Tools.c:15`–`46`:

| Constant | Value | Meaning |
|---|---:|---|
| `kToolsHigh` | 4 | palette rows |
| `kToolsWide` | 4 | palette columns |
| `kTotalTools` | **16** | `kToolsHigh * kToolsWide`; slot 0 is the selection arrow, slots 1–15 are objects |
| `kPopUpControl` | 129 | `'CNTL'` resource ID of the mode pop-up menu |
| `kFirstBlower` | 1 | first tool index for blower mode |
| `kLastBlower` | 15 | last tool index for blower mode |
| `kBlowerBase` | **1** | `'STR#'` string index base for blower names |
| `kFirstFurniture` | 1 | |
| `kLastFurniture` | 15 | |
| `kFurnitureBase` | **21** | |
| `kFirstBonus` | 1 | |
| `kLastBonus` | 15 | |
| `kBonusBase` | **41** | |
| `kFirstTransport` | 1 | |
| `kLastTransport` | **12** | the only mode with fewer than 15 tools besides switch/light/enemy |
| `kTransportBase` | **61** | |
| `kFirstSwitch` | 1 | |
| `kLastSwitch` | **9** | |
| `kSwitchBase` | **81** | |
| `kFirstLight` | 1 | |
| `kLastLight` | **8** | |
| `kLightBase` | **101** | |
| `kFirstAppliance` | 1 | |
| `kLastAppliance` | **14** | |
| `kApplianceBase` | **121** | |
| `kFirstEnemy` | 1 | |
| `kLastEnemy` | **9** | |
| `kEnemyBase` | **141** | |
| `kFirstClutter` | 1 | |
| `kLastClutter` | **15** | |
| `kClutterBase` | **161** | |
| `kToolsPictID` | **1011** | `'PICT'` holding the 15 × 9 grid of 24 × 24 tool icons |

The `*Base` values are spaced **20 apart** (1, 21, 41, 61, 81, 101, 121, 141, 161) even though no mode
has more than 15 tools — 5 unused string slots per mode. **That spacing is a fossil and is never
used**: `DrawToolName` computes its `'STR#'` index with a hard-coded stride of `0x0010` = **16**
(`Tools.c:147`), and the nine `k*Base` constants are copied only into `objectBase`, which is written
nine times and never read (§16.6, §16.8). A port should delete them. `kObjectNameStrings` = **1007**
(the `'STR#'` resource ID, `GliderDefines.h:461`).

Mode constants (`GliderPRO/Headers/GliderDefines.h`): `kSelectTool` = **0**, then
`kBlowerMode` 1, `kFurnitureMode` 2, `kBonusMode` 3, `kTransportMode` 4, `kSwitchMode` 5,
`kLightMode` 6, `kApplianceMode` 7, `kEnemyMode` 8, `kClutterMode` 9.

### 16.2 Geometry

| Rect | Value | Line |
|---|---|---|
| `toolsWindowRect` | `(0, 0, 116, 152)` — the source comment says `// 143` | `Tools.c:286` |
| `toolTextRect` | `(0, 0, 116, 12)`, `InsetRect(-1, -1)`, then `QOffsetRect(0, 157 - 15)` = `QOffsetRect(0, 142)` ⇒ `(-1, 141, 117, 155)` | `Tools.c:287`–`289` |
| `toolSrcRect` | `(0, 0, 360, 216)` — 15 columns × 24 px by 9 rows × 24 px | `Tools.c:82` |
| `toolRects[0..15]` | 28 × 28 cells on a 28 px pitch starting at (2, 29) | `Tools.c:321`–`326` |

`toolRects` construction (`Tools.c:321`–`326`):

```c
for (v = 0; v < kToolsHigh; v++)
    for (h = 0; h < kToolsWide; h++)
    {
        QSetRect(&toolRects[(v * kToolsWide) + h], 2, 29, 30, 57);   /* 28 × 28 at (2, 29) */
        QOffsetRect(&toolRects[(v * kToolsWide) + h], h * 28, v * 28);
    }
```

So cell *(v, h)* occupies `left = 2 + 28h`, `top = 29 + 28v`, 28 × 28. The 4 × 4 grid spans
x ∈ [2, 114), y ∈ [29, 141) — which is exactly why `toolTextRect` is offset to y = 142: the name
strip sits immediately below the last row.

`QSetRect` takes its arguments in **`(left, top, right, bottom)`** order (`RectUtils.c:210`–`216`), so
`QSetRect(&toolsWindowRect, 0, 0, 116, 152)` makes the windoid **116 px wide and 152 px high** — not
the other way round. The 4 × 4 grid (x up to 114) fits the 116 px width with 1 px of margin on each
side, and `toolTextRect`'s `InsetRect(-1, -1)` deliberately overhangs both side edges by 1 px so its
frame falls outside the visible area. Vertically, however, `toolTextRect` ends at y = **155**, 3 px
below the window's bottom at y = 152, so the bottom 3 rows of the name strip are clipped; the text
baseline (`bottom - 6` = 149) is still inside. The `// 143` comment on `:286` is a fossil of an
earlier, shorter window height.

`toolSrcRect` is **360 × 216**: the `'PICT'` 1011 sheet is a 15-wide × 9-tall grid of 24 × 24 icons,
one row per object mode (`kBlowerMode` 1 … `kClutterMode` 9).

### 16.3 `CreateToolsOffscreen`

`Tools.c:72`–`89`, wrapped in `#ifndef COMPILEDEMO` at `:71`/`:90`:

```
CreateToolsOffscreen():                                                    /* :72 */
 1. if (toolSrcMap == nil)                                                 /* :78 */
 2. {                                                                      /* :79 */
 3.     GetGWorld(&wasCPort, &wasWorld)                                    /* :80 */
 4.     QSetRect(&toolSrcRect, 0, 0, 360, 216)                             /* :82 */
 5.     theErr = CreateOffScreenGWorld(&toolSrcMap, &toolSrcRect,
 6.                                    kPreferredDepth)                    /* :83 */
 7.     SetGWorld(toolSrcMap, nil)                                         /* :84 */
 8.     LoadGraphic(kToolsPictID)                                          /* :85 — PICT 1011 */
 9.     SetGWorld(wasCPort, wasWorld)                                      /* :87 */
10. }                                                                      /* :88 */
```

Note the guard at `:78` is a **positive test wrapping the whole body**, not an early return — the
function is idempotent, and `CreateToolsOffscreen` is called every time the tools windoid is opened
(`Tools.c:328`). `theErr` is assigned at `:83` and **never tested**: an allocation failure leaves
`toolSrcMap` non-nil-but-broken or nil, and the palette silently draws nothing.

`kPreferredDepth` is **8** (8-bit indexed colour). There is **no mask GWorld** — tool icons are
drawn with plain `srcCopy`, so the sheet's background colour is copied along with the icon; the
28 × 28 cell with a 24 × 24 icon inset by 2 px is what provides the visual gutter.

A Go port replaces the GWorld with an in-memory paletted image (`image.Paletted`, 360 × 216) decoded
from `PICT` 1011, and `LoadGraphic` with a PICT decoder. `CreateOffScreenGWorld` allocates
360 × 216 × 1 byte = **77 760 bytes** plus a `CTabHandle`.

### 16.4 `KillToolsOffscreen`

`Tools.c:95`–`103`, verbatim:

```c
if (toolSrcMap != nil)              /* :97 */
{
    DisposeGWorld(toolSrcMap);      /* :99 */
//  KillOffScreenPixMap(toolSrcMap);/* :100 — commented out, the pre-GWorld API */
    toolSrcMap = nil;               /* :101 */
}
```

The commented-out `KillOffScreenPixMap` at `:100` is a fossil of the original 1994 offscreen-PixMap
implementation, left in place *after* the `DisposeGWorld` that replaced it. Setting the pointer to
`nil` at `:101` is what makes `CreateToolsOffscreen`'s `== nil` guard work, so the pair is a correct
lazy-allocate / explicit-free cycle. Nothing else in the program touches `toolSrcMap`.

### 16.5 `DrawToolTiles`

`Tools.c:161`–`180`:

```
DrawToolTiles():
 1. DrawCIcon(2000, toolRects[0].left, toolRects[0].top)      /* :166 — selection arrow, cell 0 */
 2. for (i = 0; i < 15; i++):
      QSetRect(&srcRect,  0, 0, 24, 24)
      QSetRect(&destRect, 0, 0, 24, 24)
      QOffsetRect(&srcRect,  i * 24, (toolMode - 1) * 24)      /* pick column i, row toolMode-1 */
      QOffsetRect(&destRect, toolRects[i + 1].left + 2,
                             toolRects[i + 1].top  + 2)        /* 2 px inset in the 28 px cell */
      CopyBits(toolSrcMap → toolsWindow port, &srcRect, &destRect, srcCopy, nil)
```

Details:

* Cell 0 is **not** from the sheet — it is a `'cicn'` colour icon, resource ID **2000**, drawn by
  `DrawCIcon` at the cell's top-left with **no** 2 px inset. So the selection arrow is 28 × 28 and
  the 15 object icons are 24 × 24 inset by 2. A port must keep the two paths distinct.
* The source row is `(toolMode - 1) * 24`, so `toolMode` 1..9 selects sheet rows 0..8.
* The loop always draws **15** icons, even for modes with fewer tools (`kTransportMode` has 12,
  `kSwitchMode` 9, `kLightMode` 8, `kEnemyMode` 9, `kApplianceMode` 14). The trailing cells show
  whatever is in the sheet at those columns — in the shipped `PICT` 1011 those columns are blank for
  the short rows. So the palette is **always a full 4 × 4 grid**; it does not shrink. Clicks on the
  blank cells are rejected by `HandleToolsClick`'s range test against `kLast*`.
* `CopyBits` here targets the tools windoid's port directly (not an offscreen), so the palette is
  redrawn synchronously on every mode change and every update event.

### 16.6 `DrawToolName`

`Tools.c:139`–`155`. `GetIndString` is at `:146`–`147`, `ColorText` at `:154`:

```
DrawToolName():
 1. if (toolSelected == 0):
        PasStringCopy("\pSelection Tool", theString)                 /* hard-coded, NOT localised */
    else:
        GetIndString(theString, kObjectNameStrings,
                     toolSelected + ((toolMode - 1) * 0x0010))        /* :146-147 */
 2. EraseRect(&toolTextRect)
 3. MoveTo(toolTextRect.left + 3, toolTextRect.bottom - 6)
 4. TextFont(applFont); TextSize(9); TextFace(bold)
 5. ColorText(theString, 171L)
```

The index arithmetic at `:146` is the whole design of the tool palette, and it is worth stating
precisely, because it is the **same expression** `DoNewObjectClick` uses to compute the new object's
`what` code (`GliderPRO/Sources/ObjectEdit.c:781`):

```c
whatObject = toolSelected + ((toolMode - 1) * 0x0010);       /* ObjectEdit.c:781 */
GetIndString(theString, kObjectNameStrings,
             toolSelected + ((toolMode - 1) * 0x0010));      /* Tools.c:146-147 */
```

Therefore:

> **`'STR#'` 1007 index == `objectType.what` == `toolSelected + (toolMode - 1) * 16`.**
>
> Equivalently, the `what` code is a **packed nibble pair**:
> `toolMode = (what >> 4) + 1` and `toolSelected = what & 0x0F`.

**Verified** by parsing `'STR#'` 1007 out of the derez'ed resource fork
(`GliderPRO/Glider PRO.r:279`-`380`; the resource is titled `"Object Names"`). It contains
**144** Pascal strings totalling **1569** bytes, and the index/`what` correspondence is exact:

| Index | `'STR#'` 1007 string | `what` | Constant |
|---:|---|---:|---|
| 1 | `Floor Vent` | 0x01 | `kFloorVent` |
| 6 | `Table Fan` | 0x06 | `kLeftFan` |
| 7 | `Table Fan` | 0x07 | `kRightFan` |
| 16 | `Lift Area` | 0x10 | `kLiftArea` |
| 17 | `Table` | 0x11 | `kTable` |
| 31 | `Invisible Rebounder` | 0x1F | `kInvisBounce` |
| 32 | `f` | 0x20 | *(no such type)* |
| 33 | `Digital Clock` | 0x21 | `kRedClock` |
| 47 | `Slide Rect` | 0x2F | `kSlider` |
| 48 | `f` | 0x30 | *(no such type)* |
| 49 | `Up Stairs` | 0x31 | `kUpStairs` |
| 64 | `Deluxe Transport` | 0x40 | `kDeluxeTrans` |
| 73 | `Sound Trigger` | 0x49 | `kSoundTrigger` |
| 74 - 80 | `9` `a` `b` `c` `d` `e` `f` | 0x4A - 0x50 | *(no such types)* |
| 81 | `Ceiling Light` | 0x51 | `kCeilingLight` |
| 101 | `T.V.` | 0x65 | `kTV` |
| 110 | `Custom Picture` | 0x6E | `kCustomPict` |
| 133 | `Flower` | 0x85 | `kFlower` |
| 143 | `Wind Chimes` | 0x8F | `kChimes` |
| **144** | **`Mermaid`** | 0x90 | **none - see below** |

So **the 16-stride is correct** and the nine `k*Base` constants (spaced 20 apart, §16.1) are **dead
in `Tools.c`** - they are written only into `objectBase`, which is never read (§16.8). They are a
fossil of an abandoned 20-per-mode string layout. A port should delete them.

Three findings fall out of this:

1. **The unused `what` codes have placeholder names.** Every index whose `what` code does not exist
   holds a single ASCII character - the low nibble printed as a hex digit
   (`8`, `9`, `a`, `b`, `c`, `d`, `e`, `f`). These are the author's spacers, and they are what
   `DrawToolName` would display if the palette ever selected an unused code. Empirically, of the 143
   codes in 1..143, **26 are never used by any of the 22 shipped houses**: 0x20, 0x30, 0x4A-0x50,
   0x59-0x60, 0x6F, 0x70, 0x7A-0x80. The other **117** all occur - matching the 117-label count used
   by every `what` switch in this document, and the 117 rows of `object-taxonomy.md`.
2. **Index 144 = `Mermaid` (0x90) is a cut object.** There is no `#define` for it in
   `GliderDefines.h`, no `case` label anywhere in the program, and no `srcRects` entry -
   `kNumSrcRects` is **0x90 = 144**, so `srcRects` is indexed 0..143 and `srcRects[0x90]` would be
   out of bounds. The name survived in the resource; the object did not ship.
3. **`srcRects[kFlower]` (0x85 = 133) is never initialised.** `InitSrcRects`
   (`GliderPRO/Sources/StructuresInit2.c:306`-`475`) sets 116 of the 117 entries and skips
   `kFlower`; its neighbours `kFireplace` (`:463`) and `kWallWindow` (`:464`) are both present.
   Because `srcRects` comes from `NewPtr` (`StructuresInit2.c:270`-`271`, 144 x 8 = **1152 bytes,
   not zeroed** - `NewPtr` does not clear memory, unlike `NewPtrClear`), `srcRects[kFlower]` is
   **uninitialised heap garbage**. `kFlower` is a real object with 547 instances across the shipped
   houses; it is drawn from `flowerSrc[0..5]` (`StructuresInit2.c:72`-`88`, six sub-rects of the
   128 x 69 `clutterSrcRect`) instead, which is why nobody noticed. A Go port that zero-initialises
   its rect table changes the behaviour here from "garbage" to "empty rect"; either way, nothing
   *does* read `srcRects[kFlower]` - §13.7 enumerates all 14 `case kFlower:` sites and shows that
   every one of them uses `data.i.bounds` or `flowerSrc[pict]`.

Other details:

* `"\pSelection Tool"` is a **hard-coded Pascal string literal**, bypassing the `'STR#'` mechanism
  entirely — so the selection tool's name is the one piece of tool UI text that cannot be localised.
  Compare `GetLocalizedString` (`GliderPRO/Sources/StringUtils.c:321`–`327`), which reads `'STR#'`
  **150** (`kLocalizedStringsID`).
* `TextFont(applFont)` — `applFont` is **1** (the "application font" pseudo-ID); at 9 pt bold this is
  Geneva 9 bold on a US System 7. A port must ship a bitmap font or accept a metric mismatch;
  `StringWidth` differences will change the truncation behaviour of `CollapseStringToWidth`
  (`StringUtils.c:282`–`296`).
* `ColorText(theString, 171L)` draws in **CLUT index 171**.
  `ColorText` (`GliderPRO/Sources/ColorUtils.c:20`–`29`) is
  `GetForeColor(&wasColor); Index2Color(color, &theRGBColor); RGBForeColor(&theRGBColor);
  DrawString(theStr); RGBForeColor(&wasColor);` — it saves and restores the foreground colour, and
  goes through `Index2Color` on the **current device's** colour table, so the actual RGB depends on
  the loaded CLUT. See `graphics-assets.md` for the palette.
* `MoveTo(toolTextRect.left + 3, toolTextRect.bottom - 6)` = `MoveTo(2, 149)` with the values from
  §16.2 — a 3 px left margin from the *inset* rect and a 6 px baseline lift from the bottom.

### 16.7 The `toolIcon` remapping quartet, and which types the palette can reach

`toolSelected` is the **logical tool number** (1..16) and `toolIcon` is the **palette cell / sheet
column** (1..15). They are not the same, because two of the nine sheet rows have fewer distinct icons
than the mode has types. Four places convert between them:

| Site | Function | Direction |
|---|---|---|
| `Tools.c:115`–`125` | `FrameSelectedTool` | forward (`toolSelected` → cell) |
| `Tools.c:197`–`207` | `EraseSelectedTool` | forward |
| `Tools.c:230`–`240` | `SelectTool` | forward |
| `Tools.c:472`–`483` | `HandleToolsClick` | **inverse** (clicked cell → `toolSelected`) |

The forward transform, verbatim, three times copy-pasted:

```c
toolIcon = toolSelected;
if ((toolMode == kBlowerMode) && (toolIcon >= 7))
{
    toolIcon--;
}
else if ((toolMode == kTransportMode) && (toolIcon >= 7))
{
    if (toolIcon >= 15)                          /* NOTE: 15, not 11 */
        toolIcon -= 4;
    else
        toolIcon = ((toolIcon - 7) / 2) + 7;     /* integer division */
}
```

and the inverse, `Tools.c:472`–`483`, inside the cell hit-test loop (`SelectTool(toolIcon)` at `:484`):

```c
toolIcon = i;                                    /* i = clicked cell, 0..15 */
if ((toolMode == kBlowerMode) && (toolIcon >= 7))
{
    toolIcon++;
}
if ((toolMode == kTransportMode) && (toolIcon >= 7))   /* plain `if`, not `else if` */
{
    if (toolIcon >= 11)                          /* NOTE: 11, not 15 */
        toolIcon += 4;
    else
        toolIcon = ((toolIcon - 7) * 2) + 7;
}
SelectTool(toolIcon);
```

**The two thresholds differ on purpose** (15 forward, 11 inverse) and the two branch structures
differ (`else if` forward, plain `if` inverse — behaviourally identical, since `toolMode` cannot be
both). `forward(inverse(cell)) == cell` for every clickable cell in every mode; `inverse(forward(t))`
is *not* the identity, and must not be, because the forward map is deliberately many-to-one.

**Blower mode** (`toolMode` = 1, `lastTool` = `kLastBlower` = 15, `what` = `toolSelected`):

| Clicked cell | `toolSelected` | `what` | Constant | Sheet column (`cell - 1`) |
|---:|---:|---:|---|---:|
| 1 – 6 | 1 – 6 | 0x01 – 0x06 | `kFloorVent` … `kLeftFan` | 0 – 5 |
| 7 | 8 | 0x08 | `kTaper` | 6 |
| 8 | 9 | 0x09 | `kCandle` | 7 |
| 9 | 10 | 0x0A | `kStubby` | 8 |
| 10 | 11 | 0x0B | `kTiki` | 9 |
| 11 | 12 | 0x0C | `kBBQ` | 10 |
| 12 | 13 | 0x0D | `kInvisBlower` | 11 |
| 13 | 14 | 0x0E | `kGrecoVent` | 12 |
| 14 | 15 | 0x0F | `kSewerBlower` | 13 |
| 15 | **16** | **0x10** | **`kLiftArea`** | 14 |

Two consequences:

* **`kLiftArea` (0x10) *is* reachable** — cell 15 yields `toolSelected` = 16, and
  `16 + (1 - 1) * 16 = 16`. Without the `toolIcon++` it would be unreachable, since the palette has
  only 15 object cells for 16 blower types. 716 instances exist across the shipped houses.
* **`kRightFan` (0x07) is *not* reachable from the palette.** `inverse` skips 7 entirely, and
  `forward(6) == forward(7) == 6`, so both fans frame and highlight cell 6. `'STR#'` 1007 names
  index 6 and index 7 **both** `Table Fan` (§16.6) — the shared name is the tell. The right-facing
  fan is created only by the **Object Info dialog's** left/right control
  (`GliderPRO/Sources/ObjectInfo.c:1014` and `:1044`, guarded by `:967`/`:1009`/`:1039`
  `if ((what == kLeftFan) || (what == kRightFan))`). 54 instances exist across the shipped houses.

**Transport mode** (`toolMode` = 4, `lastTool` = `kLastTransport` = **12**, `what` = 48 + `toolSelected`):

| Clicked cell | `toolSelected` | `what` | Constant | Sheet column |
|---:|---:|---:|---|---:|
| 1 – 6 | 1 – 6 | 0x31 – 0x36 | `kUpStairs` … `kCeilingTrans` | 0 – 5 |
| 7 | 7 | 0x37 | `kDoorInLf` | 6 |
| 8 | 9 | 0x39 | `kDoorExRt` | 7 |
| 9 | 11 | 0x3B | `kWindowInLf` | 8 |
| 10 | 13 | 0x3D | `kWindowExRt` | 9 |
| 11 | 15 | 0x3F | `kInvisTrans` | 10 |
| 12 | 16 | 0x40 | `kDeluxeTrans` | 11 |

So `kDeluxeTrans` (0x40) is reachable for the same reason `kLiftArea` is. And the **four missing
partners** — `kDoorInRt` (0x38), `kDoorExLf` (0x3A), `kWindowInRt` (0x3C), `kWindowExLf` (0x3E) —
have no palette cell of their own. `'STR#'` 1007 again gives them away: 55/56 are both
`Door (interior)`, 57/58 both `Door (exterior)`, 59/60 both `Window (interior)`, 61/62 both
`Window (exterior)`.

They are reached three ways:

1. **At placement, from the click x.** `AddNewObject` (`GliderPRO/Sources/ObjectAdd.c:368`–`437`)
   *overrides* the `what` it was passed for all eight door/window types:

   ```c
   if ((what == kDoorInLf) || (what == kDoorInRt))
   {
       if (where.h > (kRoomWide / 2))                  /* > 256 */
       { what = kDoorInRt;  ...topLeft.h = kDoorInRtLeft;  ...topLeft.v = kDoorInTop; }
       else
       { what = kDoorInLf;  ...topLeft.h = kDoorInLfLeft;  ...topLeft.v = kDoorInTop; }
   }
   else if ((what == kDoorExRt) || (what == kDoorExLf)) { ... }
   ```

   So **`DoNewObjectClick`'s `whatObject` is only a proposal** — `AddNewObject` may replace it. Click
   in the left half of the room and you get the Lf variant, right half the Rt variant, and the
   forced `topLeft` from §12.4 is applied immediately rather than waiting for `KeepObjectLegal`.
2. **On drag**, by `KeepObjectLegal`'s door/window `what`-flip (§12.7).
3. **At pairing**, by `AddObjectPairing` (`GliderPRO/Sources/ObjectEdit.c:792`–`1150`), which creates the
   matching partner in the linked room.

**Palette reachability accounting** over all nine modes:

| `toolMode` | Mode | `lastTool` | Cells 1..`lastTool` | Distinct `what` reachable |
|---:|---|---:|---:|---:|
| 1 | `kBlowerMode` | 15 | 15 | 15 (of 16 — `kRightFan` missing) |
| 2 | `kFurnitureMode` | 15 | 15 | 15 (0x11 – 0x1F, all) |
| 3 | `kBonusMode` | 15 | 15 | 15 (0x21 – 0x2F, all) |
| 4 | `kTransportMode` | **12** | 12 | 12 (of 16 — 4 partners missing) |
| 5 | `kSwitchMode` | **9** | 9 | 9 (0x41 – 0x49, all) |
| 6 | `kLightMode` | **8** | 8 | 8 (0x51 – 0x58, all) |
| 7 | `kApplianceMode` | **14** | 14 | 14 (0x61 – 0x6E, all) |
| 8 | `kEnemyMode` | **9** | 9 | 9 (0x71 – 0x79, all) |
| 9 | `kClutterMode` | 15 | 15 | 15 (0x81 – 0x8F, all) |
| | | | | **112** |

**112 of the 117 types are directly placeable from the tool palette.** The other 5
(`kRightFan`, `kDoorInRt`, `kDoorExLf`, `kWindowInRt`, `kWindowExLf`) exist only via the three
indirect routes above. A Go port that builds the palette from a naive
`for i := 1; i <= 15; i++ { what = (mode-1)*16 + i }` will offer `kRightFan` as a duplicate
`Table Fan` cell and will be unable to place `kLiftArea` or `kDeluxeTrans` at all — the two
`toolIcon++` / `+= 4` adjustments are load-bearing, not cosmetic. The cleanest port is to drop the
`toolSelected`/`toolIcon` indirection entirely and store an explicit
`[9][]struct{what int; sheetCol int}` table built from the tables above.

The `i <= lastTool` test at `Tools.c:469` is what stops clicks on the blank trailing cells (§16.5:
`DrawToolTiles` always draws 15 icons regardless of `lastTool`). Note it is `<=`, and the loop runs
`i` from **0**, so cell 0 (the selection arrow) is always clickable and `SelectTool(0)` selects
`kSelectTool`.

### 16.8 `ToggleToolsWindow`'s `isToolsOpen` bug, and the dead `objectBase`

`Tools.c:351`–`365`:

```
ToggleToolsWindow():
    if (toolsWindow == nil)     /* :354  — dispatches on the WINDOW, not on isToolsOpen */
    {
        OpenToolsWindow();
        isToolsOpen = true;     /* :357  — correct */
    }
    else
    {
        CloseToolsWindow();
        isToolsOpen = true;     /* :362  — WRONG, should be false */
    }
```

Two things to get right here, both easy to invert:

* The branch condition is **`toolsWindow == nil`**, not `isToolsOpen`. The toggle therefore *works*:
  it never double-closes, and the stale flag cannot make it misbehave. The offending assignment is the
  one at **`:362`** (the close branch); `:357` is the correct one.
* Because the close branch never clears the flag, `isToolsOpen` is **write-once-true**: after the
  first open it stays true for the rest of the session and is persisted at quit
  (`Main.c:264`, `thePrefs.wasToolsOpen = isToolsOpen`) and read back at launch (`Main.c:109`; the
  first-run default is also `true`, `Main.c:180`). The observable bug is at **`Menu.c:800`**
  (`if (isToolsOpen) OpenToolsWindow();`, inside the `theMode == kEditMode && houseUnlocked` arm):
  closing the tools windoid, leaving edit mode and coming back **re-opens it**, and so does the next
  launch. `isMapOpen` and `isCoordOpen`, sitting either side of it in the same block, do not have this
  problem.

Contrast `ToggleCoordinateWindow` (`GliderPRO/Sources/Coordinates.c:181`–`195`), which has the
identical `coordWindow == nil` structure and correctly sets `isCoordOpen = false` at `:192` — so this
really is a one-word typo, not a convention.

`SwitchToolModes` (`Tools.c:370`–`435`) has 9 arms, one per object mode (`:378`–`430`), and each arm
sets exactly three globals: `firstTool = kFirst*`, `lastTool = kLast*`, `objectBase = k*Base`.
`toolSelected` is *not* set by the arms — it is reset by the `SelectTool(kSelectTool)` at `:375`,
before the switch — and `toolMode = newMode` happens at `:433`, after it.

Of the three globals (declared together at `Tools.c:64`), **only `lastTool` is ever read**, at
`Tools.c:469`. `objectBase` and `firstTool` are written nine times each and never read anywhere in the
program — grep the whole tree. A port should drop both and keep `lastTool`.

The switch also has **no `default:`**, so calling `SwitchToolModes` with `newMode` outside 1..9 leaves
`lastTool` at its previous mode's value while `toolMode` is updated anyway at `:433` — the palette
would then draw row `newMode - 1` of the sheet but accept clicks according to the *old* mode's
`lastTool`. All five call sites pass a value already in 1..9, so it is unreachable: `:330`
(`OpenToolsWindow`, from prefs), `:462` (`HandleToolsClick`, from the pop-up's `GetControlValue`),
`:504` / `:523` (`NextToolMode` / `PrevToolMode`, both range-guarded at `:499` / `:518`) and `:540`
(`SetSpecificToolMode`, whose nine callers at `Events.c:281`–`329` all pass a `k*Mode` literal).

---

## 17. `DrawClockDigit` and the real-time-clock dependency

`GliderPRO/Sources/ObjectDraw.c:990`–`995`, prototype at `:51`. Six lines, and the only place in the
program that renders numerals from a sprite sheet.

```c
void DrawClockDigit (short number, Rect *dest)
{
    CopyBits((BitMap *)*GetGWorldPixMap(bonusSrcMap),
             (BitMap *)*GetGWorldPixMap(backSrcMap),
             &digits[number], dest, srcCopy, nil);
}
```

### 17.1 The `digits` table

Declared `Rect digits[11];` at `GliderPRO/Sources/Objects.c:36`, initialised at
`GliderPRO/Sources/StructuresInit.c:359`–`363`:

```c
for (i = 0; i < 11; i++)
{
    QSetRect(&digits[i], 0, 0, 4, 6);
    QOffsetRect(&digits[i], 28, i * 6);
}
```

So digit *i* is a **4 × 6** rect at **(28, i × 6)** inside `bonusSrcMap`:

| `i` | rect (l, t, r, b) |
|---:|---|
| 0 | (28, 0, 32, 6) |
| 1 | (28, 6, 32, 12) |
| 2 | (28, 12, 32, 18) |
| 3 | (28, 18, 32, 24) |
| 4 | (28, 24, 32, 30) |
| 5 | (28, 30, 32, 36) |
| 6 | (28, 36, 32, 42) |
| 7 | (28, 42, 32, 48) |
| 8 | (28, 48, 32, 54) |
| 9 | (28, 54, 32, 60) |
| 10 | (28, 60, 32, 66) |

**Eleven** entries, 0–10. Only 0–9 are ever passed by the four call sites, so entry 10 (at y 60–66) is
allocated but unused — presumably a blank or a colon glyph in the sheet. `bonusSrcRect` is
`(0, 0, 88, 378)` (`StructuresInit.c:350`, comment "33264 pixels"), so the digit strip occupies the
column x ∈ [28, 32) for the first 66 rows.

`number` is **not range-checked**. `digits[number]` for `number > 10` or `< 0` reads a `Rect` past the
array — but `DrawClockDigit` has exactly **one** caller, `DrawRedClock`, whose four call sites
(`ObjectDraw.c:979`, `:981`, `:983`, `:985`) pass `hour / 10`, `hour % 10`, `minutes / 10`,
`minutes % 10` with `hour ∈ 1..12` and `minutes ∈ 0..59`, so the range is 0..9 and safe.

### 17.2 `DrawRedClock`, the caller

`GliderPRO/Sources/ObjectDraw.c:959`–`986`:

```
DrawRedClock(theRect):
 1. CopyMask(bonusSrcMap, bonusMaskMap, backSrcMap,
             &srcRects[kRedClock], &srcRects[kRedClock], theRect)   /* draw the clock body, masked */
 2. GetTime(&timeRec)                                    /* Mac Toolbox: the REAL system clock */
 3. hour = timeRec.hour % 12;  if (hour == 0) hour = 12   /* 24 h → 12 h, midnight/noon → 12 */
 4. minutes = timeRec.minute
 5. QSetRect(&dest, 0, 0, 4, 6)
    QOffsetRect(&dest, theRect->left + 5, theRect->top + 7)   /* first digit cell */
 6. if (hour > 9)  DrawClockDigit(hour / 10, &dest)       /* tens of hours: only if 10, 11, 12 */
 7. QOffsetRect(&dest, 4, 0);  DrawClockDigit(hour % 10, &dest)      /* units of hours */
 8. QOffsetRect(&dest, 6, 0);  DrawClockDigit(minutes / 10, &dest)   /* tens of minutes */
 9. QOffsetRect(&dest, 4, 0);  DrawClockDigit(minutes % 10, &dest)   /* units of minutes */
```

The advance pattern, exactly:

| Step | advance before drawing | cumulative x offset from `theRect->left` |
|---|---:|---:|
| tens of hours (conditional) | — | +5 |
| units of hours | +4 | +9 |
| tens of minutes | **+6** | +15 |
| units of minutes | +4 | +19 |

So digits within a group are 4 px apart (exactly the glyph width, no letterspacing) and the
hour-to-minute gap is **6 px** — a 2 px colon gap. Total glyph run is 4 digits × 4 px + 6 = 22 px
starting 5 px in from the clock's left and 7 px down from its top. `srcRects[kRedClock]` is 28 × 17
(`StructuresInit2.c:357`), so the run fits with 1 px to spare on the right.

**Single-digit hours leave the tens cell blank but do not shift the units digit.** Step 6 is skipped,
step 7 still advances by 4 first, so 9:05 renders as ` 9:05` with a leading blank, not `9:05`
left-aligned. A port must reproduce the fixed layout, not centre the string.

**`kRedClock` is the *only* digital clock**, and `DrawRedClock` is the only user of `digits` /
`DrawClockDigit`. The other three clock objects are **analog** and share a different helper, so do not
assume they are copies of `DrawRedClock`:

| Function | Lines | Src rect | Hand origin | Hand helper |
|---|---|---|---|---|
| `DrawRedClock` | `ObjectDraw.c:959`–`986` | `srcRects[kRedClock]`, 28 × 17 (`StructuresInit2.c:357`) | — (4 digit cells) | `DrawClockDigit` ×4 |
| `DrawBlueClock` | `:999`–`1016` | `srcRects[kBlueClock]`, 28 × 25 at y 17 (`StructuresInit2.c:358`–`359`) | `(left + 13, top + 13)` | `DrawClockHands` (`:1062`) |
| `DrawYellowClock` | `:1020`–`1037` | `srcRects[kYellowClock]`, 28 × 28 at y 42 (`StructuresInit2.c:360`–`361`) | | `DrawClockHands` |
| `DrawCuckoo` | `:1041`–`1058` | `srcRects[kCuckoo]`, 40 × 80 (`StructuresInit2.c:362`) | `(left + 19, top + 31)` | `DrawLargeClockHands` (`:1178`) |

The three analog functions quantise differently too: `minutes = ((timeRec.minute + 2) / 5) % 12`
(nearest 5-minute tick, `:1014`/`:1056`), and they use `hour = timeRec.hour % 12` **without**
`DrawRedClock`'s `if (hour == 0) hour = 12` fix-up — because for a hand angle 0 and 12 are the same
position. `'STR#'` 1007 index 33 accordingly names `kRedClock` `Digital Clock` (§16.6).

### 17.3 Why this matters for a port: determinism

`GetTime(&timeRec)` reads the **host machine's wall clock**. So:

1. **Room rendering is not a pure function of house state.** Two runs of the same house at different
   times of day produce different pixels. Any port with pixel-diff regression tests must either
   inject a fake clock or exclude rooms containing any of the **four** clock objects — `kRedClock`,
   `kBlueClock`, `kYellowClock` and `kCuckoo` (`GetTime` is called at `ObjectDraw.c:970`, `:1012`,
   `:1033` and `:1054`).
2. **It is not a gameplay input.** The clock face is decorative; the clock objects' *bonus* behaviour
   (`kRedClock` = 100 points, etc.) does not consult `GetTime`. See `scoring.md`. So determinism of
   *simulation* is unaffected — only of *rendering*.
3. `DateTimeRec` is the classic-Mac broken-down time struct; `timeRec.hour` is 0–23 and
   `timeRec.minute` 0–59, in **local** time with no timezone information. Go's
   `time.Now().Hour()` / `.Minute()` is a direct substitute.
4. The clocks are re-rendered on every full room composite, so the displayed time updates whenever
   the room redraws — which in the editor is after every gesture, and in play mode every frame.

Empirically, `kRedClock` and `kBlueClock` are common enough that this is not a corner case:
`kRedClock` and `kBlueClock` together account for a large share of the 2 992 bonus objects across the
22 houses (see the bonus counts in §12.2's verification table).

---

## 18. Mac Toolbox inventory and what a Go port must replace it with

Every Toolbox facility this subsystem touches, with the exact call sites and the substitution a Go
port needs. Nothing in this table is optional: each row is load-bearing for at least one observable
behaviour documented above.

### 18.1 QuickDraw drawing state

| Toolbox call | Where it is used here | Semantics that matter | Go replacement |
|---|---|---|---|
| `PenMode(patXor)` | `Marquee.c:41` (`DoMarquee`), and `:67`, `:130`, `:151`, `:195`, `:225`, `:269`, `:352`; `ObjectEdit.c:2747` (`HiliteAllObjects`) | Pen pattern is **XORed** into the destination's *colour indices*. Drawing the same shape twice restores the original pixels exactly, with no saved background. This is the entire selection-rendering strategy (§4.1). | Do **not** emulate XOR. Draw the selection into a separate overlay layer composited over the room each frame. If you must match pixels, XOR the palette index (not the RGB). |
| `PenMode(srcOr)` | `ObjectEdit.c:2364` (dark-room dimming) | ORs the 50 % gray pattern's colour **indices** into the back buffer. On the Glider palette this darkens; on any other palette it would not. | OR the palette indices, then look up the CLUT. An alpha blend gives visibly different pixels. |
| `PenPat(GetQDGlobalsGray(&dummyPattern))` | `ObjectEdit.c:2365`, `:2746` | QuickDraw's global 50 % checkerboard `Pattern` — 8 bytes, one per raster row. The actual bytes live in QuickDraw's globals and are **not in this source tree**; `GetQDGlobalsGray` is the Carbon accessor that replaced reading `qd.gray` directly. | Hard-code the 8 bytes after reading them from the SDK, or generate a 50 % checkerboard. |
| `PenPat` from `GetIndPattern` | `Marquee.c:504` (`InitMarquee`, `kMarqueePatListID` = **128** at `Marquee.c:15`); applied at `:42`, `:47`, `:68`, `:131`, `:152`, `:196`, `:226`, `:270`, `:353` | Seven `'PAT#'` 128 patterns cycled to animate marching ants (§1.6). | Ship the 7 × 8 bytes as a literal table; do not depend on a resource file. |
| `PenSize(2, 2)` | `Tools.c:128` (`FrameSelectedTool`) and `:210` (`EraseSelectedTool`) — **only two sites**. `SelectTool` (`:218`–`:251`) calls `FrameRect` at `:244` **without setting the pen size**, then `PenNormal()` at `:245`. It draws a 2 px frame only because `EraseSelectedTool` sets `PenSize(2, 2)` at `:210` and **never restores it** — an inherited-state dependency. `FrameSelectedTool` does `PenNormal()` at `:131`, so it does not leak. | A 2 px pen makes `FrameRect` draw the border **inward** from the rect's top-left and **outward** past its bottom-right, because QuickDraw hangs the pen down-and-right of the pen location. | Draw an explicit 2 px inset/outset rectangle; a naive 2 px stroke centred on the path is 1 px off on two sides. |
| `PenNormal()` | after every pen change | Resets pattern to `black`, mode to `patCopy`, size to (1,1). | Restore your own saved state; prefer an explicit state struct over globals. |
| `FrameRect` | `ObjectEdit.c:2750`, `:2760`; `Tools.c:130`, `:212`, `:244` | Outlines the rect **excluding** `right` and `bottom` (half-open, see `PtInRect` below). | 4 line segments on the half-open rect. |
| `PaintRect` | `ObjectEdit.c:2366` | Fills with the current pen pattern **and mode**. | Pattern fill honouring the blend mode. |
| `EraseRect` | `Tools.c:149` (`DrawToolName`) | Fills with the port's **background** pattern/colour, not white. | Fill with the windoid's background colour. |
| `MoveTo` / `LineTo` | `Marquee.c` handle leader lines | Integer pixel endpoints; QuickDraw draws the pen down-right of the path. | Bresenham with the same pen-offset convention. |
| `InsetRect` / `OffsetRect` (`QSetRect`/`QOffsetRect`) | throughout | `Q*` variants are Glider's own inline macros in `RectUtils.c`; identical semantics, no Toolbox trap. | Trivial. Watch that `InsetRect` with a **negative** amount grows the rect (used at `Tools.c:288`). |
| `PtInRect` | `ObjectEdit.c:61` (`FindObjectSelected`), `Marquee.c:425`–`428` (`PtInMarqueeHandle`), `Tools.c:469` | **Half-open**: `left <= h < right` and `top <= v < bottom`. A point on the right or bottom edge is **outside**. | `p.X >= r.Min.X && p.X < r.Max.X && …` — Go's `image.Rectangle.Contains` already has exactly these semantics. |
| `ForeColor(redColor)` / `ForeColor(whiteColor)` / `ForeColor(blackColor)`, `DkGrayForeColor()` | red `Tools.c:129`, `:243`; white `:211`; black `:132`, `:246`, `:267`; dark gray `:264` | The classic QuickDraw named-colour constants, resolved through the current `GDevice`. **Their numeric values are not in this source tree** — they come from the Universal Headers' `QuickDraw.h`, which the GPL release does not include. Do not hard-code them from memory; read them from the SDK. | Fixed RGB triples. |
| `Index2Color` + `RGBForeColor` | `ColorUtils.c:25`–`26` (`ColorText`, `:20`–`29`) | Looks up an **index** in the current `GDevice`'s colour table and sets the foreground to that RGB. `ColorText(s, 171L)` therefore depends on which CLUT is loaded. | Index into your loaded palette explicitly. |
| `TextFont(applFont)` / `TextSize(9)` / `TextFace(bold)` | `Tools.c:151`–`153` | `applFont` and `bold` are Toolbox constants from `Fonts.h`/`QuickDraw.h` and are **not defined in this source tree**; `applFont` is the "application font" pseudo-ID (Geneva on a US System 7). The only Toolbox-adjacent constant the release does define is `kPreferredDepth` = **8** at `Externs.h:15`. | Ship a bitmap font. Metrics differ from any modern font and change `CollapseStringToWidth` (`StringUtils.c:282`–`296`) truncation. |
| `DrawString` / `StringWidth` | `ColorUtils.c:27`, `StringUtils.c:287`–`292` | Operate on **Pascal strings** (length byte + up to 255 bytes, MacRoman). | Go `string` (UTF-8); re-encode from MacRoman when reading resources. |
| `SetPort` / `SetPortWindowPort` | `ObjectEdit.c:2745`, `Tools.c:195`, `:227`, `:449` | Sets the global current `GrafPort`. All subsequent drawing is implicitly to it. | Pass an explicit target to every draw call. The implicit-global style is the root cause of the leak in §14.1. |

### 18.2 GWorlds (offscreen graphics worlds)

| Item | Where | Notes | Go replacement |
|---|---|---|---|
| `CreateOffScreenGWorld(&map, &rect, kPreferredDepth)` | `Tools.c:83` (tools sheet, 360 × 216), `StructuresInit2.c:158` (`workSrcMap`), `:162` (`backSrcMap`) | `kPreferredDepth` = **8** (8-bit indexed). `workSrcMap` and `backSrcMap` are both `houseRect` zeroed to the origin. | `image.Paletted` (or a `[]uint8` + `color.Palette`). |
| `CreateOffScreenGWorld(..., 1)` | `StructuresInit2.c:68`, `:132` (mask maps) | Depth **1** — masks are 1-bit bitmaps. | `image.Alpha` or a bitset. |
| `GetGWorld` / `SetGWorld` | `ObjectEdit.c:2355`–`2356`, `:2706`; `Tools.c:80`, `:84`, `:87` | Save/restore the *global* current world. The paired-call discipline is manual and is violated at `ObjectEdit.c:2358`–`2359` (§14.1). | Explicit render target parameter + `defer` for restore. |
| `GetGWorldPixMap` / `GetPortBitMapForCopyBits` | every `CopyBits`/`CopyMask` call | Carbon accessors that replaced direct `->portBits` field access; the `(BitMap *)*GetGWorldPixMap(x)` cast is the idiomatic 1990s form. | N/A. |
| `DisposeGWorld` | `Tools.c:99` (`KillToolsOffscreen`) | Must be followed by setting the pointer to `nil` so the lazy-create guard works (§16.4). | GC handles it; keep the nil-out so the re-create guard still reads correctly. |
| `CopyBits(src, dst, &srcR, &dstR, srcCopy, nil)` | `ObjectEdit.c:2726`–`2728`, `Tools.c:176`–`178`, `ObjectDraw.c:992`–`994` | Straight blit; **stretches** if the rects differ in size. Every call here uses equal sizes. | Explicit blit; assert equal sizes rather than silently scaling. |
| `CopyMask(src, mask, dst, &srcR, &maskR, &dstR)` | `ObjectEdit.c:2710`–`2724`, `ObjectDraw.c:965`–`968` | 1-bit mask; where the mask bit is **1** the source is copied. Note the source rect is passed **twice** (as src and mask rect) in the glider-marker calls. | Per-pixel masked blit. |

### 18.3 Resource Manager

| Resource | ID | Used by | Go replacement |
|---|---:|---|---|
| `'PICT'` | **1011** | `Tools.c:85` (`LoadGraphic(kToolsPictID)`) — the 360 × 216 tool sheet | Decode once at build time to PNG; embed with `go:embed`. |
| `'PICT'` | `data.g.height` of a `kCustomPict` | `ObjectEdit.c:2283` and `ObjectRects.c:222` — `GetPicture(...)`, then `(*thePict)->picFrame` **defines the object's rect** (§13.5) | A PICT ID → decoded image map. **The object's geometry depends on an external resource**; see Porting notes. |
| `'PICT'` | 4018 / 5018 | `StructuresInit2.c:66`, `:70` — clutter sheet + mask (128 × 69) | Embed. |
| `'PICT'` | 1019 / 1020 | `StructuresInit2.c:130`, `:134` — angel + mask (96 × 44) | Embed. |
| `'PICT'` | 1999 | `StructuresInit2.c:109` — floor support (512 × 44) | Embed. |
| `'cicn'` | **2000** | `Tools.c:166` — `DrawCIcon`, the selection-arrow palette cell | Embed as a 28 × 28 image. |
| `'PAT#'` | **128** (`kMarqueePatListID`, `Marquee.c:15`) | `Marquee.c:504` — 7 marching-ants patterns, `GetIndPattern(..., i + 1)` for `i` = 0..6 | Embed 7 × 8 bytes. |
| `'STR#'` | **1007** (`"Object Names"`) | `Tools.c:146`–`147` — 144 strings, 1569 bytes, index == `what` (§16.6) | Embed as a `[145]string`. |
| `'STR#'` | 150 | `StringUtils.c:325` (`GetLocalizedString`) | Embed. |
| `'STR#'` | 1006 | yellow alerts | Embed. |
| `'CNTL'` | **129** | `Tools.c` — the mode pop-up in the tools windoid | Native widget. |
| `'WDEF'` | **2048** (`kWindoidWDEF`) | the tools and coordinates windoids | Draw the windoid chrome yourself. |
| `GetPicture` returns `PicHandle`; `HLock` / `HUnlock` around `(*thePict)->picFrame` | `ObjectEdit.c:2289`–`2291` | Handles are **double-indirect and relocatable**; the Memory Manager may move the block on any allocation, so it must be locked before dereferencing. | No analogue; Go pointers are stable. Delete the lock/unlock. |
| `GetIndString` | `Tools.c:146`–`147`, `StringUtils.c:325` | 1-based index into a `'STR#'`; returns an **empty** string for out-of-range, it does not error. | Bounds-check explicitly; the original silently draws nothing. |
| `GetIndPattern`, `GetString`, `GetResource('demo', 128)` | `Marquee.c`, `StringUtils.c:308`, `StructuresInit2.c:290` | | Embed. |

### 18.4 Memory Manager

| Call | Where | Trap for a porter |
|---|---|---|
| `NewPtr(sizeof(Rect) * kNumSrcRects)` | `StructuresInit2.c:271` | **`NewPtr` does not zero memory** (unlike `NewPtrClear`). 144 × 8 = **1152 bytes** of garbage. `srcRects[kFlower]` is never written (§16.6) - and, per §13.7, never read either. Go's `make([]image.Rectangle, 144)` zeroes — a *behaviour change*, and the safer one. |
| `NewPtr(sizeof(roomType))` | `StructuresInit2.c:193` | `thisRoom` is a single 348-byte scratch room (§11.1); all editing happens in it and `CopyThisRoomToRoom` writes it back. |
| `HGetState` / `HLock` / `HUnlock` / `HSetState` | `StringUtils.c:310`–`313`, `ObjectEdit.c:2289`–`2291` | Handle locking around dereference. Delete in Go. |
| `BlockMove(src, dst, n)` | `StructuresInit2.c:295` | `memmove`. Go: `copy`. |

### 18.5 Event and input APIs

| Call | Where | Semantics | Go replacement |
|---|---|---|---|
| `GetKeys(KeyMap)` | `ObjectEdit.c:2754` (`HiliteAllObjects` busy-wait, `do` at `:2752`), `Events.c:487` | Fills a **128-bit** `KeyMap` (4 × `UInt32`) with the *current physical* key state, independent of the event queue. | Poll your platform's keyboard state. |
| `BitTst(&keyMap, n)` | same | Tests **bit offset `n` counting from the MSB of byte 0**, i.e. `(((UInt8 *)map)[n >> 3] >> (7 - (n & 7))) & 1`. Getting the bit order wrong is the classic port bug. | Reimplement exactly, or use a `map[Key]bool`. |
| `kCommandKeyMap` = **48**, `kOptionKeyMap` = **61** | `Externs.h:155`, `:161` | Raw key-map offsets, **not** virtual key codes and **not** `event.modifiers` masks. The whole key-map-offset block is `Externs.h:115`–`163`. Others in it: `kUpArrowKeyMap` **121** (`:129`), `kDownArrowKeyMap` **122** (`:130`), `kRightArrowKeyMap` **123** (`:131`), `kLeftArrowKeyMap` **124** (`:132`) — these four are what the dead arrow-nudge path of §7 would have used; then `kVKeyMap` 14, `kWKeyMap` 10, `kXKeyMap` 0, `kZKeyMap` 1, `kPeriodKeyMap` 40, `kEscKeyMap` 50, `kDeleteKeyMap` 52, `kSpaceBarMap` 54, `kTabKeyMap` 55, `kControlKeyMap` 60, `kCapsLockKeyMap` 62, `kShiftKeyMap` 63, plus the keypad offsets `kPlusKeypadMap` 66, `kMinusKeypadMap` 73, `kTimesKeypadMap` 68, `k0KeypadMap` 85 … `k9KeypadMap` 91. | Map your modifier state to these indices. |
| Raw key codes | `Externs.h:165` onward | `kTabRawKey` **0x30**, `kClearRawKey` **0x47**, `kF5RawKey` **0x60**, `kF6RawKey` **0x61**, `kF7RawKey` **0x62**, `kF3RawKey` **0x63**, `kF8RawKey` **0x64**, `kF9RawKey` **0x65**. Note the comments in the source call these "key map offset for …" — **the comments are wrong**; these are virtual key codes, a distinct namespace from the key-map bit offsets above (`kTabKeyMap` is 55, `kTabRawKey` is 0x30 = 48). | Keep the two namespaces separate; do not mix them. |
| `StillDown()` / `WaitMouseUp()` | `WaitMouseUp` at `Marquee.c:200`, `:231`, `:275`, `:358`; `StillDown` at `ObjectEdit.c:82` and `:115` (`DoSelectionClick`) and `RoomInfo.c:337` — there is **no** `StillDown` in `Marquee.c` | Peek the event queue for a mouse-up **without** dequeuing (`StillDown`) or dequeuing it (`WaitMouseUp`). All drag loops are **synchronous and blocking**. | Event-driven drag state machine; a blocking loop will not survive a modern compositor. |
| `GetMouse(&Point)` | drag loops | Mouse position in the **current port's local** coordinates. | Explicit coordinate-space conversion. |
| `DeltaPoint(a, b)` | `Marquee.c:203`, `:234`, `:278`, `:361` | Returns a **`long`** packing `(v_delta << 16) | h_delta` — i.e. the *same* `v`-first layout as `Point`. | Return two `int`s. |
| `SysBeep(1)` | `ObjectEdit.c:1166` (`DeleteObject` refusal) | The only user feedback for a refused delete. | Any audible/visual cue. |
| `GetTime(&DateTimeRec)` | `ObjectDraw.c:970` (`DrawRedClock`) | Local wall-clock time, no timezone. Makes rendering non-deterministic (§17.3). | Inject a clock interface for testability. |
| `InitCursor` / `SetCursor` | `Marquee.c:193`, `:223`, `:265`, `:267`, `:350`, and `InitCursor` again at `:254`, `:340`, `:402` | Global cursor shape; the editor swaps in **four** named cursors — `handCursor`, `vertCursor`, `horiCursor`, `diagCursor`, declared `extern` at `Marquee.c:28` (§1.7). | Platform cursor API. |

### 18.6 Data layout: big-endian, unpadded, `v`-first

| Item | Detail |
|---|---|
| Byte order | All on-disk house data is **big-endian**. `short` = 2 bytes, `long` = 4. Go: `encoding/binary.BigEndian`. |
| Struct packing | **No padding anywhere.** `objectType` is exactly **12 bytes** (`short what` + 10-byte union), `roomType` exactly **348** (§11.1). Any Go struct-based reader must use explicit offsets, not `unsafe.Sizeof`. |
| `Point` | `struct { short v; short h; }` — **`v` (vertical) comes first**. A Go port using `image.Point{X, Y}` must swap on every read and write. This is the single most common porting error in this subsystem, because `data.a.topLeft` etc. are read and written by all six gestures. |
| `Rect` | `struct { short top, left, bottom, right; }` — **top, left, bottom, right**, so also `v` before `h`. Go's `image.Rectangle` is `Min{X,Y}, Max{X,Y}` — opposite order in both dimensions. |
| Rect half-openness | `right`/`bottom` are exclusive, matching `image.Rectangle`. Width = `right - left`. |
| Colour | 8-bit **indexed**; all pixel values are CLUT indices. Blend modes (`srcOr`, `patXor`) operate on indices, not colours (§18.1). |
| Pascal strings | Length byte first, MacRoman encoding, max 255 bytes. Room `name` is a fixed **27**-byte field (§12.11) — length byte + 26. |
| `Byte` | `unsigned char`. Several packed fields use it (`kLiftArea`'s `tall × 2`, `kDeluxeTrans`'s `tall` nibble pair) — see §6.2, §6.3, §12.5. |
| 68k/PPC | The sources compile for both (`Glider PRO.mcp`, `Glider PRO CW5.mcp`). No inline assembly or endianness-conditional code in this subsystem; both targets are big-endian, so nothing here distinguishes them. |

---

## 19. Corrections to `docs/analysis/editor.md`

Three claims in `editor.md` §6–§11 are contradicted by the source. They are listed here rather than
edited into that file, per the constraint that other docs be appended to only.

### 19.1 §6.1 — empty `roomObjectRects` entries are not `(0, 0, 0, 0)`

`editor.md` §6.1 states that slots without an object hold a zero rect. They hold **two different
non-zero sentinels**, and the difference is observable:

| Sentinel | Value | Written by | Meaning |
|---|---|---|---|
| "empty or unrecognised slot" | **(−2, −2, −1, −1)** | `GetThisRoomsObjRects`, `ObjectEdit.c:2080` (the `case kObjectIsEmpty:` arm) and `:2335` (the `default:` arm) | 1 × 1 rect at (−2, −2) |
| "object was just deleted" | **(−1, −1, 0, 0)** | `DeleteObject`, `ObjectEdit.c:1182` | 1 × 1 rect at (−1, −1) |

Both are 1 × 1, not empty, and both are **outside** the room rect `(0, 0, 512, 322)`, which is what
makes `FindObjectSelected`'s `PtInRect` test (`ObjectEdit.c:61`) safely miss them — and, since
that loop covers **all 24 slots with no `what` test** (§3.1), the sentinels are the *only* thing
preventing an empty slot from being selectable. A port that
substitutes `(0,0,0,0)` still works for hit testing (an empty rect contains no point) but
**breaks `HiliteAllObjects`** (§14.2), which calls `FrameRect` on all 24 entries: `(0,0,0,0)` frames
nothing, while `(-1,-1,0,0)` frames a pixel at (−1,−1) that the clip region discards. More
importantly, the two sentinels are distinguishable in a debugger and the original relies on
neither — so a port should pick one and document it, not assume zero.

### 19.2 §11.1 — `DuplicateObject` does not copy the object wholesale

`editor.md` §11.1 gives pseudocode that finds an empty slot and `memcpy`s the source object into it.
The actual implementation (`ObjectEdit.c:1191`–`1370`, §8) **delegates to `AddNewObject`**:

```
DuplicateObject():
    tempObject = thisRoom->objects[objActive]           /* :1198 (local copy) */
    placePt = centre of roomObjectRects[objActive], then + (64, 0)
    if (AddNewObject(placePt, tempObject.what, false))  /* :1207 */
        ... 16-arm switch copying SELECTED fields per union arm ...
```

Four consequences the existing pseudocode misses:

1. **Per-type population caps are enforced.** `AddNewObject` consults the **twelve**
   `HowMany*Objects` counters and refuses with `ShoutNoMoreSpecialObjects` (`ObjectAdd.c:1075`). A
   `memcpy` implementation would let the user exceed the engine's fixed arrays.

   | Counter | Line | Counts |
   |---|---:|---|
   | `HowManyCandleObjects` | `ObjectAdd.c:888` | `kTaper`, `kCandle`, `kStubby` (`:894`–`896`) |
   | `HowManyTikiObjects` | `:904` | `kTiki` |
   | `HowManyBBQObjects` | `:918` | `kBBQ` |
   | `HowManyCuckooObjects` | `:932` | `kCuckoo` |
   | `HowManyBandsObjects` | `:946` | `kBands` |
   | `HowManyGreaseObjects` | `:960` | `kGreaseRt`, `kGreaseLf` (`:966`–`967`) |
   | `HowManyStarsObjects` | `:975` | `kStar` |
   | `HowManySoundObjects` | `:989` | `kSoundTrigger` |
   | `HowManyUpStairsObjects` | `:1003` | `kUpStairs` |
   | `HowManyDownStairsObjects` | `:1017` | `kDownStairs` |
   | `HowManyShredderObjects` | `:1031` | `kShredder` |
   | `HowManyDynamicObjects` | `:1045` | the 17 animated types listed at `:1051`–`1067`: `kSparkle`, `kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`, `kMicrowave`, `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish` |

   Each counter is a plain `for (i = 0; i < kMaxRoomObs; i++)` tally over the **current room only**,
   so the caps are per-room, not per-house. They exist because the engine preallocates fixed arrays
   for the corresponding runtime state (`flames`, `tikiFlames`, `bbqCoals`, `bands`, `grease`,
   `theStars`, `shreds`, `dinahs`, … — `StructuresInit2.c:212`–`263`).
2. **`what` may be *changed* on the way in.** For the eight door/window types `AddNewObject`
   re-derives the Lf/Rt variant from `placePt.h > 256` (`ObjectAdd.c:374`–`437`, §16.7). Duplicating a
   `kDoorInLf` whose duplicate lands past x = 256 produces a `kDoorInRt`.
3. **`AddNewObject` sets `objActive` to the new slot** and applies the type's default geometry before
   the field copy runs, so the copy only needs to restore the *data* fields — which is why the
   switch copies selected fields rather than the whole union.
4. **Not everything is copied.** Fields set by `AddNewObject`'s defaults and *not* overwritten by the
   16-arm switch retain the default, not the source value. See §8.3 for the per-arm list and §8.4 for
   what is dropped.

### 19.3 §11.2 — `DeleteObject`'s step order, guards, and cache update

`editor.md` §11.2's step list omits three things and gets the order wrong. The actual sequence
(`ObjectEdit.c:1154`–`1187`, §9):

| # | Step | Line |
|---:|---|---|
| 1 | `if ((theMode != kEditMode) \|\| (objActive == kNoObjectSelected)) return` — **there is a mode guard** | `:1159`–`1160` |
| 2 | `if ((objActive == kInitialGliderSelected) \|\| (objActive == kLeftGliderSelected) \|\| (objActive == kRightGliderSelected)) { SysBeep(1); return; }` — three explicit equality tests, not a `< 0` range test; **the refusal beeps**, so the glider pseudo-selections cannot be deleted | `:1162`–`1168` |
| 3 | Clear the 24 `retroLinkList` entries whose `.room == thisRoomNumber && .object == objActive`, by setting `.room = -1` | `:1170`–`1175` |
| 4 | `thisRoom->objects[objActive].what = kObjectIsEmpty` | `:1177` |
| 5 | `thisRoom->numObjects--` — **without compacting the slot array** | `:1178` |
| 6 | `fileDirty = true` | `:1179` |
| 7 | `UpdateMenus(false)` | `:1180` |
| 8 | `InvalWindowRect(mainWindow, &mainWindowRect)` — the **whole** window | `:1181` |
| 9 | `QSetRect(&roomObjectRects[objActive], -1, -1, 0, 0)` — the cache is patched **in place**; `GetThisRoomsObjRects()` is **not** called | `:1182` |
| 10 | `DeselectObject()` — which is what sets `objActive = kNoObjectSelected` (`:1782`) *and* calls `StopMarquee()` (`:1783`) and `UpdateMenus(false)` again (`:1784`) | `:1183` |
| 11 | `ReadyBackground(thisRoom->background, thisRoom->tiles)` | `:1184` |
| 12 | `DrawThisRoomsObjects()` | `:1185` |

Three details of the order are easy to get wrong: `fileDirty` and the first `UpdateMenus` come
*before* the cache patch, not after; `DeselectObject()` — not a bare assignment — is the
deselect, so `StopMarquee()` happens **at step 10**, not at the top of the function
(`DeleteObject` never calls `StopMarquee` directly); and `UpdateMenus(false)` runs **twice**,
once at `:1180` and again inside `DeselectObject`.

The three omissions matter:

* The **`SysBeep(1)` refusal** (step 2) is the only feedback path; a port that silently ignores the
  gesture loses it.
* **`numObjects` is decremented but the array is not compacted** (step 5), so `numObjects` is *not*
  an index bound — slot *k* can be empty while slot *k+1* is occupied. Every loop over objects in the
  program iterates `0 .. kMaxRoomObs-1`; the ones that read object data test
  `what != kObjectIsEmpty`, and `FindObjectSelected` relies on the cache sentinels instead
  (`ObjectEdit.c:59`, §3.1). **None** iterates `0 .. numObjects`. A port must preserve this, because
  slot index doubles as z-order and as the link address stored in other objects' `who` fields (§9.1).
* **The cache is patched, not rebuilt** (step 9). This is why the deleted slot's sentinel is
  `(-1,-1,0,0)` and not `(-2,-2,-1,-1)` (§19.1) — the two code paths that write the cache disagree.

---

## Open questions

1. **Is the `HiliteAllObjects` busy-wait reachable without a hang on a real machine?**
   `ObjectEdit.c:2752`–`2757` spins on `GetKeys` with no `WaitNextEvent`. On cooperative-multitasking
   System 7 this starves every other process for as long as the modifiers are held. Whether the
   author considered this acceptable, or whether the gesture was meant to be momentary, cannot be
   determined from the source. The behaviour to port is unambiguous (show the overlay while held);
   only the intent is unclear.
2. **What are `'PICT'` resources 10000–10119, 11003–11031, and 12345–12351?** §13.5 establishes
   empirically that `kCustomPict`'s `data.g.height` holds a `'PICT'` resource ID (4782 instances,
   **149** distinct IDs, in three clusters). The IDs are not present in `Glider PRO.r` — they live in
   the *house files'* resource forks, which the `.binhex` data-fork extraction used for this document
   does not decode. The three clusters strongly suggest three authoring conventions, but which is
   which cannot be determined without decoding a house's resource fork.
3. **Is `'STR#'` 1007 index 144 (`Mermaid`) referenced by any resource-level data?** No `#define`,
   `case` label, or `srcRects` slot exists for `what` = 0x90 (§16.6). Whether an unreleased build
   used it, or whether it is purely aspirational, is not answerable from the shipped source.
4. ~~**Was the `ToggleToolsWindow` `isToolsOpen` bug (§16.8) ever user-visible?**~~ **Resolved: yes.**
   The close branch's `isToolsOpen = true` at `Tools.c:362` should be `false`, and the flag is read at
   `Menu.c:800` inside the `kEditMode && houseUnlocked` arm, which calls `OpenToolsWindow()` whenever
   it is true. So closing the tools windoid and re-entering edit mode re-opens it, and the stale
   `true` is persisted to prefs at `Main.c:264` and restored at `Main.c:109`, making the fault survive
   relaunch. `ToggleToolsWindow` itself is unaffected because it dispatches on `toolsWindow == nil`
   (`Tools.c:354`), not on the flag. See §16.8.
5. **Why does `DragObject` write `rightStartGliderDest`'s x as 0 at `ObjectEdit.c:558` when the two
   other sites use `kRoomWide - 48`?** (`:1517` in `MoveObject` and `:2066` in
   `GetThisRoomsObjRects` both write the expression `kRoomWide - 48`, = 464; no site contains a
   literal 464 — §5.2.) It is almost certainly a typo, but no comment or changelog confirms it, and the
   resulting marker position after a drag differs from the position after a redraw.
6. **Does any shipped house exercise the light-arm step-2 clamp?** §12.8 proves the block at
   `HouseLegal.c:439`–`464` is unreachable *through `KeepObjectLegal`*, because `ForceRectInRect`
   always returns a rect satisfying the guard's negation, and 196 shipped tubes confirm zero
   overhangs. Whether some other caller can reach it was not exhaustively traced.
7. **What is the intended semantics of `theMarquee.dist` after a resize?** `DragHandle` never updates
   it (§6.5), so a handled object's leader-line length goes stale until the next
   `StartMarqueeHandled`. Whether the visual result was noticed or tolerated is unknown.

---

## Porting notes

Ordered by how likely each is to produce a silently-wrong port.

1. **`Point` is `{v, h}` and `Rect` is `{top, left, bottom, right}`.** Both put the vertical
   coordinate first — the opposite of `image.Point{X, Y}` and `image.Rectangle{Min, Max}`. Every one
   of the six gestures reads and writes `topLeft` or `bounds`. Swap at the serialisation boundary,
   once, and never again — do not carry a `v`-first type into the Go code.
2. **`what` is a packed nibble pair, and the tool palette depends on it.**
   `what = (toolMode - 1) * 16 + toolSelected`; `'STR#'` 1007's index *is* the `what` code (§16.6).
   The two `toolIcon` adjustments (§16.7) are what make `kLiftArea` (0x10) and `kDeluxeTrans` (0x40)
   placeable at all, and 5 of the 117 types are **not** reachable from the palette. A naive palette
   loop makes **2** types unplaceable (`kLiftArea`, `kDeluxeTrans`) and wrongly offers **5** extra
   cells with duplicated names (`kRightFan`, `kDoorInRt`, `kDoorExLf`, `kWindowInRt`, `kWindowExLf`).
3. **`KeepObjectLegal` applies containment *before* parity, and that ordering is a bug you must
   decide whether to reproduce.** §12.10: `ForceRectInRect` can push an odd-x-forced object to an
   even coordinate, and the subsequent parity step can then push it back outside the room. There is
   **one instance permanently baked into a shipped house** — `Titanic` room 161 object 9, raw
   `0065002cffff000000000101`, a `kTV` at `topLeft.h` = **−1**. If you fix the ordering, that object
   moves; if you keep it, you inherit the defect. Either way, document the choice.
4. **`numObjects` is not an index bound.** `DeleteObject` decrements it without compacting (§9.1,
   §19.3). Slot index is simultaneously the z-order and the link address other objects store in their
   `who` byte. Always iterate `0 .. 23` and test `what != kObjectIsEmpty`.
5. **Selection is not part of rendering.** No draw function reads `objActive` (§14.1), and the
   marquee is an XOR overlay (§4.1). This is why a redraw cannot lose the selection. Keep the same
   separation: a selection overlay layer, never baked into the room composite.
6. **Exactly two room-change paths omit `DeselectObject()`** — `MainWindow.c:207`–`209` and
   `Room.c:666` (§3.4). Both leave `objActive` pointing at a slot in the *new* room. Copy
   `GoToObjectInRoom` (§14.3) as the canonical pattern instead: deselect, load, reflect, then set
   `objActive` and re-arm from the rebuilt cache.
7. **The arrow-key nudge (`MoveObject`, 399 lines) is dead code in the shipped build.**
   `BUILD_ARCADE_VERSION` = **1** (`GliderDefines.h:16`) routes arrow keys to the arcade arm at
   `Events.c:191`–`207`, never reaching the editor arm at `:209`–`251` (§7.1). Port it if you want
   the feature, but know that its 109 case labels are **8 short** of the 117 the drag path handles
   (all eight door/window types are missing), and that it lacks an `objActive` guard, so an
   unguarded port corrupts the room header (§7.2, §12.11).
8. **`kCustomPict`'s geometry comes from an external resource, and failure mutates the object.**
   `GetPicture(data.g.height)` → `(*thePict)->picFrame` defines the rect; on `nil` the code writes
   **`data.g.height = 10000`** into the house and falls back to a 72 × 34 rect
   (`ObjectEdit.c:2283`–`2296`, §13.5). A port must decide what "picture missing" means *before* it
   loads a house, because the original silently edits the data.
9. **There is no undo (§15).** Every gesture mutates in place and sets a one-way `fileDirty`. A
   348-byte room snapshot per gesture is nearly free and makes items 3, 4 and 8 recoverable — but the
   snapshot must also cover `(*thisHouse)->initial`, `thisRoom->leftStart`/`rightStart`,
   `numObjects`, and `retroLinkList` entries **in other rooms**, or you restore an inconsistent house
   (§11.3, §9.1).
10. **`PtInRect` is half-open and the resize handle is `(-4, -4, 5, 5)`.** The `-4` comes from
    integer truncation of `kHandleSideLong / -2` (§4.2); a port that computes `-(kHandleSideLong/2)`
    or rounds differently gets a 1 px offset hit box on two sides, which users feel immediately.
11. **`NewPtr` does not zero.** `srcRects` is 1152 uninitialised bytes and `srcRects[kFlower]` is
    never written (§16.6). §13.7 proves no code path reads it, so zero-initialising in Go is safe as
    well as tidier - but keep flowers on the six-entry `flowerSrc` table keyed on `data.i.pict`, not
    on the 144-entry `srcRects` table keyed on `what`, or you will reintroduce the dependency.
12. **All drag loops are synchronous and blocking** (`StillDown`/`GetMouse`, §18.5), and
    `HiliteAllObjects` blocks on `GetKeys` with no event pump (§14.2). Both must become event-driven
    state machines; a straight transliteration freezes a modern compositor.
13. **Restore the render target with `defer`.** `DrawThisRoomsObjects` sets the GWorld at
    `ObjectEdit.c:2356` and returns at `:2359` without restoring it (§14.1) — a latent bug that only
    the upstream `houseUnlocked` guards keep invisible.
14. **Blend modes act on palette indices, not colours.** The dark-room dimming (`srcOr` gray,
    `ObjectEdit.c:2364`–`2368`) and the whole selection/highlight system (`patXor`) are index
    arithmetic on an 8-bit CLUT (§18.1). Alpha compositing produces visibly different pixels.
15. **The two entry markers come from `blowerSrcMap`, not the glider sheet** —
    `leftStartGliderSrc` = (0,0,48,16) at sheet offset (0, **358**), `rightStartGliderSrc` =
    (0,0,48,16) at (0, **374**) (`StructuresInit.c:280`–`284`, §14.1).
16. **`DrawRedClock` reads the wall clock**, so room rendering is not a pure function of house state
    (§17.3). Inject a clock. The digit layout is fixed-cell — 4 px within a group, **6 px** between
    hours and minutes, first cell at `(left + 5, top + 7)`, and a single-digit hour leaves the tens
    cell **blank without shifting** (§17.2).
17. **Ten of `KeepObjectLegal`'s clamp blocks mutate the object without clearing `unchanged`**
    (nine tabulated in §12.1, plus `HouseLegal.c:463` in §12.8), so the function returns `true`
    ("nothing changed") after changing something. Callers use
    the return value to decide whether to redraw. Reproduce the return value only if you also
    reproduce the missed redraws; otherwise fix it and accept the divergence.
18. **`kLiftArea` stores height as `tall × 2` in a `Byte`** (§6.2, §12.5), and `kDeluxeTrans` packs
    width and height as two `Byte` nibbles of a `short` scaled by 4 (§6.3). Both are re-derived by
    `KeepObjectLegal`, so a port that stores them unpacked must re-pack before calling it.






