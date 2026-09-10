# Glider PRO 1.0.4 — House (Level) File Format: Byte-Exact On-Disk Specification

## Scope

This document specifies, byte for byte, the on-disk format of a Glider PRO
"house" — the file that holds a complete playable level set (rooms, objects,
links, banner text, high scores, and an embedded saved game). It covers:

- The two-fork file structure, Finder type/creator codes, and sidecar files.
- Every field of `houseType`, `scoresType`, `gameType`, `roomType`, and all nine
  `objectType` union variants: type, size, absolute byte offset, endianness, and
  semantics.
- Every object type code (0x01–0x8F) and which union variant decodes it.
- Every background-PICT identifier and the resource-fork art/sound conventions.
- The (suite, floor) room grid, room adjacency, and the `where`/`who` link
  encoding, including the version-1 → version-2 migration.
- The complete `CheckHouseForProblems` legality pass, in original order, as
  numbered pseudocode.
- The load and save algorithms as numbered pseudocode.
- Empirically observed bytes from all 22 shipped houses, decoded with
  `tools/probe_house.py`, so that every layout claim below is verified rather
  than inferred.

It does **not** cover object *runtime behaviour* (dynamics, collision,
animation); those live in the sibling documents `docs/analysis/object-taxonomy.md`,
`docs/analysis/object-dynamics.md`, `docs/analysis/interactions.md`, and
`docs/analysis/enemies.md`. Where a field's *meaning* is needed to explain its
*encoding* (e.g. `transportType.tall` packing two nibbles for `kDeluxeTrans`),
this document states it.

### A correction to a common premise

There is **no resource type and no resource ID that holds the house level data.**
The level data is the entire **data fork**, written as a raw memory image of the
in-memory `houseType` structure. The resource fork of a house file holds only
*supplementary* material: custom room-background pictures, custom object
pictures, room-boundary hints, custom sounds, and the Finder icon family. This is
proved in "Part 1.2" below from `FSpOpenDF` / `GetEOF` / `FSRead` /
`NewHandle(byteCount)` in `GliderPRO/Sources/HouseIO.c`, and confirmed
empirically: `Demo House`'s data fork is exactly 16526 bytes, which equals
`866 + 45 * 348` for its 45 rooms, and the shipping binary hard-asserts both
numbers (`GliderPRO/Sources/HouseIO.c:349`, `GliderPRO/Sources/HouseIO.c:382`).

### Conventions used in this document

- All multi-byte scalars on disk are **big-endian** (68k/PowerPC Mac).
- Offsets are decimal unless prefixed `0x`. "abs" means offset from the start of
  the data fork; "rel" means offset from the start of the containing record.
- Citations take the form `GliderPRO/Sources/HouseIO.c:341`, with paths relative
  to the repository root the repository root. **Line numbers refer to
  the sources with classic-Mac CR line endings converted to LF** (`tr '\r' '\n'`),
  which is what a modern editor or `grep -n` will show after conversion. The
  original files in `GliderPRO/Sources/` use bare CR and appear as one line to
  POSIX tools.
- Pseudocode keeps the original C identifiers in parentheses so a port can be
  diffed against the C line by line.

## Sources read

Read in full (CR→LF converted copies):

| File | Lines | Why it matters here |
|---|---|---|
| `GliderPRO/Headers/GliderStructs.h` | 347 | Authoritative struct definitions with the author's own size comments |
| `GliderPRO/Headers/GliderDefines.h` | 625 | Every constant: object codes, background IDs, limits, versions |
| `GliderPRO/Headers/House.h` | 12 | Only `thisHouseName`, `houseUnlocked` |
| `GliderPRO/Sources/HouseIO.c` | 708 | Open/read/write/close; proves data fork is the payload |
| `GliderPRO/Sources/House.c` | 860 | Create/initialize; link enumeration; version migration |
| `GliderPRO/Sources/HouseInfo.c` | 343 | Banner/trailer/flags/lock editing; version display |
| `GliderPRO/Sources/HouseLegal.c` | 1219 | The complete legality / repair pass |
| `GliderPRO/Sources/Validate.c` | 398 | Copy-protection stub (touches nothing in the format) |
| `GliderPRO/Sources/Link.c` | 396 | `MergeFloorSuite` / `ExtractFloorSuite` — the link encoding |
| `GliderPRO/Sources/Room.c` | 1206 | Room create/delete, tiles, backgrounds, `bnds`, neighbours |
| `GliderPRO/Sources/RoomInfo.c` | 907 | The `bounds` bitfield writer |
| `GliderPRO/Sources/Objects.c` | (partial, link resolution) | `GetRoomLinked` / `GetObjectLinked` |
| `GliderPRO/Sources/ObjectRects.c` | (full geometry switch) | Which union field drives each object's rect |
| `GliderPRO/Sources/ObjectAdd.c` | (defaults) | Field defaults for newly created objects |
| `GliderPRO/Sources/ObjectInfo.c` | (validators) | Legal ranges for `kCustomPict`, `kSoundTrigger`, `kInvisBonus` |
| `GliderPRO/Sources/ObjectEdit.c` | (partial) | `leftStart`/`rightStart` semantics; copy/paste field sets |
| `GliderPRO/Sources/Play.c` | (partial) | `SetObjectsToDefaults` — `state` ← `initial` |
| `GliderPRO/Sources/Transit.c` | (partial) | `leftStart`/`rightStart` used on side transit |
| `GliderPRO/Sources/HighScores.c` | (partial) | `scoresType` init; sidecar score file |
| `GliderPRO/Sources/SavedGames.c` | 352 | `gameType` writer; `kSavedGameVersion` |
| `GliderPRO/Sources/SelectHouse.c` | 676 | House discovery by type/creator; icon resource |
| `GliderPRO/Sources/Map.c` | (partial) | Grid→screen mapping, `kMapGroundValue` |
| `GliderPRO/Sources/Sound.c` | (partial) | `LoadTriggerSound`, sound-slot reservation |
| `GliderPRO/Sources/Triggers.c`, `Interactions.c` | (partial) | Sound-trigger playback path |
| `GliderPRO/Sources/RoomGraphics.c` | (partial) | Background `PICT`/`Date` lookup |

Empirical decoding: all 22 files in `GliderPRO/Houses/*.binhex` were BinHex-4.0
decoded and parsed by `tools/probe_house.py`
(written for this document; reusable). Observed values appear throughout and are
collected in Part 12.

---

# Part 1 — File-level anatomy

## 1.1 Forks, type, creator

A house is a classic Mac OS file with **two forks**:

| Fork | Contents |
|---|---|
| Data fork | The entire level: a raw big-endian memory image of `houseType` followed by `nRooms` × `roomType`. No header, no magic number, no checksum, no compression. |
| Resource fork | Optional custom art (`PICT`), room-boundary hints (`bnds`), custom sounds (`snd `), the Finder icon family, and sometimes a `vers` resource. |

Finder metadata:

| Field | Value | Citation |
|---|---|---|
| File type | `'gliH'` (0x67 0x6C 0x69 0x48) | `GliderPRO/Sources/House.c:58`, `GliderPRO/Sources/House.c:85` |
| Creator | `'ozm5'` (0x6F 0x7A 0x6D 0x35) | same |
| Score sidecar type | `'gliS'`, creator `'ozm5'` | `GliderPRO/Sources/HighScores.c:727` |
| Saved-game type (dead code) | `'gliG'`, creator `'ozm5'` | `GliderPRO/Sources/SavedGames.c:122` |

Creation is:

```c
FSpCreate(&theSpec, 'ozm5', 'gliH', theReply.keyScript);   // House.c:85
HCreateResFile(theSpec.vRefNum, theSpec.parID, theSpec.name);   // House.c:88
```

so the resource fork always exists, even when it is empty. Observed: `Sampler`'s
resource fork is 286 bytes — a bare 256-byte resource header plus a 30-byte map
declaring **zero** types (`FF FF` = `numTypes − 1` = −1). See Part 12.6.

House discovery scans directories for exactly this type/creator pair:

```c
if ((myCInfo.hFileInfo.ioFlFndrInfo.fdType == 'gliH') &&
    (myCInfo.hFileInfo.ioFlFndrInfo.fdCreator == 'ozm5'))   // SelectHouse.c:592-593
```

recursing into subdirectories when `ioFlAttrib & 0x10` is set
(`GliderPRO/Sources/SelectHouse.c:603-609`), to a depth-limited list of
`kMaxDirectories` = 32 (`GliderPRO/Sources/SelectHouse.c:559`), plus up to
`kMaxExtraHouses` = 8 explicitly registered houses
(`GliderPRO/Sources/SelectHouse.c:35`). The house named `"\pDemo House"` gets
special-cased as `demoHouseIndex` (`GliderPRO/Sources/SelectHouse.c:639`).

**Go port note.** There are no forks on Linux/Windows. The practical mapping is:
data fork → the file itself; resource fork → a sidecar (`<name>.rsrc`) or an
extracted directory of assets. Type/creator must become an out-of-band
convention (e.g. a filename extension) or a synthetic magic number — but note
that adding a magic number to the data fork **breaks byte compatibility**,
because the first two bytes of a house are `version` (see Part 3).

## 1.2 Proof that the level data is the data fork

`OpenHouse` opens the *data* fork:

```c
theErr = FSpOpenDF(&theHousesSpecs[thisHouseIndex], fsCurPerm, &houseRefNum);  // HouseIO.c:189
...
OpenHouseResFork();                                                            // HouseIO.c:194
```

`ReadHouse` then reads the whole fork into one relocatable block and treats that
block as a `houseType *`:

```c
theErr = GetEOF(houseRefNum, &byteCount);      // HouseIO.c:341
#ifdef COMPILEDEMO
if (byteCount != 16526L)                       // HouseIO.c:349
    return (false);                            // HouseIO.c:350
#endif
thisHouse = (houseHand)NewHandle(byteCount);   // HouseIO.c:356
...
theErr = FSRead(houseRefNum, &byteCount, *thisHouse);   // HouseIO.c:372
...
numberRooms = (*thisHouse)->nRooms;            // HouseIO.c:380
#ifdef COMPILEDEMO
if (numberRooms != 45)                         // HouseIO.c:382
    return (false);
#endif
```

`WriteHouse` writes the whole handle back and truncates:

```c
byteCount = GetHandleSize((Handle)thisHouse);  // HouseIO.c:473
...
theErr = FSWrite(houseRefNum, &byteCount, (Ptr)*thisHouse);   // HouseIO.c:491
...
theErr = SetEOF(houseRefNum, byteCount);       // HouseIO.c:499
```

The two `COMPILEDEMO` assertions are an independent cross-check on the layout:
16526 = 866 + 45 × 348 exactly, where 866 = `offsetof(houseType, rooms)` and
348 = `sizeof(roomType)`. Observed `Demo House` data fork length: **16526**.

The resource fork is opened separately and pushed onto the Resource Manager
search chain:

```c
houseResFork = FSpOpenResFile(&theHousesSpecs[thisHouseIndex], fsCurPerm);  // HouseIO.c:571
...
UseResFile(houseResFork);                                                    // HouseIO.c:575
```

and closed with `CloseResFile(houseResFork)` (`GliderPRO/Sources/HouseIO.c:586`).

## 1.3 Sidecar files

| Sidecar | Naming | Citation |
|---|---|---|
| QuickTime background movie | `<house file name>` + `".mov"` (Pascal-string concatenation, so literally the house filename with `.mov` appended, e.g. `Demo House.mov`) | `GliderPRO/Sources/HouseIO.c:81` |
| High-score file | a file in a folder named `"\pG-PRO Scores "` (note trailing space), type `'gliS'` | `GliderPRO/Sources/HighScores.c:696`, `GliderPRO/Sources/HighScores.c:727` |

The movie is only opened if `FSpGetFInfo` on the derived spec succeeds
(`GliderPRO/Sources/HouseIO.c:83-85`); a missing `.mov` is silently ignored.
Observed: 15 of the 22 shipped houses have a `.mov` sibling; `Demo House.mov`
begins `00 01 D0 08 6D 64 61 74` — a QuickTime `mdat` atom at offset 4, i.e. a
normal single-fork `.mov` file.

The score sidecar contains exactly `sizeof(scoresType)` = **292** bytes
(`GliderPRO/Sources/HighScores.c:771`) and is read straight over the in-memory
house's score block:

```c
theErr = FSRead(scoresRefNum, &byteCount, (Ptr)&((*thisHouse)->highScores));  // HighScores.c:841
```

so the sidecar's byte layout is identical to abs 528..819 of the data fork
(Part 3.2). `ReadScoresFromDisk()` is called after loading a **read-only** house
(`GliderPRO/Sources/HouseIO.c:431`), which is how scores are kept for houses on
locked volumes.

## 1.4 The BinHex 4.0 container in the source release

The GPL source release cannot ship resource forks, so each house is stored as a
BinHex 4.0 text file: `GliderPRO/Houses/<name>.binhex`. BinHex 4.0 is:

1. A `:`-delimited body over this exact 64-character alphabet (index 0 first):

   ```
   !"#$%&'()*+,-012345689@ABCDEFGHIJKLMNPQRSTUVXYZ[`abcdefhijklmpqr
   ```

   The alphabet is deliberately non-contiguous — it skips `7`, `.`, `/`, `:`
   through `?`, `O`, `W`, `\`, `]`, `^`, `_`, `g`, `n`, `o`, and `s` through `z`
   — so it must be used as a literal table, never computed from character
   ranges. (Verified: `len(BINHEX_ALPHABET) == 64` asserted in
   `tools/probe_house.py:29`.)
2. Six bits per character, MSB-first, concatenated into a byte stream.
3. RLE-decompressed: `0x90 <n>` means "repeat the previous byte n times";
   `0x90 0x00` is a literal `0x90`.
4. The decompressed stream is: `nameLen(1) name(nameLen) 0x00 type(4) creator(4)
   flags(2) dataLen(4) rsrcLen(4) headerCRC(2) data(dataLen) dataCRC(2)
   rsrc(rsrcLen) rsrcCRC(2)`.

The three CRCs are **standard CRC-16/XMODEM: polynomial 0x1021, seed 0, MSB-first,
no final XOR**, computed over each region (header, data fork, resource fork) on its
own. `tools/probe_house.py` spells it out the long way, shifting the message bits
*into* the register and running two extra `0x00` bytes through at the end:

```python
def binhex_crc(data, crc=0):
    for b in data:
        for i in range(8):
            hibit = crc & 0x8000
            crc = ((crc << 1) | ((b >> (7 - i)) & 1)) & 0xFFFF
            if hibit:
                crc ^= 0x1021
    return crc
# called as binhex_crc(fork + b"\x00\x00")
```

That formulation is *algebraically identical* to CRC-16/XMODEM over `fork` alone —
the two trailing zero bytes are exactly what flushes the shift register, which is
what the usual table-driven "XOR the byte into the high half" loop does implicitly.

**Verified: all 22 shipped houses pass the header, data-fork and resource-fork CRC
checks under both spellings** — an independent table-driven CRC-16/XMODEM over each
raw region matched all 66 stored values (22 × 3), and `binhex_crc(region + b"\x00\x00")`
gives the same 66 results. (`tools/probe_house.py` itself only checks the two fork
CRCs, at `tools/probe_house.py:137` and `140`; the header CRC was verified
separately.) So a Go port can simply use any off-the-shelf CRC-16/XMODEM implementation
(e.g. `github.com/sigurn/crc16` with `CRC16_XMODEM`) over the fork bytes; there is
no Glider-specific CRC variant to reproduce.

`tools/probe_house.py` implements the whole container (`binhex_decode`). Observed
for `Demo House.binhex`: name `'Demo House'`, type `'gliH'`, creator `'ozm5'`,
Finder flags `0x0500`, dataLen 16526, rsrcLen 491757.

**Go port note.** Ship houses in whatever container you like, but the decoder
must be able to read the BinHex originals to bootstrap. Watch the RLE marker:
a naive implementation that treats `0x90` as data corrupts every house.
(Implementation trap found while writing `probe_house.py`: accumulating the
6-bit groups into a Python int without masking makes decoding O(n²) and hangs on
the 185 KB houses; mask the accumulator to 24 bits.)

---

# Part 2 — Primitive types, byte order, and packing

## 2.1 Scalar types

| C type | Size | Signedness | On-disk encoding |
|---|---|---|---|
| `Byte` | 1 | unsigned (0..255) | raw byte |
| `Boolean` | 1 | 0 = false, non-zero = true | raw byte; the code writes `1` for true |
| `short` | 2 | signed | big-endian two's complement |
| `unsigned long` | 4 | unsigned | big-endian |
| `long` | 4 | signed | big-endian |

`Boolean` deserves care: the source tests `if (thisRoom->visited)`, i.e. any
non-zero value is true, but always *writes* `true` (= 1). Observed across all
4070 rooms in the corpus: `visited` is only ever 0 or 1 (3634 zeros, 436 ones).
`hasGame` is 0 or 1. `unusedBoolean`, however, is **uninitialized garbage** —
observed values 0, 2, 14, 30, 37, 185, 255 (Part 12.5). A Go port must not read
`unusedBoolean` as a bool that means anything.

## 2.2 `Point` — v BEFORE h

```c
struct Point { short v; short h; };   // classic Mac OS QuickDraw
```

**`v` (vertical) is at relative offset 0; `h` (horizontal) is at relative offset
2.** This is the single most dangerous field-order trap in the format, because
`Point` appears in `houseType.initial`, `gameType.where`, and in seven of the
nine object union variants. Three independent empirical proofs (Part 12.3):

1. `Demo House` room[0] obj[0] is `kFloorVent` (0x01). Decoded as `{v, h}` its
   `topLeft.v` = **305**, which is exactly `kFloorVentTop` = 305
   (`GliderPRO/Headers/GliderDefines.h:467`) — and `HouseLegal.c:115-117`
   *forces* `data.a.topLeft.v` to `kFloorVentTop` for every `kFloorVent`.
2. Same room, obj[8] `kCeilingLight` (0x51) decodes to `v` = **4** =
   `kCeilingLightTop` (`GliderPRO/Headers/GliderDefines.h:477`); room[1] obj[0]
   `kFlourescent` (0x56) decodes to `v` = **12** = `kFlourescentTop`
   (`GliderPRO/Headers/GliderDefines.h:480`).
3. Across all 22 houses, the first `short` of `houseType.initial` is always
   ≤ 302 — the exact clamp `kTileHigh − kGliderHigh` = 322 − 20 = 302 applied at
   `GliderPRO/Sources/HouseLegal.c:63-66` — while the second `short` reaches 424
   (`In The Mirror`), which would be an illegal `v`. Therefore the first short is
   `v`.

## 2.3 `Rect` — top, left, bottom, right

```c
struct Rect { short top; short left; short bottom; short right; };
```

Relative offsets 0, 2, 4, 6. Empirical proof: `Demo House` room[0] obj[1] is
`kDresser` (0x18); decoded in this order its `bottom` = **293** =
`kDresserBottom` (`GliderPRO/Headers/GliderDefines.h:476`), and `ObjectAdd.c:188`
forces `newRect.bottom = kDresserBottom` when a dresser is created. Second proof:
room[1] obj[1] is `kShelf` (0x12) with `bottom − top` = 158 − 152 = **6** =
`kShelfThick` (`GliderPRO/Headers/GliderDefines.h:440`).

`Rect` occupies 8 bytes and appears in `furnitureType` and `clutterType`.

## 2.4 Pascal strings

| Type | Bytes | Max chars | Layout |
|---|---|---|---|
| `Str255` | 256 | 255 | `len(1)` then 255 bytes of char storage |
| `Str31` | 32 | 31 | `len(1)` then 31 bytes |
| `Str27` | 28 | 27 | `len(1)` then 27 bytes |
| `Str15` | 16 | 15 | `len(1)` then 15 bytes |

The length byte is **unsigned**. Bytes past `len` are **not** cleared — they
hold whatever the buffer previously contained. Observed: `Demo House.banner` has
`len` = 111 and the 145 trailing bytes contain residue from the longer
`trailer` string. A byte-exact writer must reproduce that residue if it wants
identical files; a *semantic* writer should zero-fill (which changes bytes but
not behaviour).

Character encoding is **MacRoman**, not UTF-8 or Latin-1. `\r` (0x0D) is the
in-string line break used by the banner/trailer word-wrapper
(`GliderPRO/Sources/HouseLegal.c:611-612`).

Room names are `Str27`, and `CheckRoomNameLength` hard-clamps the length byte:

```c
if (name[0] > 27)      // HouseLegal.c:871
    name[0] = 27;      // HouseLegal.c:873
```

Observed across 4070 rooms: `name[0]` ranges 1..27, never 0 and never > 27.

## 2.5 Struct packing and the 866-vs-868 discrepancy

All the level structs are naturally aligned with **no interior padding**: every
`short` sits at an even offset, every `long` at a multiple of 4, and every
`Byte`/`Boolean` pair is adjacent so the following `short` lands even. The
author annotated each struct with its own size (`GliderPRO/Headers/GliderStructs.h`
lines 19, 25, 34, 43, 52, 62, 72, 82, 88, 105, 114, 134, 142, 164, 180, 198) and
those annotations match a hand computation exactly.

However, the *trailing* padding differs by ABI:

- `offsetof(houseType, rooms)` = **866** on every ABI. This is the number that
  matters for reading rooms.
- `sizeof(houseType)` = **866** under 68k `mac68k` alignment, but **868** under
  PowerPC alignment, because `houseType` contains `long` members and PowerPC
  rounds the total size up to a multiple of 4.

`LopOffExtraRooms` computes the file size from `sizeof(houseType)`:

```c
newSize = sizeof(houseType) + (sizeof(roomType) * (long)r);   // HouseLegal.c:765
...
SetHandleSize((Handle)thisHouse, newSize);                    // HouseLegal.c:767
```

so a PowerPC build writes **two extra slack bytes**. This is observable:
`Sampler` has `nRooms` = 2 and a data fork of **1564** bytes, but
866 + 2 × 348 = 1562. The last object slot of room[1] runs 1550..1561; bytes
1562..1563 are `01 01`. Rooms still parse correctly from abs 866. See Part 12.4
for the hexdump.

`ValidateNumberOfRooms` uses integer division, which silently tolerates the
slack:

```c
countedRooms = (GetHandleSize((Handle)thisHouse) -
        sizeof(houseType)) / sizeof(roomType);   // HouseLegal.c:630-631
```

With `sizeof(houseType)` = 868 and size 1564: (1564 − 868) / 348 = 696 / 348 = 2.
Correct. With `sizeof(houseType)` = 866: (1564 − 866) / 348 = 698 / 348 = 2.
Also correct. The division absorbs the difference either way.

**Go port rule: derive the room count from the `nRooms` field, not from the file
size. Treat any bytes after `866 + 348 * nRooms` as slack and preserve them
verbatim if you want byte-identical round-trips.** Reject a file only if it is
*shorter* than `866 + 348 * nRooms`.
---

# Part 3 — `houseType`: the 866-byte header

Definition, verbatim (`GliderPRO/Headers/GliderStructs.h:182-198`):

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

## 3.1 Offset table

| abs offset | dec | hex | Size | C type | Field | Notes |
|---|---|---|---|---|---|---|
| 0 | 0 | 0x000 | 2 | `short` | `version` | BE. `0x0200` for all shipped houses |
| 2 | 2 | 0x002 | 2 | `short` | `unusedShort` | **uninitialized garbage**; never read |
| 4 | 4 | 0x004 | 4 | `long` | `timeStamp` | Mac epoch seconds; **bit 0 = house-locked flag** |
| 8 | 8 | 0x008 | 4 | `long` | `flags` | bitfield, see 3.4 |
| 12 | 12 | 0x00C | 2 | `short` | `initial.v` | glider start Y, clamped 0..302 |
| 14 | 14 | 0x00E | 2 | `short` | `initial.h` | glider start X, clamped 0..464 |
| 16 | 16 | 0x010 | 1 | `Byte` | `banner[0]` | length, 0..255 |
| 17 | 17 | 0x011 | 255 | `char[255]` | `banner[1..255]` | MacRoman, `\r` = line break |
| 272 | 272 | 0x110 | 1 | `Byte` | `trailer[0]` | length |
| 273 | 273 | 0x111 | 255 | `char[255]` | `trailer[1..255]` | |
| 528 | 528 | 0x210 | 292 | `scoresType` | `highScores` | see 3.5 |
| 820 | 820 | 0x334 | 40 | `gameType` | `savedGame` | see 3.6 |
| 860 | 860 | 0x35C | 1 | `Boolean` | `hasGame` | 1 ⇒ `savedGame` is valid |
| 861 | 861 | 0x35D | 1 | `Boolean` | `unusedBoolean` | **uninitialized garbage** |
| 862 | 862 | 0x35E | 2 | `short` | `firstRoom` | index into `rooms[]`, or −1 for a new house |
| 864 | 864 | 0x360 | 2 | `short` | `nRooms` | room-array length, including placeholders |
| 866 | 866 | 0x362 | 348·n | `roomType[]` | `rooms` | see Part 5 |

Total header 866 bytes; total file `866 + 348 * nRooms` (+0 or 2 slack bytes, see 2.5).

## 3.2 `version` and format migration

| Constant | Value | Citation |
|---|---|---|
| `kHouseVersion` | `0x0200` (512) | `GliderPRO/Headers/GliderDefines.h:517` |
| `kNewHouseVersion` | `0x0300` (768) | `GliderPRO/Headers/GliderDefines.h:518` |

`version` is a packed BCD-ish major/minor: displayed as
`version >> 8` "." `version % 0x0100`
(`GliderPRO/Sources/HouseInfo.c:232-233`), so `0x0200` renders as "2.0".

The load path enforces a hard ceiling but a soft floor:

```c
wasHouseVersion = (*thisHouse)->version;        // HouseIO.c:394
if (wasHouseVersion >= kNewHouseVersion)        // HouseIO.c:395
{
	YellowAlert(kYellowNewerVersion, 0);        // (HouseIO.c:397)
	...
	return(false);                              // HouseIO.c:399
}
```

So:

- `version < 0x0200` (i.e. any version 1.x house): **accepted**, and read with
  the legacy link decoding (see Part 7.3). It is upgraded to 0x0200 lazily, only
  when the editor decides to save — `QuerySaveChanges` calls
  `ConvertHouseVer1To2()` then sets `wasHouseVersion = kHouseVersion`
  (`GliderPRO/Sources/HouseIO.c:610-612`).
- `version == 0x0200`: current format.
- `version >= 0x0300`: **refused** with alert `kYellowNewerVersion`.

`WriteHouse` writes back `wasHouseVersion`, not `kHouseVersion`:

```c
(*thisHouse)->version = wasHouseVersion;   // HouseIO.c:488
```

which means a version-1 house saved **from play mode** keeps its version-1 link
encoding, while one saved through the editor's `QuerySaveChanges` is migrated.

Observed: all 22 shipped houses report `version == 0x0200`. No version-1 house is
available for empirical verification of the legacy path.

**Go port note.** Implement `version < 0x0200` link decoding even though no
sample exists; it is only two lines and third-party 1.x houses may still be in
circulation.

## 3.3 `timeStamp` and the lock bit

`timeStamp` is a `long` holding classic-Mac-epoch seconds (seconds since
1904-01-01 00:00:00 local time) from `GetDateTime`. **Bit 0 is stolen as a
"house is locked" flag:**

```c
houseUnlocked = (((*thisHouse)->timeStamp & 0x00000001) == 0);   // HouseIO.c:402
```

On save:

```c
GetDateTime(&timeStamp);                          // HouseIO.c:477
timeStamp &= 0x7FFFFFFF;                          // HouseIO.c:478
...
if (houseUnlocked)
	timeStamp &= 0x7FFFFFFE;                      // HouseIO.c:484
else
	timeStamp |= 0x00000001;                      // HouseIO.c:486
