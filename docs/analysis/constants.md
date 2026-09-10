# Glider PRO 1.0.4 — Master Constants and Magic Numbers Catalog

## Scope

This document is the exhaustive reference for **every named compile-time constant** and
**every load-bearing unnamed numeric literal** in the Glider PRO 1.0.4 source release
(John Calhoun, GPLv2). It exists so that a Go re-implementation can be checked
mechanically against the original rather than by reading prose.

It covers:

* All 1327 active `#define`s in the tree, grouped by domain, each with its literal value
  (decimal, and hex where the value is an ID/mask/bit-field), its `file:line`, and a
  one-line meaning.
* The complete struct catalogue from `Headers/GliderStructs.h` with field types, byte
  sizes and **byte offsets**, split into *on-disk* (big-endian, `mac68k`-packed, must be
  reproduced byte-exactly) and *runtime-only* (free to redesign).
* Empirical byte-level verification of the on-disk layouts, performed with `python3`
  against real BinHex-encoded house files shipped in `GliderPRO/Houses/`, with the
  observed values quoted.
* Empirical verification of the resource-ID conventions against the 15 MB Rez dump
  `GliderPRO/Glider PRO.r` (the app's own resource fork) and against a real house's
  resource fork.
* Every hard-coded numeric literal in the physics/dynamics code that is **not** behind a
  `#define` — the "fidelity landmines" — with its exact call site.
* The Mac-Toolbox-specific items each constant is entangled with, and what a Go port must
  substitute.

Out of scope: rendering algorithms, the level editor's UI behaviour, and the QuickTime
movie subsystem, except where a constant belongs to them.

## Sources read

All files were read in full. Because `GliderPRO/Sources/*.c` and `GliderPRO/Headers/*.h`
are **classic Mac text (CR-only line endings, MacRoman encoding)**, every file was first
converted into a scratch directory with line numbers preserved 1:1:

```sh
cd GliderPRO
for f in Headers/*.h Sources/*.c; do
  o="/tmp/wf-constants/$(echo $f | tr '/' '_')"
  tr '\r' '\n' < "$f" | iconv -f MAC -t UTF-8 > "$o" 2>/dev/null || tr '\r' '\n' < "$f" > "$o"
done
```

Every `file:line` citation in this document is the line number **in the CR→LF converted
copy**, which is identical to the line number a normal editor would show.

Primary (read in full, cover-to-cover):

| File | Lines | Why |
|---|---|---|
| `GliderPRO/Headers/GliderDefines.h` | 625 | 566 of the 1327 defines; the game's constant bible |
| `GliderPRO/Headers/Externs.h` | 394 | 197 defines (keys, ASCII, errors, menu items) + `prefsInfo` |
| `GliderPRO/Headers/GliderStructs.h` | 347 | every struct, with the author's own byte-size comments |
| `GliderPRO/Headers/GliderVars.h` | 59 | the global sprite-rect / world-state extern list |
| `GliderPRO/Sources/StructuresInit.c` | 725 | every sprite atlas rectangle, PICT IDs 3998–4017 |
| `GliderPRO/Sources/StructuresInit2.c` | 476 | clutter/angel/support atlases, `CreatePointers`, `InitSrcRects` |
| `GliderPRO/Sources/Player.c` | 1605 | glider physics; `kGravity`/`kHImpulse`/`kVImpulse`/`kMaxHVel` |
| `GliderPRO/Sources/Interactions.c` | 1777 | all 28 hot-spot actions, rewards, escapes, web |
| `GliderPRO/Sources/Dynamics.c` | 777 | appliance animation timers |
| `GliderPRO/Sources/Dynamics2.c` | 590 | enemy animation/physics |
| `GliderPRO/Sources/Dynamics3.c` | 556 | `AddDynamicObject` — all dynamic-object seeding |
| `GliderPRO/Sources/Modes.c` | 640 | glider mode transitions |
| `GliderPRO/Sources/Input.c` | 399 | thrust constants, demo playback |
| `GliderPRO/Sources/RubberBands.c` | 318 | band physics |
| `GliderPRO/Sources/Grease.c` | 302 | grease spill state machine |
| `GliderPRO/Sources/ObjectRects.c` | 1188 | how each object's collision rect is derived |
| `GliderPRO/Sources/InterfaceInit.c` | 221 | `fadeInSequence[16]` and global defaults |
| `GliderPRO/Sources/Trip.c` | 245 | trigger→dynamic-object dispatch timers |
| `GliderPRO/Sources/Coordinates.c` | 196 | room/coordinate maths |
| `GliderPRO/Sources/Sound.c` | 566 | `snd ` loading, `kMaxSounds`, the 20-byte header skip |

Read in part (targeted at the constant/algorithm in question): `HouseIO.c` (`ReadHouse`),
`Room.c` (`SetInitialTiles`), `RoomGraphics.c` (`DrawRoomBackground`), `ObjectDraw.c`,
`ObjectDraw2.c`, `Objects.c`, `Play.c`, `Scoreboard.c`, `Render.c`, `Triggers.c`,
`Transit.c`, `Map.c`, `Tools.c`, `Settings.c`, `HouseLegal.c`, `Environ.c`, `Validate.c`,
`Music.c`, `Prefs.c`, `SavedGames.c`, `SelectHouse.c`, `Marquee.c`, `Transitions.c`,
`GameOver.c`, `HighScores.c`, `Banner.c`, `HouseInfo.c`, `RoomInfo.c`, `ObjectInfo.c`,
`ObjectAdd.c`, `ObjectEdit.c`, `ObjectDrawAll.c`, `About.c`, `AnimCursor.c`,
`AppleEvents.c`, `DialogUtils.c`, `Events.c`, `FileError.c`, `House.c`, `Link.c`,
`Main.c`, `MainWindow.c`, `Menu.c`, `StringUtils.c`, `Utilities.c`, `WindowUtils.c`.

Binary evidence:

| Artifact | What was parsed |
|---|---|
| `GliderPRO/Glider PRO.r` | 199 843-line Rez dump of the app's resource fork; every resource type/ID enumerated |
| `GliderPRO/Houses/Empty House.binhex` | BinHex 4.0 → data fork 13 046 B, resource fork 2 670 B |
| `GliderPRO/Houses/Sampler.binhex` | BinHex 4.0 → data fork 1 564 B, resource fork 286 B |
| `GliderPRO/Houses/In The Mirror.binhex` | BinHex 4.0 → data fork, resource fork 151 870 B (custom PICTs + `snd `) |
| `GliderPRO/Houses/California or Bust!.binhex` | BinHex 4.0 → layout check, `nRooms = 16` |

---

## 0. Conventions used in this document

* `X:NNN` after a name means `GliderPRO/…/X` line `NNN` of the CR→LF-converted copy.
* "on-disk" means the byte layout is part of the house/saved-game/prefs file format and
  **must** be reproduced exactly by a Go port. "runtime" means the C struct only ever
  lives in RAM and the port may redesign it freely.
* All on-disk multi-byte integers are **big-endian** (68k/PPC). `short` = int16,
  `long` = int32, `Byte` = uint8, `Boolean` = uint8 (0 = false, non-zero = true — the
  shipped houses use `0`, `1`, `2` and `255`, see §22.5).
* `Point` is `{short v; short h;}` — **v (vertical) comes first**. `Rect` is
  `{short top; short left; short bottom; short right;}` — **top, left, bottom, right**.
  Getting either order wrong silently transposes the whole game.
* `Str15`/`Str27`/`Str31`/`Str32`/`Str255` are Pascal strings: 1 length byte followed by
  N content bytes, padded to the declared total. `Str15` occupies 16 bytes, `Str27`
  occupies 28, `Str31` occupies 32, `Str32` occupies 34 (odd length + pad), `Str255`
  occupies 256. The *declared* number is the maximum content length, the *storage* is
  one more, rounded up to even.

---

## 1. Global facts about the constant system

### 1.1 There are no enums and no `const`

Verified mechanically over the whole tree:

```sh
grep -ac '\benum\b'         GliderPRO/Sources/*.c GliderPRO/Headers/*.h   # → 0 everywhere
grep -ac '\bstatic const\b' GliderPRO/Sources/*.c GliderPRO/Headers/*.h   # → 0 everywhere
grep -ac '\bconst\b'        GliderPRO/Sources/*.c GliderPRO/Headers/*.h   # → 0 everywhere
```

**Every** symbolic constant in Glider PRO is a preprocessor `#define`. There is not a
single `enum`, `const`, or `static const` in the codebase. Consequently:

* There is no type checking on any constant. `kFloorVent` (`0x01`) and `kBumpUp` (`1`) and
  `kYellowUnaccounted` (`1`) are all interchangeable integers, and the code does in fact
  rely on some of these coincidences (see §1.4).
* Several names are **defined more than once, in different files, with different values**
  (§1.3), and several are defined *inside function bodies* (§1.2).

A Go port should turn these into typed constant groups (`type ObjectClass uint8`,
`type GliderMode int16`, …). Where this document lists a value's *domain*, that is the
suggested Go type.

### 1.2 Function-local `#define`s

THINK C idiom: a `#define` may appear *inside* a function body, scoped by convention only
(the preprocessor does not scope it — it stays defined for the rest of the translation
unit). Glider PRO uses this for one-shot tuning numbers. There are **99** of them across
the 46 `.c` files (enumerated by walking brace depth); the complete list follows.

| Name | Value | Site | Enclosing function |
|---|---|---|---|
| `kAboutDialogID` | 150 | `GliderPRO/Sources/About.c:34` | `DoAbout` |
| `kTextItemVers` | 2 | `GliderPRO/Sources/About.c:35` | `DoAbout` |
| `kPictItemMain` | 4 | `GliderPRO/Sources/About.c:36` | `DoAbout` |
| `kColorSwitchedAlert` | 1042 | `GliderPRO/Sources/Events.c:48` | `BitchAboutColorDepth` |
| `kMaxColumnsWide` | 96 | `GliderPRO/Sources/Transitions.c:20` | `PourScreenOn` |
| `kChipHigh` | 20 | `GliderPRO/Sources/Transitions.c:21` | `PourScreenOn` |
| `kChipWide` | 16 | `GliderPRO/Sources/Transitions.c:22` | `PourScreenOn` |
| `kClimbStairsSpeed` | -4 | `GliderPRO/Sources/Player.c:317` | `MoveGliderUpStairs` |
| `kVClimbStairsSpeed` | -4 | `GliderPRO/Sources/Player.c:385` | `FinishGliderUpStairs` |
| `kHClimbStairsSpeed` | -4 | `GliderPRO/Sources/Player.c:386` | `FinishGliderUpStairs` |
| `kVDropStairsSpeed` | 4 | `GliderPRO/Sources/Player.c:433` | `MoveGliderDownStairs` |
| `kHDropStairsSpeed` | 4 | `GliderPRO/Sources/Player.c:434` | `MoveGliderDownStairs` |
| `kVDropStairsSpeed` | 4 (redef) | `GliderPRO/Sources/Player.c:514` | `FinishGliderDownStairs` |
| `kHDropStairsSpeed` | 4 (redef) | `GliderPRO/Sources/Player.c:515` | `FinishGliderDownStairs` |
| `kVDropDuctSpeed` | 4 | `GliderPRO/Sources/Player.c:659` | `MoveGliderDownDuct` |
| `kVRiseDuctSpeed` | -4 | `GliderPRO/Sources/Player.c:756` | `MoveGliderUpDuct` |
| `kHPushMailSpeed` | -4 | `GliderPRO/Sources/Player.c:853` | `MoveGliderOutMailLeft` |
| `kHPushMailRtSpeed` | 4 | `GliderPRO/Sources/Player.c:891` | `MoveGliderOutMailRight` |
| `kVDropStairsSpeed` | 4 (redef) | `GliderPRO/Sources/Player.c:929` | (mail helper) |
| `kHMailPullSpeed` | 4 | `GliderPRO/Sources/Player.c:962` | `MoveGliderInMailLeft` |
| `kVMailDropSpeed` | 2 | `GliderPRO/Sources/Player.c:963` | `MoveGliderInMailLeft` |
| `kHMailPullRtSpeed` | -4 | `GliderPRO/Sources/Player.c:1053` | `MoveGliderInMailRight` |
| `kVMailDropSpeed` | 2 (redef) | `GliderPRO/Sources/Player.c:1054` | `MoveGliderInMailRight` |
| `kDropShredSlow` | 1 | `GliderPRO/Sources/Player.c:1262` | `MoveGliderShredding` |
| `kDropShredFast` | 4 | `GliderPRO/Sources/Player.c:1263` | `MoveGliderShredding` |
| `kKillWebbedGlider` | 150 | `GliderPRO/Sources/Interactions.c:1738` | `WebGlider` |
| `kFramesToBurn` | 60 | `GliderPRO/Sources/Modes.c:408` | `FlagGliderBurning` |
| `kVGliderAppearsComingUp` | 100 | `GliderPRO/Sources/Modes.c:521` | `ReadyGliderForTripUpStairs` |
| `kVGliderAppearsComingDown` | 100 | `GliderPRO/Sources/Modes.c:552` | `ReadyGliderForTripDownStairs` |
| `kSteps` | 16 | `GliderPRO/Sources/DialogUtils.c:228` | `ZoomOutDialogRect` |
| `kZoomDelay` | 1 | `GliderPRO/Sources/DialogUtils.c:229` | `ZoomOutDialogRect` |
| `kSteps` | 16 (redef) | `GliderPRO/Sources/DialogUtils.c:282` | `ZoomInDialogRect` |
| `kZoomDelay` | 1 (redef) | `GliderPRO/Sources/DialogUtils.c:283` | `ZoomInDialogRect` |
| `kStarFalls` | 8 | `GliderPRO/Sources/GameOver.c:138` | (game-over star animation) |
| `kPageSpacing` | 40 | `GliderPRO/Sources/GameOver.c:251` | (page-fly animation) |
| `kPageRightOffset` | 128 | `GliderPRO/Sources/GameOver.c:252` | (page-fly animation) |
| `kPageBackUp` | 128 | `GliderPRO/Sources/GameOver.c:253` | (page-fly animation) |
| `kScoreSpacing` | 18 | `GliderPRO/Sources/HighScores.c:90` | **none — file scope**, immediately above `DrawHighScores` (`:94`) |
| `kScoreWide` | 352 | `GliderPRO/Sources/HighScores.c:91` | **none — file scope** |
| `kKimsLifted` | 4 | `GliderPRO/Sources/HighScores.c:92` | **none — file scope** |
| `kGoToFirstButt` | 2 | `GliderPRO/Sources/House.c:668` | `GoToDialogFilter` |
| `kGoToPrevButt` | 3 | `GliderPRO/Sources/House.c:669` | `GoToDialogFilter` |
| `kGoToFSButt` | 4 | `GliderPRO/Sources/House.c:670` | `GoToDialogFilter` |
| `kFloorEditText` | 5 | `GliderPRO/Sources/House.c:671` | `GoToDialogFilter` |
| `kSuiteEditText` | 6 | `GliderPRO/Sources/House.c:672` | `GoToDialogFilter` |
| `kYellowAlert` | 1006 | `GliderPRO/Sources/HouseIO.c:642` | `YellowAlert` |
| `kSaveGameAlert` | 1041 | `GliderPRO/Sources/Input.c:385` | `QuerySaveGame` |
| `kYesSaveGameButton` | 1 | `GliderPRO/Sources/Input.c:386` | `QuerySaveGame` |
| `kResumeGameDial` | 1025 | `GliderPRO/Sources/Menu.c:712` | (resume-game dialog) |
| `kNotInDemoAlert` | 1037 | `GliderPRO/Sources/Menu.c:769` | (demo gate) |
| `kNoHighScoreAlert` | 1046 | `GliderPRO/Sources/Menu.c:781` | (high-score gate) |
| `kNoMemForMusicAlert` | 1038 | `GliderPRO/Sources/Music.c:413` | (music load failure) |
| `kNoMemForSoundsAlert` | 1039 | `GliderPRO/Sources/Sound.c:518` | (sound load failure) |
| `kNoSoundManager3Alert` | 1030 | `GliderPRO/Sources/Sound.c:529` | (Sound Manager 3 check) |
| `kSavedGameErrorAlert` | 1044 | `GliderPRO/Sources/SavedGames.c:154` | (saved-game load failure) |
| `kOkayButtPICTHiLit` | 151 | `GliderPRO/Sources/About.c:97` | (about-box button) |
| `kOkayButtPICTNotHiLit` | 150 | `GliderPRO/Sources/About.c:119` | (about-box button) |
| `kBaseBytesNeeded` | 614400L | `GliderPRO/Sources/Environ.c:564` | `CheckMemorySize` |
| `kPaddingBytes` | 204800L | `GliderPRO/Sources/Environ.c:565` | `CheckMemorySize` |
| `kMaxDirectories` | 32 | `GliderPRO/Sources/SelectHouse.c:559` | (house search) |
| `kRoomsTimesSuites` | 8192 | `GliderPRO/Sources/HouseLegal.c:648` | `CheckDuplicateFloorSuite` (declared `:646`) |
| `kAllDefaultsButton` | 11 | `GliderPRO/Sources/Settings.c:1397` | (prefs dialog) |
| `kChangesEffectAlert` | 1040 | `GliderPRO/Sources/Settings.c:1476` | (prefs dialog) |
| `kNormalSettingsIcon` | 1010 | `GliderPRO/Sources/Settings.c:1267` | (prefs dialog) |
| `kInvertedSettingsIcon` | 1014 | `GliderPRO/Sources/Settings.c:1268` | (prefs dialog) |
| `kFoilLow` | 2 | `GliderPRO/Sources/Scoreboard.c:74` | `UpdateFoil`-family (25 % warning) |
| `kBatteryLow` | 17 | `GliderPRO/Sources/Scoreboard.c:75` | (25 % warning) |
| `kHeliumLow` | -38 | `GliderPRO/Sources/Scoreboard.c:76` | (25 % warning) |
| `kBandsLow` | 2 | `GliderPRO/Sources/Scoreboard.c:77` | (25 % warning) |
| `kGray2ColorSteps` | 180 | `GliderPRO/Sources/MainWindow.c:554` | (grey-ramp fade) |
| `kBackgroundsMenuID` | 140 | `GliderPRO/Sources/RoomInfo.c:379` | (background popup) |
| `kArrowheadLength` | 4 | `GliderPRO/Sources/ObjectInfo.c:122` | (link arrow drawing) |
| `kWipeRectThick` | 4 | `GliderPRO/Sources/Transitions.c:76` | `WipeScreenOn` |
| `kTrackLightSpacing` | 64 | `GliderPRO/Sources/ObjectDraw2.c:502` | `DrawTrackLight` |
| `kWindowSillThick` | 7 | `GliderPRO/Sources/ObjectDraw2.c:1040` | (window drawing) |
| `kTikiPoleBase` | 300 | `GliderPRO/Sources/ObjectDraw.c:71` | `DrawTiki` |
| `kTableBaseTop` | 296 | `GliderPRO/Sources/ObjectDraw.c:149` | `DrawTable` |
| `kTableShadowTop` | 312 | `GliderPRO/Sources/ObjectDraw.c:150` | `DrawTable` |
| `kTableShadowOffset` | 12 | `GliderPRO/Sources/ObjectDraw.c:151` | `DrawTable` |
| `kBracketInset` | 18 | `GliderPRO/Sources/ObjectDraw.c:265` | `DrawShelf` |
| `kShelfDeep` | 4 | `GliderPRO/Sources/ObjectDraw.c:266` | `DrawShelf` |
| `kBracketThick` | 5 | `GliderPRO/Sources/ObjectDraw.c:267` | `DrawShelf` |
| `kShelfShadowOff` | 12 | `GliderPRO/Sources/ObjectDraw.c:268` | `DrawShelf` |
| `kCabinetDeep` | 4 | `GliderPRO/Sources/ObjectDraw.c:360` | `DrawCabinet` |
| `kCabinetShadowOff` | 6 | `GliderPRO/Sources/ObjectDraw.c:361` | `DrawCabinet` |
| `kCounterFooterHigh` | 12 | `GliderPRO/Sources/ObjectDraw.c:500` | `DrawCounter` |
| `kCounterStripWide` | 6 | `GliderPRO/Sources/ObjectDraw.c:501` | `DrawCounter` |
| `kCounterStripTall` | 29 | `GliderPRO/Sources/ObjectDraw.c:502` | `DrawCounter` |
| `kCounterPanelDrop` | 12 | `GliderPRO/Sources/ObjectDraw.c:503` | `DrawCounter` |
| `kDresserTopThick` | 4 | `GliderPRO/Sources/ObjectDraw.c:645` | `DrawDresser` |
| `kDresserCrease` | 9 | `GliderPRO/Sources/ObjectDraw.c:646` | `DrawDresser` |
| `kDresserDrawerDrop` | 12 | `GliderPRO/Sources/ObjectDraw.c:647` | `DrawDresser` |
| `kDresserSideSpare` | 14 | `GliderPRO/Sources/ObjectDraw.c:648` | `DrawDresser` |
| `kTableBaseTop` | 296 (redef) | `GliderPRO/Sources/ObjectDraw.c:775` | `DrawDeckTable` |
| `kTableShadowTop` | 312 (redef) | `GliderPRO/Sources/ObjectDraw.c:776` | `DrawDeckTable` |
| `kTableShadowOffset` | 12 (redef) | `GliderPRO/Sources/ObjectDraw.c:777` | `DrawDeckTable` |
| `kStoolBase` | 304 | `GliderPRO/Sources/ObjectDraw.c:888` | `DrawStool` |
| `rDeathAlertID` | 170 | `GliderPRO/Sources/Utilities.c:137` | `RedAlert` |
| `rErrTitleID` | 170 | `GliderPRO/Sources/Utilities.c:138` | `RedAlert` |
| `rErrMssgID` | 171 | `GliderPRO/Sources/Utilities.c:139` | `RedAlert` |
| `kChooserStringID` | -16096 | `GliderPRO/Sources/StringUtils.c:304` | `GetChooserName` |
| `kLocalizedStringsID` | 150 | `GliderPRO/Sources/StringUtils.c:323` | `GetLocalizedString` |
| `kMasterTitleLeft` | 6 | `GliderPRO/Sources/Validate.c:22` | **none — file scope** (in the `:16-23` block with `kEncryptMask`, `kMasterDialogID`, …) |
| `kMasterTitleTop` | 16 | `GliderPRO/Sources/Validate.c:23` | **none — file scope** |

The last five rows (`HighScores.c:90-92`, `Validate.c:22-23`) are listed here for
continuity with earlier drafts but are **file-scope** `#define`s, not function-local ones;
the other 99 rows are genuinely inside a function body.

`kMaxDirectories` at `GliderPRO/Sources/SelectHouse.c:559` and `kRoomsTimesSuites` at
`GliderPRO/Sources/HouseLegal.c:648` are the two function-local defines that gate real
allocations: `NewPtrClear(sizeof(char) * kRoomsTimesSuites)` where
`kRoomsTimesSuites = 8192 = kMaxNumRoomsH (128) × kMaxNumRoomsV (64)`
(`GliderPRO/Sources/HouseLegal.c:653`).

### 1.3 Names defined more than once

These are the same identifier with **different** values in different translation units.
A Go port must keep them distinct or it will silently pick the wrong one.

| Name | Values and sites |
|---|---|
| `kBalloonStart` | 310 at `GliderPRO/Sources/Dynamics2.c:14` and 310 at `GliderPRO/Sources/Dynamics3.c:13` (same value, duplicated) |
| `kCopterStart` | 8 at `GliderPRO/Sources/Dynamics2.c:15` and 8 at `GliderPRO/Sources/Dynamics3.c:14` (same) |
| `kDartVelocity` | 6 at `GliderPRO/Sources/Dynamics2.c:17` and 6 at `GliderPRO/Sources/Dynamics3.c:15` (same) |
| `kHandCursorID` | **1000** at `GliderPRO/Sources/AnimCursor.c:15` (as `rHandCursorID`) vs **128** at `GliderPRO/Sources/InterfaceInit.c:16` — genuinely different resources (`acur` chain vs `CURS`) |
| `kActive` / `kControlActive` | 0 at `GliderPRO/Sources/DialogUtils.c:15` and `GliderPRO/Headers/Externs.h:24` |
| `kInactive` / `kControlInactive` | 255 at `GliderPRO/Sources/DialogUtils.c:16` and `GliderPRO/Headers/Externs.h:25` |
| `k8WhiteColor` | 0 at `GliderPRO/Sources/ObjectDraw.c:16` and `GliderPRO/Sources/ObjectDraw2.c:19` |
| `kYellowColor` / `kIntenseYellowColor` | both 5, `GliderPRO/Sources/ObjectDraw.c:17` / `ObjectDraw2.c:20` |
| `kSteps`, `kZoomDelay` | 16 / 1, twice each in `GliderPRO/Sources/DialogUtils.c` (228/229 and 282/283) |
| `kVDropStairsSpeed`, `kHDropStairsSpeed` | 4, three times in `GliderPRO/Sources/Player.c` (433/434, 514/515, 929) |
| `kVMailDropSpeed` | 2, twice in `GliderPRO/Sources/Player.c` (963, 1054) |
| `kTableBaseTop`, `kTableShadowTop`, `kTableShadowOffset` | 296/312/12, twice in `GliderPRO/Sources/ObjectDraw.c` (149-151, 775-777) |
| `kSavedGameVersion` | `0x0200` at `GliderPRO/Sources/SavedGames.c:14` — numerically equal to `kHouseVersion` `0x0200` at `GliderPRO/Headers/GliderDefines.h:517` but a *separate* version space |
| `kHighNameDialogID`/`kAngelPictID` | 1020 vs 1019+1: `GliderPRO/Sources/HighScores.c:25` uses DLOG 1020 while `GliderPRO/Sources/StructuresInit2.c:134` loads PICT 1020 as the angel mask — different resource *types*, same ID |

### 1.4 Numeric coincidences the code relies on

| Coincidence | Where it matters |
|---|---|
| `kRoomWide` = 512 = `kNumTiles`(8) × `kTileWide`(64) | `GliderPRO/Headers/GliderDefines.h:499`; the tile index is `x >> 6` |
| `kVertLocalOffset` = 322 = `kTileHigh` | `GliderPRO/Headers/GliderDefines.h:501`; the comment says "kTileHigh − 39 (was 283, then 295)" but the value shipped is the *unmodified* 322 |
| `kMaxMasterObjects` = 216 = `kMaxRoomObs`(24) × 9 | `GliderPRO/Headers/GliderDefines.h:266`; 9 = the 3×3 room neighbourhood |
| `kHalfGliderWide` = 24 = `kGliderWide`/2 | `GliderPRO/Headers/GliderDefines.h:550` |
| `kNoLeftWallLimit` = -24 = 0 − `kGliderWide`/2 | `GliderPRO/Headers/GliderDefines.h:507` (comment states the formula) |
| `kNoRightWallLimit` = 536 = `kRoomWide` + `kGliderWide`/2 | `GliderPRO/Headers/GliderDefines.h:509` |
| `kNumSrcRects` = `0x90` = 144 = one past the highest object ID `kChimes` (`0x8F`) | `GliderPRO/Headers/GliderDefines.h:437`; `srcRects` is indexed directly by the object's `what` byte |
| `kTotalTools` = 16 = `kToolsHigh`(4) × `kToolsWide`(4) | `GliderPRO/Sources/Tools.c:17` (comment states the formula) |
| `kRoomsTimesSuites` = 8192 = `kMaxNumRoomsH`(128) × `kMaxNumRoomsV`(64) | `GliderPRO/Sources/HouseLegal.c:648` |
| `kMaxSounds` = 64 = number of `kXxxSound` IDs (0…63) | `GliderPRO/Sources/Sound.c:15` vs `GliderPRO/Headers/GliderDefines.h:55-118` |
| `kDemoLength` = 6702 = 1117 × `sizeof(demoType)`(6) | `GliderPRO/Headers/GliderDefines.h:625`; verified against the `demo` 128 resource, §18.8 |

### 1.5 Define census

Only two of the 25 files in `GliderPRO/Headers/` carry defines; the other 23 have zero.
There is also a **third** define-bearing header outside that directory —
`GliderPRO/Prefix.h`, the MPW/CodeWarrior prefix file — which is *not* included in the
1327 total below.

| File | Active `#define`s |
|---|---|
| `GliderPRO/Headers/GliderDefines.h` | 566 (625 lines − 50 blank − 9 header-comment; plus 3 commented-out build flags at :11-13) |
| `GliderPRO/Headers/Externs.h` | 197 |
| `GliderPRO/Sources/*.c` (46 files) | 564 |
| **Total** | **1327** |
| *(excluded)* `GliderPRO/Prefix.h` | 7 — `TARGET_CARBON` 1, `ACCESSOR_CALLS_ARE_FUNCTIONS` 1, `OPAQUE_TOOLBOX_STRUCTS` 1, `OPAQUE_UPP_TYPES` 1, `forCarbon` 1, `BUILDING_RUN_LINKED_IN` **0**, `DEBUG` 1 (`:1-7`) |

The complete alphabetical index of all 1327 is in **Appendix A** at the end of this
document; §§2-21 group them by meaning. `Prefix.h`'s 7 are build-target switches, not game
constants, but a port that reads the sources needs to know `TARGET_CARBON` and `DEBUG` are
both on and `BUILDING_RUN_LINKED_IN` is off.
---

## 2. Build-configuration macros

All in `GliderPRO/Headers/GliderDefines.h:11-16`. The three commented-out lines are shown
because the source *retains* the `#ifdef` blocks that depend on them, and a port must
decide which branch is canonical.

| Macro | State as shipped | Site | Effect when defined |
|---|---|---|---|
| `CREATEDEMODATA` | **commented out** | `GliderPRO/Headers/GliderDefines.h:11` | `demoData` becomes a 2000-entry write buffer (`GliderPRO/Sources/StructuresInit2.c:288`) instead of being loaded from the `demo` 128 resource; `LogDemoKey()` calls in `GetInput()` become live (`GliderPRO/Sources/Input.c:288` etc.); `Play.c:217` dumps the recording via `DumpToResEditFile((Ptr)demoData, sizeof(demoType) * (long)demoIndex)` |
| `COMPILEDEMO` | **commented out** | `GliderPRO/Headers/GliderDefines.h:12` | `ReadHouse()` hard-rejects any house whose data fork is not exactly **16526** bytes and whose `nRooms != 45` and whose `firstRoom != 0` (`GliderPRO/Sources/HouseIO.c:~330`) — i.e. locks the build to the demo house |
| `CAREFULDEBUG` | **commented out** | `GliderPRO/Headers/GliderDefines.h:13` | extra assertion paths |
| `COMPILENOCP` | **defined** | `GliderPRO/Headers/GliderDefines.h:14` | "no copy protection": disables the `Validate.c` site-licence/serial check (`kEncryptMask 0x05218947`, `kLegalVolumeCreation 0xAA2D3E41`) |
| `COMPILEQT` | **defined** | `GliderPRO/Headers/GliderDefines.h:15` | enables the QuickTime background-movie support (`Movie theMovie; Rect movieRect; Boolean hasMovie, tvInRoom;` at `GliderPRO/Headers/GliderVars.h:42-44`) |
| `BUILD_ARCADE_VERSION` | **1** | `GliderPRO/Headers/GliderDefines.h:16` | during demo playback, *any* control key exits the demo instead of the Command key opening menus (`GliderPRO/Sources/Input.c:~200`) |

`16526 = 866 + 348 × 45`, which is a second independent confirmation of the header/room
sizes in §22.

**Port note:** `COMPILENOCP`, `COMPILEQT` and `BUILD_ARCADE_VERSION` are the shipped
configuration. A faithful Go port should behave as if all three are on, `CREATEDEMODATA`
and `COMPILEDEMO` off.

---

## 3. Error and alert codes

### 3.1 `kErr*` — fatal (RedAlert) codes

`RedAlert(short errorType)` puts up ALRT `rDeathAlertID` = 170 with strings from
`STR# rErrTitleID` = 170 / `STR# rErrMssgID` = 171 indexed by the code
(`GliderPRO/Sources/Utilities.c:137-139`), then quits. All 13:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kErrUnnaccounted` | 1 | `GliderPRO/Headers/Externs.h:183` | catch-all (note the author's typo: two n's) |
| `kErrNoMemory` | 2 | `GliderPRO/Headers/Externs.h:184` | a `NewPtr`/`NewHandle` returned nil — every allocation in `CreatePointers()` maps here |
| `kErrDialogDidntLoad` | 3 | `GliderPRO/Headers/Externs.h:185` | `GetNewDialog` failed |
| `kErrFailedResourceLoad` | 4 | `GliderPRO/Headers/Externs.h:186` | `GetResource` failed |
| `kErrFailedGraphicLoad` | 5 | `GliderPRO/Headers/Externs.h:187` | `GetPicture` failed |
| `kErrFailedOurDirect` | 6 | `GliderPRO/Headers/Externs.h:188` | could not find/create the app's own directory |
| `kErrFailedValidation` | 7 | `GliderPRO/Headers/Externs.h:189` | copy-protection failure (dead when `COMPILENOCP`) |
| `kErrNeedSystem7` | 8 | `GliderPRO/Headers/Externs.h:190` | Gestalt says < System 7 |
| `kErrFailedGetDevice` | 9 | `GliderPRO/Headers/Externs.h:191` | `GetMainDevice`/`GetGDevice` failed |
| `kErrFailedMemoryOperation` | 10 | `GliderPRO/Headers/Externs.h:192` | `BlockMove`/handle op failed |
| `kErrFailedCatSearch` | 11 | `GliderPRO/Headers/Externs.h:193` | `PBCatSearch` failed while hunting houses |
| `kErrNeedColorQD` | 12 | `GliderPRO/Headers/Externs.h:194` | no Color QuickDraw |
| `kErrNeed16Or256Colors` | 13 | `GliderPRO/Headers/Externs.h:195` | screen depth cannot be set to 8-bit or 4-bit-grey |

### 3.2 `kYellow*` — non-fatal (YellowAlert) codes

`YellowAlert(short whichAlert, short identifier)` puts up ALRT `kYellowAlert` = 1006
(`GliderPRO/Sources/HouseIO.c:642`) with the message from `STR# 1006` ("Yellow Alerts")
indexed by the code, and continues. All 24, in order:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kYellowUnaccounted` | 1 | `GliderPRO/Headers/GliderDefines.h:18` | catch-all |
| `kYellowFailedResOpen` | 2 | `GliderPRO/Headers/GliderDefines.h:19` | `FSpOpenResFile` failed on a house |
| `kYellowFailedResAdd` | 3 | `GliderPRO/Headers/GliderDefines.h:20` | `AddResource` failed |
| `kYellowFailedResCreate` | 4 | `GliderPRO/Headers/GliderDefines.h:21` | `FSpCreateResFile` failed |
| `kYellowNoHouses` | 5 | `GliderPRO/Headers/GliderDefines.h:22` | no `'gliH'` files found anywhere |
| `kYellowNewerVersion` | 6 | `GliderPRO/Headers/GliderDefines.h:23` | house `version >= kNewHouseVersion` (0x0300) |
| `kYellowNoBackground` | 7 | `GliderPRO/Headers/GliderDefines.h:24` | room's background PICT missing |
| `kYellowIllegalRoomNum` | 8 | `GliderPRO/Headers/GliderDefines.h:25` | link/target room index out of range |
| `kYellowNoBoundsRes` | 9 | `GliderPRO/Headers/GliderDefines.h:26` | custom-structure bounds resource missing |
| `kYellowScrapError` | 10 | `GliderPRO/Headers/GliderDefines.h:27` | clipboard failure in the editor |
| `kYellowNoMemory` | 11 | `GliderPRO/Headers/GliderDefines.h:28` | non-fatal out-of-memory |
| `kYellowFailedWrite` | 12 | `GliderPRO/Headers/GliderDefines.h:29` | house save failed |
| `kYellowNoMusic` | 13 | `GliderPRO/Headers/GliderDefines.h:30` | music `snd ` load failed |
| `kYellowFailedSound` | 14 | `GliderPRO/Headers/GliderDefines.h:31` | sound channel/`snd ` failure |
| `kYellowAppleEventErr` | 15 | `GliderPRO/Headers/GliderDefines.h:32` | Apple Event handling error |
| `kYellowOpenedOldHouse` | 16 | `GliderPRO/Headers/GliderDefines.h:33` | house `version < kHouseVersion` |
| `kYellowLostAllHouses` | 17 | `GliderPRO/Headers/GliderDefines.h:34` | house list became empty |
| `kYellowFailedSaveGame` | 18 | `GliderPRO/Headers/GliderDefines.h:35` | game save failed |
| `kYellowSavedTimeWrong` | 19 | `GliderPRO/Headers/GliderDefines.h:36` | saved game's `timeStamp` ≠ house's |
| `kYellowSavedVersWrong` | 20 | `GliderPRO/Headers/GliderDefines.h:37` | saved game's `version` ≠ `kSavedGameVersion` |
| `kYellowSavedRoomsWrong` | 21 | `GliderPRO/Headers/GliderDefines.h:38` | saved game's `nRooms` ≠ house's |
| `kYellowQTMovieNotLoaded` | 22 | `GliderPRO/Headers/GliderDefines.h:39` | QuickTime movie failed to open |
| `kYellowNoRooms` | 23 | `GliderPRO/Headers/GliderDefines.h:40` | house `nRooms < 1` |
| `kYellowCantOrderLinks` | 24 | `GliderPRO/Headers/GliderDefines.h:41` | link ordering failed in the editor |

### 3.3 Every ALRT / DLOG resource ID in the program

These are Mac Resource-Manager dialog IDs. A Go port replaces them with its own UI, but
they are listed exhaustively because they are the complete inventory of user-facing
modal interactions.

| ID | Constant | Site | Purpose |
|---|---|---|---|
| 130 | `kSwitchDepthAlert` | `GliderPRO/Sources/Environ.c:18` | offer to switch screen depth |
| 140 | `rFileErrorAlert` | `GliderPRO/Sources/FileError.c:14` | file-error alert |
| 140 | `rFileErrorStrings` | `GliderPRO/Sources/FileError.c:15` | its `STR#` |
| 150 | `kAboutDialogID` | `GliderPRO/Sources/About.c:34` | About box (DLOG 150, DITL 150, DLGX 150, ictb 150) |
| 160 | `kNewPrefsAlertID` | `GliderPRO/Sources/Prefs.c:23` | "new prefs created" |
| 160 | `kPrefsStringsID` | `GliderPRO/Sources/Prefs.c:22` | its `STR#` |
| 170 | `rDeathAlertID` | `GliderPRO/Sources/Utilities.c:137` | fatal RedAlert |
| 180 | `kSetMemoryAlert` | `GliderPRO/Sources/Environ.c:19` | "increase memory partition" |
| 181 | `kLowMemoryAlert` | `GliderPRO/Sources/Environ.c:20` | "low memory" |
| 1000 | `kLoadHouseDialogID` | `GliderPRO/Sources/SelectHouse.c:20` | house picker |
| 1001 | `kHouseInfoDialogID` | `GliderPRO/Sources/HouseInfo.c:14` | house info editor |
| 1002 | `kSaveChangesAlert` | `GliderPRO/Sources/HouseIO.c:20` | save/discard/cancel |
| 1003 | `kRoomInfoDialogID` | `GliderPRO/Sources/RoomInfo.c:18` | room info editor |
| 1004 | `kNewRoomAlert` | `GliderPRO/Sources/Map.c:23` | "create new room?" |
| 1005 | `kDeleteRoomAlert` | `GliderPRO/Sources/Room.c:16` | "delete this room?" |
| 1006 | `kYellowAlert` | `GliderPRO/Sources/HouseIO.c:642` | the YellowAlert host |
| 1007 | `kBlowerInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:18` | blower object editor |
| 1008 | `kNoMoreObjectsAlert` | `GliderPRO/Sources/ObjectAdd.c:15` | room already has `kMaxRoomObs` |
| 1009 | `kHouseBannerAlert` | `GliderPRO/Sources/Play.c:18` | shows the house banner at game start |
| 1010 | `kFurnitureInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:19` | furniture editor |
| 1011 | `kSwitchInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:20` | switch editor |
| 1012 | `kMainPrefsDialID` | `GliderPRO/Sources/Settings.c:17` | preferences (main) |
| 1013 | `kLightInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:21` | light editor |
| 1014 | `kApplianceInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:22` | appliance editor |
| 1015 | `kInvisBonusInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:23` | invisible-bonus editor |
| 1016 | `kOriginalArtDialogID` | `GliderPRO/Sources/RoomInfo.c:19` | custom-PICT room structure |
| 1017 | `kDisplayPrefsDialID` | `GliderPRO/Sources/Settings.c:18` | preferences (display) |
| 1018 | `kSoundPrefsDialID` | `GliderPRO/Sources/Settings.c:19` | preferences (sound) |
| 1019 | `kGreaseInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:24` | grease editor |
| 1020 | `kHighNameDialogID` | `GliderPRO/Sources/HighScores.c:25` | "enter your name" |
| 1021 | `kHighBannerDialogID` | `GliderPRO/Sources/HighScores.c:26` | "enter a banner" |
| 1022 | `kTransInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:25` | transport editor |
| 1023 | `kControlPrefsDialID` | `GliderPRO/Sources/Settings.c:20` | preferences (controls) |
| 1024 | `kBrainsPrefsDialID` | `GliderPRO/Sources/Settings.c:21` | preferences ("brains") |
| 1025 | `kResumeGameDial` | `GliderPRO/Sources/Menu.c:712` | new vs resume |
| 1026 | `kMasterDialogID` | `GliderPRO/Sources/Validate.c:18` | site-licence dialog |
| 1027 | `kEnemyInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:26` | enemy editor |
| 1028 | `kNoMoreSpecialAlert` | `GliderPRO/Sources/ObjectAdd.c:16` | too many of a limited object |
| 1029 | `kLockHouseAlert` | `GliderPRO/Sources/HouseInfo.c:23` | "lock this house?" |
| 1030 | `kNoSoundManager3Alert` | `GliderPRO/Sources/Sound.c:529` | Sound Manager 3 required |
| 1031 | `kNoPrintingAlert` | `GliderPRO/Sources/AppleEvents.c:14` | printing unsupported |
| 1032 | `kZeroScoresAlert` | `GliderPRO/Sources/HouseInfo.c:24` | "clear high scores?" |
| 1033 | `kFlowerInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:27` | flower editor |
| 1034 | `kTriggerInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:28` | trigger editor |
| 1035 | `kMicrowaveInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:29` | microwave editor |
| 1036 | `kNoPICTFoundAlert` | `GliderPRO/Sources/RoomInfo.c:20` | custom PICT missing |
| 1037 | `kNotInDemoAlert` | `GliderPRO/Sources/Menu.c:769` | feature disabled in demo |
| 1038 | `kNoMemForMusicAlert` | `GliderPRO/Sources/Music.c:413` | music won't fit |
| 1039 | `kNoMemForSoundsAlert` | `GliderPRO/Sources/Sound.c:518` | sounds won't fit |
| 1040 | `kChangesEffectAlert` | `GliderPRO/Sources/Settings.c:1476` | "changes take effect on relaunch" |
| 1041 | `kSaveGameAlert` | `GliderPRO/Sources/Input.c:385` | "save game?" |
| 1042 | `kSavingGameDial` | `GliderPRO/Sources/Input.c:19` | modeless "saving…" |
| 1042 | `kColorSwitchedAlert` | `GliderPRO/Sources/Events.c:48` | screen depth changed under us |
| 1043 | `kGoToDialogID` | `GliderPRO/Sources/House.c:18` | "go to room" |
| 1044 | `kSavedGameErrorAlert` | `GliderPRO/Sources/SavedGames.c:154` | saved-game load error |
| 1045 | `kCustPictInfoDialogID` | `GliderPRO/Sources/ObjectInfo.c:30` | custom-PICT object editor |
| 1046 | `kNoHighScoreAlert` | `GliderPRO/Sources/Menu.c:781` | no high scores yet |

Verified against the resource fork: `GliderPRO/Glider PRO.r` contains **ALRT × 26** with
IDs in 130…1046, **DLOG × 28** with IDs in 150…1045, and **DITL × 54** (one per ALRT and
one per DLOG). The counts match: 26 + 28 = 54.

---

## 4. Timing and frame rate

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kTicksPerFrame` | **2** | `GliderPRO/Headers/GliderDefines.h:533` | Mac ticks per game frame |
| `kIdleSplashTicks` | 7200L | `GliderPRO/Headers/GliderDefines.h:197` | idle→demo timeout; the comment says "2 minutes" |
| `kDemoLength` | 6702 | `GliderPRO/Headers/GliderDefines.h:625` | byte size of the `demo` 128 resource |
| `kLengthOfZap` | 30 | `GliderPRO/Headers/GliderDefines.h:546` | frames an electrical-outlet zap lasts |
| `kStartSparkle` | 4 | `GliderPRO/Headers/GliderDefines.h:545` | sparkle animation start frame / re-arm timer |
| `kZoomDelay` | 1 | `GliderPRO/Sources/DialogUtils.c:229`, `:283` | ticks between dialog-zoom steps |
| `kSteps` | 16 | `GliderPRO/Sources/DialogUtils.c:228`, `:282` | dialog-zoom step count |

**The frame rate.** A classic Mac `TickCount()` tick is 1/60.15 s. With
`kTicksPerFrame = 2` the nominal game rate is **60.15 / 2 = 30.07 fps**. The main loop
does not sleep; it spins until the tick target is reached, so on a fast machine the game
runs at exactly 30.07 fps and on a slow machine it drops frames of *simulation*, not of
rendering — i.e. the simulation is frame-locked, never time-scaled. A Go port must run
physics at a fixed 30.075 Hz step (or 2/60.15 s = 33.2502 ms) and must not use
delta-time integration, or every tuned velocity in this document changes meaning.

**`evenFrame`.** A global `Boolean evenFrame` toggles every frame and gates half-rate
behaviour. Places that depend on it:

| Site | Half-rate behaviour |
|---|---|
| `GliderPRO/Sources/Dynamics2.c` (ball) | `if (evenFrame) vVel++;` — ball gravity is 1 px/frame² at *half* rate, i.e. 0.5 px/frame² |
| `GliderPRO/Sources/Dynamics2.c` (drip) | `frame = 9 - frame;` only on `evenFrame` |
| `GliderPRO/Sources/Dynamics.c:~` (foil wear) | `if (evenFrame && foilTotal > 0) foilTotal--;` |
| `GliderPRO/Sources/Interactions.c:1770` | web twang only fires `if (hDesiredVel != 0 && evenFrame)` |

`kTicksPerFrame` also appears as a *divisor* in every dynamic-object delay conversion:
`((short)delay * 6) / kTicksPerFrame` — i.e. the on-disk `delay` byte is in units of
**6 ticks = 1/10 s**, converted to frames by dividing by `kTicksPerFrame`, giving
3 frames per delay unit. Sites: `GliderPRO/Sources/Dynamics3.c` for outlet
(`count = ((short)who->data.g.delay * 6) / kTicksPerFrame`), VCR, balloon
(`count = ((short)who->data.h.delay * 6) / kTicksPerFrame`), copter, dart, and fish
(`hVel = ((short)who->data.h.delay * 6) / kTicksPerFrame`).

Other timing constants that are not `#define`d but are hard literals — see §24.
---

## 5. Room and world geometry

### 5.1 The tile grid

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kNumTiles` | 8 | `GliderPRO/Headers/GliderDefines.h:496` | tile columns per room |
| `kTileWide` | 64 | `GliderPRO/Headers/GliderDefines.h:497` | tile width in pixels |
| `kTileHigh` | 322 | `GliderPRO/Headers/GliderDefines.h:498` | tile height = full room height |
| `kRoomWide` | 512 | `GliderPRO/Headers/GliderDefines.h:499` | comment: "kNumTiles * kTileWide" |
| `kFloorSupportTall` | 44 | `GliderPRO/Headers/GliderDefines.h:500` | height of the drawn floor-support strip below a room |
| `kVertLocalOffset` | 322 | `GliderPRO/Headers/GliderDefines.h:501` | comment: "kTileHigh - 39 (was 283, then 295)" — value shipped is 322 |
| `kMaxViewWidth` | 1536 | `GliderPRO/Headers/GliderDefines.h:267` | 3 × `kRoomWide`; widest visible strip |
| `kMaxViewHeight` | 1026 | `GliderPRO/Headers/GliderDefines.h:268` | 3 × 342; tallest visible strip |
| `kMapRoomHeight` | 20 | `GliderPRO/Headers/GliderDefines.h:246` | map-window room cell height |
| `kMapRoomWidth` | 32 | `GliderPRO/Headers/GliderDefines.h:247` | map-window room cell width |

A room is exactly **512 × 322** pixels of playfield. The coordinate origin is the room's
top-left. `kMaxViewWidth = 1536 = 3 × 512` and `kMaxViewHeight = 1026 = 3 × 342`, matching
the 3×3 neighbourhood (`kCentralRoom`…`kNorthWestRoom`, §5.5); 342 = 322 + 20 leaves room
for the scoreboard strip (`kScoreboardTall` = 20).

**Tile semantics — verified.** `roomType.tiles[8]` (§22.3) are **column indices into the
background PICT**, not tile-type enums:

```c
QSetRect(&src, 0, 0, kTileWide, kTileHigh);          // GliderPRO/Sources/RoomGraphics.c:241
QSetRect(&dest, 0, 0, kTileWide, kTileHigh);         // GliderPRO/Sources/RoomGraphics.c:242
...
src.left  = tiles[i] * kTileWide;                    // GliderPRO/Sources/RoomGraphics.c:246
src.right = src.left + kTileWide;
...
QOffsetRect(&dest, kTileWide, 0);                    // GliderPRO/Sources/RoomGraphics.c:251
```

So each background PICT is 512 × 322 = 8 columns of 64 px, and the room paints column
`tiles[i]` at destination column `i`. The 8 columns of the *indoor* backgrounds are
authored so that specific indices carry specific meanings, and the collision code hard-codes
those meanings:

| Tile value | Meaning as used by the physics | Evidence |
|---|---|---|
| 0 | left wall / solid-left column | `GliderPRO/Sources/Room.c:822` reads `tiles[0]`; `SetInitialTiles` sets `tiles[0] = 0` for indoor rooms |
| 1 | plain wall | default fill for indoor rooms, `GliderPRO/Sources/Room.c:~52` |
| 2 | **floor opening** — *only in `kDirt` rooms* | `GliderPRO/Sources/Interactions.c` `CheckEscapeDown`: tile is 2 or 3, inside `else if (thisBackground == kDirt)` at `:389` |
| 3 | **floor opening** (variant), same `kDirt` gate | same |
| 4 | (authored decoration; no special physics) | present in shipped houses, e.g. Sampler room 0 `tiles=(0,1,4,1,2,1,1,3)` |
| 5 | **ceiling opening** — *only in `kDirt` rooms* | `GliderPRO/Sources/Interactions.c` `CheckEscapeUp`: tile is 5 or 6, inside `else if (thisBackground == kDirt)` at `:255` |
| 6 | **ceiling opening** (variant), same `kDirt` gate | same |
| 7 | right wall / solid-right column | `GliderPRO/Sources/Room.c:823` reads `tiles[kNumTiles - 1]`; `SetInitialTiles` sets `tiles[7] = 7` |

`CheckRoofCollision` (`GliderPRO/Sources/Interactions.c:450-505`, banner at `:448`) gives
1/2/5/6 a *sloped-roof*
interpretation as well — see §5.4.

`GliderPRO/Sources/Room.c:993-996` tests whether **all 8** tiles are 0 to decide a room is
"blank".

### 5.2 `SetInitialTiles(short background, Boolean doRoom)` — default tile rows

`GliderPRO/Sources/Room.c:40-153`. Numbered to mirror the C control flow:

1. If `background >= kUserBackground` (3000): `for (i = 0; i < kNumTiles; i++) tiles[i] = i;`
   (identity — a user background is assumed to be an 8-column strip used left-to-right).
2. Else `switch (background)`:
   1. `kSimpleRoom`(2000), `kPaneledRoom`(2001), `kBasement`(2002), `kChildsRoom`(2003),
      `kAsianRoom`(2004), `kUnfinishedRoom`(2005), `kSwingersRoom`(2006),
      `kBathroom`(2007), `kLibrary`(2008): all 8 set to **1**, then
      `tiles[0] = 0;` and `tiles[kNumTiles - 1] = kNumTiles - 1;` (= 7).
      Net: `{0,1,1,1,1,1,1,7}`.
   2. `kSkywalk`(2010): identity `{0,1,2,3,4,5,6,7}`.
   3. `kField`(2013), `kGarden`(2009), `kDirt`(2011): all **0**.
   4. `kMeadow`(2012): all **1**.
   5. `kRoof`(2014): all **3**.
   6. `kSky`(2015): all **2**.
   7. `kStratosphere`(2016), `kStars`(2017): identity `{0,1,2,3,4,5,6,7}`.

Note that for outdoor backgrounds the "all 0 / all 1 / all 3 / all 2" values are *column
indices*, so an outdoor room repeats a single 64-px column 8 times. They are **not** the
reason the glider can fall through sky: `bottomOpen`/`topOpen` come from
`DoesRoomHaveFloor()` / `DoesRoomHaveCeiling()` (`GliderPRO/Sources/Room.c:924-932`),
which switch on the *background ID*, not on tiles — `kSky`/`kStratosphere`/`kStars` have
no floor, and `kGarden`/`kMeadow`/`kField`/`kRoof`/`kSky`/`kStratosphere`/`kStars` have no
ceiling. Tile values 2/3/5/6 only ever gate vertical escape inside a `kDirt` room
(§5.1, §10.6).

### 5.3 Collision limits

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kCeilingLimit` | 8 | `GliderPRO/Headers/GliderDefines.h:503` | y of a solid ceiling |
| `kFloorLimit` | 312 | `GliderPRO/Headers/GliderDefines.h:504` | y of a solid floor |
| `kRoofLimit` | 122 | `GliderPRO/Headers/GliderDefines.h:505` | y of the sloped-roof plane |
| `kLeftWallLimit` | 12 | `GliderPRO/Headers/GliderDefines.h:506` | x of a solid left wall |
| `kNoLeftWallLimit` | -24 | `GliderPRO/Headers/GliderDefines.h:507` | comment: "0 - (kGliderWide / 2)" — x at which the glider has fully left through an open left wall |
| `kRightWallLimit` | 500 | `GliderPRO/Headers/GliderDefines.h:508` | x of a solid right wall |
| `kNoRightWallLimit` | 536 | `GliderPRO/Headers/GliderDefines.h:509` | comment: "kRoomWide + (kGliderWide / 2)" |
| `kNoCeilingLimit` | -10 | `GliderPRO/Headers/GliderDefines.h:510` | y at which the glider has fully left through an open ceiling |
| `kNoFloorLimit` | 332 | `GliderPRO/Headers/GliderDefines.h:511` | y at which the glider has fully left through an open floor |

Observe the asymmetry: `kLeftWallLimit` = 12 but `kRightWallLimit` = 500, so the playable
interior is x ∈ [12, 500] — 488 px wide, i.e. 12 px of wall on each side of the 512-px
room. `kCeilingLimit` = 8 and `kFloorLimit` = 312 give a 304-px interior of the 322-px
room (8 top, 10 bottom).

`kNoFloorLimit` (332) = `kFloorLimit`(312) + 20 = `kFloorLimit` + `kGliderHigh`.
`kNoCeilingLimit` (-10) = −(`kGliderHigh` ÷ 2). These are the "fully exited" thresholds.

### 5.4 `CheckRoofCollision` — the sloped-roof formula

`GliderPRO/Sources/Interactions.c:450-505` (banner comment at `:448`). This is the only
place a tile value becomes a
*geometric* slope. Numbered pseudocode with the original names:

1. `offset = (thisGlider->dest.left + kHalfGliderWide) >> 6;` — the tile column under the
   glider's horizontal centre (`>> 6` = ÷ `kTileWide`).
2. If `offset < 0 || offset > 7`, return (no collision).
3. If `thisGlider->sliding`, return (already sliding along a surface).
4. `dx = (thisGlider->dest.left + kHalfGliderWide) - (offset << 6);` — the position
   *within* the 64-px tile, 0…63. (The source writes the subtraction, not `& 0x003F`;
   they are equivalent here only because step 2 has already guaranteed `offset >= 0`.)
5. `tileOver = thisTiles[offset];` then:
   * `tileOver == 1` (roof rising to the right): collide if `dx > (250 - dest.bottom)`.
   * `tileOver == 2`: collide if `dx > (186 - dest.bottom)`.
   * `tileOver == 5` (mirror image): collide if `(64 - dx) > (186 - dest.bottom)`.
   * `tileOver == 6` (mirror image): collide if `(64 - dx) > (250 - dest.bottom)`.
   * **any other tile value**: collide *unconditionally* (the `else` at
     `GliderPRO/Sources/Interactions.c:498-503` — a flat roof deck at y = `kRoofLimit`).
6. On collision the glider is **not** placed on the roof and `sliding` is never touched.
   All five bodies do the identical three things
   (`GliderPRO/Sources/Interactions.c:463-465`, `:473-475`, `:483-485`, `:493-495`,
   `:500-502`):
   `thisGlider->vVel = kFloorLimit - thisGlider->dest.bottom;` then
   `StartGliderFadingOut(thisGlider);` then
   `PlayPrioritySound(kFadeOutSound, kFadeOutPriority);` — i.e. the glider is retargeted
   at the floor line (312) and fades out, which is how you enter the house through the
   roof. See §24.2.3 for the derived line equations.

The magic numbers **250** and **186** are *not* behind a `#define`; they are the y-intercepts
of the two roof slopes (both slope 1, i.e. 45°). See §24.

### 5.5 3×3 room neighbourhood, direction and bump codes

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kCentralRoom` | 0 | `GliderPRO/Headers/GliderDefines.h:217` | the room the player is in |
| `kNorthRoom` | 1 | `GliderPRO/Headers/GliderDefines.h:218` | room above |
| `kNorthEastRoom` | 2 | `GliderPRO/Headers/GliderDefines.h:219` | above-right |
| `kEastRoom` | 3 | `GliderPRO/Headers/GliderDefines.h:220` | right |
| `kSouthEastRoom` | 4 | `GliderPRO/Headers/GliderDefines.h:221` | below-right |
| `kSouthRoom` | 5 | `GliderPRO/Headers/GliderDefines.h:222` | below |
| `kSouthWestRoom` | 6 | `GliderPRO/Headers/GliderDefines.h:223` | below-left |
| `kWestRoom` | 7 | `GliderPRO/Headers/GliderDefines.h:224` | left |
| `kNorthWestRoom` | 8 | `GliderPRO/Headers/GliderDefines.h:225` | above-left |
| `kRoomAbove` | 1 | `GliderPRO/Headers/GliderDefines.h:200` | link direction: up |
| `kRoomBelow` | 2 | `GliderPRO/Headers/GliderDefines.h:201` | link direction: down |
| `kRoomToRight` | 3 | `GliderPRO/Headers/GliderDefines.h:202` | link direction: right |
| `kRoomToLeft` | 4 | `GliderPRO/Headers/GliderDefines.h:203` | link direction: left |
| `kBumpUp` | 1 | `GliderPRO/Headers/GliderDefines.h:205` | editor: nudge object up |
| `kBumpDown` | 2 | `GliderPRO/Headers/GliderDefines.h:206` | nudge down |
| `kBumpRight` | 3 | `GliderPRO/Headers/GliderDefines.h:207` | nudge right |
| `kBumpLeft` | 4 | `GliderPRO/Headers/GliderDefines.h:208` | nudge left |
| `kAbove` | 1 | `GliderPRO/Headers/GliderDefines.h:210` | relative-position result |
| `kToRight` | 2 | `GliderPRO/Headers/GliderDefines.h:211` | relative-position result |
| `kBelow` | 3 | `GliderPRO/Headers/GliderDefines.h:212` | relative-position result |
| `kToLeft` | 4 | `GliderPRO/Headers/GliderDefines.h:213` | relative-position result |
| `kBottomCorner` | 5 | `GliderPRO/Headers/GliderDefines.h:214` | relative-position result |
| `kTopCorner` | 6 | `GliderPRO/Headers/GliderDefines.h:215` | relative-position result |

`kCentralRoom` is special-cased in `GliderPRO/Sources/RoomGraphics.c:175`: the global
`thisTiles[]` (used by all collision code) is only refreshed
`if (where == kCentralRoom)` — `thisTiles[i] = (*thisHouse)->rooms[who].tiles[i];`

### 5.6 House-coordinate space

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kMaxNumRoomsH` | 128 | `GliderPRO/Headers/GliderDefines.h:543` | horizontal room slots ("suites") |
| `kMaxNumRoomsV` | 64 | `GliderPRO/Headers/GliderDefines.h:544` | vertical room slots ("floors") |
| `kRoomsTimesSuites` | 8192 | `GliderPRO/Sources/HouseLegal.c:648` | 128 × 64, size of the duplicate-detection array |
| `kNumUndergroundFloors` | 8 | `GliderPRO/Headers/GliderDefines.h:535` | floors below ground level |
| `kMapGroundValue` | 56 | `GliderPRO/Sources/Map.c:22` | the y-flip origin for the map: `floor` 56 is map row 0 (**not** "ground level" despite the name) |

Room position is `(floor, suite)` in `roomType` (§22.3). `floor` counts **upwards** — a
larger `floor` is physically higher — and the map view flips it:
`GliderPRO/Sources/Map.c:58` computes the map row as
`v = kMapGroundValue - thisRoom->floor`, and `GliderPRO/Sources/Map.c:139`/`:224` invert it
as `floor = kMapGroundValue - (i + mapTopRoom)`. Since a smaller row index draws higher on
screen, `floor = 56` is the **topmost** floor and negative floors are the basements.

The legal range is fixed by `ValidateRoomNumbers`
(`GliderPRO/Sources/HouseLegal.c:804-823`), which empties any room outside it:

```c
	if (((*thisHouse)->rooms[i].floor > 56) || ((*thisHouse)->rooms[i].floor < -7))   // :804-805
	if (((*thisHouse)->rooms[i].suite >= 128) || ((*thisHouse)->rooms[i].suite < 0))  // :814-815
```

so `floor` ∈ **[−7, 56]** (64 values = `kMaxNumRoomsV`) and `suite` ∈ **[0, 127]**
(128 values = `kMaxNumRoomsH`). Ground level is `floor == 0`; `kNumUndergroundFloors` = 8
covers `floor` 0 down to −7, which is exactly the negative end of the legal range and
exactly the `+ kNumUndergroundFloors` bias applied when packing into `data.d.where` /
`data.e.where` (§22.6). Both 56 and −7 are bare literals in `HouseLegal.c` — neither is
written as `kMaxNumRoomsV - kNumUndergroundFloors` or `-(kNumUndergroundFloors - 1)`.

Empirically, across all 22 shipped houses the observed `floor` values run from **−7**
(*Rainbow's End*, *Slumberland*, *Titanic*) to **39** (*SpacePods*), heavily concentrated
on 0-2; observed `suite` values span the whole legal **0…127**. Most houses cluster their
suites around 64 (the middle of 0…127) because that is where the editor starts a new
house — e.g. *Empty House* 62-70, *Sampler* 66-67, *Demo House* 62-71 — but *SpacePods*
(110-127) and *Leviathan* (0-94) show that is only a convention. `suite == kRoomIsEmpty`
(**−1**, `GliderPRO/Headers/GliderDefines.h:525`) marks a deleted room and is the sentinel
every validator tests first.

`GliderPRO/Sources/HouseLegal.c:653`:
`pidgeonHoles = (char *)NewPtrClear(sizeof(char) * kRoomsTimesSuites);` — one byte per
(floor, suite) cell to detect two rooms claiming the same coordinate.

### 5.7 Vertical placement constants (object "snap" heights)

These are absolute y coordinates in room space, used when the editor drops a new object.
All from `GliderPRO/Headers/GliderDefines.h:467-494`.

| Name | Value | Site | Object it positions |
|---|---|---|---|
| `kFloorVentTop` | 305 | `:467` | `kFloorVent` sits with its top at 305 (bottom 316) |
| `kCeilingVentTop` | 8 | `:468` | `kCeilingVent` |
| `kFloorBlowerTop` | 304 | `:469` | `kFloorBlower` |
| `kCeilingBlowerTop` | 5 | `:470` | `kCeilingBlower` |
| `kSewerGrateTop` | 303 | `:471` | `kSewerGrate` |
| `kCeilingTransTop` | 6 | `:472` | `kCeilingTrans` (ceiling duct) |
| `kFloorTransTop` | 302 | `:473` | `kFloorTrans` (floor duct) |
| `kStairsTop` | 28 | `:474` | `kUpStairs` / `kDownStairs` |
| `kCounterBottom` | 304 | `:475` | `kCounter` bottom edge |
| `kDresserBottom` | 293 | `:476` | `kDresser` bottom edge |
| `kCeilingLightTop` | 4 | `:477` | `kCeilingLight` |
| `kHipLampTop` | 23 | `:478` | `kHipLamp` |
| `kDecoLampTop` | 91 | `:479` | `kDecoLamp` |
| `kFlourescentTop` | 12 | `:480` | `kFlourescent` (author's spelling) |
| `kTrackLightTop` | 5 | `:481` | `kTrackLight` |
| `kDoorInTop` | 0 | `:483` | interior door, top |
| `kDoorInLfLeft` | 0 | `:484` | interior door left variant, x |
| `kDoorInRtLeft` | 368 | `:485` | interior door right variant, x (512 − 144) |
| `kDoorExTop` | 0 | `:486` | exterior door, top |
| `kDoorExLfLeft` | 0 | `:487` | exterior door left variant, x |
| `kDoorExRtLeft` | 496 | `:488` | exterior door right variant, x (512 − 16) |
| `kWindowInTop` | 64 | `:489` | interior window, top |
| `kWindowInLfLeft` | 0 | `:490` | interior window left variant, x |
| `kWindowInRtLeft` | 492 | `:491` | interior window right variant, x (512 − 20) |
| `kWindowExTop` | 64 | `:492` | exterior window, top |
| `kWindowExLfLeft` | 0 | `:493` | exterior window left variant, x |
| `kWindowExRtLeft` | 496 | `:494` | exterior window right variant, x (512 − 16) |

Additional placement constants living in `ObjectAdd.c` / `ObjectDraw*.c`:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kMouseholeBottom` | 295 | `GliderPRO/Sources/ObjectAdd.c:19` | mousehole bottom edge |
| `kFireplaceBottom` | 297 | `GliderPRO/Sources/ObjectAdd.c:20` | fireplace bottom edge |
| `kManholeSits` | 322 | `GliderPRO/Sources/ObjectAdd.c:21` | manhole y (= `kTileHigh`, i.e. exactly on the floor plane) |
| `kGrecoVentTop` | 303 | `GliderPRO/Sources/ObjectAdd.c:22` | Greco vent top |
| `kSewerBlowerTop` | 292 | `GliderPRO/Sources/ObjectAdd.c:23` | sewer blower top |
| `kMailboxBase` | 296 | `GliderPRO/Sources/ObjectDraw2.c:105` | mailbox baseline |
| `kTikiPoleBase` | 300 | `GliderPRO/Sources/ObjectDraw.c:71` | tiki pole baseline |
| `kTableBaseTop` | 296 | `GliderPRO/Sources/ObjectDraw.c:149`, `:775` | table leg top |
| `kTableShadowTop` | 312 | `GliderPRO/Sources/ObjectDraw.c:150`, `:776` | = `kFloorLimit`; table shadow y |
| `kTableShadowOffset` | 12 | `GliderPRO/Sources/ObjectDraw.c:151`, `:777` | shadow x skew |
| `kStoolBase` | 304 | `GliderPRO/Sources/ObjectDraw.c:888` | stool baseline |

### 5.8 Fixed-capacity maxima

Every one of these sizes a `NewPtr`-allocated fixed array in
`CreatePointers()` (`GliderPRO/Sources/StructuresInit2.c:187-299`). Exceeding them is
either an editor-time alert or a silent `return (-1)`.

| Name | Value | Site | Sizes | Overflow behaviour |
|---|---|---|---|---|
| `kMaxScores` | 10 | `GliderPRO/Headers/GliderDefines.h:249` | `scoresType.names/scores/timeStamps/levels` | high scores list truncates |
| `kMaxRoomObs` | 24 | `GliderPRO/Headers/GliderDefines.h:250` | `roomType.objects[24]` — **on-disk** | ALRT `kNoMoreObjectsAlert` 1008 |
| `kMaxSparkles` | 3 | `GliderPRO/Headers/GliderDefines.h:251` | `sparkles[3]` | oldest reused |
| `kNumSparkleModes` | 5 | `GliderPRO/Headers/GliderDefines.h:252` | sparkle animation frames | — |
| `kMaxFlyingPts` | 3 | `GliderPRO/Headers/GliderDefines.h:253` | `flyingPoints[3]` (score popups) | oldest reused |
| `kMaxFlyingPointsLoop` | 24 | `GliderPRO/Headers/GliderDefines.h:254` | frames a score popup lives | — |
| `kMaxCandles` | 20 | `GliderPRO/Headers/GliderDefines.h:255` | `flames[20]` | extra candles don't animate |
| `kMaxTikis` | 8 | `GliderPRO/Headers/GliderDefines.h:256` | `tikiFlames[8]` | as above |
| `kMaxCoals` | 8 | `GliderPRO/Headers/GliderDefines.h:257` | `bbqCoals[8]` | as above |
| `kMaxPendulums` | 8 | `GliderPRO/Headers/GliderDefines.h:258` | `pendulums[8]` | as above |
| `kMaxHotSpots` | 56 | `GliderPRO/Headers/GliderDefines.h:259` | `hotSpots[56]` | `AddActiveRect` returns −1 (`GliderPRO/Sources/ObjectRects.c:~278`) |
| `kMaxSavedMaps` | 24 | `GliderPRO/Headers/GliderDefines.h:260` | `savedMaps[24]` (torn-background GWorlds) | — |
| `kMaxRubberBands` | 2 | `GliderPRO/Headers/GliderDefines.h:261` | `bands[2]` — only 2 bands in flight | new shot refused |
| `kMaxGrease` | 16 | `GliderPRO/Headers/GliderDefines.h:262` | `grease[16]` | — |
| `kMaxStars` | 4 | `GliderPRO/Headers/GliderDefines.h:263` | `theStars[4]` | — |
| `kMaxShredded` | 4 | `GliderPRO/Headers/GliderDefines.h:264` | `shreds[4]` | — |
| `kMaxDynamicObs` | 18 | `GliderPRO/Headers/GliderDefines.h:265` | `dinahs[18]` | `AddDynamicObject` returns −1 (`GliderPRO/Sources/Dynamics3.c:194`, `:547`) |
| `kMaxMasterObjects` | 216 | `GliderPRO/Headers/GliderDefines.h:266` | `masterObjects[216]`; comment "kMaxRoomObs * 9" | — |
| `kMaxTempManholes` | 8 | `GliderPRO/Sources/Objects.c:12` | `Rect tempManholes[8]` (`GliderPRO/Sources/Objects.c:72`) | guarded `if (numTempManholes < kMaxTempManholes)` (`GliderPRO/Sources/Objects.c:355`) |
| `kMaxTriggers` | 16 | `GliderPRO/Sources/Triggers.c:12` | `trigType triggers[16]` (`GliderPRO/Sources/Triggers.c:28`) | loops at `:65`, `:83`, `:202` |
| `kMaxGarbageRects` | 48 | `GliderPRO/Sources/Render.c:20` | `work2MainRects[48]`, `back2WorkRects[48]` (`GliderPRO/Sources/Render.c:35-36`) | guarded `< (kMaxGarbageRects - 1)` at `:67`, `:86`, `:105` — i.e. effective capacity 47 |
| `kMaxSounds` | 64 | `GliderPRO/Sources/Sound.c:15` | `theSoundData[64]`, `soundLoaded[64]` | last slot reserved (§16.4) |
| `kMaxMusic` | 7 | `GliderPRO/Sources/Music.c:16` | music `snd ` buffers | — |
| `kMaxExtraHouses` | 8 | `GliderPRO/Sources/SelectHouse.c:35` | extra house search dirs | — |
| `kMaxDirectories` | 32 | `GliderPRO/Sources/SelectHouse.c:559` | house-search recursion breadth | — |
| `kMaxColumnsWide` | 96 | `GliderPRO/Sources/Transitions.c:20` | `Rect columnRects[96]` + `short columnProgress[96]` (`:23-24`) in **`PourScreenOn`**, the sand-pour transition — *not* the screen wipe (`WipeScreenOn` uses `kWipeRectThick` 4 at `:76`); with `kChipHigh` 20 / `kChipWide` 16 at `:21-22` | — |
| `kMaxSoundTriggers` | 1 | `GliderPRO/Sources/ObjectAdd.c:17` | at most 1 `kSoundTrigger` per room | ALRT `kNoMoreSpecialAlert` 1028 |
| `kMaxStairs` | 1 | `GliderPRO/Sources/ObjectAdd.c:18` | at most 1 staircase per room | ALRT 1028 |
| `kNumSrcRects` | 144 (`0x90`) | `GliderPRO/Headers/GliderDefines.h:437` | `srcRects[144]`, indexed by object `what` | see §6.3 |

---

## 6. Object type IDs

The `objectType.what` field (a **big-endian int16** on disk, §22.4) holds one of these.
The high nibble is the *class* and the low nibble is the member — a deliberate encoding:
class 0 = blowers, 1 = furniture, 2 = prizes, 3 = transport, 4 = switches, 5 = lights,
6 = appliances, 7 = enemies, 8 = clutter. `0x00`, `0x20`, `0x30`, `0x4A`-`0x50`,
`0x59`-`0x60`, `0x6F`-`0x70`, `0x7A`-`0x80` are unused.

`objectType.what` selects which arm of the 10-byte `data` union is meaningful (§7), which
`srcRects[]` entry gives its size (§6.2), and which hot-spot action(s) it registers (§8).

### 6.1 Complete object ID table

| Name | Hex | Dec | Site | Union arm | Notes |
|---|---|---|---|---|---|
| `kFloorVent` | `0x01` | 1 | `GliderPRO/Headers/GliderDefines.h:311` | `a` blower | lifts glider (`kLiftIt`) |
| `kCeilingVent` | `0x02` | 2 | `:312` | `a` | drops glider (`kDropIt`) |
| `kFloorBlower` | `0x03` | 3 | `:313` | `a` | lift, switchable |
| `kCeilingBlower` | `0x04` | 4 | `:314` | `a` | drop, switchable |
| `kSewerGrate` | `0x05` | 5 | `:315` | `a` | lift |
| `kLeftFan` | `0x06` | 6 | `:316` | `a` | `kPushItLeft` |
| `kRightFan` | `0x07` | 7 | `:317` | `a` | `kPushItRight` |
| `kTaper` | `0x08` | 8 | `:318` | `a` | tall candle; flame burns glider |
| `kCandle` | `0x09` | 9 | `:319` | `a` | candle |
| `kStubby` | `0x0A` | 10 | `:320` | `a` | short candle |
| `kTiki` | `0x0B` | 11 | `:321` | `a` | tiki torch |
| `kBBQ` | `0x0C` | 12 | `:322` | `a` | barbecue; coals |
| `kInvisBlower` | `0x0D` | 13 | `:323` | `a` | invisible lift/push |
| `kGrecoVent` | `0x0E` | 14 | `:324` | `a` | decorative vent, lifts |
| `kSewerBlower` | `0x0F` | 15 | `:325` | `a` | sewer blower |
| `kLiftArea` | `0x10` | 16 | `:326` | `a` | arbitrary-size lift zone |
| `kTable` | `0x11` | 17 | `:328` | `b` furniture | solid, thickness `kTableThick` 8 |
| `kShelf` | `0x12` | 18 | `:329` | `b` | solid, `kShelfThick` 6 |
| `kCabinet` | `0x13` | 19 | `:330` | `b` | solid box |
| `kFilingCabinet` | `0x14` | 20 | `:331` | `b` | solid |
| `kWasteBasket` | `0x15` | 21 | `:332` | `b` | solid |
| `kMilkCrate` | `0x16` | 22 | `:333` | `b` | solid |
| `kCounter` | `0x17` | 23 | `:334` | `b` | solid, bottom at `kCounterBottom` 304 |
| `kDresser` | `0x18` | 24 | `:335` | `b` | solid, bottom at `kDresserBottom` 293 |
| `kDeckTable` | `0x19` | 25 | `:336` | `b` | solid |
| `kStool` | `0x1A` | 26 | `:337` | `b` | solid, `kStoolThick` 25 |
| `kTrunk` | `0x1B` | 27 | `:338` | `b` | solid |
| `kInvisObstacle` | `0x1C` | 28 | `:339` | `b` | invisible solid |
| `kManhole` | `0x1D` | 29 | `:340` | `b` | manhole cover; `kManholeThruFloor` PICT 3957 |
| `kBooks` | `0x1E` | 30 | `:341` | `b` | solid |
| `kInvisBounce` | `0x1F` | 31 | `:342` | `b` | invisible bouncer (`kBounceIt`) |
| `kRedClock` | `0x21` | 33 | `:344` | `c` bonus | +`kRedClockPoints` 100 |
| `kBlueClock` | `0x22` | 34 | `:345` | `c` | +`kBlueClockPoints` 300 |
| `kYellowClock` | `0x23` | 35 | `:346` | `c` | +`kYellowClockPoints` 500 |
| `kCuckoo` | `0x24` | 36 | `:347` | `c` | +`kCuckooClockPoints` 1000, stops pendulum |
| `kPaper` | `0x25` | 37 | `:348` | `c` | +1 life (`mortals++`) |
| `kBattery` | `0x26` | 38 | `:349` | `c` | +`kBatterySupply` 50 |
| `kBands` | `0x27` | 39 | `:350` | `c` | +`kBandsSupply` 8 |
| `kGreaseRt` | `0x28` | 40 | `:351` | `c` | grease spill, spreads right; `length` used |
| `kGreaseLf` | `0x29` | 41 | `:352` | `c` | grease spill, spreads left |
| `kFoil` | `0x2A` | 42 | `:353` | `c` | +`kFoilSupply` 8, enters foil mode |
| `kInvisBonus` | `0x2B` | 43 | `:354` | `c` | + `data.c.points` (author-specified) |
| `kStar` | `0x2C` | 44 | `:355` | `c` | +`kStarPoints` 5000, decrements `numStarsRemaining` |
| `kSparkle` | `0x2D` | 45 | `:356` | `c` | pure decoration (no-op in `HandleRewards`) |
| `kHelium` | `0x2E` | 46 | `:357` | `c` | −`kHeliumSupply` 150 (negative battery) |
| `kSlider` | `0x2F` | 47 | `:358` | `c` | no-op in `HandleRewards`; editor helper |
| `kUpStairs` | `0x31` | 49 | `:360` | `d` transport | vertical transit up |
| `kDownStairs` | `0x32` | 50 | `:361` | `d` | vertical transit down |
| `kMailboxLf` | `0x33` | 51 | `:362` | `d` | `kMailItLeft` |
| `kMailboxRt` | `0x34` | 52 | `:363` | `d` | `kMailItRight` |
| `kFloorTrans` | `0x35` | 53 | `:364` | `d` | `kDuctItDown` |
| `kCeilingTrans` | `0x36` | 54 | `:365` | `d` | `kDuctItUp` |
| `kDoorInLf` | `0x37` | 55 | `:366` | `d` | interior door, left |
| `kDoorInRt` | `0x38` | 56 | `:367` | `d` | interior door, right |
| `kDoorExRt` | `0x39` | 57 | `:368` | `d` | exterior door, right |
| `kDoorExLf` | `0x3A` | 58 | `:369` | `d` | exterior door, left |
| `kWindowInLf` | `0x3B` | 59 | `:370` | `d` | interior window, left |
| `kWindowInRt` | `0x3C` | 60 | `:371` | `d` | interior window, right |
| `kWindowExRt` | `0x3D` | 61 | `:372` | `d` | exterior window, right |
| `kWindowExLf` | `0x3E` | 62 | `:373` | `d` | exterior window, left |
| `kInvisTrans` | `0x3F` | 63 | `:374` | `d` | invisible transport, `tall` field used |
| `kDeluxeTrans` | `0x40` | 64 | `:375` | `d` | arbitrary-size transport, `tall`+`wide` used |
| `kLightSwitch` | `0x41` | 65 | `:377` | `e` switch | toggles lights |
| `kMachineSwitch` | `0x42` | 66 | `:378` | `e` | toggles appliances |
| `kThermostat` | `0x43` | 67 | `:379` | `e` | toggles blowers |
| `kPowerSwitch` | `0x44` | 68 | `:380` | `e` | master power |
| `kKnifeSwitch` | `0x45` | 69 | `:381` | `e` | knife switch |
| `kInvisSwitch` | `0x46` | 70 | `:382` | `e` | invisible switch |
| `kTrigger` | `0x47` | 71 | `:383` | `e` | small proximity trigger (`kTriggerIt`) |
| `kLgTrigger` | `0x48` | 72 | `:384` | `e` | 48×48 proximity trigger |
| `kSoundTrigger` | `0x49` | 73 | `:385` | `e` | plays the house's custom `snd ` (`kSoundIt`) |
| `kCeilingLight` | `0x51` | 81 | `:387` | `f` light | light |
| `kLightBulb` | `0x52` | 82 | `:388` | `f` | light |
| `kTableLamp` | `0x53` | 83 | `:389` | `f` | light |
| `kHipLamp` | `0x54` | 84 | `:390` | `f` | light, top at 23 |
| `kDecoLamp` | `0x55` | 85 | `:391` | `f` | light, top at 91 |
| `kFlourescent` | `0x56` | 86 | `:392` | `f` | light, `length` used |
| `kTrackLight` | `0x57` | 87 | `:393` | `f` | light, `length` used, spacing 64 |
| `kInvisLight` | `0x58` | 88 | `:394` | `f` | invisible light |
| `kShredder` | `0x61` | 97 | `:396` | `g` appliance | shreds the glider (`kShredIt`) |
| `kToaster` | `0x62` | 98 | `:397` | `g` | launches toast; `height` = launch height |
| `kMacPlus` | `0x63` | 99 | `:398` | `g` | animated Mac |
| `kGuitar` | `0x64` | 100 | `:399` | `g` | `kStrumIt` |
| `kTV` | `0x65` | 101 | `:400` | `g` | animated TV; QuickTime host |
| `kCoffee` | `0x66` | 102 | `:401` | `g` | coffee maker |
| `kOutlet` | `0x67` | 103 | `:402` | `g` | zaps (`kBurnIt`-ish); `kLengthOfZap` |
| `kVCR` | `0x68` | 104 | `:403` | `g` | animated VCR |
| `kStereo` | `0x69` | 105 | `:404` | `g` | plays music |
| `kMicrowave` | `0x6A` | 106 | `:405` | `g` | `kMicrowaveIt`; `byte0` = kill mask |
| `kCinderBlock` | `0x6B` | 107 | `:406` | `g` | solid |
| `kFlowerBox` | `0x6C` | 108 | `:407` | `g` | solid |
| `kCDs` | `0x6D` | 109 | `:408` | `g` | decoration |
| `kCustomPict` | `0x6E` | 110 | `:409` | `g` | draws PICT `data.g.height` (ID reuse!) |
| `kBalloon` | `0x71` | 113 | `:411` | `h` enemy | rises, pops |
| `kCopterLf` | `0x72` | 114 | `:412` | `h` | helicopter drifting left |
| `kCopterRt` | `0x73` | 115 | `:413` | `h` | helicopter drifting right |
| `kDartLf` | `0x74` | 116 | `:414` | `h` | dart flying left |
| `kDartRt` | `0x75` | 117 | `:415` | `h` | dart flying right |
| `kBall` | `0x76` | 118 | `:416` | `h` | bouncing ball; `length` = bounce height |
| `kDrip` | `0x77` | 119 | `:417` | `h` | dripping faucet; `length` = fall distance |
| `kFish` | `0x78` | 120 | `:418` | `h` | leaping fish |
| `kCobweb` | `0x79` | 121 | `:419` | `h` | web (`kWebIt`) |
| `kOzma` | `0x81` | 129 | `:421` | `i` clutter | decoration |
| `kMirror` | `0x82` | 130 | `:422` | `i` | reflects the glider |
| `kMousehole` | `0x83` | 131 | `:423` | `i` | decoration, bottom 295 |
| `kFireplace` | `0x84` | 132 | `:424` | `i` | decoration, bottom 297 |
| `kFlower` | `0x85` | 133 | `:425` | `i` | decoration; **`srcRects[0x85]` is never initialised** (§6.3) |
| `kWallWindow` | `0x86` | 134 | `:426` | `i` | decoration |
| `kBear` | `0x87` | 135 | `:427` | `i` | decoration |
| `kCalendar` | `0x88` | 136 | `:428` | `i` | decoration; month from `STR# kMonthStringID` 1005 |
| `kVase1` | `0x89` | 137 | `:429` | `i` | decoration |
| `kVase2` | `0x8A` | 138 | `:430` | `i` | decoration |
| `kBulletin` | `0x8B` | 139 | `:431` | `i` | decoration |
| `kCloud` | `0x8C` | 140 | `:432` | `i` | decoration (has a mask) |
| `kFaucet` | `0x8D` | 141 | `:433` | `i` | decoration (drip source) |
| `kRug` | `0x8E` | 142 | `:434` | `i` | decoration |
| `kChimes` | `0x8F` | 143 | `:435` | `i` | wind chimes (`kChimeIt`) |

(Line numbers `:313`-`:445` above are all in `GliderPRO/Headers/GliderDefines.h`; the
declarations run `:311`-`:435` in the file with blank/comment lines interleaved — the
exact per-name lines are as tabulated.)

### 6.2 Tool-palette grouping

The editor groups objects into 9 tool "modes", each with a first/last index and a base
object ID. From `GliderPRO/Sources/Tools.c:20-49` and
`GliderPRO/Headers/GliderDefines.h:270-280`:

| Mode constant | Value | Site | First | Last | Base object ID |
|---|---|---|---|---|---|
| `kSelectTool` | 0 | `GliderPRO/Headers/GliderDefines.h:270` | — | — | (arrow) |
| `kBlowerMode` | 1 | `:272` | `kFirstBlower` 1 | `kLastBlower` 15 | `kBlowerBase` 1 (`0x01`) |
| `kFurnitureMode` | 2 | `:273` | `kFirstFurniture` 1 | `kLastFurniture` 15 | `kFurnitureBase` 21 |
| `kBonusMode` | 3 | `:274` | `kFirstBonus` 1 | `kLastBonus` 15 | `kBonusBase` 41 |
| `kTransportMode` | 4 | `:275` | `kFirstTransport` 1 | `kLastTransport` 12 | `kTransportBase` 61 |
| `kSwitchMode` | 5 | `:276` | `kFirstSwitch` 1 | `kLastSwitch` 9 | `kSwitchBase` 81 |
| `kLightMode` | 6 | `:277` | `kFirstLight` 1 | `kLastLight` 8 | `kLightBase` 101 |
| `kApplianceMode` | 7 | `:278` | `kFirstAppliance` 1 | `kLastAppliance` 14 | `kApplianceBase` 121 |
| `kEnemyMode` | 8 | `:279` | `kFirstEnemy` 1 | `kLastEnemy` 9 | `kEnemyBase` 141 |
| `kClutterMode` | 9 | `:280` | `kFirstClutter` 1 | `kLastClutter` 15 | `kClutterBase` 161 |

The *bases* are **not** object IDs — they are offsets into the `STR# kObjectNameStrings`
(1007) string list and into `PICT kToolsPictID` (1011). The bases are spaced 20 apart
(1, 21, 41, 61, 81, 101, 121, 141, 161), one slot per class of up to 20 members, whereas
the object IDs are spaced 16 apart (`0x01`, `0x11`, `0x21`, …, `0x81`). A port must keep
these two numbering systems separate.

Other tool constants: `kToolsHigh` 4, `kToolsWide` 4, `kTotalTools` 16
(`GliderPRO/Sources/Tools.c:15-17`), `kPopUpControl` 129 (`:18`),
`kToolsPictID` 1011 (`:46`). `kNumMarqueePats` 7
(`GliderPRO/Headers/GliderDefines.h:460`) and `kMarqueePatListID` 128 /
`kHandleSideLong` 9 (`GliderPRO/Sources/Marquee.c:15-16`) drive the selection marquee.

### 6.3 `srcRects[]` — the object size table, and the `kFlower` bug

`srcRects` is `Rect *srcRects` (`GliderPRO/Headers/GliderVars.h:49`), allocated as

```c
srcRects = (Rect *)NewPtr(sizeof(Rect) * kNumSrcRects);   // GliderPRO/Sources/StructuresInit2.c:271
```

`NewPtr` does **not** zero memory (unlike `NewPtrClear`). `InitSrcRects()`
(`GliderPRO/Sources/StructuresInit2.c:306-475`) then assigns **116 of the 144** slots.

Programmatic audit of which indices are assigned:

```
assigned: 116 of 144
unassigned: 0x00, 0x20, 0x30, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F, 0x50,
            0x59, 0x5A, 0x5B, 0x5C, 0x5D, 0x5E, 0x5F, 0x60,
            0x6F, 0x70, 0x7A, 0x7B, 0x7C, 0x7D, 0x7E, 0x7F, 0x80, 0x85
```

All of the unassigned indices except one are **gaps between classes** and correspond to no
object. The exception is **`0x85` = `kFlower`** — a real, placeable object type whose
`srcRects` entry is never written, so it reads uninitialised heap.

Why this is latent rather than a crash: `ObjectRects.c` routes all 15 clutter types
through the object's own stored `Rect`, not through `srcRects`:

```c
case kOzma: case kMirror: case kMousehole: case kFireplace: case kFlower:
case kWallWindow: case kBear: case kCalendar: case kVase1: case kVase2:
case kBulletin: case kCloud: case kFaucet: case kRug: case kChimes:
    *itsRect = who->data.i.bounds;                 // GliderPRO/Sources/ObjectRects.c:255-271
    break;
```

whereas enemies do use `srcRects`:

```c
case kCopterLf: ... case kCobweb:
    *itsRect = srcRects[who->what];                // GliderPRO/Sources/ObjectRects.c:240-...
    ZeroRectCorner(itsRect);
    QOffsetRect(itsRect, who->data.h.topLeft.h, who->data.h.topLeft.v);
    break;
```

A Go port should simply give `kFlower` its correct size (the flower artwork is 6 sprites,
`flowerSrc[0..5]`, §18.5) and use `NewPtrClear` semantics (zero value) everywhere.

### 6.4 Complete `InitSrcRects()` size table

All 116 assignments from `GliderPRO/Sources/StructuresInit2.c:306-475`. "atlas @(x,y)"
gives the source origin inside the atlas PICT where one is used; entries with no origin
are `QSetRect(&srcRects[k], 0, 0, w, h)` with the origin implied by the drawing code.

| Object | Hex | W × H | Atlas origin | Atlas |
|---|---|---|---|---|
| `kFloorVent` | `0x01` | 48 × 11 | (0, 0) | `kBlowerPictID` 4000 |
| `kCeilingVent` | `0x02` | 48 × 11 | (0, 11) | 4000 |
| `kFloorBlower` | `0x03` | 48 × 15 | (0, 22) | 4000 |
| `kCeilingBlower` | `0x04` | 48 × 15 | (0, 37) | 4000 |
| `kSewerGrate` | `0x05` | 48 × 17 | (0, 52) | 4000 |
| `kLeftFan` | `0x06` | 40 × 55 | (0, 69) | 4000 |
| `kRightFan` | `0x07` | 40 × 55 | (0, 124) | 4000 |
| `kTaper` | `0x08` | 20 × 59 | (0, 209) | 4000 |
| `kCandle` | `0x09` | 32 × 30 | (0, 179) | 4000 |
| `kStubby` | `0x0A` | 20 × 36 | (0, 268) | 4000 |
| `kTiki` | `0x0B` | 27 × 28 | (21, 268) | 4000 |
| `kBBQ` | `0x0C` | 64 × 33 | (0, 0) | `kBBQPictID` 3988 |
| `kInvisBlower` | `0x0D` | 24 × 24 | — | (invisible) |
| `kGrecoVent` | `0x0E` | 48 × 18 | (0, 340) | 4000 |
| `kSewerBlower` | `0x0F` | 32 × 12 | (0, 390) | 4000 |
| `kLiftArea` | `0x10` | 64 × 32 | — | (invisible) |
| `kTable` | `0x11` | 64 × `kTableThick` (8) | — | procedural |
| `kShelf` | `0x12` | 64 × `kShelfThick` (6) | — | procedural |
| `kCabinet` | `0x13` | 64 × 64 | — | procedural |
| `kFilingCabinet` | `0x14` | 74 × 107 | — | `kFilingCabinetPictID` 3995 |
| `kWasteBasket` | `0x15` | 64 × 61 | (0, 43) | `kFurniturePictID` 4001 |
| `kMilkCrate` | `0x16` | 64 × 58 | (0, 104) | 4001 |
| `kCounter` | `0x17` | 128 × 64 | — | procedural |
| `kDresser` | `0x18` | 128 × 64 | — | procedural |
| `kDeckTable` | `0x19` | 64 × `kTableThick` (8) | — | procedural |
| `kStool` | `0x1A` | 48 × 38 | (0, 183) | 4001 |
| `kTrunk` | `0x1B` | 144 × 80 | — | `kTrunkPictID` 3987 |
| `kInvisObstacle` | `0x1C` | 64 × 64 | — | (invisible) |
| `kManhole` | `0x1D` | 123 × 22 | — | `kManholePictID` 3967 |
| `kBooks` | `0x1E` | 64 × 51 | — | `kBooksPictID` 3964 |
| `kInvisBounce` | `0x1F` | 64 × 64 | — | (invisible) |
| `kRedClock` | `0x21` | 28 × 17 | (0, 0) | `kBonusPictID` 4002 |
| `kBlueClock` | `0x22` | 28 × 25 | (0, 17) | 4002 |
| `kYellowClock` | `0x23` | 28 × 28 | (0, 42) | 4002 |
| `kCuckoo` | `0x24` | 40 × 80 | (0, 148) | 4002 |
| `kPaper` | `0x25` | 48 × 21 | (0, 127) | 4002 |
| `kBattery` | `0x26` | 16 × 25 | (32, 0) | 4002 |
| `kBands` | `0x27` | 28 × 23 | (20, 70) | 4002 |
| `kGreaseRt` | `0x28` | 32 × 27 | (0, 243) | 4002 |
| `kGreaseLf` | `0x29` | 32 × 27 | (0, 324) | 4002 |
| `kFoil` | `0x2A` | 55 × 15 | (0, 228) | 4002 |
| `kInvisBonus` | `0x2B` | 24 × 24 | — | (invisible) |
| `kStar` | `0x2C` | 32 × 31 | (48, 0) | 4002 |
| `kSparkle` | `0x2D` | 20 × 19 | (0, 70) | 4002 |
| `kHelium` | `0x2E` | 56 × 16 | (32, 270) | 4002 |
| `kSlider` | `0x2F` | 64 × 16 | — | (editor helper) |
| `kUpStairs` | `0x31` | 160 × 267 | — | `kUpStairsPictID` 3997 |
| `kDownStairs` | `0x32` | 160 × 267 | — | `kDownStairsPictID` 3996 |
| `kMailboxLf` | `0x33` | 94 × 80 | — | `kMailboxLeftPictID` 3986 |
| `kMailboxRt` | `0x34` | 94 × 80 | — | `kMailboxRightPictID` 3985 |
| `kFloorTrans` | `0x35` | 56 × 15 | (0, 1) | `kTransportPictID` 4008 |
| `kCeilingTrans` | `0x36` | 56 × 15 | (0, 16) | 4008 |
| `kDoorInLf` | `0x37` | 144 × 322 | — | `kDoorInLeftPictID` 3984 |
| `kDoorInRt` | `0x38` | 144 × 322 | — | `kDoorInRightPictID` 3983 |
| `kDoorExRt` | `0x39` | 16 × 322 | — | `kDoorExRightPictID` 3982 |
| `kDoorExLf` | `0x3A` | 16 × 322 | — | `kDoorExLeftPictID` 3981 |
| `kWindowInLf` | `0x3B` | 20 × 170 | — | `kWindowInLeftPictID` 3980 |
| `kWindowInRt` | `0x3C` | 20 × 170 | — | `kWindowInRightPictID` 3979 |
| `kWindowExRt` | `0x3D` | 16 × 170 | — | `kWindowExRightPictID` 3977 |
| `kWindowExLf` | `0x3E` | 16 × 170 | — | `kWindowExLeftPictID` 3978 |
| `kInvisTrans` | `0x3F` | 64 × 32 | — | (invisible) |
| `kDeluxeTrans` | `0x40` | 64 × 64 | — | (invisible) |
| `kLightSwitch` | `0x41` | 15 × 24 | (0, 0) | `kSwitchPictID` 4003 |
| `kMachineSwitch` | `0x42` | 16 × 24 | (0, 48) | 4003 |
| `kThermostat` | `0x43` | 15 × 24 | (0, 48) | 4003 |
| `kPowerSwitch` | `0x44` | 8 × 8 | (0, 72) | 4003 |
| `kKnifeSwitch` | `0x45` | 16 × 24 | (0, 80) | 4003 |
| `kInvisSwitch` | `0x46` | 12 × 12 | — | (invisible) |
| `kTrigger` | `0x47` | 12 × 12 | — | (invisible) |
| `kLgTrigger` | `0x48` | 48 × 48 | — | (invisible) |
| `kSoundTrigger` | `0x49` | 32 × 32 | — | (invisible) |
| `kCeilingLight` | `0x51` | 64 × 20 | (0, 0) | `kLightPictID` 4004 |
| `kLightBulb` | `0x52` | 16 × 28 | (0, 20) | 4004 |
| `kTableLamp` | `0x53` | 48 × 70 | (16, 20) | 4004 |
| `kHipLamp` | `0x54` | 72 × 276 | — | `kHipLampPictID` 3994 |
| `kDecoLamp` | `0x55` | 64 × 212 | — | `kDecoLampPictID` 3993 |
| `kFlourescent` | `0x56` | 64 × 12 | — | procedural (tiled) |
| `kTrackLight` | `0x57` | 64 × 24 | — | procedural (tiled, spacing 64) |
| `kInvisLight` | `0x58` | 16 × 16 | — | (invisible) |
| `kShredder` | `0x61` | 73 × 22 | — | `kAppliancePictID` 4005 region |
| `kToaster` | `0x62` | 48 × 27 | (0, 22) | 4005 |
| `kMacPlus` | `0x63` | 48 × 58 | (0, 49) | 4005 |
| `kGuitar` | `0x64` | 64 × 172 | — | `kGuitarPictID` 3991 |
| `kTV` | `0x65` | 92 × 77 | — | `kTVPictID` 3992 |
| `kCoffee` | `0x66` | 43 × 64 | (0, 107) | 4005 |
| `kOutlet` | `0x67` | 16 × 24 | (64, 22) | 4005 |
| `kVCR` | `0x68` | 96 × 22 | — | `kVCRPictID` 3990 |
| `kStereo` | `0x69` | 128 × 53 | — | `kStereoPictID` 3989 |
| `kMicrowave` | `0x6A` | 92 × 59 | — | `kMicrowavePictID` 3971 |
| `kCinderBlock` | `0x6B` | 40 × 62 | — | `kCinderPictID` 3960 |
| `kFlowerBox` | `0x6C` | 80 × 32 | — | `kFlowerBoxPictID` 3959 |
| `kCDs` | `0x6D` | 16 × 30 | (48, 22) | 4005 |
| `kCustomPict` | `0x6E` | 72 × 34 | — | house's own PICT |
| `kBalloon` | `0x71` | 24 × 30 | — | `kBalloonPictID` 4011 |
| `kCopterLf` | `0x72` | 32 × 30 | — | `kCopterPictID` 4012 |
| `kCopterRt` | `0x73` | 32 × 30 | — | 4012 |
| `kDartLf` | `0x74` | 64 × 19 | — | `kDartPictID` 4013 |
| `kDartRt` | `0x75` | 64 × 19 | — | 4013 |
| `kBall` | `0x76` | 32 × 32 | — | `kBallPictID` 4014 |
| `kDrip` | `0x77` | 16 × 12 | — | `kDripPictID` 4015 |
| `kFish` | `0x78` | 36 × 33 | — | `kFishPictID` 4017 |
| `kCobweb` | `0x79` | 54 × 45 | — | `kCobwebPictID` 3958 |
| `kOzma` | `0x81` | 102 × 92 | — | `kOzmaPictID` 3975 |
| `kMirror` | `0x82` | 64 × 64 | — | procedural |
| `kMousehole` | `0x83` | 10 × 11 | — | `kClutterPictID` 4018 region |
| `kFireplace` | `0x84` | 180 × 142 | — | `kFireplacePictID` 3973 |
| **`kFlower`** | `0x85` | **never assigned** | — | `flowerSrc[0..5]` in `kClutterPictID` 4018 |
| `kWallWindow` | `0x86` | 64 × 80 | — | 4018 region |
| `kBear` | `0x87` | 56 × 58 | — | `kBearPictID` 3972 |
| `kCalendar` | `0x88` | 63 × 92 | — | `kCalendarPictID` 3970 |
| `kVase1` | `0x89` | 36 × 45 | — | `kVase1PictID` 3969 |
| `kVase2` | `0x8A` | 35 × 57 | — | `kVase2PictID` 3968 |
| `kBulletin` | `0x8B` | 80 × 58 | — | `kBulletinPictID` 3966 |
| `kCloud` | `0x8C` | 128 × 30 | — | `kCloudPictID` 3965 (masked) |
| `kFaucet` | `0x8D` | 56 × 18 | (0, 51) | 4018 |
| `kRug` | `0x8E` | 144 × 18 | — | `kRugPictID` 3962 |
| `kChimes` | `0x8F` | 28 × 74 | — | `kChimesPictID` 3961 |

Thickness constants used above: `kTableThick` 8
(`GliderPRO/Headers/GliderDefines.h:440`), `kShelfThick` 6 (`:440`),
`kStoolThick` 25 (`GliderPRO/Sources/ObjectRects.c:18`).
---

## 7. The 10-byte object data union

`objectType` is 12 bytes: a 2-byte `what` followed by a 10-byte union
(`GliderPRO/Headers/GliderStructs.h:90-105`). Which arm is live is determined entirely by
`what` (§6.1). This is **on-disk** data — the union means the same 10 bytes are
reinterpreted, so a Go port must decode by class.

Offsets below are relative to the start of the 12-byte `objectType`.

### 7.1 Arm `a` — `blowerType` (`GliderPRO/Headers/GliderStructs.h:11-19`)

Used by object IDs `0x01`-`0x10`.

| Off | Field | C type | Bytes | Meaning |
|---|---|---|---|---|
| 2 | `topLeft.v` | `short` | 2 | y of top-left |
| 4 | `topLeft.h` | `short` | 2 | x of top-left |
| 6 | `distance` | `short` | 2 | how far the airflow reaches (px) |
| 8 | `initial` | `Boolean` | 1 | on/off at room entry |
| 9 | `state` | `Boolean` | 1 | current on/off (runtime; saved in saved games) |
| 10 | `vector` | `Byte` | 1 | direction bitmask — see below |
| 11 | `tall` | `Byte` | 1 | height for variable-size blowers (`kLiftArea`) |

The author documents `vector` inline at `GliderPRO/Headers/GliderStructs.h:16-17`:

```
                                   F.  lf. dn. rt. up
    | x | x | x | x |  8 |  4 |  2 |  1 |
```

So bit 0 (`0x01`) = up, bit 1 (`0x02`) = right, bit 2 (`0x04`) = down, bit 3 (`0x08`) =
left, and the top nibble is unused. ("F." is the author's abbreviation for the unused
high nibble.)

### 7.2 Arm `b` — `furnitureType` (`GliderPRO/Headers/GliderStructs.h:21-25`)

Used by `0x11`-`0x1F`.

| Off | Field | C type | Bytes | Meaning |
|---|---|---|---|---|
| 2 | `bounds.top` | `short` | 2 | full rect, absolute room coords |
| 4 | `bounds.left` | `short` | 2 | |
| 6 | `bounds.bottom` | `short` | 2 | |
| 8 | `bounds.right` | `short` | 2 | |
| 10 | `pict` | `short` | 2 | custom PICT ID override (0 = use default art) |

### 7.3 Arm `c` — `bonusType` (`GliderPRO/Headers/GliderStructs.h:27-34`)

Used by `0x21`-`0x2F`.

| Off | Field | C type | Bytes | Meaning |
|---|---|---|---|---|
| 2 | `topLeft.v` | `short` | 2 | y |
| 4 | `topLeft.h` | `short` | 2 | x |
| 6 | `length` | `short` | 2 | author's comment "grease spill" — spill length for `kGreaseRt`/`kGreaseLf` |
| 8 | `points` | `short` | 2 | author's comment "invis bonus" — score value for `kInvisBonus` |
| 10 | `state` | `Boolean` | 1 | already-collected flag |
| 11 | `initial` | `Boolean` | 1 | present at room entry |

Note the **field order is `state` then `initial`** here, the reverse of `blowerType`.

### 7.4 Arm `d` — `transportType` (`GliderPRO/Headers/GliderStructs.h:36-43`)

Used by `0x31`-`0x40`.

| Off | Field | C type | Bytes | Meaning |
|---|---|---|---|---|
| 2 | `topLeft.v` | `short` | 2 | y |
| 4 | `topLeft.h` | `short` | 2 | x |
| 6 | `tall` | `short` | 2 | author's comment "invis transport" — height for `kInvisTrans`/`kDeluxeTrans` |
| 8 | `where` | `short` | 2 | destination room number (or `kRoomIsEmpty` −1) |
| 10 | `who` | `Byte` | 1 | destination object index within that room |
| 11 | `wide` | `Byte` | 1 | width for `kDeluxeTrans` |

`where`/`who` is the universal "link" pair. Link types (`GliderPRO/Headers/GliderDefines.h`,
see §22.6): `kLinkedToOther` 0, `kLinkedToLeftMailbox` 1, `kLinkedToRightMailbox` 2,
`kLinkedToCeilingDuct` 3, `kLinkedToFloorDuct` 4.

### 7.5 Arm `e` — `switchType` (`GliderPRO/Headers/GliderStructs.h:45-52`)

Used by `0x41`-`0x49`.

| Off | Field | C type | Bytes | Meaning |
|---|---|---|---|---|
| 2 | `topLeft.v` | `short` | 2 | y |
| 4 | `topLeft.h` | `short` | 2 | x |
| 6 | `delay` | `short` | 2 | frames before the switch acts |
| 8 | `where` | `short` | 2 | linked room number |
| 10 | `who` | `Byte` | 1 | linked object index |
| 11 | `type` | `Byte` | 1 | `kToggle` 0 / `kForceOn` 1 / `kForceOff` 2 / `kOneShot` 3 (`GliderPRO/Headers/GliderDefines.h:441-444`) |

### 7.6 Arm `f` — `lightType` (`GliderPRO/Headers/GliderStructs.h:54-62`)

Used by `0x51`-`0x58`.

| Off | Field | C type | Bytes | Meaning |
|---|---|---|---|---|
| 2 | `topLeft.v` | `short` | 2 | y |
| 4 | `topLeft.h` | `short` | 2 | x |
| 6 | `length` | `short` | 2 | width for `kFlourescent`/`kTrackLight` |
| 8 | `byte0` | `Byte` | 1 | unnamed spare |
| 9 | `byte1` | `Byte` | 1 | unnamed spare |
| 10 | `initial` | `Boolean` | 1 | lit at room entry |
| 11 | `state` | `Boolean` | 1 | currently lit |

### 7.7 Arm `g` — `applianceType` (`GliderPRO/Headers/GliderStructs.h:64-72`)

Used by `0x61`-`0x6E`.

| Off | Field | C type | Bytes | Meaning |
|---|---|---|---|---|
| 2 | `topLeft.v` | `short` | 2 | y |
| 4 | `topLeft.h` | `short` | 2 | x |
| 6 | `height` | `short` | 2 | author's comment "toaster, pict ID" — **overloaded**: toast launch height for `kToaster`, fish leap height for `kFish` (which reads `data.g.height` despite being an enemy), and the PICT ID for `kCustomPict` |
| 8 | `byte0` | `Byte` | 1 | for `kMicrowave`: kill mask (bit 0 = bands, bit 1 = battery, bit 2 = foil) |
| 9 | `delay` | `Byte` | 1 | delay in 6-tick units (→ frames via `× 6 / kTicksPerFrame`) |
| 10 | `initial` | `Boolean` | 1 | on at room entry |
| 11 | `state` | `Boolean` | 1 | currently on |

**The `kFish` cross-arm read is real**: `GliderPRO/Sources/Dynamics3.c` seeds the fish's
leap using `who->data.g.height` while the fish's own topLeft comes from `data.h.topLeft`.
Because both arms put `topLeft` at the same offset and `data.g.height` and
`data.h.length` are both at offset 6, this is harmless aliasing, but a Go port that
decodes by class will get the wrong field name unless it replicates the aliasing.

### 7.8 Arm `h` — `enemyType` (`GliderPRO/Headers/GliderStructs.h:74-82`)

Used by `0x71`-`0x79`.

| Off | Field | C type | Bytes | Meaning |
|---|---|---|---|---|
| 2 | `topLeft.v` | `short` | 2 | y |
| 4 | `topLeft.h` | `short` | 2 | x |
| 6 | `length` | `short` | 2 | bounce height (`kBall`), fall distance (`kDrip`) |
| 8 | `delay` | `Byte` | 1 | respawn delay in 6-tick units |
| 9 | `byte0` | `Byte` | 1 | unnamed spare |
| 10 | `initial` | `Boolean` | 1 | active at room entry |
| 11 | `state` | `Boolean` | 1 | currently active |

Note `delay` and `byte0` are in the **opposite** order to `applianceType`, which has
`byte0` then `delay`. So `data.g.byte0` aliases `data.h.delay`, and `data.g.delay` aliases
`data.h.byte0`.

### 7.9 Arm `i` — `clutterType` (`GliderPRO/Headers/GliderStructs.h:84-88`)

Used by `0x81`-`0x8F`.

| Off | Field | C type | Bytes | Meaning |
|---|---|---|---|---|
| 2 | `bounds.top` | `short` | 2 | full rect, absolute room coords |
| 4 | `bounds.left` | `short` | 2 | |
| 6 | `bounds.bottom` | `short` | 2 | |
| 8 | `bounds.right` | `short` | 2 | |
| 10 | `pict` | `short` | 2 | custom PICT ID override |

Byte-identical to `furnitureType`.

### 7.10 Union aliasing summary

Every arm is exactly 10 bytes. The aliasing table at each offset:

| Off | `a` blower | `b` furniture | `c` bonus | `d` transport | `e` switch | `f` light | `g` appliance | `h` enemy | `i` clutter |
|---|---|---|---|---|---|---|---|---|---|
| 2-3 | `topLeft.v` | `bounds.top` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `topLeft.v` | `bounds.top` |
| 4-5 | `topLeft.h` | `bounds.left` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `topLeft.h` | `bounds.left` |
| 6-7 | `distance` | `bounds.bottom` | `length` | `tall` | `delay` | `length` | `height` | `length` | `bounds.bottom` |
| 8-9 | `initial`,`state` | `bounds.right` | `points` | `where` | `where` | `byte0`,`byte1` | `byte0`,`delay` | `delay`,`byte0` | `bounds.right` |
| 10 | `vector` | `pict` (hi) | `state` | `who` | `who` | `initial` | `initial` | `initial` | `pict` (hi) |
| 11 | `tall` | `pict` (lo) | `initial` | `wide` | `type` | `state` | `state` | `state` | `pict` (lo) |

Note that for `b`/`i` the "topLeft" of an object is `bounds.top`/`bounds.left` — i.e.
**(v, h)** in `Point` order coincides with **(top, left)** in `Rect` order. This is why the
editor can treat all objects uniformly when dragging.

---

## 8. Hot-spot actions

`hotObject` (§23.2) is the runtime collision record. Each frame, `CheckForHotSpots()`
(`GliderPRO/Sources/Interactions.c:1625`) intersects the glider against `hotSpots[0..nHotSpots-1]`
and `HandleHotSpotCollision()` (`GliderPRO/Sources/Interactions.c:1196`) dispatches on
`hotSpots[i].action`. All 28 values, from `GliderPRO/Headers/GliderDefines.h:282-309`:

| Name | Value | Site | Effect (from `HandleHotSpotCollision`) |
|---|---|---|---|
| `kIgnoreIt` | 0 | `:282` | no-op (the "off" state of a switchable hot spot) |
| `kLiftIt` | 1 | `:283` | `thisGlider->vDesiredVel = kFloorVentLift;` (= −6) |
| `kDropIt` | 2 | `:284` | `thisGlider->vDesiredVel = kCeilingVentDrop;` (= 8) |
| `kPushItLeft` | 3 | `:285` | `thisGlider->hDesiredVel += -kFanStrength;` (= −12) |
| `kPushItRight` | 4 | `:286` | `thisGlider->hDesiredVel += kFanStrength;` (= +12) |
| `kDissolveIt` | 5 | `:287` | object vanishes (bonus pickup path) |
| `kRewardIt` | 6 | `:288` | calls `HandleRewards()` — §14 |
| `kMoveItUp` | 7 | `:289` | staircase/transport upward |
| `kMoveItDown` | 8 | `:290` | staircase/transport downward |
| `kSwitchIt` | 9 | `:291` | throws the linked switch |
| `kShredIt` | 10 | `:292` | glider enters `kGliderShredding`; `kShredSound` |
| `kStrumIt` | 11 | `:293` | guitar strum sound |
| `kTriggerIt` | 12 | `:294` | fires the linked trigger; `PlayPrioritySound(kTriggerSound, kTriggerPriority)` (`GliderPRO/Sources/Interactions.c:1072`, `:1618`) |
| `kBurnIt` | 13 | `:295` | glider catches fire — **unless** it has foil, in which case `vDesiredVel = kFloorVentLift` and `kSizzleSound` plays |
| `kSlideIt` | 14 | `:296` | `sliding = true; vVel = who->bounds.top - dest.bottom;` (snap onto the surface) |
| `kTransportIt` | 15 | `:297` | teleport via the linked transport |
| `kIgnoreLeftWall` | 16 | `:298` | suppress left-wall collision this frame |
| `kIgnoreRightWall` | 17 | `:299` | suppress right-wall collision |
| `kMailItLeft` | 18 | `:300` | enter left mailbox (`kGliderMailInLeft`) |
| `kMailItRight` | 19 | `:301` | enter right mailbox (`kGliderMailInRight`) |
| `kDuctItDown` | 20 | `:302` | enter floor duct (`kGliderDuctingDown`) |
| `kDuctItUp` | 21 | `:303` | enter ceiling duct (`kGliderDuctingUp`) |
| `kMicrowaveIt` | 22 | `:304` | `HandleMicrowaveAction()` — destroys inventory per bitmask, §15.4 |
| `kIgnoreGround` | 23 | `:305` | suppress floor collision (open manhole etc.) |
| `kBounceIt` | 24 | `:306` | `BounceGlider()`, §10.5 |
| `kChimeIt` | 25 | `:307` | wind-chime sound |
| `kWebIt` | 26 | `:308` | `WebGlider()`, §13.3 |
| `kSoundIt` | 27 | `:309` | plays the house's custom trigger `snd `, §16.4 |

`AddActiveRect(Rect *theRect, short action, short who, Boolean isOn, Boolean doScrutinize)`
(`GliderPRO/Sources/ObjectRects.c:276`) registers one:

```
1. if (nHotSpots >= kMaxHotSpots) return (-1);          // 56 max
2. hotSpots[nHotSpots].bounds       = *theRect;
3. hotSpots[nHotSpots].action       = action;
4. hotSpots[nHotSpots].who          = who;
5. hotSpots[nHotSpots].isOn         = isOn;
6. hotSpots[nHotSpots].stillOver    = false;
7. hotSpots[nHotSpots].doScrutinize = doScrutinize;
8. nHotSpots++;
9. return (nHotSpots - 1);
```

`stillOver` implements "one trigger per entry": `FlagStillOvers()`
(`GliderPRO/Sources/Interactions.c:1713`) clears the flag for every hot spot the glider is
no longer inside, so a `kRewardIt`/`kSwitchIt` fires once on entry rather than every frame.
`doScrutinize` asks for a pixel-accurate rather than rect-accurate test.

Hot-spot geometry constants (`GliderPRO/Sources/ObjectRects.c:13-19`):

| Name | Value | Meaning |
|---|---|---|
| `kFloorColumnWide` | 4 | width of the invisible lift column above a floor vent |
| `kCeilingColumnWide` | 24 | width of the drop column below a ceiling vent |
| `kFanColumnThick` | 16 | thickness of a fan's push zone |
| `kFanColumnDown` | 20 | how far below the fan the push zone extends |
| `kDeadlyFlameHeight` | 24 | height of the burn zone above a candle/tiki/BBQ |
| `kStoolThick` | 25 | stool collision thickness |
| `kShredderActiveHigh` | 40 | height of the shredder's intake zone |

---

## 9. The glider

### 9.1 Dimensions and sprite atlas

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kGliderWide` | 48 | `GliderPRO/Headers/GliderDefines.h:548` | glider sprite width |
| `kGliderHigh` | 20 | `GliderPRO/Headers/GliderDefines.h:549` | normal-frame height |
| `kHalfGliderWide` | 24 | `GliderPRO/Headers/GliderDefines.h:550` | = 48/2 |
| `kGliderBurningHigh` | 26 | `GliderPRO/Headers/GliderDefines.h:551` | burning-frame height (taller — flames) |
| `kShadowHigh` | 9 | `GliderPRO/Headers/GliderDefines.h:552` | shadow sprite height |
| `kShadowTop` | 306 | `GliderPRO/Headers/GliderDefines.h:553` | fixed y of the shadow (never moves vertically) |
| `kNumGliderSrcRects` | 31 | `GliderPRO/Headers/GliderDefines.h:558` | frames in the glider atlas (0…30) |
| `kNumShadowSrcRects` | 2 | `GliderPRO/Headers/GliderDefines.h:559` | shadow frames |
| `kGliderStartsDown` | 32 | `GliderPRO/Headers/GliderDefines.h:569` | y offset for a glider entering from above |
| `kFirstAboutFaceFrame` | 18 | `GliderPRO/Headers/GliderDefines.h:560` | first frame of the turn-around animation |
| `kLastAboutFaceFrame` | 20 | `GliderPRO/Headers/GliderDefines.h:561` | last frame of the turn-around animation |
| `kWasBurning` | 2 | `GliderPRO/Headers/GliderDefines.h:562` | sentinel stored in `wasMode` |
| `kLeftFadeOffset` | 7 | `GliderPRO/Headers/GliderDefines.h:563` | frame offset added when facing left during fade-in |
| `kLastFadeSequence` | 16 | `GliderPRO/Headers/GliderDefines.h:564` | length of `fadeInSequence[]` |
| `kFaceRight` | TRUE | `GliderPRO/Headers/GliderDefines.h:554` | `facing` value |
| `kFaceLeft` | FALSE | `GliderPRO/Headers/GliderDefines.h:555` | `facing` value |
| `kPlayer1` | TRUE | `GliderPRO/Headers/GliderDefines.h:556` | `which` value |
| `kPlayer2` | FALSE | `GliderPRO/Headers/GliderDefines.h:557` | `which` value |

`InitGliderMap()` (`GliderPRO/Sources/StructuresInit.c:169-238`) builds the frame table.
Numbered:

1. `glidSrcRect = {0, 0, kGliderWide, 668}` — the comment says "32112 pixels" (48 × 669;
   the author's arithmetic uses 669). Loaded with `PICT kGliderPictID` (3999).
2. `glid2SrcMap` = same rect, `PICT kGlider2PictID` (3974) — the *second player's*
   differently-coloured glider.
3. `glidMaskMap` = same rect at **depth 1**, `PICT kGliderPictID + 1000` = 4999.
4. Frames 0…20 (normal): `gliderSrc[i] = {0, 0, kGliderWide, kGliderHigh}` offset by
   `(0, kGliderHigh * i)` — so y = 0, 20, 40, …, 400.
5. Frames 21…28 (burning): `gliderSrc[i] = {0, 0, kGliderWide, kGliderBurningHigh}` offset
   by `(0, 420 + kGliderBurningHigh * (i - 21))` — y = 420, 446, 472, 498, 524, 550, 576, 602.
6. `gliderSrc[29]` = 48 × 20 at y = 628.
7. `gliderSrc[30]` = 48 × 20 at y = 648.

Total atlas height used = 648 + 20 = 668, matching step 1.

Shadow (`GliderPRO/Sources/StructuresInit.c:~220`):
`shadowSrcRect = {0, 0, kGliderWide, kShadowHigh * kNumShadowSrcRects}` = 48 × 18;
`shadowMaskMap` at depth 1 loads `PICT kShadowPictID + 1000` = 4998;
`shadowSrc[i] = {0, 0, kGliderWide, kShadowHigh}` offset by `(0, kShadowHigh * i)`.

Foil variants: `kGliderFoilPictID` 3976 (`GliderPRO/Headers/GliderDefines.h:567`) and
`kGliderFoil2PictID` 3963 (`GliderPRO/Headers/GliderDefines.h:565`) — the player-1 and
player-2 aluminium-foil-clad gliders.

### 9.2 `fadeInSequence[16]` — the materialise animation

`GliderPRO/Sources/InterfaceInit.c:167-182`, written 4 values per line with the author's
group comments `// 4`, `// 5`, `// 6`, `// 7`:

```
fadeInSequence[16] = { 4, 5, 6, 7,      // 4
                      5, 6, 7, 8,      // 5
                      6, 7, 8, 9,      // 6
                      7, 8, 9, 10 };   // 7
```

These are atlas frame indices. For a left-facing glider `kLeftFadeOffset` (7) is added, so
the left-facing sequence is 11,12,13,14, 12,13,14,15, 13,14,15,16, 14,15,16,17. The
sequence is played back over `kLastFadeSequence` = 16 frames when the glider fades in
(`kGliderFadingIn`) and in reverse when it fades out (`kGliderFadingOut`).

### 9.3 Glider modes — all 24

`gliderType.mode` (`GliderPRO/Headers/GliderStructs.h:213`). Values from
`GliderPRO/Headers/GliderDefines.h:571-594`:

| Name | Value | Site | Meaning / which `Player.c` mover runs |
|---|---|---|---|
| `kGliderNormal` | 0 | `:571` | free flight; `MoveGlider()` |
| `kGliderFadingIn` | 1 | `:572` | materialising; `fadeInSequence` forward |
| `kGliderFadingOut` | 2 | `:573` | dematerialising; `fadeInSequence` reverse |
| `kGliderGoingUp` | 3 | `:574` | walking up stairs, leaving; `kClimbStairsSpeed` −4 |
| `kGliderComingUp` | 4 | `:575` | arriving at the top of stairs; `kVClimbStairsSpeed`/`kHClimbStairsSpeed` −4 |
| `kGliderGoingDown` | 5 | `:576` | walking down stairs, leaving; `kVDropStairsSpeed`/`kHDropStairsSpeed` +4 |
| `kGliderComingDown` | 6 | `:577` | arriving at the bottom of stairs |
| `kGliderFaceLeft` | 7 | `:578` | about-face to the left; frames 18-20 |
| `kGliderFaceRight` | 8 | `:579` | about-face to the right |
| `kGliderBurning` | 9 | `:580` | on fire; `kFramesToBurn` 60 then dies |
| `kGliderTransporting` | 10 | `:581` | dissolving into a transport |
| `kGliderDuctingDown` | 11 | `:582` | inside a floor duct; `kVDropDuctSpeed` +4 |
| `kGliderDuctingUp` | 12 | `:583` | inside a ceiling duct; `kVRiseDuctSpeed` −4 |
| `kGliderDuctingIn` | 13 | `:584` | entering a duct |
| `kGliderMailInLeft` | 14 | `:585` | being pulled into the left mailbox; `kHMailPullSpeed` +4, `kVMailDropSpeed` +2 |
| `kGliderMailOutLeft` | 15 | `:586` | being pushed out of the left mailbox; `kHPushMailSpeed` −4 |
| `kGliderMailInRight` | 16 | `:587` | into the right mailbox; `kHMailPullRtSpeed` −4, `kVMailDropSpeed` +2 |
| `kGliderMailOutRight` | 17 | `:588` | out of the right mailbox; `kHPushMailRtSpeed` +4 |
| `kGliderGoingFoil` | 18 | `:589` | donning foil |
| `kGliderLosingFoil` | 19 | `:590` | losing foil |
| `kGliderShredding` | 20 | `:591` | in the shredder; `kDropShredSlow` 1 / `kDropShredFast` 4, `kShredderCountdown` −68 |
| `kGliderInLimbo` | 21 | `:592` | between rooms / dead but not respawned |
| `kGliderIdle` | 22 | `:593` | title-screen idle |
| `kGliderTransportingIn` | 23 | `:594` | arriving from a transport |

### 9.4 Escape codes

`CheckGliderInRoom()` returns one of these to say how the glider left the room
(`GliderPRO/Headers/GliderDefines.h:596-608`). All negative so they cannot collide with a
room number.

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kPlayerIsDeadForever` | **−69** | `:596` | out of lives |
| `kPlayerMailedOut` | −12 | `:597` | left via a mailbox |
| `kPlayerDuckedOut` | −11 | `:598` | left via a duct |
| `kPlayerTransportedOut` | −10 | `:599` | left via a transport |
| `kPlayerEscapingDownStairs` | −9 | `:600` | mid-animation, going down |
| `kPlayerEscapingUpStairs` | −8 | `:601` | mid-animation, going up |
| `kPlayerEscapedDownStairs` | −7 | `:602` | completed, down |
| `kPlayerEscapedUpStairs` | −6 | `:603` | completed, up |
| `kPlayerEscapedDown` | −5 | `:604` | fell through the floor |
| `kPlayerEscapedUp` | −4 | `:605` | rose through the ceiling |
| `kPlayerEscapedLeft` | −3 | `:606` | flew out the left wall |
| `kPlayerEscapedRight` | −2 | `:607` | flew out the right wall |
| `kNoOneEscaped` | −1 | `:608` | still in the room |

`kShredderCountdown` = **−68** (`GliderPRO/Sources/Player.c:17`) sits immediately adjacent
to `kPlayerIsDeadForever` = −69 in value space. They are used in different variables
(`mode`-adjacent counters vs escape codes) but the proximity is worth noting: a port that
merges these into one type will produce an off-by-one death bug.

### 9.5 Room-entry sentinels

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kRoomIsEmpty` | −1 | `GliderPRO/Headers/GliderDefines.h:525` | no such room |
| `kObjectIsEmpty` | −1 | `GliderPRO/Headers/GliderDefines.h:526` | empty object slot (`what` == −1) |
| `kNoObjectSelected` | −1 | `GliderPRO/Headers/GliderDefines.h:527` | editor: nothing selected |
| `kInitialGliderSelected` | −2 | `GliderPRO/Headers/GliderDefines.h:528` | editor: the house-start marker is selected |
| `kLeftGliderSelected` | −3 | `GliderPRO/Headers/GliderDefines.h:529` | editor: the room's left-entry marker |
| `kRightGliderSelected` | −4 | `GliderPRO/Headers/GliderDefines.h:530` | editor: the room's right-entry marker |

### 9.6 `leftStart` / `rightStart` — room entry height

`roomType.leftStart` and `rightStart` (both `Byte`, §22.3) are **vertical** offsets, not
horizontal ones. Verified at `GliderPRO/Sources/Transit.c:173-176` (entering from the west,
so the glider appears at the room's *left* edge):

```c
QSetRect(&enterRect, 0, 0, 48, 20);
QOffsetRect(&enterRect, 0,
        kGliderStartsDown + (short)thisRoom->leftStart - 2);
```

and `GliderPRO/Sources/Transit.c:207-208` (entering from the east, right edge):

```c
QSetRect(&enterRect, 0, 0, 48, 20);
QOffsetRect(&enterRect, kRoomWide - 48,
        kGliderStartsDown + (short)thisRoom->rightStart - 2);
```

So the entry rect is 48 × 20 (`kGliderWide` × `kGliderHigh`) at
`x = 0` (left entry) or `x = kRoomWide - 48 = 464` (right entry), and
`y = kGliderStartsDown + leftStart - 2 = 30 + leftStart`.

* Default for a new room: `thisRoom->leftStart = 32; thisRoom->rightStart = 32;`
  (`GliderPRO/Sources/Room.c:170-171`) → y = 62.
* Editor drag clamps to **0…255** (`GliderPRO/Sources/ObjectEdit.c:540-543`, `:552-555`).
* Observed in a real house: Sampler room 0 has `leftStart = 32, rightStart = 27`
  (→ y = 62 and 57).
* The editor's *marker* rect is 48 × 16 at `y = kGliderStartsDown + leftStart` — i.e.
  **2 px lower and 4 px shorter** than the actual entry rect
  (`GliderPRO/Sources/ObjectEdit.c:545-547`). Cosmetic only.
* The marker sprites live in the glider atlas at
  `leftStartGliderSrc = {0, 358, 16, 48}` and `rightStartGliderSrc = {0, 374, 16, 48}`
  — 48 × 16 at y = 358 and y = 374 (`GliderPRO/Sources/StructuresInit.c:280-283`).
---

## 10. Glider physics

### 10.1 The `#define`d physics constants

These five, all in `GliderPRO/Sources/Player.c`, are the entire tuning surface of the
glider's free flight:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kGravity` | **3** | `GliderPRO/Sources/Player.c:13` | the *target* downward velocity every frame (px/frame) — **not** an acceleration |
| `kHImpulse` | **2** | `GliderPRO/Sources/Player.c:14` | horizontal acceleration toward `hDesiredVel` (px/frame²) |
| `kVImpulse` | **2** | `GliderPRO/Sources/Player.c:15` | vertical acceleration toward `vDesiredVel` (px/frame²) |
| `kMaxHVel` | **16** | `GliderPRO/Sources/Player.c:16` | horizontal speed cap (px/frame), applied both signs |
| `kShredderCountdown` | **−68** | `GliderPRO/Sources/Player.c:17` | sentinel loaded into the shred counter |

Thrust and lift, from `GliderPRO/Sources/Input.c:14-16`:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kNormalThrust` | **5** | `GliderPRO/Sources/Input.c:14` | `hDesiredVel += 5` per frame the arrow key is held |
| `kHyperThrust` | **8** | `GliderPRO/Sources/Input.c:15` | battery thrust: applied directly to `hVel`, not `hDesiredVel` |
| `kHeliumLift` | **4** | `GliderPRO/Sources/Input.c:16` | `vDesiredVel = -4` while helium is engaged |

Environmental forces, from `GliderPRO/Sources/Interactions.c:13-15`:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kFloorVentLift` | **−6** | `GliderPRO/Sources/Interactions.c:13` | `vDesiredVel = -6` over a floor vent |
| `kCeilingVentDrop` | **8** | `GliderPRO/Sources/Interactions.c:14` | `vDesiredVel = 8` under a ceiling vent |
| `kFanStrength` | **12** | `GliderPRO/Sources/Interactions.c:15` | `hDesiredVel += ±12` in front of a fan |

Other movement constants:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kShoveVelocity` | **8** | `GliderPRO/Sources/Dynamics.c:17` | horizontal shove when a foil-clad glider hits a dynamic object |
| `kRubberBandVelocity` | **20** | `GliderPRO/Sources/RubberBands.c:12` | band muzzle speed |
| `kEnemyDropSpeed` | **8** | `GliderPRO/Sources/Dynamics2.c:19` | downward speed of a popped balloon / shot copter |
| `kDartVelocity` | **6** | `GliderPRO/Sources/Dynamics2.c:17`, `Dynamics3.c:15` | dart horizontal speed |

### 10.2 `MoveGlider(gliderPtr thisGlider)` — the core integrator

`GliderPRO/Sources/Player.c:64-147`. This is the single most important function to port
exactly. Numbered pseudocode, original variable names preserved:

```
 1. // --- horizontal acceleration toward the desired velocity ---
 2. if (thisGlider->hVel > thisGlider->hDesiredVel)
 3.     thisGlider->hVel -= kHImpulse;                       // -2
 4.     if (thisGlider->hVel < thisGlider->hDesiredVel)      // overshoot clamp
 5.         thisGlider->hVel = thisGlider->hDesiredVel;
 6. else if (thisGlider->hVel < thisGlider->hDesiredVel)
 7.     thisGlider->hVel += kHImpulse;                       // +2
 8.     if (thisGlider->hVel > thisGlider->hDesiredVel)
 9.         thisGlider->hVel = thisGlider->hDesiredVel;
10.
11. thisGlider->hDesiredVel = 0;                             // Player.c:78 — RESET EVERY FRAME
12.
13. // --- vertical acceleration toward the desired velocity ---
14. if (thisGlider->vVel > thisGlider->vDesiredVel)
15.     thisGlider->vVel -= kVImpulse;                       // -2
16.     if (thisGlider->vVel < thisGlider->vDesiredVel)
17.         thisGlider->vVel = thisGlider->vDesiredVel;
18. else if (thisGlider->vVel < thisGlider->vDesiredVel)
19.     thisGlider->vVel += kVImpulse;                       // +2
20.     if (thisGlider->vVel > thisGlider->vDesiredVel)
21.         thisGlider->vVel = thisGlider->vDesiredVel;
22.
23. thisGlider->vDesiredVel = kGravity;                      // Player.c:92 — RESET TO 3 EVERY FRAME
24.
25. // --- horizontal integration, with the ONLY velocity clamp in the function ---
26. if (thisGlider->hVel < 0)
27.     if (thisGlider->hVel < -kMaxHVel)  thisGlider->hVel = -kMaxHVel;   // -16
28.     thisGlider->wasHVel      = thisGlider->hVel;
29.     thisGlider->whole.right  = thisGlider->dest.right;      // "whole" = union of old+new
30.     thisGlider->dest.left   += thisGlider->hVel;
31.     thisGlider->dest.right  += thisGlider->hVel;
32.     thisGlider->whole.left   = thisGlider->dest.left;
33.     thisGlider->wholeShadow.right = thisGlider->destShadow.right;
34.     thisGlider->destShadow.left  += thisGlider->hVel;
35.     thisGlider->destShadow.right += thisGlider->hVel;
36.     thisGlider->wholeShadow.left  = thisGlider->destShadow.left;
37. else
38.     if (thisGlider->hVel > kMaxHVel)   thisGlider->hVel = kMaxHVel;    // +16
39.     thisGlider->wasHVel      = thisGlider->hVel;
40.     thisGlider->whole.left   = thisGlider->dest.left;
41.     thisGlider->dest.left   += thisGlider->hVel;
42.     thisGlider->dest.right  += thisGlider->hVel;
43.     thisGlider->whole.right  = thisGlider->dest.right;
44.     thisGlider->wholeShadow.left  = thisGlider->destShadow.left;
45.     thisGlider->destShadow.left  += thisGlider->hVel;
46.     thisGlider->destShadow.right += thisGlider->hVel;
47.     thisGlider->wholeShadow.right = thisGlider->destShadow.right;
48.
49. // --- vertical integration: NO CLAMP AT ALL ---
50. if (thisGlider->vVel < 0)
51.     thisGlider->wasVVel      = thisGlider->vVel;
52.     thisGlider->whole.bottom = thisGlider->dest.bottom;
53.     thisGlider->dest.top    += thisGlider->vVel;
54.     thisGlider->dest.bottom += thisGlider->vVel;
55.     thisGlider->whole.top    = thisGlider->dest.top;
56. else
57.     thisGlider->wasVVel      = thisGlider->vVel;
58.     thisGlider->whole.top    = thisGlider->dest.top;
59.     thisGlider->dest.top    += thisGlider->vVel;
60.     thisGlider->dest.bottom += thisGlider->vVel;
61.     thisGlider->whole.bottom = thisGlider->dest.bottom;
62.     // NOTE: destShadow / wholeShadow are NOT touched here.
```

Consequences a Go port must reproduce:

1. **`kGravity` is a target velocity, not an acceleration.** Line 23 sets
   `vDesiredVel = 3` unconditionally each frame; lines 14-21 then move `vVel` toward 3 at
   ±`kVImpulse` = 2 per frame. A glider in free fall therefore settles at exactly
   **+3 px/frame** = 90.2 px/s, never faster, unless something else writes `vVel` directly.
   This is *not* Newtonian gravity and cannot be replaced by one.
2. **`hDesiredVel` is reset to 0 every frame** (line 11). Every input and every fan/vent
   must re-apply its contribution *every* frame. `hDesiredVel += kNormalThrust` accumulates
   *within* a frame (two fans stack) but never across frames.
3. **There is no vertical velocity clamp.** `kMaxHVel` guards horizontal only. `vVel` is
   whatever the last writer set — e.g. `kCeilingVentDrop` 8, `kEnemyDropSpeed` 8, or a
   direct `vVel = who->bounds.top - dest.bottom` from `kSlideIt` which can be arbitrarily
   large. Adding a symmetric vertical clamp will change gameplay.
4. **`whole` is the dirty rectangle** (union of the pre- and post-move sprite rects) for the
   `CopyBits` blitter. It is assembled asymmetrically per direction of travel so that no
   `UnionRect` call is needed. In Go with a full-screen redraw this can be dropped, but if
   incremental redraw is kept the exact construction matters.
5. **The shadow moves horizontally but never vertically.** Lines 33-36 and 44-47 move
   `destShadow`; lines 50-61 do not. The shadow lives permanently at
   `kShadowTop` = 306 (`GliderPRO/Headers/GliderDefines.h:553`) and only tracks x. This is
   the intended look — the shadow is on the floor.
6. Because `kHImpulse` = `kVImpulse` = 2 and all the desired velocities are even, `hVel`
   and `vVel` stay **even** in normal flight. Odd values only arise from the direct writes
   (`kGravity` 3, `kSlideIt`, `hVel /= 2`, `hVel /= 4`). Do not introduce floats.

### 10.3 Terminal velocities and the resulting feel

Derived from §10.1-10.2 (all in px/frame; multiply by 30.075 for px/s):

| Situation | `vDesiredVel` | Settles at | Frames to settle from rest |
|---|---|---|---|
| Free flight / fall | `kGravity` 3 | +3 | 2 (0 → 2 → 3) |
| Over a floor vent | `kFloorVentLift` −6 | −6 | 5 (3 → 1 → −1 → −3 → −5 → −6) |
| Under a ceiling vent | `kCeilingVentDrop` 8 | +8 | 3 (3 → 5 → 7 → 8) |
| Helium engaged | `−kHeliumLift` −4 | −4 | 4 |
| Burning + foil (`kBurnIt`) | `kFloorVentLift` −6 | −6 | 5 |

| Situation | `hDesiredVel` per frame | Settles at |
|---|---|---|
| One arrow key held | ±`kNormalThrust` 5 | ±5 (**3** frames: 0 → 2 → 4 → 5) |
| Key + one fan | ±(5 + 12) = ±17 | clamped to ±`kMaxHVel` 16 |
| Two fans, no key | ±24 | clamped to ±16 |
| No input | 0 | 0 (decays at 2/frame) |
| Battery engaged | (direct `hVel ± 8`) | ±16 after clamping |

So the maximum horizontal speed is **16 px/frame = 481 px/s**, i.e. the glider crosses a
512-px room in 32 frames ≈ 1.06 s. Maximum sustained sink is **3 px/frame**; the 322-px
room height takes ~107 frames ≈ 3.6 s to fall through.

### 10.4 The four collision predicates

`GliderPRO/Sources/Interactions.c` contains **four** distinct glider-vs-rect tests with
different semantics. Confusing them is the fastest way to break the feel of the game.

| Function | Site | Test | Inset | Used by |
|---|---|---|---|---|
| `GliderHitTop` | `:54-97` | intersection, **x rewound by `wasHVel`** | 5 all sides, unconditional | `kDissolveIt` only (`:1223`) |
| `SectGlider` | `:101-130` | plain intersection | 5 all sides **iff `scrutinize`**; `top += 6` if burning | `CheckForHotSpots` (`:1639`, `:1657`, `:1679`, `:1723`), `Dynamics.c:42` |
| `GliderInRect` | `:134-150` | **containment** — glider entirely inside | **none** | `kMicrowaveIt`, `kWebIt`, `kMoveItUp/Down/Left/Right`, and 8 more |
| `BounceGlider` | `:154-167` | not a test — a resolver | none | `kBounceIt` (`:1591`) |

#### `GliderHitTop(gliderPtr thisGlider, Rect *theRect)` — `:54-97`

The comment banner at `GliderPRO/Sources/Interactions.c:52` reads `GliderHitSides` while
the function at `:54` is named `GliderHitTop`. **Both names are accurate**: the function
decides *which* it was and handles the side case itself.

```
 1. glideBounds.left   = thisGlider->dest.left   + 5;      // :60
 2. glideBounds.top    = thisGlider->dest.top    + 5;      // :61
 3. glideBounds.right  = thisGlider->dest.right  - 5;      // :62
 4. glideBounds.bottom = thisGlider->dest.bottom - 5;      // :63
 5.
 6. glideBounds.left  -= thisGlider->wasHVel;              // :65  REWIND x by last frame's hVel
 7. glideBounds.right -= thisGlider->wasHVel;              // :66
 8.
 9. hitTop = RectsIntersect(theRect, &glideBounds);        // :68-77, written as 4 early-outs
10.
11. if (!hitTop)                                           // :79 — it was a SIDE hit
12.     PlayPrioritySound(kFoilHitSound, kFoilHitPriority);// :81
13.     foilTotal--;                                       // :82
14.     if (foilTotal <= 0) StartGliderFoilLosing(thisGlider);  // :83-84
15.     glideBounds.left  += thisGlider->wasHVel;          // :86  un-rewind
16.     glideBounds.right += thisGlider->wasHVel;          // :87
17.     if (thisGlider->hVel > 0)
18.         offset = 2 + glideBounds.right - theRect->left;   // :89
19.     else
20.         offset = 2 + glideBounds.left  - theRect->right;  // :91
21.     thisGlider->hVel = -thisGlider->hVel - offset;     // :93
22.
23. return (hitTop);                                       // :96
```

The logic is: inset by 5, then **undo this frame's horizontal motion**. If the rewound box
*still* overlaps `theRect`, the overlap was caused by *vertical* motion → it is a **top**
hit and `true` is returned for the caller to handle. If the rewound box does *not* overlap,
the glider moved into the rect *sideways* → it is a **side** hit, and lines 12-21 charge one
foil and reflect. The reflection is **not** elastic: `hVel = -hVel - offset` where
`offset` = penetration depth **+ 2**. The `5`s, the `2`, and the `+6` in `SectGlider` are
**not** behind any `#define`; see §24.

**Quirk — the double foil charge.** The only caller, `kDissolveIt`
(`GliderPRO/Sources/Interactions.c:1218-1244`), does:

```c
				if (GliderHitTop(thisGlider, &(who->bounds)))
				{ StartGliderFadingOut(thisGlider); ... }
				else
				{
					if (foilTotal > 0)
					{
						foilTotal--;
						if (foilTotal <= 0) StartGliderFoilLosing(thisGlider);
					}
				}
```

so a *side* hit on a dissolve object decrements `foilTotal` **twice** — once at
`Interactions.c:82` inside the function and once at `:1232` in the caller — and can call
`StartGliderFoilLosing` twice. Reproduce as written; do not "fix" it.

#### `SectGlider(gliderPtr thisGlider, Rect *theRect, Boolean scrutinize)` — `:101-130`

```
1. glideBounds = thisGlider->dest;                                  // :106
2. if (thisGlider->mode == kGliderBurning) glideBounds.top += 6;    // :107-108  UNCONDITIONAL
3. if (scrutinize)                                                  // :110
4.     glideBounds.left += 5; glideBounds.top += 5;                 // :112-113
5.     glideBounds.right -= 5; glideBounds.bottom -= 5;             // :114-115
6. return RectsIntersect(theRect, &glideBounds);                    // :118-127
```

Note the **order**: the burning `+6` is applied first and unconditionally, so a burning
glider under scrutiny has `top += 11`. The `+6` accounts for the flame occupying the top of
the 26-px burning sprite (26 − 20 = 6).

`scrutinize` comes from the per-hot-spot field `hotObject.doScrutinize`
(`GliderPRO/Headers/GliderStructs.h:224`), passed at
`GliderPRO/Sources/Interactions.c:1640`, `:1658`, `:1680`, `:1724`. So **whether the
5-pixel inset applies is a per-object property**, set when the hot spot is built (§8).
`Dynamics.c:42` hard-codes `true`.

#### `GliderInRect(gliderPtr thisGlider, Rect *theRect)` — `:134-150`

```
1. glideBounds = thisGlider->dest;              // :138 — NO inset, NO burning adjustment
2. if (glideBounds.top    < theRect->top)    return false;   // :140-141
3. if (glideBounds.bottom > theRect->bottom) return false;   // :142-143
4. if (glideBounds.left   < theRect->left)   return false;   // :144-145
5. if (glideBounds.right  > theRect->right)  return false;   // :146-147
6. return true;                                              // :148-149
```

This is **containment, not intersection**: the whole 48 × 20 glider must fit inside
`theRect`. Every vent, fan, microwave and web uses it, which is why you must be fully over
a vent for it to lift you, and why webs only catch a fully-enclosed glider.

### 10.5 `BounceGlider` — the `kBounceIt` resolver

`GliderPRO/Sources/Interactions.c:154-167`, verbatim:

```c
void BounceGlider (gliderPtr thisGlider, Rect *theRect)
{
	Rect		glideBounds;
	
	glideBounds = thisGlider->dest;
	if ((theRect->right - glideBounds.left) < (glideBounds.right - theRect->left))
		thisGlider->hVel = theRect->right - glideBounds.left;
	else
		thisGlider->hVel = theRect->left - glideBounds.right;
	if (foilTotal > 0)
		PlayPrioritySound(kFoilHitSound, kFoilHitPriority);
	else
		PlayPrioritySound(kHitWallSound, kHitWallPriority);
}
```

Semantics:

1. Compute both penetration depths: from the left edge of the glider to the right edge of
   the rect (`theRect->right - glideBounds.left`), and from the left edge of the rect to
   the right edge of the glider (`glideBounds.right - theRect->left`).
2. Pick the **smaller** one — i.e. eject along the shallower axis.
3. **Overwrite** `hVel` with that signed depth. It is *not* a reflection and the previous
   `hVel` is discarded entirely; there is no restitution coefficient and **no `2`-pixel
   fudge**. The next `MoveGlider` then translates by exactly enough to clear the rect in
   one frame, after which `hDesiredVel` (reset to 0) decays it at 2/frame.
4. The sound depends on inventory: `kFoilHitSound` if `foilTotal > 0`, else
   `kHitWallSound` — with `kFoilHitPriority` / `kHitWallPriority` respectively. Note that
   `BounceGlider` does **not** consume foil.

Because step 3 can produce a large `hVel` (up to the full width of the rect) and
`MoveGlider` clamps to ±`kMaxHVel` = 16 *before* integrating, a glider deeply embedded in a
wide bouncer will need several frames to escape.

### 10.6 Escape detection — the `>> 6` tile lookups

All in `GliderPRO/Sources/Interactions.c`. Eight functions, all reached from
`CheckGliderInRoom` (`:689-753`):

| Function | Declared | Literal block | Tile values tested |
|---|---|---|---|
| `CheckEscapeUpTwo` | `:171` | `:200-209` | 5, 6 |
| `CheckEscapeUp` | `:244` | `:257-266` | 5, 6 |
| `CheckEscapeDownTwo` | `:283` | `:312-321` | 2, 3 |
| `CheckEscapeDown` | `:378` | `:391-397` | 2, 3 |
| `CheckEscapeLeftTwo` | `:509` | — | (uses `leftThresh`) |
| `CheckEscapeLeft` | `:572` | — | " |
| `CheckEscapeRightTwo` | `:599` | — | (uses `rightThresh`) |
| `CheckEscapeRight` | `:662` | — | " |

(The banner comments sit two lines above each declaration: `:169`, `:242`, `:281`, `:376`,
`:507`, `:570`, `:597`, `:660`.)

**Critical gate: the tile lookup only happens in `kDirt` rooms.** All four vertical
functions have the shape

```c
	if (topOpen)              /* or bottomOpen */
	{
		... normal escape via kNoCeilingLimit / kNoFloorLimit ...
	}
	else if (thisBackground == kDirt)      // :198, :255, :310, :389
	{
		... the >> 6 tile lookup below ...
	}
```

so tile values 2/3 and 5/6 are **not** general "floor/ceiling opening" markers — they are
only consulted when `thisBackground == kDirt` (2011). In any other closed room the
`else if` fails and nothing happens at all: the glider is left with
`dest.top < kCeilingLimit` / `dest.bottom > kFloorLimit` and is re-clamped by the ordinary
solid-surface handling. Inside a `kDirt` room, when the tile test *fails* the code clamps
explicitly — `thisGlider->vVel = kCeilingLimit - thisGlider->dest.top;`
(`Interactions.c:272`, and the `:274` `else` for out-of-range tiles) or the `kFloorLimit`
equivalent for the downward pair.

The vertical pattern, as literally written at `Interactions.c:200-209`:

```c
	leftTile = thisGlider->dest.left >> 6;		// ÷ 64
	rightTile = thisGlider->dest.right >> 6;	// ÷ 64
	
	if ((leftTile >= 0) && (leftTile < 8) &&
			(rightTile >= 0) && (rightTile < 8))
	{
		if (((thisTiles[leftTile] == 5) ||
				(thisTiles[leftTile] == 6)) &&
				((thisTiles[rightTile] == 5) ||
				(thisTiles[rightTile] == 6)))
```

with `(5, 6)` for **ceiling** openings (`CheckEscapeUp*`) and `(2, 3)` for **floor**
openings (`CheckEscapeDown*`). *Both* the left and right edges must be over an opening —
you cannot fall through a half-covered hole.

**Every constant in this block is a bare literal.** `6` is not `log2(kTileWide)`, `8` is
not `kNumTiles`, and the tile-code magic numbers 2/3/5/6 have no names anywhere in the
source. See §24.

`>> 6` on a negative `dest.left` is an **arithmetic** shift in C on 68k/PPC, so
`-1 >> 6 == -1` (not 0) — which is why the `>= 0` guard exists. Go's `>>` on a signed int
behaves identically, so this ports directly; a port using `/64` would get `0` for −1 and
mis-detect.

The `Two` variants are the two-player versions, which must check both gliders before
committing the room change. The horizontal variants do not use tiles at all: they compare
`dest.left`/`dest.right` against the `leftThresh`/`rightThresh` values computed by
`DetermineRoomOpenings` (§22.5), which are `kLeftWallLimit` 12 / `kNoLeftWallLimit` −24 /
`kRightWallLimit` 500 / `kNoRightWallLimit` 536.

### 10.7 Player.c function-local speeds, tabulated by mode

| Glider mode | `hVel` per frame | `vVel` per frame | Constant |
|---|---|---|---|
| `kGliderGoingUp` | — | −4 | `kClimbStairsSpeed` (`Player.c:317`) |
| `kGliderComingUp` | −4 | −4 | `kHClimbStairsSpeed`, `kVClimbStairsSpeed` (`Player.c:385-386`) |
| `kGliderGoingDown` | +4 | +4 | `kHDropStairsSpeed`, `kVDropStairsSpeed` (`Player.c:433-434`) |
| `kGliderComingDown` | +4 | +4 | same, redefined (`Player.c:514-515`) |
| `kGliderDuctingDown` | — | +4 | `kVDropDuctSpeed` (`Player.c:659`) |
| `kGliderDuctingUp` | — | −4 | `kVRiseDuctSpeed` (`Player.c:756`) |
| `kGliderMailOutLeft` | −4 | — | `kHPushMailSpeed` (`Player.c:853`) |
| `kGliderMailOutRight` | +4 | — | `kHPushMailRtSpeed` (`Player.c:891`) |
| `kGliderMailInLeft` | +4 | +2 | `kHMailPullSpeed`, `kVMailDropSpeed` (`Player.c:962-963`) |
| `kGliderMailInRight` | −4 | +2 | `kHMailPullRtSpeed`, `kVMailDropSpeed` (`Player.c:1053-1054`) |
| `kGliderShredding` | — | +1 or +4 | `kDropShredSlow` 1 / `kDropShredFast` 4 (`Player.c:1262-1263`) |

Note the mail-in speeds *pull the glider toward the mailbox*: entering the **left** mailbox
means moving right (+4), entering the **right** mailbox means moving left (−4). The
mail-out speeds are the opposite sign, pushing the glider away from the box it emerges
from.

### 10.8 Input handling

`GetInput(gliderPtr thisGlider)` — `GliderPRO/Sources/Input.c:281-379`:

```
 1. GetKeys(theKeys);                                    // Toolbox KeyMap, 128 bits
 2. if (BitTst(&theKeys, thisGlider->rightKey))          // right key held
 3.     LogDemoKey(0);                                   // #ifdef CREATEDEMODATA
 4.     if (BitTst(&theKeys, thisGlider->leftKey))       // BOTH keys held
 5.         ToggleGliderFacing(thisGlider);              // about-face instead of moving
 6.         thisGlider->heldLeft = true;
 7.     else
 8.         thisGlider->hDesiredVel += kNormalThrust;     // +5
 9.         thisGlider->tipped   = (thisGlider->facing == kFaceLeft);
10.         thisGlider->heldRight = true;
11. else if (BitTst(&theKeys, thisGlider->leftKey))
12.     LogDemoKey(1);
13.     thisGlider->hDesiredVel -= kNormalThrust;         // -5
14.     thisGlider->tipped   = (thisGlider->facing == kFaceRight);
15.     thisGlider->heldLeft = true;
16. if (BitTst(&theKeys, thisGlider->battKey)
17.  && batteryTotal != 0 && thisGlider->mode == kGliderNormal)
18.     LogDemoKey(2);
19.     if (batteryTotal > 0) DoBatteryEngaged(thisGlider);
20.     else                  DoHeliumEngaged(thisGlider);
21. if (BitTst(&theKeys, thisGlider->bandKey)
22.  && bandsTotal > 0 && thisGlider->mode == kGliderNormal)
23.     LogDemoKey(3);
24.     if (!thisGlider->fireHeld)
25.         if (AddBand(thisGlider, dest.left + 24, dest.top + 10, thisGlider->facing))
26.             bandsTotal--;
27.             if (bandsTotal <= 0) QuickBandsRefresh(false);
28.             thisGlider->fireHeld = true;
29. if (otherPlayerEscaped != kNoOneEscaped
30.  && BitTst(&theKeys, kDeleteKeyMap)
31.  && thisGlider->which && !onePlayerLeft)
32.     ForceKillGlider();                                // sacrifice yourself to catch up
```

`tipped` is the "banking" flag: pressing *against* the direction you face tips the glider.
The band spawn point is `(dest.left + 24, dest.top + 10)` — the sprite centre
(`kHalfGliderWide` = 24, `kGliderHigh`/2 = 10) — but written as bare literals **24** and
**10**, see §24.

`DoBatteryEngaged` (`GliderPRO/Sources/Input.c:~140-156`):

```
1. if (thisGlider->tipped) thisGlider->hVel -= kHyperThrust;   // -8, applied to hVel directly
2. else                    thisGlider->hVel += kHyperThrust;   // +8
3. batteryTotal--;
4. if (batteryTotal == 0)
5.     QuickBatteryRefresh(false);  PlayPrioritySound(kFizzleSound, kFizzlePriority);
6. else
7.     batteryFrame++;  if (batteryFrame >= 4) batteryFrame = 0;
8.     if (batteryFrame == 0) PlayPrioritySound(kThrustSound, kThrustPriority);
```

Note step 1-2: the battery writes **`hVel`**, bypassing the `hDesiredVel` accumulator
entirely — so battery thrust is instantaneous, and because `MoveGlider` clamps `hVel` to
±16 *after* the write, holding the battery pins you at exactly ±16.

`DoHeliumEngaged` (`GliderPRO/Sources/Input.c:160-182`):

```
1. thisGlider->vDesiredVel = -kHeliumLift;    // -4
2. batteryTotal++;                            // helium is NEGATIVE battery, counts up to 0
3. if (batteryTotal == 0)
4.     QuickBatteryRefresh(false);  PlayPrioritySound(kFizzleSound, kFizzlePriority);
5.     batteryWasEngaged = false;
6. else
7.     batteryFrame++;  if (batteryFrame >= 4) batteryFrame = 0;
8.     if (batteryFrame == 0) PlayPrioritySound(kHissSound, kHissPriority);
```

The single `batteryTotal` variable encodes **both** resources: positive = battery charges
remaining, negative = helium charges remaining, zero = empty. See §15.

The animation cycle length **4** in `batteryFrame >= 4` is a bare literal (§24).

### 10.9 Demo playback

`GetDemoInput(gliderPtr thisGlider)` — `GliderPRO/Sources/Input.c:186-277`.

```
 1. if (thisGlider->which == kPlayer1) GetKeys(theKeys);
 2. #if BUILD_ARCADE_VERSION
 3.     if (BitTst(&theKeys, thisGlider->leftKey) || ...rightKey || ...battKey || ...bandKey)
 4.         playing = false;  paused = false;            // any control exits the demo
 5. #else
 6.     if (BitTst(&theKeys, kCommandKeyMap)) DoCommandKey();
 7. #endif
 8. if (thisGlider->mode == kGliderBurning)
 9.     if (thisGlider->facing == kFaceLeft) thisGlider->hDesiredVel -= kNormalThrust;
10.     else                                 thisGlider->hDesiredVel += kNormalThrust;
11. else
12.     thisGlider->heldLeft = false;  thisGlider->heldRight = false;  thisGlider->tipped = false;
13.     if (gameFrame == (long)demoData[demoIndex].frame)
14.         switch (demoData[demoIndex].key)
15.         case 0:  thisGlider->hDesiredVel += kNormalThrust;
16.                  thisGlider->tipped = (thisGlider->facing == kFaceLeft);
17.                  thisGlider->heldRight = true;  thisGlider->fireHeld = false;  break;
18.         case 1:  thisGlider->hDesiredVel -= kNormalThrust;
19.                  thisGlider->tipped = (thisGlider->facing == kFaceRight);
20.                  thisGlider->heldLeft = true;   thisGlider->fireHeld = false;  break;
21.         case 2:  if (batteryTotal > 0) DoBatteryEngaged(thisGlider);
22.                  else DoHeliumEngaged(thisGlider);
23.                  thisGlider->fireHeld = false;  break;
24.         case 3:  if (!thisGlider->fireHeld)
25.                      if (AddBand(thisGlider, dest.left + 24, dest.top + 10, facing))
26.                          bandsTotal--;
27.                          if (bandsTotal <= 0) QuickBandsRefresh(false);
28.                          thisGlider->fireHeld = true;
29.                  break;
30.         demoIndex++;
31.     else
32.         thisGlider->fireHeld = false;
33. if ((isEscPauseKey && BitTst(&theKeys, kEscKeyMap))
34.  || (!isEscPauseKey && BitTst(&theKeys, kTabKeyMap)))
35.     // pause
```

`demoType` is `{long frame; char key; char padding;}`
(`GliderPRO/Headers/GliderStructs.h:334-339`) = **6 bytes** under `align=mac68k`. The
demo is a sparse event list: only frames where a key *state* changes are recorded, and
playback advances `demoIndex` only when `gameFrame` equals the stored frame. See §18.8 for
the empirical proof that `sizeof(demoType) == 6`.

**Defect (comment only):** the case labels in the C read `// left key` for `case 0` and
`// right key` for `case 1`, but `case 0` performs `hDesiredVel += kNormalThrust` and sets
`heldRight = true`, which is the *right*-key action, and `GetInput` calls `LogDemoKey(0)`
from its right-key branch. The behaviour is self-consistent; only the comments are
swapped. A porter reading the comments instead of the code will mirror the demo.
---

## 11. Dynamic (animated) objects

A "dynamic object" (`dynaType`, the array is named `dinahs[]`) is any object that animates
or moves. `AddDynamicObject()` (`GliderPRO/Sources/Dynamics3.c:187-554`) converts a static
`objectType` into a `dinahs[]` entry when the room is entered; `HandleDynamics()`
(`GliderPRO/Sources/Dynamics3.c:31-105`) steps them; `RenderDynamics()`
(`GliderPRO/Sources/Dynamics3.c:112-154`) draws the subset that needs sprite blitting
(only `kToaster`, `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`, `kBall`,
`kDrip`, `kFish`).

Capacity is `kMaxDynamicObs` = 18; `AddDynamicObject` returns −1 when full
(`GliderPRO/Sources/Dynamics3.c:194` and the `default:` case at `:547`).
`ZeroDinahs()` (`GliderPRO/Sources/Dynamics3.c:160-180`) marks slots free by setting
`dinahs[i].type = kObjectIsEmpty` (−1).

### 11.1 Per-object seeding table

Extracted from `GliderPRO/Sources/Dynamics3.c:187-554`. "offset" is the sprite's placement
relative to the object's own rect.

| Object | dest offset | `hVel` init | `vVel` init | `count` init | `frame` init | `timer` init | `position` init |
|---|---|---|---|---|---|---|---|
| `kSparkle` | (as object) | 0 | 0 | 0 | 0 | `RandomInt(60) + 15` (`Dynamics3.c:208`) | 0 |
| `kToaster` | `breadSrc[0]` centred, top-aligned to `where->top` | `where->top + 2` (**used as a clip line, not a velocity**) | `-velocity` (§12.1) | `velocity` | `(short)who->data.g.delay * 3` | = `frame` | 0 |
| `kMacPlus` | `+10, +7` | 0 | 0 | 0 | 0 | 0 or 30 | 0 |
| `kTV` | `+17, +10` | 0 | 0 | 0 | 0 | 0 | 0 |
| `kCoffee` | `+32, +57` | 0 | 0 | 0 | 0 | 200 if on | 0 |
| `kOutlet` | at origin | 0 | 0 | `((short)who->data.g.delay * 6) / kTicksPerFrame` | 0 | 0 | 0 |
| `kVCR` | `+64, +6` | 0 | 0 | 0 | 0 | 115 if on | 0 |
| `kStereo` | `+56, +20` | 0 | 0 | 0 | 0 | 0 | 0 |
| `kMicrowave` | `+14, +13`, then `dest.right = dest.left + 48` | 0 | 0 | 0 | 0 | 0 | 0 |
| `kBalloon` | `dest.bottom = kBalloonStart` (310) | 0 | **−2** | `((short)who->data.h.delay * 6) / kTicksPerFrame` | 0 | — | 0 |
| `kCopterLf` | `dest.top = kCopterStart` (8) | **−1** | **+2** | `((short)delay * 6) / kTicksPerFrame` | 0 | — | `dest.left` (`Dynamics3.c:429`) |
| `kCopterRt` | `dest.top = kCopterStart` (8) | **+1** | **+2** | same | 0 | — | `dest.left` |
| `kDartLf` | `dest.left = 0` | `+kDartVelocity` (6) | **+2** | same | 0 | — | `dest.top` (`Dynamics3.c:458`) |
| `kDartRt` | `dest.left = kRoomWide - RectWide(&dartSrc[0])` | `-kDartVelocity` (−6) | **+2** | same | 2 | — | `dest.top` |
| `kBall` | at `where->left, where->top` | 0 | `-velocity` (§12.2) | `-velocity` | 0 | 0 | `dest.bottom` (`Dynamics3.c:489`) |
| `kDrip` | (as object) | `dest.top` (**stored as the origin y**) | 0 | 0 | 3 | — | `dest.top + who->data.h.length` (`Dynamics3.c:507`) |
| `kFish` | `+10, +8` | `((short)who->data.h.delay * 6) / kTicksPerFrame` (**stored as the delay**) | `-velocity` (§12.2) | `-velocity` | 0 | = `hVel` | `dest.bottom` (`Dynamics3.c:538`) |

Common to every case: `whole = dest`, `room = room`, `byte0 = (Byte)index`, `byte1 = 0`,
`moving = false`, `active = isOn`.

**`hVel`, `count` and `position` are heavily overloaded.** In `dynaType` they are declared
as velocity/counter/position but for several object types they carry unrelated data:

| Object | Field | Actually holds |
|---|---|---|
| `kToaster` | `hVel` | the y of the launch clip line (`where->top + 2`) |
| `kDrip` | `hVel` | the drip's origin y (to snap back to after a drop) |
| `kFish` | `hVel` | the respawn delay in frames |
| `kOutlet` | `hVel` | the room's light count, written by `UpdateOutletsLighting()` (`GliderPRO/Sources/Trip.c`) |
| `kToaster` | `count` | the launch velocity magnitude (positive) |
| `kBall`, `kFish` | `count` | the launch velocity, **negative** |
| enemies | `count` | the respawn delay in frames |
| `kCopterLf/Rt` | `position` | the spawn x |
| `kDartLf/Rt` | `position` | the spawn y |
| `kBall`, `kFish` | `position` | the ground/water line y |
| `kDrip` | `position` | the landing y |

A Go port should split these into named fields; the aliasing carries no semantic content,
it is purely a memory economy of 1994.

### 11.2 Appliance animation timers

All from `GliderPRO/Sources/Dynamics.c`. These are frame counts at 30 fps.

| Object | Behaviour | Sites |
|---|---|---|
| `kSparkle` | re-arms with `timer = RandomInt(240) + 60` — i.e. 60…299 frames (2…10 s) | `GliderPRO/Sources/Dynamics.c:304` |
| `kMacPlus` | plays `kMacOnSound` when `timer == 30` | `GliderPRO/Sources/Dynamics.c:406` |
| `kCoffee` | re-arms with `timer = 200 + RandomInt(200)` (200…399 frames); plays `kCoffeeSound` at `timer == 100` | `GliderPRO/Sources/Dynamics.c:492`, `:506`, `:503` |
| `kOutlet` | plays `kZapSound` whenever `(timer % 5) == 0`; `frame` wraps to **1** (not 0) at `kNumOutletPicts` (4); re-arms with `timer = kLengthOfZap` (30) | `GliderPRO/Sources/Dynamics.c:560`, `:~575`, `:592` |
| `kVCR` | timer landmarks **115**, **5**, **100**, **101** | `GliderPRO/Sources/Dynamics.c:613-632` |
| `kMicrowave` | draws `microOn`/`microOff` three times at 16-px horizontal steps | `GliderPRO/Sources/Dynamics.c:730-770` |

`Trip.c` supplies the *trigger* side (when a switch throws):

| Function | Sets | Site |
|---|---|---|
| `ToggleMacPlus` | `timer = 40` when turning on, `timer = 10` when turning off | `GliderPRO/Sources/Trip.c` |
| `ToggleTV`, `ToggleCoffee`, `ToggleVCR`, `ToggleMicrowave` | `timer = 4` | `GliderPRO/Sources/Trip.c` |
| `ToggleStereos` | only acts `if (timer == 0)`, then `timer = 4` | `GliderPRO/Sources/Trip.c` |
| `TriggerToast` | `vVel = (short)-dinahs[who].count; frame = 0; moving = true;` + `kToastLaunchSound`; if already moving, `frame = timer` | `GliderPRO/Sources/Trip.c` |
| `TriggerOutlet` | `position = 1; timer = kLengthOfZap;` + `kZapSound`; else `timer = count` | `GliderPRO/Sources/Trip.c` |
| `TriggerDrip` | `if (!moving && timer > 7) timer = 7;` | `GliderPRO/Sources/Trip.c` |
| `TriggerFish` | `whole = dest; moving = true; frame = 4;` + `kFishOutSound` | `GliderPRO/Sources/Trip.c` |
| `TriggerBalloon`, `TriggerCopter`, `TriggerDart` | `timer = kStartSparkle + 1` (= 5) | `GliderPRO/Sources/Trip.c` |
| `UpdateOutletsLighting` | stores the room's `nLights` into `dinahs[i].hVel` | `GliderPRO/Sources/Trip.c` |

### 11.3 Frame counts

All from `GliderPRO/Headers/GliderDefines.h:445-458`. These are the number of animation
frames in each sprite strip.

| Name | Value | Site |
|---|---|---|
| `kNumTrackLights` | 3 | `:445` |
| `kNumOutletPicts` | 4 | `:446` |
| `kNumCandleFlames` | 5 | `:447` |
| `kNumTikiFlames` | 5 | `:448` |
| `kNumBBQCoals` | 4 | `:449` |
| `kNumPendulums` | 3 | `:450` |
| `kNumBreadPicts` | 6 | `:451` |
| `kNumBalloonFrames` | 8 | `:452` |
| `kNumCopterFrames` | 10 | `:453` |
| `kNumDartFrames` | 4 | `:454` |
| `kNumBallFrames` | 2 | `:455` |
| `kNumDripFrames` | 6 | `:456` |
| `kNumFishFrames` | 8 | `:457` |
| `kNumFlowers` | 6 | `:458` |

`kNumBreadPicts` sizes `Rect breadSrc[kNumBreadPicts]` (`GliderPRO/Sources/Dynamics.c:20`).
`kNumFlowers` is used as `wasFlower = RandomInt(kNumFlowers);`
(`GliderPRO/Sources/InterfaceInit.c:160`) to pick which flower a new `kFlower` object shows.

---

## 12. Enemies and the reverse-engineered launch velocity

### 12.1 Toaster — full-rate launch (`GliderPRO/Sources/Dynamics3.c:224-231`)

The author's comment is literally `// reverse engineer init. vel.` The problem being
solved: *given a desired peak height, find the integer initial velocity that reaches it
under 1 px/frame² gravity.*

```
1. position = who->data.g.height;     // desired launch height in pixels
2. velocity = 0;
3. do
4. {
5.     velocity++;
6.     position -= velocity;
7. }
8. while (position > 0);
9. dinahs[n].vVel  = -velocity;       // launch upward
10. dinahs[n].count =  velocity;      // remembered so TriggerToast can relaunch
```

This is the inverse of the triangular-number sum: it finds the smallest `velocity` with
1 + 2 + … + velocity ≥ `height`, i.e. `velocity = ceil((sqrt(8·height + 1) − 1) / 2)`.
Table of the mapping (verified by running the loop):

| `data.g.height` | resulting `velocity` | triangular sum |
|---|---|---|
| 1 | 1 | 1 |
| 2, 3 | 2 | 3 |
| 4, 5, 6 | 3 | 6 |
| 7…10 | 4 | 10 |
| 11…15 | 5 | 15 |
| 16…21 | 6 | 21 |
| 22…28 | 7 | 28 |
| 29…36 | 8 | 36 |
| 37…45 | 9 | 45 |
| 46…55 | 10 | 55 |
| 56…66 | 11 | 66 |
| 67…78 | 12 | 78 |
| 79…91 | 13 | 91 |
| 92…105 | 14 | 105 |
| 106…120 | 15 | 120 |
| 121…136 | 16 | 136 |
| 137…153 | 17 | 153 |
| 154…171 | 18 | 171 |
| 172…190 | 19 | 190 |
| 191…210 | 20 | 210 |
| 211…231 | 21 | 231 |
| 232…253 | 22 | 253 |
| 254…276 | 23 | 276 |
| 277…300 | 24 | 300 |
| 301…322 | 25 | 325 |

A Go port **must** reproduce the loop (or the exact `ceil` formula), not a float
approximation: `height = 21` gives 6 but `height = 22` gives 7, and toast trajectories in
authored houses depend on the step.

### 12.2 Ball and fish — half-rate launch (`GliderPRO/Sources/Dynamics3.c:472-483`, `:522-533`)

The ball and fish accelerate only on `evenFrame`, so their solver increments `velocity`
only every other iteration:

```
 1. position = who->data.h.length;   // ball;  the FISH uses who->data.g.height (Dynamics3.c:522)
 2. velocity = 0;
 3. evenFrame = true;                // <-- writes the GLOBAL, see the defect note below
 4. lilFrame = true;
 5. do
 6. {
 7.     if (lilFrame)
 8.         velocity++;
 9.     lilFrame = !lilFrame;
10.     position -= velocity;
11. }
12. while (position > 0);
13. dinahs[n].vVel  = -velocity;
14. dinahs[n].count = -velocity;     // NEGATIVE here, unlike the toaster
```

Each velocity value is applied for **two** iterations (1, 1, 2, 2, 3, 3, …), so the
cumulative distance after finishing velocity `v` is `v·(v+1)`. The mapping (computed by
running the loop for every input 1…322):

| `length` (ball) / `height` (fish) | resulting `velocity` | cumulative distance |
|---|---|---|
| 1…2 | 1 | 2 |
| 3…6 | 2 | 6 |
| 7…12 | 3 | 12 |
| 13…20 | 4 | 20 |
| 21…30 | 5 | 30 |
| 31…42 | 6 | 42 |
| 43…56 | 7 | 56 |
| 57…72 | 8 | 72 |
| 73…90 | 9 | 90 |
| 91…110 | 10 | 110 |
| 111…132 | 11 | 132 |
| 133…156 | 12 | 156 |
| 157…182 | 13 | 182 |
| 183…210 | 14 | 210 |
| 211…240 | 15 | 240 |
| 241…272 | 16 | 272 |
| 273…306 | 17 | 306 |
| 307…322 | 18 | 342 (truncated by the 322-px room) |

For comparison, the toaster's full-rate mapping (also computed by running its loop for
every input 1…322) is the triangular-number table above; the two differ substantially —
`height = 100` gives a toast velocity of **14** but a ball/fish velocity of **10**.

**Defect:** lines 3 of both loops assign to the **global** `Boolean evenFrame`
(declared `extern Boolean evenFrame;` at `GliderPRO/Sources/Dynamics3.c:23`, defined at
`GliderPRO/Sources/Play.c:53`, toggled at `GliderPRO/Sources/Play.c:435`). So *seeding a
ball or a fish resets the whole game's frame parity to `true`*. Because `evenFrame` gates
drip animation, foil wear (`if (evenFrame && foilTotal > 0) foilTotal--`) and the web twang,
entering a room that contains a ball or fish nudges all of those by one frame. The
variable was almost certainly intended to be local (`lilFrame` next to it *is* local, at
`GliderPRO/Sources/Dynamics3.c:191`).

### 12.3 Enemy constants

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kBalloonStop` | 8 | `GliderPRO/Sources/Dynamics2.c:13` | y at which a rising balloon stops (top of room) |
| `kBalloonStart` | 310 | `GliderPRO/Sources/Dynamics2.c:14`, `Dynamics3.c:13` | y at which a balloon respawns (`dest.bottom`) |
| `kCopterStart` | 8 | `GliderPRO/Sources/Dynamics2.c:15`, `Dynamics3.c:14` | y at which a copter respawns (`dest.top`) |
| `kCopterStop` | 310 | `GliderPRO/Sources/Dynamics2.c:16` | y at which a descending copter is recycled |
| `kDartVelocity` | 6 | `GliderPRO/Sources/Dynamics2.c:17`, `Dynamics3.c:15` | dart horizontal speed |
| `kDartStop` | 310 | `GliderPRO/Sources/Dynamics2.c:18` | y at which a dart is recycled |
| `kEnemyDropSpeed` | 8 | `GliderPRO/Sources/Dynamics2.c:19` | downward speed of a killed balloon/copter |

### 12.4 Per-enemy state machines

All in `GliderPRO/Sources/Dynamics2.c`. Frame indices refer to the sprite strips sized by
§11.3.

**Balloon** (`kNumBalloonFrames` = 8):
```
1. Rising: frames 0…5 cycle; vVel = -2 (set at seed).
2. Popped: frame = 6; vVel = kEnemyDropSpeed (8); PlayPrioritySound(kPopSound, kPopPriority).
3. Falling: frames 6…7 cycle.
4. Recycle: vVel = -2; dest.bottom = kBalloonStart (310).
```

**Copter** (`kNumCopterFrames` = 10):
```
1. Flying: frames 0…7 cycle; hVel = -1 (kCopterLf) or +1 (kCopterRt); vVel = 2.
2. Shot: frame = 8; hVel = 0; vVel = kEnemyDropSpeed (8);
        PlayPrioritySound(kPaperCrunchSound, kPaperCrunchPriority).
3. Falling: frames 8…9 cycle.
4. Recycle: vVel = 2; hVel = -1 or +1; dest.top = kCopterStart (8);
        dest.right = dest.left + 32.
```

**Dart** (`kNumDartFrames` = 4; frames 0/1 = left-facing normal/shot, 2/3 = right-facing):
```
1. Flying: hVel = ∓kDartVelocity (6); vVel = 2.
2. Shot: frame = 1 (kDartLf) or 3 (kDartRt).
3. Recycle when (dest.left <= 0 || dest.right >= kRoomWide || dest.bottom >= kDartStop):
        vVel = 2; frame = 0 (Lf) or 2 (Rt); hVel = ∓kDartVelocity.
```

**Ball** (`kNumBallFrames` = 2, `GliderPRO/Sources/Dynamics2.c:~370-422`):
```
1. Gravity: if (evenFrame) vVel++;                   // Dynamics2.c:409 — 0.5 px/frame^2
2. Bounce when dest.bottom >= position:
3.     dest.top = dest.bottom - 32;                  // snap the 32-px sprite onto the line
4.     if (active) vVel = count;                     // count is NEGATIVE: relaunch at full height
5.     else        vVel = -((vVel * 3) / 4);         // damped rebound: 75 % of impact speed
6.     if (vVel == 0) stop;                          // integer truncation eventually kills it
7. Idle relaunch (Dynamics2.c:416-421):
8.     if (active) { vVel = count; moving = true; evenFrame = true; }   // <-- global write again
```
The `whole` (dirty) rect is built directionally:
`if (vVel > 0) whole.top -= vVel; else whole.bottom -= vVel;`
(`GliderPRO/Sources/Dynamics2.c:404-407`).
Step 5 is a 3/4 restitution done in **truncating integer** arithmetic. A ball landing at
`vVel = +24` rebounds at −18, then lands at +18 → −13 → +13 → −9 → −6 → −4 → −3 → −2 → −1
→ 0, i.e. it dies after 9 bounces. (24·3/4 = 18, 18·3/4 = 13 not 13.5, 13·3/4 = 9, 9·3/4 = 6,
6·3/4 = 4, 4·3/4 = 3, 3·3/4 = 2, 2·3/4 = 1, 1·3/4 = 0.) Go's integer division truncates
identically, so this ports directly — but a float port would never reach exactly 0 and the
ball would bounce forever.

**Drip** (`kNumDripFrames` = 6, `HandleDrip` at `GliderPRO/Sources/Dynamics2.c:427-494`):
```
 1. if (moving):
 2.     if (evenFrame) frame = 9 - frame;            // Dynamics2.c:434 — alternates 4 <-> 5
 3.     CheckDynamicCollision(who, &theGlider, false);
 4.     VOffsetRect(&dest, vVel);
 5.     if (dest.bottom >= position)                 // hit the landing line
 6.         dest.top    = hVel;                      // hVel holds the ORIGIN y — snap back up
 7.         dest.bottom = dest.top + 12;
 8.         PlayPrioritySound(kDropSound, kDropPriority);
 9.         vVel  = 0;
10.         timer = count;
11.         frame = 3;
12.         moving = false;
13.     else
14.         whole = dest;  whole.top -= vVel;
15.         if (evenFrame) vVel++;                   // Dynamics2.c:470 — half-rate gravity
16. else if (active):
17.     timer--;
18.     if (timer == 6) frame = 0;                   // swelling
19.     else if (timer == 4) frame = 1;
20.     else if (timer == 2) frame = 2;
21.     else if (timer <= 0)
22.         VOffsetRect(&dest, 3);                   // detach: nudge down 3 px
23.         whole = dest;  moving = true;  frame = 4;
24.         PlayPrioritySound(kDripSound, kDripPriority);
```
`frame = 9 - frame` starting from 4 alternates **4 ↔ 5** — both valid indices in the
6-frame strip. Frame 3 is the landed splash; 0/1/2 are the swelling-droplet countdown.
The drip's `count` (the respawn delay) and `hVel` (the origin y) are both overloaded, §11.1.

**Fish** (`kNumFishFrames` = 8):
```
1. Leaping: frames advance while (vVel >= 0 && frame < 7).
2. Splash: dest.top = dest.bottom - 16; whole.top -= 2;
        PlayPrioritySound(kDropSound, ...); PlayPrioritySound(kFishInSound, kFishInPriority).
3. Idle wiggle when ((timer & 0x0003) == 0x0003): frames 0…3 with a ±1 vertical bob.
```

### 12.5 Foil vs dynamic objects

`CheckDynamicCollision()` (`GliderPRO/Sources/Dynamics.c:34-74`):

```
1. if the glider has foil:
2.     if (IsRectLeftOfRect(&dinahs[who].dest, &thisGlider->dest))
3.         thisGlider->hDesiredVel =  kShoveVelocity;    // +8, shoved right
4.     else
5.         thisGlider->hDesiredVel = -kShoveVelocity;    // -8, shoved left
6.     if (dinahs[who].vVel < 0)
7.         thisGlider->vDesiredVel = dinahs[who].vVel;   // inherit upward motion only
8.     PlayPrioritySound(kFoilHitSound, kFoilHitPriority);
9.     if (evenFrame && foilTotal > 0) foilTotal--;      // foil wears at 15 Hz
10. else
11.     the glider dies / burns / is destroyed per the object type
```

Note step 3-4 write `hDesiredVel` (which `MoveGlider` resets to 0 next frame), so the
shove is a one-frame impulse toward ±8, reachable only if the collision persists.

`DidBandHitDynamic()` (`GliderPRO/Sources/Dynamics.c:80-106`) — **defect**: the local
`Boolean collided;` is only assigned inside the `for (i = 0; i < numBands; i++)` loop, so
when `numBands == 0` the function returns an uninitialised value. In practice `numBands`
is ≥ 1 whenever this is called from the band update path, which is why the bug never
surfaced. A Go port gets `false` for free.

---

## 13. Rubber bands, grease, and webs

### 13.1 Rubber bands

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kMaxRubberBands` | 2 | `GliderPRO/Headers/GliderDefines.h:261` | at most 2 bands in flight at once |
| `kRubberBandVelocity` | 20 | `GliderPRO/Sources/RubberBands.c:12` | muzzle speed, px/frame |
| `kBandFallCount` | 4 | `GliderPRO/Sources/RubberBands.c:13` | frames before a band starts to fall |
| `kKillBandMode` | −1 | `GliderPRO/Sources/RubberBands.c:14` | `bandType.mode` sentinel meaning "dead" |
| `kBandsSupply` | 8 | `GliderPRO/Sources/Interactions.c:18` | bands added by a `kBands` pickup |

Firing (`GliderPRO/Sources/RubberBands.c:276`, `:282`):
`hVel = -kRubberBandVelocity;` when facing left, `hVel = kRubberBandVelocity;` when facing
right. The band spawns at `(dest.left + 24, dest.top + 10)`
(`GliderPRO/Sources/Input.c:~305`) — the glider's sprite centre.

`GliderPRO/Sources/RubberBands.c:223`: `if (bands[i].count >= kBandFallCount)` — after 4
frames the band begins arcing downward. `kKillBandMode` is written at
`GliderPRO/Sources/RubberBands.c:101`, `:113`, `:198`, `:202`, `:244`.

Sprite: `bandsSrcRect` = 16 × 18 (the comment says "304 pixels", i.e. 16 × 19) with mask
`kRubberBandsPictID + 1000` = 5007; `bandRects[0..2]` = 16 × 6 at y = 0, 6, 12
(`GliderPRO/Sources/StructuresInit.c:~230`).

### 13.2 Grease

`greaseType` state machine, `GliderPRO/Sources/Grease.c:15-18`:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kGreaseIdle` | 0 | `GliderPRO/Sources/Grease.c:17` | bottle intact, not yet tipped |
| `kGreaseFalling` | 1 | `GliderPRO/Sources/Grease.c:18` | bottle falling |
| `kGreaseSpreading` | 2 | `GliderPRO/Sources/Grease.c:19` | puddle growing toward `data.c.length` |
| `kGreaseSpiltIdle` | 3 | `GliderPRO/Sources/Grease.c:20` | puddle at full size, slippery |
| `kMaxGrease` | 16 | `GliderPRO/Headers/GliderDefines.h:262` | simultaneous puddles |

`kGreaseRt` (`0x28`) spreads rightward, `kGreaseLf` (`0x29`) leftward; the length comes
from `data.c.length` (the union arm's "grease spill" field, §7.3). The
`kGreaseSpillSound` / `kGreaseSpillPriority` (806) pair accompanies the spread. A glider on
a `kGreaseSpiltIdle` puddle gets `sliding = true`, which suppresses
`CheckRoofCollision` (§5.4 step 3).

### 13.3 Cobwebs — `WebGlider`

`GliderPRO/Sources/Interactions.c:1738-1776`. `kKillWebbedGlider` = **150** frames (5 s) at
`GliderPRO/Sources/Interactions.c:1738`.

```
 1. if (thisGlider->mode == kGliderBurning && GliderInRect(thisGlider, webBounds))
 2.     thisGlider->wasMode = 0;
 3.     StartGliderFadingOut(thisGlider);
 4.     PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
 5.     return;                       // a burning glider BURNS the web away and dies instantly
 6.
 7. hDist = ((webBounds->right  - thisGlider->dest.right)
 8.        + (webBounds->left   - thisGlider->dest.left))  >> 3;   // Interactions.c:1749-1750
 9. vDist = ((webBounds->bottom - thisGlider->dest.bottom)
10.        + (webBounds->top    - thisGlider->dest.top))   >> 3;   // Interactions.c:1751-1752
11.
12. if (thisGlider->hDesiredVel != 0)                    // player is struggling
13.     if (evenFrame)                                   // ...but only acts every other frame
14.         thisGlider->hVel = hDist;
15.         thisGlider->vVel = vDist;
16.         PlayPrioritySound(kWebTwangSound, kWebTwangPriority);
17.     // NOTE: on odd frames NOTHING happens — vDesiredVel keeps kGravity, so the
18.     //       glider sinks 1 frame in 2 while struggling.
19. else                                                 // not struggling: pinned
20.     thisGlider->hDesiredVel = 0;                     // (already 0 — a no-op)
21.     thisGlider->vDesiredVel = 0;                     // cancels kGravity: hangs motionless
22.
23. thisGlider->wasMode++;                               // reused as a stuck-frame counter
24. if (thisGlider->wasMode >= kKillWebbedGlider)         // 150 frames = ~5 s
25.     thisGlider->wasMode = 0;
26.     StartGliderFadingOut(thisGlider);
27.     PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
```

The `>> 3` is a **÷8 damping**: the sum of the two edge deltas (≈ 2× the centre delta) is
divided by 8, so the pull-back velocity is about ¼ of the displacement per twang.

Three subtleties a port will get wrong:

1. The `evenFrame` test is **nested inside** the `hDesiredVel != 0` test, not `&&`-ed with
   it. On an odd frame while struggling, neither branch runs, so `vDesiredVel` retains
   `kGravity` = 3 and the glider sinks. Rewriting lines 12-13 as a single `&&` with an
   `else` would move the "pinned" behaviour onto odd frames and the glider would never sink.
2. Line 20 assigns 0 to a variable that the guard has already proven non-zero *in the other
   branch* — it is dead code in the `else`. Harmless, but do not "fix" it into the wrong
   branch.
3. `wasMode` — normally the saved previous glider mode — is repurposed as the stuck-frame
   counter. `kWasBurning` (2, `GliderPRO/Headers/GliderDefines.h:562`) is *also* stored in
   `wasMode`, and `kFramesToBurn` (60, `GliderPRO/Sources/Modes.c:408`) is *also* counted in
   `wasMode`. A Go port must not give `wasMode` a `GliderMode` type.

### 13.4 `evenFrame` — the global frame-parity flag, and who clobbers it

`Boolean evenFrame` is defined at `GliderPRO/Sources/Play.c:53`, initialised
`evenFrame = false;` at `GliderPRO/Sources/InterfaceInit.c:131`, and toggled once per frame
at `GliderPRO/Sources/Play.c:435` (`evenFrame = !evenFrame;`). Complete list of readers and
writers:

| Site | Role |
|---|---|
| `GliderPRO/Sources/Play.c:435` | **writer**: toggles each frame |
| `GliderPRO/Sources/InterfaceInit.c:131` | **writer**: `= false` at startup |
| `GliderPRO/Sources/Dynamics2.c:420` | **writer**: `= true` when a ball relaunches |
| `GliderPRO/Sources/Dynamics3.c:474` | **writer**: `= true` while seeding a `kBall` |
| `GliderPRO/Sources/Dynamics3.c:524` | **writer**: `= true` while seeding a `kFish` |
| `GliderPRO/Sources/Dynamics.c:60` | reader: `if (evenFrame && foilTotal > 0) foilTotal--;` — foil wears at 15 Hz |
| `GliderPRO/Sources/Dynamics.c:327` | reader: toast bread frame advances at 15 Hz (`kNumBreadPicts` 6) |
| `GliderPRO/Sources/Dynamics2.c:38`, `:77` | reader: balloon / copter frame advance at 15 Hz |
| `GliderPRO/Sources/Dynamics2.c:409` | reader: ball gravity `vVel++` at 15 Hz |
| `GliderPRO/Sources/Dynamics2.c:433` | reader: drip frame flicker at 15 Hz |
| `GliderPRO/Sources/Dynamics2.c:471` | reader: drip gravity `vVel++` at 15 Hz |
| `GliderPRO/Sources/Dynamics2.c:551` | reader: fish animation at 15 Hz |
| `GliderPRO/Sources/Interactions.c:1756` | reader: web twang at 15 Hz |
| `GliderPRO/Sources/Render.c:649` | reader: `if (evenFrame) RenderFlames(); else RenderStars();` — flames and stars each render at 15 Hz, on **opposite** parities |

The three `= true` writes are almost certainly accidental (the neighbouring `lilFrame` in
`AddDynamicObject` *is* a local, `GliderPRO/Sources/Dynamics3.c:191`). Their observable
effect: entering a room containing a ball or a fish, or a ball relaunching, resets the
parity, which shifts flame/star rendering, foil wear and web struggling by one frame. A Go
port that makes `evenFrame` a clean `frameCount & 1` will not reproduce this — probably a
good thing, but note it as a deliberate deviation.

---

## 14. Scoring

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kRoomVisitScore` | 100 | `GliderPRO/Headers/GliderDefines.h:536` | awarded the first time a room is entered |
| `kRedClockPoints` | 100 | `GliderPRO/Headers/GliderDefines.h:537` | `kRedClock` |
| `kBlueClockPoints` | 300 | `GliderPRO/Headers/GliderDefines.h:538` | `kBlueClock` |
| `kYellowClockPoints` | 500 | `GliderPRO/Headers/GliderDefines.h:539` | `kYellowClock` |
| `kCuckooClockPoints` | 1000 | `GliderPRO/Headers/GliderDefines.h:540` | `kCuckoo` |
| `kStarPoints` | 5000 | `GliderPRO/Headers/GliderDefines.h:541` | `kStar` |
| `kMaxScores` | 10 | `GliderPRO/Headers/GliderDefines.h:249` | high-score table size |
| `kScoreRollAmount` | 13 | `GliderPRO/Sources/Scoreboard.c:21` | scoreboard "rolls up" 13 points per frame |
| `kMaxFlyingPts` | 3 | `GliderPRO/Headers/GliderDefines.h:253` | simultaneous floating "+100" popups |
| `kMaxFlyingPointsLoop` | 24 | `GliderPRO/Headers/GliderDefines.h:254` | frames a popup lives |

`theScore` is a `long` (`GliderPRO/Headers/GliderVars.h:~`), so scores are int32.

### 14.1 `HandleRewards` — the complete prize table

`GliderPRO/Sources/Interactions.c:754-982`. The velocity damping on clock pickups is the
non-obvious part:

| Object | Action |
|---|---|
| `kRedClock` | `AddFlyingPoint(&bounds, 100, hVel / 2, vVel / 2);` then `hVel /= 4; vVel /= 4;` then `theScore += kRedClockPoints;` |
| `kBlueClock` | same with 300 / `kBlueClockPoints` |
| `kYellowClock` | same with 500 / `kYellowClockPoints` |
| `kCuckoo` | same with 1000 / `kCuckooClockPoints`, **plus** `StopPendulum()` |
| `kPaper` | `hVel /= 2; vVel /= 2;` then `mortals++` (extra life). No score. |
| `kBattery` | `if (batteryTotal > 0) batteryTotal += kBatterySupply; else batteryTotal = kBatterySupply;` |
| `kBands` | `bandsTotal += kBandsSupply;` (8) |
| `kFoil` | `foilTotal += kFoilSupply;` (8) then `StartGliderFoilGoing()` |
| `kInvisBonus` | `theScore += who->data.c.points;` (author-specified value) |
| `kStar` | `numStarsRemaining--; if (numStarsRemaining <= 0) FlagGameOver(); theScore += kStarPoints;` |
| `kHelium` | `if (batteryTotal < 0) batteryTotal -= kHeliumSupply; else batteryTotal = -kHeliumSupply;` |
| `kSparkle` | **no-op** (pure decoration) |
| `kSlider` | **no-op** |

Note the asymmetry: clocks divide velocity by **4** and pass **half** the velocity to the
flying-points sprite; paper divides by **2** and spawns no sprite. The `2` and `4` are bare
literals (§24).

Note also `kBattery`/`kHelium`: picking up a battery while you hold helium (`batteryTotal
< 0`) **replaces** the helium with 50 battery charges rather than adding, and vice versa.

`kStar` decrements a *house-wide* counter — see `bannerStarCountOn` in §22.2.

### 14.2 Scoreboard

`GliderPRO/Sources/Scoreboard.c` and `GliderPRO/Sources/StructuresInit.c:59-163`.

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kScoreboardHigh` | 0 | `GliderPRO/Headers/GliderDefines.h:513` | scoreboard docked at the top |
| `kScoreboardLow` | 1 | `GliderPRO/Headers/GliderDefines.h:514` | scoreboard docked at the bottom |
| `kScoreboardTall` | 20 | `GliderPRO/Headers/GliderDefines.h:515` | its height in pixels |
| `kScoreboardPictID` | 1997 | `GliderPRO/Headers/GliderDefines.h:623` | its background PICT |
| `kBadgePictID` | 1996 | `GliderPRO/Sources/StructuresInit.c:39` | the 4 inventory badges (32 × 66) |
| `kGrayBackgroundColor` | 251 | `GliderPRO/Sources/Scoreboard.c:15` | 8-bit palette index for the grey behind the score |
| `kGrayBackgroundColor4` | 10 | `GliderPRO/Sources/Scoreboard.c:16` | 4-bit-grey equivalent |
| `kFoilBadge` | 0 | `GliderPRO/Sources/Scoreboard.c:17` | badge index |
| `kBandsBadge` | 1 | `GliderPRO/Sources/Scoreboard.c:18` | badge index |
| `kBatteryBadge` | 2 | `GliderPRO/Sources/Scoreboard.c:19` | badge index |
| `kHeliumBadge` | 3 | `GliderPRO/Sources/Scoreboard.c:20` | badge index |
| `kScoreRollAmount` | 13 | `GliderPRO/Sources/Scoreboard.c:21` | `displayedScore += kScoreRollAmount` (`GliderPRO/Sources/Scoreboard.c:84`) |
| `kFoilLow` | 2 | `GliderPRO/Sources/Scoreboard.c:74` | 25 % of `kFoilSupply` 8 |
| `kBatteryLow` | 17 | `GliderPRO/Sources/Scoreboard.c:75` | ~25 % of `kBatterySupply` 50 (comment: "25%") |
| `kHeliumLow` | −38 | `GliderPRO/Sources/Scoreboard.c:76` | ~25 % of `-kHeliumSupply` −150 |
| `kBandsLow` | 2 | `GliderPRO/Sources/Scoreboard.c:77` | 25 % of `kBandsSupply` 8 |

`InitScoreboardMap()` layout, from `GliderPRO/Sources/StructuresInit.c:59-163`:

```
 1. wasScoreboardMode = kScoreboardHigh;
 2. boardSrcRect = houseRect; ZeroRectCorner(&boardSrcRect); boardSrcRect.bottom = kScoreboardTall;
 3. if (boardSrcRect.right >= 640) hOffset = (RectWide(&boardSrcRect) - kMaxViewWidth) / 2;
 4. else                           hOffset = -576;
 5. draw PICT kScoreboardPictID (1997) offset by hOffset
 6. badgeSrcRect = 32 x 66  ("2144 pixels"); LoadGraphic(kBadgePictID) [1996]
 7. boardDestRect = boardSrcRect offset by (0, -kScoreboardTall)
 8. hOffset = (RectWide(&houseRect) - 640) / 2;  if (hOffset < 0) hOffset = -128;
 9. boardTSrcRect = 256 x 12,  dest at (137 + hOffset, 5)     // "Total" score field
10. boardGSrcRect =  20 x 10,  dest at (526 + hOffset, 5)     // Gliders remaining
11. boardPSrcRect =  64 x 10,  dest at (570 + hOffset, 5)     // Points   ("total = 6396 pixels")
12. boardPQDestRect = boardPSrcRect offset by (0, -kScoreboardTall)
13. boardGQDestRect = boardGSrcRect offset by (0, -kScoreboardTall)
14. text is applFont, size 12, bold
15. badgesBlankRects[0..3] at x = 0:   foil 16x16 @y0, bands 16x16 @y16,
                                       battery 16x17 @y32, helium 16x17 @y49
16. badgesBadgesRects[0..3] identical but at x = 16
17. badgesDestRects: foil    (432 + hOffset, 2 - kScoreboardTall)
                     bands   (449 + hOffset, 2 - kScoreboardTall)
                     battery (467 + hOffset, 1 - kScoreboardTall)
                     helium  (467 + hOffset, 1 - kScoreboardTall)   // SAME x as battery
```

Step 17: battery and helium share the same destination x = 467. This is deliberate —
`batteryTotal` cannot be positive and negative at once, so only one of the two badges is
ever drawn (§15).

The two different `hOffset` computations at steps 3-4 and step 8 use **different formulas**
and the magic numbers **640**, **576**, **128**, **137**, **526**, **570**, **432**, **449**,
**467** are all bare literals (§24).
---

## 15. Inventory: battery, helium, bands, foil, lives

### 15.1 The supply constants

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kBatterySupply` | **50** | `GliderPRO/Sources/Interactions.c:16` | charges granted by a `kBattery`; author's comment: "about 2 rooms worth of thrust" |
| `kHeliumSupply` | **150** | `GliderPRO/Sources/Interactions.c:17` | charges granted by a `kHelium` (stored **negated**) |
| `kBandsSupply` | **8** | `GliderPRO/Sources/Interactions.c:18` | bands granted by a `kBands` |
| `kFoilSupply` | **8** | `GliderPRO/Sources/Interactions.c:19` | foil hits granted by a `kFoil` |
| `kInitialGliders` | **2** | `GliderPRO/Sources/Play.c:19` | starting lives; `mortals = kInitialGliders` (`GliderPRO/Sources/Play.c:341`), `mortals += kInitialGliders` for 2-player (`GliderPRO/Sources/Play.c:343`) |

### 15.2 The four counters

| Global | Type | Semantics |
|---|---|---|
| `batteryTotal` | `short` | **> 0** = battery charges left; **< 0** = helium charges left (magnitude); **0** = empty |
| `bandsTotal` | `short` | rubber bands held |
| `foilTotal` | `short` | foil hits remaining; decremented at 15 Hz while a foil collision persists |
| `mortals` | `short` | lives remaining |

`batteryTotal` is the interesting one. Sign-encoded:

* `DoBatteryEngaged` (battery, `batteryTotal > 0`) does `batteryTotal--` and applies
  `hVel ± kHyperThrust` (8).
* `DoHeliumEngaged` (helium, `batteryTotal < 0`) does `batteryTotal++` and sets
  `vDesiredVel = -kHeliumLift` (−4).
* Both stop at exactly `batteryTotal == 0` and play `kFizzleSound`.
* The dispatch is in `GetInput`: `if (batteryTotal > 0) DoBatteryEngaged(); else DoHeliumEngaged();`
  — guarded by `batteryTotal != 0`, so the two never conflict
  (`GliderPRO/Sources/Input.c:~320`).
* Pickup: battery **replaces** helium (`if (batteryTotal > 0) batteryTotal += 50;
  else batteryTotal = 50;`) and helium **replaces** battery
  (`if (batteryTotal < 0) batteryTotal -= 150; else batteryTotal = -150;`).

A Go port that splits these into two unsigned fields must preserve the replacement
semantics or players will be able to stockpile both.

### 15.3 Foil

`kFoilSupply` = 8. `showFoil` (`Boolean`, `GliderPRO/Headers/GliderVars.h:~`) says whether
the glider is currently drawn with the foil sprite. Foil art:

| Name | Value | Site |
|---|---|---|
| `kGliderFoilPictID` | 3976 | `GliderPRO/Headers/GliderDefines.h:567` |
| `kGliderFoil2PictID` | 3963 | `GliderPRO/Headers/GliderDefines.h:565` |
| `kGliderPictID` | 3999 | `GliderPRO/Headers/GliderDefines.h:568` |
| `kGlider2PictID` | 3974 | `GliderPRO/Headers/GliderDefines.h:566` |

Foil is consumed by `if (evenFrame && foilTotal > 0) foilTotal--;`
(`GliderPRO/Sources/Dynamics.c:60`) — i.e. one hit per two frames of contact, so 8 foil =
16 frames ≈ 0.53 s of sustained contact.

Foil transitions use modes `kGliderGoingFoil` (18) / `kGliderLosingFoil` (19), which reuse
the about-face frames `kFirstAboutFaceFrame` (18) … `kLastAboutFaceFrame` (20). The
numerical coincidence between mode 18 and frame 18 is *not* meaningful.

### 15.4 `HandleMicrowaveAction` — inventory destruction

`GliderPRO/Sources/Interactions.c:1157-1195`:

```
1. kills = who->data.g.byte0;
2. if (kills & 0x0001) bandsTotal   = 0;    // destroy rubber bands
3. if (kills & 0x0002) batteryTotal = 0;    // destroy battery AND helium (sign-encoded)
4. if (kills & 0x0004) foilTotal    = 0;    // destroy foil
```

Bits `0x0008`…`0x0080` of `byte0` are unused. Note bit 1 kills both battery and helium
because they share one variable.

---

## 16. Sound

### 16.1 The 64 sound IDs

`GliderPRO/Headers/GliderDefines.h:55-118`. These are **indices into `theSoundData[]`**, not
resource IDs; the resource ID is `index + kBaseBufferSoundID` = `index + 1000`
(`GliderPRO/Sources/Sound.c:14`).

| Name | Idx | `snd ` ID | Site | Resource name (from `Glider PRO.r`) |
|---|---|---|---|---|
| `kHitWallSound` | 0 | 1000 | `:55` | "Wall Hit" |
| `kFadeInSound` | 1 | 1001 | `:56` | |
| `kFadeOutSound` | 2 | 1002 | `:57` | |
| `kBeepsSound` | 3 | 1003 | `:58` | |
| `kBuzzerSound` | 4 | 1004 | `:59` | |
| `kDingSound` | 5 | 1005 | `:60` | |
| `kEnergizeSound` | 6 | 1006 | `:61` | |
| `kFollowSound` | 7 | 1007 | `:62` | |
| `kMicrowavedSound` | 8 | 1008 | `:63` | |
| `kSwitchSound` | 9 | 1009 | `:64` | |
| `kBirdSound` | 10 | 1010 | `:65` | |
| `kCuckooSound` | 11 | 1011 | `:66` | |
| `kTikSound` | 12 | 1012 | `:67` | |
| `kTokSound` | 13 | 1013 | `:68` | |
| `kBlowerOn` | 14 | 1014 | `:69` | |
| `kBlowerOff` | 15 | 1015 | `:70` | |
| `kCaughtFireSound` | 16 | 1016 | `:71` | |
| `kScoreTikSound` | 17 | 1017 | `:72` | "Score Tick" |
| `kThrustSound` | 18 | 1018 | `:73` | |
| `kFizzleSound` | 19 | 1019 | `:74` | |
| `kFireBandSound` | 20 | 1020 | `:75` | |
| `kBandReboundSound` | 21 | 1021 | `:76` | |
| `kGreaseSpillSound` | 22 | 1022 | `:77` | |
| `kChordSound` | 23 | 1023 | `:78` | |
| `kVCRSound` | 24 | 1024 | `:79` | |
| `kFoilHitSound` | 25 | 1025 | `:80` | |
| `kShredSound` | 26 | 1026 | `:81` | |
| `kToastLaunchSound` | 27 | 1027 | `:82` | |
| `kToastLandSound` | 28 | 1028 | `:83` | |
| `kMacOnSound` | 29 | 1029 | `:84` | |
| `kMacBeepSound` | 30 | 1030 | `:85` | |
| `kMacOffSound` | 31 | 1031 | `:86` | |
| `kTVOnSound` | 32 | 1032 | `:87` | |
| `kTVOffSound` | 33 | 1033 | `:88` | |
| `kCoffeeSound` | 34 | 1034 | `:89` | |
| `kMysticSound` | 35 | 1035 | `:90` | |
| `kZapSound` | 36 | 1036 | `:91` | |
| `kPopSound` | 37 | 1037 | `:92` | |
| `kEnemyInSound` | 38 | 1038 | `:93` | |
| `kEnemyOutSound` | 39 | 1039 | `:94` | |
| `kPaperCrunchSound` | 40 | 1040 | `:95` | |
| `kBounceSound` | 41 | 1041 | `:96` | |
| `kDripSound` | 42 | 1042 | `:97` | |
| `kDropSound` | 43 | 1043 | `:98` | |
| `kFishOutSound` | 44 | 1044 | `:99` | |
| `kFishInSound` | 45 | 1045 | `:100` | |
| `kDontExitSound` | 46 | 1046 | `:101` | |
| `kSizzleSound` | 47 | 1047 | `:102` | "Sizzle" |
| `kPaper1Sound` | 48 | 1048 | `:103` | |
| `kPaper2Sound` | 49 | 1049 | `:104` | |
| `kPaper3Sound` | 50 | 1050 | `:105` | |
| `kPaper4Sound` | 51 | 1051 | `:106` | |
| `kTypingSound` | 52 | 1052 | `:107` | |
| `kCarriageSound` | 53 | 1053 | `:108` | |
| `kChord2Sound` | 54 | 1054 | `:109` | |
| `kPhoneRingSound` | 55 | 1055 | `:110` | |
| `kChime1Sound` | 56 | 1056 | `:111` | |
| `kChime2Sound` | 57 | 1057 | `:112` | |
| `kWebTwangSound` | 58 | 1058 | `:113` | |
| `kTransOutSound` | 59 | 1059 | `:114` | |
| `kTransInSound` | 60 | 1060 | `:115` | |
| `kBonusSound` | 61 | 1061 | `:116` | |
| `kHissSound` | 62 | 1062 | `:117` | "Hiss" |
| `kTriggerSound` | **63** | **1063 — DOES NOT EXIST** | `:118` | (reserved; see §16.4) |

### 16.2 The 60 sound priorities

`GliderPRO/Headers/GliderDefines.h:120-180`. Higher wins. The values are grouped in
hundreds by "family", which is the design: within a family the exact number is arbitrary,
but across families the hundred matters.

| Priority band | Meaning | Members |
|---|---|---|
| 100-103 | lowest — incidental physics | `kHitWallPriority` 100, `kScoreTikPriority` 101, `kBandReboundPriority` 102, `kDontExitPriority` 103 |
| 200-204 | clocks & chimes | `kTikPriority` 200, `kTokPriority` 201, `kMysticPriority` 202, `kChime1Priority` 203, `kChime2Priority` 204 |
| 300-311 | player-initiated physics | `kThrustPriority` 300, `kFireBandPriority` 301, `kChordPriority` 302, `kVCRPriority` 303, `kToastLaunchPriority` 304, `kToastLandPriority` 305, `kCoffeePriority` 306, `kBouncePriority` 307, `kDripPriority` 308, `kDropPriority` 309, `kWebTwangPriority` 310, `kHissPriority` 311 |
| 400-413 | appliances & enemies | `kFoilHitPriority` 400, `kMacOnPriority` 401, `kMacOffPriority` 402, `kMacBeepPriority` 403, `kTVOnPriority` 404, `kTVOffPriority` 405, `kZapPriority` 406, `kPopPriority` 407, `kEnemyInPriority` 408, `kEnemyOutPriority` 409, `kPaperCrunchPriority` 410, `kFishOutPriority` 411, `kFishInPriority` 412, `kSizzlePriority` 413 |
| 500 | phone | `kPhoneRingPriority` 500 |
| 700-703 | switches & blowers | `kSwitchPriority` 700, `kBlowerOnPriority` 701, `kBlowerOffPriority` 702, `kFizzlePriority` 703 |
| 800-812 | UI / prize feedback | `kBeepsPriority` 800, `kBuzzerPriority` 801, `kDingPriority` 802, `kEnergizePriority` 803, `kBirdPriority` 804, `kCuckooPriority` 805, `kGreaseSpillPriority` 806, `kPapersPriority` 807, `kTypingPriority` 808, `kCarriagePriority` 809, `kChord2Priority` 810, `kMicrowavedPriority` 811, `kBonusPriority` 812 |
| 900-906 | glider life events | `kFadeInPriority` 900, `kFadeOutPriority` 901, `kCaughtFirePriority` 902, `kShredPriority` 903, `kFollowPriority` 904, `kTransInPriority` 905, `kTransOutPriority` 906 |
| **999** | the house's custom trigger sound — highest | `kTriggerPriority` 999 |

There is **no 600 band**. `kPapersPriority` (807) is shared by all four
`kPaper1Sound`…`kPaper4Sound`.

Exact sites, in file order: `GliderPRO/Headers/GliderDefines.h:120` (`kHitWallPriority`)
through `:180` (`kTriggerPriority`), one per line with blank lines between families.

### 16.3 The three-channel mixer

`GliderPRO/Sources/Sound.c:38-...` — `PlayPrioritySound(short which, short priority)`:

```
1. if (failedSound || dontLoadSounds) return;
2. if (priority == kTriggerPriority)                      // 999
3.     if (priority0 == kTriggerPriority
4.      || priority1 == kTriggerPriority
5.      || priority2 == kTriggerPriority) return;          // refuse to overlap trigger sounds
6. find the channel among {0, 1, 2} with the LOWEST current priority
7. stop it, load theSoundData[which], set its priorityN = priority, soundPlayingN = which
```

Globals (`GliderPRO/Sources/Sound.c:~20-32`):
```c
SndCallBackUPP  callBack0UPP, callBack1UPP, callBack2UPP;
SndChannelPtr   channel0, channel1, channel2;
Ptr             theSoundData[kMaxSounds];
short           numSoundsLoaded, priority0, priority1, priority2;
short           soundPlaying0, soundPlaying1, soundPlaying2;
Boolean         soundLoaded[kMaxSounds], dontLoadSounds;
Boolean         channelOpen, isSoundOn, failedSound;
```

`kNoSoundPlaying` = −1 (`GliderPRO/Sources/Sound.c:16`) is the idle sentinel.

So: **exactly 3 simultaneous sounds**, chosen by lowest-priority eviction, with the
trigger sound protected against self-overlap. A Go port with unlimited mixing channels
will sound noticeably different (busier); to be faithful, cap at 3 with the same eviction.

### 16.4 Loading — the 20-byte `snd ` header skip, and the reserved slot

`LoadBufferSounds()` — `GliderPRO/Sources/Sound.c:316-347`:

```c
for (i = 0; i < kMaxSounds - 1; i++)             // NOTE: kMaxSounds - 1, i.e. 0..62
{
    theSound = GetResource('snd ', i + kBaseBufferSoundID);   // 1000..1062
    ...
    soundDataSize = GetHandleSize(theSound) - 20L;            // skip the 'snd ' header
    theSoundData[i] = NewPtr(soundDataSize);
    BlockMove((Ptr)(*theSound + 20L), theSoundData[i], soundDataSize);
    ReleaseResource(theSound);
}
theSoundData[kMaxSounds - 1] = nil;              // slot 63 left EMPTY
```

`LoadTriggerSound(short soundID)` — `GliderPRO/Sources/Sound.c:265-303` — fills slot 63
from the **house's own** resource fork:

```c
if (dontLoadSounds) return;
if (theSoundData[kMaxSounds - 1] != nil) return;             // already occupied
theSound = GetResource('snd ', soundID);                     // house's snd , e.g. 3001
soundDataSize = GetHandleSize(theSound) - 20L;
theSoundData[kMaxSounds - 1] = NewPtr(soundDataSize);
BlockMove((Ptr)(*theSound + 20L), theSoundData[kMaxSounds - 1], soundDataSize);
```

`DumpTriggerSound()` (`GliderPRO/Sources/Sound.c:307-312`) frees it;
`DumpBufferSounds()` (`GliderPRO/Sources/Sound.c:351-361`) loops all `kMaxSounds`.

`kMaxSounds` = **64** and `kBaseBufferSoundID` = **1000**, both file-scope defines in
`GliderPRO/Sources/Sound.c:15` and `:14` (not in `GliderDefines.h`); `kTriggerSound` = **63**
is in `GliderPRO/Headers/GliderDefines.h:118`.

**Empirical confirmation.** `GliderPRO/Glider PRO.r` is a **Rez text source**, not a
resource fork — there is no resource map in it to parse. Extracting every
`data 'snd ' (id, "name")` declaration from it:

```
'snd ' declarations present: 1000, 1001, 1002, ..., 1062   (63 resources)
                             2000, 2001, 2002, 2003, 2004, 2005, 2006  (7 music)
missing in 1000..1063: [1063]
```

So `kTriggerSound` (index 63 → nominal ID 1063) has **no shipped resource** — exactly as
the code requires, because slot 63 is always supplied by the house.

**Which houses supply it, and with which IDs.** 13 of the 22 shipped houses carry `'snd '`
resources in their own resource forks, and the IDs are **not** a contiguous run from 3000:

| House | house `'snd '` IDs |
|---|---|
| `Art Museum` | 3000-3004, 3042-3046 |
| `CD Demo House` | 3000-3008, 3042 |
| `California or Bust!` | 3001-3003 |
| `Davis Station` | 3000-3004 |
| `Demo House` | **3011 only** |
| `Grand Prix` | 3000-3002 |
| `ImagineHouse PRO II` | 3001, 3003 |
| `In The Mirror` | 3001 ("Krusty Laugh"), 3002 ("Glass breaking") |
| `Leviathan` | 3000, 3001, 3003-3012 |
| `Nemo's Market` | 3001-3005 |
| `Rainbow's End` | 3000 |
| `SpacePods` | 3000-3002 |
| `Titanic` | 3000, 3006, 3032, 3037, 3058, 3061 |

**`data.e.where` on a `kSoundTrigger` is a raw `'snd '` resource ID, not a packed room
number.** The only caller of `LoadTriggerSound` is
`GliderPRO/Sources/ObjectRects.c:926` — `LoadTriggerSound(theObject.data.e.where)` — and the
editor defaults it to literal **3000** when the object is created
(`GliderPRO/Sources/ObjectAdd.c:498-499`, versus `-1` for every other arm-`e` switch at
`:501`). `kSoundTrigger` is deliberately excluded from both `ObjectIsLinkTransport`
(`GliderPRO/Sources/Objects.c:224-226`) and `ObjectIsLinkSwitch`
(`GliderPRO/Sources/Objects.c:241-244`), so the decimal floor/suite packing of §22.6 does
**not** apply to it. A Go port that runs every `data.e.where` through the room-link decoder
will try to teleport to floor 22 / suite 30 instead of playing a sound.

> [verified: `Titanic` uses 3032/3037/3058/3061, so a port must resolve the ID as-is
> against the house's resource fork and must not assume 3000 + small index.]

**Hot rect ≠ src rect for `kSoundTrigger`.** `srcRects[kSoundTrigger]` is **32 × 32**
(`GliderPRO/Sources/StructuresInit2.c:416`) and that is what the editor draws and hit-tests
(`GliderPRO/Sources/ObjectRects.c:169-175`), but the *play-mode* active rect is hard-coded
**48 × 48** at the same top-left (`GliderPRO/Sources/ObjectRects.c:924` —
`QSetRect(&bounds, 0, 0, 48, 48)`). The trigger zone is therefore 16 px wider and taller
than the object the level author placed. A Go port must keep both rectangles.

**The 20 bytes** are a format-1 `'snd '` resource header: `format` (2), `numSynths` (2),
`synthID` (2), `initOption` (4), `numCmds` (2), and one `bufferCmd` (8) = 20 bytes, after
which the `SoundHeader` begins. Glider PRO strips the command block and keeps only the
sound header + samples, then issues its own `bufferCmd`. A Go port reading these resources
must skip 20 bytes and then parse a classic `SoundHeader` (`samplePtr` 4, `length` 4,
`sampleRate` 4 as Fixed-point 16.16, `loopStart` 4, `loopEnd` 4, `encode` 1, `baseFrequency`
1, then 8-bit unsigned PCM samples).

### 16.5 Phone and chimes timing

`GliderPRO/Sources/Play.c:18-23`:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kRingDelay` | 90 | `GliderPRO/Sources/Play.c:20` | frames between **successive rings** of one phone call (3 s at 30.07 fps) |
| `kRingSpread` | 25000 | `GliderPRO/Sources/Play.c:21` | random component of the inter-call gap |
| `kRingBaseDelay` | 5000 | `GliderPRO/Sources/Play.c:22` | minimum inter-call gap |
| `kChimeDelay` | 180 | `GliderPRO/Sources/Play.c:23` | frames for the chime sequence (6 s) |

Usage:
```
thePhone.nextRing = RandomInt(kRingSpread) + kRingBaseDelay;   // Play.c:735, :759
thePhone.delay    = kRingDelay;                                // Play.c:737, :754
theChimes.nextRing = RandomInt(kChimeDelay) + 1;               // Play.c:739
delayTime = kChimeDelay / numChimes;                           // Play.c:780
```

A call is **3 to 5 rings**, not two: `thePhone.rings = RandomInt(3) + 3`
(`GliderPRO/Sources/Play.c:736`, re-armed at `:760`), each ring `kRingDelay` frames after
the last, and only when the whole ring-group is spent is `nextRing` re-rolled.
`HandleTelephone` is called once per frame from the main loop
(`GliderPRO/Sources/Play.c:445` inside `PlayGame`), so **every one of these counters is in
frames, not ticks**. So the phone rings every 5000…30 000 frames = **166 s … 998 s** (2.8 to
16.6 minutes) — a deliberately rare event.

**`phoneBitSet` in the house flags (§22.7) *disables* the phone**, it does not enable it:
`HandleTelephone`'s entire body is wrapped in `if (!phoneBitSet)`
(`GliderPRO/Sources/Play.c:748`) and the editor checkbox that writes the bit is labelled
"No Phone". The 14 houses with `flags == 0` are the ones that ring.

`StrikeChime()` (`GliderPRO/Sources/Play.c:793-796`) force-fires a chime by zeroing
`theChimes.nextRing`; the chime interval is `RandomInt(kChimeDelay / numChimes) + 1` with the
quotient floored at 2 (`GliderPRO/Sources/Play.c:780-784`), so more wind chimes in a room
means each one rings *more* often, not less.

> [`RandomInt(range)` (`GliderPRO/Sources/Utilities.c:72-82`) is
> `(|Random()| * range) / 32768`. Because `Random()` can return −32768, `|Random()|` can be
> 32768 and the result can be `range` itself, not `range - 1`. A Go port using
> `rand.Intn(range)` will be very slightly off; the difference is 1 draw in 65536 and is
> only observable in RNG-replay tests.]

---

## 17. Music

`GliderPRO/Sources/Music.c:14-17`:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kBaseBufferMusicID` | 2000 | `GliderPRO/Sources/Music.c:15` | first music `snd ` resource ID |
| `kMaxMusic` | 7 | `GliderPRO/Sources/Music.c:16` | number of music buffers (IDs 2000…2006) |
| `kLastMusicPiece` | 16 | `GliderPRO/Sources/Music.c:17` | last index in the full score sequence |
| `kLastGamePiece` | 6 | `GliderPRO/Sources/Music.c:18` | last index in the in-game score sequence |

The 7 music resources, with names read from `GliderPRO/Glider PRO.r`:

| ID | Name |
|---|---|
| 2000 | Refrain1.22 |
| 2001 | Refrain2.22 |
| 2002 | Refrain3.22 |
| 2003 | Refrain4.22 |
| 2004 | Chorus.22 |
| 2005 | RefrainSparse1.22 |
| 2006 | RefrainSparse2.22 |

(The `.22` suffix is the sample rate, 22 kHz.)

Score modes (`GliderPRO/Headers/GliderDefines.h:47-53`):

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kProdGameScoreMode` | **−4** | `:47` | "prod" the score forward (used on horizontal room transitions, `GliderPRO/Sources/Transit.c:157`, `:188`) |
| `kKickGameScoreMode` | **−3** | `:48` | "kick" the score (vertical transitions, `GliderPRO/Sources/Transit.c:~220`) |
| `kPlayGameScoreMode` | **−2** | `:49` | play the in-game sequence (indices 0…`kLastGamePiece`) |
| `kPlayWholeScoreMode` | **−1** | `:50` | play the full sequence (0…`kLastMusicPiece`) |
| `kPlayChorus` | 4 | `:51` | jump to `Chorus.22` (index 4 = ID 2004) |
| `kPlayRefrainSparse1` | 5 | `:52` | index 5 = ID 2005 |
| `kPlayRefrainSparse2` | 6 | `:53` | index 6 = ID 2006 |

Note that the negative values are *modes* while the non-negative values are *direct
indices* into the 7 buffers — a single `short` doing double duty. `kPlayChorus` 4,
`kPlayRefrainSparse1` 5 and `kPlayRefrainSparse2` 6 correspond exactly to buffer indices
4, 5, 6 (IDs 2004-2006).

`isPlayMusicGame` (`GliderPRO/Headers/GliderVars.h:~`) and the prefs booleans
`wasIdleMusic` / `wasGameMusic` (`GliderPRO/Headers/Externs.h:259-260`) gate it.

---

## 18. Resource IDs

Everything in this section was cross-checked against the actual resource fork.

### 18.1 Complete resource-type census of `GliderPRO/Glider PRO.r`

Parsed from the 199 843-line Rez dump:

| Type | Count | ID range | Purpose |
|---|---|---|---|
| `acur` | 1 | 128 | animated-cursor chain (`rAcurID`) |
| `ALRT` | 26 | 130…1046 | alerts (§3.3) |
| `BNDL` | 1 | 128 | Finder bundle |
| `cctb` | 1 | 132 | control colour table |
| `CDEF` | 1 | 128 | custom control definition |
| `cicn` | 44 | 130…2000 | colour icons |
| `clut` | 2 | 128…129 | colour lookup tables (the 8-bit palette!) |
| `CNTL` | 5 | 128…132 | control templates |
| `crsr` | 12 | 149…160 | colour cursors |
| `CURS` | 16 | 128…160 | b/w cursors |
| `dctb` | 20 | 150…1034 | dialog colour tables |
| **`demo`** | **1** | **128** | the recorded demo (§18.8) |
| `DITL` | 54 | 130…1046 | dialog item lists |
| `DLGX` | 1 | 150 | extended dialog (About box) |
| `DLOG` | 28 | 150…1045 | dialogs (§3.3) |
| `FREF` | 6 | 128…133 | Finder file references |
| `icl4` | 6 | 128…133 | 4-bit large icons |
| `icl8` | 6 | 128…133 | 8-bit large icons |
| `ICN#` | 6 | 128…133 | 1-bit large icons |
| `ICON` | 35 | 130…1073 | b/w icons |
| `ics#` | 4 | 128…133 | 1-bit small icons |
| `ics4` | 4 | 128…133 | 4-bit small icons |
| `ics8` | 4 | 128…133 | 8-bit small icons |
| `ictb` | 1 | 150 | item colour table |
| `mctb` | 5 | 128…132 | menu colour tables |
| `MENU` | 6 | 128…141 | menus (§21.1) |
| `ozm5` | 1 | 0 | the app's own signature resource (creator `'ozm5'`) |
| `PAT#` | 1 | 128 | the 7 marquee patterns (`kMarqueePatListID`) |
| **`PICT`** | **152** | 150…10000 | all artwork (§18.2-18.6) |
| **`snd `** | **70** | 1000…2006 | 63 effects + 7 music (§16) |
| `STR#` | 10 | 128…1007 | string lists (§18.7) |
| `vers` | 2 | 1…2 | version resources |
| `wctb` | 1 | 130 | window colour table |
| `WDEF` | 2 | 128…129 | custom window definitions (`kWindoidWDEF` 2048, `kWindoidGrowWDEF` 2064 are *variation codes*, `WDEF` IDs are 2048>>4 = 128 and 2064>>4 = 129) |
| `WIND` | 3 | 128…130 | window templates |

`kWindoidWDEF` = 2048 (`GliderPRO/Headers/GliderDefines.h:531`) and `kWindoidGrowWDEF` =
2064 (`GliderPRO/Headers/GliderDefines.h:532`) are `procID`s, which the Window Manager
splits as `WDEF id = procID >> 4` and `variant = procID & 0x0F`. So 2048 → WDEF 128
variant 0, 2064 → WDEF 129 variant 0. `kFloatingKind` = 2048
(`GliderPRO/Sources/WindowUtils.c:14`) is the `windowKind` field for the same windows.

### 18.2 Sprite atlases and the "+1000 mask" convention

The core convention: for an art PICT of ID *N*, its 1-bit transparency mask is `PICT N+1000`,
loaded into a **depth-1 GWorld**. Verified by both the code and the resource fork.

| Atlas constant | Value | Site | Mask ID | Mask present? |
|---|---|---|---|---|
| `kShadowPictID` | 3998 | `GliderPRO/Sources/StructuresInit.c:19` | 4998 | yes |
| `kGliderPictID` | 3999 | `GliderPRO/Headers/GliderDefines.h:568` | 4999 | yes |
| `kBlowerPictID` | 4000 | `GliderPRO/Sources/StructuresInit.c:20` | 5000 | yes |
| `kFurniturePictID` | 4001 | `GliderPRO/Sources/StructuresInit.c:21` | 5001 | yes |
| `kBonusPictID` | 4002 | `GliderPRO/Sources/StructuresInit.c:22` | 5002 | yes |
| **`kSwitchPictID`** | **4003** | `GliderPRO/Sources/StructuresInit.c:23` | 5003 | **NO — absent, and no mask GWorld is created** (`GliderPRO/Sources/StructuresInit.c:456-458`) |
| `kLightPictID` | 4004 | `GliderPRO/Sources/StructuresInit.c:24` | 5004 | yes |
| `kAppliancePictID` | 4005 | `GliderPRO/Sources/StructuresInit.c:25` | 5005 | yes |
| `kPointsPictID` | 4006 | `GliderPRO/Sources/StructuresInit.c:26` | 5006 | yes |
| `kRubberBandsPictID` | 4007 | `GliderPRO/Sources/StructuresInit.c:27` | 5007 | yes |
| `kTransportPictID` | 4008 | `GliderPRO/Sources/StructuresInit.c:28` | 5008 | yes |
| `kToastPictID` | 4009 | `GliderPRO/Sources/StructuresInit.c:29` | 5009 | yes (`GliderPRO/Sources/StructuresInit.c:552-554`) |
| `kShreddedPictID` | 4010 | `GliderPRO/Sources/StructuresInit.c:30` | 5010 | yes |
| `kBalloonPictID` | 4011 | `GliderPRO/Sources/StructuresInit.c:31` | 5011 | yes |
| `kCopterPictID` | 4012 | `GliderPRO/Sources/StructuresInit.c:32` | 5012 | yes |
| `kDartPictID` | 4013 | `GliderPRO/Sources/StructuresInit.c:33` | 5013 | yes |
| `kBallPictID` | 4014 | `GliderPRO/Sources/StructuresInit.c:34` | 5014 | yes |
| `kDripPictID` | 4015 | `GliderPRO/Sources/StructuresInit.c:35` | 5015 | yes |
| `kEnemyPictID` | 4016 | `GliderPRO/Sources/StructuresInit.c:36` | 5016 | yes |
| `kFishPictID` | 4017 | `GliderPRO/Sources/StructuresInit.c:37` | 5017 | yes |
| `kClutterPictID` | 4018 | `GliderPRO/Sources/StructuresInit2.c:22` | 5018 | yes (`GliderPRO/Sources/StructuresInit2.c:64-70`) |
| **`kSupportPictID`** | **1999** | `GliderPRO/Sources/StructuresInit2.c:21` | 2999 | **no mask created** (`GliderPRO/Sources/StructuresInit2.c:107-109`) |
| **`kAngelPictID`** | **1019** | `GliderPRO/Sources/StructuresInit2.c:20` | **1020 = `kAngelPictID + 1`** | **exception to the +1000 rule** (`GliderPRO/Sources/StructuresInit2.c:132-134`) |
| `kBadgePictID` | 1996 | `GliderPRO/Sources/StructuresInit.c:39` | — | no mask |

Observed mask PICTs in the resource fork: **4998, 4999, 5000, 5001, 5002, 5004, 5005, 5006,
5007, 5008, 5009, 5010, 5011, 5012, 5013, 5014, 5015, 5016, 5017, 5018**, plus **1020**.
**5003 is absent** — precisely matching the code, which does not load a switch mask.

The two exceptions are the load-bearing part of this section. A Go asset pipeline that
mechanically derives `mask = art + 1000` will fail on the angel (needs +1) and will look
for a nonexistent switch mask.

### 18.3 The 3900-series individual object PICTs

`GliderPRO/Sources/ObjectDraw2.c:67-103` — 41 constants for objects drawn from their own
PICT rather than from an atlas:

| Name | ID | Name | ID |
|---|---|---|---|
| `kCobwebPictID` | 3958 | `kOzmaPictID` | 3975 |
| `kFlowerBoxPictID` | 3959 | `kGliderFoilPictID` | 3976 |
| `kCinderPictID` | 3960 | `kWindowExRightPictID` | 3977 |
| `kChimesPictID` | 3961 | `kWindowExLeftPictID` | 3978 |
| `kRugPictID` | 3962 | `kWindowInRightPictID` | 3979 |
| `kGliderFoil2PictID` | 3963 | `kWindowInLeftPictID` | 3980 |
| `kBooksPictID` | 3964 | `kDoorExLeftPictID` | 3981 |
| `kCloudPictID` | 3965 | `kDoorExRightPictID` | 3982 |
| `kBulletinPictID` | 3966 | `kDoorInRightPictID` | 3983 |
| `kManholePictID` | 3967 | `kDoorInLeftPictID` | 3984 |
| `kVase2PictID` | 3968 | `kMailboxRightPictID` | 3985 |
| `kVase1PictID` | 3969 | `kMailboxLeftPictID` | 3986 |
| `kCalendarPictID` | 3970 | `kTrunkPictID` | 3987 |
| `kMicrowavePictID` | 3971 | `kBBQPictID` | 3988 |
| `kBearPictID` | 3972 | `kStereoPictID` | 3989 |
| `kFireplacePictID` | 3973 | `kVCRPictID` | 3990 |
| `kGlider2PictID` | 3974 | `kGuitarPictID` | 3991 |
| | | `kTVPictID` | 3992 |
| | | `kDecoLampPictID` | 3993 |
| | | `kHipLampPictID` | 3994 |
| | | `kFilingCabinetPictID` | 3995 |
| | | `kDownStairsPictID` | 3996 |
| | | `kUpStairsPictID` | 3997 |

Plus `kManholeThruFloor` = 3957 (`GliderPRO/Sources/RoomGraphics.c:16`) — the manhole
drawn *through* the floor plane, and `kStarPictID` = 1995
(`GliderPRO/Headers/GliderDefines.h:533`).

### 18.4 The 28 `k*MaskID` constants — 20 of which are dead

`GliderPRO/Sources/ObjectDraw2.c:39-66` declares 28 mask IDs in the range 3900-3927.
Cross-referencing against actual `GetPicture` calls in the sources, **only 8 are ever
used**:

| Used mask constant | ID | Object |
|---|---|---|
| `kMailboxRightMaskID` | 3903 | `kMailboxRt` |
| `kMailboxLeftMaskID` | 3904 | `kMailboxLf` |
| `kTVMaskID` | 3912 | `kTV` |
| `kVCRMaskID` | 3913 | `kVCR` |
| `kStereoMaskID` | 3914 | `kStereo` |
| `kMicrowaveMaskID` | 3915 | `kMicrowave` |
| `kCloudMaskID` | 3921 | `kCloud` |
| `kCobwebMaskID` | 3927 | `kCobweb` |

And **exactly those 8 IDs exist as PICT resources** in `GliderPRO/Glider PRO.r`. The other
20 constants (3900-3902, 3905-3911, 3916-3920, 3922-3926) name resources that were never
shipped: they are leftovers from a design where every 3900-series object had a mask, and
the final art instead relies on the object's own opaque rectangle. A Go asset pipeline must
not try to load them.

### 18.5 Backgrounds and user ranges

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kBaseBackgroundID` | 2000 | `GliderPRO/Headers/GliderDefines.h:519` | first built-in background PICT |
| `kNumBackgrounds` | 18 | `GliderPRO/Headers/GliderDefines.h:521` | count (2000…2017) |
| `kFirstOutdoorBack` | 2009 | `GliderPRO/Headers/GliderDefines.h:520` | first outdoor background (`kGarden`) |
| `kUserBackground` | 3000 | `GliderPRO/Headers/GliderDefines.h:522` | house-supplied backgrounds start here |
| `kUserStructureRange` | 3300 | `GliderPRO/Headers/GliderDefines.h:523` | house-supplied *room structures* start here |
| `kSplash8BitPICT` | 1000 | `GliderPRO/Headers/GliderDefines.h:524` | title screen, 8-bit |

The 18 built-in backgrounds, `GliderPRO/Headers/GliderDefines.h:227-244`:

| Name | PICT ID | Site | Indoor/outdoor |
|---|---|---|---|
| `kSimpleRoom` | 2000 | `:227` | indoor |
| `kPaneledRoom` | 2001 | `:228` | indoor |
| `kBasement` | 2002 | `:229` | indoor |
| `kChildsRoom` | 2003 | `:230` | indoor |
| `kAsianRoom` | 2004 | `:231` | indoor |
| `kUnfinishedRoom` | 2005 | `:232` | indoor |
| `kSwingersRoom` | 2006 | `:233` | indoor |
| `kBathroom` | 2007 | `:234` | indoor |
| `kLibrary` | 2008 | `:235` | indoor |
| `kGarden` | 2009 | `:236` | **outdoor** (`kFirstOutdoorBack`) |
| `kSkywalk` | 2010 | `:237` | outdoor |
| `kDirt` | 2011 | `:238` | outdoor |
| `kMeadow` | 2012 | `:239` | outdoor |
| `kField` | 2013 | `:240` | outdoor |
| `kRoof` | 2014 | `:241` | outdoor |
| `kSky` | 2015 | `:242` | outdoor |
| `kStratosphere` | 2016 | `:243` | outdoor |
| `kStars` | 2017 | `:244` | outdoor |

**Verified in a real house.** `GliderPRO/Houses/In The Mirror.binhex` resource fork
(151 870 bytes) contains:

```
PICT 1991, 1992, 1993                     (custom banner art, overriding the app's)
PICT 3300 "Tall Room", 3301 "Illusion"    (kUserStructureRange custom room structures)
PICT 10002, 10006, 10008, 10018, 10025,
     10026, 10034, 10036, 10038, 10074    (named custom clutter/appliance art)
snd  3001 "Krusty Laugh", 3002 "Glass breaking"
vers 1, vers 2
ICN#/icl8/icl4/ics#/ics8/ics4 icon family + BNDL icon 128
```

By contrast `GliderPRO/Houses/Empty House.binhex` (resource fork 2670 bytes) has **only**
an icon family at ID −16455 — no custom art at all.

`kCustomPict` (`0x6E`) objects reference a PICT by ID stored in `data.g.height`, which is
why the observed IDs are in the 10000s (well clear of the app's own 1000-5000 range).

### 18.6 Named UI PICTs

| Name | ID | Site | Purpose |
|---|---|---|---|
| `kSplash8BitPICT` | 1000 | `GliderPRO/Headers/GliderDefines.h:524` | title screen |
| `kLoadTitlePict1` | 1001 | `GliderPRO/Sources/SelectHouse.c:29` | house-picker title, 1-bit |
| `kLoadTitlePict8` | 1002 | `GliderPRO/Sources/SelectHouse.c:30` | house-picker title, 8-bit |
| `kDefaultHousePict1` | 1003 | `GliderPRO/Sources/SelectHouse.c:31` | default house icon, 1-bit |
| `kDefaultHousePict8` | 1004 | `GliderPRO/Sources/SelectHouse.c:32` | default house icon, 8-bit |
| `kThumbnailPictID` | 1010 | `GliderPRO/Sources/Map.c:25` | map-window room thumbnails |
| `kToolsPictID` | 1011 | `GliderPRO/Sources/Tools.c:46` | tool-palette icons |
| `kEscPausePictID` | 1015 | `GliderPRO/Sources/Input.c:17` | "paused (Esc)" overlay |
| `kTabPausePictID` | 1016 | `GliderPRO/Sources/Input.c:18` | "paused (Tab)" overlay |
| `kStarsRemainingPICT` | 1017 | `GliderPRO/Sources/Banner.c:~` | banner: "N stars remaining" |
| `kStarRemainingPICT` | 1018 | `GliderPRO/Sources/Banner.c:~` | banner: "1 star remaining" (singular) |
| `kAngelPictID` | 1019 | `GliderPRO/Sources/StructuresInit2.c:20` | the angel that flies off on death |
| `kMilkywayPictID` | 1021 | `GliderPRO/Sources/GameOver.c:~` | game-over backdrop |
| `kLettersPictID` | 1988 | `GliderPRO/Sources/GameOver.c:~` | game-over letters |
| `kPagesMaskID` | 1989 | `GliderPRO/Sources/GameOver.c:~` | game-over pages mask |
| `kPagesPictID` | 1990 | `GliderPRO/Sources/GameOver.c:~` | game-over pages |
| `kBannerPageBottomMask` | 1991 | `GliderPRO/Sources/Banner.c:~` | banner page bottom mask |
| `kBannerPageBottomPICT` | 1992 | `GliderPRO/Sources/Banner.c:~` | banner page bottom |
| `kBannerPageTopPICT` | 1993 | `GliderPRO/Sources/Banner.c:~` | banner page top |
| `kHighScoresPictID` | 1994 | `GliderPRO/Sources/HighScores.c:~` | high-score backdrop |
| `kStarPictID` | 1995 | `GliderPRO/Headers/GliderDefines.h:534` | the star sprite |
| `kBadgePictID` | 1996 | `GliderPRO/Sources/StructuresInit.c:39` | scoreboard badges (32 × 66) |
| `kScoreboardPictID` | 1997 | `GliderPRO/Headers/GliderDefines.h:623` | scoreboard strip |
| `kHighScoresMaskID` | 1998 | `GliderPRO/Sources/HighScores.c:~` | high-score mask |
| `kSupportPictID` | 1999 | `GliderPRO/Sources/StructuresInit2.c:21` | floor-support strip (`kFloorSupportTall` 44) |
| `kGrayedOutUpArrow` | 1052 | `GliderPRO/Sources/SelectHouse.c:~` | disabled scroll arrow |
| `kGrayedOutDownArrow` | 1053 | `GliderPRO/Sources/SelectHouse.c:~` | disabled scroll arrow |

Note the deliberate ordering 1988-1999: art that the *house* may override (banner
1991-1993) sits immediately below the art the app always owns (1994-1999). A house that
supplies PICT 1991/1992/1993 replaces the banner page art, as `In The Mirror` does.

### 18.7 `STR#` string lists

All 10 in `GliderPRO/Glider PRO.r`, with the names read from the resource map:

| ID | Resource name | Constant | Site |
|---|---|---|---|
| 128 | "Jinjur" | (used by `Validate.c`) | — |
| 129 | "Prefmain" | (prefs strings) | — |
| 140 | "File Error" | `rFileErrorStrings` | `GliderPRO/Sources/FileError.c:15` |
| 150 | "Localized Strings" | `kLocalizedStringsID` | `GliderPRO/Sources/StringUtils.c:323` |
| 160 | "Prefs" | `kPrefsStringsID` | `GliderPRO/Sources/Prefs.c:22` |
| 170 | "Errors" | `rErrTitleID` | `GliderPRO/Sources/Utilities.c:138` |
| 171 | "Errors" | `rErrMssgID` | `GliderPRO/Sources/Utilities.c:139` |
| 1005 | "Months" | `kMonthStringID` | `GliderPRO/Sources/ObjectDraw2.c:106` |
| 1006 | "Yellow Alerts" | (indexed by `kYellow*`) | §3.2 |
| 1007 | "Object Names" | `kObjectNameStrings` | `GliderPRO/Headers/GliderDefines.h:461` |

`kChooserStringID` = **−16096** (`GliderPRO/Sources/StringUtils.c:304`) is a *System*
`STR ` resource (the Chooser's owner name), not one of the app's own — negative IDs are the
system range.

`kPrefsFNameIndex` = 1 (`GliderPRO/Sources/Prefs.c:~`) indexes `STR# 160`.

### 18.8 The `demo` resource — proving `sizeof(demoType) == 6`

| Name | Value | Site |
|---|---|---|
| `kDemoLength` | **6702** | `GliderPRO/Headers/GliderDefines.h:625` |

`CreatePointers()` (`GliderPRO/Sources/StructuresInit2.c:280-297`):

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

**Empirical check.** The `demo` 128 resource in `GliderPRO/Glider PRO.r` is exactly
**6702 bytes**. `demoType` is:

```c
typedef struct                       // GliderPRO/Headers/GliderStructs.h:334-339
{
    long    frame;                   // 4
    char    key;                     // 1
    char    padding;                 // 1
} demoType;
```

6702 = 6 × 1117, and 6 is `sizeof(demoType)` **only under 68k alignment**
(`long` at offset 0, `char` at 4, `char` at 5, struct alignment 2 → size 6). Under
PowerPC/default alignment the struct would be padded to **8** bytes and `kDemoLength`
would have to be 8936. So:

* The demo contains exactly **1117 key events**.
* The `#pragma options align=mac68k` discipline (§22.7) is load-bearing for this resource.
* A Go port must parse the resource as 1117 records of `{int32 frame; int8 key; int8 pad}`,
  big-endian.

> [verified by re-parsing `data 'demo' (128)` out of the Rez source: 6702 bytes, 1117
> six-byte records, frames strictly non-decreasing from 46 to 3414.]

**`key` is an index into a four-way switch, not a keycode**, and `padding` really is
garbage. `RecordKey()` (`GliderPRO/Sources/Input.c:46-48`) writes only `.frame` and `.key`;
the byte at record offset 5 is never assigned, so in the shipped resource it holds heap
litter (407 of the 1117 records happen to have 0 there, the rest hold 105 distinct values
including 255). The playback switch is `GliderPRO/Sources/Input.c:226-264`:

| `key` | Meaning | Effect at `GliderPRO/Sources/Input.c` |
|---|---|---|
| 0 | left key | `:228-233` — `hDesiredVel += kNormalThrust`, `heldRight = true`, `tipped = (facing == kFaceLeft)` |
| 1 | right key | `:235-240` — `hDesiredVel -= kNormalThrust`, `heldLeft = true`, `tipped = (facing == kFaceRight)` |
| 2 | battery / helium | `:242-248` — `DoBatteryEngaged` if `batteryTotal > 0`, else `DoHeliumEngaged` |
| 3 | rubber band | `:250-263` — `AddBand(…, dest.left + 24, dest.top + 10, facing)` |

Only values **0, 1 and 3** occur in the shipped demo (910 / 198 / 9 records); `key == 2` is
never exercised, so the battery branch is untested by the demo. Note the deliberately
mirrored sign convention (`case 0` "left" *adds* thrust and sets `heldRight`) — see §10.3.

Demo playback also only advances `demoIndex` on an **exact** `gameFrame ==
demoData[demoIndex].frame` match (`GliderPRO/Sources/Input.c:224`), with no catch-up loop:
if a frame is ever skipped the whole remaining demo desyncs and stalls. A Go port must
drive the demo off the same integer frame counter, not off wall-clock time.

`kDemoLength` is also added to the memory requirement:
`bytesNeeded += kDemoLength;` (`GliderPRO/Sources/Environ.c:657`).

Other demo/idle constants:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kIdleSplashMode` | 0 | `GliderPRO/Headers/GliderDefines.h:195` | title screen showing the splash |
| `kIdleDemoMode` | 1 | `GliderPRO/Headers/GliderDefines.h:196` | title screen playing the demo |
| `kIdleSplashTicks` | 7200L | `GliderPRO/Headers/GliderDefines.h:197` | ticks before switching (comment: "2 minutes"; 7200/60.15 = 119.7 s) |
| `kIdleLastMode` | 1 | `GliderPRO/Headers/GliderDefines.h:198` | highest idle mode |

`incrementModeTime = TickCount() + kIdleSplashTicks;`
(`GliderPRO/Sources/InterfaceInit.c:163`).
---

## 19. The 8-bit indexed palette

Glider PRO is an **8-bit indexed-colour** game. `kPreferredDepth` = 8
(`GliderPRO/Headers/Externs.h:15`); everything is drawn either by `CopyBits`/`CopyMask`
of pre-rendered PICTs or by `PaintRect`/`FrameRect`/`LineTo` with a foreground colour set via
`Index2Color(index, &rgb); RGBForeColor(&rgb);`. The named indices below are therefore
*palette indices*, not RGB values, and every one of them must be resolved through the same
palette a real Mac would have used.

### 19.1 Empirical verification: `clut` 128 and `clut` 129 are both the standard Mac 8-bit palette

`GliderPRO/Glider PRO.r` contains exactly two `clut` resources, IDs **128** and **129**.
Parsed with python3:

```
clut 128: 2056 bytes.  header  seed=0x00000000  flags=0x0000  size=255  => 256 entries
clut 129: 2056 bytes.  header  seed=0x00000000  flags=0x0000  size=255  => 256 entries
clut 128 == clut 129 byte-for-byte:  True
```

(`ColorTable` layout: `long ctSeed; short ctFlags; short ctSize; ColorSpec ctTable[ctSize+1]`,
each `ColorSpec` = `short value; RGBColor rgb` = 2 + 6 = 8 bytes. 8 + 256 × 8 = 2056, which matches the observed resource length.)

The 256 entries are the **standard Macintosh 8-bit system palette**, which decomposes as:

| Index range | Content | Closed form (8-bit RGB) |
|---|---|---|
| **0 … 214** | the 6 × 6 × 6 colour cube, minus its last entry | `r = 255 − 51·(i / 36)`, `g = 255 − 51·((i / 6) mod 6)`, `b = 255 − 51·(i mod 6)` |
| **215 … 224** | red ramp | `(238, 0, 0)`, `(221,0,0)`, `(187,0,0)`, `(170,0,0)`, `(136,0,0)`, `(119,0,0)`, `(85,0,0)`, `(68,0,0)`, `(34,0,0)`, `(17,0,0)` |
| **225 … 234** | green ramp | `(0,238,0)` … `(0,17,0)`, same 10 levels |
| **235 … 244** | blue ramp | `(0,0,238)` … `(0,0,17)`, same 10 levels |
| **245 … 254** | grey ramp | `(238,238,238)`, `(221,…)`, `(187,…)`, `(170,…)`, `(136,…)`, `(119,…)`, `(85,…)`, `(68,…)`, `(34,…)`, `(17,…)` |
| **255** | black | `(0, 0, 0)` |

The colour-cube formula was checked against all 216 cube slots; there is **exactly one
mismatch**, index **215**, where the formula predicts `(0,0,0)` but the resource holds
`(238,0,0)`. That is the well-known truncation of the Mac palette: the cube occupies
0…214 (215 entries) and its would-be black slot is reclaimed as the first entry of the red
ramp, because black already exists at 255.

The ten ramp levels are `238, 221, 187, 170, 136, 119, 85, 68, 34, 17` — i.e.
`0xEE, 0xDD, 0xBB, 0xAA, 0x88, 0x77, 0x55, 0x44, 0x22, 0x11`, the ten "double-nybble"
values that are *not* multiples of 51 (which the cube already covers: `0xFF, 0xCC, 0x99,
0x66, 0x33, 0x00`).

**A Go port must hard-code this exact 256-entry table** if it wants to reproduce
`Index2Color()` results for the programmatically-drawn parts of the screen (scoreboard
background, room-edit boxes, marquee, star field). Every PICT in the game is already
indexed against it, so decoding the PICTs also requires it.

### 19.2 The named indices in `ObjectDraw.c`

`GliderPRO/Sources/ObjectDraw.c:16-48`, 33 constants. RGB values are those observed in
`clut` 128 above.

| Name | Index | Hex | Observed RGB | Site |
|---|---|---|---|---|
| `k8WhiteColor` | 0 | 0x00 | (255, 255, 255) | `GliderPRO/Sources/ObjectDraw.c:16` |
| `kYellowColor` | 5 | 0x05 | (255, 255, 0) | `GliderPRO/Sources/ObjectDraw.c:17` |
| `kGoldColor` | 11 | 0x0B | (255, 204, 0) | `GliderPRO/Sources/ObjectDraw.c:18` |
| `k8RedColor` | 35 | 0x23 | (255, 0, 0) | `GliderPRO/Sources/ObjectDraw.c:19` |
| `kPaleVioletColor` | 42 | 0x2A | (204, 204, 255) | `GliderPRO/Sources/ObjectDraw.c:20` |
| `k8LtTanColor` | 52 | 0x34 | (204, 153, 51) | `GliderPRO/Sources/ObjectDraw.c:21` |
| `k8BambooColor` | 53 | 0x35 | (204, 153, 0) | `GliderPRO/Sources/ObjectDraw.c:22` |
| `kDarkFleshColor` | 58 | 0x3A | (204, 102, 51) | `GliderPRO/Sources/ObjectDraw.c:23` |
| `k8TanColor` | 94 | 0x5E | (153, 102, 51) | `GliderPRO/Sources/ObjectDraw.c:24` |
| `k8PissYellowColor` | 95 | 0x5F | (153, 102, 0) | `GliderPRO/Sources/ObjectDraw.c:25` |
| `k8OrangeColor` | 59 | 0x3B | (204, 102, 0) | `GliderPRO/Sources/ObjectDraw.c:26` |
| `k8BrownColor` | 137 | 0x89 | (102, 51, 0) | `GliderPRO/Sources/ObjectDraw.c:27` |
| `k8Red4Color` | 143 | 0x8F | (102, 0, 0) | `GliderPRO/Sources/ObjectDraw.c:28` |
| `k8SkyColor` | 150 | 0x96 | (51, 204, 255) | `GliderPRO/Sources/ObjectDraw.c:29` |
| `k8EarthBlueColor` | 170 | 0xAA | (51, 51, 153) | `GliderPRO/Sources/ObjectDraw.c:30` |
| `k8DkRedColor` | 222 | 0xDE | (68, 0, 0) | `GliderPRO/Sources/ObjectDraw.c:31` |
| `k8DkRed2Color` | 223 | 0xDF | (34, 0, 0) | `GliderPRO/Sources/ObjectDraw.c:32` |
| `kIntenseGreenColor` | 225 | 0xE1 | (0, 238, 0) | `GliderPRO/Sources/ObjectDraw.c:33` |
| `kIntenseBlueColor` | 235 | 0xEB | (0, 0, 238) | `GliderPRO/Sources/ObjectDraw.c:34` |
| `k8PumpkinColor` | 101 | 0x65 | (153, 51, 0) | `GliderPRO/Sources/ObjectDraw.c:35` |
| `k8LtstGrayColor` | 245 | 0xF5 | (238, 238, 238) | `GliderPRO/Sources/ObjectDraw.c:36` |
| `k8LtstGray2Color` | 246 | 0xF6 | (221, 221, 221) | `GliderPRO/Sources/ObjectDraw.c:37` |
| `k8LtstGray3Color` | **43** | 0x2B | (204, 204, 204) | `GliderPRO/Sources/ObjectDraw.c:38` |
| `k8LtstGray4Color` | 247 | 0xF7 | (187, 187, 187) | `GliderPRO/Sources/ObjectDraw.c:39` |
| `k8LtstGray5Color` | 248 | 0xF8 | (170, 170, 170) | `GliderPRO/Sources/ObjectDraw.c:40` |
| `k8LtGrayColor` | 249 | 0xF9 | (136, 136, 136) | `GliderPRO/Sources/ObjectDraw.c:41` |
| `k8GrayColor` | 250 | 0xFA | (119, 119, 119) | `GliderPRO/Sources/ObjectDraw.c:42` |
| `k8Gray2Color` | 251 | 0xFB | (85, 85, 85) | `GliderPRO/Sources/ObjectDraw.c:43` |
| `k8DkGrayColor` | 252 | 0xFC | (68, 68, 68) | `GliderPRO/Sources/ObjectDraw.c:44` |
| `k8DkGray2Color` | 253 | 0xFD | (34, 34, 34) | `GliderPRO/Sources/ObjectDraw.c:45` |
| `k8DkGray3Color` | **172** | 0xAC | (51, 51, 51) | `GliderPRO/Sources/ObjectDraw.c:46` |
| `k8DkstGrayColor` | 254 | 0xFE | (17, 17, 17) | `GliderPRO/Sources/ObjectDraw.c:47` |
| `k8BlackColor` | 255 | 0xFF | (0, 0, 0) | `GliderPRO/Sources/ObjectDraw.c:48` |

Two of the "gray" names come from the **colour cube**, not the grey ramp:
`k8LtstGray3Color` = 43 = `(204,204,204)` and `k8DkGray3Color` = 172 = `(51,51,51)`.
The full grey ladder the author uses, ordered light → dark, is therefore

```
0 (255) → 245 (238) → 246 (221) → 43 (204) → 247 (187) → 248 (170) → 249 (136)
       → 250 (119) → 251 (85) → 252 (68) → 172 (51) → 253 (34) → 254 (17) → 255 (0)
```

which is 14 evenly-spaced greys — deliberately assembled so drop-shadows and bevels step
smoothly. The naming (`LtstGray`, `LtstGray2`, `LtstGray3`, …) is *not* in index order and
must not be assumed to be.

### 19.3 The `ObjectDraw2.c` subset

`GliderPRO/Sources/ObjectDraw2.c:19-37` redeclares 19 of the same constants (a duplicate,
identical-value re-`#define` — see §1.3), with one renamed:

| Name | Index | Site | Note |
|---|---|---|---|
| `k8WhiteColor` | 0 | `GliderPRO/Sources/ObjectDraw2.c:19` | same as `ObjectDraw.c:16` |
| **`kIntenseYellowColor`** | **5** | `GliderPRO/Sources/ObjectDraw2.c:20` | **different name, same index 5** as `kYellowColor` |
| `kPaleVioletColor` | 42 | `GliderPRO/Sources/ObjectDraw2.c:21` | |
| `kDarkFleshColor` | 58 | `GliderPRO/Sources/ObjectDraw2.c:22` | |
| `k8TanColor` | 94 | `GliderPRO/Sources/ObjectDraw2.c:23` | |
| `k8PissYellowColor` | 95 | `GliderPRO/Sources/ObjectDraw2.c:24` | |
| `k8BrownColor` | 137 | `GliderPRO/Sources/ObjectDraw2.c:25` | |
| `k8SkyColor` | 150 | `GliderPRO/Sources/ObjectDraw2.c:26` | |
| `k8EarthBlueColor` | 170 | `GliderPRO/Sources/ObjectDraw2.c:27` | |
| `k8DkRed2Color` | 223 | `GliderPRO/Sources/ObjectDraw2.c:28` | |
| `kIntenseGreenColor` | 225 | `GliderPRO/Sources/ObjectDraw2.c:29` | |
| `kIntenseBlueColor` | 235 | `GliderPRO/Sources/ObjectDraw2.c:30` | |
| `k8LtstGrayColor` | 245 | `GliderPRO/Sources/ObjectDraw2.c:31` | |
| `k8LtstGray4Color` | 247 | `GliderPRO/Sources/ObjectDraw2.c:32` | |
| `k8LtstGray5Color` | 248 | `GliderPRO/Sources/ObjectDraw2.c:33` | |
| `k8LtGrayColor` | 249 | `GliderPRO/Sources/ObjectDraw2.c:34` | |
| `k8Gray2Color` | 251 | `GliderPRO/Sources/ObjectDraw2.c:35` | |
| `k8DkGrayColor` | 252 | `GliderPRO/Sources/ObjectDraw2.c:36` | |
| `k8DkGray2Color` | 253 | `GliderPRO/Sources/ObjectDraw2.c:37` | |

Because both files are separate translation units there is no redefinition warning; the
values agree, so behaviour is identical.

### 19.4 Other palette indices scattered through the sources

| Name | Index | Site | Meaning |
|---|---|---|---|
| `kRedOrangeColor8` | **23** | `GliderPRO/Headers/GliderDefines.h:542` | source comment says **"actually, 18"** — the author noted the mismatch and left index 23 = `(255,102,0)` in place. Index 18 would be `(255,102,255)`, a pink. Trust the code (23), not the comment. |
| `kGrayBackgroundColor` | 251 | `GliderPRO/Sources/Scoreboard.c:15` | `(85,85,85)`, the scoreboard grey at depth 8 |
| `kGrayBackgroundColor4` | 10 | `GliderPRO/Sources/Scoreboard.c:16` | the **4-bit** equivalent, used only when `thisMac.isDepth == 4` (`GliderPRO/Sources/Scoreboard.c:144`, `:202`, `:240`, `:277`, `:312`). Index 10 refers to the *system-supplied 4-bit clut*, which is not present in this source tree, so its RGB cannot be verified from these files. |
| `kFadeAmount` | ? | — | not present; fading is done by `fadeInSequence[]` frame indices (§9.2), not by palette animation |

`thisMac.isDepth` (`GliderPRO/Headers/Environ.h`) is the *current* screen depth; the game
runs at 8 but the scoreboard has a 4-bit fallback path. `k8*` prefixes throughout the
codebase mean "8-bit colour index".

### 19.5 Mac-Toolbox notes for the palette

* `Index2Color(short index, RGBColor *rgb)` resolves an index against the **current
  GDevice's** colour table, not a global one. Because Glider PRO forces its own palette on
  the main device at startup, index → RGB is stable. A Go port should just use the table
  above directly.
* The 1-bit **mask** GWorlds use QuickDraw's convention: in a 1-bit pixmap, **0 = white =
  "copy source"** and **1 = black = "leave destination"** under `CopyMask`. Note this is the
  opposite polarity from the intuitive "1 = opaque" of most modern alpha masks. Verify per
  asset when converting; §18.2 lists which atlases have masks.
* `srcCopy` is the only transfer mode used for opaque blits; `CopyMask` for masked ones;
  `PaintRect` with a `RGBForeColor` for flat fills; `transparent`/`blend` modes are
  **not** used anywhere in the game.
* The `PAT#` 128 resource (`kMarqueePatListID`, `GliderPRO/Sources/Marquee.c:15`) holds the
  animated marching-ants patterns for the editor's selection marquee; each `Pattern` is
  8 bytes = an 8 × 8 1-bit tile.

---

## 20. Keyboard

Glider PRO reads the keyboard three different ways, and they use three different numbering
systems. Confusing them is one of the easiest ways to break a port.

| Mechanism | Constants | Numbering |
|---|---|---|
| `GetKeys(&theKeys)` + `BitTst(&theKeys, n)` | `k*KeyMap`, `k*KeypadMap`, `kSpaceBarMap` | **BitTst bit offset** into a 128-bit `KeyMap` |
| `EventRecord.message & charCodeMask` | `k*ASCII` | **ASCII / MacRoman character code** |
| `(EventRecord.message & keyCodeMask) >> 8` | `k*RawKey` | **virtual key code** |

### 20.1 The `KeyMap` bit offsets and their exact relation to virtual key codes

`GetKeys` fills a `KeyMap` = `long[4]` = 16 bytes = 128 bits. `BitTst(ptr, offset)` on the
Mac numbers bits **MSB-first within each byte**:

```
byte           = offset >> 3
bit-in-byte    = 7 - (offset & 7)          // 0 = least significant
tested value   = ((unsigned char *)ptr)[offset >> 3] & (0x80 >> (offset & 7))
```

Empirically, across **all 47** `k*Map` constants in `GliderPRO/Headers/Externs.h`, the
relation to the Mac virtual key code is exactly

> **`keyMapOffset = virtualKeyCode XOR 7`**

which is the same statement as the byte/bit decomposition above (it flips the low 3 bits).
This was verified programmatically for every constant; the only in-tree cross-check
available is `kTabKeyMap` = 55 vs `kTabRawKey` = 0x30 = 48, and 55 XOR 7 = 48 — consistent.

Full table (47 rows), sorted by offset. `byte`/`bit` are the decomposition above; `vkey` is
the derived Mac virtual key code.

| Name | Offset | byte | bit | vkey | Site |
|---|---|---|---|---|---|
| `kXKeyMap` | 0 | 0 | 7 | 0x07 | `GliderPRO/Headers/Externs.h:152` |
| `kZKeyMap` | 1 | 0 | 6 | 0x06 | `GliderPRO/Headers/Externs.h:153` |
| `kGKeyMap` | 2 | 0 | 5 | 0x05 | `GliderPRO/Headers/Externs.h:140` |
| `kHKeyMap` | 3 | 0 | 4 | 0x04 | `GliderPRO/Headers/Externs.h:141` |
| `kFKeyMap` | 4 | 0 | 3 | 0x03 | `GliderPRO/Headers/Externs.h:139` |
| `kDKeyMap` | 5 | 0 | 2 | 0x02 | `GliderPRO/Headers/Externs.h:137` |
| `kSKeyMap` | 6 | 0 | 1 | 0x01 | `GliderPRO/Headers/Externs.h:148` |
| `kAKeyMap` | 7 | 0 | 0 | 0x00 | `GliderPRO/Headers/Externs.h:134` |
| `kRKeyMap` | 8 | 1 | 7 | 0x0F | `GliderPRO/Headers/Externs.h:147` |
| `kEKeyMap` | 9 | 1 | 6 | 0x0E | `GliderPRO/Headers/Externs.h:138` |
| `kWKeyMap` | 10 | 1 | 5 | 0x0D | `GliderPRO/Headers/Externs.h:151` |
| `kQKeyMap` | 11 | 1 | 4 | 0x0C | `GliderPRO/Headers/Externs.h:146` |
| `kBKeyMap` | 12 | 1 | 3 | 0x0B | `GliderPRO/Headers/Externs.h:135` |
| `kVKeyMap` | 14 | 1 | 1 | 0x09 | `GliderPRO/Headers/Externs.h:150` |
| `kCKeyMap` | 15 | 1 | 0 | 0x08 | `GliderPRO/Headers/Externs.h:136` |
| `kTKeyMap` | 22 | 2 | 1 | 0x11 | `GliderPRO/Headers/Externs.h:149` |
| `kOKeyMap` | 24 | 3 | 7 | 0x1F | `GliderPRO/Headers/Externs.h:144` |
| `kPKeyMap` | 36 | 4 | 3 | 0x23 | `GliderPRO/Headers/Externs.h:145` |
| `kPeriodKeyMap` | 40 | 5 | 7 | 0x2F | `GliderPRO/Headers/Externs.h:154` |
| `kMKeyMap` | 41 | 5 | 6 | 0x2E | `GliderPRO/Headers/Externs.h:142` |
| `kNKeyMap` | 42 | 5 | 5 | 0x2D | `GliderPRO/Headers/Externs.h:143` |
| `kCommandKeyMap` | 48 | 6 | 7 | 0x37 | `GliderPRO/Headers/Externs.h:155` |
| `kEscKeyMap` | 50 | 6 | 5 | 0x35 | `GliderPRO/Headers/Externs.h:156` |
| `kDeleteKeyMap` | 52 | 6 | 3 | 0x33 | `GliderPRO/Headers/Externs.h:157` |
| `kSpaceBarMap` | 54 | 6 | 1 | 0x31 | `GliderPRO/Headers/Externs.h:158` |
| `kTabKeyMap` | 55 | 6 | 0 | 0x30 | `GliderPRO/Headers/Externs.h:159` |
| `kControlKeyMap` | 60 | 7 | 3 | 0x3B | `GliderPRO/Headers/Externs.h:160` |
| `kOptionKeyMap` | 61 | 7 | 2 | 0x3A | `GliderPRO/Headers/Externs.h:161` |
| `kCapsLockKeyMap` | 62 | 7 | 1 | 0x39 | `GliderPRO/Headers/Externs.h:162` |
| `kShiftKeyMap` | 63 | 7 | 0 | 0x38 | `GliderPRO/Headers/Externs.h:163` |
| `kPlusKeypadMap` | 66 | 8 | 5 | 0x45 | `GliderPRO/Headers/Externs.h:115` |
| `kTimesKeypadMap` | 68 | 8 | 3 | 0x43 | `GliderPRO/Headers/Externs.h:117` |
| `kMinusKeypadMap` | 73 | 9 | 6 | 0x4E | `GliderPRO/Headers/Externs.h:116` |
| `k5KeypadMap` | 80 | 10 | 7 | 0x57 | `GliderPRO/Headers/Externs.h:123` |
| `k4KeypadMap` | 81 | 10 | 6 | 0x56 | `GliderPRO/Headers/Externs.h:122` |
| `k3KeypadMap` | 82 | 10 | 5 | 0x55 | `GliderPRO/Headers/Externs.h:121` |
| `k2KeypadMap` | 83 | 10 | 4 | 0x54 | `GliderPRO/Headers/Externs.h:120` |
| `k1KeypadMap` | 84 | 10 | 3 | 0x53 | `GliderPRO/Headers/Externs.h:119` |
| `k0KeypadMap` | 85 | 10 | 2 | 0x52 | `GliderPRO/Headers/Externs.h:118` |
| `k9KeypadMap` | 91 | 11 | 4 | 0x5C | `GliderPRO/Headers/Externs.h:127` |
| `k8KeypadMap` | 92 | 11 | 3 | 0x5B | `GliderPRO/Headers/Externs.h:126` |
| `k7KeypadMap` | 94 | 11 | 1 | 0x59 | `GliderPRO/Headers/Externs.h:125` |
| `k6KeypadMap` | 95 | 11 | 0 | 0x58 | `GliderPRO/Headers/Externs.h:124` |
| `kUpArrowKeyMap` | 121 | 15 | 6 | 0x7E | `GliderPRO/Headers/Externs.h:129` |
| `kDownArrowKeyMap` | 122 | 15 | 5 | 0x7D | `GliderPRO/Headers/Externs.h:130` |
| `kRightArrowKeyMap` | 123 | 15 | 4 | 0x7C | `GliderPRO/Headers/Externs.h:131` |
| `kLeftArrowKeyMap` | 124 | 15 | 3 | 0x7B | `GliderPRO/Headers/Externs.h:132` |

Notes:
* Every derived `vkey` matches the documented Mac ADB virtual key codes (A=0x00, S=0x01,
  D=0x02, F=0x03, H=0x04, G=0x05, Z=0x06, X=0x07, C=0x08, V=0x09, B=0x0B, Q=0x0C, W=0x0D,
  E=0x0E, R=0x0F, T=0x11, O=0x1F, P=0x23, `.`=0x2F, N=0x2D, M=0x2E, Tab=0x30, Space=0x31,
  Delete=0x33, Esc=0x35, Cmd=0x37, Shift=0x38, CapsLock=0x39, Option=0x3A, Control=0x3B,
  keypad `*`=0x43, keypad `+`=0x45, keypad `-`=0x4E, keypad 0-9 = 0x52-0x5C, arrows
  0x7B-0x7E). This is an independent confirmation that the offsets were transcribed
  correctly.
* Offset **13** (vkey 0x0A, the ISO §/± key) has no constant. Offsets for I, J, K, L, U, Y
  and the digits also have no `*KeyMap` constants — only the 20 letters actually used as
  editor shortcuts.
* `kShiftKeyMap` = 63 is the last bit of the second `long`. In Go, read the `KeyMap` as
  `[16]byte` and test `keys[n>>3] & (0x80 >> (n&7))`. If instead you store the KeyMap as
  four **big-endian** `uint32`s, the same test is `(m[n>>5] >> (31 - (n&31))) & 1`.

### 20.2 Default key bindings

Player 1 (`theGlider`) is user-configurable via prefs; player 2 (`theGlider2`) is **not**.

| Action | Player 1 default | Constant | Player 2 (fixed) | Constant |
|---|---|---|---|---|
| left | Left arrow | `kLeftArrowKeyMap` 124 | Control | `kControlKeyMap` 60 |
| right | Right arrow | `kRightArrowKeyMap` 123 | Command | `kCommandKeyMap` 48 |
| battery/helium | Down arrow | `kDownArrowKeyMap` 122 | Option | `kOptionKeyMap` 61 |
| rubber band | Up arrow | `kUpArrowKeyMap` 121 | Shift | `kShiftKeyMap` 63 |

Sites: player 1 defaults at `GliderPRO/Sources/Main.c:135-138` (used when no prefs file
exists) and re-applied by the "Use Defaults" button at
`GliderPRO/Sources/Settings.c:1237-1240`; loaded from prefs at
`GliderPRO/Sources/Main.c:69-72` (`thePrefs.wasLeftMap` … `wasBandMap`) and stored back at
`GliderPRO/Sources/Main.c:225-228`. Player 2's are hard-coded at
`GliderPRO/Sources/InterfaceInit.c:148-151`.

Player 2 using the four *modifier* keys is deliberate: modifiers do not generate key-down
events and never conflict with the menu bar, and `GetKeys` reads them fine. A Go port on a
modern OS will find Command/Option/Control intercepted by the window manager and should
offer a remap.

Two other in-game keys are read directly, not through a binding:

| Key | Constant | Purpose | Site |
|---|---|---|---|
| Delete | `kDeleteKeyMap` 52 | player-2 self-sacrifice to catch up | `GliderPRO/Sources/Input.c:~370` |
| Esc **or** Tab | `kEscKeyMap` 50 / `kTabKeyMap` 55 | pause; which one is chosen by the pref `isEscPauseKey` | `GliderPRO/Sources/Input.c:~262`, `:~355` |

`kEscPausePictID` 1015 / `kTabPausePictID` 1016 (`GliderPRO/Sources/Input.c:17-18`) are the
two corresponding "Paused" overlays, so the on-screen hint matches the chosen key.

### 20.3 ASCII / character codes

`GliderPRO/Headers/Externs.h:29-113`, 85 constants. Used only for dialog and text-field
handling (`EventRecord.message & charCodeMask`).

Control and navigation, `GliderPRO/Headers/Externs.h:29-48`:

| Name | Value | Name | Value |
|---|---|---|---|
| `kHomeKeyASCII` | 0x01 | `kFunctionKeyASCII` | 0x10 |
| `kEnterKeyASCII` | 0x03 | `kClearKeyASCII` | 0x1A |
| `kEndKeyASCII` | 0x04 | `kEscapeKeyASCII` | 0x1B |
| `kHelpKeyASCII` | 0x05 | `kLeftArrowKeyASCII` | 0x1C |
| `kDeleteKeyASCII` | 0x08 | `kRightArrowKeyASCII` | 0x1D |
| `kTabKeyASCII` | 0x09 | `kUpArrowKeyASCII` | 0x1E |
| `kPageUpKeyASCII` | 0x0B | `kDownArrowKeyASCII` | 0x1F |
| `kPageDownKeyASCII` | 0x0C | | |
| `kReturnKeyASCII` | 0x0D | | |

Printable, `GliderPRO/Headers/Externs.h:45-58`: `kSpaceBarASCII` 0x20,
`kExclamationASCII` 0x21, `kPlusKeyASCII` 0x2B, `kMinusKeyASCII` 0x2D,
`k0KeyASCII` 0x30 … `k9KeyASCII` 0x39 (`GliderPRO/Headers/Externs.h:49-58`).

Letters: `kCapAKeyASCII` 0x41 … `kCapZKeyASCII` 0x5A
(`GliderPRO/Headers/Externs.h:60-85`, 26 constants, `0x41 + (letter − 'A')`);
`kAKeyASCII` 0x61 … `kZKeyASCII` 0x7A
(`GliderPRO/Headers/Externs.h:87-112`, 26 constants, `0x61 + (letter − 'a')`);
`kForwardDeleteASCII` 0x7F (`GliderPRO/Headers/Externs.h:113`).

These are plain ASCII. Note the Mac's non-ASCII assignments in the low range: **0x08 is
Backspace/Delete** (the key labelled "delete"), **0x7F is forward-delete**, **0x1C-0x1F are
the four arrows** (not the ASCII file separators), and **0x0D is Return** with **no 0x0A
Linefeed constant** — consistent with the CR-only line endings of the source itself.

### 20.4 Virtual key codes (`*RawKey`)

`GliderPRO/Headers/Externs.h:165-181`, 17 constants. These are used where the character
code is useless (function keys, and Tab where the character code collides with dialog
navigation).

| Name | Value | Dec | Site |
|---|---|---|---|
| `kTabRawKey` | 0x30 | 48 | `GliderPRO/Headers/Externs.h:165` |
| `kClearRawKey` | 0x47 | 71 | `GliderPRO/Headers/Externs.h:166` |
| `kF5RawKey` | 0x60 | 96 | `GliderPRO/Headers/Externs.h:167` |
| `kF6RawKey` | 0x61 | 97 | `GliderPRO/Headers/Externs.h:168` |
| `kF7RawKey` | 0x62 | 98 | `GliderPRO/Headers/Externs.h:169` |
| `kF3RawKey` | 0x63 | 99 | `GliderPRO/Headers/Externs.h:170` |
| `kF8RawKey` | 0x64 | 100 | `GliderPRO/Headers/Externs.h:171` |
| `kF9RawKey` | 0x65 | 101 | `GliderPRO/Headers/Externs.h:172` |
| `kF11RawKey` | 0x67 | 103 | `GliderPRO/Headers/Externs.h:173` |
| `kF13RawKey` | 0x69 | 105 | `GliderPRO/Headers/Externs.h:174` |
| `kF14RawKey` | 0x6B | 107 | `GliderPRO/Headers/Externs.h:175` |
| `kF10RawKey` | 0x6D | 109 | `GliderPRO/Headers/Externs.h:176` |
| `kF12RawKey` | 0x6F | 111 | `GliderPRO/Headers/Externs.h:177` |
| `kF15RawKey` | 0x71 | 113 | `GliderPRO/Headers/Externs.h:178` |
| `kF4RawKey` | 0x76 | 118 | `GliderPRO/Headers/Externs.h:179` |
| `kF2RawKey` | 0x78 | 120 | `GliderPRO/Headers/Externs.h:180` |
| `kF1RawKey` | 0x7A | 122 | `GliderPRO/Headers/Externs.h:181` |

The F-key codes are **not** in F1…F15 order — the Mac's extended keyboard assigns
F1=0x7A, F2=0x78, F3=0x63, F4=0x76, F5=0x60, F6=0x61, F7=0x62, F8=0x64, F9=0x65,
F10=0x6D, F11=0x67, F12=0x6F, F13=0x69, F14=0x6B, F15=0x71. `kF16` does not exist. A Go
port must table-map these, never compute them.

---

## 21. Editor-only constants

The level editor is compiled into the same binary and gated by `theMode == kEditMode`. Its
constants do not affect gameplay but do affect what a house file can legally contain, so a
Go port that reads houses must respect the editor's clamps.

### 21.1 Menus

Menu resource IDs, `GliderPRO/Headers/GliderDefines.h:186-189`:

| Name | Value | Site |
|---|---|---|
| `kAppleMenuID` | 128 | `GliderPRO/Headers/GliderDefines.h:186` |
| `kGameMenuID` | 129 | `GliderPRO/Headers/GliderDefines.h:187` |
| `kOptionsMenuID` | 130 | `GliderPRO/Headers/GliderDefines.h:188` |
| `kHouseMenuID` | 131 | `GliderPRO/Headers/GliderDefines.h:189` |

There are 6 `MENU` resources in the fork (IDs 128-141); the two extra are the popup menus
`kBackgroundsMenuID` = **140** (`GliderPRO/Sources/RoomInfo.c:379`, a function-local
`#define`) and the tool-palette popup driven by `CNTL` `kPopUpControl` = **129**
(`GliderPRO/Sources/Tools.c:18`).

Menu item indices, `GliderPRO/Headers/Externs.h:197-222` (1-based, as the Menu Manager
requires):

| Menu | Item | Index | Site |
|---|---|---|---|
| Apple | `iAbout` | 1 | `GliderPRO/Headers/Externs.h:197` |
| Game | `iNewGame` | 1 | `GliderPRO/Headers/Externs.h:198` |
| Game | `iTwoPlayer` | 2 | `GliderPRO/Headers/Externs.h:199` |
| Game | `iOpenSavedGame` | 3 | `GliderPRO/Headers/Externs.h:200` |
| Game | `iLoadHouse` | 5 | `GliderPRO/Headers/Externs.h:201` |
| Game | `iQuit` | 7 | `GliderPRO/Headers/Externs.h:202` |
| Options | `iEditor` | 1 | `GliderPRO/Headers/Externs.h:203` |
| Options | `iHighScores` | 3 | `GliderPRO/Headers/Externs.h:204` |
| Options | `iPrefs` | 4 | `GliderPRO/Headers/Externs.h:205` |
| Options | `iHelp` | 5 | `GliderPRO/Headers/Externs.h:206` |
| House | `iNewHouse` | 1 | `GliderPRO/Headers/Externs.h:207` |
| House | `iSave` | 2 | `GliderPRO/Headers/Externs.h:208` |
| House | `iHouse` | 4 | `GliderPRO/Headers/Externs.h:209` |
| House | `iRoom` | 5 | `GliderPRO/Headers/Externs.h:210` |
| House | `iObject` | 6 | `GliderPRO/Headers/Externs.h:211` |
| House | `iCut` | 8 | `GliderPRO/Headers/Externs.h:212` |
| House | `iCopy` | 9 | `GliderPRO/Headers/Externs.h:213` |
| House | `iPaste` | 10 | `GliderPRO/Headers/Externs.h:214` |
| House | `iClear` | 11 | `GliderPRO/Headers/Externs.h:215` |
| House | `iDuplicate` | 12 | `GliderPRO/Headers/Externs.h:216` |
| House | `iBringForward` | 14 | `GliderPRO/Headers/Externs.h:217` |
| House | `iSendBack` | 15 | `GliderPRO/Headers/Externs.h:218` |
| House | `iGoToRoom` | 17 | `GliderPRO/Headers/Externs.h:219` |
| House | `iMapWindow` | 19 | `GliderPRO/Headers/Externs.h:220` |
| House | `iObjectWindow` | 20 | `GliderPRO/Headers/Externs.h:221` |
| House | `iCoordinateWindow` | 21 | `GliderPRO/Headers/Externs.h:222` |

The gaps (4, 6, 7, 13, 16, 18) are separator lines in the `MENU` resource.

### 21.2 The tool palette

`GliderPRO/Sources/Tools.c:15-46`:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kToolsHigh` | 4 | `GliderPRO/Sources/Tools.c:15` | palette rows |
| `kToolsWide` | 4 | `GliderPRO/Sources/Tools.c:16` | palette columns |
| `kTotalTools` | **16** | `GliderPRO/Sources/Tools.c:17` | source comment: "kToolsHigh * kToolsWide" |
| `kPopUpControl` | 129 | `GliderPRO/Sources/Tools.c:18` | `CNTL` resource for the category popup |
| `kToolsPictID` | 1011 | `GliderPRO/Sources/Tools.c:46` | 16 tool icons per category |

The nine categories, each `kFirst*` / `kLast*` / `k*Base`:

| Category | `kFirst*` | `kLast*` | `k*Base` | Count | Sites |
|---|---|---|---|---|---|
| Blower | 1 | 15 | **1** | 15 | `GliderPRO/Sources/Tools.c:19-21` |
| Furniture | 1 | 15 | **21** | 15 | `GliderPRO/Sources/Tools.c:22-24` |
| Bonus | 1 | 15 | **41** | 15 | `GliderPRO/Sources/Tools.c:25-27` |
| Transport | 1 | 12 | **61** | 12 | `GliderPRO/Sources/Tools.c:28-30` |
| Switch | 1 | 9 | **81** | 9 | `GliderPRO/Sources/Tools.c:31-33` |
| Light | 1 | 8 | **101** | 8 | `GliderPRO/Sources/Tools.c:34-36` |
| Appliance | 1 | 14 | **121** | 14 | `GliderPRO/Sources/Tools.c:37-39` |
| Enemy | 1 | 9 | **141** | 9 | `GliderPRO/Sources/Tools.c:40-42` |
| Clutter | 1 | 15 | **161** | 15 | `GliderPRO/Sources/Tools.c:43-45` |

Total = 15+15+15+12+9+8+14+9+15 = **112** placeable object types.

**The critical trap:** the bases are spaced **20** apart (1, 21, 41, 61, 81, 101, 121, 141,
161) but the *object type IDs* are spaced **16** apart (`kBlowerBase` object range starts at
0x01, `kFurnitureBase` at 0x11, etc. — see §6.1-6.2). The tool-palette base is a **PICT
tile index into `kToolsPictID`**, not an object ID; the two numbering systems are unrelated
and must not be conflated. The palette shows 16 tiles per page but a category may have
fewer than 16 members (only Blower/Furniture/Bonus/Clutter fill 15 of 16).

### 21.3 The map window

`GliderPRO/Sources/Map.c:17-25`:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kMapRoomsHigh` | 9 | `GliderPRO/Sources/Map.c:17` | visible rows; source comment **"was 7"** |
| `kMapRoomsWide` | 9 | `GliderPRO/Sources/Map.c:18` | visible columns; comment **"was 7"** |
| `kMapScrollBarWidth` | 16 | `GliderPRO/Sources/Map.c:19` | scroll-bar thickness |
| `kHScrollRef` | 5L | `GliderPRO/Sources/Map.c:20` | `refCon` identifying the h-scrollbar (`GliderPRO/Sources/Map.c:406`, tested at `:661`) |
| `kVScrollRef` | 27L | `GliderPRO/Sources/Map.c:21` | `refCon` for the v-scrollbar (`GliderPRO/Sources/Map.c:411`, tested at `:684`) |
| `kMapGroundValue` | 56 | `GliderPRO/Sources/Map.c:22` | ground floor's row index in map space |
| `kNewRoomAlert` | 1004 | `GliderPRO/Sources/Map.c:23` | "create a room here?" alert |
| `kYesDoNewRoom` | 1 | `GliderPRO/Sources/Map.c:24` | its OK item |
| `kThumbnailPictID` | 1010 | `GliderPRO/Sources/Map.c:25` | 20 × 32 room thumbnails |

Thumbnail cell size, `GliderPRO/Headers/GliderDefines.h:246-247`:

| Name | Value | Site |
|---|---|---|
| `kMapRoomHeight` | 20 | `GliderPRO/Headers/GliderDefines.h:246` |
| `kMapRoomWidth` | 32 | `GliderPRO/Headers/GliderDefines.h:247` |

House coordinate limits, `GliderPRO/Headers/GliderDefines.h:543-544`:

| Name | Value | Site | Note |
|---|---|---|---|
| `kMaxNumRoomsH` | 128 | `GliderPRO/Headers/GliderDefines.h:543` | horizontal suites |
| `kMaxNumRoomsV` | 64 | `GliderPRO/Headers/GliderDefines.h:544` | vertical floors |

**128 × 64 = 8192 = `kRoomsTimesSuites`** — the two are consistent, and the map scroll
range is `0 … kMaxNumRoomsH − mapRoomsWide` = 0…119 horizontally
(`GliderPRO/Sources/Map.c:406`) and `0 … kMaxNumRoomsV − mapRoomsHigh` = 0…55 vertically
(`GliderPRO/Sources/Map.c:411`).

The floor↔row conversion is the key formula, and it appears five times:

```
v      = kMapGroundValue - thisRoom->floor;                 // Map.c:58   floor -> row
floor  = kMapGroundValue - (i + mapTopRoom);                // Map.c:139, :224  row -> floor
roomV  = kMapGroundValue - (localV + mapTopRoom);           // Map.c:605  click -> floor
mapTopRoom  = (kMapGroundValue - v) - (mapRoomsHigh / 2);   // Map.c:78   centre on a room
groundLevel = kMapGroundValue - mapTopRoom;                 // Map.c:198
```

So `floor` **increases upward** while the map row index increases downward, with the ground
floor (`floor == 0`) at row 56. Floors therefore range **−7 … +56** if the full 64 rows are
used. A Go port that stores floors as unsigned will break basements.

Map window geometry (`GliderPRO/Sources/Map.c:339-353`, `:372-373`, `:401-403`):

```
content = mapRoomsWide * kMapRoomWidth  + kMapScrollBarWidth - 2   = 9*32 + 14 = 302
        × mapRoomsHigh * kMapRoomHeight + kMapScrollBarWidth - 2   = 9*20 + 14 = 194
h-scroll at (0, bottom - 16 + 2), size (right - 16 + 3) × 16
v-scroll at (right - 16 + 2, 0), size 16 × (bottom - 16 + 3)
clip = (0, 0, right + 2 - 16, bottom + 2 - 16)
```

The `+2` / `−2` / `+3` fudges overlap the scroll bars with the window frame by one pixel
on each side, the classic Mac look. `mapRoomsWide` / `mapRoomsHigh` are *variables*
initialised from the `#define`s, so the window can be resized.

### 21.4 Room-info and house-info dialogs

`GliderPRO/Sources/RoomInfo.c:18-33`:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kRoomInfoDialogID` | 1003 | `GliderPRO/Sources/RoomInfo.c:18` | the Room… dialog |
| `kOriginalArtDialogID` | 1016 | `GliderPRO/Sources/RoomInfo.c:19` | custom-PICT sub-dialog |
| `kNoPICTFoundAlert` | 1036 | `GliderPRO/Sources/RoomInfo.c:20` | "no such PICT in this house" |
| `kRoomNameItem` | 3 | `GliderPRO/Sources/RoomInfo.c:21` | the 27-char name field |
| `kRoomLocationBox` | 6 | `GliderPRO/Sources/RoomInfo.c:22` | |
| `kRoomTilesBox` | 10 | `GliderPRO/Sources/RoomInfo.c:23` | |
| `kRoomPopupItem` | 11 | `GliderPRO/Sources/RoomInfo.c:24` | background popup (menu 140) |
| `kRoomDividerLine` | 12 | `GliderPRO/Sources/RoomInfo.c:25` | |
| `kRoomTilesBox2` | 15 | `GliderPRO/Sources/RoomInfo.c:26` | |
| `kRoomFirstCheck` | 17 | `GliderPRO/Sources/RoomInfo.c:27` | "this is the first room" |
| `kLitUnlitText` | 18 | `GliderPRO/Sources/RoomInfo.c:28` | |
| `kMiniTileWide` | **16** | `GliderPRO/Sources/RoomInfo.c:29` | width of a tile preview = `kTileWide` / 4 = 64/4 |
| `kBoundsButton` | 19 | `GliderPRO/Sources/RoomInfo.c:30` | |
| `kOriginalArtworkItem` | **19** | `GliderPRO/Sources/RoomInfo.c:31` | **same item number as `kBoundsButton`** — the two dialogs (1003 and 1016) reuse item 19 for different purposes |
| `kPICTIDItem` | 5 | `GliderPRO/Sources/RoomInfo.c:32` | in dialog 1016 |
| `kFloorSupportCheck` | 12 | `GliderPRO/Sources/RoomInfo.c:33` | in dialog 1016; **collides with `kRoomDividerLine`** for the same reason |
| `kBackgroundsMenuID` | 140 | `GliderPRO/Sources/RoomInfo.c:379` | function-local; the background popup's `MENU` |

`GliderPRO/Sources/HouseInfo.c:14-24`:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kHouseInfoDialogID` | 1001 | `GliderPRO/Sources/HouseInfo.c:14` | |
| `kBannerTextItem` | 4 | `GliderPRO/Sources/HouseInfo.c:15` | edits the 255-char banner |
| `kLockHouseButton` | 6 | `GliderPRO/Sources/HouseInfo.c:16` | sets bit 0 of `timeStamp` (§22.2) |
| `kClearScoresButton` | 9 | `GliderPRO/Sources/HouseInfo.c:17` | |
| `kTrailerTextItem` | 11 | `GliderPRO/Sources/HouseInfo.c:18` | edits the 255-char trailer |
| `kNoPhoneCheck` | 14 | `GliderPRO/Sources/HouseInfo.c:19` | sets bit 1 of `flags` (`phoneBitSet`); `DITL` 1001 item 14 is labelled **"No Phone"**, so checking it *suppresses* the phone — see §22.7 |
| `kBannerNCharsItem` | 15 | `GliderPRO/Sources/HouseInfo.c:20` | live character count |
| `kTrailerNCharsItem` | 16 | `GliderPRO/Sources/HouseInfo.c:21` | live character count |
| `kHouseSizeItem` | 18 | `GliderPRO/Sources/HouseInfo.c:22` | shows the computed data-fork size |
| `kLockHouseAlert` | 1029 | `GliderPRO/Sources/HouseInfo.c:23` | "locking is irreversible" |
| `kZeroScoresAlert` | 1032 | `GliderPRO/Sources/HouseInfo.c:24` | "erase all scores?" |

### 21.5 Linking

`GliderPRO/Sources/Link.c:15-16`:

| Name | Value | Site |
|---|---|---|
| `kLinkControlID` | 130 | `GliderPRO/Sources/Link.c:15` |
| `kUnlinkControlID` | 131 | `GliderPRO/Sources/Link.c:16` |

Link kinds, from `GliderPRO/Headers/GliderDefines.h`:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kSwitchLinkOnly` | 3 | `GliderPRO/Headers/GliderDefines.h:463` | a switch may only link to an appliance/light |
| `kTriggerLinkOnly` | 4 | `GliderPRO/Headers/GliderDefines.h:464` | a trigger may only link to a sound/transport |
| `kTransportLinkOnly` | 5 | `GliderPRO/Headers/GliderDefines.h:465` | a transport may only link to another transport |

The `retroLink` struct (`GliderPRO/Headers/GliderStructs.h:341-345`) records the reverse
direction so that deleting a target can unlink its sources.

### 21.6 Floating windows and the marquee

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kWindoidWDEF` | 2048 | `GliderPRO/Headers/GliderDefines.h:531` | `procID` = WDEF 128, variant 0 |
| `kWindoidGrowWDEF` | 2064 | `GliderPRO/Headers/GliderDefines.h:532` | `procID` = WDEF 129, variant 0 |
| `kFloatingKind` | 2048 | `GliderPRO/Sources/WindowUtils.c:14` | `windowKind` marking a float |
| `kMessageWindowTall` | 48 | `GliderPRO/Sources/WindowUtils.c:15` | height of the transient message windoid |
| `kMarqueePatListID` | 128 | `GliderPRO/Sources/Marquee.c:15` | `PAT#` with the marching-ants patterns |
| `kHandleSideLong` | 9 | `GliderPRO/Sources/Marquee.c:16` | side of a selection drag handle, px |

`procID >> 4` gives the WDEF resource ID: 2048 >> 4 = **128**, 2064 >> 4 = **129**, matching
the two `WDEF` resources actually present in the fork (§18.1). `procID & 0x0F` = 0 for both,
so neither uses a variation code.

Cursors:

| Name | Value | Site | Meaning |
|---|---|---|---|
| `kArrowCursor` | 0 | `GliderPRO/Headers/GliderDefines.h:182` | logical cursor index |
| `kBeamCursor` | 1 | `GliderPRO/Headers/GliderDefines.h:183` | |
| `kHandCursor` | 2 | `GliderPRO/Headers/GliderDefines.h:184` | |
| `kHandCursorID` | 128 | `GliderPRO/Sources/InterfaceInit.c:16` | `CURS`/`crsr` resource |
| `kVertCursorID` | 129 | `GliderPRO/Sources/InterfaceInit.c:17` | vertical-resize |
| `kHoriCursorID` | 130 | `GliderPRO/Sources/InterfaceInit.c:18` | horizontal-resize |
| `kDiagCursorID` | 131 | `GliderPRO/Sources/InterfaceInit.c:19` | diagonal-resize |
| `rAcurID` | 128 | `GliderPRO/Sources/AnimCursor.c:14` | `acur` watch-spin chain |

### 21.7 The editor's clamps on object data

These bound what a legal house may contain, so they are load-bearing for a Go loader's
validation:

| Clamp | Range | Site |
|---|---|---|
| `increment` (`data.d.byte0`, `data.e.byte0`, etc.) | 0 … **255** | `GliderPRO/Sources/ObjectEdit.c:540-543`, `:552-555` |
| object `dest`/`bounds` rects | within the 512 × 322 room, per-type minimum sizes from `srcRects[]` | `GliderPRO/Sources/ObjectEdit.c` |
| `leftStart`/`rightStart` | 0 … 255 (stored as `Byte`) | `GliderPRO/Headers/GliderStructs.h:117-118` |
| `numObjects` | 0 … `kMaxRoomObs` = **24** | `GliderPRO/Headers/GliderDefines.h:250` |
| `nRooms` | **no constant** — there is no `kMaxRooms` anywhere in the tree (verified by `grep -rn kMaxRooms Headers/ Sources/` → 0 hits); bounded only by memory and by the 128 × 64 map space | — |
| `floor` | −7 … +56 (implied by `kMapGroundValue` 56 and `kMaxNumRoomsV` 64) | `GliderPRO/Sources/Map.c:22`, `GliderPRO/Headers/GliderDefines.h:544` |
| `suite` | 0 … `kMaxNumRoomsH − 1` = **127** | `GliderPRO/Headers/GliderDefines.h:543` |

**Defect:** the editor draws the right-hand glider entry marker at x = 0 rather than
`kRoomWide − 48`. `GliderPRO/Sources/ObjectEdit.c:558` sets up `rightStartGliderDest`
starting at x = 0, whereas the two other places that build the same rect —
`GliderPRO/Sources/ObjectEdit.c:1517` and `GliderPRO/Sources/ObjectEdit.c:2066` — use
`kRoomWide - 48`. The runtime entry rect (`GliderPRO/Sources/Transit.c:207-208`) uses
`kRoomWide - 48`, so the editor's initial marker position is wrong until it is next
recalculated. Separately, the editor's left-start marker rect is drawn 2 px lower and 4 px
shorter than the real entry rect built in `GliderPRO/Sources/Transit.c:173-176`.
---

## 22. On-disk data structures — sizes, offsets, and verified byte dumps

Everything in this section was **measured**, not inferred. Two independent methods were
used and cross-checked:

1. **Compiler measurement.** The nine union arms and the seven file-format structs were
   transcribed verbatim from `GliderPRO/Headers/GliderStructs.h:11-198` into a standalone
   C file with `#pragma pack(2)` (the GCC equivalent of the Mac
   `#pragma options align=mac68k`) and `long` forced to 32 bits, then `sizeof` and
   `offsetof` were printed for every field. Source: `/tmp/wf-constants/csz/g2.c`.
2. **Byte-level parsing of shipped files.** Five of the `GliderPRO/Houses/*.binhex`
   files were BinHex-4.0-decoded with `/tmp/wf-constants/binhex.py` and the resulting
   data forks parsed field-by-field with `/tmp/wf-constants/house.py`.

Where the author's own `// total = N` comments disagree with the measurement, the
measurement is reported and the discrepancy called out.

### 22.1 The alignment rule

`GliderPRO/Headers/Externs.h:231` and `:269` bracket `prefsInfo` with

```c
#pragma options align=mac68k
...
#pragma options align=reset
```

`GliderStructs.h` has **no** `#pragma` at all, but every struct in it is laid out as if
`align=mac68k` were in force, because that was the compiler default for a 68k build and
because the on-disk sizes prove it (see §22.4). The mac68k rule is:

| Type | Size | Alignment |
|---|---|---|
| `char` / `Byte` / `Boolean` / `SInt8` | 1 | 1 |
| `short` / `SInt16` | 2 | 2 |
| `long` / `SInt32` / `unsigned long` | **4** | **2** ← not 4 |
| `Point` (2 shorts) | 4 | 2 |
| `Rect` (4 shorts) | 8 | 2 |
| `Str*` (`char[n]`) | n | 1 |
| struct | rounded up to a multiple of 2 | 2 |

The one consequential difference from a modern LP64/ILP32 default is that **`long` is
2-byte aligned, not 4-byte aligned**. This is what puts `houseType.timeStamp` at offset 4
and `gameType.timeStamp` at offset 4 but `game2Type.timeStamp` at offset **74** (odd
multiple of 2, following a 70-byte `FSSpec` and a `short`). A Go port using
`encoding/binary` on explicitly-offset fields is unaffected; a Go port that relies on
`unsafe.Sizeof` of a mirrored struct **will get different numbers**.

`Point` is `{short v; short h;}` — **v first**. `Rect` is `{short top, left, bottom,
right;}`. All multi-byte integers on disk are **big-endian**.

### 22.2 The nine object union arms

`GliderPRO/Headers/GliderStructs.h:11-105`. All nine arms are exactly **10 bytes**;
`objectType` is `short what` (offset 0) + `union data` (offset 2) = **12 bytes**.
Measured `sizeof`: `blowerType` 10, `furnitureType` 10, `bonusType` 10, `transportType`
10, `switchType` 10, `lightType` 10, `applianceType` 10, `enemyType` 10, `clutterType`
10, `objectType` 12.

Offsets are given **relative to the start of `objectType`**, i.e. arm offset + 2, which
is what a Go decoder reading a 12-byte record actually needs:

| Arm | Field | Type | Arm off. | `objectType` off. | Size | Declared at |
|---|---|---|---|---|---|---|
| — | `what` | `short` | — | **0** | 2 | `GliderStructs.h:92` |
| `a` `blowerType` | `topLeft` | `Point` | 0 | **2** | 4 | `GliderStructs.h:13` |
| | `distance` | `short` | 4 | **6** | 2 | `:14` |
| | `initial` | `Boolean` | 6 | **8** | 1 | `:15` |
| | `state` | `Boolean` | 7 | **9** | 1 | `:16` |
| | `vector` | `Byte` | 8 | **10** | 1 | `:17` |
| | `tall` | `Byte` | 9 | **11** | 1 | `:18` |
| `b` `furnitureType` | `bounds` | `Rect` | 0 | **2** | 8 | `:23` |
| | `pict` | `short` | 8 | **10** | 2 | `:24` |
| `c` `bonusType` | `topLeft` | `Point` | 0 | **2** | 4 | `:29` |
| | `length` | `short` | 4 | **6** | 2 | `:30` (grease spill length) |
| | `points` | `short` | 6 | **8** | 2 | `:31` (invisible-bonus points) |
| | `state` | `Boolean` | 8 | **10** | 1 | `:32` |
| | `initial` | `Boolean` | 9 | **11** | 1 | `:33` |
| `d` `transportType` | `topLeft` | `Point` | 0 | **2** | 4 | `:38` |
| | `tall` | `short` | 4 | **6** | 2 | `:39` (invisible-transport height) |
| | `where` | `short` | 6 | **8** | 2 | `:40` (**destination room, floor/suite-packed**) |
| | `who` | `Byte` | 8 | **10** | 1 | `:41` (destination object index) |
| | `wide` | `Byte` | 9 | **11** | 1 | `:42` |
| `e` `switchType` | `topLeft` | `Point` | 0 | **2** | 4 | `:47` |
| | `delay` | `short` | 4 | **6** | 2 | `:48` |
| | `where` | `short` | 6 | **8** | 2 | `:49` (**target room, floor/suite-packed**) |
| | `who` | `Byte` | 8 | **10** | 1 | `:50` (target object index) |
| | `type` | `Byte` | 9 | **11** | 1 | `:51` |
| `f` `lightType` | `topLeft` | `Point` | 0 | **2** | 4 | `:56` |
| | `length` | `short` | 4 | **6** | 2 | `:57` |
| | `byte0` | `Byte` | 6 | **8** | 1 | `:58` |
| | `byte1` | `Byte` | 7 | **9** | 1 | `:59` |
| | `initial` | `Boolean` | 8 | **10** | 1 | `:60` |
| | `state` | `Boolean` | 9 | **11** | 1 | `:61` |
| `g` `applianceType` | `topLeft` | `Point` | 0 | **2** | 4 | `:66` |
| | `height` | `short` | 4 | **6** | 2 | `:67` (**doubles as PICT ID for `kCustomPict`**) |
| | `byte0` | `Byte` | 6 | **8** | 1 | `:68` |
| | `delay` | `Byte` | 7 | **9** | 1 | `:69` |
| | `initial` | `Boolean` | 8 | **10** | 1 | `:70` |
| | `state` | `Boolean` | 9 | **11** | 1 | `:71` |
| `h` `enemyType` | `topLeft` | `Point` | 0 | **2** | 4 | `:76` |
| | `length` | `short` | 4 | **6** | 2 | `:77` |
| | `delay` | `Byte` | 6 | **8** | 1 | `:78` |
| | `byte0` | `Byte` | 7 | **9** | 1 | `:79` |
| | `initial` | `Boolean` | 8 | **10** | 1 | `:80` |
| | `state` | `Boolean` | 9 | **11** | 1 | `:81` |
| `i` `clutterType` | `bounds` | `Rect` | 0 | **2** | 8 | `:86` |
| | `pict` | `short` | 8 | **10** | 2 | `:87` |

**The `vector` bit layout**, documented in the author's own comment at
`GliderPRO/Headers/GliderStructs.h:16-17`:

```
                    F. lf. dn. rt. up
vector = | x | x | x | x | 8 | 4 | 2 | 1 |
```

so bit 0 (**1**) = up, bit 1 (**2**) = right, bit 2 (**4**) = down, bit 3 (**8**) = left,
and bit 4 (**16**, the `F.` column) is a fifth flag. Bits 5-7 are unused.

Observed in the wild: every `blowerType`-shaped object in `Empty House` room 0 and 1 has
`vector = 1` (up) — e.g. `obj[0] what=0x51 data=000400e1004000000101` decodes to
`topLeft = (v=4, h=225)`, `distance = 0x0040 = 64`, `initial = 0`, `state = 0`,
`vector = 1`, `tall = 1`.

**Union aliasing is total and intentional.** Because `data` is a union, the same ten bytes
are `distance`+`initial`+`state`+`vector`+`tall` for a blower and
`bounds`(8)+`pict`(2) for furniture. A Go port must decode the ten bytes **as a raw
`[10]byte` first and then interpret according to `what`** — there is no discriminated
union on disk and no tag redundancy. The `what` value is the *only* thing that says which
interpretation is correct; see §6.1 for all 116 values.

The uninitialised-arm hazard is visible in the dumps: `Sampler` room 0 `obj[1]` is
`what=0x37 data=000000000000ffffff00`. Bytes 6-8 are `FF FF FF`. Read as
`blowerType.initial/state/vector` those are 255/255/255; read as the arm that object 0x37
actually uses they are simply padding the editor never wrote. **A Go loader must not
validate or normalise bytes belonging to arms other than the one selected by `what`.**

### 22.3 `objectType`, `scoresType`, `gameType`, `savedRoom` — measured layouts

#### `objectType` — 12 bytes (`GliderPRO/Headers/GliderStructs.h:90-105`)

| Off | Field | Type | Size |
|---|---|---|---|
| 0 | `what` | `short` | 2 |
| 2 | `data` | union (9 arms) | 10 |

Author's comment `// total = 12` — **correct**.

`kObjectIsEmpty` is written into `what` to mark an unused slot
(`GliderPRO/Sources/Room.c:182`).

#### `scoresType` — 292 bytes (`GliderPRO/Headers/GliderStructs.h:107-114`)

| Off | Field | Type | Size | Note |
|---|---|---|---|---|
| 0 | `banner` | `Str31` = `char[32]` | 32 | Pascal string, len byte + 31 |
| 32 | `names[10]` | `Str15` = `char[16]` × 10 | 160 | `kMaxScores` = 10 |
| 192 | `scores[10]` | `long` × 10 | 40 | |
| 232 | `timeStamps[10]` | `unsigned long` × 10 | 40 | Mac epoch seconds |
| 272 | `levels[10]` | `short` × 10 | 20 | rooms visited |

Author's comment `// total = 292` — **correct**. `kMaxScores` = **10**
(`GliderPRO/Headers/GliderDefines.h:249`).

Verified against `Sampler`: `banner = 'Your Message Here'`,
`names = ['Your Name', 'Your Name', '--------------' × 8]`,
`scores = (5200, 5100, 0 × 8)`, `levels = (2, 1, 0 × 8)`. The
`'--------------'` filler (14 hyphens) is what `ZeroHighScores()` writes into unused
slots.

#### `gameType` — 40 bytes (`GliderPRO/Headers/GliderStructs.h:116-134`)

| Off | Field | Type | Size | Note |
|---|---|---|---|---|
| 0 | `version` | `short` | 2 | `kSavedGameVersion` = **0x0200** (`GliderPRO/Sources/SavedGames.c:14`) |
| 2 | `wasStarsLeft` | `short` | 2 | `numStarsRemaining` |
| 4 | `timeStamp` | `long` | 4 | `GetDateTime` seconds |
| 8 | `where` | `Point` | 4 | `{v = theGlider.dest.top, h = theGlider.dest.left}` |
| 12 | `score` | `long` | 4 | `theScore` |
| 16 | `unusedLong` | `long` | 4 | written 0 (`SavedGames.c:325`) |
| 20 | `unusedLong2` | `long` | 4 | written 0 (`SavedGames.c:326`) |
| 24 | `energy` | `short` | 2 | `batteryTotal` — signed; negative = helium |
| 26 | `bands` | `short` | 2 | `bandsTotal` |
| 28 | `roomNumber` | `short` | 2 | `thisRoomNumber` (an **index**, not floor/suite) |
| 30 | `gliderState` | `short` | 2 | `theGlider.mode` |
| 32 | `numGliders` | `short` | 2 | `mortals` |
| 34 | `foil` | `short` | 2 | `foilTotal` |
| 36 | `unusedShort` | `short` | 2 | written 0 (`SavedGames.c:333`) |
| 38 | `facing` | `Boolean` | 1 | `theGlider.facing` |
| 39 | `showFoil` | `Boolean` | 1 | |

Author's comment `// total = 40` — **correct**. Measured `sizeof` = 40. Note the field
order in the *save* code (`GliderPRO/Sources/SavedGames.c:318-335`) is exactly the
declaration order, so the on-disk order is unambiguous.

#### `savedRoom` — 292 bytes (`GliderPRO/Headers/GliderStructs.h:136-142`)

| Off | Field | Type | Size | Note |
|---|---|---|---|---|
| 0 | `unusedShort` | `short` | 2 | written 0 (`SavedGames.c:113`) |
| 2 | `unusedByte` | `Byte` | 1 | written 0 (`SavedGames.c:114`) |
| 3 | `visited` | `Boolean` | 1 | |
| 4 | `objects[24]` | `objectType` × 24 | 288 | `kMaxRoomObs` = **24** |

Author's comment `// total = 292` — **correct**. 4 + 12 × 24 = 292.

### 22.4 `roomType` — 348 bytes, the single most important record

`GliderPRO/Headers/GliderStructs.h:166-180`. Measured `sizeof` = **348**; author's
comment `// total = 348` — **correct**.

| Off | Hex | Field | Type | Size | Meaning / observed values |
|---|---|---|---|---|---|
| 0 | 0x000 | `name` | `Str27` = `char[28]` | 28 | Pascal string, max 27 chars. Across all 22 houses the observed length byte spans **1…27**, and 51 rooms use the full 27 (e.g. `'Welcome To The Land Of ... '`), so the 27-char limit is genuinely exercised |
| 28 | 0x01C | `bounds` | `short` | 2 | packed original-artwork bounds; **0 = "not set"** (see §22.5). Across all 22 houses: **0** (2401 rooms) plus every **odd** value 1…47, and 49, 53, 55, 57, 59, 61, 63 — never an even non-zero value, which independently confirms bit 0 is the "explicitly set" sentinel |
| 30 | 0x01E | `leftStart` | `Byte` | 1 | glider entry **v** offset when entering from the left, added to `kGliderStartsDown` = 32. Default 32 (`GliderPRO/Sources/Room.c:170`), clamped 0…255 (`GliderPRO/Sources/ObjectEdit.c:540-543`). Observed across all 22 houses: the **full 0…255 byte range** — 32 dominates (3208 rooms) and 0 is common (642 rooms), with a long tail up to 255 |
| 31 | 0x01F | `rightStart` | `Byte` | 1 | same for entry from the right. Default 32 (`Room.c:171`). Observed: also the full 0…255 range; 32 dominates (3211), 0 common (639) |
| 32 | 0x020 | `unusedByte` | `Byte` | 1 | written 0 (`Room.c:173`); **0 in all 4070 rooms of all 22 houses** |
| 33 | 0x021 | `visited` | `Boolean` | 1 | player has been here; part of save state. Only 0 and 1 observed (436 rooms marked visited) |
| 34 | 0x022 | `background` | `short` | 2 | PICT resource ID of the backdrop. `kBaseBackgroundID` **2000** … 2017 = one of the `kNumBackgrounds` = **18** built-ins; ≥ `kUserBackground` **3000** = house-supplied PICT. Observed: all 18 of 2000-2017, plus 3000-3053, 3100-3104, 3207-3236, 3300-3353 |
| 36 | 0x024 | `tiles[8]` | `short` × 8 | 16 | per-64-px-column tile index into the background PICT strip; `kNumTiles` = **8** |
| 52 | 0x034 | `floor` | `short` | 2 | logical floor, stored **raw and signed — no bias**. Legal range [−7, 56] (`GliderPRO/Sources/HouseLegal.c:804-805`); observed −7…39 across the 22 houses, with negatives on disk in 12 of them. The `+ kNumUndergroundFloors` bias exists only in the *link* packing (`data.d.where` / `data.e.where`), not here — see §22.6 |
| 54 | 0x036 | `suite` | `short` | 2 | logical column, raw. Legal range [0, 127] (`HouseLegal.c:814-815`); the full range is used in shipped houses. **`suite == kRoomIsEmpty` (−1) marks a deleted/placeholder room** (`GliderPRO/Sources/House.c:766`, `GliderPRO/Sources/Room.c:190`, `GliderPRO/Sources/HouseLegal.c:755`) |
| 56 | 0x038 | `openings` | `short` | 2 | cached opening bitmask; written 0 on room creation (`Room.c:179`) and recomputed at load. **0x0000 in all 4070 rooms of all 22 houses** |
| 58 | 0x03A | `numObjects` | `short` | 2 | count of live entries in `objects[]`. Observed every value **0…24** |
| 60 | 0x03C | `objects[24]` | `objectType` × 24 | 288 | `kMaxRoomObs` = **24** (`GliderPRO/Headers/GliderDefines.h:250`) |

Note `openings` is **not** authoritative on disk — `DetermineRoomOpenings()`
(`GliderPRO/Sources/Room.c:816-933`) recomputes `leftOpen`/`rightOpen`/`topOpen`/
`bottomOpen` from `background`, `bounds` and `tiles[]` every time a room becomes current.
All 4070 rooms in all 22 shipped houses have `openings == 0`. **A Go port should ignore
the stored value and recompute.**

`numObjects` is likewise advisory: room 0 of `Sampler` reports 7 and the seven records are
real, but loops that walk objects use `kMaxRoomObs` and test
`what == kObjectIsEmpty` rather than trusting `numObjects` (e.g.
`GliderPRO/Sources/Room.c:181-182`, `GliderPRO/Sources/House.c:775`).

### 22.5 `roomType.bounds` — the packed original-artwork bounds word

Two sites define the format exactly. **Decoder**, `GliderPRO/Sources/RoomInfo.c:744-749`:

```c
	tempShort = thisRoom->bounds >> 1;			// version 2.0 house
	originalLeftOpen   = ((tempShort &  1) ==  1);
	originalTopOpen    = ((tempShort &  2) ==  2);
	originalRightOpen  = ((tempShort &  4) ==  4);
	originalBottomOpen = ((tempShort &  8) ==  8);
	originalFloor      = ((tempShort & 16) == 16);
```

**Encoder**, `GliderPRO/Sources/RoomInfo.c:767-780`:

```c
	tempShort = 0;
	if (originalLeftOpen)   tempShort += 1;
	if (originalTopOpen)    tempShort += 2;
	if (originalRightOpen)  tempShort += 4;
	if (originalBottomOpen) tempShort += 8;
	if (originalFloor)      tempShort += 16;
	tempShort = tempShort << 1;		// shift left 1 bit
	tempShort += 1;					// flag that says orginal bounds used
	thisRoom->bounds = tempShort;
```

So the layout of the 16-bit `bounds` word is:

| `bounds` bit | Value | Meaning |
|---|---|---|
| 0 | **1** | "original bounds are explicitly set" sentinel — always 1 when the word is meaningful |
| 1 | **2** | left side open |
| 2 | **4** | top open (no ceiling) |
| 3 | **8** | right side open |
| 4 | **16** | bottom open (no floor) |
| 5 | **32** | floor support present **and** "room is a structure" (see below) |
| 6-15 | — | unused, always 0 |

Therefore `bounds == 0` means *"no explicit bounds; fall back to the built-in table or to
the `'bnds'` resource"*, and any non-zero legal value is **odd**.

Observed values, from `In The Mirror`:

| Room | `bounds` | `bounds >> 1` | left | top | right | bottom | floor/structure | `tiles[]` |
|---|---|---|---|---|---|---|---|---|
| 0 `'Down to Business'` | **45** = 0b101101 | 22 = 0b10110 | 0 | 1 | 1 | 0 | **1** | `(0,1,1,1,1,2,1,1)` |
| 1 | **39** = 0b100111 | 19 = 0b10011 | 1 | 1 | 0 | 0 | **1** | — |

Both are odd, as required, and both have bit 5 set.

**The overloaded bit 5.** `GliderPRO/Sources/Room.c:777` reads the *same bit* as
"is this room a structure?":

```c
	if ((*thisHouse)->rooms[roomNum].bounds != 0)
		isStructure = (((*thisHouse)->rooms[roomNum].bounds & 32) == 32);
```

while `GliderPRO/Sources/RoomInfo.c:749-751` presents it to the user as the
**`kFloorSupportCheck`** checkbox (dialog item 12). `bounds & 32` and
`(bounds >> 1) & 16` are the identical bit. So in a version-2.0 house
"has floor support" and "is a structure (drawn as a building on the map) rather than open
sky" are **not separable** — one checkbox drives both. This is not a bug, but it is a
non-obvious coupling a Go port must preserve.

**Consumers of `bounds`**, all of which use the same `>> 1` then mask idiom:

| Site | Mask | Derives |
|---|---|---|
| `GliderPRO/Sources/Room.c:828`, `:831` | `& 0x0001` | `leftOpen` |
| `GliderPRO/Sources/Room.c:832` | `& 0x0004` | `rightOpen` |
| `GliderPRO/Sources/Room.c:1146`, `:1149` | `& 0x0008` | `hasFloor = (bit clear)` (`DoesRoomHaveFloor`) |
| `GliderPRO/Sources/Room.c:1111`, `:1114` | `& 0x0008` | `hasFloor = (bit clear)` (`IsShadowVisible`, `:1103-1134`) |
| `GliderPRO/Sources/Room.c:1180`, `:1183` | `& 0x0002` | `hasCeiling = (bit clear)` |
| `GliderPRO/Sources/Room.c:777` | `bounds & 32` | `isStructure` |
| `GliderPRO/Sources/RoomInfo.c:744-749` | 1/2/4/8/16 | editor round-trip |

Note the **inversion**: bits 1/3 of `boundsCode` mean *open*, so `hasCeiling` and
`hasFloor` are the logical negation.

**The `bounds == 0` fallback path.** `GliderPRO/Sources/Room.c:830`,
`:1113`, `:1148`, `:1182` all call `GetOriginalBounding(background)`, which is
`GliderPRO/Sources/Room.c:937-965`:

```c
	boundsRes = (boundsHand)GetResource('bnds', theID);
	if (boundsRes == nil)
	{
		if (PictIDExists(theID))
			YellowAlert(kYellowNoBoundsRes, 0);
		boundCode = 0;
	}
	else
	{
		boundCode = 0;
		if ((*boundsRes)->left)   boundCode += 1;
		if ((*boundsRes)->top)    boundCode += 2;
		if ((*boundsRes)->right)  boundCode += 4;
		if ((*boundsRes)->bottom) boundCode += 8;
	}
```

with `boundsType` = `{Boolean left, top, right, bottom;}` = 4 bytes
(`GliderPRO/Headers/GliderStructs.h:266-272`).

**`'bnds'` resources DO ship, and a Go port must implement the lookup.** A census of the
resource forks of all 22 distributed houses (BinHex-decoded, resource map re-parsed
byte-by-byte) finds `'bnds'` in **8** of them, always 4 bytes each, always with IDs that
match a `PICT` ID in the same file:

| House | `'bnds'` count | IDs |
|---|---:|---|
| `Slumberland` | 20 | 3000-3010, 3300-3308 |
| `ImagineHouse PRO II` | 14 | 3000-3008, 3300-3304 |
| `Demo House` | 10 | 3000-3005, 3300-3303 |
| `Castle o' the Air` | 9 | 3000-3003, 3300-3304 |
| `Rainbow's End` | 6 | 3001, 3301-3305 |
| `Land of Illusion` | 5 | 3001-3003, 3301-3302 |
| `Leviathan` | 4 | 3000-3001, 3300-3301 |
| `The Asylum Pro` | 2 | 3000-3001 |

The 14 other houses (`Art Museum`, `CD Demo House`, `California or Bust!`, `Davis Station`,
`Empty House`, `Fun House`, `Grand Prix`, `In The Mirror`, `Metropolis`, `Nemo's Market`,
`Sampler`, `SpacePods`, `Teddy World`, `Titanic`) have none.

The bodies are genuinely non-trivial — the four `Boolean` bytes are `0x00`/`0x01` in
`left, top, right, bottom` order, exactly as `boundsType` declares, and the resulting
`boundCode` covers most of the 0-15 range. Examples:

| House / ID | bytes | `boundCode` | meaning |
|---|---|---:|---|
| `Demo House` 3000 | `00 00 01 00` | 4 | right open only |
| `Demo House` 3004 | `01 00 00 00` | 1 | left open only |
| `Demo House` 3005 | `00 00 00 00` | 0 | fully closed |
| `Demo House` 3300 | `00 01 01 01` | 14 | top+right+bottom open |
| `Castle o' the Air` 3003 | `01 01 01 01` | 15 | all four open |
| `Slumberland` 3301 | `00 01 00 01` | 10 | top+bottom open |
| `Slumberland` 3303 | `01 00 00 01` | 9 | left+bottom open |
| `Rainbow's End` 3302 | `01 01 00 00` | 3 | left+top open |
| `Leviathan` 3000 | `00 01 00 00` | 2 | top open only |
| `The Asylum Pro` 3000/3001 | `00 00 00 00` | 0 | fully closed |

So the `bounds == 0` fallback is a **live** path in shipped content: a Go port that
hard-codes `GetOriginalBounding() == 0` will make rooms in those eight houses closed
that should be open (and will leave gliders unable to leave, or standing on floors that
should not exist). Implement the resource read.

The `boundCode == 0` case is still reachable two ways — no `'bnds'` for that PICT ID
(which also raises `kYellowNoBoundsRes` when the PICT itself exists), or an all-zero
`'bnds'` body as in `The Asylum Pro` — and means **closed on all four sides with no
floor support**.

> [verified by decoding all 22 `GliderPRO/Houses/*.binhex` resource forks and parsing
> the classic resource map directly; the earlier claim in this document that no `'bnds'`
> resource exists anywhere was wrong.]

**Built-in backgrounds bypass `bounds` entirely.** For `background < kUserBackground`
(3000), `DetermineRoomOpenings` (`GliderPRO/Sources/Room.c:844-922`) switches on the
background ID and derives openness from `tiles[0]` and `tiles[7]`:

| Backgrounds | left wall iff | right wall iff | Site |
|---|---|---|---|
| `kSimpleRoom`, `kPaneledRoom`, `kBasement`, `kChildsRoom`, `kAsianRoom`, `kUnfinishedRoom`, `kSwingersRoom`, `kBathroom`, `kLibrary`, `kSky` | `tiles[0] == 0` | `tiles[7] == kNumTiles-1` (7) | `Room.c:848-868` |
| `kDirt` | `tiles[0] == 1` | `tiles[7] == 7` | `Room.c:870-881` |
| `kMeadow` | `tiles[0] == 6` | `tiles[7] == 7` | `Room.c:883-894` |
| `kGarden`, `kSkywalk`, `kField`, `kStratosphere`, `kStars` | never (always open) | never (always open) | `Room.c:896-905` |
| default | `tiles[0] == 0` | `tiles[7] == 7` | `Room.c:907-920` |

Note the **inconsistency at `Room.c:879-880`**: the `kDirt` case tests `leftTile == 1`
for the wall threshold but then computes `leftOpen = (leftTile != 0)`, so a `kDirt` room
with `tiles[0] == 1` gets `leftThresh = kLeftWallLimit` (a wall) *and*
`leftOpen = true`. Every other case keeps the two consistent. Reproduce as written.

Floor and ceiling for built-ins (`GliderPRO/Sources/Room.c:1153-1164`,
`:1187-1202`):

| Predicate | false for backgrounds | else |
|---|---|---|
| `DoesRoomHaveFloor()` | `kSky`, `kStratosphere`, `kStars` | true |
| `IsShadowVisible()` | `kRoof`, `kSky`, `kStratosphere`, `kStars` | true |
| `DoesRoomHaveCeiling()` | `kGarden`, `kMeadow`, `kField`, `kRoof`, `kSky`, `kStratosphere`, `kStars` | true |

`IsShadowVisible` (`GliderPRO/Sources/Room.c:1103-1134`) is a near-duplicate of
`DoesRoomHaveFloor` (`:1138-1168`) — identical `bounds`/`'bnds'` path, identical
`& 0x0008` test — but its built-in switch **also** lists `kRoof`. So a `kRoof` room has a
floor for collision purposes but draws no glider shadow. Copy both functions separately;
do not collapse them.

The four wall thresholds (`GliderPRO/Headers/GliderDefines.h:506-509`):

| Name | Value | Comment in source |
|---|---|---|
| `kLeftWallLimit` | **12** | |
| `kNoLeftWallLimit` | **−24** | `// 0 - (kGliderWide / 2)` |
| `kRightWallLimit` | **500** | |
| `kNoRightWallLimit` | **536** | `// kRoomWide + (kGliderWide / 2)` |

With `kRoomWide` = 512 and `kGliderWide` = 48, so the "no wall" limits let the glider
protrude exactly half its width past the room edge before a room transition triggers.

### 22.6 `floor` / `suite` — decimal packing and the underground bias

Two separate encodings coexist.

**(a) `roomType.floor` and `roomType.suite` are stored raw and UNBIASED** at offsets 52
and 54. `floor` is a **signed** map row that may be negative (below ground); `suite` is
the map column. Three pieces of evidence:

1. `CreateNewRoom` writes `thisRoom->floor = v; thisRoom->suite = h;` straight from the
   map-window row/column (`GliderPRO/Sources/Room.c:177-178`).
2. `GetRoomNumber(short floor, short suite)` (`GliderPRO/Sources/Room.c:737-758`) does a
   linear scan comparing `rooms[i].suite == suite && rooms[i].floor == floor`, and every
   caller feeds it the **output of `ExtractFloorSuite`**, which has already *subtracted*
   the bias (`GliderPRO/Sources/House.c:350-351`, `:370-371`, `:573`, `:596`, `:720`).
3. `ValidateRoomNumbers` (`GliderPRO/Sources/HouseLegal.c:785-828`) enforces
   **`floor ∈ [−7, 56]`** (`:804-805`) and **`suite ∈ [0, 127]`** (`:814-815`), marking
   any offender `suite = kRoomIsEmpty` and incrementing `houseErrors`. A negative legal
   floor is only possible if the field is unbiased.

`CheckDuplicateFloorSuite` (`GliderPRO/Sources/HouseLegal.c:646-682`) allocates a
`kRoomsTimesSuites` = **8192**-byte pigeonhole array
(`#define` at `GliderPRO/Sources/HouseLegal.c:648`) and indexes it

```c
	bitPlace = (((*thisHouse)->rooms[i].floor + 7) * 128) + (*thisHouse)->rooms[i].suite;
```

so 8192 = **64 floors × 128 suites**, with a **+7** bias here, not +8. Two rooms sharing a
(floor, suite) is an error: the second one is destroyed
(`suite = kRoomIsEmpty`, `GliderPRO/Sources/HouseLegal.c:672`). Note the mismatch:
`kNumUndergroundFloors` is **8** but the validated floor range is −7…56 and the
pigeonhole bias is +7, giving 64 rows. `floor = −8` would pass the packed encoding
(`floor + 8 = 0`) yet be rejected by `ValidateRoomNumbers` and would index
`bitPlace = −128` in `CheckDuplicateFloorSuite`, which only prints
`DebugStr("\pBlew array")` (`:668`) and then **still writes out of bounds** at `:675`. A Go
port should treat −7…56 as the authoritative floor range and range-check before indexing.

`kMaxNumRoomsH` = **128** and `kMaxNumRoomsV` = **64**
(`GliderPRO/Headers/GliderDefines.h:543-544`) are the same 128 × 64 map grid.

Observed across all 22 shipped houses (4070 room records): `floor` spans **−7…39** and
`suite` spans the full **0…127**. Only two houses are single-floor horizontal strips —
`Sampler` (2 rooms, floor 1) and `California or Bust!` (16 rooms, floor 1); every other
house is a genuine multi-floor tower, and 16 of the 22 store **negative** floors on disk
(`Rainbow's End`, `Slumberland` and `Titanic` reach the legal minimum −7).
`SpacePods` is the outlier at the top, floors 4-39 and suites
110-127. Most houses centre their suites near 64 because that is where the editor starts a
new house, but `Leviathan` (0-94) and `Teddy World` (1-113) show that is only a
convention. This is direct evidence that `roomType.floor` is stored **signed and
unbiased**: a biased field could never hold −7.

> [verified by decoding all 22 `GliderPRO/Houses/*.binhex` data forks and reading
> `floor`/`suite` at record offsets 52/54; an earlier draft of this section claimed all
> houses were single-floor strips on floor 1, which is true only of `Sampler` and
> `California or Bust!`.]

**(b) `data.d.where` and `data.e.where` hold a decimal-packed floor/suite pair.**
`GliderPRO/Sources/Link.c:34-53`:

```c
short MergeFloorSuite (short floor, short suite)
{
	return ((suite * 100) + floor);
}

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

Three things to note, all of them traps:

1. **The field order flipped between house version 1 and 2.** For `version < 0x0200` the
   packing is `floor * 100 + suite`; for `version >= 0x0200` it is `suite * 100 + floor`.
   `MergeFloorSuite` only ever produces the **version-2** form. Every shipped house has
   `version == 0x0200`, so the version-1 branch is dead for the bundled content but must
   be kept for third-party houses.
2. **`MergeFloorSuite` does not add the bias but `ExtractFloorSuite` subtracts it.** The
   `+ kNumUndergroundFloors` is done by every caller, explicitly:
   `GliderPRO/Sources/Link.c:277`, `GliderPRO/Sources/House.c:788`, `:804`,
   `GliderPRO/Sources/Scrap.c:148`, `:153`. Miss the bias in one place and destinations
   shift by eight floors.
3. **`where == -1` means "no destination"** and must be tested *before* unpacking
   (`GliderPRO/Sources/House.c:785`, `:801`).

`ConvertHouseVer1To2()` (`GliderPRO/Sources/House.c:746-820`) is the migration: for every
non-empty room and every object whose `what` is one of `kMailboxLf`, `kMailboxRt`,
`kFloorTrans`, `kCeilingTrans`, `kInvisTrans`, `kDeluxeTrans` (arm `d`) or
`kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch`,
`kInvisSwitch`, `kTrigger`, `kLgTrigger` (arm `e`), it does
`ExtractFloorSuite(where); floor += kNumUndergroundFloors; where = MergeFloorSuite(floor, suite);`
then sets `version = kHouseVersion`.

**Verified against real bytes in `Sampler`.** `Sampler` is a two-room house:
room 0 `'Entrance'` at `floor = 1, suite = 67`, room 1 `'Welcome'` at
`floor = 1, suite = 66`. Predicted packings, version-2 rule with the +8 bias:

- room 0 → `suite*100 + (floor + 8)` = `67*100 + 9` = **6709** = 0x1A35
- room 1 → `66*100 + 9` = **6609** = 0x19D1

The observed 12-byte object records (hex is the 10 data bytes, offsets 2-11):

| Room | Obj | `what` | Object | data bytes | `topLeft` | `tall` | **`where`** | `who` | `wide` |
|---|---|---|---|---|---|---|---|---|---|
| 0 | 4 | 0x35 | `kFloorTrans` | `012e 0131 0000 1a35 05 00` | (302, 305) | 0 | **0x1A35 = 6709** | **5** | 0 |
| 0 | 5 | 0x36 | `kCeilingTrans` | `0006 0135 0000 ffff ff 00` | (6, 309) | 0 | **0xFFFF = −1** | 255 | 0 |
| 0 | 6 | 0x33 | `kMailboxLf` | `002f 0161 0000 19d1 02 00` | (47, 353) | 0 | **0x19D1 = 6609** | **2** | 0 |
| 1 | 2 | 0x33 | `kMailboxLf` | `0061 0068 0000 ffff ff 00` | (97, 104) | 0 | **0xFFFF = −1** | 255 | 0 |

Both predicted values appear **verbatim on disk**. And the links are semantically
self-consistent:

- Room 0's floor transport (`where` 6709 → suite 67 / floor 9−8 = 1 → **room 0 itself**,
  `who` = 5) targets **room 0 object 5**, which is the `kCeilingTrans` at (6, 309). Fall
  through the floor, drop out of the ceiling of the same room.
- Room 0's left mailbox (`where` 6609 → suite 66 / floor 1 → **room 1**, `who` = 2)
  targets **room 1 object 2**, which is `room 1`'s `kMailboxLf` at (97, 104).
- Both *destination* objects carry `where = −1`, `who = 255`: they are pure endpoints.
  So the link is **one-directional on disk** and `who = 0xFF` is the "unset" object index,
  matching `where = −1` as the "unset" room. `ExtractFloorSuite` is never called on them
  because of the `where != -1` guard.

`ExtractFloorSuite(6709)` → `suite = 6709/100 = 67`, `floor = 6709 % 100 − 8 = 1`.
Round-trip exact. `kMailboxLf` = 0x33, `kFloorTrans` = 0x35, `kCeilingTrans` = 0x36
(`GliderPRO/Headers/GliderDefines.h:362`, `:364`, `:365`).

**Consequence for a Go port:** the packed form breaks down for `suite > 327`
(`327*100 + 99` = 32799 fits in `int16`; `328*100` = 32800 does not) and for
`floor + 8 > 99`. Given the enforced ranges `suite ∈ [0, 127]` and `floor ∈ [−7, 56]`, the
real bounds are max packed `127*100 + 64` = 12764 and min `0*100 + 1` = 1 — both safe. But
a range check is still worth having, because the value comes from a file and nothing
validates `where` itself.

**Latent defect at `GliderPRO/Sources/Scrap.c:145-154`.** The two union arms are
**swapped**:

```c
					if (ObjectIsLinkSwitch(&thisRoom->objects[i]))
					{							// point to new room location
						thisRoom->objects[i].data.d.where = 
								(wasSuite * 100) + wasFloor + kNumUndergroundFloors;
					}
					else
					{							// point to new room location
						thisRoom->objects[i].data.e.where = 
								(wasSuite * 100) + wasFloor + kNumUndergroundFloors;
					}
```

A *switch* should write `data.e.where` and a *transport* `data.d.where`. It works only
because `transportType.where` and `switchType.where` are both `short` at the identical
union offset **8**, so the two branches are byte-identical. A Go port that models the arms
as distinct structs and dispatches on `what` will silently write the wrong field unless
this is noticed. Note also that this site **inlines** the packing formula
`(wasSuite * 100) + wasFloor + kNumUndergroundFloors` instead of calling
`MergeFloorSuite`, which is why the bias appears here as a literal addition.

### 22.7 `houseType` — the 866-byte header

`GliderPRO/Headers/GliderStructs.h:182-198`. Measured `offsetof(houseType, rooms)` =
**866**; author's comment `// total = 866 +` — **correct**.

| Off | Hex | Field | Type | Size | Meaning / observed |
|---|---|---|---|---|---|
| 0 | 0x000 | `version` | `short` | 2 | `kHouseVersion` **0x0200**. Rejected if `>= kNewHouseVersion` **0x0300** (`GliderPRO/Sources/HouseIO.c:395-400`). **All 22 shipped houses: 0x0200** — the `version < 0x0200` legacy branches are dead for bundled content |
| 2 | 0x002 | `unusedShort` | `short` | 2 | **never initialised** — see below |
| 4 | 0x004 | `timeStamp` | `long` | 4 | Mac epoch seconds with **bit 0 = house-locked flag** and bit 31 forced to 0 |
| 8 | 0x008 | `flags` | `long` | 4 | bitfield, see below |
| 12 | 0x00C | `initial` | `Point` | 4 | **pixel** position of the glider at game start; `{v, h}` |
| 16 | 0x010 | `banner` | `Str255` = `char[256]` | 256 | intro text, embedded `\r` line breaks |
| 272 | 0x110 | `trailer` | `Str255` = `char[256]` | 256 | victory text |
| 528 | 0x210 | `highScores` | `scoresType` | 292 | see §22.3 |
| 820 | 0x334 | `savedGame` | `gameType` | 40 | the only save-game field that is ever **written** (the external `.gliG` path is dead); nothing in 1.0.4 ever reads it back — see §22.9. Populated in 2 of 22 shipped houses, both with `version = 0x0100` |
| 860 | 0x35C | `hasGame` | `Boolean` | 1 | "is `savedGame` valid" — written in 3 places, **never read** (§22.9). Observed 1 for `ImagineHouse PRO II` and `Titanic`, 0 for the other 20 |
| 861 | 0x35D | `unusedBoolean` | `Boolean` | 1 | **never initialised** |
| 862 | 0x35E | `firstRoom` | `short` | 2 | room **index** to start in; −1 for an empty house |
| 864 | 0x360 | `nRooms` | `short` | 2 | number of `roomType` records that follow |
| 866 | 0x362 | `rooms[]` | `roomType` × `nRooms` | 348 × `nRooms` | |

`rooms[]` is declared as a **zero-length trailing array** (`roomType rooms[];`,
`GliderStructs.h:197`), which is why `sizeof(houseType)` is 866 and not 866 + 348. This
is load-bearing: `GliderPRO/Sources/HouseLegal.c:630-631` computes

```c
	countedRooms = (GetHandleSize((Handle)thisHouse) - sizeof(houseType)) / sizeof(roomType);
```

and overwrites `nRooms` if the two disagree. If a compiler gave `sizeof(houseType)` the
value 1214 (i.e. counted one `roomType`), `Empty House` would be recounted as
`(13046 − 1214)/348` = 34 instead of 35 and the last room would silently vanish. The
shipped file sizes prove the zero-length reading is the one that was compiled. Likewise
`GliderPRO/Sources/HouseLegal.c:765` grows/shrinks the handle to
`sizeof(houseType) + sizeof(roomType) * r` = 866 + 348 r.

**Total data-fork size = 866 + 348 × `nRooms`.** Verified on **all 22** shipped houses
(full table of sizes and `nRooms` in §22.11): 21 of them match to the byte, and the sole
exception is `Sampler`:

| House | `dataFork` | `nRooms` | 866 + 348·n | Δ |
|---|---|---|---|---|
| `Sampler` | 1564 | 2 | 1562 | **+2** |
| *(all other 21)* | — | 2…531 | — | **0** |

`Sampler` carries **two trailing bytes** (`0x01 0x01`) beyond the last room. A Go loader
must therefore (a) take `nRooms` from the header, not from the file size, and (b) tolerate
trailing slack. Note that `ValidateNumberOfRooms` would "correct" `Sampler`'s `nRooms`
from 2 to `(1564 − 866)/348` = 2 (integer division discards the 2 bytes), so no harm is
done there — but only by luck.

**`Demo House` is the hard-coded demo target.** With `COMPILEDEMO` defined, `ReadHouse`
asserts `byteCount == 16526L` (`GliderPRO/Sources/HouseIO.c:349`), `numberRooms == 45`
(`:382`), the house is **locked** (`:404`) and `firstRoom == 0` (`:412`). The measured
`Demo House` values are `dataFork = 16526`, `nRooms = 45`, `firstRoom = 0`,
`timeStamp = 0x2C53B041` (bit 0 = 1 → locked). All four assertions hold, and
866 + 348 × 45 = 16526 exactly. This is independent confirmation of the whole layout.

#### `timeStamp` and the lock bit

`GliderPRO/Sources/HouseIO.c:402`:

```c
	houseUnlocked = (((*thisHouse)->timeStamp & 0x00000001) == 0);
```

and on write, `WriteHouse` (`GliderPRO/Sources/HouseIO.c:448ff`):

```c
	GetDateTime(&timeStamp);
	timeStamp &= 0x7FFFFFFF;                 // clear bit 31
	if (changeLockStateOfHouse)
		houseUnlocked = !saveHouseLocked;
	if (houseUnlocked)  timeStamp &= 0x7FFFFFFE;   // clear bit 0
	else                timeStamp |= 0x00000001;   // set bit 0
	(*thisHouse)->timeStamp = (long)timeStamp;
	(*thisHouse)->version = wasHouseVersion;
```

So `timeStamp` is *not* a plain timestamp: **bit 0 is the "locked" (read-only, scores
count) flag and bit 31 is forced to 0**. The timestamp is only rewritten when
`fileDirty`. Observed across all 22 houses — **15 locked, 7 unlocked**, and bit 31 is
clear in every one:

| House | `timeStamp` | bit 0 | Locked? |
|---|---|---|---|
| `Art Museum` | **0x2C808C35** | 1 | **locked** |
| `CD Demo House` | **0x2C3E8559** | 1 | **locked** |
| `California or Bust!` | **0x2C7542B8** | 0 | unlocked |
| `Castle o' the Air` | **0x2C32F464** | 0 | unlocked |
| `Davis Station` | **0x2C53AEFD** | 1 | **locked** |
| `Demo House` | **0x2C53B041** | 1 | **locked** |
| `Empty House` | **0x2C3130EC** | 0 | unlocked |
| `Fun House` | **0x2C446C10** | 0 | unlocked |
| `Grand Prix` | **0x2C318A3D** | 1 | **locked** |
| `ImagineHouse PRO II` | **0x2C1DC93D** | 1 | **locked** |
| `In The Mirror` | **0x2C329401** | 1 | **locked** |
| `Land of Illusion` | **0x2C6BE354** | 0 | unlocked |
| `Leviathan` | **0x2C329E33** | 1 | **locked** |
| `Metropolis` | **0x2C34904B** | 1 | **locked** |
| `Nemo's Market` | **0x2C2C9CD1** | 1 | **locked** |
| `Rainbow's End` | **0x2C2C9DD1** | 1 | **locked** |
| `Sampler` | **0x3540BFE8** | 0 | unlocked |
| `Slumberland` | **0x2C53B149** | 1 | **locked** |
| `SpacePods` | **0x2C2CA019** | 1 | **locked** |
| `Teddy World` | **0x2C5EAB21** | 1 | **locked** |
| `The Asylum Pro` | **0x2BFD3EA1** | 1 | **locked** |
| `Titanic` | **0x2D0246FE** | 0 | unlocked |

A locked house is the "released, scores are official" state; the editor refuses to modify
it and `ReadyBackground` draws `"Nothing to show"` instead of the room
(`GliderPRO/Sources/Room.c:252-262`). `houseIsReadOnly` also forces
`houseUnlocked = false` (`GliderPRO/Sources/HouseIO.c:428-430`).

The house's `savedGame.timeStamp` is compared against the house `timeStamp` to detect a
save/house mismatch (`GliderPRO/Sources/SavedGames.c:92`,
`SavedGameMismatchError` at `:152`).

#### `flags`

`GliderPRO/Sources/HouseIO.c:416-418`:

```c
	wardBitSet        = (((*thisHouse)->flags & 0x00000001) == 0x00000001);
	phoneBitSet       = (((*thisHouse)->flags & 0x00000002) == 0x00000002);
	bannerStarCountOn = (((*thisHouse)->flags & 0x00000004) == 0x00000000);
```

| Bit | Mask | Global | Effect when the bit is **set** | Consumer |
|---|---|---|---|---|
| 0 | **0x00000001** | `wardBitSet` | empty rooms (`who == kRoomIsEmpty`) are painted **solid** instead of getting sky/ground art | `GliderPRO/Sources/RoomGraphics.c:195-207` |
| 1 | **0x00000002** | `phoneBitSet` | the telephone is **suppressed** — `HandleTelephone` runs its whole body under `if (!phoneBitSet)` | `GliderPRO/Sources/Play.c:748` |
| 2 | **0x00000004** | `bannerStarCountOn` | star counting in the banner is **off** (inverted test) | `GliderPRO/Sources/Banner.c:142` |
| 3-31 | — | unused | | |

**Two of the three bits read "backwards" from their names.**

* Bit 1 is named `phoneBitSet` but the editor checkbox that writes it is
  `kNoPhoneCheck` = item **14** of `DITL` 1001, whose literal label is **"No Phone"**
  (`GliderPRO/Sources/HouseInfo.c:19`, `:120`, `:296-297`). `HandleTelephone` guards on
  `!phoneBitSet`, so **`flags` bit 1 clear = the phone rings; bit 1 set = silent house.**
  A port that treats bit 1 as "enable phone" inverts the behaviour of all 22 shipped houses.
* Bit 2 is **inverted** in the loader itself: `bannerStarCountOn` is true when the bit is
  *clear* (`== 0x00000000`), so a house saved with `flags = 0` gets star counting enabled.
  A deliberate default-on-for-old-files choice, and very easy to get backwards.

Bit 0 has **no editor UI at all**; `InitializeEmptyHouse` forces `wardBitSet = false`
(`GliderPRO/Sources/House.c:142`) and no shipped house sets it, so it is effectively a
developer switch. It has nothing to do with an Easter-egg object.

Observed across all 22 houses, only three distinct values:

| `flags` | Houses | Decoded |
|---|---:|---|
| **0x00000000** | 14 | phone **rings**, star counting **on**, no ward |
| **0x00000002** | 7 | phone **suppressed**, star counting **on** |
| **0x00000006** | 1 (`Art Museum`) | phone **suppressed** and bit 2 set → star counting **off** |

The `0x00000002` houses are `CD Demo House`, `California or Bust!`, `Davis Station`,
`Land of Illusion`, `Nemo's Market`, `Rainbow's End`, `SpacePods`. **No shipped house sets
bit 0 (`wardBitSet`).** `Art Museum` is the only house that exercises the inverted bit 2,
so it is the one file that proves the inversion is real rather than a misreading.

**Writer quirk.** Clearing the phone bit uses the mask `0xFFFFDFFD`
(`GliderPRO/Sources/HouseInfo.c:272`), which is `~0x00002002` — it clears bit 1 *and*
bit 13. Setting it uses the clean `| 0x00000002` (`:270`). No shipped house has bit 13 set
so this never bites in 1.0.4, but a Go writer should use `&^ 0x00000002` and not reproduce
the stray bit-13 clear.

#### `initial` — a pixel position, not a room

`initial` is the glider's start rectangle origin **in pixels within the starting room**.
Proof: `GliderPRO/Sources/HouseLegal.c:59-66` clamps it against pixel constants:

```c
		if ((*thisHouse)->initial.h < 0)  (*thisHouse)->initial.h = 0;
		if ((*thisHouse)->initial.v < 0)  (*thisHouse)->initial.v = 0;
		if ((*thisHouse)->initial.h > (kRoomWide - kGliderWide))
			(*thisHouse)->initial.h = kRoomWide - kGliderWide;
		if ((*thisHouse)->initial.v > (kTileHigh - kGliderHigh))
			(*thisHouse)->initial.v = kTileHigh - kGliderHigh;
```

With `kRoomWide` = **512** (`GliderPRO/Headers/GliderDefines.h:499`), `kGliderWide` = **48**
(`:548`), `kTileHigh` = **322** (`:498`) and `kGliderHigh` = **20** (`:549`), the legal
range is **h ∈ [0, 464], v ∈ [0, 302]**. Observed values, all legal:

All 22 shipped houses, `{v, h}`, every one inside the clamp:

| House | `initial` | House | `initial` |
|---|---|---|---|
| `Art Museum` | `{56, 231}` | `Land of Illusion` | `{163, 245}` |
| `CD Demo House` | `{105, 223}` | `Leviathan` | `{44, 229}` |
| `California or Bust!` | `{62, 308}` | `Metropolis` | `{24, 245}` |
| `Castle o' the Air` | `{124, 239}` | `Nemo's Market` | `{51, 342}` |
| `Davis Station` | `{95, 189}` | `Rainbow's End` | `{200, 237}` |
| `Demo House` | `{107, 49}` | `Sampler` | `{85, 384}` |
| `Empty House` | `{64, 83}` | `Slumberland` | `{7, 361}` |
| `Fun House` | `{78, 42}` | `SpacePods` | `{72, 30}` |
| `Grand Prix` | `{58, 268}` | `Teddy World` | `{41, 362}` |
| `ImagineHouse PRO II` | `{35, 211}` | `The Asylum Pro` | `{27, 78}` |
| `In The Mirror` | `{50, 424}` | `Titanic` | `{97, 39}` |

The extremes are `v` = 7 (`Slumberland`) … 200 (`Rainbow's End`) and `h` = 30
(`SpacePods`) … 424 (`In The Mirror`) — well inside [0, 302] × [0, 464], so the clamp
itself is never observed to fire.

A brand-new house gets `initial = {v = 32, h = 32}`
(`GliderPRO/Sources/House.c:131-132`).

#### The uninitialised fields

`InitializeEmptyHouse` (`GliderPRO/Sources/House.c:108-161`) sets `version`, `firstRoom`
(**−1**), `timeStamp` (0), `flags` (0), `initial` (32, 32), `banner`, `trailer`,
`hasGame` (false), `nRooms` (0) and calls `ZeroHighScores()`. It **never writes
`unusedShort` or `unusedBoolean`**, and the allocation is `NewHandle`, not
`NewHandleClear` (`GliderPRO/Sources/House.c:116`). The observed garbage:

All 22 houses (only the non-zero cases are listed; the other 12 have `unusedShort = 0` and
the other 15 have `unusedBoolean = 0`):

| House | `unusedShort` | `unusedBoolean` |
|---|---|---|
| `Art Museum` | **−30082** (0x8A7E) | **255** |
| `California or Bust!` | **13107** (0x3333) | 0 |
| `Castle o' the Air` | **259** (0x0103) | **30** |
| `Davis Station` | **222** (0x00DE) | **37** |
| `Fun House` | **26228** (0x6674) | 0 |
| `Grand Prix` | **60** | **185** |
| `In The Mirror` | **147** (0x0093) | 0 |
| `Land of Illusion` | **259** (0x0103) | **30** |
| `Metropolis` | **196** | **14** |
| `Sampler` | 0 | **2** |
| `Titanic` | **2074** (0x081A) | 0 |

0x3333 is classic Mac uninitialised-heap filler; `unusedShort` is even observed
**negative** (`Art Museum`, −30082), so a Go reader must not treat it as unsigned either.
**A Go loader must skip these two fields entirely and never treat `unusedBoolean` as a
bool** — 2, 14, 30, 37, 185 and 255 are all observed and none of them is 0 or 1.

Two shipped houses — **`ImagineHouse PRO II` and `Titanic`** — have `hasGame = 1` with a
populated 40-byte `savedGame` block; the other 20 have `hasGame = 0`. In both cases
`savedGame.version` is **0x0100**, not `kSavedGameVersion` **0x0200**, so they were saved by
an earlier Glider version. The 40 bytes at offset 820 are therefore not guaranteed
meaningful *or* current-version even when `hasGame` is set; see §22.9 for why nothing reads
them anyway.

#### Load-time validation actually performed

In order, `GliderPRO/Sources/HouseIO.c`:

| Step | Check | On failure |
|---|---|---|
| 1 | `GetEOF` succeeds (`:341`) | `CheckFileError`, return false |
| 2 | `NewHandle(byteCount)` succeeds (`:356`) | `kYellowNoMemory` id 10 |
| 3 | `FSRead` whole file (`:372`) | `CheckFileError`, return false |
| 4 | `nRooms >= 1` **and** `byteCount != 0` (`:385`) | `kYellowNoRooms`, return false |
| 5 | `version < kNewHouseVersion` (`:395`) | `kYellowNewerVersion`, return false |
| 6 | `RealRoomNumberCount() != 0` (`:422`) | `noRoomAtAll = true` |

There is **no upper bound on `nRooms`** — grep confirms no `kMaxRooms` constant exists
anywhere in the source. `nRooms` is a `short`, so a hostile file can claim up to 32767
rooms; the read is driven by the actual EOF, so the handle is only as big as the file, and
`ValidateNumberOfRooms` (`GliderPRO/Sources/HouseLegal.c:621-640`) will later clamp
`nRooms` to `(handleSize − 866) / 348`. **But that runs only from
`CheckHouseForProblems`, i.e. in the editor.** In play mode a lying `nRooms` is trusted
and reads past the end of the handle. A Go port must bounds-check
`nRooms <= (len(data) - 866) / 348` at load.

Type/creator: `'gliH'` / `'ozm5'` — confirmed by BinHex header on all five houses.

### 22.8 `game2Type` — the external saved-game format (dead code)

`GliderPRO/Headers/GliderStructs.h:144-164`. Measured `offsetof(game2Type, savedData)` =
**110**. The author's comment `// total = 114` is **wrong** — it counted the trailing
zero-length `savedRoom savedData[]` as 4 bytes. The header is 110 bytes and the file is
`110 + 292 × nRooms` (`GliderPRO/Sources/SavedGames.c:56`:
`byteCount = sizeof(game2Type) + sizeof(savedRoom) * numRooms;`).

| Off | Field | Type | Size | Note |
|---|---|---|---|---|
| 0 | `house` | `FSSpec` | **70** | `{short vRefNum; long parID; Str63 name;}` = 2 + 4 + 64 |
| 70 | `version` | `short` | 2 | `kSavedGameVersion` **0x0200** |
| 72 | `wasStarsLeft` | `short` | 2 | |
| 74 | `timeStamp` | `long` | 4 | copied from the **house's** `timeStamp` (`SavedGames.c:92`) — the mismatch key |
| 78 | `where` | `Point` | 4 | |
| 82 | `score` | `long` | 4 | |
| 86 | `unusedLong` | `long` | 4 | 0 |
| 90 | `unusedLong2` | `long` | 4 | 0 |
| 94 | `energy` | `short` | 2 | |
| 96 | `bands` | `short` | 2 | |
| 98 | `roomNumber` | `short` | 2 | |
| 100 | `gliderState` | `short` | 2 | |
| 102 | `numGliders` | `short` | 2 | |
| 104 | `foil` | `short` | 2 | |
| 106 | `nRooms` | `short` | 2 | **replaces `gameType.unusedShort`** |
| 108 | `facing` | `Boolean` | 1 | |
| 109 | `showFoil` | `Boolean` | 1 | |
| 110 | `savedData[]` | `savedRoom` × `nRooms` | 292 × n | |

File type/creator would be `'gliG'` / `'ozm5'` (`GliderPRO/Sources/SavedGames.c:122`,
`:182`).

**This entire format is unreachable in 1.0.4.** `SaveGame2`'s body is commented out —
`/*` at `GliderPRO/Sources/SavedGames.c:33`, `*/` at `:147` — with the note
`// Add NavServices later.` at `:32`. `OpenSavedGame` is:

```c
Boolean OpenSavedGame (void)
{
return false;		// TEMP fix this iwth NavServices
/*
	... 125 lines ...
	*/
}
```

(`GliderPRO/Sources/SavedGames.c:167-296`; the unconditional `return false` is at
**`:169`**, the comment block spans `:170`-`:295`.) A Go port should implement the
in-house save (§22.9) and may skip `game2Type` entirely, or implement it as a new feature
— but must not expect any `.gliG` files to exist.

### 22.9 The live save mechanism: `houseType.savedGame`

> [verified: the **write** path below is live, but in 1.0.4 there is **no read path**.
> `hasGame` is assigned at `GliderPRO/Sources/House.c:139` and
> `GliderPRO/Sources/SavedGames.c:337`/`:341` and is **never tested anywhere**; the only
> code that reads `houseType.savedGame` is `QueryResumeGame`
> (`GliderPRO/Sources/Menu.c:710-759`, reading `.score` and `.numGliders` at `:727-728`),
> and `QueryResumeGame` is **never called** — its only other occurrence is its own
> prototype at `Menu.c:28`. So saving persists the whole house handle (which does restore
> every object's `state` and every room's `visited` flag the next time the file is loaded),
> but score / room / glider count are written and never read back. Two shipped houses
> (`ImagineHouse PRO II`, `Titanic`) still carry `hasGame = 1` blocks, both stamped
> `version = 0x0100`. A Go port should implement the resume the code *intended* rather than
> copying this dead end.]

`SaveGame(Boolean doSave)` — `GliderPRO/Sources/SavedGames.c:303-351`:

```
 1. if (twoPlayerGame) return;                            // :309-310 — no saving in 2P
 2. HLock(thisHouse); thisHousePtr = *thisHouse;
 3. if (doSave)
 4.     savedGame.version      = kSavedGameVersion;       // 0x0200
 5.     savedGame.wasStarsLeft = numStarsRemaining;
 6.     GetDateTime(&stamp); savedGame.timeStamp = (long)stamp;
 7.     savedGame.where.h = theGlider.dest.left;
 8.     savedGame.where.v = theGlider.dest.top;
 9.     savedGame.score        = theScore;
10.     savedGame.unusedLong   = 0L;  savedGame.unusedLong2 = 0L;
11.     savedGame.energy       = batteryTotal;
12.     savedGame.bands        = bandsTotal;
13.     savedGame.roomNumber   = thisRoomNumber;
14.     savedGame.gliderState  = theGlider.mode;
15.     savedGame.numGliders   = mortals;
16.     savedGame.foil         = foilTotal;
17.     savedGame.unusedShort  = 0;
18.     savedGame.facing       = theGlider.facing;
19.     savedGame.showFoil     = showFoil;
20.     thisHousePtr->hasGame  = true;
21. else
22.     thisHousePtr->hasGame  = false;
23. HSetState(thisHouse, wasState);
24. if (doSave)
25.     if (!WriteHouse(theMode == kEditMode)) YellowAlert(kYellowFailedWrite, 0);
```

Note that **`savedGame.timeStamp` here is the save time, not the house time** (line 6
calls `GetDateTime`), whereas the dead `SaveGame2` copied the *house's* `timeStamp`
(`SavedGames.c:92`). Only `SaveGame` runs, so the in-house saved game records when it was
saved. Also note **only room object state that lives in `roomType.objects[]` is saved**,
because the whole house handle is written back — there is no separate per-room delta as in
`game2Type`. Resuming therefore restores every object's `state` in every room, which is
what the `savedRoom` array was going to do explicitly.

### 22.10 The preferences file — 226 bytes, raw struct dump

`prefsInfo` is declared at `GliderPRO/Headers/Externs.h:231-267`, wrapped in
`#pragma options align=mac68k` (`Externs.h:231`) and `#pragma options align=reset`
(`Externs.h:269`). `GliderPRO/Headers/Externs.h:240` carries a commented-out
`// long encrypted, fakeLong;` — an abandoned obfuscation scheme.

Written verbatim with no serialisation layer, `GliderPRO/Sources/Prefs.c:127-129`:

```c
	byteCount = sizeof(*thePrefs);
	theErr = FSWrite(fileRefNum, &byteCount, thePrefs);
```

Measured layout — transcribe the struct into a throwaway C file with `#pragma pack(2)`,
`Str32` = `char[33]`, `Str15` = `char[16]`, `Str31` = `char[32]` and a 32-bit `long`, then
print `sizeof` and `offsetof`. `sizeof(prefsInfo)` = **226**:

| Off | Field | Type | Size |
|---|---|---|---|
| 0 | `wasDefaultName` | `Str32` (`char[33]`) | 33 |
| 33 | `wasLeftName` | `Str15` (`char[16]`) | 16 |
| 49 | `wasRightName` | `Str15` | 16 |
| 65 | `wasBattName` | `Str15` | 16 |
| 81 | `wasBandName` | `Str15` | 16 |
| 97 | `wasHighName` | `Str15` | 16 |
| 113 | `wasHighBanner` | `Str31` (`char[32]`) | 32 |
| *145* | *(1 pad byte — `Str31` ends at odd offset 145, next field is a `long`)* | | 1 |
| 146 | `wasLeftMap` | `long` | 4 |
| 150 | `wasRightMap` | `long` | 4 |
| 154 | `wasBattMap` | `long` | 4 |
| 158 | `wasBandMap` | `long` | 4 |
| 162 | `wasVolume` | `short` | 2 |
| 164 | `prefVersion` | `short` | 2 |
| 166 | `wasMaxFiles` | `short` | 2 |
| 168 | `wasEditH` | `short` | 2 |
| 170 | `wasEditV` | `short` | 2 |
| 172 | `wasMapH` | `short` | 2 |
| 174 | `wasMapV` | `short` | 2 |
| 176 | `wasMapWide` | `short` | 2 |
| 178 | `wasMapHigh` | `short` | 2 |
| 180 | `wasToolsH` | `short` | 2 |
| 182 | `wasToolsV` | `short` | 2 |
| 184 | `wasLinkH` | `short` | 2 |
| 186 | `wasLinkV` | `short` | 2 |
| 188 | `wasCoordH` | `short` | 2 |
| 190 | `wasCoordV` | `short` | 2 |
| 192 | `isMapLeft` | `short` | 2 |
| 194 | `isMapTop` | `short` | 2 |
| 196 | `wasNumNeighbors` | `short` | 2 |
| 198 | `wasDepthPref` | `short` | 2 |
| 200 | `wasToolGroup` | `short` | 2 |
| 202 | `smWarnings` | `short` | 2 |
| 204 | `wasFloor` | `short` | 2 |
| 206 | `wasSuite` | `short` | 2 |
| 208 | `wasZooms` | `Boolean` | 1 |
| 209 | `wasMusicOn` | `Boolean` | 1 |
| 210 | `wasAutoEdit` | `Boolean` | 1 |
| 211 | `wasDoColorFade` | `Boolean` | 1 |
| 212 | `wasMapOpen` | `Boolean` | 1 |
| 213 | `wasToolsOpen` | `Boolean` | 1 |
| 214 | `wasCoordOpen` | `Boolean` | 1 |
| 215 | `wasQuickTrans` | `Boolean` | 1 |
| 216 | `wasIdleMusic` | `Boolean` | 1 |
| 217 | `wasGameMusic` | `Boolean` | 1 |
| 218 | `wasEscPauseKey` | `Boolean` | 1 |
| 219 | `wasDoAutoDemo` | `Boolean` | 1 |
| 220 | `wasScreen2` | `Boolean` | 1 |
| 221 | `wasDoBackground` | `Boolean` | 1 |
| 222 | `wasHouseChecks` | `Boolean` | 1 |
| 223 | `wasPrettyMap` | `Boolean` | 1 |
| 224 | `wasBitchDialogs` | `Boolean` | 1 |
| *225* | *(1 pad byte — struct rounded up to even)* | | 1 |
| | **total** | | **226** |

The pad byte at offset 145 is the only interior padding, and it is there **because
`long` is 2-byte aligned under mac68k**: 145 is odd, so one byte is inserted. Under a
modern 4-byte-aligned ABI the padding would be 3 bytes and `wasLeftMap` would land at
148, making the file 228 bytes and every subsequent offset wrong. This is the single most
likely thing for a Go port to get wrong about the prefs file.

The four `long` map fields (`wasLeftMap`, `wasRightMap`, `wasBattMap`, `wasBandMap`) hold
**KeyMap bit offsets**, not virtual key codes — see §20.

Prefs file identity, `GliderPRO/Sources/Prefs.c:18-24`:

| Name | Value | Site |
|---|---|---|
| `kPrefCreatorType` | `'ozm5'` | `GliderPRO/Sources/Prefs.c:18` |
| `kPrefFileType` | `'gliP'` | `:19` |
| `kPrefFileName` | `"\pGlider Prefs"` | `:20` |
| `kDefaultPrefFName` | `"\pPreferences"` | `:21` |
| `kPrefsStringsID` | **160** | `:22` |
| `kNewPrefsAlertID` | **160** | `:23` |
| `kPrefsFNameIndex` | **1** | `:24` |
| `kPrefsVersion` | **0x0034** | `GliderPRO/Sources/Main.c:16` |

The file lives in the System `Preferences` folder, located via
`FindFolder(kOnSystemDisk, kPreferencesFolderType, kCreateFolder, &vRef, &dirID)` and
created with `FSpCreate(..., kPrefCreatorType, kPrefFileType, smSystemScript)`.
`LoadPrefs(short versionNeed)` deletes the file and falls back to defaults when
`thePrefs->prefVersion != versionNeed` or when the read returns `eofErr`
(`GliderPRO/Sources/Prefs.c:251-266`). So **bumping `kPrefsVersion` is the only migration
mechanism: there is none, prefs are simply discarded.**

A Go port should replace this with a versioned JSON/TOML file and must *not* attempt to
read a real classic `Glider Prefs` file unless it reproduces the 226-byte layout exactly.

### 22.11 House resource forks — measured inventories

Houses are dual-fork files: **level data in the data fork, custom art and sound in the
resource fork**. Decoded with `/tmp/wf-constants/binhex.py` and enumerated with a
resource-map parser (`/tmp/wf-constants/rsrc.py`).

All 22 shipped houses, fully enumerated (`nRooms` read from data-fork offset 864; every
data fork is exactly `866 + 348 × nRooms` bytes except `Sampler`, and every house is
`version` **0x0200**):

| House | `nRooms` | data fork | rsrc fork | Resource types present |
|---|---:|---:|---:|---|
| `Sampler` | 2 | 1564 | **286** | *none* — 286 bytes is an **empty** classic resource fork (256-byte header + 30-byte empty map) |
| `Empty House` | 35 | 13046 | 2670 | `ICN#`/`icl4`/`icl8`/`ics#`/`ics4`/`ics8` id **−16455** (custom Finder icon only) |
| `California or Bust!` | 16 | 6434 | 259542 | icons; `PICT` × 30; `snd ` × 3 |
| `Fun House` | 43 | 15830 | 662443 | icons; `PICT` × 26 |
| `Demo House` | 45 | 16526 | 491757 | icons; `PICT` × 19; **`bnds` × 10**; `snd ` × 1 |
| `Davis Station` | 65 | 23486 | 1871320 | icons; `PICT` × 54; `snd ` × 5 |
| `Castle o' the Air` | 85 | 30446 | 306364 | icons; `PICT` × 12; **`bnds` × 9** |
| `In The Mirror` | 97 | 34622 | 151870 | icons (`ICN#`/`icl4`/`icl8` × 2); `PICT` × 15; `snd ` × 2; `vers` × 2 |
| `Art Museum` | 109 | 38798 | 7476159 | icons; `PICT` × 76; `snd ` × 10 |
| `Nemo's Market` | 124 | 44018 | 590602 | icons; `PICT` × 73; `snd ` × 5 |
| `Metropolis` | 127 | 45062 | 946907 | icons; `PICT` × 39; `vers` × 1 |
| `The Asylum Pro` | 140 | 49586 | 494107 | icons; `PICT` × 17; **`bnds` × 2** |
| `Grand Prix` | 175 | 61766 | 1347764 | icons; `PICT` × 70; `snd ` × 3; `vers` × 2 |
| `CD Demo House` | 206 | 72554 | 1612342 | icons; `PICT` × 116; `snd ` × 10 |
| `Titanic` | 208 | 73250 | 845727 | icons (`ICN#` × 2); `PICT` × 48; `snd ` × 6; `vers` × 1 |
| `Rainbow's End` | 223 | 78470 | 316065 | icons; `PICT` × 17; **`bnds` × 6**; `snd ` × 1 |
| `ImagineHouse PRO II` | 279 | 97958 | 677770 | icons (`icl8` × 2); `PICT` × 45; **`bnds` × 14**; `snd ` × 2; `vers` × 2 |
| `Land of Illusion` | 303 | 106310 | 401793 | icons; `PICT` × 20; **`bnds` × 5** |
| `Slumberland` | 383 | 134150 | 1041112 | icons; `PICT` × 20; **`bnds` × 20** |
| `SpacePods` | 402 | 140762 | 636844 | icons; `PICT` × 49; `snd ` × 3 |
| `Leviathan` | 472 | 165122 | 1901282 | icons (`ICN#`/`icl8` × 2); `PICT` × 53; **`bnds` × 4**; `snd ` × 12; `vers` × 2 |
| `Teddy World` | 531 | 185654 | 3107348 | icons (all six × 2); `PICT` × 120; `vers` × 2 |

"icons" above means the six-type Finder icon family `ICN#`/`icl4`/`icl8`/`ics#`/`ics4`/`ics8`
at id −16455. Note the **`bnds`** column: 8 of the 22 houses carry `'bnds'` resources
(70 in total), which is why `GetOriginalBounding` is a live code path — see §22.5.

`In The Mirror` `PICT` inventory (id, size, resource name):

| ID | Size | Name | Role |
|---|---|---|---|
| 1991 | 2446 | — | custom art in the 1900s band |
| 1992 | 7824 | — | " |
| 1993 | 29310 | — | " |
| **3300** | 33150 | `Tall Room` | room background — exactly `kUserStructureRange` |
| **3301** | 17108 | `Illusion` | room background |
| 10002 | 2708 | `Coffee Cup` | custom object art |
| 10006 | 2706 | `Crow` | " |
| 10008 | 2898 | `Milk` | " |
| 10018 | 2666 | `Soup Can` | " |
| 10025 | 3782 | `Toilet Paper` | " |
| 10026 | 3794 | `Paper Towels` | " |
| 10034 | 2710 | `Nemo's Mug` | " |
| 10036 | 3186 | `Bagel Bag` | " |
| 10038 | 2678 | `Bird` | " |
| 10074 | 4576 | `Axe (wall mounted)` | " |

`In The Mirror` `snd ` inventory: id **3001** `Krusty Laugh` (11706 bytes), id **3002**
`Glass breaking` (13558 bytes, attrs **0x20** = `resPurgeable`).
`California or Bust!` `snd `: id **3001** `Glypha Bird` (7978), **3002**
`Glider 4.0 Bonus` (9342), **3003** `Clang Clang Clang` (18465, purgeable).

Three empirical conclusions:

1. **Custom room backgrounds are `PICT` IDs in [3000, 3800).** The editor validates
   exactly this range: `if ((longID >= 3000) && (longID < 3800) && PictIDExists(longID))`
   (`GliderPRO/Sources/RoomInfo.c:762`). `kUserBackground` = **3000**
   (`GliderPRO/Headers/GliderDefines.h:523`) and `kUserStructureRange` = **3300**
   (`:523`). `In The Mirror` room 0 uses `background = 3300` and supplies `PICT` 3300 in
   its own fork — a perfect match, and it is ≥ 3300 so the `bounds == 0` fallback would
   classify it as *not* a structure; the room instead carries `bounds = 45` with bit 5 set
   and *is* a structure.
2. **Custom object art is `PICT` 10000+.** `California or Bust!` room 0 `obj[0]` is
   `what = 0x6E` (**`kCustomPict`** = 110, `GliderPRO/Headers/GliderDefines.h:409`) with
   arm `g` `height = 0x2726 = 10022`, and `PICT` **10022** (3044 bytes) is present in the
   same file's resource fork. This confirms that for `kCustomPict` the
   `applianceType.height` field is repurposed as a **PICT resource ID**.
3. **Custom sounds are `snd ` 3001+.** Sound-trigger objects carry the `snd ` resource ID
   in `switchType.where`, and the IDs observed in the forks (3001, 3002, 3003) are
   contiguous from 3001.

`vers` 1 and 2 in `In The Mirror` are the standard Finder "Get Info" version resources,
not used by the game.

The application's own resource fork (dumped as Rez text in
`GliderPRO/Glider PRO.r`, 199843 LF lines, ~15 MB) is the source of every built-in
`PICT`, `snd `, `clut`, `ALRT`, `DLOG`, `DITL`, `STR#`, `MENU`, `acur`, `CURS`, `WDEF`
and `Mcmd` referenced by ID elsewhere in this document.

**Go port implications.** There is no resource fork on Linux or Windows and, on modern
macOS, resource forks are a legacy xattr. A Go port must:

- read the *data fork* of a house from a plain file (a `.binhex` or MacBinary/AppleDouble
  container may need unwrapping first);
- obtain the *resource fork* from wherever the container put it, then parse the classic
  resource map (offsets 0-15 of the fork header: data offset, map offset, data length,
  map length, all big-endian `uint32`) to reach `PICT`/`snd ` by type+ID;
- decode QuickDraw `PICT` v1/v2 opcodes (or pre-convert the whole corpus to PNG offline);
- decode `'snd '` format-1 resources by **skipping the 20-byte header** to the raw
  8-bit unsigned mono sample data (see §16).

### 22.12 Summary size table

| Struct | Declared | Measured `sizeof` | Author's comment | Agree? |
|---|---|---|---|---|
| `Point` | `Types.h` | 4 | — | — |
| `Rect` | `Types.h` | 8 | — | — |
| `FSSpec` | `Files.h` | 70 | `// 70` | yes |
| `blowerType` | `GliderStructs.h:11-19` | 10 | `// total = 10` | yes |
| `furnitureType` | `:21-25` | 10 | `// total = 10` | yes |
| `bonusType` | `:27-34` | 10 | `// total = 10` | yes |
| `transportType` | `:36-43` | 10 | `// total = 10` | yes |
| `switchType` | `:45-52` | 10 | `// total = 10` | yes |
| `lightType` | `:54-62` | 10 | `// total = 10` | yes |
| `applianceType` | `:64-72` | 10 | `// total = 10` | yes |
| `enemyType` | `:74-82` | 10 | `// total = 10` | yes |
| `clutterType` | `:84-88` | 10 | `// total = 10` | yes |
| `objectType` | `:90-105` | **12** | `// total = 12` | yes |
| `scoresType` | `:107-114` | **292** | `// total = 292` | yes |
| `gameType` | `:116-134` | **40** | `// total = 40` | yes |
| `savedRoom` | `:136-142` | **292** | `// total = 292` | yes |
| `game2Type` (header) | `:144-164` | **110** | `// total = 114` | **NO — comment is wrong by 4** |
| `roomType` | `:166-180` | **348** | `// total = 348` | yes |
| `houseType` (header) | `:182-198` | **866** | `// total = 866 +` | yes |
| `boundsType` | `:266-272` | 4 | — | — |
| `demoType` | `:334-339` | 6 | — | — |
| `prefsInfo` | `Externs.h:231-267` | **226** | — | — |

---

## 23. Runtime-only structs — sizes and offsets

`GliderPRO/Headers/GliderStructs.h:200-345`. None of these ever touches a file, so their
byte layout is not a compatibility requirement — but their **field sets** are the complete
mutable state of the simulation, and a Go port that omits a field will diverge. All sizes
measured with `#pragma pack(2)` and 32-bit `long` (`/tmp/wf-constants/csz/r.c`), with
`GWorldPtr` taken as 4 bytes as it would be on a 68k/PPC Mac.

| Struct | Declared | `sizeof` | Purpose |
|---|---|---|---|
| `gliderType` | `GliderStructs.h:200-216` | **110** | one player |
| `hotObject` | `:218-225` | **16** | an "active rect" — a region that reacts to the glider |
| `savedType` | `:227-233` | **20** | offscreen backing store for a torn-down object |
| `sparkleType` | `:235-239` | **10** | one sparkle animation |
| `flyingPtType` | `:241-249` | **28** | one flying points/score popup |
| `flameType` | `:251-256` | **20** | one candle/tiki/coal flame |
| `pendulumType` | `:258-264` | **26** | one clock pendulum or teddy-bear rocker |
| `boundsType` | `:266-272` | **4** | the never-shipped `'bnds'` resource (§22.5) |
| `bandType` | `:274-279` | **16** | one rubber band in flight |
| `linksType` | `:281-285` | **8** | one switch/transport link |
| `greaseType` | `:287-295` | **26** | one grease spill |
| `starType` | `:297-302` | **24** | one star / bonus item |
| `shredType` | `:304-308` | **10** | one shredder |
| `dynaType` | `:310-320` | **36** | one **dyna**mic object (enemy, ball, drip, fish, …) |
| `objDataType` | `:322-332` | **26** | one entry in the flattened `masterObjects` list |
| `demoType` | `:334-339` | **6** | one recorded demo keystroke |
| `retroLink` | `:341-345` | **4** | back-link for the link editor |

### 23.1 `gliderType` — 110 bytes, the player

| Off | Field | Type | Meaning |
|---|---|---|---|
| 0 | `src` | `Rect` | source rect in the glider sprite sheet for the current frame |
| 8 | `mask` | `Rect` | source rect in the 1-bit mask |
| 16 | `dest` | `Rect` | **the authoritative position**, 48 × 20 in play coordinates |
| 24 | `whole` | `Rect` | dirty rect: union of pre- and post-move `dest` (built by `MoveGlider`, §10.2) |
| 32 | `destShadow` | `Rect` | shadow position; **x tracks `dest`, y is pinned at `kShadowTop`** |
| 40 | `wholeShadow` | `Rect` | dirty rect for the shadow |
| 48 | `clip` | `Rect` | clip region for drawing |
| 56 | `enteredRect` | `Rect` | the rect the glider was in when it entered the room (used for `ignoreLeft`/`ignoreRight`) |
| 64 | `leftKey` | `long` | **KeyMap bit offset**, not a virtual key code (§20) |
| 68 | `rightKey` | `long` | " |
| 72 | `battKey` | `long` | " |
| 76 | `bandKey` | `long` | " |
| 80 | `hVel` | `short` | horizontal velocity, px/frame, clamped ±`kMaxHVel` 16 |
| 82 | `vVel` | `short` | vertical velocity, px/frame, **unclamped** |
| 84 | `wasHVel` | `short` | previous frame's `hVel` (used by collision resolution) |
| 86 | `wasVVel` | `short` | previous frame's `vVel` |
| 88 | `vDesiredVel` | `short` | target vVel; reset to `kGravity` **3** every frame |
| 90 | `hDesiredVel` | `short` | target hVel; reset to **0** every frame |
| 92 | `mode` | `short` | one of the 24 `kGlider*` modes (§9) |
| 94 | `frame` | `short` | animation frame / mode countdown |
| 96 | `wasMode` | `short` | mode before the current transition |
| 98 | `facing` | `Boolean` | `kFaceLeft` / `kFaceRight` |
| 99 | `tipped` | `Boolean` | banking: pressing against the facing direction |
| 100 | `sliding` | `Boolean` | on grease |
| 101 | `ignoreLeft` | `Boolean` | suppress the left-edge room transition until the glider has cleared `enteredRect` |
| 102 | `ignoreRight` | `Boolean` | same on the right |
| 103 | `fireHeld` | `Boolean` | band key was down last frame (edge detect, one band per press) |
| 104 | `which` | `Boolean` | **0 = player 1, 1 = player 2** — a `Boolean` used as a player index |
| 105 | `heldLeft` | `Boolean` | left key was down this frame |
| 106 | `heldRight` | `Boolean` | right key was down this frame |
| 107 | `dontDraw` | `Boolean` | suppress drawing (during transitions) |
| 108 | `ignoreGround` | `Boolean` | suppress floor collision |
| *109* | *(pad)* | | struct rounded to even |

Note **eight `Rect`s occupy the first 64 bytes** — over half the struct. `dest` at offset
16 is the only one that is authoritative; `src`, `mask`, `whole`, `wholeShadow` and `clip`
are all derived. A Go port can shrink this to `{X, Y int; HVel, VVel int; …}` plus a
render-time rect computation, but must keep `wasHVel`/`wasVVel` (read by collision code
*after* `MoveGlider` has already overwritten `hVel`/`vVel`).

`which` being a `Boolean` rather than a `short` is why `GetInput` tests
`thisGlider->which && !onePlayerLeft` (`GliderPRO/Sources/Input.c:283`) — it is being used
as "am I player 2".

### 23.2 `dynaType` — 36 bytes, every moving hazard

| Off | Field | Type | Meaning |
|---|---|---|---|
| 0 | `dest` | `Rect` | current position |
| 8 | `whole` | `Rect` | dirty rect |
| 16 | `hVel` | `short` | |
| 18 | `vVel` | `short` | |
| 20 | `type` | `short` | the object's `what` code |
| 22 | `count` | `short` | general-purpose counter (bounces left, etc.) |
| 24 | `frame` | `short` | animation frame |
| 26 | `timer` | `short` | countdown to the next state change |
| 28 | `position` | `short` | index into the object's patrol range |
| 30 | `room` | `short` | which room this dyna belongs to |
| 32 | `byte0` | `Byte` | copied from the object record's `byte0` |
| 33 | `byte1` | `Byte` | " |
| 34 | `moving` | `Boolean` | |
| 35 | `active` | `Boolean` | |

### 23.3 `objDataType` — 26 bytes, the flattened object list

The author's own field comments (`GliderPRO/Headers/GliderStructs.h:324-331`):

| Off | Field | Type | Author's comment |
|---|---|---|---|
| 0 | `roomNum` | `short` | `// room # object in (real number)` |
| 2 | `objectNum` | `short` | `// obj. # in house (real number)` |
| 4 | `roomLink` | `short` | `// room # object linked to (if any)` |
| 6 | `objectLink` | `short` | `// obj. # object linked to (if any)` |
| 8 | `localLink` | `short` | `// index in master list if exists` |
| 10 | `hotNum` | `short` | `// index into active rects (if any)` |
| 12 | `dynaNum` | `short` | `// index into dinahs (if any)` |
| 14 | `theObject` | `objectType` | `// actual object data` (12 bytes, §22.3) |

`masterObjects` is a `objDataPtr` array of at most `kMaxMasterObjects` = **216**
(`GliderPRO/Headers/GliderDefines.h:266`) entries, sized 216 × 26 = 5616 bytes. It is the
"loaded neighbourhood": all objects from the current room and its up-to-`wasNumNeighbors`
adjacent rooms, flattened with their cross-room links already resolved to indices.

### 23.4 `demoType` — 6 bytes

`GliderPRO/Headers/GliderStructs.h:334-339`: `{long frame; char key; char padding;}`.
`sizeof` measured = **6**: `long` at offset 0 (4 bytes), `key` at 4, `padding` at 5. The
explicit `padding` field exists precisely so the struct is 6 rather than relying on
implicit tail padding. `key` values are 0 = right, 1 = left, 2 = battery/helium,
3 = rubber band (§10.9).

### 23.5 `GliderVars.h` — the global variable manifest

`GliderPRO/Headers/GliderVars.h` is 59 lines and contains **zero `#define`s** — it is
purely a list of `extern` declarations, verified by grep. It is worth reading as a
*manifest of the shared mutable state* a Go port must place somewhere:

- **41 source-rect declarations** (`GliderPRO/Headers/GliderVars.h:11-40`), the tables
  filled in by `StructuresInit.c` / `StructuresInit2.c` and catalogued in §6.4. Note
  `extern Rect *srcRects;` at `:40` — the 144-entry heap array indexed by object `what`.
- **QuickTime**: `extern Movie theMovie; extern Rect movieRect; extern Boolean hasMovie,
  tvInRoom;` (`:42-44`). `#include <Movies.h>` at `:8`. A Go port needs a video decoder
  or must drop in-game TV playback.
- **The two players**: `extern gliderType theGlider, theGlider2;` (`:46`) — statically
  allocated, not an array, so two-player code is written out twice throughout.
- `extern objDataPtr masterObjects;` (`:47`) — see §23.3.
- **The offscreen worlds' bounds**: `workSrcRect`, `backSrcRect` (`:48-49`),
  `mainWindowRect`, `houseRect` (`:50`).
- **The house**: `extern houseHand thisHouse;` (`:51`) — a double-indirect
  `Handle` to the whole loaded file; `extern roomPtr thisRoom;` (`:52`) — a **separate
  heap copy** of the current room, allocated by
  `thisRoom = (roomPtr)NewPtr(sizeof(roomType));` (`GliderPRO/Sources/StructuresInit2.c:193`)
  and synchronised with the handle by `CopyThisRoomToRoom()` /
  `CopyRoomToThisRoom(n)`. **This copy-in/copy-out discipline is the single easiest thing
  to get wrong in a port**: edits go to `thisRoom` and are lost unless
  `CopyThisRoomToRoom()` runs before the room changes or the file is written.
- `extern WindowPtr mainWindow, coordWindow;` (`:53`).
- **Scalars**: `theScore` (`long`, `:54`), `playOriginH`, `playOriginV` (`:55`),
  `thisRoomNumber`, `theMode`, `batteryTotal`, `bandsTotal` (`:56`), `foilTotal`,
  `mortals`, `numMasterObjects`, `previousRoom` (`:57`), and the flags `fileDirty`,
  `gameDirty`, `showFoil`, `doZooms`, `isPlayMusicGame` (`:58`).

`batteryTotal` is signed and dual-purpose: **positive = battery charges, negative = helium
charges, 0 = empty** (§10.8, §15).
---

## 24. Fidelity landmines — every bare numeric literal in the physics/dynamics code

This section exists because the assignment asks for it explicitly, and because it is the
part of the catalog most likely to decide whether a Go port *feels* like Glider PRO. Every
value in §1–§23 has a name; the values below **do not**. They are typed straight into
expressions, so they are invisible to `grep -n '#define'`, invisible to a porter reading
the header files, and each one silently encodes a piece of the game's feel or its sprite
sheet geometry.

### 24.1 How this list was produced

Generated mechanically by `/tmp/wf-constants/lits.py` over the LF-converted sources, not
by hand. The extractor:

1. Splits each file on `\n` (after `tr '\r' '\n'`), so line numbers below are the
   CR→LF line numbers, same convention as everywhere else in this document.
2. Tracks the most recent `//----  Name` banner comment as the "function" column.
   *Caveat:* this is the **banner**, not the C declarator. Two banners in the corpus lie:
   `GliderPRO/Sources/Interactions.c:52` says `GliderHitSides` but the function declared
   at `:54` is `GliderHitTop` (see §10.4), and `GliderPRO/Sources/Player.c:1200` is
   literally `//----  xxxx` — an unfilled placeholder above `RemoveFoilFromGlider` at
   `GliderPRO/Sources/Player.c:1202`.
3. Strips `// …` trailing comments before scanning, so numbers that appear only in a
   comment are not reported.
4. Skips any line containing `#define` or `#include` — by construction this list is
   **only** literals that are *not* behind a named constant.
5. Discards the tokens `0` and `1` as noise (they are overwhelmingly `false`/`true`,
   array index 0, and loop initialisers). **Consequence: a handful of semantically
   meaningful `1`s are not in the tables below.** The ones that matter are called out
   separately in §24.12.
6. Reports everything else, including negative numbers and hex.

Regeneration command:

```
cd /tmp/wf-constants && python3 lits.py \
  Sources_Interactions.c Sources_Modes.c Sources_Dynamics.c Sources_Dynamics2.c \
  Sources_Dynamics3.c Sources_RubberBands.c Sources_Grease.c Sources_Input.c \
  Sources_Transit.c > lits_physics.txt
python3 lits.py Sources_Player.c > lit_Player.txt
```

Raw yield, by file:

| File | Lines with a non-trivial bare literal |
|---|---|
| `GliderPRO/Sources/Interactions.c` | 84 |
| `GliderPRO/Sources/Player.c` | 53 |
| `GliderPRO/Sources/Dynamics2.c` | 33 |
| `GliderPRO/Sources/Dynamics3.c` | 33 |
| `GliderPRO/Sources/Grease.c` | 31 |
| `GliderPRO/Sources/Modes.c` | 27 |
| `GliderPRO/Sources/RubberBands.c` | 19 |
| `GliderPRO/Sources/Dynamics.c` | 17 |
| `GliderPRO/Sources/Transit.c` | 17 |
| `GliderPRO/Sources/Input.c` | 11 |
| **Total** | **325** |

For contrast: the whole program has **1327** `#define`s (§1.5). The 325 lines below are
the part of the simulation that has *no* name at all.

### 24.2 `Interactions.c` — collision insets, roof geometry, rewards

`GliderPRO/Sources/Interactions.c` also declares seven file-local `#define`s at
`:13-19` that *are* named but are visible nowhere else in the program, so they behave like
landmines for anyone reading the headers only:

| Name | Value | file:line | Meaning |
|---|---:|---|---|
| `kFloorVentLift` | **−6** | `GliderPRO/Sources/Interactions.c:13` | `vDesiredVel` while over a floor vent (upward) |
| `kCeilingVentDrop` | **8** | `:14` | `vDesiredVel` while under a ceiling vent (downward) |
| `kFanStrength` | **12** | `:15` | `hDesiredVel` magnitude while in a fan's blast |
| `kBatterySupply` | **50** | `:16` | charges added per battery; author's comment: `// about 2 rooms worth of thrust` |
| `kHeliumSupply` | **150** | `:17` | charges added per helium balloon (3× a battery) |
| `kBandsSupply` | **8** | `:18` | rubber bands added per band pickup |
| `kFoilSupply` | **8** | `:19` | foil hit-points added per foil pickup |

Note `kFloorVentLift` (−6) is **double** `kGravity` (3, `GliderPRO/Sources/Player.c:13`)
in magnitude and of opposite sign, so a floor vent nets −3 px/frame of climb;
`kCeilingVentDrop` (8) is 2.67× gravity, so a ceiling vent is much stronger than a floor
vent. This asymmetry is deliberate and unnamed anywhere in the headers.

#### 24.2.1 Collision insets (the "5"s and the "2")

| line | function | literal | Meaning |
|---|---|---:|---|
| `Interactions.c:60` | `GliderHitTop` | **5** | `glideBounds.left = dest.left + 5` |
| `:61` | `GliderHitTop` | **5** | `glideBounds.top = dest.top + 5` |
| `:62` | `GliderHitTop` | **5** | `glideBounds.right = dest.right - 5` |
| `:63` | `GliderHitTop` | **5** | `glideBounds.bottom = dest.bottom - 5` |
| `:89` | `GliderHitTop` | **2** | `offset = 2 + glideBounds.right - theRect->left` — 2 px of extra separation pushed out of a side hit |
| `:91` | `GliderHitTop` | **2** | mirror case: `offset = 2 + glideBounds.left - theRect->right` |
| `:108` | `SectGlider` | **6** | `glideBounds.top += 6` when `mode == kGliderBurning`; **unconditional**, applied before the `scrutinize` inset |
| `:112` | `SectGlider` | **5** | `glideBounds.left += 5`, only `if (scrutinize)` |
| `:113` | `SectGlider` | **5** | `glideBounds.top += 5`, only `if (scrutinize)` |
| `:114` | `SectGlider` | **5** | `glideBounds.right -= 5`, only `if (scrutinize)` |
| `:115` | `SectGlider` | **5** | `glideBounds.bottom -= 5`, only `if (scrutinize)` |

The glider's `dest` is 48 × 20 (`kGliderWide` 48, `kGliderHigh` 20,
`GliderPRO/Headers/GliderDefines.h:548-549`). A 5-px inset on all four sides makes the
**effective collision box 38 × 10** — that is, 79% of the width and **50% of the height**
of the drawn sprite. This single number is why the glider can visually clip a shelf edge
without dying. See §10.4 for which of the four predicates applies where.

The burning `+6` at `:108` shrinks the box to 38 × 4 measured from `dest.top + 11` when
`scrutinize` is also true: a burning glider is almost all hitbox-free on top, which is
what lets it slip under things while on fire.

#### 24.2.2 Room-escape tile tests (`>> 6`, `< 8`, tile art indices 2/3/5/6)

Four functions repeat the identical block. `thisTiles[]` is the current room's
`tiles[8]` array copied at `GliderPRO/Sources/RoomGraphics.c:175`, i.e. **art-slice
indices**, not column numbers. `>> 6` is ÷ `kTileWide` (64,
`GliderPRO/Headers/GliderDefines.h:497`) as an arithmetic shift, and `< 8` is
`< kNumTiles` (8, `:496`) spelled out.

| line(s) | function | literals | Meaning |
|---|---|---:|---|
| `Interactions.c:200-201` | `CheckEscapeUpTwo` | **6, 6** | `leftTile = dest.left >> 6`, `rightTile = dest.right >> 6` |
| `:203-204` | `CheckEscapeUpTwo` | **8, 8** | bounds-check both tiles against `kNumTiles` as a bare `8` |
| `:206-209` | `CheckEscapeUpTwo` | **5, 6, 5, 6** | both tiles must be art slice **5 or 6** — the "hole in the ceiling" slices of the `kDirt` background |
| `:257-258` | `CheckEscapeUp` | **6, 6** | same |
| `:260-261` | `CheckEscapeUp` | **8, 8** | same |
| `:263-266` | `CheckEscapeUp` | **5, 6, 5, 6** | same |
| `:312-313` | `CheckEscapeDownTwo` | **6, 6** | same shift |
| `:315-316` | `CheckEscapeDownTwo` | **8, 8** | same bound |
| `:318-321` | `CheckEscapeDownTwo` | **2, 3, 2, 3** | both tiles must be art slice **2 or 3** — the "hole in the floor" slices of `kDirt` |
| `:391-392` | `CheckEscapeDown` | **6, 6** | same shift |
| `:394` | `CheckEscapeDown` | **8, 8** | same bound (written on one line here, two lines in the other three copies) |
| `:396-397` | `CheckEscapeDown` | **2, 3, 2, 3** | same |

The entire tile-test path is gated on `thisBackground == kDirt`
(`GliderPRO/Sources/Interactions.c:255`, and the three siblings). Every other background
that is not `topOpen`/`bottomOpen` just clamps: `vVel = kCeilingLimit - dest.top`
(`:275`) or `vVel = kFloorLimit - dest.bottom`. So **art slices 2, 3, 5, 6 of the
`kDirt` PICT are load-bearing game logic**, not decoration: 2 and 3 are the two halves of
a downward tunnel mouth, 5 and 6 the two halves of an upward one. A Go port must ship the
same dirt tile art *in the same slice order* or dirt rooms become impassable.

#### 24.2.3 Roof geometry — the 250 / 186 / 64 line equations

`CheckRoofCollision` (`GliderPRO/Sources/Interactions.c:450-505`) is the single densest
patch of unnamed geometry in the program. It is entered only from
`GliderPRO/Sources/Interactions.c:722-723`:

```c
else if ((thisBackground == kRoof) && (thisGlider->dest.bottom > kRoofLimit))
    CheckRoofCollision(thisGlider);
```

with `kRoofLimit` = **122** (`GliderPRO/Headers/GliderDefines.h:505`).

| line | literal | Meaning |
|---|---:|---|
| `Interactions.c:454` | **6** | `offset = (dest.left + kHalfGliderWide) >> 6` — which tile column the glider's *centre* is over (`kHalfGliderWide` = 24, `GliderDefines.h:550`) |
| `:455` | **7** | `offset <= 7`, i.e. `< kNumTiles`, written as `<= 7` |
| `:460` | **6** | `offset << 6` — tile column origin in px |
| `:461` | **250** | tile art **1**: collide when `localX > (250 - dest.bottom)` |
| `:468` | **2** | `tileOver == 2` |
| `:470` | **6** | `offset << 6` |
| `:471` | **186** | tile art **2**: collide when `localX > (186 - dest.bottom)` |
| `:478` | **5** | `tileOver == 5` |
| `:480` | **64, 6** | tile art **5**: mirrored, uses `64 - localX` |
| `:481` | **186** | tile art **5** intercept |
| `:488` | **6** | `tileOver == 6` |
| `:490` | **64, 6** | tile art **6**: mirrored |
| `:491` | **250** | tile art **6** intercept |

Solving each test for the roof surface `y` (with `localX = (dest.left + 24) - offset*64`,
range 0…63):

| Tile art | Predicate | Roof surface `y` as a function of `localX` | y range | Slope |
|---|---|---|---:|---|
| 1 | `localX > 250 - bottom` | `y = 250 - localX` | 250 → 187 | up-right, 45° |
| 2 | `localX > 186 - bottom` | `y = 186 - localX` | 186 → 123 | up-right, 45° |
| 5 | `64 - localX > 186 - bottom` | `y = 122 + localX` | 122 → 185 | down-right, 45° |
| 6 | `64 - localX > 250 - bottom` | `y = 186 + localX` | 186 → 249 | down-right, 45° |
| any other | *(unconditional `else`, `:497-503`)* | `y = kRoofLimit = 122` | 122 | flat |

Two independent checks that this reading is right:

1. The four segments **chain**: 250→187, then 186→123, ridge, then 122→185, then
   186→249. Each boundary drops by exactly 1 px, which is the expected off-by-one of a
   strict `>` between adjacent 64-px slices.
2. The apex of the chain is **y = 122**, which is *exactly* `kRoofLimit`
   (`GliderPRO/Headers/GliderDefines.h:505`) — the guard value that decides whether
   `CheckRoofCollision` is called at all. The two numbers were derived from the same art.

So a `kRoof` room is a gable roof whose left face is art slices 1 then 2 and whose right
face is 5 then 6, at 1 px of rise per px of run. Any other slice is flat roof deck at
y = 122. All five bodies do the same thing (`:463-465`, `:473-475`, `:483-485`,
`:493-495`, `:499-501`):

```c
thisGlider->vVel = kFloorLimit - thisGlider->dest.bottom;   // kFloorLimit = 312
StartGliderFadingOut(thisGlider);
PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
```

i.e. the glider is teleported to the floor line and fades into the house. Note the
sliding immunity: `(!thisGlider->sliding)` at `:455` — a glider on grease cannot fall
through a roof.

#### 24.2.4 `HandleRewards` — velocity bleed and the flying-points selector

| line(s) | literal | Meaning |
|---|---:|---|
| `Interactions.c:773` | **100, 2, 2** | `AddFlyingPoint(&bounds, 100, hVel/2, vVel/2)` for `kRedClock` |
| `:774-775` | **4, 4** | `hVel /= 4; vVel /= 4;` |
| `:789` | **300, 2, 2** | `kBlueClock` |
| `:790-791` | **4, 4** | " |
| `:805` | **500, 2, 2** | `kYellowClock` |
| `:806-807` | **4, 4** | " |
| `:822` | **1000, 2, 2** | `kCuckoo` |
| `:823-824` | **4, 4** | " |
| `:839-840` | **2, 2** | `kBands`: `hVel /= 2; vVel /= 2;` |
| `:858-859` | **2, 2** | `kBattery` |
| `:880-881` | **2, 2** | `kFoil` |
| `:908-909` | **2, 2** | `kFoil` (second arm) |
| `:925` | **2, 2** | `kInvisBonus`: `AddFlyingPoint(&bounds, points, hVel/2, vVel/2)` — `points` from `data.c.points` |
| `:926-927` | **4, 4** | " |
| `:964-965` | **2, 2** | `kHelium` |

Two distinct rules, both unnamed:

- **Scoring pickups** (clocks, bonus) divide both velocity components by **4**.
- **Supply pickups** (bands, battery, foil, helium) divide by **2**.

The flying-point sprite inherits **half** the glider's velocity at the moment of pickup,
so the popup drifts with the player.

**The `100`/`300`/`500`/`1000` at `:773`/`:789`/`:805`/`:822` are not point values** —
they duplicate `kRedClockPoints` 100, `kBlueClockPoints` 300, `kYellowClockPoints` 500,
`kCuckooClockPoints` 1000 (`GliderPRO/Headers/GliderDefines.h:537-540`), which are added
to `theScore` on the *following* lines. The literal passed to `AddFlyingPoint` is used
**only as a sprite-family selector** by a `switch` on the raw value
(`GliderPRO/Sources/DynamicMaps.c:222-247`):

| `points` argument | `flyingPoints[i].start` | `.stop` | DynamicMaps.c line |
|---:|---:|---:|---|
| 100 | 12 | 14 | `:223-226` |
| **250** | 9 | 11 | `:228-231` |
| 300 | 6 | 8 | `:233-236` |
| 500 | 3 | 5 | `:238-241` |
| *anything else* | 0 | 2 | `:243-246` |

Consequences a port must reproduce:

- `1000` falls into `default`, so slots 0–2 of `pointsSrc[]` are the "1000" graphic.
- **`250` is unreachable from any built-in object.** The only caller that can produce it
  is `kInvisBonus` at `Interactions.c:925` with a house author setting
  `data.c.points == 250`. The sprite family exists solely for that case.
- Any `kInvisBonus` value that is not exactly 100, 250, 300 or 500 renders as the
  **"1000"** graphic regardless of its true worth. A 50-point bonus visibly shows 1000.
- `AddFlyingPoint` **mutates its `Rect *` argument in place**
  (`GliderPRO/Sources/DynamicMaps.c:205-208` adds `playOriginH`/`playOriginV`). It is
  called with `&bounds`, a local copy, so this is harmless here — but a Go port that
  passes a pointer to `who->bounds` instead would corrupt the hot spot.

#### 24.2.5 Microwave kill mask

| line | literal | Meaning |
|---|---:|---|
| `Interactions.c:1172` | **0x0001** | bit 0 of `data.g.byte0` → zero `bandsTotal` |
| `:1178` | **0x0002** | bit 1 → zero `batteryTotal` |
| `:1184` | **0x0004** | bit 2 → zero `foilTotal` (and `StartGliderFoilLosing`) |

`kills = (short)masterObjects[whoLinked].theObject.data.g.byte0`
(`GliderPRO/Sources/Interactions.c:1171`). Bits 3–7 are never tested. The three effects
are independent (three separate `if`s, not `else if`), and a single flag `killed` gates
one `kMicrowavedSound` at `:1191-1192`. Guard: nothing happens unless
`data.g.state` is non-zero (`:1170`) and `who->stillOver` is false (`:1165-1166`) — so a
microwave fires **once on entry**, not every frame.

#### 24.2.6 Web damping

| line | literal | Meaning |
|---|---:|---|
| `Interactions.c:1750` | **3** | `hDist = ((webBounds->right - dest.right) + (webBounds->left - dest.left)) >> 3` |
| `:1752` | **3** | `vDist = ((webBounds->bottom - dest.bottom) + (webBounds->top - dest.top)) >> 3` |

Sum of the two edge errors, then `>> 3` — i.e. **(2 × centre error) / 8 = centre error / 4**
as an arithmetic shift (so it rounds toward −∞ for negative errors, unlike `/ 4`). This is
a proportional controller with gain ¼ that snaps the glider toward the spider web's centre
whenever `hDesiredVel != 0` and `evenFrame` (`:1754-1761`). `WebGlider` also carries a
function-local `#define kKillWebbedGlider 150` at `:1737`.

### 24.3 `Player.c` — sprite-sheet frame indices

Every literal in `GliderPRO/Sources/Player.c` is an index into `gliderSrc[]`, the glider
sprite-strip rect table built by `StructuresInit.c` (§6.4, §9). None is behind a
`#define`. `kLeftFadeOffset` *is* named (see §9) and is added to get the left-facing
half of the strip.

| line(s) | function | index | Meaning |
|---|---|---:|---|
| `Player.c:157-158` | `MoveGliderNormal` | **30** | banking/tipped-right frame |
| `:165-166` | `MoveGliderNormal` | **3** | tipped frame |
| `:170-171` | `MoveGliderNormal` | **2** | the canonical level-flight frame |
| `:179-180` | `MoveGliderNormal` | **29** | banking/tipped-left frame |
| `:206` | `MoveGliderBurning` | **3** | `if (frame > 3)` — burn animation is 4 frames (0…3) |
| `:211-212` | `MoveGliderBurning` | **25** | `gliderSrc[25 + frame]` — burning strip, right-facing base |
| `:216-217` | `MoveGliderBurning` | **21** | `gliderSrc[21 + frame]` — burning strip, other orientation |
| `:322-323` | `MoveGliderUpStairs` | **2** | level frame |
| `:336` | `MoveGliderUpStairs` | **29** | `vNotClipped = dest.bottom - 29` — stair clip height |
| `:389-390` | `FinishGliderUpStairs` | **2** | level frame |
| `:439-440` | `MoveGliderDownStairs` | **2** | level frame |
| `:665-666` | `MoveGliderDownDuct` | **2** | level frame |
| `:704` | `MoveGliderDownDuct` | **315** | `vNotClipped = 315 - dest.top` — duct clip line, 3 px below `kFloorLimit` 312 |
| `:762-763` | `MoveGliderUpDuct` | **2** | level frame |
| `:969-970` | `MoveGliderInMailLeft` | **2** | level frame |
| `:1060-1061` | `MoveGliderInMailRight` | **2** | level frame |
| `:1156-1158` | `DeckGliderInFoil` | **2** | `gliderSrc[(frame + 2) + kLeftFadeOffset]` |
| `:1163-1164` | `DeckGliderInFoil` | **2** | `gliderSrc[frame + 2]` |
| `:1173` | `MoveGliderFoilGoing` | **8** | `if (frame > 8)` — end of the foil-appear animation |
| `:1179` | `MoveGliderFoilGoing` | **5** | `if (frame < 5)` — midpoint switch |
| `:1183-1185` | `MoveGliderFoilGoing` | **10** | `gliderSrc[(10 - frame) + kLeftFadeOffset]` — **descending** index |
| `:1190-1191` | `MoveGliderFoilGoing` | **10** | `gliderSrc[10 - frame]` |
| `:1216-1218` | `RemoveFoilFromGlider` (banner `xxxx`) | **2** | `gliderSrc[(frame + 2) + kLeftFadeOffset]` |
| `:1223-1224` | `RemoveFoilFromGlider` | **2** | `gliderSrc[frame + 2]` |
| `:1233` | `MoveGliderFoilLosing` | **8** | `if (frame > 8)` |
| `:1237` | `MoveGliderFoilLosing` | **5** | `if (frame < 5)` |
| `:1241-1243` | `MoveGliderFoilLosing` | **10** | `gliderSrc[(10 - frame) + kLeftFadeOffset]` |
| `:1248-1249` | `MoveGliderFoilLosing` | **10** | `gliderSrc[10 - frame]` |
| `:1270-1271` | `MoveGliderShredding` | **2** | level frame |
| `:1498` | `OffAMortal` | **−1** | `if (mortals < -1)` — author's comment `// both players are now dead` |
| `:1541` | `OffAMortal` | **−1** | `else if ((mortals == -1) && (onePlayerLeft) && (!gameOver))` |

Notes for a porter:

- **`src` and `mask` are always assigned the same rect** in every one of these pairs.
  The mask strip is a separate 1-bit GWorld with the *same* geometry, so one rect indexes
  both. A Go port with premultiplied-alpha sprites needs one table, not two.
- **`gliderSrc[2]` is the neutral frame** used by ten different mode handlers. `3` and
  `30` / `29` are the two banked frames. `21 + frame` and `25 + frame` are the two 4-frame
  burning strips. `10 - frame` (descending!) drives both foil transitions.
- `mortals` is a signed count that goes **below** −1: `< -1` means both players dead,
  `== -1` means one dead and `onePlayerLeft`.
- `315` at `:704` and `29` at `:336` are pixel clip lines with no relation to any
  `#define`; both must be copied verbatim.

### 24.4 `Modes.c` — mode-entry sprite and geometry fixups

| line(s) | function | literal | Meaning |
|---|---|---:|---|
| `Modes.c:95` | `StartGliderFadingOut` | **20, 16** | `QOffsetRect(&tempBounds, playOriginH - 20, playOriginV - 16)` — the fade-out sparkle is drawn 20 px left and 16 px up of play origin |
| `:192-193` | `StartGliderMailingOut` | **2** | `gliderSrc[2]` |
| `:238` | `StartGliderDuctingDown` | **2** | `leftSought = bounds->left + ((RectWide(bounds) - kGliderWide) / 2)` — centre the glider in the duct |
| `:271` | `StartGliderDuctingUp` | **2** | same |
| `:343-344` | `FlagGliderNormal` | **2** | `gliderSrc[2]` |
| `:369` | `FlagGliderShredding` | **36** | `dest.left = bounds->left + 36` — shredder entry x offset |
| `:388-389` | `FlagGliderShredding` | **2** | `gliderSrc[2]` |
| `:400` | `FlagGliderShredding` | **3** | `frame = bounds->bottom - 3` — **`frame` is reused as a pixel y target**, not an animation index |
| `:419-420` | `FlagGliderBurning` | **25** | `gliderSrc[25]` — first frame of one burning strip |
| `:424-425` | `FlagGliderBurning` | **21** | `gliderSrc[21]` — first frame of the other |
| `:462` | `FlagGliderInLimbo` | **3** | `if ((sayIt) && (saidFollow < 3))` — the "follow me" hint is spoken at most 3 times per game |
| `:528-529` | `ReadyGliderForTripUpStairs` | **2** | `gliderSrc[2]` |
| `:593-594` | `StartGliderFoilGoing` | **10** | `gliderSrc[(10 - frame) + kLeftFadeOffset]` |
| `:598-599` | `StartGliderFoilGoing` | **10** | `gliderSrc[10 - frame]` |
| `:619-620` | `StartGliderFoilLosing` | **10** | `gliderSrc[(10 - frame) + kLeftFadeOffset]` |
| `:624-625` | `StartGliderFoilLosing` | **10** | `gliderSrc[10 - frame]` |
| `:638` | `TagGliderIdle` | **30** | `thisGlider->hVel = 30;` with the author's own comment `// used for 30 frame delay` |

`Modes.c:638` is the clearest example in the codebase of **field aliasing as a poor man's
union**: `hVel` (a velocity in px/frame) is loaded with 30 and counted down as a frame
timer while `mode == kGliderIdle`. A Go port that types `HVel` as a velocity and adds a
separate `IdleTimer` will not be bug-compatible unless it also suppresses the velocity
integration for that mode. The same trick appears at `Modes.c:400` (`frame` = a y
coordinate) and `Dynamics3.c:223` (`hVel` = a clip line — see §24.6).

### 24.5 `Dynamics.c` / `Dynamics2.c` — enemy timers and animation counts

`GliderPRO/Sources/Dynamics.c`:

| line | function | literal | Meaning |
|---|---|---:|---|
| `Dynamics.c:304` | `HandleSparkleObject` | **240, 60** | `timer = RandomInt(240) + 60` — next sparkle in 60…299 frames (≈2.0…9.9 s at 30.07 fps) |
| `:406` | `HandleMacPlus` | **30** | `else if (timer == 30)` — the Mac Plus's screen changes 30 frames before its cycle ends |
| `:492` | `HandleCoffee` | **200, 200** | `timer = 200 + RandomInt(200)` — 200…399 frames (≈6.7…13.3 s) |
| `:503` | `HandleCoffee` | **100** | `else if (timer == 100)` — steam starts 100 frames in |
| `:506` | `HandleCoffee` | **200, 200** | same reset |
| `:560` | `HandleOutlet` | **5** | `if ((timer % 5) == 0)` — the outlet sparks every 5th frame |
| `:613` | `HandleVCR` | **115** | `timer = 115` |
| `:615` | `HandleVCR` | **5** | `else if (timer == 5)` |
| `:626` | `HandleVCR` | **100** | `else if (timer == 100)` |
| `:629` | `HandleVCR` | **115** | reset |
| `:632` | `HandleVCR` | **101** | `else if (timer == 101)` |
| `:730` | `HandleMicrowave` | **16** | `dest.right = dest.left + 16` — the microwave's 3 zap columns are 16 px wide |
| `:735` | `HandleMicrowave` | **16** | `QOffsetRect(&dest, 16, 0)` — column 2 |
| `:740` | `HandleMicrowave` | **16** | column 3 |
| `:756` | `HandleMicrowave` | **16** | second pass, column 1 |
| `:761` | `HandleMicrowave` | **16** | column 2 |
| `:766` | `HandleMicrowave` | **16** | column 3 |

The VCR sequence is a **descending** countdown with events at 115 (reset), 101, 100 and 5.
Note that 115 and 101/100 are one frame apart, so the two branches at `:626` and `:632`
fire on consecutive frames. `Dynamics3.c:339` also seeds a `timer = 115` (§24.6), so 115
is the canonical VCR cycle length in two places with no shared name.

`GliderPRO/Sources/Dynamics2.c`:

| line | function | literal | Meaning |
|---|---|---:|---|
| `Dynamics2.c:41` | `HandleBalloon` | **6** | `if (frame >= 6)` |
| `:64` | `HandleBalloon` | **6** | `frame = 6` |
| `:80` | `HandleBalloon` | **8** | `if (frame >= 8)` — balloon strip is 8 frames, popping at 6…7 |
| `:81` | `HandleBalloon` | **6** | wrap back to 6 |
| `:99` | `HandleBalloon` | **−2** | `vVel = -2` — balloons rise 2 px/frame |
| `:143` | `HandleCopter` | **8** | `if (frame >= 8)` |
| `:164` | `HandleCopter` | **8** | `frame = 8` |
| `:184` | `HandleCopter` | **10** | `if (frame >= 10)` — copter strip is 10 frames, crashing at 8…9 |
| `:185` | `HandleCopter` | **8** | wrap back to 8 |
| `:201` | `HandleCopter` | **2** | `vVel = 2` — copters descend 2 px/frame |
| `:203` | `HandleCopter` | **−1** | `hVel = -1` — and drift 1 px/frame left |
| `:211` | `HandleCopter` | **32** | `dest.right = dest.left + 32` — copter sprite is 32 px wide |
| `:272` | `HandleDart` | **3** | `frame = 3` |
| `:307` | `HandleDart` | **2** | `vVel = 2` |
| `:318` | `HandleDart` | **2** | `frame = 2` |
| `:387` | `HandleBall` | **32** | `dest.top = dest.bottom - 32` — ball is 32 px tall |
| `:392` | `HandleBall` | **3, 4** | `vVel = -((vVel * 3) / 4)` — **coefficient of restitution = 0.75**, integer truncating |
| `:434` | `HandleDrip` | **9** | `frame = 9 - frame` — mirror the drip animation |
| `:460` | `HandleDrip` | **12** | `dest.bottom = dest.top + 12` — falling drip is 12 px tall |
| `:464` | `HandleDrip` | **3** | `frame = 3` |
| `:481` | `HandleDrip` | **6** | `if (timer == 6)` |
| `:483` | `HandleDrip` | **4** | `else if (timer == 4)` |
| `:485-486` | `HandleDrip` | **2, 2** | `else if (timer == 2)` → `frame = 2` |
| `:489` | `HandleDrip` | **3** | `VOffsetRect(&dest, 3)` — splash moves 3 px down |
| `:492` | `HandleDrip` | **4** | `frame = 4` |
| `:507` | `HandleFish` | **7** | `if ((vVel >= 0) && (frame < 7))` |
| `:534` | `HandleFish` | **16** | `dest.top = dest.bottom - 16` |
| `:536` | `HandleFish` | **2** | `whole.top -= 2` — dirty rect grown 2 px for the splash |
| `:558` | `HandleFish` | **0x0003** | `if ((timer & 0x0003) == 0x0003)` — advance the fish every 4th frame |
| `:561` | `HandleFish` | **3** | `if (frame > 3)` |
| `:563` | `HandleFish` | **2** | `if ((frame == 1) \|\| (frame == 2))` |
| `:583` | `HandleFish` | **4** | `frame = 4` |

`Dynamics2.c:392` is the only place in the whole program with a fractional physics
coefficient, and it is written as integer `* 3 / 4` — **truncating toward zero**, and
applied to a *negated* value, so `vVel = -((vVel*3)/4)`. For `vVel = 7` this gives
`-(21/4) = -5`; for `vVel = -7` it gives `-(-21/4) = -(-5) = 5` in C89 implementation-defined
truncation on 68k/MPW (which truncates toward zero). A Go port using `-vVel*3/4` gets the
same result because Go also truncates toward zero, but a port using floating point or
floor division will produce a subtly bouncier ball.

`Dynamics2.c:558` (`timer & 3`) and `Dynamics.c:560` (`timer % 5`) are the two different
sub-rate mechanisms in the codebase — mask vs modulo, with no shared helper.

### 24.6 `Dynamics3.c` — `AddDynamicObject` seeding

Every literal in `GliderPRO/Sources/Dynamics3.c` is inside the one giant `switch` in
`AddDynamicObject`, which converts a static `objectType` record into a live `dynaType`
(§11, §23.2). The recurring `(delay * 6) / kTicksPerFrame` idiom converts an author-set
"delay" in some coarse unit into frames; with `kTicksPerFrame` = 2
(`GliderPRO/Headers/GliderDefines.h:492`) that is `delay * 3`.

| line | literal | Meaning |
|---|---:|---|
| `Dynamics3.c:194` | **−1** | `return (-1)` — no free dyna slot |
| `:208` | **60, 15** | `timer = RandomInt(60) + 15` — initial stagger, 15…74 frames |
| `:223` | **2** | `hVel = where->top + 2` with author's comment `// hVel used as clip` |
| `:234` | **3** | `frame = (short)who->data.g.delay * 3` |
| `:248-249` | **10, 7** | `where->left + playOriginH + 10`, `where->top + playOriginV + 7` |
| `:268-269` | **17, 10** | `+ 17`, `+ 10` |
| `:288-289` | **32, 57** | `+ 32`, `+ 57` |
| `:296` | **200** | `timer = 200` |
| `:316` | **6** | `count = ((short)who->data.g.delay * 6) / kTicksPerFrame` |
| `:331-332` | **64, 6** | `+ 64`, `+ 6` |
| `:339` | **115** | `timer = 115` (matches `HandleVCR`, §24.5) |
| `:354-355` | **56, 20** | `+ 56`, `+ 20` |
| `:374-375` | **14, 13** | `+ 14`, `+ 13` |
| `:376` | **48** | `dest.right = dest.left + 48` |
| `:400` | **−2** | `vVel = -2` |
| `:401` | **6** | `count = ((short)who->data.h.delay * 6) / kTicksPerFrame` |
| `:422` | **−1** | `hVel = -1` |
| `:425` | **2** | `vVel = 2` |
| `:426` | **6** | `count = (delay * 6) / kTicksPerFrame` |
| `:452` | **2** | `frame = 2` |
| `:455` | **2** | `vVel = 2` |
| `:456` | **6** | `count = (delay * 6) / kTicksPerFrame` |
| `:504` | **6** | `count = (delay * 6) / kTicksPerFrame` |
| `:505` | **3** | `frame = 3` |
| `:519` | **10, 8** | `where->left + 10`, `where->top + 8` — **note: no `playOriginH`/`playOriginV`** here, unlike `:248`, `:268`, `:288`, `:331`, `:354`, `:374` |
| `:521` | **6** | `hVel = ((short)who->data.h.delay * 6) / kTicksPerFrame` — **`hVel` used as a frame count** |
| `:547` | **−1** | `return (-1)` |

Three things a Go port will get wrong here:

1. The **`(x, y)` sprite anchor offsets** (`10,7` · `17,10` · `32,57` · `64,6` · `56,20` ·
   `14,13` · `10,8`) are per-object-kind art alignment constants with no names. They are
   the offsets from an object's `topLeft` to where its dyna sprite is drawn.
2. `Dynamics3.c:519` omits `playOriginH`/`playOriginV` where its six siblings include
   them. Either an intentional coordinate-space difference for that object kind or a bug;
   either way it must be copied literally.
3. `hVel` at `:223` holds a **clip line** and at `:521` holds a **frame count**. Two
   more instances of the aliasing pattern from §24.4.

### 24.7 `RubberBands.c`

| line | function | literal | Meaning |
|---|---|---:|---|
| `RubberBands.c:23` | *(file scope)* | **3** | `Rect bandRects[3];` — the band sprite has exactly 3 animation rects |
| `:49` | `CheckBandCollision` | **16** | `dest.right = dest.left + 16` — reset band to 16 px wide after a hit |
| `:58` | `CheckBandCollision` | **16** | `dest.left = dest.right - 16` |
| `:98` | `CheckBandCollision` | **16** | " |
| `:110` | `CheckBandCollision` | **16** | " |
| `:146` | `CheckBandCollision` | **−1** | `bandHitLast = -1;` with the author's comment `// so make note of that for the next time` |
| `:165` | `CheckBandCollision` | **2** | `theGlider.hVel += (bands[who].hVel / 2)` — a band transfers **half** its horizontal velocity to player 1 |
| `:188` | `CheckBandCollision` | **2** | same for `theGlider2` |
| `:219` | `HandleBands` | **2** | `if (bands[i].mode > 2)` — modes 0…2 are the 3 `bandRects` |
| `:264` | `AddBand` | **−2** | `bands[numBands].vVel = -2` — a fired band rises 2 px/frame |
| `:267-268` | `AddBand` | **8, 8** | `dest.left = h - 8`, `dest.right = h + 8` — 16 px wide, centred on `h` |
| `:269-270` | `AddBand` | **3, 3** | `dest.top = v - 3`, `dest.bottom = v + 3` — **6 px tall**, centred on `v` |
| `:274-275` | `AddBand` | **32, 32** | if facing left, shift the whole rect **32 px left** |
| `:280-281` | `AddBand` | **32, 32** | if facing right, shift **32 px right** |
| `:285` | `AddBand` | **2** | `thisGlider->hVel -= (bands[numBands].hVel / 2)` — **recoil**: firing costs the glider half the band's velocity |

The band is spawned as a 16 × 6 rect centred on the point passed in by the caller
(`dest.left + 24`, `dest.top + 10`, i.e. the glider's centre — see §24.9), then offset
±32 px in the facing direction so it appears at the glider's nose rather than its middle.
`AddBand` sets `vVel = -2` unconditionally, so **every band arcs upward** as it travels.

Recoil (`:285`) and transfer (`:165`, `:188`) both use `/ 2` on a `short`, so C truncation
toward zero applies: an odd `hVel` loses the odd pixel. Momentum is *not* conserved.

### 24.8 `Grease.c` — 32 × 27 tiles and the 2-px-per-frame spread

The grease sprite strip is **4 frames of 32 × 27**, all unnamed:

| line | function | literal | Meaning |
|---|---|---:|---|
| `Grease.c:56` | `HandleGrease` | **3** | `if (grease[i].frame >= 3)` with comment `// grease completely tipped` |
| `:58` | `HandleGrease` | **3** | `grease[i].frame = 3` (clamp) |
| `:63` | `HandleGrease` | **−2, 2** | `QSetRect(&src, 0, -2, 2, 0)` — the 2 × 2 spread probe, rightward |
| `:65` | `HandleGrease` | **−2, −2** | `QSetRect(&src, -2, -2, 0, 0)` — leftward |
| `:71` | `HandleGrease` | **32, 27** | `QSetRect(&src, 0, 0, 32, 27)` — one grease frame |
| `:72` | `HandleGrease` | **27** | `QOffsetRect(&src, 0, grease[i].frame * 27)` — frames are stacked **vertically**, stride 27 |
| `:84` | `HandleGrease` | **2** | `QOffsetRect(&grease[i].dest, 2, 0)` — spread 2 px right per frame |
| `:86` | `HandleGrease` | **−2** | spread 2 px left |
| `:92` | `HandleGrease` | **−2, 2** | probe rect again |
| `:94` | `HandleGrease` | **2** | `grease[i].start += 2` |
| `:95` | `HandleGrease` | **2** | `hotSpots[grease[i].hotNum].bounds.right += 2` — **the hot spot grows with the spill** |
| `:99` | `HandleGrease` | **−2, −2** | probe rect, left |
| `:101` | `HandleGrease` | **2** | `grease[i].start -= 2` |
| `:102` | `HandleGrease` | **2** | `hotSpots[…].bounds.left -= 2` |
| `:146` | `BackupGrease` | **32, 27** | `QSetRect(&dest, 0, 0, 32, 27)` |
| `:147` | `BackupGrease` | **4** | `for (i = 0; i < 4; i++)` — 4 frames |
| `:159` | `BackupGrease` | **2** | `QOffsetRect(src, 2, 0)` |
| `:167` | `BackupGrease` | **−2** | `QOffsetRect(src, -2, 0)` |
| `:169` | `BackupGrease` | **27** | `QOffsetRect(&dest, 0, 27)` — next frame down |
| `:198` | `ReBackUpGrease` | **−1** | `return (-1)` |
| `:213` | `AddGrease` | **−1** | `return (-1)` — no free slot |
| `:215` | `AddGrease` | **32, 27** | one frame |
| `:218` | `AddGrease` | **32, 27, 4** | `QSetRect(&bounds, 0, 0, 32, 27 * 4)` — **the whole strip is 32 × 108** |
| `:220` | `AddGrease` | **−1** | `if (savedNum != -1)` |
| `:224` | `AddGrease` | **−8** | `QOffsetRect(&src, -8, 0)` — spill start, facing left |
| `:226` | `AddGrease` | **8** | facing right |
| `:232` | `AddGrease` | **−1** | `grease[numGrease].frame = -1` — sentinel: not yet tipped |
| `:236` | `AddGrease` | **4** | `grease[numGrease].start = src.right + 4` |
| `:242` | `AddGrease` | **4** | `grease[numGrease].start = src.left - 4` |
| `:250` | `AddGrease` | **−1** | `return (-1)` |
| `:284` | `RedrawAllGrease` | **2** | `((src.bottom - src.top) == 2) &&` — "is this the 2-px spread probe?" test |

Key derived facts:

- Grease tile = **32 × 27**; strip = 4 frames stacked vertically = **32 × 108**
  (`Grease.c:218`).
- Grease spreads **2 px per frame** in one direction, and the corresponding `hotObject`
  bounds edge is widened by the same 2 px on the same frame (`:95`, `:102`), so the
  slippery region always matches the drawn puddle exactly.
- `frame == -1` means "bottle tipped but not yet pouring"; `frame >= 3` means "fully
  poured" (`:56`, `:232`).
- `RedrawAllGrease` at `:284` identifies the spread strip by its **height being exactly
  2** — a shape test standing in for a type tag.

### 24.9 `Input.c`

| line | function | literal | Meaning |
|---|---|---:|---|
| `Input.c:82` | `DoPause` | **214, 54** | `QSetRect(&bounds, 0, 0, 214, 54)` — the "Paused" banner is 214 × 54 px |
| `:152` | `DoBatteryEngaged` | **4** | `if (batteryFrame >= 4)` — 4-frame thrust flicker |
| `:178` | `DoHeliumEngaged` | **4** | same for helium |
| `:242` | `GetDemoInput` | **2** | `case 2:` with comment `// battery key` |
| `:250` | `GetDemoInput` | **3** | `case 3:` with comment `// rubber band key` |
| `:253-254` | `GetDemoInput` | **24, 10** | `AddBand(thisGlider, dest.left + 24, dest.top + 10, facing)` |
| `:334` | `GetInput` | **2** | `LogDemoKey(2)` — battery |
| `:348` | `GetInput` | **3** | `LogDemoKey(3)` — band |
| `:352-353` | `GetInput` | **24, 10** | identical `AddBand` call |

`(+24, +10)` is exactly `(kHalfGliderWide, kGliderHigh / 2)` = the centre of the 48 × 20
glider — but it is spelled out as bare `24` and `10` in both the live-input and
demo-playback paths. `kHalfGliderWide` (24) exists at
`GliderPRO/Headers/GliderDefines.h:550` and is simply not used here.

The demo key codes 0/1/2/3 are bare integers in *both* directions
(`LogDemoKey(2)`/`LogDemoKey(3)` writing, `case 2:`/`case 3:` reading), with no shared
enum — see §10.9 and the swapped case-0/case-1 comments noted in §25(e).

### 24.10 `Transit.c` — room-entry placement

| line | function | literal | Meaning |
|---|---|---:|---|
| `Transit.c:23` | *(file scope)* | **9** | `extern short localNumbers[9];` — the 9-cell neighbourhood (self + 8 neighbours) |
| `:94` | `ReadyGliderFromTransit` | **64** | `clip.right -= 64` — stairs case: hide the rightmost tile |
| `:95` | `ReadyGliderFromTransit` | **25** | `clip.bottom -= 25` |
| `:99` | `ReadyGliderFromTransit` | **4** | `dest.bottom = clip.bottom - 4` |
| `:110` | `ReadyGliderFromTransit` | **79** | `clip.left += 79` — mirrored stairs case |
| `:111` | `ReadyGliderFromTransit` | **25** | `clip.bottom -= 25` |
| `:115` | `ReadyGliderFromTransit` | **4** | `dest.bottom = clip.bottom - 4` |
| `:174` | `MoveRoomToRoom` | **48, 20** | `QSetRect(&enterRect, 0, 0, 48, 20)` — glider-sized entry rect |
| `:176` | `MoveRoomToRoom` | **2** | `kGliderStartsDown + (short)thisRoom->leftStart - 2` |
| `:183` | `MoveRoomToRoom` | **48, 20** | same |
| `:185` | `MoveRoomToRoom` | **2** | same |
| `:206` | `MoveRoomToRoom` | **48, 20** | same |
| `:207` | `MoveRoomToRoom` | **48** | `QOffsetRect(&enterRect, kRoomWide - 48, …)` — right-edge entry |
| `:208` | `MoveRoomToRoom` | **2** | `kGliderStartsDown + (short)thisRoom->rightStart - 2` |
| `:215` | `MoveRoomToRoom` | **48, 20** | same |
| `:216` | `MoveRoomToRoom` | **48** | same |
| `:217` | `MoveRoomToRoom` | **2** | same |

`48` and `20` are `kGliderWide` / `kGliderHigh`
(`GliderPRO/Headers/GliderDefines.h:548-549`), spelled out four times each. `79` at
`:110` is **not** `64 + 15` of anything named; it is the left clip for the mirrored
staircase art. `−2` at `:176`/`:185`/`:208`/`:217` is the difference between the editor's
start-marker y and the play-mode entry y (see §25(j)):

```
entry y = kGliderStartsDown + leftStart - 2          (kGliderStartsDown = 32, GliderDefines.h:569)
editor marker y = kGliderStartsDown + leftStart      (ObjectEdit.c:549)
```

so a glider actually enters **2 px above** where the editor drew the marker. `leftStart`
and `rightStart` are `Byte`s clamped to 0…255 by the editor
(`GliderPRO/Sources/ObjectEdit.c:539-559`), and the explicit `(short)` casts at
`Transit.c:176` etc. exist because `Byte + short` would otherwise promote unhelpfully.

### 24.11 Cross-file duplicate literals worth naming in a port

These values appear as bare literals in more than one file with the same meaning. In each
case there is a real conceptual constant that was never declared:

| Value | Meaning | Bare occurrences |
|---:|---|---|
| **48** | glider width (`kGliderWide` exists!) | `Transit.c:174, 183, 206, 207, 215, 216`; `Dynamics3.c:376` |
| **20** | glider height (`kGliderHigh` exists!) | `Transit.c:174, 183, 206, 215` |
| **24** | half glider width (`kHalfGliderWide` exists!) | `Input.c:253, 352` |
| **5** | collision inset | `Interactions.c:60, 61, 62, 63, 112, 113, 114, 115` |
| **2** | glider sprite neutral frame | `Player.c:170, 322, 389, 439, 665, 762, 969, 1060, 1156, 1163, 1216, 1223, 1270`; `Modes.c:192, 343, 388, 528` |
| **10** | descending foil frame base | `Player.c:1183, 1190, 1241, 1248`; `Modes.c:593, 598, 619, 624` |
| **21 / 25** | the two burning strip bases | `Player.c:211, 216`; `Modes.c:419, 424` |
| **32, 27** | grease tile size | `Grease.c:71, 146, 215, 218` |
| **16** | band width | `RubberBands.c:49, 58, 98, 110` (and `8`+`8` at `:267-268`) |
| **115** | VCR cycle length | `Dynamics.c:613, 629`; `Dynamics3.c:339` |
| **6** | `>> 6` = ÷ `kTileWide` | `Interactions.c:200, 201, 257, 258, 312, 313, 391, 392, 454, 460, 470, 480, 490` |
| **8** | `< kNumTiles` | `Interactions.c:203, 204, 260, 261, 315, 316, 394` (and `<= 7` at `:455`) |
| **2** | `/ 2` velocity bleed on supply pickup | `Interactions.c:839, 858, 880, 908, 964` |
| **4** | `/ 4` velocity bleed on score pickup | `Interactions.c:774, 790, 806, 823, 926` |

### 24.12 Meaningful `1`s the extractor discarded

Because `lits.py` filters the tokens `0` and `1` (§24.1 step 5), the tables above omit
these. They were found by targeted grep and matter:

| file:line | Code | Why the `1` matters |
|---|---|---|
| `GliderPRO/Sources/Interactions.c:458` | `if (tileOver == 1)` | tile art **1** is the lowest roof slope (§24.2.3) |
| `GliderPRO/Sources/Dynamics2.c:563` | `if ((frame == 1) \|\| (frame == 2))` | fish frames 1 and 2 are the two splash frames |
| `GliderPRO/Sources/Room.c:871` | `leftThresh = (leftTile == 1) ? … ` | the `kDirt` `leftThresh`/`leftOpen` inconsistency, §25(q) |
| `GliderPRO/Sources/HouseIO.c:402` | `timeStamp & 0x00000001` | house lock bit |
| `GliderPRO/Sources/HouseIO.c:416-418` | `flags & 0x1 / 0x2 / 0x4` | ward / phone / banner-star-count bits (the third **inverted**) |
| `GliderPRO/Sources/RoomInfo.c:744` | `bounds & 1` | the `bounds` sentinel bit that distinguishes "encoded" from "never set" (§22.5) |

### 24.13 Ranked risk list

If a port reproduces only some of §24, reproduce these first — they change observable
gameplay rather than pixel placement:

1. **`Interactions.c:60-63` / `:112-115` — the 5-px inset.** Halves the glider's
   effective height. Getting this wrong makes the game feel either impossible or trivial.
2. **`Interactions.c:454-503` — the roof line equations** (250 / 186 / 64 / `>> 6`).
   A wrong sign or intercept makes `kRoof` rooms unplayable.
3. **`Interactions.c:774…965` — the `/ 2` vs `/ 4` velocity bleed.** Determines whether
   a pickup slows you to a stop or barely nudges you.
4. **`Interactions.c:12-14` — `kFloorVentLift −6` / `kCeilingVentDrop 8` /
   `kFanStrength 12`.** Named, but file-local, so easy to miss; they are the entire
   vertical-mobility budget of the game.
5. **`Interactions.c:1750-1752` — `>> 3` web damping** with arithmetic-shift rounding.
6. **`Dynamics2.c:392` — `-((vVel * 3) / 4)` ball restitution** with truncation toward
   zero.
7. **`RubberBands.c:264-285` — band spawn geometry (16 × 6, ±32 px), `vVel = -2`,
   half-velocity recoil and transfer.**
8. **`Transit.c:176/185/208/217` — the `- 2` entry offset.** Two pixels, but it decides
   whether a glider entering a room clips the floor.
9. **`Modes.c:638` / `Dynamics3.c:223` / `:521` / `Modes.c:400` — fields aliased as
   timers and clip lines.** A strongly typed Go port must decide, per mode, whether
   `HVel` is integrated.
10. **`Grease.c:95/:102` — hot-spot bounds growing in lockstep with the puddle.** Forget
    this and grease looks right but is not slippery where it is drawn.

---

## 25. Known defects and internal inconsistencies

Every item here was observed directly in the source; each is a place where a Go port has
to make a conscious choice between *bug-compatible* and *correct*. For a faithful port,
prefer bug-compatible: shipped houses were authored against this behaviour.

| # | Defect | Evidence | Effect / porting choice |
|---|---|---|---|
| a | **`srcRects[kFlower]` is never initialised.** `srcRects = (Rect *)NewPtr(sizeof(Rect) * kNumSrcRects);` — `NewPtr`, **not** `NewPtrClear` — allocates `kNumSrcRects` = `0x90` = **144** `Rect`s (1152 bytes), and `InitSrcRects` assigns only 116 of them. | `GliderPRO/Sources/StructuresInit2.c:271`; `kNumSrcRects` at `GliderPRO/Headers/GliderDefines.h:437` (`0x90`); §6.3 audit, §6.4 116-row table | Reading `srcRects[kFlower]` yields heap garbage. In practice `kFlower` is never drawn through `srcRects`. Go zero-initialises a `[]Rect`, so a port gets `{0,0,0,0}` — *different from the original* but strictly safer, and no shipped house exercises the path. |
| b | **`collided` is declared uninitialised** in `DidBandHitDynamic()` and would be returned unset if `numBands == 0` (the `for` body never runs). | `GliderPRO/Sources/Dynamics.c:80-105` (declaration at `:83`, `return (collided)` at `:104`) | Latent only: all three call sites guard with `(numBands > 0) && DidBandHitDynamic(who)` — `GliderPRO/Sources/Dynamics2.c:62`, `:162`, `:267`. Go zero-initialises, so the latent path is safe by construction; keep the `numBands > 0` guard anyway to match evaluation order (the C `&&` short-circuits, so `dinahs[who].dest` is not even read). |
| c | **`MoveGlider()` never clamps `vVel`.** `hVel` is clamped to ±`kMaxHVel` = **16** at `GliderPRO/Sources/Player.c:96-97` (`if (hVel < -kMaxHVel) hVel = -kMaxHVel;`) and `:113-114` (`if (hVel > kMaxHVel) hVel = kMaxHVel;`). There is **no `kMaxVVel` anywhere in the program** — `grep -n 'kMaxVVel'` over all headers and sources returns nothing. | `kMaxHVel` declared file-local at `GliderPRO/Sources/Player.c:16`; §10.2, §10.3 | Vertical terminal velocity is set only indirectly, by the floor/ceiling checks in `CheckGliderInRoom` firing before `vVel` can grow large, and by `vDesiredVel` being reset to `kGravity` = 3 every frame. Do **not** add a clamp. |
| d | **Banner/function-name mismatch:** `//----  GliderHitSides` sits above `Boolean GliderHitTop (…)`. | `GliderPRO/Sources/Interactions.c:52` vs `:54` | Both names are apt — the function returns "did it hit the top?" and handles the side case inline. Cosmetic only, but it misleads readers (it misled this document once). |
| e | **`GetDemoInput` case comments are swapped.** `GliderPRO/Sources/Input.c:228` says `case 0: // left key` but its body does `hDesiredVel += kNormalThrust; heldRight = true;`, and `:235` says `case 1: // right key` but does `hDesiredVel -= kNormalThrust; heldLeft = true;`. The **writer** settles it: `LogDemoKey(0)` is inside the *right*-key branch (`GliderPRO/Sources/Input.c:300-304`) and `LogDemoKey(1)` inside the *left*-key branch (`:319-321`). | `GliderPRO/Sources/Input.c:228`, `:235`, `:304`, `:321`; §10.9 | Demo key codes are **0 = right, 1 = left, 2 = battery/helium, 3 = band**. The code is right and only the two comments are wrong. Note `LogDemoKey` calls are all inside `#ifdef CREATEDEMODATA`, so recording is not compiled into the shipping build. |
| f | **20 of the 28 `k*MaskID` constants are dead.** `GliderPRO/Sources/ObjectDraw2.c:39-66` declares 28 file-local mask PICT IDs (3900…3927); only 8 are ever referenced. Dead: `kBBQMaskID` 3900 (`:39`), `kUpStairsMaskID` 3901 (`:40`), `kTrunkMaskID` 3902 (`:41`), `kDoorInLeftMaskID` 3905 (`:44`), `kDoorInRightMaskID` 3906 (`:45`), `kWindowInLeftMaskID` 3907 (`:46`), `kWindowInRightMaskID` 3908 (`:47`), `kHipLampMaskID` 3909 (`:48`), `kDecoLampMaskID` 3910 (`:49`), `kGuitarMaskID` 3911 (`:50`), `kFireplaceMaskID` 3916 (`:55`), `kBearMaskID` 3917 (`:56`), `kVase1MaskID` 3918 (`:57`), `kVase2MaskID` 3919 (`:58`), `kManholeMaskID` 3920 (`:59`), `kBooksMaskID` 3922 (`:60`), `kRugMaskID` 3923 (`:62`), `kChimesMaskID` 3924 (`:63`), `kCinderMaskID` 3925 (`:64`), `kFlowerBoxMaskID` 3926 (`:65`). Live: `kMailboxLeftMaskID` 3904 → `GliderPRO/Sources/ObjectDraw2.c:179`, `kMailboxRightMaskID` 3903 → `:262`, `kTVMaskID` 3912 → `:660`, `kVCRMaskID` 3913 → `:754`, `kStereoMaskID` 3914 → `:809`, `kMicrowaveMaskID` 3915 → `:864`, `kCobwebMaskID` 3927 → `:1269`, `kCloudMaskID` 3921 → `:1274`. | `GliderPRO/Sources/ObjectDraw2.c:39-66`; §18. **Verified against the resource fork:** `grep -ao "data 'PICT' ([0-9]*)" "GliderPRO/Glider PRO.r"` yields 152 PICTs, of which the only IDs in 3890…3935 are **3903, 3904, 3912, 3913, 3914, 3915, 3921, 3927** — exactly the 8 live ones, and none of the 20 dead ones. | The 20 dead constants name PICT resources that **do not exist**; the art was never drawn (or was folded into the object's colour PICT). Do not create Go assets for them. Note `kBooksMaskID` 3922 and `kCloudMaskID` 3921 are declared **out of numeric order** (`:60` and `:61`). |
| g | **The glider's shadow never moves vertically.** `destShadow` is set exactly once per glider, by `QSetRect(&destShadow, 0, 0, kGliderWide, kShadowHigh); QOffsetRect(&destShadow, dest.left, kShadowTop);` — so it occupies y = **306…315** forever (`kShadowTop` 306, `kShadowHigh` 9). Thereafter `destShadow.top` is never assigned anywhere in the program; it is only *read* to recompute `.bottom`. | `GliderPRO/Sources/Play.c:354-355`; `GliderPRO/Headers/GliderDefines.h:552-553`; the only other writes are `.left`/`.right` (`GliderPRO/Sources/Player.c:107-108`, `:124-125`, `:403`, `GliderPRO/Sources/Modes.c:382`, `:541-542`, `:572-573`, `GliderPRO/Sources/Transit.c:84-85`, `:101-102`, `:117-118`, `:132-133`) and the derived `.bottom` (`GliderPRO/Sources/Modes.c:314`, `:339`, `:384`, `:415`) | Correct behaviour to replicate: the shadow lies on a fixed floor plane at y = 306, not under the glider. The strip is 48 × 18 holding `kNumShadowSrcRects` = **2** frames of 48 × 9 (`GliderPRO/Headers/GliderDefines.h:559`, `GliderPRO/Sources/StructuresInit.c:207`, `:216-219`). |
| h | **Three writes to the global `evenFrame` from object code.** `evenFrame` is declared at `GliderPRO/Sources/Play.c:53` and is meant to be toggled only by `evenFrame = !evenFrame;` at `GliderPRO/Sources/Play.c:435` (once per frame) after `evenFrame = false;` at `GliderPRO/Sources/InterfaceInit.c:131`. Three other places force it `true`. Two of the three (`Dynamics3.c:474`, `:524`) are provably **dead writes**: they sit immediately above the "reverse engineer init. vel." triangular loop, which toggles the *local* `lilFrame`, never `evenFrame` — a copy-paste leftover from when `evenFrame` was the loop's toggle. | `GliderPRO/Sources/Dynamics2.c:420` (`HandleBall` launch), `GliderPRO/Sources/Dynamics3.c:474` (`kBall` seeding), `GliderPRO/Sources/Dynamics3.c:524` (`kFish` seeding); `extern` declarations at `Sources_Dynamics2.c:24`, `Sources_Dynamics3.c:23`, `Sources_Interactions.c:47` | `evenFrame` gates the web snap (`GliderPRO/Sources/Interactions.c:1755`) and the band-animation sub-rate, so spawning a ball or fish can put unrelated physics one frame out of phase. Bug-compatible ports must keep all three writes. |
| i | **`rightStartGliderDest` is built at x = 0** in one place and at `kRoomWide - 48` (= 464) in two others. `GliderPRO/Sources/ObjectEdit.c:558` reads `QOffsetRect(&rightStartGliderDest, 0, kGliderStartsDown + (short)thisRoom->rightStart);` while `:1517` and `:2066` read `QOffsetRect(&rightStartGliderDest, kRoomWide - 48, …)`. | `GliderPRO/Sources/ObjectEdit.c:558` vs `:1517` and `:2066` | While the user is *dragging* the right-hand start marker it jumps to the left edge of the room; it snaps back to the right edge on the next full redraw. Cosmetic editor bug; a port should use `kRoomWide - 48` in all three. |
| j | **The editor's start marker is 2 px lower and 4 px shorter** than the rect the glider actually enters in. Editor: `QSetRect(&leftStartGliderDest, 0, 0, 48, **16**)` then `QOffsetRect(…, 0, kGliderStartsDown + leftStart)`. Play: `QSetRect(&enterRect, 0, 0, 48, **20**)` then `QOffsetRect(…, 0, kGliderStartsDown + leftStart **- 2**)`. | `GliderPRO/Sources/ObjectEdit.c:545-547` and `:2061-2063` vs `GliderPRO/Sources/Transit.c:174-176` | 48 × 16 at *y* vs 48 × 20 at *y* − 2. House authors placing starts by eye were compensating for this. Keep both numbers. |
| k | **Two ID collisions in `RoomInfo.c`'s item block, in two *different* namespaces.** (1) `kBoundsButton` = **19** is a DITL item of `kRoomInfoDialogID` **1003** (`MyEnableControl(roomInfoDialog, kBoundsButton)` at `GliderPRO/Sources/RoomInfo.c:456`, `:458`, `:528`, `:533`, and `item == kBoundsButton` at `:548`), while `kOriginalArtworkItem` = **19** is a **pop-up menu item index** in `backgroundsMenu` (`SetPopUpMenuValue(theDialog, kRoomPopupItem, kOriginalArtworkItem)` at `:71`, `:436`; `EnableMenuItem(backgroundsMenu, kOriginalArtworkItem)` at `:400`; `if (was >= kOriginalArtworkItem)` at `:726`). (2) `kRoomDividerLine` = **12** is a DITL item of dialog 1003 (`FrameDialogItemC` at `:111`), while `kFloorSupportCheck` = **12** is a DITL item of `kOriginalArtDialogID` **1016** (`:751`, `:814`, `:817`). | `GliderPRO/Sources/RoomInfo.c:25`, `:30`, `:31`, `:33` (all four `#define`s within 9 lines of each other); §3.3, §21 | Not a runtime bug — the four constants are never used in the same namespace. But they sit in one flat block of 15 `#define`s with no naming discipline, so **a Go port that collapses them into a single `const` block or a single item-ID enum will produce duplicate-value bugs.** Keep DITL items of dialog 1003, DITL items of dialog 1016, and `backgroundsMenu` indices in three separate types. |
| l | **`MergeFloorSuite` is deliberately *not* the inverse of `ExtractFloorSuite`.** `short MergeFloorSuite (short floor, short suite) { return ((suite * 100) + floor); }` — **no `+ kNumUndergroundFloors`**. `ExtractFloorSuite` *does* un-bias: for `version >= 0x0200` it computes `*suite = combo / 100; *floor = (combo % 100) - kNumUndergroundFloors;` (and for `version < 0x0200`, the legacy layout, `*floor = (combo / 100) - kNumUndergroundFloors; *suite = combo % 100;`). Every caller of `Merge` must therefore add the bias itself: `floor += kNumUndergroundFloors;` at `GliderPRO/Sources/Link.c:277` and `GliderPRO/Sources/House.c:788`, `:804`. A third call site skips the helper entirely and inlines `(wasSuite * 100) + wasFloor + kNumUndergroundFloors`. | `GliderPRO/Sources/Link.c:34-36` (Merge), `:41-53` (Extract), `GliderPRO/Sources/Link.c:277`, `GliderPRO/Sources/House.c:788`, `:804`, `GliderPRO/Sources/Scrap.c:146-147`, `:152-153`; prototypes at `GliderPRO/Headers/GliderProtos.h:139-140`; §22.6 | Three different spellings of one encoding, with the ±8 bias living outside the helper. In Go, make `Merge` and `Extract` a true bijection that owns the bias, and route all four sites through it. Note the version check in `Extract` means **`floor` and `suite` swap positions between house versions 0x0100 and 0x0200** — `House.c:760-813` is the upgrader that rewrites them, and it also stamps `(*thisHouse)->version = kHouseVersion;` at `:813`. |
| m | **Double foil charge on a side hit.** `GliderHitTop` already handles the side case internally — `PlayPrioritySound(kFoilHitSound, …); foilTotal--; if (foilTotal <= 0) StartGliderFoilLosing(thisGlider);` at `GliderPRO/Sources/Interactions.c:81-84`. Its only caller, the `kDissolveIt` arm of `HandleHotSpotCollision`, then does the *same thing again* in its `else` branch: `if (foilTotal > 0) { foilTotal--; if (foilTotal <= 0) StartGliderFoilLosing(thisGlider); }` at `:1230-1235`. | `GliderPRO/Sources/Interactions.c:82` (first decrement) and `:1232` (second); call at `:1223`; `kFoilSupply` = 8 at `GliderPRO/Sources/Interactions.c:19` | Hitting a `kDissolveIt` object **from the side** costs **2** foil charges, not 1 — except on the last charge, where the `if (foilTotal > 0)` guard at `:1230` sees the already-zeroed total and skips. So a full 8-charge foil survives 4 side hits, then a 5th. Directly observable in play; a faithful port keeps both decrements. |
| n | **Union-arm swap in the clipboard code.** `GliderPRO/Sources/Scrap.c:145-154` reads `if (ObjectIsLinkSwitch(&thisRoom->objects[i])) { …data.d.where = … } else { …data.e.where = … }` — **backwards**. The authoritative reader `GetRoomLinked` (`GliderPRO/Sources/Objects.c`) uses `data.d.where` for the six **transports** (`kMailboxLf`, `kMailboxRt`, `kFloorTrans`, `kCeilingTrans`, `kInvisTrans`, `kDeluxeTrans`) and `data.e.where` for the eight **switches** (`kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch`, `kInvisSwitch`, `kTrigger`, `kLgTrigger`); `DoLink` (`GliderPRO/Sources/Link.c:279-292`) and the upgrader (`GliderPRO/Sources/House.c:779-806`) both agree with `GetRoomLinked`. Only `Scrap.c` is inverted. | `GliderPRO/Sources/Scrap.c:145-154` vs `GliderPRO/Sources/Objects.c` `GetRoomLinked`, `GliderPRO/Sources/Objects.c:219-233` (`ObjectIsLinkTransport`), `:236-251` (`ObjectIsLinkSwitch`), `GliderPRO/Sources/Link.c:279-292` | Harmless in C **only** because `data.d.where` and `data.e.where` both live at offset 8 of `objectType` and are both `short` (§22.2), so the write lands in the right place regardless. **A Go port with distinct struct types per union arm will silently write the wrong field and break every intra-room link on paste.** Either keep a single flat object record with a `Where int16` field, or fix the branch. |
| o | **The author's own `sizeof` comment is wrong.** `game2Type` is annotated `// total = 114`; the real header is **110** bytes. | `GliderPRO/Headers/GliderStructs.h` `game2Type`; measured with `/tmp/wf-constants/csz/g2.c`; §22.8 | The author counted the zero-length `savedData[]` as 4 bytes. Dead code in 1.0.4 either way. |
| p | **Out-of-bounds write after a debug break.** `CheckDuplicateFloorSuite` computes `bitPlace = ((floor + 7) * 128) + suite`, checks it against `kRoomsTimesSuites` 8192, calls `DebugStr("\pBlew array")` — and then writes anyway. | `GliderPRO/Sources/HouseLegal.c:648`, `:665`, `:668` | In a release build `DebugStr` is a no-op, so a malformed house corrupts the stack. A Go port must bounds-check and reject. |
| q | **`kDirt` threshold/openness inconsistency.** Every other background group derives both from the same test — `if (leftTile == 0) leftThresh = kLeftWallLimit;` paired with `leftOpen = (leftTile != 0);` (`GliderPRO/Sources/Room.c:856-865`, the 10-background group). The `kDirt` arm tests `if (leftTile == 1)` for the threshold but still uses `leftOpen = (leftTile != 0)`. | `GliderPRO/Sources/Room.c:871` vs `GliderPRO/Sources/Room.c:879` | For a dirt room whose `tiles[0] == 1`: `leftThresh = kLeftWallLimit` (**12**) *and* `leftOpen = true`. The room is escapable on the left but the escape only triggers once `dest.left < 12` instead of `< kNoLeftWallLimit` (**−24**), i.e. 36 px earlier. For `tiles[0] == 0`: `leftThresh = kNoLeftWallLimit` (−24) with `leftOpen = false`, so the glider walks 24 px off-screen before being clamped. Both are visible in play; copy verbatim. |
| r | **`nRooms` is unbounded in play mode.** There is no `kMaxRooms` constant anywhere; the only limits are the editor's `kMaxNumRoomsH` 128 / `kMaxNumRoomsV` 64 on *coordinates*, and `HouseLegal.c`'s 8192-cell bitmap. | `grep -c 'kMaxRooms' → 0`; §22.7, `GliderPRO/Headers/GliderDefines.h:543-544` | Room count is bounded only by handle size. Validate on load in Go. |
| s | **Several "unused" header fields hold garbage on disk**, because `InitializeEmptyHouse` uses `NewHandle`, not `NewHandleClear`, and never assigns them. | `GliderPRO/Sources/House.c:108-161`, `:116`; 10 of the 22 houses have a non-zero `unusedShort` (147, 259, 222, 2074, 13107, 26228 and **−30082**) and 7 have a non-zero `unusedBoolean` (2, 14, 30, 37, 185, 255) — §22.7 | A Go writer must not assume these are 0, and a Go *reader* must ignore them. `unusedBoolean` is **not** a bool on disk and `unusedShort` goes negative. |
| t | **`Sampler.binhex` has 2 bytes of trailing slack.** Its data fork is 1564 bytes but `866 + 348 × 2 = 1562`. | §22.7 empirical table | Parse by the header's `nRooms`, not by dividing the file length. |
| u | **`'bnds'` resources DO ship, in 8 of the 22 houses** (`Slumberland` 20, `ImagineHouse PRO II` 14, `Demo House` 10, `Castle o' the Air` 9, `Rainbow's End` 6, `Land of Illusion` 5, `Leviathan` 4, `The Asylum Pro` 2), each 4 bytes with real non-zero codes. | §22.5, resource-fork inventories in §22.11 | `GetOriginalBounding` is a live path, not dead code. A port must read `'bnds'` from the house's resource fork; hard-coding 0 breaks room openness in those eight houses. |
| v | **All saved-game *restore* is dead code.** `SaveGame2`'s body is commented out and `OpenSavedGame` begins with `return false;`. The in-house `houseType.savedGame` is still *written* by `SaveGame`, but `hasGame` is never tested and the only reader of `savedGame`, `QueryResumeGame`, is never called. | `GliderPRO/Sources/SavedGames.c:32-33`, `:147`, `:169-170`, `:337`/`:341`; `GliderPRO/Sources/Menu.c:710-759` (uncalled); §22.9 | Do not port `.gliG`. In 1.0.4 "saving" only persists room/object state via `WriteHouse`; score, room and glider count round-trip nowhere. A port should wire up the intended resume rather than copy the dead end. |
---

## Appendix A — Complete index of every `#define` in the program

Script-generated from the LF-converted sources; **1327 rows**, one per `#define`,
grouped by file and in source order. This is the mechanical backstop for §1–§23:
if a constant is not discussed in the body of this document, it is still here with its
exact value and location. Values are reproduced **verbatim** as they appear in the
source, including trailing `L` suffixes, hex spellings, parenthesised expressions and
references to other constants. The `Comment` column is the author's own trailing `//`
comment where one exists.

Regeneration:

```
cd GliderPRO && for f in Headers/*.h Sources/*.c; do \
  o="/tmp/wf-constants/$(echo $f | tr '/' '_')"; \
  tr '\r' '\n' < "$f" | iconv -f MAC -t UTF-8 > "$o" 2>/dev/null || tr '\r' '\n' < "$f" > "$o"; done
cd /tmp/wf-constants && python3 mkappendix.py
```

Counts by file:

| File | `#define` count |
|---|---:|
| `GliderPRO/Headers/GliderDefines.h` | 566 |
| `GliderPRO/Headers/Externs.h` | 197 |
| `GliderPRO/Sources/About.c` | 5 |
| `GliderPRO/Sources/AnimCursor.c` | 2 |
| `GliderPRO/Sources/AppleEvents.c` | 1 |
| `GliderPRO/Sources/Banner.c` | 5 |
| `GliderPRO/Sources/DialogUtils.c` | 6 |
| `GliderPRO/Sources/Dynamics.c` | 1 |
| `GliderPRO/Sources/Dynamics2.c` | 7 |
| `GliderPRO/Sources/Dynamics3.c` | 3 |
| `GliderPRO/Sources/Environ.c` | 12 |
| `GliderPRO/Sources/Events.c` | 1 |
| `GliderPRO/Sources/FileError.c` | 2 |
| `GliderPRO/Sources/GameOver.c` | 10 |
| `GliderPRO/Sources/Grease.c` | 4 |
| `GliderPRO/Sources/HighScores.c` | 11 |
| `GliderPRO/Sources/House.c` | 6 |
| `GliderPRO/Sources/HouseIO.c` | 4 |
| `GliderPRO/Sources/HouseInfo.c` | 11 |
| `GliderPRO/Sources/HouseLegal.c` | 1 |
| `GliderPRO/Sources/Input.c` | 8 |
| `GliderPRO/Sources/Interactions.c` | 8 |
| `GliderPRO/Sources/InterfaceInit.c` | 4 |
| `GliderPRO/Sources/Link.c` | 2 |
| `GliderPRO/Sources/Main.c` | 1 |
| `GliderPRO/Sources/MainWindow.c` | 4 |
| `GliderPRO/Sources/Map.c` | 9 |
| `GliderPRO/Sources/Marquee.c` | 2 |
| `GliderPRO/Sources/Menu.c` | 5 |
| `GliderPRO/Sources/Modes.c` | 3 |
| `GliderPRO/Sources/Music.c` | 5 |
| `GliderPRO/Sources/ObjectAdd.c` | 9 |
| `GliderPRO/Sources/ObjectDraw.c` | 55 |
| `GliderPRO/Sources/ObjectDraw2.c` | 88 |
| `GliderPRO/Sources/ObjectInfo.c` | 46 |
| `GliderPRO/Sources/ObjectRects.c` | 7 |
| `GliderPRO/Sources/Objects.c` | 1 |
| `GliderPRO/Sources/Play.c` | 6 |
| `GliderPRO/Sources/Player.c` | 23 |
| `GliderPRO/Sources/Prefs.c` | 7 |
| `GliderPRO/Sources/Render.c` | 1 |
| `GliderPRO/Sources/Room.c` | 2 |
| `GliderPRO/Sources/RoomGraphics.c` | 1 |
| `GliderPRO/Sources/RoomInfo.c` | 17 |
| `GliderPRO/Sources/RubberBands.c` | 3 |
| `GliderPRO/Sources/SavedGames.c` | 2 |
| `GliderPRO/Sources/Scoreboard.c` | 11 |
| `GliderPRO/Sources/SelectHouse.c` | 17 |
| `GliderPRO/Sources/Settings.c` | 45 |
| `GliderPRO/Sources/Sound.c` | 5 |
| `GliderPRO/Sources/StringUtils.c` | 2 |
| `GliderPRO/Sources/StructuresInit.c` | 20 |
| `GliderPRO/Sources/StructuresInit2.c` | 3 |
| `GliderPRO/Sources/Tools.c` | 32 |
| `GliderPRO/Sources/Transitions.c` | 4 |
| `GliderPRO/Sources/Triggers.c` | 1 |
| `GliderPRO/Sources/Utilities.c` | 3 |
| `GliderPRO/Sources/Validate.c` | 8 |
| `GliderPRO/Sources/WindowUtils.c` | 2 |
| **Total** | **1327** |

### `GliderPRO/Headers/GliderDefines.h` — 566 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 14 | `COMPILENOCP` |  |  |
| 15 | `COMPILEQT` |  |  |
| 16 | `BUILD_ARCADE_VERSION` | `1` |  |
| 18 | `kYellowUnaccounted` | `1` |  |
| 19 | `kYellowFailedResOpen` | `2` |  |
| 20 | `kYellowFailedResAdd` | `3` |  |
| 21 | `kYellowFailedResCreate` | `4` |  |
| 22 | `kYellowNoHouses` | `5` |  |
| 23 | `kYellowNewerVersion` | `6` |  |
| 24 | `kYellowNoBackground` | `7` |  |
| 25 | `kYellowIllegalRoomNum` | `8` |  |
| 26 | `kYellowNoBoundsRes` | `9` |  |
| 27 | `kYellowScrapError` | `10` |  |
| 28 | `kYellowNoMemory` | `11` |  |
| 29 | `kYellowFailedWrite` | `12` |  |
| 30 | `kYellowNoMusic` | `13` |  |
| 31 | `kYellowFailedSound` | `14` |  |
| 32 | `kYellowAppleEventErr` | `15` |  |
| 33 | `kYellowOpenedOldHouse` | `16` |  |
| 34 | `kYellowLostAllHouses` | `17` |  |
| 35 | `kYellowFailedSaveGame` | `18` |  |
| 36 | `kYellowSavedTimeWrong` | `19` |  |
| 37 | `kYellowSavedVersWrong` | `20` |  |
| 38 | `kYellowSavedRoomsWrong` | `21` |  |
| 39 | `kYellowQTMovieNotLoaded` | `22` |  |
| 40 | `kYellowNoRooms` | `23` |  |
| 41 | `kYellowCantOrderLinks` | `24` |  |
| 43 | `kSwitchIfNeeded` | `0` |  |
| 44 | `kSwitchTo256Colors` | `1` |  |
| 45 | `kSwitchTo16Grays` | `2` |  |
| 47 | `kProdGameScoreMode` | `-4` |  |
| 48 | `kKickGameScoreMode` | `-3` |  |
| 49 | `kPlayGameScoreMode` | `-2` |  |
| 50 | `kPlayWholeScoreMode` | `-1` |  |
| 51 | `kPlayChorus` | `4` |  |
| 52 | `kPlayRefrainSparse1` | `5` |  |
| 53 | `kPlayRefrainSparse2` | `6` |  |
| 55 | `kHitWallSound` | `0` | •••••• |
| 56 | `kFadeInSound` | `1` | •• |
| 57 | `kFadeOutSound` | `2` | •••••• |
| 58 | `kBeepsSound` | `3` | •• |
| 59 | `kBuzzerSound` | `4` | •••••• |
| 60 | `kDingSound` | `5` |  |
| 61 | `kEnergizeSound` | `6` | •••••• |
| 62 | `kFollowSound` | `7` | ••    •• |
| 63 | `kMicrowavedSound` | `8` | ••    •• |
| 64 | `kSwitchSound` | `9` | ••    •• |
| 65 | `kBirdSound` | `10` | •••••• |
| 66 | `kCuckooSound` | `11` |  |
| 67 | `kTikSound` | `12` | ••    •• |
| 68 | `kTokSound` | `13` | ••    •• |
| 69 | `kBlowerOn` | `14` | ••    •• |
| 70 | `kBlowerOff` | `15` | ••    •• |
| 71 | `kCaughtFireSound` | `16` | •••••• |
| 72 | `kScoreTikSound` | `17` |  |
| 73 | `kThrustSound` | `18` | •••   •• |
| 74 | `kFizzleSound` | `19` | ••••  •• |
| 75 | `kFireBandSound` | `20` | •• •• •• |
| 76 | `kBandReboundSound` | `21` | ••  •••• |
| 77 | `kGreaseSpillSound` | `22` | ••   ••• |
| 78 | `kChordSound` | `23` |  |
| 79 | `kVCRSound` | `24` | ••••••• |
| 80 | `kFoilHitSound` | `25` | ••    •• |
| 81 | `kShredSound` | `26` | ••    •• |
| 82 | `kToastLaunchSound` | `27` | ••    •• |
| 83 | `kToastLandSound` | `28` | ••••••• |
| 84 | `kMacOnSound` | `29` |  |
| 85 | `kMacBeepSound` | `30` |  |
| 86 | `kMacOffSound` | `31` |  |
| 87 | `kTVOnSound` | `32` |  |
| 88 | `kTVOffSound` | `33` | •••••• |
| 89 | `kCoffeeSound` | `34` | •• |
| 90 | `kMysticSound` | `35` | •••••• |
| 91 | `kZapSound` | `36` | •• |
| 92 | `kPopSound` | `37` | •••••• |
| 93 | `kEnemyInSound` | `38` |  |
| 94 | `kEnemyOutSound` | `39` | •••••• |
| 95 | `kPaperCrunchSound` | `40` | ••    •• |
| 96 | `kBounceSound` | `41` | ••    •• |
| 97 | `kDripSound` | `42` | ••    •• |
| 98 | `kDropSound` | `43` | •••••• |
| 99 | `kFishOutSound` | `44` |  |
| 100 | `kFishInSound` | `45` | ••    •• |
| 101 | `kDontExitSound` | `46` | ••    •• |
| 102 | `kSizzleSound` | `47` | ••    •• |
| 103 | `kPaper1Sound` | `48` | ••    •• |
| 104 | `kPaper2Sound` | `49` | •••••• |
| 105 | `kPaper3Sound` | `50` |  |
| 106 | `kPaper4Sound` | `51` | •••   •• |
| 107 | `kTypingSound` | `52` | ••••  •• |
| 108 | `kCarriageSound` | `53` | •• •• •• |
| 109 | `kChord2Sound` | `54` | ••  •••• |
| 110 | `kPhoneRingSound` | `55` | ••   ••• |
| 111 | `kChime1Sound` | `56` |  |
| 112 | `kChime2Sound` | `57` | ••••••• |
| 113 | `kWebTwangSound` | `58` | ••    •• |
| 114 | `kTransOutSound` | `59` | ••    •• |
| 115 | `kTransInSound` | `60` | ••    •• |
| 116 | `kBonusSound` | `61` | ••••••• |
| 117 | `kHissSound` | `62` |  |
| 118 | `kTriggerSound` | `63` |  |
| 120 | `kHitWallPriority` | `100` | •••••• |
| 121 | `kScoreTikPriority` | `101` | •• |
| 122 | `kBandReboundPriority` | `102` | •••••• |
| 123 | `kDontExitPriority` | `103` | •• |
| 124 | `kTikPriority` | `200` | •••••• |
| 125 | `kTokPriority` | `201` |  |
| 126 | `kMysticPriority` | `202` | •••••• |
| 127 | `kChime1Priority` | `203` | ••    •• |
| 128 | `kChime2Priority` | `204` | ••    •• |
| 129 | `kThrustPriority` | `300` | ••    •• |
| 130 | `kFireBandPriority` | `301` | •••••• |
| 131 | `kChordPriority` | `302` |  |
| 132 | `kVCRPriority` | `303` | ••    •• |
| 133 | `kToastLaunchPriority` | `304` | ••    •• |
| 134 | `kToastLandPriority` | `305` | ••    •• |
| 135 | `kCoffeePriority` | `306` | ••    •• |
| 136 | `kBouncePriority` | `307` | •••••• |
| 137 | `kDripPriority` | `308` |  |
| 138 | `kDropPriority` | `309` | •••   •• |
| 139 | `kWebTwangPriority` | `310` | ••••  •• |
| 140 | `kHissPriority` | `311` | •• •• •• |
| 141 | `kFoilHitPriority` | `400` | ••  •••• |
| 142 | `kMacOnPriority` | `401` | ••   ••• |
| 143 | `kMacOffPriority` | `402` |  |
| 144 | `kMacBeepPriority` | `403` | ••••••• |
| 145 | `kTVOnPriority` | `404` | ••    •• |
| 146 | `kTVOffPriority` | `405` | ••    •• |
| 147 | `kZapPriority` | `406` | ••    •• |
| 148 | `kPopPriority` | `407` | ••••••• |
| 149 | `kEnemyInPriority` | `408` |  |
| 150 | `kEnemyOutPriority` | `409` |  |
| 151 | `kPaperCrunchPriority` | `410` |  |
| 152 | `kFishOutPriority` | `411` |  |
| 153 | `kFishInPriority` | `412` |  |
| 154 | `kSizzlePriority` | `413` |  |
| 155 | `kPhoneRingPriority` | `500` |  |
| 156 | `kSwitchPriority` | `700` |  |
| 157 | `kBlowerOnPriority` | `701` |  |
| 158 | `kBlowerOffPriority` | `702` |  |
| 159 | `kFizzlePriority` | `703` |  |
| 160 | `kBeepsPriority` | `800` |  |
| 161 | `kBuzzerPriority` | `801` |  |
| 162 | `kDingPriority` | `802` |  |
| 163 | `kEnergizePriority` | `803` |  |
| 164 | `kBirdPriority` | `804` |  |
| 165 | `kCuckooPriority` | `805` |  |
| 166 | `kGreaseSpillPriority` | `806` |  |
| 167 | `kPapersPriority` | `807` |  |
| 168 | `kTypingPriority` | `808` |  |
| 169 | `kCarriagePriority` | `809` |  |
| 170 | `kChord2Priority` | `810` |  |
| 171 | `kMicrowavedPriority` | `811` |  |
| 172 | `kBonusPriority` | `812` |  |
| 173 | `kFadeInPriority` | `900` |  |
| 174 | `kFadeOutPriority` | `901` |  |
| 175 | `kCaughtFirePriority` | `902` |  |
| 176 | `kShredPriority` | `903` |  |
| 177 | `kFollowPriority` | `904` |  |
| 178 | `kTransInPriority` | `905` |  |
| 179 | `kTransOutPriority` | `906` |  |
| 180 | `kTriggerPriority` | `999` |  |
| 182 | `kArrowCursor` | `0` |  |
| 183 | `kBeamCursor` | `1` |  |
| 184 | `kHandCursor` | `2` |  |
| 186 | `kAppleMenuID` | `128` |  |
| 187 | `kGameMenuID` | `129` |  |
| 188 | `kOptionsMenuID` | `130` |  |
| 189 | `kHouseMenuID` | `131` |  |
| 191 | `kSplashMode` | `0` |  |
| 192 | `kEditMode` | `1` |  |
| 193 | `kPlayMode` | `2` |  |
| 195 | `kIdleSplashMode` | `0` |  |
| 196 | `kIdleDemoMode` | `1` |  |
| 197 | `kIdleSplashTicks` | `7200L` | 2 minutes |
| 198 | `kIdleLastMode` | `1` |  |
| 200 | `kRoomAbove` | `1` |  |
| 201 | `kRoomBelow` | `2` |  |
| 202 | `kRoomToRight` | `3` |  |
| 203 | `kRoomToLeft` | `4` |  |
| 205 | `kBumpUp` | `1` |  |
| 206 | `kBumpDown` | `2` |  |
| 207 | `kBumpRight` | `3` |  |
| 208 | `kBumpLeft` | `4` |  |
| 210 | `kAbove` | `1` |  |
| 211 | `kToRight` | `2` |  |
| 212 | `kBelow` | `3` |  |
| 213 | `kToLeft` | `4` |  |
| 214 | `kBottomCorner` | `5` |  |
| 215 | `kTopCorner` | `6` |  |
| 217 | `kCentralRoom` | `0` |  |
| 218 | `kNorthRoom` | `1` |  |
| 219 | `kNorthEastRoom` | `2` |  |
| 220 | `kEastRoom` | `3` |  |
| 221 | `kSouthEastRoom` | `4` |  |
| 222 | `kSouthRoom` | `5` |  |
| 223 | `kSouthWestRoom` | `6` |  |
| 224 | `kWestRoom` | `7` |  |
| 225 | `kNorthWestRoom` | `8` |  |
| 227 | `kSimpleRoom` | `2000` |  |
| 228 | `kPaneledRoom` | `2001` |  |
| 229 | `kBasement` | `2002` |  |
| 230 | `kChildsRoom` | `2003` |  |
| 231 | `kAsianRoom` | `2004` |  |
| 232 | `kUnfinishedRoom` | `2005` |  |
| 233 | `kSwingersRoom` | `2006` |  |
| 234 | `kBathroom` | `2007` |  |
| 235 | `kLibrary` | `2008` |  |
| 236 | `kGarden` | `2009` |  |
| 237 | `kSkywalk` | `2010` |  |
| 238 | `kDirt` | `2011` |  |
| 239 | `kMeadow` | `2012` |  |
| 240 | `kField` | `2013` |  |
| 241 | `kRoof` | `2014` |  |
| 242 | `kSky` | `2015` |  |
| 243 | `kStratosphere` | `2016` |  |
| 244 | `kStars` | `2017` |  |
| 246 | `kMapRoomHeight` | `20` |  |
| 247 | `kMapRoomWidth` | `32` |  |
| 249 | `kMaxScores` | `10` |  |
| 250 | `kMaxRoomObs` | `24` |  |
| 251 | `kMaxSparkles` | `3` |  |
| 252 | `kNumSparkleModes` | `5` |  |
| 253 | `kMaxFlyingPts` | `3` |  |
| 254 | `kMaxFlyingPointsLoop` | `24` |  |
| 255 | `kMaxCandles` | `20` |  |
| 256 | `kMaxTikis` | `8` |  |
| 257 | `kMaxCoals` | `8` |  |
| 258 | `kMaxPendulums` | `8` |  |
| 259 | `kMaxHotSpots` | `56` |  |
| 260 | `kMaxSavedMaps` | `24` |  |
| 261 | `kMaxRubberBands` | `2` |  |
| 262 | `kMaxGrease` | `16` |  |
| 263 | `kMaxStars` | `4` |  |
| 264 | `kMaxShredded` | `4` |  |
| 265 | `kMaxDynamicObs` | `18` |  |
| 266 | `kMaxMasterObjects` | `216` | kMaxRoomObs * 9 |
| 267 | `kMaxViewWidth` | `1536` |  |
| 268 | `kMaxViewHeight` | `1026` |  |
| 270 | `kSelectTool` | `0` |  |
| 272 | `kBlowerMode` | `1` |  |
| 273 | `kFurnitureMode` | `2` |  |
| 274 | `kBonusMode` | `3` |  |
| 275 | `kTransportMode` | `4` |  |
| 276 | `kSwitchMode` | `5` |  |
| 277 | `kLightMode` | `6` |  |
| 278 | `kApplianceMode` | `7` |  |
| 279 | `kEnemyMode` | `8` |  |
| 280 | `kClutterMode` | `9` |  |
| 282 | `kIgnoreIt` | `0` | •••••• |
| 283 | `kLiftIt` | `1` | ••    •• |
| 284 | `kDropIt` | `2` | •••••••• |
| 285 | `kPushItLeft` | `3` | ••    •• |
| 286 | `kPushItRight` | `4` | ••    •• |
| 287 | `kDissolveIt` | `5` |  |
| 288 | `kRewardIt` | `6` | •••••• |
| 289 | `kMoveItUp` | `7` | ••    •• |
| 290 | `kMoveItDown` | `8` | •• |
| 291 | `kSwitchIt` | `9` | ••    •• |
| 292 | `kShredIt` | `10` | •••••• |
| 293 | `kStrumIt` | `11` |  |
| 294 | `kTriggerIt` | `12` | •••••••• |
| 295 | `kBurnIt` | `13` | •• |
| 296 | `kSlideIt` | `14` | •• |
| 297 | `kTransportIt` | `15` | •• |
| 298 | `kIgnoreLeftWall` | `16` | •• |
| 299 | `kIgnoreRightWall` | `17` |  |
| 300 | `kMailItLeft` | `18` | •••••• |
| 301 | `kMailItRight` | `19` | •• |
| 302 | `kDuctItDown` | `20` | •• |
| 303 | `kDuctItUp` | `21` | •• |
| 304 | `kMicrowaveIt` | `22` | •••••• |
| 305 | `kIgnoreGround` | `23` |  |
| 306 | `kBounceIt` | `24` |  |
| 307 | `kChimeIt` | `25` | •• |
| 308 | `kWebIt` | `26` | •• |
| 309 | `kSoundIt` | `27` |  |
| 311 | `kFloorVent` | `0x01` | Blowers |
| 312 | `kCeilingVent` | `0x02` |  |
| 313 | `kFloorBlower` | `0x03` |  |
| 314 | `kCeilingBlower` | `0x04` |  |
| 315 | `kSewerGrate` | `0x05` |  |
| 316 | `kLeftFan` | `0x06` |  |
| 317 | `kRightFan` | `0x07` |  |
| 318 | `kTaper` | `0x08` |  |
| 319 | `kCandle` | `0x09` |  |
| 320 | `kStubby` | `0x0A` |  |
| 321 | `kTiki` | `0x0B` |  |
| 322 | `kBBQ` | `0x0C` |  |
| 323 | `kInvisBlower` | `0x0D` |  |
| 324 | `kGrecoVent` | `0x0E` |  |
| 325 | `kSewerBlower` | `0x0F` |  |
| 326 | `kLiftArea` | `0x10` |  |
| 328 | `kTable` | `0x11` | Furniture |
| 329 | `kShelf` | `0x12` |  |
| 330 | `kCabinet` | `0x13` |  |
| 331 | `kFilingCabinet` | `0x14` |  |
| 332 | `kWasteBasket` | `0x15` |  |
| 333 | `kMilkCrate` | `0x16` |  |
| 334 | `kCounter` | `0x17` |  |
| 335 | `kDresser` | `0x18` |  |
| 336 | `kDeckTable` | `0x19` |  |
| 337 | `kStool` | `0x1A` |  |
| 338 | `kTrunk` | `0x1B` |  |
| 339 | `kInvisObstacle` | `0x1C` |  |
| 340 | `kManhole` | `0x1D` |  |
| 341 | `kBooks` | `0x1E` |  |
| 342 | `kInvisBounce` | `0x1F` |  |
| 344 | `kRedClock` | `0x21` | Prizes |
| 345 | `kBlueClock` | `0x22` |  |
| 346 | `kYellowClock` | `0x23` |  |
| 347 | `kCuckoo` | `0x24` |  |
| 348 | `kPaper` | `0x25` |  |
| 349 | `kBattery` | `0x26` |  |
| 350 | `kBands` | `0x27` |  |
| 351 | `kGreaseRt` | `0x28` |  |
| 352 | `kGreaseLf` | `0x29` |  |
| 353 | `kFoil` | `0x2A` |  |
| 354 | `kInvisBonus` | `0x2B` |  |
| 355 | `kStar` | `0x2C` |  |
| 356 | `kSparkle` | `0x2D` |  |
| 357 | `kHelium` | `0x2E` |  |
| 358 | `kSlider` | `0x2F` |  |
| 360 | `kUpStairs` | `0x31` | Transport |
| 361 | `kDownStairs` | `0x32` |  |
| 362 | `kMailboxLf` | `0x33` |  |
| 363 | `kMailboxRt` | `0x34` |  |
| 364 | `kFloorTrans` | `0x35` |  |
| 365 | `kCeilingTrans` | `0x36` |  |
| 366 | `kDoorInLf` | `0x37` |  |
| 367 | `kDoorInRt` | `0x38` |  |
| 368 | `kDoorExRt` | `0x39` |  |
| 369 | `kDoorExLf` | `0x3A` |  |
| 370 | `kWindowInLf` | `0x3B` |  |
| 371 | `kWindowInRt` | `0x3C` |  |
| 372 | `kWindowExRt` | `0x3D` |  |
| 373 | `kWindowExLf` | `0x3E` |  |
| 374 | `kInvisTrans` | `0x3F` |  |
| 375 | `kDeluxeTrans` | `0x40` |  |
| 377 | `kLightSwitch` | `0x41` | Switches |
| 378 | `kMachineSwitch` | `0x42` |  |
| 379 | `kThermostat` | `0x43` |  |
| 380 | `kPowerSwitch` | `0x44` |  |
| 381 | `kKnifeSwitch` | `0x45` |  |
| 382 | `kInvisSwitch` | `0x46` |  |
| 383 | `kTrigger` | `0x47` |  |
| 384 | `kLgTrigger` | `0x48` |  |
| 385 | `kSoundTrigger` | `0x49` |  |
| 387 | `kCeilingLight` | `0x51` | Lights |
| 388 | `kLightBulb` | `0x52` |  |
| 389 | `kTableLamp` | `0x53` |  |
| 390 | `kHipLamp` | `0x54` |  |
| 391 | `kDecoLamp` | `0x55` |  |
| 392 | `kFlourescent` | `0x56` |  |
| 393 | `kTrackLight` | `0x57` |  |
| 394 | `kInvisLight` | `0x58` |  |
| 396 | `kShredder` | `0x61` | Appliances |
| 397 | `kToaster` | `0x62` |  |
| 398 | `kMacPlus` | `0x63` |  |
| 399 | `kGuitar` | `0x64` |  |
| 400 | `kTV` | `0x65` |  |
| 401 | `kCoffee` | `0x66` |  |
| 402 | `kOutlet` | `0x67` |  |
| 403 | `kVCR` | `0x68` |  |
| 404 | `kStereo` | `0x69` |  |
| 405 | `kMicrowave` | `0x6A` |  |
| 406 | `kCinderBlock` | `0x6B` |  |
| 407 | `kFlowerBox` | `0x6C` |  |
| 408 | `kCDs` | `0x6D` |  |
| 409 | `kCustomPict` | `0x6E` |  |
| 411 | `kBalloon` | `0x71` | Enemies |
| 412 | `kCopterLf` | `0x72` |  |
| 413 | `kCopterRt` | `0x73` |  |
| 414 | `kDartLf` | `0x74` |  |
| 415 | `kDartRt` | `0x75` |  |
| 416 | `kBall` | `0x76` |  |
| 417 | `kDrip` | `0x77` |  |
| 418 | `kFish` | `0x78` |  |
| 419 | `kCobweb` | `0x79` |  |
| 421 | `kOzma` | `0x81` | Clutter |
| 422 | `kMirror` | `0x82` |  |
| 423 | `kMousehole` | `0x83` |  |
| 424 | `kFireplace` | `0x84` |  |
| 425 | `kFlower` | `0x85` |  |
| 426 | `kWallWindow` | `0x86` |  |
| 427 | `kBear` | `0x87` |  |
| 428 | `kCalendar` | `0x88` |  |
| 429 | `kVase1` | `0x89` |  |
| 430 | `kVase2` | `0x8A` |  |
| 431 | `kBulletin` | `0x8B` |  |
| 432 | `kCloud` | `0x8C` |  |
| 433 | `kFaucet` | `0x8D` |  |
| 434 | `kRug` | `0x8E` |  |
| 435 | `kChimes` | `0x8F` |  |
| 437 | `kNumSrcRects` | `0x90` |  |
| 439 | `kTableThick` | `8` |  |
| 440 | `kShelfThick` | `6` |  |
| 441 | `kToggle` | `0` |  |
| 442 | `kForceOn` | `1` |  |
| 443 | `kForceOff` | `2` |  |
| 444 | `kOneShot` | `3` |  |
| 445 | `kNumTrackLights` | `3` |  |
| 446 | `kNumOutletPicts` | `4` |  |
| 447 | `kNumCandleFlames` | `5` |  |
| 448 | `kNumTikiFlames` | `5` |  |
| 449 | `kNumBBQCoals` | `4` |  |
| 450 | `kNumPendulums` | `3` |  |
| 451 | `kNumBreadPicts` | `6` |  |
| 452 | `kNumBalloonFrames` | `8` |  |
| 453 | `kNumCopterFrames` | `10` |  |
| 454 | `kNumDartFrames` | `4` |  |
| 455 | `kNumBallFrames` | `2` |  |
| 456 | `kNumDripFrames` | `6` |  |
| 457 | `kNumFishFrames` | `8` |  |
| 458 | `kNumFlowers` | `6` |  |
| 460 | `kNumMarqueePats` | `7` |  |
| 461 | `kObjectNameStrings` | `1007` |  |
| 463 | `kSwitchLinkOnly` | `3` |  |
| 464 | `kTriggerLinkOnly` | `4` |  |
| 465 | `kTransportLinkOnly` | `5` |  |
| 467 | `kFloorVentTop` | `305` |  |
| 468 | `kCeilingVentTop` | `8` |  |
| 469 | `kFloorBlowerTop` | `304` |  |
| 470 | `kCeilingBlowerTop` | `5` |  |
| 471 | `kSewerGrateTop` | `303` |  |
| 472 | `kCeilingTransTop` | `6` |  |
| 473 | `kFloorTransTop` | `302` |  |
| 474 | `kStairsTop` | `28` |  |
| 475 | `kCounterBottom` | `304` |  |
| 476 | `kDresserBottom` | `293` |  |
| 477 | `kCeilingLightTop` | `4` |  |
| 478 | `kHipLampTop` | `23` |  |
| 479 | `kDecoLampTop` | `91` |  |
| 480 | `kFlourescentTop` | `12` |  |
| 481 | `kTrackLightTop` | `5` |  |
| 483 | `kDoorInTop` | `0` |  |
| 484 | `kDoorInLfLeft` | `0` |  |
| 485 | `kDoorInRtLeft` | `368` |  |
| 486 | `kDoorExTop` | `0` |  |
| 487 | `kDoorExLfLeft` | `0` |  |
| 488 | `kDoorExRtLeft` | `496` |  |
| 489 | `kWindowInTop` | `64` |  |
| 490 | `kWindowInLfLeft` | `0` |  |
| 491 | `kWindowInRtLeft` | `492` |  |
| 492 | `kWindowExTop` | `64` |  |
| 493 | `kWindowExLfLeft` | `0` |  |
| 494 | `kWindowExRtLeft` | `496` |  |
| 496 | `kNumTiles` | `8` |  |
| 497 | `kTileWide` | `64` |  |
| 498 | `kTileHigh` | `322` |  |
| 499 | `kRoomWide` | `512` | kNumTiles * kTileWide |
| 500 | `kFloorSupportTall` | `44` |  |
| 501 | `kVertLocalOffset` | `322` | kTileHigh - 39 (was 283, then 295) |
| 503 | `kCeilingLimit` | `8` |  |
| 504 | `kFloorLimit` | `312` |  |
| 505 | `kRoofLimit` | `122` |  |
| 506 | `kLeftWallLimit` | `12` |  |
| 507 | `kNoLeftWallLimit` | `-24` | 0 - (kGliderWide / 2) |
| 508 | `kRightWallLimit` | `500` |  |
| 509 | `kNoRightWallLimit` | `536` | kRoomWide + (kGliderWide / 2) |
| 510 | `kNoCeilingLimit` | `-10` |  |
| 511 | `kNoFloorLimit` | `332` |  |
| 513 | `kScoreboardHigh` | `0` |  |
| 514 | `kScoreboardLow` | `1` |  |
| 515 | `kScoreboardTall` | `20` |  |
| 517 | `kHouseVersion` | `0x0200` |  |
| 518 | `kNewHouseVersion` | `0x0300` |  |
| 519 | `kBaseBackgroundID` | `2000` |  |
| 520 | `kFirstOutdoorBack` | `2009` |  |
| 521 | `kNumBackgrounds` | `18` |  |
| 522 | `kUserBackground` | `3000` |  |
| 523 | `kUserStructureRange` | `3300` |  |
| 524 | `kSplash8BitPICT` | `1000` |  |
| 525 | `kRoomIsEmpty` | `-1` |  |
| 526 | `kObjectIsEmpty` | `-1` |  |
| 527 | `kNoObjectSelected` | `-1` |  |
| 528 | `kInitialGliderSelected` | `-2` |  |
| 529 | `kLeftGliderSelected` | `-3` |  |
| 530 | `kRightGliderSelected` | `-4` |  |
| 531 | `kWindoidWDEF` | `2048` |  |
| 532 | `kWindoidGrowWDEF` | `2064` |  |
| 533 | `kTicksPerFrame` | `2` |  |
| 534 | `kStarPictID` | `1995` |  |
| 535 | `kNumUndergroundFloors` | `8` |  |
| 536 | `kRoomVisitScore` | `100` |  |
| 537 | `kRedClockPoints` | `100` |  |
| 538 | `kBlueClockPoints` | `300` |  |
| 539 | `kYellowClockPoints` | `500` |  |
| 540 | `kCuckooClockPoints` | `1000` |  |
| 541 | `kStarPoints` | `5000` |  |
| 542 | `kRedOrangeColor8` | `23` | actually, 18 |
| 543 | `kMaxNumRoomsH` | `128` |  |
| 544 | `kMaxNumRoomsV` | `64` |  |
| 545 | `kStartSparkle` | `4` |  |
| 546 | `kLengthOfZap` | `30` |  |
| 548 | `kGliderWide` | `48` |  |
| 549 | `kGliderHigh` | `20` |  |
| 550 | `kHalfGliderWide` | `24` |  |
| 551 | `kGliderBurningHigh` | `26` |  |
| 552 | `kShadowHigh` | `9` |  |
| 553 | `kShadowTop` | `306` |  |
| 554 | `kFaceRight` | `TRUE` |  |
| 555 | `kFaceLeft` | `FALSE` |  |
| 556 | `kPlayer1` | `TRUE` |  |
| 557 | `kPlayer2` | `FALSE` |  |
| 558 | `kNumGliderSrcRects` | `31` |  |
| 559 | `kNumShadowSrcRects` | `2` |  |
| 560 | `kFirstAboutFaceFrame` | `18` |  |
| 561 | `kLastAboutFaceFrame` | `20` |  |
| 562 | `kWasBurning` | `2` |  |
| 563 | `kLeftFadeOffset` | `7` |  |
| 564 | `kLastFadeSequence` | `16` |  |
| 565 | `kGliderFoil2PictID` | `3963` |  |
| 566 | `kGlider2PictID` | `3974` |  |
| 567 | `kGliderFoilPictID` | `3976` |  |
| 568 | `kGliderPictID` | `3999` |  |
| 569 | `kGliderStartsDown` | `32` |  |
| 571 | `kGliderNormal` | `0` | ••    •• |
| 572 | `kGliderFadingIn` | `1` | •••  ••• |
| 573 | `kGliderFadingOut` | `2` | •• •• •• |
| 574 | `kGliderGoingUp` | `3` | ••    •• |
| 575 | `kGliderComingUp` | `4` | ••    •• |
| 576 | `kGliderGoingDown` | `5` |  |
| 577 | `kGliderComingDown` | `6` | •••••• |
| 578 | `kGliderFaceLeft` | `7` | ••    •• |
| 579 | `kGliderFaceRight` | `8` | ••    •• |
| 580 | `kGliderBurning` | `9` | ••    •• |
| 581 | `kGliderTransporting` | `10` | •••••• |
| 582 | `kGliderDuctingDown` | `11` |  |
| 583 | `kGliderDuctingUp` | `12` | ••••••• |
| 584 | `kGliderDuctingIn` | `13` | ••    •• |
| 585 | `kGliderMailInLeft` | `14` | ••    •• |
| 586 | `kGliderMailOutLeft` | `15` | ••    •• |
| 587 | `kGliderMailInRight` | `16` | ••••••• |
| 588 | `kGliderMailOutRight` | `17` |  |
| 589 | `kGliderGoingFoil` | `18` | •••••••• |
| 590 | `kGliderLosingFoil` | `19` | •• |
| 591 | `kGliderShredding` | `20` | •••• |
| 592 | `kGliderInLimbo` | `21` | •• |
| 593 | `kGliderIdle` | `22` | •••••••• |
| 594 | `kGliderTransportingIn` | `23` |  |
| 596 | `kPlayerIsDeadForever` | `-69` |  |
| 597 | `kPlayerMailedOut` | `-12` |  |
| 598 | `kPlayerDuckedOut` | `-11` |  |
| 599 | `kPlayerTransportedOut` | `-10` |  |
| 600 | `kPlayerEscapingDownStairs` | `-9` |  |
| 601 | `kPlayerEscapingUpStairs` | `-8` |  |
| 602 | `kPlayerEscapedDownStairs` | `-7` |  |
| 603 | `kPlayerEscapedUpStairs` | `-6` |  |
| 604 | `kPlayerEscapedDown` | `-5` |  |
| 605 | `kPlayerEscapedUp` | `-4` |  |
| 606 | `kPlayerEscapedLeft` | `-3` |  |
| 607 | `kPlayerEscapedRight` | `-2` |  |
| 608 | `kNoOneEscaped` | `-1` |  |
| 610 | `kLinkedToOther` | `0` |  |
| 611 | `kLinkedToLeftMailbox` | `1` |  |
| 612 | `kLinkedToRightMailbox` | `2` |  |
| 613 | `kLinkedToCeilingDuct` | `3` |  |
| 614 | `kLinkedToFloorDuct` | `4` |  |
| 616 | `kResumeGameMode` | `0` |  |
| 617 | `kNewGameMode` | `1` |  |
| 619 | `kNormalTitleMode` | `0` |  |
| 620 | `kEscapedTitleMode` | `1` |  |
| 621 | `kSavingTitleMode` | `2` |  |
| 623 | `kScoreboardPictID` | `1997` |  |
| 625 | `kDemoLength` | `6702` |  |

### `GliderPRO/Headers/Externs.h` — 197 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 15 | `kPreferredDepth` | `8` |  |
| 18 | `kNilPointer` | `0L` |  |
| 19 | `kPutInFront` | `(WindowPtr)-1L` |  |
| 20 | `kNormalUpdates` | `TRUE` |  |
| 21 | `kOneKilobyte` | `1024` |  |
| 22 | `kOkayButton` | `1` |  |
| 23 | `kCancelButton` | `2` |  |
| 24 | `kControlActive` | `0` |  |
| 25 | `kControlInactive` | `255` |  |
| 26 | `kAsynch` | `TRUE` |  |
| 27 | `kSynch` | `FALSE` |  |
| 29 | `kHomeKeyASCII` | `0x01` |  |
| 30 | `kEnterKeyASCII` | `0x03` |  |
| 31 | `kEndKeyASCII` | `0x04` |  |
| 32 | `kHelpKeyASCII` | `0x05` |  |
| 33 | `kDeleteKeyASCII` | `0x08` |  |
| 34 | `kTabKeyASCII` | `0x09` |  |
| 35 | `kPageUpKeyASCII` | `0x0B` |  |
| 36 | `kPageDownKeyASCII` | `0x0C` |  |
| 37 | `kReturnKeyASCII` | `0x0D` |  |
| 38 | `kFunctionKeyASCII` | `0x10` |  |
| 39 | `kClearKeyASCII` | `0x1A` |  |
| 40 | `kEscapeKeyASCII` | `0x1B` |  |
| 41 | `kLeftArrowKeyASCII` | `0x1C` |  |
| 42 | `kRightArrowKeyASCII` | `0x1D` |  |
| 43 | `kUpArrowKeyASCII` | `0x1E` |  |
| 44 | `kDownArrowKeyASCII` | `0x1F` |  |
| 45 | `kSpaceBarASCII` | `0x20` |  |
| 46 | `kExclamationASCII` | `0x21` |  |
| 47 | `kPlusKeyASCII` | `0x2B` |  |
| 48 | `kMinusKeyASCII` | `0x2D` |  |
| 49 | `k0KeyASCII` | `0x30` |  |
| 50 | `k1KeyASCII` | `0x31` |  |
| 51 | `k2KeyASCII` | `0x32` |  |
| 52 | `k3KeyASCII` | `0x33` |  |
| 53 | `k4KeyASCII` | `0x34` |  |
| 54 | `k5KeyASCII` | `0x35` |  |
| 55 | `k6KeyASCII` | `0x36` |  |
| 56 | `k7KeyASCII` | `0x37` |  |
| 57 | `k8KeyASCII` | `0x38` |  |
| 58 | `k9KeyASCII` | `0x39` |  |
| 60 | `kCapAKeyASCII` | `0x41` |  |
| 61 | `kCapBKeyASCII` | `0x42` |  |
| 62 | `kCapCKeyASCII` | `0x43` |  |
| 63 | `kCapDKeyASCII` | `0x44` |  |
| 64 | `kCapEKeyASCII` | `0x45` |  |
| 65 | `kCapFKeyASCII` | `0x46` |  |
| 66 | `kCapGKeyASCII` | `0x47` |  |
| 67 | `kCapHKeyASCII` | `0x48` |  |
| 68 | `kCapIKeyASCII` | `0x49` |  |
| 69 | `kCapJKeyASCII` | `0x4A` |  |
| 70 | `kCapKKeyASCII` | `0x4B` |  |
| 71 | `kCapLKeyASCII` | `0x4C` |  |
| 72 | `kCapMKeyASCII` | `0x4D` |  |
| 73 | `kCapNKeyASCII` | `0x4E` |  |
| 74 | `kCapOKeyASCII` | `0x4F` |  |
| 75 | `kCapPKeyASCII` | `0x50` |  |
| 76 | `kCapQKeyASCII` | `0x51` |  |
| 77 | `kCapRKeyASCII` | `0x52` |  |
| 78 | `kCapSKeyASCII` | `0x53` |  |
| 79 | `kCapTKeyASCII` | `0x54` |  |
| 80 | `kCapUKeyASCII` | `0x55` |  |
| 81 | `kCapVKeyASCII` | `0x56` |  |
| 82 | `kCapWKeyASCII` | `0x57` |  |
| 83 | `kCapXKeyASCII` | `0x58` |  |
| 84 | `kCapYKeyASCII` | `0x59` |  |
| 85 | `kCapZKeyASCII` | `0x5A` |  |
| 87 | `kAKeyASCII` | `0x61` |  |
| 88 | `kBKeyASCII` | `0x62` |  |
| 89 | `kCKeyASCII` | `0x63` |  |
| 90 | `kDKeyASCII` | `0x64` |  |
| 91 | `kEKeyASCII` | `0x65` |  |
| 92 | `kFKeyASCII` | `0x66` |  |
| 93 | `kGKeyASCII` | `0x67` |  |
| 94 | `kHKeyASCII` | `0x68` |  |
| 95 | `kIKeyASCII` | `0x69` |  |
| 96 | `kJKeyASCII` | `0x6A` |  |
| 97 | `kKKeyASCII` | `0x6B` |  |
| 98 | `kLKeyASCII` | `0x6C` |  |
| 99 | `kMKeyASCII` | `0x6D` |  |
| 100 | `kNKeyASCII` | `0x6E` |  |
| 101 | `kOKeyASCII` | `0x6F` |  |
| 102 | `kPKeyASCII` | `0x70` |  |
| 103 | `kQKeyASCII` | `0x71` |  |
| 104 | `kRKeyASCII` | `0x72` |  |
| 105 | `kSKeyASCII` | `0x73` |  |
| 106 | `kTKeyASCII` | `0x74` |  |
| 107 | `kUKeyASCII` | `0x75` |  |
| 108 | `kVKeyASCII` | `0x76` |  |
| 109 | `kWKeyASCII` | `0x77` |  |
| 110 | `kXKeyASCII` | `0x78` |  |
| 111 | `kYKeyASCII` | `0x79` |  |
| 112 | `kZKeyASCII` | `0x7A` |  |
| 113 | `kForwardDeleteASCII` | `0x7F` |  |
| 115 | `kPlusKeypadMap` | `66` | key map offset for + on keypad |
| 116 | `kMinusKeypadMap` | `73` | key map offset for - on keypad |
| 117 | `kTimesKeypadMap` | `68` | key map offset for * on keypad |
| 118 | `k0KeypadMap` | `85` | key map offset for 0 on keypad |
| 119 | `k1KeypadMap` | `84` | key map offset for 1 on keypad |
| 120 | `k2KeypadMap` | `83` | key map offset for 2 on keypad |
| 121 | `k3KeypadMap` | `82` | key map offset for 3 on keypad |
| 122 | `k4KeypadMap` | `81` | key map offset for 4 on keypad |
| 123 | `k5KeypadMap` | `80` | key map offset for 5 on keypad |
| 124 | `k6KeypadMap` | `95` | key map offset for 6 on keypad |
| 125 | `k7KeypadMap` | `94` | key map offset for 7 on keypad |
| 126 | `k8KeypadMap` | `92` | key map offset for 8 on keypad |
| 127 | `k9KeypadMap` | `91` | key map offset for 9 on keypad |
| 129 | `kUpArrowKeyMap` | `121` | key map offset for up arrow |
| 130 | `kDownArrowKeyMap` | `122` | key map offset for down arrow |
| 131 | `kRightArrowKeyMap` | `123` | key map offset for right arrow |
| 132 | `kLeftArrowKeyMap` | `124` | key map offset for left arrow |
| 134 | `kAKeyMap` | `7` |  |
| 135 | `kBKeyMap` | `12` |  |
| 136 | `kCKeyMap` | `15` |  |
| 137 | `kDKeyMap` | `5` |  |
| 138 | `kEKeyMap` | `9` |  |
| 139 | `kFKeyMap` | `4` |  |
| 140 | `kGKeyMap` | `2` |  |
| 141 | `kHKeyMap` | `3` |  |
| 142 | `kMKeyMap` | `41` |  |
| 143 | `kNKeyMap` | `42` |  |
| 144 | `kOKeyMap` | `24` |  |
| 145 | `kPKeyMap` | `36` |  |
| 146 | `kQKeyMap` | `11` |  |
| 147 | `kRKeyMap` | `8` |  |
| 148 | `kSKeyMap` | `6` |  |
| 149 | `kTKeyMap` | `22` |  |
| 150 | `kVKeyMap` | `14` |  |
| 151 | `kWKeyMap` | `10` |  |
| 152 | `kXKeyMap` | `0` |  |
| 153 | `kZKeyMap` | `1` |  |
| 154 | `kPeriodKeyMap` | `40` |  |
| 155 | `kCommandKeyMap` | `48` |  |
| 156 | `kEscKeyMap` | `50` |  |
| 157 | `kDeleteKeyMap` | `52` |  |
| 158 | `kSpaceBarMap` | `54` |  |
| 159 | `kTabKeyMap` | `55` |  |
| 160 | `kControlKeyMap` | `60` |  |
| 161 | `kOptionKeyMap` | `61` |  |
| 162 | `kCapsLockKeyMap` | `62` |  |
| 163 | `kShiftKeyMap` | `63` |  |
| 165 | `kTabRawKey` | `0x30` | key map offset for Tab key |
| 166 | `kClearRawKey` | `0x47` | key map offset for Clear key |
| 167 | `kF5RawKey` | `0x60` | key map offset for F5 |
| 168 | `kF6RawKey` | `0x61` | key map offset for F6 |
| 169 | `kF7RawKey` | `0x62` | key map offset for F7 |
| 170 | `kF3RawKey` | `0x63` | key map offset for F3 |
| 171 | `kF8RawKey` | `0x64` | key map offset for F8 |
| 172 | `kF9RawKey` | `0x65` | key map offset for F9 |
| 173 | `kF11RawKey` | `0x67` | key map offset for F11 |
| 174 | `kF13RawKey` | `0x69` | key map offset for F13 |
| 175 | `kF14RawKey` | `0x6B` | key map offset for F14 |
| 176 | `kF10RawKey` | `0x6D` | key map offset for F10 |
| 177 | `kF12RawKey` | `0x6F` | key map offset for F12 |
| 178 | `kF15RawKey` | `0x71` | key map offset for F15 |
| 179 | `kF4RawKey` | `0x76` | key map offset for F4 |
| 180 | `kF2RawKey` | `0x78` | key map offset for F2 |
| 181 | `kF1RawKey` | `0x7A` | key map offset for F1 |
| 183 | `kErrUnnaccounted` | `1` |  |
| 184 | `kErrNoMemory` | `2` |  |
| 185 | `kErrDialogDidntLoad` | `3` |  |
| 186 | `kErrFailedResourceLoad` | `4` |  |
| 187 | `kErrFailedGraphicLoad` | `5` |  |
| 188 | `kErrFailedOurDirect` | `6` |  |
| 189 | `kErrFailedValidation` | `7` |  |
| 190 | `kErrNeedSystem7` | `8` |  |
| 191 | `kErrFailedGetDevice` | `9` |  |
| 192 | `kErrFailedMemoryOperation` | `10` |  |
| 193 | `kErrFailedCatSearch` | `11` |  |
| 194 | `kErrNeedColorQD` | `12` |  |
| 195 | `kErrNeed16Or256Colors` | `13` |  |
| 197 | `iAbout` | `1` |  |
| 198 | `iNewGame` | `1` |  |
| 199 | `iTwoPlayer` | `2` |  |
| 200 | `iOpenSavedGame` | `3` |  |
| 201 | `iLoadHouse` | `5` |  |
| 202 | `iQuit` | `7` |  |
| 203 | `iEditor` | `1` |  |
| 204 | `iHighScores` | `3` |  |
| 205 | `iPrefs` | `4` |  |
| 206 | `iHelp` | `5` |  |
| 207 | `iNewHouse` | `1` |  |
| 208 | `iSave` | `2` |  |
| 209 | `iHouse` | `4` |  |
| 210 | `iRoom` | `5` |  |
| 211 | `iObject` | `6` |  |
| 212 | `iCut` | `8` |  |
| 213 | `iCopy` | `9` |  |
| 214 | `iPaste` | `10` |  |
| 215 | `iClear` | `11` |  |
| 216 | `iDuplicate` | `12` |  |
| 217 | `iBringForward` | `14` |  |
| 218 | `iSendBack` | `15` |  |
| 219 | `iGoToRoom` | `17` |  |
| 220 | `iMapWindow` | `19` |  |
| 221 | `iObjectWindow` | `20` |  |
| 222 | `iCoordinateWindow` | `21` |  |

### `GliderPRO/Sources/About.c` — 5 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 34 | `kAboutDialogID` | `150` | res ID of About dialog |
| 35 | `kTextItemVers` | `2` | item number of version text |
| 36 | `kPictItemMain` | `4` | item number of main PICT |
| 97 | `kOkayButtPICTHiLit` | `151` | res ID of unhilit button PICT |
| 119 | `kOkayButtPICTNotHiLit` | `150` | res ID of hilit button PICT |

### `GliderPRO/Sources/AnimCursor.c` — 2 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 14 | `rAcurID` | `128` |  |
| 15 | `rHandCursorID` | `1000` |  |

### `GliderPRO/Sources/AppleEvents.c` — 1 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 14 | `kNoPrintingAlert` | `1031` |  |

### `GliderPRO/Sources/Banner.c` — 5 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 17 | `kBannerPageTopPICT` | `1993` |  |
| 18 | `kBannerPageBottomPICT` | `1992` |  |
| 19 | `kBannerPageBottomMask` | `1991` |  |
| 20 | `kStarsRemainingPICT` | `1017` |  |
| 21 | `kStarRemainingPICT` | `1018` |  |

### `GliderPRO/Sources/DialogUtils.c` — 6 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 15 | `kActive` | `0` |  |
| 16 | `kInactive` | `255` |  |
| 228 | `kSteps` | `16` |  |
| 229 | `kZoomDelay` | `1` |  |
| 282 | `kSteps` | `16` |  |
| 283 | `kZoomDelay` | `1` |  |

### `GliderPRO/Sources/Dynamics.c` — 1 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 17 | `kShoveVelocity` | `8` |  |

### `GliderPRO/Sources/Dynamics2.c` — 7 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 13 | `kBalloonStop` | `8` |  |
| 14 | `kBalloonStart` | `310` |  |
| 15 | `kCopterStart` | `8` |  |
| 16 | `kCopterStop` | `310` |  |
| 17 | `kDartVelocity` | `6` |  |
| 18 | `kDartStop` | `310` |  |
| 19 | `kEnemyDropSpeed` | `8` |  |

### `GliderPRO/Sources/Dynamics3.c` — 3 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 13 | `kBalloonStart` | `310` |  |
| 14 | `kCopterStart` | `8` |  |
| 15 | `kDartVelocity` | `6` |  |

### `GliderPRO/Sources/Environ.c` — 12 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 18 | `kSwitchDepthAlert` | `130` |  |
| 19 | `kSetMemoryAlert` | `180` |  |
| 20 | `kLowMemoryAlert` | `181` |  |
| 21 | `kWNETrap` | `0x60` |  |
| 22 | `kSetDepthTrap` | `0xA2` |  |
| 23 | `kUnimpTrap` | `0x9F` |  |
| 24 | `kGestaltTrap` | `0xAD` |  |
| 26 | `kDisplay9Inch` | `1` |  |
| 27 | `kDisplay12Inch` | `2` |  |
| 28 | `kDisplay13Inch` | `3` |  |
| 564 | `kBaseBytesNeeded` | `614400L` | 600K Base memory |
| 565 | `kPaddingBytes` | `204800L` | 200K Padding |

### `GliderPRO/Sources/Events.c` — 1 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 48 | `kColorSwitchedAlert` | `1042` |  |

### `GliderPRO/Sources/FileError.c` — 2 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 14 | `rFileErrorAlert` | `140` |  |
| 15 | `rFileErrorStrings` | `140` |  |

### `GliderPRO/Sources/GameOver.c` — 10 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 17 | `kNumCountDownFrames` | `16` |  |
| 18 | `kPageFrames` | `14` |  |
| 19 | `kPagesPictID` | `1990` |  |
| 20 | `kPagesMaskID` | `1989` |  |
| 21 | `kLettersPictID` | `1988` |  |
| 22 | `kMilkywayPictID` | `1021` |  |
| 138 | `kStarFalls` | `8` |  |
| 251 | `kPageSpacing` | `40` |  |
| 252 | `kPageRightOffset` | `128` |  |
| 253 | `kPageBackUp` | `128` |  |

### `GliderPRO/Sources/Grease.c` — 4 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 17 | `kGreaseIdle` | `0` |  |
| 18 | `kGreaseFalling` | `1` |  |
| 19 | `kGreaseSpreading` | `2` |  |
| 20 | `kGreaseSpiltIdle` | `3` |  |

### `GliderPRO/Sources/HighScores.c` — 11 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 23 | `kHighScoresPictID` | `1994` |  |
| 24 | `kHighScoresMaskID` | `1998` |  |
| 25 | `kHighNameDialogID` | `1020` |  |
| 26 | `kHighBannerDialogID` | `1021` |  |
| 27 | `kHighNameItem` | `2` |  |
| 28 | `kNameNCharsItem` | `5` |  |
| 29 | `kHighBannerItem` | `2` |  |
| 30 | `kBannerScoreNCharsItem` | `5` |  |
| 90 | `kScoreSpacing` | `18` |  |
| 91 | `kScoreWide` | `352` |  |
| 92 | `kKimsLifted` | `4` |  |

### `GliderPRO/Sources/House.c` — 6 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 18 | `kGoToDialogID` | `1043` |  |
| 668 | `kGoToFirstButt` | `2` |  |
| 669 | `kGoToPrevButt` | `3` |  |
| 670 | `kGoToFSButt` | `4` |  |
| 671 | `kFloorEditText` | `5` |  |
| 672 | `kSuiteEditText` | `6` |  |

### `GliderPRO/Sources/HouseIO.c` — 4 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 20 | `kSaveChangesAlert` | `1002` |  |
| 21 | `kSaveChanges` | `1` |  |
| 22 | `kDiscardChanges` | `2` |  |
| 642 | `kYellowAlert` | `1006` |  |

### `GliderPRO/Sources/HouseInfo.c` — 11 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 14 | `kHouseInfoDialogID` | `1001` |  |
| 15 | `kBannerTextItem` | `4` |  |
| 16 | `kLockHouseButton` | `6` |  |
| 17 | `kClearScoresButton` | `9` |  |
| 18 | `kTrailerTextItem` | `11` |  |
| 19 | `kNoPhoneCheck` | `14` |  |
| 20 | `kBannerNCharsItem` | `15` |  |
| 21 | `kTrailerNCharsItem` | `16` |  |
| 22 | `kHouseSizeItem` | `18` |  |
| 23 | `kLockHouseAlert` | `1029` |  |
| 24 | `kZeroScoresAlert` | `1032` |  |

### `GliderPRO/Sources/HouseLegal.c` — 1 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 648 | `kRoomsTimesSuites` | `8192` |  |

### `GliderPRO/Sources/Input.c` — 8 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 14 | `kNormalThrust` | `5` |  |
| 15 | `kHyperThrust` | `8` |  |
| 16 | `kHeliumLift` | `4` |  |
| 17 | `kEscPausePictID` | `1015` |  |
| 18 | `kTabPausePictID` | `1016` |  |
| 19 | `kSavingGameDial` | `1042` |  |
| 385 | `kSaveGameAlert` | `1041` |  |
| 386 | `kYesSaveGameButton` | `1` |  |

### `GliderPRO/Sources/Interactions.c` — 8 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 13 | `kFloorVentLift` | `-6` |  |
| 14 | `kCeilingVentDrop` | `8` |  |
| 15 | `kFanStrength` | `12` |  |
| 16 | `kBatterySupply` | `50` | about 2 rooms worth of thrust |
| 17 | `kHeliumSupply` | `150` |  |
| 18 | `kBandsSupply` | `8` |  |
| 19 | `kFoilSupply` | `8` |  |
| 1738 | `kKillWebbedGlider` | `150` |  |

### `GliderPRO/Sources/InterfaceInit.c` — 4 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 16 | `kHandCursorID` | `128` |  |
| 17 | `kVertCursorID` | `129` |  |
| 18 | `kHoriCursorID` | `130` |  |
| 19 | `kDiagCursorID` | `131` |  |

### `GliderPRO/Sources/Link.c` — 2 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 15 | `kLinkControlID` | `130` |  |
| 16 | `kUnlinkControlID` | `131` |  |

### `GliderPRO/Sources/Main.c` — 1 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 16 | `kPrefsVersion` | `0x0034` |  |

### `GliderPRO/Sources/MainWindow.c` — 4 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 16 | `kMainWindowID` | `128` |  |
| 17 | `kEditWindowID` | `129` |  |
| 18 | `kMenuWindowID` | `130` |  |
| 554 | `kGray2ColorSteps` | `180` |  |

### `GliderPRO/Sources/Map.c` — 9 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 17 | `kMapRoomsHigh` | `9` | was 7 |
| 18 | `kMapRoomsWide` | `9` | was 7 |
| 19 | `kMapScrollBarWidth` | `16` |  |
| 20 | `kHScrollRef` | `5L` |  |
| 21 | `kVScrollRef` | `27L` |  |
| 22 | `kMapGroundValue` | `56` |  |
| 23 | `kNewRoomAlert` | `1004` |  |
| 24 | `kYesDoNewRoom` | `1` |  |
| 25 | `kThumbnailPictID` | `1010` |  |

### `GliderPRO/Sources/Marquee.c` — 2 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 15 | `kMarqueePatListID` | `128` |  |
| 16 | `kHandleSideLong` | `9` |  |

### `GliderPRO/Sources/Menu.c` — 5 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 18 | `kSheWantsNewGame` | `1` |  |
| 19 | `kSheWantsResumeGame` | `2` |  |
| 712 | `kResumeGameDial` | `1025` |  |
| 769 | `kNotInDemoAlert` | `1037` |  |
| 781 | `kNoHighScoreAlert` | `1046` |  |

### `GliderPRO/Sources/Modes.c` — 3 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 408 | `kFramesToBurn` | `60` |  |
| 521 | `kVGliderAppearsComingUp` | `100` |  |
| 552 | `kVGliderAppearsComingDown` | `100` |  |

### `GliderPRO/Sources/Music.c` — 5 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 15 | `kBaseBufferMusicID` | `2000` |  |
| 16 | `kMaxMusic` | `7` |  |
| 17 | `kLastMusicPiece` | `16` |  |
| 18 | `kLastGamePiece` | `6` |  |
| 413 | `kNoMemForMusicAlert` | `1038` |  |

### `GliderPRO/Sources/ObjectAdd.c` — 9 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 15 | `kNoMoreObjectsAlert` | `1008` |  |
| 16 | `kNoMoreSpecialAlert` | `1028` |  |
| 17 | `kMaxSoundTriggers` | `1` |  |
| 18 | `kMaxStairs` | `1` |  |
| 19 | `kMouseholeBottom` | `295` |  |
| 20 | `kFireplaceBottom` | `297` |  |
| 21 | `kManholeSits` | `322` |  |
| 22 | `kGrecoVentTop` | `303` |  |
| 23 | `kSewerBlowerTop` | `292` |  |

### `GliderPRO/Sources/ObjectDraw.c` — 55 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 16 | `k8WhiteColor` | `0` |  |
| 17 | `kYellowColor` | `5` |  |
| 18 | `kGoldColor` | `11` |  |
| 19 | `k8RedColor` | `35` |  |
| 20 | `kPaleVioletColor` | `42` |  |
| 21 | `k8LtTanColor` | `52` |  |
| 22 | `k8BambooColor` | `53` |  |
| 23 | `kDarkFleshColor` | `58` |  |
| 24 | `k8TanColor` | `94` |  |
| 25 | `k8PissYellowColor` | `95` |  |
| 26 | `k8OrangeColor` | `59` |  |
| 27 | `k8BrownColor` | `137` |  |
| 28 | `k8Red4Color` | `143` |  |
| 29 | `k8SkyColor` | `150` |  |
| 30 | `k8EarthBlueColor` | `170` |  |
| 31 | `k8DkRedColor` | `222` |  |
| 32 | `k8DkRed2Color` | `223` |  |
| 33 | `kIntenseGreenColor` | `225` |  |
| 34 | `kIntenseBlueColor` | `235` |  |
| 35 | `k8PumpkinColor` | `101` |  |
| 36 | `k8LtstGrayColor` | `245` |  |
| 37 | `k8LtstGray2Color` | `246` |  |
| 38 | `k8LtstGray3Color` | `43` |  |
| 39 | `k8LtstGray4Color` | `247` |  |
| 40 | `k8LtstGray5Color` | `248` |  |
| 41 | `k8LtGrayColor` | `249` |  |
| 42 | `k8GrayColor` | `250` |  |
| 43 | `k8Gray2Color` | `251` |  |
| 44 | `k8DkGrayColor` | `252` |  |
| 45 | `k8DkGray2Color` | `253` |  |
| 46 | `k8DkGray3Color` | `172` |  |
| 47 | `k8DkstGrayColor` | `254` |  |
| 48 | `k8BlackColor` | `255` |  |
| 71 | `kTikiPoleBase` | `300` |  |
| 149 | `kTableBaseTop` | `296` |  |
| 150 | `kTableShadowTop` | `312` |  |
| 151 | `kTableShadowOffset` | `12` |  |
| 265 | `kBracketInset` | `18` |  |
| 266 | `kShelfDeep` | `4` |  |
| 267 | `kBracketThick` | `5` |  |
| 268 | `kShelfShadowOff` | `12` |  |
| 360 | `kCabinetDeep` | `4` |  |
| 361 | `kCabinetShadowOff` | `6` |  |
| 500 | `kCounterFooterHigh` | `12` |  |
| 501 | `kCounterStripWide` | `6` |  |
| 502 | `kCounterStripTall` | `29` |  |
| 503 | `kCounterPanelDrop` | `12` |  |
| 645 | `kDresserTopThick` | `4` |  |
| 646 | `kDresserCrease` | `9` |  |
| 647 | `kDresserDrawerDrop` | `12` |  |
| 648 | `kDresserSideSpare` | `14` |  |
| 775 | `kTableBaseTop` | `296` |  |
| 776 | `kTableShadowTop` | `312` |  |
| 777 | `kTableShadowOffset` | `12` |  |
| 888 | `kStoolBase` | `304` |  |

### `GliderPRO/Sources/ObjectDraw2.c` — 88 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 19 | `k8WhiteColor` | `0` |  |
| 20 | `kIntenseYellowColor` | `5` |  |
| 21 | `kPaleVioletColor` | `42` |  |
| 22 | `kDarkFleshColor` | `58` |  |
| 23 | `k8TanColor` | `94` |  |
| 24 | `k8PissYellowColor` | `95` |  |
| 25 | `k8BrownColor` | `137` |  |
| 26 | `k8SkyColor` | `150` |  |
| 27 | `k8EarthBlueColor` | `170` |  |
| 28 | `k8DkRed2Color` | `223` |  |
| 29 | `kIntenseGreenColor` | `225` |  |
| 30 | `kIntenseBlueColor` | `235` |  |
| 31 | `k8LtstGrayColor` | `245` |  |
| 32 | `k8LtstGray4Color` | `247` |  |
| 33 | `k8LtstGray5Color` | `248` |  |
| 34 | `k8LtGrayColor` | `249` |  |
| 35 | `k8Gray2Color` | `251` |  |
| 36 | `k8DkGrayColor` | `252` |  |
| 37 | `k8DkGray2Color` | `253` |  |
| 39 | `kBBQMaskID` | `3900` |  |
| 40 | `kUpStairsMaskID` | `3901` |  |
| 41 | `kTrunkMaskID` | `3902` |  |
| 42 | `kMailboxRightMaskID` | `3903` |  |
| 43 | `kMailboxLeftMaskID` | `3904` |  |
| 44 | `kDoorInLeftMaskID` | `3905` |  |
| 45 | `kDoorInRightMaskID` | `3906` |  |
| 46 | `kWindowInLeftMaskID` | `3907` |  |
| 47 | `kWindowInRightMaskID` | `3908` |  |
| 48 | `kHipLampMaskID` | `3909` |  |
| 49 | `kDecoLampMaskID` | `3910` |  |
| 50 | `kGuitarMaskID` | `3911` |  |
| 51 | `kTVMaskID` | `3912` |  |
| 52 | `kVCRMaskID` | `3913` |  |
| 53 | `kStereoMaskID` | `3914` |  |
| 54 | `kMicrowaveMaskID` | `3915` |  |
| 55 | `kFireplaceMaskID` | `3916` |  |
| 56 | `kBearMaskID` | `3917` |  |
| 57 | `kVase1MaskID` | `3918` |  |
| 58 | `kVase2MaskID` | `3919` |  |
| 59 | `kManholeMaskID` | `3920` |  |
| 60 | `kBooksMaskID` | `3922` |  |
| 61 | `kCloudMaskID` | `3921` |  |
| 62 | `kRugMaskID` | `3923` |  |
| 63 | `kChimesMaskID` | `3924` |  |
| 64 | `kCinderMaskID` | `3925` |  |
| 65 | `kFlowerBoxMaskID` | `3926` |  |
| 66 | `kCobwebMaskID` | `3927` |  |
| 67 | `kCobwebPictID` | `3958` |  |
| 68 | `kFlowerBoxPictID` | `3959` |  |
| 69 | `kCinderPictID` | `3960` |  |
| 70 | `kChimesPictID` | `3961` |  |
| 71 | `kRugPictID` | `3962` |  |
| 72 | `kBooksPictID` | `3964` |  |
| 73 | `kCloudPictID` | `3965` |  |
| 74 | `kBulletinPictID` | `3966` |  |
| 75 | `kManholePictID` | `3967` |  |
| 76 | `kVase2PictID` | `3968` |  |
| 77 | `kVase1PictID` | `3969` |  |
| 78 | `kCalendarPictID` | `3970` |  |
| 79 | `kMicrowavePictID` | `3971` |  |
| 80 | `kBearPictID` | `3972` |  |
| 81 | `kFireplacePictID` | `3973` |  |
| 82 | `kOzmaPictID` | `3975` |  |
| 83 | `kWindowExRightPictID` | `3977` |  |
| 84 | `kWindowExLeftPictID` | `3978` |  |
| 85 | `kWindowInRightPictID` | `3979` |  |
| 86 | `kWindowInLeftPictID` | `3980` |  |
| 87 | `kDoorExLeftPictID` | `3981` |  |
| 88 | `kDoorExRightPictID` | `3982` |  |
| 89 | `kDoorInRightPictID` | `3983` |  |
| 90 | `kDoorInLeftPictID` | `3984` |  |
| 91 | `kMailboxRightPictID` | `3985` |  |
| 92 | `kMailboxLeftPictID` | `3986` |  |
| 93 | `kTrunkPictID` | `3987` |  |
| 94 | `kBBQPictID` | `3988` |  |
| 95 | `kStereoPictID` | `3989` |  |
| 96 | `kVCRPictID` | `3990` |  |
| 97 | `kGuitarPictID` | `3991` |  |
| 98 | `kTVPictID` | `3992` |  |
| 99 | `kDecoLampPictID` | `3993` |  |
| 100 | `kHipLampPictID` | `3994` |  |
| 101 | `kFilingCabinetPictID` | `3995` |  |
| 102 | `kDownStairsPictID` | `3996` |  |
| 103 | `kUpStairsPictID` | `3997` |  |
| 105 | `kMailboxBase` | `296` |  |
| 106 | `kMonthStringID` | `1005` |  |
| 502 | `kTrackLightSpacing` | `64` |  |
| 1040 | `kWindowSillThick` | `7` |  |

### `GliderPRO/Sources/ObjectInfo.c` — 46 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 18 | `kBlowerInfoDialogID` | `1007` |  |
| 19 | `kFurnitureInfoDialogID` | `1010` |  |
| 20 | `kSwitchInfoDialogID` | `1011` |  |
| 21 | `kLightInfoDialogID` | `1013` |  |
| 22 | `kApplianceInfoDialogID` | `1014` |  |
| 23 | `kInvisBonusInfoDialogID` | `1015` |  |
| 24 | `kGreaseInfoDialogID` | `1019` |  |
| 25 | `kTransInfoDialogID` | `1022` |  |
| 26 | `kEnemyInfoDialogID` | `1027` |  |
| 27 | `kFlowerInfoDialogID` | `1033` |  |
| 28 | `kTriggerInfoDialogID` | `1034` |  |
| 29 | `kMicrowaveInfoDialogID` | `1035` |  |
| 30 | `kCustPictInfoDialogID` | `1045` |  |
| 31 | `kCustPictIDItem` | `7` |  |
| 32 | `kInitialStateCheckbox` | `6` |  |
| 33 | `kForceCheckbox` | `7` |  |
| 34 | `kDirectionText` | `9` |  |
| 35 | `kLeftFacingRadio` | `16` |  |
| 36 | `kRightFacingRadio` | `17` |  |
| 37 | `kToggleRadio` | `6` |  |
| 38 | `kForceOnRadio` | `7` |  |
| 39 | `kForceOffRadio` | `8` |  |
| 40 | `kDelay3Item` | `6` |  |
| 41 | `kDelayItem` | `8` |  |
| 42 | `kDelayLabelItem` | `9` |  |
| 43 | `k100PtRadio` | `6` |  |
| 44 | `k300PtRadio` | `7` |  |
| 45 | `k500PtRadio` | `8` |  |
| 46 | `kGreaseItem` | `6` |  |
| 47 | `kLinkTransButton` | `6` |  |
| 48 | `kInitialStateCheckbox3` | `13` |  |
| 49 | `kTransRoomText` | `8` |  |
| 50 | `kTransObjectText` | `9` |  |
| 51 | `kKillBandsCheckbox` | `8` |  |
| 52 | `kKillBatteryCheckbox` | `9` |  |
| 53 | `kKillFoilCheckbox` | `10` |  |
| 54 | `kDelay2Item` | `7` |  |
| 55 | `kDelay2LabelItem` | `8` |  |
| 56 | `kDelay2LabelItem2` | `9` |  |
| 57 | `kInitialStateCheckbox2` | `10` |  |
| 58 | `kRadioFlower1` | `6` |  |
| 59 | `kRadioFlower6` | `11` |  |
| 60 | `kFlowerCancel` | `12` |  |
| 61 | `kGotoButton1` | `11` |  |
| 62 | `kGotoButton2` | `14` |  |
| 122 | `kArrowheadLength` | `4` |  |

### `GliderPRO/Sources/ObjectRects.c` — 7 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 13 | `kFloorColumnWide` | `4` |  |
| 14 | `kCeilingColumnWide` | `24` |  |
| 15 | `kFanColumnThick` | `16` |  |
| 16 | `kFanColumnDown` | `20` |  |
| 17 | `kDeadlyFlameHeight` | `24` |  |
| 18 | `kStoolThick` | `25` |  |
| 19 | `kShredderActiveHigh` | `40` |  |

### `GliderPRO/Sources/Objects.c` — 1 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 12 | `kMaxTempManholes` | `8` |  |

### `GliderPRO/Sources/Play.c` — 6 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 18 | `kHouseBannerAlert` | `1009` |  |
| 19 | `kInitialGliders` | `2` |  |
| 20 | `kRingDelay` | `90` |  |
| 21 | `kRingSpread` | `25000` | 25000 |
| 22 | `kRingBaseDelay` | `5000` | 5000 |
| 23 | `kChimeDelay` | `180` |  |

### `GliderPRO/Sources/Player.c` — 23 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 13 | `kGravity` | `3` |  |
| 14 | `kHImpulse` | `2` |  |
| 15 | `kVImpulse` | `2` |  |
| 16 | `kMaxHVel` | `16` |  |
| 17 | `kShredderCountdown` | `-68` |  |
| 317 | `kClimbStairsSpeed` | `-4` |  |
| 385 | `kVClimbStairsSpeed` | `-4` |  |
| 386 | `kHClimbStairsSpeed` | `-4` |  |
| 433 | `kVDropStairsSpeed` | `4` |  |
| 434 | `kHDropStairsSpeed` | `4` |  |
| 514 | `kVDropStairsSpeed` | `4` |  |
| 515 | `kHDropStairsSpeed` | `4` |  |
| 659 | `kVDropDuctSpeed` | `4` |  |
| 756 | `kVRiseDuctSpeed` | `-4` |  |
| 853 | `kHPushMailSpeed` | `-4` |  |
| 891 | `kHPushMailRtSpeed` | `4` |  |
| 929 | `kVDropStairsSpeed` | `4` |  |
| 962 | `kHMailPullSpeed` | `4` |  |
| 963 | `kVMailDropSpeed` | `2` |  |
| 1053 | `kHMailPullRtSpeed` | `-4` |  |
| 1054 | `kVMailDropSpeed` | `2` |  |
| 1262 | `kDropShredSlow` | `1` |  |
| 1263 | `kDropShredFast` | `4` |  |

### `GliderPRO/Sources/Prefs.c` — 7 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 18 | `kPrefCreatorType` | `'ozm5'` |  |
| 19 | `kPrefFileType` | `'gliP'` |  |
| 20 | `kPrefFileName` | `"\pGlider Prefs"` |  |
| 21 | `kDefaultPrefFName` | `"\pPreferences"` |  |
| 22 | `kPrefsStringsID` | `160` |  |
| 23 | `kNewPrefsAlertID` | `160` |  |
| 24 | `kPrefsFNameIndex` | `1` |  |

### `GliderPRO/Sources/Render.c` — 1 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 20 | `kMaxGarbageRects` | `48` |  |

### `GliderPRO/Sources/Room.c` — 2 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 16 | `kDeleteRoomAlert` | `1005` |  |
| 17 | `kYesDoDeleteRoom` | `1` |  |

### `GliderPRO/Sources/RoomGraphics.c` — 1 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 16 | `kManholeThruFloor` | `3957` |  |

### `GliderPRO/Sources/RoomInfo.c` — 17 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 18 | `kRoomInfoDialogID` | `1003` |  |
| 19 | `kOriginalArtDialogID` | `1016` |  |
| 20 | `kNoPICTFoundAlert` | `1036` |  |
| 21 | `kRoomNameItem` | `3` |  |
| 22 | `kRoomLocationBox` | `6` |  |
| 23 | `kRoomTilesBox` | `10` |  |
| 24 | `kRoomPopupItem` | `11` |  |
| 25 | `kRoomDividerLine` | `12` |  |
| 26 | `kRoomTilesBox2` | `15` |  |
| 27 | `kRoomFirstCheck` | `17` |  |
| 28 | `kLitUnlitText` | `18` |  |
| 29 | `kMiniTileWide` | `16` |  |
| 30 | `kBoundsButton` | `19` |  |
| 31 | `kOriginalArtworkItem` | `19` |  |
| 32 | `kPICTIDItem` | `5` |  |
| 33 | `kFloorSupportCheck` | `12` |  |
| 379 | `kBackgroundsMenuID` | `140` |  |

### `GliderPRO/Sources/RubberBands.c` — 3 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 12 | `kRubberBandVelocity` | `20` |  |
| 13 | `kBandFallCount` | `4` |  |
| 14 | `kKillBandMode` | `-1` |  |

### `GliderPRO/Sources/SavedGames.c` — 2 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 14 | `kSavedGameVersion` | `0x0200` |  |
| 154 | `kSavedGameErrorAlert` | `1044` |  |

### `GliderPRO/Sources/Scoreboard.c` — 11 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 15 | `kGrayBackgroundColor` | `251` |  |
| 16 | `kGrayBackgroundColor4` | `10` |  |
| 17 | `kFoilBadge` | `0` |  |
| 18 | `kBandsBadge` | `1` |  |
| 19 | `kBatteryBadge` | `2` |  |
| 20 | `kHeliumBadge` | `3` |  |
| 21 | `kScoreRollAmount` | `13` |  |
| 74 | `kFoilLow` | `2` | 25% |
| 75 | `kBatteryLow` | `17` | 25% |
| 76 | `kHeliumLow` | `-38` | 25% |
| 77 | `kBandsLow` | `2` | 25% |

### `GliderPRO/Sources/SelectHouse.c` — 17 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 20 | `kLoadHouseDialogID` | `1000` |  |
| 21 | `kDispFiles` | `12` |  |
| 22 | `kLoadTitlePictItem` | `3` |  |
| 23 | `kLoadNameFirstItem` | `5` |  |
| 24 | `kLoadNameLastItem` | `16` |  |
| 25 | `kLoadIconFirstItem` | `17` |  |
| 26 | `kLoadIconLastItem` | `28` |  |
| 27 | `kScrollUpItem` | `29` |  |
| 28 | `kScrollDownItem` | `30` |  |
| 29 | `kLoadTitlePict1` | `1001` |  |
| 30 | `kLoadTitlePict8` | `1002` |  |
| 31 | `kDefaultHousePict1` | `1003` |  |
| 32 | `kDefaultHousePict8` | `1004` |  |
| 33 | `kGrayedOutUpArrow` | `1052` |  |
| 34 | `kGrayedOutDownArrow` | `1053` |  |
| 35 | `kMaxExtraHouses` | `8` |  |
| 559 | `kMaxDirectories` | `32` |  |

### `GliderPRO/Sources/Settings.c` — 45 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 17 | `kMainPrefsDialID` | `1012` |  |
| 18 | `kDisplayPrefsDialID` | `1017` |  |
| 19 | `kSoundPrefsDialID` | `1018` |  |
| 20 | `kControlPrefsDialID` | `1023` |  |
| 21 | `kBrainsPrefsDialID` | `1024` |  |
| 22 | `kDisplayButton` | `3` |  |
| 23 | `kSoundButton` | `4` |  |
| 24 | `kControlsButton` | `5` |  |
| 25 | `kBrainsButton` | `6` |  |
| 26 | `kDisplay1Item` | `3` |  |
| 27 | `kDisplay3Item` | `4` |  |
| 28 | `kDisplay9Item` | `5` |  |
| 29 | `kDoColorFadeItem` | `9` |  |
| 30 | `kCurrentDepth` | `10` |  |
| 31 | `k256Depth` | `11` |  |
| 32 | `k16Depth` | `12` |  |
| 33 | `kDispDefault` | `15` |  |
| 34 | `kUseQDItem` | `16` |  |
| 35 | `kUseScreen2Item` | `17` |  |
| 36 | `kSofterItem` | `4` |  |
| 37 | `kLouderItem` | `5` |  |
| 38 | `kVolNumberItem` | `7` |  |
| 39 | `kIdleMusicItem` | `8` |  |
| 40 | `kPlayMusicItem` | `9` |  |
| 41 | `kSoundDefault` | `13` |  |
| 42 | `kRightControl` | `5` |  |
| 43 | `kLeftControl` | `6` |  |
| 44 | `kBattControl` | `7` |  |
| 45 | `kBandControl` | `8` |  |
| 46 | `kControlDefaults` | `13` |  |
| 47 | `kESCPausesRadio` | `14` |  |
| 48 | `kTABPausesRadio` | `15` |  |
| 49 | `kMaxFilesItem` | `5` |  |
| 50 | `kQuickTransitCheck` | `7` |  |
| 51 | `kDoZoomsCheck` | `8` |  |
| 52 | `kBrainsDefault` | `9` |  |
| 53 | `kDoDemoCheck` | `10` |  |
| 54 | `kDoBackgroundCheck` | `11` |  |
| 55 | `kDoErrorCheck` | `12` |  |
| 56 | `kDoPrettyMapCheck` | `13` |  |
| 57 | `kDoBitchDlgsCheck` | `14` |  |
| 1267 | `kNormalSettingsIcon` | `1010` |  |
| 1268 | `kInvertedSettingsIcon` | `1014` |  |
| 1397 | `kAllDefaultsButton` | `11` |  |
| 1476 | `kChangesEffectAlert` | `1040` |  |

### `GliderPRO/Sources/Sound.c` — 5 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 14 | `kBaseBufferSoundID` | `1000` |  |
| 15 | `kMaxSounds` | `64` |  |
| 16 | `kNoSoundPlaying` | `-1` |  |
| 518 | `kNoMemForSoundsAlert` | `1039` |  |
| 529 | `kNoSoundManager3Alert` | `1030` |  |

### `GliderPRO/Sources/StringUtils.c` — 2 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 304 | `kChooserStringID` | `-16096` |  |
| 323 | `kLocalizedStringsID` | `150` |  |

### `GliderPRO/Sources/StructuresInit.c` — 20 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 19 | `kShadowPictID` | `3998` |  |
| 20 | `kBlowerPictID` | `4000` |  |
| 21 | `kFurniturePictID` | `4001` |  |
| 22 | `kBonusPictID` | `4002` |  |
| 23 | `kSwitchPictID` | `4003` |  |
| 24 | `kLightPictID` | `4004` |  |
| 25 | `kAppliancePictID` | `4005` |  |
| 26 | `kPointsPictID` | `4006` |  |
| 27 | `kRubberBandsPictID` | `4007` |  |
| 28 | `kTransportPictID` | `4008` |  |
| 29 | `kToastPictID` | `4009` |  |
| 30 | `kShreddedPictID` | `4010` |  |
| 31 | `kBalloonPictID` | `4011` |  |
| 32 | `kCopterPictID` | `4012` |  |
| 33 | `kDartPictID` | `4013` |  |
| 34 | `kBallPictID` | `4014` |  |
| 35 | `kDripPictID` | `4015` |  |
| 36 | `kEnemyPictID` | `4016` |  |
| 37 | `kFishPictID` | `4017` |  |
| 39 | `kBadgePictID` | `1996` |  |

### `GliderPRO/Sources/StructuresInit2.c` — 3 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 20 | `kAngelPictID` | `1019` |  |
| 21 | `kSupportPictID` | `1999` |  |
| 22 | `kClutterPictID` | `4018` |  |

### `GliderPRO/Sources/Tools.c` — 32 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 15 | `kToolsHigh` | `4` |  |
| 16 | `kToolsWide` | `4` |  |
| 17 | `kTotalTools` | `16` | kToolsHigh * kToolsWide |
| 18 | `kPopUpControl` | `129` |  |
| 19 | `kFirstBlower` | `1` |  |
| 20 | `kLastBlower` | `15` |  |
| 21 | `kBlowerBase` | `1` |  |
| 22 | `kFirstFurniture` | `1` |  |
| 23 | `kLastFurniture` | `15` |  |
| 24 | `kFurnitureBase` | `21` |  |
| 25 | `kFirstBonus` | `1` |  |
| 26 | `kLastBonus` | `15` |  |
| 27 | `kBonusBase` | `41` |  |
| 28 | `kFirstTransport` | `1` |  |
| 29 | `kLastTransport` | `12` |  |
| 30 | `kTransportBase` | `61` |  |
| 31 | `kFirstSwitch` | `1` |  |
| 32 | `kLastSwitch` | `9` |  |
| 33 | `kSwitchBase` | `81` |  |
| 34 | `kFirstLight` | `1` |  |
| 35 | `kLastLight` | `8` |  |
| 36 | `kLightBase` | `101` |  |
| 37 | `kFirstAppliance` | `1` |  |
| 38 | `kLastAppliance` | `14` |  |
| 39 | `kApplianceBase` | `121` |  |
| 40 | `kFirstEnemy` | `1` |  |
| 41 | `kLastEnemy` | `9` |  |
| 42 | `kEnemyBase` | `141` |  |
| 43 | `kFirstClutter` | `1` |  |
| 44 | `kLastClutter` | `15` |  |
| 45 | `kClutterBase` | `161` |  |
| 46 | `kToolsPictID` | `1011` |  |

### `GliderPRO/Sources/Transitions.c` — 4 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 20 | `kMaxColumnsWide` | `96` |  |
| 21 | `kChipHigh` | `20` |  |
| 22 | `kChipWide` | `16` |  |
| 76 | `kWipeRectThick` | `4` |  |

### `GliderPRO/Sources/Triggers.c` — 1 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 12 | `kMaxTriggers` | `16` |  |

### `GliderPRO/Sources/Utilities.c` — 3 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 137 | `rDeathAlertID` | `170` | alert res. ID for death error |
| 138 | `rErrTitleID` | `170` | string ID for death error title |
| 139 | `rErrMssgID` | `171` | string ID for death error message |

### `GliderPRO/Sources/Validate.c` — 8 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 16 | `kEncryptMask` | `0x05218947` |  |
| 17 | `kLegalVolumeCreation` | `0xAA2D3E41` |  |
| 18 | `kMasterDialogID` | `1026` |  |
| 19 | `kMasterFinderButton` | `1` |  |
| 20 | `kMasterNetOnlyButton` | `2` |  |
| 21 | `kMasterUserBalloon` | `3` |  |
| 22 | `kMasterTitleLeft` | `6` |  |
| 23 | `kMasterTitleTop` | `16` |  |

### `GliderPRO/Sources/WindowUtils.c` — 2 defines

| Line | Name | Value | Comment |
|---:|---|---|---|
| 14 | `kFloatingKind` | `2048` |  |
| 15 | `kMessageWindowTall` | `48` |  |

---

## Open questions

Things this document could **not** settle from the source and the shipped data. Each is
phrased so it can be closed later by a specific experiment.

1. **Is the `floor + 7` in `CheckDuplicateFloorSuite` an off-by-one?**
   `GliderPRO/Sources/HouseLegal.c:665` computes `bitPlace = ((floor + 7) * 128) + suite`,
   but the on-disk encoding bias is `kNumUndergroundFloors` = **8**
   (`GliderPRO/Headers/GliderDefines.h:535`), and the validated floor range is
   **−7 … 56** (`GliderPRO/Sources/HouseLegal.c:804-805`). With bias 7 and floor ≥ −7 the
   bitmap index is ≥ 0, so it is *self-consistent*; but the encode/decode pair uses 8. Is
   floor −8 legal on disk and merely unreachable through the editor, or is −7 the true
   floor? **Experiment:** hand-craft a house with `floor == -8` and see whether
   `HouseLegal` rejects it before `CheckDuplicateFloorSuite` can underflow. No shipped
   house has a negative floor at all, so the data cannot answer this.
   *(§22.6, §25(p))*

2. **Is `GliderPRO/Sources/Dynamics3.c:519` (`where->left + 10, where->top + 8`) missing
   `playOriginH`/`playOriginV`, or is `kFish` genuinely in a different coordinate space?**
   Its six siblings at `:248-249`, `:268-269`, `:288-289`, `:331-332`, `:354-355`,
   `:374-375` all add the play origin. **Experiment:** place a fish in a room with a
   non-zero `playOriginH` (i.e. a room narrower than the window) and observe whether it
   renders offset. *(§24.6)*

3. ~~**What is the on-disk layout of a `'bnds'` resource?**~~ **Resolved.** `boundsType` is
   4 bytes (`GliderPRO/Headers/GliderStructs.h:266-272`) and 8 of the 22 distributed
   houses ship `'bnds'` resources — 70 of them in total — each exactly 4 bytes of
   `0x00`/`0x01` in `left, top, right, bottom` order, matching the declaration and the
   read order in `GetOriginalBounding`. There is no endianness question (four single
   bytes). The path is live whenever `thisRoom->bounds == 0` for a user background;
   `thisRoom->bounds != 0` short-circuits it (`GliderPRO/Sources/Room.c:826-829`).
   *(§22.5, §25(u))*

4. **Were the 20 dead `k*MaskID` PICTs ever drawn?** All 20 are absent from
   `GliderPRO/Glider PRO.r` (only 3903, 3904, 3912–3915, 3921, 3927 of the 3900-block
   exist — §25(f)). Whether the objects were originally masked and later switched to a
   different technique, or the constants were written speculatively, cannot be determined
   from a single release.

5. **Are `unusedShort` / `unusedBoolean` / `unusedByte` / `unusedLong` vestigial or
   forward-looking?** Observed values are garbage across all 22 shipped houses
   (`unusedShort` = 147 in *In The Mirror*, **13107 = 0x3333** in *California or Bust!*,
   **−30082 = 0x8A7E** in *Art Museum*; `unusedBoolean` ∈ {0, 2, 14, 30, 37, 185, 255}), which is
   consistent with `InitializeEmptyHouse` using `NewHandle` rather than `NewHandleClear`
   (`GliderPRO/Sources/House.c:116`). `0x3333` looks like a heap fill pattern rather than
   data. **A Go port must treat them as reserved and ignore them on read.** *(§22.7,
   §25(s))*

6. **Is `roomType.bounds` bit 5 one flag or two?** It is written by the editor as "floor
   support" (`GliderPRO/Sources/RoomInfo.c:767-780`) and read by the renderer as
   `bounds & 32` meaning "isStructure" (`GliderPRO/Sources/Room.c:777`). Deliberate
   overload or an accidental collision? **Experiment:** toggle the floor-support checkbox
   in the editor on a user background and see whether the room's structure rendering
   changes. *(§22.5)*

7. **Do art slices 1, 2, 5, 6 of the `kRoof` background actually depict 45° slopes?** The
   line equations in `CheckRoofCollision` say so (§24.2.3) and the derived apex matches
   `kRoofLimit` = 122 exactly, but this document did not decode the PICT pixels to
   confirm. **Experiment:** render `PICT` for `kRoof` and measure the roof edge at
   x = 0, 63, 64, 127.

8. **Was `game2Type` / the external `.gliG` saved game ever functional?** `SaveGame2`'s
   entire body is commented out (`GliderPRO/Sources/SavedGames.c:33`…`:147`) with
   `// Add NavServices later.` at `:32`, and `OpenSavedGame` starts with `return false;`
   at `:169`. The `kSavedGameVersion 0x0200` at `:14` and the fully-written
   `byteCount = sizeof(game2Type) + sizeof(savedRoom) * numRooms;` at `:56` suggest it
   worked before the Navigation Services migration. No `.gliG` file ships, so the
   serialised layout is unverifiable. *(§22.8, §25(v))*

9. **Why does `Sampler.binhex` have 2 trailing bytes?** Its data fork is **1564** bytes
   but `866 + 348 × 2 = 1562` (§22.7). Truncated/padded write, BinHex artefact, or
   deliberate? The other four houses are exact. **Experiment:** dump the last 8 bytes and
   check whether they are zero or the tail of a third `roomType`.

10. **Are all house `'snd '` resources Sound Manager format 1?** The loader skips a
    20-byte header, which is only correct for format 1
    (§16). The three shipped custom-sound houses were inventoried by *name* but their
    `'snd '` payload headers were not parsed. **Experiment:** parse the first 6 bytes of
    each `'snd '` in *In The Mirror* and *California or Bust!* and check `format == 1`.

11. **Is `evenFrame = true` at `GliderPRO/Sources/Dynamics3.c:474` and `:524` truly
    dead, or does some path read `evenFrame` between the write and the next
    `evenFrame = !evenFrame` at `GliderPRO/Sources/Play.c:435`?** Static reading says the
    triangular loop only touches the local `lilFrame`, so the write's only effect is to
    force the global true for the remainder of the frame. Confirming it is *observably*
    dead needs a trace. *(§25(h))*

12. **Does the `BUILD_ARCADE_VERSION 1` / `COMPILENOCP` / `COMPILEQT` configuration in the
    GPL drop match the 1.0.4 binary that shipped?** `CREATEDEMODATA`, `COMPILEDEMO` and
    `CAREFULDEBUG` are commented out at `GliderPRO/Headers/GliderDefines.h:11-13`;
    `COMPILENOCP` (`:14`), `COMPILEQT` (`:15`) and `BUILD_ARCADE_VERSION` = 1 (`:16`) are
    active. `BUILD_ARCADE_VERSION` gates behaviour at `GliderPRO/Sources/Input.c:192`,
    `GliderPRO/Sources/Main.c:347`, `:366`, `GliderPRO/Sources/Events.c:191`,
    `GliderPRO/Sources/Play.c:138` (`#if !`), `:512`, `:556`, `:806` (`#if !`). Whether
    the retail 1.0.4 was built with these exact settings cannot be checked without the
    original binary. *(§2)*

13. **What is the authoritative index order of `localNumbers[9]`?** It is declared
    `extern short localNumbers[9];` at `GliderPRO/Sources/Transit.c:23` and indexed by the
    `kAbove`/`kBelow`/`kLeftRoom`/`kRightRoom`/… direction constants. The 9th slot (self)
    versus the 8 neighbours is inferred from usage, not from a comment.

14. **Is there any bound on `houseType.nRooms` at load time?** No `kMaxRooms` constant
    exists anywhere (`grep` returns nothing). `HouseLegal.c` bounds *coordinates* to
    128 × 64 and uses an 8192-cell duplicate bitmap, but nothing bounds the room count
    itself; `HouseLegal.c:630-631` derives it from the handle size. A malformed house with
    a huge `nRooms` and a small data fork will read past the handle. *(§25(r))*

15. **Does `kInvisBonus` with `data.c.points == 250` occur in any shipped house?** The 250
    branch of `AddFlyingPoint` (`GliderPRO/Sources/DynamicMaps.c:228-231`) is reachable
    only that way (§24.2.4). **Experiment:** scan all houses' object records for
    `what == kInvisBonus && data.c.points == 250`.

## Porting notes

Ordered from "will break everything" to "will bite you later".

### P1. Byte layout: `#pragma options align=mac68k` means `long` is 2-byte aligned

This is the single most consequential ABI fact in the whole codebase
(`GliderPRO/Headers/GliderStructs.h`, `#pragma options align=mac68k` … `align=reset`).
Under mac68k alignment a `long` needs only 2-byte alignment, not 4. Every on-disk size in
§22 depends on it:

- `prefsInfo` is **226** bytes, with exactly **one** byte of interior padding at offset
  145. Under a normal 4-byte-aligned ABI that pad would be 3 bytes and the file 228.
- `houseType`'s header is **866** bytes — independently confirmed by
  `GliderPRO/Sources/HouseLegal.c:630-631`
  (`countedRooms = (GetHandleSize(…) - sizeof(houseType)) / sizeof(roomType)`) and by
  four of the five shipped houses satisfying `866 + 348 × nRooms == dataForkLength`
  exactly.
- `roomType` is **348**, `objectType` **12**, all nine union arms **10**,
  `scoresType` **292**, `savedRoom` **292**, `gameType` **40**.

**In Go:** do not use `encoding/binary` on a struct and hope. Write explicit
field-by-field readers/writers with hard-coded offsets from the §22 tables, and add a test
that asserts `866 + 348*nRooms == len(dataFork)` for each shipped house. Reproducing this
with a Go struct plus `unsafe.Sizeof` is not possible — Go has no mac68k alignment mode.

### P2. Everything on disk is big-endian

68k and PowerPC are both big-endian. Every `short` and `long` in a house file, a prefs
file, a `'snd '`, a `PICT` and a resource-fork map is big-endian. Use
`binary.BigEndian` everywhere. The values that make this checkable: house `version` is
`0x0200`, and the *Demo House* data fork is exactly **16526** bytes
(`GliderPRO/Sources/HouseIO.c:349` asserts `byteCount != 16526L`) with `nRooms == 45`
(`:382`), `firstRoom == 0` (`:412`) and locked (`:404`) — and 866 + 348 × 45 = 16526.

### P3. `Point` is `{short v; short h;}` — **v first**

QuickDraw's `Point` puts the *vertical* coordinate first. `Rect` is
`{short top, left, bottom, right;}`. Both orders are the opposite of the `(x, y)` /
`(x, y, w, h)` convention a Go programmer reaches for, and both appear in on-disk data
(`objectType.data.*.topLeft` is a `Point`). Getting this wrong silently transposes every
object in every room.

### P4. The nine union arms all alias the same 10 bytes

`objectType.data` is a union of nine 10-byte structs at offset 2 of a 12-byte record
(§22.2). The program **relies** on the aliasing:

- `GliderPRO/Sources/Scrap.c:145-154` writes the *wrong arm* and works anyway because
  `data.d.where` and `data.e.where` are both `short` at offset 8 (§25(n)).
- `Interactions.c:1171` reads `data.g.byte0` as a bitmask while other code reads the same
  byte as a different field.

**In Go:** model `data` as a fixed `[10]byte` (or a single flat struct with every field
name mapped to an explicit offset), *not* as an interface or a sum type. If you use
distinct Go structs per arm, `Scrap.c`'s swapped branch becomes a real bug and pasting a
room silently destroys every intra-room link.

### P5. Frame timing is `kTicksPerFrame = 2` ticks on a 60.15 Hz tick — 30.07 fps

`kTicksPerFrame` = 2 (`GliderPRO/Headers/GliderDefines.h:533`). The Mac tick is
1/60.15 s, so a frame is ≈ 33.25 ms and the frame rate is ≈ **30.07 fps**, not 30.000.
All velocities in §10 and §24 are **pixels per frame**, all timers are **frames**. Do not
convert to per-second units and do not run the simulation at the display refresh rate:
run a fixed 30.07 Hz (or exactly 2/60.15 s) accumulator and render whatever falls out.
`(delay * 6) / kTicksPerFrame` (`Dynamics3.c:316`, `:401`, `:426`, `:456`, `:504`, `:521`)
is integer division — with `kTicksPerFrame` = 2 it is `delay * 3`.

### P6. All arithmetic is 16-bit `short` with C truncation

Velocities, positions, timers and frame counters are `short`. Divisions are integer and
truncate **toward zero**:

- `hVel /= 2`, `hVel /= 4` (§24.2.4) drop the odd pixel — momentum is not conserved.
- `vVel = -((vVel * 3) / 4)` (`GliderPRO/Sources/Dynamics2.c:392`) is the ball's
  restitution. Go's `/` also truncates toward zero, so `-(v*3/4)` is bit-identical.
  Floating point or floor division is not.
- `>> 6` (`Interactions.c:200` etc.) and `>> 3` (`:1750`, `:1752`) are **arithmetic**
  shifts on signed values, so they round toward **−∞**, unlike `/ 64` and `/ 4`. For
  negative coordinates — which happen, e.g. `kNoLeftWallLimit` = −24 — `x >> 6` and
  `x / 64` differ. Use `x >> 6` in Go on a signed integer to match.

Use `int16` (or `int` with explicit `int16` truncation at every store) for anything the C
declares `short`, or you will lose the overflow behaviour of the tile and timer maths.

### P7. Mac Toolbox facilities and their Go replacements

| Toolbox facility | What it does here | Go replacement |
|---|---|---|
| **QuickDraw `CopyBits` / `CopyMask`** | all blitting, with a separate 1-bit mask GWorld per sprite strip | pre-multiplied RGBA sprites; one rect table serves both `src` and `mask` (§24.3) |
| **GWorlds (`CGrafPtr`, `CreateOffScreenGWorld`)** | `workSrcMap` (composite), `backSrcMap` (background), one per sprite strip | plain `image.RGBA` / GPU textures |
| **8-bit indexed colour + `clut`** | `kPreferredDepth` = 8 (`GliderPRO/Headers/Externs.h:15`); all art is palettised | bake the `clut` into RGBA at load; §19 has the empirical palette evidence |
| **Resource Manager (`GetResource`, `FSpOpenResFile`, `UseResFile`)** | all art, sound and dialogs live in resource forks — the app's and each house's | parse the classic resource-fork map yourself (§22.11); note an **empty resource fork is exactly 286 bytes** |
| **Sound Manager `'snd '`** | format-1 resources, loader skips a **20-byte** header | decode to PCM at load; verify format 1 (open question 10) |
| **QuickTime `Movies.h`** | in-room TV playback (`theMovie`, `hasMovie`, `tvInRoom` — `GliderPRO/Headers/GliderVars.h:42-44`) | either a video decoder or drop the feature; `COMPILEQT` gates it |
| **`GetKeys` / `KeyMap` / `BitTst`** | key state as a 128-bit bitmap; `gliderType.leftKey` etc. hold **KeyMap bit offsets**, not virtual key codes (§20, §23.1) | keep the KeyMap-offset numbering in prefs for file compatibility, translate at the input layer |
| **`FindFolder` / `FSpCreate` / `FSpOpenDF` / `FSWrite` / `SetEOF`** | prefs in the Preferences folder, houses anywhere | `os` + `filepath`; `FSSpec` is 70 bytes (`{short vRefNum; long parID; Str63 name;}`) and is **stored in prefs**, so a Go port must decide what to write there |
| **`NewHandle` / `HLock` / `HGetState` / `PtrAndHand` / `SetHandleSize`** | the house is one relocatable block; adding a room is `PtrAndHand(…, sizeof(roomType))` (`GliderPRO/Sources/Room.c:200`) | `[]roomType` slice + `append`; but keep the *file* layout of P1 |
| **Pascal strings** | `Str15`=17, `Str27`=28, `Str31`=32, `Str32`=33, `Str63`=64, `Str255`=256 bytes; length byte first | fixed-size `[N]byte` on disk, `string` in memory; **truncate, do not error**, on overflow |
| **`DebugStr`** | `DebugStr("\pBlew array")` at `GliderPRO/Sources/HouseLegal.c:668` is a no-op in release | return an error and reject the house (§25(p)) |
| **MacRoman text encoding** | room names, house names, object strings | transcode MacRoman → UTF-8 on read, back on write |

### P8. The `thisRoom` copy-in / copy-out discipline

`thisHouse` is a `Handle` to the whole file; `thisRoom` is a **separate heap copy** of one
room allocated by `thisRoom = (roomPtr)NewPtr(sizeof(roomType));`
(`GliderPRO/Sources/StructuresInit2.c:193`), synchronised by `CopyThisRoomToRoom()` and
`CopyRoomToThisRoom(n)`. Every editor mutation writes `thisRoom` and is **lost** unless
`CopyThisRoomToRoom()` runs before the current room changes or the file is written. In Go
the natural thing is to hand out a `*Room` pointer into the slice — which changes the
semantics of every existing save/discard path. If you do that, audit every
`CopyThisRoomToRoom` call site and every place the original relies on the copy *not*
being written back.

### P9. Reproduce the bugs, not the intent

Shipped houses were authored against the behaviour in §25, not against what the code
meant to do. In particular, keep:

- the **double `foilTotal--`** on a side hit against `kDissolveIt` (§25(m));
- the three **`evenFrame = true`** writes (§25(h));
- the **`kDirt` `leftThresh`/`leftOpen` mismatch** (§25(q));
- the **shadow pinned at y = 306** (§25(g));
- the **`- 2` room-entry offset** and the editor's `48 × 16` marker (§25(j));
- **no `vVel` clamp** (§25(c)).

Make each one a named, commented, individually testable quirk rather than an accident, so
a later maintainer does not "fix" it.

### P10. The 5-pixel inset is the game's feel

If only one number from §24 survives the port, make it this one: the glider's `dest` is
48 × 20 but its collision box is inset 5 px on every side, giving **38 × 10** — half the
sprite's height (`GliderPRO/Sources/Interactions.c:60-63`, `:112-115`). Four *different*
predicates use *different* insets (§10.4): `GliderHitTop` always insets 5,
`SectGlider` insets 5 only when `scrutinize` is true and adds `+6` to the top first if
burning, `GliderInRect` is a containment test with **no** inset, and `BounceGlider` uses
the raw `dest`. Pick one and the game stops feeling like Glider PRO.

### P11. Text and file-format hygiene for reading the original

- Sources are **CR-only** (`\r`). `wc -l` reports 0 and most tools see one giant line.
  Convert with `tr '\r' '\n'` before anything else; all line numbers in this document are
  post-conversion.
- Sources are **MacRoman**, so GNU `grep` treats them as binary. Use `grep -a`, or
  `iconv -f MAC -t UTF-8` first.
- `GliderPRO/Glider PRO.r` is a 15 MB **LF** Rez text dump of the app's resource fork
  (199843 lines) — the easiest way to enumerate resource IDs
  (`grep -ao "data 'PICT' ([0-9]*)"` yields 152 PICTs).
- `GliderPRO/Houses/*.binhex` are **BinHex 4.0**. A decoder must include all 64 alphabet
  characters; a missing one silently corrupts everything after the first occurrence
  (this bit the author of this document once — assert `len(ALPHA) == 64`).
- Houses keep level data in the **data fork** and custom art/sound in the **resource
  fork**. An empty resource fork is **286** bytes, which is how *Sampler* is known to have
  no custom assets.
