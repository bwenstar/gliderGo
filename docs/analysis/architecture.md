# Glider PRO — Application Architecture, Game States and Main Loop

## Scope

This document specifies the *application-level* skeleton of Glider PRO: how the
program boots, what global states it can be in, how the outer (Toolbox) event
loop and the inner (gameplay) loop are structured, exactly how frames are paced,
how pause / suspend / resume / quit work, how the attract ("demo") mode is
driven, what the program assumes about screen size and window geometry, and
which globals constitute genuine application state.

It deliberately does **not** specify: glider physics (`Player.c`, `Dynamics*.c`),
object semantics (`Objects.c`, `Trip.c`, `Triggers.c`), the room editor UI
(`Map.c`, `Tools.c`, `ObjectEdit.c`), file formats (`HouseIO.c`), sound/music
synthesis (`Sound.c`, `Music.c`), or the scoreboard renderer (`Scoreboard.c`)
beyond the points where the main loop calls into them. Those are separate
documents. Where the main loop calls into them, the call site and its ordering
constraint *are* specified here, because ordering is load-bearing.

All line citations are of the form `GliderPRO/Sources/File.c:NNN` and are
relative to the repository root the repository root. The original
sources use classic-Mac CR-only line endings; a CR→LF conversion is byte-for-byte
length-preserving and 1:1 on line boundaries (verified: every `.c` file contains
zero `0x0A` bytes, and each converted file has exactly as many `\n` as the
original had `\r`, with identical byte length), so the line numbers below are
valid against the pristine originals as well.

## Sources read

Read in full (converted to LF in a scratch directory; originals never modified):

| File | Lines | Why |
|---|---|---|
| `GliderPRO/Sources/Main.c` | 386 | `main()`, startup order, prefs load/save, shutdown |
| `GliderPRO/Sources/Events.c` | 573 | outer event loop `HandleEvent`, all event handlers, attract trigger |
| `GliderPRO/Sources/Play.c` | 821 | `NewGame`, `PlayGame` inner loop, `DoDemoGame`, in-game event pump |
| `GliderPRO/Sources/Render.c` | 772 | `RenderFrame` — the frame pacer; dirty-rect blit lists |
| `GliderPRO/Sources/InterfaceInit.c` | 220 | `VariableInit` (all initial global values), menu bar creation, cursors |
| `GliderPRO/Sources/MainWindow.c` | 601 | `theMode` definition, window creation/sizing, splash drawing |
| `GliderPRO/Sources/Environ.c` | 734 | `macEnviron` probe, depth switching, `CheckMemorySize` |
| `GliderPRO/Sources/Menu.c` | 813 | mode transitions (`DoOptionsMenu`), game start, quit |
| `GliderPRO/Sources/Input.c` | 398 | `GetInput`, `GetDemoInput`, `DoPause`, in-game command keys |
| `GliderPRO/Sources/GameOver.c` | 507 | `FlagGameOver`, the two game-over animations (own tick loops) |
| `GliderPRO/Sources/AppleEvents.c` | 208 | required-suite handlers, external quit path |
| `GliderPRO/Sources/DebugUtilities.c` | 355 | debug hooks and whether any of them are compiled in |
| `GliderPRO/Sources/Modes.c` | 640 | (named misleadingly — glider mode transitions, not app modes) |
| `GliderPRO/Sources/StructuresInit2.c` | 476 | `CreateOffscreens`, `CreatePointers`, demo-data allocation |
| `GliderPRO/Sources/Utilities.c` | 788 | `ToolBoxInit`, `RedAlert`, `WaitForInputEvent`, `DelayTicks` |
| `GliderPRO/Sources/SelectHouse.c` | 676 | house list, `demoHouseIndex` discovery |
| `GliderPRO/Sources/Prefs.c` | 281 | preference file I/O and versioning |
| `GliderPRO/Headers/Externs.h` | 393 | key codes, error codes, menu item indices, `prefsInfo` |
| `GliderPRO/Headers/GliderDefines.h` | 625 | every constant, including the mode and pacing constants |
| `GliderPRO/Headers/GliderVars.h` | 59 | the "public" global surface |
| `GliderPRO/Headers/GliderStructs.h` | 347 | all on-disk / in-memory structs |
| `GliderPRO/Headers/GliderProtos.h` | 530 | per-file prototype grouping (used to build the module map) |
| `GliderPRO/Headers/Environ.h` | 37 | `macEnviron` |

Read in part (targeted greps and focused reads for definitions, global
inventory, and call-site confirmation): the remaining 50 `.c` files. The
repository contains exactly **67** `.c` files under `GliderPRO/Sources/`
(46,177 lines total) and **25** `.h` files under `GliderPRO/Headers/`.

