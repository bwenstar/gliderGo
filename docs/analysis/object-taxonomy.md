# Glider PRO 1.0.4 — Complete Object Taxonomy

## Scope

This document is the canonical reference for **every object type in Glider PRO 1.0.4**: its
numeric ID, which union variant its 10 payload bytes are interpreted as, what each field means,
its default size and rectangle, whether it animates, how it registers hot spots, how its
on/off state is stored, and how it is written to disk.

It covers:

- the `objectType` record and its nine union variants, with sizes and offsets proven by compiling
  the original struct declarations (`/tmp/wf-objtax/c/layout.c`, gcc, LP64) and by parsing the
  bytes of all 22 shipped house files;
- the complete ID space (117 defined types across 9 classes, IDs `0x01`–`0x8F` with holes);
- the per-class field discipline — which union member each class reads and which fields it
  leaves as junk;
- the default source rectangle (i.e. default width × height) for the 116 of 144 array slots that
  are actually initialised;
- the hot-spot / action-code table that drives all player interaction;
- the state model (`SetObjectState` / `GetObjectState`), the link model (`where` / `who`), and
  the dynamic-object (animation) registry;
- the on-disk byte layout, big-endian, offset by offset, with observed hex from real houses;
- the per-room object limit (`kMaxRoomObs` = 24) and the editor's additional per-type caps.

It does **not** cover: object *drawing* internals (the PICT sheet blitters in `ObjectDraw.c` /
`ObjectDraw2.c` beyond which routine each type uses), the editor's mouse/marquee handling,
room/house-level structure beyond what is needed to locate an object record, or physics
beyond the action code each hot spot carries.

Two conventions used throughout:

- Object IDs are given in hex as the source gives them, with decimal in parentheses on first use.
- Line citations are to the CR→LF-converted copies of the Classic-Mac text files. The original
  `Sources/*.c` and `Headers/*.h` use CR-only line endings; convert with
  `tr '\r' '\n' < src > dst` before counting lines. Line numbers are identical in the converted
  file because the conversion is 1 byte → 1 byte.

## Sources read

Read in full (converted to LF):

| File | Converted lines | What it supplies |
|---|---|---|
| `GliderPRO/Headers/GliderDefines.h` | 625 | every object ID, action code, limit, placement constant |
| `GliderPRO/Headers/GliderStructs.h` | 347 | `objectType`, the 9 variants, `roomType`, `houseType`, `dynaType`, `objDataType`, `hotObject` |
| `GliderPRO/Sources/Objects.c` | 1001 | `IsThisValid`, `GetRoomLinked`, `GetObjectLinked`, `ListAllLocalObjects`, `SetObjectState`, `GetObjectState`, `BringSendFrontBack` |
| `GliderPRO/Sources/ObjectRects.c` | 1188 | `GetObjectRect` (bounding rect per type), `AddActiveRect`, `CreateActiveRects` (hot spots per type) |
| `GliderPRO/Sources/ObjectInfo.c` | 2567 | the editor's per-class Info dialogs — the authoritative statement of what each field *means* and its legal range |
| `GliderPRO/Sources/ObjectAdd.c` | 1084 | `AddNewObject` — the default value of every field for every type, plus all per-room caps |
| `GliderPRO/Sources/StructuresInit.c` | 724 | PICT sheet IDs, sheet sizes, animation sub-rect tables |
| `GliderPRO/Sources/StructuresInit2.c` | 476 | `CreatePointers` (array sizes), `InitSrcRects` (the default-size table), `InitClutter` (`flowerSrc`) |
| `GliderPRO/Sources/ObjectDrawAll.c` | 966 | `DrawARoomsObjects` — which drawing routine and which animation registration each type uses |
| `GliderPRO/Sources/Dynamics3.c` | 555 | `AddDynamicObject` — how each animated type seeds its `dynaType` slot |
| `GliderPRO/Sources/Link.c` | 396 | `MergeFloorSuite`, `ExtractFloorSuite`, `DoLink`, `DoUnlink`, `UpdateLinkControl` (legal link targets) |
| `GliderPRO/Sources/Triggers.c` | 206 | trigger arming/firing and which linked types a trigger can fire |
| `GliderPRO/Sources/HouseIO.c` | 708 | `ReadHouse` / `WriteHouse` — proof that the on-disk image is the raw big-endian struct |

Read in part:

| File | Lines read | What it supplies |
|---|---|---|
| `GliderPRO/Sources/Interactions.c` | 13–19, 754–1640 | `HandleRewards`, `HandleSwitches`, `HandleMicrowaveAction`, `HandleHotSpotCollision` (the full action-code switch) |
| `GliderPRO/Sources/House.c` | 740–858 | `ConvertHouseVer1To2` (the v1→v2 link re-encode), `ShiftWholeHouse` (a stub) |
| `GliderPRO/Sources/Room.c` | 709–759 | `GetRoomFloorSuite`, `GetRoomNumber` (link resolution) |
| `GliderPRO/Sources/RectUtils.c` | 53–63, 197–216 | `ZeroRectCorner`, `QOffsetRect`, `QSetRect` semantics |
| `GliderPRO/Sources/Grease.c` | 43–302 | grease state transitions |
| `GliderPRO/Glider PRO.r` | resource `STR# 1007` | the user-visible name of every object, indexed by ID |

Binary evidence (all verified this session, scripts in `/tmp/wf-objtax/`):

- All 22 `GliderPRO/Houses/*.binhex` files decoded with a from-scratch BinHex 4.0 decoder
  (`/tmp/wf-objtax/binhex2.py`), yielding data + resource forks.
- 4070 rooms and **31 440 live object records** parsed and cross-tabulated
  (`/tmp/wf-objtax/stats.py`, `/tmp/wf-objtax/stats2.py`).
- Struct sizes and field offsets checked against a compiled probe (`/tmp/wf-objtax/c/layout.c`).

---

# 1. The object record

## 1.1 `objectType` — 12 bytes, tagged union

`GliderPRO/Headers/GliderStructs.h:90-105`:

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

The author's own comments give the byte counts, and they are exact. Compiled probe output:

```
Point            size= 4
Rect             size= 8
blowerType       size=10
furnitureType    size=10
bonusType        size=10
transportType    size=10
switchType       size=10
lightType        size=10
applianceType    size=10
enemyType        size=10
clutterType      size=10
objectType       size=12
roomType         size=348
objectType: what@0 data@2
```

There is **no padding anywhere** in the object record. That is not an accident: every variant is
built from `Point` (2 × `short`), `Rect` (4 × `short`), `short`, `Byte` and `Boolean` in an order
that keeps all `short`s 2-aligned. `Byte` is `unsigned char`; `Boolean` is `unsigned char`
(Classic Mac `Types.h`). A Go port must therefore treat the record as **exactly 12 bytes with no
alignment slack** and must not rely on Go's own struct layout.

`what` is a *signed* `short`. Its only negative legal value is `kObjectIsEmpty`
(`GliderPRO/Headers/GliderDefines.h:526`):

```c
#define kObjectIsEmpty				-1
```

On disk that is `0xFFFF`. A slot whose `what` is `-1` is free; the remaining 10 bytes are stale
garbage and must be ignored (see §1.7).

## 1.2 The nine variants, byte by byte

Offsets below are given twice: **U** = offset within the 10-byte union, **R** = offset within the
12-byte record (= U + 2). All values verified by the compiled probe.

### Variant `a` — `blowerType` (blowers, class `0x01`–`0x10`)

`GliderPRO/Headers/GliderStructs.h:11-19`:

```c
typedef struct
{
	Point		topLeft;				// 4
	short		distance;				// 2
	Boolean		initial;				// 1
	Boolean		state;					// 1		              F. lf. dn. rt. up
	Byte		vector;					// 1		| x | x | x | x | 8 | 4 | 2 | 1 |
	Byte		tall;					// 1
} blowerType;							// total = 10
```

| U | R | Field | Type | Size | Meaning |
|---|---|---|---|---|---|
| 0 | 2 | `topLeft.v` | `short` BE | 2 | top edge, room-local pixels |
| 2 | 4 | `topLeft.h` | `short` BE | 2 | left edge, room-local pixels |
| 4 | 6 | `distance` | `short` BE | 2 | length of the air column in pixels; for `kLiftArea`, the *width* of the lift rect |
| 6 | 8 | `initial` | `Boolean` | 1 | state at room entry (0/1) |
| 7 | 9 | `state` | `Boolean` | 1 | current runtime state (0/1) |
| 8 | 10 | `vector` | `Byte` | 1 | direction bitfield, low nibble only: `1` = up, `2` = right, `4` = down, `8` = left |
| 9 | 11 | `tall` | `Byte` | 1 | for `kLiftArea` only: half the rect height (the rect is `tall * 2` pixels high) |

Note the field order inside `Point`: QuickDraw's `Point` is `{short v; short h;}` — **vertical
first**. This trips up every port. Same for `Rect`: `{short top; short left; short bottom; short right;}`.

The `vector` comment in the header (`F. lf. dn. rt. up | x | x | x | x | 8 | 4 | 2 | 1 |`) reads
right-to-left over the low nibble: bit 0 (value 1) = up, bit 1 (2) = right, bit 2 (4) = down,
bit 3 (8) = left. The high nibble is documented as `x` (unused) and the code always masks with
`0x0F` before switching on it (`GliderPRO/Sources/ObjectRects.c:522`, `:595`;
`GliderPRO/Sources/ObjectInfo.c:1070`). Real houses do contain `vector` values with high bits set
(observed max `0x14` for `kInvisBlower`), so the mask is load-bearing.

### Variant `b` — `furnitureType` (furniture, class `0x11`–`0x1F`)

`GliderPRO/Headers/GliderStructs.h:21-25`:

```c
typedef struct
{
	Rect		bounds;					// 8
	short		pict;					// 2
} furnitureType;						// total = 10
```

| U | R | Field | Type | Size | Meaning |
|---|---|---|---|---|---|
| 0 | 2 | `bounds.top` | `short` BE | 2 | rect top |
| 2 | 4 | `bounds.left` | `short` BE | 2 | rect left |
| 4 | 6 | `bounds.bottom` | `short` BE | 2 | rect bottom |
| 6 | 8 | `bounds.right` | `short` BE | 2 | rect right |
| 8 | 10 | `pict` | `short` BE | 2 | **unused for every furniture type**; `AddNewObject` writes 0 and nothing ever reads it |

Furniture is the only class (with clutter) that stores an **explicit rectangle** rather than a
top-left plus a table-driven size, because furniture is resizable in the editor. Observed
`pict` range across all 22 houses: `0..0` for all 15 furniture types — confirming it is dead.

### Variant `c` — `bonusType` (prizes/bonuses, class `0x21`–`0x2F`)

`GliderPRO/Headers/GliderStructs.h:27-34`:

```c
typedef struct
{
	Point		topLeft;				// 4
	short		length;					// 2 grease spill
	short		points;					// 2 invis bonus
	Boolean		state;					// 1
	Boolean		initial;				// 1
} bonusType;							// total = 10
```

| U | R | Field | Type | Size | Meaning |
|---|---|---|---|---|---|
| 0 | 2 | `topLeft.v` | `short` BE | 2 | top edge |
| 2 | 4 | `topLeft.h` | `short` BE | 2 | left edge |
| 4 | 6 | `length` | `short` BE | 2 | grease spill length in pixels (`kGreaseRt`/`kGreaseLf`); slide-rect width (`kSlider`); unused otherwise |
| 6 | 8 | `points` | `short` BE | 2 | score value for `kInvisBonus` only |
| 8 | 10 | `state` | `Boolean` | 1 | present/uncollected (1) or already taken (0) |
| 9 | 11 | `initial` | `Boolean` | 1 | state at room entry; for grease, an **inverted** "was spilled" flag |

**Note that `state` precedes `initial` here** — the opposite order from `blowerType`,
`lightType`, `applianceType` and `enemyType`. This is the single easiest byte-level mistake to
make in a port.

### Variant `d` — `transportType` (transport, class `0x31`–`0x40`)

`GliderPRO/Headers/GliderStructs.h:36-43`:

```c
typedef struct
{
	Point		topLeft;				// 4
	short		tall;					// 2 invis transport
	short		where;					// 2
	Byte		who;					// 1
	Byte		wide;					// 1
} transportType;						// total = 10
```

| U | R | Field | Type | Size | Meaning |
|---|---|---|---|---|---|
| 0 | 2 | `topLeft.v` | `short` BE | 2 | top edge |
| 2 | 4 | `topLeft.h` | `short` BE | 2 | left edge |
| 4 | 6 | `tall` | `short` BE | 2 | `kInvisTrans`: rect height in pixels. `kDeluxeTrans`: **packed pair** — high byte = width/4, low byte = height/4. Unused (0) for all other transports |
| 6 | 8 | `where` | `short` BE | 2 | destination room, encoded as `suite*100 + (floor + 8)`; `-1` = unlinked |
| 8 | 10 | `who` | `Byte` | 1 | destination object slot index 0..23; `255` = unlinked |
| 9 | 11 | `wide` | `Byte` | 1 | `kInvisTrans`: extra pixels added to the rect's right edge. `kDeluxeTrans`: **two nibbles** — high nibble = `initial` state, low nibble = current `state`. Unused (0) otherwise |

`kDeluxeTrans` is the worst offender in the whole format: it packs four logical values
(width, height, initial, state) into two fields that other transports use as a plain height and a
plain byte.

### Variant `e` — `switchType` (switches/triggers, class `0x41`–`0x49`)

`GliderPRO/Headers/GliderStructs.h:45-52`:

```c
typedef struct
{
	Point		topLeft;				// 4
	short		delay;					// 2
	short		where;					// 2
	Byte		who;					// 1
	Byte		type;					// 1
} switchType;							// total = 10
```

| U | R | Field | Type | Size | Meaning |
|---|---|---|---|---|---|
| 0 | 2 | `topLeft.v` | `short` BE | 2 | top edge |
| 2 | 4 | `topLeft.h` | `short` BE | 2 | left edge |
| 4 | 6 | `delay` | `short` BE | 2 | trigger delay in *frames*, only for `kTrigger`/`kLgTrigger`; 0 for real switches |
| 6 | 8 | `where` | `short` BE | 2 | linked room (same encoding as variant `d`); `-1` = unlinked. **For `kSoundTrigger` this is a sound resource ID instead, 3000..32767** |
| 8 | 10 | `who` | `Byte` | 1 | linked object slot 0..23; `255` = unlinked |
| 9 | 11 | `type` | `Byte` | 1 | switch action: `kToggle` 0, `kForceOn` 1, `kForceOff` 2, `kOneShot` 3 |

### Variant `f` — `lightType` (lights, class `0x51`–`0x58`)

`GliderPRO/Headers/GliderStructs.h:54-62`:

```c
typedef struct
{
	Point		topLeft;				// 4
	short		length;					// 2
	Byte		byte0;					// 1
	Byte		byte1;					// 1
	Boolean		initial;				// 1
	Boolean		state;					// 1
} lightType;							// total = 10
```

