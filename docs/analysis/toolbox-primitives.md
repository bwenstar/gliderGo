# Mac Toolbox primitives: normative resolutions for a Go port

Glider PRO 1.0.4 (John Calhoun, 1994). This document resolves the four Mac Toolbox
primitives that other documents in `docs/analysis/` flag as unresolvable and that block
bit-level parity:

| # | Primitive | Blocks | Status after this document |
|---|---|---|---|
| 1 | `Random()` / `qd.randSeed` | frame-exact replay of `demo` 128, all animation | **Premise disproved** + normative generator specified |
| 2 | `srcXor` on an 8-bit indexed pixmap | marquee blink, glider stencil | **Resolved from first principles** |
| 3 | The 16-entry system 4-bit grayscale CLUT | every `thisMac.isDepth == 4` branch | **Proved**, not inferred |
| 4 | Chicago 12 pt, Geneva 9/12/14 pt metrics | scoreboard, banner, high scores, all dialogs | **Substitute specified + validated** on 270 measurements |

None of the four is recoverable as a literal 1994 artefact from the source release. For each,
this document states (a) exactly what *is* recoverable and how it was verified, (b) a single
**normative ruling** the Go port should implement, and (c) the enumerated, measured
consequences of that ruling.

---

## 0. Scope, method, notation

### 0.1 What this document decides

Four rulings, referenced by tag throughout the rest of `docs/analysis/`:

| Tag | Ruling |
|---|---|
| **R-RNG-1** | `Random()` is the Lehmer/Park-Miller minimal-standard generator on `qd.randSeed`, computed with Schrage decomposition, returning the low 16 bits of the new state reinterpreted as `int16`. Initial seed **1**. |
| **R-RNG-2** | The shipped `demo` 128 replay is **RNG-independent**. R-RNG-1 is required only for pixel-exact *animation*, never for trajectory. |
| **R-XOR-1** | `srcXor` at `Marquee.c:438` is `dest[i] ^= 0xFF` where the 1-bit source pixel is set, and `dest[i] ^= 0x00` where it is clear. |
| **R-CLUT4-1** | The depth-4 device CLUT is `entry[i].rgb = ((15-i)*0x1111, (15-i)*0x1111, (15-i)*0x1111)` for `i` in 0..15. |
| **R-CLUT4-2** | Render everything at 8 bits into the standard 256-entry palette; derive the depth-4 appearance by post-mapping, and honour the 33 authored `isDepth == 4` branches as *literal index substitutions*, not as a derivation. |
| **R-FONT-1** | Chicago 12 -> Liberation Sans Bold @ 13 px; Geneva 9 -> Liberation Sans Regular @ 9 px; Geneva 12 -> Liberation Sans Regular @ 11 px; Geneva 14 -> Liberation Sans Regular @ 13 px. Bold is QuickDraw-synthesised (+1 px advance/char). |

### 0.2 Evidence base

* Source: `GliderPRO/Sources/*.c` (67 files), `GliderPRO/Headers/*.h`, `GliderPRO/Prefix.h`.
* Resources: `GliderPRO/Glider PRO.r` (15,475,666 bytes as a Rez text dump, 538 resource records).
* Shared library: `GliderPRO/CarbonLib` (234,308 bytes, four PEF containers).
* Houses: the nine shipped BinHex `.hqx` house files, parsed with
  `tools/probe_house.py`.

**Line numbers** in citations are from a CR->LF converted copy of each file. Classic Mac
sources are CR-only; convert with `tr '\r' '\n'` before reading, or every file looks like one
line. `grep` must be given `-a` on these files: Mac Roman high bytes (`\xC9` for the ellipsis,
`\xA5` for the bullet) make GNU grep classify them as binary and silently drop matches.

Every claim in this document about binary layout was produced by parsing the actual bytes with
`python3`; the observed values are reproduced inline.

### 0.3 Notation

* `i8` = an index into the 256-entry 8-bit palette (`clut` 128).
* `i4` = an index into the 16-entry 4-bit grayscale device CLUT.
* 16-bit RGB channel values are as QuickDraw stores them (`RGBColor` is three `unsigned short`).
  An 8-bit-per-channel value `0xCC` appears as `0xCCCC` (byte-doubled), never `0xCC00`.
* Original identifiers are given in parentheses after the prose name, e.g. "the raw draw
  (`rawResult`)".

### 0.4 The four rulings in one paragraph each

**Random.** `RandomInt` is the only live consumer of `Random()`, and its 32 call sites are all
in animation, editor or launch-time code. `Player.c` contains no RNG call at all. The shipped
demo therefore replays deterministically *without* a correct `Random()`. The generator still
matters for pixel-exact candle flames, page flutter and telephone chimes, so R-RNG-1 pins it.

**srcXor.** There is exactly one `srcXor` in the whole tree. Its source is a **1-bit** GWorld,
so QuickDraw colourises it to foreground/background before the boolean op. Foreground is
invariantly black (index 255) and background white (index 0) at that call site, so the XOR
value is `0xFF` for set bits and `0x00` for clear bits. `i ^ 0xFF == 255 - i` on this palette,
and the op is its own inverse, which is exactly the property the marquee relies on.

**4-bit CLUT.** Proved rather than assumed: the standard 8-bit palette contains exactly one
index for each of the sixteen `(15-i)*0x11` gray levels, and 11 of the 13 pure-gray authored
`(i4, i8)` pairs are *exact identity hits* under that mapping. The two exceptions are
independently explainable as authoring slips.

**Fonts.** No `FOND`/`NFNT`/`FONT`/`sfnt` resource exists anywhere in the 538 records, so glyph
bitmaps are unrecoverable in principle. Every vertical position in the game is a hard-coded
baseline constant, and only one code path (`TETextBox`) consults font ascent, so only horizontal
advances matter. Substitutes were chosen to satisfy **270** measured layout constraints - 230
single-line `DITL` items (§4.11) plus 40 hard-coded game sites (§4.13) - and **all 230 DITL items
fit their raw boxes**, with only one game-site row (`SEPTEMBER`, 1 px) attributable to the
substitute rather than to an original overflow.

---

# Part 1 - `Random()` and determinism

## 1.1 The complete RNG call graph

`Random()` is a QuickDraw trap. Its declaration is in `QuickDraw.h`
(`pascal short Random(void)`), it reads and writes the global `qd.randSeed` (a `long` inside
the QuickDraw globals), and it returns a `short`.

Every reference to `Random()` in the tree:

| # | Site | Wrapper | Live? |
|---|---|---|---|
| 1 | `GliderPRO/Sources/Utilities.c:76` | `RandomInt` | **yes** |
| 2 | `GliderPRO/Sources/Utilities.c:96` | `RandomLong` (high word) | no - `RandomLong` has zero callers |
| 3 | `GliderPRO/Sources/Utilities.c:101` | `RandomLong` (low word) | no |
| 4 | `GliderPRO/Sources/Main.c:232` | `thePrefs.fakeLong = Random();` | no - inside `#ifndef COMPILENOCP` |

Site 4 is compiled out because `COMPILENOCP` **is** defined:

```
GliderPRO/Headers/GliderDefines.h:11    //#define CREATEDEMODATA
GliderPRO/Headers/GliderDefines.h:12    //#define COMPILEDEMO
GliderPRO/Headers/GliderDefines.h:13    //#define CAREFULDEBUG
GliderPRO/Headers/GliderDefines.h:14    #define COMPILENOCP
GliderPRO/Headers/GliderDefines.h:15    #define COMPILEQT
GliderPRO/Headers/GliderDefines.h:16    #define BUILD_ARCADE_VERSION    1
```

`RandomLong` (`Utilities.c:88-109`) was grepped for callers across all 67 `.c` files and all
headers: the only occurrences are the definition and the prototype in `Utilities.h`. Same for
`InitRandomLongQUS` (`Utilities.c:115-118`) and `RandomLongQUS` (`Utilities.c:124-128`).

**Consequence: `RandomInt` is the sole live entry point to the RNG, and it consumes exactly one
`Random()` per call.**

## 1.2 `RandomInt` verbatim

```c
// GliderPRO/Sources/Utilities.c:69-82
//--------------------------------------------------------------  RandomInt
// Returns a random integer (short) within "range".
short RandomInt (short range)
{
	register long	rawResult;

	rawResult = Random();
	if (rawResult < 0L)
		rawResult *= -1L;
	rawResult = (rawResult * (long)range) / 32768L;

	return ((short)rawResult);
}
```

Numbered restatement (original names in parentheses):

1. Draw one 16-bit signed value from the Toolbox (`rawResult = Random()`); the `short` return is
   sign-extended into a `long`, so the domain is [-32768, 32767].
2. If negative, negate in place (`rawResult *= -1L`). Domain becomes [0, 32768] - note the
   **asymmetry**: 32768 is reachable (from -32768), 32767 is reachable, so 32769 distinct values
   map onto the folded range.
3. Multiply by the requested range (`rawResult * (long)range`) in 32-bit signed arithmetic, then
   integer-divide by 32768 (`/ 32768L`). C division truncates toward zero; both operands are
   non-negative here, so this is a floor.
4. Truncate to `short` and return.

The multiply overflows `long` when `|rawResult| * range > 2^31 - 1`, i.e. when
`range > 65535` (since `rawResult <= 32768`). The largest range in the tree is
`kRingSpread` = 25000 (`GliderPRO/Sources/Play.c:21`), so no overflow occurs in practice.

**The off-by-one.** Step 3 yields exactly `range` when `rawResult == 32768`, which happens iff
the raw word is `0x8000`. So `RandomInt(range)` returns a value in **[0, range] inclusive**, not
`[0, range)`. This is 1 outcome in 65536. A Go port that writes `rng.Intn(range)` is
*not* equivalent: it can never return `range`. See §1.7 for the exact distributions and §1.10
for the two call sites where the extra value is an out-of-bounds array index.

## 1.3 Seeding

```c
// GliderPRO/Sources/Utilities.c:43-67
void ToolBoxInit (void)
{
#if !TARGET_CARBON
	InitGraf(&qd.thePort);
	InitFonts();
	FlushEvents(everyEvent, 0);
	InitWindows();
	InitMenus();
	TEInit();
	InitDialogs(nil);

	MaxApplZone();

	MoreMasters();
	MoreMasters();
	MoreMasters();
	MoreMasters();

	GetDateTime((UInt32 *)&qd.randSeed);

#endif

	InitCursor();
	switchedOut = false;
}
```

`GliderPRO/Prefix.h` (7 lines, the whole file):

```c
#define TARGET_CARBON               1
#define ACCESSOR_CALLS_ARE_FUNCTIONS 1
#define OPAQUE_TOOLBOX_STRUCTS      1
#define OPAQUE_UPP_TYPES            1
#define forCarbon                   1
#define BUILDING_RUN_LINKED_IN      0
#define DEBUG                       1
```

`TARGET_CARBON` is 1, therefore **`GetDateTime((UInt32 *)&qd.randSeed)` is compiled out of the
shipped Carbon target.** Classic `InitGraf` initialises `randSeed` to **1**; Carbon has no
`InitGraf` and CarbonLib's QuickDraw initialisation also leaves it at 1. So the shipped 1.0.4
Carbon build starts every launch from the *same* seed and produces an identical animation
stream every time.

The project file `GliderPRO/Glider PRO CW5.mcp` contains both a Carbon and a non-Carbon target,
so a 68k/classic-PPC build would have been date-seeded. **This document normatively adopts the
Carbon behaviour** (fixed seed 1), because that is what `Prefix.h` selects and because it is the
only choice that makes the game reproducible. See §1.15 for what changes if you pick the other.

`GetDateTime` returns seconds since 1904-01-01 00:00:00 local time as an unsigned 32-bit value.
For every date from 1972-02-11 onwards that value exceeds `0x7FFFFFFF`, so reinterpreted as the
`SInt32` that `qd.randSeed` actually is, it is **negative** for the entire 1994-2040 window.
A Lehmer generator requires a state in [1, m-1]; the exact behaviour of Apple's `Random()` on a
negative seed is not recoverable from this source release, which is a second, independent reason
the date-seeded path cannot be specified exactly.

## 1.4 What `CarbonLib` does and does not contain

`GliderPRO/CarbonLib` is 234,308 bytes and holds four concatenated PEF containers. Parsed
layout of a PEF container (all big-endian):

| Offset | Size | Field | Observed |
|---|---|---|---|
| 0 | 4 | `tag1` | `'Joy!'` |
| 4 | 4 | `tag2` | `'peff'` |
| 8 | 4 | `architecture` | `'pwpc'` |
| 12 | 4 | `formatVersion` | 1 |
| 16 | 4 | `dateTimeStamp` | - |
| 20 | 4 | `oldDefVersion` | - |
| 24 | 4 | `oldImpVersion` | - |
| 28 | 4 | `currentVersion` | - |
| **32** | 2 | `sectionCount` | 1 in all four containers |
| 34 | 2 | `instSectionCount` | 0 |
| 36 | 4 | `reservedA` | 0 |
| 40 | 28*n | section headers | - |

Section-header `sectionKind` values: 0 = code, 1 = unpackedData, 2 = patternInitData,
3 = constant, **4 = loader**, 5 = debug, 6 = executableData, 7 = exception, 8 = traceback.

**All four containers have `sectionCount` 1 and that single section is kind 4 (loader).
There is not one byte of PPC code in the file.** These are pure stub import libraries.

The loader section begins with a `PEFLoaderInfoHeader` of 14 fields = **56 bytes**. The export
area that follows is: a hash table of `2^exportHashTablePower` 4-byte slots, then
`exportedSymbolCount` 4-byte key entries (the symbol's name length is `keyword >> 16`), then
`exportedSymbolCount` 10-byte entries laid out `>IIh` = (classAndName, symbolValue,
sectionIndex).

Container 0 **does** export the symbol `Random` (and `randomx`), with symbolClass 2 (data/TVector
distinction irrelevant for a stub), `symbolValue` 0, `sectionIndex` -2 (= "imported/absolute"
sentinel used by stub libraries). So the file proves Apple's `Random` exists and is exported by
CarbonLib; it contains no implementation.

**Conclusion: the exact 1994 algorithm cannot be recovered offline. A normative choice is
mandatory.**

## 1.5 R-RNG-1 - the normative generator

Implement `Random()` as the Lehmer / Park-Miller *minimal standard* generator:

```
state (qd.randSeed) : int32, invariant 1 <= state <= 2147483646
multiplier          a = 16807       = 0x41A7   = 7^5
modulus             m = 2147483647  = 0x7FFFFFFF = 2^31 - 1 (prime)
Schrage constants   q = m / a = 127773
                    r = m % a = 2836
```

```
1. hi   := state / q                     // integer division
2. lo   := state % q
3. test := a*lo - r*hi                   // fits in int32 for all legal state
4. if test > 0 then state := test
                else state := test + m
5. word := uint16(state & 0xFFFF)
6. return int16(word)                    // reinterpret, do NOT clamp
```

Properties, verified by exhaustive/long-run computation:

* Period = **2147483646** (`m` is prime and 16807 is a primitive root mod `m`).
* The two degenerate states are 0 and `m = 2147483647`, but neither maps to 0 under the Schrage
  form above. Computed directly: `schrage(0) = 2147483647` (`hi = lo = test = 0`, and because the
  test is not *strictly* positive the code adds `m`), and `schrage(2147483647) = 2147483647`
  (`hi = 16807`, `lo = 2836`, `test = 16807*2836 - 2836*16807 = 0`, again `+m`). So **`m` is the
  single absorbing state**, and 0 falls into it in one step. Note this is where Schrage and the
  naive `(state*a) % m` disagree: the latter sends both 0 and `m` to 0. A port must never let the
  seed reach either value; the invariant `1 <= state <= 2147483646` is what keeps them unreachable.
* Step 5-6 must reinterpret, not saturate: half of all draws are negative, and `RandomInt`
  depends on that (§1.2 step 2).

Justification for choosing this generator over alternatives:

1. It is the generator Apple's own `Random()` is documented (in *Inside Macintosh: Imaging With
   QuickDraw*, and reproduced in the 1990s Mac developer literature) to implement, and the only
   one for which the "returns the low word of a 31-bit Lehmer state" behaviour is standard.
2. It is deterministic, portable, and has no platform-dependent overflow behaviour when
   implemented with Schrage decomposition.
3. Because of R-RNG-2 (§1.12) the choice cannot alter gameplay, only cosmetic animation, so the
   cost of being wrong is bounded and measurable.

## 1.6 Verified seed-1 stream

Computed with the algorithm of §1.5 starting from `state = 1`:

| draw n | state after (`qd.randSeed`) | hex | low 16 bits | `Random()` as int16 | `RandomInt(2)` | `RandomInt(3)` | `RandomInt(6)` |
|---|---|---|---|---|---|---|---|
| 1 | 16807 | 0x000041A7 | 0x41A7 | 16807 | 1 | 1 | 3 |
| 2 | 282475249 | 0x10D63AF1 | 0x3AF1 | 15089 | 0 | 1 | 2 |
| 3 | 1622650073 | 0x60B7ACD9 | 0xACD9 | -21287 | 1 | 1 | 3 |
| 4 | 984943658 | 0x3AB50C2A | 0x0C2A | 3114 | 0 | 0 | 0 |
| 5 | 1144108930 | 0x4431B782 | 0xB782 | -18558 | 1 | 1 | 3 |
| 6 | 470211272 | 0x1C06DAC8 | 0xDAC8 | -9528 | 0 | 0 | 1 |
| 7 | 101027544 | 0x06058ED8 | 0x8ED8 | -28968 | 1 | 2 | 5 |
| 8 | 1457850878 | 0x56E509FE | 0x09FE | 2558 | 0 | 0 | 0 |
| 9 | 1458777923 | 0x56F32F43 | 0x2F43 | 12099 | 0 | 1 | 2 |
| 10 | 2007237709 | 0x77A4044D | 0x044D | 1101 | 0 | 0 | 0 |
| 11 | 823564440 | 0x31169898 | 0x9898 | -26472 | 1 | 2 | 4 |
| 12 | 1115438165 | 0x427C3C55 | 0x3C55 | 15445 | 0 | 1 | 2 |
| 13 | 1784484492 | 0x6A5D128C | 0x128C | 4748 | 0 | 0 | 0 |
| 14 | 74243042 | 0x046CDBE2 | 0xDBE2 | -9246 | 0 | 0 | 1 |
| 15 | 114807987 | 0x06D7D4B3 | 0xD4B3 | -11085 | 0 | 1 | 2 |
| 16 | 1137522503 | 0x43CD3747 | 0x3747 | 14151 | 0 | 1 | 2 |
| 17 | 1441282327 | 0x55E83917 | 0x3917 | 14615 | 0 | 1 | 2 |
| 18 | 16531729 | 0x00FC4111 | 0x4111 | 16657 | 1 | 1 | 3 |
| 19 | 823378840 | 0x3113C398 | 0xC398 | -15464 | 0 | 1 | 2 |
| 20 | 143542612 | 0x088E4954 | 0x4954 | 18772 | 1 | 1 | 3 |
| 21 | 896544303 | 0x35702E2F | 0x2E2F | 11823 | 0 | 1 | 2 |
| 22 | 1474833169 | 0x57E82B11 | 0x2B11 | 11025 | 0 | 1 | 2 |
| 23 | 1264817709 | 0x4B63962D | 0x962D | -27091 | 1 | 2 | 4 |
| 24 | 1998097157 | 0x77188B05 | 0x8B05 | -29947 | 1 | 2 | 5 |

The `RandomInt` columns are illustrative only; the three ranges shown are ones the game really
asks for - 2 (pendulum direction, `GliderPRO/Sources/DynamicMaps.c:594`), 3 (number of telephone
rings, `GliderPRO/Sources/Play.c:736`) and 6 (star phase, `DynamicMaps.c:685`, and flower species,
`InterfaceInit.c:160`, which is `RandomInt(kNumFlowers)` with `kNumFlowers` = 6). The flame
registrars use 5, 5 and 4 (`kNumCandleFlames`, `kNumTikiFlames`, `kNumBBQCoals`), not 3 and 6.
A Go port that reproduces this table byte for byte is compliant with R-RNG-1.

## 1.7 `RandomInt` output distribution

Enumerated over all 65536 raw 16-bit words (i.e. assuming a uniform generator, which is what a
full-period Lehmer sequence approximates):

| range | distribution (value:count) | max-value note |
|---|---|---|
| 2 | 0:32767, 1:32768, **2:1** | `range` occurs once |
| 3 | 0:21845, 1:21846, 2:21844, **3:1** | once |
| 4 | 0:16383, 1:16384, 2:16384, 3:16384, **4:1** | once |
| 5 | 0:13107, 1:13108, 2:13106, 3:13108, 4:13106, **5:1** | once |
| 6 | 0:10923, 1:10922, 2:10922, 3:10924, 4:10922, 5:10922, **6:1** | once |
| 8 | 0:8191, 1..7:8192 each, **8:1** | once |
| 10 | 0:6553, 1:6554, 2:6554, 3:6554, 4:6552, 5:6554, 6:6554, 7:6554, 8:6554, 9:6552, **10:1** | once |
| 16 | 0:4095, 1..15:4096 each, **16:1** | once |
| 25000 | 0:3 ... 24999:2, **25000:1** | once |

Two structural facts a Go port must reproduce:

* Value 0 is always slightly *under*-represented (it needs `rawResult < 32768/range`, and the
  folded domain has 32769 values, so the bucket boundaries are not uniform).
* The value `range` occurs exactly once per 65536 raw words.

## 1.8 The naive-modular trap

A tempting one-liner is `state = (state * 16807) % 2147483647`. Under Python or Go `int64` this
agrees with Schrage everywhere. Under **strict C `int32`** (i.e. a literal 68k/PPC translation
with `long`) it does not, because `state * 16807` overflows when
`state > 2147483647 / 16807 = 127773`.

Measured: the first divergence is at **seed 127774**, and **172226 of the 299999 seeds in
[1, 299999]** produce a different successor. Because the sequence visits large states almost
immediately (draw 2 from seed 1 is already 282475249), a naive int32 implementation diverges from
the correct stream within two draws.

**Port rule:** compute in `int64`, or use Schrage in `int32`. Never `int32` multiply-then-mod.

## 1.9 The dead RNG paths (do not port)

```c
// GliderPRO/Sources/Utilities.c:88-109
long RandomLong (long range)
{
	register long	highWord, lowWord;
	register long	rawResultHi, rawResultLo;

	highWord = (range & 0xFFFF0000) >> 16;
	lowWord = range & 0x0000FFFF;

	rawResultHi = Random();
	if (rawResultHi < 0L)  rawResultHi *= -1L;
	rawResultHi = (rawResultHi * highWord) / 32768L;

	rawResultLo = Random();
	if (rawResultLo < 0L)  rawResultLo *= -1L;
	rawResultLo = (rawResultLo * lowWord) / 32768L;

	rawResultHi = (rawResultHi << 16) + rawResultLo;

	return (rawResultHi);
}
```

Zero callers. Note it consumes **two** draws and is not a uniform generator over `[0, range]`
at all (the two halves are scaled independently).

```c
// GliderPRO/Sources/Utilities.c:115-118 and 124-128
void InitRandomLongQUS (void)
{
	GetDateTime(&theSeed);
}
UInt32 RandomLongQUS (void)
{
	theSeed = theSeed * 1103515245 + 12345;
	return (theSeed);
}
```

The classic ANSI-C sample LCG (multiplier 1103515245 = 0x41C64E6D, increment 12345 = 0x3039,
implicit modulus 2^32). Zero callers.

Consequence: `PourScreenOn` (`GliderPRO/Sources/Transitions.c`, `RandomInt` at :44 inside an
unbounded rejection loop) is itself dead - it has zero callers - so the one place where an RNG
could have caused an unbounded stall never runs.

## 1.10 Census of all 32 `RandomInt` call sites (31 live, 1 dead)

Ranges and purposes below are transcribed from the call sites, not inferred from the variable
names.