(*thisHouse)->timeStamp = (long)timeStamp;        // HouseIO.c:487
```

Note both maskings: `& 0x7FFFFFFF` clears the sign bit so the stamp is always a
non-negative `long`, and the low bit is then forced to the lock state. A locked
house cannot be edited; `HouseInfo.c:282-290` is the UI that sets
`changeLockStateOfHouse = true; saveHouseLocked = true;` — note there is **no
unlock path in the UI**, locking is one-way from the user's point of view.

Because bit 0 is a flag, the stamp has 2-second granularity.

Observed lock bits across the corpus: 15 locked, 7 unlocked. `Demo House`
`timeStamp` = `0x2C53B041` = 743682113 (odd ⇒ locked).

## 3.4 `flags` bitfield

| Bit | Mask | Meaning when set | Read at | Written at |
|---|---|---|---|---|
| 0 | `0x00000001` | `wardBitSet` — non-existent neighbour rooms are painted flat instead of being filled with sky/meadow/dirt tiles | `GliderPRO/Sources/HouseIO.c:416`; used `GliderPRO/Sources/RoomGraphics.c:195-206` | never (no editor UI in 1.0.4) |
| 1 | `0x00000002` | `phoneBitSet` — **suppresses** the random ringing-telephone event | `GliderPRO/Sources/HouseIO.c:417`; used `GliderPRO/Sources/Play.c:748` | `GliderPRO/Sources/HouseInfo.c:270` (set), `GliderPRO/Sources/HouseInfo.c:272` (clear) |
| 2 | `0x00000004` | **inverted**: `bannerStarCountOn = ((flags & 4) == 0)`, i.e. bit 2 set *hides* the "n stars remaining" line on the banner | `GliderPRO/Sources/HouseIO.c:418`; used `GliderPRO/Sources/Banner.c:142` | never |
| 3..31 | — | unused | — | — |

Two things a port must reproduce exactly:

1. **Bit 2 is inverted.** `bannerStarCountOn` is true when the bit is *clear*.
   Getting this backwards changes the banner of every house.
2. **The clear mask for bit 1 is `0xFFFFDFFD`, not `0xFFFFFFFD`**
   (`GliderPRO/Sources/HouseInfo.c:272`). `0xFFFFDFFD` = `~0x00002002`, so
   turning the phone back on also clears **bit 13**. Bit 13 is never used
   elsewhere, so this is harmless in 1.0.4, but a byte-exact writer must
   reproduce it.

Observed `flags` values across the corpus: `0x00000000` (14 houses),
`0x00000002` (7 houses), `0x00000006` (1 house — `Art Museum`, which therefore
sets both the phone-suppress and the hide-star-count bits). No house sets bit 0.

## 3.5 `scoresType` — the embedded high-score table

Definition (`GliderPRO/Headers/GliderStructs.h:107-114`), `kMaxScores` = 10
(`GliderPRO/Headers/GliderDefines.h:249`):

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

| rel | abs | Size | Type | Field |
|---|---|---|---|---|
| 0 | 528 | 32 | `Str31` | `banner` — the "hall of fame" title string |
| 32 | 560 | 160 | `Str15[10]` | `names[i]` at abs `560 + 16*i` |
| 192 | 720 | 40 | `long[10]` | `scores[i]` at abs `720 + 4*i` |
| 232 | 760 | 40 | `unsigned long[10]` | `timeStamps[i]` at abs `760 + 4*i` |
| 272 | 800 | 20 | `short[10]` | `levels[i]` at abs `800 + 2*i` |
| 292 | 820 | | | end |

Semantics:

- The list is kept **descending by score**; `TestHighScore` inserts by scanning
  `if (theScore > scores[i])` (`GliderPRO/Sources/HighScores.c:392`), stages the
  new entry in the last slot `kMaxScores - 1`
  (`GliderPRO/Sources/HighScores.c:404`, `410-412`), then calls `SortHighScores()`.
- `levels[i]` is **not** a level number; it is `CountRoomsVisited()` — the number
  of rooms whose `visited` byte was set during that run
  (`GliderPRO/Sources/HighScores.c`, and `CountRoomsVisited` at
  `GliderPRO/Sources/House.c:506-531` reading `rooms[r].visited` at
  `GliderPRO/Sources/House.c:525`).
- `timeStamps[i]` is an **unsigned** `long` Mac-epoch seconds with no flag bits.
- `ZeroHighScores` (`GliderPRO/Sources/HighScores.c:323-343`) sets
  `banner` ← the house's own name (`GliderPRO/Sources/HighScores.c:333`), every
  `names[i]` ← `"\p--------------"` (14 hyphens,
  `GliderPRO/Sources/HighScores.c:336`), and all scores/timeStamps/levels ← 0
  (`GliderPRO/Sources/HighScores.c:337-339`).
- `ZeroAllButHighestScore` (`GliderPRO/Sources/HighScores.c:348`) zeroes indices
  1..9 only (`GliderPRO/Sources/HighScores.c:358-363`).

Observed `Demo House`: `highScores.banner` = `'The Return of Ozma!'`,
`names[0]` = `'Ozma'`, `scores[0]` = 7400, `timeStamps[0]` = 2888823855,
`levels[0]` = 13; slots 1..9 are the 14-hyphen placeholder with zeros.
Observed `Sampler`: banner `'Your Message Here'`, two entries
(`'Your Name'` 5200 level 2, `'Your Name'` 5100 level 1).

## 3.6 `gameType` — the embedded saved game

Definition (`GliderPRO/Headers/GliderStructs.h:116-134`):

| rel | abs | Size | Type | Field | Notes |
|---|---|---|---|---|---|
| 0 | 820 | 2 | `short` | `version` | `kSavedGameVersion` = `0x0200` (`GliderPRO/Sources/SavedGames.c:14`) |
| 2 | 822 | 2 | `short` | `wasStarsLeft` | stars remaining when saved |
| 4 | 824 | 4 | `long` | `timeStamp` | `GetDateTime` at save (`GliderPRO/Sources/SavedGames.c:320-321`) |
| 8 | 828 | 2 | `short` | `where.v` | `theGlider.dest.top` (`GliderPRO/Sources/SavedGames.c:323`) |
| 10 | 830 | 2 | `short` | `where.h` | `theGlider.dest.left` (`GliderPRO/Sources/SavedGames.c:322`) |
| 12 | 832 | 4 | `long` | `score` | |
| 16 | 836 | 4 | `long` | `unusedLong` | forced to 0 on save (`GliderPRO/Sources/SavedGames.c:325`) |
| 20 | 840 | 4 | `long` | `unusedLong2` | forced to 0 (`GliderPRO/Sources/SavedGames.c:326`) |
| 24 | 844 | 2 | `short` | `energy` | |
| 26 | 846 | 2 | `short` | `bands` | rubber bands carried |
| 28 | 848 | 2 | `short` | `roomNumber` | index into `rooms[]` |
| 30 | 850 | 2 | `short` | `gliderState` | |
| 32 | 852 | 2 | `short` | `numGliders` | lives |
| 34 | 854 | 2 | `short` | `foil` | |
| 36 | 856 | 2 | `short` | `unusedShort` | forced to 0 (`GliderPRO/Sources/SavedGames.c:333`) |
| 38 | 858 | 1 | `Boolean` | `facing` | |
| 39 | 859 | 1 | `Boolean` | `showFoil` | |
| 40 | 860 | | | end | |

`hasGame` at abs 860 gates validity: `SaveGame` sets it true
(`GliderPRO/Sources/SavedGames.c:337`) or false
(`GliderPRO/Sources/SavedGames.c:341`) and then calls
`WriteHouse(theMode == kEditMode)` (`GliderPRO/Sources/SavedGames.c:348`), so the
saved game rides along inside the house file.

**When `hasGame` is 0 the whole 40-byte block is stale garbage**, because
`InitializeEmptyHouse` never touches it (`GliderPRO/Sources/House.c:108-145` sets
`version`, `firstRoom`, `timeStamp`, `flags`, `initial`, high scores, banner,
trailer, `hasGame`, `nRooms` — and nothing else) and `NewHandle` does not zero
memory. Observed `Sampler` with `hasGame` = 0: `savedGame.version` = 1,
`wasStarsLeft` = 8, `unusedLong` = 4668932, `bands` = −1, `facing` = 204,
`showFoil` = 204 — obvious heap residue. Observed `Demo House`, also
`hasGame` = 0, still carries a plausible-looking `version` = 0x0100,
`score` = 3000, `energy` = 50, `roomNumber` = 15 from a previous session.

Only 2 of 22 shipped houses have `hasGame` = 1: `ImagineHouse PRO II` and
`Titanic`.

There is a **second, larger** saved-game struct, `game2Type`
(`GliderPRO/Headers/GliderStructs.h:144-164`, 114 bytes + `savedRoom[]` at 292
bytes each), intended for a standalone `'gliG'` file. Its writer
(`SaveGame2`, `GliderPRO/Sources/SavedGames.c:33-147`) and its reader
(`OpenSavedGame`, `GliderPRO/Sources/SavedGames.c:169+`) are **entirely commented
out** in 1.0.4 — `OpenSavedGame` short-circuits with
`return false;  // TEMP fix this iwth NavServices`
(`GliderPRO/Sources/SavedGames.c:169`). A Go port can ignore `game2Type` and
`savedRoom` for file-format purposes; they never reach disk.

## 3.7 `firstRoom` and `nRooms`

`firstRoom` is the index into `rooms[]` where play begins. It is set to −1 for a
brand-new house (`GliderPRO/Sources/House.c:128`) and clamped **on use, in a local
variable only** — the stored field is never repaired:

```c
short GetFirstRoomNumber (void)          // House.c:191
{
	short	firstRoom;
	...
	if ((*thisHouse)->nRooms <= 0)
	{
		firstRoom = -1;
		noRoomAtAll = true;              // House.c:203-207
	}
	else
	{
		firstRoom = (*thisHouse)->firstRoom;
		if ((firstRoom >= (*thisHouse)->nRooms) || (firstRoom < 0))
			firstRoom = 0;               // House.c:209-213
	}
	...
	return (firstRoom);
}
```

Note the two consequences for a port: (a) an out-of-range `firstRoom` on disk stays
out of range on disk — nothing writes 0 back into the house header, so the bad value
survives a load/save round trip; and (b) when `nRooms <= 0` the function returns −1
*and* sets the global `noRoomAtAll`, rather than clamping to 0.

`nRooms` counts **array slots**, including deleted placeholders. The count of
*real* rooms is:

```c
short RealRoomNumberCount (void)              // House.c:164
{
	...
	for (i = 0; i < (*thisHouse)->nRooms; i++)
		if ((*thisHouse)->rooms[i].suite == kRoomIsEmpty)   // House.c:~183
			count--;   // subtract placeholders
	...
}
```

(`GliderPRO/Sources/House.c:170-189`), and if that count is zero the house is
flagged `noRoomAtAll` (`GliderPRO/Sources/HouseIO.c:422`).

Observed: **zero placeholder rooms in all 22 shipped houses**, because
`CompressHouse` and `LopOffExtraRooms` run unconditionally on every editor save
(Part 9.2). Observed `firstRoom`: 0, 1, 4, 6, 8, 14, 20, 29, 30, 33, 39, 43, 70,
91, 92, 126, 127, 259 — all within `[0, nRooms)`.

---

# Part 4 — `roomType`: the 348-byte room record

Definition, verbatim (`GliderPRO/Headers/GliderStructs.h:166-180`):

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

Room `r` starts at absolute offset `866 + 348 * r`.

## 4.1 Offset table

| rel | Size | Type | Field | Notes |
|---|---|---|---|---|
| 0 | 1 | `Byte` | `name[0]` | length, clamped to ≤ 27 |
| 1 | 27 | `char[27]` | `name[1..27]` | MacRoman; default `"Untitled Room"` |
| 28 | 2 | `short` | `bounds` | wall-opening bitfield, see 4.2 |
| 30 | 1 | `Byte` | `leftStart` | glider entry Y offset when entering from the **east** |
| 31 | 1 | `Byte` | `rightStart` | glider entry Y offset when entering from the **west** |
| 32 | 1 | `Byte` | `unusedByte` | force-zeroed by validation |
| 33 | 1 | `Boolean` | `visited` | runtime residue; reset by `SetObjectsToDefaults` |
| 34 | 2 | `short` | `background` | PICT resource ID, see 4.5 |
| 36 | 16 | `short[8]` | `tiles[0..7]` | tile column indices, `tiles[i]` at rel `36 + 2*i` |
| 52 | 2 | `short` | `floor` | vertical grid coordinate, legal −7..56 |
| 54 | 2 | `short` | `suite` | horizontal grid coordinate, legal 0..127; **−1 = deleted** |
| 56 | 2 | `short` | `openings` | **dead field**, always 0 |
| 58 | 2 | `short` | `numObjects` | count of non-empty slots in `objects[]` |
| 60 | 288 | `objectType[24]` | `objects[0..23]` | object `k` at rel `60 + 12*k` |
| 348 | | | end | |

`floor` is at rel 52 and `suite` at rel 54 — the declaration order in
`short floor, suite;` (`GliderPRO/Headers/GliderStructs.h:176`). Empirically
pinned in Part 12.3: `Sampler` room[0] has (rel52, rel54) = (1, 67) and room[1]
has (1, 66); room[0]'s `kMailboxLf` carries `where` = 6609, which
`ExtractFloorSuite` decodes as suite = 6609/100 = **66**, floor = 6609%100 − 8 =
**1** — matching room[1] only if `suite` is the rel-54 field. Additionally the
corpus-wide range of rel 52 is −7..39 (inside the legal floor range −7..56) while
rel 54 spans 0..127 (the full legal suite range 0..127); swapping them would put
`floor` at 127, which validation would reject.

## 4.2 `bounds` — the wall-opening bitfield

`bounds` is only meaningful for **user-supplied backgrounds** (PICT ID in
`[3000, 3800)`). It is written by the room-info dialog
(`GliderPRO/Sources/RoomInfo.c:762-780`) and read back at
`GliderPRO/Sources/RoomInfo.c:744-749`:

```c
tempShort = thisRoom->bounds >> 1;              // RoomInfo.c:744  version 2.0 house
originalLeftOpen   = ((tempShort & 1) == 1);    // RoomInfo.c:745
originalTopOpen    = ((tempShort & 2) == 2);    // RoomInfo.c:746
originalRightOpen  = ((tempShort & 4) == 4);    // RoomInfo.c:747
originalBottomOpen = ((tempShort & 8) == 8);    // RoomInfo.c:748
originalFloor      = ((tempShort & 16) == 16);  // RoomInfo.c:749
```

and the writer:

```c
if ((longID >= 3000) && (longID < 3800) && (PictIDExists((short)longID)))  // RoomInfo.c:762
{	tempShort = 0;                              // RoomInfo.c:767
	if (originalLeftOpen)   tempShort += 1;      // RoomInfo.c:768-769
	if (originalTopOpen)    tempShort += 2;      // RoomInfo.c:770-771
	if (originalRightOpen)  tempShort += 4;      // RoomInfo.c:772-773
	if (originalBottomOpen) tempShort += 8;      // RoomInfo.c:774-775
	if (originalFloor)      tempShort += 16;     // RoomInfo.c:776-777
	tempShort = tempShort << 1;                  // RoomInfo.c:778  shift left 1 bit
	tempShort += 1;                              // RoomInfo.c:779  flag that says orginal bounds used
	thisRoom->bounds = tempShort;                // RoomInfo.c:780
}
```

So the encoding is:

| Bit of `bounds` | Mask | Meaning |
|---|---|---|
| 0 | `0x0001` | **"original bounds used"** marker. When 0 the whole field is meaningless and the code falls back to a `'bnds'` resource. |
| 1 | `0x0002` | left wall open (`boundsCode` bit 0) |
| 2 | `0x0004` | top (ceiling) open (`boundsCode` bit 1) |
| 3 | `0x0008` | right wall open (`boundsCode` bit 2) |
| 4 | `0x0010` | bottom open, i.e. **no floor** (`boundsCode` bit 3) |
| 5 | `0x0020` | room has floor support / **is a structure** (`boundsCode` bit 4) |
| 6..15 | | unused |

Equivalently: `boundCode = bounds >> 1`, and `bounds != 0` iff the field is
authoritative. Note `bounds & 32` is tested directly (without shifting) in
`IsRoomAStructure`:

```c
if ((thisRoom->background >= kUserBackground) && (thisRoom->bounds != 0))   // Room.c:~770
	isStructure = ((thisRoom->bounds & 32) == 32);
```

which reads bit 5 of the *unshifted* field — consistent with bit 4 of
`boundCode` because of the `<< 1`.

**Empirical confirmation of the marker bit:** across 4070 rooms in 22 houses,
`bounds` takes 32 distinct values and **every single non-zero value is odd**:
1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23, 25, 27, 29, 31, 33, 35, 37, 39, 41,
43, 45, 47, 49, 53, 55, 57, 59, 61, 63. 2401 rooms have `bounds` = 0. The two
most common non-zero values are 31 (542 rooms — left+top+right+bottom open, no
floor support) and 1 (167 rooms — marker only, everything closed). No even
non-zero value ever occurs, exactly as the `<< 1; += 1` writer guarantees.

### 4.2.1 Fallback to the `'bnds'` resource

When `bounds == 0`, `DetermineRoomOpenings` calls
`GetOriginalBounding(theID)` (`GliderPRO/Sources/Room.c:937-966`):

```c
boundsRes = (boundsHand)GetResource('bnds', theID);   // Room.c:942
...
boundCode = 0;
if ((*boundsRes)->left)   boundCode += 1;
if ((*boundsRes)->top)    boundCode += 2;
if ((*boundsRes)->right)  boundCode += 4;
if ((*boundsRes)->bottom) boundCode += 8;
```

The resource is a raw 4-byte record (`GliderPRO/Headers/GliderStructs.h:266-272`):

```c
typedef struct { Boolean left; Boolean top; Boolean right; Boolean bottom; } boundsType;
```

byte 0 = left, byte 1 = top, byte 2 = right, byte 3 = bottom. Note there is **no
"has floor support" byte** — a `'bnds'`-driven room can never be a structure by
this route; `IsRoomAStructure` instead falls back to
`background < kUserStructureRange` when `bounds == 0`
(`GliderPRO/Sources/Room.c:763-812`). If the resource is missing,
`boundCode = 0` and a `kYellowNoBoundsRes` alert fires when the PICT does exist.

**Observed `'bnds'` resources in `Demo House` (exactly 4 bytes each):**

| ID | Bytes | left | top | right | bottom | `boundCode` |
|---|---|---|---|---|---|---|
| 3000 | `00 00 01 00` | 0 | 0 | 1 | 0 | 4 |
| 3001 | `01 00 01 00` | 1 | 0 | 1 | 0 | 5 |
| 3002 | `01 00 01 00` | 1 | 0 | 1 | 0 | 5 |
| 3003 | `01 00 01 00` | 1 | 0 | 1 | 0 | 5 |
| 3004 | `01 00 00 00` | 1 | 0 | 0 | 0 | 1 |
| 3005 | `00 00 00 00` | 0 | 0 | 0 | 0 | 0 |
| 3300 | `00 01 01 01` | 0 | 1 | 1 | 1 | 14 |
| 3301 | `00 01 01 00` | 0 | 1 | 1 | 0 | 6 |
| 3302 | `01 01 01 00` | 1 | 1 | 1 | 0 | 7 |
| 3303 | `01 01 01 00` | 1 | 1 | 1 | 0 | 7 |

`Demo House` room[0] uses `background` = 3000 with `bounds` = 0, so its openings
come from `'bnds'` 3000 → right wall open only — consistent with room[0] sitting
at suite 63 and room[1] at suite 64 (its east neighbour).

Across the whole corpus there is **no** case of `bounds == 0` with a user
background lacking its `'bnds'` resource (verified, Part 12.7), so the
missing-resource path never triggers on shipped content.

### 4.2.2 Built-in-background opening rules

For built-in backgrounds (2000–2017) the `bounds` field is **ignored entirely**;
openings are derived from `tiles[0]` and `tiles[7]`
(`GliderPRO/Sources/Room.c:816-933`) plus a set of pixel thresholds
(`GliderPRO/Headers/GliderDefines.h:503-511`):

| Constant | Value |
|---|---|
| `kCeilingLimit` | 8 |
| `kFloorLimit` | 312 |
| `kRoofLimit` | 122 |
| `kLeftWallLimit` | 12 |
| `kNoLeftWallLimit` | −24 |
| `kRightWallLimit` | 500 |
| `kNoRightWallLimit` | 536 |
| `kNoCeilingLimit` | −10 |
| `kNoFloorLimit` | 332 |

For user art, `boundsCode` drives the same thresholds:
`leftOpen = ((boundsCode & 0x0001) == 0x0001)`,
`rightOpen = ((boundsCode & 0x0004) == 0x0004)`, and
`DoesRoomHaveFloor` / `DoesRoomHaveCeiling` test
`(boundsCode & 0x0008) != 0x0008` (`GliderPRO/Sources/Room.c:1138`) and
`(boundsCode & 0x0002) != 0x0002` (`GliderPRO/Sources/Room.c:1172`) respectively.

**Stale-`bounds` warning.** 27 rooms in the corpus have a non-zero `bounds` on a
*built-in* background (`CD Demo House` 23, `Grand Prix` 2, `Land of Illusion` 1,
`Teddy World` 1). Because `IsRoomAStructure` tests
`background >= kUserBackground` first, and `DetermineRoomOpenings` likewise only
consults `boundsCode` for user art, those values are dead residue. A port must
apply the same ordering, i.e. **check the background range before the `bounds`
field**, or those 27 rooms will render and collide differently.

## 4.3 `leftStart` / `rightStart`

Unsigned bytes (0..255) giving the vertical offset, below `kGliderStartsDown`
= 32 (`GliderPRO/Headers/GliderDefines.h:569`), at which the glider appears when
it enters the room horizontally. From `GliderPRO/Sources/Transit.c:174-176` and
`206-208`:

```c
// entering from the east (moving left into this room):
QSetRect(&enterRect, 0, 0, 48, 20);
QOffsetRect(&enterRect, 0, kGliderStartsDown + (short)thisRoom->leftStart - 2);
// entering from the west (moving right into this room):
QOffsetRect(&enterRect, kRoomWide - 48, kGliderStartsDown + (short)thisRoom->rightStart - 2);
```

So `leftStart` positions the glider at the **left** edge (it arrived from the
east) and `rightStart` at the **right** edge. The editor moves them with
`increment = thisRoom->leftStart + deltaV;` clamped to `[0, 255]` and stored back
as a `Byte` (`GliderPRO/Sources/ObjectEdit.c:529-560`). Defaults on room creation
are 32 / 32 (`GliderPRO/Sources/Room.c:170-171`).

Observed: both fields span the full 0..255 across the corpus, so a port must
treat them as **unsigned** — reading them as `int8` will put gliders above the
ceiling.

## 4.4 `unusedByte`, `visited`, `openings`

- **`unusedByte`** (rel 32) is force-zeroed by validation:
  `(*thisHouse)->rooms[i].unusedByte = 0;`
  (`GliderPRO/Sources/HouseLegal.c:868`), and initialized to 0 on room creation
  (`GliderPRO/Sources/Room.c:173`). Observed: **0 in all 4070 corpus rooms.**
- **`visited`** (rel 33) is set at runtime as the player enters rooms, feeds
  `CountRoomsVisited()` (`GliderPRO/Sources/House.c:506-531`), and is **cleared
  for every room** at the start of play:
  `thisHousePtr->rooms[r].visited = false;` (`GliderPRO/Sources/Play.c:619`).
  Whatever value is on disk is therefore residue from the last save. Observed:
  0 or 1 only; 436 of 4070 rooms have 1 (e.g. `Demo House` rooms 0 and 1).
- **`openings`** (rel 56) is written once, to 0, on room creation
  (`GliderPRO/Sources/Room.c:179`), and **never read anywhere in the code base**.
  Observed: **0 in all 4070 corpus rooms.** It is a dead field; the real opening
  information lives in `bounds` / `'bnds'` / the tile heuristics. A port should
  write 0 and ignore it.

## 4.5 `background` — the room's PICT ID

| Constant | Value | Meaning | Citation |
|---|---|---|---|
| `kBaseBackgroundID` | 2000 | first built-in background | `GliderPRO/Headers/GliderDefines.h:519` |
| `kNumBackgrounds` | 18 | count of built-ins (2000..2017) | `GliderPRO/Headers/GliderDefines.h:521` |
| `kFirstOutdoorBack` | 2009 | 2009..2017 are outdoor | `GliderPRO/Headers/GliderDefines.h:520` |
| `kUserBackground` | 3000 | first user-art background | `GliderPRO/Headers/GliderDefines.h:522` |
| `kUserStructureRange` | 3300 | user art ≥ 3300 defaults to "is a structure" | `GliderPRO/Headers/GliderDefines.h:523` |
| (implicit) | 3800 | exclusive upper bound for user art | `GliderPRO/Sources/RoomInfo.c:762` |

The 18 built-ins (`GliderPRO/Headers/GliderDefines.h:227-244`):

| ID | Constant | Indoor/outdoor | Structure? |
|---|---|---|---|
| 2000 | `kSimpleRoom` | indoor | yes |
| 2001 | `kPaneledRoom` | indoor | yes |
| 2002 | `kBasement` | indoor | no |
| 2003 | `kChildsRoom` | indoor | yes |
| 2004 | `kAsianRoom` | indoor | yes |
| 2005 | `kUnfinishedRoom` | indoor | yes |
| 2006 | `kSwingersRoom` | indoor | yes |
| 2007 | `kBathroom` | indoor | yes |
| 2008 | `kLibrary` | indoor | yes |
| 2009 | `kGarden` | outdoor | no |
| 2010 | `kSkywalk` | outdoor | yes |
| 2011 | `kDirt` | outdoor | no |
| 2012 | `kMeadow` | outdoor | no |
| 2013 | `kField` | outdoor | no |
| 2014 | `kRoof` | outdoor | yes |
| 2015 | `kSky` | outdoor | no |
| 2016 | `kStratosphere` | outdoor | no |
| 2017 | `kStars` | outdoor | no |

("Structure?" from `IsRoomAStructure`, `GliderPRO/Sources/Room.c:763-812`: the
explicit list is kPaneledRoom, kSimpleRoom, kChildsRoom, kAsianRoom,
kUnfinishedRoom, kSwingersRoom, kBathroom, kLibrary, kSkywalk, kRoof.)

Lookup is chain-wide with a fallback type:

```c
thePicture = GetPicture(theID);                           // Room.c:269
if (thePicture == nil)
	thePicture = (PicHandle)GetResource('Date', theID);   // Room.c:272
```

The same pattern appears at `GliderPRO/Sources/RoomGraphics.c:139-145` (which
additionally falls back to `GetPicture(2000)`),
`GliderPRO/Sources/Map.c:169-172`, and `GliderPRO/Sources/RoomInfo.c:842-845`.
`'Date'` is an alternate resource type carrying a PICT payload — presumably an
art-protection trick. **No shipped house uses it** (Part 12.7 type histogram), so
a port may treat it as a no-op fallback but should still recognize the type.

Observed background IDs in use across the corpus: all 18 built-ins 2000–2017,
plus user IDs 3000–3053, 3100, 3102–3104, 3207–3236, 3300–3312, 3314–3327, and
3352–3353. Every one resolves to a `PICT` resource in the owning house (verified,
Part 12.7 — zero dangling background references).

## 4.6 `tiles[8]` — the background tiling

| Constant | Value | Citation |
|---|---|---|
| `kNumTiles` | 8 | `GliderPRO/Headers/GliderDefines.h:496` |
| `kTileWide` | 64 | `GliderPRO/Headers/GliderDefines.h:497` |
| `kTileHigh` | 322 | `GliderPRO/Headers/GliderDefines.h:498` |
| `kRoomWide` | 512 | `GliderPRO/Headers/GliderDefines.h:499` |

A room is 8 tiles × 64 px = 512 px wide and 322 px tall. **`tiles[i]` is a column
index into the background PICT**, not a pixel offset:

```c
QSetRect(&src, 0, 0, kTileWide, kTileHigh);
...
src.left = theTiles[i] * kTileWide;        // Room.c:291
src.right = src.left + kTileWide;          // Room.c:292
CopyBits(... &src, &dest, srcCopy, nil);
QOffsetRect(&dest, kTileWide, 0);
```

(Also at `GliderPRO/Sources/RoomGraphics.c:246-251`.) So the background PICT is a
horizontal strip of `k` tiles, each 64 × 322, and the room selects 8 of them (with
repetition) left to right. There is **no bounds check** on `tiles[i]`: a value
larger than the number of tiles in the PICT reads off the right edge of the
GWorld.

Defaults are per-background (`SetInitialTiles`, `GliderPRO/Sources/Room.c:40-153`):

| Background | Default `tiles[0..7]` |
|---|---|
| ≥ `kUserBackground` (3000+) | `0,1,2,3,4,5,6,7` (identity) |
| `kSimpleRoom`, `kPaneledRoom`, `kBasement`, `kChildsRoom`, `kAsianRoom`, `kUnfinishedRoom`, `kSwingersRoom`, `kBathroom`, `kLibrary` | all 1, then `tiles[0] = 0`, `tiles[7] = 7` |
| `kSkywalk` | `0,1,2,3,4,5,6,7` |
| `kField`, `kGarden`, `kDirt` | all 0 |
| `kMeadow` | all 1 |
| `kRoof` | all 3 |
| `kSky` | all 2 |
| `kStratosphere`, `kStars` | `0,1,2,3,4,5,6,7` |

Observed tile-index distribution over all 4070 × 8 = 32560 slots:
0 → 5114, 1 → 7036, 2 → 6395, 3 → 3229, 4 → 3925, 5 → 2263, 6 → 2161, 7 → 2437.
**No value outside 0..7 occurs anywhere in the corpus**, so all shipped background
PICTs are exactly 8 tiles (512 px) wide.

The opening heuristics for built-in backgrounds key off `tiles[0]` (`leftTile`)
and `tiles[7]` (`rightTile`) — for example `kDirt` compares `leftTile` to 1 and
`kMeadow` compares `leftTile` to 6 and `rightTile` to 7
(`GliderPRO/Sources/Room.c:816-933`).

Filler tiles for **non-existent** neighbour rooms are synthesized, not read from
the file (`GliderPRO/Sources/RoomGraphics.c:209-226`):

| `elevation` | PICT | tiles |
|---|---|---|
| > 1 | `kSky` (2015) | all 2 |
| == 1 | `kMeadow` (2012) | all 0 |
| ≤ 0 | `kDirt` (2011) | all 0 |

unless `wardBitSet` (flags bit 0), in which case the area is just painted flat
(`GliderPRO/Sources/RoomGraphics.c:195-206`).

## 4.7 `numObjects`

The number of `objects[]` slots whose `what` is not `kObjectIsEmpty`. It is
**recomputed and overwritten** by validation, so it is advisory:

```c
void MakeSureNumObjectsJives (void)      // HouseLegal.c:~890
{
	...
	for (i = 0; i < kMaxRoomObs; i++)
		if ((*thisHouse)->rooms[r].objects[i].what != kObjectIsEmpty)
			count++;                      // HouseLegal.c:899-907
	if (count != (*thisHouse)->rooms[r].numObjects)
		(*thisHouse)->rooms[r].numObjects = count;
	...
}
```

Observed: `numObjects` ranges 0..24 and matches the live-slot count in every
shipped room checked (`Demo House` room[0] says 10 and has 10; room[1] says 18
and has 18; `Sampler` room[0] says 7 / has 7, room[1] says 4 / has 4).

Note the slots are **not** compacted: live objects can be interleaved with empty
ones, and object slot indices are load-bearing because links reference objects by
slot number (Part 7). A port must never renumber slots.
---

# Part 5 — `objectType`: the 12-byte object record

Definition, verbatim (`GliderPRO/Headers/GliderStructs.h:90-105`):

```c
typedef struct
{
	short			what;					// 2
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

Object `k` of room `r` starts at absolute file offset
`866 + 348*r + 60 + 12*k`.

| rel | Size | Type | Field |
|---|---|---|---|
| 0 | 2 | `short` | `what` — the object code, or `kObjectIsEmpty` (−1) |
| 2 | 10 | union | `data` — interpretation selected by `what` |

**Every one of the nine union variants is exactly 10 bytes with no padding**, so
the union is exactly 10 bytes and the record exactly 12. This is only true because
each variant's largest member is a `short` (2 bytes) and every `short`/`Point`/
`Rect` member sits at an even offset within the variant. Verified against the
author's own size comments on each `typedef`
(`GliderPRO/Headers/GliderStructs.h:19, 25, 34, 43, 52, 62, 72, 82, 88`), each of
which reads `// 10`.

`kObjectIsEmpty` = −1 = `0xFFFF` (`GliderPRO/Headers/GliderDefines.h:526`). An
empty slot's remaining 10 bytes are **not cleared** — `CreateNewRoom` only sets
`what` (`GliderPRO/Sources/Room.c:181-182`) — so they are residue.

## 5.1 The `what` → variant mapping

The authoritative mapping is the `switch (theObject->what)` in `KeepObjectLegal`
(`GliderPRO/Sources/HouseLegal.c`), which is the only place every code is
enumerated against its union member:

| `what` range | Group | Union member | Struct | Cases at |
|---|---|---|---|---|
| `0x01`–`0x10` | blowers / air | `data.a` | `blowerType` | `GliderPRO/Sources/HouseLegal.c:75-90` |
| `0x11`–`0x1F` | furniture / obstacles | `data.b` | `furnitureType` | `GliderPRO/Sources/HouseLegal.c:196-210` |
| `0x21`–`0x2F` | prizes / bonuses | `data.c` | `bonusType` | `GliderPRO/Sources/HouseLegal.c:229-243` |
| `0x31`–`0x40` | transport / doors / windows | `data.d` | `transportType` | `GliderPRO/Sources/HouseLegal.c:282-297` |
| `0x41`–`0x49` | switches / triggers | `data.e` | `switchType` | `GliderPRO/Sources/HouseLegal.c:387-395` |
| `0x51`–`0x58` | lights | `data.f` | `lightType` | `GliderPRO/Sources/HouseLegal.c:410-417` |
| `0x61`–`0x6E` | appliances | `data.g` | `applianceType` | `GliderPRO/Sources/HouseLegal.c:467-480` |
| `0x71`–`0x79` | enemies | `data.h` | `enemyType` | `GliderPRO/Sources/HouseLegal.c:514-522` |
| `0x81`–`0x8F` | clutter / decoration | `data.i` | `clutterType` | `GliderPRO/Sources/HouseLegal.c:556-570` |

`kNumSrcRects` = `0x90` = 144 (`GliderPRO/Headers/GliderDefines.h:437`) is the size
of the global `srcRects[]` array, indexed directly by `what`, so `what` is
effectively constrained to `0x00`–`0x8F`. Note the **gaps** at `0x00`, `0x20`,
`0x30`, `0x4A`–`0x50`, `0x59`–`0x60`, `0x6F`–`0x70`, `0x7A`–`0x80`: those codes are
undefined, fall through every `switch` default, and index unset `srcRects[]`
entries. A port should treat them as invalid.

**Do not derive the variant from the high nibble alone.** `0x10` (`kLiftArea`) is
a blower but has high nibble 1; `0x40` (`kDeluxeTrans`) is a transport but has high
nibble 4. The correct test is the inclusive range in the table above.

## 5.2 The complete `what` enumeration

All 117 defined codes, from `GliderPRO/Headers/GliderDefines.h:311-435`. The
"observed" column is the number of instances across all 4070 rooms of the 22
shipped houses (Part 12.6). **Every defined code occurs at least once**, and no
undefined code occurs anywhere.

### Blowers / air currents — `data.a` (`blowerType`)

