# Glider PRO resource fork: complete inventory and the `Glider PRO.r` Rez dump

## Scope

This document specifies **the entire resource fork of the Glider PRO application**, as
preserved in the DeRez text dump `GliderPRO/Glider PRO.r`, and maps every resource to the
code that consumes it.

It covers three things:

1. **The Rez text format** used by `Glider PRO.r` — enough detail to write a byte-exact
   extractor. A reference extractor is shipped at
   `tools/probe_rez.py`.
2. **A complete, verified inventory** — all 35 resource types, all 538 resources, all
   3,168,309 payload bytes, with per-type ID lists and per-resource sizes.
3. **The on-disk binary layout of every resource type present**, verified by parsing the
   bytes with `python3`, plus the cross-reference from each ID to the `#define` constant
   and source line that uses it.

Out of scope (owned by sibling documents): the *house file* format
(`'gliH'` files, `Houses/*.binhex`), the interpretation of PICT opcodes, and the gameplay
semantics of the objects whose art is catalogued here. Where a resource type lives only in
house files and never in the application (`'Date'`, `'bnds'`), this document says so and
points at the load site, but does not specify the house format.

Every ID range, count, byte total, header field and offset in this document was produced by
actually parsing the file. Where a number is quoted it is an **observed** number, not a
number derived from documentation.

## Sources read