| U | R | Field | Type | Size | Meaning |
|---|---|---|---|---|---|
| 0 | 2 | `topLeft.v` | `short` BE | 2 | top edge |
| 2 | 4 | `topLeft.h` | `short` BE | 2 | left edge |
| 4 | 6 | `length` | `short` BE | 2 | **rect width in pixels** for `kFlourescent` and `kTrackLight` (it replaces the rect's `right` while `left` is still 0); written but unread for `kCeilingLight` (always 64); 0 otherwise |
| 6 | 8 | `byte0` | `Byte` | 1 | never read anywhere; always 0 in all 22 houses |
| 7 | 9 | `byte1` | `Byte` | 1 | never read anywhere; always 0 in all 22 houses |
| 8 | 10 | `initial` | `Boolean` | 1 | lit at room entry |
| 9 | 11 | `state` | `Boolean` | 1 | currently lit |

`length` **overrides the sprite's width** for these two types — it is *not* an absolute room
coordinate. In `GliderPRO/Sources/ObjectRects.c:190-198` the sequence is
`*itsRect = srcRects[what]` → `ZeroRectCorner` (so `left == 0`) → `itsRect->right = data.f.length`
→ `QOffsetRect(itsRect, topLeft.h, topLeft.v)`, so the final rect is
`left = topLeft.h`, `right = topLeft.h + length`, i.e. `length` is the **width**. The editor agrees:
`KeepObjectLegal` clamps with `if (topLeft.h + length > bounds.right) length = bounds.right -
topLeft.h` (`GliderPRO/Sources/HouseLegal.c:429-430`), and `ObjectEdit.c:299-301` drags `length`
directly as a marquee width.

> [verified against the shipped data] Of the 196 `kFlourescent`/`kTrackLight` records in the 22
> houses, **zero** have `topLeft.h + length > 512` (the maximum is exactly 512 = `kRoomWide`), while
> **54** have `length < topLeft.h`, which is impossible if `length` were an absolute right edge.
> Observed ranges: `kFlourescent length = 24..487`, `kTrackLight length = 64..512`.

### Variant `g` — `applianceType` (appliances, class `0x61`–`0x6E`)

`GliderPRO/Headers/GliderStructs.h:64-72`:

```c
typedef struct
{
	Point		topLeft;				// 4
	short		height;					// 2 toaster, pict ID
	Byte		byte0;					// 1
	Byte		delay;					// 1
	Boolean		initial;				// 1
	Boolean		state;					// 1
} applianceType;						// total = 10
```

| U | R | Field | Type | Size | Meaning |
|---|---|---|---|---|---|
| 0 | 2 | `topLeft.v` | `short` BE | 2 | top edge |
| 2 | 4 | `topLeft.h` | `short` BE | 2 | left edge |
| 4 | 6 | `height` | `short` BE | 2 | `kToaster`: toast launch height in pixels. `kCustomPict`: PICT resource ID (10000..32767). Unused (0) otherwise |
| 6 | 8 | `byte0` | `Byte` | 1 | `kMicrowave` only: kill-mask, bit 0 (1) = rubber bands, bit 1 (2) = battery, bit 2 (4) = foil. 0 elsewhere |
| 7 | 9 | `delay` | `Byte` | 1 | animation period seed, 0..255 (`kToaster`, `kOutlet`); ignored by the other appliances |
| 8 | 10 | `initial` | `Boolean` | 1 | running at room entry |
| 9 | 11 | `state` | `Boolean` | 1 | currently running |

### Variant `h` — `enemyType` (enemies, class `0x71`–`0x79`)

`GliderPRO/Headers/GliderStructs.h:74-82`:

```c
typedef struct
{
	Point		topLeft;					// 4
	short		length;						// 2
	Byte		delay;						// 1
	Byte		byte0;						// 1
	Boolean		initial;					// 1
	Boolean		state;						// 1
} enemyType;								// total = 10
```

| U | R | Field | Type | Size | Meaning |
|---|---|---|---|---|---|
| 0 | 2 | `topLeft.v` | `short` BE | 2 | top edge |
| 2 | 4 | `topLeft.h` | `short` BE | 2 | left edge |
| 4 | 6 | `length` | `short` BE | 2 | `kBall`: bounce height in pixels. `kDrip`: drip fall distance in pixels. `kFish`: jump height in pixels. Unused (0) otherwise |
| 6 | 8 | `delay` | `Byte` | 1 | period seed, 0..255 |
| 7 | 9 | `byte0` | `Byte` | 1 | never read; always 0 in all 22 houses |
| 8 | 10 | `initial` | `Boolean` | 1 | active at room entry |
| 9 | 11 | `state` | `Boolean` | 1 | currently active |

**`delay` and `byte0` are in the opposite order from `applianceType`.** In variant `g` the pair is
`byte0` at U6, `delay` at U7; in variant `h` it is `delay` at U6, `byte0` at U7. Any port that
shares one Go struct for both classes will silently corrupt one of them.

### Variant `i` — `clutterType` (clutter, class `0x81`–`0x8F`)

`GliderPRO/Headers/GliderStructs.h:84-88`:

```c
typedef struct
{
	Rect		bounds;						// 8
	short		pict;						// 2
} clutterType;								// total = 10
```

| U | R | Field | Type | Size | Meaning |
|---|---|---|---|---|---|
| 0 | 2 | `bounds.top` | `short` BE | 2 | rect top |
| 2 | 4 | `bounds.left` | `short` BE | 2 | rect left |
| 4 | 6 | `bounds.bottom` | `short` BE | 2 | rect bottom |
| 6 | 8 | `bounds.right` | `short` BE | 2 | rect right |
| 8 | 10 | `pict` | `short` BE | 2 | `kFlower` only: which of the 6 flower sprites, 0..5. 0 for all other clutter |

`clutterType` is byte-identical to `furnitureType`; the two are separate typedefs purely for
documentation. Note the type-punning consequence: code that reads `data.b.bounds` on a clutter
object, or `data.i.bounds` on a furniture object, works correctly by accident and the source does
exactly that in `AddTempManholeRect` and in editor resize paths.

## 1.3 Union aliasing map

This is the table to keep open while porting. Every row is one byte of the 12-byte record; the
cells show what each variant calls that byte.

| R | U | `a` blower | `b` furniture | `c` bonus | `d` transport | `e` switch | `f` light | `g` appliance | `h` enemy | `i` clutter |
|---|---|---|---|---|---|---|---|---|---|---|
| 0–1 | — | `what` | `what` | `what` | `what` | `what` | `what` | `what` | `what` | `what` |
| 2–3 | 0–1 | `topLeft.v` | `bounds.top` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `bounds.top` |
| 4–5 | 2–3 | `topLeft.h` | `bounds.left` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `bounds.left` |
| 6–7 | 4–5 | `distance` | `bounds.bottom` | `length` | `tall` | `delay` | `length` | `height` | `length` | `bounds.bottom` |
| 8–9 | 6–7 | `initial`,`state` | `bounds.right` | `points` | `where` | `where` | `byte0`,`byte1` | `byte0`,`delay` | `delay`,`byte0` | `bounds.right` |
| 10 | 8 | `vector` | `pict` (hi) | `state` | `who` | `who` | `initial` | `initial` | `initial` | `pict` (hi) |
| 11 | 9 | `tall` | `pict` (lo) | `initial` | `wide` | `type` | `state` | `state` | `state` | `pict` (lo) |

Consequences the original code actually exploits or trips over:

1. `who` sits at U8 in **both** `transportType` and `switchType`. `BringSendFrontBack`
   (`GliderPRO/Sources/Objects.c:961-979`) assigns `data.d.who` — the *transport* spelling — in the
   eight switch/trigger cases (`kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`,
   `kKnifeSwitch`, `kInvisSwitch`, `kTrigger`, `kLgTrigger`, `:971-972`) and `data.e.who` — the
   *switch* spelling — in the `default:` case that catches the transports (`:976-977`). Both
   spellings are backwards relative to every other site in the code, and it still works because
   both names resolve to record byte 10.
2. `AddDynamicObject`'s `kFish` case reads `who->data.g.height`
   (`GliderPRO/Sources/Dynamics3.c`, kFish branch) on an *enemy* object. `data.g.height` and
   `data.h.length` are both at U4, so this is harmless aliasing — but a Go port with separate
   typed structs will crash or read zero unless the aliasing is reproduced.
3. `IsThisValid` (`GliderPRO/Sources/Objects.c:89-122`) reads `data.c.state` for the bonus class
   only, so all other classes are "valid" as long as `what != -1`.

## 1.4 The on-disk image is the raw big-endian struct

`ReadHouse` (`GliderPRO/Sources/HouseIO.c:315-443`) does:

1. `GetEOF(houseRefNum, &byteCount)` — take the data fork's whole length.
2. `thisHouse = (houseHand)NewHandle(byteCount)` — allocate exactly that.
3. `FSRead(houseRefNum, &byteCount, *thisHouse)` — one read of the entire fork.
4. `numberRooms = (*thisHouse)->nRooms` — start using it as a `houseType` immediately.

There is **no serialization layer, no field-by-field read, and no byte swapping anywhere in the
program.** The file *is* a memory image of `houseType` as laid out by MPW/CodeWarrior on a
big-endian 68k/PPC. `WriteHouse` (`GliderPRO/Sources/HouseIO.c:448-519`) is the mirror image: it
writes `GetHandleSize((Handle)thisHouse)` bytes straight out.

So the full on-disk layout is:

```
offset  size            field
     0     2   short    version                (always 0x0200 in shipped houses)
     2     2   short    unusedShort
     4     4   long     timeStamp              (bit 0 = "locked" flag, see below)
     8     4   long     flags                  (bit 0 = wardBit, bit 1 = phoneBit, bit 2 clear = bannerStarCountOn)
    12     4   Point    initial                (v, h)
    16   256   Str255   banner
   272   256   Str255   trailer
   528   292   scoresType highScores
   820    40   gameType   savedGame
   860     1   Boolean  hasGame
   861     1   Boolean  unusedBoolean
   862     2   short    firstRoom
   864     2   short    nRooms
   866   348*n roomType rooms[nRooms]
```

(`GliderPRO/Headers/GliderStructs.h:182-198`; the 866-byte header size is the author's own
comment `// total = 866 +`.)

And `roomType`, from the compiled probe:

```
roomType offsets:
  name         @   0   Str27, 28 bytes  (1 length byte + 27 chars)
  bounds       @  28   short
  leftStart    @  30   Byte
  rightStart   @  31   Byte
  unusedByte   @  32   Byte
  visited      @  33   Boolean
  background   @  34   short   (PICT ID; 2000..2017 built-in, 3000+ user)
  tiles        @  36   short[8]  (16 bytes)
  floor        @  52   short
  suite        @  54   short
  openings     @  56   short
  numObjects   @  58   short
  objects      @  60   objectType[24]  (288 bytes)
                348   total
```

So **object *i* of room *r* begins at file offset `866 + r*348 + 60 + i*12`** and is 12 bytes long.

## 1.5 Empirical verification across all 22 shipped houses

Decoded from `GliderPRO/Houses/*.binhex` with `/tmp/wf-objtax/binhex2.py`. Every one is Mac file
type `gliH`, creator `ozm5`, and carries `version == 0x0200` (`kHouseVersion`,
`GliderPRO/Headers/GliderDefines.h:517`).

Arithmetic check `866 + nRooms*348 == len(dataFork)`:

| House | data-fork bytes | nRooms | 866 + 348·n | delta |
|---|---|---|---|---|
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

21 of 22 match to the byte. The 866-byte header and 348-byte room stride are therefore proven,
not inferred. (`Sampler` has 2 trailing bytes of slop — see Open questions.)

`Demo House` is 16526 bytes with 45 rooms, which exactly matches the hard-coded assertion in the
demo build at `GliderPRO/Sources/HouseIO.c:349`.

Across the 22 houses: **4070 rooms, 31 440 live object records, 66 240 empty slots
(`what == -1`), and zero unrecognised `what` values.** Every one of the 117 defined IDs occurs at
least once in shipped content.

## 1.6 Worked byte-level examples

First-seen raw 12-byte record for a representative from each class, taken directly from the
decoded data forks. Bytes are shown in file order (big-endian).

**`kFloorVent` (0x01)** — Art Museum, room 0, slot 1: `00 01 01 31 00 73 01 0a 01 01 01 00`

| bytes | field | value |
|---|---|---|
| `0001` | `what` | 1 = `kFloorVent` |
| `0131` | `data.a.topLeft.v` | 305 = `kFloorVentTop` |
| `0073` | `data.a.topLeft.h` | 115 |
| `010a` | `data.a.distance` | 266 |
| `01` | `data.a.initial` | true |
| `01` | `data.a.state` | true |
| `01` | `data.a.vector` | 1 = up |
| `00` | `data.a.tall` | 0 (unused) |

**`kTable` (0x11)** — Art Museum, room 56, slot 7: `00 11 00 f6 01 90 00 fe 01 d0 00 00`

| bytes | field | value |
|---|---|---|
| `0011` | `what` | 17 = `kTable` |
| `00f6` | `data.b.bounds.top` | 246 |
| `0190` | `data.b.bounds.left` | 400 |
| `00fe` | `data.b.bounds.bottom` | 254 |
| `01d0` | `data.b.bounds.right` | 464 |
| `0000` | `data.b.pict` | 0 (unused) |

Height 254 − 246 = 8 = `kTableThick`, width 464 − 400 = 64 = `kTileWide`: the default table.

**`kInvisBonus` (0x2B)** — Art Museum, room 0, slot 3: `00 2b 00 e0 00 f6 00 00 01 f4 00 01`

| bytes | field | value |
|---|---|---|
| `002b` | `what` | 43 = `kInvisBonus` |
| `00e0` | `data.c.topLeft.v` | 224 |
| `00f6` | `data.c.topLeft.h` | 246 |
| `0000` | `data.c.length` | 0 (unused) |
| `01f4` | `data.c.points` | **500** |
| `00` | `data.c.state` | false (already taken when saved) |
| `01` | `data.c.initial` | true |

**`kFloorTrans` (0x35), linked** — Art Museum, room 0, slot 4: `00 35 01 2e 00 e9 00 00 18 a6 04 00`

| bytes | field | value |
|---|---|---|
| `0035` | `what` | 53 = `kFloorTrans` |
| `012e` | `data.d.topLeft.v` | 302 = `kFloorTransTop` |
| `00e9` | `data.d.topLeft.h` | 233 |
| `0000` | `data.d.tall` | 0 (unused for this type) |
| `18a6` | `data.d.where` | 6310 → suite 63, floor 10 − 8 = 2 |
| `04` | `data.d.who` | 4 |
| `00` | `data.d.wide` | 0 (unused) |

**`kUpStairs` (0x31), unlinked** — CD Demo House, room 192, slot 1: `00 31 00 1c 01 33 00 00 ff ff ff 00`

`where = 0xFFFF = -1` and `who = 0xFF = 255`: the canonical "not linked" encoding.
`topLeft.v = 0x001C = 28 = kStairsTop`.

**`kMailboxLf` (0x33), linked** — CD Demo House, room 69, slot 0: `00 33 00 1b 01 a2 00 00 02 c5 06 00`

`where = 0x02C5 = 709` → suite 7, floor 9 − 8 = 1. `who = 6`.

## 1.7 Empty slots and `numObjects`

`kMaxRoomObs` is 24 (`GliderPRO/Headers/GliderDefines.h:250`):

```c
#define kMaxRoomObs					24
```

Every room carries a fixed array of 24 records — 288 bytes — whether or not they are used. There
is no compaction and no free-list: `FindEmptyObjectSlot`
(`GliderPRO/Sources/ObjectAdd.c:806-820`) linearly scans slots 0..23 for the first
`what == kObjectIsEmpty`, and returns `-1` if none is free.

`numObjects` (`roomType` offset 58) is documented nowhere but behaves as a **high-water mark, not
a count**. Measured over all 4070 shipped rooms:

- `numObjects == (highest used slot index) + 1` in **4064 of 4070** rooms;
- **6 rooms have a hole** — an empty slot below a used one — and in exactly those 6 rooms
  `numObjects != liveCount`;
- histogram of `numObjects`: `{0: 840, 1: 588, 2: 132, 3: 150, 4: 124, 5: 114, 6: 140, 7: 155,
  8: 164, 9: 154, 10: 147, 11: 146, 12: 161, 13: 116, 14: 114, 15: 127, 16: 85, 17: 81, 18: 59,
  19: 50, 20: 39, 21: 49, 22: 53, 23: 60, 24: 222}` — 222 rooms are completely full.

The runtime never trusts `numObjects` as a count anyway: `ListOneRoomsObjects`
(`GliderPRO/Sources/Objects.c:254-296`) and `DrawARoomsObjects`
(`GliderPRO/Sources/ObjectDrawAll.c:23`) both iterate all 24 slots and filter with
`IsThisValid`. **A Go port should do the same and treat `numObjects` as advisory metadata only.**

`IsThisValid` (`GliderPRO/Sources/Objects.c:89-122`) is a single `switch` on
`(*thisHouse)->rooms[where].objects[who].what` — it reads the **house handle**, not the `thisRoom`
cache, and it does **not** validate `where` or `who` at all:

1. `itsGood = true` (`:94`).
2. `switch ((*thisHouse)->rooms[where].objects[who].what)`:
   - `case kObjectIsEmpty:` (−1) → `itsGood = false` (`:100-102`).
   - the 12 *collectible* prizes — `kRedClock`, `kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`,
     `kBattery`, `kBands`, `kFoil`, `kInvisBonus`, `kStar`, **`kSparkle`**, `kHelium` (`:104-117`) →
     `itsGood = data.c.state`. Note `kGreaseRt`/`kGreaseLf` are **not** in this list even though
     `SetObjectState`'s prize arm includes them, so spilled grease stays "valid".
   - no `default:` → every other type is valid.

That is the only place a *field* (not just `what`) decides validity. Because there is no bounds
check, a corrupt `where`/`who` reads out of the house handle; a port should bounds-check and return
false.

## 1.8 The master-object list: 9 rooms at once

`kMaxMasterObjects` is 216 (`GliderPRO/Headers/GliderDefines.h:266`):

```c
#define kMaxMasterObjects			216		// kMaxRoomObs * 9
```

The playfield shows the current room plus its 8 neighbours, so at most 9 × 24 = 216 objects can
be simultaneously live. `ListAllLocalObjects` (`GliderPRO/Sources/Objects.c:300-347`) fills
`masterObjects[]` in a fixed neighbour order:

1. central room always;
2. if `numNeighbors > 1`: east, west;
3. if `numNeighbors > 3`: north, north-east, south-east, south, south-west, north-west.

Then it walks the assembled list a second time and, for each object that has a link, records
`localLink` = the index in `masterObjects[]` of the linked object, or leaves it at -1 if the
target is outside the 9-room window.

Each entry is an `objDataType` (`GliderPRO/Headers/GliderStructs.h:322-332`):

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

Critically, `theObject` is a **copy**, not a pointer. Runtime state changes therefore have to be
written back to the room record explicitly (which `SetObjectState` does — see §6).

**The list is dense with stride 24.** `ListOneRoomsObjects`
(`GliderPRO/Sources/Objects.c:254-296`) loops `for (n = 0; n < kMaxRoomObs; n++)` and copies
**every** slot including the empty ones. The copy and the `numMasterObjects++` (`:289`) are guarded
by `if (numMasterObjects < kMaxMasterObjects)`, which can never fail at 9 rooms × 24 slots = exactly
216, so in practice the increment happens for every slot. So the mapping from (neighbour, slot) to
master index is exactly:

```
masterIndex = neighbourOrdinal * 24 + slot
```

with `neighbourOrdinal` following the fill order above (central = 0, east = 1, west = 2, north = 3,
north-east = 4, south-east = 5, south = 6, south-west = 7, north-west = 8). This is the single most
important invariant in the whole subsystem, because several places in the original abuse it:
`masterObjects[i]` with `i` a *room slot index* silently resolves to the **central room's** entry
regardless of which neighbour is being processed. That is correct for the central room and wrong for
the other eight (see §14 items 6 and 9).

Two consequences a port must honour:

- `numMasterObjects` is `24 * numRoomsListed` (24, 72, or 216), never the number of *live* objects.
- `hotNum` is only ever set for the central room:
  `if ((where == kCentralRoom) && (IsThisValid(roomNum, n))) hotNum = CreateActiveRects(n); else hotNum = -1;`
  (`GliderPRO/Sources/Objects.c:283-286`). Neighbour rooms are drawn but not interactive.
- `numLocalMasterObjects` counts only the central room's 24 entries
  (`GliderPRO/Sources/Objects.c:291-292`).

---

# 2. The ID space

## 2.1 Class ranges

All object IDs are `#define`d in `GliderPRO/Headers/GliderDefines.h:311-437` as hexadecimal
literals, in nine contiguous runs. The class of an object is determined **entirely by which run
its ID falls in** — there is no class field. Every `switch (what)` in the program enumerates the
members explicitly; nothing does arithmetic on the high nibble. (A port *may* use the high nibble
as a fast class test, but must handle the holes.)

| Class | ID range | Count | Union variant | Palette mode | Source lines |
|---|---|---|---|---|---|
| Blowers | `0x01`–`0x10` (1–16) | 16 | `a` `blowerType` | `kBlowerMode` = 1 | `GliderDefines.h:311-326` |
| Furniture | `0x11`–`0x1F` (17–31) | 15 | `b` `furnitureType` | `kFurnitureMode` = 2 | `GliderDefines.h:328-342` |
| Prizes / bonuses | `0x21`–`0x2F` (33–47) | 15 | `c` `bonusType` | `kBonusMode` = 3 | `GliderDefines.h:344-358` |
| Transport | `0x31`–`0x40` (49–64) | 16 | `d` `transportType` | `kTransportMode` = 4 | `GliderDefines.h:360-375` |
| Switches | `0x41`–`0x49` (65–73) | 9 | `e` `switchType` | `kSwitchMode` = 5 | `GliderDefines.h:377-385` |
| Lights | `0x51`–`0x58` (81–88) | 8 | `f` `lightType` | `kLightMode` = 6 | `GliderDefines.h:387-394` |
| Appliances | `0x61`–`0x6E` (97–110) | 14 | `g` `applianceType` | `kApplianceMode` = 7 | `GliderDefines.h:396-409` |
| Enemies | `0x71`–`0x79` (113–121) | 9 | `h` `enemyType` | `kEnemyMode` = 8 | `GliderDefines.h:411-419` |
| Clutter | `0x81`–`0x8F` (129–143) | 15 | `i` `clutterType` | `kClutterMode` = 9 | `GliderDefines.h:421-435` |

**Total: 117 defined object types.** The palette modes (with `kSelectTool` = 0) are at
`GliderPRO/Headers/GliderDefines.h:270-280`.

The array bound is `kNumSrcRects` (`GliderPRO/Headers/GliderDefines.h:437`):

```c
#define kNumSrcRects				0x90
```

= 144 = `kChimes + 1`. Both `srcRects[]` and the `STR# 1007` name list are 144 entries, indexed
directly by `what`, so slot 0 and every hole are wasted.

**Holes in the ID space** (27 unused values in `0x00`–`0x8F`):
`0x00`, `0x20`, `0x30`, `0x4A`–`0x50` (7), `0x59`–`0x60` (8), `0x6F`–`0x70` (2), `0x7A`–`0x80` (7).
1 + 1 + 1 + 7 + 8 + 2 + 7 = 27, and 144 − 27 = 117 defined types, which is the class-table total
above.

`srcRects` is allocated with `NewPtr`, **not** `NewPtrClear`
(`GliderPRO/Sources/StructuresInit2.c:271`):

```c
srcRects = (Rect *)NewPtr(sizeof(Rect) * kNumSrcRects);
```

`InitSrcRects` (`GliderPRO/Sources/StructuresInit2.c:306-475`) fills only **116** of the 144 slots.
The 27 unused indices (slot 0 among them) **and slot `0x85` (`kFlower`)** are left as uninitialised
heap garbage — 144 − 27 − 1 = 116, counted class by class as 16 + 15 + 15 + 16 + 9 + 8 + 14 + 9 + 14
in the §3 tables. `kFlower`
never reads `srcRects` — it uses `flowerSrc[data.i.pict]` instead — so no bug results, but a port
must not expect a `srcRects[kFlower]` entry to exist.

## 2.2 The complete ID → name table

The user-visible name of an object is `GetIndString(str, kObjectNameStrings, what)` — the object ID
used **directly as the 1-based `STR#` index**. `kObjectNameStrings` is 1007
(`GliderPRO/Headers/GliderDefines.h:461`). This call appears 13 times in `ObjectInfo.c`, once per
Info dialog: lines 945, 1135, 1184, 1280, 1403, 1558, 1645, 1760, 1884, 1958, 2071, 2195, 2304.

I extracted `STR# 1007` ("Object Names") from `GliderPRO/Glider PRO.r` and decoded it: a
count prefix of `0x0090` = **144**, followed by 144 Pascal strings, total 1569 bytes, consuming the
resource exactly. Names for the hole indices are one-character junk placeholders (`f`, `9`,
`a`…`e`) — a hand-maintained list that the author padded to keep the indices aligned.

| ID | Constant | STR# 1007 name |
|---|---|---|
| `0x01` | `kFloorVent` | Floor Vent |
| `0x02` | `kCeilingVent` | Ceiling Vent |
| `0x03` | `kFloorBlower` | Floor Duct |
| `0x04` | `kCeilingBlower` | Ceiling Duct |
| `0x05` | `kSewerGrate` | Sewer Grate |
| `0x06` | `kLeftFan` | Table Fan |
| `0x07` | `kRightFan` | Table Fan |
| `0x08` | `kTaper` | Taper |
| `0x09` | `kCandle` | Simple Candle |
| `0x0A` | `kStubby` | Stubby Candle |
| `0x0B` | `kTiki` | Tiki Torch |
| `0x0C` | `kBBQ` | Barbecue Grill |
| `0x0D` | `kInvisBlower` | Invisible Blower |
| `0x0E` | `kGrecoVent` | Greco-Roman Vent |
| `0x0F` | `kSewerBlower` | Sewer Blower |
| `0x10` | `kLiftArea` | Lift Area |
| `0x11` | `kTable` | Table |
| `0x12` | `kShelf` | Shelf |
| `0x13` | `kCabinet` | Cabinet |
| `0x14` | `kFilingCabinet` | Filing Cabinet |
| `0x15` | `kWasteBasket` | Wastebasket |
| `0x16` | `kMilkCrate` | Milk Crate |
| `0x17` | `kCounter` | Counter |
| `0x18` | `kDresser` | Dresser |
| `0x19` | `kDeckTable` | Deck Table |
| `0x1A` | `kStool` | Bar Stool |
| `0x1B` | `kTrunk` | Steamer Trunk |
| `0x1C` | `kInvisObstacle` | Invisible Obstacle |
| `0x1D` | `kManhole` | Manhole |
| `0x1E` | `kBooks` | Books |
| `0x1F` | `kInvisBounce` | Invisible Rebounder |
| `0x20` | *(hole)* | `f` |
| `0x21` | `kRedClock` | Digital Clock |
| `0x22` | `kBlueClock` | Wall Clock |
| `0x23` | `kYellowClock` | Alarm Clock |
| `0x24` | `kCuckoo` | Cuckoo Clock |
| `0x25` | `kPaper` | Extra Glider |
| `0x26` | `kBattery` | Battery |
| `0x27` | `kBands` | Rubber Bands (8) |
| `0x28` | `kGreaseRt` | Grease (spills rt.) |
| `0x29` | `kGreaseLf` | Grease (spills lf.) |
| `0x2A` | `kFoil` | Aluminum Foil |
| `0x2B` | `kInvisBonus` | Invisible Bonus |
| `0x2C` | `kStar` | Magic Star |
| `0x2D` | `kSparkle` | Sparkle |
| `0x2E` | `kHelium` | Helium (He) |
| `0x2F` | `kSlider` | Slide Rect |
| `0x30` | *(hole)* | `f` |
| `0x31` | `kUpStairs` | Up Stairs |
| `0x32` | `kDownStairs` | Down Stairs |
| `0x33` | `kMailboxLf` | Mailbox (faces lf.) |
| `0x34` | `kMailboxRt` | Mailbox (faces rt.) |
| `0x35` | `kFloorTrans` | Floor Trans. Duct |
| `0x36` | `kCeilingTrans` | Ceiling Trans. Duct |
| `0x37` | `kDoorInLf` | Door (interior) |
| `0x38` | `kDoorInRt` | Door (interior) |
| `0x39` | `kDoorExRt` | Door (exterior) |
| `0x3A` | `kDoorExLf` | Door (exterior) |
| `0x3B` | `kWindowInLf` | Window (interior) |
| `0x3C` | `kWindowInRt` | Window (interior) |
| `0x3D` | `kWindowExRt` | Window (exterior) |
| `0x3E` | `kWindowExLf` | Window (exterior) |
| `0x3F` | `kInvisTrans` | Invisible Transport |
| `0x40` | `kDeluxeTrans` | Deluxe Transport |
| `0x41` | `kLightSwitch` | Light Switch |
| `0x42` | `kMachineSwitch` | Machine Switch |
| `0x43` | `kThermostat` | Thermostat |
| `0x44` | `kPowerSwitch` | Digital Switch |
| `0x45` | `kKnifeSwitch` | Knife Switch |
| `0x46` | `kInvisSwitch` | Invisible Switch |
| `0x47` | `kTrigger` | Trigger |
| `0x48` | `kLgTrigger` | Large Trigger |
| `0x49` | `kSoundTrigger` | Sound Trigger |
| `0x4A`–`0x50` | *(holes)* | `9`,`a`,`b`,`c`,`d`,`e`,`f` |
| `0x51` | `kCeilingLight` | Ceiling Light |
| `0x52` | `kLightBulb` | Simple Bulb |
| `0x53` | `kTableLamp` | Table Lamp |
| `0x54` | `kHipLamp` | Hip Pole Lamp |
| `0x55` | `kDecoLamp` | Deco Lamp |
| `0x56` | `kFlourescent` | Flourescent Light |
| `0x57` | `kTrackLight` | Track Lighting |
| `0x58` | `kInvisLight` | Invisible Light |
| `0x59`–`0x60` | *(holes)* | `8`,`9`,`a`,`b`,`c`,`d`,`e`,`f` |
| `0x61` | `kShredder` | Paper Shredder |
| `0x62` | `kToaster` | Toaster |
| `0x63` | `kMacPlus` | Mac Plus |
| `0x64` | `kGuitar` | Guitar |
| `0x65` | `kTV` | T.V. |
| `0x66` | `kCoffee` | Coffee Machine |
| `0x67` | `kOutlet` | Electrical Outlet |
| `0x68` | `kVCR` | VCR |
| `0x69` | `kStereo` | Stereo System |
| `0x6A` | `kMicrowave` | Microwave Oven |
| `0x6B` | `kCinderBlock` | Cinder Block |
| `0x6C` | `kFlowerBox` | Flower Box |
| `0x6D` | `kCDs` | Compact Discs |
| `0x6E` | `kCustomPict` | Custom Picture |
| `0x6F`–`0x70` | *(holes)* | `e`,`f` |
| `0x71` | `kBalloon` | Balloon |
| `0x72` | `kCopterLf` | 'Copter (lf. drift) |
| `0x73` | `kCopterRt` | 'Copter (rt. drift) |
| `0x74` | `kDartLf` | Dart (lf. moving) |
| `0x75` | `kDartRt` | Dart (rt. moving) |
| `0x76` | `kBall` | Bouncing Ball |
| `0x77` | `kDrip` | Water Drip |
| `0x78` | `kFish` | Fish Bowl & Fish |
| `0x79` | `kCobweb` | Cobweb |
| `0x7A`–`0x80` | *(holes)* | `9`,`a`,`b`,`c`,`d`,`e`,`f` |
| `0x81` | `kOzma` | Ozma |
| `0x82` | `kMirror` | Mirror |
| `0x83` | `kMousehole` | Mouse Hole |
| `0x84` | `kFireplace` | Fireplace |
| `0x85` | `kFlower` | Flower |
| `0x86` | `kWallWindow` | Window (closed) |
| `0x87` | `kBear` | Teddy Bear |
| `0x88` | `kCalendar` | Calendar |
| `0x89` | `kVase1` | Broad Vase |
| `0x8A` | `kVase2` | Narrow Vase |
| `0x8B` | `kBulletin` | Bulletin Board |
| `0x8C` | `kCloud` | Cloud |
| `0x8D` | `kFaucet` | Faucet |
| `0x8E` | `kRug` | Throw Rug |
| `0x8F` | `kChimes` | Wind Chimes |
| `0x90` | *(no object)* | **Mermaid** — index 144 exists in the resource but there is no `0x90` object type. A cut feature. |

## 2.3 Population in shipped content

Live-record counts across the 22 shipped houses (31 440 objects, 4070 rooms). Sorted descending;
this is the practical priority order for a port.

| Rank | ID | Constant | Count |
|---|---|---|---|
| 1 | `0x6E` | `kCustomPict` | 4782 |
| 2 | `0x0D` | `kInvisBlower` | 2336 |
| 3 | `0x8C` | `kCloud` | 1860 |
| 4 | `0x58` | `kInvisLight` | 1764 |
| 5 | `0x1F` | `kInvisBounce` | 1670 |
| 6 | `0x01` | `kFloorVent` | 1458 |
| 7 | `0x10` | `kLiftArea` | 716 |
| 8 | `0x82` | `kMirror` | 667 |
| 9 | `0x1C` | `kInvisObstacle` | 666 |
| 10 | `0x46` | `kInvisSwitch` | 635 |
| 11 | `0x85` | `kFlower` | 547 |
| 12 | `0x05` | `kSewerGrate` | 507 |
| 13 | `0x71` | `kBalloon` | 500 |
| 14 | `0x2D` | `kSparkle` | 486 |
| 15 | `0x77` | `kDrip` | 477 |
| 16 | `0x36` | `kCeilingTrans` | 465 |
| 17 | `0x13` | `kCabinet` | 457 |
| 18 | `0x12` | `kShelf` | 389 |
| 19 | `0x3F` | `kInvisTrans` | 385 |
| 20 | `0x2F` | `kSlider` | 354 |

The rarest types (each present, so all 117 are exercised by shipped content): `kDoorInRt` 11,
`kDoorExLf` 11, `kCeilingBlower` 12, `kWindowExRt` 19, `kDoorExRt` 21, `kWindowInLf` 21,
`kDoorInLf` 23, `kVCR` 26, `kHipLamp` 26, `kCeilingVent` 28, `kFaucet` 28, `kGuitar` 29,
`kWindowExLf` 29, `kDeckTable` 30, `kWindowInRt` 32, `kFireplace` 33.

---

# 3. Default sizes: `srcRects[144]`

`InitSrcRects` (`GliderPRO/Sources/StructuresInit2.c:306-475`) is a single 170-line function that
sets one `Rect` per object type. **The rect serves double duty:**

- its **size** (width × height) is the object's default/native size;
- its **origin** (left, top) is the sprite's position inside the class's PICT sheet.

Consumers that want the size call `ZeroRectCorner` first
(`GliderPRO/Sources/RectUtils.c:57-63`):

```c
void ZeroRectCorner (Rect *theRect)		// Offset rect to (0, 0)
{
	theRect->right -= theRect->left;
	theRect->bottom -= theRect->top;
	theRect->left = 0;
	theRect->top = 0;
}
```

Consumers that want to blit call the rect directly as the source rect. Helper primitives, all in
`GliderPRO/Sources/RectUtils.c`: `QSetRect(r, l, t, right, bottom)` at :210-216 (note the
**left, top, right, bottom** argument order, unlike the struct's top-first layout) and
`QOffsetRect(r, h, v)` at :197-203.

The full table. "W×H" is the default size; "sheet (l,t)" is the sprite origin in the PICT sheet;
the PICT sheet ID per class is in §3.2.

### Blowers (sheet `kBlowerPictID` 4000, 48 × 402)

| ID | Constant | W×H | sheet (l,t) | Line |
|---|---|---|---|---|
| `0x01` | `kFloorVent` | 48 × 11 | (0, 0) | :308-309 |
| `0x02` | `kCeilingVent` | 48 × 11 | (0, 11) | :310-311 |
| `0x03` | `kFloorBlower` | 48 × 15 | (0, 22) | :312-313 |
| `0x04` | `kCeilingBlower` | 48 × 15 | (0, 37) | :314-315 |
| `0x05` | `kSewerGrate` | 48 × 17 | (0, 52) | :316-317 |
| `0x06` | `kLeftFan` | 40 × 55 | (0, 69) | :318-319 |
| `0x07` | `kRightFan` | 40 × 55 | (0, 124) | :320-321 |
| `0x08` | `kTaper` | 20 × 59 | (0, 209) | :322-323 |
| `0x09` | `kCandle` | 32 × 30 | (0, 179) | :324-325 |
| `0x0A` | `kStubby` | 20 × 36 | (0, 268) | :326-327 |
| `0x0B` | `kTiki` | 27 × 28 | (21, 268) | :328-329 |
| `0x0C` | `kBBQ` | 64 × 33 | (0, 0) | :330 |
| `0x0D` | `kInvisBlower` | 24 × 24 | (0, 0) | :331 |
| `0x0E` | `kGrecoVent` | 48 × 18 | (0, 340) | :332-333 |
| `0x0F` | `kSewerBlower` | 32 × 12 | (0, 390) | :334-335 |
| `0x10` | `kLiftArea` | 64 × 32 | (0, 0) | :336 |

`kBBQ`, `kInvisBlower` and `kLiftArea` get no `QOffsetRect` — the first because its sprite is
drawn from a different path, the last two because they are invisible.

### Furniture (sheet `kFurniturePictID` 4001, 64 × 278)

| ID | Constant | W×H | sheet (l,t) | Line |
|---|---|---|---|---|
| `0x11` | `kTable` | 64 × 8 (`kTableThick`) | (0, 0) | :338 |
| `0x12` | `kShelf` | 64 × 6 (`kShelfThick`) | (0, 0) | :339 |
| `0x13` | `kCabinet` | 64 × 64 | (0, 0) | :340 |
| `0x14` | `kFilingCabinet` | 74 × 107 | (0, 0) | :341 |
| `0x15` | `kWasteBasket` | 64 × 61 | (0, 43) | :342-343 |
| `0x16` | `kMilkCrate` | 64 × 58 | (0, 104) | :344-345 |
| `0x17` | `kCounter` | 128 × 64 | (0, 0) | :346 |
| `0x18` | `kDresser` | 128 × 64 | (0, 0) | :347 |
| `0x19` | `kDeckTable` | 64 × 8 (`kTableThick`) | (0, 0) | :348 |
| `0x1A` | `kStool` | 48 × 38 | (0, 183) | :349-350 |
| `0x1B` | `kTrunk` | 144 × 80 | (0, 0) | :351 |
| `0x1C` | `kInvisObstacle` | 64 × 64 | (0, 0) | :352 |
| `0x1D` | `kManhole` | 123 × 22 | (0, 0) | :353 |
| `0x1E` | `kBooks` | 64 × 51 | (0, 0) | :354 |
| `0x1F` | `kInvisBounce` | 64 × 64 | (0, 0) | :355 |

`kTableThick` = 8 and `kShelfThick` = 6 are at `GliderPRO/Headers/GliderDefines.h:439-440`.

### Prizes / bonuses (sheet `kBonusPictID` 4002, 88 × 378)

| ID | Constant | W×H | sheet (l,t) | Line |
|---|---|---|---|---|
| `0x21` | `kRedClock` | 28 × 17 | (0, 0) | :357 |
| `0x22` | `kBlueClock` | 28 × 25 | (0, 17) | :358-359 |
| `0x23` | `kYellowClock` | 28 × 28 | (0, 42) | :360-361 |
| `0x24` | `kCuckoo` | 40 × 80 | (0, 148) | :362-363 |
| `0x25` | `kPaper` | 48 × 21 | (0, 127) | :364-365 |
| `0x26` | `kBattery` | 16 × 25 | (32, 0) | :366-367 |
| `0x27` | `kBands` | 28 × 23 | (20, 70) | :368-369 |
| `0x28` | `kGreaseRt` | 32 × 27 | (0, 243) | :370-371 |
| `0x29` | `kGreaseLf` | 32 × 27 | (0, 324) | :372-373 |
| `0x2A` | `kFoil` | 55 × 15 | (0, 228) | :374-375 |
| `0x2B` | `kInvisBonus` | 24 × 24 | (0, 0) | :376 |
| `0x2C` | `kStar` | 32 × 31 | (48, 0) | :377-378 |
| `0x2D` | `kSparkle` | 20 × 19 | (0, 70) | :379-380 |
| `0x2E` | `kHelium` | 56 × 16 | (32, 270) | :381-382 |
| `0x2F` | `kSlider` | 64 × 16 | (0, 0) | :383 |

### Transport (sheet `kTransportPictID` 4008 for the ducts, 56 × 32; stairs/doors/windows/mailboxes come from house-specific PICTs)

| ID | Constant | W×H | sheet (l,t) | Line |
|---|---|---|---|---|
| `0x31` | `kUpStairs` | 160 × 267 | (0, 0) | :385 |
| `0x32` | `kDownStairs` | 160 × 267 | (0, 0) | :386 |
| `0x33` | `kMailboxLf` | 94 × 80 | (0, 0) | :387 |
| `0x34` | `kMailboxRt` | 94 × 80 | (0, 0) | :388 |
| `0x35` | `kFloorTrans` | 56 × 15 | (0, 1) | :389-390 |
| `0x36` | `kCeilingTrans` | 56 × 15 | (0, 16) | :391-392 |
| `0x37` | `kDoorInLf` | 144 × 322 | (0, 0) | :393 |
| `0x38` | `kDoorInRt` | 144 × 322 | (0, 0) | :394 |
| `0x39` | `kDoorExRt` | 16 × 322 | (0, 0) | :395 |
| `0x3A` | `kDoorExLf` | 16 × 322 | (0, 0) | :396 |
| `0x3B` | `kWindowInLf` | 20 × 170 | (0, 0) | :397 |
| `0x3C` | `kWindowInRt` | 20 × 170 | (0, 0) | :398 |
| `0x3D` | `kWindowExRt` | 16 × 170 | (0, 0) | :399 |
| `0x3E` | `kWindowExLf` | 16 × 170 | (0, 0) | :400 |
| `0x3F` | `kInvisTrans` | 64 × 32 | (0, 0) | :401 |
| `0x40` | `kDeluxeTrans` | 64 × 64 | (0, 0) | :402 |

The doors' 322-pixel height is exactly `kTileHigh` — a door spans the full room height.

### Switches (sheet `kSwitchPictID` 4003, 32 × 104)

| ID | Constant | W×H | sheet (l,t) | Line |
|---|---|---|---|---|
| `0x41` | `kLightSwitch` | 15 × 24 | (0, 0) | :404 |
| `0x42` | `kMachineSwitch` | 16 × 24 | (0, 48) | :405-406 |
| `0x43` | `kThermostat` | 15 × 24 | (0, 48) | :407-408 |
| `0x44` | `kPowerSwitch` | 8 × 8 | (0, 72) | :409-410 |
| `0x45` | `kKnifeSwitch` | 16 × 24 | (0, 80) | :411-412 |
| `0x46` | `kInvisSwitch` | 12 × 12 | (0, 0) | :413 |
| `0x47` | `kTrigger` | 12 × 12 | (0, 0) | :414 |
| `0x48` | `kLgTrigger` | 48 × 48 | (0, 0) | :415 |
| `0x49` | `kSoundTrigger` | 32 × 32 | (0, 0) | :416 |

`kMachineSwitch` and `kThermostat` share sheet origin (0, 48) with different widths — the
thermostat is the same art one pixel narrower.

### Lights (sheet `kLightPictID` 4004, 72 × 126)

| ID | Constant | W×H | sheet (l,t) | Line |
|---|---|---|---|---|
| `0x51` | `kCeilingLight` | 64 × 20 | (0, 0) | :418-419 |
| `0x52` | `kLightBulb` | 16 × 28 | (0, 20) | :420-421 |
| `0x53` | `kTableLamp` | 48 × 70 | (16, 20) | :422-423 |
| `0x54` | `kHipLamp` | 72 × 276 | (0, 0) | :424 |
| `0x55` | `kDecoLamp` | 64 × 212 | (0, 0) | :425 |
| `0x56` | `kFlourescent` | 64 × 12 | (0, 0) | :426 |
| `0x57` | `kTrackLight` | 64 × 24 | (0, 0) | :427 |
| `0x58` | `kInvisLight` | 16 × 16 | (0, 0) | :428 |

### Appliances (sheet `kAppliancePictID` 4005, 80 × 269)

| ID | Constant | W×H | sheet (l,t) | Line |
|---|---|---|---|---|
| `0x61` | `kShredder` | 73 × 22 | (0, 0) | :430 |
| `0x62` | `kToaster` | 48 × 27 | (0, 22) | :431-432 |
| `0x63` | `kMacPlus` | 48 × 58 | (0, 49) | :433-434 |
| `0x64` | `kGuitar` | 64 × 172 | (0, 0) | :435 |
| `0x65` | `kTV` | 92 × 77 | (0, 0) | :436 |
| `0x66` | `kCoffee` | 43 × 64 | (0, 107) | :437-438 |
| `0x67` | `kOutlet` | 16 × 24 | (64, 22) | :439-440 |
| `0x68` | `kVCR` | 96 × 22 | (0, 0) | :441 |
| `0x69` | `kStereo` | 128 × 53 | (0, 0) | :442 |
| `0x6A` | `kMicrowave` | 92 × 59 | (0, 0) | :443 |
| `0x6B` | `kCinderBlock` | 40 × 62 | (0, 0) | :444 |
| `0x6C` | `kFlowerBox` | 80 × 32 | (0, 0) | :445 |
| `0x6D` | `kCDs` | 16 × 30 | (48, 22) | :446-447 |
| `0x6E` | `kCustomPict` | 72 × 34 | (0, 0) | :448 |

`kCustomPict`'s 72 × 34 is only a **fallback**: the real size comes from the referenced PICT's
`picFrame` (see §5, `GetObjectRect`).

### Enemies (per-type sheets, see §3.2)

| ID | Constant | W×H | sheet (l,t) | Line |
|---|---|---|---|---|
| `0x71` | `kBalloon` | 24 × 30 | (0, 0) | :450 |
| `0x72` | `kCopterLf` | 32 × 30 | (0, 0) | :451 |
| `0x73` | `kCopterRt` | 32 × 30 | (0, 0) | :452 |
| `0x74` | `kDartLf` | 64 × 19 | (0, 0) | :453 |
| `0x75` | `kDartRt` | 64 × 19 | (0, 0) | :454 |
| `0x76` | `kBall` | 32 × 32 | (0, 0) | :455 |
| `0x77` | `kDrip` | 16 × 12 | (0, 0) | :456 |
| `0x78` | `kFish` | 36 × 33 | (0, 0) | :457 |
| `0x79` | `kCobweb` | 54 × 45 | (0, 0) | :458 |

### Clutter (sheet `kClutterPictID` 4018, 128 × 69, only for `kMousehole`/`kFaucet`/`kFlower`; the rest come from house PICTs)

| ID | Constant | W×H | sheet (l,t) | Line |
|---|---|---|---|---|
| `0x81` | `kOzma` | 102 × 92 | (0, 0) | :460 |
| `0x82` | `kMirror` | 64 × 64 | (0, 0) | :461 |
| `0x83` | `kMousehole` | 10 × 11 | (0, 0) | :462 |
| `0x84` | `kFireplace` | 180 × 142 | (0, 0) | :463 |
| `0x85` | `kFlower` | **not set** | — | — |
| `0x86` | `kWallWindow` | 64 × 80 | (0, 0) | :464 |
| `0x87` | `kBear` | 56 × 58 | (0, 0) | :465 |
| `0x88` | `kCalendar` | 63 × 92 | (0, 0) | :466 |
| `0x89` | `kVase1` | 36 × 45 | (0, 0) | :467 |
| `0x8A` | `kVase2` | 35 × 57 | (0, 0) | :468 |
| `0x8B` | `kBulletin` | 80 × 58 | (0, 0) | :469 |
| `0x8C` | `kCloud` | 128 × 30 | (0, 0) | :470 |
| `0x8D` | `kFaucet` | 56 × 18 | (0, 51) | :471-472 |
| `0x8E` | `kRug` | 144 × 18 | (0, 0) | :473 |
| `0x8F` | `kChimes` | 28 × 74 | (0, 0) | :474 |

`kFlower`'s six sizes come from `flowerSrc[6]`, set in `InitClutter`
(`GliderPRO/Sources/StructuresInit2.c:72-88`):

| index | W×H | sheet (l,t) | Line |
|---|---|---|---|
| 0 | 10 × 28 | (0, 23) | :72-73 |
| 1 | 24 × 35 | (10, 16) | :75-76 |
| 2 | 34 × 35 | (34, 16) | :78-79 |
| 3 | 27 × 23 | (68, 14) | :81-82 |
| 4 | 27 × 14 | (68, 37) | :84-85 |
| 5 | 32 × 51 | (95, 0) | :87-88 |

`kNumFlowers` = 6 (`GliderPRO/Headers/GliderDefines.h:458`).

## 3.2 PICT sheet IDs

From `GliderPRO/Sources/StructuresInit.c:19-39` and `GliderPRO/Sources/StructuresInit2.c:20-22`
(`kAngelPictID`, `kSupportPictID`, `kClutterPictID` are defined in the latter).
*Most* sheets have a companion 1-bit mask PICT at **ID + 1000**, loaded by a second `LoadGraphic`
call. The exceptions matter: **`kSwitchPictID` (4003) has no mask at all** (`StructuresInit.c:458`
is the only `LoadGraphic` for it), `kSupportPictID` and `kBadgePictID` have none, and
`kAngelPictID`'s mask is at **ID + 1** (1020, `StructuresInit2.c:130-134`).

| Constant | ID | Sheet size | Used by |
|---|---|---|---|
| `kShadowPictID` | 3998 | — | glider shadow |
| `kBlowerPictID` | 4000 | 48 × 402 | all 16 blowers |
| `kFurniturePictID` | 4001 | 64 × 278 | all 15 furniture types |
| `kBonusPictID` | 4002 | 88 × 378 | all 15 prizes |
| `kSwitchPictID` | 4003 | 32 × 104 | 5 visible switches |
| `kLightPictID` | 4004 | 72 × 126 | 8 lights |
| `kAppliancePictID` | 4005 | 80 × 269 | 14 appliances |
| `kPointsPictID` | 4006 | 24 × 120 | flying score digits |
| `kRubberBandsPictID` | 4007 | — | in-flight rubber bands |
| `kTransportPictID` | 4008 | 56 × 32 | floor/ceiling transport ducts |
| `kToastPictID` | 4009 | 32 × 174 | toaster's 6 bread frames |
| `kShreddedPictID` | 4010 | 40 × 35 | shredder confetti |
| `kBalloonPictID` | 4011 | 24 × 30 × 8 | balloon |
| `kCopterPictID` | 4012 | 32 × 30 × 10 | 'copters |
| `kDartPictID` | 4013 | 64 × 19 × 4 | darts |
| `kBallPictID` | 4014 | 32 × 32 × 2 | bouncing ball |
| `kDripPictID` | 4015 | 16 × 12 × 6 | water drip |
| `kEnemyPictID` | 4016 | — | shared enemy art |
| `kFishPictID` | 4017 | 16 × 16 × 8 | fish |
| `kClutterPictID` | 4018 | 128 × 69 | mousehole, faucet, flowers |
| `kBadgePictID` | 1996 | — | high-score badge |
| `kAngelPictID` | 1019 | 96 × 44 | glider-lost angel |
| `kSupportPictID` | 1999 | 512 × 44 | floor support (`kRoomWide` × `kFloorSupportTall`) |
| `kStarPictID` | 1995 | — | **not the `kStar` object** — its only use is `LoadScaledGraphic(kStarPictID, ...)` in the high-score screen (`GliderPRO/Sources/HighScores.c:67`). The `kStar` object's six 32 × 31 frames (`starSrc[6]`) live in the **bonus** sheet 4002 at origin (48, 31·i) (`GliderPRO/Sources/StructuresInit.c:391-392`). |

House-specific art (doors, windows, stairs, mailboxes, `kOzma`, `kMirror`, `kFireplace`,
`kWallWindow`, `kBear`, `kCalendar`, vases, `kBulletin`, `kCloud`, `kRug`, `kChimes`,
`kCinderBlock`, `kFlowerBox`, `kGuitar`, `kTrunk`, `kBooks`, `kHipLamp`, `kDecoLamp`) is loaded
from the house file's own resource fork; `kUserStructureRange` = 3300
(`GliderPRO/Headers/GliderDefines.h:523`) is the base of that range, and `kCustomPict` accepts
any PICT ID from 10000 upward.

## 3.3 Animation frame counts

`GliderPRO/Headers/GliderDefines.h:439-458`, with sub-rect tables in
`GliderPRO/Sources/StructuresInit.c`.

| Constant | Value | Sub-rect array | Frame size | Sheet stride |
|---|---|---|---|---|
| `kNumCandleFlames` | 5 | `flame[5]` | 16 × 15 | (32, 179 + 15·i) |
| `kNumTikiFlames` | 5 | `tikiFlame[5]` | 8 × 10 | (40, 69 + 10·i) |
| `kNumBBQCoals` | 4 | `coals[4]` | 32 × 9 | (0, 304 + 9·i) |
| `kNumPendulums` | 3 | `pendulumSrc[3]` | 32 × 28 | (56, 186 + 28·i) |
| `kNumTrackLights` | 3 | `trackLightSrc[3]` | 24 × 24 | (24·i, 102) |
| `kNumOutletPicts` | 4 | `outletSrc[4]` | 16 × 24 | (64, 22 + 24·i) |
| `kNumBreadPicts` | 6 | `breadSrc[6]` | 32 × 29 | — |
| `kNumBalloonFrames` | 8 | `balloonSrc[8]` | 24 × 30 | — |
| `kNumCopterFrames` | 10 | `copterSrc[10]` | 32 × 30 | — |
| `kNumDartFrames` | 4 | `dartSrc[4]` | 64 × 19 | — |
| `kNumBallFrames` | 2 | `ballSrc[2]` | 32 × 32 | — |
| `kNumDripFrames` | 6 | `dripSrc[6]` | 16 × 12 | — |
| `kNumFishFrames` | 8 | `fishSrc[8]` | 16 × 16 | — |
| `kNumSparkleModes` | 5 | `sparkleSrc[5]` | 20 × 19 | 2..4 at (0, 70 + 19·i); `[0] = [4]`, `[1] = [3]` (ping-pong) |
| — | 6 | `starSrc[6]` | 32 × 31 | (48, 31·i) |
| — | 4 | `greaseSrcRt[4]`, `greaseSrcLf[4]` | 32 × 27 | — |
| — | 15 | `pointsSrc[15]` | 24 × 8 | — |
| — | 11 | `digits[11]` | 4 × 6 | (28, 6·i) |
| — | 2 | `flourescentSrc1/2` | 16 × 12 | (0, 78) / (0, 90) |
| — | 2 | `plusScreen1/2` | 32 × 22 | — |
| — | 2 | `tvScreen1/2` | 64 × 49 | — |
| — | 2 | `coffeeLight1/2` | 8 × 4 | — |
| — | 2 | `vcrTime1/2` | 16 × 4 | — |
| — | 2 | `stereoLight1/2` | 4 × 1 | — |
| — | 2 | `microOn`/`microOff` | 16 × 35 | (64, 222) / (64, 187) |

Note `kNumSparkleModes` = 5 but only 3 distinct bitmaps exist: `sparkleSrc[0] = sparkleSrc[4]` and
`sparkleSrc[1] = sparkleSrc[3]`, giving a grow/shrink ping-pong from 3 frames.

---

# 4. Room geometry and forced placement

## 4.1 Room coordinate system

`GliderPRO/Headers/GliderDefines.h:496-511`:

| Constant | Value | Meaning |
|---|---|---|
| `kNumTiles` | 8 | background tiles per room |
| `kTileWide` | 64 | tile width in pixels |
| `kTileHigh` | 322 | tile (= room) height in pixels |
| `kRoomWide` | 512 | `kNumTiles * kTileWide` — room width |
| `kFloorSupportTall` | 44 | height of the floor-support strip drawn below a room |
| `kVertLocalOffset` | 322 | vertical distance between vertically adjacent rooms (comment: "kTileHigh - 39 (was 283, then 295)" — the comment is stale; the value equals `kTileHigh`) |
| `kCeilingLimit` | 8 | glider cannot rise above this y in a closed room |
| `kFloorLimit` | 312 | floor y in a closed room |
| `kRoofLimit` | 122 | roof y for outdoor rooms |
| `kLeftWallLimit` | 12 | left wall x in a closed room |
| `kNoLeftWallLimit` | −24 | left limit when the left wall is open (`0 - kGliderWide/2`) |
| `kRightWallLimit` | 500 | right wall x in a closed room |
| `kNoRightWallLimit` | 536 | right limit when open (`kRoomWide + kGliderWide/2`) |
| `kNoCeilingLimit` | −10 | ceiling limit when open |
| `kNoFloorLimit` | 332 | floor limit when open |

Also `kGliderWide` = 48 and `kGliderHigh` = 20 (`GliderPRO/Headers/GliderDefines.h:548-549`).

Object coordinates are **room-local**: (0, 0) is the room's top-left, x grows right to 512, y grows
down to 322. To draw a neighbour room's object, `VerticalRoomOffset`
(`GliderPRO/Sources/ObjectRects.c:1067-1091`) and `OffsetRectRoomRelative`
(`GliderPRO/Sources/ObjectRects.c:1093-1133`) shift by ±`kVertLocalOffset` (322) vertically and
±`kRoomWide` (512) horizontally, per the 9-neighbour index.

Rooms are addressed by (floor, suite) with a combined key
(`GliderPRO/Sources/Link.c:34-37`):

```c
short MergeFloorSuite (short floor, short suite)
{
	return ((suite * 100) + floor);
}
```

Callers pass `floor + kNumUndergroundFloors` (8), so floor 0 encodes as 8 and the 8 underground
floors encode as 0..7. Observed range across shipped houses: floor −7..39, suite 0..127
(`kMaxNumRoomsH` = 128, `kMaxNumRoomsV` = 64, `GliderPRO/Headers/GliderDefines.h:543-544`).

## 4.2 Types with a forced coordinate

Twenty-six types have their `topLeft.v` (or in two cases — `kCounter`, `kDresser` —
`bounds.bottom`) forced to a constant by `AddNewObject`, because they must sit flush against a
specific structural feature; three more (`kMousehole`, `kFireplace`, `kManhole`) use local
constants, for 29 in all. From
`GliderPRO/Headers/GliderDefines.h:467-494`, cross-checked against the observed value range in all
22 shipped houses — **every single one matches exactly**, which is strong evidence that the editor
is the only thing that ever wrote these files.

| Constant | Value | Applies to | Field forced | Observed range in 22 houses |
|---|---|---|---|---|
| `kFloorVentTop` | 305 | `kFloorVent` | `topLeft.v` | 305..305 ✓ |
| `kCeilingVentTop` | 8 | `kCeilingVent` | `topLeft.v` | 8..8 ✓ |
| `kFloorBlowerTop` | 304 | `kFloorBlower` | `topLeft.v` | 304..304 ✓ |
| `kCeilingBlowerTop` | 5 | `kCeilingBlower` | `topLeft.v` | 5..5 ✓ |
| `kSewerGrateTop` | 303 | `kSewerGrate` | `topLeft.v` | 303..303 ✓ |
| (local `kGrecoVentTop`) | 303 | `kGrecoVent` | `topLeft.v` | 303..303 ✓ |
| (local `kSewerBlowerTop`) | 292 | `kSewerBlower` | `topLeft.v` | 292..292 ✓ |
| `kFloorTransTop` | 302 | `kFloorTrans` | `topLeft.v` | 300..302 (one outlier at 300) |
| `kCeilingTransTop` | 6 | `kCeilingTrans` | `topLeft.v` | 6..6 ✓ |
| `kStairsTop` | 28 | `kUpStairs`, `kDownStairs` | `topLeft.v` | 28..28 ✓ (both) |
| `kCounterBottom` | 304 | `kCounter` | `bounds.bottom` | 304..304 ✓ |
| `kDresserBottom` | 293 | `kDresser` | `bounds.bottom` | 293..293 ✓ |
| `kCeilingLightTop` | 4 | `kCeilingLight` | `topLeft.v` | 4..4 ✓ |
| `kHipLampTop` | 23 | `kHipLamp` | `topLeft.v` | 23..23 ✓ |
| `kDecoLampTop` | 91 | `kDecoLamp` | `topLeft.v` | 91..91 ✓ |
| `kFlourescentTop` | 12 | `kFlourescent` | `topLeft.v` | 12..12 ✓ |
| `kTrackLightTop` | 5 | `kTrackLight` | `topLeft.v` | 5..5 ✓ |
| `kDoorInTop` | 0 | `kDoorInLf`, `kDoorInRt` | `topLeft.v` | 0..0 ✓ (both) |
| `kDoorInLfLeft` | 0 | `kDoorInLf` | `topLeft.h` | 0..0 ✓ |
| `kDoorInRtLeft` | 368 | `kDoorInRt` | `topLeft.h` | 368..368 ✓ |
| `kDoorExTop` | 0 | `kDoorExRt`, `kDoorExLf` | `topLeft.v` | 0..0 ✓ (both) |
| `kDoorExLfLeft` | 0 | `kDoorExLf` | `topLeft.h` | 0..0 ✓ |
| `kDoorExRtLeft` | 496 | `kDoorExRt` | `topLeft.h` | 496..496 ✓ |
| `kWindowInTop` | 64 | `kWindowInLf`, `kWindowInRt` | `topLeft.v` | 64..64 ✓ (both) |
| `kWindowInLfLeft` | 0 | `kWindowInLf` | `topLeft.h` | 0..0 ✓ |
| `kWindowInRtLeft` | 492 | `kWindowInRt` | `topLeft.h` | 492..492 ✓ |
| `kWindowExTop` | 64 | `kWindowExRt`, `kWindowExLf` | `topLeft.v` | 64..64 ✓ (both) |
| `kWindowExLfLeft` | 0 | `kWindowExLf` | `topLeft.h` | 0..0 ✓ |
| `kWindowExRtLeft` | 496 | `kWindowExRt` | `topLeft.h` | 496..496 ✓ |

Additional forced coordinates defined locally in `GliderPRO/Sources/ObjectAdd.c:15-23`:

| Constant | Value | Applies to | Field | Observed |
|---|---|---|---|---|
| `kMouseholeBottom` | 295 | `kMousehole` | `bounds.bottom` | 295..295 ✓ (top always 284) |
| `kFireplaceBottom` | 297 | `kFireplace` | `bounds.bottom` | 297..297 ✓ (top always 155) |
| `kManholeSits` | 322 | `kManhole` | `bounds.bottom` | 322..322 ✓ (top always 300) |

`kManhole` is additionally **snapped to the tile grid**: `left = ((left - 3) / 64) * 64 + 3`
(`GliderPRO/Sources/ObjectAdd.c`, `kManhole` branch). Observed lefts are 67, 131, 195, 259, 323 —
all ≡ 3 (mod 64). ✓

Doors and windows choose left vs. right variant from the click position:
`if (where.h > kRoomWide / 2)` → the "Rt" variant, else the "Lf" variant.

Enemies also get a default vertical placement: `kBalloon`, `kCopterLf`, `kCopterRt` are placed at
`topLeft.v = (kTileHigh / 2) - halfTall` = 161 − 15 = **146**, and the observed value is 146..146
for all 914 balloons and 'copters in shipped houses. ✓ `kDartLf` is placed at
`topLeft.h = kRoomWide - RectWide(srcRects[kDartLf])` = 512 − 64 = 448 (observed 436..448) and
`kDartRt` at `topLeft.h = 0` (observed 0..2).

---

# 5. `GetObjectRect` — the bounding rectangle of every type

`GliderPRO/Sources/ObjectRects.c:32-273`. This is the function every other subsystem calls to get
an object's screen rectangle: the editor's selection marquee, the drawing loop, the map view. It is
a single `switch (who->what)` with 16 arms (the 16 numbered groups below).

Numbered pseudocode, original variable names in parentheses:

1. `case kObjectIsEmpty:` → `QSetRect(itsRect, 0,0,0,0)` (`:39-41`).
2. **All 15 simple blowers** (`kFloorVent` … `kSewerBlower`, i.e. `0x01`–`0x0F`) (`:43-61`):
   `*itsRect = srcRects[what]`; `ZeroRectCorner(itsRect)`; `QOffsetRect(itsRect, data.a.topLeft.h,
   data.a.topLeft.v)`.
3. **`kLiftArea`** (`:63-66`): `QSetRect(itsRect, 0, 0, data.a.distance, data.a.tall * 2)`;
   offset by `data.a.topLeft`. So `distance` is the *width* and `tall * 2` the *height* — the field
   names lie for this one type.
   *(Lines `:68-71` are unreachable dead code: a stray copy of the step-2 body after the `break`.)*
4. **All 15 furniture types** (`:73-89`): `*itsRect = data.b.bounds` verbatim. No table lookup.
5. **14 bonuses** (`kRedClock` … `kHelium`, excluding `kSlider`) (`:91-110`): `srcRects[what]`
   zeroed, offset by `data.c.topLeft`.
6. **`kSlider`** (`:112-119`): `srcRects[kSlider]` zeroed (64 × 16), offset by `data.c.topLeft`,
   then `itsRect->right = itsRect->left + data.c.length` — so only the width is variable and the
   height stays 16.
7. **14 transports** (`kUpStairs` … `kWindowExLf`) (`:121-140`): `srcRects[what]` zeroed, offset by
   `data.d.topLeft`.
8. **`kInvisTrans`** (`:142-150`): `srcRects[kInvisTrans]` (64 × 32) zeroed, offset by
   `data.d.topLeft`, then `itsRect->bottom = itsRect->top + data.d.tall` and
   `itsRect->right += (short)data.d.wide`. So the height is `tall` and the width is `64 + wide`,
   with `wide` a `Byte` (0..255; observed 0..127).
9. **`kDeluxeTrans`** (`:152-159`):
   ```c
   wide = (who->data.d.tall & 0xFF00) >> 8;		// Get high byte
   tall = who->data.d.tall & 0x00FF;			// Get low byte
   QSetRect(itsRect, 0, 0, wide * 4, tall * 4);	// Scale by 4
   QOffsetRect(itsRect, who->data.d.topLeft.h, who->data.d.topLeft.v);
   ```
   Width = `hiByte(tall) * 4`, height = `loByte(tall) * 4`. Granularity is therefore 4 pixels and
   the maximum is 255 × 4 = 1020 in each axis. The default `tall` written by `AddNewObject` is
   `0x1010`, giving 16 × 4 = 64 by 16 × 4 = 64.
   *Note that `data.d.tall` is a signed `short`, so the observed range −32722..32591 is just the
   packed byte pair reinterpreted; a port must mask, never compare.*
10. **9 switches/triggers** (`kLightSwitch` … `kSoundTrigger`) (`:161-175`): `srcRects[what]`
    zeroed, offset by `data.e.topLeft`.
11. **6 lights** (`kCeilingLight`, `kLightBulb`, `kTableLamp`, `kHipLamp`, `kDecoLamp`,
    `kInvisLight`) (`:177-188`): `srcRects[what]` zeroed, offset by `data.f.topLeft`.
12. **`kFlourescent`, `kTrackLight`** (`:190-198`): `srcRects[what]` zeroed, then
    `itsRect->right = data.f.length` **before** the offset, then offset by `data.f.topLeft`. Final
    right = `length + topLeft.h`; final width = `length` (since left was zeroed). Height stays 12
    or 24.
13. **13 appliances** (`kShredder` … `kCDs`, excluding `kCustomPict`) (`:200-218`): `srcRects[what]`
    zeroed, offset by `data.g.topLeft`.
14. **`kCustomPict`** (`:220-237`):
    ```c
    thePict = GetPicture(who->data.g.height);
    if (thePict == nil)
    {
        who->data.g.height = 10000;              // reset to the default ID
        *itsRect = srcRects[who->what];          // 72 x 34 fallback
    }
    else
    {
        HLock((Handle)thePict);
        *itsRect = (*thePict)->picFrame;          // the PICT's own frame
        HUnlock((Handle)thePict);
    }
    ZeroRectCorner(itsRect);
    QOffsetRect(itsRect, who->data.g.topLeft.h, who->data.g.topLeft.v);
    ```
    **This is a mutating getter**: a missing PICT rewrites the object's `height` field in place.
15. **8 enemies + `kCobweb`** (`kBalloon` … `kCobweb`) (`:239-253`): `srcRects[what]` zeroed, offset
    by `data.h.topLeft`.
16. **15 clutter types** (`:255-271`): `*itsRect = data.i.bounds` verbatim.

Summary of the four rect strategies:

| Strategy | Types | Count |
|---|---|---|
| Explicit `Rect` in the record | all furniture (`b.bounds`) + all clutter (`i.bounds`) | 30 |
| `srcRects[what]` size + `topLeft` | all blowers except `kLiftArea` (15), all bonuses except `kSlider` (14), all transports except `kInvisTrans`/`kDeluxeTrans` (14), all switches (9), 6 lights, 13 appliances, all enemies incl. `kCobweb` (9) | 80 |
| `srcRects` size with one dimension overridden | `kSlider` (width), `kInvisTrans` (both), `kFlourescent`/`kTrackLight` (width) | 4 |
| Fully computed | `kLiftArea` (`distance` × `tall`·2), `kDeluxeTrans` (packed bytes × 4), `kCustomPict` (PICT `picFrame`) | 3 |
|   |   | **117** |

30 + 80 + 4 + 3 = 117, i.e. every defined type is covered. (`kFlower` is one of the 15 clutter types
counted in the first row: it takes the `data.i.bounds` path, but the editor recomputes those bounds
from `flowerSrc[data.i.pict]` whenever `pict` changes — `GliderPRO/Sources/ObjectInfo.c:2330-2333`,
`:2362-2365`. This is why it needs no `srcRects` entry.)

---

# 6. Hot spots and action codes

## 6.1 The 28 action codes

`GliderPRO/Headers/GliderDefines.h:282-309`. These are the only way an object affects the player.

| Value | Constant | Effect on the glider (from `HandleHotSpotCollision`, `GliderPRO/Sources/Interactions.c:1198-1623`) |
|---|---|---|
| 0 | `kIgnoreIt` | no effect; the rect exists only so the object can be found/toggled |
| 1 | `kLiftIt` | `vDesiredVel = kFloorVentLift` = **−6** |
| 2 | `kDropIt` | `vDesiredVel = kCeilingVentDrop` = **+8** |
| 3 | `kPushItLeft` | `hDesiredVel -= kFanStrength` (**12**) |
| 4 | `kPushItRight` | `hDesiredVel += kFanStrength` (**12**) |
| 5 | `kDissolveIt` | glider is destroyed (fade out) unless `foilTotal > 0` absorbs the hit |
| 6 | `kRewardIt` | collect the prize; dispatches to `HandleRewards` |
| 7 | `kMoveItUp` | go up the stairs — requires `!thisGlider->heldRight` |
| 8 | `kMoveItDown` | go down the stairs — requires `!thisGlider->heldLeft` |
| 9 | `kSwitchIt` | throw a switch; dispatches to `HandleSwitches` |
| 10 | `kShredIt` | shredder: destroys the glider and spawns confetti |
| 11 | `kStrumIt` | guitar: play a chord |
| 12 | `kTriggerIt` | arm a trigger (`ArmTrigger`) |
| 13 | `kBurnIt` | flame: destroys the glider (foil does **not** save you) |
| 14 | `kSlideIt` | grease/slide rect: `sliding = true; vVel = bounds.top - dest.bottom` |
| 15 | `kTransportIt` | invisible/deluxe transport: teleport to the linked object |
| 16 | `kIgnoreLeftWall` | suppress the left wall while overlapping (doors/windows) |
| 17 | `kIgnoreRightWall` | suppress the right wall while overlapping |
| 18 | `kMailItLeft` | enter a left-facing mailbox (needs correct facing and not tipped) |
| 19 | `kMailItRight` | enter a right-facing mailbox |
| 20 | `kDuctItDown` | floor transport duct: travel down |
| 21 | `kDuctItUp` | ceiling transport duct: travel up |
| 22 | `kMicrowaveIt` | microwave: `HandleMicrowaveAction` destroys carried items |
| 23 | `kIgnoreGround` | manhole: suppress the floor so the glider falls through |
| 24 | `kBounceIt` | invisible rebounder: reflect the glider |
| 25 | `kChimeIt` | wind chimes: play a chime |
| 26 | `kWebIt` | cobweb: `WebGlider` sticks the glider |
| 27 | `kSoundIt` | sound trigger: play the linked `snd ` resource |

Other constants used by these actions (`GliderPRO/Sources/Interactions.c:13-19`):
`kFloorVentLift` −6, `kCeilingVentDrop` 8, `kFanStrength` 12, `kBatterySupply` 50
(comment: "about 2 rooms worth of thrust"), `kHeliumSupply` 150, `kBandsSupply` 8, `kFoilSupply` 8.

## 6.2 `hotObject` and `AddActiveRect`

`GliderPRO/Headers/GliderStructs.h:218-225`:

```c
typedef struct
{
	Rect		bounds;
	short		action;
	short		who;
	Boolean		isOn, stillOver;
	Boolean		doScrutinize;
} hotObject, *hotPtr;
```

`AddActiveRect` (`GliderPRO/Sources/ObjectRects.c:277-292`):

The counter is named **`nHotSpots`** in the original (not `numHotSpots`):

1. If `nHotSpots >= kMaxHotSpots` (**56**, `GliderPRO/Headers/GliderDefines.h:259`) → return −1.
2. `hotSpots[nHotSpots].bounds = *bounds`
3. `.action = action`, `.who = who` (index into `masterObjects[]`),
   `.isOn = isOn`, `.stillOver = false`, `.doScrutinize = doScrutinize`
4. `nHotSpots++`; `return (nHotSpots - 1)`.

`isOn` gates the hot spot at collision time (a switched-off blower has an inert column).
`doScrutinize` selects the precise per-pixel-ish `SectGlider` test instead of the cheap rect test
(`GliderPRO/Sources/Interactions.c:101-132`) — it is set for solid objects the glider dissolves
against, so that a near-miss does not kill you.

**One object may create more than one hot spot.** `CreateActiveRects` returns only the *last*
index it created; objects with two rects therefore have their first rect unreferenced by
`masterObjects[].hotNum`. Types that create 2 hot spots: `kLeftFan`, `kRightFan`, `kMicrowave`,
and the five flame types when their column is taller than 24 (`kTaper`, `kCandle`, `kStubby`,
`kTiki`, `kBBQ` — those create 3).

## 6.3 `CreateActiveRects` — per type

`GliderPRO/Sources/ObjectRects.c:296-1063`. Local constants at `:13-19`:

```c
#define kFloorColumnWide		4
#define kCeilingColumnWide		24
#define kFanColumnThick			16
#define kFanColumnDown			20
#define kDeadlyFlameHeight		24
#define kStoolThick				25
#define kShredderActiveHigh		40
```

### Blowers

| Type | Hot spot(s) | `isOn` | `doScrutinize` | Lines |
|---|---|---|---|---|
| `kFloorVent` | `(0, −distance, 4, 0)` offset by `HalfRectWide(srcRects[kFloorVent]) − 2` horizontally, then by `topLeft` → `kLiftIt` | `data.a.state` | false | :311-319 |
| `kCeilingVent` | `(0, 0, 24, distance)` offset by `HalfRectWide − 12`, then `topLeft` → `kDropIt` | `data.a.state` | false | :321-331 |
| `kFloorBlower` | as `kFloorVent` → `kLiftIt` | `data.a.state` | false | :333-343 |
| `kCeilingBlower` | as `kCeilingVent` → `kDropIt` | `data.a.state` | false | :345-355 |
| `kSewerGrate` | as `kFloorVent` → `kLiftIt` | `data.a.state` | false | :357-367 |
| `kGrecoVent` | as `kFloorVent` → `kLiftIt` | `data.a.state` | false | :566-576 |
| `kSewerBlower` | as `kFloorVent` → `kLiftIt` | `data.a.state` | false | :578-588 |
| `kLeftFan` | (1) `13 × 43` at `topLeft + (16, 12)` → `kDissolveIt`, always on, scrutinized. (2) `distance × 16` offset `(−distance, 20)` then `topLeft` → `kPushItLeft` | (1) true (2) `data.a.state` | (1) true (2) false | :369-382 |
| `kRightFan` | (1) `13 × 43` at `topLeft + (6, 12)` → `kDissolveIt`. (2) `distance × 16` offset `(RectWide(srcRects[kRightFan]) = 40, 20)` then `topLeft` → `kPushItRight` | (1) true (2) `data.a.state` | (1) true (2) false | :384-397 |
| `kInvisBlower` | one rect, direction from `vector & 0x0F`: `1`(up) `(0, −distance−24, 4, 0)` offset `(10, 24)` → `kLiftIt`; `2`(right) `(0, 0, distance+24, 16)` offset `(0, 4)` → `kPushItRight`; `4`(down) `(0, 0, 4, distance+24)` offset `(10, 0)` → `kDropIt`; `8`(left) `(0, 0, distance+24, 16)` offset `(−distance, 4)` → `kPushItLeft` | `data.a.state` | false | :522-564 |
| `kLiftArea` | `distance × (tall*2)` at `topLeft`, action from `vector & 0x0F`: `1`→`kLiftIt`, `2`→`kPushItRight`, `4`→`kDropIt`, `8`→`kPushItLeft` | `data.a.state` | false | :590-617 |

`HalfRectWide(&srcRects[kFloorVent])` = 24, minus `kFloorColumnWide/2` = 2 → the 4-pixel lift
column is centred on the 48-pixel vent sprite at x + 22. `kCeilingColumnWide` is 24 — ceiling
columns are 6× wider than floor columns.

### Flames (`kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ`)

Each builds a lift column `(0, −distance, kFloorColumnWide, 0)` centred on the sprite, then:

```
1.  if (column height > kDeadlyFlameHeight /* 24 */)
2.      bounds.bottom -= 24;   AddActiveRect(bounds, kLiftIt,  who, true, false)
3.      bounds.bottom += 24;   bounds.top = bounds.bottom - 24 + 2
4.      AddActiveRect(bounds, kBurnIt, who, true, false)
5.  else
6.      AddActiveRect(bounds, kBurnIt, who, true, false)
7.  AddActiveRect(bodyRect, kDissolveIt, who, true, true)
```

So a tall candle lifts you from a distance and burns you only in the bottom 22 pixels
(`24 − 24 + 2`); a short candle burns you throughout. Per-type numbers:

| Type | Column centring | Body `kDissolveIt` rect | Lines |
|---|---|---|---|
| `kTaper` | `HalfRectWide(20) − 2` = 8 | `7 × 48` at `topLeft + (6, 11)` | :399-422 |
| `kCandle` | `HalfRectWide(32) − 2` = 14, and the second `QOffsetRect` uses `topLeft.h − 2` — the column is shifted 2 px further **left**, not down (`:430`) | `8 × 20` at `topLeft + (9, 11)` | :424-447 |
| `kStubby` | `HalfRectWide(20) − 2 − 1` = 7 | `15 × 26` at `topLeft + (1, 11)` | :449-472 |
| `kTiki` | `HalfRectWide(27) − 2` = 11 (27-wide sprite) | `15 × 14` at `topLeft + (6, 6)` | :474-497 |
| `kBBQ` | column `bottom = 8` instead of 0 (`:500`) | `52 × 17` at `topLeft + (6, 8)` | :499-520 |

`kBurnIt` is the only lethal action that foil does **not** protect against.

### Furniture

| Type | Hot spot | Action | `isOn` | `doScrutinize` | Lines |
|---|---|---|---|---|---|
| `kTable`, `kShelf`, `kCabinet`, `kFilingCabinet`, `kWasteBasket`, `kMilkCrate`, `kCounter`, `kDresser`, `kDeckTable`, `kTrunk`, `kInvisObstacle` | `data.b.bounds` verbatim | `kDissolveIt` | true | true | :619-632 |
| `kBooks` | `bounds` with `right -= 2` | `kDissolveIt` | true | true | :634-638 |
| `kManhole` | `bounds` with `left += kGliderWide + 3` (= +51), `right -= 51`, `top = kFloorLimit − 1` (**311**), `bottom = kTileHigh` (**322**) | `kIgnoreGround` | true | false | :640-647 |
| `kInvisBounce` | `data.b.bounds` verbatim | `kBounceIt` | true | true | :649-652 |
| `kStool` | `InsetRect(bounds, 1, 1)` then `bottom = top + kStoolThick` (**25**) | `kDissolveIt` | true | true | :654-659 |

The manhole rect ignores the object's own vertical extent entirely and always occupies y 311..322 —
the floor band — inset by a glider's width plus 3 on each side so you must be reasonably centred to
fall through.

### Prizes / bonuses

| Type | Hot spot | Action | `isOn` | Lines |
|---|---|---|---|---|
| `kRedClock`, `kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`, `kBattery`, `kBands`, `kFoil`, `kInvisBonus`, `kStar`, `kHelium` | `srcRects[what]` zeroed, offset by `data.c.topLeft` | `kRewardIt` | **`data.c.state`** | :661-679 |
| `kGreaseRt` — standing (`state` true) | same as above | `kRewardIt` | true | :681-689 |
| `kGreaseRt` — spilled (`state` false) | `(0, −2, length − 5, 0)` offset `(32 − 1, 27)` then `topLeft` | `kSlideIt` | true | :690-697 |
| `kGreaseLf` — standing | same as prizes | `kRewardIt` | true | :700-708 |
| `kGreaseLf` — spilled | `(−length + 5, −2, 0, 0)` offset `(1, 27)` then `topLeft` | `kSlideIt` | true | :709-716 |
| `kSparkle` | **none** — the rect is computed but never registered | — | — | :719-725 |
| `kSlider` | `(0, 0, data.c.length, 16)` offset by `topLeft` | `kSlideIt` | true | :727-732 |

The spilled-grease rect is only **2 pixels tall** (`top = −2`, `bottom = 0`) and sits 27 pixels
below the can's top-left — a thin slick on the floor. The `− 5` shortens it by 5 pixels at the
far end.

### Transport

| Type | Hot spot | Action | Gate | Lines |
|---|---|---|---|---|
| `kUpStairs` | `112 × 32` at `topLeft` | `kMoveItUp` | — | :734-740 |
| `kDownStairs` | `(−80, −56, 0, 0)` offset `(srcRects[kDownStairs].right, 170)` then `topLeft` | `kMoveItDown` | — | :742-749 |
| `kMailboxLf` | `(−72, 0, 0, 40)` offset `(30, 16)` then `topLeft` | `kMailItLeft` | `data.d.who != 255` | :751-761 |
| `kMailboxRt` | `(0, 0, 72, 40)` offset `(79, 16)` then `topLeft` | `kMailItRight` | `data.d.who != 255` | :763-773 |
| `kFloorTrans` | `(0, −48, 76, 0)` offset `(−8, RectTall(srcRects[kFloorTrans]) = 15)` then `topLeft` | `kDuctItDown` | `data.d.who != 255` | :775-785 |
| `kCeilingTrans` | `(0, 0, 76, 48)` offset `(−8, 0)` then `topLeft` | `kDuctItUp` | `data.d.who != 255` | :787-797 |
| `kDoorInLf` | `16 × 240` offset `(0, 52)` then `topLeft` | `kIgnoreLeftWall` | — | :799-806 |
| `kDoorInRt` | `16 × 240` offset `(128, 52)` then `topLeft` | `kIgnoreRightWall` | — | :808-815 |
| `kDoorExRt` | `16 × 240` offset `(0, 52)` then `topLeft` | `kIgnoreRightWall` | — | :817-824 |
| `kDoorExLf` | `16 × 240` offset `(0, 52)` then `topLeft` | `kIgnoreLeftWall` | — | :826-833 |
| `kWindowInLf` | `16 × 44` offset `(0, 96)` then `topLeft` | `kIgnoreLeftWall` | — | :835-842 |
| `kWindowInRt` | `16 × 44` offset `(4, 96)` then `topLeft` | `kIgnoreRightWall` | — | :844-851 |
| `kWindowExRt` | `16 × 44` offset `(0, 96)` then `topLeft` | `kIgnoreRightWall` | — | :853-860 |
| `kWindowExLf` | `16 × 44` offset `(0, 96)` then `topLeft` | `kIgnoreLeftWall` | — | :862-869 |
| `kInvisTrans` | `64 × 32` at `topLeft`, then `bottom = top + data.d.tall`, `right += data.d.wide` | `kTransportIt` | `data.d.who != 255` | :871-882 |
| `kDeluxeTrans` | `(hiByte(tall) * 4) × (loByte(tall) * 4)` at `topLeft`; `isOn = data.d.wide & 0x0F` | `kTransportIt` | `data.d.who != 255` | :884-896 |

Note that **stairs, doors and windows are never gated on `who`** — they are geometric, not linked.
Mailboxes and the two transport ducts *are* gated: an unlinked mailbox is inert scenery.

### Switches and triggers

| Type | Hot spot | Action | Gate | Lines |
|---|---|---|---|---|
| `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch`, `kInvisSwitch` | `srcRects[what]` zeroed, offset by `data.e.topLeft` | `kSwitchIt` | `data.e.where != -1` | :898-921 |
| `kTrigger`, `kLgTrigger` | same | `kTriggerIt` | `data.e.where != -1` | :898-921 |
| `kSoundTrigger` | `48 × 48` at `data.e.topLeft` — **not** the 32 × 32 `srcRects` size | `kSoundIt` | `LoadTriggerSound(data.e.where) == noErr` | :923-928 |

`kSoundTrigger`'s hot spot is 48 × 48 while its editor/selection rect (from `GetObjectRect`) is
32 × 32. The gate is a *successful resource load*, not a link check — a sound trigger pointing at a
missing `snd ` resource is silently inert.

### Lights

**No light of any kind creates a hot spot** (`:930-938` is an empty `break`). Lights act only
through the global `numLights` count that gates all drawing.

### Appliances

| Type | Hot spot(s) | Action | `isOn` | `doScrutinize` | Lines |
|---|---|---|---|---|---|
| `kShredder` | `srcRects[kShredder]` with `bottom = top + kShredderActiveHigh` (**40**) and `right += 48`, zeroed, offset by `topLeft`, then by `(−24, −36)` | `kShredIt` | `data.g.state` | true | :940-950 |
| `kGuitar` | `8 × 96` at `topLeft + (34, 32)` | `kStrumIt` | true | false | :952-957 |
| `kOutlet` | `srcRects[kOutlet]` zeroed, offset by `topLeft` | `kIgnoreIt` | `data.g.state` | false | :959-967 |
| `kMicrowave` | **two rects**: (1) `srcRects[kMicrowave]` zeroed + `topLeft`; (2) the same rect with `bottom = top; top = 0` — i.e. the vertical strip from the room ceiling down to the microwave's top | (1) `kDissolveIt`, (2) `kMicrowaveIt` | true (both) | true (both) | :969-979 |
| `kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kVCR`, `kStereo`, `kCinderBlock`, `kFlowerBox`, `kCDs` | `srcRects[what]` zeroed + `topLeft` | `kDissolveIt` | true | true | :981-996 |
| `kCustomPict` | **none** | — | — | — | :998-999 |

The microwave's second rect spans `y = 0 .. microwaveTop` and the full sprite width: flying *over*
a microwave is what zaps your inventory, not touching it.

`kCustomPict` having no hot spot is why it is safe to have 4782 of them — they are pure decoration.

### Enemies

| Type | Hot spot | Action | `isOn` | `doScrutinize` | Lines |
|---|---|---|---|---|---|
| `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip` | `srcRects[what]` zeroed + `data.h.topLeft` | `kIgnoreIt` | true | false | :1001-1014 |
| `kFish` | `srcRects[kFish]` zeroed + `topLeft` | `kDissolveIt` | true | true | :1016-1023 |
| `kCobweb` | `srcRects[kCobweb]` zeroed + `topLeft`, then `InsetRect(bounds, -24, -10)` (i.e. **grown** by 24 horizontally and 10 vertically → 102 × 65) | `kWebIt` | true | true | :1025-1033 |

The seven moving enemies register a `kIgnoreIt` rect at their *static* position purely so the
object can be located and switched; their actual lethal collision is handled by the dynamic-object
system each frame, not by a hot spot. `kFish` and `kCobweb` do not move, so they use real hot spots.

### Clutter

| Type | Hot spot | Action | Lines |
|---|---|---|---|
| `kOzma`, `kMirror`, `kMousehole`, `kFireplace`, `kFlower`, `kWallWindow`, `kBear`, `kCalendar`, `kVase1`, `kVase2`, `kBulletin`, `kCloud`, `kFaucet`, `kRug` | **none** | — | :1035-1049 |
| `kChimes` | `numChimes++`; `srcRects[kChimes]` zeroed, offset by `data.i.bounds.left`, `data.i.bounds.top` | `kChimeIt` | :1051-1059 |

`kChimes` is the only interactive clutter type, and it is also the only place `CreateActiveRects`
has a side effect on a global counter (`numChimes`).

**Note the subtle inconsistency:** `kChimes` uses `srcRects[kChimes]` (28 × 74) for its hot spot
size while `GetObjectRect` returns `data.i.bounds`, which the editor lets you resize. A resized
chime therefore has a hot spot of the wrong size.

## 6.4 Hot-spot budget

`kMaxHotSpots` = 56. With 24 objects per room × 9 visible rooms = up to 216 objects, and several
types creating 2–3 rects each, **the hot-spot table can overflow**, in which case `AddActiveRect`
returns −1 and the object silently loses its interaction. The original has no diagnostic for this.
A port should keep the limit for fidelity but is free to log it.

---

# 7. The state model

## 7.1 Which field holds "on/off", per class

There is no uniform state field. `SetObjectState`
(`GliderPRO/Sources/Objects.c:366-699`) and `GetObjectState`
(`GliderPRO/Sources/Objects.c:703-871`) are the only authorities. This table is the complete map.

| Types | State storage | Switchable? |
|---|---|---|
| `kFloorVent`, `kCeilingVent`, `kFloorBlower`, `kCeilingBlower`, `kLeftFan`, `kRightFan`, `kSewerGrate`, `kInvisBlower`, `kGrecoVent`, `kSewerBlower`, `kLiftArea` | `data.a.state` | **yes** |
| `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ` | `data.a.state` exists but is never written | no (`changed = false`) |
| all 15 furniture types | — | no |
| `kRedClock`, `kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`, `kBattery`, `kBands`, `kFoil`, `kInvisBonus`, `kStar`, **`kSparkle`**, `kHelium`, `kGreaseRt`, `kGreaseLf` | `data.c.state` | **yes**, but only ever forced *false* (collected) by gameplay — the arm ignores `action` entirely (`Objects.c:460-462`) |
| `kSlider` | — | no — its arm is a bare `break;` in **both** functions (`Objects.c:475-476`, `:769-770`), so `SetObjectState` returns the uninitialised `changed` for it exactly as it does for `kKnifeSwitch` (see §14 item 3) |
| `kUpStairs`, `kDownStairs`, `kMailboxLf`, `kMailboxRt`, `kFloorTrans`, `kCeilingTrans`, all 8 doors/windows, `kInvisTrans` | — | no |
| **`kDeluxeTrans`** | **low nibble of `data.d.wide`** | **yes** |
| `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kInvisSwitch`, `kTrigger`, `kLgTrigger`, `kSoundTrigger` | — | no |
| **`kKnifeSwitch`** | — | **absent from both switch statements** (see §12) |
| all 8 lights | `data.f.state` | **yes** |
| **`kStereo`** | global `isPlayMusicGame` — *not* stored in the object at all | yes, but the `Set` arm (`Objects.c:584-588`) **ignores `action`** and always toggles, so `kForceOn`/`kForceOff` behave as `kToggle` |
| `kShredder`, `kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kMicrowave` | `data.g.state` | **yes** |
| **`kGuitar`** | `data.g.state` — **read** by `GetObjectState` (`Objects.c:821`, it is in the appliance arm) but never written: `SetObjectState`'s arm is `changed = false;` with the comment *"really no point to change this state"* (`Objects.c:580-582`) | no |
| `kCinderBlock`, `kFlowerBox`, `kCDs`, `kCustomPict` | — | no |
| `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish` | `data.h.state` | **yes** |
| `kCobweb` | — | no |
| all 15 clutter types | — | no |

## 7.2 `SetObjectState(room, object, action, local)`

Signature: `Boolean SetObjectState (short room, short object, short action, short local)`.
`action` is one of `kToggle` (0), `kForceOn` (1), `kForceOff` (2) — note `kOneShot` (3) is *not*
accepted here; a one-shot switch passes `kToggle`. `local` is the index in `masterObjects[]` or −1.
`newState` is a **file-scope global** at `GliderPRO/Sources/Objects.c:78`, not a local.

Canonical body, using the blower arm (`GliderPRO/Sources/Objects.c:375-417`) as the template:

1. `wasState = HGetState((Handle)thisHouse); HLock((Handle)thisHouse);` — the house is a
   relocatable `Handle`, so it must be locked before taking interior pointers. (Go: nothing to do.)
2. `switch (house->rooms[room].objects[object].what)` → per-class arm.
3. Inside the arm, `switch (action)`:
   - `kToggle`: `newState = !currentState;` write it; `changed = true`.
   - `kForceOn`: `changed = (currentState == false); newState = true;` write it.
   - `kForceOff`: `changed = (currentState == true); newState = false;` write it.
4. `if (changed && local != -1)`:
   1. write `newState` into `masterObjects[local].theObject.<field>` (the cached copy);
   2. `if (room == thisRoomNumber)` also write it into `thisRoom->objects[object].<field>`
      (a *third* copy — the current room is cached separately);
   3. play the class's on/off sound (`kBlowerOn`/`kBlowerOff` with
      `kBlowerOnPriority`/`kBlowerOffPriority` for blowers);
   4. `if (masterObjects[local].hotNum != -1) hotSpots[...].isOn = newState;`
5. `HSetState((Handle)thisHouse, wasState); return changed;`

The prize arm (`GliderPRO/Sources/Objects.c:446-473`) deviates from that template twice, and the
second deviation is a real bug:

- it ignores `action` — it is hard-wired to `changed = (data.c.state == true); newState = false;`
  and always clears the state, so a prize can only ever be *taken*;
- writing back to the master copy it says
  `masterObjects[local].theObject.data.a.state = false;` (`:465`) — **`data.a.state`, not
  `data.c.state`**. `blowerType.state` is at union offset 7 and `bonusType.state` is at union
  offset 8, so this zeroes the **low byte of `bonusType.points`** in the cached copy and leaves the
  cached `state` untouched. Only the house record (`:462`) and, when the object is in the current
  room, `thisRoom` (`:468`) get the real update. A port must decide deliberately whether to copy
  this; the visible effect is limited because `HandleRewards` reads `points` from the master copy
  *before* calling `SetObjectState` (`GliderPRO/Sources/Interactions.c:920-921`), and validity is
  re-checked against the house record via `IsThisValid`.

Also note that in this arm the `hotSpots[].isOn` update is nested **inside** the
`if (room == thisRoomNumber)` block (`:466-471`), whereas the blower template has it as a sibling
(`:409-416`).

**There are three copies of every object's state**: the authoritative one in
`(*thisHouse)->rooms[room].objects[object]`, the cached one in `masterObjects[local].theObject`,
and (for the current room only) the one in `thisRoom->objects[object]`. All three must be kept in
sync. A Go port that keeps one canonical record and derives views is simpler and equivalent —
but must not forget to also update `hotSpots[].isOn`.

### The `kDeluxeTrans` arm (`GliderPRO/Sources/Objects.c:496-530`)

```c
case kToggle:
newState = (*thisHouse)->rooms[room].objects[object].data.d.wide & 0x0F;
newState = !newState;
(*thisHouse)->rooms[room].objects[object].data.d.wide &= 0xF0;
(*thisHouse)->rooms[room].objects[object].data.d.wide += newState;
changed = true;
break;
```

`kForceOn` / `kForceOff` do the same read-mask-add, with `changed` computed from
`(wide & 0x0F) == 0x00` / `!= 0x00`. And `GetObjectState`
(`GliderPRO/Sources/Objects.c:789-791`):

```c
case kDeluxeTrans:
theState = (*thisHouse)->rooms[room].objects[object].data.d.wide & 0x0F;
break;
```

So the low nibble of `wide` is a boolean stored as 0 or 1, and the high nibble is the *initial*
state (`ObjectInfo.c:2098-2145` reads `(data.d.wide & 0xF0) >> 4` and writes back
`wasState << 4`, **clobbering the low nibble in the process**). Observed `wide` values in shipped
houses: 0..17 — i.e. 0x00, 0x01, 0x10, 0x11 — consistent with two 1-bit nibbles.

## 7.3 `GetObjectState(room, object)`

Returns `Boolean`, initialised to **`true`** (`GliderPRO/Sources/Objects.c:708`) and only lowered
by an arm that reads a real state field. Every type with no state (furniture, clutter, doors,
switches, `kCustomPict`, `kCobweb`, …) therefore reports **on**. That default matters: the
drawing code calls `GetObjectState` to pick the lit/unlit sprite for switches, and an unhandled
type would draw in its "on" pose.

`kStereo` is special (`GliderPRO/Sources/Objects.c`, appliance arm): its state is the global
`isPlayMusicGame`, which is a *player preference*, not room data. A port must keep this: the
stereo's on-screen indicator follows the music setting.

## 7.4 `initial` vs `state`

Every switchable class carries both. `initial` is what the editor set and what the room resets to;
`state` is the live value. Observed evidence that houses ship with `state` already diverged from
`initial` (i.e. the author saved mid-play, or the editor toggled state without touching initial):
`kRedClock` has `initial = 1..1` but `state = 0..1` across 140 records; the same pattern holds for
every prize. In practice a fresh game must **copy `initial` into `state` on room entry** for all
classes.

`kGreaseRt`/`kGreaseLf` invert the convention. `ObjectInfo.c:1892-1903` presents `data.c.initial`
as a "grease already spilled" checkbox and stores the *negation*, so `initial == false` means "start
spilled". Observed: grease `initial = 0..1` and `state = 0..1`, with `length` 0..474 for `kGreaseRt`
and 0..472 for `kGreaseLf`.

Non-boolean junk really occurs in these fields. Across all 22 houses the only offenders are
three `kInvisBlower` records with `initial == 23` and `state == 23`. Any port must treat these
`Boolean` bytes as **"non-zero means true"**, not "must equal 1".

---

# 8. The link model

## 8.1 Encoding

Links live in two fields that alias to the same bytes:

- transports use `data.d.where` (record bytes 8–9) and `data.d.who` (byte 10);
- switches and triggers use `data.e.where` and `data.e.who` — same offsets.

`where` is a room key: `suite * 100 + (floor + kNumUndergroundFloors)` where
`kNumUndergroundFloors` = 8 (`GliderPRO/Headers/GliderDefines.h:535`). `who` is the destination's
object slot 0..23.

Unlinked is `where == -1` (on disk `0xFFFF`) **and** `who == 255` (`0xFF`).

Decoding, `ExtractFloorSuite` (`GliderPRO/Sources/Link.c:41-53`):

```c
void ExtractFloorSuite (short combo, short *floor, short *suite)
{
	if ((*thisHouse)->version < 0x0200)		// old floor/suite combo
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

**The two halves are swapped between house version 1 and version 2.** The test is against the
*currently loaded* house's `version` field, read live from the handle on every call — not against a
captured value — so a port must plumb the house version through to the decoder. All 22 shipped
houses are version `0x0200`, so the else-branch applies: suite is the high part, floor the low part.

`GetRoomNumber(floor, suite)` (`GliderPRO/Sources/Room.c:~730-759`) then **linearly searches**
`rooms[0 .. numberRooms)` for a room whose `suite` and `floor` match, returning `kRoomIsEmpty`
(−1) if there is none. Rooms are not indexed by coordinate; the array order is arbitrary.
`GetRoomFloorSuite` (`GliderPRO/Sources/Room.c:711-733`) returns false and sets
`*suite = kRoomIsEmpty` for a *deleted* room, which the format marks with `suite == -1`.

`GetRoomLinked` (`GliderPRO/Sources/Objects.c:126-173`) and `GetObjectLinked`
(`GliderPRO/Sources/Objects.c:177-215`) are the accessors: they switch on `what` to pick
`data.d.*` vs `data.e.*`, return −1 for `where == -1` / `who == 255`, and otherwise
`ExtractFloorSuite` + `GetRoomNumber`.

`DoLink` (`GliderPRO/Sources/Link.c:270-319`) writes the pair; `DoUnlink`
(`GliderPRO/Sources/Link.c:325-361`) writes `where = -1, who = 255`.

## 8.2 Verified end-to-end in the `Sampler` house

`Sampler` has 2 rooms. Decoded:

- Room index 0, name `Entrance`, floor 1, suite 67.
- Room index 1, name `Welcome`, floor 1, suite 66.

(`firstRoom` in the house header is 1, i.e. play starts in `Welcome`, not in room index 0.
Room order on disk is not play order.)

In `Entrance` (room index 0):

- slot 4 is `kFloorTrans` with `where = 6709`, `who = 5`. Decode: suite = 6709/100 = 67,
  floor = 6709 % 100 − 8 = 9 − 8 = 1. That is `Entrance` itself, object slot 5 — which is its
  own `kCeilingTrans`. A self-linked duct pair.
- slot 6 is `kMailboxLf` with `where = 6609`, `who = 2`. Decode: suite 66, floor 1 → room
  `Welcome`, slot 2.

Unlinked objects in the same house read `where = 0xFFFF`, `who = 0xFF`, `wide = 0x00`.

## 8.3 The second "unlinked" encoding: `where == -100`

Across all 22 shipped houses there are **exactly two** negative `where` values, and no others:
`-1` (815 records) and `-100` (165 records). Every one of the 165 `-100` records also has
`who == 255`.

| Breakdown | Counts |
|---|---|
| By house | Slumberland 69, Land of Illusion 48, Rainbow's End 36, Castle o' the Air 12 |
| By type | `kCeilingTrans` 110, `kInvisTrans` 38, `kFloorTrans` 12, `kMailboxRt` 2, `kThermostat` 1, `kKnifeSwitch` 1, `kLightSwitch` 1 |

`-100` is exactly what you get by round-tripping `-1` through the **version-1** decode and the
**version-2** encode. `ExtractFloorSuite` is version-sensitive
(`GliderPRO/Sources/Link.c:41-53`) — for `version < 0x0200` it reads floor from the *high* part and
suite from the *low* part, i.e. the two fields are swapped relative to v2:

```c
void ExtractFloorSuite (short combo, short *floor, short *suite)
{
	if ((*thisHouse)->version < 0x0200)		// old floor/suite combo
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

and `MergeFloorSuite` only ever encodes the v2 order
(`GliderPRO/Sources/Link.c:34-37`, `return ((suite * 100) + floor);`, with the caller responsible
for having already added `kNumUndergroundFloors` to `floor`). So the conversion of a v1 combo is:

1. v1 decode: `floor = (combo / 100) - 8`, `suite = combo % 100`.
2. `floor += kNumUndergroundFloors` → `floor = combo / 100`.
3. v2 encode: `result = (combo % 100) * 100 + (combo / 100)`.

Applied to `combo == -1` (C truncates toward zero, so `-1 / 100 == 0` and `-1 % 100 == -1`):
`result = (-1 * 100) + 0 = -100`. **That is the whole explanation.**

The converter is `ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746-820`), and it converts
exactly two type groups:

| Group | Line | Types | Field |
|---|---|---|---|
| transports | :779-791 | `kMailboxLf`, `kMailboxRt`, `kFloorTrans`, `kCeilingTrans`, `kInvisTrans`, `kDeluxeTrans` | `data.d.where` |
| switches | :793-807 | `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch`, `kInvisSwitch`, `kTrigger`, `kLgTrigger` | `data.e.where` |

Two deliberate exclusions confirm the design: **stairs, doors and windows are absent** (they carry
no link), and **`kSoundTrigger` is absent** (its `where` is a `snd ` resource ID, not a room code —
converting it would have destroyed it). And indeed, in the shipped data no stair, door, window or
sound trigger record carries `-100`: every `-100` record belongs to one of the 14 types above.

The shipped 1.0.4 converter guards the transform:

```c
if (thisRoom->objects[h].data.d.where != -1)
{
	ExtractFloorSuite(thisRoom->objects[h].data.d.where, &floor, &suite);
	floor += kNumUndergroundFloors;
	thisRoom->objects[h].data.d.where = MergeFloorSuite(floor, suite);
}
```

so with *this* build `-1` stays `-1`. The 165 `-100` records therefore came from an **earlier build
whose conversion loop lacked the `!= -1` guard** — the type distribution matches the two lists above
exactly, which no other mechanism explains. After conversion the routine sets
`(*thisHouse)->version = kHouseVersion` (`:814`), which is why all 22 shipped houses report
`0x0200`.

At runtime `-100` and `-1` behave identically: `ExtractFloorSuite(-100)` gives `suite = -1`,
`floor = -8`, and `GetRoomNumber(-8, -1)` finds no room and returns `-1`. **A port must treat any
`where` that fails to resolve as unlinked, not only the literal `-1`.**

Note also that `ShiftWholeHouse` (`GliderPRO/Sources/House.c:824-859`) is a **stub**: it has
`#pragma unused (howFar)` and its inner `for (h = 0; h < kMaxRoomObs; h++) { }` body is empty. It
walks every room, spins the cursor, and changes nothing.

## 8.4 Legal link targets

`UpdateLinkControl` (`GliderPRO/Sources/Link.c:57-194`) enumerates, for each of the three
link-source kinds, exactly which object types may be the destination. The mode constants are at
`GliderPRO/Headers/GliderDefines.h:463-465`: `kSwitchLinkOnly` 3, `kTriggerLinkOnly` 4,
`kTransportLinkOnly` 5.

**`kSwitchLinkOnly`** — a switch (`kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`,
`kKnifeSwitch`, `kInvisSwitch`) may link to:

- the 11 switchable blowers: `kFloorVent`, `kCeilingVent`, `kFloorBlower`, `kCeilingBlower`,
  `kSewerGrate`, `kLeftFan`, `kRightFan`, `kInvisBlower`, `kGrecoVent`, `kSewerBlower`, `kLiftArea`
- 10 prizes (`Link.c:82-91`): `kRedClock`, `kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`,
  `kBattery`, `kBands`, `kFoil`, `kInvisBonus`, `kHelium` — note `kGreaseRt`, `kGreaseLf`,
  `kStar`, `kSparkle` and `kSlider` are *not* switchable
- `kDeluxeTrans`
- all 8 lights
- 9 appliances: `kShredder`, `kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`,
  `kStereo`, `kMicrowave`
- 8 enemies: `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish`

**`kTriggerLinkOnly`** — a `kTrigger` / `kLgTrigger` may link to:

- `kGreaseRt`, `kGreaseLf`
- `kToaster`, `kGuitar`, `kCoffee`, `kOutlet`
- `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kDrip`, `kFish`
- plus the 6 real switches, **but only when `linkRoom == thisRoomNumber`** (a trigger can throw a
  switch only in its own room)

**`kTransportLinkOnly`** — a transport (`kMailboxLf`, `kMailboxRt`, `kFloorTrans`, `kCeilingTrans`,
`kInvisTrans`, `kDeluxeTrans`) may link to:

- `kMailboxLf`, `kMailboxRt`, `kCeilingTrans`, `kInvisTrans`, `kDeluxeTrans`
- `kInvisLight`
- `kOzma`, `kMirror`, `kFireplace`, `kWallWindow`, `kCalendar`, `kBulletin`, `kCloud`

The last group is the trick that makes a mirror or a fireplace a teleport destination: the arriving
glider is positioned relative to the *clutter* object's rect. Note `kFloorTrans` is **not** a legal
destination — ducts always terminate at a ceiling duct.

Two helper predicates classify a source:

- `ObjectIsLinkTransport` (`GliderPRO/Sources/Objects.c:219-232`) → `kMailboxLf`, `kMailboxRt`,
  `kFloorTrans`, `kCeilingTrans`, `kInvisTrans`, `kDeluxeTrans`.
- `ObjectIsLinkSwitch` (`GliderPRO/Sources/Objects.c:236-250`) → the 6 real switches plus
  `kTrigger` and `kLgTrigger`. **`kSoundTrigger` is deliberately excluded** because its `where`
  is a sound ID, not a room key.

## 8.5 Observed link statistics

Of the 31 440 live objects, **3411 belong to a class with a `who` field** (variants `d` and `e`).
Full histogram of `who` over those 3411 records:

```
0:203  1:172  2:149  3:120  4:141  5:190  6:146  7:123  8:131  9:117
10:100 11:123 12:79  13:77  14:61  15:60  16:62  17:58  18:44  19:41
20:21  21:31  22:23  23:22  35:1   255:1116
```

Sum = 3411 exactly. Reading it:

- **`who == 255` (1116 records) means "not linked."** `GetObjectLinked`
  (`GliderPRO/Sources/Objects.c:177-211`) maps 255 → −1 for both the transport and switch groups,
  and `ListAllLocalObjects` only sets `localLink` when **both** `roomLink != -1` **and**
  `objectLink != -1`. The 1116 decompose exactly:
  815 with `where == -1`, 165 with `where == -100`, 122 `kSoundTrigger` (whose `where` is a sound
  ID), and **14 half-broken links** — records whose `where` is a perfectly valid room code but
  whose `who` is the unlinked sentinel: `kCeilingTrans` 7, `kInvisTrans` 5, `kLightSwitch` 1,
  `kMailboxLf` 1; 12 of the 14 are in *CD Demo House* (e.g. room 33 slot 8 `kInvisTrans`
  `where = 1410, who = 255`) and the other 2 are in *Leviathan* (room 336 slot 1 `kInvisTrans`
  `where = 554`, room 389 slot 1 `kCeilingTrans` `where = 1375`). They behave as unlinked.
- The converse never happens: **0 records** have a real `who` with `where` set to −1 or −100.
- **One record has `who == 35`, which is out of range** (`kMaxRoomObs` is 24) — a corrupt link in
  shipped content. Neither `GetObjectLinked` nor the `localLink` correlation loop bounds-checks it;
  it simply fails to match any `masterObjects[n].objectNum` and stays unlinked. A port must
  bounds-check `who` before indexing anyway.
- `who` ∈ 0..23 (3411 − 1116 − 1 = 2294 records) is a slot index within the linked room. The
  monotone decline from 203 at slot 0 to 22 at slot 23 simply mirrors how full rooms are.

Note that `GetRoomLinked` (`GliderPRO/Sources/Objects.c:126-172`) tests only `!= -1`, so a `-100`
`where` *is* fed through `ExtractFloorSuite` and `GetRoomNumber`; the linear search finds no room
with `(floor, suite) == (-8, -1)` and returns −1. Same outcome, longer path.

`kSoundTrigger` never links: `who = 255` for all 122 records, `type = 0`, and
`where = 3000..10000` (sound resource IDs). `kTrigger` and `kLgTrigger` always have
`type = 3` (`kOneShot`) and `who` 0..23 — they are *always* linked.

---

# 9. Dynamic objects: what animates

## 9.1 The registry

`kMaxDynamicObs` = 18 (`GliderPRO/Headers/GliderDefines.h:265`). The array is `dinahs[18]` of
`dynaType` (`GliderPRO/Headers/GliderStructs.h:310-320`):

```c
typedef struct
{
	Rect		dest;
	Rect		whole;
	short		hVel, vVel;
	short		type, count;
	short		frame, timer;
	short		position, room;
	Byte		byte0, byte1;
	Boolean		moving, active;
} dynaType, *dynaPtr;
```

`ZeroDinahs` (`GliderPRO/Sources/Dynamics3.c:160-180`) marks slots free with
`type = kObjectIsEmpty` (−1).

## 9.2 The 17 animated types (15 `case` groups)

`AddDynamicObject(short what, Rect *where, objectType *who, short room, short index, Boolean isOn)`
(`GliderPRO/Sources/Dynamics3.c:187-554`) returns −1 if `numDynamics >= kMaxDynamicObs` **or if
`what` is not one of the 17 handled types** (`default: return (-1);` at `:546-547`). The 17 types are
grouped into 15 `case` groups — `kCopterLf`/`kCopterRt` and `kDartLf`/`kDartRt` share an arm — which
is why the table below has 15 rows. Every arm sets `byte0 = (Byte)index` (the room's object slot),
`byte1 = 0`, `moving = false` and `active = isOn`; all of them also set `room = room` **except the
`kDartLf`/`kDartRt` arm** (`:437-463`), which is bug 8 in §14.

| ID | Type | Seed values | Animated by |
|---|---|---|---|
| `0x2D` | `kSparkle` | `dest = sparkleSrc[0]` at `(left, top)`; `timer = RandomInt(60) + 15` | `HandleDynamics` only |
| `0x62` | `kToaster` | `dest = breadSrc[0]` centred horizontally at rect top; `hVel = where->top + 2` (a clip line, **not a velocity**); then a ballistic solve (below); `frame = timer = (short)who->data.g.delay * 3` | `HandleDynamics` + `RenderDynamics` |
| `0x63` | `kMacPlus` | `dest = plusScreen1` at `topLeft + (10, 7)` + `playOrigin` | `HandleDynamics` |
| `0x65` | `kTV` | `dest = tvScreen1` at `topLeft + (17, 10)` | `HandleDynamics` |
| `0x66` | `kCoffee` | `dest = coffeeLight1` at `topLeft + (32, 57)`; `timer = isOn ? 200 : 0` | `HandleDynamics` |
| `0x67` | `kOutlet` | `dest = outletSrc[0]`; `hVel = numLights`; `count = timer = ((short)who->data.g.delay * 6) / kTicksPerFrame` | `HandleDynamics` |
| `0x68` | `kVCR` | `dest = vcrTime1` at `topLeft + (64, 6)`; `timer = isOn ? 115 : 0` | `HandleDynamics` |
| `0x69` | `kStereo` | `dest = stereoLight1` at `topLeft + (56, 20)` | `HandleDynamics` |
| `0x6A` | `kMicrowave` | `dest = microOn` at `topLeft + (14, 13)`, then `right = left + 48` | `HandleDynamics` |
| `0x71` | `kBalloon` | `dest.bottom = kBalloonStart` (**310**); `vVel = -2`; `count = timer = (delay * 6) / 2` | both |
| `0x72`/`0x73` | `kCopterLf`/`kCopterRt` | `dest.top = kCopterStart` (**8**); `hVel = -1` / `+1`; `vVel = 2`; `position = dest.left` | both |
| `0x74`/`0x75` | `kDartLf`/`kDartRt` | `kDartLf`: `x = kRoomWide - RectWide(dartSrc[0])`, `hVel = -kDartVelocity` (**−6**), `frame = 0`. `kDartRt`: `x = 0`, `hVel = +6`, `frame = 2`. Both `vVel = 2`, `position = dest.top` | both |
| `0x76` | `kBall` | `position = who->data.h.length`; velocity accumulated on alternate frames (`lilFrame`) → `vVel = count = -velocity`; then `position = dest.bottom` | both |
| `0x77` | `kDrip` | `dest = dripSrc[0]` centred; `hVel = dest.top` (remembered origin); `count = timer = (delay * 6) / 2`; `frame = 3`; `position = dest.top + who->data.h.length` | both |
| `0x78` | `kFish` | `hVel = ((short)who->data.h.delay * 6) / kTicksPerFrame`; `position = who->data.g.height` (**union aliasing** — same offset as `data.h.length`); alternate-frame velocity like `kBall`; `timer = hVel` | both |

`GliderPRO/Sources/Dynamics3.c:13-15`:

```c
#define kBalloonStart		310
#define kCopterStart		8
#define kDartVelocity		6
```

`kTicksPerFrame` = 2 (`GliderPRO/Headers/GliderDefines.h:533`), so `(delay * 6) / kTicksPerFrame`
= `delay * 3`.

`HandleDynamics` (`GliderPRO/Sources/Dynamics3.c:31-105`) advances all slots; `RenderDynamics`
(`:112-154`) draws only the 9 types that move (`kToaster`, `kBalloon`, `kCopterLf`, `kCopterRt`,
`kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish`) — the appliance indicators are drawn in place by
`HandleDynamics` itself.

### The toaster's ballistic solve

```
1.  position = who->data.g.height;      // launch height in pixels
2.  velocity = 0;
3.  do {
4.      velocity++;
5.      position -= velocity;
6.  } while (position > 0);
7.  vVel   = -velocity;
8.  count  =  velocity;
```

This inverts constant-acceleration motion: it finds the smallest integer `velocity` whose
triangular number `1+2+…+velocity` reaches `height`, and launches the toast with that upward
velocity so it peaks at approximately `height` pixels. Observed `data.g.height` in shipped houses:
**9..277** across 140 toasters. A Go port must reproduce the *loop*, not a closed-form
`sqrt(2h)`, or trajectories will differ by a pixel here and there.

## 9.3 Non-animated but registered

Every animated type is also the subject of a `kIgnoreIt` (or `kDissolveIt`) hot spot so switches can
find it — see §6.3. Conversely, `kSparkle` has **no** hot spot at all and is *only* a dynamic
object.

`kFish` and `kCobweb` are in the enemy class but only `kFish` animates; `kCobweb` is a static hot
spot. `kBall` and `kDrip` animate but have `kIgnoreIt` hot spots — their damage is dealt by the
dynamics collision path.

---

# 10. Drawing dispatch

`DrawARoomsObjects(short neighbor, Boolean redraw)`
(`GliderPRO/Sources/ObjectDrawAll.c:23-965`) is the single place every object type's visual
representation is selected. Structure:

1. If `localNumbers[neighbor] == kRoomIsEmpty` → return.
2. `isLit = (numLights > 0)` — the room's lighting is a *count of lit lights*, not a flag.
3. For `i` in 0..`kMaxRoomObs`−1, reset `dynamicNum = legit = -1`, then
   `if (IsThisValid(localNumbers[neighbor], i)) { ... }` — note this is an `if`, **not** a `continue`:
   step 5 runs for every slot, valid or not.
4. Inside that `if`: `switch (thisObject.what)` → per-type drawing. Almost every arm is
   `GetObjectRect` → `OffsetRectRoomRelative(&itsRect, neighbor)` → `if (SectRect(&itsRect,
   &testRect, &whoCares) && isLit) Draw…`, where `testRect` is `houseRect` with its corner zeroed.
5. After each slot, if `!redraw`, every `masterObjects[n]` whose `(objectNum, roomNum)` matches gets
   `dynaNum = dynamicNum` (the dynamic slot just allocated, or −1).

Per-type routine, exhaustively:

| Types | Routine / behaviour |
|---|---|
| `kFloorVent`, `kCeilingVent`, `kFloorBlower`, `kCeilingBlower`, `kSewerGrate`, `kGrecoVent`, `kSewerBlower` | `DrawSimpleBlowers` — **only when `isLit`** |
| `kLeftFan`, `kRightFan` | `DrawSimpleBlowers`, lit only |
| `kTaper`, `kCandle`, `kStubby` | draw sprite + `AddCandleFlame` at `topLeft + (10, 7)` / `(14, 7)` / `(9, 7)`. For a *neighbour* room, the flame is added only if the 16 × 15 flame rect does **not** intersect the central room expanded by `kFloorSupportTall` (44) |
| `kTiki` | `DrawTiki` + `AddTikiFlame` at `topLeft + (10, −9)` |
| `kBBQ` | `DrawPictSansWhiteObject` + `AddBBQCoals` at `topLeft + (16, 9)` |
| `kInvisBlower`, `kLiftArea`, `kInvisObstacle`, `kInvisBounce`, `kInvisBonus`, `kSlider`, `kInvisTrans`, `kDeluxeTrans`, `kInvisLight`, `kTrigger`, `kLgTrigger`, `kSoundTrigger` | **nothing drawn** (12 invisible types) |
| `kTable` | `DrawTable(playOriginV)` |
| `kShelf` | `DrawShelf` |
| `kCabinet` | `DrawCabinet` |
| `kFilingCabinet`, `kOzma` | `DrawPictObject` |
| `kWasteBasket`, `kMilkCrate` | `DrawSimpleFurniture` |
| `kCounter` | `DrawCounter` |
| `kDresser` | `DrawDresser` |
| `kDeckTable` | `DrawDeckTable` |
| `kStool` | `DrawStool(playOriginV + VerticalRoomOffset(neighbor))` |
| `kManhole` | `AddTempManholeRect` + `DrawPictSansWhiteObject` |
| `kRedClock`, `kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`, `kBattery`, `kBands`, `kHelium`, `kFoil`, `kStar` | first `BackUpToSavedMap` / `ReBackUpSavedMap` (saves the background so the prize can vanish later); draw only if the returned `legit != -1`. `kCuckoo` additionally `AddPendulum(left + 4, top + 46)`; `kStar` additionally `AddStar` |
| `kGreaseRt`, `kGreaseLf` | if `data.c.state` (standing): `AddGrease` / `ReBackUpGrease`, then `DrawGreaseRt/Lf(rect, length, true)`; else `DrawGreaseRt/Lf(rect, length, false)` |
| `kSparkle` | `AddDynamicObject(kSparkle, …, data.c.state)` — central room only, and only when `!redraw` |
| `kUpStairs`, `kDoorInLf`, `kDoorInRt`, `kWindowInLf`, `kWindowInRt` | `DrawPictSansWhiteObject` |
| `kDownStairs`, `kDoorExRt`, `kDoorExLf`, `kWindowExRt`, `kWindowExLf` | `DrawPictObject` |
| `kMailboxLf`, `kMailboxRt` | `DrawMailboxLeft` / `DrawMailboxRight(playOriginV + VerticalRoomOffset)` |
| `kFloorTrans`, `kCeilingTrans` | `DrawSimpleTransport` |
| `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch` | decode `data.e.where` / `data.e.who` via `ExtractFloorSuite` + `GetRoomNumber`, then draw the switch in the pose given by `GetObjectState(linkedRoom, linkedObj)` — a switch's appearance is derived from **its target's** state, not its own |
| `kCeilingLight`, `kLightBulb`, `kTableLamp` | `DrawSimpleLight` |
| `kFlourescent` | `DrawFlourescent` |
| `kTrackLight` | `DrawTrackLight` |
| `kHipLamp`, `kDecoLamp` | `DrawPictSansWhiteObject` |
| `kCustomPict` | `DrawCustPictSansWhite(data.g.height, &itsRect)` |
| `kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`, `kMicrowave` | `AddDynamicObject(what, …, data.g.state)`. `kStereo` draws with the global `isPlayMusicGame` rather than its own state |
| `kShredder`, `kCDs` | `DrawSimpleAppliance` |
| `kGuitar`, `kCinderBlock`, `kFlowerBox` | `DrawPictSansWhiteObject` |
| `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall` | `AddDynamicObject(what, …, data.h.state)` in the **central room only**, with no static draw at all |
| `kDrip` | `DrawDrip` + `AddDynamicObject` |
| `kFish` | `DrawFish` + `AddDynamicObject` |
| `kCobweb`, `kCloud` | `DrawPictWithMaskObject` |
| `kMirror` | `DrawMirror` + `AddToMirrorRegion(InsetRect(bounds, 4, 4))` |
| `kMousehole`, `kFaucet` | `DrawSimpleClutter` |
| `kFlower` | `DrawFlower(&itsRect, data.i.pict)` |
| `kWallWindow` | `DrawWallWindow` — **not** gated on `isLit` |
| `kCalendar` | `DrawCalendar` |
| `kBulletin` | `DrawBulletin` |
| `kTrunk`, `kBooks`, `kFireplace`, `kBear`, `kVase1`, `kVase2`, `kRug`, `kChimes` | `DrawPictSansWhiteObject` |

Three drawing primitives matter for a port:

- `DrawPictObject` — straight `CopyBits` of the PICT.
- `DrawPictSansWhiteObject` — `CopyBits` with white treated as transparent (`srcCopy` against a
  derived mask). Classic Mac 8-bit indexed colour: "white" is a *palette index*, not RGB.
- `DrawPictWithMaskObject` — uses the explicit companion mask PICT at ID + 1000.

`kRedOrangeColor8` = 23 with the source comment "actually, 18"
(`GliderPRO/Headers/GliderDefines.h:542`) — a hard-coded palette index that a Go port must map to
an RGB value from the original 8-bit CLUT.

---

# 11. Creation defaults and per-room caps

`AddNewObject(Point where, short what, Boolean showItNow)`
(`GliderPRO/Sources/ObjectAdd.c:48-801`) is the editor's object factory and therefore the
authoritative statement of every field's default value. Local constants at
`GliderPRO/Sources/ObjectAdd.c:15-23`:

```c
#define kMaxSoundTriggers	1
#define kMaxStairs			1
#define kMouseholeBottom	295
#define kFireplaceBottom	297
#define kManholeSits		322
#define kGrecoVentTop		303
#define kSewerBlowerTop		292
```

## 11.1 Defaults per class

### Blowers

| Field | Default | Exceptions |
|---|---|---|
| `distance` | 64 | ceiling vents/blowers 32; `kLeftFan` 32; `kRightFan` 32 |
| `initial` | true | — |
| `state` | true | — |
| `vector` | `0x01` (up) | ceiling vent/blower `0x04` (down); `kLeftFan` `0x08` (left); `kRightFan` `0x02` (right) |
| `tall` | `0x00` | `kLiftArea` `0x10` (= 16, giving a 32-pixel-tall rect) |
| `topLeft.v` | forced per type (see §4.2) | `kInvisBlower`, `kLiftArea`, `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ`, `kLeftFan`, `kRightFan` are free-placed |

### Furniture

`bounds` = the type's default `srcRects` size centred on the click point, then:
`kCounter.bottom = kCounterBottom` (304), `kDresser.bottom = kDresserBottom` (293),
`kManhole` snapped to `((left − 3) / 64) * 64 + 3` with `bottom = kManholeSits` (322).
`pict = 0` always.

### Prizes / bonuses

| Field | Default | Exceptions |
|---|---|---|
| `length` | 0 | `kGreaseRt`/`kGreaseLf` **64**; `kSlider` **64** |
| `points` | 0 | `kInvisBonus` **100** |
| `state` | true | — |
| `initial` | true | — |

### Transport

| Field | Default | Exceptions |
|---|---|---|
| `tall` | 0 | `kInvisTrans` = the created rect's height; **`kDeluxeTrans` = `0x1010`** (16 × 4 by 16 × 4 = 64 × 64) |
| `where` | −1 | — |
| `who` | 255 | — |
| `wide` | 0 | **`kDeluxeTrans` = `0x10`** (initial on, state off) |
| `topLeft` | click point | stairs `v = kStairsTop` (28); doors/windows snapped to the fixed edges chosen by `where.h > kRoomWide/2` |

### Switches

| Field | Default | Exceptions |
|---|---|---|
| `delay` | 0 | — |
| `where` | −1 | **`kSoundTrigger` = 3000** (the first legal sound ID) |
| `who` | 255 | — |
| `type` | `kToggle` (0) | **`kTrigger`, `kLgTrigger` = `kOneShot` (3)** |

### Lights

| Field | Default | Exceptions |
|---|---|---|
| `length` | 0 | `kCeilingLight`, `kFlourescent`, `kTrackLight` = **64** |
| `byte0`, `byte1` | 0 | — |
| `initial`, `state` | true | — |
| `topLeft.v` | click | forced per type (`kCeilingLightTop` 4, `kHipLampTop` 23, `kDecoLampTop` 91, `kFlourescentTop` 12, `kTrackLightTop` 5) |

### Appliances

| Field | Default | Exceptions |
|---|---|---|
| `height` | 0 | `kToaster` **64**; `kCustomPict` **10000** |
| `byte0` | 0 | `kMicrowave` **7** (all three kill bits set) |
| `delay` | 0 | `kToaster` and `kOutlet` = **`10 + RandomInt(10)`** |
| `initial`, `state` | true | — |

### Enemies

| Field | Default | Exceptions |
|---|---|---|
| `length` | 0 | `kBall`, `kDrip`, `kFish` = **64** |
| `delay` | `10 + RandomInt(10)` | `kCobweb`, `kBall` = **0** |
| `byte0` | 0 | — |
| `initial`, `state` | true | — |
| `topLeft` | click | `kDartLf.h = kRoomWide − width`; `kDartRt.h = 0`; `kBalloon`, `kCopterLf`, `kCopterRt` get `v = (kTileHigh / 2) − halfTall` = 146 (`ObjectAdd.c:676-680`). `kBall`, `kDrip`, `kFish` are in a **separate** `case` group (`:695-719`) and are free-placed at the click point in both axes — they do *not* get the 146 default. |

### Clutter

`bounds` = default size centred on the click, then `kMousehole.bottom = 295`,
`kFireplace.bottom = 297`. `kFlower` picks a **random** `pict` in 0..5 unless the Shift key is held
(in which case it keeps the last-used flower) and sets `bounds` from `flowerSrc[pict]`
(`GliderPRO/Sources/ObjectAdd.c:743`).

## 11.2 Per-room caps beyond `kMaxRoomObs`

Enforced only by the editor, via the `HowMany*Objects()` counters at
`GliderPRO/Sources/ObjectAdd.c:888-1073`. A hand-edited or converted house may violate them; the
*runtime* arrays are the real limits and overflow silently.

| Cap | Value | Constant | Applies to | Counter |
|---|---|---|---|---|
| candles | 20 | `kMaxCandles` | `kTaper` + `kCandle` + `kStubby` **combined** | `HowManyCandleObjects` :888-902 |
| tiki torches | 8 | `kMaxTikis` | `kTiki` | `HowManyTikiObjects` :904-916 |
| barbecues | 8 | `kMaxCoals` | `kBBQ` | `HowManyBBQObjects` :918-930 |
| cuckoo clocks | 8 | `kMaxPendulums` | `kCuckoo` | `HowManyCuckooObjects` :932-944 |
| rubber bands | 2 | `kMaxRubberBands` | `kBands` | `HowManyBandsObjects` :946-958 |
| grease | 16 | `kMaxGrease` | `kGreaseRt` + `kGreaseLf` | `HowManyGreaseObjects` :960-973 |
| stars | 4 | `kMaxStars` | `kStar` | `HowManyStarsObjects` :975-987 |
| sound triggers | 1 | `kMaxSoundTriggers` | `kSoundTrigger` | `HowManySoundObjects` :989-1001 |
| up stairs | 1 | `kMaxStairs` | `kUpStairs` | `HowManyUpStairsObjects` :1003-1015 |
| down stairs | 1 | `kMaxStairs` | `kDownStairs` | `HowManyDownStairsObjects` :1017-1029 |
| shredders | 4 | `kMaxShredded` | `kShredder` | `HowManyShredderObjects` :1031-1043 |
| dynamic objects | 18 | `kMaxDynamicObs` | the 17-type list below | `HowManyDynamicObjects` :1045-1073 |

The dynamic-object counter's type list (`GliderPRO/Sources/ObjectAdd.c:1051-1067`) is
**`kSparkle`, `kToaster`, `kMacPlus`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`,
`kMicrowave`, `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip`,
`kFish`** — exactly the same 17 types `AddDynamicObject` handles (§9.2), in the same order, so the
editor's counter and the runtime registry agree. (`kSparkle` also feeds a per-room budget of 18
rather than the
`kMaxSparkles` = 3 that gates the *visual* sparkle effects.)

Other runtime array bounds that constrain object counts indirectly
(`GliderPRO/Headers/GliderDefines.h:249-268`): `kMaxHotSpots` 56, `kMaxSavedMaps` 24,
`kMaxSparkles` 3, `kMaxFlyingPts` 3, `kMaxFlyingPointsLoop` 24, `kMaxMasterObjects` 216.
All are allocated once in `CreatePointers` (`GliderPRO/Sources/StructuresInit2.c:187-299`).

## 11.3 Triggers

`GliderPRO/Sources/Triggers.c`:

```c
#define kMaxTriggers	16
typedef struct { short object, room, index, timer, what; Boolean armed; } trigType;
```

`ArmTrigger` (`:34-55`) sets
`timer = masterObjects[whoLinked].theObject.data.e.delay * 3` — so `delay` is in units of
3 frames, i.e. `delay * 3 * kTicksPerFrame` = `delay * 6` ticks ≈ `delay / 10` seconds at 60 Hz.
Observed `delay` in shipped houses: 0..120 for both `kTrigger` and `kLgTrigger`.

`HandleTriggers` (`:79-96`) decrements each armed timer once per frame and fires at 0.
`FireTrigger` (`:100-194`) dispatches on the **linked** object's type:

| Linked type | Line | Effect |
|---|---|---|
| `kGreaseRt`, `kGreaseLf` | :112-121 | `if (SetObjectState(triggers[index].room, triggers[index].object, kForceOn, triggeredIs)) SpillGrease(dynaNum, hotNum)` |
| `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch`, `kInvisSwitch` | :122-129 | `TriggerSwitch(masterObjects[triggeredIs].dynaNum)` |
| `kSoundTrigger` | :131-133 | `PlayPrioritySound(kChordSound, kChordPriority);` with the source comment `// Change me` — it plays the **guitar chord**, *not* the sound named by `data.e.where`. A trigger-fired sound trigger and a fly-over sound trigger make different noises. |
| `kGuitar` | :139-141 | `PlayPrioritySound(kChordSound, kChordPriority)` |
| `kCoffee` | :143-145 | `PlayPrioritySound(kCoffeeSound, kCoffeePriority)` |
| `kToaster` | :135-137 | `TriggerToast(dynaNum)` |
| `kOutlet` | :147-149 | `TriggerOutlet(dynaNum)` |
| `kBalloon` | :151-153 | `TriggerBalloon(dynaNum)` |
| `kCopterLf`, `kCopterRt` | :155-158 | `TriggerCopter(dynaNum)` |
| `kDartLf`, `kDartRt` | :160-163 | `TriggerDart(dynaNum)` |
| `kDrip` | :165-167 | `TriggerDrip(dynaNum)` |
| `kFish` | :169-171 | `TriggerFish(dynaNum)` |

Anything else linked to a trigger does nothing at all — notably `kMacPlus`, `kTV`, `kVCR`,
`kStereo`, `kMicrowave`, `kShredder`, `kBall`, and every prize other than grease.

## 11.4 Reward values

`HandleRewards` (`GliderPRO/Sources/Interactions.c:756-981`), per prize. Every branch ends with
`who->isOn = false` so a prize can be collected only once.

Every arm is gated on `SetObjectState(thisRoomNumber, masterObjects[whoLinked].objectNum, 0, whoLinked)`
returning true — i.e. the prize is only paid out if the state actually changed from on to off. The
`action` argument is the literal `0`, which is `kToggle`.

| Type | Line | Sound | Velocity | Effect | Doubles in 2P |
|---|---|---|---|---|---|
| `kRedClock` | :766-780 | `kBeepsSound` | `hVel /= 4`, `vVel /= 4` | `AddFlyingPoint(&bounds, 100, hVel/2, vVel/2)`; `theScore += kRedClockPoints` (100) | no |
| `kBlueClock` | :782-796 | `kBuzzerSound` | `/= 4` | `AddFlyingPoint(..., 300, ...)`; `theScore += 300` | no |
| `kYellowClock` | :798-812 | `kDingSound` | `/= 4` | `AddFlyingPoint(..., 500, ...)`; `theScore += 500` | no |
| `kCuckoo` | :814-829 | `kCuckooSound` | `/= 4` | `StopPendulum`; `AddFlyingPoint(..., 1000, ...)`; `theScore += 1000` | no |
| `kPaper` | :831-848 | `kEnergizeSound` | `hVel /= 2`, `vVel /= 2` | `AddSparkle`; `mortals++` (extra glider); `QuickGlidersRefresh` | **yes** |
| `kBattery` | :850-870 | `kEnergizeSound` | `/= 2` | `batteryTotal += 50` if `batteryTotal > 0`, else `batteryTotal = 50` | **yes** (`+= 50` again) |
| `kBands` | :872-889 | `kEnergizeSound` | `/= 2` | `bandsTotal += 8`; `QuickBandsRefresh` | **yes** |
| `kFoil` | :900-917 | `kEnergizeSound` | `/= 2` | `foilTotal += 8`; `StartGliderFoilGoing` | **yes** |
| `kInvisBonus` | :919-931 | `kBonusSound` | `/= 4` | `points = data.c.points` read **before** the `SetObjectState` call (`:920`); `AddFlyingPoint(&bounds, points, ...)`; `theScore += points` | no |
| `kStar` | :933-951 | `kEnergizeSound` | unchanged | `AddSparkle`; `StopStar`; `numStarsRemaining--`; `FlagGameOver()` if `<= 0` else `DisplayStarsRemaining()`; `theScore += 5000` | no |
| `kHelium` | :956-976 | `kEnergizeSound` | `/= 2` | `batteryTotal -= 150` if `batteryTotal < 0`, else `batteryTotal = -150` | **yes** (`-= 150` again) |
| `kGreaseRt`, `kGreaseLf` | :891-898 | none | unchanged | `SpillGrease(dynaNum, hotNum)`; no score | n/a |
| `kSparkle` | :953-954 | — | — | empty arm — falls straight through | n/a |
| `kSlider` | :978-979 | — | — | empty arm | n/a |

The 2-player doubling condition is always `if ((twoPlayerGame) && (!onePlayerLeft))`. Note that
`batteryTotal` is a **signed** reservoir: positive = battery thrust, negative = helium buoyancy, and
crossing zero *replaces* rather than accumulates. Every arm that removes an object also calls
`RestoreFromSavedMap(thisRoomNumber, masterObjects[whoLinked].objectNum, false)` and
`RedrawAllGrease()` — **including all four clocks** (`:777`, `:793`, `:809`, `:826`). The only
scoring arm that does neither is `kInvisBonus` (`:919-931`), which has nothing to erase.

Also `kRoomVisitScore` = 100 for entering a new room
(`GliderPRO/Headers/GliderDefines.h:536`).

`HandleMicrowaveAction` (`GliderPRO/Sources/Interactions.c:1159-1194`) reads `data.g.byte0`:

```
bit 0 (value 1) set  ->  bandsTotal   = 0
bit 1 (value 2) set  ->  batteryTotal = 0
bit 2 (value 4) set  ->  foilTotal    = 0
```

Editor UI at `GliderPRO/Sources/ObjectInfo.c:1773-1811`. Observed `byte0` for `kMicrowave`
across 56 records: **0..7** — every combination is used.

---

# 12. Master per-type reference

Everything above, condensed into one row per object type. "Rect from" is how `GetObjectRect`
derives the bounding box; "Hot spot" is the action code(s) `CreateActiveRects` registers;
"Anim" is Y if `AddDynamicObject` handles it.

## Blowers — variant `a`, `blowerType`

| ID | Constant | Default W×H | Rect from | Fields used | Hot spot | Anim | Semantics |
|---|---|---|---|---|---|---|---|
| `0x01` | `kFloorVent` | 48 × 11 | srcRects + `topLeft` | `distance`, `initial`, `state`, `vector`=1 | `kLiftIt`, 4 × `distance` column above, centred | — | floor grille blowing up |
| `0x02` | `kCeilingVent` | 48 × 11 | srcRects + `topLeft` | `distance`, `initial`, `state`, `vector`=4 | `kDropIt`, 24 × `distance` column below | — | ceiling grille blowing down |
| `0x03` | `kFloorBlower` | 48 × 15 | srcRects + `topLeft` | as `kFloorVent` | `kLiftIt` | — | floor duct, taller art |
| `0x04` | `kCeilingBlower` | 48 × 15 | srcRects + `topLeft` | as `kCeilingVent` | `kDropIt` | — | ceiling duct |
| `0x05` | `kSewerGrate` | 48 × 17 | srcRects + `topLeft` | as `kFloorVent` | `kLiftIt` | — | outdoor grate |
| `0x06` | `kLeftFan` | 40 × 55 | srcRects + `topLeft` | `distance`, `state`, `vector`=8 | `kDissolveIt` (13 × 43 blade) + `kPushItLeft` (`distance` × 16) | — | table fan facing left |
| `0x07` | `kRightFan` | 40 × 55 | srcRects + `topLeft` | `distance`, `state`, `vector`=2 | `kDissolveIt` + `kPushItRight` | — | table fan facing right |
| `0x08` | `kTaper` | 20 × 59 | srcRects + `topLeft` | `distance` (flame column) | `kLiftIt`/`kBurnIt` split + `kDissolveIt` 7 × 48 | flame | tall candle |
| `0x09` | `kCandle` | 32 × 30 | srcRects + `topLeft` | `distance` | as `kTaper`, `kDissolveIt` 8 × 20 | flame | candle in a holder |
| `0x0A` | `kStubby` | 20 × 36 | srcRects + `topLeft` | `distance` | as `kTaper`, `kDissolveIt` 15 × 26 | flame | short fat candle |
| `0x0B` | `kTiki` | 27 × 28 | srcRects + `topLeft` | `distance` | as `kTaper`, `kDissolveIt` 15 × 14 | tiki flame | tiki torch |
| `0x0C` | `kBBQ` | 64 × 33 | srcRects + `topLeft` | `distance` | column `bottom = 8`, `kDissolveIt` 52 × 17 | coals | barbecue grill |
| `0x0D` | `kInvisBlower` | 24 × 24 | srcRects + `topLeft` | `distance`, `state`, **`vector` (all 4 dirs)** | one of `kLiftIt`/`kDropIt`/`kPushItLeft`/`kPushItRight`, `distance + 24` long | — | invisible air source, any direction |
| `0x0E` | `kGrecoVent` | 48 × 18 | srcRects + `topLeft` | as `kFloorVent` | `kLiftIt` | — | ornamental floor vent |
| `0x0F` | `kSewerBlower` | 32 × 12 | srcRects + `topLeft` | as `kFloorVent` | `kLiftIt` | — | small sewer vent |
| `0x10` | `kLiftArea` | 64 × 32 | **`distance` × `tall`·2** | `distance` (width!), `tall` (½ height), `state`, `vector` | direction from `vector & 0x0F` | — | invisible arbitrary-size force field |

## Furniture — variant `b`, `furnitureType`

All 15 store an explicit `bounds` and read `pict` never. All 15 are `kDissolveIt` (i.e. solid,
lethal on contact) except the two noted in bold — `kManhole` (`kIgnoreGround`) and `kInvisBounce`
(`kBounceIt`). `kStool` and `kBooks` are still `kDissolveIt`; only their rects are adjusted.

| ID | Constant | Default W×H | Hot spot | Semantics |
|---|---|---|---|---|
| `0x11` | `kTable` | 64 × 8 | `kDissolveIt` | table top; drawn with legs down to `playOriginV` |
| `0x12` | `kShelf` | 64 × 6 | `kDissolveIt` | wall shelf with brackets |
| `0x13` | `kCabinet` | 64 × 64 | `kDissolveIt` | tiled cabinet, resizable |
| `0x14` | `kFilingCabinet` | 74 × 107 | `kDissolveIt` | fixed-art filing cabinet |
| `0x15` | `kWasteBasket` | 64 × 61 | `kDissolveIt` | wastebasket |
| `0x16` | `kMilkCrate` | 64 × 58 | `kDissolveIt` | milk crate |
| `0x17` | `kCounter` | 128 × 64 | `kDissolveIt` | kitchen counter; `bottom` forced to 304 |
| `0x18` | `kDresser` | 128 × 64 | `kDissolveIt` | dresser; `bottom` forced to 293 |
| `0x19` | `kDeckTable` | 64 × 8 | `kDissolveIt` | outdoor table |
| `0x1A` | `kStool` | 48 × 38 | `kDissolveIt` on `InsetRect(1,1)` with `bottom = top + 25` | bar stool; only the seat is solid |
| `0x1B` | `kTrunk` | 144 × 80 | `kDissolveIt` | steamer trunk |
| `0x1C` | `kInvisObstacle` | 64 × 64 | `kDissolveIt` | invisible lethal block |
| `0x1D` | `kManhole` | 123 × 22 | **`kIgnoreGround`** on y 311..322 inset 51 px each side | hole in the floor you fall through |
| `0x1E` | `kBooks` | 64 × 51 | `kDissolveIt` on `bounds` with `right − 2` | row of books |
| `0x1F` | `kInvisBounce` | 64 × 64 | **`kBounceIt`** | invisible rebounder |

## Prizes / bonuses — variant `c`, `bonusType`

| ID | Constant | Default W×H | Fields used | Hot spot | Anim | Semantics |
|---|---|---|---|---|---|---|
| `0x21` | `kRedClock` | 28 × 17 | `state`, `initial` | `kRewardIt`, `isOn = state` | — | +100 points |
| `0x22` | `kBlueClock` | 28 × 25 | `state`, `initial` | `kRewardIt` | — | +300 points |
| `0x23` | `kYellowClock` | 28 × 28 | `state`, `initial` | `kRewardIt` | — | +500 points |
| `0x24` | `kCuckoo` | 40 × 80 | `state`, `initial` | `kRewardIt` | pendulum | +1000 points; adds a `pendulumType` at `topLeft + (4, 46)` |
| `0x25` | `kPaper` | 48 × 21 | `state`, `initial` | `kRewardIt` | — | extra glider (`mortals++`) |
| `0x26` | `kBattery` | 16 × 25 | `state`, `initial` | `kRewardIt` | — | +50 battery thrust |
| `0x27` | `kBands` | 28 × 23 | `state`, `initial` | `kRewardIt` | — | +8 rubber bands; max 2 per room |
| `0x28` | `kGreaseRt` | 32 × 27 | **`length`**, `state`, `initial` (inverted) | standing → `kRewardIt`; spilled → `kSlideIt` on `(0, −2, length−5, 0) + (31, 27)` | — | grease can spilling right |
| `0x29` | `kGreaseLf` | 32 × 27 | **`length`**, `state`, `initial` (inverted) | standing → `kRewardIt`; spilled → `kSlideIt` on `(−length+5, −2, 0, 0) + (1, 27)` | — | grease can spilling left |
| `0x2A` | `kFoil` | 55 × 15 | `state`, `initial` | `kRewardIt` | — | +8 aluminium foil (absorbs `kDissolveIt`) |
| `0x2B` | `kInvisBonus` | 24 × 24 | **`points`** (100/300/500), `state`, `initial` | `kRewardIt` | — | invisible score pickup |
| `0x2C` | `kStar` | 32 × 31 | `state`, `initial` | `kRewardIt` | star | +5000; last one ends the game; max 4 per room |
| `0x2D` | `kSparkle` | 20 × 19 | `state` | **none** | Y | decorative twinkle; `timer = RandomInt(60)+15` |
| `0x2E` | `kHelium` | 56 × 16 | `state`, `initial` | `kRewardIt` | — | −150 battery (i.e. buoyancy) |
| `0x2F` | `kSlider` | 64 × 16 | **`length`** (width only; height fixed 16) | `kSlideIt` | — | invisible slippery strip |

## Transport — variant `d`, `transportType`

| ID | Constant | Default W×H | Fields used | Hot spot | Link gate | Semantics |
|---|---|---|---|---|---|---|
| `0x31` | `kUpStairs` | 160 × 267 | `topLeft` (v = 28) | `kMoveItUp` 112 × 32 at `topLeft` | none | staircase up; 1 per room |
| `0x32` | `kDownStairs` | 160 × 267 | `topLeft` (v = 28) | `kMoveItDown` 80 × 56 at `(right, 170)` | none | staircase down; 1 per room |
| `0x33` | `kMailboxLf` | 94 × 80 | `where`, `who` | `kMailItLeft` 72 × 40 at `topLeft + (30−72, 16)` | `who != 255` | left-facing mailbox teleport |
| `0x34` | `kMailboxRt` | 94 × 80 | `where`, `who` | `kMailItRight` 72 × 40 at `topLeft + (79, 16)` | `who != 255` | right-facing mailbox |
| `0x35` | `kFloorTrans` | 56 × 15 | `where`, `who` (v = 302) | `kDuctItDown` 76 × 48 at `topLeft + (−8, 15−48)` | `who != 255` | floor duct going down |
| `0x36` | `kCeilingTrans` | 56 × 15 | `where`, `who` (v = 6) | `kDuctItUp` 76 × 48 at `topLeft + (−8, 0)` | `who != 255` | ceiling duct going up |
| `0x37` | `kDoorInLf` | 144 × 322 | `topLeft` = (0, 0) | `kIgnoreLeftWall` 16 × 240 at `(0, 52)` | none | interior door, left wall |
| `0x38` | `kDoorInRt` | 144 × 322 | `topLeft` = (368, 0) | `kIgnoreRightWall` 16 × 240 at `(128, 52)` | none | interior door, right wall |
| `0x39` | `kDoorExRt` | 16 × 322 | `topLeft` = (496, 0) | `kIgnoreRightWall` 16 × 240 at `(0, 52)` | none | exterior door, right |
| `0x3A` | `kDoorExLf` | 16 × 322 | `topLeft` = (0, 0) | `kIgnoreLeftWall` 16 × 240 at `(0, 52)` | none | exterior door, left |
| `0x3B` | `kWindowInLf` | 20 × 170 | `topLeft` = (0, 64) | `kIgnoreLeftWall` 16 × 44 at `(0, 96)` | none | interior window, left |
| `0x3C` | `kWindowInRt` | 20 × 170 | `topLeft` = (492, 64) | `kIgnoreRightWall` 16 × 44 at `(4, 96)` | none | interior window, right |
| `0x3D` | `kWindowExRt` | 16 × 170 | `topLeft` = (496, 64) | `kIgnoreRightWall` 16 × 44 at `(0, 96)` | none | exterior window, right |
| `0x3E` | `kWindowExLf` | 16 × 170 | `topLeft` = (0, 64) | `kIgnoreLeftWall` 16 × 44 at `(0, 96)` | none | exterior window, left |
| `0x3F` | `kInvisTrans` | 64 × 32 | **`tall`** (height), **`wide`** (extra width), `where`, `who` | `kTransportIt` `(64 + wide)` × `tall` | `who != 255` | invisible teleport pad |
| `0x40` | `kDeluxeTrans` | 64 × 64 | **`tall`** = packed (w/4 << 8 \| h/4), **`wide`** = (initial << 4 \| state), `where`, `who` | `kTransportIt`, `isOn = wide & 0x0F` | `who != 255` | switchable invisible teleport, 4-px granularity |

## Switches — variant `e`, `switchType`

| ID | Constant | Default W×H | Fields used | Hot spot | Semantics |
|---|---|---|---|---|---|
| `0x41` | `kLightSwitch` | 15 × 24 | `where`, `who`, `type` | `kSwitchIt` if `where != -1` | wall light switch; art reflects the *target's* state |
| `0x42` | `kMachineSwitch` | 16 × 24 | `where`, `who`, `type` | `kSwitchIt` | industrial rocker switch |
| `0x43` | `kThermostat` | 15 × 24 | `where`, `who`, `type` | `kSwitchIt` | thermostat dial |
| `0x44` | `kPowerSwitch` | 8 × 8 | `where`, `who`, `type` | `kSwitchIt` | tiny digital button |
| `0x45` | `kKnifeSwitch` | 16 × 24 | `where`, `who`, `type` | `kSwitchIt` | knife switch (**missing from `Set`/`GetObjectState`**) |
| `0x46` | `kInvisSwitch` | 12 × 12 | `where`, `who`, `type` | `kSwitchIt` | invisible switch |
| `0x47` | `kTrigger` | 12 × 12 | **`delay`**, `where`, `who`, `type` = 3 | `kTriggerIt` if `where != -1` | invisible timed trigger |
| `0x48` | `kLgTrigger` | 48 × 48 | **`delay`**, `where`, `who`, `type` = 3 | `kTriggerIt` | large version of the above |
| `0x49` | `kSoundTrigger` | 32 × 32 | **`where` = sound resource ID** | `kSoundIt` on a **48 × 48** rect, gated on `LoadTriggerSound(where) == noErr` | plays a `snd ` resource; 1 per room |

`type` values: `kToggle` 0, `kForceOn` 1, `kForceOff` 2, `kOneShot` 3
(`GliderPRO/Headers/GliderDefines.h:441-444`). The editor's radio group offers only the first three
for real switches (`GliderPRO/Sources/ObjectInfo.c:1298-1339`); triggers are hard-wired to
`kOneShot`.

## Lights — variant `f`, `lightType`

| ID | Constant | Default W×H | Fields used | Hot spot | Semantics |
|---|---|---|---|---|---|
| `0x51` | `kCeilingLight` | 64 × 20 | `initial`, `state`; `length` written (64) but unused | none | hanging ceiling fixture; v = 4 |
| `0x52` | `kLightBulb` | 16 × 28 | `initial`, `state` | none | bare bulb |
| `0x53` | `kTableLamp` | 48 × 70 | `initial`, `state` | none | table lamp |
| `0x54` | `kHipLamp` | 72 × 276 | `initial`, `state` | none | tall floor lamp; v = 23 |
| `0x55` | `kDecoLamp` | 64 × 212 | `initial`, `state` | none | art-deco lamp; v = 91 |
| `0x56` | `kFlourescent` | 64 × 12 | **`length` = rect width** (right edge = `length + h`), `initial`, `state` | none | stretchable fluorescent tube; v = 12 |
| `0x57` | `kTrackLight` | 64 × 24 | **`length` = rect width** (right edge = `length + h`), `initial`, `state` | none | stretchable track lighting, 3 lamp sprites; v = 5 |
| `0x58` | `kInvisLight` | 16 × 16 | `initial`, `state` | none | invisible light source; also a legal transport destination |

A room is lit iff `numLights > 0` where `numLights` counts lights with `state` true. Unlit rooms
draw almost nothing (see §10). `kWallWindow` is the one clutter type that draws regardless.

## Appliances — variant `g`, `applianceType`

| ID | Constant | Default W×H | Fields used | Hot spot | Anim | Semantics |
|---|---|---|---|---|---|---|
| `0x61` | `kShredder` | 73 × 22 | `initial`, `state` | `kShredIt`, `(73+48) × 40` offset `(−24, −36)`, `isOn = state` | — | paper shredder; max 4 per room |
| `0x62` | `kToaster` | 48 × 27 | **`height`** = launch height, **`delay`**, `initial`, `state` | `kDissolveIt` | Y | launches toast upward ballistically |
| `0x63` | `kMacPlus` | 48 × 58 | `initial`, `state` | `kDissolveIt` | Y | Mac Plus with a flickering screen |
| `0x64` | `kGuitar` | 64 × 172 | — | `kStrumIt` 8 × 96 at `topLeft + (34, 32)` | — | strummable guitar |
| `0x65` | `kTV` | 92 × 77 | `initial`, `state` | `kDissolveIt` | Y | television with animated screen |
| `0x66` | `kCoffee` | 43 × 64 | `initial`, `state` | `kDissolveIt` | Y | coffee machine; `timer = isOn ? 200 : 0` |
| `0x67` | `kOutlet` | 16 × 24 | **`delay`**, `initial`, `state` | `kIgnoreIt`, `isOn = state` | Y | electrical outlet that arcs; `hVel = numLights` |
| `0x68` | `kVCR` | 96 × 22 | `initial`, `state` | `kDissolveIt` | Y | VCR with a blinking clock; `timer = isOn ? 115 : 0` |
| `0x69` | `kStereo` | 128 × 53 | `initial`, `state`; **live state is the global `isPlayMusicGame`** | `kDissolveIt` | Y | stereo; plays the house music |
| `0x6A` | `kMicrowave` | 92 × 59 | **`byte0`** kill-mask, `initial`, `state` | `kDissolveIt` on the body **+ `kMicrowaveIt`** on the strip from y = 0 to the body top | Y | destroys carried items when you fly over it |
| `0x6B` | `kCinderBlock` | 40 × 62 | — | `kDissolveIt` | — | cinder block |
| `0x6C` | `kFlowerBox` | 80 × 32 | — | `kDissolveIt` | — | window flower box |
| `0x6D` | `kCDs` | 16 × 30 | — | `kDissolveIt` | — | stack of CDs |
| `0x6E` | `kCustomPict` | 72 × 34 (fallback) | **`height` = PICT resource ID** (10000..32767) | **none** | — | arbitrary decorative picture; 4782 instances in shipped houses |

Delay-field visibility in the editor: hidden for `kShredder`, `kMacPlus`, `kTV`, `kCoffee`, `kVCR`,
`kMicrowave` (`GliderPRO/Sources/ObjectInfo.c:1658-1663`) — only `kToaster` and `kOutlet` use it.

## Enemies — variant `h`, `enemyType`

| ID | Constant | Default W×H | Fields used | Hot spot | Anim | Semantics |
|---|---|---|---|---|---|---|
| `0x71` | `kBalloon` | 24 × 30 | **`delay`**, `initial`, `state` | `kIgnoreIt` | Y | rises from y = 310 at `vVel = −2`; lethal on contact via dynamics |
| `0x72` | `kCopterLf` | 32 × 30 | **`delay`**, `initial`, `state` | `kIgnoreIt` | Y | toy helicopter descending from y = 8, drifting left (`hVel = −1`) |
| `0x73` | `kCopterRt` | 32 × 30 | **`delay`**, `initial`, `state` | `kIgnoreIt` | Y | as above, drifting right (`hVel = +1`) |
| `0x74` | `kDartLf` | 64 × 19 | **`delay`**, `initial`, `state` | `kIgnoreIt` | Y | dart flying left at `hVel = −6`, `vVel = 2`; spawns at x = 448 |
| `0x75` | `kDartRt` | 64 × 19 | **`delay`**, `initial`, `state` | `kIgnoreIt` | Y | dart flying right at `hVel = +6`; spawns at x = 0 |
| `0x76` | `kBall` | 32 × 32 | **`length`** = bounce height, `initial`, `state` | `kIgnoreIt` | Y | bouncing ball; velocity solved from `length` on alternate frames |
| `0x77` | `kDrip` | 16 × 12 | **`length`** = fall distance, **`delay`**, `initial`, `state` | `kIgnoreIt` | Y | water drip; `frame = 3`, remembers origin in `hVel` |
| `0x78` | `kFish` | 36 × 33 | **`length`** = jump height, **`delay`**, `initial`, `state` | `kDissolveIt` (scrutinized) | Y | fish bowl; fish jumps out |
| `0x79` | `kCobweb` | 54 × 45 | — | **`kWebIt`** on `InsetRect(rect, −24, −10)` → 102 × 65 | — | static cobweb; sticks the glider. Hot spot is 48 px wider and 20 px taller than the sprite |

## Clutter — variant `i`, `clutterType`

| ID | Constant | Default W×H | Fields used | Hot spot | Semantics |
|---|---|---|---|---|---|
| `0x81` | `kOzma` | 102 × 92 | `bounds` | none | Ozma poster; legal transport destination |
| `0x82` | `kMirror` | 64 × 64 | `bounds` | none | mirror; reflects the glider (`AddToMirrorRegion` on `InsetRect(4,4)`); transport destination |
| `0x83` | `kMousehole` | 10 × 11 | `bounds` (`bottom` = 295) | none | mouse hole in the baseboard |
| `0x84` | `kFireplace` | 180 × 142 | `bounds` (`bottom` = 297) | none | fireplace; transport destination |
| `0x85` | `kFlower` | from `flowerSrc[pict]` | `bounds`, **`pict`** 0..5 | none | one of six potted plants; **no `srcRects` entry** |
| `0x86` | `kWallWindow` | 64 × 80 | `bounds` | none | closed window; drawn even in an unlit room; transport destination |
| `0x87` | `kBear` | 56 × 58 | `bounds` | none | teddy bear |
| `0x88` | `kCalendar` | 63 × 92 | `bounds` | none | wall calendar; transport destination |
| `0x89` | `kVase1` | 36 × 45 | `bounds` | none | broad vase |
| `0x8A` | `kVase2` | 35 × 57 | `bounds` | none | narrow vase |
| `0x8B` | `kBulletin` | 80 × 58 | `bounds` | none | bulletin board; transport destination |
| `0x8C` | `kCloud` | 128 × 30 | `bounds` | none | cloud (masked PICT); transport destination; 1860 instances |
| `0x8D` | `kFaucet` | 56 × 18 | `bounds` | none | dripping faucet art |
| `0x8E` | `kRug` | 144 × 18 | `bounds` | none | throw rug |
| `0x8F` | `kChimes` | 28 × 74 | `bounds` | **`kChimeIt`** on `srcRects[kChimes]` (28 × 74) at `bounds.left/top`; also `numChimes++` | wind chimes — the only interactive clutter |

---

# 13. Observed field ranges across all 22 shipped houses

Generated by `/tmp/wf-objtax/mkranges.py`, which walks every 12-byte object record in all 22
BinHex-decoded house data forks. Totals: **4070 rooms, 31 440 live objects, 66 240 empty slots,
117 distinct `what` values, zero unrecognised `what` values.** A single value means the field is
constant across every instance in every shipped house; that is strong (though not conclusive)
evidence the field is either forced by the editor or unused.

`h`/`v` are `topLeft.h`/`topLeft.v`; `left`/`top`/`right`/`bottom` are the `bounds` components.
Remember the on-disk order is `v` then `h`, and `top left bottom right`.

### Blowers (variant `a`)

| ID | Constant | n | h | v | distance | initial | state | vector | tall |
|---|---|---|---|---|---|---|---|---|---|
| `0x01` | `kFloorVent` | 1458 | 0..464 | 305 | 0..269 | 0..1 | 0..1 | 1 | 0..61 |
| `0x02` | `kCeilingVent` | 28 | 53..450 | 8 | 82..303 | 0..1 | 0..1 | 4 | 0..1 |
| `0x03` | `kFloorBlower` | 83 | 13..457 | 304 | 49..268 | 0..1 | 0..1 | 1 | 0..3 |
| `0x04` | `kCeilingBlower` | 12 | 20..418 | 5 | 61..225 | 1 | 1 | 4 | 0..1 |
| `0x05` | `kSewerGrate` | 507 | 0..464 | 303 | 22..303 | 0..1 | 0..1 | 1 | 0..61 |
| `0x06` | `kLeftFan` | 45 | 89..466 | 3..241 | 3..300 | 0..1 | 0..1 | 8 | 0..41 |
| `0x07` | `kRightFan` | 54 | 18..439 | 15..244 | 0..368 | 0..1 | 0..1 | 2..8 | 0..85 |
| `0x08` | `kTaper` | 90 | 8..480 | 61..262 | 0..206 | 1 | 1 | 1 | 0..5 |
| `0x09` | `kCandle` | 180 | 10..472 | 55..292 | 0..240 | 1 | 1 | 1 | 0..5 |
| `0x0A` | `kStubby` | 127 | 3..491 | 43..286 | 0..237 | 1 | 1 | 1 | 0..5 |
| `0x0B` | `kTiki` | 58 | 46..474 | 20..262 | 0..213 | 1 | 1 | 1 | 0..4 |
| `0x0C` | `kBBQ` | 43 | 2..446 | 83..289 | 32..276 | 1 | 1 | 1 | 0..5 |
| `0x0D` | `kInvisBlower` | 2336 | 0..488 | 0..298 | 0..488 | 0..23 | 0..23 | 1..20 | 0..47 |
| `0x0E` | `kGrecoVent` | 116 | 8..457 | 303 | 64..303 | 0..1 | 0..1 | 1 | 0..4 |
| `0x0F` | `kSewerBlower` | 191 | 1..476 | 292 | 26..292 | 0..1 | 0..1 | 1 | 0..41 |
| `0x10` | `kLiftArea` | 716 | 0..506 | 0..302 | 1..512 | 0..1 | 0..1 | 1..8 | 1..161 |

### Furniture (variant `b`)

| ID | Constant | n | left | top | right | bottom | pict |
|---|---|---|---|---|---|---|---|
| `0x11` | `kTable` | 170 | 5..410 | 58..269 | 69..495 | 66..277 | 0 |
| `0x12` | `kShelf` | 389 | 0..428 | 38..315 | 57..512 | 44..321 | 0 |
| `0x13` | `kCabinet` | 457 | 0..479 | 0..313 | 32..512 | 22..322 | 0 |
| `0x14` | `kFilingCabinet` | 107 | 0..425 | 33..212 | 74..499 | 140..319 | 0 |
| `0x15` | `kWasteBasket` | 103 | 13..446 | 29..259 | 77..510 | 90..320 | 0 |
| `0x16` | `kMilkCrate` | 252 | 3..444 | 6..264 | 67..508 | 64..322 | 0 |
| `0x17` | `kCounter` | 287 | 0..441 | 7..276 | 54..512 | 304 | 0 |
| `0x18` | `kDresser` | 122 | 13..409 | 17..272 | 80..500 | 293 | 0 |
| `0x19` | `kDeckTable` | 30 | 0..403 | 125..276 | 127..495 | 133..284 | 0 |
| `0x1A` | `kStool` | 91 | 21..464 | 18..263 | 69..512 | 56..301 | 0 |
| `0x1B` | `kTrunk` | 101 | 0..353 | 40..242 | 144..497 | 120..322 | 0 |
| `0x1C` | `kInvisObstacle` | 666 | 0..487 | 0..321 | 21..512 | 27..322 | 0 |
| `0x1D` | `kManhole` | 40 | 67..323 | 300 | 190..446 | 322 | 0 |
| `0x1E` | `kBooks` | 210 | 1..448 | 7..271 | 65..512 | 58..322 | 0 |
| `0x1F` | `kInvisBounce` | 1670 | 0..510 | 0..311 | 5..512 | 4..322 | 0 |

### Prizes / bonuses (variant `c`)

| ID | Constant | n | h | v | length | points | state | initial |
|---|---|---|---|---|---|---|---|---|
| `0x21` | `kRedClock` | 140 | 2..464 | 5..285 | 0 | 0 | 0..1 | 1 |
| `0x22` | `kBlueClock` | 163 | 32..474 | 34..277 | 0 | 0 | 0..1 | 1 |
| `0x23` | `kYellowClock` | 330 | 2..472 | 10..280 | 0 | 0 | 0..1 | 1 |
| `0x24` | `kCuckoo` | 100 | 24..446 | 0..242 | 0 | 0 | 0..1 | 1 |
| `0x25` | `kPaper` | 283 | 0..464 | 0..293 | 0 | 0 | 0..1 | 1 |
| `0x26` | `kBattery` | 119 | 32..474 | 28..283 | 0 | 0 | 0..1 | 1 |
| `0x27` | `kBands` | 150 | 36..468 | 24..278 | 0 | 0 | 0..1 | 1 |
| `0x28` | `kGreaseRt` | 195 | 0..412 | 28..293 | 0..474 | 0 | 0..1 | 0..1 |
| `0x29` | `kGreaseLf` | 143 | 82..480 | 17..289 | 0..472 | 0 | 0..1 | 0..1 |
| `0x2A` | `kFoil` | 83 | 48..432 | 43..300 | 0 | 0 | 0..1 | 1 |
| `0x2B` | `kInvisBonus` | 327 | 0..488 | 17..298 | 0 | 100..500 | 0..1 | 1 |
| `0x2C` | `kStar` | 69 | 34..452 | 0..277 | 0..4 | 0..145 | 0..1 | 1 |
| `0x2D` | `kSparkle` | 486 | 0..492 | 0..303 | 0..10 | 0 | 1 | 1 |
| `0x2E` | `kHelium` | 50 | 36..366 | 69..300 | 0 | 0 | 0..1 | 1 |
| `0x2F` | `kSlider` | 354 | 0..466 | 0..306 | 18..512 | 0 | 1 | 1 |

### Transport (variant `d`)

| ID | Constant | n | h | v | tall | where | who | wide |
|---|---|---|---|---|---|---|---|---|
| `0x31` | `kUpStairs` | 163 | 16..350 | 28 | 0 | -1 | 255 | 0 |
| `0x32` | `kDownStairs` | 163 | 13..338 | 28 | 0 | -1 | 255 | 0 |
| `0x33` | `kMailboxLf` | 44 | 76..418 | 6..204 | 0 | -1..8909 | 0..255 | 0 |
| `0x34` | `kMailboxRt` | 34 | 0..350 | 16..202 | 0 | -100..9109 | 0..255 | 0 |
| `0x35` | `kFloorTrans` | 244 | 6..451 | 300..302 | 0 | -100..9014 | 0..255 | 0 |
| `0x36` | `kCeilingTrans` | 465 | 5..456 | 6 | 0 | -100..9209 | 0..255 | 0 |
| `0x37` | `kDoorInLf` | 23 | 0 | 0 | 0 | -1 | 255 | 0 |
| `0x38` | `kDoorInRt` | 11 | 368 | 0 | 0 | -1 | 255 | 0 |
| `0x39` | `kDoorExRt` | 21 | 496 | 0 | 0 | -1 | 255 | 0 |
| `0x3A` | `kDoorExLf` | 11 | 0 | 0 | 0 | -1 | 255 | 0 |
| `0x3B` | `kWindowInLf` | 21 | 0 | 64 | 0 | -1 | 255 | 0 |
| `0x3C` | `kWindowInRt` | 32 | 492 | 64 | 0 | -1 | 255 | 0 |
| `0x3D` | `kWindowExRt` | 19 | 496 | 64 | 0 | -1 | 255 | 0 |
| `0x3E` | `kWindowExLf` | 29 | 0 | 64 | 0 | -1 | 255 | 0 |
| `0x3F` | `kInvisTrans` | 385 | 0..448 | 0..300 | 21..322 | -100..12523 | 0..255 | 0..127 |
| `0x40` | `kDeluxeTrans` | 61 | 0..430 | 0..286 | -32722..32591 | -1..11528 | 0..255 | 0..17 |

### Switches (variant `e`)

| ID | Constant | n | h | v | delay | where | who | type |
|---|---|---|---|---|---|---|---|---|
| `0x41` | `kLightSwitch` | 113 | 26..484 | 41..243 | 0 | -100..12332 | 0..255 | 0..2 |
| `0x42` | `kMachineSwitch` | 79 | 0..472 | 29..250 | 0 | -1..12328 | 0..255 | 0..2 |
| `0x43` | `kThermostat` | 108 | 10..496 | 31..263 | 0 | -100..12033 | 0..255 | 0..2 |
| `0x44` | `kPowerSwitch` | 78 | 54..490 | 18..247 | 0 | -1..12241 | 1..255 | 0..2 |
| `0x45` | `kKnifeSwitch` | 230 | 18..482 | 15..278 | 0 | -100..12330 | 0..255 | 0..2 |
| `0x46` | `kInvisSwitch` | 635 | 0..500 | 0..310 | 0 | -1..12531 | 0..255 | 0..2 |
| `0x47` | `kTrigger` | 239 | 12..474 | 0..290 | 0..120 | 505..12325 | 0..23 | 3 |
| `0x48` | `kLgTrigger` | 81 | 0..464 | 0..274 | 0..120 | 810..8009 | 0..23 | 3 |
| `0x49` | `kSoundTrigger` | 122 | 20..478 | 5..290 | 0 | 3000..10000 | 255 | 0 |

### Lights (variant `f`)

| ID | Constant | n | h | v | length | byte0 | byte1 | initial | state |
|---|---|---|---|---|---|---|---|---|---|
| `0x51` | `kCeilingLight` | 160 | 20..418 | 4 | 64 | 0 | 0 | 0..1 | 0..1 |
| `0x52` | `kLightBulb` | 252 | 3..473 | 0..294 | 0 | 0 | 0 | 0..1 | 0..1 |
| `0x53` | `kTableLamp` | 70 | 15..454 | 29..245 | 0 | 0 | 0 | 0..1 | 0..1 |
| `0x54` | `kHipLamp` | 26 | 5..436 | 23 | 0 | 0 | 0 | 0..1 | 0..1 |
| `0x55` | `kDecoLamp` | 62 | 6..438 | 91 | 0 | 0 | 0 | 0..1 | 0..1 |
| `0x56` | `kFlourescent` | 115 | 4..395 | 12 | 24..487 | 0 | 0 | 0..1 | 0..1 |
| `0x57` | `kTrackLight` | 81 | 0..309 | 5 | 64..512 | 0 | 0 | 0..1 | 0..1 |
| `0x58` | `kInvisLight` | 1764 | 0..496 | 0..306 | 0 | 0 | 0 | 0..1 | 0..1 |

### Appliances (variant `g`)

| ID | Constant | n | h | v | height | byte0 | delay | initial | state |
|---|---|---|---|---|---|---|---|---|---|
| `0x61` | `kShredder` | 50 | 32..425 | 43..300 | 0 | 0 | 0 | 0..1 | 0..1 |
| `0x62` | `kToaster` | 140 | 14..434 | 67..294 | 9..277 | 0 | 0..255 | 0..1 | 0..1 |
| `0x63` | `kMacPlus` | 74 | 18..460 | 33..253 | 0 | 0 | 0 | 0..1 | 0..1 |
| `0x64` | `kGuitar` | 29 | 14..448 | 19..141 | 0 | 0 | 0 | 1 | 1 |
| `0x65` | `kTV` | 80 | -1..419 | 0..245 | 0 | 0 | 0 | 0..1 | 0..1 |
| `0x66` | `kCoffee` | 38 | 22..462 | 29..258 | 0 | 0 | 0 | 0..1 | 0..1 |
| `0x67` | `kOutlet` | 100 | 18..464 | 8..265 | 0 | 0 | 0..240 | 0..1 | 0..1 |
| `0x68` | `kVCR` | 26 | 20..398 | 43..297 | 0 | 0 | 0 | 0..1 | 0..1 |
| `0x69` | `kStereo` | 36 | 0..382 | 34..266 | 0 | 0 | 0 | 0..1 | 0..1 |
| `0x6A` | `kMicrowave` | 56 | 14..380 | 31..263 | 0 | 0..7 | 0 | 0..1 | 0..1 |
| `0x6B` | `kCinderBlock` | 43 | 20..453 | 73..256 | 0 | 0 | 0 | 1 | 1 |
| `0x6C` | `kFlowerBox` | 35 | 12..418 | 7..288 | 0 | 0 | 0 | 1 | 1 |
| `0x6D` | `kCDs` | 63 | 29..475 | 9..290 | 0 | 0 | 0 | 1 | 1 |
| `0x6E` | `kCustomPict` | 4782 | 0..509 | 0..319 | 10000..12351 | 0 | 0 | 1 | 1 |

### Enemies (variant `h`)

| ID | Constant | n | h | v | length | delay | byte0 | initial | state |
|---|---|---|---|---|---|---|---|---|---|
| `0x71` | `kBalloon` | 500 | 34..474 | 146 | 0 | 0..255 | 0 | 0..1 | 0..1 |
| `0x72` | `kCopterLf` | 259 | 72..480 | 146 | 0 | 0..255 | 0 | 0..1 | 0..1 |
| `0x73` | `kCopterRt` | 155 | 18..418 | 146 | 0 | 0..255 | 0 | 0..1 | 0..1 |
| `0x74` | `kDartLf` | 151 | 436..448 | 0..270 | 0 | 0..53 | 0 | 0..1 | 1 |
| `0x75` | `kDartRt` | 117 | 0..2 | 5..280 | 0 | 2..255 | 0 | 0..1 | 0..1 |
| `0x76` | `kBall` | 212 | 30..464 | 31..289 | 0..281 | 0 | 0 | 0..1 | 0..1 |
| `0x77` | `kDrip` | 477 | 24..470 | 0..249 | 1..310 | 0..255 | 0 | 0..1 | 0..1 |
| `0x78` | `kFish` | 120 | 28..466 | 25..289 | 0..261 | 0..255 | 0 | 0..1 | 0..1 |
| `0x79` | `kCobweb` | 86 | 8..458 | 0..249 | 0 | 0 | 0 | 1 | 1 |

### Clutter (variant `i`)

| ID | Constant | n | left | top | right | bottom | pict |
|---|---|---|---|---|---|---|---|
| `0x81` | `kOzma` | 77 | 12..397 | 26..163 | 114..499 | 118..255 | 0 |
| `0x82` | `kMirror` | 667 | 0..502 | 0..299 | 12..512 | 16..322 | 0 |
| `0x83` | `kMousehole` | 174 | 10..498 | 284 | 20..508 | 295 | 0 |
| `0x84` | `kFireplace` | 33 | 35..286 | 155 | 215..466 | 297 | 0 |
| `0x85` | `kFlower` | 547 | 0..487 | 3..308 | 32..504 | 53..322 | 0..5 |
| `0x86` | `kWallWindow` | 195 | 8..426 | 0..292 | 90..507 | 30..322 | 0 |
| `0x87` | `kBear` | 169 | 5..451 | 28..264 | 61..507 | 86..322 | 0 |
| `0x88` | `kCalendar` | 58 | 2..423 | 5..158 | 65..486 | 97..250 | 0 |
| `0x89` | `kVase1` | 72 | 22..465 | 51..275 | 58..501 | 96..320 | 0 |
| `0x8A` | `kVase2` | 86 | 8..465 | 2..255 | 43..500 | 59..312 | 0 |
| `0x8B` | `kBulletin` | 38 | 45..409 | 21..165 | 125..489 | 79..223 | 0 |
| `0x8C` | `kCloud` | 1860 | 0..384 | 0..292 | 128..512 | 30..322 | 0 |
| `0x8D` | `kFaucet` | 28 | 35..393 | 52..237 | 91..449 | 70..255 | 0 |
| `0x8E` | `kRug` | 81 | 0..353 | 77..304 | 144..497 | 95..322 | 0 |
| `0x8F` | `kChimes` | 54 | 0..484 | 0..248 | 28..512 | 74..322 | 0 |

## 13.1 What the ranges prove

1. **Forced-placement constants are real.** `kFloorVent` v is 305 in all 1458 instances,
   `kCeilingVent` 8 in all 28, `kFloorBlower` 304, `kCeilingBlower` 5, `kSewerGrate` 303,
   `kGrecoVent` 303, `kSewerBlower` 292, `kCeilingTrans` 6, `kUpStairs`/`kDownStairs` 28,
   `kCeilingLight` 4, `kHipLamp` 23, `kDecoLamp` 91, `kFlourescent` 12, `kTrackLight` 5,
   `kBalloon`/`kCopterLf`/`kCopterRt` 146, `kCounter.bottom` 304, `kDresser.bottom` 293,
   `kManhole` top 300 / bottom 322, `kMousehole` 284/295, `kFireplace` 155/297. Every door and
   window is pinned to exactly the `kDoor*`/`kWindow*` constants of
   `GliderPRO/Headers/GliderDefines.h:483-494` (`kDoorInTop` through `kWindowExRtLeft`; `:481` is
   `kTrackLightTop`). Confirmed exhaustively: across all 22 houses each of the eight door/window
   types occurs at exactly one `(h, v)` pair, matching those constants.
2. **`kFloorTrans` v is 300..302, not the constant 302.** `kFloorTransTop` is 302
   (`GliderPRO/Headers/GliderDefines.h:473`) but 300 and 301 also occur — evidence of hand editing
   or of the v1 house converter, since `AddNewObject` can only produce 302.
3. **Unused fields really are unused, and really do contain junk.** `blowerType.tall` is read only
   by `kLiftArea`, and every other blower type carries garbage there (up to 85 on a `kRightFan`,
   61 on a `kFloorVent`). `bonusType.length` is read only by grease and `kSlider`, and carries
   1..10 on `kSparkle` and 0..4 on `kStar`. `bonusType.points` is read only by `kInvisBonus` yet
   `kStar` records hold 0..145. `kRightFan.vector` is 2 in most records but 8 in some — harmless,
   because `CreateActiveRects` hard-codes `kPushItRight` for that type
   (`GliderPRO/Sources/ObjectRects.c:384-397`).
4. **`Boolean` is not restricted to 0/1.** `kInvisBlower.initial` and `.state` range 0..23. Any
   port must test `!= 0`, never `== 1`.
5. **`kDeluxeTrans.tall` is a packed pair and looks like noise as a signed short**: −32722..32591.
   Decoding it as `(w = (tall & 0xFF00) >> 8, h = tall & 0x00FF)` and multiplying by 4 is the only
   interpretation that produces sane rectangles.
6. **`lightType.byte0`/`byte1` and `enemyType.byte0` are 0 in all 31 440 objects.** They are
   never written by `AddNewObject` and never read anywhere in the source. They are free bytes.
7. **`furnitureType.pict` and `clutterType.pict` are 0 everywhere except `kFlower`,** where they
   are 0..5 and select one of the six `flowerSrc` sprites.
8. **`kCustomPict.height` (the PICT resource ID) ranges 10000..12351.** The editor enforces
   10000..32767 (`GliderPRO/Sources/ObjectInfo.c:1185-1255`); the shipped houses use at most 2352
   distinct custom pictures.
9. **`kToaster.height` (launch height) ranges 9..277** and `kToaster.delay` uses the full
   0..255 byte. `kOutlet.delay` reaches 240. No other appliance uses `delay`.
10. **`who` is a 0..23 object index or the sentinel 255**, with exactly one exception: a single
    record with `who == 35`, which is out of range for `kMaxRoomObs` = 24 and would index past
    the end of a room's object array.
11. **`kSoundTrigger.where` is a resource ID (3000..10000), not a room code**, and its `who` is
    always 255. A port must special-case this before treating `where` as a link.
12. **Trigger delays are 0..120** for both `kTrigger` and `kLgTrigger`; the editor's cap is 32767.
13. **`kTV` has one instance with `topLeft.h == -1`** — a legal but off-by-one placement that
    proves negative coordinates survive round-tripping.

---

# 14. Latent bugs and traps in the original

Faithful re-implementation means reproducing the *behaviour*, which sometimes means reproducing a
bug and sometimes means deliberately not reproducing undefined behaviour. Each item below was
verified by reading the cited lines.

| # | Location | Problem | Observable? |
|---|---|---|---|
| 1 | `GliderPRO/Sources/Objects.c:961-979` | `BringSendFrontBack` has the two spellings **swapped**: the eight switch/trigger cases (`:963-970`) write `data.d.`**`who`** — the transport field — at `:971-972`, and the `default:` case that catches the transports (mailboxes, ducts, stairs, doors, windows) writes `data.e.`**`who`** — the switch field — at `:976-977`. Harmless: both `who` fields are at union offset 8. | No |
| 2 | `GliderPRO/Sources/ObjectRects.c:68-71` | Four unreachable **statements** (`*itsRect = srcRects[who->what]; ZeroRectCorner; QOffsetRect; break;`) sitting between `case kLiftArea:`'s `break;` (`:66`) and the next `case kTable:` (`:73`) — they carry **no `case` label**, so nothing can reach them. Dead code left from an earlier layout. | No |
| 3 | `GliderPRO/Sources/Objects.c:366-698`, `:703-871` | **`kKnifeSwitch` appears in neither `SetObjectState`'s nor `GetObjectState`'s switch** (the switch arms are `:533-542` and `:793-801`, both listing the other five switches plus the three triggers). `SetObjectState` declares `Boolean changed;` uninitialised and `return (changed);` at `:698`, so a knife switch as a *state target* returns garbage. **`kSlider` has the same defect for a different reason**: its arm is a bare `break;` (`:475-476`), so `changed` is never assigned there either. Unreachable in practice: `UpdateLinkControl` never offers a switch as a link destination. | No (in shipped houses) |
| 4 | `GliderPRO/Sources/Interactions.c:1050` | In `HandleSwitches`, `Rect newRect, bounds;` is declared at `:987` and `bounds` is **never assigned**, yet `AddSparkle(&bounds)` is called at `:1050` when a switch turns off a linked prize. Reads an uninitialised stack `Rect`. (`HandleRewards` does the same call correctly — it assigns `bounds = who->bounds` at `:762`.) | Yes: sparkle at a garbage position when a switch removes a prize |
| 5 | `GliderPRO/Sources/Interactions.c:1352` | `case kLgTrigger:` inside a switch on *action codes*. `kLgTrigger` is 0x48 = 72; `CreateActiveRects` only ever emits action codes 0..27, so the arm is dead. | No |
| 6 | `GliderPRO/Sources/Triggers.c:50` | `triggers[where].what = masterObjects[triggers[where].object].theObject.what;` indexes the master list by `objectLink`, the linked object's *slot number within its own room* (0..23), instead of by `localLink`, its master index. Because the master list has stride 24 with the central room first (§1.8), this resolves to the **central room's** slot `objectLink`. `triggers[].what` is therefore the wrong object's type whenever the trigger's target lives in a neighbour room. | Yes: `FireTrigger` dispatches on the wrong type |
| 7 | `GliderPRO/Sources/Triggers.c:174-189` | `FireTrigger`'s `else` branch is taken exactly when `masterObjects[triggerIs].localLink == -1`, yet it opens with `triggeredIs = masterObjects[triggerIs].localLink;` (`:178`) — so `triggeredIs` is **−1** — and then dereferences `masterObjects[triggeredIs].dynaNum` / `.hotNum` at `:187-188`. Reads 14 bytes before the start of the array. | Yes: out-of-bounds read whenever a trigger's target is outside the 9-room window |
| 7b | `GliderPRO/Sources/Triggers.c:131-133` | The `kSoundTrigger` arm of `FireTrigger` plays `kChordSound` with the source comment `// Change me` instead of loading the sound named by `data.e.where`. Fly-over and trigger-fired sound triggers make different noises. | Yes |
| 8 | `GliderPRO/Sources/Dynamics3.c:437-466` | `dinahs[].type` is assigned generically at `:196` and **every** other `AddDynamicObject` arm assigns `dinahs[numDynamics].room = room` (14 sites: `:210, 237, 257, 277, 300, 320, 343, 363, 384, 405, 430, 490, 509, 539`) — the `kDartLf`/`kDartRt` arm is the only one that does not. The slot keeps whatever `room` the previous occupant had (or 0, from `ZeroDinahs` at `:175`). | Yes: darts culled/kept in the wrong room |
| 9 | `GliderPRO/Sources/ObjectDrawAll.c:518, 531, 544, 557, 570, 574` (the six switch arms) | Each reads `dynamicNum = masterObjects[i].hotNum;` where the loop variable `i` runs 0..23 over *room slots*, so by the stride-24 layout it always names the **central room's** master entry. For the 8 neighbour rooms it therefore picks up an unrelated object's `hotNum`. Note the `hotNum → dynamicNum → masterObjects[].dynaNum` copy is **deliberate**, not a typo: switches have no dynamic object, and `FireTrigger` calls `TriggerSwitch(masterObjects[triggeredIs].dynaNum)` (`Triggers.c:128`), where `TriggerSwitch(who)` is `HandleSwitches(&hotSpots[who])` (`Trip.c:146-149`). `dynaNum` is overloaded to carry the hot-spot index for switch objects; only the `[i]` indexing is wrong. | Yes, for non-central rooms |
| 10 | `GliderPRO/Sources/ObjectRects.c:1051-1059` | The `kChimes` hot spot is built from `srcRects[kChimes]` (fixed 28 × 74) placed at `bounds.left/top`, while `GetObjectRect` returns the *resizable* `data.i.bounds`. A stretched chime's interactive area does not match its art. | Yes |
| 11 | `GliderPRO/Sources/ObjectRects.c:923-928` vs `:161-175` | `kSoundTrigger`'s hot spot is 48 × 48 but its `GetObjectRect` (and hence editor selection rect) is 32 × 32. | Yes |
| 12 | `GliderPRO/Sources/ObjectRects.c:220-237` | `GetObjectRect` for `kCustomPict` **mutates the object** — it writes `data.g.height = 10000` when `GetPicture` returns nil. A read-only query has a side effect, and it dirties the house. | Yes |
| 13 | `GliderPRO/Sources/ObjectInfo.c:2098-2145` | `kDeluxeTrans`'s info dialog writes back `wasState << 4`, clobbering the low nibble — editing a deluxe transport's *initial* state silently resets its *current* state to off. | Yes |
| 14 | `GliderPRO/Sources/StructuresInit2.c:271` | `srcRects` is allocated with `NewPtr` (uninitialised), and the 27 unused ID indices — plus `kFlower` (`0x85`) — are never filled in by `InitSrcRects`, leaving only 116 of 144 slots valid. Reading `srcRects[0x20]` yields heap garbage. Nothing does, but a port that pre-fills the table with zeroes is *safer*, not equivalent. | No |
| 15 | `GliderPRO/Headers/GliderDefines.h:501` | `kVertLocalOffset` is 322 with the comment `kTileHigh - 39 (was 283, then 295)` — the comment is stale and describes neither the value nor the arithmetic (322 *equals* `kTileHigh`). Trust the number. | No |
| 16 | `GliderPRO/Headers/GliderDefines.h:542` | `kRedOrangeColor8` is `23` with the comment `actually, 18`. The value 23 is what ships. | No |

---

## Open questions

1. **`Sampler.binhex` is 2 bytes longer than `866 + nRooms * 348`** (1564 vs 1562 for 2 rooms). All
   21 other houses match exactly. Is the extra pair trailing slop written by an older `WriteHouse`,
   or a 2-byte field appended after `rooms[]` that the released `houseType` no longer declares?
   `WriteHouse` (`GliderPRO/Sources/HouseIO.c:448-519`) writes `GetHandleSize(thisHouse)` bytes, so
   the handle itself must have been 2 bytes long — the excess is invisible to the loader and can be
   ignored, but its origin is unexplained.
2. **The 165 `where == -100` records cannot have been produced by the *shipped* 1.0.4 converter.**
   `ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746-820`) guards its transform with
   `if (where != -1)`, so `-1` survives unchanged; yet the affected type distribution matches its two
   conversion lists exactly (§8.3). The only consistent explanation is an earlier build without the
   guard. Which release was it, and did it corrupt anything else? In particular, did the unguarded
   version also touch `kSoundTrigger.where` (a `snd ` resource ID) — no shipped record shows damage,
   but the 1.0.4 list's deliberate exclusion of `kSoundTrigger` suggests the omission was a *fix*.
3. **`lightType.byte0`, `lightType.byte1` and `enemyType.byte0` are never written and never read.**
   Were they reserved for planned features (per-light colour? per-enemy variant?) or are they
   vestigial from a pre-1.0 layout? Zero in all 31 440 shipped objects.
4. **`furnitureType.pict` and `clutterType.pict` share offset 8 and the same meaning-slot, but only
   `kFlower` uses it.** Why does `furnitureType` have a `pict` field at all — was arbitrary-PICT
   furniture planned before `kCustomPict` (0x6E) took over that role?
5. **STR# 1007 has 144 strings, index 144 (`0x90`) being `"Mermaid"`, but `kNumSrcRects` is
   0x90 = 144, so the highest valid object ID is 0x8F.** "Mermaid" therefore names an object that
   does not exist. Was it cut, or is it a marker string? No `kMermaid` constant, `srcRects` entry,
   `GetObjectRect` arm or drawing routine exists.
6. **The 26 hole indices in STR# 1007 hold single junk characters** (`'f'`, `'9'`, `'a'`, ...
   see §2.2). These look like the residue of a hand-maintained resource, not deliberate padding.
   Is there an authoring tool that produced them?