| `what` | Constant | Line | Observed |
|---|---|---|---|
| `0x01` | `kFloorVent` | `GliderPRO/Headers/GliderDefines.h:311` | 1458 |
| `0x02` | `kCeilingVent` | `GliderPRO/Headers/GliderDefines.h:312` | 28 |
| `0x03` | `kFloorBlower` | `GliderPRO/Headers/GliderDefines.h:313` | 83 |
| `0x04` | `kCeilingBlower` | `GliderPRO/Headers/GliderDefines.h:314` | 12 |
| `0x05` | `kSewerGrate` | `GliderPRO/Headers/GliderDefines.h:315` | 507 |
| `0x06` | `kLeftFan` | `GliderPRO/Headers/GliderDefines.h:316` | 45 |
| `0x07` | `kRightFan` | `GliderPRO/Headers/GliderDefines.h:317` | 54 |
| `0x08` | `kTaper` | `GliderPRO/Headers/GliderDefines.h:318` | 90 |
| `0x09` | `kCandle` | `GliderPRO/Headers/GliderDefines.h:319` | 180 |
| `0x0A` | `kStubby` | `GliderPRO/Headers/GliderDefines.h:320` | 127 |
| `0x0B` | `kTiki` | `GliderPRO/Headers/GliderDefines.h:321` | 58 |
| `0x0C` | `kBBQ` | `GliderPRO/Headers/GliderDefines.h:322` | 43 |
| `0x0D` | `kInvisBlower` | `GliderPRO/Headers/GliderDefines.h:323` | 2336 |
| `0x0E` | `kGrecoVent` | `GliderPRO/Headers/GliderDefines.h:324` | 116 |
| `0x0F` | `kSewerBlower` | `GliderPRO/Headers/GliderDefines.h:325` | 191 |
| `0x10` | `kLiftArea` | `GliderPRO/Headers/GliderDefines.h:326` | 716 |

### Furniture / obstacles — `data.b` (`furnitureType`)

| `what` | Constant | Line | Observed |
|---|---|---|---|
| `0x11` | `kTable` | `GliderPRO/Headers/GliderDefines.h:328` | 170 |
| `0x12` | `kShelf` | `GliderPRO/Headers/GliderDefines.h:329` | 389 |
| `0x13` | `kCabinet` | `GliderPRO/Headers/GliderDefines.h:330` | 457 |
| `0x14` | `kFilingCabinet` | `GliderPRO/Headers/GliderDefines.h:331` | 107 |
| `0x15` | `kWasteBasket` | `GliderPRO/Headers/GliderDefines.h:332` | 103 |
| `0x16` | `kMilkCrate` | `GliderPRO/Headers/GliderDefines.h:333` | 252 |
| `0x17` | `kCounter` | `GliderPRO/Headers/GliderDefines.h:334` | 287 |
| `0x18` | `kDresser` | `GliderPRO/Headers/GliderDefines.h:335` | 122 |
| `0x19` | `kDeckTable` | `GliderPRO/Headers/GliderDefines.h:336` | 30 |
| `0x1A` | `kStool` | `GliderPRO/Headers/GliderDefines.h:337` | 91 |
| `0x1B` | `kTrunk` | `GliderPRO/Headers/GliderDefines.h:338` | 101 |
| `0x1C` | `kInvisObstacle` | `GliderPRO/Headers/GliderDefines.h:339` | 666 |
| `0x1D` | `kManhole` | `GliderPRO/Headers/GliderDefines.h:340` | 40 |
| `0x1E` | `kBooks` | `GliderPRO/Headers/GliderDefines.h:341` | 210 |
| `0x1F` | `kInvisBounce` | `GliderPRO/Headers/GliderDefines.h:342` | 1670 |

### Prizes / bonuses — `data.c` (`bonusType`)

| `what` | Constant | Line | Observed |
|---|---|---|---|
| `0x21` | `kRedClock` | `GliderPRO/Headers/GliderDefines.h:344` | 140 |
| `0x22` | `kBlueClock` | `GliderPRO/Headers/GliderDefines.h:345` | 163 |
| `0x23` | `kYellowClock` | `GliderPRO/Headers/GliderDefines.h:346` | 330 |
| `0x24` | `kCuckoo` | `GliderPRO/Headers/GliderDefines.h:347` | 100 |
| `0x25` | `kPaper` | `GliderPRO/Headers/GliderDefines.h:348` | 283 |
| `0x26` | `kBattery` | `GliderPRO/Headers/GliderDefines.h:349` | 119 |
| `0x27` | `kBands` | `GliderPRO/Headers/GliderDefines.h:350` | 150 |
| `0x28` | `kGreaseRt` | `GliderPRO/Headers/GliderDefines.h:351` | 195 |
| `0x29` | `kGreaseLf` | `GliderPRO/Headers/GliderDefines.h:352` | 143 |
| `0x2A` | `kFoil` | `GliderPRO/Headers/GliderDefines.h:353` | 83 |
| `0x2B` | `kInvisBonus` | `GliderPRO/Headers/GliderDefines.h:354` | 327 |
| `0x2C` | `kStar` | `GliderPRO/Headers/GliderDefines.h:355` | 69 |
| `0x2D` | `kSparkle` | `GliderPRO/Headers/GliderDefines.h:356` | 486 |
| `0x2E` | `kHelium` | `GliderPRO/Headers/GliderDefines.h:357` | 50 |
| `0x2F` | `kSlider` | `GliderPRO/Headers/GliderDefines.h:358` | 354 |

### Transport / doors / windows — `data.d` (`transportType`)

| `what` | Constant | Line | Observed |
|---|---|---|---|
| `0x31` | `kUpStairs` | `GliderPRO/Headers/GliderDefines.h:360` | 163 |
| `0x32` | `kDownStairs` | `GliderPRO/Headers/GliderDefines.h:361` | 163 |
| `0x33` | `kMailboxLf` | `GliderPRO/Headers/GliderDefines.h:362` | 44 |
| `0x34` | `kMailboxRt` | `GliderPRO/Headers/GliderDefines.h:363` | 34 |
| `0x35` | `kFloorTrans` | `GliderPRO/Headers/GliderDefines.h:364` | 244 |
| `0x36` | `kCeilingTrans` | `GliderPRO/Headers/GliderDefines.h:365` | 465 |
| `0x37` | `kDoorInLf` | `GliderPRO/Headers/GliderDefines.h:366` | 23 |
| `0x38` | `kDoorInRt` | `GliderPRO/Headers/GliderDefines.h:367` | 11 |
| `0x39` | `kDoorExRt` | `GliderPRO/Headers/GliderDefines.h:368` | 21 |
| `0x3A` | `kDoorExLf` | `GliderPRO/Headers/GliderDefines.h:369` | 11 |
| `0x3B` | `kWindowInLf` | `GliderPRO/Headers/GliderDefines.h:370` | 21 |
| `0x3C` | `kWindowInRt` | `GliderPRO/Headers/GliderDefines.h:371` | 32 |
| `0x3D` | `kWindowExRt` | `GliderPRO/Headers/GliderDefines.h:372` | 19 |
| `0x3E` | `kWindowExLf` | `GliderPRO/Headers/GliderDefines.h:373` | 29 |
| `0x3F` | `kInvisTrans` | `GliderPRO/Headers/GliderDefines.h:374` | 385 |
| `0x40` | `kDeluxeTrans` | `GliderPRO/Headers/GliderDefines.h:375` | 61 |

### Switches / triggers — `data.e` (`switchType`)

| `what` | Constant | Line | Observed |
|---|---|---|---|
| `0x41` | `kLightSwitch` | `GliderPRO/Headers/GliderDefines.h:377` | 113 |
| `0x42` | `kMachineSwitch` | `GliderPRO/Headers/GliderDefines.h:378` | 79 |
| `0x43` | `kThermostat` | `GliderPRO/Headers/GliderDefines.h:379` | 108 |
| `0x44` | `kPowerSwitch` | `GliderPRO/Headers/GliderDefines.h:380` | 78 |
| `0x45` | `kKnifeSwitch` | `GliderPRO/Headers/GliderDefines.h:381` | 230 |
| `0x46` | `kInvisSwitch` | `GliderPRO/Headers/GliderDefines.h:382` | 635 |
| `0x47` | `kTrigger` | `GliderPRO/Headers/GliderDefines.h:383` | 239 |
| `0x48` | `kLgTrigger` | `GliderPRO/Headers/GliderDefines.h:384` | 81 |
| `0x49` | `kSoundTrigger` | `GliderPRO/Headers/GliderDefines.h:385` | 122 |

### Lights — `data.f` (`lightType`)

| `what` | Constant | Line | Observed |
|---|---|---|---|
| `0x51` | `kCeilingLight` | `GliderPRO/Headers/GliderDefines.h:387` | 160 |
| `0x52` | `kLightBulb` | `GliderPRO/Headers/GliderDefines.h:388` | 252 |
| `0x53` | `kTableLamp` | `GliderPRO/Headers/GliderDefines.h:389` | 70 |
| `0x54` | `kHipLamp` | `GliderPRO/Headers/GliderDefines.h:390` | 26 |
| `0x55` | `kDecoLamp` | `GliderPRO/Headers/GliderDefines.h:391` | 62 |
| `0x56` | `kFlourescent` | `GliderPRO/Headers/GliderDefines.h:392` | 115 |
| `0x57` | `kTrackLight` | `GliderPRO/Headers/GliderDefines.h:393` | 81 |
| `0x58` | `kInvisLight` | `GliderPRO/Headers/GliderDefines.h:394` | 1764 |

### Appliances — `data.g` (`applianceType`)

| `what` | Constant | Line | Observed |
|---|---|---|---|
| `0x61` | `kShredder` | `GliderPRO/Headers/GliderDefines.h:396` | 50 |
| `0x62` | `kToaster` | `GliderPRO/Headers/GliderDefines.h:397` | 140 |
| `0x63` | `kMacPlus` | `GliderPRO/Headers/GliderDefines.h:398` | 74 |
| `0x64` | `kGuitar` | `GliderPRO/Headers/GliderDefines.h:399` | 29 |
| `0x65` | `kTV` | `GliderPRO/Headers/GliderDefines.h:400` | 80 |
| `0x66` | `kCoffee` | `GliderPRO/Headers/GliderDefines.h:401` | 38 |
| `0x67` | `kOutlet` | `GliderPRO/Headers/GliderDefines.h:402` | 100 |
| `0x68` | `kVCR` | `GliderPRO/Headers/GliderDefines.h:403` | 26 |
| `0x69` | `kStereo` | `GliderPRO/Headers/GliderDefines.h:404` | 36 |
| `0x6A` | `kMicrowave` | `GliderPRO/Headers/GliderDefines.h:405` | 56 |
| `0x6B` | `kCinderBlock` | `GliderPRO/Headers/GliderDefines.h:406` | 43 |
| `0x6C` | `kFlowerBox` | `GliderPRO/Headers/GliderDefines.h:407` | 35 |
| `0x6D` | `kCDs` | `GliderPRO/Headers/GliderDefines.h:408` | 63 |
| `0x6E` | `kCustomPict` | `GliderPRO/Headers/GliderDefines.h:409` | 4782 |

### Enemies — `data.h` (`enemyType`)

| `what` | Constant | Line | Observed |
|---|---|---|---|
| `0x71` | `kBalloon` | `GliderPRO/Headers/GliderDefines.h:411` | 500 |
| `0x72` | `kCopterLf` | `GliderPRO/Headers/GliderDefines.h:412` | 259 |
| `0x73` | `kCopterRt` | `GliderPRO/Headers/GliderDefines.h:413` | 155 |
| `0x74` | `kDartLf` | `GliderPRO/Headers/GliderDefines.h:414` | 151 |
| `0x75` | `kDartRt` | `GliderPRO/Headers/GliderDefines.h:415` | 117 |
| `0x76` | `kBall` | `GliderPRO/Headers/GliderDefines.h:416` | 212 |
| `0x77` | `kDrip` | `GliderPRO/Headers/GliderDefines.h:417` | 477 |
| `0x78` | `kFish` | `GliderPRO/Headers/GliderDefines.h:418` | 120 |
| `0x79` | `kCobweb` | `GliderPRO/Headers/GliderDefines.h:419` | 86 |

### Clutter / decoration — `data.i` (`clutterType`)

| `what` | Constant | Line | Observed |
|---|---|---|---|
| `0x81` | `kOzma` | `GliderPRO/Headers/GliderDefines.h:421` | 77 |
| `0x82` | `kMirror` | `GliderPRO/Headers/GliderDefines.h:422` | 667 |
| `0x83` | `kMousehole` | `GliderPRO/Headers/GliderDefines.h:423` | 174 |
| `0x84` | `kFireplace` | `GliderPRO/Headers/GliderDefines.h:424` | 33 |
| `0x85` | `kFlower` | `GliderPRO/Headers/GliderDefines.h:425` | 547 |
| `0x86` | `kWallWindow` | `GliderPRO/Headers/GliderDefines.h:426` | 195 |
| `0x87` | `kBear` | `GliderPRO/Headers/GliderDefines.h:427` | 169 |
| `0x88` | `kCalendar` | `GliderPRO/Headers/GliderDefines.h:428` | 58 |
| `0x89` | `kVase1` | `GliderPRO/Headers/GliderDefines.h:429` | 72 |
| `0x8A` | `kVase2` | `GliderPRO/Headers/GliderDefines.h:430` | 86 |
| `0x8B` | `kBulletin` | `GliderPRO/Headers/GliderDefines.h:431` | 38 |
| `0x8C` | `kCloud` | `GliderPRO/Headers/GliderDefines.h:432` | 1860 |
| `0x8D` | `kFaucet` | `GliderPRO/Headers/GliderDefines.h:433` | 28 |
| `0x8E` | `kRug` | `GliderPRO/Headers/GliderDefines.h:434` | 81 |
| `0x8F` | `kChimes` | `GliderPRO/Headers/GliderDefines.h:435` | 54 |

## 5.3 Variant `a` — `blowerType` (10 bytes)

`GliderPRO/Headers/GliderStructs.h:11-19`:

```c
typedef struct
{
	Point		topLeft;					// 4
	short		distance;					// 2
	Boolean		initial;					// 1
	Boolean		state;						// 1
	Byte		vector;						// 1
	//			F.  lf. dn. rt. up
	//	| x | x | x | x | 8 | 4 | 2 | 1 |
	Byte		tall;						// 1
} blowerType;								// 10
```

| rel to object | rel to `data` | Size | Type | Field |
|---|---|---|---|---|
| 2 | 0 | 2 | `short` | `topLeft.v` |
| 4 | 2 | 2 | `short` | `topLeft.h` |
| 6 | 4 | 2 | `short` | `distance` |
| 8 | 6 | 1 | `Boolean` | `initial` |
| 9 | 7 | 1 | `Boolean` | `state` |
| 10 | 8 | 1 | `Byte` | `vector` |
| 11 | 9 | 1 | `Byte` | `tall` |

- **`topLeft`** places the object's `srcRects[what]` rectangle
  (`GliderPRO/Sources/ObjectRects.c:43-61`).
- **`distance`** is the reach of the air column in pixels. For `kGrecoVent` and
  `kSewerBlower` it is applied *upward* as `-data.a.distance`
  (`GliderPRO/Sources/ObjectRects.c:566-576`, `578-588`).
- **`initial`** is the authored on/off state; **`state`** is the runtime copy.
  `SetObjectsToDefaults` does `data.a.state = data.a.initial`
  (`GliderPRO/Sources/Play.c:635-636`), so the on-disk `state` byte is stale.
- **`vector`** is a 4-bit direction mask, read as `data.a.vector & 0x0F`:

  | Value | Direction | Action |
  |---|---|---|
  | 1 | up | `kLiftIt` |
  | 2 | right | `kPushItRight` |
  | 4 | down | `kDropIt` |
  | 8 | left | `kPushItLeft` |

  (`GliderPRO/Sources/ObjectRects.c:522-564` for `kInvisBlower` and `590-617` for
  `kLiftArea`; the same switch appears in the editor at
  `GliderPRO/Sources/ObjectEdit.c:1834`.) Only `kInvisBlower` (`0x0D`) and
  `kLiftArea` (`0x10`) consult `vector`; the other 14 blowers have hard-coded
  directions. **`vector` is not the same enum as the `kAbove`/`kToRight`/
  `kBelow`/`kToLeft` = 1/2/3/4 direction codes**
  (`GliderPRO/Headers/GliderDefines.h:210-213`) used by `ObjectHasHandle`.
  Confusing them turns "down" into "right".
- **`tall`** is the column thickness. It is only meaningful for `kLiftArea`, where
  the object's rect is built from `distance` and `tall` and **not** from
  `srcRects`:

  ```c
  case kLiftArea:
  	QSetRect(itsRect, 0, 0, who->data.a.distance, who->data.a.tall * 2);   // ObjectRects.c:64
  ```

  i.e. width = `distance` px, height = `tall * 2` px. (Lines
  `GliderPRO/Sources/ObjectRects.c:68-71` after this are unreachable dead code.)

## 5.4 Variant `b` — `furnitureType` (10 bytes)

`GliderPRO/Headers/GliderStructs.h:21-25`:

```c
typedef struct
{
	Rect		bounds;						// 8
	short		pict;						// 2
} furnitureType;							// 10
```

| rel to object | Size | Type | Field |
|---|---|---|---|
| 2 | 2 | `short` | `bounds.top` |
| 4 | 2 | `short` | `bounds.left` |
| 6 | 2 | `short` | `bounds.bottom` |
| 8 | 2 | `short` | `bounds.right` |
| 10 | 2 | `short` | `pict` |

`GetObjectRect` returns `who->data.b.bounds` verbatim
(`GliderPRO/Sources/ObjectRects.c:88`) — furniture is the only group besides
clutter that stores an explicit rectangle rather than a top-left plus a source
rect. `pict` is **unused for furniture** in 1.0.4 (observed 0 in every corpus
instance sampled); it exists because `furnitureType` and `clutterType` are
structurally identical.

Validation forces specific thicknesses (`GliderPRO/Sources/HouseLegal.c:214-223`)
using `kTableThick` = 8 (`GliderPRO/Headers/GliderDefines.h:439`) and
`kShelfThick` = 6 (`GliderPRO/Headers/GliderDefines.h:440`).

## 5.5 Variant `c` — `bonusType` (10 bytes)

`GliderPRO/Headers/GliderStructs.h:27-34`:

```c
typedef struct
{
	Point		topLeft;					// 4
	short		length;						// 2
	short		points;						// 2
	Boolean		state;						// 1
	Boolean		initial;					// 1
} bonusType;								// 10
```

| rel to object | Size | Type | Field |
|---|---|---|---|
| 2 | 2 | `short` | `topLeft.v` |
| 4 | 2 | `short` | `topLeft.h` |
| 6 | 2 | `short` | `length` |
| 8 | 2 | `short` | `points` |
| 10 | 1 | `Boolean` | `state` |
| 11 | 1 | `Boolean` | `initial` |

**This is the only variant where `state` precedes `initial`.** Every other
variant with both flags orders them `initial` then `state`. Getting this backwards
silently swaps authored and runtime state for all 15 prize types. Confirmed by the
declaration order above and by `SetObjectsToDefaults`, which still copies in the
same logical direction: `data.c.state = data.c.initial`
(`GliderPRO/Sources/Play.c:653-654`).

- **`length`** is the sliding/travel distance for `kGreaseRt`/`kGreaseLf` and, for
  `kSlider` (`0x2F`), the object's **width**:

  ```c
  case kSlider:
  	itsRect->right = itsRect->left + who->data.c.length;   // ObjectRects.c:118
  ```

  Default on creation is 64 (`GliderPRO/Sources/ObjectAdd.c:291`).
- **`points`** is only meaningful for `kInvisBonus` (`0x2B`); every other prize
  has a fixed value from `GliderPRO/Headers/GliderDefines.h:537-541`:

  | Constant | Value | Applies to |
  |---|---|---|
  | `kRedClockPoints` | 100 | `kRedClock` |
  | `kBlueClockPoints` | 300 | `kBlueClock` |
  | `kYellowClockPoints` | 500 | `kYellowClock` |
  | `kCuckooClockPoints` | 1000 | `kCuckoo` |
  | `kStarPoints` | 5000 | `kStar` |

  The editor restricts `kInvisBonus` `points` to exactly 100, 300, or 500
  (`GliderPRO/Sources/ObjectInfo.c:1993-2038`), and `CountTotalHousePoints` adds
  `data.c.points` only for `kInvisBonus`
  (`GliderPRO/Sources/HouseInfo.c:92`). Default 0
  (`GliderPRO/Sources/ObjectAdd.c:292`).
- **`initial`/`state`** default to `true`/`true`
  (`GliderPRO/Sources/ObjectAdd.c:293-294`).

## 5.6 Variant `d` — `transportType` (10 bytes)

`GliderPRO/Headers/GliderStructs.h:36-43`:

```c
typedef struct
{
	Point		topLeft;					// 4
	short		tall;						// 2
	short		where;						// 2
	Byte		who;						// 1
	Byte		wide;						// 1
} transportType;							// 10
```

| rel to object | Size | Type | Field |
|---|---|---|---|
| 2 | 2 | `short` | `topLeft.v` |
| 4 | 2 | `short` | `topLeft.h` |
| 6 | 2 | `short` | `tall` |
| 8 | 2 | `short` | `where` — **link destination room**, encoded (Part 7) |
| 10 | 1 | `Byte` | `who` — **link destination object slot**, 255 = none |
| 11 | 1 | `Byte` | `wide` |

Unlinked defaults are `where = -1`, `who = 255`
(`GliderPRO/Sources/ObjectAdd.c:316-319` and `GliderPRO/Sources/Link.c:333-334`).
Note the asymmetry: `where` uses **−1** as its sentinel (it is a `short`) while
`who` uses **255** (it is a `Byte`).

Field overloads by `what`:

| `what` | `tall` | `wide` |
|---|---|---|
| `kUpStairs` `0x31`, `kDownStairs` `0x32`, `kMailboxLf` `0x33`, `kMailboxRt` `0x34`, `kFloorTrans` `0x35`, `kCeilingTrans` `0x36` | 0, unused | 0, unused |
| doors/windows `0x37`–`0x3E` | clamped to `kTileHigh - topLeft.v` (`GliderPRO/Sources/HouseLegal.c:372-376`) | clamped to ≥ 0 (`GliderPRO/Sources/HouseLegal.c:380-382`) |
| `kInvisTrans` `0x3F` | rect **height** in px: `itsRect->bottom = itsRect->top + who->data.d.tall;` (`GliderPRO/Sources/ObjectRects.c:148`) | extra width in px: `itsRect->right += (short)who->data.d.wide;` (`GliderPRO/Sources/ObjectRects.c:149`) |
| `kDeluxeTrans` `0x40` | **packed** `(wide<<8) \| tall`, each in units of 4 px | low nibble = current state, high nibble = initial state |

### 5.6.1 `kDeluxeTrans` size packing

Written by `KeepObjectLegal`:

```c
theObject->data.d.tall = ((RectWide(&bounds) / 4) << 8) + (RectTall(&bounds) / 4);   // HouseLegal.c:306-307
```

Read by `GetObjectRect`:

```c
wide = (who->data.d.tall & 0xFF00) >> 8;      // ObjectRects.c:153
tall = who->data.d.tall & 0x00FF;             // ObjectRects.c:154
QSetRect(itsRect, 0, 0, wide * 4, tall * 4);  // ObjectRects.c:155
```

So actual width = `((tall >> 8) & 0xFF) * 4` and actual height =
`(tall & 0xFF) * 4`. The default on creation is `tall = 0x1010` with the comment
`// 64 x 64` (`GliderPRO/Sources/ObjectAdd.c:469`) — 0x10 = 16, ×4 = 64. Maximum
representable size is 255×4 = 1020 px in each axis.

**Sign trap.** `data.d.tall` is a *signed* `short`. Observed corpus values include
`-0x7FD2` and `-0x7FB0`, i.e. sizes whose width byte has the high bit set. In C,
`(short)0x802E & 0xFF00` promotes to `int` giving `0xFFFF802E & 0xFFFFFF00`… in
practice the 68k/PPC compiler sign-extends `data.d.tall` to `int` first, so
`0x802E` becomes `0xFFFF802E`, `& 0xFF00` yields `0x8000`, and `>> 8` on the
*negative* int yields `0xFFFFFF80` = −128 in the C `short` assignment — but the
subsequent `wide * 4` is used only as a rectangle `right`, so the object renders
zero- or negative-width. A Go port must decide deliberately; the safe faithful
reading is `wide := int(uint16(tall) >> 8)` and `tall := int(uint16(tall) & 0xFF)`,
which gives the *authoring* intent (a 128×… box). Do not write
`int(tall)>>8` in Go — Go's `>>` on a negative `int16` is arithmetic and will
produce a negative width.

### 5.6.2 `kDeluxeTrans` state nibbles

`data.d.wide` low nibble = current state, high nibble = authored initial state.
`SetObjectsToDefaults` reconciles them at game start:

```c
initState = (thisHousePtr->rooms[r].objects[i].data.d.wide & 0xF0) >> 4;   // Play.c:658
thisHousePtr->rooms[r].objects[i].data.d.wide &= 0xF0;                     // Play.c:659
thisHousePtr->rooms[r].objects[i].data.d.wide += initState;                // Play.c:660
```

The editor writes `data.d.wide = wasState << 4`
(`GliderPRO/Sources/ObjectInfo.c:2117-2145`), and the hot-spot code reads
`isOn = theObject.data.d.wide & 0x0F`
(`GliderPRO/Sources/ObjectRects.c:893`). Default on creation is `0x10`
(`// Initially on`, `GliderPRO/Sources/ObjectAdd.c:472`).

### 5.6.3 Fixed horizontal positions for doors and windows

Doors and windows snap to fixed `topLeft.h` values
(`GliderPRO/Headers/GliderDefines.h:467-494`, enforced at
`GliderPRO/Sources/HouseLegal.c:310-369` and
`GliderPRO/Sources/ObjectAdd.c:366-451`):

| `what` | Constant | `topLeft.h` | `topLeft.v` constant | value |
|---|---|---|---|---|
| `kDoorInLf` `0x37` | `kDoorInLfLeft` | 0 | `kDoorInTop` | 0 |
| `kDoorInRt` `0x38` | `kDoorInRtLeft` | 368 | `kDoorInTop` | 0 |
| `kDoorExRt` `0x39` | `kDoorExRtLeft` | 496 | `kDoorExTop` | 0 |
| `kDoorExLf` `0x3A` | `kDoorExLfLeft` | 0 | `kDoorExTop` | 0 |
| `kWindowInLf` `0x3B` | `kWindowInLfLeft` | 0 | `kWindowInTop` | 64 |
| `kWindowInRt` `0x3C` | `kWindowInRtLeft` | 492 | `kWindowInTop` | 64 |
| `kWindowExRt` `0x3D` | `kWindowExRtLeft` | 496 | `kWindowExTop` | 64 |
| `kWindowExLf` `0x3E` | `kWindowExLfLeft` | 0 | `kWindowExTop` | 64 |

`ObjectAdd` picks the Lf or Rt variant by testing
`where.h > (kRoomWide / 2)` i.e. > 256.

Other forced tops for `data.d` objects:

| Constant | Value | Applies to | Citation |
|---|---|---|---|
| `kStairsTop` | 28 | `kUpStairs`, `kDownStairs` | `GliderPRO/Headers/GliderDefines.h:474`; forced at `GliderPRO/Sources/ObjectAdd.c:311` |
| `kFloorTransTop` | 302 | `kFloorTrans` | `GliderPRO/Headers/GliderDefines.h:473`; forced at `GliderPRO/Sources/ObjectAdd.c:~344` and `GliderPRO/Sources/HouseLegal.c:133-136` |
| `kCeilingTransTop` | 6 | `kCeilingTrans` | `GliderPRO/Headers/GliderDefines.h:472`; forced at `GliderPRO/Sources/ObjectAdd.c:~358` |

## 5.7 Variant `e` — `switchType` (10 bytes)

`GliderPRO/Headers/GliderStructs.h:45-52`:

```c
typedef struct
{
	Point		topLeft;					// 4
	short		delay;						// 2
	short		where;						// 2
	Byte		who;						// 1
	Byte		type;						// 1
} switchType;								// 10
```

| rel to object | Size | Type | Field |
|---|---|---|---|
| 2 | 2 | `short` | `topLeft.v` |
| 4 | 2 | `short` | `topLeft.h` |
| 6 | 2 | `short` | `delay` |
| 8 | 2 | `short` | `where` — link destination room (or a `snd ` ID, see below) |
| 10 | 1 | `Byte` | `who` — link destination object slot, 255 = none |
| 11 | 1 | `Byte` | `type` — switch behaviour |

`type` values (`GliderPRO/Headers/GliderDefines.h:441-444`):

| Constant | Value | Meaning |
|---|---|---|
| `kToggle` | 0 | flips the target's state |
| `kForceOn` | 1 | always turns the target on |
| `kForceOff` | 2 | always turns the target off |
| `kOneShot` | 3 | fires once |

Defaults on creation (`GliderPRO/Sources/ObjectAdd.c:497-506`): `delay = 0`;
`type = kOneShot` for `kTrigger`/`kLgTrigger`, else `kToggle`; `who = 255`; and
`where = -1` **except** `kSoundTrigger`, which gets `where = 3000`.

A hot spot is created only when the link exists:
`if (theObject.data.e.where != -1)`
(`GliderPRO/Sources/ObjectRects.c:898-921`); `kTrigger`/`kLgTrigger` register the
`kTriggerIt` action and the other six register `kSwitchIt`.

### 5.7.1 `kSoundTrigger` (`0x49`) overloads `where` as a `snd ` resource ID

This is the single most surprising field overload in the format.

```c
case kSoundTrigger:
	QSetRect(&bounds, 0, 0, 48, 48);                                        // ObjectRects.c:924
	QOffsetRect(&bounds, theObject.data.e.topLeft.h, theObject.data.e.topLeft.v);
	if (LoadTriggerSound(theObject.data.e.where) == noErr)                  // ObjectRects.c:926
		hotSpotNumber = AddActiveRect(&bounds, kSoundIt, who, true, false); // ObjectRects.c:927
	break;
```

So for `kSoundTrigger` the `where` field is **not** a room link; it is the ID of a
`'snd '` resource loaded from the house's own resource fork. Consequences:

- The hot spot is 48 × 48 px, fixed, regardless of `srcRects`.
- If `LoadTriggerSound` fails (no such `'snd '` resource, or sounds disabled), the
  object is **silently inert** — no hot spot at all.
- `LoadTriggerSound` (`GliderPRO/Sources/Sound.c:265-303`) loads into the reserved
  last slot: `theSoundData[kMaxSounds - 1]` with `kMaxSounds` = 64
  (`GliderPRO/Sources/Sound.c:15`), and `LoadBufferSounds` deliberately leaves
  that slot `nil` (`GliderPRO/Sources/Sound.c:344`) and loops only
  `i < kMaxSounds - 1` (`GliderPRO/Sources/Sound.c:325`, `499`). Only **one**
  custom trigger sound can be loaded at a time — the guard
  `if ((dontLoadSounds) || (theSoundData[kMaxSounds - 1] != nil)) theErr = -1;`
  (`GliderPRO/Sources/Sound.c:271-272`) means the *first* `kSoundTrigger` in the
  room wins and every later one is inert. `ObjectAdd.c:17` even documents this
  with `#define kMaxSoundTriggers 1`.
- The played sound ID is `kTriggerSound` = 63 = `kMaxSounds - 1`
  (`GliderPRO/Headers/GliderDefines.h:118`), played on contact:
  `if (!who->stillOver) { PlayPrioritySound(kTriggerSound, kTriggerPriority); who->stillOver = true; }`
  (`GliderPRO/Sources/Interactions.c:1615-1621`).
- The `snd ` payload is used raw from offset 20: `soundDataSize = GetHandleSize(theSound) - 20L;`
  and `BlockMove((Ptr)(*theSound + 20L), ...)`
  (`GliderPRO/Sources/Sound.c:286`, `296`) — i.e. the first 20 bytes of the
  format-1 `'snd '` resource header are skipped and the rest is treated as raw
  8-bit sample data.
