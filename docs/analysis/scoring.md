# Glider PRO Scoring, Lives, Inventory, Scoreboard and High Scores

## Scope

This document specifies, exhaustively, the *bookkeeping* half of Glider PRO
(John Calhoun, 1994; GPLv2 source release) — everything that answers the
questions "how many points do I have", "how many gliders do I have left", "how
much battery / helium / rubber band / foil do I have left", "what does the strip
at the edge of the screen say", "have I won yet", and "did I make the top ten" —
so that the Go port can reproduce it without reference to the original binary,
to Inside Macintosh, or to guesswork.

It covers:

- the exact set of global variables that constitute the player's score and
  inventory, where each is defined, and every single site in the tree that reads
  or writes each one (mechanically enumerated with `grep -a`, not sampled);
- the point value of every scoring event in the game — there are exactly six
  distinct sources of points and exactly seven `theScore` mutation sites — plus
  independent corroboration from the level editor's own
  `CountTotalHousePoints()` total-points calculator;
- the room-visit rule, including the non-obvious detail that a room is scored
  when the glider **leaves** it, not when it enters, so the first room of a
  house is worth nothing until it is re-entered;
- the "flying points" animation that accompanies a scoring pickup: its 15-frame
  sprite sheet decoded from `PICT 4006` pixel by pixel, its unreachable
  250-point range, and its exact 72-frame lifetime;
- the lives system: `kInitialGliders 2` (i.e. three lives), the doubling in
  two-player mode, the *only* extra-life source in the game (the sheet of
  paper, `kPaper`, object code `0x25`), the complete death-and-respawn routine
  `OffAMortal()`, and the fact that **there are no score-threshold extra lives
  whatsoever**;
- the four consumables — battery, helium, rubber bands, aluminium foil — sharing
  three `short` counters, with every supply amount, every drain rate, every
  frame-parity subtlety, and the microwave bitmask that wipes them;
- the scoreboard: five offscreen GWorlds, their exact pixel sizes, the two
  vertical layout modes, all 11 destination rectangles with numeric offsets, the
  scoreboard artwork `PICT 1997` decoded column by column with the observed
  palette indices, the badge sheet `PICT 1996`, the two-pass drop-shadow text
  rendering, the eight-phase low-supply flash cycle, and a worked example for a
  640x480 and a 1024x768 screen;
- the question "is there a game timer" (answer: no — and a complete enumeration
  of every counter that might be mistaken for one);
- the high-score table: the 292-byte `scoresType` struct, its **verified**
  location at byte offset 528 inside the house file's data fork (hexdumped from
  three shipped houses), the ten-slot insertion and sort algorithm and its
  score-only ordering quirk, the two modal dialogs (DITL/DLOG dumps included),
  the on-screen layout of the high-score display with pixel positions, the
  dead-code `G-PRO Scores ƒ` side-car file, and what the preferences file does
  and does not store;
- the two end-of-game sequences — `DoGameOver()` (you win: trailer text over the
  Milky Way, with an angel and falling stars) and `DoDiedGameOver()` (you lose:
  eight sheets of paper tumble down and spell a word) — frame by frame;
- how all of the above is serialised into `gameType` for save/resume, and why
  resuming a saved game makes you ineligible for the high scores.

It does **not** cover: the physics of the glider itself (see
`player-physics.md`), the object/hot-spot dispatch machinery that *calls*
`HandleRewards()` (see `interactions.md`), the dirty-rectangle blitter that the
scoreboard and flying points feed (see `rendering.md`), or the house file format
outside the `scoresType` and `gameType` regions (see `house-format.md`).

Every line-number citation is of the form `GliderPRO/Sources/Player.c:123` and
is relative to the repository root. The shipped `.c`/`.h` files use
classic-Mac CR-only line endings; all line numbers were taken from copies
converted with `tr '\r' '\n'`, which is a one-for-one byte substitution and
therefore preserves line numbering exactly. `GliderPRO/Glider PRO.r` is already
LF-terminated and is cited directly.

Every claim about binary layout in this document was produced by parsing the
actual bytes with `python3` — the house files were BinHex-decoded and hexdumped,
and the `PICT`/`DITL`/`DLOG`/`ALRT`/`STR#` resources were extracted from the
derez'd `Glider PRO.r` and decoded — and the observed values are shown inline.
Nothing about a byte offset in this document is inferred from the C struct
declaration alone.

## Sources read

Read in full (line counts are from the CR->LF converted copies):

| File | Lines | Why |
|---|---:|---|
| `GliderPRO/Sources/Scoreboard.c` | 457 | the whole scoreboard |
| `GliderPRO/Sources/HighScores.c` | 856 | the whole high-score subsystem |
| `GliderPRO/Sources/GameOver.c` | 507 | both end-of-game sequences |
| `GliderPRO/Sources/Player.c` | 1605 | `theScore` definition, `OffAMortal()`, foil modes |
| `GliderPRO/Headers/Scoreboard.h` | 16 | 5 externs, nothing else |
| `GliderPRO/Headers/GameOver.h` | 12 | 2 externs, nothing else |
| `GliderPRO/Headers/GliderStructs.h` | 348 | `scoresType`, `gameType`, `houseType`, `flyingPtType`, `bonusType` |
| `GliderPRO/Headers/GliderDefines.h` | 626 | every scoring/inventory/scoreboard constant |
| `GliderPRO/Sources/Banner.c` | 237 | `CountStarsInHouse()`, `DisplayStarsRemaining()` |
| `GliderPRO/Sources/Prefs.c` | 282 | where the remembered name/banner live |

Read in the regions that touch this subsystem (all call sites were located
mechanically with `grep -arn`, so the enumerations below are complete, not
sampled):

| File | Regions | Why |
|---|---|---|
| `GliderPRO/Sources/Interactions.c` | 13-19, 79-94, 154-167, 530-546, 756-981, 1159-1196, 1218-1248, 1324-1379 | `HandleRewards()` — every point and every consumable |
| `GliderPRO/Sources/Play.c` | 18-31, 47-68, 74-186, 307-365, 430-598, 603-640 | `NewGame()`, `InitGlider()`, `PlayGame()`, `SetObjectsToDefaults()` |
| `GliderPRO/Sources/Input.c` | 14-19, 121-183, 186-263, 325-370 | battery/helium/band consumption |
| `GliderPRO/Sources/Transit.c` | 155, 319, 357, 396, 428-469 | `HandleRoomVisitation()` |
| `GliderPRO/Sources/StructuresInit.c` | 26, 39, 50, 59-163, 389-416 | every scoreboard and flying-point rectangle |
| `GliderPRO/Sources/StructuresInit2.c` | 37, 153-160, 195-235, 287-295 | `flyingPoints` allocation, `workSrcRect` |
| `GliderPRO/Sources/InterfaceInit.c` | 161, 196-218 | `houseRect`, `playOriginH/V`, `localRoomsDest[]` |
| `GliderPRO/Sources/MainWindow.c` | 40, 68-80, 157-169, 212-260 | window geometry, `UpdateMenuBarWindow()` |
| `GliderPRO/Sources/Render.c` | 52, 65, 320-380, 673-691 | `RenderFlyingPoints()`, `InitGarbageRects()` |
| `GliderPRO/Sources/DynamicMaps.c` | 20-40, 195-254 | `AddFlyingPoint()`, `pointsSrc[15]` |
| `GliderPRO/Sources/House.c` | 133, 164-190, 505-531 | `RealRoomNumberCount()`, `CountRoomsVisited()` |
| `GliderPRO/Sources/HouseIO.c` | 34-36, 187, 315-530, 659-670 | when the house (and therefore the score table) is written |
| `GliderPRO/Sources/HouseInfo.c` | 20-24, 43-121, 319-342 | `CountTotalHousePoints()`, "clear scores" |
| `GliderPRO/Sources/SavedGames.c` | 14-20, 30-300, 303-352 | `gameType` serialisation |
| `GliderPRO/Sources/Menu.c` | 32-33, 49-90, 301-330, 366-425, 779-790 | menu gating, `resumedSavedGame`, alert 1046 |
| `GliderPRO/Sources/Modes.c` | 27, 52, 406-430, 581-630 | foil mode entry/exit, `kFramesToBurn` |
| `GliderPRO/Sources/Dynamics.c` | 34-72 | the one `evenFrame`-gated foil decrement |
| `GliderPRO/Sources/RubberBands.c` | 12, 256-290 | `AddBand()` |
| `GliderPRO/Sources/RoomGraphics.c` | 27-33, 44-130, 417 | `numNeighbors`, `localNumbers[]` |
| `GliderPRO/Sources/Room.c` | 174 | `visited = false` on new room |
| `GliderPRO/Sources/Utilities.c` | 317-349 | `LoadGraphic()`, `LoadScaledGraphic()` |
| `GliderPRO/Sources/RectUtils.c` | 57-63 | `ZeroRectCorner()` |
| `GliderPRO/Sources/ColorUtils.c` | 20-45 | `ColorText()`, `ColorRect()` |
| `GliderPRO/Sources/DialogUtils.c` | 24-40, 105-130 | `BringUpDialog()`, the disabled `CenterDialog()` |
| `GliderPRO/Sources/StringUtils.c` | 319-326 | `GetLocalizedString()` |
| `GliderPRO/Sources/Settings.c` | 888, 1131-1190, 1257 | `numNeighbors` |
| `GliderPRO/Sources/Main.c` | 111, 133-134, 146-192 | prefs defaults, `numNeighbors` clamp |
| `GliderPRO/Sources/Environ.c` | 60, 334-342, 463, 542 | `thisMac.screen`, `thisMac.isDepth` |
| `GliderPRO/Sources/ObjectInfo.c` | 1947-2050 | the invis-bonus editor constraint |
| `GliderPRO/Sources/ObjectAdd.c` | 244, 264, 278, 292 | default `bonusType.points` |
| `GliderPRO/Sources/Sound.c` | 14-15, 40-120, 279, 327, 501 | `snd ` id = sound number + 1000 |
| `GliderPRO/Headers/Externs.h` | 15, 198-205, 232-267, 338 | `kPreferredDepth`, `prefsInfo`, menu items |
| `GliderPRO/Headers/GliderProtos.h` | 87-93, 149, 375, 430 | prototypes |
| `GliderPRO/Headers/GliderVars.h` | 50 | `mainWindowRect`, `houseRect` |
| `GliderPRO/Headers/Objects.h` | 14-15 | `pointsSrcMap`, `pointsMaskMap` |

Binary data actually parsed:

| Artefact | What was done |
|---|---|
| `GliderPRO/Houses/*.binhex` (22 files) | BinHex 4.0-decoded; `houseType` header and `scoresType` block parsed and hexdumped; all 24 object slots of all 4 268 rooms walked to tally prize objects and read `bonusType.points` |
| `GliderPRO/Glider PRO.r` `PICT 1997` | full QuickDraw v2 PackBitsRect decode: 1536x20, 8-bit, 256-entry CLUT, 20 PackBits rows; per-column profile computed |
| `PICT 1996` | full decode: 32x66 badge sheet; confirmed left half blank / right half icons |
| `PICT 4006` | full decode: 24x120; 15 frames read as glyphs; palette indices per frame |
| `PICT 5006` | hand-decoded QuickDraw **v1** PICT (single-byte opcodes) 1-bit mask for `PICT 4006` |
| `PICT 1994`, `1998`, `1995`, `1017`, `1018`, `1021`, `1988`-`1993` | `picFrame` and size read |
| `DITL 1020`, `1021`, `1046`, `1032`, `1015` | fully decoded item lists |
| `DLOG 1020`, `1021`; `ALRT 1032`, `1046` | fully decoded |
| `STR# 150`, `STR# 160` | fully decoded |
| `snd ` 1003-1061 | names and sizes read |
| `vers` 1, `vers` 2, `ozm5` 0, `demo` 128 | decoded |

## 0. Conventions and prerequisites

A handful of facts from elsewhere in the codebase are load-bearing for
everything below. They are restated here so this document stands alone.

### 0.1 The frame clock

`kTicksPerFrame` is 2 (`GliderPRO/Headers/GliderDefines.h:533`). A Mac tick is
1/60 s, so the game runs a fixed 30 frames per second. `gameFrame` is a `long`
incremented once per frame (`GliderPRO/Sources/Play.c:435`, definition at
`GliderPRO/Sources/Play.c:51`), reset to `0L` by `NewGame()`
(`GliderPRO/Sources/Play.c:112`). `evenFrame` is a `Boolean` toggled every frame
(`GliderPRO/Sources/Play.c:436`, definition at `GliderPRO/Sources/Play.c:53`).
Both are used by this subsystem: `gameFrame & 0x00000007` drives the scoreboard
low-supply flash cycle, and `evenFrame` halves one of the six foil drain rates.

### 0.2 Rectangles

Mac `Rect` is `{short top, left, bottom, right}` in that order, and this
document writes rectangles as `(top, left, bottom, right)` to match. Width is
`right - left`, height is `bottom - top`. `RectWide()`/`RectTall()` are the
game's helpers. `QSetRect(&r, left, top, right, bottom)` — note the argument
order is **l, t, r, b**, the opposite grouping from the struct. `QOffsetRect(&r,
dh, dv)` adds `dh` to left/right and `dv` to top/bottom.

`ZeroRectCorner()` (`GliderPRO/Sources/RectUtils.c:57-63`) normalises a rect to
the origin while preserving its size:

```c
theRect->right -= theRect->left;
theRect->bottom -= theRect->top;
theRect->left = 0;
theRect->top = 0;
```

### 0.3 Offscreen buffers and colour depth

`kPreferredDepth` is 8 (`GliderPRO/Headers/Externs.h:15`). Every graphics buffer
in this subsystem except the 1-bit masks is an 8-bit indexed GWorld using the
standard Mac 256-colour system palette. I verified by aggregating the colour
tables of every 8-bit `PICT` in the application that **all of them ship the same
256-entry CLUT with zero conflicts**, so a Go port can hard-code one palette
table and index into it. The specific indices this subsystem depends on are
tabulated in section 5.4.

`thisMac.isDepth` (`GliderPRO/Sources/Environ.c:542`) is the actual screen depth
and is tested in exactly one place inside this subsystem — the scoreboard's
background paint — where 4 (16-level grayscale) selects a different palette
index. See section 5.6.

### 0.4 Pascal strings

All names and banners are Mac Pascal strings: a length byte followed by that
many `mac-roman` bytes, in a fixed-size array, **not** NUL-terminated and with
no guarantee about the bytes after the last character. The relevant types:

| Type | Array size | Max chars |
|---|---:|---:|
| `Str15` | 16 | 15 |
| `Str27` | 28 | 27 |
| `Str31` | 32 | 31 |
| `Str255` | 256 | 255 |

The padding bytes are *stale garbage in real files* — see the hexdumps in
section 7.3, where a house's `highScores.banner` field has 12 bytes of unrelated
data after the string. A Go port must slice by the length byte and must not
assume the tail is zeroed.

### 0.5 On-disk byte order and struct packing

The house file is a raw memory image of the in-RAM `houseType`, written by a
single `FSWrite` of the whole handle. It is therefore **big-endian**, and laid
out per `#pragma options align=mac68k` — which for this subsystem means
"everything is at its natural alignment with no trailing padding beyond what the
2-byte fields imply". Every offset asserted in this document was confirmed by
hexdumping real files (section 7.3).

### 0.6 A note on `grep`

`file` reports the CR-only C sources as "C source, ISO-8859 text" and GNU grep
therefore treats them as binary and *suppresses match output*. Every enumeration
in this document was produced with `grep -a`. This is not a stylistic note: a
first pass without `-a` wrongly showed only one `theScore +=` site in the whole
tree, and the missing five are the entire prize-scoring table.

## 1. The state this subsystem owns

### 1.1 The variables

| Variable | Type | Defined at | Meaning |
|---|---|---|---|
| `theScore` | `long` | `GliderPRO/Sources/Player.c:50` | the player's score |
| `displayedScore` | `long` | `GliderPRO/Sources/Scoreboard.c:40` | the score currently *shown*, which lags `theScore` |
| `doRollScore` | `Boolean` | `GliderPRO/Sources/Scoreboard.c:42` | whether the display animates towards `theScore` |
| `mortals` | `short` | `GliderPRO/Sources/Play.c:52` | **spare** gliders, not total gliders |
| `batteryTotal` | `short` | `GliderPRO/Sources/Play.c:52` | signed: `>0` battery charge, `<0` helium charge, `0` empty |
| `bandsTotal` | `short` | `GliderPRO/Sources/Play.c:52` | rubber bands held |
| `foilTotal` | `short` | `GliderPRO/Sources/Play.c:52` | aluminium-foil hit points remaining |
| `numStarsRemaining` | `short` | `GliderPRO/Sources/Banner.c:28` | stars not yet collected in this house |
| `gameOver` | `Boolean` | `GliderPRO/Sources/GameOver.c:46` | set once, ends the game after a countdown |
| `countDown` | `short` | `GliderPRO/Sources/GameOver.c:45` | frames left before the end sequence runs |
| `wasScoreboardMode` | `short` | `GliderPRO/Sources/Scoreboard.c:41` | `kScoreboardHigh` / `kScoreboardLow` |
| `numFlyingPts` | `short` | `GliderPRO/Sources/DynamicMaps.c:31` | active flying-point sprites, 0..3 |
| `flyingPoints` | `flyingPtPtr` | `GliderPRO/Sources/DynamicMaps.c:25` | heap array of 3 `flyingPtType` |
| `highName` | `Str15` | `GliderPRO/Sources/HighScores.c:46` | remembered player name, persisted in prefs |
| `highBanner` | `Str31` | `GliderPRO/Sources/HighScores.c:45` | remembered banner, persisted in prefs |
| `lastHighScore` | `short` | `GliderPRO/Sources/HighScores.c:47` | index of the row to highlight, or -1 |
| `keyStroke` | `Boolean` | `GliderPRO/Sources/HighScores.c:48` | dialog filter flag for the live character counter |
| `gameDirty` | `Boolean` | `GliderPRO/Sources/HouseIO.c:34` | "the house needs writing because a score changed" |
| `resumedSavedGame` | `Boolean` | `GliderPRO/Sources/Menu.c:33` | disqualifies the run from the high scores |

`playerSuicide`, `onePlayerLeft`, `playerDead` (all
`GliderPRO/Sources/Player.c:53`) and `otherPlayerEscaped`
(`GliderPRO/Sources/Play.c` extern at `:63`, defined in
`GliderPRO/Sources/Transit.c`) belong to the two-player death dance and are
described in section 3.4.

### 1.2 `mortals` is a count of *spares*

This is the single most likely off-by-one in a port. `mortals` is set to
`kInitialGliders` = 2 for a new one-player game
(`GliderPRO/Sources/Play.c:341`), the scoreboard displays `mortals` verbatim,
and death is detected by `mortals--` followed by `if (mortals < 0)`
(`GliderPRO/Sources/Player.c:1492-1493`). So:

- a fresh game shows **2** on the scoreboard and the player can die **three**
  times (2 -> 1 -> 0 -> -1);
- the transition that ends a one-player game is `0 -> -1`;
- `RefreshNumGliders()` clamps a negative value to 0 for display
  (`GliderPRO/Sources/Scoreboard.c:209-210`) but `QuickGlidersRefresh()` does
  **not** (`GliderPRO/Sources/Scoreboard.c:284`), so the two refresh paths can
  disagree — see section 5.7.

### 1.3 Complete reference census

These tables were produced with `grep -arn '\btheScore\b' Sources/*.c` and so
on, with `extern` declarations filtered out. They are complete.

`theScore` — 9 mutation sites, 7 of which add points:

| Site | Statement |
|---|---|
| `GliderPRO/Sources/Play.c:318` | `theScore = smallGame.score;` (resume) |
| `GliderPRO/Sources/Play.c:340` | `theScore = 0L;` (new game) |
| `GliderPRO/Sources/Transit.c:442` | `theScore += kRoomVisitScore;` |
| `GliderPRO/Sources/Interactions.c:776` | `theScore += kRedClockPoints;` |
| `GliderPRO/Sources/Interactions.c:792` | `theScore += kBlueClockPoints;` |
| `GliderPRO/Sources/Interactions.c:808` | `theScore += kYellowClockPoints;` |
| `GliderPRO/Sources/Interactions.c:825` | `theScore += kCuckooClockPoints;` |
| `GliderPRO/Sources/Interactions.c:928` | `theScore += points;` (invisible bonus) |
| `GliderPRO/Sources/Interactions.c:948` | `theScore += kStarPoints;` |

Readers: `GliderPRO/Sources/Scoreboard.c:80`, `:85`, `:86`, `:89`, `:247`,
`:263`; `GliderPRO/Sources/HighScores.c:392`, `:410`, `:509`;
`GliderPRO/Sources/SavedGames.c:95` (inside the commented-out `SaveGame2()`),
`:324`.

`mortals` — 6 mutation sites:

| Site | Statement |
|---|---|
| `GliderPRO/Sources/Play.c:319` | `mortals = smallGame.numGliders;` (resume) |
| `GliderPRO/Sources/Play.c:341` | `mortals = kInitialGliders;` (new game) |
| `GliderPRO/Sources/Play.c:343` | `mortals += kInitialGliders;` (two-player) |
| `GliderPRO/Sources/Interactions.c:841` | `mortals++;` (paper pickup) |
| `GliderPRO/Sources/Interactions.c:843` | `mortals++;` (paper pickup, two-player bonus) |
| `GliderPRO/Sources/Player.c:1492` | `mortals--;` (death) |

There is **no other write**. In particular there is nothing anywhere in the tree
of the form "if score crosses N, award a life". Section 3.3 spells this out.

`batteryTotal` — 11 mutation sites; `bandsTotal` — 7; `foilTotal` — 9. All are
tabulated in section 4.

## 2. Scoring

### 2.1 The point constants

All six are in one block (`GliderPRO/Headers/GliderDefines.h:536-541`):

```c
#define kRoomVisitScore         100
#define kRedClockPoints         100
#define kBlueClockPoints        300
#define kYellowClockPoints      500
#define kCuckooClockPoints      1000
#define kStarPoints             5000
```

There is no seventh constant and no multiplier, combo, or difficulty scaling
anywhere. The complete scoring table for the whole game is:

| Event | Points | Awarded at | Object code |
|---|---:|---|---|
| Leave a room for the first time | 100 | `GliderPRO/Sources/Transit.c:442` | — |
| Red alarm clock | 100 | `GliderPRO/Sources/Interactions.c:776` | `kRedClock` = `0x21` |
| Blue alarm clock | 300 | `GliderPRO/Sources/Interactions.c:792` | `kBlueClock` = `0x22` |
| Yellow alarm clock | 500 | `GliderPRO/Sources/Interactions.c:808` | `kYellowClock` = `0x23` |
| Cuckoo clock | 1000 | `GliderPRO/Sources/Interactions.c:825` | `kCuckoo` = `0x24` |
| Invisible bonus | 100, 300 or 500 (per-object) | `GliderPRO/Sources/Interactions.c:928` | `kInvisBonus` = `0x2B` |
| Star | 5000 | `GliderPRO/Sources/Interactions.c:948` | `kStar` = `0x2C` |

Object codes are from `GliderPRO/Headers/GliderDefines.h:344-358`.

Everything else in the prize family — paper `0x25`, battery `0x26`, bands
`0x27`, grease right `0x28`, grease left `0x29`, foil `0x2A`, sparkle `0x2D`,
helium `0x2E`, slider `0x2F` — is worth **zero points**.

### 2.2 Independent corroboration: `CountTotalHousePoints()`

The level editor has a "House Info" dialog that shows the total points available
in a house. Its implementation is an independent enumeration of the scoring
table by the same author, and it agrees exactly
(`GliderPRO/Sources/HouseInfo.c:51-105`):

```c
pointTotal = (long)RealRoomNumberCount() * (long)kRoomVisitScore;
...
for (i = 0; i < numRooms; i++)
  if ((*thisHouse)->rooms[i].suite != kRoomIsEmpty)
    for (h = 0; h < kMaxRoomObs; h++)
      switch ((*thisHouse)->rooms[i].objects[h].what)
      {
        case kRedClock:    pointTotal += kRedClockPoints;    break;
        case kBlueClock:   pointTotal += kBlueClockPoints;   break;
        case kYellowClock: pointTotal += kYellowClockPoints; break;
        case kCuckoo:      pointTotal += kCuckooClockPoints; break;
        case kStar:        pointTotal += kStarPoints;        break;
        case kInvisBonus:  pointTotal += (*thisHouse)->rooms[i].objects[h].data.c.points; break;
        default: break;
      }
```

Six cases, matching the six rows of the table above one for one. `kMaxRoomObs`
is 24 (`GliderPRO/Headers/GliderDefines.h:250`); note the loop scans all 24
object slots regardless of the room's `numObjects` field.

`RealRoomNumberCount()` (`GliderPRO/Sources/House.c:170-189`) starts from
`nRooms` and subtracts one for every room whose `suite == kRoomIsEmpty` (-1,
`GliderPRO/Headers/GliderDefines.h:525`) — those are tombstones left behind by
room deletion in the editor.

The result is displayed by `UpdateHouseInfoDialog()`
(`GliderPRO/Sources/HouseInfo.c:118`) into DITL item 18 (`kHouseSizeItem`,
`GliderPRO/Sources/HouseInfo.c:22`).

I reimplemented `CountTotalHousePoints()` in Python and ran it over all 22
shipped houses. The observed totals, together with the object census that
produces them, are in section 2.9.

### 2.3 Room-visit scoring, and the departure rule

`HandleRoomVisitation()` (`GliderPRO/Sources/Transit.c:430-445`) in full:

```c
void HandleRoomVisitation (void)
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

Two copies of the flag are set: the authoritative one inside the locked house
handle (`rooms[localNumbers[kCentralRoom]].visited`, where `localNumbers[0]` is
the current room's real index, `GliderPRO/Sources/RoomGraphics.c:29` and the
fill loop at `:56`) and the working copy `thisRoom->visited` that the renderer
uses.

**It is called at the top of each transition function, before the room actually
changes.** The four call sites, all in `GliderPRO/Sources/Transit.c`, are the
complete set:

| Call site | Function | Position in the function |
|---|---|---|
| `GliderPRO/Sources/Transit.c:155` | `MoveRoomToRoom()` | first statement |
| `GliderPRO/Sources/Transit.c:319` | `TransportRoomToRoom()` | after `SetMusicalMode(kKickGameScoreMode)` |
| `GliderPRO/Sources/Transit.c:357` | `MoveDuctToDuct()` | after `SetMusicalMode(kKickGameScoreMode)` |
| `GliderPRO/Sources/Transit.c:396` | `MoveMailToMail()` | after `SetMusicalMode(kKickGameScoreMode)` |

Consequences a port must reproduce:

1. The room you are standing in when the game starts is **not** scored on
   arrival. You get its 100 points the first time you leave it. (If you then
   re-enter it, nothing more happens, because the flag is now set.)
2. A room you enter and die in without ever leaving is never scored.
3. `visited` therefore means "has been departed from", which is why
   `CountRoomsVisited()` — the value stored in the high-score table's
   `levels[]` column — can be one less than the number of rooms the player has
   actually seen.
4. Because both the flag and the score are guarded by the same `if`, the 100
   points are awarded exactly once per room per game.

### 2.4 The `visited` flag's whole lifecycle

There are exactly 8 references to `visited` in the tree (excluding the
`savedRoom` declaration):

| Site | What it does |
|---|---|
| `GliderPRO/Headers/GliderStructs.h:140` | `savedRoom.visited` field declaration |
| `GliderPRO/Headers/GliderStructs.h:173` | `roomType.visited` field declaration |
| `GliderPRO/Sources/Play.c:619` | `thisHousePtr->rooms[r].visited = false;` — the **only** reset, inside `SetObjectsToDefaults()` |
| `GliderPRO/Sources/Room.c:174` | `thisRoom->visited = false;` — initialising a brand-new room in the editor |
| `GliderPRO/Sources/Transit.c:435` | the `if (!thisRoom->visited)` test |
| `GliderPRO/Sources/Transit.c:440` | set the house-handle copy |
| `GliderPRO/Sources/Transit.c:443` | set the working copy |
| `GliderPRO/Sources/House.c:525` | `CountRoomsVisited()`'s test |
| `GliderPRO/Sources/SavedGames.c:115`, `:281` | copies in the fully commented-out `SaveGame2()` |

`SetObjectsToDefaults()` is called from `NewGame()` only when the mode is not
`kResumeGameMode` (`GliderPRO/Sources/Play.c:102-103`):

```c
if (mode != kResumeGameMode)
    SetObjectsToDefaults();
```

So a resumed game keeps the `visited` flags that were saved in the house file
— which is how a resumed run continues to score correctly for rooms it has not
yet been to. It also means the flags persist in the house file across
application launches, and the *editor* writing a house preserves whatever they
were. Because a resumed game is ineligible for the high scores (section 7.6),
this leakage never affects the score table's `levels[]` column for a legitimate
entry.

`CountRoomsVisited()` (`GliderPRO/Sources/House.c:511-531`):

```c
count = 0;
for (r = 0; r < numRooms; r++)
    if (thisHousePtr->rooms[r].visited)
        count++;
return (count);
```

Note it iterates `nRooms`, not `RealRoomNumberCount()`, so a deleted-room
tombstone with a stale `visited` byte would be counted. In practice
`SetObjectsToDefaults()` clears all `nRooms` entries at the start of every new
game, so this cannot happen within a game.

### 2.5 `HandleRewards()`: the only in-room scoring

`HandleRewards(gliderPtr thisGlider, hotPtr who)`
(`GliderPRO/Sources/Interactions.c:756-981`) is reached from
`HandleHotSpotCollision()` case `kRewardIt` (action code 6,
`GliderPRO/Headers/GliderDefines.h:288`) at
`GliderPRO/Sources/Interactions.c:1246-1248`.

The function's shape, in numbered pseudocode with the original identifiers in
parentheses:

```
 1  whoLinked  <- who->who                  // index into masterObjects[]
 2  bounds     <- who->bounds               // the hot spot's rect, ROOM-LOCAL
 3  switch (masterObjects[whoLinked].theObject.what)
 4    each arm:
 5      (kInvisBonus only) points <- masterObjects[whoLinked].theObject.data.c.points
 6      if SetObjectState(thisRoomNumber, masterObjects[whoLinked].objectNum, 0, whoLinked)
 7          ...the arm body...
 8      who->isOn <- false
```

Line 6 is the crucial guard: `SetObjectState(..., 0, ...)` attempts to set the
object's state to 0 (off/taken) and returns false if it was already 0. That
makes every reward strictly one-shot per game (per house-file object), and it is
what prevents a glider that lingers on a hot spot from farming points every
frame. Line 8 runs unconditionally, so the hot spot is switched off even if the
state change failed.

Note that for `kInvisBonus` the `points` field is read on line 5 **before** the
guard (`GliderPRO/Sources/Interactions.c:920`). Harmless, but it means the read
happens even for an already-taken bonus.

`masterObjects[]` is the flattened per-game object array
(`objDataType`, `GliderPRO/Headers/GliderStructs.h:322-332`), sized
`kMaxMasterObjects` = 216 = `kMaxRoomObs * 9`
(`GliderPRO/Headers/GliderDefines.h:266`). Its `.objectNum` is the object's real
index within its room, and `.dynaNum` / `.hotNum` are indices into the dynamics
and hot-spot arrays.

### 2.6 Every arm of `HandleRewards()`, in source order

The table below is a complete transcription. "vel/4" means
`thisGlider->hVel /= 4; thisGlider->vVel /= 4;` and "vel/2" likewise with 2.
"Restore" means `RestoreFromSavedMap(thisRoomNumber, objectNum, false)` — repaint
the background where the object used to be. "Sparkle" means
`AddSparkle(&bounds)`. "Grease" means `RedrawAllGrease()`.

| Object | Lines | Sound | Restore | Sparkle | Flying pts | Velocity | Score | Other effects | Grease |
|---|---|---|:-:|:-:|---|---|---:|---|:-:|
| `kRedClock` `0x21` | 766-780 | `kBeepsSound`/`kBeepsPriority` | yes | no | 100, `hVel/2`, `vVel/2` | vel/4 | +100 | — | yes |
| `kBlueClock` `0x22` | 782-796 | `kBuzzerSound`/`kBuzzerPriority` | yes | no | 300, `hVel/2`, `vVel/2` | vel/4 | +300 | — | yes |
| `kYellowClock` `0x23` | 798-812 | `kDingSound`/`kDingPriority` | yes | no | 500, `hVel/2`, `vVel/2` | vel/4 | +500 | — | yes |
| `kCuckoo` `0x24` | 814-829 | `kCuckooSound`/`kCuckooPriority` | yes | no | 1000, `hVel/2`, `vVel/2` | vel/4 | +1000 | `StopPendulum(thisRoomNumber, objectNum)` | yes |
| `kPaper` `0x25` | 831-848 | `kEnergizeSound`/`kEnergizePriority` | yes | yes | none | vel/2 | 0 | `mortals++`; again if `twoPlayerGame && !onePlayerLeft`; `QuickGlidersRefresh()` | yes |
| `kBattery` `0x26` | 850-870 | `kEnergizeSound`/`kEnergizePriority` | yes | yes | none | vel/2 | 0 | see 4.2; `QuickBatteryRefresh(false)` | yes |
| `kBands` `0x27` | 872-889 | `kEnergizeSound`/`kEnergizePriority` | yes | yes | none | vel/2 | 0 | `bandsTotal += kBandsSupply` (x2 in 2P); `QuickBandsRefresh(false)` | yes |
| `kGreaseRt` `0x28` | 891-898 | none | no | no | none | unchanged | 0 | `SpillGrease(dynaNum, hotNum)` | no |
| `kGreaseLf` `0x29` | 891-898 | none | no | no | none | unchanged | 0 | `SpillGrease(dynaNum, hotNum)` | no |
| `kFoil` `0x2A` | 900-917 | `kEnergizeSound`/`kEnergizePriority` | yes | yes | none | vel/2 | 0 | `foilTotal += kFoilSupply` (x2 in 2P); `StartGliderFoilGoing(thisGlider)` | yes |
| `kInvisBonus` `0x2B` | 919-931 | `kBonusSound`/`kBonusPriority` | **no** | **no** | `points`, `hVel/2`, `vVel/2` | vel/4 | +`points` | — | **no** |
| `kStar` `0x2C` | 933-951 | `kEnergizeSound`/`kEnergizePriority` | yes | yes | **none** | **unchanged** | +5000 | `StopStar()`; `numStarsRemaining--`; win check | yes |
| `kSparkle` `0x2D` | 953-954 | — | — | — | — | — | 0 | empty case, falls straight to `break` | — |
| `kHelium` `0x2E` | 956-976 | `kEnergizeSound`/`kEnergizePriority` | yes | yes | none | vel/2 | 0 | see 4.2; `QuickBatteryRefresh(false)` | yes |
| `kSlider` `0x2F` | 978-979 | — | — | — | — | — | 0 | empty case | — |

Four asymmetries worth calling out, because they look like oversights and a
faithful port must keep them:

1. **`kInvisBonus` does not restore the background, does not sparkle and does
   not redraw grease.** It is invisible, so there is nothing to erase — but the
   missing `RedrawAllGrease()` is inconsistent with every other scoring arm.
2. **`kStar` does not slow the glider and does not spawn a flying-point
   sprite**, despite awarding the largest single score in the game. The player
   gets the `DisplayStarsRemaining()` overlay instead (section 5.9).
3. **The two grease arms play no sound and do not switch the object off
   visually** — `SpillGrease()` owns all of that.
4. `kPaper`, `kBattery`, `kBands`, `kFoil`, `kHelium` and `kStar` all use the
   same `kEnergizeSound`; only the four clocks and the invisible bonus have
   distinct sounds.

Sound constants and their `snd ` resource IDs (the ID is the constant + 1000,
`GliderPRO/Sources/Sound.c:327` and `:501`):

| Constant | Value | `snd ` ID | Resource name | Bytes | Priority constant | Value |
|---|---:|---:|---|---:|---|---:|
| `kBeepsSound` | 3 | 1003 | 'Beeps' | 9722 | `kBeepsPriority` | 800 |
| `kBuzzerSound` | 4 | 1004 | 'Buzzer' | 8074 | `kBuzzerPriority` | 801 |
| `kDingSound` | 5 | 1005 | 'Ding' | 9770 | `kDingPriority` | 802 |
| `kEnergizeSound` | 6 | 1006 | 'Energize' | 11946 | `kEnergizePriority` | 803 |
| `kCuckooSound` | 11 | 1011 | 'Cuckoo' | 5178 | `kCuckooPriority` | 805 |
| `kScoreTikSound` | 17 | 1017 | 'Score Tick' | 686 | `kScoreTikPriority` | 101 |
| `kBonusSound` | 61 | 1061 | 'Bonus' | 5430 | `kBonusPriority` | 812 |

(`GliderPRO/Headers/GliderDefines.h:58-116` for the sound numbers,
`:120-180` for the priorities; resource names and byte sizes read from
`GliderPRO/Glider PRO.r`.)

### 2.7 The invisible bonus and its `points` field

`kInvisBonus` is the only object whose point value is stored per instance. It
lives in the `bonusType` variant of the object union
(`GliderPRO/Headers/GliderStructs.h:27-34`):

```c
typedef struct
{
    Point       topLeft;                // 4
    short       length;                 // 2 grease spill
    short       points;                 // 2 invis bonus
    Boolean     state;                  // 1
    Boolean     initial;                // 1
} bonusType;                            // total = 10
```

and `objectType` is (`GliderPRO/Headers/GliderStructs.h:90-105`) a `short what`
followed by the 10-byte union, total 12 bytes. So within one 12-byte object
record:

| Offset | Size | Field |
|---:|---:|---|
| 0 | 2 | `what` |
| 2 | 2 | `data.c.topLeft.v` |
| 4 | 2 | `data.c.topLeft.h` |
| 6 | 2 | `data.c.length` |
| **8** | **2** | **`data.c.points`** |
| 10 | 1 | `data.c.state` |
| 11 | 1 | `data.c.initial` |

I verified this offset by walking every object slot of every room of all 22
shipped houses and reading the big-endian `short` at object offset 8 for every
`what == 0x2B`. The result is a clean histogram over exactly three values —
which is what the editor constrains it to.

The editor's constraint: `DoInvisBonusObjectInfo()`
(`GliderPRO/Sources/ObjectInfo.c:1947-2050`, dialog `kInvisBonusInfoDialogID`
1015) maps an existing value of 300 to radio button 1, 500 to radio button 2,
and anything else to radio button 0, then writes back exactly 100, 300 or 500.
`ObjectAdd.c:278` sets `data.c.points = 100` when a new invisible bonus is
placed; `ObjectAdd.c:244`, `:264` and `:292` set `points = 0` for the other
`bonusType` users (grease and sparkle), which never read it.

DITL 1015 "Invis Bonus Info", decoded from `GliderPRO/Glider PRO.r` (9 items):

| # | Type | Rect (t,l,b,r) | Size | Text / resource |
|---:|---|---|---|---|
| 1 | Button | 127,190,147,248 | 58x20 | 'Okay' |
| 2 | StaticText | — | — | 'Object Number: ^0' |
| 3 | StaticText | — | — | 'Object Kind: ^1' |
| 4 | userItem | 117,... | 240x1 | divider line |
| 5 | Picture | — | 257x32 | `PICT` 1002 |
| 6 | RadioButton | 57,141,75,233 | 92x18 | '100 Points' |
| 7 | RadioButton | 75,141,93,233 | 92x18 | '300 Points' |
| 8 | RadioButton | 93,141,111,233 | 92x18 | '500 Points' |
| 9 | Button | 127,8,147,104 | 96x20 | 'Linked From?' |

So a Go port can treat the invisible bonus as a three-valued enum and still
round-trip every shipped house — but it should read and write the raw `short`,
because nothing in the file format enforces the three values.

### 2.8 Flying points: the score-popup animation

When a scoring object other than a star is taken, a small numeral sprite flies
away from the pickup point. `AddFlyingPoint()`
(`GliderPRO/Sources/DynamicMaps.c:199-254`) in full:

```c
void AddFlyingPoint (Rect *theRect, short points, short hVel, short vVel)
{
    Rect        centeredRect;
    short       i;

    if (numFlyingPts < kMaxFlyingPts)
    {
        theRect->left   += playOriginH;
        theRect->right  += playOriginH;
        theRect->top    += playOriginV;
        theRect->bottom += playOriginV;

        centeredRect = pointsSrc[0];
        CenterRectInRect(&centeredRect, theRect);

        for (i = 0; i < kMaxFlyingPts; i++)
            if (flyingPoints[i].mode == -1)
            {
                flyingPoints[i].dest  = centeredRect;
                flyingPoints[i].whole = centeredRect;
                flyingPoints[i].loops = 0;
                flyingPoints[i].hVel  = hVel;
                flyingPoints[i].vVel  = vVel;
                switch (points)
                {
                    case 100: flyingPoints[i].start = 12; flyingPoints[i].stop = 14; break;
                    case 250: flyingPoints[i].start =  9; flyingPoints[i].stop = 11; break;
                    case 300: flyingPoints[i].start =  6; flyingPoints[i].stop =  8; break;
                    case 500: flyingPoints[i].start =  3; flyingPoints[i].stop =  5; break;
                    default:  flyingPoints[i].start =  0; flyingPoints[i].stop =  2; break;
                }
                flyingPoints[i].mode = flyingPoints[i].start;
                numFlyingPts++;
                break;
            }
    }
}
```

Facts:

- `kMaxFlyingPts` is 3 (`GliderPRO/Headers/GliderDefines.h:253`), so at most
  three numerals can be in flight; a fourth simultaneous pickup silently gets no
  sprite (the score is still awarded — the `AddFlyingPoint` call is separate
  from the `theScore +=`).
- **`theRect` is mutated in place.** The caller passes `&bounds`, a local copy
  of the hot spot's rect (`GliderPRO/Sources/Interactions.c:762`), so this is
  safe, but a port that passes a pointer into shared state would corrupt it.
  The offsets convert room-local coordinates into main-window coordinates:
  `playOriginH`, `playOriginV` (`GliderPRO/Sources/MainWindow.c:40`) are the
  top-left of the central room on screen (section 5.3).
- `pointsSrc[0]` is 24x8 (section 2.8.2), so `CenterRectInRect` centres a 24x8
  box on the hot spot.
- The `case 250` range (frames 9-11) is **unreachable**: nothing in the tree
  passes 250. `HandleRewards()` passes literal 100, 300, 500, 1000 and
  `data.c.points` (constrained to 100/300/500 by the editor).
- 1000 lands in the `default` arm (frames 0-2). So does any other unexpected
  value, including 0 and 5000 — but neither is ever passed.

`RenderFlyingPoints()` (`GliderPRO/Sources/Render.c:325-380`) advances them:

```
 1  if numFlyingPts == 0: return
 2  for i in 0..kMaxFlyingPts-1:
 3    if flyingPoints[i].mode == -1: continue
 4    if flyingPoints[i].mode > flyingPoints[i].stop:
 5        flyingPoints[i].mode  <- flyingPoints[i].start
 6        flyingPoints[i].loops <- flyingPoints[i].loops + 1
 7    if flyingPoints[i].loops >= kMaxFlyingPointsLoop:      // 24
 8        AddRectToWorkRects(&flyingPoints[i].dest)          // erase
 9        flyingPoints[i].mode <- -1
10        numFlyingPts--
11    else:
12        dest.left  += hVel ; dest.right  += hVel
13        whole.right <- dest.right   if hVel > 0  else  whole.left <- dest.left
14        dest.top   += vVel ; dest.bottom += vVel
15        whole.bottom <- dest.bottom if vVel > 0  else  whole.top  <- dest.top
16        CopyMask(pointsSrcMap, pointsMaskMap -> workSrcMap,
17                 pointsSrc[mode], pointsSrc[mode], dest)
18        AddRectToWorkRects(&whole)     // union of old+new, to erase next frame
19        AddRectToBackRects(&dest)
20        whole <- dest
21        mode++
```

Lifetime arithmetic: the frame range is 3 wide, so each `loops` increment costs
3 drawn frames. `kMaxFlyingPointsLoop` is 24
(`GliderPRO/Headers/GliderDefines.h:254`), so a sprite lives 3 x 24 = **72
frames = 2.4 seconds** at 30 fps, travelling `72 * hVel` horizontally and
`72 * vVel` vertically, where `hVel`/`vVel` are half the glider's velocity at
pickup time. Because the glider's velocity is then quartered, the numeral
appears to be flung out of the glider.

`flyingPoints` is a heap array allocated once at startup
(`GliderPRO/Sources/StructuresInit2.c:207-210`, `NewPtr(sizeof(flyingPtType) *
kMaxFlyingPts)`) and every slot is reset to `mode = -1` by
`InitGarbageRects()` (`GliderPRO/Sources/Render.c:686-688`), which is called
from `NewGame()` (`GliderPRO/Sources/Play.c:184`) and from
`GliderPRO/Sources/RoomGraphics.c:417`.

`flyingPtType` (`GliderPRO/Headers/GliderStructs.h:241-249`):

```c
typedef struct
{
    Rect        dest, whole;    // 8 + 8
    short       start;          // 2
    short       stop;           // 2
    short       mode;           // 2   (-1 == free slot)
    short       loops;          // 2
    short       hVel, vVel;     // 2 + 2
} flyingPtType, *flyingPtPtr;   // total = 28
```

#### 2.8.1 The `pointsSrc[15]` frame rects

`GliderPRO/Sources/StructuresInit.c:413-416`:

```c
for (i = 0; i < 15; i++)
{
    QSetRect(&pointsSrc[i], 0, 0, 24, 8);
    QOffsetRect(&pointsSrc[i], 0, i * 8);
}
```

so frame *i* is the 24x8 strip at `(top=8i, left=0, bottom=8i+8, right=24)` of a
24x120 sheet. `pointsSrc` is declared `Rect pointsSrc[15]` at
`GliderPRO/Sources/DynamicMaps.c:30` and `extern`ed into `Render.c:52` and
`StructuresInit.c:50`.

The two GWorlds (`GliderPRO/Sources/StructuresInit.c:400-410`, globals at
`GliderPRO/Sources/Objects.c:32-34`):

```c
QSetRect(&pointsSrcRect, 0, 0, 24, 120);        // 2880 pixels
CreateOffScreenGWorld(&pointsSrcMap, &pointsSrcRect, kPreferredDepth);   // 8-bit
SetGWorld(pointsSrcMap, nil);
LoadGraphic(kPointsPictID);                     // PICT 4006
CreateOffScreenGWorld(&pointsMaskMap, &pointsSrcRect, 1);                // 1-bit
SetGWorld(pointsMaskMap, nil);
LoadGraphic(kPointsPictID + 1000);              // PICT 5006
```

`kPointsPictID` is 4006 (`GliderPRO/Sources/StructuresInit.c:26`). The mask is
therefore `PICT 5006`. The "+1000 is the mask" convention is used throughout the
game.

#### 2.8.2 `PICT 4006` decoded

I decoded `PICT 4006` from `GliderPRO/Glider PRO.r` completely. Observed:

| Property | Value |
|---|---|
| Resource size | 4682 bytes, matching the `picSize` header field |
| `picFrame` | (0, 0, 120, 24) — i.e. 24 wide, 120 tall |
| Version | QuickDraw v2 (`0x0011 0x02FF`, header opcode `0x0C00`) |
| Bits opcode | `0x0098` PackBitsRect |
| `pixelSize` | 8 |
| Colour table | 256 entries, standard Mac system CLUT |
| Distinct indices actually used | 4 |

The four indices, with their RGB from the embedded CLUT:

| Index | RGB | Role |
|---:|---|---|
| 0 | `#FFFFFF` | the masked-out background |
| 35 | `#FF0000` | red numerals |
| 210 | `#0000FF` | blue numerals |
| 225 | `#00EE00` | green numerals |

Per-frame colour, read out of the decoded bitmap:

| Frames | Colour index |
|---|---:|
| 0, 3, 6, 9, 12 | 225 green |
| 1, 4, 7, 10, 13 | 35 red |
| 2, 5, 8, 11, 14 | 210 blue |

And the glyphs, read as pixels:

| Frames | Glyph | Reached by |
|---|---|---|
| 0-2 | `1000` | `default` arm — the cuckoo clock |
| 3-5 | `500` | `case 500` — yellow clock, 500-point invisible bonus |
| 6-8 | `300` | `case 300` — blue clock, 300-point invisible bonus |
| 9-11 | `250` | `case 250` — **unreachable** |
| 12-14 | `100` | `case 100` — red clock, 100-point invisible bonus |

This is direct proof, not inference: the sprite sheet contains a "250" that no
code path can display, and the three frames per value are a
green -> red -> blue **colour cycle**, not a fade or a motion sequence. So the
numeral visibly strobes through three colours at 30 fps for 2.4 s while drifting.

#### 2.8.3 `PICT 5006` — the mask, and a format trap

`PICT 5006` is **a QuickDraw version-1 PICT**, unlike every other picture in
this subsystem. Its version bytes at offset 10 are `0x11 0x01`, not
`0x11 0x02FF`, and its opcodes are therefore **single bytes**. A decoder written
for v2 reads a 2-byte opcode and immediately desynchronises. Hand-decoded:

| Offset | Bytes | Meaning |
|---:|---|---|
| 0 | `02 1B` | `picSize` = 539 (file is 539 bytes) |
| 2 | `00 00 00 5B 00 78 00 73` | `picFrame` = (0, 91, 120, 115) |
| 10 | `11 01` | version 1 |
| 12 | `A0 00 82` | ShortComment, kind `0x0082` |
| 15 | `01 00 0A 00 00 00 00 02 D0 02 40` | Clip, length 10, rect (0,0,720,576) |
| 26 | `90` | BitsRect (1-bit, unpacked) |
| 27 | `00 04` | `rowBytes` = 4 |
| 29 | `00 00 00 58 00 78 00 78` | `bitmapBounds` = (0, 88, 120, 120) — 32 px wide |
| 37 | ... | `srcRect` = (0, 91, 120, 115) |
| 45 | ... | `dstRect` = (0, 91, 120, 115) |
| 53 | `00 00` | `mode` = srcCopy |
| 55 | 480 bytes | 120 rows x 4 bytes, uncompressed |
| 535 | — | end of meaningful data (539 with padding) |

Two oddities:

1. The `picFrame` is **not** at the origin: `left = 91`, `right = 115`, so the
   frame is 24 wide but offset 91 px to the right. `LoadGraphic()`
   (`GliderPRO/Sources/Utilities.c:317-333`) normalises this:

```c
thePicture = GetPicture(resID);
if (thePicture == nil) RedAlert(kErrFailedGraphicLoad);
HLock((Handle)thePicture);
bounds = (*thePicture)->picFrame;
HUnlock((Handle)thePicture);
OffsetRect(&bounds, -bounds.left, -bounds.top);
DrawPicture(thePicture, &bounds);
ReleaseResource((Handle)thePicture);
```

   The `OffsetRect(&bounds, -bounds.left, -bounds.top)` moves the destination
   back to (0,0), and `DrawPicture` scales/translates the picture into it, so the
   24 useful columns land at x 0..23 in the 1-bit GWorld. A Go port that loads
   the mask by taking `picFrame` at face value will place it 91 px to the right
   and the numerals will be entirely masked out.

2. `rowBytes` is 4 = 32 px, wider than the 24 useful columns, because the
   `bitmapBounds` is (0,88,120,120). The extra 8 columns are outside `srcRect`
   and are discarded by `DrawPicture`.

I confirmed the mask contents by rendering rows 0..7: frame 0 spells `1000`
(e.g. row 2 is `.##..####...####...####.`), matching frame 0 of `PICT 4006`.

`CopyMask()` semantics: `CopyMask(src, mask, dst, srcRect, maskRect, dstRect)`
copies `src` to `dst` only where `mask` has a 1 bit. Since the mask is 1-bit and
`PICT 5006` has the numerals set, the white (index 0) background of `PICT 4006`
never reaches the screen.

### 2.9 The maximum achievable score of every shipped house

I reimplemented `CountTotalHousePoints()` in Python against the BinHex-decoded
data forks, walking `nRooms` rooms x 24 object slots at the verified offsets
(room stride 348, objects at room offset 60, 12 bytes each, `what` at object
offset 0, `points` at object offset 8, `suite` at room offset 54). The observed
object census and total for all 22 shipped houses:

| House | `nRooms` | real rooms | stars | red | blue | yel | cuckoo | invis (100/300/500) | total points |
|---|---:|---:|---:|---:|---:|---:|---:|---|---:|
| Art Museum | 109 | 109 | 6 | 0 | 5 | 11 | 2 | 0/0/18 | 58 900 |
| CD Demo House | 206 | 206 | 9 | 3 | 4 | 9 | 2 | 0/0/8 | 77 600 |
| California or Bust! | 16 | 16 | 1 | 1 | 1 | 1 | 0 | 4/0/0 | 7 900 |
| Castle o' the Air | 85 | 85 | 4 | 0 | 0 | 0 | 5 | 0/0/24 | 45 500 |
| Davis Station | 65 | 65 | 4 | 0 | 1 | 6 | 2 | 0/0/0 | 31 800 |
| Demo House | 45 | 45 | 1 | 2 | 0 | 2 | 1 | 0/1/1 | 12 500 |
| Empty House | 35 | 35 | 1 | 0 | 0 | 0 | 0 | 0/0/0 | 8 500 |
| Fun House | 43 | 43 | **0** | 0 | 2 | 4 | 1 | 0/0/0 | 7 900 |
| Grand Prix | 175 | 175 | 3 | 5 | 6 | 6 | 2 | 0/1/9 | 44 600 |
| ImagineHouse PRO II | 279 | 279 | 3 | 8 | 16 | 10 | 9 | 2/6/8 | 68 500 |
| In The Mirror | 97 | 97 | 1 | 7 | 14 | 5 | 5 | 1/5/4 | 30 700 |
| Land of Illusion | 303 | 303 | 5 | 8 | 8 | 20 | 6 | 12/0/4 | 77 700 |
| Leviathan | 472 | 472 | 6 | 12 | 20 | 30 | 8 | 26/7/12 | 118 100 |
| Metropolis | 127 | 127 | 4 | 5 | 2 | 5 | 1 | 0/0/4 | 39 300 |
| Nemo's Market | 124 | 124 | 5 | 14 | 11 | 14 | 1 | 11/0/2 | 52 200 |
| Rainbow's End | 223 | 223 | 5 | 15 | 24 | 26 | 6 | 25/0/0 | 77 500 |
| Sampler | 2 | 2 | 1 | 0 | 0 | 0 | 0 | 0/0/0 | 5 200 |
| Slumberland | 383 | 383 | 6 | 29 | 19 | 73 | 13 | 20/5/23 | 141 400 |
| SpacePods | 402 | 402 | 1 | 18 | 14 | 11 | 13 | 28/0/1 | 73 000 |
| Teddy World | 531 | 531 | 1 | 6 | 8 | 25 | 3 | 4/0/7 | 80 500 |
| The Asylum Pro | 140 | 140 | 1 | 5 | 3 | 21 | 1 | 0/2/11 | 38 000 |
| Titanic | 208 | 208 | 1 | 2 | 5 | 51 | 19 | 3/0/28 | 86 300 |

Observations a port can use as regression fixtures:

- No shipped house has any tombstone room: `real rooms == nRooms` everywhere.
- Every observed `kInvisBonus.points` value is exactly 100, 300 or 500 — 327
  invisible bonuses across all houses, zero exceptions. This is the empirical
  confirmation of the editor constraint from section 2.7.
- **Fun House has zero stars.** Since the only `FlagGameOver()`-by-winning path
  is the `numStarsRemaining <= 0` test *inside* the star-pickup arm
  (`GliderPRO/Sources/Interactions.c:942-944`), a house with no stars **can
  never be won**. `numStarsRemaining` starts at 0 and nothing decrements it.
  The editor warns about this: `STR# 150` string 35 is
  `'You have no stars in the house!'`.
- The observed high scores in the shipped houses are far below these maxima
  (the highest anywhere is 47 000 in ImagineHouse PRO II, whose maximum is
  68 500), which is consistent with `theScore` being a plain running total with
  no bonuses.

### 2.10 The score display roll

The scoreboard does not snap to the new score; it counts up.
`HandleDynamicScoreboard()` (`GliderPRO/Sources/Scoreboard.c:80-91`):

```c
if (theScore > displayedScore)
{
    if (doRollScore)
    {
        displayedScore += kScoreRollAmount;
        if (displayedScore > theScore)
            displayedScore = theScore;
    }
    else
        displayedScore = theScore;
    PlayPrioritySound(kScoreTikSound, kScoreTikPriority);
    QuickScoreRefresh();
}
```

`kScoreRollAmount` is 13 (`GliderPRO/Sources/Scoreboard.c:21`). So:

- The display climbs **13 points per frame** = 390 points per second.
- A 5000-point star takes ceil(5000/13) = 385 frames = **12.8 seconds** to
  finish rolling up. A 100-point room visit takes 8 frames.
- `kScoreTikSound` (`snd ` 1017, 'Score Tick', 686 bytes) plays **once per
  frame** for the entire roll, at priority 101 — the second-lowest priority in
  the game, above only `kHitWallPriority` 100 — so it is a continuous ticking
  that any other sound interrupts. This is the game's characteristic
  score-counting chatter.
- The roll only ever goes **up**: there is no `theScore < displayedScore`
  branch, so a score decrease (impossible in normal play, but possible after
  resuming a lower-scoring saved game) would leave the display stuck high until
  the next full `RefreshPoints()`.

`doRollScore` is a curiosity. It is set to `true` at the top of
`RefreshScoreboard()` (`GliderPRO/Sources/Scoreboard.c:55`) and **is never set
to `false` anywhere in the tree**. Its only other reference is the test at
`:82`. C static initialisation makes it `false` before the first
`RefreshScoreboard()`, which `NewGame()` performs at
`GliderPRO/Sources/Play.c:164` before any frame runs. So the `else
displayedScore = theScore;` branch is **dead code** in the shipped build and the
score always rolls. A port can hard-code the roll; keeping the flag costs
nothing and documents intent.

`RefreshPoints()` ends with `displayedScore = theScore;`
(`GliderPRO/Sources/Scoreboard.c:263`), so any full scoreboard refresh — which
happens on every room transition, via `RefreshScoreboard()` — snaps the display
to the true score and cancels an in-progress roll. Practically this means a room
change truncates the ticking.

## 3. Lives (`mortals`)

### 3.1 Starting lives

`kInitialGliders` is 2 (`GliderPRO/Sources/Play.c:19`). `InitGlider()`'s
new-game branch (`GliderPRO/Sources/Play.c:338-352`):

```c
else if (mode == kNewGameMode)
{
    theScore = 0L;
    mortals = kInitialGliders;
    if (twoPlayerGame)
        mortals += kInitialGliders;
    batteryTotal = 0;
    bandsTotal = 0;
    foilTotal = 0;
    thisGlider->mode = kGliderNormal;
    thisGlider->facing = kFaceRight;
    thisGlider->src = gliderSrc[0];
    thisGlider->mask = gliderSrc[0];
    showFoil = false;
}
```

| Mode | `mortals` after init | Deaths available |
|---|---:|---:|
| one player | 2 | 3 |
| two players | 4 | 5 (3 for the first player to run out, then the second) |

Note that `InitGlider()` is called **twice** in a two-player game
(`GliderPRO/Sources/Play.c:122-123`: `InitGlider(&theGlider, kNewGameMode);
InitGlider(&theGlider2, kNewGameMode);`) and `mortals` is a single shared
global, so the two-player total is set to 4 by the *first* call and then set to 4
again by the second. The `if (twoPlayerGame) mortals += kInitialGliders;` is not
cumulative across the two calls because line 341 (`mortals = kInitialGliders;`)
resets it first. A port must keep the assignment-then-add order.

`kFaceRight` is `TRUE` and `kFaceLeft` is `FALSE`
(`GliderPRO/Headers/GliderDefines.h:556-557`); `kPlayer1` is `TRUE`, `kPlayer2`
is `FALSE` (`:558-559`). `gliderType.which` holds one of these.

### 3.2 Resuming

`InitGlider()`'s resume branch (`GliderPRO/Sources/Play.c:316-337`) restores the
whole bookkeeping state from the `gameType` copy `smallGame`:

```c
if (mode == kResumeGameMode)
{
    theScore     = smallGame.score;
    mortals      = smallGame.numGliders;
    batteryTotal = smallGame.energy;
    bandsTotal   = smallGame.bands;
    foilTotal    = smallGame.foil;
    thisGlider->mode   = smallGame.gliderState;
    thisGlider->facing = smallGame.facing;
    showFoil           = smallGame.showFoil;
    switch (thisGlider->mode)
    {
        case kGliderBurning:  FlagGliderBurning(thisGlider); break;
        default:              FlagGliderNormal(thisGlider);  break;
    }
}
```

and immediately before it (`GliderPRO/Sources/Play.c:311-314`):

```c
if (mode == kResumeGameMode)
    numStarsRemaining = smallGame.wasStarsLeft;
else if (mode == kNewGameMode)
    numStarsRemaining = CountStarsInHouse();
```

`SaveGame()` refuses to save a two-player game
(`GliderPRO/Sources/SavedGames.c:305`: `if (twoPlayerGame) return;`), so
`mortals` from a saved game is always a one-player value.

### 3.3 The only extra life: `kPaper`

There is exactly one way to gain a life. `HandleRewards()` case `kPaper`
(`GliderPRO/Sources/Interactions.c:831-848`), which awards **no points**:

```c
case kPaper:
if (SetObjectState(thisRoomNumber, masterObjects[whoLinked].objectNum, 0, whoLinked))
{
    PlayPrioritySound(kEnergizeSound, kEnergizePriority);
    RestoreFromSavedMap(thisRoomNumber, masterObjects[whoLinked].objectNum, false);
    AddSparkle(&bounds);
    thisGlider->hVel /= 2;
    thisGlider->vVel /= 2;
    mortals++;
    if ((twoPlayerGame) && (!onePlayerLeft))
        mortals++;
    QuickGlidersRefresh();
    RedrawAllGrease();
}
who->isOn = false;
break;
```

- One paper = **+1 life** in one-player mode, **+2 lives** in two-player mode
  while both players are still alive.
- `onePlayerLeft` (`GliderPRO/Sources/Player.c:53`, set at
  `GliderPRO/Sources/Player.c:1507`) is the "one of the two players has run out
  of gliders" flag. Once set, papers (and every other consumable pickup) revert
  to single value. This doubling-while-both-alive rule is applied identically to
  `kBattery`, `kBands`, `kFoil` and `kHelium`.
- There is **no cap**: `mortals` is a `short` and a house with 46 papers
  (Leviathan) can push it to 48 in one player mode. `NumToString()` renders any
  value; the 20-px-wide glider-count box (section 5.2) will clip at three
  digits.
- **There is no score-threshold extra life.** The complete list of `mortals`
  writes in section 1.3 contains no comparison against `theScore`. Nor is there
  any `every N points` counter anywhere: `theScore` is only ever compared in
  `HandleDynamicScoreboard()` (against `displayedScore`) and in
  `TestHighScore()` (against the high-score table).

Paper counts per shipped house, from the census in section 2.9: Art Museum 6,
CD Demo House 15, California or Bust! 0, Castle o' the Air 10, Davis Station 2,
Demo House 0, Empty House 0, Fun House 1, Grand Prix 15, ImagineHouse PRO II 21,
In The Mirror 8, Land of Illusion 20, Leviathan 46, Metropolis 13, Nemo's Market
6, Rainbow's End 12, Sampler 0, Slumberland 37, SpacePods 9, Teddy World 36,
The Asylum Pro 13, Titanic 13.

### 3.4 Dying: `OffAMortal()`

`OffAMortal(gliderPtr thisGlider)` (`GliderPRO/Sources/Player.c:1484-1604`) is
the single funnel for every death. It is called from exactly **two** places:

| Call site | Condition |
|---|---|
| `GliderPRO/Sources/Player.c:295` | `FadeGliderOut()`: `thisGlider->frame--; if (thisGlider->frame < 0) OffAMortal(thisGlider);` |
| `GliderPRO/Sources/Player.c:1317` | `MoveGliderShredding()`: `thisGlider->frame++; if (thisGlider->frame >= 0) OffAMortal(thisGlider);` |

So every death goes through one of two animations:

**Fade-out.** `StartGliderFadingOut()` (`GliderPRO/Sources/Modes.c:75-116`) sets
`mode = kGliderFadingOut` and `frame = kLastFadeSequence - 1` = 15
(`kLastFadeSequence` = 16, `GliderPRO/Headers/GliderDefines.h:564`). Each frame
`FadeGliderOut()` decrements and draws `gliderSrc[fadeInSequence[frame]]`
(plus `kLeftFadeOffset` = 7 when facing left,
`GliderPRO/Headers/GliderDefines.h:565`), so frames 14 down to 0 are drawn — 15
drawn frames — and on the 16th call `frame` reaches -1 and `OffAMortal()` runs.
**16 frames = 0.53 s.** There are 30 `StartGliderFadingOut()` call sites across
`Interactions.c`, `Dynamics.c`, `Player.c` and `Transit.c` (enumerated in
`interactions.md`); it is the generic "you touched something lethal" death.

**Shredding.** `FlagGliderShredding(thisGlider, bounds)`
(`GliderPRO/Sources/Modes.c:366-402`) plays `kCaughtFireSound` and sets
`frame = bounds->bottom - 3`, i.e. a positive screen coordinate. While
`frame > 0` the glider is dragged downward 1 px per frame (`kDropShredSlow` = 1)
until it is fully inside the shredder, then 4 px per frame
(`kDropShredFast` = 4), with `kShredSound` playing each slow frame. When it is
fully consumed, `AddAShreddedGlider(&dest)` spawns the confetti and
`frame = kShredderCountdown` = **-68** (`GliderPRO/Sources/Player.c:17`). From
then the `else` branch counts up one per frame until `frame >= 0`, which is 68
frames later — **68 frames = 2.27 s** of confetti before `OffAMortal()`.
The only `FlagGliderShredding()` call site is
`GliderPRO/Sources/Interactions.c:1339` (hot-spot action `kShredIt` = 10).

**Burning is not a third path.** `FlagGliderBurning()`
(`GliderPRO/Sources/Modes.c:406-434`) sets `mode = kGliderBurning` and
`wasMode = kFramesToBurn` = **60**, and `MoveGliderBurning()`
(`GliderPRO/Sources/Player.c:203-227`) decrements `wasMode` each frame and calls
`StartGliderFadingOut()` when it hits 0. So burning is a 60-frame (2 s) prelude
to a fade-out: **76 frames total** from ignition to `OffAMortal()`. There are 6
`FlagGliderBurning()` call sites: `GliderPRO/Sources/Play.c:330` (restoring a
saved burning glider), `GliderPRO/Sources/Player.c:418`, `:547`, `:880`, `:918`,
`:950` and `GliderPRO/Sources/Interactions.c:1372` (hot-spot action `kBurnIt`).

`OffAMortal()` in full:

```c
void OffAMortal (gliderPtr thisGlider)
{
    if (gameOver)
        return;                                  //  1

    if (numShredded > 0)
        RemoveShreds();                          //  2

    mortals--;                                   //  3
    if (mortals < 0)
    {
        HideGlider(thisGlider);
        if (twoPlayerGame)
        {
            if (mortals < -1)                    //  4  both players are now dead
            {
                FlagGameOver();
                thisGlider->dontDraw = true;
            }
            else                                 //  5  this player only
            {
                FlagGliderInLimbo(thisGlider, false);
                thisGlider->dontDraw = true;
                onePlayerLeft = true;
                playerDead = thisGlider->which;
            }
        }
        else                                     //  6  one-player game over
        {
            FlagGameOver();
            thisGlider->dontDraw = true;
        }
    }
    else                                         //  7  ordinary death
    {
        QuickGlidersRefresh();
        HideGlider(thisGlider);
    }

    if (mortals >= 0)                            //  8  respawn
    {
        if (thisGlider->mode == kGliderGoingFoil)
            DeckGliderInFoil(thisGlider);
        FlagGliderNormal(thisGlider);
        if (playerSuicide)
            FollowTheLeader();                   //  9
        else
        {
            StartGliderFadingIn(thisGlider);     // 10
            thisGlider->dest  = thisGlider->enteredRect;
            thisGlider->whole = thisGlider->dest;
            thisGlider->destShadow.left  = thisGlider->dest.left;
            thisGlider->destShadow.right = thisGlider->dest.right;
            thisGlider->wholeShadow      = thisGlider->destShadow;
        }
    }
    else if ((mortals == -1) && (onePlayerLeft) && (!gameOver))
    {                                            // 11  drag the survivor along
        switch (otherPlayerEscaped) { ... }
        otherPlayerEscaped = kPlayerIsDeadForever;
    }
}
```

Numbered notes:

1. **The `gameOver` early return.** Once the game is flagged over, further
   deaths are ignored. This matters because `gameOver` is set 16 frames before
   the game actually stops (section 8.1), and during those 16 frames the other
   glider in a two-player game could still die.
2. `RemoveShreds()` clears the confetti sprites; `numShredded` is capped at
   `kMaxShredded` = 4 (`GliderPRO/Headers/GliderDefines.h:263`).
3. The decrement is unconditional and happens before any branch.
7. On an ordinary death the scoreboard is updated with
   `QuickGlidersRefresh()` — the *unclamped* variant, which is fine here
   because `mortals >= 0`.
   Note that the `mortals < 0` branches deliberately do **not** call
   `QuickGlidersRefresh()`, so the scoreboard keeps showing the last positive
   count during the death animation and the end sequence. The full
   `RefreshScoreboard()` issued at `GliderPRO/Sources/Play.c:513` when the
   countdown expires *does* refresh it, using the clamped
   `RefreshNumGliders()`, which is why the count shows 0 and not -1.
8. **Respawn position is `thisGlider->enteredRect`** — the rect the glider
   occupied when it last entered the current room, captured by
   `FadeGliderIn()` at `GliderPRO/Sources/Player.c:240`. So you always
   reappear where you came into the room, not at the house's start point.
9. `playerSuicide` (`GliderPRO/Sources/Player.c:53`) is set only by
   `ForceKillGlider()` (`GliderPRO/Sources/Transit.c:457` and `:466`), which is
   the two-player Delete-key "I'm stuck, kill me so I respawn next to my
   partner" escape hatch. `FollowTheLeader()`
   (`GliderPRO/Sources/Transit.c:473`) teleports the dead player to the live
   one's position and clears `playerSuicide` at `:478`.
10. `StartGliderFadingIn()` begins with `if (foilTotal <= 0) showFoil = false;`
    (`GliderPRO/Sources/Modes.c:27`), which is the only reason a respawned
    glider stops being drawn with foil.
11. The last branch handles the case where the *other* player had already
    walked off the edge of the room when this player died: the surviving glider
    is moved through the same exit so that the camera follows it. The 10
    `otherPlayerEscaped` cases and their constants:

| `otherPlayerEscaped` | Value | Action taken |
|---|---:|---|
| `kPlayerEscapedUp` | -4 | `MoveRoomToRoom(survivor, kAbove)` |
| `kPlayerEscapingUpStairs` | -8 | `MoveRoomToRoom(survivor, kAbove)` |
| `kPlayerEscapedUpStairs` | -6 | `MoveRoomToRoom(survivor, kAbove)` |
| `kPlayerEscapedDown` | -5 | `MoveRoomToRoom(survivor, kBelow)` |
| `kPlayerEscapingDownStairs` | -9 | `MoveRoomToRoom(survivor, kBelow)` |
| `kPlayerEscapedDownStairs` | -7 | `MoveRoomToRoom(survivor, kBelow)` |
| `kPlayerEscapedLeft` | -3 | `MoveRoomToRoom(survivor, kToLeft)` |
| `kPlayerEscapedRight` | -2 | `MoveRoomToRoom(survivor, kToRight)` |
| `kPlayerTransportedOut` | -10 | `TransportRoomToRoom(survivor)` |
| `kPlayerMailedOut` | -12 | `MoveMailToMail(survivor)` |
| `kPlayerDuckedOut` | -11 | `MoveDuctToDuct(survivor)` |
| `kNoOneEscaped` | -1 | `default:` — nothing |

(Constants at `GliderPRO/Headers/GliderDefines.h:596-608`.) `survivor` is
`&theGlider2` when `playerDead == kPlayer1`, else `&theGlider`. Afterwards
`otherPlayerEscaped = kPlayerIsDeadForever` = **-69**.

Note that every one of these four transition functions calls
`HandleRoomVisitation()` (section 2.3), so **a two-player death can award 100
room-visit points**. That is a real, reachable interaction between the lives
system and the scoring system.

### 3.5 What ends a game

| Condition | Where | Result |
|---|---|---|
| one-player, `mortals` reaches -1 | `GliderPRO/Sources/Player.c:1513` | `FlagGameOver()` |
| two-player, `mortals` reaches -2 | `GliderPRO/Sources/Player.c:1500` | `FlagGameOver()` |
| `numStarsRemaining` reaches 0 | `GliderPRO/Sources/Interactions.c:944` | `FlagGameOver()` |

Those are the only three `FlagGameOver()` call sites. Which of the two end
sequences runs is decided later, in `PlayGame()`, purely by the sign of
`mortals` (section 8.2).

## 4. Inventory and consumables

### 4.1 The four supply constants

`GliderPRO/Sources/Interactions.c:13-19`:

```c
#define kFloorVentLift          -6
#define kCeilingVentDrop        8
#define kFanStrength            12
#define kBatterySupply          50      // about 2 rooms worth of thrust
#define kHeliumSupply           150
#define kBandsSupply            8
#define kFoilSupply             8
```

| Consumable | Counter | Sign | Supply per pickup | Unit |
|---|---|---|---:|---|
| Battery | `batteryTotal` | positive | 50 | frames of thrust |
| Helium | `batteryTotal` | negative | 150 | frames of lift |
| Rubber bands | `bandsTotal` | positive | 8 | bands |
| Foil | `foilTotal` | positive | 8 | hits absorbed |

The author's comment "about 2 rooms worth of thrust" for 50 frames is the only
hint at the intended pacing.

### 4.2 Battery and helium share one signed counter

This is the subsystem's most surprising design decision and a port must
replicate it exactly, including the asymmetric top-up rules.

`HandleRewards()` case `kBattery` (`GliderPRO/Sources/Interactions.c:860-865`):

```c
if (batteryTotal > 0)          // positive number means battery power
    batteryTotal += kBatterySupply;
else                           // negative number means helium gas
    batteryTotal = kBatterySupply;
if ((twoPlayerGame) && (!onePlayerLeft))
    batteryTotal += kBatterySupply;
```

`HandleRewards()` case `kHelium` (`GliderPRO/Sources/Interactions.c:966-971`):

```c
if (batteryTotal < 0)          // if negative, it is already helium gas
    batteryTotal -= kHeliumSupply;
else                           // if positive, it is battery power
    batteryTotal = -kHeliumSupply;
if ((twoPlayerGame) && (!onePlayerLeft))
    batteryTotal -= kHeliumSupply;
```

Truth table for a one-player game:

| Before | Pick up battery | Pick up helium |
|---:|---:|---:|
| 0 | 50 (`else` branch: assign) | -150 (`else` branch: assign) |
| +30 | +80 (accumulate) | -150 (**+30 of battery is destroyed**) |
| -40 | +50 (**-40 of helium is destroyed**) | -190 (accumulate) |

So picking up the opposite kind **discards** whatever you were carrying rather
than adding to it — and note the boundary: `batteryTotal == 0` takes the `else`
branch in both cases, which coincidentally gives the same answer as the
accumulate branch would.

Two-player, both alive: the amounts double to 100 and -300 in the accumulate
case, and to 100 / -300 in the assign case too (assign then add).

**Battery drain.** `DoBatteryEngaged(gliderPtr thisGlider)`
(`GliderPRO/Sources/Input.c:121-158`):

```
 1  if facing == kFaceRight and not tipped:  hVel += kHyperThrust
 2  if facing == kFaceRight and     tipped:  hVel -= kHyperThrust
 3  if facing == kFaceLeft  and not tipped:  hVel -= kHyperThrust
 4  if facing == kFaceLeft  and     tipped:  hVel += kHyperThrust
 5  batteryTotal--                                     // Input.c:138
 6  if batteryTotal == 0:
 7      QuickBatteryRefresh(false)
 8      PlayPrioritySound(kFizzleSound, kFizzlePriority)
 9  else:
10      if not batteryWasEngaged: batteryFrame <- 0
11      if batteryFrame == 0: PlayPrioritySound(kThrustSound, kThrustPriority)
12      batteryFrame++
13      if batteryFrame >= 4: batteryFrame <- 0
14      batteryWasEngaged <- true
```

`kHyperThrust` is 8 and `kNormalThrust` is 5
(`GliderPRO/Sources/Input.c:14-15`); `kHeliumLift` is 4 (`:16`). So the battery
costs **1 unit per frame** and gives 8 units of horizontal acceleration instead
of 5. `kThrustSound` re-triggers every 4th frame (lines 11-13), giving a
looping thrust noise.

**Helium drain.** `DoHeliumEngaged(gliderPtr thisGlider)`
(`GliderPRO/Sources/Input.c:160-183`):

```
 1  thisGlider->vDesiredVel <- -kHeliumLift        //  -4, upward
 2  batteryTotal++                                 // Input.c:163  toward zero
 3  if batteryTotal == 0:
 4      QuickBatteryRefresh(false)
 5      PlayPrioritySound(kFizzleSound, kFizzlePriority)
 6      batteryWasEngaged <- false
 7  else:
 8      if not batteryWasEngaged: batteryFrame <- 0
 9      if batteryFrame == 0: PlayPrioritySound(kHissSound, kHissPriority)
10      batteryFrame++
11      if batteryFrame >= 4: batteryFrame <- 0
12      batteryWasEngaged <- true
```

So helium also costs **1 unit per frame** (moving the negative counter toward
zero), gives a constant -4 desired vertical velocity, and hisses instead of
thrusting. Because `kHeliumSupply` is 150 versus the battery's 50, one helium
canister lasts three times as long as one battery: **5 s vs 1.67 s** at 30 fps.

Note that `DoHeliumEngaged()` clears `batteryWasEngaged` when it runs out but
`DoBatteryEngaged()` does not — an asymmetry with no observable effect, since
`GetInput()` clears it anyway when the key is released.

**The key gate.** `GetInput()` (`GliderPRO/Sources/Input.c:330-342`):

```c
if ((BitTst(&theKeys, thisGlider->battKey)) && (batteryTotal != 0) &&
        (thisGlider->mode == kGliderNormal))
{
    if (batteryTotal > 0)
        DoBatteryEngaged(thisGlider);
    else
        DoHeliumEngaged(thisGlider);
}
else
    batteryWasEngaged = false;
```

Three conditions: the key is down, the counter is non-zero, and the glider is in
`kGliderNormal` mode (0). The last one means battery and helium are unusable
while fading, burning, shredding, tipping, transporting, ducting or mailing.

**The demo path skips two of the three gates.** `GetDemoInput()`
(`GliderPRO/Sources/Input.c:243-245`), `case 2`:

```c
if (batteryTotal > 0)
    DoBatteryEngaged(thisGlider);
else
    DoHeliumEngaged(thisGlider);
```

There is no `batteryTotal != 0` test and no mode test. With `batteryTotal == 0`
this calls `DoHeliumEngaged()`, which increments to 1 — so the recorded demo can
drive the counter positive from empty and then start draining a battery it never
picked up. A port replaying the shipped `demo` resource must reproduce this or
the demo will desynchronise.

### 4.3 Rubber bands

`kMaxRubberBands` is 2 (`GliderPRO/Headers/GliderDefines.h:261`) — that is the
number of bands that can be **in flight** at once, not the number you can carry.
`kRubberBandVelocity` is 20 (`GliderPRO/Sources/RubberBands.c:12`).

`AddBand(gliderPtr thisGlider, short h, short v, Boolean direction)`
(`GliderPRO/Sources/RubberBands.c:256-290`):

```
 1  if numBands >= kMaxRubberBands: return false        // no free slot
 2  bands[numBands].mode  <- 0
 3  bands[numBands].count <- 0
 4  bands[numBands].vVel  <- thisGlider->tipped ? -2 : 0
 5  QSetRect(&bands[numBands].dest, h - 8, v - 3, h + 8, v + 3)
 6  if direction == kFaceLeft:
 7      dest.left -= 32 ; dest.right -= 32 ; hVel <- -kRubberBandVelocity
 8  else:
 9      dest.left += 32 ; dest.right += 32 ; hVel <- +kRubberBandVelocity
10  thisGlider->hVel -= (bands[numBands].hVel / 2)      // recoil
11  numBands++
12  PlayPrioritySound(kFireBandSound, kFireBandPriority)
13  return true
```

The recoil at line 10 is 10 px/frame of horizontal velocity **against** the shot
direction. `bandType` is `{Rect dest; short mode, count; short hVel, vVel;}`
(`GliderPRO/Headers/GliderStructs.h:274-279`).

`GetInput()`'s band gate (`GliderPRO/Sources/Input.c:344-364`):

```c
if ((BitTst(&theKeys, thisGlider->bandKey)) && (bandsTotal > 0) &&
        (thisGlider->mode == kGliderNormal))
{
    if (!thisGlider->fireHeld)
    {
        if (AddBand(thisGlider, thisGlider->dest.left + 24,
                    thisGlider->dest.top + 10, thisGlider->facing))
        {
            bandsTotal--;
            if (bandsTotal <= 0)
                QuickBandsRefresh(false);
            thisGlider->fireHeld = true;
        }
    }
}
else
    thisGlider->fireHeld = false;
```

Facts:

- The band spawns from the glider's centre-ish: `dest.left + 24` (which is
  `kHalfGliderWide`, `GliderPRO/Headers/GliderDefines.h:551`) and
  `dest.top + 10`, then `AddBand` offsets it 32 px in the fire direction.
- **`bandsTotal` is only decremented if `AddBand()` returned true.** With two
  bands already in flight the shot is silently swallowed and no band is spent.
- `fireHeld` makes it one band per key press, not per frame. It is cleared the
  instant the key is released — or on any frame where `bandsTotal` is 0 or the
  mode is not `kGliderNormal`, because those take the `else`.
- `QuickBandsRefresh(false)` on reaching 0 erases the badge (section 5.8).

The demo path again differs (`GliderPRO/Sources/Input.c:250-262`, `case 3`):
there is **no `bandsTotal > 0` test**, so a demo can fire bands it does not have
and drive `bandsTotal` negative.

`numBands` is reset to 0 by `NewGame()` (`GliderPRO/Sources/Play.c:113`).

### 4.4 Aluminium foil

Foil is a damage shield. `foilTotal` counts the number of hits it will absorb.
`kFoilSupply` is 8. It is the only consumable with **six distinct decrement
sites**, and they do not all behave the same way.

| # | Site | Trigger | Rate | Extra |
|---:|---|---|---|---|
| 1 | `GliderPRO/Sources/Interactions.c:82` | `GliderHitTop()` — hitting the underside of something with the top of the glider | 1 per hit | plays `kFoilHitSound`; reflects `hVel` |
| 2 | `GliderPRO/Sources/Interactions.c:1232` | hot-spot action `kDissolveIt` (5) when `GliderHitTop()` was false | 1 per frame in contact | — |
| 3 | `GliderPRO/Sources/Interactions.c:1333` | hot-spot action `kShredIt` (10) | 1 per frame in contact | plays `kFoilHitSound` |
| 4 | `GliderPRO/Sources/Interactions.c:1366` | hot-spot action `kBurnIt` (13) | 1 per frame in contact | plays `kSizzleSound`; also sets `vDesiredVel = kFloorVentLift` (-6) |
| 5 | `GliderPRO/Sources/Dynamics.c:62` | collision with a moving object (`dinahs[]`) | 1 per **even** frame | plays `kFoilHitSound`; shoves the glider |
| 6 | `GliderPRO/Sources/Interactions.c:1186` | microwave (section 4.5) | **all of it** | sets `foilTotal = 0` |

All five of sites 1-5 use the identical idiom:

```c
foilTotal--;
if (foilTotal <= 0)
    StartGliderFoilLosing(thisGlider);
```

Site 5 is the odd one out (`GliderPRO/Sources/Dynamics.c:44-72`):

```c
if ((foilTotal > 0) || (thisGlider->mode == kGliderLosingFoil))
{
    thisGlider->hDesiredVel = ±kShoveVelocity;      // sign from IsRectLeftOfRect()
    if (dinahs[who].vVel < 0)
        thisGlider->vDesiredVel = dinahs[who].vVel;
    PlayPrioritySound(kFoilHitSound, kFoilHitPriority);
    if ((evenFrame) && (foilTotal > 0))
    {
        foilTotal--;
        if (foilTotal <= 0)
            StartGliderFoilLosing(thisGlider);
    }
}
else
{
    StartGliderFadingOut(thisGlider);
    PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
}
```

**`evenFrame` halves the rate**, so foil lasts 16 frames against a moving object
instead of 8. This is the only place in the game where `evenFrame` gates a
consumable. `CheckDynamicCollision()` applies only when the glider's mode is one
of `kGliderNormal` (0), `kGliderFaceLeft` (7), `kGliderFaceRight` (8),
`kGliderBurning` (9), `kGliderGoingFoil` (18) or `kGliderLosingFoil` (19)
(`GliderPRO/Sources/Dynamics.c:44-50`).

The shape shared by sites 2, 3 and 4 is "if you have foil, spend it; if you do
not, die". `kDissolveIt` (`GliderPRO/Sources/Interactions.c:1218-1244`):

```c
if (thisGlider->mode != kGliderFadingOut)
{
    if ((foilTotal > 0) || (thisGlider->mode == kGliderLosingFoil))
    {
        if (GliderHitTop(thisGlider, &(who->bounds)))
        {
            StartGliderFadingOut(thisGlider);
            PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
        }
        else
        {
            if (foilTotal > 0)
            {
                foilTotal--;
                if (foilTotal <= 0)
                    StartGliderFoilLosing(thisGlider);
            }
        }
    }
    else
    {
        StartGliderFadingOut(thisGlider);
        PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
    }
}
```

Note the `|| (thisGlider->mode == kGliderLosingFoil)` in every one of these
tests: during the ~9-frame "losing foil" animation you are still protected even
though `foilTotal` is already 0. And note the nested `if (foilTotal > 0)` guard
before each decrement, which prevents `foilTotal` going negative during that
animation.

**Foil hit sounds without foil consumption.** Five sites test `foilTotal > 0`
only to pick a sound, and never decrement:

| Site | Context |
|---|---|
| `GliderPRO/Sources/Interactions.c:163` | `BounceGlider()` — bouncing off furniture |
| `GliderPRO/Sources/Interactions.c:540` | hitting the left wall |
| `GliderPRO/Sources/Interactions.c:585` | hitting the right wall (`CheckEscapeRight`) |
| `GliderPRO/Sources/Interactions.c:630` | wall variant |
| `GliderPRO/Sources/Interactions.c:675` | wall variant |

All five read:

```c
if (foilTotal > 0)
    PlayPrioritySound(kFoilHitSound, kFoilHitPriority);
else
    PlayPrioritySound(kHitWallSound, kHitWallPriority);
```

So walls and furniture sound metallic while you are wrapped in foil but cost you
nothing.

**Foil visual state.** `showFoil` (`GliderPRO/Sources/Play.c:53`) is the
"draw the foil-wrapped glider sprite" flag, distinct from `foilTotal`. It is set
by `DeckGliderInFoil()` (`GliderPRO/Sources/Player.c:1142`) and cleared by
`RemoveFoilFromGlider()` (`GliderPRO/Sources/Player.c:1202`), and both are
called in pairs from the mode-transition functions
(`GliderPRO/Sources/Modes.c:83/85`, `123/125`, `140/142`, `184/186`, `222`).
`StartGliderFadingIn()` and `StartGliderTransportingIn()` both begin
`if (foilTotal <= 0) showFoil = false;` (`GliderPRO/Sources/Modes.c:27` and
`:52`), which is the resynchronisation point.

In a **two-player** game the foil sprite sheet is swapped wholesale:
`DeckGliderInFoil()` (`GliderPRO/Sources/Player.c:1142-1168`) reloads
`kGliderFoilPictID` = 3976 into `glidSrcMap` and `kGliderFoil2PictID` = 3963
into `glid2SrcMap`, and `RemoveFoilFromGlider()` reloads `kGliderPictID` = 3999
and `kGlider2PictID` = 3974 (`GliderPRO/Headers/GliderDefines.h:565-568`).
Because both gliders share one `showFoil` and one `foilTotal`, **foil is a
shared resource in two-player mode and both gliders wear it at once.**

The two transition animations are 9 frames each:

`MoveGliderFoilGoing()` (`GliderPRO/Sources/Player.c:1170-1200`):

```
1  thisGlider->frame++
2  if frame > 8:  FlagGliderNormal(thisGlider)
3  else if frame < 5:  src = mask = gliderSrc[(10 - frame) (+kLeftFadeOffset if facing left)]
4  else:               DeckGliderInFoil(thisGlider)
5  MoveGlider(thisGlider)
```

`MoveGliderFoilLosing()` (`GliderPRO/Sources/Player.c:1230-1258`) is identical
except line 4 calls `RemoveFoilFromGlider()`. So frames 1-4 are a
sparkle/wrapping animation and frames 5-8 show the final sprite; at frame 9 the
glider returns to `kGliderNormal`.

`StartGliderFoilGoing()` (`GliderPRO/Sources/Modes.c:581`) and
`StartGliderFoilLosing()` (`GliderPRO/Sources/Modes.c:605`) both early-return if
already in that mode or in `kGliderInLimbo` (21), both set `frame = 0` and
`src = mask = gliderSrc[10 - frame]` (+7 if facing left), and both call
`QuickFoilRefresh(false)`. `StartGliderFoilLosing()` additionally plays
`kFizzleSound` at `kFizzlePriority` (703).

### 4.5 The microwave: the only bulk-wipe

`HandleMicrowaveAction(hotPtr who, gliderPtr thisGlider)`
(`GliderPRO/Sources/Interactions.c:1159-1196`), reached from hot-spot action
`kMicrowaveIt` (22):

```c
if (who->stillOver)
    return;                                     // only fires on entry
killed = false;
whoLinked = who->who;
if (masterObjects[whoLinked].theObject.data.g.state)
{
    kills = (short)masterObjects[whoLinked].theObject.data.g.byte0;
    if (((kills & 0x0001) == 0x0001) && (bandsTotal > 0))
    {
        bandsTotal = 0;
        killed = true;
        QuickBandsRefresh(false);
    }
    if (((kills & 0x0002) == 0x0002) && (batteryTotal != 0))
    {
        batteryTotal = 0;
        killed = true;
        QuickBatteryRefresh(false);
    }
    if (((kills & 0x0004) == 0x0004) && (foilTotal > 0))
    {
        foilTotal = 0;
        killed = true;
        StartGliderFoilLosing(thisGlider);
    }
}
if (killed)
    PlayPrioritySound(kMicrowavedSound, kMicrowavedPriority);
```

The bitmask lives in the object's `applianceType.byte0`
(`GliderPRO/Headers/GliderStructs.h:64-72`; within a 12-byte object record,
`data.g.byte0` is at object offset 8):

| Bit | Mask | Destroys |
|---:|---|---|
| 0 | `0x0001` | rubber bands |
| 1 | `0x0002` | battery **and** helium (the test is `!= 0`, not `> 0`) |
| 2 | `0x0004` | foil |

Notes:

- `who->stillOver` makes this a one-shot on entering the microwave's hot spot,
  not a per-frame drain.
- The microwave must be switched on (`data.g.state`).
- Bit 1 destroys helium too, because the test is `batteryTotal != 0`.
- `kMicrowavedSound` is 8, `snd ` 1008 'Miked' (10 922 bytes), priority
  `kMicrowavedPriority` = 811. It only plays if something was actually
  destroyed.
- `kMicrowave` object code is `0x6A` and `kShredder` is `0x61`
  (`GliderPRO/Headers/GliderDefines.h`).

### 4.6 Complete inventory mutation census

Every write to the three counters, in file order. This is the whole table; there
are no others.

`batteryTotal` — 11 writes:

| Site | Statement | Context |
|---|---|---|
| `GliderPRO/Sources/Play.c:320` | `= smallGame.energy` | resume |
| `GliderPRO/Sources/Play.c:344` | `= 0` | new game |
| `GliderPRO/Sources/Input.c:138` | `--` | battery thrust, 1/frame |
| `GliderPRO/Sources/Input.c:163` | `++` | helium lift, 1/frame |
| `GliderPRO/Sources/Interactions.c:861` | `+= kBatterySupply` | battery pickup, accumulate |
| `GliderPRO/Sources/Interactions.c:863` | `= kBatterySupply` | battery pickup, replacing helium |
| `GliderPRO/Sources/Interactions.c:865` | `+= kBatterySupply` | two-player doubling |
| `GliderPRO/Sources/Interactions.c:967` | `-= kHeliumSupply` | helium pickup, accumulate |
| `GliderPRO/Sources/Interactions.c:969` | `= -kHeliumSupply` | helium pickup, replacing battery |
| `GliderPRO/Sources/Interactions.c:971` | `-= kHeliumSupply` | two-player doubling |
| `GliderPRO/Sources/Interactions.c:1180` | `= 0` | microwave bit 1 |

`bandsTotal` — 7 writes:

| Site | Statement | Context |
|---|---|---|
| `GliderPRO/Sources/Play.c:321` | `= smallGame.bands` | resume |
| `GliderPRO/Sources/Play.c:345` | `= 0` | new game |
| `GliderPRO/Sources/Input.c:256` | `--` | demo fired a band |
| `GliderPRO/Sources/Input.c:355` | `--` | player fired a band |
| `GliderPRO/Sources/Interactions.c:882` | `+= kBandsSupply` | bands pickup |
| `GliderPRO/Sources/Interactions.c:884` | `+= kBandsSupply` | two-player doubling |
| `GliderPRO/Sources/Interactions.c:1174` | `= 0` | microwave bit 0 |

`foilTotal` — 10 writes:

| Site | Statement | Context |
|---|---|---|
| `GliderPRO/Sources/Play.c:322` | `= smallGame.foil` | resume |
| `GliderPRO/Sources/Play.c:346` | `= 0` | new game |
| `GliderPRO/Sources/Interactions.c:82` | `--` | `GliderHitTop()` |
| `GliderPRO/Sources/Interactions.c:910` | `+= kFoilSupply` | foil pickup |
| `GliderPRO/Sources/Interactions.c:912` | `+= kFoilSupply` | two-player doubling |
| `GliderPRO/Sources/Interactions.c:1186` | `= 0` | microwave bit 2 |
| `GliderPRO/Sources/Interactions.c:1232` | `--` | `kDissolveIt` |
| `GliderPRO/Sources/Interactions.c:1333` | `--` | `kShredIt` |
| `GliderPRO/Sources/Interactions.c:1366` | `--` | `kBurnIt` |
| `GliderPRO/Sources/Dynamics.c:62` | `--` | moving-object collision, even frames only |

`numStarsRemaining` — 3 writes:

| Site | Statement |
|---|---|
| `GliderPRO/Sources/Play.c:312` | `= smallGame.wasStarsLeft` (resume) |
| `GliderPRO/Sources/Play.c:314` | `= CountStarsInHouse()` (new game) |
| `GliderPRO/Sources/Interactions.c:942` | `--` (star taken) |

`CountStarsInHouse()` (`GliderPRO/Sources/Banner.c:89-111`):

```c
numStars = 0;
numRooms = (*thisHouse)->nRooms;
for (i = 0; i < numRooms; i++)
    if ((*thisHouse)->rooms[i].suite != kRoomIsEmpty)
        for (h = 0; h < kMaxRoomObs; h++)
            if ((*thisHouse)->rooms[i].objects[h].what == kStar)
                numStars++;
return (numStars);
```

Note it scans all 24 object slots regardless of `numObjects`, and skips
tombstone rooms. My Python reimplementation reproduces the star counts in
section 2.9.

## 5. The scoreboard

### 5.1 What it is made of

The scoreboard is a 20-px-tall metal plate that spans the full width of the
play area. It is assembled in five offscreen GWorlds, all created in
`InitScoreboardMap()` (`GliderPRO/Sources/StructuresInit.c:59-163`) at
`kPreferredDepth` (8 bpp, `GliderPRO/Headers/GliderDefines.h`), and declared in
`GliderPRO/Headers/Scoreboard.h:11-15`:

| GWorld | Size | Content | Source rect |
|---|---|---|---|
| `boardSrcMap` | `W` x 20 | the composited plate, from `PICT 1997` | `boardSrcRect` |
| `badgeSrcMap` | 32 x 66 | the 4 consumable badges + 4 blanks, from `PICT 1996` | `badgeSrcRect` |
| `boardTSrcMap` | 256 x 12 | scratch pad for the room-name text | `boardTSrcRect` |
| `boardGSrcMap` | 20 x 10 | scratch pad for the glider count | `boardGSrcRect` |
| `boardPSrcMap` | 64 x 10 | scratch pad for the score | `boardPSrcRect` |

where `W = RectWide(&houseRect) = min(screenWidth, kMaxViewWidth)` and
`kMaxViewWidth` = 1536 (`GliderPRO/Headers/GliderDefines.h:267`).

The memory budget in `GliderPRO/Sources/Environ.c:594-596` accounts for these as
`W * 21 * depth / 8` for the board plus a flat `6396 * depth / 8` for "more
scoreboard". (6396 does not equal 2112 + 3072 + 200 + 640 = 6024, the true pixel
count of the four fixed maps, so the author's arithmetic is approximate. It is
only a "do we have enough RAM" estimate and has no effect on behaviour.)

Constants private to `GliderPRO/Sources/Scoreboard.c:15-21`:

| Name | Value | Meaning |
|---|---:|---|
| `kGrayBackgroundColor` | 251 | plate grey (`#555555`) used to clear the text pads at 8 bpp |
| `kGrayBackgroundColor4` | 10 | the same at 4 bpp |
| `kFoilBadge` | 0 | index into the three badge rect arrays |
| `kBandsBadge` | 1 | |
| `kBatteryBadge` | 2 | |
| `kHeliumBadge` | 3 | |
| `kScoreRollAmount` | 13 | points added to `displayedScore` per frame |

Module globals (`GliderPRO/Sources/Scoreboard.c:29-42`): `boardSrcRect`,
`badgeSrcRect`, `boardDestRect`, `boardTSrcRect`, `boardTDestRect`,
`boardGSrcRect`, `boardGDestRect`, `boardPSrcRect`, `boardPDestRect`,
`boardPQDestRect`, `boardGQDestRect`, `badgesBlankRects[4]`,
`badgesBadgesRects[4]`, `badgesDestRects[4]`, `long displayedScore`,
`short wasScoreboardMode`, `Boolean doRollScore`.

### 5.2 Exact geometry

`ZeroRectCorner()` (`GliderPRO/Sources/RectUtils.c:57-63`) slides a rect so its
top-left is (0,0) by doing `right -= left; bottom -= top; left = top = 0;`.
`QSetRect(r, l, t, r, b)` (`GliderPRO/Sources/RectUtils.c:210-216`) takes
**left, top, right, bottom** in that order.

Derivation, in order of execution:

```
 1  houseRect          = thisMac.screen                     // InterfaceInit.c:196
 2  houseRect.bottom  -= kScoreboardTall                    //  = 20     :197
 3  if houseRect.right  > 1536: houseRect.right  = 1536      //  :198-199
 4  if houseRect.bottom > 1026: houseRect.bottom = 1026      //  :200-201
 5  W = RectWide(&houseRect)                                 //  = min(screenW,1536)
 6  boardSrcRect = houseRect; ZeroRectCorner(&boardSrcRect)  // StructuresInit.c:71-72
 7  boardSrcRect.bottom = kScoreboardTall                    //  = (0,0,20,W)  :73
 8  boardHOffset = (W >= 640) ? (W - 1536)/2 : -576          //  :77-80
 9  draw PICT 1997 into boardSrcMap at x = boardHOffset      //  :87-89
10  boardDestRect = boardSrcRect offset (0, -20) = (-20,0,0,W)  // :97-98
11  textHOffset = (W - 640)/2 ; if (textHOffset < 0) textHOffset = -128   // :100-102
```

Note the C semantics at step 8 and 11: integer division of a negative dividend
truncates **toward zero**, so `(640-1536)/2 == -448` but `(641-1536)/2 == -447`
(a Go `/` on ints behaves the same way, but a naive `>>1` or a Python `//` does
not). For every even `W >= 640` the identity
`boardHOffset == textHOffset - 448` holds exactly, and it also holds for every
`W < 640` because both branches are hard-coded (`-576` and `-128`, difference
-448). For odd `W` the two disagree by 1 px.

All eleven rectangles, in main-window-local coordinates. Let `t = textHOffset`:

| Rect | Value | Notes |
|---|---|---|
| `boardSrcRect` | (0, 0, 20, W) | l,t,r,b = 0, 0, W, 20 |
| `boardDestRect` | (-20, 0, 0, W) | **before** `AdjustScoreboardHeight()` |
| `badgeSrcRect` | (0, 0, 32, 66) | whole badge sheet |
| `boardTSrcRect` | (0, 0, 256, 12) | room-title pad |
| `boardTDestRect` | l=137+t, t=5, r=393+t, b=17 | into `boardSrcMap` |
| `boardGSrcRect` | (0, 0, 20, 10) | glider-count pad |
| `boardGDestRect` | l=526+t, t=5, r=546+t, b=15 | into `boardSrcMap` |
| `boardPSrcRect` | (0, 0, 64, 10) | score pad |
| `boardPDestRect` | l=570+t, t=5, r=634+t, b=15 | into `boardSrcMap` |
| `boardGQDestRect` | l=526+t, t=-15, r=546+t, b=-5 | `boardGDestRect` offset (0,-20), into the window |
| `boardPQDestRect` | l=570+t, t=-15, r=634+t, b=-5 | `boardPDestRect` offset (0,-20), into the window |

Badge rectangles (`GliderPRO/Sources/StructuresInit.c:135-160`), all three arrays
indexed by `kFoilBadge`/`kBandsBadge`/`kBatteryBadge`/`kHeliumBadge` = 0/1/2/3:

| i | `badgesBlankRects[i]` (in the 32x66 sheet) | `badgesBadgesRects[i]` | `badgesDestRects[i]` (window) |
|---:|---|---|---|
| 0 foil | l=0, t=0, r=16, b=16 | l=16, t=0, r=32, b=16 | l=432+t, t=-18, r=448+t, b=-2 |
| 1 bands | l=0, t=16, r=16, b=32 | l=16, t=16, r=32, b=32 | l=449+t, t=-18, r=465+t, b=-2 |
| 2 battery | l=0, t=32, r=16, b=49 | l=16, t=32, r=32, b=49 | l=467+t, t=-19, r=483+t, b=-2 |
| 3 helium | l=0, t=49, r=16, b=66 | l=16, t=49, r=32, b=66 | l=467+t, t=-19, r=483+t, b=-2 |

**The battery and helium badges share one screen slot** (identical dest rects) —
they are mutually exclusive because they share one counter (section 4.2).

Worked example, 640x480 screen (`t` = 0, `W` = 640, `boardHOffset` = -448),
positions given relative to the top of the 20-px plate:

| Element | Plate-relative x | Plate-relative y |
|---|---|---|
| "Glider" wordmark (artwork) | 34 .. 85 | 4 .. 15 |
| gold glyph after the wordmark (artwork) | 88 .. 94 | 4 .. 15 |
| vertical divider bar (artwork) | 126 .. 129 | 0 .. 19 |
| room-name text pad | 137 .. 392 | 5 .. 16 |
| foil badge | 432 .. 447 | 2 .. 17 |
| rubber-band badge | 449 .. 464 | 2 .. 17 |
| battery / helium badge | 467 .. 482 | 1 .. 17 |
| paper-glider icon (artwork) | 491 .. 524 | 2 .. 16 |
| glider-count text pad | 526 .. 545 | 5 .. 14 |
| star icon (artwork) | 552 .. 568 | 2 .. 16 |
| score text pad | 570 .. 633 | 5 .. 14 |

The artwork extents above are the full inked footprint **including the
one-pixel drop shadow** (colour-table index 253) that the wordmark and the
gold glyph both carry; the glyph's gold body alone is x 88 .. 93, y 4 .. 14.

Worked example, 1024x768 screen: `t` = 192, `boardHOffset` = -256, `W` = 1024.
Every text/badge x above shifts right by 192, and the artwork shifts right by
192 as well, so the relative layout is identical; the extra 384 px are filled by
the decorative wings of `PICT 1997` (see 5.3).

Worked example, 512x384 screen: `numNeighbors` is forced to 1
(`GliderPRO/Sources/Main.c:191-192`, which tests
`thisMac.screen.right <= 512`), `t` = -128, `boardHOffset` = -576, `W` = 512.
Text pads land at: title 9..264, badges 304..319 / 321..336 / 339..354,
glider count 398..417, score 442..505; artwork paper-glider icon 363..396 and
star icon 424..440. Still consistent, because the -448 invariant holds.

### 5.3 `PICT 1997`, the plate artwork, decoded

Verified by decoding the resource out of `GliderPRO/Glider PRO.r` with a
`PackBitsRect` decoder:

```
PICT 1997  picFrame = (0, 0, 20, 1536)   pixelSize = 8   rowBytes = 1536
opcode 0x0098 (PackBitsRect), 20 rows, 1536 columns
palette indices present: 0, 2, 4, 11, 17, 23, 42, 43, 51, 78, 137, 172,
                         248, 249, 250, 251, 252, 253, 254, 255
```

Every "plain" column of the plate has this exact 20-entry vertical profile
(observed identically in 536 of the 1536 columns):

| y | index | RGB |
|---:|---:|---|
| 0 | 248 | `#AAAAAA` |
| 1 | 250 | `#777777` |
| 2 .. 16 | 251 | `#555555` |
| 17 | 172 | `#333333` |
| 18 | 254 | `#111111` |
| 19 | 255 | `#000000` |

That is a light top highlight, a 15-row body in `#555555` — exactly
`kGrayBackgroundColor` 251, which is why the three text pads clear to that index
— and a three-row dark bevel at the bottom.

The non-plain column ranges, measured, are:

| PICT x range | Width | Content |
|---|---:|---|
| 0 .. 445 | 446 | decorative "scribble" pattern, index 42 (`#CCCCFF`) with 172 shading |
| 446 .. 449 | 4 | vertical divider bar: 172, 254, 248, 250 left to right |
| 450 .. 481 | 32 | plain |
| 482 .. 533 | 52 | the word "Glider" in gold (11 `#FFCC00`, 17 `#FF9900`, 23 `#FF6600`) with 4 `#FFFF33` highlights and 253/172 shadow |
| 534 .. 535 | 2 | plain |
| 536 .. 542 | 7 | a 6x12 gold glyph made of three 3-row horizontal bars at y 4-6, 8-10 and 12-14 |
| 543 .. 573 | 31 | plain |
| 574 .. 577 | 4 | vertical divider bar (same profile as 446..449) |
| 578 .. 938 | 361 | plain — this is where the room-name pad lands |
| 939 .. 972 | 34 | paper-glider icon in white (0) and light blue (78 `#99CCFF`) with a 253 dotted trail |
| 973 .. 999 | 27 | plain |
| 1000 .. 1016 | 17 | five-pointed star icon in pale yellow (2 `#FFFF99`) with brown outline (51 `#CC9966`, 137 `#663300`) |
| 1017 .. 1085 | 69 | plain |
| 1086 .. 1089 | 4 | vertical divider bar |
| 1090 .. 1535 | 446 | decorative scribble, mirror image of 0 .. 445 |

The three divider bars and the two decorative wings are exactly symmetric about
x = 768 (`1535 - 449 == 1086`, `1535 - 445 == 1090`, `1535 - 0 == 1535`), but the
central content (wordmark at 482..542, icons at 939..1016) is **not** — it is
positioned for a 640-px-wide viewport centred at 768, i.e. PICT x 448..1087.
On a 640-wide screen you therefore see exactly 2 px of the left divider at
plate x 0..1 and 2 px of the right divider at plate x 638..639.

The vertical divider bar profile, read out of the decode: for its four columns
the y=2..16 body indices are 172, 254, 248, 250 — an engraved groove.

### 5.4 `PICT 1996`, the badge sheet, decoded

```
PICT 1996  picFrame = (0, 0, 66, 32)   pixelSize = 8
```

The sheet is split vertically: **x 0..15 is the "blank" half and x 16..31 is the
"badge" half.** The four horizontal bands are y 0..15 (foil), 16..31 (rubber
bands), 32..48 (battery) and 49..65 (helium), matching the rect table in 5.2.

Observed blank-half column content, per band:

| Band | Blank rows | Indices top to bottom |
|---|---|---|
| foil (dest y 2..17) | 0..15 | 251 x 15 then 250 |
| bands (dest y 2..17) | 16..31 | 251 x 15 then 250 |
| battery (dest y 1..17) | 32..48 | 250, 251 x 15, 250 |
| helium (dest y 1..17) | 49..65 | 250, 251 x 15, 250 |

Compare with the plate profile in 5.3: the blank strips reproduce the plate's
y=1 highlight (250) correctly at the top, but their **last row is 250 where the
plate has 172**. So erasing a badge leaves a one-pixel-high `#777777` line at
plate y 17 instead of `#333333`. This is a genuine artwork defect in the shipped
resource, verified by decoding it; a pixel-exact port must reproduce it (or
knowingly not).

The four badge images themselves (x 16..31) are, by dominant palette index:
foil — a wrapped roll in mid greys with 237/246 highlights; rubber bands — a box
with red (index 227-ish) bands; battery — a cell body with a `#00EE00`/`#0000FF`
label; helium — a vertical gas cylinder. All four are drawn on the 251 plate
background with 253/254 drop shadows on their right and bottom edges, and all
four are outlined in 255.

Badges are **never** composited into `boardSrcMap`; they are always blitted
straight from `badgeSrcMap` to the window. That is why `RefreshScoreboard()`
re-issues all three badge refreshes after blitting the plate
(`GliderPRO/Sources/Scoreboard.c:65-67`) — the plate blit erases them.

### 5.5 Text rendering

All three text pads use the same code shape. Taking `RefreshPoints()`
(`GliderPRO/Sources/Scoreboard.c:231-264`) as the canonical form:

```
 1  SetPort((GrafPtr)boardPSrcMap)
 2  GetForeColor(&wasColor)
 3  Index2Color(thisMac.isDepth == 4 ? kGrayBackgroundColor4 : kGrayBackgroundColor, &rgb)
 4  RGBForeColor(&rgb)
 5  PaintRect(&boardPSrcRect)                     // clear the pad to plate grey
 6  RGBForeColor(&wasColor)
 7  NumToString(theScore, scoreStr)
 8  MoveTo(1, 10) ; ForeColor(blackColor) ; DrawString(scoreStr)   // shadow
 9  MoveTo(0, 9)  ; ForeColor(whiteColor) ; DrawString(scoreStr)   // face
10  ForeColor(blackColor)
11  CopyBits(boardPSrcMap -> boardSrcMap, boardPSrcRect -> boardPDestRect, srcCopy)
12  displayedScore = theScore
```

- The text is **left-aligned at x = 0**, never centred and never right-aligned.
  A five-digit score in Geneva 12 bold is about 35 px, so it sits at the left of
  the 64-px pad; a score of 1 000 000 would need 49 px and still fit, but the
  pad clips at 64 px (the GWorld bounds do the clipping, `DrawString` has no
  idea).
- The drop shadow is a **black copy at (1,10) drawn first**, then the **white
  face at (0,9)**. Baseline y = 9 in a 10-row pad; y = 10 for the shadow.
- Font state is set once at creation time —
  `TextFont(applFont); TextSize(12); TextFace(bold);` on each pad's port
  (`GliderPRO/Sources/StructuresInit.c:109-111`, `:118-120`, `:131-133`) — and
  never changed. `applFont` is the application font (Geneva on a stock system).
- `Index2Color()` maps the palette index through the **current device's** colour
  table, so the clear colour follows whatever `clut` is installed.

The three variants differ only in pad, source, and what they stringify:

| Function | Lines | Pad | Stringifies | Blits to |
|---|---|---|---|---|
| `RefreshRoomTitle(mode)` | 136-188 | `boardTSrcMap` 256x12 | see below | `boardSrcMap` at `boardTDestRect` |
| `RefreshNumGliders()` | 192-227 | `boardGSrcMap` 20x10 | `mortals`, **clamped to >= 0** | `boardSrcMap` at `boardGDestRect` |
| `RefreshPoints()` | 231-264 | `boardPSrcMap` 64x10 | `theScore` | `boardSrcMap` at `boardPDestRect` |
| `QuickGlidersRefresh()` | 268-299 | `boardGSrcMap` | `mortals`, **not clamped** | the window at `boardGQDestRect` |
| `QuickScoreRefresh()` | 303-334 | `boardPSrcMap` | `displayedScore` | the window at `boardPQDestRect` |

Two asymmetries that matter:

1. `RefreshNumGliders()` clamps (`GliderPRO/Sources/Scoreboard.c:209-211`):
   `displayMortals = mortals; if (displayMortals < 0) displayMortals = 0;`
   `QuickGlidersRefresh()` does **not**
   (`GliderPRO/Sources/Scoreboard.c:284`: `NumToString((long)mortals, ...)`).
   Since `OffAMortal()` only calls `QuickGlidersRefresh()` on the
   `mortals >= 0` path (`GliderPRO/Sources/Player.c:1519`), a negative count is
   never actually printed — but a port that unifies the two functions will
   change behaviour if it ever is.
2. `RefreshPoints()` prints `theScore` and then sets
   `displayedScore = theScore` (line 263), whereas `QuickScoreRefresh()` prints
   `displayedScore`. So a full scoreboard refresh **snaps the rolling score
   counter to its final value**, cancelling any roll in progress (section 2.10).

`RefreshRoomTitle(short mode)` switches on the title mode
(`GliderPRO/Headers/GliderDefines.h:619-621`), drawing the same string twice for
the shadow pass:

| `mode` | Value | String drawn |
|---|---:|---|
| `kNormalTitleMode` | 0 | `thisRoom->name` (a `Str27`, so at most 27 characters) |
| `kEscapedTitleMode` | 1 | `"Hit Delete key if unable to Follow"` |
| `kSavingTitleMode` | 2 | `"Saving Game…"` (with a real Mac-Roman ellipsis, byte `0xC9`) |

Call-site census for `RefreshScoreboard(mode)` — 27 sites, complete:

| Mode | Sites |
|---|---|
| `kNormalTitleMode` (9) | `GliderPRO/Sources/Input.c:71`, `GliderPRO/Sources/Play.c:164`, `:404`, `:510`, `:815`, `GliderPRO/Sources/Transit.c:299`, `:336`, `:375`, `:414` |
| `kEscapedTitleMode` (17) | `GliderPRO/Sources/Interactions.c:182`, `:216`, `:294`, `:328`, `:522`, `:553`, `:612`, `:643`, `:1267`, `:1302`; `GliderPRO/Sources/Player.c:364`, `:492`, `:628`, `:735`, `:832`, `:1031`, `:1122` |
| `kSavingTitleMode` (1) | `GliderPRO/Sources/Input.c:67` |

`RefreshScoreboard()` itself (`GliderPRO/Sources/Scoreboard.c:53-68`):

```
1  doRollScore = true
2  RefreshRoomTitle(mode)
3  RefreshNumGliders()
4  RefreshPoints()
5  CopyBits(boardSrcMap -> mainWindow, boardSrcRect -> boardDestRect, srcCopy)
6  QuickBatteryRefresh(false)
7  QuickBandsRefresh(false)
8  QuickFoilRefresh(false)
```

Note there is **no** "quick" path for the room title: changing the title always
costs a full re-composite and a full-width blit.

### 5.6 The badge refreshers

All three have the identical shape — a two- or three-way choice of source rect
within `badgeSrcMap`, always blitted to the same dest rect, always
`CopyBits(..., srcCopy, nil)` straight to the window.

`QuickBatteryRefresh(Boolean flash)` (`GliderPRO/Sources/Scoreboard.c:338-364`):

| Condition | Source rect |
|---|---|
| `batteryTotal > 0 && !flash` | `badgesBadgesRects[kBatteryBadge]` |
| `batteryTotal < 0 && !flash` | `badgesBadgesRects[kHeliumBadge]` |
| otherwise | `badgesBlankRects[kBatteryBadge]` |

Note the third branch always uses the **battery** blank, never the helium blank —
harmless, because the two blanks are pixel-identical 16x17 strips and the dest
rects are identical too.

`QuickBandsRefresh(Boolean flash)` (`:368-386`) and
`QuickFoilRefresh(Boolean flash)` (`:390-408`) are the same with
`bandsTotal > 0` / `foilTotal > 0` and the corresponding badge index.

Call-site census (complete, `flash` argument in brackets):

| Function | Sites |
|---|---|
| `QuickBatteryRefresh` | `Scoreboard.c:65` (false, via `RefreshScoreboard`), `:105` (true), `:107` (true), `:117` (false), `:119` (false); `Input.c:142` (false, battery exhausted), `:167` (false, helium exhausted); `Interactions.c:866` (false, battery pickup), `:972` (false, helium pickup), `:1182` (false, microwaved) |
| `QuickBandsRefresh` | `Scoreboard.c:66` (false), `:112` (false), `:129` (true); `Input.c:258` (false, demo fired last band), `:357` (false, player fired last band); `Interactions.c:885` (false, bands pickup), `:1176` (false, microwaved) |
| `QuickFoilRefresh` | `Scoreboard.c:67` (false), `:100` (false), `:124` (true); `Modes.c:586` (false, `StartGliderFoilGoing`), `:611` (false, `StartGliderFoilLosing`) |

### 5.7 `HandleDynamicScoreboard()` — the per-frame update

`GliderPRO/Sources/Scoreboard.c:72-132`. Called from exactly two places, both in
the main loop: `GliderPRO/Sources/Play.c:470` (two-player branch) and `:495`
(one-player branch).

Local constants (`GliderPRO/Sources/Scoreboard.c:74-77`):

| Name | Value | Author's comment |
|---|---:|---|
| `kFoilLow` | 2 | `// 25%` |
| `kBatteryLow` | 17 | `// 25%` |
| `kHeliumLow` | -38 | `// 25%` |
| `kBandsLow` | 2 | `// 25%` |

The comments are all wrong except arguably the foil one: 25% of
`kFoilSupply` 8 is 2, 25% of `kBandsSupply` 8 is 2, but 25% of
`kBatterySupply` 50 is 12.5 (not 17) and 25% of `kHeliumSupply` 150 is -37.5
(not -38, though that rounds). And the tests are strict (`< kBandsLow`), so
"low" actually means **1 band left** and **1 foil hit left**.

Part 1 — the score roll:

```
1  if theScore > displayedScore:
2      if doRollScore:  displayedScore += kScoreRollAmount        // +13
3                       if displayedScore > theScore: displayedScore = theScore
4      else:            displayedScore = theScore
5      PlayPrioritySound(kScoreTikSound, kScoreTikPriority)
6      QuickScoreRefresh()
```

`doRollScore` is set `true` at `GliderPRO/Sources/Scoreboard.c:55` and is
**never set false anywhere in the tree**, so branch 4 is unreachable dead code
and the roll is always 13 points per frame. `kScoreTikSound` therefore ticks
once per frame for the whole duration of the roll: a 100-point pickup rolls for
`ceil(100/13)` = 8 frames, 5000 points (a star) for 385 frames = 12.8 s.
There is no de-duplication of the sound.

Part 2 — the low-supply flash, an 8-phase cycle keyed on the frame counter:

```
whosTurn = gameFrame & 0x00000007
```

| `whosTurn` | Action | Condition |
|---:|---|---|
| 0 | `QuickFoilRefresh(false)` — show foil | `0 < foilTotal < 2` |
| 1 | `QuickBatteryRefresh(true)` — hide | `0 < batteryTotal < 17` or `-38 < batteryTotal < 0` |
| 2 | `QuickBandsRefresh(false)` — show | `0 < bandsTotal < 2` |
| 3 | nothing | — |
| 4 | `QuickBatteryRefresh(false)` — show | same as phase 1 |
| 5 | `QuickFoilRefresh(true)` — hide | `0 < foilTotal < 2` |
| 6 | nothing | — |
| 7 | `QuickBandsRefresh(true)` — hide | `0 < bandsTotal < 2` |

So each badge blinks with period 8 frames = 0.267 s at 30 fps, but with
different duty cycles and phases:

| Badge | Shown at phase | Hidden at phase | Visible fraction |
|---|---:|---:|---|
| foil | 0 | 5 | phases 0-4 visible, 5-7 hidden → 5/8 |
| battery / helium | 4 | 1 | phases 4-7+0 visible, 1-3 hidden → 5/8 |
| bands | 2 | 7 | phases 2-6 visible, 7-1 hidden → 5/8 |

`gameFrame` is a `long` incremented once per game frame and zeroed by
`NewGame()` (`GliderPRO/Sources/Play.c:112`), so the phase is deterministic
from the start of a game.

### 5.8 `AdjustScoreboardHeight()` and the two vertical modes

`GliderPRO/Sources/Scoreboard.c:412-449`. Modes
(`GliderPRO/Headers/GliderDefines.h:513-515`): `kScoreboardHigh` = 0,
`kScoreboardLow` = 1, `kScoreboardTall` = 20.

```
 1  newMode = (numNeighbors == 9) ? kScoreboardHigh : kScoreboardLow
 2  if wasScoreboardMode == newMode: return                 // it is a toggle
 3  switch newMode:
 4    kScoreboardHigh:  offset = -localRoomsDest[kCentralRoom].top
 5                      justRoomsRect = workSrcRect                   // whole view
 6    kScoreboardLow:   offset =  localRoomsDest[kCentralRoom].top
 7                      justRoomsRect = workSrcRect
 8                      justRoomsRect.top    = localRoomsDest[kCentralRoom].top
 9                      justRoomsRect.bottom = localRoomsDest[kCentralRoom].bottom
10  QOffsetRect by (0, offset): boardDestRect, boardGQDestRect, boardPQDestRect,
11                              badgesDestRects[0..3]
12  wasScoreboardMode = newMode
```

`wasScoreboardMode` is initialised to `kScoreboardHigh` at
`GliderPRO/Sources/StructuresInit.c:70`. `localRoomsDest[kCentralRoom].top` is
`playOriginV = (RectTall(&thisMac.screen) - kTileHigh) / 2` with `kTileHigh` =
322 (`GliderPRO/Sources/InterfaceInit.c:204`, `:208-209`); `kCentralRoom` = 0
(`GliderPRO/Headers/GliderDefines.h:217`).

The only call site is `GliderPRO/Sources/Play.c:81`, the first statement of
`NewGame()` after its preamble.

**Consequence, and it is a big one.** The main window is created at screen
`(left, top+20)` with size `(W, screenH-20)` and its port is clipped to
`mainWindowRect` = `(0, 0, screenH-20, W)`
(`GliderPRO/Sources/MainWindow.c:222-232`); `WIND` 128 in the resource fork has
`boundsRect (0,0,384,512)` and `procID 2`, confirming the port origin is the
content top-left. In `kScoreboardHigh` mode `boardDestRect` is
`(-20, 0, 0, W)` — entirely above `y = 0` — so **the whole scoreboard, the
score, the glider count and all four badges are clipped away and never appear.**
The intent is clear (main-window-local y -20..0 is exactly the 20-px strip the
separate `menuWindow` occupies at the top of the screen, and
`UpdateMenuBarWindow()` at `GliderPRO/Sources/MainWindow.c:160-169` exists to
paint over the real menu bar), but a window's port cannot draw outside itself,
so nothing is drawn.

And `numNeighbors` defaults to **9** (`GliderPRO/Sources/Main.c:154`,
`GliderPRO/Sources/Settings.c:888`, `:1257`), forced to 1 only when
`thisMac.screen.right <= 512` (`GliderPRO/Sources/Main.c:191-192`). So on a
stock configuration with a screen wider than 512 px, **the shipped code draws no
scoreboard at all**. Setting neighbours to 1 or 3 in the preferences dialog
(`GliderPRO/Sources/Settings.c:1153`, `:1164`) moves `boardDestRect` down to
`(playOriginV - 20, 0, playOriginV, W)`, immediately above the central room, and
everything becomes visible.

In `kScoreboardLow` mode `justRoomsRect` is also narrowed to just the central
room's vertical band, which is what limits the dirty-rect blitter to the single
visible room (`AddRectToWorkRects()` clamps to `justRoomsRect`,
`GliderPRO/Sources/Render.c:65-80`).

### 5.9 `BlackenScoreboard()`

`GliderPRO/Sources/Scoreboard.c:453-456` is a one-liner: `UpdateMenuBarWindow();`
which paints `menuWindow`'s local rect with the current fore colour
(`GliderPRO/Sources/MainWindow.c:160-169`), or returns immediately if
`menuWindow == nil`. Its only call site is `GliderPRO/Sources/Play.c:255`, in
the tail of `NewGame()`. In other words: "cover the real menu bar with black".

### 5.10 `DisplayStarsRemaining()` — the star HUD

Not part of the scoreboard proper; it is a modal overlay shown for at least 2 s
every time a star is collected. `GliderPRO/Sources/Banner.c:205-236`:

```
 1  SetPortWindowPort(mainWindow)
 2  QSetRect(&bounds, 0, 0, 256, 64)
 3  CenterRectInRect(&bounds, &thisMac.screen)
 4  QOffsetRect(&bounds, -thisMac.screen.left, -thisMac.screen.top)   // -> window-local
 5  src = bounds ; InsetRect(&src, 64, 32)             // src is never used: dead
 6  TextFont(applFont) ; TextFace(bold) ; TextSize(12)
 7  NumToString((long)numStarsRemaining, theStr)
 8  QOffsetRect(&bounds, 0, -20)                       // compensate for the menu bar
 9  if numStarsRemaining < 2:
10      LoadScaledGraphic(kStarRemainingPICT, &bounds)         // PICT 1018
11  else:
12      LoadScaledGraphic(kStarsRemainingPICT, &bounds)        // PICT 1017
13      MoveTo(bounds.left + 102 - (StringWidth(theStr) / 2), bounds.top + 23)
14      ColorText(theStr, 4L)                                  // index 4 = #FFFF33
15  DelayTicks(60)                                             // 1 second
16  if WaitForInputEvent(30): RestoreEntireGameScreen()         // up to 0.5 s more
17  CopyRectWorkToMain(&bounds)
```

- PICT IDs: `kStarsRemainingPICT` = 1017, `kStarRemainingPICT` = 1018
  (`GliderPRO/Sources/Banner.c:20-21`).
- The singular artwork (1018) is used when **one** star is left and carries no
  number; the plural artwork (1017) gets the count drawn on it centred on
  x = `bounds.left + 102`, baseline `bounds.top + 23`, in bright yellow.
- Total blocking time: 60 ticks (1 s) unconditionally, then up to 30 more ticks
  (0.5 s) waiting for a key or click, so **60-90 ticks = 1.0-1.5 s**. The game
  loop is stalled for the whole time; `nextFrame` is not adjusted, so the frame
  clock catches up by running frames back-to-back afterwards.
- Called from exactly one place: `GliderPRO/Sources/Interactions.c:946`, the
  `else` of the "was that the last star" test.
- `src` at step 5 is computed and never read — with a 256x64 rect, insetting by
  (64, 32) yields a 128x0 degenerate rect anyway.
- `ColorText()` (`GliderPRO/Sources/ColorUtils.c:20-45`) does
  `Index2Color(index, &rgb); RGBForeColor(&rgb); DrawString(s);` and restores
  the previous colour.

## 6. Timers: there is no game timer

The player is never under time pressure. `theScore` has no time component, no
bonus is awarded for speed, and nothing counts down toward a loss. The only
wall-clock value stored anywhere is `houseType.savedGame.timeStamp` and
`scoresType.timeStamps[]`, both of which are cosmetic date stamps.

For completeness, here is every counter in the tree that could be mistaken for
a game timer, with its purpose:

| Counter / constant | Value | Where | What it actually times |
|---|---:|---|---|
| `kTicksPerFrame` | 2 | `GliderPRO/Headers/GliderDefines.h:533` | the frame budget: 2 ticks = 1/30 s |
| `nextFrame` | — | `GliderPRO/Sources/Render.c:40` | `TickCount() + kTicksPerFrame`, the next frame deadline |
| `gameFrame` | `long` | `GliderPRO/Sources/Play.c` | frames since `NewGame()`; drives the badge flash phase and object animation |
| `evenFrame` | `Boolean` | `GliderPRO/Sources/Render.c:59` | frame parity; halves the foil drain against moving objects |
| `countDown` | 16 down to 0 | `GliderPRO/Sources/GameOver.c:48`, `kNumCountDownFrames` = 16 at `:17` | frames between `FlagGameOver()` and the end sequence |
| `kFramesToBurn` | 60 | `GliderPRO/Sources/Modes.c:408` | how long a burning glider burns (2 s) |
| `kShredderCountdown` | -68 | `GliderPRO/Sources/Player.c:17` | confetti frames after being shredded (2.27 s) |
| `kLastFadeSequence` | 16 | `GliderPRO/Headers/GliderDefines.h:564` | frames in a fade in/out (0.53 s) |
| `kMaxFlyingPointsLoop` | 24 | `GliderPRO/Headers/GliderDefines.h:254` | animation cycles of a flying-point sprite (3 frames each, so 72 frames, 2.4 s) |
| `kIdleSplashTicks` | 7200 | `GliderPRO/Headers/GliderDefines.h:197` | 2 minutes of idle on the splash screen before the auto-demo |
| `kDemoLength` | 6702 | `GliderPRO/Headers/GliderDefines.h:625` | number of recorded demo input frames |
| `kNumCountDownFrames` | 16 | `GliderPRO/Sources/GameOver.c:17` | see `countDown` |
| `kStarFalls` | 8 | `GliderPRO/Sources/GameOver.c:138` (function-local to `DoGameOverStarAnimation`) | pixels per frame the win-screen stars fall |
| `DelayTicks(60)` | 60 | `GliderPRO/Sources/Banner.c:232` | the star-remaining overlay dwell |

There is likewise no per-room timer, no oxygen/air meter, and no penalty for
lingering. The *only* resources that deplete with time are the battery and the
helium (section 4.2), and they only deplete while their key is held.

---

## 7. High scores

### 7.1 Three candidate stores, only one of which is real

Glider PRO can conceivably keep high scores in three places. Only the first
is ever actually used by the shipped code path:

| # | Store | Identity | What it holds | Live? |
|---|---|---|---|---|
| 1 | **The house file itself**, data fork, byte offset **528**, length **292** | house file type `'gliH'`, creator `'ozm5'` | the whole `scoresType` (banner + 10 slots) | **YES** — written by `WriteHouse()` as part of one whole-handle `FSWrite` |
| 2 | `Preferences:G-PRO Scores ƒ:<houseName>` side-car | type `'gliS'`, creator `'ozm5'`, exactly 292 bytes | a copy of the same `scoresType` | **NO** — only reached when `houseIsReadOnly`, which is hard-wired `false` |
| 3 | `Preferences:Glider Prefs` | type `'gliP'`, creator `'ozm5'` | only `wasHighName` (`Str15`) and `wasHighBanner` (`Str31`) — the *remembered defaults for the entry dialogs*, not scores | YES, but no scores |

The folder name is spelled with Mac Roman byte `0xC4`, which renders as `ƒ`
(the classic Mac "folder" glyph), not a diamond:

```
GliderPRO/Sources/HighScores.c:654   FSMakeFSSpec(volRefNum, prefsDirID, "\pG-PRO Scores \xC4", &scoresSpec)
GliderPRO/Sources/HighScores.c:679   PasStringCopy("\pG-PRO Scores \xC4", nameString)
GliderPRO/Sources/HighScores.c:696   EqualString(theBlock.dirInfo.ioNamePtr, "\pG-PRO Scores \xC4", true, true)
```

Verified by reading the raw bytes of the CR-terminated source file:

```
$ python3 -c "d=open('Sources/HighScores.c','rb').read(); i=d.find(b'G-PRO Scores'); print(repr(d[i-4:i+18]))"
b' "\\pG-PRO Scores \xc4", &'
$ python3 -c "print(bytes([0xC4]).decode('mac-roman'))"
ƒ
```

**Why store 2 is dead.** All three call sites of `WriteScoresToDisk()` /
`ReadScoresFromDisk()` are guarded by `houseIsReadOnly`:

| Guard site | Code |
|---|---|
| `GliderPRO/Sources/HouseIO.c:329-338` | `if (houseIsReadOnly) { if (!WriteScoresToDisk()) …} else if (!WriteHouse(false)) return false;` |
| `GliderPRO/Sources/HouseIO.c:428-434` | `if (houseIsReadOnly) { houseUnlocked = false; if (ReadScoresFromDisk()) { } }` — note the empty `if` body |
| `GliderPRO/Sources/HouseIO.c:531-540` | `if (gameDirty) { if (houseIsReadOnly) { if (!WriteScoresToDisk()) YellowAlert(kYellowFailedWrite, 0); } else if (!WriteHouse(theMode == kEditMode)) … }` |

and `houseIsReadOnly` is assigned exactly once, at
`GliderPRO/Sources/HouseIO.c:187`, from `IsFileReadOnly()`, whose entire body
is:

```c
Boolean IsFileReadOnly (FSSpec *theSpec)
{
#pragma unused (theSpec)

	return false;
	/*
	... the real PBGetVInfo / PBHGetFInfo implementation, commented out ...
```
(`GliderPRO/Sources/HouseIO.c:659-664`; the `/*` at line 664 runs to the end
of the function.)

So `houseIsReadOnly` is permanently `false`, the side-car is never created or
read, and the only high-score persistence that happens is
`WriteHouse()` rewriting the entire house file. `houseIsReadOnly` still has one
visible effect: the splash-screen house-name colour at
`GliderPRO/Sources/MainWindow.c:76-79` (`ColorText(houseLoadedStr, 5L)` if
read-only, `28L` otherwise) — which is therefore always `28L`.

### 7.2 `scoresType`: field-by-field layout

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
(`GliderPRO/Headers/GliderStructs.h:107-114`; `kMaxScores` = **10** at
`GliderPRO/Headers/GliderDefines.h:249`.)

`scoresType` sits inside `houseType`:

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
(`GliderPRO/Headers/GliderStructs.h:182-198`.)

Absolute byte offsets in the house-file data fork, all big-endian, all
verified by parsing shipped BinHex'd houses (§7.3):

| Field | House offset | Size | Element size | Type |
|---|---:|---:|---:|---|
| `highScores` (whole struct) | **528** | 292 | — | `scoresType` |
| `highScores.banner` | **528** | 32 | — | `Str31` = `unsigned char[32]`, length byte + up to 31 chars |
| `highScores.names[0]` | **560** | 16 | 16 | `Str15` = `unsigned char[16]`, length byte + up to 15 chars |
| `highScores.names[i]` | `560 + 16*i` | — | 16 | — |
| `highScores.names[9]` | **704** | 16 | — | — |
| `highScores.scores[0]` | **720** | 40 | 4 | signed `long` (big-endian) |
| `highScores.scores[i]` | `720 + 4*i` | — | 4 | — |
| `highScores.timeStamps[0]` | **760** | 40 | 4 | `unsigned long`, seconds since the Mac epoch 1904-01-01 00:00:00 local |
| `highScores.timeStamps[i]` | `760 + 4*i` | — | 4 | — |
| `highScores.levels[0]` | **800** | 20 | 2 | `short` — **a count of rooms visited**, not a level index |
| `highScores.levels[i]` | `800 + 2*i` | — | 2 | — |
| end of `highScores` | 819 (last byte) | — | — | — |
| `savedGame` | 820 | 40 | — | `gameType`, §9 |
| `hasGame` | 860 | 1 | — | `Boolean` |
| `unusedBoolean` | 861 | 1 | — | `Boolean` |
| `firstRoom` | 862 | 2 | — | `short` |
| `nRooms` | 864 | 2 | — | `short` |
| `rooms[0]` | 866 | 348 | 348 | `roomType` |

There is **no padding** anywhere in `scoresType`: `Str31`/`Str15` are `char`
arrays (alignment 1), `long`/`unsigned long` land on offsets 720 and 760 which
are already 4-aligned, and `short` lands on 800. `#pragma options
align=mac68k` (declared at `GliderPRO/Headers/Externs.h:231` for `prefsInfo`
and in the equivalent position for `GliderStructs.h`) therefore adds nothing
here. The struct is 292 bytes on both 68k and PowerPC, and
`sizeof(scoresType)` is the literal value written to disk at
`GliderPRO/Sources/HighScores.c:771`.

**Verified hexdump.** Decoding `GliderPRO/Houses/Demo House.binhex` and
dumping the region:

```
scoresType.banner (Str31)  (house offset 528..559)
    528  13 54 68 65 20 52 65 74 75 72 6E 20 6F 66 20 4F  |.The Return of O|
    544  7A 6D 61 21 00 E8 72 50 01 26 33 08 01 26 32 DC  |zma!..rP.&3..&2.|
scoresType.names[0..1] (Str15 x2)  (house offset 560..591)
    560  04 4F 7A 6D 61 26 2B B0 DD DD DD DD DD DD DD DD  |.Ozma&+.........|
    576  0E 2D 2D 2D 2D 2D 2D 2D 2D 2D 2D 2D 2D 2D 2D 36  |.--------------6|
scoresType.scores[0..9] (long x10)  (house offset 720..759)
    720  00 00 1C E8 00 00 00 00 00 00 00 00 00 00 00 00  |................|
    736  00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00  |................|
    752  00 00 00 00 00 00 00 00 AC 2F F4 2F 00 00 00 00  |........././....|
scoresType.timeStamps[0..9] (unsigned long x10)  (house offset 760..799)
    760  AC 2F F4 2F 00 00 00 00 00 00 00 00 00 00 00 00  |././............|
    ...
scoresType.levels[0..9] (short x10)  (house offset 800..819)
    800  00 0D 00 00 00 00 00 00 00 00 00 00 00 00 00 00  |................|
    816  00 00 00 00 01 00 00 01 AA BB 15 14 00 0B 00 3B  |...............;|
```

Reading it out:

* `banner` = `0x13` = 19 chars, `"The Return of Ozma!"`.
* Bytes 548..559 (`00 E8 72 50 01 26 33 08 01 26 32 DC`) are **stale garbage**
  — the tail of the 32-byte array that `PasStringCopy` never touched (see the
  note on `PasStringCopy` below).
* `names[0]` = `0x04` = 4 chars, `"Ozma"`; bytes 565..575 are garbage.
* `names[1]` = `0x0E` = 14 chars, `"--------------"` (the "empty slot"
  placeholder written by `ZeroHighScores`); byte 591 (`0x36`) is garbage.
* `scores[0]` = `0x00001CE8` = **7400**; `scores[1..9]` = 0.
* `timeStamps[0]` = `0xAC2FF42F` = 2888823855. Converting with the Mac epoch
  offset: 2888823855 − 2082844800 = 805979055 Unix seconds =
  **1995-07-17 11:04:15 UTC** (the Mac stamp is local time, so the wall-clock
  date is 1995-07-17 in the machine's own time zone).
* `levels[0]` = `0x000D` = **13** rooms visited.

**`PasStringCopy` does not clear padding.** Its whole body is

```c
void PasStringCopy (StringPtr p1, StringPtr p2)
{
	register short		stringLength;

	stringLength = *p2++ = *p1++;
	while (--stringLength >= 0)
		*p2++ = *p1++;
}
```
(`GliderPRO/Sources/StringUtils.c:18-25`) — exactly `length + 1` bytes. Every
`Str31`/`Str15` in a shipped house therefore has uninitialised bytes after the
string, and those bytes were written to disk verbatim. A Go port must **not**
assume the padding is zero, and if it wants byte-identical round-trips it must
preserve the padding it read. (`SortHighScores()` makes this worse: it copies
into an uninitialised stack `scoresType tempScores` and then does a whole-struct
assignment, so the padding written back to the house is *stack garbage from the
moment of sorting* — see §7.6.)

### 7.3 Observed high-score tables in all 22 shipped houses

Produced by BinHex-decoding every `GliderPRO/Houses/*.binhex` and reading
offsets 528..819. Rows whose `scores[i]`, `timeStamps[i]` and `levels[i]` are
all zero and whose name is the dash placeholder are omitted. Dates are the Mac
stamp minus 2082844800 s rendered as UTC.

| House | `banner` | rows | slot | `names[i]` | `scores[i]` | `levels[i]` | `timeStamps[i]` | date |
|---|---|---:|---:|---|---:|---:|---:|---|
| Art Museum | `Your Message Here` | 2 | 1 | `Your Name` | 18500 | 22 | 2904337875 | 1996-01-13 |
| | | | 2 | `Ozma` | 12100 | 16 | 2891166272 | 1995-08-13 |
| CD Demo House | `The Return of Ozma!` | 3 | 1 | `Ozma` | 25300 | 28 | 2889804643 | 1995-07-28 |
| | | | 2 | `Ozma` | 8700 | 11 | 2889804183 | 1995-07-28 |
| | | | 3 | `Ozma` | 1600 | 5 | 2889803965 | 1995-07-28 |
| California or Bust! | `The Return of Ozma!` | 2 | 1 | `Ozma` | 7500 | 13 | 2893424028 | 1995-09-08 |
| | | | 2 | `Ozma` | 1600 | 7 | 2893414270 | 1995-09-08 |
| Castle o' the Air | `The Return of Ozma!` | 3 | 1 | `Ozma` | 7500 | 15 | 2888838341 | 1995-07-17 |
| | | | 2 | `Ozma` | 800 | 8 | 2888864063 | 1995-07-17 |
| | | | 3 | `Ozma` | 100 | 1 | 2889020553 | 1995-07-19 |
| Davis Station | `Johnner beat the Kimmer.` | 3 | 1 | `Johnner` | 17800 | 38 | 2928439337 | 1996-10-17 |
| | | | 2 | `Kimmer` | 17500 | 40 | 2917200379 | 1996-06-09 |
| | | | 3 | `Ozma` | 14800 | 28 | 2891166501 | 1995-08-13 |
| Demo House | `The Return of Ozma!` | 1 | 1 | `Ozma` | 7400 | 13 | 2888823855 | 1995-07-17 |
| Empty House | `Empty House` | 0 | — | all `--------------` | 0 | 0 | 0 | — |
| Fun House | `The Return of Ozma!` | **10** | 1 | `Ozma` | 1600 | 8 | 2889347518 | 1995-07-23 |
| | | | 2 | `Ozma` | 1300 | 5 | 2889346902 | 1995-07-23 |
| | | | 3 | `Ozma` | 1300 | 5 | 2889690906 | 1995-07-27 |
| | | | 4 | `Ozma` | 800 | 8 | 2889298569 | 1995-07-22 |
| | | | 5 | `Your Name` | 700 | 2 | 2889608957 | 1995-07-26 |
| | | | 6 | `Ozma` | 700 | 2 | 2889609137 | 1995-07-26 |
| | | | 7 | `Ozma` | 600 | 6 | 2889640799 | 1995-07-26 |
| | | | 8 | `Ozma` | 500 | 5 | 2889297286 | 1995-07-22 |
| | | | 9 | `Ozma` | 400 | 1 | 2889276127 | 1995-07-22 |
| | | | 10 | `Ozma` | 300 | 3 | 2889599727 | 1995-07-26 |
| Grand Prix | `Uhhh… Hi, Officer…` | 3 | 1 | `Paul` | 34400 | 88 | 2888927647 | 1995-07-18 |
| | | | 2 | `Ozma` | 10100 | 33 | 2894867240 | 1995-09-25 |
| | | | 3 | `Ozma` | 2900 | 13 | 2894671507 | 1995-09-23 |
| ImagineHouse PRO II | `Thankyuh… Thankyuh verra much…` | 1 | 1 | `Paul` | 47000 | 108 | 2887633088 | 1995-07-03 |
| In The Mirror | `The Return of Ozma!` | 1 | 1 | `Ozma` | 4100 | 11 | 2888995793 | 1995-07-19 |
| Land of Illusion | `Land of Illusion` | 0 | — | all `--------------` | 0 | 0 | 0 | — |
| Leviathan | `The Return of Ozma!` | 1 | 1 | `Ozma` | 8400 | 40 | 2888998283 | 1995-07-19 |
| Metropolis | `The Return of Ozma!` | 3 | 1 | `Ozma` | 2300 | 17 | 2889522948 | 1995-07-25 |
| | | | 2 | `Ozma` | 1400 | 8 | 2889125729 | 1995-07-20 |
| | | | 3 | `Ozma` | 400 | 4 | 2889522718 | 1995-07-25 |
| Nemo's Market | `The Return of Ozma!` | 1 | 1 | `Ozma` | 20700 | 23 | 2888604832 | 1995-07-14 |
| Rainbow's End | `The Return of Ozma!` | 1 | 1 | `Ozma` | 2000 | 8 | 2888605111 | 1995-07-14 |
| Sampler | `Your Message Here` | 2 | 1 | `Your Name` | 5200 | 2 | 3040919567 | 2000-05-11 |
| | | | 2 | `Your Name` | 5100 | 1 | 3040919419 | 2000-05-11 |
| Slumberland | `Your Message Here` | 3 | 1 | `Your Name` | 10800 | 25 | 3040890608 | 2000-05-11 |
| | | | 2 | `Your Name` | 7800 | 16 | 3040888772 | 2000-05-11 |
| | | | 3 | `Ozma` | 4900 | 16 | 2888914094 | 1995-07-18 |
| SpacePods | `The Return of Ozma!` | 1 | 1 | `Ozma` | 6700 | 43 | 2888605688 | 1995-07-14 |
| Teddy World | `The Return of Ozma!` | 1 | 1 | `Ozma` | 3400 | 16 | 2889106381 | 1995-07-20 |
| The Asylum Pro | `Spam Is Good For You.` | 1 | 1 | `Albert` | 4100 | 16 | 2885500804 | 1995-06-09 |
| Titanic | `The Return of Ozma!` | 2 | 1 | `Ozma` | 15300 | 33 | 2889099492 | 1995-07-20 |
| | | | 2 | `Your Name` | 100 | 1 | 2902607556 | 1995-12-23 |

Facts this evidence establishes:

1. **Every table is sorted strictly descending or with equal-value ties
   adjacent** (Fun House has 1300/1300 and 700/700). Consistent with
   `SortHighScores()`.
2. **Every score is a multiple of 100.** Consistent with §2: every point award
   in the game is a multiple of 100 (`kRoomVisitPoints` 100, star bonus,
   `points` fields of invisible bonuses).
3. **`levels[i]` is a room count, not a difficulty/level number.** Grand Prix's
   88 and ImagineHouse's 108 exceed any plausible "level", and
   `TestHighScore()` assigns it from `CountRoomsVisited()`
   (`GliderPRO/Sources/HighScores.c:412`, `GliderPRO/Sources/House.c:511-531`).
4. **`levels[i]` is not monotone in score.** Davis Station slot 1 (17800 pts,
   38 rooms) beats slot 2 (17500 pts, 40 rooms). The sort key is the score
   only.
5. **The three houses with a `Your Message Here` banner (`Art Museum`,
   `Sampler`, `Slumberland`) have never had a #1
   score entered through the dialogs** — `Your Message Here` is the
   *preferences default* (`GliderPRO/Sources/Main.c:134`), so those tables were
   built by a build in which the banner prompt was skipped or cancelled, or the
   banner was left at its default.
6. **`Empty House` and `Land of Illusion` have `banner == thisHouseName`,
   which is exactly what `ZeroHighScores()` writes**
   (`GliderPRO/Sources/HighScores.c:333`). Both have all ten slots zeroed with
   the 14-dash placeholder name. These two houses were shipped after an
   editor "Clear All".
7. **Fun House is the only shipped house with all ten slots filled**, which is
   consistent with it having 0 stars and therefore being unwinnable (§2.9):
   every play ends in `DoDiedGameOver`.

The banners use Mac Roman high bytes: `0xC9` for `…` in
`Uhhh… Hi, Officer…` and `Thankyuh… Thankyuh verra much…`. A Go port must
decode `Str31`/`Str15` as Mac Roman, not Latin-1 or UTF-8.

### 7.4 Invariants and the meaning of each field

| Field | Meaning | Range observed | Invariant |
|---|---|---|---|
| `banner` | the champion's message, drawn in a framed box on the high-score screen | 11..30 chars | set only by `GetHighScoreBanner()` (placing 0) or `ZeroHighScores()` (→ `thisHouseName`) |
| `names[i]` | player name | 4..14 chars | `--------------` (14 dashes) marks an empty slot; max 15 chars enforced by `PasStringCopyNum(tempStr, highName, 15)` |
| `scores[i]` | `theScore` at game over | 0..47000 | sorted descending; `0` means "empty"; `-1` occurs only transiently inside `SortHighScores()` |
| `timeStamps[i]` | `GetDateTime()` at the moment the score was entered | 0, or 2885500804..3040919567 | `0` when empty |
| `levels[i]` | `CountRoomsVisited()` at game over | 0..108 | `0` when empty; used only to choose singular/plural and to print a number |

`DrawHighScores()` treats `scores[i] > 0L` as "slot occupied"
(`GliderPRO/Sources/HighScores.c:176`). A score of exactly 0 is therefore
never displayed even if a name is present. Since `TestHighScore()` needs
`theScore > scores[i]`, a player who finishes with 0 points can never get on
the board (all empty slots hold 0 and `0 > 0` is false).

### 7.5 `TestHighScore()` — the qualification and insertion algorithm

Source: `GliderPRO/Sources/HighScores.c:374-426`.

```
TestHighScore() -> Boolean
 1. if (resumedSavedGame) return false               // hard ineligibility, see 7.13
 2. wasState = HGetState(thisHouse); HLock(thisHouse); thisHousePtr = *thisHouse
 3. lastHighScore = -1                              // module global, drives the white highlight
    placing       = -1
 4. for (i = 0; i < kMaxScores /*10*/; i++)
 5.     if (theScore > thisHousePtr->highScores.scores[i])
 6.         placing = i;  lastHighScore = i;  break  // first strictly-greater slot wins
 7. if (placing != -1)
 8.     FlushEvents(everyEvent, 0)
 9.     GetHighScoreName(placing + 1)                // modal dialog 1020, 1-based placing
10.     PasStringCopy(highName, thisHousePtr->highScores.names[kMaxScores-1])   // into slot 9
11.     if (placing == 0)
12.         GetHighScoreBanner()                     // modal dialog 1021
13.         PasStringCopy(highBanner, thisHousePtr->highScores.banner)
14.     thisHousePtr->highScores.scores[kMaxScores-1]     = theScore
15.     GetDateTime(&thisHousePtr->highScores.timeStamps[kMaxScores-1])
16.     thisHousePtr->highScores.levels[kMaxScores-1]     = CountRoomsVisited()
17.     SortHighScores()
18.     gameDirty = true                             // makes CloseHouse()/WriteHouse() save it
19. HSetState(thisHouse, wasState)
20. if (placing != -1) { DoHighScores(); return true } else return false
```

Notes a porter needs:

1. **Slot 9 is always the eviction slot.** The new entry is written into
   `[kMaxScores - 1]` = index 9 regardless of `placing`, overwriting whatever
   was in tenth place, and `SortHighScores()` then bubbles it up. The tenth
   entry is destroyed; there is no shifting.
2. **`lastHighScore` is set *before* the sort but is correct *after* it.** The
   pre-existing table is sorted descending, so `placing` equals the number of
   entries `>= theScore`. `SortHighScores()` is a first-wins-on-ties selection
   sort over indices 0..9, and the new entry lives at index 9 (the highest
   index), so every tie resolves in favour of the older entry and the new
   entry lands at exactly index `placing`. `lastHighScore` therefore names the
   right row for the white highlight in `DrawHighScores()`.
3. **`lastHighScore` is never reset except here and once at startup**
   (`lastHighScore = -1` at `GliderPRO/Sources/InterfaceInit.c:161`). If the
   player later opens the high-score screen from the Options menu
   (`GliderPRO/Sources/Menu.c:417-419`) the row from the *last* qualifying game
   is still highlighted white — even if a different house is loaded, since
   `lastHighScore` is a bare index.
4. **`theScore` is not clamped or validated.** It is whatever the roll left it
   at; see §2.10.
5. **The name defaults to the remembered `highName`**, which persists across
   launches via the prefs file, so consecutive games auto-fill the last name
   used.
6. **The banner prompt only appears for `placing == 0`.** If the player edits
   the name but Cancel is impossible (the dialogs have only an Okay button —
   see §7.11), so `GetHighScoreName` always returns a name; an empty edit field
   yields a zero-length `Str15`.
7. `gameDirty = true` is what eventually causes the house file to be rewritten
   (§7.13). If the app is force-quit before `CloseHouse()`, the score is lost.
8. `CountRoomsVisited()` walks all `nRooms` rooms and counts `rooms[r].visited`
   (`GliderPRO/Sources/House.c:511-531`). It counts the *in-memory* house, so
   it includes the current room (which `HandleRoomVisitation` marked on entry)
   but excludes rooms only glimpsed through a window.

### 7.6 `SortHighScores()` — selection sort with a padding hazard

Source: `GliderPRO/Sources/HighScores.c:281-318`.

```
SortHighScores()
 1. HLock(thisHouse); thisHousePtr = *thisHouse
 2. scoresType tempScores;                 // UNINITIALISED local, 292 bytes of stack
 3. for (h = 0; h < 10; h++)
 4.     greatest = -1L ; which = -1
 5.     for (i = 0; i < 10; i++)
 6.         if (thisHousePtr->highScores.scores[i] > greatest)
 7.             greatest = scores[i] ; which = i         // STRICTLY greater => first wins ties
 8.     if (which != -1)
 9.         PasStringCopy(scores.names[which], tempScores.names[h])
10.         tempScores.scores[h]     = scores[which]
11.         tempScores.timeStamps[h] = timeStamps[which]
12.         tempScores.levels[h]     = levels[which]
13.         scores[which] = -1L                          // mark consumed
14. PasStringCopy(thisHousePtr->highScores.banner, tempScores.banner)
15. thisHousePtr->highScores = tempScores                // whole-struct assignment
16. HSetState(...)
```

* **Complexity** 10x10 = 100 comparisons, always.
* **`which` can never stay -1**: on pass `h` there are `10 - h >= 1` slots
  still holding a value `>= 0`, and `>= 0 > -1`. Consumed slots hold `-1`, and
  `-1 > -1` is false, so they are never re-picked.
* **Stability**: because the comparison is strictly `>`, on a tie the *lowest
  index* is chosen. Older entries (lower indices) therefore keep their place,
  and the freshly inserted entry (index 9) sinks to the bottom of its tie
  group. Fun House's two 1300s and two 700s (§7.3) are consistent with this.
* **Zeros participate**: empty slots hold 0, so they sort to the end in index
  order and the dash names come along. `-1` never survives into the final
  table.
* **Padding hazard**: `tempScores` is never zeroed and `PasStringCopy` only
  writes `length + 1` bytes per string, so lines 9 and 14 leave up to 15 (name)
  and 31 (banner) bytes of *stack garbage* in `tempScores`, and line 15 copies
  all of it into the house handle, which `WriteHouse` then writes to disk. This
  is the origin of the garbage bytes visible in the §7.2 hexdump. A Go port
  producing files for the original binary should zero the padding (harmless);
  a port that wants bit-identical rewrites must carry the padding through
  verbatim.

### 7.7 Clearing the table: `ZeroHighScores()` and `ZeroAllButHighestScore()`

```
ZeroHighScores()                                   // HighScores.c:323-343
 1. HLock(thisHouse)
 2. PasStringCopy(thisHouseName, highScores.banner)    // banner := the house's name
 3. for (i = 0; i < 10; i++)
 4.     PasStringCopy("\p--------------", names[i])    // 14 '-' characters, 0x2D
 5.     scores[i]     = 0L
 6.     timeStamps[i] = 0L
 7.     levels[i]     = 0
```

```
ZeroAllButHighestScore()                           // HighScores.c:348-367
 1. HLock(thisHouse)
 2. for (i = 1; i < 10; i++)                          // NOTE: starts at 1
 3.     PasStringCopy("\p--------------", names[i])
 4.     scores[i]     = 0L
 5.     timeStamps[i] = 0L
 6.     levels[i]     = 0
    // banner and slot 0 are left alone
```

Call sites:

| Function | Called from | Context |
|---|---|---|
| `ZeroHighScores()` | `GliderPRO/Sources/House.c:133` | `CreateNewHouse()` — a brand-new house starts with an empty board whose banner is the house name |
| `ZeroHighScores()` | `GliderPRO/Sources/HouseInfo.c:329` | editor: House Info → Clear Scores → "Clear All" |
| `ZeroAllButHighestScore()` | `GliderPRO/Sources/HouseInfo.c:335` | editor: House Info → Clear Scores → "All But #1" |

The editor flow, verbatim:

```c
void HowToZeroScores (void)
{
	short		hitWhat;

//	CenterAlert(kZeroScoresAlert);
	hitWhat = Alert(kZeroScoresAlert, nil);

	switch (hitWhat)
	{
		case 2:		// zero all
		ZeroHighScores();
		fileDirty = true;
		UpdateMenus(false);
		break;

		case 3:		// zero all but highest
		ZeroAllButHighestScore();
		fileDirty = true;
		UpdateMenus(false);
		break;
	}
}
```
(`GliderPRO/Sources/HouseInfo.c:319-340`; `kZeroScoresAlert` = **1032** at
`GliderPRO/Sources/HouseInfo.c:24`; reached from
`GliderPRO/Sources/HouseInfo.c:292-293` when the House Info dialog's
`kClearScoresButton` is hit.)

`ALRT 1032` and its `DITL 1032`, dumped from `GliderPRO/Glider PRO.r`:

```
ALRT 1032 'Clear Scores'  bounds (T=40 L=40 B=129 R=320) = 280x89, itemsID 1032, stages 0x4444
DITL 1032 (High Scores): 5 items, 192 bytes
  item  1  rect(T=61 L=214 B=81 R=272) 58x20   Button      enabled  'Cancel'
  item  2  rect(T=61 L=134 B=81 R=206) 72x20   Button      enabled  'Clear All'
  item  3  rect(T=61 L=54  B=81 R=126) 72x20   Button      enabled  'All But #1'
  item  4  rect(T=8  L=8   B=56 R=232) 224x48  StaticText  disabled 'Do what?  You can clear all but the highest score, all the scores, or cancel this operation.'
  item  5  rect(T=8  L=240 B=40 R=272) 32x32   Icon        disabled resID=910
```

Item 1 is `Cancel` (the default button, hence no action), item 2 is
`Clear All` → `case 2`, item 3 is `All But #1` → `case 3`. The mapping in the
C matches the DITL exactly. Note the reversed left-to-right order: the buttons
read `All But #1 | Clear All | Cancel` on screen but are numbered 3, 2, 1.

`fileDirty = true` (not `gameDirty`) means clearing scores is treated as an
*editor* change: on close the user gets the standard "Save changes?" query
(`GliderPRO/Sources/HouseIO.c:541-547`).

### 7.8 `DoHighScores()` — and the fact that nothing reaches the screen

```c
void DoHighScores (void)
{
	Rect		tempRect;

	SpinCursor(3);
	SetPort((GrafPtr)workSrcMap);
	PaintRect(&workSrcRect);
	QSetRect(&tempRect, 0, 0, 640, 480);
	QOffsetRect(&tempRect, splashOriginH, splashOriginV);
	LoadScaledGraphic(kStarPictID, &tempRect);
//	if (quickerTransitions)
//		DissBitsChunky(&workSrcRect);
//	else
//		DissBits(&workSrcRect);
	SpinCursor(3);
	SetPort((GrafPtr)workSrcMap);
	DrawHighScores();
	SpinCursor(3);
//	if (quickerTransitions)
//		DissBitsChunky(&workSrcRect);
//	else
//		DissBits(&workSrcRect);
	InitCursor();
	DelayTicks(60);
	WaitForInputEvent(30);

	RedrawSplashScreen();
}
```
(`GliderPRO/Sources/HighScores.c:58-85`.)

Step by step:

1. `SetPort((GrafPtr)workSrcMap)` — the *offscreen work GWorld* becomes the
   current port. Everything after this draws offscreen.
2. `PaintRect(&workSrcRect)` — fills the whole work map with the current
   foreground colour (black at this point).
3. `QSetRect(&tempRect, 0, 0, 640, 480)` then offset by
   `(splashOriginH, splashOriginV)` — a 640x480 destination centred on the
   screen. `splashOriginH = max(0, (screenW - 640)/2)`,
   `splashOriginV = max(0, (screenH - 480)/2)`
   (`GliderPRO/Sources/MainWindow.c:238-243`).
4. `LoadScaledGraphic(kStarPictID /* 1995 */, &tempRect)` — `GetPicture(1995)`
   then `DrawPicture(pic, &tempRect)` then `ReleaseResource`
   (`GliderPRO/Sources/Utilities.c:340-349`). **PICT 1995's `picFrame` is
   640x460**, so `DrawPicture` *stretches* it vertically by 480/460 = 1.0435.
   Verified: `PICT 1995 picFrame(t=0,l=0,b=460,r=640) 640x460, 20746 bytes`.
5. `DrawHighScores()` — all the text and the plaque, §7.9.
6. `DelayTicks(60)` = `Delay(60, &whoCares)` = a 1-second unconditional pause
   (`GliderPRO/Sources/Utilities.c:731-736`).
7. `WaitForInputEvent(30)` — spins for up to 30 s (1800 ticks) waiting for a
   `mouseDown`, `keyDown`, or any of Command/Option/Shift/Control being held
   (`GetKeys` + `BitTst`), then returns
   (`GliderPRO/Sources/Utilities.c:439-479`).
8. `RedrawSplashScreen()`.

**The high-score screen is drawn but never displayed in this source release.**
The two `DissBits(&workSrcRect)` calls that would have transferred `workSrcMap`
to the window are commented out (`GliderPRO/Sources/HighScores.c:68-71` and
`:76-79`), and nothing else in `DoHighScores()` blits. Worse,
`RedrawSplashScreen()` finishes with a copy in the **wrong direction**:

```c
void RedrawSplashScreen (void)
{
	Rect		tempRect;

	SetPort((GrafPtr)workSrcMap);
	PaintRect(&workSrcRect);
	QSetRect(&tempRect, 0, 0, 640, 460);
	QOffsetRect(&tempRect, splashOriginH, splashOriginV);
	LoadScaledGraphic(kSplash8BitPICT, &tempRect);
	DrawOnSplash();
	SetPortWindowPort(mainWindow);
//	if (quickerTransitions)
//		DissBitsChunky(&workSrcRect);
//	else
//		DissBits(&workSrcRect);
	CopyRectMainToWork(&workSrcRect);
}
```
(`GliderPRO/Sources/MainWindow.c:98-114`.) `CopyRectMainToWork` copies
**main window → work map** (`GliderPRO/Sources/Render.c:722-727`), i.e. it
throws away the splash it just composed and replaces it with whatever is
already on screen. The intended call was clearly `CopyRectWorkToMain`
(`GliderPRO/Sources/Render.c:713-718`).

Consequence for a faithful port: if you reproduce this literally, the player
sees the previous screen frozen for 1 s + up to 30 s and never sees the high
scores. Every other `DissBits` call site in the tree is commented out the same
way (`GliderPRO/Sources/Banner.c:179,181,195,197`,
`GliderPRO/Sources/HighScores.c:68,71,77,79`,
`GliderPRO/Sources/MainWindow.c:110,112`,
`GliderPRO/Sources/Play.c:157,159,166,168,817,819`), so this is a
transition-effects refactor that was left half-finished, not a deliberate
design. **A port should treat the intent as `CopyRectWorkToMain(&workSrcRect)`
after `DrawHighScores()` and again inside `RedrawSplashScreen()`, and say so.**

### 7.9 `DrawHighScores()` — pixel-exact layout

Source: `GliderPRO/Sources/HighScores.c:94-276`. Layout constants
(`:90-92`):

| Constant | Value | Meaning |
|---|---:|---|
| `kScoreSpacing` | **18** | vertical pitch between rows, px |
| `kScoreWide` | **352** | nominal width of the whole score block, px |
| `kKimsLifted` | **4** | extra lift applied to the banner and to row 0 |
| `kHighScoresPictID` | **1994** | the 332x30 colour plaque |
| `kHighScoresMaskID` | **1998** | its 1-bit mask |
| `kMaxScores` | **10** | rows |

Two derived anchors:

```
scoreLeft = ((thisMac.screen.right - thisMac.screen.left) - kScoreWide) / 2      // HighScores.c:106
dropIt    = 129 + splashOriginV                                                  // HighScores.c:107
```

`scoreLeft` is computed from the **screen width**, but everything is drawn into
`workSrcMap`, whose rect is `houseRect` zeroed at the corner — i.e. the same
width as the screen. So `scoreLeft` is a work-map x coordinate. `dropIt` is a
work-map y coordinate: 129 plus the splash vertical centring offset. Note the
asymmetry — the horizontal anchor recentres on the screen width directly while
the vertical anchor uses `splashOriginV`, which is 0 for any screen shorter
than 480 px and `(screenH - 480)/2` otherwise. C integer division truncates
toward zero; all the operands here are non-negative so `/` and `>>` agree, but
a Go port should still use integer division on the same signs.

#### 7.9.1 Drawing order and every coordinate

| Step | Source lines | What | Destination geometry |
|---|---|---|---|
| 1 | `:111-114` | create a 332x30 8-bit GWorld `tempMap`, `LoadGraphic(1994)` into it | `QSetRect(&tempRect, 0, 0, 332, 30)` |
| 2 | `:116-118` | create a 332x30 **1-bit** GWorld `tempMask`, `LoadGraphic(1998)` into it | same rect, depth 1 |
| 3 | `:120-126` | `CopyMask(tempMap, tempMask, workSrcMap, tempRect, tempRect, tempRect2)` | `tempRect2` = `tempRect` offset by `(scoreLeft + (352-332)/2, dropIt - 60)` = `(scoreLeft + 10, dropIt - 60)`; occupies rows `dropIt-60 .. dropIt-31`, columns `scoreLeft+10 .. scoreLeft+341` |
| 4 | `:128-129` | `DisposeGWorld` both | — |
| 5 | `:133-135` | `TextFont(applFont); TextFace(bold); TextSize(14)` | — |
| 6 | `:137-139` | `tempStr = "\p\xA5 " + thisHouseName + "\p \xA5"` — a **bullet** (`•`, Mac Roman 0xA5) on each side | — |
| 7 | `:140-142` | black shadow of the title | baseline `MoveTo(scoreLeft + (352 - StringWidth(tempStr))/2 - 1, dropIt - 66)` |
| 8 | `:143-145` | cyan face of the title | baseline `MoveTo(scoreLeft + (352 - StringWidth(tempStr))/2, dropIt - 65)` |
| 9 | `:146` | `ForeColor(blackColor)` | — |
| 10 | `:148-150` | `TextFont(applFont); TextFace(bold); TextSize(12)` — everything below is 12 pt bold | — |
| 11 | `:152-154` | `HLock(thisHouse)` for the rest of the function | — |
| 12 | `:156-157` | `tempStr = highScores.banner`; `bannerWidth = StringWidth(tempStr)` | — |
| 13 | `:158-160` | black shadow of the banner | `MoveTo(scoreLeft + (352 - bannerWidth)/2, dropIt - 4)` |
| 14 | `:161-163` | **yellow** face of the banner | `MoveTo(scoreLeft + (352 - bannerWidth)/2, dropIt - 5)` |
| 15 | `:165-169` | black frame around the banner | `QSetRect(&tempRect, 0, 0, bannerWidth + 8, 18)` offset by `(scoreLeft - 3 + (352 - bannerWidth)/2, dropIt - 17)`, i.e. `(T=dropIt-17, L=scoreLeft-3+(352-bw)/2, B=dropIt+1, R=L+bw+8)`; `FrameRect` |
| 16 | `:170-172` | yellow frame, offset `(-1,-1)` | same rect moved to `(T=dropIt-18, L=L-1, B=dropIt, R=R-1)`; `FrameRect` |
| 17 | `:174-264` | the per-row loop, see 7.9.2 | — |
| 18 | `:266-272` | `ForeColor(blueColor)`, `applFont` bold **9 pt**, footer text `GetLocalizedString(8)` | `MoveTo(scoreLeft + 80, dropIt - 1 + 10*18)` = `(scoreLeft + 80, dropIt + 179)` |
| 19 | `:274-275` | `ForeColor(blackColor)`; `HSetState` | — |

The title's shadow is **up and to the left** of the face (`-1, -1` relative to
the face at `(x, dropIt-65)`), whereas the banner's yellow face is **one pixel
above** its black shadow with no horizontal offset, and the score rows put the
coloured face one pixel *above* the black shadow as well. This is inconsistent
with the scoreboard (§5.5), where the shadow is down-and-right. Reproduce it
literally.

#### 7.9.2 The per-row loop

For each `i` in `0..9`, only if `highScores.scores[i] > 0L`
(`GliderPRO/Sources/HighScores.c:176`), five columns are drawn. Each column is
drawn twice: a black shadow at `(x_shadow, y_shadow)` then a coloured face at
`(x_shadow - 1, y_shadow - 1)`.

Row baselines:

```
if (i == 0)  y_shadow = dropIt - kScoreSpacing - kKimsLifted   =  dropIt - 22
else         y_shadow = dropIt + i * kScoreSpacing             =  dropIt + 18*i
y_face = y_shadow - 1
```

So **row 0 is drawn *above* the banner box** and rows 1..9 below it. The order
on screen from top to bottom is: plaque, row 0 (rank 1), banner box, rows 1..9,
footer.

| Column | Content | `x_shadow` | `x_face` | shadow colour | face colour |
|---|---|---|---|---|---|
| 1 | `NumToString(i + 1)` — the placing, 1..10 | `scoreLeft + 1` | `scoreLeft + 0` | `blackColor` | `whiteColor` if `i == lastHighScore` else `cyanColor` |
| 2 | `names[i]` | `scoreLeft + 31` | `scoreLeft + 30` | `blackColor` | `whiteColor` if `i == lastHighScore` else `yellowColor` |
| 3 | `NumToString(levels[i])` — rooms visited | `scoreLeft + 161` | `scoreLeft + 160` | `blackColor` | `whiteColor` if `i == lastHighScore` else `yellowColor` |
| 4 | `GetLocalizedString(6)` = `room` if `levels[i] == 1`, else `GetLocalizedString(7)` = `rooms` | `scoreLeft + 193` | `scoreLeft + 192` | `blackColor` | **always `cyanColor`** — no `lastHighScore` test |
| 5 | `NumToString(scores[i])` | `scoreLeft + 291` | `scoreLeft + 290` | `blackColor` | `whiteColor` if `i == lastHighScore` else `yellowColor` |

Column 4 is the anomaly: `GliderPRO/Sources/HighScores.c:240` unconditionally
does `ForeColor(cyanColor)` where the other four columns branch on
`i == lastHighScore`. So the highlighted row reads
*white white white **cyan** white*. This is almost certainly an oversight, but
it is observable and a faithful port should reproduce it.

`SpinCursor(1)` is called once per drawn row (`:178`).

The localized strings come from `STR# 150` via
`GetIndString(theString, 150, index)`
(`GliderPRO/Sources/StringUtils.c:321-327`). Verified from
`GliderPRO/Glider PRO.r`: `STR# 150` has **51** strings; index 6 = `room`,
index 7 = `rooms`, index 8 = `Click Mouse or Hit a Key to Exit`.

#### 7.9.3 Worked example: 640x480 screen

`splashOriginH = 0`, `splashOriginV = 0`, `scoreLeft = (640 - 352)/2 = 144`,
`dropIt = 129`.

| Element | Work-map geometry |
|---|---|
| backdrop PICT 1995 (640x460 art) | scaled into `(T=0, L=0, B=480, R=640)` |
| plaque PICT 1994 + mask 1998 | `(T=69, L=154, B=99, R=486)` |
| title `• Demo House •` shadow baseline | `y = 63`, `x = 144 + (352 - w)/2 - 1` |
| title face baseline (cyan) | `y = 64`, `x = 144 + (352 - w)/2` |
| row 0 shadow baseline | `y = 107` |
| row 0 face baseline | `y = 106` |
| banner box, black | `(T=112, L=141 + (352-bw)/2, B=130, R=L + bw + 8)` |
| banner box, yellow | `(T=111, L=140 + (352-bw)/2, B=129, R=L + bw + 8)` |
| banner shadow baseline | `y = 125` |
| banner face baseline (yellow) | `y = 124` |
| rows 1..9 shadow baselines | `y = 147, 165, 183, 201, 219, 237, 255, 273, 291` |
| rows 1..9 face baselines | `y = 146, 164, 182, 200, 218, 236, 254, 272, 290` |
| footer baseline (blue, 9 pt) | `y = 308`, `x = 224` |
| column x (shadow / face) | 145/144, 175/174, 305/304, 337/336, 435/434 |

#### 7.9.4 Worked example: 1024x768 screen

`splashOriginH = (1024 - 640)/2 = 192`, `splashOriginV = (768 - 480)/2 = 144`,
`scoreLeft = (1024 - 352)/2 = 336`, `dropIt = 129 + 144 = 273`.

| Element | Work-map geometry |
|---|---|
| backdrop PICT 1995 | `(T=144, L=192, B=624, R=832)` |
| plaque PICT 1994 | `(T=213, L=346, B=243, R=678)` |
| title face baseline | `y = 208` |
| row 0 face baseline | `y = 250` |
| banner box, yellow | `(T=255, L=332 + (352-bw)/2, B=273, R=L + bw + 8)` |
| banner face baseline | `y = 268` |
| rows 1..9 face baselines | `y = 290, 308, 326, 344, 362, 380, 398, 416, 434` |
| footer baseline | `y = 452`, `x = 416` |
| column x (face) | 336, 366, 496, 528, 626 |

#### 7.9.5 Worked example: 512x384 screen

`splashOriginH = 0` (clamped from -64), `splashOriginV = 0` (clamped from
-48), `scoreLeft = (512 - 352)/2 = 80`, `dropIt = 129`.

The 640x480 backdrop rect is drawn at `(0,0,480,640)` into a work map that is
only 512x364 wide/high, so the backdrop is clipped on the right and bottom.
The score block still fits horizontally (80 + 352 = 432 <= 512) but the footer
baseline at `y = 308` and row 9 at `y = 291` are inside the 364-tall map, so
the list survives. This screen size is the smallest Glider PRO supports
(`GliderPRO/Sources/Environ.c` rejects smaller) and is the one that also
forces `numNeighbors = 1` (`GliderPRO/Sources/Main.c:191-192`).

#### 7.9.6 QuickDraw colour constants used

`DrawHighScores()` uses `ForeColor()` with the eight classic 1-bit QuickDraw
colour constants, **not** palette indices:

| Constant | Value | Nominal colour | Used for |
|---|---:|---|---|
| `blackColor` | 33 | black | every shadow pass; restored at the end |
| `whiteColor` | 30 | white | the `lastHighScore` row (columns 1,2,3,5) |
| `yellowColor` | 69 | yellow | name, rooms count, score, banner |
| `cyanColor` | 273 | cyan | placing number, the word `room`/`rooms`, the house title |
| `blueColor` | 409 | blue | the footer |

On an 8-bit indexed device `ForeColor` resolves these through the current
`GDevice`'s colour table to the nearest entry, so the exact RGB depends on the
loaded `clut` (see §5.3 for the palette Glider PRO installs). A Go port working
in RGB should map them to the classic QuickDraw values: black `#000000`, white
`#FFFFFF`, yellow `#FCF305`, cyan `#02ABEA`, blue `#0000D4`. Compare §5.5,
which uses `ColorText(str, index)` with *palette indices* instead — the two
subsystems use two different colour APIs.

### 7.10 Artwork resources for the high-score screen

All dimensions read out of `GliderPRO/Glider PRO.r` by parsing the PICT
headers (`picFrame` at offset 2..9, version word at offset 10):

| Res | ID | `picFrame` | Size | Bytes | Version | Notes |
|---|---:|---|---|---:|---|---|
| `PICT` | 1995 (`kStarPictID`) | `(0,0,460,640)` | 640x460 | 20746 | 0x0011 (v2) | starfield backdrop; **stretched to 640x480** by `DoHighScores()` |
| `PICT` | 1994 (`kHighScoresPictID`) | `(0,0,30,332)` | 332 wide x 30 tall | 4972 | 0x0011 (v2) | the colour plaque; decodes as 8-bit, PackBitsRect op `0x0098` |
| `PICT` | 1998 (`kHighScoresMaskID`) | `(0,0,30,332)` | 332x30 | 1049 | **0x1101 → a version-1 PICT** | 1-bit mask for the plaque; my decoder rejects it ("unhandled opcode 0xA000 at 12") because v1 PICTs use **1-byte** opcodes |
| `PICT` | 1000 (`kSplash8BitPICT`) | — | 640x460 | — | — | the splash, restored afterwards |

Verified transcript:

```
PICT 1994 picFrame(t=0,l=0,b=30,r=332) 332x30  sizeField=4972 actual=4972 ver=0x0011
PICT 1998 picFrame(t=0,l=0,b=30,r=332) 332x30  sizeField=1049 actual=1049 ver=0x1101
PICT 1995 picFrame(t=0,l=0,b=460,r=640) 640x460 sizeField=20746 actual=20746
1994  W 332  H 30  pixelSize 8  op 0x98
1998  ERR unhandled opcode 0xA000 at 12
```

PICT 1998 being a version-1 PICT is worth flagging: a Go PICT decoder must
handle **both** the v1 form (`version` word `0x1101`, single-byte opcodes) and
the v2 form (`version` word `0x0011` followed by `0x02FF` and the `0x0C00`
header opcode, two-byte opcodes). The same is true of PICT 5006 (§2.8). All
other PICTs in this subsystem are v2.

Because the plaque is drawn with `CopyMask(tempMap, tempMask, workSrcMap, …)`,
the 1-bit mask selects which of the 332x30 colour pixels reach the work map.
`CreateOffScreenGWorld(&tempMask, &tempRect, 1)` makes a 1-bit GWorld
(`GliderPRO/Sources/HighScores.c:116`) and `LoadGraphic(kHighScoresMaskID)`
draws the v1 PICT into it, so the mask's black pixels become 1 (opaque).

### 7.11 The two entry dialogs

#### 7.11.1 `DLOG`/`DITL` 1020 — "High Name"

```
DLOG 1020 'High Name'  bounds (T=0 L=0 B=109 R=316) = 316x109
          procID 1, visible 1, goAway 1, refCon 0, itemsID 1020, title '' (21 bytes)
DITL 1020 (High Name): 6 items, 200 bytes
  item  1  rect(T=81 L=250 B=101 R=308) 58x20   Button      enabled  'Okay'
  item  2  rect(T=52 L=157 B=68  R=297) 140x16  EditText    enabled  'Your Name'
  item  3  rect(T=8  L=8   B=41  R=308) 300x33  StaticText  disabled 'Your score of ^0 is #^1 on the top ten high scores for ^2.'
  item  4  rect(T=49 L=16  B=81  R=132) 116x32  StaticText  disabled 'Enter your name:\r(15 letters max.)'
  item  5  rect(T=81 L=154 B=97  R=173) 19x16   StaticText  disabled ''
  item  6  rect(T=81 L=175 B=97  R=224) 49x16   StaticText  disabled 'letters'
```

`^0`, `^1`, `^2` are filled by `ParamText`:

```c
void GetHighScoreName (short place)
{
	...
	InitCursor();
	NumToString(theScore, scoreStr);
	NumToString((long)place, placeStr);
	ParamText(scoreStr, placeStr, thisHouseName, "\p");
	PlayPrioritySound(kEnergizeSound, kEnergizePriority);
	BringUpDialog(&theDial, kHighNameDialogID);
	FlushEvents(everyEvent, 0);
	SetDialogString(theDial, kHighNameItem, highName);
	SelectDialogItemText(theDial, kHighNameItem, 0, 1024);
	leaving = false;

	while (!leaving)
	{
		ModalDialog(nameFilterUPP, &item);

		if (item == kOkayButton)
		{
			GetDialogString(theDial, kHighNameItem, tempStr);
			PasStringCopyNum(tempStr, highName, 15);
			leaving = true;
		}
	}

	DisposeDialog(theDial);
	DisposeModalFilterUPP(nameFilterUPP);
}
```
(`GliderPRO/Sources/HighScores.c:498-533`.) So `^0` = `theScore`, `^1` = the
1-based placing, `^2` = `thisHouseName`, `^3` = empty. There is **no Cancel**;
the loop only exits on item 1.

`PasStringCopyNum(tempStr, highName, 15)` truncates to 15 characters
(`GliderPRO/Sources/StringUtils.c:89-103`) and writes only `1 + n` bytes, so
`highName`'s padding is again left stale.

`BringUpDialog` (`GliderPRO/Sources/DialogUtils.c:24-33`):

```c
void BringUpDialog (DialogPtr *theDialog, short dialogID)
{
//	CenterDialog(dialogID);
	*theDialog = GetNewDialog(dialogID, nil, kPutInFront);
	if (*theDialog == nil)
		RedAlert(kErrDialogDidntLoad);
	SetPort((GrafPtr)*theDialog);
	ShowWindow(GetDialogWindow(*theDialog));
	DrawDefaultButton(*theDialog);
}
```

`CenterDialog` is commented out, so the dialog appears at the position stored
in its `DLOG` resource. **`DLOG 1020`'s bounds are `(0,0,109,316)`** — the
top-left corner of the screen, partially under the menu bar. `DLOG 1021`'s
bounds are `(40,40,162,356)`, a more sensible position. So in this build the
name prompt appears jammed into the screen corner and the banner prompt does
not. Faithful ports should reproduce both positions and note the discrepancy.

`DrawDefaultButton` draws a 3-px-wide rounded outline 4 px outside item 1
(`InsetRect(&itemRect, -4, -4); PenSize(3, 3); FrameRoundRect(&itemRect, 16, 16)`),
`GliderPRO/Sources/DialogUtils.c:352-363`.

#### 7.11.2 `DLOG`/`DITL` 1021 — "Banner"

```
DLOG 1021 'Banner'  bounds (T=40 L=40 B=162 R=356) = 316x122
          procID 1, visible 1, goAway 1, refCon 0, itemsID 1021
DITL 1021 (High Banner): 5 items, 178 bytes
  item  1  rect(T=94 L=250 B=114 R=308) 58x20   Button      enabled  'Okay'
  item  2  rect(T=67 L=11  B=83  R=305) 294x16  EditText    enabled  ''
  item  3  rect(T=8  L=11  B=56  R=305) 294x48  StaticText  disabled 'Getting #1 on the high scores entitles you to change the high score banner.\r(31 letters max.)'
  item  4  rect(T=94 L=29  B=110 R=78)  49x16   StaticText  disabled 'letters'
  item  5  rect(T=94 L=8   B=110 R=27)  19x16   StaticText  disabled ''
```

`GetHighScoreBanner()` (`GliderPRO/Sources/HighScores.c:608-638`) is the same
shape, with `PasStringCopyNum(tempStr, highBanner, 31)` and no `ParamText`.
Note that items 4 and 5 are swapped relative to DITL 1020 (here `letters` is
item 4 and the counter is item 5), but the code uses
`kBannerScoreNCharsItem = 5` (`GliderPRO/Sources/HighScores.c:30`), which is
the empty one — correct.

#### 7.11.3 The live character counters and key filters

Both dialogs show a live count of characters typed. The counter item is
updated in two places:

```
UpdateNameDialog(theDialog)                        // HighScores.c:431-440
 1. DrawDialog(theDialog)
 2. DrawDefaultButton(theDialog)
 3. nChars = GetDialogStringLen(theDialog, kHighNameItem /*2*/)
 4. SetDialogNumToStr(theDialog, kNameNCharsItem /*5*/, (long)nChars)
```

```
NameFilter(dial, event, item) -> Boolean           // HighScores.c:445-493
 1. if (keyStroke)                                 // module-global Boolean, HighScores.c:48
 2.     nChars = GetDialogStringLen(dial, 2)
 3.     SetDialogNumToStr(dial, 5, nChars)
 4.     keyStroke = false
 5. switch (event->what)
 6.   case keyDown:
 7.     keyStroke = true
 8.     switch (event->message & charCodeMask)
 9.       case kReturnKeyASCII: case kEnterKeyASCII:
10.           PlayPrioritySound(kCarriageSound, kCarriagePriority)
11.           FlashDialogButton(dial, kOkayButton)   // hilite, Delay(8), unhilite
12.           *item = kOkayButton; return true
13.       case kTabKeyASCII:
14.           SelectDialogItemText(dial, 2, 0, 1024); return false
15.       default:
16.           PlayPrioritySound(kTypingSound, kTypingPriority); return false
17.   case updateEvt:
18.     BeginUpdate(GetDialogWindow(dial))
19.     UpdateNameDialog(dial)
20.     EndUpdate(GetDialogWindow(dial))
21.     event->what = nullEvent
22.     return false
23.   default: return false
```

`BannerFilter` (`GliderPRO/Sources/HighScores.c:552-601`) is identical with
item 2 / item 5 of dialog 1021 and `UpdateBannerDialog`.

The counter lags by one keystroke: `keyStroke` is set on the `keyDown` *after*
the character has been inserted by `ModalDialog`'s default handling, and the
counter is only refreshed on the *next* filter entry. Since `ModalDialog` calls
the filter on null events too, the visible lag is one event-loop iteration.

`FlashDialogButton` hilites the control, `Delay(8, …)` (8 ticks = 0.133 s),
unhilites (`GliderPRO/Sources/DialogUtils.c:335-346`).

Sounds used: `kCarriageSound` (53) at `kCarriagePriority` (809) on
Return/Enter, `kTypingSound` (52) at `kTypingPriority` (808) on every other
key, `kEnergizeSound` (6) at `kEnergizePriority` (803) when the dialog comes
up.

### 7.12 `ALRT`/`DITL` 1046 — "you are ineligible"

```
ALRT 1046 'No High Score'  bounds (T=40 L=40 B=132 R=300) = 260x92, itemsID 1046, stages 0x4444
DITL 1046: 3 items, 142 bytes
  item  1  rect(T=64 L=180 B=84 R=252) 72x20   Button      enabled  'So What?'
  item  2  rect(T=8  L=8   B=56 R=216) 208x48  StaticText  disabled 'If you resume a saved game, you are ineligible to get on the high scores for that game.'
  item  3  rect(T=8  L=220 B=40 R=252) 32x32   Icon        disabled resID=1073
```

Shown by

```c
void HeyYourPissingAHighScore (void)
{
	#define		kNoHighScoreAlert	1046
	short		whoCares;

//	CenterAlert(kNoHighScoreAlert);
	whoCares = Alert(kNoHighScoreAlert, nil);
}
```
(`GliderPRO/Sources/Menu.c:779-786`) from exactly one place:
`GliderPRO/Sources/Menu.c:319`, immediately before `OpenSavedGame()` in the
`iOpenSavedGame` menu handler. Since `OpenSavedGame()` unconditionally returns
`false` in this build (§9), the alert is the *only* observable effect of the
Open Saved Game menu item.

### 7.13 How a new high score reaches the disk

```
TestHighScore()  --sets-->  gameDirty = true                  HighScores.c:414
                                  |
CloseHouse() / SaveHouse()  --reads gameDirty-->
    if (houseIsReadOnly)   { WriteScoresToDisk() }             // dead, 7.1
    else                   { WriteHouse(theMode == kEditMode) } HouseIO.c:531-539
                                  |
WriteHouse(checkIt):                                            HouseIO.c:448-505
 1. SetFPos(houseRefNum, fsFromStart, 0)
 2. CopyThisRoomToRoom()
 3. if (checkIt) CheckHouseForProblems()
 4. HLock(thisHouse)
 5. byteCount = GetHandleSize((Handle)thisHouse)                // 866 + 348*nRooms
 6. if (fileDirty)
 7.     GetDateTime(&timeStamp);  timeStamp &= 0x7FFFFFFF
 8.     if (changeLockStateOfHouse) houseUnlocked = !saveHouseLocked
 9.     if (houseUnlocked) timeStamp &= 0x7FFFFFFE else timeStamp |= 0x00000001
10.     version = wasHouseVersion
11. FSWrite(houseRefNum, &byteCount, *thisHouse)                // ONE write, whole file
12. SetEOF(houseRefNum, byteCount)
13. HUnlock(thisHouse)
```

Consequences:

* **The high-score table is never written on its own.** The entire house — all
  rooms, all objects, the `savedGame` blob, everything — is rewritten. A Go
  port can legitimately write only bytes 528..819 (the `scoresType` slice) and
  produce a byte-identical file, provided nothing else in the handle changed.
* **`gameDirty` alone does not update `timeStamp` or `version`** (step 6 tests
  `fileDirty`, which `TestHighScore` does not set). So a house whose only
  change is a new high score keeps its original `timeStamp` and the low bit
  that encodes `houseUnlocked`.
* **`houseUnlocked` does *not* gate the high-score write.** The only guard on
  the `WriteHouse` branch is `houseIsReadOnly`, which is always false (7.1). A
  house locked against editing still accumulates high scores.
* **`houseType.timeStamp` decoded (verified).** It is
  `GetDateTime() & 0x7FFFFFFF` with **bit 0 repurposed as the house-lock
  flag**: `houseUnlocked = ((timeStamp & 0x00000001) == 0)`
  (`GliderPRO/Sources/HouseIO.c:402`). Because Mac date-times in the mid-1990s
  are around `2.87e9`, the `& 0x7FFFFFFF` **discards bit 31**; to recover the
  real wall-clock date you must add `0x80000000` back. Verified against all 22
  shipped houses — every one then decodes to a plausible authoring date, and
  the low bit matches the expected lock state:

| house | raw `timeStamp` | bit 0 (locked) | `+ 0x80000000` as a Mac date |
|---|---|---|---|
| The Asylum Pro | 738016929 | 1 | 1995-06-08 23:56:17 |
| ImagineHouse PRO II | 740149565 | 1 | 1995-07-03 16:20:13 |
| Nemo's Market | 741121233 | 1 | 1995-07-14 22:14:41 |
| Rainbow's End | 741121489 | 1 | 1995-07-14 22:18:57 |
| SpacePods | 741122073 | 1 | 1995-07-14 22:28:41 |
| Empty House | 741421292 | 0 | 1995-07-18 09:35:40 |
| Grand Prix | 741444157 | 1 | 1995-07-18 15:56:45 |
| In The Mirror | 741512193 | 1 | 1995-07-19 10:50:41 |
| Leviathan | 741514803 | 1 | 1995-07-19 11:34:11 |
| Castle o' the Air | 741536868 | 0 | 1995-07-19 17:41:56 |
| Metropolis | 741642315 | 1 | 1995-07-20 22:59:23 |
| CD Demo House | 742294873 | 1 | 1995-07-28 12:15:21 |
| Fun House | 742681616 | 0 | 1995-08-01 23:41:04 |
| Davis Station | 743681789 | 1 | 1995-08-13 13:30:37 |
| Demo House | 743682113 | 1 | 1995-08-13 13:36:01 |
| Slumberland | 743682377 | 1 | 1995-08-13 13:40:25 |
| Teddy World | 744401697 | 1 | 1995-08-21 21:29:05 |
| Land of Illusion | 745268052 | 0 | 1995-08-31 22:08:20 |
| California or Bust! | 745882296 | 0 | 1995-09-08 00:45:44 |
| Art Museum | 746622005 | 1 | 1995-09-16 14:14:13 |
| Titanic | 755123966 | 0 | 1995-12-23 23:53:34 |
| Sampler | 893435880 | 0 | 2000-05-11 19:52:08 |

  So 15 of the 22 are locked and 7 unlocked, and `Sampler` is a much later
  addition (2000) than the 1995 originals. A Go port must keep bit 0 as a flag
  and must **not** widen the field to a full 32-bit time, or every existing
  house file will read as five and a half years too old and will flip its lock
  state on the first save.
* One more editor-side consequence: `Menu.c:456` guards the editor's Save with
  `if ((fileDirty) && (houseUnlocked))`, and the `SaveGame(false)` call that
  would have cleared `hasGame` on an editor save is **commented out**
  (`GliderPRO/Sources/Menu.c:458`), which is why the shipped houses still
  carry stale `savedGame` blobs (section 9.5).
* File size arithmetic verified on two houses:
  `ImagineHouse PRO II` `nRooms = 279`, data fork **97958** = `866 + 348*279`
  exactly; `Titanic` `nRooms = 208`, data fork **73250** = `866 + 348*208`
  exactly. `Sampler` is 2 bytes longer than the formula predicts (see Open
  questions).
* `wasHouseVersion` is captured at load (`GliderPRO/Sources/HouseIO.c:394`)
  and, if it was below `kHouseVersion` = `0x0200`, promoted to `0x0200`
  (`GliderPRO/Sources/HouseIO.c:610-612`). `kNewHouseVersion` = `0x0300`
  triggers a `YellowAlert(kYellowNewerVersion)`
  (`GliderPRO/Sources/HouseIO.c:395`).

### 7.14 The side-car file format (dead, but specified)

If `houseIsReadOnly` were ever true, the side-car would be:

* **Path** `<system disk>:System Folder:Preferences:G-PRO Scores ƒ:<thisHouseName>`.
  The folder is located by `FindFolder(kOnSystemDisk, kPreferencesFolderType,
  kCreateFolder, &volRefNum, &prefsDirID)` and then a `PBGetCatInfo` scan of
  `prefsDirID` with `ioFDirIndex = 1, 2, 3, …` looking for a *directory*
  (`ioFlAttrib & 0x10`) whose name `EqualString`s `"G-PRO Scores ƒ"`
  case- and diacritical-insensitively
  (`GliderPRO/Sources/HighScores.c:665-716`). If the scan ends in `fnfErr`,
  `CreateScoresFolder()` makes it with `FSpDirCreate`
  (`GliderPRO/Sources/HighScores.c:642-661`).
* **Type/creator** `'gliS'` / `'ozm5'`, created with `FSpCreate(scoreSpec,
  'ozm5', 'gliS', smSystemScript)` — note the argument order:
  `FSpCreate(spec, creator, fileType, scriptTag)`, so the **creator is `ozm5`
  and the type is `gliS`** (`GliderPRO/Sources/HighScores.c:727`).
* **Contents** exactly `sizeof(scoresType)` = **292** bytes, the raw struct,
  big-endian, written from `&((*thisHouse)->highScores)` at file offset 0,
  followed by `SetEOF(refNum, 292)`
  (`GliderPRO/Sources/HighScores.c:771-790`).
* **Reading** `GetEOF(scoresRefNum, &byteCount)` then
  `FSRead(scoresRefNum, &byteCount, theScores)` where `theScores` points at the
  292-byte `highScores` member inside the house handle
  (`GliderPRO/Sources/HighScores.c:823-841`). **`byteCount` is never clamped to
  292.** A side-car larger than 292 bytes overruns `highScores` and corrupts
  `savedGame`, `hasGame`, `firstRoom`, `nRooms` and then the room array — and,
  for a big enough file, the Memory Manager block after the handle. A Go port
  must clamp.

### 7.15 What the preferences file stores

`Preferences:Glider Prefs`, type `'gliP'`, creator `'ozm5'`
(`GliderPRO/Sources/Prefs.c:18-20`, created at `:113`). The relevant members of
`prefsInfo` (`GliderPRO/Headers/Externs.h:233-267`, under
`#pragma options align=mac68k`):

```c
typedef struct
{
	Str32		wasDefaultName;
	Str15		wasLeftName, wasRightName;
	Str15		wasBattName, wasBandName;
	Str15		wasHighName;
	Str31		wasHighBanner;
//	long		encrypted, fakeLong;
	long		wasLeftMap, wasRightMap;
	...
} prefsInfo;
```

Round trip:

| Direction | Code |
|---|---|
| load | `PasStringCopy(thePrefs.wasHighName, highName); PasStringCopy(thePrefs.wasHighBanner, highBanner);` — `GliderPRO/Sources/Main.c:67-68` |
| defaults when no prefs file | `PasStringCopy("\pYour Name", highName); PasStringCopy("\pYour Message Here", highBanner);` — `GliderPRO/Sources/Main.c:133-134` |
| save | `PasStringCopy(highName, thePrefs.wasHighName); PasStringCopy(highBanner, thePrefs.wasHighBanner);` — `GliderPRO/Sources/Main.c:223-224` |

Those two strings are the *pre-filled contents of the entry dialogs only*. No
score, timestamp or room count is ever stored in the prefs file. That neatly
explains the shipped houses whose banner is literally `Your Message Here`
(§7.3): the champion accepted the default.

`prefVersion` is checked on load (`GliderPRO/Sources/Prefs.c:261`: `if
(thePrefs->prefVersion != versionNeed)` → treat as absent) and stamped on save
(`:153`).

### 7.16 Complete entry-point census for the high-score subsystem

| Function | Defined | Called from |
|---|---|---|
| `DoHighScores()` | `GliderPRO/Sources/HighScores.c:58` | `GliderPRO/Sources/Menu.c:418` (Options → High Scores); `GliderPRO/Sources/HighScores.c:421` (tail of `TestHighScore`) |
| `DrawHighScores()` | `GliderPRO/Sources/HighScores.c:94` | `GliderPRO/Sources/HighScores.c:74` only |
| `SortHighScores()` | `GliderPRO/Sources/HighScores.c:281` | `GliderPRO/Sources/HighScores.c:413` only |
| `ZeroHighScores()` | `GliderPRO/Sources/HighScores.c:323` | `GliderPRO/Sources/House.c:133`; `GliderPRO/Sources/HouseInfo.c:329` |
| `ZeroAllButHighestScore()` | `GliderPRO/Sources/HighScores.c:348` | `GliderPRO/Sources/HouseInfo.c:335` |
| `TestHighScore()` | `GliderPRO/Sources/HighScores.c:374` | `GliderPRO/Sources/GameOver.c:67`; `GliderPRO/Sources/GameOver.c:503` |
| `GetHighScoreName()` | `GliderPRO/Sources/HighScores.c:498` | `GliderPRO/Sources/HighScores.c:403` |
| `GetHighScoreBanner()` | `GliderPRO/Sources/HighScores.c:608` | `GliderPRO/Sources/HighScores.c:407` |
| `WriteScoresToDisk()` | `GliderPRO/Sources/HighScores.c:742` | `GliderPRO/Sources/HouseIO.c:331`, `:535` — both dead |
| `ReadScoresFromDisk()` | `GliderPRO/Sources/HighScores.c:801` | `GliderPRO/Sources/HouseIO.c:431` — dead |
| `CreateScoresFolder()` | `GliderPRO/Sources/HighScores.c:642` | `GliderPRO/Sources/HighScores.c:709` |
| `FindHighScoresFolder()` | `GliderPRO/Sources/HighScores.c:665` | `GliderPRO/Sources/HighScores.c:751`, `:810` |
| `OpenHighScoresFile()` | `GliderPRO/Sources/HighScores.c:720` | `GliderPRO/Sources/HighScores.c:758`, `:817` |
| `HeyYourPissingAHighScore()` | `GliderPRO/Sources/Menu.c:779` | `GliderPRO/Sources/Menu.c:319` |
| `HowToZeroScores()` | `GliderPRO/Sources/HouseInfo.c:319` | `GliderPRO/Sources/HouseInfo.c:293` |
| `CountRoomsVisited()` | `GliderPRO/Sources/House.c:511` | `GliderPRO/Sources/HighScores.c:412` (and the Map window) |

Module globals owned by `HighScores.c` (`GliderPRO/Sources/HighScores.c:45-48`):

| Global | Type | Meaning | Reset |
|---|---|---|---|
| `highBanner` | `Str31` | remembered banner text for the dialog | prefs load / `"Your Message Here"` |
| `highName` | `Str15` | remembered player name for the dialog | prefs load / `"Your Name"` |
| `lastHighScore` | `short` | index of the row to highlight white, or -1 | `-1` at `GliderPRO/Sources/InterfaceInit.c:161`, and at the top of every `TestHighScore()` |
| `keyStroke` | `Boolean` | "a key was typed, refresh the counter next filter call" | set/cleared inside the two filters; **never initialised** at startup |

`keyStroke` is a file-scope `Boolean`, so it is zero-initialised by the C
runtime; it is fine, but a Go port should not assume any other lifetime.

---

## 8. Game over and win

There are exactly **two** endings, and they are selected by the *sign of
`mortals`* at the moment the countdown expires
(`GliderPRO/Sources/Play.c:546-549`):

| `mortals` at countdown expiry | function called | meaning |
|---|---|---|
| `>= 0` | `DoGameOver()` (`GliderPRO/Sources/GameOver.c:60`) | **you win** — the last star was collected while at least one glider remained |
| `< 0` | `DoDiedGameOver()` (`GliderPRO/Sources/GameOver.c:443`) | **you lost** — the last glider was destroyed |

```c
if (mortals < 0)
    DoDiedGameOver();
else
    DoGameOver();
```
— `GliderPRO/Sources/Play.c:546-549`

Nothing else distinguishes the two paths. In particular there is no
"did you collect the last star" flag; the win/lose decision is a
one-line sign test on the life counter. Section 8.9 shows the one case
where that test gives the wrong answer.

### 8.1 File-scope state owned by `GameOver.c`

`GliderPRO/Sources/GameOver.c:39-46`:

| declaration | line | purpose |
|---|---|---|
| `pageType pages[8]` | :39 | doubles as (a) the 8 fluttering "GAME OVER" pages and (b) the 5 falling stars of the win animation |
| `Rect pageSrcRect` | :40 | scratch; reused three times during `InitDiedGameOver()` |
| `Rect pageSrc[kPageFrames]` | :40 | 14 source cells, each 32x32, into `pageSrcMap` |
| `Rect lettersSrc[8]` | :40 | 8 source cells, each 25x32, into `gameOverSrcMap` |
| `Rect angelSrcRect` | :40 | 96x44; set once by `StructuresInit2.c:127`, **not** by `GameOver.c` |
| `RgnHandle roomRgn` | :41 | `RectRgn(justRoomsRect)`; clips the letter blits |
| `GWorldPtr pageSrcMap` | :42 | 32 x 448, 8-bit, PICT 1990 |
| `GWorldPtr gameOverSrcMap` | :42 | 25 x 256, 8-bit, PICT 1988 |
| `GWorldPtr angelSrcMap` | :42 | 96 x 44, 8-bit, PICT 1019 (created in `StructuresInit2.c:128-130`) |
| `GWorldPtr pageMaskMap` | :43 | 32 x 448, **1-bit**, PICT 1989 |
| `GWorldPtr angelMaskMap` | :43 | 96 x 44, **1-bit**, PICT 1020 (created in `StructuresInit2.c:132-134`) |
| `short countDown` | :44 | frames left before the ending starts |
| `short stopPages` | :44 | y coordinate at which the pages stop falling |
| `short pagesStuck` | :44 | how many of the 8 pages have landed; loop terminator |
| `Boolean gameOver` | :45 | the flag itself |

`pages[]` is deliberately shared between the two endings. The win path
only ever uses `pages[0..4]`; the loss path uses all 8. They never run
in the same game, so the aliasing is safe — but a Go port that gives
each ending its own slice must not be surprised that the C code reads
stale `pages[]` values (`frame`, `counter`, `stuck`) left over from a
previous game: **`InitDiedGameOver()` re-initialises all four fields,
and `SetUpFinalScreen()` re-initialises `dest`, `was` and `frame` but
NOT `stuck` or `counter`.** The win animation never reads `stuck` or
`counter`, so this is latent, not live.

Constants (`GliderPRO/Sources/GameOver.c:17-22`):

| macro | value | used for |
|---|---|---|
| `kNumCountDownFrames` | 16 | frames between `FlagGameOver()` and the ending |
| `kPageFrames` | 14 | animation cells in the page sprite sheet |
| `kPagesPictID` | 1990 | 32 x 448 8-bit page sheet |
| `kPagesMaskID` | 1989 | 32 x 448 1-bit page mask |
| `kLettersPictID` | 1988 | 25 x 256 8-bit "GAME OVER" letter sheet |
| `kMilkywayPictID` | 1021 | 640 x 460 starfield for the win screen |

Local constants declared inside functions:

| macro | value | line | used for |
|---|---|---|---|
| `kStarFalls` | 8 | `GliderPRO/Sources/GameOver.c:138` | px/frame a win-screen star falls |
| `kPageSpacing` | 40 | `GliderPRO/Sources/GameOver.c:251` | horizontal pitch of the 8 letters |
| `kPageRightOffset` | 128 | `GliderPRO/Sources/GameOver.c:252` | **never referenced** — dead |
| `kPageBackUp` | 128 | `GliderPRO/Sources/GameOver.c:253` | **never referenced** — dead |

### 8.2 `FlagGameOver()` — the trigger

```c
void FlagGameOver (void)
{
    gameOver = true;
    countDown = kNumCountDownFrames;
    SetMusicalMode(kPlayWholeScoreMode);
}
```
— `GliderPRO/Sources/GameOver.c:237-242`

`kPlayWholeScoreMode` is `-1` (`GliderPRO/Headers/GliderDefines.h:50`).

There are exactly **three** call sites (already tabulated in section
3.6; repeated here because they are the complete set of ways a game can
end normally):

| call site | condition | `mortals` at the call |
|---|---|---|
| `GliderPRO/Sources/Interactions.c:944` | `numStarsRemaining--` reached `<= 0` in the `kStar` arm | `>= 0` normally (see 8.9) |
| `GliderPRO/Sources/Player.c:1513` | `mortals--` went below 0 (1-player), i.e. the last glider died | `-1` |
| `GliderPRO/Sources/Player.c:1500` | 2-player: both gliders gone | `-1` |

`gameOver` is cleared in exactly one place, `GliderPRO/Sources/Play.c:82`
(`gameOver = false;` inside `NewGame`), so it is a strictly one-shot
latch per game.

### 8.3 The 16-frame countdown, and what keeps running during it

`PlayGame()` (`GliderPRO/Sources/Play.c:430-497`) does **not** stop the
simulation when `gameOver` becomes true. It suppresses only the three
input/collision/glider calls:

```
 1. while ((playing) && (!quitting)):
 2.     gameFrame++                              // still counting
 3.     evenFrame = !evenFrame
 4.     if (doBackground): do HandlePlayEvent() while (switchedOut)
 5.     HandleTelephone()
 6.     HandleDynamics()                         // ALWAYS
 7.     if (!gameOver):
 8.         GetInput(&theGlider)  [+ GetInput(&theGlider2) if twoPlayerGame]
 9.         HandleInteraction()
10.     HandleTriggers()                         // ALWAYS
11.     HandleBands()                            // ALWAYS
12.     if (!gameOver):
13.         HandleGlider(&theGlider)  [+ theGlider2]
14.     if (playing):
15.         MoviesTask() if QT + hasMovie + tvInRoom + tvOn
16.         RenderFrame()                        // ALWAYS - this is the 2-tick pacer
17.         HandleDynamicScoreboard()            // ALWAYS
18.     if (gameOver):
19.         countDown--
20.         if (countDown <= 0): ...run the ending...
```

Consequences a port must reproduce:

* `HandleDynamics()`, `HandleTriggers()` and `HandleBands()` continue,
  so blowers keep blowing, clocks keep ticking, dynamic objects keep
  moving and in-flight rubber bands keep travelling for 16 more frames.
* `HandleDynamicScoreboard()` continues, so **the score roll keeps
  rolling and `kScoreTikSound` keeps ticking during the countdown**
  (`GliderPRO/Sources/Scoreboard.c:91`; see section 2.9). Because the
  roll advances only 13 points per frame, a large final award is still
  mid-roll when the ending starts and the remaining animation is simply
  abandoned; the *stored* `theScore` is already final, so the high-score
  test is unaffected.
* The glider is frozen in place (no `HandleGlider`) but still drawn by
  `RenderFrame()`.
* Frame pacing is `kTicksPerFrame` = 2 (`GliderPRO/Headers/GliderDefines.h:533`),
  enforced by `while (TickCount() < nextFrame)` at
  `GliderPRO/Sources/Render.c:662-665`. 16 frames x 2 ticks = **32 ticks
  = 8/15 s ~= 0.533 s** between the trigger and the ending.

### 8.4 The dispatch block

`GliderPRO/Sources/Play.c:499-553` (non-arcade build; the
`BUILD_ARCADE_VERSION` branch additionally blacks out the scoreboard
GWorld and re-draws `kScoreboardPictID` at `hOffset`):

```
 1. if (gameOver):
 2.     countDown--
 3.     if (countDown <= 0):
 4.         GetGWorld(&wasCPort, &wasWorld)          // save the current port
 5.         HideGlider(&theGlider)                   // erase the glider sprite
 6.         RefreshScoreboard(kNormalTitleMode)      // drop "Saving..."/etc titles
 7.         if (mortals < 0) DoDiedGameOver() else DoGameOver()
 8.         SetGWorld(wasCPort, wasWorld)            // restore
```

Note `HideGlider(&theGlider)` only — in a 2-player game **`theGlider2`
is never hidden**, so the second player's sprite is left burned into
`workSrcMap` and appears in the ending backdrop that
`DoDiedGameOver()` copies up from `mainWindow`.

### 8.5 The win path: `DoGameOver()`

```c
void DoGameOver (void)
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
— `GliderPRO/Sources/GameOver.c:60-69`

`playing = false` ends `PlayGame()`'s loop as soon as control returns,
so everything from here on happens *inside* frame N of the play loop.

`mainWindowRect` in play mode is the screen rect with its corner zeroed
and its bottom reduced by the 20-pixel menu bar
(`GliderPRO/Sources/MainWindow.c:222-225`), i.e. `(0,0,460,640)` on a
640x480 screen.

#### 8.5.1 Colour index 244, and the app's palette — verified

`ColorRect(&r, long color)` is
`Index2Color(color, &rgb); RGBForeColor(&rgb); PaintRect(r);`
(`GliderPRO/Sources/ColorUtils.c:36-45`). So `244` is a **palette
index**, not a QuickDraw colour constant.

The app ships two `clut` resources, 128 and 129. I decoded both:

```
clut 128 seed=0x00000000 flags=0x0000 size=255 (=> 256 entries)   2056 bytes
clut 129 seed=0x00000000 flags=0x0000 size=255 (=> 256 entries)   2056 bytes
clut128 == clut129 ?  True          (byte-for-byte identical)
value fields sequential 0..255?  True
```

and verified that **every 8-bit PICT in the application embeds exactly
this same 256-entry colour table**:

```
PICT 1988  palette entries=256  mismatches vs clut128=0
PICT 1990  palette entries=256  mismatches vs clut128=0
PICT 4002  palette entries=256  mismatches vs clut128=0
PICT 1021  palette entries=256  mismatches vs clut128=0
PICT 1995  palette entries=256  mismatches vs clut128=0
PICT 1994  palette entries=256  mismatches vs clut128=0
```

The table is the **standard Macintosh 8-bit system palette**, which I
confirmed structurally (zero mismatches on both halves):

* indices **0..214**: the 6x6x6 colour cube. For index `i`,
  `red = L[i / 36]`, `green = L[(i % 36) / 6]`, `blue = L[i % 6]`
  with `L = {255, 204, 153, 102, 51, 0}` (i.e. `0xFF, 0xCC, 0x99,
  0x66, 0x33, 0x00`). Note the cube is *descending*: index 0 is white.
* indices **215..224**: pure-red ramp, R = `0xEE, 0xDD, 0xBB, 0xAA,
  0x88, 0x77, 0x55, 0x44, 0x22, 0x11`, G = B = 0.
* indices **225..234**: the same 10 levels in green only.
* indices **235..244**: the same 10 levels in blue only.
* indices **245..254**: the same 10 levels as greys (R = G = B).
* index **255**: `#000000`.

Therefore the indices this document quotes resolve as:

| index | RGB | where used |
|---|---|---|
| 0 | `#FFFFFF` | PICT background / paper white |
| 23 | `#FF6600` | `kRedOrangeColor8` (`GliderPRO/Headers/GliderDefines.h`) |
| 172 | `#333333` | scoreboard badge fill (section 5) |
| **244** | **`#000011`** | `ColorRect(&mainWindowRect, 244)` and `ColorRect(&workSrcRect, 244)` — the near-black deep blue of the win screen |
| 250 | `#777777` | letter-sheet cell borders |
| 255 | `#000000` | true black |

Caveat for a Go port: the GWorlds are made with
`NewGWorld(theGWorld, 8, bounds, nil /* cTable */, nil, useTempMem)`
(`GliderPRO/Sources/Utilities.c`, `CreateOffScreenGWorld`), i.e. a **nil**
colour table, so classic Mac OS gives them the default 8-bit device
table — the same standard system palette. The app installs no `pltt`
resource and `SetPaletteToGrays()` is entirely commented out
(`GliderPRO/Sources/MainWindow.c:449-...`, and its one call site
`:253` is commented out too). So the palette is *fixed* and a Go port
can hard-code the 256 RGB triples from the formula above rather than
carrying a `clut` blob.

#### 8.5.2 `SetUpFinalScreen()` — the winner's backdrop

`GliderPRO/Sources/GameOver.c:76-129`.

```
 1. SetPort(workSrcMap)
 2. ColorRect(&workSrcRect, 244)                      // fill #000011
 3. QSetRect(&tempRect, 0, 0, 640, 460)
 4. CenterRectInRect(&tempRect, &workSrcRect)
 5. LoadScaledGraphic(kMilkywayPictID /*1021*/, &tempRect)
 6. textDown = tempRect.top;  if (textDown < 0) textDown = 0
 7. HLock(thisHouse);  PasStringCopy((*thisHouse)->trailer, tempStr);  HSetState
 8. count = 0
 9. do:
10.     GetLineOfText(tempStr, count, subStr)         // split on ASCII 0x0D
11.     offset = ((thisMac.screen.right - thisMac.screen.left)
12.               - TextWidth(subStr, 1, subStr[0])) / 2
13.     TextFont(applFont); TextFace(bold); TextSize(12)
14.     ForeColor(blackColor)                          // QuickDraw constant 33
15.     MoveTo(offset + 1, textDown + 33 + count*20);  DrawString(subStr)
16.     ForeColor(whiteColor)                          // QuickDraw constant 30
17.     MoveTo(offset,     textDown + 32 + count*20);  DrawString(subStr)
18.     ForeColor(blackColor)
19.     count++
20. while (subStr[0] > 0)
21. CopyRectWorkToBack(&workSrcRect)                   // work -> back buffer
22. for i in 0..4:                                     // seed 5 falling stars
23.     pages[i].dest = starSrc[0]                     // 32 x 31
24.     QOffsetRect(&pages[i].dest,
25.         workSrcRect.right + RandomInt(workSrcRect.right / 5)
26.                            + (workSrcRect.right / 4) * i,
27.         RandomInt(workSrcRect.bottom) - workSrcRect.bottom / 2)
28.     pages[i].was = pages[i].dest
29.     pages[i].frame = RandomInt(6)
```

Details:

* `LoadScaledGraphic(resID, rect)` is `GetPicture` / `DrawPicture(pic, rect)`
  / `ReleaseResource` (`GliderPRO/Sources/Utilities.c:340-349`), so
  **the destination rect scales the art**. PICT 1021 is natively
  640 x 460 (verified: `picFrame = (0,0,460,640)`, 51520 bytes, PICT v2,
  8-bit PackBitsRect; note its 16-bit `picSize` field reads `-14016`
  because 51520 overflows a signed short — a port must ignore `picSize`
  and use the resource length), and the destination is also 640 x 460,
  so on any screen the starfield is drawn **1:1, centred, not scaled**.
* On a 640 x 480 screen `workSrcRect` is `(0,0,460,640)` (section 5),
  so `tempRect` centres to exactly `(0,0,460,640)`, `textDown = 0`,
  and the first trailer line's baseline is y = 32.
  On 1024 x 768, `workSrcRect` is `(0,0,748,1024)`, `tempRect` becomes
  `(144,192,604,832)`, `textDown = 144`, first baseline y = 176.
* The text is drawn **twice**: black at `(offset+1, base+1)` then white
  at `(offset, base)` — a 1-pixel bottom-right drop shadow.
* `offset` is computed from the **screen** width, but drawn into
  `workSrcMap`, whose width is `min(screenWidth, kMaxViewWidth /*1536*/)`
  (`GliderPRO/Sources/InterfaceInit.c:196-201`). They differ only on
  screens wider than 1536 px, where the trailer text would be pushed
  off the right edge.
* `GetLineOfText(srcStr, index, textLine)`
  (`GliderPRO/Sources/StringUtils.c`) splits a Pascal string on
  `kReturnKeyASCII` = **0x0D** and returns line `index` (0-based),
  or an empty string when `index` is past the last line. The do/while
  therefore draws every line and then one final empty line before
  exiting. Line spacing is a hard-coded **20 px**.
* The trailer comes from `houseType.trailer`, a `Str255` at house-file
  offset **272** (section 7.2). It is the "credits" text the house
  author typed in the editor.
* `starSrc[0]` is `(0,0,31,32)` — 32 wide, 31 tall — offset from
  `bonusSrcRect`; `starSrc[i] = QSetRect(0,0,32,31)` then
  `QOffsetRect(48, i*31)` for i in 0..5
  (`GliderPRO/Sources/StructuresInit.c:389-393`). The sheet is
  `bonusSrcMap`, 88 x 378, PICT 4002, with a 1-bit mask from PICT 5002
  (`GliderPRO/Sources/StructuresInit.c:350-357`).
* Star seeding on a 640 x 480 screen (`workSrcRect` = `(0,0,460,640)`):
  `left = 640 + RandomInt(128) + 160*i`, i.e. i=0 -> 640..767,
  i=1 -> 800..927, i=2 -> 960..1087, i=3 -> 1120..1247,
  i=4 -> 1280..1407 — **all off the right edge**, and
  `top = RandomInt(460) - 230`, i.e. -230..229. These seeded positions
  are then *overwritten* by `DoGameOverStarAnimation()` before the star
  is ever drawn (see below), so the only thing that survives seeding is
  `pages[i].frame = RandomInt(6)`.

`RandomInt(range)` is `rawResult = Random(); if (rawResult < 0) rawResult *= -1;
return (rawResult * range) / 32768;` (`GliderPRO/Sources/Utilities.c:72-80`).
Because the Toolbox `Random()` can return `-32768`, whose negation is
`+32768`, **`RandomInt(n)` can return `n` itself**, not just `0..n-1`.
A Go port using `rand.Intn(n)` will be very slightly different; to be
exact, emulate `abs(int16 Random()) * n / 32768`.

#### 8.5.3 `DoGameOverStarAnimation()` — the angel

`GliderPRO/Sources/GameOver.c:136-230`.

```
 1. angelDest = angelSrcRect            // (0,0,44,96)
 2. QOffsetRect(&angelDest, -96, 0)     // -> (0,-96,44,0): just off the left edge
 3. noInteruption = true
 4. nextLoop = TickCount() + 2
 5. count = 0                           // how many stars are live
 6. pass  = 0                           // frames since the angel left the screen
 7. FlushEvents(everyEvent, 0)
 8. while (noInteruption):
 9.     if ((angelDest.left % 32) == 0):                 // every 32 px of travel
10.         PlayPrioritySound(kMysticSound, kMysticPriority)     // 35 / 202
11.         which = (angelDest.left / 32) % 5
12.         ZeroRectCorner(&pages[which].dest)           // normalise to (0,0,31,32)
13.         QOffsetRect(&pages[which].dest, angelDest.left, angelDest.bottom)
14.         if (count < which + 1) count = which + 1
15.     for i in 0..count-1:
16.         pages[i].frame++;  if (pages[i].frame >= 6) pages[i].frame = 0
17.         CopyMask(bonusSrcMap, bonusMaskMap, workSrcMap,
18.                  &starSrc[frame], &starSrc[frame], &pages[i].dest)
19.         pages[i].was = pages[i].dest;  pages[i].was.top -= kStarFalls /*8*/
20.         AddRectToWorkRectsWhole(&pages[i].was)
21.         AddRectToBackRects(&pages[i].dest)
22.         if (pages[i].dest.top < workSrcRect.bottom)
23.             QOffsetRect(&pages[i].dest, 0, kStarFalls)   // fall 8 px
24.     if (angelDest.left <= workSrcRect.right + 2):
25.         CopyMask(angelSrcMap, angelMaskMap, workSrcMap,
26.                  &angelSrcRect, &angelSrcRect, &angelDest)
27.         angelDest.left -= 2;  AddRectToWorkRectsWhole(&angelDest)
28.         angelDest.left += 2;  AddRectToBackRects(&angelDest)
29.         QOffsetRect(&angelDest, 2, 0)                   // fly right 2 px
30.         pass = 0                                        // reset the exit timer
31.     CopyRectsQD()
32.     numWork2Main = 0;  numBack2Work = 0
33.     do:
34.         GetKeys(theKeys)
35.         if Command || Option || Shift || Control held: noInteruption = false
36.         if GetNextEvent(everyEvent, &theEvent):
37.             if (what == mouseDown || what == keyDown): noInteruption = false
38.     while (TickCount() < nextLoop)
39.     nextLoop = TickCount() + 2
40. .   if (pass < 80) pass++
41.     else: WaitForInputEvent(5); noInteruption = false
```

Behavioural notes:

* The angel is `angelSrcRect` = `QSetRect(0,0,44,96)` -> **96 wide,
  44 tall** (`GliderPRO/Sources/StructuresInit2.c:127`), drawn from
  `angelSrcMap` (PICT `kAngelPictID` = **1019**, verified 96 x 44,
  PICT v2, 4064 bytes) through `angelMaskMap` (PICT
  `kAngelPictID + 1` = **1020**, verified 96 x 44, **PICT v1**,
  457 bytes) (`GliderPRO/Sources/StructuresInit2.c:20, 128-134`).
  These two GWorlds are created once at startup and are **never
  disposed** by `GameOver.c`.
* **Naming hazard:** resource ID 1020 is *both* PICT 1020 (the angel
  mask) and DLOG/DITL 1020 (the "enter your high score name" dialog,
  section 7.11). Different resource types, same number.
* The angel starts at `left = -96` and advances **+2 px per frame**, so
  `angelDest.left` is always even and the `% 32 == 0` test fires every
  **16 frames** = 32 ticks = 0.533 s. A star is (re)spawned and
  `kMysticSound` plays on each of those frames.
* `which = (angelDest.left / 32) % 5` — C integer division truncates
  **toward zero**, so for the negative starting positions
  `-96/32 = -3`, `-64/32 = -2`, `-32/32 = -1`, `0/32 = 0`. In C,
  `-3 % 5 == -3` (implementation-defined pre-C99, but -3 on every Mac
  compiler of the era). So the first three iterations index
  `pages[-3]`, `pages[-2]`, `pages[-1]` — **out-of-bounds 8-byte writes
  at 66, 44 and 22 bytes below the start of `pages[]`**
  (`sizeof(pageType)` = 8 + 8 + 2 + 2 + 1 = 21, padded to **22** under
  `#pragma options align=mac68k`), i.e. into whatever the linker placed
  before `pages`, and `count` is set to `which + 1` which is
  `-2`, `-1`, `0` -> the `for i in 0..count-1` loop body does not run.
  A Go port must not reproduce the memory corruption, but it **must**
  reproduce the visible effect: no star is drawn for the first three
  32-px steps (angel left = -96, -64, -32), and the first real star
  appears when the angel reaches `left = 0`.
  Go's `%` also returns `-3` for `-3 % 5`, so the arithmetic ports
  directly; guard `which < 0` and skip.
* Star respawn uses `ZeroRectCorner` then offsets to
  `(angelDest.left, angelDest.bottom)` — i.e. each star is dropped from
  the angel's trailing bottom-left corner, `angelDest.bottom` being
  `44` for the whole animation since the angel only moves horizontally.
* `count` therefore grows 1, 2, 3, 4, 5 as the angel passes
  `left = 0, 32, 64, 96, 128`, and thereafter stays 5 while slot
  `which` cycles 0,1,2,3,4,0,1,... So a star already falling is
  teleported back up to the angel every 5th spawn.
* Star animation is a 6-frame cycle (`frame++`, wrap at 6) advancing
  **every frame**, and the star falls 8 px/frame until
  `dest.top >= workSrcRect.bottom`, after which it freezes at the
  bottom (it is still drawn every frame, clipped).
* Termination: `pass` is reset to 0 on every frame the angel is still
  within `workSrcRect.right + 2`. Once the angel has flown off the
  right edge, `pass` counts up; at `pass == 80` (80 frames = 160 ticks
  = **2.67 s**) the loop calls `WaitForInputEvent(5)` and exits.
  Total run time on a 640-wide view: the angel travels from -96 to 642
  at 2 px/frame = 369 frames, plus 80 = **449 frames ~= 15 s**, plus up
  to 5 s of `WaitForInputEvent`.
* `WaitForInputEvent(short seconds)`
  (`GliderPRO/Sources/Utilities.c:439-479`) waits up to
  `60 * seconds` ticks, returning early on any of Command/Option/Shift/
  Control held, a `mouseDown`, a `keyDown`, or an `osEvt` resume;
  `seconds == -1` means wait forever.
* Any modifier key or click aborts the whole animation immediately.
* `CopyRectsQD()` flushes the accumulated dirty rects
  (work->main and back->work), then both counters are zeroed
  (`GliderPRO/Sources/GameOver.c:204-207`). This is the standard
  double-buffer flush used by the play loop.

### 8.6 The loss path: `DoDiedGameOver()`

```
 1. userAborted = false
 2. InitDiedGameOver()
 3. CopyRectMainToWork(&workSrcRect)     // grab the frozen play field...
 4. CopyRectMainToBack(&workSrcRect)     // ...into both buffers
 5. FlushEvents(everyEvent, 0)
 6. nextLoop = TickCount() + 2
 7. while (pagesStuck < 8):
 8.     HandlePages()
 9.     DrawPages()
10.     do:
11.         GetKeys(theKeys)
12.         if Command||Option||Shift||Control: pagesStuck = 8; userAborted = true
13.         if GetNextEvent(...) && (mouseDown || keyDown):
14.                                       pagesStuck = 8; userAborted = true
15.     while (TickCount() < nextLoop)
16.     nextLoop = TickCount() + 2
17. if (roomRgn != nil) DisposeRgn(roomRgn)
18. DisposeGWorld(pageSrcMap);      pageSrcMap    = nil
19. DisposeGWorld(pageMaskMap);     pageMaskMap   = nil
20. DisposeGWorld(gameOverSrcMap);  gameOverSrcMap = nil
21. playing = false
22. if (demoGoing):
23.     if (!userAborted) WaitForInputEvent(1)
24. else:
25.     if (!userAborted) WaitForInputEvent(10)
26.     TestHighScore()                    // return value discarded
27. RedrawSplashScreen()
```
— `GliderPRO/Sources/GameOver.c:443-506`

Differences from the win path that matter:

* The dying ending **keeps the last rendered play field as its
  backdrop** (steps 3-4), so the room, the scoreboard strip and any
  surviving glider sprite stay on screen while the pages flutter down.
  The win path throws the play field away and paints the starfield.
* `roomRgn` is never set to `nil` after `DisposeRgn`, so a second
  `DoDiedGameOver()` in the same session passes a dangling handle to
  the `if (roomRgn != nil)` test. In practice `InitDiedGameOver()` runs
  first and reassigns it, so this is latent.
* `TestHighScore()`'s return value is **ignored** and
  `RedrawSplashScreen()` runs unconditionally — the opposite of
  `DoGameOver()`, which suppresses the redraw when the high-score
  screen was shown.
* In a demo, `TestHighScore()` is skipped entirely and the wait is 1 s
  instead of 10 s.

#### 8.6.1 `InitDiedGameOver()`

`GliderPRO/Sources/GameOver.c:249-310`.

```
 1. GetGWorld(&wasCPort, &wasWorld)
 2. QSetRect(&pageSrcRect, 0, 0, 25, 32*8)   // 25 x 256
 3. CreateOffScreenGWorld(&gameOverSrcMap, &pageSrcRect, kPreferredDepth /*8*/)
 4. SetGWorld(gameOverSrcMap, nil);  LoadGraphic(kLettersPictID /*1988*/)
 5. QSetRect(&pageSrcRect, 0, 0, 32, 32*kPageFrames)   // 32 x 448
 6. CreateOffScreenGWorld(&pageSrcMap, &pageSrcRect, 8)
 7. SetGWorld(pageSrcMap, nil);  LoadGraphic(kPagesPictID /*1990*/)
 8. CreateOffScreenGWorld(&pageMaskMap, &pageSrcRect, 1)      // 1-bit!
 9. SetGWorld(pageMaskMap, nil);  LoadGraphic(kPagesMaskID /*1989*/)
10. for i in 0..13: pageSrc[i] = (0,0,32,32) offset by (0, 32*i)
11. for i in 0..7:
12.     pages[i].dest = (0,0,32,32)
13.     CenterRectInRect(&pages[i].dest, &thisMac.screen)
14.     QOffsetRect(&pages[i].dest, -thisMac.screen.left, -thisMac.screen.top)
15.     if (i < 4) QOffsetRect(&pages[i].dest, -kPageSpacing * (4 - i), 0)
16.     else       QOffsetRect(&pages[i].dest,  kPageSpacing * (i - 3), 0)
17.     QOffsetRect(&pages[i].dest,
18.                 (thisMac.screen.right - thisMac.screen.left) / -2,
19.                 (thisMac.screen.right - thisMac.screen.left) / -2)   // <-- BUG
20.     if (pages[i].dest.left % 2 == 1) QOffsetRect(&pages[i].dest, 1, 0)
21.     pages[i].was = pages[i].dest
22.     pages[i].frame = 0
23.     pages[i].counter = RandomInt(32)
24.     pages[i].stuck = false
25. for i in 0..7: lettersSrc[i] = (0,0,25,32) offset by (0, 32*i)
26. roomRgn = NewRgn();  RectRgn(roomRgn, &justRoomsRect)
27. pagesStuck = 0
28. stopPages = ((thisMac.screen.bottom - thisMac.screen.top) / 2) - 16
```

`wasCPort`/`wasWorld` are captured at step 1 and **never restored** —
`InitDiedGameOver()` leaves the current GWorld set to `pageMaskMap`.
It works only because `DrawPages()` uses explicit `CopyBits`/`CopyMask`
destinations rather than the current port, and because `Play.c:551`
restores the port after the ending returns.

**The vertical-offset bug (line 19):** the second argument of the final
`QOffsetRect` uses `(screen.right - screen.left) / -2` — the screen
**width** — where the **height** was clearly intended. On a 640 x 480
screen the pages are therefore lifted 320 px instead of 240, starting
at `top = -96` rather than `top = -16`. On a 512 x 384 screen the
horizontal and vertical shifts are 256 each; on 1024 x 768 they are
512 each (so the pages start at `top = 368 - 512 = -144`). A Go port
that "fixes" this will change the length and the landing positions of
the whole animation. **Keep the bug.**

Worked example, 640 x 480 screen at origin (0,0), so
`thisMac.screen = (0,0,480,640)`:

| i | after `CenterRectInRect` | + spacing | + (-320,-320) | after even-align | initial `(left, top)` |
|---|---|---|---|---|---|
| 0 | left 304, top 224 | left 144 | left -176, top -96 | -176 even | (-176, -96) |
| 1 | 304 | 184 | -136 | even | (-136, -96) |
| 2 | 304 | 224 | -96 | even | (-96, -96) |
| 3 | 304 | 264 | -56 | even | (-56, -96) |
| 4 | 304 | 344 | 24 | even | (24, -96) |
| 5 | 304 | 384 | 64 | even | (64, -96) |
| 6 | 304 | 424 | 104 | even | (104, -96) |
| 7 | 304 | 464 | 144 | even | (144, -96) |

`stopPages` = 480/2 - 16 = **224**.

Note the pitch: -176, -136, -96, -56, **[gap of 80]**, 24, 64, 104,
144. `kPageSpacing` is 40 but the i<4 / i>=4 split skips the
`0 * kPageSpacing` slot, producing a **double-width gap between the
4th and 5th letter** — exactly the space in "GAME OVER".

The `left % 2 == 1` even-alignment nudge exists because the 1-bit
`CopyMask` path was faster on 68k when the source and destination were
word-aligned. Note that `%` on a negative `left` yields `0` or `-1` in
C, never `+1`, so **the nudge never fires for pages that start at a
negative x** — another quirk to preserve. (In the 640x480 table above
every value is even anyway.)

#### 8.6.2 `HandlePages()` — the flutter state machine

`GliderPRO/Sources/GameOver.c:316-394`. Run once per frame for all 8
pages.

```
for i in 0..7:
  if ((pages[i].dest.bottom + RandomInt(8)) > stopPages):        // LANDED
      pages[i].frame = 0
      if (!pages[i].stuck):
          pages[i].dest.right = pages[i].dest.left + 25          // 32 -> 25 wide
          pages[i].stuck = true
          pagesStuck++
  else if (pages[i].frame == 0):                                 // HOVER
      pages[i].counter--
      if (pages[i].counter <= 0) pages[i].frame = 1
      // no movement
  else if (pages[i].frame == 7):                                 // TUMBLE
      pages[i].counter--
      if (pages[i].counter <= 0):
          pages[i].frame = 8
          PlayPrioritySound(RandomInt(2)==0 ? kPaper3Sound : kPaper4Sound,
                            kPapersPriority)
      else:
          QOffsetRect(&pages[i].dest, 10, 10)
  else:                                                          // ADVANCE
      pages[i].frame++
      switch (pages[i].frame):
          case  5: offset (6,6)
          case  6: offset (8,8)
          case  7: offset (8,8); counter = RandomInt(4) + 4
          case  8: offset (8,8)                 // UNREACHABLE (see below)
          case  9: offset (8,8)
          case 10: offset (6,6)
          case 14: offset (8,0); frame = 0; counter = RandomInt(8) + 8
                   PlayPrioritySound(RandomInt(2)==0 ? kPaper1Sound
                                                     : kPaper2Sound,
                                     kPapersPriority)
          // cases 2,3,4,11,12,13: no offset
```

Per-frame displacement table (the value applied when `frame` *becomes* N):

| new `frame` | dx | dy | side effect |
|---|---|---|---|
| 1 | 0 | 0 | (set by the HOVER branch, not the switch) |
| 2 | 0 | 0 | |
| 3 | 0 | 0 | |
| 4 | 0 | 0 | |
| 5 | +6 | +6 | |
| 6 | +8 | +8 | |
| 7 | +8 | +8 | `counter = RandomInt(4)+4` (4..7) |
| 7 (held) | +10 | +10 | once per held frame except the last |
| 8 | 0 | 0 | plays `kPaper3Sound`/`kPaper4Sound`; the `case 8:` offset is dead |
| 9 | +8 | +8 | |
| 10 | +6 | +6 | |
| 11 | 0 | 0 | |
| 12 | 0 | 0 | |
| 13 | 0 | 0 | |
| 14 | +8 | **0** | `frame = 0`, `counter = RandomInt(8)+8` (8..15), plays `kPaper1Sound`/`kPaper2Sound` |

**`case 8:` is dead code.** The ADVANCE branch is only entered when
`frame != 0 && frame != 7`, i.e. `frame` in
`{1,2,3,4,5,6,8,9,10,11,12,13}`, so `frame++` can only produce
`{2,3,4,5,6,7,9,10,11,12,13,14}`. 8 is never produced there; frame 8 is
only ever reached from the TUMBLE branch, which applies no offset.

**`frame` never reaches 14 as an array index.** Case 14 resets it to 0
in the same statement, so `pageSrc[]` (14 entries, 0..13) is never
over-indexed.

Per complete flutter cycle (frame 0 -> ... -> 14 -> 0):

* `dx = 6+8+8+8+6+8 + 10*(counter7 - 1)` = `44 + 10*(c-1)`
* `dy = 6+8+8+8+6 + 10*(counter7 - 1)` = `36 + 10*(c-1)`
* so **`dx - dy = 8` per completed cycle** — the only source of
  horizontal drift relative to the vertical fall.
* frames consumed = `counter0 + 6 + counter7 + 6`, where
  `counter0` in 8..15 (after the first cycle; `RandomInt(32)` = 0..31
  on the very first) and `counter7` in 4..7 -> **24..34 frames/cycle**.

Because every page falls the same vertical distance (from `top = -96`
to the landing line) and the horizontal excess is a fixed 8 px per
completed cycle, the pages land almost exactly on their initial
40-px pitch — but a page that happens to complete one more cycle than
its neighbour lands 8-10 px further right. A Monte-Carlo run of the
exact state machine (uniform `RandomInt`) on a 640 x 480 screen:

```
seed 0  frames 136  lefts [138, 178, 218, 258, 338, 378, 418, 458]  tops all 194
seed 1  frames 128  lefts [138, 168, 218, 258, 340, 376, 418, 458]  tops 192..196
seed 2  frames 120  lefts [138, 178, 218, 258, 338, 378, 418, 458]  tops all 194
seed 3  frames 130  lefts [138, 178, 218, 258, 338, 378, 418, 458]  tops all 194
```

So: **the animation lasts about 120-140 frames = 4-4.7 s at 30 fps**,
the letters normally come to rest at x = 138, 178, 218, 258, [gap],
338, 378, 418, 458 with y ~= 194, and occasionally one letter is a few
pixels out of line. The landing y is pinned by the test
`top + 32 + RandomInt(8) > 224` -> `top > 192 - RandomInt(8)`, giving
`top` in 186..194 depending on where the last 6/8/10-px step lands it.

#### 8.6.3 `DrawPages()`

`GliderPRO/Sources/GameOver.c:401-435`.

```
for i in 0..7:
    if (pages[i].stuck):
        CopyBits(gameOverSrcMap, workSrcMap,
                 &lettersSrc[i], &pages[i].dest, srcCopy, roomRgn)
    else:
        CopyMask(pageSrcMap, pageMaskMap, workSrcMap,
                 &pageSrc[pages[i].frame], &pageSrc[pages[i].frame],
                 &pages[i].dest)
    QUnionSimilarRect(&pages[i].dest, &pages[i].was, &pages[i].was)
    AddRectToWorkRects(&pages[i].was)
    AddRectToBackRects(&pages[i].dest)
    CopyRectsQD()                    // <-- inside the loop!
    numWork2Main = 0
    numBack2Work = 0
    pages[i].was = pages[i].dest
```

* **`CopyRectsQD()` is called once per page, i.e. 8 times per frame.**
  Every other animation in the game calls it once per frame after
  accumulating all the dirty rects. This is almost certainly an
  oversight; it makes the loss ending 8x more blit-syscall-heavy than
  necessary but is visually indistinguishable. A port is free to hoist
  it out of the loop.
* The stuck letters are blitted with plain `CopyBits ... srcCopy` and
  **no mask**, clipped to `roomRgn` (= `justRoomsRect`). The letter
  cell's own 1-px `#777777` (index 250) border therefore shows around
  each landed letter.
* The fluttering pages are blitted through the 1-bit mask
  `pageMaskMap`, so only the paper shape is drawn.
* `QUnionSimilarRect(a, b, c)` writes the bounding union of `a` and `b`
  into `c` (`GliderPRO/Sources/RectUtils.c`). Here `c` aliases `b`
  (`was`), so `was` becomes "old position union new position" — the
  rect that has to be restored from the back buffer.

#### 8.6.4 PICT 1988: the letters really are G-A-M-E-O-V-E-R (verified)

`lettersSrc[i]` is the i-th 25 x 32 cell of a 25 x 256 sheet. I decoded
PICT 1988 pixel-by-pixel (PICT v2, 5900 bytes, `picFrame = (0,0,256,25)`,
8-bit `PackBitsRect` opcode `0x0098`, `rowBytes = 26`) and rendered each
cell as ASCII art. The eight cells are, in order:

| cell `i` | source rect | glyph |
|---|---|---|
| 0 | `(0,0,32,25)` | **G** |
| 1 | `(32,0,64,25)` | **A** |
| 2 | `(64,0,96,25)` | **M** |
| 3 | `(96,0,128,25)` | **E** |
| 4 | `(128,0,160,25)` | **O** |
| 5 | `(160,0,192,25)` | **V** |
| 6 | `(192,0,224,25)` | **E** |
| 7 | `(224,0,256,25)` | **R** |

Combined with the 80-px gap between page 3 and page 4 (8.6.1), the
final image reads `GAME OVER`.

Sheet structure, measured:

* Columns 0 and 24 are index **250** (`#777777`) for all 256 rows —
  a 1-px grey frame down both sides.
* Rows 0, 31, 32, 63, 64, 95, 96, 127, 128, 159, 160, 191, 192, 223,
  224 and 255 are *uniformly* index 250 — i.e. each 32-row cell has a
  solid grey top row and a solid grey bottom row.
* Palette histogram over the whole 25 x 256 sheet:

| index | RGB | pixel count | role |
|---|---|---|---|
| 0 | `#FFFFFF` | 2192 | background |
| 36 | `#CCFFFF` | 1157 | pale-cyan dither on odd scanlines |
| 126 | `#6666FF` | 1147 | letter body fill |
| 250 | `#777777` | 880 | cell borders |
| 120 | `#6699FF` | 477 | letter highlight |
| 133 | `#6633CC` | 370 | letter shadow |
| 6 | `#FFCCFF` | 101 | anti-alias |
| 42 | `#CCCCFF` | 54 | anti-alias |
| 255 | `#000000` | 22 | anti-alias |

* Bounding box of the letter body (indices 120/126/133) per cell:

| cell | x range | y range (within cell) |
|---|---|---|
| 0 (G) | 1..22 | 5..26 |
| 1 (A) | 4..22 | 9..25 |
| 2 (M) | 1..23 | 8..24 |
| 3 (E) | 3..21 | 9..25 |
| 4 (O) | 2..22 | 4..25 |
| 5 (V) | 2..22 | 10..26 |
| 6 (E) | 3..21 | 8..24 |
| 7 (R) | 3..22 | 7..23 |

(The first ASCII rendering I produced was unreadable because I treated
`index != 0` as ink, which lit the 250 borders and the 36 dither.
Restricting ink to indices 120/126/133 and sampling even scanlines only
made all eight glyphs legible.)

#### 8.6.5 Ending artwork inventory

Every PICT dimension and version below was read out of the resource
dump, not guessed:

| PICT | size | version | bytes | GWorld | depth | created at | disposed at |
|---|---|---|---|---|---|---|---|
| 1988 | 25 x 256 | v2 | 5900 | `gameOverSrcMap` | 8 | `GameOver.c:263-265` | `GameOver.c:497` |
| 1990 | 32 x 448 | v2 | 10052 | `pageSrcMap` | 8 | `GameOver.c:267-270` | `GameOver.c:491` |
| 1989 | 32 x 448 | **v1** | 1851 | `pageMaskMap` | **1** | `GameOver.c:272-274` | `GameOver.c:494` |
| 1021 | 640 x 460 | v2 | 51520 | (drawn straight into `workSrcMap`) | 8 | — | — |
| 1019 | 96 x 44 | v2 | 4064 | `angelSrcMap` | 8 | `StructuresInit2.c:128-130` | never |
| 1020 | 96 x 44 | **v1** | 457 | `angelMaskMap` | **1** | `StructuresInit2.c:132-134` | never |
| 4002 | 88 x 378 | v2 | 15432 | `bonusSrcMap` | 8 | `StructuresInit.c:350-353` | never |
| 5002 | 88 x 378 | **v1** | 4500 | `bonusMaskMap` | **1** | `StructuresInit.c:350-357` | never |

The 1-bit masks are stored as **version-1 PICTs** (version word
`0x1101`, 1-byte opcodes) while all the colour art is version 2
(`0x0011` + `0x02FF` + `0x0C00`). A Go PICT reader must handle both, or
special-case the masks. (My decoder handles v2 only; PICT 5002/1989/
1020 dimensions above come from the `picFrame` in the 10-byte header,
which is identical in both versions.)

### 8.7 Who calls `TestHighScore()`

| site | context | return used? | `RedrawSplashScreen()` after? |
|---|---|---|---|
| `GliderPRO/Sources/GameOver.c:67` | win ending | yes — suppresses the redraw | only when it returns false |
| `GliderPRO/Sources/GameOver.c:503` | loss ending, non-demo only | no | always |

Those are the only two call sites in the program. See section 7.5 for
the function itself. Note that the win path calls it **even in a demo**
(`DoGameOver()` has no `demoGoing` guard, unlike `DoDiedGameOver()`),
so an attract-mode demo that happened to finish the demo house would
pop up the "enter your name" dialog. In practice the recorded demo
input runs out and the glider dies, taking the guarded path.

`TestHighScore()` also returns `false` immediately when
`resumedSavedGame` is true (`GliderPRO/Sources/HighScores.c:380-381`).
`resumedSavedGame` is set at only three places, all in `DoGameMenu`
(`GliderPRO/Sources/Menu.c:307, 313, 318`): `false` for New Game and
Two Player, `true` for Open Saved Game. Since `OpenSavedGame()` is
stubbed to `return false` (`GliderPRO/Sources/SavedGames.c:167-169`),
choosing "Open Saved Game" leaves `resumedSavedGame == true` without
starting a game, and **any demo game started afterwards
(`DoDemoGame()` -> `NewGame(kNewGameMode)`,
`GliderPRO/Sources/Play.c:282-302`, which does not reset the flag) also
suppresses the high-score test.** Choosing New Game or Two Player
clears it again.

### 8.8 Ways a game ends *without* a high-score test

| exit | code | score kept? |
|---|---|---|
| Cmd-Q during play | `GliderPRO/Sources/Input.c:53-64`: `playing = false; paused = false;` then `if (QuerySaveGame()) SaveGame2();` | **No.** `gameOver` is never set, so `PlayGame()` returns without running the countdown block, and `TestHighScore()` is never reached. The score is silently discarded. `SaveGame2()` is a stub (section 9), so "Save" does nothing either. |
| Arcade build: any control key during a demo | `GliderPRO/Sources/Input.c` (`BUILD_ARCADE_VERSION` branch of `GetDemoInput`): `playing = false; paused = false;` | No |
| `InterfaceInit.c:130` | `playing = false` at startup | n/a |
| Quitting the app | `quitting` terminates `PlayGame()`'s `while` | No |

`QuerySaveGame()` is `Alert(1041, nil)` returning true for item 1
(`GliderPRO/Sources/Input.c:383-397`); `CenterAlert(kSaveGameAlert)` is
commented out, so the alert appears wherever ALRT 1041's rect puts it.

**This is a real fidelity decision for a port:** in the shipped source,
quitting mid-game throws the score away with no warning that it is
being thrown away (the dialog says "save your game?", and answering yes
does nothing).

### 8.9 The two-player win bug

`Interactions.c:933-951` (the `kStar` arm) calls `FlagGameOver()` the
moment `numStarsRemaining` hits 0, regardless of `mortals`.

In a two-player game `mortals` starts at `2 * kInitialGliders` = 4 and
`OffAMortal()` sets `onePlayerLeft` when one player is exhausted
(section 3). If the surviving player then collects the last star while
`mortals` has already reached `-1`, the dispatch at
`GliderPRO/Sources/Play.c:546` sees `mortals < 0` and runs
**`DoDiedGameOver()`** — the fluttering "GAME OVER" pages — on what is
actually a *victory*. The high-score entry is still recorded (the loss
path calls `TestHighScore()` too), but the winner gets the losing
animation and never sees the trailer text or the angel.

A port should decide explicitly whether to reproduce this. Reproducing
it costs nothing; "fixing" it changes observable behaviour.

### 8.10 Sound cues in the endings

| event | sound constant | id | priority constant | value |
|---|---|---|---|---|
| each star spawn in the win animation (every 16 frames) | `kMysticSound` | 35 | `kMysticPriority` | 202 |
| a page finishes its tumble (frame 7 -> 8) | `kPaper3Sound` or `kPaper4Sound` | 50 / 51 | `kPapersPriority` | 807 |
| a page completes a flutter cycle (frame 14) | `kPaper1Sound` or `kPaper2Sound` | 48 / 49 | `kPapersPriority` | 807 |

Both paper pairs are chosen with `if (RandomInt(2) == 0)`.
`GliderPRO/Sources/GameOver.c:158, 347, 349, 386, 388`. See section 10
for the resource-ID mapping.

---

## 9. Save and resume — the score/inventory snapshot (dead code)

Everything in this section is **unreachable in the shipped source**. It
is documented because (a) the `savedGame` blob occupies 40 bytes in the
middle of every house file and a port's house reader/writer must handle
it, (b) 20 of the 22 shipped houses carry non-zero bytes there and two
of them have `hasGame == 1`, and (c) it defines the canonical
"everything the scoring/inventory subsystem needs to persist" tuple.

### 9.1 `gameType` — the in-house snapshot, byte-exact

`GliderPRO/Headers/GliderStructs.h:116-134`, embedded in `houseType`
at offset **820** (`GliderPRO/Headers/GliderStructs.h:192`).
`#pragma options align=mac68k` (2-byte alignment); every field is
naturally aligned, so there is **no padding**: 40 bytes exactly.

| field | C type | size | offset in `gameType` | absolute offset in the house file | meaning |
|---|---|---|---|---|---|
| `version` | `short` | 2 | 0 | **820** | should be `kSavedGameVersion` = `0x0200` |
| `wasStarsLeft` | `short` | 2 | 2 | 822 | `numStarsRemaining` |
| `timeStamp` | `long` | 4 | 4 | 824 | copy of `houseType.timeStamp` (see 9.5) |
| `where` | `Point` | 4 | 8 | 828 | glider position; **`v` at 828, `h` at 830** (Mac `Point` is `{short v; short h;}`) |
| `score` | `long` | 4 | 12 | 832 | `theScore` |
| `unusedLong` | `long` | 4 | 16 | 836 | written as `0L` |
| `unusedLong2` | `long` | 4 | 20 | 840 | written as `0L` |
| `energy` | `short` | 2 | 24 | 844 | `batteryTotal` (negative = helium; see section 4) |
| `bands` | `short` | 2 | 26 | 846 | `bandsTotal` |
| `roomNumber` | `short` | 2 | 28 | 848 | `thisRoomNumber` |
| `gliderState` | `short` | 2 | 30 | 850 | `theGlider.mode` |
| `numGliders` | `short` | 2 | 32 | 852 | `mortals` |
| `foil` | `short` | 2 | 34 | 854 | `foilTotal` |
| `unusedShort` | `short` | 2 | 36 | 856 | written as `0` |
| `facing` | `Boolean` | 1 | 38 | 858 | `theGlider.facing` |
| `showFoil` | `Boolean` | 1 | 39 | 859 | `showFoil` |
| | | **40** | | ends at 859 | |

Followed immediately by `hasGame` (`Boolean`) at **860** and
`unusedBoolean` at **861** — see section 7.2 for the whole-house map.

Note what is **not** saved: `otherPlayerEscaped`, `onePlayerLeft`,
`playerSuicide`, `numBands`/`bands[]`, `doRollScore`/`scoreRollAmount`,
`gameFrame`, and the second glider. `SaveGame()` refuses two-player
games outright (`GliderPRO/Sources/SavedGames.c:309-310`).

### 9.2 `SaveGame(Boolean doSave)` — the in-house writer

`GliderPRO/Sources/SavedGames.c:303-351`. Referenced from **one**
place, `GliderPRO/Sources/Menu.c:458`, which is **commented out**
(`//	SaveGame(false);` in the editor's `iSave` handler).

```
 1. if (twoPlayerGame) return                                  // 2P is never saved
 2. wasState = HGetState(thisHouse);  HLock(thisHouse)
 3. thisHousePtr = *thisHouse
 4. if (doSave):
 5.     savedGame.version      = kSavedGameVersion   // 0x0200
 6.     savedGame.wasStarsLeft = numStarsRemaining
 7.     GetDateTime(&stamp);  savedGame.timeStamp = (long)stamp
 8.     savedGame.where.h      = theGlider.dest.left
 9.     savedGame.where.v      = theGlider.dest.top
10.     savedGame.score        = theScore
11.     savedGame.unusedLong   = 0L
12.     savedGame.unusedLong2  = 0L
13.     savedGame.energy       = batteryTotal
14.     savedGame.bands        = bandsTotal
15.     savedGame.roomNumber   = thisRoomNumber
16.     savedGame.gliderState  = theGlider.mode
17.     savedGame.numGliders   = mortals
18.     savedGame.foil         = foilTotal
19.     savedGame.unusedShort  = 0
20.     savedGame.facing       = theGlider.facing
21.     savedGame.showFoil     = showFoil
22.     hasGame = true
23. else:
24.     hasGame = false
25. HSetState(thisHouse, wasState)
26. if (doSave):
27.     if (!WriteHouse(theMode == kEditMode))
28.         YellowAlert(kYellowFailedWrite, 0)
```

**Note the contradiction at step 7.** The comment in
`SaveGame2()`'s (commented-out) body sets
`savedGame->timeStamp = thisHousePtr->timeStamp` — the *house* stamp,
which is what `OpenSavedGame()` then validates against
(`GliderPRO/Sources/SavedGames.c:235`). But `SaveGame()` at
`GliderPRO/Sources/SavedGames.c:320-321` stores a **fresh
`GetDateTime()`** instead. The two writers disagree; if the in-house
blob had ever been validated the same way, it would always have failed.
Nothing validates it in this build.

### 9.3 The three stubs

| function | line | state |
|---|---|---|
| `SaveGame2()` | `GliderPRO/Sources/SavedGames.c:29-148` | body is entirely inside `/* */` (open at `:33`, close at `:147`); the only live statement is the comment `// Add NavServices later.` |
| `OpenSavedGame()` | `GliderPRO/Sources/SavedGames.c:167-295` | first live line is `return false;		// TEMP fix this iwth NavServices`; body commented `:170-294` |
| `SaveGame(Boolean)` | `GliderPRO/Sources/SavedGames.c:303-351` | fully implemented, but its one call site is commented out |

`SaveGame2()` is what Cmd-S and the Cmd-Q "save?" prompt call
(`GliderPRO/Sources/Input.c:62, 67`), so **both of the user-visible
save affordances are no-ops**.

### 9.4 The abandoned side-car save format (`'gliG'`)

From the commented-out `SaveGame2()`/`OpenSavedGame()` bodies. This is
the format a Go port would have to invent anyway, so it is specified
here as the author's intent.

File type `'gliG'`, creator `'ozm5'`
(`GliderPRO/Sources/SavedGames.c:122` in the commented body).
Default file name: first word of `thisHouseName` (via
`GetFirstWordOfString`), clamped to 23 characters, then
`PasStringConcat` with `"\p Game"`. Presented through
`StandardPutFile("\pSave Game As:", ...)`; opened through
`StandardGetFile` filtered on `theList[0] = 'gliG'`.

`game2Type` (`GliderPRO/Headers/GliderStructs.h:144-164`):

| field | C type | size | offset | notes |
|---|---|---|---|---|
| `house` | `FSSpec` | 70 | 0 | `{short vRefNum; long parID; Str63 name}` = 2 + 4 + 64 |
| `version` | `short` | 2 | 70 | `kSavedGameVersion` = `0x0200` |
| `wasStarsLeft` | `short` | 2 | 72 | |
| `timeStamp` | `long` | 4 | 74 | **copy of `houseType.timeStamp`** — the anti-tamper check |
| `where` | `Point` | 4 | 78 | v @78, h @80 |
| `score` | `long` | 4 | 82 | |
| `unusedLong` | `long` | 4 | 86 | 0 |
| `unusedLong2` | `long` | 4 | 90 | 0 |
| `energy` | `short` | 2 | 94 | |
| `bands` | `short` | 2 | 96 | |
| `roomNumber` | `short` | 2 | 98 | |
| `gliderState` | `short` | 2 | 100 | |
| `numGliders` | `short` | 2 | 102 | |
| `foil` | `short` | 2 | 104 | |
| `nRooms` | `short` | 2 | 106 | replaces `gameType`'s `unusedShort` |
| `facing` | `Boolean` | 1 | 108 | |
| `showFoil` | `Boolean` | 1 | 109 | |
| `savedData[]` | `savedRoom[]` | 292 each | 110 | flexible array |

The header's declared "total = 114" in the source comment does not
match the field arithmetic, which sums to **110**. See Open questions.

`savedRoom` (`GliderPRO/Headers/GliderStructs.h:136-142`), 292 bytes:

| field | C type | size | offset |
|---|---|---|---|
| `unusedShort` | `short` | 2 | 0 |
| `unusedByte` | `Byte` | 1 | 2 |
| `visited` | `Boolean` | 1 | 3 |
| `objects[24]` | `objectType[kMaxRoomObs]` | 288 | 4 |

`kMaxRoomObs` = **24** (`GliderPRO/Headers/GliderDefines.h:250`);
`sizeof(objectType)` = **12** = `short what` + a 10-byte union of the
nine object payload structs (`GliderPRO/Headers/GliderStructs.h:93-105`).

So a saved game is `110 + 292 * nRooms` bytes (or `114 + 292 * nRooms`
if the author's arithmetic is right), written with a single `FSWrite`
and `SetEOF`.

Writer (commented body, `GliderPRO/Sources/SavedGames.c:89-118`): all
of `gameType`'s fields plus `nRooms`, then for each room `r`:
`unusedShort = 0`, `unusedByte = 0`, `visited = srcRoom->visited`, and a
straight copy of all 24 `objectType`s. Note it copies the **live**
`thisHouse` objects, i.e. the *current* on/off states, not the
`initial` states.

Reader (commented body, `GliderPRO/Sources/SavedGames.c:224-283`) — four
validation gates, each with its own alert, all rejecting the file:

| gate | test | failure |
|---|---|---|
| 1 | `EqualString(savedGame->house.name, thisHouseName, true, true)` | `SavedGameMismatchError(name)` -> `Alert(1044)` with `ParamText(gameName, thisHouseName, "", "")` |
| 2 | `thisHousePtr->timeStamp == savedGame->timeStamp` | `YellowAlert(kYellowSavedTimeWrong, 0)` |
| 3 | `savedGame->version == kSavedGameVersion` | `YellowAlert(kYellowSavedVersWrong, kSavedGameVersion)` |
| 4 | `savedGame->nRooms == thisHousePtr->nRooms` | `YellowAlert(kYellowSavedRoomsWrong, savedGame->nRooms - thisHousePtr->nRooms)` |

On success it fills the module-global `smallGame` (a `gameType`) — but
**not** `smallGame.version` and **not** `smallGame.timeStamp`; it sets
`smallGame.unusedShort = 0` explicitly — and then copies `visited` plus
all 24 objects back into each live room.

Note that gate 2 makes a saved game invalid the moment the house is
re-saved in the editor with `fileDirty` set, because `WriteHouse` then
re-stamps `timeStamp` (7.13).

Two memory-management bugs in the dead writer, for the record: the
`if (!theReply.sfGood) return;` at
`GliderPRO/Sources/SavedGames.c:70-71` and the `sfReplacing` error
returns at `:78, :84` all leak the `savedGame` pointer allocated at
`:57`.

### 9.5 What the 22 shipped houses actually contain at offset 820

Parsed directly out of the BinHex data forks. `hG` = `hasGame` (byte
860), `uB` = `unusedBoolean` (byte 861):

| house | hG | uB | `version` | `wasStarsLeft` | `timeStamp` | `where` (v,h) | `score` | `energy` | `bands` | `roomNumber` | `gliderState` | `numGliders` | `foil` | `facing` | `showFoil` |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Art Museum | 0 | 255 | -2314 | -2305 | 4143380214 | (-2305, 0) | 16774902 | -1537 | 0 | 255 | -2314 | -2314 | -2305 | 42 | 42 |
| CD Demo House | 0 | 0 | 256 | 5 | 2875627985 | (59, 236) | 28200 | 46 | 6 | 40 | 0 | 4 | 3 | 1 | 1 |
| California or Bust! | 0 | 0 | 256 | 3 | 2883328148 | (76, 254) | 0 | 0 | 0 | 0 | 0 | 2 | 0 | 1 | 0 |
| Castle o' the Air | 0 | 30 | 256 | 2 | 2864988030 | (29, 267) | 13200 | 27 | 0 | 7 | 2 | 2 | 0 | 1 | 0 |
| Davis Station | 0 | 37 | 0 | 1 | 65535 | (225, -8848) | 65536 | 0 | 0 | 225 | -8512 | -21760 | 12 | 0 | 32 |
| Demo House | 0 | 0 | 256 | 1 | 2864387348 | (11, 59) | 3000 | 50 | 5 | 15 | 0 | 0 | 0 | 1 | 0 |
| Empty House | 0 | 0 | 256 | 0 | 2858260248 | (14, 85) | 0 | 0 | 0 | 0 | 0 | 2 | 0 | 1 | 0 |
| Fun House | 0 | 0 | 0 | 0 | 1 | (0, -32747) | -1423057163 | 30976 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| Grand Prix | 0 | 185 | 169 | 401 | 11076063 | (169, 480) | 11993488 | 183 | 480 | 185 | 401 | 185 | 403 | 1 | 221 |
| **ImagineHouse PRO II** | **1** | 0 | 256 | 3 | 2869500669 | (61, **-10**) | 5900 | **-150** | 23 | 45 | 0 | 5 | 8 | 1 | 1 |
| In The Mirror | 0 | 0 | 256 | 1 | 2882026898 | (45, 424) | 100 | 0 | 0 | 1 | 0 | 2 | 0 | 1 | 0 |
| Land of Illusion | 0 | 30 | 256 | 1 | 2874404759 | (205, 243) | 43800 | -600 | 5 | 134 | 0 | 9 | 0 | 0 | 0 |
| Leviathan | 0 | 0 | 256 | 3 | 2881651229 | (176, 422) | 55600 | 262 | 78 | 449 | 0 | 24 | 15 | 0 | 1 |
| Metropolis | 0 | 14 | 256 | 0 | 2875207095 | (17, 71) | 300 | 0 | 0 | 9 | 0 | 2 | 0 | 1 | 0 |
| Nemo's Market | 0 | 0 | 256 | 1 | 2881122394 | (17, 240) | 25600 | 0 | 0 | 202 | 0 | 6 | 0 | 1 | 0 |
| Rainbow's End | 0 | 0 | 256 | 2 | 2875270003 | (146, 29) | 46200 | 0 | 13 | 112 | 0 | 10 | 0 | 1 | 0 |
| Sampler | 0 | 2 | 1 | 8 | 0 | (0, 0) | 0 | 0 | -1 | -1 | -1 | 1 | -1 | 204 | 204 |
| Slumberland | 0 | 0 | 256 | 5 | 2875627985 | (59, 236) | 28200 | 46 | 6 | 40 | 0 | 4 | 3 | 1 | 1 |
| SpacePods | 0 | 0 | 256 | 1 | 2881122394 | (17, 240) | 25600 | 0 | 0 | 202 | 0 | 6 | 0 | 1 | 0 |
| Teddy World | 0 | 0 | 256 | 0 | 2883419730 | (88, -2) | 2500 | 46 | 8 | 75 | 0 | 1 | 0 | 1 | 0 |
| The Asylum Pro | 0 | 0 | 256 | 1 | 2869853691 | (21, 234) | 3900 | 17 | 0 | 117 | 0 | 1 | 0 | 1 | 0 |
| **Titanic** | **1** | 0 | 256 | 1 | 2880300087 | (240, 85) | 4700 | 0 | 0 | 104 | 0 | 2 | 0 | 1 | 0 |

Facts a porter should take from this table:

1. **Exactly two houses have `hasGame == 1`**: `ImagineHouse PRO II` and
   `Titanic`. Every other house has `hasGame == 0` but *most* still
   carry a plausible-looking record.
2. **Every single `version` field that is not garbage reads `256` =
   `0x0100`, never `kSavedGameVersion` = `0x0200`.** These records were
   written by an earlier Glider PRO build. Nothing in the shipped source
   ever reads or validates them, so they are inert — but a Go port that
   decides to *honour* `hasGame` must reject `version != 0x0200`, or it
   will resume two houses with 1.0-era data.
3. **The `timeStamp` values in the blobs are genuine
   `GetDateTime()` results** (1994-07 through 1995-05 when converted with
   the Mac epoch offset 2082844800), and they do **not** equal the
   corresponding `houseType.timeStamp` (which is `& 0x7FFFFFFF`-masked;
   see the table in 7.13). Concretely: `Slumberland`'s house stamp
   decodes to 1995-08-13 but its `savedGame.timeStamp` is
   2875627985 = 1995-02-14. Whatever wrote these blobs used
   `GetDateTime()` directly, matching `SaveGame()`'s behaviour and not
   `SaveGame2()`'s.
4. **Five houses hold obvious uninitialised heap**, and the byte
   patterns prove it:
   * `Art Museum` bytes 820..865 are
     `f6 f6 f6 ff f6 f6 f6 f6 f6 ff 00 00 00 ff ...` — the classic
     classic-Mac-OS `0xF6` debug fill.
   * `Fun House` contains the Pascal string `08 "DiskCopy"` at offset
     836 — leaked from another application's heap block.
   * `Davis Station` contains `00 e1 dd 70` and `00 e1 de c0` — 24-bit
     Mac heap addresses.
   * `Grand Prix` contains a run of plausible `Point` pairs
     (`00 a9 01 91`, `00 a9 01 df`, `00 b7 01 90`, ...) — leaked object
     coordinates.
   * `Sampler` holds a mix of zeros and `ff`/`cc` fill.
   In each of those five the companion `unusedBoolean` (byte 861) is
   also non-zero (255, 0, 0, 37, 185, 2 respectively), which is a cheap
   heuristic for "this blob is garbage".
5. **Two pairs of unrelated houses have byte-identical `savedGame`
   records**: `CD Demo House` and `Slumberland`
   (`256, 5, 2875627985, (59,236), 28200, 46, 6, 40, 0, 4, 3, 1, 1`),
   and `Nemo's Market` and `SpacePods`
   (`256, 1, 2881122394, (17,240), 25600, 0, 0, 202, 0, 6, 0, 1, 0`).
   The houses themselves are completely different (72554 vs 134150 bytes
   and 44018 vs 140762 bytes, different banners), so this is not
   file duplication. See Open questions.
6. `ImagineHouse PRO II`'s record has **`where.h = -10`** — a negative
   x for the glider's saved position — and **`energy = -150`**, i.e. a
   full helium charge (`-kHeliumSupply`), which is consistent with the
   sign convention in section 4.
7. `unusedBoolean` (byte 861) is `0` in 16 houses and non-zero in 6
   (`Art Museum` 255, `Castle o' the Air` 30, `Davis Station` 37,
   `Grand Prix` 185, `Land of Illusion` 30, `Metropolis` 14,
   `Sampler` 2). It is never written by any code in the source tree, so
   a port must **preserve it byte-for-byte** on rewrite rather than
   normalising it to 0, if byte-identical round-tripping matters.

### 9.6 The resume path in `NewGame` / `InitGlider` (unreachable)

`kResumeGameMode` = **0**, `kNewGameMode` = **1**
(`GliderPRO/Headers/GliderDefines.h:616-617`). Reachable only from
`GliderPRO/Sources/Menu.c:323`, guarded by `OpenSavedGame()` which
always returns false.

`NewGame(short mode)` differences when `mode == kResumeGameMode`
(`GliderPRO/Sources/Play.c:74-215`):

| step | new game | resume |
|---|---|---|
| `SetObjectsToDefaults()` | called (`:102-103`, `if (mode != kResumeGameMode)`) | **skipped** — the objects keep the states `OpenSavedGame()` restored |
| room | `SetHouseToFirstRoom()` -> `ForceThisRoom(GetFirstRoomNumber())` (`:107-108`, `:370-374`) | `SetHouseToSavedRoom()` -> `ForceThisRoom(smallGame.roomNumber)` (`:105-106`, `:380-383`) |
| intro | `BringUpBanner(); DumpScreenOn(&justRoomsRect)` (`:169-172`) | `DisplayStarsRemaining(); DumpScreenOn(&justRoomsRect)` (`:174-177`) |

`InitGlider(gliderPtr thisGlider, short mode)`
(`GliderPRO/Sources/Play.c:306-365`), the part that owns scoring and
inventory state:

```
 1. WhereDoesGliderBegin(&thisGlider->dest, mode)
 2. if (mode == kResumeGameMode) numStarsRemaining = smallGame.wasStarsLeft
 3. else if (mode == kNewGameMode) numStarsRemaining = CountStarsInHouse()
 4. if (mode == kResumeGameMode):
 5.     theScore          = smallGame.score
 6.     mortals           = smallGame.numGliders
 7.     batteryTotal      = smallGame.energy
 8.     bandsTotal        = smallGame.bands
 9.     foilTotal         = smallGame.foil
10.     thisGlider->mode  = smallGame.gliderState
11.     thisGlider->facing= smallGame.facing
12.     showFoil          = smallGame.showFoil
13.     switch (thisGlider->mode):
14.         case kGliderBurning: FlagGliderBurning(thisGlider); break
15.         default:             FlagGliderNormal(thisGlider);  break
16. else:
17.     theScore     = 0L
18.     mortals      = kInitialGliders            // 2  -> 3 lives
19.     if (twoPlayerGame) mortals += kInitialGliders   // -> 4 -> 6 lives
20.     batteryTotal = 0
21.     bandsTotal   = 0
22.     foilTotal    = 0
23.     thisGlider->mode = kGliderNormal
24.     thisGlider->facing = kFaceRight
25.     thisGlider->src  = gliderSrc[0]
26.     thisGlider->mask = gliderSrc[0]
27.     showFoil = false
28. ...velocity/shadow/flag resets common to both...
```

Two things a porter needs from this even though the resume half is
dead:

* the **new-game** half (steps 17-27) is the authoritative "reset the
  scoring and inventory state" list, and it is called *per glider*, so
  in a two-player game `InitGlider(&theGlider, kNewGameMode)` runs and
  then `InitGlider(&theGlider2, kNewGameMode)` runs and **re-runs steps
  17-22**, zeroing `theScore`/`batteryTotal`/`bandsTotal`/`foilTotal` a
  second time and setting `mortals = 2 + 2 = 4` again from scratch
  (`GliderPRO/Sources/Play.c:120-124`). The result is the same because
  the values are absolute, not incremental — but a port that made them
  incremental would double them.
* `WhereDoesGliderBegin(Rect*, short mode)`
  (`GliderPRO/Sources/House.c:227-240`) leaves its local `initialPt`
  **uninitialised** if `mode` is neither 0 nor 1. There is no third
  mode, so this is latent.

### 9.7 `QueryResumeGame()` — the dialog nobody sees

`GliderPRO/Sources/Menu.c:710-758`. **Never called** — the only
reference is its own prototype at `GliderPRO/Sources/Menu.c:28`. It is
the only code in the program that *reads* `houseType.savedGame`:

```c
hadPoints  = thisHousePtr->savedGame.score;
hadGliders = thisHousePtr->savedGame.numGliders;
NumToString(hadPoints, scoreStr);
NumToString((long)hadGliders, glidStr);
if (hadGliders == 1)
    ParamText(glidStr, "\p",  scoreStr, "\p");
else
    ParamText(glidStr, "\ps", scoreStr, "\p");
```
— `GliderPRO/Sources/Menu.c:727-735`

so the dialog text is templated as `^0 glider^1 ... ^2 points`, with
`^1` supplying the plural "s". Dialog is `DLOG` **1025**
(`kResumeGameDial`), items `kSheWantsNewGame` = **1** and
`kSheWantsResumeGame` = **2** (`GliderPRO/Sources/Menu.c:18-19`).
`CenterDialog(kResumeGameDial)` is commented out
(`GliderPRO/Sources/Menu.c:738`), so like DLOG 1020 (7.11) it would
appear wherever the resource's own rect puts it. Filter is
`ResumeFilter` (`GliderPRO/Sources/Menu.c:668`).

**Therefore `houseType.savedGame` is write-only in this build.** No
live code path reads any of its 40 bytes.

### 9.8 The one live consequence: `HeyYourPissingAHighScore()`

`iOpenSavedGame` (`GliderPRO/Sources/Menu.c:317-324`) does:

```c
resumedSavedGame = true;
HeyYourPissingAHighScore();      // ALRT 1046, "So What?" - section 7.12
if (OpenSavedGame())             // always false
{ ... }
```

So the whole feature reduces to: show an alert, and permanently set
`resumedSavedGame = true` until the user picks New Game or Two Player,
which suppresses `TestHighScore()` for any game (including demos)
started in between (8.7).

---

## 10. Sounds owned by this subsystem

### 10.1 The sound engine, in one page

`GliderPRO/Sources/Sound.c`. Three `SndChannelPtr`s (`channel0`,
`channel1`, `channel2`), all opened with
`SndNewChannel(&channelN, sampledSynth, initNoInterp + initMono, callbackNUPP)`
(`GliderPRO/Sources/Sound.c:377-397`). So **at most three sounds are
audible at once**.

Constants (`GliderPRO/Sources/Sound.c:14-16`):

| constant | value | meaning |
|---|---|---|
| `kBaseBufferSoundID` | **1000** | resource ID of sound index 0 |
| `kMaxSounds` | **64** | size of `theSoundData[]`; only 63 are preloaded |
| `kNoSoundPlaying` | **-1** | sentinel in `soundPlayingN` |

`LoadBufferSounds()` (`GliderPRO/Sources/Sound.c:317-346`):

```
for (i = 0; i < kMaxSounds - 1; i++)          // 0 .. 62
    theSound = GetResource('snd ', i + kBaseBufferSoundID)   // 1000 .. 1062
    soundDataSize   = GetHandleSize(theSound) - 20L
    theSoundData[i] = NewPtr(soundDataSize)
    BlockMove(*theSound + 20L, theSoundData[i], soundDataSize)
    ReleaseResource(theSound)
theSoundData[kMaxSounds - 1] = nil            // slot 63 reserved
```

**So the resource ID of sound `n` is `n + 1000`, and slot 63 is a
scratch slot** used by `LoadTriggerSound()`
(`GliderPRO/Sources/Sound.c:264-305`) for the per-room `snd ` a room
trigger names — same `- 20L` skip.

Verified against the resource fork: `Glider PRO.r` contains
**70** `snd ` resources — exactly IDs **1000 through 1062 with no
gaps** (63 of them, matching the loop bound), plus 2000-2006 (the seven
music tracks `Refrain1.22`, `Refrain2.22`, `Refrain3.22`,
`Refrain4.22`, `Chorus.22`, `RefrainSparse1.22`, `RefrainSparse2.22`).

`PlayPrioritySound(short which, short priority)`
(`GliderPRO/Sources/Sound.c:40-88`) — the only entry point this
subsystem uses:

```
 1. if (failedSound || dontLoadSounds) return
 2. if (priority == kTriggerPriority (999) && any of priority0/1/2 == 999) return
 3. whosLowest = 0; lowestPriority = priority0
 4. if (priority1 < lowestPriority) { lowestPriority = priority1; whosLowest = 1 }
 5. if (priority2 < lowestPriority) { lowestPriority = priority2; whosLowest = 2 }
 6. if (priority >= lowestPriority) PlaySound<whosLowest>(which, priority)
```

Note the `>=`: a new sound of *equal* priority steals the channel.
Ties between channels resolve to the **lowest-numbered** channel
because steps 4-5 use strict `<`.

`PlaySoundN(soundID, priority)` (`GliderPRO/Sources/Sound.c:212-260`)
sets `priorityN = priority`, `soundPlayingN = soundID`, issues
`bufferCmd` with `param2 = (long)theSoundData[soundID]` via
`SndDoImmediate`, then queues a `callBackCmd` with
`param2 = SetCurrentA5()`. `CallBackN`
(`GliderPRO/Sources/Sound.c:180-208`) resets `priorityN = 0` and
`soundPlayingN = kNoSoundPlaying`, which is how a finished channel
becomes the "lowest priority" candidate again. It is guarded by
`if (isSoundOn)`; when sound is off, **`priorityN` is never updated at
all**, so `PlayPrioritySound` still runs and still returns without
audible effect.

### 10.2 `snd ` resource format, verified

Every one of the 19 sounds in this subsystem was parsed out of the
resource fork. All 19 are identical in structure:

* `format` = **1**, `numSynths` = 1 (synth id 5 = `sampledSynth`, with the
  per-synth initialisation longword = **160** = `0x000000A0`, not 0, in all 19)
* `numCmds` = 1, the single command being
  `cmd = 0x8051` (= `bufferCmd 0x0051` | `dataOffsetFlag 0x8000`),
  `param1 = 0`, `param2 = 20`
* so the `SoundHeader` starts at byte **20** — exactly the `- 20L`
  offset the loader uses
* `SoundHeader` (22 bytes): `samplePtr` 4 (0), `length` 4,
  `sampleRate` 4, `loopStart` 4, `loopEnd` 4, `encode` 1, `baseFrequency` 1
* `encode` = **0** (`stdSH`) for all 19 -> **8-bit unsigned linear PCM**,
  no compression
* `sampleRate` = **0x56EE8BA3** = **22254.5455 Hz** for all 19 (the
  classic Mac 22 kHz rate = 22254 + 6/11)
* `baseFrequency` = **60** (middle C) for all 19

| index | constant | priority constant | value | `snd ` ID | resource name | samples | bytes | duration @22254.5 Hz | `loopStart..loopEnd` |
|---|---|---|---|---|---|---|---|---|---|
| 3 | `kBeepsSound` | `kBeepsPriority` | 800 | 1003 | `Beeps` | 9680 | 9722 | 0.435 s | 9678..9679 |
| 4 | `kBuzzerSound` | `kBuzzerPriority` | 801 | 1004 | `Buzzer` | 8032 | 8074 | 0.361 s | 8030..8031 |
| 5 | `kDingSound` | `kDingPriority` | 802 | 1005 | `Ding` | 9728 | 9770 | 0.437 s | 9726..9727 |
| 6 | `kEnergizeSound` | `kEnergizePriority` | 803 | 1006 | `Energize` | 11904 | 11946 | 0.535 s | 11902..11903 |
| 8 | `kMicrowavedSound` | `kMicrowavedPriority` | 811 | 1008 | `Miked` | 10880 | 10922 | 0.489 s | 10878..10879 |
| 11 | `kCuckooSound` | `kCuckooPriority` | 805 | 1011 | `Cuckoo` | 5136 | 5178 | 0.231 s | 5134..5135 |
| 17 | `kScoreTikSound` | `kScoreTikPriority` | **101** | 1017 | `Score Tick` | 644 | 686 | 0.0289 s | 642..643 |
| 18 | `kThrustSound` | `kThrustPriority` | 300 | 1018 | `Thrust` | 5759 | 5801 | 0.259 s | **0..5758** |
| 19 | `kFizzleSound` | `kFizzlePriority` | 703 | 1019 | `Fizzle` | 2816 | 2858 | 0.127 s | 2814..2815 |
| 26 | `kShredSound` | `kShredPriority` | 903 | 1026 | `Shred` | 1934 | 1976 | 0.0869 s | **0..1933** |
| 35 | `kMysticSound` | `kMysticPriority` | **202** | 1035 | `Mystic` | 9696 | 9738 | 0.436 s | 9694..9695 |
| 48 | `kPaper1Sound` | `kPapersPriority` | 807 | 1048 | `Paper1` | 4432 | 4474 | 0.199 s | 4430..4431 |
| 49 | `kPaper2Sound` | `kPapersPriority` | 807 | 1049 | `Paper2` | 4096 | 4138 | 0.184 s | 4094..4095 |
| 50 | `kPaper3Sound` | `kPapersPriority` | 807 | 1050 | `Paper3` | 1648 | 1690 | 0.0741 s | 1646..1647 |
| 51 | `kPaper4Sound` | `kPapersPriority` | 807 | 1051 | `Paper4` | 1584 | 1626 | 0.0712 s | 1582..1583 |
| 52 | `kTypingSound` | `kTypingPriority` | 808 | 1052 | `Keystroke` | 2524 | 2566 | 0.113 s | 2522..2523 |
| 53 | `kCarriageSound` | `kCarriagePriority` | 809 | 1053 | `Carriage Return` | 5248 | 5290 | 0.236 s | 5246..5247 |
| 61 | `kBonusSound` | `kBonusPriority` | 812 | 1061 | `Bonus` | 5388 | 5430 | 0.242 s | 5386..5387 |
| 62 | `kHissSound` | `kHissPriority` | 311 | 1062 | `Hiss` | 2960 | 3002 | 0.133 s | **0..2959** |

Constants, sound indices: `GliderPRO/Headers/GliderDefines.h:58`
(`kBeepsSound`), `:59` (`kBuzzerSound`), `:60` (`kDingSound`), `:61`
(`kEnergizeSound`), `:63` (`kMicrowavedSound`), `:66` (`kCuckooSound`),
`:72` (`kScoreTikSound`), `:73` (`kThrustSound`), `:74`
(`kFizzleSound`), `:81` (`kShredSound`), `:90` (`kMysticSound`),
`:103`-`:106` (`kPaper1Sound`..`kPaper4Sound`), `:107`
(`kTypingSound`), `:108` (`kCarriageSound`), `:116` (`kBonusSound`),
`:117` (`kHissSound`).
Priorities: `GliderPRO/Headers/GliderDefines.h:121` (`kScoreTikPriority`), `:126`
(`kMysticPriority`), `:129` (`kThrustPriority`), `:140`
(`kHissPriority`), `:159` (`kFizzlePriority`), `:163`
(`kEnergizePriority`), `:167` (`kPapersPriority`), `:168`
(`kTypingPriority`), `:169` (`kCarriagePriority`), `:171`
(`kMicrowavedPriority`), `:172` (`kBonusPriority`), `:176`
(`kShredPriority`), `:160` (`kBeepsPriority`), `:161`
(`kBuzzerPriority`), `:162` (`kDingPriority`), `:165`
(`kCuckooPriority`). Note that `loopStart`/`loopEnd` are ignored because
`bufferCmd` plays once; the three sounds whose loop range spans the
whole sample (`Thrust`, `Shred`, `Hiss`) were authored as loops but are
played as one-shots.

### 10.3 Which event plays what, exhaustively

Every `PlayPrioritySound` call site that belongs to scoring, lives,
inventory, the scoreboard, high scores or game over:

| # | site | sound | priority | trigger |
|---|---|---|---|---|
| 1 | `GliderPRO/Sources/Scoreboard.c:91` | `kScoreTikSound` | 101 | one tick per frame while the score roll is running (section 2.9) |
| 2 | `GliderPRO/Sources/Interactions.c:770` | `kBeepsSound` (3) | `kBeepsPriority` 800 | red alarm clock (`kRedClock` `0x21`), +100 |
| 3 | `GliderPRO/Sources/Interactions.c:786` | `kBuzzerSound` (4) | `kBuzzerPriority` 801 | blue alarm clock (`kBlueClock` `0x22`), +300 |
| 4 | `GliderPRO/Sources/Interactions.c:802` | `kDingSound` (5) | `kDingPriority` 802 | yellow alarm clock (`kYellowClock` `0x23`), +500 |
| 5 | `GliderPRO/Sources/Interactions.c:818` | `kCuckooSound` (11) | `kCuckooPriority` 805 | cuckoo clock (`kCuckoo` `0x24`), +1000 |
| 6 | `GliderPRO/Sources/Interactions.c:835` | `kEnergizeSound` | 803 | picked up **paper** (`kPaper` `0x25`) — the extra life |
| 7 | `GliderPRO/Sources/Interactions.c:854` | `kEnergizeSound` | 803 | picked up a **battery** (`kBattery` `0x26`) |
| 8 | `GliderPRO/Sources/Interactions.c:876` | `kEnergizeSound` | 803 | picked up **rubber bands** (`kBands` `0x27`) |
| 9 | `GliderPRO/Sources/Interactions.c:904` | `kEnergizeSound` | 803 | picked up **foil** (`kFoil` `0x2A`) |
| 10 | `GliderPRO/Sources/Interactions.c:924` | `kBonusSound` | **812** | **invisible bonus** (`kInvisBonus` `0x2B`) — the only use of `Bonus` |
| 11 | `GliderPRO/Sources/Interactions.c:937` | `kEnergizeSound` | 803 | picked up a **star** (`kStar` `0x2C`), +5000 |
| 12 | `GliderPRO/Sources/Interactions.c:960` | `kEnergizeSound` | 803 | picked up **helium** (`kHelium` `0x2E`) |
| 13 | `GliderPRO/Sources/Interactions.c:1193` | `kMicrowavedSound` | 811 | the glider is microwaved (section 4.7) |
| 14 | `GliderPRO/Sources/Player.c:1287` | `kShredSound` | 903 | the glider is shredded (life lost) |
| 15 | `GliderPRO/Sources/Input.c:143` | `kFizzleSound` | 703 | pressed battery with `batteryTotal <= 0` |
| 16 | `GliderPRO/Sources/Input.c:150` | `kThrustSound` | 300 | battery thrust engaged |
| 17 | `GliderPRO/Sources/Input.c:168` | `kFizzleSound` | 703 | pressed helium with `batteryTotal >= 0` |
| 18 | `GliderPRO/Sources/Input.c:176` | `kHissSound` | 311 | helium engaged |
| 19 | `GliderPRO/Sources/GameOver.c:158` | `kMysticSound` | 202 | start of the **win** animation, once, in `DoGameOver()` |
| 20 | `GliderPRO/Sources/GameOver.c:347` | `kPaper3Sound` | 807 | loss animation, a page changed direction, `pageIs2`-odd branch |
| 21 | `GliderPRO/Sources/GameOver.c:349` | `kPaper4Sound` | 807 | ditto, even branch |
| 22 | `GliderPRO/Sources/GameOver.c:386` | `kPaper1Sound` | 807 | loss animation, a page came to rest, odd branch |
| 23 | `GliderPRO/Sources/GameOver.c:388` | `kPaper2Sound` | 807 | ditto, even branch |
| 24 | `GliderPRO/Sources/HighScores.c:464` | `kCarriageSound` | 809 | Return/Enter in the **name** dialog |
| 25 | `GliderPRO/Sources/HighScores.c:476` | `kTypingSound` | 808 | any other key in the **name** dialog |
| 26 | `GliderPRO/Sources/HighScores.c:512` | `kEnergizeSound` | 803 | the **name** dialog opens |
| 27 | `GliderPRO/Sources/HighScores.c:572` | `kCarriageSound` | 809 | Return/Enter in the **banner** dialog |
| 28 | `GliderPRO/Sources/HighScores.c:584` | `kTypingSound` | 808 | any other key in the **banner** dialog |
| 29 | `GliderPRO/Sources/HighScores.c:618` | `kEnergizeSound` | 803 | the **banner** dialog opens |

Consequences of the priority numbers worth porting exactly:

* `kScoreTikPriority` = **101** is the second-lowest priority in the
  whole program (only `kHitWallPriority` 100 is lower). Since the roll
  fires a tick *every frame*, any other event — a pickup at 803, a
  shred at 903 — instantly steals its channel, and the tick will not
  restart on the next frame if all three channels are busy with
  something louder. **The score roll is deliberately the first thing
  to be drowned out.**
* `kMysticPriority` = **202** for the win jingle is *low* (below
  `kThrustPriority` 300). Nothing else is playing at that moment, so it
  is audible — but a port that starts the animation while a pickup
  sound is still in flight would suppress it.
* All four `Paper` sounds share **`kPapersPriority` 807**, so during the
  loss animation the eight tumbling pages compete for the three
  channels on equal footing and the `>=` in step 6 means the newest
  page always wins.

### 10.4 Sound gating flags a port must reproduce

| flag | where set | effect |
|---|---|---|
| `dontLoadSounds` | `GliderPRO/Sources/Sound.c:33` (global); set from prefs | `LoadBufferSounds`, `PlayPrioritySound`, `PlaySoundN` all return immediately; nothing is ever loaded |
| `failedSound` | `GliderPRO/Sources/Sound.c:462, 470` on `LoadBufferSounds`/`OpenSoundChannels` error, after a `YellowAlert(kYellowFailedSound, theErr)` | same as above, but memory may be partly allocated |
| `isSoundOn` | user preference | `PlaySoundN` skips the `bufferCmd` **and** skips updating `priorityN`/`soundPlayingN` |
| `channelOpen` | `GliderPRO/Sources/Sound.c:381, 389, 396` | guards channel open/close |


---

## 11. Mac Toolbox surface used by this subsystem, and what Go must do instead

Every Toolbox call reachable from the files in scope, grouped, with the
concrete replacement obligation. "Fidelity" flags whether getting it
subtly wrong changes observable behaviour.

### 11.1 QuickDraw drawing

| Toolbox | used at | what it does here | Go replacement | fidelity |
|---|---|---|---|---|
| `SetGWorld` / `GetGWorld` | `Scoreboard.c` throughout, `HighScores.c:96, 279`, `GameOver.c:174, 234` | redirect all subsequent drawing to an offscreen buffer | an explicit `*image.Paletted` destination parameter; there is no ambient "current port" in Go and emulating one with a global is the single biggest source of porting bugs | high |
| `CopyBits(src, dst, &srcR, &dstR, srcCopy, nil)` | `Scoreboard.c:141, 173, ...`, `GameOver.c:401` | blit a rect | index-preserving `draw`-style copy on `image.Paletted`; **must not** go through RGBA or the 8-bit indices stop matching | high |
| `CopyMask(src, mask, dst, &srcR, &maskR, &dstR)` | `Scoreboard.c` (glider badge), `GameOver.c:404` | 1-bit-mask stencil blit | copy pixel only where mask index != 0; the mask PICTs are 1-bit, index 0 = white = "draw", so **the sense is inverted from the intuitive one** | high |
| `PaintRect(&r)` | `Scoreboard.c:148, 206, 244, 281, 316`, `HighScores.c:64` | fill with the current fore colour | `draw.Draw` with a uniform paletted colour | medium |
| `FrameRect(&r)` | `HighScores.c` (score box) | 1-px rectangle outline | four 1-px fills; note `FrameRect` draws *inside* the rect, i.e. `[left, right-1] x [top, bottom-1]` | medium |
| `ForeColor(long)` | `Scoreboard.c:152, 168, 183, 215, 219, 222, 250, 254, 257, 287, ...`, `HighScores.c` (row highlighting), `GameOver.c:106, 109, 112` | classic-QuickDraw colour constant, **not** a palette index | map through the 8-value table in section 8.5.1 (`whiteColor` 30, `blackColor` 33, `yellowColor` 69, `magentaColor` 137, `redColor` 205, `cyanColor` 273, `greenColor` 341, `blueColor` 409) | high |
| `Index2Color(index, &rgb)` + `RGBForeColor` | `Scoreboard.c` (`kRedOrangeColor8` 23 etc.) | fetch the RGB of a palette index and use it as the pen | just use the index directly; the palette is fixed (8.5.1) | high |
| `TextFont(applFont)` / `TextFace(bold)` / `TextSize(n)` | scoreboard: `StructuresInit.c:109-111` (title), `:119-121` (gliders), `:131-133` (points) — all **applFont / 12 / bold**, set once into the GWorld at creation, never again; high scores: `HighScores.c:133-135` (**14** bold, the banner), `:148-150` (**12** bold, the title), `:267-269` (**9** bold, the table rows); game over: `GameOver.c:103-105` (**12** bold) | select the application font | the bitmap font is **not** in the resource fork; it is the system's Geneva/Chicago. A port must ship a substitute; see Porting notes | high |
| `MoveTo(h, v)` + `DrawString(pStr)` | everywhere text is drawn | draw a Pascal string at a baseline | note the coordinate is the **baseline left**, not the top-left | high |
| `TextWidth(ptr, offset, len)` | `Scoreboard.c` (score right-alignment), `GameOver.c:207` | measure a string in the current font | must match the substitute font's metrics or the right-aligned score will drift | high |
| `DrawPicture(pict, &r)` | `GameOver.c` via `LoadScaledGraphic` | draw and scale a PICT | decode PICT offline into paletted bitmaps; only PICT 1995 is actually scaled (640x460 into 640x480) | medium |
| `ClipRect(&r)` / `NewRgn` / `RectRgn` / `DisposeRgn` | `GameOver.c:236-241, 275-282`, `Scoreboard.c` | clip drawing | an explicit clip rect intersected into every blit | medium |
| `QSetRect` / `QOffsetRect` / `QUnionSimilarRect` / `CenterRectInRect` / `ZeroRectCorner` | `RectUtils.c`, used pervasively | rect algebra | trivial; but `QUnionSimilarRect(a, b, c)` is component-wise min/max and **aliases** (`GameOver.c` passes `was` as both input and output) | medium |
| `NewGWorld(&gw, 8, &r, nil, nil, useTempMem)` | `Scoreboard.c:200-260`, `GameOver.c` | allocate an 8-bit offscreen buffer with **a nil colour table** | `image.NewPaletted(r, macSystemPalette)`; the nil cTable is why the palette is the fixed system one (8.5.1) | high |
| `LockPixels` / `GetGWorldPixMap` / `DisposeGWorld` | ditto | Memory-Manager plumbing | nothing; Go's GC handles it. But note the **leaks and use-after-dispose** documented in 8.6 (`roomRgn` not nil'd) do not reproduce in Go and must not be "faithfully" emulated | low |

### 11.2 Resource Manager

| Toolbox | used for | Go replacement |
|---|---|---|
| `GetPicture(id)` / `GetResource('PICT', id)` | PICTs 1019, 1020, 1021, 1988, 1989, 1990, 1994, 1995, 1996, 1997, 1998, 4002, 4006, 5002, 5006 | decode once at build time into embedded paletted PNGs or raw index arrays; **do not** write a PICT decoder at runtime |
| `GetResource('snd ', 1000+n)` | the 63 preloaded sounds | embed 8-bit unsigned PCM @22254.5455 Hz (10.2) |
| `GetIndString(str, listID, index)` | high-score and game-over strings | a `[]string` per `STR#` id; **1-based index** |
| `GetNewDialog` / `GetNewCWindow` (`DLOG`/`DITL`/`WIND`) | the two high-score entry dialogs (1020, 1021) and ALRT 1046 | hand-built UI; the DITL geometry is transcribed in 7.11 |
| `GetResource('clut', 128)` | never actually used for the play palette | hard-code the 256-entry table (8.5.1) |
| `HLock` / `HUnlock` / `HGetState` / `HSetState` / `GetHandleSize` | house handle pinning during `WriteHouse` and `SaveGame` | nothing; Go slices don't move |
| `ReleaseResource` | after each sound copy | nothing |

### 11.3 File Manager (high-score persistence)

| Toolbox | used at | Go replacement | fidelity |
|---|---|---|---|
| `FSpOpenDF` / `FSpCreate` / `FSpDelete` | `HighScores.c:642-740`, `HouseIO.c` | `os.OpenFile` etc. | low |
| `FSRead` / `FSWrite` / `FSClose` | `HighScores.c:742-830`, `HouseIO.c:448-505` | `io.ReadFull` / `Write` / `Close`; **the whole house handle is written in one call**, so the high-score block cannot be updated in isolation | high |
| `SetFPos(refNum, fsFromStart, 0)` | `HouseIO.c:455` | `Seek(0, io.SeekStart)` | high |
| `GetEOF` / `SetEOF` | `HouseIO.c:497`, `HighScores.c` | `Stat().Size()` / `Truncate` — **`SetEOF` after the write is what shrinks a house that got smaller** | high |
| `FSMakeFSSpec` | `HighScores.c:665-718` | a `filepath.Join` | low |
| `FindFolder(kOnSystemDisk, kPreferencesFolderType, ...)` | `HighScores.c:665-718` | `os.UserConfigDir()` or equivalent; the original path is `<boot>:System Folder:Preferences:G-PRO Scores {f-hook}:<houseName>` | low |
| `StandardPutFile` / `StandardGetFile` | dead `SaveGame2`/`OpenSavedGame` | n/a — the feature is dead | none |
| Mac type/creator (`'gliH'`/`'gliS'`/`'gliG'`, creator `'ozm5'`) | house, score side-car, saved game | no equivalent; use a magic number or extension | medium |
| Big-endian on disk | every multi-byte field in the house file, high-score table and `savedGame` | `encoding/binary.BigEndian` everywhere. This is the single most mechanical and most easily botched part of the port | high |
| Mac epoch 1904-01-01 | `highScoresType.timeStamps[10]`, `houseType.timeStamp`, `gameType.timeStamp` | `unix = mac - 2082844800` | high |

### 11.4 Sound Manager

| Toolbox | Go replacement | fidelity |
|---|---|---|
| `SndNewChannel(sampledSynth, initNoInterp + initMono, cb)` x3 | a 3-voice mixer; `initNoInterp` means **nearest-neighbour resampling**, which audibly aliases at 22254.5 Hz -> 44100/48000 Hz. Linear interpolation will sound *different* (cleaner) than the original | medium |
| `SndDoImmediate(bufferCmd)` | start a voice, replacing whatever it was playing | high |
| `SndDoCommand(callBackCmd, true)` | completion callback that frees the voice's priority slot | high — the priority arbitration in 10.1 depends on it |
| `SetCurrentA5()` / `SetA5()` | 68k global-data-register plumbing inside the callback | nothing |

### 11.5 Events, keys, time

| Toolbox | used at | Go replacement | fidelity |
|---|---|---|---|
| `TickCount()` | `Render.c:662-665` frame pacing, `HighScores.c` | 1 tick = 1/60.15 s exactly (`60.146...` Hz on real hardware; the code assumes 60). `kTicksPerFrame` = 2 -> nominal **30.07 fps** | high |
| `Delay(n, &junk)` | `GameOver.c` waits | `time.Sleep(n * 16.6ms)` | low |
| `GetKeys(km)` + `BitTst(&km, 0x?)` | `Input.c` | keyboard state polling, not events. Uses **raw Mac virtual key codes** bit-tested in a 128-bit map with the odd `BitTst` bit ordering (`BitTst(ptr, bit)` tests bit `7 - (bit & 7)` of byte `bit >> 3`) | high |
| `ModalDialog(filterUPP, &item)` | `HighScores.c:498-640` | a modal loop; the filters (`NameFilter`, `BannerFilter`) intercept keys to play the typing sounds and to enforce the 15/31-char limits | medium |
| `FlushEvents(everyEvent, 0)` | before alerts | discard queued input | low |
| `GetDateTime(&secs)` | `HighScores.c:407`, `HouseIO.c:475`, `SavedGames.c:320` | `time.Now()` converted to the Mac epoch | high |
| `NumToString(long, Str255)` | score/level formatting | `strconv.FormatInt`; note it produces **no thousands separators** | high |
| `ParamText(a, b, c, d)` | `Menu.c:731-734`, `SavedGames.c` | substitute `^0`..`^3` in the dialog's static text | low |
| `EqualString(a, b, true, true)` | `SavedGames.c:224`, `HighScores.c` | case-**in**sensitive, diacritical-**in**sensitive Mac Roman comparison. Go's `strings.EqualFold` is not the same function for Mac Roman accented characters | medium |
| `Alert(id, nil)` / `CenterAlert` (commented out) | ALRT 1041, 1044, 1046 | a message box; and note the missing centering means the original shows them at the resource's own coordinates | low |
| `HideCursor` / `InitCursor` | `Play.c:104`, `Input.c:384` | cursor visibility | low |
| `MaxMem(&growBytes)` | `Play.c:210` | records free memory into `freeBytes` for the debug display only | none |

### 11.6 Data representation

| Mac thing | detail | Go |
|---|---|---|
| `#pragma options align=mac68k` | 2-byte alignment; a `short` after a `Boolean` gets 1 pad byte | write explicit `binary.Read`/`Write` field by field, or a struct tag scheme; **do not** rely on Go's own layout |
| `Str15`/`Str27`/`Str31`/`Str63`/`Str255` | `unsigned char[16/28/32/64/256]`, length in byte 0 | a fixed `[N]byte` plus explicit length handling. **Padding bytes after the length are stale garbage** because `PasStringCopy` writes only `length+1` bytes — this is why the high-score name fields in shipped houses contain junk past the terminator (sections 0.4, 7.2 and 7.6) |
| `Point` | `struct { short v; short h; }` — **v first** | easy to get backwards; `gameType.where` is (v@828, h@830) |
| `Rect` | `struct { short top, left, bottom, right; }` | note the order is **top, left, bottom, right** |
| `Boolean` | 1 byte, non-zero = true | Mac code sometimes stores 255; compare `!= 0`, don't compare `== 1` |
| Mac Roman text | 0xA5 = bullet, 0xC4 = florin (the folder-name hook), 0xC9 = ellipsis, 0xD7 = lozenge | decode with a Mac Roman table for display; keep bytes as-is on disk |
| 8-bit indexed colour | the fixed system palette (8.5.1) | `color.Palette` of 256 entries, generated from the formula |
| C integer division/modulo | truncates toward zero; `%` of a negative is non-positive | matters at `GameOver.c:159-160` (`angelDest.left / 32`, then `% 5` -> negative indices, 8.5.3) and `GameOver.c:340` (`dest.left % 2`) |

---

## Open questions

Ordered roughly by how much a wrong guess would cost a port. Each is
something I could **not** resolve from the source and the shipped data
alone; none of them is speculation dressed as fact elsewhere in this
document.

### Behaviour I could not determine

1. **Was `houseIsReadOnly` ever true in a shipping build?**
   `IsFileReadOnly()` (`GliderPRO/Sources/HouseIO.c`) returns `false`
   unconditionally, so the entire
   `Preferences:G-PRO Scores {f}:<houseName>` side-car path
   (`ReadScoresFromDisk` / `WriteScoresToDisk`, `'gliS'`/`'ozm5'`, 292
   bytes) is dead and every high score goes into the house file at
   offset 528 (7.6, 7.13). Whether a later 1.0.x build restored the
   locked-volume check — which would move high scores for CD-ROM houses
   to the side-car — I cannot tell. **A port should implement both and
   pick based on write-permission on the house file**, because that is
   plainly the intent.

2. **Is the high-score screen supposed to be visible?**
   `DoHighScores()` draws a complete, pixel-perfect table into
   `workSrcMap` and then **never blits it** — both `DissBits` calls are
   commented out (7.9), and `RedrawSplashScreen()` ends with
   `CopyRectMainToWork` (main -> work) where `CopyRectWorkToMain` was
   obviously meant. So in this source drop the player literally never
   sees the high-score table. This is either a late regression or a
   deliberate temporary disable. A port must decide; I would **enable
   it** (fix the direction) because otherwise 350 lines of carefully
   laid-out drawing code have no purpose.

3. **Why is there a `case 250:` in `AddFlyingPoint`?**
   The 24x120 sheet is 15 cells of 24x8, grouped as five 3-frame
   animations (2.8.1). `AddFlyingPoint`
   (`GliderPRO/Sources/DynamicMaps.c:221-247`) maps them:

   | `points` | `start`..`stop` | reachable from |
   |---|---|---|
   | `default` | 0..2 | cuckoo clock (1000), and any `kInvisBonus` value not 100/250/300/500 |
   | 500 | 3..5 | yellow clock, `kInvisBonus` 500 |
   | 300 | 6..8 | blue clock, `kInvisBonus` 300 |
   | **250** | **9..11** | **`kInvisBonus` with `points == 250` only** |
   | 100 | 12..14 | red clock, `kInvisBonus` 100 |

   No literal 250 is passed anywhere; the only route in is an invisible
   bonus whose `points` field is 250, and the editor constrains that
   field to 100/300/500 (2.7). None of the 22 shipped houses has one.
   So frames 9-11 are dead in practice but *not* dead in principle —
   a port must still handle the case or a third-party house with a
   250-point bonus will show the wrong numeral. Note also that the
   **star (5000) calls `AddFlyingPoint` not at all** — there are only
   five call sites (`GliderPRO/Sources/Interactions.c:773, 789, 805,
   822, 925`), so a star awards its 5000 points with no flying numeral.

4. **What was `unusedShort` at house offset 2 for?**
   16 houses have 0; the rest have `Titanic` 2074 (0x081A),
   `Art Museum` 0x8A7E, `Fun House` 0x6674, `Davis Station` 0x00DE,
   `Grand Prix` 0x003C, `Castle o' the Air` and `Land of Illusion` 259,
   `In The Mirror` 147, `Metropolis` 196,
   `California or Bust!` 13107 (0x3333). No code reads or writes it.
   0x3333 and 0x6674 look like ASCII/filler; 259 = 0x0103 looks like a
   version. **A port must preserve it verbatim.**

5. **Why do two pairs of unrelated houses have byte-identical
   `savedGame` records?** `CD Demo House` == `Slumberland` and
   `Nemo's Market` == `SpacePods` (9.5), despite being different sizes,
   different room counts and different banners. Copy-paste of a house
   template followed by heavy editing is the likeliest explanation, but
   the shared `timeStamp`s (2875627985 and 2881122394) would then have
   to predate the divergence, which does not fit the other fields'
   plausibility. Unresolved.

6. **`kBatteryLow` is 17 but its comment says 25%.** The four
   low-supply thresholds are function-local `#define`s inside
   `HandleDynamicScoreboard()`
   (`GliderPRO/Sources/Scoreboard.c:74-77`), each commented `// 25%`:

   | constant | value | supply | true 25% | matches? |
   |---|---:|---:|---:|---|
   | `kFoilLow` | 2 | `kFoilSupply` 8 | 2 | yes |
   | `kBatteryLow` | **17** | `kBatterySupply` 50 | 12.5 | **no** — 17 is 1/3 of 50 |
   | `kHeliumLow` | **-38** | `-kHeliumSupply` -150 | -37.5 | yes (rounded away from zero) |
   | `kBandsLow` | 2 | `kBandsSupply` 8 | 2 | yes |

   So `kBatteryLow` is the outlier: it is one **third** of the battery
   supply, not one quarter, and the comment is stale. I documented the
   literal. If a port's low-battery blink starts at the wrong moment
   against a recording, this is why.

7. **Does the `game2Type` header occupy 110 or 114 bytes?**
   Field arithmetic gives **110**; the struct's own comment says
   "total = 114" (9.4). Only matters for the dead `'gliG'` side-car
   format, so a port inventing its own save format should ignore it —
   but anyone trying to *read* a `.gliG` file produced by a
   pre-release build needs to know which.

8. **`kRedOrangeColor8` = 23, comment says "actually, 18".**
   Index 23 in the system palette is `#FF6600`, which is red-orange, so
   23 is right and the comment is stale. Index 18 is `#FF9933`. I used
   23 (5.x); if a port's low-battery bar looks too orange, this is the
   knob.

9. **Was the scoreboard meant to survive `numNeighbors == 9`?**
   With nine neighbours the room view is wide enough that
   `AdjustScoreboardHeight()` pushes the board off the bottom of the
   window (5.7). No shipped house triggers it on a 640x480 screen. On a
   larger screen it would. The original probably never saw this.

10. **Why is `doRollScore` never cleared?**
    `RefreshPoints()` has a "snap the score" branch guarded by
    `if (!doRollScore)` that can never run because nothing ever sets
    `doRollScore = false` after init (2.9). Either dead defensive code
    or a lost feature ("instant score" preference).

11. **Is `Fun House` intentionally unwinnable?** It contains zero
    `kStar` objects, so `numStarsRemaining` starts at 0 and the win
    condition `numStarsRemaining <= 0` in `HandleRewards` never fires
    because that path requires collecting a star (3.x, 8.x). Possibly a
    joke house, possibly a shipped bug.

12. **`vers` says 1.1.2, the task brief says 1.0.4.** The two `vers`
    resources in `Glider PRO.r` decode to version 1.1.2 with the string
    "(c) 1994-95 Casady & Greene, Inc." I documented what the resource
    says. Which retail build this source corresponds to is unresolved.

### Bugs I found but cannot be sure were fixed later

13. `InitDiedGameOver()` at `GliderPRO/Sources/GameOver.c:290-291`
    uses the screen **width** where it clearly wants the height, so the
    eight "GAME OVER" pages start at a vertical offset derived from the
    horizontal resolution (8.6.1). Visible only on non-4:3 screens.

14. `DoGameOverStarAnimation()` (`GliderPRO/Sources/GameOver.c:135`)
    computes `which = angelDest.left / 32; which = which % 5;`
    (`:159-160`, dereferenced at `:161-162`) while
    `angelDest.left` is still negative, producing `which` = -3, -2, -1
    on the first three firings and writing 8 bytes at 66, 44 and 22
    bytes **below** `pages[]` (8.5.3). `angelDest` is negative because
    `:147` starts it at `QOffsetRect(&angelDest, -96, 0)`. A Go port will panic where the
    original silently corrupted a neighbouring global. The visible
    effect in the original is that no star is drawn until the angel
    reaches x = 0, which a port should reproduce by clamping rather than
    by reproducing the overrun.

15. `ReadScoresFromDisk()` reads with no length bound
    (`GliderPRO/Sources/HighScores.c:801-830`), so a truncated or
    oversized side-car file overruns. Dead code today (see 1).

16. Two-player: if one glider has already died forever
    (`kPlayerIsDeadForever` = -69) and the other collects the last star,
    the win is routed to `DoDiedGameOver()` because the test is
    `if (mortals < 0)` (8.9). The winning player sees the loss
    animation. A port should decide whether to preserve this.

17. `DrawPages()` calls `CopyRectsQD()` **once per page**, i.e. 8 times
    per frame instead of once (8.6.3). Harmless correctness-wise,
    8x the blit cost.

18. The "rooms" column of the highlighted high-score row never turns
    white (`GliderPRO/Sources/HighScores.c:240`) — the `ForeColor`
    switch is missing for that one column (7.10).

19. Cmd-Q during play forfeits the score silently: `playing = false`
    without `gameOver = true`, so `PlayGame()` returns and
    `TestHighScore()` is never called (8.8). And the "Save" button in
    the resulting confirmation dialog calls `SaveGame2()`, which is a
    stub (9.3).

20. `WhereDoesGliderBegin()` leaves `initialPt` uninitialised for any
    `mode` other than 0 or 1 (`GliderPRO/Sources/House.c:227-240`).
    Latent — there is no third mode.

21. DLOG 1020 (the high-score name dialog) appears at the resource's own
    coordinates because `CenterDialog(kHighNameDialogID)` is commented
    out (7.11). On the original 640x480 that is near (0, 0).

22. `LoadScaledGraphic(kMilkywayPictID, &tempRect)` stretches the
    640x460 PICT 1995 into a 640x480 rect on the loss path (8.6.1),
    a 4% vertical stretch. PICT 1021 on the win path is natively
    640x460 into a 640x460 rect, so it is 1:1.

### Data I could not decode (item 23 has since been resolved)

23. **PICT 1998 now decodes.** It is `kHighScoresMaskID`
    (`GliderPRO/Sources/HighScores.c:24`, loaded at `:118`) — the 1-bit
    mask for the high-score plaque PICT 1994, *not* a scoreboard
    background variant. It is a version-1 PICT (version word `0x1101`)
    whose opcode stream is `0xA0` shortComment, `0x01` clip, then
    `0x98` `PackBitsRect` over an **old-style 10-byte `BitMap`** (the
    rowBytes high bit is clear, so there is no `PixMap` and no colour
    table): `rowBytes` 42, `pixelSize` 1, bitmap bounds
    `(0, 0, 30, 336)` against `picFrame` / src / dst
    `(0, 0, 30, 332)` — i.e. 4 columns of rowBytes padding beyond the
    332-pixel frame. PICTs 5006, 5002, 1989 and 1020 are version 1 in
    the same way and all decode as 1-bit masks.

    > [verified by re-parsing the resource bytes; the earlier claim that
    > this PICT was a second scoreboard background was wrong, and the
    > rest of this document (5.3, 7.x) already identifies it correctly
    > as the plaque mask.]

24. The `Sampler` house's data fork is **2 bytes longer** than
    `866 + 348 * nRooms`; the trailing bytes are `01 01`. Every other
    house matches exactly (7.2). Unexplained.

25. `houseType.unusedBoolean` at offset 861 holds 255, 30, 37, 185, 30,
    14 or 2 in six houses and 0 in the rest (9.5). Never written by any
    code. Preserve verbatim.

26. The badge blank strip's last row is index **250** in one place and
    **172** in another (5.5); I recorded the observation but not a
    reason.

27. `Environ.c` contains pixel arithmetic that produces 6396 where 6024
    would be expected. Not in scope for scoring, but it feeds the screen
    metrics the scoreboard geometry derives from, so a port that
    reproduces one number and not the other will get a 1-pixel
    scoreboard offset on some resolutions.

28. The application font. `TextFont(applFont)` resolves to the
    *system's* application font (Geneva on the era's Macs), which is not
    in the resource fork. Every pixel position in section 5 and 7 was
    derived from the code, not from a rendered screenshot, so **the
    string widths and baselines in this document are exact only if the
    port's font metrics match Geneva 9/12**. This is the largest
    single source of visual divergence in the whole subsystem.

---

## Porting notes

Concrete advice, in the order a Go implementation should tackle it.

### A. Get the numbers in before the pixels

The entire scoring/lives/inventory model is 6 point constants, 4 supply
constants, 3 thrust/lift constants, 2 band constants and one starting
life count. There is no hidden state. Implement and unit-test this
first, from these tables, before touching graphics:

| group | constants |
|---|---|
| points | `kRoomVisitScore` **100**, `kRedClockPoints` **100**, `kBlueClockPoints` **300**, `kYellowClockPoints` **500**, `kCuckooClockPoints` **1000**, `kStarPoints` **5000**, plus the per-object `bonusType.points` for `kInvisBonus` (section 2.1) |
| lives | `kInitialGliders` **2** (= **3** lives; the counter is "spares"), doubled for two-player; `kPlayerIsDeadForever` **-69** (section 3) |
| supplies | `kBatterySupply` 50, `kHeliumSupply` 150, `kBandsSupply` 8, `kFoilSupply` 8 (section 4) |
| rates | `kNormalThrust` 5, `kHyperThrust` 8, `kHeliumLift` 4 (section 4) |
| bands | `kMaxRubberBands` 2, `kRubberBandVelocity` 20 (section 4.6) |
| roll | 13 points per frame (section 2.9) |
| frame | `kTicksPerFrame` 2 -> 30.07 fps (section 6) |

**The single most important non-obvious fact: `mortals` counts *spare*
gliders, not lives.** `kInitialGliders` = 2 means the player gets three
attempts. The loss condition is `mortals < 0`, checked in
`GliderPRO/Sources/Play.c:546`, i.e. you lose after the *fourth* death
would be needed. Off-by-one here is the easiest way to ship a game
that is 33% too easy or too hard.

**Second most important: `batteryTotal` is one signed counter for two
resources.** Positive = battery charge, negative = helium charge, zero
= empty (section 4.5). Picking up helium *sets* it to `-kHeliumSupply`,
clobbering any battery charge, and vice versa. Do not model them as two
fields; the pickup semantics and the `< 0` / `> 0` tests everywhere
depend on the single-counter representation.

**Third: the only extra life in the game is the `kPaper` object**
(section 3.2). There are **no** score-threshold extra lives. I verified
this by enumerating all six write sites of `mortals`. If a port adds
"extra life every 20000 points" it is not Glider PRO.

### B. Reproduce the score roll exactly or the game feels wrong

`theScore` is not what the scoreboard shows. The board shows a separate
`displayedScore` that chases `theScore` at
`kScoreRollAmount` = **13** points per frame
(`GliderPRO/Sources/Scoreboard.c:21, 84`), clamped so it never
overshoots, with a `kScoreTikSound` on every frame of the chase
(`GliderPRO/Sources/Scoreboard.c:80-92`; section 2.9). At 30.07 fps that
is **390.9 points/second**, so a 100-point room visit rolls for 8 frames
(0.27 s), a 1000-point cuckoo clock for 77 frames (2.56 s), and a
**5000-point star for 385 frames = 12.8 seconds**. This is a
load-bearing piece of game feel and it also means
the displayed score lags the real score across a room transition and
across the 16-frame game-over countdown (8.3) — the roll keeps running
during the countdown, so the final displayed number may still be
climbing when the animation starts.

### C. The scoreboard geometry is derived, not authored

Do not hard-code the scoreboard rects. They are computed in 11 steps
from the screen width (5.2), and the derivation includes an
`AdjustScoreboardHeight()` pass that depends on `numNeighbors`, which
depends on the room's exits. Port the derivation. Then verify against
the worked 640x480 numbers in section 5.2, which I computed by hand
from the code.

Five separate GWorlds back the board (`boardSrcMap`, `badgeSrcMap`,
`boardTSrcMap`, `boardGSrcMap`, `boardPSrcMap` —
`GliderPRO/Headers/Scoreboard.h:11-15`). Only the "T"/"G"/"P" ones are
redrawn per frame; the base and badge are static. Keeping that split
matters for performance in a naive Go blitter.

### D. High scores: one file, one write, whole handle

The high-score table lives **inside the house file's data fork at byte
offset 528, 292 bytes long** (7.6). It is not a separate file, not a
preference, not a resource. It is written by `WriteHouse()` as part of
**one `FSWrite` of the entire house handle** followed by `SetEOF`
(7.13). Practical consequences for a port:

* You cannot update a high score without rewriting the whole house
  file. Either accept that, or add a targeted 292-byte write at
  offset 528 — which is safe, because nothing else changes when only
  `gameDirty` is set.
* `houseType.timeStamp` is `GetDateTime() & 0x7FFFFFFF` **with bit 0
  repurposed as the house-lock flag** (7.13). Bit 31 is discarded, so to
  recover the real date you add `0x80000000` back and mask off bit 0.
  I verified this against all 22 shipped houses; without the fix they
  all decode to 1927-1932.
* `timeStamp` and `version` are only re-stamped when `fileDirty` is
  set. A high-score-only save (`gameDirty` only) leaves them alone, so
  scores can be recorded without invalidating the house.
* `houseIsReadOnly` gates high-score persistence;
  `houseUnlocked` does **not** (7.13). Getting these backwards means
  either no scores are ever saved or locked houses get modified.

`scoresType` layout (`GliderPRO/Headers/GliderStructs.h:106-114`, 292
bytes, embedded in `houseType` as the field `highScores` at
`GliderPRO/Headers/GliderStructs.h:191`), which a port must match byte
for byte if it wants to read shipped houses (7.2):

| offset | field | type | size |
|---|---|---|---|
| 0 | `banner` | `Str31` | 32 |
| 32 | `names[10]` | `Str15[10]` | 160 |
| 192 | `scores[10]` | `long[10]` | 40 |
| 232 | `timeStamps[10]` | `unsigned long[10]` | 40 |
| 272 | `levels[10]` | `short[10]` | 20 |

(`kMaxScores` = **10**, `GliderPRO/Headers/GliderDefines.h:249`;
`sizeof(scoresType)` = **292**, used as the side-car's `byteCount` at
`GliderPRO/Sources/HighScores.c:771`.)

**The padding bytes inside every `Str31`/`Str15` are stale garbage**
(sections 0.4, 7.2 and 7.6) because `PasStringCopy` writes only `length + 1` bytes.
A port that zero-fills them will not produce byte-identical files. If
byte-identical round-tripping matters, preserve the tail bytes on read
and write them back.

`SortHighScores()` is a **first-wins-on-ties selection sort** (7.6), so
a new score equal to an existing one sorts *below* it. `lastHighScore`
is recorded before the sort and still names the right row after it,
because the sort is stable with respect to the inserted element's
identity — I verified this by reading the loop, not by assuming it.

### E. Game over: two completely different code paths

`if (mortals < 0) DoDiedGameOver(); else DoGameOver();`
(`GliderPRO/Sources/Play.c:546-549`). These share almost nothing:

| | win (`DoGameOver`) | loss (`DoDiedGameOver`) |
|---|---|---|
| backdrop | PICT 1021 (Milky Way), 640x460, drawn 1:1 | the live play field, left as-is |
| animation | angel (PICT 1019 + mask 1020, 96x44) walks right at +2 px/frame, dropping a 6-frame falling star every 16 frames | 8 pages (PICT 1990 + mask 1989) tumble in from the left and settle into "GAME OVER" |
| pages used | `pages[0..4]` | `pages[0..7]` |
| sound | `kMysticSound` once | `kPaper1..4` per page event |
| duration | ~449 frames ~= 15 s at 640 wide, then `WaitForInputEvent(5)` | 120-136 frames (seed-dependent), measured by Monte Carlo (8.6.2) |
| high score | `TestHighScore()` return value used | return value ignored; `RedrawSplashScreen()` unconditional |

The `pages[8]` array is shared between the two, and
`SetUpFinalScreen()` re-initialises `dest`/`was`/`frame` but **not**
`stuck`/`counter` (8.1) — so a win immediately after a loss inherits
stale values. Reproduce or fix, but know it is there.

The loss animation's per-frame displacement table (8.6.2) is worth
transcribing literally: `dx = 44 + 10 * (c - 1)` and
`dy = 36 + 10 * (c - 1)` per tumble cycle `c`, so each page drifts
`dx - dy = 8` px right per cycle, over 24-34 frames per cycle. `case 8`
in the frame switch is **unreachable** — I proved this by exhausting the
advance branch's reachable frame values. Do not port it.

PICT 1988's eight cells are, in order, **G A M E O V E R** (verified by
pixel decode, 8.6.4). The word break in "GAME OVER" comes from
`kPageSpacing` = 40 producing a **double gap** (80 px) between page 3
and page 4 (8.6.1).

### F. The palette is fixed; hard-code it

`NewGWorld(..., 8, ..., nil, nil, useTempMem)` passes a nil colour
table, there is no `pltt` resource, and `SetPaletteToGrays()` is
commented out. `clut` 128 and `clut` 129 are byte-for-byte identical
(2056 bytes) and match the embedded ctTable of every 8-bit PICT I
checked (1988, 1990, 4002, 1021, 1995, 1994) with **zero** mismatches.
It is the standard Macintosh 8-bit system palette, generatable from a
formula (8.5.1):

```
L := [6]uint8{255, 204, 153, 102, 51, 0}
for i := 0; i <= 214; i++ {
    pal[i] = RGB{L[i/36], L[(i%36)/6], L[i%6]}
}
// 215..224 red ramp, 225..234 green, 235..244 blue, 245..254 grey,
// each using levels 0xEE,0xDD,0xBB,0xAA,0x88,0x77,0x55,0x44,0x22,0x11
pal[255] = RGB{0, 0, 0}
```

Note the trap: **index 244 is `#000011`, not black**, and index 255 is
the only true black. Index 0 is white. `ForeColor()` constants are *not*
palette indices (see the table in 11.1).

### G. Byte order, alignment, and Pascal strings

* Everything on disk is **big-endian**. Use `binary.BigEndian`
  explicitly on every field; do not `unsafe`-cast structs.
* `#pragma options align=mac68k` = 2-byte alignment. The places it
  actually bites in this subsystem: `pageType` is
  `8 + 8 + 2 + 2 + 1 = 21` bytes padded to **22** (8.5.3), and
  `savedRoom` is `2 + 1 + 1 + 288 = 292` with no padding (9.4).
* `Point` is `{v, h}`, `Rect` is `{top, left, bottom, right}`.
* Pascal strings are length-prefixed with garbage past the terminator.

### H. Things that look like features but are dead code

Do not spend time on these unless you are deliberately restoring them:

| feature | why it's dead | citation |
|---|---|---|
| Save game (`SaveGame2`) | body entirely commented out; both Cmd-S and the Cmd-Q prompt call it | 9.3 |
| Open saved game (`OpenSavedGame`) | `return false;` before the commented body | 9.3 |
| Resume dialog (`QueryResumeGame`) | never called | 9.7 |
| `NewGame(kResumeGameMode)` | unreachable, because `OpenSavedGame()` is always false | 9.6 |
| `houseType.savedGame` (40 bytes @820) | write-only; only readers are in the two dead functions above | 9.7 |
| The `'gliS'` high-score side-car | `IsFileReadOnly()` always false | 7.6 |
| The high-score *display* | both `DissBits` calls commented out | 7.9 |
| `doRollScore == false` branch | never set false | 2.9 |
| `pointsSrc[14]` (250 points) | no event awards 250 | 2.7 |
| `case 8` in `HandlePages` | unreachable frame value | 8.6.2 |

But **do** implement the `savedGame` block as 40 opaque preserved bytes
in the house reader/writer, because 20 of the 22 shipped houses have
non-zero content there (much of it leaked heap — `Art Museum` is full of
the Mac Memory Manager's `0xF6` debug fill and `Fun House` contains the
literal Pascal string `"DiskCopy"`), and two houses have
`hasGame == 1`. If a port ever honours `hasGame`, it must **reject
`version != 0x0200`**, because every shipped record says `0x0100`
(9.5).

### I. Fidelity risks ranked

1. **Font metrics** (open question 28). Everything text-shaped in
   sections 5 and 7 depends on Geneva's widths. Highest visual risk,
   and unavoidable without the original font.
2. **`mortals` off-by-one** (three lives, not two).
3. **`batteryTotal` sign convention** (one counter, not two).
4. **Score-roll rate and its sound** (13/frame, priority 101 so it is
   the first thing drowned out).
5. **Big-endian field order in the 292-byte high-score block**, and the
   stale padding bytes.
6. **`houseType.timeStamp`'s bit-0 lock flag and discarded bit 31.**
7. **The `numNeighbors`-dependent scoreboard height.**
8. **Negative-modulo behaviour** at `GameOver.c:326` and `:340` — C
   truncates toward zero and yields non-positive remainders for
   negative operands; Go's `%` behaves identically, so a literal
   translation is correct, but the resulting negative array index will
   panic in Go where it silently corrupted memory in C. Clamp.
9. **`initNoInterp` nearest-neighbour resampling** of 22254.5455 Hz
   8-bit PCM. Linear interpolation sounds cleaner than the original.
10. **Frame pacing**: `kTicksPerFrame` 2 against a 60.15 Hz tick is
    30.07 fps, not 30.00. Over a 15-second game-over animation that is
    a ~5-frame difference.

---

# Appendix Z. Open question 28 (the application font) - bounded and de-risked

*Appended section. Added by the toolbox-primitives investigation; nothing above this line was
modified. Full evidence in `docs/analysis/toolbox-primitives.md` (see especially §4.2, §4.5,
§4.13, §4.14 and §5.5).*

Open question 28 (this document, line ~5401, and cited again as fidelity risk #1) says the
string widths and baselines here are exact only if the port's font metrics match Geneva 9/12, and
calls that "the largest single source of visual divergence in the whole subsystem". That
assessment can now be narrowed considerably.

## Z.1 The font resources do not exist, so a substitute is mandatory

Searched all 538 resource records in `GliderPRO/Glider PRO.r`: **zero** `FOND`, `NFNT`, `FONT`,
`sfnt`, `fdsc` or `fmtx`. Chicago and Geneva lived in the System file, not in the application.
Glyph bitmaps are unrecoverable in principle from the GPLv2 release.

`systemFont` = 0 = Chicago; `applFont` = 1 = Geneva (the user-selectable application font).

## Z.2 Baselines are not at risk at all

**`GetFontInfo` and `FontMetrics` are called zero times in the entire tree.** So is `TextMode`,
`CharExtra`, `SpaceExtra`, `GetFNum`, `RealFont`, `SetFractEnable`, `TruncString`, `TruncText`,
`StdTxMeas` and `MeasureText`. Every vertical position in this subsystem is a hard-coded integer
lifted straight from the source, so every baseline recorded in this document is exact
independent of the font.

The single exception anywhere in the program is `TETextBox` at
`GliderPRO/Sources/DialogUtils.c:642` (Geneva 9, in `DrawDialogUserText`), which uses TextEdit's
own ascent+descent+leading. It is not in the scoring subsystem.

## Z.3 Widths are at risk only where a `StringWidth` result feeds a position

There are exactly 11 `StringWidth` sites and 1 `TextWidth` site in the tree. The
scoring-relevant ones:

| Site | Use |
|---|---|
| `GliderPRO/Sources/Banner.c:233` | centres the stars-remaining count: `bounds.left + 102 - (StringWidth(theStr)/2)` |
| `GliderPRO/Sources/HighScores.c:142` | centres the house title: `((kScoreWide - StringWidth(...))/2) - 1` |
| `GliderPRO/Sources/HighScores.c:157` | sizes the banner box: `bannerWidth + 8` |
| `GliderPRO/Sources/ObjectDraw2.c:1162` | centres the calendar month in a 64-px field |
| `GliderPRO/Sources/DialogUtils.c:639` | `inset = ((iRect.right - iRect.left) - (StringWidth(stringCopy) + 2)) / 2` |
| `GliderPRO/Sources/GameOver.c:102` | `TextWidth`, and note it is measured **before** the font is set |

Every one is a centring computation. An error of `d` pixels in `StringWidth` shifts the string by
`d/2` pixels and moves nothing else. There is no reflow, no wrapping by width (`WrapText` wraps by
**character count** - `GliderPRO/Sources/StringUtils.c:216-247`), and no vertical consequence.

Note the C truncation in every one of those expressions: `(64 - 65) / 2` is **0**, not -1, so a
1-px-over-wide centred string starts at the left edge and clips on the right.

## Z.4 The normative substitute, and its measured cost on the scoring surfaces

| key | substitute | ppem | asc | desc | lead | lineHeight | digit advance |
|---|---|---|---|---|---|---|---|
| CHI12 (Chicago 12 plain) | `LiberationSans-Bold.ttf` | 13 | 12 | 3 | 1 | 16 | 7 |
| GEN9 (Geneva 9 bold) | `LiberationSans-Regular.ttf` | 9 | 9 | 2 | 0 | 11 | 5 |
| GEN12 (Geneva 12 bold) | `LiberationSans-Regular.ttf` | 11 | 10 | 3 | 1 | 14 | 6 |
| GEN14 (Geneva 14 bold) | `LiberationSans-Regular.ttf` | 13 | 12 | 3 | 1 | 16 | 7 |

QuickDraw synthetic bold adds **+1 px to every character's advance**, so
`StringWidth_bold(s) = StringWidth_plain(s) + len(s)`.

GEN12 is pinned to 11 ppem by two independent constraints inside this very subsystem:

1. The scoreboard score field `boardPSrcRect` is **64 px** wide
   (`GliderPRO/Sources/StructuresInit.c`) and takes up to 9 digits. At 6 px plain + 1 px
   synthetic bold, 9 digits = **exactly 63 px** - one pixel of slack, and 10 digits (70 px)
   overflows precisely as the original does.
2. `boardTSrcRect` is **12 px** tall with the room-title baseline at v = 10
   (`GliderPRO/Sources/Scoreboard.c:135-175`), so the ink descent must be <= 2. GEN12's is
   exactly 2.

At 12 ppem both constraints fail. This is strong evidence the substitute is correctly sized.

Measured outcome across the 27 scoring-related layout sites (rows 1-27 of §4.13 in
`toolbox-primitives.md`): **26 behave exactly as the original does** - they fit where the
original fits and clip where the original clips. The clips are all original overflows, not
substitute artefacts:

| Site | String | Substitute px | Budget px | Also clips in the original? |
|---|---|---|---|---|
| `Scoreboard.c:164` | 27-char room name of `W`s | 297 | 255 | yes (`Str27` vs a 255-px field) |
| `Scoreboard.c:251` | 10-digit score | 70 | 63 | yes |
| `Banner.c:137` | 39 `W`s (`WrapText` 40) | 429 | 314 | yes (`WrapText` counts characters) |
| `GameOver.c:108` | 63 `W`s (`WrapText` 64) | 693 | 639 | yes |
| `HighScores.c:202` | 15 `W`s (`Str15` name) | 165 | 129 | yes |
| `HighScores.c:142` | 31-char house title | 425 | 352 | yes |
| `MainWindow.c:71` | 31-char house name on the splash | 341 | 204 | yes |

The **one** substitute-induced deviation on a scoring surface is `HighScores.c:253`: a 9-digit
high score measures 63 px in a 61-px column, a 2-px clip. (The original clipped there too, by an
unknown amount.) One further substitute-induced deviation exists outside the scoring subsystem:
`SEPTEMBER` at 65 px in the 64-px calendar field (`ObjectDraw2.c:1162`), clipping 1 px right.

## Z.5 Revised statement of the residual risk

Open question 28's residual risk is:

1. Half-pixel-scale horizontal shifts on **five** centred strings (the five `StringWidth` sites
   listed in Z.3), each bounded by `d/2` where `d` is the substitute's width error.
2. A **2-px** right-edge clip on 9-digit high scores.
3. Nothing vertical, anywhere.

It is no longer the largest source of visual divergence in the subsystem. For comparison, the
`kTicksPerFrame` 2 against a 60.15 Hz tick (item 10 in the porting-risk list above) accumulates a
~5-frame difference over a 15-second game-over animation, which is a larger visible deviation
than any of the three above.

## Z.6 Related original bugs that affect text in this subsystem

| Bug | Citation | Effect |
|---|---|---|
| `TextWidth` measured before the font is set | `GliderPRO/Sources/GameOver.c:101-105` | the death trailer is centred against whatever font the port last had |
| Font state inherited from another drawable | `GliderPRO/Sources/Room.c:259, :261` | `"No rooms"` renders in whatever font `workSrcMap` last had |
| `TextFace(applFont)` where a size/font call was meant | `GliderPRO/Sources/Coordinates.c:154` | `applFont` = 1 = `bold`, and there is no preceding `SetPort`, so a stray bold lands on an unrelated port; the coordinate window itself stays Chicago 12 plain |
| `CollapseStringToWidth` byte underflow | `GliderPRO/Sources/StringUtils.c:287-291` | `theStr[0]--` with no floor; if `StringWidth("\p…") > wide` the loop never terminates and eventually reads 255 bytes of adjacent memory. Never fires in the shipped game (narrowest call is 94 px) but must be guarded in a port. |

The scoreboard's drop-shadow idiom, for completeness, is two draws per field: black at
`MoveTo(1, 10)` then white at `MoveTo(0, 9)` (`GliderPRO/Sources/Scoreboard.c:135-330`, five
fields, with the depth-4 background substitution
`kGrayBackgroundColor4` = 10 for `kGrayBackgroundColor` = 251 at `:143`, `:201`, `:239`, `:276`,
`:311`).
