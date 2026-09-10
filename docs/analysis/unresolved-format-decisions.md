# Glider PRO 1.0.4 — Unresolved Format Decisions: index and redirect

## What this file is

Six binary-format and build-identity questions in this analysis set were left
open in three to five documents each, in slightly different words, with
occasionally contradictory advice. They are now closed in one place:

> **[format-decisions.md](format-decisions.md)** — the normative source.
> `docs/analysis/format-decisions.md`

**Nothing in *this* file is normative.** It exists for one purpose: if you
arrive at a sibling document's `## Open questions` list and read an item that is
in fact settled, this table tells you where the ruling is, so you do not
re-litigate it. Sibling documents were left textually unchanged; where their
open-question wording survives, format-decisions.md supersedes it.

Read this file if you want:

* [§1](#1-the-six-decisions-at-a-glance) — the six one-line answers.
* [§2](#2-redirect-table-of-formerly-open-items) — every formerly-open item, cited by `document:line`,
  with its verdict and the section that closes it. 46 rows.
* [§3](#3-what-is-still-genuinely-unresolved) — the 8 residue questions that remain genuinely open,
  and the artefact that would settle each.
* [§4](#4-corrections-two-sibling-doc-claims-that-are-wrong) — two sibling-doc claims that are *wrong*, not merely open.
* [§5](#5-adjacent-items-deliberately-not-covered) — adjacent items deliberately not covered, so nobody hunts for them.
* [§6](#6-how-to-re-verify-any-of-it) — how to re-derive every number from the tree.

## 1. The six decisions at a glance

| ID | Question | Decision |
|---|---|---|
| D1 | `sizeof(game2Type)`: 110 or 114? | **110** (0x6E). The header's `// total = 114` counts the flexible array member's phantom `// 4`. `game2Type` is dead code either way. |
| D2 | Which struct-alignment model shipped? | **`align=mac68k`** (== GCC `pack(2)`): `sizeof(houseType)` = **866**, `sizeof(roomType)` = **348**, `sizeof(demoType)` = **6**. Exactly three records change size between the two models — `houseType` 866→868, `game2Type` 110→112, `demoType` 6→8; `roomType` is **348 in both** (all its members are 1 or 2 bytes wide), as are `objectType` 12, `gameType` 40, `scoresType` 292, `savedRoom` 292. Field offsets inside `houseType` (incl. `rooms` at +866) and `roomType` are alignment-invariant; `game2Type`'s are **not** — `timeStamp` and everything after it shift by 2 (74→76 … `showFoil` 109→111), which is one more reason the `.gliG` format is unportable. |
| D3 | `Sampler.binhex`'s 2 trailing bytes? | **Allocation slack** from a natural-alignment (868-byte header) build. **Accept**, **preserve** byte-for-byte, **never reject**, and **never derive `nRooms` from file length alone**. |
| D4 | Must `houseType.unusedShort` be preserved for byte-exact round-trips? | **Yes** — and so must `unusedBoolean` (+861), `hasGame` (+860), the whole 40-byte `savedGame` block (+820..859), and **every** `StrNN` tail. All are nondeterministic residue. `roomType.unusedByte` (+32) is the one exception: the original force-zeroes it. |
| D5 | What bounds `houseType.nRooms`? | No `kMaxRooms` exists. The port's normative load rule is `nRooms = min(headerNRooms, (fileLen - 866) / 348)` with `1 <= nRooms <= 32767` — note this is a *hardening* rule, not a transcription: the original's `ReadHouse` takes `nRooms` straight from the header and checks only `nRooms >= 1 && byteCount != 0` (`HouseIO.c:380-391`), and `ValidateNumberOfRooms` (editor path only, `HouseLegal.c:621-640`) *overwrites* the header value with the counted one rather than taking a minimum. Semantic ceiling **8192** addressable rooms (64 floors x 128 suites). Corpus max **531**. |
| D6 | Retail build config, and 1.0.4 vs 1.1.2? | Retail was **`BUILD_ARCADE_VERSION` = 0**; this GPL drop is a later Carbon work-in-progress that could not even link as retail. Product version is **1.1.2**; "1.0.4" is a stale comment at `GliderPRO/Sources/Main.c:3` and appears **zero** times in the resource fork. |
| D7 | What terminates demo playback? | The **game**, not the stream: death -> `countDown` 16 -> `DoDiedGameOver` -> `playing = false`. `'demo'` 128 is 6702 bytes = **1117** six-byte records, frames 46..3414. A port must add the bound the original lacks (`demoIndex < len(demo)`). Key codes are **0 = right, 1 = left, 2 = battery/helium, 3 = rubber band** — the source comments on cases 0 and 1 are swapped. |

Section anchors in format-decisions.md:

| ID | Anchor |
|---|---|
| D1 | [`#d1--sizeofgame2type-is-110-not-114`](format-decisions.md#d1--sizeofgame2type-is-110-not-114) |
| D2 | [`#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant`](format-decisions.md#d2--the-alignment-model-housetype-is-866-bytes-and-every-field-offset-is-alignment-invariant) |
| D3 | [`#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length`](format-decisions.md#d3--samplerbinhexs-two-trailing-bytes-accept-preserve-never-reject-never-size-by-file-length) |
| D4 | [`#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate`](format-decisions.md#d4--housetypeunusedshort-and-the-other-reserved-fields-preserve-verbatim-never-validate) |
| D5 | [`#d5--the-bound-on-housetypenrooms`](format-decisions.md#d5--the-bound-on-housetypenrooms) |
| D6 | [`#d6--the-retail-build-configuration-and-the-version-identity`](format-decisions.md#d6--the-retail-build-configuration-and-the-version-identity) |
| D7 | [`#d7--demo-playback-termination`](format-decisions.md#d7--demo-playback-termination) |

## 2. Redirect table of formerly open items

Sibling documents number their open questions implicitly, as ordered-list items
under `## Open questions`; porting notes likewise under `## Porting notes`. The
audit that commissioned format-decisions.md called them `OQ<n>` and `PN<n>`, and
that is the numbering used here. The **Line** column is the exact line of the
list item in the sibling document as it stands today, so you can jump straight
to it.

Verdicts:

* **closed** — answered; the sibling item should be struck through and pointed here.
* **confirmed** — the sibling document was already right; format-decisions.md
  supplies the proof it asserted without.
* **narrowed** — part is answered, a genuinely unanswerable residue remains
  (carried into [§3](#3-what-is-still-genuinely-unresolved)).
* **corrected** — the sibling document asserts something format-decisions.md
  disproves. See [§4](#4-corrections-two-sibling-doc-claims-that-are-wrong).

### 2.1 architecture.md

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| OQ1 | `:3479` | "Which version is this?" | closed | **D6** — 1.1.2. `Main.c:3` is a stale comment; `grep -c '1\.0\.4'` over `Glider PRO.r` returns **0**, `1\.1\.2` returns 3 |
| OQ10 | `:3545` | "The `idleMode` state machine is vestigial" | **corrected** | **D7** ev. 10 — `idleMode` *is* vestigial (written once at `InterfaceInit.c:162`, never read), but the claim that `kIdleSplashTicks` (7200L) is "likewise unused" is wrong: it has **9 live assignments**. It *is* the attract-mode interval |
| OQ11 | `:3554` | "Do the padding bytes matter?" | closed | **D7** ev. 4 (`demoType.padding` is `LogDemoKey` residue — 109 distinct values, 407 zeros — preserve it to round-trip) + **D6** (`prefsInfo` is 226 bytes in this build, **234** in retail, so pad offsets differ between the two) |
| OQ12 | `:3559` | "What terminates demo playback?" | closed | **D7** |
| OQ14 | `:3571` | "`game2Type` on-disk size" | closed | **D1** — 110 |
| PN13 | `:3668` | "`BUILD_ARCADE_VERSION` is 1" | **corrected** | **D6** — true of *this source*, but retail was 0. Override the note |

### 2.2 progression.md

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| OQ1 | `:5050` | "`sizeof(game2Type)` under the shipping compiler" | closed | **D1** |
| OQ2 | `:5056` | "Which build was actually shipped as 1.0.4?" | closed | **D6** — arcade = 0; and nothing about progression changes either way (`TestHighScore()` at `GameOver.c:503` and `SaveGame`/`WriteHouse` are outside every arcade conditional) |
| OQ4 | `:5072` | "`houseType.unusedShort` (offset 2) is uninitialised garbage" | closed | **D4** |
| OQ5 | `:5077` | "`roomType.unusedByte` ... `savedRoom.unusedShort`/`unusedByte`" | closed | **D4** — `roomType.unusedByte` is force-zeroed (`Room.c:173`, `HouseLegal.c:868`) and is 0 in all 4070 corpus rooms; the `savedRoom` pair is unreachable because `SaveGame2`'s body is commented out (**D1**) |
| PN3 | `:5117` | "Read the house as `866 + 348 * nRooms` bytes, but accept a longer file" | confirmed | **D2** + **D3** + **D5** — made normative, with the `Demo House` = 16526 arithmetic proof |
| PN6 | `:5130` | "`houseType.unusedShort` at offset 2 is garbage" | confirmed, extended | **D4** — also `unusedBoolean`, `hasGame`, the whole `savedGame` block, and every `StrNN` tail |

### 2.3 scoring.md

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| OQ4 | `:5238` | "What was `unusedShort` at house offset 2 for?" | closed | **D4** (value + handling); intent is [§3 item 1](#3-what-is-still-genuinely-unresolved) |
| OQ5 | `:5247` | "Why do two pairs of unrelated houses have byte-identical ..." | closed | **D4** — the shared artefact is the 40-byte `savedGame` block at +820, not the file: `CD Demo House` and `Slumberland` carry identical blocks (`timeStamp` 2875627985 = 1995-02-14 17:33:05, `score` 28200), as do `Nemo's Market` and `SpacePods` (2881122394 = 1995-04-19 07:46:34, `score` 25600). The **files are not duplicates** (72554 vs 134150 bytes; 44018 vs 140762; no two data forks in the corpus are byte-equal). Each pair descends from a common template file that was then edited heavily, and the residue block was never rewritten |
| OQ7 | `:5273` | "Does the `game2Type` header occupy 110 or 114 bytes?" | closed | **D1** — 110 |
| OQ12 | `:5304` | "`vers` says 1.1.2, the task brief says 1.0.4" | closed | **D6** — use 1.1.2 |
| OQ24 | `:5383` | "The `Sampler` house's data fork is 2 bytes longer" | closed | **D3** |
| OQ25 | `:5387` | "`houseType.unusedBoolean` at offset 861 holds 255, 30, 37, 185, 30, ..." | closed | **D4** — heap residue; preserve verbatim |

### 2.4 house-format.md

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| §12.4 | `:3653` | "The `Sampler` 2-byte tail" | closed | **D3** |
| OQ8 | `:4080` | "`savedGame.roomNumber` is a room index that `CompressHouse` does not patch" | narrowed | **D4** — moot for 20 of 22 houses (the block is residue) and unreachable in code (`QueryResumeGame` at `Menu.c:710` has **zero** call sites; `OpenSavedGame` returns `false` at `SavedGames.c:169`). Residue: [§3 item 2](#3-what-is-still-genuinely-unresolved) |
| PN2 | `:4101` | "Parse rooms at `866 + 348*r`, not at `sizeof(struct)*r`" | confirmed | **D2** |
| PN19 | `:4155` | "`state` bytes, `visited`, and the whole `savedGame` block when ..." | confirmed, extended | **D4** |
| PN20 | `:4160` | "`unusedShort`, `unusedBoolean`, and the 10 data bytes of an empty object slot" | confirmed, extended | **D4** |

### 2.5 object-taxonomy.md

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| OQ1 | `:2911` | "`Sampler.binhex` is 2 bytes longer than `866 + nRooms * 348`" | closed | **D3** |
| PN3 | `:2967` | "Room record: 348 bytes, objects at +60 ... House header: 866 bytes" | confirmed | **D2** |
| PN9 | `:2991` | "Round-trip test: decode and re-encode all 22 shipped houses and require byte equality" | narrowed | **D3** + **D4** — the test is achievable, but only if trailing slack **and all** `StrNN` tails are preserved. A naive Pascal-string-to-`string` round-trip fails **22 of 22** houses |

### 2.6 interactions.md

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| OQ5 | `:4724` | "`Sampler.binhex`'s 2 extra bytes" | closed | **D3** |

### 2.7 constants.md

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| OQ1 | `:7930` | "Is the `floor + 7` in `CheckDuplicateFloorSuite` an off-by-one?" | closed | **D5** corollary — **no**. `bitPlace = ((floor + 7) * 128) + suite` spans exactly 0..8191 over the legal domain, filling `kRoomsTimesSuites` 8192 with no slack. The bug is the **call order** (`HouseLegal.c:1089` before `:1110`), not the bias |
| OQ5 | `:7964` | "Are `unusedShort` / `unusedBoolean` / `unusedByte` / `unusedLong` vestigial or ..." | closed | **D4** — vestigial *and* uninitialised; per-field table there |
| OQ8 | `:7986` | "Was `game2Type` / the external `.gliG` saved game ever functional?" | closed | **D1** — no. `SaveGame2`'s body is commented out (`SavedGames.c:33-147`) and `OpenSavedGame` returns `false` at `:169` before its body |
| OQ9 | `:7994` | "Why does `Sampler.binhex` have 2 trailing bytes?" | closed | **D3** |
| OQ12 | `:8012` | "Does the `BUILD_ARCADE_VERSION 1` / `COMPILENOCP` / `COMPILEQT` configuration ..." | closed | **D6** — per-macro table in its Decision |
| OQ14 | `:8027` | "Is there any bound on `houseType.nRooms` at load time?" | closed | **D5** |

### 2.8 original-houses.md

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| OQ1 | `:4667` | "What exactly does `unusedShort` at `houseType` offset 2 hold?" | closed | **D4** |
| OQ2 | `:4672` | "What was `roomType.unusedByte` (offset 32) for?" | narrowed | **D4** — its *value* is settled (0 on disk in all 4070 rooms, force-zeroed at `Room.c:173` and `HouseLegal.c:868`); its original *purpose* is not recoverable. [§3 item 6](#3-what-is-still-genuinely-unresolved) |
| OQ8 | `:4715` | "Which of the 22 houses were shipped with the retail 1.0.4 disk versus added ..." | narrowed | **D6** — "retail 1.0.4" is the wrong frame (retail was 1.1.2, (c) 1994-95). **D4**'s shared-`savedGame` finding plus the 2000-05-11 activity give partial provenance: `Sampler`'s *house* `timeStamp` is 2000-05-11 19:52:08 (every other house body dates to 1995), while `Slumberland`'s house stamp stays at 1995-08-13 and it is only its `highScores.timeStamps[0]` (2000-05-11 11:50:08) that lands on that day. [§3 item 3](#3-what-is-still-genuinely-unresolved) |
| PN1 | `:4730` | "`houseType` is 866 bytes" | confirmed | **D2** |

### 2.9 resource-fork.md

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| OQ5 | `:3888` | "The About box's copyright disagrees with `'vers'`" | closed | **D6** — `DITL` 150 says `(c) 1994-2000` (`Glider PRO.r:5178-5180`), `'vers'` says `(c) 1994-95` (`:6135-6140`). The `DITL` was updated during the Carbon port, the `'vers'` was not; the running About box shows **both** strings, since item 2 is a zero-length `statText` filled at runtime from `'vers'` 1's long string (`About.c:55-61`) |
| OQ6 | `:3893` | "The fork claims version 1.1.2, the source release is called 1.0.4" | closed | **D6** — use 1.1.2 |

### 2.10 input.md

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| OQ1 | `:3010` | "Is the swapped `// left key` / `// right key` comment pair in `GetDemoInput` ..." | closed | **D7** ev. 3 — **yes, the comments are swapped.** `GetInput`'s recorder call sites are the authority because they produced the shipped bytes: `LogDemoKey(0)` in the rightKey branch (`Input.c:304`), `(1)` in leftKey (`:321`), `(2)` battery (`:334`), `(3)` band (`:348`) |
| OQ6 | `:3038` | "Does any house rely on the out-of-bounds `demoData[demoIndex]` read past 1117 records?" | closed | **D7** ev. 6, 8 — no house is involved. Once `demoIndex` reaches 1117 the compare at `Input.c:224` re-reads `demoData[1117].frame` (bytes 6702..6705 of a 6702-byte `NewPtr` block) on **every** frame from 3415 until `gameOver` stops `GetDemoInput` being called at all (`Play.c:475-483`) — not once. It happens in every demo and is benign only because Mac `NewPtr` blocks carry slack. In Go it is a panic |
| PN16 | `:3136` | "Prefs are 226 bytes with two uninitialised pad bytes" | confirmed | **D6** — 226 verified by compilation; **retail's was 234**, with `prefVersion` at offset 172 instead of 164, because `long encrypted, fakeLong` is commented out of `prefsInfo` at `Externs.h:240` in this drop |
| PN17 | `:3141` | "The demo stream is 1117 six-byte big-endian records" | confirmed, extended | **D7** — plus the frame range (46..3414), the key histogram, the gap histogram, and the fact that 1117 depends on `sizeof(demoType)` being 6 not 8, which is **D2** |

### 2.11 editor.md

editor.md uses `### Q<n>` headings rather than an ordered list.

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| Q4 | `:11406` | "Why does `Sampler` have 2 trailing bytes and an empty resource fork?" | closed (bytes) / narrowed (fork) | **D3** for the trailing bytes. The empty resource fork is a separate, unrelated question and is not addressed (its 286 bytes are a bare Mac resource header + empty map, i.e. zero resources) |
| Q10 | `:11479` | "What did the reserved fields hold?" | closed | **D4** |

### 2.12 structs.md

| Item | Line | First words | Verdict | Ruling |
|---|---|---|---|---|
| §2 row `game2Type` | `:255` | "110 / 112 / yes" | confirmed | **D1** — structs.md was already right. The other four documents were not |

### 2.13 Roll-up by decision

| Decision | Closes | Confirms | Narrows | Corrects |
|---|---|---|---|---|
| D1 | architecture.md OQ14, progression.md OQ1, scoring.md OQ7, constants.md OQ8 | structs.md §2 | — | — |
| D2 | — | house-format.md PN2, original-houses.md PN1, progression.md PN3, object-taxonomy.md PN3 | — | — |
| D3 | house-format.md §12.4, object-taxonomy.md OQ1, interactions.md OQ5, editor.md Q4, constants.md OQ9, scoring.md OQ24 | — | object-taxonomy.md PN9 | — |
| D4 | original-houses.md OQ1, progression.md OQ4 + OQ5, scoring.md OQ4 + OQ5 + OQ25, constants.md OQ5, editor.md Q10 | progression.md PN6, house-format.md PN19 + PN20 | house-format.md OQ8, original-houses.md OQ2, object-taxonomy.md PN9 | — |
| D5 | constants.md OQ1 + OQ14 | — | — | — |
| D6 | architecture.md OQ1, progression.md OQ2, constants.md OQ12, resource-fork.md OQ5 + OQ6, scoring.md OQ12 | input.md PN16 | original-houses.md OQ8 | architecture.md PN13 |
| D7 | architecture.md OQ11 + OQ12, input.md OQ1 + OQ6 | input.md PN17 | — | architecture.md OQ10 |

Totals: **46** items touched across 12 documents — **30 closed**, **10 confirmed**,
**4 narrowed**, **2 corrected**. editor.md Q4 is counted as closed, because the
half of it that was open in five other documents (the trailing bytes) is closed;
its unrelated second half (the empty resource fork) is out of scope here.

## 3. What is still genuinely unresolved

These are the residue. They are all *historical* or *provenance* questions; none
of them blocks a byte-exact port, because every one has a normative handling
rule in format-decisions.md even though the underlying fact is unknown. The full
statements, with the evidence that exists for each, are at
[format-decisions.md `## Open questions`](format-decisions.md#open-questions).

| # | Residue | Would be settled by | Blocks a port? |
|---|---|---|---|
| 1 | What `houseType.unusedShort` (+2) was *intended* for. D4 settles what it holds and what to do with it. | an earlier revision of `GliderStructs.h`, or a pre-PRO Glider 3.0 file format | no — preserve verbatim |
| 2 | Whether `savedGame.roomNumber` is stale in the two houses that carry a real save (`ImagineHouse PRO II`, `Titanic`). | a working `QueryResumeGame` call path, which does not exist | no — the block has no readers |
| 3 | Which of the 22 houses shipped on the retail disk. | the retail CD-ROM or its catalogue | no |
| 4 | Whether `GliderPRO/Houses/Demo House.binhex` is bit-for-bit the file `'demo'` 128 was recorded against. Its house timeStamp is 1995-08-13; the app's `DITL` was updated in 2000. | a recording-era build, or replaying the demo and comparing | no, but it gates item 8 |
| 5 | Which compiler and project settings produced `Sampler`'s 868-byte header. | the CodeWarrior project file, absent from the GPL drop | no |
| 6 | Why `roomType.unusedByte` needed force-zeroing in **two** places (`Room.c:173` and `HouseLegal.c:868`). | version history | no |
| 7 | Whether `'demo'` 128's `padding` residue encodes anything recoverable about the recording machine. | nothing available; it is stack/heap noise by construction | no — preserve verbatim |
| 8 | The exact frame at which the demo glider dies. Frame 3414 is the last *input*, not the death. | running the demo against `Demo House` in a faithful port | no — this is a port acceptance test, not a prerequisite |

## 4. Corrections: two sibling-doc claims that are wrong

Not open questions — errors. Both are in architecture.md.

**1. architecture.md OQ10 (`:3545`)** says the `idleMode` state machine is
vestigial and that `kIdleSplashTicks` is "likewise unused", with the attract
timer "seeded with a different interval".

Half right. `idleMode` (`GliderPRO/Sources/Events.c:30`) is indeed written once
(`InterfaceInit.c:162`) and never read. But `kIdleSplashTicks` (7200L,
`GliderDefines.h:197`) has **nine live assignments** and *is* the attract
interval — 7200 ticks / 60 = **120 s**:

| Site | Context |
|---|---|
| `GliderPRO/Sources/AppleEvents.c:111` | after an Apple Event |
| `GliderPRO/Sources/Events.c:427` | on any user event |
| `GliderPRO/Sources/InterfaceInit.c:163` | at startup |
| `GliderPRO/Sources/Menu.c:336` | menu dismissal |
| `GliderPRO/Sources/Menu.c:402` | menu dismissal |
| `GliderPRO/Sources/Menu.c:419` | menu dismissal |
| `GliderPRO/Sources/Menu.c:424` | menu dismissal |
| `GliderPRO/Sources/Play.c:277` | end of a real game |
| `GliderPRO/Sources/Play.c:302` | end of a demo game |

and the trigger is `GliderPRO/Sources/Events.c:536-539`:

```c
if ((theMode == kSplashMode) && doAutoDemo && !switchedOut)
{
    if (TickCount() >= incrementModeTime)
        DoDemoGame();
}
```

Note what is *not* in that condition: `demoHouseIndex != -1`. With no house
named `Demo House` in the search path, `DoDemoGame` (`Play.c:282-303`) sets
`thisHouseIndex = demoHouseIndex` = -1 and the house loader indexes
`theHousesSpecs[-1]`. A Go port must guard this. See D7.

**2. architecture.md PN13 (`:3668`)** states, as a porting instruction, that
`BUILD_ARCADE_VERSION` is 1 and that the arrow keys are therefore remapped to
menu actions. The macro value is correct (`GliderDefines.h:16`), but the
instruction is wrong: retail shipped with 0. Treat the arcade paths as an
optional kiosk mode. See D6.

## 5. Adjacent items deliberately not covered

Listed so nobody searches format-decisions.md for them.

| Document | Item | Line | Why not |
|---|---|---|---|
| house-format.md | OQ1 "No version-1 house exists in the corpus" | `:4045` | Needs a pre-2.0 house file; none exists in this repository. `ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746-820`) is the only surviving description of the v1 layout |
| editor.md | Q9 "What was `kNewHouseVersion` (`0x0300`) for?" | `:11471` | `0x0300` (`GliderDefines.h:518`) is only ever compared against (`HouseIO.c:395`), never written. No 3.0 format exists to document |
| resource-fork.md | OQ7 "`'PICT'` 10000 (4,182 bytes)" | `:3899` | A graphics question, unrelated to the six items |
| scoring.md | OQ1 "Was `houseIsReadOnly` ever true in a shipping build?" | `:5192` | `IsFileReadOnly` returns `false` unconditionally (`HouseIO.c:659-663`, rest of the body commented out); a behaviour question, not a format one |
| constants.md | OQ11 "Is `evenFrame = true` at `Dynamics3.c:474`/`:524` truly ..." | `:8005` | Physics/determinism question |
| original-houses.md | OQ4 "`Fun House`'s room 29 `where = 5812`" | `:4687` | Link-encoding data question; see original-houses.md |

## 6. How to re-verify any of it

Everything in format-decisions.md was derived from the tree in front of it; no
external reference was used. To reproduce:

1. **Normalise line endings.** Every `.c`/`.h` under `GliderPRO/` is CR-only
   (0x0D). All citations are line numbers in the `tr '\r' '\n'` copy:

   ```bash
   mkdir -p /tmp/wf-fmtdec
   for f in GliderPRO/Sources/*.c \
            GliderPRO/Headers/*.h; do
       tr '\r' '\n' < "$f" > "/tmp/wf-fmtdec/$(basename "$f")"
   done
   ```

2. **Use `command grep -an`, not `grep`.** The sources contain MacRoman high
   bytes -- across all of `Sources/*.c` + `Headers/*.h` exactly five distinct
   ones occur: 0xA5 `*` bullet (552x), 0xC9 `...` ellipsis (219x), 0xD6 `/`
   division sign (9x), 0xC4 `f` florin (3x) and 0xCA non-breaking space (1x).
   (0xAA `(tm)` and 0xA9 `(c)` appear in `Glider PRO.r`, *not* in the sources.)
   So any tool that auto-detects
   binary files -- `ugrep -I`, and any `grep` shell function wrapping it --
   reports "no match" for constants that are plainly present, with exit status 1
   and no output. This produced false negatives during the investigation and is
   worth stating twice.

3. **Compile the structs, do not eyeball them.** `/tmp/wf-fmtdec/sizes.c`
   transcribes every persisted record twice, once under `#pragma pack(push,2)`
   and once under natural alignment, and prints `sizeof`, `_Alignof` and a full
   `offsetof` map. `FSSpec` must be forced to `pack(2)` in **both** variants,
   because Apple's `Files.h` wraps its own declaration in `align=mac68k`: a Mac
   `FSSpec` is 70 bytes (2 + 4 + 64) in every compilation mode, never 72.
   Getting that one wrong is the easiest way to mis-size `game2Type`.

4. **Parse the shipped files.** `tools/probe_house.py`
   decodes BinHex 4.0. Note two things: `binhex_decode(path)` defaults to
   `check_crc=False`, so pass `check_crc=True` to actually validate (all 22
   houses pass), and the returned dict's resource-fork key is `rsrc`, not `res`.

5. **The corpus invariants** any re-derivation should reproduce: 22 houses, all
   `version == 0x0200`, **4070** rooms total, `nRooms` from 2 (`Sampler`) to 531
   (`Teddy World`), 7 unlocked / 15 locked (`timeStamp & 1`, `HouseIO.c:402`),
   `flags` in {0x00 x**14**, 0x02 x**7**, 0x06 x1 (`Art Museum`)} -- so bit 0
   (`wardBit`) is never set on disk -- `houseType.unusedShort` non-zero in 10 of
   22 and `unusedBoolean` non-zero in 7 of 22, `hasGame == 1` in exactly 2
   (`ImagineHouse PRO II`, `Titanic`), `roomType.unusedByte == 0` in all 4070
   rooms, `StrNN` tail residue in **22 of 22** houses, and **21 of 22** data
   forks exactly `866 + 348 * nRooms` bytes -- `Sampler` alone is +2 (1564 vs
   1562, trailing bytes `01 01`; its rooms still start at +866, as
   `offsetof(houseType, rooms)` is 866 under *both* alignment models).