7. **One object has `who == 35`**, beyond `kMaxRoomObs` = 24. Which house/room is it, and does the
   shipped game visibly misbehave there, or does the link simply resolve to an empty slot?
   (`GetObjectLinked` at `GliderPRO/Sources/Objects.c:177` does not bounds-check `who`.)
8. **6 of 4070 rooms have a hole in their object array** (an empty slot below a used one) while
   `numObjects` still equals highest-used-slot + 1. `numObjects` is therefore a high-water mark,
   not a count — but is anything in the original relying on it being a *count*?
9. **`kFloorTrans` at v = 300 and 301** cannot be produced by `AddNewObject`, which forces
   `kFloorTransTop` = 302. Editor bug, older constant, or converter artefact?
10. **`kSparkle` is counted against the 18-slot dynamic-object budget by
    `HowManyDynamicObjects`, but `kMaxSparkles` is 3.** Which of the two limits actually binds at
    runtime, and what happens visually in a room with 10 sparkles?
11. **`houseType.flags` bit 0 is documented as `wardBit`** in
    `GliderPRO/Headers/GliderStructs.h:187`. No object field references it. What did it gate?
12. **`roomType.bounds` is a `short`, not a `Rect`** (`GliderPRO/Headers/GliderStructs.h:169`),
    despite the name. It is written by the editor but its per-bit meaning is not derivable from the
    object code alone.