| # | File:line | Call | Purpose | Category |
|---|---|---|---|---|
| 1 | `GliderPRO/Sources/DynamicMaps.c:338` | `RandomInt(kNumCandleFlames)` (5) | initial candle flame phase (`mode`, then `src` offset by `mode*15`) | animation |
| 2 | `GliderPRO/Sources/DynamicMaps.c:422` | `RandomInt(kNumTikiFlames)` (5) | initial tiki flame phase | animation |
| 3 | `GliderPRO/Sources/DynamicMaps.c:508` | `RandomInt(kNumBBQCoals)` (4) | initial coal phase | animation |
| 4 | `GliderPRO/Sources/DynamicMaps.c:594` | `RandomInt(2)` | initial pendulum *direction* (`toOrFro = (draw == 0)`); the pendulum's `mode` is hard-coded to 1 on the line above | animation |
| 5 | `GliderPRO/Sources/DynamicMaps.c:685` | `RandomInt(6)` (literal) | initial **star** phase in `AddStar` (`mode`, then `src` offset by `mode*31`) - not a flower | animation |
| 6 | `GliderPRO/Sources/Dynamics.c:304` | `RandomInt(240) + 60` | `HandleSparkleObject`: idle delay before the next sparkle, in frames | animation |
| 7 | `GliderPRO/Sources/Dynamics.c:492` | `200 + RandomInt(200)` | `HandleCoffee`: re-arm timer after the cycle completes | animation |
| 8 | `GliderPRO/Sources/Dynamics.c:506` | `200 + RandomInt(200)` | `HandleCoffee`: re-arm timer at the `timer == 100` sound cue | animation |
| 9 | `GliderPRO/Sources/Dynamics3.c:208` | `RandomInt(60) + 15` | `AddDynamicObject` `case kSparkle`: initial idle delay | animation |
| 10-18 | `GliderPRO/Sources/GameOver.c:123, 125, 127, 296, 322, 346, 369, 384, 385` | `right/5`, `bottom`, 6, 32, 8, 2, `4+4`, `8+8`, 2 | page-flutter animation | animation |
| 19-23 | `GliderPRO/Sources/ObjectAdd.c:621, 626, 689, 715, 742` | `10 + RandomInt(10)` x4, `RandomInt(kNumFlowers)` | editor: randomise new object's delay / flower species | editor |
| 24 | `GliderPRO/Sources/Transitions.c:44` | `RandomInt(colWide)` | `PourScreenOn` | **dead** |
| 25 | `GliderPRO/Sources/InterfaceInit.c:160` | `RandomInt(kNumFlowers)` (6) | `wasFlower` - the flower shown on the splash screen; not a "splash variant" selector | launch |
| 26-32 | `GliderPRO/Sources/Play.c:735, 736, 739, 759, 760, 775, 784` | `RandomInt(kRingSpread) + kRingBaseDelay`, `RandomInt(3) + 3`, `RandomInt(kChimeDelay) + 1`, then the same ring pair again, `RandomInt(2)`, `RandomInt(delayTime) + 1` | telephone ring + chimes | audio |

**`GliderPRO/Sources/Player.c` contains zero RNG calls.** Grepped for `Random`, `RandomInt`,
`RandomLong`, `rand`: no matches. The player physics integrator is entirely deterministic given
its inputs. The same is true of `PlayerControl.c`, `Interactions.c` and the enemy movers
(`Dynamics*.c` use the RNG only for the cosmetic timers listed above - the sparkle idle delay and
the coffee-maker cycle - never for anything the player collides with).

Per-file totals: `DynamicMaps.c` 5, `Dynamics.c` 3, `Dynamics3.c` 1, `GameOver.c` 9,
`ObjectAdd.c` 5, `Transitions.c` 1, `InterfaceInit.c` 1, `Play.c` 7 = **32 call sites**.
By category: **18 animation** (DynamicMaps 5 + Dynamics 3 + Dynamics3 1 + GameOver 9),
**5 editor**, **7 audio scheduling**, **1 launch-time**, **1 dead**.

## 1.11 `demo` 128, parsed byte-exactly

The demo record type:

```c
// GliderPRO/Headers/GliderStructs.h:334-339 - demoType
typedef struct
{
	long	frame;      // offset 0, 4 bytes, big-endian
	char	key;        // offset 4, 1 byte
	char	padding;    // offset 5, 1 byte
} demoType;             // sizeof == 6
```

`kDemoLength` = **6702** (`GliderPRO/Headers/GliderDefines.h:625`), so 6702 / 6 = **1117
records**.

Load path:

```c
// GliderPRO/Sources/StructuresInit2.c:281-297
demoData = (demoPtr)NewPtr(kDemoLength);
...
tempHandle = GetResource('demo', 128);
...
BlockMove(*tempHandle, demoData, kDemoLength);
```

Observed by parsing `data 'demo' (128)` out of the Rez dump:

```
length = 6702 bytes = 1117 records of 6 bytes
first 36 bytes: 00 00 00 2E 00 72 | 00 00 00 2F 00 72 | 00 00 00 30 00 72
                00 00 00 31 00 45 | 00 00 00 38 00 06 | 00 00 00 39 00 72
frame range: 46 .. 3414, strictly increasing: True, span 3368 frames (3369 frames inclusive)
key histogram: {0: 910, 1: 198, 3: 9}
gap histogram (top 8): {1:1009, 7:8, 4:6, 8:6, 3:6, 10:6, 14:4, 13:4}, max gap 100
padding byte: 109 distinct values; zeros = 407; top 5 = (0,407) (255,42) (32,39) (114,33) (111,24)
records with key 3 (banding) at record indices: 438, 439, 440, 459, 460, 461, 467, 468, 469
```

Reading the first record: frame `0x0000002E` = 46, key `0x00` = left, padding `0x72` = 'r'.

**Key encoding** (from `GliderPRO/Sources/Input.c:186-277`): 0 = left, 1 = right, 2 = battery,
3 = rubber band. The demo uses only 0, 1 and 3 - it never fires the battery.

**`padding` is uninitialised garbage.** The recorder writes only two fields:

```c
// GliderPRO/Sources/Input.c:44-49
void LogDemoKey (char keyIs)
{
	demoData[demoIndex].frame = gameFrame;
	demoData[demoIndex].key = keyIs;
	demoIndex++;
}
```

109 distinct padding values with 407 zeros is exactly the signature of stack/heap garbage from
the 1994 recording session frozen into the resource. **A Go port must ignore byte 5 entirely**
and must not attempt to reproduce it - but it *must* keep the record stride at 6, because the
resource is indexed by stride.

**`demoIndex` is unbounded.** `GetDemoInput` increments it on every consumed record with no
`< 1117` guard, so after frame 3414 the original reads 6 bytes past the end of the
`NewPtr(kDemoLength)` block. In practice the demo ends first (the glider dies or the timeout
fires), but a Go port must bounds-check or it will panic where C silently read garbage.

**`BUILD_ARCADE_VERSION` is 1**, and `GetDemoInput` contains a block that aborts the demo if the
human presses any of the four gameplay keys. So the demo is interruptible at any frame.

## 1.12 The replay experiment - R-RNG-2

`docs/analysis/determinism.md` Part 2 asserts that the shipped `demo` 128 replay depends on
`Random()`. **That assertion is false.** Proof, in four independent steps.

**Step 1 - the demo's input stream is a pure function of the frame counter.** `GetDemoInput`
(`GliderPRO/Sources/Input.c:186-277`) dispatches on
`if (gameFrame == (long)demoData[demoIndex].frame)` and then switches on `key`. No RNG. The
frame loop selects it with:

```c
// GliderPRO/Sources/Play.c:478-481
if (demoGoing)
	GetDemoInput(&theGlider);
else
	GetInput(&theGlider);
```

**Step 2 - the physics has no RNG.** §1.10: zero `Random`/`RandomInt` references in
`Player.c`. Trajectory is therefore a deterministic function of (initial state, input stream,
room geometry).

**Step 3 - the RNG-consuming objects are absent from Demo House.** Re-parsed with
`probe_house.py`:

| Property | Observed |
|---|---|
| rooms | 45 |
| object *slots* | 1080 = 45 rooms x `kMaxRoomObs` 24 |
| live objects (slot `what` != `kObjectIsEmpty` = -1) | 138 (942 slots empty) |
| `kCandle` instances | 1 |
| `kTaper` instances | 2 |
| `kStubby` instances | 0 |
| `kTiki` instances | 0 |
| `kBBQ` instances | 0 |
| `kCuckoo` instances | 1 |
| `kStar` instances | 1 |
| `kSparkle` (0x2D) instances | **0** |
| `kCoffee` (0x66) instances | **0** |
| `kChimes` (0x8F) instances | **0** |
| `phoneBit` | False |
| `wardBit` | False |
| `flags` | 0 |
| `initial` (start room, object) | (49, 107) |
| `bannerStarCountOn` | True |
| fileSize == expectedSize | 16526 == 16526 |

Only **5** RNG-consuming instances exist in the whole house (1 `kCandle` + 2 `kTaper` = 3 candle
flames, 1 `kCuckoo` pendulum, 1 `kStar`), and all five are one-shot *phase* initialisers that run
once at room entry. Re-verified by walking all 1080 slots rather than trusting each room's
`numObjects` count: both the candle (room 2 "Switches & Candles", slot 3, `numObjects` 13) and the
cuckoo (room 5 "Windows", slot 8, `numObjects` 10) sit inside their room's live range, and in any
case `IsThisValid` (`GliderPRO/Sources/Objects.c:89`) dispatches on the `what` field alone and
never consults `numObjects`.

**Step 4 - zero RNG draws occur inside the demo frame loop.** The only per-frame RNG consumer
that could fire during a demo is the telephone:

```c
// GliderPRO/Sources/Play.c:733-740 (verbatim)
void InitTelephone (void)
{
	thePhone.nextRing = RandomInt(kRingSpread) + kRingBaseDelay;
	thePhone.rings = RandomInt(3) + 3;
	thePhone.delay = kRingDelay;
	
	theChimes.nextRing = RandomInt(kChimeDelay) + 1;
}
```

(`numChimes` is *not* set here; it is a house-load count of `kChimes` objects.)

with `kRingSpread` = 25000 (`Play.c:21`), `kRingBaseDelay` = 5000 (`Play.c:22`),
`kChimeDelay` = 180 (`Play.c:23`). `InitTelephone` is called **once**, at `Play.c:192`, during
`NewGame`. It consumes 3 draws. Therefore:

* `thePhone.nextRing >= 5000` frames and `HandleTelephone` only decrements it once per frame
  (`Play.c:445` calls it from the frame loop), but the demo spans frames **46..3414**. The phone
  can never ring during the demo, so the two re-arm draws at `Play.c:759/760` never fire either.
* `numChimes == 0` because Demo House has zero `kChimes` objects, which kills the whole
  `if (numChimes > 0)` chime branch (`Play.c:775/784`) - the `theChimes.nextRing` value seeded by
  `InitTelephone` is simply never read.
* Even if the phone did ring, `HandleTelephone` (`Play.c:744-789`) only calls
  `PlayPrioritySound` and updates its own `thePhone`/`theChimes` counters. It has no effect on any
  physics variable.

**Total RNG consumption for a complete demo playthrough: at most 8 draws (3 at `InitTelephone`,
plus up to 5 flame/pendulum/star phase initialisers as rooms are entered), none of which reaches
gameplay state.**

## 1.13 The containment proof - the RNG only touches pixels

Grepped every array written by an RNG-consuming registrar and enumerated every reader:

| Array | Written at | Read at | Effect of the read |
|---|---|---|---|
| `flames[]` | `DynamicMaps.c:337-341` | `Render.c:202-216` | advance mode, pick src rect, `CopyBits(..., srcCopy, nil)` |
| `tikiFlames[]` | `DynamicMaps.c:421-426` | `Render.c:221-235` | same |
| `bbqCoals[]` | `DynamicMaps.c:507-511` | `Render.c:240-254` | same |
| `pendulums[]` | `DynamicMaps.c:592-603` | `Render.c:278-317` | same |
| `theStars[]` | `DynamicMaps.c:684-690` | `Render.c:429-445` | same |

There are **no other readers** of any of the five arrays anywhere in the tree. Every reader is
a `CopyBits` source-rect selection. No collision rect, no `IsThisValid` test, no hot-spot list,
no player field ever consults them.

**R-RNG-2 (normative):** the shipped demo's glider trajectory is RNG-independent. A Go port can
replay `demo` 128 frame-exactly with a stub `Random()` that returns 0 forever, and the glider
will follow the identical path. The RNG is required only for pixel-exact *animation*, which is
where R-RNG-1 applies.

## 1.14 Where R-RNG-1 *is* observable

Enumerated, with the observable difference if the port's stream differs from R-RNG-1:

| Observable | Sites | Visible difference |
|---|---|---|
| Candle / taper / stubby flame phase | `DynamicMaps.c:338` | which of 5 flame frames a candle shows on the first frame you enter the room; self-synchronising thereafter (mode advances +1/frame) |
| Tiki torch phase | `DynamicMaps.c:422` | as above, 5 frames |
| BBQ coal phase | `DynamicMaps.c:508` | as above, 4 frames |
| Pendulum swing direction | `DynamicMaps.c:594` | whether the cuckoo-clock pendulum starts swinging to or fro; there are only **2** pendulum src frames (`Render.c:286` flips at `mode >= 2`) and `mode` itself is hard-coded to 1, so the draw is a 1-bit phase choice, not a 12-phase one |
| Star phase | `DynamicMaps.c:685` | which of 6 star frames a `kStar` starts on; self-synchronising thereafter (`Render.c:431-438` advances +1/frame, wrapping at 6) |
| Sparkle timing | `Dynamics.c:304`, `Dynamics3.c:208` | when an idle `kSparkle` next twinkles: `RandomInt(60)+15` frames for the first wait, `RandomInt(240)+60` for each subsequent one |
| Coffee-maker cycle | `Dynamics.c:492, :506` | `200 + RandomInt(200)` frames between brew cycles, so the sound cue and light drift out of phase with the original if the stream differs |
| Game-over page flutter | `GameOver.c:123, 125, 127, 296, 322, 346, 369, 384, 385` | the whole death animation's trajectory - the single most visible RNG effect in the game |
| Splash flower | `InterfaceInit.c:160` | `wasFlower = RandomInt(kNumFlowers)` - which of 6 flower bitmaps the splash screen shows at launch; **persistent**, so this is the most conspicuous single draw |
| Telephone ring schedule | `Play.c:735, 736, 739` | when the phone rings in a house that has one |
| Chime schedule | `Play.c:759, 760, 775, 784` | chime timing in a house with `kChimes` |
| Editor object randomisation | `ObjectAdd.c:621, 626, 689, 715, 742` | initial state of a newly placed object |

The game-over animation is the one place where a wrong RNG is *obviously* wrong to a player who
knows the original, and it is purely cosmetic.

## 1.15 Consequences of choosing the other seeding branch

If a port instead emulates the non-Carbon target (`GetDateTime` into `qd.randSeed`):

1. The splash flower at `InterfaceInit.c:160` becomes 1-of-6 per launch instead of fixed.
2. Every flame/coal/pendulum/star in every room gets a different initial phase per launch.
3. The game-over page flutter differs per launch.
4. The telephone ring time differs per launch.
5. Trajectory, scoring, star counts, the demo, and every recorded/replayed input stream are
   **unaffected** (by R-RNG-2 and §1.13).
6. You inherit the unresolvable question of Apple's negative-seed behaviour (§1.3).

Recommendation: implement R-RNG-1 with a fixed seed of 1 as the default, and expose the seed as
a configuration knob. Anything that claims frame-exactness must use seed 1.

## 1.16 Go implementation sketch

```go
// qdRand mirrors the QuickDraw global qd.randSeed and the Random() trap.
type qdRand struct{ seed int32 }

const (
	lehmerA = 16807      // 0x41A7
	lehmerM = 2147483647 // 0x7FFFFFFF
	lehmerQ = 127773     // m / a
	lehmerR = 2836       // m % a
)

func newQDRand() *qdRand { return &qdRand{seed: 1} } // R-RNG-1: Carbon leaves randSeed == 1

// Random reproduces pascal short Random(void).
func (g *qdRand) Random() int16 {
	hi := g.seed / lehmerQ
	lo := g.seed % lehmerQ
	t := lehmerA*lo - lehmerR*hi
	if t > 0 {
		g.seed = t
	} else {
		g.seed = t + lehmerM
	}
	return int16(uint16(g.seed)) // low word, reinterpreted; do not clamp
}

// RandomInt reproduces GliderPRO/Sources/Utilities.c:72-82 exactly,
// including the inclusive upper bound: the result is in [0, rng] not [0, rng).
func (g *qdRand) RandomInt(rng int16) int16 {
	raw := int32(g.Random())
	if raw < 0 {
		raw = -raw
	}
	raw = raw * int32(rng) / 32768
	return int16(raw)
}
```

Do **not** substitute `math/rand`. Do **not** use `rand.Intn`, which excludes the upper bound
and would eliminate the 1-in-65536 `range` outcome.

---

# Part 2 - `srcXor` on an 8-bit indexed pixmap

## 2.1 There is exactly one `srcXor` in the tree

Complete transfer-mode census, obtained by grepping all 67 `.c` files for every QuickDraw
transfer-mode constant:

| Mode | Numeric value | Occurrences | Where |
|---|---|---|---|
| `srcCopy` | 0 | 119 | everywhere |
| `srcOr` | 1 | 1 | `GliderPRO/Sources/ObjectEdit.c:2364` - `PenMode(srcOr)`, see §2.9 |
| `srcXor` | 2 | **1** | `GliderPRO/Sources/Marquee.c:438` |
| `srcBic` | 3 | 0 | - |
| `notSrcCopy` | 4 | 0 | - |
| `notSrcOr` | 5 | 0 | - |
| `notSrcXor` | 6 | 0 | - |
| `notSrcBic` | 7 | 0 | - |
| `patCopy` | 8 | 0 | - |
| `patOr` | 9 | 6 | pen modes in the editor |
| `patXor` | 10 | 15 | marquee marching ants, all enumerated in §2.8 |
| `patBic` | 11 | 0 | - |
| `notPatCopy` | 12 | 0 | - |
| `notPatOr` | 13 | 0 | - |
| `notPatXor` | 14 | 0 | - |
| `notPatBic` | 15 | 0 | - |
| `transparent` | 36 | 2 | `ObjectDraw2.c:1403`, `ObjectDraw2.c:1430` |

So resolving `srcXor` means resolving exactly one call:

```c
// GliderPRO/Sources/Marquee.c:432-439
void DrawGliderMarquee (void)
{
	CopyBits((BitMap *)*GetGWorldPixMap(blowerMaskMap),
			GetPortBitMapForCopyBits(GetWindowPort(mainWindow)),
			&leftStartGliderSrc,
			&marqueeGliderRect,
			srcXor, nil);
}
```

## 2.2 The source is 1 bit deep

```c
// GliderPRO/Sources/StructuresInit.c:253-260
QSetRect(&blowerSrcRect, 0, 0, 48, 402);	// 19344 pixels
theErr = CreateOffScreenGWorld(&blowerSrcMap, &blowerSrcRect, kPreferredDepth);
SetGWorld(blowerSrcMap, nil);
LoadGraphic(kBlowerPictID);

theErr = CreateOffScreenGWorld(&blowerMaskMap, &blowerSrcRect, 1);
SetGWorld(blowerMaskMap, nil);
LoadGraphic(kBlowerPictID + 1000);
```

`blowerMaskMap` is created at **depth 1** (the literal `1` third argument). `kPreferredDepth`
is **8** (`GliderPRO/Headers/Externs.h:15`), used for the colour twin `blowerSrcMap`.

```c
// GliderPRO/Sources/Utilities.c:266-278
OSErr CreateOffScreenGWorld (GWorldPtr *theGWorld, Rect *bounds, short depth)
{
	...
	theErr = NewGWorld(theGWorld, depth, bounds, nil, nil, useTempMem);
	if (theErr != noErr)
		theErr = NewGWorld(theGWorld, depth, bounds, nil, nil, 0);
	...
	LockPixels(GetGWorldPixMap(*theGWorld));
	...
}
```

The source rect:

```c
// GliderPRO/Sources/StructuresInit.c:280-284
QSetRect(&leftStartGliderSrc, 0, 0, 48, 16);
QOffsetRect(&leftStartGliderSrc, 0, 358);

QSetRect(&rightStartGliderSrc, 0, 0, 48, 16);
QOffsetRect(&rightStartGliderSrc, 0, 374);
```

So the stencil is a 48 x 16 region of a 1-bit mask, taken from `(0, 358)`-`(48, 374)` in the
`blowerMaskMap`. (Note `kGliderWide` = 48, `kGliderHigh` = 20, `kHalfGliderWide` = 24 -
`GliderDefines.h:548-550` - the *mask* strip is 16 tall, not 20.)

Destination is the main window's port, which is `kPreferredDepth` = 8 bits indexed (or 4, but
the marquee is editor-only and the editor is 8-bit; §3 covers depth 4).

## 2.3 What QuickDraw does with a 1-bit source and a deeper destination

This is the crux, and it is what makes rendering.md Q1's "two candidate answers" collapse into
one. Classic QuickDraw's rule for `CopyBits` from a 1-bit source into a destination of greater
depth is:

1. **Colourise first.** Each source bit is expanded to the destination's pixel size using the
   port's colours: a **set** bit (1) becomes the port's **foreground** colour, a **clear** bit
   (0) becomes the port's **background** colour. This happens before the boolean operation.
2. **Then apply the boolean transfer mode** on the resulting *pixel values* - i.e. on **palette
   indices**, not on RGB triples. `srcXor` computes `dest = dest ^ src` index-wise.

The consequence: the value XOR'd into the destination is `Index2Color`-free. It is literally the
palette index that the current foreground/background colour resolves to on the destination
device.

## 2.4 What foreground and background are at the call site

`DrawGliderMarquee` sets no colours. Its callers are `SetMarqueeGliderRect`
(`Marquee.c:443-451`), `StopMarquee` (`Marquee.c:141-144`) and `DrawMarquee`
(`Marquee.c:493-494`); none of the three touches `ForeColor`/`RGBForeColor` either. So the port
state is whatever the last drawing operation left.

`GliderPRO/Sources/ColorUtils.c` proves that state is invariantly **black foreground, white
background**. Every colour helper in the file follows the identical save/restore idiom:

```c
// GliderPRO/Sources/ColorUtils.c:20-29   (ColorText; :36-45 ColorRect, :52-61 ColorOval,
//                                          :68-77 ColorRegion, :84-94 ColorLine,
//                                          :103-113 HiliteRect, :120-129 ColorFrameRect,
//                                          :137+   ColorFrameWHRect)
void ColorText (StringPtr theStr, long color)
{
	RGBColor	theRGBColor, wasColor;

	GetForeColor(&wasColor);
	Index2Color(color, &theRGBColor);
	RGBForeColor(&theRGBColor);
	DrawString(theStr);
	RGBForeColor(&wasColor);
}
```