Binary evidence parsed with `python3`: the `'demo' 128` resource, `PICT`
dimensions, `MENU 128`–`131`, `WIND 128`–`130`, `vers`, and the whole resource
type inventory, all out of `GliderPRO/Glider PRO.r` (a 15,475,666-byte DeRez
text dump of the shipped application's resource fork). Struct sizes and field
offsets were verified by compiling the real headers' layouts under
`#pragma pack(2)` with 32-bit `long` substituted. See
[Appendix A](#appendix-a-empirical-verification).

---

## 1. Build configuration

Everything in this document is stated for the configuration actually committed
in `GliderPRO/Headers/GliderDefines.h:11-16`:

```c
11  //#define CREATEDEMODATA
12  //#define COMPILEDEMO
13  //#define CAREFULDEBUG
14  #define COMPILENOCP
15  #define COMPILEQT
16  #define BUILD_ARCADE_VERSION		1
```

| Flag | State | Effect when on | Consequence for a port |
|---|---|---|---|
| `CREATEDEMODATA` | **off** (`GliderDefines.h:11`) | `GetInput` records keystrokes into `demoData` via `LogDemoKey`; `CreatePointers` allocates `sizeof(demoType) * 2000` instead of `kDemoLength` (`StructuresInit2.c:280-284`) | Recording path is dead code. The shipped `'demo'` resource was produced by an earlier build with this on. Port only needs playback. |
| `COMPILEDEMO` | **off** (`GliderDefines.h:12`) | Crippled shareware build; `DoNotInDemo()` stubs out menu items (`GliderProtos.h:194-196`) | Ignore entirely. |
| `CAREFULDEBUG` | **off** (`GliderDefines.h:13`) | Would enable the paranoid rect checks in `DebugUtilities.c` | This is the *only* occurrence of the token in the entire tree, and it is commented out. Nothing in `DebugUtilities.c` is reachable in the shipped build. |
| `COMPILENOCP` | **ON** (`GliderDefines.h:14`) | Disables copy protection: `main()` hardcodes `copyGood = true` (`Main.c:308`) instead of calling `ValidInstallation()` | A port has no copy protection. `Validate.c` is dead. |
| `COMPILEQT` | **ON** (`GliderDefines.h:15`) | QuickTime movie playback on the in-room TV object; `EnterMovies()` at startup (`Main.c:330-335`), `MoviesTask()` in the frame loop (`Play.c:463-467`, `Play.c:488-492`) | Port needs *some* video decode path, or must degrade `hasMovie` to false. |
| `BUILD_ARCADE_VERSION` | **1** (`GliderDefines.h:16`) | Kiosk behaviour: arrow keys on the splash screen become menu shortcuts (`Events.c:191-207`); any glider key aborts the demo (`Input.c:192-201`); an extra scoreboard-blanking pass wraps the game-over sequence (`Play.c:512-541`, `Play.c:556-598`) | Decide up front whether the Go port is the arcade build or the retail build; the two differ in *observable input behaviour*, not just cosmetics. |

There is also a `#ifdef powerc` / `TARGET_CARBON` split. `ToolBoxInit`
(`Utilities.c:43-67`) wraps the classic init calls in `#if !TARGET_CARBON`, and
`DrawOnSplash` (`MainWindow.c:82-93`) stamps "PowerPC Native!" on the splash for
PowerPC builds. Neither affects logic.

---

## 2. Toolbox dependency map

Everything the architecture layer touches, and what a Go port must substitute.
This table is the "what will bite you" index; details follow in the body.

| Toolbox facility | Where used (architecture layer) | Go replacement |
|---|---|---|
| `WaitNextEvent` / `GetNextEvent` with a *sleep* argument | `Events.c:499`, `Events.c:504`; `Play.c:393` | A non-blocking poll of the windowing library's event queue plus an explicit sleep/yield. The `sleep = 2` ticks (`Events.c:483`, `Play.c:391`) is a cooperative-multitasking yield, not a timer. |
| `TickCount()` (1/60 s since boot, `UInt32`) | `Events.c:427`, `Events.c:538`; `Render.c:662-665`, `Render.c:690`; `GameOver.c:149`, `GameOver.c:219-220`, `GameOver.c:456`, `GameOver.c:478`; `Utilities.c:446`, `Utilities.c:474` | A monotonic clock scaled to 1/60 s. Do **not** use wall clock. Beware 32-bit wrap (~828 days uptime); the original is vulnerable and a port should use 64-bit. |
| `Delay(ticks, &dummy)` | `Utilities.c:731-736` (`DelayTicks`) | `time.Sleep(ticks * time.Second/60)`. |
| `GetKeys(KeyMap)` + `BitTst` | `Events.c:486-497`; `Input.c:190`, `Input.c:284`; `DoPause` `Input.c:90-116`; `Utilities.c:453-456`, `Utilities.c:490-497` | A polled 128-bit keyboard state bitmap. `BitTst` indexes it by *KeyMap bit offset*, which is **not** the raw Mac virtual key code: within each byte the bit order is reversed, so `offset = (rawKey & 0xF8) \| (7 - (rawKey & 7))`. The original does exactly this conversion in `KeyMapOffsetFromRawKey` (`Utilities.c:504-517`), used by `GetKeyMapFromMessage` (`Utilities.c:522-530`) to turn a `keyDown` event's virtual code into an offset. Keep the *offsets*: that is what is stored in the preferences file and in the glider struct (`gliderType.leftKey` etc. are `long` KeyMap bit offsets, `GliderStructs.h:205-206`), and what every `k…KeyMap` constant in `Externs.h:115-163` holds. (The `k…RawKey` constants at `Externs.h:165-170` are raw virtual codes despite their `// key map offset` comments.) |
| `GWorld` offscreen pixmaps (`NewGWorld`, `GetGWorldPixMap`, `LockPixels`, `SetGWorld`) | `StructuresInit2.c:145-181`; every `*SrcMap` global | 8-bit indexed images in RAM (`image.Paletted`) or GPU textures. |
| `CopyBits(srcBits, dstBits, &srcR, &dstR, srcCopy, maskRgn)` | `Render.c:616-635`, `Render.c:695-736`; `MainWindow.c:130-135`, `MainWindow.c:146-149`; `Play.c:401-403` | A blit with identical rect semantics (src and dst rects may differ in size ⇒ stretch). |
| `CopyMask` | `GameOver.c:173-178` and the object drawers | Alpha/stencil blit from a 1-bit or 8-bit mask. |
| `RgnHandle` (`NewRgn`, `RectRgn`, `UnionRgn`, `DisposeRgn`) | `Render.c:740-771` (`mirrorRgn`); `GameOver.c:42` (`roomRgn`) | A rect list or clip stack. |
| Resource Manager (`GetPicture`, `GetMenu`, `GetCursor`, `GetResource('demo',128)`, `GetIndString`) | `StructuresInit2.c:289`; `InterfaceInit.c:49-68`, `InterfaceInit.c:80-111`; `Utilities.c:135-161`; `Play.c:531`; `GameOver.c:19-22` | Embedded assets (`go:embed`) keyed by the original `(type, id)` pair. Preserve the IDs — they are load-bearing (e.g. background PICT = `2000 + index`, object PICTs are computed). |
| Palette Manager / 8-bit indexed colour | `kPreferredDepth 8` (`Externs.h:15`); `ColorText(..., 5L)` / `(..., 28L)` (`MainWindow.c:77-79`); `ColorRect(&mainWindowRect, 244)` (`GameOver.c:65`); `kRedOrangeColor8 23` (`GliderDefines.h:542`) | The Mac 8-bit system palette, embedded as a 256-entry table. Colour *indices* are hardcoded throughout; you cannot substitute RGB values without a table. |
| `SetDepth` / `HasDepth` / `GetMainDevice` / `GetDeviceList` / `GDevice` | `Environ.c:232-432` | Nothing. A port runs at native depth and treats all of §10 as a no-op (but must keep the derived `thisMac.screen`). |
| `GetGrayRgn()` | `Environ.c:468-469` (fills `thisMac.gray`) | Union of all display work areas; only used to bound window dragging (`Events.c:114-117`). |
| Sound Manager 3 (`SndChannel`) | `InitSound`/`InitMusic` called from `main` (`Main.c:337-338`) | Any mixer. Three channels + one music channel; see the sound doc. |
| Apple Events (`AEInstallEventHandler`, `AEProcessAppleEvent`) | `AppleEvents.c:175-207`, `Events.c:447-454` | OS file-open association and a "quit requested" signal. |
| MultiFinder suspend/resume `osEvt` | `Events.c:390-442`, `Play.c:408-424`, `Utilities.c:461-471` | Window focus-lost / focus-gained. **Must be implemented**: the game's pause-on-background and music-stop behaviour hangs off it, and `switchedOut` gates the frame loop (`Play.c:437-443`). |
| `Gestalt` | `Environ.c:196-226` | Feature detection — replace with compile-time truth. |
| `FindFolder` / `FSSpec` / `PBGetCatInfo` | `Prefs.c:39-69`; `SelectHouse.c:557-646` | OS-appropriate config dir + a directory walk. Note `FSSpec` is 70 bytes and is **embedded in the saved-game format** (`game2Type.house`, `GliderStructs.h:146`). |
| `MaxApplZone`, `MoreMasters`, `FreeMem`, `MaxMem` | `Utilities.c:58-60`; `Environ.c:659`; `Play.c:195` | Nothing. §11 becomes a no-op. |
| `ExitToShell()` | `Utilities.c:160`; `Environ.c:430`, `Environ.c:689` | `os.Exit(1)` after presenting the alert. Note: these paths bypass all cleanup, including preference saving. |
| `#pragma options align=mac68k` | `Externs.h:231-269` (`prefsInfo`) | 2-byte struct alignment for on-disk records. See [Appendix A.2](#a2-struct-sizes-and-offsets). |
| Big-endian on-disk data | house files, saved games, prefs, `'demo'` | `encoding/binary.BigEndian` everywhere. |
| Pascal strings (`Str15`/`Str27`/`Str31`/`Str32`/`Str255` = 16/28/32/33/256 bytes) | prefs, house banner, room names, high scores | length-prefixed byte arrays, fixed size, **not** Go strings. Note `Str32` is 33 bytes, which is why `prefsInfo` has an internal pad byte. |

---

## 3. Startup sequence

`main()` lives at `GliderPRO/Sources/Main.c:284-385`. This is the complete,
in-order boot sequence. Every call is listed; nothing is elided.

| # | Line | Call | What it does / what state it establishes |
|---|---|---|---|
| 1 | `Main.c:291` | `ToolBoxInit()` | `Utilities.c:43-67`. Under `#if !TARGET_CARBON`: `InitGraf(&qd.thePort)`, `InitFonts()`, `FlushEvents(everyEvent, 0)`, `InitWindows()`, `InitMenus()`, `TEInit()`, `InitDialogs(nil)`, `MaxApplZone()`, `MoreMasters()` ×4, `GetDateTime((UInt32*)&qd.randSeed)`. Then unconditionally `InitCursor()` and `switchedOut = false` (`Utilities.c:65-66`). |
| 2 | `Main.c:292` | `CheckOurEnvirons()` | `Environ.c:438-470`. Fills `thisMac` (see §10). Critically sets `thisMac.screen` from the main `GDevice`'s `gdRect`. |
| 3 | `Main.c:293-294` | `if (!thisMac.hasColor) RedAlert(kErrNeedColorQD)` | `kErrNeedColorQD` = 12 (`Externs.h:194`). Unreachable: `hasColor` is hardcoded `true` (`Environ.c:448`). |
| 4 | `Main.c:295-296` | `if (!thisMac.hasSystem7) RedAlert(kErrNeedSystem7)` | `kErrNeedSystem7` = 8 (`Externs.h:190`). Unreachable: hardcoded `true` (`Environ.c:450`). |
| 5 | `Main.c:297-298` | `if (thisMac.numScreens == 0) RedAlert(kErrNeed16Or256Colors)` | `kErrNeed16Or256Colors` = 13 (`Externs.h:195`). |
| 6 | `Main.c:300` | `SetUpAppleEvents()` | `AppleEvents.c:175-207`. Installs the four required-suite handlers. Must happen before the event loop but the ordering vs. the rest is not otherwise constrained. |
| 7 | `Main.c:301` | `LoadCursors()` | `AnimCursor.c`. Loads the animated "spinning beachball"-style cursor set used by `SpinCursor()` during the long init below. |
| 8 | `Main.c:302` | `ReadInPrefs()` | `Main.c:52-201`. Loads `prefsInfo` from disk and scatters it into globals; supplies defaults on failure. See §12. |
| 9 | `Main.c:308` | `copyGood = true` | `COMPILENOCP` branch. The `#else` would call `ValidInstallation(true)` (`Validate.c`). |
| 10 | `Main.c:320` | `HandleDepthSwitching()` | `Environ.c:505-543`. Possibly calls `SetDepth`; always ends with `thisMac.isDepth = WhatsOurDepth()`. |
| 11 | `Main.c:321` | `VariableInit()` | `InterfaceInit.c:118-219`. **The authoritative table of initial global values** — see §4.3. Also computes `houseRect`, `playOriginH/V`, `localRoomsDest[9]`. Must run *after* `HandleDepthSwitching` because it reads `thisMac.isDepth` (`InterfaceInit.c:132-135`) and *after* `CheckOurEnvirons` because it reads `thisMac.screen` (`InterfaceInit.c:124`, `196-204`). |
| 12 | `Main.c:322` | `CheckMemorySize()` | `Environ.c:562-692`. Sums the byte cost of every offscreen buffer and array, compares against `FreeMem()`, and picks exactly one of three outcomes: proceed, drop music only, drop music *and* sounds, or `ExitToShell()`. Must precede all the allocation below. See §11. |
| 13 | `Main.c:323` | `GetExtraCursors()` | `InterfaceInit.c:80-111`. `GetCursor` for `handCursor` (ID 128), `beamCursor` (`iBeamCursor`), `vertCursor` (129), `horiCursor` (130), `diagCursor` (131); each `HLock`ed and dereferenced into a `Cursor` value. |
| 14 | `Main.c:324` | `InitMarquee()` | `Marquee.c`. Builds the 7 marching-ants patterns (`kNumMarqueePats` = 7, `GliderDefines.h:460`). Editor-only, but initialised unconditionally. |
| 15 | `Main.c:325` | `CreatePointers()` | `StructuresInit2.c:187-299`. `NewPtr` for 17 arrays plus `demoData`, and loads `'demo' 128`. See §4.4 and §8.2. Any failure ⇒ `RedAlert(kErrNoMemory)`. |
| 16 | `Main.c:326` | `InitSrcRects()` | `StructuresInit2.c:306-475`. Fills `srcRects[0x90]` — the source rectangle for every object type code. |
| 17 | `Main.c:327` | `CreateOffscreens()` | `StructuresInit2.c:145-181`. Creates `workSrcMap`, `backSrcMap` and every art GWorld. Depends on `houseRect` from step 11. See §7.1. |
| 18 | `Main.c:328` | `OpenMainWindow()` | `MainWindow.c:174-261`. Creates the play/splash window (or the editor window), the fake menu-bar cover window, and computes `splashOriginH/V`. See §9. |
| 19 | `Main.c:330-335` | `if (thisMac.hasQT) { theErr = EnterMovies(); if (theErr != noErr) thisMac.hasQT = false; }` | QuickTime init; failure silently downgrades the capability flag. |
| 20 | `Main.c:337` | `InitSound()` | `Sound.c`. Opens up to 3 `SndChannel`s and preloads the 64 `'snd '` resources unless `dontLoadSounds`. |
| 21 | `Main.c:338` | `InitMusic()` | `Music.c`. Opens the music channel and loads the score unless `dontLoadMusic`. |
| 22 | `Main.c:339` | `BuildHouseList()` | `SelectHouse.c:650-664` → `DoDirSearch`. Enumerates `'gliH'`/`'ozm5'` files, sorts them, and finds `demoHouseIndex` by matching the name `"Demo House"` (`SelectHouse.c:636-644`). |
| 23 | `Main.c:340-341` | `if (OpenHouse()) whoCares = ReadHouse();` | Opens and reads the preferred house (`thisHouseName`, default `"Slumberland"`, `Main.c:127`). |
| 24 | `Main.c:343` | `PlayPrioritySound(kBirdSound, kBirdPriority)` | The startup bird chirp. `kBirdSound` = 10 (`GliderDefines.h:65`), `kBirdPriority` = 804 (`GliderDefines.h:164`). |
| 25 | `Main.c:344` | `DelayTicks(6)` | 6 ticks = 100 ms, purely so the chirp is audible before the menu bar snaps in. |
| 26 | `Main.c:345` | `InitializeMenus()` | `InterfaceInit.c:47-73`. Builds the menu bar. **This is the last init**: `menusUp` goes true here, and `UpdateMenus` is a no-op before it (`Menu.c:245-246`). |
| 27 | `Main.c:345` | `InitCursor()` | Restores the arrow cursor after all the `SpinCursor` churn. |
| 28 | `Main.c:363-364` | `while (!quitting) HandleEvent();` | The main loop. Source comment "this is the main loop" is a trailing comment on `Main.c:363` itself. |

Shutdown, after the loop exits:

| # | Line | Call |
|---|---|---|
| 29 | `Main.c:370` | `KillMusic()` |
| 30 | `Main.c:371` | `KillSound()` |
| 31 | `Main.c:372-379` | `if (houseOpen) { if (!CloseHouse()) { CloseHouseResFork(); fileErr = FSClose(houseRefNum); houseOpen = false; } }` |
| 32 | `Main.c:381` | `WriteOutPrefs()` → `SavePrefs(&thePrefs, kPrefsVersion)` (`Main.c:275`) |
| 33 | `Main.c:382` | `RestoreColorDepth()` (`Environ.c:549-554`) |
| 34 | `Main.c:383` | `FlushEvents(everyEvent, 0)` |

### 3.1 Ordering constraints a port must preserve

1. `CheckOurEnvirons` before `VariableInit` — `thisMac.screen` feeds
   `houseRect`, `shieldRect`, `playOriginH/V` (`InterfaceInit.c:124`,
   `196-204`).
2. `HandleDepthSwitching` before `VariableInit` — `fadeGraysOut` is
   `(thisMac.isDepth == 8)` (`InterfaceInit.c:132-135`).
3. `ReadInPrefs` before `VariableInit` — `willMaxFiles = maxFiles`
   (`InterfaceInit.c:164`), `numNeighbors` clamp (`Main.c:191-192`).
4. `VariableInit` before `CreateOffscreens` — `houseRect` sizes
   `workSrcMap`/`backSrcMap` (`StructuresInit2.c:156-162`).
5. `CreatePointers` before `InitSrcRects` — `srcRects` is the buffer
   `InitSrcRects` writes into (`StructuresInit2.c:271`).
6. `CreateOffscreens` before `OpenMainWindow` — `OpenMainWindow` paints
   `workSrcMap` and loads the splash into it (`MainWindow.c:245-247`).
7. `BuildHouseList` before `OpenHouse` — `OpenHouse` indexes
   `theHousesSpecs[thisHouseIndex]`.
8. `InitializeMenus` last — everything else calls `UpdateMenus`, which
   early-returns while `menusUp` is false.

### 3.2 Failure exits during startup

`RedAlert(short errorNumber)` (`Utilities.c:135-161`) is the terminal error
path: it `GetIndString`s from `STR#` 171 (`rErrMssgID`) — clamping index ≤ 1 to
1 — plus a title from `STR#` 170 (`rErrTitleID`), `ParamText`s them, shows
`ALRT` 170 (`rDeathAlertID`), then `ExitToShell()`. It never returns and never
saves preferences.

| Code | Name | Value | Raised from |
|---|---|---|---|
| 1 | `kErrUnnaccounted` | 1 | `Externs.h:183` (generic) |
| 2 | `kErrNoMemory` | 2 | every `NewPtr` failure in `CreatePointers` (`StructuresInit2.c:194-278`), the `'demo'` buffer and resource load (`StructuresInit2.c:289`, `292`), and `CheckMemorySize`'s `SoundBytesNeeded`/`MusicBytesNeeded` returning ≤ 0 (`Environ.c:577`, `582`) |
| 3 | `kErrDialogDidntLoad` | 3 | dialog helpers |
| 4 | `kErrFailedResourceLoad` | 4 | resource helpers; also every `GetCursor` in `GetExtraCursors` (`InterfaceInit.c:83-84`, `89-90`, `95-96`, `101-102`, `107-108`) and every `GetMenu` in `InitializeMenus` (`InterfaceInit.c:50-51`, `56-57`, `61-62`, `69-70`) |
| 5 | `kErrFailedGraphicLoad` | 5 | `LoadGraphic`/`LoadScaledGraphic` |
| 6 | `kErrFailedOurDirect` | 6 | — |
| 7 | `kErrFailedValidation` | 7 | `Validate.c` (dead under `COMPILENOCP`) |
| 8 | `kErrNeedSystem7` | 8 | `Main.c:296` |
| 9 | `kErrFailedGetDevice` | 9 | `FindOurDevice` (`Utilities.c:167-172`) |
| 10 | `kErrFailedMemoryOperation` | 10 | — |
| 11 | `kErrFailedCatSearch` | 11 | `SelectHouse.c` directory walk |
| 12 | `kErrNeedColorQD` | 12 | `Main.c:294` |
| 13 | `kErrNeed16Or256Colors` | 13 | `Main.c:298` |

A second, softer channel is `YellowAlert(short whichAlert, short identifier)`
(declared `GliderProtos.h:122`, defined in `HouseIO.c`) with 24 codes
`kYellowUnaccounted` = 1 … `kYellowCantOrderLinks` = 24
(`GliderDefines.h:18-41`). These are non-fatal.

A third is `SwitchDepthOrAbort()` (`Environ.c:404-432`), which puts up `ALRT`
130 and, on button 3, calls `ExitToShell()`.

A fourth is `CheckMemorySize`'s out-of-memory branch: `ALRT` 180
(`kSetMemoryAlert`) with the required KB substituted, then `ExitToShell()`
(`Environ.c:684-690`; the `COMPILEDEMO` build uses `ALRT` 181 instead,
`Environ.c:679-682`).

---

## 4. The global mode state machine

### 4.1 `theMode`

There is exactly one application state variable. It is declared
`extern short theMode` at `GliderPRO/Headers/GliderVars.h:56` and **defined** at
`GliderPRO/Sources/MainWindow.c:42`:

```c
42  short			theMode;
```

Note the naming trap: `GliderPRO/Sources/Modes.c` does **not** contain this
state machine. `Modes.c` holds 26 *glider* mode-transition functions
(`StartGliderFadingIn` at `Modes.c:25` through `TagGliderIdle` at `Modes.c:631`)
and one global, `short saidFollow` (`Modes.c:13`). A porter looking for the app
FSM must look in `MainWindow.c`, `Menu.c`, `Play.c` and `Events.c`.

The three legal values are `GliderDefines.h:191-193`:

| Constant | Value | Meaning |
|---|---|---|
| `kSplashMode` | **0** | Idle / attract. Full-screen splash `PICT` 1000 is showing, the menu bar is live, the attract-mode timer runs. |
| `kEditMode` | **1** | House editor. 512×322 document window plus up to four floating palettes; the `House` menu is inserted; the marquee animates on idle. |
| `kPlayMode` | **2** | A game is running. `PlayGame()` (`Play.c:430`) owns the CPU; the outer event loop is not executing. |

`theMode` is initialised to `kSplashMode` in `VariableInit`
(`InterfaceInit.c:154`).

### 4.2 Transitions

Complete list of every write to `theMode` in the tree:

| From | To | Site | Trigger | Side effects at the transition |
|---|---|---|---|---|
| — | `kSplashMode` | `InterfaceInit.c:154` | boot | (initial) |
| `kSplashMode` | `kEditMode` | `Menu.c:406` | Options ▸ Room Editor (Cmd-E) while in splash | `StopTheMusic()` (`Menu.c:407`); `CloseMainWindow(); OpenMainWindow();` (`Menu.c:408-409`); `OpenCloseEditWindows()` (`Menu.c:410`); then `InitCursor(); UpdateMenus(true);` (`Menu.c:412-413`) |
| `kEditMode` | `kSplashMode` | `Menu.c:384` | Options ▸ Room Editor (Cmd-E) while in edit | `SortHouseObjects()` if `fileDirty` (`Menu.c:380-381`); `if (!QuerySaveChanges()) break;` — **the transition is abortable** (`Menu.c:382-383`); `CloseMapWindow(); CloseToolsWindow(); CloseCoordWindow(); CloseLinkWindow();` (`Menu.c:385-388`); `DeselectObject()` (`Menu.c:389`); `StopMarquee()` (`Menu.c:390`); `StartMusic()` if `isPlayMusicIdle` (`Menu.c:391-399`); `CloseMainWindow(); OpenMainWindow();` (`Menu.c:400-401`); `incrementModeTime = TickCount() + kIdleSplashTicks;` (`Menu.c:402`); `InitCursor(); UpdateMenus(true);` (`Menu.c:412-413`) |
| `kSplashMode` | `kPlayMode` | `Play.c:83` | `NewGame(mode)` — from Game ▸ New Game / Two Player Game / Open Saved Game (`Menu.c:308`, `314`, `324`), or from `DoDemoGame()` (`Play.c:295`) | see §6.1 |
| `kPlayMode` | `kSplashMode` | `Play.c:233` | `PlayGame()` returned | see §6.4 |

Observations that matter for a port:

- **There is no direct `kEditMode` ↔ `kPlayMode` transition.** You cannot start
  a game from the editor: the `Game` menu is disabled in edit mode
  (`UpdateMenusEditMode`, called from `Menu.c:256-267`).
- The `kEditMode → kSplashMode` transition is the only one that can be
  *cancelled* by the user (unsaved-changes dialog).
- Every transition destroys and recreates the main window
  (`CloseMainWindow(); OpenMainWindow();`) because the two modes use different
  `WIND` resources and different sizes (§9).
- `UpdateMenus(true)` is called only on the splash↔edit transitions; the `true`
  argument is what inserts/removes the `House` menu (`Menu.c:248-254`).

### 4.3 Initial values of all state globals (`VariableInit`)

`GliderPRO/Sources/InterfaceInit.c:118-219` is the single authoritative list.
Reproduced exhaustively:

| Line | Assignment | Notes |
|---|---|---|
| `122-123` | `shieldPt.h = 0; shieldPt.v = 0;` | cursor-shield origin |
| `124` | `shieldRect = thisMac.screen;` | |
| `126` | `menusUp = false;` | gates `UpdateMenus` |
| `127` | `quitting = false;` | main-loop predicate |
| `128` | `houseOpen = false;` | |
| `129` | `newRoomNow = false;` | editor auto-info trigger |
| `130` | `playing = false;` | inner-loop predicate |
| `131` | `evenFrame = false;` | flame/star alternation |
| `132-135` | `fadeGraysOut = (thisMac.isDepth == 8);` | |
| `136` | `twoPlayerGame = false;` | |
| `137` | `paused = false;` | |
| `138` | `hasMirror = false;` | |
| `139` | `demoGoing = false;` | |
| `141` | `splashDrawn = false;` | gates Game ▸ Load House (`Menu.c:327`) |
| `147` | `theGlider.which = kPlayer1;` | `kPlayer1` = `TRUE` (`GliderDefines.h:556`) |
| `148` | `theGlider2.leftKey = kControlKeyMap;` | 60 (`Externs.h:160`) |
| `149` | `theGlider2.rightKey = kCommandKeyMap;` | 48 (`Externs.h:155`) |
| `150` | `theGlider2.battKey = kOptionKeyMap;` | 61 (`Externs.h:161`) |
| `151` | `theGlider2.bandKey = kShiftKeyMap;` | 63 (`Externs.h:163`) |
| `152` | `theGlider2.which = kPlayer2;` | `kPlayer2` = `FALSE` (`GliderDefines.h:557`) |
| `154` | `theMode = kSplashMode;` | **0** |
| `155` | `thisRoomNumber = 0;` | |
| `156` | `previousRoom = -1;` | |
| `157` | `toolSelected = kSelectTool;` | 0 (`GliderDefines.h:270`) |
| `158` | `houseResFork = -1;` | |
| `159` | `lastBackground = kBaseBackgroundID;` | 2000 (`GliderDefines.h:519`) |
| `160` | `wasFlower = RandomInt(kNumFlowers);` | `kNumFlowers` = 6 (`GliderDefines.h:458`) |
| `161` | `lastHighScore = -1;` | |
| `162` | `idleMode = kIdleSplashMode;` | 0 (`GliderDefines.h:195`) — see §4.5 |
| `163` | `incrementModeTime = TickCount() + kIdleSplashTicks;` | +7200 ticks = +120 s |
| `164` | `willMaxFiles = maxFiles;` | from prefs |
| `165` | `numExtraHouses = 0;` | |
| `167-182` | `fadeInSequence[16] = {4,5,6,7, 5,6,7,8, 6,7,8,9, 7,8,9,10}` | `kLastFadeSequence` = 16 (`GliderDefines.h:564`) |
| `184` | `doubleTime = GetDblTime();` | user's double-click interval, in ticks |
| `186-194` | `mirrorRgn = nil; mainWindow = nil; mapWindow = nil; toolsWindow = nil; linkWindow = nil; coordWindow = nil; toolSrcMap = nil; nailSrcMap = nil; menuWindow = nil;` | |
| `196-201` | `houseRect = thisMac.screen; houseRect.bottom -= kScoreboardTall; if (houseRect.right > kMaxViewWidth) houseRect.right = kMaxViewWidth; if (houseRect.bottom > kMaxViewHeight) houseRect.bottom = kMaxViewHeight;` | `kScoreboardTall` = 20, `kMaxViewWidth` = 1536, `kMaxViewHeight` = 1026 |
| `203` | `playOriginH = (RectWide(&thisMac.screen) - kRoomWide) / 2;` | `kRoomWide` = 512 |
| `204` | `playOriginV = (RectTall(&thisMac.screen) - kTileHigh) / 2;` | `kTileHigh` = 322 |
| `206-210` | `localRoomsDest[0..8]` each = `{0,0,kRoomWide,kTileHigh}` offset by `(playOriginH, playOriginV)` | central room rect |
| `211-218` | per-neighbour displacement | see §9.4 |

`fadeInSequence` deserves note because the values are not monotone: it is a
16-entry table of glider source-rect indices used to cross-fade the glider in.

### 4.4 What `CreatePointers` allocates

`StructuresInit2.c:187-299`. All are `NewPtr`; every failure ⇒
`RedAlert(kErrNoMemory)`.

| Line | Global | Element type | Count | Constant |
|---|---|---|---|---|
| `193` | `thisRoom` | `roomType` | 1 | — (348 bytes) |
| `198` | `hotSpots` | `hotObject` | 56 | `kMaxHotSpots` (`GliderDefines.h:259`) |
| `203` | `sparkles` | `sparkleType` | 3 | `kMaxSparkles` (251) |
| `208` | `flyingPoints` | `flyingPtType` | 3 | `kMaxFlyingPts` (253) |
| `213` | `flames` | `flameType` | 20 | `kMaxCandles` (255) |
| `218` | `tikiFlames` | `flameType` | 8 | `kMaxTikis` (256) |
| `223` | `bbqCoals` | `flameType` | 8 | `kMaxCoals` (257) |
| `228` | `pendulums` | `pendulumType` | 8 | `kMaxPendulums` (258) |
| `233` | `savedMaps` | `savedType` | 24 | `kMaxSavedMaps` (260); `.map = nil` loop at `237-238` |
| `241` | `bands` | `bandType` | 2 | `kMaxRubberBands` (261) |
| `246` | `grease` | `greaseType` | 16 | `kMaxGrease` (262) |
| `251` | `theStars` | `starType` | 4 | `kMaxStars` (263) |
| `256` | `shreds` | `shredType` | 4 | `kMaxShredded` (264) |
| `261` | `dinahs` | `dynaType` | 18 | `kMaxDynamicObs` (265) |
| `266` | `masterObjects` | `objDataType` | 216 | `kMaxMasterObjects` = `kMaxRoomObs * 9` (266) |
| `271` | `srcRects` | `Rect` | 0x90 = 144 | `kNumSrcRects` (`GliderDefines.h:437`) |
| `276` | `theHousesSpecs` | `FSSpec` | `maxFiles` | prefs, clamped to [12, 500] (`Main.c:88-89`) |
| `287` | `demoData` | raw bytes | `kDemoLength` = 6702 | `GliderDefines.h:625` |

The demo load, verbatim (`StructuresInit2.c:280-298`). Note the whole thing is
`#ifdef CREATEDEMODATA`-switched; the ship build takes the `#else` arm:

```c
280  #ifdef CREATEDEMODATA
281      demoData = nil;
282      demoData = (demoPtr)NewPtr(sizeof(demoType) * 2000);
283      if (demoData == nil)
284          RedAlert(kErrNoMemory);
285  #else
286      demoData = nil;
287      demoData = (demoPtr)NewPtr(kDemoLength);
288      if (demoData == nil)
289          RedAlert(kErrNoMemory);
290      tempHandle = GetResource('demo', 128);
291      if (tempHandle == nil)
292          RedAlert(kErrNoMemory);
293      else
294      {
295          BlockMove(*tempHandle, demoData, kDemoLength);
296          ReleaseResource(tempHandle);
297      }
298  #endif
```

Note the size is a *byte count*, not an element count: `kDemoLength` = 6702 =
1117 × `sizeof(demoType)` (6). See §8.2.

### 4.5 The vestigial idle sub-state machine

`GliderDefines.h:195-198` defines what looks like a second FSM:

```c
195  #define kIdleSplashMode				0
196  #define kIdleDemoMode				1
197  #define kIdleSplashTicks			7200L		// 2 minutes
198  #define kIdleLastMode				1
```

Reality:

- `short idleMode` is defined at `Events.c:30`.
- It is **written exactly once**, to `kIdleSplashMode`, at
  `InterfaceInit.c:162`.
- `kIdleDemoMode` and `kIdleLastMode` are **never referenced** anywhere in the
  67 `.c` files.
- `void IncrementMode (void);` is forward-declared at `Events.c:24` but is
  **never defined and never called**.

Only `kIdleSplashTicks` survives, as the attract-mode countdown length. A port
should implement one boolean-free countdown and ignore `idleMode` entirely; but
it should *know* these constants exist so it does not go hunting for a missing
state machine.

### 4.6 How the game distinguishes idle / demo / play / edit

There is no single enum covering all four. The distinction is a *pair*:

| Situation | `theMode` | `playing` | `demoGoing` | `twoPlayerGame` | Who owns the CPU |
|---|---|---|---|---|---|
| Idle / attract-waiting | `kSplashMode` (0) | false | false | false | `HandleEvent` (`Events.c:479`) |
| Attract demo running | `kPlayMode` (2) | true | **true** | false | `PlayGame` (`Play.c:430`) |
| One-player game | `kPlayMode` (2) | true | false | false | `PlayGame` |
| Two-player game | `kPlayMode` (2) | true | false | **true** | `PlayGame` |
| Game-over animation | `kPlayMode` (2) | true→false | either | either | `DoGameOver` / `DoDiedGameOver` (own loops) |
| Paused | `kPlayMode` (2) | true | either | either | `DoPause` (`Input.c:77`) |
| Editing | `kEditMode` (1) | false | false | false | `HandleEvent` |

The four gating booleans:

| Global | Type | Defined | Meaning |
|---|---|---|---|
| `playing` | `Boolean` | `Play.c:53` | the `PlayGame` while-loop predicate; cleared by quit-game, glider death handling, or game over |
| `demoGoing` | `Boolean` | `Play.c:53` | input comes from `demoData` instead of the keyboard; also shortens the post-game wait (`GameOver.c:494-497`) and suppresses high-score entry |
| `twoPlayerGame` | `Boolean` | `Play.c:53` | second glider active; disables saving and pausing-to-save |
| `paused` | `Boolean` | `Input.c:34` | `DoPause`'s inner while-loop predicate |

`demoGoing` is the flag a port most easily gets wrong, because it changes
*three* things: the input source (`Play.c:478-481`), the post-death wait
duration — `WaitForInputEvent`'s argument is in **seconds**, so 1 s vs. 10 s
(`GameOver.c:494-502`) — and whether a high score can be
recorded at all (`TestHighScore()` is only reached on the non-demo arm,
`GameOver.c:503`).

---

## 5. The outer event loop

### 5.1 `main()`'s loop

```c
363  while (!quitting)		// this is the main loop
364      HandleEvent();
```
(`Main.c:363-364`; the `// this is the main loop` comment trails the `while` on
`Main.c:363`.)

`Boolean quitting` is defined at `Main.c:26` and initialised false at
`InterfaceInit.c:127`. It is set true from exactly two places:

1. `Menu.c:348-356` — Game ▸ Quit: `quitting = true;` then
   `if (!QuerySaveChanges()) quitting = false;` (the save dialog can veto).
2. `AppleEvents.c:147-157` — `DoQuitAE`, the `kAEQuitApplication` handler.
   **This one does not ask about unsaved changes.**

`quitting` is also read inside the inner game loop
(`while ((playing) && (!quitting))`, `Play.c:432`), so a quit Apple Event
arriving during a game breaks out of the game *and* the main loop. But note the
inner loop only pumps events at all when `doBackground` is true (§6.3), so in
the normal configuration a quit AE cannot be seen mid-game.

### 5.2 `HandleEvent()` — full control flow

`GliderPRO/Sources/Events.c:479-541`. The comment at `Events.c:477` reads
"Not called during and actual game" [sic] — this is the *outer* loop only.

```
 1. (Events.c:481-483) locals: KeyMap eventKeys; EventRecord theEvent;
    long sleep = 2; Boolean itHappened;
 2. (486) GetKeys(eventKeys)                       // poll modifier state BEFORE
                                                   // dequeuing any event
 3. (487-488) if BitTst(&eventKeys, kCommandKeyMap)  // 48
          and BitTst(&eventKeys, kOptionKeyMap):    // 61
 4.   (490) HiliteAllObjects()                      // editor: flash every object
 5. (492-493) else if BitTst(&eventKeys, kOptionKeyMap)
          and theMode == kEditMode
          and houseUnlocked:
 6.   (495) EraseSelectedTool()
 7.   (496) SelectTool(kSelectTool)                 // Option = temporary arrow
 8. (499) if thisMac.hasWNE:
 9.   (500) itHappened = WaitNextEvent(everyEvent, &theEvent, sleep, nil)
10. (501-502) else:
11.   (504) itHappened = GetNextEvent(everyEvent, &theEvent)
              // the commented-out SystemTask() is at 503
12. (507) if itHappened:
13.   (509) switch theEvent.what:
14.     (511) case mouseDown:      HandleMouseEvent(&theEvent)      // Events.c:512
15.     (515) case keyDown:
16.     (516) case autoKey:        HandleKeyEvent(&theEvent)        // Events.c:517
17.     (520) case updateEvt:      HandleUpdateEvent(&theEvent)     // Events.c:521
18.     (524) case osEvt:          HandleOSEvent(&theEvent)         // Events.c:525
19.     (528) case kHighLevelEvent:HandleHighLevelEvent(&theEvent)  // Events.c:529
              // no default: any other event type is silently dropped
20. (533) else:
21.   (534) HandleIdleTask()
22. (536) if theMode == kSplashMode and doAutoDemo and not switchedOut:
23.   (538) if TickCount() >= incrementModeTime:
24.     (539) DoDemoGame()
```

Structural facts a port must not lose:

- **Modifier polling happens every iteration, before the event dequeue**
  (step 2). The Option/Command "spring-loaded tool" behaviour is level-triggered
  off `GetKeys`, not edge-triggered off key events. If you implement it from
  key-down/key-up events you will get different behaviour when the app regains
  focus with a modifier already held.
- `HandleIdleTask` runs **only when no event was dequeued** (step 20). With
  `sleep = 2`, a busy event stream can starve the marquee animation.
- The attract check (step 22) runs unconditionally at the bottom, whether or not
  an event was handled.
- `sleep = 2` ticks ≈ 33 ms is the maximum block; so the outer loop spins at
  ≥ 30 Hz even when idle. This is the *only* pacing in the outer loop; there is
  no frame counter and no `nextFrame` outside `PlayGame`.

### 5.3 `HandleMouseEvent`

`Events.c:60-157`. `FindWindow(theEvent->where, &whichWindow)` result switch:

| `part` | Lines | Behaviour |
|---|---|---|
| `inSysWindow` | `70-73` | body is `SystemClick(theEvent, whichWindow);` **commented out** — desk accessories are dead |
| `inMenuBar` | `75-78` | `DoMenuChoice(MenuSelect(theEvent->where))` |
| `inDrag` | `80-96` | `DragWindow(whichWindow, theEvent->where, &thisMac.screen)`; then an if/else-if chain that remembers the new top-left of whichever window moved: `mainWindow` → `SendBehind(mainWindow, nil)` + `GetWindowLeftTop(→ isEditH, isEditV)` (`82-86`), `mapWindow` → `isMapH/isMapV` (`87-88`), `toolsWindow` → `isToolsH/isToolsV` (`89-90`), `linkWindow` → `isLinkH/isLinkV` (`91-92`), `coordWindow` → `isCoordH/isCoordV` (`93-94`); then `HiliteAllWindows()` (`95`). Those five position pairs are what get written to the prefs file (§12). |
| `inGoAway` | `98-110` | `TrackGoAway`, then close the matching floating palette |
| `inGrow` | `112-118` | map window only; `GrowWindow(whichWindow, theEvent->where, &thisMac.gray)` then `ResizeMapWindow(LoWord(growArea), HiWord(growArea))` |
| `inZoomIn` / `inZoomOut` | `120-124` | handled together |
| `inContent` | `126-152` | dispatch to `HandleMainClick` / `HandleMapClick` / `HandleToolsClick` / `HandleLinkClick` depending on window |

Double-click detection (`Events.c:135-143`) — note it is an `if`/`else`, and the
`lastUp`/`lastWhere` anchor is updated **only on the non-double-click branch**:

```c
135  if (((theEvent->when - lastUp) < doubleTime) && (hDelta < 5) &&
136          (vDelta < 5))
137      isDoubleClick = true;
138  else
139  {
140      isDoubleClick = false;
141      lastUp = theEvent->when;
142      lastWhere = theEvent->where;
143  }
```

That asymmetry is load-bearing: because a click that *is* recognised as a double
does not re-anchor `lastUp`, a rapid burst of clicks all measure their interval
from the *first* click of the burst. Three clicks 10 ticks apart therefore yield
double-click, double-click (if `doubleTime` ≥ 20), not double-click, single. A
port that unconditionally stores the timestamp will behave differently.

`hDelta`/`vDelta` are `|where - lastWhere|` (`Events.c:129-134`), `long lastUp` and
`Point lastWhere` are at `Events.c:27` and `Events.c:29`, and
`UInt32 doubleTime` (`Events.c:28`) comes from `GetDblTime()`
(`InterfaceInit.c:184`). `IgnoreThisClick()` (`Events.c:567-572`) poisons the
state so the next click cannot be a double:

```c
569  lastUp -= doubleTime;
570  lastWhere.h = -100;
571  lastWhere.v = -100;
```

### 5.4 `HandleKeyEvent`

`Events.c:162-336`. Decode preamble (`Events.c:167-170`):

```c
167  theChar     = theEvent->message & charCodeMask;
168  shiftDown   = ((theEvent->modifiers & shiftKey)   != 0);
169  commandDown = ((theEvent->modifiers & cmdKey)     != 0);
170  optionDown  = ((theEvent->modifiers & optionKey)  != 0);
```

Then, in order:

| Lines | Key | Behaviour | Gate |
|---|---|---|---|
| `172-173` | any with Command and **not** Option | `DoMenuChoice(MenuKey(theChar))` | — |
| `178` | `kHelpKeyASCII` (0x05) | no-op (empty case) | — |
| `181-184` | `kPageUpKeyASCII` (0x0B) | `PrevToolMode()` | `houseUnlocked` only (**no** `theMode` test — `Events.c:182`) |
| `186-189` | `kPageDownKeyASCII` (0x0C) | `NextToolMode()` | `houseUnlocked` only (`Events.c:187`) |
| `191-207` | arrows, **arcade build** | Left ⇒ `DoOptionsMenu(iHighScores)`; Right ⇒ `DoOptionsMenu(iHelp)`; Up ⇒ `DoGameMenu(iNewGame)`; Down ⇒ `DoGameMenu(iNewGame)` | `#if BUILD_ARCADE_VERSION`, otherwise ungated |
| `209-251` | arrows, retail build | move the selected object, or step to the neighbouring room | `houseUnlocked` only (`Events.c:212`, `222`, `232`, `242`) — **not** `theMode == kEditMode` |
| `253-261` | `kDeleteKeyASCII` (0x08) | `DeleteRoom(true)` if nothing selected, else `DeleteObject()` | `houseUnlocked` only (`Events.c:254`) |
| `263-271` | `kTabKeyASCII` (0x09) | `SelectPrevObject()` if shift, else `SelectNextObject()` | edit + unlocked |
| `273-276` | `kEscapeKeyASCII` (0x1B) | `DeselectObject()` | edit + unlocked |
| `278-282` | `a`/`A` | `SetSpecificToolMode(kApplianceMode)` (7) | edit + unlocked |
| `284-288` | `b`/`B` | `kBlowerMode` (1) | edit + unlocked |
| `290-294` | `c`/`C` | `kClutterMode` (9) | edit + unlocked |
| `296-300` | `e`/`E` | `kEnemyMode` (8) | edit + unlocked |
| `302-306` | `f`/`F` | `kFurnitureMode` (2) | edit + unlocked |
| `308-312` | `l`/`L` | `kLightMode` (6) | edit + unlocked |
| `314-318` | `p`/`P` | `kBonusMode` (3) | edit + unlocked |
| `320-324` | `s`/`S` | `kSwitchMode` (5) | edit + unlocked |
| `326-330` | `t`/`T` | `kTransportMode` (4) | edit + unlocked |

Note the gate discipline is inconsistent in the original: Page Up/Down, the
retail arrows and Delete test `houseUnlocked` *alone*, while Tab, Escape and the
nine tool letters test `(theMode == kEditMode) && houseUnlocked`. So in the
shipped retail build an arrow key or Delete pressed on the splash screen will
still move/delete editor objects if a house happens to be unlocked. Reproduce the
gates exactly rather than normalising them.

The arcade arrow remap is the single most porting-visible consequence of
`BUILD_ARCADE_VERSION`: on a kiosk, the cabinet's joystick starts a game and
browses high scores from the attract screen. Note both Up **and** Down map to
`iNewGame` — there is no two-player shortcut on the joystick.

`iHelp` = 5 (`Externs.h:206`) but the Options menu's fifth item is literally
**"Demo…"** with Cmd-D (verified from `MENU` 130, [Appendix A.4](#a4-menu-resources)),
and `DoOptionsMenu`'s `case iHelp:` calls `DoDemoGame()` (`Menu.c:427-429`).
The identifier name is a leftover.

### 5.5 `HandleUpdateEvent`

`Events.c:341-385`. A flat `if`-chain on which window needs redrawing; each arm
is `SetPort((GrafPtr)w); BeginUpdate(w); Update…(); EndUpdate(w);` — the
`SetPort` is part of every arm and must not be dropped, since the `Update…`
routines draw into the current port.

| Lines | Window | Updater |
|---|---|---|
| `343-349` | `mainWindow` | `UpdateMainWindow()` (`MainWindow.c:120`) |
| `350-356` | `mapWindow` | `UpdateMapWindow()` |
| `357-363` | `toolsWindow` | `UpdateToolsWindow()` |
| `364-370` | `linkWindow` | `UpdateLinkWindow()` |
| `371-377` | `coordWindow` | `UpdateCoordWindow()` |
| `378-384` | `menuWindow` | `UpdateMenuBarWindow()` (`MainWindow.c:160`) |

### 5.6 `HandleOSEvent` — suspend and resume

`Events.c:390-442`. This is the load-bearing focus handler.

```
 1. (395) if (theEvent->message & 0x01000000)          // it is a suspend/resume
 2.   (397) if (theEvent->message & 0x00000001)        // RESUME
 3.     (399) if WhatsOurDepth() != thisMac.isDepth:
 4.       (401) buttonHit = BitchAboutColorDepth()     // ALRT 1042
 5.       (402) if buttonHit == 1:                      // "Quit"
 6.         (405) if QuerySaveChanges(): quitting = true   // #ifndef COMPILEDEMO;
                                                          // the demo build just
                                                          // sets quitting (408)
 7.       (411) else:                                  // "Switch back"
 8.         (413) SwitchToDepth(thisMac.isDepth, thisMac.wasColorOrGray)
 9.     (416) switchedOut = false
10.     (417) InitCursor()
11.     (418) if isPlayMusicIdle and theMode != kEditMode:
12.       (420) theErr = StartMusic()
13.       (421) if theErr != noErr:
14.         (423)   YellowAlert(kYellowNoMusic, theErr)
15.         (424)   failedMusic = true                 // latches: music is never retried
16.     (427) incrementModeTime = TickCount() + kIdleSplashTicks
17.   (434) else:                                       // SUSPEND
18.     (436) switchedOut = true
19.     (437) InitCursor()
20.     (438) if isMusicOn and theMode != kEditMode:
21.       (439) StopTheMusic()
```

The two message bits are the classic MultiFinder encoding: bit 24
(`0x01000000`) marks the event as suspend/resume, bit 0 (`0x00000001`)
distinguishes resume (1) from suspend (0).

`switchedOut` is defined at `Events.c:31`, initialised false in `ToolBoxInit`
(`Utilities.c:66`), and has three consumers:

1. `Events.c:536` — suppresses the attract-mode trigger while backgrounded.
2. `Play.c:437-443` — the in-game background-pumping loop:
   `if (doBackground) { do { HandlePlayEvent(); } while (switchedOut); }`.
   That inner `do/while` is what actually **freezes the game** when the app is
   sent to the back: the frame loop blocks inside `HandlePlayEvent`'s
   `WaitNextEvent` until a resume event flips the flag.
3. `Utilities.c:461-471` — `WaitForInputEvent` treats a resume as input.

Note asymmetry: resume checks `isPlayMusicIdle`; suspend checks `isMusicOn`.

Also note that `incrementModeTime` is pushed forward on resume (step 14) but
**not** on suspend, so a long background period does not immediately trigger the
attract mode when you come back.

### 5.7 `HandleHighLevelEvent`

`Events.c:447-454`:

```c
451  theErr = AEProcessAppleEvent(theEvent);
452  if ((theErr != noErr) && (theErr != errAEEventNotHandled))
453      YellowAlert(kYellowAppleEventErr, theErr);
```
`kYellowAppleEventErr` = 15 (`GliderDefines.h:32`).

### 5.8 `HandleIdleTask`

`Events.c:459-473`. Runs only when the event queue was empty.

```
1. (461) if theMode == kEditMode:
2.   (463) SetPort((GrafPtr)mainWindow)
3.   (464) DoMarquee()                       // Marquee.c — advances marching ants
4.   (466) if autoRoomEdit and newRoomNow:
5.     (468) if theMode == kEditMode:        // redundant: already inside the (461) test
6.       (469) DoRoomInfo()                  // pops the room-info dialog
7.     (470) newRoomNow = false              // cleared AFTER the dialog returns
```

The order is the reverse of what you would expect: `DoRoomInfo()` is a modal
dialog and it runs *before* `newRoomNow` is cleared (`Events.c:469`, then
`:470`), so anything that re-enters the idle path from inside the dialog still
sees `newRoomNow == true`. Note also the redundant inner `theMode == kEditMode`
test at `Events.c:468`.

In splash and play mode `HandleIdleTask` does nothing at all. There is **no**
idle animation on the splash screen.

### 5.9 `HiliteAllWindows`

`Events.c:548-560`. Calls `HiliteWindow(w, true)` for `mainWindow`,
`mapWindow`, `toolsWindow`, `coordWindow`, `linkWindow` when each is non-nil.
Used after a drag so the floating palettes do not look deactivated.

### 5.10 Menus

Menu IDs (`GliderDefines.h:186-189`): `kAppleMenuID` 128, `kGameMenuID` 129,
`kOptionsMenuID` 130, `kHouseMenuID` 131.

`InitializeMenus` (`InterfaceInit.c:47-73`):

```
1. (49) appleMenu = GetMenu(kAppleMenuID)
2. (52) AppendResMenu(appleMenu, 'DRVR')     // desk accessories
3. (53) InsertMenu(appleMenu, 0)
4. (55) gameMenu = GetMenu(kGameMenuID)
5. (58) InsertMenu(gameMenu, 0)
6. (60) optionsMenu = GetMenu(kOptionsMenuID)
7. (63) InsertMenu(optionsMenu, 0)
8. (65) menusUp = true
9. (66) DrawMenuBar()
10.(68) houseMenu = GetMenu(kHouseMenuID)    // loaded but NOT inserted
11.(72) UpdateMenus(false)
```

Each of the four `GetMenu` calls is followed by a nil check that calls
`RedAlert(kErrFailedResourceLoad)` (`InterfaceInit.c:50-51, 56-57, 61-62, 69-70`),
so a missing `MENU` resource is a hard startup failure.

`UpdateMenus(Boolean newMode)` (`Menu.c:243-272`):

```
1. (245-246) if not menusUp: return
2. (248) if newMode:
3.   (250-251) if theMode == kEditMode: InsertMenu(houseMenu, 0)
4.   (252-253) else: DeleteMenu(kHouseMenuID)
5. (256) if theMode == kEditMode:
6.   (258) UpdateMenusEditMode()
7.   (259) if houseOpen: UpdateMenusHouseOpen() (261);
                         UpdateClipboardMenus() (262)
8.   (264) else:         UpdateMenusHouseClosed() (265)
9.   (266) UpdateLinkControl()
10.(268-269) else: UpdateMenusNonEditMode()
11.(271) DrawMenuBar()
```

`DoMenuChoice(long menuChoice)` (`Menu.c:591-621`):

```
1. (595-596) if menuChoice == 0: return
2. (598) theMenu = HiWord(menuChoice)
3. (599) theItem = LoWord(menuChoice)
4. (601) switch theMenu:
5.   (603) kAppleMenuID   -> DoAppleMenu(theItem)
6.   (607) kGameMenuID    -> DoGameMenu(theItem)
7.   (611) kOptionsMenuID -> DoOptionsMenu(theItem)
8.   (615) kHouseMenuID   -> DoHouseMenu(theItem)
9. (620) HiliteMenu(0)
```

Because `DoGameMenu(iNewGame)` calls `NewGame()` which calls `PlayGame()`, the
entire game runs *inside* this call, three frames deep on the C stack:
`HandleEvent → HandleKeyEvent/HandleMouseEvent → DoMenuChoice → DoGameMenu →
NewGame → PlayGame`. See §14.1 for why this matters for a Go port.

Menu item constants (`Externs.h:197-222`), grouped by menu:

| Menu | Item | Constant | Value |
|---|---|---|---|
| Apple (128) | About Glider PRO… | `iAbout` | 1 |
| Game (129) | New Game | `iNewGame` | 1 |
| Game (129) | Two Player Game | `iTwoPlayer` | 2 |
| Game (129) | Open Saved Game… | `iOpenSavedGame` | 3 |
| Game (129) | *(separator)* | — | 4 |
| Game (129) | Load House… | `iLoadHouse` | 5 |
| Game (129) | *(separator)* | — | 6 |
| Game (129) | Quit | `iQuit` | 7 |
| Options (130) | Room Editor | `iEditor` | 1 |
| Options (130) | *(separator)* | — | 2 |
| Options (130) | High Scores… | `iHighScores` | 3 |
| Options (130) | Preferences… | `iPrefs` | 4 |
| Options (130) | Demo… | `iHelp` | 5 |
| House (131) | New House… | `iNewHouse` | 1 |
| House (131) | Save | `iSave` | 2 |
| House (131) | House… | `iHouse` | 4 |
| House (131) | Room… | `iRoom` | 5 |
| House (131) | Object… | `iObject` | 6 |
| House (131) | Cut | `iCut` | 8 |
| House (131) | Copy | `iCopy` | 9 |
| House (131) | Paste | `iPaste` | 10 |
| House (131) | Clear | `iClear` | 11 |
| House (131) | Duplicate | `iDuplicate` | 12 |
| House (131) | Bring Forward | `iBringForward` | 14 |
| House (131) | Send Back | `iSendBack` | 15 |
| House (131) | Go To Room… | `iGoToRoom` | 17 |
| House (131) | Map Window | `iMapWindow` | 19 |
| House (131) | Object Window | `iObjectWindow` | 20 |
| House (131) | Coordinate Window | `iCoordinateWindow` | 21 |

The item *text* and Cmd-key equivalents were read out of the resource fork; see
[Appendix A.4](#a4-menu-resources).

### 5.11 Game menu actions

`DoGameMenu(short theItem)` (`Menu.c:301-361`):

| Item | Lines | Behaviour |
|---|---|---|
| `iNewGame` | `305-309` | `twoPlayerGame = false; resumedSavedGame = false; NewGame(kNewGameMode);` |
| `iTwoPlayer` | `311-315` | `twoPlayerGame = true; resumedSavedGame = false; NewGame(kNewGameMode);` |
| `iOpenSavedGame` | `317-325` | `resumedSavedGame = true; HeyYourPissingAHighScore(); if (OpenSavedGame()) { twoPlayerGame = false; NewGame(kResumeGameMode); }` |
| `iLoadHouse` | `327-346` | gated on `splashDrawn`; `DoLoadHouse(); OpenCloseEditWindows(); UpdateMenus(false); incrementModeTime = TickCount() + kIdleSplashTicks;` then `InvalWindowRect` of the 166×12 house-name strip on the splash |
| `iQuit` | `348-356` | `quitting = true; if (!QuerySaveChanges()) quitting = false;` |

`kNewGameMode` = 1, `kResumeGameMode` = 0 (`GliderDefines.h:616-617`).

`HeyYourPissingAHighScore()` (`Menu.c:779-786`) shows `ALRT` 1046
(`kNoHighScoreAlert`) warning that resuming a saved game forfeits high-score
eligibility.

### 5.12 Options menu actions

`DoOptionsMenu(short theItem)` (`Menu.c:366-431`). `case iEditor:` is the mode
toggle documented in §4.2. The rest:

| Item | Lines | Behaviour |
|---|---|---|
| `iHighScores` | `417-420` | `DoHighScores(); incrementModeTime = TickCount() + kIdleSplashTicks;` |
| `iPrefs` | `422-425` | `DoSettingsMain(); incrementModeTime = TickCount() + kIdleSplashTicks;` |
| `iHelp` | `427-429` | `DoDemoGame();` — i.e. the "Demo…" item manually launches the attract demo |

---

## 6. The inner game loop

### 6.1 `NewGame(short mode)` — game setup

`GliderPRO/Sources/Play.c:74-278`. `mode` is `kNewGameMode` (1) or
`kResumeGameMode` (0). The author's own trailing comments on `Play.c:213-214`
mark the split: `playing = true;  // everything before this line is game set-up`
and `PlayGame();  // everything following is after a game has ended`.

Setup, in order:

```
 1. (81)      AdjustScoreboardHeight()
 2. (82)      gameOver = false
 3. (83)      theMode = kPlayMode                       // <-- the state change
 4. (84-101)  if isPlayMusicGame: StartMusic(); SetMusicalMode(kPlayGameScoreMode)
              else if isMusicOn:  StopTheMusic()
 5. (102-103) if mode != kResumeGameMode: SetObjectsToDefaults()
 6. (104)     HideCursor()
 7. (106-108) if mode == kResumeGameMode: SetHouseToSavedRoom()
              else:                       SetHouseToFirstRoom()
 8. (109)     DetermineRoomOpenings()
 9. (110)     NilSavedMaps()
10. (112)     gameFrame  = 0L
11. (113)     numBands   = 0
12. (114)     demoIndex  = 0
13. (115)     saidFollow = 0
14. (116)     otherPlayerEscaped = kNoOneEscaped         // -1
15. (117)     onePlayerLeft  = false
16. (118)     playerSuicide  = false
17. (120-136) glider init + art load. `InitGlider`'s second parameter is a
              `short mode` (`Play.c:307`), not a Boolean:
                two-player -> InitGlider(&theGlider,  kNewGameMode)   // 1, hard-coded
                              InitGlider(&theGlider2, kNewGameMode)   // 1, hard-coded
                              LoadGraphic(kGliderPictID 3999) into glidSrcMap
                              LoadGraphic(kGlider2PictID 3974) into glid2SrcMap
                one-player -> InitGlider(&theGlider, mode)            // forwards NewGame's mode
                              LoadGraphic(kGliderPictID 3999) into glidSrcMap
                              LoadGraphic(kGliderFoilPictID 3976) into glid2SrcMap
              Note the two-player path *ignores* `mode`, so resuming a saved
              two-player game still starts both gliders at the first-room
              position.
18. (142-145) PaintRect a 20-pixel-tall black strip across the bottom of the screen
19. (147-152) SetMovieGWorld(theMovie, (CGrafPtr)mainWindow, nil)   // QuickTime;
              the movie draws straight to the *window*, not to `workSrcMap`
20. (154-155) SetPort(workSrcMap); PaintRect(&workSrcRect)  // clear the work buffer
21. (163)     DrawLocale()                                // draw the 3x3 neighbourhood
22. (164)     RefreshScoreboard(kNormalTitleMode)          // 0
23. (169-182) if mode == kNewGameMode:
                BringUpBanner(); DumpScreenOn(&justRoomsRect)
              else if mode == kResumeGameMode:
                DisplayStarsRemaining(); DumpScreenOn(&justRoomsRect)
              else:                                       // unreachable third arm
                DumpScreenOn(&justRoomsRect)
24. (184)     InitGarbageRects()                          // also seeds nextFrame!
25. (185)     StartGliderFadingIn(&theGlider)
26. (186-191) if twoPlayerGame: StartGliderFadingIn(&theGlider2) (188);
                                TagGliderIdle(&theGlider2) (189);
                                theGlider2.dontDraw = true (190)
27. (192)     InitTelephone()
28. (195)     MaxMem(&whoCares)                            // compact the heap
29. (201-211) if hasQT and hasMovie and tvInRoom: SetMovieActive(); StartMovie()
30. (213)     playing = true
31. (214)     PlayGame()                                   // <=== blocks here
```

Step 24 is easy to miss and essential: `InitGarbageRects` is what sets
`nextFrame = TickCount() + kTicksPerFrame` (`Render.c:690`) before the first
frame. Without it the first `RenderFrame` would busy-wait against a stale
`nextFrame`.

### 6.2 `PlayGame()` — the frame loop

`GliderPRO/Sources/Play.c:430-599`. This is the heart of the program.

```
 1. (432) while (playing && !quitting):
 2.   (434)   gameFrame++
 3.   (435)   evenFrame = !evenFrame
 4.   (437)   if doBackground:
 5.     (439)   do { HandlePlayEvent(); } while (switchedOut)
 6.   (445)   HandleTelephone()
 7.   (447)   if twoPlayerGame:                              // ---- two-player
 8.     (449)   HandleDynamics()
 9.     (450)   if !gameOver:
10.       (452)     GetInput(&theGlider)
11.       (453)     GetInput(&theGlider2)
12.       (454)     HandleInteraction()
13.     (456)   HandleTriggers()
14.     (457)   HandleBands()
15.     (458)   if !gameOver:
16.       (460)     HandleGlider(&theGlider)
17.       (461)     HandleGlider(&theGlider2)
18.     (463)   if playing:
19.       (466-467)  if hasQT and hasMovie and tvInRoom and tvOn: MoviesTask(theMovie, 0)
                     // inside #ifdef COMPILEQT (465-468)
20.       (469)     RenderFrame()                             // <-- pacing lives here
21.       (470)     HandleDynamicScoreboard()
22.   (473) else:                                             // ---- one-player
23.     (475)   HandleDynamics()
24.     (476)   if !gameOver:
25.       (478-479) if demoGoing: GetDemoInput(&theGlider)
26.       (480-481) else:         GetInput(&theGlider)
27.       (482)     HandleInteraction()
28.     (484)   HandleTriggers()
29.     (485)   HandleBands()
30.     (486-487) if !gameOver: HandleGlider(&theGlider)
31.     (488)   if playing:
32.       (491-492)  if hasQT and hasMovie and tvInRoom and tvOn: MoviesTask(theMovie, 0)
                     // inside #ifdef COMPILEQT (490-493)
33.       (494)     RenderFrame()
34.       (495)     HandleDynamicScoreboard()
35.   (499)   if gameOver:
36.     (501)     countDown--
37.     (502)     if countDown <= 0:
38.       (504-505)   CGrafPtr wasCPort; GDHandle wasWorld;   // block-local decls
39.       (507)       GetGWorld(&wasCPort, &wasWorld)
40.       (509)       HideGlider(&theGlider)
41.       (510)       RefreshScoreboard(kNormalTitleMode)
42.       (512-544)   [arcade only] blank the scoreboard and redraw PICT 1997
43.       (546-547)   if mortals < 0: DoDiedGameOver()
44.       (548-549)   else:           DoGameOver()
45.       (551)       SetGWorld(wasCPort, wasWorld)
46. (554) end while
47. (556-598) [arcade only] blank the scoreboard and redraw PICT 1997 again
```

Critical ordering properties:

1. **`gameFrame` increments exactly once per iteration**, at the very top
   (`Play.c:434`). Everything time-dependent — demo playback, physics counters,
   `batteryFrame` sound throttling — is a function of `gameFrame`, so the
   simulation is *frame*-locked, and the frame rate is what makes it
   *tick*-locked (§7.2).
2. **`evenFrame` toggles every frame** (`Play.c:435`) and selects flames vs.
   stars in `RenderFrame` (`Render.c:649-652`). Its initial value is `false`
   (`InterfaceInit.c:131`), and it is **not** reset by `NewGame`, so the
   flame/star phase carries over between games. A faithful port should carry it
   over too (or accept a one-frame cosmetic difference).
3. **Dynamics run before input.** `HandleDynamics()` precedes `GetInput`
   (`Play.c:449` vs `452`, `Play.c:475` vs `478`). Reordering changes collision
   outcomes.
4. **Input for both gliders is gathered before either interaction is resolved**
   (`Play.c:452-454`), so player 2's keypress in frame *N* is visible to
   `HandleInteraction` in frame *N*, not *N+1*.
5. **`HandleTriggers`/`HandleBands` run even when `gameOver`** (`Play.c:456-457`,
   `484-485`) — deliberately, so in-flight rubber bands finish their arcs during
   the death countdown.
6. **`RenderFrame` is guarded by `playing`, not by `gameOver`**
   (`Play.c:463`, `488`), so the world keeps animating for `countDown` frames
   after death.
7. The `gameOver` countdown block (`Play.c:499-553`) runs *after* rendering, so
   the last rendered frame is drawn before the game-over animation takes over.

### 6.3 `HandlePlayEvent` — the in-game event pump

`GliderPRO/Sources/Play.c:387-426`. Called only when `doBackground` is true.

```
 1. (391) long sleep = 2
 2. (393) if WaitNextEvent(everyEvent, &theEvent, sleep, nil):
          // not a switch: a two-arm if / else-if chain
 3.   (395-396) if theEvent.what == updateEvt
                  and (WindowPtr)theEvent.message == mainWindow:
 4.     (398)      GetPort(&wasPort)
 5.     (399)      SetPortWindowPort(mainWindow)
 6.     (400)      BeginUpdate(mainWindow)
 7.     (401-403)  CopyBits(workSrcMap -> mainWindow, justRoomsRect, justRoomsRect, srcCopy, nil)
 8.     (404)      RefreshScoreboard(kNormalTitleMode)
 9.     (405)      EndUpdate(mainWindow)
10.     (406)      SetPort(wasPort)
11.   (408) else if theEvent.what == osEvt and (message & 0x01000000):
12.     (410)      if message & 0x00000001:                  // resume
13.       (412)        switchedOut = false
14.       (413)        ToggleMusicWhilePlaying()
15.       (414)        HideCursor()
16.     (417)      else:                                     // suspend
17.       (419)        InitCursor()
18.       (420)        switchedOut = true
19.       (421)        ToggleMusicWhilePlaying()
```

Note the update arm saves and restores the port (`398`/`406`) — unlike
`HandleUpdateEvent` in the outer loop, which just `SetPort`s and leaves it.

Only two event kinds are handled: window update and suspend/resume. **Key and
mouse events are deliberately ignored** — all in-game input goes through
`GetKeys` polling in `GetInput` (`Input.c:284`), never through the event queue.
This is why a port must keep a *polled* keyboard model for gameplay even if it
uses events for menus.

`Boolean doBackground` is defined at `Play.c:54`, preference-backed
(`prefsInfo.wasDoBackground`, `Externs.h:265`), and defaults to **false**
(`Main.c:186`). So in the shipped default configuration the game **does not
process any events at all while playing** — you cannot bring another app
forward, you cannot get an update event, and a quit Apple Event cannot arrive.
The only escape is the polled Command-Q handler (§6.6).

### 6.4 `NewGame` teardown

Resuming `Play.c` after `PlayGame()` returns:

```
 1. (216-218) #ifdef CREATEDEMODATA: DumpToResEditFile(demoData, sizeof(demoType)*demoIndex)
              — off in the ship build
 2. (220)     isPlayMusicGame = wasPlayMusicPref         // saved at Play.c:193;
              a Stereo object can flip isPlayMusicGame mid-game, so it is restored here
 3. (221)     ZeroMirrorRegion()
 4. (223-230) #ifdef COMPILEQT, if hasQT && hasMovie && tvInRoom:
                tvInRoom = false; StopMovie(theMovie); SetMovieActive(theMovie, false)
 5. (232)     twoPlayerGame = false
 6. (233)     theMode = kSplashMode                      // <-- back to idle
 7. (234)     InitCursor()
 8. (235-252) if isPlayMusicIdle: StartMusic() (+ YellowAlert on failure)
              else if isMusicOn:  StopTheMusic()
 9. (253)     NilSavedMaps()
10. (254)     SetPortWindowPort(mainWindow)
11. (255)     BlackenScoreboard()
12. (256)     UpdateMenus(false)
13. (258-274) if !gameOver:                              // player quit early
                (263)    GetGWorld(&wasCPort, &wasWorld)
                (265)    InvalWindowRect(mainWindow, &mainWindowRect)
                (267)    SetGWorld(workSrcMap, nil)
                (268)    PaintRect(&workSrcRect)
                (269)    QSetRect(&tempRect, 0, 0, 640, 460)
                (270)    QOffsetRect(&tempRect, splashOriginH, splashOriginV)
                (271)    LoadScaledGraphic(kSplash8BitPICT, &tempRect)
                (273)    SetGWorld(wasCPort, wasWorld)
14. (275)     WaitCommandQReleased()
15. (276)     demoGoing = false
16. (277)     incrementModeTime = TickCount() + kIdleSplashTicks
```

Notes:

- `twoPlayerGame` is forcibly cleared (step 5) so the next `New Game` is
  single-player unless the user picks Two Player again.
- The `!gameOver` branch (step 13) restores the splash *only when the player
  bailed out*; if the game ended naturally, `DoGameOver`/`DoDiedGameOver` has
  already called `RedrawSplashScreen()` (`GameOver.c:68`, `GameOver.c:505`).
- `WaitCommandQReleased()` (`Utilities.c:485-499`) spins until Command or Q is
  physically released, then `FlushEvents`. Without it, the Command-Q that ended
  the game would immediately be seen by the outer loop as a Quit menu command.
- `incrementModeTime` is reset last, so the attract timer restarts from the
  moment the game ends.

### 6.5 Pause

`DoPause()` (`GliderPRO/Sources/Input.c:77-117`) is called from *inside*
`GetInput` (`Input.c:373-377`) and `GetDemoInput` (`Input.c:271-275`), i.e. from
inside the frame loop, and it blocks there.

Which key pauses is a preference: `Boolean isEscPauseKey` (`Input.c:34`),
default **false** (`Main.c:184`), meaning **Tab** pauses by default and Escape
pauses if the preference is set.

| `isEscPauseKey` | Pause key | KeyMap offset | Overlay PICT |
|---|---|---|---|
| `false` (default) | Tab | `kTabKeyMap` = 55 (`Externs.h:159`) | `kTabPausePictID` = 1016 (`Input.c:18`) |
| `true` | Esc | `kEscKeyMap` = 50 (`Externs.h:156`) | `kEscPausePictID` = 1015 (`Input.c:17`) |

Algorithm:

```
 1. (81)     SetPort((GrafPtr)mainWindow)
 2. (82)     QSetRect(&bounds, 0, 0, 214, 54)
 3. (83)     CenterRectInRect(&bounds, &houseRect)
 4. (84-87)  LoadScaledGraphic(isEscPauseKey ? kEscPausePictID : kTabPausePictID, &bounds)
 5. (89-94)  do { GetKeys(theKeys); } while (pause key still down)   // wait for release
 6. (96)     paused = true
 7. (97)     while (paused):
 8.   (99)     GetKeys(theKeys)
 9.   (100)    if BitTst(&theKeys, pause key): paused = false
10.   (103)    else if BitTst(&theKeys, kCommandKeyMap): DoCommandKey()
11. (107-109) CopyBits(workSrcMap -> mainWindow) over the same 214x54 bounds  // erase overlay
12. (111-116) do { GetKeys(theKeys); } while (pause key still down)   // wait for release again
```

Properties a port must reproduce:

- The pause loop is a **tight spin with no sleep and no `TickCount` throttle**.
  On modern hardware this pegs a core. A port should sleep ~16 ms per iteration;
  this is unobservable to the player.
- `gameFrame` does **not** advance while paused, and `nextFrame` is **not**
  updated. So on unpause, `TickCount()` is far past `nextFrame` and the very
  next `RenderFrame` does zero waiting — one frame runs "instantly" and then
  pacing resumes. Harmless, but it means you cannot assert
  `nextFrame >= TickCount()` as an invariant.
- The pause overlay is 214×54 (verified: `PICT` 1015 and 1016 are both
  214×54, [Appendix A.3](#a3-pict-dimensions)) and is centred in `houseRect`,
  i.e. in the play area *excluding* the scoreboard.
- Erasing the overlay copies from `workSrcMap`, which is why the pause dialog
  must never be composited into `workSrcMap`.
- Command keys still work while paused (step 10) — this is how you save or quit
  from the pause screen.

### 6.6 In-game command keys

`DoCommandKey()` (`GliderPRO/Sources/Input.c:53-73`). Called from `GetInput`
(`Input.c:286-287`), `GetDemoInput` (`Input.c:205-206`, non-arcade only) and
`DoPause` (`Input.c:103-104`).

| Key | KeyMap offset | Lines | Behaviour |
|---|---|---|---|
| Q | `kQKeyMap` = 11 (`Externs.h:146`) | `55-64` | `playing = false; paused = false;` then, if `!twoPlayerGame && !demoGoing`, `if (QuerySaveGame()) SaveGame2();` |
| S | `kSKeyMap` = 6 (`Externs.h:148`) | `67-72` | if `!twoPlayerGame`: `RefreshScoreboard(kSavingTitleMode)`, `SaveGame2()`, `HideCursor()`, `CopyRectWorkToMain(&workSrcRect)`, `RefreshScoreboard(kNormalTitleMode)` |

`QuerySaveGame()` (`Input.c:383-397`) shows `ALRT` 1041 (`kSaveGameAlert`) and
returns true for button 1 (`kYesSaveGameButton`).

`kSavingTitleMode` = 2, `kNormalTitleMode` = 0, `kEscapedTitleMode` = 1
(`GliderDefines.h:619-621`).

Note **Command-Q is the only in-game quit**, it is polled (not event-driven),
and it is the reason `WaitCommandQReleased()` exists in `NewGame`'s teardown.

### 6.7 Game over

`FlagGameOver()` (`GliderPRO/Sources/GameOver.c:237-242`) is the single entry
point:

```c
239  gameOver = true;
240  countDown = kNumCountDownFrames;
241  SetMusicalMode(kPlayWholeScoreMode);
```

`kNumCountDownFrames` = 16 (`GameOver.c:17`), `kPlayWholeScoreMode` = -1
(`GliderDefines.h:50`). So there are exactly **16 more simulated frames**
(16 × 2 ticks ≈ 533 ms) between the triggering event and the game-over
animation.

Which animation runs is decided at `Play.c:546-549` by the sign of `mortals`:

| Condition | Function | Meaning |
|---|---|---|
| `mortals < 0` | `DoDiedGameOver()` (`GameOver.c:443-506`) | ran out of gliders |
| `mortals >= 0` | `DoGameOver()` (`GameOver.c:60-69`) | completed the house |

`DoGameOver` (house completed):

```
1. (62) playing = false
2. (63) SetUpFinalScreen()
3. (65) ColorRect(&mainWindowRect, 244)          // colour index 244
4. (66) DoGameOverStarAnimation()
5. (67) if !TestHighScore(): RedrawSplashScreen()
```

`DoDiedGameOver` (out of gliders):

```
 1. (451)     InitDiedGameOver()
 2. (452)     CopyRectMainToWork(&workSrcRect)     // workSrcRect, not mainWindowRect
 3. (453)     CopyRectMainToBack(&workSrcRect)
 4. (454)     FlushEvents(everyEvent, 0)
 5. (456)     nextLoop = TickCount() + 2
 6. (457)     while (pagesStuck < 8):
 7.   (459)     HandlePages()
 8.   (460)     DrawPages()
 9.   (461-477) do { poll } while (TickCount() < nextLoop):   // `while` at :477
                  (464-469) if Cmd/Option/Shift/Control down: pagesStuck = 8; userAborted = true
                  (470-475) if a mouseDown/keyDown arrives:   pagesStuck = 8; userAborted = true
10.   (478)     nextLoop = TickCount() + 2
11. (481-491) DisposeRgn(roomRgn); DisposeGWorld(pageSrcMap/pageMaskMap/gameOverSrcMap)
12. (492)     playing = false
13. (494-504) if demoGoing:
                  if !userAborted: WaitForInputEvent(1)      // 497
              else:
                  if !userAborted: WaitForInputEvent(10)     // 502
                  TestHighScore()                            // 503
14. (505)     RedrawSplashScreen()
```

Both animations run **their own independent 2-tick loops** and do not use
`RenderFrame`'s `nextFrame`. `DoGameOverStarAnimation` (`GameOver.c:136-230`)
does the same: `nextLoop = TickCount() + 2` at `GameOver.c:149` and `220`, with
`while (TickCount() < nextLoop)` at `GameOver.c:219`; it gives up after
`pass >= 80` and then calls `WaitForInputEvent(5)` (`GameOver.c:222-228`, the call itself at `:226`).
`kStarFalls` = 8 (`GameOver.c:138`) and a star is spawned every 32 pixels of
angel travel (`GameOver.c:156-165`).

The post-game wait is asymmetric on purpose: **1 second in demo mode, 10 seconds
in a real game** (`GameOver.c:497`, `502`), and only a real game gets
`TestHighScore()`.

`WaitForInputEvent(short seconds)` (`Utilities.c:439-479`):

```
1. (446) timeToBail = TickCount() + 60L * (long)seconds
2. (447) FlushEvents(everyEvent, 0)
3. (451) while waiting:
4.   (453)  GetKeys(theKeys); if Cmd/Option/Shift/Control down: waiting = false
5.   (457)  if GetNextEvent(...) and (mouseDown or keyDown): waiting = false
6.   (461)  if osEvt and resume: didResume = true; waiting = false
7.   (470)  if osEvt and suspend: InitCursor()
8.   (474)  if seconds != -1 and TickCount() >= timeToBail: waiting = false
9. (477) FlushEvents(everyEvent, 0)
10.      return didResume
```
A `seconds` value of `-1` means "wait forever".

---

## 7. Frame and tick pacing

### 7.1 The buffers

Three surfaces, created in `CreateOffscreens` (`StructuresInit2.c:145-181`):

| Surface | Type | Rect global | Created at | Purpose |
|---|---|---|---|---|
| `backSrcMap` | `GWorldPtr` (`Room.c:26`) | `backSrcRect` (`Room.c:25`) | `StructuresInit2.c:160-162` | The static room artwork: background tiles, non-animated objects. Written once per room transition by `DrawLocale()`. |
| `workSrcMap` | `GWorldPtr` (`MainWindow.c:36`) | `workSrcRect` (`MainWindow.c:35`) | `StructuresInit2.c:156-158` | The composite frame: `backSrcMap` content plus everything dynamic. |
| `mainWindow` | `WindowPtr` (`MainWindow.c:38`) | `mainWindowRect` (`MainWindow.c:37`) | `MainWindow.c:225` | The screen. |

```c
153  justRoomsRect = houseRect;
154  ZeroRectCorner(&justRoomsRect);
156  workSrcRect = houseRect;
157  ZeroRectCorner(&workSrcRect);
158  CreateOffScreenGWorld(&workSrcMap, &workSrcRect, kPreferredDepth);
160  backSrcRect = houseRect;
161  ZeroRectCorner(&backSrcRect);
162  CreateOffScreenGWorld(&backSrcMap, &backSrcRect, kPreferredDepth);
```

So `workSrcRect` and `backSrcRect` are both exactly `houseRect` normalised to
origin (0,0). `kPreferredDepth` = 8 (`Externs.h:15`). `justRoomsRect` is the
same rect and is what gets blitted to the window during in-game update events
(`Play.c:401-403`).

The rest of `CreateOffscreens`, in order, each followed by `SpinCursor(1)`:
`InitScoreboardMap()` (`164`), `InitGliderMap()` (`165`), `InitBlowers()`
(`166`), `InitFurniture()` (`167`), `InitPrizes()` (`168`), `InitTransports()`
(`169`), `InitSwitches()` (`170`), `InitLights()` (`171`), `InitAppliances()`
(`172`), `InitEnemies()` (`173`), `InitClutter()` (`174`), `InitSupport()`
(`175`), `InitAngel()` (`176`); then `QSetRect(&tileSrcRect, 0, 0, 128, 80);`
and `tileSrcMap = nil;` (`178-179`).

### 7.2 The dirty-rect lists

`GliderPRO/Sources/Render.c:20` defines the capacity:

```c
20  #define kMaxGarbageRects	48
```

and `Render.c:35-42` the state:

```c
35  Rect		work2MainRects[kMaxGarbageRects];
36  Rect		back2WorkRects[kMaxGarbageRects];
37  Rect		shieldRect;
38  RgnHandle	mirrorRgn;
39  Point		shieldPt;
40  long		nextFrame;
41  short		numWork2Main, numBack2Work;
42  Boolean		hasMirror;
```

| Function | Lines | Behaviour |
|---|---|---|
| `AddRectToWorkRects(Rect *theRect)` | `Render.c:65-80` | intersect with `justRoomsRect`; append to `work2MainRects` if `numWork2Main < kMaxGarbageRects - 1` (i.e. **max 47 entries**) |
| `AddRectToBackRects(Rect *theRect)` | `Render.c:84-99` | intersect with `workSrcRect`; append to `back2WorkRects` under the same bound |
| `AddRectToWorkRectsWhole(Rect *theRect)` | `Render.c:103-132` | as above but rejects rects that are entirely offscreen or degenerate |

Semantics: `back2WorkRects` are regions of `workSrcMap` that must be *restored*
from the clean `backSrcMap` (erasing last frame's sprites); `work2MainRects` are
regions of `workSrcMap` that must be *presented* to the window.

`CopyRectsQD()` (`Render.c:616-635`) performs both passes:

```
1. (620) for i in 0 .. numWork2Main-1:
2.   (622)  CopyBits(workSrcMap -> mainWindow, work2MainRects[i], work2MainRects[i], srcCopy, nil)
3. (628) for i in 0 .. numBack2Work-1:
4.   (630)  CopyBits(backSrcMap -> workSrcMap, back2WorkRects[i], back2WorkRects[i], srcCopy, nil)
```

Note the ordering: **present first, then erase**. The erase pass prepares
`workSrcMap` for the *next* frame; it is not part of this frame's presentation.

The `Render.c` prototype list (`GliderProtos.h:363-373`) also names
`DirectWork2Main8`, `DirectBack2Work8`, `DirectGeneric2Work8`, the 4-bit
variants, `CopyRectsAssm`, `DirectFillBack8/Work8/Back4/Work4` — a hand-written
direct-to-pixmap fast path. In the committed source `RenderFrame` calls
`CopyRectsQD` only; the direct paths are the legacy 68k optimisation and are not
on the live path.

Single-rect helpers, all plain `CopyBits(..., srcCopy, nil)`:

| Function | Lines | Direction |
|---|---|---|
| `CopyRectBackToWork` | `Render.c:695-700` | back → work |
| `CopyRectWorkToBack` | `Render.c:704-709` | work → back |
| `CopyRectWorkToMain` | `Render.c:713-718` | work → window |
| `CopyRectMainToWork` | `Render.c:722-727` | window → work |
| `CopyRectMainToBack` | `Render.c:731-736` | window → back |

The mirror region (an object type `kMirror` = 0x82, `GliderDefines.h:422`) is a
`RgnHandle` accumulated across the room's mirrors:

```c
740  void AddToMirrorRegion (Rect *theRect)   // NewRgn / RectRgn / UnionRgn
760      hasMirror = true;
765  void ZeroMirrorRegion (void)             // DisposeRgn; mirrorRgn = nil; hasMirror = false
```
(`Render.c:740-761`, `Render.c:765-771`.)

### 7.3 `RenderFrame` — the pacer

`GliderPRO/Sources/Render.c:639-671`, quoted in full because every line matters:

```c
639  void RenderFrame (void)
640  {
641      if (hasMirror)
642      {
643          DrawReflection(&theGlider, true);
644          if (twoPlayerGame)
645              DrawReflection(&theGlider2, false);
646      }
647      HandleGrease();
648      RenderPendulums();
649      if (evenFrame)
650          RenderFlames();
651      else
652          RenderStars();
653      RenderDynamics();
654      RenderFlyingPoints();
655      RenderSparkles();
656      RenderGlider(&theGlider, true);
657      if (twoPlayerGame)
658          RenderGlider(&theGlider2, false);
659      RenderShreds();
660      RenderBands();
661
662      while (TickCount() < nextFrame)
663      {
664      }
665      nextFrame = TickCount() + kTicksPerFrame;
666
667      CopyRectsQD();
668
669      numWork2Main = 0;
670      numBack2Work = 0;
671  }
```

With `kTicksPerFrame` = **2** (`GliderDefines.h:533`) and one tick = 1/60 s:

- **Nominal frame rate: 60 / 2 = 30 frames per second.**
- **Simulation rate: 30 updates per second**, because `gameFrame` advances once
  per `PlayGame` iteration (`Play.c:434`) and there is exactly one `RenderFrame`
  per iteration.
- **The logic is tick-locked, not frame-rate-independent.** There is no delta
  time anywhere in the codebase. Every velocity, counter and timer is
  "per frame" and therefore implicitly "per 2 ticks".

Three subtleties a port must get right:

1. **The wait is a hard busy-loop** (`Render.c:662-664`) with an empty body. It
   burns CPU. A port should sleep, but must sleep to the *same deadline*.
2. **`nextFrame` is recomputed from `TickCount()` after the wait**
   (`Render.c:665`), not as `nextFrame += kTicksPerFrame`. Consequence: the
   scheduler has **no catch-up and no accumulated credit**. If a frame overruns
   by *k* ticks, that time is permanently lost; the game runs slower rather than
   fast-forwarding. Implementing `nextFrame += kTicksPerFrame` instead would
   change observed behaviour on slow frames (the game would sprint to catch up).
   Reproduce the original: recompute from "now".
3. **The wait happens *between* simulation and presentation**, not at the top of
   the loop. So the pacing is: simulate frame *N*, draw sprites into
   `workSrcMap`, *then* wait out the remainder of frame *N-1*'s budget, then
   present. That means presentation of frame *N* happens at the deadline of
   frame *N*, and input for frame *N+1* is sampled immediately after — i.e.
   input latency is one frame plus whatever slack exists.

`nextFrame` is seeded in `InitGarbageRects` (`Render.c:675-691`):

```
1. (679) numWork2Main = 0
2. (680) numBack2Work = 0
3. (682) numSparkles = 0;   for i in 0..kMaxSparkles-1:   sparkles[i].mode = -1
4. (686) numFlyingPts = 0;  for i in 0..kMaxFlyingPts-1:  flyingPoints[i].mode = -1
5. (690) nextFrame = TickCount() + kTicksPerFrame
```

### 7.4 Every timing constant in the program

| Constant / expression | Value | Where | Meaning |
|---|---|---|---|
| `kTicksPerFrame` | 2 | `GliderDefines.h:533` | in-game frame budget ⇒ 30 fps |
| `kIdleSplashTicks` | 7200 (`7200L`) | `GliderDefines.h:197` | attract-mode countdown = 120 s |
| `sleep` (outer loop) | 2 | `Events.c:483` | `WaitNextEvent` max block ≈ 33 ms |
| `sleep` (in-game pump) | 2 | `Play.c:391` | ditto |
| `nextLoop = TickCount() + 2` | 2 | `GameOver.c:149`, `220`, `456`, `478` | game-over animations, also 30 fps |
| `DelayTicks(6)` | 6 | `Main.c:344` | 100 ms pause after the startup chirp |
| `WaitForInputEvent(1)` | 60 | `GameOver.c:497` | 1 s post-demo wait |
| `WaitForInputEvent(10)` | 600 | `GameOver.c:502` | 10 s post-game wait |
| `WaitForInputEvent(5)` | 300 | `GameOver.c:226` | 5 s after the star animation |
| `kNumCountDownFrames` | 16 | `GameOver.c:17` | frames of continued simulation after game over ≈ 533 ms |
| `kPageFrames` | 14 | `GameOver.c:18` | animation frames in the falling-pages art |
| `kRingDelay` | 90 | `Play.c:20` | telephone: frames between rings within a burst = 3 s |
| `kRingSpread` | 25000 | `Play.c:21` | telephone: random spread, in frames, until the next call |
| `kRingBaseDelay` | 5000 | `Play.c:22` | telephone: minimum frames until the next call |
| `kChimeDelay` | 180 | `Play.c:23` | wind-chime spacing, in frames = 6 s |
| `doubleTime` | `GetDblTime()` | `InterfaceInit.c:184` | user's double-click threshold, ticks |
| `batteryFrame` throttle | every 4th frame | `Input.c:121-156`, `160-182` | thrust/hiss sound repeat rate |
| `kLengthOfZap` | 30 | `GliderDefines.h:546` | frames an electrical zap lasts |
| `kStartSparkle` | 4 | `GliderDefines.h:545` | initial sparkle animation frame |

The telephone/chime timers all live in `phoneType { short nextRing, rings, delay; }`
(`Play.c:26-31`) and every one of their counters is decremented at most once per
call to `HandleTelephone`, which `PlayGame` calls exactly once per frame
(`Play.c:445`). So **all three fields count frames, not ticks** — `kRingDelay` 90
is 3 s and `kChimeDelay` 180 is 6 s at 30 fps, and a port that runs the
simulation at a different rate changes the phone and chime cadence. `InitTelephone`
is at `Play.c:733-740`, `HandleTelephone` at `Play.c:744-789`, `StrikeChime` at
`Play.c:793-796`.

Note also that the two `phoneType` instances use different subsets of the struct:
`thePhone` uses all three fields, `theChimes` uses only `nextRing`
(`Play.c:739`, `771-788`), and `theChimes.nextRing` is reseeded from
`kChimeDelay / numChimes` (floor 2) rather than `kChimeDelay` directly
(`Play.c:780-784`), so more chime objects in a room ring more often.

### 7.5 Is anything frame-rate independent?

No. Concretely:

- Glider velocities are integers in pixels/frame (`gliderType.hVel`, `vVel`,
  `GliderStructs.h:207`).
- Thrust constants are per-frame deltas: `kNormalThrust` 5, `kHyperThrust` 8,
  `kHeliumLift` 4 (`Input.c:14-16`).
- Demo playback keys off `gameFrame` equality (`Input.c:224`).
- `countDown` is a frame counter (`GameOver.c:240`, `Play.c:501`).
- `evenFrame` alternates per frame (`Play.c:435`).

Therefore a Go port **must run its simulation at exactly 30 Hz** in fixed steps.
If you want a higher render rate, you must interpolate presentation only, and
you must accept that `RenderFrame` in the original *is* both the simulation-step
boundary and the present.

---

## 8. Attract ("demo") mode

### 8.1 The trigger

Bottom of `HandleEvent` (`GliderPRO/Sources/Events.c:536-540`):

```c
536  if ((theMode == kSplashMode) && doAutoDemo && !switchedOut)
537  {
538      if (TickCount() >= incrementModeTime)
539          DoDemoGame();
540  }
```

Three conditions:

| Condition | Global | Defined | Default | Notes |
|---|---|---|---|---|
| in idle mode | `theMode == kSplashMode` | `MainWindow.c:42` | 0 | never fires in edit mode |
| enabled | `doAutoDemo` (`Boolean`) | `Events.c:31` | **true** (`Main.c:183`) | preference `prefsInfo.wasDoAutoDemo` (`Externs.h:264`) |
| foregrounded | `!switchedOut` | `Events.c:31` | false | see §5.6 |

`long incrementModeTime` (`Events.c:27`) is the deadline. Every write to it,
exhaustively:

| Site | When |
|---|---|
| `InterfaceInit.c:163` | boot |
| `Events.c:427` | on resume from background |
| `Play.c:277` | end of `NewGame` (any game ends, including a demo) |
| `Play.c:302` | end of `DoDemoGame` |
| `Menu.c:336` | Game ▸ Load House |
| `Menu.c:402` | Options ▸ Room Editor, leaving edit mode |
| `Menu.c:419` | Options ▸ High Scores |
| `Menu.c:424` | Options ▸ Preferences |
| `AppleEvents.c:111` | a document was opened via Apple Event |

All nine writes are the same expression:
`incrementModeTime = TickCount() + kIdleSplashTicks;` with `kIdleSplashTicks` =
`7200L` = **120 seconds** (`GliderDefines.h:197`).

Note what is **not** in the list: mouse movement, key presses, menu tracking,
window drags. Sitting on the splash screen and wiggling the mouse for two
minutes *will* launch the demo. Only the specific actions above defer it.

### 8.2 `DoDemoGame`

`GliderPRO/Sources/Play.c:282-303`:

```
 1. (287) wasHouseIndex = thisHouseIndex
 2. (288) whoCares = CloseHouse()                      // return value DISCARDED:
                                                       // there is no abort path here
 3. (289) thisHouseIndex = demoHouseIndex
 4. (290) PasStringCopy(theHousesSpecs[thisHouseIndex].name, thisHouseName)
 5. (291) if OpenHouse():
 6.   (293)   whoCares = ReadHouse()
 7.   (294)   demoGoing = true
 8.   (295)   NewGame(kNewGameMode)                    // theMode := kPlayMode
 9. (297) whoCares = CloseHouse()                      // close the DEMO house first...
10. (298) thisHouseIndex = wasHouseIndex               // ...then restore the index
11. (299) PasStringCopy(theHousesSpecs[thisHouseIndex].name, thisHouseName)
12. (300) if OpenHouse(): whoCares = ReadHouse()        // restore the user's house
13. (302) incrementModeTime = TickCount() + kIdleSplashTicks
```

Every `CloseHouse()`/`ReadHouse()` result is assigned to a throwaway `Boolean
whoCares` (`Play.c:285`) and never tested, so **the demo cannot be vetoed** — not
by a failed save, not by a failed read. The only guard is `if (OpenHouse())`. Note
also the close-then-restore order: `CloseHouse()` at `:297` runs *before*
`thisHouseIndex` is put back at `:298`, i.e. it closes the demo house, not the
user's.

The demo therefore *swaps the loaded house*, runs a real game, and swaps back.
`demoGoing` is set here (step 7) and cleared in `NewGame`'s teardown
(`Play.c:276`).

`short demoHouseIndex` is defined at `SelectHouse.c:52` and located by name in
`DoDirSearch` (`SelectHouse.c:636-644`):

```c
636  demoHouseIndex = -1;
     for (...)
         if (EqualString(theHousesSpecs[i].name, "\pDemo House", false, true))
         { demoHouseIndex = i; break; }
```

That is a case-insensitive, diacritical-insensitive Pascal-string compare
against the literal `"Demo House"`. **A port must ship a house file with exactly
that name** or the attract mode will index `theHousesSpecs[-1]`. Note there is
no guard: `DoDemoGame` does not test `demoHouseIndex >= 0` before using it.

### 8.3 The `'demo'` resource — verified byte layout

`demoType` (`GliderPRO/Headers/GliderStructs.h:334-339`):

```c
334  typedef struct
335  {
336      long		frame;
337      char		key;
338      char		padding;
339  } demoType, *demoPtr;
```

Under `mac68k`/2-byte alignment `sizeof(demoType)` = **6** (verified — see
[Appendix A.2](#a2-struct-sizes-and-offsets)): a 4-byte big-endian `long`
followed by two signed bytes.

Empirically parsed from `data 'demo' (128)` in `GliderPRO/Glider PRO.r`:

| Property | Observed |
|---|---|
| Resource size | **6702 bytes**, exactly `kDemoLength` (`GliderDefines.h:625`) |
| Record count | **1117** (= 6702 / 6) |
| First 8 records `(frame, key, padding)` | `(46,0,114) (47,0,114) (48,0,114) (49,0,69) (56,0,6) (57,0,114) (58,0,111) (59,0,114)` |
| Last 4 records | `(3411,0,254) (3412,0,248) (3413,0,7) (3414,0,1)` |
| Frame range | 46 … 3414 |
| Frames strictly increasing? | **yes** (all 1116 successive pairs) |
| Frame gaps | min 1, max 100, mean 3.02 |
| Key histogram | `{0: 910, 1: 198, 3: 9}` |
| Distinct `padding` values | **109** (0…254 observed) — pure garbage |
| Demo length | 3414 frames ⇒ 3414 / 30 ≈ **113.8 seconds** of play |

The `padding` byte is garbage because `LogDemoKey` never writes it
(`Input.c:44-49`):

```c
46  demoData[demoIndex].frame = gameFrame;
47  demoData[demoIndex].key = keyIs;
48  demoIndex++;
```

So the field's value is whatever was in the freshly-`NewPtr`'d block when the
recording build ran. **A port must ignore `padding` entirely.** If you
round-trip the resource, do not assume it is zero.

Key `2` (battery) never appears: the shipped demo contains **no battery
presses**, only 910 right-thrusts, 198 left-thrusts and 9 rubber bands.

### 8.4 `GetDemoInput` — playback

`GliderPRO/Sources/Input.c:186-277`.

```
 1. (190) GetKeys(theKeys)
 2. (192) #if BUILD_ARCADE_VERSION
 3.   (194) if BitTst(&theKeys, thisGlider->leftKey)  or
 4.         BitTst(&theKeys, thisGlider->rightKey) or
 5.         BitTst(&theKeys, thisGlider->battKey)  or
 6.         BitTst(&theKeys, thisGlider->bandKey):
 7.     (199)   playing = false
 8.     (200)   paused  = false
              // NOTE: there is no `return` here; execution falls through
              // into the physics below for one final frame
 9.       (203) #else
10.   (205) if BitTst(&theKeys, kCommandKeyMap): DoCommandKey()
11.       (208) #endif
12. (211) if thisGlider->mode == kGliderBurning:      // 9
13.   (213-216) hDesiredVel += (facing == kFaceRight) ? kNormalThrust : -kNormalThrust
                              // kNormalThrust (5), NOT kHyperThrust
14. (218) else:
16.   (220) thisGlider->heldLeft  = false
17.   (221) thisGlider->heldRight = false
18.   (222) thisGlider->tipped    = false
19.   (224) if gameFrame == (long)demoData[demoIndex].frame:
20.     (226)   switch demoData[demoIndex].key:
21.       (228)   case 0:   // labelled "left key" but IS the right-key action
22.                          hDesiredVel += kNormalThrust
23.                          tipped = (facing == kFaceLeft)
24.                          heldRight = true
25.                          fireHeld = false
26.       (235)   case 1:   // labelled "right key" but IS the left-key action
27.                          hDesiredVel -= kNormalThrust
28.                          tipped = (facing == kFaceRight)
29.                          heldLeft = true
30.                          fireHeld = false
31.       (242)   case 2:   if batteryTotal > 0: DoBatteryEngaged(thisGlider)
                              else:               DoHeliumEngaged(thisGlider)
                              fireHeld = false
32.       (250)   case 3:   band:    fire a rubber band if !fireHeld
33.     (266)   demoIndex++
34.   (268) else:
35.     (269)   thisGlider->fireHeld = false
36. (271-272) if BitTst(&theKeys, isEscPauseKey ? kEscKeyMap : kTabKeyMap):
37.   (274)   DoPause()
```

**The case comments in the original are wrong.** `Input.c:228` is commented
`// left key` but its body is byte-for-byte the *right*-key branch of `GetInput`
(`Input.c:301-317`, which logs `LogDemoKey(0)` at `Input.c:304`); `Input.c:235`
is commented `// right key` but matches the *left*-key branch (`Input.c:318-326`,
which logs `LogDemoKey(1)` at `Input.c:321`). The recording and playback agree
with each other, so the demo plays back correctly; only the comments lie.
Document the behaviour, not the comment.

The authoritative key encoding, cross-checked against the `LogDemoKey` call
sites in `GetInput`:

| `key` byte | Recorded at | Meaning |
|---|---|---|
| 0 | `Input.c:304` (`LogDemoKey(0)`) | right thrust (`hDesiredVel += kNormalThrust`) |
| 1 | `Input.c:321` (`LogDemoKey(1)`) | left thrust (`hDesiredVel -= kNormalThrust`) |
| 2 | `Input.c:334` (`LogDemoKey(2)`) | battery / helium engaged |
| 3 | `Input.c:348` (`LogDemoKey(3)`) | fire rubber band |

### 8.5 Two latent bugs in demo playback

Both are real in the original and both must be *decided about* by a porter.

**(a) One record per frame maximum.** The dispatch is
`if (gameFrame == demoData[demoIndex].frame)` with a single `demoIndex++`
(`Input.c:224`, `266`). If two records shared a frame number, the second would
never match — `gameFrame` would have moved on — and `demoIndex` would be stuck
forever, freezing all further input. The shipped data is strictly increasing
(verified, §8.3) so this never triggers, but it means the format cannot express
"two keys pressed in the same frame". Diagonal or thrust-plus-band inputs are
simply unrepresentable in a recorded demo.

**(b) Unbounded read past the end.** Nothing compares `demoIndex` against
1117 or against `kDemoLength / sizeof(demoType)`. After the last record
(frame 3414) is consumed, `demoIndex` becomes 1117 and every subsequent frame
evaluates `demoData[1117].frame`, which is 4 bytes **past the end of the
`NewPtr(kDemoLength)` block** (`StructuresInit2.c:287`; the resource is
`BlockMove`d into it at `:295`). On classic Mac OS this
read lands in the heap block header of the next block and is harmless in
practice; in Go it is either a panic or a silent garbage read. A port **must**
bound-check, and should treat "index past end" as "no input", which is what the
original does 99.99% of the time (the garbage `long` almost never equals the
current `gameFrame`).

The demo therefore normally ends by one of:
- the glider dying / the house being completed (`FlagGameOver` → `Play.c:499`),
- the arcade abort (any glider key, `Input.c:192-201`),
- `Command-Q` in the retail build (`Input.c:205-206` → `DoCommandKey`),
- the pause key, then Command-Q.

There is no "demo finished, stop cleanly" path: the demo simply plays out of
input and the glider drifts until it dies.

### 8.6 Demo mode differences summary

| Aspect | Real game | Demo |
|---|---|---|
| `demoGoing` | false | true |
| House | user's selection | `"Demo House"` (`SelectHouse.c:639`) |
| Input | `GetInput` (polled keyboard) | `GetDemoInput` (recorded stream) |
| Two-player possible | yes | no (`Play.c:294-295` never sets `twoPlayerGame`) |
| Command-Q quit | yes | retail only; arcade aborts on any glider key |
| Save game on quit | yes (`Input.c:60-64`) | suppressed by `!demoGoing` guard |
| Post-death wait | `WaitForInputEvent(10)` (`GameOver.c:502`) | `WaitForInputEvent(1)` (`GameOver.c:497`) |
| High score entry | `TestHighScore()` | skipped (`GameOver.c:494-504`) |
| Music | per `isPlayMusicGame` | same |

---

## 9. Screen resolution and window geometry

### 9.1 The environment's view of the screen

`thisMac.screen` is filled by `GetDeviceRect(&thisMac.screen)`
(`Environ.c:463` → `Environ.c:334-342`), which is literally
`*theRect = (*thisGDevice)->gdRect` — the *global* bounds of the main
`GDevice`, i.e. the main display's rectangle in global coordinates.
`thisGDevice` comes from `GetMainDevice()` (`Utilities.c:167-172`).

`thisMac.gray` is `GetRegionBounds(GetGrayRgn(), &(thisMac.gray))`
(`Environ.c:468-469`) — the union of all displays minus the menu bar. It is used
only to bound window growing (`Events.c:114-117`).

The game therefore **assumes a single fixed-size main display for its entire
run**. There is no resolution-change handler. `thisMac.screen` is captured once
in `CheckOurEnvirons` at startup and every derived rect
(`houseRect`, `mainWindowRect`, `workSrcRect`, `backSrcRect`, `playOriginH/V`,
`splashOriginH/V`, `localRoomsDest[]`, `shieldRect`) is computed from it. If the
display resolution changed at runtime, the offscreen buffers would be the old
size. A Go port that supports window resizing must either re-run the whole
`VariableInit`/`CreateOffscreens`/`OpenMainWindow` chain or lock the window size.

### 9.2 The three `WIND` resources (verified)

Decoded from `GliderPRO/Glider PRO.r`:

| ID | Constant | Resource bounds | `procID` | Window kind | Visible | GoAway | Title |
|---|---|---|---|---|---|---|---|
| 128 | `kMainWindowID` (`MainWindow.c:16`) | `t=0 l=0 b=384 r=512` (512×384) | **2** | `plainDBox` — no title bar, no controls, 1-px frame | 0 | 0 | `"Main Window"` |
| 129 | `kEditWindowID` (`MainWindow.c:17`) | `t=0 l=0 b=322 r=512` (512×322) | **4** | `noGrowDocProc` — titled document window, no grow box | 0 | 0 | `"Main Window"` |
| 130 | `kMenuWindowID` (`MainWindow.c:18`) | `t=0 l=0 b=20 r=640` (640×20) | **2** | `plainDBox` | 0 | 0 | `"New Window"` |

Raw bytes for WIND 128:
`0000 0000 0180 0200 0002 0000 0000 00000000 0b "Main Window"` — 8-byte
`boundsRect`, 2-byte `procID`, 2-byte `visible`, 2-byte `goAwayFlag`, 4-byte
`refCon`, Pascal title.

All three are created invisible and explicitly `ShowWindow`n after being sized
and moved, so there is no flash of a wrongly-sized window.

The resource bounds are essentially placeholders: every window is `SizeWindow`d
immediately after creation.

### 9.3 `OpenMainWindow` — exact geometry

`GliderPRO/Sources/MainWindow.c:174-261`.

Guard (`MainWindow.c:179-183`): if `mainWindow != nil`, raise
`YellowAlert(kYellowUnaccounted, 6)` and return — the function is not
idempotent.

**Edit mode** (`MainWindow.c:185-211`):

```
 1. (187-189) if menuWindow != nil: DisposeWindow(menuWindow); menuWindow = nil
 2. (191)     QSetRect(&mainWindowRect, 0, 0, 512, 322)
 3. (192)     mainWindow = GetNewCWindow(kEditWindowID, nil, kPutInFront)
 4. (193)     SizeWindow(mainWindow, 512, 322, false)
 5. (196-200) if OptionKeyDown(): isEditH = 3; isEditV = 41    // rescue a lost window
 6. (201)     MoveWindow(mainWindow, isEditH, isEditV, true)
 7. (202)     ShowWindow(mainWindow)
 8. (203)     SetPortWindowPort(mainWindow)
 9. (204)     ClipRect(&mainWindowRect)
10. (205-206) ForeColor(blackColor); BackColor(whiteColor)
11. (208)     whichRoom = GetFirstRoomNumber()
12. (209)     CopyRoomToThisRoom(whichRoom)
13. (210)     ReflectCurrentRoom(false)
```

The editor window is **exactly one room**: `kRoomWide` × `kTileHigh` = 512×322
(`GliderDefines.h:499`, `498`). Its position is the preference pair
`isEditH`/`isEditV` (`MainWindow.c:39`), defaulting to (3, 41)
(`Main.c:158-159`), which is also the Option-key rescue position.

**Splash / play mode** (`MainWindow.c:212-260`):

```
 1. (214-221) if menuWindow == nil:
 2.   (216)     menuWindow = GetNewCWindow(kMenuWindowID, nil, kPutInFront)
 3.   (217)     SizeWindow(menuWindow, RectWide(&thisMac.screen), 20, false)
 4.   (218)     MoveWindow(menuWindow, thisMac.screen.left, thisMac.screen.top, true)
 5.   (220)     ShowWindow(menuWindow)
 6. (222)     mainWindowRect = thisMac.screen
 7. (223)     ZeroRectCorner(&mainWindowRect)              // normalise to (0,0)
 8. (224)     mainWindowRect.bottom -= 20                   // comment: thisMac.menuHigh
 9. (225)     mainWindow = GetNewCWindow(kMainWindowID, nil, kPutInFront)
10. (226)     SizeWindow(mainWindow, width, height, false)
11. (228)     MoveWindow(mainWindow, thisMac.screen.left, thisMac.screen.top + 20, true)
12. (230)     ShowWindow(mainWindow)
13. (231)     SetPortWindowPort(mainWindow)
14. (232)     ClipRect(&mainWindowRect)
15. (234-236) ForeColor(black); BackColor(white); PaintRect(&mainWindowRect)
16. (238-240) splashOriginH = (screenWidth  - 640) / 2;  clamp >= 0
17. (241-243) splashOriginV = (screenHeight - 480) / 2;  clamp >= 0
18. (245-247) SetPort(workSrcMap); PaintRect(&workSrcRect); LoadGraphic(kSplash8BitPICT)
19. (259)     SetPortWindowPort(mainWindow)
```

So in play mode:

- `menuWindow` is a **640-wide-resource but screen-wide-sized, 20-px-tall
  `plainDBox` window pinned to the top-left of the screen**. Its only job is to
  paint over the real Mac menu bar (`UpdateMenuBarWindow` just
  `PaintRect`s it black, `MainWindow.c:160-169`). This is Glider PRO's
  full-screen trick: it never actually hides the menu bar, it covers it. The
  commented-out `ShowMenuBarOld`/`HideMenuBarOld` at `MainWindow.c:364-448` is
  the abandoned alternative.
- `mainWindow` is the screen minus the top 20 pixels, positioned at
  `(screen.left, screen.top + 20)`.
- The hardcoded `20` appears three times in `MainWindow.c` (`:217`, `:224`,
  `:229`) — plus once more in `NewGame` (`Play.c:144`) — and
  matches `kScoreboardTall` = 20 (`GliderDefines.h:515`) numerically but is a
  *different* quantity — the two 20s are the menu-bar height and the scoreboard
  height respectively. The source comments (`// thisMac.menuHigh`) confirm a
  `menuHigh` field was intended but never added to `macEnviron`.

**The 480 is 460 + 20, not a bug.** `splashOriginV` is computed against **480**
(`MainWindow.c:241`) while every place that actually draws the splash uses a
**640×460** rect:

| Site | Rect |
|---|---|
| `MainWindow.c:104-106` (`RedrawSplashScreen`) | `QSetRect(&tempRect,0,0,640,460)` offset by `splashOriginH/V` |
| `MainWindow.c:141-143` (`UpdateMainWindow`) | same |
| `Play.c:269-271` (`NewGame` teardown) | same |

`PICT` 1000 (`kSplash8BitPICT`, `GliderDefines.h:524`) is verified to be
**640×460** ([Appendix A.3](#a3-pict-dimensions)). The 20-pixel difference is
deliberate: all three sites offset the rect inside **`workSrcMap`**, whose rect
is `houseRect` zeroed (`StructuresInit2.c:156`), and `houseRect` is the screen
*minus* `kScoreboardTall` = 20 (`InterfaceInit.c:196-197`). So

```
splashOriginV = (screenHigh - 480) / 2
              = ((screenHigh - 20) - 460) / 2
              = (workSrcRect height - 460) / 2
```

which is exactly the centred position for a 460-tall image in the work map. The
same holds horizontally: `splashOriginH = (screenWide - 640) / 2` and
`workSrcRect` is `min(screenWide, kMaxViewWidth = 1536)` wide, so on any screen
narrower than 1536 the 640-wide art is exactly centred too. A port should
reproduce the formula against the *house* rect, not the screen rect, and the two
will agree.

`DrawOnSplash` (`MainWindow.c:56-94`) then stamps the house name onto the splash:

```
1. (58-63)  build "House: " + thisHouseName, appending " (QT)" if hasQT && hasMovie
2. (64-66)  TextSize(9); TextFace(1 /* bold */); TextFont(applFont)
3. (67)     MoveTo(splashOriginH + 436, splashOriginV + 314)
4. (76-79)  if thisMac.isDepth == 4: white DrawString
            else if houseIsReadOnly: ColorText(str, 5L)
            else:                    ColorText(str, 28L)
5. (82-93)  #ifdef powerc: "PowerPC Native!" at (+5,+457) black then (+4,+456) white
```

Colour indices 5 and 28 are raw 8-bit palette indices.

`UpdateMainWindow` (`MainWindow.c:120-155`):

| `theMode` | Lines | Behaviour |
|---|---|---|
| `kEditMode` | `128-136` | `PauseMarquee()`; `CopyBits(workSrcMap → mainWindow, mainWindowRect, mainWindowRect, srcCopy, visRgn-as-mask)`; `ResumeMarquee()` |
| `kSplashMode` or `kPlayMode` | `137-151` | repaint `workSrcMap`, `LoadScaledGraphic(kSplash8BitPICT, 640×460 @ splashOrigin)`, `CopyBits(workSrcRect → mainWindowRect)`, `DrawOnSplash()` |
| — | `154` | `splashDrawn = true` |

`splashDrawn` is what gates Game ▸ Load House (`Menu.c:327`): the item does
nothing until the splash has been painted at least once.

### 9.4 The play-area geometry

Computed in `VariableInit` (`InterfaceInit.c:196-218`):

```c
196  houseRect = thisMac.screen;
197  houseRect.bottom -= kScoreboardTall;              // 20
198  if (houseRect.right > kMaxViewWidth)              // 1536
199      houseRect.right = kMaxViewWidth;
200  if (houseRect.bottom > kMaxViewHeight)            // 1026
201      houseRect.bottom = kMaxViewHeight;
203  playOriginH = (RectWide(&thisMac.screen) - kRoomWide) / 2;   // (W - 512)/2
204  playOriginV = (RectTall(&thisMac.screen) - kTileHigh) / 2;   // (H - 322)/2
```

| Constant | Value | Where |
|---|---|---|
| `kRoomWide` | 512 (= `kNumTiles` 8 × `kTileWide` 64) | `GliderDefines.h:499`, `496-497` |
| `kTileHigh` | 322 | `GliderDefines.h:498` |
| `kVertLocalOffset` | 322 | `GliderDefines.h:501` |
| `kScoreboardTall` | 20 | `GliderDefines.h:515` |
| `kMaxViewWidth` | 1536 (= 3 × 512) | `GliderDefines.h:267` |
| `kMaxViewHeight` | 1026 | `GliderDefines.h:268` |
| `kFloorSupportTall` | 44 | `GliderDefines.h:500` |

So the play field is a 3×3 grid of 512×322 rooms, clipped to at most 1536 wide
and 1026 tall, with the current room centred on screen. `kMaxViewHeight` 1026 is
not 3 × 322 = 966; it is 966 + 60, so it accommodates a little more than three
rooms vertically. (This is one of the unexplained constants — see Open
questions.)

The nine destination rects, `localRoomsDest[9]` (`RoomGraphics.c:29`), built at
`InterfaceInit.c:206-218`:

```
1. (206-210) for i in 0 .. 8:
                 QSetRect(&localRoomsDest[i], 0, 0, kRoomWide, kTileHigh)
                 QOffsetRect(&localRoomsDest[i], playOriginH, playOriginV)
2. (211-218) then offset each by its neighbour displacement:
```

| Index | Constant | Value | Displacement (h, v) |
|---|---|---|---|
| 0 | `kCentralRoom` | 0 | (0, 0) |
| 1 | `kNorthRoom` | 1 | (0, −322) |
| 2 | `kNorthEastRoom` | 2 | (+512, −322) |
| 3 | `kEastRoom` | 3 | (+512, 0) |
| 4 | `kSouthEastRoom` | 4 | (+512, +322) |
| 5 | `kSouthRoom` | 5 | (0, +322) |
| 6 | `kSouthWestRoom` | 6 | (−512, +322) |
| 7 | `kWestRoom` | 7 | (−512, 0) |
| 8 | `kNorthWestRoom` | 8 | (−512, −322) |

(Indices from `GliderDefines.h:217-225`.)

`short numNeighbors` (`RoomGraphics.c:31`) selects how many of the nine are
actually drawn; it defaults to 9 (`Main.c:154`) but is forced to 1 on small
screens (`Main.c:191-192`):

```c
191  if ((numNeighbors > 1) && (thisMac.screen.right <= 512))
192      numNeighbors = 1;
```

So a 512-wide (or narrower) display gets single-room rendering, i.e. classic
Glider 4.0 behaviour.

### 9.5 Minimum and expected resolutions

Nothing enforces a minimum, but the design targets are legible:

| Screen | Consequence |
|---|---|
| 512×342 (Mac Plus / SE 9-inch) | `numNeighbors` forced to 1; `splashOriginH/V` clamp to 0 so the 640×460 splash is cropped; `houseRect` = 512×322 |
| 640×480 (13-inch) | `splashOriginH = 0`, `splashOriginV = 0` — the intended layout; `houseRect` = 640×460; `playOriginH = 64`, `playOriginV = 79` |
| 832×624 | `splashOriginH = 96`, `splashOriginV = 72`; `houseRect` = 832×604 |
| 1024×768 | `splashOriginH = 192`, `splashOriginV = 144`; `houseRect` = 1024×748 |
| > 1536 wide | `houseRect.right` clamped to 1536; the window is still full-screen but the offscreen buffers are only 1536 wide, so the right edge of the window is never drawn |

`Environ.c:26-28` names three display classes — `kDisplay9Inch` 1,
`kDisplay12Inch` 2, `kDisplay13Inch` 3 — but the constants are unreferenced in
the shipped code.

### 9.6 The scoreboard band

`AdjustScoreboardHeight()` is called first thing in `NewGame` (`Play.c:81`).
`NewGame` then blacks out a 20-pixel strip across the bottom of the screen
(`Play.c:142-145`) before drawing anything. `kScoreboardHigh` = 0 and
`kScoreboardLow` = 1 (`GliderDefines.h:513-514`) select which end the scoreboard
lives at; `kScoreboardPictID` = 1997 (`GliderDefines.h:623`) is the artwork,
verified to be **1536×20** ([Appendix A.3](#a3-pict-dimensions)) — exactly
`kMaxViewWidth` × `kScoreboardTall`.

The arcade build blanks and redraws it around the game-over sequence with a
computed horizontal offset (`Play.c:527-530`, repeated verbatim at `Play.c:578-581`):

```c
527  if (boardSrcRect.right >= 640)
528      hOffset = (RectWide(&boardSrcRect) - kMaxViewWidth) / 2;
529  else
530      hOffset = -576;
531  thePicture = GetPicture(kScoreboardPictID);   // then DrawPicture at bounds+hOffset
```

The magic `-576` is `(64 - 640)`: on a 512-wide screen it slides the 1536-wide
artwork so its centre section lands on screen.

---

## 10. The environment record (`macEnviron`)

### 10.1 The struct

`GliderPRO/Headers/Environ.h:11-32`:

```c
11  typedef struct
12  {
13      Rect		screen;
14      Rect		gray;
15      long		dirID;
16      short		wasDepth;
17      short		isDepth;
18      short		thisResFile;
19      short		numScreens;
20      short		vRefNum;
21      Boolean		can1Bit;
22      Boolean		can4Bit;
23      Boolean		can8Bit;
24      Boolean		wasColorOrGray;
25      Boolean		hasWNE;
26      Boolean		hasSystem7;
27      Boolean		hasColor;
28      Boolean		hasGestalt;
29      Boolean		canSwitch;
30      Boolean		canColor;
31      Boolean		hasSM3;
32      Boolean		hasQT;
33      Boolean		hasDrag;
34  } macEnviron;
```
with `extern macEnviron thisMac;` at `Environ.h:35` and the definition at
`GliderPRO/Sources/Environ.c:60`.

| Field | Type | Meaning | Set where |
|---|---|---|---|
| `screen` | `Rect` | main display bounds, global coords | `Environ.c:463` |
| `gray` | `Rect` | bounds of `GetGrayRgn()` (all displays minus menu bar) | `Environ.c:468-469` |
| `dirID` | `long` | app's directory ID | `Environ.c:445` — hardcoded `0` with a `// TEMP` comment |
| `wasDepth` | `short` | display depth at launch, to restore on quit | `Environ.c:465` |
| `isDepth` | `short` | current depth (8 or 4) | `Environ.c:542`, `Environ.c:397` |
| `thisResFile` | `short` | `CurResFile()` at startup | `Environ.c:443` |
| `numScreens` | `short` | count of screen devices | `Environ.c:462` |
| `vRefNum` | `short` | app's volume ref | `Environ.c:444` — hardcoded `0` (`// TEMP`) |
| `can1Bit`,`can4Bit`,`can8Bit` | `Boolean` | depth capability | `Environ.c:458-460` — all hardcoded `true` (no `// TEMP` comment, unlike the others) |
| `wasColorOrGray` | `Boolean` | was the display in colour (vs grayscale) at launch | `Environ.c:466` |
| `hasWNE` | `Boolean` | `WaitNextEvent` available | `Environ.c:447` — hardcoded `true` |
| `hasSystem7` | `Boolean` | System 7 or later | `Environ.c:450` — hardcoded `true` |
| `hasColor` | `Boolean` | Color QuickDraw | `Environ.c:448` — hardcoded `true` |
| `hasGestalt` | `Boolean` | `Gestalt` trap present | `Environ.c:446` — hardcoded `true` |
| `canSwitch` | `Boolean` | can change depth | `Environ.c:449` — hardcoded `true` |
| `canColor` | `Boolean` | *never assigned anywhere* | — |
| `hasSM3` | `Boolean` | Sound Manager 3 | `Environ.c:451` — hardcoded `true` |
| `hasQT` | `Boolean` | QuickTime | `Environ.c:452` (real Gestalt probe), possibly cleared at `Main.c:334` |
| `hasDrag` | `Boolean` | Drag Manager | `Environ.c:453` (real Gestalt probe) |

### 10.2 `CheckOurEnvirons`

`GliderPRO/Sources/Environ.c:438-470`:

```
 1. (443) thisMac.thisResFile = CurResFile()
 2. (444) thisMac.vRefNum    = 0        // TEMP
 3. (445) thisMac.dirID      = 0        // TEMP
 4. (446) thisMac.hasGestalt = true     // TEMP
 5. (447) thisMac.hasWNE     = true     // TEMP
 6. (448) thisMac.hasColor   = true     // TEMP
 7. (449) thisMac.canSwitch  = true     // TEMP
 8. (450) thisMac.hasSystem7 = true     // TEMP
 9. (451) thisMac.hasSM3     = true     // TEMP
10. (452) thisMac.hasQT      = DoWeHaveQuickTime()
11. (453) thisMac.hasDrag    = DoWeHaveDragManager()
12. (455) FindOurDevice()                          // thisGDevice = GetMainDevice()
13. (456-457) wasState = HGetState((Handle)thisGDevice); HLock(...)
14. (458) thisMac.can1Bit    = true     // (no TEMP comment)
15. (459) thisMac.can4Bit    = true
16. (460) thisMac.can8Bit    = true
17. (461) HSetState((Handle)thisGDevice, wasState)
18. (462) thisMac.numScreens = HowManyUsableScreens(false, true, true)
19. (463) GetDeviceRect(&thisMac.screen)
20. (465) thisMac.wasDepth   = WhatsOurDepth()
21. (466) thisMac.wasColorOrGray = AreWeColorOrGrayscale()
22. (468-469) grayRegion = GetGrayRgn(); GetRegionBounds(grayRegion, &(thisMac.gray))
```

**Eight of the 21 fields are hardcoded `true`/`0` with a `// TEMP` comment**
(`vRefNum`, `dirID`, `hasGestalt`, `hasWNE`, `hasColor`, `canSwitch`,
`hasSystem7`, `hasSM3`), and three more (`can1Bit`/`can4Bit`/`can8Bit`) are
hardcoded `true` without one — 11 hardcoded in total, plus `canColor`, which is
never assigned at all.
The by-the-book Gestalt/trap probing that a 1994 Mac program would do is present
in the file but not called. The only meaningful capability tests remaining are
QuickTime, Drag Manager, screen count, screen rect, depth and colour/gray.

A Go port should treat all nine as compile-time `true` and delete them.

### 10.3 The probe functions (for completeness)

| Function | Lines | Behaviour |
|---|---|---|
| `DoWeHaveQuickTime` | `Environ.c:196-209` | `Gestalt(gestaltQuickTime, &response)`; true iff `noErr` |
| `DoWeHaveDragManager` | `Environ.c:213-226` | `Gestalt(gestaltDragMgrAttr, &response)` then test bit `gestaltDragMgrPresent` |
| `WhatsOurDepth` | `Environ.c:232-253` | `thisDepth = (**(**thisGDevice).gdPMap).pixelSize;` bracketed by `HGetState`/`HLock`/`HSetState` on the pixmap handle; returns 1 if `!hasColor` |
| `CanWeDisplay8Bit` | `Environ.c:259-270` | `HasDepth(theDevice, 8, 1, 0) != 0` |
| `HowManyUsableScreens` | `Environ.c:308-328` | walks `GetDeviceList()`/`GetNextDevice()` counting devices where `TestDeviceAttribute(dev, screenDevice)`. **The depth filters are commented out**, so it counts every screen regardless of capability, and its three `Boolean` parameters are ignored. |
| `GetDeviceRect` | `Environ.c:334-342` | `*theRect = (*thisGDevice)->gdRect` |
| `AreWeColorOrGrayscale` | `Environ.c:348-368` | `(**thisGDevice).gdFlags & 0x0001` — the `gdDevType` bit; 1 = colour, 0 = monochrome/gray |
| `SwitchToDepth` | `Environ.c:374-398` | `SetDepth(thisGDevice, newDepth, 1, colorFlag)` then `thisMac.isDepth = newDepth` |
| `SwitchDepthOrAbort` | `Environ.c:404-432` | `Alert(kSwitchDepthAlert /* 130 */)`: 1 ⇒ `SwitchToDepth(8, true)`, 2 ⇒ `SwitchToDepth(4, false)`, 3 ⇒ `ExitToShell()` |

### 10.4 Depth switching policy

`HandleDepthSwitching()` (`Environ.c:505-543`) wraps the whole `switch` in
`if (thisMac.hasColor)` (`Environ.c:507`) and switches on the preference
`isDepthPref` (`Main.c:25`), whose values are `GliderDefines.h:43-45`:

| Constant | Value | Behaviour |
|---|---|---|
| `kSwitchIfNeeded` | 0 (default, `Main.c:146`) | `Environ.c:511-515`: if `(wasDepth != 8) && ((wasDepth != 4) \|\| wasColorOrGray)` then `SwitchDepthOrAbort()` — i.e. accept 8-bit colour, or 4-bit *grayscale*, and prompt otherwise |
| `kSwitchTo256Colors` | 1 | `Environ.c:517-525`: if `wasDepth != 8` then `can8Bit ? SwitchToDepth(8, true) : SwitchDepthOrAbort()`. `can8Bit` is hardcoded `true` (§10.2), so in practice this always forces 8-bit colour |
| `kSwitchTo16Grays` | 2 | `Environ.c:527-535`: if `wasDepth != 4 \|\| wasColorOrGray` then `can4Bit ? SwitchToDepth(4, false) : SwitchDepthOrAbort()` — in practice forces 4-bit grayscale |

and always ends with `thisMac.isDepth = WhatsOurDepth();` (`Environ.c:542`).

`RestoreColorDepth()` (`Environ.c:549-554`), called on quit (`Main.c:382`):

```c
551  if ((thisMac.hasColor) && ((thisMac.wasDepth != thisMac.isDepth) ||
552      (thisMac.wasColorOrGray != AreWeColorOrGrayscale())))
553      SwitchToDepth(thisMac.wasDepth, true);
```

There is also a runtime guard: if the depth changed while the app was in the
background, the resume handler notices (`Events.c:399`) and puts up
`BitchAboutColorDepth()` — `ALRT` 1042 (`kColorSwitchedAlert`,
`Events.c:46-55`) — offering "quit" or "switch back".

**For a Go port this whole section collapses to nothing**, with one exception:
the game genuinely renders in **two different depths**, 8-bit colour and 4-bit
grayscale, and there are depth-dependent branches in drawing code
(e.g. `MainWindow.c:76-79` picks white text at depth 4). A port that only
supports one depth should hardcode `thisMac.isDepth = 8` and take the 8-bit
branch everywhere.

---

## 11. Memory sizing (`CheckMemorySize`)

`GliderPRO/Sources/Environ.c:562-692`. Not needed by a Go port, but documented
because it is a complete inventory of every large allocation the game makes, and
because it *changes behaviour* (it can disable music and sound).

Constants (`Environ.c:564-565`):

```c
564  #define kBaseBytesNeeded		614400L		// 600 KB
565  #define kPaddingBytes			204800L		// 200 KB
```

Algorithm:

```
 1. (571-572) dontLoadMusic = false; dontLoadSounds = false
 2. (574)     bytesNeeded = kBaseBytesNeeded
 3. (575-579) soundBytes = SoundBytesNeeded(); if <= 0: RedAlert(kErrNoMemory)
              else bytesNeeded += soundBytes                 // Sound.c
 4. (580-584) musicBytes = MusicBytesNeeded(); if <= 0: RedAlert(kErrNoMemory)
              else bytesNeeded += musicBytes                 // Music.c
 5. (585)     += 4L * thisMac.screen.bottom                  // main-screen row table
 6. (586-593) += workSrcMap and backSrcMap pixel bytes, each followed by
              += 4L * houseRect.bottom                       // their row tables
 7. (594-596) += scoreboard map, then += (6396 * isDepth) / 8
 8. (597-598) += (32112 * isDepth) / 8, twice                // glider, glider2
 9. (599-641) += every remaining art GWorld as (literal * isDepth) / 8 and every
              1-bit mask as literal / 8 (a hand-written per-GWorld tally)
10. (642-656) += sizeof() of fifteen struct arrays (a subset of what CreatePointers
              allocates; note it does *not* charge for gameType/prefsInfo etc.)
11. (657)     += kDemoLength                                 // 6702
12. (659)     bytesAvail = FreeMem()
13. (661)     if bytesAvail < bytesNeeded:
14.   (663)     InitCursor()
15.   (664-668) if bytesAvail >= bytesNeeded - musicBytes:
                  TellHerNoMusic(); dontLoadMusic = true; return
16.   (670-675) else if bytesAvail >= bytesNeeded - (musicBytes + soundBytes):
                  TellHerNoSounds(); dontLoadMusic = true;
                  dontLoadSounds = true; return
17.   (680-687) else NumToString((bytesNeeded + kPaddingBytes) / 1024L, sizeStr);
                  ParamText(sizeStr, ...);
                  Alert(kLowMemoryAlert /* 181 */) under #ifdef COMPILEDEMO (682),
                  Alert(kSetMemoryAlert /* 180 */) otherwise (687) — the ship build
                  takes the 687 path
18.   (690)     ExitToShell()
```

Note the *degradation is not cumulative and does not adjust `bytesNeeded`*: the
three cases at 664 / 670 / 678 are a single if / else-if / else chain, each of the
first two arms `return`s immediately, and the comparisons are made against
`bytesNeeded - musicBytes` and `bytesNeeded - (musicBytes + soundBytes)` rather
than by subtracting from `bytesNeeded`. Also note the sounds arm sets **both**
`dontLoadMusic` and `dontLoadSounds` (673-674) — there is no configuration that
plays sounds but not music. `SpinCursor(1)` is called at 657 and 659.

`kSetMemoryAlert` = 180 and `kLowMemoryAlert` = 181 (`Environ.c:19-20`).
`Boolean dontLoadMusic` lives in `Music.c:36`, `dontLoadSounds` in `Sound.c:33`.

**The `<= 0` tests at steps 3-4 are missing-resource tests, not memory tests.**
`SoundBytesNeeded` (`Sound.c:491-512`) and `MusicBytesNeeded`
(`Music.c:386-407`) both do `SetResLoad(false)`, walk a fixed ID range calling
`GetResource('snd ', id)` + `GetMaxResourceSize`, and **`return (long)ResError()`
the moment one is `nil`** — a negative `OSErr` such as `resNotFound` (−192).
So `soundBytes <= 0` means "a `'snd '` resource is missing", and
`RedAlert(kErrNoMemory)` is a misleading choice of alert. The ranges are:

| Function | IDs walked | Count | Constants |
|---|---|---:|---|
| `SoundBytesNeeded` | `'snd '` 1000-1062 | 63 | `kBaseBufferSoundID` 1000, `kMaxSounds` 64, loop is `i < kMaxSounds - 1` (`Sound.c:499`) |
| `MusicBytesNeeded` | `'snd '` 2000-2006 | 7 | `kBaseBufferMusicID` 2000, `kMaxMusic` 7 (`Music.c:15-16`) |

63 + 7 = 70, which is exactly the number of `'snd '` resources in the fork
(§A.5) — so the `- 1` in `kMaxSounds - 1` is deliberate and the inventory is
complete. Because `SetResLoad(false)` is in effect neither function loads any
data; the commented-out `ReleaseResource` calls (`Sound.c:508`, `Music.c:403`)
leave the (dataless) master pointers in the resource map.

`SetAppMemorySize(long)` (`Environ.c:699-733`) rewrites the application's own
`'SIZE'` resource: it removes `'SIZE'` 0 and 1 and patches `'SIZE'` −1's
`mem1`/`mem2` fields, then `UpdateResFile` + `FlushVol`. It is **dead code** —
the only call site is commented out (`Environ.c:688`); it is merely declared in
`Externs.h:322`. The struct is (`Environ.c:31-36`):

```c
31  typedef struct
32  {
33      short	flags;
34      long	mem1;
35      long	mem2;
36  } sizeType;
```

Self-modifying resource forks have no Go analogue; delete.

Also relevant to memory: `Play.c:195` calls `freeBytes = MaxMem(&growBytes)`
during game set-up, before the frame loop starts, purely to compact and purge the
heap so that no allocation stalls occur mid-game — the result is stored and never
read. A Go port's equivalent — if any — is `runtime.GC()` before the loop plus a
frame loop that allocates as little as possible.

The frame loop is *nearly* allocation-free, but not entirely: `RenderFrame` calls
`DrawReflection` when `hasMirror` (`Render.c:641-646`), and `DrawReflection` does
a `NewRgn()` / `GetClip` / `SetClip` / `DisposeRgn` round trip every frame, per
glider (`Render.c:153`, `Render.c:183`) — and bails out of drawing the reflection
entirely if the `NewRgn` fails (`Render.c:154-155`). Outside the per-frame path,
`AddToMirrorRegion` (`NewRgn`, `Render.c:746-752`) and `BackUpToSavedMap`
(`NewGWorld`) allocate on room transitions.

---

## 12. Preferences

### 12.1 Version and file identity

| Constant | Value | Where |
|---|---|---|
| `kPrefsVersion` | `0x0034` (52) | `Main.c:16` |
| `kPrefCreatorType` | `'ozm5'` | `Prefs.c:18` |
| `kPrefFileType` | `'gliP'` | `Prefs.c:19` |
| `kPrefFileName` | `"Glider Prefs"` | `Prefs.c:20` |
| `kDefaultPrefFName` | `"Preferences"` | `Prefs.c:21` |
| `kPrefsStringsID` | 160 | `Prefs.c:22` |
| `kNewPrefsAlertID` | 160 | `Prefs.c:23` |
| `kPrefsFNameIndex` | 1 | `Prefs.c:24` |

`'ozm5'` is also the house-file creator (`SelectHouse.c` accepts
`fdCreator == 'ozm5'`, `fdType == 'gliH'`) — a nod to *Ozma of Oz*, and the
reason the resource type `'ozm5'` exists in the resource fork.

`LoadPrefs` (`Prefs.c:240-269`) returns false — and deletes the file after
showing `ALRT` 160 — if the read hits `eofErr` or if
`thePrefs->prefVersion != versionNow`. `SavePrefs` (`Prefs.c:148-162`) stamps
`thePrefs->prefVersion = versionNow` before writing. `WritePrefs`
(`Prefs.c:97-144`) writes `byteCount = sizeof(*thePrefs)` bytes — a raw struct
dump, so **the on-disk format is the C struct layout**, big-endian, `mac68k`
aligned.

`CanUseFindFolder` (`Prefs.c:39-55`) and `GetPrefsFPath` (`Prefs.c:59-69`) locate
the Preferences folder via `Gestalt(gestaltFindFolderAttr)` / `FindFolder`,
falling back to the System Folder.

### 12.2 `prefsInfo` — verified layout

Declared inside `#pragma options align=mac68k` … `align=reset`
(`GliderPRO/Headers/Externs.h:231-269`). Verified size **226 bytes**; offsets
observed by compiling the identical layout under `#pragma pack(2)` with 32-bit
`long`:

| Offset | Size | Field | Type | Maps to global | Default |
|---:|---:|---|---|---|---|
| 0 | 33 | `wasDefaultName` | `Str32` | `thisHouseName` | `"Slumberland"` (`Main.c:127`) |
| 33 | 16 | `wasLeftName` | `Str15` | left-key display name | `"lf arrow"` (`Main.c:129`) |
| 49 | 16 | `wasRightName` | `Str15` | right-key display name | `"rt arrow"` (`Main.c:130`) |
| 65 | 16 | `wasBattName` | `Str15` | battery-key display name | `"dn arrow"` (`Main.c:131`) |
| 81 | 16 | `wasBandName` | `Str15` | band-key display name | `"up arrow"` (`Main.c:132`) |
| 97 | 16 | `wasHighName` | `Str15` | `highName` | `"Your Name"` (`Main.c:133`) |
| 113 | 32 | `wasHighBanner` | `Str31` | `highBanner` | `"Your Message Here"` (`Main.c:134`) |
| *145* | *1* | *(pad)* | — | — | inserted because `Str31` ends on an odd offset and the next field is a `long` |
| 146 | 4 | `wasLeftMap` | `long` | `theGlider.leftKey` | `kLeftArrowKeyMap` = 124 (`Main.c:135`) |
| 150 | 4 | `wasRightMap` | `long` | `theGlider.rightKey` | `kRightArrowKeyMap` = 123 (`Main.c:136`) |
| 154 | 4 | `wasBattMap` | `long` | `theGlider.battKey` | `kDownArrowKeyMap` = 122 (`Main.c:137`) |
| 158 | 4 | `wasBandMap` | `long` | `theGlider.bandKey` | `kUpArrowKeyMap` = 121 (`Main.c:138`) |
| 162 | 2 | `wasVolume` | `short` | `isVolume` | on the *defaults* path only: read from the system and clamped to [1, 3] (`Main.c:140-144`). The load path copies it verbatim with **no** clamp (`Main.c:78`) |
| 164 | 2 | `prefVersion` | `short` | — | `0x0034` |
| 166 | 2 | `wasMaxFiles` | `short` | `maxFiles` | 48, clamped to [12, 500] else 12 (`Main.c:88-89`, `156`) |
| 168 | 2 | `wasEditH` | `short` | `isEditH` | 3 (`Main.c:158`) |
| 170 | 2 | `wasEditV` | `short` | `isEditV` | 41 (`Main.c:159`) |
| 172 | 2 | `wasMapH` | `short` | `isMapH` | 3 (`Main.c:160`) |
| 174 | 2 | `wasMapV` | `short` | `isMapV` | 100 (`Main.c:161`) |
| 176 | 2 | `wasMapWide` | `short` | `mapRoomsWide` | 15 (`Main.c:163`) |
| 178 | 2 | `wasMapHigh` | `short` | `mapRoomsHigh` | 4 (`Main.c:164`) |
| 180 | 2 | `wasToolsH` | `short` | `isToolsH` | 100 (`Main.c:166`) |
| 182 | 2 | `wasToolsV` | `short` | `isToolsV` | 35 (`Main.c:167`) |
| 184 | 2 | `wasLinkH` | `short` | `isLinkH` | 50 (`Main.c:168`) |
| 186 | 2 | `wasLinkV` | `short` | `isLinkV` | 80 (`Main.c:169`) |
| 188 | 2 | `wasCoordH` | `short` | `isCoordH` | 50 (`Main.c:171`) |
| 190 | 2 | `wasCoordV` | `short` | `isCoordV` | 204 (`Main.c:172`) |
| 192 | 2 | `isMapLeft` | `short` | `mapLeftRoom` | 60 (`Main.c:173`) |
| 194 | 2 | `isMapTop` | `short` | `mapTopRoom` | 50 (`Main.c:174`) |
| 196 | 2 | `wasNumNeighbors` | `short` | `numNeighbors` | 9 (`Main.c:154`), forced to 1 if screen ≤ 512 wide (`Main.c:191-192`) |
| 198 | 2 | `wasDepthPref` | `short` | `isDepthPref` | `kSwitchIfNeeded` = 0 (`Main.c:146`) |
| 200 | 2 | `wasToolGroup` | `short` | `toolMode` | `kBlowerMode` = 1 (`Main.c:182`) |
| 202 | 2 | `smWarnings` | `short` | `numSMWarnings` | 0 (`Main.c:177`) |
| 204 | 2 | `wasFloor` | `short` | `wasFloor` | 0 (`Main.c:175`) |
| 206 | 2 | `wasSuite` | `short` | `wasSuite` | 0 (`Main.c:176`) |
| 208 | 1 | `wasZooms` | `Boolean` | `doZooms` | true (`Main.c:151`) |
| 209 | 1 | `wasMusicOn` | `Boolean` | `isMusicOn` | true (`Main.c:148`) |
| 210 | 1 | `wasAutoEdit` | `Boolean` | `autoRoomEdit` | true (`Main.c:178`) |
| 211 | 1 | `wasDoColorFade` | `Boolean` | `isDoColorFade` | true (`Main.c:155`) |
| 212 | 1 | `wasMapOpen` | `Boolean` | `isMapOpen` | true (`Main.c:179`) |
| 213 | 1 | `wasToolsOpen` | `Boolean` | `isToolsOpen` | true (`Main.c:180`) |
| 214 | 1 | `wasCoordOpen` | `Boolean` | `isCoordOpen` | false (`Main.c:181`) |
| 215 | 1 | `wasQuickTrans` | `Boolean` | `quickerTransitions` | false (`Main.c:153`) |
| 216 | 1 | `wasIdleMusic` | `Boolean` | `isPlayMusicIdle` | true (`Main.c:149`) |
| 217 | 1 | `wasGameMusic` | `Boolean` | `isPlayMusicGame` | true (`Main.c:150`) |
| 218 | 1 | `wasEscPauseKey` | `Boolean` | `isEscPauseKey` | false (`Main.c:184`) |
| 219 | 1 | `wasDoAutoDemo` | `Boolean` | `doAutoDemo` | **true** (`Main.c:183`) |
| 220 | 1 | `wasScreen2` | `Boolean` | `isUseSecondScreen` | false (`Main.c:185`) |
| 221 | 1 | `wasDoBackground` | `Boolean` | `doBackground` | **false** (`Main.c:186`) |
| 222 | 1 | `wasHouseChecks` | `Boolean` | `isHouseChecks` | true (`Main.c:152`) |
| 223 | 1 | `wasPrettyMap` | `Boolean` | `doPrettyMap` | false (`Main.c:187`) |
| 224 | 1 | `wasBitchDialogs` | `Boolean` | `doBitchDialogs` | true (`Main.c:188`) |
| *225* | *1* | *(tail pad)* | — | — | struct rounded up to an even size |

Total: **226 bytes**. Two padding bytes, at offsets 145 and 225. A Go port
reading real preference files must reproduce both.

The commented-out line at `Externs.h:240` (`// long encrypted, fakeLong;`) is
the copy-protection remnant; it is *not* in the layout, so a shipped prefs file
from this build has no such fields. This is self-consistent: the only code that
would touch those members (`Main.c:75` on read, `Main.c:231-232` on write) sits
inside `#ifndef COMPILENOCP`, and `COMPILENOCP` is defined in the ship build
(§1), so it is compiled out. Had the struct members been present the file would
have been 234 bytes with the two longs at offsets 146 and 150.

### 12.3 `ReadInPrefs` / `WriteOutPrefs`

`ReadInPrefs()` (`Main.c:52-201`) does: `LoadPrefs(&thePrefs, kPrefsVersion)`;
on success scatter each field into its global; on failure fill every global with
the default listed above (`Main.c:122-189`). Then, unconditionally:

```c
191  if ((numNeighbors > 1) && (thisMac.screen.right <= 512))
192      numNeighbors = 1;
193
194  UnivGetSoundVolume(&wasVolume, thisMac.hasSM3);
195  UnivSetSoundVolume(isVolume, thisMac.hasSM3);
196
197  if (isVolume == 0)
198      isSoundOn = false;
199  else
200      isSoundOn = true;
```

`short isVolume, wasVolume;` are at `Main.c:24`. The system volume at launch is
saved into `wasVolume` so it can be restored at quit; the game's own volume
preference `isVolume` is then pushed to the *system* volume unconditionally —
including when it is 0. There is **no** "fall back to the system volume if the
pref is 0" behaviour; instead a zero pref simply sets `isSoundOn = false`
(197-200), which is the flag the rest of the program tests.

`WriteOutPrefs()` (`Main.c:208-279`) is the inverse, with two things at its edges
that are not simple field copies:

```c
212  UnivGetSoundVolume(&isVolume, thisMac.hasSM3);   // capture current volume
     ... scatter every global into thePrefs (213-273) ...
275  if (!SavePrefs(&thePrefs, kPrefsVersion))
276      SysBeep(1);                                  // the only failure report
278  UnivSetSoundVolume(wasVolume, thisMac.hasSM3);   // restore launch volume
```

Because `ExitToShell()` paths (`RedAlert`, `SwitchDepthOrAbort`,
`CheckMemorySize`) bypass `WriteOutPrefs`, a crash or fatal error loses
preference changes made in that session — including the system volume
restoration.

---

## 13. Apple Events

`GliderPRO/Sources/AppleEvents.c`. `SetUpAppleEvents()` (`AppleEvents.c:175-207`)
installs four `kCoreEventClass` handlers and then calls
`AESetInteractionAllowed(kAEInteractWithAll)` (`AppleEvents.c:204`).

| Event | Handler | Lines | Behaviour |
|---|---|---|---|
| `kAEOpenApplication` | `DoOpenAppAE` | `37-44` | only `MyGotRequiredParams(theAppleEvent)` — nothing else |
| `kAEOpenDocuments` | `DoOpenDocAE` | `49-127` | see below |
| `kAEPrintDocuments` | `DoPrintDocAE` | `132-142` | `Alert(kNoPrintingAlert /* 1031 */)`, returns `errAEEventNotHandled` |
| `kAEQuitApplication` | `DoQuitAE` | `147-157` | `quitting = true` — **no unsaved-changes prompt** |

The UPP globals are at `AppleEvents.c:24`: `openAppAEUPP`, `openDocAEUPP`,
`printDocAEUPP`, `quitAEUPP`.

`DoOpenDocAE`:

```
 1. (62)      AEGetParamDesc(theAE, keyDirectObject, typeAEList, &docList)
 2. (69)      MyGotRequiredParams(theAE)          // each of 1/2/3 bails on error,
 3. (76)      AECountItems(&docList, &itemsInList)  //   disposing docList first
             // everything from here to step 15 is inside #ifndef COMPILEDEMO (83-123)
 4. (84-94)   for i = 1..itemsInList: AEGetNthPtr(i) + FSpGetFInfo;
                 if fdType == 'gliH': AddExtraHouse(&oneFSS)
             // then, separately, item 1 is re-fetched and *opened*:
 5. (95)      if (itemsInList > 0):
 6.   (97-98)   AEGetNthPtr(&docList, 1, typeFSS, ...) -> oneFSS
 7.   (101-102) if FSpGetFInfo ok and fdType == 'gliH':
 8.     (104)     CloseHouse()                    // return value discarded
 9.     (105)     PasStringCopy(oneFSS.name, thisHouseName)
10.     (106)     BuildHouseList()
11.     (107-108) if OpenHouse(): ReadHouse()
12.     (109)     PasStringCopy(theHousesSpecs[thisHouseIndex].name, thisHouseName)
13.     (110)     OpenCloseEditWindows()
14.     (111)     incrementModeTime = TickCount() + kIdleSplashTicks
15.     (112-118) if theMode is kSplashMode or kPlayMode:
16.       (116)     SetRect(&updateRect, splashOriginH + 474, splashOriginV + 304,
                                        splashOriginH + 474 + 166, splashOriginV + 304 + 12)
17.       (117)     InvalWindowRect(mainWindow, &updateRect)
18.   (121)     InitCursor()                      // inside the itemsInList > 0 arm
19. (124)     AEDisposeDesc(&docList)
```

Note the two-pass structure: **every** dropped `'gliH'` file is registered as an
extra house (step 4), but only the **first** item in the list is actually opened
(steps 5-18), and if that first item is not a `'gliH'` file nothing is opened at
all even when a later item is one.

The 166×12 rect at splash-relative (474, 304) is the house-name strip on the
splash screen — the same rect `DoGameMenu(iLoadHouse)` invalidates
(`Menu.c:337-343`). Note `DrawOnSplash` draws its text baseline at
splash-relative (436, 314) (`MainWindow.c:67`), so the invalidation rect is
positioned to cover the *string*, not the whole line.

`AddExtraHouse` (`SelectHouse.c:668-675`) appends to
`extraHouseSpecs[kMaxExtraHouses]` (`SelectHouse.c:48`) with `kMaxExtraHouses` =
8 (`SelectHouse.c:35`) and increments `numExtraHouses` (`SelectHouse.c:52`).

`MyGotRequiredParams` (`AppleEvents.c:162-170`) is the standard
"were there any unexpected required parameters" check.

For a Go port: `kAEOpenDocuments` is "the OS handed us a file to open" and
`kAEQuitApplication` is "the OS wants us to quit". Both matter. The others do
not.

---

## 14. Structural consequences for a Go port

### 14.1 The loop is not re-entrant, and the call stack is the state machine

The full nesting when a game is running:

```
main                                    (Main.c:284)
 └─ while (!quitting) HandleEvent       (Main.c:363-364)
     └─ HandleKeyEvent / HandleMouseEvent   (Events.c:162 / 60)
         └─ DoMenuChoice                    (Menu.c:591)
             └─ DoGameMenu                  (Menu.c:301)
                 └─ NewGame                 (Play.c:74)     theMode := kPlayMode
                     └─ PlayGame            (Play.c:430)
                         ├─ HandlePlayEvent (Play.c:387)    [only if doBackground]
                         ├─ GetInput / GetDemoInput (Input.c:281 / 186)
                         │   └─ DoPause     (Input.c:77)    [blocks here]
                         │       └─ DoCommandKey (Input.c:53)
                         ├─ RenderFrame     (Render.c:639)  [busy-waits here]
                         └─ DoGameOver / DoDiedGameOver (GameOver.c:60 / 443)
                             └─ own 2-tick animation loops
```

There are therefore **four different blocking loops** in the program
(`HandleEvent`'s `WaitNextEvent`, `PlayGame`, `DoPause`, and each game-over
animation), and the "current state" is partly encoded in *which one you are
inside*. `theMode` is a shadow of that, maintained by hand.

A Go port has two reasonable shapes:

1. **Faithful**: keep the nesting. `PlayGame` is a function that blocks; the
   outer loop calls it. Simplest to diff against the C, and correct by
   construction. The cost is that the window system's event loop must tolerate
   being starved (most Go game libraries own the loop and call *you*, which
   makes this shape awkward).
2. **Inverted**: one top-level `Update()` called at 30 Hz by the framework, with
   an explicit state enum that includes the sub-states the C encodes as call
   depth: `Splash`, `Edit`, `Playing`, `Paused`, `GameOverDied`,
   `GameOverWon`, `PostGameWait`. This fits Ebiten/raylib-go naturally but you
   must hand-translate `DoPause`, `DoDiedGameOver`, `DoGameOverStarAnimation`
   and `WaitForInputEvent` from blocking loops into state machines.

If you choose (2), the blocking loops you must convert are exactly:

| Blocking loop | Site | Converts to |
|---|---|---|
| `while (TickCount() < nextFrame) {}` | `Render.c:662-664` | the framework's frame tick |
| `while (paused) { GetKeys… }` | `Input.c:97-105` | a `Paused` state |
| `do { HandlePlayEvent(); } while (switchedOut)` | `Play.c:439-443` | a `Backgrounded` state (or just skip the update) |
| `while (pagesStuck < 8)` + `nextLoop` | `GameOver.c:457-479` | a `GameOverDied` state with a frame counter |
| `while (noInteruption)` in `DoGameOverStarAnimation` | `GameOver.c:154-229` (`nextLoop` seeded at `:149`, re-armed at `:220`) | a `GameOverWon` state |
| `WaitForInputEvent(n)` | `Utilities.c:439-479` | a timed `PostGameWait` state |
| `WaitCommandQReleased()` | `Utilities.c:485-499` | an "ignore input until released" latch |
| the two release-waits in `DoPause` | `Input.c:89-94`, `111-116` | the same latch |

### 14.2 What must remain polled

In-game input is **entirely** `GetKeys`-polled (`Input.c:190`, `Input.c:285`),
sampled once per frame. Menu/editor input is event-driven. If a port routes
gameplay through key-down/key-up events it will change behaviour:
`heldLeft`/`heldRight`/`tipped` are recomputed from scratch every frame
(`Input.c:220-222`), so a key held across a frame boundary must read as held in
*both* frames — which an edge-triggered implementation gets wrong on the first
frame after a repeat gap.

### 14.3 What must remain non-allocating

`RenderFrame` and everything it calls should not allocate — with the one
exception noted in §11, `DrawReflection`'s per-frame `NewRgn`/`DisposeRgn`. The dirty-rect lists
are fixed 48-entry arrays (`Render.c:35-36`) and silently drop the 48th entry
(`Render.c:67`, `Render.c:86`, `Render.c:105`: the bound is `< kMaxGarbageRects - 1`, so the
usable capacity is **47**). A port that grows a slice instead will not be
bug-compatible in the pathological case; whether that matters is a judgement
call, but the 47 limit is observable (with more than 47 dirty regions, some
sprites visibly fail to erase).

---

## 15. Debug utilities

`GliderPRO/Sources/DebugUtilities.c` (355 lines) contains one global,
`short barGraphHori = 0;` (`DebugUtilities.c:14`), and 14 functions:

| Function | Line | Purpose |
|---|---|---|
| `MonitorWait` | 20 | spin until a key/click, for stepping frame by frame |
| `DisplayRect` | 40 | frame a rect on screen |
| `FlashRect` | 58 | invert a rect briefly |
| `CheckLegitRect` | 79 | assert a rect is inside another (the `CAREFULDEBUG` hook) |
| `DisplayLong` | 93 | draw a `long` on screen |
| `DisplayShort` | 125 | draw a `short` on screen |
| `FlashLong` | 157 | draw a `long`, then erase it |
| `FlashShort` | 181 | ditto for `short` |
| `DoBarGraph` | 205 | draw a profiling bar, advancing `barGraphHori` |
| `BetaOkay` | 261 | expiry check for beta builds |
| `DebugNum` | 281 | numeric breadcrumb |
| `DisplayCTSeed` | 290 | show a colour-table seed (palette debugging) |
| `FillScreenRed` | 300 | flood the screen with colour index for red |
| `DumpToResEditFile` | 316 | write an arbitrary buffer to a file as a resource — **this is how the `'demo'` resource was produced** |

None of it is on the live path. `CAREFULDEBUG` is defined nowhere (only the
commented-out line `GliderDefines.h:13`), and `DumpToResEditFile` is only called
under `CREATEDEMODATA`, which is also off. **A Go port should not port this
file**, with one exception: if you want to *re-record* a demo, you need the
`DumpToResEditFile` equivalent plus `LogDemoKey` (`Input.c:44-49`) and the
`CREATEDEMODATA` branches in `GetInput` (`Input.c:304`, `321`, `334`, `348`).

---

## 16. Global state inventory

Glider PRO has no context object; every piece of state is a file-scope global.
This section first isolates the globals that are *genuinely* application/game
state (§16.1–§16.5), then gives the complete per-file inventory (§16.6) so a
porter can find any name.

### 16.1 Application mode and lifecycle

| Name | Type | Defined | Meaning |
|---|---|---|---|
| `theMode` | `short` | `MainWindow.c:42` | **The** application state: `kSplashMode` 0 / `kEditMode` 1 / `kPlayMode` 2 |
| `quitting` | `Boolean` | `Main.c:26` | main-loop exit flag; set by Game ▸ Quit (`Menu.c:348`) or `kAEQuitApplication` (`AppleEvents.c:152`) |
| `playing` | `Boolean` | `Play.c:53` | `PlayGame` loop predicate |
| `paused` | `Boolean` | `Input.c:34` | `DoPause` loop predicate |
| `demoGoing` | `Boolean` | `Play.c:53` | attract mode is running |
| `twoPlayerGame` | `Boolean` | `Play.c:53` | second glider active |
| `gameOver` | `Boolean` | `GameOver.c:46` | game-over countdown has begun |
| `countDown` | `short` | `GameOver.c:45` | frames remaining before the game-over animation (init 16) |
| `switchedOut` | `Boolean` | `Events.c:31` | app is in the background |
| `menusUp` | `Boolean` | `Menu.c:33` | the menu bar has been built (gates `UpdateMenus`) |
| `splashDrawn` | `Boolean` | `MainWindow.c:43` | the splash has been painted at least once (gates Load House) |
| `resumedSavedGame` | `Boolean` | `Menu.c:33` | this session came from Open Saved Game |
| `idleMode` | `short` | `Events.c:30` | **vestigial** (§4.5); only ever `kIdleSplashMode` |
| `doAutoDemo` | `Boolean` | `Events.c:31` | attract mode enabled (pref, default true) |
| `doBackground` | `Boolean` | `Play.c:54` | pump events during play (pref, default **false**) |
| `isEscPauseKey` | `Boolean` | `Input.c:34` | Esc instead of Tab pauses (pref, default false) |

### 16.2 Timing

| Name | Type | Defined | Meaning |
|---|---|---|---|
| `gameFrame` | `long` | `Play.c:51` | frames elapsed in the current game; reset to 0 in `NewGame` (`Play.c:112`), incremented at `Play.c:434`. The demo stream is indexed by this. |
| `nextFrame` | `long` | `Render.c:40` | `TickCount()` deadline for the current frame; see §7.3 |
| `evenFrame` | `Boolean` | `Play.c:53` | toggles every frame; selects flames vs stars (`Render.c:649`) |
| `incrementModeTime` | `long` | `Events.c:27` | `TickCount()` deadline after which the attract demo launches |
| `doubleTime` | `UInt32` | `Events.c:28` | double-click threshold in ticks, from `GetDblTime()` |
| `lastUp` | `long` | `Events.c:27` | `when` of the previous mouse-up, for double-click detection |
| `lastWhere` | `Point` | `Events.c:29` | location of the previous click |
| `lastWhenClick` | `long` | `SelectHouse.c:49` | same, but private to the Load House dialog |
| `lastWhereClick` | `Point` | `SelectHouse.c:50` | ditto |
| `theSeed` | `UInt32` | `Utilities.c:19` | LCG state for `RandomLongQUS` (`theSeed = theSeed * 1103515245 + 12345`, `Utilities.c:126`) |
| `batteryFrame` | `short` | `Input.c:33` | throttles the thrust/hiss sound to every 4th frame |
| `thePhone`, `theChimes` | `phoneType` | `Play.c:47` | telephone-ring and clock-chime schedulers |
| `clockFrame` | `short` | `DynamicMaps.c:33` | animation phase for clock objects |

### 16.3 Game session state

| Name | Type | Defined | Meaning |
|---|---|---|---|
| `theGlider` | `gliderType` | `Player.c:43` | player 1 (110 bytes; see [A.2](#a2-struct-sizes-and-offsets)) |
| `theGlider2` | `gliderType` | `Player.c:43` | player 2 |
| `theScore` | `long` | `Player.c:50` | current score |
| `mortals` | `short` | `Play.c:52` | gliders remaining; **`< 0` means death game-over** (`Play.c:546`) |
| `batteryTotal` | `short` | `Play.c:52` | battery charge; negative values encode helium |
| `bandsTotal` | `short` | `Play.c:52` | rubber bands held |
| `foilTotal` | `short` | `Play.c:52` | aluminium-foil charge |
| `showFoil` | `Boolean` | `Play.c:53` | is the glider wearing foil |
| `playerSuicide` | `Boolean` | `Play.c:54` | player pressed the self-destruct (Delete) key |
| `playerDead` | `Boolean` | `Player.c:53` | glider is dead this frame |
| `onePlayerLeft` | `Boolean` | `Player.c:53` | two-player game reduced to one |
| `otherPlayerEscaped` | `short` | `Interactions.c:42` | escape code (`kNoOneEscaped` −1 … `kPlayerIsDeadForever` −69) |
| `activeRectEscaped` | `short` | `Interactions.c:42` | which hot rect the escape happened through |
| `saidFollow` | `short` | `Modes.c:13` | "follow me" prompt suppression counter |
| `numStarsRemaining` | `short` | `Banner.c:28` | stars left in the house |
| `bannerStarCountOn` | `Boolean` | `Banner.c:29` | show the star count in the banner |
| `displayedScore` | `long` | `Scoreboard.c:40` | score currently drawn (rolls toward `theScore`) |
| `doRollScore` | `Boolean` | `Scoreboard.c:42` | animate the score roll |
| `wasScoreboardMode` | `short` | `Scoreboard.c:41` | last `RefreshScoreboard` mode |
| `demoIndex` | `short` | `Input.c:33` | cursor into `demoData`; reset in `NewGame` (`Play.c:114`) |
| `demoData` | `demoPtr` | `Input.c:30` | the 1117-record recorded input stream |
| `theKeys` | `KeyMap` | `Input.c:31` | last `GetKeys` snapshot (128-bit key bitmap) |
| `tvOn` | `Boolean` | `Play.c:54` | the in-room TV is playing |
| `phoneBitSet` | `Boolean` | `Play.c:54` | this house has a telephone |
| `numBands` | `short` | `RubberBands.c:26` | live rubber bands (max 2) |
| `bandHitLast` | `short` | `RubberBands.c:26` | last band collision index |
| `numDynamics` | `short` | `Dynamics3.c:19` | live dynamic objects (max 18) |
| `numSparkles`, `numFlyingPts`, `numChimes` | `short` | `DynamicMaps.c:31` | live effect counts |
| `numFlames`, `numSavedMaps`, `numTikiFlames`, `numCoals` | `short` | `DynamicMaps.c:32` | ditto |
| `numPendulums`, `numStars`, `numShredded` | `short` | `DynamicMaps.c:33` | ditto |
| `numGrease` | `short` | `Grease.c:27` | live grease spills (max 16) |
| `numTempManholes` | `short` | `Objects.c:77` | temporary manhole rects this room |
| `smallGame` | `gameType` | `SavedGames.c:20` | the 40-byte saved-game record being read/written |

### 16.4 World / room state

| Name | Type | Defined | Meaning |
|---|---|---|---|
| `thisHouse` | `houseHand` | `House.c:25` | the whole loaded house, as a Handle |
| `thisHouseName` | `Str32` | `House.c:27` | its filename |
| `thisRoom` | `roomPtr` | `Room.c:24` | working copy of the current room (348 bytes) |
| `thisRoomNumber` | `short` | `Room.c:27` | index into `thisHouse`'s room array |
| `previousRoom` | `short` | `Room.c:27` | last room, or −1 |
| `numberRooms` | `short` | `Room.c:27` | rooms in the house |
| `localNumbers[9]` | `short` | `RoomGraphics.c:32` | room numbers of the 3×3 neighbourhood |
| `isStructure[9]` | `Boolean` | `RoomGraphics.c:33` | which neighbours are "structures" (outdoor) |
| `localRoomsDest[9]` | `Rect` | `RoomGraphics.c:29` | screen rects of the 3×3 neighbourhood |
| `numNeighbors` | `short` | `RoomGraphics.c:31` | how many of the 9 to draw (1 or 9) |
| `thisTiles[8]` | `short` | `RoomGraphics.c:31` | background tile indices for the current room |
| `thisBackground` | `short` | `RoomGraphics.c:32` | background PICT id |
| `lastBackground` | `short` | `Room.c:28` | previous background, for transition logic |
| `numLights` | `short` | `RoomGraphics.c:31` | lights on in this room (0 ⇒ dark) |
| `wardBitSet` | `Boolean` | `RoomGraphics.c:33` | house flags bit 0 |
| `leftOpen`, `rightOpen`, `topOpen`, `bottomOpen` | `Boolean` | `Room.c:30` | room openings, from `DetermineRoomOpenings` |
| `leftThresh`, `rightThresh` | `short` | `Room.c:28` | horizontal room-exit thresholds |
| `noRoomAtAll` | `Boolean` | `Room.c:29` | house has zero rooms |
| `newRoomNow` | `Boolean` | `Room.c:29` | a room was just created (editor auto-info trigger) |
| `masterObjects` | `objDataPtr` | `Objects.c:74` | flattened object list for the 3×3 neighbourhood (max 216) |
| `numMasterObjects`, `numLocalMasterObjects`, `nLocalObj` | `short` | `Objects.c:76` | counts into it |
| `hotSpots` | `hotPtr` | `Objects.c:75` | active interaction rects (max 56) |
| `nHotSpots` | `short` | `Objects.c:76` | count |
| `savedMaps[24]` | `savedType` | `Objects.c:73` | saved background patches under dynamic objects |
| `linksList` | `linksPtr` | `House.c:26` | object link table |
| `houseUnlocked` | `Boolean` | `House.c:32` | editing permitted |
| `houseOpen`, `fileDirty`, `gameDirty` | `Boolean` | `HouseIO.c:34` | file state |
| `houseIsReadOnly`, `saveHouseLocked`, `changeLockStateOfHouse` | `Boolean` | `HouseIO.c:35` | lock state |
| `houseRefNum`, `houseResFork`, `wasHouseVersion` | `short` | `HouseIO.c:33` | open file refs and the version read |
| `hasMovie`, `tvInRoom` | `Boolean` | `HouseIO.c:36` | QuickTime presence |
| `theMovie` | `Movie` | `HouseIO.c:31` | the loaded movie |
| `movieRect` | `Rect` | `HouseIO.c:32` | its bounds |
| `tvWithMovieNumber` | `short` | `Objects.c:77` | which TV object shows the movie |
| `wasFloor`, `wasSuite` | `short` | `House.c:30` | last edited floor/suite (also a pref) |
| `houseErrors`, `wasRoom` | `short` | `HouseLegal.c:29` | validation results |

### 16.5 Rendering / environment state

| Name | Type | Defined | Meaning |
|---|---|---|---|
| `thisMac` | `macEnviron` | `Environ.c:60` | the environment record (§10) |
| `thisGDevice` | `GDHandle` | `Utilities.c:18` | main graphics device |
| `mainWindow` | `WindowPtr` | `MainWindow.c:38` | the play/splash/edit window |
| `menuWindow` | `WindowPtr` | `MainWindow.c:38` | the 20-px menu-bar cover |
| `mainWindowRect` | `Rect` | `MainWindow.c:37` | its local bounds |
| `workSrcMap` / `workSrcRect` | `GWorldPtr` / `Rect` | `MainWindow.c:36`, `35` | composite frame buffer |
| `backSrcMap` / `backSrcRect` | `GWorldPtr` / `Rect` | `Room.c:26`, `25` | static room artwork |
| `houseRect` | `Rect` | `RoomGraphics.c:30` | screen minus scoreboard, clamped to 1536×1026 |
| `justRoomsRect` | `Rect` | `Play.c:48` | `houseRect` at origin — the in-game blit rect |
| `playOriginH`, `playOriginV` | `short` | `MainWindow.c:40` | top-left of the centred current room |
| `splashOriginH`, `splashOriginV` | `short` | `MainWindow.c:41` | top-left of the centred 640×460 splash |
| `isEditH`, `isEditV` | `short` | `MainWindow.c:39` | editor window position (pref) |
| `work2MainRects[48]`, `numWork2Main` | `Rect[]`, `short` | `Render.c:35`, `41` | present list |
| `back2WorkRects[48]`, `numBack2Work` | `Rect[]`, `short` | `Render.c:36`, `41` | erase list |
| `mirrorRgn`, `hasMirror` | `RgnHandle`, `Boolean` | `Render.c:38`, `42` | accumulated mirror region |
| `shieldRect`, `shieldPt` | `Rect`, `Point` | `Render.c:37`, `39` | cursor-shield bookkeeping |
| `fadeGraysOut`, `isDoColorFade` | `Boolean` | `MainWindow.c:43` | palette-fade flags |
| `theCTab`, `thePMap`, `wasColors`, `newColors` | palette handles | `MainWindow.c:27-30` | colour-table manipulation (mostly commented out) |
| `handCursorH`…`diagCursor` | `CursHandle`/`Cursor` | `MainWindow.c:31-34` | editor cursors |
| `animCursorH`, `useColorCursor` | `acurHandle`, `Boolean` | `AnimCursor.c:35-36` | spinning-cursor state |
| `gliderSrc[31]`, `shadowSrc[2]`, `fadeInSequence[16]` | `Rect[]`/`short[]` | `Player.c:48`, `47`, `51` | glider animation tables |
| `srcRects` | `Rect *` | `Objects.c:71` | 144 object source rects |
| `dataResFile` | `short` | `Main.c:25` | resource file ref for the data fork |

### 16.6 Complete per-file global inventory

Every file-scope definition in the 67 `.c` files. (`static` = file-private.)

| File | Line | Definition |
|---|---|---|
| `About.c` | 23-25 | `static RgnHandle okayButtRgn;` `static Rect okayButtonBounds, mainPICTBounds;` `static Boolean okayButtIsHiLit, clickedDownInOkay;` |
| `AnimCursor.c` | 35-36 | `acurHandle animCursorH = nil;` `Boolean useColorCursor = false;` |
| `AppleEvents.c` | 24 | `AEEventHandlerUPP openAppAEUPP, openDocAEUPP, printDocAEUPP, quitAEUPP;` |
| `Banner.c` | 28-29 | `short numStarsRemaining;` `Boolean bannerStarCountOn;` |
| `ColorUtils.c` | — | none |
| `Coordinates.c` | 16-20 | `Rect coordWindowRect;` `WindowPtr coordWindow;` `short isCoordH, isCoordV;` `short coordH, coordV, coordD;` `Boolean isCoordOpen;` |
| `DebugUtilities.c` | 14 | `short barGraphHori = 0;` |
| `DialogUtils.c` | — | none |
| `DynamicMaps.c` | 24-33 | `sparklePtr sparkles;` `flyingPtPtr flyingPoints;` `flamePtr flames, tikiFlames, bbqCoals;` `pendulumPtr pendulums;` `starPtr theStars;` `shredPtr shreds;` `Rect pointsSrc[15];` `short numSparkles, numFlyingPts, numChimes;` `short numFlames, numSavedMaps, numTikiFlames, numCoals;` `short numPendulums, clockFrame, numStars, numShredded;` |
| `Dynamics.c` | 20 | `Rect breadSrc[kNumBreadPicts];` |
| `Dynamics2.c` | — | none |
| `Dynamics3.c` | 18-19 | `dynaPtr dinahs;` `short numDynamics;` |
| `Environ.c` | 60 | `macEnviron thisMac;` |
| `Events.c` | 27-31 | `long lastUp, incrementModeTime;` `UInt32 doubleTime;` `Point lastWhere;` `short idleMode;` `Boolean doAutoDemo, switchedOut;` |
| `FileError.c` | — | none |
| `GameOver.c` | 40-46 | `pageType pages[8];` `Rect pageSrcRect, pageSrc[kPageFrames], lettersSrc[8], angelSrcRect;` `RgnHandle roomRgn;` `GWorldPtr pageSrcMap, gameOverSrcMap, angelSrcMap;` `GWorldPtr pageMaskMap, angelMaskMap;` `short countDown, stopPages, pagesStuck;` `Boolean gameOver;` |
| `Grease.c` | 26-27 | `greasePtr grease;` `short numGrease;` |
| `HighScores.c` | 45-48 | `Str31 highBanner;` `Str15 highName;` `short lastHighScore;` `Boolean keyStroke;` |
| `House.c` | 25-32 | `houseHand thisHouse;` `linksPtr linksList;` `Str32 thisHouseName;` `short srcLocations[24];` `short destLocations[24];` `short wasFloor, wasSuite;` `retroLink retroLinkList[24];` `Boolean houseUnlocked;` |
| `HouseInfo.c` | 34-37 | `Str255 banner, trailer;` `Rect houseEditText1, houseEditText2;` `short houseCursorIs;` `Boolean keyHit, tempPhoneBit;` |
| `HouseIO.c` | 31-36 | `Movie theMovie;` `Rect movieRect;` `short houseRefNum, houseResFork, wasHouseVersion;` `Boolean houseOpen, fileDirty, gameDirty;` `Boolean changeLockStateOfHouse, saveHouseLocked, houseIsReadOnly;` `Boolean hasMovie, tvInRoom;` |
| `HouseLegal.c` | 29-30 | `short houseErrors, wasRoom;` `Boolean isHouseChecks;` |
| `Input.c` | 30-34 | `demoPtr demoData;` `KeyMap theKeys;` `DialogPtr saveDial;` `short demoIndex, batteryFrame;` `Boolean isEscPauseKey, paused, batteryWasEngaged;` |
| `Interactions.c` | 42 | `short otherPlayerEscaped, activeRectEscaped;` |
| `InterfaceInit.c` | — | none (it only *writes* other files' globals) |
| `Link.c` | 23-28 | `Rect linkWindowRect;` `ControlHandle linkControl, unlinkControl;` `WindowPtr linkWindow;` `short isLinkH, isLinkV, linkRoom, linkType;` `Byte linkObject;` `Boolean isLinkOpen, linkerIsSwitch;` |
| `Main.c` | 24-26 | `short isVolume, wasVolume;` `short isDepthPref, dataResFile, numSMWarnings;` `Boolean quitting, doZooms, quickerTransitions, isUseSecondScreen;` |
| `MainWindow.c` | 27-43 | `CTabHandle theCTab;` `PixMapHandle thePMap;` `ColorSpec *wasColors;` `ColorSpec *newColors;` `CursHandle handCursorH, beamCursorH, vertCursorH, horiCursorH;` `CursHandle diagCursorH;` `Cursor handCursor, beamCursor, vertCursor, horiCursor;` `Cursor diagCursor;` `Rect workSrcRect;` `GWorldPtr workSrcMap;` `Rect mainWindowRect;` `WindowPtr mainWindow, menuWindow;` `short isEditH, isEditV;` `short playOriginH, playOriginV;` `short splashOriginH, splashOriginV;` `short theMode;` `Boolean fadeGraysOut, isDoColorFade, splashDrawn;` |
| `Map.c` | 36-44 | `Rect nailSrcRect, activeRoomRect, wasActiveRoomRect;` `Rect mapHScrollRect, mapVScrollRect, mapCenterRect;` `Rect mapWindowRect;` `GWorldPtr nailSrcMap;` `WindowPtr mapWindow;` `ControlHandle mapHScroll, mapVScroll;` `short isMapH, isMapV, mapRoomsHigh, mapRoomsWide;` `short mapLeftRoom, mapTopRoom;` `Boolean isMapOpen, doPrettyMap;` |
| `Marquee.c` | 23-25 | `marquee theMarquee;` `Rect marqueeGliderRect;` `Boolean gliderMarqueeUp;` |
| `Menu.c` | 32-33 | `MenuHandle appleMenu, gameMenu, optionsMenu, houseMenu;` `Boolean menusUp, resumedSavedGame;` |
| `Modes.c` | 13 | `short saidFollow;` |
| `Music.c` | 28-36 | `SndCallBackUPP musicCallBackUPP;` `SndChannelPtr musicChannel;` `Ptr theMusicData[kMaxMusic];` `short musicSoundID, musicCursor;` `short musicScore[kLastMusicPiece];` `short gameScore[kLastGamePiece];` `short musicMode;` `Boolean isMusicOn, isPlayMusicIdle, isPlayMusicGame;` `Boolean failedMusic, dontLoadMusic;` |
| `ObjectAdd.c` | 42 | `short wasFlower;` |
| `ObjectDraw.c` | — | none |
| `ObjectDraw2.c` | — | none |
| `ObjectDrawAll.c` | — | none |
| `ObjectEdit.c` | 28-33 | `Rect roomObjectRects[24];` `Rect initialGliderRect;` `Rect leftStartGliderSrc, rightStartGliderSrc;` `Rect leftStartGliderDest, rightStartGliderDest;` `short objActive;` `Boolean isFirstRoom;` |
| `ObjectInfo.c` | 106-107 | `short newDirection, newPoint;` `Byte newType;` |
| `ObjectRects.c` | — | none |
| `Objects.c` | 20-78 | all object art `GWorldPtr`s and source `Rect`s, plus `Rect tempManholes[kMaxTempManholes];` `savedType savedMaps[24];` `objDataPtr masterObjects;` `hotPtr hotSpots;` `short nLocalObj, nHotSpots, numMasterObjects, numLocalMasterObjects;` `short numTempManholes, tvWithMovieNumber;` `Boolean newState;` |
| `Play.c` | 47-54 | `phoneType thePhone, theChimes;` `Rect glidSrcRect, justRoomsRect;` `GWorldPtr glidSrcMap, glid2SrcMap;` `GWorldPtr glidMaskMap;` `long gameFrame;` `short batteryTotal, bandsTotal, foilTotal, mortals;` `Boolean playing, evenFrame, twoPlayerGame, showFoil, demoGoing;` `Boolean doBackground, playerSuicide, phoneBitSet, tvOn;` |
| `Player.c` | 43-53 | `gliderType theGlider, theGlider2;` `Rect shadowSrcRect;` `GWorldPtr shadowSrcMap;` `GWorldPtr shadowMaskMap;` `Rect shadowSrc[2];` `Rect gliderSrc[31];` `Rect transRect;` `long theScore;` `short fadeInSequence[16];` `short rightClip, leftClip, transRoom;` `Boolean shadowVisible, onePlayerLeft, playerDead;` |
| `Prefs.c` | — | none |
| `RectUtils.c` | — | none |
| `Render.c` | 35-42 | `Rect work2MainRects[48];` `Rect back2WorkRects[48];` `Rect shieldRect;` `RgnHandle mirrorRgn;` `Point shieldPt;` `long nextFrame;` `short numWork2Main, numBack2Work;` `Boolean hasMirror;` |
| `Room.c` | 24-31 | `roomPtr thisRoom;` `Rect backSrcRect;` `GWorldPtr backSrcMap;` `short numberRooms, thisRoomNumber, previousRoom;` `short leftThresh, rightThresh, lastBackground;` `Boolean autoRoomEdit, newRoomNow, noRoomAtAll;` `Boolean leftOpen, rightOpen, topOpen, bottomOpen;` `Boolean doBitchDialogs;` |
| `RoomGraphics.c` | 27-33 | `Rect suppSrcRect;` `GWorldPtr suppSrcMap;` `Rect localRoomsDest[9];` `Rect houseRect;` `short numNeighbors, numLights, thisTiles[8];` `short localNumbers[9], thisBackground;` `Boolean isStructure[9], wardBitSet;` |
| `RoomInfo.c` | 48-54 | `Rect tileSrc, tileDest, tileSrcRect, editTETextBox;` `Rect leftBound, topBound, rightBound, bottomBound;` `CGrafPtr tileSrcMap;` `short tempTiles[8];` `short tileOver, tempBack, cursorIs;` `Boolean originalLeftOpen, originalTopOpen, originalRightOpen, originalBottomOpen;` `Boolean originalFloor;` |
| `RubberBands.c` | 21-26 | `bandPtr bands;` `Rect bandsSrcRect;` `Rect bandRects[3];` `GWorldPtr bandsSrcMap;` `GWorldPtr bandsMaskMap;` `short numBands, bandHitLast;` |
| `SavedGames.c` | 20 | `gameType smallGame;` |
| `Scoreboard.c` | 29-42 | scoreboard `GWorldPtr`s and `Rect`s, `long displayedScore;` `short wasScoreboardMode;` `Boolean doRollScore;` |
| `Scrap.c` | 17 | `Boolean hasScrap, scrapIsARoom;` |
| `SelectHouse.c` | 46-53 | `Rect loadHouseRects[12];` `FSSpecPtr theHousesSpecs;` `FSSpec extraHouseSpecs[8];` `long lastWhenClick;` `Point lastWhereClick;` `short housesFound, thisHouseIndex, maxFiles, willMaxFiles;` `short housePage, demoHouseIndex, numExtraHouses;` `char fileFirstChar[12];` |
| `Settings.c` | 86-93 | `Rect prefButton[4], controlRects[4];` `Str15 leftName, rightName, batteryName, bandName;` `Str15 tempLeftStr, tempRightStr, tempBattStr, tempBandStr;` `long tempLeftMap, tempRightMap, tempBattMap, tempBandMap;` `short whichCtrl, wasDepthPref;` `Boolean wasFade, wasIdle, wasPlay, wasTransit, wasZooms, wasBackground;` `Boolean wasEscPauseKey, wasDemos, wasScreen2, nextRestartChange, wasErrorCheck;` `Boolean wasPrettyMap, wasBitchDialogs;` |
| `Sound.c` | 28-34 | `SndCallBackUPP callBack0UPP, callBack1UPP, callBack2UPP;` `SndChannelPtr channel0, channel1, channel2;` `Ptr theSoundData[kMaxSounds];` `short numSoundsLoaded, priority0, priority1, priority2;` `short soundPlaying0, soundPlaying1, soundPlaying2;` `Boolean soundLoaded[kMaxSounds], dontLoadSounds;` `Boolean channelOpen, isSoundOn, failedSound;` |
| `StringUtils.c` | — | none |
| `StructuresInit.c` | — | none |
| `StructuresInit2.c` | — | none |
| `Tools.c` | 57-65 | `Rect toolsWindowRect, toolSrcRect, toolTextRect;` `Rect toolRects[kTotalTools];` `ControlHandle classPopUp;` `GWorldPtr toolSrcMap;` `WindowPtr toolsWindow;` `short isToolsH, isToolsV;` `short toolSelected, toolMode;` `short firstTool, lastTool, objectBase;` `Boolean isToolsOpen;` |
| `Transit.c` | 17-18 | `short linkedToWhat;` `Boolean takingTheStairs, firstPlayer;` |
| `Transitions.c` | — | none |
| `Trip.c` | — | none |
| `Triggers.c` | 28 | `trigType triggers[kMaxTriggers];` |
| `Utilities.c` | 18-19 | `GDHandle thisGDevice;` `UInt32 theSeed;` |
| `Validate.c` | 37-39 | `long encryptedNumber;` `short theSystemVol;` `Boolean legitMasterDisk, bailOut, didValidation;` (dead under `COMPILENOCP`) |
| `WindowUtils.c` | 18 | `WindowPtr mssgWindow;` |

`GliderVars.h` re-exports only a curated subset of these as `extern`
(`GliderVars.h:11-58`); the rest are `extern`-declared ad hoc at the top of the
files that use them. A Go port that wants a single `type Game struct` should use
§16.1–§16.5 as the field list and treat everything in §16.6 not listed there as
either art tables (immutable after init) or subsystem-private.

---

## 17. Module map

All 67 `.c` files, alphabetical, with line counts (LF-converted; identical to the
CR line count of the originals) and role. Total: **46,177** lines of C across 67
`.c` files plus **2,314** lines across 25 `.h` files.

| File | Lines | Role |
|---|---:|---|
| `About.c` | 257 | The About box: modal dialog with a custom `AboutFilter` event filter, a hand-drawn OK button (`okayButtRgn`), and a PICT redraw hook (`UpdateMainPict`). |
| `AnimCursor.c` | 263 | Animated ("spinning beachball") cursor from an `acur` resource; `LoadCursors`/`SpinCursor`/`BackSpinCursor`, mono and colour variants. |
| `AppleEvents.c` | 208 | The four required Apple Event handlers (oapp/odoc/pdoc/quit) and `SetUpAppleEvents`. See §13. |
| `Banner.c` | 237 | Draws the house's banner/trailer text on the splash screen and the "stars remaining" count; `CountStarsInHouse`. |
| `ColorUtils.c` | 224 | Thin QuickDraw colour wrappers: `ColorText`, `ColorRect`, `ColorOval`, `ColorRegion`, `ColorLine`, `HiliteRect`, `ColorFrameRect`, `LtGray/Gray/DkGrayForeColor`, `RestoreColorsSlam`. |
| `Coordinates.c` | 196 | Editor "Coordinates" floating windoid showing h/v/d of the selected object. |
| `DebugUtilities.c` | 355 | 14 developer-only probes (§15). None is called on the shipping path. |
| `DialogUtils.c` | 806 | ~40 Dialog Manager helpers: centring, zoom-rect animation, get/set item text and values, radio groups, pop-up menus, item framing/shadowing. The whole UI layer sits on this. |
| `DynamicMaps.c` | 798 | The 24-slot `savedMaps` background-patch cache plus the per-effect object pools: sparkles, flying points, candle flames, tiki flames, BBQ coals, pendulums, stars, shredded gliders. |
| `Dynamics.c` | 776 | Collision of the glider with dynamic objects, plus per-object render and per-frame handlers for toast, Mac Plus, TV, coffee, outlet, VCR, stereo, microwave. |
| `Dynamics2.c` | 589 | Per-frame behaviour of the six moving enemies: balloon, copter, dart, ball, drip, fish. |
| `Dynamics3.c` | 555 | The `dinahs` dynamic-object array itself (max 18): `AddDynamicObject`, `HandleDynamics`, `RenderDynamics`, `ZeroDinahs`. |
| `Environ.c` | 734 | `thisMac` environment probing, colour-depth switch/restore, `CheckOurEnvirons`, `CheckMemorySize`, `SetAppMemorySize`. See §10-§11. |
| `Events.c` | 573 | The outer `WaitNextEvent` loop and every event dispatcher. See §5. |
| `FileError.c` | 102 | One function, `CheckFileError`, mapping an `OSErr` to an alert. |
| `GameOver.c` | 507 | Game-over sequencing: `FlagGameOver`, the falling-pages/star/angel animations, `DoDiedGameOver`. See §6.7. |
| `Grease.c` | 302 | Grease spills (max 16): add, re-back-up, spill, redraw. |
| `HighScores.c` | 856 | The high-score table UI, sorting, and its on-disk read/write. |
| `House.c` | 860 | House-level operations on `thisHouse`: create/init, room counting, glider start position, link list generation, retro-links, Go-To dialog, v1→v2 conversion, whole-house shifting. |
| `HouseInfo.c` | 343 | The editor's House Info dialog (banner, trailer, telephone bit). |
| `HouseIO.c` | 708 | House file open/read/write/close, resource-fork handling, QuickTime movie load, `QuerySaveChanges`, `YellowAlert`. |
| `HouseLegal.c` | 1219 | House validation and repair: `KeepObjectLegal`, room-count/number validation, duplicate floor-suite detection, `CompressHouse`, staircase pairing, `CheckHouseForProblems`. |
| `Input.c` | 398 | Keyboard polling (`GetInput`), demo playback (`GetDemoInput`), demo recording (`LogDemoKey`), `DoPause`, `DoCommandKey`, battery/helium engage. See §6.5-§6.6, §8.4. |
| `Interactions.c` | 1777 | The interaction engine: `SectGlider`, `HandleInteraction` (the 28-action switch), `HandleSwitches`, `FlagStillOvers`. The largest single file. |
| `InterfaceInit.c` | 220 | `InitializeMenus`, `GetExtraCursors`, `VariableInit`. See §4.3. |
| `Link.c` | 396 | Editor object-linking windoid, `MergeFloorSuite`/`ExtractFloorSuite`, `DoLink`/`DoUnlink`. |
| `Main.c` | 386 | `main()`, the 28-step boot, the outer `while (!quitting)` loop, and prefs read/write. See §3. |
| `MainWindow.c` | 601 | The main window and the splash screen: `OpenMainWindow`, `UpdateMainWindow`, `RedrawSplashScreen`, `DrawOnSplash`, `ZoomBetweenWindows`, `HandleMainClick`; owns `theMode`. See §9. |
| `Map.c` | 796 | Editor map window: room thumbnails, live scroll bars, room drag/move, `CenterMapOnRoom`. |
| `Marquee.c` | 511 | Editor selection marquee: animated dashed outline, drag-out, drag-move, handle and corner dragging. |
| `Menu.c` | 813 | `DoMenuChoice` and the four per-menu handlers, `UpdateMenus`, `OpenCloseEditWindows`. See §5.10-§5.12. |
| `Modes.c` | 640 | **Glider** mode transitions (not app modes): the 26 `StartGlider…`/`FlagGlider…` functions that drive `gliderType.mode`. See §4.1. |
| `Music.c` | 419 | Background music: one `SndChannel`, a callback-driven playlist, `SetMusicalMode`, `MusicBytesNeeded`. |
| `ObjectAdd.c` | 1084 | Editor object creation: `AddNewObject`, slot finding, per-class default field values. |
| `ObjectDraw.c` | 1406 | Object rendering, part 1: blowers, furniture, clocks, prizes, grease, foil, slider. |
| `ObjectDraw2.c` | 1437 | Object rendering, part 2: transports, switches, lights, appliances, enemies, clutter, custom PICTs. |
| `ObjectDrawAll.c` | 966 | `DrawARoomsObjects` — the dispatcher that walks a room's 24 object slots and calls the right `Draw…` routine, registering dynamic objects and hot spots as it goes. |
| `ObjectEdit.c` | 2810 | The editor's object-manipulation layer: selection, click handling, delete/duplicate/move, handles, next/prev, `GoToObjectInRoom`. Second-largest file. |
| `ObjectInfo.c` | 2567 | The per-object-class Info dialogs (one dialog layout per object family). |
| `ObjectRects.c` | 1188 | `GetObjectRect` (the geometry of all **117** object types — one `case` each, `ObjectRects.c:33-273`, plus `kObjectIsEmpty`; 117 is the count of object-type `#define`s in `GliderDefines.h:311-435`, whose *highest code* is 143 = `kChimes` = `0x8F` and whose index space is 144 = `kNumSrcRects`) and `CreateActiveRects` (the interaction hot-spot generator), plus `VerticalRoomOffset`/`OffsetRectRoomRelative`. |
| `Objects.c` | 1001 | Object art GWorlds and source rects, the `masterObjects` / `hotSpots` / `savedMaps` arrays, `ListAllLocalObjects`, `GetObjectState`/`SetObjectState`, link queries. |
| `Play.c` | 821 | `NewGame`, `PlayGame`, `DoDemoGame`, `SetObjectsToDefaults`, telephone/chime schedulers. The inner game loop. See §6. |
| `Player.c` | 1605 | `HandleGlider` — glider physics, animation, stair/duct/mail completion, foil, `OffAMortal`; owns `theGlider`/`theGlider2`/`theScore`. |
| `Prefs.c` | 281 | Generic preferences-file plumbing (FindFolder, create/read/write a resource-based prefs file). See §12. |
| `RectUtils.c` | 318 | 19 `Rect` helpers: normalize, zero-corner, centre-on-point, half-wide/tall, global/local conversion, `ForceRectInRect`, `QUnionSimilarRect`, `FrameRectSansCorners`. |
| `Render.c` | 772 | The frame compositor: dirty-rect lists, `CopyRectsQD`, `RenderFrame`, `InitGarbageRects`, the five single-rect copiers, the mirror region, and the (unused) direct-blit fast paths. See §7. |
| `Room.c` | 1206 | Room-level state: `thisRoom`, room create/delete, `CopyRoomToThisRoom`/`CopyThisRoomToRoom`, `ReflectCurrentRoom`, neighbour lookup, `DetermineRoomOpenings`, lighting count. |
| `RoomGraphics.c` | 462 | `ReadyLevel` and `DrawLocale`: builds `backSrcMap` for the 3×3 neighbourhood, computes `houseRect`/`localRoomsDest`, `RedrawRoomLighting`. See §9.4. |
| `RoomInfo.c` | 907 | The editor's Room Info dialog (name, background, 8 tiles, openings, floor/suite). |
| `RubberBands.c` | 318 | Rubber bands (max 2): `AddBand`, `HandleBands`, `KillAllBands`. |
| `SavedGames.c` | 352 | Saved-game read/write: the 40-byte `gameType` and the variable-length `game2Type`. |
| `Scoreboard.c` | 457 | The 20-pixel scoreboard band: full refresh, rolling score, quick refresh of gliders/battery/bands/foil, `AdjustScoreboardHeight`. See §9.6. |
| `Scrap.c` | 517 | Clipboard copy/paste of rooms and objects, plus Drag Manager room dragging. |
| `SelectHouse.c` | 676 | The Load House browser: directory scan (`DoDirSearch`), house list paging, `demoHouseIndex` discovery. See §8.2. |
| `Settings.c` | 1482 | The Preferences dialog: four panes (controls, sounds, display, brains) writing the `was…` temporaries back into the live prefs globals. |
| `Sound.c` | 535 | Three `SndChannel`s with a priority scheme, the 64-entry `theSoundData` cache, `PlayPrioritySound`, trigger-sound load/dump. |
| `StringUtils.c` | 329 | Pascal-string helpers: copy, compare, concat, line extraction, `WrapText`, `CollapseStringToWidth`, `GetLocalizedString`. |
| `StructuresInit.c` | 724 | One-time build of the art GWorlds and their source-rect tables (scoreboard, glider, blowers, furniture, prizes, transports, switches, lights, appliances, enemies). |
| `StructuresInit2.c` | 476 | `CreateOffscreens` (the two big GWorlds), `CreatePointers` (all the `NewPtr` pools + the demo resource), `InitSrcRects` (the 144-entry `srcRects` table). See §4.4, §7.1. |
| `Tools.c` | 544 | Editor tool palette windoid: 10 tool modes, tool tiles, selection framing. |
| `Transit.c` | 558 | Room-to-room transit: `MoveRoomToRoom`, `TransportRoomToRoom`, `MoveDuctToDuct`, `MoveMailToMail`, `FollowTheLeader`, `ForceKillGlider`. |
| `Transitions.c` | 145 | Three screen transitions: `PourScreenOn`, `WipeScreenOn`, `DumpScreenOn`. |
| `Triggers.c` | 205 | The 16-slot delayed-action queue (`kMaxTriggers` 16, `Triggers.c:12`): `ArmTrigger`, `FindEmptyTriggerSlot`, `HandleTriggers` (called once per frame from `PlayGame`), `FireTrigger`, `ZeroTriggers`. |
| `Trip.c` | 245 | "Tripping" objects: 14 `Toggle…` and 7 `Trigger…` functions that flip appliance/enemy state, plus `UpdateOutletsLighting`. |
| `Utilities.c` | 788 | `ToolBoxInit`, `RandomInt`/`RandomLongQUS`, `RedAlert`, `FindOurDevice`, `WaitForInputEvent`, `WaitCommandQReleased`, `DelayTicks`, misc. |
| `Validate.c` | 398 | Master-disk copy protection. Entirely dead: `COMPILENOCP` is defined (`GliderDefines.h:14`). |
| `WindowUtils.c` | 172 | Window helpers: `GetWindowLeftTop`/`GetWindowRect`, `FlagWindowFloating`, and the progress "message window". |

### 17.1 Files a Go port needs first

In dependency order, the minimum set to get a playable single-room build:

1. `Main.c` + `InterfaceInit.c` + `Environ.c` — boot and globals.
2. `StructuresInit.c` + `StructuresInit2.c` + `Objects.c` — art tables and buffers.
3. `HouseIO.c` + `House.c` + `Room.c` — load a house, pick a room.
4. `RoomGraphics.c` + `ObjectDrawAll.c` + `ObjectDraw.c` + `ObjectDraw2.c` — build `backSrcMap`.
5. `ObjectRects.c` — hot spots.
6. `Play.c` + `Render.c` + `Input.c` + `Player.c` + `Modes.c` — the loop.
7. `Interactions.c` + `Transit.c` — make the world respond.
8. `Scoreboard.c` + `GameOver.c` — HUD and end conditions.

The entire editor (`ObjectEdit.c`, `ObjectInfo.c`, `Map.c`, `Tools.c`, `Link.c`,
`Coordinates.c`, `Marquee.c`, `RoomInfo.c`, `HouseInfo.c`, `Scrap.c`,
`HouseLegal.c` — 10,806 lines, 23.4% of the codebase) is orthogonal to gameplay
and can be deferred or omitted.

---

## Appendix A: Empirical verification

Everything in this appendix was produced by actually parsing the bytes, not by
reading the C. Method:

* `GliderPRO/Glider PRO.r` is a DeRez text dump. A parser
  (`/tmp/wf-arch/verify/rez.py`) matches `^data 'TYPE' (ID[, "name"]...) {`,
  concatenates every `$"…"` hex run until `};`, and yields raw resource bytes.
  It recovers **538 resources across 35 types**.
* Struct sizes/offsets were obtained by compiling `GliderStructs.h` and the
  `prefsInfo` block from `Externs.h` with `#pragma pack(2)` (to emulate
  `#pragma options align=mac68k`), `typedef int32_t SInt32` (to defeat LP64
  `long`), and `Point`/`Rect`/`FSSpec`/`Str…` shims, then printing
  `sizeof`/`offsetof`.

### A.1 The `'demo'` resource (attract-mode input stream)

Observed:

```
resource:            'demo' 128, name = "" (no resource name)
length:              6702 bytes   == kDemoLength (GliderDefines.h:625)   MATCH
sizeof(demoType):    6 bytes      (long frame; char key; char padding;)
records:             6702 / 6 = 1117 exactly, remainder 0
first 36 bytes:      00 00 00 2e 00 72  00 00 00 2f 00 72  00 00 00 30 00 72
                     00 00 00 31 00 45  00 00 00 38 00 06  00 00 00 39 00 72
```

Decoded as big-endian `(int32 frame, uint8 key, uint8 padding)`:

| # | frame | key | padding |
|---:|---:|---:|---:|
| 1 | 46 | 0 | 114 |
| 2 | 47 | 0 | 114 |
| 3 | 48 | 0 | 114 |
| 4 | 49 | 0 | 69 |
| 5 | 56 | 0 | 6 |
| 6 | 57 | 0 | 114 |
| 7 | 58 | 0 | 111 |
| 8 | 59 | 0 | 114 |
| … | … | … | … |
| 1114 | 3411 | 0 | 254 |
| 1115 | 3412 | 0 | 248 |
| 1116 | 3413 | 0 | 7 |
| 1117 | 3414 | 0 | 1 |

Aggregate properties:

| Property | Observed |
|---|---|
| `frame` strictly increasing | **true** for all 1117 records |
| `frame` range | 46 … 3414 |
| gap between consecutive frames | min 1, max 100, mean 3.02 |
| `key` histogram | `{0: 910, 1: 198, 3: 9}` |
| distinct `padding` values | **109** — uninitialised garbage; `LogDemoKey` (`Input.c:44-49`) never writes it |
| playback duration at 30 fps | 3414 / 30 = **113.8 s** |

Consequences:

1. `key` only ever takes the values 0, 1 and 3. Cross-referencing the
   `LogDemoKey` call sites in `GetInput` (§8.4): 0 = right key, 1 = left key,
   2 = battery, 3 = rubber band. Value 2 **never appears**, so the recorded demo
   never fires the battery. (`GetDemoInput`'s `case 0:` / `case 1:` comments say
   the opposite; the comments are wrong, see §8.4.)
2. The stream is a **one-shot-per-frame** stream, not a toggle stream and not a
   held-state stream. `GetDemoInput` clears `heldLeft`, `heldRight` and `tipped`
   at the top of *every* frame (`Input.c:220-222`) and only applies a key when
   `gameFrame == demoData[demoIndex].frame` (`Input.c:224`), consuming exactly
   one record. So a record means "this key was down on this one frame"; a key
   held for *n* frames was recorded as *n* consecutive records (which is why the
   mean frame gap is 3.02 and there are 1117 records for 3369 frames of play).
   A porter must **not** implement toggling: apply the record's action for that
   single frame and nothing otherwise.
3. `padding` must be preserved byte-for-byte if a port ever re-serialises the
   resource, but has no semantic content.
4. Nothing terminates the stream. Playback runs off the end of the array
   (§8.5) — `demoIndex` keeps incrementing past 1116 and reads whatever follows
   the 6702-byte block. Because the trailing `frame` values read as garbage they
   are essentially never equal to `gameFrame`, so in practice no further input is
   injected; but this is an out-of-bounds read a Go port must not reproduce.

### A.2 Struct sizes and offsets

Compiled with 2-byte alignment (`mac68k`) and 32-bit `long`:

| Type | Header comment | Observed `sizeof` | Agrees? |
|---|---|---:|---|
| `Point` | — | 4 | — |
| `Rect` | — | 8 | — |
| `FSSpec` | 70 (`GliderStructs.h:146`) | 70 | yes |
| `blowerType` | "total = 10" (`:19`) | 10 | yes |
| `furnitureType` | "total = 10" (`:25`) | 10 | yes |
| `bonusType` | "total = 10" (`:34`) | 10 | yes |
| `transportType` | "total = 10" (`:43`) | 10 | yes |
| `switchType` | "total = 10" (`:52`) | 10 | yes |
| `lightType` | "total = 10" (`:62`) | 10 | yes |
| `applianceType` | "total = 10" (`:72`) | 10 | yes |
| `enemyType` | "total = 10" (`:82`) | 10 | yes |
| `clutterType` | "total = 10" (`:88`) | 10 | yes |
| `objectType` | "total = 12" (`:105`) | 12 | yes |
| `scoresType` | "total = 292" (`:114`) | 292 | yes |
| `gameType` | "total = 40" (`:134`) | 40 | yes |
| `savedRoom` | "total = 292" (`:142`) | 292 | yes |
| `game2Type` | "total = 114" (`:164`) | **110** | **NO** |
| `roomType` | "total = 348" (`:180`) | 348 | yes |
| `houseType` | "total = 866 +" (`:198`) | 866 | yes |
| `gliderType` | (none) | 110 | — |
| `hotObject` | (none) | 16 | — |
| `objDataType` | (none) | 26 | — |
| `demoType` | (none) | 6 | — |
| `prefsInfo` | (none) | 226 | — |

The `game2Type` discrepancy is explained by the flexible array member:
`savedData[]` contributes 0 bytes to `sizeof`, but the author's comment counted
it as 4 (`GliderStructs.h:163` is annotated `// 4`). 110 + 4 = 114. **A Go port
must use 110 as the header size when reading a saved game**; the room array
starts at offset 110.

`houseType` field offsets (the on-disk house file layout, big-endian):

| Field | Type | Offset | Size |
|---|---|---:|---:|
| `version` | `short` | 0 | 2 |
| `unusedShort` | `short` | 2 | 2 |
| `timeStamp` | `long` | 4 | 4 |
| `flags` | `long` | 8 | 4 |
| `initial` | `Point` | 12 | 4 |
| `banner` | `Str255` | 16 | 256 |
| `trailer` | `Str255` | 272 | 256 |
| `highScores` | `scoresType` | 528 | 292 |
| `savedGame` | `gameType` | 820 | 40 |
| `hasGame` | `Boolean` | 860 | 1 |
| `unusedBoolean` | `Boolean` | 861 | 1 |
| `firstRoom` | `short` | 862 | 2 |
| `nRooms` | `short` | 864 | 2 |
| `rooms[]` | `roomType[]` | **866** | 348 × `nRooms` |

`roomType` field offsets (each room record inside the house file):

| Field | Type | Offset | Size |
|---|---|---:|---:|
| `name` | `Str27` | 0 | 28 |
| `bounds` | `short` | 28 | 2 |
| `leftStart` | `Byte` | 30 | 1 |
| `rightStart` | `Byte` | 31 | 1 |
| `unusedByte` | `Byte` | 32 | 1 |
| `visited` | `Boolean` | 33 | 1 |
| `background` | `short` | 34 | 2 |
| `tiles[8]` | `short[8]` | 36 | 16 |
| `floor` | `short` | 52 | 2 |
| `suite` | `short` | 54 | 2 |
| `openings` | `short` | 56 | 2 |
| `numObjects` | `short` | 58 | 2 |
| `objects[24]` | `objectType[24]` | **60** | 288 |

60 + 288 = 348. No padding anywhere: every field is naturally 2-byte aligned.

`game2Type` field offsets:

| Field | Type | Offset |
|---|---|---:|
| `house` | `FSSpec` | 0 |
| `version` | `short` | 70 |
| `wasStarsLeft` | `short` | 72 |
| `timeStamp` | `long` | 74 |
| `where` | `Point` | 78 |
| `score` | `long` | 82 |
| `unusedLong` | `long` | 86 |
| `unusedLong2` | `long` | 90 |
| `energy` | `short` | 94 |
| `bands` | `short` | 96 |
| `roomNumber` | `short` | 98 |
| `gliderState` | `short` | 100 |
| `numGliders` | `short` | 102 |
| `foil` | `short` | 104 |
| `nRooms` | `short` | 106 |
| `facing` | `Boolean` | 108 |
| `showFoil` | `Boolean` | 109 |
| `savedData[]` | `savedRoom[]` | **110** |

Note `timeStamp` at offset 74 is a `long` on a **2-byte-aligned** boundary. On
a 4-byte-aligned compiler it would be pushed to 76 and the whole record would
shift. This is exactly the class of bug `#pragma options align=mac68k`
(`Externs.h:231`) exists to prevent, and it is the single most likely thing for a
Go port to get wrong.

`gliderType` selected offsets (in-memory only, never serialised):

| Field | Offset |
|---|---:|
| `src`, `mask`, `dest`, `whole`, `destShadow`, `wholeShadow`, `clip`, `enteredRect` | 0, 8, 16, 24, 32, 40, 48, 56 |
| `leftKey` (`long`) | 64 |
| `rightKey`, `battKey`, `bandKey` | 68, 72, 76 |
| `hVel` | 80 |
| `mode` | 92 |
| `facing` | 98 |
| `dontDraw` | 107 |
| (total) | 110 |

The four key fields are `long` (`GliderStructs.h:205-206`) but hold a plain
`KeyMap` **bit offset in the range 0-127** — nothing is packed or shifted. They
are assigned straight from the `k…KeyMap` constants (e.g.
`theGlider.leftKey = kLeftArrowKeyMap;` = 124, `Settings.c:89`, `:1237`) and
consumed straight by `BitTst(&theKeys, thisGlider->leftKey)` (`Input.c:301`,
`:306`, `:318`, `:330`, `:344`). The `long` width is simply what `BitTst`'s
`long` offset parameter wanted. Remember that a bit offset is *not* the raw
virtual key code: see §2 and `KeyMapOffsetFromRawKey` (`Utilities.c:504-517`).

`prefsInfo` (the preferences resource payload) — `sizeof` = **226**, exactly two
pad bytes, at offsets **145** and **225**:

| Offset | Field | Type |
|---:|---|---|
| 0 | `wasDefaultName` | `Str32` (33) |
| 33 | `wasLeftName` | `Str15` (16) |
| 49 | `wasRightName` | `Str15` |
| 65 | `wasBattName` | `Str15` |
| 81 | `wasBandName` | `Str15` |
| 97 | `wasHighName` | `Str15` |
| 113 | `wasHighBanner` | `Str31` (32) |
| *145* | *(pad)* | 1 byte, inserted to 2-align the next `long` |
| 146 | `wasLeftMap` | `long` |
| 150 | `wasRightMap` | `long` |
| 154 | `wasBattMap` | `long` |
| 158 | `wasBandMap` | `long` |
| 162 | `wasVolume` | `short` |
| 164 | `prefVersion` | `short` |
| 166 | `wasMaxFiles` | `short` |
| 168 | `wasEditH` | `short` |
| 170 | `wasEditV` | `short` |
| 172 | `wasMapH` | `short` |
| 174 | `wasMapV` | `short` |
| 176 | `wasMapWide` | `short` |
| 178 | `wasMapHigh` | `short` |
| 180 | `wasToolsH` | `short` |
| 182 | `wasToolsV` | `short` |
| 184 | `wasLinkH` | `short` |
| 186 | `wasLinkV` | `short` |
| 188 | `wasCoordH` | `short` |
| 190 | `wasCoordV` | `short` |
| 192 | `isMapLeft` | `short` |
| 194 | `isMapTop` | `short` |
| 196 | `wasNumNeighbors` | `short` |
| 198 | `wasDepthPref` | `short` |
| 200 | `wasToolGroup` | `short` |
| 202 | `smWarnings` | `short` |
| 204 | `wasFloor` | `short` |
| 206 | `wasSuite` | `short` |
| 208 | `wasZooms` | `Boolean` |
| 209 | `wasMusicOn` | `Boolean` |
| 210 | `wasAutoEdit` | `Boolean` |
| 211 | `wasDoColorFade` | `Boolean` |
| 212 | `wasMapOpen` | `Boolean` |
| 213 | `wasToolsOpen` | `Boolean` |
| 214 | `wasCoordOpen` | `Boolean` |
| 215 | `wasQuickTrans` | `Boolean` |
| 216 | `wasIdleMusic` | `Boolean` |
| 217 | `wasGameMusic` | `Boolean` |
| 218 | `wasEscPauseKey` | `Boolean` |
| 219 | `wasDoAutoDemo` | `Boolean` |
| 220 | `wasScreen2` | `Boolean` |
| 221 | `wasDoBackground` | `Boolean` |
| 222 | `wasHouseChecks` | `Boolean` |
| 223 | `wasPrettyMap` | `Boolean` |
| 224 | `wasBitchDialogs` | `Boolean` |
| *225* | *(pad)* | 1 byte, tail padding to make 226 even |

17 `Boolean`s at offsets 208-224 is an odd count, hence the tail pad.

### A.3 PICT dimensions

Every PICT the architecture layer references. `picSize` is the resource's own
16-bit length field; `frame` is the `Rect` at bytes 2-9.

| PICT | Constant | bytes | `picSize` field | frame (t,l,b,r) | w × h | PICT ver |
|---:|---|---:|---:|---|---|---|
| 1000 | `kSplash8BitPICT` (`GliderDefines.h:524`) | 108482 | 42946 | (0,0,460,640) | **640 × 460** | 2 |
| 1015 | `kEscPausePictID` (`Input.c:17`) | 6092 | 6092 | (0,0,54,214) | 214 × 54 | 2 |
| 1016 | `kTabPausePictID` (`Input.c:18`) | 6106 | 6106 | (0,0,54,214) | 214 × 54 | 2 |
| 1019 | — (game-over "angel") | 4064 | 4064 | (0,0,44,96) | 96 × 44 | 2 |
| 1021 | `kMilkywayPictID` (`GameOver.c:22`) | 51520 | 51520 | (0,0,460,640) | 640 × 460 | 2 |
| 1988 | `kLettersPictID` (`GameOver.c:21`) | 5900 | 5900 | (0,0,256,25) | 25 × 256 | 2 |
| 1989 | `kPagesMaskID` (`GameOver.c:20`) | 1851 | 1851 | (0,0,448,32) | 32 × 448 | **1** |
| 1990 | `kPagesPictID` (`GameOver.c:19`) | 10052 | 10052 | (0,0,448,32) | 32 × 448 | 2 |
| 1995 | `kStarPictID` (`GliderDefines.h:534`) | 20746 | 20746 | (0,0,460,640) | 640 × 460 | 2 |
| 1997 | `kScoreboardPictID` (`GliderDefines.h:623`) | 13296 | 13296 | (0,0,20,1536) | **1536 × 20** | 2 |
| 1999 | — (floor support strip) | 16850 | 16850 | (0,0,44,512) | **512 × 44** | 2 |
| 2000 | `kSimpleRoom` / `kBaseBackgroundID` | 31838 | 31838 | (0,0,322,512) | **512 × 322** | 2 |
| 2017 | `kStars` (last background) | 12624 | 12624 | (0,0,322,512) | 512 × 322 | 2 |
| 3963 | `kGliderFoil2PictID` | 15258 | 15258 | (0,0,668,48) | **48 × 668** | 2 |
| 3974 | `kGlider2PictID` | 17604 | 17604 | (0,0,668,48) | 48 × 668 | 2 |
| 3976 | `kGliderFoilPictID` | 14846 | 14846 | (0,0,668,48) | 48 × 668 | 2 |
| 3999 | `kGliderPictID` | 18688 | 18688 | (0,0,668,48) | 48 × 668 | 2 |
| 4018 | — (scoreboard digits/icons) | 4864 | 4864 | (0,0,69,128) | 128 × 69 | 2 |

Cross-checks this confirms:

| Claim | Evidence |
|---|---|
| Room artwork is exactly `kRoomWide` × `kTileHigh` | PICT 2000 and 2017 are 512 × 322; `kRoomWide` 512 (`GliderDefines.h:499`), `kTileHigh` 322 (`:498`) |
| The scoreboard strip is `kMaxViewWidth` × `kScoreboardTall` | PICT 1997 is 1536 × 20; `kMaxViewWidth` 1536 (`:267`), `kScoreboardTall` 20 (`:515`) |
| The floor-support strip is `kRoomWide` × `kFloorSupportTall` | PICT 1999 is 512 × 44; `kFloorSupportTall` 44 (`:500`) |
| Glider sheets are `kGliderWide` wide | all four are 48 wide; `kGliderWide` 48 (`:548`) |
| The falling-pages sheet holds `kPageFrames` 32×32 cells | PICT 1990 is 32 × 448 = 32 × (14 × 32); `kPageFrames` 14 (`GameOver.c:18`) |
| The splash art is 460 tall and the `480` in `splashOriginV` is 460 + `kScoreboardTall` | PICT 1000 frame is (0,0,**460**,640); `OpenMainWindow` (`MainWindow.c:241`) subtracts **480** from the screen height, which is exactly the centred offset inside `workSrcMap` (= `houseRect` = screen − 20). Not an off-by-20 bug — see [§9.3](#93-openmainwindow--exact-geometry). |

Two Toolbox gotchas visible in the raw bytes:

1. **`picSize` overflows.** PICT 1000 is 108,482 bytes but its `picSize` word
   reads 42,946 (= 108482 mod 65536). It is the only PICT in this table that
   exceeds 65,535 bytes, and the only one whose `picSize` disagrees with its
   actual length (PICT 1021, at 51,520 bytes, reports 51,520 correctly). A Go
   PICT decoder **must ignore `picSize`** and use the actual resource length.
   QuickDraw itself ignored it for `DrawPicture`.
2. **Mixed PICT versions.** 17 of the 18 are v2 (`… 00 11 02 FF 0C 00 …`, i.e.
   opcode `0x0011` = version, data `0x02FF`, then the `0x0C00` header opcode);
   PICT 1989 alone is **v1** (`… 11 01 …`: a one-byte opcode `0x11` with data
   `0x01`). A v1 decoder uses 1-byte opcodes and no header opcode. A Go decoder
   must handle both or special-case 1989.

### A.4 Menu resources

Decoded `MENU` layout: `menuID`(2) `menuWidth`(2) `menuHeight`(2) `menuProc`(2)
`filler`(2) `enableFlags`(4) `title`(Pascal) then per item `text`(Pascal)
`iconIndex`(1) `keyEquiv`(1) `markChar`(1) `style`(1), terminated by a zero byte.
`enableFlags` bit 0 = whole menu; bits 1-31 = items 1-31.

| Res ID | `menuID` | Title | `enableFlags` | Items | Res bytes |
|---:|---:|---|---|---:|---:|
| 128 | 128 | `'\x14'` (the Apple glyph in Chicago) | `0xFFFFFFFB` | 2 | 45 |
| 129 | 129 | `"Game"` | `0xFFFFFFAF` | 7 | 112 |
| 130 | 130 | `"Options"` | `0xFFFFFFFB` | 5 | 89 |
| 131 | 131 | `"House"` | `0xFFFADF77` | 21 | 314 |
| 140 | **133** | `"Rooms"` | `0xFFF7FFFF` | 19 | 281 |
| 141 | **133** | `"Tools"` | `0xFFFFFFFF` | 9 | 142 |

Resource IDs 128-131 match `kAppleMenuID` 128, `kGameMenuID` 129,
`kOptionsMenuID` 130, `kHouseMenuID` 131 (`GliderDefines.h:186-189`). Resources
140 and 141 are pop-up menus embedded in dialogs and both carry the internal
`menuID` 133 — they are never in the menu bar.

Items, with observed key equivalents:

| Menu | # | Text | Cmd key |
|---|---:|---|---|
| Apple (128) | 1 | `About Glider PRO…` | — |
| | 2 | `-` (separator, disabled) | — |
| Game (129) | 1 | `New Game` + a stray `\x00` byte (item length 9) | **N** |
| | 2 | `Two Player Game` | **2** |
| | 3 | `Open Saved Game…` | **O** |
| | 4 | `-` (disabled) | — |
| | 5 | `Load House…` | **L** |
| | 6 | `-` (disabled) | — |
| | 7 | `Quit` | **Q** |
| Options (130) | 1 | `Room Editor` | **E** |
| | 2 | `-` (disabled) | — |
| | 3 | `High Scores…` | **H** |
| | 4 | `Preferences…` | **P** |
| | 5 | `Demo…` | **D** |
| House (131) | 1 | `New House…` | **N** |
| | 2 | `Save House` | **S** |
| | 3 | `-` | — |
| | 4 | `House Info…` | — |
| | 5 | `Room Info…` | **R** |
| | 6 | `Object Info…` | **I** |
| | 7 | `-` | — |
| | 8 | `Cut Room` | **X** |
| | 9 | `Copy Room` | **C** |
| | 10 | `Paste Room` | **V** |
| | 11 | `Delete Room` | — |
| | 12 | `Duplicate Object` | **D** |
| | 13 | `-` | — |
| | 14 | `Bring To Front` | **=** |
| | 15 | `Send To Back` | **-** |
| | 16 | `-` | — |
| | 17 | `Go To Room…` | **G** |
| | 18 | `-` | — |
| | 19 | `Map Window` | **M** |
| | 20 | `Tools Window` | **T** |
| | 21 | `Coordinate Window` | **K** |

Three collisions a Go port must resolve exactly as the Toolbox did — by menu
search order, `appleMenu`, `gameMenu`, `optionsMenu`, `houseMenu` (the order
`InsertMenu` is called in `InitializeMenus`, `InterfaceInit.c:47-73`):

| Key | Claimed by | Winner |
|---|---|---|
| **N** | Game ▸ New Game (129/1); House ▸ New House… (131/1) | Game, because 129 is inserted before 131 |
| **D** | Options ▸ Demo… (130/5); House ▸ Duplicate Object (131/12) | Options |
| **-** and **=** | House ▸ Send To Back / Bring To Front | no conflict |

Note also that `Cmd-Q` appears both as a menu equivalent (Game ▸ Quit) and as a
raw `KeyMap` poll (`kQKeyMap` 11 with `kCommandKeyMap` 48) inside the game loop
(`Input.c:286-287` → `DoCommandKey`, `Input.c:53-73`, testing `kQKeyMap` at
`Input.c:55`), which is why `WaitCommandQReleased` (`Utilities.c:485-499`)
exists: it drains the key-down so the menu handler does not also fire.

For the four menu-bar menus, the observed `enableFlags` exactly match the
separator positions: every bit that is 0 corresponds to a `"-"` item. (This does
*not* generalise: `MENU` 140's `0xFFF7FFFF` clears bit 19, and item 19 is
`"Original Artwork"` — a genuinely disabled item, not a separator.)

* 129 `0xFFFFFFAF` → bits 4 and 6 clear → items 4 and 6 are the separators. Yes.
* 131 `0xFFFADF77` → bits 3, 7, 13, 16, 18 clear → items 3, 7, 13, 16, 18 are
  the separators. Yes.

### A.5 Resource inventory

All 538 resources by type, from the DeRez dump:

| Type | Count | Type | Count | Type | Count |
|---|---:|---|---:|---|---:|
| `ALRT` | 26 | `ICON` | 35 | `dctb` | 20 |
| `BNDL` | 1 | `MENU` | 6 | `demo` | **1** |
| `CDEF` | 1 | `PAT#` | 1 | `icl4` | 6 |
| `CNTL` | 5 | `PICT` | **152** | `icl8` | 6 |
| `CURS` | 16 | `STR#` | 10 | `ics#` | 4 |
| `DITL` | 54 | `WDEF` | 2 | `ics4` | 4 |
| `DLGX` | 1 | `WIND` | **3** | `ics8` | 4 |
| `DLOG` | 28 | `acur` | 1 | `ictb` | 1 |
| `FREF` | 6 | `cctb` | 1 | `mctb` | 5 |
| `ICN#` | 6 | `cicn` | 44 | `ozm5` | 1 |
| | | `clut` | 2 | `snd ` | **70** |
| | | `crsr` | 12 | `vers` | 2 |
| | | | | `wctb` | 1 |

Three `STR#` counts independently confirm three enumerations in the headers:

| `STR#` | Res name | First word (count) | Confirms |
|---:|---|---:|---|
| 170 | "Errors" | `0x000D` = 13 | the 13 `kRedAlert…` codes (`Externs.h:183-195`) |
| 1006 | "Yellow Alerts" | `0x0018` = 24 | the 24 `kYellow…` codes (`GliderDefines.h:18-41`) |
| 1007 | "Object Names" | `0x0090` = 144 | `kNumSrcRects` `0x90` = 144 (`GliderDefines.h:437`) |
| 160 | "Prefs" | `0x0001` = 1, string 1 = `"Preferences"` | `kDefaultPrefFName` / `kPrefsFNameIndex` 1 (`Prefs.c:21`, `:24`) |
| 1005 | "Months" | `0x000C` = 12 | month names for high-score timestamps |

`'acur' 128` decodes as `numFrames = 0x000C` (12), `used = 0`, then 12 `(resID,
pad)` pairs: cursor IDs **160, 159, 158, 157, 156, 155, 154, 153, 152, 151, 150,
149** — the animated beachball, run backwards. Total 4 + 12×4 = 52 bytes, matching
the observed length.

`'ozm5' 0` is the 32-byte "Owner resource": `1F A9 20 31 39 39 34 …` = the
Pascal string `"© 1994-95 Casady & Greene, Inc."` (0x1F = length 31, 0xA9 = `©`
in Mac Roman). `'ozm5'` is also `kPrefCreatorType` (`Prefs.c:18`).

**There is no `'SIZE'` resource.** The 35 types present are listed above and
`SIZE` is not among them. `SetAppMemorySize` (`Environ.c:699-733`) is the function
that would have needed one: it `UseResFile`s the application's own fork, removes
`'SIZE'` 0 and 1 (`Environ.c:708-717`), sets both `mem1` and `mem2` of `'SIZE'`
−1 to the requested size (`:719-728`), then `UpdateResFile` + `FlushVol`
(`:730-731`). Its absence is consistent rather than surprising: `SetAppMemorySize`
is **never called** — the only call site is commented out (`Environ.c:688`) — and
both its `Get1Resource` calls are nil-guarded, so nothing would break. Note also
that a real `'SIZE'` −1 would normally live in the *application* resource fork
produced by the linker, not in the `.r` dump of the game's own resources.

### A.6 Window resources

| Res ID | Constant | bounds (t,l,b,r) | size | `procID` | `visible` | `goAway` | `refCon` | title |
|---:|---|---|---|---:|---:|---:|---:|---|
| 128 | `kMainWindowID` (`MainWindow.c:16`) | (0,0,384,512) | 512 × 384 | 2 (`plainDBox`) | 0 | 0 | 0 | `"Main Window"` |
| 129 | `kEditWindowID` (`MainWindow.c:17`) | (0,0,322,512) | 512 × 322 | 4 (`noGrowDocProc`) | 0 | 0 | 0 | `"Main Window"` |
| 130 | `kMenuWindowID` (`MainWindow.c:18`) | (0,0,20,640) | 640 × 20 | 2 (`plainDBox`) | 0 | 0 | 0 | `"New Window"` |

Raw bytes, for reference:

```
WIND 128: 00 00 00 00 01 80 02 00  00 02  00 00  00 00  00 00 00 00  0B "Main Window"
WIND 129: 00 00 00 00 01 42 02 00  00 04  00 00  00 00  00 00 00 00  0B "Main Window"
WIND 130: 00 00 00 00 00 14 02 80  00 02  00 00  00 00  00 00 00 00  0A "New Window"
```

Observations:

1. **All three are `visible = 0`.** Every one is explicitly shown by code:
   `ShowWindow` at `MainWindow.c:202` (edit), `:220` (menu-bar cover), `:230`
   (main). A Go port creating windows visible-by-default will flash uninitialised
   content.
2. The resource bounds are placeholders. `OpenMainWindow` immediately
   `SizeWindow`s / `MoveWindow`s the main window to the screen dimensions
   (`MainWindow.c:174-261`), and 129's 512 × 322 is the *editor's* room-sized
   window.
3. `procID` 2 = `plainDBox` = a borderless 1-pixel-frame box with no title bar,
   which is how a 1994 Mac game got a full-screen look. 4 = `noGrowDocProc` = a
   title bar with a close box but no grow box.
4. `WIND 130`'s 640 × 20 is the strip that *covers the real menu bar* during
   play. 20 == `kScoreboardTall` == the classic Mac menu-bar height. Its width is
   hard-wired to 640; on a wider screen the code re-sizes it.

### A.7 Version resources

| Res ID | BCD version | stage | non-release rev | region | short | long |
|---:|---|---|---:|---:|---|---|
| `'vers' 1` | `0x0112` = **1.1.2** | `0x80` (final/release) | 0 | 0 (US) | `"1.1.2"` | `"Glider PRO™ 1.1.2\r© 1994-95 Casady & Greene, Inc."` |
| `'vers' 2` | `0x0112` = **1.1.2** | `0x80` | 0 | 0 | `"1.1.2"` | `"© 1994-95 Casady & Greene, Inc."` |

Raw: `01 12 80 00 00 00 05 "1.1.2" …`. The stage byte `0x80` is
`finalStage` (development `0x20`, alpha `0x40`, beta `0x60`, final `0x80`).

**This contradicts the source.** Every `.c` file's header comment names the
program differently from the resource fork; `Sources/Main.c:3` heads the file
"Glider PRO 1.0.4". The resource fork in this tree is from **1.1.2**. See Open
questions — a porter must decide which version's behaviour is authoritative, and
the answer is "the source, because that is all we have executable semantics for".

---

## Open questions

Things this document could not settle from the source in front of it. Each is
flagged with what evidence exists and what would settle it.

1. **Which version is this?** `Sources/Main.c:3` heads the file "Glider PRO 1.0.4",
   but both `'vers'` resources in `Glider PRO.r` decode to **1.1.2** with
   copyright "© 1994-95 Casady & Greene, Inc." (§A.7). The resource fork and the
   source tree are from different builds. Since only the source has executable
   semantics, this document treats the `.c` files as authoritative and the
   resources as data that happens to be one point release newer. A porter should
   assume any resource ID referenced by the source exists in the fork (all the
   ones checked here do) but should not assume the reverse.

2. ~~**Where did the `'SIZE'` resource go?**~~ *Resolved.* `SetAppMemorySize`
   (`Environ.c:699-733`) would rewrite `'SIZE'` −1's preferred/minimum partition
   sizes, and there is no `'SIZE'` resource in the dump (§A.5) — but the function
   is dead code: its only call site is commented out (`Environ.c:688`), so the
   path never runs. A `'SIZE'` −1 would in any case be linker output in the
   application fork rather than part of the game's own `.r` resources. A Go port
   has no analogue.

3. **Why is `kMaxViewHeight` 1026 and not 966?** `kMaxViewWidth` 1536
   (`GliderDefines.h:267`) is exactly 3 × `kRoomWide` 512, which matches the 3×3
   neighbourhood. But `kMaxViewHeight` 1026 (`:268`) is *not* 3 × `kTileHigh`
   322 = 966; it is 966 + 60. `houseRect` is clamped to both
   (`InterfaceInit.c:198-201`). The extra 60 rows are unexplained; possibly
   headroom for the scoreboard plus slop, possibly a leftover from when
   `kVertLocalOffset` was 283 or 295 (the comment at `GliderDefines.h:501`
   records both older values). A port should just use 1026 verbatim.

4. ~~**Is the splash vertical centring off by 20 px?**~~ *Resolved: no.*
   `OpenMainWindow` (`MainWindow.c:241-243`) computes `splashOriginV` by
   subtracting **480** from the screen height, and `PICT 1000` is verifiably
   **460** tall (§A.3) — but `splashOriginV` is used to offset the splash rect
   inside `workSrcMap`, not the screen. `workSrcRect` is `houseRect` zeroed
   (`StructuresInit2.c:156`) and `houseRect` is the screen with
   `kScoreboardTall` = 20 subtracted (`InterfaceInit.c:196-197`), so
   `(screenHigh − 480)/2 == ((screenHigh − 20) − 460)/2`, the exact centred
   offset. The 480 is 460 + 20. No compensation in `DrawOnSplash` is needed.

5. **`macEnviron.canColor` is dead.** It is declared (`Environ.h:28`) and never
   assigned or read anywhere in the 67 `.c` files. `hasColor` does the job. Omit
   it from a port.

6. **`macEnviron` has no `menuHigh` field**, yet four sites reference
   `thisMac.menuHigh` (`MainWindow.c:378`, `:388`, `:432`, `:524`). All four are
   inside block-commented-out functions (`ShowMenuBarOld`, `HideMenuBarOld`,
   `HardDrawMainWindow`). The live code hardcodes **20** with `// thisMac.menuHigh`
   as an apologetic comment (`MainWindow.c:224`, `:229`, `Play.c:144`). So the menu
   bar height is a compile-time 20 and the field was removed from the struct at
   some point. Anyone porting must not resurrect it.

7. **`doZooms` is a live preference wired to nothing.** It is read from prefs
   (`Main.c:81`), defaulted true (`Main.c:152`), written back (`Main.c:238`), and
   exposed in the Preferences dialog (`Settings.c:242`, `:272`, `:1226`), but the
   only function it could gate — `ZoomBetweenWindows` (`MainWindow.c:279`) — is
   **entirely inside a block comment** and is never called. Its prototype is still
   live at `GliderProtos.h:152`. So the checkbox does nothing in this build.

8. **`kDisplay9Inch` (1), `kDisplay12Inch` (2), `kDisplay13Inch` (3)** are defined
   at `Environ.c:26-28` and referenced nowhere. A display-class classifier was
   planned and removed.

9. **`isUseSecondScreen` is a live preference wired to nothing.** Read at
   `Main.c:115`, forced false at `:117` and `:185`, written back at `:270`,
   exposed in the Settings dialog (`Settings.c:1128-1141`) — but the only site
   that would act on it is **commented out** (`Main.c:318`:
   `// if ((thisMac.numScreens > 1) && (isUseSecondScreen))`). Multi-monitor play
   was cut.

10. **The `idleMode` state machine is vestigial.** `idleMode` (`Events.c:30`) and
    the constants `kIdleSplashMode` 0 / `kIdleDemoMode` 1 / `kIdleLastMode` 1
    (`GliderDefines.h:195-198`) describe a rotating attract sequence that does not
    exist: `idleMode` is set to `kIdleSplashMode` in `VariableInit` and never
    changes. `kIdleSplashTicks` 7200L (2 minutes at 60 Hz) is likewise unused —
    the real attract timer is `incrementModeTime` compared against `TickCount()`
    (§8.1), seeded with a different interval. A port should implement only what
    `HandleIdleTask`/`HandleEvent` actually do.

11. **Do the padding bytes matter?** `demoType.padding` holds 109 distinct garbage
    values across 1117 records (§A.1) and `prefsInfo` has pad bytes at offsets 145
    and 225 (§A.2). Neither is read. A port only needs to preserve them if it
    intends to byte-compare a re-serialised resource against the original.

12. **What terminates demo playback?** Nothing in `GetDemoInput` bounds-checks
    `demoIndex` against 1116 (§8.5, §A.1). Playback of the recorded stream ends
    only because `DoDemoGame`'s outer game ends (the glider runs out of lives, or
    a key/click aborts). Whether the original ever read past the resource in
    practice depends on how long the demo game survives; 3414 frames of input is
    113.8 s of play. A port must clamp.

13. **`hotObject.doScrutinize` and `objDataType.hotNum`/`dynaNum` semantics.**
    These are in the structs (`GliderStructs.h:224`, `:329-330`) but their exact
    invariants live in `Interactions.c` and `ObjectRects.c`, outside this
    document's scope. Flagged so a porter knows to look there rather than infer.

14. **`game2Type` on-disk size.** The header says 114 but `sizeof` is 110 (§A.2).
    Which one the original writer actually wrote to disk depends on whether
    `SavedGames.c` used `sizeof(game2Type)` or the literal. That file is outside
    this document's scope; a porter must check `SavedGames.c:20` onwards before
    trusting either number.

---

## Porting notes

Concrete, ordered advice for the Go implementation, with the fidelity risk each
item guards against.

### The shape of the program

1. **Do not port the call stack as a call stack.** In the original, "playing a
   game" is a function seven levels deep: `HandleEvent` → `HandleKeyEvent` →
   `DoMenuChoice` → `DoGameMenu` → `NewGame` → `PlayGame` → `RenderFrame`
   (§6, §14). `theMode` is a *description* of where the program counter is, not a
   dispatcher. In Go, invert this: make `theMode` (or a Go `Mode` type) the actual
   switch in a single top-level loop, and convert `NewGame`/`PlayGame` into
   enter/step/exit methods on a `playState`. Every nested blocking loop in the
   original (`PlayGame`, `DoPause`, `DoDiedGameOver`, `WaitForInputEvent`, the
   `switchedOut` freeze, the game-over animations, the modal dialogs) becomes a
   sub-state, not a `for` loop that pumps events.

2. **Keep `theMode`'s three values exactly**: `kSplashMode` 0, `kEditMode` 1,
   `kPlayMode` 2 (`GliderDefines.h:191-193`). They are compared numerically in
   many places and one of them (`Play.c:83` setting `kPlayMode`, `Play.c:233`
   restoring `kSplashMode`) brackets the entire game.

3. **`Sources/Modes.c` is not the app state machine.** It is the *glider*
   animation-mode machine (24 `kGlider…` modes, `GliderDefines.h:571-594`). Two
   unrelated state machines with confusable names.

### Timing — the single highest-risk area

4. **The game is tick-locked at exactly 30 Hz with no catch-up.**
   `RenderFrame` busy-waits `while (TickCount() < nextFrame) {}` then sets
   `nextFrame = TickCount() + kTicksPerFrame` (`Render.c:662-665`), with
   `kTicksPerFrame` = 2 (`GliderDefines.h:533`) and `TickCount()` at 60.15 Hz.
   Because `nextFrame` is recomputed from *now* rather than incremented, a frame
   that overruns permanently loses that time — the game silently slows down
   instead of dropping frames. **Reproduce the "recompute from now" behaviour, not
   an accumulator**, or physics will diverge from the original on any machine that
   ever stutters. In Go: `next = time.Now().Add(2 * tickDuration)` after the wait,
   never `next = next.Add(...)`.

5. **Do not busy-wait in Go.** Use a timer/sleep. But keep the *semantics*: sleep
   until `next`, then recompute `next` from the post-sleep clock.

6. **`kTicksPerFrame` is not only a pacing constant — it is in the physics.**
   Six sites in `Dynamics3.c` (lines 316, 401, 426, 456, 504, 521) compute
   `(delay * 6) / kTicksPerFrame` to turn an author-authored delay into a
   per-frame counter or velocity. If a port renders at 60 Hz it must still
   evaluate these with the literal 2, or every drip, ball, dart, balloon and
   copter changes speed.

7. **`evenFrame` is a mutable global, not `gameFrame & 1`.** It is initialised
   false in `VariableInit` (`InterfaceInit.c:131`), toggled once per frame in
   `PlayGame` (`Play.c:435`), and **force-set to `true`** in three gameplay
   sites: `Dynamics2.c:420` (a drip starts moving) and `Dynamics3.c:474`, `:524`
   (a drip or ball is added). It gates enemy animation and drip acceleration
   (`Dynamics2.c:38`, `:77`, `:409`, `:433`, `:471`). So spawning one object
   re-phases every other object. A port that computes parity from the frame counter
   will desynchronise from the recorded demo. Model it as a real `bool` field.

8. **`evenFrame` is never reset by `NewGame`.** Its parity at the start of game
   *n+1* depends on how many frames game *n* ran. If a port wants deterministic
   demo playback it must decide whether to reproduce this (bug-compatible) or
   reset it (clean) — and know that choosing "reset" changes attract-mode output.

9. **`gameFrame` is a `long` reset to 0 in `NewGame` (`Play.c:112`) and
   incremented at `Play.c:434`, before input is read.** The demo stream is indexed
   by it, so off-by-one here shifts the entire recorded demo by one frame.

10. **The game-over animations run their own independent loops** with
    `nextLoop = TickCount() + 2`, not `RenderFrame` (§6.7). Same 30 Hz, separate
    clock. They also do not advance `gameFrame`.

### Input

11. **Gameplay input is polled, not evented.** `GetInput` (`Input.c:281-379`) calls
    `GetKeys(theKeys)` and tests bits with `BitTst` against `KeyMap` offsets
    (`Externs.h:129-163`), e.g. `kUpArrowKeyMap` 121, `kDownArrowKeyMap` 122,
    `kRightArrowKeyMap` 123, `kLeftArrowKeyMap` 124, `kQKeyMap` 11,
    `kCommandKeyMap` 48, `kSpaceBarMap` 54, `kTabKeyMap` 55, `kEscKeyMap` 50,
    `kDeleteKeyMap` 52. Menu and UI input is evented via `WaitNextEvent`. **A Go
    port must keep both models**: a per-frame key-state snapshot for gameplay
    (physics reads held-state, not transitions) and an event queue for the shell.
    Using only events for gameplay will break "hold to thrust".

12. **`gliderType.leftKey/rightKey/battKey/bandKey` are `long`s holding packed
    `KeyMap` bit positions** (`GliderStructs.h:205-206`), remappable via
    Preferences. They are not ASCII. A port needs an indirection layer from
    logical action → physical key.

13. **`BUILD_ARCADE_VERSION` is 1** (`GliderDefines.h:16`), which remaps the arrow
    keys in `HandleKeyEvent` (`Events.c:191-207`). The retail behaviour differs.
    Pick one and document it; do not silently ship the arcade mapping as "the"
    behaviour.

14. **`WaitCommandQReleased` (`Utilities.c:485-499`) exists because Cmd-Q is both
    a menu equivalent and a polled key** (`Input.c:286-287` calls `DoCommandKey`,
    `Input.c:53-73`, which tests `kQKeyMap` at `Input.c:55`). Without an
    equivalent latch, a Go port will quit twice or quit-then-immediately-act.

15. **Pause is a nested blocking loop** (`DoPause`, `Input.c:77-117`) that
    overlays `PICT 1015` (Esc) or `PICT 1016` (Tab), both 214 × 54 (§A.3),
    selected by the `isEscPauseKey` pref. It does *not* pump the game; it only
    waits for the key to be pressed again. Convert to a state.

### Attract mode

16. **The demo is a one-shot-per-frame stream, not a toggle stream and not a
    held-state stream.** `GetDemoInput` clears `heldLeft`, `heldRight` and
    `tipped` at the top of *every* frame (`Input.c:220-222`) and then applies at
    most one record, only when `gameFrame == demoData[demoIndex].frame`
    (`Input.c:224`). A record therefore means "this key was down on this one
    frame" — nothing is latched across frames (§A.1). Keys observed: 0 (910×),
    1 (198×), 3 (9×). Battery (2) never occurs. Getting this wrong turns the demo
    into noise.

17. **The comments in `GetDemoInput` are wrong.** `case 0` is commented
    `// left key` but its body performs the **right**-key action
    (`hDesiredVel += kNormalThrust`, `heldRight = true`, `Input.c:228-233`), and
    `case 1` is commented `// right key` but performs the **left**-key action
    (`Input.c:235-240`). The recorder confirms which is which:
    `LogDemoKey(0)` sits in `GetInput`'s **rightKey** branch (`Input.c:301-304`)
    and `LogDemoKey(1)` in its **leftKey** branch (`Input.c:318-321`). Trust the
    code, not the comments.

18. **One record per frame, maximum.** `GetDemoInput` consumes at most one record
    per call, so if two records ever shared a frame number the second would be
    delayed. In the shipped stream frame numbers are strictly increasing (§A.1) so
    this never bites, but a port that re-records must maintain the invariant.

19. **Clamp `demoIndex`.** There is no bound check against the 1117-record array
    (§8.5). Go will panic where C read garbage.

20. **Attract mode depends on a house literally named "Demo House".**
    `DoDirSearch` sets `demoHouseIndex` by comparing filenames against
    `"\pDemo House"` (`SelectHouse.c:639`, inside `SelectHouse.c:636-644`), and
    `DoDemoGame` (`Play.c:282-303`)
    uses it **without checking `demoHouseIndex >= 0`**. Ship the file or guard the
    index.

21. **The attract timer is `incrementModeTime`,** a `TickCount()` deadline
    compared in `HandleEvent` (`Events.c:536`) under
    `(theMode == kSplashMode) && doAutoDemo && !switchedOut`. It is rewritten at
    nine sites (§8.2); miss one and the demo fires at the wrong time or never.

### Rendering

22. **Three buffers, two dirty-rect lists.** `backSrcMap` (static room art) →
    `workSrcMap` (composited frame) → the window. `back2WorkRects` erases,
    `work2MainRects` presents (§7.1-§7.2). Both arrays are
    `Rect[kMaxGarbageRects]` with `kMaxGarbageRects` = 48 (`Render.c:20`), but the
    add functions bound with `< kMaxGarbageRects - 1` (`Render.c:67`, `:86`, `:105`), so
    **the usable capacity is 47**. Overflowing silently drops the rect, which
    leaves visual garbage rather than crashing. Reproduce the cap if you want
    bug-for-bug rendering; otherwise use a growable slice and know you have
    diverged.

23. **`RenderFrame` order is load-bearing** (`Render.c:639-671`): erase from back,
    draw dynamics, draw the two gliders, draw effects (flames on odd frames,
    stars on even — gated by `evenFrame` at `:649`), then `CopyRectsQD`, then the
    tick wait. Reordering changes what occludes what.

24. **The mirror region is a QuickDraw `RgnHandle` accumulated across the frame**
    (`AddToMirrorRegion`, `Render.c:740-761`). Go has no region primitive; a
    port needs either a rect list with union semantics or a stencil.

25. **Everything is 8-bit indexed colour.** `kPreferredDepth` 8 (`Externs.h:15`).
    Palette indices appear as bare literals in the code (e.g. `kRedOrangeColor8`
    23, `GliderDefines.h:542`, whose comment admits "actually, 18"). A port that
    goes RGBA must build a 256-entry palette from the `'clut'` resources (two are
    present, IDs 128 and 129, 2056 bytes each = 8-byte header + 256 × 8-byte
    `ColorSpec`) and map literals through it.

26. **`picSize` in a PICT is a 16-bit field that overflows.** PICT 1000 is 108,482
    bytes with a `picSize` of 42,946 (§A.3). Ignore the field; use the resource
    length.

27. **PICT 1989 is a version-1 PICT; the other 17 checked are version 2** (§A.3).
    A decoder must handle 1-byte opcodes and the absence of the `0x0C00` header
    opcode, or special-case that one mask.

### Geometry

28. **A room is 512 × 322** (`kRoomWide` 512, `kTileHigh` 322) built from 8 tiles
    of 64 px (`kNumTiles` 8, `kTileWide` 64) — verified against PICT 2000/2017
    being exactly 512 × 322 (§A.3).

29. **The visible world is the 3×3 neighbourhood** at ±512 horizontally and
    ±`kVertLocalOffset` 322 vertically, indices `kCentralRoom` 0 …
    `kNorthWestRoom` 8 (`GliderDefines.h:217-225`), laid out in `localRoomsDest[9]`
    (§9.4). `numNeighbors` is 9 unless the screen is ≤ 512 wide, in which case
    `Main.c:191-192` forces it to 1.

30. **`houseRect` is clamped to `kMaxViewWidth` 1536 × `kMaxViewHeight` 1026**
    (`InterfaceInit.c:198-201`). Note 1026 ≠ 3 × 322 (Open question 3).

31. **The scoreboard is a 20-pixel band** (`kScoreboardTall` 20) sourced from
    PICT 1997, verified 1536 × 20 (§A.3). `Play.c:530` and `Play.c:581` offset by
    **−576**, which
    is `64 − 640` — a magic number, not a constant.

32. **All three `WIND` resources are `visible = 0`** and are explicitly shown
    (`MainWindow.c:202`, `:220`, `:230`). Their resource bounds are placeholders;
    the real size comes from `thisMac.screen` (§A.6).

33. **Single display is assumed.** `HowManyUsableScreens` exists
    (`Environ.c:308-328`) but its filters are commented out, and the second-screen
    pref is dead (Open question 9). Target one window.

### Data formats

34. **All on-disk data is big-endian with 2-byte struct alignment.** `prefsInfo`
    is wrapped in `#pragma options align=mac68k` / `align=reset`
    (`Externs.h:231`, `:269`). The consequence is visible in `game2Type`, where a
    4-byte `timeStamp` sits at offset **74** (§A.2) — a 4-byte-aligned reader gets
    every subsequent field wrong. In Go, read with explicit
    `binary.BigEndian.Uint16/32` at hardcoded offsets; never `binary.Read` into a
    struct and hope.

35. **Use the verified offsets**, not the header comments: `houseType` rooms start
    at **866**, `roomType` objects at **60** with `sizeof(roomType)` **348**,
    `game2Type` saved rooms at **110** (not 114), `objectType` **12**,
    `scoresType` **292**, `gameType` **40**, `prefsInfo` **226** with pads at 145
    and 225 (§A.2).

36. **Pascal strings everywhere.** `Str15` = 16 bytes, `Str27` = 28, `Str31` = 32,
    `Str32` = 33, `Str255` = 256 — length byte first, fixed-size field, trailing
    bytes are garbage. `roomType.name` is `Str27` (28 bytes at offset 0).
    Text is **Mac Roman**, not UTF-8: 0xA9 is `©`, 0x14 is the Apple glyph (the
    Apple menu title, §A.4).

37. **Resource IDs are the API.** There is no filesystem layout to port; art,
    sound, dialogs and the demo all live in the resource fork by numeric ID. The
    architecture layer alone depends on `PICT` 1000/1015/1016/1019/1021/1988/1989/
    1990/1995/1997/1999/2000-2017/3963/3974/3976/3999/4018, `MENU` 128-131,
    `WIND` 128-130, `STR#` 140/150/160/170/171/1005/1006/1007, `demo` 128,
    `acur` 128, `clut` 128/129, `ALRT` 1031/1042/160, and 70 `snd ` resources
    (§A.3-§A.6). Extract them to files and keep the ID as the key.

### Behaviour that will surprise a porter

38. **`doBackground` defaults to `false`** (`Main.c`'s prefs defaults), and when it
    is false `PlayGame` **does not process events at all** — no
    `WaitNextEvent`, no menu, no update events for the duration of the game
    (§6.2). The only way out is a polled key. A Go port on a modern OS cannot stop
    servicing the window system, so it must pump events but *discard* them when
    `doBackground` is false, otherwise clicks and keys that the original ignored
    will suddenly act.

39. **Suspend/resume freezes the game in a spin.** `HandlePlayEvent`
    (`Play.c:408-424`) sets `switchedOut` on an `osEvt` (`Play.c:412` resume, `:420` suspend), and `PlayGame`'s
    `do { … } while (switchedOut)` (`Play.c:442`) blocks the frame loop until the
    app is foregrounded. `Utilities.c:66` clears it at startup. Model this as a
    paused state, and remember `nextFrame` will be far in the past on resume — so
    the first frame after resume runs immediately (no catch-up, per note 4).

40. **Quitting via `ExitToShell` skips the prefs write.** The normal path is
    `quitting = true` → fall out of `main`'s loop → `WriteOutPrefs`
    (`Main.c:208-279`). `RedAlert` (`Utilities.c:135-161`) calls `ExitToShell`
    directly, so a fatal error loses preference changes. Preserve or fix
    deliberately.

41. **`mortals < 0` means game over, not `== 0`** (`Play.c:546`). `kInitialGliders`
    is 2 (`Play.c:19`) so a normal game has 2 spare gliders plus the one in play.

42. **`batteryTotal` encodes helium as negative values.** One `short` carries two
    resources with opposite signs; `DoBatteryEngaged`/`DoHeliumEngaged`
    (`Input.c:121-182`) branch on the sign. Do not split it into two fields
    without auditing every read.

43. **`countDown` starts at `kNumCountDownFrames` 16** (`GameOver.c:17`) and the
    game keeps simulating while it drains, so the last 16 frames after death are
    still live (§6.7).

44. **`RandomLongQUS` is a specific LCG**: `theSeed = theSeed * 1103515245 + 12345`
    (`Utilities.c:126`), 32-bit unsigned wraparound, seeded from the clock. To
    reproduce demo playback or any recorded sequence bit-exactly, reimplement this
    exact generator with `uint32` arithmetic — not `math/rand`.

45. **The editor is 23.4% of the code and fully separable** (§17.1). Skip it for a
    first playable port.

46. **`Validate.c` (copy protection) is dead** because `COMPILENOCP` is defined
    (`GliderDefines.h:14`). Do not port it. Likewise `COMPILEDEMO`,
    `CREATEDEMODATA` and `CAREFULDEBUG` are all commented out
    (`GliderDefines.h:11-13`), so every `#ifdef` guarded by them is dead, and
    `COMPILEQT` (`:15`) is on, so the QuickTime TV code *is* live.

47. **Nine `macEnviron` fields are hardcoded, not probed.** `CheckOurEnvirons`
    (`Environ.c:438-470`) assigns `vRefNum`, `dirID`, `hasGestalt`, `hasWNE`,
    `hasColor`, `canSwitch`, `hasSystem7`, `hasSM3`, `can1Bit`, `can4Bit`,
    `can8Bit` as constants with `// TEMP` comments. Only `hasQT`, `hasDrag`,
    `numScreens`, `screen`, `wasDepth`, `wasColorOrGray`, `gray` and
    `thisResFile` are real. In a Go port the whole struct collapses to a couple of
    display facts.