- Editor range check: `data.e.where` must be in **[3000, 32767]**
  (`GliderPRO/Sources/ObjectInfo.c:1237`), stored at
  `GliderPRO/Sources/ObjectInfo.c:1246`, with the dialog hinting "Sound"/"3000"
  (`GliderPRO/Sources/ObjectInfo.c:1186-1188`).
- **`kSoundTrigger` is deliberately excluded from all three link enumerators** —
  `CountHouseLinks` (`GliderPRO/Sources/House.c:279-296`),
  `GenerateLinksList` (`GliderPRO/Sources/House.c:339-366`), and
  `GenerateRetroLinks` (`GliderPRO/Sources/House.c:561-591`) all list switches
  `0x41`–`0x48` only. Likewise `GetRoomLinked` stops at `0x48`
  (`GliderPRO/Sources/Objects.c:149-156`). A port that treats `0x49`'s `where` as
  a room reference will produce phantom links and, worse, mangle the sound ID
  during any link-remapping operation.
- A separate code path exists where a `kSoundTrigger` is fired *by another
  object's* link; that path plays a fixed chord, **not** the custom sound:
  `case kSoundTrigger: PlayPrioritySound(kChordSound, kChordPriority);  // Change me`
  (`GliderPRO/Sources/Triggers.c:131-133`), with `kChordSound` = 23
  (`GliderPRO/Headers/GliderDefines.h:78`).

Observed: 122 `kSoundTrigger` objects in the corpus. Two of them have
`where` = 10000 (out of the editor's own [3000, 32767] range only in the sense
that 10000 looks like a `kCustomPict` ID — probably a mis-set field), and 13 more
reference `snd ` IDs that do not exist in the owning house's resource fork
(`In The Mirror` 2, `Teddy World` 11), making them permanently inert.

## 5.8 Variant `f` — `lightType` (10 bytes)

`GliderPRO/Headers/GliderStructs.h:54-62`:

```c
typedef struct
{
	Point		topLeft;					// 4
	short		length;						// 2
	Byte		byte0;						// 1
	Byte		byte1;						// 1
	Boolean		initial;					// 1
	Boolean		state;						// 1
} lightType;								// 10
```

| rel to object | Size | Type | Field |
|---|---|---|---|
| 2 | 2 | `short` | `topLeft.v` |
| 4 | 2 | `short` | `topLeft.h` |
| 6 | 2 | `short` | `length` |
| 8 | 1 | `Byte` | `byte0` |
| 9 | 1 | `Byte` | `byte1` |
| 10 | 1 | `Boolean` | `initial` |
| 11 | 1 | `Boolean` | `state` |

- **`length`** for `kFlourescent` (`0x56`) and `kTrackLight` (`0x57`) **overrides
  the sprite width**, i.e. it is a width in pixels, not an absolute edge. The
  derivation is a three-step idiom and the order matters:

  ```c
  case kFlourescent:
  case kTrackLight:
  	*itsRect = srcRects[who->what];               // ObjectRects.c:192
  	ZeroRectCorner(itsRect);                      // ObjectRects.c:193  -> top-left becomes (0,0)
  	itsRect->right = who->data.f.length;          // ObjectRects.c:194  -> rect is 0 .. length
  	QOffsetRect(itsRect,                          // ObjectRects.c:195-197
  			who->data.f.topLeft.h,
  			who->data.f.topLeft.v);
  ```

  Because `ZeroRectCorner` runs **before** the assignment, the final rect is
  `left = topLeft.h`, `right = topLeft.h + length`, `top = topLeft.v`,
  `bottom = topLeft.v + spriteHeight`. So `length` is the fixture's **width**.
  (`GliderPRO/Sources/ObjectRects.c:190-198`; the identical idiom appears in the
  editor at `GliderPRO/Sources/ObjectEdit.c:2251-2259`.)

  Both validation rules are consistent with `length` being a width:
  `if (topLeft.h + length > bounds.right) length = bounds.right - topLeft.h`
  (`GliderPRO/Sources/HouseLegal.c:429-430`) and
  `length = kRoomWide - bounds.left`
  (`GliderPRO/Sources/HouseLegal.c:463`).

  Observed: `Demo House` room[1] obj[0] is a `kFlourescent` with
  `topLeft` = (h 104, v 12) and `length` = 308, giving a rect
  `(104, 12, 412, 12 + spriteHeight)` — comfortably inside the 512-wide room.

  For the other six light types `length` is not consulted by `GetObjectRect`
  (`GliderPRO/Sources/ObjectRects.c:177-188` uses `srcRects` unmodified); the
  default written on creation is 64 (`GliderPRO/Sources/ObjectAdd.c:526`) or 0
  (`GliderPRO/Sources/ObjectAdd.c:537`, `548`). Observed corpus values for those
  six codes are all even and never affect rendering.
- **`byte0` and `byte1` are dead in 1.0.4.** They are copied by the editor's
  copy/paste (`GliderPRO/Sources/ObjectEdit.c:1298-1300`) but never read by any
  gameplay or drawing code. Write them through unchanged.
- **`initial`/`state`**: `data.f.state = data.f.initial` at game start
  (`GliderPRO/Sources/Play.c:671-672`).

Lights add **no** active/hot rectangle (`GliderPRO/Sources/ObjectRects.c:930-938`);
they only contribute to `GetNumberOfLights` (`GliderPRO/Sources/Room.c:970-1099`),
and a room whose light count is zero is drawn entirely black
(`GliderPRO/Sources/RoomGraphics.c:179-191`).

Forced tops (`GliderPRO/Headers/GliderDefines.h:477-482`):

| Constant | Value | Applies to |
|---|---|---|
| `kCeilingLightTop` | 4 | `kCeilingLight` `0x51` |
| `kHipLampTop` | 23 | `kHipLamp` `0x54` |
| `kDecoLampTop` | 91 | `kDecoLamp` `0x55` |
| `kFlourescentTop` | 12 | `kFlourescent` `0x56` |
| `kTrackLightTop` | 5 | `kTrackLight` `0x57` |

`kNumTrackLights` = 3 (`GliderPRO/Headers/GliderDefines.h:445`) — a track light
draws three lamp sprites across its span.

## 5.9 Variant `g` — `applianceType` (10 bytes)

`GliderPRO/Headers/GliderStructs.h:64-72`:

```c
typedef struct
{
	Point		topLeft;					// 4
	short		height;						// 2
	Byte		byte0;						// 1
	Byte		delay;						// 1
	Boolean		initial;					// 1
	Boolean		state;						// 1
} applianceType;							// 10
```

| rel to object | Size | Type | Field |
|---|---|---|---|
| 2 | 2 | `short` | `topLeft.v` |
| 4 | 2 | `short` | `topLeft.h` |
| 6 | 2 | `short` | `height` |
| 8 | 1 | `Byte` | `byte0` |
| 9 | 1 | `Byte` | `delay` |
| 10 | 1 | `Boolean` | `initial` |
| 11 | 1 | `Boolean` | `state` |

### 5.9.1 `kCustomPict` (`0x6E`) overloads `height` as a PICT resource ID

```c
case kCustomPict:
	thePict = GetPicture(who->data.g.height);        // ObjectRects.c:~222
	if (thePict == nil)
	{
		who->data.g.height = 10000;                  // ObjectRects.c:~228
		*itsRect = srcRects[who->what];
	}
	else
		*itsRect = (*thePict)->picFrame;             // ObjectRects.c:~234
```

(`GliderPRO/Sources/ObjectRects.c:220-237`.) So for `kCustomPict` the `height`
field is a `PICT` resource ID and the object's size comes from the picture's own
`picFrame`, not from any stored rectangle. Notes:

- The editor restricts it to **[10000, 32767]**
  (`GliderPRO/Sources/ObjectInfo.c:1213`), with the dialog hinting
  "PICT"/"10000" (`GliderPRO/Sources/ObjectInfo.c:1186-1188`).
- On a missing PICT the code **mutates the in-memory house**, rewriting `height`
  to 10000. If the house is subsequently saved, the broken ID is lost.
- `kCustomPict` is by far the most common object in the corpus (4782 of 31440 live
  objects), so this path is central, not exotic.
- The picture is fetched with chain-wide `GetPicture`, so an ID that collides with
  an application PICT resolves to the house's copy only because the house's
  resource file is pushed on top of the chain by `UseResFile`
  (`GliderPRO/Sources/HouseIO.c:575`).
- Observed `Demo House` room[0] obj[9]: `what` = `0x6E`, `topLeft` = (h 33, v 61),
  `height` = **10014**, and the house's resource fork does contain
  `PICT` 10014 named `'Quilt on Wall'`.

### 5.9.2 Other `data.g` fields

- **`height`** for `kToaster` (`0x62`) is the toast-launch height, clamped so the
  toast cannot start above the ceiling:
  `if (bounds.top - data.g.height < 0) data.g.height = bounds.top;`
  (`GliderPRO/Sources/HouseLegal.c:488-491`).
- **`delay`** is a `Byte` (0..255), range-checked by the dialog
  (`GliderPRO/Sources/ObjectInfo.c:1679`) and stored as `(Byte)delay`
  (`GliderPRO/Sources/ObjectInfo.c:1688`). The dialog **hides** the delay field
  for `kShredder`, `kMacPlus`, `kTV`, `kCoffee`, `kVCR`, and `kMicrowave`
  (`GliderPRO/Sources/ObjectInfo.c:1658-1663`), so those six ignore it.
- **`byte0`** for `kMicrowave` (`0x6A`) is a **kill bitmask**
  (`GliderPRO/Sources/ObjectInfo.c:1773-1782`, consumed at
  `GliderPRO/Sources/Interactions.c:1171` as
  `kills = (short)masterObjects[whoLinked].theObject.data.g.byte0;`):

  | Bit | Mask | Effect |
  |---|---|---|
  | 0 | `0x01` | destroys rubber bands |
  | 1 | `0x02` | destroys the battery |
  | 2 | `0x04` | destroys the foil |

  For all other appliances `byte0` is unused (copy/pasted only,
  `GliderPRO/Sources/ObjectEdit.c:1311-1312`, `1322-1323`).
- **`initial`/`state`**: `data.g.state = data.g.initial`
  (`GliderPRO/Sources/Play.c:688-689`), **except** `kStereo` (`0x69`), whose state
  is forced from a global preference: `data.g.state = isPlayMusicGame`
  (`GliderPRO/Sources/Play.c:676`) — so a stereo's stored `initial` is ignored at
  runtime.

Appliance hot rectangles are hand-built, not derived from `srcRects`
(`GliderPRO/Sources/ObjectRects.c:940-979`), using
`kShredderActiveHigh` = 40 (`GliderPRO/Sources/ObjectRects.c:19`):

| `what` | Hot rect |
|---|---|
| `kShredder` `0x61` | `bottom = top + kShredderActiveHigh`, `right += 48`, offset by (−24, −36) |
| `kGuitar` `0x64` | 8 × 96, offset by (+34, +32) |
| `kOutlet` `0x67` | hand-built (`GliderPRO/Sources/ObjectRects.c:959-967`) |
| `kMicrowave` `0x6A` | **two** rects (`GliderPRO/Sources/ObjectRects.c:969-979`) |

## 5.10 Variant `h` — `enemyType` (10 bytes)

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
} enemyType;								// 10
```

| rel to object | Size | Type | Field |
|---|---|---|---|
| 2 | 2 | `short` | `topLeft.v` |
| 4 | 2 | `short` | `topLeft.h` |
| 6 | 2 | `short` | `length` |
| 8 | 1 | `Byte` | `delay` |
| 9 | 1 | `Byte` | `byte0` |
| 10 | 1 | `Boolean` | `initial` |
| 11 | 1 | `Boolean` | `state` |

Note `delay` and `byte0` are in the **opposite order** to `applianceType`
(`byte0` then `delay`). A port that shares one 10-byte decoder between variants
`g` and `h` will silently swap them.

- **`length`** is the patrol/travel span. For `kBall` (`0x76`) and `kFish`
  (`0x78`) it is clamped (`GliderPRO/Sources/HouseLegal.c:530-536`); for `kDrip`
  (`0x77`) validation forces `data.h.length = kTileHigh - bounds.bottom`
  (`GliderPRO/Sources/HouseLegal.c:537-542`), i.e. the drip always falls to the
  floor.
- **`delay`** is a `Byte`, dialog-validated and stored as `(Byte)delay`
  (`GliderPRO/Sources/ObjectInfo.c:2203-2272`).
- **`byte0` is dead** — copy/pasted only (`GliderPRO/Sources/ObjectEdit.c:1333`).
- **`initial`/`state`**: `data.h.state = data.h.initial`
  (`GliderPRO/Sources/Play.c:700-701`).

## 5.11 Variant `i` — `clutterType` (10 bytes)

`GliderPRO/Headers/GliderStructs.h:84-88`:

```c
typedef struct
{
	Rect		bounds;						// 8
	short		pict;						// 2
} clutterType;								// 10
```

| rel to object | Size | Type | Field |
|---|---|---|---|
| 2 | 2 | `short` | `bounds.top` |
| 4 | 2 | `short` | `bounds.left` |
| 6 | 2 | `short` | `bounds.bottom` |
| 8 | 2 | `short` | `bounds.right` |
| 10 | 2 | `short` | `pict` |

Structurally identical to `furnitureType` but semantically different: clutter is
decoration only, `GetObjectRect` returns `who->data.i.bounds`
(`GliderPRO/Sources/ObjectRects.c:270`), and validation rewrites the stored rect
from the canonical `srcRects` size: `theObject->data.i.bounds = bounds;`
(`GliderPRO/Sources/HouseLegal.c:574`).

- **`pict`** is a *relative* index, not a resource ID, and is only used by
  `kFlower` (`0x85`): the drawn picture is `data.i.pict + kRadioFlower1`
  (`GliderPRO/Sources/ObjectInfo.c:2312-2366`). Observed `Demo House` room[0]
  obj[7]: `kFlower` with `pict` = 2.
- **`kMirror`** (`0x82`) has its `left` and `right` forced **even**
  (`GliderPRO/Sources/HouseLegal.c:577-589`) so the reflection blit stays
  word-aligned — a 68k QuickDraw performance constraint that a Go port need not
  honour but must preserve on write to stay byte-exact.

## 5.12 Summary: the nine variants side by side

Byte offsets are relative to the start of the 12-byte object record.

| off | `a` blower | `b` furniture | `c` bonus | `d` transport | `e` switch | `f` light | `g` appliance | `h` enemy | `i` clutter |
|---|---|---|---|---|---|---|---|---|---|
| 0–1 | `what` | `what` | `what` | `what` | `what` | `what` | `what` | `what` | `what` |
| 2–3 | `topLeft.v` | `bounds.top` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `bounds.top` |
| 4–5 | `topLeft.h` | `bounds.left` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `bounds.left` |
| 6–7 | `distance` | `bounds.bottom` | `length` | `tall` | `delay` | `length` | `height` | `length` | `bounds.bottom` |
| 8 | `initial`(B) | `bounds.right` | `points` (short 8–9) | `where` (short 8–9) | `where` (short 8–9) | `byte0`(B) | `byte0`(B) | `delay`(B) | `bounds.right` |
| 9 | `state`(B) | ″ | ″ | ″ | ″ | `byte1`(B) | `delay`(B) | `byte0`(B) | ″ |
| 10 | `vector`(B) | `pict` (short 10–11) | `state`(B) | `who`(B) | `who`(B) | `initial`(B) | `initial`(B) | `initial`(B) | `pict` (short 10–11) |
| 11 | `tall`(B) | ″ | `initial`(B) | `wide`(B) | `type`(B) | `state`(B) | `state`(B) | `state`(B) | ″ |

"(B)" marks single-byte fields. Note the four distinct layouts of bytes 8–11:
two shorts (`b`, `i`), a short then two bytes (`c`, `d`, `e`), and four bytes
(`a`, `f`, `g`, `h`).

---

# Part 6 — Coordinate systems

Glider PRO uses three nested coordinate systems. All three must be modelled
exactly because they interact.

## 6.1 Room-local pixel coordinates

Every object position and rectangle in the file is in **room-local pixels**, with
the origin at the room's top-left corner, +x right and +y down (standard
QuickDraw).

| Constant | Value | Meaning | Citation |
|---|---|---|---|
| `kRoomWide` | 512 | room width in px | `GliderPRO/Headers/GliderDefines.h:499` |
| `kTileHigh` | 322 | room height in px | `GliderPRO/Headers/GliderDefines.h:498` |
| `kTileWide` | 64 | tile width in px | `GliderPRO/Headers/GliderDefines.h:497` |
| `kNumTiles` | 8 | tiles across | `GliderPRO/Headers/GliderDefines.h:496` |
| `kVertLocalOffset` | 322 | vertical stride between stacked rooms | `GliderPRO/Headers/GliderDefines.h:501` |
| `kFloorSupportTall` | 44 | height of the floor-support strip drawn under a structure | `GliderPRO/Headers/GliderDefines.h:500` |
| `kShadowTop` | 306 | y of the glider's ground shadow | `GliderPRO/Headers/GliderDefines.h:553` |

The legal envelope for an object is exactly the room rect:

```c
QSetRect(&roomRect, 0, 0, kRoomWide, kTileHigh);   // HouseLegal.c:71
```

Note `kVertLocalOffset == kTileHigh == 322`: vertically adjacent rooms are stacked
with no gap, so a neighbour one floor up occupies local y ∈ [−322, 0).

The glider start point is clamped into a sub-rect
(`GliderPRO/Sources/HouseLegal.c:59-66`) using `kGliderWide` = 48 and
`kGliderHigh` = 20 (`GliderPRO/Headers/GliderDefines.h:548-549`):

- `initial.h` ∈ [0, `kRoomWide - kGliderWide`] = [0, **464**]
- `initial.v` ∈ [0, `kTileHigh - kGliderHigh`] = [0, **302**]

Observed across the corpus: `initial.h` spans 30..424 and `initial.v` spans
7..200 — both inside the clamps, and (crucially for the field-order proof in
Part 12.3) the *first* stored short never exceeds 302 while the second reaches
424.

## 6.2 The (suite, floor) room grid

Rooms are placed on a 2-D integer grid. **There is no explicit adjacency list:
rooms connect purely by being neighbours on this grid.**

| Axis | Field | Direction | Legal range | Range constant |
|---|---|---|---|---|
| horizontal | `suite` | +1 = east | **0 .. 127** | `kMaxNumRoomsH` = 128 (`GliderPRO/Headers/GliderDefines.h:543`) |
| vertical | `floor` | **+1 = north / up** | **−7 .. 56** | `kMaxNumRoomsV` = 64 (`GliderPRO/Headers/GliderDefines.h:544`) |

The asymmetric floor range comes from `kNumUndergroundFloors` = 8
(`GliderPRO/Headers/GliderDefines.h:535`): floors −7..0 are the 8 underground
floors and 1..56 are above ground, giving exactly 64 rows. Validation enforces
this (`GliderPRO/Sources/HouseLegal.c:804-817`):

```c
if (((*thisHouse)->rooms[i].floor > 56) || ((*thisHouse)->rooms[i].floor < -7))
	(*thisHouse)->rooms[i].suite = kRoomIsEmpty;                      // HouseLegal.c:804-807
...
if (((*thisHouse)->rooms[i].suite >= kMaxNumRoomsH) || ((*thisHouse)->rooms[i].suite < 0))
	(*thisHouse)->rooms[i].suite = kRoomIsEmpty;                      // HouseLegal.c:814-817
```

Note the failure mode: an out-of-range room is not repaired, it is **deleted** by
setting `suite = kRoomIsEmpty` (−1).

Duplicate grid positions are also deleted, using a bitmap pigeonhole
(`GliderPRO/Sources/HouseLegal.c:648`, `665-666`):

```c
#define kRoomsTimesSuites 8192                                        // HouseLegal.c:648
bitPlace = (((*thisHouse)->rooms[i].floor + 7) * 128) + (*thisHouse)->rooms[i].suite;   // HouseLegal.c:665-666
```

`8192 = 64 * 128`, and `(floor + 7) ∈ [0, 63]` exactly spans it — independent
confirmation of the −7..56 floor range.

`suite == -1` (`kRoomIsEmpty`, `GliderPRO/Headers/GliderDefines.h:525`) marks a
**deleted placeholder slot**; `DeleteRoom` sets it
(`GliderPRO/Sources/Room.c:462-463`) and reassigns `firstRoom`
(`GliderPRO/Sources/Room.c:476`). `floor` is left untouched, so a placeholder's
`floor` is meaningless residue. **Test `suite == -1`, never `floor == -1`** —
`floor` = −1 is a perfectly legal underground floor.

### 6.2.1 The nine neighbour deltas

`GetNeighborRoomNumber(short where)` (`GliderPRO/Sources/Room.c:562-635`) maps a
direction constant to a (`suite`, `floor`) delta and then looks the room up:

| Constant | Δ`suite` (h) | Δ`floor` (v) |
|---|---|---|
| `kCentralRoom` | 0 | 0 |
| `kNorthRoom` | 0 | +1 |
| `kNorthEastRoom` | +1 | +1 |
| `kEastRoom` | +1 | 0 |
| `kSouthEastRoom` | +1 | −1 |
| `kSouthRoom` | 0 | −1 |
| `kSouthWestRoom` | −1 | −1 |
| `kWestRoom` | −1 | 0 |
| `kNorthWestRoom` | −1 | +1 |

so **north = floor + 1** confirms `floor` increases upward. The reverse lookup is:

```c
short GetRoomNumber (short floor, short suite)      // Room.c:737
{
	...
	for (i = 0; i < (*thisHouse)->nRooms; i++)
		if (((*thisHouse)->rooms[i].floor == floor) &&
				((*thisHouse)->rooms[i].suite == suite))
			return(i);
	return(kRoomIsEmpty);                            // Room.c:~757
}
```

— a linear scan returning **−1** when no room occupies that cell. This is why
dangling links degrade gracefully to "unlinked" rather than crashing (Part 7.4).

`SetToNearestNeighborRoom` (`GliderPRO/Sources/Room.c:639-707`) walks the eight
neighbours in clockwise order looking for any existing room.

### 6.2.2 The map view's row mapping

The editor's map draws floors top-down using `kMapGroundValue` = 56
(`GliderPRO/Sources/Map.c:22`):

```c
v = kMapGroundValue - thisRoom->floor;      // Map.c:58
```

so floor 56 → row 0 and floor −7 → row 63 — again exactly 64 rows. The inverse,
used when the user clicks:

```c
suite = h + mapLeftRoom;                    // Map.c:138, 223
floor = kMapGroundValue - (i + mapTopRoom);  // Map.c:139, 224
```

Map cells are `kMapRoomWidth` = 32 by `kMapRoomHeight` = 20 px
(`GliderPRO/Headers/GliderDefines.h:246-247`) in a
`kMapRoomsWide` = `kMapRoomsHigh` = 9 grid (`GliderPRO/Sources/Map.c:17-18`).
Scroll position defaults for a new house are `mapLeftRoom` = 60,
`mapTopRoom` = 50 (`GliderPRO/Sources/House.c:148-149`); these live in
preferences, **not** in the house file.

Observed corpus ranges: `floor` ∈ [−7, 39], `suite` ∈ [0, 127] — the full legal
suite range is actually used, so the 128 limit is a real constraint, not a
generous one.

## 6.3 Room index space

`rooms[]` indices (`0 .. nRooms-1`) are a **third, independent** coordinate: they
are what `firstRoom`, `savedGame.roomNumber`, and the decoded link target all
name. Indices are *not* stable across editor operations — `CompressHouse`
(`GliderPRO/Sources/HouseLegal.c:688-734`) moves live rooms into freed slots and
patches `firstRoom` (`GliderPRO/Sources/HouseLegal.c:713`) — but they *are* what
is stored, so a port must preserve them byte-for-byte on round-trip.

Object slot indices (`0 .. 23`) are the fourth coordinate, referenced by the
`who` byte of a link. These are never compacted.

---

# Part 7 — Room links: the `where` / `who` encoding

## 7.1 What a link is

A link connects a *source object* (a switch, trigger, or transport) to a *target
object* in a possibly different room. It is stored in the source object as two
fields:

| Field | Type | Meaning | Sentinel |
|---|---|---|---|
| `where` | `short` | destination room, **encoded as a merged floor/suite value** | −1 = unlinked |
| `who` | `Byte` | destination object slot index (0..23) | 255 = unlinked |

Crucially, `where` does **not** store a room *index*. It stores the destination
room's grid position, packed into one `short`. This makes links survive room
renumbering (`CompressHouse`) at the cost of a lookup on every dereference.

## 7.2 `MergeFloorSuite` / `ExtractFloorSuite`

`GliderPRO/Sources/Link.c:34-53`, verbatim:

```c
short MergeFloorSuite (short floor, short suite)                          // Link.c:34
{
	return ((suite * 100) + floor);                                        // Link.c:36
}

void ExtractFloorSuite (short combo, short *floor, short *suite)          // Link.c:41
{
	if ((*thisHouse)->version < 0x0200)   // old floor/suite combo         // Link.c:43
	{
		*floor = (combo / 100) - kNumUndergroundFloors;                     // Link.c:45
		*suite = combo % 100;                                              // Link.c:46
	}
	else
	{
		*suite = combo / 100;                                              // Link.c:50
		*floor = (combo % 100) - kNumUndergroundFloors;                      // Link.c:51
	}
}
```

The caller pre-biases the floor before merging
(`GliderPRO/Sources/Link.c:275-277`):

```c
GetRoomFloorSuite(thisRoomNumber, &floor, &suite);
floor += kNumUndergroundFloors;                    // Link.c:277
```

then merges (`GliderPRO/Sources/Link.c:283`, `290`, `302`, `309`).

So the **version ≥ 0x0200 encoding** is:

```
where = suite * 100 + (floor + 8)
```

with the inverse

```
suite =  where / 100
floor = (where % 100) - 8
```

and the **version < 0x0200 encoding swaps the two components**:

```
where = (floor + 8) * 100 + suite      (inferred from the extract)
suite =  where % 100
floor = (where / 100) - 8
```

Consequences of the `% 100` field width:

- `floor + 8` ∈ [1, 64] for the legal floor range −7..56, which fits in two
  decimal digits. Good.
- `suite` ∈ [0, 127], so `where` ∈ [0, 12764] for version 2 — fits comfortably in
  a signed `short` (max 32767).
- For **version 1** the *suite* occupies the low two digits, so a version-1 house
  can only address suites 0..99. Suites 100..127 are unreachable by link in
  version 1. This is precisely why the migration in Part 11 exists.
- The **encoding is lossy for illegal input**: a `floor + 8` of 100 or more would
  overflow into the suite digits. Validation's floor range check prevents it.

Empirical verification (`Sampler`, version 0x0200):

| Room | `suite` | `floor` | Encoded `where` should be |
|---|---|---|---|
| room[0] `'Entrance'` | 67 | 1 | `67*100 + (1+8)` = **6709** |
| room[1] `'Welcome'` | 66 | 1 | `66*100 + (1+8)` = **6609** |

and the observed link fields are:

- room[0] obj[4] `kFloorTrans` (`0x35`): `where` = **6709**, `who` = 5 —
  decodes to suite 67, floor 1 = room[0] itself, object slot 5, which is
  `kCeilingTrans` (`0x36`). A floor transport pointing at a ceiling transport in
  the same room: correct.
- room[0] obj[6] `kMailboxLf` (`0x33`): `where` = **6609**, `who` = 2 —
  decodes to suite 66, floor 1 = room[1], object slot 2, which **is** a
  `kMailboxLf`. Correct.

This simultaneously verifies the merge formula, the +8 bias, the field order
(suite in the high digits for version 2), and the room-record offsets of `floor`
(rel 52) and `suite` (rel 54).

## 7.3 Which objects carry links

Exactly two field sets, enumerated identically in three places —
`CountHouseLinks` (`GliderPRO/Sources/House.c:258-307`), `GenerateLinksList`
(`GliderPRO/Sources/House.c:317-385`), and `GenerateRetroLinks`
(`GliderPRO/Sources/House.c:533-613`):

**Switch links, using `data.e.where` / `data.e.who`**
(`GliderPRO/Sources/House.c:279-286`, `339-346`, `561-568`):

| `what` | Constant |
|---|---|
| `0x41` | `kLightSwitch` |
| `0x42` | `kMachineSwitch` |
| `0x43` | `kThermostat` |
| `0x44` | `kPowerSwitch` |
| `0x45` | `kKnifeSwitch` |
| `0x46` | `kInvisSwitch` |
| `0x47` | `kTrigger` |
| `0x48` | `kLgTrigger` |

**Transport links, using `data.d.where` / `data.d.who`**
(`GliderPRO/Sources/House.c:291-296`, `361-366`, `586-591`):

| `what` | Constant |
|---|---|
| `0x33` | `kMailboxLf` |
| `0x34` | `kMailboxRt` |
| `0x35` | `kFloorTrans` |
| `0x36` | `kCeilingTrans` |
| `0x3F` | `kInvisTrans` |
| `0x40` | `kDeluxeTrans` |

**Not links, despite living in the same variants:**

- `kSoundTrigger` (`0x49`) — `data.e.where` is a `snd ` resource ID (Part 5.7.1).
  Absent from all three enumerators and from `GetRoomLinked`
  (`GliderPRO/Sources/Objects.c:149-156`, which stops at `0x48`).
- `kUpStairs` (`0x31`) / `kDownStairs` (`0x32`) — they use `data.d` but are
  **implicitly** linked by grid adjacency, not by `where`. Validation checks that
  a `kUpStairs` has a matching `kDownStairs` in the room to the **north**
  (`GliderPRO/Sources/HouseLegal.c:978-1008`) and vice versa to the **south**
  (`GliderPRO/Sources/HouseLegal.c:1009-1039`). Their `where`/`who` are set to
  −1/255 on creation (`GliderPRO/Sources/ObjectAdd.c:316-319`) and never used.
- doors and windows (`0x37`–`0x3E`) — also implicit, by side adjacency and the
  room's `bounds` openings.

The link data structure used in memory (`GliderPRO/Headers/GliderStructs.h:281-285`):

```c
typedef struct { short srcRoom, srcObj; short destRoom, destObj; } linksType;
```

and the reverse index (`GliderPRO/Headers/GliderStructs.h:341-345`):

```c
typedef struct { short room; short object; } retroLink;
```

Neither reaches disk.

Link creation/removal is gated by target category
(`GliderPRO/Sources/Link.c:63-192`) using
`kSwitchLinkOnly` = 3, `kTriggerLinkOnly` = 4, `kTransportLinkOnly` = 5
(`GliderPRO/Headers/GliderDefines.h:463-465`). `DoLink` sets `who = objActive`
and `DoUnlink` writes `where = -1; who = 255;`
(`GliderPRO/Sources/Link.c:333-334`, `338-339`, `348-349`, `353-354`).

## 7.4 Dereferencing a link at runtime

`GetRoomLinked` (`GliderPRO/Sources/Objects.c:126-170`):

1. Switch on `who->what`.
2. For transports `0x33`–`0x36`, `0x3F`, `0x40`: `compoundRoomNumber = who->data.d.where`
   (`GliderPRO/Sources/Objects.c:133-138`).
3. For switches `0x41`–`0x48`: `compoundRoomNumber = who->data.e.where`
   (`GliderPRO/Sources/Objects.c:149-156`).
4. Anything else: `whereLinked = -1` (`GliderPRO/Sources/Objects.c:167-169`).
5. If `compoundRoomNumber != -1`:
   `ExtractFloorSuite(compoundRoomNumber, &floor, &suite);`
   `whereLinked = GetRoomNumber(floor, suite);`
   else `whereLinked = -1`.

`GetObjectLinked` (`GliderPRO/Sources/Objects.c:177-193`):

```c
if (who->data.d.who != 255)
	whoLinked = (short)who->data.d.who;      // Objects.c:~190
else
	whoLinked = -1;                           // Objects.c:~192