`GetForeColor` -> draw -> `RGBForeColor(wasColor)`. Nothing in the tree leaves a non-black
foreground behind on the main window's port, and the scattered explicit `ForeColor(blackColor)`
calls (e.g. `Scoreboard.c`'s drop-shadow idiom, `Banner.c:117`) restore it as well.

On the standard 8-bit palette (`clut` 128, verified below):

| Colour | QuickDraw constant | value | 8-bit palette index | 16-bit RGB | hex |
|---|---|---|---|---|---|
| black | `blackColor` | 33 | **255** | 0000 0000 0000 | `#000000` |
| white | `whiteColor` | 30 | **0** | FFFF FFFF FFFF | `#FFFFFF` |

Observed by parsing `data 'clut' (128)`:

```
clut 128 len 2056  ctSeed 00000000  ctFlags 0000  ctSize 255   (ctSize is count-1, so 256 entries)
  idx   0 value=  0 rgb16=FFFF FFFF FFFF  #FFFFFF
  idx 255 value=255 rgb16=0000 0000 0000  #000000
```

## 2.5 R-XOR-1 - the normative ruling

For the single `srcXor` call at `GliderPRO/Sources/Marquee.c:438`, on an 8-bit indexed
destination with black foreground and white background:

```
for each destination pixel p covered by the blit:
    if the corresponding 1-bit source pixel is SET:
        dest[p] = dest[p] ^ 0xFF        // fg = black = index 255
    else:
        dest[p] = dest[p] ^ 0x00        // bg = white = index 0  -> no-op
```

Verified identities over all 256 indices:

* `i ^ 255 == 255 - i` for every `i` in 0..255. On the standard palette this is *not* a
  photometric negation - it is an index reflection - which is why the glider stencil looks like
  garish false colour over a coloured room and like clean inverse video over gray.
* `(i ^ 255) ^ 255 == i` for every `i`. The operation is an involution.

## 2.6 Where the involution is relied on

Three places, all in `Marquee.c`:

**(a) `SetMarqueeGliderRect` draws, `StopMarquee` undoes.**

```c
// GliderPRO/Sources/Marquee.c:443-451
void SetMarqueeGliderRect (short h, short v)
{
	marqueeGliderRect = leftStartGliderSrc;
	ZeroRectCorner(&marqueeGliderRect);
	QOffsetRect(&marqueeGliderRect, h - kHalfGliderWide, v - kGliderHigh);

	DrawGliderMarquee();
	gliderMarqueeUp = true;
}
```

```c
// GliderPRO/Sources/Marquee.c:139-157
void StopMarquee (void)
{
	if (gliderMarqueeUp)
	{
		DrawGliderMarquee();          // second XOR -> restores the screen
		gliderMarqueeUp = false;
	}

	if (!theMarquee.active)
		return;

	SetPortWindowPort(mainWindow);
	PenMode(patXor);
	PenPat(&theMarquee.pats[theMarquee.index]);
	DrawMarquee();
	PenNormal();
	theMarquee.active = false;
	SetCoordinateHVD(-1, -1, -1);
}
```

Note the ordering hazard: `StopMarquee` XORs the glider stencil **before** it re-establishes
`patXor`, and it does not `SetPortWindowPort` until after. In practice the port is already the
main window, but a Go port that models ports explicitly must replicate the order or the stencil
will be erased from the wrong drawable.

**(b) `DrawMarquee` re-XORs the stencil on every ant-march tick.**

```c
// GliderPRO/Sources/Marquee.c:493-494 (the last two lines of DrawMarquee, 455-495)
	if (gliderMarqueeUp)
		DrawGliderMarquee();
```

`DrawMarquee` is itself called under `PenMode(patXor)` (see §2.8) both to draw and to erase the
marching-ants rectangle, so each blink calls `DrawGliderMarquee` **twice** - once on the erase
pass and once on the draw pass. **Every marquee blink applies an even number of XORs to the
stencil**, which is why the stencil appears steady while the ants march. A Go port that
"optimises" the erase pass away will make the glider stencil flicker at the ant-march rate.

**(c) `PauseMarquee` / `ResumeMarquee` bracket a nested save/restore.**

```c
// GliderPRO/Sources/Marquee.c:161-168 and 172-184
void PauseMarquee (void)
{
	if (!theMarquee.active)  return;
	theMarquee.paused = true;
	StopMarquee();
}

void ResumeMarquee (void)
{
	if (!theMarquee.paused)  return;
	if (theMarquee.handled)
	{
		StartMarqueeHandled(&theMarquee.bounds, theMarquee.direction, theMarquee.dist);
		HandleBlowerGlider();
	}
	else
		StartMarquee(&theMarquee.bounds);
}
```

`ResumeMarquee` re-arms the stencil only via `HandleBlowerGlider`
(`GliderPRO/Sources/ObjectEdit.c:1972`, the **only** caller of `SetMarqueeGliderRect` in the
entire tree), and only when `theMarquee.handled` is true. If a marquee was paused with
`gliderMarqueeUp` true but `handled` false, the stencil is lost on resume. That is an original
bug; it is benign because the two flags are set together.

## 2.7 The marquee state record

```c
// GliderPRO/Sources/Marquee.h:14-20
typedef struct
{
	Pattern		pats[kNumMarqueePats];    // 7 * 8 = 56 bytes, offset 0
	Rect		bounds, handle;           // 8 + 8 = 16 bytes, offsets 56, 64
	short		index, direction, dist;   // 6 bytes, offsets 72, 74, 76
	Boolean		active, paused, handled;  // 3 bytes, offsets 78, 79, 80
} marquee;                                // sizeof == 82 (1 byte tail pad, 2-byte alignment)
```

with `kNumMarqueePats` = 7 (`GliderPRO/Headers/GliderDefines.h:460`),
`kMarqueePatListID` = 128 (`Marquee.c:15`), `kHandleSideLong` = 9 (`Marquee.c:16`).

Module globals (`Marquee.c:23-25`): `marquee theMarquee; Rect marqueeGliderRect;
Boolean gliderMarqueeUp;`.

## 2.8 `PAT#` 128 and the `patXor` marching ants

```c
// GliderPRO/Sources/Marquee.c:499-510
void InitMarquee (void)
{
	short		i;

	for (i = 0; i < kNumMarqueePats; i++)
		GetIndPattern(&theMarquee.pats[i], kMarqueePatListID, i + 1);
	theMarquee.index = 0;
	theMarquee.active = false;
	theMarquee.paused = false;
	theMarquee.handled = false;
	gliderMarqueeUp = false;
}
```

Observed by parsing `data 'PAT#' (128)`:

```
PAT# 128 len 58 bytes, count 7   (2-byte big-endian count, then 7 * 8 bytes)
   pat 0: F8 F1 E3 C7 8F 1F 3E 7C
   pat 1: 3E 7C F8 F1 E3 C7 8F 1F
   pat 2: 1F 3E 7C F8 F1 E3 C7 8F
   pat 3: 8F 1F 3E 7C F8 F1 E3 C7
   pat 4: C7 8F 1F 3E 7C F8 F1 E3
   pat 5: E3 C7 8F 1F 3E 7C F8 F1
   pat 6: F1 E3 C7 8F 1F 3E 7C F8
```

A `Pattern` is 8 bytes = an 8x8 1-bit tile, one byte per row, MSB = leftmost pixel. Pattern 0 is
a 45-degree diagonal stripe (5 on, 3 off per row - `0xF8` - each row being the previous row rotated
left by 1). Fixing one convention: **pattern `k` is pattern 0 rotated *down* by `k + 1` rows** for
`k` = 1..6 (`pats[k][i] == pats[0][(i - k - 1) mod 8]`), verified byte by byte on the table above.
Because each row is the row above rotated left by 1, a vertical rotation of this particular tile is
indistinguishable from a horizontal one: pattern `k` is also pattern 0 shifted **right** by
`k + 1` pixels (mod 8).

So cycling `theMarquee.index` 0,1,2,...,6,0,... walks the stripe right by 2,1,1,1,1,1 pixels and
then 1 more on the 6 -> 0 wrap. Seven patterns covering an 8-pixel period means the ants advance
one *double* step per cycle - the sequence is deliberately not a uniform +1, and a Go port that
generates the seven tiles procedurally with a uniform shift will produce a visibly smoother march
than the original.

A pattern under `patXor` toggles the destination where the pattern bit is set. On an 8-bit
destination with black pen, the same `^0xFF` rule as R-XOR-1 applies (a `Pattern` is a 1-bit
object and is colourised fg/bg identically).

All 15 `PenMode(patXor)` sites in the tree, enumerated by grep (note that four of them are *not*
in the marquee/editor drag paths, and that only the eight `Marquee.c` ones use the `PAT#` 128
ants - the other seven use the QuickDraw 50% `gray` pattern):

| File:line | Function | Pen pattern |
|---|---|---|
| `Marquee.c:41` | `DoMarquee` (the ant-march tick) | `theMarquee.pats[index]` |
| `Marquee.c:67` | `StartMarquee` | `theMarquee.pats[index]` |
| `Marquee.c:130` | `StartMarqueeHandled` | `theMarquee.pats[index]` |
| `Marquee.c:151` | `StopMarquee` (erase) | `theMarquee.pats[index]` |
| `Marquee.c:195` | `DragOutMarqueeRect` | `theMarquee.pats[index]` |
| `Marquee.c:225` | `DragMarqueeRect` | `theMarquee.pats[index]` |
| `Marquee.c:269` | `DragMarqueeHandle` | `theMarquee.pats[index]` |
| `Marquee.c:352` | `DragMarqueeCorner` | `theMarquee.pats[index]` |
| `DialogUtils.c:261` | dialog zoom-open animation | `qd.gray` |
| `DialogUtils.c:315` | alert zoom-open animation | `qd.gray` |
| `ObjectEdit.c:2747` | "show object rects" overlay (`kMaxRoomObs` frames) | `qd.gray` |
| `RoomInfo.c:132`, `:172`, `:193`, `:213` | room-info tile picker drag feedback | `qd.gray` |

There is no `AdvanceMarqueeIndex` function in the tree; the index is advanced inside `DoMarquee`.
`DragOutMarqueeRect` is the clearest illustration of the involution being used as an eraser:

```c
// GliderPRO/Sources/Marquee.c:188-214
void DragOutMarqueeRect (Point start, Rect *theRect)
{
	Point		wasPt, newPt;

	SetPortWindowPort(mainWindow);
	InitCursor();
	QSetRect(theRect, start.h, start.v, start.h, start.v);
	PenMode(patXor);
	PenPat(&theMarquee.pats[theMarquee.index]);
	FrameRect(theRect);
	wasPt = start;

	while (WaitMouseUp())
	{
		GetMouse(&newPt);
		if (DeltaPoint(wasPt, newPt))
		{
			FrameRect(theRect);          // erase old
			QSetRect(theRect, start.h, start.v, newPt.h, newPt.v);
			NormalizeRect(theRect);
			FrameRect(theRect);          // draw new
			wasPt = newPt;
		}
	}
	FrameRect(theRect);                  // final erase
	PenNormal();
}
```

## 2.9 The one anomalous mode

`GliderPRO/Sources/ObjectEdit.c:2364` calls `PenMode(srcOr)`. `srcOr` is 1 and is a *source*
mode, not a *pattern* mode; the pattern-mode family starts at `patCopy` = 8. Passing 1 to
`PenMode` selects `srcOr` semantics for pattern drawing, which classic QuickDraw tolerates but
which is documented as undefined for pen operations. The intent is plainly `patOr` = 9.

The site verbatim (inside `DrawObjectRects`, guarded by "this room has no lights"):

```c
// GliderPRO/Sources/ObjectEdit.c:2362-2367
if (GetNumberOfLights(thisRoomNumber) <= 0)
{
	PenMode(srcOr);
	PenPat(GetQDGlobalsGray(&dummyPattern));
	PaintRect(&backSrcRect);
	PenNormal();
}
```

The pen pattern is therefore the QuickDraw **50% gray** stipple, *not* solid black, and the
destination is the offscreen `backSrcMap` - the effect is to darken an unlit room in the editor by
laying black on every other pixel. Under either interpretation the "or" arithmetic is the same
(pattern bits set -> foreground black; pattern bits clear -> destination untouched), so a Go port
can implement it as `patOr` with a 50% pattern. This is worth a code comment, not a behaviour
change.

## 2.10 Go implementation sketch

```go
// XorStencil applies R-XOR-1: CopyBits(src1bit, dst8, srcXor) with fg=black(255), bg=white(0).
//
// srcMask is the 1-bit source, one bit per pixel, MSB-first within each byte.
// Callers must supply srcRect in source coordinates and dstRect in destination coordinates;
// GliderPRO always uses equal-size rects here (48 x 16), so no scaling is implemented.
func XorStencil(dst *Indexed8, dstRect Rect, srcMask *Bitmap1, srcRect Rect) {
	const fgIndex = 255 // black on clut 128
	const bgIndex = 0   // white on clut 128 -> XOR by 0 is a no-op, so clear bits are skipped
	_ = bgIndex
	for y := 0; y < srcRect.Height(); y++ {
		for x := 0; x < srcRect.Width(); x++ {
			if srcMask.At(srcRect.Left+x, srcRect.Top+y) {
				p := dst.Offset(dstRect.Left+x, dstRect.Top+y)
				dst.Pix[p] ^= fgIndex
			}
		}
	}
}
```

Two things a port must resist:

* Do **not** convert to RGB and XOR the channels. `0xCC ^ 0xFF` in RGB space is not the same
  pixel as palette index `i ^ 0xFF`, and the visual result differs on every non-gray index.
* Do **not** treat clear source bits as "leave alone" *by design* - they are XOR-by-0, which
  happens to be a no-op only because background is index 0. If a port ever renders with a
  non-white background, the clear bits must XOR by that index. In Glider PRO's 8-bit path they
  never do.

---

# Part 3 - the 16-entry system 4-bit grayscale CLUT

## 3.1 Depth 4 is always grayscale

Three functions establish this.

```c
// GliderPRO/Sources/Environ.c:348-368
Boolean AreWeColorOrGrayscale (void)
{
	GDHandle	thisGDevice;
	Boolean		colorOrGray;

	thisGDevice = GetMainDevice();
	colorOrGray = (**thisGDevice).gdFlags & 0x0001;   // dead store, immediately overwritten
	...
	colorOrGray = (**thisGDevice).gdFlags & 0x0001;
	return (colorOrGray);
}
```

(The first read is a dead store - an original bug, harmless.) `gdFlags` bit 0 is `gdDevType`:
1 = colour device, 0 = monochrome/grayscale.

```c
// GliderPRO/Sources/Environ.c:374-398
void SwitchToDepth (short newDepth, Boolean doColor)
{
	...
	colorFlag = (doColor) ? 1 : 0;
	theErr = SetDepth(thisGDevice, newDepth, 1, colorFlag);
	...
}
```

`SetDepth(gd, depth, whichFlags, flags)` with `whichFlags == 1` means "set the gdDevType bit
from `flags`".

```c
// GliderPRO/Sources/Environ.c:505-543   (HandleDepthSwitching, abridged to the branches)
	case kSwitchTo256Colors:      //  1
		SwitchToDepth(8, true);
		break;
	case kSwitchTo16Grays:        //  2
		SwitchToDepth(4, false);   // <-- doColor FALSE
		break;
	case kSwitchIfNeeded:         //  0
		... accepts depth 4 only when (!wasColorOrGray) ...
```

with `kSwitchIfNeeded` = 0, `kSwitchTo256Colors` = 1, `kSwitchTo16Grays` = 2
(`GliderPRO/Headers/GliderDefines.h:43-45`).

**Therefore depth 4 is *only ever* entered with `doColor == false`, i.e. as a grayscale
device.** The one exception is a bug:

```c
// GliderPRO/Sources/Environ.c:549-554
void RestoreColorDepth (void)
{
	if (isDepthChangeable && (wasDepth != thisMac.isDepth))
		SwitchToDepth(wasDepth, true);      // always true, even when restoring to 4
}
```

`RestoreColorDepth` unconditionally passes `doColor = true`, so quitting from a 4-bit session
restores the display as a 4-bit *colour* device. A Go port should reproduce the depth change but
need not reproduce the flag bug (nothing in the game observes it after the switch).

The depth-switch alert:

```c
// GliderPRO/Sources/Environ.c:404-431   SwitchDepthOrAbort
//   Alert(kSwitchDepthAlert, nil) result:
//     1 -> SwitchToDepth(8, true)
//     2 -> SwitchToDepth(4, false)
//     3 -> ExitToShell()
```

and `CheckOurEnvirons` (`Environ.c:437-470`) hard-codes `can1Bit`, `can4Bit`, `can8Bit` all
`true`, so the game always believes 4-bit is available.

## 3.2 Why depth matters at all: `Index2Color` resolves against the current device

Every colour in Glider PRO is drawn through `ColorUtils.c` (§2.4) or an inline copy of the same
idiom, and the pivotal call is `Index2Color(color, &theRGBColor)`. `Index2Color` is a Palette
Manager / Color Manager call that looks up an **index into the current `GDevice`'s colour
table** and returns the RGB. So the same literal index means a different colour at depth 8 and
depth 4:

* At depth 8 the device CLUT is the standard 256-entry Apple palette (`clut` 128's content).
* At depth 4 the device CLUT has **16** entries.

That is the entire reason 33 sites in the tree are written as
`if (thisMac.isDepth == 4) ColorXxx(..., i4); else ColorXxx(..., i8);`.

It is also why an index > 15 at depth 4 is a read past the end of a 16-entry table - see §3.11.

## 3.3 R-CLUT4-1 - the normative ramp

```
for i in 0..15:
    level16 := uint16((15 - i) * 0x1111)
    clut4[i] = RGBColor{red: level16, green: level16, blue: level16}
```

Equivalently, in 8-bit-per-channel terms, `level8(i) = (15 - i) * 17`:

| `i4` | `(15-i)*17` | hex | 16-bit channel | Apple's name |
|---|---|---|---|---|
| 0 | 255 | 0xFF | 0xFFFF | white |
| 1 | 238 | 0xEE | 0xEEEE | |
| 2 | 221 | 0xDD | 0xDDDD | |
| 3 | 204 | 0xCC | 0xCCCC | |
| 4 | 187 | 0xBB | 0xBBBB | |
| 5 | 170 | 0xAA | 0xAAAA | |
| 6 | 153 | 0x99 | 0x9999 | |
| 7 | 136 | 0x88 | 0x8888 | |
| 8 | 119 | 0x77 | 0x7777 | |
| 9 | 102 | 0x66 | 0x6666 | |
| 10 | 85 | 0x55 | 0x5555 | |
| 11 | 68 | 0x44 | 0x4444 | |
| 12 | 51 | 0x33 | 0x3333 | |
| 13 | 34 | 0x22 | 0x2222 | |
| 14 | 17 | 0x11 | 0x1111 | |
| 15 | 0 | 0x00 | 0x0000 | black |

Index 0 is white and index 15 is black - the same polarity as the 8-bit palette (index 0 white,
index 255 black), which is why `ForeColor(blackColor)`/`ForeColor(whiteColor)` work unchanged at
both depths.

## 3.4 The proof (not an inference)

The ramp is not merely plausible; it is forced by the data. **The standard 8-bit palette
contains exactly one index for each of the sixteen `(15-i)*0x1111` gray levels.** Parsed from
`data 'clut' (128)`:

```
gray8 = [0, 245, 246, 43, 247, 248, 86, 249, 250, 129, 251, 252, 172, 253, 254, 255]
all-unique: True
```

i.e.

| `i4` | level | 8-bit index (`gray8[i4]`) | verified 16-bit RGB |
|---|---|---|---|
| 0 | 0xFF | 0 | FFFF FFFF FFFF |
| 1 | 0xEE | 245 | EEEE EEEE EEEE |
| 2 | 0xDD | 246 | DDDD DDDD DDDD |
| 3 | 0xCC | **43** | CCCC CCCC CCCC |
| 4 | 0xBB | 247 | BBBB BBBB BBBB |
| 5 | 0xAA | 248 | AAAA AAAA AAAA |
| 6 | 0x99 | **86** | 9999 9999 9999 |
| 7 | 0x88 | 249 | 8888 8888 8888 |
| 8 | 0x77 | 250 | 7777 7777 7777 |
| 9 | 0x66 | **129** | 6666 6666 6666 |
| 10 | 0x55 | 251 | 5555 5555 5555 |
| 11 | 0x44 | 252 | 4444 4444 4444 |
| 12 | 0x33 | **172** | 3333 3333 3333 |
| 13 | 0x22 | 253 | 2222 2222 2222 |
| 14 | 0x11 | 254 | 1111 1111 1111 |
| 15 | 0x00 | 255 | 0000 0000 0000 |

The four bolded indices (43, 86, 129, 172) come from the 6x6x6 colour cube (`0xCC`, `0x99`,
`0x66`, `0x33` are cube levels), and the other twelve come from index 0 plus the ten
"extra grays" 245-254 plus index 255. This explains a detail that would otherwise look
arbitrary: the game's own colour constants `k8LtstGray3Color` = 43 and `k8DkGray3Color` = 172
are cube entries rather than members of the contiguous 245-254 run.

**Now the decisive test.** Of the 13 authored `(i4, i8)` pairs whose `i8` is a *pure gray*,
**11 satisfy `gray8[i4] == i8` exactly**: `i4` in {1, 2, 3, 4, 5, 7, 10, 11, 13, 14, 15}. The
author was, demonstrably, hand-transcribing this exact ramp. The two misses are:

* `(15, 254)` - `#111111`, which is `gray8[14]`, not `gray8[15]`. 5 sites, all inline
  drop-shadows (`ObjectDraw.c:185, 310, 405, 547, 692`), i.e. "one step lighter than black"
  written as black.
* `(8, 251)` - `#555555`, which is `gray8[10]`, not `gray8[8]`. 1 site
  (`ObjectDraw2.c:512`).

Two authoring slips out of thirteen. No competing ramp hypothesis explains eleven exact
identity hits.

## 3.5 The 72 authored `(i4, i8)` pairs

Extracted mechanically from the 33 `thisMac.isDepth == 4` conditional sites in the tree, then
hand-verified. (Extraction details worth recording: a regex over integer literals must strip a
trailing `L`/`l` - `Index2Color` takes a `long`, so the literals are written `7L`, `251L` etc.
Two `ColorRegion(shadowRgn, dkGrayC)` sites pass a local variable and were resolved by hand
(`ObjectDraw.c:405, 547`, where the enclosing function's own depth block has already set
`dkGrayC = k8DkstGrayColor` = 254). Eight of the pairs are two-line `ColorLine` calls
(`ObjectDraw.c:578-593` / `597-612`) whose index argument sits on the *second* line, so a
line-oriented extractor must join continuations or it will miss them.

Only 6 of the 33 depth-4 sites contribute no `(i4, i8)` pair at all: `MainWindow.c:68` switches to
`ForeColor(whiteColor)`/`ForeColor(blackColor)` rather than to a palette index, and the five
`DynamicMaps.c` sites (`:326, 409, 495, 584, 671`) are the even-column *coordinate* fixups of
§3.13, not colour choices.)

| # | file:line (4-bit) | file:line (8-bit) | i4 | i8 | 8-bit RGB | lum | (15-i4)*17 | nearest i4 | R1 ok |
|---|---|---|---|---|---|---|---|---|---|
| 1 | ObjectDraw2.c:437 | ObjectDraw2.c:445 | 3 | 42 | #CCCCFF | 208 | 204 | 3 | yes |
| 2 | ObjectDraw.c:582 | ObjectDraw.c:601 | 3 | 43 | #CCCCCC | 203 | 204 | 3 | yes |
| 3 | ObjectDraw.c:279 | ObjectDraw.c:287 | 7 | 52 | #CC9933 | 157 | 136 | 6 | **NO** |
| 4 | ObjectDraw.c:373 | ObjectDraw.c:382 | 7 | 52 | #CC9933 | 157 | 136 | 6 | **NO** |
| 5 | ObjectDraw.c:662 | ObjectDraw.c:670 | 7 | 52 | #CC9933 | 157 | 136 | 6 | **NO** |
| 6 | ObjectDraw.c:82 | ObjectDraw.c:88 | 6 | 53 | #CC9900 | 152 | 153 | 6 | yes |
| 7 | ObjectDraw.c:788 | ObjectDraw.c:794 | 6 | 53 | #CC9900 | 152 | 153 | 6 | yes |
| 8 | ObjectDraw.c:162 | ObjectDraw.c:169 | 9 | 94 | #996633 | 111 | 102 | 8 | **NO** |
| 9 | ObjectDraw.c:280 | ObjectDraw.c:288 | 9 | 94 | #996633 | 111 | 102 | 8 | **NO** |
| 10 | ObjectDraw.c:374 | ObjectDraw.c:383 | 9 | 94 | #996633 | 111 | 102 | 8 | **NO** |
| 11 | ObjectDraw.c:516 | ObjectDraw.c:524 | 9 | 94 | #996633 | 111 | 102 | 8 | **NO** |
| 12 | ObjectDraw2.c:1050 | ObjectDraw2.c:1056 | 9 | 94 | #996633 | 111 | 102 | 8 | **NO** |
| 13 | ObjectDraw.c:83 | ObjectDraw.c:89 | 9 | 95 | #996600 | 106 | 102 | 9 | yes |
| 14 | ObjectDraw.c:659 | ObjectDraw.c:667 | 9 | 95 | #996600 | 106 | 102 | 9 | yes |
| 15 | ObjectDraw2.c:128 | ObjectDraw2.c:134 | 9 | 95 | #996600 | 106 | 102 | 9 | yes |
| 16 | ObjectDraw2.c:211 | ObjectDraw2.c:217 | 9 | 95 | #996600 | 106 | 102 | 9 | yes |
| 17 | ObjectDraw.c:161 | ObjectDraw.c:168 | 11 | 137 | #663300 | 60 | 68 | 11 | yes |
| 18 | ObjectDraw.c:278 | ObjectDraw.c:286 | 11 | 137 | #663300 | 60 | 68 | 11 | yes |
| 19 | ObjectDraw.c:514 | ObjectDraw.c:522 | 11 | 137 | #663300 | 60 | 68 | 11 | yes |
| 20 | ObjectDraw.c:660 | ObjectDraw.c:668 | 11 | 137 | #663300 | 60 | 68 | 11 | yes |
| 21 | ObjectDraw.c:789 | ObjectDraw.c:795 | 11 | 137 | #663300 | 60 | 68 | 11 | yes |
| 22 | ObjectDraw2.c:129 | ObjectDraw2.c:135 | 11 | 137 | #663300 | 60 | 68 | 11 | yes |
| 23 | ObjectDraw2.c:212 | ObjectDraw2.c:218 | 11 | 137 | #663300 | 60 | 68 | 11 | yes |
| 24 | ObjectDraw2.c:1049 | ObjectDraw2.c:1055 | 11 | 137 | #663300 | 60 | 68 | 11 | yes |
| 24a | ObjectDraw.c:371 | ObjectDraw.c:380 | 11 | 137 | #663300 | 60 | 68 | 11 | yes |
| 25 | ObjectDraw2.c:1110 | ObjectDraw2.c:1112 | 5 | 150 | #33CCFF | 162 | 170 | 5 | yes |
| 26 | ObjectDraw2.c:1120 | ObjectDraw2.c:1122 | 5 | 150 | #33CCFF | 162 | 170 | 5 | yes |
| 27 | ObjectDraw.c:163 | ObjectDraw.c:170 | 14 | 223 | #220000 | 10 | 17 | 14 | yes |
| 28 | ObjectDraw.c:281 | ObjectDraw.c:289 | 14 | 223 | #220000 | 10 | 17 | 14 | yes |
| 29 | ObjectDraw.c:375 | ObjectDraw.c:384 | 14 | 223 | #220000 | 10 | 17 | 14 | yes |
| 30 | ObjectDraw.c:518 | ObjectDraw.c:526 | 15 | 223 | #220000 | 10 | 0 | 14 | **NO** |
| 31 | ObjectDraw.c:663 | ObjectDraw.c:671 | 15 | 223 | #220000 | 10 | 0 | 14 | **NO** |
| 32 | ObjectDraw2.c:1051 | ObjectDraw2.c:1057 | 15 | 223 | #220000 | 10 | 0 | 14 | **NO** |
| 33 | ObjectDraw.c:578 | ObjectDraw.c:597 | 1 | 245 | #EEEEEE | 236 | 238 | 1 | yes |
| 34 | ObjectDraw.c:592 | ObjectDraw.c:611 | 1 | 245 | #EEEEEE | 236 | 238 | 1 | yes |
| 35 | ObjectDraw2.c:436 | ObjectDraw2.c:444 | 1 | 245 | #EEEEEE | 236 | 238 | 1 | yes |
| 36 | ObjectDraw.c:580 | ObjectDraw.c:599 | 2 | 246 | #DDDDDD | 220 | 221 | 2 | yes |
| 37 | ObjectDraw.c:584 | ObjectDraw.c:603 | 4 | 247 | #BBBBBB | 186 | 187 | 4 | yes |
| 38 | ObjectDraw2.c:435 | ObjectDraw2.c:443 | 4 | 247 | #BBBBBB | 186 | 187 | 4 | yes |
| 39 | ObjectDraw2.c:513 | ObjectDraw2.c:520 | 4 | 247 | #BBBBBB | 186 | 187 | 4 | yes |
| 40 | ObjectDraw.c:586 | ObjectDraw.c:605 | 5 | 248 | #AAAAAA | 170 | 170 | 5 | yes |
| 41 | ObjectDraw.c:588 | ObjectDraw.c:607 | 5 | 248 | #AAAAAA | 170 | 170 | 5 | yes |
| 42 | ObjectDraw.c:590 | ObjectDraw.c:609 | 5 | 248 | #AAAAAA | 170 | 170 | 5 | yes |
| 43 | ObjectDraw2.c:434 | ObjectDraw2.c:442 | 5 | 248 | #AAAAAA | 170 | 170 | 5 | yes |
| 44 | ObjectDraw2.c:433 | ObjectDraw2.c:441 | 7 | 249 | #888888 | 134 | 136 | 7 | yes |
| 45 | ObjectDraw2.c:511 | ObjectDraw2.c:518 | 7 | 249 | #888888 | 134 | 136 | 7 | yes |
| 46 | ObjectDraw2.c:512 | ObjectDraw2.c:519 | 8 | 251 | #555555 | 84 | 119 | 10 | **NO** |
| 47 | Scoreboard.c:144 | Scoreboard.c:146 | 10 | 251 | #555555 | 84 | 85 | 10 | yes |
| 48 | Scoreboard.c:202 | Scoreboard.c:204 | 10 | 251 | #555555 | 84 | 85 | 10 | yes |
| 49 | Scoreboard.c:240 | Scoreboard.c:242 | 10 | 251 | #555555 | 84 | 85 | 10 | yes |
| 50 | Scoreboard.c:277 | Scoreboard.c:279 | 10 | 251 | #555555 | 84 | 85 | 10 | yes |
| 51 | Scoreboard.c:312 | Scoreboard.c:314 | 10 | 251 | #555555 | 84 | 85 | 10 | yes |
| 52 | ObjectDraw2.c:514 | ObjectDraw2.c:521 | 11 | 252 | #444444 | 66 | 68 | 11 | yes |
| 53 | ObjectDraw.c:898 | ObjectDraw.c:903 | 13 | 253 | #222222 | 33 | 34 | 13 | yes |
| 54 | ObjectDraw2.c:127 | ObjectDraw2.c:133 | 13 | 253 | #222222 | 33 | 34 | 13 | yes |
| 55 | ObjectDraw2.c:210 | ObjectDraw2.c:216 | 13 | 253 | #222222 | 33 | 34 | 13 | yes |
| 56 | ObjectDraw2.c:993 | ObjectDraw2.c:997 | 13 | 253 | #222222 | 33 | 34 | 13 | yes |
| 57 | ObjectDraw.c:81 | ObjectDraw.c:87 | 14 | 254 | #111111 | 16 | 17 | 14 | yes |
| 58 | ObjectDraw.c:372 | ObjectDraw.c:381 | 14 | 254 | #111111 | 16 | 17 | 14 | yes |
| 59 | ObjectDraw.c:515 | ObjectDraw.c:523 | 14 | 254 | #111111 | 16 | 17 | 14 | yes |
| 60 | ObjectDraw.c:661 | ObjectDraw.c:669 | 14 | 254 | #111111 | 16 | 17 | 14 | yes |
| 61 | ObjectDraw.c:790 | ObjectDraw.c:796 | 14 | 254 | #111111 | 16 | 17 | 14 | yes |
| 62 | ObjectDraw.c:899 | ObjectDraw.c:904 | 14 | 254 | #111111 | 16 | 17 | 14 | yes |
| 63 | ObjectDraw.c:185 | ObjectDraw.c:187 | 15 | 254 | #111111 | 16 | 0 | 14 | **NO** |
| 64 | ObjectDraw.c:310 | ObjectDraw.c:312 | 15 | 254 | #111111 | 16 | 0 | 14 | **NO** |
| 65 | ObjectDraw.c:405 | ObjectDraw.c:407 | 15 | 254 | #111111 | 16 | 0 | 14 | **NO** |
| 66 | ObjectDraw.c:547 | ObjectDraw.c:549 | 15 | 254 | #111111 | 16 | 0 | 14 | **NO** |
| 67 | ObjectDraw.c:692 | ObjectDraw.c:694 | 15 | 254 | #111111 | 16 | 0 | 14 | **NO** |
| 68 | ObjectDraw.c:164 | ObjectDraw.c:171 | 15 | 255 | #000000 | 0 | 0 | 15 | yes |
| 69 | ObjectDraw.c:282 | ObjectDraw.c:290 | 15 | 255 | #000000 | 0 | 0 | 15 | yes |
| 70 | ObjectDraw.c:376 | ObjectDraw.c:385 | 15 | 255 | #000000 | 0 | 0 | 15 | yes |
| 71 | ObjectDraw.c:517 | ObjectDraw.c:525 | 15 | 255 | #000000 | 0 | 0 | 15 | yes |

All paths are relative to `GliderPRO/Sources/`. `lum` is the game's own luminance formula
(§3.7 R1) applied to the 8-bit channel values. This is a strict superset of the 20 rows
`docs/analysis/rendering.md` §22.4 measured; the two extra combos it did not have are
`(2, 246)` and `(3, 43)`.

Row **24a** (`ObjectDraw.c:371` / `:380`, `brownC`) was missing from the first draft of this table,
which is why earlier revisions said "71 sites" and gave `(11, 137)` 8 sites: the correct totals are
**72 sites** and 9 sites for `(11, 137)`. Re-derived by re-extracting every depth conditional from
scratch; the site count per file is ObjectDraw.c 46, ObjectDraw2.c 21, Scoreboard.c 5.

## 3.6 The 22 unique combinations

| i4 | i8 | sites | 8-bit RGB | 16-bit RGB | pure gray? | lum | (15-i4)*17 | nearest | `gray8[i4]` | identity hit? |
|---|---|---|---|---|---|---|---|---|---|---|
| 3 | 42 | 1 | #CCCCFF | CCCC CCCC FFFF | no | 208 | 204 | 3 | 43 | - |
| 3 | 43 | 1 | #CCCCCC | CCCC CCCC CCCC | **yes** | 203 | 204 | 3 | 43 | **yes** |
| 7 | 52 | 3 | #CC9933 | CCCC 9999 3333 | no | 157 | 136 | 6 | 249 | - |
| 6 | 53 | 2 | #CC9900 | CCCC 9999 0000 | no | 152 | 153 | 6 | 86 | - |
| 9 | 94 | 5 | #996633 | 9999 6666 3333 | no | 111 | 102 | 8 | 129 | - |
| 9 | 95 | 4 | #996600 | 9999 6666 0000 | no | 106 | 102 | 9 | 129 | - |
| 11 | 137 | 9 | #663300 | 6666 3333 0000 | no | 60 | 68 | 11 | 252 | - |
| 5 | 150 | 2 | #33CCFF | 3333 CCCC FFFF | no | 162 | 170 | 5 | 248 | - |
| 14 | 223 | 3 | #220000 | 2222 0000 0000 | no | 10 | 17 | 14 | 254 | - |
| 15 | 223 | 3 | #220000 | 2222 0000 0000 | no | 10 | 0 | 14 | 255 | - |
| 1 | 245 | 3 | #EEEEEE | EEEE EEEE EEEE | **yes** | 236 | 238 | 1 | 245 | **yes** |
| 2 | 246 | 1 | #DDDDDD | DDDD DDDD DDDD | **yes** | 220 | 221 | 2 | 246 | **yes** |
| 4 | 247 | 3 | #BBBBBB | BBBB BBBB BBBB | **yes** | 186 | 187 | 4 | 247 | **yes** |
| 5 | 248 | 4 | #AAAAAA | AAAA AAAA AAAA | **yes** | 170 | 170 | 5 | 248 | **yes** |
| 7 | 249 | 2 | #888888 | 8888 8888 8888 | **yes** | 134 | 136 | 7 | 249 | **yes** |
| 8 | 251 | 1 | #555555 | 5555 5555 5555 | **yes** | 84 | 119 | 10 | 250 | no |
| 10 | 251 | 5 | #555555 | 5555 5555 5555 | **yes** | 84 | 85 | 10 | 251 | **yes** |
| 11 | 252 | 1 | #444444 | 4444 4444 4444 | **yes** | 66 | 68 | 11 | 252 | **yes** |
| 13 | 253 | 4 | #222222 | 2222 2222 2222 | **yes** | 33 | 34 | 13 | 253 | **yes** |
| 14 | 254 | 6 | #111111 | 1111 1111 1111 | **yes** | 16 | 17 | 14 | 254 | **yes** |
| 15 | 254 | 5 | #111111 | 1111 1111 1111 | **yes** | 16 | 0 | 14 | 255 | no |
| 15 | 255 | 4 | #000000 | 0000 0000 0000 | **yes** | 0 | 0 | 15 | 255 | **yes** |

Totals: 22 unique combos, **72** call sites, 13 pure-gray combos, 11 identity hits.

## 3.7 Nine candidate derivation rules, scored

The question "could a Go port compute `i4` from `i8` instead of hard-coding the 22 pairs?" is
answerable: score every plausible rule against the authored data.

The game's own luminance formula is recoverable from a **dead** function:

```c
// GliderPRO/Sources/MainWindow.c:454-498   SetPaletteToGrays
// ...but the whole function sits inside a /* ... */ block comment spanning lines 453-499,
// so it is not even compiled; its one call site (MainWindow.c:253) is commented out too.
	for (i = 0; i < 256; i++)
	{
		wasColors[i] = (*theCTab)->ctTable[i];
		newColors[i] = (*theCTab)->ctTable[i];

		if (i != 5)                         // index 5 (#FFFF00) is deliberately left alone
		{
			longGray = ((long)newColors[i].rgb.red * 3L) / 10L +
					((long)newColors[i].rgb.green * 6L) / 10L +
					((long)newColors[i].rgb.blue * 1L) / 10L;

			newColors[i].rgb.red = (unsigned short)longGray;
			newColors[i].rgb.green = (unsigned short)longGray;
			newColors[i].rgb.blue = (unsigned short)longGray;
		}
	}
	...
	SetEntries(0, 255, newColors);
```

Note the shape: **three separate integer divisions**, not one division of a summed product.
`(r*3)/10 + (g*6)/10 + (b*1)/10` differs from `(3r + 6g + b)/10` by up to 2 units (verified: the
maximum discrepancy over all 256 palette entries, 8-bit channels, is exactly 2). A port that
"simplifies" this changes results.

Two caveats about using it as the prior:

* The weights are applied to `RGBColor` fields, i.e. to **16-bit** channels, and the result is
  stored back as a 16-bit channel. R1 below re-applies the same weights to 8-bit channels, which is
  a deliberate (and, on 9 palette entries, visible) reinterpretation - see the R1/R9 bullet.
* The function is doubly dead: commented out *and* never called. It is still the only statement of
  Calhoun's own luminance weights anywhere in the tree, so it remains the right prior, but no
  shipped pixel was ever produced by it.

Rules scored against the 22 combos / 72 sites / 13 pure-gray combos:

| rule | unique combos hit (of 22) | call sites hit (of 72) | pure-gray combos hit (of 13) |
|---|---|---|---|
| **R1** nearest level, `lum = (r*3)/10 + (g*6)/10 + (b*1)/10` (8-bit channels) | **17** | **55** | **11** |
| R2 nearest level, `lum` = NTSC 0.299/0.587/0.114 | 17 | 55 | 11 |
| R3 nearest level, `lum = (r+g+b)/3` | 15 | 47 | 11 |
| R4 nearest level, `lum = max(r,g,b)` | 11 | 34 | 11 |
| R5 nearest level, `lum = (max+min)/2` | 13 | 42 | 11 |
| R6 truncate: `i4 = 15 - lum/17` | 7 | 26 | 3 |
| R7 round: `i4 = 15 - (lum+8)/17` | 17 | 55 | 11 |
| R8 nearest level, `lum = min(r,g,b)` | 13 | 38 | 11 |
| R9 nearest level, 16-bit `lum = (3R + 6G + B)/10` | 17 | 55 | 11 |

Findings:

* R1, R2, R7 and R9 are tied at the top and are **behaviourally identical on this corpus**.
  R1 and R7 are in fact identical on all 256 palette entries, not just on the corpus:
  `15 - (lum+8)/17` *is* nearest-level rounding for every integer `lum` in 0..255, so they are two
  spellings of one rule.
  R1 and R9 (8-bit vs 16-bit channel arithmetic) disagree on **9 of the 256 palette entries** -
  indices 5 `#FFFF00`, 15 `#FF9966`, 38 `#CCFF99`, 48 `#CC99FF`, 81 `#99CC66`, 84 `#9999FF`,
  114 `#66CCFF`, 147 `#33FF66` and 180 `#00FFFF` - because truncating each channel to 8 bits before
  weighting loses up to a level of luminance. (At index 84 the 8-bit rule picks `i4` = 6 and the
  16-bit rule picks 5.) None of the 9 is among the 22 authored combos, so the corpus cannot
  discriminate; use the 8-bit form (R1), which matches the game's own integer style, but be aware
  the choice is visible on those 9 colours. The three-division and single-division 16-bit variants
  agree with each other everywhere.
* **R6 (truncation) is decisively wrong** - 7/22, 26/72, 3/13. Do not truncate; round. This is
  the single most likely accidental error in a port.
* No rule reaches 22/22. Four combos are unreachable by *any* luminance rule (§3.8), so the
  authored pairs must be honoured literally where they exist.

## 3.8 The five R1 failures, individually

| i4 (authored) | i8 | 8-bit RGB | lum | R1 predicts | sites | explanation |
|---|---|---|---|---|---|---|
| 9 | 94 | #996633 | 111 | 8 | 5 | Wood brown. Author chose one step darker than luminance says. Consistent across 5 sites, so deliberate. |
| 15 | 254 | #111111 | 16 | 14 | 5 | Inline drop-shadow sites (`ObjectDraw.c:185, 310, 405, 547, 692`). Author wrote pure black where the 8-bit twin is `#111111`. |
| 7 | 52 | #CC9933 | 157 | 6 | 3 | Light wood. One step darker than luminance. Same object family as row 1, same direction. |
| 15 | 223 | #220000 | 10 | 14 | 3 | Same shape as row 2: near-black rendered as black. |
| 8 | 251 | #555555 | 84 | 10 | 1 | `ObjectDraw2.c:512`. A pure gray whose exact ramp index is 10; author wrote 8. Single site, almost certainly a slip. |

Pattern: **all five failures move `i4` in the *darker* direction** relative to R1 except the
`(8, 251)` slip, which moves lighter. The three "darker" families are the wood objects
(rows 1 and 3) and the near-black shadows (rows 2 and 4), i.e. deliberate contrast boosting for
a 16-level display.

## 3.9 The map is not a function

Three 8-bit indices are authored with two different 4-bit partners:

```
i8 223 -> i4 {14, 15}
i8 251 -> i4 { 8, 10}
i8 254 -> i4 {14, 15}
```

and correspondingly:

```
i4  3 -> i8 {42, 43}
i4  5 -> i8 {150, 248}
i4  7 -> i8 { 52, 249}
i4  9 -> i8 { 94,  95}
i4 11 -> i8 {137, 252}
i4 14 -> i8 {223, 254}
i4 15 -> i8 {223, 254, 255}
```

**Therefore no `i8 -> i4` table can reproduce the original**; the choice is per-call-site. A Go
port must either (a) carry the 33 conditional sites verbatim, keyed on the source line, or
(b) accept the 17-site deviation implied by picking any single rule. R-CLUT4-2 (§3.14)
recommends (a) as the default with (b) as the fallback for any site not in the table.

## 3.10 The `clut` resources in the file are never used

Complete inventory: exactly **two** `clut` resources exist, IDs 128 and 129. Both parse as:

```
length 2056 bytes
ctSeed  = 0x00000000
ctFlags = 0x0000
ctSize  = 255            (count - 1, so 256 ColorSpec entries)
entries = 256 * 8 bytes  (ColorSpec = short value; RGBColor rgb  -> 2 + 6 = 8 bytes)
```

and they are **byte-identical to each other**. `ColorSpec.value` equals the entry index for
every one of the 256 entries in both.

**`GetCTable` call count across the whole tree: 0.** Neither resource is referenced by any code.
They are era-standard leftovers (a `clut` is what `PICT`/`GWorld` machinery consumed
implicitly). They *are* useful to a port as the authoritative statement of the standard 256-entry
palette - that is how §3.4's `gray8[]` was derived - but they are not the 4-bit CLUT, and the
4-bit CLUT **is not in the application at all**. It comes from the system, which is why R-CLUT4-1
has to be a normative substitution.

Spot-check of the entries this document depends on:

```
  idx   0 value=  0 rgb16=FFFF FFFF FFFF  #FFFFFF     (white, bg)
  idx   1 value=  1 rgb16=FFFF FFFF CCCC  #FFFFCC
  idx  18 value= 18 rgb16=FFFF 6666 FFFF  #FF66FF     (magenta - see 3.12)
  idx  23 value= 23 rgb16=FFFF 6666 0000  #FF6600     (kRedOrangeColor8)
  idx  43 value= 43 rgb16=CCCC CCCC CCCC  #CCCCCC
  idx  86 value= 86 rgb16=9999 9999 9999  #999999
  idx 129 value=129 rgb16=6666 6666 6666  #666666
  idx 172 value=172 rgb16=3333 3333 3333  #333333
  idx 245 value=245 rgb16=EEEE EEEE EEEE  #EEEEEE
  idx 246 value=246 rgb16=DDDD DDDD DDDD  #DDDDDD
  idx 247 value=247 rgb16=BBBB BBBB BBBB  #BBBBBB
  idx 248 value=248 rgb16=AAAA AAAA AAAA  #AAAAAA
  idx 249 value=249 rgb16=8888 8888 8888  #888888
  idx 250 value=250 rgb16=7777 7777 7777  #777777
  idx 251 value=251 rgb16=5555 5555 5555  #555555     (kGrayBackgroundColor)
  idx 252 value=252 rgb16=4444 4444 4444  #444444
  idx 253 value=253 rgb16=2222 2222 2222  #222222
  idx 254 value=254 rgb16=1111 1111 1111  #111111
  idx 255 value=255 rgb16=0000 0000 0000  #000000     (black, fg)
```

Note the byte-doubling: an 8-bit channel `0xCC` is stored `0xCCCC`, never `0xCC00`. A port that
scales with `<<8` instead of `*0x0101` will be off by one step on every non-zero channel and
will break the `gray8[]` identity test.

## 3.11 Unguarded indices > 15 - undefined behaviour at depth 4

These eight sites pass a bare integer **literal** > 15 to a `ColorXxx` helper with **no**
`isDepth == 4` guard. At depth 4 the device CLUT has 16 entries, so `Index2Color` reads past its
end:

| # | Site | index | 8-bit colour | what it draws |
|---|---|---|---|---|
| 1 | `GliderPRO/Sources/GameOver.c:65` | 244 | `#000011` | game-over page tint |
| 2 | `GliderPRO/Sources/GameOver.c:84` | 244 | `#000011` | game-over page tint |
| 3 | `GliderPRO/Sources/ObjectDraw.c:128` | 192 | `#0099FF` | object detail |
| 4 | `GliderPRO/Sources/ObjectDraw.c:141` | 192 | `#0099FF` | object detail |
| 5 | `GliderPRO/Sources/ObjectDraw.c:1390` | 227 | `#00BB00` | object detail |
| 6 | `GliderPRO/Sources/ObjectDraw2.c:295` | 32 | `#FF0099` | object detail |
| 7 | `GliderPRO/Sources/ObjectDraw2.c:593` | 17 | `#FF9900` | object detail |
| 8 | `GliderPRO/Sources/Tools.c:154` | 171 | `#333366` | tool palette label |

Bare literals are only the visible tip. Counted mechanically over all 211 `ColorXxx` call sites
(§3.15), **43** pass a constant > 15 with no depth guard: the 8 raw literals above plus 35 named
constants (`k8DkGrayColor` 252 x8, `k8LtGrayColor` 249 x7, `kRedOrangeColor8` 23 x5,
`k8EarthBlueColor` 170 x2, `k8Red4Color` 143 x2, and singletons `k8GrayColor`, `k8Gray2Color`,
`k8DkGray3Color`, `k8DkstGrayColor`, `k8BrownColor`, `k8RedColor`, `k8OrangeColor`,
`k8PumpkinColor`, `k8PissYellowColor`, `kIntenseGreenColor`, `kIntenseBlueColor`,
`kDarkFleshColor`, `k8LtstGray*`). On top of that, **26** `FrameDialogItemC(dialog, item,
kRedOrangeColor8)` calls (§3.12) go through the same `Index2Color` path in
`GliderPRO/Sources/DialogUtils.c:706-719` without being `ColorXxx` calls at all.

**Total unguarded out-of-range draws: 69** (43 + 26).

What the original actually did: `Index2Color` on an out-of-range index against a 16-entry table
returns whatever memory follows the table. In practice on a 4-bit grayscale device the Color
Manager clamps or wraps depending on the exact system version, so 1994 behaviour was
system-dependent - i.e. **the original game's depth-4 appearance was not itself well-defined at
these 69 sites.**

**Port rule:** clamp any index >= 16 at depth 4 to the R1-derived nearest gray level. That is
defined, stable, and visually sensible. Document that it is a deliberate divergence.

This is also the strongest argument for R-CLUT4-2 (§3.14): if you render at 8 bits and
post-map, the problem evaporates - all 69 sites resolve normally in the 256-entry palette and
then get mapped down.

## 3.12 `kRedOrangeColor8`

```c
// GliderPRO/Headers/GliderDefines.h:542
#define kRedOrangeColor8        23      // actually, 18
```

The comment is **wrong**. Verified from `clut` 128:

* index **23** = `#FF6600` - a genuine red-orange.
* index **18** = `#FF66FF` - magenta.

Use 23. The constant appears at 31 sites, of which **26** are
`FrameDialogItemC(theDialog, item, kRedOrangeColor8)` (dialog-item outlines in `ObjectInfo.c` x16,
`Settings.c` x6, `RoomInfo.c` x2, `House.c`, `HouseInfo.c`), 4 are `ColorFrameRect`
(`Settings.c:1297-1300`, the four preference buttons) and 1 is `ColorFrameWHRect`
(`GliderPRO/Sources/SelectHouse.c:80`):

```c
	ColorFrameWHRect(8, 39, 413, 184, kRedOrangeColor8);	// box around files
```

which frames the 12-house file list in the Load House dialog (`kLoadHouseDialogID` = 1000,
`kDispFiles` = 12, `SelectHouse.c:20-21`).

## 3.13 The five depth-4 sites that are not about colour

Not every `isDepth == 4` branch is a palette substitution. The five in `DynamicMaps.c` adjust
**byte alignment**, because a 4-bit pixmap packs 2 pixels per byte and QuickDraw blits are
faster (and, in the original, correct) on byte boundaries:

| Site | Registrar | Effect |
|---|---|---|
| `GliderPRO/Sources/DynamicMaps.c:326` | `AddCandleFlame` | `if (isDepth == 4 && (src.left % 2) == 1)` -> decrement, i.e. round the source left edge down to an even column |
| `GliderPRO/Sources/DynamicMaps.c:409` | `AddTikiFlame` | `if (isDepth == 4 && (h % 2) == 1)` -> `h--` |
| `GliderPRO/Sources/DynamicMaps.c:495` | `AddBBQCoals` | same |
| `GliderPRO/Sources/DynamicMaps.c:584` | `AddPendulum` | same |
| `GliderPRO/Sources/DynamicMaps.c:671` | `AddStar` | same (this is the **star** registrar, not a flower one) |

Each of the four `h`-based sites is written `h--; if (h < 0) h += 2;`, and the `h < 0` arm is
unreachable: `h % 2 == 1` implies `h >= 1`, so `h--` cannot go negative. A port may drop it.

Net visible effect: **the flame/coal/pendulum/star sprites sit 1 pixel to the left at depth 4
relative to depth 8 whenever their `h` is odd.** A port that renders internally at 8 bits (R-CLUT4-2) should *not* apply these nudges,
and should document the 1-pixel divergence.

## 3.14 R-CLUT4-2 - render at 8, derive 4

Normative architecture for the Go port:

1. All rendering happens into an 8-bit indexed framebuffer against the standard 256-entry
   palette (the content of `clut` 128, reproduced in §3.10).
2. Palette indices are resolved with an `Index2Color` equivalent that reads the 256-entry table
   unconditionally. There is no per-device CLUT in the port's renderer.
3. When the user selects "16 grays" (`kSwitchTo16Grays` = 2), the port keeps rendering at 8 bits
   and applies a **display-time** post-map:
   * For any pixel written by one of the 33 authored `isDepth == 4` sites, use the authored `i4`
     from the §3.5 table (keyed by draw call, not by resulting index).
   * For every other pixel, use R1: `i4 = argmin_j | lum(palette[i8]) - (15-j)*17 |` with
     `lum(r,g,b) = (r*3)/10 + (g*6)/10 + (b*1)/10` on 8-bit channels, ties to the lower `j`.
   * Emit `clut4[i4]` from R-CLUT4-1 as the displayed grey.
4. The 69 unguarded-index sites (§3.11) fall out of step 3's second bullet automatically.
5. Do not apply the `DynamicMaps.c` byte-alignment nudges (§3.13).

Divergences this introduces, enumerated: 17 of 72 authored sites would differ if you skipped
step 3's first bullet; the 69 unguarded sites become defined where they were not; five sprites
lose a 1-pixel depth-4 shift. Nothing else.

## 3.15 Colour-constant inventory

Constants defined in each file's colour block (all are file-local `#define`s; there is no shared
colour header):

| File | Lines | Count |
|---|---|---|
| `GliderPRO/Sources/ObjectDraw.c` | 16-48 | 33 defines |
| `GliderPRO/Sources/ObjectDraw2.c` | 19-37 | 19 defines |
| `GliderPRO/Sources/Scoreboard.c` | 15-21 | 7 defines (only the first two are colours) |

`Scoreboard.c`'s seven, verbatim values:

```c
// GliderPRO/Sources/Scoreboard.c:15-21
#define kGrayBackgroundColor    251     // #555555  (8-bit)
#define kGrayBackgroundColor4    10     //          (4-bit)  -> exact gray8[10] hit
#define kFoilBadge                0
#define kBandsBadge               1
#define kBatteryBadge             2
#define kHeliumBadge              3
#define kScoreRollAmount         13
```

`ColorXxx` call-site totals across the tree (these are the functions whose `color` argument is
depth-sensitive):

| Helper | Definition | Call sites |
|---|---|---|
| `ColorLine` | `ColorUtils.c:84-94` | 153 |
| `ColorRect` | `ColorUtils.c:36-45` | 22 |
| `ColorFrameRect` | `ColorUtils.c:120-129` | 17 |
| `ColorRegion` | `ColorUtils.c:68-77` | 8 |
| `ColorText` | `ColorUtils.c:20-29` | 5 |
| `ColorOval` | `ColorUtils.c:52-61` | 3 |
| `ColorFrameOval` | `ColorUtils.c` | 2 |
| `ColorFrameWHRect` | `ColorUtils.c:137+` | 1 |
| **total** | | **211** |

The 211 total counts every call, including 5 calls made *inside* `ColorUtils.c` itself
(`ColorLine` x4 from the shadow-box helper at `:105-111`, and `ColorFrameRect` from
`ColorFrameWHRect` at `:145`); the eight function definitions are excluded. External call sites
alone number 206.

Classifying all 211 by what their `color` argument is (mechanical, then spot-checked):

| Class | Count | Note |
|---|---|---|
| a depth-selected local (`brownC`, `tanC`, `dkGrayC`, ...) assigned by a §3.5 pair | 112 | depth-safe |
| lexically inside an `isDepth == 4` if/else | 32 | 30 of these are the 15 `ColorOval`/`ColorRegion`/`ColorLine`/`ColorRect` pairs in the §3.5 table; the other 2 are `MainWindow.c:78-79` (`ColorText(..., 5L)` / `ColorText(..., 28L)`) |
| an unguarded constant > 15 | 43 | §3.11 - undefined at depth 4 |
| a constant in 0..15 | 19 | valid at both depths (e.g. `Banner.c:229` uses index 4) |
| a parameter of another `ColorUtils.c` helper | 5 | the internal calls listed above |
| **total** | **211** | |

Note that this is *not* the same partition as "72 authored pairs": most of the 72 pairs are
assignments to a local, and each such local is then used by one or more of the 112 calls in row 1,
while 5 of the 72 pairs are `Index2Color` calls in `Scoreboard.c` that are not `ColorXxx` calls at
all. Earlier revisions of this section stated "71 / 39 / 101", which conflated the two censuses.

## 3.16 Go implementation sketch

```go
// clut4 is R-CLUT4-1: the 16-entry system grayscale device CLUT.
var clut4 [16]RGB16
func init() {
	for i := 0; i < 16; i++ {
		lv := uint16((15 - i) * 0x1111)
		clut4[i] = RGB16{lv, lv, lv}
	}
}

// gray8 maps a 4-bit gray level to the single 8-bit palette index with the same RGB.
// Verified against clut 128: exactly one hit per level, all 16 distinct.
var gray8 = [16]uint8{0, 245, 246, 43, 247, 248, 86, 249, 250, 129, 251, 252, 172, 253, 254, 255}

// lumGlider is Calhoun's own weighting from the dead SetPaletteToGrays
// (GliderPRO/Sources/MainWindow.c:454-498). THREE separate integer divisions -
// do not algebraically simplify, the truncation is observable.
func lumGlider(r, g, b uint8) int {
	return int(r)*3/10 + int(g)*6/10 + int(b)*1/10
}

// index8To4 is rule R1: nearest gray level, ties to the lighter (lower) index.
// Scores 17/22 combos, 55/72 sites, 11/13 pure grays against the authored data.
func index8To4(i8 uint8) uint8 {
	r, g, b := palette8[i8].RGB8()
	l := lumGlider(r, g, b)
	best, bestErr := uint8(0), 1<<30
	for j := 0; j < 16; j++ {
		e := l - (15-j)*17
		if e < 0 {
			e = -e
		}
		if e < bestErr { // strict <, so ties keep the lower (lighter) j
			best, bestErr = uint8(j), e
		}
	}
	return best
}
```

Explicitly do not write `15 - l/17`: that is rule R6, which scores 7/22 and 26/72.

---

# Part 4 - Chicago 12 pt and Geneva 9 / 12 / 14 pt

## 4.1 No font resources exist

Searched all 538 resource records in `GliderPRO/Glider PRO.r` for every font-bearing type:

| Type | Meaning | Count |
|---|---|---|
| `FOND` | font family descriptor | **0** |
| `NFNT` | new bitmap font | **0** |
| `FONT` | old bitmap font | **0** |
| `sfnt` | TrueType outline | **0** |
| `fdsc`/`fmtx` | TrueType tables | **0** |

`Chicago` and `Geneva` were system fonts, shipped in the System file, not in applications.
**Glyph bitmaps are therefore unrecoverable in principle from this source release on an
airgapped machine.** Option (b) from the assignment - a named substitute plus measured deltas -
is the only available answer, and this part supplies it.

For completeness, the two font selectors the game uses:

| Constant | Value | 1994 resolution |
|---|---|---|
| `systemFont` | 0 | Chicago |
| `applFont` | 1 | Geneva (the user-selectable application font; Geneva on the era's Macs) |

## 4.2 Complete census of text-state calls

| Call | Occurrences | Detail |
|---|---|---|
| `TextFont` | **16** | 14 x `applFont`, 2 x `systemFont` |
| `TextSize` | **16** | **9 (x7), 12 (x8), 14 (x1)** - no other size anywhere |
| `TextFace` | **14** | `bold` x12 effective, `0`/plain x1, plus 1 stray `TextFace(applFont)` (§4.15) |
| `TextMode` | **0** | text mode is always the port default `srcOr` |
| `CharExtra` | **0** | |
| `SpaceExtra` | **0** | |
| `GetFontInfo` | **0** | |
| `FontMetrics` | **0** | |
| `GetFNum` | **0** | |
| `RealFont` | **0** | |
| `SetFractEnable` | **0** | |
| `TruncString` / `TruncText` | **0** | the game rolls its own, §4.14 |
| `StdTxMeas` / `MeasureText` | **0** | |
| `StringWidth` | **11** | the only horizontal measurement primitive used |
| `TextWidth` | **1** | `GliderPRO/Sources/GameOver.c:102` |
| `DrawString` | **49** | |
| `TETextBox` | **1** | `GliderPRO/Sources/DialogUtils.c:642` |
| `NumToString` | **72** | |
| `GetIndString` | **28** | |

**The single most important consequence: because `GetFontInfo`/`FontMetrics` are never called,
every vertical position in the entire game is a hard-coded baseline constant.** Ascent, descent
and leading of the substitute font are therefore irrelevant to layout *except* in the one
`TETextBox` call, which uses TextEdit's own line-height computation. All the port has to get
right is **horizontal advance widths**.

This corrects `docs/analysis/rendering.md` §23.6, which lists the sizes as "9, 10 and 12".
There is no size 10 anywhere; there is a size 14 (§4.4 row 4).

## 4.3 The port text-state model

QuickDraw text state lives in the `CGrafPort`, not in a global. A fresh port defaults to:

| Field | Default | Meaning |
|---|---|---|
| `txFont` | 0 | `systemFont` = Chicago |
| `txSize` | 0 | "use the font's default", i.e. 12 |
| `txFace` | 0 | plain |
| `txMode` | `srcOr` (1) | |

**State persists across calls and across `SetPort`/`SetGWorld` boundaries per port.** This is
why several of the game's draw functions rely on state set somewhere else entirely:

```c
// GliderPRO/Sources/StructuresInit.c:88-145 (excerpt, the pattern repeats per GWorld)
	hOffset = (RectWide(&houseRect) - 640) / 2;
	if (hOffset < 0)
		hOffset = -128;
	...
	QSetRect(&boardTSrcRect, 0, 0, 256, 12);
	...
	SetGWorld(boardTSrcMap, nil);
	...
	TextFont(applFont);
	TextSize(12);
	TextFace(bold);
```

Each scoreboard GWorld is created, made current, and then has Geneva 12 bold stamped onto its
port *once at init*; the per-frame `Scoreboard.c` draw functions then never set the font again.
A Go port with a stateless text API must carry the (font, size, face) triple per drawable and
initialise it in the same order, or the scoreboard will render in Chicago 12 plain.

Other rects established in the same block (with their comments preserved):

```
badgeSrcRect    0,0,32,66      // 2144 pixels
boardTSrcRect   0,0,256,12  -> QOffsetRect(&boardTDestRect, 137 + hOffset, 5)
boardGSrcRect   0,0,20,10   -> +526 + hOffset, +5
boardPSrcRect   0,0,64,10   -> +570 + hOffset, +5     // "total = 6396 pixels"
```

## 4.4 The four (font, size, face) tuples actually used

| key | Toolbox calls | 1994 font | Used for |
|---|---|---|---|
| **CHI12** | port default (`txFont` 0, `txSize` 0, `txFace` 0) | Chicago 12 plain | coordinate window (`Coordinates.c:61-110`), message window (`WindowUtils.c`), `"No rooms"` (`Room.c:259/261`), `"PowerPC Native!"` (`MainWindow.c:82-93`, drawn at `:88` and `:91`), and **every DITL item that has no `ictb` override** - i.e. all dialog text (§4.10) |
| **GEN9** | `TextFont(applFont); TextSize(9); TextFace(bold)` | Geneva 9 bold | splash "House:" line (`MainWindow.c:56-94`), tool names (`Tools.c:151-154`), calendar month (`ObjectDraw2.c:1157-1163`), high-score footer (`HighScores.c:267-272`) |
| **GEN12** | `TextFont(applFont); TextSize(12); TextFace(bold)` | Geneva 12 bold | scoreboard (all of it), banner (`Banner.c:113-236`), game-over trailer, high-score rows |
| **GEN14** | `TextFont(applFont); TextSize(14); TextFace(bold)` | Geneva 14 bold | exactly one site: the high-score page house title (state set at `HighScores.c:133-135`, drawn at `:142` and `:145`) |

`DrawDialogUserText` (`DialogUtils.c:616-648`) and `DrawDialogUserText2` (`:655-671`) each call
`TextFont(applFont); TextSize(9);` and **neither calls `TextFace` at all**, so they draw Geneva 9
in whatever face the dialog's port happens to be carrying - plain in practice, because nothing in
the dialog paths sets bold first. A port must therefore *not* assume GEN9-bold for these two; it
must model the inherited `txFace`. See §4.15.

The splash-screen block, verbatim, because it is the clearest example of the depth-4 /
colour-index interaction plus a PowerPC-conditional tail:

```c
// GliderPRO/Sources/MainWindow.c:56-94 (DrawOnSplash)
	TextSize(9);
	TextFace(1);
	TextFont(applFont);
	MoveTo(splashOriginH + 436, splashOriginV + 314);
	if (thisMac.isDepth == 4)
	{
		ForeColor(whiteColor);
		DrawString(houseLoadedStr);
		ForeColor(blackColor);
	}
	else
	{
		if (houseIsReadOnly)
			ColorText(houseLoadedStr, 5L);
		else
			ColorText(houseLoadedStr, 28L);
	}
#if defined(powerc) || defined(__powerc)
	TextSize(12);
	TextFace(0);
	TextFont(systemFont);
	... "\pPowerPC Native!" black at +5,+457 / white at +4,+456 ...
#endif
```

Note `TextFace(1)`: `bold` is 1, so this is bold written as a literal. Also note the order -
size, face, **then** font - which is harmless in QuickDraw but is the reason a naive
"set font then derive metrics" port can pick the wrong metrics if it caches eagerly.

## 4.5 R-FONT-1 - the substitutes

| key | Substitute file | ppem | ascent | descent | leading | lineHeight | inkAsc | inkDesc | digit advance | space |
|---|---|---|---|---|---|---|---|---|---|---|
| **CHI12** | `/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf` | 13 | 12 | 3 | 1 | 16 | 11 | 3 | 7 | 4 |
| **GEN9** | `/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf` | 9 | 9 | 2 | 0 | 11 | 8 | 2 | 5 | 2 |
| **GEN12** | `/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf` | 11 | 10 | 3 | 1 | 14 | 9 | 2 | 6 | 3 |
| **GEN14** | `/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf` | 13 | 12 | 3 | 1 | 16 | 11 | 3 | 7 | 4 |

Ascent, descent, ink extents and advances measured with
`PIL.ImageFont.truetype(...).getlength()` / `.getbbox()` / `.getmetrics()` at integer ppem with
hinting on, then rounded to whole pixels (QuickDraw bitmap fonts have integer advances;
fractional advances must be disabled - `SetFractEnable` is never called, so the original ran with
fractional widths off).

Three qualifications on that table, all of which a port must implement deliberately:

- **The rounding rule is round-half-to-even** (Python's `round`, i.e. IEEE 754 nearest-even), not
  round-half-up. This is not cosmetic: 40 of the 416 advances in §4.7-§4.10 land exactly on `.5`
  and change value under the other rule - 3 in CHI12 (`z`, `“`, `”`), **29 in GEN9** (including
  `space`, `!`, `,`, `.`, `/`, `:`, `;`, `I` at 2.5 and `C`, `D`, `H`, `J`, ... at x.5), 0 in
  GEN12, 8 in GEN14. A port that uses `floor(x+0.5)` will not reproduce the tables below, and its
  GEN9 strings will run wide by roughly one pixel per two characters.
- **`leading` and therefore `lineHeight` are *not* measurements of the substitute file.**
  Liberation Sans' `hhea.lineGap` is 67/2048 em, which rounds to **0** at all four ppems used
  here; `ascent + descent` from PIL is 15 / 11 / 13 / 15 for CHI12 / GEN9 / GEN12 / GEN14. The
  leading column (1 / 0 / 1 / 1) is the *classic Mac* `FOND` leading for Chicago 12, Geneva 9,
  Geneva 12 and Geneva 14, asserted so that `lineHeight` comes out at the 16 / 11 / 14 / 16 the
  original's own layout implies (§4.11). Keep the asserted values - they are what the DITL
  geometry demands - but do not describe them as measured.
- **The ink extents are ASCII-only** (0x21-0x7E). Adding the Mac Roman accented letters that
  really can appear in house and room names raises `inkAsc` by 2 in every one of the four keys
  (CHI12 11->13, GEN9 8->10, GEN12 9->11, GEN14 11->13) and raises GEN12's `inkDesc` from 2 to 3.
  That last one matters for the constraint in note 2 below.

Why these four, specifically:

1. **CHI12 must be bold-weight.** Chicago is a heavy display face; Liberation Sans *Regular* at
   any ppem is far too light and, critically, too narrow - 34 of the 230 measured DITL items
   would then be laid out with visibly loose tracking. Liberation Sans Bold at 13 px gives a
   16-px line height, which exactly matches the **76** DITL StaticText items whose box height is
   16 (§4.11) - i.e. the original's own statement of Chicago 12's line height.
2. **GEN12 at 11 ppem is forced by two independent hard constraints.** The scoreboard score
   field is `boardPSrcRect` = 64 x 10 px (`StructuresInit.c:122`) and holds up to 9 digits. With
   digit advance 6 plus synthetic bold's +1, 9 digits = 9 x 7 = **63 px** - fits with 1 px to
   spare, and 10 digits (70 px) overflows exactly as it does in the original. Separately,
   `boardTSrcRect` is 256 x 12 px (`StructuresInit.c:104`) and `RefreshRoomTitle`
   (`Scoreboard.c:136-186`) sets the black shadow copy's baseline at v = 10 (`MoveTo(1, 10)` at
   `:151`; the white copy is one pixel up and left at `MoveTo(0, 9)`, `:167`), so the font's ink
   descent must be <= 2; GEN12's ASCII inkDesc is exactly 2. At 12 ppem both constraints fail.
   Caveat: a room name containing an accented lower-case letter with a descending diacritic
   pushes GEN12's inkDesc to 3, which clips one row of that diacritic against the 12-px band -
   the original, with a real Geneva bitmap, would have clipped it too.
3. **GEN9 at 9 ppem** puts the 32-character high-score prompt at 150 px against a 272-px budget
   and the longest tool name at 96 px against 115 - comfortable - while keeping the 9-px
   drop-shadow offsets in `MainWindow.c`/`HighScores.c` visually correct.
4. **GEN14 at 13 ppem** is used at exactly one site and only needs to not overflow 352 px for
   real house names; `"• Demo House •"` measures 106 px.

## 4.6 Synthetic bold

QuickDraw does not have a bold Geneva bitmap at these sizes; it **synthesises** bold by drawing
each glyph twice, offset 1 pixel right, and adding **1 to the advance of every character**.

```
StringWidth_bold(s) = StringWidth_plain(s) + len(s)
```

This is exact, not approximate, and it is why the per-character tables below are given plain:
add `len(s)` for any bold string. The port must apply it per *character*, including spaces, and
must not apply it to the CHI12 tables (those are already measured from a bold-weight file, which
stands in for Chicago's inherent weight; Chicago has no separate synthesised-bold path in this
game because `TextFace(bold)` is never applied to `systemFont`).

## 4.7 CHI12 per-character advances (plain)

Add +1 per character for QuickDraw synthetic bold. Mac Roman code points.

| hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0x20 | `space` | 4 | 0x21 | `!` | 4 | 0x22 | `"` | 6 | 0x23 | `#` | 7 | 0x24 | `$` | 7 | 0x25 | `%` | 12 |
| 0x26 | `&` | 9 | 0x27 | `'` | 3 | 0x28 | `(` | 4 | 0x29 | `)` | 4 | 0x2A | `*` | 5 | 0x2B | `+` | 8 |
| 0x2C | `,` | 4 | 0x2D | `-` | 4 | 0x2E | `.` | 4 | 0x2F | `/` | 4 | 0x30 | `0` | 7 | 0x31 | `1` | 7 |
| 0x32 | `2` | 7 | 0x33 | `3` | 7 | 0x34 | `4` | 7 | 0x35 | `5` | 7 | 0x36 | `6` | 7 | 0x37 | `7` | 7 |
| 0x38 | `8` | 7 | 0x39 | `9` | 7 | 0x3A | `:` | 4 | 0x3B | `;` | 4 | 0x3C | `<` | 8 | 0x3D | `=` | 8 |
| 0x3E | `>` | 8 | 0x3F | `?` | 8 | 0x40 | `@` | 13 | 0x41 | `A` | 9 | 0x42 | `B` | 9 | 0x43 | `C` | 9 |
| 0x44 | `D` | 9 | 0x45 | `E` | 9 | 0x46 | `F` | 8 | 0x47 | `G` | 10 | 0x48 | `H` | 9 | 0x49 | `I` | 4 |
| 0x4A | `J` | 7 | 0x4B | `K` | 9 | 0x4C | `L` | 8 | 0x4D | `M` | 11 | 0x4E | `N` | 9 | 0x4F | `O` | 10 |
| 0x50 | `P` | 9 | 0x51 | `Q` | 10 | 0x52 | `R` | 9 | 0x53 | `S` | 9 | 0x54 | `T` | 8 | 0x55 | `U` | 9 |
| 0x56 | `V` | 9 | 0x57 | `W` | 12 | 0x58 | `X` | 9 | 0x59 | `Y` | 9 | 0x5A | `Z` | 8 | 0x5B | `[` | 4 |
| 0x5C | `\` | 4 | 0x5D | `]` | 4 | 0x5E | `^` | 8 | 0x5F | `_` | 7 | 0x60 | `` ` `` | 4 | 0x61 | `a` | 7 |
| 0x62 | `b` | 8 | 0x63 | `c` | 7 | 0x64 | `d` | 8 | 0x65 | `e` | 7 | 0x66 | `f` | 4 | 0x67 | `g` | 8 |
| 0x68 | `h` | 8 | 0x69 | `i` | 4 | 0x6A | `j` | 4 | 0x6B | `k` | 7 | 0x6C | `l` | 4 | 0x6D | `m` | 12 |
| 0x6E | `n` | 8 | 0x6F | `o` | 8 | 0x70 | `p` | 8 | 0x71 | `q` | 8 | 0x72 | `r` | 5 | 0x73 | `s` | 7 |
| 0x74 | `t` | 4 | 0x75 | `u` | 8 | 0x76 | `v` | 7 | 0x77 | `w` | 10 | 0x78 | `x` | 7 | 0x79 | `y` | 7 |
| 0x7A | `z` | 6 | 0x7B | `{` | 5 | 0x7C | `\|` | 4 | 0x7D | `}` | 5 | 0x7E | `~` | 8 | 0xA5 | `•` | 5 |
| 0xA9 | `©` | 10 | 0xAA | `™` | 13 | 0xC4 | `ƒ` | 7 | 0xC9 | `…` | 13 | 0xD0 | `–` | 7 | 0xD2 | `“` | 6 |
| 0xD3 | `”` | 6 | 0xD5 | `’` | 4 | | | | | | | | | | | | |

## 4.8 GEN9 per-character advances (plain)

| hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0x20 | `space` | 2 | 0x21 | `!` | 2 | 0x22 | `"` | 3 | 0x23 | `#` | 5 | 0x24 | `$` | 5 | 0x25 | `%` | 8 |
| 0x26 | `&` | 6 | 0x27 | `'` | 2 | 0x28 | `(` | 3 | 0x29 | `)` | 3 | 0x2A | `*` | 4 | 0x2B | `+` | 5 |
| 0x2C | `,` | 2 | 0x2D | `-` | 3 | 0x2E | `.` | 2 | 0x2F | `/` | 2 | 0x30 | `0` | 5 | 0x31 | `1` | 5 |
| 0x32 | `2` | 5 | 0x33 | `3` | 5 | 0x34 | `4` | 5 | 0x35 | `5` | 5 | 0x36 | `6` | 5 | 0x37 | `7` | 5 |
| 0x38 | `8` | 5 | 0x39 | `9` | 5 | 0x3A | `:` | 2 | 0x3B | `;` | 2 | 0x3C | `<` | 5 | 0x3D | `=` | 5 |
| 0x3E | `>` | 5 | 0x3F | `?` | 5 | 0x40 | `@` | 9 | 0x41 | `A` | 6 | 0x42 | `B` | 6 | 0x43 | `C` | 6 |
| 0x44 | `D` | 6 | 0x45 | `E` | 6 | 0x46 | `F` | 6 | 0x47 | `G` | 7 | 0x48 | `H` | 6 | 0x49 | `I` | 2 |
| 0x4A | `J` | 4 | 0x4B | `K` | 6 | 0x4C | `L` | 5 | 0x4D | `M` | 8 | 0x4E | `N` | 6 | 0x4F | `O` | 7 |
| 0x50 | `P` | 6 | 0x51 | `Q` | 7 | 0x52 | `R` | 6 | 0x53 | `S` | 6 | 0x54 | `T` | 6 | 0x55 | `U` | 6 |
| 0x56 | `V` | 6 | 0x57 | `W` | 8 | 0x58 | `X` | 6 | 0x59 | `Y` | 6 | 0x5A | `Z` | 6 | 0x5B | `[` | 2 |
| 0x5C | `\` | 2 | 0x5D | `]` | 2 | 0x5E | `^` | 4 | 0x5F | `_` | 5 | 0x60 | `` ` `` | 3 | 0x61 | `a` | 5 |
| 0x62 | `b` | 5 | 0x63 | `c` | 4 | 0x64 | `d` | 5 | 0x65 | `e` | 5 | 0x66 | `f` | 2 | 0x67 | `g` | 5 |
| 0x68 | `h` | 5 | 0x69 | `i` | 2 | 0x6A | `j` | 2 | 0x6B | `k` | 4 | 0x6C | `l` | 2 | 0x6D | `m` | 8 |
| 0x6E | `n` | 5 | 0x6F | `o` | 5 | 0x70 | `p` | 5 | 0x71 | `q` | 5 | 0x72 | `r` | 3 | 0x73 | `s` | 4 |
| 0x74 | `t` | 2 | 0x75 | `u` | 5 | 0x76 | `v` | 4 | 0x77 | `w` | 6 | 0x78 | `x` | 4 | 0x79 | `y` | 4 |
| 0x7A | `z` | 4 | 0x7B | `{` | 3 | 0x7C | `\|` | 2 | 0x7D | `}` | 3 | 0x7E | `~` | 5 | 0xA5 | `•` | 3 |
| 0xA9 | `©` | 7 | 0xAA | `™` | 9 | 0xC4 | `ƒ` | 5 | 0xC9 | `…` | 9 | 0xD0 | `–` | 5 | 0xD2 | `“` | 3 |
| 0xD3 | `”` | 3 | 0xD5 | `’` | 2 | | | | | | | | | | | | |

