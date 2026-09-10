# Glider PRO 1.0.4 — UI Shell: Menus, Dialogs, Preferences, House Selection, Saved Games

## Scope

This document specifies the complete user-facing shell of Glider PRO 1.0.4 (John Calhoun /
Casady & Greene, GPLv2 source release) with enough precision to re-implement it in Go:

* the menu bar (4 menus, 34 live items) and every enable/disable/check/rename rule;
* the dialog toolkit (`DialogUtils.c`) and which of its helpers are live vs. commented out;
* every `DLOG` (28) and `ALRT` (26) resource with byte-verified bounds, and full item-level
  (`DITL`) dumps for every dialog in the UI shell;
* the five Preferences panes (main, Display, Sound, Controls, Brains), every option, every
  keyboard shortcut in every modal filter, and all three (mutually inconsistent) sets of
  defaults;
* the on-disk preferences record: exact 226-byte `mac68k`-packed layout, field offsets verified
  by compiling the real struct, storage location, file type/creator, and version constant;
* the house-selection browser: directory search, sorting, paging, type-select, double-click;
* saved games: the in-house `gameType` blob (verified against two real house files), the dead
  external `game2Type` / `'gliG'` format, and proof that the whole feature is inert in 1.0.4;
* high scores: the `scoresType` record (verified against real data), the score screen geometry,
  the name/banner entry dialogs, and the `'gliS'` per-house score file;
* the splash screen, the house banner sequence, and the About box;
* the error-reporting UI (`RedAlert`, `YellowAlert`, `CheckFileError`) and all its string tables;
* all 10 `STR#` string-list resources dumped in full (280 strings), with their consumers;
* a complete inventory of the UI that exists in the source and resource fork but cannot be
  reached in the shipped build, plus the 20 user-visible bugs a literal port would inherit.

Out of scope: gameplay/physics, the room editor's own dialogs (Room Info, Object Info, the
per-object-class dialogs `DLOG` 1007/1010/1011/1013/1014/1015/1019/1022/1027/1033/1034/1035/1045),
sound synthesis, and the music player. Those are named where the shell touches them.

Every non-obvious claim carries a citation. C citations are `GliderPRO/Sources/File.c:N` where
`N` is the line number **after** converting the classic-Mac CR line endings to LF
(`tr '\r' '\n'`); the conversion is 1:1 so line numbers match the original file exactly.
Resource citations are `GliderPRO/Glider PRO.r:N` (that file already uses LF).

## Sources read

Read in full (CR→LF converted copies):

| File | Lines | Why |
|---|---|---|
| `GliderPRO/Sources/Menu.c` | 813 | menu bar, all enable rules, all menu handlers |
| `GliderPRO/Sources/DialogUtils.c` | 806 | the dialog helper layer |
| `GliderPRO/Sources/Settings.c` | 1482 | all five Preferences panes |
| `GliderPRO/Sources/About.c` | 257 | About box |
| `GliderPRO/Sources/SelectHouse.c` | 676 | house browser + directory search |
| `GliderPRO/Sources/Prefs.c` | 281 | prefs file read/write/delete |
| `GliderPRO/Sources/SavedGames.c` | 352 | saved-game read/write (mostly dead) |
| `GliderPRO/Sources/Banner.c` | 237 | house banner / stars-remaining |
| `GliderPRO/Sources/HighScores.c` | 856 | score screen, name entry, score files |
| `GliderPRO/Sources/FileError.c` | 102 | file-error alert |
| `GliderPRO/Sources/Main.c` | 386 | `ReadInPrefs` / `WriteOutPrefs` / `main` |
| `GliderPRO/Sources/MainWindow.c` | 299 (of 299 relevant) | splash geometry, window creation |
| `GliderPRO/Sources/InterfaceInit.c` | 220 | `InitializeMenus` |
| `GliderPRO/Headers/Externs.h` | 394 | `prefsInfo`, menu item indices, error codes |
| `GliderPRO/Headers/GliderDefines.h` | 626 | all mode/sound/yellow-alert constants |
| `GliderPRO/Headers/GliderStructs.h` | 348 | `scoresType`, `gameType`, `game2Type`, `houseType` |
| `GliderPRO/Headers/Environ.h` | 38 | `macEnviron thisMac` |

Read in the parts the shell depends on: `HouseIO.c` (open/close/read/write house,
`QuerySaveChanges`, `YellowAlert`, `IsFileReadOnly`), `House.c` (new house, `ZeroHighScores`
call site), `HouseInfo.c` (lock/zero-scores alerts), `Events.c` (event loop, command keys,
auto-demo), `Input.c` (Cmd-Q / Cmd-S / pause), `Play.c` (`NewGame`, `DoDemoGame`, `InitGlider`),
`GameOver.c` (`TestHighScore` call sites), `Utilities.c` (`RedAlert`, `LoadGraphic`,
`WaitForInputEvent`, `GetKeyName`, `UnivGet/SetSoundVolume`), `StringUtils.c`
(`WhichStringFirst`, `CollapseStringToWidth`, `GetLocalizedString`), `Environ.c` (depth
switching, memory alerts), `Sound.c`, `AppleEvents.c`, and — for the `STR#` consumer audit in §12 —
`Tools.c`, `Scrap.c`, `ObjectInfo.c`, `ObjectDraw2.c`, `HouseLegal.c`, `StructuresInit2.c`.

Binary evidence:

* `GliderPRO/Glider PRO.r` — 15 MB Rez dump of the entire resource fork, 538 resources in
  33 types. Parsed with a purpose-built Python `data '...' (id, "name")` parser; every
  `MENU`/`DLOG`/`DITL`/`ALRT`/`STR#`/`PICT` figure below was decoded from the hex bytes, not
  from the Rez comments.
* `GliderPRO/Houses/Empty House.binhex` and `GliderPRO/Houses/Slumberland.binhex` — decoded with
  a hand-written BinHex 4.0 decoder; the resulting data forks (13,046 and 134,150 bytes) were
  parsed field-by-field to confirm the `houseType`/`scoresType`/`gameType` on-disk layout.
* `prefsInfo` and the house structs were compiled with `gcc -c` under `#pragma pack(2)` (the
  gcc equivalent of `#pragma options align=mac68k`) with `long` forced to `int32_t`, and their
  `sizeof`/`offsetof` printed. Observed values are quoted inline.

---

## 1. Conventions and the Mac Toolbox surface a Go port must replace

### 1.1 Pascal strings

Every user-visible string in this subsystem is a Pascal string: a length byte followed by that
many bytes of MacRoman text. The fixed-size typedefs used here are:

| Type | Bytes | Max chars |
|---|---|---|
| `Str15` | 16 | 15 |
| `Str27` | 28 | 27 |
| `Str31` | 32 | 31 |
| `Str32` | 33 | 32 |
| `Str63` | 64 | 63 |
| `Str255` | 256 | 255 |

`Str32` is 33 bytes, which is odd; under `mac68k` alignment the struct that contains it gets a
pad byte inserted after it (see §6.1). A Go port must keep these as fixed-size byte arrays,
not `string`, because they are written verbatim to disk.

MacRoman bytes that actually occur in Glider PRO's UI strings:

| Byte | Glyph | Where |
|---|---|---|
| `0x14` | Apple logo | `MENU` 128's title is this single byte (`GliderPRO/Glider PRO.r:5291`) |
| `0xA9` | `©` | About box, `vers` resources |
| `0xAA` | `™` | "Glider PRO™" throughout |
| `0xC9` | `…` | every menu item that opens a dialog; `CollapseStringToWidth` ellipsis |
| `0xC4` | `ƒ` | the scores folder name is literally `"G-PRO Scores \xC4"` — byte-verified with `cat -v` on `GliderPRO/Sources/HighScores.c:654`; `ƒ` was the classic-Mac folder-name suffix convention |
| `0xC7`/`0xC8` | `«` `»` | high-score screen house title |

Go port: decode MacRoman explicitly. A naive UTF-8 assumption corrupts every one of these.

### 1.2 Structure alignment

`Externs.h` wraps `prefsInfo` in:

```c
#pragma options align=mac68k      // GliderPRO/Headers/Externs.h:231
...
#pragma options align=reset       // GliderPRO/Headers/Externs.h:269
```

`mac68k` alignment means: `short`/`long`/pointers align to 2, not 4; arrays and structs align to
2; the total size is rounded up to a multiple of 2. It is exactly gcc's `#pragma pack(2)`.
`long` is 32 bits. All multi-byte integers on disk are **big-endian**.

Verified: compiling `prefsInfo` with `#pragma pack(2)` and `#define long int32_t` gives
`sizeof(prefsInfo) == 226`, which is precisely the byte count `WritePrefs` transfers
(`GliderPRO/Sources/Prefs.c:127` uses `sizeof(*thePrefs)`).

### 1.3 Resource census

Full type census of `GliderPRO/Glider PRO.r` (538 resources):

| Type | Count | Type | Count | Type | Count |
|---|---|---|---|---|---|
| `ALRT` | 26 | `PICT` | 152 | `icl4` | 6 |
| `BNDL` | 1 | `STR#` | 10 | `icl8` | 6 |
| `CDEF` | 1 | `WDEF` | 2 | `ics#` | 4 |
| `CNTL` | 5 | `WIND` | 3 | `ics4` | 4 |
| `CURS` | 16 | `acur` | 1 | `ics8` | 4 |
| `DITL` | 54 | `cctb` | 1 | `ictb` | 1 |
| `DLGX` | 1 | `cicn` | 44 | `mctb` | 5 |
| `DLOG` | 28 | `clut` | 2 | `ozm5` | 1 |
| `FREF` | 6 | `crsr` | 12 | `snd ` | 70 |
| `ICN#` | 6 | `dctb` | 20 | `vers` | 2 |
| `ICON` | 35 | `demo` | 1 | `wctb` | 1 |
| `MENU` | 6 | `PAT#` | 1 | | |

Only `MENU`, `DLOG`, `DITL`, `ALRT`, `STR#`, `PICT`, `ICON`, `cicn`, `icl8`, `WIND` and `vers`
matter for the UI shell. The `dctb`/`cctb`/`wctb`/`mctb` colour tables and the `ictb` item colour
table are cosmetic Toolbox colour overrides; a Go port can ignore them and hard-code the
theme (§Porting notes).

The `'ozm5'` resource of type `'ozm5'` is the application's own signature resource; `'ozm5'` is
also the creator code of every file the game writes.

### 1.4 On-disk resource layouts (decoded, not guessed)

#### `MENU`

```
offset  size  field
0       2     menuID              (big-endian short)
2       2     menuWidth           (0 = compute at runtime)
4       2     menuHeight          (0 = compute at runtime)
6       2     menuProc            (resource ID of MDEF; 0 = standard)
8       2     filler              (0)
10      4     enableFlags         bit 0 = whole menu enabled,
                                  bit N = item N enabled (N = 1..31)
14      *     menuTitle           Pascal string
then, repeated until a 0x00 length byte:
        *     itemText            Pascal string
        1     iconID              (0 = none; else icon# = value + 256)
        1     cmdKey              (0 = none; ASCII char otherwise)
        1     markChar            (0 = none)
        1     style               (QuickDraw style bits: 1=bold 2=italic ...)
0x00 terminator
```

#### `DITL`

```
offset  size  field
0       2     itemCount - 1       (big-endian short!)
per item:
        4     placeholder (handle/proc slot, 0 on disk)
        8     rect: top, left, bottom, right (big-endian signed shorts)
        1     type byte; high bit (0x80) set = item DISABLED
        1     data length
        n     data, padded to an even total
```

Item type codes actually used by Glider PRO:

| Code | Meaning | Data |
|---|---|---|
| 0 | userItem | none |
| 1 | helpItem | (not used) |
| 4 | button (`ctrlItem+btnCtrl`) | Pascal title |
| 5 | checkBox | Pascal title |
| 6 | radioButton | Pascal title |
| 7 | resCtrl (`CNTL` resource) | 2-byte resource ID |
| 8 | statText | raw text (no length byte; length is the item's data length) |
| 16 | editText | raw text |
| 32 | icon | 2-byte `ICON`/`cicn` resource ID |
| 64 | picture | 2-byte `PICT` resource ID |

#### `DLOG`

21 bytes for all Glider PRO dialogs (the title is always empty):

```
offset  size  field
0       8     boundsRect (top, left, bottom, right) — GLOBAL screen coords
8       2     procID
10      1     visible          (1 = true)
11      1     filler
12      1     goAwayFlag
13      1     filler
14      4     refCon
18      2     itemsID          (DITL resource ID)
20      1     title length (0)
```

`DLOG` 150 (About) is 24 bytes: the extra 3 bytes `00 02 2F` are the optional
positioning/`DLGX` extension (`GliderPRO/Glider PRO.r:6283`). Observed raw bytes for
`DLOG` 1012 (Preferences):

```
00 28 00 28 00 aa 01 48   -> top=40 left=40 bottom=170 right=328
00 01                     -> procID 1  (dBoxProc: modal, no title bar, 2px black frame)
01 00                     -> visible=1
01 00                     -> goAway=1
00 00 00 00               -> refCon 0
03 f4                     -> DITL 1012
00                        -> title ""
```

Every Glider PRO `DLOG` uses `procID == 1` (`dBoxProc`). A Go port should render these as
borderless modal panels with the classic 1-pixel black frame + 3-pixel shadow that `dBoxProc`
draws, or simply as a plain rectangle with the dialog's own `PICT` header art on top.

#### `ALRT`

```
offset  size  field
0       8     boundsRect (global)
8       2     itemsID (DITL)
10      2     stages
```

`stages` is four nibbles, one per alert stage (4th..1st, high nibble = stage 4). Each nibble is
`(sound & 3) | (drawAlert ? 4 : 0) | ((defaultItem == 2) ? 8 : 0)` — i.e. the low 2 bits are the
"beep this many times" sound level (0-3), bit 2 **enables** drawing the alert box (`visible` = 4,
`invisible` = 0 in Rez), and bit 3 makes item 2 the default (`Cancel` = 8, `OK` = 0 in Rez).
The values observed in Glider PRO (all 26 `ALRT`s use one of these four, and every one repeats the
same nibble in all four stages):

| stages | Meaning | Used by |
|---|---|---|
| `0x5555` | draw, 1 beep, default item 1 | 140, 160, 170, 180, 181, 1002, 1004, 1005, 1006, 1008, 1028, 1031, 1042, 1044 |
| `0x4444` | draw, **no** sound, default item 1 | 130, 1009, 1030, 1032, 1036, 1037, 1038, 1039, 1040, 1046 |
| `0xCCCC` | draw, no sound, **default item 2** | 1041 (Save Game?) |
| `0xFFFF` | draw, **3 beeps**, **default item 2** | 1029 (Lock House) |

Go port: alerts are just modal dialogs whose default button is item 1 (or item 2 for 1041/1029)
and which `SysBeep` zero, one or three times on display depending on the low two bits. Every
Glider PRO alert has bit 2 set, so the alert box is always drawn — there is no invisible alert.

#### `STR#`

```
0   2  count
then count Pascal strings back to back
```

### 1.5 The modal dialog / filter-proc model

Every dialog in this subsystem is a *modal* dialog driven by `ModalDialog(filterUPP, &itemHit)`.
`ModalDialog` pulls one event, offers it to the filter proc, and if the filter returns `false`
lets the Dialog Manager handle it (tracking buttons, editing text, etc.), yielding an item
number in `itemHit`. If the filter returns `true`, `*itemHit` is whatever the filter stored.

The universal Glider PRO idioms inside a filter (`Settings.c`, `SelectHouse.c`, `HighScores.c`,
`About.c` all follow this shape):

1. `switch (event->what)`.
2. `case keyDown:` extract `theChar = event->message & charCodeMask` (low byte).
   Return/Enter ⇒ `FlashDialogButton(dial, kOkayButton)`, `*item = kOkayButton`, return `true`.
   Escape ⇒ `FlashDialogButton(dial, kCancelButton)`, `*item = kCancelButton`, return `true`.
   Letter shortcuts ⇒ set `*item` to the target item, return `true` (sometimes after a flash).
3. `case updateEvt:` if `(WindowPtr)event->message == mainWindow` repaint the game window and
   return `true`; if it is `GetDialogWindow(dial)` run the panel's own `Update...` function and
   return `false` so the Dialog Manager still draws its controls.
4. `default:` return `false`.

Go port: this maps cleanly onto a per-dialog `HandleKey(rune) (itemHit int, handled bool)` plus
a `Draw()`; the important fidelity detail is *which* keys are shortcuts in *which* panel, since
they are inconsistent (fully tabulated in §6).

Toolbox calls used and what they mean for a port:

| Toolbox call | Semantics to reproduce |
|---|---|
| `GetNewDialog(id, nil, (WindowPtr)-1L)` | instantiate `DLOG`+`DITL`, insert in front |
| `ModalDialog` | modal event pump, one event per call |
| `Alert(id, nil)` | load `ALRT`, run modally, return item hit |
| `ParamText(a,b,c,d)` | substitute `^0`..`^3` in *all subsequently drawn* static text |
| `GetDialogItem` / `SetDialogItem` | item handle/type/rect access |
| `GetDialogItemText`/`SetDialogItemText` | edit/static text contents (Pascal) |
| `GetControlValue`/`SetControlValue` | checkbox/radio 0 or 1 |
| `HiliteControl(h, 255)` | disabled (grey) control |
| `HiliteControl(h, kControlButtonPart=10)` | pressed-in look |
| `SelectDialogItemText(d,i,0,1024)` | select all text in an edit field |
| `ShowDialogItem`/`HideDialogItem` | show/hide (used for the house-list arrows) |
| `DrawDialog` | redraw all items |
| `InvalWindowRect` | mark dirty; triggers a later `updateEvt` |
| `SysBeep(1)` | error feedback |

`ParamText` is global state, not per-dialog: `QuerySaveChanges` sets it before `Alert`
(`GliderPRO/Sources/HouseIO.c:606`), `GetHighScoreName` sets it before `BringUpDialog`
(`GliderPRO/Sources/HighScores.c:509-512`). A Go port should pass substitutions explicitly and
make the `^0`..`^3` expansion a property of the dialog invocation.
---

## 2. Application shell: modes, startup order, idle behaviour

### 2.1 The three modes

`GliderPRO/Headers/GliderDefines.h:191-193`:

| Constant | Value | Meaning |
|---|---|---|
| `kSplashMode` | 0 | full-screen splash; menus show Game + Options |
| `kEditMode` | 1 | 512×322 editor window; House menu inserted |
| `kPlayMode` | 2 | in game; menu bar covered |

The global is `short theMode` (`GliderPRO/Sources/MainWindow.c:41`). `theMode` drives menu
insertion (§3.5), which window is created (§2.4), and idle behaviour (§2.5).

Idle-mode constants (`GliderPRO/Headers/GliderDefines.h:195-198`):

| Constant | Value |
|---|---|
| `kIdleSplashMode` | 0 |
| `kIdleDemoMode` | 1 |
| `kIdleSplashTicks` | `7200L` (= 120 s at 60 ticks/s) |
| `kIdleLastMode` | 1 |

### 2.2 `main()` startup order

`GliderPRO/Sources/Main.c:284-370`, in order — the ordering matters because `ReadInPrefs` runs
*before* the screen depth is switched and *before* the window exists, but *after*
`CheckOurEnvirons` has filled in `thisMac`:

1. `ToolBoxInit()`
2. `CheckOurEnvirons()` — fills `thisMac` (screen rect, depth capabilities, `numScreens`,
   `hasSystem7`, `hasColor`, `hasSM3`, `hasQT`, `dirID`, `vRefNum`, `thisResFile`).
3. `if (!thisMac.hasColor) RedAlert(kErrNeedColorQD)` → fatal, error 12.
4. `if (!thisMac.hasSystem7) RedAlert(kErrNeedSystem7)` → fatal, error 8.
5. `if (thisMac.numScreens == 0) RedAlert(kErrNeed16Or256Colors)` → fatal, error 13.
6. `SetUpAppleEvents()`
7. `LoadCursors()`
8. **`ReadInPrefs()`** (§6.4)
9. `copyGood = true;` (because `COMPILENOCP` is defined, `GliderPRO/Headers/GliderDefines.h:14`;
   the copy-protection `ValidInstallation()` path is compiled out)
10. `HandleDepthSwitching()` — may show `ALRT` 130 (§11.5)
11. `VariableInit()`
12. `CheckMemorySize()` — may show `ALRT` 180 or 181 (§11.6) and can `ExitToShell`
13. `GetExtraCursors()`, `InitMarquee()`, `CreatePointers()`, `InitSrcRects()`,
    `CreateOffscreens()`
14. **`OpenMainWindow()`** — computes `splashOriginH/V` and loads the splash `PICT` (§2.4)
15. `if (thisMac.hasQT) EnterMovies();`
16. `InitSound()`, `InitMusic()`
17. **`BuildHouseList()`** (§7.4)
18. `if (OpenHouse()) ReadHouse();`
19. `PlayPrioritySound(kBirdSound, kBirdPriority)` — sound 10, priority 804
20. `DelayTicks(6)`
21. **`InitializeMenus()`** (§3.4), `InitCursor()`
22. `while (!quitting) HandleEvent();`
23. `KillMusic()`, `KillSound()`, `CloseHouse()`, **`WriteOutPrefs()`**, `RestoreColorDepth()`,
    `FlushEvents(everyEvent, 0)`

`BitchAboutSM3()` (`ALRT` 1030, "Where is Sound Manager 3.0?") is **commented out** at
`GliderPRO/Sources/Main.c:357-361`, so that alert never appears in 1.0.4.

Note the ordering consequence: the menu bar does not exist until step 21, so the splash is
drawn and a house is opened before any menu can be pulled down. `menusUp` (`Menu.c:33`) guards
every menu-update function against being called before step 21.

### 2.3 Window resources

`WIND` resources (`GliderPRO/Glider PRO.r`, decoded):

| ID | bounds (t,l,b,r) | procID | title | Used for |
|---|---|---|---|---|
| 128 | 0,0,384,512 | 2 | `"Main Window"` | splash/play window (`kMainWindowID`) |
| 129 | 0,0,322,512 | 4 | `"Main Window"` | editor window (`kEditWindowID`) |
| 130 | 0,0,20,640 | 2 | `"New Window"` | menu-bar cover strip (`kMenuWindowID`) |

`GliderPRO/Sources/MainWindow.c:16-18` defines `kMainWindowID 128`, `kEditWindowID 129`,
`kMenuWindowID 130`. `procID` 2 is `plainDBox` (no title bar, plain 1-pixel frame); `procID` 4
is `noGrowDocProc` (title bar, no grow box) — hence the editor window has a drag bar and the
splash window does not.

### 2.4 Splash-screen geometry (`OpenMainWindow`)

`GliderPRO/Sources/MainWindow.c:174-265`:

```
1. if (mainWindow != nil) { YellowAlert(kYellowUnaccounted, 6); return; }   // 176-180
2. if (theMode == kEditMode):
      QSetRect(&mainWindowRect, 0, 0, 512, 322)
      mainWindow = GetNewCWindow(kEditWindowID /*129*/, nil, kPutInFront)
      SizeWindow(mainWindow, 512, 322, true)
      if (OptionKeyDown()) { isEditH = 3; isEditV = 41; }      // recover an off-screen window
      MoveWindow(mainWindow, isEditH, isEditV, true)
      ClipRect(&mainWindowRect)
3. else (kSplashMode or kPlayMode):
      // 20-pixel strip over the menu bar so the game can own the whole screen
      QSetRect(&menuWindowRect, 0, 0, RectWide(&thisMac.screen), 20)
      menuWindow = GetNewCWindow(kMenuWindowID /*130*/, nil, kPutInFront)
      SizeWindow(menuWindow, RectWide(&thisMac.screen), 20, true)
      MoveWindow(menuWindow, thisMac.screen.left, thisMac.screen.top, true)
      mainWindowRect = thisMac.screen
      ZeroRectCorner(&mainWindowRect)        // normalise to (0,0)
      mainWindowRect.bottom -= 20
      mainWindow = GetNewCWindow(kMainWindowID /*128*/, nil, kPutInFront)
      SizeWindow(...)
      MoveWindow(mainWindow, thisMac.screen.left, thisMac.screen.top + 20, true)
      PaintRect(&mainWindowRect)             // black
      splashOriginH = ((thisMac.screen.right - thisMac.screen.left) - 640) / 2
      if (splashOriginH < 0) splashOriginH = 0                     // MainWindow.c:238-240
      splashOriginV = ((thisMac.screen.bottom - thisMac.screen.top) - 480) / 2
      if (splashOriginV < 0) splashOriginV = 0                     // MainWindow.c:241-243
      SetPortWindowPort(mainWindow)
      LoadGraphic(kSplash8BitPICT /*1000*/)  // into workSrcMap
```

`kSplash8BitPICT` is 1000 (`GliderPRO/Headers/GliderDefines.h:524`). Decoded `PICT` 1000 is
**640×460**, so the splash is a 640×460 image centered inside a 640×480 nominal box; the extra
20 px at the bottom is where the scoreboard would go. `PICT` 1021 is a **second** 640×460
full-screen splash present in the resource fork but referenced by no constant in
`GliderDefines.h` — probably the demo/alternate splash.

`RedrawSplashScreen` (`GliderPRO/Sources/MainWindow.c:98-114`):

```
1. SetPortWindowPort(mainWindow); PaintRect over workSrcMap (black)
2. QSetRect(&tempRect, 0, 0, 640, 460)
3. QOffsetRect(&tempRect, splashOriginH, splashOriginV)
4. LoadScaledGraphic(kSplash8BitPICT, &tempRect)
5. DrawOnSplash()
6. CopyRectMainToWork(&workSrcRect)
   // the two DissBits() transition calls are commented out at MainWindow.c:109-112
```

`DrawOnSplash` (`GliderPRO/Sources/MainWindow.c:56-94`) overlays the currently loaded house
name:

```
1. houseLoadedStr = "House: " ; PasStringConcat(houseLoadedStr, thisHouseName)
2. if (thisMac.hasQT && hasMovie) PasStringConcat(houseLoadedStr, " (QT)")
3. TextSize(9); TextFace(1 /*bold*/); TextFont(applFont)
4. MoveTo(splashOriginH + 436, splashOriginV + 314)
5. if (thisMac.isDepth == 4) ForeColor(whiteColor), DrawString
   else if (houseIsReadOnly) ColorText(houseLoadedStr, 5L)
   else ColorText(houseLoadedStr, 28L)
6. #ifdef powerc:  "PowerPC Native!" at (splashOriginH+5, splashOriginV+457) in black,
                   then (+4,+456) in white, systemFont size 12
```

Colour indices 5 and 28 are indices into the 8-bit CLUT, not RGB. `houseIsReadOnly` is always
`false` in 1.0.4 because `IsFileReadOnly` unconditionally `return false;`
(`GliderPRO/Sources/HouseIO.c:659-663`), so the house name is always drawn in colour 28.

`UpdateMainWindow` (`GliderPRO/Sources/MainWindow.c:120-155`):

```
if (theMode == kEditMode):
    PauseMarquee(); CopyBits(workSrcMap -> mainWindow, srcCopy); ResumeMarquee()
else:                                     // splash or play
    PaintRect(black) over workSrcMap
    QSetRect(&tempRect,0,0,640,460); QOffsetRect(splashOriginH, splashOriginV)
    LoadScaledGraphic(kSplash8BitPICT, &tempRect)
    CopyBits(workSrcMap -> mainWindow, srcCopy)
    DrawOnSplash()
splashDrawn = true;                        // MainWindow.c:154
```

`splashDrawn` gates the Load House menu item's post-action repaint (§3.8.3).

`UpdateMenuBarWindow` (`GliderPRO/Sources/MainWindow.c:160-169`) just paints `menuWindow`
black — a kludge so that when the game runs on a second monitor the primary monitor's menu bar
area is covered.

`ZoomBetweenWindows` (`GliderPRO/Sources/MainWindow.c:277-299`) is **entirely commented out**;
so the `doZooms` preference (§6.11) has no effect on window transitions in 1.0.4 — it is stored,
edited, and never read for zooming.

### 2.5 Idle behaviour and the auto-demo

`GliderPRO/Sources/Events.c:479-545`, `HandleEvent`:

```
1. GetKeys(eventKeys)
2. if (Cmd held && Option held) HiliteAllObjects()
   else if (Option held && theMode == kEditMode && houseUnlocked) {
            EraseSelectedTool(); SelectTool(kSelectTool /*0*/); }
3. if (thisMac.hasWNE) WaitNextEvent(everyEvent, &theEvent, 2 /*sleep ticks*/, nil)
   else { SystemTask(); GetNextEvent(everyEvent, &theEvent) }
4. switch (theEvent.what):
      mouseDown  -> HandleMouseEvent
      keyDown / autoKey -> HandleKeyEvent
      updateEvt  -> HandleUpdateEvent
      osEvt      -> HandleOSEvent
      kHighLevelEvent -> AEProcessAppleEvent
   default (no event):  HandleIdleTask()
5. // Events.c:536-540
   if ((theMode == kSplashMode) && (doAutoDemo) && (!switchedOut))
        if (TickCount() >= incrementModeTime) DoDemoGame();
```

So: after 7200 ticks (2 min) of sitting on the splash with `doAutoDemo` on and the app not
switched out, the demo house auto-plays. `incrementModeTime` is reset to
`TickCount() + kIdleSplashTicks` at: the end of `NewGame` (`Play.c`), at the end of `DoDemoGame`
(`Play.c:302`), after Load House / High Scores / Preferences (§3.8), i.e. any user interaction
with the shell postpones the demo by another two minutes.

`DoDemoGame` (`GliderPRO/Sources/Play.c:282-303`):

```
1. wasIndex = thisHouseIndex
2. CloseHouse()
3. thisHouseIndex = demoHouseIndex           // -1 if no "Demo House" file found
4. PasStringCopy(theHousesSpecs[thisHouseIndex].name, thisHouseName)
5. if (OpenHouse()) { ReadHouse(); demoGoing = true; NewGame(kNewGameMode); }
6. CloseHouse()
7. thisHouseIndex = wasIndex
   PasStringCopy(theHousesSpecs[thisHouseIndex].name, thisHouseName)
8. if (OpenHouse()) ReadHouse()
9. incrementModeTime = TickCount() + kIdleSplashTicks
```

`demoHouseIndex` is set by `DoDirSearch` to the index of the file literally named
`"Demo House"`, else `-1` (`GliderPRO/Sources/SelectHouse.c:635-645`). When it is `-1`, the
Options ▸ Demo… menu item is disabled (§3.5.2). The demo is driven by the `demo` 128 resource — a
list of `demoType {long frame; char key; char padding;}` records
(`GliderPRO/Headers/GliderStructs.h:334-339`). `kDemoLength` = 6702
(`GliderPRO/Headers/GliderDefines.h:625`) is the resource's length in **bytes**, not a frame count:
the resource is exactly 6702 bytes, i.e. 1117 six-byte records. It is used purely as a byte count —
`NewPtr(kDemoLength)` and `BlockMove(*tempHandle, demoData, kDemoLength)`
(`GliderPRO/Sources/StructuresInit2.c:287, 295`) — and `Environ.c:657` adds it to the memory
estimate. Playback (`GliderPRO/Sources/Input.c:224-267`) only advances `demoIndex` when
`gameFrame == demoData[demoIndex].frame`; there is **no bounds check on `demoIndex`**, so the demo is
terminated by the glider running out of lives (`demoGoing` is tested at
`GliderPRO/Sources/Play.c:478` and `GliderPRO/Sources/GameOver.c:494`), not by exhausting the array.

### 2.6 `BUILD_ARCADE_VERSION` changes what the arrow keys do

`GliderPRO/Headers/GliderDefines.h:16` defines `BUILD_ARCADE_VERSION 1`, and
`GliderPRO/Sources/Events.c:191-207` is compiled in as a result. `HandleKeyEvent`
(`GliderPRO/Sources/Events.c:162`) runs whenever the app is *not* in the game loop, i.e. in
both `kSplashMode` **and** `kEditMode`, and the arcade branch is not gated on the mode. So in
this build the arrow keys fire menu commands even while the room editor is open:

| Key | Arcade build (this build) | Non-arcade build (`houseUnlocked` only) |
|---|---|---|
| Left arrow | `DoOptionsMenu(iHighScores)` → high-score screen | no object selected: `SelectNeighborRoom(kRoomToLeft)`; else `MoveObject(kBumpLeft, shiftDown)` |
| Right arrow | `DoOptionsMenu(iHelp)` → demo | `SelectNeighborRoom(kRoomToRight)` / `MoveObject(kBumpRight, shiftDown)` |
| Up arrow | `DoGameMenu(iNewGame)` | `SelectNeighborRoom(kRoomAbove)` / `MoveObject(kBumpUp, shiftDown)` |
| Down arrow | `DoGameMenu(iNewGame)` | `SelectNeighborRoom(kRoomBelow)` / `MoveObject(kBumpDown, shiftDown)` |

Both Up *and* Down start a new game (`Events.c:198-206`). This is a real behaviour of 1.0.4,
not a bug in the transcription — an arcade cabinet has a joystick, not a menu bar.

In the non-arcade `#else` branch (`Events.c:209-251`), Delete maps to `DeleteRoom`/`DeleteObject`
and Tab to `SelectPrevObject`/`SelectNextObject`.
---

## 3. The menu bar

### 3.1 Menu resource inventory (byte-decoded from `Glider PRO.r`)

| `MENU` res ID | internal `menuID` | Title bytes | Items | `enableFlags` | Rez line |
|---|---|---|---|---|---|
| 128 | 128 | `\x14` (Apple logo) | 2 | `0xFFFFFFFB` | `GliderPRO/Glider PRO.r:5291` |
| 129 | 129 | `Game` | 7 | `0xFFFFFFAF` | `GliderPRO/Glider PRO.r:5297` |
| 130 | 130 | `Options` | 5 | `0xFFFFFFFB` | `GliderPRO/Glider PRO.r:5307` |
| 131 | 131 | `House` | 21 | `0xFFFADF77` | `GliderPRO/Glider PRO.r:5316` |
| 140 | **133** | `Rooms` | 19 | `0xFFF7FFFF` | `GliderPRO/Glider PRO.r:5339` |
| 141 | **133** | `Tools` | 9 | `0xFFFFFFFF` | `GliderPRO/Glider PRO.r:5360` |

All six have `menuWidth = 0`, `menuHeight = 0`, `menuProc = 0` (standard MDEF), `filler = 0`,
and no item has an icon, mark character, or style.

`MENU` 140 and 141 are the editor's Room-Info background popup and the tools-palette popup;
**both carry internal `menuID = 133`**, which is a resource-authoring artefact. They are pop-up
menus attached to `CNTL` items, not menu-bar menus, so the collision is harmless in the original
(the Menu Manager only cares about `menuID` for menus actually inserted in the bar). A Go port
should not reuse `menuID` as a key.

Decoding `enableFlags` (bit 0 = the whole menu, bit N = item N):

| Menu | Flags | Items disabled *in the resource* |
|---|---|---|
| 128 | `0xFFFFFFFB` | item 2 (the `-` separator) |
| 129 | `0xFFFFFFAF` | items 4 and 6 (both `-` separators) |
| 130 | `0xFFFFFFFB` | item 2 (`-`) |
| 131 | `0xFFFADF77` | items 3, 7, 13, 16, 18 (all `-`) |
| 140 | `0xFFF7FFFF` | item 19 (`Original Artwork`) |
| 141 | `0xFFFFFFFF` | none |

Worked example for `0xFFFADF77`, byte by byte, bit 0 = LSB of the low byte:
`0x77 = 0111_0111` → bits 3 and 7 clear; `0xDF = 1101_1111` → bit 13 clear; `0xFA = 1111_1010` → bits 16 and 18
clear; `0xFF` → bits 24–31 set. Clear bits = {3, 7, 13, 16, 18} = exactly the five `-`
separators in the House menu. Separators are disabled in the resource so they never highlight.
(Similarly `0xFFFFFFAF` clears bits 4 and 6 = Game's two separators; `0xFFFFFFFB` clears bit 2;
`0xFFF7FFFF` clears bit 19.)

### 3.2 Menu ID and item-index constants

`GliderPRO/Headers/GliderDefines.h:186-189`:

| Constant | Value |
|---|---|
| `kAppleMenuID` | 128 |
| `kGameMenuID` | 129 |
| `kOptionsMenuID` | 130 |
| `kHouseMenuID` | 131 |

`GliderPRO/Headers/Externs.h:197-222` — note the item numbers are *not* contiguous, because the
separators occupy indices:

| Constant | Value | Menu |
|---|---|---|
| `iAbout` | 1 | Apple |
| `iNewGame` | 1 | Game |
| `iTwoPlayer` | 2 | Game |
| `iOpenSavedGame` | 3 | Game |
| `iLoadHouse` | 5 | Game |
| `iQuit` | 7 | Game |
| `iEditor` | 1 | Options |
| `iHighScores` | 3 | Options |
| `iPrefs` | 4 | Options |
| `iHelp` | 5 | Options |
| `iNewHouse` | 1 | House |
| `iSave` | 2 | House |
| `iHouse` | 4 | House |
| `iRoom` | 5 | House |
| `iObject` | 6 | House |
| `iCut` | 8 | House |
| `iCopy` | 9 | House |
| `iPaste` | 10 | House |
| `iClear` | 11 | House |
| `iDuplicate` | 12 | House |
| `iBringForward` | 14 | House |
| `iSendBack` | 15 | House |
| `iGoToRoom` | 17 | House |
| `iMapWindow` | 19 | House |
| `iObjectWindow` | 20 | House |
| `iCoordinateWindow` | 21 | House |

### 3.3 The complete menu tree

#### Apple menu (`MENU` 128, title = `0x14`)

| # | Text | ⌘ | Action |
|---|---|---|---|
| 1 | `About Glider PRO…` | — | `DoAbout()` (§10.3) |
| 2 | `-` | — | separator, disabled in resource |

Plus every item that `AppendResMenu(appleMenu, 'DRVR')` appends at startup
(`GliderPRO/Sources/InterfaceInit.c:53`) — i.e. the user's Apple-menu items / desk accessories.
**The handler for those appended items is commented out** (`GliderPRO/Sources/Menu.c:281-296`),
so selecting any of them does nothing. A Go port should simply not append them.

#### Game menu (`MENU` 129, title `Game`)

| # | Text (exact bytes) | ⌘ | Action |
|---|---|---|---|
| 1 | `New Game\x00` — Pascal length 9, **trailing NUL byte** | `N` | `twoPlayerGame=false; resumedSavedGame=false; NewGame(kNewGameMode)` |
| 2 | `Two Player Game` | `2` | `twoPlayerGame=true; resumedSavedGame=false; NewGame(kNewGameMode)` |
| 3 | `Open Saved Game…` | `O` | `resumedSavedGame=true; HeyYourPissingAHighScore(); if (OpenSavedGame()) { twoPlayerGame=false; NewGame(kResumeGameMode); }` |
| 4 | `-` | — | separator |
| 5 | `Load House…` | `L` | `DoLoadHouse()` (§7) |
| 6 | `-` | — | separator |
| 7 | `Quit` | `Q` | `quitting = true; if (!QuerySaveChanges()) quitting = false;` |

The `New Game\x00` text is byte-verified: the Pascal length byte is 9 and the ninth byte is
`0x00`. On a real Mac the Menu Manager draws it as "New Game" (the NUL renders as nothing/zero
width). A Go port should trim it.

#### Options menu (`MENU` 130, title `Options`)

| # | Text | ⌘ | Action |
|---|---|---|---|
| 1 | `Room Editor` | `E` | toggle `kEditMode` ↔ `kSplashMode` (§3.8.4); shows a checkmark while in edit mode |
| 2 | `-` | — | separator |
| 3 | `High Scores…` | `H` | `DoHighScores(); incrementModeTime = TickCount()+kIdleSplashTicks;` |
| 4 | `Preferences…` | `P` | `DoSettingsMain(); incrementModeTime = TickCount()+kIdleSplashTicks;` |
| 5 | `Demo…` | `D` | `DoDemoGame()` |

#### House menu (`MENU` 131, title `House`) — inserted only in `kEditMode`

| # | Text | ⌘ | Action (all gated on `houseUnlocked`) |
|---|---|---|---|
| 1 | `New House…` | `N` | `if (CreateNewHouse()) { ... }` |
| 2 | `Save House` | `S` | deselect, sort, version-convert, `WriteHouse(true)`, re-render room |
| 3 | `-` | — | separator |
| 4 | `House Info…` | — | `DoHouseInfo()` (`DLOG` 1001) — no command key |
| 5 | `Room Info…` | `R` | `DoRoomInfo()` (`DLOG` 1003) |
| 6 | `Object Info…` | `I` | `DoObjectInfo()` (per-class `DLOG`) |
| 7 | `-` | — | separator |
| 8 | `Cut Room` / `Cut Object` | `X` | text swapped at runtime (§3.5.3) |
| 9 | `Copy Room` / `Copy Object` | `C` | text swapped at runtime |
| 10 | `Paste Room` → always `Nothing To Paste` | `V` | permanently disabled (§3.5.3) |
| 11 | `Delete Room` / `Clear Room` / `Clear Object` | — | text swapped at runtime |
| 12 | `Duplicate Object` | `D` | `DuplicateObject()` |
| 13 | `-` | — | separator |
| 14 | `Bring To Front` | `=` | `BringSelectedObjectFront()` |
| 15 | `Send To Back` | `-` | `SendSelectedObjectBack()` |
| 16 | `-` | — | separator |
| 17 | `Go To Room…` | `G` | `DoGoToDialog()` (`DLOG` 1043) |
| 18 | `-` | — | separator |
| 19 | `Map Window` | `M` | toggle map window; checkmark reflects `isMapOpen` |
| 20 | `Tools Window` | `T` | toggle tools window; checkmark reflects `isToolsOpen` |
| 21 | `Coordinate Window` | `K` | toggle coordinate window; checkmark reflects `isCoordOpen` |

Note the ⌘ collisions across menus: `N` is both Game▸New Game and House▸New House, and `D` is
both Options▸Demo and House▸Duplicate Object. Because the House menu is only inserted in
`kEditMode` and the Game menu items are disabled in `kEditMode` (§3.5.1), `MenuKey` resolves
them unambiguously in practice: `MenuKey` scans menus in bar order and returns the first
*enabled* match. `Options▸Demo…` (⌘D) and `House▸Duplicate Object` (⌘D) *do* both exist and
are both enabled in edit mode when an object is selected — Options is earlier in the bar, so
⌘D runs the demo. This is a genuine 1.0.4 conflict.

#### Editor pop-up menus (not in the menu bar)

`MENU` 140 `Rooms`, 19 items, attached to the Room Info dialog's background pop-up. Items 1–18
map 1:1 onto the background PICT IDs `kSimpleRoom 2000` … `kStars 2017`
(`GliderPRO/Headers/GliderDefines.h:227-244`). Item 19 `Original Artwork` is disabled in the
resource and is enabled at runtime only when the house has custom background art.

| # | Text | Background ID |
|---|---|---|
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
| 19 | `Original Artwork` (disabled in resource) | `kUserBackground` 3000 range |

`MENU` 141 `Tools`, 9 items, maps 1:1 onto the tool-mode constants
(`GliderPRO/Headers/GliderDefines.h:272-280`):

| # | Text | Constant | Value |
|---|---|---|---|
| 1 | `Blowers` | `kBlowerMode` | 1 |
| 2 | `Furniture` | `kFurnitureMode` | 2 |
| 3 | `Prizes` | `kBonusMode` | 3 |
| 4 | `Transport` | `kTransportMode` | 4 |
| 5 | `Switches` | `kSwitchMode` | 5 |
| 6 | `Lighting` | `kLightMode` | 6 |
| 7 | `Appliances Etc.` | `kApplianceMode` | 7 |
| 8 | `Enemies` | `kEnemyMode` | 8 |
| 9 | `Clutter` | `kClutterMode` | 9 |

### 3.4 `InitializeMenus`

`GliderPRO/Sources/InterfaceInit.c:47-74`:

```
1. appleMenu = GetMenu(kAppleMenuID /*128*/)
   if (appleMenu == nil) RedAlert(kErrFailedResourceLoad)      // fatal, error 4
   AppendResMenu(appleMenu, 'DRVR')
   InsertMenu(appleMenu, 0)                                    // 0 = append to end of bar
2. gameMenu = GetMenu(kGameMenuID /*129*/)
   if (gameMenu == nil) RedAlert(kErrFailedResourceLoad)
   InsertMenu(gameMenu, 0)
3. optionsMenu = GetMenu(kOptionsMenuID /*130*/)
   if (optionsMenu == nil) RedAlert(kErrFailedResourceLoad)
   InsertMenu(optionsMenu, 0)
4. menusUp = true
   DrawMenuBar()
5. houseMenu = GetMenu(kHouseMenuID /*131*/)
   if (houseMenu == nil) RedAlert(kErrFailedResourceLoad)
   // NOTE: NOT inserted here — InterfaceInit.c:68-70
6. UpdateMenus(false)
```

So the initial bar is: Apple, Game, Options. The House menu handle exists but is not in the bar
until `UpdateMenus(true)` is called with `theMode == kEditMode`.

Globals (`GliderPRO/Sources/Menu.c:32-33`):

```c
MenuHandle  appleMenu, gameMenu, optionsMenu, houseMenu;
Boolean     menusUp, resumedSavedGame;
```

`resumedSavedGame` gates high-score eligibility (§9.5).

### 3.5 Enable / disable / check / rename rules

`UpdateMenus(Boolean newMode)` (`GliderPRO/Sources/Menu.c:243-275`) is the single entry point:

```
1. if (!menusUp) return;                                       // Menu.c:245-246
2. if (newMode) {
       if (theMode == kEditMode) InsertMenu(houseMenu, 0);
       else                      DeleteMenu(kHouseMenuID);
   }
3. if (theMode == kEditMode) {
       UpdateMenusEditMode();
       if (houseOpen) { UpdateMenusHouseOpen(); UpdateClipboardMenus(); }
       else           UpdateMenusHouseClosed();
       UpdateLinkControl();
   } else {
       UpdateMenusNonEditMode();
   }
4. DrawMenuBar();
```

Every caller that changes the mode passes `newMode = true`; everything else passes `false`.
`DoOptionsMenu(iEditor)` is the only `true` caller (`GliderPRO/Sources/Menu.c:410`).

#### 3.5.1 `UpdateMenusEditMode` (`GliderPRO/Sources/Menu.c:49-60`)

```
DisableMenuItem(gameMenu, iNewGame);          // 1
DisableMenuItem(gameMenu, iTwoPlayer);        // 2
DisableMenuItem(gameMenu, iOpenSavedGame);    // 3   (Menu.c:53)
DisableMenuItem(optionsMenu, iHighScores);    // 3
DisableMenuItem(optionsMenu, iHelp);          // 5
CheckMenuItem(optionsMenu, iEditor, true);
```

So in edit mode you cannot start a game, view high scores, or run the demo from the menus.
(Game▸Load House and Game▸Quit remain enabled — they are handled by
`UpdateMenusHouseOpen`/`Closed`.)

Note: because `BUILD_ARCADE_VERSION` is on, the arrow keys still reach
`DoOptionsMenu(iHighScores)` / `DoGameMenu(iNewGame)` in edit mode (§2.6), bypassing these
disables entirely, since `HandleKeyEvent` calls the handlers directly rather than through
`MenuKey`/`MenuSelect`.

#### 3.5.2 `UpdateMenusNonEditMode` (`GliderPRO/Sources/Menu.c:62-96`)

```
1. if ((noRoomAtAll) || (!houseOpen) || (numberRooms <= 0)) {
       DisableMenuItem(gameMenu, iNewGame);
       DisableMenuItem(gameMenu, iTwoPlayer);
       DisableMenuItem(gameMenu, iOpenSavedGame);            // Menu.c:68
       if (houseOpen) { EnableMenuItem(gameMenu, iLoadHouse);
                        EnableMenuItem(optionsMenu, iHighScores); }
       else           { DisableMenuItem(gameMenu, iLoadHouse);
                        DisableMenuItem(optionsMenu, iHighScores); }
   } else {
       EnableMenuItem(gameMenu, iNewGame);
       EnableMenuItem(gameMenu, iTwoPlayer);
       EnableMenuItem(gameMenu, iLoadHouse);
       EnableMenuItem(optionsMenu, iHighScores);
       EnableMenuItem(gameMenu, iOpenSavedGame);             // Menu.c:84
   }
2. if (demoHouseIndex == -1) DisableMenuItem(optionsMenu, iHelp);
   else                      EnableMenuItem(optionsMenu, iHelp);
3. CheckMenuItem(optionsMenu, iEditor, false);
```

Enable-rule summary for splash mode:

| Item | Enabled when |
|---|---|
| Game▸New Game | house open, has rooms, `numberRooms > 0` |
| Game▸Two Player Game | same |
| Game▸Open Saved Game… | same (but the action always fails — §8.4) |
| Game▸Load House… | `houseOpen` (either branch) |
| Game▸Quit | always (never disabled anywhere) |
| Options▸Room Editor | always |
| Options▸High Scores… | `houseOpen` |
| Options▸Preferences… | always (never disabled anywhere) |
| Options▸Demo… | `demoHouseIndex != -1` |

#### 3.5.3 `UpdateMenusHouseOpen` (`GliderPRO/Sources/Menu.c:98-144`)

```
1. EnableMenuItem(gameMenu, iLoadHouse);
2. if (fileDirty && houseUnlocked) EnableMenuItem(houseMenu, iSave);
   else                            DisableMenuItem(houseMenu, iSave);
3. if (houseUnlocked) EnableMenuItem(houseMenu, iHouse);   else DisableMenuItem(...);
4. if (noRoomAtAll || !houseUnlocked) DisableMenuItem(houseMenu, iRoom);
   else                               EnableMenuItem(houseMenu, iRoom);
5. if ((objActive == kNoObjectSelected) || (!houseUnlocked)) {
       DisableMenuItem(houseMenu, iObject);
       DisableMenuItem(houseMenu, iBringForward);
       DisableMenuItem(houseMenu, iSendBack);
   } else {
       EnableMenuItem(houseMenu, iObject);
       if ((objActive == kInitialGliderSelected) ||     // -2
           (objActive == kLeftGliderSelected) ||        // -3
           (objActive == kRightGliderSelected)) {       // -4
           DisableMenuItem(houseMenu, iBringForward);
           DisableMenuItem(houseMenu, iSendBack);
       } else {
           EnableMenuItem(houseMenu, iBringForward);
           EnableMenuItem(houseMenu, iSendBack);
       }
   }
```

The pseudo-selection sentinels (`GliderPRO/Headers/GliderDefines.h:525-530`):

| Constant | Value |
|---|---|
| `kRoomIsEmpty` | −1 |
| `kObjectIsEmpty` | −1 |
| `kNoObjectSelected` | −1 |
| `kInitialGliderSelected` | −2 |
| `kLeftGliderSelected` | −3 |
| `kRightGliderSelected` | −4 |

#### 3.5.4 `UpdateMenusHouseClosed` (`GliderPRO/Sources/Menu.c:146-163`)

Disables, unconditionally: `iLoadHouse` (Game), and House `iSave`, `iHouse`, `iRoom`, `iObject`,
`iCut`, `iCopy`, `iPaste`, `iClear`, `iDuplicate`.

#### 3.5.5 `UpdateClipboardMenus` (`GliderPRO/Sources/Menu.c:165-241`)

This is where the Cut/Copy/Clear items are *renamed* depending on selection. All strings come
from `STR#` 150 via `GetLocalizedString(index, str)`:

```
1. if (!houseOpen) return;
2. if (houseUnlocked) {
       if (objActive != kNoObjectSelected) {
            SetMenuItemText(houseMenu, iCut,   GetLocalizedString(36))  // "Cut Object"
            SetMenuItemText(houseMenu, iCopy,  GetLocalizedString(37))  // "Copy Object"
            SetMenuItemText(houseMenu, iClear, GetLocalizedString(38))  // "Clear Object"
            EnableMenuItem(houseMenu, iDuplicate);
       } else {
            SetMenuItemText(houseMenu, iCut,   GetLocalizedString(39))  // "Cut Room"
            SetMenuItemText(houseMenu, iCopy,  GetLocalizedString(40))  // "Copy Room"
            SetMenuItemText(houseMenu, iClear, GetLocalizedString(41))  // "Clear Room"
            DisableMenuItem(houseMenu, iDuplicate);
       }
       EnableMenuItem(houseMenu, iCut);
       EnableMenuItem(houseMenu, iCopy);

       // --- the entire hasScrap branch is COMMENTED OUT: Menu.c:197-216 ---
       // it would have set "Paste Room" (42) or "Paste Object" (43) and enabled iPaste
       SetMenuItemText(houseMenu, iPaste, GetLocalizedString(44))       // "Nothing To Paste"
       DisableMenuItem(houseMenu, iPaste);

       EnableMenuItem(houseMenu, iClear);
       EnableMenuItem(houseMenu, iGoToRoom);
       EnableMenuItem(houseMenu, iMapWindow);
       EnableMenuItem(houseMenu, iObjectWindow);
       EnableMenuItem(houseMenu, iCoordinateWindow);
   } else {
       DisableMenuItem(houseMenu, iCut);
       DisableMenuItem(houseMenu, iCopy);
       DisableMenuItem(houseMenu, iPaste);
       DisableMenuItem(houseMenu, iClear);
       DisableMenuItem(houseMenu, iDuplicate);
       DisableMenuItem(houseMenu, iGoToRoom);
       DisableMenuItem(houseMenu, iMapWindow);
       DisableMenuItem(houseMenu, iObjectWindow);
       DisableMenuItem(houseMenu, iCoordinateWindow);
   }
```

The relevant `STR#` 150 entries (`GliderPRO/Glider PRO.r:396`, `STR#` 150 "Localized Strings"):

| Index | String |
|---|---|
| 36 | `Cut Object` |
| 37 | `Copy Object` |
| 38 | `Clear Object` |
| 39 | `Cut Room` |
| 40 | `Copy Room` |
| 41 | `Clear Room` |
| 42 | `Paste Room` (never used — dead) |
| 43 | `Paste Object` (never used — dead) |
| 44 | `Nothing To Paste` |

So in 1.0.4 the Paste item permanently reads `Nothing To Paste` and is permanently disabled;
`MENU` 131's resource text `Paste Room` is only ever visible for the split second before the
first `UpdateMenus` call.

`GetLocalizedString` is `GetIndString(theString, kLocalizedStringsID /*150*/, index)`
(`GliderPRO/Sources/StringUtils.c:321-330`).

#### 3.5.6 Checkmark helpers

`GliderPRO/Sources/Menu.c:626-657`:

```c
void UpdateMapCheckmark (Boolean checkIt)         // Menu.c:626
{ if (!menusUp) return; CheckMenuItem(houseMenu, iMapWindow, checkIt); }

void UpdateToolsCheckmark (Boolean checkIt)       // Menu.c:637
{ if (!menusUp) return; CheckMenuItem(houseMenu, iObjectWindow, checkIt); }

void UpdateCoordinateCheckmark (Boolean checkIt)  // Menu.c:648
{ if (!menusUp) return; CheckMenuItem(houseMenu, iCoordinateWindow, checkIt); }
```

### 3.6 Command-key and menu-selection routing

`GliderPRO/Sources/Events.c:172-173`:

```c
if ((commandDown) && (!optionDown))
    DoMenuChoice(MenuKey(theChar));
```

Command+Option is deliberately *not* routed to the menus — it is the "highlight all objects"
editor gesture (`GliderPRO/Sources/Events.c:481-489`).

`DoMenuChoice` (`GliderPRO/Sources/Menu.c:591-624`):

```
1. if (menuChoice == 0) return;
2. theMenu = HiWord(menuChoice);
   theItem = LoWord(menuChoice);
3. switch (theMenu):
       kAppleMenuID   (128) -> DoAppleMenu(theItem)
       kGameMenuID    (129) -> DoGameMenu(theItem)
       kOptionsMenuID (130) -> DoOptionsMenu(theItem)
       kHouseMenuID   (131) -> DoHouseMenu(theItem)
4. HiliteMenu(0);                    // un-highlight the menu title
```

The same function receives `MenuSelect(where)` results from `HandleMouseEvent` for clicks in
the menu bar.

### 3.7 In-game command keys (menu bar is covered)

During `kPlayMode` the event loop is the game loop, so the menu bar is not consulted;
`DoCommandKey` handles command keys directly (`GliderPRO/Sources/Input.c:53-75`):

| Key | Action |
|---|---|
| ⌘Q | `playing = false; paused = false;` then, `if (!twoPlayerGame && !demoGoing)`, `if (QuerySaveGame()) SaveGame2();` (`Input.c:55-64`) |
| ⌘S | `RefreshScoreboard(kSavingTitleMode /*2*/); SaveGame2(); HideCursor(); CopyRectWorkToMain(&workSrcRect); RefreshScoreboard(kNormalTitleMode /*0*/);` (`Input.c:65-72`) |

Both call `SaveGame2()`, whose body is 100 % commented out (§8.3), so ⌘S flashes "Saving…" on
the scoreboard and does nothing, and ⌘Q asks whether to save and then does not save.

`QuerySaveGame` (`GliderPRO/Sources/Input.c:383-395`):

```c
#define kSaveGameAlert     1041
#define kYesSaveGameButton 1
InitCursor();
FlushEvents(everyEvent, 0);
hitWhat = Alert(kSaveGameAlert, nil);
return (hitWhat == kYesSaveGameButton);
```

`ALRT` 1041 has `stages = 0xCCCC`, i.e. **item 2 ("Don't Save") is the default button** and no
beep — so Return means "Don't Save". This is the only alert in the program with a non-item-1
default besides 1029.

`DoPause` (`GliderPRO/Sources/Input.c:77-...`):

```
1. QSetRect(&bounds, 0, 0, 214, 54)
2. CenterRectInRect(&bounds, &houseRect)
3. LoadScaledGraphic(isEscPauseKey ? kEscPausePictID : kTabPausePictID, &bounds)
4. loop on GetKeys() until the pause key is released and pressed again
   (a Command keypress during the pause routes to DoCommandKey)
```

`PICT` 1015 and 1016 are both 214×54 — the "Paused (Esc)" and "Paused (Tab)" overlays. Which
one is used depends on the `isEscPauseKey` preference (§6.10).

### 3.8 Menu handlers

#### 3.8.1 `DoAppleMenu` (`GliderPRO/Sources/Menu.c:277-299`)

```
switch (theItem) {
    case iAbout:  DoAbout();  break;
    // default: (open the selected desk accessory) is COMMENTED OUT, Menu.c:281-296
}
```

#### 3.8.2 `DoGameMenu` (`GliderPRO/Sources/Menu.c:301-364`)

```
case iNewGame:                                            // 1
     twoPlayerGame = false;
     resumedSavedGame = false;
     NewGame(kNewGameMode /*1*/);
     break;
case iTwoPlayer:                                          // 2
     twoPlayerGame = true;
     resumedSavedGame = false;
     NewGame(kNewGameMode);
     break;
case iOpenSavedGame:                                      // 3   Menu.c:317-324
     resumedSavedGame = true;
     HeyYourPissingAHighScore();                          // ALRT 1046
     if (OpenSavedGame()) {                               // ALWAYS false in 1.0.4
         twoPlayerGame = false;
         NewGame(kResumeGameMode /*0*/);
     }
     break;
case iLoadHouse:                                          // 5
     if (splashDrawn) {
         DoLoadHouse();
         OpenCloseEditWindows();
         UpdateMenus(false);
         incrementModeTime = TickCount() + kIdleSplashTicks;
         if ((theMode == kSplashMode) || (theMode == kPlayMode)) {
             // repaint just the "House: <name>" strip on the splash
             QSetRect(&tempRect, splashOriginH + 474, splashOriginV + 304,
                                 splashOriginH + 474 + 166, splashOriginV + 304 + 12);
             InvalWindowRect(mainWindow, &tempRect);
         }
     }
     break;
case iQuit:                                               // 7
     quitting = true;
     if (!QuerySaveChanges()) quitting = false;
     break;
```

The 166×12 strip at splash-relative (474, 304) is where `DrawOnSplash` writes the house name
(`DrawOnSplash` moves to `splashOriginH + 436, splashOriginV + 314` — the strip starts 38 px to
the right of the pen and 10 px above the baseline, so it covers the tail of a long house name;
this is why `RedrawSplashScreen`, not just the strip invalidation, is what actually refreshes a
long name).

`HeyYourPissingAHighScore` (`GliderPRO/Sources/Menu.c:779-790`):

```c
#define kNoHighScoreAlert 1046
whoCares = Alert(kNoHighScoreAlert, nil);
```

`ALRT` 1046 (40,40,132,300), stages `0x4444`, one button `So What?` at (64,180,84,252), text
`If you resume a saved game, you are ineligible to get on the high scores for that game.`
at (8,8,56,216), icon 1073. Since `OpenSavedGame()` then returns `false`, in 1.0.4 choosing
Open Saved Game… shows this alert and then silently does nothing.

#### 3.8.3 `DoOptionsMenu` (`GliderPRO/Sources/Menu.c:366-434`)

```
case iEditor:                                             // 1
     if (theMode == kEditMode) {
         // ---- leaving the editor ----
         if (fileDirty) SortHouseObjects();
         if (QuerySaveChanges()) {                        // ALRT 1002, may cancel
             theMode = kSplashMode;
             CloseMapWindow(); CloseToolsWindow();
             CloseCoordWindow(); CloseLinkWindow();
             DeselectObject();
             StopMarquee();
             if (isPlayMusicIdle) {
                 if (!StartMusic()) YellowAlert(kYellowNoMusic, 0);
             }
             CloseMainWindow();
             OpenMainWindow();
             incrementModeTime = TickCount() + kIdleSplashTicks;
         }
     } else {
         // ---- entering the editor ----
         theMode = kEditMode;
         StopTheMusic();
         CloseMainWindow();
         OpenMainWindow();
         OpenCloseEditWindows();
     }
     InitCursor();
     UpdateMenus(true);            // <-- the only newMode=true call site, Menu.c:410
     break;
case iHighScores:                                         // 3
     DoHighScores();
     incrementModeTime = TickCount() + kIdleSplashTicks;
     break;
case iPrefs:                                              // 4
     DoSettingsMain();
     incrementModeTime = TickCount() + kIdleSplashTicks;
     break;
case iHelp:                                               // 5
     DoDemoGame();
     break;
```

`OpenCloseEditWindows` (`GliderPRO/Sources/Menu.c:792-813`):

```
1. if (theMode != kEditMode) return;
2. if (houseUnlocked) {
       if (isMapOpen)   OpenMapWindow();
       if (isToolsOpen) OpenToolsWindow();
       if (isCoordOpen) OpenCoordWindow();
   } else {
       CloseMapWindow(); CloseToolsWindow(); CloseCoordWindow();
   }
```

#### 3.8.4 `DoHouseMenu` (`GliderPRO/Sources/Menu.c:436-589`)

All 17 live cases; every one is gated on `houseUnlocked` (the menu items are also disabled, so
this is belt-and-braces):

| Item | Body |
|---|---|
| `iNewHouse` (1) | `if (houseUnlocked) { if (CreateNewHouse()) ... }` — the `// SaveGame(false);` call is **commented out at `GliderPRO/Sources/Menu.c:458`** |
| `iSave` (2) | `DeselectObject(); if (fileDirty) SortHouseObjects(); if (fileDirty && houseUnlocked) { if (wasHouseVersion < kHouseVersion) ConvertHouseVer1To2(); wasHouseVersion = kHouseVersion; WriteHouse(true); ForceThisRoom(thisRoomNumber); ReadyBackground(thisRoom->background, thisRoom->tiles); GetThisRoomsObjRects(); DrawThisRoomsObjects(); }` |
| `iHouse` (4) | `DoHouseInfo()` |
| `iRoom` (5) | `DoRoomInfo()` |
| `iObject` (6) | `DoObjectInfo()` |
| `iCut` (8) | `// PutRoomScrap()` / `// PutObjectScrap()` **commented out**; then `DeleteRoom(true)` or `DeleteObject()` |
| `iCopy` (9) | `// PutRoomScrap()` / `// PutObjectScrap()` **commented out**; only `UpdateClipboardMenus()` remains |
| `iPaste` (10) | body **entirely commented out** |
| `iClear` (11) | `DeleteRoom(true)` or `DeleteObject()` |
| `iDuplicate` (12) | `DuplicateObject()` |
| `iBringForward` (14) | `BringSelectedObjectFront()` |
| `iSendBack` (15) | `SendSelectedObjectBack()` |
| `iGoToRoom` (17) | `DoGoToDialog()` (`DLOG` 1043, `kGoToDialogID` in `House.c`) |
| `iMapWindow` (19) | toggle map window |
| `iObjectWindow` (20) | toggle tools window |
| `iCoordinateWindow` (21) | toggle coordinate window |

The clipboard is therefore entirely non-functional in 1.0.4: `PutObjectScrap`, `PutRoomScrap`,
`GetRoomScrap` and `GetObjectScrap` are all commented out at their call sites, and Cut/Clear are
identical (both just delete).

### 3.9 Dead menu code

`QueryResumeGame` (`GliderPRO/Sources/Menu.c:710-759`) with
`#define kResumeGameDial 1025`, plus its helpers `UpdateResumeDialog`
(`GliderPRO/Sources/Menu.c:659-663`) and `ResumeFilter` (`GliderPRO/Sources/Menu.c:668-703`),
is **dead code**: the only references are its own prototype (`Menu.c:28`), its definition, and
its internal `GetNewDialog` call. Nothing calls it. What it would have done:

```
1. NumToString(thisHousePtr->savedGame.score, scoreStr)
2. hadGliders = thisHousePtr->savedGame.numGliders
   NumToString(hadGliders, glidStr)
3. if (hadGliders == 1) ParamText(glidStr, "\p",  scoreStr, "\p")
   else                 ParamText(glidStr, "\ps", scoreStr, "\p")
4. theDial = GetNewDialog(kResumeGameDial /*1025*/, nil, kPutInFront)
5. loop ModalDialog(resumeFilterUPP, &item) until item == 1 (New Game) or 2 (Resume)
6. return (item == kSheWantsResumeGame /*2*/)
```

`#define kSheWantsNewGame 1` and `#define kSheWantsResumeGame 2`
(`GliderPRO/Sources/Menu.c:18-19`). The `^1` "s" trick is how the dialog text
`(You had ^0 glider^1 & ^2 points.)` is pluralised.

`DoNotInDemo` (`GliderPRO/Sources/Menu.c:767-777`) with `#define kNotInDemoAlert 1037` is inside
`#ifdef COMPILEDEMO`, and `COMPILEDEMO` is commented out
(`GliderPRO/Headers/GliderDefines.h:12`), so `ALRT` 1037 is unreachable in this build.
---

## 4. `DialogUtils.c` — the dialog toolkit

`GliderPRO/Sources/DialogUtils.c` is 806 lines and defines two local constants:

```c
#define kActive     0        // DialogUtils.c
#define kInactive   255
```

matching `kControlActive 0` / `kControlInactive 255`
(`GliderPRO/Headers/Externs.h:24-25`).

### 4.1 Live vs. commented-out helpers

**Commented out in their entirety** — this matters enormously for fidelity, because it means
*no dialog in Glider PRO 1.0.4 is centered or zoomed*; every dialog appears at the literal
global `boundsRect` stored in its `DLOG` resource:

| Function | Line | Would have done |
|---|---|---|
| `GetPutDialogCorner` | `GliderPRO/Sources/DialogUtils.c:39` | compute a corner for `StandardPutFile` |
| `GetGetDialogCorner` | `GliderPRO/Sources/DialogUtils.c:73` | ditto for `StandardGetFile` |
| `CenterDialog` | `GliderPRO/Sources/DialogUtils.c:105` | center a `DLOG` on the main screen |
| `TrueCenterDialog` | `GliderPRO/Sources/DialogUtils.c:157` | exact center (no menu-bar bias) |
| `CenterAlert` | `GliderPRO/Sources/DialogUtils.c:192` | center an `ALRT` |
| `ZoomOutDialogRect` | `GliderPRO/Sources/DialogUtils.c:226` | zoom-rect open animation |
| `ZoomOutAlertRect` | `GliderPRO/Sources/DialogUtils.c:280` | ditto for alerts |
| `AddMenuToPopUp` | `GliderPRO/Sources/DialogUtils.c:549` | populate a pop-up `CNTL` |

There are commented-out `CenterAlert(...)` call sites left behind in
`GliderPRO/Sources/Prefs.c:278`, `GliderPRO/Sources/HouseIO.c:605`, and
`GliderPRO/Sources/HouseIO.c:650`, which is how we know the intent was to center them.

Consequence: `DLOG`s whose `boundsRect` starts at `(0,0)` — 1003, 1007, 1010, 1011, 1013, 1014,
1015, 1016, 1019, **1020 (High Score Name)**, 1022, 1027, 1033, 1034, 1035, 1045 — appear with
their top edge at the very top of the screen, i.e. **underneath the menu bar** on the original
Mac. A Go port that "helpfully" centers them will not look like Glider PRO. See §Porting notes.

### 4.2 `BringUpDialog` (`GliderPRO/Sources/DialogUtils.c:24-37`)

The standard way every panel is opened:

```
1. *theDialog = GetNewDialog(dialogID, nil, kPutInFront /*(WindowPtr)-1L*/)
2. if (*theDialog == nil) RedAlert(kErrDialogDidntLoad)     // fatal, error 3
3. SetPort((GrafPtr)*theDialog)      // == SetPortDialogPort
4. ShowWindow(GetDialogWindow(*theDialog))
5. DrawDefaultButton(*theDialog)
```

`GliderPRO/Sources/Settings.c:502-511` (`DoControlPrefs`) is the one exception: it calls
`GetNewDialog` directly so it can measure the four icon rects *before* the window is shown.

### 4.3 Every live helper

| Function | Line | Behaviour |
|---|---|---|
| `BringUpDialog` | 24 | see §4.2 |
| `GetDialogRect` | 137 | copy the dialog window's port rect |
| `FlashDialogButton` | 335 | `GetDialogItem`; `HiliteControl(h, kControlButtonPart /*10*/)`; `Delay(8, &dummy)`; `HiliteControl(h, 0)` — an 8-tick (≈133 ms) button flash |
| `DrawDefaultButton` | 352 | item 1's rect, `InsetRect(-4,-4)`, `PenSize(3,3)`, `FrameRoundRect(&itemRect, 16, 16)`, `PenNormal()` |
| `GetDialogString` | 368 | `GetDialogItemText` wrapper |
| `SetDialogString` | 381 | `SetDialogItemText` wrapper |
| `GetDialogStringLen` | 394 | `GetDialogItemText` then return `theStr[0]` |
| `GetDialogItemValue` | 410 | `GetControlValue` |
| `SetDialogItemValue` | 424 | `SetControlValue` |
| `ToggleDialogItemValue` | 437 | `SetControlValue(h, GetControlValue(h) == 0 ? 1 : 0)` |
| `SetDialogNumToStr` | 456 | `NumToString(theNum, theStr); SetDialogItemText(...)` |
| `GetDialogNumFromStr` | 472 | `GetDialogItemText`; `StringToNum(theStr, theNum)` |
| `GetDialogItemRect` | 487 | `GetDialogItem` → rect |
| `SetDialogItemRect` | 499 | `SetDialogItem` with a new rect |
| `OffsetDialogItemRect` | 513 | get rect, `OffsetRect(h,v)`, set rect |
| `SelectFromRadioGroup` | 529 | see below |
| `GetPopUpMenuValue` | 562 | `GetControlValue` on a pop-up `CNTL` |
| `SetPopUpMenuValue` | 575 | `SetControlValue` on a pop-up `CNTL` |
| `MyEnableControl` | 588 | `HiliteControl(h, kActive /*0*/)` |
| `MyDisableControl` | 601 | `HiliteControl(h, kInactive /*255*/)` |
| `DrawDialogUserText` | 616 | see §4.4 |
| `DrawDialogUserText2` | 655 | see §4.5 |
| `LoadDialogPICT` | 677 | `GetPicture(pictID)`; `DrawPicture(thePict, &itemRect)` |
| `FrameDialogItem` | 693 | `FrameRect` around the item |
| `FrameDialogItemC` | 706 | `Index2Color(color, &rgb); RGBForeColor(&rgb); FrameRect; ForeColor(blackColor)` |
| `FrameOvalDialogItem` | 724 | `FrameOval` |
| `BorderDialogItem` | 738 | selective edges by bitmask, see §4.6 |
| `ShadowDialogItem` | 779 | draws a drop shadow |
| `EraseDialogItem` | 797 | `EraseRect` on the item |

`SelectFromRadioGroup(DialogPtr, short which, short first, short last)`
(`GliderPRO/Sources/DialogUtils.c:529-547`):

```
for (i = first; i <= last; i++) SetDialogItemValue(theDialog, i, 0);
SetDialogItemValue(theDialog, which, 1);
```

### 4.4 `DrawDialogUserText` — the workhorse for `userItem` labels

`GliderPRO/Sources/DialogUtils.c:616-653`. Glider PRO renders most of its variable text into
`userItem`s by hand instead of using `statText`, so that it can invert them for selection (the
house browser) and colour them. Exact algorithm:

```
 1. GetDialogItemRect(theDialog, item, &iRect)
 2. TextFont(applFont)              // the application font, size 9
    TextSize(9)
 3. PasStringCopy(theString, stringCopy)
 4. width  = iRect.right - iRect.left
 5. if (StringWidth(stringCopy) + 2 > width) CollapseStringToWidth(stringCopy, width - 2)
 6. textLong = (long)stringCopy[0]                  // length
    BlockMove(&stringCopy[1], newString, textLong)  // strip the length byte
 7. tempRect = iRect; OffsetRect(&tempRect, 0, 1); EraseRect(&tempRect)
 8. inset = (width - (StringWidth(stringCopy) + 2)) / 2
    iRect.left  += inset
    iRect.right -= inset
 9. TETextBox(newString, textLong, &iRect, teCenter)
10. if (invert) { OffsetRect(&iRect, 0, 1); InvertRect(&iRect); }
```

`CollapseStringToWidth` (`GliderPRO/Sources/StringUtils.c:282-300`):

```
dotsWide = StringWidth("\p…")          // width of the single MacRoman 0xC9 ellipsis
while (StringWidth(theStr) + dotsWide > wide && theStr[0] > 0) theStr[0]--;
PasStringConcat(theStr, "\p…");
```

So a too-long house name in the browser becomes `Long House Nam…`.

Go port: reproduce the "erase one pixel lower, then draw centered, then invert one pixel lower"
sequence or the selection highlight will be off by a pixel against the original screenshots.

### 4.5 `DrawDialogUserText2`

`GliderPRO/Sources/DialogUtils.c:655-675`. Same font/collapse logic, but left-aligned on the
item's baseline:

```
1. GetDialogItemRect; TextFont(applFont); TextSize(9)
2. PasStringCopy; collapse if too wide
3. MoveTo(iRect.left, iRect.bottom); DrawString(stringCopy);
```

Used only by the About box for the Memory:/Screen: lines (§10.3).

### 4.6 `BorderDialogItem` bitmask

`GliderPRO/Sources/DialogUtils.c:738-777`. The `sides` argument is decoded from the high bit
down, so the order in the source is bottom-ish first; the bit values are:

| Bit | Value | Edge |
|---|---|---|
| 0 | 1 | left |
| 1 | 2 | top |
| 2 | 4 | bottom |
| 3 | 8 | right |

i.e. `sides = 15` draws all four edges, `sides = 6` draws top + bottom. Exact geometry (note the
−1 offsets on top and left, so the border sits *outside* the item):

| Edge | Line drawn |
|---|---|
| right (8) | `MoveTo(right, top)` → `LineTo(right, bottom)` |
| bottom (4) | `MoveTo(left, bottom)` → `LineTo(right, bottom)` |
| top (2) | `MoveTo(left, top-1)` → `LineTo(right, top-1)` |
| left (1) | `MoveTo(left-1, top)` → `LineTo(left-1, bottom)` |

`ShadowDialogItem(theDialog, item, thickness)`
(`GliderPRO/Sources/DialogUtils.c:779-795`):

```
PenSize(thickness, thickness);
MoveTo(left + thickness, bottom);  Line(right - left - thickness, 0);
MoveTo(right, top + thickness);    Line(0, bottom - top - thickness);
PenNormal();
```

### 4.7 Colour: `FrameDialogItemC` and `kRedOrangeColor8`

`FrameDialogItemC(DialogPtr, short item, long color)`
(`GliderPRO/Sources/DialogUtils.c:706-722`):

```
GetDialogItemRect(theDialog, item, &iRect);
Index2Color(color, &theRGBColor);      // CLUT index -> RGB
RGBForeColor(&theRGBColor);
FrameRect(&iRect);
ForeColor(blackColor);
```

`kRedOrangeColor8` is `23` (`GliderPRO/Headers/GliderDefines.h:542`) with the source comment
`// actually, 18`. It is a **CLUT index** into the 8-bit indexed palette, resolved via
`Index2Color`. Every Preferences panel frames its "group box" `userItem`s in this colour.

A Go port must decide what index 23 actually is. There are two `clut` resources in the resource
fork; the game runs at `kPreferredDepth 8` (`GliderPRO/Headers/Externs.h:15`) with the
application's own palette. Because the code also refers to raw indices 4, 5, 28, 244 (the
game-over background) and to `ColorText(theStr, 4L)`, a faithful port needs the actual CLUT.
That is a data dependency on the `clut` resources, not on this document. See §Open questions.
---

## 5. Dialog and alert resource inventory

### 5.1 All 28 `DLOG` resources (byte-decoded)

`proc` is `1` (`dBoxProc`) for every one; `refCon` is `0` for every one; the title is empty for
every one. `vis` = the `visible` byte, `go` = `goAwayFlag`.

| ID | Purpose | bounds (t,l,b,r) | W×H | vis | go | `DITL` | Rez line |
|---|---|---|---|---|---|---|---|
| 150 | About Glider PRO | 62,62,242,446 | 384×180 | 1 | 1 | 150 | `GliderPRO/Glider PRO.r:6283` |
| 1000 | Load House (browser) | 36,16,300,446 | 430×264 | 0 | 0 | 1000 | `:6148` |
| 1001 | House Info (editor) | 32,68,357,380 | 312×325 | 0 | 0 | 1001 | `:6153` |
| 1003 | Room Info (editor) | 0,0,267,384 | 384×267 | 0 | 0 | 1003 | `:6158` |
| 1007 | Object: Blower | 0,0,167,256 | 256×167 | 0 | 0 | 1007 | `:6163` |
| 1010 | Object: Furniture | 0,0,120,256 | 256×120 | 0 | 0 | 1010 | `:6168` |
| 1011 | Object: Switch | 0,0,187,256 | 256×187 | 0 | 0 | 1011 | `:6173` |
| **1012** | **Preferences (main)** | 40,40,170,328 | **288×130** | 1 | 1 | 1012 | `:6178` |
| 1013 | Object: Light | 0,0,143,256 | 256×143 | 0 | 0 | 1013 | `:6183` |
| 1014 | Object: Appliance | 0,0,153,256 | 256×153 | 0 | 0 | 1014 | `:6188` |
| 1015 | Object: Invisible Bonus | 0,0,155,256 | 256×155 | 0 | 0 | 1015 | `:6193` |
| 1016 | Select Original Art | 0,0,144,256 | 256×144 | 1 | 1 | 1016 | `:6198` |
| **1017** | **Preferences: Display** | 40,40,280,373 | **333×240** | 0 | 0 | 1017 | `:6203` |
| **1018** | **Preferences: Sound** | 40,40,216,356 | **316×176** | 0 | 0 | 1018 | `:6208` |
| 1019 | Object: Grease | 0,0,146,256 | 256×146 | 0 | 0 | 1019 | `:6213` |
| **1020** | **High-score name entry** | 0,0,109,316 | **316×109** | 1 | 1 | 1020 | `:6218` |
| **1021** | **High-score banner entry** | 40,40,162,356 | **316×122** | 1 | 1 | 1021 | `:6223` |
| 1022 | Object: Transport | 0,0,185,256 | 256×185 | 0 | 0 | 1022 | `:6228` |
| **1023** | **Preferences: Controls** | 40,40,216,356 | **316×176** | 0 | 0 | 1023 | `:6233` |
| **1024** | **Preferences: Brains** | 40,40,232,356 | **316×192** | 0 | 0 | 1024 | `:6238` |
| **1025** | **Resume Game (DEAD)** | 40,40,164,352 | 312×124 | 0 | 0 | 1025 | `:6268` |
| 1026 | Validate (insert original disk) | 50,50,154,362 | 312×104 | 0 | 0 | 1026 | `:6243` |
| 1027 | Object: Enemy | 0,0,160,256 | 256×160 | 0 | 0 | 1027 | `:6248` |
| 1033 | Object: Flower | 0,0,175,256 | 256×175 | 0 | 0 | 1033 | `:6253` |
| 1034 | Object: Trigger | 0,0,187,256 | 256×187 | 0 | 0 | 1034 | `:6258` |
| 1035 | Object: Microwave | 0,0,160,278 | 278×160 | 0 | 0 | 1035 | `:6263` |
| 1043 | Go To Room… | 40,40,201,320 | 280×161 | 1 | 0 | 1043 | `:6273` |
| 1045 | Object: PICT Object | 0,0,141,256 | 256×141 | 0 | 0 | 1045 | `:6278` |

The dialogs marked with `vis = 0` are made visible by `BringUpDialog`'s explicit `ShowWindow`;
the `vis = 1` ones flash into existence at `GetNewDialog` time. `DoControlPrefs` uses this
deliberately: `DLOG` 1023 is invisible, so it can measure and inset the four control icon rects
before showing the window (`GliderPRO/Sources/Settings.c:502-521`).

### 5.2 All 26 `ALRT` resources (byte-decoded)

| ID | Purpose | bounds (t,l,b,r) | `DITL` | `stages` | Rez line | Called from |
|---|---|---|---|---|---|---|
| 130 | Color depth choice | 0,0,96,274 | 130 | `0x4444` | `:5195` | `SwitchDepthOrAbort` (`Environ.c` ≈404) |
| 140 | File error | 92,60,220,446 | 140 | `0x5555` | `:5187` | `CheckFileError` (`FileError.c:76`) |
| 160 | New Preferences file | 40,40,140,340 | 160 | `0x5555` | `:5199` | `BringUpDeletePrefsAlert` (`Prefs.c:273`) |
| 170 | Fatal ("Red") error | 92,84,220,432 | 170 | `0x5555` | `:5191` | `RedAlert` (`Utilities.c:135`) |
| 180 | Set memory size (fatal) | 40,40,156,334 | 180 | `0x5555` | `:5231` | `CheckMemorySize` (`Environ.c` ≈640) |
| 181 | Low memory (demo) | 40,40,168,328 | 181 | `0x5555` | `:5259` | `CheckMemorySize` |
| 1002 | Save changes? | 0,0,128,320 | 1002 | `0x5555` | `:5203` | `QuerySaveChanges` (`HouseIO.c:607`) |
| 1004 | Create new room? | 40,40,136,328 | 1004 | `0x5555` | `:5207` | editor |
| 1005 | Delete room? | 40,40,136,328 | 1005 | `0x5555` | `:5211` | editor |
| 1006 | Yellow Alert (non-fatal) | 40,40,168,340 | 1006 | `0x5555` | `:5215` | `YellowAlert` (`HouseIO.c:654`) |
| 1008 | No more objects | 40,40,136,320 | 1008 | `0x5555` | `:5227` | editor |
| 1009 | House banner (**unused**) | 0,0,150,346 | 1009 | `0x4444` | `:5219` | `kHouseBannerAlert` declared at `Play.c:18`, never referenced |
| 1028 | No more "special" objects | 40,40,148,320 | 1028 | `0x5555` | `:5223` | editor |
| 1029 | Lock house warning | 40,40,164,338 | 1029 | **`0xFFFF`** | `:5235` | `WarnLockingHouse` (`HouseInfo.c` ≈306) |
| 1030 | Sound Manager 3.0 nag | 40,40,148,314 | 1030 | `0x4444` | `:5239` | `BitchAboutSM3` (`Sound.c:527`) — **call site commented out** |
| 1031 | No printing | 40,40,112,308 | 1031 | `0x5555` | `:5243` | `DoPrintDocAE` (`AppleEvents.c` ≈131) |
| 1032 | Clear scores? | 40,40,129,320 | 1032 | `0x4444` | `:5247` | `HowToZeroScores` (`HouseInfo.c:319`) |
| 1036 | Room PICT missing | 40,40,148,314 | 1036 | `0x4444` | `:5251` | background loader |
| 1037 | Not in demo | 40,40,164,326 | 1037 | `0x4444` | `:5255` | `DoNotInDemo` — **compiled out** |
| 1038 | Music out of memory | 54,96,162,370 | 1038 | `0x4444` | `:5263` | `InitMusic` |
| 1039 | Sound out of memory | 54,96,162,370 | 1039 | `0x4444` | `:5267` | `TellHerNoSounds` (`Sound.c` ≈515) |
| 1040 | Changes need relaunch | 54,96,126,377 | 1040 | `0x4444` | `:5271` | `BitchAboutChanges` (`Settings.c:1474`) |
| 1041 | Save game before quitting? | 40,40,116,302 | 1041 | **`0xCCCC`** | `:5275` | `QuerySaveGame` (`Input.c:383`) |
| 1042 | Color depth was switched | 40,40,148,296 | 1042 | `0x5555` | `:5279` | `kColorSwitchedAlert` (`Events.c:46`) |
| 1044 | Saved-game/house mismatch | 40,40,168,340 | 1044 | `0x5555` | `:5283` | `SavedGameMismatchError` (`SavedGames.c:152`) |
| 1046 | Resume forfeits high score | 40,40,132,300 | 1046 | `0x4444` | `:5287` | `HeyYourPissingAHighScore` (`Menu.c:779`) |

### 5.3 Complete `DITL` dumps for the shell dialogs

Rects below are `(top, left, bottom, right)` **relative to the dialog**, exactly as stored.
`en` = enabled flag from the type byte (`0x80` clear = enabled).

#### `DITL` 150 — About Glider PRO (9 items, `GliderPRO/Glider PRO.r:5172`)

| # | Type | Rect | Content |
|---|---|---|---|
| 1 | picture | 113,317,176,380 | `PICT` 150 — 63×63, the un-highlit "Okay" diamond |
| 2 | statText | 155,56,171,184 | `''` — filled at runtime with the long `vers` string |
| 3 | icon | 139,11,171,43 | `ICON` 150 (32×32) |
| 4 | picture | 5,6,105,378 | `PICT` 153 — 372×100, the title art |
| 5 | statText | 139,56,155,184 | `by john calhoun` |
| 6 | statText | 114,11,130,304 | `© 1994-2000 Casady & Greene, Inc.` |
| 7 | userItem | 139,192,148,312 | 120×9 — "Memory:  ...K" |
| 8 | userItem | 148,192,157,312 | 120×9 — "Screen:  WxHxD" |
| 9 | userItem | 157,192,166,312 | 120×9 — **never drawn** |

Item 1 is a `picture`, not a `button`: the About box's Okay is a fake diamond-shaped button
implemented with a hand-built region (§10.3).

#### `DITL` 1012 — Preferences main (11 items, `GliderPRO/Glider PRO.r:4486`)

| # | Type | Rect | Content |
|---|---|---|---|
| 1 | button | 102,222,122,280 | `Okay` (58×20) |
| 2 | picture | 0,0,32,288 | `PICT` 1013 — 289×32 header art |
| 3 | icon | 44,32,76,64 | `ICON`/`cicn` 1010 — Display button |
| 4 | icon | 44,96,76,128 | 1011 — Sounds button |
| 5 | icon | 44,160,76,192 | 1012 — Controls button |
| 6 | icon | 44,224,76,256 | 1013 — Brains button |
| 7 | userItem | 80,19,92,79 | 60×12 label under icon 3 |
| 8 | userItem | 80,83,92,143 | label under icon 4 |
| 9 | userItem | 80,147,92,207 | label under icon 5 |
| 10 | userItem | 80,211,92,271 | label under icon 6 |
| 11 | button | 102,8,122,104 | `All Defaults` (96×20) |

#### `DITL` 1017 — Preferences ▸ Display (17 items, `GliderPRO/Glider PRO.r:4599`)

| # | Type | Rect | Content |
|---|---|---|---|
| 1 | button | 211,267,231,325 | `Okay` |
| 2 | button | 211,201,231,259 | `Cancel` |
| 3 | icon | 40,209,72,241 | 1020 — "1 room" preview |
| 4 | icon | 40,249,72,281 | 1021 — "3 rooms" |
| 5 | icon | 40,289,72,321 | 1022 — "9 rooms" |
| 6 | statText | 40,8,72,203 | `Number of Rooms to Display:\r(the less rooms, the faster)` |
| 7 | picture | 0,0,32,333 | `PICT` 1006 — 333×32 header |
| 8 | userItem | 80,8,81,325 | 317×1 divider rule |
| 9 | checkBox | 142,8,160,264 | `Beautiful opening color fade` |
| 10 | radioButton | 85,8,101,264 | `Use current depth when possible` |
| 11 | radioButton | 101,8,117,264 | `Always play in 256 colors` |
| 12 | radioButton | 117,8,133,264 | `Always play in 16 grays` |
| 13 | userItem | 137,8,138,325 | divider rule |
| 14 | userItem | 200,8,201,325 | divider rule |
| 15 | button | 211,8,231,72 | `Defaults` (64×20) |
| 16 | checkBox | 160,8,178,264 | `Use Quickdraw™ (slower)` — **dead control** (§6.8) |
| 17 | checkBox | 178,8,196,264 | `Run on second monitor` |

#### `DITL` 1018 — Preferences ▸ Sound (13 items, `GliderPRO/Glider PRO.r:4632`)

| # | Type | Rect | Content |
|---|---|---|---|
| 1 | button | 148,250,168,308 | `Okay` |
| 2 | button | 148,184,168,242 | `Cancel` |
| 3 | icon | 52,244,84,276 | 1030, `en=False` (decoration: the speaker) |
| 4 | icon | 69,276,101,308 | 1032, `en=True` — **Softer** |
| 5 | icon | 35,276,67,308 | 1031, `en=True` — **Louder** |
| 6 | statText | 60,163,76,217 | `Volume:` |
| 7 | statText | 60,217,76,243 | `''` — 26×16, the volume number |
| 8 | checkBox | 96,7,114,187 | `Play music when idle` |
| 9 | checkBox | 114,7,131,187 | `Play music during game` |
| 10 | statText | 40,8,88,153 | `Set the game volume and background music options.` |
| 11 | userItem | 138,8,139,308 | 300×1 divider rule |
| 12 | picture | 0,0,32,316 | `PICT` 1008 — 316×32 header |
| 13 | button | 148,8,168,72 | `Defaults` |

Note items 4 and 5: the *Louder* icon (item 5) sits **above** the *Softer* icon (item 4) in
screen coordinates (tops 35 vs 69), but Softer has the lower item number. The
`kSofterItem 4` / `kLouderItem 5` constants match the item numbers, not the visual order.

#### `DITL` 1023 — Preferences ▸ Controls (15 items, `GliderPRO/Glider PRO.r:4724`)

| # | Type | Rect | Content |
|---|---|---|---|
| 1 | button | 148,250,168,308 | `Okay` |
| 2 | button | 148,184,168,242 | `Cancel` |
| 3 | userItem | 138,8,139,308 | divider rule |
| 4 | picture | 0,0,32,316 | `PICT` 1012 — 316×32 header |
| 5 | icon | 43,80,75,112 | 1040 — **Right** (`kRightControl`) |
| 6 | icon | 43,20,75,52 | 1041 — **Left** (`kLeftControl`) |
| 7 | icon | 43,140,75,172 | 1042 — **Battery** (`kBattControl`) |
| 8 | icon | 43,200,75,232 | 1043 — **Bands** (`kBandControl`) |
| 9 | userItem | 80,68,92,124 | 56×12 key-name label under icon 5 |
| 10 | userItem | 80,8,92,64 | label under icon 6 |
| 11 | userItem | 80,128,92,184 | label under icon 7 |
| 12 | userItem | 80,188,92,244 | label under icon 8 |
| 13 | button | 148,8,168,72 | `Defaults` |
| 14 | radioButton | 99,8,115,140 | `Esc Pauses Game` |
| 15 | radioButton | 115,8,131,140 | `Tab Pauses Game` |

Visual left-to-right order is Left (left=20), Right (left=80), Battery (left=140),
Bands (left=200), but the item numbers are Right=5, Left=6, Battery=7, Bands=8. The label
`userItem` for icon *n* is item *n+4* — see `UpdateControlKeyName` (§6.10).

#### `DITL` 1024 — Preferences ▸ Brains (15 items, `GliderPRO/Glider PRO.r:4744`)

| # | Type | Rect | Content |
|---|---|---|---|
| 1 | button | 164,250,184,308 | `Okay` |
| 2 | button | 164,184,184,242 | `Cancel` |
| 3 | userItem | 155,8,156,308 | divider rule |
| 4 | picture | 0,0,32,316 | `PICT` 1014 — 316×32 header |
| 5 | editText | 47,269,63,305 | 36×16 — max-houses field |
| 6 | statText | 40,8,72,264 | `Maximum houses displayed (12-500): \r(larger = more memory)` |
| 7 | checkBox | 74,8,92,148 | `Quick Transitions` |
| 8 | checkBox | 93,8,111,148 | `Zoom Windows` |
| 9 | button | 164,8,184,72 | `Defaults` |
| 10 | checkBox | 112,8,130,148 | `Automatic Demo` |
| 11 | checkBox | 131,8,149,148 | `Background Tasks` |
| 12 | checkBox | 93,156,111,308 | `Error-Check House` |
| 13 | checkBox | 112,156,130,308 | `Use "Pretty Map"` |
| 14 | checkBox | 131,156,149,308 | `Do Create Dialog` |
| 15 | statText | 76,156,92,308 | `Editor Options:` |

#### `DITL` 1000 — Load House browser (30 items, `GliderPRO/Glider PRO.r:4314`)

| # | Type | Rect | Content |
|---|---|---|---|
| 1 | button | 235,359,255,417 | `Okay` |
| 2 | button | 235,286,255,344 | `Cancel` |
| 3 | picture | 0,0,32,430 | `PICT` 1001 (`kLoadTitlePict1`). Note the two widths differ: the `DITL` item rect is **430**×32 while `PICT` 1001's own picFrame is **431**×32, so the Dialog Manager scales it down by one pixel horizontally. |
| 4 | userItem | 240,8,256,24 | 16×16, `en=False` — spinner/decoration |
| 5–16 | userItem ×12 | see below | file-name labels (96×12), `kLoadNameFirstItem` 5 … `kLoadNameLastItem` 16 |
| 17–28 | userItem ×12 | see below | file-icon cells (32×32), `kLoadIconFirstItem` 17 … `kLoadIconLastItem` 28 |
| 29 | icon | 224,183,256,215 | `ICON` 1050 — scroll-up arrow (`kScrollUpItem`) |
| 30 | icon | 224,215,256,247 | `ICON` 1051 — scroll-down arrow (`kScrollDownItem`) |

Name-label `userItem` positions (12 items, 3 rows × 4 columns), 96 wide × 12 tall:

| Item | Row | Col | top | left |
|---|---|---|---|---|
| 5 | 1 | 1 | 81 | 16 |
| 6 | 1 | 2 | 81 | 116 |
| 7 | 1 | 3 | 81 | 216 |
| 8 | 1 | 4 | 81 | 316 |
| 9 | 2 | 1 | 142 | 16 |
| 10 | 2 | 2 | 142 | 116 |
| 11 | 2 | 3 | 142 | 216 |
| 12 | 2 | 4 | 142 | 316 |
| 13 | 3 | 1 | 203 | 17 |
| 14 | 3 | 2 | 203 | 117 |
| 15 | 3 | 3 | 203 | 217 |
| 16 | 3 | 4 | 203 | 317 |

(Row 3's lefts are 1 px greater than rows 1–2 — an authoring slip preserved in the resource.)

Icon-cell `userItem` positions, 32×32:

| Item | Row | Col | top | left |
|---|---|---|---|---|
| 17 | 1 | 1 | 48 | 48 |
| 18 | 1 | 2 | 48 | 148 |
| 19 | 1 | 3 | 48 | 248 |
| 20 | 1 | 4 | 48 | 348 |
| 21 | 2 | 1 | 109 | 48 |
| 22 | 2 | 2 | 109 | 148 |
| 23 | 2 | 3 | 109 | 248 |
| 24 | 2 | 4 | 109 | 348 |
| 25 | 3 | 1 | 170 | 48 |
| 26 | 3 | 2 | 170 | 148 |
| 27 | 3 | 3 | 170 | 248 |
| 28 | 3 | 4 | 170 | 348 |

Row pitch is 61 px; column pitch is 100 px. The icon is centered 32 px to the right of the
name label's left edge, i.e. the name is *below* the icon, 96 px wide, aligned to the 100 px
column grid.

#### `DITL` 1020 — High-score name entry (6 items, `GliderPRO/Glider PRO.r:4671`)

| # | Type | Rect | Content |
|---|---|---|---|
| 1 | button | 81,250,101,308 | `Okay` |
| 2 | editText | 52,157,68,297 | 140×16, prefilled `Your Name` |
| 3 | statText | 8,8,41,308 | `Your score of ^0 is #^1 on the top ten high scores for ^2.` |
| 4 | statText | 49,16,81,132 | `Enter your name:\r(15 letters max.)` |
| 5 | statText | 81,154,97,173 | `''` — 19×16 live character counter |
| 6 | statText | 81,175,97,224 | `letters` |

#### `DITL` 1021 — High-score banner entry (5 items, `GliderPRO/Glider PRO.r:4687`)

| # | Type | Rect | Content |
|---|---|---|---|
| 1 | button | 94,250,114,308 | `Okay` |
| 2 | editText | 67,11,83,305 | 294×16 |
| 3 | statText | 8,11,56,305 | `Getting #1 on the high scores entitles you to change the high score banner.\r(31 letters max.)` |
| 4 | statText | 94,29,110,78 | `letters` |
| 5 | statText | 94,8,110,27 | `''` — live character counter |

Note that 1020 and 1021 put the counter and the word "letters" in the *opposite* order
(1020: counter=5, word=6; 1021: word=4, counter=5). The code refers to them by the constants
`kNameNCharsItem 5` and `kBannerScoreNCharsItem 5` (`GliderPRO/Sources/HighScores.c:28,31`), so
both happen to be item 5 — the layouts differ but the item number agrees.

#### `DITL` 1025 — Resume Game (5 items, `GliderPRO/Glider PRO.r:5070`) — DEAD

| # | Type | Rect | Content |
|---|---|---|---|
| 1 | button | 96,224,116,304 | `New Game` (80×20) |
| 2 | button | 96,136,116,216 | `Resume` (80×20) |
| 3 | statText | 56,8,88,264 | `(You had ^0 glider^1 & ^2 points.)` |
| 4 | statText | 8,8,56,264 | `You have a saved game.  Beginning a new game will overwrite your saved game.  What to do?` |
| 5 | icon | 8,272,40,304 | `ICON` 1001 |

#### `DITL` 1026 — Validation (4 items, `GliderPRO/Glider PRO.r:4773`) — unreachable (`COMPILENOCP`)

| # | Type | Rect | Content |
|---|---|---|---|
| 1 | button | 76,246,96,304 | `Cancel` |
| 2 | statText | 8,8,24,264 | `Insert your original Glider PRO disk.` |
| 3 | statText | 32,8,64,264 | `Your orginal disk is required only once, right after you install the game.` (typo `orginal` is in the resource) |
| 4 | icon | 8,272,40,304 | `ICON` 140 |

### 5.4 Complete `DITL` dumps for the alerts

`^0`..`^3` are `ParamText` substitution points.

| `ALRT` | Items |
|---|---|
| **130** | 1 button `256` (68,208,88,266); 2 button `16` (68,142,88,200); 3 button `Quit` (68,52,88,110); 4 icon 130; 5 statText `Glider PRO™ requires 256 colors (or grays) or 16 grays.  Select the option you would like.` |
| **140** | 1 button `Okay` (92,313,112,371); 2 statText `^0` (29,14,89,295); 3 statText `(error = ^1)` (93,14,111,252); 4 icon 140; 5 statText `A File Error Loading/Saving ^2` (6,14,23,332) |
| **160** | 1 button `Okay` (69,230,89,288); 2 statText `You have a new Preferences file.  All preferences have been set to their defaults.` (13,14,61,231); 3 icon 160 |
| **170** | 1 button `Okay` (92,274,112,332); 2 statText `^0` (8,10,24,257); 3 statText `^1` (28,10,96,257); 4 statText `(program error = ^2)` (100,10,116,257); 5 icon 170 |
| **180** | 1 button `Quit` (88,228,108,286); 2 statText `Glider PRO™ requires more memory.  Get Info on Glider PRO™ in the Finder to increase its memory size or try quitting other applications.`; 3 icon 180; 4 statText `(We need about: ^0K)` (76,8,92,215) |
| **181** | 1 button `Okay` (100,222,120,280); 2 statText `Glider PRO™ Demo requires more memory to run on this Mac.  For this Mac, try giving the game ^0K.  If the game is on a locked volume, copy it to your hard drive.` |
| **1002** | 1 button `Save` (99,253,119,311); 2 button `Discard` (71,253,91,311); 3 button `Cancel` (99,184,119,242); 4 statText `You have made changes to ^0 and haven't saved these changes.  Save the house before proceeding?`; 5 icon 1000 |
| **1006** | 1 button `Okay` (100,234,120,292); 2 statText `^0` (25,9,89,253); 3 statText `A problem came up:` (9,9,25,195); 4 statText `Error #: ^1` (102,9,118,148); 5 icon 1006 |
| **1009** | 1 button `Begin` (122,280,142,338); 2 statText `^0` (8,8,24,298); 3 statText `^1` (33,8,113,298); 4 icon 1000 — **never shown** |
| **1029** | 1 button `Lock It!` (96,232,116,290); 2 button `Don't Lock` (96,150,116,224) 74×20; 3 statText `Wait!!!  If you lock this house, you will not be able to edit it in the future!  You should keep an unlocked copy.  This is your only warning!`; 4 icon 1060 |
| **1030** | 1 button `Okay`; 2 statText `Where is Sound Manager 3.0?  I highly recommend you install it.  It's fast, it's unobtrusive, and it's the law!`; 3 icon 1070 — **never shown** |
| **1031** | 1 button `Bye!` (48,202,68,260); 2 statText `Glider PRO doesn't know what the word "Print" means.  Sorry.`; 3 icon 900 |
| **1032** | 1 button `Cancel` (61,214,81,272); 2 button `Clear All` (61,134,81,206) 72×20; 3 button `All But #1` (61,54,81,126) 72×20; 4 statText `Do what?  You can clear all but the highest score, all the scores, or cancel this operation.`; 5 icon 910 |
| **1036** | 1 button `Okay`; 2 statText `Where is your PICT?  I'm going to drop back to the Simple room background and see if that comes up.`; 3 icon 1071 |
| **1037** | 1 button `Okay` (96,220,116,278); 2 statText `This feature is not in the Demo version of Glider PRO™.  Glider PRO™ is published by Casady & Greene, Inc.  Call them at (408) 484-9228 during normal business hours (Pacific time, USA).` — **compiled out** |
| **1038** | 1 button `Whatever` (81,186,101,266) 80×20; 2 statText `Okay, with your monitor & color depth, there isn't enough memory.  In order to run Glider PRO™, the music won't be loaded.`; 3 icon 1011 |
| **1039** | 1 button `Whatever`; 2 statText `…the music & sounds won't be loaded.`; 3 icon 1011 |
| **1040** | 1 button `Okay` (44,217,64,273) 56×20; 2 statText `Some preference changes will not take effect until Glider PRO™ is re-launched.` (8,8,40,273) |
| **1041** | 1 button `Save First` (48,174,68,254) 80×20; 2 button `Don't Save` (48,82,68,162) 80×20; 3 statText `Do you want to save the state of the game before quitting?` (8,8,40,209); 4 icon 1072 |
| **1042** | 1 button `Quit` (80,190,100,248); 2 button `Restore` (80,124,100,182); 3 statText `You switched the number of colors!  You must either Quit Glider PRO™ or Restore your monitor to its original colors.`; 4 icon 130 |
| **1044** | 1 button `Okay` (100,234,120,292); 2 statText `This saved game was saved for the house "^0".  "^1" is the current house open.  Load "^0" before using this saved game.`; 3 icon 1006 |
| **1046** | 1 button `So What?` (64,180,84,252) 72×20; 2 statText `If you resume a saved game, you are ineligible to get on the high scores for that game.` (8,8,56,216); 3 icon 1073 |

### 5.5 `PICT` and icon assets used by the shell

Decoded dimensions of every UI-relevant `PICT`:

| `PICT` | W×H | Use |
|---|---|---|
| 150 | 63×63 | About "Okay" diamond, not highlit (`kOkayButtPICTNotHiLit`) |
| 151 | 63×63 | About "Okay" diamond, highlit (`kOkayButtPICTHiLit`) |
| 153 | 372×100 | About title art |
| **1000** | **640×460** | `kSplash8BitPICT` — the splash screen |
| 1001 | 431×32 | Load House header, **1-bit** (`kLoadTitlePict1` = 1001, `GliderPRO/Sources/SelectHouse.c:29`) — the one `DITL` 1000 actually references |
| 1002 | 257×32 | Load House header, **8-bit** (`kLoadTitlePict8` = 1002, `GliderPRO/Sources/SelectHouse.c:30`) |
| 1003 | 32×32 | default house icon, 1-bit (`kDefaultHousePict1`) |
| 1004 | 32×32 | default house icon, 8-bit (`kDefaultHousePict8`) |
| 1005 | 313×32 | (editor dialog header) |
| **1006** | **333×32** | Display Prefs header |
| 1007 | 385×32 | (editor dialog header) |
| **1008** | **316×32** | Sound Prefs header |
| 1009 | 76×28 | (editor) |
| 1010 | 32×380 | (editor tool palette strip) |
| 1011 | 360×216 | (editor) |
| **1012** | **316×32** | Controls Prefs header |
| **1013** | **289×32** | Preferences main header |
| **1014** | **316×32** | Brains Prefs header |
| **1015** | **214×54** | "Paused" overlay, Esc variant |
| **1016** | **214×54** | "Paused" overlay, Tab variant |
| **1017** | **256×64** | "N stars remaining" (plural), `kStarsRemainingPICT` |
| **1018** | **256×64** | "1 star remaining" (singular), `kStarRemainingPICT` |
| 1019 | 96×44 | (editor) |
| 1020 | 96×44 | (editor) |
| **1021** | **640×460** | a second full-screen splash — **referenced by no constant** in `GliderDefines.h` |
| 1022 | 279×32 | (editor dialog header) |
| 1023 | 280×32 | Go To Room dialog header |
| 1988 | 25×256 | (game art) |
| 1989 | 32×448 | (game art) |
| 1990 | 32×448 | (game art) |
| **1991** | **330×30** | banner bottom **mask** (1-bit), `kBannerPageBottomMask` |
| **1992** | **330×30** | banner bottom, `kBannerPageBottomPICT` |
| **1993** | **330×190** | banner top, `kBannerPageTopPICT` |
| **1994** | **332×30** | high-scores header, `kHighScoresPictID` |
| **1995** | **640×460** | `kStarPictID` — high-scores / game-over starfield backdrop |
| 1996 | 32×66 | (game art) |
| **1997** | **1536×20** | `kScoreboardPictID` — the in-game scoreboard strip |
| **1998** | **332×30** | high-scores header **mask**, `kHighScoresMaskID` |
| 1999 | 512×44 | (game art) |

`ICON` resource IDs present: 130, 140, 150, 160, 170, 180, 900, 910, 1000, 1001, 1004, 1005,
1006, 1008, 1010, 1011, 1012, 1013, 1020, 1021, 1022, 1030, 1031, 1032, 1040, 1041, 1042, 1043,
1050, 1051, 1060, 1070, 1071, 1072, 1073 (35 total).

`cicn` (colour icon) IDs add: **1014, 1015, 1016, 1017** (the *inverted* Preferences buttons —
`kInvertedSettingsIcon 1014` + `who`), **1033** (Louder), **1034** (Softer), **1052** (grayed
scroll-up arrow), **1053** (grayed scroll-down arrow), and 2000. Every `DrawCIcon` target in
`Settings.c` and `SelectHouse.c` therefore exists in the fork — verified by cross-referencing
the `cicn` ID list against the code's literals.

`icl4`/`icl8`/`ics#`/`ics4`/`ics8` + `ICN#` + `BNDL` + 6 `FREF`s are the Finder file-type icons
for the application and its four document types (`gliH`, `gliP`, `gliS`, `gliG`).

### 5.6 `vers` resources

| `vers` | Bytes | Numeric | Short | Long |
|---|---|---|---|---|
| 1 | 62 | `01 12 80 00 00 00` | `1.1.2` | `Glider PRO™ 1.1.2\r© 1994-95 Casady & Greene, Inc.` |
| 2 | 44 | `01 12 80 00 00 00` | `1.1.2` | `© 1994-95 Casady & Greene, Inc.` |

The numeric version is BCD: major `0x01`, minor/bugfix `0x12` (1.2 → "1.1.2" with the
`shortVersion` string overriding), stage `0x80` (final), non-release `0x00`, region `0x0000`.

**Version discrepancy:** `GliderPRO/Sources/Main.c:3` labels the source "1.0.4", `DITL` 150
item 6 says `© 1994-2000`, and the `vers` resources say 1.1.2 / © 1994-95. The `.r` resource
dump and the `Sources` tree therefore come from different builds. The About box reads its
version string from `vers` 1 at runtime (§10.3), so on the shipping binary it displayed
`Glider PRO™ 1.1.2 © 1994-95 Casady & Greene, Inc.` while the code says 1.0.4.
---

## 6. Preferences

### 6.1 The `prefsInfo` record — exact 226-byte layout

Declared at `GliderPRO/Headers/Externs.h:233-267`, inside
`#pragma options align=mac68k` … `#pragma options align=reset`
(`GliderPRO/Headers/Externs.h:231`, `:269`). One field is commented out:
`// long encrypted, fakeLong;` at `GliderPRO/Headers/Externs.h:240` — so an 8-byte hole that a
1.0.x-era build might have had is **not** present.

Verified by compiling the exact declaration with `#pragma pack(2)` and `long` forced to
`int32_t`; observed `sizeof(prefsInfo) = 226`:

| Off (dec) | Off (hex) | Size | Type | Field | Meaning |
|---|---|---|---|---|---|
| 0 | `0x00` | 33 | `Str32` | `wasDefaultName` | last-played house file name |
| 33 | `0x21` | 16 | `Str15` | `wasLeftName` | display name of the "left" key |
| 49 | `0x31` | 16 | `Str15` | `wasRightName` | "right" key name |
| 65 | `0x41` | 16 | `Str15` | `wasBattName` | "battery" key name |
| 81 | `0x51` | 16 | `Str15` | `wasBandName` | "bands" key name |
| 97 | `0x61` | 16 | `Str15` | `wasHighName` | high-score player name |
| 113 | `0x71` | 32 | `Str31` | `wasHighBanner` | high-score banner text |
| **145** | `0x91` | **1** | — | **(alignment pad)** | **uninitialized garbage on disk** |
| 146 | `0x92` | 4 | `long` | `wasLeftMap` | key-map offset for "left" |
| 150 | `0x96` | 4 | `long` | `wasRightMap` | key-map offset for "right" |
| 154 | `0x9A` | 4 | `long` | `wasBattMap` | key-map offset for "battery" |
| 158 | `0x9E` | 4 | `long` | `wasBandMap` | key-map offset for "bands" |
| 162 | `0xA2` | 2 | `short` | `wasVolume` | 0–7 |
| **164** | `0xA4` | 2 | `short` | **`prefVersion`** | must equal `kPrefsVersion` |
| 166 | `0xA6` | 2 | `short` | `wasMaxFiles` | max houses to enumerate, 12–500 |
| 168 | `0xA8` | 2 | `short` | `wasEditH` | editor window left |
| 170 | `0xAA` | 2 | `short` | `wasEditV` | editor window top |
| 172 | `0xAC` | 2 | `short` | `wasMapH` | map window left |
| 174 | `0xAE` | 2 | `short` | `wasMapV` | map window top |
| 176 | `0xB0` | 2 | `short` | `wasMapWide` | map window width in rooms |
| 178 | `0xB2` | 2 | `short` | `wasMapHigh` | map window height in rooms |
| 180 | `0xB4` | 2 | `short` | `wasToolsH` | tools window left |
| 182 | `0xB6` | 2 | `short` | `wasToolsV` | tools window top |
| 184 | `0xB8` | 2 | `short` | `wasLinkH` | link window left |
| 186 | `0xBA` | 2 | `short` | `wasLinkV` | link window top |
| 188 | `0xBC` | 2 | `short` | `wasCoordH` | coordinate window left |
| 190 | `0xBE` | 2 | `short` | `wasCoordV` | coordinate window top |
| 192 | `0xC0` | 2 | `short` | `isMapLeft` | map scroll: leftmost room |
| 194 | `0xC2` | 2 | `short` | `isMapTop` | map scroll: topmost room |
| 196 | `0xC4` | 2 | `short` | `wasNumNeighbors` | 1, 3 or 9 |
| 198 | `0xC6` | 2 | `short` | `wasDepthPref` | 0/1/2 (see below) |
| 200 | `0xC8` | 2 | `short` | `wasToolGroup` | current editor tool mode 1–9 |
| 202 | `0xCA` | 2 | `short` | `smWarnings` | Sound Manager nag count |
| 204 | `0xCC` | 2 | `short` | `wasFloor` | editor: current floor |
| 206 | `0xCE` | 2 | `short` | `wasSuite` | editor: current suite |
| 208 | `0xD0` | 1 | `Boolean` | `wasZooms` | zoom window transitions |
| 209 | `0xD1` | 1 | `Boolean` | `wasMusicOn` | music currently loaded/on |
| 210 | `0xD2` | 1 | `Boolean` | `wasAutoEdit` | auto-open Room Info on room change |
| 211 | `0xD3` | 1 | `Boolean` | `wasDoColorFade` | "beautiful opening color fade" |
| 212 | `0xD4` | 1 | `Boolean` | `wasMapOpen` | map window open |
| 213 | `0xD5` | 1 | `Boolean` | `wasToolsOpen` | tools window open |
| 214 | `0xD6` | 1 | `Boolean` | `wasCoordOpen` | coordinate window open |
| 215 | `0xD7` | 1 | `Boolean` | `wasQuickTrans` | quicker room transitions |
| 216 | `0xD8` | 1 | `Boolean` | `wasIdleMusic` | play music on the splash |
| 217 | `0xD9` | 1 | `Boolean` | `wasGameMusic` | play music during the game |
| 218 | `0xDA` | 1 | `Boolean` | `wasEscPauseKey` | true = Esc pauses, false = Tab |
| 219 | `0xDB` | 1 | `Boolean` | `wasDoAutoDemo` | auto-demo after 2 min idle |
| 220 | `0xDC` | 1 | `Boolean` | `wasScreen2` | run on the second monitor |
| 221 | `0xDD` | 1 | `Boolean` | `wasDoBackground` | do background tasks |
| 222 | `0xDE` | 1 | `Boolean` | `wasHouseChecks` | error-check houses on load |
| 223 | `0xDF` | 1 | `Boolean` | `wasPrettyMap` | draw the pretty (rendered) map |
| 224 | `0xE0` | 1 | `Boolean` | `wasBitchDialogs` | show the "create" confirm dialogs |
| **225** | `0xE1` | **1** | — | **(tail pad to even size)** | **uninitialized garbage on disk** |
| | | **226** | | | total |

Both pad bytes (offsets 145 and 225) are written to disk **uninitialized**: `WriteOutPrefs`
declares `prefsInfo thePrefs;` on the stack (`GliderPRO/Sources/Main.c:210`), never zeroes it,
assigns exactly the 50 named fields, and `WritePrefs` transfers `sizeof(*thePrefs)` = 226 bytes
(`GliderPRO/Sources/Prefs.c:127`). A Go port must write two defined bytes there (0 is fine —
nothing reads them) but must not shrink the record to 224.

`wasDepthPref` values (`GliderPRO/Headers/GliderDefines.h:43-45`):

| Constant | Value | Meaning |
|---|---|---|
| `kSwitchIfNeeded` | 0 | use the current monitor depth if it is 4- or 8-bit |
| `kSwitchTo256Colors` | 1 | always switch to 8-bit (256 colours) |
| `kSwitchTo16Grays` | 2 | always switch to 4-bit grayscale |

### 6.2 Storage: file location, type, creator, version

`GliderPRO/Sources/Prefs.c:18-24`:

| Constant | Value |
|---|---|
| `kPrefCreatorType` | `'ozm5'` |
| `kPrefFileType` | `'gliP'` |
| `kPrefFileName` | `"\pGlider Prefs"` |
| `kDefaultPrefFName` | `"\pPreferences"` (used as the `CheckFileError` display name) |
| `kPrefsStringsID` | 160 (`STR#` 160, single string `"Preferences"`) |
| `kNewPrefsAlertID` | 160 (`ALRT` 160) |
| `kPrefsFNameIndex` | 1 |

`GliderPRO/Sources/Main.c:16`:

```c
#define kPrefsVersion  0x0034      // = 52 decimal
```

`GetPrefsFPath` (`GliderPRO/Sources/Prefs.c:59-71`):

```c
theErr = FindFolder(kOnSystemDisk, kPreferencesFolderType, kCreateFolder,
                    systemVolRef, prefDirID);
```

So the file is `<boot volume>/System Folder/Preferences/Glider Prefs`, creator `'ozm5'`, type
`'gliP'`, data fork exactly 226 bytes, no resource fork.

Two helpers exist but are **never called**:

* `CanUseFindFolder` (`GliderPRO/Sources/Prefs.c:39-57`) — `Gestalt(gestaltFindFolderAttr,
  &theFeature)` then `BitTst(&theFeature, 31 - gestaltFindFolderPresent)`.
* `CreatePrefsFolder` (`GliderPRO/Sources/Prefs.c:73-95`) — `GetIndString(folderName,
  kPrefsStringsID /*160*/, kPrefsFNameIndex /*1*/)` then `PBDirCreate` — the pre-System-7
  fallback.

### 6.3 Read / write / delete algorithms

`SavePrefs(prefsInfo *thePrefs, short versionNow)` (`GliderPRO/Sources/Prefs.c:148-164`):

```
1. thePrefs->prefVersion = versionNow;                 // Prefs.c:153 — the ONLY writer
2. if (!GetPrefsFPath(&prefsName, &volRef, &dirID)) return false;
3. return WritePrefs(&prefsName, volRef, dirID, thePrefs);
```

`WritePrefs` (`GliderPRO/Sources/Prefs.c:97-146`):

```
 1. theErr = FSMakeFSSpec(volRef, dirID, fileName, &theSpecs);
 2. if (theErr == fnfErr)                               // file does not exist yet
        theErr = FSpCreate(&theSpecs, kPrefCreatorType /*'ozm5'*/,
                                      kPrefFileType    /*'gliP'*/, smSystemScript);
    if (theErr != noErr) { CheckFileError(theErr, "\pPreferences"); return false; }
 3. theErr = FSpOpenDF(&theSpecs, fsRdWrPerm, &fileRefNum);
    if (theErr != noErr) { CheckFileError(...); return false; }
 4. byteCount = sizeof(*thePrefs);                      // 226
 5. theErr = FSWrite(fileRefNum, &byteCount, thePrefs);
    if (theErr != noErr) { FSClose; CheckFileError(...); return false; }
 6. theErr = FSClose(fileRefNum);
    if (theErr != noErr) { CheckFileError(...); return false; }
 7. return true;
```

Note there is no `SetEOF`: if a longer prefs file already existed, the tail is left behind.
In practice the size never changes within a version.

`LoadPrefs(prefsInfo *thePrefs, short versionNeed)` (`GliderPRO/Sources/Prefs.c:240-271`):

```
1. if (!GetPrefsFPath(&prefsName, &volRef, &dirID)) return false;
2. theErr = ReadPrefs(&prefsName, volRef, dirID, thePrefs);
3. if (theErr == eofErr) {                              // file shorter than 226 bytes
       BringUpDeletePrefsAlert();                       // ALRT 160
       DeletePrefs(&prefsName, volRef, dirID);
       return false;
   } else if (theErr != noErr) {
       return false;                                    // includes fnfErr: silent
   }
4. if (thePrefs->prefVersion != versionNeed) {          // Prefs.c:261
       BringUpDeletePrefsAlert();                       // ALRT 160
       DeletePrefs(&prefsName, volRef, dirID);
       return false;
   }
5. return true;
```

`ReadPrefs` (`GliderPRO/Sources/Prefs.c:166-218`):

```
1. theErr = FSMakeFSSpec(volRef, dirID, fileName, &theSpecs);
   if (theErr == fnfErr) return theErr;                 // silent: first launch
2. theErr = FSpOpenDF(&theSpecs, fsRdWrPerm, &fileRefNum);
3. byteCount = sizeof(*thePrefs);                       // 226
4. theErr = FSRead(fileRefNum, &byteCount, thePrefs);   // eofErr if the file is short
5. FSClose(fileRefNum);
```

`DeletePrefs` (`GliderPRO/Sources/Prefs.c:220-238`): `FSMakeFSSpec` + `FSpDelete`.

`BringUpDeletePrefsAlert` (`GliderPRO/Sources/Prefs.c:273-280`):

```c
InitCursor();
// CenterAlert(kNewPrefsAlertID);      // commented out at Prefs.c:278
whoCares = Alert(kNewPrefsAlertID /*160*/, nil);
```

`ALRT` 160 text: `You have a new Preferences file.  All preferences have been set to their
defaults.` So bumping `kPrefsVersion` is deliberately destructive: **the old preferences are
deleted, not migrated.**

### 6.4 `ReadInPrefs` — the prefs → globals mapping

`GliderPRO/Sources/Main.c:52-200`. Globals declared at `GliderPRO/Sources/Main.c:24-26`:

```c
short   isVolume, wasVolume;
short   isDepthPref, dataResFile, numSMWarnings;
Boolean quitting, doZooms, quickerTransitions, isUseSecondScreen;
```

If `LoadPrefs(&thePrefs, kPrefsVersion)` succeeds:

| Prefs field | Global set |
|---|---|
| `wasDefaultName` | `thisHouseName` (via `PasStringCopy`) |
| `wasLeftName` | `leftName` |
| `wasRightName` | `rightName` |
| `wasBattName` | `batteryName` |
| `wasBandName` | `bandName` |
| `wasHighName` | `highName` |
| `wasHighBanner` | `highBanner` |
| `wasLeftMap` | `theGlider.leftKey` |
| `wasRightMap` | `theGlider.rightKey` |
| `wasBattMap` | `theGlider.battKey` |
| `wasBandMap` | `theGlider.bandKey` |
| `wasVolume` | `isVolume` |
| `wasDepthPref` | `isDepthPref` |
| `wasMusicOn` | `isMusicOn` |
| `wasZooms` | `doZooms` |
| `wasQuickTrans` | `quickerTransitions` |
| `wasDoColorFade` | `isDoColorFade` |
| `wasIdleMusic` | `isPlayMusicIdle` |
| `wasGameMusic` | `isPlayMusicGame` |
| `wasHouseChecks` | `isHouseChecks` |
| `wasMaxFiles` | `maxFiles` — then clamped, see below |
| `wasEditH` / `wasEditV` | `isEditH` / `isEditV` |
| `wasMapH` / `wasMapV` | `isMapH` / `isMapV` |
| `wasMapWide` / `wasMapHigh` | `mapRoomsWide` / `mapRoomsHigh` |
| `wasToolsH` / `wasToolsV` | `isToolsH` / `isToolsV` |
| `wasLinkH` / `wasLinkV` | `isLinkH` / `isLinkV` |
| `wasCoordH` / `wasCoordV` | `isCoordH` / `isCoordV` |
| `isMapLeft` / `isMapTop` | `mapLeftRoom` / `mapTopRoom` |
| `wasFloor` / `wasSuite` | `wasFloor` / `wasSuite` |
| `smWarnings` | `numSMWarnings` |
| `wasAutoEdit` | `autoRoomEdit` |
| `wasMapOpen` / `wasToolsOpen` / `wasCoordOpen` | `isMapOpen` / `isToolsOpen` / `isCoordOpen` |
| `wasNumNeighbors` | `numNeighbors` |
| `wasToolGroup` | `toolMode` |
| `wasDoAutoDemo` | `doAutoDemo` |
| `wasEscPauseKey` | `isEscPauseKey` |
| `wasScreen2` | `isUseSecondScreen` — then forced false, see below |
| `wasDoBackground` | `doBackground` |
| `wasPrettyMap` | `doPrettyMap` |
| `wasBitchDialogs` | `doBitchDialogs` |

Sanity clamps applied on load:

```
// GliderPRO/Sources/Main.c:88-89
if ((maxFiles < 12) || (maxFiles > 500)) maxFiles = 12;      // NOTE: falls back to 12, not 48

// GliderPRO/Sources/Main.c:116-117
if (thisMac.numScreens < 2) isUseSecondScreen = false;
```

Common tail, run in **both** the loaded and the defaults path
(`GliderPRO/Sources/Main.c:191-200`):

```
1. if ((numNeighbors > 1) && (thisMac.screen.right <= 512)) numNeighbors = 1;
2. UnivGetSoundVolume(&wasVolume, thisMac.hasSM3);   // remember the SYSTEM volume
3. UnivSetSoundVolume(isVolume, thisMac.hasSM3);     // apply ours
4. isSoundOn = (isVolume != 0);
```

Step 1 means: on a 512-wide (or narrower) screen the multi-room view is forced to a single room
regardless of the stored preference. Steps 2–4 mean Glider PRO **hijacks the system output
volume** for the duration of the session and restores it on quit (§6.13).

### 6.5 The no-prefs defaults

`GliderPRO/Sources/Main.c:124-188` — the `else` branch when `LoadPrefs` fails (first launch, or
after a version mismatch deleted the file):

| Global | Default | Line |
|---|---|---|
| `thisHouseName` | `"\pSlumberland"` | `Main.c:126` |
| `leftName` | `"\plf arrow"` | `Main.c:127` |
| `rightName` | `"\prt arrow"` | `Main.c:128` |
| `batteryName` | `"\pdn arrow"` | `Main.c:129` |
| `bandName` | `"\pup arrow"` | `Main.c:130` |
| `highName` | `"\pYour Name"` | `Main.c:133` |
| `highBanner` | `"\pYour Message Here"` | `Main.c:134` |
| `theGlider.leftKey` | `kLeftArrowKeyMap` = 124 | |
| `theGlider.rightKey` | `kRightArrowKeyMap` = 123 | |
| `theGlider.battKey` | `kDownArrowKeyMap` = 122 | |
| `theGlider.bandKey` | `kUpArrowKeyMap` = 121 | |
| `isVolume` | `UnivGetSoundVolume(...)` then clamped: `if (< 1) = 1; else if (> 3) = 3;` | |
| `isDepthPref` | `kSwitchIfNeeded` (0) | |
| `isSoundOn` | `true` | |
| `isMusicOn` | `true` | |
| `isPlayMusicIdle` | `true` | |
| `isPlayMusicGame` | `true` | |
| `isHouseChecks` | `true` | |
| `doZooms` | `true` | |
| `quickerTransitions` | `false` | |
| `numNeighbors` | `9` | |
| `isDoColorFade` | `true` | |
| `maxFiles` | `48` | `Main.c:156` |
| `willMaxFiles` | `48` | |
| `isEditH` | `3` | |
| `isEditV` | `41` | |
| `isMapH` | `3` | |
| `isMapV` | `100` | |
| `mapRoomsWide` | `15` | |
| `mapRoomsHigh` | `4` | |
| `isToolsH` | `100` | |
| `isToolsV` | `35` | |
| `isLinkH` | `50` | |
| `isLinkV` | `80` | |
| `isCoordH` | `50` | |
| `isCoordV` | `204` | |
| `mapLeftRoom` | `60` | |
| `mapTopRoom` | `50` | |
| `wasFloor` | `0` | |
| `wasSuite` | `0` | |
| `numSMWarnings` | `0` | |
| `autoRoomEdit` | `true` | |
| `isMapOpen` | `true` | |
| `isToolsOpen` | `true` | |
| `isCoordOpen` | `false` | |
| `toolMode` | `kBlowerMode` (1) | |
| `doAutoDemo` | `true` | |
| `isEscPauseKey` | `false` (⇒ **Tab** pauses) | |
| `isUseSecondScreen` | `false` | |
| `doBackground` | `false` | |
| `doPrettyMap` | **`false`** | `Main.c:187` |
| `doBitchDialogs` | `true` | |

The initial-volume clamp `[1,3]` is notable: on a fresh install the game *lowers* a system volume
above 3 down to 3, and raises a muted system (0) to 1.

### 6.6 `WriteOutPrefs`

`GliderPRO/Sources/Main.c:208-282`:

```
 1. UnivGetSoundVolume(&isVolume, thisMac.hasSM3);      // capture whatever it is NOW
 2. copy every global back into the corresponding thePrefs field
    (the inverse of the §6.4 table — all 50 named fields)
 3. thePrefs.wasMaxFiles = willMaxFiles;                // Main.c:244 — NOT maxFiles!
 4. if (!SavePrefs(&thePrefs, kPrefsVersion)) SysBeep(1);   // Main.c:275-276
 5. UnivSetSoundVolume(wasVolume, thisMac.hasSM3);      // Main.c:278 — restore the SYSTEM volume
```

Step 3 is the mechanism by which the "Maximum houses displayed" setting takes effect only after
a relaunch: `maxFiles` (used by `DoDirSearch`) keeps the value loaded at startup for the whole
session, while `willMaxFiles` (edited in the Brains pane) is what gets persisted. Hence
`ALRT` 1040 (§6.12).

`prefVersion` is *not* written here — `SavePrefs` sets it (`GliderPRO/Sources/Prefs.c:153`).

### 6.7 The Preferences main dialog (`DLOG`/`DITL` 1012)

`GliderPRO/Sources/Settings.c:17-21` dialog IDs:

| Constant | Value |
|---|---|
| `kMainPrefsDialID` | 1012 |
| `kDisplayPrefsDialID` | 1017 |
| `kSoundPrefsDialID` | 1018 |
| `kControlPrefsDialID` | 1023 |
| `kBrainsPrefsDialID` | 1024 |

Main-panel item constants (`GliderPRO/Sources/Settings.c:21-24`, plus `Settings.c:1397`):

| Constant | Value | Item |
|---|---|---|
| `kOkayButton` | 1 | `Okay` |
| `kDisplayButton` | 3 | Display icon |
| `kSoundButton` | 4 | Sounds icon |
| `kControlsButton` | 5 | Controls icon |
| `kBrainsButton` | 6 | Brains icon |
| `kAllDefaultsButton` | 11 | `All Defaults` |

Globals (`GliderPRO/Sources/Settings.c:83-91`):

```c
Rect    prefButton[4], controlRects[4];
Str15   leftName, rightName, batteryName, bandName;
Str15   tempLeftStr, tempRightStr, tempBattStr, tempBandStr;
long    tempLeftMap, tempRightMap, tempBattMap, tempBandMap;
short   whichCtrl, wasDepthPref;
Boolean wasFade, wasIdle, wasPlay, wasTransit, wasZooms, wasBackground;
Boolean wasEscPauseKey, wasDemos, wasScreen2, nextRestartChange, wasErrorCheck;
Boolean wasPrettyMap, wasBitchDialogs;
```

`DoSettingsMain` (`GliderPRO/Sources/Settings.c:1395-1472`):

```
 1. prefsFilterUPP = NewModalFilterUPP(PrefsFilter)
 2. BringUpDialog(&prefDlg, kMainPrefsDialID /*1012*/)
 3. for (i = 0; i < 4; i++) {
        GetDialogItemRect(prefDlg, i + kDisplayButton /*3..6*/, &prefButton[i]);
        InsetRect(&prefButton[i], -4, -4);         // grow the icon rect by 4 px all round
    }
 4. nextRestartChange = false
 5. leaving = false
 6. while (!leaving) {
        ModalDialog(prefsFilterUPP, &itemHit);
        switch (itemHit) {
            case kOkayButton:       leaving = true; break;
            case kDisplayButton:    FlashSettingsButton(0); DoDisplayPrefs();  break;
            case kSoundButton:      FlashSettingsButton(1); DoSoundPrefs();
                                    FlushEvents(everyEvent, 0);               break;
            case kControlsButton:   FlashSettingsButton(2); DoControlPrefs();  break;
            case kBrainsButton:     FlashSettingsButton(3);
                                    if ((OptionKeyDown()) && (!houseUnlocked)) {
                                        houseUnlocked = true;             // Settings.c:1448-1453
                                        changeLockStateOfHouse = true;
                                        saveHouseLocked = false;
                                    }
                                    DoBrainsPrefs();                          break;
            case kAllDefaultsButton: SetAllDefaults();                        break;
        }
        // each sub-panel returns, then the main dialog is redrawn by its own updateEvt
    }
 7. DisposeDialog(prefDlg); DisposeModalFilterUPP(prefsFilterUPP)
 8. if (nextRestartChange) BitchAboutChanges();          // ALRT 1040
```

**Secret feature:** holding Option while clicking the Brains button unlocks a locked house
(`GliderPRO/Sources/Settings.c:1448-1453`). `changeLockStateOfHouse = true` and
`saveHouseLocked = false` mean the unlock is persisted on the next house write. There is no
visual indication that this happened other than the House menu becoming usable.

`UpdateSettingsMain` (`GliderPRO/Sources/Settings.c:1280-1303`):

```
1. DrawDialog(theDialog); DrawDefaultButton(theDialog);
2. for (i = 0; i < 4; i++) {
       GetIndString(theStr, 129, i + 1);                // STR# 129 "Prefmain"
       DrawDialogUserText(theDialog, i + 7 /*items 7..10*/, theStr, false);
   }
3. for (i = 0; i < 4; i++) ColorFrameRect(&prefButton[i], kRedOrangeColor8);
```

`STR#` 129 ("Prefmain", `GliderPRO/Glider PRO.r:381`) — the four button captions:

| Index | String |
|---|---|
| 1 | `Display` |
| 2 | `Sounds` |
| 3 | `Controls` |
| 4 | `Brains` |

`FlashSettingsButton(short who)` (`GliderPRO/Sources/Settings.c:1265-1278`):

```c
#define kNormalSettingsIcon    1010
#define kInvertedSettingsIcon  1014
DrawCIcon(kInvertedSettingsIcon + who, prefButton[who].left + 4, prefButton[who].top + 4);
DelayTicks(8);
DrawCIcon(kNormalSettingsIcon   + who, prefButton[who].left + 4, prefButton[who].top + 4);
```

`prefButton[who]` was inset by −4, so `+4` puts the 32×32 icon back at the original item
rect. `cicn` 1014/1015/1016/1017 are the inverted variants of `ICON`/`cicn` 1010/1011/1012/1013.
`DrawCIcon(id, h, v)` (`GliderPRO/Sources/Utilities.c:393`) does
`GetCIcon(id); SetRect(&r,0,0,32,32); OffsetRect(&r,h,v); PlotCIcon(&r, icon); DisposeCIcon`.

`PrefsFilter` (`GliderPRO/Sources/Settings.c:1305-1393`):

| Event | Response |
|---|---|
| keyDown Return / Enter | `FlashDialogButton(dial, kOkayButton)`, `*item = kOkayButton`, `true` |
| keyDown `B`/`b` | `*item = kBrainsButton` (6) |
| keyDown `C`/`c` | `*item = kControlsButton` (5) |
| keyDown `D`/`d` | `*item = kDisplayButton` (3) |
| keyDown `S`/`s` | `*item = kSoundButton` (4) |
| mouseDown | `testPt = event->where; GlobalToLocal(&testPt);` then for `i = 0..3`, `if (PtInRect(testPt, &prefButton[i])) { *item = kDisplayButton + i; return true; }` |
| updateEvt on `mainWindow` | `UpdateMainWindow()`, return `true` |
| updateEvt on the dialog | `BeginUpdate`, `UpdateSettingsMain(dial)`, `EndUpdate`, `event->what = nullEvent`, return `false` |

**There is no Escape handling in `PrefsFilter`** — the main Preferences dialog has no Cancel
button (`DITL` 1012 item 2 is the header picture), so Esc does nothing there.

The explicit `mouseDown` hit-testing exists because the four buttons are `icon` items, not
`button` controls; the Dialog Manager would report them, but only for the *exact* 32×32 rect,
whereas Glider PRO wants the 40×40 inset-by-−4 hot zone.
### 6.8 Display pane (`DLOG`/`DITL` 1017)

`DLOG` 1017: bounds `(40,40,280,373)` = 333×240, `procID` 1 (`dBoxProc`), `visible = 0`
(shown by `BringUpDialog`'s `ShowWindow`), `goAwayFlag = 0`, `itemsID` 1017.

Item constants (`kDisplay1Item`..`kUseScreen2Item` at `GliderPRO/Sources/Settings.c:26-35`;
`kOkayButton` = 1 and `kCancelButton` = 2 come from `GliderPRO/Headers/Externs.h:22-23`). Captions
below are the literal `DITL` 1017 strings, byte-verified against `Glider PRO.r`:

| Constant | Item | Kind | Caption |
|---|---|---|---|
| `kOkayButton` | 1 | button | `Okay` |
| `kCancelButton` | 2 | button | `Cancel` |
| `kDisplay1Item` | 3 | icon | 1-room view (`ICON` 1020) |
| `kDisplay3Item` | 4 | icon | 3-room view (`ICON` 1021) |
| `kDisplay9Item` | 5 | icon | 9-room view (`ICON` 1022) |
| — | 6 | statText (disabled) | `Number of Rooms to Display:\r(the less rooms, the faster)` |
| — | 7 | picture (disabled) | `PICT` 1006 (333×32 title bar) |
| — | 8 | userItem | group frame (framed red-orange) |
| `kDoColorFadeItem` | 9 | checkBox | `Beautiful opening color fade` |
| `kCurrentDepth` | 10 | radioButton | `Use current depth when possible` |
| `k256Depth` | 11 | radioButton | `Always play in 256 colors` |
| `k16Depth` | 12 | radioButton | `Always play in 16 grays` |
| — | 13 | userItem | group frame (framed red-orange) |
| — | 14 | userItem | group frame (framed red-orange) |
| `kDispDefault` | 15 | button | `Defaults` |
| `kUseQDItem` | 16 | checkBox | `Use Quickdraw™ (slower)` — **dead** (`DoDisplayPrefs` case body commented out, `GliderPRO/Sources/Settings.c:1203-1206`) |
| `kUseScreen2Item` | 17 | checkBox | `Run on second monitor` |

Note the item order: the static text is item **6** and the `PICT` header is item **7**, not the other
way round.

The three depth radios map to `wasDepthPref` by subtraction:
`wasDepthPref = itemHit - kCurrentDepth` (`GliderPRO/Sources/Settings.c:1191`), which is exactly
`kSwitchIfNeeded`/`kSwitchTo256Colors`/`kSwitchTo16Grays` = 0/1/2.

`DisplayDefaults` (`GliderPRO/Sources/Settings.c:886-892`):

```c
numNeighbors  = 9;
wasDepthPref  = kSwitchIfNeeded;   // 0
wasFade       = true;
wasScreen2    = false;
```

`FrameDisplayIcon` (`GliderPRO/Sources/Settings.c:896-924`) — draws the selection box around
whichever of the three room-count icons is current, using the *current* pen colour (callers set
white to erase and red to draw):

```
1. switch (numNeighbors) {
       case 1:  GetDialogItemRect(theDialog, kDisplay1Item /*3*/, &theRect); break;
       case 3:  GetDialogItemRect(theDialog, kDisplay3Item /*4*/, &theRect); break;
       default: GetDialogItemRect(theDialog, kDisplay9Item /*5*/, &theRect); break;
   }
2. theRect.left -= 3; theRect.top += 0; theRect.right += 3; theRect.bottom -= 1;
3. FrameRect(&theRect);
4. InsetRect(&theRect, 1, 1);
5. FrameRect(&theRect);
```

Note the asymmetric fudge: 3 px out on the left and right, 0 on top, 1 px *in* on the bottom.
Two nested `FrameRect`s = a 2-px-thick box.

`DisplayUpdate` (`GliderPRO/Sources/Settings.c:926-945`):

```
1. DrawDialog(theDialog); DrawDefaultButton(theDialog);
2. SetDialogItemValue(theDialog, kDoColorFadeItem, (short)wasFade);
3. SelectFromRadioGroup(theDialog, kCurrentDepth + wasDepthPref, kCurrentDepth, k16Depth);
   // SetDialogItemValue(theDialog, kUseQDItem, (short)wasQD);   // commented out, Settings.c:934
4. SetDialogItemValue(theDialog, kUseScreen2Item, (short)wasScreen2);
5. ForeColor(redColor); FrameDisplayIcon(theDialog); ForeColor(blackColor);
6. FrameDialogItemC(theDialog,  8, kRedOrangeColor8);
   FrameDialogItemC(theDialog, 13, kRedOrangeColor8);
   FrameDialogItemC(theDialog, 14, kRedOrangeColor8);
```

`DisplayFilter` (`GliderPRO/Sources/Settings.c:947-1104`):

| Key | Effect |
|---|---|
| Return / Enter | flash + `*item = kOkayButton` |
| Esc | flash + `*item = kCancelButton` |
| Left arrow | cycle room count backwards: `1→9`, `3→1`, `9→3` (sets `*item` to the corresponding icon item) |
| Right arrow | cycle forwards: `1→3`, `3→9`, `9→1` |
| Up arrow | cycle depth radio: `kSwitchIfNeeded→k16Depth`, `kSwitchTo256Colors→kCurrentDepth`, `kSwitchTo16Grays→k256Depth` |
| Down arrow | the reverse of Up |
| `1` | `*item = kDisplay1Item` |
| `3` | `*item = kDisplay3Item` |
| `9` | `*item = kDisplay9Item` |
| `B`/`b` | `*item = kDoColorFadeItem` ("**B**eautiful color fade") |
| `D`/`d` | `*item = kDispDefault`, `FlashDialogButton(dial, kDispDefault)` |
| `R`/`r` | `*item = kUseScreen2Item` but `FlashDialogButton(dial, kUseQDItem)` — **mismatch bug**, `GliderPRO/Sources/Settings.c:1067-1071` |
| `U`/`u` | `*item = kUseQDItem` (a no-op, the case body is commented out) |

`DoDisplayPrefs` (`GliderPRO/Sources/Settings.c:1106-1216`):

```
 1. displayFilterUPP = NewModalFilterUPP(DisplayFilter)
 2. BringUpDialog(&prefDlg, kDisplayPrefsDialID /*1017*/)
 3. if (!thisMac.can8Bit) { MyDisableControl(prefDlg, kDoColorFadeItem);
                            MyDisableControl(prefDlg, k256Depth); }
 4. if (!thisMac.can4Bit)        MyDisableControl(prefDlg, k16Depth)
 5. if (thisMac.numScreens < 2)  MyDisableControl(prefDlg, kUseScreen2Item)
 6. wasNeighbors = numNeighbors            // the ONLY saved-for-cancel value
    wasFade      = isDoColorFade
    wasDepthPref = isDepthPref
    wasScreen2   = isUseSecondScreen
 7. loop ModalDialog:
      kOkayButton:      isDoColorFade = wasFade
                        isDepthPref   = wasDepthPref
                        if (isUseSecondScreen != wasScreen2) nextRestartChange = true
                        isUseSecondScreen = wasScreen2
                        leaving = true
      kCancelButton:    numNeighbors = wasNeighbors; leaving = true
      kDisplay1Item:    white FrameDisplayIcon; numNeighbors = 1;
                        red FrameDisplayIcon; black
      kDisplay3Item:    same with numNeighbors = 3, but ONLY if thisMac.screen.right > 512
      kDisplay9Item:    same with numNeighbors = 9, but ONLY if thisMac.screen.right > 512
      kDoColorFadeItem: wasFade = !wasFade; SetDialogItemValue(...)
      kCurrentDepth / k256Depth / k16Depth:
                        wasDepthPref = itemHit - kCurrentDepth
                        SelectFromRadioGroup(prefDlg, itemHit, kCurrentDepth, k16Depth)
      kDispDefault:     white FrameDisplayIcon; black; DisplayDefaults(); DisplayUpdate(prefDlg)
      kUseQDItem:       /* commented out */
      kUseScreen2Item:  wasScreen2 = !wasScreen2; SetDialogItemValue(...)
 8. DisposeDialog; DisposeModalFilterUPP
```

`numNeighbors` is the only setting mutated live (so the red selection box can move), hence the
only one restored on Cancel. Changing the second-monitor setting is the one Display change that
requires a relaunch.

The `thisMac.screen.right > 512` guard means on a 512×342 Mac Plus-sized screen the 3- and 9-room
views cannot be selected at all — matching the `ReadInPrefs` clamp
(`GliderPRO/Sources/Main.c:191-192`).

### 6.9 Sound pane (`DLOG`/`DITL` 1018)

`DLOG` 1018: bounds `(40,40,216,356)` = 316×176, `dBoxProc`, `visible = 0`, `itemsID` 1018.

Item constants (`kSofterItem`..`kSoundDefault` at `GliderPRO/Sources/Settings.c:36-41`). Captions
and item kinds below are byte-verified against `DITL` 1018 in `Glider PRO.r`:

| Constant | Item | Kind | Caption |
|---|---|---|---|
| `kOkayButton` | 1 | button | `Okay` |
| `kCancelButton` | 2 | button | `Cancel` |
| — | 3 | icon (disabled) | `ICON` 1030 — the speaker decoration at (52,244,84,276) |
| `kSofterItem` | 4 | icon | Softer (`ICON` 1032 normal / `cicn` 1034 pressed) |
| `kLouderItem` | 5 | icon | Louder (`ICON` 1031 normal / `cicn` 1033 pressed) |
| — | 6 | statText (disabled) | `Volume:` |
| `kVolNumberItem` | 7 | statText (disabled) | the numeric volume readout (empty in the resource) |
| `kIdleMusicItem` | 8 | checkBox | `Play music when idle` |
| `kPlayMusicItem` | 9 | checkBox | `Play music during game` |
| — | 10 | statText (disabled) | `Set the game volume and background music options.` |
| — | 11 | userItem | group frame (framed red-orange) |
| — | 12 | picture (disabled) | `PICT` 1008 (316×32 title bar) |
| `kSoundDefault` | 13 | button | `Defaults` |

Note that unlike the other panes, the `PICT` title bar is the **last** item (12) here, and item 3 is
a disabled decoration icon, not the picture.

**Volume model.** The volume lives in the *system* output volume, not in a Glider variable; the
pane reads and writes it directly on every keystroke and click, and the value is only copied into
`isVolume` in `WriteOutPrefs` (`GliderPRO/Sources/Main.c:212`).

`UnivGetSoundVolume` (`GliderPRO/Sources/Utilities.c:742-764`):

```
1. GetDefaultOutputVolume(&longVol);
2. *volume = LoWord(longVol) / 0x0024;      // 36 decimal
3. if (*volume < 0) *volume = 0; else if (*volume > 7) *volume = 7;
```

`UnivSetSoundVolume` (`GliderPRO/Sources/Utilities.c:766-790`):

```
1. if (volume < 0) volume = 0; else if (volume > 7) volume = 7;
2. longVol = (long)volume * 0x0025;                    // 37 decimal
3. if (longVol > 0x00000100) longVol = 0x00000100;     // clamp to unity gain
4. longVol = longVol + (longVol << 16);                // duplicate into left AND right channel
5. SetDefaultOutputVolume(longVol);
```

So the 0–7 scale maps to per-channel gains `0, 37, 74, 111, 148, 185, 222, 259→clamped 256`
(`0x100` = unity). The asymmetric divisor (36) vs multiplier (37) means a round-trip
set-then-get is stable: `7*37 = 259` clamps to `256`, `256/36 = 7`; `3*37 = 111`, `111/36 = 3`.

**The "11" quirk.** Level 7 is displayed as `11` — a *This Is Spinal Tap* joke. Three places do
it: `UpdateSettingsSound` (`GliderPRO/Sources/Settings.c:621-624`, `if (howLoudNow >= 7)`),
`SoundFilter`'s digit handler (`GliderPRO/Sources/Settings.c:702-705`, `if (newVolume == 7L)`;
the `k0KeyASCII`..`k7KeyASCII` cases it belongs to start at `:693`),
and `DoSoundPrefs`'s Louder case (`GliderPRO/Sources/Settings.c:836-839`). The Softer case
(`GliderPRO/Sources/Settings.c:822`) always prints the raw number, which is correct because
stepping down from 7 lands on 6.

`SoundDefaults` (`GliderPRO/Sources/Settings.c:599-608`):

```c
wasIdle = true;  wasPlay = true;
SetDialogItemValue(theDialog, kIdleMusicItem, (short)wasIdle);
SetDialogItemValue(theDialog, kPlayMusicItem, (short)wasPlay);
UnivSetSoundVolume(3, thisMac.hasSM3);
SetDialogNumToStr(theDialog, kVolNumberItem, 3L);
HandleSoundMusicChange(3, true);
```

`HandleSoundMusicChange(short newVolume, Boolean sayIt)`
(`GliderPRO/Sources/Settings.c:631-659`):

```
1. isSoundOn = (newVolume != 0);
2. if (wasIdle) {                                  // idle music is enabled
       if (newVolume == 0) StopTheMusic();
       else if (!isMusicOn) {
           theErr = StartMusic();
           if (theErr != noErr) { YellowAlert(kYellowNoMusic, theErr); failedMusic = true; }
       }
   }
3. if ((newVolume != 0) && (sayIt)) PlayPrioritySound(kChord2Sound, kChord2Priority);
```

`kChord2Sound` = 54, `kChord2Priority` = 810 (`GliderPRO/Headers/GliderDefines.h:109`, `:170`) —
the audible feedback chord played on every volume change.

`SoundFilter` (`GliderPRO/Sources/Settings.c:661-756`):

| Key | Effect |
|---|---|
| Return / Enter | flash + `kOkayButton` |
| Esc | flash + `kCancelButton` |
| Up arrow | `*item = kLouderItem` |
| Down arrow | `*item = kSofterItem` |
| `0`–`7` | `newVolume = char - '0'`; write the readout (11 if 7); `UnivSetSoundVolume(newVolume, …)`; `HandleSoundMusicChange(newVolume, true)`; **return `false`** (no item hit) |
| `D`/`d` | `*item = kSoundDefault` + flash |
| `G`/`g` | `*item = kPlayMusicItem` ("play during **g**ame") |
| `I`/`i` | `*item = kIdleMusicItem` ("**i**dle") |
| mouseDown | `false` |
| updateEvt | `UpdateSettingsSound(dial)`, swallow the event, `false` |

`DoSoundPrefs` (`GliderPRO/Sources/Settings.c:758-882`):

```
 1. soundFilterUPP = NewModalFilterUPP(SoundFilter)
 2. BringUpDialog(&prefDlg, kSoundPrefsDialID /*1018*/)
 3. UnivGetSoundVolume(&wasLoudness, thisMac.hasSM3)       // remember for Cancel
 4. wasIdle = isPlayMusicIdle; wasPlay = isPlayMusicGame
    SetDialogItemValue(kIdleMusicItem, wasIdle); SetDialogItemValue(kPlayMusicItem, wasPlay)
 5. loop ModalDialog:
      kOkayButton:   isPlayMusicIdle = wasIdle; isPlayMusicGame = wasPlay; leaving = true
                     UnivGetSoundVolume(&tempVolume, …); isSoundOn = (tempVolume != 0)
      kCancelButton: UnivSetSoundVolume(wasLoudness, …)
                     HandleSoundMusicChange(wasLoudness, false)
                     if (isPlayMusicIdle != wasIdle) {
                         if (isPlayMusicIdle) { if (wasLoudness != 0) StartMusic()... }
                         else StopTheMusic();
                     }
                     leaving = true
      kSofterItem:   UnivGetSoundVolume(&tempVolume, …)
                     if (tempVolume > 0) {
                         GetDialogItemRect(prefDlg, kSofterItem, &tempRect)
                         DrawCIcon(1034, tempRect.left, tempRect.top)   // pressed art
                         tempVolume--
                         SetDialogNumToStr(prefDlg, kVolNumberItem, tempVolume)
                         UnivSetSoundVolume(tempVolume, …)
                         HandleSoundMusicChange(tempVolume, true)
                         InvalWindowRect(GetDialogWindow(prefDlg), &tempRect)  // restore art
                         DelayTicks(8)
                     }
      kLouderItem:   same, mirrored: if (tempVolume < 7), DrawCIcon(1033), tempVolume++,
                     print 11 when it reaches 7
      kIdleMusicItem: wasIdle = !wasIdle; SetDialogItemValue(...)
                      if (wasIdle) { if (volume != 0) StartMusic()... } else StopTheMusic()
      kPlayMusicItem: wasPlay = !wasPlay; SetDialogItemValue(...)
      kSoundDefault:  SoundDefaults(prefDlg)
 6. DisposeDialog; DisposeModalFilterUPP
```

Cancel restores the volume but *not* `isSoundOn` (which `HandleSoundMusicChange` recomputes from
`wasLoudness`, so it is in fact consistent). Note that Cancel does **not** undo a music
start/stop caused by the volume having been dragged to 0 and back, beyond what
`HandleSoundMusicChange(wasLoudness, false)` does.

After `DoSoundPrefs` returns, `DoSettingsMain` calls `FlushEvents(everyEvent, 0)`
(`GliderPRO/Sources/Settings.c:1437`) — the only panel that does, presumably to swallow key
repeats from the 0–7 shortcuts.

### 6.10 Controls pane (`DLOG`/`DITL` 1023)

`DLOG` 1023: bounds `(40,40,216,356)` = 316×176, `dBoxProc`, `visible = 0`, `itemsID` 1023.

Item constants (`kRightControl`..`kTABPausesRadio` at `GliderPRO/Sources/Settings.c:42-48`).
Item kinds and captions are byte-verified against `DITL` 1023 in `Glider PRO.r`:

| Constant | Item | Kind | Purpose |
|---|---|---|---|
| `kOkayButton` | 1 | button | `Okay` |
| `kCancelButton` | 2 | button | `Cancel` |
| — | 3 | userItem | group frame (framed red-orange) |
| — | 4 | picture (disabled) | `PICT` 1012 (316×32 title bar) |
| `kRightControl` | 5 | icon | `ICON` 1040 — "move right" key slot |
| `kLeftControl` | 6 | icon | `ICON` 1041 — "move left" key slot |
| `kBattControl` | 7 | icon | `ICON` 1042 — "use battery" key slot |
| `kBandControl` | 8 | icon | `ICON` 1043 — "use rubber bands" key slot |
| — | 9 | userItem | name display for item 5 (`kRightControl + 4`) |
| — | 10 | userItem | name display for item 6 |
| — | 11 | userItem | name display for item 7 |
| — | 12 | userItem | name display for item 8 |
| `kControlDefaults` | 13 | button | `Defaults` |
| `kESCPausesRadio` | 14 | radioButton | `Esc Pauses Game` |
| `kTABPausesRadio` | 15 | radioButton | `Tab Pauses Game` |

`whichCtrl` (0..3) selects which of the four slots is armed. The mapping
`whichCtrl = itemHit - kRightControl` gives 0 = Right, 1 = Left, 2 = Battery, 3 = Bands
(`GliderPRO/Sources/Settings.c:572`).

`SetControlsToDefaults` (`GliderPRO/Sources/Settings.c:333-346`):

```c
PasStringCopy("\plf arrow", tempLeftStr);   tempLeftMap  = kLeftArrowKeyMap;   // 124
PasStringCopy("\prt arrow", tempRightStr);  tempRightMap = kRightArrowKeyMap;  // 123
PasStringCopy("\pdn arrow", tempBattStr);   tempBattMap  = kDownArrowKeyMap;   // 122
PasStringCopy("\pup arrow", tempBandStr);   tempBandMap  = kUpArrowKeyMap;     // 121
wasEscPauseKey = false;
SelectFromRadioGroup(theDialog, kTABPausesRadio, kESCPausesRadio, kTABPausesRadio);
```

`UpdateControlKeyName` (`GliderPRO/Sources/Settings.c:350-356`) — draws all four names, inverting
the one that is armed:

```c
DrawDialogUserText(theDialog, kRightControl + 4 /*9*/,  tempRightStr, (whichCtrl == 0));
DrawDialogUserText(theDialog, kLeftControl  + 4 /*10*/, tempLeftStr,  (whichCtrl == 1));
DrawDialogUserText(theDialog, kBattControl  + 4 /*11*/, tempBattStr,  (whichCtrl == 2));
DrawDialogUserText(theDialog, kBandControl  + 4 /*12*/, tempBandStr,  (whichCtrl == 3));
```

`UpdateSettingsControl` (`GliderPRO/Sources/Settings.c:360-378`):

```
1. DrawDialog(theDialog)
2. PenSize(2, 2)
3. ForeColor(whiteColor); for (i=0;i<4;i++) FrameRect(&controlRects[i]);   // clear all
4. ForeColor(redColor);   FrameRect(&controlRects[whichCtrl]);             // mark the armed one
5. ForeColor(blackColor); PenNormal()
6. UpdateControlKeyName(theDialog)
7. FrameDialogItemC(theDialog, 3, kRedOrangeColor8)
```

`controlRects[i]` are the four icon item rects, each `InsetRect(-3,-3)`
(`GliderPRO/Sources/Settings.c:517-520`).

**Key capture.** `ControlFilter` (`GliderPRO/Sources/Settings.c:380-500`) has four
near-identical `case whichCtrl` arms. For the armed slot:

```
1. wasKeyMap = (long)GetKeyMapFromMessage(event->message);
2. if (wasKeyMap is any of the OTHER THREE slots' maps
       || wasKeyMap == kTabKeyMap   /*55*/
       || wasKeyMap == kEscKeyMap   /*50*/
       || wasKeyMap == kDeleteKeyMap/*52*/) {
       if (wasKeyMap == kEscKeyMap) { FlashDialogButton(dial, kCancelButton);
                                      *item = kCancelButton; return true; }
       else SysBeep(1);
   } else {
       GetKeyName(event->message, temp<Slot>Str);
       temp<Slot>Map = wasKeyMap;
   }
3. UpdateControlKeyName(dial);
4. return false;
```

So Escape both cancels the dialog and is a reserved key; Tab and Delete are reserved (Tab because
it is the alternate pause key and the Dialog Manager's focus key, Delete because the editor uses
it); duplicates beep. Note the check compares against the *other three* temps only, so re-typing
the key already assigned to the armed slot is accepted (a harmless no-op re-assignment).

**`GetKeyMapFromMessage` — the key identity actually stored in prefs.**
`GliderPRO/Sources/Utilities.c:522-530`:

```c
theVirtual = (message & keyCodeMask) >> 8;       // raw/virtual key code, 0x00..0x7F
offset     = KeyMapOffsetFromRawKey((char)theVirtual);
```

`KeyMapOffsetFromRawKey` (`GliderPRO/Sources/Utilities.c:504-517`):

```c
hiByte = rawKeyCode & 0xF0;
loByte = rawKeyCode & 0x0F;
if (loByte <= 0x07) theOffset = hiByte + (0x07 - loByte);
else                theOffset = hiByte + (0x17 - loByte);
return theOffset;
```

This converts an Apple *virtual key code* into a bit offset into the 16-byte `KeyMap` array
returned by `GetKeys`, which is what the game tests with `BitTst`. The transform reverses the low
three bits within each group of eight and is its own inverse. Verified against the constants in
`GliderPRO/Headers/Externs.h:129-163`:

| Key | Raw/virtual code | Computed offset | Declared constant |
|---|---|---|---|
| Left arrow | `0x7B` | `0x70 + (0x17-0x0B)` = 124 | `kLeftArrowKeyMap` 124 |
| Right arrow | `0x7C` | `0x70 + (0x17-0x0C)` = 123 | `kRightArrowKeyMap` 123 |
| Down arrow | `0x7D` | `0x70 + (0x17-0x0D)` = 122 | `kDownArrowKeyMap` 122 |
| Up arrow | `0x7E` | `0x70 + (0x17-0x0E)` = 121 | `kUpArrowKeyMap` 121 |
| Tab | `0x30` | `0x30 + (0x07-0x00)` = 55 | `kTabKeyMap` 55 |
| Esc | `0x35` | `0x30 + (0x07-0x05)` = 50 | `kEscKeyMap` 50 |
| Delete | `0x33` | `0x30 + (0x07-0x03)` = 52 | `kDeleteKeyMap` 52 |
| Space | `0x31` | `0x30 + (0x07-0x01)` = 54 | `kSpaceBarMap` 54 |
| Command | `0x37` | `0x30 + (0x07-0x07)` = 48 | `kCommandKeyMap` 48 |
| Shift | `0x38` | `0x30 + (0x17-0x08)` = 63 | `kShiftKeyMap` 63 |
| `A` | `0x00` | `0x00 + 7` = 7 | `kAKeyMap` 7 |
| `X` | `0x07` | `0x00 + 0` = 0 | `kXKeyMap` 0 |
| `Z` | `0x06` | `0x00 + 1` = 1 | `kZKeyMap` 1 |
| `C` | `0x08` | `0x00 + (0x17-0x08)` = 15 | `kCKeyMap` 15 |
| `R` | `0x0F` | `0x00 + (0x17-0x0F)` = 8 | `kRKeyMap` 8 |

**This is the single most important porting detail in the whole preferences record:**
`wasLeftMap`/`wasRightMap`/`wasBattMap`/`wasBandMap` (offsets 146/150/154/158) hold **Mac KeyMap
bit offsets**, not virtual key codes and not ASCII. A Go port that wants to read original
`Glider Prefs` files must run the inverse of `KeyMapOffsetFromRawKey` (which is the same
function) to recover the virtual key code, then map that to its own key enum.

`GetKeyName(long message, StringPtr theName)` (`GliderPRO/Sources/Utilities.c:536-694`) produces
the human-readable label stored in `wasLeftName` etc.:

```
1. theASCII   = message & charCodeMask
   theVirtual = (message & keyCodeMask) >> 8
2. if (theASCII >= kExclamationASCII /*0x21*/ && theASCII <= kZKeyASCII /*0x7A*/) {
       if (theVirtual >= 0x41 && theVirtual <= 0x5C) {     // numeric keypad
           PasStringCopy("\p( )", theName); theName[2] = theASCII;   // e.g. "(5)"
       } else {
           PasStringCopy("\p  key", theName); theName[1] = theASCII; // e.g. "A key"
       }
   }
3. else switch (theASCII) { ... fixed table ... }
```

The fixed table (`GliderPRO/Sources/Utilities.c:558-693`):

| ASCII | Name produced |
|---|---|
| `kHomeKeyASCII` 0x01 | `home` |
| `kEnterKeyASCII` 0x03 | `enter` |
| `kEndKeyASCII` 0x04 | `end` |
| `kHelpKeyASCII` 0x05 | `help` |
| `kDeleteKeyASCII` 0x08 | `delete` |
| `kTabKeyASCII` 0x09 | `tab` |
| `kPageUpKeyASCII` 0x0B | `pg up` |
| `kPageDownKeyASCII` 0x0C | `pg dn` |
| `kReturnKeyASCII` 0x0D | `return` |
| `kFunctionKeyASCII` 0x10 | by virtual code: `0x60`→`F5`, `0x61`→`F6`, `0x62`→`F7`, `0x63`→`F3`, `0x64`→`F8`, `0x65`→`F9`, `0x67`→`F11`, `0x69`→`F13`, `0x6B`→`F14`, `0x6D`→`F10`, `0x6F`→`F12`, `0x71`→`F15`, `0x76`→`F4`, `0x78`→`F2`, `0x7A`→`F1`, default `NumToString(theVirtual)` |
| `kClearKeyASCII` 0x1A | `clear` |
| `kEscapeKeyASCII` 0x1B | `esc`, **or** `clear` when `theVirtual == kClearRawKey` 0x47 |
| `kLeftArrowKeyASCII` 0x1C | `lf arrow` |
| `kRightArrowKeyASCII` 0x1D | `rt arrow` |
| `kUpArrowKeyASCII` 0x1E | `up arrow` |
| `kDownArrowKeyASCII` 0x1F | `dn arrow` |
| `kSpaceBarASCII` 0x20 | `space` |
| `kForwardDeleteASCII` 0x7F | `frwd del` |
| default | `????` |

These names are exactly the four default strings in §6.5 (`lf arrow`, `rt arrow`, `dn arrow`,
`up arrow`).

`DoControlPrefs` (`GliderPRO/Sources/Settings.c:502-596`) — the one panel that does **not** use
`BringUpDialog`, because it must measure the four icon rects before the window becomes visible:

```
 1. controlFilterUPP = NewModalFilterUPP(ControlFilter)
    // CenterDialog(kControlPrefsDialID);           // commented out, Settings.c:510
 2. prefDlg = GetNewDialog(kControlPrefsDialID /*1023*/, nil, kPutInFront)
    if (prefDlg == nil) RedAlert(kErrDialogDidntLoad)
    SetPort((GrafPtr)prefDlg)
 3. for (i = 0; i < 4; i++) {
        GetDialogItemRect(prefDlg, i + kRightControl /*5..8*/, &controlRects[i])
        InsetRect(&controlRects[i], -3, -3)
    }
 4. whichCtrl = 1                                  // "Left" is armed on entry
 5. copy leftName/rightName/batteryName/bandName -> tempLeftStr/tempRightStr/tempBattStr/tempBandStr
    copy theGlider.leftKey/rightKey/battKey/bandKey -> tempLeftMap/tempRightMap/tempBattMap/tempBandMap
    wasEscPauseKey = isEscPauseKey
 6. ShowWindow(GetDialogWindow(prefDlg))
 7. SelectFromRadioGroup(prefDlg, isEscPauseKey ? kESCPausesRadio : kTABPausesRadio,
                         kESCPausesRadio, kTABPausesRadio)
 8. loop ModalDialog:
      kOkayButton:      commit the four names into leftName/rightName/batteryName/bandName,
                        the four maps into theGlider.leftKey/rightKey/battKey/bandKey,
                        isEscPauseKey = wasEscPauseKey; leaving = true
      kCancelButton:    leaving = true             // nothing to undo: all edits were in temps
      kRightControl..kBandControl:
                        PenSize(2,2); white FrameRect(&controlRects[whichCtrl]);
                        whichCtrl = itemHit - kRightControl;
                        red FrameRect(&controlRects[whichCtrl]); black; PenNormal();
                        UpdateControlKeyName(prefDlg)
      kESCPausesRadio / kTABPausesRadio:
                        SelectFromRadioGroup(prefDlg, itemHit, kESCPausesRadio, kTABPausesRadio)
                        wasEscPauseKey = !wasEscPauseKey        // Settings.c:583 — see below
      kControlDefaults: SetControlsToDefaults(prefDlg); UpdateControlKeyName(prefDlg)
 9. DisposeDialog; DisposeModalFilterUPP
```

**Bug worth reproducing or not, your call:** the radio handler at
`GliderPRO/Sources/Settings.c:583` *toggles* `wasEscPauseKey` rather than assigning
`(itemHit == kESCPausesRadio)`. Clicking the radio that is already selected therefore desyncs the
boolean from the visible radio state; clicking it twice resyncs it. A faithful port should
either replicate this or (better) note it as a deliberate deviation.

`isEscPauseKey` is consumed by `DoPause` (`GliderPRO/Sources/Input.c:77`), which draws
`kEscPausePictID` or `kTabPausePictID` into a `214×54` rect centered in `houseRect` (`PICT`
1015/1016 are both 214×54).

### 6.11 Brains pane (`DLOG`/`DITL` 1024)

`DLOG` 1024: bounds `(40,40,232,356)` = 316×192, `dBoxProc`, `visible = 0`, `itemsID` 1024.

Item constants (`kMaxFilesItem`..`kBrainsDefault` at `GliderPRO/Sources/Settings.c:49-52`,
`kDoDemoCheck`..`kDoBitchDlgsCheck` at `:53-57`). `DITL` 1024 has **15** items; captions below are
the literal resource strings, byte-verified:

| Constant | Item | Kind | Caption |
|---|---|---|---|
| `kOkayButton` | 1 | button | `Okay` |
| `kCancelButton` | 2 | button | `Cancel` |
| — | 3 | userItem | group frame (framed red-orange) |
| — | 4 | picture (disabled) | `PICT` 1014 (316×32 title bar) |
| `kMaxFilesItem` | 5 | editText | max number of house files to list (empty in the resource) |
| — | 6 | statText (disabled) | `Maximum houses displayed (12-500): \r(larger = more memory)` |
| `kQuickTransitCheck` | 7 | checkBox | `Quick Transitions` |
| `kDoZoomsCheck` | 8 | checkBox | `Zoom Windows` |
| `kBrainsDefault` | 9 | button | `Defaults` |
| `kDoDemoCheck` | 10 | checkBox | `Automatic Demo` |
| `kDoBackgroundCheck` | 11 | checkBox | `Background Tasks` |
| `kDoErrorCheck` | 12 | checkBox | `Error-Check House` (right column, left = 156) |
| `kDoPrettyMapCheck` | 13 | checkBox | `Use "Pretty Map"` (right column) |
| `kDoBitchDlgsCheck` | 14 | checkBox | `Do Create Dialog` (right column) |
| — | 15 | statText (disabled) | `Editor Options:` — heading over items 12-14 |

The seven checkboxes are edited into the seven `was*` temps and committed only on Okay:

| Item | Temp | Live global |
|---|---|---|
| `kQuickTransitCheck` 7 | `wasTransit` | `quickerTransitions` |
| `kDoZoomsCheck` 8 | `wasZooms` | `doZooms` |
| `kDoDemoCheck` 10 | `wasDemos` | `doAutoDemo` |
| `kDoBackgroundCheck` 11 | `wasBackground` | `doBackground` |
| `kDoErrorCheck` 12 | `wasErrorCheck` | `isHouseChecks` |
| `kDoPrettyMapCheck` 13 | `wasPrettyMap` | `doPrettyMap` |
| `kDoBitchDlgsCheck` 14 | `wasBitchDialogs` | `doBitchDialogs` |

`SetBrainsToDefaults` (`GliderPRO/Sources/Settings.c:106-127`):

```c
SetDialogNumToStr(theDialog, kMaxFilesItem, 24L);   // Settings.c:108 — 24, NOT 48
#ifdef powerc
    wasTransit = false;                             // Settings.c:110-111
#else
    wasTransit = true;                              // Settings.c:112-113
#endif
wasZooms        = true;
wasDemos        = true;
wasBackground   = false;
wasErrorCheck   = true;
wasPrettyMap    = true;                             // Settings.c:118 — true here
wasBitchDialogs = true;
SetDialogItemValue(theDialog, kQuickTransitCheck,  (short)wasTransit);
SetDialogItemValue(theDialog, kDoZoomsCheck,       (short)wasZooms);
SetDialogItemValue(theDialog, kDoDemoCheck,        (short)wasDemos);
SetDialogItemValue(theDialog, kDoBackgroundCheck,  (short)wasBackground);
SetDialogItemValue(theDialog, kDoErrorCheck,       (short)wasErrorCheck);
SetDialogItemValue(theDialog, kDoPrettyMapCheck,   (short)wasPrettyMap);
SetDialogItemValue(theDialog, kDoBitchDlgsCheck,   (short)wasBitchDialogs);
```

Two discrepancies against the other two defaults paths, both real and both citable:

* max-files default is **24** here, **48** in `SetAllDefaults` (`GliderPRO/Sources/Settings.c:1225`)
  and in the no-prefs path (`GliderPRO/Sources/Main.c:156-157`, which sets both `maxFiles` and
  `willMaxFiles`);
* it only writes the *text field*, never `willMaxFiles`, so clicking Brains-Defaults and then
  Cancel discards it, whereas clicking Defaults then Okay reads 24 back out of the field.

`quickerTransitions` defaults differ **by architecture**: false on PowerPC, true on 68k. The
`#ifdef powerc` is a compile-time switch, so a Go port must pick one; the shipping PowerPC
build (and the no-prefs path, `GliderPRO/Sources/Main.c:153`) uses `false`.

`UpdateSettingsBrains` (`GliderPRO/Sources/Settings.c:131-142`):

```
1. DrawDialog(theDialog); DrawDefaultButton(theDialog)
2. SetDialogNumToStr(theDialog, kMaxFilesItem, (long)willMaxFiles)
3. SelectDialogItemText(theDialog, kMaxFilesItem, 0, 1024)      // select all
4. FrameDialogItemC(theDialog, 3, kRedOrangeColor8)
```

`BrainsFilter` (`GliderPRO/Sources/Settings.c:144-225`):

| Key | Effect |
|---|---|
| Return / Enter | flash + `kOkayButton` |
| Esc | flash + `kCancelButton` |
| `A`/`a` | `*item = kDoDemoCheck` (**a**uto-demo) |
| `B`/`b` | `*item = kDoBackgroundCheck` |
| `D`/`d` | `*item = kBrainsDefault` + flash |
| `E`/`e` | `*item = kDoErrorCheck` |
| `Q`/`q` | `*item = kQuickTransitCheck` |
| `Z`/`z` | `*item = kDoZoomsCheck` |
| any other | `false` (so digits reach the `kMaxFilesItem` edit field) |
| mouseDown | `false` |
| updateEvt | `UpdateSettingsBrains(dial)`, swallow, `false` |

**`kDoPrettyMapCheck` (13) and `kDoBitchDlgsCheck` (14) have no keyboard shortcut** — the two
newest checkboxes were never wired into the filter.

`DoBrainsPrefs` (`GliderPRO/Sources/Settings.c:227-331`):

```
 1. brainsFilterUPP = NewModalFilterUPP(BrainsFilter)
 2. BringUpDialog(&prefDlg, kBrainsPrefsDialID /*1024*/)
 3. wasMaxFiles      = willMaxFiles                  // remember for Cancel / change detection
    wasTransit       = quickerTransitions
    wasZooms         = doZooms
    wasDemos         = doAutoDemo
    wasBackground    = doBackground
    wasErrorCheck    = isHouseChecks
    wasPrettyMap     = doPrettyMap
    wasBitchDialogs  = doBitchDialogs
    (SetDialogItemValue for all seven; SetDialogNumToStr(kMaxFilesItem, willMaxFiles))
 4. loop ModalDialog:
      kOkayButton:
          GetDialogNumFromStr(prefDlg, kMaxFilesItem, &tempLong)
          if (tempLong > 500) tempLong = 500;
          else if (tempLong < 12) tempLong = 12;
          willMaxFiles = tempLong
          if (willMaxFiles != wasMaxFiles) nextRestartChange = true
          quickerTransitions = wasTransit
          doZooms            = wasZooms
          doAutoDemo         = wasDemos
          doBackground       = wasBackground
          isHouseChecks      = wasErrorCheck
          doPrettyMap        = wasPrettyMap
          doBitchDialogs     = wasBitchDialogs
          leaving = true
      kCancelButton:  willMaxFiles = wasMaxFiles; leaving = true
      kQuickTransitCheck: wasTransit = !wasTransit; SetDialogItemValue(...)
      kDoZoomsCheck:      wasZooms = !wasZooms; SetDialogItemValue(...)
      kDoDemoCheck:       wasDemos = !wasDemos; SetDialogItemValue(...)
      kDoBackgroundCheck: wasBackground = !wasBackground; SetDialogItemValue(...)
      kDoErrorCheck:      wasErrorCheck = !wasErrorCheck; SetDialogItemValue(...)
      kDoPrettyMapCheck:  wasPrettyMap = !wasPrettyMap; SetDialogItemValue(...)
      kDoBitchDlgsCheck:  wasBitchDialogs = !wasBitchDialogs; SetDialogItemValue(...)
      kBrainsDefault:     SetBrainsToDefaults(prefDlg)
 5. DisposeDialog; DisposeModalFilterUPP
```

Max-files valid range is **[12, 500]** here; `ReadInPrefs` uses the same bounds test but snaps
out-of-range values to **12**, not to the nearest bound (`GliderPRO/Sources/Main.c:88-89`).

Effects of the seven flags elsewhere in the program, for completeness:

| Flag | Where it is read |
|---|---|
| `quickerTransitions` | room-transition code (`RoomInfo`/`Transitions`); `extern` at `GliderPRO/Sources/MainWindow.c:48` |
| `doZooms` | **nothing** — `ZoomBetweenWindows` is commented out (`GliderPRO/Sources/MainWindow.c:277-299`) |
| `doAutoDemo` | `GliderPRO/Sources/Events.c:537` (auto-demo trigger) |
| `doBackground` | room drawing: whether the background `PICT` is composited |
| `isHouseChecks` | `HouseIO` house validation pass (localized strings 24–34) |
| `doPrettyMap` | the editor's map window rendering |
| `doBitchDialogs` | the "are you sure" confirmation dialogs in the editor |

### 6.12 `SetAllDefaults` and `nextRestartChange`

`SetAllDefaults` (`GliderPRO/Sources/Settings.c:1221-1263`), reached from the
`All Defaults` button (item 11) on the main Preferences dialog. Unlike the per-pane Defaults
buttons this writes the **live globals** directly, immediately, with no Okay/Cancel:

```c
willMaxFiles       = 48;              // Settings.c:1225
doZooms            = true;
doAutoDemo         = true;
doBackground       = false;
isHouseChecks      = true;
doPrettyMap        = true;
doBitchDialogs     = true;
PasStringCopy("\plf arrow", leftName);      theGlider.leftKey  = kLeftArrowKeyMap;   // 124
PasStringCopy("\prt arrow", rightName);     theGlider.rightKey = kRightArrowKeyMap;  // 123
PasStringCopy("\pdn arrow", batteryName);   theGlider.battKey  = kDownArrowKeyMap;   // 122
PasStringCopy("\pup arrow", bandName);      theGlider.bandKey  = kUpArrowKeyMap;     // 121
isEscPauseKey      = false;
isPlayMusicIdle    = true;
isPlayMusicGame    = true;
UnivSetSoundVolume(3, thisMac.hasSM3);
isSoundOn          = true;
if (!isMusicOn) { theErr = StartMusic(); if (theErr != noErr) {
                      YellowAlert(kYellowNoMusic, theErr); failedMusic = true; } }
numNeighbors       = 9;
quickerTransitions = false;
isDepthPref        = kSwitchIfNeeded;  // 0
isDoColorFade      = true;
```

Note `SetAllDefaults` does **not** touch `isUseSecondScreen` and does **not** set
`nextRestartChange`, so changing `willMaxFiles` from e.g. 200 to 48 via All Defaults silently
takes effect only on the next launch with no warning.

Three-way defaults comparison — every place a default is defined:

| Setting | No-prefs (`Main.c:124-188`) | `SetAllDefaults` (`Settings.c:1221`) | Per-pane Defaults |
|---|---|---|---|
| max house files | `maxFiles = 48`, `willMaxFiles = 48` | `willMaxFiles = 48` | Brains: text field **24** only |
| `quickerTransitions` | `false` | `false` | Brains: `false` on PPC / `true` on 68k |
| `doZooms` | `true` | `true` | Brains: `true` |
| `doAutoDemo` | `true` | `true` | Brains: `true` |
| `doBackground` | `false` | `false` | Brains: `false` |
| `isHouseChecks` | `true` | `true` | Brains: `true` |
| `doPrettyMap` | **`false`** | `true` | Brains: `true` |
| `doBitchDialogs` | `true` | `true` | Brains: `true` |
| left/right/batt/band keys | 124/123/122/121 | 124/123/122/121 | Controls: 124/123/122/121 |
| `isEscPauseKey` | `false` (Tab pauses) | `false` | Controls: `false` |
| `isPlayMusicIdle` | `true` | `true` | Sound: `true` |
| `isPlayMusicGame` | `true` | `true` | Sound: `true` |
| volume | system volume clamped to `[1,3]` | `3` | Sound: `3` |
| `isSoundOn` | `true` | `true` | derived from volume |
| `numNeighbors` | `9` | `9` | Display: `9` |
| `isDepthPref` | `kSwitchIfNeeded` | `kSwitchIfNeeded` | Display: `kSwitchIfNeeded` |
| `isDoColorFade` | `true` | `true` | Display: `true` |
| `isUseSecondScreen` | `false` | *untouched* | Display: `false` |
| `isMusicOn` | `true` | *derived* | — |

`nextRestartChange` (declared `GliderPRO/Sources/Settings.c:92`) is set in exactly two places:

| Trigger | Line |
|---|---|
| `willMaxFiles != wasMaxFiles` on Brains-Okay | `GliderPRO/Sources/Settings.c:270` |
| `isUseSecondScreen != wasScreen2` on Display-Okay | `GliderPRO/Sources/Settings.c:1140` |

It is cleared at `GliderPRO/Sources/Settings.c:1417` and tested at `:1468`, just before the
`BitchAboutChanges` call:

`BitchAboutChanges` (`GliderPRO/Sources/Settings.c:1474-1482`):

```c
#define kChangesEffectAlert  1040
short hitWhat;
InitCursor();
hitWhat = Alert(kChangesEffectAlert /*1040*/, nil);
```

`ALRT` 1040: bounds `(54,96,126,377)` = 281×72, `stages` `0x4444`. Its `DITL` says
`Some of the changes you made will not take effect until the next time you launch Glider PRO™.`
with a single `Okay` button.

### 6.13 Session-scoped side effects a Go port must reproduce (or deliberately not)

1. Glider PRO takes over the **system** output volume for the whole session:
   `ReadInPrefs` saves it into `wasVolume` and applies `isVolume`
   (`GliderPRO/Sources/Main.c:196-198`); `WriteOutPrefs` restores `wasVolume`
   (`GliderPRO/Sources/Main.c:278`). A hard crash therefore leaves the user's Mac at Glider's
   volume. A modern port should use a per-application mixer gain instead.
2. Glider PRO switches the **screen depth** (`isDepthPref` → `HandleDepthSwitching`) and restores
   it with `RestoreColorDepth()` on quit (`GliderPRO/Sources/Main.c:~366`). `thisMac.wasDepth`
   and `thisMac.isDepth` (`GliderPRO/Headers/Environ.h`) hold the before/after values.
3. `maxFiles` vs `willMaxFiles`: the *effective* value is frozen at launch; the *edited* value is
   what persists. Anything else will make the Brains pane feel wrong.
4. `numNeighbors` can be silently forced to 1 at launch on a ≤512-px-wide screen
   (`GliderPRO/Sources/Main.c:191-192`) *without* rewriting the preference — so the stored 9 comes
   back on a bigger monitor.
5. `isUseSecondScreen` is silently forced to `false` at launch when `thisMac.numScreens < 2`
   (`GliderPRO/Sources/Main.c:116-117`), and this *does* get written back on quit, so plugging in
   a second monitor does not restore the preference.
---

## 7. House selection

`GliderPRO/Sources/SelectHouse.c` (676 lines) implements both the *discovery* of house files (a
recursive volume scan, no standard file dialog) and the *Load House* picker.

### 7.1 Constants and globals

`GliderPRO/Sources/SelectHouse.c:20-35`:

| Constant | Value | Meaning |
|---|---|---|
| `kLoadHouseDialogID` | 1000 | the picker `DLOG`/`DITL` |
| `kDispFiles` | 12 | houses shown per page (4 across × 3 down) |
| `kLoadTitlePictItem` | 3 | title-bar picture item |
| `kLoadNameFirstItem` | 5 | first name `userItem` |
| `kLoadNameLastItem` | 16 | last name `userItem` (5..16 = 12 slots) |
| `kLoadIconFirstItem` | 17 | first icon item |
| `kLoadIconLastItem` | 28 | last icon item (17..28 = 12 slots) |
| `kScrollUpItem` | 29 | page-up arrow icon |
| `kScrollDownItem` | 30 | page-down arrow icon |
| `kLoadTitlePict1` | 1001 | 1-bit title bar (431×32); the constant is **not referenced by any code** — but 1001 *is* the resource ID baked into `DITL` 1000 item 3, so it is what actually draws |
| `kLoadTitlePict8` | 1002 | 8-bit title bar (257×32) — **never used** |
| `kDefaultHousePict1` | 1003 | 1-bit generic house icon (32×32) — **never used** |
| `kDefaultHousePict8` | 1004 | 8-bit generic house icon (32×32) — the fallback actually used |
| `kGrayedOutUpArrow` | 1052 | `cicn`, dimmed page-up arrow (the live one is `ICON` 1050) |
| `kGrayedOutDownArrow` | 1053 | `cicn`, dimmed page-down arrow (live: `ICON` 1051) |
| `kMaxExtraHouses` | 8 | slots for houses injected by Apple Events / New House |

`GliderPRO/Sources/SelectHouse.c:45-53`:

```c
Rect       loadHouseRects[12];      // union of each slot's icon + name rect, for InvalWindowRect
FSSpecPtr  theHousesSpecs;          // NewPtr(sizeof(FSSpec) * maxFiles)  == 70 * maxFiles bytes
FSSpec     extraHouseSpecs[8];
long       lastWhenClick;           // double-click timing scratch
Point      lastWhereClick;          // double-click position scratch
short      housesFound, thisHouseIndex, maxFiles, willMaxFiles;
short      housePage, demoHouseIndex, numExtraHouses;
char       fileFirstChar[12];       // upper-cased first letter of each visible name, for type-select
extern UInt32 doubleTime;           // the system double-click interval, in ticks
```

`theHousesSpecs` is allocated once, at `GliderPRO/Sources/StructuresInit2.c:275-278`:

```c
theHousesSpecs = (FSSpecPtr)NewPtr(sizeof(FSSpec) * maxFiles);
if (theHousesSpecs == nil) RedAlert(kErrNoMemory);
```

`sizeof(FSSpec)` is **70** (verified: `short vRefNum` 2 + `long parID` 4 + `Str63 name` 64), so at
the default `maxFiles = 48` the table is 3360 bytes. This is why `maxFiles` is frozen at launch:
the array is sized from it before any UI exists.

### 7.2 Discovery: `DoDirSearch`

`GliderPRO/Sources/SelectHouse.c:557-648`. A **breadth-first, single-volume, depth-capped**
directory walk using `PBGetCatInfo`. It is *not* recursive and it does *not* look at other
volumes.

```
 1. #define kMaxDirectories 32
    long theDirs[32];  for (i=0;i<32;i++) theDirs[i] = 0L;
 2. currentDir = 0; theDirs[0] = thisMac.dirID;   // the application's own directory
    numDirs = 1
 3. theBlock.hFileInfo.ioCompletion = nil
    theBlock.hFileInfo.ioVRefNum    = thisMac.vRefNum
    theBlock.hFileInfo.ioNamePtr    = nameString
 4. while ((currentDir < numDirs) && (currentDir < kMaxDirectories)) {
        count = 1; theErr = noErr;
        while (theErr == noErr) {
            SpinCursor(1)
            theBlock.hFileInfo.ioFDirIndex = count
            theBlock.hFileInfo.ioDirID     = theDirs[currentDir]
            theErr = PBGetCatInfo(&theBlock, false)     // fnfErr ends the enumeration
            if (theErr == noErr) {
                if ((ioFlAttrib & 0x10) == 0x00) {              // it is a FILE
                    if ((ioFlFndrInfo.fdType    == 'gliH') &&
                        (ioFlFndrInfo.fdCreator == 'ozm5') &&
                        (housesFound < maxFiles)) {
                        notherErr = FSMakeFSSpec(thisMac.vRefNum, ioFlParID,
                                                 nameString, &theHousesSpecs[housesFound]);
                        if (notherErr == noErr) housesFound++;
                    }
                } else if ((ioFlAttrib & 0x10) == 0x10) {       // it is a DIRECTORY
                    if (numDirs < kMaxDirectories) { theDirs[numDirs] = ioDirID; numDirs++; }
                }
                count++
            }
        }
        currentDir++
    }
 5. if (housesFound < 1) {
        thisHouseIndex = -1
        YellowAlert(kYellowNoHouses /*5*/, 0)
    } else {
        SortHouseList()
        thisHouseIndex = 0
        for (i = 0; i < housesFound; i++)
            if (EqualString(theHousesSpecs[i].name, thisHouseName, false, true))
                { thisHouseIndex = i; break; }
        PasStringCopy(theHousesSpecs[thisHouseIndex].name, thisHouseName)
        demoHouseIndex = -1
        for (i = 0; i < housesFound; i++)
            if (EqualString(theHousesSpecs[i].name, "\pDemo House", false, true))
                { demoHouseIndex = i; break; }
    }
```

Behavioural details a port must match:

* The scan starts at `thisMac.dirID` (the folder containing the application) and only ever
  descends into directories it *sees while enumerating*, capped at **32 directories total**
  including the start directory. Directories discovered after slot 31 are silently dropped.
* `ioFlAttrib` bit 4 (`0x10`) distinguishes directory from file — this is the standard
  `ioDirFlg` bit.
* Selection is by **Finder type + creator**, not by filename extension: type `'gliH'`, creator
  `'ozm5'`. A Go port has no Finder info; see §Porting notes.
* `EqualString(a, b, false, true)` is *case-insensitive, diacritic-sensitive*. `EqualString(a, b,
  true, true)` (used in `SortHouseList`) is *case-sensitive, diacritic-sensitive*.
* If nothing is found: `thisHouseIndex = -1` and `ALRT` 1006 fires with `STR#` 1006 index 5:
  `There are no houses on this drive!  About your only option is to create your own new house with
  the Editor.`
* `demoHouseIndex` is the index of the file literally named `Demo House`, else −1. That single
  value drives whether Options ▸ Demo… is enabled (§3.5.2).
* `thisHouseName` is *rewritten* from the matched spec, which normalises its case to whatever the
  file system reports.

### 7.3 `SortHouseList`

`GliderPRO/Sources/SelectHouse.c:516-554`. Two phases.

Phase 1 — duplicate removal:

```c
i = 0;
while (i < housesFound) {
    h = i + 1;
    while (h < housesFound) {
        if ((EqualString(theHousesSpecs[i].name, theHousesSpecs[h].name, true, true)) &&
            (theHousesSpecs[i].vRefNum == theHousesSpecs[i].vRefNum) &&    // BUG: [i] vs [i]
            (theHousesSpecs[i].parID   == theHousesSpecs[i].parID))        // BUG: [i] vs [i]
        {
            theHousesSpecs[h] = theHousesSpecs[housesFound - 1];   // swap-with-last removal
            housesFound--;
        }
        h++;
    }
    i++;
}
```

**Two self-comparison bugs at `GliderPRO/Sources/SelectHouse.c:527-528`:** the `vRefNum` and
`parID` tests compare element `i` with itself, so they are always true. The effect is that the
predicate degenerates to "same name (case-sensitive)", and two houses with the same name in
*different folders* are collapsed into one — the second is replaced by the last element and
`housesFound` shrinks. Because the replacement element is not re-examined (`h++` runs
unconditionally), a run of three same-named files can also leak one through. Reproduce or fix
deliberately, but know that the original loses same-named houses in sibling folders.

Phase 2 — bubble sort, ascending by name, using `WhichStringFirst`:

```c
for (i = 0; i < housesFound - 1; i++)
    for (h = 0; h < (housesFound - i - 1); h++)
        if (WhichStringFirst(theHousesSpecs[h].name, theHousesSpecs[h+1].name) == 1)
            swap(theHousesSpecs[h], theHousesSpecs[h+1]);
```

`WhichStringFirst(p1, p2)` (`GliderPRO/Sources/StringUtils.c:34-87`) returns:

| Return | Meaning |
|---|---|
| 0 | the strings are equal (case-insensitively) |
| 1 | `p1` sorts *after* `p2` (so the caller swaps) |
| 2 | `p2` sorts after `p1` |

It compares byte by byte after folding ASCII lowercase to uppercase (`if (c > 0x60 && c < 0x7B) c -= 0x20`),
and on a common prefix the **shorter string sorts first**. Bytes ≥ 0x80 (MacRoman accented
characters) are *not* folded and compare by raw byte value, so `École` sorts after `Zoo`.

### 7.4 `BuildHouseList` and `AddExtraHouse`

`GliderPRO/Sources/SelectHouse.c:650-664`:

```c
void BuildHouseList (void) {
    if (thisMac.hasSystem7) {
        housesFound = 0;
        for (i = 0; i < numExtraHouses; i++) {
            theHousesSpecs[housesFound] = extraHouseSpecs[i];
            housesFound++;
        }
        DoDirSearch();
    }
}
```

So "extra" houses are inserted *first* and then `DoDirSearch` appends and sorts everything
together (extras are subject to the same duplicate-collapse). The `hasSystem7` guard means on a
System 6 machine there is no house list at all — but `main()` already `RedAlert`s on
`!hasSystem7` (`GliderPRO/Sources/Main.c:~300`), so it is unreachable.

`AddExtraHouse` (`GliderPRO/Sources/SelectHouse.c:666-676`):

```c
if (numExtraHouses >= kMaxExtraHouses /*8*/) return;
extraHouseSpecs[numExtraHouses] = *newHouse;
numExtraHouses++;
```

Two call sites:

| Caller | Purpose |
|---|---|
| `GliderPRO/Sources/AppleEvents.c:92` | every `'gliH'` file in an `odoc` Apple Event's document list (drag onto the app icon / double-click a house in the Finder) |
| `GliderPRO/Sources/House.c:93` | the newly created file from `CreateNewHouse` (House ▸ New House…) |

`numExtraHouses` is never reset, so across a session the 8 slots fill permanently.

`DoOpenDocAE` (`GliderPRO/Sources/AppleEvents.c:~84-120`) then, for the *first* `'gliH'` item:
`CloseHouse()`, `PasStringCopy(oneFSS.name, thisHouseName)`, `BuildHouseList()`, `OpenHouse()` +
`ReadHouse()`, `OpenCloseEditWindows()`, `incrementModeTime = TickCount() + kIdleSplashTicks`, and
if in splash or play mode invalidates the exact splash rect
`(splashOriginH+474, splashOriginV+304, +166, +12)` so the "House: …" caption redraws — the same
166×12 rect that Game ▸ Load House… uses (§3.8.3).

### 7.5 Alias resolution

`OpenHouse` (`GliderPRO/Sources/HouseIO.c:176-178`) resolves aliases **in place**:

```c
theErr = ResolveAliasFile(&theHousesSpecs[thisHouseIndex], true, &targetIsFolder, &wasAliased);
if (!CheckFileError(theErr, thisHouseName)) return (false);
```

So a Finder alias whose *own* Finder type is `'gliH'`/`'ozm5'` shows up in the list and is
followed on open, and the resolved spec permanently replaces the alias's spec in the table.

### 7.6 `UpdateLoadDialog` — drawing one page

`GliderPRO/Sources/SelectHouse.c:62-136`:

```
 1. theWindow = GetDialogWindow(theDialog)
    GetWindowBounds(theWindow, kWindowContentRgn, &dialogRect)
    // the old visRgn-poking version is commented out at SelectHouse.c:70-76
 2. DrawDialog(theDialog)
 3. ColorFrameWHRect(8, 39, 413, 184, kRedOrangeColor8)   // left, top, width, height
    // == the rect (39,8,223,421) in (top,left,bottom,right) order
 4. houseStart = housePage
    houseStop  = housesFound
    if ((houseStop - houseStart) > kDispFiles) houseStop = houseStart + kDispFiles
 5. wasResFile = CurResFile(); count = 0
 6. for (i = 0; i < 12; i++) fileFirstChar[i] = 0x7F      // 0x7F = "empty slot" sentinel
 7. for (i = houseStart; i < houseStop; i++) {
        SpinCursor(1)
        GetDialogItemRect(theDialog, kLoadIconFirstItem + i - housePage, &tempRect)
        if (SectRect(&dialogRect, &tempRect, &dummyRect)) {      // only if actually visible
            isResFile = HOpenResFile(theHousesSpecs[i].vRefNum, theHousesSpecs[i].parID,
                                     theHousesSpecs[i].name, fsRdPerm);
            if (isResFile != -1) {
                if (Get1Resource('icl8', -16455) != nil)
                    LargeIconPlot(&tempRect, -16455);           // the file's own custom icon
                else
                    LoadDialogPICT(theDialog, kLoadIconFirstItem + i - housePage,
                                   kDefaultHousePict8 /*1004*/);
                CloseResFile(isResFile);
            } else
                LoadDialogPICT(theDialog, kLoadIconFirstItem + i - housePage,
                               kDefaultHousePict8 /*1004*/);
        }
        fileFirstChar[count] = theHousesSpecs[i].name[1];         // first char of the Pascal name
        if ((fileFirstChar[count] <= 0x7A) && (fileFirstChar[count] > 0x60))
            fileFirstChar[count] -= 0x20;                        // fold a-z -> A-Z
        count++
        DrawDialogUserText(theDialog, kLoadNameFirstItem + i - housePage,
                           theHousesSpecs[i].name,
                           i == (thisHouseIndex + housePage));    // invert the selected one
    }
 8. InitCursor(); UseResFile(wasResFile)
```

Notes:

* **Each house's own custom Finder icon is used**, by opening the house file's resource fork and
  looking for `'icl8'` **ID −16455**. −16455 is the standard Mac OS custom-file-icon family ID
  (`kCustomIconResource`); the whole family is `ICN#`/`icl4`/`icl8`/`ics#`/`ics4`/`ics8` at that
  ID, and `LargeIconPlot` (`GliderPRO/Sources/Utilities.c:379-391`) uses
  `GetIconSuite(&theSuite, theID, svAllLargeData)` + `PlotIconSuite(theRect, atNone, ttNone,
  theSuite)`, i.e. it plots the whole large family with masking. `Get1Resource` (not
  `GetResource`) restricts the search to the just-opened house file.
* The fallback is `PICT` 1004 (32×32, 8-bit) drawn with `DrawPicture` into the item rect.
* `fileFirstChar` is filled **only for slots that were drawn**, indexed by `count` (0-based within
  the page), and the whole array is reset to `0x7F` first. `0x7F` therefore means "no house in
  this slot" for the type-select scan in `LoadFilter`.
* `SpinCursor(1)` animates the `acur` spinning-watch cursor while resource forks are opened —
  opening 12 resource forks over an AppleShare volume was slow.
* Every redraw re-opens up to 12 resource forks. A Go port should cache the icons.
* `ColorFrameWHRect(8, 39, 413, 184, …)` takes (left, top, width, height) — see
  `GliderPRO/Sources/ColorUtils.c`. The erase rect used by paging is `(8, 39, 421, 223)` in
  (left, top, right, bottom) order, i.e. 8 px wider than the frame.

### 7.7 Paging

`PageUpHouses` (`GliderPRO/Sources/SelectHouse.c:138-166`):

```
1. if (housePage < kDispFiles) { SysBeep(1); return; }        // already on page 0
2. housePage -= kDispFiles
   thisHouseIndex = kDispFiles - 1                            // select the LAST slot
3. ShowDialogItem(theDial, kScrollDownItem)                   // down-arrow becomes live again
4. if (housePage < kDispFiles) {                              // we are now on page 0
       GetDialogItemRect(theDial, kScrollUpItem, &tempRect)
       HideDialogItem(theDial, kScrollUpItem)
       DrawCIcon(kGrayedOutUpArrow /*1052*/, tempRect.left, tempRect.top)
   }
5. QSetRect(&tempRect, 8, 39, 421, 223); EraseRect(&tempRect)
   InvalWindowRect(GetDialogWindow(theDial), &tempRect)
```

`PageDownHouses` (`GliderPRO/Sources/SelectHouse.c:168-196`) is the mirror image:

```
1. if (housePage >= (housesFound - kDispFiles)) { SysBeep(1); return; }
2. housePage += kDispFiles;  thisHouseIndex = 0                // select the FIRST slot
3. ShowDialogItem(theDial, kScrollUpItem)
4. if (housePage >= (housesFound - kDispFiles)) {
       HideDialogItem(theDial, kScrollDownItem)
       DrawCIcon(kGrayedOutDownArrow /*1053*/, tempRect.left, tempRect.top)
   }
5. same erase + invalidate
```

The "disabled arrow" idiom is: `HideDialogItem` the live icon item, then hand-draw a dimmed
`cicn` (1052 / 1053) at the same top-left. Since the item is hidden, clicks on it do nothing.
There is no code that *re*-hides the dimmed art — `ShowDialogItem` on the opposite arrow simply
lets `DrawDialog` paint the live icon over it on the next update.

### 7.8 `LoadFilter` — keyboard navigation and double-click detection

`GliderPRO/Sources/SelectHouse.c:198-353`. `screenCount` is recomputed in every arm as
`min(housesFound - housePage, kDispFiles)`, i.e. the number of occupied slots on the current page.

| Key | Effect |
|---|---|
| Return / Enter | flash + `*item = kOkayButton` |
| Esc | flash + `*item = kCancelButton` |
| Page Up (0x0B) | `*item = kScrollUpItem` |
| Page Down (0x0C) | `*item = kScrollDownItem` |
| Up arrow | `thisHouseIndex -= 4`; if `< 0`: `thisHouseIndex += 4; thisHouseIndex = (((screenCount-1)/4)*4) + (thisHouseIndex % 4); if (thisHouseIndex >= screenCount) thisHouseIndex -= 4;` — wraps to the same column in the last *occupied* row |
| Down arrow | `thisHouseIndex += 4`; if `>= screenCount`: `thisHouseIndex %= 4` — wraps to the same column in row 0 |
| Left arrow | `thisHouseIndex--`; if `< 0`: `thisHouseIndex = screenCount - 1` |
| Tab **or** Right arrow | `thisHouseIndex++`; if `>= screenCount`: `thisHouseIndex = 0` |
| `A`–`Z` / `a`–`z` | type-select, see below |
| anything else | return `false` |

All movement arms bracket the change with
`InvalWindowRect(GetDialogWindow(dial), &loadHouseRects[thisHouseIndex])` before and after, so
exactly the old and new slots are repainted (and `UpdateLoadDialog` redraws the whole page —
including re-opening resource forks — because the update event covers the union).

Type-select (`GliderPRO/Sources/SelectHouse.c:290-320`):

```
1. if theChar in (0x41..0x5A) or (0x61..0x7A):
2.     if theChar in (0x61..0x7A): theChar -= 0x20            // fold to upper case
3.     wasIndex = thisHouseIndex; thisHouseIndex = -1; i = 0
4.     do {
           if ((fileFirstChar[i] >= theChar) && (fileFirstChar[i] != 0x7F))
               thisHouseIndex = i;
           i++;
       } while ((thisHouseIndex == -1) && (i < 12));
5.     if (thisHouseIndex == -1) thisHouseIndex = screenCount - 1;   // past the end: last slot
6.     if (wasIndex != thisHouseIndex) { Inval both slots }
7.     return true
```

Because the list is sorted, "first slot whose initial ≥ the typed letter" is a linear search for
the insertion point. It searches **only the current page** — typing `Z` on page 1 of 3 selects
the last item on page 1, it does not page forward. The `!= 0x7F` test skips empty slots.

Double-click detection (`GliderPRO/Sources/SelectHouse.c:322-336`):

```c
case mouseDown:
    lastWhenClick = event->when - lastWhenClick;   // ticks since the previous mouseUP
    SubPt(event->where, &lastWhereClick);          // lastWhereClick -= event->where
    return(false);
case mouseUp:
    lastWhenClick = event->when;                   // remember when/where the click ENDED
    lastWhereClick = event->where;
    return(false);
```

Then in `DoLoadHouse`'s item handling
(`GliderPRO/Sources/SelectHouse.c:~443-450` and `~478-485`):

```c
if (lastWhereClick.h < 0) lastWhereClick.h = -lastWhereClick.h;   // abs()
if (lastWhereClick.v < 0) lastWhereClick.v = -lastWhereClick.v;
if ((lastWhenClick < doubleTime) && (lastWhereClick.h < 5) && (lastWhereClick.v < 5))
    ... open the house and leave ...
```

`doubleTime` is the system double-click interval in ticks (`extern UInt32 doubleTime`,
`GliderPRO/Sources/SelectHouse.c:53`; it is the low-memory global `DoubleTime`). Slop is
**5 pixels** in each axis. Note `SubPt(src, &dst)` computes `dst = dst - src`, so
`lastWhereClick` ends up holding `previousUpPoint - thisDownPoint`; the `abs()` above turns it
into a distance. On the very first click of the dialog `lastWhenClick`/`lastWhereClick` hold
stale values from the previous invocation, so a stray "double click" on entry is possible — the
game does not clear them.

`updateEvt` handling: `BeginUpdate`, `UpdateLoadDialog(dial)`, `EndUpdate`,
`event->what = nullEvent`, return `false`. Unlike the Preferences panels this one does **not**
`SetPort` first, relying on `BringUpDialog` having done it.

### 7.9 `DoLoadHouse`

`GliderPRO/Sources/SelectHouse.c:355-514`:

```
 1. loadFilterUPP = NewModalFilterUPP(LoadFilter)
 2. BringUpDialog(&theDial, kLoadHouseDialogID /*1000*/)
 3. if (housesFound <= kDispFiles) {
        hide + gray BOTH arrows (cicn 1052 and 1053)
    } else {
        if (thisHouseIndex < kDispFiles)               hide + gray the UP arrow
        else if (thisHouseIndex > (housesFound - kDispFiles)) hide + gray the DOWN arrow
    }
 4. wasIndex = thisHouseIndex                          // remember for Cancel
    housePage = (thisHouseIndex / kDispFiles) * kDispFiles
    thisHouseIndex -= housePage                        // now 0..11, a SLOT index
 5. for (i = 0; i < 12; i++) {
        GetDialogItemRect(theDial, kLoadNameFirstItem + i, &loadHouseRects[i]);
        GetDialogItemRect(theDial, kLoadIconFirstItem + i, &tempRect);
        loadHouseRects[i].top = tempRect.top;          // extend upward to cover the icon
        loadHouseRects[i].bottom++;                    // +1 px for the inverted-name overhang
    }
 6. loop ModalDialog(loadFilterUPP, &item):
      item == kOkayButton:
          thisHouseIndex += housePage                  // back to an ABSOLUTE index
          if (thisHouseIndex != wasIndex) {
              CloseHouse()
              PasStringCopy(theHousesSpecs[thisHouseIndex].name, thisHouseName)
              if (OpenHouse()) ReadHouse()
          }
          leaving = true
      item == kCancelButton:
          thisHouseIndex = wasIndex                    // restore the ABSOLUTE index
          leaving = true
      kLoadNameFirstItem <= item <= kLoadNameLastItem   (5..16)
      or kLoadIconFirstItem <= item <= kLoadIconLastItem (17..28):
          screenCount = min(housesFound - housePage, kDispFiles)
          slot = item - kLoadNameFirstItem   (resp. - kLoadIconFirstItem)
          if ((slot != thisHouseIndex) && (slot < screenCount)) {
              Inval old slot; thisHouseIndex = slot; Inval new slot
          }
          if (double-click test passes) {
              thisHouseIndex += housePage
              if (thisHouseIndex != wasIndex) {
                  MyDisableControl(theDial, kOkayButton)
                  MyDisableControl(theDial, kCancelButton)
                  CloseHouse()
                  PasStringCopy(theHousesSpecs[thisHouseIndex].name, thisHouseName)
                  if (OpenHouse()) ReadHouse()
              }
              leaving = true
          }
      item == kScrollUpItem:   PageUpHouses(theDial)
      item == kScrollDownItem: PageDownHouses(theDial)
 7. DisposeDialog(theDial); DisposeModalFilterUPP(loadFilterUPP)
```

Observations for a port:

* `thisHouseIndex` is **overloaded**: absolute index outside the dialog, slot index (0..11) inside
  it. Steps 4 and 6 convert back and forth. Getting this wrong is the easiest way to break the
  picker.
* Clicking a slot that is empty (`slot >= screenCount`) does not change the selection, but the
  double-click test still runs and can therefore open the currently selected house.
* The double-click path disables Okay and Cancel before the (potentially slow) house load, giving
  visual feedback; the Okay path does not.
* Selecting the house that is already open is a no-op (`thisHouseIndex != wasIndex` guard) — the
  house is *not* reloaded.
* The initial gray-arrow logic in step 3 uses the *absolute* `thisHouseIndex` against
  `housesFound - kDispFiles`, whereas paging uses `housePage`. With e.g. `housesFound = 20` and
  `thisHouseIndex = 15`, step 3 grays the down arrow (15 > 8) even though page 1 is the last page
  anyway — harmless. But with `housesFound = 30` and `thisHouseIndex = 20`, neither arrow is
  grayed on entry even though `housePage` becomes 12 and page-up is legal; also harmless. The
  logic is only cosmetically approximate, and `PageUpHouses`/`PageDownHouses` beep at the true
  limits.

The caller is `DoGameMenu`'s `iLoadHouse` case
(`GliderPRO/Sources/Menu.c:~331-350`), which afterwards runs `OpenCloseEditWindows()`,
`UpdateMenus(false)`, resets `incrementModeTime`, and invalidates the 166×12 splash caption rect.

### 7.10 The `DITL` 1000 grid

`DLOG` 1000: bounds `(36,16,300,446)` = **430×264**, `procID` 1 (`dBoxProc`), `visible = 0`,
`goAwayFlag = 0`, `refCon` 0, `itemsID` 1000, empty title
(`GliderPRO/Glider PRO.r:6148`).

`DITL` 1000 (`GliderPRO/Glider PRO.r:4314`), all 30 items, byte-verified by parsing the resource.
Rects are `(top, left, bottom, right)` in dialog-local coordinates; `D` marks the
disabled bit (`0x80`) in the type byte:

| # | Constant | Type | Rect | W×H | Payload |
|---|---|---|---|---|---|
| 1 | `kOkayButton` | button | (235,359,255,417) | 58×20 | `Okay` |
| 2 | `kCancelButton` | button | (235,286,255,344) | 58×20 | `Cancel` |
| 3 | `kLoadTitlePictItem` | picture `D` | (0,0,32,430) | 430×32 | `PICT` **1001** |
| 4 | — | userItem `D` | (240,8,256,24) | 16×16 | (never drawn by any code) |
| 5 | `kLoadNameFirstItem` | userItem | (81,16,93,112) | 96×12 | slot 0 name |
| 6 | — | userItem | (81,116,93,212) | 96×12 | slot 1 name |
| 7 | — | userItem | (81,216,93,312) | 96×12 | slot 2 name |
| 8 | — | userItem | (81,316,93,412) | 96×12 | slot 3 name |
| 9 | — | userItem | (142,16,154,112) | 96×12 | slot 4 name |
| 10 | — | userItem | (142,116,154,212) | 96×12 | slot 5 name |
| 11 | — | userItem | (142,216,154,312) | 96×12 | slot 6 name |
| 12 | — | userItem | (142,316,154,412) | 96×12 | slot 7 name |
| 13 | — | userItem | (203,**17**,215,113) | 96×12 | slot 8 name |
| 14 | — | userItem | (203,**117**,215,213) | 96×12 | slot 9 name |
| 15 | — | userItem | (203,**217**,215,313) | 96×12 | slot 10 name |
| 16 | `kLoadNameLastItem` | userItem | (203,**317**,215,413) | 96×12 | slot 11 name |
| 17 | `kLoadIconFirstItem` | userItem | (48,48,80,80) | 32×32 | slot 0 icon |
| 18 | — | userItem | (48,148,80,180) | 32×32 | slot 1 icon |
| 19 | — | userItem | (48,248,80,280) | 32×32 | slot 2 icon |
| 20 | — | userItem | (48,348,80,380) | 32×32 | slot 3 icon |
| 21 | — | userItem | (109,48,141,80) | 32×32 | slot 4 icon |
| 22 | — | userItem | (109,148,141,180) | 32×32 | slot 5 icon |
| 23 | — | userItem | (109,248,141,280) | 32×32 | slot 6 icon |
| 24 | — | userItem | (109,348,141,380) | 32×32 | slot 7 icon |
| 25 | — | userItem | (170,48,202,80) | 32×32 | slot 8 icon |
| 26 | — | userItem | (170,148,202,180) | 32×32 | slot 9 icon |
| 27 | — | userItem | (170,248,202,280) | 32×32 | slot 10 icon |
| 28 | `kLoadIconLastItem` | userItem | (170,348,202,380) | 32×32 | slot 11 icon |
| 29 | `kScrollUpItem` | icon | (224,183,256,215) | 32×32 | `ICON` **1050** |
| 30 | `kScrollDownItem` | icon | (224,215,256,247) | 32×32 | `ICON` **1051** |

Geometry facts:

* Name items are **96×12**, icon items **32×32**; the icon is centered 32 px above and 32 px right
  of its name's left edge.
* Row pitch **61 px** (icon tops 48 / 109 / 170; name tops 81 / 142 / 203), column pitch
  **100 px**.
* Row 3's name items are shifted **+1 px in x** relative to rows 1 and 2 (lefts 17/117/217/317
  vs 16/116/216/316) — almost certainly an authoring slip in the `DITL`, but it is in the shipped
  resource and a pixel-faithful port should keep it.
* The 12 icon slots are `userItem`s (type 0), *not* `icon` items — the code plots into them by
  hand (§7.6). Only the two scroll arrows are real `icon` items, so only they get free
  Dialog-Manager drawing.
* `loadHouseRects[i]` = the name rect with `top` taken from the icon rect and `bottom++`
  (`GliderPRO/Sources/SelectHouse.c:~398-402`), e.g. slot 0 = `(48, 16, 94, 112)` — 96×46.
* Item 3's picture ID is **1001** (`kLoadTitlePict1`), and it is disabled so it never reports
  clicks. The red-orange file box drawn by `ColorFrameWHRect(8, 39, 413, 184, …)` is the rect
  `(39, 8, 223, 421)`, which encloses all 24 slot items with a 7-px left margin.
---

## 8. Saved games

**Executive summary: the saved-game feature is completely inert in 1.0.4.** Every entry point
exists, the menu item exists and is enabled, the alerts exist — but the two functions that would
do the work are a `return false;` stub and a fully commented-out body. This section documents both
the (dead) intended design and exactly what a user actually experiences, because a Go port has to
choose.

There are **two** saved-game designs in the source, from two different eras:

| Design | Container | Struct | Status in 1.0.4 |
|---|---|---|---|
| "old" — one save slot inside the house file | `houseType.savedGame` at offset 820, `hasGame` at 860 | `gameType`, 40 bytes | writer `SaveGame()` exists but has **no live callers**; the on-disk field exists in every shipped house |
| "new" — a standalone `'gliG'` document | its own file | `game2Type` header + `savedRoom[nRooms]` | writer `SaveGame2()` body **entirely commented out**; reader `OpenSavedGame()` is `return false;` |

### 8.1 `kSavedGameVersion` and the version constants

`GliderPRO/Sources/SavedGames.c:14`:

```c
#define kSavedGameVersion   0x0200      // = 512 decimal
```

Related on-disk version constants, for contrast
(`GliderPRO/Headers/GliderDefines.h:517-518`):

| Constant | Value | Applies to |
|---|---|---|
| `kHouseVersion` | `0x0200` | `houseType.version` |
| `kNewHouseVersion` | `0x0300` | rejected as "newer than me" |
| `kSavedGameVersion` | `0x0200` | `gameType.version` / `game2Type.version` |

**Empirically, 18 of the 22 shipped houses store `savedGame.version = 0x0100`**, i.e. a *pre*-0x0200
saved game left over from an older build (see §8.7); the other four hold `0x0000`, `0x0001`, `0x00A9`
and `0xF6F6`, i.e. uninitialised bytes (§8.8 has the per-house table). `hasGame` is 0 in 20 of them,
but **`ImagineHouse PRO II` and `Titanic` ship with `hasGame == 1`**, so a port that revives the
resume-game UI will find a live, version-`0x0100` saved game in two retail houses. In 1.0.4 this is
harmless because `hasGame` is **write-only**: the only three references are the assignments at
`GliderPRO/Sources/House.c:139` and `GliderPRO/Sources/SavedGames.c:337, 341`, and nothing ever
reads it. Nothing validates `savedGame.version` either.

### 8.2 `gameType` — the in-house save slot (40 bytes)

`GliderPRO/Headers/GliderStructs.h:116-134`, packed `align=mac68k`, big-endian.
Offsets verified by compiling the declaration with `#pragma pack(2)` and `long = int32_t`:

| Off | Size | Type | Field | Set from (in `SaveGame`) |
|---|---|---|---|---|
| 0 | 2 | `short` | `version` | `kSavedGameVersion` (0x0200) |
| 2 | 2 | `short` | `wasStarsLeft` | `numStarsRemaining` |
| 4 | 4 | `long` | `timeStamp` | `GetDateTime(&stamp)` — a **raw, unmasked** Mac date |
| 8 | 4 | `Point` | `where` | `.h = theGlider.dest.left`, `.v = theGlider.dest.top` |
| 12 | 4 | `long` | `score` | `theScore` |
| 16 | 4 | `long` | `unusedLong` | `0L` |
| 20 | 4 | `long` | `unusedLong2` | `0L` |
| 24 | 2 | `short` | `energy` | `batteryTotal` |
| 26 | 2 | `short` | `bands` | `bandsTotal` |
| 28 | 2 | `short` | `roomNumber` | `thisRoomNumber` |
| 30 | 2 | `short` | `gliderState` | `theGlider.mode` |
| 32 | 2 | `short` | `numGliders` | `mortals` |
| 34 | 2 | `short` | `foil` | `foilTotal` |
| 36 | 2 | `short` | `unusedShort` | `0` |
| 38 | 1 | `Boolean` | `facing` | `theGlider.facing` |
| 39 | 1 | `Boolean` | `showFoil` | `showFoil` |
| | **40** | | | |

Note `Point` on the Mac is `{short v; short h;}` — **v first**. So bytes 8–9 are `where.v` and
10–11 are `where.h`, even though the source assigns `.h` first.

`SaveGame(Boolean doSave)` (`GliderPRO/Sources/SavedGames.c:303-351`):

```
 1. if (twoPlayerGame) return;                          // two-player games are never saved
 2. wasState = HGetState((Handle)thisHouse); HLock(...); thisHousePtr = *thisHouse;
 3. if (doSave) {
        thisHousePtr->savedGame.version      = kSavedGameVersion;   // 0x0200
        thisHousePtr->savedGame.wasStarsLeft = numStarsRemaining;
        GetDateTime(&stamp);
        thisHousePtr->savedGame.timeStamp    = (long)stamp;
        thisHousePtr->savedGame.where.h      = theGlider.dest.left;
        thisHousePtr->savedGame.where.v      = theGlider.dest.top;
        thisHousePtr->savedGame.score        = theScore;
        thisHousePtr->savedGame.unusedLong   = 0L;
        thisHousePtr->savedGame.unusedLong2  = 0L;
        thisHousePtr->savedGame.energy       = batteryTotal;
        thisHousePtr->savedGame.bands        = bandsTotal;
        thisHousePtr->savedGame.roomNumber   = thisRoomNumber;
        thisHousePtr->savedGame.gliderState  = theGlider.mode;
        thisHousePtr->savedGame.numGliders   = mortals;
        thisHousePtr->savedGame.foil         = foilTotal;
        thisHousePtr->savedGame.unusedShort  = 0;
        thisHousePtr->savedGame.facing       = theGlider.facing;
        thisHousePtr->savedGame.showFoil     = showFoil;
        thisHousePtr->hasGame                = true;
    } else {
        thisHousePtr->hasGame = false;
    }
 4. HSetState((Handle)thisHouse, wasState);
 5. if (doSave) { if (!WriteHouse(theMode == kEditMode)) YellowAlert(kYellowFailedWrite /*12*/, 0); }
```

Critically, this design **does not save room state** — no `visited` flags, no object states. Only
the glider's position, inventory and score. It writes the whole house file back to disk to persist
40 bytes.

**Live callers of `SaveGame`: none.** The only reference is commented out at
`GliderPRO/Sources/Menu.c:458` (inside the House ▸ New House… case:
`// SaveGame(false);` — the intent was to clear any stale save when a new house is created).
`hasGame` is set to `false` in exactly one live place, `InitializeEmptyHouse`
(`GliderPRO/Sources/House.c:139`), and never to `true`.

The source comment above the function is candid
(`GliderPRO/Sources/SavedGames.c:300-301`):

```
// This is probably about 3 days away from becoming the "old" function
// for saving games.
```

### 8.3 `savedRoom` — per-room state (292 bytes)

`GliderPRO/Headers/GliderStructs.h:136-142`:

| Off | Size | Type | Field |
|---|---|---|---|
| 0 | 2 | `short` | `unusedShort` |
| 2 | 1 | `Byte` | `unusedByte` |
| 3 | 1 | `Boolean` | `visited` |
| 4 | 288 | `objectType objects[24]` | the room's 24 object records, 12 bytes each |
| | **292** | | |

`objectType` (`GliderPRO/Headers/GliderStructs.h:90-105`) is `short what` plus a 10-byte union =
**12 bytes**; `kMaxRoomObs` = 24 (`GliderPRO/Headers/GliderDefines.h:250`).

Note `savedRoom` is exactly the same size as `scoresType` (292) by coincidence.

### 8.4 `game2Type` — the standalone `'gliG'` document header (110 bytes, **not** 114)

`GliderPRO/Headers/GliderStructs.h:144-164`. The source's own running comment tally claims
`// total = 114`; that is **wrong**. Verified by compilation with `#pragma pack(2)`:

| Off | Size | Type | Field | Notes |
|---|---|---|---|---|
| 0 | 70 | `FSSpec` | `house` | `short vRefNum` + `long parID` + `Str63 name` |
| 70 | 2 | `short` | `version` | `kSavedGameVersion` 0x0200 |
| 72 | 2 | `short` | `wasStarsLeft` | |
| 74 | 4 | `long` | `timeStamp` | copied from `thisHousePtr->timeStamp` (the *masked* house stamp) |
| 78 | 4 | `Point` | `where` | v then h |
| 82 | 4 | `long` | `score` | |
| 86 | 4 | `long` | `unusedLong` | |
| 90 | 4 | `long` | `unusedLong2` | |
| 94 | 2 | `short` | `energy` | |
| 96 | 2 | `short` | `bands` | |
| 98 | 2 | `short` | `roomNumber` | |
| 100 | 2 | `short` | `gliderState` | |
| 102 | 2 | `short` | `numGliders` | |
| 104 | 2 | `short` | `foil` | |
| 106 | 2 | `short` | `nRooms` | |
| 108 | 1 | `Boolean` | `facing` | |
| 109 | 1 | `Boolean` | `showFoil` | |
| **110** | — | `savedRoom savedData[]` | flexible array | |

So a `.gliG` file would be exactly **`110 + 292 × nRooms`** bytes — which is what
`byteCount = sizeof(game2Type) + sizeof(savedRoom) * numRooms`
(`GliderPRO/Sources/SavedGames.c:56`) computes, because `sizeof` of a struct ending in a
zero-length array is the header size. For Slumberland (383 rooms) that would be
`110 + 292 × 383 = 111,946` bytes.

Note the discrepancy: `game2Type` has **no `unusedShort`** field where `gameType` has one at
offset 36, so the two are *not* prefix-compatible past `foil`.

### 8.5 The intended (dead) write path: `SaveGame2`

`GliderPRO/Sources/SavedGames.c:30-148`. Body is commented out `/* … */` from line 33 to line 147
with the note `// Add NavServices later.` at line 31. Reconstructed:

```
 1. FlushEvents(everyEvent, 0)
 2. HLock(thisHouse); numRooms = thisHousePtr->nRooms; HSetState(...)
 3. byteCount = sizeof(game2Type) + sizeof(savedRoom) * numRooms      // 110 + 292*nRooms
    savedGame = (gamePtr)NewPtr(byteCount)
    if (savedGame == nil) { YellowAlert(kYellowFailedSaveGame /*18*/, MemError()); return; }
 4. GetFirstWordOfString(thisHouseName, gameNameStr)     // "Slumberland Deluxe" -> "Slumberland"
    if (gameNameStr[0] > 23) gameNameStr[0] = 23         // truncate to 23 chars
    PasStringConcat(gameNameStr, "\p Game")              // -> "Slumberland Game", <= 28 chars
 5. StandardPutFile("\pSave Game As:", gameNameStr, &theReply)
    if (!theReply.sfGood) return;
 6. if (theReply.sfReplacing) {
        FSMakeFSSpec(...); CheckFileError(theErr, "\pSaved Game");
        FSpDelete(&tempSpec); CheckFileError(theErr, "\pSaved Game");
    }
 7. HLock(thisHouse); thisHousePtr = *thisHouse
    savedGame->house        = theHousesSpecs[thisHouseIndex]
    savedGame->version      = kSavedGameVersion
    savedGame->wasStarsLeft = numStarsRemaining
    savedGame->timeStamp    = thisHousePtr->timeStamp     // the HOUSE stamp, not "now"
    savedGame->where.h      = theGlider.dest.left
    savedGame->where.v      = theGlider.dest.top
    savedGame->score        = theScore
    savedGame->unusedLong   = 0L;   savedGame->unusedLong2 = 0L
    savedGame->energy       = batteryTotal
    savedGame->bands        = bandsTotal
    savedGame->roomNumber   = thisRoomNumber
    savedGame->gliderState  = theGlider.mode
    savedGame->numGliders   = mortals
    savedGame->foil         = foilTotal
    savedGame->nRooms       = numRooms
    savedGame->facing       = theGlider.facing
    savedGame->showFoil     = showFoil
 8. for (r = 0; r < numRooms; r++) {
        destRoom = &savedGame->savedData[r];  srcRoom = &thisHousePtr->rooms[r];
        destRoom->unusedShort = 0; destRoom->unusedByte = 0;
        destRoom->visited = srcRoom->visited;
        for (i = 0; i < kMaxRoomObs /*24*/; i++) destRoom->objects[i] = srcRoom->objects[i];
    }
    HSetState(...)
 9. FSpCreate(&theReply.sfFile, 'ozm5', 'gliG', theReply.sfScript)
10. FSpOpenDF(&theReply.sfFile, fsCurPerm, &gameRefNum)
11. SetFPos(gameRefNum, fsFromStart, 0L)
12. FSWrite(gameRefNum, &byteCount, (Ptr)savedGame)
13. SetEOF(gameRefNum, byteCount)
14. FSClose(gameRefNum)
15. DisposePtr((Ptr)savedGame)
```

Every File Manager step is wrapped in `CheckFileError(theErr, "\pSaved Game")` (§11.2), so the
user-visible file name in error alerts would literally be `Saved Game`, not the chosen name.

**`SaveGame2()` has two live call sites**, both in `GliderPRO/Sources/Input.c` — so the user *can*
reach it, and it does nothing at all:

| Call site | Trigger |
|---|---|
| `GliderPRO/Sources/Input.c:62` | ⌘Q during play: `playing = false; paused = false; if ((!twoPlayerGame) && (!demoGoing)) { if (QuerySaveGame()) SaveGame2(); }` |
| `GliderPRO/Sources/Input.c:68` | ⌘S during play: `RefreshScoreboard(kSavingTitleMode); SaveGame2(); HideCursor(); CopyRectWorkToMain(&workSrcRect); RefreshScoreboard(kNormalTitleMode);` |

So pressing ⌘S in game flips the scoreboard to `kSavingTitleMode` (2), does nothing, and flips it
back — a visible "Saving…" flicker with no file written.

### 8.6 The intended (dead) read path: `OpenSavedGame`

`GliderPRO/Sources/SavedGames.c:167-295`. Line 169 is literally:

```c
Boolean OpenSavedGame (void)
{
return false;       // TEMP fix this iwth NavServices
/*
   ... 125 lines of real implementation ...
*/
}
```

The commented-out implementation, with its four validation gates — these matter because they
define the compatibility contract a port should honour if it revives the format:

```
 1. theList[0] = 'gliG'
    StandardGetFile(nil, 1, theList, &theReply)
    if (!theReply.sfGood) return false
 2. FSpOpenDF(&theReply.sfFile, fsCurPerm, &gameRefNum)      + CheckFileError("\pSaved Game")
 3. GetEOF(gameRefNum, &byteCount)                            + CheckFileError
 4. savedGame = (gamePtr)NewPtr(byteCount)
    if (nil) { YellowAlert(kYellowFailedSaveGame /*18*/, MemError()); FSClose; return false; }
 5. SetFPos(gameRefNum, fsFromStart, 0L)                      + CheckFileError
 6. FSRead(gameRefNum, &byteCount, savedGame)                 + CheckFileError
 7. HLock(thisHouse); thisHousePtr = *thisHouse
 8. GATE 1: if (!EqualString(savedGame->house.name, thisHouseName, true, true))
                SavedGameMismatchError(savedGame->house.name);      // ALRT 1044
                -> return false
            // note: CASE-SENSITIVE, diacritic-sensitive; compares only the NAME,
            //       not vRefNum/parID, even though the whole FSSpec was stored
 9. GATE 2: else if (thisHousePtr->timeStamp != savedGame->timeStamp)
                YellowAlert(kYellowSavedTimeWrong /*19*/, 0) -> return false
            // the house has been re-saved since; its rooms may have moved
10. GATE 3: else if (savedGame->version != kSavedGameVersion /*0x0200*/)
                YellowAlert(kYellowSavedVersWrong /*20*/, kSavedGameVersion) -> return false
11. GATE 4: else if (savedGame->nRooms != thisHousePtr->nRooms)
                YellowAlert(kYellowSavedRoomsWrong /*21*/,
                            savedGame->nRooms - thisHousePtr->nRooms) -> return false
12. else {
        smallGame.wasStarsLeft = savedGame->wasStarsLeft
        smallGame.where.h      = savedGame->where.h
        smallGame.where.v      = savedGame->where.v
        smallGame.score        = savedGame->score
        smallGame.unusedLong   = savedGame->unusedLong
        smallGame.unusedLong2  = savedGame->unusedLong2
        smallGame.energy       = savedGame->energy
        smallGame.bands        = savedGame->bands
        smallGame.roomNumber   = savedGame->roomNumber
        smallGame.gliderState  = savedGame->gliderState
        smallGame.numGliders   = savedGame->numGliders
        smallGame.foil         = savedGame->foil
        smallGame.unusedShort  = 0
        smallGame.facing       = savedGame->facing
        smallGame.showFoil     = savedGame->showFoil
        // NOTE: smallGame.version and smallGame.timeStamp are NEVER assigned
        for (r = 0; r < savedGame->nRooms; r++) {
            srcRoom  = &savedGame->savedData[r]
            destRoom = &thisHousePtr->rooms[r]
            destRoom->visited = srcRoom->visited
            for (i = 0; i < kMaxRoomObs; i++) destRoom->objects[i] = srcRoom->objects[i]
        }
    }
13. HSetState(...); DisposePtr((Ptr)savedGame); FSClose; return true
```

Note step 12 mutates the **in-memory house** (room `visited` flags and all object records) without
setting `fileDirty`, so the modified house is not written back — the save is applied to the live
session only.

**`smallGame` is the resume hand-off buffer** (`gameType smallGame;`,
`GliderPRO/Sources/SavedGames.c:20`). Its only writers are inside the commented-out block above
(`GliderPRO/Sources/SavedGames.c:261-275`); its readers are all live:

| Reader | Line | Uses |
|---|---|---|
| `WhereDoesGliderBegin` | `GliderPRO/Sources/House.c:233` | `initialPt = smallGame.where` when `mode == kResumeGameMode` |
| `InitGlider` | `GliderPRO/Sources/Play.c:312, 318-325` | `numStarsRemaining`, `theScore`, `mortals`, `batteryTotal`, `bandsTotal`, `foilTotal`, `thisGlider->mode`, `thisGlider->facing`, `showFoil` |
| `SetHouseToSavedRoom` | `GliderPRO/Sources/Play.c:382` | `ForceThisRoom(smallGame.roomNumber)` |

Since nothing writes it, `smallGame` is all zeroes for the life of the process. It is a BSS
global, so a hypothetical resume would start the glider at (0,0) in room 0 with 0 lives.

### 8.7 What the user actually sees

| Surface | Where | Actual behaviour in 1.0.4 |
|---|---|---|
| Game ▸ Open Saved Game… (⌘O), `MENU` 129 item 3 | `GliderPRO/Sources/Menu.c:317-324` | `resumedSavedGame = true; HeyYourPissingAHighScore(); if (OpenSavedGame()) {...}` — the alert fires, then `OpenSavedGame` returns false, so nothing happens. **`resumedSavedGame` stays `true`, which permanently disables high scores for the rest of the session** (§9.5). |
| `ALRT` 1046 "you can't get a high score on a resumed game" | `HeyYourPissingAHighScore`, `GliderPRO/Sources/Menu.c:779-786` | shown *before* the (failing) file open, so the user gets a warning about a feature that then does not run |
| ⌘S during play | `GliderPRO/Sources/Input.c:65-72` | scoreboard flashes `kSavingTitleMode`, no file written |
| ⌘Q during play | `GliderPRO/Sources/Input.c:55-64` | `ALRT` 1041 "Save game?" → Yes calls the no-op `SaveGame2()` |
| `ALRT` 1041 (`QuerySaveGame`) | `GliderPRO/Sources/Input.c:383-393` | `InitCursor(); FlushEvents(everyEvent, 0); hitWhat = Alert(1041, nil); return (hitWhat == kYesSaveGameButton /*1*/);` — bounds `(40,40,116,302)` = 262×76, `stages` **0xCCCC** (all four stages identical: no sound, default item 2) |
| `ALRT` 1044 (`SavedGameMismatchError`) | `GliderPRO/Sources/SavedGames.c:152-164` | unreachable: only called from inside the commented-out `OpenSavedGame` |
| `DLOG` 1025 "Resume Game" + `QueryResumeGame()` | `GliderPRO/Sources/Menu.c:710-759` | fully implemented, **never called** (§3.9) — the older design that would have offered the in-house `savedGame` slot |

`HeyYourPissingAHighScore` (`GliderPRO/Sources/Menu.c:779-786`):

```c
#define kNoHighScoreAlert   1046
short whoCares;
whoCares = Alert(kNoHighScoreAlert, nil);
```

`ALRT` 1046: bounds `(40,40,132,300)` = 260×92, `stages` `0x4444`.

`resumedSavedGame` is set `true` only here and `false` on New Game / Two Player Game
(`GliderPRO/Sources/Menu.c:307, 313`), and is read by `TestHighScore`
(`GliderPRO/Sources/HighScores.c:374-378`), which returns `false` immediately when it is set. So
one click on the dead Open Saved Game… item suppresses high scores until the next New Game.

### 8.8 Empirical verification against the shipped houses

`GliderPRO/Houses/` contains **22** `.binhex` houses, not two. All 22 were decoded with a BinHex 4.0
decoder and the data forks parsed directly; the results below cover the whole set.

Every one has `type = 'gliH'`, `creator = 'ozm5'`; the Finder flags word is `0x0500`, `0x0100` or
`0x0504` depending on the file. Two examples:

```
Empty House.binhex  -> name='Empty House'  type=b'gliH' creator=b'ozm5'
                       dlen=13046    rlen=2670
Slumberland.binhex  -> name='Slumberland'  type=b'gliH' creator=b'ozm5'
                       dlen=134150   rlen=1041112
```

`houseType` header offsets (verified by compiling the declaration with `#pragma pack(2)`,
`long = int32_t`):

| Off | Size | Field |
|---|---|---|
| 0 | 2 | `version` |
| 2 | 2 | `unusedShort` |
| 4 | 4 | `timeStamp` |
| 8 | 4 | `flags` |
| 12 | 4 | `initial` (`Point`, v then h) |
| 16 | 256 | `banner` (`Str255`) |
| 272 | 256 | `trailer` (`Str255`) |
| 528 | 292 | `highScores` (`scoresType`) |
| **820** | **40** | **`savedGame` (`gameType`)** |
| **860** | **1** | **`hasGame`** |
| 861 | 1 | `unusedBoolean` |
| 862 | 2 | `firstRoom` |
| 864 | 2 | `nRooms` |
| **866** | — | `rooms[]` (`roomType`, 348 bytes each) |

Observed values:

```
=== Slumberland   len 134150   version=0x0200  nRooms=383  firstRoom=126
    866 + 348*383 = 134150   match=True
    timeStamp=743682377  flags=0x00000000  initial=(v7,h361)
    hasGame=0  unusedBoolean=0
    savedGame: version=0x0100  wasStarsLeft=5  timeStamp=2875627985 (1995-02-14T17:33:05)
               where=(v59,h236)  score=28200  unusedLong=0  unusedLong2=0
               energy=46  bands=6  roomNumber=40  gliderState=0  numGliders=4  foil=3
               unusedShort=0  facing=1  showFoil=1

=== Empty House   len 13046    version=0x0200  nRooms=35   firstRoom=0
    866 + 348*35 = 13046     match=True
    timeStamp=741421292  flags=0x00000000  initial=(v64,h83)
    hasGame=0  unusedBoolean=0
    savedGame: version=0x0100  wasStarsLeft=0  timeStamp=2858260248 (1994-07-28T17:10:48)
               where=(v14,h85)  score=0  ...  numGliders=2  facing=1  showFoil=0
```

All 22 houses, re-parsed with the offsets above (`ts` = `houseType.timeStamp`, `bit0` = the lock bit,
`flags` = `houseType.flags`, `sgVer` = `savedGame.version`):

| House | dlen | rlen | nRooms | firstRoom | `flags` | `hasGame` | `ts` | lock bit | `sgVer` |
|---|---|---|---|---|---|---|---|---|---|
| Art Museum | 38798 | 7476159 | 109 | 91 | **6** | 0 | 746622005 | 1 | `0xF6F6` |
| CD Demo House | 72554 | 1612342 | 206 | 70 | **2** | 0 | 742294873 | 1 | `0x0100` |
| California or Bust! | 6434 | 259542 | 16 | 14 | **2** | 0 | 745882296 | 0 | `0x0100` |
| Castle o' the Air | 30446 | 306364 | 85 | 33 | 0 | 0 | 741536868 | 0 | `0x0100` |
| Davis Station | 23486 | 1871320 | 65 | 4 | **2** | 0 | 743681789 | 1 | `0x0000` |
| Demo House | 16526 | 491757 | 45 | 0 | 0 | 0 | 743682113 | 1 | `0x0100` |
| Empty House | 13046 | 2670 | 35 | 0 | 0 | 0 | 741421292 | 0 | `0x0100` |
| Fun House | 15830 | 662443 | 43 | 29 | 0 | 0 | 742681616 | 0 | `0x0000` |
| Grand Prix | 61766 | 1347764 | 175 | 127 | 0 | 0 | 741444157 | 1 | `0x00A9` |
| ImagineHouse PRO II | 97958 | 677770 | 279 | 1 | 0 | **1** | 740149565 | 1 | `0x0100` |
| In The Mirror | 34622 | 151870 | 97 | 6 | 0 | 0 | 741512193 | 1 | `0x0100` |
| Land of Illusion | 106310 | 401793 | 303 | 43 | **2** | 0 | 745268052 | 0 | `0x0100` |
| Leviathan | 165122 | 1901282 | 472 | 39 | 0 | 0 | 741514803 | 1 | `0x0100` |
| Metropolis | 45062 | 946907 | 127 | 8 | 0 | 0 | 741642315 | 1 | `0x0100` |
| Nemo's Market | 44018 | 590602 | 124 | 0 | **2** | 0 | 741121233 | 1 | `0x0100` |
| Rainbow's End | 78470 | 316065 | 223 | 30 | **2** | 0 | 741121489 | 1 | `0x0100` |
| Sampler | **1564** | 286 | 2 | 1 | 0 | 0 | 893435880 | 0 | `0x0001` |
| Slumberland | 134150 | 1041112 | 383 | 126 | 0 | 0 | 743682377 | 1 | `0x0100` |
| SpacePods | 140762 | 636844 | 402 | 259 | **2** | 0 | 741122073 | 1 | `0x0100` |
| Teddy World | 185654 | 3107348 | 531 | 0 | 0 | 0 | 744401697 | 1 | `0x0100` |
| The Asylum Pro | 49586 | 494107 | 140 | 20 | 0 | 0 | 738016929 | 1 | `0x0100` |
| Titanic | 73250 | 845727 | 208 | 92 | 0 | **1** | 755123966 | 0 | `0x0100` |

Conclusions:

* `houseType.version` is `0x0200` in **all 22** files — no house ships at `kNewHouseVersion`
  (`0x0300`).
* `dlen == 866 + 348 × nRooms` **exactly** for 21 of the 22 files. The exception is **`Sampler`**:
  `nRooms = 2` gives 1562, but the data fork is **1564** bytes — two bytes of trailing slack. Any
  port that uses this identity as a validity check (see P5) will reject `Sampler`; the check must be
  `dlen >= 866 + 348 × nRooms` (or the room count must be derived from the length, not the reverse).
* `hasGame` is 1 in **two** houses — **`ImagineHouse PRO II`** and **`Titanic`** — so the
  "no shipped house has a saved game" assumption is false. The other 20 have `hasGame == 0` and
  therefore carry stale `savedGame` bytes that are dead data.
* `savedGame.version` is `0x0100` in 18 files, but `0x0000` (Davis Station, Fun House), `0x0001`
  (Sampler), `0x00A9` (Grand Prix) and `0xF6F6` (Art Museum) elsewhere — i.e. in the `hasGame == 0`
  houses the whole `savedGame` block is uninitialised garbage, not a consistent old record. Since
  `hasGame` gates every read, this does not matter at runtime, but a port must not validate
  `savedGame.version` unconditionally.
* `houseType.flags` is **not** always 0: 14 files have 0, seven have `0x00000002`
  (`phoneBitSet`), and `Art Museum` has `0x00000006` (`phoneBitSet` **and** bit 2 set, which per
  `GliderPRO/Sources/HouseIO.c:418` means `bannerStarCountOn == false`). Art Museum is therefore the
  one shipped house that suppresses the banner star count. No shipped house sets bit 0
  (`wardBitSet`).
* The lock bit (`timeStamp & 1`) is set in **15** of the 22 houses, i.e. most retail houses ship
  **locked** (uneditable). The **7** that ship unlocked are: California or Bust!, Castle o' the Air,
  Empty House, Fun House, Land of Illusion, Sampler, Titanic.
* The `savedGame.timeStamp` values in the `0x0100` files decode as real dates in the Mac epoch
  (seconds since 1904-01-01), which is how you can tell they were written by `SaveGame()` (raw
  `GetDateTime`) and not by `SaveGame2()` (which would have copied the *masked* house stamp).

### 8.9 `houseType.timeStamp` — a date **and** the house lock bit

This matters to the saved-game gate 2 above and to the entire "unlock the editor" UI, so it is
documented here.

`WriteHouse` (`GliderPRO/Sources/HouseIO.c:477-487`):

```c
GetDateTime(&timeStamp);
timeStamp &= 0x7FFFFFFF;                    // clear the high bit
if (changeLockStateOfHouse)
    houseUnlocked = !saveHouseLocked;
if (houseUnlocked)  timeStamp &= 0x7FFFFFFE;   // bit 0 CLEAR  = unlocked
else                timeStamp |= 0x00000001;   // bit 0 SET    = locked
(*thisHouse)->timeStamp = (long)timeStamp;
(*thisHouse)->version   = wasHouseVersion;
```

`ReadHouse` (`GliderPRO/Sources/HouseIO.c:402`):

```c
houseUnlocked = (((*thisHouse)->timeStamp & 0x00000001) == 0);
```

So `houseType.timeStamp` is a Mac date with **bit 31 forcibly cleared** and **bit 0 repurposed as
the lock flag** (0 = unlocked/editable, 1 = locked). The date is therefore quantised to 2 seconds
and shifted by 2³¹ seconds whenever the real date has bit 31 set — which it always does for any
date after 1972-01-19.

Verified by re-parsing all 22 shipped houses (§8.8 has the full table; two worked examples here):

| House | stored `timeStamp` | bit 0 | state | `+ 0x80000000` | real date |
|---|---|---|---|---|---|
| Slumberland | 743682377 | 1 | **locked** | 2891166025 | 1995-08-13T13:40:25 |
| Empty House | 741421292 | 0 | **unlocked** | 2888904940 | 1995-07-18T09:35:40 |

15 of the 22 ship locked, 7 unlocked.

That is exactly right: `Slumberland` is the commercial house you may not edit; `Empty House` is
the template that says `This is an empty house.  You can fly through it but there are no
obstacles or prizes.  Make a copy of this house and feel free to fill it up.`
(observed at `houseType` offset 16, length 141, `\r`-separated).

The Option-click-Brains unlock (§6.7) sets `changeLockStateOfHouse = true; saveHouseLocked =
false;`, which makes the *next* `WriteHouse` clear bit 0 permanently.

Consequence for gate 2 of `OpenSavedGame`: because `WriteHouse` rewrites `timeStamp` on **every**
save where `fileDirty` is set, any edit to the house invalidates every existing `.gliG` file for
it — including a save triggered only by the high-score table changing.

### 8.10 What a Go port should do

Three defensible choices, in increasing fidelity cost:

1. **Reproduce 1.0.4 exactly**: Open Saved Game… is enabled, shows `ALRT` 1046, then silently
   does nothing; ⌘S flickers the scoreboard; ⌘Q offers to save and does not. This is bug-for-bug
   faithful and will confuse every player.
2. **Revive the `'gliG'` design** (`game2Type` 110-byte header + `savedRoom[nRooms]`), implementing
   `SaveGame2`/`OpenSavedGame` as written, keeping all four validation gates. This is what the
   author intended and is forward-compatible with the struct layout above.
3. **Revive the in-house slot** (`gameType` at offset 820 + `hasGame` at 860) and wire up the
   already-written `QueryResumeGame()` / `DLOG` 1025 UI. This needs no new file format and the
   dialog already exists, but it cannot restore room/object state.

If option 2 or 3 is taken, the `where` field is a `Point` in **v,h** order and every multi-byte
field is **big-endian**; `Boolean` is one byte where any non-zero means true (`GetControlValue`
returns 1, but `facing` is `theGlider.facing` which is a plain 0/1).
---

## 9. High scores

`GliderPRO/Sources/HighScores.c`, 856 lines. High scores live **inside the house file**, one table
of 10 per house, plus a 31-character "banner" message that the #1 player gets to write.

### 9.1 `scoresType` — 292 bytes, stored at `houseType` + 528

`GliderPRO/Headers/GliderStructs.h:107-114`, packed `align=mac68k`, big-endian. Offsets verified by
compiling with `#pragma pack(2)`, `long = int32_t`:

| Off | Size | Type | Field | Meaning |
|---|---|---|---|---|
| 0 | 32 | `Str31` | `banner` | the #1 player's message, 1 length byte + 31 chars |
| 32 | 160 | `Str15 names[10]` | | 10 × 16 bytes (1 length byte + 15 chars) |
| 192 | 40 | `long scores[10]` | | points |
| 232 | 40 | `unsigned long timeStamps[10]` | | raw `GetDateTime` (Mac epoch, seconds since 1904-01-01) |
| 272 | 20 | `short levels[10]` | | rooms visited |
| | **292** | | | |

Constants: `kMaxScores 10` (`GliderPRO/Headers/GliderDefines.h:249`).

`Str15` is `unsigned char[16]` and `Str31` is `unsigned char[32]`, so the maximum name is 15
characters and the maximum banner 31 characters. `TestHighScore` enforces this with
`PasStringCopyNum(tempStr, highName, 15)` (`GliderPRO/Sources/HighScores.c:526`) and
`PasStringCopyNum(tempStr, highBanner, 31)` (`GliderPRO/Sources/HighScores.c:631`).

Note `timeStamps` is `unsigned long` while `houseType.timeStamp` is signed `long` — the high-score
timestamps are **not** masked and carry no lock bit.

Observed in `Slumberland` (data fork offset 528):

```
banner = 'Your Message Here'
 1 'Your Name'        score=10800  levels=25  timeStamp=0
 2 'Your Name'        score= 7800  levels=16  timeStamp=0
 3 'Ozma'             score= 4900  levels=16  timeStamp=2888914094  (1995-07-18T12:08:14)
 4 '--------------'   score=    0  levels= 0  timeStamp=0
 5 '--------------'   score=    0  levels= 0  timeStamp=0
 6 '--------------'   score=    0  levels= 0  timeStamp=0
 7 '--------------'   score=    0  levels= 0  timeStamp=0
 8 '--------------'   score=    0  levels= 0  timeStamp=0
 9 '--------------'   score=    0  levels= 0  timeStamp=0
10 '--------------'   score=    0  levels= 0  timeStamp=0
```

Observed in `Empty House`: `banner = 'Empty House'`, all 10 entries `'--------------'`, score 0,
level 0, timeStamp 0. That is exactly the output of `ZeroHighScores()`, whose placeholder is the
literal 14-hyphen string `"\p--------------"` (`GliderPRO/Sources/HighScores.c:336`) and whose
banner default is `thisHouseName` (`GliderPRO/Sources/HighScores.c:333`).

### 9.2 Session globals

| Global | Type | Declared | Init | Notes |
|---|---|---|---|---|
| `highName` | `Str15` | `GliderPRO/Sources/HighScores.c:46` | from prefs `wasHighName`, else `"\pYour Name"` | pre-fills the name dialog |
| `highBanner` | `Str31` | `GliderPRO/Sources/HighScores.c:45` | from prefs `wasHighBanner`, else `"\pYour Message Here"` | pre-fills the banner dialog |
| `lastHighScore` | `short` | `GliderPRO/Sources/HighScores.c:47` | `-1` at `GliderPRO/Sources/InterfaceInit.c:161` | index of *this* session's new entry; drawn in white instead of cyan/yellow |
| `keyStroke` | `Boolean` | `GliderPRO/Sources/HighScores.c:48` | (uninitialized) | one-shot flag used to refresh the character counter |
| `resumedSavedGame` | `Boolean` | `GliderPRO/Sources/Menu.c:33` | — | when true, `TestHighScore` returns false immediately |

`highName`/`highBanner` round-trip through the preferences file
(`GliderPRO/Sources/Main.c:67-68` on read, `:223-224` on write; defaults at `:133-134`), so the
player's name persists across launches and across houses.

### 9.3 `TestHighScore` — the scoring gate

`GliderPRO/Sources/HighScores.c:374-426`. Called from two places, both in
`GliderPRO/Sources/GameOver.c`:

| Call site | Context |
|---|---|
| `GliderPRO/Sources/GameOver.c:67` | `if (!TestHighScore())` — normal game over |
| `GliderPRO/Sources/GameOver.c:503` | `TestHighScore();` — the "you won" path |

```
 1. if (resumedSavedGame) return false;              // hard veto, see §8.7
 2. HLock(thisHouse); thisHousePtr = *thisHouse
 3. lastHighScore = -1; placing = -1
 4. for (i = 0; i < kMaxScores /*10*/; i++)
        if (theScore > thisHousePtr->highScores.scores[i]) { placing = i; lastHighScore = i; break; }
    // strictly greater; ties do NOT displace. Since ZeroHighScores writes 0,
    // ANY score >= 1 makes the list of an unplayed house.
 5. if (placing != -1) {
        FlushEvents(everyEvent, 0)
        GetHighScoreName(placing + 1)                                     // DLOG 1020
        PasStringCopy(highName, thisHousePtr->highScores.names[9])
        if (placing == 0) {
            GetHighScoreBanner()                                          // DLOG 1021
            PasStringCopy(highBanner, thisHousePtr->highScores.banner)
        }
        thisHousePtr->highScores.scores[9]     = theScore
        GetDateTime(&thisHousePtr->highScores.timeStamps[9])
        thisHousePtr->highScores.levels[9]     = CountRoomsVisited()
        SortHighScores()
        gameDirty = true
    }
 6. HSetState(...)
 7. if (placing != -1) { DoHighScores(); return true; } else return false;
```

The trick at step 5 is worth spelling out for a porter: the new entry is **always written into slot
9** (the last slot), clobbering whatever was in tenth place, and then `SortHighScores()` bubbles it
to its correct position. `placing` is used only to decide *whether* the player qualifies and what
number to print in the dialog ("#^1"). This is why `placing == 0` (a new #1) triggers the banner
dialog even though the record has not been sorted yet.

`gameDirty = true` (`GliderPRO/Sources/HighScores.c:414`) is what eventually persists the table —
see §9.6.

### 9.4 `SortHighScores` — selection sort, destructive in place

`GliderPRO/Sources/HighScores.c:281-318`:

```
 1. HLock(thisHouse); thisHousePtr = *thisHouse
 2. for (h = 0; h < 10; h++) {
        greatest = -1L; which = -1
        for (i = 0; i < 10; i++)
            if (thisHousePtr->highScores.scores[i] > greatest) { greatest = scores[i]; which = i; }
        if (which != -1) {
            PasStringCopy(names[which], tempScores.names[h])
            tempScores.scores[h]     = scores[which]
            tempScores.timeStamps[h] = timeStamps[which]
            tempScores.levels[h]     = levels[which]
            thisHousePtr->highScores.scores[which] = -1L      // mark consumed
        }
    }
 3. PasStringCopy(thisHousePtr->highScores.banner, tempScores.banner)   // banner preserved
 4. thisHousePtr->highScores = tempScores                               // 292-byte struct copy
 5. HSetState(...)
```

Ten passes over ten entries, O(n²) = 100 comparisons. The sentinel is `-1L`: consumed entries are
set to `-1` and the search predicate is `> greatest` with `greatest` starting at `-1`, so a
consumed entry can never be picked again. Because all real scores are `>= 0`, every slot is
consumed exactly once and `which` is never `-1` in practice — `tempScores` is fully written before
being copied back. **`tempScores` is a stack local and is never initialized**, so if that
invariant were ever broken (a negative score in the table) the uninitialized bytes would be copied
into the house.

Sort is **stable-ish but not stable**: equal scores are picked in ascending index order because the
predicate is strict `>`, so the first occurrence wins. Correct behaviour for a port: iterate
forward, take the first strictly-greatest.

### 9.5 `ZeroHighScores` / `ZeroAllButHighestScore`

| Function | Lines | Range cleared | Also |
|---|---|---|---|
| `ZeroHighScores` | `GliderPRO/Sources/HighScores.c:323-343` | `i = 0..9` | sets `banner = thisHouseName` |
| `ZeroAllButHighestScore` | `GliderPRO/Sources/HighScores.c:348-367` | `i = 1..9` | leaves `banner` and slot 0 alone |

Cleared values per slot: `names[i] = "\p--------------"` (14 hyphens), `scores[i] = 0L`,
`timeStamps[i] = 0L`, `levels[i] = 0`.

Callers:

| Caller | Line | Trigger |
|---|---|---|
| `InitializeEmptyHouse` | `GliderPRO/Sources/House.c:133` | `ZeroHighScores()` when a brand-new house is created |
| `HowToZeroScores` | `GliderPRO/Sources/HouseInfo.c:329` | `ALRT` 1032 button 2 "Clear All" |
| `HowToZeroScores` | `GliderPRO/Sources/HouseInfo.c:335` | `ALRT` 1032 button 3 "All But #1" |

`HowToZeroScores` (`GliderPRO/Sources/HouseInfo.c:319-340`), inside `#ifndef COMPILEDEMO`:

```c
//	CenterAlert(kZeroScoresAlert);          // commented out
hitWhat = Alert(kZeroScoresAlert /*1032*/, nil);
switch (hitWhat) {
    case 2:  ZeroHighScores();        fileDirty = true; UpdateMenus(false); break;
    case 3:  ZeroAllButHighestScore(); fileDirty = true; UpdateMenus(false); break;
}
// case 1 (Cancel) falls through with no action
```

**`ALRT` 1032 "Clear Scores"** — byte-verified from the resource fork:
bounds `(40,40,129,320)` = 280 × 89, DITL 1032, `stages` `0x4444`.

| Item | Type | Rect | Size | Text |
|---|---|---|---|---|
| 1 | button | (61,214,81,272) | 58×20 | `Cancel` |
| 2 | button | (61,134,81,206) | 72×20 | `Clear All` |
| 3 | button | (61,54,81,126) | 72×20 | `All But #1` |
| 4 | statText (disabled) | (8,8,56,232) | 224×48 | `Do what?  You can clear all but the highest score, all the scores, or cancel this operation.` |
| 5 | icon (disabled) | (8,240,40,272) | 32×32 | `ICON` 910 |

Note the button order in the DITL is right-to-left: item 1 (Cancel) is rightmost, item 3
(All But #1) leftmost. `stages` `0x4444` = the same for all four stages: default item **1**
(Cancel), no beep. A port must map *button index*, not screen position.

### 9.6 Persistence — where the table actually gets written

The high-score table is part of the house file, so it is saved by `WriteHouse`. The relevant gates
in `GliderPRO/Sources/HouseIO.c`:

| Line | Code | Effect |
|---|---|---|
| `HouseIO.c:327` | `if (gameDirty \|\| fileDirty)` | in the room-switch path |
| `HouseIO.c:329` | `if (houseIsReadOnly) WriteScoresToDisk() else WriteHouse(false)` | |
| `HouseIO.c:428` | `if (houseIsReadOnly) { houseUnlocked = false; if (ReadScoresFromDisk()) {} }` | note the empty `if` body |
| `HouseIO.c:531-535` | `if (gameDirty) { if (houseIsReadOnly) WriteScoresToDisk() else WriteHouse(theMode == kEditMode) }` | in `CloseHouse` |

**All three `'gliS'` calls are gated on `houseIsReadOnly`, and `IsFileReadOnly` is stubbed:**

```c
// GliderPRO/Sources/HouseIO.c:659
static Boolean IsFileReadOnly (FSSpec *theSpec)
{
#pragma unused (theSpec)
    return false;
}
```

So `houseIsReadOnly` is **always false** (set once at `GliderPRO/Sources/HouseIO.c:187`) and the
entire external score-file mechanism in §9.7 is **dead code** in 1.0.4. High scores are only ever
stored in the house file, which means a locked/read-only house on a CD-ROM or a locked floppy
silently loses scores.

It also means `MainWindow.c:76`'s `if (houseIsReadOnly)` branch never runs.

### 9.7 The dead `'gliS'` external score file

Intended for read-only houses: a per-house score file in a subfolder of the system Preferences
folder.

Folder name: `"\pG-PRO Scores \xC4"` — the last byte is **0xC4**, MacRoman **`ƒ`** (florin), which
in the Chicago system font of the era rendered as a distinct glyph often used as a folder marker.
Verified with `grep -a | cat -v` at `GliderPRO/Sources/HighScores.c:654` and `:679` and `:696`.
The literal is therefore `G-PRO Scores ƒ` — note the space before the florin.

| Function | Lines | Behaviour |
|---|---|---|
| `CreateScoresFolder` | `GliderPRO/Sources/HighScores.c:642-661` | `FindFolder(kOnSystemDisk, kPreferencesFolderType, kCreateFolder, &volRefNum, &prefsDirID)` → `CheckFileError(theErr, "\pPrefs Folder")`; `FSMakeFSSpec(volRefNum, prefsDirID, "\pG-PRO Scores ƒ", &scoresSpec)`; `FSpDirCreate(&scoresSpec, smSystemScript, scoresDirID)` → `CheckFileError(theErr, "\pHigh Scores Folder")` |
| `FindHighScoresFolder` | `GliderPRO/Sources/HighScores.c:665-716` | `FindFolder(...)`, then walks the Preferences folder with `PBGetCatInfo`, `ioFDirIndex = count` from 1, testing `(ioFlAttrib & 0x10) == 0x10` (is-a-directory) and `EqualString(name, "\pG-PRO Scores ƒ", true, true)`. On `fnfErr` (ran off the end) calls `CreateScoresFolder`. |
| `OpenHighScoresFile` | `GliderPRO/Sources/HighScores.c:720-738` | `FSpOpenDF(scoreSpec, fsCurPerm, scoresRefNum)`; on `fnfErr`, `FSpCreate(scoreSpec, 'ozm5', 'gliS', smSystemScript)` then re-open. Error strings: `"\pNew High Scores File"`, `"\pHigh Score"` |
| `WriteScoresToDisk` | `GliderPRO/Sources/HighScores.c:742-797` | `FindHighScoresFolder` (fail → `SysBeep(1)`); `FSMakeFSSpec(volRefNum, dirID, thisHouseName, &scoreSpec)` — **the file is named after the house**; `SetFPos(…,fsFromStart,0)`; `byteCount = sizeof(scoresType)` = 292; `FSWrite`; `SetEOF(292)`; `FSClose`. Every step `CheckFileError(theErr, "\pHigh Scores File")` |
| `ReadScoresFromDisk` | `GliderPRO/Sources/HighScores.c:801-855` | same setup, then `GetEOF(scoresRefNum, &byteCount)`, `SetFPos(0)`, `FSRead(scoresRefNum, &byteCount, theScores)` where `theScores = &((*thisHouse)->highScores)` |

**Bug in `ReadScoresFromDisk`:** the read length comes from `GetEOF` (line 823), not from
`sizeof(scoresType)`, and the destination is a fixed 292-byte field inside the house handle
(line 839-841). A `'gliS'` file longer than 292 bytes overruns the house record — corrupting
`savedGame`, `hasGame`, `firstRoom`, `nRooms` and then the room array. Because `OpenHighScoresFile`
creates the file if it is absent, an *empty* file also succeeds with `byteCount = 0`, leaving the
table untouched. Dead code, but a port that revives it must clamp to 292.

Note also that `FSpOpenDF` uses `fsCurPerm`, and the file is opened, read, and closed on **every
room transition** in the read-only case (`HouseIO.c:428`).

### 9.8 `DoHighScores` — the high-score screen

`GliderPRO/Sources/HighScores.c:58-85`:

```
 1. SpinCursor(3)
 2. SetPort(workSrcMap); PaintRect(&workSrcRect)                     // black out the offscreen
 3. QSetRect(&tempRect, 0, 0, 640, 480); QOffsetRect(&tempRect, splashOriginH, splashOriginV)
    LoadScaledGraphic(kStarPictID /*1995*/, &tempRect)               // starfield backdrop
 4. // if (quickerTransitions) DissBitsChunky(&workSrcRect); else DissBits(&workSrcRect);
    //   BOTH COMMENTED OUT at HighScores.c:68-71
 5. SpinCursor(3); SetPort(workSrcMap); DrawHighScores()
 6. // second DissBits pair also COMMENTED OUT at HighScores.c:76-79
 7. InitCursor(); DelayTicks(60)                                      // 1 second
 8. WaitForInputEvent(30)                                            // up to 30 SECONDS
 9. RedrawSplashScreen()
```

`WaitForInputEvent(short seconds)` (`GliderPRO/Sources/Utilities.c:439-479`) is the shared
"press anything to continue" primitive:

```
 1. timeToBail = TickCount() + 60L * (long)seconds
 2. FlushEvents(everyEvent, 0); waiting = true; didResume = false
 3. while (waiting) {
        GetKeys(theKeys)
        if (BitTst(&theKeys, kCommandKeyMap /*48*/) || BitTst(&theKeys, kOptionKeyMap /*61*/) ||
            BitTst(&theKeys, kShiftKeyMap /*63*/)   || BitTst(&theKeys, kControlKeyMap /*60*/))
            waiting = false                          // modifier keys alone dismiss it
        if (GetNextEvent(everyEvent, &theEvent)) {
            if (theEvent.what == mouseDown || theEvent.what == keyDown) waiting = false
            else if (theEvent.what == osEvt && (theEvent.message & 0x01000000)) {   // suspend/resume
                if (theEvent.message & 0x00000001) { didResume = true; waiting = false; }
                else InitCursor();
            }
        }
        if ((seconds != -1) && (TickCount() >= timeToBail)) waiting = false
    }
 4. FlushEvents(everyEvent, 0); return didResume
```

`seconds == -1` means wait forever. The return value is `true` only if the loop exited because the
app was **resumed** from the background, which callers use to skip a redraw.

Because both dissolve calls are commented out, the high-score screen is composed entirely in
`workSrcMap` and **never blitted to the screen by `DoHighScores` itself** — it becomes visible
only via whatever `RedrawSplashScreen`/the update mechanism does. This is one of the transition
regressions introduced when the `DissBits` code was disabled project-wide (see also
`GliderPRO/Sources/Play.c`, where all four `quickerTransitions` branches are commented out). A
faithful port should implement the dissolve; a *literal* port reproduces a screen that flashes.

`kStarPictID` = 1995 (`GliderPRO/Headers/GliderDefines.h:534`).

Called from exactly two places:

| Caller | Line | Trigger |
|---|---|---|
| `DoHighScores` menu handler | `GliderPRO/Sources/Menu.c:418` | Options ▸ High Scores… |
| `TestHighScore` | `GliderPRO/Sources/HighScores.c:421` | after a qualifying game |

### 9.9 `DrawHighScores` — exact geometry

`GliderPRO/Sources/HighScores.c:94-276`. Constants (`GliderPRO/Sources/HighScores.c:90-92`):

| Constant | Value |
|---|---|
| `kScoreSpacing` | 18 |
| `kScoreWide` | 352 |
| `kKimsLifted` | 4 |
| `kHighScoresPictID` | 1994 |
| `kHighScoresMaskID` | 1998 |

Origin:

```c
scoreLeft = ((thisMac.screen.right - thisMac.screen.left) - kScoreWide) / 2;   // horizontally centred
dropIt    = 129 + splashOriginV;
```

Note the asymmetry: `scoreLeft` is derived from the **screen width**, but `dropIt` from
`splashOriginV` (the 640×480 splash box origin). On a 640×480 screen they agree
(`scoreLeft = (640-352)/2 = 144`, `splashOriginV = 0`, `dropIt = 129`).

**Step 1 — the masked title graphic.** A 332×30 offscreen GWorld holds `PICT` 1994 at
`kPreferredDepth` (8); a second 332×30 1-bit GWorld holds `PICT` 1998 as the mask; `CopyMask`
composites them into `workSrcMap` at:

```
tempRect2 = (0,0,332,30) offset by (scoreLeft + (352-332)/2, dropIt - 60)
          = (scoreLeft + 10, dropIt - 60) .. (+332, +30)
```

**Step 2 — the house name.** `TextFont(applFont); TextFace(bold); TextSize(14)`. The string is
`"\p\xA5 "` + `thisHouseName` + `"\p \xA5"` — byte **0xA5** is MacRoman **`•`** (bullet), verified
with `sed -n '137,139p' HighScores.c | cat -v` → `M-%` = 0x80|0x25 = 0xA5. The rendered text is
therefore `• Slumberland •`. Drawn twice for a drop shadow:

| Pass | Colour | `MoveTo` |
|---|---|---|
| shadow | `blackColor` | `(scoreLeft + (352 - StringWidth)/2 - 1, dropIt - 66)` |
| fill | `cyanColor` | `(scoreLeft + (352 - StringWidth)/2, dropIt - 65)` |

**Step 3 — the banner** (`highScores.banner`, "message for score #1").
`TextFont(applFont); TextFace(bold); TextSize(12)` for everything from here down.

| Pass | Colour | `MoveTo` |
|---|---|---|
| shadow | `blackColor` | `(scoreLeft + (352 - bannerWidth)/2, dropIt - kKimsLifted)` = `dropIt - 4` |
| fill | `yellowColor` | `(scoreLeft + (352 - bannerWidth)/2, dropIt - kKimsLifted - 1)` = `dropIt - 5` |

Then a double frame around it:

```
QSetRect(&tempRect, 0, 0, bannerWidth + 8, kScoreSpacing)     // signature is (rect, l, t, r, b)
    -> left=0, top=0, right=bannerWidth+8, bottom=18   i.e. (bannerWidth+8) x 18
QOffsetRect(&tempRect, scoreLeft - 3 + (352 - bannerWidth)/2,
                       dropIt + 5 - kScoreSpacing - kKimsLifted)   // dy = dropIt - 17
ForeColor(blackColor); FrameRect(&tempRect)
QOffsetRect(&tempRect, -1, -1); ForeColor(yellowColor); FrameRect(&tempRect)
```

`QSetRect` is `void QSetRect (Rect *theRect, short l, short t, short r, short b)`
(`GliderPRO/Sources/RectUtils.c:210-217`) — **left, top, right, bottom**, not the QuickDraw
`SetRect(r, left, top, right, bottom)` order coincidence: it is the same order, but note it is
*not* (t,l,b,r) like a raw `Rect` literal in Rez. Every `QSetRect` call in this document uses
(l,t,r,b).

So the banner box spans y = `dropIt - 17` .. `dropIt + 5` (black) and one pixel up-left (yellow).

**Step 4 — the ten rows.** Only rows with `scores[i] > 0L` are drawn
(`GliderPRO/Sources/HighScores.c:176`). Each row draws five fields, each **twice** (black shadow
at (x, y), then colour at (x-1, y-1)). The baseline y depends on `i`:

| `i` | shadow baseline | colour baseline |
|---|---|---|
| 0 | `dropIt - kScoreSpacing - kKimsLifted` = `dropIt - 22` | `dropIt - 23` |
| 1..9 | `dropIt + i*18` | `dropIt + i*18 - 1` |

Row 0 is therefore drawn **above** the banner box, rows 1-9 below it. Visually top to bottom:
title graphic, house name, **#1 row**, banner box, #2..#10 rows.

Field x positions (shadow x, then colour x = shadow x − 1):

| Field | Source | shadow x | colour (non-winner) | colour (`i == lastHighScore`) |
|---|---|---|---|---|
| place number | `NumToString(i+1)` | `scoreLeft + 1` | `cyanColor` at `scoreLeft + 0` | `whiteColor` |
| name | `highScores.names[i]` | `scoreLeft + 31` | `yellowColor` at `scoreLeft + 30` | `whiteColor` |
| level number | `NumToString(highScores.levels[i])` | `scoreLeft + 161` | `yellowColor` at `scoreLeft + 160` | `whiteColor` |
| the word room(s) | localized 6 or 7 | `scoreLeft + 193` | `cyanColor` at `scoreLeft + 192` | *always cyan* |
| points | `NumToString(highScores.scores[i])` | `scoreLeft + 291` | `yellowColor` at `scoreLeft + 290` | `whiteColor` |

The room/rooms word is the only field **not** highlighted for the winner
(`GliderPRO/Sources/HighScores.c:240` unconditionally sets `cyanColor`) — a genuine
inconsistency, not a bug with consequences. Singular/plural:

```c
if (thisHousePtr->highScores.levels[i] == 1) GetLocalizedString(6, tempStr);   // "room"
else                                        GetLocalizedString(7, tempStr);   // "rooms"
```

`SpinCursor(1)` is called once per drawn row (`GliderPRO/Sources/HighScores.c:178`).

**Step 5 — the footer.** `ForeColor(blueColor); TextFont(applFont); TextFace(bold); TextSize(9);
MoveTo(scoreLeft + 80, dropIt - 1 + (10 * kScoreSpacing))` = `dropIt + 179`, then
`GetLocalizedString(8, tempStr)` = `Click Mouse or Hit a Key to Exit`. Drawn once, no shadow.

Finally `ForeColor(blackColor)` and `HSetState`.

The colour names are QuickDraw's classic 8-colour constants (`blackColor`, `whiteColor`,
`cyanColor`, `yellowColor`, `blueColor`) — `ForeColor` on a Color GrafPort maps these to the
nearest CLUT entry. A Go port should use the literal RGBs QuickDraw used:
black `#000000`, white `#FFFFFF`, cyan `#00FFFF`, yellow `#FFFF00`, blue `#0000FF`.

### 9.10 `DLOG` 1020 — "Enter High Score Name"

Byte-verified: `DLOG` 1020 `'High Name'` at `GliderPRO/Glider PRO.r:6218`, bounds
`(0,0,109,316)` = **316 × 109**, `procID` 1 = `dBoxProc` (the classic WDEF IDs are
`documentProc` = 0, `dBoxProc` = 1, `plainDBox` = 2, `altDBoxProc` = 3 — so this is the same modal
frame every other Glider PRO dialog uses, not a document window), visible 1, goAwayFlag 1,
`refCon` 0, DITL 1020. Because `dBoxProc` has no title bar or close box, the `goAwayFlag` is inert.

**The bounds are (0,0)-anchored and all centering is commented out** (`BringUpDialog` calls
`CenterDialog` — see §4.2 — but in 1.0.4 that call is `//`-ed out), so this dialog appears at the
very top-left of the screen, with its top ~20 rows behind the menu bar (there is no title bar —
`dBoxProc` has none). 18 other dialogs share this problem; see P3.

DITL 1020 (`GliderPRO/Glider PRO.r:4671`), 6 items:

| Item | Type | Rect (t,l,b,r) | Size | Text / notes |
|---|---|---|---|---|
| 1 | button | (81,250,101,308) | 58×20 | `Okay` — `kOkayButton` |
| 2 | editText | (52,157,68,297) | 140×16 | `Your Name` — `kHighNameItem` |
| 3 | statText (disabled) | (8,8,41,308) | 300×33 | `Your score of ^0 is #^1 on the top ten high scores for ^2.` |
| 4 | statText (disabled) | (49,16,81,132) | 116×32 | `Enter your name:\r(15 letters max.)` |
| 5 | statText (disabled) | (81,154,97,173) | 19×16 | *(empty)* — `kNameNCharsItem`, the live character counter |
| 6 | statText (disabled) | (81,175,97,224) | 49×16 | `letters` |

Item constants: `kHighNameDialogID 1020`, `kHighNameItem 2`, `kNameNCharsItem 5`
(`GliderPRO/Sources/HighScores.c:25, 27, 28`).

`ParamText` substitution (`GliderPRO/Sources/HighScores.c:509-511`):

| Token | Value |
|---|---|
| `^0` | `NumToString(theScore)` |
| `^1` | `NumToString(place)` where `place = placing + 1`, i.e. 1..10 |
| `^2` | `thisHouseName` |
| `^3` | `"\p"` (empty) |

`GetHighScoreName(short place)` (`GliderPRO/Sources/HighScores.c:498-533`):

```
 1. nameFilterUPP = NewModalFilterUPP(NameFilter)
 2. InitCursor()
 3. NumToString(theScore, scoreStr); NumToString(place, placeStr)
    ParamText(scoreStr, placeStr, thisHouseName, "\p")
 4. PlayPrioritySound(kEnergizeSound /*6*/, kEnergizePriority)
 5. BringUpDialog(&theDial, kHighNameDialogID)
 6. FlushEvents(everyEvent, 0)
 7. SetDialogString(theDial, kHighNameItem, highName)          // pre-fill with last name used
 8. SelectDialogItemText(theDial, kHighNameItem, 0, 1024)      // select all
 9. leaving = false
    while (!leaving) {
        ModalDialog(nameFilterUPP, &item)
        if (item == kOkayButton) {
            GetDialogString(theDial, kHighNameItem, tempStr)
            PasStringCopyNum(tempStr, highName, 15)            // truncate to 15
            leaving = true
        }
    }
10. DisposeDialog(theDial); DisposeModalFilterUPP(nameFilterUPP)
```

**There is no Cancel.** The loop exits only on `kOkayButton`, and the only way to produce
`kOkayButton` is to click Okay or press Return/Enter. An empty name is accepted (it becomes a
zero-length Pascal string, which draws as nothing in the table).

`UpdateNameDialog` (`GliderPRO/Sources/HighScores.c:431-440`):

```c
DrawDialog(theDialog);
DrawDefaultButton(theDialog);                       // the 3-pixel rounded outline around item 1
nChars = GetDialogStringLen(theDialog, kHighNameItem);
SetDialogNumToStr(theDialog, kNameNCharsItem, (long)nChars);
```

`NameFilter` (`GliderPRO/Sources/HighScores.c:445-493`):

```
 0. if (keyStroke) {                       // set by the PREVIOUS keyDown, so the counter is
        nChars = GetDialogStringLen(dial, kHighNameItem)   // one event behind — see below
        SetDialogNumToStr(dial, kNameNCharsItem, nChars)
        keyStroke = false
    }
 1. switch (event->what)
    case keyDown:
        keyStroke = true
        switch (event->message & charCodeMask)
            case kReturnKeyASCII (13):
            case kEnterKeyASCII (3):
                PlayPrioritySound(kCarriageSound /*53*/, kCarriagePriority)
                FlashDialogButton(dial, kOkayButton)
                *item = kOkayButton; return true
            case kTabKeyASCII (9):
                SelectDialogItemText(dial, kHighNameItem, 0, 1024); return false
            default:
                PlayPrioritySound(kTypingSound /*52*/, kTypingPriority)
                return false
    case updateEvt:
        BeginUpdate(GetDialogWindow(dial)); UpdateNameDialog(dial); EndUpdate(...)
        event->what = nullEvent; return false
    default: return false
```

The `keyStroke` deferral at the top is deliberate: `ModalDialog` calls the filter *before* the
Dialog Manager inserts the typed character into the edit field, so the count must be recomputed on
the *next* pass through the filter (which happens on the next event, including null events, since
`ModalDialog` calls the filter for those too). Net effect: the counter updates one event late but
appears instantaneous because null events arrive continuously. **A Go port that recomputes the
count synchronously after insertion is more correct and visually identical.**

Every keystroke plays `kTypingSound` (52) at `kTypingPriority`; Return/Enter plays
`kCarriageSound` (53) — a typewriter carriage return. This is the only place in the UI where
typing is audible besides the banner dialog.

### 9.11 `DLOG` 1021 — "Enter High Score Banner"

Byte-verified: `DLOG` 1021 `'Banner'` at `GliderPRO/Glider PRO.r:6223`, bounds `(40,40,162,356)`
= **316 × 122**, `procID` 1, visible, goAway, DITL 1021 (resource name `'High Banner'`).
Unlike 1020 this one has a sane (40,40) origin.

DITL 1021 (`GliderPRO/Glider PRO.r:4687`), 5 items:

| Item | Type | Rect (t,l,b,r) | Size | Text / notes |
|---|---|---|---|---|
| 1 | button | (94,250,114,308) | 58×20 | `Okay` |
| 2 | editText | (67,11,83,305) | 294×16 | *(empty)* — `kHighBannerItem` |
| 3 | statText (disabled) | (8,11,56,305) | 294×48 | `Getting #1 on the high scores entitles you to change the high score banner.\r(31 letters max.)` |
| 4 | statText (disabled) | (94,29,110,78) | 49×16 | `letters` |
| 5 | statText (disabled) | (94,8,110,27) | 19×16 | *(empty)* — `kBannerScoreNCharsItem`, the counter |

Item constants: `kHighBannerDialogID 1021`, `kHighBannerItem 2`, `kBannerScoreNCharsItem 5`
(`GliderPRO/Sources/HighScores.c:26, 29, 30`). Note items 4 and 5 are in the opposite DITL order
from dialog 1020 (counter is item 5 in both, but here it sits to the *left* of the word "letters"
by rect, same as 1020 — 1020 has counter at x 154..173 and "letters" at 175..224; 1021 has counter
at 8..27 and "letters" at 29..78).

`GetHighScoreBanner(void)` (`GliderPRO/Sources/HighScores.c:608-638`) is the same shape as
`GetHighScoreName` minus `InitCursor`, `NumToString`, `ParamText` and `FlushEvents`:

```
 1. bannerFilterUPP = NewModalFilterUPP(BannerFilter)
 2. PlayPrioritySound(kEnergizeSound, kEnergizePriority)
 3. BringUpDialog(&theDial, kHighBannerDialogID)
 4. SetDialogString(theDial, kHighBannerItem, highBanner)
 5. SelectDialogItemText(theDial, kHighBannerItem, 0, 1024)
 6. while (!leaving) { ModalDialog(bannerFilterUPP, &item);
        if (item == kOkayButton) { GetDialogString(...); PasStringCopyNum(tempStr, highBanner, 31); leaving = true; } }
 7. DisposeDialog; DisposeModalFilterUPP
```

`UpdateBannerDialog` (`GliderPRO/Sources/HighScores.c:538-547`) and `BannerFilter`
(`GliderPRO/Sources/HighScores.c:552-601`) are byte-for-byte the same logic as their name-dialog
counterparts, substituting `kHighBannerItem` / `kBannerScoreNCharsItem`. Same missing Cancel, same
`keyStroke` deferral, same two sounds.

Both dialogs share the single global `keyStroke`, which is fine because they are never open
simultaneously (`GetHighScoreBanner` is called from inside `TestHighScore` *after*
`GetHighScoreName` returns).

### 9.12 `ALRT` 1046 — the "no high score on a resumed game" alert

Reached from `HeyYourPissingAHighScore()` (`GliderPRO/Sources/Menu.c:779-786`):

```c
#define		kNoHighScoreAlert	1046
short		whoCares;
InitCursor();
//	CenterAlert(kNoHighScoreAlert);
whoCares = Alert(kNoHighScoreAlert, nil);
```

`ALRT` 1046 `'No High Score'`: bounds `(40,40,132,300)` = 260 × 92, DITL 1046, `stages` `0x4444`
(default item 1, no beep, all four stages identical).

| Item | Type | Rect | Size | Text |
|---|---|---|---|---|
| 1 | button | (64,180,84,252) | 72×20 | `So What?` |
| 2 | statText (disabled) | (8,8,56,216) | 208×48 | `If you resume a saved game, you are ineligible to get on the high scores for that game.` |
| 3 | icon (disabled) | (8,220,40,252) | 32×32 | `ICON` 1073 |

Called from exactly one place, `GliderPRO/Sources/Menu.c:319`, immediately after
`resumedSavedGame = true` and immediately before the `OpenSavedGame()` that always fails (§8.7).
So in 1.0.4 this alert is a warning about a consequence the user then never gets to enjoy — and the
`resumedSavedGame = true` it warns about **is not undone**, permanently disabling high scores until
the next New Game (`GliderPRO/Sources/Menu.c:307`) or Two Player Game
(`GliderPRO/Sources/Menu.c:313`).
---

## 10. Splash screen, banner, and About box

### 10.1 The splash screen

The splash screen *is* the main window in `kSplashMode` and `kPlayMode`. There is no separate
"title screen" state machine; the app opens one full-screen window and paints `PICT` 1000 into it.

Constants:

| Constant | Value | Defined | Notes |
|---|---|---|---|
| `kSplash8BitPICT` | 1000 | `GliderPRO/Headers/GliderDefines.h:524` | verified 640 × 460 in the resource fork |
| `kStarPictID` | 1995 | `GliderPRO/Headers/GliderDefines.h:534` | 640 × 460 starfield, used by the high-score screen |
| `kIdleSplashTicks` | `7200L` | `GliderPRO/Headers/GliderDefines.h:197` | 120 seconds at 60 ticks/s |

Splash origin (`GliderPRO/Sources/MainWindow.c:238-243`):

```c
splashOriginH = ((thisMac.screen.right - thisMac.screen.left) - 640) / 2;
if (splashOriginH < 0) splashOriginH = 0;
splashOriginV = ((thisMac.screen.bottom - thisMac.screen.top) - 480) / 2;
if (splashOriginV < 0) splashOriginV = 0;
```

Note the asymmetry: **the horizontal centring uses 640 (the PICT width) but the vertical uses 480,
while the PICT is only 460 tall.** So on a 640×480 screen `splashOriginV = 0` and the 460-tall
splash sits at the top with a 20-pixel black strip at the bottom (which is where the menu-bar cover
window lives). On a 832×624 screen `splashOriginV = (624-480)/2 = 72`, i.e. the splash is centred as
if it were 480 tall, leaving 92 px below and 72 above — deliberately slightly high.

Window setup for splash/play mode (`GliderPRO/Sources/MainWindow.c:212-236`):

```
 1. if (menuWindow == nil) {
        menuWindow = GetNewCWindow(kMenuWindowID, nil, kPutInFront)
        SizeWindow(menuWindow, RectWide(&thisMac.screen), 20, false)
        MoveWindow(menuWindow, thisMac.screen.left, thisMac.screen.top, true)
        ShowWindow(menuWindow)
    }                                       // a 20-px-tall window that covers the menu bar
 2. mainWindowRect = thisMac.screen; ZeroRectCorner(&mainWindowRect)
    mainWindowRect.bottom -= 20             // literal 20, comment says "thisMac.menuHigh"
 3. mainWindow = GetNewCWindow(kMainWindowID, nil, kPutInFront)
    SizeWindow(mainWindow, width, height, false)
    MoveWindow(mainWindow, thisMac.screen.left, thisMac.screen.top + 20, true)
    ShowWindow(mainWindow); SetPortWindowPort(mainWindow); ClipRect(&mainWindowRect)
 4. ForeColor(blackColor); BackColor(whiteColor); PaintRect(&mainWindowRect)
 5. compute splashOriginH/V (above)
 6. SetPort(workSrcMap); PaintRect(&workSrcRect); LoadGraphic(kSplash8BitPICT)
 7. // the (fadeGraysOut && isDoColorFade) colour-fade block is COMMENTED OUT (MainWindow.c:250+)
```

The `20` for menu-bar height is hard-coded in three places with a `// thisMac.menuHigh` comment
each time (`GliderPRO/Sources/MainWindow.c:224, 229` and `:217`). A port on a modern windowing
system has no menu bar to hide; `menuWindow` and `UpdateMenuBarWindow`
(`GliderPRO/Sources/MainWindow.c:160-169`, "Ugly kludge to cover over the menu bar when playing
game on 2nd monitor") should be dropped.

### 10.2 `DrawOnSplash` — the text overlaid on the splash PICT

`GliderPRO/Sources/MainWindow.c:56-94`:

```
 1. PasStringCopy("\pHouse: ", houseLoadedStr)
    PasStringConcat(houseLoadedStr, thisHouseName)
    if (thisMac.hasQT && hasMovie) PasStringConcat(houseLoadedStr, "\p (QT)")
 2. TextSize(9); TextFace(1 /*bold*/); TextFont(applFont)
 3. MoveTo(splashOriginH + 436, splashOriginV + 314)
 4. if (thisMac.isDepth == 4) {                       // 4-bit (16-colour) screen
        ForeColor(whiteColor); DrawString(houseLoadedStr); ForeColor(blackColor)
    } else {
        if (houseIsReadOnly) ColorText(houseLoadedStr, 5L)     // CLUT index 5
        else                 ColorText(houseLoadedStr, 28L)    // CLUT index 28
    }
 5. #if defined(powerc) || defined(__powerc)
        TextSize(12); TextFace(0); TextFont(systemFont)
        ForeColor(blackColor); MoveTo(splashOriginH + 5, splashOriginV + 457)
        DrawString("\pPowerPC Native!")                        // 1-px drop shadow
        ForeColor(whiteColor);  MoveTo(splashOriginH + 4, splashOriginV + 456)
        DrawString("\pPowerPC Native!")
        ForeColor(blackColor)
    #endif
```

So the splash carries a live `House: <name>` label at (436, 314) relative to the splash origin, in
9-point bold application font, coloured CLUT index **28** normally and **5** for a read-only house
(never, because `houseIsReadOnly` is always false — §9.6). `PowerPC Native!` at the bottom left is
compile-time conditional; the shipped PowerPC build has it, the 68k build does not.

The `(436, 314)` label position matches the `InvalWindowRect` that `DoOpenDocAE` issues after
switching houses via drag-and-drop
(`GliderPRO/Sources/AppleEvents.c`: `SetRect(&updateRect, splashOriginH + 474, splashOriginV + 304,
splashOriginH + 474 + 166, splashOriginV + 304 + 12)`) — note the rect starts at **474**, 38 px to
the right of where the text is actually drawn, so a long house name's left portion is not
invalidated and stale pixels can remain.

### 10.3 `RedrawSplashScreen` and `UpdateMainWindow`

`RedrawSplashScreen` (`GliderPRO/Sources/MainWindow.c:98-114`):

```
 1. SetPort(workSrcMap); PaintRect(&workSrcRect)               // black
 2. QSetRect(&tempRect, 0, 0, 640, 460); QOffsetRect(&tempRect, splashOriginH, splashOriginV)
    LoadScaledGraphic(kSplash8BitPICT, &tempRect)
 3. DrawOnSplash()
 4. SetPortWindowPort(mainWindow)
 5. // if (quickerTransitions) DissBitsChunky(&workSrcRect); else DissBits(&workSrcRect);
    //   BOTH COMMENTED OUT at MainWindow.c:109-112
 6. CopyRectMainToWork(&workSrcRect)
```

**Step 6 copies the wrong way.** `CopyRectMainToWork` copies *from* the main window *to* the work
GWorld, which discards the freshly composed splash instead of showing it. Combined with the
commented-out `DissBits`, `RedrawSplashScreen` leaves the screen unchanged and clobbers `workSrcMap`
with whatever was on screen. Callers are `GliderPRO/Sources/HighScores.c:84` (end of
`DoHighScores`), `GliderPRO/Sources/GameOver.c:68` and `GliderPRO/Sources/GameOver.c:505`. A port
should use the `Work→Main` direction here; this is one of the clearest artefacts of the disabled
transition code.

`UpdateMainWindow` (`GliderPRO/Sources/MainWindow.c:120-155`) is the update-event handler and does
work out correctly:

```
 1. dummyRgn = NewRgn(); SetPortWindowPort(mainWindow)
 2. if (theMode == kEditMode) {
        PauseMarquee()
        CopyBits(workSrcMap -> mainWindow port, mainWindowRect, mainWindowRect, srcCopy,
                 GetPortVisibleRegion(...))
        ResumeMarquee()
    }
 3. else if (theMode == kSplashMode || theMode == kPlayMode) {
        SetPort(workSrcMap); PaintRect(&workSrcRect)
        QSetRect(&tempRect, 0, 0, 640, 460); QOffsetRect(&tempRect, splashOriginH, splashOriginV)
        LoadScaledGraphic(kSplash8BitPICT, &tempRect)
        CopyBits(workSrcMap -> mainWindow port, workSrcRect, mainWindowRect, srcCopy, visRgn)
        SetPortWindowPort(mainWindow); DrawOnSplash()
    }
 4. DisposeRgn(dummyRgn); splashDrawn = true
```

Note that in `kPlayMode` an update event **repaints the splash screen**, not the game. That is
acceptable only because in play mode the menu bar is hidden and updates are not expected; if a port
delivers resize/expose events during gameplay it must not reuse this branch.

`LoadScaledGraphic` re-reads and re-scales `PICT` 1000 from the resource fork on **every** update
event. A Go port should decode the splash once into an image and blit.

### 10.4 Idle behaviour: the auto-demo

`GliderPRO/Sources/Events.c:536-540`:

```c
if ((theMode == kSplashMode) && (doAutoDemo) && (!switchedOut))
{
    if (TickCount() >= incrementModeTime)
        DoDemoGame();
}
```

`incrementModeTime` is reset to `TickCount() + kIdleSplashTicks` (`7200L` = 2 minutes) on every
user interaction and after every mode change. `doAutoDemo` is a preference
(`prefsInfo.wasAutoDemo`, default `true`). `DoDemoGame`
(`GliderPRO/Sources/Play.c:282-303`) plays back the `'demo'` 128 resource, which is
`kDemoLength` = 6702 bytes of `demoType {long frame; char key; char padding;}` records
(6 bytes each = 1117 events) loaded once at startup into `demoData`
(`GliderPRO/Sources/StructuresInit2.c`).

### 10.5 The opening banner — `Banner.c`

The banner is a picture of a sheet of ruled notebook paper with the house author's message typed on
it, shown once at the start of a new game.

Resource IDs (`GliderPRO/Sources/Banner.c:17-21`), all byte-verified from the resource fork:

| Constant | ID | Dimensions | Depth | Purpose |
|---|---|---|---|---|
| `kBannerPageTopPICT` | 1993 | 330 × 190 | colour | top 190 px of the notebook page |
| `kBannerPageBottomPICT` | 1992 | 330 × 30 | colour | bottom 30 px (the torn/ragged edge) |
| `kBannerPageBottomMask` | 1991 | 330 × 30 | 1-bit | mask for the ragged edge |
| `kStarsRemainingPICT` | 1017 | 256 × 64 | colour | "N stars remaining" plaque (plural) |
| `kStarRemainingPICT` | 1018 | 256 × 64 | colour | "1 star remaining" plaque (singular) |

Globals (`GliderPRO/Sources/Banner.c:28-29`):

| Global | Type | Set by |
|---|---|---|
| `numStarsRemaining` | `short` | `CountStarsInHouse()` on new game, `smallGame.wasStarsLeft` on resume (`GliderPRO/Sources/Play.c:312`) |
| `bannerStarCountOn` | `Boolean` | `GliderPRO/Sources/HouseIO.c:418`: `bannerStarCountOn = (((*thisHouse)->flags & 0x00000004) == 0x00000000);` |

**The star-count flag is inverted.** `houseType.flags` bit 2 (`0x00000004`) *set* means
"do **not** show the star count". Of the 22 shipped houses only **Art Museum** has bit 2 set
(`flags == 0x00000006`); the other 21 show the star count (§8.8).
The full `flags` decode, for reference:

| Bit | Mask | Name | Sense |
|---|---|---|---|
| 0 | `0x00000001` | `wardBit` | set = on |
| 1 | `0x00000002` | `phoneBitSet` | set = on |
| 2 | `0x00000004` | `bannerStarCountOn` | **set = OFF** (inverted) |

#### `DrawBanner (Point *topLeft)` — `GliderPRO/Sources/Banner.c:41-84`

```
 1. GetGWorld(&wasCPort, &wasWorld)
 2. QSetRect(&wholePage, 0, 0, 330, 220)                  // 330 wide x 220 tall
    mapBounds = thisMac.screen; ZeroRectCorner(&mapBounds)
    CenterRectInRect(&wholePage, &mapBounds)              // centre on the screen
    topLeft->h = wholePage.left; topLeft->v = wholePage.top      // returned to caller
 3. partPage = wholePage; partPage.bottom = partPage.top + 190
    SetGWorld(workSrcMap, nil)
    LoadScaledGraphic(kBannerPageTopPICT /*1993*/, &partPage)    // top 190 px, unmasked
 4. partPage = wholePage; partPage.top = partPage.bottom - 30    // bottom 30 px
    mapBounds = partPage; ZeroRectCorner(&mapBounds)
    CreateOffScreenGWorld(&tempMap,  &mapBounds, kPreferredDepth /*8*/)
    SetGWorld(tempMap, nil);  LoadGraphic(kBannerPageBottomPICT /*1992*/)
    CreateOffScreenGWorld(&tempMask, &mapBounds, 1)
    SetGWorld(tempMask, nil); LoadGraphic(kBannerPageBottomMask /*1991*/)
 5. CopyMask(tempMap, tempMask, workSrcMap, &mapBounds, &mapBounds, &partPage)
 6. SetPort(workSrcMap); SetGWorld(wasCPort, wasWorld)
    DisposeGWorld(tempMap); DisposeGWorld(tempMask)
```

So the page is 330 × 220, screen-centred; the top 190 rows are a plain blit and the bottom 30 rows
are masked so the ragged paper edge shows the room behind it.

#### `DrawBannerMessage (Point topLeft)` — `GliderPRO/Sources/Banner.c:117-166`

```
 1. HLock(thisHouse); PasStringCopy((*thisHouse)->banner, bannerStr); HSetState(...)
    // banner is Str255 at houseType offset 16
 2. TextFont(applFont); TextFace(bold); TextSize(12); ForeColor(blackColor)
 3. count = 0
    do {
        GetLineOfText(bannerStr, count, subStr)
        MoveTo(topLeft.h + 16, topLeft.v + 32 + (count * 20))
        DrawString(subStr)
        count++
    } while (subStr[0] > 0)
    // CR-delimited lines, 20 px apart, first baseline at topLeft.v + 32, left margin 16 px.
    // NOTE: the loop draws one extra (empty) line before exiting.
 4. if (bannerStarCountOn) {
        bannerStr = GetLocalizedString(numStarsRemaining != 1 ? 1 : 2)   // "There are " / "There is "
        PasStringConcat(bannerStr, NumToString(numStarsRemaining))
        PasStringConcat(bannerStr, GetLocalizedString(numStarsRemaining != 1 ? 3 : 4))
                                              // " stars in the house." / " star in the house."
        ForeColor(redColor)
        MoveTo(topLeft.h + 16, topLeft.v + 164); DrawString(bannerStr)
        MoveTo(topLeft.h + 16, topLeft.v + 180); DrawString(GetLocalizedString(5))
                                              // "Get every star to win."
    }
 5. ForeColor(blackColor)
```

The exact localized strings, byte-verified from `STR#` 150 `'Localized Strings'`
(`GliderPRO/Glider PRO.r:396`):

| Index | String |
|---|---|
| 1 | `There are ` (trailing space) |
| 2 | `There is ` (trailing space) |
| 3 | ` stars in the house.` (leading space) |
| 4 | ` star in the house.` (leading space) |
| 5 | `Get every star to win.` |

So with 5 stars the two red lines read `There are 5 stars in the house.` and
`Get every star to win.`

Because the star lines are drawn at fixed y = `topLeft.v + 164` and `+ 180`, a banner message with
more than **6** lines (`32 + 6*20 = 152`, `32 + 7*20 = 172`) collides with them. The house editor's
banner field allows 255 characters (localized string 11: `Enter New-Game Message here (max. 255
characters)`), so this is reachable.

`GetLineOfText (StringPtr srcStr, short index, StringPtr textLine)`
(`GliderPRO/Sources/StringUtils.c:140-208`):

```
 1. textLine = ""; srcLength = srcStr[0]
 2. if (index == 0) start = 1
    else {
        start = 0; count = 0; i = 0; foundIt = false
        do { i++;
             if (srcStr[i] == kReturnKeyASCII /*13*/) { count++;
                 if (count == index) { start = i + 1; foundIt = true; } }
        } while (i < srcLength && !foundIt)
    }
 3. if (start != 0) {
        i = start; foundIt = false
        do { if (srcStr[i] == 13) { stop = i; foundIt = true; } i++ } while (i < srcLength && !foundIt)
        if (!foundIt) {
            if (start > srcLength) { start = srcLength; stop = srcLength - 1; }
            else stop = i;
        }
        count = 0
        for (i = start; i <= stop; i++) { count++; textLine[count] = srcStr[i]; }
        textLine[0] = count
    }
```

Quirks a port must replicate or deliberately fix:

* When a CR terminates the line, `stop = i` is the index **of the CR**, and the copy loop is
  `i <= stop`, so **the CR itself is copied into the returned line**. `DrawString` renders CR as a
  glyph (or nothing) depending on the font — in the application font of the era it draws nothing
  visible, which is why nobody noticed.
* When the last line has no trailing CR, `stop = i` after the loop increments past `srcLength`, so
  `stop == srcLength + 1` in the worst case and the copy reads 1-2 bytes past the string's logical
  end (still inside the 256-byte `Str255` buffer, so harmless but garbage-appending).
* `start == 0` (index past the last CR) returns an empty line, which is the loop's exit condition
  in `DrawBannerMessage`.

#### `CountStarsInHouse` — `GliderPRO/Sources/Banner.c:89-111`

```
 1. numStars = 0; HLock(thisHouse); numRooms = (*thisHouse)->nRooms
 2. for (i = 0; i < numRooms; i++)
        if ((*thisHouse)->rooms[i].suite != kRoomIsEmpty /*-1*/)
            for (h = 0; h < kMaxRoomObs /*24*/; h++)
                if ((*thisHouse)->rooms[i].objects[h].what == kStar /*0x2C = 44*/) numStars++
 3. HSetState(...); return numStars
```

`kStar` = `0x2C` = 44 (`GliderPRO/Headers/GliderDefines.h:355`), `kRoomIsEmpty` = −1
(`GliderPRO/Headers/GliderDefines.h:525`).

#### `BringUpBanner` — `GliderPRO/Sources/Banner.c:171-198`

```
 1. DrawBanner(&topLeft)
 2. DrawBannerMessage(topLeft)
 3. // if (quickerTransitions) DissBitsChunky(&justRoomsRect); else DissBits(&justRoomsRect);
    //   COMMENTED OUT at Banner.c:178-181 (comment: "was workSrcRect")
 4. QSetRect(&wholePage, 0, 0, 330, 220); QOffsetRect(&wholePage, topLeft.h, topLeft.v)
    CopyBits(backSrcMap -> workSrcMap, &wholePage, &wholePage, srcCopy, nil)
        // restores the room graphics UNDER where the banner was, into workSrcMap
 5. if (demoGoing) WaitForInputEvent(4) else WaitForInputEvent(15)
 6. // second DissBits pair also COMMENTED OUT at Banner.c:194-197
```

Because both dissolves are commented out, the banner is composed into `workSrcMap`, then step 4
immediately paints over it with the clean room from `backSrcMap`, and then the code waits 15
seconds. Result in 1.0.4: **the banner is never displayed**, and a new game begins with a 15-second
(4-second in demo) unexplained pause. The caller is `NewGame(kNewGameMode)`
(`GliderPRO/Sources/Play.c`), followed by `DumpScreenOn(&justRoomsRect)`.

A faithful-to-intent port must: compose the banner into the work buffer, dissolve it onto the
screen, wait for input (up to 15 s / 4 s), restore the room from the back buffer, and dissolve back.

`WaitForInputEvent` timeouts: **15 seconds** normally, **4 seconds** when `demoGoing`.

#### `DisplayStarsRemaining` — `GliderPRO/Sources/Banner.c:205-236`

Shown when the player collects a star, and on `NewGame(kResumeGameMode)`.

```
 1. SetPortWindowPort(mainWindow)
 2. QSetRect(&bounds, 0, 0, 256, 64)                       // the plaque PICT size
    CenterRectInRect(&bounds, &thisMac.screen)
    QOffsetRect(&bounds, -thisMac.screen.left, -thisMac.screen.top)
 3. src = bounds; InsetRect(&src, 64, 32)                  // computed but NEVER USED
 4. TextFont(applFont); TextFace(bold); TextSize(12)
    NumToString(numStarsRemaining, theStr)
 5. QOffsetRect(&bounds, 0, -20)                           // nudge 20 px up
 6. if (numStarsRemaining < 2)
        LoadScaledGraphic(kStarRemainingPICT /*1018*/, &bounds)      // singular art, no number
    else {
        LoadScaledGraphic(kStarsRemainingPICT /*1017*/, &bounds)
        MoveTo(bounds.left + 102 - (StringWidth(theStr) / 2), bounds.top + 23)
        ColorText(theStr, 4L)                              // CLUT index 4
    }
 7. DelayTicks(60)                                         // 1 second, input ignored
 8. if (WaitForInputEvent(30)) RestoreEntireGameScreen()    // up to 30 s; true == app resumed
 9. CopyRectWorkToMain(&bounds)
```

Notes:

* `numStarsRemaining < 2` uses the singular plaque — so **0 stars remaining also draws the
  singular "1 star" art**, with no number. That is the win condition, so it is immediately
  superseded, but a port should match the `< 2` test rather than `== 1`.
* The number is drawn centred on x = `bounds.left + 102` with baseline `bounds.top + 23`, in CLUT
  index **4**, only in the plural case.
* `src` (line 214-215) is dead.
* This one *does* copy `Work → Main` (step 9), so the plaque is actually visible — unlike the
  banner.

### 10.6 The About box

`GliderPRO/Sources/About.c`, 257 lines. This is the most hand-rolled dialog in the program: the
"Okay" button is not a control at all, it is a **PICT with a diagonal hit region**.

Constants (`GliderPRO/Sources/About.c:34-36, 97, 119`):

| Constant | Value | Meaning |
|---|---|---|
| `kAboutDialogID` | 150 | `DLOG`/`DITL` resource ID |
| `kTextItemVers` | 2 | item number of the version statText |
| `kPictItemMain` | 4 | item number of the main PICT |
| `kOkayButtPICTHiLit` | 151 | PICT drawn when the fake button is pressed |
| `kOkayButtPICTNotHiLit` | 150 | PICT drawn when it is not (note the swapped comments in the source) |
| `kOkayButton` | 1 | `Externs.h:22`; here it is a *disabled picture* item |

File-static state (`GliderPRO/Sources/About.c:23-25`):

```c
static RgnHandle  okayButtRgn;
static Rect       okayButtonBounds, mainPICTBounds;
static Boolean    okayButtIsHiLit, clickedDownInOkay;
```

**`DLOG` 150 `'About'`** — byte-verified at `GliderPRO/Glider PRO.r:6283`:
bounds `(62,62,242,446)` = **384 × 180**, `procID` 1, visible 1, goAwayFlag 1, `refCon` 0,
DITL 150. Full resource (24 bytes):
`003E 003E 00F2 01BE 0001 0100 0100 0000 0000 0096 0000 022F`. After the `itemsID` (`0096` = 150)
come four more bytes: `00` = the empty Pascal title, `00` = the pad byte that realigns to an even
offset, then the 2-byte extended-`DLOG` positioning word `0x022F`. Every other `DLOG` in the file is
21 bytes (header + empty title, no positioning word), so About 150 is the only one carrying it.
`0x022F` does not match any of the documented Rez positioning constants
(`noAutoCenter` 0x0000, `centerMainScreen` 0x280A, `alertPositionMainScreen` 0x300A, …), so its
meaning here is **unverified**. Note that `About.c:51` calls plain
`GetNewDialog(kAboutDialogID, nil, (WindowRef)-1L)` with no centring call of any kind, so the box
appears at the literal `(62,62)` global origin from the resource; a port can hard-code that or
ignore the trailing word and centre it.

**`DITL` 150** (`GliderPRO/Glider PRO.r:5172`), 9 items — **every item is disabled** (`0x80` bit
set), which is essential: the Dialog Manager must never report a hit, so the filter has total
control.

| Item | Type | Rect (t,l,b,r) | Size | Content |
|---|---|---|---|---|
| 1 | picture (disabled) | (113,317,176,380) | 63×63 | `PICT` 150 — the fake Okay button |
| 2 | statText (disabled) | (155,56,171,184) | 128×16 | *(empty)* — filled from `'vers'` 1 at runtime |
| 3 | icon (disabled) | (139,11,171,43) | 32×32 | `ICON` 150 |
| 4 | picture (disabled) | (5,6,105,378) | 372×100 | `PICT` **153** — the Glider PRO logo art |
| 5 | statText (disabled) | (139,56,155,184) | 128×16 | `by john calhoun` |
| 6 | statText (disabled) | (114,11,130,304) | 293×16 | `© 1994-2000 Casady & Greene, Inc.` |
| 7 | userItem (disabled) | (139,192,148,312) | 120×9 | drawn by `DrawDialogUserText2` — `Memory:   NNNNK` |
| 8 | userItem (disabled) | (148,192,157,312) | 120×9 | drawn by `DrawDialogUserText2` — `Screen:   WxHxD` |
| 9 | userItem (disabled) | (157,192,166,312) | 120×9 | **never drawn** — a third, unused text slot |

`kPictItemMain` is 4, so `mainPICTBounds` (`GliderPRO/Sources/About.c:75`) = `(5,6,105,378)`.
It is stored and **never used** — dead like `src` in `DisplayStarsRemaining`.

PICT dimensions confirmed from the resource fork: `PICT` 150 = 63 × 63 (632 bytes),
`PICT` 151 = 63 × 63 (632 bytes), `PICT` 153 = 372 × 100 (15194 bytes). `PICT` 152 does not exist.
`ICON` 150 is a standard 128-byte 32×32 1-bit icon.

#### `DoAbout` — `GliderPRO/Sources/About.c:32-89`

```
 1. aboutFilterUPP = NewModalFilterUPP(AboutFilter)
 2. wasResFile = CurResFile(); UseResFile(thisMac.thisResFile)
       // forces lookups to the APPLICATION's fork, so a house's resource fork
       // cannot shadow PICT 150/151/153 or 'vers' 1
 3. aboutDialog = GetNewDialog(kAboutDialogID /*150*/, nil, (WindowRef)-1L /*kPutInFront*/)
    // if (aboutDialog == nil) RedAlert(kErrDialogDidntLoad);   <- COMMENTED OUT at About.c:52-53
 4. version = (VersRecHndl)GetResource('vers', 1)
    if (version != nil) {
        messagePtr = (StringPtr)( (UInt32)&(**version).shortVersion[1]
                                  + (**version).shortVersion[0] );
        BlockMove((Ptr)messagePtr, &longVersion, ((UInt8)*messagePtr) + 1);
        SetDialogString(aboutDialog, kTextItemVers /*2*/, longVersion);
    }
 5. GetDialogItem(aboutDialog, kOkayButton /*1*/, &itemType, &itemHandle, &okayButtonBounds)
 6. okayButtRgn = NewRgn(); OpenRgn();
        MoveTo(okayButtonBounds.left + 1, okayButtonBounds.top + 45);
        Line( 44, -44);
        Line( 16,  16);
        Line(-44,  44);
        Line(-16, -16);
    CloseRgn(okayButtRgn)
 7. okayButtIsHiLit = false; clickedDownInOkay = false
 8. GetDialogItem(aboutDialog, kPictItemMain /*4*/, &itemType, &itemHandle, &mainPICTBounds)
 9. do { ModalDialog(aboutFilterUPP, &hit); } while ((hit != kOkayButton) && (okayButtRgn != nil))
10. if (okayButtRgn != nil) DisposeRgn(okayButtRgn)
    DisposeDialog(aboutDialog); DisposeModalFilterUPP(aboutFilterUPP)
11. UseResFile(wasResFile)
```

**The `'vers' 1` pointer arithmetic.** `VersRec` is
`{NumVersion numericVersion; short countryCode; Str255 shortVersion; Str255 reserved;}` — but the
two Pascal strings are *packed*, so `reserved` (the long version string) starts immediately after
`shortVersion`'s actual bytes, not at a fixed offset. Line 58-59 computes that address by hand:
`&shortVersion[1] + shortVersion[0]` = one past the last character of the short version = the
length byte of the long version.

Byte-verified `'vers' 1` from the resource fork:

```
01 12 80 00  00 00  05 '1' '.' '1' '.' '2'  31 'Glider PRO\xAA 1.1.2\r\xA9 1994-95 Casady & Greene, Inc.'
```

| Field | Bytes | Value |
|---|---|---|
| `numericVersion.majorRev` | `01` | 1 (BCD) |
| `numericVersion.minorAndBugRev` | `12` | minor 1, bug 2 (BCD nibbles) |
| `numericVersion.stage` | `80` | `finalStage` |
| `numericVersion.nonRelRev` | `00` | 0 |
| `countryCode` | `00 00` | 0 (verUS) |
| `shortVersion` | `05` + 5 bytes | `1.1.2` |
| `reserved` (long version) | `31` (=49) + 49 bytes | `Glider PRO™ 1.1.2\r© 1994-95 Casady & Greene, Inc.` |

0xAA is MacRoman `™`, 0xA9 is `©`. So the About box's item 2 is set to a **49-character string
containing an embedded CR**, placed in a statText item that is only 16 pixels tall — the Dialog
Manager wraps at the CR and only the first line, `Glider PRO™ 1.1.2`, is visible.

`'vers' 2` is `01 12 80 00 00 00 05 '1.1.2' 1F '© 1994-95 Casady & Greene, Inc.'` (0x1F = 31).

**Version discrepancy, three ways.** The resource fork says **1.1.2**, © 1994-95; `DITL` 150
item 6 says © **1994-2000**; and the source tree is labelled 1.0.4. The `Glider PRO.r` dump and the
`Sources` tree are therefore from different builds. A port should pick one and document it; do not
assume the shipped resource fork matches the shipped source.

**The diagonal Okay region.** The path traced by step 6, with
`(L, T) = (okayButtonBounds.left, okayButtonBounds.top) = (317, 113)`:

| Op | Ends at |
|---|---|
| `MoveTo(L+1, T+45)` | (318, 158) |
| `Line(44, -44)` | (362, 114) |
| `Line(16, 16)` | (378, 130) |
| `Line(-44, 44)` | (334, 174) |
| `Line(-16, -16)` | (318, 158) — closes |

That is a **parallelogram rotated 45°**: a diagonal band 16√2 ≈ 22.6 px wide running from lower-left
to upper-right across the 63×63 PICT. The art in `PICT` 150/151 is presumably a diagonally-oriented
"OK" tab, and this region is its hit-test. A Go port must reproduce the polygon
`(318,158) → (362,114) → (378,130) → (334,174) → close` in dialog-local coordinates, i.e. relative
to the item rect: `(+1,+45) → (+45,+1) → (+61,+17) → (+17,+61)`.

`HiLiteOkayButton` / `UnHiLiteOkayButton` (`GliderPRO/Sources/About.c:95-133`) each guard on the
`okayButtIsHiLit` flag, `GetPicture(151 or 150)`, `DrawPicture(thePict, &okayButtonBounds)`,
`ReleaseResource((Handle)thePict)`, then flip the flag. Note the `#define` comments are swapped:
`kOkayButtPICTHiLit 151` is commented `// res ID of unhilit button PICT` and vice versa. The code is
correct; the comments are not.

#### `UpdateMainPict` — `GliderPRO/Sources/About.c:138-163`

```
 1. DrawDialog(theDial)
 2. theStr = "Memory:   "
    PurgeSpace(&totalSize, &contigSize); totalSize /= 1024
    theStr += NumToString(totalSize) + "K"
    DrawDialogUserText2(theDial, 7, theStr)
 3. theStr = "Screen:   "
    theStr += NumToString(thisMac.screen.right - thisMac.screen.left) + "x"
    theStr += NumToString(thisMac.screen.bottom - thisMac.screen.top) + "x"
    theStr += NumToString(thisMac.isDepth)
    DrawDialogUserText2(theDial, 8, theStr)
```

Note the three spaces after each colon are literal (`"\pMemory:   "`, `"\pScreen:   "`). On a
640×480 8-bit screen item 8 reads `Screen:   640x480x8`. `contigSize` is fetched and discarded.
Item 9 (the third 120×9 userItem) is never written to.

`DrawDialogUserText2` is the left-aligned, no-background variant (see §4.5).

#### `AboutFilter` — `GliderPRO/Sources/About.c:168-256`

```
 0. if (Button() && clickedDownInOkay) {         // continuous tracking while held
        GetMouse(&mousePt)
        if (PtInRgn(mousePt, okayButtRgn)) HiLiteOkayButton() else UnHiLiteOkayButton()
    }
 1. switch (theEvent->what)
    case keyDown:
        switch (theEvent->message & charCodeMask)
            case kReturnKeyASCII (13):
            case kEnterKeyASCII (3):
                HiLiteOkayButton(); Delay(8, &dummyLong); UnHiLiteOkayButton()
                *hit = kOkayButton; handledIt = true; break
            default: handledIt = false
        break
    case mouseDown:
        mousePt = theEvent->where; GlobalToLocal(&mousePt)
        if (PtInRgn(mousePt, okayButtRgn)) { clickedDownInOkay = true; handledIt = false; }
        else                                 handledIt = false
        break
    case mouseUp:
        mousePt = theEvent->where; GlobalToLocal(&mousePt)
        if (PtInRgn(mousePt, okayButtRgn) && clickedDownInOkay) {
            UnHiLiteOkayButton(); *hit = kOkayButton; handledIt = true;
        } else { clickedDownInOkay = false; handledIt = false; }
        break
    case updateEvt:
        if ((WindowPtr)theEvent->message == mainWindow) {
            SetPort(mainWindow); BeginUpdate(...); UpdateMainWindow(); EndUpdate(...)
            SetPortDialogPort(theDial); handledIt = true
        } else if ((WindowPtr)theEvent->message == (WindowPtr)theDial) {
            SetPortDialogPort(theDial); BeginUpdate(...); UpdateMainPict(theDial); EndUpdate(...)
            handledIt = false
        }
        // NO else -> handledIt UNINITIALIZED
        break
    default: handledIt = false; break
 2. return handledIt
```

Behaviour notes:

* **No Escape, no ⌘., no Cancel.** Only Return, Enter, or a click in the diagonal region dismisses
  the About box. The dialog has `goAwayFlag = 1` in the `DLOG`, but a `dBoxProc`-family modal has
  no close box, and no `inGoAway` handling exists, so that flag is inert.
* **`handledIt` is uninitialized** when an `updateEvt` arrives for a window that is neither
  `mainWindow` nor the dialog (About.c:230-248) — e.g. the `menuWindow` menu-bar cover, or a
  desk-accessory window. The returned garbage may make `ModalDialog` believe the item in `*hit`
  (also uninitialized on that path) was selected. A port must initialize the flag to `false`.
* The `mouseDown` case sets `clickedDownInOkay` but returns `false`, so `ModalDialog` also
  processes the click. Because every DITL item is disabled, that is a no-op — this is why the
  all-disabled DITL matters.
* Step 0 uses `Button()` + `GetMouse()` (already dialog-local) rather than the event record, giving
  press-and-drag tracking with no explicit tracking loop. In Go, track this in the mouse-move
  handler while a button is down.
* `Delay(8, &dummyLong)` on Return/Enter = 8 ticks ≈ 133 ms of visible button flash.

Only caller: `GliderPRO/Sources/Menu.c:286` (Apple ▸ About Glider PRO…).
---

## 11. Error-reporting UI

Glider PRO has a three-tier error UI, colour-coded by the author:

| Tier | Function | Alert | Behaviour |
|---|---|---|---|
| **Red** | `RedAlert(short errorNumber)` | `ALRT` 170 | fatal — shows the alert then `ExitToShell()` |
| **Yellow** | `YellowAlert(short whichAlert, short identifier)` | `ALRT` 1006 | non-fatal — informs and continues |
| **File** | `CheckFileError(short resultCode, StringPtr fileName)` | `ALRT` 140 | maps an `OSErr` to a human string; returns `Boolean` so callers can branch |

All three call `InitCursor()` first (to un-spin the beachball/animated cursor) and all three have
their `CenterAlert(...)` call commented out, so they appear at their literal `ALRT` global bounds.

### 11.1 `RedAlert` — fatal errors

`GliderPRO/Sources/Utilities.c:135-161`:

```c
#define rDeathAlertID  170      // ALRT resource ID
#define rErrTitleID    170      // STR# ID for the title line
#define rErrMssgID     171      // STR# ID for the body text

InitCursor();
if (errorNumber > 1) {              // "<= 0 is unaccounted for"
    GetIndString(errTitle,   rErrTitleID, errorNumber);
    GetIndString(errMessage, rErrMssgID,  errorNumber);
} else {
    GetIndString(errTitle,   rErrTitleID, 1);
    GetIndString(errMessage, rErrMssgID,  1);
}
NumToString((long)errorNumber, errNumberString);
ParamText(errTitle, errMessage, errNumberString, "\p");
//  CenterAlert(rDeathAlertID);
dummyInt = Alert(rDeathAlertID, nil);
ExitToShell();
```

Note the guard is `errorNumber > 1`, not `>= 1`, so index 1 is reached by both branches — a
harmless redundancy. There is no upper-bound check: `RedAlert(99)` would call
`GetIndString(…, 170, 99)`, which the Resource Manager answers with an empty string.

**Error codes** (`GliderPRO/Headers/Externs.h:183-195`) with their two strings, byte-verified from
`STR#` 170 `'Errors'` (`GliderPRO/Glider PRO.r:5`) and `STR#` 171 `'Errors'`
(`GliderPRO/Glider PRO.r:27`):

| # | Constant | `STR#` 170 (title) | `STR#` 171 (body) |
|---|---|---|---|
| 1 | `kErrUnnaccounted` | `Unaccounted for Error` | `An error of unknown origin has occurred.  Call Casady & Greene's technical support.` |
| 2 | `kErrNoMemory` | `Out of Memory Error!` | `We ran out of memory.  You can give Glider PRO™ more memory using Get Info from the Finder.` |
| 3 | `kErrDialogDidntLoad` | `A Dialog Couldn't be Loaded` | `A dialog was unable to load.  Most likely, you need to increase the memory allocated for Glider PRO™.` |
| 4 | `kErrFailedResourceLoad` | `Couldn't Load a Resource` | `A required resource was unable to load.  Most likely, you need to increase the memory allocated for Glider PRO™.` |
| 5 | `kErrFailedGraphicLoad` | `Couldn't Load a Graphic` | `A graphic was unable to load.  Most likely, you need to increase the memory allocated for Glider PRO™.` |
| 6 | `kErrFailedOurDirect` | `Directory Look-up Failed` | `Failed attempting to determine our directory.  Call Casady & Greene's technical support.` |
| 7 | `kErrFailedValidation` | `Couldn't Validate` | `Without your original disk, you cannot complete the installation.` |
| 8 | `kErrNeedSystem7` | `Need System 7` | `Glider PRO™ requires System 7.0 (or more recent) to run.  You'll need a more current System version from Apple.` |
| 9 | `kErrFailedGetDevice` | `Graphics Device Look-up Failed` | `Who knows how this could have possibly happened.  Call Casady & Greene's technical support.` |
| 10 | `kErrFailedMemoryOperation` | `Memory Operation Failed` | `Likely memory is running low or is fragmented.  You can give Glider PRO™ more memory using Get Info from the Finder.` |
| 11 | `kErrFailedCatSearch` | `Failed House Search` | `The act of looking for house files failed.  Call Casady & Greene's technical support and report this error.` |
| 12 | `kErrNeedColorQD` | `Need Color Quickdraw` | `This Macintosh is incapable of color (or even grayscale).  Glider PRO™ requires a Mac with Color Quickdraw built in.` |
| 13 | `kErrNeed16Or256Colors` | `Need Color Monitor` | `Glider PRO runs only in 16 shades of gray or 256 colors.  None of the monitors hooked up to this Mac are capable of either of these modes.` |

**`ALRT` 170 `'Death Error'`** — byte-verified at `GliderPRO/Glider PRO.r:5191`:
bounds `(92,84,220,432)` = **348 × 128**, DITL 170, `stages` `0x5555`.

| Item | Type | Rect (t,l,b,r) | Size | Text |
|---|---|---|---|---|
| 1 | button | (92,274,112,332) | 58×20 | `Okay` |
| 2 | statText (disabled) | (8,10,24,257) | 247×16 | `^0` — the title |
| 3 | statText (disabled) | (28,10,96,257) | 247×68 | `^1` — the message |
| 4 | statText (disabled) | (100,10,116,257) | 247×16 | `(program error = ^2)` |
| 5 | icon (disabled) | (13,298,45,330) | 32×32 | `ICON` 170 |

`stages` `0x5555` = each nibble `5` = `0101`: default item 1, **one beep**, alert visible, for all
four stages.

Ten live `RedAlert` call sites exist across the program; the ones inside the UI subsystem are
`GliderPRO/Sources/Utilities.c:171` (`FindOurDevice` → `kErrFailedGetDevice`) and
`GliderPRO/Sources/StructuresInit2.c:277` (`theHousesSpecs` allocation → `kErrNoMemory`). The
`if (aboutDialog == nil) RedAlert(kErrDialogDidntLoad);` guard in `DoAbout` is **commented out**
(`GliderPRO/Sources/About.c:52-53`), and so are several others in `DialogUtils.c` (§4.1) — so a
missing DLOG produces a nil-dereference crash in 1.0.4 rather than a clean alert. **A Go port
should restore these guards.**

### 11.2 `CheckFileError` — File Manager errors

`GliderPRO/Sources/FileError.c:28-100`. Constants at `GliderPRO/Sources/FileError.c:14-15`:
`rFileErrorAlert 140`, `rFileErrorStrings 140`.

```
 1. if (resultCode == noErr) return true;         // "No problems?  Then cruise"
 2. stringIndex = switch(resultCode) { ...table below...; default: 1 }
 3. InitCursor()
 4. GetIndString(errMessage, 140, stringIndex)
    NumToString((long)resultCode, errNumString)
    ParamText(errMessage, errNumString, fileName, "\p")
 5. // CenterAlert(rFileErrorAlert);
    dummyInt = Alert(rFileErrorAlert /*140*/, 0L)
 6. return false
```

The return value is the API contract: **`true` means "no error, continue"**, `false` means "an
alert was shown". Callers use `if (!CheckFileError(theErr, "\pName")) return false;`.

The complete `OSErr` → `STR#` 140 index mapping, with the classic Mac OS error numbers and the
byte-verified strings from `STR#` 140 `'File Error'` (`GliderPRO/Glider PRO.r:465`):

| `OSErr` | Value | Index | String |
|---|---|---|---|
| *(any other)* | — | 1 | `A miscellaneous input/output error occurred (see error code below).` |
| `dirFulErr` | −33 | 2 | `An error occurred because the directory is full.` |
| `dskFulErr` | −34 | 3 | `An error occurred because the disk is full.  Try another disk.` |
| `ioErr` | −36 | 4 | `An unspecified input output error occurred.` |
| `bdNamErr` | −37 | 5 | `An error occurred because the name you chose is unacceptable.  Try a different name.` |
| `fnOpnErr` | −38 | 6 | `An error occurred because the file is not open.  Call tech support.` |
| `mFulErr` | −41 | 7 | `An error occurred because memory was too full.  Close windows, save, and bail out!` |
| `tmfoErr` | −42 | 8 | `An error occurred because there are too many files open (no more than 12 at once allowed).` |
| `wPrErr` | −44 | 9 | `An error occurred because the disk is write protected.  Save onto an unlocked disk.` |
| `fLckdErr` | −45 | 10 | `An error occurred because the file is locked.  Save under a different name.` |
| `vLckdErr` | −46 | 11 | `An error occurred because the volume you selected is locked.  Save onto an unlocked disk.` |
| `fBsyErr` | −47 | 12 | `An error occurred because the file is busy (perhaps another application has opened it).` |
| `dupFNErr` | −48 | 13 | `An error occurred because the name you chose has already been used.  Save with another name.` |
| `opWrErr` | −49 | 14 | `An error occurred because the file is already open for writing.  Call tech support.` |
| `volOffLinErr` | −53 | 15 | `An error occurred because that volume is off-line.  Save onto another disk.` |
| `permErr` | −54 | 16 | `An error occurred because of a permission violation.  Save onto an unlocked disk.` |
| `wrPermErr` | −61 | 17 | `An error occurred because write permission was denied.  Save onto an unlocked disk.` |

Note `fnfErr` (−43, file not found) is **not** in the table — it maps to index 1. That is
deliberate: several callers treat `fnfErr` as "create it" *before* calling `CheckFileError`
(e.g. `OpenHighScoresFile`, `LoadPrefs`).

The index-8 string's "(no more than 12 at once allowed)" is where the magic **12** in the prefs
`maxFiles` clamp comes from (§6.4) — the classic Mac per-application open-file limit.

**`ALRT` 140 `'File Error'`** — byte-verified at `GliderPRO/Glider PRO.r:5187`:
bounds `(92,60,220,446)` = **386 × 128**, DITL 140, `stages` `0x5555`.

| Item | Type | Rect (t,l,b,r) | Size | Text |
|---|---|---|---|---|
| 1 | button | (92,313,112,371) | 58×20 | `Okay` |
| 2 | statText (disabled) | (29,14,89,295) | 281×60 | `^0` — the message |
| 3 | statText (disabled) | (93,14,111,252) | 238×18 | `(error = ^1)` |
| 4 | icon (disabled) | (13,341,45,373) | 32×32 | `ICON` 140 |
| 5 | statText (disabled) | (6,14,23,332) | 318×17 | `A File Error Loading/Saving ^2` |

`^2` is the `fileName` argument. Note it is a **descriptive label**, not the actual file name: the
callers pass fixed literals such as `"\pPrefs Folder"`, `"\pHigh Scores File"`,
`"\pNew High Scores File"`, `"\pHigh Score"`, `"\pSaved Game"`, `"\pHigh Scores Folder"`. So the
alert reads e.g. `A File Error Loading/Saving High Scores File`.

Item 4's icon rect is `(13,341,45,373)`, whose right edge (373) **exceeds the alert's own width**
(386 − 60 = ... the alert is 386 wide, so local x runs 0..385 — 373 fits, but only just: the icon's
right edge is 12 px from the frame while every other alert leaves 20+). Cosmetic.

### 11.3 `YellowAlert` — non-fatal errors

`GliderPRO/Sources/HouseIO.c:640-655`:

```c
#define kYellowAlert  1006          // both the ALRT id and the STR# id
InitCursor();
GetIndString(errStr, kYellowAlert, whichAlert);
NumToString((long)identifier, errNumStr);
//  CenterAlert(kYellowAlert);
ParamText(errStr, errNumStr, "\p", "\p");
whoCares = Alert(kYellowAlert, nil);
```

The `identifier` second argument is free-form: sometimes an `OSErr`, sometimes a `MemError()`,
sometimes a call-site discriminator (`YellowAlert(kYellowUnaccounted, 2)` vs `…, 6`), sometimes a
delta (`savedGame->nRooms - thisHousePtr->nRooms`), sometimes a version constant.

**`ALRT` 1006 `'Yellow Alert'`** — byte-verified at `GliderPRO/Glider PRO.r:5215`:
bounds `(40,40,168,340)` = **300 × 128**, DITL 1006, `stages` `0x5555`.

| Item | Type | Rect (t,l,b,r) | Size | Text |
|---|---|---|---|---|
| 1 | button | (100,234,120,292) | 58×20 | `Okay` |
| 2 | statText (disabled) | (25,9,89,253) | 244×64 | `^0` — the message |
| 3 | statText (disabled) | (9,9,25,195) | 186×16 | `A problem came up:` |
| 4 | statText (disabled) | (102,9,118,148) | 139×16 | `Error #: ^1` |
| 5 | icon (disabled) | (9,260,41,292) | 32×32 | `ICON` 1006 |

**The 24 yellow-alert messages**, constants from `GliderPRO/Headers/GliderDefines.h:18-41`, strings
byte-verified from `STR#` 1006 `'Yellow Alerts'` (`GliderPRO/Glider PRO.r:118`):

| # | Constant | Message |
|---|---|---|
| 1 | `kYellowUnaccounted` | `A never-before-seen error has arisen.  Proceed with caution!  (Save and Quit immediately.)` |
| 2 | `kYellowFailedResOpen` | `I failed to open the house's resource fork.  Any unique room backgrounds are not accessible.` |
| 3 | `kYellowFailedResAdd` | `I failed to add a resource to the house's resource fork.  See error number.` |
| 4 | `kYellowFailedResCreate` | `I failed to create a new resource fork for the house.  See error number for problem.` |
| 5 | `kYellowNoHouses` | `There are no houses on this drive!  About your only option is to create your own new house with the Editor.` |
| 6 | `kYellowNewerVersion` | `This house is incompatible with us!  You'll need to upgrade Glider PRO to use this house.  Do not attempt to play/edit this house!` |
| 7 | `kYellowNoBackground` | `The background specified by this room was not found!  Try re-selecting a new background (the Room Info menu).` |
| 8 | `kYellowIllegalRoomNum` | `The room number is out of bounds.  I suspect the house file is corrupt.  Try deleting this "illegal" room though.` |
| 9 | `kYellowNoBoundsRes` | `The data is missing that specifies where the openings in this room are.  The house may be damaged.  Try selecting a new background though.` |
| 10 | `kYellowScrapError` | `There was a problem with the clipboard (Cut, Copy and Paste commands).  I couldn't guess why.` |
| 11 | `kYellowNoMemory` | `I think we just ran out of memory.  Quit now and give Glider PRO™ more memory.` |
| 12 | `kYellowFailedWrite` | `We failed to write the house to disk.  (That shouldn't have happened.)` |
| 13 | `kYellowNoMusic` | `Well, the music didn't load.  Glider PRO™ will still run, you'll just be musically challenged.` |
| 14 | `kYellowFailedSound` | `Wow, there was a problem bringing sounds up.  You might try giving Glider PRO™ more memory - otherwise ... silence.` |
| 15 | `kYellowAppleEventErr` | `Some kind of strange Apple Event error.  I think I would just ignore it.  Or call Casady & Greene with the error number.` |
| 16 | `kYellowOpenedOldHouse` | `Did you save the house on the same volume Glider PRO is on?  I saved the house but had to re-open the old house because I couldn't find the new one.` |
| 17 | `kYellowLostAllHouses` | `Wow, I couldn't find the old or new house.  Go to the Select House menu item and see if it's there.  If not, make sure they're on the same volume as Glider PRO.` |
| 18 | `kYellowFailedSaveGame` | `Couldn't create a saved game structure.  Memory is probably too low.` |
| 19 | `kYellowSavedTimeWrong` | `The saved game doesn't match the house.  Either this game was saved for a different house or the house was modified recently.` |
| 20 | `kYellowSavedVersWrong` | `This saved game is an old version.  We cannot use this game with this house.` |
| 21 | `kYellowSavedRoomsWrong` | `The number of rooms saved doesn't match the number of house rooms.  We cannot use this game with this house.` |
| 22 | `kYellowQTMovieNotLoaded` | `The QuickTime™ movie that goes with this house will not be used.  Glider PRO™ must have enough memory to easily load the entire movie into RAM.` |
| 23 | `kYellowNoRooms` | `This house has no rooms!  Do not attempt to play this house!  Select a new house to play.` |
| 24 | `kYellowCantOrderLinks` | `There was an error generating or parsing a links list.  Memory may be tight.` |

Indices 19-21 (the saved-game mismatch messages) are **unreachable** in 1.0.4 because they only
fire from inside the commented-out `OpenSavedGame` body (§8.6). Index 18 is reachable from the
commented-out `SaveGame2` only, so it is also dead.

### 11.4 `QuerySaveChanges` and `ALRT` 1002

`GliderPRO/Sources/HouseIO.c:596-632`, wrapped in `#ifndef COMPILEDEMO`. Constants at
`GliderPRO/Sources/HouseIO.c:20-22`: `kSaveChangesAlert 1002`, `kSaveChanges 1`,
`kDiscardChanges 2`.

```
 1. if (!fileDirty) return true;              // nothing to save -> silently proceed
 2. InitCursor()
 3. // CenterAlert(kSaveChangesAlert);
    ParamText(thisHouseName, "\p", "\p", "\p")
 4. hitWhat = Alert(kSaveChangesAlert /*1002*/, nil)
 5. if (hitWhat == kSaveChanges /*1*/) {
        if (wasHouseVersion < kHouseVersion /*0x0200*/) ConvertHouseVer1To2();
        wasHouseVersion = kHouseVersion;
        return WriteHouse(true);              // true == also write the resource fork
    }
 6. else if (hitWhat == kDiscardChanges /*2*/) {
        fileDirty = false;
        if (!quitting) { CloseHouse(); if (OpenHouse()) ReadHouse(); }   // reload from disk
        UpdateMenus(false);
        return true;
    }
 7. else return false;                        // Cancel: caller must abort whatever it was doing
```

Return-value contract: **`true` means "proceed", `false` means "the user cancelled"**.

**`ALRT` 1002 `'Save Changes'`** — byte-verified at `GliderPRO/Glider PRO.r:5203`:
bounds `(0,0,128,320)` = **320 × 128**, DITL 1002 (`'Save Changes?'`), `stages` `0x5555`.
**Its bounds are (0,0)-anchored** and centering is commented out, so it draws at the top-left of the
screen, partly under the menu bar.

| Item | Type | Rect (t,l,b,r) | Size | Text |
|---|---|---|---|---|
| 1 | button | (99,253,119,311) | 58×20 | `Save` — `kSaveChanges` |
| 2 | button | (71,253,91,311) | 58×20 | `Discard` — `kDiscardChanges` |
| 3 | button | (99,184,119,242) | 58×20 | `Cancel` |
| 4 | statText (disabled) | (6,7,80,244) | 237×74 | `You have made changes to ^0 and haven't saved these changes.  Save the house before proceeding?` |
| 5 | icon (disabled) | (7,278,39,310) | 32×32 | `ICON` 1000 |

`^0` is `thisHouseName`. Note the odd layout: Discard is *above* Save (rect tops 71 vs 99), with
Cancel to Save's left. `stages` `0x5555` → default item 1 (Save), one beep.

Four live call sites:

| Call site | Context |
|---|---|
| `GliderPRO/Sources/Menu.c:351` | Game ▸ Quit (`quitting = true; if (!QuerySaveChanges()) quitting = false;`) |
| `GliderPRO/Sources/Menu.c:382` | Options ▸ Editor, when leaving edit mode |
| `GliderPRO/Sources/HouseIO.c:544` | `CloseHouse` when `fileDirty` but not `gameDirty` |
| `GliderPRO/Sources/Events.c:405` | resume event, after `BitchAboutColorDepth()` returns "Quit" |

### 11.5 Environment alerts (`ALRT` 130, 180, 181, 1042)

These fire during startup and on suspend/resume, before or around the main window.

#### `ALRT` 130 `'Color Depth'` — `SwitchDepthOrAbort`

`GliderPRO/Sources/Environ.c:405-432`, constant `kSwitchDepthAlert 130`
(`GliderPRO/Sources/Environ.c:18`):

```c
if (thisMac.canSwitch) {
    InitCursor();
//  CenterAlert(kSwitchDepthAlert);
    usersDecision = Alert(kSwitchDepthAlert, nil);
    switch (usersDecision) {
        case 1: SwitchToDepth(8, true);  break;      // 256 colours
        case 2: SwitchToDepth(4, false); break;      // 16 greys
        case 3: ExitToShell();           break;      // Quit
    }
} else
    RedAlert(kErrUnnaccounted /*1*/);
```

Byte-verified: bounds `(0,0,96,274)` = **274 × 96**, DITL 130 (`'Switch Depth'`), `stages` `0x4444`.
**(0,0)-anchored** with centering commented out.

| Item | Type | Rect | Size | Text |
|---|---|---|---|---|
| 1 | button | (68,208,88,266) | 58×20 | `256` |
| 2 | button | (68,142,88,200) | 58×20 | `16` |
| 3 | button | (68,52,88,110) | 58×20 | `Quit` |
| 4 | icon (disabled) | (8,234,40,266) | 32×32 | `ICON` 130 |
| 5 | statText (disabled) | (8,8,56,222) | 214×48 | `Glider PRO™ requires 256 colors (or grays) or 16 grays.  Select the option you would like.` |

`SwitchToDepth(8, true)` = 8-bit colour; `SwitchToDepth(4, false)` = 4-bit greyscale. Related
constants: `kSwitchIfNeeded 0`, `kSwitchTo256Colors 1`, `kSwitchTo16Grays 2`
(`GliderPRO/Headers/GliderDefines.h:43-45`); `kPreferredDepth 8`
(`GliderPRO/Headers/Externs.h:15`).

A Go port renders in true colour and has no depth to switch, so this alert, `SwitchToDepth`,
`WhatsOurDepth`, `thisMac.canSwitch`, `thisMac.isDepth`, `thisMac.wasColorOrGray`, and the whole
`isDepthPref` / `kSwitchTo16Grays` preference path collapse to nothing. But note that the
**8-bit CLUT indices** used by `ColorText` (4, 5, 28), `kRedOrangeColor8` (23), and the room
graphics *do* still matter — they must be resolved against the game's `clut` resources.

#### `ALRT` 180 `'Set Memory'` / `ALRT` 181 `'Low Memory'`

`GliderPRO/Sources/Environ.c:678-692`, constants `kSetMemoryAlert 180`, `kLowMemoryAlert 181`
(`GliderPRO/Sources/Environ.c:19-20`):

```c
#ifdef COMPILEDEMO
//  CenterAlert(kLowMemoryAlert);
    NumToString((bytesNeeded + kPaddingBytes) / 1024L, sizeStr);
    ParamText(sizeStr, "\p", "\p", "\p");
    hitWhat = Alert(kLowMemoryAlert, nil);
#else
//  CenterAlert(kSetMemoryAlert);
    NumToString((bytesNeeded + kPaddingBytes) / 1024L, sizeStr);
    ParamText(sizeStr, "\p", "\p", "\p");
    hitWhat = Alert(kSetMemoryAlert, nil);
//  SetAppMemorySize(bytesNeeded + kPaddingBytes);      <- COMMENTED OUT
#endif
ExitToShell();
```

Since `COMPILEDEMO` is commented out (`GliderPRO/Headers/GliderDefines.h:12`), the shipped build
uses **180**; 181 is dead. `SetAppMemorySize` (which would rewrite the app's own `'SIZE'` resource,
`GliderPRO/Sources/Environ.c:699+`) is commented out at the call site, so the alert only tells the
user to do it manually.

`ALRT` 180: bounds `(40,40,156,334)` = 294 × 116, DITL 180, `stages` `0x5555`.

| Item | Type | Rect | Size | Text |
|---|---|---|---|---|
| 1 | button | (88,228,108,286) | 58×20 | `Quit` |
| 2 | statText (disabled) | (8,8,72,246) | 238×64 | `Glider PRO™ requires more memory.  Get Info on Glider PRO™ in the Finder to increase its memory size or try quitting other applications.` |
| 3 | icon (disabled) | (8,254,40,286) | 32×32 | `ICON` 180 |
| 4 | statText (disabled) | (76,8,92,215) | 207×16 | `(We need about: ^0K)` |

`ALRT` 181: bounds `(40,40,168,328)` = 288 × 128, DITL 181, `stages` `0x5555`, 2 items:
button 1 `Okay` at (100,222,120,280), and statText 2 at (8,8,92,280):
`Glider PRO™ Demo requires more memory to run on this Mac.  For this Mac, try giving the game ^0K.
If the game is on a locked volume, copy it to your hard drive.`

#### `ALRT` 1042 `'Color Switched'` — `BitchAboutColorDepth`

`GliderPRO/Sources/Events.c:46-55`, constant `kColorSwitchedAlert 1042`
(`GliderPRO/Sources/Events.c:48`):

```c
short BitchAboutColorDepth (void)
{
//  CenterAlert(kColorSwitchedAlert);
    sheSaid = Alert(kColorSwitchedAlert, nil);
    return (sheSaid);
}
```

Note: **no `InitCursor()`**, unlike every other alert wrapper.

Called from the resume-event handler (`GliderPRO/Sources/Events.c:396-415`):

```c
if (theEvent->message & 0x00000001)              // resume event
    if (WhatsOurDepth() != thisMac.isDepth) {
        buttonHit = BitchAboutColorDepth();
        if (buttonHit == 1) {                    // player wants to Quit
            if (QuerySaveChanges()) quitting = true;
        } else
            SwitchToDepth(thisMac.isDepth, thisMac.wasColorOrGray);
    }
```

`ALRT` 1042: bounds `(40,40,148,296)` = 256 × 108, DITL 1042 (`'Monitors Switched'`),
`stages` `0x5555`.

| Item | Type | Rect | Size | Text |
|---|---|---|---|---|
| 1 | button | (80,190,100,248) | 58×20 | `Quit` |
| 2 | button | (80,124,100,182) | 58×20 | `Restore` |
| 3 | statText (disabled) | (8,8,72,208) | 200×64 | `You switched the number of colors!  You must either Quit Glider PRO™ or Restore your monitor to its original colors.` |
| 4 | icon (disabled) | (8,216,40,248) | 32×32 | `ICON` 130 (shared with alert 130) |

Note `kSavingGameDial 1042` is also `#define`d in `GliderPRO/Sources/Input.c:19` — an **ID
collision** with this alert. `kSavingGameDial` is never used (the saving indicator became the
scoreboard's `kSavingTitleMode`), so no harm, but a port must not treat 1042 as a dialog.

### 11.6 Sound / Sound Manager alerts (`ALRT` 1030, 1039)

#### `ALRT` 1039 `'Sound Mem'` — `TellHerNoSounds`

`GliderPRO/Sources/Sound.c:514-523`, constant `kNoMemForSoundsAlert 1039`:

```c
void TellHerNoSounds (void)
{
    // CenterAlert(kNoMemForSoundsAlert);
    hitWhat = Alert(kNoMemForSoundsAlert, nil);
}
```

Called from `GliderPRO/Sources/Environ.c:672` when there is enough memory to run only if both music
and sounds are skipped; the caller then sets `dontLoadMusic = true; dontLoadSounds = true;`.

Bounds `(54,96,162,370)` = 274 × 108, DITL 1039, `stages` `0x4444`.

| Item | Type | Rect | Size | Text |
|---|---|---|---|---|
| 1 | button | (81,186,101,266) | 80×20 | `Whatever` |
| 2 | statText (disabled) | (8,8,72,233) | 225×64 | `Okay, with your monitor & color depth, there isn't enough memory.  To run Glider PRO™, the music & sounds won't be loaded.` |
| 3 | icon (disabled) | (8,234,40,266) | 32×32 | `ICON` 1011 |

#### `ALRT` 1030 `'Sound Manager'` — `BitchAboutSM3` (DEAD)

`GliderPRO/Sources/Sound.c:527-535`, constant `kNoSoundManager3Alert 1030`:

```c
void BitchAboutSM3 (void)
{
    // CenterAlert(kNoSoundManager3Alert);
    hitWhat = Alert(kNoSoundManager3Alert, nil);
}
```

Its only call site is **commented out** at `GliderPRO/Sources/Main.c:357-361`, so this alert never
appears in 1.0.4. Bounds `(40,40,148,314)` = 274 × 108, DITL 1030, `stages` `0x4444`, 3 items:
button 1 `Okay` at (80,208,100,266); statText 2 at (8,8,72,225):
`Where is Sound Manager 3.0?  I highly recommend you install it.  It's fast, it's unobtrusive, and
it's the law!`; icon 3 = `ICON` 1070.

### 11.7 Apple Event alerts

#### `ALRT` 1031 `'No Printing'` — `DoPrintDocAE`

`GliderPRO/Sources/AppleEvents.c:132-142`, constant `kNoPrintingAlert 1031`
(`GliderPRO/Sources/AppleEvents.c:14`):

```c
pascal OSErr DoPrintDocAE (const AppleEvent *theAE, AppleEvent *reply, UInt32 ref)
{
#pragma unused (theAE, reply, ref)
//  CenterAlert(kNoPrintingAlert);
    hitWhat = Alert(kNoPrintingAlert, nil);
    return errAEEventNotHandled;
}
```

Reachable by dropping a house file onto the app's icon while holding the Print modifier, or by an
AppleScript `print`. Bounds `(40,40,112,308)` = 268 × 72, DITL 1031, `stages` `0x5555`.

| Item | Type | Rect | Size | Text |
|---|---|---|---|---|
| 1 | button | (48,202,68,260) | 58×20 | `Bye!` |
| 2 | statText (disabled) | (8,8,40,220) | 212×32 | `Glider PRO doesn't know what the word "Print" means.  Sorry.` |
| 3 | icon (disabled) | (8,228,40,260) | 32×32 | `ICON` 900 |

Note this is the only user-visible string that says `Glider PRO` without the `™`.

Apple Event failures elsewhere go through `YellowAlert(kYellowAppleEventErr /*15*/, …)`.

### 11.8 `ALRT` 1037 `'Not In Demo'` — `DoNotInDemo` (DEAD in the retail build)

`GliderPRO/Sources/Menu.c:766-775`, wrapped in `#ifdef COMPILEDEMO`:

```c
#ifdef COMPILEDEMO
void DoNotInDemo (void)
{
    #define kNotInDemoAlert  1037
//  CenterAlert(kNotInDemoAlert);
    whoCares = Alert(kNotInDemoAlert, nil);
}
#endif
```

Its two call sites are also `#ifdef COMPILEDEMO` (`GliderPRO/Sources/Menu.c:329` for
Game ▸ Select House…, `GliderPRO/Sources/Menu.c:376` for Options ▸ Editor). Since `COMPILEDEMO` is
commented out at `GliderPRO/Headers/GliderDefines.h:12`, none of this compiles into the shipped
build.

Bounds `(40,40,164,326)` = 286 × 124, DITL 1037 (`'Demo'`), `stages` `0x4444`, 2 items:
button 1 `Okay` at (96,220,116,278); statText 2 at (8,8,88,278):
`This feature is not in the Demo version of Glider PRO™.  Glider PRO™ is published by Casady &
Greene, Inc.  Call them at (408) 484-9228 during normal business hours (Pacific time, USA).`

### 11.9 `ALRT` 160 `'Prefs'` — new-preferences-file notice

`GliderPRO/Sources/Prefs.c:270-278` (`BringUpDeletePrefsAlert`), fired when `LoadPrefs` finds a
prefs file whose `prefVersion != kPrefsVersion` (0x0034). See §6.3.

Bounds `(40,40,140,340)` = 300 × 100, DITL 160, `stages` `0x5555`, 3 items:
button 1 `Okay` at (69,230,89,288); statText 2 at (13,14,61,231):
`You have a new Preferences file.  All preferences have been set to their defaults.`;
icon 3 = `ICON` 160 at (13,259,45,291).

Note the icon's rect is `(13,259,45,291)` while the alert is 300 wide — it fits, but the icon is
32×32 and the alert's right edge is at local x 299, so there are 8 px of margin.

### 11.10 `ALRT` 1009 `'Banner'` — DEAD

`kHouseBannerAlert 1009` is `#define`d at `GliderPRO/Sources/Play.c:18` and **never used**. The
resource still ships: bounds `(0,0,150,346)` = 346 × 150, DITL 1009, `stages` `0x4444`.

| Item | Type | Rect | Size | Text |
|---|---|---|---|---|
| 1 | button | (122,280,142,338) | 58×20 | `Begin` |
| 2 | statText (disabled) | (8,8,24,298) | 290×16 | `^0` |
| 3 | statText (disabled) | (33,8,113,298) | 290×80 | `^1` |
| 4 | icon (disabled) | (8,306,40,338) | 32×32 | `ICON` 1000 |

This is clearly the *predecessor* of the graphical notebook-paper banner in `Banner.c` (§10.5): a
plain alert with a title line (`^0`, presumably the house name) and an 80-pixel body (`^1`, the
house's banner text) and a `Begin` button. A port that cannot reproduce the notebook-paper art
could fall back to this.

### 11.11 Summary: alert `stages` decoding

`ALRT` resources carry a 16-bit `stages` word, four nibbles, one per alert stage (1st through 4th
occurrence). Each nibble is `b3 b2 b1b0`:

| Bits | Meaning |
|---|---|
| bit 3 (`8`) | 0 = default item is **1**, 1 = default item is **2** |
| bit 2 (`4`) | 1 = draw the alert (0 = silent, no alert shown) |
| bits 1-0 | number of beeps (0-3) |

Observed values in Glider PRO:

| `stages` | Nibble | Decoded | Alerts using it |
|---|---|---|---|
| `0x4444` | `4` = `0100` | default item 1, visible, **0 beeps** | 130, 1009, 1030, 1032, 1036, 1037, 1038, 1039, 1040, 1046 |
| `0x5555` | `5` = `0101` | default item 1, visible, **1 beep** | 140, 160, 170, 180, 181, 1002, 1004, 1005, 1006, 1008, 1028, 1031, 1042, 1044 |
| `0xCCCC` | `C` = `1100` | default item **2**, visible, 0 beeps | 1041 |
| `0xFFFF` | `F` = `1111` | default item **2**, visible, **3 beeps** | 1029 |

(Re-derived by parsing bytes 10-11 of all 26 `ALRT` resources in `Glider PRO.r`; the four rows above
account for every one of them.)

Two alerts default to **item 2** rather than item 1: 1041 (Save Game? → `Don't Save`) and 1029 (Lock
House → `Don't Lock`) — deliberate safe defaults. `ALRT` 1029 is also the only alert
that beeps **three** times; everything else beeps once or not at all. Every alert in the program uses
the **same nibble for all four stages**, i.e. none of them escalate, so the nibble ordering within
the word (stage 1 first vs. stage 4 first) is unobservable here and a port can ignore it.

A Go port should render `stages` as: which button responds to Return/Enter, and whether to play the
system beep on presentation.
---

## 12. String resources

Glider PRO ships exactly **10 `STR#` resources** (string-list resources). Every user-visible string
that is not baked into a `DITL` lives in one of them. All ten are dumped in full below, byte-decoded
from the resource fork and transcoded from MacRoman.

`STR#` on-disk layout: `short count` followed by `count` back-to-back Pascal strings (1 length byte
+ N data bytes, **no padding, no alignment**). `GetIndString(dest, id, index)` is **1-based** and
returns an empty string for out-of-range indices without signalling an error.

| ID | Rez name | Count | `.r` line | Consumer |
|---|---|---|---|---|
| 128 | `Jinjur` | 1 | 114 | **none — dead** |
| 129 | `Prefmain` | 4 | 381 | `UpdateSettingsMain`, `GliderPRO/Sources/Settings.c:1288-1294` |
| 140 | `File Error` | 17 | 465 | `CheckFileError`, `GliderPRO/Sources/FileError.c:92` |
| 150 | `Localized Strings` | 51 | 396 | `GetLocalizedString`, `GliderPRO/Sources/StringUtils.c:325` |
| 160 | `Prefs` | 1 | 1 | `GetPrefsFPath` fallback folder name, `GliderPRO/Sources/Prefs.c:79` |
| 170 | `Errors` | 13 | 5 | `RedAlert` titles, `GliderPRO/Sources/Utilities.c:147, 152` |
| 171 | `Errors` | 13 | 27 | `RedAlert` bodies, `GliderPRO/Sources/Utilities.c:148, 153` |
| 1005 | `Months` | 12 | 387 | `ObjectDraw2.c:1161` (calendar object) |
| 1006 | `Yellow Alerts` | 24 | 118 | `YellowAlert`, `GliderPRO/Sources/HouseIO.c:648` |
| 1007 | `Object Names` | 144 | 279 | `Tools.c:146`, `Scrap.c:80`, `ObjectInfo.c` ×13 (lines 945, 1135, 1184, 1280, 1403, 1558, 1645, 1760, 1884, 1958, 2071, 2195, 2304) |

### 12.1 `STR#` 128 `'Jinjur'` — dead

| # | String |
|---|---|
| 1 | `Crystal` |

No `GetIndString(…, 128, …)` call exists anywhere in the source. `Jinjur` and `Crystal` are both
*Oz* references (as is the creator code `'ozm5'` = Ozma, and the object name `Ozma` at
`STR#` 1007 index 129). Leftover or Easter-egg scaffolding. A port can ignore it.

### 12.2 `STR#` 129 `'Prefmain'` — Preferences pane names

| # | String | Used for |
|---|---|---|
| 1 | `Display` | `DrawDialogUserText2(theDialog, kDisplayNameItem, theStr)` |
| 2 | `Sounds` | `kSoundNameItem` |
| 3 | `Controls` | `kControlNameItem` |
| 4 | `Brains` | `kBrainsNameItem` |

Drawn by `UpdateSettingsMain` (`GliderPRO/Sources/Settings.c:1280-1303`) as labels *under* the four
icon buttons in `DLOG` 1012. Note the plural mismatch: `Sounds` and `Controls` are plural,
`Display` and `Brains` are not.

### 12.3 `STR#` 140 `'File Error'` — File Manager error messages

Dumped in full with the `OSErr` mapping in §11.2.

### 12.4 `STR#` 150 `'Localized Strings'` — the general string pool

`GetLocalizedString(short index, StringPtr theString)`
(`GliderPRO/Sources/StringUtils.c:320-326`):

```c
#define kLocalizedStringsID  150
GetIndString(theString, kLocalizedStringsID, index);
```

All 51 strings:

| # | String | Used by |
|---|---|---|
| 1 | `There are ` | `DrawBannerMessage` (plural star count) |
| 2 | `There is ` | `DrawBannerMessage` (singular) |
| 3 | ` stars in the house.` | `DrawBannerMessage` |
| 4 | ` star in the house.` | `DrawBannerMessage` |
| 5 | `Get every star to win.` | `DrawBannerMessage` |
| 6 | `room` | `DrawHighScores` (levels == 1) |
| 7 | `rooms` | `DrawHighScores` |
| 8 | `Click Mouse or Hit a Key to Exit` | `DrawHighScores` footer |
| 9 | `Name for New House:` | new-house naming (`House.c`) |
| 10 | `Untitled House` | default new-house name |
| 11 | `Enter New-Game Message here (max. 255 characters)` | `InitializeEmptyHouse` → `houseType.banner` |
| 12 | `Enter Finished-House Message here (max. 255 characters)` | `InitializeEmptyHouse` → `houseType.trailer` |
| 13 | `Converting 1.0 House to 2.0` | `ConvertHouseVer1To2` progress |
| 14 | `Converting Room ` | ditto |
| 15 | `Save copy of house as:` | duplicate-house prompt |
| 16 | `Failed House Shrinkage` | house-integrity check |
| 17 | ` Floor Number Bad` | `HouseLegal.c` |
| 18 | ` Suite Number Bad` | `HouseLegal.c` |
| 19 | `Object Bad` | `HouseLegal.c` |
| 20 | `No room upstairs!` | editor link validation |
| 21 | `No downstairs to match!` | editor link validation |
| 22 | `No room downstairs!` | editor link validation |
| 23 | `No upstairs to match!` | editor link validation |
| 24 | `Checking House File` | `HouseLegal.c` progress title |
| 25 | `Checking House…` | (0xC9 = `…`) |
| 26 | `Checking Rooms…` | |
| 27 | `Room Number Errors` | |
| 28 | ` Duplicate Floor/Suites` | |
| 29 | ` Room Errors` | |
| 30 | ` 'Untitled' Rooms` | |
| 31 | ` Room Names Too Long` | |
| 32 | ` Room's # of Objects Wrong` | |
| 33 | `Checking Objects…` | |
| 34 | ` Object Errors` | |
| 35 | `You have no stars in the house!` | |
| 36 | `Cut Object` | Edit-menu item rename (`UpdateClipboardMenus`) |
| 37 | `Copy Object` | ditto |
| 38 | `Clear Object` | ditto |
| 39 | `Cut Room` | ditto |
| 40 | `Copy Room` | ditto |
| 41 | `Clear Room` | ditto |
| 42 | `Paste Room` | ditto |
| 43 | `Paste Object` | ditto |
| 44 | `Nothing To Paste` | **the permanent Paste item text** — see §3.5.5 |
| 45 | `Object Pair Added` | editor feedback |
| 46 | `Exterior Door Added in Next Room` | editor feedback |
| 47 | `Interior Door Added in Next Room` | editor feedback |
| 48 | `Ext. Window Added in Next Room` | editor feedback |
| 49 | `Int. Window Added in Next Room` | editor feedback |
| 50 | `Down Stairs Added in Room Above` | editor feedback |
| 51 | `Up Stairs Added in Room Below` | editor feedback |

Strings 25, 26, 33 contain MacRoman byte **0xC9** = `…` (horizontal ellipsis), not three periods.

### 12.5 `STR#` 160 `'Prefs'` — the fallback Preferences folder name

| # | String |
|---|---|
| 1 | `Preferences` |

Constants (`GliderPRO/Sources/Prefs.c:22-24`): `kPrefsStringsID 160`, `kPrefsFNameIndex 1`, and
`kDefaultPrefFName "\pPreferences"` (the same literal, hard-coded as a backup of the backup).

Used only at `GliderPRO/Sources/Prefs.c:79`, inside `GetPrefsFPath`, on the path where
`CanUseFindFolder()` returns false (no Gestalt, or `gestaltFindFolderAttr` reports the Folder
Manager absent). In that case the code looks for a directory literally named `Preferences` in the
System Folder rather than calling `FindFolder(kOnSystemDisk, kPreferencesFolderType, …)`.

`CanUseFindFolder` (`GliderPRO/Sources/Prefs.c:39-55`):

```c
if (!thisMac.hasGestalt) return false;
theErr = Gestalt(gestaltFindFolderAttr, &theFeature);
if (theErr != noErr) return false;
if (!BitTst(&theFeature, 31 - gestaltFindFolderPresent)) return false;
else return true;
```

The `31 - gestaltFindFolderPresent` idiom converts a Gestalt bit *number* to a `BitTst` bit
*offset* (`BitTst` counts from the most-significant bit of the byte at the given address). A Go port
does not need any of this; it has one preferences path.

### 12.6 `STR#` 170 and 171 `'Errors'` — `RedAlert` titles and bodies

Dumped in full in §11.1. Both lists have exactly 13 entries and are indexed by the same
`kErr*` constant, so they must always be kept in lockstep.

### 12.7 `STR#` 1005 `'Months'`

| # | String | | # | String |
|---|---|---|---|---|
| 1 | `JANUARY` | | 7 | `JULY` |
| 2 | `FEBRUARY` | | 8 | `AUGUST` |
| 3 | `MARCH` | | 9 | `SEPTEMBER` |
| 4 | `APRIL` | | 10 | `OCTOBER` |
| 5 | `MAY` | | 11 | `NOVEMBER` |
| 6 | `JUNE` | | 12 | `DECEMBER` |

All upper case. Used only at `GliderPRO/Sources/ObjectDraw2.c:1161`
(`GetIndString(monthStr, kMonthStringID, timeRec.month)`) to draw the **Calendar** room object
(`STR#` 1007 index 136) with the real current month. `timeRec.month` is 1-based, matching the
1-based `GetIndString`.

### 12.8 `STR#` 1006 `'Yellow Alerts'`

All 24 dumped in full in §11.3, indexed by `kYellow*` (1-24).

### 12.9 `STR#` 1007 `'Object Names'`

144 entries indexed **directly by `objectType.what`** (`kObjectNameStrings 1007`,
`GliderPRO/Headers/GliderDefines.h:461`). Because the object codes are grouped into 16-wide
classes, the unused slots inside each class carry placeholder single-character names (`8`, `9`,
`a`…`f`) rather than being blank — which is how you can tell exactly which codes are legal.

| # | Name | | # | Name | | # | Name |
|---|---|---|---|---|---|---|---|
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
| 15 | `Sewer Blower` | | 63 | `Invisible Transport` | | 111 | `e` *(unused)* |
| 16 | `Lift Area` | | 64 | `Deluxe Transport` | | 112 | `f` *(unused)* |
| 17 | `Table` | | 65 | `Light Switch` | | 113 | `Balloon` |
| 18 | `Shelf` | | 66 | `Machine Switch` | | 114 | `'Copter (lf. drift)` |
| 19 | `Cabinet` | | 67 | `Thermostat` | | 115 | `'Copter (rt. drift)` |
| 20 | `Filing Cabinet` | | 68 | `Digital Switch` | | 116 | `Dart (lf. moving)` |
| 21 | `Wastebasket` | | 69 | `Knife Switch` | | 117 | `Dart (rt. moving)` |
| 22 | `Milk Crate` | | 70 | `Invisible Switch` | | 118 | `Bouncing Ball` |
| 23 | `Counter` | | 71 | `Trigger` | | 119 | `Water Drip` |
| 24 | `Dresser` | | 72 | `Large Trigger` | | 120 | `Fish Bowl & Fish` |
| 25 | `Deck Table` | | 73 | `Sound Trigger` | | 121 | `Cobweb` |
| 26 | `Bar Stool` | | 74 | `9` *(unused)* | | 122 | `9` *(unused)* |
| 27 | `Steamer Trunk` | | 75 | `a` *(unused)* | | 123 | `a` *(unused)* |
| 28 | `Invisible Obstacle` | | 76 | `b` *(unused)* | | 124 | `b` *(unused)* |
| 29 | `Manhole` | | 77 | `c` *(unused)* | | 125 | `c` *(unused)* |
| 30 | `Books` | | 78 | `d` *(unused)* | | 126 | `d` *(unused)* |
| 31 | `Invisible Rebounder` | | 79 | `e` *(unused)* | | 127 | `e` *(unused)* |
| 32 | `f` *(unused)* | | 80 | `f` *(unused)* | | 128 | `f` *(unused)* |
| 33 | `Digital Clock` | | 81 | `Ceiling Light` | | 129 | `Ozma` |
| 34 | `Wall Clock` | | 82 | `Simple Bulb` | | 130 | `Mirror` |
| 35 | `Alarm Clock` | | 83 | `Table Lamp` | | 131 | `Mouse Hole` |
| 36 | `Cuckoo Clock` | | 84 | `Hip Pole Lamp` | | 132 | `Fireplace` |
| 37 | `Extra Glider` | | 85 | `Deco Lamp` | | 133 | `Flower` |
| 38 | `Battery` | | 86 | `Flourescent Light` | | 134 | `Window (closed)` |
| 39 | `Rubber Bands (8)` | | 87 | `Track Lighting` | | 135 | `Teddy Bear` |
| 40 | `Grease (spills rt.)` | | 88 | `Invisible Light` | | 136 | `Calendar` |
| 41 | `Grease (spills lf.)` | | 89 | `8` *(unused)* | | 137 | `Broad Vase` |
| 42 | `Aluminum Foil` | | 90 | `9` *(unused)* | | 138 | `Narrow Vase` |
| 43 | `Invisible Bonus` | | 91 | `a` *(unused)* | | 139 | `Bulletin Board` |
| 44 | `Magic Star` | | 92 | `b` *(unused)* | | 140 | `Cloud` |
| 45 | `Sparkle` | | 93 | `c` *(unused)* | | 141 | `Faucet` |
| 46 | `Helium (He)` | | 94 | `d` *(unused)* | | 142 | `Throw Rug` |
| 47 | `Slide Rect` | | 95 | `e` *(unused)* | | 143 | `Wind Chimes` |
| 48 | `f` *(unused)* | | 96 | `f` *(unused)* | | 144 | `Mermaid` |

Sanity checks against the object-code constants: `kStar` = `0x2C` = 44 → index 44 = `Magic Star` ✓.
`Flourescent` (index 86) is misspelled in the shipped resource — a faithful port keeps the typo.
Index 129 `Ozma` matches the creator code `'ozm5'`.

Note this list is indexed 1-based by `what`, so `what == 0` has no name; `kRoomIsEmpty` is −1 and
`what` values above 144 are out of range and return an empty string from `GetIndString`.

---

## 13. Dead and unreachable UI — complete inventory

Everything below exists in the source and/or the resource fork but cannot be reached by a user of
the shipped 1.0.4 build. A Go port must decide, item by item, whether to reproduce the dead state
(bug-for-bug fidelity) or to revive the intent. Reviving is usually better UX and always more work.

### 13.1 Compile-time-disabled

| Symbol | Where | Effect |
|---|---|---|
| `COMPILEDEMO` | commented out at `GliderPRO/Headers/GliderDefines.h:12` | `DoNotInDemo`/`ALRT` 1037 and `ALRT` 181 never compile; `QuerySaveChanges` and `HowToZeroScores` **do** compile (they are `#ifndef COMPILEDEMO`) |
| `CREATEDEMODATA` | commented out at `GliderPRO/Headers/GliderDefines.h:11` | demo-recording code excluded |
| `CAREFULDEBUG` | commented out at `GliderPRO/Headers/GliderDefines.h:13` | debug assertions excluded |
| `COMPILENOCP` | **defined**, no value, `GliderPRO/Headers/GliderDefines.h:14` | "no copy protection" — `DLOG`/`DITL` 1026 (Validation) unreachable, `kErrFailedValidation` (7) unreachable |
| `COMPILEQT` | **defined**, no value, `GliderPRO/Headers/GliderDefines.h:15` | QuickTime support compiled in |
| `BUILD_ARCADE_VERSION` | `1` at `GliderPRO/Headers/GliderDefines.h:16` | arrow keys in splash/edit mode trigger High Scores / Demo / New Game, bypassing menu enable rules (§2.6) |

### 13.2 Commented-out functions and call sites

| Item | Location | What is lost |
|---|---|---|
| `CenterDialog` / `CenterAlert` | every call site, e.g. `GliderPRO/Sources/DialogUtils.c` and 20+ `// CenterAlert(...)` lines | **all dialogs and alerts appear at their literal `DLOG`/`ALRT` global bounds.** `DLOG` 1002, 1009, 130, 1020 have `(0,0)` origins and land under the menu bar |
| `ZoomBetweenWindows` and friends | `GliderPRO/Sources/DialogUtils.c` §4.1 | the `doZooms` preference (`wasZooms`, offset **208**) is stored, toggled, and **read by nothing** |
| eight `DialogUtils.c` helpers | §4.1 | including the `RedAlert(kErrDialogDidntLoad)` nil-checks |
| `DissBits` / `DissBitsChunky` | `GliderPRO/Sources/HighScores.c:68-71, 76-79`; `GliderPRO/Sources/Banner.c:178-181, 194-197`; `GliderPRO/Sources/MainWindow.c:109-112`; four sites in `GliderPRO/Sources/Play.c` | **the `quickerTransitions` preference (`wasQuickTrans`, offset 215) does nothing**, the banner is never seen (§10.5), the high-score screen and `RedrawSplashScreen` mis-composite |
| `SaveGame2` body | `GliderPRO/Sources/SavedGames.c:33-147` | ⌘S and ⌘Q-then-Save are no-ops |
| `OpenSavedGame` body | `GliderPRO/Sources/SavedGames.c:170-295` | Game ▸ Open Saved Game… always fails |
| `SaveGame(false)` | `GliderPRO/Sources/Menu.c:458` | stale in-house saves are never cleared on New House |
| `SaveHouseAs` body | `/*` at `GliderPRO/Sources/HouseIO.c:245` (marked `// TEMP - fix this later -- use NavServices`), sole caller commented out at `GliderPRO/Sources/Menu.c:470-471`, `iSaveAs` enable/disable commented out at `GliderPRO/Sources/Menu.c:107, 112, 150` | House ▸ Save As is inert: the `StandardPutFile`-era duplicate-house path (localized string 15) never runs |
| `BitchAboutSM3()` | `GliderPRO/Sources/Main.c:357-361` | `ALRT` 1030 never appears |
| `SetAppMemorySize(...)` | `GliderPRO/Sources/Environ.c:688` | `ALRT` 180 tells the user to raise memory manually instead of doing it |
| the `hasScrap` branch | `GliderPRO/Sources/Menu.c:197-216` | **Paste is permanently disabled** and permanently reads `Nothing To Paste` (localized string 44) |
| the `kUseQDItem` case body | `case` at `GliderPRO/Sources/Settings.c:1203`, body commented out at `:1204-1205`, bare `break;` at `:1206` | the "Use QuickDraw" checkbox (`kUseQDItem` = 16, `GliderPRO/Sources/Settings.c:34`) in the Display pane does nothing. The `wasQD` variable it would have toggled appears **only** in comments (`Settings.c:934, 1204, 1205`) and has **no field in `prefsInfo`**, so the checkbox is not even persisted |
| `HideMenuBarOld()` | defined at `GliderPRO/Sources/MainWindow.c:409`, prototype commented out at `GliderPRO/Headers/Externs.h:360`, all four call sites commented out (`Main.c:348`, `Play.c:139`, `Play.c:415`, `Play.c:807`) | the pre-`BUILD_ARCADE_VERSION` menu-bar hiding |
| the colour-fade block | `GliderPRO/Sources/MainWindow.c:250+` | `fadeGraysOut` / `isDoColorFade` do nothing |

### 13.3 Unreachable functions (compiled, never called)

| Function | Definition | Associated resource |
|---|---|---|
| `QueryResumeGame` | `GliderPRO/Sources/Menu.c:710-759` (with `UpdateResumeDialog` at `:659-663` and `ResumeFilter` at `:668-703`) | `DLOG`/`DITL` 1025 `'Resume Game'` — the whole older saved-game UI |
| `SaveGame` | `GliderPRO/Sources/SavedGames.c:303-351` | writes `houseType.savedGame` + `hasGame` |
| `SavedGameMismatchError` | `GliderPRO/Sources/SavedGames.c:152-164` | `ALRT` 1044 |
| `BitchAboutSM3` | `GliderPRO/Sources/Sound.c:527-535` | `ALRT` 1030 |
| `SetAppMemorySize` | `GliderPRO/Sources/Environ.c:699+` | rewrites the app's own `'SIZE'` resource |
| `WriteScoresToDisk` | `GliderPRO/Sources/HighScores.c:742` | reachable only when `houseIsReadOnly`, which is always false |
| `ReadScoresFromDisk` | `GliderPRO/Sources/HighScores.c:801` | ditto |
| `CreateScoresFolder` | `GliderPRO/Sources/HighScores.c:642` | ditto — `G-PRO Scores ƒ` folder is never created |
| `FindHighScoresFolder` | `GliderPRO/Sources/HighScores.c:665` | ditto |
| `OpenHighScoresFile` | `GliderPRO/Sources/HighScores.c:720` | ditto — `'gliS'` files are never created |

The root cause of the last five is one stub:

```c
// GliderPRO/Sources/HouseIO.c:659-663
Boolean IsFileReadOnly (FSSpec *theSpec)
{
#pragma unused (theSpec)
    return false;
```

so `houseIsReadOnly` (`GliderPRO/Sources/HouseIO.c:187`) is permanently false, killing three
branches (`HouseIO.c:329`, `:428`, `:533`) plus `MainWindow.c:76` (the read-only splash label
colour).

### 13.4 Resources present but never referenced

| Resource | Notes |
|---|---|
| `DLOG`/`DITL` 1025 `'Resume Game'` | only `QueryResumeGame` uses it |
| `DLOG`/`DITL` 1026 `'Validation'` | gated out by `COMPILENOCP 1` |
| `ALRT`/`DITL` 1009 `'Banner'` | `kHouseBannerAlert 1009` is `#define`d at `GliderPRO/Sources/Play.c:18` and never used — the pre-graphical banner UI |
| `ALRT`/`DITL` 1030 `'Sound Manager'` | call site commented out |
| `ALRT`/`DITL` 1037 `'Not In Demo'` | `#ifdef COMPILEDEMO` |
| `ALRT`/`DITL` 181 `'Low Memory'` | `#ifdef COMPILEDEMO` |
| `ALRT`/`DITL` 1044 `'Saved Game Mismatch'` | caller unreachable |
| `STR#` 128 `'Jinjur'` (`Crystal`) | no `GetIndString` call |
| `PICT` 1002 (`kLoadTitlePict8`) | the 8-bit variant of the Load House title; `DITL` 1000 item 3 hard-codes `PICT` **1001** |
| `PICT` 1003 (`kDefaultHousePict1`) | the 1-bit default-house icon; `UpdateLoadDialog` always uses `kDefaultHousePict8` |
| `STR#` 1007 indices 32, 48, 74-80, 89-96, 111, 112, 122-128 | placeholder names for unused object codes |
| `STR#` 170/171 index 7 | `Couldn't Validate` / `Without your original disk…` — gated out by `COMPILENOCP` |
| `STR#` 1006 indices 18-21 | the four saved-game yellow alerts |
| `PICT` 152 | **does not exist** — About uses 150 (button normal), 151 (button hilit), 153 (logo) |
| `DITL` 150 item 9 | a third 120×9 `userItem`; only items 7 and 8 are ever drawn |

### 13.5 Preferences fields with no effect

Field names and offsets verified against the `prefsInfo` declaration at
`GliderPRO/Headers/Externs.h:233-267` and the gcc `#pragma pack(2)` probe in §6.1.

| Field | Offset (dec / hex) | Global | UI control | Why dead |
|---|---|---|---|---|
| `wasZooms` | 208 / `0xD0` | `doZooms` | Brains pane, `kDoZoomsCheck` | `ZoomBetweenWindows` and every zoom helper is commented out (§4.1) |
| `wasQuickTrans` | 215 / `0xD7` | `quickerTransitions` | Brains pane, `kQuickTransitCheck` | every `DissBits`/`DissBitsChunky` call in the program is commented out (§13.2) |
| `wasDoColorFade` | 211 / `0xD3` | `isDoColorFade` | Display pane, `kDoColorFadeItem` | both consumers are commented out: `if ((fadeGraysOut) && (isDoColorFade))` at `GliderPRO/Sources/MainWindow.c:249` and `if ((isDoColorFade) && (thisMac.isDepth == 8))` at `GliderPRO/Sources/Main.c:351`. `fadeGraysOut` is still *assigned* (`GliderPRO/Sources/InterfaceInit.c:133, 135`) but never read |

All three are still read from and written to the prefs file, still toggled by their checkboxes, and
still persist across launches — they simply have no consumer. A port must keep them in the file
layout (they occupy real bytes at fixed offsets) even if it ignores their values.

There is **no** preferences field for the "Use QuickDraw" checkbox at all: `wasQD` exists only in
comments (`GliderPRO/Sources/Settings.c:934, 1204, 1205`), so that checkbox is neither honoured nor
saved and always reads back as whatever `GetNewDialog` gives it (`CNTL`/`DITL` default 0).

### 13.6 Bugs that are user-visible

Collected here so a porter can decide deliberately rather than accidentally.

| # | Bug | Location |
|---|---|---|
| 1 | Pause-key radio **toggles** `wasEscPauseKey` instead of assigning it, so re-clicking the already-selected radio desyncs the boolean from the radio state | `GliderPRO/Sources/Settings.c:583` |
| 2 | `DisplayFilter` maps the `R`/`r` key to `kUseScreen2Item` (item 17) but flashes `kUseQDItem` (item 16) — the wrong checkbox blinks. `U`/`u` maps to `kUseQDItem`, whose handler is empty, so that shortcut does nothing at all | `GliderPRO/Sources/Settings.c:1067-1072` (R) and `:1074-1077` (U) |
| 3 | `BrainsFilter` has **no** keyboard shortcut for items 13 (`kDoPrettyMapCheck`) or 14 (`kDoBitchDlgsCheck`) | §6.11 |
| 4 | Brains "Defaults" sets the max-files **text field** to 24 without touching `willMaxFiles`; `SetAllDefaults` and the no-prefs path use 48 | `SetDialogNumToStr(theDialog, kMaxFilesItem, 24L)` at `GliderPRO/Sources/Settings.c:108` vs `maxFiles = 48; willMaxFiles = 48;` at `GliderPRO/Sources/Main.c:156-157` |
| 5 | `doPrettyMap` no-prefs default is `false` but both Defaults buttons set `true` | `doPrettyMap = false;` at `GliderPRO/Sources/Main.c:187` vs `wasPrettyMap = true;` at `GliderPRO/Sources/Settings.c:119` |
| 6 | `SortHouseList` compares two entries against themselves — `theHousesSpecs[i].vRefNum == theHousesSpecs[i].vRefNum` and `theHousesSpecs[i].parID == theHousesSpecs[i].parID` are both tautologies — so the dedupe collapses same-named houses in *different* folders | `GliderPRO/Sources/SelectHouse.c:527-528` |
| 7 | `AboutFilter` leaves `handledIt` **uninitialized** when an `updateEvt` targets neither `mainWindow` nor the dialog | `GliderPRO/Sources/About.c:230-248` |
| 8 | `RedrawSplashScreen` copies **Main → Work** instead of Work → Main, so it shows nothing and clobbers the work buffer | `GliderPRO/Sources/MainWindow.c:113` |
| 9 | `BringUpBanner` paints the clean room over the banner before waiting, so the banner is never seen; the user experiences a silent 15-second pause | `GliderPRO/Sources/Banner.c:184-192` |
| 10 | `ReadScoresFromDisk` reads `GetEOF` bytes into a fixed 292-byte field — buffer overrun (dead path) | `GliderPRO/Sources/HighScores.c:823, 841` |
| 11 | `GetLineOfText` copies the terminating CR into the returned line and can read 1-2 bytes past the logical end | `GliderPRO/Sources/StringUtils.c:201-206` |
| 12 | `DrawHighScores` highlights four of five fields for the new entry but always draws the word `room`/`rooms` in cyan | `GliderPRO/Sources/HighScores.c:240` |
| 13 | ⌘D collides: Options ▸ Demo… and House ▸ Duplicate Object. `MenuKey`'s bar order makes Options win | §3.6 |
| 14 | `kSavingGameDial 1042` collides with `ALRT` 1042 `'Color Switched'` (the former is unused) | `GliderPRO/Sources/Input.c:19` |
| 15 | `DITL` 1000 row 3's name items are shifted +1 px in x (lefts 17/117/217/317 vs 16/116/216/316 in rows 1-2) | §7.10 |
| 16 | Game ▸ Open Saved Game… sets `resumedSavedGame = true` and never clears it, permanently disabling high scores until the next New Game — even though the load then fails | `GliderPRO/Sources/Menu.c:318-324` |
| 17 | `prefsInfo` has two **uninitialized pad bytes** (offsets 145 and 225) written to disk, so the file is not byte-reproducible | §6.1 |
| 18 | `DoOpenDocAE`'s invalidation rect starts 38 px right of where `DrawOnSplash` writes the `House:` label, leaving stale pixels for long names | §10.2 |
| 19 | `BitchAboutColorDepth` is the only alert wrapper that omits `InitCursor()`, so a spinning cursor can persist over the alert | `GliderPRO/Sources/Events.c:46-55` |
| 20 | Version strings disagree three ways: `'vers'` 1/2 say 1.1.2 © 1994-95, `DITL` 150 item 6 says © 1994-2000, the source tree says 1.0.4 | §10.6 |
---

## Open questions

Things this document could not settle from the source tree and the resource fork alone. Each entry
says what is unknown, why it matters, and what evidence would settle it.

### Q1. The source tree and the resource fork are from different builds

`vers` 1 and `vers` 2 both carry `01 12 80 00 00 00` = version **1.1.2**, release stage `0x80`
(final), non-release `0x00`, and both have short version string `1.1.2`. Only **`vers` 1** carries
the long string `Glider PRO™ 1.1.2\r© 1994-95 Casady & Greene, Inc.` (0x31 = 49 bytes); `vers` 2's
long string is just `© 1994-95 Casady & Greene, Inc.` (0x1F = 31 bytes) — see §5.6, §10.6. The About box's `DITL` 150 item 6 is a *separate* static string reading
`© 1994-2000 Casady & Greene, Inc.`. The task brief and the `Sources` tree describe 1.0.4.

Consequence: the `DITL`/`DLOG`/`MENU`/`STR#` geometry documented here is the geometry of the
**resource fork that shipped in `Glider PRO.r`**, and a few item numbers in `Settings.c` and
`SelectHouse.c` might not correspond one-for-one if the `Sources` tree is older. Everywhere in this
document that a source constant and a resource item number are cross-checked, they agreed; but the
check is not exhaustive for the editor-only dialogs (1013-1016, 1019, 1022, 1027, 1028).

What would settle it: the `.r` dump of the 1.0.4 fork, or the build's `MPW`/`CodeWarrior` project
file listing the resource file version.

### Q2. The two `clut` resources are not decoded here

`ColorText`, `Index2Color`, and `kRedOrangeColor8` (= 23) all index a 256-entry colour lookup table
by *index*, not by RGB (§4.7, §9.9, §10.2). The specific indices used by the UI code are **4, 5, 23,
28, and 244**. The document reports the indices faithfully but does **not** report their RGB values,
because the palette lives in the two `clut` resources (and 20 `dctb`/1 `cctb`/1 `wctb` colour tables)
which were not parsed.

Consequence: a Go port that renders in RGB must dump the `clut` to get the actual colours, or the
high-score screen, splash label, and error text will be the wrong hue.

What would settle it: parse the `clut` resources out of `Glider PRO.r` (`clut` layout: `long seed;
short flags; short size; then (size+1) × {short value; ushort r,g,b}`) and record entries 4, 5, 23,
28, 244.

### Q3. Whether the missing dissolves were intentional

Every `DissBits`/`DissBitsChunky` call in the shipped source is commented out (§13.2). Three of them
leave visible breakage: `RedrawSplashScreen` shows nothing, `BringUpBanner` shows nothing for 15
seconds, `DoHighScores` shows nothing. This looks like a mid-refactor state (the `// TEMP` comments
in `Play.c` and `HouseIO.c` support that reading), not a design decision.

Consequence: a port must choose. Reproducing the dead state gives a game where the banner and high
score screens are invisible, which no player of the shipped game experienced — so the *shipped
binary* presumably had them working, and the `Sources` tree is a snapshot mid-surgery. This document
recommends implementing the transitions (see P11 below) and treating the commented-out calls as the
intended behaviour.

What would settle it: running the shipped 1.0.4/1.1.2 binary, or finding a `DissBits`
implementation in `Utilities.c` that is itself intact (it is — only the *call sites* are commented).

### Q4. `prefsInfo` pad-byte contents

The 226-byte prefs record has two structural pad bytes inserted by `#pragma options align=mac68k`
(offsets **145** and **225**, §6.1). `SavePrefs` does `BlockMove`/`FSWrite` of `sizeof(prefsInfo)`
bytes from a stack local, so the two pad bytes contain whatever was on the stack.

Consequence: prefs files are not byte-reproducible and cannot be compared with a hash. A Go port
should write them as `0x00`, which is compatible on read (nothing reads them) but will not match a
Mac-written file byte-for-byte.

What would settle it: nothing in the source; only a corpus of real `Glider Prefs` files.

### Q5. Was a saved-game file format ever shipped?

`game2Type` (110-byte header + `nRooms` × 292-byte `savedRoom`, §8.4-8.5) is fully declared and
`SaveGame2`/`OpenSavedGame` are fully written — and then entirely commented out or stubbed to
`return false`. No `'gliG'` file exists in `GliderPRO/Houses/`. 20 of the 22 shipped houses have
`hasGame == 0` with a stale `savedGame` block, but **ImagineHouse PRO II and Titanic ship with
`hasGame == 1` and a `savedGame.version == 0x0100` record** (§8.8) — so in-house saves were being
produced at some point by an older build, even though no external `'gliG'` file format ever shipped.

Consequence: a port has no reference file to validate against, and no user expectation to honour.
The recommendation in §8.10 is to implement a *new* save format rather than resurrect `game2Type`.

What would settle it: a real `'gliG'` file from the wild, or a version of the source with the code
uncommented.

### Q6. The exact `houseType.flags` bit assignments beyond bit 2

`ReadHouse` decodes exactly three bits, all at `GliderPRO/Sources/HouseIO.c:416-418`:
bit 0 → `wardBitSet = ((flags & 1) == 1)`, bit 1 → `phoneBitSet = ((flags & 2) == 2)`, and bit 2 →
`bannerStarCountOn = ((flags & 4) == 0)`, the last being inverted (§10.5). What the ward and phone
bits *mean* is the open question, not which bits are read. Across the 22 shipped houses only two
values occur besides zero: `0x00000002` (seven houses, phone bit) and `0x00000006` (Art Museum,
phone bit + star count off) — see §8.8. No shipped house sets bit 0, nothing above bit 2 is ever
set, and no code writes `flags` at all.

Consequence: a port must preserve `flags` verbatim on round-trip rather than reconstruct it.

### Q7. `smWarnings` semantics

`prefsInfo.smWarnings` (offset 202, §6.1) is described in the code only by its name. It is loaded,
saved, and — as far as the greps in this document reach — never compared or incremented in the UI
code. It is plausibly a Sound Manager nag counter that a removed `BitchAboutSM3` path used to bump
(§13.3), since that function is the only "warn once" candidate and it is itself dead.

What would settle it: a grep of the audio subsystem beyond `Sound.c` (this document's assignment
stopped at the UI boundary).

### Q8. Whether `maxFiles` was meant to be live-resizable

`theHousesSpecs` is `NewPtr(70 * maxFiles)` once at launch
(`GliderPRO/Sources/StructuresInit2.c:275-278`), so `maxFiles` is frozen for the session, and the
Brains panel edits `willMaxFiles`, which takes effect next launch (§6.11-6.12). But
`ReadInPrefs` snaps out-of-range values to **12** while the Brains panel clamps to **[12, 500]**
(§6.4, §6.11), and `SetBrainsToDefaults` writes **24** into the text field while `SetAllDefaults`
and the no-prefs path use **48** (§13.6 bug 4). Four different numbers for the same concept.

Consequence: a port with dynamic slices has no reason to keep any of them, but if it wants to
reproduce the "restart required" nag (`nextRestartChange`, ALRT 1040) it must pick one.

### Q9. The `'demo'` resource's record semantics

`'demo'` 128 is `kDemoLength` = **6702** bytes (`GliderPRO/Headers/GliderDefines.h:625`) of 6-byte
`demoType` records (1117 records) (§2.5,
§10.4). This document establishes the size and the playback trigger but not the field meanings —
that belongs to the input/physics subsystem.

### Q10. Second-monitor support

`isUseSecondScreen` / `prefsInfo.wasScreen2` (offset 220) is persisted and has a Display-pane
checkbox (`kUseScreen2Item` = 17). This document did not trace whether the multi-`GDevice` walk that
would honour it exists in `Environ.c` beyond `WhatsOurDepth`/`SwitchToDepth`.

### Q11. Keyboard layout dependence of the stored key bindings

Prefs store **KeyMap bit offsets** (§6.10), which are hardware/ADB positions, not characters. The
`GetKeyName` table in §6.10 maps offsets to display names for a US layout. Whether Glider PRO
shipped localized `GetKeyName` tables (it has no `KCHR`-driven lookup) is unknown; the table is
hard-coded in `GliderPRO/Sources/Utilities.c`.

Consequence: a Go port keying off `KeyEvent.Code` (USB HID / physical) is closer to the original
than one keying off runes.

### Q12. What `ictb` 1000-something and the 5 `mctb`/1 `cctb` resources customize

The fork contains 1 `ictb` (dialog item colour table), 5 `mctb` (menu colour tables), 1 `cctb`
(control colour table), 20 `dctb` (dialog colour tables) and 1 `wctb` (window colour table) (§1.3).
These override the System's default greys for specific dialogs, menus, and controls. This document
records their existence and counts but not their contents.

Consequence: a pixel-exact port needs them; a merely faithful port does not.

---

## Porting notes

Ordered roughly by how much damage getting it wrong does.

### P1. Reproduce the preferences record byte-for-byte, then never read it again

The 226-byte big-endian `prefsInfo` (§6.1) is the only on-disk format in this subsystem that a Go
port might plausibly need to interoperate with (a user's existing `Glider Prefs`). Everything about
it is a trap:

- `#pragma options align=mac68k` == `#pragma pack(2)`. Not `pack(1)`, not natural alignment. The
  gcc probe in §6.1 was only correct after `#define long int32_t`, because native `long` is 8 bytes
  on this machine and would have produced 242.
- `long` is **32-bit**. The four key-binding fields (`wasLeftMap`, `wasRightMap`, `wasBattMap`,
  `wasBandMap`, offsets 146/150/154/158) are 4-byte KeyMap **bit offsets**, not key codes.
- Pascal strings are fixed-size arrays including the length byte: `Str32` = 33 bytes,
  `Str15` = 16, `Str31` = 32. `wasDefaultName` at offset 0 is `Str32` = **33** bytes, which is why
  `wasLeftName` starts at 33 (odd) and why the compiler inserts the pad at 145.
- The version gate is `kPrefsVersion` = **0x0034** (52 decimal, `GliderPRO/Sources/Main.c:16`) at
  offset **164**. `LoadPrefs` compares `!=` and on mismatch shows ALRT 160, **deletes the file**,
  and returns false (`GliderPRO/Sources/Prefs.c:261-266`). There is no migration path in the
  original. A Go port should migrate instead of deleting, or at minimum warn.
- Two uninitialized pad bytes (offsets 145, 225). Write zeros.

Recommended Go approach: parse the legacy 226-byte record once on first run to seed a native
config (JSON/TOML), then stop touching it. Do **not** make the Go config a 226-byte binary blob.

### P2. `KeyMapOffsetFromRawKey` is an involution — implement it, do not invert it

`GliderPRO/Sources/Utilities.c:504-517` converts between a virtual key code and a KeyMap bit offset
by reversing the nibble ordering within each byte-group. The function is its own inverse, and the
original calls it in both directions with the same code. §6.10 has the function and a 15-row
verification table. Get this wrong and every key binding in an imported prefs file is garbage.

A Go port keyed to physical scancodes should store its own representation and only run this
transform at the legacy-import boundary.

### P3. All dialog centring is dead — do not "fix" it silently

Every `CenterDialog`/`CenterAlert` call in the program is commented out (§13.2). Dialogs therefore
appear at the literal global-coordinate bounds in their `DLOG`/`ALRT` resource. **19** of them are
`(0,0)`-anchored and so appear **partly under the 20-pixel menu bar** (re-derived by scanning
bytes 0-7 of every `DLOG` and `ALRT`; no other dialog has `top < 20`):

* `ALRT` 130 (Color Depth), `ALRT` 1002 (Save Changes), `ALRT` 1009 (Banner, dead anyway)
* `DLOG` 1003 (Room Info), 1020 (High Name)
* the 14 editor object-info dialogs: `DLOG` 1007, 1010, 1011, 1013, 1014, 1015, 1016, 1019, 1022,
  1027, 1033, 1034, 1035, 1045

The three that matter for the non-editor UI covered by this document are `ALRT` 130, `ALRT` 1002 and
`DLOG` 1020; the rest are editor-only.

A Go port on a modern display should centre them — but be aware that this is a deliberate deviation,
and that the (0,0) origins are the fingerprint that tells you which resources the original author
had already converted to the centred path and which he had not.

### P4. Toolbox replacements, item by item

| Mac Toolbox facility | Used for | Go replacement |
|---|---|---|
| Dialog Manager (`GetNewDialog`, `ModalDialog`, `Alert`) | every dialog in §5 | your own modal loop over a widget tree built from the `DITL` tables in this document |
| `DITL` item types (§1.4) | button/checkBox/radioButton/statText/editText/icon/picture/userItem | a `DialogItem` struct with a kind enum; the `0x80` disabled bit becomes a `Disabled bool` |
| `ParamText` + `^0`…`^3` | ALRT 140, 170, 1002, 1006, 1009, 1020, 1044, 180, 181 | `strings.NewReplacer("^0", …)`; note the substitution is **positional and global**, and ALRT 1044 uses `^0` **twice** |
| `ALRT` `stages` word (§11.11) | default item + beep count per stage | in practice every alert uses one nibble for all four stages, so: `defaultItem = (nibble&8)!=0 ? 2 : 1`, `beeps = nibble&3`. ALRT 1041 and 1029 are the only two with default item 2; 1029 is the only one with 3 beeps |
| Resource Manager (`GetIndString`, `GetPicture`, `GetResource`, `UseResFile`) | `STR#` (§12), `PICT`, `ICON`, `cicn` | `embed.FS` + generated Go tables. `GetIndString` is 1-based and returns "" out of range — replicate that, several call sites rely on it |
| `NewModalFilterUPP` filter procs | keyboard shortcuts inside every prefs pane (§6.7-6.11) | a `func(ev Event) (item int, handled bool)` hook per dialog |
| `NewRgn`/`OpenRgn`/`Line`/`PtInRgn` | the About box's **diamond-shaped** OK hot zone (§10.6) | a point-in-polygon test on the 4-segment path; the polygon is not the button's rect |
| `KeyMap` + `BitTst` | modifier polling in `WaitForInputEvent` (§9.8) | poll modifier state; note `BitTst` counts bits from the MSB, hence the `31 - bitNumber` idiom (§12.5) |
| `FindFolder(kPreferencesFolderType)` with a `STR#` 160 fallback | prefs location (§6.2, §12.5) | `os.UserConfigDir()`; drop the fallback entirely |
| `PBGetCatInfo` breadth-first volume walk | house discovery (§7.3) | `filepath.WalkDir` with the same 32-directory cap if you want fidelity, or no cap |
| Finder type/creator (`'gliH'`/`'ozm5'`) | house identification (§7.2) | **see P5** |
| `PurgeSpace` | About box "Memory: NNNK" (§10.6) | `runtime.MemStats`, or omit the line |
| `Gestalt` | `CanUseFindFolder`, `hasQT`, `hasSM3` | compile-time constants |
| `GetDefaultOutputVolume`/`SetDefaultOutputVolume` | the Sounds pane (§6.9) | **see P7** |
| 8-bit indexed colour + `Index2Color` | `ColorText` indices 4/5/23/28/244 | see Q2; resolve via the `clut` |
| big-endian on-disk data | every format in §6, §8, §9 | `encoding/binary.BigEndian` everywhere; never `unsafe` casts |
| MacRoman text | all `STR#` and `DITL` strings | decode at build time to UTF-8. The bytes that matter: `0xA5` `•` (§9.9), `0xA9` `©`, `0xAA` `™`, `0xC4` `ƒ` (§9.7), `0xC9` `…` (§12.4) |

### P5. House discovery must be replaced, not translated

The original finds houses by scanning for Finder **type `'gliH'` and creator `'ozm5'`**
(§7.2-7.3), not by filename or extension. A Go port has no HFS type/creator. Options, best first:

1. Extension convention (`*.glh`) plus a magic-number sniff.
2. Sniff only: read the first 866 bytes and require `version == 0x0200` **and**
   `866 + 348*nRooms <= fileLength`. Checked against all 22 shipped houses (§8.8): the strict
   equality `866 + 348*nRooms == fileLength` holds for 21 of them (Slumberland
   `866 + 348*383 == 134150` ✓, Empty House `866 + 348*35 == 13046` ✓) but **fails for `Sampler`**,
   whose data fork is 1564 bytes where `nRooms = 2` predicts 1562. Use `<=`, or the port will refuse
   to list a house that the original happily opens.

Option 2 is still strong: an arithmetic identity derived from 2 header bytes and bounded by the file
length is a very low false-positive test. Note `version == 0x0200` is safe — all 22 shipped houses
have exactly that, and `kNewHouseVersion` (`0x0300`) is unused on disk.

### P6. The saved-game feature is inert — decide up front

`OpenSavedGame` is `return false;` (`GliderPRO/Sources/SavedGames.c:169`), `SaveGame2`'s body is
commented out (`:33-147`), and `SaveGame`'s only call site is commented out
(`GliderPRO/Sources/Menu.c:458`). Observable consequences a port will otherwise reproduce by
accident (§8.7, §13.6 bug 16):

- ⌘S flickers `kSavingTitleMode` and writes nothing.
- ⌘Q shows ALRT 1041 ("Save First" / "Don't Save"); "Save First" is a no-op, then it quits.
- Game ▸ Open Saved Game… shows ALRT 1046 ("So What?"), fails to load, **and permanently sets
  `resumedSavedGame = true`**, which makes `TestHighScore` return false for the rest of the session
  (`GliderPRO/Sources/HighScores.c:380-381`). Only starting a New Game or Two Player Game clears it
  (`GliderPRO/Sources/Menu.c:307, 313`).

Recommendation: implement real saves with a new format, and either remove the ALRT 1046 path or
make it correct. If you keep the menu item, **clear `resumedSavedGame` when the load fails.**

### P7. The Sounds pane hijacks the system volume

Read: `LoWord(GetDefaultOutputVolume(&count)) / 0x24`, clamped to 0-7. Write:
`min(volume * 0x25, 0x100)` duplicated into **both** 16-bit channel fields of the long (§6.9). The
game changes the **system** output volume for the session and restores it only on a clean quit —
crash and the user's Mac stays at whatever Glider set.

Also: level 7 is displayed as the string **"11"** in three places in the UI (a *Spinal Tap* joke).
Keep the joke; do not keep the system-volume hijack. A Go port should use a per-app gain.

### P8. Menu enable/disable is centralized in one function — port it as one function

`UpdateMenus(Boolean newMode)` (§3.4-3.5) is the single authority. It is called from 20+ sites after
any state change. Do not scatter the rules; replicate the function, including:

- The three-mode dispatch (`kSplashMode` 0, `kPlayMode` 1, `kEditMode` 2, §2.1).
- **Paste is permanently disabled** and its text is permanently localized string 44
  `Nothing To Paste`, because the `hasScrap` branch is commented out
  (`GliderPRO/Sources/Menu.c:197-216`).
- The Edit-menu item **renaming** from localized strings 36-43 (`Cut Object`/`Cut Room`/…) based on
  the current editor selection.
- `iSaveAs` enable/disable is commented out at `GliderPRO/Sources/Menu.c:107, 112, 150`, so House ▸
  Save As stays in whatever state the `MENU` resource shipped with.

### P9. `BUILD_ARCADE_VERSION` arrow keys bypass the menu rules

With `BUILD_ARCADE_VERSION` = 1 (`GliderPRO/Headers/GliderDefines.h:16`), the raw event loop maps
arrow keys in **both** splash and edit mode (`GliderPRO/Sources/Events.c:191-207`): Left → High
Scores, Right → Demo, and **both** Up and Down → New Game (§2.6). These fire without consulting
menu enable state. A port that only wires the menus will silently lose the arcade behaviour; a port
that wires the arrows but keeps modern menu gating may allow a new game at a moment the original
would not.

### P10. `houseIsReadOnly` is permanently false — three code paths are unreachable

`IsFileReadOnly` is `#pragma unused (theSpec) return false;`
(`GliderPRO/Sources/HouseIO.c:659-663`). This kills:

- the entire `'gliS'` external high-score file (`WriteScoresToDisk`, `ReadScoresFromDisk`,
  `CreateScoresFolder`, `FindHighScoresFolder`, `OpenHighScoresFile`, and the `G-PRO Scores ƒ`
  folder, §9.6-9.7),
- the read-only splash-label colour at `GliderPRO/Sources/MainWindow.c:76`,
- the score-reload branch at `GliderPRO/Sources/HouseIO.c:428`.

A Go port on a modern OS **will** encounter read-only houses (bundled/system-installed content,
read-only mounts), so this is the one dead path worth reviving. If you do, fix
`ReadScoresFromDisk`'s length bug first: it reads `GetEOF`-many bytes into a fixed 292-byte field
(`GliderPRO/Sources/HighScores.c:823, 839-841`). Clamp to `sizeof(scoresType)`.

### P11. Fix the three broken composites, or the UI is invisible

| Symptom | Cause | Fix |
|---|---|---|
| `RedrawSplashScreen` does nothing and corrupts the work buffer | `CopyRectMainToWork(&workSrcRect)` at `GliderPRO/Sources/MainWindow.c:113` is the wrong direction, and the `DissBits` that should have done the real work is commented out at `:109-112` | copy Work → Main |
| The house banner is never seen; the game pauses 15 s (4 s in demo) on a clean room | `BringUpBanner` step 4 blits `backSrcMap` → `workSrcMap` over the freshly composed banner (`GliderPRO/Sources/Banner.c:184-192`) because the surrounding `DissBits` calls are commented out | present the banner, wait, *then* restore |
| The high-score screen is never seen | both `DissBits` pairs commented out at `GliderPRO/Sources/HighScores.c:68-71, 76-79` | blit `workSrcMap` → main |

All three are the same root cause: a transition refactor left half-finished (Q3).

### P12. `houseType.timeStamp` bit 0 is a lock flag, not part of the date

`WriteHouse` (`GliderPRO/Sources/HouseIO.c:477-487`) does
`GetDateTime(&timeStamp); timeStamp &= 0x7FFFFFFF;` and then sets or clears **bit 0** as the
"house is locked" flag; `ReadHouse` (`GliderPRO/Sources/HouseIO.c:402`) reads it back with
`houseUnlocked = ((timeStamp & 0x00000001) == 0)` (§8.9). So the stored value is a Mac epoch
(1904-01-01) seconds count with the top bit cleared and the bottom bit stolen.

Verified empirically against all 22 shipped houses (§8.8) — 15 locked, 7 unlocked; two examples:

```
Slumberland  stored 743682377  bit0=1 -> LOCKED
Empty House  stored 741421292  bit0=0 -> UNLOCKED
```

To recover a real date you must OR `0x80000000` back in *and* mask bit 0, then convert from the 1904
epoch. Do not treat the field as a plain timestamp: a port that writes `time.Now()` into it will
flip the lock bit roughly half the time.

The secret editor unlock is Option-clicking the **Brains** button in the Preferences main dialog
(`GliderPRO/Sources/Settings.c:1448-1453`): `houseUnlocked = true; changeLockStateOfHouse = true;
saveHouseLocked = false;`, persisted through this bit on the next `WriteHouse`.

### P13. High scores live inside the house file

`scoresType` is **292 bytes** embedded at offset **528** of the house header (§9.1, §8.4), not in a
separate file. Layout: `banner` `Str31` at 0 (32 B), `names[10]` `Str15` at 32 (160 B),
`scores[10]` `long` at 192 (40 B), `timeStamps[10]` `long` at 232 (40 B), `levels[10]` `short` at
272 (20 B). Consequences:

- Saving a high score dirties the **house file** (`gameDirty = true`,
  `GliderPRO/Sources/HighScores.c:414`), which means a read-only house cannot record scores at all
  in the shipped build (P10).
- `TestHighScore` always writes into slot `kMaxScores-1` (index 9) and then calls `SortHighScores`
  (§9.3). Reproduce that order; the sort is what determines the final rank shown in `DLOG` 1020.
- `SortHighScores` copies from an **uninitialized** `tempScores` stack local if every remaining
  score is negative (`GliderPRO/Sources/HighScores.c:283, 305`). Scores are never negative in
  practice, but a Go port must not translate the sentinel logic literally.
- `ZeroHighScores` writes the 14-character string `"--------------"` into all 10 name slots and
  copies `thisHouseName` into the banner (`GliderPRO/Sources/HighScores.c:323-343`), which matches
  the observed Empty House bytes exactly (§8.8).

### P14. Exact geometry matters for the high-score and banner screens

Both are drawn with hard-coded offsets into the 640×460 splash, not laid out. §9.9 and §10.5 have
the full arithmetic. The load-bearing constants:

- `kScoreSpacing` 18, `kScoreWide` 352, `kKimsLifted` 4
  (`GliderPRO/Sources/HighScores.c:90-92`).
- `scoreLeft = (screenWidth - 352) / 2`, `dropIt = 129 + splashOriginV`
  (`GliderPRO/Sources/HighScores.c:106-107`).
- Row 0 baseline is `dropIt - kScoreSpacing - kKimsLifted`; rows 1-9 are `dropIt + i*18`.
- Every field is drawn twice, a 1-pixel drop shadow then the colour text, at x offsets
  1/0 (place), 31/30 (name), 161/160 (level), 193/192 (room word), 291/290 (points).
- Footer: `MoveTo(scoreLeft + 80, dropIt - 1 + 10*18)`, `TextSize(9)`, blue, localized string 8.
- Banner page is 330×220: `PICT` 1993 (330×190) unmasked on top, `PICT` 1992 (330×30) `CopyMask`ed
  with `PICT` 1991 below (`GliderPRO/Sources/Banner.c:41-84`).
- Banner text lines at `topLeft.h + 16`, `topLeft.v + 32 + line*20`; the star-count lines are at
  fixed `topLeft.v + 164` and `+180` in red, so **a banner longer than 6 lines collides with them**
  (`GliderPRO/Sources/Banner.c:117-166`).

### P15. `splashOriginV` centres for the wrong height

`GliderPRO/Sources/MainWindow.c:238-243` centres the splash as if it were 640×**480**, but `PICT`
1000 is 640×**460** (byte-verified, §5.5). The result is a 10-pixel upward bias on tall screens.
Every hard-coded offset in P14 is relative to that origin, so if you "fix" the centring you shift
the high-score table too. Fix both or neither.

Relatedly, `DoOpenDocAE`'s invalidation rect starts at `splashOriginH + 474` while `DrawOnSplash`
writes the `House:` label at `splashOriginH + 436` (§10.2), so a long house name leaves stale pixels
after a drag-and-drop open.

### P16. `GetLineOfText` is subtly wrong — do not translate it literally

`GliderPRO/Sources/StringUtils.c:140-208` splits a CR-delimited `Str255` into lines and
**includes the terminating CR in the returned line** (the copy loop is `i <= stop` where `stop` is
the CR's index), and when the source has no trailing CR the `stop` computation overshoots by 1-2
bytes. It is used for the banner and trailer messages (§10.5). Write a correct splitter; the visible
difference is a trailing space-like glyph at the end of every banner line.

### P17. Small fidelity details that are easy to miss

- `MENU` 129 item 1's text is literally `"New Game\0"` — Pascal length **9** with a trailing NUL
  byte (§3.1). Trim it or the menu shows a box glyph.
- `MENU` 128's title is the single byte `0x14` (the Apple logo in the Chicago font's MacRoman
  slot). A port needs its own Apple/About affordance.
- `MENU` 140 and 141 both carry an internal `menuID` of **133** (§3.1) — they are alternate
  versions of the same pop-up. Do not key a map on the internal ID.
- ⌘D is bound twice: Options ▸ Demo… and House ▸ Duplicate Object. `MenuKey` walks the menu bar in
  order, so **Options wins** (§3.6).
- `DisplayStarsRemaining` tests `numStarsRemaining < 2`, so **zero** stars draws the singular
  artwork `PICT` 1018 (`GliderPRO/Sources/Banner.c:223`).
- `DrawHighScores` highlights four of the five fields for a new entry but always draws the
  `room`/`rooms` word in cyan (`GliderPRO/Sources/HighScores.c:240`).
- `bannerStarCountOn` is **inverted**: bit 2 of `houseType.flags` *set* means do **not** show the
  count (`GliderPRO/Sources/HouseIO.c:418`).
- The auto-demo trigger is `kIdleSplashTicks` = **7200L** ticks = 120 s
  (`GliderPRO/Headers/GliderDefines.h:197`, comment `// 2 minutes`), and only fires in
  `kSplashMode` with `doAutoDemo` set and `switchedOut` clear
  (`GliderPRO/Sources/Events.c:536-540`).
- `WaitForInputEvent(short seconds)` takes **seconds**, not ticks
  (`timeToBail = TickCount() + 60L * (long)seconds;`, `GliderPRO/Sources/Utilities.c:446`), and `-1` means
  wait forever (§9.8). Getting this wrong makes the banner pause 15× too short or 900× too long.
- `DITL` 1000's Load House grid: row pitch **61** px, column pitch **100** px,
  `loadHouseRects[0] = (48, 16, 94, 112)`, and row 3's name items are shifted **+1 px** in x
  (lefts 17/117/217/317 vs 16/116/216/316) (§7.10, §13.6 bug 15). Reproduce or normalize, but know
  it is in the resource, not a bug in this document.

### P18. Restore the nil-resource guards

`RedAlert(kErrDialogDidntLoad)` checks after `GetNewDialog` are commented out in `DoAbout`
(`GliderPRO/Sources/About.c:52-53`) and throughout `DialogUtils.c` (§4.1). In the original a missing
`DLOG` crashes. In Go the equivalent is a nil-map or index panic on your embedded resource table —
add the check back and fail loudly with the resource ID.

### P19. Error reporting has three tiers — keep them distinct

| Tier | Function | Resource | Behaviour |
|---|---|---|---|
| Fatal | `RedAlert(short errorNumber)` | `ALRT` 170, `STR#` 170 (titles) + `STR#` 171 (bodies), 13 entries each | `ParamText(title, body, number, "")`, `Alert(170, nil)`, then **`ExitToShell()`**. Only shows the number when `errorNumber > 1` (`GliderPRO/Sources/Utilities.c:135-161`) |
| Recoverable | `YellowAlert(short whichAlert, short identifier)` | `ALRT` 1006, `STR#` 1006, 24 entries | `GetIndString` + `NumToString`, `ParamText`, `Alert(1006, nil)`. Returns (`GliderPRO/Sources/HouseIO.c:640-655`) |
| File I/O | `CheckFileError(OSErr, StringPtr fileName)` | `ALRT` 140, `STR#` 140, 17 entries | maps `OSErr` → string index via the switch at `GliderPRO/Sources/FileError.c:36-89`, `ParamText(msg, errNum, fileName, "")`. Returns **`true` on `noErr`**, false after alerting |

The `STR#` 170/171 pair is indexed by the same `kErr*` constant, so the two lists must stay in
lockstep — a Go port should fuse them into one `[]struct{Title, Body string}`.

### P20. Suggested Go package layout for this subsystem

```
internal/ui/
    resources/          generated from Glider PRO.r: strings, DITL tables, ALRT stages
        strings_gen.go      STR# 129/140/150/160/170/171/1005/1006/1007 as []string
        dialogs_gen.go      DLOG/ALRT bounds + DITL item lists
    dialog/             modal loop, item widgets, filter-proc hooks, ^0..^3 substitution
    menu/               menu tree + the single UpdateMenus(mode) authority
    prefs/              native config + legacy 226-byte importer (P1)
    housepick/          scan + the DITL 1000 grid (P5, P14)
    highscore/          scoresType codec + the DrawHighScores geometry (P13, P14)
    splash/             splash, banner, about (P11, P14, P15)
    errors/             the three tiers of P19
```

Keep the generated resource tables generated. Every table in §5, §11, and §12 of this document was
byte-verified out of `Glider PRO.r`, so a generator that reproduces them is checkable against this
file.