```

Note it reads `data.d.who` unconditionally — which is safe only because `data.d.who`
and `data.e.who` are at the **same byte offset** (rel 10) in both variants.

**Because `GetRoomNumber` returns `kRoomIsEmpty` for an unoccupied cell, a link to
a deleted or never-created room silently becomes "unlinked."** A port must
reproduce this graceful degradation rather than erroring out: 6 of the 22 shipped
houses contain dangling links (worst case `Slumberland` with **69**).

**Unbounded `who`.** `GenerateRetroLinks` initializes
`retroLinkList[i].room = -1` for `i < kMaxRoomObs`
(`GliderPRO/Sources/House.c:546-547`) and then indexes
`retroLinkList[objectLinked]` with the raw `data.e.who` / `data.d.who`
(`GliderPRO/Sources/House.c:577`, `600`) **with no bound check**. The corpus
contains exactly one out-of-range value: `CD Demo House` room[72] (floor 1,
suite 21) object[22], a `kMailboxRt` with `where` = 2109 and **`who` = 35** —
past the end of `objects[24]`. On the original this reads 11 × 12 = 132 bytes
into the following room's record; in Go it panics. A port must clamp or reject
`who >= kMaxRoomObs` (treating it as unlinked is the safest faithful choice).

---

# Part 8 — The load algorithm (`ReadHouse`)

`GliderPRO/Sources/HouseIO.c:315-441`. Numbered pseudocode, original variable
names in parentheses.

```
ReadHouse():
 1. if (gameDirty || fileDirty):                               // HouseIO.c:327
      if (houseIsReadOnly): WriteScoresToDisk()                 // HouseIO.c:331
      else:                 WriteHouse(false)                   // HouseIO.c:337
 2. byteCount <- GetEOF(houseRefNum)                            // HouseIO.c:341
      on error -> CheckFileError, return false
 3. [demo build only] if (byteCount != 16526) return false      // HouseIO.c:349-350
 4. dispose old handle; thisHouse <- NewHandle(byteCount)       // HouseIO.c:353-356
      nil -> YellowAlert(kYellowNoMemory, 10), return false
 5. MoveHHi(thisHouse)                                          // HouseIO.c:362
 6. SetFPos(houseRefNum, fsFromStart, 0)                        // HouseIO.c:364
 7. HLock(thisHouse); FSRead(houseRefNum, &byteCount, *thisHouse)  // HouseIO.c:371-372
      // one flat read of the ENTIRE data fork into the struct
 8. numberRooms <- (*thisHouse)->nRooms                         // HouseIO.c:380
 9. [demo build only] if (numberRooms != 45) return false       // HouseIO.c:382
10. if (numberRooms < 1 || byteCount == 0):                     // HouseIO.c:385
      numberRooms <- 0; noRoomAtAll <- true
      YellowAlert(kYellowNoRooms, 0); return false              // HouseIO.c:387-391
11. wasHouseVersion <- (*thisHouse)->version                    // HouseIO.c:394
12. if (wasHouseVersion >= kNewHouseVersion /*0x0300*/):        // HouseIO.c:395
      YellowAlert(kYellowNewerVersion, 0); return false         // HouseIO.c:397-400
13. houseUnlocked <- ((timeStamp & 1) == 0)                      // HouseIO.c:402
14. [demo build only] if (houseUnlocked) return false           // HouseIO.c:404-405
15. changeLockStateOfHouse <- false; saveHouseLocked <- false   // HouseIO.c:407-408
16. whichRoom <- (*thisHouse)->firstRoom                        // HouseIO.c:410
17. [demo build only] if (whichRoom != 0) return false          // HouseIO.c:412-413
18. wardBitSet        <- ((flags & 0x00000001) == 0x00000001)   // HouseIO.c:416
    phoneBitSet       <- ((flags & 0x00000002) == 0x00000002)   // HouseIO.c:417
    bannerStarCountOn <- ((flags & 0x00000004) == 0x00000000)   // HouseIO.c:418   NOTE: inverted
19. HUnlock(thisHouse)                                          // HouseIO.c:420
20. noRoomAtAll <- (RealRoomNumberCount() == 0)                 // HouseIO.c:422
21. thisRoomNumber <- -1; previousRoom <- -1                    // HouseIO.c:423-424
22. if (!noRoomAtAll): CopyRoomToThisRoom(whichRoom)            // HouseIO.c:425-426
23. if (houseIsReadOnly):                                       // HouseIO.c:428
      houseUnlocked <- false                                    // HouseIO.c:430
      ReadScoresFromDisk()   // pull highScores from the 'gliS' sidecar  // HouseIO.c:431
24. objActive <- kNoObjectSelected; ReflectCurrentRoom(true)    // HouseIO.c:436-437
25. gameDirty <- false; fileDirty <- false; UpdateMenus(false)  // HouseIO.c:438-440
26. return true
```

Observations a port must not miss:

- **The whole data fork is read in one `FSRead` directly over the struct.** There
  is no field-by-field parsing, no length validation beyond `nRooms >= 1`, and no
  byte-swapping — the file is a raw big-endian memory image.
- **`nRooms` from the file is trusted at step 8** and is only cross-checked later,
  during `CheckHouseForProblems` → `ValidateNumberOfRooms`, which runs *on save*
  and only if the `isHouseChecks` preference is on
  (`GliderPRO/Sources/HouseLegal.c:1070-1075`). So a house whose `nRooms` exceeds
  what the file length supports will read out of bounds on the original. **A Go
  port must clamp `nRooms` to `(len(data) - 866) / 348` at load time**, which is
  precisely what `ValidateNumberOfRooms` computes
  (`GliderPRO/Sources/HouseLegal.c:630-631`).
- The version check is `>=`, so 0x0300 exactly is already refused.
- Nothing in the load path validates room `floor`/`suite`, object `what` codes,
  link targets, or `tiles[]` indices.

The file is opened earlier, in `OpenHouse` (`GliderPRO/Sources/HouseIO.c:163-199`):

```
OpenHouse():
 1. if (houseOpen): CloseHouse()                                       // HouseIO.c:169-173
 2. if (housesFound < 1 || thisHouseIndex == -1) return false          // HouseIO.c:174-175
 3. ResolveAliasFile(&theHousesSpecs[thisHouseIndex], true, ...)       // HouseIO.c:177-178
 4. houseIsReadOnly <- IsFileReadOnly(&theHousesSpecs[thisHouseIndex]) // HouseIO.c:187
 5. FSpOpenDF(&theHousesSpecs[thisHouseIndex], fsCurPerm, &houseRefNum)// HouseIO.c:189
 6. houseOpen <- true                                                  // HouseIO.c:193
 7. OpenHouseResFork()                                                 // HouseIO.c:194
 8. hasMovie <- false; tvInRoom <- false; tvWithMovieNumber <- -1      // HouseIO.c:196-198
 9. OpenHouseMovie()                                                   // HouseIO.c:199
```

`IsFileReadOnly` is a **stub that always returns `false`** in 1.0.4
(`GliderPRO/Sources/HouseIO.c:663`), so the read-only branch (and therefore the
`'gliS'` sidecar path) is effectively dead unless the caller sets
`houseIsReadOnly` some other way.

`OpenHouseResFork` (`GliderPRO/Sources/HouseIO.c:567-577`):

```
 1. houseResFork <- FSpOpenResFile(&theHousesSpecs[thisHouseIndex], fsCurPerm)  // HouseIO.c:571
 2. if (houseResFork == -1): YellowAlert(kYellowFailedResOpen, ResError())      // HouseIO.c:573
 3. else: UseResFile(houseResFork)                                              // HouseIO.c:575
```

Step 3 pushes the house's resource file to the **top of the resource chain**, which
is why house PICTs can shadow the application's own (Part 13.2).

`OpenHouseMovie` (`GliderPRO/Sources/HouseIO.c:67-140`) looks for a QuickTime
sidecar named `<house name>.mov`:

```
 1. theSpec <- theHousesSpecs[thisHouseIndex]                     // HouseIO.c:80
 2. PasStringConcat(theSpec.name, "\p.mov")                       // HouseIO.c:81
 3. if (FSpGetFInfo(&theSpec, &finderInfo) != noErr) return       // HouseIO.c:83-85
 4. OpenMovieFile(&theSpec, &movieRefNum, fsCurPerm)              // HouseIO.c:87
 5. NewMovieFromFile(&theMovie, movieRefNum, ...)                 // HouseIO.c:94
 6. spaceSaver <- NewHandle(307200L)   // 300 KB scratch          // HouseIO.c:104
 7. LoadMovieIntoRam(theMovie, ...)                               // HouseIO.c:113
 8. PrerollMovie(theMovie, 0, 0x000F0000)  // rate 15.0 fixed     // HouseIO.c:124
 9. SetTimeBaseFlags(theTime, loopTimeBase)                       // HouseIO.c:133
10. hasMovie <- true                                              // HouseIO.c:139
```

Observed: 15 of the 22 shipped houses have a `.mov` sidecar in
`GliderPRO/Houses/`. The first 16 bytes of
`Demo House.mov` are
`00 01 D0 08 6D 64 61 74 40 00 0B 55 00 08 00 00` — a `mdat` atom of length
0x0001D008 = 118792, i.e. a classic QuickTime `mdat`-first movie file. The movie
plays inside a `kTV` object (`tvInRoom`, `tvWithMovieNumber`).

---

# Part 9 — The save algorithm

## 9.1 `WriteHouse`

`GliderPRO/Sources/HouseIO.c:448-518`:

```
WriteHouse(checkIt):
 1. if (!houseOpen): YellowAlert(kYellowUnaccounted, 4); return false   // HouseIO.c:454-458
 2. SetFPos(houseRefNum, fsFromStart, 0)                                // HouseIO.c:460
 3. CopyThisRoomToRoom()      // flush the editor's working room copy    // HouseIO.c:467
 4. if (checkIt): CheckHouseForProblems()                                // HouseIO.c:469-470
 5. HLock(thisHouse); byteCount <- GetHandleSize(thisHouse)              // HouseIO.c:472-473
 6. if (fileDirty):                                                      // HouseIO.c:475
      GetDateTime(&timeStamp)                                            // HouseIO.c:477
      timeStamp &= 0x7FFFFFFF                                            // HouseIO.c:478
      if (changeLockStateOfHouse): houseUnlocked <- !saveHouseLocked     // HouseIO.c:480-481
      if (houseUnlocked): timeStamp &= 0x7FFFFFFE                        // HouseIO.c:483-484
      else:               timeStamp |= 0x00000001                        // HouseIO.c:485-486
      (*thisHouse)->timeStamp <- (long)timeStamp                         // HouseIO.c:487
      (*thisHouse)->version   <- wasHouseVersion                         // HouseIO.c:488
 7. FSWrite(houseRefNum, &byteCount, *thisHouse)                         // HouseIO.c:491
 8. SetEOF(houseRefNum, byteCount)                                       // HouseIO.c:499
 9. HUnlock(thisHouse)                                                   // HouseIO.c:507
10. if (changeLockStateOfHouse):
      changeLockStateOfHouse <- false; ReflectCurrentRoom(true)          // HouseIO.c:511-512
11. gameDirty <- false; fileDirty <- false; UpdateMenus(false)           // HouseIO.c:515-517
12. return true
```

Notes:

- **`byteCount` is `GetHandleSize`, not a recomputed `866 + 348*nRooms`.** This is
  the mechanism by which the 2 slack bytes observed in `Sampler` reach disk
  (Part 12.4).
- Steps 7 and 8 in that order mean a *shrinking* save first writes the new
  (shorter) content and then truncates. A crash between them leaves a file with
  trailing garbage that `ValidateNumberOfRooms` would later reinterpret as extra
  rooms.
- **The timestamp and version are only rewritten when `fileDirty`.** A pure
  score-only save (`gameDirty` but not `fileDirty`) leaves both untouched.
- No checksum, no magic number, no length field is written. The only structural
  invariant is `len(file) == 866 + 348*nRooms` (or `868 + 348*nRooms`).

`CloseHouse` (`GliderPRO/Sources/HouseIO.c:522-561`) decides how to save:

```
 1. if (!houseOpen) return true
 2. if (gameDirty):
      if (houseIsReadOnly): WriteScoresToDisk()
      else:                 WriteHouse(theMode == kEditMode)    // HouseIO.c:538
 3. else if (fileDirty): QuerySaveChanges()                     // HouseIO.c:544
 4. CloseHouseResFork(); CloseHouseMovie(); FSClose(houseRefNum)
 5. houseOpen <- false
```

so `checkIt` (the full legality pass) is passed **true only in edit mode**.

`QuerySaveChanges` (`GliderPRO/Sources/HouseIO.c:596+`) is where migration
happens:

```
 ... user chose "Save" ...
 if (wasHouseVersion < kHouseVersion):        // HouseIO.c:610
     ConvertHouseVer1To2()                    // HouseIO.c:611
 wasHouseVersion <- kHouseVersion             // HouseIO.c:612
 ... WriteHouse(...) ...
```

`SaveHouseAs` is **entirely commented out** and returns `false`
(`GliderPRO/Sources/HouseIO.c:307`); 1.0.4 can only save in place.

## 9.2 House creation

`CreateNewHouse` (`GliderPRO/Sources/House.c:42-101`):

```
 1. NavPutFile(..., 'gliH', 'ozm5', nil)             // House.c:58   -> user picks a name
 2. FSpCreate(&theSpec, 'ozm5', 'gliH', theReply.keyScript)  // House.c:85  data fork
 3. HCreateResFile(...)                              // House.c:88   empty resource fork
 4. ... InitializeEmptyHouse() ...
```

Note the **argument order flip**: `NavPutFile` takes (type, creator) =
`('gliH', 'ozm5')` while `FSpCreate` takes (creator, type) =
`('ozm5', 'gliH')`. Both name type `'gliH'` and creator `'ozm5'`.

`InitializeEmptyHouse` (`GliderPRO/Sources/House.c:108-162`):

```
 1. thisHouse <- NewHandle(sizeof(houseType))     // House.c:116  -- NOT zeroed
 2. version   <- kHouseVersion (0x0200)           // House.c:127
 3. firstRoom <- -1                               // House.c:128
 4. timeStamp <- 0L                               // House.c:129
 5. flags     <- 0L                               // House.c:130
 6. initial.h <- 32; initial.v <- 32              // House.c:131-132
 7. ZeroHighScores()                              // House.c:133
 8. banner  <- GetLocalizedString(11)             // House.c:135-136
 9. trailer <- GetLocalizedString(12)             // House.c:137-138
10. hasGame <- false                              // House.c:139
11. nRooms  <- 0                                  // House.c:140
12. wardBitSet <- false; phoneBitSet <- false     // House.c:142-143
13. numberRooms <- 0                              // House.c:147
14. mapLeftRoom <- 60; mapTopRoom <- 50           // House.c:148-149
15. thisRoomNumber <- kRoomIsEmpty                // House.c:150
16. houseUnlocked <- true                         // House.c:152
17. noRoomAtAll   <- true                         // House.c:155
```

**`unusedShort`, `savedGame`, and `unusedBoolean` are never initialized**, and
`NewHandle` (unlike `NewHandleClear`) does not zero. That is the direct source of
the garbage observed in those fields across the corpus (Part 12.5): `unusedShort`
values include −30082, 13107, 26228, 2074, 259, 222, 196, 147, 60; `unusedBoolean`
values include 255, 185, 37, 30, 14, 2.

**A Go port that writes zeros there will not reproduce shipped files byte-for-byte,
but will produce files the original accepts.** For a round-trip-exact tool, carry
the original bytes through verbatim.

## 9.3 Room creation

`CreateNewRoom` (`GliderPRO/Sources/Room.c:159-239`):

```
 1. PasStringCopy("\pUntitled Room", thisRoom->name)      // Room.c:169
 2. leftStart <- 32; rightStart <- 32                      // Room.c:170-171
 3. bounds <- 0                                            // Room.c:172
 4. unusedByte <- 0                                        // Room.c:173
 5. visited <- false                                       // Room.c:174
 6. background <- lastBackground                            // Room.c:175
 7. SetInitialTiles(background, ...)                        // Room.c:176 (see 4.6)
 8. floor <- v; suite <- h                                  // Room.c:177-178
 9. openings <- 0                                           // Room.c:179
10. numObjects <- 0                                         // Room.c:180
11. for i in 0..kMaxRoomObs-1: objects[i].what <- kObjectIsEmpty   // Room.c:181-182
        // NOTE: only 'what' is written; data[] keeps whatever was there
12. scan rooms[] for a slot with suite == kRoomIsEmpty       // Room.c:188-194
      if found: rooms[slot] <- *thisRoom
      else:     PtrAndHand((Ptr)thisRoom, (Handle)thisHouse, sizeof(roomType))  // Room.c:200
                nRooms++                                     // Room.c:210
```

Step 12 is the reason the format tolerates placeholder rooms in the middle of the
array: deletion sets `suite = -1`, creation reuses the slot.

Step 11 is the reason an empty object slot's 10 data bytes are arbitrary: on a
freshly extended handle they are `NewHandle`/`SetHandleSize` residue; on a reused
slot they are the deleted room's old object data.

## 9.4 Where the 2 slack bytes come from

`LopOffExtraRooms` computes the shrunken handle size as

```c
newSize = sizeof(houseType) + (sizeof(roomType) * (long)r);   // HouseLegal.c:765
```

using **`sizeof(houseType)`**, whereas the data actually begins at
**`offsetof(houseType, rooms)`**. Those differ: `houseType`'s largest alignment
requirement is 4 (from `long timeStamp` / `long flags` / `long scores[]`), so under
PowerPC alignment the compiler rounds `sizeof` up from 866 to **868**, while under
68k (`mac68k`, 2-byte) alignment it stays **866**. `sizeof(roomType)` = 348 is a
multiple of 4 either way.

Consequently a PPC-built editor writes `868 + 348*nRooms` bytes and a 68k-built
one writes `866 + 348*nRooms`. `ValidateNumberOfRooms` uses the same
`sizeof(houseType)` in its division
(`GliderPRO/Sources/HouseLegal.c:630-631`), so each build is self-consistent, and
the integer division tolerates the other build's files.

Observed: **21 of 22 shipped houses are exactly `866 + 348*nRooms`; only `Sampler`
is `868 + 348*nRooms`** (1564 = 868 + 2×348). See Part 12.4 for the hexdump.

**Go port rule:** parse room `r` at `866 + 348*r`, derive the room count as
`min(nRooms, (len(data) - 866) / 348)`, and ignore up to 2 trailing bytes. Do not
assume `len(data) % 348 == 866 % 348`.

---

# Part 10 — The legality / validation pass

Run from `WriteHouse` when `checkIt` is true, i.e. on an edit-mode save
(`GliderPRO/Sources/HouseIO.c:469-470`, `GliderPRO/Sources/HouseIO.c:538`).

## 10.1 `CheckHouseForProblems` driver order

`GliderPRO/Sources/HouseLegal.c:1052-1218`. `isHouseChecks` is a user preference;
the three steps outside its guard always run.

| # | Step | Guarded by `isHouseChecks`? | Citation | Mutates the file? |
|---|---|---|---|---|
| 0 | `houseErrors <- 0`; `CopyThisRoomToRoom()`; save `wasRoom`, `wasActive` | — | `GliderPRO/Sources/HouseLegal.c:1058-1061` | yes (flushes working room) |
| 1 | `WrapBannerAndTrailer()` | **no** | `GliderPRO/Sources/HouseLegal.c:1068` | yes |
| 2 | `ValidateNumberOfRooms()` | yes | `GliderPRO/Sources/HouseLegal.c:1075` | yes (`nRooms`) |
| 3 | `CheckDuplicateFloorSuite()` | yes | `GliderPRO/Sources/HouseLegal.c:1089` | yes (deletes rooms) |
| 4 | `CompressHouse()` | **no** | `GliderPRO/Sources/HouseLegal.c:1103` | yes (reorders rooms) |
| 5 | `LopOffExtraRooms()` | **no** | `GliderPRO/Sources/HouseLegal.c:1105` | yes (shrinks handle, `nRooms`) |
| 6 | `ValidateRoomNumbers()` | yes | `GliderPRO/Sources/HouseLegal.c:1110` | yes (deletes rooms) |
| 7 | `CountUntitledRooms()` | yes | `GliderPRO/Sources/HouseLegal.c:1127` | no (count only) |
| 8 | `CheckRoomNameLength()` | yes | `GliderPRO/Sources/HouseLegal.c:1144` | yes (`unusedByte`, `name[0]`) |
| 9 | `MakeSureNumObjectsJives()` | yes | `GliderPRO/Sources/HouseLegal.c:1161` | yes (`numObjects`) |
| 10 | `KeepAllObjectsLegal()` | yes | `GliderPRO/Sources/HouseLegal.c:1180` | yes (every object) |
| 11 | `CheckForStaircasePairs()` | yes | `GliderPRO/Sources/HouseLegal.c:1197` | no (warn only) |
| 12 | `if (CountStarsInHouse() < 1)` warn | yes | `GliderPRO/Sources/HouseLegal.c:1203` | no |

**Steps 4 and 5 are unconditional.** That is the reason no shipped house contains a
single placeholder room and 21 of 22 are exactly `866 + 348*nRooms`.

## 10.2 Step 1 — `WrapBannerAndTrailer`

`GliderPRO/Sources/HouseLegal.c:604-615`:

```c
WrapText((*thisHouse)->banner,  40);      // HouseLegal.c:611
WrapText((*thisHouse)->trailer, 64);      // HouseLegal.c:612
```

`WrapText(theText, maxChars)` (`GliderPRO/Sources/StringUtils.c:216-250`) is a
destructive, in-place greedy word-wrap that **overwrites space characters with
carriage returns**:

```
 1. lastChar <- theText[0]; count <- 0
 2. repeat:
 3.   chars <- 0; foundEdge <- false; foundSpace <- false
 4.   repeat:
 5.     count++; chars++
 6.     if theText[count] == 0x0D (kReturnKeyASCII): foundEdge <- true
 7.     elif theText[count] == 0x20 (kSpaceBarASCII): foundSpace <- true; spaceIs <- count
 8.   while (count < lastChar) and (chars < maxChars) and (not foundEdge)
 9.   if (not foundEdge) and (count < lastChar) and foundSpace:
10.     theText[spaceIs] <- 0x0D
11.     count <- spaceIs + 1
12. while (count < lastChar)
```

So the banner is hard-wrapped at 40 characters and the trailer at 64, using
0x0D (`\r`) as the line break. **The wrap is destructive and idempotent-ish**: a
re-save of an already-wrapped string produces the same bytes, because step 6
short-circuits on the existing `\r`. `kReturnKeyASCII` = `0x0D` and
`kSpaceBarASCII` = `0x20` (`GliderPRO/Headers/Externs.h:37`, `45`).

A word longer than `maxChars` with no space is left un-wrapped (step 9 requires
`foundSpace`), and it will be clipped when drawn.

Observed `Demo House` banner (length byte 0x6F = 111 at abs 16), with `.` marking
0x0D:

```
Welcome to the Demo House!.This is a small begi nner house that.acts as a sort o...
```

— the `\r` after `House!` is authored, the one after `that` is a wrap. Rendering
draws one line per `\r` at 20-px intervals:
`MoveTo(topLeft.h + 16, topLeft.v + 32 + (count * 20))`
(`GliderPRO/Sources/Banner.c:136`).

## 10.3 Step 2 — `ValidateNumberOfRooms`

`GliderPRO/Sources/HouseLegal.c:621-640`:

```
 1. reportsRooms <- (long)(*thisHouse)->nRooms                          // HouseLegal.c:629
 2. countedRooms <- (GetHandleSize(thisHouse) - sizeof(houseType)) / sizeof(roomType)   // HouseLegal.c:630-631
 3. if (reportsRooms != countedRooms):
 4.    (*thisHouse)->nRooms <- (short)countedRooms                       // HouseLegal.c:634
 5.    numberRooms <- (*thisHouse)->nRooms                              // HouseLegal.c:635
 6.    houseErrors++                                                    // HouseLegal.c:636
```

**The handle size wins; the stored count is corrected to match.** This is the
authoritative rule for resolving a `nRooms` / file-length disagreement, and the
one a Go loader should adopt (with `sizeof(houseType)` = 866).

## 10.4 Step 3 — `CheckDuplicateFloorSuite`

`GliderPRO/Sources/HouseLegal.c:646-682`:

```
 1. pidgeonHoles <- NewPtrClear(8192 bytes)        // kRoomsTimesSuites, HouseLegal.c:648,653
      nil -> return (silently skip)
 2. for i in 0 .. nRooms-1:
 3.   if rooms[i].suite == kRoomIsEmpty: continue
 4.   bitPlace <- ((rooms[i].floor + 7) * 128) + rooms[i].suite       // HouseLegal.c:665-666
 5.   if (bitPlace < 0 || bitPlace >= 8192): DebugStr("\pBlew array") // HouseLegal.c:667-668  (no bail-out!)
 6.   if pidgeonHoles[bitPlace] != 0:
 7.      houseErrors++; rooms[i].suite <- kRoomIsEmpty                 // HouseLegal.c:671-672
 8.   else: pidgeonHoles[bitPlace]++
```

Note step 5: on an out-of-range `bitPlace` the original prints a debug string and
then **still writes `pidgeonHoles[bitPlace]`** — a genuine out-of-bounds write.
This is reachable because `ValidateRoomNumbers` (which would have deleted the
offending room) runs *after* this step. A Go port must range-check and skip.

The keeper is the **lowest-indexed** room at a given cell; later duplicates are
deleted.

## 10.5 Steps 4 and 5 — `CompressHouse` and `LopOffExtraRooms`

`CompressHouse` (`GliderPRO/Sources/HouseLegal.c:688-734`):

```
 1. wasFirstRoom <- (*thisHouse)->firstRoom                     // HouseLegal.c:697
 2. compressing <- true; roomNumber <- nRooms - 1                // HouseLegal.c:698-699
 3. do:
 4.   if rooms[roomNumber].suite != kRoomIsEmpty:
 5.     probe <- 0; probing <- true
 6.     do:
 7.       if rooms[probe].suite == kRoomIsEmpty:
 8.          rooms[probe] <- rooms[roomNumber]                   // HouseLegal.c:710
 9.          rooms[roomNumber].suite <- kRoomIsEmpty             // HouseLegal.c:711
10.          if roomNumber == wasFirstRoom: firstRoom <- probe   // HouseLegal.c:712-713
11.          if roomNumber == wasRoom:      wasRoom   <- probe   // HouseLegal.c:714-715
12.          probing <- false
13.       probe++
14.       if probing and probe >= roomNumber:
15.          probing <- false; compressing <- false
16.     while probing
17.   roomNumber--
18.   if roomNumber <= 0: compressing <- false
19. while compressing
```

It moves live rooms from the **end** into holes near the **front**, so it reverses
relative order of the moved rooms. `firstRoom` is patched; **link `where` fields
need no patching because they store grid coordinates, not indices** (Part 7.1) —
this is exactly why the format encodes links that way.

Caveat: `savedGame.roomNumber` is an **index** and is *not* patched. A saved game
inside a house that gets compressed will resume in the wrong room. This is a
latent bug in the original; a port should reproduce the storage but may reasonably
choose to patch it.

`LopOffExtraRooms` (`GliderPRO/Sources/HouseLegal.c:740-779`):

```
 1. count <- 0; r <- nRooms
 2. do:
 3.   r--
 4.   if rooms[r].suite == kRoomIsEmpty: count++
 5.   else: r <- 0                    // sentinel that breaks the loop
 6. while r > 0
 7. if count > 0:
 8.   r <- nRooms - count
 9.   newSize <- sizeof(houseType) + sizeof(roomType) * (long)r     // HouseLegal.c:765
10.   HUnlock; SetHandleSize(thisHouse, newSize); HLock            // HouseLegal.c:766-767, 774
11.   nRooms -= count                                               // HouseLegal.c:775
12.   numberRooms <- nRooms
```

Note the shape of the loop: `r` is decremented *before* the test, so `rooms[0]` **is**
inspected (a lone empty room in slot 0 is counted and the handle is shrunk to
`sizeof(houseType)` with `nRooms` = 0). The `else r = 0` is a break sentinel, which
means a non-empty room in slot 0 also ends the scan — indistinguishable from running
out of slots, but harmless because `count` is already final. One genuine defect: if
`nRooms` is 0 on entry the first `r--` makes `r` = −1 and `rooms[-1].suite` is read
out of bounds before the `while (r > 0)` test stops the loop. A Go port must guard
`nRooms == 0`.

## 10.6 Step 6 — `ValidateRoomNumbers`

`GliderPRO/Sources/HouseLegal.c:785-828`:

```
 numRooms <- nRooms
 if numRooms < 0:                                     // HouseLegal.c:795
     nRooms <- 0; numRooms <- 0                       // HouseLegal.c:797-798  (silent, no houseErrors++)
 for i in 0 .. numRooms-1:
   if rooms[i].suite == kRoomIsEmpty: continue
   if rooms[i].floor > 56 or rooms[i].floor < -7:      // HouseLegal.c:804-805
       rooms[i].suite <- kRoomIsEmpty                  // HouseLegal.c:807
       houseErrors++                                   // HouseLegal.c:811
   if rooms[i].suite >= 128 or rooms[i].suite < 0:     // HouseLegal.c:814-815
       rooms[i].suite <- kRoomIsEmpty                  // HouseLegal.c:817
       houseErrors++                                   // HouseLegal.c:821
```

So the **legal grid is `floor` ∈ [−7, 56] and `suite` ∈ [0, 127]**, and violations
are punished by deletion, not clamping. All four bounds are hard-coded literals
(`56`, `-7`, `128`, `0`); none is written in terms of `kMaxNumRoomsH` (128),
`kMaxNumRoomsV` (64) or `kNumUndergroundFloors`
(`GliderPRO/Headers/GliderDefines.h:543-544`) even though the values coincide.

This step is also the only place a **negative `nRooms`** is repaired: it is zeroed
in the header without counting as an error. The two `if`s are sequential, not
`else if`, so a room with both a bad `floor` and a bad `suite` increments
`houseErrors` twice.

## 10.7 Steps 7 and 8 — room names

`CountUntitledRooms` (`GliderPRO/Sources/HouseLegal.c:834-851`) counts rooms whose
name equals `"\pUntitled Room"` case-insensitively
(`EqualString(..., false, true)`, `GliderPRO/Sources/HouseLegal.c:846`) and only
warns.

`CheckRoomNameLength` (`GliderPRO/Sources/HouseLegal.c:857-879`):

```
 for i in 0 .. nRooms-1:
   rooms[i].unusedByte <- 0                     // HouseLegal.c:868   -- ALL rooms, even empty ones
   if rooms[i].suite != kRoomIsEmpty and rooms[i].name[0] > 27:   // HouseLegal.c:870-871
      rooms[i].name[0] <- 27                    // HouseLegal.c:873
      houseErrors++                             // HouseLegal.c:874
```

Only the **length byte** is clamped; bytes 1..27 are left as they were, so
truncation is non-destructive and the residue beyond the length stays in the file.
`Str27` is 28 bytes (1 length + 27 chars), so a length byte of 28..255 would read
into the following `bounds` field — hence the clamp.

Observed: name length bytes across the corpus span 1..27; none exceeds 27.

## 10.8 Step 9 — `MakeSureNumObjectsJives`

See 4.7. Recount and overwrite; live objects need not be contiguous.

## 10.9 Step 10 — `KeepAllObjectsLegal` / `KeepObjectLegal`

`KeepAllObjectsLegal` (`GliderPRO/Sources/HouseLegal.c:919-954`):

```
 for i in 0 .. nRooms-1:
   if rooms[i].suite == kRoomIsEmpty: continue
   ForceThisRoom(i)                              // load into the working copy
   for h in 0 .. kMaxRoomObs-1:
     objActive <- h
     if thisRoom->objects[h].what != kObjectIsEmpty:
        if !KeepObjectLegal(): report error       // HouseLegal.c:939-947
   CopyThisRoomToRoom()                           // write back