## 4.9 GEN12 per-character advances (plain)

| hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0x20 | `space` | 3 | 0x21 | `!` | 3 | 0x22 | `"` | 4 | 0x23 | `#` | 6 | 0x24 | `$` | 6 | 0x25 | `%` | 10 |
| 0x26 | `&` | 7 | 0x27 | `'` | 2 | 0x28 | `(` | 4 | 0x29 | `)` | 4 | 0x2A | `*` | 4 | 0x2B | `+` | 6 |
| 0x2C | `,` | 3 | 0x2D | `-` | 4 | 0x2E | `.` | 3 | 0x2F | `/` | 3 | 0x30 | `0` | 6 | 0x31 | `1` | 6 |
| 0x32 | `2` | 6 | 0x33 | `3` | 6 | 0x34 | `4` | 6 | 0x35 | `5` | 6 | 0x36 | `6` | 6 | 0x37 | `7` | 6 |
| 0x38 | `8` | 6 | 0x39 | `9` | 6 | 0x3A | `:` | 3 | 0x3B | `;` | 3 | 0x3C | `<` | 6 | 0x3D | `=` | 6 |
| 0x3E | `>` | 6 | 0x3F | `?` | 6 | 0x40 | `@` | 11 | 0x41 | `A` | 7 | 0x42 | `B` | 7 | 0x43 | `C` | 8 |
| 0x44 | `D` | 8 | 0x45 | `E` | 7 | 0x46 | `F` | 7 | 0x47 | `G` | 9 | 0x48 | `H` | 8 | 0x49 | `I` | 3 |
| 0x4A | `J` | 6 | 0x4B | `K` | 7 | 0x4C | `L` | 6 | 0x4D | `M` | 9 | 0x4E | `N` | 8 | 0x4F | `O` | 9 |
| 0x50 | `P` | 7 | 0x51 | `Q` | 9 | 0x52 | `R` | 8 | 0x53 | `S` | 7 | 0x54 | `T` | 7 | 0x55 | `U` | 8 |
| 0x56 | `V` | 7 | 0x57 | `W` | 10 | 0x58 | `X` | 7 | 0x59 | `Y` | 7 | 0x5A | `Z` | 7 | 0x5B | `[` | 3 |
| 0x5C | `\` | 3 | 0x5D | `]` | 3 | 0x5E | `^` | 5 | 0x5F | `_` | 6 | 0x60 | `` ` `` | 4 | 0x61 | `a` | 6 |
| 0x62 | `b` | 6 | 0x63 | `c` | 6 | 0x64 | `d` | 6 | 0x65 | `e` | 6 | 0x66 | `f` | 3 | 0x67 | `g` | 6 |
| 0x68 | `h` | 6 | 0x69 | `i` | 2 | 0x6A | `j` | 2 | 0x6B | `k` | 6 | 0x6C | `l` | 2 | 0x6D | `m` | 9 |
| 0x6E | `n` | 6 | 0x6F | `o` | 6 | 0x70 | `p` | 6 | 0x71 | `q` | 6 | 0x72 | `r` | 4 | 0x73 | `s` | 6 |
| 0x74 | `t` | 3 | 0x75 | `u` | 6 | 0x76 | `v` | 6 | 0x77 | `w` | 8 | 0x78 | `x` | 6 | 0x79 | `y` | 6 |
| 0x7A | `z` | 6 | 0x7B | `{` | 4 | 0x7C | `\|` | 3 | 0x7D | `}` | 4 | 0x7E | `~` | 6 | 0xA5 | `•` | 4 |
| 0xA9 | `©` | 8 | 0xAA | `™` | 11 | 0xC4 | `ƒ` | 6 | 0xC9 | `…` | 11 | 0xD0 | `–` | 6 | 0xD2 | `“` | 4 |
| 0xD3 | `”` | 4 | 0xD5 | `’` | 2 | | | | | | | | | | | | |