---

## Porting notes

### Decoding the record

1. Read the whole data fork into memory; there is **no serialisation layer** in the original. The
   on-disk image is a verbatim big-endian 68k/PPC struct dump. In Go, decode explicitly with
   `encoding/binary.BigEndian` — do **not** `unsafe`-cast, because Go gives no layout guarantees
   and the original relies on zero padding everywhere.
2. Object record: `what int16` at +0, then a 10-byte payload at +2. Total 12 bytes, always.
   `what == -1` (`kObjectIsEmpty`) means the slot is empty; skip it without interpreting the
   payload.
3. Room record: 348 bytes, objects at +60, 24 of them (`kMaxRoomObs`). House header: 866 bytes.
   Room `r`, object `i` starts at `866 + r*348 + 60 + i*12`. Verified against all 22 shipped houses.
4. **`Point` is `{v, h}` — vertical first.** Every `topLeft` on disk is (v, h). Getting this
   backwards produces plausible-looking but wrong geometry, because both components are in
   0..512-ish range.
5. **`Rect` is `{top, left, bottom, right}`** — top first, and *not* (x0, y0, x1, y1). Note that
   `QSetRect(r, left, top, right, bottom)` (`GliderPRO/Sources/RectUtils.c:210-216`) takes its
   arguments in the *opposite* order to the struct. If you write a `SetRect` helper in Go, pick one
   order and be consistent; the original's inconsistency is a bug farm.