```

`KeepObjectLegal` (`GliderPRO/Sources/HouseLegal.c:42-597`) is the single largest
mutator of object data. Its structure:

```
 1. unchanged <- true                                                            // HouseLegal.c:50
 2. #ifndef COMPILEDEMO   // the ENTIRE body is compiled out of the demo          // HouseLegal.c:51
 3. theObject <- &thisRoom->objects[objActive]                                    // HouseLegal.c:53
 4. if (objActive == kInitialGliderSelected):     // -2                           // HouseLegal.c:55
      HGetState/HLock(thisHouse)                                                  // HouseLegal.c:57-58
      clamp (*thisHouse)->initial.h to [0, kRoomWide - kGliderWide]  = [0, 464]   // HouseLegal.c:59,63-64
      clamp (*thisHouse)->initial.v to [0, kTileHigh - kGliderHigh]  = [0, 302]   // HouseLegal.c:61,65-66
      HSetState; return true                                                      // HouseLegal.c:67-68
 5. QSetRect(&roomRect, 0, 0, kRoomWide, kTileHigh)   // 0,0,512,322              // HouseLegal.c:71
 6. switch (theObject->what): ... nine per-group blocks (below) ...               // HouseLegal.c:73-592
 7. #endif                                                                       // HouseLegal.c:594
 8. return unchanged                                                             // HouseLegal.c:596
```

Two things to internalise before reading the per-group rules:

- **The return value is `unchanged`, not "is legal".** `KeepAllObjectsLegal`
  reports an error precisely when `KeepObjectLegal` returned `false`, i.e. when it
  *changed* something (`GliderPRO/Sources/HouseLegal.c:939-947`). "House has
  errors" therefore means "the checker had to repair something".
- **Every group starts the same way**:
  `GetObjectRect(&thisRoom->objects[objActive], &bounds)` followed by
  `if (ForceRectInRect(&bounds, &roomRect))`, and the fields are only rewritten
  from `bounds` when the object actually stuck out of the 512 × 322 room. An
  object already inside the room falls through to the group's specific rules with
  its stored fields untouched.
- The unqualified constants: `kRoomWide` = 512, `kTileHigh` = 322
  (`GliderPRO/Headers/GliderDefines.h:498-499`), `kGliderWide` = 48,
  `kGliderHigh` = 20 (`GliderPRO/Headers/GliderDefines.h:548-549`), giving the
  initial-glider clamps [0, 464] × [0, 302].

### 10.9.1 Blowers (`0x01`–`0x10`, `data.a`)

Case list at `GliderPRO/Sources/HouseLegal.c:75-90` — exactly the 16 codes
`kFloorVent`, `kCeilingVent`, `kFloorBlower`, `kCeilingBlower`, `kSewerGrate`,
`kLeftFan`, `kRightFan`, `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ`,
`kInvisBlower`, `kGrecoVent`, `kSewerBlower`, `kLiftArea`.

| # | Rule | Citation |
|---|---|---|
| 1 | if forced in: `topLeft.h <- bounds.left`, `topLeft.v <- bounds.top` | `GliderPRO/Sources/HouseLegal.c:92-95` |
| 2 | if forced in **and** `what == kLiftArea`: `distance <- RectWide(&bounds)`, `tall <- RectTall(&bounds) / 2` | `GliderPRO/Sources/HouseLegal.c:97-101` |
| 3 | `kStubby` only: if `topLeft.h % 2 == 0` (**even**) → `topLeft.h--` | `GliderPRO/Sources/HouseLegal.c:103-107` |
| 4 | `kTaper`, `kCandle`, `kTiki`, `kBBQ` only: if `topLeft.h % 2 != 0` (**odd**) → `topLeft.h--` | `GliderPRO/Sources/HouseLegal.c:108-114` |
| 5 | `kFloorVent`: if `topLeft.v != kFloorVentTop (305)` → set it **and** `distance += 2` | `GliderPRO/Sources/HouseLegal.c:115-119` |
| 6 | `kFloorBlower`: if `topLeft.v != kFloorBlowerTop (304)` → set it **and** `distance += 2` | `GliderPRO/Sources/HouseLegal.c:120-125` |
| 7 | `kSewerGrate`: if `topLeft.v != kSewerGrateTop (303)` → set it **and** `distance += 2` | `GliderPRO/Sources/HouseLegal.c:126-131` |
| 8 | `kFloorTrans`: if `topLeft.v != kFloorTransTop (302)` → set it **and** `distance += 2` — **DEAD CODE** | `GliderPRO/Sources/HouseLegal.c:132-137` |
| 9 | `if (ObjectHasHandle(&direction, &dist))` → direction-specific clearance | `GliderPRO/Sources/HouseLegal.c:138-193` |

Rule 8 is **unreachable**: `kFloorTrans` (`0x35`) is not in this case list — it is
handled by the transport block at `GliderPRO/Sources/HouseLegal.c:286`. A Go port
should not implement it (and should not be surprised that `kFloorTrans` objects in
the corpus have `topLeft.v` values other than 302).

Rules 5–8 deliberately **do not set `unchanged = false`**, so those repairs are
silent — they never raise a "house has errors" report even though they mutate the
file. Note also that `distance += 2` is cumulative: an object whose `v` is wrong
gains 2 to its blow distance on *every* check pass until `v` is correct, which it
is immediately after the first pass. So the increment fires at most once per
authoring mistake.

Rule 9's switch keys on the `kAbove`/`kToRight`/`kBelow`/`kToLeft` direction enum
= 1/2/3/4 (`GliderPRO/Headers/GliderDefines.h:210-213`) returned by
`ObjectHasHandle`, **not** on `data.a.vector`:

| direction | computation | clamp | Citation |
|---|---|---|---|
| `kAbove` (1) | `dist = bounds.top - dist` | for `kFloorVent`, `kFloorBlower`, `kTaper`, `kCandle`, `kStubby`: if `dist < 36` then `distance += dist - 36`; for every other blower: if `dist < 0` then `distance += dist` | `GliderPRO/Sources/HouseLegal.c:142-164` |
| `kToRight` (2) | `dist = bounds.right + dist` | if `dist > kRoomWide (512)` then `distance += (512 - dist)` | `GliderPRO/Sources/HouseLegal.c:166-173` |
| `kBelow` (3) | `dist = bounds.bottom + dist` | if `dist > kTileHigh (322)` then `distance += (322 - dist)` | `GliderPRO/Sources/HouseLegal.c:175-182` |
| `kToLeft` (4) | `dist = bounds.left - dist` | if `dist < 0` then `distance += dist` | `GliderPRO/Sources/HouseLegal.c:184-191` |

The `36` in the `kAbove` case is a hard-coded literal, not a named constant: the
five floor-standing flame/vent objects must keep 36 px of clearance above their
blow column.

The odd/even `topLeft.h` forcing is a 68k QuickDraw word-alignment optimization.
It has no gameplay effect but **must be reproduced on write** for byte-exactness.

Observed confirmation: every `kFloorVent` in the corpus has `topLeft.v` = 305,
including `Demo House` room[0] objects 0, 3, 5.

### 10.9.2 Furniture (`0x11`–`0x1F`, `data.b`)

Case list at `GliderPRO/Sources/HouseLegal.c:196-210` — the 15 codes `kTable`,
`kShelf`, `kCabinet`, `kFilingCabinet`, `kWasteBasket`, `kMilkCrate`, `kCounter`,
`kDresser`, `kDeckTable`, `kStool`, `kTrunk`, `kInvisObstacle`, `kManhole`,
`kBooks`, `kInvisBounce`.

| # | Rule | Citation |
|---|---|---|
| 1 | if forced in: `data.b.bounds <- bounds` (the whole rect) | `GliderPRO/Sources/HouseLegal.c:211-216` |
| 2 | `kManhole` only: if `(bounds.left - 3) % 64 != 0` then `bounds.left <- ((bounds.left + 29) / 64) * 64 + 3` and `bounds.right <- bounds.left + RectWide(&srcRects[kManhole])` | `GliderPRO/Sources/HouseLegal.c:217-226` |

Rule 2 snaps manholes to the **tile grid with a 3-px bias**: legal lefts are
3, 67, 131, 195, 259, 323, 387, 451 (i.e. `64*n + 3` for n = 0..7), and the
`+ 29` makes it round to the nearest tile rather than down. This is the only
grid-snapping rule in the whole checker.

Note what is **not** here: there is no thickness enforcement. `kTableThick` = 8,
`kShelfThick` = 6 (`GliderPRO/Headers/GliderDefines.h:439-440`),
`kStoolThick` = 25 (`GliderPRO/Sources/ObjectRects.c:18`),
`kDresserBottom` = 293 and `kCounterBottom` = 304
(`GliderPRO/Headers/GliderDefines.h:475-476`) are applied when the object is
*created* (`GliderPRO/Sources/ObjectAdd.c`) and when its rect is *derived*
(`GliderPRO/Sources/ObjectRects.c:39-270`), not by `KeepObjectLegal`. A furniture
rect that is inside the room but the wrong thickness therefore survives a save
unchanged.

Observed anyway: `Demo House` room[1] obj[1] is a `kShelf` with
`bounds.top` = 152 and `bounds.bottom` = 158 — height exactly `kShelfThick` = 6.
`Demo House` room[0] obj[1] is a `kDresser` with `bounds.bottom` = 293 =
`kDresserBottom`.

### 10.9.3 Prizes (`0x21`–`0x2F`, `data.c`)

Case list at `GliderPRO/Sources/HouseLegal.c:229-243` — the 15 codes `kRedClock`,
`kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`, `kBattery`, `kBands`,
`kGreaseRt`, `kGreaseLf`, `kFoil`, `kInvisBonus`, `kStar`, `kSparkle`, `kHelium`,
`kSlider`.

| # | Rule | Citation |
|---|---|---|
| 1 | if forced in: `topLeft.h <- bounds.left`, `topLeft.v <- bounds.top` | `GliderPRO/Sources/HouseLegal.c:244-250` |
| 2 | `kGreaseRt`: if `bounds.right + length > 512` then `length <- 512 - bounds.right` | `GliderPRO/Sources/HouseLegal.c:251-256` |
| 3 | else `kGreaseLf`: if `bounds.left - length < 0` then `length <- bounds.left` | `GliderPRO/Sources/HouseLegal.c:257-262` |
| 4 | else `kSlider`: if `bounds.left + length > 512` then `length <- 512 - bounds.left` | `GliderPRO/Sources/HouseLegal.c:263-268` |
| 5 | **all** prizes: if `topLeft.h % 2 != 0` (**odd**) → `topLeft.h--` | `GliderPRO/Sources/HouseLegal.c:269-273` |
| 6 | all prizes **except `kStar`**: if `length % 2 != 0` (**odd**) → `length--` | `GliderPRO/Sources/HouseLegal.c:274-279` |

Rules 2, 3, 4 are a single `if / else if / else if` chain, so at most one fires —
harmless here because a given object can only be one of the three codes.

Rule 5 forces `topLeft.h` **even** for every prize (the decrement fires when it is
odd). Rule 6 forces `length` **even** for 14 of the 15 codes; `kStar` is exempt
because it does not use `length` at all. Note that rules 5 and 6 do not depend on
`ForceRectInRect`, so they apply to every prize on every check pass.

Observed: `Demo House` room[1] obj[8] is a `kGreaseRt` with `length` = 166 (even),
and its `topLeft.h` is even.

### 10.9.4 Transports / doors / windows (`0x31`–`0x40`, `data.d`)

Case list at `GliderPRO/Sources/HouseLegal.c:282-297` — the 16 codes `kUpStairs`,
`kDownStairs`, `kMailboxLf`, `kMailboxRt`, `kFloorTrans`, `kCeilingTrans`,
`kDoorInLf`, `kDoorInRt`, `kDoorExRt`, `kDoorExLf`, `kWindowInLf`, `kWindowInRt`,
`kWindowExRt`, `kWindowExLf`, `kInvisTrans`, `kDeluxeTrans`.

| # | Rule | Citation |
|---|---|---|
| 1 | if forced in: `topLeft.h <- bounds.left`, `topLeft.v <- bounds.top` | `GliderPRO/Sources/HouseLegal.c:298-303` |
| 2 | if forced in **and** `what == kDeluxeTrans`: `data.d.tall <- ((RectWide(&bounds) / 4) << 8) + (RectTall(&bounds) / 4)` | `GliderPRO/Sources/HouseLegal.c:304-308` |
| 3 | `kDoorInLf`/`kDoorInRt` pair: pick a side and **rewrite `what`** | `GliderPRO/Sources/HouseLegal.c:310-324` |
| 4 | `kDoorExRt`/`kDoorExLf` pair: same | `GliderPRO/Sources/HouseLegal.c:325-339` |
| 5 | `kWindowInLf`/`kWindowInRt` pair: same | `GliderPRO/Sources/HouseLegal.c:340-354` |
| 6 | `kWindowExRt`/`kWindowExLf` pair: same | `GliderPRO/Sources/HouseLegal.c:355-369` |
| 7 | `kInvisTrans` only: if `topLeft.v + tall > kTileHigh (322)` then `tall <- 322 - topLeft.v` | `GliderPRO/Sources/HouseLegal.c:371-378` |
| 8 | `kInvisTrans` only: if `wide < 0` then `wide <- 0` | `GliderPRO/Sources/HouseLegal.c:379-384` |

Rule 2 is the **only** writer of the `kDeluxeTrans` packed size, and it only runs
when the rect had to be forced back inside the room. A `kDeluxeTrans` already
inside the room keeps whatever `tall` the editor wrote
(`GliderPRO/Sources/ObjectInfo.c:2117-2145`).

Rules 3–6 are more than a snap — **they rewrite the `what` code.** Each is the
same shape (shown for rule 3):

```c
if (theObject->data.d.topLeft.h +
        HalfRectWide(&srcRects[kDoorInLf]) > (kRoomWide / 2))   // > 256
{
    theObject->data.d.topLeft.h = kDoorInRtLeft;
    theObject->what = kDoorInRt;
}
else
{
    theObject->data.d.topLeft.h = kDoorInLfLeft;
    theObject->what = kDoorInLf;
}
```

So a door or window is snapped to the left or right wall depending on which half
of the room its **centre** falls in, and its `what` is changed to match. There are
only ever eight legal positions:

| code | forced `topLeft.h` constant | value | Constant defined at | Assigned at |
|---|---|---|---|---|
| `kDoorInLf` (`0x37`) | `kDoorInLfLeft` | **0** | `GliderPRO/Headers/GliderDefines.h:484` | `GliderPRO/Sources/HouseLegal.c:321` |
| `kDoorInRt` (`0x38`) | `kDoorInRtLeft` | **368** | `GliderPRO/Headers/GliderDefines.h:485` | `GliderPRO/Sources/HouseLegal.c:316` |
| `kDoorExLf` (`0x3A`) | `kDoorExLfLeft` | **0** | `GliderPRO/Headers/GliderDefines.h:487` | `GliderPRO/Sources/HouseLegal.c:336` |
| `kDoorExRt` (`0x39`) | `kDoorExRtLeft` | **496** | `GliderPRO/Headers/GliderDefines.h:488` | `GliderPRO/Sources/HouseLegal.c:331` |
| `kWindowInLf` (`0x3B`) | `kWindowInLfLeft` | **0** | `GliderPRO/Headers/GliderDefines.h:490` | `GliderPRO/Sources/HouseLegal.c:351` |
| `kWindowInRt` (`0x3C`) | `kWindowInRtLeft` | **492** | `GliderPRO/Headers/GliderDefines.h:491` | `GliderPRO/Sources/HouseLegal.c:346` |
| `kWindowExLf` (`0x3E`) | `kWindowExLfLeft` | **0** | `GliderPRO/Headers/GliderDefines.h:493` | `GliderPRO/Sources/HouseLegal.c:366` |
| `kWindowExRt` (`0x3D`) | `kWindowExRtLeft` | **496** | `GliderPRO/Headers/GliderDefines.h:494` | `GliderPRO/Sources/HouseLegal.c:361` |

Note the asymmetry: interior doors sit at 0 / 368 while exterior doors and exterior
windows sit at 0 / 496 and interior windows at 0 / 492. The "Lf" variants are all
at `h = 0`.

These rules run **unconditionally** (outside the `ForceRectInRect` guard), so every
door and window in a checked house sits at exactly one of those four distinct
values (0, 368, 492, 496 — eight constants, but four distinct values, since all four
"Lf" constants are 0 and `kDoorExRtLeft` = `kWindowExRtLeft` = 496). Confirmed
empirically: all 167 doors and windows in the
corpus have an **even** `topLeft.h` — 0, 368, 492 and 496 are all even, and the
parity survey in 12.6.1 shows zero odd instances for any of the eight codes.

Rule 8: `wide` is declared `Byte` (`GliderPRO/Headers/GliderStructs.h:41`), i.e.
`unsigned char` on the Mac, so `wide < 0` can never be true and the branch is
**dead**. A Go port must not "fix" this into a real clamp, or `kInvisTrans` widths
> 127 would be silently zeroed.

Rule 7 clamps only `kInvisTrans`. `kDeluxeTrans`'s height (packed into the low byte
of `tall`) is **not** clamped here.

### 10.9.5 Switches (`0x41`–`0x49`, `data.e`)

Case list at `GliderPRO/Sources/HouseLegal.c:387-395` — all nine codes
`kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch`,
`kInvisSwitch`, `kTrigger`, `kLgTrigger`, `kSoundTrigger`.

| # | Rule | Citation |
|---|---|---|
| 1 | if forced in: `topLeft.h <- bounds.left`, `topLeft.v <- bounds.top` | `GliderPRO/Sources/HouseLegal.c:396-402` |
| 2 | if `topLeft.h % 2 != 0` (**odd**) → `topLeft.h--`, i.e. force **even** | `GliderPRO/Sources/HouseLegal.c:403-407` |

That is the entire block — the shortest of the nine. Note `kSoundTrigger` **is**
included; it is excluded only from the *link* enumerators
(`GliderPRO/Sources/House.c:279-286`), so its `topLeft.h` gets the same
even-forcing as a real switch. Confirmed empirically: all 1685 switch-group objects
in the corpus have an even `topLeft.h` (12.6.1).

### 10.9.6 Lights (`0x51`–`0x58`, `data.f`)

Case list at `GliderPRO/Sources/HouseLegal.c:410-417` — the eight codes
`kCeilingLight`, `kLightBulb`, `kTableLamp`, `kHipLamp`, `kDecoLamp`,
`kFlourescent`, `kTrackLight`, `kInvisLight`.

| # | Rule | Citation |
|---|---|---|
| 1 | if forced in **and** `what` is `kFlourescent` or `kTrackLight`: raise `topLeft.h` to `bounds.left` if lower; raise `topLeft.v` to `bounds.top` if lower; if `topLeft.h + length > bounds.right` then `length <- bounds.right - topLeft.h` | `GliderPRO/Sources/HouseLegal.c:421-431` |
| 2 | if forced in and `what` is any of the other six: `topLeft.h <- bounds.left`, `topLeft.v <- bounds.top` | `GliderPRO/Sources/HouseLegal.c:432-436` |
| 3 | `kFlourescent`/`kTrackLight` only, and only if `bounds.right > 512` or `bounds.left < 0`: clamp `topLeft.h` to [0, 512], clamp the local `bounds` to [0, 512], then **unconditionally** `length <- kRoomWide - bounds.left` | `GliderPRO/Sources/HouseLegal.c:439-464` |

The two span lights get the gentler rule 1 (monotone raises, not assignment) so
that a long fixture is not collapsed onto the room edge. Rule 3's final statement
(`GliderPRO/Sources/HouseLegal.c:463`) is outside all the inner `if`s, so once the
outer condition is met the `length` is rewritten to the full remaining room width
regardless of which sub-clamp fired.

Observed: `Demo House` room[0] obj[8] `kCeilingLight` `topLeft.v` = 4 =
`kCeilingLightTop`; room[1] obj[0] `kFlourescent` `topLeft.v` = 12 =
`kFlourescentTop` with `length` = 308. Note per 5.8 that for `kFlourescent` and
`kTrackLight` `length` is a **width**, not an absolute right edge: that object's
rect is 104 .. 104 + 308 = 412. That is exactly what rules 1 and 3 assume when they
compare `topLeft.h + length` against `bounds.right` and assign
`kRoomWide - bounds.left`.
The six point lights show **mixed** `topLeft.h` parity in the corpus (12.6.1),
confirming that no parity rule applies to them.

### 10.9.7 Appliances (`0x61`–`0x6E`, `data.g`)

Case list at `GliderPRO/Sources/HouseLegal.c:467-480` — the 14 codes `kShredder`,
`kToaster`, `kMacPlus`, `kGuitar`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`,
`kMicrowave`, `kCinderBlock`, `kFlowerBox`, `kCDs`, `kCustomPict`.

| # | Rule | Citation |
|---|---|---|
| 1 | if forced in: `topLeft.h <- bounds.left`, `topLeft.v <- bounds.top` | `GliderPRO/Sources/HouseLegal.c:481-487` |
| 2 | `kToaster` only: if `bounds.top - height < 0` then `height <- bounds.top` | `GliderPRO/Sources/HouseLegal.c:488-493` |
| 3 | `kTV` only: if `topLeft.h % 2 == 0` (**even**) → `topLeft.h--`, i.e. force **odd** | `GliderPRO/Sources/HouseLegal.c:494-499` |
| 4 | `kToaster`, `kMacPlus`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`, `kMicrowave` (**7 codes**): if `topLeft.h % 2 != 0` (**odd**) → `topLeft.h--`, i.e. force **even** | `GliderPRO/Sources/HouseLegal.c:500-511` |

Rule 2 stops a toaster from launching its crumbs above the ceiling: `height` is the
launch height above the object (`GliderPRO/Sources/ObjectInfo.c:1658-1700`).

`kTV` is the **only** appliance forced odd; seven are forced even; the remaining six
(`kShredder`, `kGuitar`, `kCinderBlock`, `kFlowerBox`, `kCDs`, `kCustomPict`) get no
parity fixup at all. Confirmed empirically in 12.6.1: `kTV` is 0 even / 80 odd; the
seven even-forced codes have zero odd instances; the six exempt codes are mixed.

Note that `kCustomPict` — the single most common object in the corpus, 4782
instances — is exempt, so custom art can sit at any `h`.

### 10.9.8 Enemies (`0x71`–`0x79`, `data.h`)

Case list at `GliderPRO/Sources/HouseLegal.c:514-522` — the nine codes `kBalloon`,
`kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`, `kDrip`, `kFish`,
`kCobweb`.

| # | Rule | Citation |
|---|---|---|
| 1 | if forced in: `topLeft.h <- bounds.left`, `topLeft.v <- bounds.top` | `GliderPRO/Sources/HouseLegal.c:523-529` |
| 2 | `kBall`, `kFish`: if `bounds.top - length < 0` then `length <- bounds.top` | `GliderPRO/Sources/HouseLegal.c:530-536` |
| 3 | `kDrip`: if `bounds.bottom + length > kTileHigh (322)` then `length <- 322 - bounds.bottom` | `GliderPRO/Sources/HouseLegal.c:537-542` |
| 4 | `kBalloon`, `kCopterLf`, `kCopterRt`, `kBall`, `kDrip`, `kFish` (**6 codes**): if `topLeft.h % 2 != 0` (**odd**) → `topLeft.h--`, i.e. force **even** | `GliderPRO/Sources/HouseLegal.c:543-553` |

Rules 2 and 3 confirm the direction of travel encoded in `length`: `kBall` and
`kFish` bounce **upward** from their rest position (clamped against `bounds.top`),
while `kDrip` falls **downward** (clamped against `bounds.bottom`).

`kDartLf`, `kDartRt`, and `kCobweb` are exempt from rule 4. Empirically
(12.6.1) `kDartLf` (151) and `kDartRt` (117) nevertheless happen to be entirely
even — an authoring habit, not an enforced invariant — while `kCobweb` is genuinely
mixed (42 even / 44 odd), which proves the exemption is real.

### 10.9.9 Clutter (`0x81`–`0x8F`, `data.i`)

Case list at `GliderPRO/Sources/HouseLegal.c:556-570` — the 15 codes `kOzma`,
`kMirror`, `kMousehole`, `kFireplace`, `kFlower`, `kWallWindow`, `kBear`,
`kCalendar`, `kVase1`, `kVase2`, `kBulletin`, `kCloud`, `kFaucet`, `kRug`,
`kChimes`.

| # | Rule | Citation |
|---|---|---|
| 1 | **if forced in only**: `data.i.bounds <- bounds` (the whole rect) | `GliderPRO/Sources/HouseLegal.c:571-576` |
| 2 | `kMirror` only: if `bounds.left % 2 != 0` → `bounds.left--`; if `bounds.right % 2 != 0` → `bounds.right--` (force both **even**) | `GliderPRO/Sources/HouseLegal.c:577-589` |

Rule 1 is guarded by `ForceRectInRect`, so a clutter rect that already lies inside
the 512 × 322 room is **not** normalised to the sprite size — it keeps whatever
`bounds` the editor wrote. `GetObjectRect` for clutter is a single statement,
`*itsRect = who->data.i.bounds;`
(`GliderPRO/Sources/ObjectRects.c:255-271`, assignment at line 270) — the stored
rect **is** the object rect, so the "canonical sprite size" only enters when the
object is first added. Clutter is therefore the one group whose on-disk rectangle a
level author can freely resize.

Rule 2 forces the mirror's horizontal extent to even coordinates. Like the other
parity rules it is a 68k QuickDraw alignment accommodation (the mirror is the only
clutter object drawn with a live `CopyBits` of the room behind it); no gameplay
behaviour depends on it, but a port that re-saves must reproduce it.

Observed, with clutter `bounds` printed by the probe as
**(left, top, right, bottom)**:

- `Demo House` room[1] obj[7] `kMirror` = (130, 30, 388, 118) — `left` 130 and
  `right` 388 are both **even**, exactly as rule 2 requires. Corpus-wide, all 667
  `kMirror` instances have even `left` and even `right`, with zero exceptions
  (12.6.1) — the strongest single confirmation of any parity rule in the checker.
- `Demo House` room[0] obj[4] `kWallWindow` = (416, 35, 485, 275): 69 × 240 px at
  room position (416, 35), entirely inside the room.
- `Demo House` room[0] obj[6] `kVase2` = (235, 97, 270, 154) — 35 × 57 px.
- `Demo House` room[0] obj[1] `kDresser` (furniture, same rect layout) =
  (217, 154, 341, 293) — `bottom` 293 = `kDresserBottom`.

## 10.10 Step 11 — `CheckForStaircasePairs`

`GliderPRO/Sources/HouseLegal.c:961-1045`:

```
 for each non-empty room i:
   for h in 0 .. kMaxRoomObs-1:
     if objects[h].what == kUpStairs:                        // HouseLegal.c:978
        thisRoomNumber <- i                                  // HouseLegal.c:980
        neighbor <- GetNeighborRoomNumber(kNorthRoom)        // HouseLegal.c:981
        if neighbor == kRoomIsEmpty: warn (string 20)        // HouseLegal.c:982-989
        else if no kDownStairs in neighbor: warn (string 21) // HouseLegal.c:990-1007
     else if objects[h].what == kDownStairs:                 // HouseLegal.c:1009
        thisRoomNumber <- i                                  // HouseLegal.c:1011
        neighbor <- GetNeighborRoomNumber(kSouthRoom)        // HouseLegal.c:1012
        if neighbor == kRoomIsEmpty: warn (string 22)        // HouseLegal.c:1013-1020
        else if no kUpStairs in neighbor: warn (string 23)   // HouseLegal.c:1021-1038
```

This is a **warning only** — no field is changed, and unlike the other steps it
never increments `houseErrors` (there is no `houseErrors++` anywhere in the
function), so a staircase mismatch does not contribute to the final error tally.
Note also that the two `what` tests are an `if / else if` chain and that the routine
clobbers the global `thisRoomNumber` as a side effect of calling
`GetNeighborRoomNumber`. It documents the implicit
staircase link: a `kUpStairs` moves the player to the room at (`suite`,
`floor + 1`) and a `kDownStairs` to (`suite`, `floor - 1`).

Observed: 163 `kUpStairs` and exactly 163 `kDownStairs` across the corpus — the
pairing invariant holds globally, though the check is per-instance.

## 10.11 Step 12 — star count

`CountStarsInHouse()` (`GliderPRO/Sources/Banner.c:89-112`) counts objects with
`what == kStar` (`0x2C`) over all non-empty rooms. Fewer than one triggers a
warning (`GliderPRO/Sources/HouseLegal.c:1203`) because the house would be
unwinnable. It is also the source of `numStarsRemaining`
(`GliderPRO/Sources/Play.c:314`).

Observed: 69 `kStar` objects total; every shipped house has at least one.

## 10.12 Summary of what the format actually guarantees

A house that has survived a version-1.0.4 edit-mode save satisfies:

1. `len(dataFork) == sizeof(houseType) + 348 * nRooms` for that build's
   `sizeof(houseType)` (866 or 868).
2. `version` ∈ [0, 0x02FF]; in practice `0x0200`.
3. `banner` wrapped at 40 chars, `trailer` at 64, both with `\r` breaks.
4. No two non-empty rooms share a (`floor`, `suite`) cell.
5. No placeholder rooms at the end of the array (and, thanks to `CompressHouse`,
   in practice none anywhere).
6. Every non-empty room: `floor` ∈ [−7, 56], `suite` ∈ [0, 127],
   `name[0]` ≤ 27, `unusedByte` = 0, `numObjects` = live-slot count.
7. Every live object's `what` matched a `KeepObjectLegal` case (an unknown code
   falls through the switch and is left alone — so unknown codes are *not*
   guaranteed absent, they are merely absent from the shipped corpus).
8. `topLeft.h` parity, forced tops, and forced sizes per 10.9.

It does **not** guarantee: valid link targets, valid `snd `/`PICT` IDs, `bounds`
consistency with the background range, in-range `who`, `tiles[]` within the
background PICT, or `firstRoom` being in range (`GetFirstRoomNumber` substitutes 0
for an out-of-range value in a *local* variable at
`GliderPRO/Sources/House.c:209-213`, but never writes the correction back to the
header, and it does not check whether the target room is empty).

---

# Part 11 — Version 1 → version 2 migration

`ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746-820`), called from
`QuerySaveChanges` when `wasHouseVersion < kHouseVersion`
(`GliderPRO/Sources/HouseIO.c:610-611`).

```
ConvertHouseVer1To2():
 1. CopyThisRoomToRoom(); wasRoom <- thisRoomNumber                  // House.c:753-754
 2. open a progress window (localized string 13)                     // House.c:755-756
 3. HLock(thisHouse); numRooms <- (*thisHouse)->nRooms                // House.c:761-764
 4. for i in 0 .. numRooms-1:
 5.   if rooms[i].suite == kRoomIsEmpty: continue                     // House.c:766
 6.   ForceThisRoom(i)                                                // House.c:774
 7.   for h in 0 .. kMaxRoomObs-1:
 8.     switch (thisRoom->objects[h].what):
 9.       case kMailboxLf, kMailboxRt, kFloorTrans, kCeilingTrans,
             kInvisTrans, kDeluxeTrans:                               // House.c:779-784
10.         if (data.d.where != -1):                                  // House.c:785
11.            ExtractFloorSuite(data.d.where, &floor, &suite)        // House.c:787
12.            floor += kNumUndergroundFloors    // 8                 // House.c:788
13.            data.d.where <- MergeFloorSuite(floor, suite)          // House.c:789
14.       case kLightSwitch, kMachineSwitch, kThermostat, kPowerSwitch,
             kKnifeSwitch, kInvisSwitch, kTrigger, kLgTrigger:        // House.c:793-800