## 4.10 GEN14 per-character advances (plain)

| hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv | hex | ch | adv |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0x20 | `space` | 4 | 0x21 | `!` | 4 | 0x22 | `"` | 5 | 0x23 | `#` | 7 | 0x24 | `$` | 7 | 0x25 | `%` | 12 |
| 0x26 | `&` | 9 | 0x27 | `'` | 2 | 0x28 | `(` | 4 | 0x29 | `)` | 4 | 0x2A | `*` | 5 | 0x2B | `+` | 8 |
| 0x2C | `,` | 4 | 0x2D | `-` | 4 | 0x2E | `.` | 4 | 0x2F | `/` | 4 | 0x30 | `0` | 7 | 0x31 | `1` | 7 |
| 0x32 | `2` | 7 | 0x33 | `3` | 7 | 0x34 | `4` | 7 | 0x35 | `5` | 7 | 0x36 | `6` | 7 | 0x37 | `7` | 7 |
| 0x38 | `8` | 7 | 0x39 | `9` | 7 | 0x3A | `:` | 4 | 0x3B | `;` | 4 | 0x3C | `<` | 8 | 0x3D | `=` | 8 |
| 0x3E | `>` | 8 | 0x3F | `?` | 7 | 0x40 | `@` | 13 | 0x41 | `A` | 9 | 0x42 | `B` | 9 | 0x43 | `C` | 9 |
| 0x44 | `D` | 9 | 0x45 | `E` | 9 | 0x46 | `F` | 8 | 0x47 | `G` | 10 | 0x48 | `H` | 9 | 0x49 | `I` | 4 |
| 0x4A | `J` | 6 | 0x4B | `K` | 9 | 0x4C | `L` | 7 | 0x4D | `M` | 11 | 0x4E | `N` | 9 | 0x4F | `O` | 10 |
| 0x50 | `P` | 9 | 0x51 | `Q` | 10 | 0x52 | `R` | 9 | 0x53 | `S` | 9 | 0x54 | `T` | 8 | 0x55 | `U` | 9 |
| 0x56 | `V` | 9 | 0x57 | `W` | 12 | 0x58 | `X` | 9 | 0x59 | `Y` | 9 | 0x5A | `Z` | 8 | 0x5B | `[` | 4 |
| 0x5C | `\` | 4 | 0x5D | `]` | 4 | 0x5E | `^` | 6 | 0x5F | `_` | 7 | 0x60 | `` ` `` | 4 | 0x61 | `a` | 7 |
| 0x62 | `b` | 7 | 0x63 | `c` | 6 | 0x64 | `d` | 7 | 0x65 | `e` | 7 | 0x66 | `f` | 4 | 0x67 | `g` | 7 |
| 0x68 | `h` | 7 | 0x69 | `i` | 3 | 0x6A | `j` | 3 | 0x6B | `k` | 6 | 0x6C | `l` | 3 | 0x6D | `m` | 11 |
| 0x6E | `n` | 7 | 0x6F | `o` | 7 | 0x70 | `p` | 7 | 0x71 | `q` | 7 | 0x72 | `r` | 4 | 0x73 | `s` | 6 |
| 0x74 | `t` | 4 | 0x75 | `u` | 7 | 0x76 | `v` | 6 | 0x77 | `w` | 9 | 0x78 | `x` | 6 | 0x79 | `y` | 6 |
| 0x7A | `z` | 6 | 0x7B | `{` | 4 | 0x7C | `\|` | 3 | 0x7D | `}` | 4 | 0x7E | `~` | 8 | 0xA5 | `•` | 5 |
| 0xA9 | `©` | 10 | 0xAA | `™` | 13 | 0xC4 | `ƒ` | 7 | 0xC9 | `…` | 13 | 0xD0 | `–` | 7 | 0xD2 | `“` | 4 |
| 0xD3 | `”` | 4 | 0xD5 | `’` | 3 | | | | | | | | | | | | |

