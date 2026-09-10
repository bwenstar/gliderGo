# Glider PRO 1.0.4 — Data Structures: In-Memory and On-Disk

## Scope

This document is a complete, byte-level reference for every data structure in
Glider PRO 1.0.4 (John Calhoun / Casady & Greene, GPLv2 source release), covering:

* Every `typedef struct` in `GliderPRO/Headers/GliderStructs.h` (33 of them) —
  field types, sizes, cumulative offsets, semantics, and observed value ranges.
* Every struct declared outside that header that a port must reproduce:
  `prefsInfo` (Externs.h), `macEnviron` (Environ.h), `marquee` (Marquee.h),
  `sizeType` (Environ.c), `pageType` (GameOver.c), `acurRec` (AnimCursor.c),
  `phoneType` (Play.c), `trigType` (Triggers.c).
* The four on-disk file formats (house, prefs, high scores, saved game) and the
  four on-disk resource formats a port must parse (`'bnds'`, `'demo'`, `'acur'`,
  `'PAT#'`).
* Every place where a C struct layout is *assumed* to equal a file layout
  (blind `FSRead`/`FSWrite`/`BlockMove` of a whole struct). These are the
  byte-exactness constraints for the Go port and are listed in one table in
  [§10](#10-blind-io-couplings-the-byte-exactness-constraints).
* Recommended idiomatic Go type definitions, side by side with the C.

Everything in this document that concerns binary layout was verified two ways:
(a) by compiling the struct declarations with `gcc` under both classic-Mac
alignment models and printing `sizeof`/`offsetof`, and (b) by decoding all 22
shipped BinHex'd house files and the application's resource fork with `python3`
and reading the observed bytes. Observed values are reported inline. Where a
claim could **not** be verified against real bytes, it is marked
"**unverified**" and repeated in [§13 Open questions](#13-open-questions).

What this document does *not* cover: gameplay physics, rendering, sound, the
editor UI, and the Room/House *behaviour* algorithms beyond what is needed to
explain a field's meaning. Those belong to sibling documents.

## Sources read

All `.c`/`.h` files under `GliderPRO/` use classic-Mac CR-only line endings.
Every file below was converted with `tr '\r' '\n'` into `/tmp/wf-structs/`
before reading; **all line numbers cited in this document are line numbers in
the CR→LF converted copy**, which is what any modern tool (including `git`
after a `.gitattributes` normalisation) will report.

Headers, read in full:

| File | Lines | What it contributes |
|---|---:|---|
| `GliderPRO/Headers/GliderStructs.h` | 347 | 33 `typedef struct`s; the entire object/room/house model |
| `GliderPRO/Headers/Externs.h` | 394 | `prefsInfo` + the `align=mac68k` pragma |
| `GliderPRO/Headers/GliderDefines.h` | 625 | every constant: object codes, limits, versions |
| `GliderPRO/Headers/GliderVars.h` | 59 | global declarations (which globals hold which struct) |
| `GliderPRO/Headers/Environ.h` | 37 | `macEnviron` |
| `GliderPRO/Headers/Marquee.h` | 24 | `marquee` |
| `GliderPRO/Headers/House.h` | 12 | no structs: `Str32 thisHouseName`, `Boolean houseUnlocked` |
| `GliderPRO/Headers/Room.h` | 12 | no structs: `GWorldPtr backSrcMap` |
| `GliderPRO/Headers/Player.h` | 12 | no structs: `shadowSrcMap`, `shadowMaskMap` |
| `GliderPRO/Headers/Map.h` | 12 | no structs: `nailSrcMap`, `mapWindow` |
| `GliderPRO/Headers/Scoreboard.h` | 15 | no structs: 5 `GWorldPtr` scoreboard buffers |
| `GliderPRO/Headers/Objects.h` | 42 | no structs: 35 `extern GWorldPtr` src/mask declarations |

Sources, read in full or in the cited regions:

`HouseIO.c`, `Prefs.c`, `HighScores.c`, `SavedGames.c`, `HouseLegal.c`,
`House.c`, `Link.c`, `Room.c`, `RoomInfo.c`, `ObjectAdd.c`, `ObjectInfo.c`,
`ObjectEdit.c`, `ObjectRects.c`, `Objects.c`, `Interactions.c`, `Triggers.c`,
`Transit.c`, `Play.c`, `Player.c`, `Input.c`, `Main.c`, `Menu.c`,
`StructuresInit2.c`, `Environ.c`, `AnimCursor.c`, `GameOver.c`, `Marquee.c`,
`RoomGraphics.c`, `SelectHouse.c`, `Settings.c`.

Binary data, decoded and parsed:

* `GliderPRO/Houses/*.binhex` — all 22 shipped houses (BinHex 4.0; data fork +
  resource fork extracted).
* `GliderPRO/Glider PRO.r` — the 15 475 666-byte DeRez text dump of the application's
  resource fork (199 843 LF-terminated lines), from which `'demo' 128`,
  `'acur' 128`, `'PAT#' 128`, `'vers'` and `'ozm5'` were re-assembled to bytes.

---

## Table of contents

1. [Primitive types and conventions](#1-primitive-types-and-conventions)
2. [Struct alignment: `mac68k` vs PowerPC natural](#2-struct-alignment-mac68k-vs-powerpc-natural)
3. [The object model: 9 payload variants + `objectType`](#3-the-object-model-9-payload-variants--objecttype)
4. [`roomType`](#4-roomtype)
5. [`houseType`](#5-housetype)
6. [High scores and saved games: `scoresType`, `gameType`, `savedRoom`, `game2Type`](#6-high-scores-and-saved-games)
7. [`prefsInfo`](#7-prefsinfo)
8. [Runtime-only structures](#8-runtime-only-structures)
9. [On-disk formats](#9-on-disk-formats)
10. [Blind I/O couplings: the byte-exactness constraints](#10-blind-io-couplings-the-byte-exactness-constraints)
11. [Empirical evidence](#11-empirical-evidence)
12. [Recommended Go type definitions](#12-recommended-go-type-definitions)
13. [Open questions](#13-open-questions)
14. [Porting notes](#14-porting-notes)

---

## 1. Primitive types and conventions

### 1.1 Scalar types

| C type | Bytes | Signed? | Notes |
|---|---:|---|---|
| `Boolean` | 1 | n/a | Mac Toolbox `unsigned char`. **Not** restricted to 0/1 on disk — see §11.6. |
| `Byte` | 1 | unsigned | Mac Toolbox `unsigned char`. |
| `char` | 1 | signed | Only used in `demoType`. |
| `short` | 2 | signed | Big-endian on disk. |
| `long` | 4 | signed | 32-bit even on PowerPC (classic Mac ABI). |
| `unsigned long` | 4 | unsigned | Only `scoresType.timeStamps`. |
| `GWorldPtr`, `Handle`, `Ptr`, `WindowPtr` | 4 | n/a | Runtime pointers, 4 bytes on 68k **and** classic PPC. Never persisted (except by accident — see §11.6). |

**Every multi-byte integer on disk is big-endian.** This is a hard requirement:
the on-disk formats are raw memory images of 68k/PPC structs.

### 1.2 QuickDraw aggregates

```c
struct Point { short v; short h; };                      /* 4 bytes — v FIRST */
struct Rect  { short top, left, bottom, right; };        /* 8 bytes */
typedef unsigned char Pattern[8];                        /* 8 bytes */
```

`Point` puts the **vertical** coordinate first. This is the single most common
source of porting bugs, because Glider PRO writes `topLeft.h` before
`topLeft.v` in source order all over the codebase while the bytes are `v` then
`h`. Verified against real data: `Demo House` room 0 object 0 is a
`kFloorVent` whose payload begins `01 31 00 ab` → `topLeft.v = 0x0131 = 305`,
`topLeft.h = 0x00AB = 171`. 305 is exactly `kFloorVentTop`
(`GliderPRO/Headers/GliderDefines.h:467`), which `KeepObjectLegal` forces onto
every floor vent (`GliderPRO/Sources/HouseLegal.c:115-118`), so the `v`-first
ordering is proven, not assumed.

`Rect` is `{top, left, bottom, right}` — also not the intuitive order.
Verified: `Demo House` room 0 object 1 is `kDresser` (0x18) with payload
`00 9a 00 d9 01 25 01 55 00 00` → `bounds = {top=154, left=217, bottom=293,
right=341}`. `bottom == 293 == kDresserBottom`
(`GliderPRO/Headers/GliderDefines.h:476`). Unlike the vent case this is *not* a
`KeepObjectLegal` invariant — `kDresserBottom` appears exactly once in the whole
program, when the editor first inserts a dresser
(`GliderPRO/Sources/ObjectAdd.c:188`, `newRect.bottom = kDresserBottom;`), and
the user may then drag the dresser vertically. It still pins down the field
order here, because the shipped `Demo House` dresser is unmoved. Proven.

### 1.3 Pascal strings

Mac `StrN` types are `unsigned char[N+1]`: a length byte followed by an `N`-byte
buffer. Glider PRO uses five of them.

| Type | Declared size | Max chars | Where used |
|---|---:|---:|---|
| `Str15` | 16 | 15 | `scoresType.names[]`, four `prefsInfo` key names, `prefsInfo.wasHighName` |
| `Str27` | 28 | 27 | `roomType.name` |
| `Str31` | 32 | 31 | `scoresType.banner`, `prefsInfo.wasHighBanner` |
| `Str32` | **33** | 32 | `prefsInfo.wasDefaultName`, global `Str32 thisHouseName` |
| `Str63` | 64 | 63 | `FSSpec.name` (inside `game2Type`) |
| `Str255` | 256 | 255 | `houseType.banner`, `houseType.trailer` |

`Str255 = 256` and `Str31 = 32` and `Str27 = 28` are **verified from real
house bytes** (§11.4): in `houseType` the `banner` field starts at offset 16 and
`trailer` starts at 272, a delta of exactly 256; in `scoresType` `banner`
starts at 0 and `names[0]` at 32; in `roomType` `name` starts at 0 and `bounds`
at 28.

`Str32 = 33` (an odd size!) and `Str63 = 64` come from Apple's `MacTypes.h`
convention `StrN[N+1]` and could **not** be verified against shipped bytes
because no prefs file and no saved-game file ships with the source. `Str32`'s
oddness is load-bearing: it makes `prefsInfo.wasLeftName` start at offset 33
and shifts every subsequent field. See §13.

**The bytes after the length byte are garbage, not zeros.** Verified: in
`Demo House` room 0 the 28-byte `name` field reads

```
09 41 69 72 20 56 65 6e 74 73 | 6d 6f 6f 6d 55 70 00 00 00 00 07 e0 1f f8 3f fc 7f fe
^len 'A  i  r     V  e  n  t  s'  'm  o  o  m  U  p'                (CURS bitmap bytes)
```

i.e. `"Air Vents"` followed by the tail of a previous longer name (`"moomUp"`)
and then eight bytes that are recognisably a 16×16 cursor bitmap row pair
(`07 e0 1f f8 3f fc 7f fe`) left over in the heap block the house handle was
allocated from. A Go port must (a) never read past the length byte and (b)
preserve these bytes verbatim if it wants byte-identical round-trips.

### 1.4 `FSSpec`

```c
struct FSSpec { short vRefNum; long parID; Str63 name; };   /* 2 + 4 + 64 = 70 */
```

70 bytes under `align=mac68k` (`parID` at offset 2). Under PowerPC natural
alignment `parID` would move to offset 4 and `FSSpec` would become 72, which is
why Apple declares `FSSpec` inside its own `align=mac68k` pragma. `FSSpec`
appears in persisted data only inside `game2Type`, which is dead code
([§6.4](#64-game2type-dead-code)), and in the runtime array
`theHousesSpecs` (`GliderPRO/Sources/StructuresInit2.c:276-279`).

### 1.5 Terminology used in the field tables

* **Off** — cumulative byte offset from the start of the struct under
  `align=mac68k` (2-byte alignment), which is what the shipping build used.
* **Off (nat)** — the same under PowerPC natural alignment; shown only where it
  differs.
* **Disk** — `yes` if the field reaches a file byte-for-byte, `no` if
  runtime-only, `stale` if it occupies file bytes but the value is meaningless.

---

## 2. Struct alignment: `mac68k` vs PowerPC natural

### 2.1 What the source actually does

`GliderPRO/Headers/Externs.h` wraps **only** `prefsInfo` in the pragma:

```c
#pragma options align=mac68k          /* Externs.h:231 */
typedef struct { ... } prefsInfo;     /* Externs.h:233-267 */
#pragma options align=reset           /* Externs.h:269 */
```

`GliderStructs.h` is `#include`d at `GliderPRO/Headers/Externs.h:392` — i.e.
**outside** any pragma — so every house/room/object struct is laid out with the
compiler's *default* alignment. On CodeWarrior for 68k the default is 2-byte
(`mac68k`); on CodeWarrior for PowerPC the default project setting is normally
`mac68k` too (the "PowerPC" alignment option exists but breaks Toolbox
compatibility), but it *can* be switched.

This matters, and the shipped data proves both settings were used in practice —
see §2.3 and §11.1.

### 2.2 Verified sizes under both models

Reproduced with `gcc` (`/tmp/wf-structs/layout.c` with `#pragma pack(push,2)`
for `mac68k`; `/tmp/wf-structs/layout_nat.c` with natural alignment but
`Point`/`Rect`/`FSSpec`/`Pattern` still 2-packed as Apple declares them, and
4-byte Mac pointers). `sizeof(houseType)`/`sizeof(game2Type)` are measured on
a one-element variant of the flexible array, then reported as header size.

| Type | mac68k | natural | Differ? | Author's comment |
|---|---:|---:|:--:|---|
| `blowerType` | 10 | 10 | | `// total = 10` |
| `furnitureType` | 10 | 10 | | `// total = 10` |
| `bonusType` | 10 | 10 | | `// total = 10` |
| `transportType` | 10 | 10 | | `// total = 10` |
| `switchType` | 10 | 10 | | `// total = 10` |
| `lightType` | 10 | 10 | | `// total = 10` |
| `applianceType` | 10 | 10 | | `// total = 10` |
| `enemyType` | 10 | 10 | | `// total = 10` |
| `clutterType` | 10 | 10 | | `// total = 10` |
| `objectType` | 12 | 12 | | `// total = 12` |
| `scoresType` | 292 | 292 | | `// total = 292` |
| `gameType` | 40 | 40 | | `// total = 40` |
| `savedRoom` | 292 | 292 | | `// total = 292` |
| `roomType` | 348 | 348 | | `// total = 348` |
| `houseType` (header) | **866** | **868** | **yes** | `// total = 866 +` |
| `game2Type` (header) | **110** | **112** | **yes** | `// total = 114` (author counted `savedData[]` as 4) |
| `prefsInfo` | **226** | **228** | **yes** | — |
| `gliderType` | **110** | **112** | **yes** | — |
| `hotObject` | 16 | 16 | | — |
| `savedType` | 16 | 16 | | — |
| `sparkleType` | 10 | 10 | | — |
| `flyingPtType` | 28 | 28 | | — |
| `flameType` | 20 | 20 | | — |
| `pendulumType` | 26 | 26 | | — |
| `boundsType` | 4 | 4 | | — |
| `bandType` | 16 | 16 | | — |
| `linksType` | 8 | 8 | | — |
| `greaseType` | 26 | 26 | | — |
| `starType` | 24 | 24 | | — |
| `shredType` | 10 | 10 | | — |
| `dynaType` | 36 | 36 | | — |
| `objDataType` | 26 | 26 | | — |
| `demoType` | **6** | **8** | **yes** | — |
| `retroLink` | 4 | 4 | | — |
| `phoneType` | 6 | 6 | | — |
| `trigType` | 12 | 12 | | — |
| `marquee` | 82 | 82 | | — |
| `macEnviron` | 44 | 44 | | — |
| `sizeType` | **10** | **12** | **yes** | — |
| `pageType` | 22 | 22 | | — |
| `acurRec` (1 frame) | 8 | 8 | | — |

**Every one of the author's own `// total = N` comments is correct under
`mac68k`.** That, plus the `demoType` result below, is strong evidence the
shipping build used 2-byte alignment.

### 2.3 The four differences that matter

**(a) `demoType` (6 vs 8).** The `'demo' 128` resource is exactly 6702 bytes
(`kDemoLength`, `GliderPRO/Headers/GliderDefines.h:625`) and is `BlockMove`d
verbatim into a `demoType[]` array
(`GliderPRO/Sources/StructuresInit2.c:280-298`). 6702 / 6 = **1117 exactly**;
6702 / 8 = 837.75. Parsing the resource as 1117 6-byte records yields
monotonically non-decreasing frame numbers 46…3414 and key values
`{0: 910, 1: 198, 3: 9}` — exactly the four cases `Input.c` switches on
(§9.6). So the shipping build had `sizeof(demoType) == 6`, i.e. **mac68k
alignment**.

**(b) `prefsInfo` (226 vs 228).** `Str31 wasHighBanner` occupies offsets 113-144,
i.e. its last byte is at the odd offset 144, so `long wasLeftMap` lands at
**146** under `mac68k` (one pad byte at 145) and **148** under natural alignment
(three pad bytes at 145-147). See the field table in §7. The pragma at
`Externs.h:231` exists precisely to pin this. Without it a 68k-written prefs file
would be misread by a PPC build (and vice versa) —
the `prefVersion` check at `GliderPRO/Sources/Prefs.c:261` would fire and the
file would be deleted.

**(c) `houseType` (866 vs 868).** This one is subtle and it is the single most
important porting fact in this document, so it gets its own subsection.

**(d) `game2Type` (110 vs 112) / `gliderType` (110 vs 112) / `sizeType`
(10 vs 12).** `gliderType` and `sizeType` are runtime-only, so only the total
memory budget in `GliderPRO/Sources/Environ.c:562-698` is affected.
`game2Type` is dead code.

### 2.4 The 866 / 868 puzzle, resolved

`offsetof(houseType, rooms)` is **866 under both alignment models**, and every
persisted `houseType` field sits at the same offset in both:

| Field | Off (mac68k) | Off (natural) |
|---|---:|---:|
| `version` | 0 | 0 |
| `unusedShort` | 2 | 2 |
| `timeStamp` | 4 | 4 |
| `flags` | 8 | 8 |
| `initial` | 12 | 12 |
| `banner` | 16 | 16 |
| `trailer` | 272 | 272 |
| `highScores` | 528 | 528 |
| `savedGame` | 820 | 820 |
| `hasGame` | 860 | 860 |
| `unusedBoolean` | 861 | 861 |
| `firstRoom` | 862 | 862 |
| `nRooms` | 864 | 864 |
| `rooms[0]` | **866** | **866** |

Only `sizeof(houseType)` differs, because `houseType` contains `long` members
and therefore has alignment 4 under the natural model, so its size rounds up
866 → 868. `roomType` has alignment 2 in both models, so `rooms[]` can and does
start at 866 either way.

`sizeof(houseType)` leaks into the file because two functions size the handle
arithmetically:

```c
/* HouseLegal.c:629-634 — ValidateNumberOfRooms.  countedRooms and
   reportsRooms are `long` (:623); the assignment is line-wrapped at :630-631. */
reportsRooms = (long)(*thisHouse)->nRooms;
countedRooms = (GetHandleSize((Handle)thisHouse) - sizeof(houseType)) / sizeof(roomType);
if (reportsRooms != countedRooms)
	(*thisHouse)->nRooms = (short)countedRooms;   /* trust the file length */

/* HouseLegal.c:762-767 — LopOffExtraRooms */
newSize = sizeof(houseType) + (sizeof(roomType) * (long)r);
SetHandleSize((Handle)thisHouse, newSize);
```

and `WriteHouse` writes exactly `GetHandleSize()` bytes:

```c
byteCount = GetHandleSize((Handle)thisHouse);      /* HouseIO.c:473 */
theErr = FSWrite(houseRefNum, &byteCount, *thisHouse);  /* HouseIO.c:491 */
theErr = SetEOF(houseRefNum, byteCount);           /* HouseIO.c:499 */
```

So a house last saved by a natural-alignment build has **two extra trailing
garbage bytes**. Observed (§11.1): 21 of 22 shipped houses satisfy
`dataLen == 866 + 348*nRooms` exactly; `Sampler` is 1564 bytes with
`nRooms == 2`, i.e. `868 + 348*2 = 1564`, and its two trailing bytes are
`01 01`. Parsing `Sampler`'s rooms at 866 yields the names `"Entrance"` and
`"Welcome"`; parsing at 868 yields garbage. `Sampler` is also the newest file
(timestamp decodes to 2000-05-11 vs 1995 for the rest), consistent with a later
PowerPC build.

**Rule for a Go port:** the rooms array always begins at file offset **866**;
the room count comes from the `nRooms` header field at offset 864 (not from the
file length); tolerate `len - (866 + 348*nRooms)` being 0 or 2 (or, defensively,
anything in `[0, 4)`).

### 2.5 What a Go port must replace

Go has no `#pragma pack`. `encoding/binary` on a Go struct is not an option
either, because Go's own alignment rules differ again. The only safe approach is
**explicit offset-based marshalling**: hand-written `Unmarshal([]byte)` /
`Marshal() []byte` methods that read and write each field at a literal offset
with `binary.BigEndian`. Every offset in this document is a constant a Go port
should hard-code (ideally as named constants so they can be unit-tested against
the shipped houses).

---

## 3. The object model: 9 payload variants + `objectType`

### 3.1 `objectType` — the 12-byte universal object record

`GliderPRO/Headers/GliderStructs.h:90-105`

```c
typedef struct
{
	short		what;						// 2
	union
	{
		blowerType		a;
		furnitureType	b;
		bonusType		c;
		transportType	d;
		switchType		e;
		lightType		f;
		applianceType	g;
		enemyType		h;
		clutterType		i;
	} data;									// 10
} objectType, *objectPtr;					// total = 12
```

| Field | C type | Size | Off | Disk | Notes |
|---|---|---:|---:|:--:|---|
| `what` | `short` | 2 | 0 | yes | Object class code, or `kObjectIsEmpty` = **-1** for an empty slot |
| `data` | union of 9 | 10 | 2 | yes | Interpretation selected by `what` |

`sizeof(objectType) == 12` under both alignment models. All nine union members
are exactly 10 bytes, so there is no padding anywhere and the union offset is a
fixed 2.

**`kObjectIsEmpty` is `-1`, not `0`** (`GliderPRO/Headers/GliderDefines.h:526`).
Verified across all 22 houses: 4070 rooms × 24 slots = 97 680 slots, of which
**66 240** have `what == -1` and **31 440** are live — and `what == 0` never
occurs (§11.3). Getting this wrong is a silent data-corruption bug: `0` is not
a valid object code either, so a naive `what == 0` test would treat all 66 240
empty slots as live.

**Empty slots retain their old payload.** Verified: `Demo House` room 0 slot 23
has `what == -1` but `data == 00 c9 00 3a 27 1b 00 00 01 01`, where
`0x271B == 10011` is a valid `kCustomPict` PICT ID and 10011 is in fact one of
`Demo House`'s PICT resources. A Go port must ignore `data` when
`what == -1` but preserve the bytes on write-back.

### 3.2 The nine payload variants

All nine are exactly 10 bytes with identical layout in both alignment models.

#### 3.2.1 `blowerType` (union member `a`) — air sources and flames

`GliderPRO/Headers/GliderStructs.h:11-19`

| Field | C type | Size | Off | Notes |
|---|---|---:|---:|---|
| `topLeft` | `Point` | 4 | 0 | `v` at 0, `h` at 2 |
| `distance` | `short` | 2 | 4 | Length of the air column, in pixels |
| `initial` | `Boolean` | 1 | 6 | State the object is reset to at room entry |
| `state` | `Boolean` | 1 | 7 | Current on/off state (persisted, but reset from `initial`) |
| `vector` | `Byte` | 1 | 8 | Direction bitmask, see below |
| `tall` | `Byte` | 1 | 9 | Secondary size; `kLiftArea` uses it as height/2 |

`vector` is documented in the header itself
(`GliderPRO/Headers/GliderStructs.h:16-17`):

```
                        F. lf. dn. rt. up
| x | x | x | x | 8 | 4 | 2 | 1 |
```

| Bit | Value | Direction | Observed count |
|---:|---:|---|---:|
| 0 | `0x01` | up | 4900 |
| 1 | `0x02` | right | 412 |
| 2 | `0x04` | down | 326 |
| 3 | `0x08` | left | 402 |

The consumer masks the low nibble and treats the value as an enum, **not** as a
combinable bitmask (`GliderPRO/Sources/ObjectRects.c:523-560`):

```c
switch (theObject.data.a.vector & 0x0F)
{
	case 1: /* up    */ ...
	case 2: /* right */ ...
	case 4: /* down  */ ...
	case 8: /* left  */ ...
}
```

Observed values across all shipped houses: `{1, 2, 4, 8}` plus two corrupt
outliers (`17` ×3 and `20` ×1) which mask to 1 and 4 respectively — so the
`& 0x0F` and the enum-style switch are load-bearing and a Go port must replicate
both, not validate strictly.

Defaults on insertion (`GliderPRO/Sources/ObjectAdd.c:108-167`):
`distance = 64` for floor blowers, `32` for ceiling blowers and fans;
`initial = true`; `state = true`; `vector = 0x01` (up), `0x04` (down),
`0x08` (`kLeftFan`), `0x02` (`kRightFan`); `tall = 0x10` for `kLiftArea`, else
`0x00`.

Observed `(initial, state)` pairs: `(1,1)` ×5751, `(0,0)` ×241, `(0,1)` ×29,
`(1,0)` ×20, and **`(23,23)` ×3** — proof that `Boolean` fields on disk are
*not* restricted to 0/1 and must be read as "non-zero is true".

#### 3.2.2 `furnitureType` (member `b`) — solid obstacles

`GliderPRO/Headers/GliderStructs.h:21-25`

| Field | C type | Size | Off | Notes |
|---|---|---:|---:|---|
| `bounds` | `Rect` | 8 | 0 | `top, left, bottom, right` |
| `pict` | `short` | 2 | 8 | Unused: **0 in all 4695 shipped furniture objects** |

#### 3.2.3 `bonusType` (member `c`) — prizes

`GliderPRO/Headers/GliderStructs.h:27-34`

| Field | C type | Size | Off | Notes |
|---|---|---:|---:|---|
| `topLeft` | `Point` | 4 | 0 | |
| `length` | `short` | 2 | 4 | Header comment: `// 2 grease spill` |
| `points` | `short` | 2 | 6 | Header comment: `// 2 invis bonus` |
| `state` | `Boolean` | 1 | 8 | |
| `initial` | `Boolean` | 1 | 9 | **Note the reversed order vs `blowerType`** |

`state` at offset 8 and `initial` at 9 is the opposite of `blowerType`,
`lightType`, `applianceType` and `enemyType`, which all put `initial` at 8. This
is a real inconsistency in the original header and a classic transcription trap.

`points` is only meaningful for `kInvisBonus` (0x2B). Observed by class:
`kInvisBonus` → `{100: 136, 300: 27, 500: 164}`; every other prize class → 0,
except `kStar` (0x2C) which has two stray values (85 ×1, 145 ×1).

#### 3.2.4 `transportType` (member `d`) — stairs, doors, windows, transports

`GliderPRO/Headers/GliderStructs.h:36-43`

| Field | C type | Size | Off | Notes |
|---|---|---:|---:|---|
| `topLeft` | `Point` | 4 | 0 | |
| `tall` | `short` | 2 | 4 | Header comment: `// 2 invis transport`. For `kDeluxeTrans` this is a **packed** width/height (see below) |
| `where` | `short` | 2 | 6 | Destination room, merged floor/suite; `-1` = unlinked |
| `who` | `Byte` | 1 | 8 | Destination object index 0…23, or **255** = "no specific object" |
| `wide` | `Byte` | 1 | 9 | Width for invisible transports |

`kDeluxeTrans` (0x40) packs two dimensions into `tall`. The encoder and decoder
are both in the editor's corner-drag handler
(`GliderPRO/Sources/ObjectEdit.c:281-289`, verbatim):

```c
case kDeluxeTrans:
	hDelta = ((thisRoom->objects[objActive].data.d.tall & 0xFF00) >> 8) * 4;
	vDelta =  (thisRoom->objects[objActive].data.d.tall & 0x00FF) * 4;
	DragMarqueeCorner(where, &hDelta, &vDelta, false);
	if (hDelta < 64)
		hDelta = 64;
	if (vDelta < 32)
		vDelta = 32;
	thisRoom->objects[objActive].data.d.tall = ((hDelta / 4) << 8) + (vDelta / 4);
```

i.e. `wide/4` in the high byte and `tall/4` in the low byte, both in units of
4 pixels. The `hDelta < 64 → 64` / `vDelta < 32 → 32` clamps mean the smallest
legal encoding is `(16 << 8) + 8 = 4104 = 0x1008`, which is exactly the most
common observed value. Observed `kDeluxeTrans.tall` values include 4104
(`0x1008` → 16×4=64 wide, 8×4=32 tall), 4112, 5136, 9744 … plus two clearly
corrupt negatives (-32722, -32688) — which is why the field must be read as an
**unsigned** 16-bit quantity when unpacking (§11.12).

`where`/`who` are the link fields; see §3.5.

#### 3.2.5 `switchType` (member `e`) — switches and triggers

`GliderPRO/Headers/GliderStructs.h:45-52`

| Field | C type | Size | Off | Notes |
|---|---|---:|---:|---|
| `topLeft` | `Point` | 4 | 0 | |
| `delay` | `short` | 2 | 4 | Frames of delay before the switch fires. Observed 0…120 |
| `where` | `short` | 2 | 6 | Target room (merged floor/suite), `-1` = unlinked. **For `kSoundTrigger` this is a sound resource ID instead** |
| `who` | `Byte` | 1 | 8 | Target object index 0…23, or 255 |
| `type` | `Byte` | 1 | 9 | Action, see below |

`type` enum (`GliderPRO/Headers/GliderDefines.h:441-444`):

| Name | Value | Observed count |
|---|---:|---:|
| `kToggle` | 0 | 609 |
| `kForceOn` | 1 | 393 |
| `kForceOff` | 2 | 363 |
| `kOneShot` | 3 | 320 |

Observed values are exactly `{0,1,2,3}` — no corruption. Defaults on insertion
(`GliderPRO/Sources/ObjectAdd.c:498-506`): `delay = 0`; `where = -1` except
`kSoundTrigger` which gets `3000`; `who = 255`;
`type = kOneShot` for `kTrigger`/`kLgTrigger`, else `kToggle`.

**`kSoundTrigger` (0x49) overloads `where` as a sound resource ID.** Verified
two ways. Source: `ObjectAdd.c:498-501` seeds it to 3000, and
`GliderPRO/Sources/ObjectInfo.c:1172-1265` (`DoCustPictObjectInfo`) validates it
against `[3000, 32767]` while showing the user the prompt `"Sound"` / `"3000"`.
Data: across all 22 houses, `kSoundTrigger.where` takes 21 distinct values, all
in 3000…3058 except two instances of 10000 — never `-1`, and never a
floor/suite-shaped number. `CountHouseLinks`
(`GliderPRO/Sources/House.c:279-287`) deliberately **excludes** `kSoundTrigger`
from the link-bearing switch cases for exactly this reason. Note also that the
1.0.4 runtime ignores the ID and plays a hard-coded chord
(`GliderPRO/Sources/Triggers.c:131-133`, comment `// Change me`).

#### 3.2.6 `lightType` (member `f`) — lights

`GliderPRO/Headers/GliderStructs.h:54-62`

| Field | C type | Size | Off | Notes |
|---|---|---:|---:|---|
| `topLeft` | `Point` | 4 | 0 | |
| `length` | `short` | 2 | 4 | Cord/fixture length. Observed 0…512 |
| `byte0` | `Byte` | 1 | 6 | Unused: **0 in all 2530 shipped lights** |
| `byte1` | `Byte` | 1 | 7 | Unused: **0 in all 2530 shipped lights** |
| `initial` | `Boolean` | 1 | 8 | |
| `state` | `Boolean` | 1 | 9 | |

#### 3.2.7 `applianceType` (member `g`) — appliances and custom pictures

`GliderPRO/Headers/GliderStructs.h:64-72`

| Field | C type | Size | Off | Notes |
|---|---|---:|---:|---|
| `topLeft` | `Point` | 4 | 0 | |
| `height` | `short` | 2 | 4 | Header comment: `// 2 toaster, pict ID`. For `kCustomPict` this is a **PICT resource ID** |
| `byte0` | `Byte` | 1 | 6 | Unused |
| `delay` | `Byte` | 1 | 7 | Frames between activations. Observed 0…255 |
| `initial` | `Boolean` | 1 | 8 | |
| `state` | `Boolean` | 1 | 9 | |

**`kCustomPict` (0x6E) overloads `height` as a PICT resource ID**, valid range
`[10000, 32767]` (`GliderPRO/Sources/ObjectInfo.c:1211-1216`). Verified: across
all houses `kCustomPict.height` has 149 distinct values spanning
**10000…12351**, and `Demo House`'s custom-pict IDs (10008, 10011, 10012, 10014,
10017, 10018, 10036, 10049, 10065) match exactly the PICT resource IDs ≥ 10000
in its resource fork.

#### 3.2.8 `enemyType` (member `h`) — enemies

`GliderPRO/Headers/GliderStructs.h:74-82`

| Field | C type | Size | Off | Notes |
|---|---|---:|---:|---|
| `topLeft` | `Point` | 4 | 0 | |
| `length` | `short` | 2 | 4 | Patrol length / travel distance. Observed 0…310 |
| `delay` | `Byte` | 1 | 6 | **at offset 6** — swapped vs `applianceType` |
| `byte0` | `Byte` | 1 | 7 | Unused: **0 in all 2077 shipped enemies** |
| `initial` | `Boolean` | 1 | 8 | |
| `state` | `Boolean` | 1 | 9 | |

Note the `delay`/`byte0` order is the reverse of `applianceType`. Another
transcription trap.

#### 3.2.9 `clutterType` (member `i`) — decorative, non-interactive

`GliderPRO/Headers/GliderStructs.h:84-88`

| Field | C type | Size | Off | Notes |
|---|---|---:|---:|---|
| `bounds` | `Rect` | 8 | 0 | |
| `pict` | `short` | 2 | 8 | Variant index. Non-zero **only** for `kFlower` |

Verified per class: every clutter class has `pict == 0` except `kFlower` (0x85),
whose values are exactly `{0,1,2,3,4,5}` — matching
`kNumFlowers 6` (`GliderPRO/Headers/GliderDefines.h:458`).

### 3.3 Object class codes and the class → union member mapping

There are **117** object class codes. The mapping from `what` to union member is
a clean function of the high nibble, and it is exhaustively established by
`KeepObjectLegal` (`GliderPRO/Sources/HouseLegal.c:42-597`), which touches every
code exactly once and uses exactly one union member for each.

| Range | Member | C type | Class | Count |
|---|---|---|---|---:|
| `0x01`–`0x10` | `data.a` | `blowerType` | Blowers | 16 |
| `0x11`–`0x1F` | `data.b` | `furnitureType` | Furniture | 15 |
| `0x21`–`0x2F` | `data.c` | `bonusType` | Prizes | 15 |
| `0x31`–`0x40` | `data.d` | `transportType` | Transport | 16 |
| `0x41`–`0x49` | `data.e` | `switchType` | Switches | 9 |
| `0x51`–`0x58` | `data.f` | `lightType` | Lights | 8 |
| `0x61`–`0x6E` | `data.g` | `applianceType` | Appliances | 14 |
| `0x71`–`0x79` | `data.h` | `enemyType` | Enemies | 9 |
| `0x81`–`0x8F` | `data.i` | `clutterType` | Clutter | 15 |

Note `kMaxMasterObjects 216 = kMaxRoomObs * 9`
(`GliderPRO/Headers/GliderDefines.h:266`) — the 9 is the number of classes.

Full enumeration (`GliderPRO/Headers/GliderDefines.h:311-435`). Gaps in the
numbering: `0x00`, `0x20`, `0x30`, `0x4A`–`0x50`, `0x59`–`0x60`, `0x6F`–`0x70`,
`0x7A`–`0x80`, and `0x90` and up (`kNumSrcRects 0x90`, line 437, is the
sentinel/array bound). "Seen" = present in at least one of the 22 shipped
houses.

| Code | Name | Member | Seen |
|---:|---|---|:--:|
| `0x01` | `kFloorVent` | `a` | yes |
| `0x02` | `kCeilingVent` | `a` | yes |
| `0x03` | `kFloorBlower` | `a` | yes |
| `0x04` | `kCeilingBlower` | `a` | yes |
| `0x05` | `kSewerGrate` | `a` | yes |
| `0x06` | `kLeftFan` | `a` | yes |
| `0x07` | `kRightFan` | `a` | yes |
| `0x08` | `kTaper` | `a` | yes |
| `0x09` | `kCandle` | `a` | yes |
| `0x0A` | `kStubby` | `a` | yes |
| `0x0B` | `kTiki` | `a` | yes |
| `0x0C` | `kBBQ` | `a` | yes |
| `0x0D` | `kInvisBlower` | `a` | yes |
| `0x0E` | `kGrecoVent` | `a` | yes |
| `0x0F` | `kSewerBlower` | `a` | yes |
| `0x10` | `kLiftArea` | `a` | yes |
| `0x11` | `kTable` | `b` | yes |
| `0x12` | `kShelf` | `b` | yes |
| `0x13` | `kCabinet` | `b` | yes |
| `0x14` | `kFilingCabinet` | `b` | yes |
| `0x15` | `kWasteBasket` | `b` | yes |
| `0x16` | `kMilkCrate` | `b` | yes |
| `0x17` | `kCounter` | `b` | yes |
| `0x18` | `kDresser` | `b` | yes |
| `0x19` | `kDeckTable` | `b` | yes |
| `0x1A` | `kStool` | `b` | yes |
| `0x1B` | `kTrunk` | `b` | yes |
| `0x1C` | `kInvisObstacle` | `b` | yes |
| `0x1D` | `kManhole` | `b` | yes |
| `0x1E` | `kBooks` | `b` | yes |
| `0x1F` | `kInvisBounce` | `b` | yes |
| `0x21` | `kRedClock` | `c` | yes |
| `0x22` | `kBlueClock` | `c` | yes |
| `0x23` | `kYellowClock` | `c` | yes |
| `0x24` | `kCuckoo` | `c` | yes |
| `0x25` | `kPaper` | `c` | yes |
| `0x26` | `kBattery` | `c` | yes |
| `0x27` | `kBands` | `c` | yes |
| `0x28` | `kGreaseRt` | `c` | yes |
| `0x29` | `kGreaseLf` | `c` | yes |
| `0x2A` | `kFoil` | `c` | yes |
| `0x2B` | `kInvisBonus` | `c` | yes |
| `0x2C` | `kStar` | `c` | yes |
| `0x2D` | `kSparkle` | `c` | yes |
| `0x2E` | `kHelium` | `c` | yes |
| `0x2F` | `kSlider` | `c` | yes |
| `0x31` | `kUpStairs` | `d` | yes |
| `0x32` | `kDownStairs` | `d` | yes |
| `0x33` | `kMailboxLf` | `d` | yes |
| `0x34` | `kMailboxRt` | `d` | yes |
| `0x35` | `kFloorTrans` | `d` | yes |
| `0x36` | `kCeilingTrans` | `d` | yes |
| `0x37` | `kDoorInLf` | `d` | yes |
| `0x38` | `kDoorInRt` | `d` | yes |
| `0x39` | `kDoorExRt` | `d` | yes |
| `0x3A` | `kDoorExLf` | `d` | yes |
| `0x3B` | `kWindowInLf` | `d` | yes |
| `0x3C` | `kWindowInRt` | `d` | yes |
| `0x3D` | `kWindowExRt` | `d` | yes |
| `0x3E` | `kWindowExLf` | `d` | yes |
| `0x3F` | `kInvisTrans` | `d` | yes |
| `0x40` | `kDeluxeTrans` | `d` | yes |
| `0x41` | `kLightSwitch` | `e` | yes |
| `0x42` | `kMachineSwitch` | `e` | yes |
| `0x43` | `kThermostat` | `e` | yes |
| `0x44` | `kPowerSwitch` | `e` | yes |
| `0x45` | `kKnifeSwitch` | `e` | yes |
| `0x46` | `kInvisSwitch` | `e` | yes |
| `0x47` | `kTrigger` | `e` | yes |
| `0x48` | `kLgTrigger` | `e` | yes |
| `0x49` | `kSoundTrigger` | `e` | yes |
| `0x51` | `kCeilingLight` | `f` | yes |
| `0x52` | `kLightBulb` | `f` | yes |
| `0x53` | `kTableLamp` | `f` | yes |
| `0x54` | `kHipLamp` | `f` | yes |
| `0x55` | `kDecoLamp` | `f` | yes |
| `0x56` | `kFlourescent` | `f` | yes |
| `0x57` | `kTrackLight` | `f` | yes |
| `0x58` | `kInvisLight` | `f` | yes |
| `0x61` | `kShredder` | `g` | yes |
| `0x62` | `kToaster` | `g` | yes |
| `0x63` | `kMacPlus` | `g` | yes |
| `0x64` | `kGuitar` | `g` | yes |
| `0x65` | `kTV` | `g` | yes |
| `0x66` | `kCoffee` | `g` | yes |
| `0x67` | `kOutlet` | `g` | yes |
| `0x68` | `kVCR` | `g` | yes |
| `0x69` | `kStereo` | `g` | yes |
| `0x6A` | `kMicrowave` | `g` | yes |
| `0x6B` | `kCinderBlock` | `g` | yes |
| `0x6C` | `kFlowerBox` | `g` | yes |
| `0x6D` | `kCDs` | `g` | yes |
| `0x6E` | `kCustomPict` | `g` | yes |
| `0x71` | `kBalloon` | `h` | yes |
| `0x72` | `kCopterLf` | `h` | yes |
| `0x73` | `kCopterRt` | `h` | yes |
| `0x74` | `kDartLf` | `h` | yes |
| `0x75` | `kDartRt` | `h` | yes |
| `0x76` | `kBall` | `h` | yes |
| `0x77` | `kDrip` | `h` | yes |
| `0x78` | `kFish` | `h` | yes |
| `0x79` | `kCobweb` | `h` | yes |
| `0x81` | `kOzma` | `i` | yes |
| `0x82` | `kMirror` | `i` | yes |
| `0x83` | `kMousehole` | `i` | yes |
| `0x84` | `kFireplace` | `i` | yes |
| `0x85` | `kFlower` | `i` | yes |
| `0x86` | `kWallWindow` | `i` | yes |
| `0x87` | `kBear` | `i` | yes |
| `0x88` | `kCalendar` | `i` | yes |
| `0x89` | `kVase1` | `i` | yes |
| `0x8A` | `kVase2` | `i` | yes |
| `0x8B` | `kBulletin` | `i` | yes |
| `0x8C` | `kCloud` | `i` | yes |
| `0x8D` | `kFaucet` | `i` | yes |
| `0x8E` | `kRug` | `i` | yes |
| `0x8F` | `kChimes` | `i` | yes |

All 117 codes appear in at least one shipped house.

### 3.4 `KeepObjectLegal` — the normalisation rules that shape stored bytes

`GliderPRO/Sources/HouseLegal.c:42-597` is called after every editor mutation
(`KeepAllObjectsLegal`, `HouseLegal.c:919-955`, walks every object in every
room). Its rules therefore describe *invariants a Go port can rely on for
shipped houses*, and *rules a Go port must reimplement if it ever writes house
files*.

Numbered pseudocode of the non-obvious cases (original names in parentheses):

1. **Forced vertical position + `distance` bump** — for `kFloorVent`,
   `kFloorBlower`, `kSewerGrate` and `kFloorTrans`, `topLeft.v` is overwritten
   with the class constant and `distance` is incremented by 2:
   `kFloorVentTop = 305`, `kFloorBlowerTop = 304`, `kSewerGrateTop = 303`,
   `kFloorTransTop = 302` (`GliderDefines.h:467-473`). `kCeilingVentTop = 8`,
   `kCeilingBlowerTop = 5`, `kCeilingTransTop = 6` similarly pin ceiling
   objects.
2. **Parity constraints on `topLeft.h`** — `kStubby` and `kTV` are forced to an
   **odd** `h`; `kTaper`, `kCandle`, `kTiki`, `kBBQ`, all switches and most
   appliances/enemies are forced to an **even** `h`.
3. **`kManhole` snap** — `bounds.left = ((bounds.left + 29) / 64) * 64 + 3`,
   i.e. snapped to the 64-pixel tile grid with a 3-pixel bias.
4. **`kLiftArea`** — `distance = RectWide(bounds)`,
   `tall = RectTall(bounds) / 2`.
5. **`kDeluxeTrans`** — `tall = ((RectWide/4) << 8) + (RectTall/4)` (§3.2.4).
6. **`kInitialGliderSelected` (-2)** — clamps the house's `initial` Point to
   `[0, kRoomWide - kGliderWide] × [0, kTileHigh - kGliderHigh]`, i.e.
   `[0, 464] × [0, 302]` (`HouseLegal.c:63-66`; `kGliderWide 48`,
   `kGliderHigh 20`, `GliderDefines.h:548-549`).

### 3.5 Room links: the `where` / `who` encoding

Links live in `switchType.where`/`.who` (switches 0x41–0x48) and
`transportType.where`/`.who` (transports 0x33, 0x34, 0x35, 0x36, 0x3F, 0x40).
The authoritative list of link-bearing classes is `CountHouseLinks`
(`GliderPRO/Sources/House.c:277-299`) — note it excludes `kSoundTrigger` (§3.2.5)
and excludes stairs/doors/windows (0x31, 0x32, 0x37–0x3E), which pair up
geometrically rather than by stored link (`CheckForStaircasePairs`,
`GliderPRO/Sources/HouseLegal.c:961-1050`). Verified: stairs/doors/windows have
`where == -1` in **all 493** shipped instances.

`where` packs a (floor, suite) pair (`GliderPRO/Sources/Link.c:34-53`):

```c
short MergeFloorSuite (short floor, short suite)
{
	return ((suite * 100) + floor);
}

void ExtractFloorSuite (short combo, short *floor, short *suite)
{
	if ((*thisHouse)->version < 0x0200)                 /* version 1 house */
	{
		*floor = (combo / 100) - kNumUndergroundFloors;
		*suite = combo % 100;
	}
	else                                                /* version 2 house */
	{
		*suite = combo / 100;
		*floor = (combo % 100) - kNumUndergroundFloors;
	}
}
```

`kNumUndergroundFloors = 8` (`GliderDefines.h:535`). Callers add the offset
*before* merging (`DoLink`, `GliderPRO/Sources/Link.c:277`:
`floor += kNumUndergroundFloors;`), so the stored value is

```
where = suite * 100 + (floor + 8)
```

**Version 1 and version 2 houses swap the roles of the quotient and the
remainder.** A v1 house is upgraded in place by `ConvertHouseVer1To2`
(`GliderPRO/Sources/House.c:746-816`), which for every linked switch/transport
does `ExtractFloorSuite(where, &floor, &suite); floor += kNumUndergroundFloors;
where = MergeFloorSuite(floor, suite);` and finally sets
`(*thisHouse)->version = kHouseVersion` (line 814). All 22 shipped houses are
already `version == 0x0200`, so a Go port that only *reads* shipped content can
implement the v2 branch and treat v1 as an upgrade path.

`DoUnlink` (`GliderPRO/Sources/Link.c:325-361`) sets `where = -1; who = 255;`.

Verified empirically across all 22 houses: **2474** link-bearing objects have
`where != -1`. `who` values are `0…23` (matching `kMaxRoomObs 24`) plus 255
(×179) plus a single corrupt `35`. Decoding `where` with C's
truncate-toward-zero division, **172 of 2474 (7 %)** produce an out-of-range
(floor, suite) — mostly `where == -100` in `Castle o' the Air` and
`Land of Illusion` (which decodes to suite -1, floor -8) and one `where == 971`
in `ImagineHouse PRO II` (suite 9, floor 63). These are genuine corruption in
the shipped content; the original code does not validate them at load time, so a
Go port must decode without erroring and fail the *lookup* gracefully.

> **Language trap:** C and Go integer division truncate toward zero; Python's
> `//` and `%` floor. For `where == -100`, C gives `suite = -1, floor = -8`;
> Python gives `suite = -1, floor = 92`. Any verification script must emulate C.

Legality bounds used elsewhere: `ValidateRoomNumbers`
(`GliderPRO/Sources/HouseLegal.c:785-828`) requires `floor ∈ [-7, 56]` and
`suite ∈ [0, 127]`, else it sets `suite = kRoomIsEmpty`. `kMaxNumRoomsH = 128`
(suites), `kMaxNumRoomsV = 64` (floors) (`GliderDefines.h:543-544`).
`CheckDuplicateFloorSuite` (`HouseLegal.c:646-682`) uses a
`kRoomsTimesSuites = 8192` (= 128 × 64) **byte** array (`char *pidgeonHoles`
= `NewPtrClear(8192)`, one byte per cell, not a bitmap) indexed by
`bitPlace = ((floor + 7) * 128) + suite` and blanks duplicates by setting
`suite = kRoomIsEmpty`.

(`kRoomsTimesSuites` is a *function-local* `#define` at
`GliderPRO/Sources/HouseLegal.c:648`, not a global in `GliderDefines.h`.)

Note the `+7` here versus the `+8` (`kNumUndergroundFloors`) that the link
encoding applies: the link encoding and the duplicate-detection index use
**different** floor biases. A Go port must not unify them.

---

## 4. `roomType`

`GliderPRO/Headers/GliderStructs.h:166-180`

```c
typedef struct
{
	Str27		name;						// 28
	short		bounds;						// 2
	Byte		leftStart;					// 1
	Byte		rightStart;					// 1
	Byte		unusedByte;					// 1
	Boolean		visited;					// 1
	short		background;					// 2
	short		tiles[kNumTiles];			// 2 * 8
	short		floor, suite;				// 2 + 2
	short		openings;					// 2
	short		numObjects;					// 2
	objectType	objects[kMaxRoomObs];		// 24 * 12
} roomType, *roomPtr;						// total = 348
```

`sizeof(roomType) == 348` under both alignment models.

| Field | C type | Size | Off | Disk | Notes |
|---|---|---:|---:|:--:|---|
| `name` | `Str27` | 28 | 0 | yes | Pascal string, ≤ 27 chars. Tail is garbage |
| `bounds` | `short` | 2 | 28 | yes | Bit-packed openings override; **0 or odd only**. See §4.2 |
| `leftStart` | `Byte` | 1 | 30 | yes | Glider entry `v` offset when entering from the left, 0…255. Default 32 |
| `rightStart` | `Byte` | 1 | 31 | yes | Same for entry from the right. Default 32 |
| `unusedByte` | `Byte` | 1 | 32 | yes | **Always 0**; forced by `CheckRoomNameLength` |
| `visited` | `Boolean` | 1 | 33 | yes | Genuinely persisted: 436 true / 3634 false across 4070 rooms |
| `background` | `short` | 2 | 34 | yes | PICT resource ID. See §4.3 |
| `tiles[8]` | `short[8]` | 16 | 36 | yes | Column indices into the background strip, each 0…7 |
| `floor` | `short` | 2 | 52 | yes | Floor number; 0 is ground, negatives are basements |
| `suite` | `short` | 2 | 54 | yes | Suite (column) number 0…127, or `kRoomIsEmpty` = -1 for a deleted room |
| `openings` | `short` | 2 | 56 | **stale** | Written 0 at room creation, **never read anywhere**. 0 in all 4070 shipped rooms |
| `numObjects` | `short` | 2 | 58 | yes | Count of slots with `what != -1`; recomputed by `MakeSureNumObjectsJives` |
| `objects[24]` | `objectType[24]` | 288 | 60 | yes | 24 fixed slots; `kMaxRoomObs = 24` |

Constants (`GliderPRO/Headers/GliderDefines.h`): `kNumTiles = 8` (line 496),
`kTileWide = 64` (497), `kTileHigh = 322` (498),
`kRoomWide = 512` (499, commented `// kNumTiles * kTileWide`),
`kFloorSupportTall = 44` (500),
`kVertLocalOffset = 322` (501, commented `// kTileHigh - 39 (was 283, then 295)`);
`kMaxRoomObs = 24` (250); `kRoomIsEmpty = -1` (525); `kObjectIsEmpty = -1` (526);
`kGliderStartsDown = 32` (569).

Room-geometry limits (`GliderPRO/Headers/GliderDefines.h:503-511`):

| Constant | Value | Comment in source |
|---|---:|---|
| `kCeilingLimit` | 8 | |
| `kFloorLimit` | 312 | |
| `kRoofLimit` | 122 | |
| `kLeftWallLimit` | 12 | |
| `kNoLeftWallLimit` | -24 | `// 0 - (kGliderWide / 2)` |
| `kRightWallLimit` | 500 | |
| `kNoRightWallLimit` | 536 | `// kRoomWide + (kGliderWide / 2)` |
| `kNoCeilingLimit` | -10 | |
| `kNoFloorLimit` | 332 | |

`leftStart`/`rightStart` are used as
`kGliderStartsDown + (short)thisRoom->leftStart - 2` when placing the glider
after a horizontal room transition (`GliderPRO/Sources/Transit.c:176`, `:208`),
and are edited with a clamp to `[0, 255]`
(`GliderPRO/Sources/ObjectEdit.c:539-559`). Observed range across all shipped
rooms: 0…255, 112 distinct values for `leftStart`, 115 for `rightStart` — so
they really are used as free-form byte offsets and must be read **unsigned**.

Defaults for a new room (`GliderPRO/Sources/Room.c:169-182`):
`name = "Untitled Room"`, `leftStart = rightStart = 32`, `bounds = 0`,
`unusedByte = 0`, `visited = false`, `background = lastBackground`,
`floor = v`, `suite = h`, `openings = 0`, `numObjects = 0`, and all 24 object
slots set to `what = kObjectIsEmpty`.

### 4.1 Deleted rooms and the room array

A deleted room is marked by `suite == kRoomIsEmpty` (-1); the slot stays in the
array. `CreateNewRoom` reuses the first such slot, or grows the handle with
`PtrAndHand((Ptr)thisRoom, (Handle)thisHouse, sizeof(roomType))`
(`GliderPRO/Sources/Room.c:199-200`). `CompressHouse`
(`GliderPRO/Sources/HouseLegal.c:686-730`) squeezes them out and
`LopOffExtraRooms` (`:736-779`) shrinks the handle and decrements `nRooms`.

Observed: **zero** rooms with `suite == -1` across all 22 shipped houses — they
all ship compressed. But `floor` ranges from **-7 to 39** and `suite` from
**0 to 127** in the shipped data, exactly matching `ValidateRoomNumbers`'
`floor ∈ [-7, 56]`, `suite ∈ [0, 127]`.

### 4.2 `roomType.bounds` — the openings-override bitfield

This is the least obvious field in the whole format. The canonical codec is in
`GliderPRO/Sources/RoomInfo.c:744-780`:

```c
/* decode */
tempShort = thisRoom->bounds >> 1;			// version 2.0 house
originalLeftOpen   = ((tempShort &  1) ==  1);
originalTopOpen    = ((tempShort &  2) ==  2);
originalRightOpen  = ((tempShort &  4) ==  4);
originalBottomOpen = ((tempShort &  8) ==  8);
originalFloor      = ((tempShort & 16) == 16);

/* encode — RoomInfo.c:767-780, verbatim (note the original's typo "orginal") */
tempShort = 0;
if (originalLeftOpen)
	tempShort += 1;
if (originalTopOpen)
	tempShort += 2;
if (originalRightOpen)
	tempShort += 4;
if (originalBottomOpen)
	tempShort += 8;
if (originalFloor)
	tempShort += 16;
tempShort = tempShort << 1;		// shift left 1 bit
tempShort += 1;					// flag that says orginal bounds used
thisRoom->bounds = tempShort;
```

So:

| Bit of raw `bounds` | Mask | Bit after `>> 1` | Mask after `>> 1` | Meaning |
|---:|---:|---:|---:|---|
| 0 | 0x01 | — | — | 1 = "this field is populated"; 0 = "fall back to the `'bnds'` resource" |
| 1 | 0x02 | 0 | 0x01 | left wall open |
| 2 | 0x04 | 1 | 0x02 | top (ceiling) open |
| 3 | 0x08 | 2 | 0x04 | right wall open |
| 4 | 0x10 | 3 | 0x08 | bottom (floor) open |
| 5 | 0x20 | 4 | 0x10 | has floor support (`originalFloor` / `kFloorSupportCheck`) |

There is no bit 6: the encoder can only produce 0 or an odd value ≤ 63.
**Verified over all 4070 shipped rooms** — 32 distinct values occur
(0, then every odd number 1…63 except 51), 2401 rooms have `bounds == 0`,
1669 are non-zero, `bounds & 0x40` is **never** set, and `bounds & 0x20`
is set in exactly **293** rooms.

`IsRoomAStructure` (`GliderPRO/Sources/Room.c:773-786`, condensed — the original
spells the inner assignments as an `if`/`else` pair at `:781-784`):

```c
if ((*thisHouse)->rooms[roomNum].background >= kUserBackground)
{
	if ((*thisHouse)->rooms[roomNum].bounds != 0)
		isStructure = (((*thisHouse)->rooms[roomNum].bounds & 32) == 32);
	else
		isStructure = ((*thisHouse)->rooms[roomNum].background < kUserStructureRange);
}
/* else: a switch over the built-in background IDs, Room.c:789-... */
```

Note this test applies `& 32` to the **raw** `bounds`, without the `>> 1` that
`RoomInfo.c` applies. Raw bit 5 is the bit the editor writes for
`originalFloor`, i.e. **"has floor support"**. So `Room.c` reads the
floor-support bit as "this room is a structure", while `RoomInfo.c`'s dialog
presents the very same bit as the *Floor Support* checkbox. This is a real
double-meaning in the original — a Go port should replicate `Room.c`'s raw
`& 32` test verbatim rather than rationalise it, because that is what the
shipping binary did.

Floor / ceiling queries (`GliderPRO/Sources/Room.c:1138-1205`; the shape below
is condensed — the original uses `switch` statements for the built-in cases at
`:1153-1164` and `:1187-1202`):

```c
/* DoesRoomHaveFloor, Room.c:1143-1149 */
if (thisRoom->background >= kUserBackground) {
	if (thisRoom->bounds != 0)			// is this a version 2.0 house?
		boundsCode = (thisRoom->bounds >> 1);
	else
		boundsCode = GetOriginalBounding(thisRoom->background);
	hasFloor = ((boundsCode & 0x0008) != 0x0008);
} else
	/* switch: kSky, kStratosphere, kStars -> false; default -> true */

/* DoesRoomHaveCeiling, Room.c:1177-1183 — identical but for the mask */
	hasCeiling = ((boundsCode & 0x0002) != 0x0002);
	/* switch: kGarden, kMeadow, kField, kRoof, kSky, kStratosphere, kStars
	   -> false; default -> true */
```

`GetOriginalBounding` (`GliderPRO/Sources/Room.c:937-966`) loads the
4-byte `'bnds'` resource whose ID equals the background PICT ID and recomposes
the code (verbatim, `:951-960`):

```c
boundCode = 0;
HLock((Handle)boundsRes);
if ((*boundsRes)->left)
	boundCode += 1;
if ((*boundsRes)->top)
	boundCode += 2;
if ((*boundsRes)->right)
	boundCode += 4;
if ((*boundsRes)->bottom)
	boundCode += 8;
```

returning 0 if the resource is absent (and raising
`YellowAlert(kYellowNoBoundsRes, 0)` if the PICT exists but the `'bnds'` does
not).

**Verified against real bytes:** `bounds` takes only the values
`{0, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23, 25, 27, 29, 31, 33, 35, 37, 39,
41, 43, 45, 47, 49, 53, 55, 57, 59, 61, 63}` across all 4070 shipped rooms —
i.e. **zero, or odd, never even-and-non-zero**, exactly as the `+= 1` in the
encoder predicts. Maximum 63 = all six bits set.

### 4.3 `background` and `tiles[]`

`background` is a PICT (or `'Date'`) resource ID.

| Range | Meaning | Constant |
|---|---|---|
| 2000…2017 | The 18 built-in backgrounds | `kBaseBackgroundID 2000`, `kNumBackgrounds 18` |
| 3000…3299 | House-supplied "room" backgrounds | `kUserBackground 3000` |
| 3300…3799 | House-supplied "structure" backgrounds | `kUserStructureRange 3300` |
| 10000…32767 | Custom object pictures (`kCustomPict.height`) | — |

The editor accepts a background ID only if
`longID >= 3000 && longID < 3800 && PictIDExists(...)`
(`GliderPRO/Sources/RoomInfo.c:762`).

The 18 built-ins (`GliderPRO/Headers/GliderDefines.h:227-244`), with observed
usage counts across all 4070 shipped rooms:

| ID | Name | Rooms |
|---:|---|---:|
| 2000 | `kSimpleRoom` | 171 |
| 2001 | `kPaneledRoom` | 105 |
| 2002 | `kBasement` | 103 |
| 2003 | `kChildsRoom` | 49 |
| 2004 | `kAsianRoom` | 40 |
| 2005 | `kUnfinishedRoom` | 54 |
| 2006 | `kSwingersRoom` | 49 |
| 2007 | `kBathroom` | 23 |
| 2008 | `kLibrary` | 10 |
| 2009 | `kGarden` (`kFirstOutdoorBack`) | 14 |
| 2010 | `kSkywalk` | 28 |
| 2011 | `kDirt` | 421 |
| 2012 | `kMeadow` | 97 |
| 2013 | `kField` | 25 |
| 2014 | `kRoof` | 181 |
| 2015 | `kSky` | 595 |
| 2016 | `kStratosphere` | 62 |
| 2017 | `kStars` | 246 |

`tiles[i]` is a **column index**, not a resource ID. `ReadyBackground`
(`GliderPRO/Sources/Room.c:244-305`) draws the whole background PICT into a
work GWorld, then blits eight 64×322 columns:

```
for i in 0..7:
    src  = { top=0, left = tiles[i] * kTileWide, bottom = kTileHigh,
             right = tiles[i] * kTileWide + kTileWide }
    dest = { 0, i*64, 322, i*64+64 }
    CopyBits(workSrcMap -> backSrcMap, src, dest, srcCopy)
```

So the background PICT is a horizontal strip of at least `max(tiles)+1` tiles of
64 × 322 pixels, and the room is assembled by picking eight of them (with
repeats allowed). Verified: `tiles[]` values across all 4070 shipped rooms are
exactly `0…7` (histogram: 0→5114, 1→7036, 2→6395, 3→3229, 4→3925, 5→2263,
6→2161, 7→2437), and **591** rooms use the identity mapping `(0,1,2,3,4,5,6,7)`.
`Demo House` room 0 (`"Air Vents"`, background 3000) uses the identity mapping.

If `GetPicture(theID)` fails, `ReadyBackground` retries
`GetResource('Date', theID)` (`GliderPRO/Sources/Room.c:271-272`) — `'Date'` is
an alternate resource type carrying PICT data. The same fallback appears in
`Map.c:172`, `RoomGraphics.c:142` and `RoomInfo.c:845`. **No shipped house
actually contains a `'Date'` resource** (verified: the 22 resource forks contain
only `PICT`, `bnds`, `snd `, `vers`, `ICN#`, `icl8`, `icl4`, `ics#`, `ics8`,
`ics4`), so this path is vestigial — but a Go port that loads third-party houses
should implement it.

### 4.4 Room normalisation rules

`CheckRoomNameLength` (`GliderPRO/Sources/HouseLegal.c:857-880`) forces
`unusedByte = 0` and clamps `name[0]` to 27. `MakeSureNumObjectsJives`
(`:886-910`) recounts `numObjects` as the number of slots with
`what != kObjectIsEmpty`. Verified: **zero** mismatches between the stored
`numObjects` and the actual live-slot count across all 4070 shipped rooms.

---

## 5. `houseType`

`GliderPRO/Headers/GliderStructs.h:182-198`

```c
typedef struct
{
	short		version;					// 2
	short		unusedShort;				// 2
	long		timeStamp;					// 4
	long		flags;						// 4 (bit 0 = wardBit)
	Point		initial;					// 4
	Str255		banner;						// 256
	Str255		trailer;					// 256
	scoresType	highScores;					// 292
	gameType	savedGame;					// 40
	Boolean		hasGame;					// 1
	Boolean		unusedBoolean;				// 1
	short		firstRoom;					// 2
	short		nRooms;						// 2
	roomType	rooms[];					// 348 * nRooms
} houseType, *housePtr, **houseHand;		// total = 866 +
```

| Field | C type | Size | Off | Disk | Notes |
|---|---|---:|---:|:--:|---|
| `version` | `short` | 2 | 0 | yes | `0x0200` in all 22 shipped houses |
| `unusedShort` | `short` | 2 | 2 | **garbage** | Observed: 0 ×12, and 259, 259, -30082, 13107, 222, 26228, 60, 147, 196, 2074 |
| `timeStamp` | `long` | 4 | 4 | yes | Mac seconds since 1904, **with bit 31 cleared and bit 0 repurposed**. See §5.1 |
| `flags` | `long` | 4 | 8 | yes | Bitfield, see §5.2 |
| `initial` | `Point` | 4 | 12 | yes | Starting glider position (`v` at 12, `h` at 14) |
| `banner` | `Str255` | 256 | 16 | yes | Intro text; `\r` (0x0D) separates lines |
| `trailer` | `Str255` | 256 | 272 | yes | End-of-game text |
| `highScores` | `scoresType` | 292 | 528 | yes | See §6.1 |
| `savedGame` | `gameType` | 40 | 820 | **stale** | Vestigial; see §6.2 and §6.5 |
| `hasGame` | `Boolean` | 1 | 860 | yes | true in only 2 of 22 shipped houses |
| `unusedBoolean` | `Boolean` | 1 | 861 | **garbage** | Observed: 0 ×15, 255, 30, 30, 37, 185, 14, 2 |
| `firstRoom` | `short` | 2 | 862 | yes | Index into `rooms[]`; clamped to `[0, nRooms)` by `GetFirstRoomNumber` |
| `nRooms` | `short` | 2 | 864 | yes | Room count, 2…531 in shipped houses |
| `rooms[]` | `roomType[]` | 348·n | **866** | yes | Flexible array. `sizeof(houseType)` is 866 or 868 — the array offset is always 866 |

`sizeof(houseType)` = **866** (`mac68k`) or **868** (natural) — see §2.4 for why
and what it means for the file length.

Version constants (`GliderPRO/Headers/GliderDefines.h:517-518`):
`kHouseVersion = 0x0200`, `kNewHouseVersion = 0x0300`. A house with
`version >= kNewHouseVersion` is rejected outright with
`YellowAlert(kYellowNewerVersion, 0)` (`GliderPRO/Sources/HouseIO.c:395-400`).
A house with `version < kHouseVersion` is upgraded by `ConvertHouseVer1To2`
(`GliderPRO/Sources/HouseIO.c:610-612`).

`InitializeEmptyHouse` sets `version = kHouseVersion`
(`GliderPRO/Sources/House.c:127`) and `hasGame = false` (`:139`).

### 5.1 `timeStamp` — the "house locked" bit and the lost high bit

`WriteHouse` (`GliderPRO/Sources/HouseIO.c:475-489`):

```c
GetDateTime(&timeStamp);          /* UInt32, seconds since 1904-01-01 local */
timeStamp &= 0x7FFFFFFF;          /* HouseIO.c:478 — CLEARS BIT 31 */
if (changeLockStateOfHouse)
	houseUnlocked = !saveHouseLocked;
if (houseUnlocked)
	timeStamp &= 0x7FFFFFFE;      /* HouseIO.c:484 — clear bit 0 */
else
	timeStamp |= 0x00000001;      /* HouseIO.c:486 — set bit 0 */
(*thisHouse)->timeStamp = (long)timeStamp;
```

and `ReadHouse` (`GliderPRO/Sources/HouseIO.c:402`):

```c
houseUnlocked = (((*thisHouse)->timeStamp & 0x00000001) == 0);
```

Consequences a Go port must reproduce:

1. **Bit 0 is a flag, not time.** `bit0 == 0` ⇒ the house is *unlocked*
   (editable); `bit0 == 1` ⇒ *locked* (play only). Observed: **15 of 22** shipped
   houses are locked; the 7 unlocked ones are `California or Bust!`,
   `Castle o' the Air`, `Empty House`, `Fun House`, `Land of Illusion`,
   `Sampler` and `Titanic`.
2. **Bit 31 is destroyed.** Mac time for 1995 is ≈ 2.87 × 10⁹, which needs bit
   31. Clearing it means the stored value is `true_time - 2³¹`. Verified: adding
   2³¹ back to all 22 stored values yields plausible dates 1995-06-08 through
   1995-12-23 for 21 houses and 2000-05-11 for `Sampler`; interpreting the
   stored value as-is yields 1927–1932. So a Go port that wants to display the
   date must add `0x80000000` before converting (and must expect the loss of
   1-second resolution from bit 0).
3. The timestamp is only rewritten when `fileDirty` is set
   (`GliderPRO/Sources/HouseIO.c:475`); an unmodified house keeps its stamp.
4. `houseUnlocked` is additionally forced to `false` if the file is read-only
   (`GliderPRO/Sources/HouseIO.c:428-430`).

There is also a *demo build* check: `if (houseUnlocked) return (false);` under
`#ifdef COMPILEDEMO` (`GliderPRO/Sources/HouseIO.c:403-406`).

### 5.2 `flags`

`GliderPRO/Sources/HouseIO.c:416-418`:

```c
wardBitSet        = (((*thisHouse)->flags & 0x00000001) == 0x00000001);
phoneBitSet       = (((*thisHouse)->flags & 0x00000002) == 0x00000002);
bannerStarCountOn = (((*thisHouse)->flags & 0x00000004) == 0x00000000);
```

| Bit | Mask | Global | Semantics |
|---:|---|---|---|
| 0 | `0x00000001` | `wardBitSet` | Header comment says `bit 0 = wardBit`. Set ⇒ true |
| 1 | `0x00000002` | `phoneBitSet` | Set ⇒ true |
| 2 | `0x00000004` | `bannerStarCountOn` | **Inverted**: set ⇒ *false* |
| 3…31 | — | — | Unused, 0 in all shipped houses |

Observed `flags` values across the 22 shipped houses: `0x00000000` ×14,
`0x00000002` ×7, `0x00000006` ×1 (`Art Museum`). Note bit 2 being *inverted*
means the default (`flags == 0`) has `bannerStarCountOn == true`.

### 5.3 `ReadHouse` — the load path, as pseudocode

`GliderPRO/Sources/HouseIO.c:315-443`. Original identifiers in parentheses.

1. If the house is not open (`houseOpen`), `YellowAlert(kYellowUnaccounted, 2)`
   and fail.
2. If `gameDirty || fileDirty`: if `houseIsReadOnly`, call `WriteScoresToDisk()`
   (which writes a separate 292-byte scores file, §9.4) — else
   `WriteHouse(false)` (`HouseIO.c:327-339`).
3. `GetEOF(houseRefNum, &byteCount)` (`:341`).
4. *(demo build only)* `if (byteCount != 16526L) return false;` (`:349`).
5. Dispose any previous `thisHouse`; `thisHouse = NewHandle(byteCount)` (`:356`);
   `MoveHHi` (`:362`).
6. `SetFPos(houseRefNum, fsFromStart, 0)` (`:364`); `HLock`;
   **`FSRead(houseRefNum, &byteCount, *thisHouse)`** (`:372`) — the entire file
   becomes the struct, no field-by-field parsing.
7. `numberRooms = (*thisHouse)->nRooms` (`:380`).
   *(demo build only)* `if (numberRooms != 45) return false;` (`:382`).
8. `if ((numberRooms < 1) || (byteCount == 0L))` ⇒ `numberRooms = 0`,
   `noRoomAtAll = true`, `YellowAlert(kYellowNoRooms, 0)`, fail (`:385-392`).
9. `wasHouseVersion = (*thisHouse)->version` (`:394`);
   `if (wasHouseVersion >= kNewHouseVersion)` ⇒
   `YellowAlert(kYellowNewerVersion, 0)`, fail (`:395-400`).
10. `houseUnlocked = ((timeStamp & 1) == 0)` (`:402`).
    *(demo build only)* `if (houseUnlocked) return false;`
11. `whichRoom = (*thisHouse)->firstRoom` (`:410`).
    *(demo build only)* `if (whichRoom != 0) return false;`
12. Decode `flags` into `wardBitSet` / `phoneBitSet` / `bannerStarCountOn`
    (`:416-418`).
13. `HUnlock`; `noRoomAtAll = (RealRoomNumberCount() == 0)`;
    `thisRoomNumber = -1`; `previousRoom = -1`;
    `if (!noRoomAtAll) CopyRoomToThisRoom(whichRoom)`.
14. If `houseIsReadOnly`, force `houseUnlocked = false` and call
    `ReadScoresFromDisk()` — the side-car scores file *overwrites* the
    `highScores` field that was just read from the house (`:428-434`; note the
    original's empty `if (ReadScoresFromDisk()) { }` body).
15. `objActive = kNoObjectSelected`; `ReflectCurrentRoom(true)`;
    `gameDirty = fileDirty = false`; `UpdateMenus(false)`; return `true`
    (`:436-442`).

**There is no magic number, no checksum, and no length sanity check beyond
`nRooms >= 1`.** The only structural validation is that the demo build hard-codes
`byteCount == 16526` and `nRooms == 45` — and `866 + 45 × 348 = 16526` exactly,
which independently confirms both the 866 header size and the 348 room stride.
Verified: `Demo House`'s data fork is **exactly 16526** bytes with
`nRooms == 45`.

### 5.4 `WriteHouse` — the save path, as pseudocode

`GliderPRO/Sources/HouseIO.c:448-519`.

1. If `!houseOpen`, `YellowAlert(kYellowUnaccounted, 4)`, fail.
2. `SetFPos(houseRefNum, fsFromStart, 0)`.
3. `CopyThisRoomToRoom()` — flush the editing scratch room back into the handle.
4. `if (checkIt) CheckHouseForProblems();` — the whole `HouseLegal.c` validation
   suite, gated at the call site on the `isHouseChecks` preference.
5. `HLock`; `byteCount = GetHandleSize((Handle)thisHouse)` (`:473`) — **this is
   where 866 vs 868 leaks into the file**.
6. If `fileDirty` (`:475-489`): rewrite `timeStamp` (§5.1) — including
   `if (changeLockStateOfHouse) houseUnlocked = !saveHouseLocked;` (`:480-481`) —
   and `(*thisHouse)->version = wasHouseVersion` (`:488`).
7. **`FSWrite(houseRefNum, &byteCount, *thisHouse)`** (`:491`).
8. `SetEOF(houseRefNum, byteCount)` (`:499`) — truncates if the house shrank.
9. `HUnlock` (`:507`); if `changeLockStateOfHouse`, clear it and
   `ReflectCurrentRoom(true)` (`:509-513`); clear `gameDirty` and `fileDirty`;
   `UpdateMenus(false)`; return `true` (`:515-518`).

Note step 6 writes back `wasHouseVersion` (the version as loaded), not
`kHouseVersion` — except that `HouseIO.c:610-612` sets
`wasHouseVersion = kHouseVersion` after a v1→v2 conversion.

### 5.5 House file creation

`FSpCreate(&theReply.sfFile, 'ozm5', 'gliH', theReply.sfScript)`
(`GliderPRO/Sources/HouseIO.c:267`), followed by `HCreateResFile` for the
resource fork. So a house is creator `'ozm5'`, type `'gliH'`. Verified: all 22
shipped BinHex files report exactly `type=gliH creator=ozm5`.

---

## 6. High scores and saved games

### 6.1 `scoresType`

`GliderPRO/Headers/GliderStructs.h:107-114`

```c
typedef struct
{
	Str31			banner;					// 32		= 32
	Str15			names[kMaxScores];		// 16 * 10	= 160
	long			scores[kMaxScores];		// 4 * 10	= 40
	unsigned long	timeStamps[kMaxScores];	// 4 * 10	= 40
	short			levels[kMaxScores];		// 2 * 10	= 20
} scoresType;								// total 	= 292
```

`kMaxScores = 10` (`GliderPRO/Headers/GliderDefines.h:249`).
`sizeof(scoresType) == 292` under both alignment models (`scores` lands at 192,
which is already 4-aligned).

| Field | C type | Size | Off | Off within `houseType` | Notes |
|---|---|---:|---:|---:|---|
| `banner` | `Str31` | 32 | 0 | 528 | Message from the top scorer |
| `names[10]` | `Str15[10]` | 160 | 32 | 560 | 16 bytes each |
| `scores[10]` | `long[10]` | 40 | 192 | 720 | **signed** |
| `timeStamps[10]` | `unsigned long[10]` | 40 | 232 | 760 | Raw `GetDateTime` — **not** masked like `houseType.timeStamp` |
| `levels[10]` | `short[10]` | 20 | 272 | 800 | Rooms visited, from `CountRoomsVisited()` |

Entry 0 is the best score; the array is kept sorted descending.

`ZeroHighScores` (`GliderPRO/Sources/HighScores.c:323-343`) sets
`banner = thisHouseName`, every `names[i] = "\p--------------"` (14 hyphens),
and all `scores`/`timeStamps`/`levels` to 0.
`ZeroAllButHighestScore` (`:348-367`) does the same but **starts the loop at
i = 1**, preserving entry 0.

`SortHighScores` (`GliderPRO/Sources/HighScores.c:281-317`) is a 10-pass
selection sort that, after picking the maximum, sets the source entry's
`scores[which] = -1L` as a "consumed" marker, then assigns the whole rebuilt
`tempScores` back into `thisHousePtr->highScores`. **A negative sentinel of -1
therefore transiently appears in the score array**, which is why `scores` is
signed.

`TestHighScore` (`GliderPRO/Sources/HighScores.c:374-425`):

1. `if (resumedSavedGame) return false;` — resumed games can never score.
2. Find the first `i` with `theScore > scores[i]`; if none, return false.
3. Write the player's name into `names[kMaxScores - 1]` (index **9**),
   `scores[9] = theScore`, `GetDateTime(&timeStamps[9])`,
   `levels[9] = CountRoomsVisited()`.
4. `SortHighScores()`.
5. If the new entry placed at index 0, also update `banner`.

Verified against real bytes (`Demo House`, `highScores` at offset 528):

```
banner : len=19 "The Return of Ozma!"   tail = 00 e8 72 50 01 26 33 08 01 26 32 dc  (garbage)
name[0]: len=4  "Ozma"                  tail = 26 2b b0 dd dd dd dd dd dd dd dd     (0xDD heap fill)
name[1..9]: len=14 "--------------"     tail = one garbage byte each (36, 00, 00, d0, b4, 12, fc, 0c, 48)
scores    : (7400, 0, 0, 0, 0, 0, 0, 0, 0, 0)
timeStamps: (2888823855, 0, 0, 0, 0, 0, 0, 0, 0, 0)
levels    : (13, 0, 0, 0, 0, 0, 0, 0, 0, 0)
```

Note `timeStamps[0] = 2888823855` exceeds `int32` max — it really is an
*unsigned* long, and unlike `houseType.timeStamp` it is **not** masked, so it
converts directly (2888823855 − 2082844800 = 1995-11-26).

### 6.2 `gameType`

`GliderPRO/Headers/GliderStructs.h:116-134`

```c
typedef struct
{
	short		version;					// 2
	short		wasStarsLeft;				// 2
	long		timeStamp;					// 4
	Point		where;						// 4
	long		score;						// 4
	long		unusedLong;					// 4
	long		unusedLong2;				// 4
	short		energy;						// 2
	short		bands;						// 2
	short		roomNumber;					// 2
	short		gliderState;				// 2
	short		numGliders;					// 2
	short		foil;						// 2
	short		unusedShort;				// 2
	Boolean		facing;						// 1
	Boolean		showFoil;					// 1
} gameType;									// total = 40
```

`sizeof(gameType) == 40` under both alignment models.

| Field | C type | Size | Off | Off in `houseType` | Source at save time |
|---|---|---:|---:|---:|---|
| `version` | `short` | 2 | 0 | 820 | `kSavedGameVersion` = `0x0200` |
| `wasStarsLeft` | `short` | 2 | 2 | 822 | `numStarsRemaining` |
| `timeStamp` | `long` | 4 | 4 | 824 | `GetDateTime` raw, unmasked |
| `where` | `Point` | 4 | 8 | 828 | `where.h = theGlider.dest.left`, `where.v = theGlider.dest.top` |
| `score` | `long` | 4 | 12 | 832 | `theScore` |
| `unusedLong` | `long` | 4 | 16 | 836 | explicitly `0L` |
| `unusedLong2` | `long` | 4 | 20 | 840 | explicitly `0L` |
| `energy` | `short` | 2 | 24 | 844 | `batteryTotal` — **can be negative** (helium) |
| `bands` | `short` | 2 | 26 | 846 | `bandsTotal` |
| `roomNumber` | `short` | 2 | 28 | 848 | `thisRoomNumber` |
| `gliderState` | `short` | 2 | 30 | 850 | `theGlider.mode` |
| `numGliders` | `short` | 2 | 32 | 852 | `mortals` |
| `foil` | `short` | 2 | 34 | 854 | `foilTotal` |
| `unusedShort` | `short` | 2 | 36 | 856 | explicitly `0` |
| `facing` | `Boolean` | 1 | 38 | 858 | `theGlider.facing` |
| `showFoil` | `Boolean` | 1 | 39 | 859 | `showFoil` |

`kSavedGameVersion = 0x0200` (`GliderPRO/Sources/SavedGames.c:14`).
`SaveGame(Boolean doSave)` (`GliderPRO/Sources/SavedGames.c:303-351`) writes all
16 members into `thisHousePtr->savedGame` (17 assignment statements, because
`where.h` and `where.v` are set separately), sets `hasGame = true`, and calls
`WriteHouse(theMode == kEditMode)`. `SaveGame(false)` just clears `hasGame`.
It returns early if `twoPlayerGame`.

Read back by `InitGlider` in the `kResumeGameMode` branch
(`GliderPRO/Sources/Play.c:306+`), which restores `wasStarsLeft`, `score`,
`numGliders`, `energy`, `bands`, `foil`, `gliderState`, `facing`, `showFoil`;
and by `WhereDoesGliderBegin` (`GliderPRO/Sources/House.c:224-240`), which for
`kResumeGameMode` uses `smallGame.where` and for `kNewGameMode` uses
`(*thisHouse)->initial`, building a `kGliderWide × kGliderHigh` (48 × 20) rect
offset by `(h, v)`.

`gameType smallGame;` is the module-level copy
(`GliderPRO/Sources/SavedGames.c:20`).

Verified against real bytes — the two houses with `hasGame == 1`:

| House | version | starsLeft | timeStamp | where (v,h) | score | energy | bands | room | state | gliders | foil | facing | showFoil |
|---|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `ImagineHouse PRO II` | `0x0100` | 3 | 2869500669 → 1994-12-05 | (61, -10) | 5900 | **-150** | 23 | 45 | 0 | 5 | 8 | 1 | 1 |
| `Titanic` | `0x0100` | 1 | 2880300087 → 1995-04-09 | (240, 85) | 4700 | 0 | 0 | 104 | 0 | 2 | 0 | 1 | 0 |

Two facts fall out: (a) both were saved by an **older** build whose
`kSavedGameVersion` was `0x0100`, and (b) `energy` really can be negative.

### 6.3 `savedRoom`

`GliderPRO/Headers/GliderStructs.h:136-142`

```c
typedef struct
{
	short		unusedShort;				// 2
	Byte		unusedByte;					// 1
	Boolean		visited;					// 1
	objectType	objects[kMaxRoomObs];		// 24 * 12
} savedRoom, *saveRoomPtr;					// total = 292
```

| Field | C type | Size | Off |
|---|---|---:|---:|
| `unusedShort` | `short` | 2 | 0 |
| `unusedByte` | `Byte` | 1 | 2 |
| `visited` | `Boolean` | 1 | 3 |
| `objects[24]` | `objectType[24]` | 288 | 4 |

`sizeof(savedRoom) == 292` in both models. This is a *deliberate prefix-compatible
subset* of `roomType`: the first four bytes mirror `roomType.bounds` /
`leftStart` / `rightStart`… **no** — the offsets do not line up
(`roomType.visited` is at 33, `savedRoom.visited` at 3), so it is not a memcpy
of a `roomType`. It exists only inside `game2Type`, which is dead code.

### 6.4 `game2Type` (dead code)

`GliderPRO/Headers/GliderStructs.h:144-164`

| Field | C type | Size | Off (mac68k) | Off (natural) |
|---|---|---:|---:|---:|
| `house` | `FSSpec` | 70 | 0 | 0 |
| `version` | `short` | 2 | 70 | 70 |
| `wasStarsLeft` | `short` | 2 | 72 | 72 |
| `timeStamp` | `long` | 4 | **74** | **76** |
| `where` | `Point` | 4 | 78 | 80 |
| `score` | `long` | 4 | 82 | 84 |
| `unusedLong` | `long` | 4 | 86 | 88 |
| `unusedLong2` | `long` | 4 | 90 | 92 |
| `energy` | `short` | 2 | 94 | 96 |
| `bands` | `short` | 2 | 96 | 98 |
| `roomNumber` | `short` | 2 | 98 | 100 |
| `gliderState` | `short` | 2 | 100 | 102 |
| `numGliders` | `short` | 2 | 102 | 104 |
| `foil` | `short` | 2 | 104 | 106 |
| `nRooms` | `short` | 2 | 106 | 108 |
| `facing` | `Boolean` | 1 | 108 | 110 |
| `showFoil` | `Boolean` | 1 | 109 | 111 |
| `savedData[]` | `savedRoom[]` | 292·n | **110** | **112** |

Header size 110 (`mac68k`) / 112 (natural). The author's `// total = 114`
comment counts the flexible array as 4 bytes (`// 4` on line 163).

This is the **only** struct in the persisted set whose *field offsets*, not just
its total size, differ between the two alignment models — because `FSSpec` is 70
bytes (even but not a multiple of 4), so the following `long timeStamp` lands at
74 under `mac68k` and 76 under natural alignment.

### 6.5 Saved games are dead code in 1.0.4

* `SaveGame2()` (`GliderPRO/Sources/SavedGames.c:30-163`): the entire body is
  commented out with the note `// Add NavServices later.` It *would* have used
  `byteCount = sizeof(game2Type) + sizeof(savedRoom) * numRooms` (`:56`) and
  `FSpCreate(..., 'ozm5', 'gliG', ...)` (`:122`).
* `OpenSavedGame()` (`:167`) begins with
  `return false; // TEMP fix this iwth NavServices`, followed by a commented
  body whose validation would have been: house name match,
  `timeStamp != savedGame->timeStamp`, `version != kSavedGameVersion`,
  `nRooms != thisHousePtr->nRooms`.
* `DoCommandKey` (`GliderPRO/Sources/Input.c:53-72`) wires Cmd-Q to
  `if (QuerySaveGame()) SaveGame2();` and Cmd-S to `SaveGame2();` — both
  no-ops.
* `QueryResumeGame` is never called and `SaveGame(false)` is commented out at
  `GliderPRO/Sources/Menu.c:458`.

**Therefore `houseType.savedGame` and `houseType.hasGame` are vestigial in
1.0.4 and `kResumeGameMode` is unreachable.** Verified: 20 of 22 shipped houses
have `hasGame == 0`, and their `savedGame` blocks contain obvious junk — e.g.
`Art Museum`'s is `f6 f6 f6 ff f6 f6 f6 f6 f6 ff 00 00 00 ff …`, which is
8-bit indexed PICT pixel data, and `Sampler`'s contains `cc cc` fill. A Go port
should parse the field for round-trip fidelity but must not trust it.

---

## 7. `prefsInfo` — the preferences file

`GliderPRO/Headers/Externs.h:231-269`. **This is the only struct in the program
explicitly wrapped in the alignment pragma:**

```c
#pragma options align=mac68k        /* Externs.h:231 */
typedef struct { ... } prefsInfo;
#pragma options align=reset        /* Externs.h:269 */
```

`sizeof(prefsInfo)` = **226** (`mac68k`) / 228 (natural). The pragma is
load-bearing: without it, `wasLeftMap` moves from offset 146 to 148 and every
field after it shifts by 2, making a 68k-written prefs file unreadable by a PPC
build. Calhoun clearly hit this and fixed it here (and *only* here), which is
strong evidence that everything else — including `houseType` — was compiled at
whatever the project default was, and hence the 866/868 discrepancy of §2.4.

| Field | C type | Size | Off (mac68k) | Off (natural) | Meaning / global it feeds |
|---|---|---:|---:|---:|---|
| `wasDefaultName` | `Str32` | **33** | 0 | 0 | `thisHouseName` — last-played house file name |
| `wasLeftName` | `Str15` | 16 | 33 | 33 | `leftName` — display name of the "left" key |
| `wasRightName` | `Str15` | 16 | 49 | 49 | `rightName` |
| `wasBattName` | `Str15` | 16 | 65 | 65 | `batteryName` |
| `wasBandName` | `Str15` | 16 | 81 | 81 | `bandName` |
| `wasHighName` | `Str15` | 16 | 97 | 97 | `highName` — player's name for the score table |
| `wasHighBanner` | `Str31` | 32 | 113 | 113 | `highBanner` — player's boast message |
| *(interior pad)* | — | 1 / 3 | **145** | **145-147** | `Str31` ends at 144 (odd next offset); `mac68k` inserts **1** pad byte before the `long`, natural alignment inserts **3**. Written from uninitialised stack. |
| `wasLeftMap` | `long` | 4 | **146** | **148** | `theGlider.leftKey` — KeyMap bit offset |
| `wasRightMap` | `long` | 4 | 150 | 152 | `theGlider.rightKey` |
| `wasBattMap` | `long` | 4 | 154 | 156 | `theGlider.battKey` |
| `wasBandMap` | `long` | 4 | 158 | 160 | `theGlider.bandKey` |
| `wasVolume` | `short` | 2 | 162 | 164 | `isVolume`, clamped 1…3 on a fresh install |
| `prefVersion` | `short` | 2 | 164 | 166 | Must equal `kPrefsVersion` = `0x0034` (52) |
| `wasMaxFiles` | `short` | 2 | 166 | 168 | `maxFiles`; clamped to 12 if `< 12 \|\| > 500` |
| `wasEditH` | `short` | 2 | 168 | 170 | `isEditH` — editor window left |
| `wasEditV` | `short` | 2 | 170 | 172 | `isEditV` |
| `wasMapH` | `short` | 2 | 172 | 174 | `isMapH` — map window left |
| `wasMapV` | `short` | 2 | 174 | 176 | `isMapV` |
| `wasMapWide` | `short` | 2 | 176 | 178 | `mapRoomsWide` |
| `wasMapHigh` | `short` | 2 | 178 | 180 | `mapRoomsHigh` |
| `wasToolsH` | `short` | 2 | 180 | 182 | `isToolsH` |
| `wasToolsV` | `short` | 2 | 182 | 184 | `isToolsV` |
| `wasLinkH` | `short` | 2 | 184 | 186 | `isLinkH` |
| `wasLinkV` | `short` | 2 | 186 | 188 | `isLinkV` |
| `wasCoordH` | `short` | 2 | 188 | 190 | `isCoordH` |
| `wasCoordV` | `short` | 2 | 190 | 192 | `isCoordV` |
| `isMapLeft` | `short` | 2 | 192 | 194 | `mapLeftRoom` — map scroll position (suite) |
| `isMapTop` | `short` | 2 | 194 | 196 | `mapTopRoom` — map scroll position (floor) |
| `wasNumNeighbors` | `short` | 2 | 196 | 198 | `numNeighbors` (1 or 9) |
| `wasDepthPref` | `short` | 2 | 198 | 200 | `isDepthPref`: 0 `kSwitchIfNeeded`, 1 `kSwitchTo256Colors`, 2 `kSwitchTo16Grays` |
| `wasToolGroup` | `short` | 2 | 200 | 202 | `toolMode`: 0 `kSelectTool` … 9 `kClutterMode` |
| `smWarnings` | `short` | 2 | 202 | 204 | `numSMWarnings` — Sound Manager nag counter |
| `wasFloor` | `short` | 2 | 204 | 206 | `wasFloor` — last edited room's floor |
| `wasSuite` | `short` | 2 | 206 | 208 | `wasSuite` |
| `wasZooms` | `Boolean` | 1 | 208 | 210 | `doZooms` |
| `wasMusicOn` | `Boolean` | 1 | 209 | 211 | `isMusicOn` |
| `wasAutoEdit` | `Boolean` | 1 | 210 | 212 | `autoRoomEdit` |
| `wasDoColorFade` | `Boolean` | 1 | 211 | 213 | `isDoColorFade` |
| `wasMapOpen` | `Boolean` | 1 | 212 | 214 | `isMapOpen` |
| `wasToolsOpen` | `Boolean` | 1 | 213 | 215 | `isToolsOpen` |
| `wasCoordOpen` | `Boolean` | 1 | 214 | 216 | `isCoordOpen` |
| `wasQuickTrans` | `Boolean` | 1 | 215 | 217 | `quickerTransitions` |
| `wasIdleMusic` | `Boolean` | 1 | 216 | 218 | `isPlayMusicIdle` |
| `wasGameMusic` | `Boolean` | 1 | 217 | 219 | `isPlayMusicGame` |
| `wasEscPauseKey` | `Boolean` | 1 | 218 | 220 | `isEscPauseKey` |
| `wasDoAutoDemo` | `Boolean` | 1 | 219 | 221 | `doAutoDemo` |
| `wasScreen2` | `Boolean` | 1 | 220 | 222 | `isUseSecondScreen`; forced false if `thisMac.numScreens < 2` |
| `wasDoBackground` | `Boolean` | 1 | 221 | 223 | `doBackground` |
| `wasHouseChecks` | `Boolean` | 1 | 222 | 224 | `isHouseChecks` |
| `wasPrettyMap` | `Boolean` | 1 | 223 | 225 | `doPrettyMap` |
| `wasBitchDialogs` | `Boolean` | 1 | 224 | 226 | `doBitchDialogs` |
| *(tail pad)* | — | 1 | 225 | 227 | Struct alignment is 2 ⇒ one pad byte, written from **uninitialised stack** |

**51 real fields + 2 pad bytes = 226 bytes** (33 + 5×16 + 32 = 145 bytes of
Pascal strings, 1 pad, 4×4 = 16 bytes of key maps, 23×2 = 46 bytes of shorts,
17×1 = 17 Booleans, 1 tail pad). `Str32` really is **33** bytes
(1 length + 32 chars) — an odd size, which is why `wasLeftName` starts at the odd
offset 33, and the run of six odd-sized/odd-offset Pascal strings is what pushes
`wasHighBanner` to end on the odd offset 144 and forces the interior pad.

### 7.1 The two commented-out `long`s

`GliderPRO/Headers/Externs.h:240` reads
`//	long		encrypted, fakeLong;` — commented out. But
`GliderPRO/Sources/Main.c:75` (`encryptedNumber = thePrefs.encrypted;`) and
`:231-232` (`thePrefs.encrypted = encryptedNumber; thePrefs.fakeLong = Random();`)
still reference them, each guarded by `#ifndef COMPILENOCP`
(`GliderPRO/Sources/Main.c:74`/`:76` and `:230`/`:233`). The declaration
`extern long encryptedNumber;` is itself commented out at
`GliderPRO/Sources/Main.c:32`.
**The GPL source therefore only compiles with `COMPILENOCP` defined** (copy
protection removed), and the shipping 1.0.4 prefs record is 226 bytes with no
`encrypted`/`fakeLong`. If those two `long`s were re-enabled, `prefsInfo` would be
234 bytes and every offset from 146 onwards would shift by 8 — so a Go port must
not "restore" them.

### 7.2 Prefs file location and identity

| Property | Value | Citation |
|---|---|---|
| Creator | `'ozm5'` (`kPrefCreatorType`) | `GliderPRO/Sources/Prefs.c:18` |
| File type | `'gliP'` (`kPrefFileType`) | `GliderPRO/Sources/Prefs.c:19` |
| File name | `"Glider Prefs"` (`kPrefFileName`) | `GliderPRO/Sources/Prefs.c:20` |
| Fallback folder name | `"Preferences"` (`kDefaultPrefFName`) | `GliderPRO/Sources/Prefs.c:21` |
| Folder | `FindFolder(kOnSystemDisk, kPreferencesFolderType, kCreateFolder, …)` | `GliderPRO/Sources/Prefs.c:63-64` |
| Fallback folder creation | `PBDirCreate` with name from `GetIndString(_, 160, 1)` | `GliderPRO/Sources/Prefs.c:79-86` |
| Version constant | `kPrefsVersion = 0x0034` | `GliderPRO/Sources/Main.c:16` |
| Bad-version alert | `kNewPrefsAlertID = 160` | `GliderPRO/Sources/Prefs.c:23` |
| Data-fork size | `sizeof(prefsInfo)` = 226 | `GliderPRO/Sources/Prefs.c:127`, `:193` |

`CanUseFindFolder()` (`GliderPRO/Sources/Prefs.c:39-55`) gates on
`Gestalt(gestaltFindFolderAttr)` bit `31 - gestaltFindFolderPresent` — but it is
never called from `SavePrefs`/`LoadPrefs`, which use `GetPrefsFPath`
unconditionally.

### 7.3 `LoadPrefs` / `SavePrefs`, as pseudocode

`SavePrefs(thePrefs, versionNow)` — `GliderPRO/Sources/Prefs.c:148-162`:

1. `thePrefs->prefVersion = versionNow;` — the *only* field `SavePrefs` sets.
2. `GetPrefsFPath(&prefDirID, &systemVolRef)`; fail ⇒ false.
3. `WritePrefs(...)`:
   a. `FSMakeFSSpec(volRef, dirID, "\pGlider Prefs", &theSpecs)`; on `fnfErr`,
      `FSpCreate(&theSpecs, 'ozm5', 'gliP', smSystemScript)`.
   b. `FSpOpenDF(&theSpecs, fsRdWrPerm, &fileRefNum)`.
   c. `byteCount = sizeof(*thePrefs);` (`Prefs.c:127`)
   d. **`FSWrite(fileRefNum, &byteCount, thePrefs)`** (`Prefs.c:129`) — blind
      whole-struct write. Note there is **no `SetEOF`**, so if a longer prefs file
      already existed, the tail survives; the reader ignores it because it only
      asks for 226 bytes.
   e. `FSClose`.

`LoadPrefs(thePrefs, versionNeed)` — `GliderPRO/Sources/Prefs.c:240-269`:

1. `GetPrefsFPath(...)`; fail ⇒ false (⇒ defaults).
2. `ReadPrefs(...)`:
   a. `FSMakeFSSpec`; `fnfErr` propagates.
   b. `FSpOpenDF(..., fsRdWrPerm, ...)` — note **read/write** permission even to
      read.
   c. `byteCount = sizeof(*thePrefs);` (`Prefs.c:193`)
   d. **`FSRead(fileRefNum, &byteCount, thePrefs)`** (`Prefs.c:195`).
   e. `FSClose`.
3. `if (theErr == eofErr)` ⇒ `BringUpDeletePrefsAlert(); DeletePrefs(...); return false;`
   — i.e. a *short* prefs file (from an older, smaller `prefsInfo`) is deleted
   and defaults are used.
4. `else if (theErr != noErr)` ⇒ false.
5. `if (thePrefs->prefVersion != versionNeed)` ⇒
   `BringUpDeletePrefsAlert(); DeletePrefs(...); return false;`
   — note this is an **exact inequality**, not `<`, so a *newer* prefs file is
   also nuked.
6. Return true.

### 7.4 Defaults when no prefs exist

`GliderPRO/Sources/Main.c:122-189`. A Go port needs these verbatim to reproduce
the first-launch experience.

| Global | Default | Line |
|---|---|---:|
| `thisHouseName` | `"Slumberland"` (`"Demo House"` in the demo build) | 127 / 125 |
| `leftName` | `"lf arrow"` | 129 |
| `rightName` | `"rt arrow"` | 130 |
| `batteryName` | `"dn arrow"` | 131 |
| `bandName` | `"up arrow"` | 132 |
| `highName` | `"Your Name"` | 133 |
| `highBanner` | `"Your Message Here"` | 134 |
| `theGlider.leftKey` | `kLeftArrowKeyMap` = 124 | 135 |
| `theGlider.rightKey` | `kRightArrowKeyMap` = 123 | 136 |
| `theGlider.battKey` | `kDownArrowKeyMap` = 122 | 137 |
| `theGlider.bandKey` | `kUpArrowKeyMap` = 121 | 138 |
| `isVolume` | system volume, clamped to `[1, 3]` | 140-144 |
| `isDepthPref` | `kSwitchIfNeeded` = 0 | 146 |
| `isSoundOn` | true | 147 |
| `isMusicOn` | true | 148 |
| `isPlayMusicIdle` | true | 149 |
| `isPlayMusicGame` | true | 150 |
| `isHouseChecks` | true | 151 |
| `doZooms` | true | 152 |
| `quickerTransitions` | false | 153 |
| `numNeighbors` | 9 | 154 |
| `isDoColorFade` | true | 155 |
| `maxFiles` | 48 | 156 |
| `willMaxFiles` | 48 | 157 |
| `isEditH`, `isEditV` | 3, 41 | 158-159 |
| `isMapH`, `isMapV` | 3, 100 | 160-162 |
| `mapRoomsWide`, `mapRoomsHigh` | 15, 4 | 163-164 |
| `isToolsH`, `isToolsV` | 100, 35 | 166-167 |
| `isLinkH`, `isLinkV` | 50, 80 | 168-169 |
| `isCoordH`, `isCoordV` | 50, 204 | 171-172 |
| `mapLeftRoom`, `mapTopRoom` | 60, 50 | 173-174 |
| `wasFloor`, `wasSuite` | 0, 0 | 175-176 |
| `numSMWarnings` | 0 | 177 |
| `autoRoomEdit` | true | 178 |
| `isMapOpen` | true | 179 |
| `isToolsOpen` | true | 180 |
| `isCoordOpen` | false | 181 |
| `toolMode` | `kBlowerMode` = 1 | 182 |
| `doAutoDemo` | true | 183 |
| `isEscPauseKey` | false | 184 |
| `isUseSecondScreen` | false | 185 |
| `doBackground` | false | 186 |
| `doPrettyMap` | false | 187 |
| `doBitchDialogs` | true | 188 |

Post-processing applied to **both** paths (`GliderPRO/Sources/Main.c:191-200`):

```c
if ((numNeighbors > 1) && (thisMac.screen.right <= 512))
	numNeighbors = 1;
UnivGetSoundVolume(&wasVolume, thisMac.hasSM3);   /* remember the system volume */
UnivSetSoundVolume(isVolume, thisMac.hasSM3);

if (isVolume == 0)
	isSoundOn = false;
else
	isSoundOn = true;
```

`WriteOutPrefs` (`GliderPRO/Sources/Main.c:208-279`) writes back all 49 fields
from a **stack-local `prefsInfo thePrefs;` that is never zeroed**, so the unused
tails of the six Pascal strings and the single pad byte at offset 225 contain
whatever was on the stack. A Go port must therefore never compare prefs files
byte-for-byte, and should zero-fill its own output.

Two fields are read from prefs but written from a *different* global:
`wasMaxFiles` is loaded into `maxFiles` (`Main.c:87`) but saved from
`willMaxFiles` (`Main.c:244`). `wasQuickTrans`, `wasHouseChecks`, `wasIdleMusic`
etc. round-trip normally.

---

## 8. Runtime-only structures

None of these ever touch the disk. They are documented because a Go port needs
equivalents, and because their *sizes* enter the memory budget check.

### 8.1 `gliderType` — the player

`GliderPRO/Headers/GliderStructs.h:200-216`. Size **110** (`mac68k`) /
**112** (natural — the four `long` key maps force 4-byte alignment for the whole
struct). Runtime-only, so the difference is harmless.

| Field | C type | Size | Off | Purpose |
|---|---|---:|---:|---|
| `src` | `Rect` | 8 | 0 | Source rect in the glider sprite sheet |
| `mask` | `Rect` | 8 | 8 | Source rect in the mask |
| `dest` | `Rect` | 8 | 16 | Current on-screen rect (48 × 20) |
| `whole` | `Rect` | 8 | 24 | Union of `dest` and previous `dest` (dirty rect) |
| `destShadow` | `Rect` | 8 | 32 | Shadow's destination |
| `wholeShadow` | `Rect` | 8 | 40 | Shadow dirty rect |
| `clip` | `Rect` | 8 | 48 | Clip rect |
| `enteredRect` | `Rect` | 8 | 56 | Rect of the object the glider entered through |
| `leftKey` | `long` | 4 | 64 | KeyMap bit offset for "go left" |
| `rightKey` | `long` | 4 | 68 | |
| `battKey` | `long` | 4 | 72 | Battery / helium |
| `bandKey` | `long` | 4 | 76 | Fire rubber band |
| `hVel` | `short` | 2 | 80 | Horizontal velocity |
| `vVel` | `short` | 2 | 82 | Vertical velocity |
| `wasHVel` | `short` | 2 | 84 | Previous frame's `hVel` |
| `wasVVel` | `short` | 2 | 86 | |
| `vDesiredVel` | `short` | 2 | 88 | Target vertical velocity (drafts) |
| `hDesiredVel` | `short` | 2 | 90 | |
| `mode` | `short` | 2 | 92 | State machine; saved as `gameType.gliderState` |
| `frame` | `short` | 2 | 94 | Animation frame |
| `wasMode` | `short` | 2 | 96 | |
| `facing` | `Boolean` | 1 | 98 | `true` = facing right; saved to `gameType.facing` |
| `tipped` | `Boolean` | 1 | 99 | Tipping over (banking) |
| `sliding` | `Boolean` | 1 | 100 | On grease |
| `ignoreLeft` | `Boolean` | 1 | 101 | Pass through the left wall |
| `ignoreRight` | `Boolean` | 1 | 102 | Pass through the right wall |
| `fireHeld` | `Boolean` | 1 | 103 | Band key held (debounce) |
| `which` | `Boolean` | 1 | 104 | Which player (two-player mode) |
| `heldLeft` | `Boolean` | 1 | 105 | |
| `heldRight` | `Boolean` | 1 | 106 | |
| `dontDraw` | `Boolean` | 1 | 107 | Suppress drawing this frame |
| `ignoreGround` | `Boolean` | 1 | 108 | Pass through the floor |
| *(pad)* | — | 1 | 109 | Trailing pad to 110 |

Two instances exist: `theGlider` and `theGlider2` (two-player mode).
`kGliderWide = 48`, `kGliderHigh = 20`
(`GliderPRO/Headers/GliderDefines.h:548-549`).

### 8.2 The other runtime aggregates

All sizes verified by compiling the declarations under both models
(`Rect`/`Point` 2-packed, pointers modelled as 4 bytes).

| Struct | Header line | Size | Fields |
|---|---|---:|---|
| `hotObject` | `GliderStructs.h:218-225` | 16 | `Rect bounds; short action, who; Boolean isOn, stillOver, doScrutinize;` |
| `savedType` | `:227-233` | **16** | `Rect dest; GWorldPtr map; short where, who;` — `map` is a 4-byte pointer on the Mac |
| `sparkleType` | `:235-239` | 10 | `Rect bounds; short mode;` |
| `flyingPtType` | `:241-249` | 28 | `Rect dest, whole; short start, stop, mode, loops, hVel, vVel;` |
| `flameType` | `:251-256` | 20 | `Rect dest, src; short mode, who;` |
| `pendulumType` | `:258-264` | 26 | `Rect dest, src; short mode, where, who, link; Boolean toOrFro, active;` |
| `boundsType` | `:266-272` | **4** | `Boolean left, top, right, bottom;` — **also the on-disk `'bnds'` resource**, §9.3 |
| `bandType` | `:274-279` | 16 | `Rect dest; short mode, count, hVel, vVel;` |
| `linksType` | `:281-285` | 8 | `short srcRoom, srcObj, destRoom, destObj;` |
| `greaseType` | `:287-295` | 26 | `Rect dest; short mapNum, mode, who, where, start, stop, frame, hotNum; Boolean isRight;` |
| `starType` | `:297-302` | 24 | `Rect dest, src; short mode, who, link, where;` |
| `shredType` | `:304-308` | 10 | `Rect bounds; short frame;` |
| `dynaType` | `:310-320` | 36 | `Rect dest, whole; short hVel, vVel, type, count, frame, timer, position, room; Byte byte0, byte1; Boolean moving, active;` |
| `objDataType` | `:322-332` | 26 | `short roomNum, objectNum, roomLink, objectLink, localLink, hotNum, dynaNum; objectType theObject;` |
| `demoType` | `:334-339` | **6** | `long frame; char key; char padding;` — **also the on-disk `'demo'` resource**, §9.6 |
| `retroLink` | `:341-345` | 4 | `short room, object;` |
| `macEnviron` | `Environ.h:7-35` | 44 | See §8.4 |
| `marquee` | `Marquee.h:14-20` | 82 | `Pattern pats[7]; Rect bounds, handle; short index, direction, dist; Boolean active, paused, handled;` |
| `sizeType` | `Environ.c:31-36` | 10 | `short flags; long mem1, mem2;` (mac68k) — 12 under natural |
| `pageType` | `GameOver.c:25-30` | 22 | `Rect dest, was; short frame, counter; Boolean stuck;` |
| `phoneType` | `Play.c:26-31` | 6 | `short nextRing, rings, delay;` |
| `trigType` | `Triggers.c:15-21` | 12 | `short object, room, index, timer, what; Boolean armed;` |
| `acurRec` | `AnimCursor.c:18-27` | 4 + 4·n | See §9.7 — **an on-disk resource format** |

`objDataType`'s per-field comments (`GliderPRO/Headers/GliderStructs.h:322-332`)
are worth quoting because the names are opaque:

```c
typedef struct
{
	short		roomNum;	// room # object in (real number)
	short		objectNum;	// obj. # in house (real number)
	short		roomLink;	// room # object linked to (if any)
	short		objectLink;	// obj. # object linked to (if any)
	short		localLink;	// index in master list if exists
	short		hotNum;		// index into active rects (if any)
	short		dynaNum;	// index into dinahs (if any)
	objectType	theObject;	// actual object data
} objDataType, *objDataPtr;
```

### 8.3 `CreatePointers` — the runtime allocation table

`GliderPRO/Sources/StructuresInit2.c:185-300`. Every `NewPtr` in the program's
startup path, with the resulting byte count under `mac68k` sizes:

| Global | Element | Count constant | Count | Bytes |
|---|---|---|---:|---:|
| `thisRoom` | `roomType` | 1 | 1 | 348 |
| `hotSpots` | `hotObject` | `kMaxHotSpots` | 56 | 896 |
| `sparkles` | `sparkleType` | `kMaxSparkles` | 3 | 30 |
| `flyingPoints` | `flyingPtType` | `kMaxFlyingPts` | 3 | 84 |
| `flames` | `flameType` | `kMaxCandles` | 20 | 400 |
| `tikiFlames` | `flameType` | `kMaxTikis` | 8 | 160 |
| `bbqCoals` | `flameType` | `kMaxCoals` | 8 | 160 |
| `pendulums` | `pendulumType` | `kMaxPendulums` | 8 | 208 |
| `savedMaps` | `savedType` | `kMaxSavedMaps` | 24 | 384 |
| `bands` | `bandType` | `kMaxRubberBands` | 2 | 32 |
| `grease` | `greaseType` | `kMaxGrease` | 16 | 416 |
| `theStars` | `starType` | `kMaxStars` | 4 | 96 |
| `shreds` | `shredType` | `kMaxShredded` | 4 | 40 |
| `dinahs` | `dynaType` | `kMaxDynamicObs` | 18 | 648 |
| `masterObjects` | `objDataType` | `kMaxMasterObjects` | 216 | 5616 |
| `srcRects` | `Rect` | `kNumSrcRects` | — | 8 each |
| `theHousesSpecs` | `FSSpec` | `maxFiles` | 48 (default) | 70 each |
| `demoData` | raw bytes | `kDemoLength` | — | 6702 |

After allocating `savedMaps`, the loop sets `savedMaps[i].map = nil` — the
`GWorldPtr` is filled in lazily. `kMaxMasterObjects = 216` is commented
`// kMaxRoomObs * 9` (`GliderPRO/Headers/GliderDefines.h:266`): 24 objects ×
(the room itself + its 8 neighbours).

Full limit table (`GliderPRO/Headers/GliderDefines.h:249-268`):

| Constant | Value |
|---|---:|
| `kMaxScores` | 10 |
| `kMaxRoomObs` | 24 |
| `kMaxSparkles` | 3 |
| `kMaxFlyingPts` | 3 |
| `kMaxCandles` | 20 |
| `kMaxTikis` | 8 |
| `kMaxCoals` | 8 |
| `kMaxPendulums` | 8 |
| `kMaxHotSpots` | 56 |
| `kMaxSavedMaps` | 24 |
| `kMaxRubberBands` | 2 |
| `kMaxGrease` | 16 |
| `kMaxStars` | 4 |
| `kMaxShredded` | 4 |
| `kMaxDynamicObs` | 18 |
| `kMaxMasterObjects` | 216 |
| `kMaxViewWidth` | 1536 |
| `kMaxViewHeight` | 1026 |
| `kMaxTriggers` (`Triggers.c:12`) | 16 |
| `kMaxSoundTriggers` (`ObjectAdd.c:17`) | 1 |

### 8.4 `macEnviron` (`GliderPRO/Headers/Environ.h:11-32`)

Size **44** in both models. `extern macEnviron thisMac;` (`Environ.h:35`)

| Field | C type | Size | Off | Meaning |
|---|---|---:|---:|---|
| `screen` | `Rect` | 8 | 0 | Main screen bounds |
| `gray` | `Rect` | 8 | 8 | Desktop ("gray") region bounds |
| `dirID` | `long` | 4 | 16 | Application's directory ID |
| `wasDepth` | `short` | 2 | 20 | Bit depth on launch (restored on quit) |
| `isDepth` | `short` | 2 | 22 | Current bit depth |
| `thisResFile` | `short` | 2 | 24 | Application resource file refNum |
| `numScreens` | `short` | 2 | 26 | Monitor count |
| `vRefNum` | `short` | 2 | 28 | Application's volume refNum |
| `can1Bit` | `Boolean` | 1 | 30 | Screen supports 1-bit |
| `can4Bit` | `Boolean` | 1 | 31 | |
| `can8Bit` | `Boolean` | 1 | 32 | |
| `wasColorOrGray` | `Boolean` | 1 | 33 | |
| `hasWNE` | `Boolean` | 1 | 34 | `WaitNextEvent` available (trap `0x60`) |
| `hasSystem7` | `Boolean` | 1 | 35 | |
| `hasColor` | `Boolean` | 1 | 36 | Color QuickDraw |
| `hasGestalt` | `Boolean` | 1 | 37 | |
| `canSwitch` | `Boolean` | 1 | 38 | `SetDepth` available (trap `0xA2`) |
| `canColor` | `Boolean` | 1 | 39 | |
| `hasSM3` | `Boolean` | 1 | 40 | Sound Manager 3.0 |
| `hasQT` | `Boolean` | 1 | 41 | QuickTime |
| `hasDrag` | `Boolean` | 1 | 42 | Drag Manager |
| *(pad)* | — | 1 | 43 | |

Related constants (`GliderPRO/Sources/Environ.c:18-28`): `kSwitchDepthAlert 130`,
`kSetMemoryAlert 180`, `kLowMemoryAlert 181`, `kWNETrap 0x60`,
`kSetDepthTrap 0xA2`, `kUnimpTrap 0x9F`, `kGestaltTrap 0xAD`,
`kDisplay9Inch 1`, `kDisplay12Inch 2`, `kDisplay13Inch 3`.

`CheckMemorySize` (`GliderPRO/Sources/Environ.c:562-698`) sums the memory budget
into `bytesNeeded`, and at `:657` adds `kDemoLength` explicitly — the demo buffer
is a fixed 6702-byte cost. Most of the other terms are of the form
`bytesNeeded += (<pixels> * (long)thisMac.isDepth) / 8L;` for an offscreen map
plus `bytesNeeded += <pixels> / 8L;` for its 1-bit mask, which is where the
8-bit-indexed-colour assumption shows up as an allocation size.

### 8.5 Headers with no structs

Four of the headers named in this assignment declare **only** externs, no types:

| Header | Lines | Contents |
|---|---:|---|
| `GliderPRO/Headers/House.h` | 12 | `extern Str32 thisHouseName; extern Boolean houseUnlocked;` (plus prototypes) |
| `GliderPRO/Headers/Room.h` | 12 | `extern GWorldPtr backSrcMap;` |
| `GliderPRO/Headers/Player.h` | 12 | `extern GWorldPtr shadowSrcMap, shadowMaskMap;` |
| `GliderPRO/Headers/Map.h` | 12 | `extern GWorldPtr nailSrcMap; extern WindowPtr mapWindow;` |
| `GliderPRO/Headers/Scoreboard.h` | 15 | `#include <QDOffscreen.h>` + `extern GWorldPtr boardSrcMap, badgeSrcMap, boardTSrcMap, boardGSrcMap, boardPSrcMap;` |
| `GliderPRO/Headers/Objects.h` | 42 | 35 `extern GWorldPtr` declarations, nothing else — the sprite-sheet / mask GWorld pair for each object family: `blower`, `furniture`, `bonus`, `points`, `trans`, `switch` (no mask), `light`, `appliance`, `toast`, `shred`, `balloon`, `copter`, `dart`, `ball`, `drip`, `enemy`, `fish`, `clutter` |

`kScoreboardTall = 20` (`GliderPRO/Headers/GliderDefines.h:515`) is the only
scoreboard geometry constant in the headers; `kScoreboardHigh = 0` /
`kScoreboardLow = 1` (lines 513-514) select its screen position.

---

## 9. On-disk formats

Six distinct on-disk artefacts. Everything is **big-endian**.

| # | Artefact | Type / Creator | Fork | Layout | Section |
|---|---|---|---|---|---|
| 1 | House | `'gliH'` / `'ozm5'` | data | `houseType` + `nRooms` × `roomType` | §9.1 |
| 2 | House media | — | resource | `PICT`, `bnds`, `snd `, icon family, `vers` | §9.2, §9.3 |
| 3 | House movie | — | separate `<name>.mov` file | QuickTime | §9.9 |
| 4 | Preferences | `'gliP'` / `'ozm5'` | data | `prefsInfo`, 226 bytes | §7.2 |
| 5 | High scores side-car | `'gliS'` / `'ozm5'` | data | `scoresType`, 292 bytes | §9.4 |
| 6 | Saved game | `'gliG'` / `'ozm5'` | data | `game2Type` + n × `savedRoom` — **never written by 1.0.4** | §6.5 |

Plus three resource formats defined *inside* the program that a Go port has to
re-implement as data tables: `'demo'` (§9.6), `'acur'` (§9.7), `'PAT#'` (§9.8).

### 9.1 The house data fork

```
offset 0        houseType header                    866 bytes
offset 866      roomType rooms[0]                   348 bytes
offset 1214     roomType rooms[1]                   348 bytes
...
offset 866 + 348*(nRooms-1)   roomType rooms[nRooms-1]
[optional 0 or 2 trailing bytes of slop]
```

`ValidateNumberOfRooms` (`GliderPRO/Sources/HouseLegal.c:621-640`) derives the
count the other way round (verbatim, `:629-637`; `countedRooms` and
`reportsRooms` are declared `long` at `:623`):

```c
reportsRooms = (long)(*thisHouse)->nRooms;
countedRooms = (GetHandleSize((Handle)thisHouse) -
		sizeof(houseType)) / sizeof(roomType);
if (reportsRooms != countedRooms)
{
	(*thisHouse)->nRooms = (short)countedRooms;
	numberRooms = (*thisHouse)->nRooms;
	houseErrors++;
}
```

so `sizeof(houseType)` is used as the room-array base **as well as** the header
size. In a `mac68k` build both are 866 and the arithmetic is exact.

**Verified across all 22 shipped houses** (data forks extracted from the BinHex
wrappers):

| House | bytes | nRooms | 866+348·n | Δ | version | unusedShort | flags | timeStamp | lock | decoded date | hasGame | unusedBool | firstRoom | rsrc bytes |
|---|---:|---:|---:|---:|---|---:|---|---:|---:|---|---:|---:|---:|---:|
| Art Museum | 38798 | 109 | 38798 | 0 | 0x0200 | -30082 | 0x0006 | 746622005 | 1 | 1995-09-16 | 0 | 255 | 91 | 7476159 |
| CD Demo House | 72554 | 206 | 72554 | 0 | 0x0200 | 0 | 0x0002 | 742294873 | 1 | 1995-07-28 | 0 | 0 | 70 | 1612342 |
| California or Bust! | 6434 | 16 | 6434 | 0 | 0x0200 | 13107 | 0x0002 | 745882296 | 0 | 1995-09-08 | 0 | 0 | 14 | 259542 |
| Castle o' the Air | 30446 | 85 | 30446 | 0 | 0x0200 | 259 | 0x0000 | 741536868 | 0 | 1995-07-19 | 0 | 30 | 33 | 306364 |
| Davis Station | 23486 | 65 | 23486 | 0 | 0x0200 | 222 | 0x0002 | 743681789 | 1 | 1995-08-13 | 0 | 37 | 4 | 1871320 |
| **Demo House** | **16526** | **45** | **16526** | 0 | 0x0200 | 0 | 0x0000 | 743682113 | 1 | 1995-08-13 | 0 | 0 | 0 | 491757 |
| Empty House | 13046 | 35 | 13046 | 0 | 0x0200 | 0 | 0x0000 | 741421292 | 0 | 1995-07-18 | 0 | 0 | 0 | 2670 |
| Fun House | 15830 | 43 | 15830 | 0 | 0x0200 | 26228 | 0x0000 | 742681616 | 0 | 1995-08-01 | 0 | 0 | 29 | 662443 |
| Grand Prix | 61766 | 175 | 61766 | 0 | 0x0200 | 60 | 0x0000 | 741444157 | 1 | 1995-07-18 | 0 | 185 | 127 | 1347764 |
| ImagineHouse PRO II | 97958 | 279 | 97958 | 0 | 0x0200 | 0 | 0x0000 | 740149565 | 1 | 1995-07-03 | **1** | 0 | 1 | 677770 |
| In The Mirror | 34622 | 97 | 34622 | 0 | 0x0200 | 147 | 0x0000 | 741512193 | 1 | 1995-07-19 | 0 | 0 | 6 | 151870 |
| Land of Illusion | 106310 | 303 | 106310 | 0 | 0x0200 | 259 | 0x0002 | 745268052 | 0 | 1995-08-31 | 0 | 30 | 43 | 401793 |
| Leviathan | 165122 | 472 | 165122 | 0 | 0x0200 | 0 | 0x0000 | 741514803 | 1 | 1995-07-19 | 0 | 0 | 39 | 1901282 |
| Metropolis | 45062 | 127 | 45062 | 0 | 0x0200 | 196 | 0x0000 | 741642315 | 1 | 1995-07-20 | 0 | 14 | 8 | 946907 |
| Nemo's Market | 44018 | 124 | 44018 | 0 | 0x0200 | 0 | 0x0002 | 741121233 | 1 | 1995-07-14 | 0 | 0 | 0 | 590602 |
| Rainbow's End | 78470 | 223 | 78470 | 0 | 0x0200 | 0 | 0x0002 | 741121489 | 1 | 1995-07-14 | 0 | 0 | 30 | 316065 |
| **Sampler** | **1564** | **2** | **1562** | **+2** | 0x0200 | 0 | 0x0000 | 893435880 | 0 | 2000-05-11 | 0 | 2 | 1 | **286** |
| Slumberland | 134150 | 383 | 134150 | 0 | 0x0200 | 0 | 0x0000 | 743682377 | 1 | 1995-08-13 | 0 | 0 | 126 | 1041112 |
| SpacePods | 140762 | 402 | 140762 | 0 | 0x0200 | 0 | 0x0002 | 741122073 | 1 | 1995-07-14 | 0 | 0 | 259 | 636844 |
| Teddy World | 185654 | 531 | 185654 | 0 | 0x0200 | 0 | 0x0000 | 744401697 | 1 | 1995-08-21 | 0 | 0 | 0 | 3107348 |
| The Asylum Pro | 49586 | 140 | 49586 | 0 | 0x0200 | 0 | 0x0000 | 738016929 | 1 | 1995-06-08 | 0 | 0 | 20 | 494107 |
| Titanic | 73250 | 208 | 73250 | 0 | 0x0200 | 2074 | 0x0000 | 755123966 | 0 | 1995-12-23 | **1** | 0 | 92 | 845727 |

Total: **4070 rooms** across the corpus. Notes:

* 21 of 22 are exactly `866 + 348·nRooms`. **`Sampler` is 2 bytes longer**, i.e.
  `868 + 348·2`, and its rooms verifiably start at **866** (parsing at 866 gives
  room names `"Entrance"` and `"Welcome"`; parsing at 868 gives garbage). It was
  saved in 2000 by a PowerPC-natural-alignment build — see §2.4.
* `SpacePods` has `firstRoom == 259` with `nRooms == 402`, and `Grand Prix` has
  `firstRoom == 127` with `nRooms == 175`. Both in range, but the clamp in
  `GetFirstRoomNumber` (`GliderPRO/Sources/House.c:196-217`) exists because
  out-of-range values were evidently possible.
* `version` is `0x0200` in every shipped house. No `0x0100` house survives in
  the corpus, so `ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746-816`) is
  untestable against real data.

### 9.2 The house resource fork

Full inventory across all 22 houses:

| Type | Total count | Purpose |
|---|---:|---|
| `PICT` | 919 | Backgrounds (3000-3799) and custom object pictures (10000+) |
| `bnds` | 70 | Per-background opening/floor flags, §9.3 |
| `snd ` | 63 | House-supplied sounds for `kSoundTrigger` (IDs 3000+) |
| `ICN#` | 25 | Finder icon, 1-bit |
| `icl8` | 25 | Finder icon, 8-bit |
| `icl4` | 23 | Finder icon, 4-bit |
| `ics#` | 22 | Finder small icon, 1-bit |
| `ics8` | 22 | Finder small icon, 8-bit |
| `ics4` | 22 | Finder small icon, 4-bit |
| `vers` | 12 | Version strings (7 houses) |

Per house:

| House | Resources |
|---|---|
| Art Museum | PICT ×76, snd ×10, ICN#/icl8/icl4/ics#/ics8/ics4 ×1 each |
| CD Demo House | PICT ×116, snd ×10, icon family ×1 |
| California or Bust! | PICT ×30, snd ×3, icon family ×1 |
| Castle o' the Air | PICT ×12, **bnds ×9**, icon family ×1 |
| Davis Station | PICT ×54, snd ×5, icon family ×1 |
| Demo House | PICT ×19, **bnds ×10**, snd ×1, icon family ×1 |
| Empty House | icon family ×1 only |
| Fun House | PICT ×26, icon family ×1 |
| Grand Prix | PICT ×70, snd ×3, vers ×2, icon family ×1 |
| ImagineHouse PRO II | PICT ×45, **bnds ×14**, snd ×2, vers ×2, icl8 ×2, ICN# ×1 … |
| In The Mirror | PICT ×15, snd ×2, vers ×2, icl8 ×2, icl4 ×2, ICN# ×2 … |
| Land of Illusion | PICT ×20, **bnds ×5**, icon family ×1 |
| Leviathan | PICT ×53, **bnds ×4**, snd ×12, vers ×2, icl8 ×2, ICN# ×2 … |
| Metropolis | PICT ×39, vers ×1, icon family ×1 |
| Nemo's Market | PICT ×73, snd ×5, icon family ×1 |
| Rainbow's End | PICT ×17, **bnds ×6**, snd ×1, icon family ×1 |
| **Sampler** | **(empty fork — 286 bytes)** |
| Slumberland | PICT ×20, **bnds ×20**, icon family ×1 |
| SpacePods | PICT ×49, snd ×3, icon family ×1 |
| Teddy World | PICT ×120, vers ×2, icon family ×2 each |
| The Asylum Pro | PICT ×17, **bnds ×2**, icon family ×1 |
| Titanic | PICT ×48, snd ×6, vers ×1, ICN# ×2, icon family ×1 |

The house resource fork is opened separately at
`GliderPRO/Sources/HouseIO.c:571`:
`houseResFork = FSpOpenResFile(&theHousesSpecs[thisHouseIndex], fsCurPerm);`
This pushes the house's resource map on top of the application's, so
`GetPicture(2000)` finds the built-in and `GetPicture(3000)` finds the house's.

**Verified: the canonical empty resource fork is 286 bytes.** `Sampler.rsrc`:

```
offset 0   : 00 00 01 00   dataOff  = 0x100 = 256
offset 4   : 00 00 01 00   mapOff   = 0x100 = 256
offset 8   : 00 00 00 00   dataLen  = 0
offset 12  : 00 00 00 1e   mapLen   = 30
offset 16..255 : reserved (NOT zero — contains stale bytes)
offset 256 : 00 00 01 00 00 00 01 00 00 00 00 00 00 00 00 1e   (map's copy of the header)
offset 272 : 00 00 00 00   nextMap = 0
offset 276 : 00 00        fileRef = 0
offset 278 : 00 00        attrs   = 0
offset 280 : 00 1c        typeListOff = 28
offset 282 : 00 1e        nameListOff = 30
offset 284 : ff ff        type count - 1 = 0xFFFF  ==>  ZERO types
```

**The `0xFFFF` is the trap:** the classic resource-map count fields are stored as
`count - 1`, so an empty fork stores `0xFFFF`, not `0`. A naive `raw + 1` yields
65536. A Go resource-fork reader must special-case `raw == 0xFFFF` ⇒ 0 types.
The same `count - 1` convention applies to the per-type reference count.

Resource map layout (as used by the parser that produced the tables above):

| Offset | Size | Field |
|---:|---:|---|
| `mapOff + 0` | 16 | Copy of the fork header |
| `mapOff + 16` | 4 | `nextResourceMap` (0 in files) |
| `mapOff + 20` | 2 | `fileRef` (0 in files) |
| `mapOff + 22` | 2 | `fileAttributes` |
| `mapOff + 24` | 2 | `typeListOffset` (relative to `mapOff`) |
| `mapOff + 26` | 2 | `nameListOffset` (relative to `mapOff`) |
| `mapOff + typeListOffset + 0` | 2 | `numTypes - 1` |
| `… + 2 + 8·i` | 4 | resource type (`'PICT'`) |
| `… + 6 + 8·i` | 2 | `numRefs - 1` for that type |
| `… + 8 + 8·i` | 2 | offset of the reference list, relative to `mapOff + typeListOffset` |
| ref `+ 0` | 2 | resource ID (`short`, **signed**) |
| ref `+ 2` | 2 | name offset into name list, `0xFFFF` = unnamed |
| ref `+ 4` | 1 | attributes |
| ref `+ 5` | 3 | 24-bit offset into the data area |
| ref `+ 8` | 4 | reserved handle field |
| `dataOff + dataOffset24 + 0` | 4 | resource body length |
| `dataOff + dataOffset24 + 4` | *len* | resource body |

### 9.3 The `'bnds'` resource

Exactly 4 bytes, laid out as `boundsType` (`GliderStructs.h:266-272`):

```c
typedef struct { Boolean left, top, right, bottom; } boundsType, *boundsPtr, **boundsHand;
```

Read by `GetOriginalBounding` (`GliderPRO/Sources/Room.c:937-966`) with the
resource **ID equal to the background PICT ID**. Every value observed is 0 or 1,
so it is genuinely four booleans, in the order `left, top, right, bottom` — note
this is **not** QuickDraw `Rect` order (`top, left, bottom, right`).

All 14 distinct bodies observed across the 70 shipped `'bnds'` resources:

| Body (hex) | left | top | right | bottom | Count | Composed code |
|---|:--:|:--:|:--:|:--:|---:|---:|
| `00 00 00 00` | 0 | 0 | 0 | 0 | 5 | 0 |
| `00 00 01 00` | 0 | 0 | 1 | 0 | 2 | 4 |
| `00 00 01 01` | 0 | 0 | 1 | 1 | 3 | 12 |
| `00 01 00 00` | 0 | 1 | 0 | 0 | 2 | 2 |
| `00 01 00 01` | 0 | 1 | 0 | 1 | 1 | 10 |
| `00 01 01 00` | 0 | 1 | 1 | 0 | 2 | 6 |
| `00 01 01 01` | 0 | 1 | 1 | 1 | 2 | 14 |
| `01 00 00 00` | 1 | 0 | 0 | 0 | 7 | 1 |
| `01 00 00 01` | 1 | 0 | 0 | 1 | 1 | 9 |
| `01 00 01 00` | 1 | 0 | 1 | 0 | 13 | 5 |
| `01 00 01 01` | 1 | 0 | 1 | 1 | 1 | 13 |
| `01 01 00 00` | 1 | 1 | 0 | 0 | 2 | 3 |
| `01 01 01 00` | 1 | 1 | 1 | 0 | 12 | 7 |
| `01 01 01 01` | 1 | 1 | 1 | 1 | 17 | 15 |

`01 01 01 01` (fully open) and `01 00 01 00` (left+right open — a corridor) are
the two commonest, which matches how Glider PRO houses are laid out.

Note the composed code has **no bit 4 (`floorSupport`)** — `GetOriginalBounding`
can only ever return 0…15, so a room that relies on the `'bnds'` fallback can
never claim floor support. Only a room with `bounds != 0` can (`bit 5` of the raw
`bounds`).

### 9.4 The high-scores side-car file

Glider PRO stores high scores **twice**: inside the house
(`houseType.highScores`) and, when the house file is read-only, in a separate
file. `GliderPRO/Sources/HouseIO.c:327-339` (verbatim):

```c
if (gameDirty || fileDirty)
{
	if (houseIsReadOnly)
	{
		if (!WriteScoresToDisk())
		{
			YellowAlert(kYellowFailedWrite, 0);
			return(false);
		}
	}
	else if (!WriteHouse(false))
		return(false);
}
```

| Property | Value | Citation |
|---|---|---|
| Folder | `<System Folder>/Preferences/G-PRO Scores ƒ/` — `FindFolder(kOnSystemDisk, kPreferencesFolderType, kCreateFolder, …)` then `FSpDirCreate` | `GliderPRO/Sources/HighScores.c:649-650`, `:654`, `:656` |
| Folder name bytes | `"\pG-PRO Scores \xC4"` — the last byte is MacRoman `0xC4` = `ƒ` | `GliderPRO/Sources/HighScores.c:654`, `:679`, `:696`; verified from the raw source bytes |
| File name | `thisHouseName` (the house's file name, verbatim) | `GliderPRO/Sources/HighScores.c:757`, `:816` |
| Type / creator | `'gliS'` / `'ozm5'` | `GliderPRO/Sources/HighScores.c:727` |
| Size | `sizeof(scoresType)` = 292 | `GliderPRO/Sources/HighScores.c:771` |
| Layout | Raw `scoresType`, no header | `GliderPRO/Sources/HighScores.c:776` |

`FindHighScoresFolder` (`:665-716`) walks the Preferences folder with
`PBGetCatInfo` looking for a directory whose name equals `"G-PRO Scores ƒ"`
(case-insensitive, diacritic-sensitive: `EqualString(..., true, true)`), and
creates it with `FSpDirCreate` on `fnfErr`.

**`ReadScoresFromDisk` has a heap-overflow bug** worth knowing about because it
constrains what a faithful port may do
(`GliderPRO/Sources/HighScores.c:823-841`):

```c
theErr = GetEOF(scoresRefNum, &byteCount);       /* WHOLE FILE LENGTH */
...
theScores = &((*thisHouse)->highScores);         /* a 292-byte field at offset 528 */
theErr = FSRead(scoresRefNum, &byteCount, theScores);   /* reads byteCount bytes! */
```

If the `'gliS'` file is longer than 292 bytes, the excess overwrites
`houseType.savedGame`, `hasGame`, `unusedBoolean`, `firstRoom`, `nRooms` and then
the room array. `WriteScoresToDisk` always `SetEOF`s to 292, so this only bites
with a hand-edited or truncated-then-grown file. A Go port should read at most
292 bytes and ignore the rest.

### 9.5 The saved-game file (unimplemented)

Type `'gliG'` / creator `'ozm5'`
(`GliderPRO/Sources/SavedGames.c:122`, in commented-out code). Would have been
`game2Type` header (110 bytes `mac68k`) followed by `nRooms` × `savedRoom`
(292 bytes each): `byteCount = sizeof(game2Type) + sizeof(savedRoom) * numRooms`
(`GliderPRO/Sources/SavedGames.c:56`). **Never produced by 1.0.4.** See §6.5.

### 9.6 The `'demo'` resource (ID 128)

Loaded once at startup (`GliderPRO/Sources/StructuresInit2.c:280-298`):

```c
#ifdef CREATEDEMODATA
	demoData = (demoPtr)NewPtr(sizeof(demoType) * 2000);
#else
	demoData = (demoPtr)NewPtr(kDemoLength);
	tempHandle = GetResource('demo', 128);
	BlockMove(*tempHandle, demoData, kDemoLength);
	ReleaseResource(tempHandle);
#endif
```

`kDemoLength = 6702` (`GliderPRO/Headers/GliderDefines.h:625`). This is a
**`BlockMove` of a resource straight into a `demoType` array**, so the struct
layout *is* the file layout.

`demoType` = `{ long frame; char key; char padding; }` = **6 bytes** under
`mac68k`, **8 bytes** under natural alignment. 6702 / 6 = **1117 exactly**;
6702 / 8 is not an integer. **This is the strongest single proof that the
shipping build used `mac68k` alignment globally** — under natural alignment the
demo playback would read garbage from record 1 onwards.

**Verified from the resource fork dump:**

```
'demo' 128: len = 6702 = 1117 x 6
records[0..5].frame = 46, 47, 48, 49, 56, 57      (monotonically non-decreasing)
records[-3:].frame  = 3412, 3413, 3414
first four records  = 00 00 00 2e 00 72 | 00 00 00 2f 00 72 | 00 00 00 30 00 72 | 00 00 00 31 00 45
                      ^^^^^^^^^^^ frame ^^ key
key histogram (offset 4) = { 0: 910, 1: 198, 3: 9 }
padding (offset 5) = 109 distinct values  ==>  uninitialised
```

The playback switch (`GliderPRO/Sources/Input.c:224-263`):

| `key` | Action |
|---:|---|
| 0 | Move left |
| 1 | Move right |
| 2 | Battery / helium — **never appears in the shipped demo** |
| 3 | Fire rubber band |

Note `key` is at offset **4** and `padding` at offset **5**: the byte at offset 5
is uninitialised stack/heap junk (109 distinct values), which is why reading the
key from offset 5 produces nonsense. `frame` is a `long` counter of game ticks;
the recorder is `LogDemoKey` (`GliderPRO/Sources/Input.c:42-48`).

### 9.7 The `'acur'` resource (ID 128) — animated cursor

`GliderPRO/Sources/AnimCursor.c:18-27`:

```c
typedef struct
{
	short	n;					/* number of cursor frames */
	short	index;				/* current frame (runtime scratch) */
	union {
		Handle	cursorHdl;		/* runtime: 4-byte Handle */
		short	resID;			/* on disk: 2-byte CURS/crsr resource ID */
	} frame[1];
} acurRec, *acurPtr, **acurHandle;
```

Header 4 bytes, then `n` × 4 bytes per frame. The union is **4 bytes wide** but
on disk only the first 2 bytes are meaningful (the resource ID); at runtime
`GetMonoCursors` / `GetColorCursors` overwrite the 4 bytes **in place** with the
`Handle`:

```c
/* GliderPRO/Sources/AnimCursor.c:51-64 (mono), verbatim */
j = (*ballCursH)->n;					/* number of frames */
for (i = 0; i < j; i++)
{
	cursHdl = GetCursor((*ballCursH)->frame[i].resID);
	...
	DetachResource((Handle)cursHdl);
	(*ballCursH)->frame[i].cursorHdl = (Handle)cursHdl;
}
```

**Verified: the shipped `'acur' 128` is 52 bytes = 4 + 12 × 4:**

```
00 0c 00 00                     n = 12,  index = 0
00 a0 00 00   frame[ 0] resID = 160
00 9f 00 00   frame[ 1] resID = 159
00 9e 00 00   frame[ 2] resID = 158
00 9d 00 00   frame[ 3] resID = 157
00 9c 00 00   frame[ 4] resID = 156
00 9b 00 00   frame[ 5] resID = 155
00 9a 00 00   frame[ 6] resID = 154
00 99 00 00   frame[ 7] resID = 153
00 98 00 00   frame[ 8] resID = 152
00 97 00 00   frame[ 9] resID = 151
00 96 00 00   frame[10] resID = 150
00 95 00 00   frame[11] resID = 149
```

Descending IDs 160 → 149, and the application fork indeed contains
`CURS 149…160` (plus `CURS 128…131`) and `crsr 149…160`, so both the mono and
colour animation paths resolve. `rAcurID = 128`, `rHandCursorID = 1000`
(`GliderPRO/Sources/AnimCursor.c:14-15`). `InitAnimatedCursor` (`:110-133`) does
`GetResource('acur', 128)`, `HNoPurge`, `MoveHHi`, `HLock`, then
`(*ballCursH)->index = 0`.

### 9.8 The `'PAT#'` resource (ID 128) — marquee patterns

`marquee.pats[kNumMarqueePats]` with `kNumMarqueePats = 7`
(`GliderPRO/Headers/GliderDefines.h:460`), filled at
`GliderPRO/Sources/Marquee.c:504`:

```c
GetIndPattern(&theMarquee.pats[i], kMarqueePatListID, i + 1);   /* 1-based! */
```

`kMarqueePatListID = 128` (`GliderPRO/Sources/Marquee.c:15`); the call is at
`GliderPRO/Sources/Marquee.c:504`, inside `InitMarquee`'s
`for (i = 0; i < kNumMarqueePats; i++)` loop (`:503`).
Format: `short count` then `count` × 8-byte `Pattern`. **Verified: 58 bytes =
2 + 7 × 8** with `count = 7`:

| i | 8 pattern bytes |
|---:|---|
| 0 | `f8 f1 e3 c7 8f 1f 3e 7c` |
| 1 | `3e 7c f8 f1 e3 c7 8f 1f` |
| 2 | `1f 3e 7c f8 f1 e3 c7 8f` |
| 3 | `8f 1f 3e 7c f8 f1 e3 c7` |
| 4 | `c7 8f 1f 3e 7c f8 f1 e3` |
| 5 | `e3 c7 8f 1f 3e 7c f8 f1` |
| 6 | `f1 e3 c7 8f 1f 3e 7c f8` |

These are the classic 8×8 1-bit "barber pole" diagonals — seven of the eight
1-row rotations of a single 45° stripe, giving the crawling marching-ants
selection border. Each entry is the previous one rotated **down 1 row**, *except*
entry 1, which is entry 0 rotated down **2** rows: the down-1 rotation of entry 0
(`7c f8 f1 e3 c7 8f 1f 3e`) is the one rotation missing from the table.
Re-derived byte-for-byte from the resource. `marquee.index`
cycles `0..6` (`GliderPRO/Sources/Marquee.c:44-46`) and
`kHandleSideLong = 9` (`:16`) is the selection-handle size.

A Go port should hard-code this 56-byte table; it is the entire animation.

### 9.9 The house movie side-car

Contrary to what the file naming suggests, `<house>.mov` is a **separate file**,
not the data fork of the house. `OpenHouseMovie`
(`GliderPRO/Sources/HouseIO.c:67-140`, guarded by `#ifdef COMPILEQT`):

1. `if (!thisMac.hasQT) return;`
2. `theSpec = theHousesSpecs[thisHouseIndex];` then
   `PasStringConcat(theSpec.name, "\p.mov");` — **the movie's file name is the
   house file name with `.mov` appended**, in the same directory.
3. `FSpGetFInfo(&theSpec, &finderInfo)`; on any error, silently return (no
   movie).
4. `OpenMovieFile` → `NewMovieFromFile(..., newMovieActive, ...)` →
   `CloseMovieFile`.
5. Allocate a 307200-byte (`= 640 × 480`) `spaceSaver` handle to guarantee
   headroom; bail with `YellowAlert(kYellowQTMovieNotLoaded, 749)` if it fails.
6. `GoToBeginningOfMovie` → `LoadMovieIntoRam(theMovie, 0, GetMovieDuration, 0)`
   → free `spaceSaver`.
7. `PrerollMovie(theMovie, 0, 0x000F0000)` — rate `0x000F0000` is fixed-point
   15.0.
8. `SetTimeBaseFlags(GetMovieTimeBase(theMovie), loopTimeBase)`;
   `SetMovieMasterTimeBase`; `LoopMovie()` which installs an empty `'LOOP'`
   user-data item (`GliderPRO/Sources/HouseIO.c:48-63`).
9. `GetMovieBox(theMovie, &movieRect); hasMovie = true;`

The movie is shown only on a TV object: `Dynamics.c:437-450` draws it when
`thisMac.hasQT && hasMovie && tvInRoom && (who == tvWithMovieNumber)`.

Verified: 15 of the 22 shipped houses have a `.mov` sibling (Art Museum,
Castle o' the Air, CD Demo House, Davis Station, Demo House, Grand Prix,
ImagineHouse PRO II, Land of Illusion, Leviathan, Nemo's Market,
Rainbow's End, Slumberland, SpacePods, Teddy World, Titanic), ranging from
6534 bytes (Titanic) to 119789 bytes (Demo House).

### 9.10 House discovery

`BuildHouseList` (`GliderPRO/Sources/SelectHouse.c:650-664`) →
`DoDirSearch` (`:557-640`):

1. `kMaxDirectories = 32`. Seed `theDirs[0] = thisMac.dirID` (the application's
   own folder), `numDirs = 1`.
2. Breadth-first: for each pending directory, iterate `ioFDirIndex = 1, 2, 3…`
   with `PBGetCatInfo` until it errors.
3. If `(ioFlAttrib & 0x10) == 0x00` (a file) **and** `fdType == 'gliH'` **and**
   `fdCreator == 'ozm5'` **and** `housesFound < maxFiles`, `FSMakeFSSpec` it into
   `theHousesSpecs[housesFound++]`.
4. If `(ioFlAttrib & 0x10) == 0x10` (a directory) and `numDirs < 32`, enqueue it.
5. If `housesFound < 1`: `thisHouseIndex = -1; YellowAlert(kYellowNoHouses, 0);`
6. Else `SortHouseList()`, then select the entry whose name equals
   `thisHouseName` (default index 0), and copy that name back into
   `thisHouseName`. Also record `demoHouseIndex` = the index of `"Demo House"`,
   or -1.

Up to `kMaxExtraHouses = 8` houses passed in by Apple Events
(`AddExtraHouse`, `GliderPRO/Sources/SelectHouse.c:668-675`) are inserted at the
head of the list before the search. Note the whole function is gated on
`thisMac.hasSystem7`; without System 7 no houses are found at all.

`theHousesSpecs` is `maxFiles` × `FSSpec` (70 bytes each), default 48
(`GliderPRO/Sources/StructuresInit2.c`), user-settable 12…500.

### 9.11 The BinHex 4.0 wrapper (distribution format only)

The shipped `Houses/*.binhex` files are BinHex 4.0 — a text transport format, not
something Glider PRO itself reads. Documented here only because it is how the
sample data reaches a modern machine.

Structure:

```
line 1 : "(This file must be converted with BinHex 4.0)"
then   : ':' <base-N body, 64-line-wrapped> ':'
```

The body uses a **custom 64-character alphabet** (not standard base64):

```
!"#$%&'()*+,-012345689@ABCDEFGHIJKLMNPQRSTUVXYZ[`abcdefhijklmpqr
```

6 bits per character. Decode by translating into the standard base64 alphabet and
calling a base64 decoder, then truncating to `len(body) * 6 / 8` bytes.

After the 6-to-8 unpacking, a **run-length pass** applies: byte `0x90` means
"repeat the previous byte", with the *next* byte as the total run length;
`0x90 0x00` is a literal `0x90`.

The decoded stream is:

| Field | Size |
|---|---|
| name length | 1 |
| name | *len* |
| version (always 0) | 1 |
| file type | 4 |
| creator | 4 |
| Finder flags | 2 |
| data fork length | 4 |
| resource fork length | 4 |
| header CRC | 2 |
| **data fork** | *dataLen* |
| data CRC | 2 |
| **resource fork** | *rsrcLen* |
| resource CRC | 2 |

**Verified:**

```
Demo House   name=b'Demo House' type=gliH creator=ozm5 flags=0x0500 dataLen=16526 rsrcLen=491757
Sampler      name=b'Sampler'    type=gliH creator=ozm5 flags=0x0100 dataLen=1564  rsrcLen=286
```

All 22 files decode with `type == 'gliH'` and `creator == 'ozm5'`.

### 9.12 Application resource fork inventory

For completeness, `GliderPRO/Glider PRO.r` (a DeRez text dump, 199843 LF-terminated
lines) contains:

| Type | Count | IDs (first few) |
|---|---:|---|
| `PICT` | 152 | 150, 151, 153, 1000-… |
| `snd ` | 70 | 1000-… |
| `DITL` | 54 | 130, 140, 150, 160, 170, 180, 181, 1000-… |
| `cicn` | 44 | 130, 140, 150, 160, 170, 180, 900, 910, 1000-… |
| `ICON` | 35 | 130, 140, 150, 160, 170, 180, 900, 910, 1000-… |
| `DLOG` | 28 | 150, 1000-… |
| `ALRT` | 26 | 130, 140, 160, 170, 180, 181, 1002-… |
| `dctb` | 20 | 150, 1000-… |
| `CURS` | 16 | 128-131, 149-160 |
| `crsr` | 12 | 149-160 |
| `STR#` | 10 | 128, 129, 140, 150, 160, 170, 171, 1005, 1006, 1007 |
| `ICN#`/`icl4`/`icl8` | 6 each | 128-133 |
| `MENU` | 6 | 128-131, 140, 141 |
| `FREF` | 6 | 128-133 |
| `CNTL` | 5 | 128-132 |
| `mctb` | 5 | 128-132 |
| `ics#`/`ics4`/`ics8` | 4 each | 128, 129, 130, 133 |
| `WDEF` | 2 | 128, 129 |
| `clut` | 2 | 128, 129 |
| `vers` | 2 | 1, 2 |
| `WIND` | 3 | 128, 129, 130 |
| `acur`, `BNDL`, `CDEF`, `cctb`, `demo`, `DLGX`, `ictb`, `PAT#`, `wctb` | 1 each | see §9.6-§9.8 |
| `ozm5` | 1 | ID 0 — the creator signature resource |

`'ozm5' 0` body (32 bytes): `1f a9 20 31 39 39 34 2d …` = the Pascal string
`"© 1994-95 Casady & Greene, Inc."`.

`'vers' 1` decodes as `0x01 0x12 0x80 0x00 0x00 0x00` + `"1.1.2"` +
`"Glider PRO™ 1.1.2\r© 1994-95 Casady & Greene, Inc."`. **The resource fork in
this GPL drop is from build 1.1.2, while `GliderPRO/Sources/Main.c:3` says
`Glider PRO 1.0.4`.** Anything derived from the `.r` file (dialog geometry, PICT
IDs, the demo recording) is therefore 1.1.2-era, not strictly 1.0.4. The
`'demo'`, `'acur'` and `'PAT#'` contents documented above are still consistent
with the 1.0.4 sources that read them.

`kObjectNameStrings = 1007` (`GliderPRO/Headers/GliderDefines.h:461`) is the
`STR#` holding the human-readable object names, indexed by object class.
`kPrefsStringsID = 160` (`GliderPRO/Sources/Prefs.c:22`) holds the Preferences
folder name at index 1.

---

## 10. Blind I/O couplings — the byte-exactness constraints

These are the places where the compiler's struct layout **is** the file (or
resource) layout, with no marshalling step. Getting any of them wrong silently
corrupts data.

| # | Struct | Direction | Call | Citation | Constrains |
|---:|---|---|---|---|---|
| 1 | `houseType` + `roomType[]` | read | `FSRead(houseRefNum, &byteCount, *thisHouse)` with `byteCount = GetEOF(...)` | `GliderPRO/Sources/HouseIO.c:341`, `:372` | **Everything** in §4, §5, §3, §6.1, §6.2 |
| 2 | `houseType` + `roomType[]` | write | `FSWrite(houseRefNum, &byteCount, *thisHouse)` with `byteCount = GetHandleSize(...)` | `GliderPRO/Sources/HouseIO.c:473`, `:491` | Same, plus the 866-vs-868 slop |
| 3 | `roomType` array base | arithmetic | `(GetHandleSize(...) - sizeof(houseType)) / sizeof(roomType)` | `GliderPRO/Sources/HouseLegal.c:630-631` | `sizeof(houseType) == 866`, `sizeof(roomType) == 348` |
| 4 | `roomType` handle growth | arithmetic | `sizeof(houseType) + (sizeof(roomType) * r)` → `SetHandleSize` | `GliderPRO/Sources/HouseLegal.c:767` | Same |
| 5 | `roomType` append | arithmetic | `PtrAndHand((Ptr)thisRoom, (Handle)thisHouse, sizeof(roomType))` | `GliderPRO/Sources/Room.c:200` | `sizeof(roomType) == 348` |
| 6 | `prefsInfo` | write | `byteCount = sizeof(*thePrefs); FSWrite(...)` | `GliderPRO/Sources/Prefs.c:127-129` | All of §7; **explicitly pinned to `mac68k` by the pragma** |
| 7 | `prefsInfo` | read | `byteCount = sizeof(*thePrefs); FSRead(...)` | `GliderPRO/Sources/Prefs.c:193-195` | Same |
| 8 | `scoresType` | write | `byteCount = sizeof(scoresType); FSWrite(..., &((*thisHouse)->highScores))` | `GliderPRO/Sources/HighScores.c:771-776` | §6.1, and `offsetof(houseType, highScores) == 528` |
| 9 | `scoresType` | read | `GetEOF` then `FSRead` into `&((*thisHouse)->highScores)` | `GliderPRO/Sources/HighScores.c:823-841` | Same — **plus the overflow bug of §9.4** |
| 10 | `demoType[1117]` | read | `BlockMove(*tempHandle, demoData, kDemoLength)` | `GliderPRO/Sources/StructuresInit2.c:295` | `sizeof(demoType) == 6` **exactly**; 6702 = 1117 × 6 |
| 11 | `boundsType` | read | `boundsRes = (boundsHand)GetResource('bnds', theID)` then `(*boundsRes)->left` etc. | `GliderPRO/Sources/Room.c:942-960` | `sizeof(boundsType) == 4`, field order `left, top, right, bottom` |
| 12 | `acurRec` | read + mutate | `GetResource('acur', 128)`, then `frame[i].resID` read and `frame[i].cursorHdl` written in place | `GliderPRO/Sources/AnimCursor.c:51-64`, `:116` | Header 4 bytes, stride 4 bytes, resID in the **first** 2 bytes |
| 13 | `game2Type` + `savedRoom[]` | write | `byteCount = sizeof(game2Type) + sizeof(savedRoom) * numRooms` | `GliderPRO/Sources/SavedGames.c:56` | Dead code; would have pinned `sizeof(game2Type) == 110` |

Item 10 is the decisive one: it makes the alignment model observable from shipped
data, and it says `mac68k`. Item 6 is the one Calhoun *knew* about, since it is
the only pragma in the codebase. Items 1-5 are the ones that actually broke —
hence `Sampler`'s two extra bytes.

**Not** a blind coupling: `Pattern` from `'PAT#'` goes through
`GetIndPattern`, which the Toolbox marshals; and the resource fork itself is
parsed by the Resource Manager.

---

## 11. Empirical evidence

Every number in this section was produced by parsing the actual shipped bytes
with `python3`, not by reading the C. The corpus is the 22 BinHex-encoded houses
in `GliderPRO/Houses/*.binhex` (decoded to separate
data and resource forks) plus the derez'ed application resource fork
`GliderPRO/Glider PRO.r`.

**Corpus totals: 22 houses, 4070 rooms, 97 680 object slots (4070 × 24), of which
66 240 are empty (`what == -1`) and 31 440 are live.**

### 11.1 Struct sizes measured by compiling the headers

`/tmp/wf-structs/layout.c` transcribes every `GliderStructs.h` typedef, wraps
them in `#pragma pack(push,2)` (the exact semantics of `align=mac68k`), and
prints `sizeof`/`offsetof`. `/tmp/wf-structs/layout_nat.c` is the same file with
the pragma applied only to `Point`, `Rect`, `FSSpec` and `Pattern` (which the
Toolbox itself packs) so that the remaining structs get PowerPC natural
alignment. Both were compiled with the host `gcc`, with `long` forced to
`int32_t` and pointers modelled as `uint32_t` so the host's 64-bit ABI does not
distort the result. The full 41-row output is the table in §2.2; the persisted
subset is:

| Struct | `mac68k` | PPC natural | Author's `// total` comment | Verdict |
|---|---:|---:|---|---|
| `blowerType` … `clutterType` (all 9) | 10 | 10 | `// total = 10` | matches |
| `objectType` | 12 | 12 | `// total = 12` | matches |
| `scoresType` | 292 | 292 | `// total = 292` | matches |
| `gameType` | 40 | 40 | `// total = 40` | matches |
| `savedRoom` | 292 | 292 | `// total = 292` | matches |
| `roomType` | 348 | 348 | `// total = 348` | matches |
| `houseType` header | **866** | **868** | `// total = 866 +` | matches `mac68k` |
| `game2Type` header | **110** | **112** | (none) | — |
| `prefsInfo` | **226** | 228 | (none) | pragma forces 226 |
| `demoType` | **6** | 8 | (none) | 6702 / 6 = 1117 exactly |

The two that differ are the two that matter, and shipped data resolves both:
`sizeof(demoType)` must be 6 (§11.7) and `offsetof(houseType, rooms)` must be
866 (§11.2).

### 11.2 House data-fork sizes: 866, not 868

For every one of the 22 houses, `nRooms` is read from offset 864 and the
predicted length `866 + 348 × nRooms` is compared with the actual data-fork
length. The full per-house table (with every header field) is §9.1 and is not
repeated here; the result is:

**21 of 22 data forks are exactly `866 + 348·nRooms`. `Sampler` alone is 2 bytes
longer: 1564 = `868 + 348·2`.**

`Sampler` was therefore saved by a build compiled with PowerPC natural
alignment. Parsing its rooms at base **866** succeeds:

```
--- parsing rooms at base 866   (correct)
  room[0] nameLen=8  name='Entrance' bg=2003 floor=1 suite=67 nObj=7 tiles=[0,1,4,1,2,1,1,3]
  room[1] nameLen=7  name='Welcome'  bg=2012 floor=1 suite=66 nObj=4 tiles=[1,1,1,2,3,1,0,1]

--- parsing rooms at base 868   (wrong: shifted by 2)
  room[0] nameLen=110 name='trance Room\x00...' bg=0 floor=67 suite=0 nObj=81
  room[1] nameLen=101 name='lcomed Room\x00...' bg=1 floor=66 suite=0 nObj=57

trailing 2 bytes of the file = 01 01
```

**So even the "PPC-sized" file has its rooms at offset 866** — the two extra
bytes are trailing slop written *past* the end of the room array, because
`sizeof(houseType)` (868) exceeds `offsetof(houseType, rooms)` (866). Under
`mac68k` both are 866 and `FSWrite(…, GetHandleSize(handle))` writes exactly
`866 + 348·n`; under PPC the handle is allocated 868 + 348·n and the last two
bytes are whatever the allocator left there (here `01 01`).

Conclusion for a port: use 866 as the rooms base unconditionally, trust
`nRooms`, and accept `len(data) - 866 - 348·nRooms ∈ {0, 2}`.

### 11.3 House header field census (22 houses)

| Field | Offset | Observed |
|---|---:|---|
| `version` | 0 | `0x0200` in **all 22** — no v1 (`0x0100`) house survives, and no `0x0300` |
| `unusedShort` | 2 | **garbage**: `{0: 12, 60: 1, 147: 1, 196: 1, 222: 1, 259: 2, 2074: 1, 13107: 1, 26228: 1, -30082: 1}` |
| `timeStamp` | 4 | always `>= 0` (bit 31 cleared by the `& 0x7FFFFFFF` mask at `GliderPRO/Sources/HouseIO.c:478`) |
| `flags` | 8 | `{0x00000000: 14, 0x00000002: 7, 0x00000006: 1}` |
| `initial` | 12 | `Point{v, h}` — see §5 |
| `hasGame` | 860 | `{0: 20, 1: 2}` (ImagineHouse PRO II, Titanic) |
| `unusedBoolean` | 861 | **garbage**: `{0: 15, 2: 1, 14: 1, 30: 2, 37: 1, 185: 1, 255: 1}` |

Per-house `flags`, the only houses with a nonzero value:

| `flags` | Houses |
|---|---|
| `0x0002` | CD Demo House, California or Bust!, Davis Station, Land of Illusion, Nemo's Market, Rainbow's End, SpacePods (7) |
| `0x0006` | Art Museum (1) |
| `0x0000` | the remaining 14 |

So `phoneBitSet` (bit 1) is true for 8 houses, `bannerStarCountOn` is *false*
for exactly one (**Art Museum**, bit 2 set — the flag is inverted, §5.2), and
`wardBitSet` (bit 0) is **false in every shipped house**. A port therefore has no
shipped-data coverage of the ward code path at all.

`unusedShort` and `unusedBoolean` being nonzero garbage in 10 and 7 files
respectively is direct proof that Glider PRO writes **uninitialised heap** to
disk. A Go writer that zeroes them produces a byte-different but semantically
identical file; a Go *reader* must ignore them.

### 11.4 Timestamp decode

`timeStamp` is masked with `0x7FFFFFFF` on write, and bit 0 is repurposed as the
house-locked flag. To recover a date you must add 2³¹ back and interpret the
result as seconds since 1904-01-01 (Mac epoch = Unix epoch − 2 082 844 800).

Worked example, `Demo House` data fork:

```
data fork bytes 0..15 : 02 00 00 00 2c 53 b0 41 00 00 00 00 00 6b 00 31
                        ^^^^^ ^^^^^ ^^^^^^^^^^^ ^^^^^^^^^^^ ^^^^^ ^^^^^
                        ver   unused  timeStamp     flags      v     h
                        0x0200  0                    0        107   49

raw      = 0x2C53B041 = 743 682 113
locked   = raw & 1 = 1                            -> LOCKED
true Mac = raw + 0x80000000 = 2 891 165 761
Unix     = 2 891 165 761 - 2 082 844 800 = 808 320 961
date     = 1995-08-13 13:36:01 UTC
```

Whole-corpus decode:

| House | raw `timeStamp` | locked | + 2³¹ | decoded date |
|---|---:|:--:|---:|---|
| The Asylum Pro | 738 016 929 | 1 | 2 885 500 577 | 1995-06-08 |
| ImagineHouse PRO II | 740 149 565 | 1 | 2 887 633 213 | 1995-07-03 |
| Nemo's Market | 741 121 233 | 1 | 2 888 604 881 | 1995-07-14 |
| Rainbow's End | 741 121 489 | 1 | 2 888 605 137 | 1995-07-14 |
| SpacePods | 741 122 073 | 1 | 2 888 605 721 | 1995-07-14 |
| Empty House | 741 421 292 | 0 | 2 888 904 940 | 1995-07-18 |
| Grand Prix | 741 444 157 | 1 | 2 888 927 805 | 1995-07-18 |
| In The Mirror | 741 512 193 | 1 | 2 888 995 841 | 1995-07-19 |
| Leviathan | 741 514 803 | 1 | 2 888 998 451 | 1995-07-19 |
| Castle o' the Air | 741 536 868 | 0 | 2 889 020 516 | 1995-07-19 |
| Metropolis | 741 642 315 | 1 | 2 889 125 963 | 1995-07-20 |
| CD Demo House | 742 294 873 | 1 | 2 889 778 521 | 1995-07-28 |
| Fun House | 742 681 616 | 0 | 2 890 165 264 | 1995-08-01 |
| Davis Station | 743 681 789 | 1 | 2 891 165 437 | 1995-08-13 |
| Demo House | 743 682 113 | 1 | 2 891 165 761 | 1995-08-13 |
| Slumberland | 743 682 377 | 1 | 2 891 166 025 | 1995-08-13 |
| Teddy World | 744 401 697 | 1 | 2 891 885 345 | 1995-08-21 |
| Land of Illusion | 745 268 052 | 0 | 2 892 751 700 | 1995-08-31 |
| California or Bust! | 745 882 296 | 0 | 2 893 365 944 | 1995-09-08 |
| Art Museum | 746 622 005 | 1 | 2 894 105 653 | 1995-09-16 |
| Titanic | 755 123 966 | 0 | 2 902 607 614 | 1995-12-23 |
| **Sampler** | **893 435 880** | 0 | 3 040 919 528 | **2000-05-11** |

21 houses land in 1995-06-08 … 1995-12-23; `Sampler` lands in 2000, consistent
with it being the PowerPC-built odd one out of §11.2. **15 of 22 have bit 0 set
(locked).**

Every raw value is < 2³¹ (the mask guarantees it) and they all fall in
738 016 929 … 893 435 880, so naively treating the field as a Mac timestamp
without re-adding 2³¹ yields dates in **1927–1932** — plausible-looking but
wrong. That is the trap.

`gameType.timeStamp` (house offset 824) and `scoresType.timeStamps[]` (house
offset 760) are **not** masked; `Demo House`'s single high-score timestamp is
`2 888 823 855` with bit 31 set, i.e. a raw `GetDateTime` value.

### 11.5 `roomType` field census (4070 rooms)

| Field | Offset | Observed distribution |
|---|---:|---|
| `name` | 0 | `Str27`; length byte 0…27; tail after the string is uninitialised garbage |
| `bounds` | 28 | `{0: 2401, 1: 167, 3: 30, 5: 9, 7: 18, 9: 29, 11: 63, 13: 25, 15: 93, 17: 13, 19: 29, 21: 115, 23: 22, 25: 110, 27: 101, 29: 10, 31: 542, 33: 47, 35: 36, 37: 8, 39: 14, 41: 26, 43: 39, 45: 16, 47: 51, 49: 1, 53: 1, 55: 5, 57: 2, 59: 23, 61: 4, 63: 20}` |
| `leftStart` | 30 | 112 distinct values, 0…255 |
| `rightStart` | 31 | 115 distinct values, 0…255 |
| `unusedByte` | 32 | `{0: 4070}` — always zero, unlike the house-level unused fields |
| `visited` | 33 | `{0: 3634, 1: 436}` — genuinely persisted |
| `background` | 34 | 135 distinct IDs, see §11.6 |
| `tiles[8]` | 36 | values only 0…7: `{0: 5114, 1: 7036, 2: 6395, 3: 3229, 4: 3925, 5: 2263, 6: 2161, 7: 2437}` |
| `floor` | 52 | −7 … 39 (46 distinct) |
| `suite` | 54 | 0 … 127 (127 distinct) — **never −1**, so every shipped house is fully compressed |
| `openings` | 56 | `{0: 4070}` — dead field, written 0 and never read |
| `numObjects` | 58 | 0…24; `{0: 840, 1: 588, 2: 132, 3: 150, 4: 124, 5: 114, 6: 140, 7: 155, 8: 164, 9: 154, 10: 147, 11: 146, 12: 161, 13: 116, 14: 114, 15: 127, 16: 85, 17: 81, 18: 59, 19: 50, 20: 39, 21: 49, 22: 53, 23: 60, 24: 222}` |

Two structural invariants hold with **zero** exceptions across 4070 rooms:

1. `numObjects` always equals the number of slots with `what != -1`.
2. `bounds` is either 0 or **odd** — never even-and-nonzero. Bit 0 is the
   "field is populated" flag of §4.2, so an even nonzero value would be
   meaningless, and the editor never produces one.

`tiles[]` never exceeding 7 confirms the field is a **column index** into a
64-pixel-wide strip of a 512-pixel-wide background PICT (8 columns), not a pixel
offset. 591 of the 4070 rooms use the identity mapping `[0,1,2,3,4,5,6,7]`.

### 11.6 Background PICT ID distribution (4070 rooms, 135 distinct IDs)

Built-in backgrounds (`kBaseBackgroundID = 2000` … 2017, i.e. all 18 of
`kNumBackgrounds`):

| ID | Rooms | ID | Rooms | ID | Rooms |
|---:|---:|---:|---:|---:|---:|
| 2000 | 171 | 2006 | 49 | 2012 | 97 |
| 2001 | 105 | 2007 | 23 | 2013 | 25 |
| 2002 | 103 | 2008 | 10 | 2014 | 181 |
| 2003 | 49 | 2009 | 14 | 2015 | 595 |
| 2004 | 40 | 2010 | 28 | 2016 | 62 |
| 2005 | 54 | 2011 | 421 | 2017 | 246 |

Custom ranges:

| Range | Meaning | Distinct IDs | Rooms |
|---|---|---:|---:|
| 2000 – 2017 | built-in (`kBaseBackgroundID`…, `kNumBackgrounds = 18`) | 18 | 2273 |
| 3000 – 3299 | user background (`kUserBackground = 3000`) | 88 | 1096 |
| 3300 – 3799 | structure room (`kUserStructureRange = 3300`) | 29 | 701 |

Observed min/max in the custom space is 3000 … 3353. **No ID falls outside
2000–2017 ∪ 3000–3353** — nothing in 2018–2999, nothing ≥ 3354. All 18 built-ins
are used by at least 10 rooms, so a port must ship all 18.

### 11.7 The `'demo'` resource

`'demo' 128` in the application fork is **6702 bytes = `kDemoLength`
(`GliderPRO/Headers/GliderDefines.h:625`) = 1117 × 6 exactly**. Parsed as
`demoType { long frame; char key; char padding; }`:

```
frame=46    key=0 pad=0x72   raw = 00 00 00 2e 00 72
frame=47    key=0 pad=0x72   raw = 00 00 00 2f 00 72
frame=48    key=0 pad=0x72   raw = 00 00 00 30 00 72
frame=49    key=0 pad=0x45   raw = 00 00 00 31 00 45
frame=56    key=0 pad=0x06   raw = 00 00 00 38 00 06
frame=57    key=0 pad=0x72   raw = 00 00 00 39 00 72
frame=58    key=0 pad=0x6F   raw = 00 00 00 3a 00 6f
frame=59    key=0 pad=0x72   raw = 00 00 00 3b 00 72
```

- `frame` is a big-endian `long`, monotonically non-decreasing, first = 46, last
  = 3414.
- `key` histogram: `{0: 910, 1: 198, 3: 9}`. Per `GliderPRO/Sources/Input.c`
  these are left (0), right (1), and 3 = both/battery-band composite; **key 2 is
  never used** in the shipped demo.
- `padding` is uninitialised: 109 distinct byte values, most common
  `{0x00: 407, 0xFF: 42, 0x20: 39, 0x72: 33, 0x6F: 24, 0x65: 24}`. Those look
  like fragments of ASCII text left in the heap.

This is the load-bearing proof of the alignment model: under PowerPC natural
alignment `sizeof(demoType)` would be 8 (the `long` forces 4-byte alignment and
the struct is padded to 8), 6702 is not divisible by 8, and
`BlockMove(*tempHandle, demoData, kDemoLength)`
(`GliderPRO/Sources/StructuresInit2.c:295`) would desynchronise on the second
record. **The shipping build used `mac68k` alignment globally.**

If `key` were at offset 5 instead of 4, the histogram would be the 109-value
garbage set instead of `{0, 1, 3}` — so the field order in the struct is
confirmed as `frame, key, padding`.

### 11.8 The `'acur' 128` resource

52 bytes = 4-byte header + 12 × 4-byte frame records:

```
short n     = 12       // number of frames
short index = 0        // current frame (mutated at run time)
frame[ 0] resID=160 fill=0      frame[ 6] resID=154 fill=0
frame[ 1] resID=159 fill=0      frame[ 7] resID=153 fill=0
frame[ 2] resID=158 fill=0      frame[ 8] resID=152 fill=0
frame[ 3] resID=157 fill=0      frame[ 9] resID=151 fill=0
frame[ 4] resID=156 fill=0      frame[10] resID=150 fill=0
frame[ 5] resID=155 fill=0      frame[11] resID=149 fill=0
```

`resID` is in the **first** two bytes of each record and the `fill`/`cursorHdl`
slot is in the second two; the code reads `(*theAcur)->frame[i].resID` and then
*overwrites the same 4-byte record's second half* with a `CursHandle`
(`GliderPRO/Sources/AnimCursor.c:51-64`, `:116`). IDs run 160 → 149 descending,
matching the app fork's `CURS` 149–160 and `crsr` 149–160. Note the app also has
`CURS` 128–131 (the static cursors) which are **not** part of the animation.

### 11.9 The `'PAT#' 128` resource

58 bytes = `short n = 7` + 7 × 8-byte `Pattern` (`kNumMarqueePats = 7`,
`GliderPRO/Headers/GliderDefines.h:460`). All seven, with the 8×8 bitmap
rendered (`#` = 1 = black):

| i | Bytes | Bitmap |
|--:|---|---|
| 0 | `f8 f1 e3 c7 8f 1f 3e 7c` | `#####...` `####...#` `###...##` `##...###` `#...####` `...#####` `..#####.` `.#####..` |
| 1 | `3e 7c f8 f1 e3 c7 8f 1f` | `..#####.` `.#####..` `#####...` `####...#` `###...##` `##...###` `#...####` `...#####` |
| 2 | `1f 3e 7c f8 f1 e3 c7 8f` | `...#####` `..#####.` `.#####..` `#####...` `####...#` `###...##` `##...###` `#...####` |
| 3 | `8f 1f 3e 7c f8 f1 e3 c7` | `#...####` `...#####` `..#####.` `.#####..` `#####...` `####...#` `###...##` `##...###` |
| 4 | `c7 8f 1f 3e 7c f8 f1 e3` | `##...###` `#...####` `...#####` `..#####.` `.#####..` `#####...` `####...#` `###...##` |
| 5 | `e3 c7 8f 1f 3e 7c f8 f1` | `###...##` `##...###` `#...####` `...#####` `..#####.` `.#####..` `#####...` `####...#` |
| 6 | `f1 e3 c7 8f 1f 3e 7c f8` | `####...#` `###...##` `##...###` `#...####` `...#####` `..#####.` `.#####..` `#####...` |

They are a single 45° barber-pole rotated by one scanline per step — cycling
`index` 0→6 animates the marching-ants marquee. Note the sequence is **not** a
uniform rotate-by-1: six of the seven transitions (1→2, 2→3, 3→4, 4→5, 5→6, and
the 6→0 wrap) are a 1-row downward rotation, but 0→1 is a **2-row** rotation,
because the down-1 rotation of pattern 0 is not in the list at all. Equivalently,
pattern *i* is pattern 0 rotated down `i+1` rows for `i >= 1`, and pattern 0 is
the unrotated stripe.

### 11.10 `Sampler.rsrc` is a canonical empty resource fork

`Sampler`'s resource fork is exactly **286 bytes** and contains zero resources.
Field by field:

```
offset  0 : dataOff     = 0x00000100 = 256
offset  4 : mapOff      = 0x00000100 = 256
offset  8 : dataLen     = 0x00000000 = 0
offset 12 : mapLen      = 0x0000001E = 30       (256 + 30 = 286 = file length)
offset 16..255 : system-reserved 240 bytes — NOT zero (73 non-zero bytes,
                 e.g. bytes 16..31 = 00 00 00 45 6f 03 04 00 00 00 10 00 00 00 04 80)
offset 256 : map, 30 bytes = 00 00 01 00 00 00 01 00 00 00 00 00 00 00 00 1e
                             00 00 00 00 00 00 00 00 00 1c 00 1e ff ff
             (first 16 bytes = copy of the header, then nextMap=0, refNum=0,
              attrs=0x0000, typeListOff=28, nameListOff=30)
offset 284 : type count = 0xFFFF
```

The type count is stored as **count − 1**, so `0xFFFF` means *zero types*, not
65 536. A naive parser that computes `nTypes = raw + 1` reads 0 as 1 and then
walks off the end of the file. This bit me while writing `/tmp/wf-structs/rsrc.py`
and it will bite a Go port too.

The 240 reserved bytes being non-zero is more evidence of uninitialised memory
reaching disk (this time from the Resource Manager, not Glider PRO).

### 11.11 Object class census (31 440 live objects)

**All 117 object classes defined in `GliderPRO/Headers/GliderDefines.h:311-435`
appear in the shipped corpus** — there is no dead class. Codes are not
contiguous: the nine class bands are `0x01-0x10`, `0x11-0x20`,
`0x21-0x30`, `0x31-0x40`, `0x41-0x50`, `0x51-0x60`, `0x61-0x70`, `0x71-0x80` and
`0x81-0x8F` — eight of 16 codes each plus a final band of 15, because the highest
class defined is `kChimes = 0x8F` and `kNumSrcRects = 0x90` is the exclusive
upper bound. The top of each band is often unused (`0x20`, `0x30`, `0x50`,
`0x60`, `0x70`, `0x80` never occur), which is exactly why the union selector in
§3.3 must be a range test and not a table index.

| code | name | count | code | name | count | code | name | count |
|---:|---|---:|---:|---|---:|---:|---|---:|
| 0x01 | `kFloorVent` | 1458 | 0x29 | `kGreaseLf` | 143 | 0x58 | `kInvisLight` | 1764 |
| 0x02 | `kCeilingVent` | 28 | 0x2A | `kFoil` | 83 | 0x61 | `kShredder` | 50 |
| 0x03 | `kFloorBlower` | 83 | 0x2B | `kInvisBonus` | 327 | 0x62 | `kToaster` | 140 |
| 0x04 | `kCeilingBlower` | 12 | 0x2C | `kStar` | 69 | 0x63 | `kMacPlus` | 74 |
| 0x05 | `kSewerGrate` | 507 | 0x2D | `kSparkle` | 486 | 0x64 | `kGuitar` | 29 |
| 0x06 | `kLeftFan` | 45 | 0x2E | `kHelium` | 50 | 0x65 | `kTV` | 80 |
| 0x07 | `kRightFan` | 54 | 0x2F | `kSlider` | 354 | 0x66 | `kCoffee` | 38 |
| 0x08 | `kTaper` | 90 | 0x31 | `kUpStairs` | 163 | 0x67 | `kOutlet` | 100 |
| 0x09 | `kCandle` | 180 | 0x32 | `kDownStairs` | 163 | 0x68 | `kVCR` | 26 |
| 0x0A | `kStubby` | 127 | 0x33 | `kMailboxLf` | 44 | 0x69 | `kStereo` | 36 |
| 0x0B | `kTiki` | 58 | 0x34 | `kMailboxRt` | 34 | 0x6A | `kMicrowave` | 56 |
| 0x0C | `kBBQ` | 43 | 0x35 | `kFloorTrans` | 244 | 0x6B | `kCinderBlock` | 43 |
| 0x0D | `kInvisBlower` | 2336 | 0x36 | `kCeilingTrans` | 465 | 0x6C | `kFlowerBox` | 35 |
| 0x0E | `kGrecoVent` | 116 | 0x37 | `kDoorInLf` | 23 | 0x6D | `kCDs` | 63 |
| 0x0F | `kSewerBlower` | 191 | 0x38 | `kDoorInRt` | 11 | 0x6E | `kCustomPict` | 4782 |
| 0x10 | `kLiftArea` | 716 | 0x39 | `kDoorExRt` | 21 | 0x71 | `kBalloon` | 500 |
| 0x11 | `kTable` | 170 | 0x3A | `kDoorExLf` | 11 | 0x72 | `kCopterLf` | 259 |
| 0x12 | `kShelf` | 389 | 0x3B | `kWindowInLf` | 21 | 0x73 | `kCopterRt` | 155 |
| 0x13 | `kCabinet` | 457 | 0x3C | `kWindowInRt` | 32 | 0x74 | `kDartLf` | 151 |
| 0x14 | `kFilingCabinet` | 107 | 0x3D | `kWindowExRt` | 19 | 0x75 | `kDartRt` | 117 |
| 0x15 | `kWasteBasket` | 103 | 0x3E | `kWindowExLf` | 29 | 0x76 | `kBall` | 212 |
| 0x16 | `kMilkCrate` | 252 | 0x3F | `kInvisTrans` | 385 | 0x77 | `kDrip` | 477 |
| 0x17 | `kCounter` | 287 | 0x40 | `kDeluxeTrans` | 61 | 0x78 | `kFish` | 120 |
| 0x18 | `kDresser` | 122 | 0x41 | `kLightSwitch` | 113 | 0x79 | `kCobweb` | 86 |
| 0x19 | `kDeckTable` | 30 | 0x42 | `kMachineSwitch` | 79 | 0x81 | `kOzma` | 77 |
| 0x1A | `kStool` | 91 | 0x43 | `kThermostat` | 108 | 0x82 | `kMirror` | 667 |
| 0x1B | `kTrunk` | 101 | 0x44 | `kPowerSwitch` | 78 | 0x83 | `kMousehole` | 174 |
| 0x1C | `kInvisObstacle` | 666 | 0x45 | `kKnifeSwitch` | 230 | 0x84 | `kFireplace` | 33 |
| 0x1D | `kManhole` | 40 | 0x46 | `kInvisSwitch` | 635 | 0x85 | `kFlower` | 547 |
| 0x1E | `kBooks` | 210 | 0x47 | `kTrigger` | 239 | 0x86 | `kWallWindow` | 195 |
| 0x1F | `kInvisBounce` | 1670 | 0x48 | `kLgTrigger` | 81 | 0x87 | `kBear` | 169 |
| 0x21 | `kRedClock` | 140 | 0x49 | `kSoundTrigger` | 122 | 0x88 | `kCalendar` | 58 |
| 0x22 | `kBlueClock` | 163 | 0x51 | `kCeilingLight` | 160 | 0x89 | `kVase1` | 72 |
| 0x23 | `kYellowClock` | 330 | 0x52 | `kLightBulb` | 252 | 0x8A | `kVase2` | 86 |
| 0x24 | `kCuckoo` | 100 | 0x53 | `kTableLamp` | 70 | 0x8B | `kBulletin` | 38 |
| 0x25 | `kPaper` | 283 | 0x54 | `kHipLamp` | 26 | 0x8C | `kCloud` | 1860 |
| 0x26 | `kBattery` | 119 | 0x55 | `kDecoLamp` | 62 | 0x8D | `kFaucet` | 28 |
| 0x27 | `kBands` | 150 | 0x56 | `kFlourescent` | 115 | 0x8E | `kRug` | 81 |
| 0x28 | `kGreaseRt` | 195 | 0x57 | `kTrackLight` | 81 | 0x8F | `kChimes` | 54 |

The ten commonest classes account for 21 407 of the 31 440 live objects (68 %):

| Rank | Code | Class | Count | Union |
|---:|---|---|---:|---|
| 1 | 0x6E | `kCustomPict` | 4782 | `applianceType` |
| 2 | 0x0D | `kInvisBlower` | 2336 | `blowerType` |
| 3 | 0x8C | `kCloud` | 1860 | `clutterType` |
| 4 | 0x58 | `kInvisLight` | 1764 | `lightType` |
| 5 | 0x1F | `kInvisBounce` | 1670 | `furnitureType` |
| 6 | 0x01 | `kFloorVent` | 1458 | `blowerType` |
| 7 | 0x82 | `kMirror` | 667 | `clutterType` |
| 8 | 0x1C | `kInvisObstacle` | 666 | `furnitureType` |
| 9 | 0x46 | `kInvisSwitch` | 635 | `switchType` |
| 10 | 0x05 | `kSewerGrate` | 507 | `blowerType` |

`what == 0` (`kRug`'s numeric neighbour, and the value a zeroed slot would have)
**never occurs** in any of the 97 680 slots. The empty sentinel is `-1`
(`kObjectIsEmpty`, `GliderPRO/Headers/GliderDefines.h:526`), *not* 0. A Go loader
that treats 0 as empty will silently drop `kFloorVent`-adjacent data if the
sentinel is ever mis-set, and worse, a Go *writer* that zeroes unused slots
produces files the original will read as 24 `what == 0` objects — which
`KeepObjectLegal` does not recognise.

### 11.12 Per-union field census

Objects were bucketed by the §3.3 range rule and each union's fields
histogrammed. `n` is the number of live objects in that bucket.

#### `blowerType` (classes 0x01–0x10, n = 6044)

| Field | Offset in payload | Observed |
|---|---:|---|
| `topLeft` | 0 | `Point{v, h}` |
| `distance` | 4 | 389 distinct, 0 … 512 |
| `initial` | 6 | `{0: 270, 1: 5771, 23: 3}` |
| `state` | 7 | `{0: 261, 1: 5780, 23: 3}` |
| `vector` | 8 | `{1: 4900, 2: 412, 4: 326, 8: 402, 17: 3, 20: 1}` |
| `tall` | 9 | 132 distinct, 0 … 161 |

Three observations that matter for a port:

1. `initial`/`state` are declared `Boolean` but **three objects hold the value
   23**, so they are not safely truthy/falsy-only. Anything comparing `== 1` will
   diverge from anything comparing `!= 0`. The C code does `if (theObj->initial)`
   (truthiness), so a Go port must model them as `uint8` and test `!= 0`, not as
   `bool` decoded from `b == 1`.
2. `vector` holds **17 (0x11) and 20 (0x14)** in four objects, i.e. junk in the
   high nibble. `ObjectRects.c:523-560` masks with `& 0x0F`, turning them into 1
   and 4. **A port must apply that mask**; without it a switch on `vector` falls
   through and the blower silently does nothing.
3. `tall` for `kLiftArea` (0x10) is a genuine **height in pixels** (1 … 161), not
   a boolean or an index, despite the `Byte` type.

#### `furnitureType` (classes 0x11–0x20, n = 4695)

| Field | Offset | Observed |
|---|---:|---|
| `bounds` | 0 | `Rect{top, left, bottom, right}` |
| `pict` | 8 | `{0: 4695}` — **always zero** |

`furnitureType.pict` is dead in shipped data: every furniture object uses the
built-in art for its class. A port can ignore the field but must still reserve
the 2 bytes.

#### `bonusType` (classes 0x21–0x30, n = 2992)

| Field | Offset | Observed |
|---|---:|---|
| `topLeft` | 0 | `Point` |
| `length` | 4 | 164 distinct, 0 … 512 |
| `points` | 6 | `{0: 2663, 85: 1, 100: 136, 145: 1, 300: 27, 500: 164}` |
| `state` | 8 | `{0: 451, 1: 2541}` |
| `initial` | 9 | `{0: 104, 1: 2888}` |

`points` is non-zero only for `kInvisBonus` (0x2B) with the three round values
100 / 300 / 500 — **plus two anomalies**, both `kStar` (0x2C) objects in
`Slumberland`:

```
Slumberland room 61,  slot 6 : what=0x2C  data = 00 3b 01 10 00 04 00 55 01 01
                                                            ^^^^^ points = 0x0055 = 85
Slumberland room 104, slot 5 : what=0x2C  data = 00 62 00 6e 00 04 00 91 01 01
                                                            ^^^^^ points = 0x0091 = 145
```

`kStar` scoring is hard-coded in the play code and never reads `points`, so these
85/145 values are leftover editor junk with no effect. But they prove `points`
must be read as a plain `short` and not validated against `{0,100,300,500}`.

Note also that `state` and `initial` are in the **opposite order** to every other
union (`state` at 8, `initial` at 9); in `blowerType`, `lightType`,
`applianceType` and `enemyType` it is `initial` then `state`. Getting this
backwards is a silent 1-byte swap that changes which bonuses start collected.

#### `transportType` (classes 0x31–0x40, n = 1726)

| Field | Offset | Observed |
|---|---:|---|
| `topLeft` | 0 | `Point` |
| `tall` | 4 | 125 distinct, **−32 722 … 32 591** |
| `where` | 6 | 325 distinct, **−100 … 12 523** |
| `who` | 8 | 26 distinct, 0 … 255 |
| `wide` | 9 | 55 distinct, 0 … 127 |

`tall`/`wide` are **zero for every class except `kInvisTrans` (0x3F) and
`kDeluxeTrans` (0x40)**:

| Class | `tall` | `wide` |
|---|---|---|
| `kInvisTrans` | 79 distinct, 21 … 322 | 55 distinct, 0 … 127 |
| `kDeluxeTrans` | 45 distinct, **−32 722 … 32 591** | 3 distinct, 0 … 17 |
| all others (stairs, doors, windows, mailboxes, floor/ceiling transports) | 0 | 0 |

`kDeluxeTrans`'s negative `tall` is not corruption. The editor packs *two*
dimensions into the one `short` — on drag-resize in
`GliderPRO/Sources/ObjectEdit.c:281-289` and again when
`KeepObjectLegal` re-normalises the object
(`GliderPRO/Sources/HouseLegal.c:304-308`):

```
tall = ((wide / 4) << 8) + (tall / 4)
```

so a `wide` of 512 gives `(128 << 8) = 0x8000` and the `short` goes negative.
Observed low value −32 722 = `0x802E` decodes as `wide/4 = 0x80 = 128` (→ 512 px)
and `tall/4 = 0x2E = 46` (→ 184 px). **A Go port must read this field as
`uint16`**, not `int16`, before unpacking.

#### `switchType` (classes 0x41–0x50, n = 1685)

| Field | Offset | Observed |
|---|---:|---|
| `topLeft` | 0 | `Point` |
| `delay` | 4 | 51 distinct, 0 … 120 |
| `where` | 6 | 414 distinct, **−100 … 12 531** |
| `who` | 8 | 25 distinct, 0 … 255 |
| `type` | 9 | `{0: 609, 1: 393, 2: 363, 3: 320}` |

`type` is a clean 4-valued enum (toggle / force-on / force-off / delayed per
`ObjectAdd.c`). `delay` ≤ 120 (two seconds at 60 Hz).

#### `lightType` (classes 0x51–0x60, n = 2530)

| Field | Offset | Observed |
|---|---:|---|
| `topLeft` | 0 | `Point` |
| `length` | 4 | 123 distinct, 0 … 512 |
| `byte0` | 6 | `{0: 2530}` — dead |
| `byte1` | 7 | `{0: 2530}` — dead |
| `initial` | 8 | `{0: 99, 1: 2431}` |
| `state` | 9 | `{0: 89, 1: 2441}` |

#### `applianceType` (classes 0x61–0x70, n = 5552)

| Field | Offset | Observed |
|---|---:|---|
| `topLeft` | 0 | `Point` |
| `height` | 4 | 231 distinct, **0 … 12 351** |
| `byte0` | 6 | `{0: 5502, 1: 3, 2: 9, 3: 2, 4: 3, 6: 11, 7: 22}` |
| `delay` | 7 | 32 distinct, 0 … 255 |
| `initial` | 8 | `{0: 171, 1: 5381}` |
| `state` | 9 | `{0: 193, 1: 5359}` |

`height` is heavily overloaded and this is the most dangerous field in the file
format:

| Class | `height` meaning | Observed range |
|---|---|---|
| `kToaster` (0x62) | genuine pixel height of the toast's flight | 81 distinct, **9 … 277** |
| `kCustomPict` (0x6E) | **a PICT resource ID**, not a height | 149 distinct, **10 000 … 12 351** (none below 10 000) |
| others | 0 or a small height | — |

`byte0` is non-zero **only for `kMicrowave` (0x6A)**, where it selects which of
eight food sprites is inside: `{0: 6, 1: 3, 2: 9, 3: 2, 4: 3, 6: 11, 7: 22}` —
note **value 5 never occurs**, so a port cannot infer the count from data alone
(it is 8 in the code).

#### `enemyType` (classes 0x71–0x80, n = 2077)

| Field | Offset | Observed |
|---|---:|---|
| `topLeft` | 0 | `Point` |
| `length` | 4 | 241 distinct, 0 … 310 |
| `delay` | 6 | 67 distinct, 0 … 255 |
| `byte0` | 7 | `{0: 2077}` — dead |
| `initial` | 8 | `{0: 85, 1: 1992}` |
| `state` | 9 | `{0: 89, 1: 1988}` |

#### `clutterType` (classes 0x81–0x8F, n = 4139)

| Field | Offset | Observed |
|---|---:|---|
| `bounds` | 0 | `Rect` |
| `pict` | 8 | `{0: 3676, 1: 83, 2: 56, 3: 81, 4: 117, 5: 126}` |

`pict` is non-zero **only for `kFlower` (0x85)**, where it is a 0-based variant
index 0 … 5 matching `kNumFlowers = 6`
(`GliderPRO/Headers/GliderDefines.h:458`). All 6 values occur
(`{0: 84, 1: 83, 2: 56, 3: 81, 4: 117, 5: 126}`). For every other clutter class
`pict` is 0.

### 11.13 Link encoding: full classification

Link-capable classes are the eight switches (0x41–0x48) and the six
"transport-like" classes that `CountHouseLinks`
(`GliderPRO/Sources/House.c:258-307`) walks: `kMailboxLf` (0x33),
`kMailboxRt` (0x34), `kFloorTrans` (0x35), `kCeilingTrans` (0x36),
`kInvisTrans` (0x3F), `kDeluxeTrans` (0x40). `kSoundTrigger` (0x49) is
deliberately **excluded** — it reuses `where` as a sound ID, not a room number
(see below).

```
link-capable objects (14 classes)              : 2796
  where == -1    (explicitly unlinked)         :  322    (all have who == 255)
  where == -100  (destination room deleted)    :  165    (all have who == 255)
  where resolves to a live (floor,suite),
        who in [0,23]                          : 2272
  where does not resolve to any room           :   24
  where resolves but who outside [0,23]        :   13
  --------------------------------------------------
  what CountHouseLinks reports (where != -1)   : 2474
```

`where` is decoded by `ExtractFloorSuite` (`GliderPRO/Sources/Link.c:34-53`):

```c
suite = where / 100;                    /* C truncation toward zero */
floor = where - (suite * 100) - 8;      /* the 8-underground-floor bias */
```

**`where == -100` therefore decodes to `suite = -1`, i.e. `kRoomIsEmpty`.** That
is not an error value the code writes deliberately: it is what
`MergeFloorSuite(floor, suite)` produces when the *destination room record* has
already been marked deleted (`suite == -1`) and then re-serialised. 165 objects
are in this state, spread as:

| Class | Count |
|---|---:|
| `kCeilingTrans` | 110 |
| `kInvisTrans` | 38 |
| `kFloorTrans` | 12 |
| `kMailboxRt` | 2 |
| `kThermostat` | 1 |
| `kKnifeSwitch` | 1 |
| `kLightSwitch` | 1 |

Both `-1` and `-100` always pair with `who == 255` (0xFF), which is the "no
object" marker. So a robust port should treat **`where < 0`** as "unlinked",
which subsumes both. `CountHouseLinks` only tests `!= -1`, so its 2474 figure
over-counts by 165 + 24 + 13 = 202 dangling links.

The 24 unresolvable `where` values, enumerated:

| House | Class | `where` | decodes to (floor, suite) |
|---|---|---:|---|
| ImagineHouse PRO II | `kMailboxRt` | 971 | (63, 9) |
| Leviathan | `kTrigger` ×15 | 960 | (52, 9) |
| Leviathan | `kLightSwitch` | 953 | (45, 9) |
| Leviathan | `kInvisTrans` | 554 | (46, 5) |
| Leviathan | `kTrigger` | 1479 | (71, 14) |
| Leviathan | `kCeilingTrans` | 1375 | (67, 13) |
| Leviathan | `kTrigger` ×4 | 879 | (71, 8) |

All 24 are in just two houses: 23 in `Leviathan` and the single `kMailboxRt` in
`ImagineHouse PRO II` (1 + 15 + 1 + 1 + 1 + 1 + 4 = 24). The floors
(45 … 71) exceed the highest real floor in the corpus (39), so these are stale
links to rooms that were deleted *and* compressed away.

The 13 "resolves but bad `who`" cases are all in `CD Demo House`:

| Class | `where` | `who` |
|---|---:|---:|
| `kInvisTrans` | 1410 | 255 |
| **`kMailboxRt`** | **2109** | **35** |
| `kCeilingTrans` | 3210 | 255 |
| `kInvisTrans` | 4509 | 255 |
| `kLightSwitch` | 4809 | 255 |
| `kMailboxLf` | 5109 | 255 |
| `kCeilingTrans` | 5209 | 255 |
| `kInvisTrans` | 5010 | 255 |
| `kInvisTrans` | 5107 | 255 |
| `kCeilingTrans` | 720 | 255 |
| `kCeilingTrans` | 1121 | 255 |
| `kCeilingTrans` | 1120 | 255 |
| `kCeilingTrans` | 920 | 255 |

Twelve are `who == 255` with a valid room (a room-level link with no object
target — probably a hand-edited half-link), and **one is `who == 35`**, which is
past `kMaxRoomObs - 1 = 23`. A Go port must bounds-check `who` before indexing
`rooms[r].objects[who]` or it will read out of the room struct.

**Stairs, doors and windows never store a link**: all 493 objects in
`kUpStairs`, `kDownStairs`, `kDoorInLf`/`Rt`, `kDoorExLf`/`Rt`,
`kWindowInLf`/`Rt`, `kWindowExLf`/`Rt` have `where == -1` (histogram
`{-1: 493}`). Their destination is computed at run time from the room geometry
(`GliderPRO/Sources/Transit.c:176/185/208/217`), never stored.

`kSoundTrigger` (0x49, 122 objects) reuses `where` as a **sound resource ID**:
`{3000: 35, 3001: 18, 3002: 14, 3003: 6, 3004: 6, 3005: 5, 3006: 13, 3007: 2,
3008: 2, 3009: 1, 3010: 1, 3011: 2, 3012: 1, 3037: 1, 3042: 6, 3043: 1,
3044: 3, 3045: 1, 3046: 1, 3058: 1, 10000: 2}`. It is **never −1**, so if it were
included in the link walk every one of the 122 would be mis-read as a link.
Decoding them with `suite = where / 100`, `floor = where % 100 - 8` gives
suite 30 with floors −8 (for 3000) through 50 (for 3058), plus suite 100 /
floor −8 for the two 10000s — i.e. deep-underground floors in a suite number no
house uses. This is why `CountHouseLinks` excludes it, and why a port must too.

### 11.14 Evidence of uninitialised memory on disk

Glider PRO never zero-fills before writing, so garbage from the Mac heap is part
of every file. Four independent demonstrations:

1. **Pascal-string tails.** `Demo House` room 0's 28-byte `Str27` name field:

   ```
   09 41 69 72 20 56 65 6e 74 73 | 6d 6f 6f 6d 55 70 00 00 00 00 07 e0 1f f8 3f fc 7f fe
   ^^ len=9      "Air Vents"      | 18 bytes of heap garbage
   ```

   The tail is text (`6d 6f 6f 6d 55 70` = `"moomUp"`, evidently a fragment of
   some other string) followed by `07 e0 1f f8 3f fc 7f fe`, which as bit
   patterns is `.....###/###.....`, `...#####/#####...`, `..######/######..`,
   `.#######/#######.` — a widening symmetric ramp, i.e. almost certainly a
   fragment of a QuickDraw bitmap/mask that shared the heap block.

2. **House-level unused fields.** `unusedShort` is garbage in 10 of 22 houses
   and `unusedBoolean` in 7 of 22 (§11.3).

3. **`'demo'` record padding.** 109 distinct values in the 1-byte pad, mostly
   ASCII fragments (§11.7).

4. **Resource-fork reserved area.** Even the empty `Sampler.rsrc` has 73 non-zero
   bytes in the 240-byte system-reserved region (§11.10).

Consequence: **a byte-for-byte round-trip of a shipped house is impossible
without preserving the garbage.** A port that wants `read → write → identical
bytes` must retain the raw padding. A port that only wants semantic fidelity can
zero it, but then must not use file hashes as a correctness test.

`Demo House`'s high-score block shows the same thing in the persisted
`scoresType` (house offset 528):

```
banner  : len=19 "The Return of Ozma!"   tail = 00 e8 72 50 01 26 33 08 01 26 32 dc
names[0]: len=4  "Ozma"                  tail = 26 2b b0 dd dd dd dd dd dd dd dd
names[1]: len=14 "--------------"         tail = 36
names[2]: len=14 "--------------"         tail = 00
names[3]: len=14 "--------------"         tail = 00
names[4]: len=14 "--------------"         tail = d0
names[5]: len=14 "--------------"         tail = b4
names[6]: len=14 "--------------"         tail = 12
names[7]: len=14 "--------------"         tail = fc
names[8]: len=14 "--------------"         tail = 0c
names[9]: len=14 "--------------"         tail = 48
scores     = (7400, 0, 0, 0, 0, 0, 0, 0, 0, 0)
timeStamps = (2888823855, 0, 0, 0, 0, 0, 0, 0, 0, 0)
levels     = (13, 0, 0, 0, 0, 0, 0, 0, 0, 0)
```

Note `timeStamps[0] = 2 888 823 855` has bit 31 **set** — unlike
`houseType.timeStamp` this field is not masked: `0xAC2FF42F`, bit 31 set. It
decodes directly (no 2³¹ fixup) to 1995-07-17 11:04:15. And the
placeholder name `"--------------"` is exactly 14 dashes, which is what
`GliderPRO/Sources/HighScores.c` writes for an empty slot.

### 11.15 The `'vers'` resource contradicts `Main.c`

```
'vers' 1 (62 bytes) : 01 12 80 00 00 00 05 31 2e 31 2e 32 31 47 6c 69 ...
   numeric version = 01 12 -> 1.1.2
   development stage = 0x80 (final), prerelease = 0
   region = 0x0000
   short string  = Pascal len 5  "1.1.2"
   long  string  = Pascal len 49 "Glider PRO\xAA 1.1.2\r\xA9 1994-95 Casady & Greene, Inc."
'vers' 2 (44 bytes) : short "1.1.2", long "\xA9 1994-95 Casady & Greene, Inc."
'ozm5' 0 (32 bytes) : Pascal len 31 "\xA9 1994-95 Casady & Greene, Inc."
```

`GliderPRO/Sources/Main.c:3` says version **1.0.4**. The resource fork shipped in
this GPL drop says **1.1.2**. So the `.r` dump is from a *later* build than the
`.c` sources. Any claim in this document that rests on a resource body ('demo',
'acur', 'PAT#', PICTs, sounds, DITLs) is therefore a claim about 1.1.2, while
every claim about struct layout and code flow is about 1.0.4. For the data
structures in this document the two agree (all sizes are consistent with the
shipped house files, which predate both), but a porter chasing a pixel-level
discrepancy should keep the split in mind.

The `\xAA` and `\xA9` bytes are MacRoman `™` and `©`. In UTF-8 they must be
re-encoded; in a fixed-width Pascal buffer they occupy one byte each, so a naive
UTF-8 conversion **changes the string length** and can overflow a `Str31`.

### 11.16 `'bnds'` resources in shipped houses

70 `'bnds'` resources exist, in 8 of the 22 houses:

| House | Count |
|---|---:|
| Slumberland | 20 |
| ImagineHouse PRO II | 14 |
| Demo House | 10 |
| Castle o' the Air | 9 |
| Rainbow's End | 6 |
| Land of Illusion | 5 |
| Leviathan | 4 |
| The Asylum Pro | 2 |

The 14 distinct 4-byte bodies and their decoded meanings are tabulated in §9.3.
Every byte is 0 or 1, confirming `boundsType` really is four `Boolean`s in the
order `left, top, right, bottom`.

---

## 12. Recommended Go type definitions

Two layers are recommended, because the C code conflates them:

* **Wire types** (`package housefile`) — fixed-size, `encoding/binary`-friendly
  mirrors of the on-disk layout, using only `int8/uint8/int16/uint16/int32/uint32`
  and fixed-size arrays. These exist to be `binary.Read`/`binary.Write`-ed with
  `binary.BigEndian` and nothing else. **No `bool`, no `string`, no slices, no
  pointers** — anything with a platform-dependent size or an internal pointer
  breaks `binary.Size`.
* **Domain types** (`package house`) — idiomatic Go with `string`, `bool`, typed
  enums, slices and pointers, converted from/to the wire types explicitly.

`encoding/binary` on a struct of fixed-size fields is exactly equivalent to
`#pragma options align=mac68k` on big-endian, because Go's `binary` package packs
with **no padding at all**. Every Glider PRO persisted struct happens to have no
`mac68k` padding either (the only pad byte in the whole set is the trailing one in
`prefsInfo`, §7), so `binary.Size(wireX) == sizeof(X)` for every struct below.
**Assert that in a test.**

### 12.1 Primitives

```c
/* C — QuickDraw, big-endian, v BEFORE h */
struct Point { short v, h; };                       /* 4  */
struct Rect  { short top, left, bottom, right; };   /* 8  */
struct FSSpec { short vRefNum; long parID; Str63 name; };  /* 70 */
typedef unsigned char Str27[28];   /* len byte + 27 */
typedef unsigned char Str31[32];
typedef unsigned char Str255[256];
```

```go
// Go — wire layer
type Point struct{ V, H int16 }                  // 4 bytes; V FIRST
type Rect  struct{ Top, Left, Bottom, Right int16 } // 8 bytes; NOT x,y,w,h

// Pascal strings: fixed buffer, length byte at [0], garbage after the text.
type PStr27  [28]byte
type PStr31  [32]byte
type PStr15  [16]byte
type PStr32  [33]byte // NOTE: 33, an ODD size — see §1.3
type PStr63  [64]byte
type PStr255 [256]byte

// Decode: MacRoman -> UTF-8, honouring the length byte and IGNORING the tail.
func (p PStr27) String() string { return macRomanToUTF8(p[1 : 1+min(int(p[0]), 27)]) }

// Encode: truncate to the buffer capacity, ZERO the tail (differs from the
// original, which leaves heap garbage; see §11.14).
func MakePStr27(s string) (p PStr27) {
    b := utf8ToMacRoman(s)
    if len(b) > 27 { b = b[:27] }
    p[0] = byte(len(b)); copy(p[1:], b); return
}
```

Do **not** model `Point` as `image.Point`: `image.Point` is `{X, Y}` in that
order, so a blind conversion transposes every coordinate in the game. Likewise
`Rect` is `(top, left, bottom, right)` — reading it as `image.Rectangle`'s
`(Min.X, Min.Y, Max.X, Max.Y)` swaps both corners.

`FSSpec` should **not** be ported. It appears only in `game2Type` (dead code,
§6.5) and in the runtime `theHousesSpecs[]` array. Replace with a plain
`string` path, or `os.DirEntry`.

### 12.2 The object payload union

```c
/* C — 9 variants, all exactly 10 bytes, in an untagged union */
typedef struct { Point topLeft; short distance; Boolean initial, state;
                 Byte vector, tall; } blowerType;
typedef struct { Rect bounds; short pict; } furnitureType;
typedef struct { Point topLeft; short length, points;
                 Boolean state, initial; } bonusType;   /* NOTE state BEFORE initial */
typedef struct { Point topLeft; short tall, where;
                 Byte who, wide; } transportType;
typedef struct { Point topLeft; short delay, where;
                 Byte who, type; } switchType;
typedef struct { Point topLeft; short length; Byte byte0, byte1;
                 Boolean initial, state; } lightType;
typedef struct { Point topLeft; short height; Byte byte0, delay;
                 Boolean initial, state; } applianceType;
typedef struct { Point topLeft; short length; Byte delay, byte0;
                 Boolean initial, state; } enemyType;
typedef struct { Rect bounds; short pict; } clutterType;

typedef struct { short what; union { ... } data; } objectType;  /* 12 */
```

Go has no unions. The wire type keeps the raw 10 bytes; the domain type is a
discriminated set of structs behind an interface, or (simpler and faster) a
single flat struct with an explicit `Class`.

```go
// ---- wire layer: byte-exact, 12 bytes ----
type WireObject struct {
    What int16      // object class, or -1 == ObjectIsEmpty
    Data [10]byte   // untagged payload; interpret via ClassOf(What)
}

const ObjectIsEmpty int16 = -1  // kObjectIsEmpty; NOT 0

// ---- payload accessors: pure functions over the 10 bytes ----
type Blower struct {
    TopLeft  Point
    Distance int16
    Initial  uint8 // NOT bool: the value 23 occurs in shipped data (§11.12)
    State    uint8
    Vector   uint8 // MUST be masked & 0x0F before use (§11.12)
    Tall     uint8 // for kLiftArea this is a pixel height, 1..161
}
type Furniture struct { Bounds Rect; Pict int16 }         // Pict always 0 in shipped data
type Bonus struct {
    TopLeft Point
    Length  int16
    Points  int16
    State   uint8 // NOTE: State comes FIRST in this variant
    Initial uint8
}
type Transport struct {
    TopLeft Point
    Tall    uint16 // UNSIGNED: kDeluxeTrans packs ((wide/4)<<8)|(tall/4) (§11.12)
    Where   int16  // packed floor/suite; < 0 means unlinked
    Who     uint8  // destination object index, 0..23; 255 == none
    Wide    uint8
}
type Switch struct {
    TopLeft Point
    Delay   int16
    Where   int16
    Who     uint8
    Type    uint8 // 0..3
}
type Light struct {
    TopLeft Point
    Length  int16
    Byte0   uint8 // dead: always 0
    Byte1   uint8 // dead: always 0
    Initial uint8
    State   uint8
}
type Appliance struct {
    TopLeft Point
    Height  int16 // OVERLOADED: kCustomPict -> PICT id 10000..; kToaster -> height
    Byte0   uint8 // non-zero only for kMicrowave (food index 0..7)
    Delay   uint8
    Initial uint8
    State   uint8
}
type Enemy struct {
    TopLeft Point
    Length  int16
    Delay   uint8
    Byte0   uint8 // dead: always 0
    Initial uint8
    State   uint8
}
type Clutter struct { Bounds Rect; Pict int16 } // non-zero only for kFlower (0..5)

// ---- class -> variant, by range (codes are NOT contiguous, §11.11) ----
type Variant uint8

const (
    VarBlower Variant = iota; VarFurniture; VarBonus; VarTransport; VarSwitch
    VarLight; VarAppliance; VarEnemy; VarClutter; VarNone
)

func VariantOf(what int16) Variant {
    switch {
    case what >= 0x01 && what <= 0x10: return VarBlower
    case what >= 0x11 && what <= 0x20: return VarFurniture
    case what >= 0x21 && what <= 0x30: return VarBonus
    case what >= 0x31 && what <= 0x40: return VarTransport
    case what >= 0x41 && what <= 0x50: return VarSwitch
    case what >= 0x51 && what <= 0x60: return VarLight
    case what >= 0x61 && what <= 0x70: return VarAppliance
    case what >= 0x71 && what <= 0x80: return VarEnemy
    case what >= 0x81 && what <= 0x8F: return VarClutter
    default: return VarNone // includes -1 (empty) and 0 (never occurs)
    }
}
```

A range test, not a lookup table: 0x20, 0x30, 0x50, 0x60, 0x70 and 0x80 are
defined band boundaries with no object assigned, and 0x00 and 0x90+ are invalid
(§11.11).

If you prefer a tagged union, `Object` as an interface with a
`Bounds() Rect` method plus a `Class` field is fine, but keep the raw
`[10]byte` around for round-tripping: the *unused* bytes of a payload carry
garbage you cannot regenerate (§11.14).

### 12.3 `roomType`

```c
typedef struct {                      /* 348 bytes */
    Str27   name;                     /*   0 */
    short   bounds;                   /*  28 */
    Byte    leftStart, rightStart;    /*  30, 31 */
    Byte    unusedByte;               /*  32 */
    Boolean visited;                  /*  33 */
    short   background;               /*  34 */
    short   tiles[kNumTiles];         /*  36 (8 x 2 = 16) */
    short   floor, suite;             /*  52, 54 */
    short   openings;                 /*  56 */
    short   numObjects;               /*  58 */
    objectType objects[kMaxRoomObs];  /*  60 (24 x 12 = 288) */
} roomType;                           /* 60 + 288 = 348 */
```

```go
// ---- wire layer: binary.Size == 348, assert it ----
type WireRoom struct {
    Name        PStr27       //   0 .. 27
    Bounds      int16        //  28   bitfield, see §4.2; 0 or ODD only
    LeftStart   uint8        //  30
    RightStart  uint8        //  31
    UnusedByte  uint8        //  32   always 0 in shipped data
    Visited     uint8        //  33   genuinely persisted (436/4070 true)
    Background  int16        //  34   PICT id: 2000..2017 | 3000..3299 | 3300..3799
    Tiles       [8]int16     //  36   COLUMN INDICES 0..7, not pixel offsets
    Floor       int16        //  52   -7..39 observed; legal -7..56
    Suite       int16        //  54   0..127; -1 == kRoomIsEmpty (deleted room)
    Openings    int16        //  56   DEAD: written 0, never read
    NumObjects  int16        //  58   must equal count of Objects[i].What != -1
    Objects     [24]WireObject // 60 .. 347
}

// ---- domain layer ----
type Room struct {
    Name       string
    Bounds     RoomBounds // decoded bitfield, see below
    LeftStart  uint8
    RightStart uint8
    Visited    bool
    Background int16
    Tiles      [8]uint8
    Floor      int16
    Suite      int16
    Objects    []Object // len() is the live count; NumObjects is derived on write
    Deleted    bool     // Suite == -1
}

// The bounds bitfield of the RAW on-disk short (§4.2). Bit 0 = "this field is
// authoritative"; if clear, fall back to the 'bnds' resource keyed by Background.
// The other bits are the RoomInfo.c code shifted left by 1.
type RoomBounds struct {
    Populated    bool // bit 0 (0x01)  the "+= 1" flag
    LeftOpen     bool // bit 1 (0x02)
    TopOpen      bool // bit 2 (0x04)
    RightOpen    bool // bit 3 (0x08)
    BottomOpen   bool // bit 4 (0x10)
    FloorSupport bool // bit 5 (0x20)  -- unreachable via the 'bnds' fallback
    // There is NO bit 6. IsRoomAStructure (Room.c:777) tests the RAW value with
    // & 32, i.e. this same FloorSupport bit, and calls it "is a structure".
    // Replicate Room.c verbatim -- see the caveat in §4.2. Max value observed
    // across all 4070 shipped rooms is 63, and & 0x40 is never set.
}
```

### 12.4 `houseType`

The C struct's variable tail is the reason for all the 866/868 trouble. In Go,
split it: a fixed 866-byte header struct plus a `[]WireRoom`.

```c
typedef struct {                      /* 866 + nRooms*348 */
    short      version;               /*   0 */
    short      unusedShort;           /*   2 */
    long       timeStamp;             /*   4 */
    long       flags;                 /*   8 */
    Point      initial;               /*  12 */
    Str255     banner;                /*  16 */
    Str255     trailer;               /* 272 */
    scoresType highScores;            /* 528 */
    gameType   savedGame;             /* 820 */
    Boolean    hasGame;               /* 860 */
    Boolean    unusedBoolean;         /* 861 */
    short      firstRoom;             /* 862 */
    short      nRooms;                /* 864 */
    roomType   rooms[];               /* 866 */
} houseType;
```

```go
const (
    HouseHeaderSize = 866             // == offsetof(houseType, rooms)
    RoomSize        = 348             // == sizeof(roomType)
    HouseVersion    = 0x0200          // kHouseVersion
    NewHouseVersion = 0x0300          // kNewHouseVersion (never written by 1.0.4)
)

// binary.Size(WireHouseHeader{}) MUST be 866.
type WireHouseHeader struct {
    Version       int16      //   0  0x0200 in every shipped house
    UnusedShort   int16      //   2  GARBAGE on disk; ignore on read
    TimeStamp     int32      //   4  (macSeconds & 0x7FFFFFFF); bit 0 = LOCKED
    Flags         int32      //   8  bit0 ward, bit1 phone, bit2 !starCount
    Initial       Point      //  12  starting glider position (v, h)
    Banner        PStr255    //  16
    Trailer       PStr255    // 272
    HighScores    WireScores // 528  (292 bytes)
    SavedGame     WireGame   // 820  (40 bytes)
    HasGame       uint8      // 860
    UnusedBoolean uint8      // 861  GARBAGE on disk; ignore on read
    FirstRoom     int16      // 862
    NRooms        int16      // 864
}

type House struct {
    Header WireHouseHeader // keep verbatim so garbage round-trips
    Rooms  []Room
    Slop   []byte // 0 or 2 trailing bytes past 866+348*n (§11.2)
}

func ReadHouse(data []byte) (*House, error) {
    if len(data) < HouseHeaderSize { return nil, errShort }
    var h WireHouseHeader
    if err := binary.Read(bytes.NewReader(data), binary.BigEndian, &h); err != nil { ... }
    // Trust nRooms, but cross-check like ValidateNumberOfRooms does:
    counted := (len(data) - HouseHeaderSize) / RoomSize
    n := int(h.NRooms)
    if counted != n { /* warn; the original trusts `counted` -- §9.1 */ n = counted }
    ...
    slop := data[HouseHeaderSize+n*RoomSize:]  // len(slop) in {0, 2}
}

// Locked / timestamp helpers.
func (h *WireHouseHeader) Locked() bool { return h.TimeStamp&1 == 1 }
func (h *WireHouseHeader) Modified() time.Time {
    macSecs := uint32(h.TimeStamp) | 0x8000_0000       // re-add the masked bit 31
    return time.Unix(int64(macSecs)-2082844800, 0).UTC()
}
func (h *WireHouseHeader) SetModified(t time.Time, locked bool) {
    macSecs := uint32(t.Unix() + 2082844800)
    v := int32(macSecs & 0x7FFF_FFFF)
    if locked { v |= 1 } else { v &^= 1 }
    h.TimeStamp = v
}
```

`Modified`/`SetModified` are lossy in bit 0 by design — that is what the original
does. Do **not** try to preserve the true second; the format sacrifices it for
the lock flag.

### 12.5 `scoresType`, `gameType`, `savedRoom`, `game2Type`

```c
typedef struct {                        /* 292 */
    Str31 banner;                       /*   0 */
    Str15 names[kMaxScores];            /*  32 (10 x 16 = 160) */
    long  scores[kMaxScores];           /* 192 (10 x 4  =  40) */
    unsigned long timeStamps[kMaxScores];/* 232 (10 x 4 =  40) */
    short levels[kMaxScores];           /* 272 (10 x 2  =  20) */
} scoresType;

typedef struct {                        /* 40 */
    short version, wasStarsLeft;        /*  0,  2 */
    long  timeStamp;                    /*  4 */
    Point where;                        /*  8 */
    long  score;                        /* 12 */
    long  unusedLong, unusedLong2;      /* 16, 20 */
    short energy, bands, roomNumber,    /* 24, 26, 28 */
          gliderState, numGliders,      /* 30, 32 */
          foil, unusedShort;            /* 34, 36 */
    Boolean facing, showFoil;           /* 38, 39 */
} gameType;
```

```go
const MaxScores = 10 // kMaxScores

// binary.Size == 292
type WireScores struct {
    Banner     PStr31              //   0
    Names      [MaxScores]PStr15   //  32
    Scores     [MaxScores]int32    // 192  signed: see the overflow note in §9.4
    TimeStamps [MaxScores]uint32   // 232  raw Mac seconds, NOT masked
    Levels     [MaxScores]int16    // 272
}

// binary.Size == 40
type WireGame struct {
    Version      int16 //  0   kSavedGameVersion = 0x0200; shipped data has 0x0100
    WasStarsLeft int16 //  2
    TimeStamp    int32 //  4   NOT masked, unlike houseType.timeStamp
    Where        Point //  8
    Score        int32 // 12
    UnusedLong   int32 // 16   dead
    UnusedLong2  int32 // 20   dead
    Energy       int16 // 24   CAN BE NEGATIVE: -150 observed (§6.5)
    Bands        int16 // 26
    RoomNumber   int16 // 28
    GliderState  int16 // 30
    NumGliders   int16 // 32
    Foil         int16 // 34
    UnusedShort  int16 // 36   dead
    Facing       uint8 // 38
    ShowFoil     uint8 // 39
}

// binary.Size == 292 -- only used by the dead saved-game path (§6.5)
type WireSavedRoom struct {
    UnusedShort int16            // 0
    UnusedByte  uint8            // 2
    Visited     uint8            // 3
    Objects     [24]WireObject   // 4 .. 291
}
```

`game2Type` (the external `.gliS`-style saved game) is **dead code in 1.0.4**
(§6.5): `OpenSavedGame` returns `false` unconditionally and `SaveGame2` is
commented out. A Go port should implement its own save format and not attempt
byte compatibility. If you do implement it, the header is 110 bytes under
`mac68k` (112 natural) and starts with a 70-byte `FSSpec` you cannot meaningfully
reproduce off a Mac.

### 12.6 `prefsInfo`

```c
#pragma options align=mac68k     /* Externs.h:231 -- LOAD-BEARING */
typedef struct {                 /* 226 bytes with the pragma, 228 without */
    Str32 wasDefaultName;        /*   0  (33 bytes -- odd!) */
    Str15 wasLeftName, wasRightName, wasBattName, wasBandName, wasHighName;
    Str31 wasHighBanner;
    long  wasLeftMap, wasRightMap, wasBattMap, wasBandMap;
    short wasVolume, prefVersion, wasMaxFiles, ... ;
    Boolean wasZooms, ... wasBitchDialogs;
} prefsInfo;
#pragma options align=reset      /* Externs.h:269 */
```

```go
const PrefsVersion = 0x0034 // kPrefsVersion, Main.c:16

// binary.Size == 226, but ONLY if you spell both pad bytes out explicitly --
// Go's encoding/binary never inserts padding, so the C compiler's two pad bytes
// have to become real fields.
//
// The 33-byte Str32 at offset 0 is what creates the first one: the six Str15s
// and the Str31 end at 145 (odd), and mac68k aligns `long` to 2, so wasLeftMap
// starts at 146 with ONE pad byte at 145. PowerPC natural alignment would align
// it to 4 -> offset 148, THREE pad bytes, and every field after it shifts by 2.
// That is exactly what the #pragma at Externs.h:231 exists to prevent.
type WirePrefs struct {
    DefaultName [33]byte // 0    Str32 -- yes, 33 bytes
    LeftName    PStr15   // 33
    RightName   PStr15   // 49
    BattName    PStr15   // 65
    BandName    PStr15   // 81
    HighName    PStr15   // 97
    HighBanner  PStr31   // 113 .. 144
    Pad145      uint8    // 145  compiler pad; uninitialised stack on disk
    LeftMap     int32    // 146
    RightMap    int32    // 150
    BattMap     int32    // 154
    BandMap     int32    // 158
    // 23 shorts: 162 .. 207  (wasVolume, prefVersion, wasMaxFiles, wasEditH,
    //   wasEditV, wasMapH, wasMapV, wasMapWide, wasMapHigh, wasToolsH,
    //   wasToolsV, wasLinkH, wasLinkV, wasCoordH, wasCoordV, isMapLeft,
    //   isMapTop, wasNumNeighbors, wasDepthPref, wasToolGroup, smWarnings,
    //   wasFloor, wasSuite)
    // 17 Booleans: 208 .. 224
    Pad225      uint8    // 225  struct tail pad; uninitialised stack on disk
}                        // total 226
```

Full 49-field table with both offset columns is §7. **Do not port this format.**
It is a raw dump of an uninitialised stack local
(`GliderPRO/Sources/Main.c:208-279` writes every field of a `prefsInfo thePrefs;`
that was never zeroed) into a file with no `SetEOF`
(`GliderPRO/Sources/Prefs.c:127-129`), version-gated by an exact `!=` comparison
that **deletes the file** if the version differs in either direction
(`GliderPRO/Sources/Prefs.c:261-266`). A Go port should use JSON/TOML in the
user config dir and ignore the original file entirely.

### 12.7 Runtime-only types

These never touch disk, so port them for clarity, not layout.

```c
typedef struct {                          /* 110 bytes, mac68k */
    Rect src, mask, dest, whole;
    Rect destShadow, wholeShadow, clip, enteredRect;
    long leftKey, rightKey, battKey, bandKey;
    short hVel, vVel, wasHVel, wasVVel, vDesiredVel, hDesiredVel;
    short mode, frame, wasMode;
    Boolean facing, tipped, sliding, ignoreLeft, ignoreRight;
    Boolean fireHeld, which, heldLeft, heldRight, dontDraw, ignoreGround;
} gliderType;                             /* + 1 pad byte at 109 */
```

```go
type Glider struct {
    Src, Mask, Dest, Whole             Rect
    DestShadow, WholeShadow            Rect
    Clip, EnteredRect                  Rect
    LeftKey, RightKey, BattKey, BandKey uint32 // KeyMap bit indices, not codes
    HVel, VVel                         int16
    WasHVel, WasVVel                   int16
    VDesiredVel, HDesiredVel           int16
    Mode, Frame, WasMode               int16
    Facing                             bool
    Tipped, Sliding                    bool
    IgnoreLeft, IgnoreRight            bool
    FireHeld, Which                    bool
    HeldLeft, HeldRight                bool
    DontDraw, IgnoreGround             bool
}
```

`leftKey`/`rightKey`/`battKey`/`bandKey` are `long`s holding **`KeyMap` bit
positions** (Mac virtual key codes indexed into a 128-bit `KeyMap`), not ASCII.
A Go port replaces them with whatever its input layer uses; the *persisted* copies
live in `prefsInfo.wasLeftMap` etc. (§7), which you are not porting anyway.

The other runtime aggregates map one-to-one and need no special care, except:

| C type | Size | Go note |
|---|---:|---|
| `savedType` | 16 | Contains a `void *map` (a `GWorldPtr`); replace with an image handle/index. **Not** 20 bytes: the Mac pointer is 4 bytes, not 8 |
| `boundsType` | 4 | This one *is* an on-disk format (`'bnds'`), so it needs a wire type too: `struct{ Left, Top, Right, Bottom uint8 }` — note the **non-QuickDraw order** |
| `demoType` | **6** | `struct{ Frame int32; Key uint8; Pad uint8 }`. `binary.Size` must be 6; a Go struct of `{int32; uint8; uint8}` is 8 bytes in memory but `binary` writes 6, which is what you want |
| `marquee` | 82 | Holds `Pattern pats[7]` = the `'PAT#'` data of §11.9; bake the 7 patterns in as a literal |
| `macEnviron` | 44 | Delete. It is entirely Gestalt/GDevice capability probing (§8.4) |
| `objDataType` | 26 | Embeds a whole `objectType` by value; in Go prefer an index into the room |

### 12.8 The `'bnds'`, `'demo'`, `'acur'` and `'PAT#'` resources

All four are tiny and fixed, so a Go port should **compile them in as literals**
rather than parse a resource fork:

```go
// 'bnds' -- 4 bytes, one per background PICT id. Field order is
// left,top,right,bottom -- NOT QuickDraw's top,left,bottom,right.
type WireBnds struct{ Left, Top, Right, Bottom uint8 }

// Composed into the SHIFTED code space, i.e. the same space as (bounds >> 1),
// which is what GetOriginalBounding (Room.c:937-966) returns:
func (b WireBnds) Code() int16 {
    var c int16
    if b.Left   != 0 { c |= 0x01 }
    if b.Top    != 0 { c |= 0x02 }
    if b.Right  != 0 { c |= 0x04 }
    if b.Bottom != 0 { c |= 0x08 }
    return c // 0..15 only; the 0x10 (floorSupport) bit is UNREACHABLE -- §9.3
}

// 'demo' -- 1117 records, 6 bytes each, 6702 bytes total (kDemoLength).
type WireDemoStep struct {
    Frame int32 // big-endian, monotonically non-decreasing, 46..3414
    Key   uint8 // 0 = left, 1 = right, 3 = both; 2 never occurs
    Pad   uint8 // uninitialised garbage; ignore
}
const DemoLength = 6702 // kDemoLength; 6702 / 6 == 1117 exactly

// 'acur' -- 12 frames, CURS ids 160 down to 149.
var animCursorIDs = [12]int16{160, 159, 158, 157, 156, 155, 154, 153, 152, 151, 150, 149}

// 'PAT#' 128 -- the 7 marquee barber-pole patterns, verbatim (§11.9).
var marqueePats = [7][8]byte{
    {0xf8, 0xf1, 0xe3, 0xc7, 0x8f, 0x1f, 0x3e, 0x7c},
    {0x3e, 0x7c, 0xf8, 0xf1, 0xe3, 0xc7, 0x8f, 0x1f},
    {0x1f, 0x3e, 0x7c, 0xf8, 0xf1, 0xe3, 0xc7, 0x8f},
    {0x8f, 0x1f, 0x3e, 0x7c, 0xf8, 0xf1, 0xe3, 0xc7},
    {0xc7, 0x8f, 0x1f, 0x3e, 0x7c, 0xf8, 0xf1, 0xe3},
    {0xe3, 0xc7, 0x8f, 0x1f, 0x3e, 0x7c, 0xf8, 0xf1},
    {0xf1, 0xe3, 0xc7, 0x8f, 0x1f, 0x3e, 0x7c, 0xf8},
}
```

### 12.9 The link codec

```c
/* GliderPRO/Sources/Link.c:34-37 -- NOTE: no bias inside the function. */
short MergeFloorSuite (short floor, short suite)
    { return ((suite * 100) + floor); }

/* GliderPRO/Sources/Link.c:277 -- the caller (DoLink) applies the +8: */
    floor += kNumUndergroundFloors;
    ...data.d.where = MergeFloorSuite(floor, suite);

/* GliderPRO/Sources/Link.c:41-53 -- the reverse DOES remove the bias, and
   branches on house version. */
void ExtractFloorSuite (short combo, short *floor, short *suite)
{
    if ((*thisHouse)->version < 0x0200)   /* v1: quotient is the FLOOR  */
    {
        *floor = (combo / 100) - kNumUndergroundFloors;
        *suite = combo % 100;
    }
    else                                  /* v2: quotient is the SUITE  */
    {
        *suite = combo / 100;
        *floor = (combo % 100) - kNumUndergroundFloors;
    }
}
```

```go
const NumUndergroundFloors = 8 // kNumUndergroundFloors

// DELIBERATE REFACTOR: the C MergeFloorSuite takes an already-biased floor and
// every one of its four call sites (Link.c:283,290,302,309) does
// `floor += kNumUndergroundFloors` first (Link.c:277). Folding the bias in makes
// Merge/Extract exact inverses. If you instead transcribe the C literally, you
// MUST replicate the caller-side bias or every link lands 8 floors low.
func MergeFloorSuite(floor, suite int16) int16 {
    return suite*100 + floor + NumUndergroundFloors
}

// C integer division TRUNCATES toward zero. Go's / does too, so this is a direct
// transcription -- but Python's // FLOORS, which is why an early analysis of the
// negative `where` values in this document was wrong before it was corrected.
// houseVersion < 0x0200 swaps quotient and remainder (Link.c:43-47).
func ExtractFloorSuite(where int16, houseVersion uint16) (floor, suite int16) {
    if houseVersion < 0x0200 { // legacy v1 house
        floor = where/100 - NumUndergroundFloors
        suite = where % 100
        return
    }
    suite = where / 100
    floor = where - suite*100 - NumUndergroundFloors
    return
}

// Robust link test. The original only checks != -1, which lets 202 dangling
// links through (§11.13).
func IsLinked(where int16, who uint8) bool {
    return where >= 0 && who <= 23   // kMaxRoomObs-1
}
```

The `+8` bias is what lets `floor` be negative (basements): floor −8 maps to
`where` offset 0. `where == -1` is the explicit "unlinked" marker;
`where == -100` decodes to `suite == -1 == kRoomIsEmpty` and means the
destination room was deleted (§11.13).

### 12.10 What the Go port must supply in place of the Toolbox

| Mac Toolbox facility | Used for | Go replacement |
|---|---|---|
| Resource Manager (`GetResource`, `FSpOpenResFile`) | house PICTs/sounds/`'bnds'`, app art | Own asset bundle (`embed.FS`), plus a resource-fork reader for legacy houses |
| `Handle` / `NewHandle` / `HLock` / `SetHandleSize` / `PtrAndHand` | the whole in-memory house | `[]byte` and `[]Room` slices; `append` replaces `PtrAndHand` |
| `BlockMove` | `'demo'` load, struct copies | `copy()` |
| `FSSpec`, `FSpOpenDF`, `FSRead`/`FSWrite`, `GetEOF`/`SetEOF` | house + prefs + scores I/O | `os.ReadFile`/`os.WriteFile`; note `SetEOF` is what *truncates*, and `WritePrefs` omits it (§7.3) |
| `PBGetCatInfo` recursive scan, `fdType=='gliH'`, `fdCreator=='ozm5'` | finding houses | `filepath.WalkDir` + extension/magic sniff; type/creator codes do not exist off HFS |
| `FindFolder(kPreferencesFolderType)` | prefs + `"G-PRO Scores ƒ"` folder | `os.UserConfigDir()` |
| GWorlds (`NewGWorld`, `CopyBits`, `CopyMask`) | all rendering | Any 2-D blitter; `image.Paletted` + a custom `DrawMasked` |
| 8-bit indexed colour + `clut` resources | all art | Decode the CLUT once, convert to RGBA at load time |
| Sound Manager (`SndPlay`, `'snd '`) | all audio | Any mixer; `'snd '` format 1/2 must be decoded |
| QuickTime (`NewMovieFromFile`, `PrerollMovie`, `'LOOP'` user data) | in-house TV screens (`.mov` side-car) | Any video decoder, or pre-extract frames |
| `Gestalt`, `macEnviron` | capability probing | Delete entirely (§8.4) |
| `GetIndString` / `STR#` | all UI strings | Go string tables |
| `KeyMap` / `GetKeys` bit positions | input | Your input layer's key enum |
| MacRoman text encoding | every Pascal string | `golang.org/x/text/encoding/charmap.Macintosh`, or a 128-entry table (airgap-friendly) |

---

## 13. Open questions

Things this document could not settle from the source and shipped data alone.
None of them block a port, but each is a place where a fidelity bug could hide.

1. **`Str32` really is 33 bytes — but nothing on disk proves it.** The Apple
   header defines `Str32` as `unsigned char[33]`, which is the only odd-sized
   Pascal string type in the family and the sole reason `prefsInfo`'s
   `wasLeftMap` lands at 146 rather than 148. No `Glider Prefs` file ships with
   the GPL drop, so the 226-byte figure is derived from the pragma plus the
   headers, never observed. **If `Str32` were 32 bytes, `prefsInfo` would be 224
   and every offset from 145 on would shift.** A porter with access to a real
   `Glider Prefs` file should check its length first.

2. **`Str63` = 64 is likewise unverified** for the same reason: it appears only
   inside `FSSpec` (70 bytes) and `FSSpec` appears only in `game2Type`, which is
   dead code. No saved-game file ships.

3. **The `'vers'` resource says 1.1.2; `Main.c:3` says 1.0.4** (§11.15). The
   resource fork in this GPL drop is from a later build than the sources. Which
   parts of `Glider PRO.r` correspond to 1.0.4 and which drifted is unknowable
   from here. All *struct layouts* agree with the shipped house files (which
   predate both), so this only affects art, sounds, dialogs and strings.

4. **`IsRoomAStructure` and `RoomInfo.c` disagree about bit 5 of `bounds`.**
   `RoomInfo.c:744-780` shifts right by 1 and then tests `& 16` for
   `floorSupport`; `Room.c:773-786` tests the **raw** value with `& 32`, which is
   the same bit — but `RoomInfo.c` has no notion of a "structure" bit at all, and
   the natural reading of the encoder is that bit 5 raw = `floorSupport`. So
   either `IsRoomAStructure` is reusing `floorSupport` as "is a structure", or one
   of the two is off by one shift. Shipped data cannot distinguish them
   (`bounds` maxes at 63, so bit 6 is never set). **Replicate `Room.c` verbatim.**

5. **Should `where == -100` be treated as "no link"?** It decodes to
   `suite == -1 == kRoomIsEmpty`, i.e. the destination room was deleted. 165
   shipped objects are in this state (§11.13). `CountHouseLinks` counts them as
   links (it only tests `!= -1`), and `DoLink`/`DoUnlink`
   (`GliderPRO/Sources/Link.c:270-361`) would happily try to resolve them. What
   the *play* code does when it follows one is not established here — it depends
   on `Transit.c`'s room lookup returning `-1` and the caller's handling.

6. **No version-1 (`0x0100`) house survives.** All 22 shipped houses are
   `0x0200`, so `ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746-816`) is
   untestable against real data. Its transformation (§9.1 note,
   `GliderPRO/Sources/HouseIO.c:610-612`) is transcribed from the C but never
   exercised.

7. **`kNewHouseVersion = 0x0300` is defined but never written.** Nothing in
   1.0.4 produces it and no shipped house has it. Whether a 0x0300 format was ever
   specified is unknown.

8. **The `'Date'` resource fallback is never exercised.** `ReadyBackground`
   (`GliderPRO/Sources/Room.c:244-305`) will `GetResource('Date', theID)` under
   some conditions, but **no shipped house contains a `'Date'` resource** (§9.2
   inventory). The code path is vestigial in practice.

9. **`furnitureType.pict`, `lightType.byte0`/`byte1` and `enemyType.byte0` are
   dead in all shipped data** (§11.12). They may have been live in an earlier
   format version, or reserved for a later one. A port must preserve the bytes but
   has no data to test any semantics against.

10. **`applianceType.byte0` for `kMicrowave` never takes the value 5** across
    all 56 shipped microwaves (`{0,1,2,3,4,6,7}` occur). The count of food
    sprites is 8 in the code; whether index 5 is a real-but-unused sprite or a
    gap is not determinable from data.

11. **The two `kStar` objects with `points = 85` and `145`** (Slumberland rooms
    61 and 104, §11.12) are almost certainly editor junk, since `kStar` scoring is
    hard-coded. But "almost certainly" is not "verified": confirming it requires
    tracing every read of `bonusType.points`, which is outside this document's
    scope.

12. **`roomType.openings` is 0 in all 4070 shipped rooms and `grep` finds no
    read of it.** Concluding it is dead required an exhaustive search that could
    have missed an indirect access through a cast. Treat as dead but preserve.

13. **`gameType.unusedLong`, `unusedLong2`, `unusedShort` and
    `wasStarsLeft`** — the `was` prefix implies these once had meaning. Both
    houses with `hasGame == 1` have `version == 0x0100`, older than
    `kSavedGameVersion = 0x0200`, so the embedded saves are from a format the
    shipping code would reject anyway. Their field semantics are unrecoverable.

14. **`demoType.key` value 2** never appears in the shipped `'demo'` (§11.7).
    `GliderPRO/Sources/Input.c`'s `LogDemoKey` can emit it (battery/helium), so
    the recorded demo simply never used the item. A port replaying the demo will
    never exercise that branch.

15. **What exactly `houseType.flags` bit 0 (`wardBitSet`) does** is not
    determinable from data: **no shipped house sets it** (§11.3). The name
    suggests a "ward" (a guarded area?) but there is no coverage.

16. **The 240 system-reserved bytes at the head of a resource fork** are
    non-zero even in `Sampler`'s empty fork (§11.10). Whether any Mac tool cares
    about their content is unknown; a Go writer producing a new resource fork
    should probably zero them, accepting that this is not byte-identical.

17. **BinHex is a distribution wrapper, not the file format.** The task brief
    described `Houses/*.mov` as "the data fork of some houses". That is **wrong**:
    `OpenHouseMovie` (`GliderPRO/Sources/HouseIO.c:67-140`) takes the *house
    file's* name and appends `"\p.mov"` — the `.mov` files are separate sibling
    files holding QuickTime video for in-house TV screens
    (`GliderPRO/Sources/HouseIO.c:80-81`). The house data fork is the `houseType`
    structure, and every `.binhex` carries it (e.g. `Demo House` dataLen = 16 526,
    exactly `866 + 348 × 45`). Recorded here so the error is not propagated.

18. **`ReadScoresFromDisk` reads the whole file into a 292-byte field.** It
    calls `GetEOF` into `byteCount` and then `FSRead(scoresRefNum, &byteCount,
    theScores)` where `theScores` points at `&((*thisHouse)->highScores)` inside
    the house handle (`GliderPRO/Sources/HighScores.c:823-841`). A high-score
    side-car longer than 292 bytes overwrites `savedGame`, `hasGame`,
    `firstRoom`, `nRooms` and then the room array. Whether the original ever hit
    this in the wild is unknown; a Go port must clamp the read to 292 bytes, and
    should decide deliberately whether to *reject* over-long files (safe) or
    truncate them (bug-compatible).

19. **`WritePrefs` never calls `SetEOF`** (`GliderPRO/Sources/Prefs.c:97-144`,
    compare `WriteScoresToDisk` at `HighScores.c:785` which does). If an existing
    prefs file were longer than 226 bytes, the tail would survive. Since
    `LoadPrefs` reads exactly `sizeof(prefsInfo)` it would not notice. Harmless in
    practice, but it means "the prefs file is 226 bytes" is not guaranteed.

20. **`WriteOutPrefs` writes `thePrefs.wasMaxFiles = willMaxFiles`
    (`GliderPRO/Sources/Main.c:244`) while `ReadInPrefs` assigns the loaded value
    to `maxFiles` (`GliderPRO/Sources/Main.c:87`).** Two different globals. Whether
    this is intentional (pending-vs-active setting) or a bug is not determinable
    from the source alone.

---

## 14. Porting notes

### 14.1 The five numbers to get right

If a Go port gets these five constants wrong, nothing else matters:

| Value | What it is | Where it comes from |
|---:|---|---|
| **866** | offset of `rooms[0]` in a house data fork | `offsetof(houseType, rooms)` under `align=mac68k`; verified against 21 of 22 shipped houses (§11.2) |
| **348** | `sizeof(roomType)` | `GliderStructs.h:180` `// total = 348`; verified by compilation and by file sizes |
| **12** | `sizeof(objectType)` | `GliderStructs.h:105`; 24 × 12 = 288 = the object array |
| **10** | size of every one of the 9 payload variants | `GliderStructs.h:11-88`, nine `// total = 10` comments, all verified |
| **6** | `sizeof(demoType)` | 6702 / 1117 (§11.7); **8 under natural alignment, which would break the demo** |

Everything else in the on-disk format follows from these plus big-endian and
`Point{v, h}`.

### 14.2 Ranked pitfalls

**1. `Point` is `{v, h}` — vertical first.** Every coordinate in every object
payload, plus `houseType.initial`, is stored this way. Go's `image.Point` is
`{X, Y}`. Silently transposing every coordinate in the game is the single easiest
way to produce a port that "loads houses fine" and is completely broken.

**2. `Rect` is `{top, left, bottom, right}`.** Not `{x, y, w, h}`, not
`{left, top, right, bottom}`. Appears in `furnitureType.bounds` and
`clutterType.bounds` on disk, and pervasively at run time.

**3. `'bnds'` is `{left, top, right, bottom}` — a *different* order from `Rect`.**
The one place in the format where the QuickDraw convention does *not* apply
(§9.3, `GliderStructs.h:266-272`). Four bytes, easy to get backwards, and the
symptom is rooms with the wrong walls open.

**4. The empty-object sentinel is `-1`, not `0`.** `kObjectIsEmpty = -1`
(`GliderPRO/Headers/GliderDefines.h:526`). `what == 0` occurs in **zero** of the
97 680 shipped slots (§11.11). A Go writer that zeroes unused slots produces a
file the original reads as 24 objects of class 0.

**5. Same for rooms: `kRoomIsEmpty = -1`** (`GliderDefines.h:525`) in
`roomType.suite`. No shipped house has a deleted room (all are compressed), so a
port's own editor is the first thing that will ever produce one.

**6. `transportType.tall` must be read as `uint16`.** `kDeluxeTrans` packs
`((wide/4) << 8) | (tall/4)` into it, so a `wide` of 512 sets bit 15 and the value
goes negative as `int16`. Observed: −32 722 = `0x802E` (§11.12). Reading it signed
gives a nonsense negative size.

**7. `blowerType.vector` must be masked `& 0x0F`.** Four shipped objects have
junk in the high nibble (17, 20). `ObjectRects.c:523-560` masks; anything that
doesn't will fail its `switch` and the blower does nothing (§11.12).

**8. `Boolean` fields are not booleans.** `blowerType.initial` and `.state` hold
the value **23** in three shipped objects (§11.12). The C tests truthiness
(`if (theObj->initial)`), so port them as `uint8` and test `!= 0`. Decoding as
`b == 1` diverges.

**9. `bonusType` has `state` *before* `initial`.** Every other variant with those
two fields (`blowerType`, `lightType`, `applianceType`, `enemyType`) has `initial`
first. A 1-byte swap that changes which bonuses start collected
(`GliderStructs.h:11-88`).

**10. `applianceType.height` is overloaded.** For `kCustomPict` (0x6E — the
single commonest object class at 4782 instances) it is a **PICT resource ID**
≥ 10 000, not a pixel height (§11.12). For `kToaster` it really is a height. A
port that clamps "height" to a sane pixel range destroys every custom picture in
every house.

**11. `houseType.timeStamp` is masked with `0x7FFFFFFF` and bit 0 is the lock
flag.** `GliderPRO/Sources/HouseIO.c:478` masks; `:484-486` sets/clears bit 0.
Decoding without re-adding 2³¹ gives dates in 1927–1932 (§11.4) — wrong but
plausible enough to ship. Note `gameType.timeStamp` and
`scoresType.timeStamps[]` are **not** masked, so the same helper cannot be used
for all three.

**12. Pascal strings have garbage after the text.** Read only
`buf[1 : 1+buf[0]]` and clamp `buf[0]` to the buffer capacity. `Str27`'s length
byte can legally be up to 27 but a corrupt file could hold 255 — `Sampler`
mis-parsed at offset 868 produced exactly that (`nameLen=110`, §11.2).

**13. Text is MacRoman, not Latin-1 and not UTF-8.** `0xC4` is `ƒ` (the
high-scores folder is literally `"G-PRO Scores ƒ"`,
`GliderPRO/Sources/HighScores.c`), `0xA9` is `©`, `0xAA` is `™`. `grep` treats
`GliderDefines.h`, `HighScores.c` and `Glider PRO.r` as **binary** because of
these bytes — use `grep -a`. In a fixed-width Pascal buffer each is one byte, so
converting to UTF-8 changes the length and can overflow a `Str31`.

**14. Sources are CR-only (classic Mac line endings).** `wc -l` reports 0 on
every `.c`/`.h` in `GliderPRO/Sources` and `GliderPRO/Headers`. Convert with
`tr '\r' '\n'` before reading. Every line-number citation in this document is
against the converted copy. (`Glider PRO.r` is the exception — it uses LF.)

**15. Resource-fork counts are stored as "count − 1", and `0xFFFF` means zero.**
`Sampler.rsrc` is a 286-byte fork with `0xFFFF` types (§11.10). `nTypes = raw + 1`
reads that as 1 and walks off the end of the file.

**16. C integer division truncates toward zero; Python's `//` floors.**
`ExtractFloorSuite` (`GliderPRO/Sources/Link.c:34-53`) divides a possibly negative
`where` by 100. Go's `/` matches C, so a direct transcription is correct — but any
Python-based analysis tooling must emulate the truncation, or negative `where`
values decode to the wrong room. This is a real bug that occurred while producing
this document.

**17. `where < 0` is the safe "unlinked" test, not `where != -1`.**
`CountHouseLinks` (`GliderPRO/Sources/House.c:258-307`) tests `!= -1` and thereby
counts 165 objects whose destination room was deleted (`where == -100`) plus 24
whose `where` resolves to nothing plus 13 with an out-of-range `who` — 202
dangling links out of the 2474 it reports (§11.13).

**18. Bounds-check `who` before indexing.** One shipped object
(`CD Demo House`, `kMailboxRt`) has `who == 35`, past `kMaxRoomObs - 1 = 23`
(§11.13). Twelve more have `who == 255` with a valid room.

**19. `kSoundTrigger` (0x49) reuses `where` as a sound ID, not a room.** Values
3000–3058 plus two 10 000s, never −1 (§11.13). Including it in the link walk
mis-reads all 122 as links.

**20. `sizeof(houseType)` is used as the room-array base *and* as the header
size**, in four separate places
(`GliderPRO/Sources/HouseLegal.c:630-631`, `:765-767`, `GliderPRO/Sources/Room.c:200`,
`GliderPRO/Sources/HouseIO.c:473`). Under `mac68k` both are 866 and it works.
Under natural alignment they are 868 and 866 and it breaks — which is exactly
what happened to `Sampler`. **Use 866 for the base and tolerate 0 or 2 trailing
bytes** (§11.2).

**21. Never assume `nRooms` and the file length agree.** `ValidateNumberOfRooms`
(`GliderPRO/Sources/HouseLegal.c:621-640`) computes the count from the handle size
and **trusts the computed value over the stored one**. A port should do the same
(or at least warn), because that is the behaviour houses were saved against.

**22. `visited` is persisted; `openings` and `unusedByte` are not meaningful.**
436 of 4070 shipped rooms have `visited == 1` (§11.5). It is real state, not a
runtime cache. `openings` is 0 everywhere and never read. `roomType.unusedByte`
is 0 everywhere (unlike the house-level unused fields, which hold garbage).

**23. `tiles[i]` is a column index 0…7, not a pixel offset.** Multiply by
`kTileWide = 64` (`GliderDefines.h:497`) to get the source x in the background
PICT. All 4070 rooms confirm the range (§11.5).

**24. `prefsInfo`'s `align=mac68k` pragma is load-bearing.** It is the *only*
alignment pragma in the codebase (`GliderPRO/Headers/Externs.h:231`, reset at
`:269`). Without it, `wasLeftMap` moves 146 → 148 and all 38 following fields
shift by 2 (§7, §12.6). Do not port this format at all — but if you do, spell out
both pad bytes (offset 145 and offset 225) as explicit fields, because
`encoding/binary` inserts no padding.

**25. `LoadPrefs` deletes the prefs file on any version mismatch, in either
direction.** `if (thePrefs->prefVersion != versionNeed)` → alert → `DeletePrefs`
→ return false (`GliderPRO/Sources/Prefs.c:261-266`). A *newer* prefs file is
destroyed, not ignored. `kPrefsVersion = 0x0034`
(`GliderPRO/Sources/Main.c:16`).

**26. Both `prefsInfo` and `houseType`'s unused fields are written from
uninitialised memory.** `WriteOutPrefs` fills a bare stack local
(`GliderPRO/Sources/Main.c:208-279`); `unusedShort` is garbage in 10 of 22 houses
and `unusedBoolean` in 7 (§11.3); `'demo'` padding has 109 distinct values
(§11.7). **A byte-exact read→write round-trip is impossible unless the raw
padding is retained.** Decide up front whether the port aims for byte fidelity
(keep the bytes) or semantic fidelity (zero them, and do not use file hashes as
a test).

### 14.3 Fidelity risks specific to Go

* **`encoding/binary` and `binary.Size` are your friend, but verify.** Go packs
  with no padding, which coincidentally matches `align=mac68k` for every persisted
  Glider PRO struct *except* `prefsInfo` (two pad bytes) — so write a test that
  asserts `binary.Size(WireRoom{}) == 348`, `binary.Size(WireObject{}) == 12`,
  `binary.Size(WireHouseHeader{}) == 866`, `binary.Size(WireScores{}) == 292`,
  `binary.Size(WireGame{}) == 40`, `binary.Size(WireSavedRoom{}) == 292`,
  `binary.Size(WireDemoStep{}) == 6`, `binary.Size(WireBnds{}) == 4`.

* **`binary.Read` on a struct containing a Go `bool` works but is a trap.** It
  decodes any non-zero byte as `true` and re-encodes `true` as `1`, so the value
  23 in `blowerType.initial` becomes 1 on write. Use `uint8`.

* **Do not use `unsafe` + struct casts to mimic the C.** The temptation is real
  (the C does exactly that), but Go structs have platform alignment and no
  guarantee of field order stability across architectures. `encoding/binary` on
  explicit wire types is both correct and about as fast once you buffer.

* **`int16` vs `int` in arithmetic.** `MergeFloorSuite(floor, suite)` is
  `suite*100 + floor + 8` in `short` arithmetic, so it **wraps at 32 767**. With
  `suite` up to 127 the max is 12 707, well inside range, so no shipped house
  wraps — but a port using `int` will not reproduce overflow behaviour if a
  malformed file ever triggers it. Keep the intermediate types 16-bit if you want
  bug compatibility.

* **8-bit indexed colour.** All art is `PICT` with a CLUT. Convert to RGBA once at
  load and cache; do not try to emulate `CopyBits` colour-table remapping unless
  you are also porting the colour-fade effects.

* **`GWorldPtr` fields inside runtime structs** (`savedType.map`) are 4 bytes on
  a Mac, 8 on a 64-bit host. `layout.c` prints `sizeof(savedType) == 20` on this
  machine for exactly that reason; the true Mac value is **16**. Any size
  measurement of a struct containing pointers must model them as `uint32`.

* **Type/creator codes do not exist off HFS.** House discovery
  (`GliderPRO/Sources/SelectHouse.c:557-640`) matches `fdType == 'gliH' &&
  fdCreator == 'ozm5'`, walks up to `kMaxDirectories = 32` directories breadth
  first from the application's own folder, and caps results at `maxFiles`. A Go
  port must sniff content instead: `version == 0x0200` at offset 0 plus
  `len(data) == 866 + 348*nRooms` is a strong and cheap test.

### 14.4 Suggested port order

1. **Wire types + `binary.Size` assertions** (§12). Nothing works until 866/348/12
   are right.
2. **BinHex 4.0 decoder** so the 22 shipped houses are usable as test fixtures
   (§9.11). It is ~60 lines: translate the 64-char alphabet to base64, decode,
   then expand the `0x90` RLE.
3. **House data-fork reader** with the `nRooms`-vs-length cross-check, then a
   round-trip test over all 22 houses that asserts `len(out) == len(in)` and, if
   you keep the padding, `bytes.Equal`.
4. **Object payload decoders** with the `& 0x0F` mask, the `uint16` transport
   `tall`, and the `-1`/`-100`/`who > 23` link guards.
5. **Resource-fork reader** (`0xFFFF` = zero types!) for PICTs, sounds and
   `'bnds'`.
6. **`'bnds'`/`'demo'`/`'acur'`/`'PAT#'` as compiled-in literals** (§12.8) — they
   are 4, 6702, 52 and 58 bytes respectively and never change.
7. **Skip `prefsInfo` and `game2Type` entirely.** Use JSON in
   `os.UserConfigDir()` and your own save format. Neither is needed for
   compatibility with anything that ships.