6. **`Boolean` is a byte; test `!= 0`.** `kInvisBlower` records with `state == 23` exist.
7. **Do not model the union with a single flat struct.** Nine variants share the 10 bytes with
   *deliberately different* field orders:
   - `bonusType` has `state` at +8 and `initial` at +9 — **the opposite of** `blowerType`,
     `lightType`, `applianceType` and `enemyType`, which all have `initial` at +8 and `state` at +9.
   - `applianceType` is `byte0` at +6, `delay` at +7; `enemyType` is `delay` at +6, `byte0` at +7.
     **They are swapped.** A shared "appliance-or-enemy" struct will read the wrong byte.
   The safest Go shape is a `[10]byte` payload plus typed accessor methods per class, or nine
   distinct structs plus a `switch` on the class of `what`. Either way, keep the raw 10 bytes so
   round-tripping is byte-exact.
8. **Preserve the aliasing where the original exploits it.** Three known cases:
   `kFish` is dispatched as an enemy for its rect and hot spot but its `initial`/`state` are read
   through the appliance spelling in places; `kCustomPict` stores a resource ID in
   `applianceType.height`; `kSoundTrigger` stores a resource ID in `switchType.where`. If you keep
   nine separate structs, make sure the *same bytes* back all nine views.
9. **Round-trip test**: decode and re-encode all 22 shipped houses and require byte equality
   (modulo Sampler's 2 trailing bytes). This is cheap and catches every field-order mistake at once.

### Semantics to preserve

10. `numObjects` is a **high-water mark**, not a count. Always iterate all 24 slots and skip
    `what == -1`; use `numObjects` only when you must reproduce the original's own loop bounds.
11. **A link needs both halves.** Treat an object as linked only when `where` resolves to a real
    room **and** `who != 255`, exactly as `ListAllLocalObjects` does. There are two "no link"
    `where` encodings in shipped data (`-1`, 815 records; `-100`, 165 records) plus 14 records that
    have a valid `where` and `who == 255`. Do not normalise `-100` to `-1` on load if you want
    byte-exact saves.
12. **Bounds-check `who` against 24** before indexing; one shipped record has `who == 35`.
13. `where = suite*100 + (floor + 8)` (`kNumUndergroundFloors` = 8). Decode with
    `suite = where / 100`, `floor = (where % 100) - 8` — matching `ExtractFloorSuite`'s v2 branch
    (`GliderPRO/Sources/Link.c:41-53`). Go's `/` and `%` truncate toward zero exactly as C's do, so
    a direct transcription is correct — but do **not** "improve" it to floor division: that would
    change what `-1` and `-100` decode to, and `-100` decoding to `(suite -1, floor -8)` is what
    makes the shipped half-converted houses behave as unlinked. Also remember the v1 branch exists
    and swaps the two fields; a loader that must handle `version < 0x0200` needs both.
14. `kDeluxeTrans` packs **two** values into `tall` (`width/4` in the high byte, `height/4` in the
    low byte) and **two** into `wide` (`initial` in the high nibble, `state` in the low nibble).
    Multiply by 4 to get pixels. `kToggle` on a deluxe transport flips the low nibble only.
15. `kFlourescent` and `kTrackLight` store a **width** in `length`: it is assigned to `right` while
    `left` is still 0, *before* the room-relative offset is applied, so the room-absolute right edge
    is `length + topLeft.h` (`GliderPRO/Sources/ObjectRects.c:190-198`, clamp at
    `GliderPRO/Sources/HouseLegal.c:429-430`).
    `kSlider` and `kInvisTrans` store *relative* extents. `kLiftArea` stores its **width** in
    `distance` and **half its height** in `tall`. There is no single "size field" convention.
16. Reproduce the toaster's **iterative** ballistic solve, not a closed form: the original loops
    to find the initial `vVel` that reaches `data.g.height` under integer gravity, and the integer
    truncation is part of the observable trajectory.
17. Blower `vector` is a bit field read as `vector & 0x0F`: `1` = up, `2` = right, `4` = down,
    `8` = left (`GliderPRO/Headers/GliderStructs.h:16-17`). Only `kInvisBlower` and `kLiftArea`
    expose it in the editor; every other blower type has it forced, and `kRightFan` ignores it
    entirely.
18. Grease's `initial` is **inverted** in the editor UI (`GliderPRO/Sources/ObjectInfo.c:1892-1903`)
    and in the meaning of `state`: `state == true` means *not yet spilled*.
19. Hot spots are rebuilt from scratch on every room transition by `CreateActiveRects` over the
    9-room neighbourhood; the budget is `kMaxHotSpots` = 56 and the original **silently drops
    overflow**. A port should either reproduce the truncation or grow the array — but note that
    growing it changes gameplay in dense houses.
20. **The master-object list has stride 24 and includes empty slots** (§1.8):
    `masterIndex = neighbourOrdinal*24 + slot`, `numMasterObjects` is always a multiple of 24. If
    your port compacts the list (skipping `what == -1`), you change every `who`/`localLink`/`hotNum`
    relationship and you *fix* bugs #6 and #9 by accident. Decide deliberately whether you want
    that: fixing them changes gameplay in houses that were authored around the broken behaviour.
21. There are **three** live copies of every object's mutable state: the house handle
    (`(*thisHouse)->rooms[r].objects[i]`), `masterObjects[n].theObject`, and
    `thisRoom->objects[i]`. `SetObjectState` writes the house handle and `masterObjects`, plus
    `thisRoom->objects[object]` as well **but only when `room == thisRoomNumber`**; the
    drawing code reads `masterObjects`; the editor reads `thisRoom`. In Go, use **one** source of
    truth (pointers into a single room slice) and audit every place the original wrote only one
    copy — those are behavioural details, not just cleanliness issues.

### Toolbox replacements

21. **QuickDraw `CopyBits`/`CopyMask` from GWorlds** → blit from a decoded sprite atlas into an
    offscreen `*image.Paletted` (or a GPU texture). Every object sprite is a sub-rect of one of the
    24 PICT sheets listed in §3.2. The mask PICT is at `ID + 1000` for *most* sheets, but not all:
    `kAngelPictID`'s mask is at **`ID + 1`** (= 1020, `GliderPRO/Sources/StructuresInit2.c:134`), and
    `kSwitchPictID`, `kSupportPictID` and `kBadgePictID` have **no** mask PICT at all — see §3.2.
22. **Resource Manager** (`GetPicture`, `GetIndString`, `GetResource('snd ', n)`) → a static asset
    table keyed by the same numeric IDs. Keep the IDs: houses reference PICT IDs 10000..32767 in
    `kCustomPict.height` and `snd ` IDs 3000..32767 in `kSoundTrigger.where`, and those IDs live in
    the house's own resource fork, which BinHex preserves. Object names come from
    `STR# kObjectNameStrings` = 1007, indexed **by object ID** (1-based `GetIndString`).
23. **8-bit indexed colour.** `kRedOrangeColor8` = 23 is a CLUT index, not an RGB value. The
    original ships an 8-bit system palette; any port must either carry the 256-entry CLUT or
    translate the handful of hard-coded indices.
24. **Sound Manager** channels → any mixer, but note `LoadTriggerSound` failing is *load-bearing*:
    a `kSoundTrigger` whose sound is missing gets **no hot spot at all**
    (`GliderPRO/Sources/ObjectRects.c:923-928`).
25. **`NewPtr` vs `NewPtrClear`.** The original allocates `srcRects` uninitialised. Zero-filling in
    Go is fine and safer, but remember it is a *difference*: any code path that reads a hole index
    behaved unpredictably before and will behave deterministically now.
26. **Pascal strings.** `Str27 name` in `roomType` is 28 bytes: a length byte plus up to 27
    MacRoman characters. Decode as MacRoman, not UTF-8; several shipped house and room names use
    high bytes (this is exactly why `grep` treats the sources as binary).
27. **Big-endian everywhere on disk, including inside the resource fork.** BinHex itself is
    big-endian too (its 4-byte `dlen`/`rlen` header fields).