15.         if (data.e.where != -1):                                  // House.c:801
16.            ExtractFloorSuite(data.e.where, &floor, &suite)        // House.c:803
17.            floor += kNumUndergroundFloors                          // House.c:804
18.            data.e.where <- MergeFloorSuite(floor, suite)          // House.c:805
19.   CopyThisRoomToRoom()                                            // House.c:810
20. (*thisHouse)->version <- kHouseVersion   // 0x0200                // House.c:814
21. restore cursor, close window, ForceThisRoom(wasRoom)              // House.c:817-819
```

The critical subtlety: **step 20 happens after the loop**, so during steps 11 and
16 `(*thisHouse)->version` is still < 0x0200 and `ExtractFloorSuite` therefore
takes its *legacy* branch (`GliderPRO/Sources/Link.c:43-47`):

```
floor = (combo / 100) - 8
suite =  combo % 100
```

while `MergeFloorSuite` always uses the version-2 layout
(`GliderPRO/Sources/Link.c:36`):

```
where = suite * 100 + floor
```

Combining, with step 12's `floor += 8`:

```
oldWhere  ->  suite_old = oldWhere % 100
              floor_old = (oldWhere / 100) - 8
              floor_biased = floor_old + 8 = oldWhere / 100
newWhere  =  (oldWhere % 100) * 100 + (oldWhere / 100)
```

i.e. **the migration is exactly a swap of the two decimal digit-pairs.** Worked
example: a version-1 link with `where` = 906 (floor_biased 9 ⇒ floor 1, suite 6)
becomes `where` = 6 × 100 + 9 = **609**, which the version-2 decoder reads as
suite 6, floor 9 − 8 = 1. Same room. Correct.

Consequences:

- The `+= 8` at steps 12/17 exactly cancels the `- 8` inside the legacy extract;
  it is not an additional bias.
- The migration is **not idempotent**: running it twice on the same house swaps the
  digits back, silently corrupting every link. The guard is the version field, so
  a port must set `version = 0x0200` before any second pass.
- Because the *legacy* format put `suite` in the low two decimal digits, version-1
  houses could only reference suites 0..99. The swap gives version 2 the full
  0..127 suite range at the cost of limiting `floor + 8` to 0..99 (fine: the legal
  maximum is 64).
- `kSoundTrigger` (`0x49`) is **not** in either case list, so its `snd ` ID is
  correctly left alone. Good.
- `kUpStairs`/`kDownStairs`/doors/windows are also absent — correct, they have no
  `where`.

`ShiftWholeHouse` (`GliderPRO/Sources/House.c:824-859`) — which would have shifted
every room's grid coordinates — is a **stub with an empty inner loop** in 1.0.4.

---

# Part 12 — Empirical verification

All values below were produced by
`tools/probe_house.py` against the 22 BinHex houses in
`GliderPRO/Houses/`. Every BinHex stream was
CRC-verified (data fork and resource fork) before parsing, so the byte sources are
known-good.

## 12.1 `Demo House` header hexdump

Absolute offsets 0x000–0x05F of the data fork:

```
00000000  02 00 00 00 2C 53 B0 41 00 00 00 00 00 6B 00 31  |....,S.A.....k.1|
00000010  6F 57 65 6C 63 6F 6D 65 20 74 6F 20 74 68 65 20  |oWelcome to the |
00000020  44 65 6D 6F 20 48 6F 75 73 65 21 0D 54 68 69 73  |Demo House!.This|
00000030  20 69 73 20 61 20 73 6D 61 6C 6C 20 62 65 67 69  | is a small begi|
00000040  6E 6E 65 72 20 68 6F 75 73 65 20 74 68 61 74 0D  |nner house that.|
00000050  61 63 74 73 20 61 73 20 61 20 73 6F 72 74 20 6F  |acts as a sort o|
```

Decoded against the Part 3.1 offset table:

| Offset | Bytes | Field | Value |
|---|---|---|---|
| 0 | `02 00` | `version` | 0x0200 → "2.0" |
| 2 | `00 00` | `unusedShort` | 0 |
| 4 | `2C 53 B0 41` | `timeStamp` | 0x2C53B041 = 743682113; **bit 0 = 1 ⇒ locked** |
| 8 | `00 00 00 00` | `flags` | 0 ⇒ ward=false, phone=false, **bannerStarCountOn=true** (bit 2 inverted) |
| 12 | `00 6B` | `initial.v` | 107 (≤ 302 ✓) |
| 14 | `00 31` | `initial.h` | 49 (≤ 464 ✓) |
| 16 | `6F` | `banner[0]` | 111 |
| 17 | `57 65 6C ...` | `banner[1..]` | `Welcome to the Demo House!\rThis is a small begi nner house that\racts as a sort o…` |

The `0D` at offset 0x2B and 0x4F are the authored and wrapped line breaks
(Part 10.2).

Remaining header fields (parsed, not hexdumped):
`highScores.banner` = `'The Return of Ozma!'`; `highScores.names[0]` = `'Ozma'`,
`scores[0]` = 7400, `timeStamps[0]` = 2888823855, `levels[0]` = 13;
`names[1..9]` = `'--------------'` with zeros;
`savedGame` = `{version 0x0100, wasStarsLeft 1, timeStamp -1430579948, where.v 11,
where.h 59, score 3000, energy 50, bands 5, roomNumber 15, gliderState 0,
numGliders 0, foil 0, facing 1, showFoil 0}`; `hasGame` = **0** (so that block is
stale residue); `unusedBoolean` = 0; `firstRoom` = 0; `nRooms` = 45.
`866 + 348*45 = 16526` = the observed data-fork length exactly.

## 12.2 `Demo House` rooms 0 and 1

Room 0 at absolute 0x362 = 866:

| Field | Value |
|---|---|
| `name` | `'Air Vents'` |
| `bounds` | 0 (⇒ fall back to `'bnds'` 3000 ⇒ boundCode 4 = right open) |
| `leftStart` / `rightStart` | 0 / 0 |
| `unusedByte` | 0 |
| `visited` | 1 |
| `background` | 3000 |
| `tiles[0..7]` | 0, 1, 2, 3, 4, 5, 6, 7 (identity — the user-art default) |
| `floor` / `suite` | 1 / 63 |
| `openings` | 0 |
| `numObjects` | 10 (matches 10 live slots ✓) |

| slot | `what` | name | decoded fields |
|---|---|---|---|
| 0 | `0x01` | kFloorVent | `topLeft` (h 171, v **305** = `kFloorVentTop`), `distance` 269, `initial` 1, `state` 1, `vector` 1, `tall` 1 |
| 1 | `0x18` | kDresser | `bounds` (l 217, t 154, r 341, b **293** = `kDresserBottom`), `pict` 0 |
| 2 | `0x21` | kRedClock | `topLeft` (310, 137), `state` 0, `initial` 1 |
| 3 | `0x01` | kFloorVent | (430, 305), `distance` 225 |
| 4 | `0x86` | kWallWindow | `bounds` (left 416, top 35, right 485, bottom 275) — on disk `00 23 01 A0 01 13 01 E5`, i.e. top 35, left 416, bottom 275, right 485 |
| 5 | `0x01` | kFloorVent | (48, 305), `distance` 224 |
| 6 | `0x8A` | kVase2 | `bounds` (235, 97, 270, 154) |
| 7 | `0x85` | kFlower | `bounds` (246, 64, 280, 99), `pict` 2 |
| 8 | `0x51` | kCeilingLight | `topLeft` (137, **4** = `kCeilingLightTop`), `length` 64 |
| 9 | `0x6E` | kCustomPict | `topLeft` (33, 61), `height` **10014** → `PICT` 10014 exists in this house, named `'Quilt on Wall'` |

### 12.2.1 Raw object bytes, decoded field by field

Every one of the following 12-byte records was dumped straight out of the data
fork. Object slot `k` of room `r` lives at `866 + 348*r + 60 + 12*k`.

| Room / slot | Absolute offset | Raw 12 bytes | Decode |
|---|---|---|---|
| r0 s0 | 926 (0x39E) | `00 01 01 31 00 AB 01 0D 01 01 01 01` | `what` 0x01 `kFloorVent`; `topLeft.v` 0x0131 = **305** = `kFloorVentTop`; `topLeft.h` 0x00AB = 171; `distance` 0x010D = 269; `initial` 1; `state` 1; `vector` 1; `tall` 1 |
| r0 s1 | 938 (0x3AA) | `00 18 00 9A 00 D9 01 25 01 55 00 00` | `what` 0x18 `kDresser`; `bounds.top` 154; `.left` 217; `.bottom` 0x0125 = **293** = `kDresserBottom`; `.right` 0x0155 = 341; `pict` 0 |
| r0 s2 | 950 (0x3B6) | `00 21 00 89 01 36 00 00 00 00 00 01` | `what` 0x21 `kRedClock`; `topLeft.v` 137; `topLeft.h` 0x0136 = 310 (**even**, per prize rule 5); `length` 0; `points` 0; **`state` 0; `initial` 1** — the `bonusType` field-order proof |
| r0 s4 | 974 (0x3CE) | `00 86 00 23 01 A0 01 13 01 E5 00 00` | `what` 0x86 `kWallWindow`; `bounds.top` 35; `.left` 0x01A0 = 416; `.bottom` 0x0113 = 275; `.right` 0x01E5 = 485; `pict` 0 |
| r0 s6 | 998 (0x3E6) | `00 8A 00 61 00 EB 00 9A 01 0E 00 00` | `what` 0x8A `kVase2`; `bounds` top 97, left 235, bottom 154, right 270; `pict` 0 |
| r0 s7 | 1010 (0x3F2) | `00 85 00 40 00 F6 00 63 01 18 00 02` | `what` 0x85 `kFlower`; `bounds` top 64, left 246, bottom 99, right 280; **`pict` 2** (a flower *index*, added to `kRadioFlower1`, not a resource ID) |
| r0 s8 | 1022 (0x3FE) | `00 51 00 04 00 89 00 40 00 00 01 01` | `what` 0x51 `kCeilingLight`; `topLeft.v` **4** = `kCeilingLightTop`; `topLeft.h` 137 (odd — lights have no parity rule); `length` 0x0040 = 64; `byte0` 0; `byte1` 0; `initial` 1; `state` 1 |
| r0 s9 | 1034 (0x40A) | `00 6E 00 3D 00 21 27 1E 00 00 01 01` | `what` 0x6E `kCustomPict`; `topLeft.v` 61; `topLeft.h` 33; **`height` 0x271E = 10014** — a `PICT` **resource ID**, and `PICT` 10014 (`'Quilt on Wall'`) is present in this house's resource fork; `byte0` 0; `delay` 0; `initial` 1; `state` 1 |
| r1 s0 | 1274 (0x4FA) | `00 56 00 0C 00 68 01 34 00 00 01 01` | `what` 0x56 `kFlourescent`; `topLeft.v` **12** = `kFlourescentTop`; `topLeft.h` 104; `length` 0x0134 = 308 (a **width**, so the fixture spans h 104..412); `initial` 1; `state` 1 |
| r1 s7 | 1358 (0x54E) | `00 82 00 1E 00 82 00 76 01 84 00 00` | `what` 0x82 `kMirror`; `bounds.top` 30; `.left` 0x0082 = **130 (even)**; `.bottom` 118; `.right` 0x0184 = **388 (even)** — `GliderPRO/Sources/HouseLegal.c:577-589` |

Three independent facts fall straight out of this table:

1. **`Point` is v-then-h.** In r0 s0 the first short is 305 and the second 171. Only
   v-first puts the constant `kFloorVentTop` = 305 in `topLeft.v`. Same for r0 s8
   (4 = `kCeilingLightTop`) and r1 s0 (12 = `kFlourescentTop`).
2. **`Rect` is top-left-bottom-right.** In r0 s1 the third short is 293 =
   `kDresserBottom`; in r1 s7 the second and fourth shorts (130, 388) are the two
   that `KeepObjectLegal` forces even for a mirror.
3. **`bonusType` stores `state` before `initial`.** r0 s2's last two bytes are
   `00 01`; a red clock that has never been collected must have `initial` = 1, and
   `SetObjectsToDefaults` copies `initial` → `state`
   (`GliderPRO/Sources/Play.c:653-654`), so `state` = 0 is exactly the expected
   post-run residue.

Room 1 at absolute 0x4BE = 866 + 348 = 1214:

`name` `'Some Prizes'`, `background` 3001, `floor` 1, `suite` 64, `numObjects` 18.
Selected objects: slot 0 `0x56` kFlourescent `topLeft` (104, **12** =
`kFlourescentTop`) `length` 308; slot 1 `0x12` kShelf `bounds` (106, 152, 413,
**158**) → height 6 = `kShelfThick`; slots 2–4 `0x1A` kStool; slot 5 `0x26`
kBattery; slots 6, 9, 11 `0x01` kFloorVent; slot 7 `0x82` kMirror `bounds`
(**130**, 30, **388**, 118) — left and right both even; slot 8 `0x28` kGreaseRt
`length` 166; slot 10 `0x27` kBands; slots 12, 13, 15, 16, 17 `0x6E` kCustomPict
with `height` 10017 / 10049 / 10036 / 10065 / 10065; slot 14 `0x16` kMilkCrate.

Room 0's `suite` 63 and room 1's `suite` 64 at the same `floor` 1 makes them east
neighbours, matching room 0's right-open `'bnds'`.

## 12.3 Field-order proofs

### `Point` is `{short v; short h}` — v FIRST

`GliderPRO/Headers/GliderStructs.h` declares `Point topLeft;` and the classic Mac
`Point` is `struct { short v; short h; }`. Three independent empirical
confirmations:

1. `Demo House` room[0] slot 0 is a `kFloorVent`, whose `topLeft.v` is forced to
   `kFloorVentTop` = 305 (`GliderPRO/Sources/HouseLegal.c:115-118`). The stored
   shorts at object-relative 2 and 4 are **305** and 171. Only the v-first reading
   puts 305 in `topLeft.v`. All 1458 `kFloorVent` instances in the corpus have 305
   in the first short.
2. `Demo House` room[0] slot 8 is a `kCeilingLight`, `topLeft.v` forced to
   `kCeilingLightTop` = 4. Stored shorts: **4**, 137.
3. `Demo House` room[1] slot 0 is a `kFlourescent`, `topLeft.v` forced to
   `kFlourescentTop` = 12. Stored shorts: **12**, 104.

Additionally the house-level `initial` `Point`: across all 22 houses the first
short spans 7..200 (all ≤ 302, the `initial.v` clamp) while the second spans
30..424 (up to the 464 `initial.h` clamp). Swapping them would put 424 in a field
clamped to 302.

### `Rect` is `{short top, left, bottom, right}`

1. `Demo House` room[0] slot 1 `kDresser`: shorts at rel 2/4/6/8 are 154, 217,
   **293**, 341. `kDresserBottom` = 293 (`GliderPRO/Headers/GliderDefines.h:476`)
   lands in the third short ⇒ `bottom` is third.
2. `Demo House` room[1] slot 1 `kShelf`: 152, 106, **158**, 413. `158 - 152 = 6`
   = `kShelfThick` (`GliderPRO/Headers/GliderDefines.h:440`) ⇒ first is `top`,
   third is `bottom`.

### `floor` at rel 52, `suite` at rel 54

See Part 7.2: `Sampler` room[0] has (52, 54) = (1, 67) and room[1] = (1, 66); the
`kMailboxLf` in room[0] stores `where` = 6609, and only the suite-at-54 reading
makes `6609 = 66*100 + (1+8)` resolve to room[1], whose object slot 2 (`who` = 2)
is itself a `kMailboxLf`.

### `bonusType` has `state` before `initial`

Declared that way at `GliderPRO/Headers/GliderStructs.h:32-33`. Empirically,
`Demo House` room[0] slot 2 (`kRedClock`) has bytes `00 01` at object-relative
10/11 — i.e. `state` = 0, `initial` = 1. A prize with `initial` = 0 would be
invisible from the start, which is not what the shipped level does; and
`SetObjectsToDefaults` copies `initial` → `state`
(`GliderPRO/Sources/Play.c:653-654`), so `state` = 0 is exactly the expected
stale-residue value after a run in which the clock was collected.

## 12.4 The `Sampler` 2-byte tail

`Sampler` data fork is 1564 bytes with `nRooms` = 2; `866 + 2*348 = 1562`. Tail
hexdump at absolute 0x610 = 1552:

```
00000610  01 2F 00 A1 00 20 00 00 01 01 01 01
```

Object slot 23 of room 1 occupies absolute 1550–1561, i.e. bytes
`?? ?? 01 2F 00 A1 00 20 00 00 01 01` (a `kSlider`, `what` = 0x012F… no — the
`01 2F` here is at 1552 and is *inside* slot 23). The two trailing bytes at
**1562–1563** are `01 01`. They are not part of any room record.

Diagnosis (Part 9.4): `Sampler` was last saved by a **PowerPC** build, where
`sizeof(houseType)` = 868 rather than
`offsetof(houseType, rooms)` = 866, so `LopOffExtraRooms`
(`GliderPRO/Sources/HouseLegal.c:765`) sized the handle 2 bytes long. The
`ValidateNumberOfRooms` division `(1564 - 868) / 348 = 2`
(`GliderPRO/Sources/HouseLegal.c:630-631`) is self-consistent under the same
build, and a 68k build computing `(1564 - 866) / 348 = 698 / 348 = 2` also gets 2.
Hence the discrepancy is invisible to the program.

## 12.5 Whole-corpus header survey

22 houses, all with type `'gliH'` and creator `'ozm5'`, all `version` = 0x0200.
`lk` = `timeStamp & 1` (1 = locked); `us` = `unusedShort`; `uB` = `unusedBoolean`;
`sz` = does `len == 866 + 348*nRooms`.

| house | dataLen | rsrcLen | ver | us | flags | lk | initial (h, v) | firstRoom | nRooms | empty | sz | hasGame | uB |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Art Museum | 38798 | 7476159 | 0x0200 | −30082 | 0x00000006 | 1 | (231, 56) | 91 | 109 | 0 | ✓ | 0 | 255 |
| CD Demo House | 72554 | 1612342 | 0x0200 | 0 | 0x00000002 | 1 | (223, 105) | 70 | 206 | 0 | ✓ | 0 | 0 |
| California or Bust! | 6434 | 259542 | 0x0200 | 13107 | 0x00000002 | 0 | (308, 62) | 14 | 16 | 0 | ✓ | 0 | 0 |
| Castle o' the Air | 30446 | 306364 | 0x0200 | 259 | 0x00000000 | 0 | (239, 124) | 33 | 85 | 0 | ✓ | 0 | 30 |
| Davis Station | 23486 | 1871320 | 0x0200 | 222 | 0x00000002 | 1 | (189, 95) | 4 | 65 | 0 | ✓ | 0 | 37 |
| Demo House | 16526 | 491757 | 0x0200 | 0 | 0x00000000 | 1 | (49, 107) | 0 | 45 | 0 | ✓ | 0 | 0 |
| Empty House | 13046 | 2670 | 0x0200 | 0 | 0x00000000 | 0 | (83, 64) | 0 | 35 | 0 | ✓ | 0 | 0 |
| Fun House | 15830 | 662443 | 0x0200 | 26228 | 0x00000000 | 0 | (42, 78) | 29 | 43 | 0 | ✓ | 0 | 0 |
| Grand Prix | 61766 | 1347764 | 0x0200 | 60 | 0x00000000 | 1 | (268, 58) | 127 | 175 | 0 | ✓ | 0 | 185 |
| ImagineHouse PRO II | 97958 | 677770 | 0x0200 | 0 | 0x00000000 | 1 | (211, 35) | 1 | 279 | 0 | ✓ | **1** | 0 |
| In The Mirror | 34622 | 151870 | 0x0200 | 147 | 0x00000000 | 1 | (424, 50) | 6 | 97 | 0 | ✓ | 0 | 0 |
| Land of Illusion | 106310 | 401793 | 0x0200 | 259 | 0x00000002 | 0 | (245, 163) | 43 | 303 | 0 | ✓ | 0 | 30 |
| Leviathan | 165122 | 1901282 | 0x0200 | 0 | 0x00000000 | 1 | (229, 44) | 39 | 472 | 0 | ✓ | 0 | 0 |
| Metropolis | 45062 | 946907 | 0x0200 | 196 | 0x00000000 | 1 | (245, 24) | 8 | 127 | 0 | ✓ | 0 | 14 |
| Nemo's Market | 44018 | 590602 | 0x0200 | 0 | 0x00000002 | 1 | (342, 51) | 0 | 124 | 0 | ✓ | 0 | 0 |
| Rainbow's End | 78470 | 316065 | 0x0200 | 0 | 0x00000002 | 1 | (237, 200) | 30 | 223 | 0 | ✓ | 0 | 0 |
| Sampler | **1564** | 286 | 0x0200 | 0 | 0x00000000 | 0 | (384, 85) | 1 | 2 | 0 | **+2** | 0 | 2 |
| Slumberland | 134150 | 1041112 | 0x0200 | 0 | 0x00000000 | 1 | (361, 7) | 126 | 383 | 0 | ✓ | 0 | 0 |
| SpacePods | 140762 | 636844 | 0x0200 | 0 | 0x00000002 | 1 | (30, 72) | 259 | 402 | 0 | ✓ | 0 | 0 |
| Teddy World | 185654 | 3107348 | 0x0200 | 0 | 0x00000000 | 1 | (362, 41) | 0 | 531 | 0 | ✓ | 0 | 0 |
| The Asylum Pro | 49586 | 494107 | 0x0200 | 0 | 0x00000000 | 1 | (78, 27) | 20 | 140 | 0 | ✓ | 0 | 0 |
| Titanic | 73250 | 845727 | 0x0200 | 2074 | 0x00000000 | 0 | (39, 97) | 92 | 208 | 0 | ✓ | **1** | 0 |

Aggregates:

- 4070 rooms total; **0 placeholder (`suite == -1`) rooms anywhere.**
- `flags` takes only 3 distinct values: `0x0` (14 houses), `0x2` (7),
  `0x6` (1 — `Art Museum`). Bit 0 (`wardBit`) is never set by any shipped house.
- `firstRoom` always in `[0, nRooms)`.
- `nRooms` ranges 2..531. `SpacePods` has the largest `firstRoom` (259).

## 12.6 Whole-corpus room and object field statistics

4070 rooms × 24 slots = 97680 object slots: **66240 empty, 31440 live.**

Room fields:

| Field | Observed domain | Notes |
|---|---|---|
| `name[0]` | 1..27 | never exceeds the `Str27` capacity |
| `bounds` | 0 (2401 rooms) and 31 odd values 1..63 | **every non-zero value is odd** ⇒ bit 0 marker confirmed; most common non-zero: 31 (542 rooms), 1 (167) |
| `leftStart`, `rightStart` | 0..255 | full unsigned byte range used |
| `unusedByte` | 0 in **all** 4070 rooms | force-zeroed by `GliderPRO/Sources/HouseLegal.c:868` |
| `visited` | 0 or 1 (1 in 436 rooms) | runtime residue |
| `background` | 2000..2017 and 3000..3353 (see 12.7) | all resolve to a real PICT |
| `tiles[i]` | **0..7 only**, over all 32560 slots | distribution 0:5114 1:7036 2:6395 3:3229 4:3925 5:2263 6:2161 7:2437 |
| `floor` | −7..39 | inside the legal −7..56 |
| `suite` | 0..127 | the **full** legal range is used |
| `openings` | 0 in **all** 4070 rooms | dead field |
| `numObjects` | 0..24 | matches live-slot count everywhere checked |

Object `what` codes: **all 117 defined codes occur at least once, and no
undefined code occurs at all.** Full counts are in the Part 5.2 tables. The five
most common are `kCustomPict` 4782, `kInvisBlower` 2336, `kCloud` 1860,
`kInvisLight` 1764, `kInvisBounce` 1670 — i.e. more than a third of all objects are
invisible mechanism objects or custom art.

### 12.6.1 Corpus-wide verification of every `KeepObjectLegal` parity rule

For each of the 117 object codes I counted the parity of the horizontal coordinate
(`topLeft.h` for the seven `Point`-based groups, `bounds.left` and `bounds.right`
for the two `Rect`-based groups) and of `length`, over all 31440 live objects. The
result matches the rules in 10.9 **exactly, with zero exceptions**.

| Rule (from 10.9) | Codes | Prediction | Observed |
|---|---|---|---|
| blower rule 3 | `kStubby` | `h` always **odd** | 0 even / **127 odd** ✓ |
| blower rule 4 | `kTaper`, `kCandle`, `kTiki`, `kBBQ` | `h` always **even** | 90/0, 180/0, 58/0, 43/0 ✓ |
| (no blower rule) | the other 11 blowers | `h` **mixed** | e.g. `kFloorVent` 758/700, `kInvisBlower` 1259/1077, `kLiftArea` 461/255 ✓ |
| furniture rule 2 | `kManhole` | `left` always **odd** (`64n + 3`), `right` always **even** | left 0 even / **40 odd**; right **40 even** / 0 odd ✓ |
| (no furniture rule) | the other 14 | `left`/`right` **mixed** | e.g. `kInvisBounce` 1078/592 and 1017/653 ✓ |
| prize rule 5 | **all 15** prizes | `h` always **even** | every code 100 % even (e.g. `kSparkle` 486/0, `kSlider` 354/0, `kStar` 69/0) ✓ |
| prize rule 6 | 14 prizes (not `kStar`) | `length` always **even** | every code 100 % even ✓ (`kStar`'s stored `length` is also even, being unused) |
| transport rules 3–6 | the 8 door/window codes | `h` ∈ {0, 368, 492, 496}, all **even** | 23/0, 11/0, 21/0, 11/0, 21/0, 32/0, 19/0, 29/0 ✓ |
| (no transport rule) | `kUpStairs`, `kDownStairs`, `kMailboxLf`, `kMailboxRt`, `kFloorTrans`, `kCeilingTrans`, `kInvisTrans`, `kDeluxeTrans` | `h` **mixed** | e.g. `kCeilingTrans` 234/231, `kDeluxeTrans` 30/31 ✓ |
| switch rule 2 | **all 9** switches incl. `kSoundTrigger` | `h` always **even** | 113/0, 79/0, 108/0, 78/0, 230/0, 635/0, 239/0, 81/0, **122/0** ✓ |
| (no light rule) | all 8 lights | `h` **mixed** | `kInvisLight` 803/961, `kCeilingLight` 85/75, `kFlourescent` 51/64 ✓ |
| appliance rule 3 | `kTV` | `h` always **odd** | **0 even / 80 odd** ✓ |
| appliance rule 4 | `kToaster`, `kMacPlus`, `kCoffee`, `kOutlet`, `kVCR`, `kStereo`, `kMicrowave` | `h` always **even** | 140/0, 74/0, 38/0, 100/0, 26/0, 36/0, 56/0 ✓ |
| (no appliance rule) | `kShredder`, `kGuitar`, `kCinderBlock`, `kFlowerBox`, `kCDs`, `kCustomPict` | `h` **mixed** | 30/20, 13/16, 11/32, 20/15, 32/31, **2574/2208** ✓ |
| enemy rule 4 | `kBalloon`, `kCopterLf`, `kCopterRt`, `kBall`, `kDrip`, `kFish` | `h` always **even** | 500/0, 259/0, 155/0, 212/0, 477/0, 120/0 ✓ |
| (no enemy rule) | `kDartLf`, `kDartRt`, `kCobweb` | `h` **unconstrained** | Darts happen to be all even (151/0, 117/0) by authoring habit; **`kCobweb` is 42 even / 44 odd**, proving the exemption is real ✓ |
| clutter rule 2 | `kMirror` | `left` **and** `right` always **even** | **667 / 0** and **667 / 0** ✓ |
| (no clutter rule) | the other 14 | `left`/`right` **mixed** | e.g. `kCloud` 953/907, `kFlower` 271/276 ✓ |

`kCobweb`'s mixed parity is the single most useful negative control in the set: it
proves the survey is measuring a real enforced invariant rather than an artefact of
the level editor's mouse grid.

The two "always even for a group with no rule" cases —
`kDartLf`/`kDartRt` and every light's `length` — are **not** invariants. A Go port
must not enforce them.

## 12.7 Resource fork inventory

Exactly **10 resource types** appear across all 22 houses:

| Type | Houses containing it | Observed IDs |
|---|---|---|
| `PICT` | 20 | 1015–1018, 1991–1993, 1999, 2014–2015, 3000–3053, 3100, 3102–3104, 3207–3236, 3300–3312, 3314–3327, 3352–3353, 3904, 3964, 3975–3976, 3981–3984, 3986, 8253, 10000–10094, 10100–10119, 11003–11019, 11021–11031, 12345, 12347–12351 |
| `bnds` | 8 | 3000–3010, 3300–3308 |
| `snd ` | 13 | 3000–3012, 3032, 3037, 3042–3046, 3058, 3061 |
| `vers` | 7 | 1, 2 |
| `ICN#` | 21 | −16455, 128 |
| `icl4` | 21 | −16455, 128 |
| `icl8` | 21 | −16455, 128 |
| `ics#` | 21 | −16455, 128 |
| `ics4` | 21 | −16455, 128 |
| `ics8` | 21 | −16455, 128 |

Per-house detail (counts in parentheses):

```
Art Museum             ICN#(1) PICT(76)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1) snd (10)
CD Demo House          ICN#(1) PICT(116) icl4(1) icl8(1) ics#(1) ics4(1) ics8(1) snd (10)
California or Bust!    ICN#(1) PICT(30)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1) snd (3)
Castle o' the Air      ICN#(1) PICT(12)  bnds(9)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1)
Davis Station          ICN#(1) PICT(54)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1) snd (5)
Demo House             ICN#(1) PICT(19)  bnds(10) icl4(1) icl8(1) ics#(1) ics4(1) ics8(1) snd (1)
Empty House            ICN#(1)           icl4(1) icl8(1) ics#(1) ics4(1) ics8(1)
Fun House              ICN#(1) PICT(26)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1)
Grand Prix             ICN#(1) PICT(70)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1) snd (3) vers(2)
ImagineHouse PRO II    ICN#(1) PICT(45)  bnds(14) icl4(1) icl8(2) ics#(1) ics4(1) ics8(1) snd (2) vers(2)
In The Mirror          ICN#(2) PICT(15)  icl4(2) icl8(2) ics#(1) ics4(1) ics8(1) snd (2) vers(2)
Land of Illusion       ICN#(1) PICT(20)  bnds(5)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1)
Leviathan              ICN#(2) PICT(53)  bnds(4)  icl4(1) icl8(2) ics#(1) ics4(1) ics8(1) snd (12) vers(2)
Metropolis             ICN#(1) PICT(39)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1) vers(1)
Nemo's Market          ICN#(1) PICT(73)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1) snd (5)
Rainbow's End          ICN#(1) PICT(17)  bnds(6)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1) snd (1)
Sampler                (none)
Slumberland            ICN#(1) PICT(20)  bnds(20) icl4(1) icl8(1) ics#(1) ics4(1) ics8(1)
SpacePods              ICN#(1) PICT(49)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1) snd (3)
Teddy World            ICN#(2) PICT(120) icl4(2) icl8(2) ics#(2) ics4(2) ics8(2) vers(2)
The Asylum Pro         ICN#(1) PICT(17)  bnds(2)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1)
Titanic                ICN#(2) PICT(48)  icl4(1) icl8(1) ics#(1) ics4(1) ics8(1) snd (6) vers(1)
```

Observations:

- **`Sampler`'s resource fork is 286 bytes and contains zero resources.** Its
  header is `00 00 01 00 | 00 00 01 00 | 00 00 00 00 | 00 00 00 1E`
  (dataOffset 256, mapOffset 256, dataLen 0, mapLen 30), and the map at 256 ends
  `... 00 1C 00 1E FF FF` (typeListOffset 28, nameListOffset 30,
  numTypes − 1 = −1 ⇒ **zero types**). So a valid house may have a completely empty
  resource fork, and every background reference must then resolve against the
  application's own resources. `Sampler` uses only built-in backgrounds
  (2003 and 2012), consistent with this.