Characters outside these tables (the rest of Mac Roman) do occur in house-authored strings.
Measure them from the same TTF at the same ppem; the tables above cover every character that
appears in the game's own resource strings and in the nine shipped houses.

## 4.11 Validation A - the 428-item DITL corpus

`DITL` (dialog item list) resources give the *authored* box for every piece of dialog text. If
the substitute font is right, every single-line item's rendered width must fit its box. That is
**270** independent width constraints - 230 single-line text-bearing items out of the 428 items
in the corpus, plus the 40 game-site measurements of §4.13. (Earlier drafts said "658
constraints"; 658 was 428 + 230, which double-counts every text item. The number of *width*
constraints is 230 + 40 = 270. The 428 items are still all parsed, but 198 of them - userItems,
icons, pictures, empty-title controls and the multi-line StaticText blocks that `TETextBox`
wraps - carry no single-line width constraint.)

The 230 single-line items are defined mechanically as: every `Button`, `CheckBox` and
`RadioButton` with a non-empty title, plus every `StaticText` whose box is <= 18 px tall (one
line of Chicago 12 plus slop). That predicate yields exactly 230 items.

`DITL` binary layout, verified by parsing all 54 resources:

| Offset | Size | Field |
|---|---|---|
| 0 | 2 | item count **minus 1**, big-endian |
| 2 | ... | items |

each item:

| Offset | Size | Field |
|---|---|---|
| 0 | 4 | reserved (placeholder for the item's handle at runtime; 0 on disk) |
| 4 | 8 | bounding `Rect` as (top, left, bottom, right), four big-endian `short` |
| 12 | 1 | item type |
| 13 | 1 | data length |
| 14 | n | data (title text for buttons/statictext; resource ID for icons/pictures) |
| | | padded to an even byte boundary |

(An early parse attempt that read the count from offset 2 instead of 0 produced garbage; the
count really is `(n - 1)` at offset 0.)

Observed census:

```
resources: 54 ; items: 428
```

| item type | count |
|---|---|
| StaticText | 115 |
| Button | 113 |
| userItem | 74 |
| Icon | 42 |
| CheckBox | 26 |
| Picture | 25 |
| RadioButton | 19 |
| EditText | 13 |
| Control | 1 |

StaticText box heights, as a histogram (height:count):

```
{16:76, 17:1, 18:1, 32:10, 33:1, 37:1, 40:1, 48:7, 52:1, 60:1, 64:8, 68:1, 74:1, 80:4, 84:1}
```

The clustering at multiples of 16 (16, 32, 48, 64, 80) with 76 items at exactly 16 is the
original's own statement that **Chicago 12's line height is 16 px**. That is why CHI12's
substitute is specified at 13 ppem (ascent 12 + descent 3 + leading 1 = 16).

Button box heights:

```
{16:2, 18:6, 20:105}
```

20 px is the standard 1990s push-button height; the six 18s and two 16s are the small buttons in
the tool palettes.

**Result.** All **230** single-line items were measured under CHI12 (§4.7) - summing the
per-character advances of §4.7 over the Mac Roman title bytes, no character falling outside the
table - against their **raw** box width. **Zero items overflow.** The five tightest raw margins,
all `StaticText`:

| slack | DITL | resource name | kind | text | CHI12 px | raw box px |
|---|---|---|---|---|---|---|
| +2 | 1016 | Original Artwork | StaticText | `Bounded` | 56 | 58 |
| +2 | 1018 | Sound Prefs | StaticText | `Volume:` | 52 | 54 |
| +3 | 1016 | Original Artwork | StaticText | `PICT ID:` | 51 | 54 |
| +4 | 1007 | Blower Info | StaticText | `Direction:` | 60 | 64 |
| +5 | 1001 | House Info | StaticText | `Highest Score:` | 91 | 96 |

(An earlier draft's version of this table listed thirteen rows at +2, twelve of them buttons -
`Discard` "48 / 50" and eleven copies of `Linked From?` "86 / 88". Those figures were neither
raw box widths nor correct widths: `Discard` measures **47** in a **58**-px box and
`Linked From?` measures **87** in a **96**-px box, so their raw slacks are +11 and +9. The 50 and
88 were box widths already reduced by an 8-px button inset, i.e. the two conventions were mixed
inside one table, which is what made the buttons appear tightest. Also, `Linked From?` occurs
**twelve** times, not eleven: DITL 1007, 1010, 1011, 1013, 1014, 1015, 1019, 1022, 1027, 1033,
1034 and **1035** (Microwave Info).)

Under a **conservative** padding assumption (buttons reserve 4 px of horizontal inset on each
side, checkboxes and radio buttons reserve 16 px for the box and gap), **still nothing goes
negative.** The single tightest item lands at exactly zero slack:

| slack | DITL | resource name | kind | text | CHI12 px | available px | raw box px |
|---|---|---|---|---|---|---|---|
| **0** | 1029 | Lock House | Button | `Don't Lock` | 66 | 66 | 74 |
| +1 | 1007 etc. (x12) | *Info dialogs* | Button | `Linked From?` | 87 | 88 | 96 |

So the corpus imposes no substitute-induced clipping at all, even under the pessimistic inset
assumption. (An earlier draft reported `Don't Lock` as a 1-px clip at "67 vs 66"; the width is
66, not 67, so the margin is 0.)

## 4.12 The one per-item font override, and the 27 dialogs with none

The Mac dialog machinery allows per-item font overrides via an `ictb` resource, and the
Appearance Manager allows a whole-dialog font via `DLGX`. Full inventory:

| Type | Count | IDs | Content |
|---|---|---|---|
| `DLOG` | 28 | - | dialog templates |
| `DITL` | 54 | - | item lists (54 > 28 because the editor's info dialogs share templates) |
| `dctb` | 20 | - | dialog colour tables (colour only, no font data) |
| `cctb` | 1 | - | control colour table (colour only) |
| `ictb` | **1** | 150 | **36 bytes, every byte zero** |
| `DLGX` | **1** | 150 | 190 bytes |

Both `ictb` 150 and `DLGX` 150 belong to `DLOG`/`DITL` 150, the About box
(`GliderPRO/Sources/About.c`: `kAboutDialogID` 150, `kTextItemVers` 2, `kPictItemMain` 4).
DITL 150 has 9 items.

`ictb` 150 being **all zeros** means it specifies no override for any item - a zero `ictb` entry
means "inherit". So **not one dialog item in the game overrides the port default font.**

`DLGX` 150's observed bytes: a Pascal string `08 "Charcoal"` at offset 0, then 63 big-endian
`short` starting at offset 0x40, the first of which is 12. Nonzero shorts occur at short indices
0 (=12), 4 (=4), 5 (=4), 8 (=9), 9 (=9), then at a stride of 6 from index 15: 15 (=6), 21 (=8),
27 (=9), 33 (=6), 39 (=6), 45 (=10), 51 (=10), 57 (=10). That is 8 stride-6 values for DITL
150's items 2..9. The `08 "Charcoal"` + leading 12 clearly names a font and size (Charcoal was
the Mac OS 8 system font, a Chicago successor).

**Caveat, stated deliberately:** beyond `"Charcoal"` and the leading 12, I cannot name the
`DLGX` byte layout with confidence from the material available offline. The raw bytes are
recorded here so a future reader can decode them; **do not build a port on an inferred `DLGX`
schema.** The practical impact is confined to the About box, which is nine items.

**Conclusion for the port:** every dialog string in Glider PRO renders in the port default
(CHI12), with the possible exception of the About box, where the original may have used
Charcoal 12 on Mac OS 8+. Rendering the About box in CHI12 is a defensible approximation.

## 4.13 Validation B - 40 game-site layout deltas

These are the non-dialog sites, where the geometry is hard-coded in C rather than in a `DITL`.
Every row was computed from the substitute advance tables against the budget the source actually
provides. "budget px" is the drawable width minus the hard-coded left offset (for left-aligned
text) or the full field width (for centred text).

| # | site | citation | font | string (len) | substitute width | budget px | slack | note |
|---|---|---|---|---|---|---|---|---|
| 1 | scoreboard room title (escaped msg) | `Scoreboard.c:156` | GEN12+bold | `Hit Delete key if unable to Follow` (34) | 191 | 255 | +64 | fits |
| 2 | scoreboard room title (saving) | `Scoreboard.c:160` | GEN12+bold | `Saving Game…` (12) | 89 | 255 | +166 | fits |
| 3 | scoreboard room title, 27x'W' (worst legal) | `Scoreboard.c:164` | GEN12+bold | `WWWWWWWWWWWWWWWWWWWWWWWWWWW` (27) | 297 | 255 | **-42** | clips in original too |
| 4 | scoreboard room title, 27x'm' | `Scoreboard.c:164` | GEN12+bold | `mmmmmmmmmmmmmmmmmmmmmmmmmmm` (27) | 270 | 255 | **-15** | clips in original too |
| 5 | scoreboard room title, longest real name | `Scoreboard.c:164` | GEN12+bold | `Fahrenheit 451 Fodder` (21) | 132 | 255 | +123 | fits |
| 6 | scoreboard glider count, 2 digits | `Scoreboard.c:216` | GEN12+bold | `88` (2) | 14 | 19 | +5 | fits |
| 7 | scoreboard score, 9 digits | `Scoreboard.c:251` | GEN12+bold | `888888888` (9) | 63 | 63 | **+0** | exact - this is the constraint that fixes GEN12 at 11 ppem |
| 8 | scoreboard score, 10 digits | `Scoreboard.c:251` | GEN12+bold | `8888888888` (10) | 70 | 63 | **-7** | clips in original too |
| 9 | banner line, 39x'W' (WrapText 40) | `Banner.c:137` | GEN12+bold | 39 x `W` | 429 | 314 | **-115** | clips in original too |
| 10 | banner line, demo house longest | `Banner.c:137` | GEN12+bold | `This is a small beginner house that` (35) | 203 | 314 | +111 | fits |
| 11 | banner star line (plural, 3 digits) | `Banner.c:160` | GEN12+bold | `There are 999 stars in the house.` (33) | 195 | 314 | +119 | fits |
| 12 | banner star line (singular) | `Banner.c:160` | GEN12+bold | `There is 1 star in the house.` (29) | 165 | 314 | +149 | fits |
| 13 | stars-remaining count, 3 digits (centred at +102) | `Banner.c:228-229` | GEN12+bold | `999` (3) | 21 | 204 | +183 | fits |
| 14 | game-over trailer, 63x'W' (WrapText 64) | `GameOver.c:108` | GEN12+bold | 63 x `W` | 693 | 639 | **-54** | clips in original too |
| 15 | game-over trailer, demo longest line | `GameOver.c:108` | GEN12+bold | `That's the extent of the Demo House t...` (54) | 329 | 639 | +310 | fits |
| 16 | high scores rank, 2 digits | `HighScores.c:185` | GEN12+bold | `10` (2) | 14 | 29 | +15 | fits |
| 17 | high scores name, 15x'W' (`Str15`) | `HighScores.c:202` | GEN12+bold | 15 x `W` | 165 | 129 | **-36** | clips in original too |
| 18 | high scores name, 15x'n' | `HighScores.c:202` | GEN12+bold | 15 x `n` | 105 | 129 | +24 | fits |
| 19 | high scores level, 3 digits | `HighScores.c:219` | GEN12+bold | `999` (3) | 21 | 31 | +10 | fits |
| 20 | high scores `rooms` | `HighScores.c:239` | GEN12+bold | `rooms` (5) | 36 | 97 | +61 | fits |
| 21 | high scores score, 9 digits | `HighScores.c:253` | GEN12+bold | `888888888` (9) | 63 | 61 | **-2** | clips in original too |
| 22 | high scores score, 7 digits | `HighScores.c:253` | GEN12+bold | `8888888` (7) | 49 | 61 | +12 | fits |
| 23 | high scores banner, `Str31` 31x'W' | `HighScores.c:157` | GEN12+bold | 31 x `W` | 341 | 352 | +11 | fits |
| 24 | high scores banner, demo house | `HighScores.c:157` | GEN12+bold | `The Return of Ozma!` (19) | 122 | 352 | +230 | fits |
| 25 | high scores house title, 31x'W' | `HighScores.c:142` | GEN14+bold | `• ` + 31 x `W` + ` •` (35) | 425 | 352 | **-73** | clips in original too |
| 26 | high scores house title, Demo House | `HighScores.c:142` | GEN14+bold | `• Demo House •` (14) | 106 | 352 | +246 | fits |
| 27 | high scores footer prompt | `HighScores.c:272` | GEN9+bold | `Click Mouse or Hit a Key to Exit` (32) | 150 | 272 | +122 | fits |
| 28 | tool name, longest (19 ch) | `Tools.c:150` | GEN9+bold | `Invisible Rebounder` (19) | 96 | 115 | +19 | fits |
| 29 | tool name, `Grease (spills rt.)` | `Tools.c:150` | GEN9+bold | `Grease (spills rt.)` (19) | 84 | 115 | +31 | fits |
| 30 | splash `House: <name> (QT)`, 31-ch name | `MainWindow.c:71` | GEN9+bold | `House: ` + 31 x `W` + ` (QT)` (43) | 341 | 204 | **-137** | clips in original too |
| 31 | splash `House: Demo House (QT)` | `MainWindow.c:71` | GEN9+bold | `House: Demo House (QT)` (22) | 123 | 204 | +81 | fits |
| 32 | calendar month `SEPTEMBER` | `ObjectDraw2.c:1162` | GEN9+bold | `SEPTEMBER` (9) | 65 | 64 | **-1** | **substitute-induced**, see below |
| 33 | calendar month `FEBRUARY` | `ObjectDraw2.c:1162` | GEN9+bold | `FEBRUARY` (8) | 56 | 64 | +8 | fits |
| 34 | calendar month `MAY` | `ObjectDraw2.c:1162` | GEN9+bold | `MAY` (3) | 23 | 64 | +41 | fits |
| 35 | load-dialog house name (`TETextBox`) | `DialogUtils.c:642` | GEN9 | 31 x `W` | 248 | 94 | **-154** | `CollapseStringToWidth` trims first, so never actually clips |
| 36 | coord window `h: 999` | `Coordinates.c:82-83` | CHI12 | `h: 999` (6) | 37 | 45 | +8 | fits |
| 37 | coord window `v: -999` | `Coordinates.c:93-94` | CHI12 | `v: -999` (7) | 40 | 46 | +6 | fits |
| 38 | message window longest string | `WindowUtils.c:146` | CHI12 | `Converting 1.0 House to 2.0` (27) | 171 | 224 | +53 | fits |
| 39 | `No rooms` (inherits `workSrcMap` state) | `Room.c:259` | CHI12 | `No rooms` (8) | 61 | 630 | +569 | fits |
| 40 | `PowerPC Native!` | `MainWindow.c:88` (shadow) / `:91` (face) | CHI12 plain | `PowerPC Native!` (15) | 103 | 635 | +532 | fits |

**11 of 40 rows are negative. Ten of the eleven are genuine original overflows**, i.e. the 1994
game clipped there too:

* Rows 3, 4: the room-name field is 255 px but room names may be 27 characters
  (`Str27`); wide characters overflow.
* Row 8: `Str` conversion of a 10-digit score into a 64-px field.
* Rows 9, 14: `WrapText` wraps by **character count**, not by pixel width
  (`GliderPRO/Sources/StringUtils.c:216-247`, `WrapText(StringPtr, short maxChars)`), so a line
  of 40 or 64 wide characters exceeds the pixel budget by construction.
* Row 17: `Str15` player names, 15 wide characters into 129 px.
* Row 21: 9-digit high scores into 61 px.
* Rows 25, 30: 31-character house names.
* Row 35: over-wide by construction, but `CollapseStringToWidth` (§4.16) truncates before the
  draw, so nothing actually clips.

**Exactly one row is substitute-induced**: row 32, `SEPTEMBER` at 65 px in a 64-px calendar
field. The draw is centred with `(fieldWidth - StringWidth) / 2`; in C, `(64 - 65) / 2` is
`-1 / 2` = **0** (truncation toward zero), so the string starts at the left edge and clips 1 px
on the right. Two acceptable fixes: (a) accept the 1-px clip, which is what a faithful port
does; (b) drop `SEPTEMBER` to 8 px of tracking. Prefer (a).

## 4.14 The exact geometry of the four text-heavy screens

Recorded here because the layout is meaningless without it, and because it is what the font
tables have to satisfy.

### 4.14.1 Scoreboard (`GliderPRO/Sources/Scoreboard.c:136-334`)

Constants (`Scoreboard.c:15-21`): `kGrayBackgroundColor` 251, `kGrayBackgroundColor4` 10,
`kFoilBadge` 0, `kBandsBadge` 1, `kBatteryBadge` 2, `kHeliumBadge` 3, `kScoreRollAmount` 13.

Five depth-4 branches at `:143`, `:201`, `:239`, `:276`, `:311`, each of the form
`Index2Color(thisMac.isDepth == 4 ? kGrayBackgroundColor4 : kGrayBackgroundColor)`.

The universal drop-shadow idiom (repeated at each of the five fields):

```
1. MoveTo(1, 10);  ForeColor(blackColor); DrawString(theStr);   // shadow, +1,+1
2. MoveTo(0,  9);  ForeColor(whiteColor); DrawString(theStr);   // face
```

so every scoreboard string is drawn **twice**, black then white, offset (1, 1). Baselines are
10 (shadow) and 9 (face) within the 12-px-tall `boardTSrcRect`, which is why GEN12's ink descent
must be <= 2.

### 4.14.2 Banner (`GliderPRO/Sources/Banner.c:113-236`)

```c
// DrawBannerMessage
	TextFont(applFont); TextFace(bold); TextSize(12); ForeColor(blackColor);
	count = 0;
	do
	{
		GetLineOfText(bannerStr, count, subStr);
		MoveTo(topLeft.h + 16, topLeft.v + 32 + (count * 20));
		DrawString(subStr);
		count++;
	}
	while (subStr[0] > 0);
```

* Left inset **16 px**, first baseline at **+32**, line pitch **20 px**.
* The loop is a `do/while (subStr[0] > 0)`, so it always draws at least one line and stops after
  the first empty one - meaning the *last* iteration draws an empty string at the next baseline.
* Star lines are drawn in `redColor` at `topLeft.v + 164` and `topLeft.v + 180`.
* `BringUpBanner` uses `QSetRect(&wholePage, 0, 0, 330, 220)` and
  `WaitForInputEvent(demoGoing ? 4 : 15)` - **4 ticks during a demo, 15 otherwise**.
* `DisplayStarsRemaining` uses `QSetRect(&bounds, 0, 0, 256, 64)`, `InsetRect(&src, 64, 32)`,
  `QOffsetRect(&bounds, 0, -20)`, then
  `MoveTo(bounds.left + 102 - (StringWidth(theStr) / 2), bounds.top + 23); ColorText(theStr, 4L);`
  followed by `DelayTicks(60)` and `WaitForInputEvent(30)`.

The 314-px banner budget in §4.13 is `330 - 16` = the page width minus the left inset.

### 4.14.3 High scores (`GliderPRO/Sources/HighScores.c`)

Constants: `kHighScoresPictID` 1994, `kHighScoresMaskID` 1998, `kHighNameDialogID` 1020,
`kHighBannerDialogID` 1021, `kHighNameItem` 2, `kNameNCharsItem` 5, `kHighBannerItem` 2,
`kBannerScoreNCharsItem` 5 (`HighScores.c:23-30`), and `kScoreSpacing` **18**,
`kScoreWide` **352**, `kKimsLifted` **4** (`HighScores.c:90-92`).

Derived origins:

```
scoreLeft = ((thisMac.screen.right - thisMac.screen.left) - kScoreWide) / 2
dropIt    = 129 + splashOriginV
```

Layout, in draw order:

| Element | Font | Position |
|---|---|---|
| 332 x 30 title graphic | - | `CopyMask` to `scoreLeft + (kScoreWide - 332)/2, dropIt - 60` (1-bit mask) |
| House title, shadow | GEN14+bold | black at `((kScoreWide - StringWidth)/2) - 1, dropIt - 66`; string is `"\p• " + thisHouseName + "\p •"` |
| House title, face | GEN14+bold | cyan at `+0, dropIt - 65` |
| Banner, shadow | GEN12+bold | black at `dropIt - kKimsLifted` |
| Banner, face | GEN12+bold | yellow at `dropIt - kKimsLifted - 1` |
| Banner box | - | `bannerWidth + 8` x `kScoreSpacing` at `scoreLeft - 3 + (kScoreWide - bannerWidth)/2, dropIt + 5 - kScoreSpacing - kKimsLifted`; framed black, then offset (-1, -1) and framed yellow |
| Row `i` rank | GEN12+bold | shadow `+1`, face `+0` |
| Row `i` name | GEN12+bold | shadow `+31`, face `+30` |
| Row `i` level | GEN12+bold | shadow `+161`, face `+160` |
| Row `i` `"rooms"` | GEN12+bold | shadow `+193`, face `+192`, **always cyan** |
| Row `i` score | GEN12+bold | shadow `+291`, face `+290` |
| Footer prompt | GEN9+bold | `blueColor` at `scoreLeft + 80, dropIt - 1 + (10 * kScoreSpacing)`, text from `GetLocalizedString(8, ...)` = `"Click Mouse or Hit a Key to Exit"` |

Row baselines: **row 0 is special-cased** at `dropIt - kScoreSpacing - kKimsLifted`; rows 1..9
are at `dropIt + (i * kScoreSpacing)`. Face colour is `whiteColor` when `i == lastHighScore`
(the just-achieved score), otherwise cyan for the rank and yellow for name/level/score.

`kMaxScores` is 10 (`GliderPRO/Headers/GliderDefines.h`), hence the 10 rows and the
`10 * kScoreSpacing` footer offset.

### 4.14.4 Coordinate window (`GliderPRO/Sources/Coordinates.c`)

```c
// OpenCoordWindow, :116-167 (abridged)
	QSetRect(&coordWindowRect, 0, 0, 50, 38);
	...
	coordWindow = NewCWindow(nil, &coordWindowRect, "\pTools", false,
			kWindoidWDEF, kPutInFront, true, 0L);
	...
	TextFace(applFont);     // :154  <-- BUG, see 4.16
	TextSize(9);            // :155
```

`UpdateCoordWindow` (`:61-110`) does `SetPort((GrafPtr)coordWindow)` and then draws with **no
font calls at all**, so the coordinate window renders in the port default = **Chicago 12
plain**. Draw positions: `MoveTo(5, 12)` for `h:`, `MoveTo(4, 22)` for `v:`, `MoveTo(5, 32)` for
`d:` in `blueColor` (`:82-83`, `:93-94`, `:103-104`). Those are rows 36 and 37 of §4.13's table,
and they fit under CHI12. Note that `OpenCoordWindow` never calls `SetPort` before its
`TextFace`/`TextSize` pair, so those two calls do **not** stamp the coord window's port - they
stamp whatever port was current at the time (§4.17 bug 1), which is why the window really does
render in the port default.

## 4.15 `TETextBox` - the one path that consults font ascent

```c
// GliderPRO/Sources/DialogUtils.c:616-648   DrawDialogUserText
	TextFont(applFont);                                          // :623   (no TextFace call)
	TextSize(9);                                                 // :624
	...
	if ((StringWidth(stringCopy) + 2) > (iRect.right - iRect.left))          // :628
		CollapseStringToWidth(stringCopy, iRect.right - iRect.left - 2);     // :629
	...
	inset = ((iRect.right - iRect.left) - (StringWidth(stringCopy) + 2)) / 2; // :638
	iRect.left += inset;  iRect.right -= inset;                              // :639-640
	TETextBox(newString, textLong, &iRect, teCenter);                        // :642
```

Note the two details an earlier draft dropped: the collapse is **guarded** (it only runs when the
string plus 2 px exceeds the box) and its budget is `width - 2`, not `width`. The `+ 2` appears
twice and must be reproduced in both places or the centring inset drifts by one pixel.

`TETextBox` is TextEdit, not raw QuickDraw: it computes its own line height from the port font's
`FontInfo` (ascent + descent + leading) and positions the first baseline at `iRect.top + ascent`.
So this **one** call is sensitive to the substitute's vertical metrics. GEN9's specified
ascent 9 / descent 2 / leading 0 gives line height 11 and a first baseline at `top + 9`.

The sibling function does *not* use TextEdit:

```c
// GliderPRO/Sources/DialogUtils.c:655-671   DrawDialogUserText2
	TextFont(applFont);                                          // :662   (no TextFace call)
	TextSize(9);                                                 // :663
	...
	if ((StringWidth(stringCopy) + 2) > (iRect.right - iRect.left))          // :667
		CollapseStringToWidth(stringCopy, iRect.right - iRect.left - 2);     // :668
	MoveTo(iRect.left, iRect.bottom);                                        // :669
	DrawString(stringCopy);                                                  // :670
```

Baseline pinned to `iRect.bottom`, so vertical metrics are irrelevant. `DrawDialogUserText2` is
what the About box uses for its two live diagnostic lines:

```c
// GliderPRO/Sources/About.c:138-163   UpdateMainPict
//   item 7: "Memory:   <K>K"
//   item 8: "Screen:   <w>x<h>x<depth>"
```

DITL 150 item **9** is a third 120 x 9 `userItem` that is **never drawn** - dead layout.

## 4.16 `CollapseStringToWidth`, and a latent infinite loop

Verbatim (an earlier draft paraphrased this function with an early `return` and the
`PasStringConcat` after the loop; the real control flow is a flag plus an in-loop append):

```c
// GliderPRO/Sources/StringUtils.c:282-296
void CollapseStringToWidth (StringPtr theStr, short wide)
{
	short		dotsWide;
	Boolean 	tooWide;

	dotsWide = StringWidth("\p…");                              // :287  0xC9 = Mac Roman ellipsis
	tooWide = StringWidth(theStr) > wide;                       // :288
	while (tooWide)                                             // :289
	{
		theStr[0]--;                                            // :291
		tooWide = ((StringWidth(theStr) + dotsWide) > wide);    // :292
		if (!tooWide)
			PasStringConcat(theStr, "\p…");                     // :294
	}
}
```

Behaviourally: if the string already fits, the loop never runs and **no ellipsis is appended**;
otherwise bytes are dropped one at a time until `width + dotsWide <= wide`, at which point the
ellipsis is appended and the loop exits on its next test. Note the asymmetry - the entry test is
`StringWidth > wide` but the exit test is `StringWidth + dotsWide > wide`, so the result is always
at most `wide` px *including* the ellipsis.

Two hazards a Go port must handle:

1. If `dotsWide > wide` (i.e. the field is narrower than an ellipsis), `wide - dotsWide` is
   negative and the loop **never terminates at all**. `StringWidth` is never negative, so the
   condition `StringWidth(theStr) > (wide - dotsWide)` is permanently true; `theStr[0]--` walks
   the *unsigned* length byte down to 0, wraps to 255 - which does **not** end the loop, it just
   makes `StringWidth` read 255 bytes of adjacent memory - and then cycles 255, 254, ... 0, 255
   forever. (An earlier draft said the loop "terminates only when `theStr[0]` wraps from 0 to
   255"; it does not terminate at that point or at any other.) Under CHI12 the ellipsis is 13 px;
   under GEN9 it is 9 px. Both call sites (`DialogUtils.c:629`, `:668`) pass
   `iRect.right - iRect.left - 2` from a `DITL` user item, and the narrowest such field in the
   shipped resources is far wider than 13 px, so this never fires in the shipped game - but any
   house-authored or resized dialog could trigger it.
2. `theStr[0]--` truncates mid-character with no UTF-8 or Mac Roman awareness. That is fine for
   Mac Roman (single-byte) and must be preserved as a **byte** truncation, not a rune
   truncation, or widths will differ.

Related string helpers:

| Function | Lines | Behaviour |
|---|---|---|
| `GetLineOfText` | `StringUtils.c:140-206` | extracts line `n` from a `\r`-separated Pascal string |
| `WrapText` | `StringUtils.c:216-247` | signature `WrapText(StringPtr, short maxChars)` - wraps by **character count**, never by pixel width; this is the source of the row 9 and row 14 overflows |
| `GetFirstWordOfString` | `StringUtils.c:256-273` | |
| `CollapseStringToWidth` | `StringUtils.c:282-296` | above |

## 4.17 Original text bugs to preserve (or knowingly not)

| # | Bug | Citation | Effect | Recommendation |
|---|---|---|---|---|
| 1 | `TextFace(applFont)` where `TextSize`/`TextFont` was meant | `Coordinates.c:154` | `applFont` is 1 and `bold` is 1, so this sets **bold**; and there is no preceding `SetPort`, so the bold+9 lands on whatever port happened to be current | Preserve the *observable* result: the coord window stays Chicago 12 plain. Do not propagate the stray bold. |
| 2 | `TextWidth` measured before the font is set | `GameOver.c:101-105` | `TextWidth` at `:101-102` precedes `TextFont`/`TextFace`/`TextSize` at `:103-105` inside a `do` loop, so only the **first** line of the trailer is centred against the *previous* port font; lines 2..n are already GEN12+bold | Preserve if you want frame-exact centring; document. |
| 3 | Font state inherited from another drawable | `Room.c:259, :261` | `"No rooms"` is drawn into `workSrcMap` in whatever font that GWorld last had | Preserve by modelling per-drawable text state (§4.3). |
| 4 | `CollapseStringToWidth` byte underflow | `StringUtils.c:287-291` | infinite loop + 255-byte read if `dotsWide > wide` | **Fix.** Clamp: if `wide <= dotsWide`, set `theStr[0] = 0` and return. |
| 5 | `RestoreColorDepth` always passes `doColor = true` | `Environ.c:549-554` | restores a 4-bit display as colour rather than gray | Harmless; fix silently. |
| 6 | `AreWeColorOrGrayscale` reads `gdFlags` twice | `Environ.c:348-368` | first read is a dead store | Harmless. |
| 7 | `demoIndex` unbounded | `Input.c:186-277` | reads 6 bytes past `demoData` after frame 3414 | **Fix.** Bounds-check against 1117 records. |
| 8 | `PenMode(srcOr)` where `patOr` was meant | `ObjectEdit.c:2364` | same pixels in this case | Implement as `patOr`. |
| 9 | `kRedOrangeColor8` comment says "actually, 18" | `GliderDefines.h:542` | comment is wrong; 23 is correct | Ignore the comment. |

## 4.18 The string corpus

The file holds **ten** `STR#` resources, not two. Parsed verbatim (2-byte big-endian count, then
Pascal strings back to back):

| ID | name | bytes | strings | role |
|---|---|---|---|---|
| 128 | Jinjur | 10 | 1 | single string |
| 129 | Prefmain | 33 | 4 | preferences |
| 140 | File Error | 1310 | 17 | file-error messages |
| **150** | **Localized Strings** | 1048 | **51** | the general corpus, tabulated below |
| 160 | Prefs | 14 | 1 | single string |
| 170 | Errors | 294 | 13 | error text |
| 171 | Errors | 1336 | 13 | error text (second set) |
| **1005** | **Months** | 88 | **12** | calendar object |
| 1006 | Yellow Alerts | 2527 | 24 | alert text |
| **1007** | **Object Names** | 1569 | **144** | the editor's tool names (`Tools.c:146-147`, `kObjectNameStrings` = 1007, `GliderDefines.h:461`) |

Only three of the ten matter for layout - 150, 1005 and 1007 - and the tool names in 1007 are the
source of §4.13 rows 28-29, so the earlier claim that "two `STR#` resources supply every
localizable string" understated the corpus in the one place a port has to measure.

`STR# 150` and `STR# 1005` in full:

`STR# 150 "Localized Strings"` - 1048 bytes, **51** strings:

| # | string | # | string |
|---|---|---|---|
| 1 | `There are ` | 27 | `Room Number Errors` |
| 2 | `There is ` | 28 | ` Duplicate Floor/Suites` |
| 3 | ` stars in the house.` | 29 | ` Room Errors` |
| 4 | ` star in the house.` | 30 | ` 'Untitled' Rooms` |
| 5 | `Get every star to win.` | 31 | ` Room Names Too Long` |
| 6 | `room` | 32 | ` Room's # of Objects Wrong` |
| 7 | `rooms` | 33 | `Checking Objects…` |
| 8 | `Click Mouse or Hit a Key to Exit` | 34 | ` Object Errors` |
| 9 | `Name for New House:` | 35 | `You have no stars in the house!` |
| 10 | `Untitled House` | 36 | `Cut Object` |
| 11 | `Enter New-Game Message here (max. 255 characters)` | 37 | `Copy Object` |
| 12 | `Enter Finished-House Message here (max. 255 characters)` | 38 | `Clear Object` |
| 13 | `Converting 1.0 House to 2.0` | 39 | `Cut Room` |
| 14 | `Converting Room ` | 40 | `Copy Room` |
| 15 | `Save copy of house as:` | 41 | `Clear Room` |
| 16 | `Failed House Shrinkage` | 42 | `Paste Room` |
| 17 | ` Floor Number Bad` | 43 | `Paste Object` |
| 18 | ` Suite Number Bad` | 44 | `Nothing To Paste` |
| 19 | `Object Bad` | 45 | `Object Pair Added` |
| 20 | `No room upstairs!` | 46 | `Exterior Door Added in Next Room` |
| 21 | `No downstairs to match!` | 47 | `Interior Door Added in Next Room` |
| 22 | `No room downstairs!` | 48 | `Ext. Window Added in Next Room` |
| 23 | `No upstairs to match!` | 49 | `Int. Window Added in Next Room` |
| 24 | `Checking House File` | 50 | `Down Stairs Added in Room Above` |
| 25 | `Checking House…` | 51 | `Up Stairs Added in Room Below` |
| 26 | `Checking Rooms…` | | |

Note strings 25, 26, 33 end with the Mac Roman ellipsis `0xC9`, not three periods. String 8 is
the high-score footer (§4.14.3). Strings 1-7 build the banner's star sentence (§4.14.2).

`STR# 1005` - 88 bytes, **12** strings: `JANUARY`, `FEBRUARY`, `MARCH`, `APRIL`, `MAY`, `JUNE`,
`JULY`, `AUGUST`, `SEPTEMBER`, `OCTOBER`, `NOVEMBER`, `DECEMBER`. Drawn by the calendar object
(§4.13 rows 32-34); `SEPTEMBER` is the 9-character worst case.

The longest string that reaches the message window is #13, `Converting 1.0 House to 2.0`,
27 characters = 171 px under CHI12 against a 224-px budget (§4.13 row 38).

**Rez-dump gotcha:** these resources appear in `GliderPRO/Glider PRO.r` as
`data 'STR#' (150, "Localized Strings") {` - the `data` keyword, not `resource`. A parser
looking for `resource 'STR#' (150` finds nothing.

## 4.19 Go implementation sketch

```go
// FontKey selects one of the four (font, size, face) tuples the game actually uses.
type FontKey int

const (
	CHI12 FontKey = iota // systemFont, size 0/12, plain  -> Liberation Sans Bold    @ 13 px
	GEN9                 // applFont,   size 9,    bold   -> Liberation Sans Regular @  9 px
	GEN12                // applFont,   size 12,   bold   -> Liberation Sans Regular @ 11 px
	GEN14                // applFont,   size 14,   bold   -> Liberation Sans Regular @ 13 px
)

type FontMetrics struct {
	Ascent, Descent, Leading int   // only TETextBox (DialogUtils.c:642) consults these
	Advance                  [256]int8 // Mac Roman code point -> plain advance in px
	SyntheticBold            bool      // GEN9/GEN12/GEN14: add 1 px per character
}

// StringWidth reproduces the QuickDraw call. Operates on Mac Roman BYTES,
// not runes: CollapseStringToWidth truncates by byte (StringUtils.c:291).
func (m *FontMetrics) StringWidth(s []byte) int {
	w := 0
	for _, b := range s {
		w += int(m.Advance[b])
	}
	if m.SyntheticBold {
		w += len(s) // QuickDraw synthesised bold: +1 advance per character
	}
	return w
}

// TextState is per-drawable, mirroring CGrafPort.txFont/txSize/txFace.
// It must survive across draw calls - Scoreboard.c relies on state stamped
// once at StructuresInit.c:88-145 and never re-set per frame.
type TextState struct{ Key FontKey }
```

Do not use a system font lookup. Do not let a text-shaping library apply kerning, ligatures or
fractional advances: QuickDraw with `SetFractEnable` never called does none of those, and any of
them shifts every centred string.

---

# Part 5 - open questions closed elsewhere

This part records which pre-existing open questions this document answers, so they can be struck
from their home documents. Each is answered by the ruling named.

## 5.1 `docs/analysis/rendering.md` Q1 - `srcXor`/`patXor` on an 8-bit destination

**Closed by R-XOR-1 (Part 2).** rendering.md offers two candidate answers (XOR of the RGB values
vs XOR of the palette indices) and cannot choose. The choice is forced, not by preference but by
the source's depth: the source pixmap is **1 bit** (`StructuresInit.c:258`), so QuickDraw
colourises to fg/bg *before* the boolean op, and the boolean op therefore operates on the
destination's pixel values = palette indices. With fg = black = index 255 and bg = white =
index 0 (proved invariant by `ColorUtils.c`'s save/restore idiom, §2.4), the two candidates
**coincide**: the XOR value is 0xFF either way.

Additional facts rendering.md does not have: there is exactly **one** `srcXor` in the tree; the
involution is exercised in three places, not one (§2.6); and `DrawMarquee` re-XORs the stencil at
`Marquee.c:493-494`, so every ant-march blink applies an even number of XORs.

## 5.2 `docs/analysis/rendering.md` Q2 - whether the 4-bit CLUT is `(15-i)*17`

**Closed by R-CLUT4-1 (Part 3), and upgraded from "inferred" to "proved".** §3.4's argument is
independent of any assumption about the system: the standard 8-bit palette contains exactly one
index per `(15-i)*0x1111` level, and **11 of the 13 pure-gray authored `(i4, i8)` pairs are exact
identity hits** under `gray8[i4] == i8`. The two exceptions - `(15, 254)` at 5 drop-shadow sites
and `(8, 251)` at 1 site - are explainable as authoring slips in a corpus where every other pure
gray is exact.

rendering.md §22.4's 20-row table is superseded by §3.5's **71-site / 22-combo** table (which
contains it plus `(2, 246)` and `(3, 43)`). §22.4's luminance column is off by one on two rows:
`#CC9933` is **157**, not 158, and `#996633` is **111**, not 112, under the game's own
three-division formula.

rendering.md also asks whether the map can be *derived*. It cannot: the relation is not a
function in either direction (§3.9 - `i8` 223, 251 and 254 each have two authored `i4`
partners). The best available rule is R1 at 17/22 combos and 55/72 sites; truncation (R6) is
7/22 and must not be used.

## 5.3 `docs/analysis/rendering.md` Q7 - fonts

**Closed by R-FONT-1 (Part 4).** No `FOND`/`NFNT`/`FONT`/`sfnt` exists in the 538 resource
records, so option (a) - real metrics - is impossible offline. Option (b) is supplied: four named
substitutes, four full per-character advance tables, and 270 validated measurements (230 DITL
items with zero raw-box overflows, and zero even under a conservative button/checkbox inset
assumption, plus 40 game-site rows of which exactly one negative row is substitute-induced).

**Correction to rendering.md §23.6:** the point sizes used are **9, 12 and 14**, not "9, 10 and
12". There is no `TextSize(10)` anywhere; `TextSize(14)` occurs once, at `HighScores.c:142`.

## 5.4 `docs/analysis/rendering.md` Q8 - the exact `Random()` sequence

**Closed by R-RNG-1 and R-RNG-2 (Part 1).** The generator is specified (Lehmer 16807 mod
2147483647 with Schrage decomposition, low word as `int16`, seed 1) and the seed-1 stream is
tabulated for 24 draws (§1.6). More importantly the *stakes* are established: the RNG feeds only
five arrays, all five read exclusively by `CopyBits` source-rect selection in `Render.c`
(§1.13), so no RNG value can reach physics, collision, scoring or the demo trajectory.

## 5.5 `docs/analysis/scoring.md` open question 28 - the application font

scoring.md item 28 (at `scoring.md:5401`, and cited again as fidelity risk #1 at
`scoring.md:5630`) reads, in part: *"the string widths and baselines in this document are exact
only if the port's font metrics match Geneva 9/12. This is the largest single source of visual
divergence in the whole subsystem."*

**Partly closed, and materially de-risked, by R-FONT-1 (Part 4).**

* **Baselines are not at risk at all.** `GetFontInfo` and `FontMetrics` are never called
  (§4.2), so every baseline in scoring.md is a hard-coded integer from the source. The only
  exception in the entire program is `TETextBox` at `DialogUtils.c:642` (Geneva 9), which is not
  in the scoring subsystem.
* **Widths are at risk only where a `StringWidth` result feeds a position.** There are 11
  `StringWidth` sites and 1 `TextWidth` site in the whole tree; the scoring-relevant ones are
  `Banner.c:228` (stars-remaining centring), `HighScores.c:140` and `:143` (house-title shadow
  and face centring), `HighScores.c:157` (banner box width), `ObjectDraw2.c:1162` (calendar month
  centring) and `DialogUtils.c:638` (`DrawDialogUserText`'s centring inset; `:628` and `:667` use
  `StringWidth` only as a fits/doesn't-fit test, and `StringUtils.c:287`, `:288`, `:292` are
  inside `CollapseStringToWidth`). Every one is a *centring* computation, so an error of `d` px in
  `StringWidth` shifts the string by `d/2` px and nothing else moves.
* **Measured divergence on the scoring surfaces**: of §4.13's 27 scoring-related rows
  (rows 1-27), **26 behave as the original does** (fits where the original fits, clips where the
  original clips) and 1 - row 21, the 9-digit high-score at 63 px in a 61-px column - clips by
  2 px where the original clipped by an unknown amount. The 9-digit score field's own constraint
  (row 7, `boardPSrcRect` = 64 px) is satisfied **exactly**, at +0 px slack, which is the
  strongest available evidence that GEN12 at 11 ppem is the right size.

So item 28's residual risk is: sub-pixel-scale horizontal shifts on five centred strings, and a
2-px clip on 9-digit high scores. It is no longer "the largest single source of visual
divergence"; it is bounded and enumerated.

## 5.6 `docs/analysis/determinism.md` Part 2 - the demo's RNG dependence

**Disproved by R-RNG-2 (§1.12).** determinism.md Part 2 correctly identifies `RandomInt` as the
only live RNG and correctly transcribes `RandomInt(range) = (|Random()| * range) / 32768`
(`Utilities.c:72-82`). Its further claim - that the shipped `demo` 128 replay depends on
`Random()` - does not survive the four-step argument of §1.12: the demo's input stream is a pure
function of `gameFrame`, `Player.c` has zero RNG calls, Demo House contains only 5
RNG-consuming object instances and zero `kChimes`/`kSparkle`/`kCoffee`, and the one per-frame
RNG consumer (the telephone) cannot fire inside the demo's 46..3414 frame window because
`nextRing >= kRingBaseDelay` = 5000.

**Correction to determinism.md §2.2 point 4:** the discussion of `GetDateTime` seeding applies to
the *non-Carbon* target only. `GliderPRO/Prefix.h` sets `TARGET_CARBON 1`, which compiles
`Utilities.c:61` out, so the shipped 1.0.4 build starts from `qd.randSeed == 1` on every launch.
Both targets exist in `Glider PRO CW5.mcp`; this document normatively selects Carbon (§1.3).

## 5.7 Summary of corrections to existing documents

| Document | Claim | Correction | Evidence |
|---|---|---|---|
| `rendering.md` §22.4 | 20 `(i4, i8)` pairs | **72 sites / 22 unique combos**; the two extra combos are `(2, 246)` and `(3, 43)` | §3.5 |
| `rendering.md` §22.4 | `#CC9933` luminance 158 | **157** | §3.5 row 3 |
| `rendering.md` §22.4 | `#996633` luminance 112 | **111** | §3.5 row 8 |
| `rendering.md` §22.4 | "15 of 20 fit `(15-i)*17`" | 17 of 22 combos / 55 of 72 sites fit under R1; and **11 of 13 pure grays are exact identity hits**, which is the decisive statistic | §3.4, §3.7 |
| `rendering.md` §23.6 | sizes 9, 10, 12 | sizes **9, 12, 14** | §4.2, §4.4 |
| `rendering.md` Q1 | two candidate XOR semantics | they coincide; the value is 0xFF | §5.1 |
| `determinism.md` §2.2 pt 4 | date-seeded `randSeed` | Carbon target compiles the seeding out; seed is 1 | §1.3 |
| `determinism.md` Part 2 | demo replay depends on `Random()` | it does not | §1.12 |
| `scoring.md` item 28 | font metrics are the largest divergence source | bounded to 5 centred strings + a 2-px clip | §5.5 |

---

## Open questions

Numbered so they can be cited. Each states what would settle it.

1. **Apple's actual `Random()` algorithm.** `GliderPRO/CarbonLib` exports the symbol
   (container 0, symbolClass 2, `symbolValue` 0, `sectionIndex` -2) but contains **zero code
   sections** across all four of its PEF containers - every one has `sectionCount == 1` and that
   section is kind 4 (loader). The implementation is therefore not in this repository.
   R-RNG-1 is a well-motivated substitution, not a recovered artefact. *Settled by:* disassembling
   a real `QuickDraw`/`InterfaceLib` binary, or a hardware/emulator capture of the first 24
   `Random()` values from `randSeed == 1`. If those match §1.6, R-RNG-1 is exact.

2. **`Random()` on a negative seed.** Every `GetDateTime` value from 1972 onward exceeds
   `0x7FFFFFFF` and so is negative when reinterpreted as the `SInt32` that `qd.randSeed` is.
   A Lehmer generator requires state in [1, m-1]. Whether Apple masked, took absolute value, or
   let it run is unknown. This only matters for the non-Carbon target (§1.3), which this document
   does not adopt. *Settled by:* the same capture as (1), seeded from a known date.

3. **Whether `qd.randSeed` really is 1 under CarbonLib.** Classic `InitGraf` documented
   `randSeed = 1`. Carbon has no `InitGraf`, and `Utilities.c:43-67` calls nothing that would set
   it. The assumption is that CarbonLib's implicit QuickDraw initialisation also uses 1.
   *Settled by:* reading `qd.randSeed` under CarbonLib before the first `Random()`.

4. **The `DLGX` 150 byte layout.** Confirmed: a Pascal string `08 "Charcoal"` at offset 0, then
   63 big-endian shorts from offset 0x40, first = 12, nonzero at short indices 0, 4, 5, 8, 9,
   then stride-6 from 15 (values 6, 8, 9, 6, 6, 10, 10, 10 - eight values for DITL 150's items
   2..9). Beyond "a font name and a size", the schema is not one I can name with confidence from
   the material available offline, and this document deliberately does not guess. Impact is
   confined to the 9-item About box. *Settled by:* the Appearance Manager's `DLGX` documentation
   or a second sample resource.

5. **`transparent` mode's key colour.** Two sites (`ObjectDraw2.c:1403`, `ObjectDraw2.c:1430`).
   Classic QuickDraw's `transparent` (36) suppresses pixels matching the *port's background
   colour*, but some sources describe it as keying on white. Both are index 0 here, so the two
   readings coincide in Glider PRO. This is rendering.md Q11 and remains formally open, though
   practically moot. *Settled by:* a test blit with a non-white background colour.

6. **1994 `Index2Color` behaviour on an index >= 16 at depth 4.** The 69 unguarded sites (§3.11,
   §3.12) read past a 16-entry table. Whether the Color Manager clamped, wrapped, or returned
   adjacent memory was system-version dependent, which means **the original's own depth-4
   appearance at those sites was not well-defined**. This document prescribes clamping to the
   R1-nearest gray. *Settled by:* running 1.0.4 at 4-bit depth on period hardware.

7. **Which of R1, R2, R7, R9 the author actually had in mind.** All four score identically
   (17/22 combos, 55/72 sites, 11/13 pure grays) on the authored corpus. R1 and R7 are provably
   the same rule on all 256 entries; R1 (8-bit channels) and R9 (16-bit channels) disagree on
   **9** of the 256 palette entries - indices 5, 15, 38, 48, 81, 84, 114, 147, 180 - and **none of
   the 9 is an authored `i8`**, so the corpus cannot discriminate. Note that the one statement of
   Calhoun's weights, the commented-out `SetPaletteToGrays` (`MainWindow.c:449-499`), actually
   applies them to **16-bit** `RGBColor` fields, which is R9's arithmetic, not R1's; this document
   still prescribes R1 because the rest of the program's index arithmetic is 8-bit and because R1
   is the cheaper form, not because the dead code says so. *Settled by:* nothing available; the
   corpus is exhausted.

8. **Whether `(9, 94)` and `(7, 52)` are deliberate or coincidental.** Five sites and three sites
   respectively pick an `i4` one step darker than R1 predicts, and both are wood-coloured objects.
   The consistency argues deliberate contrast boosting, but that is inference. *Settled by:*
   nothing available.

9. **The `demo` 128 `padding` byte's provenance.** 109 distinct values, 407 zeros, top values
   `(0, 407) (255, 42) (32, 39) (114, 33) (111, 24)`. `LogDemoKey` (`Input.c:44-49`) never writes
   it. This is 1994 stack/heap garbage frozen into a resource and is unreproducible by design.
   Harmless: nothing reads it.

10. **Whether the shipped `demo` 128 actually completes.** The record stream covers frames
    46..3414 and `demoIndex` is unbounded, so what the original did after frame 3414 depended on
    whichever 6 bytes followed the `NewPtr(kDemoLength)` block. *Settled by:* replaying against a
    complete port and observing whether the glider dies or the banner timeout fires first.

11. **Chicago's true per-character advances.** The substitute (Liberation Sans Bold @ 13 px) was
    chosen to satisfy the 16-px line height that 76 DITL StaticText boxes assert and to fit all
    230 single-line items. It is not Chicago. *Settled by:* a `FONT`/`NFNT` resource from a
    period System file.

12. **Geneva's true per-character advances.** Same, for 9/11/13 ppem. Two strong constraints were
    used as anchors (the 64-px 9-digit score field and the 12-px room-title strip, §4.5), so the
    substitute is pinned tightly, but it is not Geneva. *Settled by:* the same source.

13. **Whether Glider PRO 1.0.4 shipped with Charcoal or Chicago as the effective system font.**
    `DLGX` 150 names Charcoal, which means the About box at least was authored on Mac OS 8+.
    Whether other dialogs picked up Charcoal at runtime depends on the user's system, not on the
    application. *Settled by:* nothing in the repository.

14. **`ictb` 150 being all 36 bytes zero.** A zero entry means "inherit", so the resource is a
    no-op. Whether it was left over from an earlier revision or intentionally neutralised is
    unknown, and does not matter.

## Porting notes

Ordered by the cost of getting it wrong.

### RNG

1. **Compute the Lehmer step in `int64` or with Schrage.** `state * 16807` overflows `int32` for
   any `state > 127773`, and the sequence reaches 282475249 on draw 2 from seed 1. A naive
   `int32` multiply-then-mod diverges within two draws and differs on **172226 of the 299999**
   seeds in [1, 299999].
2. **`RandomInt` is inclusive of `range`.** `RandomInt(n)` returns `n` when the raw word is
   `0x8000` (1 in 65536). Never translate it as `rand.Intn(n)`. Two of the 32 call sites index
   arrays with the result; the extra value is an out-of-bounds index that C tolerated and Go will
   panic on. Bounds-check, do not clamp the RNG.
3. **`Random()` returns the low 16 bits reinterpreted as `int16`, not a clamped positive.** Half
   of all draws are negative and `RandomInt` depends on the negation branch.
4. **Never let the state reach 0 or 0x7FFFFFFF** - both are absorbing.
5. **Seed 1, not time.** `Prefix.h` sets `TARGET_CARBON 1`, which compiles `Utilities.c:61` out.
   Expose the seed as a knob but default it to 1; anything claiming frame-exactness must use 1.
6. **Do not port `RandomLong`, `InitRandomLongQUS`, `RandomLongQUS` or `PourScreenOn`.** All have
   zero callers, and `PourScreenOn` contains an unbounded rejection loop.
7. **Bounds-check `demoIndex`** against 1117 records; the original read past the buffer.
8. **Ignore `demoType.padding`** but keep the record stride at 6 bytes.
9. `BUILD_ARCADE_VERSION` is 1, so any of the four gameplay keys aborts a running demo.

### srcXor

10. **XOR palette indices, never RGB channels.** The single `srcXor` (`Marquee.c:438`) blits a
    **1-bit** source into an 8-bit destination; QuickDraw colourises fg/bg first, then XORs pixel
    values. `dest ^= 0xFF` where the source bit is set.
11. **Preserve the even-XOR parity.** `DrawMarquee` calls `DrawGliderMarquee` on both the erase
    and the draw pass of every ant-march tick (`Marquee.c:493-494`). Optimising the erase pass
    away makes the glider stencil flicker.
12. **Preserve `StopMarquee`'s ordering** (`Marquee.c:139-157`): it XORs the stencil *before*
    `SetPortWindowPort` and *before* re-establishing `patXor`.
13. The seven `PAT#` 128 tiles *are* a uniform rotation of tile 0 (tile `k` = tile 0 rotated down
    `k + 1` rows, equivalently right `k + 1` pixels), but there are **seven** tiles for an
    **eight**-pixel period, so the animation steps +2, +1, +1, +1, +1, +1 and then +1 across the
    6 -> 0 wrap: one double step per cycle. Reproduce the 7 tiles literally and do not generate an
    eighth (§2.8).
14. Implement `PenMode(srcOr)` at `ObjectEdit.c:2364` as `patOr`.

### 4-bit grayscale

15. **Round, do not truncate.** `i4 = 15 - lum/17` (rule R6) scores 7 of 22 combos and 3 of 13
    pure grays. Use nearest-level matching (R1) or the equivalent `15 - (lum+8)/17`.
16. **Use `(r*3)/10 + (g*6)/10 + (b*1)/10`, as three separate integer divisions.** The
    algebraically "equivalent" `(3r+6g+b)/10` differs by up to 2 units, and the truncation is
    observable in the ramp assignment. Caveat (§3.7): the only place in the tree that states these
    weights is `SetPaletteToGrays`, which is entirely commented out and applies the divisions to
    **16-bit** channels; this note adopts the 8-bit reading, which assigns the same `i4` as the
    16-bit reading for 247 of the 256 `clut` 128 entries (they differ at indices 5, 15, 38, 48,
    81, 84, 114, 147 and 180, none of which is an authored `(i4, i8)` partner).
17. **Byte-double, do not left-shift, when widening channels.** `0xCC -> 0xCCCC`, i.e. `*0x0101`.
    `<<8` gives `0xCC00` and breaks the `gray8[]` identity test that proves R-CLUT4-1.
18. **The `i8 <-> i4` relation is not a function in either direction.** `i8` 223, 251 and 254 each
    have two authored `i4` partners. Carry all **72** authored pair sites (22 unique combinations,
    extracted from the 33 `thisMac.isDepth == 4` conditional blocks), keyed by draw call.
19. **Clamp any index >= 16 at depth 4.** There are 69 unguarded sites: 43 `ColorXxx` calls whose
    colour constant exceeds 15 (only 8 of them bare literals), plus 26
    `FrameDialogItemC(..., kRedOrangeColor8)` calls. The original's behaviour there was
    system-dependent.
20. **`kRedOrangeColor8` is 23 (`#FF6600`).** The `// actually, 18` comment is wrong (18 is
    magenta `#FF66FF`).
21. **`clut` 128 and 129 are never referenced by code** (`GetCTable` count = 0) but they are the
    authoritative statement of the 256-entry palette. Use their content; do not synthesise a
    palette.
22. **Do not port the five `DynamicMaps.c` depth-4 byte-alignment nudges** if you render
    internally at 8 bits; document the resulting 1-px sprite shift.
23. Prefer R-CLUT4-2 (render at 8, post-map for display) over a genuine 4-bit renderer. It makes
    the 69 unguarded sites well-defined for free.

### Fonts and text

24. **Model text state per drawable, not globally.** `StructuresInit.c:88-145` stamps Geneva 12
    bold onto each scoreboard GWorld once at init, and `Scoreboard.c` never sets the font again.
    A stateless text API renders the scoreboard in Chicago 12 plain.
25. **Vertical metrics matter in exactly one place** - `TETextBox` at `DialogUtils.c:642`. Every
    other baseline in the program is a hard-coded integer, because `GetFontInfo` and
    `FontMetrics` are never called.
26. **Synthetic bold is `+1 px per character`, including spaces**, applied to GEN9/GEN12/GEN14.
    Do not apply it to CHI12 (the substitute file is already bold-weight).
27. **Measure and truncate in Mac Roman bytes, not runes.** `CollapseStringToWidth` does
    `theStr[0]--` (`StringUtils.c:291`), a byte-level truncation, and Pascal strings are
    byte-counted with a 255-byte maximum.
28. **Fix `CollapseStringToWidth`'s underflow.** If `wide <= StringWidth("\p…")` the loop never
    terminates at all - the length byte cycles 0 -> 255 -> ... -> 0 forever, reading 255 bytes of
    adjacent memory on every wrap. Guard the entry and return an empty string instead (§4.16).
29. **`WrapText` wraps by character count, not pixel width** (`StringUtils.c:216-247`). Reproduce
    that, including the resulting overflows (§4.13 rows 9 and 14) - "fixing" it changes the
    banner and game-over layouts.
30. **Disable kerning, ligatures, hinting drift and fractional advances.** `SetFractEnable` is
    never called and `CharExtra`/`SpaceExtra` are never used, so advances are integers summed
    without adjustment. Any shaping engine feature shifts every centred string.
31. **`(fieldWidth - StringWidth) / 2` truncates toward zero in C.** For a 1-px overflow the
    result is 0, not -1, so an over-wide centred string starts at the left edge and clips right
    (§4.13 row 32, `SEPTEMBER`).
32. **The scoreboard drop shadow is two draws, black at (+1,+1) then white at (0,0)**, per field,
    five fields (`Scoreboard.c:136-334`; e.g. `MoveTo(1, 10)` at `:151` then `MoveTo(0, 9)` at
    `:167`). Same idiom on the splash and the high-score page.
33. **High-score row 0 is special-cased** at `dropIt - kScoreSpacing - kKimsLifted`; rows 1..9 are
    at `dropIt + i*kScoreSpacing` with `kScoreSpacing` 18, `kScoreWide` 352, `kKimsLifted` 4.
34. **Ten of the eleven negative rows in §4.13 are original overflows.** Do not "fix" them; the
    original clipped there. Only row 32 is substitute-induced.
35. Preserve the observable result of `Coordinates.c:154`'s `TextFace(applFont)` bug: the
    coordinate window renders in **Chicago 12 plain**, and the stray bold/9 lands on an unrelated
    port.
36. Preserve `GameOver.c:101-105`'s measure-before-set-font ordering if you want frame-exact
    trailer centring.
37. **No dialog item overrides the port font.** `ictb` 150 is 36 zero bytes and is the only
    `ictb`; there is one `DLGX` (150, About box). Render all 428 DITL items in CHI12.
38. `STR#` 150 strings 25, 26 and 33 end with the Mac Roman ellipsis byte `0xC9`, not `...`. The
    same byte is what `CollapseStringToWidth` appends.

### Toolbox surfaces a Go port must replace wholesale

| Toolbox facility | Used for | Go replacement |
|---|---|---|
| QuickDraw `Random()` / `qd.randSeed` | animation phase, telephone, editor | §1.16's `qdRand` |
| `CopyBits` with transfer modes | all blitting | an indexed-8 blitter with per-mode paths; only srcCopy, srcXor, patXor, patOr, transparent are needed |
| `Index2Color` / `RGBForeColor` / `GetForeColor` | every coloured draw | a 256-entry palette lookup plus an explicit fg/bg pair per drawable |
| Palette Manager / `GDevice` CLUTs | depth-dependent index meaning | R-CLUT4-2's render-at-8-then-post-map |
| `SetDepth` / `gdFlags` | the 8-vs-4 bit switch | a display-mode flag; no real mode switching |
| `GWorld` + `LockPixels` | all offscreen buffers | plain indexed-8 pixel slices; `useTempMem` fallback is irrelevant |
| `CGrafPort` text state (`txFont`/`txSize`/`txFace`) | implicit font inheritance | a per-drawable `TextState` (§4.19) |
| `DrawString` / `StringWidth` / `TETextBox` | all text | the §4.7-4.10 advance tables; `TETextBox` needs ascent |
| Resource Manager (`GetResource`, `GetIndString`, `GetIndPattern`, `Get1Resource`) | `demo`, `STR#`, `PAT#`, `clut`, `PICT`, `DITL`, `icl8` | a preparsed asset bundle; note big-endian on-disk layout throughout |
| `PAT#` / `PenPat` / `PenMode` | marching ants | 8x8 1-bit tiles, MSB-left, with a pattern-mode blitter |
| Dialog Manager (`DLOG`/`DITL`/`dctb`/`ictb`/`DLGX`) | 28 dialogs, 428 items | a layout engine driven by the parsed DITL rects |
| `GetDateTime` | seeding (dead), timestamps | `time.Now()`; note the 1904 epoch and that the value is negative as `SInt32` |
| `Str255`/`Str32`/`Str31`/`Str27`/`Str15` | every string | length-prefixed `[]byte`, Mac Roman, 255-byte cap; the shorter aliases are the *authored* limits and are why §4.13's worst cases are 15, 27 and 31 characters |

### Verification checklist for a port

38 items above are prescriptive; these nine are testable assertions a port can assert in CI.

1. `Random()` from seed 1 produces the 24 values in §1.6's "Random() as int16" column.
2. `RandomInt(n)` over all 65536 raw words reproduces §1.7's distributions exactly, including the
   single occurrence of `n`.
3. For all `i` in 0..255: `i ^ 255 == 255 - i` and `(i ^ 255) ^ 255 == i`.
4. `gray8[i] == the unique 8-bit palette index whose RGB is ((15-i)*0x1111)^3` for all
   `i` in 0..15, with all 16 indices distinct.
5. Replaying `demo` 128 with `Random()` stubbed to return 0 produces the identical glider
   trajectory as with R-RNG-1. (This is the assertion that R-RNG-2 is true in the port, not just
   in the original.)
6. The authored depth-4 substitution table has exactly **72** entries and **22** distinct
   `(i4, i8)` combinations, and lookup by `i8` is *not* single-valued for 223, 251 and 254
   (§3.5, §3.6).
7. Every `ColorXxx`-family draw with a constant colour index > 15 that is not lexically inside a
   depth conditional is routed through the depth-4 clamp: **69** sites (43 direct + 26
   `FrameDialogItemC(..., kRedOrangeColor8)`), §3.11-§3.12.
8. All **230** single-line `DITL` items fit their raw boxes under the CHI12 advance table, with
   zero overflows, and the five tightest are `Bounded` +2, `Volume:` +2, `PICT ID:` +3,
   `Direction:` +4, `Highest Score:` +5 (§4.11). Under the conservative inset assumption the
   minimum is exactly 0 (`Don't Lock`), never negative.
9. The four advance tables (§4.7-§4.10) reproduce exactly when the substitute is measured with
   **round-half-to-even**; `floor(x+0.5)` changes 40 of the 416 entries, 29 of them in GEN9
   (§4.5).