Read in full (CR→LF converted copies; line numbers below are 1-based in the converted file
and are identical to the original's logical line numbers):

| File | Why |
| --- | --- |
| `GliderPRO/Glider PRO.r` | the resource fork itself (15,475,666 bytes, 199,843 LF lines) |
| `GliderPRO/Sources/Utilities.c` | `RedAlert`, `LoadGraphic`, `LoadScaledGraphic`, `LargeIconPlot`, `DrawCIcon`, `CreateOffScreenGWorld`, volume conversion |
| `GliderPRO/Sources/StructuresInit.c` | sprite-sheet GWorlds and the `+1000` mask convention |
| `GliderPRO/Sources/StructuresInit2.c` | clutter/support/angel GWorlds, `'demo'` load, `InitSrcRects` |
| `GliderPRO/Sources/AnimCursor.c` | `'acur'` / `'CURS'` / `'crsr'` animated cursor |
| `GliderPRO/Sources/About.c` | `'vers'` parsing, About dialog art |
| `GliderPRO/Sources/DialogUtils.c` | `'DLOG'`/`'ALRT'` handle poking, `LoadDialogPICT` |
| `GliderPRO/Headers/GliderDefines.h` | 625 lines of constants: sound indices, backgrounds, PICT IDs, menu IDs |
| `GliderPRO/Headers/Externs.h` | prototypes for every loader named here |

Consulted at specific line numbers (each citation below is exact):

`Sources/AppleEvents.c`, `Sources/Banner.c`, `Sources/Coordinates.c`,
`Sources/DebugUtilities.c`, `Sources/Environ.c`, `Sources/Events.c`, `Sources/FileError.c`,
`Sources/GameOver.c`, `Sources/HighScores.c`, `Sources/House.c`, `Sources/HouseInfo.c`,
`Sources/HouseIO.c`, `Sources/HouseLegal.c`, `Sources/Input.c`, `Sources/InterfaceInit.c`,
`Sources/Link.c`, `Sources/MainWindow.c`, `Sources/Map.c`, `Sources/Marquee.c`,
`Sources/Menu.c`, `Sources/Music.c`, `Sources/ObjectAdd.c`, `Sources/ObjectDraw.c`,
`Sources/ObjectDraw2.c`, `Sources/ObjectDrawAll.c`, `Sources/ObjectEdit.c`,
`Sources/ObjectInfo.c`, `Sources/ObjectRects.c`, `Sources/Objects.c`, `Sources/Play.c`,
`Sources/Player.c`, `Sources/Prefs.c`, `Sources/Room.c`, `Sources/RoomGraphics.c`,
`Sources/RoomInfo.c`, `Sources/SavedGames.c`, `Sources/Scoreboard.c`, `Sources/Scrap.c`,
`Sources/SelectHouse.c`, `Sources/Settings.c`, `Sources/Sound.c`, `Sources/StringUtils.c`,
`Sources/Tools.c`, `Sources/Validate.c`, `Sources/WindowUtils.c`,
`Headers/GliderStructs.h`, `Headers/GliderProtos.h`, `Headers/GliderVars.h`,
`Headers/Marquee.h`, `Headers/Objects.h`.

Method note: GNU `grep` treats the original CR-terminated MacRoman `.c` files as binary and
reports one enormous line, so every citation in this document was taken from a byte-for-byte
ASCII-safe mirror built with `LC_ALL=C tr '\200-\377' '?'` over the CR→LF copies. That
substitution is 1 byte → 1 byte, so **line numbers are preserved exactly** and are valid
against the originals.

---

# Part 1 — The Rez text format

## 1.1 What the file is

`Glider PRO.r` is the output of Apple's `DeRez` tool run over the shipped application's
resource fork. DeRez emits, for each resource it cannot describe with a type template, a
*raw data declaration*:

```
data 'TYPE' (ID [, "name"] [, attribute...]) {
	$"48656C 6C6F"                                        /* Hello */
};
```

The whole of `Glider PRO.r` consists of **nothing but** these declarations. There is no
`#include`, no `resource` statement, no `read`, no `type` template, no preprocessor
directive, no `/* */` comment outside a data body.

Because raw `data` blocks reproduce the resource payload verbatim, the file is a
**lossless, byte-exact image of every resource's data**, plus the resource's type, ID,
optional name, and any set attribute keyword. It does *not* preserve resource-map ordering
inside a type, resource-fork attributes at file level, or the `resFileAttrs` word.

## 1.2 Verified file-level metrics

Counted by iterating over the raw bytes of `GliderPRO/Glider PRO.r`:

| Metric | Observed |
| --- | --- |
| File size | 15,475,666 bytes |
| Line terminator | LF (`0x0A`) only; **zero** `0x0D` bytes in the file |
| Total lines | 199,843 |
| `data ...` header lines | 538 |
| Hex body lines | 198,229 |
| `};` terminator lines | 538 |
| Blank separator lines | 538 |
| Any other line | 0 |
| Longest raw line (incl. LF) | 78 bytes |
| Resources | 538 |
| Distinct resource types | 35 |
| Total payload bytes | 3,168,309 |
| Resources carrying a name | 215 |
| Resources marked `purgeable` | 73 |
| Duplicate `(type, id)` pairs | 0 |
| Empty (0-byte) resources | 0 |
| ID range | 0 … 10000 (all non-negative, all decimal) |

`538 + 198229 + 538 + 538 = 199843`, so the line classification is exhaustive.

> **Caution for `file(1)` and text tools.** The file is MacRoman, not UTF-8, and it contains
> `0x85` bytes inside the DeRez comment column (MacRoman `à`). `file(1)` reports
> *"NEL line terminators"* for this file, which is wrong. Always open it in **binary** mode
> and split on `\n` only. `probe_rez.py` does this
> (`tools/probe_rez.py:145`).

## 1.3 The `data` statement grammar as it actually appears

Every header line matches, in column 0:

```
data '<4 chars>' ( <id> [ , "<name>" ] [ , <attr> ]* ) {
```

Observed specifics:

* The type is always exactly 4 characters between single quotes. **The trailing space in
  `'snd '` is significant** — the type is `s`, `n`, `d`, `0x20`.
* The ID is a plain non-negative decimal integer. No negative IDs, no `0x` IDs, no
  `$`-prefixed IDs anywhere in this file.
* The name, when present, is a double-quoted MacRoman string. **It may contain
  parentheses.** The only two such cases are:

  ```
  data 'WDEF' (128, "Infinity Windoid 2.6", purgeable) {
  data 'WDEF' (129, "Infinity Windoid 2.6 (grow)", purgeable) {
  ```

  This is why a `\(([^)]*)\)` regex is *wrong* for parsing the header — it truncates at the
  `)` inside `(grow)`. `probe_rez.py` walks the characters, tracking quote state and paren
  depth (`tools/probe_rez.py:78`).
* The only attribute keyword that ever appears is `purgeable`. (The literal word
  `resource` also appears in the file — but inside the *name* of `data 'ozm5' (0, "Owner
  resource")`, not as a keyword.)
* The opening brace is on the same line as the header, separated by a single space.
* The terminator is `};` in column 0, followed by exactly one blank line.

## 1.4 Body line format

Every one of the 198,229 body lines matches:

```
\t $" HHHH HHHH HHHH HHHH "   <spaces>   /* <MacRoman rendering> */
```

Concretely:

* Exactly one leading **TAB** (`0x09`).
* `$"`, then upper-case hex digits grouped in **runs of 4 (2 bytes)** separated by single
  spaces, then `"`.
* Bytes per line, observed histogram (`bytes: count`):

  | Bytes on line | Lines |
  | --- | --- |
  | 16 | 197,806 |
  | 10 | 107 |
  | 12 | 52 |
  | 8 | 41 |
  | 4 | 37 |
  | 14 | 33 |
  | 6 | 28 |
  | 5 | 27 |
  | 2 | 58 |
  | 7 | 12 |
  | 11 | 8 |
  | 9 | 7 |
  | 13 | 7 |
  | 15 | 2 |
  | 3 | 1 |
  | 1 | 3 |

  So the canonical line carries 16 bytes as `"XXXX XXXX XXXX XXXX"`; the final line of a
  resource carries the remainder, and the last group may be 1 byte (2 nibbles). The nibble
  count on a line is always even.
* **Every** body line (198,229 of 198,229) has a trailing ` /* ... */` comment. This is
  DeRez annotation: the MacRoman rendering of that line's bytes with non-printables shown
  as `.`. It carries **no data** and must be discarded. It can itself contain `"` , `$`,
  `{`, `}` and `*/`-like sequences, so a parser must consume the `$"..."` literal first and
  ignore everything after the closing quote.

Worked example, the first resource in the file:

```
     1 |data 'STR#' (160, "Prefs") {
     2 |	$"0001 0B50 7265 6665 7265 6E63 6573"                 /* ...Preferences */
     3 |};
     4 |
```

That decodes to 13 bytes `00 01 0B 50 72 65 66 65 72 65 6E 63 65 73` — a string list with
count 1 and one Pascal string of length 0x0B, `"Preferences"`.

## 1.5 Physical layout of the file

Each of the 35 types occupies exactly **one contiguous run** of `data` blocks. Verified run
order and starting line:

| Start line | Type | Count |
| --- | --- | --- |
| 1 | `'STR#'` | 10 |
| 550 | `'ICON'` | 35 |
| 935 | `'cicn'` | 44 |
| 3881 | `'acur'` | 1 |
| 3888 | `'CURS'` | 16 |
| 4016 | `'crsr'` | 12 |
| 4279 | `'DITL'` | 54 |
| 5187 | `'ALRT'` | 26 |
| 5291 | `'MENU'` | 6 |
| 5372 | `'BNDL'` | 1 |
| 5380 | `'ozm5'` | 1 |
| 5385 | `'FREF'` | 6 |
| 5409 | `'ICN#'` | 6 |
| 5523 | `'icl8'` | 6 |
| 5925 | `'icl4'` | 6 |
| 6135 | `'vers'` | 2 |
| 6148 | `'DLOG'` | 28 |
| 6288 | `'dctb'` | 20 |
| 6408 | `'PICT'` | 152 |
| 126905 | `'WDEF'` | 2 |
| 127657 | `'snd '` | 70 |
| 198479 | `'mctb'` | 5 |
| 198553 | `'CDEF'` | 1 |
| 198930 | `'CNTL'` | 5 |
| 198955 | `'PAT#'` | 1 |
| 198962 | `'ics8'` | 4 |
| 199038 | `'ics4'` | 4 |
| 199082 | `'ics#'` | 4 |
| 199110 | `'WIND'` | 3 |
| 199125 | `'clut'` | 2 |
| 199389 | `'demo'` | 1 |
| 199811 | `'wctb'` | 1 |
| 199817 | `'ictb'` | 1 |
| 199823 | `'cctb'` | 1 |
| 199829 | `'DLGX'` | 1 |

Within a run, IDs are **not** sorted. E.g. `'STR#'` appears in the order
160, 170, 171, 128, 1006, 1007, 129, 150, 140, 1005. Do not rely on ordering; index by
`(type, id)`.

## 1.6 Reference extractor

`tools/probe_rez.py` (330 lines, Python 3, stdlib only).
It defaults to `../GliderPRO/Glider PRO.r` relative to the script
(`tools/probe_rez.py:48`), so it runs with no arguments.

```
probe_rez.py inventory  [<file>]        # type -> count, bytes, compressed id ranges
probe_rez.py ids <TYPE> [<file>]        # every id of one type, with size, line, name, attrs
probe_rez.py sizes [<file>]             # every resource, largest first
probe_rez.py dump <TYPE> <ID> [<file>]  # hexdump one resource
probe_rez.py extract <outdir> [<file>]  # <outdir>/<TYPE>/<ID>.bin for all 538
probe_rez.py verify [<file>]            # grammar self-checks
```

Key API for reuse:

* `parse(path) -> generator of Resource` — streams the file, yields in file order.
* `class Resource` with `__slots__ = ("type", "id", "name", "attrs", "data", "line")`;
  `type` is 4 raw bytes (`b'snd '`), `name` is raw MacRoman bytes or `None`, `attrs` is a
  list of raw byte strings, `data` is the payload, `line` is the 1-based header line.
* `ranges(ids) -> "128-131, 149-160"` — compresses a sorted ID list.

Its `verify` command asserts, and the shipped file satisfies: all `(type, id)` unique, all
types exactly 4 bytes, no empty payloads, only `purgeable` as an attribute, IDs 0..10000.

Pseudocode of the parse loop, matching `tools/probe_rez.py:139`:

1. Open the file in binary mode (`"rb"`).
2. For each raw line, strip trailing `\r\n`; skip if empty.
3. If the line starts with `data ` — match `^data '(....)' \(`, then hand-parse the rest of
   the header (see below). Start a new body buffer.
4. Else if the line starts with `};` — emit the accumulated resource, clear state.
5. Else match `^\t\$"([0-9A-Fa-f ]*)"`; take group 1, delete spaces, assert an even nibble
   count, `bytes.fromhex`, append to the body buffer.
6. Anything else is a hard error (never happens on the shipped file).

Header parse (`tools/probe_rez.py:78`), which must not use a naive regex:

1. `depth = 1`, `cur = ""`, `fields = []`.
2. Scan characters. On `"`, copy the whole quoted string including `\"` / `\\` escapes into
   `cur` and continue (so parens inside it are inert).
3. On `(` increment `depth` and append; on `)` decrement `depth` — if it hits 0, push `cur`
   and stop; otherwise append.
4. On `,` at depth 1, push `cur` and reset it.
5. `fields[0]` is the ID (`int(f, 0)`); any later field starting with `"` is the name (strip
   the quotes, unescape `\"` and `\\`); every other later field is an attribute keyword.

## 1.7 What DeRez did **not** emit — and where the code looks for it

The dump is complete for the types it contains, but four resource categories referenced by
the code are absent. All four are legitimately absent, for different reasons.

| Looked-for | Where | Why it is not in this fork |
| --- | --- | --- |
| `'SIZE'` 0, 1, −1 | `Sources/Environ.c:710`, `Sources/Environ.c:719` | The Finder-visible memory-partition resource. `SetAppMemorySize()` rewrites it, but its only call site is **commented out** (`Sources/Environ.c:687`), so the code is dead. DeRez omitted it. A Go port has no equivalent. |
| `'STR '` −16096 | `Sources/StringUtils.c:304-308` (`#define kChooserStringID -16096`, `GetString(kChooserStringID)`) | A **System file** resource (the Chooser's "Owner Name"). Never in the app. A Go port must read the machine/user name from the host OS. |
| `'DLOG'` `sfPutDialogID` / `sfGetDialogID` | `Sources/DialogUtils.c:51`, `Sources/DialogUtils.c:85` | Standard-File's own dialogs, in the System file. Both call sites are inside `/* ... */` blocks — dead. |
| `'Date'`, `'bnds'` | `Sources/Map.c:172`, `Sources/Room.c:272`, `Sources/Room.c:942`, `Sources/RoomGraphics.c:142`, `Sources/RoomInfo.c:845` | These live in **house files** (`'gliH'`), whose resource fork is opened separately (`Sources/SelectHouse.c:102`, `Sources/HouseIO.c:575`). See §3.1.9. |

Additionally, the following types named in the assignment brief are **not present** and
never referenced: `'CICN'` (upper case — the real type is lower-case `'cicn'`), `'ppat'`,
`'sfnt'`, `'PAT '` (singular), `'ICN8'`, `'kcs#'`. And the app's own fork contains no
`'CODE'` resources in the dump either — DeRez skipped them; the only 68k code that *did*
survive is the third-party `'WDEF'`/`'CDEF'` (§3.11).

---

# Part 2 — Complete inventory

## 2.1 Master table

Produced by `python3 tools/probe_rez.py inventory`. `bytes` is the sum of payload lengths
(the resource data only — no resource-map overhead).

| Type | Count | Bytes | IDs |
| --- | ---: | ---: | --- |
| `'PICT'` | 152 | 1,919,598 | 150-151, 153, 1000-1023, 1202, 1211, 1216-1217, 1988-2017, 3903-3904, 3912-3915, 3921, 3927, 3957-4018, 4998-5002, 5004-5018, 10000 |
| `'snd '` | 70 | 1,129,344 | 1000-1062, 2000-2006 |
| `'DITL'` | 54 | 11,576 | 130, 140, 150, 160, 170, 180-181, 1000-1046 |
| `'cicn'` | 44 | 44,584 | 130, 140, 150, 160, 170, 180, 900, 910, 1000-1001, 1004-1006, 1008, 1010-1017, 1020-1022, 1030-1034, 1040-1043, 1050-1053, 1060, 1070-1073, 2000 |
| `'ICON'` | 35 | 4,480 | 130, 140, 150, 160, 170, 180, 900, 910, 1000-1001, 1004-1006, 1008, 1010-1013, 1020-1022, 1030-1032, 1040-1043, 1050-1051, 1060, 1070-1073 |
| `'DLOG'` | 28 | 591 | 150, 1000-1001, 1003, 1007, 1010-1027, 1033-1035, 1043, 1045 |
| `'ALRT'` | 26 | 312 | 130, 140, 160, 170, 180-181, 1002, 1004-1006, 1008-1009, 1028-1032, 1036-1042, 1044, 1046 |
| `'dctb'` | 20 | 960 | 150, 1000-1001, 1003, 1007, 1010-1015, 1017-1019, 1022-1024, 1027, 1033-1034 |
| `'CURS'` | 16 | 1,088 | 128-131, 149-160 |
| `'crsr'` | 12 | 3,504 | 149-160 |
| `'STR#'` | 10 | 8,229 | 128-129, 140, 150, 160, 170-171, 1005-1007 |
| `'FREF'` | 6 | 42 | 128-133 |
| `'ICN#'` | 6 | 1,536 | 128-133 |
| `'MENU'` | 6 | 983 | 128-131, 140-141 |
| `'icl4'` | 6 | 3,072 | 128-133 |
| `'icl8'` | 6 | 6,144 | 128-133 |
| `'CNTL'` | 5 | 125 | 128-132 |
| `'mctb'` | 5 | 910 | 128-132 |
| `'ics#'` | 4 | 256 | 128-130, 133 |
| `'ics4'` | 4 | 512 | 128-130, 133 |
| `'ics8'` | 4 | 1,024 | 128-130, 133 |
| `'WIND'` | 3 | 89 | 128-130 |
| `'WDEF'` | 2 | 11,928 | 128-129 |
| `'clut'` | 2 | 4,112 | 128-129 |
| `'vers'` | 2 | 106 | 1-2 |
| `'BNDL'` | 1 | 68 | 128 |
| `'CDEF'` | 1 | 5,978 | 128 |
| `'DLGX'` | 1 | 190 | 150 |
| `'PAT#'` | 1 | 58 | 128 |
| `'acur'` | 1 | 52 | 128 |
| `'cctb'` | 1 | 40 | 132 |
| `'demo'` | 1 | 6,702 | 128 |
| `'ictb'` | 1 | 36 | 150 |
| `'ozm5'` | 1 | 32 | 0 |
| `'wctb'` | 1 | 48 | 130 |
| **TOTAL** | **538** | **3,168,309** | 35 types |

Two types dominate: `'PICT'` (60.6% of bytes) and `'snd '` (35.6%). Together they are 96.2%
of the fork. Everything else totals 119,367 bytes.

## 2.2 Ten largest individual resources

| Type | ID | Bytes | What |
| --- | ---: | ---: | --- |
| `'snd '` | 2004 | 194,070 | music piece 4 (the only double-length track) |
| `'PICT'` | 1000 | 108,482 | `kSplash8BitPICT`, 640x460 title screen |
| `'PICT'` | 2006 | 105,140 | room background 6, "Swinger's Room" |
| `'snd '` | 2006 | 97,376 | music piece 6 |
| `'snd '` | 2003 | 97,326 | music piece 3 |
| `'snd '` | 2001 | 97,144 | music piece 1 |
| `'snd '` | 2000 | 97,076 | music piece 0 |
| `'snd '` | 2002 | 97,074 | music piece 2 |
| `'snd '` | 2005 | 96,930 | music piece 5 |
| `'PICT'` | 2001 | 89,338 | room background 1, "Paneled Room" |

## 2.3 `purgeable` census (73 resources)

The Resource Manager may throw a purgeable resource out of memory when it needs space, so
the code must re-`GetResource` it every time. Everything else in this fork is
non-purgeable. Observed set:

| Type | Purgeable IDs | Count |
| --- | --- | ---: |
| `'snd '` | 1000-1013, 1018-1019, 1021, 1023-1025, 1029-1035, 1038-1039, 1043-1051, 1054-1062, 2000-2006 | 54 |
| `'crsr'` | 149-160 | 12 |
| `'CURS'` | 128-131 | 4 |
| `'WDEF'` | 128-129 | 2 |
| `'CNTL'` | 132 | 1 |

Note that all seven music tracks and 47 of the 63 sound effects are purgeable, while the
other 16 sound effects (1014-1017, 1020, 1022, 1026-1028, 1036-1037, 1040-1042, 1052-1053)
are not. This is almost certainly an accident of how the resources were pasted together in
ResEdit, not a design decision: the code copies **all** sounds into its own pointers at
startup in `LoadBufferSounds` (`Sources/Sound.c:316-342` — `GetResource` at `:327`, then
`BlockMove` of `GetHandleSize - 20` bytes from offset 20 at `:340`, then `ReleaseResource`), so
purge state is irrelevant at run time. (`Sound.c:501` is the *size probe* in
`SoundBytesNeeded`, not the copy.)

**Go port implication: none.** There is no purgeable memory in a Go port; treat every
resource as always-resident.

## 2.4 Resource names (215 of 538)

Names are pure documentation in this app — the code never calls `GetNamedResource` or
`Get1NamedResource` (verified by grep over all of `Sources/` and `Headers/`: zero hits). The
only place a name is *written* is `AddResource(newResource, 'demo', 128, "\p")`
(`GliderPRO/Sources/DebugUtilities.c:352`), which passes an empty name.

Exactly which types carry names (`named / total`):

| Type | Named | Total |
| --- | ---: | ---: |
| `'snd '` | 70 | 70 |
| `'DITL'` | **53** | 54 |
| `'DLOG'` | 28 | 28 |
| `'ALRT'` | 26 | 26 |
| `'dctb'` | 20 | 20 |
| `'STR#'` | 10 | 10 |
| `'mctb'` | 5 | 5 |
| `'WDEF'` | 2 | 2 |
| `'ozm5'` | 1 | 1 |
| all 26 other types | 0 | 322 |

Total 215. The single unnamed DITL is **1046**. `'PICT'`, `'cicn'`, `'ICON'`, `'CURS'`,
`'crsr'`, `'MENU'`, `'CNTL'`, `'WIND'` and every icon-family type are entirely unnamed, so
the names give no help identifying art assets — those had to be resolved from the code
(§3.1, §3.4).

The names are, however, an authoritative independent confirmation of what each dialog and
sound *is*, and they are the only human-readable label for the sounds. Full lists are in
§3.2.6 (`'snd '`), §3.3.8 (`'DITL'`/`'DLOG'`/`'ALRT'`/`'dctb'`), §3.7.6 (`'mctb'`) and §3.11
(`'WDEF'`).

## 2.5 Set relationships (all verified)

These invariants hold for the shipped fork and are worth asserting in a port's asset loader:

1. `set(DLOG ids) ∪ set(ALRT ids) == set(DITL ids)` — **true**, and
   `set(DLOG) ∩ set(ALRT) == ∅`. So each of the 54 item lists belongs to exactly one dialog
   or one alert, and the DITL ID always equals the DLOG/ALRT ID.
2. `set(ICON) ⊂ set(cicn)`. The 9 cicn-only IDs are 1014, 1015, 1016, 1017, 1033, 1034,
   1052, 1053, 2000 — exactly the nine that are drawn programmatically with `DrawCIcon`
   rather than being placed as a DITL `icon` item. See §3.4.3.
3. `set(dctb) ⊂ set(DLOG)`. The 8 DLOGs with no colour table are 1016, 1020, 1021, 1025,
   1026, 1035, 1043, 1045. All 20 `'dctb'` are byte-identical, so this is cosmetic.
4. `'ics#'`/`'ics4'`/`'ics8'` exist for 128, 129, 130, 133 only. Bundle members 131
   (`'gliG'` saved game) and 132 (`'gliS'` high scores) have a large icon family but **no
   small icon**, so the Finder synthesised their small icons by downscaling.
5. For every sprite sheet 3998-4018 there is a mask at `id + 1000` — **except 4003, which
   has no 5003.** This is not corruption; `Sources/StructuresInit.c:458` loads only
   `kSwitchPictID` for that GWorld and never asks for a mask.
6. `picSize == len(resource) & 0xFFFF` for all 152 PICTs (see §3.1.2).

## 2.6 Negative results

The following do **not** appear anywhere in `Glider PRO.r`:

`'CICN'` (upper case), `'ppat'`, `'sfnt'`, `'FOND'`, `'FONT'`, `'NFNT'`, `'PAT '`,
`'STR '`, `'SIZE'`, `'CODE'`, `'DRVR'`, `'INIT'`, `'ADBS'`, `'MBAR'`, `'dftb'`, `'alrx'`,
`'ALRX'`, `'Date'`, `'bnds'`, `'movv'`, `'moov'`, `'TEXT'`, `'PREC'`, `'kind'`, `'open'`,
`'aete'`, `'aedt'`, `'hfdr'`, `'hdlg'`, `'hwin'`, `'hmnu'`, `'hrct'`, `'hovr'`.

Notably: **there is no `'MBAR'`** — the menu bar is built by hand in `InitializeMenus`
(`Sources/InterfaceInit.c:47-73`) with **four** `GetMenu` calls (`:49` apple, `:55` game,
`:60` options, `:68` house) but only **three** `InsertMenu(…, 0)` calls (`:53`, `:58`, `:63`);
`houseMenu` is only loaded there, and is inserted (and deleted) later by `UpdateMenus` as the
app enters and leaves `kEditMode` (`Sources/Menu.c:249-254`). And **there are
no Balloon Help resources** (`'hdlg'`/`'hmnu'`/…), so all the on-screen help is baked into
DITL static text.

---

# Part 3 — Per-type specification

## 3.1 `'PICT'` — 152 resources, 1,919,598 bytes

### 3.1.1 Groups

Every PICT ID falls into exactly one functional group. Byte totals verified:

| Group | IDs | Count | Bytes |
| --- | --- | ---: | ---: |
| Room backgrounds | 2000-2017 | 18 | 1,037,846 |
| Dialog art (banners, splash, tool palettes) | 1000-1023 | 24 | 315,116 |
| Room-object art | 3957-3997 | 41 | 239,722 |
| Game-over / banner / scoreboard art | 1988-1999 | 12 | 131,633 |
| Animated sprite sheets | 3998-4018 | 21 | 129,585 |
| Sprite-sheet 1-bit masks | 4998-5018 (no 5003) | 20 | 23,673 |
| Background thumbnails for Room Info | 1202, 1211, 1216, 1217 | 4 | 17,310 |
| About-box art | 150, 151, 153 | 3 | 16,458 |
| Custom-picture placeholder | 10000 | 1 | 4,182 |
| Object 1-bit masks | 3903, 3904, 3912-3915, 3921, 3927 | 8 | 4,073 |
| **Total** | | **152** | **1,919,598** |

### 3.1.2 Verified header facts

A classic `'PICT'` resource begins with:

| Offset | Size | Field | Notes |
| ---: | ---: | --- | --- |
| 0 | 2 | `picSize` | big-endian **16-bit** byte count — overflows above 64 KB |
| 2 | 8 | `picFrame` | `Rect` = top, left, bottom, right, each `int16` big-endian |
| 10 | 2 | version opcode | v2: `0x0011` (a 2-byte opcode). v1: the opcode is the **single byte** `0x11`, immediately followed by the 1-byte version number `0x01` |
| 12 | 2 | v2 marker | `0x02FF` only if version 2 |

Note the asymmetry: PICT v1 uses **1-byte** opcodes throughout, so a v1 picture's bytes 10..11
are `11 01` (`versionOp`, then version 1) and real drawing opcodes begin at offset 12. PICT v2
uses **2-byte** opcodes, so bytes 10..13 are `0011 02FF`. Reading bytes 10..11 as a 16-bit word
gives `0x0011` for v2 but `0x1101` for v1; a port must dispatch on the first *byte*.

Observed across all 152 — the census of bytes 10..13 is `0011 02FF` x116, `1101 0100` x4,
`1101 A000` x32:

* **116 are PICT v2** (bytes 10..13 = `0011 02FF`), **36 are PICT v1** (bytes 10..11 = `11 01`).
  All 36 v1 pictures are 1-bit masks or the two tiny About buttons: 150, 151, 1009, 1020,
  1989, 1991, 1998, 3903, 3904, 3912-3915, 3921, 3927, 3998, and all 20 of 4998-5018.
* `picSize == len(resource) & 0xFFFF` for **all 152**, i.e. the field is a truncated length.
  Nine pictures exceed 64 KB and therefore have a *misleading* `picSize`:

  | ID | Real bytes | `picSize` |
  | ---: | ---: | ---: |
  | 1000 | 108,482 | 42,946 |
  | 2001 | 89,338 | 23,802 |
  | 2004 | 83,276 | 17,740 |
  | 2005 | 70,572 | 5,036 |
  | 2006 | 105,140 | 39,604 |
  | 2008 | 82,922 | 17,386 |
  | 2011 | 81,036 | 15,500 |
  | 2014 | 75,132 | 9,596 |
  | 2016 | 76,176 | 10,640 |

  **A Go PICT reader must ignore `picSize` and use the resource's actual length.** Using
  `picSize` would truncate the splash screen and 8 of the 18 room backgrounds mid-image.
* `picFrame` has origin (0,0) for **150 of 152**. The two exceptions are both masks:
  * PICT 5006 `picFrame = (top=0, left=91, bottom=120, right=115)` → 24x120
  * PICT 5010 `picFrame = (top=195, left=0, bottom=230, right=40)` → 40x35

  Both still describe the correct *dimensions*; the origin offset is an artefact of how the
  mask was cropped in the paint program. The origin is discarded before drawing: `LoadGraphic`
  reads `picFrame` and then normalises it with `OffsetRect(&bounds, -bounds.left, -bounds.top)`
  before `DrawPicture` (`GliderPRO/Sources/Utilities.c:327-330`), so every PICT lands at the
  top-left of the current port at its natural size. The destination GWorld's own bounds come
  from the hard-coded `QSetRect` calls in `Sources/StructuresInit.c`, not from the PICT.
  **A naive Go loader that honours `picFrame.origin` will place these two masks off by (91,0)
  and (0,195).**
* One genuine off-by-one in the data: PICT **4005 is 80x269** but its mask **5005 is
  80x268**. Both GWorlds are sized from the same `applianceSrcRect` (0,0,80,269) at
  `Sources/StructuresInit.c:538-545`, so the mask's last row is simply left blank.
* A much larger version of the same slack: PICT **4001 (furniture) is only 64x221**, but
  `furnitureSrcRect` is **0,0,64,278** (`Sources/StructuresInit.c:301`), so the bottom **57
  rows** of both the furniture GWorld and its mask are never painted. Nothing reads them —
  the lowest furniture sub-rect is `deckSrc`, whose bottom is y=183
  (`Sources/StructuresInit.c:331-332`) — so the slack is harmless, but it does mean **GWorld
  height is not derivable from PICT height**: the hard-coded `QSetRect` is the authority.

### 3.1.3 About-box art: 150, 151, 153

| ID | Size | Constant | Cite |
| ---: | --- | --- | --- |
| 150 | 63x63 v1 | `kOkayButtPICTNotHiLit` | `GliderPRO/Sources/About.c:119` |
| 151 | 63x63 v1 | `kOkayButtPICTHiLit` | `GliderPRO/Sources/About.c:97` |
| 153 | 372x100 v2 | (no constant; placed by DITL 150 item 4) | `GliderPRO/Sources/About.c:36` names item 4 `kPictItemMain` |

> **The comments on both `#define`s are swapped relative to the behaviour.**
> `Sources/About.c:97` reads `#define kOkayButtPICTHiLit 151 // res ID of unhilit button PICT`
> inside `HiLiteOkayButton()`, and `Sources/About.c:119` reads
> `#define kOkayButtPICTNotHiLit 150 // res ID of hilit button PICT` inside
> `UnHiLiteOkayButton()`. Trust the *function names*: 151 is drawn when highlighted, 150 when
> not. A port should ship `about_ok_normal = PICT 150`, `about_ok_pressed = PICT 151`.

Both are drawn with `LoadDialogPICT`-style code but from within the modal loop, into the
rect of DITL item 1 at (t=113, l=317, b=176, r=380) — 63x63, exactly the PICT size, so no
scaling occurs here.

### 3.1.4 Dialog art: 1000-1023

Twenty-four pictures. Sixteen are "banner" strips 32 pixels tall that sit at the top of one
or more dialogs; the rest are functional graphics.

| ID | WxH | Ver | Bytes | Role | Cite |
| ---: | --- | --- | ---: | --- | --- |
| 1000 | 640x460 | v2 | 108,482 | `kSplash8BitPICT` — the title screen | `GliderPRO/Headers/GliderDefines.h:524` |
| 1001 | 431x32 | v2 | 5,998 | Load-House dialog banner | DITL 1000 item 3 |
| 1002 | 257x32 | v2 | 4,986 | shared object-info banner, used by **12** DITLs: 1007, 1010, 1011, 1013, 1014, 1015, 1019, 1022, 1027, 1033, 1034, 1045 | see §3.3.4 |
| 1003 | 32x32 | v2 | 282 | `kDefaultHousePict1` — **dead**, zero references | `GliderPRO/Sources/SelectHouse.c` |
| 1004 | 32x32 | v2 | 2,938 | `kDefaultHousePict8` — generic house icon in the Load-House list | `GliderPRO/Sources/SelectHouse.c:116` |
| 1005 | 313x32 | v2 | 5,226 | House-Info banner | DITL 1001 item 8 |
| 1006 | 333x32 | v2 | 5,870 | Display-Prefs banner | DITL 1017 item 7 |
| 1007 | 385x32 | v2 | 5,708 | Room-Info banner | DITL 1003 item 13 |
| 1008 | 316x32 | v2 | 5,580 | Sound-Prefs banner | DITL 1018 item 12 |
| 1009 | 76x28 | **v1** | 329 | Room-Info "Tiles:" strip | DITL 1003 item 16 |
| 1010 | 32x380 | v2 | 6,850 | `kThumbnailPictID` — the Map window's room thumbnail strip | `GliderPRO/Sources/Map.c:25` |
| 1011 | 360x216 | v2 | 48,728 | `kToolsPictID` — the entire Tools palette artwork | `GliderPRO/Sources/Tools.c:46` |
| 1012 | 316x32 | v2 | 5,644 | Control-Prefs banner | DITL 1023 item 4 |
| 1013 | 289x32 | v2 | 5,080 | Main-Prefs banner | DITL 1012 item 2 |
| 1014 | 316x32 | v2 | 5,582 | Brains-Prefs banner | DITL 1024 item 4 |
| 1015 | 214x54 | v2 | 6,092 | `kEscPausePictID` — "Paused, press Esc" overlay | `GliderPRO/Sources/Input.c:17` |
| 1016 | 214x54 | v2 | 6,106 | `kTabPausePictID` — "Paused, press Tab" overlay | `GliderPRO/Sources/Input.c:18` |
| 1017 | 256x64 | v2 | 9,770 | `kStarsRemainingPICT` — plural form of the banner text | `GliderPRO/Sources/Banner.c:20` |
| 1018 | 256x64 | v2 | 9,742 | `kStarRemainingPICT` — singular form | `GliderPRO/Sources/Banner.c:21` |
| 1019 | 96x44 | v2 | 4,064 | `kAngelPictID` — the death angel sprite | `GliderPRO/Sources/StructuresInit2.c:20` |
| 1020 | 96x44 | **v1** | 457 | angel **mask**, loaded as `kAngelPictID + 1` | `GliderPRO/Sources/StructuresInit2.c:134` |
| 1021 | 640x460 | v2 | 51,520 | `kMilkywayPictID` — game-over starfield | `GliderPRO/Sources/GameOver.c:22` |
| 1022 | 279x32 | v2 | 5,066 | Microwave-Info banner | DITL 1035 item 7 |
| 1023 | 280x32 | v2 | 5,016 | Go-To-Room banner | DITL 1043 item 9 |

`kEscPausePictID`/`kTabPausePictID` are selected by the user's preference at
`Sources/Input.c:85` and `Sources/Input.c:87`.

**Scaling matters here.** `LoadDialogPICT` (`Sources/DialogUtils.c`) is

```
GetDialogItem(theDialog, item, &itemType, &itemHandle, &iRect);
thePict = GetPicture(theID);
if (thePict != nil)
	DrawPicture(thePict, &iRect);
```

`DrawPicture` **stretches** the picture to fill `iRect`. For every banner in the table above
the DITL rect happens to equal the PICT's natural size, so no stretching occurs — but a port
must implement the *scaling* semantics, not blind blitting, because
`LoadScaledGraphic(id, &rect)` (`Sources/Utilities.c`) is deliberately used for
`kBannerPageTopPICT` (`Sources/Banner.c:61`) and the star banners
(`Sources/Banner.c:224`, `Sources/Banner.c:227`), where the destination rect is *not* the
picture size. Also note `LoadDialogPICT` never releases the handle.

### 3.1.5 Background thumbnails: 1202, 1211, 1216, 1217

Four 128x80 v2 pictures. They are loaded as `tempBack - 800`:

```
GliderPRO/Sources/RoomInfo.c:418   thePicture = GetPicture(tempBack - 800);
GliderPRO/Sources/RoomInfo.c:541   thePicture = GetPicture(tempBack - 800);
```

so 1202 ← background 2002, 1211 ← 2011, 1216 ← 2016, 1217 ← 2017. **Only these four of the
eighteen backgrounds have a preview thumbnail**; for the other fourteen `GetPicture` returns
`nil` and `Sources/RoomInfo.c:420`/`:543` falls through leaving the preview area blank. This
is a data gap in the shipped app, not a code bug — reproduce it, or generate all 18
thumbnails and accept a cosmetic divergence.

### 3.1.6 Banner / game-over / scoreboard art: 1988-1999

| ID | WxH | Ver | Constant | Cite |
| ---: | --- | --- | --- | --- |
| 1988 | 25x256 | v2 | `kLettersPictID` | `GliderPRO/Sources/GameOver.c:21` |
| 1989 | 32x448 | **v1** | `kPagesMaskID` | `GliderPRO/Sources/GameOver.c:20` |
| 1990 | 32x448 | v2 | `kPagesPictID` | `GliderPRO/Sources/GameOver.c:19` |
| 1991 | 330x30 | **v1** | `kBannerPageBottomMask` | `GliderPRO/Sources/Banner.c:19`, loaded `:73` |
| 1992 | 330x30 | v2 | `kBannerPageBottomPICT` | `GliderPRO/Sources/Banner.c:18`, loaded `:69` |
| 1993 | 330x190 | v2 | `kBannerPageTopPICT` | `GliderPRO/Sources/Banner.c:17`, loaded `:61` (scaled) |
| 1994 | 332x30 | v2 | `kHighScoresPictID` | `GliderPRO/Sources/HighScores.c:23` |
| 1995 | 640x460 | v2 | `kStarPictID` | `GliderPRO/Headers/GliderDefines.h:534` |
| 1996 | 32x66 | v2 | `kBadgePictID` | `GliderPRO/Sources/StructuresInit.c:39` |
| 1997 | 1536x20 | v2 | `kScoreboardPictID` | `GliderPRO/Headers/GliderDefines.h:623` |
| 1998 | 332x30 | **v1** | `kHighScoresMaskID` | `GliderPRO/Sources/HighScores.c:24` |
| 1999 | 512x44 | v2 | `kSupportPictID` — the floor-support beam | `GliderPRO/Sources/StructuresInit2.c:21` |

**The mask-numbering conventions in this block are all different**, and there are four of
them in the whole fork:

| Convention | Instance |
| --- | --- |
| `mask = art + 1000` | sprite sheets 3998-4018 → 4998-5018 |
| `mask = art + 1` | angel 1019 → 1020 |
| `mask = art − 1` | game-over pages 1990 → 1989; banner bottom 1992 → 1991 |
| `mask = art + 4` | high-score banner 1994 → 1998 |
| hand-numbered | eight object masks in 3900-3927 |

A Go asset table must be explicit; do not derive mask IDs by formula.

**Verified: 28 object-mask IDs are `#define`d but only 8 exist, and the 8 that exist are exactly
the 8 that are used.** `GliderPRO/Sources/ObjectDraw2.c:39-66` declares 28 consecutive mask
constants covering 3900-3927:

| ID | Constant | Line | PICT present? | Used in code? |
| ---: | --- | ---: | :---: | :---: |
| 3900 | `kBBQMaskID` | `:39` | no | no |
| 3901 | `kUpStairsMaskID` | `:40` | no | no |
| 3902 | `kTrunkMaskID` | `:41` | no | no |
| 3903 | `kMailboxRightMaskID` | `:42` | **yes** | **yes** (`:262`) |
| 3904 | `kMailboxLeftMaskID` | `:43` | **yes** | **yes** (`:179`) |
| 3905 | `kDoorInLeftMaskID` | `:44` | no | no |
| 3906 | `kDoorInRightMaskID` | `:45` | no | no |
| 3907 | `kWindowInLeftMaskID` | `:46` | no | no |
| 3908 | `kWindowInRightMaskID` | `:47` | no | no |
| 3909 | `kHipLampMaskID` | `:48` | no | no |
| 3910 | `kDecoLampMaskID` | `:49` | no | no |
| 3911 | `kGuitarMaskID` | `:50` | no | no |
| 3912 | `kTVMaskID` | `:51` | **yes** | **yes** (`:660`) |
| 3913 | `kVCRMaskID` | `:52` | **yes** | **yes** (`:754`) |
| 3914 | `kStereoMaskID` | `:53` | **yes** | **yes** (`:809`) |
| 3915 | `kMicrowaveMaskID` | `:54` | **yes** | **yes** (`:864`) |
| 3916 | `kFireplaceMaskID` | `:55` | no | no |
| 3917 | `kBearMaskID` | `:56` | no | no |
| 3918 | `kVase1MaskID` | `:57` | no | no |
| 3919 | `kVase2MaskID` | `:58` | no | no |
| 3920 | `kManholeMaskID` | `:59` | no | no |
| 3921 | `kCloudMaskID` | `:61` | **yes** | **yes** (`:1274`) |
| 3922 | `kBooksMaskID` | `:60` | no | no |
| 3923 | `kRugMaskID` | `:62` | no | no |
| 3924 | `kChimesMaskID` | `:63` | no | no |
| 3925 | `kCinderMaskID` | `:64` | no | no |
| 3926 | `kFlowerBoxMaskID` | `:65` | no | no |
| 3927 | `kCobwebMaskID` | `:66` | **yes** | **yes** (`:1269`) |

The set of `'PICT'` IDs actually present in 3900-3999 is
`{3903, 3904, 3912, 3913, 3914, 3915, 3921, 3927}` followed by the object-art block starting at
3957 — verified by parsing the fork. The set of mask constants referenced anywhere outside their
own `#define` is the **same eight**, verified by grepping all 28 names. There is no
inconsistency: the 20 unused constants were reserved for masks that were never drawn, and
because `LoadGraphic` calls `RedAlert(kErrFailedGraphicLoad)` on a nil `'PICT'`
(`GliderPRO/Sources/Utilities.c:323-324`), referencing any of the 20 would be a fatal error at
run time. Note also that `:60` and `:61` are out of numeric order (`kBooksMaskID` 3922 is
declared before `kCloudMaskID` 3921).

`kScoreboardPictID` at 1536x20 is 3 x 512 wide — three horizontally-tiled scoreboard states.
`Sources/Environ.c` sizes its GWorld as `(6396L * thisMac.isDepth) / 8L` bytes, consistent
with 1536x20 at `isDepth` bits per pixel plus rounding.

### 3.1.7 Room backgrounds: 2000-2017

Eighteen pictures, all exactly **512x322** v2, totalling 1,037,846 bytes (54% of all PICT
data). The controlling constants:

```
GliderPRO/Headers/GliderDefines.h:519   #define kBaseBackgroundID     2000
GliderPRO/Headers/GliderDefines.h:520   #define kFirstOutdoorBack     2009
GliderPRO/Headers/GliderDefines.h:521   #define kNumBackgrounds       18
GliderPRO/Headers/GliderDefines.h:522   #define kUserBackground       3000
GliderPRO/Headers/GliderDefines.h:523   #define kUserStructureRange   3300
```

`512 == kRoomWide` (`GliderPRO/Headers/GliderDefines.h:499`) and
`322 == kTileHigh` (`:498`); a room is `kNumTiles == 8` tiles of `kTileWide == 64` (`:496`,
`:497`) — 8 x 64 = 512, so one background PICT is exactly one screen-wide room image.

Names come from `'MENU'` 140 (§3.7.5), in ID order:

| ID | Name | Bytes | Indoor/outdoor |
| ---: | --- | ---: | --- |
| 2000 | Simple Room | 31,838 | indoor |
| 2001 | Paneled Room | 89,338 | indoor |
| 2002 | Basement | 52,328 | indoor |
| 2003 | Child's Room | 43,158 | indoor |
| 2004 | Asian Room | 83,276 | indoor |
| 2005 | Unfinished Room | 70,572 | indoor |
| 2006 | Swinger's Room | 105,140 | indoor |
| 2007 | Bathroom | 61,926 | indoor |
| 2008 | Library | 82,922 | indoor |
| 2009 | Garden | 34,856 | **first outdoor** (`kFirstOutdoorBack`) |
| 2010 | Skywalk | 43,020 | outdoor |
| 2011 | Dirt | 81,036 | outdoor |
| 2012 | Meadow | 37,790 | outdoor |
| 2013 | Field | 32,966 | outdoor |
| 2014 | Roof | 75,132 | outdoor |
| 2015 | Sky | 23,748 | outdoor |
| 2016 | Stratosphere | 76,176 | outdoor |
| 2017 | Stars | 12,624 | outdoor |

Rooms may instead specify a *house-supplied* background at an ID ≥ `kUserBackground` (3000).
The three candidate upper bounds disagree: `kUserStructureRange - 1` = 3299 (constant), the
DITL 1016 item 6 label `'(3000 - 3499)'` (UI text), and `(longID >= 3000) && (longID < 3800)`
— the only bound actually **enforced**, at `GliderPRO/Sources/RoomInfo.c:762`. No code compares
a typed background ID against 3299; `kUserStructureRange` is used only at
`GliderPRO/Sources/Room.c:781`, to classify a custom background as a "structure" when the
room's `bounds` field is 0. DITL 1016 item 5 defaults its edit field to `'3000'`. See §3.3.5
and "Open questions" #4.

### 3.1.8 Room-object art: 3957-3997 and 3903-3927 masks

41 art pictures plus 8 masks. Every art PICT here is drawn by `Sources/ObjectDraw.c` /
`Sources/ObjectDraw2.c` through one of four routines: plain `CopyBits`,
`DrawPictSansWhiteObject` (white treated as transparent),
`DrawPictWithMaskObject` (uses a paired 1-bit mask), or via a preloaded GWorld.

The only two objects that go through `DrawPictWithMaskObject`
(`GliderPRO/Sources/ObjectDraw2.c:1251-1297`) are:

| Object | Art PICT | Mask PICT |
| --- | ---: | ---: |
| `kCobweb` | 3958 | 3927 |
| `kCloud` | 3965 | 3921 |

The routine builds two temporary GWorlds (one at `kPreferredDepth`, one at depth 1),
`LoadGraphic`s art and mask into them, then

```
CopyMask(tempMap, tempMask, backSrcMap, &srcRects[what], &srcRects[what], theRect);
```

and disposes both GWorlds — i.e. **the mask path allocates and frees two offscreens on every
draw**. A Go port should hoist this to a cached RGBA texture with an alpha channel.

The remaining six 1-bit masks (3903, 3904, 3912, 3913, 3914, 3915) exist in the fork but
are only reachable through constants in `Sources/ObjectDraw2.c`; the twenty other mask
constants declared there (3900, 3901, 3902, 3905-3911, 3916-3920, 3922-3926) have **no
matching resource** and are dead. Do not port them.

Four PICTs are the glider itself — three of them in this 3900s block, plus 3999 — all
48x668. The strip is **31 sub-rects**
(`kNumGliderSrcRects` = 31, `GliderPRO/Headers/GliderDefines.h:558`), and the rows are *not*
uniform (`Sources/StructuresInit.c:191-205`):

* frames 0-20: 48x**20** (`kGliderHigh`), at y = 20*i, i.e. y 0-419
* frames 21-28: 48x**26** (`kGliderBurningHigh`), at y = 420 + 26*(i-21), i.e. y 420-627
* frame 29: 48x20 at y=628, frame 30: 48x20 at y=648

20*21 + 26*8 + 20*2 = 668, so the strip is exactly consumed. The four sheets are:

| ID | Constant | Cite |
| ---: | --- | --- |
| 3963 | `kGliderFoil2PictID` | `GliderPRO/Headers/GliderDefines.h:565` |
| 3974 | `kGlider2PictID` | `GliderPRO/Headers/GliderDefines.h:566` |
| 3976 | `kGliderFoilPictID` | `GliderPRO/Headers/GliderDefines.h:567` |
| 3999 | `kGliderPictID` | `GliderPRO/Headers/GliderDefines.h:568` |

3999 is also 48x668 (all four are, verified from their `picFrame`s). There are only ever **two**
glider GWorlds, `glidSrcMap` and `glid2SrcMap`, created once at `Sources/StructuresInit.c:178-189`
(plus the single shared 1-bit mask 4999 in `glidMaskMap`, loaded at `:187-189`); all four
pictures are swapped through that pair at runtime with `SetPort` + `LoadGraphic`:

| When | `glidSrcMap` | `glid2SrcMap` | Cite |
| --- | --- | --- | --- |
| startup | 3999 | 3974 | `StructuresInit.c:181`, `:185` |
| game start, 2 players | 3999 (P1) | 3974 (P2) | `Sources/Play.c:124-127` |
| game start, 1 player | 3999 (normal) | **3976** (foil) | `Sources/Play.c:132-135` |
| foil picked up, 2 players | 3976 | 3963 | `Sources/Player.c:1147-1151` (`DeckGliderInFoil`) |
| foil lost, 2 players | 3999 | 3974 | `Sources/Player.c:1207-1211` (`RemoveFoilFromGlider`) |

Note the asymmetry: in a one-player game the "player 2" GWorld is repurposed to hold that
player's foil sheet, and `DeckGliderInFoil` / `RemoveFoilFromGlider` reload nothing (both
bodies are guarded by `if (twoPlayerGame)`). A Go port can simply keep all four sheets
resident and select between them.

3957 is `kManholeThruFloor`, 123x44.

### 3.1.9 Sprite sheets 3998-4018 and masks 4998-5018

These are the *animated* / *multi-state* object sheets that `Sources/StructuresInit.c`
preloads into long-lived GWorlds at startup, one GWorld per functional family. The mask, if
present, is loaded into a second GWorld at depth 1 and the pair is later composited with
`CopyMask`.

All line numbers below are the `QSetRect` … `LoadGraphic(id)` … `LoadGraphic(id + 1000)` span
in `GliderPRO/Sources/StructuresInit.c`. Note that the GWorld families do **not** map 1:1 to
the `Init*` functions: `InitGliderMap` also builds the shadow and rubber-band sheets,
`InitPrizes` also builds the points sheet, and `InitAppliances` also builds toast and shredded.

| Art | WxH | Mask | Family GWorld built at (`Sources/StructuresInit.c`) |
| ---: | --- | ---: | --- |
| 3998 | 48x18 | 4998 | `kShadowPictID` — glider shadow, `:207-214` (in `InitGliderMap`) |
| 3999 | 48x668 | 4999 | `kGliderPictID`, `:178-189` (`InitGliderMap`) |
| 4000 | 48x402 | 5000 | `kBlowerPictID`, `:253-260` (`InitBlowers`) |
| 4001 | 64x221 | 5001 | `kFurniturePictID`, `:301-308` (`InitFurniture`; GWorld is 64x**278**) |
| 4002 | 88x378 | 5002 | `kBonusPictID`, `:350-357` (`InitPrizes`) |
| 4003 | 32x104 | **none** | `kSwitchPictID`, `:455-458` (`InitSwitches`) — the single maskless sheet |
| 4004 | 72x126 | 5004 | `kLightPictID`, `:501-508` (`InitLights`) |
| 4005 | 80x269 | 5005 (80x**268**) | `kAppliancePictID`, `:538-545` (`InitAppliances`) |
| 4006 | 24x120 | 5006 (origin 0,91) | `kPointsPictID`, `:403-410` (in `InitPrizes`) |
| 4007 | 16x18 | 5007 | `kRubberBandsPictID`, `:222-229` (in `InitGliderMap`) |
| 4008 | 56x32 | 5008 | `kTransportPictID`, `:431-438` (`InitTransports`) |
| 4009 | 32x174 | 5009 | `kToastPictID`, `:547-554` (in `InitAppliances`) |
| 4010 | 40x35 | 5010 (origin 195,0) | `kShreddedPictID`, `:556-563` (in `InitAppliances`) |
| 4011 | 24x240 | 5011 | `kBalloonPictID`, `:623-630` (`InitEnemies`) |
| 4012 | 32x300 | 5012 | `kCopterPictID`, `:632-639` (`InitEnemies`) |
| 4013 | 64x76 | 5013 | `kDartPictID`, `:641-648` (`InitEnemies`) |
| 4014 | 32x64 | 5014 | `kBallPictID`, `:650-657` (`InitEnemies`) |
| 4015 | 16x72 | 5015 | `kDripPictID`, `:659-666` (`InitEnemies`) |
| 4016 | 36x33 | 5016 | `kEnemyPictID`, `:668-675` (`InitEnemies`) |
| 4017 | 16x128 | 5017 | `kFishPictID`, `:677-684` (`InitEnemies`) |
| 4018 | 128x69 | 5018 | `kClutterPictID`, `Sources/StructuresInit2.c:22`, GWorlds `:63-70` |

All 20 masks are PICT **v1** (1-bit), and their byte sizes (95 to 4,102) confirm 1-bit
depth: e.g. 4999 art is 18,688 bytes at 8bpp while mask 4999 is 4,102 bytes at 1bpp for the
same 48x668.

The clutter sheet 4018 is subdivided into six flower sprites by hard-coded rects
(`GliderPRO/Sources/StructuresInit2.c:72-88` — a `QSetRect` giving the size followed by a
`QOffsetRect` giving the position, per flower), within a source rect of 0,0,128,69:

| Flower index | Rect (l, t, w, h) |
| ---: | --- |
| 0 | 0, 23, 10x28 |
| 1 | 10, 16, 24x35 |
| 2 | 34, 16, 34x35 |
| 3 | 68, 14, 27x23 |
| 4 | 68, 37, 27x14 |
| 5 | 95, 0, 32x51 |

These map to DITL 1033's six radio buttons: Dandelion, Tulip, Orchid, Violets, Daisies,
Sunflower (§3.3.4).

### 3.1.10 `'PICT'` 10000 — the custom-picture placeholder

A 72x34 v2 picture, 4,182 bytes. It is the *fallback* art shown for a `kCustomPict` object
whose house-supplied PICT is missing:

```
GliderPRO/Sources/ObjectRects.c:220-232
	case kCustomPict:
	thePict = GetPicture(who->data.g.height);
	if (thePict == nil)
	{
		who->data.g.height = 10000;
		*itsRect = srcRects[who->what];
	}
	else { HLock((Handle)thePict); *itsRect = (*thePict)->picFrame; HUnlock(...); }
```

Note the object stores its PICT ID in `data.g.height` — a field named "height" reused as a
resource ID. Newly-added custom pictures are initialised to 10000
(`GliderPRO/Sources/ObjectAdd.c:628-631`), the editor also resets to 10000
(`GliderPRO/Sources/ObjectEdit.c:2285`), the entry dialog's default text is `"\p10000"`
(`GliderPRO/Sources/ObjectInfo.c:1186`), and the validity test is

```
GliderPRO/Sources/ObjectInfo.c:1213
	if ((wasPict < 10000L) || (wasPict > 32767L)) SysBeep(1);
```

so **custom-picture IDs are legal in 10000..32767 inclusive**, and 10000 doubles as both the
sentinel and a real drawable resource. `srcRects[kCustomPict]` is `0,0,72,34`
(`GliderPRO/Sources/StructuresInit2.c:448`), matching PICT 10000's dimensions exactly.

The same DLOG (1045) is reused for custom *sounds*, with `ParamText` swapping the labels:

```
GliderPRO/Sources/ObjectInfo.c:1186   ParamText(numberStr, kindStr, "\pPICT", "\p10000");
GliderPRO/Sources/ObjectInfo.c:1188   ParamText(numberStr, kindStr, "\pSound", "\p3000");
```

### 3.1.11 Complete PICT table

`id`, payload bytes, `picSize` field, version, `picFrame`, and derived WxH — every one of
the 152, as parsed:

| ID | Bytes | picSize | Ver | picFrame (t,l,b,r) | WxH |
| ---: | ---: | ---: | --- | --- | --- |
| 150 | 632 | 632 | v1 | 0,0,63,63 | 63x63 |
| 151 | 632 | 632 | v1 | 0,0,63,63 | 63x63 |
| 153 | 15194 | 15194 | v2 | 0,0,100,372 | 372x100 |
| 1000 | 108482 | 42946 | v2 | 0,0,460,640 | 640x460 |
| 1001 | 5998 | 5998 | v2 | 0,0,32,431 | 431x32 |
| 1002 | 4986 | 4986 | v2 | 0,0,32,257 | 257x32 |
| 1003 | 282 | 282 | v2 | 0,0,32,32 | 32x32 |
| 1004 | 2938 | 2938 | v2 | 0,0,32,32 | 32x32 |
| 1005 | 5226 | 5226 | v2 | 0,0,32,313 | 313x32 |
| 1006 | 5870 | 5870 | v2 | 0,0,32,333 | 333x32 |
| 1007 | 5708 | 5708 | v2 | 0,0,32,385 | 385x32 |
| 1008 | 5580 | 5580 | v2 | 0,0,32,316 | 316x32 |
| 1009 | 329 | 329 | v1 | 0,0,28,76 | 76x28 |
| 1010 | 6850 | 6850 | v2 | 0,0,380,32 | 32x380 |
| 1011 | 48728 | 48728 | v2 | 0,0,216,360 | 360x216 |
| 1012 | 5644 | 5644 | v2 | 0,0,32,316 | 316x32 |
| 1013 | 5080 | 5080 | v2 | 0,0,32,289 | 289x32 |
| 1014 | 5582 | 5582 | v2 | 0,0,32,316 | 316x32 |
| 1015 | 6092 | 6092 | v2 | 0,0,54,214 | 214x54 |
| 1016 | 6106 | 6106 | v2 | 0,0,54,214 | 214x54 |
| 1017 | 9770 | 9770 | v2 | 0,0,64,256 | 256x64 |
| 1018 | 9742 | 9742 | v2 | 0,0,64,256 | 256x64 |
| 1019 | 4064 | 4064 | v2 | 0,0,44,96 | 96x44 |
| 1020 | 457 | 457 | v1 | 0,0,44,96 | 96x44 |
| 1021 | 51520 | 51520 | v2 | 0,0,460,640 | 640x460 |
| 1022 | 5066 | 5066 | v2 | 0,0,32,279 | 279x32 |
| 1023 | 5016 | 5016 | v2 | 0,0,32,280 | 280x32 |
| 1202 | 5826 | 5826 | v2 | 0,0,80,128 | 128x80 |
| 1211 | 5272 | 5272 | v2 | 0,0,80,128 | 128x80 |
| 1216 | 3328 | 3328 | v2 | 0,0,80,128 | 128x80 |
| 1217 | 2884 | 2884 | v2 | 0,0,80,128 | 128x80 |
| 1988 | 5900 | 5900 | v2 | 0,0,256,25 | 25x256 |
| 1989 | 1851 | 1851 | v1 | 0,0,448,32 | 32x448 |
| 1990 | 10052 | 10052 | v2 | 0,0,448,32 | 32x448 |
| 1991 | 477 | 477 | v1 | 0,0,30,330 | 330x30 |
| 1992 | 9360 | 9360 | v2 | 0,0,30,330 | 330x30 |
| 1993 | 43936 | 43936 | v2 | 0,0,190,330 | 330x190 |
| 1994 | 4972 | 4972 | v2 | 0,0,30,332 | 332x30 |
| 1995 | 20746 | 20746 | v2 | 0,0,460,640 | 640x460 |
| 1996 | 3144 | 3144 | v2 | 0,0,66,32 | 32x66 |
| 1997 | 13296 | 13296 | v2 | 0,0,20,1536 | 1536x20 |
| 1998 | 1049 | 1049 | v1 | 0,0,30,332 | 332x30 |
| 1999 | 16850 | 16850 | v2 | 0,0,44,512 | 512x44 |
| 2000 | 31838 | 31838 | v2 | 0,0,322,512 | 512x322 |
| 2001 | 89338 | 23802 | v2 | 0,0,322,512 | 512x322 |
| 2002 | 52328 | 52328 | v2 | 0,0,322,512 | 512x322 |
| 2003 | 43158 | 43158 | v2 | 0,0,322,512 | 512x322 |
| 2004 | 83276 | 17740 | v2 | 0,0,322,512 | 512x322 |
| 2005 | 70572 | 5036 | v2 | 0,0,322,512 | 512x322 |
| 2006 | 105140 | 39604 | v2 | 0,0,322,512 | 512x322 |
| 2007 | 61926 | 61926 | v2 | 0,0,322,512 | 512x322 |
| 2008 | 82922 | 17386 | v2 | 0,0,322,512 | 512x322 |
| 2009 | 34856 | 34856 | v2 | 0,0,322,512 | 512x322 |
| 2010 | 43020 | 43020 | v2 | 0,0,322,512 | 512x322 |
| 2011 | 81036 | 15500 | v2 | 0,0,322,512 | 512x322 |
| 2012 | 37790 | 37790 | v2 | 0,0,322,512 | 512x322 |
| 2013 | 32966 | 32966 | v2 | 0,0,322,512 | 512x322 |
| 2014 | 75132 | 9596 | v2 | 0,0,322,512 | 512x322 |
| 2015 | 23748 | 23748 | v2 | 0,0,322,512 | 512x322 |
| 2016 | 76176 | 10640 | v2 | 0,0,322,512 | 512x322 |
| 2017 | 12624 | 12624 | v2 | 0,0,322,512 | 512x322 |
| 3903 | 774 | 774 | v1 | 0,0,80,94 | 94x80 |
| 3904 | 784 | 784 | v1 | 0,0,80,94 | 94x80 |
| 3912 | 598 | 598 | v1 | 0,0,77,92 | 92x77 |
| 3913 | 212 | 212 | v1 | 0,0,22,96 | 96x22 |
| 3914 | 440 | 440 | v1 | 0,0,53,128 | 128x53 |
| 3915 | 477 | 477 | v1 | 0,0,59,92 | 92x59 |
| 3921 | 338 | 338 | v1 | 0,0,30,128 | 128x30 |
| 3927 | 450 | 450 | v1 | 0,0,45,54 | 54x45 |
| 3957 | 2970 | 2970 | v2 | 0,0,44,123 | 123x44 |
| 3958 | 3208 | 3208 | v2 | 0,0,45,54 | 54x45 |
| 3959 | 2668 | 2668 | v2 | 0,0,32,80 | 80x32 |
| 3960 | 3170 | 3170 | v2 | 0,0,62,40 | 40x62 |
| 3961 | 3954 | 3954 | v2 | 0,0,74,28 | 28x74 |
| 3962 | 2750 | 2750 | v2 | 0,0,18,144 | 144x18 |
| 3963 | 15258 | 15258 | v2 | 0,0,668,48 | 48x668 |
| 3964 | 4762 | 4762 | v2 | 0,0,51,64 | 64x51 |
| 3965 | 2946 | 2946 | v2 | 0,0,30,128 | 128x30 |
| 3966 | 4964 | 4964 | v2 | 0,0,58,80 | 80x58 |
| 3967 | 2800 | 2800 | v2 | 0,0,22,123 | 123x22 |
| 3968 | 3000 | 3000 | v2 | 0,0,57,35 | 35x57 |
| 3969 | 3096 | 3096 | v2 | 0,0,45,36 | 36x45 |
| 3970 | 4748 | 4748 | v2 | 0,0,92,63 | 63x92 |
| 3971 | 4702 | 4702 | v2 | 0,0,59,92 | 92x59 |
| 3972 | 3392 | 3392 | v2 | 0,0,58,56 | 56x58 |
| 3973 | 10946 | 10946 | v2 | 0,0,142,180 | 180x142 |
| 3974 | 17604 | 17604 | v2 | 0,0,668,48 | 48x668 |
| 3975 | 6552 | 6552 | v2 | 0,0,92,102 | 102x92 |
| 3976 | 14846 | 14846 | v2 | 0,0,668,48 | 48x668 |
| 3977 | 3270 | 3270 | v2 | 0,0,170,16 | 16x170 |
| 3978 | 3270 | 3270 | v2 | 0,0,170,16 | 16x170 |
| 3979 | 5386 | 5386 | v2 | 0,0,170,20 | 20x170 |
| 3980 | 5386 | 5386 | v2 | 0,0,170,20 | 20x170 |
| 3981 | 3794 | 3794 | v2 | 0,0,322,16 | 16x322 |
| 3982 | 3798 | 3798 | v2 | 0,0,322,16 | 16x322 |
| 3983 | 15272 | 15272 | v2 | 0,0,322,144 | 144x322 |
| 3984 | 15282 | 15282 | v2 | 0,0,322,144 | 144x322 |
| 3985 | 3396 | 3396 | v2 | 0,0,80,94 | 94x80 |
| 3986 | 4056 | 4056 | v2 | 0,0,80,94 | 94x80 |
| 3987 | 4294 | 4294 | v2 | 0,0,80,144 | 144x80 |
| 3988 | 3160 | 3160 | v2 | 0,0,33,64 | 64x33 |
| 3989 | 5144 | 5144 | v2 | 0,0,53,128 | 128x53 |
| 3990 | 2814 | 2814 | v2 | 0,0,22,96 | 96x22 |
| 3991 | 5726 | 5726 | v2 | 0,0,172,64 | 64x172 |
| 3992 | 6124 | 6124 | v2 | 0,0,77,92 | 92x77 |
| 3993 | 3740 | 3740 | v2 | 0,0,212,64 | 64x212 |
| 3994 | 5832 | 5832 | v2 | 0,0,276,72 | 72x276 |
| 3995 | 4706 | 4706 | v2 | 0,0,107,74 | 74x107 |
| 3996 | 5986 | 5986 | v2 | 0,0,267,160 | 160x267 |
| 3997 | 10950 | 10950 | v2 | 0,0,267,160 | 160x267 |
| 3998 | 167 | 167 | v1 | 0,0,18,48 | 48x18 |
| 3999 | 18688 | 18688 | v2 | 0,0,668,48 | 48x668 |
| 4000 | 11800 | 11800 | v2 | 0,0,402,48 | 48x402 |
| 4001 | 9416 | 9416 | v2 | 0,0,221,64 | 64x221 |
| 4002 | 15432 | 15432 | v2 | 0,0,378,88 | 88x378 |
| 4003 | 4650 | 4650 | v2 | 0,0,104,32 | 32x104 |
| 4004 | 5284 | 5284 | v2 | 0,0,126,72 | 72x126 |
| 4005 | 12578 | 12578 | v2 | 0,0,269,80 | 80x269 |
| 4006 | 4682 | 4682 | v2 | 0,0,120,24 | 24x120 |
| 4007 | 2384 | 2384 | v2 | 0,0,18,16 | 16x18 |
| 4008 | 2740 | 2740 | v2 | 0,0,32,56 | 56x32 |
| 4009 | 4768 | 4768 | v2 | 0,0,174,32 | 32x174 |
| 4010 | 3308 | 3308 | v2 | 0,0,35,40 | 40x35 |
| 4011 | 5618 | 5618 | v2 | 0,0,240,24 | 24x240 |
| 4012 | 5770 | 5770 | v2 | 0,0,300,32 | 32x300 |
| 4013 | 4434 | 4434 | v2 | 0,0,76,64 | 64x76 |
| 4014 | 3526 | 3526 | v2 | 0,0,64,32 | 32x64 |
| 4015 | 2822 | 2822 | v2 | 0,0,72,16 | 16x72 |
| 4016 | 2752 | 2752 | v2 | 0,0,33,36 | 36x33 |
| 4017 | 3902 | 3902 | v2 | 0,0,128,16 | 16x128 |
| 4018 | 4864 | 4864 | v2 | 0,0,69,128 | 128x69 |
| 4998 | 167 | 167 | v1 | 0,0,18,48 | 48x18 |
| 4999 | 4102 | 4102 | v1 | 0,0,668,48 | 48x668 |
| 5000 | 2471 | 2471 | v1 | 0,0,402,48 | 48x402 |
| 5001 | 1908 | 1908 | v1 | 0,0,221,64 | 64x221 |
| 5002 | 4500 | 4500 | v1 | 0,0,378,88 | 88x378 |
| 5004 | 1334 | 1334 | v1 | 0,0,126,72 | 72x126 |
| 5005 | 2124 | 2124 | v1 | 0,0,268,80 | 80x268 |
| 5006 | 539 | 539 | v1 | **0,91,120,115** | 24x120 |
| 5007 | 95 | 95 | v1 | 0,0,18,16 | 16x18 |
| 5008 | 313 | 313 | v1 | 0,0,32,56 | 56x32 |
| 5009 | 755 | 755 | v1 | 0,0,174,32 | 32x174 |
| 5010 | 269 | 269 | v1 | **195,0,230,40** | 40x35 |
| 5011 | 1019 | 1019 | v1 | 0,0,240,24 | 24x240 |
| 5012 | 1259 | 1259 | v1 | 0,0,300,32 | 32x300 |
| 5013 | 702 | 702 | v1 | 0,0,76,64 | 64x76 |
| 5014 | 315 | 315 | v1 | 0,0,64,32 | 32x64 |
| 5015 | 203 | 203 | v1 | 0,0,72,16 | 16x72 |
| 5016 | 257 | 257 | v1 | 0,0,33,36 | 36x33 |
| 5017 | 315 | 315 | v1 | 0,0,128,16 | 16x128 |
| 5018 | 1026 | 1026 | v1 | 0,0,69,128 | 128x69 |
| 10000 | 4182 | 4182 | v2 | 0,0,34,72 | 72x34 |

### 3.1.12 PICT loading in code

`Sources/Utilities.c` has the two entry points everything else funnels through:

* `LoadGraphic(short resID)` — `GetPicture`, `HNoPurge`, `DrawPicture` into the *current*
  port at the picture's natural `picFrame`, `HPurge`, `ReleaseResource`.
* `LoadScaledGraphic(short resID, Rect *theRect)` — same but `DrawPicture(thePict, theRect)`,
  stretching.

`RedAlert(kErrFailedGraphicLoad)` (STR# 170/171 index 5) fires if `GetPicture` returns nil.

Complete list of `GetPicture` / `LoadGraphic` / `LoadScaledGraphic` / `LoadDialogPICT` call
sites, so a port can enumerate every asset consumer:

`Sources/Banner.c:61`, `:69`, `:73`, `:224`, `:227`;
`Sources/GameOver.c:87`, `:264`, `:269`, `:273`;
`Sources/HighScores.c:67`, `:114`, `:118`;
`Sources/Input.c:85`, `:87`;
`Sources/MainWindow.c:106`, `:143`, `:247`;
`Sources/Map.c:746`;
`Sources/ObjectDraw2.c:175`, `:179`, `:258`, `:262`, `:656`, `:660`, `:750`, `:754`,
`:805`, `:809`, `:860`, `:864`, `:1281`, `:1285`, `:1399`, `:1426`;
`Sources/ObjectRects.c:221`;
`Sources/Play.c:125`, `:127`, `:133`, `:135`, `:271`;
`Sources/Player.c:1149`, `:1151`, `:1209`, `:1211`;
`Sources/RoomGraphics.c:239`, `:282`, `:300`, `:317`, `:335`, `:353`, `:371`;
`Sources/RoomInfo.c:418`, `:420`, `:514`, `:541`, `:543`, `:555`;
`Sources/SelectHouse.c:111`, `:116`;
`Sources/StructuresInit.c:95`, `:181`, `:185`, `:189`, `:210`, `:214`, `:225`, `:229`,
`:256`, `:260`, `:304`, `:308`, `:353`, `:357`, `:406`, `:410`, `:434`, `:438`, `:458`,
`:504`, `:508`, `:541`, `:545`, `:550`, `:554`, `:559`, `:563`, `:626`, `:630`, `:635`,
`:639`, `:644`, `:648`, `:653`, `:657`, `:662`, `:666`, `:671`, `:675`, `:680`, `:684`;
`Sources/StructuresInit2.c:66`, `:70`, `:109`, `:130`, `:134`;
`Sources/Tools.c:85`.

---

## 3.2 `'snd '` — 70 resources, 1,129,344 bytes

### 3.2.1 Verified binary layout

Every one of the 70 is a **format-1** `'snd '` resource with a single `sampledSynth`
reference and a single `bufferCmd`, pointing at a **standard sound header** (`stdSH`) of
8-bit unsigned PCM. Parsed layout, identical for all 70:

| Offset | Size | Field | Observed value |
| ---: | ---: | --- | --- |
| 0 | 2 | `format` | `0x0001` |
| 2 | 2 | `numSynths` | `0x0001` |
| 4 | 2 | `synthID` | `0x0005` = `sampledSynth` |
| 6 | 4 | `initOption` | `0x000000A0` = `initMono \| initNoInterp` |
| 10 | 2 | `numCommands` | `0x0001` |
| 12 | 2 | `cmd` | `0x8051` = `bufferCmd (0x0051)` with `dataOffsetFlag (0x8000)` |
| 14 | 2 | `param1` | `0x0000` |
| 16 | 4 | `param2` | `20` — offset from resource start to the sound header |
| 20 | 4 | `samplePtr` | `0x00000000` (means "samples follow the header") |
| 24 | 4 | `length` | frame count |
| 28 | 4 | `sampleRate` | `0x56EE8BA3` (16.16 fixed) = **22254.5455 Hz** |
| 32 | 4 | `loopStart` | see below |
| 36 | 4 | `loopEnd` | see below |
| 40 | 1 | `encode` | `0x00` = `stdSH` |
| 41 | 1 | `baseFrequency` | `60` = middle C |
| 42 | `length` | sample bytes | 8-bit **unsigned, offset-binary** (silence = 128) |

Verified invariants across all 70 resources:

* `20 + 22 + length == len(resource)` — **exact for all 70**. There is no trailing padding.
* `initOption` is `0xA0` in all 70 (single distinct value observed).
* `sampleRate` is `0x56EE8BA3` in all 70. `0x56EE8BA3 / 65536 = 22254.5454...` Hz — the Mac
  "22k" rate, which is `SANE`-exact `22254 + 3/11` Hz, **not** 22050 Hz.
* `baseFrequency` is 60 in all 70.
* `encode` is 0 (`stdSH`) in all 70 — no compressed (`cmpSH`) or extended (`extSH`) headers.
* Sample bytes span the full range: global min 0, global max 255.
* `loopStart`/`loopEnd` fall into exactly three patterns, enumerated exhaustively:

  | Pattern | Resources | Count |
  | --- | --- | ---: |
  | `(length-2, length-1)` — degenerate 1-frame loop, Sound Manager's "do not loop" idiom | 59 sound effects (all of 1000-1062 except the four below) | 59 |
  | `(0, length-1)` — loop the whole sample | 1018 `kThrustSound`, 1026 `kShredSound`, 1047 `kSizzleSound`, 1062 `kHissSound` | 4 |
  | `(0, length)` — loop the whole sample | 2000-2006 (all music) | 7 |

  The four loopable effects are exactly the four *sustained* noises (glider thrust, paper
  shredder, sizzle, gas hiss). A `bufferCmd` honours these loop points only when the caller
  keeps the channel running; the code plays effects one-shot, so the loop points are
  informational — but they are the right hint for a port that wants to sustain those four.

**Go port:** convert each payload to signed by `int16(b) - 128`, resample from
22254.5455 Hz, and treat `length` as the authoritative frame count.

### 3.2.2 Sound effects: `'snd '` 1000-1062, mapped to game events

The sound *index* used everywhere in the code is `id - 1000`.
`GliderPRO/Headers/GliderDefines.h:55-118` declares 64 indices (0..63); index 63 is special
(see §3.2.4). `kBaseBufferSoundID` is 1000.

| ID | Index | Constant | Bytes | Frames | Seconds |
| ---: | ---: | --- | ---: | ---: | ---: |
| 1000 | 0 | `kHitWallSound` | 1,882 | 1,840 | 0.083 |
| 1001 | 1 | `kFadeInSound` | 8,874 | 8,832 | 0.397 |
| 1002 | 2 | `kFadeOutSound` | 9,994 | 9,952 | 0.447 |
| 1003 | 3 | `kBeepsSound` | 9,722 | 9,680 | 0.435 |
| 1004 | 4 | `kBuzzerSound` | 8,074 | 8,032 | 0.361 |
| 1005 | 5 | `kDingSound` | 9,770 | 9,728 | 0.437 |
| 1006 | 6 | `kEnergizeSound` | 11,946 | 11,904 | 0.535 |
| 1007 | 7 | `kFollowSound` | 7,338 | 7,296 | 0.328 |
| 1008 | 8 | `kMicrowavedSound` | 10,922 | 10,880 | 0.489 |
| 1009 | 9 | `kSwitchSound` | 1,306 | 1,264 | 0.057 |
| 1010 | 10 | `kBirdSound` | 5,354 | 5,312 | 0.239 |
| 1011 | 11 | `kCuckooSound` | 5,178 | 5,136 | 0.231 |
| 1012 | 12 | `kTikSound` | 654 | 612 | 0.027 |
| 1013 | 13 | `kTokSound` | 494 | 452 | 0.020 |
| 1014 | 14 | `kBlowerOn` | 9,418 | 9,376 | 0.421 |
| 1015 | 15 | `kBlowerOff` | 8,170 | 8,128 | 0.365 |
| 1016 | 16 | `kCaughtFireSound` | 7,706 | 7,664 | 0.344 |
| 1017 | 17 | `kScoreTikSound` | 686 | 644 | 0.029 |
| 1018 | 18 | `kThrustSound` | 5,801 | 5,759 | 0.259 |
| 1019 | 19 | `kFizzleSound` | 2,858 | 2,816 | 0.127 |
| 1020 | 20 | `kFireBandSound` | 632 | 590 | 0.027 |
| 1021 | 21 | `kBandReboundSound` | 1,090 | 1,048 | 0.047 |
| 1022 | 22 | `kGreaseSpillSound` | 4,586 | 4,544 | 0.204 |
| 1023 | 23 | `kChordSound` | 20,010 | 19,968 | 0.897 |
| 1024 | 24 | `kVCRSound` | 11,562 | 11,520 | 0.518 |
| 1025 | 25 | `kFoilHitSound` | 3,818 | 3,776 | 0.170 |
| 1026 | 26 | `kShredSound` | 1,976 | 1,934 | 0.087 |
| 1027 | 27 | `kToastLaunchSound` | 1,274 | 1,232 | 0.055 |
| 1028 | 28 | `kToastLandSound` | 2,138 | 2,096 | 0.094 |
| 1029 | 29 | `kMacOnSound` | 1,374 | 1,332 | 0.060 |
| 1030 | 30 | `kMacBeepSound` | 12,202 | 12,160 | 0.546 |
| 1031 | 31 | `kMacOffSound` | 1,226 | 1,184 | 0.053 |
| 1032 | 32 | `kTVOnSound` | 9,826 | 9,784 | 0.440 |
| 1033 | 33 | `kTVOffSound` | 7,690 | 7,648 | 0.344 |
| 1034 | 34 | `kCoffeeSound` | 8,874 | 8,832 | 0.397 |
| 1035 | 35 | `kMysticSound` | 9,738 | 9,696 | 0.436 |
| 1036 | 36 | `kZapSound` | 4,381 | 4,339 | 0.195 |
| 1037 | 37 | `kPopSound` | 522 | 480 | 0.022 |
| 1038 | 38 | `kEnemyInSound` | 6,522 | 6,480 | 0.291 |
| 1039 | 39 | `kEnemyOutSound` | 4,922 | 4,880 | 0.219 |
| 1040 | 40 | `kPaperCrunchSound` | 6,970 | 6,928 | 0.311 |
| 1041 | 41 | `kBounceSound` | 3,018 | 2,976 | 0.134 |
| 1042 | 42 | `kDripSound` | 1,050 | 1,008 | 0.045 |
| 1043 | 43 | `kDropSound` | 1,418 | 1,376 | 0.062 |
| 1044 | 44 | `kFishOutSound` | 6,730 | 6,688 | 0.301 |
| 1045 | 45 | `kFishInSound` | 3,802 | 3,760 | 0.169 |
| 1046 | 46 | `kDontExitSound` | 2,410 | 2,368 | 0.106 |
| 1047 | 47 | `kSizzleSound` | 2,282 | 2,240 | 0.101 |
| 1048 | 48 | `kPaper1Sound` | 4,474 | 4,432 | 0.199 |
| 1049 | 49 | `kPaper2Sound` | 4,138 | 4,096 | 0.184 |
| 1050 | 50 | `kPaper3Sound` | 1,690 | 1,648 | 0.074 |
| 1051 | 51 | `kPaper4Sound` | 1,626 | 1,584 | 0.071 |
| 1052 | 52 | `kTypingSound` | 2,566 | 2,524 | 0.113 |
| 1053 | 53 | `kCarriageSound` | 5,290 | 5,248 | 0.236 |
| 1054 | 54 | `kChord2Sound` | 5,098 | 5,056 | 0.227 |
| 1055 | 55 | `kPhoneRingSound` | 11,050 | 11,008 | 0.495 |
| 1056 | 56 | `kChime1Sound` | 7,914 | 7,872 | 0.354 |
| 1057 | 57 | `kChime2Sound` | 9,546 | 9,504 | 0.427 |
| 1058 | 58 | `kWebTwangSound` | 2,314 | 2,272 | 0.102 |
| 1059 | 59 | `kTransOutSound` | 9,994 | 9,952 | 0.447 |
| 1060 | 60 | `kTransInSound` | 10,026 | 9,984 | 0.449 |
| 1061 | 61 | `kBonusSound` | 5,430 | 5,388 | 0.242 |
| 1062 | 62 | `kHissSound` | 3,002 | 2,960 | 0.133 |
| **—** | **63** | `kTriggerSound` | — | — | — |

Sound-effect subtotal: **352,348 bytes** for 63 resources. Longest effect is 1023
`kChordSound` at 0.897 s; shortest is 1013 `kTokSound` at 0.020 s.

### 3.2.3 Music: `'snd '` 2000-2006

Seven tracks, `kBaseBufferMusicID` = 2000, index `id - 2000`.

| ID | Index | Bytes | Frames | Seconds |
| ---: | ---: | ---: | ---: | ---: |
| 2000 | 0 | 97,076 | 97,034 | 4.360 |
| 2001 | 1 | 97,144 | 97,102 | 4.363 |
| 2002 | 2 | 97,074 | 97,032 | 4.360 |
| 2003 | 3 | 97,326 | 97,284 | 4.371 |
| 2004 | 4 | 194,070 | 194,028 | **8.719** |
| 2005 | 5 | 96,930 | 96,888 | 4.354 |
| 2006 | 6 | 97,376 | 97,334 | 4.374 |

Music subtotal: **776,996 bytes**, **34.9 seconds** of audio (sum of the column above =
34.901 s). Six pieces are ~4.36 s and
**piece 4 is exactly double-length (8.719 s)**. All seven have `loopStart = 0`,
`loopEnd = length`. `Sources/Music.c` chains them into a continuous stream, so a Go port must
respect the ordering logic there — the resource data alone does not tell you the sequence.

`MusicBytesNeeded()` (`GliderPRO/Sources/Music.c:386-407`) pre-flights the total with
`SetResLoad(false)` + `GetMaxResourceSize`, looping `i < kMaxMusic` (7). If any is missing it
returns `ResError()` and the caller raises ALRT 1038 `kNoMemForMusicAlert`
(`GliderPRO/Sources/Music.c:413`).

### 3.2.4 `'snd '` 1063 is deliberately absent

Sound index **63** is `kTriggerSound`, and there is no `'snd '` 1063 in the fork. The slot is
filled at run time from a *house-supplied* sound:

```
GliderPRO/Sources/Sound.c:279   (LoadTriggerSound)
```

`SoundBytesNeeded()` (`GliderPRO/Sources/Sound.c:490-511`) loops `i < kMaxSounds - 1`, i.e.
0..62 — it stops one short precisely because 1063 does not exist. A port must special-case
index 63 as "dynamic, loaded from the current house" and must not treat its absence as an
error.

### 3.2.5 Sound Manager usage and what Go must replace

```
GliderPRO/Sources/Sound.c:501   theSound = GetResource('snd ', kBaseBufferSoundID + i);
GliderPRO/Sources/Sound.c:518   RedAlert / kNoMemForSoundsAlert (ALRT 1039)
GliderPRO/Sources/Sound.c:522   Alert(kNoMemForSoundsAlert, nil);
GliderPRO/Sources/Sound.c:529   #define kNoSoundManager3Alert 1030
GliderPRO/Sources/Sound.c:533   Alert(kNoSoundManager3Alert, nil);
```

The app requires **Sound Manager 3.0** and refuses to make noise without it
(ALRT 1030, "Where is Sound Manager 3.0?"). It plays through `SndNewChannel` +
`SndDoCommand(bufferCmd)` on a small pool of channels with priority arbitration; the 61
priority constants live at `GliderPRO/Headers/GliderDefines.h:120-180` with values in
100..999. Go replacement: a mixer with N voices, per-voice priority, and the same
"higher priority steals a busy channel" rule. `callBackCmd` is used to detect completion —
in Go, use a per-voice done-channel.

### 3.2.6 `'snd '` resource names and playback priorities

All 70 sounds carry a ResEdit name. These are the only human-readable labels for the audio
and they resolve several constants whose names are opaque (`kMicrowavedSound` = "Miked",
`kWebTwangSound` = "Twunk", `kCaughtFireSound` = "Yow!").

The playback priority is a separate constant block at
`GliderPRO/Headers/GliderDefines.h:120-180` — **61 constants**, values 100 to 999. Higher wins
when all channels are busy. There are 61 priority constants for 64 sound indices because the
four paper sounds (indices 48-51, `kPaper1Sound`..`kPaper4Sound`) all share the single
`kPapersPriority` = 807 (`GliderDefines.h:167`), collapsing four indices into one constant and
saving exactly three. `kPapersPriority` is the *only* shared priority; every other index has
its own constant, and `kTikPriority` (200) and `kTokPriority` (201) are two distinct values.

| ID | Resource name | Priority constant | Value |
| ---: | --- | --- | ---: |
| 1000 | `Wall Hit` | `kHitWallPriority` | 100 |
| 1001 | `Fade In` | `kFadeInPriority` | 900 |
| 1002 | `Fade Out` | `kFadeOutPriority` | 901 |
| 1003 | `Beeps` | `kBeepsPriority` | 800 |
| 1004 | `Buzzer` | `kBuzzerPriority` | 801 |
| 1005 | `Ding` | `kDingPriority` | 802 |
| 1006 | `Energize` | `kEnergizePriority` | 803 |
| 1007 | `Follow` | `kFollowPriority` | 904 |
| 1008 | `Miked` | `kMicrowavedPriority` | 811 |
| 1009 | `PowerSwitch` | `kSwitchPriority` | 700 |
| 1010 | `Bird` | `kBirdPriority` | 804 |
| 1011 | `Cuckoo` | `kCuckooPriority` | 805 |
| 1012 | `Tik` | `kTikPriority` | 200 |
| 1013 | `Tok` | `kTokPriority` | 201 |
| 1014 | `Blowers On` | `kBlowerOnPriority` | 701 |
| 1015 | `Blowers Off` | `kBlowerOffPriority` | 702 |
| 1016 | `Yow!` | `kCaughtFirePriority` | 902 |
| 1017 | `Score Tick` | `kScoreTikPriority` | 101 |
| 1018 | `Thrust` | `kThrustPriority` | 300 |
| 1019 | `Fizzle` | `kFizzlePriority` | 703 |
| 1020 | `Fire Band` | `kFireBandPriority` | 301 |
| 1021 | `Band Rebound` | `kBandReboundPriority` | 102 |
| 1022 | `Grease Spill` | `kGreaseSpillPriority` | 806 |
| 1023 | `A Chord` | `kChordPriority` | 302 |
| 1024 | `VCR` | `kVCRPriority` | 303 |
| 1025 | `Foil Hit` | `kFoilHitPriority` | 400 |
| 1026 | `Shred` | `kShredPriority` | 903 |
| 1027 | `Toast Launch` | `kToastLaunchPriority` | 304 |
| 1028 | `Toast Land` | `kToastLandPriority` | 305 |
| 1029 | `MacClickOn` | `kMacOnPriority` | 401 |
| 1030 | `MacOn` | `kMacBeepPriority` | 403 |
| 1031 | `MacClickOff` | `kMacOffPriority` | 402 |
| 1032 | `TVOn` | `kTVOnPriority` | 404 |
| 1033 | `TVOff` | `kTVOffPriority` | 405 |
| 1034 | `Coffee` | `kCoffeePriority` | 306 |
| 1035 | `Mystic` | `kMysticPriority` | 202 |
| 1036 | `Zap` | `kZapPriority` | 406 |
| 1037 | `Pop!` | `kPopPriority` | 407 |
| 1038 | `Enemy In` | `kEnemyInPriority` | 408 |
| 1039 | `Enemy Out` | `kEnemyOutPriority` | 409 |
| 1040 | `Paper Crunch` | `kPaperCrunchPriority` | 410 |
| 1041 | `Bounce` | `kBouncePriority` | 307 |
| 1042 | `Drip` | `kDripPriority` | 308 |
| 1043 | `Drop` | `kDropPriority` | 309 |
| 1044 | `Fish Leap` | `kFishOutPriority` | 411 |
| 1045 | `Fish Land` | `kFishInPriority` | 412 |
| 1046 | `Dont Exit` | `kDontExitPriority` | 103 |
| 1047 | `Sizzle` | `kSizzlePriority` | 413 |
| 1048 | `Paper1` | `kPapersPriority` | 807 |
| 1049 | `Paper2` | `kPapersPriority` | 807 |
| 1050 | `Paper3` | `kPapersPriority` | 807 |
| 1051 | `Paper4` | `kPapersPriority` | 807 |
| 1052 | `Keystroke` | `kTypingPriority` | 808 |
| 1053 | `Carriage Return` | `kCarriagePriority` | 809 |
| 1054 | `Check` | `kChord2Priority` | 810 |
| 1055 | `Phone` | `kPhoneRingPriority` | 500 |
| 1056 | `Ding 1` | `kChime1Priority` | 203 |
| 1057 | `Ding 2` | `kChime2Priority` | 204 |
| 1058 | `Twunk` | `kWebTwangPriority` | 310 |
| 1059 | `TransOut` | `kTransOutPriority` | 906 |
| 1060 | `TransIn` | `kTransInPriority` | 905 |
| 1061 | `Bonus` | `kBonusPriority` | 812 |
| 1062 | `Hiss` | `kHissPriority` | 311 |
| — (index 63) | *house-supplied* | `kTriggerPriority` | 999 |
| 2000 | `Refrain1.22` | (music, no priority) | — |
| 2001 | `Refrain2.22` | (music) | — |
| 2002 | `Refrain3.22` | (music) | — |
| 2003 | `Refrain4.22` | (music) | — |
| 2004 | `Chorus.22` | (music) | — |
| 2005 | `RefrainSparse1.22` | (music) | — |
| 2006 | `RefrainSparse2.22` | (music) | — |

The music names document the composition structure: four "Refrain" variants, two
"RefrainSparse" variants, and one double-length "Chorus". The `.22` suffix is the sample rate
in kHz, consistent with the verified 22254.5455 Hz. `kTriggerPriority` = **999**, the highest
in the game, which is why the house-supplied trigger sound always interrupts.

---

## 3.3 Dialogs: `'DLOG'` 28, `'ALRT'` 26, `'DITL'` 54, `'dctb'` 20, `'DLGX'` 1, `'ictb'` 1

### 3.3.1 `'DLOG'` binary layout — verified

The classic `DialogTemplate` is 20 bytes plus a Pascal title string. **This fork's records
are 21 bytes** (title = a single length byte 0), except DLOG 150 which is 24. Verified field
map:

| Offset | Size | Field | Notes |
| ---: | ---: | --- | --- |
| 0 | 8 | `boundsRect` | top, left, bottom, right — `int16` big-endian, **global screen coords** |
| 8 | 2 | `procID` | window definition; `1` = `dBoxProc` in all 28 |
| 10 | 1 | `visible` | + 1 filler byte |
| 12 | 1 | `goAwayFlag` | + 1 filler byte |
| 14 | 4 | `refCon` | `0` in all 28 |
| 18 | 2 | `itemsID` | the DITL resource ID; **equals the DLOG ID in all 28** |
| 20 | 1..n | `title` | Pascal string; length byte `0` in all 28 |

All 28 records:

| DLOG | Len | boundsRect (t,l,b,r) | WxH | procID | vis | goAway | itemsID | Owner |
| ---: | ---: | --- | --- | ---: | ---: | ---: | ---: | --- |
| 150 | **24** | 62, 62, 242, 446 | 384x180 | 1 | 1 | 1 | 150 | About (`Sources/About.c:34`) |
| 1000 | 21 | 36, 16, 300, 446 | 430x264 | 1 | 0 | 0 | 1000 | Load House (`Sources/SelectHouse.c`) |
| 1001 | 21 | 32, 68, 357, 380 | 312x325 | 1 | 0 | 0 | 1001 | House Info (`Sources/HouseInfo.c`) |
| 1003 | 21 | 0, 0, 267, 384 | 384x267 | 1 | 0 | 0 | 1003 | Room Info (`Sources/RoomInfo.c`) |
| 1007 | 21 | 0, 0, 167, 256 | 256x167 | 1 | 0 | 0 | 1007 | Object Info — blower |
| 1010 | 21 | 0, 0, 120, 256 | 256x120 | 1 | 0 | 0 | 1010 | Object Info — read-only |
| 1011 | 21 | 0, 0, 187, 256 | 256x187 | 1 | 0 | 0 | 1011 | Object Info — switch w/ link |
| 1012 | 21 | 40, 40, 170, 328 | 288x130 | 1 | 1 | 1 | 1012 | Preferences root (`Sources/Settings.c`) |
| 1013 | 21 | 0, 0, 143, 256 | 256x143 | 1 | 0 | 0 | 1013 | Object Info — on/off |
| 1014 | 21 | 0, 0, 153, 256 | 256x153 | 1 | 0 | 0 | 1014 | Object Info — interval |
| 1015 | 21 | 0, 0, 155, 256 | 256x155 | 1 | 0 | 0 | 1015 | Object Info — point value |
| 1016 | 21 | 0, 0, 144, 256 | 256x144 | 1 | 1 | 1 | 1016 | `kOriginalArtDialogID` (`Sources/RoomInfo.c:19`) |
| 1017 | 21 | 40, 40, 280, 373 | 333x240 | 1 | 0 | 0 | 1017 | Display prefs |
| 1018 | 21 | 40, 40, 216, 356 | 316x176 | 1 | 0 | 0 | 1018 | Sound prefs |
| 1019 | 21 | 0, 0, 146, 256 | 256x146 | 1 | 0 | 0 | 1019 | Object Info — grease |
| 1020 | 21 | 0, 0, 109, 316 | 316x109 | 1 | 1 | 1 | 1020 | High-score name entry (`Sources/HighScores.c`) |
| 1021 | 21 | 40, 40, 162, 356 | 316x122 | 1 | 1 | 1 | 1021 | High-score #1 entry |
| 1022 | 21 | 0, 0, 185, 256 | 256x185 | 1 | 0 | 0 | 1022 | Object Info — transport |
| 1023 | 21 | 40, 40, 216, 356 | 316x176 | 1 | 0 | 0 | 1023 | Control prefs |
| 1024 | 21 | 40, 40, 232, 356 | 316x192 | 1 | 0 | 0 | 1024 | Brains prefs |
| 1025 | 21 | 40, 40, 164, 352 | 312x124 | 1 | 0 | 0 | 1025 | `kResumeGameDial` (`Sources/Menu.c:712`, `GetNewDialog` `:738`) |
| 1026 | 21 | 50, 50, 154, 362 | 312x104 | 1 | 0 | 0 | 1026 | `kMasterDialogID` (`Sources/Validate.c:18`, `:337`) |
| 1027 | 21 | 0, 0, 160, 256 | 256x160 | 1 | 0 | 0 | 1027 | Object Info — idle delay |
| 1033 | 21 | 0, 0, 175, 256 | 256x175 | 1 | 0 | 0 | 1033 | Object Info — flower picker |
| 1034 | 21 | 0, 0, 187, 256 | 256x187 | 1 | 0 | 0 | 1034 | Object Info — delay + link |
| 1035 | 21 | 0, 0, 160, 278 | 278x160 | 1 | 0 | 0 | 1035 | Object Info — microwave |
| 1043 | 21 | 40, 40, 201, 320 | 280x161 | 1 | 1 | 1 | 1043 | `kGoToDialogID` (`Sources/House.c:18`) |
| 1045 | 21 | 0, 0, 141, 256 | 256x141 | 1 | 0 | 0 | 1045 | Object Info — custom PICT/sound |

**DLOG 150 is the only anomaly.** Its 24 raw bytes are:

```
00 3E 00 3E 00 F2 01 BE   boundsRect = 62,62,242,446
00 01                     procID = 1 (dBoxProc)
01 00                     visible = 1, filler
01 00                     goAwayFlag = 1, filler
00 00 00 00               refCon = 0
00 96                     itemsID = 150
00                        title length = 0
00 02 2F                  <-- 3 extra bytes
```

The three trailing bytes are one pad byte and the 16-bit value `0x022F` (559). This matches
**none** of the documented `kWindow*` auto-position constants (`0x280A`, `0x300A`, `0x380A`,
`0xA80A`, `0xB00A`, `0xB80A`, `0x080A`, `0x100A`, `0x180A`, `0x200A`), and no code reads past
the title. It is almost certainly ResEdit residue. **A port should ignore bytes 21..23 of
DLOG 150** and use `boundsRect` verbatim. See "Open questions".

Sixteen of the 28 DLOGs have `boundsRect` origin (0,0) — 1003, 1007, 1010, 1011, 1013, 1014,
1015, 1016, 1019, 1020, 1022, 1027, 1033, 1034, 1035, 1045. Fourteen of those sixteen also
have `visible = 0`; the exceptions are 1016 and 1020, which are `visible = 1` despite the
(0,0) origin. These are the object-info dialogs, which are positioned by the caller before
showing. The other twelve have a hard-coded top-left, of which `(40,40)` occurs eight times
(1012, 1017, 1018, 1021, 1023, 1024, 1025, 1043). Across all 28, `visible = 0` in 22 and
`visible = 1` in 6 (150, 1012, 1016, 1020, 1021, 1043).

> **Crucially, there is no dialog centring.** Every positioning helper in
> `GliderPRO/Sources/DialogUtils.c` is commented out: `GetPutDialogCorner` (`:39-66`),
> `GetGetDialogCorner` (`:73-100`), `CenterDialog` (`:105-132`), `TrueCenterDialog`
> (`:157-187`), `CenterAlert` (`:192-219`), `ZoomOutDialogRect` (`:226-274`),
> `ZoomOutAlertRect` (`:280-328`). The one live helper `GetDialogRect` (`:137-151`) has zero
> callers. And `BringUpDialog` itself has the centring call commented out:
>
> ```
> GliderPRO/Sources/DialogUtils.c:24-33
>     void BringUpDialog (DialogPtr *theDialog, short dialogID)
>     {
>     //  CenterDialog(dialogID);
>         *theDialog = GetNewDialog(dialogID, nil, kPutInFront);
>         if (*theDialog == nil) RedAlert(kErrDialogDidntLoad);
>         SetPort((GrafPtr)*theDialog);
>         ShowWindow((WindowPtr)*theDialog);
>         DrawDefaultButton(*theDialog);
>     }
> ```
>
> **So the shipped behaviour is: dialogs appear at exactly their `boundsRect` in global screen
> coordinates.** A Go port that centres dialogs will not match the original. Reproduce the
> literal coordinates.

`kPutInFront` is `(WindowRef)-1L` (see `GliderPRO/Sources/About.c:51` which spells it out).

### 3.3.2 `'ALRT'` binary layout — verified

All 26 are **exactly 12 bytes**. There is no title, no filler, no colour flag.

| Offset | Size | Field |
| ---: | ---: | --- |
| 0 | 8 | `boundsRect` (t, l, b, r) |
| 8 | 2 | `itemsID` — the DITL ID; **equals the ALRT ID in all 26** |
| 10 | 2 | `stages` — packed 4x4-bit alert-stage word |

| ALRT | boundsRect | WxH | stages | Constant | Cite |
| ---: | --- | --- | --- | --- | --- |
| 130 | 0, 0, 96, 274 | 274x96 | `0x4444` | `kSwitchDepthAlert` | `Sources/Environ.c:18`, `Alert` `:414` |
| 140 | 92, 60, 220, 446 | 386x128 | `0x5555` | `rFileErrorAlert` | `Sources/FileError.c`, `Alert` `:97` |
| 160 | 40, 40, 140, 340 | 300x100 | `0x5555` | `kNewPrefsAlertID` | `Sources/Prefs.c:23`, `Alert` `:279` |
| 170 | 92, 84, 220, 432 | 348x128 | `0x5555` | `rDeathAlertID` | `Sources/Utilities.c:159` |
| 180 | 40, 40, 156, 334 | 294x116 | `0x5555` | `kSetMemoryAlert` | `Sources/Environ.c:19`, `Alert` `:687` |
| 181 | 40, 40, 168, 328 | 288x128 | `0x5555` | `kLowMemoryAlert` | `Sources/Environ.c:20`, `Alert` `:682` |
| 1002 | 0, 0, 128, 320 | 320x128 | `0x5555` | `kSaveChangesAlert` | `Sources/HouseIO.c:20` |
| 1004 | 40, 40, 136, 328 | 288x96 | `0x5555` | `kNewRoomAlert` | `Sources/Map.c:23` |
| 1005 | 40, 40, 136, 328 | 288x96 | `0x5555` | `kDeleteRoomAlert` | `Sources/Room.c:16` |
| 1006 | 40, 40, 168, 340 | 300x128 | `0x5555` | `kYellowAlert` | `Sources/HouseIO.c:642-654` |
| 1008 | 40, 40, 136, 320 | 280x96 | `0x5555` | `kNoMoreObjectsAlert` | `Sources/ObjectAdd.c:15` |
| 1009 | 0, 0, 150, 346 | 346x150 | `0x4444` | `kHouseBannerAlert` | `Sources/Play.c:18` |
| 1028 | 40, 40, 148, 320 | 280x108 | `0x5555` | `kNoMoreSpecialAlert` | `Sources/ObjectAdd.c:16` |
| 1029 | 40, 40, 164, 338 | 298x124 | **`0xFFFF`** | `kLockHouseAlert` | `Sources/HouseInfo.c:23` |
| 1030 | 40, 40, 148, 314 | 274x108 | `0x4444` | `kNoSoundManager3Alert` | `Sources/Sound.c:529`, `Alert` `:533` |
| 1031 | 40, 40, 112, 308 | 268x72 | `0x5555` | `kNoPrintingAlert` | `Sources/AppleEvents.c:14` |
| 1032 | 40, 40, 129, 320 | 280x89 | `0x4444` | `kZeroScoresAlert` | `Sources/HouseInfo.c:24` |
| 1036 | 40, 40, 148, 314 | 274x108 | `0x4444` | `kNoPICTFoundAlert` | `Sources/RoomInfo.c:20` |
| 1037 | 40, 40, 164, 326 | 286x124 | `0x4444` | `kNotInDemoAlert` | `Sources/Menu.c:769`, `Alert` `:773` |
| 1038 | 54, 96, 162, 370 | 274x108 | `0x4444` | `kNoMemForMusicAlert` | `Sources/Music.c:413` |
| 1039 | 54, 96, 162, 370 | 274x108 | `0x4444` | `kNoMemForSoundsAlert` | `Sources/Sound.c:518`, `Alert` `:522` |
| 1040 | 54, 96, 126, 377 | 281x72 | `0x4444` | `kChangesEffectAlert` | `Sources/Settings.c:1476`, `Alert` `:1480` |
| 1041 | 40, 40, 116, 302 | 262x76 | **`0xCCCC`** | `kSaveGameAlert` | `Sources/Input.c:385`, `Alert` `:392` |
| 1042 | 40, 40, 148, 296 | 256x108 | `0x5555` | **unreachable** — see below | — |
| 1044 | 40, 40, 168, 340 | 300x128 | `0x5555` | `kSavedGameErrorAlert` | `Sources/SavedGames.c:154`, `Alert` `:162` |
| 1046 | 40, 40, 132, 300 | 260x92 | `0x4444` | `kNoHighScoreAlert` | `Sources/Menu.c:781`, `Alert` `:785` |

**`stages` decoding.** The 16-bit word packs four 4-bit stage descriptors, stage 4 in the
high nibble down to stage 1 in the low nibble. Each nibble is
`bit3 = boldItm-1 (default button is item 2 rather than 1)`,
`bit2 = boxDrwn (draw the alert box)`, `bits1-0 = sound number (0..3)`.

| Word | Every stage's nibble | Meaning |
| --- | --- | --- |
| `0x4444` | `0100` | draw box, **no** sound, default = item 1. Used by 10 alerts. |
| `0x5555` | `0101` | draw box, **sound 1** (one beep), default = item 1. Used by 14 alerts. |
| `0xCCCC` | `1100` | draw box, no sound, **default = item 2** ("Don't Save" is default in ALRT 1041). |
| `0xFFFF` | `1111` | draw box, sound 3, default = item 2 ("Don't Lock" is default in ALRT 1029). |

So a Go port needs: for each alert, "beep or not" and "which button is the default"; the
per-stage escalation is never observed (all four nibbles are always equal).

**ALRT 1042 is dead.** `GliderPRO/Sources/Input.c:19` declares
`#define kSavingGameDial 1042` and the identifier is never used again anywhere in
`Sources/` or `Headers/`. There is no `'DLOG'` 1042 either, so 1042 exists only as an alert
plus DITL 1042 ("You switched the number of colors…", buttons Quit / Restore, icon 130).
It appears to be an abandoned second copy of the depth-switch alert. **Do not port it.**

### 3.3.3 `'DITL'` binary layout — verified byte-exact for all 54

```
+0   int16   itemCount - 1        (so 0 means one item)
then, repeated (itemCount) times, each item aligned to an even offset:
+0   int32   placeholder          (0 in the resource; the Dialog Manager overwrites it
                                   with the item's Handle/ProcPtr at run time)
+4   Rect    displayRect          top, left, bottom, right (int16 x4)
+12  uint8   itemType             bit 7 = itemDisable; bits 0-6 = type code
+13  uint8   dataLength
+14  n bytes data                 padded with one 0 byte if dataLength is odd
```

Type codes observed in this fork:

| Code | Constant | `data` contents | Count in fork |
| ---: | --- | --- | ---: |
| 0 | `userItem` | empty | 74 |
| 4 | `ctrlItem + btnCtrl` (button) | raw (non-Pascal) title text | 113 |
| 5 | `ctrlItem + chkCtrl` (checkBox) | title text | 26 |
| 6 | `ctrlItem + radCtrl` (radioButton) | title text | 19 |
| 7 | `ctrlItem + resCtrl` | `int16` CNTL resource ID | 1 |
| 8 | `statText` | the text | 115 |
| 16 | `editText` | initial text | 13 |
| 32 | `iconItem` | `int16` ICON/cicn resource ID | 42 |
| 64 | `picItem` | `int16` PICT resource ID | 25 |

Totals: **428 items across 54 DITLs** (74 + 113 + 26 + 19 + 1 + 115 + 13 + 42 + 25 = 428),
and every DITL's bytes are fully consumed with no slack (verified for all 54). Type codes
1 (`helpItem`), 2, 3 and 9-15 never appear.

`itemDisable` (bit 7, `0x80`) is set on **217** of the 428 items; 211 are enabled. The
disabled ones are all the static text, icons and pictures, plus most user items. Disabled
items do not respond to clicks; the Dialog Manager still draws them.

The `data` for a button/checkBox/radioButton/statText/editText item is **raw text with a
leading length byte supplied by `dataLength`, not a Pascal string** — i.e. the length lives
in the item header, and `data` is the characters. Text may contain embedded `\r` (`0x0D`) for
a line break, e.g. DITL 1014 item 9 is `Interval:\r(1/10 sec)` and DITL 1017 item 6 is
`Number of Rooms to Display:\r(the less rooms, the faster)`.

The single `resCtrl` is **DITL 1003 item 11 → CNTL 128**, the Room-Info backgrounds pop-up
menu.

### 3.3.4 Complete DITL item inventory

`!` prefix marks `itemDisable`. Text is truncated at 32 characters here; the full strings are
recoverable with `probe_rez.py dump DITL <id>`.

| DITL | Items | Item list |
| ---: | ---: | --- |
| 130 | 5 | 1:button:`256` \| 2:button:`16` \| 3:button:`Quit` \| 4:!icon:130 \| 5:!statText:`Glider PRO™ requires 256 colors…` |
| 140 | 5 | 1:button:`Okay` \| 2:!statText:`^0` \| 3:!statText:`(error = ^1)` \| 4:!icon:140 \| 5:!statText:`A File Error Loading/Saving ^2` |
| 150 | 9 | 1:!picture:150 \| 2:!statText:`` \| 3:!icon:150 \| 4:!picture:153 \| 5:!statText:`by john calhoun` \| 6:!statText:`© 1994-2000 Casady & Greene, Inc.` \| 7:!userItem \| 8:!userItem \| 9:!userItem |
| 160 | 3 | 1:button:`Okay` \| 2:!statText:`You have a new Preferences file…` \| 3:!icon:160 |
| 170 | 5 | 1:button:`Okay` \| 2:!statText:`^0` \| 3:!statText:`^1` \| 4:!statText:`(program error = ^2)` \| 5:!icon:170 |
| 180 | 4 | 1:button:`Quit` \| 2:!statText:`Glider PRO™ requires more memor…` \| 3:!icon:180 \| 4:!statText:`(We need about: ^0K)` |
| 181 | 2 | 1:button:`Okay` \| 2:!statText:`Glider PRO™ Demo requires more …` |
| 1000 | 30 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!picture:1001 \| 4:!userItem \| 5-28:userItem (24 house-list rows) \| 29:icon:1050 \| 30:icon:1051 |
| 1001 | 18 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`v. ^0.^1` \| 4:editText \| 5:!statText:`Number of rooms: ^2` \| 6:button:`Lock House` \| 7:!userItem \| 8:!picture:1005 \| 9:button:`Clear Scores` \| 10:!userItem \| 11:editText \| 12:!statText:`Opening Message:` \| 13:!statText:`Finished House Message:` \| 14:checkBox:`No Phone` \| 15:!statText:`()` \| 16:!statText:`()` \| 17:!statText:`Highest Score:` \| 18:!statText |
| 1002 | 5 | 1:button:`Save` \| 2:button:`Discard` \| 3:button:`Cancel` \| 4:!statText:`You have made changes to ^0 and…` \| 5:!icon:1000 |
| 1003 | 19 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:editText \| 4:!statText:`Floor: ^0` \| 5:!statText:`Suite: ^1` \| 6:!userItem \| 7:!statText:`Location:` \| 8:!statText:`Room Name:` \| 9:!statText:`Number of Objects: ^2` \| 10:!userItem \| **11:resCtrl:128** \| 12:!userItem \| 13:!picture:1007 \| 14:!statText:`Tiles:` \| 15:!userItem \| 16:!picture:1009 \| 17:checkBox:`First Room` \| 18:!statText \| 19:button:`Bounds` |
| 1004 | 4 | 1:button:`Create` \| 2:button:`Cancel` \| 3:!statText:`Do you wish to create a New Roo…` \| 4:!icon:1004 |
| 1005 | 4 | 1:button:`Delete` \| 2:button:`Cancel` \| 3:!statText:`Do you really want to delete th…` \| 4:!icon:1005 |
| 1006 | 5 | 1:button:`Okay` \| 2:!statText:`^0` \| 3:!statText:`A problem came up:` \| 4:!statText:`Error #: ^1` \| 5:!icon:1006 |
| 1007 | 17 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`Object Number: ^0` \| 4:!statText:`Object Kind: ^1` \| 5:!userItem \| 6:checkBox:`Initially On` \| 7:checkBox:`Extra Forceful` \| 8:!userItem \| 9:!statText:`Direction:` \| 10:!picture:1002 \| 11-14:userItem \| 15:button:`Linked From?` \| 16:radioButton:`Left Facing` \| 17:radioButton:`Right Facing` |
| 1008 | 3 | 1:button:`Okay` \| 2:!statText:`A room can have no more than 24…` \| 3:!icon:1008 |
| 1009 | 4 | 1:button:`Begin` \| 2:!statText:`^0` \| 3:!statText:`^1` \| 4:!icon:1000 |
| 1010 | 6 | 1:button:`Okay` \| 2:!statText:`Object Number: ^0` \| 3:!statText:`Object Kind: ^1` \| 4:!userItem \| 5:!picture:1002 \| 6:button:`Linked From?` |
| 1011 | 15 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`Object Kind: ^1` \| 4:!userItem \| 5:!statText:`Object Number: ^0` \| 6:radioButton:`Toggle` \| 7:radioButton:`Force On` \| 8:radioButton:`Force Off` \| 9:button:`Link` \| 10:!picture:1002 \| 11:!statText:`Room Link: ^2` \| 12:!statText:`Object Link: ^3` \| 13:!userItem \| 14:button:`Go To` \| 15:button:`Linked From?` |
| 1012 | 11 | 1:button:`Okay` \| 2:!picture:1013 \| 3:!icon:1010 \| 4:!icon:1011 \| 5:!icon:1012 \| 6:!icon:1013 \| 7-10:!userItem \| 11:button:`All Defaults` |
| 1013 | 8 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`Object Number: ^0` \| 4:!statText:`Object Kind: ^1` \| 5:!userItem \| 6:checkBox:`Initially On` \| 7:!picture:1002 \| 8:button:`Linked From?` |
| 1014 | 10 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`Object Number: ^0` \| 4:!statText:`Object Kind: ^1` \| 5:!userItem \| 6:checkBox:`Initially On` \| 7:!picture:1002 \| 8:editText \| 9:!statText:`Interval:\r(1/10 sec)` \| 10:button:`Linked From?` |
| 1015 | 9 | 1:button:`Okay` \| 2:!statText:`Object Number: ^0` \| 3:!statText:`Object Kind: ^1` \| 4:!userItem \| 5:!picture:1002 \| 6:radioButton:`100 Points` \| 7:radioButton:`300 Points` \| 8:radioButton:`500 Points` \| 9:button:`Linked From?` |
| 1016 | 12 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`Enter the ID number of the PICT…` \| 4:!statText:`PICT ID:` \| 5:editText:`3000` \| 6:!statText:`(3000 - 3499)` \| 7-10:!userItem \| 11:!statText:`Bounded` \| 12:checkBox:`Floor Support` |
| 1017 | 17 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:icon:1020 \| 4:icon:1021 \| 5:icon:1022 \| 6:!statText:`Number of Rooms to Display:\r(t…` \| 7:!picture:1006 \| 8:!userItem \| 9:checkBox:`Beautiful opening color fade` \| 10:radioButton:`Use current depth when possible` \| 11:radioButton:`Always play in 256 colors` \| 12:radioButton:`Always play in 16 grays` \| 13:!userItem \| 14:!userItem \| 15:button:`Defaults` \| 16:checkBox:`Use Quickdraw™ (slower)` \| 17:checkBox:`Run on second monitor` |
| 1018 | 13 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!icon:1030 \| 4:icon:1032 \| 5:icon:1031 \| 6:!statText:`Volume:` \| 7:!statText \| 8:checkBox:`Play music when idle` \| 9:checkBox:`Play music during game` \| 10:!statText:`Set the game volume and backgro…` \| 11:!userItem \| 12:!picture:1008 \| 13:button:`Defaults` |
| 1019 | 8 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`Object Number: ^0` \| 4:!statText:`Object Kind: ^1` \| 5:!userItem \| 6:checkBox:`Grease initially tipped` \| 7:!picture:1002 \| 8:button:`Linked From?` |
| 1020 | 6 | 1:button:`Okay` \| 2:editText:`Your Name` \| 3:!statText:`Your score of ^0 is #^1 on the …` \| 4:!statText:`Enter your name:\r(15 letters m…` \| 5:!statText \| 6:!statText:`letters` |
| 1021 | 5 | 1:button:`Okay` \| 2:editText \| 3:!statText:`Getting #1 on the high scores e…` \| 4:!statText:`letters` \| 5:!statText |
| 1022 | 13 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`Object Kind: ^1` \| 4:!userItem \| 5:!statText:`Object Number: ^0` \| 6:button:`Link` \| 7:!picture:1002 \| 8:!statText:`Room Link: ^2` \| 9:!statText:`Object Link: ^3` \| 10:!userItem \| 11:button:`Go To` \| 12:button:`Linked From?` \| 13:checkBox:`Initially On` |
| 1023 | 15 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!userItem \| 4:!picture:1012 \| 5:icon:1040 \| 6:icon:1041 \| 7:icon:1042 \| 8:icon:1043 \| 9-12:!userItem \| 13:button:`Defaults` \| 14:radioButton:`Esc Pauses Game` \| 15:radioButton:`Tab Pauses Game` |
| 1024 | 15 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!userItem \| 4:!picture:1014 \| 5:editText \| 6:!statText:`Maximum houses displayed (12-50…` \| 7:checkBox:`Quick Transitions` \| 8:checkBox:`Zoom Windows` \| 9:button:`Defaults` \| 10:checkBox:`Automatic Demo` \| 11:checkBox:`Background Tasks` \| 12:checkBox:`Error-Check House` \| 13:checkBox:`Use "Pretty Map"` \| 14:checkBox:`Do Create Dialog` \| 15:!statText:`Editor Options:` |
| 1025 | 5 | 1:button:`New Game` \| 2:button:`Resume` \| 3:!statText:`(You had ^0 glider^1 & ^2 points.)` \| 4:!statText:`You have a saved game.  Beginni…` \| 5:!icon:1001 |
| 1026 | 4 | 1:button:`Cancel` \| 2:!statText:`Insert your original Glider PRO…` \| 3:!statText:`Your orginal disk is required o…` \| 4:!icon:140 |
| 1027 | 11 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`Object Kind: ^1` \| 4:!userItem \| 5:!statText:`Object Number: ^0` \| 6:!picture:1002 \| 7:editText \| 8:!statText:`Idle Delay:` \| 9:!statText:`(1/10 secs.)` \| 10:checkBox:`Initially "On"` \| 11:button:`Linked From?` |
| 1028 | 3 | 1:button:`Okay` \| 2:!statText:`There is an absolute maximum nu…` \| 3:!icon:1008 |
| 1029 | 4 | 1:button:`Lock It!` \| 2:button:`Don't Lock` \| 3:!statText:`Wait!!!  If you lock this house…` \| 4:!icon:1060 |
| 1030 | 3 | 1:button:`Okay` \| 2:!statText:`Where is Sound Manager 3.0?  I …` \| 3:!icon:1070 |
| 1031 | 3 | 1:button:`Bye!` \| 2:!statText:`Glider PRO doesn't know what th…` \| 3:!icon:900 |
| 1032 | 5 | 1:button:`Cancel` \| 2:button:`Clear All` \| 3:button:`All But #1` \| 4:!statText:`Do what?  You can clear all but…` \| 5:!icon:910 |
| 1033 | 13 | 1:button:`Okay` \| 2:!statText:`Object Number: ^0` \| 3:!statText:`Object Kind: ^1` \| 4:!userItem \| 5:!picture:1002 \| 6:radioButton:`Dandelion` \| 7:radioButton:`Tulip` \| 8:radioButton:`Orchid` \| 9:radioButton:`Violets` \| 10:radioButton:`Daisies` \| 11:radioButton:`Sunflower` \| 12:button:`Cancel` \| 13:button:`Linked From?` |
| 1034 | 15 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`Object Kind: ^1` \| 4:!userItem \| 5:!statText:`Object Number: ^0` \| 6:editText \| 7:!statText:`Delay:` \| 8:!statText:`(1/10 secs)` \| 9:button:`Link` \| 10:!picture:1002 \| 11:!statText:`Room Link: ^2` \| 12:!statText:`Object Link: ^3` \| 13:!userItem \| 14:button:`Go To` \| 15:button:`Linked From?` |
| 1035 | 11 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`Object Number: ^0` \| 4:!statText:`Object Kind: ^1` \| 5:!userItem \| 6:checkBox:`Initially On` \| 7:!picture:1022 \| 8:checkBox:`Zero Bands` \| 9:checkBox:`Zero Battery/He` \| 10:checkBox:`Zero Foil` \| 11:button:`Linked From?` |
| 1036 | 3 | 1:button:`Okay` \| 2:!statText:`Where is your PICT?  I'm going …` \| 3:!icon:1071 |
| 1037 | 2 | 1:button:`Okay` \| 2:!statText:`This feature is not in the Demo…` |
| 1038 | 3 | 1:button:`Whatever` \| 2:!statText:`Okay, with your monitor & color…` \| 3:!icon:1011 |
| 1039 | 3 | 1:button:`Whatever` \| 2:!statText:`Okay, with your monitor & color…` \| 3:!icon:1011 |
| 1040 | 2 | 1:button:`Okay` \| 2:!statText:`Some preference changes will no…` |
| 1041 | 4 | 1:button:`Save First` \| 2:button:`Don't Save` \| 3:!statText:`Do you want to save the state o…` \| 4:!icon:1072 |
| 1042 | 4 | 1:button:`Quit` \| 2:button:`Restore` \| 3:!statText:`You switched the number of colo…` \| 4:!icon:130 |
| 1043 | 10 | 1:button:`Cancel` \| 2:button:`Go to First Room` \| 3:button:`Go to Previous Room` \| 4:button:`Go to Room…` \| 5:editText \| 6:editText \| 7:!statText:`floor:` \| 8:!statText:`suite:` \| 9:!picture:1023 \| 10:!userItem |
| 1044 | 3 | 1:button:`Okay` \| 2:!statText:`This saved game was saved for t…` \| 3:!icon:1006 |
| 1045 | 9 | 1:button:`Okay` \| 2:button:`Cancel` \| 3:!statText:`Object Kind: ^1` \| 4:!statText:`Object Number: ^0` \| 5:!userItem \| 6:!picture:1002 \| 7:editText \| 8:!statText:`^2 ID:` \| 9:!statText:`(I.D. > ^3)` |
| 1046 | 3 | 1:button:`So What?` \| 2:!statText:`If you resume a saved game, you…` \| 3:!icon:1073 |

`^0`..`^3` are `ParamText` substitution markers; the Dialog Manager replaces them at draw
time with the four strings from the most recent `ParamText` call. A Go port must implement
the same four-slot global substitution, or refactor to explicit formatting — but note that
DLOG 1045 *depends* on the shared-DITL/`ParamText` trick to serve as both the custom-PICT and
the custom-sound dialog (`GliderPRO/Sources/ObjectInfo.c:1186`, `:1188`).

Two notable strings inside the DITLs:

* **DITL 150 item 6 says `© 1994-2000 Casady & Greene, Inc.`** while `'vers'` 1 and 2 say
  `© 1994-95` (§3.10.2). The About box and the Finder's Get Info disagree in the shipped app.
* **DITL 1016 item 6 says `(3000 - 3499)`** while `kUserBackground 3000` and
  `kUserStructureRange 3300` (`GliderPRO/Headers/GliderDefines.h:522-523`) imply the range is
  3000-3299. Neither matches the code that runs: `GliderPRO/Sources/RoomInfo.c:762` accepts
  `3000 <= id < 3800`. Port the *validation*, not the label or the constant — see
  "Open questions" #4.

### 3.3.5 Which DITL items each subsystem reads by index

The two settings dialogs use icon items as clickable "tab" buttons, and the code derives
their rects from the items:

```
GliderPRO/Sources/Settings.c:1407-1414
	GetDialogItemRect(prefDlg, 3, &prefButton[0]);  InsetRect(&prefButton[0], -4, -4);
	GetDialogItemRect(prefDlg, 4, &prefButton[1]);  InsetRect(&prefButton[1], -4, -4);
	GetDialogItemRect(prefDlg, 5, &prefButton[2]);  InsetRect(&prefButton[2], -4, -4);
	GetDialogItemRect(prefDlg, 6, &prefButton[3]);  InsetRect(&prefButton[3], -4, -4);
GliderPRO/Sources/Settings.c:1297-1300
	ColorFrameRect(&prefButton[0..3], kRedOrangeColor8);
```

`kRedOrangeColor8` is **23** (`GliderPRO/Headers/GliderDefines.h:542`) — an index into the
Mac 8-bit system palette, not an RGB triple. §5.2 covers the palette problem.

The click-flash uses a second icon family:

```
GliderPRO/Sources/Settings.c:1265-1276   FlashSettingsButton(short who)
	#define kNormalSettingsIcon    1010
	#define kInvertedSettingsIcon  1014
	theID = kInvertedSettingsIcon + who;
	DrawCIcon(theID, prefButton[who].left + 4, prefButton[who].top + 4);
	DelayTicks(8);
	theID = kNormalSettingsIcon + who;
	DrawCIcon(theID, prefButton[who].left + 4, prefButton[who].top + 4);
```

So `cicn` 1010-1013 are the four normal tab icons (also placed as DITL 1012 items 3-6) and
`cicn` **1014-1017 are the inverted versions, which exist only as `cicn`** — there is no ICON
1014-1017. `DelayTicks(8)` is 8/60 s ≈ 133 ms of flash.

The tab names come from `'STR#'` 129 "Prefmain": Display, Sounds, Controls, Brains, read at
`GliderPRO/Sources/Settings.c:1288-1294`.

### 3.3.6 `'dctb'` — 20 dialog colour tables, all byte-identical

Each is 48 bytes:

| Offset | Size | Field | Observed |
| ---: | ---: | --- | --- |
| 0 | 4 | `ctSeed` | 0 |
| 4 | 2 | `ctFlags` | 0 |
| 6 | 2 | `ctSize` | 4 → **5 entries** |
| 8 | 8x5 | `ColorSpec[5]` | see below |

Each `ColorSpec` is `int16 value` + `RGBColor rgb` (3 x `uint16`). The five entries are the
window parts in the fixed order defined by the Window Manager:

| Index | Part | RGB | Colour |
| ---: | --- | --- | --- |
| 0 | `wContentColor` | FFFF FFFF FFFF | white |
| 1 | `wFrameColor` | 0000 0000 0000 | black |
| 2 | `wTextColor` | 0000 0000 0000 | black |
| 3 | `wHiliteColor` | 0000 0000 0000 | black |
| 4 | `wTitleBarColor` | FFFF FFFF FFFF | white |

That is the Mac default black-on-white dialog. **All 20 `'dctb'` are functionally no-ops**,
and 8 DLOGs have no `'dctb'` at all with no visible difference. **A Go port can ignore
`'dctb'` entirely.**

`'wctb'` **130** (48 bytes, same layout, `ctSize` 4) differs in one entry:
`wContentColor` is **black**, then black, black, black, white. It is the colour table for
`'WIND'` 130 (`kMenuWindowID`), giving that 640x20 strip a black background.

`'cctb'` **132** (40 bytes, `ctSize` **3** → 4 entries) is the control colour table for
`'CNTL'` 132: black, white, black, white. Since CNTL 132 is unused (§3.8.1), so is this.

`'ictb'` **150** (36 bytes) is **all zero bytes** — an item colour table for DITL 150 that
specifies nothing. Grep for `ictb` in `Sources/` and `Headers/`: zero hits. Inert.

### 3.3.7 `'DLGX'` 150 — 190 bytes, undocumented and inert

`'DLGX'` is the System 7 "extended dialog" resource (used with `NewFeaturesDialog`), which
carries per-item font/style overrides. This one begins:

```
offset 0x00:  08 43 68 61 72 63 6F 61 6C   Pascal string "Charcoal"
offset 0x09..0x3F: all zero
offset 0x40:  00 0C                        = 12
offset 0x42..:  a table with a 12-byte stride. Read as big-endian 16-bit words, every
               non-zero field in 0x42..0xBD is: 0x48=4, 0x4A=4, 0x50=9, 0x52=9, 0x5E=6,
               0x6A=8, 0x76=9, 0x82=6, 0x8E=6, 0x9A=10, 0xA6=10, 0xB2=10.
offset 0xB4..0xBD: 10 trailing zero bytes
```

`"Charcoal"` is the System 8 UI font name, and 12 is a point size, so this is a per-item font
override table for the About box. **Nothing in the source refers to it**: grep for `DLGX`,
`dlgx`, `NewFeaturesDialog`, `GetNewFeaturesDialog` across `Sources/` and `Headers/` returns
zero hits, and `Sources/About.c:51` uses plain `GetNewDialog`, which ignores `'DLGX'`. It is
dead weight left over from an authoring tool. **Do not port.** The exact byte layout of the
per-item records was not determined — see "Open questions".

### 3.3.8 Dialog resource names

The ResEdit names on the 53 named DITLs, 28 DLOGs, 26 ALRTs and 20 dctbs are the
author's own labels. They independently confirm the owner mapping in §3.3.1/§3.3.2 and
disambiguate the object-info family (`O -` prefix on the DLOG names).

| ID | `'DITL'` name | `'DLOG'` name | `'ALRT'` name | `'dctb'` name |
| ---: | --- | --- | --- | --- |
| 130 | `Switch Depth` | — | `Color Depth` | — |
| 140 | `File Error` | — | `File Error` | — |
| 150 | `About` | `About` | — | `About` |
| 160 | `Prefs` | — | `Prefs` | — |
| 170 | `Death Error` | — | `Death Error` | — |
| 180 | `Set Memory` | — | `Set Memory` | — |
| 181 | `Low Memory` | — | `Low Memory` | — |
| 1000 | `Load House` | `Load House` | — | `Load House` |
| 1001 | `House Info` | `House Info` | — | `House Info` |
| 1002 | `Save Changes?` | — | `Save Changes` | — |
| 1003 | `Room Info` | `Room Info` | — | `Room Info` |
| 1004 | `New Room?` | — | `New Room?` | — |
| 1005 | `Delete Room?` | — | `Delete Room?` | — |
| 1006 | `Yellow Alert` | — | `Yellow Alert` | — |
| 1007 | `Blower Info` | `O - Blower Info` | — | `Blower Info` |
| 1008 | `No More Objects` | — | `No More Objects` | — |
| 1009 | `Banner` | — | `Banner` | — |
| 1010 | `Furniture Info` | `O - Furniture Info` | — | `Object Info` |
| 1011 | `Switch Info` | `O - Switch Info` | — | `Switch Info` |
| 1012 | `Prefs` | `Prefs` | — | `Prefs` |
| 1013 | `Light Info` | `O - Light Info` | — | `Light Info` |
| 1014 | `Appliance Info` | `O - Appliance Info` | — | `Appliance Info` |
| 1015 | `Invis Bonus Info` | `O - Invis Bonus Info` | — | `Invis Bonus Info` |
| 1016 | `Original Artwork` | `Select Original Art` | — | — |
| 1017 | `Display Prefs` | `Display Prefs` | — | `Display Prefs` |
| 1018 | `Sound Prefs` | `Sound Prefs` | — | `Sound Prefs` |
| 1019 | `Grease Object Info` | `O - Grease Object Info` | — | `Grease Object Info` |
| 1020 | `High Name` | `High Name` | — | — |
| 1021 | `High Banner` | `Banner` | — | — |
| 1022 | `Transport Info` | `O - Transport Info` | — | `Transport Info` |
| 1023 | `Controls Prefs` | `Controls Prefs` | — | `Controls Prefs` |
| 1024 | `Brains Prefs` | `Brains Prefs` | — | `Brains Prefs` |
| 1025 | `Resume` | `Resume Game` | — | — |
| 1026 | `Valid` | `Valid` | — | — |
| 1027 | `Enemy Info` | `O - Enemy Info` | — | `Enemy Info` |
| 1028 | `Too Many Objects` | — | `No More Special` | — |
| 1029 | `Lock House` | — | `Lock House` | — |
| 1030 | `Sound Manager` | — | `Sound Manager` | — |
| 1031 | `No Printing` | — | `No Printing` | — |
| 1032 | `High Scores` | — | `Clear Scores` | — |
| 1033 | `Flower Info` | `O - Flower Info` | — | `Flower Info` |
| 1034 | `Trigger Info` | `O - Trigger Info` | — | `Trigger Info` |
| 1035 | `Microwave Info` | `O - Microwave Info` | — | — |
| 1036 | `PICT Gone` | — | `PICT Gone` | — |
| 1037 | `Demo` | — | `Not In Demo` | — |
| 1038 | `Music Mem` | — | `Music Mem` | — |
| 1039 | `Sound Mem` | — | `Sound Mem` | — |
| 1040 | `Changes` | — | `Changes` | — |
| 1041 | `Save Game?` | — | `Save Game?` | — |
| 1042 | `Monitors Switched` | — | `Color Switched` | — |
| 1043 | `Go To Room…` | `Go To Room…` | — | — |
| 1044 | `Saved Game Mismatch` | — | `Yellow Alert` | — |
| 1045 | `Pict Object Info` | `O - Pict Object Info` | — | — |
| 1046 | **(unnamed)** | — | `No High Score` | — |

Confirmations worth noting:

* DITL 1010's `dctb` is named the generic `Object Info` while its DITL is `Furniture Info` —
  1010 is the read-only variant reused for furniture.
* DITL 1028 is named `Too Many Objects` but its ALRT is `No More Special`, matching
  `kNoMoreSpecialAlert` (`GliderPRO/Sources/ObjectAdd.c:16`).
* DITL 1032 is `High Scores` but the ALRT is `Clear Scores`, matching `kZeroScoresAlert`
  (`GliderPRO/Sources/HouseInfo.c:24`).
* DITL 1042 is `Monitors Switched` / ALRT 1042 `Color Switched` — a duplicate of ALRT 130
  `Color Depth`, and dead (§3.3.2).
* ALRT 1044 shares the name `Yellow Alert` with 1006 because it reuses the same icon
  (`cicn` 1006) and the same "A problem came up / Error #" shape.

---

## 3.4 Colour icons and icon families

### 3.4.1 `'cicn'` — 44 resources, 44,584 bytes, all 32x32

A `'cicn'` (`CIcon`) is the classic colour-icon-with-mask. Verified layout, exact for
**all 44** (`82 + maskBits + iconBits + colourTable + pixelData == len(resource)`):

| Offset | Size | Field | Observed in all 44 |
| ---: | ---: | --- | --- |
| 0 | 50 | `iconPMap` — a `PixMap` | see below |
| 50 | 14 | `iconMask` — a `BitMap` | `rowBytes = 4`, `bounds = 0,0,32,32` |
| 64 | 14 | `iconBMap` — a `BitMap` | `rowBytes = 4`, `bounds = 0,0,32,32` |
| 78 | 4 | `iconData` — a `Handle` | `0x00000000` |
| 82 | 128 | mask bits | `(32 rows) x (4 rowBytes)` = 128 bytes, 1 bit/pixel |
| 210 | 128 | 1-bit icon bits | 128 bytes |
| 338 | 8 + 8n | `ColorTable` | `ctSeed = 0`, `ctFlags = 0`, `ctSize = n-1` |
| … | `32 x rowBytes` | colour pixel data | at `iconPMap.pixelSize` bpp |

The 50-byte `PixMap`, big-endian, format string `'>IHhhhhhhIIIhhhhIII'`:

| Offset | Size | Field | Observed |
| ---: | ---: | --- | --- |
| 0 | 4 | `baseAddr` | 0 |
| 4 | 2 | `rowBytes` | high bit set on disk; masked with `& 0x3FFF` → 4, 16 or 32 |
| 6 | 8 | `bounds` | `0, 0, 32, 32` in all 44 |
| 14 | 2 | `pmVersion` | 0 |
| 16 | 2 | `packType` | 0 (unpacked) |
| 18 | 4 | `packSize` | 0 |
| 22 | 4 | `hRes` | `0x00480000` = 72.0 dpi |
| 26 | 4 | `vRes` | `0x00480000` = 72.0 dpi |
| 30 | 2 | `pixelType` | 0 (`chunky`, i.e. indexed) |
| 32 | 2 | `pixelSize` | **1, 4 or 8** |
| 34 | 2 | `cmpCount` | 1 |
| 36 | 2 | `cmpSize` | equals `pixelSize` |
| 38 | 4 | `planeBytes` | 0 |
| 42 | 4 | `pmTable` | 0 |
| 46 | 4 | `pmReserved` | 0 |

**`rowBytes` on disk has bit 15 set** (the "this is a PixMap not a BitMap" flag). Mask it.
`pixelSize` census with the implied `rowBytes`:

| `pixelSize` | `rowBytes` | Count | IDs |
| ---: | ---: | ---: | --- |
| 8 | 32 | 7 | 130, 1000, 1001, 1005, 1006, 1008, 1042 |
| 4 | 16 | 35 | all others |
| 1 | 4 | 2 | 1052, 1053 |

Colour-table entry counts range from **2** (the two 1-bit icons) to **44** (cicn 130, the
"requires 256 colors" icon). Distribution: 2 entries x2, 5 x1, 6 x6, 7 x6, 8 x4, 9 x5,
10 x2, 11 x4, 13 x4, 14 x2, 16 x1, 18 x1, 19 x2, 20 x1, 24 x1, 34 x1, 44 x1.

Each `ColorTable` entry is 8 bytes: `int16 value` + `RGBColor{uint16 red, green, blue}`.
For a `'cicn'` the `value` field is the pixel index; the table is *sparse* — a 4-bit icon may
carry only 6 entries, so indices with no entry are undefined. **A Go decoder must build a
256-entry palette initialised to something (black or transparent) and then fill in only the
indices present.** Do not assume `value == arrayIndex` for `'cicn'` (that assumption *is*
valid for `'clut'`, §3.10.3).

Byte-size arithmetic sanity check for a 4-bit cicn with 6 colour entries:
`50 + 14 + 14 + 4 + 128 + 128 + (8 + 8*6) + (32*16) = 906` — and cicn 1011, 1020, 1030, 1031,
1050, 1072 are all exactly 906 bytes.

### 3.4.2 `'ICON'` — 35 resources, 4,480 bytes

Every `'ICON'` is a flat **128 bytes**: a 32x32 1-bit bitmap, 4 bytes per row, no mask, no
header. `35 x 128 = 4480`. Verified.

The ICON ID set is a strict subset of the cicn ID set. On a colour Mac the Dialog Manager
prefers the `'cicn'`; the `'ICON'` is the black-and-white fallback. **A Go port only needs the
`'cicn'`s** — but see §3.4.3 for the nine cicn that have no ICON.

### 3.4.3 Which dialog uses which icon, and the nine code-drawn cicn

| cicn / ICON ID | Used by | Cite |
| ---: | --- | --- |
| 130 | DITL 130 item 4, DITL 1042 item 4 | — |
| 140 | DITL 140 item 4, DITL 1026 item 4 | — |
| 150 | DITL 150 item 3 (About) | — |
| 160 | DITL 160 item 3 | — |
| 170 | DITL 170 item 5 | — |
| 180 | DITL 180 item 3 | — |
| 900 | DITL 1031 item 3 (No Printing) | — |
| 910 | DITL 1032 item 5 (Clear Scores) | — |
| 1000 | DITL 1002 item 5, DITL 1009 item 4 | — |
| 1001 | DITL 1025 item 5 (Resume) | — |
| 1004 | DITL 1004 item 4 | — |
| 1005 | DITL 1005 item 4 | — |
| 1006 | DITL 1006 item 5, DITL 1044 item 3 | — |
| 1008 | DITL 1008 item 3, DITL 1028 item 3 | — |
| 1010 | DITL 1012 item 3; `kNormalSettingsIcon + 0` | `GliderPRO/Sources/Settings.c:1267`, `:1274` |
| 1011 | DITL 1012 item 4; `kNormalSettingsIcon + 1`; also DITL 1038 item 3 and DITL 1039 item 3 | `GliderPRO/Sources/Settings.c:1267` |
| 1012 | DITL 1012 item 5; `kNormalSettingsIcon + 2` | `GliderPRO/Sources/Settings.c:1267` |
| 1013 | DITL 1012 item 6; `kNormalSettingsIcon + 3` | `GliderPRO/Sources/Settings.c:1267` |
| **1014** | **cicn only** — `kInvertedSettingsIcon + 0` | `GliderPRO/Sources/Settings.c:1268`, `:1271` |
| **1015** | **cicn only** — `kInvertedSettingsIcon + 1` | `GliderPRO/Sources/Settings.c:1268` |
| **1016** | **cicn only** — `kInvertedSettingsIcon + 2` | `GliderPRO/Sources/Settings.c:1268` |
| **1017** | **cicn only** — `kInvertedSettingsIcon + 3` | `GliderPRO/Sources/Settings.c:1268` |
| 1020 | DITL 1017 item 3 | — |
| 1021 | DITL 1017 item 4 | — |
| 1022 | DITL 1017 item 5 | — |
| 1030 | DITL 1018 item 3 | — |
| 1031 | DITL 1018 item 5 | — |
| 1032 | DITL 1018 item 4 | — |
| **1033** | **cicn only** — volume-softer button, pressed state | `DrawCIcon(1033, ...)` `GliderPRO/Sources/Settings.c:836` |
| **1034** | **cicn only** — volume-louder button, pressed state | `DrawCIcon(1034, ...)` `GliderPRO/Sources/Settings.c:821` |
| 1040 | DITL 1023 item 5 | — |
| 1041 | DITL 1023 item 6 | — |
| 1042 | DITL 1023 item 7 | — |
| 1043 | DITL 1023 item 8 | — |
| 1050 | DITL 1000 item 29 (scroll-up arrow) | — |
| 1051 | DITL 1000 item 30 (scroll-down arrow) | — |
| **1052** | **cicn only, 1-bit** — `kGrayedOutUpArrow` | `GliderPRO/Sources/SelectHouse.c:156`, `:370`, `:382` |
| **1053** | **cicn only, 1-bit** — `kGrayedOutDownArrow` | `GliderPRO/Sources/SelectHouse.c:186`, `:374`, `:388` |
| 1060 | DITL 1029 item 4 (Lock House) | — |
| 1070 | DITL 1030 item 3 (Sound Manager) | — |
| 1071 | DITL 1036 item 3 (PICT Gone) | — |
| 1072 | DITL 1041 item 4 (Save Game?) | — |
| 1073 | DITL 1046 item 3 (No High Score) | — |
| **2000** | **cicn only** — the Tools palette Selection Tool | `DrawCIcon(2000, ...)` `GliderPRO/Sources/Tools.c:166` |

`DrawCIcon(short theID, short h, short v)` lives in `GliderPRO/Sources/Utilities.c` and is a
`GetCIcon` + `PlotCIcon(&bounds, theCIcon)` + `DisposeCIcon` at an offset position. In Go this
is a straightforward alpha blit at (h, v) using the mask as the alpha channel.

Note the ordering quirk in DITL 1018 (Sound Prefs): item 3 → cicn 1030, item **4 → 1032**,
item **5 → 1031**. Reproduce the item→resource mapping literally.

### 3.4.4 Finder icon families: `'ICN#'`, `'icl4'`, `'icl8'`, `'ics#'`, `'ics4'`, `'ics8'`

All flat, unheadered bitmaps. Verified sizes:

| Type | IDs | Each | Total | Layout |
| --- | --- | ---: | ---: | --- |
| `'ICN#'` | 128-133 | 256 | 1,536 | 32x32 1-bit **icon then mask**, 128 + 128 |
| `'icl4'` | 128-133 | 512 | 3,072 | 32x32 at 4 bpp (16 rowBytes x 32) |
| `'icl8'` | 128-133 | 1,024 | 6,144 | 32x32 at 8 bpp (32 rowBytes x 32) |
| `'ics#'` | 128-130, 133 | 64 | 256 | 16x16 1-bit icon then mask, 32 + 32 |
| `'ics4'` | 128-130, 133 | 128 | 512 | 16x16 at 4 bpp (8 rowBytes x 16) |
| `'ics8'` | 128-130, 133 | 256 | 1,024 | 16x16 at 8 bpp (16 rowBytes x 16) |

`'icl4'`/`'icl8'`/`'ics4'`/`'ics8'` carry **no mask and no palette** — the mask comes from the
matching `'ICN#'`/`'ics#'`, and the palette is the fixed Mac 4-bit or 8-bit system palette.
This is a hard dependency: to render `'icl8'` correctly a Go port needs the standard Macintosh
256-colour system palette. Conveniently, `'clut'` 128/129 in this same fork *are* that palette
(§3.10.3), so the fork is self-sufficient.

**IDs 131 and 132 have no small-icon family.** All six have the large family. The bundle
(§3.9) maps local IDs 0-5 to 128-133.

### 3.4.5 `LargeIconPlot` and the Icon Utilities path

`GliderPRO/Sources/Utilities.c:379-387` exposes `LargeIconPlot`, which is exactly two calls:

```
GliderPRO/Sources/Utilities.c:384   theErr = GetIconSuite(&theSuite, theID, svAllLargeData);
GliderPRO/Sources/Utilities.c:386   theErr = PlotIconSuite(theRect, atNone, ttNone, theSuite);
```

There is **no `DisposeIconSuite`** — the suite is leaked on every call.
`svAllLargeData` is an Icon Utilities selector mask; `Icons.h` is **not part of this source
drop**, so its numeric value cannot be verified here. What matters for a port is its
documented meaning: *load the large (32 × 32) members of the icon family* — `'ICN#'`, `'icl4'`
and `'icl8'` — for one ID, which is exactly the three types present for IDs 128-133 (§3.4.4).
`atNone` and `ttNone` are the "no alignment" and "no transform" selectors.

A *house file's* custom icon is fetched separately — see
`GliderPRO/Sources/SelectHouse.c:106`, which reads

```
if (Get1Resource('icl8', -16455) != nil)
```

from the **house file's** resource fork, not the app's — `Get1Resource` (not `GetResource`)
restricts the search to the just-opened house file. `-16455` appears as a bare literal in the
source; it is the well-known Finder "custom icon" family ID, but no `#define` for it exists
anywhere in `GliderPRO/Headers/` or `GliderPRO/Sources/`, so treat the *name* as inference and
the *number* as fact. The app fork has no such resource. A Go port that
wants per-house icons must parse the house file's own resource fork.

---

## 3.5 Cursors: `'CURS'` 16, `'crsr'` 12, `'acur'` 1

### 3.5.1 `'CURS'` — 68 bytes each, verified

| Offset | Size | Field |
| ---: | ---: | --- |
| 0 | 32 | `data` — 16x16 1-bit image, 2 bytes per row |
| 32 | 32 | `mask` — 16x16 1-bit mask, 2 bytes per row |
| 64 | 4 | `hotSpot` — `Point` = `int16 v`, `int16 h` (**vertical first**) |

`16 x 68 = 1088` bytes total. Every one is exactly 68 bytes. Hot spots:

| CURS IDs | `hotSpot` (v, h) | Constant | Cite |
| --- | --- | --- | --- |
| 128 | (8, 8) | `kHandCursorID` | `GliderPRO/Sources/InterfaceInit.c:16` |
| 129 | (7, 7) | `kVertCursorID` | `GliderPRO/Sources/InterfaceInit.c:17` |
| 130 | (7, 7) | `kHoriCursorID` | `GliderPRO/Sources/InterfaceInit.c:18` |
| 131 | (7, 7) | `kDiagCursorID` | `GliderPRO/Sources/InterfaceInit.c:19` |
| 149-160 (12) | (8, 7) | the 12-frame beach-ball spinner | `GliderPRO/Sources/AnimCursor.c` |

128-131 are the editor's resize cursors (hand, vertical, horizontal, diagonal), loaded with
`GetCursor` at `GliderPRO/Sources/InterfaceInit.c:82` (128), `:94` (129), `:100` (130) and
`:106` (131), and applied with `SetCursor`. (The intervening `GetCursor` at `:88` fetches the
system `iBeamCursor`, not a resource in this fork.)

`kArrowCursor` is **0** (`GliderPRO/Headers/GliderDefines.h:182`) and refers to QuickDraw's
built-in `arrow` global, not a resource.

### 3.5.2 `'crsr'` — colour cursors, 12 resources, IDs 149-160

A `'crsr'` (`CCrsr`) is the colour version of the same 12 spinner frames. Verified layout:

| Offset | Size | Field | Observed in all 12 |
| ---: | ---: | --- | --- |
| 0 | 2 | `crsrType` | `0x8001` = colour cursor (`0x8000` would be 1-bit) |
| 2 | 4 | `crsrMap` — offset to the `PixMap` | **96** |
| 6 | 4 | `crsrData` — offset to the pixel data | **146** |
| 10 | 4 | `crsrXData` | 0 |
| 14 | 2 | `crsrXValid` | **0** (expansion cache invalid — always, on disk) |
| 16 | 4 | `crsrXHandle` | 0 |
| 20 | 32 | `crsr1Data` — the 1-bit fallback image | 16x16, 2 bytes/row |
| 52 | 32 | `crsrMask` — the 1-bit mask | 16x16, 2 bytes/row |
| 84 | 4 | `crsrHotSpot` | **(v=8, h=7)** in all 12 |
| 88 | 4 | `crsrXTable` | 0 |
| 92 | 4 | `crsrID` | **0** |
| 96 | 50 | `PixMap` | see below |
| 146 | n | colour pixel data | |
| 146+n | 8+8m | `ColorTable` | |

The embedded `PixMap` in all 12: `bounds = 0,0,16,16`, `pmVersion = 0`, `packType = 0`,
`hRes = vRes = 0x00480000` (72 dpi), `pixelType = 0`, `cmpCount = 1`.

There are exactly **two size variants**:

| Variant | Bytes | `rowBytes` (raw) | `pixelSize` | `cmpSize` | `pmTable` | Pixel bytes | `ColorTable` entries | IDs |
| --- | ---: | --- | ---: | ---: | ---: | ---: | ---: | --- |
| A | 322 | `0x8008` → 8 | 4 | 4 | 274 | 128 | 5 | 149, 150, 156, 157, 158, 159, 160 |
| B | 250 | `0x8004` → 4 | 2 | 2 | 210 | 64 | 4 | 151, 152, 153, 154, 155 |

`7 x 322 + 5 x 250 = 2254 + 1250 = 3504` — matches the inventory total exactly.

**`pixelSize = 2` (variant B) is unusual** — a 2-bit-per-pixel cursor. QuickDraw supports it;
most modern image libraries do not. A Go decoder must handle 2 bpp as well as 4 bpp here.

`crsrID` is 0 on disk; the Cursor Manager fills it in at `GetCCursor` time. Do not rely on it.

### 3.5.3 `'acur'` 128 — the animated cursor, 52 bytes

| Offset | Size | Field | Observed |
| ---: | ---: | --- | --- |
| 0 | 2 | `n` — frame count | **12** |
| 2 | 2 | `index` — current frame | 0 |
| 4 | 4 x 12 | per-frame entries | see below |

`4 + 4*12 = 52` — exact. Each 4-byte entry is `int16 cursorID` followed by `int16` (a
`CursHandle` placeholder, 0 on disk). The observed CURS IDs, **in resource order**, are:

```
160, 159, 158, 157, 156, 155, 154, 153, 152, 151, 150, 149
```

i.e. **descending**. The spinner therefore rotates in the direction implied by 160→149, not
149→160. `GliderPRO/Sources/AnimCursor.c` walks this list; getting the direction wrong gives a
backwards-spinning beach ball.

`GliderPRO/Sources/AnimCursor.c` call sites: `:54`, `:63`, `:87`, `:116`, `:142`, `:201`,
`:229`. `SpinCursor(1)` is called after each GWorld init in
`GliderPRO/Sources/StructuresInit2.c:145-181`.

**`rHandCursorID 1000`** is declared at `GliderPRO/Sources/AnimCursor.c:15` and never used;
there is no `'CURS'` or `'acur'` 1000 in the fork. Dead.

---

## 3.6 `'STR#'` — 10 string lists, 8,229 bytes, 280 strings

### 3.6.1 Binary layout — verified

| Offset | Size | Field |
| ---: | ---: | --- |
| 0 | 2 | `numStrings` — `uint16` big-endian |
| 2 | … | `numStrings` Pascal strings, back to back |

Each Pascal string is one length byte followed by that many MacRoman bytes. **There is no
padding, no alignment, and no terminator.** Verified: for all 10 resources
`2 + Σ(1 + len) == len(resource)` exactly, and the parsed string count equals `numStrings`.

Access is 1-based via `GetIndString(Str255 theString, short strListID, short index)`.

| ID | Name | Bytes | Strings | Read at |
| ---: | --- | ---: | ---: | --- |
| 128 | `Jinjur` | 10 | 1 | **nowhere — dead** |
| 129 | `Prefmain` | 33 | 4 | `GliderPRO/Sources/Settings.c:1288`, `:1290`, `:1292`, `:1294` |
| 140 | `File Error` | 1,310 | 17 | `GliderPRO/Sources/FileError.c:92` |
| 150 | `Localized Strings` | 1,048 | 51 | `GliderPRO/Sources/StringUtils.c:325` (`kLocalizedStringsID`) |
| 160 | `Prefs` | 14 | 1 | `GliderPRO/Sources/Prefs.c:79` |
| 170 | `Errors` | 294 | 13 | `GliderPRO/Sources/Utilities.c:147`, `:152` (alert *titles*) |
| 171 | `Errors` | 1,336 | 13 | `GliderPRO/Sources/Utilities.c:148`, `:153` (alert *messages*) |
| 1005 | `Months` | 88 | 12 | `GliderPRO/Sources/ObjectDraw2.c:1161` |
| 1006 | `Yellow Alerts` | 2,527 | 24 | `GliderPRO/Sources/HouseIO.c:648` |
| 1007 | `Object Names` | 1,569 | 144 | `GliderPRO/Sources/ObjectInfo.c`, `GliderPRO/Sources/Tools.c:146` |

`10 + 33 + 1310 + 1048 + 14 + 294 + 1336 + 88 + 2527 + 1569 = 8229` — matches.

### 3.6.2 `'STR#'` 128 "Jinjur" — dead

One string: `Crystal`. Grep for `128` used as a `GetIndString` list ID: no hits. Grep for
`Jinjur` or `Crystal` anywhere in `Sources/` or `Headers/`: no hits. It is a leftover. (Jinjur
and the Glass Cat are *Wizard of Oz* characters, matching the `'ozm5'` creator code and the
`Ozma` object name — see §3.9.3.)

### 3.6.3 `'STR#'` 129 "Prefmain" — the four Preferences tabs

| # | String |
| ---: | --- |
| 1 | `Display` |
| 2 | `Sounds` |
| 3 | `Controls` |
| 4 | `Brains` |

Read one-per-`GetIndString` at `GliderPRO/Sources/Settings.c:1288`, `:1290`, `:1292`, `:1294`,
and drawn under the four tab icons (`cicn` 1010-1013, §3.4.3).

### 3.6.4 `'STR#'` 140 "File Error" — 17 messages

Indexed by a code derived from the OSErr at `GliderPRO/Sources/FileError.c:92`, then shown
through ALRT 140 as `^0` (DITL 140 item 2).

| # | String |
| ---: | --- |
| 1 | `A miscellaneous input/output error occurred (see error code below).` |
| 2 | `An error occurred because the directory is full.` |
| 3 | `An error occurred because the disk is full.  Try another disk.` |
| 4 | `An unspecified input output error occurred.` |
| 5 | `An error occurred because the name you chose is unacceptable.  Try a different name.` |
| 6 | `An error occurred because the file is not open.  Call tech support.` |
| 7 | `An error occurred because memory was too full.  Close windows, save, and bail out!` |
| 8 | `An error occurred because there are too many files open (no more than 12 at once allowed).` |
| 9 | `An error occurred because the disk is write protected.  Save onto an unlocked disk.` |
| 10 | `An error occurred because the file is locked.  Save under a different name.` |
| 11 | `An error occurred because the volume you selected is locked.  Save onto an unlocked disk.` |
| 12 | `An error occurred because the file is busy (perhaps another application has opened it).` |
| 13 | `An error occurred because the name you chose has already been used.  Save with another name.` |
| 14 | `An error occurred because the file is already open for writing.  Call tech support.` |
| 15 | `An error occurred because that volume is off-line.  Save onto another disk.` |
| 16 | `An error occurred because of a permission violation.  Save onto an unlocked disk.` |
| 17 | `An error occurred because write permission was denied.  Save onto an unlocked disk.` |

Note the doubled spaces after sentences — they are in the data and a byte-exact port must
preserve them.

### 3.6.5 `'STR#'` 150 "Localized Strings" — 51 strings

`kLocalizedStringsID` = 150; every one is read through
`GetIndString(theString, kLocalizedStringsID, index)` at
`GliderPRO/Sources/StringUtils.c:325`.

| # | String |
| ---: | --- |
| 1 | `There are ` |
| 2 | `There is ` |
| 3 | ` stars in the house.` |
| 4 | ` star in the house.` |
| 5 | `Get every star to win.` |
| 6 | `room` |
| 7 | `rooms` |
| 8 | `Click Mouse or Hit a Key to Exit` |
| 9 | `Name for New House:` |
| 10 | `Untitled House` |
| 11 | `Enter New-Game Message here (max. 255 characters)` |
| 12 | `Enter Finished-House Message here (max. 255 characters)` |
| 13 | `Converting 1.0 House to 2.0` |
| 14 | `Converting Room ` |
| 15 | `Save copy of house as:` |
| 16 | `Failed House Shrinkage` |
| 17 | ` Floor Number Bad` |
| 18 | ` Suite Number Bad` |
| 19 | `Object Bad` |
| 20 | `No room upstairs!` |
| 21 | `No downstairs to match!` |
| 22 | `No room downstairs!` |
| 23 | `No upstairs to match!` |
| 24 | `Checking House File` |
| 25 | `Checking House…` |
| 26 | `Checking Rooms…` |
| 27 | `Room Number Errors` |
| 28 | ` Duplicate Floor/Suites` |
| 29 | ` Room Errors` |
| 30 | ` 'Untitled' Rooms` |
| 31 | ` Room Names Too Long` |
| 32 | ` Room's # of Objects Wrong` |
| 33 | `Checking Objects…` |
| 34 | ` Object Errors` |
| 35 | `You have no stars in the house!` |
| 36 | `Cut Object` |
| 37 | `Copy Object` |
| 38 | `Clear Object` |
| 39 | `Cut Room` |
| 40 | `Copy Room` |
| 41 | `Clear Room` |
| 42 | `Paste Room` |
| 43 | `Paste Object` |
| 44 | `Nothing To Paste` |
| 45 | `Object Pair Added` |
| 46 | `Exterior Door Added in Next Room` |
| 47 | `Interior Door Added in Next Room` |
| 48 | `Ext. Window Added in Next Room` |
| 49 | `Int. Window Added in Next Room` |
| 50 | `Down Stairs Added in Room Above` |
| 51 | `Up Stairs Added in Room Below` |

Strings 1-5 are assembled into the house banner: `"There are " + N + " stars in the house."`
versus `"There is " + N + " star in the house."` — a plural rule expressed as two index pairs.
Strings 6/7 are the `room`/`rooms` plural. Strings 13/14 are the 1.0→2.0 house upgrade
progress text. Note the leading spaces in 3, 4, 14, 17, 18, 28-32, 34 — those strings are
suffixes/prefixes concatenated with a number.

The `…` characters are MacRoman `0xC9` (single-byte ellipsis), not three periods.

### 3.6.6 `'STR#'` 160 "Prefs" — the preferences file name

One string: `Preferences`. Read at `GliderPRO/Sources/Prefs.c:79` to build the prefs filename
in the System Folder's Preferences directory.

### 3.6.7 `'STR#'` 170 / 171 "Errors" — `RedAlert` titles and messages

The two lists are parallel: `GetIndString(title, 170, whichErr)` and
`GetIndString(message, 171, whichErr)`, then `ParamText(title, message, errNumStr, nil)` and
`Alert(rDeathAlertID /* 170 */, nil)` (`GliderPRO/Sources/Utilities.c:147-159`). The alert's
DITL 170 has `^0` = title, `^1` = message, `^2` = program error number.

| # | `'STR#'` 170 (title) | `'STR#'` 171 (message) |
| ---: | --- | --- |
| 1 | `Unaccounted for Error` | `An error of unknown origin has occurred.  Call Casady & Greene's technical support.` |
| 2 | `Out of Memory Error!` | `We ran out of memory.  You can give Glider PRO™ more memory using Get Info from the Finder.` |
| 3 | `A Dialog Couldn't be Loaded` | `A dialog was unable to load.  Most likely, you need to increase the memory allocated for Glider PRO™.` |
| 4 | `Couldn't Load a Resource` | `A required resource was unable to load.  Most likely, you need to increase the memory allocated for Glider PRO™.` |
| 5 | `Couldn't Load a Graphic` | `A graphic was unable to load.  Most likely, you need to increase the memory allocated for Glider PRO™.` |
| 6 | `Directory Look-up Failed` | `Failed attempting to determine our directory.  Call Casady & Greene's technical support.` |
| 7 | `Couldn't Validate` | `Without your original disk, you cannot complete the installation.` |
| 8 | `Need System 7` | `Glider PRO™ requires System 7.0 (or more recent) to run.  You'll need a more current System version from Apple.` |
| 9 | `Graphics Device Look-up Failed` | `Who knows how this could have possibly happened.  Call Casady & Greene's technical support.` |
| 10 | `Memory Operation Failed` | `Likely memory is running low or is fragmented.  You can give Glider PRO™ more memory using Get Info from the Finder.` |
| 11 | `Failed House Search` | `The act of looking for house files failed.  Call Casady & Greene's technical support and report this error.` |
| 12 | `Need Color Quickdraw` | `This Macintosh is incapable of color (or even grayscale).  Glider PRO™ requires a Mac with Color Quickdraw built in.` |
| 13 | `Need Color Monitor` | `Glider PRO runs only in 16 shades of gray or 256 colors.  None of the monitors hooked up to this Mac are capable of either of these modes.` |

Index 3 is `kErrDialogDidntLoad` (raised by `BringUpDialog`,
`GliderPRO/Sources/DialogUtils.c:28`); index 4 is `kErrFailedResourceLoad`; index 5 is
`kErrFailedGraphicLoad` (raised by `LoadGraphic`). In a Go port, indices 2, 3, 4, 5, 10 are all
memory-exhaustion paths that simply cannot happen and can be collapsed into a generic
"asset missing" failure.

### 3.6.8 `'STR#'` 1005 "Months" — the calendar object

Twelve upper-case month names, drawn onto the Calendar object at
`GliderPRO/Sources/ObjectDraw2.c:1161`:

`JANUARY`, `FEBRUARY`, `MARCH`, `APRIL`, `MAY`, `JUNE`, `JULY`, `AUGUST`, `SEPTEMBER`,
`OCTOBER`, `NOVEMBER`, `DECEMBER`.

Index is 1-based, so `month` from the system clock maps directly.

### 3.6.9 `'STR#'` 1006 "Yellow Alerts" — 24 non-fatal warnings

Shown through ALRT 1006 (`kYellowAlert`) as `^0`, with the numeric code as `^1`:
`GliderPRO/Sources/HouseIO.c:642-654`. The 24 indices are declared as constants at
`GliderPRO/Headers/GliderDefines.h:18-41`.

| # | String |
| ---: | --- |
| 1 | `A never-before-seen error has arisen.  Proceed with caution!  (Save and Quit immediately.)` |
| 2 | `I failed to open the house's resource fork.  Any unique room backgrounds are not accessible.` |
| 3 | `I failed to add a resource to the house's resource fork.  See error number.` |
| 4 | `I failed to create a new resource fork for the house.  See error number for problem.` |
| 5 | `There are no houses on this drive!  About your only option is to create your own new house with the Editor.` |
| 6 | `This house is incompatible with us!  You'll need to upgrade Glider PRO to use this house.  Do not attempt to play/edit this house!` |
| 7 | `The background specified by this room was not found!  Try re-selecting a new background (the Room Info menu).` |
| 8 | `The room number is out of bounds.  I suspect the house file is corrupt.  Try deleting this "illegal" room though.` |
| 9 | `The data is missing that specifies where the openings in this room are.  The house may be damaged.  Try selecting a new background though.` |
| 10 | `There was a problem with the clipboard (Cut, Copy and Paste commands).  I couldn't guess why.` |
| 11 | `I think we just ran out of memory.  Quit now and give Glider PRO™ more memory.` |
| 12 | `We failed to write the house to disk.  (That shouldn't have happened.)` |
| 13 | `Well, the music didn't load.  Glider PRO™ will still run, you'll just be musically challenged.` |
| 14 | `Wow, there was a problem bringing sounds up.  You might try giving Glider PRO™ more memory - otherwise ... silence.` |
| 15 | `Some kind of strange Apple Event error.  I think I would just ignore it.  Or call Casady & Greene with the error number.` |
| 16 | `Did you save the house on the same volume Glider PRO is on?  I saved the house but had to re-open the old house because I couldn't find the new one.` |
| 17 | `Wow, I couldn't find the old or new house.  Go to the Select House menu item and see if it's there.  If not, make sure they're on the same volume as Glider PRO.` |
| 18 | `Couldn't create a saved game structure.  Memory is probably too low.` |
| 19 | `The saved game doesn't match the house.  Either this game was saved for a different house or the house was modified recently.` |
| 20 | `This saved game is an old version.  We cannot use this game with this house.` |
| 21 | `The number of rooms saved doesn't match the number of house rooms.  We cannot use this game with this house.` |
| 22 | `The QuickTime™ movie that goes with this house will not be used.  Glider PRO™ must have enough memory to easily load the entire movie into RAM.` |
| 23 | `This house has no rooms!  Do not attempt to play this house!  Select a new house to play.` |
| 24 | `There was an error generating or parsing a links list.  Memory may be tight.` |

Index 2 is the one a Go port will hit most often: it is raised when a house's resource fork
cannot be opened, which is exactly the case for a house with no custom backgrounds.

### 3.6.10 `'STR#'` 1007 "Object Names" — 144 entries, indexed by object type code

`kObjectNameStrings` = 1007 (`GliderPRO/Headers/GliderDefines.h:461`) and
`kNumSrcRects` = `0x90` = **144** (`GliderPRO/Headers/GliderDefines.h:437`). The list is indexed
directly by the object's `what` field, which runs 0x01..0x8F
(`GliderPRO/Headers/GliderDefines.h:311-435`) — so `GetIndString(name, 1007, what)` and the
string list is exactly 144 long to cover `what` up to 0x8F = 143, plus one extra.

**Fifteen entries are single-hex-character filler** for unallocated type codes: indices 32,
48, 74, 75, 76, 77, 78, 79, 80, 89, 90, 91, 92, 93, 94, 95, 96, 111, 112, 122, 123, 124, 125,
126, 127, 128. Their contents are literally `'f'`, `'9'`, `'a'`, `'b'`, `'c'`, `'d'`, `'e'`,
`'8'` — the low nibble of the unused type code, typed as a placeholder. Do not display them.

**Index 144 is `Mermaid`, one past `kChimes` (0x8F = 143).** There is no object type 144 and
no `srcRects[144]`, so this entry is unreachable — an unimplemented object.

| # | Name | | # | Name | | # | Name |
| ---: | --- | --- | ---: | --- | --- | ---: | --- |
| 1 | `Floor Vent` | | 49 | `Up Stairs` | | 97 | `Paper Shredder` |
| 2 | `Ceiling Vent` | | 50 | `Down Stairs` | | 98 | `Toaster` |
| 3 | `Floor Duct` | | 51 | `Mailbox (faces lf.)` | | 99 | `Mac Plus` |
| 4 | `Ceiling Duct` | | 52 | `Mailbox (faces rt.)` | | 100 | `Guitar` |
| 5 | `Sewer Grate` | | 53 | `Floor Trans. Duct` | | 101 | `T.V.` |
| 6 | `Table Fan` | | 54 | `Ceiling Trans. Duct` | | 102 | `Coffee Machine` |
| 7 | `Table Fan` | | 55 | `Door (interior)` | | 103 | `Electrical Outlet` |
| 8 | `Taper` | | 56 | `Door (interior)` | | 104 | `VCR` |
| 9 | `Simple Candle` | | 57 | `Door (exterior)` | | 105 | `Stereo System` |
| 10 | `Stubby Candle` | | 58 | `Door (exterior)` | | 106 | `Microwave Oven` |
| 11 | `Tiki Torch` | | 59 | `Window (interior)` | | 107 | `Cinder Block` |
| 12 | `Barbecue Grill` | | 60 | `Window (interior)` | | 108 | `Flower Box` |
| 13 | `Invisible Blower` | | 61 | `Window (exterior)` | | 109 | `Compact Discs` |
| 14 | `Greco-Roman Vent` | | 62 | `Window (exterior)` | | 110 | `Custom Picture` |
| 15 | `Sewer Blower` | | 63 | `Invisible Transport` | | 111 | *filler* `e` |
| 16 | `Lift Area` | | 64 | `Deluxe Transport` | | 112 | *filler* `f` |
| 17 | `Table` | | 65 | `Light Switch` | | 113 | `Balloon` |
| 18 | `Shelf` | | 66 | `Machine Switch` | | 114 | `'Copter (lf. drift)` |
| 19 | `Cabinet` | | 67 | `Thermostat` | | 115 | `'Copter (rt. drift)` |
| 20 | `Filing Cabinet` | | 68 | `Digital Switch` | | 116 | `Dart (lf. moving)` |
| 21 | `Wastebasket` | | 69 | `Knife Switch` | | 117 | `Dart (rt. moving)` |
| 22 | `Milk Crate` | | 70 | `Invisible Switch` | | 118 | `Bouncing Ball` |
| 23 | `Counter` | | 71 | `Trigger` | | 119 | `Water Drip` |
| 24 | `Dresser` | | 72 | `Large Trigger` | | 120 | `Fish Bowl & Fish` |
| 25 | `Deck Table` | | 73 | `Sound Trigger` | | 121 | `Cobweb` |
| 26 | `Bar Stool` | | 74 | *filler* `9` | | 122 | *filler* `9` |
| 27 | `Steamer Trunk` | | 75 | *filler* `a` | | 123 | *filler* `a` |
| 28 | `Invisible Obstacle` | | 76 | *filler* `b` | | 124 | *filler* `b` |
| 29 | `Manhole` | | 77 | *filler* `c` | | 125 | *filler* `c` |
| 30 | `Books` | | 78 | *filler* `d` | | 126 | *filler* `d` |
| 31 | `Invisible Rebounder` | | 79 | *filler* `e` | | 127 | *filler* `e` |
| 32 | *filler* `f` | | 80 | *filler* `f` | | 128 | *filler* `f` |
| 33 | `Digital Clock` | | 81 | `Ceiling Light` | | 129 | `Ozma` |
| 34 | `Wall Clock` | | 82 | `Simple Bulb` | | 130 | `Mirror` |
| 35 | `Alarm Clock` | | 83 | `Table Lamp` | | 131 | `Mouse Hole` |
| 36 | `Cuckoo Clock` | | 84 | `Hip Pole Lamp` | | 132 | `Fireplace` |
| 37 | `Extra Glider` | | 85 | `Deco Lamp` | | 133 | `Flower` |
| 38 | `Battery` | | 86 | `Flourescent Light` | | 134 | `Window (closed)` |
| 39 | `Rubber Bands (8)` | | 87 | `Track Lighting` | | 135 | `Teddy Bear` |
| 40 | `Grease (spills rt.)` | | 88 | `Invisible Light` | | 136 | `Calendar` |
| 41 | `Grease (spills lf.)` | | 89 | *filler* `8` | | 137 | `Broad Vase` |
| 42 | `Aluminum Foil` | | 90 | *filler* `9` | | 138 | `Narrow Vase` |
| 43 | `Invisible Bonus` | | 91 | *filler* `a` | | 139 | `Bulletin Board` |
| 44 | `Magic Star` | | 92 | *filler* `b` | | 140 | `Cloud` |
| 45 | `Sparkle` | | 93 | *filler* `c` | | 141 | `Faucet` |
| 46 | `Helium (He)` | | 94 | *filler* `d` | | 142 | `Throw Rug` |
| 47 | `Slide Rect` | | 95 | *filler* `e` | | 143 | `Wind Chimes` |
| 48 | *filler* `f` | | 96 | *filler* `f` | | 144 | `Mermaid` (unreachable) |

The type-code blocks are visible in the filler pattern: blowers 1-16, furniture 17-32,
bonus/prizes 33-48, transport 49-64, switches 65-80, lights 81-96, appliances 97-112,
enemies 113-128, clutter 129-144. **Each block is 16 codes wide** and the unused tail of each
block is filled. This matches the nine Tools-palette groups in `'MENU'` 141 (§3.7.5) — except
that the palette has 9 groups plus a Selection Tool, and "Prizes"/"Bonus" is one group.

### 3.6.11 There is no `'STR '` in this fork

`GetString` is called exactly once:

```
GliderPRO/Sources/StringUtils.c:304   #define kChooserStringID  -16096
GliderPRO/Sources/StringUtils.c:308   theNameHandle = (Handle)GetString(kChooserStringID);
```

`-16096` is the **System file's** "Chooser name" string (the owner name from the Sharing Setup
control panel). It is used to pre-fill the high-score name field. A Go port should substitute
the host user name (`os/user.Current().Name`) and fall back to an empty string.

---

## 3.7 Menus: `'MENU'` 6, `'mctb'` 5

### 3.7.1 `'MENU'` binary layout — verified

| Offset | Size | Field | Observed in all 6 |
| ---: | ---: | --- | --- |
| 0 | 2 | `menuID` | see §3.7.5 for the 140/141 anomaly |
| 2 | 2 | `menuWidth` | **0** (computed at run time) |
| 4 | 2 | `menuHeight` | **0** |
| 6 | 2 | `menuProc` (defProcID) | **0** (standard MDEF) |
| 8 | 2 | filler | **0** |
| 10 | 4 | `enableFlags` | bit 0 = whole menu, bits 1-31 = items 1-31 |
| 14 | … | `menuTitle` | Pascal string |
| … | … | items | repeated, see below |
| … | 1 | terminator | `0x00` |

Each item is:

| Size | Field |
| ---: | --- |
| 1 | text length |
| n | text (MacRoman) |
| 1 | `iconIndex` (0 = none; otherwise ICON/cicn `256 + index`) |
| 1 | `keyEquivalent` (0 = none; else the command-key character) |
| 1 | `markCharacter` |
| 1 | `style` (QuickDraw text-face bits) |

A `-` as the item text means a separator line. Verified: every resource's bytes are consumed
exactly, terminator included.

`enableFlags` observed values decode as follows (`0` bit = disabled):

| MENU | `enableFlags` | Disabled items |
| ---: | --- | --- |
| 128 | `0xFFFFFFFB` | item 2 (the `-`) |
| 129 | `0xFFFFFFAF` | items 4, 6 (the two `-`) |
| 130 | `0xFFFFFFFB` | item 2 (the `-`) |
| 131 | `0xFFFADF77` | items 3, 7, 13, 16, 18 (all five `-`) |
| 140 | `0xFFF7FFFF` | item 19 (`Original Artwork`) |
| 141 | `0xFFFFFFFF` | none |

So in every menu the disabled bits are exactly the separators, plus `Original Artwork` in
MENU 140 which is enabled at run time only when the house has custom background PICTs.

### 3.7.2 `'MENU'` 128 — the Apple menu, 45 bytes

`menuID` **128** (`kAppleMenuID`, `GliderPRO/Headers/GliderDefines.h:186`).
`enableFlags = 0xFFFFFFFB`. **Title is the single byte `0x14`** — MacRoman for the Apple
logo glyph. In Unicode there is no Apple logo; a Go port must draw a bitmap or use a
platform-specific glyph.

| # | Text | Key |
| ---: | --- | --- |
| 1 | `About Glider PRO…` | — |
| 2 | `-` | — |

`AppendResMenu(appleMenu, 'DRVR')` is called at `GliderPRO/Sources/InterfaceInit.c:52` to append
the desk accessories after the separator. In Go there is nothing to append; drop it.

### 3.7.3 `'MENU'` 129 "Game" — 112 bytes

`menuID` **129** (`kGameMenuID`, `GliderPRO/Headers/GliderDefines.h:187`).

| # | Text | Key |
| ---: | --- | --- |
| 1 | `New Game` **+ a trailing NUL** | `N` |
| 2 | `Two Player Game` | `2` |
| 3 | `Open Saved Game…` | `O` |
| 4 | `-` | — |
| 5 | `Load House…` | `L` |
| 6 | `-` | — |
| 7 | `Quit` | `Q` |

> **Data defect:** item 1's length byte is **9** for the 8-character string `New Game`; the
> ninth byte is `0x00`. The Menu Manager would draw a trailing null glyph. A Go port should
> trim it. This is the only such case in the six menus.

### 3.7.4 `'MENU'` 130 "Options" — 89 bytes, and `'MENU'` 131 "House" — 314 bytes

MENU 130, `menuID` **130** (`kOptionsMenuID`, `GliderPRO/Headers/GliderDefines.h:188`):

| # | Text | Key |
| ---: | --- | --- |
| 1 | `Room Editor` | `E` |
| 2 | `-` | — |
| 3 | `High Scores…` | `H` |
| 4 | `Preferences…` | `P` |
| 5 | `Demo…` | `D` |

MENU 131, `menuID` **131** (`kHouseMenuID`, `GliderPRO/Headers/GliderDefines.h:189`) — the
Room Editor menu, inserted only in editor mode:

| # | Text | Key |
| ---: | --- | --- |
| 1 | `New House…` | `N` |
| 2 | `Save House` | `S` |
| 3 | `-` | — |
| 4 | `House Info…` | — |
| 5 | `Room Info…` | `R` |
| 6 | `Object Info…` | `I` |
| 7 | `-` | — |
| 8 | `Cut Room` | `X` |
| 9 | `Copy Room` | `C` |
| 10 | `Paste Room` | `V` |
| 11 | `Delete Room` | — |
| 12 | `Duplicate Object` | `D` |
| 13 | `-` | — |
| 14 | `Bring To Front` | `=` |
| 15 | `Send To Back` | `-` |
| 16 | `-` | — |
| 17 | `Go To Room…` | `G` |
| 18 | `-` | — |
| 19 | `Map Window` | `M` |
| 20 | `Tools Window` | `T` |
| 21 | `Coordinate Window` | `K` |

Note the command-key collisions across menus: `N` is New Game (129/1) *and* New House
(131/1); `D` is Demo (130/5) *and* Duplicate Object (131/12). The Menu Manager resolves them
by scanning menus in insertion order, so **MENU 129/130 win when the House menu is also
inserted**. Item 15's key equivalent is the literal `-` character, which is also its own
menu-separator marker in other items — a port must not confuse the two (separator-ness is
decided by the *text*, not the key).

### 3.7.5 `'MENU'` 140 "Rooms" and 141 "Tools" — both carry in-resource `menuID` 133

These two are *popup* menus attached to controls, not menu-bar menus.

> **Both resources have `menuID == 133` in their header bytes**, despite being resources 140
> and 141. That is legal — the Menu Manager renumbers a popup's menu when the CDEF loads it —
> but a naive port that keys menus by the in-resource `menuID` would collapse them into one.
> **Key by resource ID.**

MENU 140 "Rooms", 281 bytes, 19 items, `enableFlags = 0xFFF7FFFF` (item 19 disabled). Items
1-18 are the room-background names in `'PICT'` ID order 2000..2017, and item 19 selects a
house-supplied PICT:

| # | Text | Background PICT |
| ---: | --- | ---: |
| 1 | `Simple Room` | 2000 |
| 2 | `Paneled Room` | 2001 |
| 3 | `Basement` | 2002 |
| 4 | `Child's Room` | 2003 |
| 5 | `Asian Room` | 2004 |
| 6 | `Unfinished Room` | 2005 |
| 7 | `Swinger's Room` | 2006 |
| 8 | `Bathroom` | 2007 |
| 9 | `Library` | 2008 |
| 10 | `Garden` | 2009 |
| 11 | `Skywalk` | 2010 |
| 12 | `Dirt` | 2011 |
| 13 | `Meadow` | 2012 |
| 14 | `Field` | 2013 |
| 15 | `Roof` | 2014 |
| 16 | `Sky` | 2015 |
| 17 | `Stratosphere` | 2016 |
| 18 | `Stars` | 2017 |
| 19 | `Original Artwork` (disabled) | → DLOG 1016 `kOriginalArtDialogID` |

So `backgroundPICT = 2000 + (menuItem - 1)` for items 1-18, i.e.
`kBaseBackgroundID + index`. Item 19 opens DLOG 1016 to type a PICT ID; the code accepts
`3000 <= id < 3800` (`GliderPRO/Sources/RoomInfo.c:762`) even though the dialog's own label
says `(3000 - 3499)` and `kUserStructureRange` is 3300 — see §3.1.7 and "Open questions" #4.

This menu is driven by `'CNTL'` 128 (`max = 18`, `min = 1`) — a popup control whose 18 values
map to the 18 items. Note that **CNTL 128's `max` is 18, so the popup does not include item
19**; `Original Artwork` is reachable only through the separate code path. Verified: DITL 1003
item 11 is the sole `resCtrl` and points at CNTL 128.

MENU 141 "Tools", 142 bytes, 9 items, `enableFlags = 0xFFFFFFFF` (all enabled):

| # | Text | Object type-code block |
| ---: | --- | --- |
| 1 | `Blowers` | 0x01-0x10 |
| 2 | `Furniture` | 0x11-0x20 |
| 3 | `Prizes` | 0x21-0x30 |
| 4 | `Transport` | 0x31-0x40 |
| 5 | `Switches` | 0x41-0x50 |
| 6 | `Lighting` | 0x51-0x60 |
| 7 | `Appliances Etc.` | 0x61-0x70 |
| 8 | `Enemies` | 0x71-0x80 |
| 9 | `Clutter` | 0x81-0x8F |

Referenced by `'CNTL'` 129, whose `refCon` is **141** (§3.8.1) — the popup CDEF reads the
menu resource ID out of the control's `refCon`.

### 3.7.6 `'mctb'` — 5 menu colour tables, 910 bytes

Layout: `int16 count`, then `count` entries of **30 bytes**:

| Offset | Size | Field |
| ---: | ---: | --- |
| 0 | 2 | `mctID` — the menu ID this entry applies to |
| 2 | 2 | `mctItem` — the item number (0 = the menu as a whole) |
| 4 | 6 | `mctRGB1` — title / item text colour |
| 10 | 6 | `mctRGB2` |
| 16 | 6 | `mctRGB3` |
| 22 | 6 | `mctRGB4` |
| 28 | 2 | `mctReserved` |

Verified: `2 + 30*count == len(resource)` for all 5 (32, 152, 122, 452, 152).

| `'mctb'` | Name | Entries | `mctItem` values |
| ---: | --- | ---: | --- |
| 128 | `\0x14 menu` | 1 | 1 |
| 129 | `Game menu` | 5 | 3, 1, 2, 7, 5 |
| 130 | `Options menu` | 4 | 4, 5, 1, 3 |
| 131 | `House menu` | 15 | 10, 9, 8, 6, 5, 4, 2, 1, 21, 20, 19, 17, 15, 14, 12 |
| 132 | **`Edit menu`** | 5 | 1, 3, 4, 5, 8 |

Every `mctID` equals the resource ID, and every entry's colour words are zero (black) except:

* `mctRGB4` is always `FFFF FFFF FFFF` (white) — the background.
* One component is `0x9999` (39321 decimal) in a mid slot — a mid-grey, presumably an
  intended "dimmed" colour.
* A handful of entries in `'mctb'` 130 and 131 additionally carry `0xFFFF` in the first
  component of `mctRGB1`.

Practically these tables are near-no-ops (black text on white). **A Go port can ignore
`'mctb'` entirely.**

> **`'mctb'` 132 is named `Edit menu` but there is no `'MENU'` 132 and no
> `kEditMenuID` constant.** The Edit menu was designed and then cut; only its colour table and
> its `'cctb'`/`'CNTL'` 132 leftovers survive. Confirms `'CNTL'` 132 (§3.8.1) as another piece
> of the same removed feature.

---

## 3.8 Controls and windows: `'CNTL'` 5, `'WIND'` 3

### 3.8.1 `'CNTL'` binary layout — verified

| Offset | Size | Field |
| ---: | ---: | --- |
| 0 | 8 | `boundsRect` (top, left, bottom, right) — `int16` each |
| 8 | 2 | `value` |
| 10 | 1 | `visible` (Boolean) |
| 11 | 1 | filler — 0 in all 5 |
| 12 | 2 | `max` |
| 14 | 2 | `min` |
| 16 | 2 | `procID` (CDEF resource ID × 16 + variation code) |
| 18 | 4 | `refCon` |
| 22 | … | `title` Pascal string |

Verified: `22 + 1 + len(title) == len(resource)` for all 5.

| ID | Rect (t,l,b,r) | value | max | min | procID | refCon | Title | Bytes | Attrs |
| ---: | --- | ---: | ---: | ---: | ---: | ---: | --- | ---: | --- |
| 128 | 0, 0, 20, 190 | 1 | 18 | 1 | 2050 | 0 | *(empty)* | 23 | — |
| 129 | 2, 4, 22, 112 | 1 | 3 | 1 | 2050 | **141** | *(empty)* | 23 | — |
| 130 | 5, 70, 25, 124 | 0 | 0 | 0 | 0 | 0 | `Link` | 27 | — |
| 131 | 5, 5, 25, 59 | 0 | 0 | 0 | 0 | 0 | `Unlink` | 29 | — |
| 132 | 132, 184, 195, 247 | 150 | 151 | 3 | **176** | 0 | *(empty)* | 23 | `purgeable` |

`procID` decodes as `CDEF_resource_id * 16 + variant`:

| `procID` | Hex | CDEF ID | Variant | Meaning |
| ---: | --- | ---: | ---: | --- |
| 0 | `0x0000` | 0 | 0 | `pushButProc` — the standard system button CDEF |
| 2050 | `0x0802` | **128** | 2 | `'CDEF'` 128 in this fork (§3.11.2), variant 2 = pop-up menu |
| 176 | `0x00B0` | **11** | 0 | **`'CDEF'` 11 — does not exist in this fork or in the System** |

* **`'CNTL'` 128** — the Room Info background pop-up. Not created with `GetNewControl`; it is
  embedded as the single `resCtrl` item in DITL 1003 (item 11), so the Dialog Manager
  instantiates it. `min = 1`, `max = 18` matches `'MENU'` 140's 18 background items and
  `kNumBackgrounds` = 18 (`GliderPRO/Headers/GliderDefines.h:521`). The menu itself is fetched
  separately by ID with `GetMenu(kBackgroundsMenuID /* 140 */)`
  (`GliderPRO/Sources/RoomInfo.c:379`, `:397`).

  > **Quirk:** `SetPopUpMenuValue(roomInfoDialog, kRoomPopupItem, kOriginalArtworkItem /* 19 */)`
  > at `GliderPRO/Sources/RoomInfo.c:436` sets the control's value to **19, which is greater
  > than `max` = 18**. The Control Manager does not clamp values set through `SetControlValue`
  > on a custom CDEF, so this works by accident. A Go port must allow value 19 (= "Original
  > Artwork") even though the popup range nominally stops at 18. See also
  > `GliderPRO/Sources/RoomInfo.c:726`, `if (was >= kOriginalArtworkItem)`.

* **`'CNTL'` 129** — the Tools palette class pop-up. `kPopUpControl` = 129
  (`GliderPRO/Sources/Tools.c:18`), created with
  `classPopUp = GetNewControl(kPopUpControl, toolsWindow)` at
  `GliderPRO/Sources/Tools.c:315`. Its `refCon` is **141** — the popup CDEF reads the
  `'MENU'` resource ID out of `refCon`. `max = 3` is wrong for a 9-item menu (`'MENU'` 141 has
  9 items); the CDEF overwrites `max` with the real item count when it first attaches the menu.

* **`'CNTL'` 130 / 131** — the two push buttons in the floating Link windoid.
  `kLinkControlID` = 130, `kUnlinkControlID` = 131
  (`GliderPRO/Sources/Link.c:15-16`), created at `GliderPRO/Sources/Link.c:238` and `:242`.
  Their rects are relative to the 129 × 30 windoid content region created at
  `GliderPRO/Sources/Link.c:220` (`QSetRect(&linkWindowRect, 0, 0, 129, 30)`).

* **`'CNTL'` 132** — **an orphan.** Nothing in `Sources/` mentions control 132; its `procID`
  176 points at a nonexistent CDEF 11; it is the only `purgeable` `'CNTL'`; and its
  `min`/`max`/`value` (3 / 151 / 150) look like a scrolling list. Together with `'mctb'` 132
  `Edit menu` and `'cctb'` 132 (§3.3.6) it is debris from a removed Edit-menu / list feature.
  **Skip it.**

### 3.8.2 `'WIND'` binary layout — verified

| Offset | Size | Field |
| ---: | ---: | --- |
| 0 | 8 | `boundsRect` (top, left, bottom, right) |
| 8 | 2 | `procID` (WDEF ID × 16 + variant) |
| 10 | 1 | `visible` |
| 11 | 1 | filler |
| 12 | 1 | `goAwayFlag` |
| 13 | 1 | filler |
| 14 | 4 | `refCon` |
| 18 | … | `title` Pascal string |

Verified: `18 + 1 + len(title) == len(resource)` for all 3, no trailing `'WCTB'` index.

| ID | Constant | Rect (t,l,b,r) | Size | `procID` | visible | goAway | `refCon` | Title | Bytes |
| ---: | --- | --- | --- | ---: | ---: | ---: | ---: | --- | ---: |
| 128 | `kMainWindowID` | 0, 0, 384, 512 | 512 × 384 | 2 | 0 | 0 | 0 | `Main Window` | 30 |
| 129 | `kEditWindowID` | 0, 0, 322, 512 | 512 × 322 | 4 | 0 | 0 | 0 | `Main Window` | 30 |
| 130 | `kMenuWindowID` | 0, 0, 20, 640 | 640 × 20 | 2 | 0 | 0 | 0 | `New Window` | 29 |

Constants at `GliderPRO/Sources/MainWindow.c:16-18`. `procID` 2 = `plainDBox` (borderless,
2-pixel frame); `procID` 4 = `noGrowDocProc` (a title-bar document window with no size box).
All three have `visible = 0`, so the code shows them explicitly.

Creation sites:

| Window | Call | Site |
| --- | --- | --- |
| 129 (editor) | `mainWindow = GetNewCWindow(kEditWindowID, nil, kPutInFront)` | `GliderPRO/Sources/MainWindow.c:192` |
| 130 (fake menu bar) | `menuWindow = GetNewCWindow(kMenuWindowID, nil, kPutInFront)` | `GliderPRO/Sources/MainWindow.c:216` |
| 128 (play) | `mainWindow = GetNewCWindow(kMainWindowID, nil, kPutInFront)` | `GliderPRO/Sources/MainWindow.c:225` |

`'WIND'` 130 is 640 × 20 — a **fake menu bar** window used in full-screen play mode, where the
real menu bar is hidden. 640 is wider than `'WIND'` 128's 512 because it must span the widest
supported screen; the code moves and resizes it at run time.

The floating windoids (Map, Tools, Coordinates, Link) are **not** `'WIND'` resources. They are
built with `NewCWindow(nil, &rect, "\pTitle", false, kWindoidWDEF, kPutInFront, true, 0L)`:

| Windoid | `procID` | WDEF | Site |
| --- | ---: | ---: | --- |
| `Link` | `kWindoidWDEF` 2048 | 128 | `GliderPRO/Sources/Link.c:223` (colour) / `:226` (b&w `NewWindow`) |
| `Tools` | `kWindoidWDEF` 2048 | 128 | `GliderPRO/Sources/Tools.c:292` / `:295` |
| `Tools` (coordinates) | `kWindoidWDEF` 2048 | 128 | `GliderPRO/Sources/Coordinates.c:128` / `:131` |
| `Map` | `kWindoidGrowWDEF` 2064 | 129 | `GliderPRO/Sources/Map.c:375` |

`2048 = 128 × 16 + 0` and `2064 = 129 × 16 + 0` — see §3.11.1.

> `kFloatingKind` = 2048 (`GliderPRO/Sources/WindowUtils.c:14`) is a `windowKind` value that
> happens to equal `kWindoidWDEF`. They are unrelated fields. Do not conflate them.

---

## 3.9 Finder metadata: `'FREF'` 6, `'BNDL'` 1, `'ozm5'` 1, `'vers'` 2

### 3.9.1 `'FREF'` — 6 file references, 7 bytes each

Layout: `OSType fileType` (4) + `int16 localIconID` (2) + `pstring name` (1 byte length = 0).

| ID | `fileType` | `localIconID` | Meaning |
| ---: | --- | ---: | --- |
| 128 | `APPL` | 0 | the application itself |
| 129 | `gliP` | 1 | **P**references file |
| 130 | `gliH` | 2 | **H**ouse (level) file |
| 131 | `gliG` | 3 | saved **G**ame |
| 132 | `gliS` | 4 | **S**ounds file (?) |
| 133 | `gliM` | 5 | **M**ovie / music file (?) |

`localIconID` is a *bundle-local* index, not a resource ID; `'BNDL'` 128 maps index → real
`'ICN#'` ID.

Creator/type codes as used in the code:

| Where | Code |
| --- | --- |
| `GliderPRO/Sources/Prefs.c` — prefs file type | `'gliP'` |
| house files — `HouseIO.c` / `SelectHouse.c` file filter | `'gliH'` |
| saved games | `'gliG'` |
| **creator** for every file the game writes | `'ozm5'` |

`'ozm5'` is the application's creator code and is why the owner resource (§3.9.3) has type
`'ozm5'`. "ozm" = Ozma, matching the `Ozma` object (`'STR#'` 1007 index 129) and the dead
`'STR#'` 128 `Jinjur`/`Crystal`.

### 3.9.2 `'BNDL'` 128 — 68 bytes

| Offset | Size | Field | Observed |
| ---: | ---: | --- | --- |
| 0 | 4 | `owner` OSType | `'ozm5'` |
| 4 | 2 | `version` | **0** |
| 6 | 2 | `numTypes - 1` | 1 (⇒ 2 type arrays) |
| 8 | … | type arrays | see below |

Each type array is `OSType` (4) + `int16 count-1` (2) + `count` × (`int16 localID`,
`int16 resourceID`).

| Type | count | (localID → resID) pairs |
| --- | ---: | --- |
| `FREF` | 6 | 0→128, 1→129, 2→130, 3→131, 4→132, 5→133 |
| `ICN#` | 6 | 0→128, 1→129, 2→130, 3→131, 4→132, 5→133 |

`8 + (4 + 2 + 6*4) * 2 = 68` — matches exactly, all bytes consumed.

Because `localID` == the array index for both arrays, the mapping is the identity
`FREF n ↔ ICN# n`, i.e. FREF 128 (`APPL`) uses ICN# 128, FREF 129 (`gliP`) uses ICN# 129, and
so on. **A Go port has no use for `'BNDL'`/`'FREF'`** — they exist purely so the classic Finder
can pick document icons. Note `version = 0`, not the more usual 1.

### 3.9.3 `'ozm5'` 0 "Owner resource" — 32 bytes

One Pascal string, length `0x1F` = 31: `© 1994-95 Casady & Greene, Inc.` (`©` is MacRoman
`0xA9`). No trailing bytes. This is the classic-Mac "owner resource" convention: a resource
whose *type* equals the application's creator code and whose *ID* is 0, referenced from
`'BNDL'`'s `owner` field. Inert.

### 3.9.4 `'vers'` 1 and 2 — the version the fork actually claims

Layout:

| Offset | Size | Field | `'vers'` 1 | `'vers'` 2 |
| ---: | ---: | --- | --- | --- |
| 0 | 1 | major (BCD) | `0x01` | `0x01` |
| 1 | 1 | minor+bugfix (BCD, 2 nibbles) | `0x12` | `0x12` |
| 2 | 1 | `stage` | `0x80` (`final`) | `0x80` |
| 3 | 1 | `preReleaseLevel` | 0 | 0 |
| 4 | 2 | `regionCode` | 0 (`verUS`) | 0 |
| 6 | … | short version string (pstring) | `1.1.2` | `1.1.2` |
| … | … | long version string (pstring) | `Glider PRO™ 1.1.2\r© 1994-95 Casady & Greene, Inc.` | `© 1994-95 Casady & Greene, Inc.` |
| | | total | 62 bytes | 44 bytes |

`stage` byte values are `0x20` development, `0x40` alpha, `0x60` beta, `0x80` final.
`0x80` + `preReleaseLevel` 0 = a shipping release. `™` is MacRoman `0xAA`; the embedded `\r`
in the long string of `'vers'` 1 is a literal `0x0D`, which the Finder's Get Info window
renders as a line break.

> **The resource fork says 1.1.2. The source distribution is labelled 1.0.4.** These are two
> different version numbering schemes (`'vers'` 1 is the *product* version shown in Get Info;
> "1.0.4" is the name the GPL source release was given). The `.r` file is therefore a dump of
> a **1.1.2-era** resource fork. See "Open questions".

`'vers'` 1 is read at run time:

```
GliderPRO/Sources/About.c:48   wasResFile = CurResFile();
GliderPRO/Sources/About.c:49   UseResFile(thisMac.thisResFile);
GliderPRO/Sources/About.c:55   version = (VersRecHndl)GetResource('vers', 1);
GliderPRO/Sources/About.c:88   UseResFile(wasResFile);
```

The `UseResFile(thisMac.thisResFile)` bracket is required because a house's resource fork may
be on top of the search chain and could shadow `'vers'` 1. `thisMac.thisResFile` is captured
once at startup: `thisMac.thisResFile = CurResFile()`
(`GliderPRO/Sources/Environ.c:443`). **A Go port must reproduce this "always read app
resources from the app, not from the house" discipline** anywhere the house fork is open — the
Resource Manager's implicit search chain is the only reason the code gets away with unqualified
`GetResource` elsewhere.

The About box draws the version by extracting the *short* string from the handle, then draws
its own hard-coded `© 1994-2000 Casady & Greene, Inc.` from DITL 150 item 6 — which disagrees
with `'vers'`.

---

## 3.10 `'clut'` 2, `'PAT#'` 1

### 3.10.1 `'clut'` 128 and 129 — byte-identical copies of the standard Mac 8-bit system palette

Layout (`ColorTable`):

| Offset | Size | Field | Observed |
| ---: | ---: | --- | --- |
| 0 | 4 | `ctSeed` | **0** |
| 4 | 2 | `ctFlags` | **0x0000** (a `'clut'` resource; `0x8000` would mean a PixMap table) |
| 6 | 2 | `ctSize` | **255** = `entryCount - 1` |
| 8 | 8 × 256 | `ColorSpec[256]`: `int16 value` + `RGBColor{uint16 red, green, blue}` | `value == index` for all 256 |

`8 + 8*256 = 2056` bytes — matches both resources exactly. **`'clut'` 128 and `'clut'` 129 are
byte-for-byte identical** (verified with a direct `bytes` comparison).

The palette is the canonical Macintosh 8-bit system CLUT:

| Index range | Count | Content |
| --- | ---: | --- |
| 0-214 | 215 | the 6×6×6 colour cube in order `index = 36·r + 6·g + b`, component value `0xFFFF - 0x3333·i` — i.e. levels `FFFF, CCCC, 9999, 6666, 3333, 0000`. The cube's 216th entry (0,0,0) is **omitted** here and lives at index 255. |
| 215-224 | 10 | pure-red ramp `EE, DD, BB, AA, 88, 77, 55, 44, 22, 11` (high byte doubled: `0xEEEE` …) |
| 225-234 | 10 | pure-green ramp, same 10 levels |
| 235-244 | 10 | pure-blue ramp, same 10 levels |
| 245-254 | 10 | grey ramp, same 10 levels |
| 255 | 1 | black `0000 0000 0000` |

Sampled entries (verified):

| Index | R | G | B | Note |
| ---: | --- | --- | --- | --- |
| 0 | `FFFF` | `FFFF` | `FFFF` | white |
| 1 | `FFFF` | `FFFF` | `CCCC` | |
| 5 | `FFFF` | `FFFF` | `0000` | pure yellow — **special-cased by the fade code** |
| 23 | `FFFF` | `6666` | `0000` | `kRedOrangeColor8` |
| 214 | `0000` | `0000` | `3333` | last cube entry |
| 215 | `EEEE` | `0000` | `0000` | first red-ramp entry |
| 254 | `1111` | `1111` | `1111` | darkest grey |
| 255 | `0000` | `0000` | `0000` | black |

**Neither `'clut'` is loaded by any code path.** `GetCTable` is never called; there is no
`'clut'` 8 (the ID the Palette Manager would use as an application default); the only colour
table the code touches is the *current graphics device's* `pmTable`:

```
GliderPRO/Sources/MainWindow.c:463   thePMap = (*thisGDevice)->gdPMap;
GliderPRO/Sources/MainWindow.c:466   theCTab = (*thePMap)->pmTable;
```

So `'clut'` 128/129 are documentation of the *expected* palette, not an input. That is exactly
what makes them valuable to a port: **they are the authoritative 256-entry RGB table that every
8-bit `'PICT'`, `'cicn'`, `'icl8'` and `'ics8'` index in this fork resolves against.** Hard-code
this table.

The colour→greyscale fade at `GliderPRO/Sources/MainWindow.c:477-496` (and its reverse at
`:560-591`) walks all 256 entries and **skips index 5** (`if (i != 5)`), leaving pure yellow
alone while everything else desaturates. Luminance weights are integer thirds-of-tenths:

```
longGray = (red * 3)/10 + (green * 6)/10 + (blue * 1)/10
```

`kGray2ColorSteps` frames of linear interpolation, abortable with `Button()`.
Both functions are inside `/* ... */` comment blocks in this source drop — the shipped 1.1.2
binary had them, the source release has them disabled.

### 3.10.2 `'PAT#'` 128 — the 7-phase marquee barber pole, 58 bytes

Layout: `int16 count`, then `count` × 8 bytes of 1-bit 8×8 pattern.

`2 + 7*8 = 58` — matches. `count` = 7 = `kNumMarqueePats`
(`GliderPRO/Headers/GliderDefines.h:460`), which also sizes the in-memory array
`Pattern pats[kNumMarqueePats]` (`GliderPRO/Headers/Marquee.h:16`) and the animation
wrap test `if (theMarquee.index >= kNumMarqueePats)` (`GliderPRO/Sources/Marquee.c:45`).

| Index | Pattern rows (8 bytes, MSB = leftmost pixel) |
| ---: | --- |
| 0 | `F8 F1 E3 C7 8F 1F 3E 7C` |
| 1 | `3E 7C F8 F1 E3 C7 8F 1F` |
| 2 | `1F 3E 7C F8 F1 E3 C7 8F` |
| 3 | `8F 1F 3E 7C F8 F1 E3 C7` |
| 4 | `C7 8F 1F 3E 7C F8 F1 E3` |
| 5 | `E3 C7 8F 1F 3E 7C F8 F1` |
| 6 | `F1 E3 C7 8F 1F 3E 7C F8` |

Every phase is the same 8-byte cyclic sequence
`F8, F1, E3, C7, 8F, 1F, 3E, 7C` rotated. Phases 0,1,2,3,4,5,6 begin at sequence indices
**0, 6, 5, 4, 3, 2, 1** — so each phase after the first advances the stripe by exactly one row,
and the rotation that would start at index 7 (`7C F8 F1 E3 C7 8F 1F 3E`) is simply absent from
the list. Rendering the animation is therefore *"scroll the diagonal stripe by one row per
phase"*, with a two-row jump on the 0→1 transition, wrapping after 7 phases (not 8 — the
missing eighth rotation is why the loop is 7 frames and the stripe appears to creep).

Each byte's bit pattern: `0xF8` = `11111000`, `0x7C` = `01111100`, `0x3E` = `00111110`,
`0x1F` = `00011111`, `0x8F` = `10001111`, `0xC7` = `11000111`, `0xE3` = `11100011`,
`0xF1` = `11110001` — a 5-on/3-off run shifting right one bit per row: a 45° stripe.

Loaded once into a 7-element array:

```
GliderPRO/Sources/Marquee.c:15    #define kMarqueePatListID    128
GliderPRO/Sources/Marquee.c:503   for (i = 0; i < kNumMarqueePats; i++)
GliderPRO/Sources/Marquee.c:504       GetIndPattern(&theMarquee.pats[i], kMarqueePatListID, i + 1);
```

`GetIndPattern` is **1-based**, so `i` 0..6 fetches patterns 1..7.

> Mac Toolbox note: `PenPat`/`FrameRect` with a `Pattern` is a 1-bit stipple applied in the
> current fore/back colour, aligned to the *port origin*, not to the shape being drawn.
> A Go port must anchor the pattern to the same origin (the window's top-left) or the marquee
> will visibly slide when the selection rectangle moves.

---

## 3.11 Third-party 68k code: `'WDEF'` 2, `'CDEF'` 1

These three resources are **compiled 68000 machine code**, not data. They are the only
executable resources in the dump — there are no `'CODE'` resources (see §3.11.3).

### 3.11.1 `'WDEF'` 128 and 129 — "Infinity Windoid 2.6"

| ID | Name | Bytes | Attrs | First 16 bytes |
| ---: | --- | ---: | --- | --- |
| 128 | `Infinity Windoid 2.6` | 5,452 | `purgeable` | `4E56 FFFC 48E7 1708 3C2E 000C 2E2E 0008` |
| 129 | `Infinity Windoid 2.6 (grow)` | 6,476 | `purgeable` | `4E56 FFFC 48E7 1708 3C2E 000C 2E2E 0008` |

Both begin with the identical 68k prologue:

| Bytes | 68k instruction | Meaning |
| --- | --- | --- |
| `4E56 FFFC` | `LINK A6, #-4` | build a stack frame with 4 bytes of locals |
| `48E7 1708` | `MOVEM.L D3/D5-D7/A4, -(SP)` | save registers (mask `0x1708` = bits 12,10,9,8,3 → D3, D5, D6, D7, A4; **D4 is not saved**) |
| `3C2E 000C` | `MOVE.W 12(A6), D6` | load the `message` parameter (the third of the four; with Pascal left-to-right pushes the frame is `8(A6)=param`, `12(A6)=message`, `14(A6)=theWindow`, `18(A6)=varCode`) |
| `2E2E 0008` | `MOVE.L 8(A6), D7` | load the last-pushed parameter (`param`) |

This is a Pascal-calling-convention `pascal long WDEF(short varCode, WindowPtr theWindow,
short message, long param)` entry point compiled by MPW C or THINK C.

Referenced only through window `procID`s:

```
GliderPRO/Headers/GliderDefines.h:531   #define kWindoidWDEF        2048   // = 128*16 + 0
GliderPRO/Headers/GliderDefines.h:532   #define kWindoidGrowWDEF    2064   // = 129*16 + 0
```

so WDEF 128 = plain floating windoid (Link, Tools, Coordinates), WDEF 129 = growable floating
windoid (Map). See the creation-site table in §3.8.2.

**Infinity Windoid** was a widely-licensed shareware WDEF by Troy Gaul that drew the small
"floating palette" title bar of the era: a thin (roughly 11-pixel) title bar with horizontal
racing stripes, a small close box on the left, and — in the `(grow)` variant — a size box.
`purgeable` is set on both because a WDEF is only needed while a windoid is being drawn.

> **A Go port cannot execute these.** It must re-implement the windoid chrome from scratch, or
> more sensibly draw the four palettes as plain rectangles with a custom title strip. Nothing
> about game behaviour depends on the WDEF; only the pixel appearance of four editor palettes
> does. The exact stripe geometry is *not recoverable from this source drop* — it lives inside
> 12 KB of 68k object code. The only ground truth available is the content-rect sizes the code
> requests (`Link` 129 × 30 at `GliderPRO/Sources/Link.c:220`).

### 3.11.2 `'CDEF'` 128 — the pop-up menu control definition, 5,978 bytes

| ID | Name | Bytes | Attrs | First 16 bytes |
| ---: | --- | ---: | --- | --- |
| 128 | *(unnamed)* | 5,978 | — | `600E 0000 4344 4546 0002 0000 0000 0000` |

Header decode:

| Bytes | Meaning |
| --- | --- |
| `600E` | 68k `BRA.S *+16` — jump over the 14-byte signature block to the real entry |
| `0000` | padding |
| `43444546` | ASCII **`CDEF`** — a self-identifying signature |
| `0002` | version / variant marker |
| `0000 0000 0000` | reserved |

Referenced by `'CNTL'` 128 and 129, both with `procID` = 2050 = `128 × 16 + 2`, i.e. **CDEF
128, variation code 2**. Variation code 2 is this CDEF's "pop-up menu whose `'MENU'` ID comes
from the control's `refCon`" mode — which is exactly how `'CNTL'` 129 carries `refCon = 141`.

`'CNTL'` 128 has `refCon = 0` and is instantiated by the Dialog Manager from DITL 1003's
`resCtrl` item; the code fetches `'MENU'` 140 itself
(`GliderPRO/Sources/RoomInfo.c:397`) and attaches it, so `refCon` is unused there.

> This predates Apple's own `popupMenuCDEFproc` (`procID` 1008 = CDEF 63) becoming reliable, so
> a third-party pop-up CDEF was shipped. A Go port replaces it with a native combo box /
> dropdown; the only behavioural contract to preserve is the value ↔ menu-item-number
> identity, and the out-of-range value 19 quirk noted in §3.8.1.

### 3.11.3 There are no `'CODE'` and no `'SIZE'` resources — and that tells us what this file is

Neither type appears anywhere in the 538 resources. Yet:

* Every classic Mac 68k application has at least `'CODE'` 0 (the jump table) and `'CODE'` 1.
* `GliderPRO/Sources/Environ.c:698-732` (`SetAppMemorySize`) reads and rewrites `'SIZE'`
  resources −1, 0 and 1 in `thisMac.thisResFile`, so a `'SIZE'` resource certainly exists in
  the shipped application.

Conclusion: **`Glider PRO.r` is a DeRez of the *build-input* resource file** (the "resources
only" file the linker merges into the application), **not of the finished application**.
`'CODE'` comes from compiling `Sources/`, and `'SIZE'` from the project's memory settings; both
are added at link time and are therefore absent here. This is good news for a port — the file
contains exactly the hand-authored assets and nothing machine-generated, except the three
third-party 68k blobs in §3.11.1-§3.11.2 which were themselves hand-added.

For completeness, the `'SIZE'` layout the code assumes:

```
GliderPRO/Sources/Environ.c:31-36     typedef struct { short flags; long mem1; long mem2; } sizeType;
GliderPRO/Sources/Environ.c:710        tempResource = Get1Resource('SIZE', i);   // i = 0, 1  -> removed
GliderPRO/Sources/Environ.c:719        tempResource = Get1Resource('SIZE', -1);  // the template -> patched
GliderPRO/Sources/Environ.c:723-724    ->mem1 = newSize;  ->mem2 = newSize;
```

i.e. `flags` (2 bytes) then `preferredSize` and `minimumSize` as `long`s. The call site is
commented out (`GliderPRO/Sources/Environ.c:687`), so the shipped code only *warns* about low
memory (ALRT 180 / 1037) and calls `ExitToShell()`. **Go port: delete this entirely.**

---

## 3.12 `'demo'` 128 — the recorded attract-mode demo, 6,702 bytes

### 3.12.1 Record layout

```
GliderPRO/Headers/GliderStructs.h:335-339
typedef struct
{
    long        frame;      // 4 bytes, big-endian
    char        key;        // 1 byte
    char        padding;    // 1 byte
} demoType, *demoPtr;
```

`sizeof(demoType)` = **6** with 2-byte alignment (68k / PowerPC `long` alignment does not pad
this struct further because `long` is already at offset 0).

`kDemoLength` = **6702** (`GliderPRO/Headers/GliderDefines.h:625`).
`6702 / 6 = 1117` — **exactly 1,117 records, no remainder.** Verified.

### 3.12.2 Verified contents

| Property | Observed |
| --- | --- |
| Record count | 1,117 |
| `frame` range | 46 … 3,414 |
| `frame` monotonic | **strictly increasing** — no duplicate frames |
| `key` histogram | `{0: 910, 1: 198, 3: 9}` — **key 2 never appears** |
| `padding` distinct values | **over 100** distinct bytes, `{0: 407, 1: 18, …, 255: 42}` |
| First 8 records `(frame, key, padding)` | `(46,0,114) (47,0,114) (48,0,114) (49,0,69) (56,0,6) (57,0,114) (58,0,111) (59,0,114)` |

The `padding` byte is **uninitialised heap garbage** and must be ignored. Proof from the
recorder — it writes `frame` and `key` and never touches `padding`:

```
GliderPRO/Sources/Input.c:44-51
void LogDemoKey (char keyIs)
{
    demoData[demoIndex].frame = gameFrame;
    demoData[demoIndex].key = keyIs;
    demoIndex++;
}
```

The garbage values cluster around printable ASCII (114 = `'r'`, 111 = `'o'`, 69 = `'E'`) —
the demo buffer was allocated over memory that had recently held text.

### 3.12.3 Loading

```
GliderPRO/Sources/StructuresInit2.c:285-298
    demoData = nil;
    demoData = (demoPtr)NewPtr(kDemoLength);
    if (demoData == nil)
        RedAlert(kErrNoMemory);
    tempHandle = GetResource('demo', 128);
    if (tempHandle == nil)
        RedAlert(kErrNoMemory);
    else
    {
        BlockMove(*tempHandle, demoData, kDemoLength);
        ReleaseResource(tempHandle);
    }
```

Note it `BlockMove`s exactly `kDemoLength` bytes without checking the resource's real size, and
under `#ifdef CREATEDEMODATA` it instead allocates `sizeof(demoType) * 2000` = 12,000 bytes and
records into it (`GliderPRO/Sources/StructuresInit2.c:281-283`), later dumped with
`DumpToResEditFile((Ptr)demoData, sizeof(demoType) * (long)demoIndex)`
(`GliderPRO/Sources/Play.c:217`) and written with
`AddResource(newResource, 'demo', 128, "\p")` / `ChangedResource`
(`GliderPRO/Sources/DebugUtilities.c:352-353`). So the shipped `'demo'` 128 was produced by
playing the game with `CREATEDEMODATA` defined and 1,117 keystroke events were captured.

`kDemoLength` is also charged to the memory budget:
`bytesNeeded += kDemoLength;` (`GliderPRO/Sources/Environ.c:657`).

### 3.12.4 Playback semantics

```
GliderPRO/Sources/Input.c:224-267   (inside GetDemoInput)
 1. if (gameFrame == (long)demoData[demoIndex].frame)
 2.     switch (demoData[demoIndex].key)
 3.       case 0:  // comment says "left key"
 4.           thisGlider->hDesiredVel += kNormalThrust;
 5.           thisGlider->tipped = (thisGlider->facing == kFaceLeft);
 6.           thisGlider->heldRight = true;
 7.           thisGlider->fireHeld = false;
 8.       case 1:  // comment says "right key"
 9.           thisGlider->hDesiredVel -= kNormalThrust;
10.           thisGlider->tipped = (thisGlider->facing == kFaceRight);
11.           thisGlider->heldLeft  = true;
12.           thisGlider->fireHeld = false;
13.       case 2:  // "battery key"
14.           if (batteryTotal > 0) DoBatteryEngaged(thisGlider);
15.           else                  DoHeliumEngaged(thisGlider);
16.           thisGlider->fireHeld = false;
17.       case 3:  // "rubber band key"
18.           if (!thisGlider->fireHeld)
19.               if (AddBand(thisGlider, dest.left + 24, dest.top + 10, facing))
20.                   bandsTotal--;  if (bandsTotal <= 0) QuickBandsRefresh(false);
21.                   thisGlider->fireHeld = true;
22.     demoIndex++;
23. else
24.     thisGlider->fireHeld = false;
```

Key observations a port must copy exactly:

1. **`key` is an *event*, not a held state.** A record fires on the single frame where
   `gameFrame == frame`, then `demoIndex` advances. On every other frame `fireHeld` is cleared.
   Because `frame` is strictly increasing and `gameFrame` increments by 1, a record can never
   be skipped — but if a port ever lets `gameFrame` jump, playback desynchronises permanently
   (there is no re-sync).
2. **The case-0 / case-1 comments are inverted relative to the code.** `case 0` is commented
   `// left key` yet does `hDesiredVel += kNormalThrust` and `heldRight = true`; `case 1` is
   commented `// right key` yet does `-= kNormalThrust` and `heldLeft = true`. Compare
   `GetInput` (`GliderPRO/Sources/Input.c:281`), where the real key handler is authoritative.
   **Port the code, not the comments**; otherwise the demo glider flies backwards.
3. `key == 2` (battery/helium) is **never used** in the shipped recording, so
   `DoBatteryEngaged`/`DoHeliumEngaged` are unexercised by the demo.
4. Playback compares `gameFrame` (a `long`) to `demoData[].frame` cast to `long`. With frames
   46..3414 and `kTicksPerFrame` = 2 (`GliderPRO/Headers/GliderDefines.h:533`), the demo runs
   3,414 frames ≈ 6,828 ticks ≈ **114 seconds** at 60 ticks/second.
5. `demoIndex` is never bounds-checked against 1,117. The demo ends because the *game* ends
   (`demoGoing` is cleared elsewhere), not because the record list runs out. If a port lets the
   demo run past frame 3,414 it will read past the buffer.

---

# Part 4 — Resource Manager usage in the code

This part is the complete API surface. It is what a Go port has to replace with a loader.

## 4.1 Census of Resource Manager and resource-fetching calls

Counted over `GliderPRO/Sources/*.c` and `GliderPRO/Headers/*.h` (converted to LF). Counts
include commented-out lines; those are called out individually below.

| Call | Sites | Notes |
| --- | ---: | --- |
| `GetResource` | 22 | see §4.2 |
| `Get1Resource` | 3 | `'SIZE'` ×2 (dead), `'icl8'` ×1 |
| `Get1IndResource` | 2 | 1 live (`'PICT'` index 1), 1 commented out |
| `Count1Resources` | 2 | 1 live (`'PICT'`), 1 commented out |
| `GetResInfo` | 2 | 1 live, 1 commented out |
| `DetachResource` | 1 | `'CURS'` frames of the animated cursor |
| `AddResource` | 1 | `'demo'` 128, under `#ifdef CREATEDEMODATA` |
| `RemoveResource` | 1 | `'SIZE'` (dead) |
| `ChangedResource` | 2 | `'demo'` (dev-only), `'SIZE'` (dead) |
| `WriteResource` | 1 | `'SIZE'` (dead) |
| `UpdateResFile` | 1 | `'SIZE'` (dead) |
| `ReleaseResource` | 27 | |
| `SetResLoad` | 6 | 3 in `Sound.c`, 3 in `Music.c` |
| `GetMaxResourceSize` | 2 | `Sound.c:507`, `Music.c:402` |
| `CurResFile` | 4 | `About.c:48`, `Environ.c:443`, `Environ.c:705`, `SelectHouse.c:87` |
| `UseResFile` | 6 | `About.c:49`/`:88`, `Environ.c:706`/`:732`, `HouseIO.c:575`, `SelectHouse.c:131` |
| `HOpenResFile` | 1 | `SelectHouse.c:102` |
| `FSpOpenResFile` | 1 | `HouseIO.c:571` |
| `CloseResFile` | 2 | `HouseIO.c:586`, `SelectHouse.c:113` |
| `CreateResFile` | 1 | `DebugUtilities.c:342` (dev-only; the second textual hit at `:344` is a `DebugStr` literal, not a call) |
| `HCreateResFile` | 2 | **live code**: `House.c:88` and `HouseIO.c:270`, each right after `FSpCreate(&spec, 'ozm5', 'gliH', script)`, to give a brand-new house file an (empty) resource fork; failure raises `YellowAlert(kYellowFailedResCreate, ResError())` |
| `ResError` | 10 | |
| `GetPicture` | 18 | typed wrapper for `GetResource('PICT', …)` |
| `GetIndString` | 28 | `'STR#'` |
| `GetString` | 1 | `'STR '` −16096, **from the System file** |
| `GetMenu` | 5 | `'MENU'` 128, 129, 130, 131, 140 |
| `GetNewDialog` | 9 | `'DLOG'` |
| `GetNewCWindow` | 3 | `'WIND'` 128, 129, 130 |
| `GetNewControl` | 3 | `'CNTL'` 129, 130, 131 |
| `GetCursor` | 8 | `'CURS'` |
| `GetCCursor` | 1 | `'crsr'`, `AnimCursor.c:87` |
| `GetCIcon` | 1 | `'cicn'`, `Utilities.c:398` |
| `GetIconSuite` | 1 | `'ICON'` family, `Utilities.c:384` |
| `PlotIconSuite` | 1 | `Utilities.c:386` |
| `GetIndPattern` | 1 | `'PAT#'` 128, `Marquee.c:504` |

**Never called anywhere:** `GetNamedResource`, `Get1NamedResource`, `CountResources`,
`SetResInfo`, `GetResAttrs`, `SetResAttrs`, `HomeResFile`, `SizeResource`, `MaxSizeRsrc`,
`GetCTable`, `GetPattern`, `GetIcon`, `GetResFileAttrs`, `SetResPurge`. In particular
**nothing is ever looked up by resource *name*** — the 215 names in the fork (§2.4) are purely
documentary. A Go asset loader can key everything on `(type, id)`.

## 4.2 Every `GetResource` call site, with the resource type

| Site | Type | ID expression | Purpose |
| --- | --- | --- | --- |
| `GliderPRO/Sources/About.c:55` | `'vers'` | `1` | version string in the About box |
| `GliderPRO/Sources/AnimCursor.c:116` | `'acur'` | `128` | beach-ball cursor sequence |
| `GliderPRO/Sources/AnimCursor.c:142` | `'acur'` | `rAcurID` = 1000 | **dead** — no `'acur'` 1000 exists |
| `GliderPRO/Sources/Music.c:234` | `'snd '` | `i + kBaseBufferMusicID` (2000-2006) | load a music track for playback |
| `GliderPRO/Sources/Music.c:396` | `'snd '` | `i + kBaseBufferMusicID` | size probe under `SetResLoad(false)` |
| `GliderPRO/Sources/DialogUtils.c:51` | `'DLOG'` | `sfPutDialogID` | commented-out `GetPutDialogCorner` |
| `GliderPRO/Sources/DialogUtils.c:85` | `'DLOG'` | `sfGetDialogID` | commented-out `GetGetDialogCorner` |
| `GliderPRO/Sources/DialogUtils.c:115` | `'DLOG'` | `dialogID` | commented-out `CenterDialog` |
| `GliderPRO/Sources/DialogUtils.c:142` | `'DLOG'` | `dialogID` | `GetDialogRect` — **no callers** |
| `GliderPRO/Sources/DialogUtils.c:167` | `'DLOG'` | `dialogID` | commented-out `TrueCenterDialog` |
| `GliderPRO/Sources/DialogUtils.c:202` | `'ALRT'` | `alertID` | commented-out `CenterAlert` |
| `GliderPRO/Sources/DialogUtils.c:242` | `'DLOG'` | `dialogID` | commented-out `ZoomOutDialogRect` |
| `GliderPRO/Sources/DialogUtils.c:296` | `'ALRT'` | `alertID` | commented-out `ZoomOutAlertRect` |
| `GliderPRO/Sources/Room.c:272` | **`'Date'`** | `theID` | fallback for a house's custom background |
| `GliderPRO/Sources/Room.c:942` | **`'bnds'`** | `theID` | which edges of a custom background are solid |
| `GliderPRO/Sources/Map.c:172` | **`'Date'`** | `resID` | same fallback, map thumbnails |
| `GliderPRO/Sources/RoomInfo.c:845` | **`'Date'`** | `theID` | same fallback, `PictIDExists` |
| `GliderPRO/Sources/RoomGraphics.c:142` | **`'Date'`** | `resID` | same fallback, room render |
| `GliderPRO/Sources/StructuresInit2.c:290` | `'demo'` | `128` | attract-mode recording |
| `GliderPRO/Sources/Sound.c:279` | `'snd '` | `soundID` | `LoadTriggerSound` — fills the `kTriggerSound` slot |
| `GliderPRO/Sources/Sound.c:327` | `'snd '` | `i + kBaseBufferSoundID` (1000-1062) | load an effect into the sound buffer |
| `GliderPRO/Sources/Sound.c:501` | `'snd '` | `i + kBaseBufferSoundID` | size probe under `SetResLoad(false)` |

### Two resource types the application fork does **not** contain

`'Date'` and `'bnds'` are fetched by ID but exist in **house files**, never in `Glider PRO.r`
(verified: neither type is among the 35 in §2.1).

* **`'Date'`** — a second home for a custom background picture. The lookup is always a *fallback
  after* `GetPicture` fails:

  ```
  GliderPRO/Sources/Room.c:269-278
   1. thePicture = GetPicture(theID);
   2. if (thePicture == nil)
   3.     thePicture = (PicHandle)GetResource('Date', theID);
   4.     if (thePicture == nil)
   5.         YellowAlert(kYellowNoBackground, 0);  // 'STR#' 1006 index 7
   6.         return;
   7. dest = (*thePicture)->picFrame;
   8. QOffsetRect(&dest, -dest.left, -dest.top);
   9. DrawPicture(thePicture, &dest);
  10. ReleaseResource((Handle)thePicture);
  ```

  The handle is cast straight to `PicHandle` and its `picFrame` read, so a `'Date'` resource is
  bit-for-bit a `'PICT'` under a different four-char code. The same three-step
  `GetPicture` → `'Date'` → `YellowAlert` pattern appears at `Map.c:172`,
  `RoomInfo.c:845` and `RoomGraphics.c:142`. **A Go house loader must accept `'Date'` as an
  alias for `'PICT'`.**

* **`'bnds'`** — 4 bytes, one `Boolean` per edge:

  ```
  GliderPRO/Headers/GliderStructs.h:265-272
  typedef struct { Boolean left; Boolean top; Boolean right; Boolean bottom; }
      boundsType, *boundsPtr, **boundsHand;
  ```

  Decoded into a bit code:

  ```
  GliderPRO/Sources/Room.c:937-965   GetOriginalBounding(short theID)
   1. boundsRes = (boundsHand)GetResource('bnds', theID);
   2. if (boundsRes == nil)
   3.     if (PictIDExists(theID)) YellowAlert(kYellowNoBoundsRes, 0);  // 'STR#' 1006 index 9
   4.     boundCode = 0;
   5. else
   6.     boundCode = 0;
   7.     if ((*boundsRes)->left)   boundCode += 1;
   8.     if ((*boundsRes)->top)    boundCode += 2;
   9.     if ((*boundsRes)->right)  boundCode += 4;
  10.     if ((*boundsRes)->bottom) boundCode += 8;
  11.     ReleaseResource((Handle)boundsRes);
  12. return boundCode;
  ```

  So the bit weights are **left = 1, top = 2, right = 4, bottom = 8**, and a missing `'bnds'`
  means `0` (all four edges open). Note the asymmetry: a missing `'bnds'` only warns if a
  picture with that ID *does* exist.

## 4.3 The resource search chain

Classic Mac resource lookups walk a per-process chain of open resource files, most recently
opened first, then the application, then the System file. Glider PRO opens **three** kinds:

| File | Opened at | Closed at | Held in |
| --- | --- | --- | --- |
| the application (implicit) | at launch, by the Process Manager | never | `thisMac.thisResFile` (`GliderPRO/Sources/Environ.c:443`) |
| the current house | `houseResFork = FSpOpenResFile(&theHousesSpecs[thisHouseIndex], fsCurPerm)` — `GliderPRO/Sources/HouseIO.c:571` | `CloseResFile(houseResFork)` — `GliderPRO/Sources/HouseIO.c:586` | `houseResFork`, sentinel `-1` |
| each candidate house, while scanning | `isResFile = HOpenResFile(theHousesSpecs[i].vRefNum, …)` — `GliderPRO/Sources/SelectHouse.c:102` | `CloseResFile(isResFile)` — `GliderPRO/Sources/SelectHouse.c:113` | local |

```
GliderPRO/Sources/HouseIO.c:567-578   OpenHouseResFork()
 1. if (houseResFork == -1)
 2.     houseResFork = FSpOpenResFile(&theHousesSpecs[thisHouseIndex], fsCurPerm);
 3.     if (houseResFork == -1)
 4.         YellowAlert(kYellowFailedResOpen, ResError());   // 'STR#' 1006 index 2
 5.     else
 6.         UseResFile(houseResFork);
```

Consequences a port must reproduce:

1. **While a house is open, unqualified `GetResource`/`GetPicture` search the house first.**
   That is exactly how a house supplies custom background `'PICT'`s at IDs ≥ `kUserBackground`
   = 3000 (`GliderPRO/Headers/GliderDefines.h:522`; the enforced upper bound is 3800, at
   `GliderPRO/Sources/RoomInfo.c:762` — see "Open questions" #4) that shadow nothing in the app.
2. Because the game's own `'PICT'` IDs are 150-153, 1000-1023, 1202-1217, 1988-2017, 3903-4018,
   4998-5018 and 10000, **a house could shadow game art by using one of those IDs**. Nothing
   prevents it. `Count1Resources('PICT')` at `GliderPRO/Sources/House.c:250` uses the `1`-suffixed
   ("current file only") variant precisely to count *the house's* PICTs and nothing else:

   ```
   GliderPRO/Sources/House.c:246-252   HouseHasOriginalPicts()
    1. nPicts = Count1Resources('PICT');
    2. return (nPicts > 0);
   ```

   and `GetFirstPICT` picks the house's first PICT ID:

   ```
   GliderPRO/Sources/RoomInfo.c:876-895   GetFirstPICT()
    1. resHandle = Get1IndResource('PICT', 1);
    2. if (resHandle != nil)
    3.     GetResInfo(resHandle, &resID, &resType, resName);
    4.     ReleaseResource(resHandle);
    5.     return resID;
    6. return -1;
   ```

   **`Get1IndResource` index order is the resource map's physical order, not sorted by ID.** In
   this application's own fork the IDs within a type run are *not* sorted (§2.5, e.g. `'STR#'`
   order 160, 170, 171, 128, 1006, 1007, 129, 150, 140, 1005). A Go port must therefore
   preserve a house's on-disk resource order if it wants `GetFirstPICT` to return the same ID.
3. `About.c` and `SelectHouse.c` both bracket their lookups with
   `wasResFile = CurResFile(); UseResFile(thisMac.thisResFile); … UseResFile(wasResFile);`
   to force app-only lookups (`GliderPRO/Sources/About.c:48-49`/`:88`,
   `GliderPRO/Sources/SelectHouse.c:87`/`:131`).
4. `Get1Resource('icl8', -16455)` at `GliderPRO/Sources/SelectHouse.c:106` is executed against
   the *house* file that was just `HOpenResFile`d, to test whether the house has a custom Finder
   icon (see §3.4.5).

## 4.4 `SetResLoad(false)` — sizing without loading

Both audio subsystems compute their memory footprint by walking the resources with loading
disabled, so `GetResource` returns an *empty* handle whose logical size can still be queried
with `GetMaxResourceSize`:

```
GliderPRO/Sources/Sound.c:491-512   SoundBytesNeeded()
 1. totalBytes = 0L;
 2. SetResLoad(false);
 3. for (i = 0; i < kMaxSounds - 1; i++)          // kMaxSounds = 64, so i = 0..62
 4.     theSound = GetResource('snd ', i + kBaseBufferSoundID);   // 1000..1062
 5.     if (theSound == nil)
 6.         SetResLoad(true);
 7.         return (long)ResError();
 8.     totalBytes += GetMaxResourceSize(theSound);
 9.     // ReleaseResource(theSound);            <- commented out: handles are leaked
10. SetResLoad(true);
11. return totalBytes;
```

```
GliderPRO/Sources/Music.c:386-407   MusicBytesNeeded()
 1. totalBytes = 0L;
 2. SetResLoad(false);
 3. for (i = 0; i < kMaxMusic; i++)               // kMaxMusic = 7, so i = 0..6
 4.     theSound = GetResource('snd ', i + kBaseBufferMusicID);   // 2000..2006
 5.     if (theSound == nil) { SetResLoad(true); return (long)ResError(); }
 6.     totalBytes += GetMaxResourceSize(theSound);
 7. SetResLoad(true);
 8. return totalBytes;
```

Constants: `kBaseBufferSoundID` = 1000 and `kMaxSounds` = 64
(`GliderPRO/Sources/Sound.c:14-15`); `kBaseBufferMusicID` = 2000 and `kMaxMusic` = 7
(`GliderPRO/Sources/Music.c:15-16`).

> **The `- 1` in `i < kMaxSounds - 1` is load-bearing.** It stops the loop at ID 1062, skipping
> slot 63 = `'snd '` **1063**, which does not exist in the fork; slot 63 is `kTriggerSound`
> (`GliderPRO/Headers/GliderDefines.h:118`) and is filled at run time by `LoadTriggerSound`
> (`GliderPRO/Sources/Sound.c:279`) with whatever `'snd '` ID the room's sound trigger names.
> A Go port must keep 64 slots with slot 63 dynamic.
> Verified against the data: `'snd '` IDs present are exactly 1000-1062 and 2000-2006 (§2.1),
> so `SoundBytesNeeded` returns `Σ|snd 1000..1062|` = **352,348** bytes and `MusicBytesNeeded`
> returns `Σ|snd 2000..2006|` = **776,996** bytes.

Both loops **leak** their handles (the `ReleaseResource` is commented out). Harmless with
`SetResLoad(false)` because the handles carry no data, but it means the resource map keeps 63
and 7 empty handles alive.

## 4.5 `DetachResource` — the only "own the bytes" case

```
GliderPRO/Sources/AnimCursor.c:49-68   GetMonoCursors(acurHandle ballCursH)
 1. j = (*ballCursH)->n;                       // 12, from 'acur' 128
 2. for (i = 0; i < j; i++)
 3.     cursHdl = GetCursor((*ballCursH)->frame[i].resID);   // 'CURS' 160..149
 4.     if (cursHdl == nil)
 5.         for (j = 0; j < i; j++) DisposeHandle((*ballCursH)->frame[j].cursorHdl);
 6.         return false;
 7.     DetachResource((Handle)cursHdl);        // sever from the resource map
 8.     (*ballCursH)->frame[i].cursorHdl = (Handle)cursHdl;
 9. return true;
```

`DetachResource` turns a resource handle into an ordinary heap handle so the resource map no
longer owns it (and `ReleaseResource` must not be called on it — `DisposeHandle` is used
instead, line 5). The colour path (`GetColorCursors`,
`GliderPRO/Sources/AnimCursor.c:75-105`) uses `GetCCursor` / `DisposeCCursor` and does **not**
detach, because `GetCCursor` already returns a non-resource handle.

**Go port:** every asset is an owned value; `DetachResource` has no analogue and disappears.

## 4.6 Typed wrappers the code actually uses

| Wrapper | Resource | Definition |
| --- | --- | --- |
| `LoadGraphic(resID)` | `'PICT'`, unscaled, drawn at the port's top-left | `GliderPRO/Sources/Utilities.c:317-333` |
| `LoadScaledGraphic(resID, theRect)` | `'PICT'`, **scaled to `theRect`** | `GliderPRO/Sources/Utilities.c:340-349` |
| `LargeIconPlot(theRect, theID)` | `'ICON'` family via icon suite | `GliderPRO/Sources/Utilities.c:379-387` |
| `DrawCIcon(theID, h, v)` | `'cicn'`, always 32 × 32 at (h, v) | `GliderPRO/Sources/Utilities.c:393-406` |
| `GetLocalizedString(index, theString)` | `'STR#'` 150 | `GliderPRO/Sources/StringUtils.c:325` |
| `RedAlert(errorNumber)` | `'STR#'` 170/171 + ALRT 170, then `ExitToShell()` | `GliderPRO/Sources/Utilities.c:135-161` |
| `YellowAlert(whichAlert, identifier)` | `'STR#'` 1006 + ALRT 1006 | `GliderPRO/Sources/HouseIO.c:642-654` |
| `BringUpDialog(dialogID)` | `'DLOG'` via `GetNewDialog` | `GliderPRO/Sources/DialogUtils.c:24-33` |
| `LoadDialogPICT(theDialog, item, resID)` | `'PICT'` into a dialog item rect (scaled) | `GliderPRO/Sources/DialogUtils.c` |

```
GliderPRO/Sources/Utilities.c:317-333   LoadGraphic(short resID)
 1. thePicture = GetPicture(resID);
 2. if (thePicture == nil) RedAlert(kErrFailedGraphicLoad);   // 'STR#' 170/171 index 5
 3. HLock((Handle)thePicture);
 4. bounds = (*thePicture)->picFrame;
 5. HUnlock((Handle)thePicture);
 6. OffsetRect(&bounds, -bounds.left, -bounds.top);   // normalise origin to (0,0)
 7. DrawPicture(thePicture, &bounds);                 // 1:1, no scaling
 8. ReleaseResource((Handle)thePicture);
```

```
GliderPRO/Sources/Utilities.c:340-349   LoadScaledGraphic(short resID, Rect *theRect)
 1. thePicture = GetPicture(resID);
 2. if (thePicture == nil) RedAlert(kErrFailedGraphicLoad);
 3. DrawPicture(thePicture, theRect);                 // QuickDraw scales picFrame -> theRect
 4. ReleaseResource((Handle)thePicture);
```

Note step 6 of `LoadGraphic`: `picFrame` is normalised to a (0,0) origin **before** drawing, so
the two `'PICT'`s with non-zero `picFrame` origins — 5006 `(top=0, left=91, bottom=120,
right=115)` and 5010 `(top=195, left=0, bottom=230, right=40)` (§3.1) — are drawn at the port's
top-left with their *sizes* (24 × 120 and 40 × 35) rather than at their recorded offsets. `LoadScaledGraphic` does not
normalise, so with those two the destination rect determines everything.

```
GliderPRO/Sources/Utilities.c:379-387   LargeIconPlot(Rect *theRect, short theID)
 1. theErr = GetIconSuite(&theSuite, theID, svAllLargeData);   // 'ICN#' + 'icl4' + 'icl8'
 2. if (theErr == noErr)
 3.     theErr = PlotIconSuite(theRect, atNone, ttNone, theSuite);
 4. // theSuite is never disposed -> leaked on every call
```

```
GliderPRO/Sources/Utilities.c:393-406   DrawCIcon(short theID, short h, short v)
 1. theIcon = GetCIcon(theID);
 2. if (theIcon != nil)
 3.     SetRect(&theRect, 0, 0, 32, 32);
 4.     OffsetRect(&theRect, h, v);
 5.     PlotCIcon(&theRect, theIcon);
 6.     DisposeCIcon(theIcon);
```

`DrawCIcon` hard-codes 32 × 32 — consistent with the measurement that **all 44 `'cicn'` are
32 × 32** (§3.4.1). It reloads and disposes the icon on *every* call, i.e. once per redraw.

`RedAlert` is the fatal path:

```
GliderPRO/Sources/Utilities.c:135-161   RedAlert(short errorNumber)
 1. #define rDeathAlertID 170 ; rErrTitleID 170 ; rErrMssgID 171
 2. InitCursor();
 3. if (errorNumber > 1)
 4.     GetIndString(errTitle,   rErrTitleID, errorNumber);
 5.     GetIndString(errMessage, rErrMssgID,  errorNumber);
 6. else                                   // <= 0 and 1 both fall back to index 1
 7.     GetIndString(errTitle,   rErrTitleID, 1);
 8.     GetIndString(errMessage, rErrMssgID,  1);
 9. NumToString((long)errorNumber, errNumberString);
10. ParamText(errTitle, errMessage, errNumberString, "\p");
11. dummyInt = Alert(rDeathAlertID, nil);
12. ExitToShell();
```

Note `errorNumber > 1` at line 3: index 1 (`Unaccounted for Error`) is the clamp for `<= 1`.
No upper clamp — `errorNumber > 13` would read past the list; `GetIndString` returns an empty
string in that case rather than crashing.

## 4.7 Read/write asymmetry

The application **only reads** its own fork at run time. The two write paths are both
effectively disabled:

| Write path | Status |
| --- | --- |
| `SetAppMemorySize` (`'SIZE'`) — `GliderPRO/Sources/Environ.c:698-732` | call site commented out at `GliderPRO/Sources/Environ.c:687`; the code `ExitToShell()`s instead |
| `'demo'` 128 authoring — `GliderPRO/Sources/DebugUtilities.c:342-353` | inside `#ifdef CREATEDEMODATA` |

House files, by contrast, are written: the house's resource fork gains `'PICT'`/`'Date'` and
`'bnds'` resources when the editor imports custom art, guarded by `'STR#'` 1006 indices 3 and 4
(`I failed to add a resource…`, `I failed to create a new resource fork…`).

**A Go port needs a resource *reader* only for the app's assets. It needs a reader *and a
writer* for house forks if the level editor is in scope.**

---

# Part 5 — Mac Toolbox specifics and what a Go port must replace

## 5.1 Byte order, alignment and word sizes

Every multi-byte field in every resource in this fork is **big-endian**. There are no
exceptions and no little-endian sub-blocks. Types map as:

| Mac type | Bytes | Go |
| --- | ---: | --- |
| `char`, `SignedByte`, `Boolean` | 1 | `int8` / `uint8` / `bool` (non-zero = true) |
| `short`, `SInt16`, `Integer` | 2 | `int16` via `binary.BigEndian.Uint16` |
| `long`, `SInt32`, `LongInt` | 4 | `int32` |
| `Fixed` | 4 | 16.16 fixed point — `sampleRate 0x56EE8BA3` = 22254.5454… |
| `OSType`, `ResType` | 4 | `[4]byte`, MacRoman, space-padded (`'snd '`) |
| `Point` | 4 | `struct { V, H int16 }` — **vertical first** |
| `Rect` | 8 | `struct { Top, Left, Bottom, Right int16 }` — **top, left, bottom, right** |
| `RGBColor` | 6 | `struct { R, G, B uint16 }` |
| `ColorSpec` | 8 | `struct { Value int16; RGB RGBColor }` |
| `Str255` / pstring | 1+n | length byte then MacRoman bytes, **no NUL** |

Rect and Point field order is the single most common porting mistake: `Rect` is
**(top, left, bottom, right)** and `Point` is **(v, h)**. Both orders are the opposite of the
`(x, y, w, h)` convention Go graphics libraries use. Verified in every rect in this document
(e.g. `'WIND'` 128 = `0, 0, 384, 512` is a 512-wide × 384-high window).

Resource *bodies* are byte-packed with **no C struct padding**, except where a format
explicitly aligns (DITL item data is padded to an even length; §3.3.3). Do not model resources
with Go structs and `binary.Read` unless every field is fixed-size — use explicit offset
arithmetic.

## 5.2 8-bit indexed colour

`kPreferredDepth` = **8** (`GliderPRO/Headers/Externs.h:15`). Every GWorld the game creates for
artwork is 8 bits deep; masks are created at depth 1. Consequences:

1. **All pixel data in `'PICT'`, `'cicn'`, `'icl8'`, `'ics8'`, `'crsr'` is palette indices, not
   RGB.** The palette is the standard Mac 8-bit system CLUT, reproduced byte-for-byte as
   `'clut'` 128 = `'clut'` 129 (§3.10.1). A Go port should embed that 256-entry table as a
   `[256]color.RGBA` and use it as the default for every 8-bit source.
2. `'cicn'` and `'crsr'` carry **their own** ColorTables with **arbitrary depths** (1, 2, 4 or 8
   bpp) and **sparse `value` fields** — §3.4.1 measured `'cicn'` ColorTables from 2 to 44
   entries at 1, 4 and 8 bpp, and §3.5.2 measured `'crsr'` at 2 and 4 bpp with 4 and 5 entries.
   In those resources `value` is the *palette index the entry maps to*, and it is **not
   necessarily equal to the array position**. A port must build `pixel → value → RGB` in two
   steps, not one.
3. The named index constant the code uses is `kRedOrangeColor8` = **23**
   (`GliderPRO/Headers/GliderDefines.h:542`, with the developer's own note `// actually, 18`).
   Index 23 in the system CLUT is `FFFF 6666 0000` = RGB(255, 102, 0). It is passed to
   `FrameDialogItemC(theDialog, item, kRedOrangeColor8)` at 10+ sites including
   `GliderPRO/Sources/HouseInfo.c:119`, `GliderPRO/Sources/House.c:622`,
   `GliderPRO/Sources/ObjectInfo.c:127`, `:227`, `:236`, `:247`, `:248`, `:257`, `:258`, `:267`.
   **Port it as the literal RGB(255, 102, 0), not as index 23** — a Go port has no indexed
   framebuffer.
4. `SetEntries(0, 255, colorSpecs)` (`GliderPRO/Sources/MainWindow.c:496`, `:586`, `:591`)
   rewrites the *hardware* CLUT to fade the whole screen. In Go this becomes a per-pixel colour
   transform (or a shader) applied at composite time. The luminance formula and the
   index-5 exception are given in §3.10.1.
5. Because the CLUT is global, the original could recolour everything on screen for free.
   A Go port pays for it. If the colour→grey fade is in scope, do it as a palette
   transformation on the *source* images once, not per frame.

## 5.3 QuickDraw

| Toolbox call | Used for | Go replacement |
| --- | --- | --- |
| `DrawPicture(pic, &rect)` | rendering `'PICT'` (18 `GetPicture` sites) | decode PICT opcodes offline into PNG/raw; blit or scale |
| `CopyBits(src, dst, &s, &d, srcCopy, nil)` | most sprite blits | `draw.Draw` with `draw.Src` |
| `CopyMask(src, mask, dst, &s, &m, &d)` | 1-bit-masked sprite blits | `draw.DrawMask` with an `image.Alpha` built from the 1-bit mask |
| `PlotCIcon(&rect, cicn)` | `'cicn'` (via `DrawCIcon`) | decode `'cicn'` to RGBA+alpha once, then `draw.DrawMask` |
| `PlotIconSuite(&rect, atNone, ttNone, suite)` | `'ICON'`/`'ICN#'` families | same |
| `GetIndPattern` + `PenPat` | the 7-frame marquee (§3.10.2) | tile an 8×8 1-bit stipple anchored to the port origin |
| `FrameRect`, `PaintRect`, `EraseRect`, `InvertRect` | dialog decoration | trivial |
| `SetPort`, `GetPort`, `ClipRect`, `RectRgn`, visRgn | drawing state | a `draw.Image` + clip rect |
| `SetGWorld` / `GetGWorld` / `NewGWorld` / `LockPixels` | offscreen buffers | `image.RGBA` / `image.Paletted` |

Key semantics to preserve:

* `srcCopy` is a straight copy of *palette indices*, not an alpha blend. Copying index 0
  (white) copies white; there is no transparency without a mask.
* `CopyMask`'s mask is a 1-bit BitMap where **1 = copy the source pixel** and 0 = leave the
  destination. All the `'PICT'` mask resources in the 4998-5018 range and the 3900-3927 object
  masks follow this convention (§3.1).
* `DrawPicture(pic, dst)` maps `picFrame` onto `dst`, **scaling if the sizes differ.** This is
  how the game upscales its dialog art and background previews, and it is why the
  PICT 4005 (80 × 269) / mask 5005 (80 × 268) one-row size mismatch (§3.1) does not crash — it
  silently rescales the mask by 269/268. A Go port that assumes 1:1 blits will be off by a row
  on that pair.
* `rowBytes` in an on-disk PixMap has **bit 15 set** to mark it as a PixMap rather than a
  BitMap. Always mask with `& 0x3FFF` before using it as a stride (verified for `'cicn'`
  §3.4.1 and `'crsr'` §3.5.2: `0x8008` → 8, `0x8004` → 4).

## 5.4 Resource Manager

There is no Go equivalent; the port must decide on an asset model. Behaviours the code relies
on:

| Behaviour | Where it matters | Go approach |
| --- | --- | --- |
| lookup by `(type, id)` | everywhere | a `map[[4]byte]map[int16][]byte`, or generated Go constants + embedded files |
| a **search chain** (house shadows app) | custom backgrounds at IDs ≥ 3000 | an ordered slice of asset sources, first hit wins |
| "current file only" variants (`Get1…`, `Count1…`) | `House.c:250`, `RoomInfo.c:885` | an explicit "this house only" query |
| resource *map order* for `Get1IndResource` | `GetFirstPICT` (`RoomInfo.c:885`) | preserve the on-disk order in a slice, don't sort |
| `purgeable` attribute | 73 resources (§2.3) | ignore — it is a memory hint |
| `SetResLoad(false)` + `GetMaxResourceSize` | `SoundBytesNeeded`, `MusicBytesNeeded` | ignore — Go has no fixed heap |
| `DetachResource` | `AnimCursor.c:63` | ignore — assets are owned values |
| `ReleaseResource` / `DisposeHandle` distinction | `AnimCursor.c:56` vs `:63` | ignore |
| `ResError()` propagated into `YellowAlert(…, ResError())` | `HouseIO.c:574` | keep a Go error, render the number as 0 or a mapped code |

**Nothing is looked up by name**, so the 215 names (§2.4) can be dropped from the asset table.

## 5.5 Sound Manager

`'snd '` format 1 with one `bufferCmd` and a standard `SoundHeader` (§3.2.1). All 70 sounds are
8-bit unsigned **offset-binary** PCM (silence = 0x80) at exactly 22254.5454… Hz
(`0x56EE8BA3` as 16.16 Fixed = the classic Mac `rate22khz`). Conversion to signed 16-bit:

```
s16 = (int16(u8) - 128) << 8
```

Loop points are in the header (§3.2.2): 59 effects have the degenerate
`(length-2, length-1)`, four sustained noises have `(0, length-1)` — 1018 `kThrustSound`,
1026 `kShredSound`, 1047 `kSizzleSound`, 1062 `kHissSound` — and all 7 music tracks have
`(0, length)`.

| Toolbox | Go |
| --- | --- |
| `SndNewChannel(&chan, sampledSynth, initStereo, callback)` | one mixer voice |
| `SndDoCommand(chan, {bufferCmd, 0, sndPtr}, false)` | enqueue a buffer on that voice |
| `SndDoCommand(chan, {callBackCmd, …})` | a done-channel or callback per voice |
| `SndDisposeChannel(chan, true)` | stop and free the voice |
| priority-based channel stealing (`GliderPRO/Headers/GliderDefines.h:120-180`, 61 constants) | keep the same integer priorities and the same steal rule |

Resampling to 44100 or 48000 Hz is required on any modern output device; 22254.5454 is not a
rational multiple of either, so a port needs real resampling (linear interpolation is
sufficient at this fidelity, but a fixed-point phase accumulator matching
`0x56EE8BA3 / outputRate` keeps drift under one sample over a whole music track).

## 5.6 Dialog Manager, Menu Manager, Control Manager

The port must re-implement, not translate:

* **`'DLOG'`/`'DITL'`** define pixel-exact layouts in **global screen coordinates** with no
  centring, because every positioning helper in `DialogUtils.c` is commented out
  (§3.3, `GliderPRO/Sources/DialogUtils.c:39-328`). Reproducing the original look means
  honouring the literal `boundsRect` values, then choosing your own centring policy.
* **`userItem`s (74 of the 428 DITL items)** are drawn entirely by application code through
  `SetDialogItem` draw procs; the DITL only supplies the rectangle. Their appearance is *not*
  in the resource fork.
* **`ParamText(^0..^3)`** substitution is a Dialog-Manager service applied to `statText` items
  at draw time. 30+ DITL items contain `^0`-`^3`. Implement it as a per-dialog 4-string
  substitution table.
* **Menus** need the Apple-logo glyph (MacRoman `0x14`, `'MENU'` 128's title) replaced, and
  `AppendResMenu(appleMenu, 'DRVR')` (`GliderPRO/Sources/InterfaceInit.c:52`) deleted.
* **The two popup `'CNTL'`s** depend on a third-party CDEF (§3.11.2) and on the
  value-can-exceed-`max` quirk (§3.8.1).
* **The four floating windoids** depend on a third-party WDEF whose pixels are unrecoverable
  from this source drop (§3.11.1).

## 5.7 Text encoding

Every string in this fork is **MacRoman**, not UTF-8 and not Latin-1. The bytes that actually
occur and matter:

| Byte | MacRoman | Unicode | Where |
| ---: | --- | --- | --- |
| `0x14` | Apple logo | *(no Unicode codepoint)* | `'MENU'` 128 title |
| `0xA9` | `©` | U+00A9 | `'ozm5'` 0, `'vers'` 1/2, DITL 150 item 6 |
| `0xAA` | `™` | U+2122 | `'vers'` 1, `'STR#'` 171, `'STR#'` 1006, many DITLs |
| `0xC9` | `…` | U+2026 | menu items, `'STR#'` 150 indices 25, 26, 33 |
| `0xD0` | `–` en dash | U+2013 | DITL text |
| `0xD2`/`0xD3` | `“`/`”` | U+201C / U+201D | DITL text |
| `0xD4`/`0xD5` | `‘`/`’` | U+2018 / U+2019 | DITL text |
| `0x0D` | CR | line break *inside* a string | `'vers'` 1 long string, DITL 1014 item 9, DITL 1017 item 6, DITL 1020 item 4 |

Note the last row: `0x0D` inside a `statText` or `'vers'` string is a **line break the renderer
must honour**, not a terminator. Go should convert MacRoman → UTF-8 at asset-extraction time
and translate embedded `0x0D` to `\n`. The Apple logo has no Unicode equivalent and needs a
bitmap or a substitute glyph.

`GetString(kChooserStringID /* -16096 */)` at `GliderPRO/Sources/StringUtils.c:308` reads the
**System file's** owner name (see §3.6.11) — substitute the host user name.

## 5.8 68k / PowerPC artefacts

* `Handle` (a `**void` with a movable master pointer), `HLock`/`HUnlock`/`HGetState`/
  `HSetState` — all disappear in Go. Anywhere the C locks a handle before taking an interior
  pointer, Go just uses a slice.
* `BlockMove(src, dst, n)` → `copy(dst, src)`. Note `StructuresInit2.c:295` copies exactly
  `kDemoLength` = 6702 bytes without checking the resource size — a Go port should bounds-check.
* Pascal calling convention (`pascal` keyword) on WDEF/CDEF/dialog filters and
  `NewModalFilterUPP` / UPP glue exist only to satisfy the 68k↔PPC transition. Delete.
* `char` is **signed** on classic Mac compilers. `demoType.key` and `demoType.padding` are
  `char`; the observed `padding` values run to 255, i.e. they are read as negative numbers by
  the C. Nothing depends on it here, but any `char` field carrying values ≥ 128 must be modelled
  as `int8` if it is ever compared or arithmetic'd.
* The `sizeof(demoType) == 6` result relies on no trailing padding after two `char`s following
  a `long`. Verified against the data (`6702 / 6 = 1117` exactly).

---

## Open questions

1. **`'DLOG'` 150 has three extra bytes.** All 28 `'DLOG'`s are 21 bytes except 150, which is
   **24** bytes, with `00 02 2F` after the zero-length title (§3.3.1). `0x022F` = 559 is a
   plausible `'dctb'` ID or a positioning hint, and the trailing `0x2F` is unexplained. The
   Dialog Manager reads only the first 21 bytes plus the title, so these bytes are inert — but
   what wrote them is unknown.

2. **`'DLGX'` 150's per-item record layout.** The resource is 190 bytes: a pstring `Charcoal`
   (the font name) then `0x000C` = 12 and what looks like a 12-byte-stride table, all zeros in
   the region examined. `'DLGX'` is the extended-dialog resource (font/style per item);
   nothing in `Sources/` mentions it and no `'ictb'`-style code path uses it, so it is
   presumed inert. Its exact field layout has not been derived from the bytes here.

3. **The music track sequence.** `'snd '` 2000-2006 are named `Refrain1.22`, `Refrain2.22`,
   `Refrain3.22`, `Refrain4.22`, `Chorus.22`, `RefrainSparse1.22`, `RefrainSparse2.22`, and
   2004 (`Chorus.22`) is about twice as long as the others (194,070 bytes vs 96,930-97,376;
   `2 x 97,076 = 194,152`, so it is 82 bytes short of exactly double). The
   *order* in which `Music.c` sequences them (and whether "Sparse" variants are chosen by game
   state) is a `Music.c` question, not a resource-fork question, and is not answered here.

4. **DITL 1016 says "(3000 - 3499)", the constants imply 3000-3299, and the validation allows
   3000-3799.** DITL 1016 item 6 is the
   literal help text `(3000 - 3499)`, and item 5's default `editText` is `3000`. The constants
   are `kUserBackground` = 3000 and `kUserStructureRange` = 3300
   (`GliderPRO/Headers/GliderDefines.h:522-523`), and `GliderPRO/Sources/RoomInfo.c:762` validates
   `(longID >= 3000) && (longID < 3800)`. **Three different upper bounds: 3499 (UI text), 3300
   (constant), 3800 (validation).** Which one the intended contract is, is unresolved; a port
   should accept 3000-3799 to match the code that actually runs.

5. **The About box's copyright disagrees with `'vers'`.** DITL 150 item 6 is
   `© 1994-2000 Casady & Greene, Inc.` while `'vers'` 1 and 2 both say
   `© 1994-95 Casady & Greene, Inc.` (§3.9.4). The DITL was updated for a later release and the
   `'vers'` resource was not, or vice versa.

6. **The fork claims version 1.1.2, the source release is called 1.0.4.** Both `'vers'` 1 and
   `'vers'` 2 encode BCD `01 12`, stage `0x80` (final), short string `1.1.2` (§3.9.4). Either
   the `.r` was dumped from a 1.1.2 build and shipped with 1.0.4 sources, or "1.0.4" refers to
   something other than the product version. **Anything that displays a version number should
   be treated as unverified.**

7. **`'PICT'` 10000 (4,182 bytes).** Isolated at the very top of the ID space with no
   `#define` referring to 10000 anywhere in `Headers/` or `Sources/`. Presumed a placeholder or
   test image. Its content has not been rendered.

8. **`'PICT'` 5006 and 5010 have non-zero `picFrame` origins** — `(0,91,120,115)` and
   `(195,0,230,40)`. Every other PICT in the fork is origin-(0,0). Because `LoadGraphic`
   normalises the origin away (§4.6) and `LoadScaledGraphic` ignores it, the offsets appear to be
   authoring residue rather than intent — but if some code path draws them with
   `DrawPicture(pic, &(*pic)->picFrame)` unnormalised, the offsets matter.

9. **`'CNTL'` 132, `'mctb'` 132 and `'cctb'` 132** are a matched set of leftovers from a removed
   Edit menu / scrolling list (§3.7.6, §3.8.1). What the feature was is not recoverable.

10. **`'STR#'` 1007 index 144 is `Mermaid`,** one past the highest object type
    `kChimes` = 0x8F = 143, and `kNumSrcRects` = 144 (§3.6.10). There is no object 144. Was a
    Mermaid object cut, or is the list simply one entry long?

11. **What produced the `'Date'` resource type?** It is a `'PICT'` under a different four-char
    code (§4.2) and is only ever a fallback. Whether some house-authoring tool wrote `'Date'`
    instead of `'PICT'`, or whether `'Date'` is a deliberate "don't let the Finder preview this"
    trick, is unknown. Houses in `GliderPRO/Houses/*.binhex` would settle it.

12. **`'WDEF'` 128/129 and `'CDEF'` 128 pixel appearance.** 17,906 bytes of 68k object code with
    no source. The windoid title-bar and popup-control appearance cannot be reconstructed from
    this repository; only the content-rect sizes the code requests are known.

13. **`'ICON'` 900, 910, 1060, 1070-1073** exist as both `'ICON'` and `'cicn'` but their
    referring DITLs (1031, 1032, 1029, 1030, 1036, 1041, 1046) are alerts whose text was
    transcribed only in truncated form in the DeRez comments. Their exact full text was read
    from the DITL data (§3.3.4) but the *situations* that raise 1046 (`So What?` /
    `If you resume a saved game, you…`) are not fully traced.

14. **`'acur'` 128's frame order is descending (160→149).** Whether the beach-ball animation is
    genuinely meant to run backwards through the `'CURS'` IDs, or whether the IDs were assigned
    in reverse when the art was made, cannot be determined without rendering the 12 cursors.

## Porting notes

**Extraction strategy**

1. Run `python3 tools/probe_rez.py extract <outdir>` once. It writes
   `<outdir>/<TYPE>/<ID>.bin` for all 538 resources, preserving exact bytes; `'snd '` becomes
   the directory `snd_` (the space is replaced with `_`), and the 35 directories are otherwise
   the literal four-character type codes including `ICN#`, `ics#`, `PAT#` and `STR#`.
   Verified end-to-end: `du -sb` on the output tree reports **3,168,309** bytes, exactly the
   inventory total in §2.1. Then convert each type to a modern format *offline*, in a build
   step, and commit the results. Do **not** ship a Rez parser or a PICT decoder in the game
   binary.
2. `'PICT'` is the only genuinely hard format (116 v2 + 36 v1, §3.1). Decode it once with a
   trusted decoder and check in PNGs. If no decoder is available offline, the fallback is to
   implement only the opcodes these 152 pictures actually use — but that set has not been
   enumerated in this document, so budget for it.
3. `'snd '`, `'cicn'`, `'crsr'`, `'CURS'`, `'STR#'`, `'PAT#'`, `'clut'`, `'demo'`, `'DITL'`,
   `'DLOG'`, `'ALRT'`, `'MENU'`, `'CNTL'`, `'WIND'`, `'acur'` are all trivially parseable with
   the layouts in Part 3; every one has been verified byte-exact against the data.
4. Generate Go constants from the `#define`s rather than retyping IDs. The ID↔meaning mappings
   in §3.1.1, §3.2.2, §3.3, §3.4.3, §3.5, §3.6, §3.7 are the authority.

**Fidelity risks, in rough order of how likely they are to bite**

1. **`Rect` is (top, left, bottom, right) and `Point` is (v, h).** Every rect in this document
   is in that order. Getting it wrong produces plausible-looking but wrong geometry everywhere.
2. **`rowBytes & 0x3FFF`.** Forgetting the mask gives strides of 32,776 instead of 8.
3. **`'cicn'`/`'crsr'` ColorTable `value` fields are not array indices.** Two-step lookup is
   mandatory (§5.2).
4. **The four different sprite-mask ID conventions** (§3.1): `art + 1000` for 3998-5018 with
   **4003 having no mask**, `art + 1` for 1019→1020, `art − 1` for 1990→1989 and 1992→1991,
   `art + 4` for 1994→1998, plus 8 hand-numbered object masks in 3900-3927. A single formula
   will silently mis-mask.
5. **PICT 4005 is 80 × 269 but mask 5005 is 80 × 268.** QuickDraw rescales; a 1:1 blit will not.
6. **`'snd '` slot 63 (`'snd '` 1063) does not exist** and must stay dynamic (§4.4). Eagerly
   loading 64 sounds fails.
7. **`'snd '` loop points**: only four effects loop (1018, 1026, 1047, 1062). Treating
   `(length-2, length-1)` as a real loop turns 59 one-shots into buzzing.
8. **Sample format is offset-binary unsigned 8-bit at 22254.5454 Hz.** Skipping the `-128` bias
   produces loud clipping; skipping resampling produces a pitch error of about +5.7% at 44100
   ÷ 2 assumption, or −6% the other way.
9. **`'DITL'` items are even-aligned.** The data length byte is the *logical* length; the reader
   must skip one extra pad byte when it is odd (§3.3.3). Off-by-one desynchronises the rest of
   the item list.
10. **`'DITL'` type byte bit 7 is the *disable* flag**, so mask with `& 0x7F` to get the type and
    test `& 0x80` for disabled. 217 of 428 items are disabled (§3.3.4).
11. **`'MENU'` 140 and 141 both carry in-resource `menuID` 133.** Key menus by resource ID
    (§3.7.5).
12. **`'MENU'` 129 item 1 is `"New Game\0"`** — a 9-byte length for an 8-character string.
    Trim the NUL (§3.7.3).
13. **`'CNTL'` 128's value legally exceeds its `max`** (19 > 18) to mean "Original Artwork"
    (§3.8.1).
14. **Dialogs are positioned at their literal global `boundsRect`** because every centring helper
    is commented out (§3.3, §5.6). Reproducing "the original layout" and "sensible on a modern
    display" are different goals; pick one deliberately.
15. **`'clut'` 128 = `'clut'` 129 = the standard Mac 8-bit CLUT.** Embed it. Every 8-bit index in
    the fork resolves against it. Index 5 is pure yellow and is deliberately exempt from the
    fade (§3.10.1).
16. **`kRedOrangeColor8` = 23 must become RGB(255, 102, 0)**, not the number 23 (§5.2).
17. **The demo's `key` 0/1 cases contradict their own comments.** Port the code (§3.12.4).
18. **`demoType.padding` is uninitialised garbage.** Never read it (§3.12.2).
19. **`'Date'` is an alias for `'PICT'` in house forks, and `'bnds'` bit weights are
    left = 1, top = 2, right = 4, bottom = 8** (§4.2). Missing `'bnds'` means 0 = all edges open.
20. **`Get1IndResource('PICT', 1)` returns the *first resource in map order*, not the lowest ID**
    (§4.3). This fork's own IDs are unsorted within every type run, so the distinction is real.
21. **MacRoman, and embedded `0x0D` inside strings means "line break"** (§5.7). Also: the Apple
    logo (`0x14`) has no Unicode codepoint.
22. **`'vers'` says 1.1.2, the About box says © 1994-2000, the release is called 1.0.4.** Do not
    invent a reconciliation; surface whichever you choose and note it.

**Things that can simply be deleted**

`'BNDL'`, `'FREF'` ×6, `'ozm5'`, `'ICN#'` ×6, `'icl4'` ×6, `'icl8'` ×6, `'ics#'` ×4, `'ics4'` ×4,
`'ics8'` ×4 (Finder icons); `'dctb'` ×20, `'wctb'`, `'cctb'`, `'ictb'`, `'DLGX'`, `'mctb'` ×5
(near-no-op colour tables); `'CNTL'` 132 (orphan); `'STR#'` 128 (dead); `'PICT'` 10000
(unreferenced); the `'SIZE'` machinery; `AppendResMenu('DRVR')`; and the whole `'ICON'` /
icon-suite path — verified: **all 35 `'ICON'` IDs also exist as `'cicn'`** (set difference
`ICON - cicn` is empty), so the 8-bit colour icon can always be used instead. That accounts for
**70** of the 538 resources — `'BNDL'` 1 + `'FREF'` 6 + `'ozm5'` 1 + `'ICN#'` 6 + `'icl4'` 6 +
`'icl8'` 6 + `'ics#'` 4 + `'ics4'` 4 + `'ics8'` 4 = 38, plus `'dctb'` 20 + `'wctb'` 1 +
`'cctb'` 1 + `'ictb'` 1 + `'DLGX'` 1 + `'mctb'` 5 = 29, plus `'CNTL'` 132, `'STR#'` 128 and
`'PICT'` 10000 = 3 — and none of the game's behaviour. (Dropping the 35 `'ICON'`s too would make
it 105.)

**What must be preserved bit-exactly**

The 152 `'PICT'`s, the 70 `'snd '`s, the 44 `'cicn'`s, the 280 `'STR#'` strings, the 54
`'DITL'`s' geometry and text, the 7 `'PAT#'` phases, the 256-entry `'clut'`, the 16 `'CURS'`
bitmaps + hotspots, the 12 `'crsr'`s, and the 1,117 `'demo'` records. That is 1,919,598 +
1,129,344 + 44,584 + 8,229 + 11,576 + 58 + 2,056 + 1,088 + 3,504 + 6,702 = **3,126,739 bytes**
of the fork's 3,168,309 — **98.7%**.
