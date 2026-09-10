# Glider PRO 1.0.4 — Normative Format Decisions

## Scope

Six binary-format and build-identity questions in this analysis set were left
open in three to five documents each, with subtly different — sometimes
contradictory — advice in each. This document closes all six. It is the single
normative source for them. Where a sibling document's open-question list still
carries the old wording, **this document supersedes it**; the mapping is in
[§9 Which open question each decision closes](#9-which-open-question-each-decision-closes).

The six:

| # | Question | Previously open in |
|---|---|---|
| [D1](#d1--sizeofgame2type-is-110-not-114) | Is `sizeof(game2Type)` 110 or 114? | architecture.md OQ14, progression.md OQ1, scoring.md OQ7, constants.md OQ8, house-format.md §12.4 |
| [D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant) | Which struct-alignment model did the shipping build use? | *(prerequisite to D1 and D3; asserted without proof in structs.md, house-format.md, progression.md)* |
| [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length) | What are `Sampler.binhex`'s 2 trailing bytes, and must a loader accept / reject / preserve them? | house-format.md §12.4, object-taxonomy.md OQ1, interactions.md OQ5, editor.md Q4, constants.md OQ9, scoring.md OQ24 |
| [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) | Does byte-exact round-tripping require preserving `houseType.unusedShort`? | original-houses.md OQ1, progression.md OQ4, scoring.md OQ4, constants.md OQ5 |
| [D5](#d5--the-bound-on-housetypenrooms) | What bounds `houseType.nRooms`? | constants.md OQ14, house-format.md, editor.md |
| [D6](#d6--the-retail-build-configuration-and-the-version-identity) | Which build shipped as retail (`BUILD_ARCADE_VERSION` 0 or 1), and is the product 1.0.4 or 1.1.2? | architecture.md OQ1, progression.md OQ2, constants.md OQ12, resource-fork.md OQ6, scoring.md OQ12, original-houses.md OQ8 |
| [D7](#d7--demo-playback-termination) | What terminates demo playback, given nothing bounds-checks `demoData[demoIndex]`? | architecture.md OQ12, input.md OQ6, input.md OQ1 |

Each decision below is laid out identically:

* **The question as previously posed** — quoting the sibling docs so a reader
  arriving from one of them recognises the item.
* **Evidence** — compiled `sizeof`/`offsetof` output, or bytes parsed out of the
  shipped files with `python3`, or source citations. No inference is presented as
  observation.
* **Decision** — normative, imperative, addressed to the Go port.
* **Consequence of deciding otherwise** — the specific observable failure.
* **Go** — the implementation shape that satisfies the decision.

## How to use this document

If you are implementing a loader, read [§0.2](#02-the-two-alignment-models-compiled),
[D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant),
[D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length),
[D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) and
[D5](#d5--the-bound-on-housetypenrooms) — that is the whole house-file contract.
[D1](#d1--sizeofgame2type-is-110-not-114) matters only if you intend to read a
`'gliG'` side-car saved game, which no shipped build could produce.
[D6](#d6--the-retail-build-configuration-and-the-version-identity) decides which
of two mutually exclusive behaviour sets to implement.
[D7](#d7--demo-playback-termination) is a runtime-safety decision, not a format one,
but it was open in the same scattered way and belongs here.

## Decision summary

| ID | Decision (one line) | Binding on |
|---|---|---|
| D1 | `sizeof(game2Type)` is **110** (0x6E). The header's `// total = 114` is an arithmetic slip that counts the flexible array's phantom `// 4`. | `.gliG` reader, if any |
| D2 | The shipping build used **`#pragma options align=mac68k`** (== `pack(2)`), so `sizeof(houseType)` is **866** and `sizeof(roomType)` is **348**. Every *field offset* in `houseType`, `roomType`, `gameType`, `savedRoom` and `scoresType` is identical under both alignment models; only `sizeof(houseType)` (866 vs 868) and `sizeof(game2Type)` (110 vs 112) differ. | everything |
| D3 | Trailing bytes beyond `866 + 348 * nRooms` are **allocation slack**: **accept** them, **preserve** them byte-for-byte on rewrite, **never** reject the file, and **never** derive `nRooms` from the file length alone. In `Sampler` they are the 2-byte tail pad of a `houseType` compiled at natural alignment (868). | loader + writer |
| D4 | `houseType.unusedShort` (+2), `houseType.unusedBoolean` (+861), the whole `houseType.savedGame` block (+820..861) and `houseType.hasGame` (+860) hold **nondeterministic residue**. Byte-exact round-tripping **requires** preserving all of them verbatim. Never validate, never zero, never interpret. `roomType.unusedByte` (+32) is the exception: the original force-zeroes it. | loader + writer |
| D5 | There is no `kMaxRooms`. The normative load-time bound is `nRooms = min(headerNRooms, (fileLen - 866) / 348)` with `1 <= nRooms <= 32767`; the largest *legal* house holds **8192** addressable rooms plus any number of placeholders. Observed corpus maximum is **531**. | loader |
| D6 | Retail was **`BUILD_ARCADE_VERSION` = 0**. This GPL drop is a later Carbon-era work-in-progress, not the retail build. The product version is **1.1.2**; "1.0.4" is a stale source-header comment. Implement the non-arcade behaviours; expose the arcade paths as an optional kiosk mode. | behaviour |
| D7 | Demo playback is terminated by the **game** (death → `countDown` 16 → `DoDiedGameOver` → `playing = false`), not by the stream. There is no sentinel and no length field; `'demo'` 128 is 6702 bytes = **1117** six-byte records spanning frames 46..3414. A port must add the one bound the original lacks — `demoIndex < len(demo)` — and thereafter feed no input. Demo key codes are **0 = right, 1 = left, 2 = battery/helium, 3 = rubber band**; the source comments on cases 0 and 1 are swapped. | runtime |

---

## 0. Method and evidence apparatus

### 0.1 Line-number convention

Every `.c` / `.h` file under `GliderPRO/` uses classic-Mac **CR-only** (0x0D)
line endings. Each was converted with `tr '\r' '\n'` into `/tmp/wf-fmtdec/`
before reading. **All line numbers cited here are line numbers in the CR→LF
converted copy** — which is what `git`, `grep`, `sed` and every modern editor
will report once the files are normalised. Paths are given relative to the
repository root, e.g. `GliderPRO/Sources/HouseIO.c:473`.

One operational warning, because it produced false negatives during this
investigation: the sources contain MacRoman high bytes (0xC5 `…`, 0xAA `™`,
0xA9 `©`), so tools that auto-detect binary files skip them silently. `ugrep`
with `-I`, and any `grep` shell function that adds `-I`, will report "no match"
for constants that are plainly there. Use `command grep -an` (bypass the
function, force text mode) or the search will lie to you.

### 0.2 The two alignment models, compiled

Classic-Mac C compilers offered two struct layout rules, and Glider PRO's record
header selects **neither** explicitly — `GliderPRO/Headers/GliderStructs.h` has
no `#pragma options align=` anywhere in its 347 lines. (Contrast
`GliderPRO/Headers/Externs.h:231`, which *does* wrap `prefsInfo` in
`#pragma options align=mac68k` and closes it at `:269`.) So `GliderStructs.h`
inherits whatever the project-level setting was, and the project file is not in
the GPL drop. That is the root cause of D1, D2 and D3 all being open.

The two candidate rules:

| Model | Rule | CodeWarrior name | Apple pragma |
|---|---|---|---|
| 68k / mac68k | every member aligned to `min(2, sizeof(member))`; struct aligned to 2 | "68K" | `#pragma options align=mac68k` |
| PowerPC natural | every member aligned to `min(4, sizeof(member))` (so `long` on 4); struct aligned to its widest member | "PowerPC" | `#pragma options align=power` (default) |

`#pragma options align=mac68k` is byte-for-byte equivalent to GCC's
`#pragma pack(2)` for every struct in this program, because no member is wider
than 4 bytes and none is a `double`.

`/tmp/wf-fmtdec/sizes.c` transcribes all of the relevant records twice — once
under `#pragma pack(push,2)` (identifiers prefixed `p2`) and once under GCC's
natural alignment (prefixed `n`) — and prints `sizeof`, `_Alignof` and a full
`offsetof` map for each. Mac scalar definitions used:

```c
typedef unsigned char Boolean;      /* Boolean and Byte are both 1 byte */
typedef unsigned char Byte;
typedef int32_t       Long;         /* Mac 'long' is ALWAYS 32-bit, even on PPC */
typedef struct { short v, h; }                     Point;   /* v FIRST */
typedef struct { short top, left, bottom, right; } Rect;
typedef unsigned char Str15[16], Str27[28], Str31[32], Str63[64], Str255[256];
typedef struct { short vRefNum; Long parID; Str63 name; } FSSpec;   /* 70 */
```

`FSSpec` is forced to `pack(2)` in *both* variants, because Apple's `Files.h`
wraps its own declaration in `align=mac68k`: a Mac `FSSpec` is 70 bytes
(2 + 4 + 64) in every compilation mode, never 72. Getting this wrong is the
single easiest way to mis-size `game2Type`.

Constants come from `GliderPRO/Headers/GliderDefines.h`: `kMaxScores` 10
(`:249`), `kMaxRoomObs` 24 (`:250`), `kNumTiles` 8 (`:496`).

Compiled output, `pack(2)` column and natural column side by side:

| Record | `GliderStructs.h` | pack(2) size | pack(2) align | natural size | natural align | header comment |
|---|---:|---:|---:|---:|---:|---|
| `Point` | *(Apple)* | 4 | 2 | 4 | 2 | — |
| `Rect` | *(Apple)* | 8 | 2 | 8 | 2 | — |
| `FSSpec` | *(Apple)* | 70 | 2 | 70 | 2 | `// 70` |
| `blowerType` | `:11-19` | 10 | 2 | 10 | 2 | `// total = 10` |
| `furnitureType` | `:21-25` | 10 | 2 | 10 | 2 | `// total = 10` |
| `bonusType` | `:27-34` | 10 | 2 | 10 | 2 | `// total = 10` |
| `transportType` | `:36-43` | 10 | 2 | 10 | 2 | `// total = 10` |
| `switchType` | `:45-52` | 10 | 2 | 10 | 2 | `// total = 10` |
| `lightType` | `:54-62` | 10 | 2 | 10 | 2 | `// total = 10` |
| `applianceType` | `:64-72` | 10 | 2 | 10 | 2 | `// total = 10` |
| `enemyType` | `:74-82` | 10 | 2 | 10 | 2 | `// total = 10` |
| `clutterType` | `:84-88` | 10 | 2 | 10 | 2 | `// total = 10` |
| `objectType` | `:90-105` | **12** | 2 | **12** | 2 | `// total = 12` |
| `scoresType` | `:107-114` | **292** | 2 | **292** | 4 | `// total = 292` |
| `gameType` | `:116-134` | **40** | 2 | **40** | 4 | `// total = 40` |
| `savedRoom` | `:136-142` | **292** | 2 | **292** | 2 | `// total = 292` |
| `game2Type` | `:144-164` | **110** | 2 | **112** | 4 | `// total = 114` ← **wrong** |
| `roomType` | `:166-180` | **348** | 2 | **348** | 2 | `// total = 348` |
| `houseType` | `:182-198` | **866** | 2 | **868** | 4 | `// total = 866 +` |
| `demoType` | `:334-339` | **6** | 2 | **8** | 4 | *(none)* |

Two observations that carry the rest of this document:

1. Only **three** records change size between the models: `game2Type`
   (110 → 112), `houseType` (866 → 868) and `demoType` (6 → 8). Every other
   record — including `roomType`, the one that tiles the file — is
   alignment-invariant, because all of its members are 1 or 2 bytes wide.
2. Of those three, `game2Type` and `houseType` change size **only in their
   trailing pad**. Every declared field keeps its offset. `demoType` is the
   exception: its `key`/`padding` bytes move from 4/5 to 4/5 but the record
   stride changes from 6 to 8, which *would* garble the `'demo'` resource. See
   [D7](#d7--demo-playback-termination) for why that settles the model
   independently.

### 0.3 The corpus: 22 shipped houses

`GliderPRO/Houses/` holds 22 BinHex 4.0 files. `tools/probe_house.py` decodes
them (`binhex_decode(path)` returns a dict with `name, version, type, creator,
flags, data, rsrc, hdr_crc, data_crc, rsrc_crc`). All 22 decode with valid CRCs;
all 22 have `houseType.version == 0x0200` (== `kHouseVersion`,
`GliderPRO/Headers/GliderDefines.h:517`); together they contain **4070** rooms.

Full observed header table. `slack` is `len(dataFork) - (866 + 348 * nRooms)`.
`timeStamp` is the raw stored `long`; `date(|bit31)` is that value with bit 31
restored, because `WriteHouse` masks bit 31 off before writing
(`GliderPRO/Sources/HouseIO.c:478`) and the Mac epoch is 1904-01-01, so every
stored stamp decodes to a nonsensical 1927/1932 date unless you put bit 31 back.

| House | data fork | rsrc fork | nRooms | 866+348n | slack | unusedShort | uBool | hasGame | firstRoom | flags | timeStamp | date (bit 31 restored) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| Art Museum | 38798 | 7476159 | 109 | 38798 | 0 | 0x8A7E | 255 | 0 | 91 | 0x06 | 746622005 | 1995-09-16 14:14:13 |
| CD Demo House | 72554 | 1612342 | 206 | 72554 | 0 | 0x0000 | 0 | 0 | 70 | 0x02 | 742294873 | 1995-07-28 12:15:21 |
| California or Bust! | 6434 | 259542 | 16 | 6434 | 0 | 0x3333 | 0 | 0 | 14 | 0x02 | 745882296 | 1995-09-08 00:45:44 |
| Castle o' the Air | 30446 | 306364 | 85 | 30446 | 0 | 0x0103 | 30 | 0 | 33 | 0x00 | 741536868 | 1995-07-19 17:41:56 |
| Davis Station | 23486 | 1871320 | 65 | 23486 | 0 | 0x00DE | 37 | 0 | 4 | 0x02 | 743681789 | 1995-08-13 13:30:37 |
| Demo House | 16526 | 491757 | 45 | 16526 | 0 | 0x0000 | 0 | 0 | 0 | 0x00 | 743682113 | 1995-08-13 13:36:01 |
| Empty House | 13046 | 2670 | 35 | 13046 | 0 | 0x0000 | 0 | 0 | 0 | 0x00 | 741421292 | 1995-07-18 09:35:40 |
| Fun House | 15830 | 662443 | 43 | 15830 | 0 | 0x6674 | 0 | 0 | 29 | 0x00 | 742681616 | 1995-08-01 23:41:04 |
| Grand Prix | 61766 | 1347764 | 175 | 61766 | 0 | 0x003C | 185 | 0 | 127 | 0x00 | 741444157 | 1995-07-18 15:56:45 |
| ImagineHouse PRO II | 97958 | 677770 | 279 | 97958 | 0 | 0x0000 | 0 | **1** | 1 | 0x00 | 740149565 | 1995-07-03 16:20:13 |
| In The Mirror | 34622 | 151870 | 97 | 34622 | 0 | 0x0093 | 0 | 0 | 6 | 0x00 | 741512193 | 1995-07-19 10:50:41 |
| Land of Illusion | 106310 | 401793 | 303 | 106310 | 0 | 0x0103 | 30 | 0 | 43 | 0x02 | 745268052 | 1995-08-31 22:08:20 |
| Leviathan | 165122 | 1901282 | 472 | 165122 | 0 | 0x0000 | 0 | 0 | 39 | 0x00 | 741514803 | 1995-07-19 11:34:11 |
| Metropolis | 45062 | 946907 | 127 | 45062 | 0 | 0x00C4 | 14 | 0 | 8 | 0x00 | 741642315 | 1995-07-20 22:59:23 |
| Nemo's Market | 44018 | 590602 | 124 | 44018 | 0 | 0x0000 | 0 | 0 | 0 | 0x02 | 741121233 | 1995-07-14 22:14:41 |
| Rainbow's End | 78470 | 316065 | 223 | 78470 | 0 | 0x0000 | 0 | 0 | 30 | 0x02 | 741121489 | 1995-07-14 22:18:57 |
| **Sampler** | **1564** | **286** | **2** | **1562** | **+2** | 0x0000 | 2 | 0 | 1 | 0x00 | 893435880 | **2000-05-11 19:52:08** |
| Slumberland | 134150 | 1041112 | 383 | 134150 | 0 | 0x0000 | 0 | 0 | 126 | 0x00 | 743682377 | 1995-08-13 13:40:25 |
| SpacePods | 140762 | 636844 | 402 | 140762 | 0 | 0x0000 | 0 | 0 | 259 | 0x02 | 741122073 | 1995-07-14 22:28:41 |
| Teddy World | 185654 | 3107348 | **531** | 185654 | 0 | 0x0000 | 0 | 0 | 0 | 0x00 | 744401697 | 1995-08-21 21:29:05 |
| The Asylum Pro | 49586 | 494107 | 140 | 49586 | 0 | 0x0000 | 0 | 0 | 20 | 0x00 | 738016929 | 1995-06-08 23:56:17 |
| Titanic | 73250 | 845727 | 208 | 73250 | 0 | 0x081A | 0 | **1** | 92 | 0x00 | 755123966 | 1995-12-23 23:53:34 |

Derived facts used later:

* 21 of 22 satisfy `len == 866 + 348 * nRooms` **exactly**. `Sampler` is the
  only exception, by exactly **2** bytes.
* Every non-`Sampler` house body was last written between **1995-06-08** and
  **1995-12-23**. `Sampler` was last written **2000-05-11**, four and a half
  years later, and is the only house with an effectively empty resource fork
  (286 bytes, zero resources).
* 7 of 22 are unlocked (`timeStamp & 1 == 0`): California or Bust!,
  Castle o' the Air, Empty House, Fun House, Land of Illusion, Sampler, Titanic.
  The other 15 are locked.
* `flags` only ever takes the values 0x00 (**14** houses), 0x02 (**7**: CD Demo
  House, California or Bust!, Davis Station, Land of Illusion, Nemo's Market,
  Rainbow's End, SpacePods) and 0x06 (1, Art Museum). Bit 0 (`wardBit`,
  `GliderPRO/Sources/HouseIO.c:416`) is never set in the corpus. Bit 1 is
  `phoneBitSet` (`:417`); bit 2 is **inverted** —
  `bannerStarCountOn = ((flags & 4) == 0)` (`:418`) — so Art Museum is the only
  house with the banner star count *off*.
* `Demo House` is 16526 bytes with 45 rooms. **16526 == 866 + 348 × 45**
  exactly. This is the load-bearing arithmetic for
  [D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant).

### 0.4 Resource extraction

`GliderPRO/Glider PRO.r` is a derez'ed text dump of the application's resource
fork (199,843 lines; unlike the sources, this file already uses LF endings). Two
resources matter here:

* `data 'demo' (128)` — begins at line 199389, closes at line 199809; 6702 bytes
  when the hex is reassembled. Extracted to `/tmp/wf-fmtdec/demo128.bin`.
* `data 'vers' (1)` at line 6135 and `data 'vers' (2)` at line 6142.

The string `1.0.4` occurs **nowhere** in `Glider PRO.r`.

---

## D1 — `sizeof(game2Type)` is 110, not 114

### The question as previously posed

* **architecture.md OQ14:** "`game2Type` on-disk size. The header says 114 but
  `sizeof` is 110 (§A.2). Which one the original writer actually wrote to disk
  depends on whether `SavedGames.c` used `sizeof(game2Type)` or the literal.
  That file is outside this document's scope; a porter must check
  `SavedGames.c:20` onwards before trusting either number."
* **progression.md OQ1:** "the struct comment says 114 … but the fields sum to
  110 under `align=mac68k` … the on-disk offset of `savedData[0]` cannot be
  determined from the sources."
* **scoring.md OQ7:** "Field arithmetic gives **110**; the struct's own comment
  says 'total = 114' … anyone trying to *read* a `.gliG` file produced by a
  pre-release build needs to know which."
* **constants.md OQ8:** "Was `game2Type` / the external `.gliG` saved game ever
  functional?"

Three documents give three different framings and none commits. The task
instruction was explicit: transcribe the struct, compile it under `pragma
pack(2)` with 32-bit `long`, and fix the number. Done below.

### Evidence

The declaration, verbatim, with the author's own per-field byte comments
(`GliderPRO/Headers/GliderStructs.h:144-164`):

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
    savedRoom   savedData[];    // 4      <-- the culprit
} game2Type, *gamePtr;          // total = 114
```

Compiled `offsetof` map, both models:

| Field | Type | Size | Offset, pack(2) | Offset, natural |
|---|---|---:|---:|---:|
| `house` | `FSSpec` | 70 | 0 (0x00) | 0 |
| `version` | `short` | 2 | 70 (0x46) | 70 |
| `wasStarsLeft` | `short` | 2 | 72 (0x48) | 72 |
| `timeStamp` | `long` | 4 | **74 (0x4A)** | **76** |
| `where` | `Point` | 4 | 78 (0x4E) | 80 |
| `score` | `long` | 4 | 82 (0x52) | 84 |
| `unusedLong` | `long` | 4 | 86 (0x56) | 88 |
| `unusedLong2` | `long` | 4 | 90 (0x5A) | 92 |
| `energy` | `short` | 2 | 94 (0x5E) | 96 |
| `bands` | `short` | 2 | 96 (0x60) | 98 |
| `roomNumber` | `short` | 2 | 98 (0x62) | 100 |
| `gliderState` | `short` | 2 | 100 (0x64) | 102 |
| `numGliders` | `short` | 2 | 102 (0x66) | 104 |
| `foil` | `short` | 2 | 104 (0x68) | 106 |
| `nRooms` | `short` | 2 | 106 (0x6A) | 108 |
| `facing` | `Boolean` | 1 | 108 (0x6C) | 110 |
| `showFoil` | `Boolean` | 1 | 109 (0x6D) | 111 |
| `savedData[0]` | `savedRoom` | 292 | **110 (0x6E)** | **112** |
| | | | **`sizeof` = 110** | **`sizeof` = 112** |

Note the divergence point: `timeStamp` at offset 74 is 2-byte-aligned. Under
natural alignment a `long` must sit on a multiple of 4, so the compiler inserts
2 pad bytes after `wasStarsLeft` and everything after shifts by 2. That single
pad is the whole 110-vs-112 story; it is *not* the 114 story.

Where does 114 come from? Add up the author's comments including the phantom
`// 4` on the flexible array:

```
70+2+2+4+4+4+4+4+2+2+2+2+2+2+2+1+1 = 110      (real fields)
                                 + 4 = 114     (the "// 4" on savedData[])
```

So 114 is 110 plus a byte count the author wrote next to a zero-length array.
Corroboration that this is a clerical slip and not a layout claim: every *other*
`// total =` comment in the header is arithmetically correct under `pack(2)`
(`scoresType` 292 at `:114`, `gameType` 40 at `:134`, `savedRoom` 292 at `:142`,
`roomType` 348 at `:180`, all nine object variants 10, `objectType` 12 at
`:105`), and the one other flexible-array record gets it *right* by refusing to
sum: `houseType`'s comment is `// total = 866 +` (`:198`) — with a trailing plus
sign, explicitly acknowledging the open-ended tail. `game2Type` is the only
record in the header where the author folded a flexible array into the total.

The pre-C99 idiom is also ruled out. If the field had been declared
`savedRoom savedData[1]` (the portable trick before C99 flexible array members),
`sizeof` would be `110 + 292 = 402`, not 114. Compiled and confirmed:
`p2game2Type_1 sizeof=402`.

Finally, does any *code* depend on the number? `game2Type`, `gamePtr`,
`savedRoom` and `saveRoomPtr` appear in exactly two files: their declaration in
`GliderPRO/Headers/GliderStructs.h`, and `GliderPRO/Sources/SavedGames.c`. In
`SavedGames.c` there is exactly one size computation, and it uses `sizeof`, not a
literal (`GliderPRO/Sources/SavedGames.c:56`):

```c
byteCount = sizeof(game2Type) + sizeof(savedRoom) * numRooms;
savedGame = (gamePtr)NewPtr(byteCount);
```

So the on-disk header size is whatever the compiler said — 110 under the
alignment model that [D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant)
establishes.

And both endpoints are dead in this build:

* `SaveGame2` (`GliderPRO/Sources/SavedGames.c:30-148`) has its entire body
  commented out, `:33` through `:147`, behind the note
  `// Add NavServices later.` at `:32`. The function compiles to nothing.
* `OpenSavedGame` (`GliderPRO/Sources/SavedGames.c:167-296`) begins with
  `return false;` at `:169` — the comment on that line is
  `// TEMP fix this iwth NavServices` — and the rest of the body, `:170-295`, is
  commented out.
* `OpenSavedGame` is called from exactly one place,
  `GliderPRO/Sources/Menu.c:320` (`case iOpenSavedGame:`), so
  `NewGame(kResumeGameMode)` is unreachable in this build.
* The file type it would have created is `'gliG'` with creator `'ozm5'`
  (`GliderPRO/Sources/SavedGames.c:122`); the type filter on read is
  `theList[0] = 'gliG'` (`:182`). No `'gliG'` file exists anywhere in the GPL
  release.

The *live* save path is entirely different: `SaveGame(Boolean doSave)`
(`GliderPRO/Sources/SavedGames.c:303-351`) writes a `gameType` (40 bytes) into
`houseType.savedGame` at house offset 820 and then calls
`WriteHouse(theMode == kEditMode)` at `:348`. That is the only saved-game format
the shipped program can produce, and it is part of the house file, not a
side-car.

### Decision

**`sizeof(game2Type)` is 110 (0x6E), and `savedData[0]` begins at offset 110.**
The `'gliG'` record layout is:

```
offset  0  .. 109   game2Type header (110 bytes, table above)
offset 110 + 292*r  savedRoom[r], r = 0 .. nRooms-1
total size          110 + 292 * nRooms
```

The header comment `// total = 114` in `GliderPRO/Headers/GliderStructs.h:164`
is **wrong** and must be ignored. Treat the field comment `// 4` on
`savedData[]` at `:163` as a typo for `// 292 * nRooms`.

**A Go port should not implement `'gliG'` at all** unless it specifically wants
to read files produced by some pre-1995 build that nobody has. Implement the
live in-house `savedGame` block instead
(`houseType` +820, 40 bytes, [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate)).
If you do implement `'gliG'`, use 110, and validate `version == 0x0200`
(`kSavedGameVersion`, `GliderPRO/Sources/SavedGames.c:14`) as
`OpenSavedGame` would have (`:243`).

### Consequence of deciding otherwise

If you use 114, every `savedData[r]` lands 4 bytes past its true position.
`savedRoom` is `{short unusedShort; Byte unusedByte; Boolean visited;
objectType objects[24];}` (offsets 0/2/3/4, `GliderStructs.h:136-142`), so a
4-byte shift consumes `unusedShort` + `unusedByte` + `visited` and starts
`objects[0]` inside what should have been `objects[0].what`... no — it starts
`objects[0]` at what was `objects[0].data` bytes 2..3. Concretely, reading a
2-room file:

| What you read at 114 + 292*r | What it actually is |
|---|---|
| `savedData[0].unusedShort` | bytes 4-5 of the real `savedData[0]`, i.e. `objects[0].what`'s high half plus a data byte |
| `savedData[0].visited` | byte 7 of the real record, a data byte of `objects[0]` |
| `savedData[0].objects[0].what` | bytes 8-9, `objects[0].data` bytes 4-5 |
| `savedData[1]` | starts 8 bytes late (4 for the header slip plus 4 accumulated) |

Observable symptom: on resuming, every room's "visited" flag is garbage (so the
map screen lights up wrong rooms), and every prize the player already collected
reappears or every uncollected prize vanishes, because `objects[i].what` is read
from the middle of the previous object's data. With `nRooms = 279`
(ImagineHouse PRO II) the last room's header would be read 4 × 279 = 1116 bytes
past the end of the file's real data, i.e. past the end of the buffer for a
tightly-sized read — a panic in Go, silent garbage on a Mac.

### Go

```go
// gliGHeaderSize is sizeof(game2Type) under mac68k alignment. It is 110, NOT
// the 114 written in GliderStructs.h:164 — that comment adds a phantom 4 bytes
// for the zero-length savedData[] member.
const gliGHeaderSize = 110
const savedRoomSize  = 292

// Do not implement this unless you have a real 'gliG' file. The shipping build
// could not create one: SavedGames.c:30-148 and :167-296 are both commented out.
```

---

## D2 — The alignment model: `houseType` is 866 bytes, and every field offset is alignment-invariant

### The question as previously posed

This one was never written down as an open question, which is worse: three
documents *assert* 866 without proof, and the header the assertion rests on has
no alignment pragma at all (`GliderPRO/Headers/GliderStructs.h`, 347 lines, zero
`#pragma options align` directives — contrast `GliderPRO/Headers/Externs.h:231`
and `:269`, which do wrap `prefsInfo`). D1 and D3 both reduce to this question,
so it is settled here explicitly.

### Evidence

**1. The `COMPILEDEMO` magic number pins it exactly.** `ReadHouse` contains a
hard-coded file-length check used by the demo/shareware build
(`GliderPRO/Sources/HouseIO.c:348-351`):

```c
theErr = GetEOF(houseRefNum, &byteCount);
...
#ifdef COMPILEDEMO
if (byteCount != 16526L)
    return (false);
#endif
```

and, 30 lines later, the matching room count (`:380-384`):

```c
numberRooms = (*thisHouse)->nRooms;
#ifdef COMPILEDEMO
if (numberRooms != 45)
    return (false);
#endif
```

Both refer to the single house the demo build was allowed to open. `Demo House`
in the corpus is **16526 bytes with `nRooms == 45`** — an exact match. And:

```
866 + 348 * 45 = 866 + 15660 = 16526     <-- matches
868 + 348 * 45 = 868 + 15660 = 16528     <-- does not
```

The author typed `16526` while compiling against his own `houseType`. That is a
direct, unambiguous measurement of `sizeof(houseType)` in the build that wrote
that literal: **866**.

**2. `ValidateNumberOfRooms` would have destroyed the corpus under the other
model.** The house checker recomputes the room count from the handle size
(`GliderPRO/Sources/HouseLegal.c:629-637`):

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

For a house of exactly `866 + 348n` bytes:

| `sizeof(houseType)` | `countedRooms` | verdict |
|---|---|---|
| 866 | `(866 + 348n − 866) / 348 = n` | agrees, no error |
| 868 | `(866 + 348n − 868) / 348 = (348n − 2) / 348 = n − 1` (integer division) | disagrees: silently deletes the last room and bumps `houseErrors` |

`ValidateNumberOfRooms` runs on every edit-mode save
(`WriteHouse(theMode == kEditMode)` → `CheckHouseForProblems()` at
`GliderPRO/Sources/HouseIO.c:470` → `ValidateNumberOfRooms()` at
`GliderPRO/Sources/HouseLegal.c:1075`, gated only by `isHouseChecks`). If the
shipping editor had been compiled at natural alignment, every one of the 21
exact houses would have lost its last room and displayed the localised
"room count corrected" message the first time it was saved. All 21 shipped with
`nRooms` consistent with their length. Therefore the editor that produced them
computed `sizeof(houseType) == 866`.

**3. Every field offset is the same under both models, so nothing else in the
loader depends on the answer.** Compiled `offsetof` maps, side by side:

| `houseType` field | Type | Size | pack(2) | natural | Same? |
|---|---|---:|---:|---:|:-:|
| `version` | `short` | 2 | 0 (0x000) | 0 | yes |
| `unusedShort` | `short` | 2 | 2 (0x002) | 2 | yes |
| `timeStamp` | `long` | 4 | 4 (0x004) | 4 | yes |
| `flags` | `long` | 4 | 8 (0x008) | 8 | yes |
| `initial` | `Point` | 4 | 12 (0x00C) | 12 | yes |
| `banner` | `Str255` | 256 | 16 (0x010) | 16 | yes |
| `trailer` | `Str255` | 256 | 272 (0x110) | 272 | yes |
| `highScores` | `scoresType` | 292 | 528 (0x210) | 528 | yes |
| `savedGame` | `gameType` | 40 | 820 (0x334) | 820 | yes |
| `hasGame` | `Boolean` | 1 | 860 (0x35C) | 860 | yes |
| `unusedBoolean` | `Boolean` | 1 | 861 (0x35D) | 861 | yes |
| `firstRoom` | `short` | 2 | 862 (0x35E) | 862 | yes |
| `nRooms` | `short` | 2 | 864 (0x360) | 864 | yes |
| `rooms[0]` | `roomType` | 348 | **866 (0x362)** | **866** | yes |
| | | | `sizeof` = **866** | `sizeof` = **868** | **no** |

The reason `rooms[]` does not move is that `houseType`'s two `long`s are at
offsets 4 and 8 — already 4-aligned — and no member after them is wider than 2
bytes, so natural alignment inserts no interior padding. The only difference is
the 2-byte *trailing* pad natural alignment appends to round `sizeof` up to the
struct's 4-byte alignment (imposed by `gameType`, which contains `long`s and
therefore has `_Alignof == 4`).

`roomType` is likewise identical in both models — 348 bytes, and every field at
the same place:

| `roomType` field | Type | Size | Offset | Notes |
|---|---|---:|---:|---|
| `name` | `Str27` | 28 | 0 (0x00) | Pascal string, length byte + up to 27 MacRoman chars |
| `bounds` | `short` | 2 | 28 (0x1C) | a bit field, **not** a `Rect` |
| `leftStart` | `Byte` | 1 | 30 (0x1E) | |
| `rightStart` | `Byte` | 1 | 31 (0x1F) | |
| `unusedByte` | `Byte` | 1 | 32 (0x20) | force-zeroed, see D4 |
| `visited` | `Boolean` | 1 | 33 (0x21) | |
| `background` | `short` | 2 | 34 (0x22) | `PICT` ID |
| `tiles[8]` | `short[8]` | 16 | 36 (0x24) | `kNumTiles` = 8 |
| `floor` | `short` | 2 | 52 (0x34) | |
| `suite` | `short` | 2 | 54 (0x36) | `-1` == `kRoomIsEmpty` |
| `openings` | `short` | 2 | 56 (0x38) | recomputed at load |
| `numObjects` | `short` | 2 | 58 (0x3A) | |
| `objects[24]` | `objectType[24]` | 288 | 60 (0x3C) | `kMaxRoomObs` = 24, stride 12 |
| | | | **348 (0x15C)** | |

And `gameType` (the in-house saved game, 40 bytes), whose offsets are also
alignment-invariant because its `long`s land on 4-aligned offsets by luck:

| `gameType` field | Type | Size | Offset in `gameType` | Absolute in `houseType` |
|---|---|---:|---:|---:|
| `version` | `short` | 2 | 0 | 820 |
| `wasStarsLeft` | `short` | 2 | 2 | 822 |
| `timeStamp` | `long` | 4 | 4 | 824 |
| `where` | `Point` | 4 | 8 | 828 |
| `score` | `long` | 4 | 12 | 832 |
| `unusedLong` | `long` | 4 | 16 | 836 |
| `unusedLong2` | `long` | 4 | 20 | 840 |
| `energy` | `short` | 2 | 24 | 844 |
| `bands` | `short` | 2 | 26 | 846 |
| `roomNumber` | `short` | 2 | 28 | 848 |
| `gliderState` | `short` | 2 | 30 | 850 |
| `numGliders` | `short` | 2 | 32 | 852 |
| `foil` | `short` | 2 | 34 | 854 |
| `unusedShort` | `short` | 2 | 36 | 856 |
| `facing` | `Boolean` | 1 | 38 | 858 |
| `showFoil` | `Boolean` | 1 | 39 | 859 |
| | | | **40** | |

`scoresType` (the in-house high-score table, 292 bytes at house offset 528):

| `scoresType` field | Type | Size | Offset | Absolute |
|---|---|---:|---:|---:|
| `banner` | `Str31` | 32 | 0 | 528 |
| `names[10]` | `Str15[10]` | 160 | 32 | 560 |
| `scores[10]` | `long[10]` | 40 | 192 | 720 |
| `timeStamps[10]` | `unsigned long[10]` | 40 | 232 | 760 |
| `levels[10]` | `short[10]` | 20 | 272 | 800 |
| | | | **292** | |

**4. A cross-check that the on-disk stamps really are what the code says.**
`WriteHouse` masks bit 31 out of `houseType.timeStamp`
(`GliderPRO/Sources/HouseIO.c:477-488`):

```c
GetDateTime(&timeStamp);
timeStamp &= 0x7FFFFFFF;
if (changeLockStateOfHouse)
    houseUnlocked = !saveHouseLocked;
if (houseUnlocked)                  // house unlocked
    timeStamp &= 0x7FFFFFFE;
else
    timeStamp |= 0x00000001;
(*thisHouse)->timeStamp = (long)timeStamp;      /* :487 */
(*thisHouse)->version = wasHouseVersion;        /* :488 */
```

(The `version` write at `:488` is part of the same `if (fileDirty)` block:
`wasHouseVersion` is captured from the file at load (`HouseIO.c:394`) and written
back on every dirty save, so a port must round-trip `version` from the loaded
value rather than hard-coding `kHouseVersion`.)

but it does **not** touch `highScores.timeStamps[]`, which is filled from a raw
`GetDateTime(&thisHousePtr->highScores.timeStamps[kMaxScores - 1])` at
`GliderPRO/Sources/HighScores.c:411` and then bubbled up the table. Observed: all 22 stored
`houseType.timeStamp` values have bit 31 **clear** (raw values 738016929 …
893435880, all < 2^31), while every non-zero `highScores.timeStamps[0]` has bit
31 **set** (2885500804 … 3040919567, all > 2^31). Two fields, four bytes apart
in spirit, with exactly the bit-31 asymmetry the code predicts. This confirms
that offsets 4 and 760 are being read correctly, i.e. that the 866-byte layout
above is the real one.

### Decision

**Compile every record in `GliderPRO/Headers/GliderStructs.h` as if wrapped in
`#pragma options align=mac68k`, i.e. GCC `#pragma pack(2)`.** Normatively:

```
sizeof(objectType)   =  12
sizeof(roomType)     = 348   (0x15C)
sizeof(gameType)     =  40
sizeof(scoresType)   = 292
sizeof(savedRoom)    = 292
sizeof(houseType)    = 866   (0x362)   <-- NOT 868
sizeof(game2Type)    = 110   (0x06E)   <-- NOT 112, NOT 114
sizeof(demoType)     =   6              <-- NOT 8
```

House file length is `866 + 348 * nRooms` (plus possible slack, see
[D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length)).
All multi-byte scalars are **big-endian**. `Point` is `{short v; short h;}` —
**vertical first**; `Rect` is `{top, left, bottom, right}`.

In Go, do not rely on `unsafe.Sizeof` of a mirrored struct — Go's layout rules
are neither of the two Mac models. Write explicit `encoding/binary` marshalling
against the offset tables above, and define the sizes as named constants.

### Consequence of deciding otherwise

Choosing 868 breaks two things, one loudly and one silently:

1. **Silently, on write:** a port that recomputes the file length as
   `868 + 348 * nRooms` writes 2 extra bytes onto every house. The original's
   `ValidateNumberOfRooms` (which real Glider PRO users would still run against
   your files) then computes `(868 + 348n − 866) / 348 = n`, so it *happens* to
   survive — but a second round-trip through your writer gives `870 + 348n`,
   then `872 + 348n`, and at the 174th round-trip the slack exceeds 348 and the
   original loader counts a phantom extra room, reads a `roomType` made of
   slack, and `suite` is whatever the slack held. Since `suite == -1` means
   "empty", a phantom room whose slack happens to be `0xFFFF` at +54 is
   compressed away by `CompressHouse`; anything else becomes a real room at a
   garbage floor/suite.
2. **Loudly, on read, if you mirror `ValidateNumberOfRooms`:** you would drop
   the last room of all 21 exact houses. `Rainbow's End` would lose room 222,
   `Teddy World` room 530. If the lost room is on the path to the exit, the
   house becomes unwinnable; if it is `firstRoom`, `GetFirstRoomNumber`
   (`GliderPRO/Sources/House.c:196-217`) clamps `firstRoom` to 0 and the player
   starts in the wrong room.

Choosing 8 for `sizeof(demoType)` is unambiguously fatal — see
[D7](#d7--demo-playback-termination).

### Go

```go
const (
    SizeObject    = 12
    SizeRoom      = 348
    SizeGame      = 40
    SizeScores    = 292
    SizeSavedRoom = 292
    SizeHouse     = 866 // sizeof(houseType) under align=mac68k
    SizeGame2     = 110 // sizeof(game2Type)  under align=mac68k
    SizeDemoRec   = 6   // sizeof(demoType)   under align=mac68k

    OffHouseVersion   = 0
    OffHouseUnused    = 2
    OffHouseTimeStamp = 4
    OffHouseFlags     = 8
    OffHouseInitial   = 12
    OffHouseBanner    = 16
    OffHouseTrailer   = 272
    OffHouseScores    = 528
    OffHouseSavedGame = 820
    OffHouseHasGame   = 860
    OffHouseUnusedB   = 861
    OffHouseFirstRoom = 862
    OffHouseNRooms    = 864
    OffHouseRooms     = 866
)

func RoomOffset(i int) int { return OffHouseRooms + SizeRoom*i }
```

---

## D3 — `Sampler.binhex`'s two trailing bytes: accept, preserve, never reject, never size by file length

### The question as previously posed

Five documents noticed the same 2 bytes and gave four different answers:

* **object-taxonomy.md OQ1:** "Is the extra pair trailing slop written by an
  older `WriteHouse`, or a 2-byte field appended after `rooms[]` that the
  released `houseType` no longer declares? … the excess is invisible to the
  loader and **can be ignored**, but its origin is unexplained."
* **interactions.md OQ5:** "The trailing bytes are `... 00 20 00 00 01 01 01 01`.
  Is this a truncated third room record, or editor slack?"
* **constants.md OQ9:** "Truncated/padded write, BinHex artefact, or deliberate?
  … **Experiment:** dump the last 8 bytes and check whether they are zero or the
  tail of a third `roomType`."
* **editor.md Q4:** "**What a port should do.** Ignore trailing bytes beyond
  `866 + 348 * nRooms` on load; **do not write them back.**"
* **house-format.md §12.4** and **scoring.md OQ24** carry the same observation.

Note that editor.md Q4 and object-taxonomy.md OQ1 give *incompatible* advice
("do not write them back" vs "invisible … can be ignored"). This decision
supersedes editor.md Q4 on the write side.

### Evidence

**1. The bytes.** Decoded data fork is 1564 bytes; `nRooms == 2`; `rooms[1]`
occupies 1214..1561. Last 32 bytes:

```
05FC  00 20 00 00 01 01 ff ff 01 2f 00 d0 00 20 00 00
060C  01 01 ff ff 01 2f 00 a1 00 20 00 00 01 01 01 01
                                          ^^^^^ ^^^^^
                                          |     +--- 1562..1563  = the slack
                                          +--------- 1560..1561  = end of rooms[1]
```

So the slack is `01 01` and it is **byte-identical to the two bytes immediately
before it**. That is not a coincidence; see point 4.

**2. It is not a truncated third room, and it is not BinHex damage.** A
`roomType` is 348 bytes; 2 is not a prefix of anything meaningful (`name[0]`
would be 1, `name[1]` = 0x01, and then the record would stop). The BinHex
decode is clean: `tools/probe_house.py`'s `binhex_decode(path, check_crc=True)`
recomputes the CRC-16 (`x^16 + x^12 + x^5 + 1`, `probe_house.py:78-92`) over both
forks and compares against the stored values; **all 22 houses pass, 0 fail**. So
1564 is the length the file really claims and the bytes are the bytes that were
written. Note that `check_crc` defaults to `False`, so a casual call does not
validate — pass it explicitly.

**3. `1564 = 868 + 348 × 2` exactly** — where 868 is `sizeof(houseType)` under
**natural (PowerPC) alignment** ([§0.2](#02-the-two-alignment-models-compiled)).
The only structural difference between the two alignment models is that
2-byte trailing pad, and `Sampler` is exactly 2 bytes long. No other arithmetic
produces 1564 from 348-byte rooms.

**4. The mechanism, traced through the editor.** `InitializeEmptyHouse`
allocates the bare header (`GliderPRO/Sources/House.c:116`):

```c
thisHouse = (houseHand)NewHandle(sizeof(houseType));
```

— so a fresh house handle is `sizeof(houseType)` bytes: 866 under `pack(2)`,
**868** under natural alignment. `CreateNewRoom` then grows it by appending at
the *end of the handle* (`GliderPRO/Sources/Room.c:198-200`):

```c
howMuch = sizeof(roomType);          // add new room to end of house
theErr = PtrAndHand((Ptr)thisRoom, (Handle)thisHouse, howMuch);
```

and immediately afterwards writes the room again, this time through the struct
member, via `CopyThisRoomToRoom()` at `GliderPRO/Sources/Room.c:226`. Trace an
868-byte build creating `Sampler`'s two rooms:

| Step | Handle size | `PtrAndHand` wrote to | `rooms[i]` really lives at | Result |
|---|---:|---|---|---|
| `InitializeEmptyHouse` | 868 | — | — | bytes 866..867 are uninitialised (`NewHandle`, not `NewHandleClear`) |
| `CreateNewRoom` #1 | 1216 | 868..1215 | `rooms[0]` = 866..1213 | `CopyThisRoomToRoom` rewrites 866..1213 correctly; 1214..1215 keep the tail of the `PtrAndHand` copy |
| `CreateNewRoom` #2 | 1564 | 1216..1563 | `rooms[1]` = 1214..1561 | `CopyThisRoomToRoom` rewrites 1214..1561 correctly; **1562..1563 keep the tail of the `PtrAndHand` copy** |

The last row makes a falsifiable prediction. The final `PtrAndHand` copied
`thisRoom[0..347]` to 1216..1563, so byte 1562 held `thisRoom[346]` and byte 1563
held `thisRoom[347]`. `CopyThisRoomToRoom` then wrote the *same, unchanged*
`thisRoom` to 1214..1561, so `thisRoom[346]` also landed at 1560 and
`thisRoom[347]` at 1561. Prediction: **`d[1562] == d[1560]` and
`d[1563] == d[1561]`.** Observed: `d[1560:1564] == 01 01 01 01` — the prediction
holds. (Both bytes happen to be 0x01, so this is corroboration rather than
proof; the values come from `objects[23]`'s last two data bytes in a slot whose
`what == -1`, and slots **16..23** of that room all show the same
`01 2f 00 xx 00 20 00 00 01 01` residue pattern — see the dump in point 6.
Slots 4..15 hold different residue, so the repeating pattern covers the last
eight slots, not the whole `objects[4..23]` tail.)

Note that the mis-landing self-heals: because `CopyThisRoomToRoom` rewrites the
room at the struct-defined offset, the appended copy is only ever used to grow
the handle. That is why the file is otherwise perfectly well-formed and why
nobody noticed for 25 years.

**5. Both alignment models load `Sampler` without complaint**, which is the other
half of why nobody noticed. `ValidateNumberOfRooms` computes with integer
division:

```
866-build:  (1564 − 866) / 348 = 698 / 348 = 2 == nRooms   -> no error
868-build:  (1564 − 868) / 348 = 696 / 348 = 2 == nRooms   -> no error
```

And the 2 bytes are past `rooms[nRooms-1]`, so no field read ever touches them.

**6. The date corroborates a different build.** `Sampler`'s stored
`houseType.timeStamp` is 893435880 (0x3540BFE8); restoring the bit-31 that
`WriteHouse` masked off (`GliderPRO/Sources/HouseIO.c:478`) gives 3040919528 →
**2000-05-11 19:52:08**. Its `highScores.timeStamps[0]` is 3040919567 →
**2000-05-11 19:52:47**, 39 seconds later. Every other house body dates to
1995-06-08 … 1995-12-23. `Sampler` is also the only house whose resource fork is
empty (286 bytes, no `PICT`, no `snd `, no `bnds`). And `Slumberland`'s
`highScores.timeStamps[0]` is 3040890608 → **2000-05-11 11:50:08** — the same
day — while its *house* stamp stays at 1995-08-13, exactly as
`WriteHouse`'s `if (fileDirty)` gate predicts (`GliderPRO/Sources/HouseIO.c:475`:
a play-mode high-score write goes through `WriteHouse(false)` with
`fileDirty == false`, so `timeStamp` is not refreshed). So on 2000-05-11 someone
ran a Glider PRO build, played `Slumberland`, and created `Sampler` from
scratch — and that build's `sizeof(houseType)` was 868.

Which build? The GPL drop itself is from that era: `GliderPRO/Prefix.h` sets
`TARGET_CARBON 1`, `ACCESSOR_CALLS_ARE_FUNCTIONS 1`, `OPAQUE_TOOLBOX_STRUCTS 1`,
`OPAQUE_UPP_TYPES 1`, `forCarbon 1` and `DEBUG 1`. A Carbon build is a PowerPC
build, and CodeWarrior's PowerPC target defaults to natural alignment. The
missing `align=mac68k` pragma in `GliderStructs.h` is exactly the bug that would
produce an 868-byte `houseType` when the project settings changed from 68K to
PowerPC. `Sampler` is that bug's only surviving artefact.

For reference, `Sampler`'s room[1] object table, showing the residue that fills
`objects[4..23]` (`numObjects == 4`, so slots 4..23 are `what == -1` with live
garbage in `data`):

```
rooms[1] @ 1214, numObjects = 4
  slot[ 0] @1274 what=   57 (0x0039) data=00 00 01 f0 00 00 ff ff ff 00
  slot[ 1] @1286 what=   12 (0x000C) data=01 1e 01 7a 00 83 01 01 01 00
  slot[ 2] @1298 what=   51 (0x0033) data=00 61 00 68 00 00 ff ff ff 00
  slot[ 3] @1310 what=   44 (0x002C) data=00 c9 00 36 00 00 00 00 00 01
  slot[ 4] @1322 what=   -1 (0xFFFF) data=01 2b 00 47 01 42 00 62 00 03
  ...
  slot[22] @1538 what=   -1 (0xFFFF) data=01 2f 00 d0 00 20 00 00 01 01
  slot[23] @1550 what=   -1 (0xFFFF) data=01 2f 00 a1 00 20 00 00 01 01
                                                              ^^^^^^^ -> replicated at 1562..1563
```

**7. What the original actually does with the slack: preserves it.** Read path
(`GliderPRO/Sources/HouseIO.c:341-377`): `GetEOF` → `byteCount` (1564) →
`NewHandle(byteCount)` → `SetFPos(…, 0)` → `FSRead(houseRefNum, &byteCount,
*thisHouse)`. The handle is exactly the file length; the slack is in it.
Write path (`GliderPRO/Sources/HouseIO.c:472-503`):
`byteCount = GetHandleSize((Handle)thisHouse)` → `FSWrite` → `SetEOF(houseRefNum,
byteCount)`. The handle length is written back verbatim, including the slack.

The only routine that can shrink the handle is `LopOffExtraRooms`
(`GliderPRO/Sources/HouseLegal.c:740-779`), and it only fires when there are
trailing rooms with `suite == kRoomIsEmpty`:

```c
newSize = sizeof(houseType) + (sizeof(roomType) * (long)r);
HUnlock((Handle)thisHouse);
SetHandleSize((Handle)thisHouse, newSize);
```

Note the shape of that expression: it *recomputes* the size from
`sizeof(houseType)`, so on the one occasion the original does drop the slack, it
also normalises the file to the compiling build's own idea of the header size.
On an 868-build the newly written file keeps 2 bytes of slack; on an 866-build it
loses them.

### Decision

1. **Accept** any house file whose length satisfies
   `len >= 866 + 348 * nRooms`. Do not require equality.
2. **Never reject** a file for having trailing bytes, and never emit an error;
   at most log at debug level.
3. **Never derive `nRooms` from the file length alone.** Read the header field
   and clamp it (see [D5](#d5--the-bound-on-housetypenrooms)).
4. **Preserve** the slack byte-for-byte and write it back at the end of the file,
   so that load→save is byte-exact. Model it as an opaque
   `trailingSlack []byte` captured at load.
5. **Never interpret** the slack, and never let its presence shift `rooms[i]`:
   `rooms[i]` is always at `866 + 348*i`, regardless of slack.
6. **When you create a new house, emit no slack** — a fresh file is exactly
   `866 + 348 * nRooms` bytes. Do not reproduce the 868 bug.
7. **When you append a room, insert it at `866 + 348*nRooms`, i.e. *before* the
   slack**, keeping the slack at the very end of the file. This is what the
   original ends up doing (the `PtrAndHand` copy is overwritten by
   `CopyThisRoomToRoom`), and it is the only rule that keeps existing rooms at
   their existing offsets.
8. **When you delete trailing empty rooms** (`LopOffExtraRooms` equivalent), you
   may drop the slack, matching the original's behaviour on an 866-build. This is
   the one sanctioned place to lose it.

Point 4 is where this document overrules editor.md Q4's "do not write them back".
Rationale: the original preserves them (evidence point 7), so preserving them is
the faithful behaviour and is the only way a Go re-implementation can pass a
`load(f) == save(load(f))` test over the shipped corpus. Dropping them costs
nothing functionally but makes `Sampler` the one file your round-trip test has to
special-case, which is exactly the kind of exception that rots.

### Consequence of deciding otherwise

| If you… | Observable consequence |
|---|---|
| reject files where `len != 866 + 348*nRooms` | `Sampler.binhex` fails to load — the one house shipped as the format's own worked example, and the only one small enough to use in unit tests |
| derive `nRooms = (len − 866) / 348` and *trust* it | works for `Sampler` by luck (698/348 = 2), but see D5: it is also how you silently truncate any file whose header count is right and whose length is off by less than 348 |
| derive `nRooms = (len − 868) / 348` | drops the last room of all 21 exact houses (D2's failure mode) |
| drop the slack on write | `Sampler` round-trips to 1562 bytes; a byte-exactness test over the corpus fails on 1 of 22 with a 2-byte diff, which will be misdiagnosed as a marshalling bug in `roomType` for as long as it takes someone to hexdump the tail |
| append a new room *after* the slack (literal `PtrAndHand` semantics, without the `CopyThisRoomToRoom` fix-up) | the new room's bytes land at `868 + 348n` while every reader looks at `866 + 348n`; the new room reads as the last 2 bytes of the previous room's `objects[23]` followed by 346 bytes of the intended room, i.e. `name[0]` = 0x01, a 1-character room name, and every subsequent field shifted by 2 — `numObjects` reads what should be `openings`, `objects[]` is off by 2, and the last 2 bytes of the room are lost |
| let the slack shift `rooms[i]` (e.g. by computing offsets from the end of the file) | every room in `Sampler` reads 2 bytes late: `rooms[0].name[0]` becomes 0x00 (an empty name) |

### Go

```go
type House struct {
    Header  HouseHeader
    Rooms   []Room
    // Slack is any bytes in the file past 866+348*len(Rooms). Observed only in
    // Houses/Sampler.binhex (2 bytes: 01 01), which was written by a build whose
    // sizeof(houseType) was 868 rather than 866. Opaque; preserved for
    // byte-exact round-tripping. Never interpret it.
    Slack   []byte
}

func ParseHouse(b []byte) (*House, error) {
    if len(b) < SizeHouse {
        return nil, fmt.Errorf("house too short: %d < %d", len(b), SizeHouse)
    }
    h := &House{}
    h.Header.unmarshal(b[:SizeHouse])
    // D5: clamp, never trust, never derive.
    max := (len(b) - OffHouseRooms) / SizeRoom
    n := int(h.Header.NRooms)
    if n < 0 || n > max {
        log.Printf("house nRooms=%d exceeds file capacity %d; clamping", n, max)
        n = max
    }
    h.Rooms = make([]Room, n)
    for i := range h.Rooms {
        h.Rooms[i].unmarshal(b[RoomOffset(i) : RoomOffset(i)+SizeRoom])
    }
    if end := RoomOffset(n); end < len(b) {
        h.Slack = append([]byte(nil), b[end:]...) // D3: preserve
    }
    return h, nil
}

func (h *House) Marshal() []byte {
    out := make([]byte, RoomOffset(len(h.Rooms)), RoomOffset(len(h.Rooms))+len(h.Slack))
    h.Header.marshal(out[:SizeHouse])
    for i := range h.Rooms {
        h.Rooms[i].marshal(out[RoomOffset(i) : RoomOffset(i)+SizeRoom])
    }
    return append(out, h.Slack...) // D3: slack goes last, always
}

// AddRoom appends before the slack, matching the net effect of
// Room.c:198-200 + Room.c:226 (the CopyThisRoomToRoom call).
func (h *House) AddRoom(r Room) { h.Rooms = append(h.Rooms, r) }
```

---

## D4 — `houseType.unusedShort` and the other reserved fields: preserve verbatim, never validate

### The question as previously posed

* **original-houses.md OQ1:** "What exactly does `unusedShort` at `houseType`
  offset 2 hold? Ten houses carry non-zero values ranging from −30082 to 26228
  with no visible pattern; the source never reads or writes it. It may be a
  version-1 field abandoned by `ConvertHouseVer1To2` … but the conversion routine
  does not clear it either."
* **progression.md OQ4:** "**A loader must not validate it.**"
* **scoring.md OQ4:** "0x3333 and 0x6674 look like ASCII/filler; 259 = 0x0103
  looks like a version. **A port must preserve it verbatim.**"
* **constants.md OQ5:** "**A Go port must treat them as reserved and ignore them
  on read.**"

progression.md and scoring.md agree on "preserve"; constants.md says "ignore on
read", which is not the same instruction and does not say what to do on write.
The task was to decide whether byte-exact round-tripping *requires* preserving
it. It does, and the reason is stronger than "it's garbage".

### Evidence

**1. No code anywhere reads or writes it.** Searching all 67 files under
`GliderPRO/Sources/` and `GliderPRO/Headers/` (with `command grep -an` — see
[§0.1](#01-line-number-convention) for why the naive `grep` lies here) finds
`unusedShort` only in these places:

| Site | Which `unusedShort` | What it does |
|---|---|---|
| `GliderPRO/Headers/GliderStructs.h:131` | `gameType.unusedShort` | declaration |
| `GliderPRO/Headers/GliderStructs.h:138` | `savedRoom.unusedShort` | declaration |
| `GliderPRO/Headers/GliderStructs.h:185` | **`houseType.unusedShort`** | declaration — and that is the only mention of it in the whole program |
| `GliderPRO/Sources/SavedGames.c:113` | `savedRoom.unusedShort` | `destRoom->unusedShort = 0;` (inside the commented-out `SaveGame2`) |
| `GliderPRO/Sources/SavedGames.c:273` | `gameType.unusedShort` | `smallGame.unusedShort = 0;` (inside the commented-out `OpenSavedGame`) |
| `GliderPRO/Sources/SavedGames.c:333` | `gameType.unusedShort` | `thisHousePtr->savedGame.unusedShort = 0;` — **live**, in `SaveGame` |

So `houseType.unusedShort` has **zero** readers and **zero** writers.
`gameType.unusedShort` (house offset 856) *is* zeroed, but only when a game is
saved.

**2. It is never initialised, by construction.** `InitializeEmptyHouse`
(`GliderPRO/Sources/House.c:108-161`) allocates with `NewHandle`, not
`NewHandleClear` (`:116`), then assigns exactly these fields:

```
version    = kHouseVersion (0x0200)     House.c:127
firstRoom  = -1                         House.c:128
timeStamp  = 0                          House.c:129
flags      = 0                          House.c:130
initial.h  = 32, initial.v = 32         House.c:131-132
ZeroHighScores()                        House.c:133  <- highScores IS cleared
banner     = localised string 11        House.c:135-136
trailer    = localised string 12        House.c:137-138
hasGame    = false                      House.c:139
nRooms     = 0                          House.c:140
```

Note that `highScores` *is* explicitly cleared, by `ZeroHighScores()` — so it is
not residue and is not on the preserve-blindly list. But `unusedShort`,
`unusedBoolean` and the entire 40-byte `savedGame` block are
**never touched**. Whatever the Memory Manager handed back is what gets written
to disk on the first save. `ConvertHouseVer1To2`
(`GliderPRO/Sources/House.c:746-820`) sets `version = kHouseVersion` at `:814`
and likewise never clears them.

**3. Observed values across the corpus — 10 of 22 non-zero.**

| House | `unusedShort` (hex) | decimal (signed) | decimal (unsigned) |
|---|---|---:|---:|
| Art Museum | 0x8A7E | −30082 | 35454 |
| California or Bust! | 0x3333 | 13107 | 13107 |
| Castle o' the Air | 0x0103 | 259 | 259 |
| Davis Station | 0x00DE | 222 | 222 |
| Fun House | 0x6674 | 26228 | 26228 |
| Grand Prix | 0x003C | 60 | 60 |
| In The Mirror | 0x0093 | 147 | 147 |
| Land of Illusion | 0x0103 | 259 | 259 |
| Metropolis | 0x00C4 | 196 | 196 |
| Titanic | 0x081A | 2074 | 2074 |
| *the other 12* | 0x0000 | 0 | 0 |

There is no pattern, no monotonicity, no correlation with `version` (all 22 are
0x0200), `nRooms`, file size or date. 0x3333 is a classic heap fill; 0x6674 is
ASCII `"ft"`; 0x8A7E is not printable MacRoman text.

**4. `unusedBoolean` (offset 861) behaves identically — 7 of 22 non-zero**, and
with values no `Boolean` should have:

| House | `unusedBoolean` |
|---|---:|
| Art Museum | 255 |
| Castle o' the Air | 30 |
| Davis Station | 37 |
| Grand Prix | 185 |
| Land of Illusion | 30 |
| Metropolis | 14 |
| Sampler | 2 |
| *the other 15* | 0 |

A Mac `Boolean` is `unsigned char` and the Toolbox convention is 0 / 1, so 255,
185, 37, 30, 14 and 2 are all residue. `unusedBoolean` has zero readers and zero
writers in the program.

**5. The `savedGame` block (offsets 820..859) has zero *live* readers, and its
contents prove the residue story.** `hasGame` (offset 860) likewise:

| Site | What it does |
|---|---|
| `GliderPRO/Sources/House.c:139` | `thisHousePtr->hasGame = false;` (in `InitializeEmptyHouse`) — write |
| `GliderPRO/Sources/SavedGames.c:318-337` | writes all 16 `savedGame` fields + `hasGame = true` (in `SaveGame(true)`) — write |
| `GliderPRO/Sources/SavedGames.c:341` | `hasGame = false` (in `SaveGame(false)`) — write |
| `GliderPRO/Sources/Menu.c:727-728` | `hadPoints = thisHousePtr->savedGame.score; hadGliders = thisHousePtr->savedGame.numGliders;` — **read**, but inside `QueryResumeGame`, which is declared at `GliderPRO/Sources/Menu.c:28`, defined at `:710`, and **never called from anywhere** |

`hasGame` is therefore never read at all, and `savedGame` is read only by dead
code. The resume path that would have consumed it
(`NewGame(kResumeGameMode)` → `SetHouseToSavedRoom()` →
`ForceThisRoom(smallGame.roomNumber)`, `GliderPRO/Sources/Play.c:380-383`) reads
the *file-scope* `gameType smallGame` (`GliderPRO/Sources/SavedGames.c:20`),
which is only ever filled by the commented-out `OpenSavedGame`
(`:261-275`) — so `smallGame` is all zeros — and is reachable only via
`case iOpenSavedGame:` → `if (OpenSavedGame())`, which returns `false`
unconditionally (`GliderPRO/Sources/Menu.c:317-325`,
`GliderPRO/Sources/SavedGames.c:169`).

Decoded `savedGame` blocks for all 22 houses:

| House | `hasGame` | `savedGame.version` | `savedGame.timeStamp` decoded | score | numGliders | roomNumber |
|---|---:|---|---|---:|---:|---:|
| Art Museum | 0 | **0xF6F6** | 2035-04-18 18:56:54 | 16774902 | −2314 | 255 |
| CD Demo House | 0 | 0x0100 | 1995-02-14 17:33:05 | 28200 | 4 | 40 |
| California or Bust! | 0 | 0x0100 | 1995-05-14 20:29:08 | 0 | 2 | 0 |
| Castle o' the Air | 0 | 0x0100 | 1994-10-14 14:00:30 | 13200 | 2 | 7 |
| Davis Station | 0 | **0x0000** | 1904-01-01 18:12:15 | 65536 | −21760 | 225 |
| Demo House | 0 | 0x0100 | 1994-10-07 15:09:08 | 3000 | 0 | 15 |
| Empty House | 0 | 0x0100 | 1994-07-28 17:10:48 | 0 | 2 | 0 |
| Fun House | 0 | **0x0000** | 1904-01-01 00:00:01 | −1423057163 | 0 | 0 |
| Grand Prix | 0 | **0x00A9** | 1904-05-08 04:41:03 | 11993488 | 185 | 185 |
| ImagineHouse PRO II | **1** | 0x0100 | 1994-12-05 19:31:09 | 5900 | 5 | 45 |
| In The Mirror | 0 | 0x0100 | 1995-04-29 19:01:38 | 100 | 2 | 1 |
| Land of Illusion | 0 | 0x0100 | 1995-01-31 13:45:59 | 43800 | 9 | 134 |
| Leviathan | 0 | 0x0100 | 1995-04-25 10:40:29 | 55600 | 24 | 449 |
| Metropolis | 0 | 0x0100 | 1995-02-09 20:38:15 | 300 | 2 | 9 |
| Nemo's Market | 0 | 0x0100 | 1995-04-19 07:46:34 | 25600 | 6 | 202 |
| Rainbow's End | 0 | 0x0100 | 1995-02-10 14:06:43 | 46200 | 10 | 112 |
| Sampler | 0 | **0x0001** | 1904-01-01 00:00:00 | 0 | 1 | −1 |
| Slumberland | 0 | 0x0100 | 1995-02-14 17:33:05 | 28200 | 4 | 40 |
| SpacePods | 0 | 0x0100 | 1995-04-19 07:46:34 | 25600 | 6 | 202 |
| Teddy World | 0 | 0x0100 | 1995-05-15 21:55:30 | 2500 | 1 | 75 |
| The Asylum Pro | 0 | 0x0100 | 1994-12-09 21:34:51 | 3900 | 1 | 117 |
| Titanic | **1** | 0x0100 | 1995-04-09 19:21:27 | 4700 | 2 | 104 |

Three things fall out of that table, and all three matter:

**(a) `savedGame.version` is 0x0100 in 17 of 22 houses, and this source defines
`kSavedGameVersion` as 0x0200** (`GliderPRO/Sources/SavedGames.c:14`). So every
one of those 17 blocks was written by an *earlier* build than the one whose source
we have. Not one shipped house carries a 0x0200 saved game. A port that validates
`savedGame.version == 0x0200` before honouring `hasGame` would reject the saved
games in both houses that have one (ImagineHouse PRO II and Titanic, both
0x0100).

**(b) The 5 remaining houses hold pure fill patterns**, and you can read the
allocator's fingerprint in them:

* `Art Museum`: `version` 0xF6F6, `unusedLong` 0xF6F6F6FF, `unusedLong2`
  0xF6F6F9F9, `gliderState` = `numGliders` = −2314 = 0xF6F6, `facing` =
  `showFoil` = 42, `unusedBoolean` = 255. A 0xF6 fill.
* `Sampler`: raw block
  `00 01 00 08 00 00 00 00 00 00 00 00 00 00 00 00 00 47 3e 04 00 00 00 ff 00 00 ff ff ff ff ff ff 00 01 ff ff ff ff cc cc`
  → `facing` = `showFoil` = 0xCC, `bands` = `roomNumber` = `gliderState` =
  `foil` = `unusedShort` = −1. A 0xCC/0xFF fill.
* `Grand Prix`: raw block
  `00 a9 01 91 00 a9 01 df 00 a9 01 e0 00 b7 01 90 00 b7 01 91 00 b7 01 df 00 b7 01 e0 00 b9 01 91 00 b9 01 93 00 b9 01 dd`
  — ten consecutive `(0x00a9…0x00b9, 0x0190…0x01e0)` pairs, i.e. a run of
  `Point`s or `Rect` corners left over from the editor, not a byte fill.
  `numGliders` = `roomNumber` = `unusedBoolean` = 185 (0x00B9) falls out of that
  run rather than from a fill byte.
* `Davis Station`: mostly zeros, `score` = 65536, `timeStamp` = 65535.
* `Fun House`: mostly zeros, `score` = −1423057163 (0xAB2DDEF5) — but
  `unusedLong`, `unusedLong2` and the high byte of `energy` (house offsets
  836..844) hold the bytes `08 44 69 73 6b 43 6f 70 79`, i.e. the Pascal string
  `"\pDiskCopy"` (which is why `energy` reads 30976 = 0x7900). The
  residue in this block is a leftover Apple **Disk Copy** string, which is a
  concrete demonstration that the block is heap garbage and not a partly-valid
  saved game.

**(c) Two pairs of unrelated houses have byte-identical `savedGame` blocks** —
`CD Demo House` == `Slumberland` (both 1995-02-14 17:33:05, score 28200,
4 gliders, room 40) and `Nemo's Market` == `SpacePods` (both 1995-04-19 07:46:34,
score 25600, 6 gliders, room 202). This is scoring.md OQ5, and it resolves here as
a corollary: each pair descends from a **common ancestor file** that was duplicated
and then had its rooms rebuilt, so the stale `savedGame` block came along verbatim.
Only that 40-byte block survives intact — the rest of the 866-byte header diverged
(the `highScores` differ, and so do the file sizes: 72554 vs 134150 and 44018 vs
140762), which is exactly what "no code path ever writes `savedGame` but ordinary
play rewrites the scores" predicts. The corroborating signal is
`Castle o' the Air` and `Land of Illusion`, which are the only two houses sharing
*both* `unusedShort` (0x0103) *and* `unusedBoolean` (30) while having different
`savedGame` blocks: the same duplication, followed by a later `SaveGame(true)` in
the copy that overwrote `savedGame` but could not touch the two reserved fields
because nothing writes them.

**6. Contrast: `roomType.unusedByte` is NOT residue — it is force-zeroed.** This
is the one reserved field a port must normalise, and it is why "treat all
`unused*` fields alike" is wrong:

* `CreateNewRoom` sets `thisRoom->unusedByte = 0;`
  (`GliderPRO/Sources/Room.c:173`).
* `CheckRoomNameLength` sets `(*thisHouse)->rooms[i].unusedByte = 0;` for
  **every** room, in a loop over `nRooms`
  (`GliderPRO/Sources/HouseLegal.c:868`), and it runs from
  `CheckHouseForProblems` at `GliderPRO/Sources/HouseLegal.c:1144` — but note
  that call is **inside** `if (isHouseChecks)` (`:1140`), unlike `CompressHouse`
  (`:1103`) and `LopOffExtraRooms` (`:1105`), which are unconditional. So the
  re-zeroing happens on a full house check, not on literally every save.

Observed: **all 4070 rooms in the corpus have `unusedByte == 0`** (verified by
re-parsing every house). So `roomType.unusedByte` is 0 in every room of every
shipped house, and the original will re-zero it if it is not.

**7. The same problem exists inside every fixed-size Pascal string, on a much
larger scale, and no sibling document mentions it.** `PasStringCopy`
(`GliderPRO/Sources/StringUtils.c:18-25`) is the program's universal string
assignment:

```c
void PasStringCopy (StringPtr p1, StringPtr p2)
{
    register short      stringLength;
    stringLength = *p2++ = *p1++;
    while (--stringLength >= 0)
        *p2++ = *p1++;
}
```

It copies exactly `1 + length` bytes and **never pads the remainder**. So in
every `StrNN` field, the bytes from `1 + s[0]` to `NN` inclusive are whatever was
in that memory before. Measured across the corpus:

| Field | Type | Bytes | Instances with residue past the length byte |
|---|---|---:|---|
| `houseType.banner` | `Str255` | 256 | **22 of 22** houses |
| `houseType.trailer` | `Str255` | 256 | **22 of 22** houses |
| `houseType.highScores.banner` | `Str31` | 32 | **21 of 22** houses |
| `houseType.highScores.names[i]` | `Str15` | 16 | **183 of 220** slots |
| `roomType.name` | `Str27` | 28 | **4008 of 4070** rooms |

Sample, `Art Museum` rooms 0-5 (`name[0]` then the residue after it):

```
room 0  len= 9  72 63 69 73 73 75 73 00 00 00 07 e0 1f f8 3f fc 7f fe   "rcissus"...
room 1  len= 9  52 6f 6f 6d 29 00 00 00 00 00 07 e0 1f f8 3f fc 7f fe   "Room)"...
room 2  len=15  68 00 00 00 07 e0 1f f8 3f fc 7f fe                     "h"...
room 3  len=15  00 00 00 00 07 e0 1f f8 3f fc 7f fe
room 4  len=12  6d 29 00 00 00 00 00 07 e0 1f f8 3f fc 7f fe            "m)"...
room 5  len=14  00 00 00 00 00 07 e0 1f f8 3f fc 7f fe
```

Two distinct residue sources are visible: tails of *longer previous room names*
("rcissus", "Room)", "m)"), and the recurring `07 e0 1f f8 3f fc 7f fe` — an
anti-aliased-disc bitmask row pattern, i.e. leftover graphics data from whatever
occupied the heap block before the house handle did.

The only routine that rewrites these strings in bulk is `WrapBannerAndTrailer`
(`GliderPRO/Sources/HouseLegal.c:604-615`), which calls
`WrapText((*thisHouse)->banner, 40)` and `WrapText((*thisHouse)->trailer, 64)`
(`:611-612`) — it inserts line breaks in place and shortens/lengthens `[0]`, again
without padding. `CheckRoomNameLength` tests `name[0] > 27`
(`GliderPRO/Sources/HouseLegal.c:871`) and clamps `name[0] = 27`
(`:873`) without touching the bytes past it. (No room in the corpus has
`name[0] > 27`, so the clamp never fires on the shipped houses.)

### Decision

**Byte-exact round-tripping requires preserving all of the following verbatim.
Model each as opaque bytes, not as a typed field. Never validate, never zero,
never interpret, never surface in the UI:**

| Field | Offset | Size | Reason |
|---|---:|---:|---|
| `houseType.unusedShort` | 2 | 2 | zero readers, zero writers, non-zero in 10 of 22 |
| `houseType.unusedBoolean` | 861 | 1 | zero readers, zero writers, non-zero in 7 of 22 |
| `houseType.savedGame` | 820 | 40 | written only by `SaveGame(true)`; read only by uncalled `QueryResumeGame`; residue or stale-0x0100 in every shipped house |
| `houseType.hasGame` | 860 | 1 | written but never read; `1` in exactly 2 of 22 |
| `savedRoom.unusedShort` / `.unusedByte` | 0 / 2 | 2 / 1 | in the dead `'gliG'` format; zeroed by dead code |
| `gameType.unusedLong`, `.unusedLong2`, `.unusedShort` | +16, +20, +36 | 4, 4, 2 | zeroed by the live `SaveGame` (`SavedGames.c:325`, `:326`, `:333`) — so preserve on load, but write 0 when *you* save a game |
| `demoType.padding` | +5 | 1 | 109 distinct values across 1117 records; see D7 |
| every `StrNN` tail: bytes `1+s[0]` .. `NN−1` of `houseType.banner`, `.trailer`, `highScores.banner`, `highScores.names[0..9]`, `roomType.name` | — | varies | `PasStringCopy` never pads (`StringUtils.c:18-25`); residue present in 22/22 banners, 22/22 trailers, 183/220 name slots and **4008/4070 room names** |

**Exception — normalise these:**

| Field | Offset | Rule | Citation |
|---|---:|---|---|
| `roomType.unusedByte` | +32 | write **0** for every room whenever you run the house checker or create a room | `GliderPRO/Sources/HouseLegal.c:868`, `GliderPRO/Sources/Room.c:173` |
| `houseType.timeStamp` bit 31 | 4 | always write 0 (`timeStamp &= 0x7FFFFFFF`) | `GliderPRO/Sources/HouseIO.c:478` |
| `houseType.timeStamp` bit 0 | 4 | 0 == unlocked, 1 == locked; set from the lock state on every dirty write | `GliderPRO/Sources/HouseIO.c:483-486`, read at `:402` |

**Additionally: do not gate anything on `hasGame`, and do not validate
`savedGame.version`.** If you implement a resume feature, use `hasGame` as a hint
only, and accept `savedGame.version` ∈ {0x0100, 0x0200} — 17 of 22 shipped houses
carry 0x0100 and both houses that actually have a saved game are among them.

### Consequence of deciding otherwise

| If you… | Observable consequence |
|---|---|
| zero `unusedShort` on write | 10 of 22 houses change on save with no user-visible edit. A `load → save → diff` regression suite reports 10 failures; anyone bisecting will look at `banner`/`trailer` marshalling first because offset 2 is the last place you'd suspect |
| zero `unusedBoolean` on write | 7 more of the same |
| zero the whole `savedGame` block when `hasGame == 0` | 20 of 22 houses change on save, by up to 40 bytes each. Worse, `Titanic` and `ImagineHouse PRO II` have `hasGame == 1`, so a naive "clear it if unused" rule silently destroys the only two real saved games in the corpus if the flag is misread |
| *validate* `unusedShort == 0` and reject | 10 of 22 houses fail to load, including `Titanic`, `Metropolis`, `Grand Prix`, `Art Museum`, `Land of Illusion` and `Castle o' the Air` — six of the largest hand-built houses |
| *interpret* `unusedShort` as a minor version (0x0103 "looks like a version") | `Castle o' the Air` and `Land of Illusion` claim v1.3, `California or Bust!` claims v0x3333, `Art Museum` claims v0x8A7E; any version-dependent branch fires at random across the corpus |
| *interpret* `unusedBoolean` as a flag | 7 houses get the flag set, including `Sampler` (value 2) — and since C's `if (b)` is true for any non-zero, so is Go's `b != 0`; a "two-player only" or "read-only" style flag would engage on 7 houses at random |
| require `savedGame.version == kSavedGameVersion` (0x0200) | no shipped house has a resumable game; the feature appears to work and never does |
| *preserve* `roomType.unusedByte` instead of zeroing it | harmless on the shipped corpus (all zero), but your writer and the original writer disagree the moment a third-party house has residue there, and the original will zero it on its next edit-mode save while yours does not — a spurious diff in the opposite direction |
| model Pascal strings as Go `string` and re-serialise as `len` + bytes + zero padding | **the largest failure mode of all.** 4008 of 4070 room names, all 22 banners, all 22 trailers and 183 of 220 high-score name slots change on save. Every single house in the corpus fails a byte-exactness test, by hundreds of bytes each, and the diffs are scattered through `rooms[]` so they look like a `roomType` layout bug |

### Go

```go
// HouseHeader mirrors houseType's first 866 bytes. The Reserved* fields hold
// uninitialised heap residue from InitializeEmptyHouse's NewHandle
// (House.c:116 — note: NewHandle, not NewHandleClear). They have no readers and
// no writers anywhere in the original. Preserve them; do not validate them.
type HouseHeader struct {
    Version   int16     // +0   always 0x0200 in the corpus
    Reserved2 [2]byte   // +2   houseType.unusedShort: 0x8A7E, 0x3333, 0x0103, ...
    TimeStamp uint32    // +4   bit 31 always 0 on disk; bit 0 = locked
    Flags     uint32    // +8   bit0 ward, bit1 phone, bit2 !bannerStarCount
    Initial   Point     // +12  {v, h}
    Banner    [256]byte // +16  Str255
    Trailer   [256]byte // +272 Str255
    HighScores ScoresRec // +528
    SavedGame [40]byte  // +820 opaque: gameType, version 0x0100 in 17/22 houses
    HasGame   byte      // +860 written, never read
    Reserved861 byte    // +861 houseType.unusedBoolean: 255, 185, 37, 30, 14, 2
    FirstRoom int16     // +862
    NRooms    int16     // +864
}
```

Keeping `SavedGame` as `[40]byte` rather than a struct is deliberate: it makes it
impossible to "helpfully" normalise a field inside it, and it documents at the
type level that the block is not trustworthy. Decode it lazily, on demand, only
if you build a resume feature.

Pascal strings must be kept as fixed-size byte arrays for the same reason, with
accessors rather than conversion at load time:

```go
// PStr is a Mac Pascal string in a fixed-size field: s[0] is the length, s[1:1+s[0]]
// is MacRoman text, and s[1+s[0]:] is UNINITIALISED RESIDUE that PasStringCopy
// (StringUtils.c:18-25) never pads. Keep the whole array; never rebuild it.
type PStr[N int] [N]byte   // conceptually; in practice [28]byte, [32]byte, [256]byte

func (s *[28]byte) Text() string { return macRoman(s[1 : 1+int(s[0])]) }

// SetText writes length+bytes ONLY, exactly as PasStringCopy does, leaving the
// tail alone. Do not zero the tail.
func (s *[28]byte) SetText(v string) {
    b := toMacRoman(v)
    if len(b) > 27 { b = b[:27] }   // CheckRoomNameLength clamps name[0] to 27
    s[0] = byte(len(b))
    copy(s[1:], b)
}
```

---

## D5 — The bound on `houseType.nRooms`

### The question as previously posed

* **constants.md OQ14:** "Is there any bound on `houseType.nRooms` at load time?
  No `kMaxRooms` constant exists anywhere (`grep` returns nothing).
  `HouseLegal.c` bounds *coordinates* to 128 × 64 and uses an 8192-cell duplicate
  bitmap, but nothing bounds the room count itself; `HouseLegal.c:630-631` derives
  it from the handle size. A malformed house with a huge `nRooms` and a small data
  fork will read past the handle."
* **house-format.md** and **editor.md** repeat the observation without a rule.

The task asked for an explicit bound. There isn't one in the source, so one has
to be derived and stated normatively. Three separate bounds apply, and they
disagree; the loader needs all three.

### Evidence

**1. There is no `kMaxRooms`.** Confirmed with `command grep -an 'kMaxRooms'`
across `GliderPRO/Headers/*.h` and `GliderPRO/Sources/*.c`: zero matches. For
comparison, the constants that *do* exist and bound related things
(`GliderPRO/Headers/GliderDefines.h`):

| Constant | Value | Line | Bounds |
|---|---:|---:|---|
| `kMaxRoomObs` | 24 | 250 | objects per room |
| `kMaxMasterObjects` | 216 | 266 | `kMaxRoomObs * 9`, the 9-room neighbourhood |
| `kMaxScores` | 10 | 249 | high-score table entries |
| `kMaxSparkles` | 3 | 251 | live sparkle animations |
| `kMaxHotSpots` | 56 | 259 | live hot spots |
| `kNumTiles` | 8 | 496 | background tiles per room |
| `kRoomIsEmpty` | −1 | 525 | `roomType.suite` sentinel for a placeholder room |
| *(none)* | — | — | **rooms per house** |

**2. The structural bound: `nRooms` is a `short`.** The field is
`short nRooms;` at `GliderPRO/Headers/GliderStructs.h:196`, and
`ValidateNumberOfRooms` narrows a `long` count back into it
(`GliderPRO/Sources/HouseLegal.c:629-637`; the division is at `:630-631`, the
narrowing store at `:634`):

```c
countedRooms = (GetHandleSize((Handle)thisHouse) -
        sizeof(houseType)) / sizeof(roomType);
...
(*thisHouse)->nRooms = (short)countedRooms;
```

So the representable range is −32768 .. 32767, and the largest self-consistent
file is `866 + 348 × 32767 = 11,403,782` bytes (≈10.9 MiB). A file of
`866 + 348 × 32768 = 11,404,130` bytes makes `countedRooms` 32768, which narrows
to **−32768**; `ReadHouse`'s `if ((numberRooms < 1) || (byteCount == 0L))` check
(`GliderPRO/Sources/HouseIO.c:385`) would catch the negative — but only on the
*read* path, and `ValidateNumberOfRooms` runs later and can reintroduce it.

**3. The semantic bound: 8192 addressable rooms.** A room's position in the house
is `(floor, suite)`. `ValidateRoomNumbers`
(`GliderPRO/Sources/HouseLegal.c:785-828`) clamps them:

```c
numRooms = (*thisHouse)->nRooms;                    /* :794 */
if (numRooms < 0)                                   /* :795 */
{
    (*thisHouse)->nRooms = 0;                       /* :797 */
    numRooms = 0;
}
for (i = 0; i < numRooms; i++)
    if ((*thisHouse)->rooms[i].suite != kRoomIsEmpty)          /* :802 */
    {
        if (((*thisHouse)->rooms[i].floor > 56) ||             /* :804 */
                ((*thisHouse)->rooms[i].floor < -7))           /* :805 */
        {  rooms[i].suite = kRoomIsEmpty; ... houseErrors++;  }
        if (((*thisHouse)->rooms[i].suite >= 128) ||           /* :814 */
                ((*thisHouse)->rooms[i].suite < 0))            /* :815 */
        {  rooms[i].suite = kRoomIsEmpty; ... houseErrors++;  }
    }
```

Note the `numRooms < 0` guard at `:795-799`: `ValidateRoomNumbers` is the only
place that repairs a negative room count, and it runs **after** `LopOffExtraRooms`
(`:1110` vs `:1105`), which is what can create one (see 7(b)). Also note that an
out-of-range `floor` or `suite` is not clamped into range — the whole room is
**deleted** (`suite = kRoomIsEmpty`).

So `floor` ∈ [−7, 56] — 64 distinct values — and `suite` ∈ [0, 127] — 128
distinct values. 64 × 128 = **8192**. `CheckDuplicateFloorSuite` encodes exactly
that and even names the number (`GliderPRO/Sources/HouseLegal.c:648, 665-666`):

```c
#define     kRoomsTimesSuites   8192
...
bitPlace = (((*thisHouse)->rooms[i].floor + 7) * 128) +
        (*thisHouse)->rooms[i].suite;
if ((bitPlace < 0) || (bitPlace >= 8192))
    DebugStr("\pBlew array");
if (pidgeonHoles[bitPlace] != 0)
{
    houseErrors++;
    (*thisHouse)->rooms[i].suite = kRoomIsEmpty;   /* duplicate -> deleted */
}
else
    pidgeonHoles[bitPlace]++;
```

Because duplicates are *deleted* (turned into placeholders), a legal house can
contain at most 8192 rooms that a player can reach. Beyond that every extra room
is forced to `suite == kRoomIsEmpty`.

**4. But placeholders are unbounded.** `rooms[i].suite == kRoomIsEmpty` (−1)
marks a deleted room. `CheckDuplicateFloorSuite` skips them (`:663`),
`ValidateRoomNumbers` skips them, and `RealRoomNumberCount`
(`GliderPRO/Sources/House.c:170-189`) counts only non-empty ones. Placeholders
are removed by `CompressHouse` (`GliderPRO/Sources/HouseLegal.c:688-734`) and
trailing ones by `LopOffExtraRooms` (`:740-779`), both of which run
**unconditionally** inside `CheckHouseForProblems`
(`GliderPRO/Sources/HouseLegal.c:1103` and `:1105` — outside every
`if (isHouseChecks)` guard). So in a *saved-from-the-editor* house the count is
compressed; in an arbitrary file it need not be. The theoretical maximum is
therefore the `short` bound (32767), not 8192.

**5. Observed:** the 22 shipped houses have `nRooms` from **2** (Sampler) to
**531** (Teddy World), 4070 rooms in total. Nothing is near either bound.

**6. The load path does not check anything.** `ReadHouse` reads `nRooms` from the
file and uses it directly (`GliderPRO/Sources/HouseIO.c:380`) with only the
`< 1` test. `numberRooms` is then the global room count and every consumer
indexes `rooms[i]` for `i < numberRooms` — `RealRoomNumberCount`
(`House.c:170-189`), `CheckDuplicateFloorSuite`, `ValidateRoomNumbers`,
`CheckRoomNameLength`, `CompressHouse`, `LopOffExtraRooms`, the map window, the
link enumerator. A file claiming `nRooms = 30000` with a 2 KiB data fork causes
all of them to walk ~10 MiB past a 2 KiB handle. On a classic Mac that corrupts
the heap; in Go it panics.

**7. Two adjacent robustness bugs a port must not reproduce.**

*(a) `CheckDuplicateFloorSuite` writes out of bounds before anything clamps
`floor`/`suite`.* The `DebugStr("\pBlew array")` at
`GliderPRO/Sources/HouseLegal.c:667-668` is a no-op in a release build and is
**not** followed by a `continue`, so `pidgeonHoles[bitPlace]` is indexed anyway.
`bitPlace` is a `short` computed as `(floor + 7) * 128 + suite`; with
`floor = 32767` it overflows, and with e.g. `floor = 200, suite = 0` it is 26496 —
an 18 KiB write past a 8192-byte `NewPtrClear` block. And the call order in
`CheckHouseForProblems` puts `CheckDuplicateFloorSuite` (`:1089`) **before**
`ValidateRoomNumbers` (`:1110`), so the clamping happens after the overflow.

*(b) `LopOffExtraRooms` underflows when `nRooms == 0`.*
`GliderPRO/Sources/HouseLegal.c:750-760`:

```c
count = 0;
r = (*thisHouse)->nRooms;       // begin at last room
do
{
    r--;                        // look for trailing empties
    if ((*thisHouse)->rooms[r].suite == kRoomIsEmpty)
        count++;
    else
        r = 0;
}
while (r > 0);
```

With `nRooms == 0` the first iteration reads `rooms[-1].suite`, i.e. house offset
`866 − 348 + 54 = 572`, which is inside `highScores.names[0]` (bytes 12-13 of the
first high-score name). If those two bytes happen to be `0xFFFF`, `count`
becomes 1, the loop exits, and then
`r = 0 − 1 = −1; newSize = 866 + 348 × (−1) = 518; SetHandleSize(…, 518)` —
truncating the house header mid-`highScores` — and `nRooms` becomes −1. `nRooms`
is 0 exactly between `InitializeEmptyHouse` (`House.c:140`) and the first
`CreateNewRoom`, so "New House, then Save" is the trigger.

`ValidateRoomNumbers`'s `if (numRooms < 0) nRooms = 0;` (`:795-799`) would repair
the −1 — but it runs at `:1110`, five lines *after* `LopOffExtraRooms` at `:1105`,
so by then the handle has already been shrunk to 518 bytes. Offset 572 is
`highScores.names[0][12..13]`: `scoresType` starts at house offset 528, its
`names[]` array at 528 + 32 = 560, and each `Str15` is 16 bytes.

### Corollary: `floor + 7` is exactly right, not an off-by-one

**constants.md OQ1** asks whether the `+ 7` in
`bitPlace = ((floor + 7) * 128) + suite` is an off-by-one. It is not. The
arithmetic is exact and the array is exactly full:

| Quantity | Expression | Value |
|---|---|---:|
| lowest legal `floor` | `ValidateRoomNumbers`, `HouseLegal.c:804-805` | −7 |
| highest legal `floor` | same (`HouseLegal.c:804`) | 56 |
| distinct floors | `56 − (−7) + 1` | 64 |
| lowest legal `suite` | `HouseLegal.c:814` (`suite < 0` rejected) | 0 |
| highest legal `suite` | `HouseLegal.c:814` (`suite >= 128` rejected) | 127 |
| distinct suites | `127 − 0 + 1` | 128 |
| minimum `bitPlace` | `(−7 + 7) * 128 + 0` | **0** |
| maximum `bitPlace` | `(56 + 7) * 128 + 127` = `8064 + 127` | **8191** |
| array size | `kRoomsTimesSuites`, `HouseLegal.c:648` | **8192** |

`+7` is the bias that maps the 8 underground floors (`kNumUndergroundFloors` = 8,
i.e. floors −7..0) onto indices 0..1023, and `* 128` is the suite stride. The
`(bitPlace < 0) || (bitPlace >= 8192)` test at `HouseLegal.c:667` is dead for any
house that has already passed `ValidateRoomNumbers` — the bug is purely that
`CheckDuplicateFloorSuite` runs *before* it (`:1089` vs `:1110`).

Note the different bias used by the *link* encoding: `MergeFloorSuite` is
`(suite * 100) + floor` (`GliderPRO/Sources/Link.c:34-37`) and the on-disk
`where` field is `suite * 100 + (floor + kNumUndergroundFloors)` — bias **8**, not
7, and stride 100, not 128. Two different packings of the same pair coexist in the
program; do not unify them.

### Decision

**There is no `kMaxRooms` in the original. Adopt this three-part bound.**

1. **Load-time hard bound (the normative rule):**

   ```
   capacity = (len(dataFork) − 866) / 348          // integer division, floor
   nRooms   = min(int(headerNRooms), capacity)
   reject the file if headerNRooms < 1 or len(dataFork) < 866
   ```

   Take `nRooms` from the **header** and clamp it down to `capacity`; never take
   it from `capacity` alone ([D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length)),
   and never trust it up ([D5](#d5--the-bound-on-housetypenrooms), this rule). Log
   at warning level when clamping fires. This mirrors the intent of
   `ValidateNumberOfRooms` while being safe on the read path, where the original
   is not.

2. **Representational bound:** `1 <= nRooms <= 32767`. `nRooms` is a signed
   `short`; anything outside is a corrupt file. Do not allow your writer to
   produce a house whose room count would not narrow to a positive `short`.

3. **Semantic bound for the editor:** at most **8192** rooms may have a
   `(floor, suite)` in `floor` ∈ [−7, 56] × `suite` ∈ [0, 127]. Refuse to create
   the 8193rd addressable room. Placeholders (`suite == −1`) are exempt but
   should be compressed away on save, as `CompressHouse` does.

**Also normatively:**

* Validate `floor` and `suite` **before** using them as an index, unlike
  `CheckDuplicateFloorSuite`. Use a `map[int32]int` or a bounds-checked slice; in
  Go a slice index panics rather than corrupting, which turns a silent heap
  smash into a crash — still not acceptable in a loader.
* Bound the `LopOffExtraRooms` loop: `for r := len(rooms) - 1; r >= 0 && rooms[r].Suite == kRoomIsEmpty; r--`.
* Recommended practical cap for a *new* house created by your editor: 8192.
  Recommended sanity cap for logging purposes: anything over 1000 rooms (nearly
  twice the corpus maximum of 531) is worth a note.

### Consequence of deciding otherwise

| If you… | Observable consequence |
|---|---|
| trust `headerNRooms` without clamping | a 1 KiB file claiming `nRooms = 30000` panics with `index out of range` in the room loop; a hostile or truncated house crashes the game rather than reporting a bad file |
| use `capacity` instead of the header value | you inherit the D2/D3 truncation family: with the wrong header size you drop the last room of 21 houses; and for any file with 1..347 bytes of trailing slack you silently *add* a room made of slack whose `suite` is arbitrary |
| omit the `>= 1` check | `nRooms = 0` makes the room slice empty, and every consumer that indexes `rooms[…]` without its own guard panics — most immediately `LopOffExtraRooms`, which reads `rooms[-1]` (see 7(b)). Note that `GetFirstRoomNumber` itself is *not* the failure site: at `House.c:203-207` it returns **−1** and sets `noRoomAtAll = true` when `nRooms <= 0`; the clamp-to-0 at `:211-212` only applies when `nRooms > 0` and `firstRoom` is out of range. The original's own guard is `ReadHouse`'s `numberRooms < 1` at `HouseIO.c:385`, which yields `kYellowNoRooms` alert 0 and refuses the file |
| enforce 8192 as a *load-time* limit | rejects nothing in the shipped corpus, but rejects legal third-party houses that carry uncompressed placeholders past 8192; the bound belongs in the editor, not the loader |
| index a duplicate-detection array by `(floor+7)*128+suite` without validating first | reproduces `CheckDuplicateFloorSuite`'s heap smash as a Go panic on any house with `floor > 56`, `floor < −7` or `suite > 127` — and the original's *own* checker calls this before its own clamping pass, so such houses can exist |
| copy `LopOffExtraRooms` literally | "New House" → "Save" reads `rooms[-1]` and can truncate the file to 518 bytes with `nRooms = −1` |

### Go

```go
const (
    kRoomIsEmpty   = -1
    minFloor       = -7   // HouseLegal.c:805
    maxFloor       = 56   // HouseLegal.c:804
    maxSuite       = 127  // HouseLegal.c:814  (suite < 128)
    addressableMax = (maxFloor - minFloor + 1) * (maxSuite + 1) // 64*128 = 8192
                                                               // == kRoomsTimesSuites
    maxRoomsRepresentable = 32767 // nRooms is a short (GliderStructs.h:196)
)

func roomCount(headerNRooms int16, fileLen int) (int, error) {
    if fileLen < SizeHouse {
        return 0, fmt.Errorf("house shorter than the %d-byte header", SizeHouse)
    }
    if headerNRooms < 1 {
        // HouseIO.c:385 -> YellowAlert(kYellowNoRooms, 0)
        return 0, fmt.Errorf("house declares %d rooms", headerNRooms)
    }
    capacity := (fileLen - OffHouseRooms) / SizeRoom
    n := int(headerNRooms)
    if n > capacity {
        log.Printf("house declares %d rooms but holds at most %d; clamping", n, capacity)
        n = capacity
    }
    return n, nil
}

func floorSuiteKey(floor, suite int16) (int, bool) {
    if floor < minFloor || floor > maxFloor || suite < 0 || suite > maxSuite {
        return 0, false // validate BEFORE indexing; HouseLegal.c:667 does not
    }
    return (int(floor)-minFloor)*(maxSuite+1) + int(suite), true
}
```

---

## D6 — The retail build configuration and the version identity

### The question as previously posed

* **architecture.md OQ1:** "`GliderDefines.h:16` sets `BUILD_ARCADE_VERSION 1`. Is
  that the retail configuration or a kiosk variant? The arcade branches change
  input handling and the scoreboard."
* **progression.md OQ2:** "Does the arcade build change progression (no save, no
  high scores)?"
* **constants.md OQ12:** "Should a port define `BUILD_ARCADE_VERSION` as 1,
  matching the source, or 0?"
* **resource-fork.md OQ6 / scoring.md OQ12:** "`Main.c:3` says `Glider PRO 1.0.4`
  but both `'vers'` resources say 1.1.2. Which is the version?"

These are two questions and they have to be answered together, because the answer
to both is "the released source tree is not the retail build."

### Evidence: the eight `BUILD_ARCADE_VERSION` sites

`command grep -an 'BUILD_ARCADE_VERSION' GliderPRO/Sources/*.c GliderPRO/Headers/*.h`
returns exactly nine lines — the definition plus eight conditionals:

| Site | Sense | `#if` branch (arcade) | `#else` branch (normal) |
|---|---|---|---|
| `GliderPRO/Headers/GliderDefines.h:16` | `#define BUILD_ARCADE_VERSION 1` | — | — |
| `GliderPRO/Sources/Events.c:191` | `#if` | four arrow keys become menu commands (`DoOptionsMenu(iHighScores)`, `DoOptionsMenu(iHelp)`, `DoGameMenu(iNewGame)` ×2) | four arrow keys do editor work (`SelectNeighborRoom` / `MoveObject`), gated on `houseUnlocked` |
| `GliderPRO/Sources/Input.c:192` | `#if` | **any** of `leftKey`/`rightKey`/`battKey`/`bandKey` sets `playing = false; paused = false` — i.e. touch anything to end the attract-mode demo | `if (BitTst(&theKeys, kCommandKeyMap)) DoCommandKey();` — Cmd-Q quits, Cmd-S saves, during the demo |
| `GliderPRO/Sources/Main.c:347` | `#if` | `// HideMenuBarOld();` (commented) | *(no `#else`)* |
| `GliderPRO/Sources/Main.c:366` | `#if` | `ShowMenuBarOld();` — but the whole block is inside `/* … */` from `:365` to `:369` | *(no `#else`)* |
| `GliderPRO/Sources/Play.c:138` | `#if !` | *(no `#else`)* | `// HideMenuBarOld(); // TEMP` (commented) |
| `GliderPRO/Sources/Play.c:512` | `#if` | at `countDown <= 0`: paint `boardSrcRect` black, `CopyBits` it over the scoreboard, then `DrawPicture(GetPicture(kScoreboardPictID))` with `hOffset = (RectWide(&boardSrcRect) - kMaxViewWidth)/2` if `boardSrcRect.right >= 640` else `-576` | `// ShowMenuBarOld(); // TEMP` (commented) |
| `GliderPRO/Sources/Play.c:556` | `#if` | the same blackout-and-redraw, at the end of `PlayGame` | `// ShowMenuBarOld(); // TEMP` (commented) |
| `GliderPRO/Sources/Play.c:806` | `#if !` | *(no `#else`)* | `// HideMenuBarOld(); // TEMP` (commented) |

Read as a pair, the menu-bar sites are coherent and tell you which mode is which:

* **Arcade** hides the menu bar **once, at launch** (`Main.c:347`) and restores it
  **once, at quit** (`Main.c:366`) — a kiosk that never shows a menu bar.
* **Normal** hides the menu bar **per game** (`Play.c:138`, `RestoreEntireGameScreen`
  at `:806`) and shows it again **at game over** (`Play.c:542-544`, `:594-598`) — a
  desktop application that has menus between games.

All four `HideMenuBarOld`/`ShowMenuBarOld` calls are commented out because
`HideMenuBar` is not a Carbon API; on this tree the menu bar is never hidden in
either configuration. That is a Carbon-port artefact, not a design decision.

### Evidence: the arcade build removes features the shipped game had

1. **The editor loses its arrow keys.** `Events.c:191-208` is inside
   `HandleKeyEvent`'s ASCII switch. `HandleKeyEvent` (defined at `Events.c:162`)
   has exactly **one** call site — the main event loop at `Events.c:517` — so it
   runs in every mode reached through that loop, including `kEditMode`; during
   `PlayGame` events go to `HandlePlayEvent` (`Play.c:387`, called from
   `Play.c:441`) instead. Editing is therefore fully affected. Under arcade,
   Left/Right/Up/Down become
   Options▸High Scores / Options▸Help / Game▸New / Game▸New. The editor's
   documented room navigation and single-pixel object nudge
   (`SelectNeighborRoom(kRoomToLeft)`, `MoveObject(kBumpLeft, shiftDown)`) become
   unreachable from the keyboard. Glider PRO shipped *with* the house editor as
   its headline feature; a retail build cannot have had a keyboard-crippled
   editor.
2. **Cmd-S / Cmd-Q stop working during a demo, and the demo becomes
   interruptible by gameplay keys.** `Input.c:194-201` is in `GetDemoInput`, so it
   only affects demo playback — precisely the "press anything to start" behaviour
   of an arcade attract loop.
3. **The scoreboard is blacked out and repainted from `PICT`** at game over
   (`Play.c:512-541`) and at the end of every game (`Play.c:556-592`), with an
   `hOffset` of `-576` when the board is narrower than 640 — cabinet-specific
   screen-geometry code with no analogue in the normal build.
4. **Nothing in the arcade branches touches saving, scoring or progression.**
   `TestHighScore()` is still called from `DoDiedGameOver`
   (`GliderPRO/Sources/GameOver.c:503`) unconditionally on the non-demo path, and
   `SaveGame`/`WriteHouse` are untouched. **progression.md OQ2 is answered "no":
   the arcade flag changes input routing and two repaint calls, nothing else.**

### Evidence: this tree cannot even be built the way retail was

`COMPILENOCP` ("no copy protection") is defined at
`GliderPRO/Headers/GliderDefines.h:14`, and the tree **does not compile without
it**:

| Symbol used | Where | Why it fails when `COMPILENOCP` is undefined |
|---|---|---|
| `encryptedNumber` | `GliderPRO/Sources/Main.c:75`, `:231`, `:313` | its `extern` is commented out at `Main.c:32` (`//extern long encryptedNumber;`); defined in `GliderPRO/Sources/Validate.c:37` but not declared here |
| `didValidation` | `Main.c:310`, `:314` | `extern` commented out at `Main.c:43`; defined `Validate.c:39` |
| `thePrefs.encrypted` | `Main.c:75`, `:231` | the field is commented out of `prefsInfo` at `GliderPRO/Headers/Externs.h:240` (`// long encrypted, fakeLong;`) |
| `thePrefs.fakeLong` | `Main.c:232` | same line |

`GliderPRO/Sources/Validate.c` is still in the tree but is **gutted**: the whole
file is wrapped in `#ifndef COMPILEDEMO` (`:13`), everything from `GetSystemVolume`
to `GetMasterDisk` is inside one `/* … */` block (`:41` to `:365`), and
`ValidInstallation` (`:370`) is a stub whose first statement is `return true;`
(`:374`) with its real body commented out at `:375-394`. `VolumeMatchesPrefs` is
*declared* at `:28` and *defined* at `:96` — the `:380` occurrence is a call inside
that commented-out body. So even a build without `COMPILENOCP` would pass
validation unconditionally; what actually breaks such a build is that the globals
it needs are no longer declared where `Main.c` uses them. `Main.c:304-316` shows
the three-way choice:

```c
#if defined COMPILEDEMO
	copyGood = true;
#elif defined COMPILENOCP
//	didValidation = false;
	copyGood = true;
#else
	didValidation = false;
	copyGood = ValidInstallation(true);
	if (!copyGood)
		encryptedNumber = 0L;
	else if (didValidation)
		WriteOutPrefs();
#endif
```

So **retail had copy protection and this source does not.** The two `long`s were
deleted from `prefsInfo`, which changes the preferences file layout by 8 bytes:

| Build | `sizeof(prefsInfo)` | offset of `prefVersion` |
|---|---:|---:|
| retail (with `long encrypted, fakeLong;`) | **234** | 172 |
| this source (`COMPILENOCP`) | **226** | 164 |

(Compiled under `#pragma pack(2)`, `Str32` = 34 bytes, `Str31` = 32, `Str15` = 16;
verified with `gcc`. `prefsInfo` is wrapped in `#pragma options align=mac68k` at
`Externs.h:231` / `align=reset` at `:269` — the only explicit align pragma in the
whole tree, and the reason D2's alignment argument had to be made from the file
sizes instead.)

### Evidence: the version identity

| Source of truth | Says | Citation |
|---|---|---|
| top-of-file comment in `Main.c` | `// Glider PRO 1.0.4` | `GliderPRO/Sources/Main.c:3` |
| `'vers' (1)` — the Finder "Get Info" version | BCD `01 12` → **1.1.2**, stage `0x80` (= final/release), region `0x0000`; short string `"1.1.2"` (5 bytes); long string `"Glider PRO\xAA 1.1.2\r\xA9 1994-95 Casady & Greene, Inc."` (0x31 = 49 bytes) | `GliderPRO/Glider PRO.r:6135-6140` |
| `'vers' (2)` — the "part of package" version | same BCD `01 12 80 00 0000`; short `"1.1.2"`; long `"\xA9 1994-95 Casady & Greene, Inc."` (0x1F = 31 bytes) | `GliderPRO/Glider PRO.r:6142-6146` |
| the About box at runtime | reads `'vers'` 1 and displays the **long** string, by skipping past `shortVersion`: `messagePtr = (StringPtr)(((UInt32)&(**version).shortVersion[1]) + ((**version).shortVersion[0]))` | `GliderPRO/Sources/About.c:55-61` |
| `command grep -c '1\.0\.4' 'GliderPRO/Glider PRO.r'` | **0** | — |
| `command grep -c '1\.1\.2' 'GliderPRO/Glider PRO.r'` | 3 (two lines in `'vers'` 1, one in `'vers'` 2) | — |
| the About dialog's own static text | `© 1994-2000 Casady & Greene, Inc.` — length `0x21` = 33 bytes, type byte `0x88` = disabled `statText`, rect (114, 11, 130, 304) | `GliderPRO/Glider PRO.r:5179-5181` (`data 'DITL' (150, "About")`, which opens at `:5172`) |
| `GliderPRO/Prefix.h` | `TARGET_CARBON 1`, `ACCESSOR_CALLS_ARE_FUNCTIONS 1`, `OPAQUE_TOOLBOX_STRUCTS 1`, `OPAQUE_UPP_TYPES 1`, `forCarbon 1`, `DEBUG 1` | — |

**The `© 1994-2000` in `DITL` 150 dates the drop and closes resource-fork.md OQ5.**
`DITL` 150 declares 9 items (`0x0008` = count − 1). Item 1 is a `picItem`
referencing `PICT` 150; **item 2** is a `statText` with rect (155, 56, 171, 184) and
a **zero-length** string, because `About.c:61` fills it at runtime with `'vers'` 1's
long string (`#define kTextItemVers 2` at `About.c:35`). So the running About box displays two copyrights at once — `© 1994-95`
injected from `'vers'`, and `© 1994-2000` baked into the `DITL` — which is exactly
what you get when someone reopens a 1995 project in 2000 to Carbonise it, updates
the dialog text, and never touches the `'vers'` resource. Combined with
`Prefix.h`'s `TARGET_CARBON 1` and `DEBUG 1`, this drop is a **2000-or-later
work-in-progress**, not the artefact that shipped on the retail disk. That is the
single fact that makes every other question in this section answerable.

None of the four on-disk version numbers is the application version:
`kHouseVersion 0x0200` (`GliderDefines.h:517`), `kNewHouseVersion 0x0300` (`:518`),
`kSavedGameVersion 0x0200` (`GliderPRO/Sources/SavedGames.c:14`),
`kPrefsVersion 0x0034` (`GliderPRO/Sources/Main.c:16`). The application version
appears **only** in the `'vers'` resources and the About box.

### Decision

1. **Retail is `BUILD_ARCADE_VERSION = 0`.** Port the `#else` semantics: arrow
   keys drive the editor, `Cmd` during play/demo runs `DoCommandKey`, and there is
   no scoreboard blackout. Implement the arcade behaviour, if at all, behind a
   runtime flag (`--arcade`), never a compile-time one — the two differ only in
   input routing and two repaint calls, so a boolean is enough.
2. **The `1` in the header is the state of Calhoun's own working copy at the time
   of the 2000-ish Carbon port**, not the shipped configuration. Treat every
   `#define` in `GliderDefines.h:11-16` the same way — a snapshot of one
   developer's build, not a spec:

   | Macro | State in the source | Line | Meaning for a port |
   |---|---|---:|---|
   | `CREATEDEMODATA` | commented out | 11 | do not build the demo *recorder* by default |
   | `COMPILEDEMO` | commented out | 12 | not a crippled shareware demo |
   | `CAREFULDEBUG` | commented out | 13 | no extra assertions |
   | `COMPILENOCP` | **defined** | 14 | keep it defined; do not port `Validate.c` |
   | `COMPILEQT` | **defined** | 15 | QuickTime movie support in rooms; needs a Go replacement or a stub |
   | `BUILD_ARCADE_VERSION` | **1** | 16 | **override to 0** |
3. **The version of the software you are porting is 1.1.2, released by Casady &
   Greene, © 1994-95.** "1.0.4" is a stale comment at `Main.c:3`. Use 1.1.2 in any
   About box, any `'vers'`-equivalent metadata, and any user-visible string. Say
   "Glider PRO 1.1.2 (source release labelled 1.0.4 in `Main.c`)" in
   documentation so nobody re-opens this.
4. **Nothing on disk depends on either answer.** Neither the build flag nor the
   application version appears in any house file, saved game, or preferences
   record. There is **no format consequence** — which is why this decision belongs
   in one place and should never again be listed as an open format question.

### Consequence of deciding otherwise

| If you… | Observable consequence |
|---|---|
| build with `BUILD_ARCADE_VERSION = 1` | in the editor, Left/Right/Up/Down open the High Scores dialog, open Help, and start a new game (twice) instead of moving between rooms and nudging objects. Room navigation and 1-pixel object placement become mouse-only. This is the single most visible regression available from a one-line choice |
| build with `BUILD_ARCADE_VERSION = 1` | during a demo, pressing any of the four gameplay keys ends the demo instead of doing nothing; `Cmd-Q` and `Cmd-S` do nothing during a demo instead of quitting/saving |
| build with `BUILD_ARCADE_VERSION = 1` | the scoreboard is painted black and re-blitted from `PICT` at every game over and at the end of every game, so the final score flashes and disappears |
| make `BUILD_ARCADE_VERSION` a compile-time constant in Go | you need two binaries to test both, and Go has no `#if`; the `//go:build` tag route means the arcade path is never compiled in CI. Use a runtime bool |
| port `Validate.c` / re-add `encrypted` + `fakeLong` to the prefs record | your preferences file becomes 234 bytes with `prefVersion` at offset 172, which is not what any surviving Glider PRO prefs file looks like, and the copy-protection check keys off volume identity — meaningless on a modern machine and guaranteed to fail |
| write "1.0.4" in the About box | contradicts the `'vers'` resource the original itself reads at `About.c:55-61`; a user comparing your port with a real copy sees a version mismatch |
| assume the arcade flag changes scoring or saving | it does not; you will go looking for a save-disabled code path that does not exist. `TestHighScore()` at `GameOver.c:503` and `SaveGame`/`WriteHouse` are outside every arcade conditional |

### Go

```go
// Build configuration. In the original these are C preprocessor macros in
// GliderDefines.h:11-16; here they are runtime flags so both paths compile.
type BuildConfig struct {
    Arcade        bool // BUILD_ARCADE_VERSION; retail == false (see D6)
    QuickTime     bool // COMPILEQT; original == true
    CreateDemoData bool // CREATEDEMODATA; original == false
    Demo          bool // COMPILEDEMO; original == false
    CarefulDebug  bool // CAREFULDEBUG; original == false
    // COMPILENOCP is not represented: copy protection is never ported.
}

var Retail = BuildConfig{Arcade: false, QuickTime: true}

const (
    AppVersion      = "1.1.2"                              // 'vers' 1 and 2
    AppCopyright    = "© 1994-95 Casady & Greene, Inc." // 0xA9 in MacRoman
    AppLongVersion  = "Glider PRO™ 1.1.2"              // 0xAA in MacRoman is (tm)
    // Main.c:3 says "Glider PRO 1.0.4"; that comment is stale. Do not use it.
)
```

Note the two MacRoman code points in the version strings: `0xAA` is TRADE MARK
SIGN (U+2122) and `0xA9` is COPYRIGHT SIGN (U+00A9). Under Latin-1 they would
decode as `ª` and `©` — get the table right or the About box reads
"Glider PROª 1.1.2".

---

## D7 — Demo-playback termination

### The question as previously posed

* **architecture.md OQ12:** "`GetDemoInput` compares `gameFrame` with
  `demoData[demoIndex].frame` and never bounds `demoIndex`. What stops it reading
  past the 1117th record?"
* **input.md OQ6:** "How does demo playback end? There is no sentinel record and
  no length check."

### Evidence

**1. The buffer, exactly.**

```c
#define kDemoLength                 6702      /* GliderDefines.h:625 */

typedef struct {                              /* GliderStructs.h:334-339 */
    long        frame;      /* 4 bytes, big-endian, offset 0 */
    char        key;        /* 1 byte,  signed,     offset 4 */
    char        padding;    /* 1 byte,  signed,     offset 5 */
} demoType, *demoPtr;                          /* sizeof == 6 under pack(2) */
```

`GliderPRO/Sources/StructuresInit2.c:280-298`:

```c
#ifdef CREATEDEMODATA
    demoData = (demoPtr)NewPtr(sizeof(demoType) * 2000);      /* 12000 bytes */
    if (demoData == nil) RedAlert(kErrNoMemory);
#else
    demoData = (demoPtr)NewPtr(kDemoLength);                  /*  6702 bytes */
    if (demoData == nil) RedAlert(kErrNoMemory);
    tempHandle = GetResource('demo', 128);
    if (tempHandle == nil) RedAlert(kErrNoMemory);
    else { BlockMove(*tempHandle, demoData, kDemoLength); ReleaseResource(tempHandle); }
#endif
```

`6702 / 6 = 1117` records exactly, remainder 0. Confirmed by extracting
`data 'demo' (128)` from `GliderPRO/Glider PRO.r:199389-199809` — 6702 bytes.
`kDemoLength` is **not a capacity**: `DumpToResEditFile((Ptr)demoData, sizeof(demoType) * (long)demoIndex)` at
`GliderPRO/Sources/Play.c:217` writes exactly `demoIndex` records, so 6702 is the
size of the one recording Calhoun happened to make (`demoIndex` was 1117 when he
stopped). `sizeof(demoType)` would be **8** under natural alignment, which would
make `BlockMove` of 6702 bytes cover only 837.75 records and garble every frame
number after the first — an independent confirmation of [D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant).

**2. The observed content of `'demo'` 128** (parsed with `python3`, big-endian):

| Property | Value |
|---|---|
| length | 6702 bytes = 1117 records, remainder 0 |
| `frame` range | 46 … 3414 |
| `frame` monotonicity | strictly increasing, no duplicates, no zeros |
| first record | `frame=46, key=0, padding=114` |
| last record | `frame=3414, key=0, padding=1` |
| `key` histogram | `0`: 910, `1`: 198, `3`: 9, **`2`: 0** |
| `key` values outside 0..3 | none |
| `padding` | 109 distinct values, 407 zeros — heap residue (see 4 below) |
| inter-record gap histogram (top 6) | `1`: 1009, `7`: 8, `4`: 6, `8`: 6, `3`: 6, `10`: 6 |
| largest gap | 100 frames |
| last 12 records | `(3401,1) (3402,1) (3403,1) (3404,1) (3405,1) (3408,0) (3409,0) (3410,0) (3411,0) (3412,0) (3413,0) (3414,0)` |

**3. The demo key encoding, and the fact that the source comments are wrong.**
`GliderPRO/Sources/Input.c:226-264` dispatches on `demoData[demoIndex].key`:

| `key` | Comment in the source | What the code does | Actual meaning |
|---:|---|---|---|
| 0 | `// left key` | `hDesiredVel += kNormalThrust; tipped = (facing == kFaceLeft); heldRight = true;` | **right** |
| 1 | `// right key` | `hDesiredVel -= kNormalThrust; tipped = (facing == kFaceRight); heldLeft = true;` | **left** |
| 2 | `// battery key` | `if (batteryTotal > 0) DoBatteryEngaged() else DoHeliumEngaged()` | battery / helium |
| 3 | `// rubber band key` | `AddBand(…, dest.left + 24, dest.top + 10, facing); bandsTotal--;` | rubber band |

Cross-checked against the recorder, `GetInput` (`GliderPRO/Sources/Input.c:281-379`),
which is the authority because it produced the bytes in the resource:

| Recorder call | Branch it is in | Line |
|---|---|---:|
| `LogDemoKey(0)` | `BitTst(&theKeys, thisGlider->rightKey)` | `Input.c:304` |
| `LogDemoKey(1)` | `BitTst(&theKeys, thisGlider->leftKey)` | `Input.c:321` |
| `LogDemoKey(2)` | `BitTst(&theKeys, thisGlider->battKey)` | `Input.c:334` |
| `LogDemoKey(3)` | `BitTst(&theKeys, thisGlider->bandKey)` | `Input.c:348` |

So **0 = right, 1 = left, 2 = battery, 3 = band**, and the two comments at
`Input.c:228` and `Input.c:235` are swapped. This is self-consistent (the
recorder's 0 and the player's 0 both mean right); only the comments lie. `key == 2`
never appears in the shipped demo, so the demo glider never uses the battery.

**4. `padding` is uninitialised, by construction.**

```c
void LogDemoKey (char keyIs)            /* GliderPRO/Sources/Input.c:44-49 */
{
    demoData[demoIndex].frame = gameFrame;
    demoData[demoIndex].key = keyIs;
    demoIndex++;
}
```

It writes `frame` and `key` and never `padding`, and the recording buffer comes
from `NewPtr` (`StructuresInit2.c:282`), not `NewPtrClear`. Hence 109 distinct
`padding` values with 407 zeros: 1994 Mac heap residue, frozen into a resource in
1994 and shipped ever since. It is the exact analogue of D3 and D4 at record
granularity.

**5. Only one record can fire per frame, and records are consumed one at a time.**
`GliderPRO/Sources/Input.c:224-267`:

```c
if (gameFrame == (long)demoData[demoIndex].frame)
{
    switch (demoData[demoIndex].key) { … }
    demoIndex++;
}
else
    thisGlider->fireHeld = false;
```

The comparison is `==`, not `>=`, and the body is not a loop. One record per frame
is nevertheless sufficient for the *shipped* stream, because its frames are
strictly increasing — verified above, empirically.

**It is not structurally guaranteed.** `GetInput`'s four `LogDemoKey` call sites
are **not** four alternatives: `LogDemoKey(0)` (`Input.c:304`) and `LogDemoKey(1)`
(`:321`) are the two arms of one `if` / `else if` (`:301` / `:318`), but
`LogDemoKey(2)` (`:334`) sits under a **separate** `if (BitTst(&theKeys,
thisGlider->battKey) …)` at `:330` and `LogDemoKey(3)` (`:348`) under another
separate `if` at `:344`. So a recorder frame in which the player held a direction
*and* the battery *and* the band key would emit **three** records with the same
`frame`, and playback — which consumes one record per matching frame — would drop
two of them and then be permanently off by two records. The shipped resource
happens to contain no duplicate frames (`key == 2` never appears at all), so this
never bites; a port that re-records demos must either emit one record per frame or
consume records in a loop. It is also **fragile** in the other direction: see 7.

**6. `demoIndex` is never bounds-checked, in either direction.**

| Fact | Citation |
|---|---|
| `short demoIndex;` — a signed 16-bit index | `GliderPRO/Sources/Input.c:33` |
| `demoIndex = 0` at the start of every game | `GliderPRO/Sources/Play.c:114` (in `NewGame`) |
| read of `demoData[demoIndex].frame` with no upper bound | `GliderPRO/Sources/Input.c:224` |
| `demoIndex++` with no upper bound | `GliderPRO/Sources/Input.c:266` (playback), `:48` (recording) |
| `demoData` is a bare `Ptr`, 6702 bytes, no length carried | `GliderPRO/Sources/StructuresInit2.c:287` |

So after the 1117th record is consumed, `demoIndex == 1117` and `Input.c:224`
reads `demoData[1117].frame` — bytes 6702..6705 of a 6702-byte block. On classic
Mac OS a `NewPtr` block is rounded up and preceded/followed by heap bookkeeping,
so the read is harmless and returns junk; the junk is compared against
`gameFrame` and almost never matches, so playback simply stops producing input.
**In Go this is a panic**, and it is guaranteed to happen on every single demo.

**7. Two ways the stream can stall early — both real.**

*(a) Burning mode skips the whole dispatch.* `GetDemoInput`'s body is
`if (mode == kGliderBurning) { … } else { …the demoData dispatch… }`
(`GliderPRO/Sources/Input.c:211-276`). While the glider burns, frames advance but
no record is consumed. If the pending record's `frame` passes during the burn, the
`==` never matches again and **the demo stream is permanently stalled** — the
glider flies the rest of the demo with no input at all.

*(b) `gameOver` skips it too.* `GliderPRO/Sources/Play.c:475-483`:

```c
HandleDynamics();
if (!gameOver)
{
    if (demoGoing)  GetDemoInput(&theGlider);
    else            GetInput(&theGlider);
    HandleInteraction();
}
```

Once `gameOver` is set, `GetDemoInput` is not called at all, so `demoIndex`
freezes — which is what keeps the out-of-bounds read from repeating during the
16-frame countdown.

**8. What actually ends the demo: the game, not the stream.** The termination path
is a chain, none of whose links looks at `demoIndex`:

1. `gameFrame++` at the top of `PlayGame`'s loop (`GliderPRO/Sources/Play.c:434`),
   so the first frame is **1**. The first record is at frame 46, i.e. the demo
   glider does nothing for 45 frames.
2. `RenderFrame()` blocks on `while (TickCount() < nextFrame)`
   (`GliderPRO/Sources/Render.c:662`) and then sets
   `nextFrame = TickCount() + kTicksPerFrame` (`:665`, `:690`) with
   `kTicksPerFrame 2` (`GliderDefines.h:533`) → 60/2 = **30 fps**. Frame 46 is
   1.53 s in; frame 3414 is **113.8 s** in.
3. When the last glider dies, `FlagGameOver` (`GliderPRO/Sources/GameOver.c:237-242`)
   sets `gameOver = true; countDown = kNumCountDownFrames;` with
   `kNumCountDownFrames 16` (`GameOver.c:17`).
4. `PlayGame` decrements `countDown` each frame, inside `if (gameOver)`
   (`Play.c:499-502`), and at
   `<= 0` calls `DoDiedGameOver()` if `mortals < 0`, else `DoGameOver()`
   (`Play.c:546-549`).
5. `DoDiedGameOver` (`GliderPRO/Sources/GameOver.c:443-506`) runs the
   page-flutter animation `while (pagesStuck < 8)` (`:457`), abortable by
   Cmd/Option/Shift/Control or any `mouseDown`/`keyDown` (`:464-475`, setting
   `userAborted`), then `playing = false` (`:492`), which exits `PlayGame`'s
   `while ((playing) && (!quitting))`.
6. The demo-specific bit (`GameOver.c:494-504`):

   ```c
   if (demoGoing)  { if (!userAborted) WaitForInputEvent(1); }
   else            { if (!userAborted) WaitForInputEvent(10); TestHighScore(); }
   ```

   A demo waits **1 tick** instead of 10 and **never records a high score**.
7. Back in `NewGame` (`GliderPRO/Sources/Play.c:276-277`): `demoGoing = false;
   incrementModeTime = TickCount() + kIdleSplashTicks;`, and `DoDemoGame`
   (`Play.c:302`) sets it again after reopening the player's own house.

There is a third demo-aware timing site: `Banner.c:189-192` waits 4 ticks instead
of 15 for banner/trailer pages when `demoGoing`.

**9. The demo is fully deterministic between frame 3414 and the death.** Between
the last record and game over there is *no* input; the glider's fate is decided
entirely by `HandleDynamics`/`HandleGlider`/`HandleInteraction`. So a port whose
physics differ by one pixel will produce a demo that dies at a different frame —
and, because the out-of-bounds read is the *only* thing that happens after 3414,
a port that silently clamps `demoIndex` will look correct while diverging.

**10. How the demo starts.** `GliderPRO/Sources/Events.c:536-539`:

```c
if ((theMode == kSplashMode) && doAutoDemo && !switchedOut)
{
    if (TickCount() >= incrementModeTime)
        DoDemoGame();
}
```

`kIdleSplashTicks 7200L` (`GliderDefines.h:197`) = 7200/60 = **120 s** of idle on
the splash screen. `doAutoDemo` defaults to `true` (`Main.c:183` when no prefs
file exists; `Settings.c:1227` in the "restore defaults" path) and is persisted as
`thePrefs.wasDoAutoDemo` (`Main.c:113`, `:268`). `DoDemoGame`
(`GliderPRO/Sources/Play.c:282-303`) closes the current house, switches
`thisHouseIndex = demoHouseIndex`, reads that house, sets `demoGoing = true`,
runs `NewGame(kNewGameMode)`, then restores the previous house.

`demoHouseIndex` is set in `GliderPRO/Sources/SelectHouse.c:636-641`: initialised
to `-1` and set to `i` only when a house file's name equals the literal
`"\pDemo House"`. **`Events.c:536` does not test `demoHouseIndex != -1`**, so on an
installation with no file named exactly `Demo House` the auto-demo fires after two
idle minutes and indexes `theHousesSpecs[-1]` — reading an `FSSpec` from 70 bytes
before the array. (`Menu.c:88-91` *does* check, but only to grey out
Options▸Help.)

**The house the demo was recorded against is present in this repository.**
`GliderPRO/Houses/Demo House.binhex` decodes to a data fork of exactly **16526
bytes** with `nRooms == 45` and `firstRoom == 0` — satisfying all **four**
`COMPILEDEMO` gates in `ReadHouse` simultaneously (`HouseIO.c:349`
`byteCount != 16526L`, `:382` `numberRooms != 45`, `:403-406`
`if (houseUnlocked) return false;`, `:412` `whichRoom != 0`; the third is
satisfied because `timeStamp` 743682113 is odd, so bit 0 is set and the house is
locked), and
matching `866 + 348 × 45 = 16526`. Its BinHex header gives
`name = 'Demo House'`, `type = 'gliH'`, `creator = 'ozm5'`, resource fork 491,757
bytes, `version = 0x0200`, `timeStamp = 743682113` (bit 31 restored:
1995-08-13 13:36:01), `flags = 0x00`, `unusedShort = 0x0000`, `hasGame = 0`, and
banner `"Welcome to the Demo House!\rThis is a small beginner house that\racts as a
sort of tutorial.\r(house by Kim Money)"` (111 bytes). So a Go port can replay
`'demo'` 128 against the real house and compare, frame by frame, with no missing
artefacts. **This is the strongest end-to-end determinism test available for a
Glider PRO port.**

### Decision

**The demo terminates when the game does, not when the record stream does.
Playback is a pure overlay on the normal game loop.** Implement it as:

1. Load `'demo'` 128 into a `[]DemoRecord` of `len(resource)/6` records. Do **not**
   hard-code 1117; do reject a resource whose length is not a multiple of 6, and
   warn if it is not exactly 6702.
2. `demoIndex = 0` at `NewGame`, exactly as `Play.c:114`.
3. In `GetDemoInput`, guard the read:
   `if demoIndex < len(demo) && gameFrame == int32(demo[demoIndex].Frame)`.
   Keep `==` (not `>=`) and keep it a single `if` (not a loop), because that is
   what the original does and the stall behaviour in 7(a) is observable.
4. **Do not add a sentinel, a length field, or an "end of demo" event.** There is
   no such thing in the format. When the records run out the glider keeps flying
   with no input until it dies.
5. Termination is `playing = false`, set by `DoDiedGameOver`/`DoGameOver` after the
   16-frame `countDown`; plus the user abort inside `DoDiedGameOver`; plus the
   arcade-only "any gameplay key" abort ([D6](#d6--the-retail-build-configuration-and-the-version-identity), off in retail).
6. Preserve `padding` byte-for-byte if you ever re-emit the resource; it is
   residue (evidence 4) and re-emitting zeros changes 710 of 1117 bytes.
7. Use **0 = right, 1 = left, 2 = battery, 3 = band**. Copy the comment correction
   into your code so the next reader does not "fix" it back.
8. Guard `demoHouseIndex != -1` before `DoDemoGame()`, unlike `Events.c:536`.

### Consequence of deciding otherwise

| If you… | Observable consequence |
|---|---|
| transcribe `Input.c:224` literally | `panic: index out of range [1117] with length 1117` at the end of **every** demo — the demo always outlives its records, since frame 3414 is reached ~114 s in and the glider dies later |
| trust the `// left key` / `// right key` comments | the demo glider flies mirror-image: 910 right-presses become left-presses. It walks off the wrong side of the first room and dies in seconds. Nothing crashes, so this looks like a physics bug |
| change `==` to `>=`, or wrap the dispatch in a `while` | you "fix" the burning-mode stall of 7(a) and the demo diverges from the original from the first burn onward. Any accumulated timing difference now also replays as a burst of catch-up input |
| consume records while `gameOver` is set | `demoIndex` advances during the 16-frame countdown, so the out-of-bounds read happens 16 times instead of once, and any port that clamps instead of panicking changes the countdown's visuals |
| add a sentinel record or an explicit end-of-demo | the demo ends early, the splash screen returns before the glider dies, and the game-over page-flutter animation (which *is* part of the demo, `GameOver.c:457`) never plays |
| zero-fill `padding` when re-emitting `'demo'` 128 | 710 of the 1117 padding bytes change; the resource is no longer byte-identical and any checksum over the resource fork fails |
| call `TestHighScore()` on the demo path | the demo's score is entered into the player's high-score table. The original explicitly does not (`GameOver.c:494-504`) |
| use `WaitForInputEvent(10)` on the demo path | the post-demo pause is 10 ticks instead of 1 — visible as a 150 ms hitch before the splash screen returns |
| assume `sizeof(demoType)` is 8 | you read 837 records out of 6702 bytes with `frame` fields spanning record boundaries: frame numbers like 0x0000002E00000000. Nothing matches `gameFrame`, so the demo glider never receives input at all and dies in the first room |
| omit the `demoHouseIndex != -1` guard | after two idle minutes on the splash screen with no `Demo House` installed, you read `theHousesSpecs[-1]` — a panic in Go, garbage in C |
| hard-code 1117 | correct for the shipped resource, wrong for a re-recorded one; `kDemoLength` is the *length of one recording*, not a format constant |

### Go

```go
// DemoRecord mirrors demoType (GliderStructs.h:334-339): 6 bytes, big-endian.
// padding is NOT a reserved field: LogDemoKey (Input.c:44-49) never writes it and
// the recording buffer came from NewPtr, so it is 1994 heap residue. Preserve it.
type DemoRecord struct {
    Frame   int32 // offset 0, 4 bytes, big-endian
    Key     int8  // offset 4: 0=RIGHT 1=LEFT 2=battery 3=band  (comments in the
                  //           original at Input.c:228 and :235 are swapped)
    Padding int8  // offset 5: residue
}

const (
    SizeDemoRecord   = 6    // NOT 8; see D2
    KDemoLength      = 6702 // GliderDefines.h:625 == the shipped recording's length
    ShippedDemoCount = 1117 // 6702/6, for a sanity warning only
)

const (
    DemoKeyRight   int8 = 0
    DemoKeyLeft    int8 = 1
    DemoKeyBattery int8 = 2 // never appears in the shipped demo
    DemoKeyBand    int8 = 3
)

func ParseDemo(b []byte) ([]DemoRecord, error) {
    if len(b)%SizeDemoRecord != 0 {
        return nil, fmt.Errorf("'demo' 128 is %d bytes, not a multiple of %d",
            len(b), SizeDemoRecord)
    }
    if len(b) != KDemoLength {
        log.Printf("'demo' 128 is %d bytes; the shipped resource is %d", len(b), KDemoLength)
    }
    recs := make([]DemoRecord, len(b)/SizeDemoRecord)
    for i := range recs {
        p := b[i*SizeDemoRecord:]
        recs[i] = DemoRecord{
            Frame:   int32(binary.BigEndian.Uint32(p[0:4])),
            Key:     int8(p[4]),
            Padding: int8(p[5]),
        }
    }
    return recs, nil
}

// GetDemoInput mirrors Input.c:186-277. The ONLY deviation from the original is
// the `demoIndex < len(demo)` guard, which replaces an out-of-bounds read that is
// benign on Mac OS and fatal in Go.
func (g *Game) GetDemoInput(gl *Glider) {
    if gl.Which == Player1 {
        g.GetKeys(&g.theKeys)
        if g.cfg.Arcade {
            if g.theKeys.Any(gl.LeftKey, gl.RightKey, gl.BattKey, gl.BandKey) {
                g.playing, g.paused = false, false
            }
        } else if g.theKeys.Test(KCommandKeyMap) {
            g.DoCommandKey()
        }
    }

    if gl.Mode == GliderBurning {
        // NOTE: no record is consumed while burning. If the pending record's
        // frame elapses here, the == below never matches again and the demo
        // runs input-free to the end. This is original behaviour. Keep it.
        if gl.Facing == FaceLeft { gl.HDesiredVel -= KNormalThrust } else { gl.HDesiredVel += KNormalThrust }
        return
    }

    gl.HeldLeft, gl.HeldRight, gl.Tipped = false, false, false

    if g.demoIndex < len(g.demo) && g.gameFrame == int64(g.demo[g.demoIndex].Frame) {
        switch g.demo[g.demoIndex].Key {
        case DemoKeyRight: // case 0; the original comment says "left key"
            gl.HDesiredVel += KNormalThrust
            gl.Tipped = gl.Facing == FaceLeft
            gl.HeldRight = true
            gl.FireHeld = false
        case DemoKeyLeft: // case 1; the original comment says "right key"
            gl.HDesiredVel -= KNormalThrust
            gl.Tipped = gl.Facing == FaceRight
            gl.HeldLeft = true
            gl.FireHeld = false
        case DemoKeyBattery:
            if g.batteryTotal > 0 { g.DoBatteryEngaged(gl) } else { g.DoHeliumEngaged(gl) }
            gl.FireHeld = false
        case DemoKeyBand:
            if !gl.FireHeld {
                if g.AddBand(gl, gl.Dest.Left+24, gl.Dest.Top+10, gl.Facing) {
                    g.bandsTotal--
                    if g.bandsTotal <= 0 { g.QuickBandsRefresh(false) }
                    gl.FireHeld = true
                }
            }
        }
        g.demoIndex++ // exactly one record per frame; never a loop
    } else {
        gl.FireHeld = false
    }

    if (g.isEscPauseKey && g.theKeys.Test(KEscKeyMap)) ||
        (!g.isEscPauseKey && g.theKeys.Test(KTabKeyMap)) {
        g.DoPause()
    }
}
```

Timing constants a Go port needs for the demo to look right:

| Constant | Value | Citation | Meaning |
|---|---:|---|---|
| `kTicksPerFrame` | 2 | `GliderDefines.h:533` | 60 Hz tick clock / 2 = **30 fps** |
| `kIdleSplashTicks` | 7200 | `GliderDefines.h:197` | 120 s idle before the auto-demo |
| `kNumCountDownFrames` | 16 | `GliderPRO/Sources/GameOver.c:17` | frames between `gameOver` and `DoDiedGameOver` |
| `pagesStuck` limit | 8 | `GliderPRO/Sources/GameOver.c:457` | page-flutter animation length |
| `WaitForInputEvent` (demo) | 1 tick | `GliderPRO/Sources/GameOver.c:497` | vs 10 for a real game (`:502`) |
| banner wait (demo) | 4 ticks | `GliderPRO/Sources/Banner.c:190` | vs 15 for a real game (`:192`) |
| first demo record | frame 46 | `'demo'` 128 | 1.53 s of no input at the start |
| last demo record | frame 3414 | `'demo'` 128 | 113.8 s; the glider then flies uncontrolled until it dies |

---

## 9. Which open question each decision closes

The sibling documents number their open questions implicitly, as ordered list
items under `## Open questions`. The audit that commissioned this document
referred to them as `OQ<n>`; that is the numbering used here, and the "first
words" column lets you find each item by `command grep` without counting.

Verdicts: **closed** = the item is answered and its `## Open questions` entry
should be struck through and pointed here; **narrowed** = part of the item is
answered and a genuinely unanswerable residue remains (carried into
[§ Open questions](#open-questions) below); **corrected** = the sibling document
asserts something this document disproves.

The same mapping, indexed the other way — by sibling document, with the exact
line number of each list item as it stands today — is in
[unresolved-format-decisions.md](unresolved-format-decisions.md). That file is a
navigational index only; this document is the normative one.

### 9.1 By sibling document

| Document | Item | First words | Verdict | Resolved by |
|---|---|---|---|---|
| architecture.md | OQ1 | "Which version is this?" | **closed** | [D6](#d6--the-retail-build-configuration-and-the-version-identity) — 1.1.2; `Main.c:3` is stale |
| architecture.md | OQ10 | "The `idleMode` state machine is vestigial" | **corrected** | [D7](#d7--demo-playback-termination) ev. 10 — `idleMode` *is* vestigial (written once at `InterfaceInit.c:162`, never read), but `kIdleSplashTicks` 7200L is **not** unused: 9 live assignments (`Events.c:427`, `InterfaceInit.c:163`, `Menu.c:336`/`:402`/`:419`/`:424`, `Play.c:277`/`:302`, `AppleEvents.c:111`). It *is* the attract interval |
| architecture.md | OQ11 | "Do the padding bytes matter?" | **closed** | [D7](#d7--demo-playback-termination) ev. 4 (`demoType.padding` is `LogDemoKey` residue; preserve to round-trip) + [D6](#d6--the-retail-build-configuration-and-the-version-identity) (`prefsInfo` is 226 bytes in this build, 234 in retail — the pad-byte offsets differ between the two) |
| architecture.md | OQ12 | "What terminates demo playback?" | **closed** | [D7](#d7--demo-playback-termination) |
| architecture.md | OQ14 | "`game2Type` on-disk size" | **closed** | [D1](#d1--sizeofgame2type-is-110-not-114) — 110 |
| architecture.md | PN13 | "`BUILD_ARCADE_VERSION` is 1" | **corrected** | [D6](#d6--the-retail-build-configuration-and-the-version-identity) — true of the source, but retail was 0; override it |
| progression.md | OQ1 | "`sizeof(game2Type)` under the shipping compiler" | **closed** | [D1](#d1--sizeofgame2type-is-110-not-114) |
| progression.md | OQ2 | "Which build was actually shipped as 1.0.4?" | **closed** | [D6](#d6--the-retail-build-configuration-and-the-version-identity) — arcade = 0, and nothing about progression changes either way |
| progression.md | OQ4 | "`houseType.unusedShort` (offset 2) is uninitialised garbage" | **closed** | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) |
| progression.md | OQ5 | "`roomType.unusedByte` … `savedRoom.unusedShort`/`unusedByte`" | **closed** | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) — `roomType.unusedByte` is force-zeroed (`Room.c:173`, `HouseLegal.c:868`); the `savedRoom` pair is unreachable because `SaveGame2` is commented out ([D1](#d1--sizeofgame2type-is-110-not-114)) |
| progression.md | PN3 | "Read the house as `866 + 348 * nRooms` bytes, but accept a longer file" | **closed** (confirmed and made normative) | [D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant) + [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length) + [D5](#d5--the-bound-on-housetypenrooms) |
| progression.md | PN6 | "`houseType.unusedShort` at offset 2 is garbage" | **closed** (confirmed) | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) — and extended: also `.unusedBoolean`, `.hasGame`, the whole `.savedGame` block, and every `StrNN` tail |
| scoring.md | OQ4 | "What was `unusedShort` at house offset 2 for?" | **closed** | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) |
| scoring.md | OQ5 | "Why do two pairs of unrelated houses have byte-identical …" | **closed** | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) — the pairs share a stale 40-byte `savedGame` block at +820 (`CD Demo House` / `Slumberland`, `Nemo's Market` / `SpacePods`); the **files** are not duplicates (72554 vs 134150 bytes; 44018 vs 140762) |
| scoring.md | OQ7 | "Does the `game2Type` header occupy 110 or 114 bytes?" | **closed** | [D1](#d1--sizeofgame2type-is-110-not-114) |
| scoring.md | OQ12 | "`vers` says 1.1.2, the task brief says 1.0.4" | **closed** | [D6](#d6--the-retail-build-configuration-and-the-version-identity) |
| scoring.md | OQ24 | "The `Sampler` house's data fork is 2 bytes longer" | **closed** | [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length) |
| scoring.md | OQ25 | "`houseType.unusedBoolean` at offset 861 holds 255, 30, 37, 185, 30, …" | **closed** | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) |
| house-format.md | §12.4 | "The `Sampler` 2-byte tail" | **closed** | [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length) |
| house-format.md | OQ8 | "`savedGame.roomNumber` is a room index that `CompressHouse` does not patch" | **narrowed** | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) — moot for 20 of 22 houses (the block is residue) and unreachable in code (`QueryResumeGame` has no callers, `OpenSavedGame` returns `false` at `SavedGames.c:169`). Still genuinely unanswerable for the 2 real saves |
| house-format.md | PN2 | "Parse rooms at `866 + 348*r`, not at `sizeof(struct)*r`" | **closed** (confirmed) | [D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant) — with the `Demo House` = 16526 proof the sibling docs assert without |
| house-format.md | PN19, PN20 | "the whole `savedGame` block when …", "`unusedShort`, `unusedBoolean`, and the 10 data bytes of an empty object slot" | **closed** (confirmed and extended) | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) |
| object-taxonomy.md | OQ1 | "`Sampler.binhex` is 2 bytes longer than `866 + nRooms * 348`" | **closed** | [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length) |
| object-taxonomy.md | PN9 | "Round-trip test: decode and re-encode all 22 shipped houses and require byte equality" | **narrowed** | [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length) + [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) — the test is achievable, but only if trailing slack and **all** `StrNN` tails are preserved; naive `string` round-tripping fails 22/22 |
| interactions.md | OQ5 | "`Sampler.binhex`'s 2 extra bytes" | **closed** | [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length) |
| constants.md | OQ1 | "Is the `floor + 7` in `CheckDuplicateFloorSuite` an off-by-one?" | **closed** | [D5](#d5--the-bound-on-housetypenrooms) corollary — no: `bitPlace` ranges exactly 0..8191 over the legal `(floor, suite)` domain, filling `kRoomsTimesSuites` 8192 with no slack. The bug is the *call order* (`:1089` before `:1110`), not the bias |
| constants.md | OQ5 | "Are `unusedShort` / `unusedBoolean` / `unusedByte` / `unusedLong` vestigial or …" | **closed** | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) — vestigial *and* uninitialised; the distinction is per-field and tabulated there |
| constants.md | OQ8 | "Was `game2Type` / the external `.gliG` saved game ever functional?" | **closed** | [D1](#d1--sizeofgame2type-is-110-not-114) — no: `SaveGame2`'s body is commented out (`SavedGames.c:33-147`) and `OpenSavedGame` returns `false` before its body (`:169`) |
| constants.md | OQ9 | "Why does `Sampler.binhex` have 2 trailing bytes?" | **closed** | [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length) |
| constants.md | OQ12 | "Does the `BUILD_ARCADE_VERSION 1` / `COMPILENOCP` / `COMPILEQT` configuration …" | **closed** | [D6](#d6--the-retail-build-configuration-and-the-version-identity) — per-macro table in the Decision |
| constants.md | OQ14 | "Is there any bound on `houseType.nRooms` at load time?" | **closed** | [D5](#d5--the-bound-on-housetypenrooms) |
| original-houses.md | OQ1 | "What exactly does `unusedShort` at `houseType` offset 2 hold?" | **closed** | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) |
| original-houses.md | OQ2 | "What was `roomType.unusedByte` (offset 32) for?" | **narrowed** | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) — its *value* is settled (always 0 on disk, force-zeroed at `Room.c:173` and `HouseLegal.c:868`); its original *purpose* is not recoverable |
| original-houses.md | OQ8 | "Which of the 22 houses were shipped with the retail 1.0.4 disk versus added …" | **narrowed** | [D6](#d6--the-retail-build-configuration-and-the-version-identity) — "retail 1.0.4" is the wrong frame (retail was 1.1.2, © 1994-95); [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate)'s shared-`savedGame` finding (common ancestor files, not duplicate files) and the 2000-05-11 `Sampler`/`Slumberland` timestamps give partial provenance. Full provenance needs the CD-ROM |
| original-houses.md | PN1 | "`houseType` is 866 bytes" | **closed** (confirmed) | [D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant) |
| resource-fork.md | OQ5 | "The About box's copyright disagrees with `'vers'`" | **closed** | [D6](#d6--the-retail-build-configuration-and-the-version-identity) — the `DITL` was updated in 2000 during the Carbon port; the `'vers'` was not. The running About box shows both strings |
| resource-fork.md | OQ6 | "The fork claims version 1.1.2, the source release is called 1.0.4" | **closed** | [D6](#d6--the-retail-build-configuration-and-the-version-identity) — use 1.1.2 |
| input.md | OQ1 | "Is the swapped `// left key` / `// right key` comment pair in `GetDemoInput` …" | **closed** | [D7](#d7--demo-playback-termination) ev. 3 — yes, the comments are swapped; `GetInput`'s `LogDemoKey` call sites (`Input.c:304`/`:321`/`:334`/`:348`) are the authority, since they produced the shipped bytes |
| input.md | OQ6 | "Does any house rely on the out-of-bounds `demoData[demoIndex]` read past 1117 records?" | **closed** | [D7](#d7--demo-playback-termination) ev. 6, 8 — no house is involved; the read happens once, in every demo, after frame 3414, and is benign only because Mac `NewPtr` blocks have slack |
| input.md | PN16 | "Prefs are 226 bytes with two uninitialised pad bytes" | **closed** (confirmed) | [D6](#d6--the-retail-build-configuration-and-the-version-identity) — 226 verified by compilation; retail's was **234** with `prefVersion` at 172 instead of 164 |
| input.md | PN17 | "The demo stream is 1117 six-byte big-endian records" | **closed** (confirmed and extended) | [D7](#d7--demo-playback-termination) — plus the frame range, key histogram, and the `sizeof(demoType)` = 6-not-8 dependency on [D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant) |
| editor.md | Q4 | "Why does `Sampler` have 2 trailing bytes and an empty resource fork?" | **closed** (trailing bytes) / **narrowed** (empty fork) | [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length); the empty resource fork is a separate, unrelated question |
| editor.md | Q10 | "What did the reserved fields hold?" | **closed** | [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) |
| structs.md | §2 row `game2Type` | "110 / 112 / yes" | **closed** (confirmed) | [D1](#d1--sizeofgame2type-is-110-not-114) — structs.md was already right; the other four documents were not |

### 9.2 By decision

| Decision | Closes | Narrows | Corrects |
|---|---|---|---|
| [D1](#d1--sizeofgame2type-is-110-not-114) | architecture.md OQ14, progression.md OQ1, scoring.md OQ7, constants.md OQ8, structs.md §2 | — | — |
| [D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant) | house-format.md PN2, original-houses.md PN1, progression.md PN3, object-taxonomy.md PN3 | — | — |
| [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length) | house-format.md §12.4, object-taxonomy.md OQ1, interactions.md OQ5, editor.md Q4, constants.md OQ9, scoring.md OQ24 | object-taxonomy.md PN9 | — |
| [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) | original-houses.md OQ1, progression.md OQ4 + OQ5 + PN6, scoring.md OQ4 + OQ5 + OQ25, constants.md OQ5, house-format.md PN19 + PN20, editor.md Q10 | house-format.md OQ8, original-houses.md OQ2, object-taxonomy.md PN9 | — |
| [D5](#d5--the-bound-on-housetypenrooms) | constants.md OQ14, constants.md OQ1 | — | — |
| [D6](#d6--the-retail-build-configuration-and-the-version-identity) | architecture.md OQ1, progression.md OQ2, constants.md OQ12, resource-fork.md OQ5 + OQ6, scoring.md OQ12, input.md PN16 | original-houses.md OQ8 | architecture.md PN13 |
| [D7](#d7--demo-playback-termination) | architecture.md OQ11 + OQ12, input.md OQ1 + OQ6 + PN17 | — | architecture.md OQ10 |

### 9.3 Items in the same neighbourhood that this document does *not* close

Listed so nobody expects them here:

| Document | Item | Why not |
|---|---|---|
| house-format.md | OQ1 "No version-1 house exists in the corpus" | needs a pre-2.0 house file; none exists in this repository (all 22 have `version == 0x0200`). `ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746-820`) is the only description |
| editor.md | Q9 "What was `kNewHouseVersion` (`0x0300`) for?" | `0x0300` is only ever compared against (`HouseIO.c:395`), never written. No 3.0 format exists to describe |
| resource-fork.md | OQ7 "`'PICT'` 10000" | a graphics question, unrelated to the six items |
| scoring.md | OQ1 "Was `houseIsReadOnly` ever true in a shipping build?" | `IsFileReadOnly` is a stub; behaviour question, not format |
| constants.md | OQ11 "Is `evenFrame = true` at `Dynamics3.c:474`/`:524` truly …" | physics/determinism question |
| original-houses.md | OQ4 "`Fun House`'s room 29 `where = 5812`" | link-encoding data question; see original-houses.md itself |

## Open questions

Genuinely unanswerable from the material in this repository. Each states what
evidence exists and what artefact would settle it.

1. **What `houseType.unusedShort` was *intended* for.** [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate)
   settles what it *holds* (residue; 10 of 22 houses non-zero) and what a port must
   do (preserve verbatim). It does not recover the intent, because no code in the
   tree reads or writes it and `InitializeEmptyHouse` (`GliderPRO/Sources/House.c:108-161`)
   does not initialise it. Its position — immediately after `version` and before
   `timeStamp` — is where a 68k compiler would want padding if `version` had once
   been a `Byte`, or where a second version/subversion field would sit. **Would be
   settled by:** an earlier revision of `GliderStructs.h`, or a Glider 3.0 (pre-PRO)
   house file.
2. **Whether `savedGame.roomNumber` is stale in the two houses that carry a real
   save.** `CompressHouse` (`GliderPRO/Sources/HouseLegal.c:688-734`) renumbers
   rooms and does not patch `savedGame.roomNumber`, and `SaveGame`
   (`GliderPRO/Sources/SavedGames.c:329`) stores `thisRoomNumber` — a *pre*-compress
   index. `ImagineHouse PRO II` and `Titanic` are the only houses with
   `hasGame == 1`. The question is unobservable in practice because
   `QueryResumeGame` (`GliderPRO/Sources/Menu.c:710`) has **zero call sites** and
   `OpenSavedGame` returns `false` at `SavedGames.c:169`, so nothing ever reads the
   field. **Would be settled by:** replaying those two saves in a build with the
   resume path wired up.
3. **Which of the 22 houses shipped on the retail disk.** [D6](#d6--the-retail-build-configuration-and-the-version-identity)
   establishes that "retail 1.0.4" is a misnomer (retail was 1.1.2, © 1994-95), and
   [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate)
   establishes that `CD Demo House` / `Slumberland` and `Nemo's Market` / `SpacePods`
   share a byte-identical stale `savedGame` block — evidence that each pair descends
   from a common ancestor file, not that the files themselves are duplicates (they
   differ in size: 72554 vs 134150, 44018 vs 140762) — and that `Sampler` and `Slumberland` were both
   *touched* on 2000-05-11 (house `timeStamp` 0x3540BFE8 | 0x80000000 = 3040919528 →
   2000-05-11 19:52:08; `Slumberland.highScores.timeStamps[0]` = 3040890608 →
   2000-05-11 11:50:08 while its house stamp stays at 1995-08-13, exactly as
   `WriteHouse`'s `if (fileDirty)` gate at `HouseIO.c:475` predicts). That is
   provenance for 4 files, not 22. **Would be settled by:** the retail CD-ROM's
   file listing.
4. **Whether the `Demo House` in this repository is bit-for-bit the file the demo
   was recorded against.** It is certainly the right file by every available test:
   `GliderPRO/Houses/Demo House.binhex` is 16526 bytes with 45 rooms,
   `firstRoom == 0` and bit 0 of `timeStamp` set (locked), satisfying all **four**
   `COMPILEDEMO` gates in `ReadHouse`
   (`HouseIO.c:349`, `:382`, `:403-406`, `:412`) at once. But its house `timeStamp` is
   743682113 → **1995-08-13 13:36:01**, and `'demo'` 128 is embedded in a resource
   fork that also carries a `DITL` updated in **2000**
   ([D6](#d6--the-retail-build-configuration-and-the-version-identity)), so it is
   not provable that the recording predates the last edit of the house. If the
   house was edited after the demo was recorded, the demo will diverge partway
   through even in a perfect port. **Would be settled by:** replaying it. If the
   glider completes a plausible route and dies naturally, the pairing is right; if
   it walks into a wall at a specific frame, the house was edited afterwards. This
   is a test a port can run on day one, and it is the highest-value one available.
5. **Which compiler and project settings produced the `Sampler` file's 868-byte
   header.** [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length)
   shows the 2 trailing bytes are consistent with a `houseType` compiled at natural
   alignment while every other house was written by an `align=mac68k` build, and the
   `d[1562] == d[1560] && d[1563] == d[1561]` prediction held (`01 01 01 01`). But
   both predicted bytes are `0x01`, so the test corroborates rather than proves.
   **Would be settled by:** any second house file whose length is `868 + 348 * n`,
   or the CodeWarrior project file (`Glider PRO.mcp` / `.µ`), which is not in the
   GPL drop.
6. **Why `roomType.unusedByte` needed force-zeroing in two places.**
   `CreateNewRoom` sets it (`GliderPRO/Sources/Room.c:173`) *and* `CheckRoomNameLength`
   sets it again (`GliderPRO/Sources/HouseLegal.c:868`) — belt-and-braces against
   something. All 4070 rooms in the corpus have it 0, so whatever wrote garbage
   there was fixed before these files were saved. **Would be settled by:** a house
   file written by an older editor build.
7. **Whether `'demo'` 128's `padding` residue encodes anything recoverable about the
   1994 machine.** 109 distinct values, 407 zeros, in a `NewPtr` buffer of 12000
   bytes (`StructuresInit2.c:282`). The distribution might identify what was in the
   heap, but there is no way to validate a guess. **Would be settled by:** nothing
   available; treat as noise and preserve.
8. **The exact frame at which the demo glider dies.** [D7](#d7--demo-playback-termination)
   establishes that the last record is at frame 3414 (113.8 s at 30 fps) and that the
   glider then flies uncontrolled until `mortals < 0`. The actual death frame is a
   function of the full physics pipeline and cannot be computed from the demo
   resource. **Would be settled by:** replaying `'demo'` 128 against
   `GliderPRO/Houses/Demo House.binhex` (both are in this repository) and comparing
   with a recording of the original binary running under an emulator. Nothing
   blocks this test but the port itself.

## Porting notes

1. **These seven decisions are load-bearing in this order.** [D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant)
   (866/348) must be right before anything else can be; [D5](#d5--the-bound-on-housetypenrooms)
   (the `nRooms` clamp) and [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length)
   (trailing slack) together define "how many rooms are in this file";
   [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate)
   (residue preservation) defines whether your writer can round-trip;
   [D1](#d1--sizeofgame2type-is-110-not-114) matters only for a file format nothing
   ever produced; [D6](#d6--the-retail-build-configuration-and-the-version-identity)
   and [D7](#d7--demo-playback-termination) are behaviour, not format.
2. **Write the three sizes as named constants and assert them at `init()`.**

   ```go
   const (
       SizeHouse = 866 // sizeof(houseType) under align=mac68k, header only
       SizeRoom  = 348 // sizeof(roomType)
       SizeObject = 12 // sizeof(objectType)
       SizeScores = 292 // sizeof(scoresType)
       SizeGame   = 40  // sizeof(gameType)
       SizeSavedRoom = 292 // sizeof(savedRoom)
       SizeGame2  = 110 // sizeof(game2Type) header; NOT 114
       SizeDemoRecord = 6 // sizeof(demoType); NOT 8
       OffHouseRooms = SizeHouse // rooms[] begins immediately after the header
   )
   ```

   Then `func init() { if unsafe.Sizeof(houseHeader{}) != SizeHouse { panic(...) } }`
   is worthless in Go, because Go's own alignment rules will not reproduce 866.
   **Do not use `encoding/binary.Read` on a Go struct for these records.** Write
   explicit field-by-field readers over a `[]byte` with hard-coded offsets. That is
   the only way to guarantee 866, and it is also faster.
3. **Every on-disk integer is big-endian.** Use `binary.BigEndian` explicitly at
   every site. `long` is 4 bytes, `short` is 2, `Boolean`/`Byte` is 1. There are no
   8-byte integers and no floats anywhere in the house format.
4. **`Point` is `{v, h}` — vertical first.** This is the single most common porting
   error in this codebase, and it is silent: a swapped `Point` still parses.
   `Rect` is `{top, left, bottom, right}`.
5. **Keep the raw bytes.** The pattern that satisfies
   [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length)
   and [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate)
   simultaneously is: read the whole data fork into one `[]byte`, keep it, decode
   *views* over it, and on write mutate the original buffer field-by-field rather
   than re-serialising from the decoded model. That reproduces the original exactly,
   because the original literally does this — `thisHouse` is one `Handle` that
   `WriteHouse` dumps with a single `FSWrite` of `GetHandleSize` bytes
   (`GliderPRO/Sources/HouseIO.c:473`, `:491`).
6. **Pascal-string tails are data.** `PasStringCopy`
   (`GliderPRO/Sources/StringUtils.c:18-25`) copies exactly `1 + length` bytes and
   never pads, so 4008 of 4070 room names, 22 of 22 banners, 22 of 22 trailers and
   183 of 220 high-score name slots carry residue past their length byte. Model a
   fixed Pascal string as `[N]byte` with `Text()`/`SetText()` accessors, never as a
   Go `string`.
7. **MacRoman, not Latin-1 and not UTF-8.** `0xAA` is `™` (U+2122) and `0xA9` is `©`
   (U+00A9) — both appear in the version strings ([D6](#d6--the-retail-build-configuration-and-the-version-identity)).
   Line breaks inside banners and trailers are bare `\r` (0x0D).
8. **The Mac epoch is 1904-01-01 00:00:00 local time**, and `timeStamp` is a
   32-bit *unsigned* seconds count. Bit 0 of `houseType.timeStamp` is the
   **house-lock flag**, not part of the time (`HouseIO.c:402`), and `WriteHouse`
   masks bit 31 off (`:478`) — so all 22 stored house stamps have bit 31 clear while
   every non-zero `highScores.timeStamps[0]` has it set, because `TestHighScore`
   writes that one unmasked (`GliderPRO/Sources/HighScores.c:411`).
9. **Validate before you index, everywhere.** The original has four indexing bugs
   in the areas this document covers, all benign on a Mac heap and all fatal in Go:
   `pidgeonHoles[bitPlace]` after a no-op `DebugStr` (`HouseLegal.c:667`),
   `rooms[-1]` when `nRooms == 0` (`HouseLegal.c:755`), `demoData[1117]`
   (`Input.c:224`), and `theHousesSpecs[-1]` when no `Demo House` exists
   (`Events.c:536`). Fix all four; document each fix in a comment citing the
   original line, so a future reader does not "restore fidelity" by removing the
   guard.
10. **Do not port the copy protection.** `GliderPRO/Sources/Validate.c` exists but
    `COMPILENOCP` is defined (`GliderDefines.h:14`) and the tree does not compile
    without it ([D6](#d6--the-retail-build-configuration-and-the-version-identity)).
    Its absence also means your preferences record is 226 bytes, not retail's 234 —
    which is fine, because a Go port should not be writing a byte-compatible
    `prefsInfo` at all. Use JSON or TOML for preferences; nothing reads the Mac
    format but the Mac.
11. **Make `BUILD_ARCADE_VERSION` a runtime bool, default `false`.** Go has no
    `#if`, and a `//go:build arcade` tag means the arcade path is never compiled in
    CI. The two configurations differ only in input routing (`Events.c:191-208`,
    `Input.c:192-208`) and two repaint blocks (`Play.c:512-541`, `:556-592`), so one
    boolean covers it.
12. **The demo is the best determinism test you have, and it needs one added
    bound.** Transcribe `GetDemoInput` exactly (`Input.c:186-277`) except for
    `demoIndex < len(demo)`. Keep `==` rather than `>=`; keep the single `if`
    rather than a loop; keep the burning-mode early return that stalls the stream.
    (If you ever build a *recorder*, note that the original's `GetInput` can emit
    more than one record for the same frame — the battery and band branches are
    independent `if`s, not alternatives ([D7](#d7--demo-playback-termination) ev. 5)
    — so a re-recorded stream is only replayable by this playback loop if you
    coalesce to one record per frame.)
    Use key codes `0 = right, 1 = left, 2 = battery, 3 = band` and put a comment
    next to them saying the original's comments are swapped. Both halves of the
    test are in this repository: the stream is `data 'demo' (128)` in
    `GliderPRO/Glider PRO.r`, and the house is `GliderPRO/Houses/Demo House.binhex`
    (16526 bytes, 45 rooms, `firstRoom = 0` — the exact file
    `ReadHouse`'s `COMPILEDEMO` literals describe).
13. **Frame timing: 30 fps, no catch-up.** `kTicksPerFrame` 2 over a 60.15 Hz Mac
    tick (`GliderDefines.h:533`, `GliderPRO/Sources/Render.c:662`/`:665`/`:690`).
    `gameFrame++` happens at the **top** of the loop (`Play.c:434`), so the first
    frame is 1, not 0 — which is why the demo's first record is at frame 46 and not
    45. Off-by-one here desynchronises the entire demo.
14. **Round-trip test, concretely.** For each of the 22 files in
    `GliderPRO/Houses/`: BinHex-decode (verify the CRC-16 with polynomial
    `x^16 + x^12 + x^5 + 1` — all 22 pass; `tools/probe_house.py`'s
    `binhex_decode` defaults to `check_crc=False`, so pass `True` explicitly),
    parse, re-serialise, and require `bytes.Equal`. This test catches every
    [D2](#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant),
    [D3](#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length)
    and [D4](#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate)
    error at once. Expected first failures if you get it wrong: 21 files off by
    the last 348 bytes (D2), 1 file off by 2 bytes (D3), 22 files off by hundreds
    of scattered bytes (D4).
15. **Version strings:** product **1.1.2**, `© 1994-95 Casady & Greene, Inc.`,
    long form `Glider PRO™ 1.1.2`. "1.0.4" is a stale comment at
    `GliderPRO/Sources/Main.c:3` and appears nowhere in the resource fork. The
    About dialog's own `DITL` says `© 1994-2000`, which is how you know this source
    drop postdates the retail build.