- **`Empty House` has only the Finder icon family.** No PICTs at all.
- `bnds` IDs only ever appear in the 3000/3300 user-art ranges, matching
  `GliderPRO/Sources/RoomInfo.c:762`'s `[3000, 3800)` gate.
- `snd ` IDs are all ≥ 3000, matching the `kSoundTrigger` editor range
  `[3000, 32767]` (`GliderPRO/Sources/ObjectInfo.c:1237`).
- Custom object PICT IDs are all ≥ 10000, matching the `kCustomPict` editor range
  `[10000, 32767]` (`GliderPRO/Sources/ObjectInfo.c:1213`). Some houses use a
  second block at 11000+ and a stray 12345/12347–12351.
- **Custom object PICTs carry human-readable resource names; backgrounds do not.**
  `Demo House` `PICT` names: 10008 `'Milk'`, 10011 `'3-Legged Table'`, 10012
  `'Home Needlepoint'`, 10014 `'Quilt on Wall'`, 10017 `'Frying Pan'`, 10018
  `'Soup Can'`, 10036 `'Bagel Bag'`, 10049 `'Salt/Pepper/Napkins'`, 10065
  `'Two Bananas'`. `Demo House` background PICT sizes: 3000 = 26392 B,
  3001 = 50132, 3002 = 14718, 3003 = 100432, 3004 = 20872, 3005 = 74032,
  3300 = 15884, 3301 = 40676, 3302 = 48342, 3303 = 46230.
- **Houses override application PICT IDs.** Every observed ID in the 1015–2015
  range collides with an application-owned constant:

  | ID | Application constant | Defined at |
  |---|---|---|
  | 1015 | `kEscPausePictID` (also `kInvisBonusInfoDialogID`) | `GliderPRO/Sources/Input.c:17`, `GliderPRO/Sources/ObjectInfo.c:23` |
  | 1016 | `kTabPausePictID` (also `kOriginalArtDialogID`) | `GliderPRO/Sources/Input.c:18`, `GliderPRO/Sources/RoomInfo.c:19` |
  | 1017 | `kStarsRemainingPICT` (also `kDisplayPrefsDialID`) | `GliderPRO/Sources/Banner.c:20`, `GliderPRO/Sources/Settings.c:18` |
  | 1018 | `kStarRemainingPICT` (also `kSoundPrefsDialID`) | `GliderPRO/Sources/Banner.c:21`, `GliderPRO/Sources/Settings.c:19` |
  | 1991 | `kBannerPageBottomMask` | `GliderPRO/Sources/Banner.c:19` |
  | 1992 | `kBannerPageBottomPICT` | `GliderPRO/Sources/Banner.c:18` |
  | 1993 | `kBannerPageTopPICT` | `GliderPRO/Sources/Banner.c:17` |
  | 1999 | `kSupportPictID` | `GliderPRO/Sources/StructuresInit2.c:21` |
  | 2014 | `kRoof` | `GliderPRO/Headers/GliderDefines.h:241` |
  | 2015 | `kSky` | `GliderPRO/Headers/GliderDefines.h:242` |

  Plus the un-attributed IDs 3904, 3964, 3975–3976, 3981–3984, 3986, 8253, which
  fall outside every documented range and are almost certainly leftovers from the
  authors' own art pipelines (3904/3964/3975+ are inside the `[3000, 3800)` user-art
  window only for 3300–3799; 3904+ and 8253 are outside it entirely and can never be
  referenced by a `background` field). This is legal
  and intentional: every lookup uses chain-wide `GetPicture`/`GetResource`, and
  `UseResFile(houseResFork)` (`GliderPRO/Sources/HouseIO.c:575`) puts the house on
  top of the chain, so a house can re-skin the banner, the pause screen, and the
  floor supports.
- The **one** exception is `HouseHasOriginalPicts`, which uses the current-file-only
  form: `Count1Resources('PICT') > 0`
  (`GliderPRO/Sources/House.c:250-251`) — the only `Count1Resources`/`Get1Resource`
  style call touching house art in the program. (`SelectHouse.c` also uses
  `Get1Resource('icl8', -16455)` at `GliderPRO/Sources/SelectHouse.c:102-106`, but
  that is the Finder icon, not level art.)
- `vers` resources exist in 7 houses but are **never read**: `About.c` switches to
  `thisMac.thisResFile` before `GetResource('vers', 1)` and switches back
  (`GliderPRO/Sources/About.c:48-49`, `55`, `88`).
- The Finder icon family lives at ID **−16455** (the standard custom-file-icon ID);
  the ID-128 family is the *document* icon that the house's own bundle would
  reference. `SelectHouse` reads `icl8` −16455 to draw the house-picker list.

## 12.8 Integrity findings in shipped content

These are real defects in the shipped houses that a port must survive.

| Finding | Detail |
|---|---|
| Dangling links | 6 of 22 houses contain `where` values whose (`floor`, `suite`) cell has no room. Worst: `Slumberland` with **69**. `GetRoomNumber` returns −1 (`GliderPRO/Sources/Room.c:743`, `758`) so the link silently becomes "unlinked". |
| Out-of-range `who` | Exactly one: `CD Demo House` room[72] (floor 1, suite 21) slot 22, a `kMailboxRt` with `where` = 2109 and **`who` = 35** ≥ `kMaxRoomObs` (24). Indexing `retroLinkList[35]` (`GliderPRO/Sources/House.c:600`) reads out of bounds on the original; a Go port must guard. |
| `kSoundTrigger` with a bad `snd ` ID | 2 objects have `where` = 10000 (a `kCustomPict`-style ID). 13 more reference `snd ` IDs absent from their house: `In The Mirror` 2, `Teddy World` 11. All are permanently inert because `LoadTriggerSound` fails (`GliderPRO/Sources/ObjectRects.c:926`). |
| Stale `bounds` on built-in backgrounds | 27 rooms: `CD Demo House` 23, `Grand Prix` 2, `Land of Illusion` 1, `Teddy World` 1. Ignored because the code range-checks the background first. |
| Stale `savedGame` with `hasGame` = 0 | 20 of 22 houses. `Sampler`'s block is obvious heap garbage (`version` 1, `bands` −1, `facing` 204). |
| Uninitialized `unusedShort` / `unusedBoolean` | 9 houses have non-zero `unusedShort` (−30082 … 26228) and 6 have non-zero `unusedBoolean` (up to 255). |
| 2-byte tail | `Sampler` only (Part 12.4). |
| Negative `kDeluxeTrans` `tall` | Observed values include −0x7FD2 and −0x7FB0, i.e. the packed width byte has its high bit set. See 5.6.1. |

## 12.9 The probe tool

`tools/probe_house.py` is the verified decoder used
throughout. Capabilities:

- BinHex 4.0 decode with the exact 64-character alphabet, `0x90` RLE, and
  **CRC verification** of both forks (`binhex_decode(path, check_crc=True)`).
  All 22 shipped houses verify.
- `parse_resource_fork(res)` → `{typeString: [(id, name, absDataOffset, length), …]}`.
- `parse_house(data)`, `parse_room(...)`, `parse_object(...)` implementing the
  offset tables in Parts 3–5, with embedded constants
  `SIZEOF_HOUSE_HEADER = 866`, `SIZEOF_ROOM = 348`, `SIZEOF_OBJECT = 12`,
  `kMaxRoomObs = 24`, `kNumTiles = 8`, `kMaxScores = 10`,
  `kNumUndergroundFloors = 8`, `kRoomIsEmpty = -1`, `kObjectIsEmpty = -1`.
- `extract_floor_suite(combo, version)` per Part 7.2.
- CLI flags `--forks --resmap --rooms N --room N --allrooms --raw --dump-data
  --dump-rsrc --hexhdr N`. Caveat: `--forks` returns before `--hexhdr` is honoured.

Printing conventions to keep in mind when reading its output: `topLeft` is printed
as `(h, v)` (h first, opposite to the on-disk order), and furniture/clutter
`bounds` is printed as `(left, top, right, bottom)`.

The verified BinHex CRC-16, spelled out bitwise. It is standard CRC-16/XMODEM: this
formulation applied to `fork + b"\0\0"` and the common table-driven "XOR the byte
into the high half" XMODEM loop applied to `fork` are the same function, and both
match all 66 stored CRCs (header + data + rsrc across the 22 files):

```python
def binhex_crc(data, crc=0):
    for b in data:
        for i in range(8):
            hibit = crc & 0x8000
            crc = ((crc << 1) | ((b >> (7 - i)) & 1)) & 0xFFFF
            if hibit:
                crc ^= 0x1021
    return crc
# usage: binhex_crc(fork + b"\0\0") == storedCRC
```

---

# Part 13 — Mac Toolbox dependencies and their Go replacements

## 13.1 Dependency table

| Toolbox facility | Used for | Citations | Go replacement |
|---|---|---|---|
| **File Manager**: `FSpOpenDF`, `FSClose`, `GetEOF`, `SetEOF`, `SetFPos`, `FSRead`, `FSWrite` | reading and writing the data fork | `GliderPRO/Sources/HouseIO.c:189`, `341`, `364`, `372`, `491`, `499` | `os.ReadFile` / `os.WriteFile`, or `*os.File` with `Seek`/`Truncate` |
| **`FSSpec`** (`{vRefNum, parID, name}`) | identifying houses; embedded in `game2Type` but never on disk for houses | `GliderPRO/Headers/GliderStructs.h:146` | a plain path `string` |
| **`FSpCreate` / `HCreateResFile`** with type/creator | creating `'gliH'`/`'ozm5'` files with both forks | `GliderPRO/Sources/House.c:85`, `88` | one file for the data fork plus a decision about where resources live (see 13.3) |
| **`ResolveAliasFile`** | following Finder aliases | `GliderPRO/Sources/HouseIO.c:177-178` | `filepath.EvalSymlinks` |
| **`PBGetCatInfo`** directory walk with `ioFlAttrib & 0x10` | finding houses by type/creator | `GliderPRO/Sources/SelectHouse.c:592-609` | `filepath.WalkDir` + an extension/sniff convention |
| **Resource Manager**: `FSpOpenResFile`, `UseResFile`, `CurResFile`, `CloseResFile`, `HOpenResFile` | opening the house's resource fork and pushing it on the search chain | `GliderPRO/Sources/HouseIO.c:571`, `575`, `586`; `GliderPRO/Sources/SelectHouse.c:87`, `131` | an explicit two-level asset lookup: house map, then app map |
| **`GetPicture(id)` / `GetResource('Date', id)`** | background and custom-object art, chain-wide | `GliderPRO/Sources/Room.c:269`, `272`; `GliderPRO/Sources/RoomGraphics.c:139-145`; `GliderPRO/Sources/Map.c:169-172`; `GliderPRO/Sources/ObjectRects.c:222` | a PICT decoder (see 13.4) fronted by `func (h *House) Picture(id int16) *image.Paletted` |
| **`Count1Resources('PICT')`** | "does this house ship its own art?" | `GliderPRO/Sources/House.c:250-251` | `len(house.pictIDs) > 0` |
| **`GetResource('bnds', id)`** | 4-byte room-opening records | `GliderPRO/Sources/Room.c:942` | a `map[int16][4]bool` |
| **`GetResource('snd ', id)`** + skip 20 bytes | `kSoundTrigger` custom sound | `GliderPRO/Sources/Sound.c:279`, `286`, `296` | parse the `'snd '` format-1 header properly, or replicate the +20 skip for bit-exactness |
| **Memory Manager**: `NewHandle`, `NewPtrClear`, `DisposeHandle`, `SetHandleSize`, `GetHandleSize`, `PtrAndHand`, `HLock`/`HUnlock`/`HGetState`/`HSetState`, `MoveHHi`, `BlockMove` | the whole house lives in one relocatable handle whose size **is** the room count | `GliderPRO/Sources/HouseIO.c:356`, `473`; `GliderPRO/Sources/Room.c:200`; `GliderPRO/Sources/HouseLegal.c:630`, `767` | `[]byte` plus `[]Room`; note `GetHandleSize` semantics must be emulated when computing the written length (Part 9.4) |
| **QuickDraw**: `CopyBits`, `DrawPicture`, `PaintRect`, `QSetRect`, `QOffsetRect`, `MoveTo`, `TextFont`, `ForeColor` | tile blitting, banner text, map thumbnails | `GliderPRO/Sources/RoomGraphics.c:246-251`; `GliderPRO/Sources/Banner.c:136` | any 2-D blitter; `image/draw` for a reference implementation |
| **GWorlds** (offscreen `GWorldPtr`) | the room compositing buffers | throughout `RoomGraphics.c` | `*image.Paletted` / `*image.RGBA` back buffers |
| **8-bit indexed colour** | all art is 8-bit with the game's `clut`; `kSplash8BitPICT` = 1000 (`GliderPRO/Headers/GliderDefines.h:524`) | `GliderPRO/Sources/Play.c:271` | keep a palette; `image.Paletted` |
| **Sound Manager**: `PlayPrioritySound`, the `theSoundData[64]` slot array | `kSoundTrigger` and every other effect | `GliderPRO/Sources/Sound.c:14-15`, `265-303`; `GliderPRO/Sources/Interactions.c:1618` | any mixer with 64 slots and the reserved slot 63 |
| **QuickTime**: `OpenMovieFile`, `NewMovieFromFile`, `LoadMovieIntoRam`, `PrerollMovie`, `SetTimeBaseFlags(loopTimeBase)` | the `<house>.mov` sidecar shown in a `kTV` | `GliderPRO/Sources/HouseIO.c:87`, `94`, `113`, `124`, `133` | optional; a modern video decoder or a still frame |
| **`GetDateTime`** (Mac epoch 1904-01-01, local time) | `timeStamp`, `highScores.timeStamps[]`, `savedGame.timeStamp` | `GliderPRO/Sources/HouseIO.c:477`; `GliderPRO/Sources/SavedGames.c:320` | `time.Now()` converted: `macSeconds = unixSeconds + 2082844800` |
| **`NumToString` / `PasStringCopy` / `PasStringConcat` / `PasStringCopyNum` / `EqualString`** | Pascal-string manipulation, case-insensitive compare | `GliderPRO/Sources/HouseInfo.c:232-233`, `264-265`; `GliderPRO/Sources/HouseLegal.c:846`; `GliderPRO/Sources/RoomInfo.c:472` | a `PStr` type with explicit length + MacRoman codec |
| **MacRoman text encoding** | every string field | — | `golang.org/x/text/encoding/charmap.Macintosh`, or a hand-rolled 128-entry table (airgapped-friendly) |
| **script codes** | `FSpCreate` takes a script code; the source passes `theReply.keyScript` from the Navigation/StandardFile reply, not `smSystemScript` | `GliderPRO/Sources/House.c:85` | not applicable |
| **`DebugStr`** | the out-of-range pigeonhole report | `GliderPRO/Sources/HouseLegal.c:669` | `log` / a real bounds check |
| **68k/PPC struct alignment** | `sizeof(houseType)` = 866 vs 868 | `GliderPRO/Sources/HouseLegal.c:765` | must be modelled explicitly, not inferred from `unsafe.Sizeof` |

## 13.2 Resource-chain semantics a port must emulate

Lookup order for any `PICT`, `bnds`, or `snd ` ID:

1. the currently open **house** resource fork (pushed by `UseResFile`,
   `GliderPRO/Sources/HouseIO.c:575`),
2. then the **application**'s own resource fork,
3. then the System file.

So a Go port needs a small chain abstraction, e.g.

```go
type ResChain []ResMap                    // index 0 = topmost (house)
func (c ResChain) Get(typ ResType, id int16) ([]byte, bool)
```

with the single exception of `Count1Resources`/`Get1Resource`, which must consult
**only** the house map (`GliderPRO/Sources/House.c:250-251`,
`GliderPRO/Sources/SelectHouse.c:102-106`).

## 13.3 Where the resource fork goes on a non-Mac filesystem

Classic resource forks cannot be represented as a byte range of the data file. A
Go port has to pick one of:

- **AppleDouble** (`._Name`) sidecars — what macOS itself writes on foreign
  volumes; already the de-facto layout for extracted Glider content.
- **MacBinary / BinHex** single-file containers — the format the shipped houses
  actually arrive in. `tools/probe_house.py` already
  reads BinHex 4.0 and verifies its CRCs, so this is the lowest-risk path for
  reading the original corpus.
- A **converted asset bundle** (e.g. a directory or zip of PNGs + a JSON index)
  produced by an import step. Best for the shipping game, but then the port can no
  longer round-trip original files byte-for-byte.

The recommendation: read BinHex/AppleDouble for import, keep the resource map in
memory as `map[ResType]map[int16]Resource`, and write the data fork alone when
saving (matching `WriteHouse`, which never touches the resource fork).

## 13.4 `PICT` is not optional

Backgrounds (2000–2017, 3000–3799), custom object art (≥ 10000), and the map
thumbnails are all QuickDraw `PICT` version 1/2 opcode streams. There is no
alternative representation in the file. A port must either implement a `PICT`
decoder covering the opcodes the shipped art uses (`PackBitsRect`,
`DirectBitsRect`, `Clip`, `LongComment`, …) or run a one-time conversion pass. The
`'Date'` fallback type (`GliderPRO/Sources/Room.c:272`) holds the same payload
under a different type code; no shipped house uses it.

Also note the background PICT geometry contract: the picture must be **at least
`8 * 64 = 512` px wide and `322` px tall**, laid out as N side-by-side 64-px
columns, because `tiles[i]` indexes columns (Part 4.6). Every shipped background
is exactly 8 columns.

## 13.5 Endianness and integer width

Every multi-byte scalar in the file is **big-endian**. Every `short` is signed
16-bit; every `long` is signed 32-bit except `scoresType.timeStamps[]`, which is
`unsigned long`. `Byte` is unsigned 8-bit; `Boolean` is a single byte where the
only values written are 0 and 1 (but arbitrary residue can be read — always test
`!= 0`, never `== 1`).

Recommended Go decoding primitives:

```go
be16 := func(b []byte) int16  { return int16(binary.BigEndian.Uint16(b)) }
be32 := func(b []byte) int32  { return int32(binary.BigEndian.Uint32(b)) }
beU32:= func(b []byte) uint32 { return binary.BigEndian.Uint32(b) }
```

and for the `kDeluxeTrans` packing specifically, `uint16` first (Part 5.6.1).

---

## Open questions

1. **No version-1 house exists in the corpus**, so the legacy
   `ExtractFloorSuite` branch (`GliderPRO/Sources/Link.c:43-47`) and
   `ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746`) are verified by code
   reading and by the digit-swap algebra in Part 11, but not by observation. The
   version-1 `MergeFloorSuite` is *inferred* to be
   `(floor + 8) * 100 + suite` from its inverse; Glider PRO 1.x's own merge
   function is not in this source drop.
2. **The `srcRects[]` table itself is not in the assigned file set.** Its 0x90
   entries (`kNumSrcRects`, `GliderPRO/Headers/GliderDefines.h:437`) are the
   canonical sprite sizes referenced by `GetObjectRect`
   (`GliderPRO/Sources/ObjectRects.c:39-270`), by the door/window snapping
   (`GliderPRO/Sources/HouseLegal.c:314`), and by the `kManhole` grid snap
   (`GliderPRO/Sources/HouseLegal.c:224`). The **file format** does not depend on
   them — every stored rect and coordinate is documented above without them — but a
   port cannot reproduce `KeepObjectLegal`'s output byte-for-byte until it has the
   table. It is initialised in the sprite-setup code
   (`GliderPRO/Sources/StructuresInit.c` / `StructuresInit2.c`).
3. **What exactly does `data.b.pict` mean for furniture?** It is structurally
   present, observed 0 everywhere sampled, and no reader was found. It may be
   vestigial from a shared editor path with `clutterType`.
4. **`data.f.byte0`, `data.f.byte1`, `data.h.byte0`** have no reader in 1.0.4 — only
   copy/paste writers (`GliderPRO/Sources/ObjectEdit.c:1298-1300`, `1333`). Were
   they live in an earlier version? Their observed values were not surveyed field
   by field.
5. **The `'Date'` resource type** (`GliderPRO/Sources/Room.c:272`) is read as a
   `PicHandle`. No shipped house uses it. Was it an art-protection or
   third-party-tool convention?
6. **The `snd ` +20-byte skip** (`GliderPRO/Sources/Sound.c:286`, `296`) assumes a
   specific `'snd '` format-1 header shape (one synth, one command, one
   `soundHeader` prefix). Whether all 13 sound-bearing houses conform was not
   verified; a malformed one would produce noise rather than an error.
7. **`houseIsReadOnly` is always false** because `IsFileReadOnly` is a stub
   (`GliderPRO/Sources/HouseIO.c:663`). The `'gliS'` high-score sidecar therefore
   appears to be dead code in 1.0.4. Was it live in 1.0.0–1.0.3, and do sidecar
   files exist in the wild?
8. **`savedGame.roomNumber` is a room index that `CompressHouse` does not patch**
   (`GliderPRO/Sources/HouseLegal.c:688-734`). Is there a code path that
   invalidates `hasGame` on compression, or is a saved game genuinely corrupted by
   any edit-mode save? Only 2 shipped houses have `hasGame` = 1, so this was not
   observable.
9. **`gliderState`, `foil`, and `wasStarsLeft` semantics** in `gameType` were not
   traced to their consumers; the saved-game reader is commented out
   (`GliderPRO/Sources/SavedGames.c:169`), so the only writer is
   `GliderPRO/Sources/SavedGames.c:303-351`.
10. **Are `bounds` bits 6–15 ever set?** Observed maximum is 63, so no — but the
    writer only ever produces values ≤ 63 and there is no reader-side mask, so a
    third-party editor could set them.

## Porting notes

Ordered roughly by risk.

1. **The level data is the DATA fork, not a resource.** There is no resource type
   or ID holding the house. `FSpOpenDF` + one `FSRead` of the whole fork
   (`GliderPRO/Sources/HouseIO.c:189`, `372`). The resource fork holds only art,
   `bnds`, sounds, `vers`, and the Finder icon.
2. **Parse rooms at `866 + 348*r`, not at `sizeof(struct)*r`.** Derive the room
   count as `min(nRooms, (len(data) - 866) / 348)` and tolerate up to 2 trailing
   bytes. 21 of 22 shipped houses are exactly `866 + 348n`; `Sampler` is
   `868 + 348n` (Part 12.4).
3. **`Point` is `{v, h}` — vertical FIRST.** Three-quarters of the object variants
   start with a `Point`, so getting this backwards mirrors and transposes every
   object in the game. Verified empirically three ways (Part 12.3).
4. **`Rect` is `{top, left, bottom, right}`** — also verified (Part 12.3).
5. **`bonusType` alone puts `state` before `initial`.** Every other variant is
   `initial` then `state`. Do not share one decoder across variants without
   respecting this.
6. **`enemyType` is `delay` then `byte0`; `applianceType` is `byte0` then
   `delay`.** Opposite order at the same offsets.
7. **`flags` bit 2 is inverted**: `bannerStarCountOn = ((flags & 4) == 0)`
   (`GliderPRO/Sources/HouseIO.c:418`).
8. **`timeStamp` bit 0 is the house-lock flag**, not part of the time
   (`GliderPRO/Sources/HouseIO.c:402`). Mask it out before converting to a
   `time.Time`; Mac epoch offset is 2082844800 s.
9. **`kSoundTrigger` (`0x49`) overloads `data.e.where` as a `'snd '` resource ID,
   not a room link** (`GliderPRO/Sources/ObjectRects.c:926`). Exclude it from every
   link walk exactly as the original does
   (`GliderPRO/Sources/House.c:279-296`, `339-366`, `561-591`;
   `GliderPRO/Sources/Objects.c:149-156`), or link remapping will destroy sound
   IDs.
10. **`kCustomPict` (`0x6E`) overloads `data.g.height` as a `PICT` ID**
    (`GliderPRO/Sources/ObjectRects.c:222`) and is the single most common object
    in the corpus. On a missing PICT the original *rewrites the field to 10000*;
    decide deliberately whether to reproduce that mutation.
11. **`kDeluxeTrans` (`0x40`) packs its size into `data.d.tall` as
    `(w/4)<<8 | (h/4)`.** Decode via `uint16` (`GliderPRO/Sources/ObjectRects.c:153-155`);
    Go's arithmetic `>>` on a negative `int16` gives a negative width for the
    observed −0x7FD2 / −0x7FB0 values.
12. **`where` = −1 but `who` = 255** are the two unlinked sentinels — a `short`
    sentinel and a `Byte` sentinel (`GliderPRO/Sources/Link.c:333-334`).
13. **`suite == -1` marks a deleted room; `floor == -1` is a legal floor.** Never
    test `floor` for emptiness (`GliderPRO/Sources/Room.c:462-463`).
14. **Rooms connect by grid adjacency, not by stored pointers.** `floor + 1` is
    NORTH (`GliderPRO/Sources/Room.c:562-635`). `suite` is the horizontal axis.
    Links store the *grid position* (`suite*100 + floor + 8`), which is why
    `CompressHouse` can renumber rooms freely.
15. **Clamp or reject `who >= 24`.** One shipped house has `who` = 35
    (`CD Demo House` room[72] slot 22), which the original happily reads out of
    bounds (`GliderPRO/Sources/House.c:600`).
16. **Dangling links must degrade to "unlinked", not error.** `GetRoomNumber`
    returns −1 for an empty cell (`GliderPRO/Sources/Room.c:743`, `758`); 6 of 22 shipped
    houses depend on this.
17. **`tiles[i]` is a column index, not a pixel offset**:
    `src.left = tiles[i] * 64` (`GliderPRO/Sources/Room.c:291`). Backgrounds are
    8 columns × 64 px × 322 px.
18. **`bounds` bit 0 is a marker, not data.** `boundCode = bounds >> 1`; if
    `bounds == 0`, fall back to the 4-byte `'bnds'` resource
    (`GliderPRO/Sources/Room.c:942`). And **check the background range before
    consulting `bounds`** — 27 shipped rooms have stale non-zero `bounds` on
    built-in backgrounds.
19. **`state` bytes, `visited`, and the whole `savedGame` block when
    `hasGame == 0` are stale residue.** `SetObjectsToDefaults`
    (`GliderPRO/Sources/Play.c:603-708`) overwrites every `state` from `initial`
    and clears every `visited` at game start. Do not treat them as authoritative,
    but do preserve them for byte-exact round-tripping.
20. **`unusedShort`, `unusedBoolean`, and the 10 data bytes of an empty object slot
    are uninitialized heap garbage** (`GliderPRO/Sources/House.c:116` uses
    `NewHandle`, not `NewHandleClear`; `GliderPRO/Sources/Room.c:181-182` writes
    only `what`). Carry them through verbatim if you want bit-identical saves.
21. **`openings` is dead** (written 0 at `GliderPRO/Sources/Room.c:179`, never
    read) and **`unusedByte` is force-zeroed** for every room, empty or not
    (`GliderPRO/Sources/HouseLegal.c:868`).
22. **Reproduce the `topLeft.h` parity forcing** if you re-save. The exact rules
    (all verified against 31440 real objects in 12.6.1):
    - forced **even**: `kTaper`, `kCandle`, `kTiki`, `kBBQ`; **all 15 prizes**;
      **all 9 switches including `kSoundTrigger`**; the 8 door/window codes (via
      their fixed lefts 0/368/492/496); `kToaster`, `kMacPlus`, `kCoffee`,
      `kOutlet`, `kVCR`, `kStereo`, `kMicrowave`; `kBalloon`, `kCopterLf`,
      `kCopterRt`, `kBall`, `kDrip`, `kFish`.
    - forced **odd**: `kStubby`, `kTV`, and `kManhole`'s `bounds.left`
      (which is `64n + 3`).
    - `kMirror` forces `bounds.left` **and** `bounds.right` even.
    - `length` forced even for 14 of the 15 prizes (`kStar` exempt).
    - everything else is unconstrained — in particular `kCustomPict`, `kCobweb`,
      all 8 lights, and 11 of the 16 blowers.
    These are 68k word-alignment artefacts with no gameplay effect, but they are
    the difference between a byte-identical save and a diff on thousands of objects.
23. **`WrapText` is destructive**: banner at 40 chars, trailer at 64, overwriting
    spaces with `\r` (`GliderPRO/Sources/HouseLegal.c:611-612`,
    `GliderPRO/Sources/StringUtils.c:216-248`). It runs unconditionally on every
    edit-mode save.
24. **`CompressHouse` and `LopOffExtraRooms` run unconditionally**
    (`GliderPRO/Sources/HouseLegal.c:1103`, `1105`) — everything else in
    `CheckHouseForProblems` is gated by the `isHouseChecks` preference. If you
    reimplement the save path, keep that split or you will change which files get
    repaired.
25. **`ValidateNumberOfRooms` resolves a `nRooms` / length disagreement in favour
    of the length** (`GliderPRO/Sources/HouseLegal.c:630-634`). Adopt that rule at
    load time; the original's own load path trusts `nRooms` blindly and can read
    out of bounds.
26. **`ConvertHouseVer1To2` is exactly a decimal digit-pair swap and is NOT
    idempotent.** Set `version = 0x0200` (`GliderPRO/Sources/House.c:814`) before
    any possibility of a second pass, and note that during the pass
    `ExtractFloorSuite` still sees the old version
    (`GliderPRO/Sources/Link.c:43`).
27. **Room-name and banner text is MacRoman**, and `\r` (0x0D) — not `\n` — is the
    in-string line break. Pascal strings are fixed-size with an unsigned length
    byte and uncleared residue past the length: `Str255` 256 B, `Str31` 32 B,
    `Str27` 28 B, `Str15` 16 B.
28. **A house may have an entirely empty resource fork** (`Sampler`, 286 bytes,
    zero resource types) or only a Finder icon (`Empty House`). Do not require
    house-local art.
29. **Houses legitimately override application PICT IDs** (1015–1019, 1991–1993,
    1999, 2014–2015, and more) because every lookup is chain-wide and
    `UseResFile(houseResFork)` puts the house on top
    (`GliderPRO/Sources/HouseIO.c:575`). The sole current-file-only lookups are
    `Count1Resources('PICT')` (`GliderPRO/Sources/House.c:250-251`) and
    `Get1Resource('icl8', -16455)` (`GliderPRO/Sources/SelectHouse.c:102-106`).
30. **Only one `kSoundTrigger` sound can be live at a time** — slot 63 of 64
    (`kTriggerSound` = `kMaxSounds - 1`,
    `GliderPRO/Headers/GliderDefines.h:118`, `GliderPRO/Sources/Sound.c:271-272`).
    The editor refuses to add a second one per room:
    `#define kMaxSoundTriggers 1` (`GliderPRO/Sources/ObjectAdd.c:17`) enforced by
    `if ((what == kSoundTrigger) && (HowManySoundObjects() >= kMaxSoundTriggers))`
    (`GliderPRO/Sources/ObjectAdd.c:484`). The **file format does not enforce it**,
    though — a hand-built house could put 2 or more in one room. Verified: the corpus
    holds 122 `kSoundTrigger` objects spread over 122 distinct rooms, i.e. **exactly
    one per room, never two** — the editor's guard has held everywhere.
31. **There is no magic number, no checksum, and no length field in the house
    file.** Identification relies on the Mac type code `'gliH'`
    (`GliderPRO/Sources/SelectHouse.c:592-593`). A Go port on a plain filesystem
    needs its own sniff heuristic; the strongest available signals are
    `version <= 0x02FF`, `1 <= nRooms`, `len == 866 + 348*nRooms` (± 2), and
    `0 <= firstRoom < nRooms`.
32. **Write path detail:** `WriteHouse` writes `GetHandleSize` bytes then
    `SetEOF` to the same value (`GliderPRO/Sources/HouseIO.c:491`, `499`), and
    updates `timeStamp`/`version` **only when `fileDirty`**
    (`GliderPRO/Sources/HouseIO.c:475`). `SaveHouseAs` does not exist in 1.0.4
    (`GliderPRO/Sources/HouseIO.c:307`).
