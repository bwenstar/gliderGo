# Glider PRO — The Original Game

**Canonical consolidated reference for the `gliderGo` port.**

Subject: Glider PRO by John Calhoun / Casady & Greene, GPLv2 source release, vendored at
`GliderPRO/`. The source tree calls itself 1.0.4
(`GliderPRO/Sources/Main.c:3`); the shipped resource fork says **1.1.2**. See §2.

---

## How to use this document

This file is the **consolidated overview**. It exists so that a porter can answer 90% of
"what does the original do here?" questions without opening 123,000 lines of subsystem
analysis, and so that the facts everybody needs constantly — room geometry, the frame model,
the physics constants, the object ID table, the collision limits — live in one place.

**The per-subsystem documents in `docs/analysis/` are the detailed authority.** Where this
file and a subsystem doc disagree, the subsystem doc wins; where a subsystem doc and the C
source disagree, the source wins. Every consolidated claim below carries either a
`GliderPRO/...:line` citation into the vendored source or a pointer to the subsystem doc
section that specifies it in full. Nothing here is a design proposal: the port's
implementation plan is a separate document (`docs/PLAN.md`). The only forward-looking section
is §19, the **fidelity contract**, which states which original behaviours are non-negotiable.

Three global reading rules that apply to every citation in this file and in every analysis doc:

1. **Line numbers are into the LF-normalised copy.** Every `.c`/`.h` under `GliderPRO/` uses
   classic-Mac CR-only line endings. Normalise before citing:
   `tr '\r' '\n' < GliderPRO/Sources/Player.c`. Recipe:
   `docs/analysis/unresolved-format-decisions.md` §6.1.
2. **Use `command grep -an`.** The sources contain MacRoman high bytes (0xC5, 0xAA, 0xA9), so
   binary-detecting greps report "no match" for constants that are plainly present.
   `docs/analysis/unresolved-format-decisions.md` §6.2.
3. **QuickDraw ordering traps.** A Mac `Rect` is `{top, left, bottom, right}` and a `Point` is
   `{v, h}` — **vertical first**. But Glider's own helper `QSetRect(r, l, t, r, b)`
   (`GliderPRO/Sources/RectUtils.c:210-216`) takes **left, top, right, bottom**. Rect literals
   in the analysis docs are printed in `QSetRect` order unless stated otherwise.

### Precedence notice: docs that correct their siblings

Several analysis docs were fact-checked after the initial fan-out and carry corrections to
their own earlier text or to other docs. These supersede whatever they contradict:

| Corrective document / section | Supersedes |
|---|---|
| `format-decisions.md` D1-D7 | Six format/build questions left open in 3-5 docs each |
| `unresolved-format-decisions.md` | Index of all 46 formerly-open format items + 2 outright errors in `architecture.md` |
| `toolbox-primitives.md` R-RNG-1, R-XOR-1, R-CLUT4-1, R-FONT-1 | "unresolvable Toolbox behaviour" claims in `rendering.md`, `determinism.md`, `scoring.md` |
| `rendering.md` Appendix Z / ZB | Its own open questions Q1, Q2, Q7, Q8 |
| `enemies.md` Appendix A | Its own `TriggerSwitch(dynaNum)` claim and open question 2 |
| `object-draw-all.md` §13 | Draw-order claims in sibling docs |
| `editor-object-manipulation.md` §19 | Clamping claims in `editor.md` |
| `scoring.md` §Data-I-could-not-decode item 23 | Its own earlier claim about `PICT` 1998 |
| `house-format.md` §3.3.1 | Its own §3.3, which noted `timeStamp &= 0x7FFFFFFF` but read it as merely clearing a sign bit. It destroys the date: see §13.2 below |

---

## 1. Index of `docs/analysis/` — the detailed authority

28 documents, 123,418 lines. Read the one that owns the subsystem you are implementing.

| Document | Lines | What lives there |
|---|---:|---|
| [`architecture.md`](analysis/architecture.md) | 3,872 | Boot order, the three application modes, the Toolbox event loop, `RenderFrame`'s busy-wait pacer, window geometry, global state inventory |
| [`audio.md`](analysis/audio.md) | 3,675 | The 4-channel Sound Manager engine, the complete 64-entry sound table with priorities, the 7-piece music engine, `'snd '` format-1 parsing, PCM extraction plan |
| [`constants.md`](analysis/constants.md) | 8,192 | Every named compile-time constant and every load-bearing unnamed literal, grouped by subsystem; the master timing/geometry/object-ID catalogue |
| [`determinism.md`](analysis/determinism.md) | 4,505 | Whether the simulation is bit-reproducible: the integer timestep, RNG containment, per-frame ordering, shared mutable room state |
| [`determinism-networking.md`](analysis/determinism-networking.md) | 2,498 | Part 10 of the above: what a networked Go port would have to do; the consolidated 20-item porting-risk ranking |
| [`editor.md`](analysis/editor.md) | 11,823 | The built-in house editor: tools palette, map window, room/object/house info dialogs, link editing, legality enforcement |
| [`editor-object-manipulation.md`](analysis/editor-object-manipulation.md) | 4,715 | `ObjectEdit.c` in full: the six user gestures, clamping rules, the marquee, the geometry cache |
| [`enemies.md`](analysis/enemies.md) | 3,795 | Every hostile entity (balloon, copter, dart, ball, drip, fish, cobweb, shredder, toaster, microwave) and every path by which it can hurt the player |
| [`format-decisions.md`](analysis/format-decisions.md) | 2,602 | **Normative.** D1-D7: struct sizes, alignment model, `Sampler`'s trailing bytes, reserved-field handling, the `nRooms` bound, retail build identity, demo termination |
| [`graphics-assets.md`](analysis/graphics-assets.md) | 2,546 | How the art is stored (`PICT`/`clut`/`CURS`/icons), the 4 alpha strategies, the 116-row sprite atlas, and the runnable 4-stage extraction pipeline |
| [`house-format.md`](analysis/house-format.md) | 4,233 | **Byte-exact** on-disk house spec: 866-byte header, 348-byte rooms, 12-byte objects, BinHex 4.0 container, link encoding |
| [`houses-inventory.md`](analysis/houses-inventory.md) | 1,529 | Machine-generated inventory of all 22 shipped houses (sizes, room counts, flags, CRCs, resource forks) |
| [`input.md`](analysis/input.md) | 3,169 | Every input path: the `GetKeys` KeyMap model, default and player-2 key maps, remapping, prefs storage, the recorded demo stream, editor keys, mouse handling |
| [`interactions.md`](analysis/interactions.md) | 4,842 | The collision layer: `hotSpots[]`, the 28 action codes, `SectGlider`, room-boundary clamping, everything that kills you |
| [`object-draw-all.md`](analysis/object-draw-all.md) | 4,330 | `DrawARoomsObjects` alone: object draw order, dynamic-object registration, hot-spot emission, budget exhaustion |
| [`object-dynamics.md`](analysis/object-dynamics.md) | 4,137 | Per-frame runtime behaviour of every non-enemy dynamic object: blowers, transports, switches, lights, appliances, triggers |
| [`object-taxonomy.md`](analysis/object-taxonomy.md) | 3,072 | Canonical per-type reference: ID, union variant, field-by-field meaning, and a census across all 22 houses |
| [`original-houses.md`](analysis/original-houses.md) | 4,951 | Design analysis of the 22 shipped houses: object mix, difficulty, link topology, provenance |
| [`player-physics.md`](analysis/player-physics.md) | 3,869 | The player entity: the 24-mode state machine, `MoveGlider`'s integrator, every physics constant, `gliderType` layout |
| [`progression.md`](analysis/progression.md) | 5,356 | Room addressing, link codec, the play-session lifecycle, room transitions, the two-player protocol, death/respawn, win/lose, persistence map |
| [`quicktime-movies.md`](analysis/quicktime-movies.md) | 2,667 | The 15 per-house `.mov` files that play on `kTV` screens: container parsing, the `rle `/`raw `/`smc ` codecs, the 14 headerless movies |
| [`rendering.md`](analysis/rendering.md) | 8,401 | The three-surface compositor, the complete geometry constant table, all offscreen GWorlds, masking strategies, dirty rects, `RenderFrame` order, scoreboard, set pieces, marquee, palette |
| [`resource-fork.md`](analysis/resource-fork.md) | 4,025 | Complete inventory of the 538 resources in `Glider PRO.r` across 35 types, plus how to parse the DeRez dump |
| [`scoring.md`](analysis/scoring.md) | 5,777 | Points, lives, the four consumables, the scoreboard's pixel-exact geometry, high scores, game-over/win sequences |
| [`structs.md`](analysis/structs.md) | 4,525 | Byte-level reference for every in-memory and on-disk struct, with offsets under both alignment models |
| [`toolbox-primitives.md`](analysis/toolbox-primitives.md) | 3,123 | **Normative.** Rulings on `Random()`, `srcXor` on 8-bit, the 4-bit grayscale CLUT, and font substitution |
| [`ui-dialogs.md`](analysis/ui-dialogs.md) | 6,852 | The whole shell: menu tree, all 28 `DLOG`/26 `ALRT`/`DITL` dumps, the 226-byte prefs record, house selection, saved games, high-score screens |
| [`unresolved-format-decisions.md`](analysis/unresolved-format-decisions.md) | 325 | Redirect index: which sibling open questions are actually closed, where, and the 8 that genuinely remain |

---

## 2. Identity, provenance and build configuration

| Fact | Value | Citation |
|---|---|---|
| Product version | **1.1.2**, BCD `01 12`, stage `0x80` (final) | `'vers'` 1 and 2; `resource-fork.md` §3.9.4 |
| "1.0.4" | A stale comment. `grep -c '1\.0\.4'` over `Glider PRO.r` returns **0**; `1\.1\.2` returns 3 | `GliderPRO/Sources/Main.c:3`; `format-decisions.md` D6 |
| Copyright strings | `DITL` 150 says `(c) 1994-2000`; `'vers'` says `(c) 1994-95`. The About box shows **both** | `Glider PRO.r:5178-5180`, `:6135-6140`; `About.c:55-61` |
| What this drop actually is | A later **Carbon work-in-progress** that could not link as retail | `format-decisions.md` D6 |
| `BUILD_ARCADE_VERSION` in this drop | `1` — but **retail shipped with 0**. Treat arcade paths as an optional kiosk mode | `GliderPRO/Headers/GliderDefines.h:16`; `format-decisions.md` D6; correction in `unresolved-format-decisions.md` §4.2 |
| `COMPILENOCP` | Defined — copy protection (`ValidInstallation`) is compiled out; `copyGood = true` unconditionally | `GliderDefines.h:14`; `Main.c` step 9 |
| `COMPILEDEMO` | **Commented out** (`//#define COMPILEDEMO`) — the two demo asserts (`byteCount != 16526L`, `numberRooms != 45`) are compiled out, so this build has no demo restrictions | `GliderDefines.h:12`; dead code at `HouseIO.c:348-351`, `:381-384` |
| Host requirements enforced at boot | Colour QuickDraw, System 7, ≥1 screen at 16/256 colours; otherwise `RedAlert` and exit | `Main.c:284-370`; `ui-dialogs.md` §2.2 |
| Preferred colour depth | `kPreferredDepth` = **8** (256-colour indexed) | `GliderPRO/Headers/Externs.h:15` |

---

## 3. Geometry: screen, room, and the 3x3 neighbourhood

Authority: `rendering.md` §2 (complete constant table + worked examples), `progression.md` §1.2-1.3.

### 3.1 The fixed sizes

| Constant | Value | Line in `GliderDefines.h` | Meaning |
|---|---:|---|---|
| `kNumTiles` | 8 | :496 | Vertical tile strips per room |
| `kTileWide` | 64 | :497 | Width of one tile strip |
| `kTileHigh` | 322 | :498 | Height of a tile strip = **height of a room** |
| `kRoomWide` | 512 | :499 | `kNumTiles * kTileWide` = **width of a room** |
| `kVertLocalOffset` | 322 | :501 | Vertical spacing between stacked rooms (== `kTileHigh`) |
| `kFloorSupportTall` | 44 | :500 | Between-floors support band |
| `kScoreboardTall` | 20 | :515 | Scoreboard band height |
| `kMaxViewWidth` | 1536 | :267 | Hard cap on `houseRect` width (= 3 rooms) |
| `kMaxViewHeight` | 1026 | :268 | Hard cap on `houseRect` height |
| `kGliderWide` / `kGliderHigh` | 48 / 20 | :548-549 | Glider sprite |
| `kHalfGliderWide` | 24 | :550 | |
| `kGliderBurningHigh` | 26 | :551 | Burning frames are taller |
| `kShadowHigh` / `kShadowTop` | 9 / 306 | :552-553 | Shadow sprite and its fixed room-space y |
| `kMapRoomWidth` / `kMapRoomHeight` | 32 / 20 | :247, :246 | Editor map thumbnails |

**A room is always 512 x 322 device pixels. There is no scaling anywhere in the room rendering
path** (`rendering.md` §2.6). A larger screen shows *more of the neighbouring rooms*, never a
bigger room. The only scaling in the program is `LoadScaledGraphic`
(`GliderPRO/Sources/Utilities.c:340`), used for the splash, star field, banner and map
thumbnails.

### 3.2 Derived window geometry

`VariableInit` (`GliderPRO/Sources/InterfaceInit.c:196-218`):

```c
houseRect = thisMac.screen;
houseRect.bottom -= kScoreboardTall;             /* -20, the SCOREBOARD's 20 — not the menu bar's */
if (houseRect.right  > kMaxViewWidth)  houseRect.right  = kMaxViewWidth;  /* 1536 */
if (houseRect.bottom > kMaxViewHeight) houseRect.bottom = kMaxViewHeight; /* 1026 */
playOriginH = (RectWide(&thisMac.screen) - kRoomWide) / 2;   /* (W - 512) / 2 */
playOriginV = (RectTall(&thisMac.screen) - kTileHigh) / 2;   /* (H - 322) / 2 */
```

Non-obvious consequences a port must reproduce:

* `playOrigin*` derive from **`thisMac.screen`, not `houseRect`**, so the central room is *not*
  vertically centred in the offscreen surface — it sits 10 px lower than centre. On 640x480,
  `playOriginV = 79` where centring in the 460-tall work map would give 69.
  `rendering.md` §2.4.
* `houseRect` is **not** zero-cornered; it keeps `thisMac.screen`'s origin (non-zero on a
  secondary monitor). `workSrcRect` and `backSrcRect` are zero-cornered copies
  (`GliderPRO/Sources/StructuresInit2.c:150-160`).
* On screens taller than 1026 the `-= 20` / clamp identity breaks and the window becomes taller
  than the offscreen surface. `rendering.md` §2.3 note 2.
* The *window* loses 20 px to the **menu bar** (hard-coded 20 at `MainWindow.c:224`, `:228-229`,
  `Play.c:144`) while the *offscreen* loses 20 px to the **scoreboard** (`kScoreboardTall`,
  `GliderDefines.h:515`). **These two 20s are numerically equal but semantically unrelated; a
  port must not conflate them** (`rendering.md` §2.2). They agree by construction only on
  screens ≤ 1026 px tall.

The nine 3x3 destination rects (`InterfaceInit.c:206-218`) tile a 1536 x 966 area whose centre
cell is at `(playOriginH, playOriginV)`, offset by `±kRoomWide` horizontally and
`±kVertLocalOffset` vertically. `numNeighbors` selects 9 (all), 3 (central + west + east), or 1
(central only). Default 9 (`GliderPRO/Sources/Settings.c:888`, `:1257`); **forced to 1 when
`thisMac.screen.right <= 512`** (`GliderPRO/Sources/Main.c:191-192`).

### 3.3 Worked example: 640 x 480, 8-bit, `numNeighbors == 9`

| Quantity | Value |
|---|---|
| `thisMac.screen` | `(t 0, l 0, b 480, r 640)` |
| `houseRect` = `workSrcRect` = `backSrcRect` = initial `justRoomsRect` | `(0, 0, 460, 640)` |
| `mainWindowRect`, window position on screen | `(0, 0, 460, 640)` at `(0, 20)` |
| `playOriginH`, `playOriginV` | 64, 79 |
| `localRoomsDest[kCentralRoom]` | `(t 79, l 64, b 401, r 576)` |
| `localRoomsDest[kNorth]` / `[kSouth]` | `(-243, 64, 79, 576)` / `(401, 64, 723, 576)` |
| `localRoomsDest[kWest]` / `[kEast]` | `(79, -448, 401, 64)` / `(79, 576, 401, 1088)` |
| Central room on the physical screen | y 99..421, x 64..576 |
| `boardSrcRect`, initial `boardDestRect` | `(0, 0, 20, 640)`, `(-20, 0, 0, 640)` |

Room-space `(0,0)` lands at work-map `(64, 79)` and physical screen `(64, 99)`.
`rendering.md` §2.7; 832x624 and 512x342 in §2.8-2.9.

### 3.4 Windows and the full-screen trick

| `WIND` | bounds (t,l,b,r) | procID | Used for |
|---|---|---|---|
| 128 `kMainWindowID` | 0,0,384,512 | 2 = `plainDBox` | splash / play window |
| 129 `kEditWindowID` | 0,0,322,512 | 4 = `noGrowDocProc` | editor window (has a drag bar) |
| 130 `kMenuWindowID` | 0,0,20,640 | 2 = `plainDBox` | **20-px strip that covers the menu bar** |

`GliderPRO/Sources/MainWindow.c:16-18`, `:174-265`; `ui-dialogs.md` §2.3-2.4. Full-screen is
achieved by parking a plain 20-px window over the menu bar, then sizing the main window to
`screen` minus 20 and moving it to `(screen.left, screen.top + 20)`.

Splash: `splashOriginH = (screenW - 640)/2` clamped to ≥0, `splashOriginV = (screenH - 480)/2`
clamped to ≥0 (`MainWindow.c:238-243`). `kSplash8BitPICT` = 1000 (`GliderDefines.h:524`)
decodes to **640 x 460**, not 640 x 480 — the missing 20 px is where the scoreboard would go.
`PICT` 1021 is a second unreferenced 640 x 460 splash.

### 3.5 The scoreboard band (20 px, above the play area)

`scoring.md` §5.2 is the pixel-exact authority. Derivation:

```
W            = RectWide(&houseRect) = min(screenW, 1536)
boardSrcRect = (l 0, t 0, r W, b 20)
boardHOffset = (W >= 640) ? (W - 1536)/2 : -576      /* PICT 1997 draw x */
boardDestRect= boardSrcRect offset (0, -20)          /* i.e. above the window origin */
textHOffset  = (W - 640)/2 ; if < 0 then -128        /* call it t */
```

C integer division truncates **toward zero**, so `(640-1536)/2 == -448` but
`(641-1536)/2 == -447`. Go's `/` matches; a `>>1` or Python `//` does not. For every even
`W >= 640` the invariant `boardHOffset == textHOffset - 448` holds exactly.

| Element | x (plate-relative, `t` added) | y |
|---|---|---|
| Room-name text pad `boardTSrcMap` 256x12 | 137+t .. 392+t | 5..16 |
| Foil badge | 432+t .. 447+t | 2..17 |
| Rubber-band badge | 449+t .. 464+t | 2..17 |
| Battery **or** helium badge (**shared slot**) | 467+t .. 482+t | 1..17 |
| Glider-count pad `boardGSrcMap` 20x10 | 526+t .. 545+t | 5..14 |
| Score pad `boardPSrcMap` 64x10 | 570+t .. 633+t | 5..14 |

Artwork on the plate (`PICT` 1997, `picFrame (0,0,20,1536)`, 8 bpp): "Glider" wordmark 34..85,
gold glyph 88..94, divider bar 126..129, paper-glider icon 491..524, star icon 552..568 (all
plate-relative, including the 1-px drop shadow at palette index 253). Badge sheet is
`PICT` 1996 into a 32 x 66 GWorld. `scoring.md` §5.3-5.4.

Two vertical modes: `kScoreboardHigh` 0 (overlay at the top of the work map) and
`kScoreboardLow` 1 (directly above the central room), switched by `AdjustScoreboardHeight`,
which also narrows `justRoomsRect`. `scoring.md` §5.8, `rendering.md` §13.10.

---

## 4. The tick / frame model

Authority: `architecture.md` §7, `determinism.md` §1, `progression.md` §18.

| Fact | Value |
|---|---|
| Tick source | `TickCount()`, 1 tick = 1/60.15 s |
| `kTicksPerFrame` | **2** (`GliderDefines.h:533`) |
| Frame rate | 60.15 / 2 = **30.07 fps** (call it 30 Hz) |
| Delta time | **Does not exist.** Every velocity, gravity value and timer is an integer *per frame* |
| Pacing | Busy-wait, **no catch-up**, deadline recomputed from "now" |

The pacer, verbatim, is the tail of `RenderFrame` (`GliderPRO/Sources/Render.c:639-671`):

```c
void RenderFrame (void) {
    if (hasMirror) { DrawReflection(&theGlider, true);
                     if (twoPlayerGame) DrawReflection(&theGlider2, false); }
    HandleGrease();  RenderPendulums();
    if (evenFrame) RenderFlames(); else RenderStars();
    RenderDynamics(); RenderFlyingPoints(); RenderSparkles();
    RenderGlider(&theGlider, true);
    if (twoPlayerGame) RenderGlider(&theGlider2, false);
    RenderShreds(); RenderBands();
    while (TickCount() < nextFrame) { }        /* spin */
    nextFrame = TickCount() + kTicksPerFrame;  /* recomputed from NOW, not += */
    CopyRectsQD();                             /* present */
    numWork2Main = 0;  numBack2Work = 0;       /* then clear the queues */
}
```

Consequences that are load-bearing:

* **A slow frame is simply a slow frame.** Because `nextFrame` is recomputed from `TickCount()`
  rather than advanced by 2, the game never runs a double step and never drops one. Under load
  the whole simulation runs slower in wall-clock terms and remains frame-exact.
* `kTicksPerFrame` appears in the *simulation* only as a compile-time divisor converting an
  author's tick delay into frames (`Dynamics3.c:316`, `:401`, `:426`, `:456`, `:504`, `:521`) —
  constant-folded, therefore harmless. `determinism.md` porting note 3.
* `evenFrame` gates half-rate behaviour: flames render on even frames, stars on odd.
* **Present happens before erase.** `CopyRectsQD` runs, *then* both dirty-rect counters are
  zeroed.

Everything is measured in frames: 16-frame game-over countdown, 16-frame fade
(`kLastFadeSequence`), `kShredderCountdown` -68, `kLengthOfZap` 30, battery/helium spent one
unit per frame held. `progression.md` §18.2 has the full table.

Idle / attract mode: `kIdleSplashTicks` = `7200L` = **120 s** (`GliderDefines.h:197`), 9 live
assignment sites, triggering `DoDemoGame()` from `Events.c:536-539` when
`theMode == kSplashMode && doAutoDemo && !switchedOut`. `idleMode` itself is vestigial
(written once at `InterfaceInit.c:162`, never read).
`unresolved-format-decisions.md` §4.1 — this corrects `architecture.md` OQ10.

---

## 5. The rendering pipeline

Authority: `rendering.md` (whole document), `object-draw-all.md` for object order.

### 5.1 Three surfaces, two queues

| Surface | Size | Role |
|---|---|---|
| `backSrcMap` | `houseRect` zero-cornered | **static** room art: backgrounds, tiles, floor supports, non-animated objects |
| `workSrcMap` | same | per-frame composite: `backSrcMap` content + all moving things |
| `mainWindow` | same | what the user sees |

Plus per-class sprite GWorlds (`blowerSrcMap`, `furnitureSrcMap`, `bonusSrcMap`,
`switchSrcMap`, `lightSrcMap`, `applianceSrcMap`, `transSrcMap`, `balloonSrcMap`,
`copterSrcMap`, `dartSrcMap`, `ballSrcMap`, `dripSrcMap`, `enemySrcMap`, `clutterSrcMap`, plus
`gliderSrcMap` family, `shadowSrcMap`, `suppSrcMap`, `badgeSrcMap`, `boardSrcMap` and four
scoreboard scratch pads) — full inventory with sizes and `PICT` IDs at `rendering.md` §3.

Two dirty-rect queues drive the frame (`rendering.md` §9):

* `back2WorkRects` — "erase": copy this rect from `backSrcMap` to `workSrcMap`. Clamped to
  `workSrcRect`.
* `work2MainRects` — "present": copy this rect from `workSrcMap` to `mainWindow`. Clamped to
  `justRoomsRect`.

`CopyRectsQD` performs both. The order in `RenderFrame` is: draw everything into `workSrcMap`,
spin to the deadline, present, then reset both counters. Note that the *erase* for frame N+1 is
enqueued during frame N's drawing.

Three traps for a porter (`rendering.md` §9.1-9.3):

* Both queues are fixed `Rect[kMaxGarbageRects]` arrays with `kMaxGarbageRects` = **48**
  (`Render.c:20`), never coalesced and never sorted; but the guards are
  `< kMaxGarbageRects - 1` (`Render.c:67`, `:86`), so at most **47** entries are ever stored and
  any further rect that frame is **silently dropped**.
* The clamps are `if / else if`, not two independent `if`s (`Render.c:69-76`, `:88-95`), so a
  rect wider (or taller) than the clip box has only **one** edge clamped.
* Clamping `work2MainRects` to `justRoomsRect` is the *sole* mechanism preventing gameplay blits
  from overwriting the scoreboard.

### 5.2 Four masking strategies

`rendering.md` §8, `graphics-assets.md` §5:

| Strategy | Mechanism | Used by |
|---|---|---|
| A | Pre-built 1-bit mask GWorld, `CopyMask`, no allocation | 13 of 14 sprite sheets, 8 paired objects |
| B | On-demand temp GWorld pair, `CopyMask`, dispose | some object draws |
| C | White colour key via QuickDraw `transparent` mode | 21 colour-keyed objects |
| D | Plain opaque `srcCopy` | the `switch` sheet, backgrounds, scoreboard, 9 objects |

Mask resources live at **`sprite PICT id + 1000`** (`rendering.md` §4.6). For the sheet range
3998-4018 that means 4998-5018, **except 4003 which has no 5003** — the `switch` sheet is
opaque and `switchMaskMap` is not even declared (`GliderPRO/Headers/Objects.h:18-19`).
Two further documented exceptions to the `+1000` convention: the **angel** sheet's mask is
`1019 + 1 = 1020`, *not* 2019 (`StructuresInit2.c:130`, `:134`), and the **floor-support** strip
`1999` has **no mask at all** — it is drawn opaque (`StructuresInit2.c:109`).
`kMirror`, `kCounter` and `kWallWindow` are drawn **procedurally** in C and have no art at all.

Where a sheet has a mask, **the mask wins**: bit value 1 (a black pixel in the 1-bit PICT) means
opaque, and the mask carries information the colour plane does not — 681 pixels of the angel
`PICT` 1019 are white *and* mask-opaque (the robe and wings), so a colour-key blit would punch
681 holes in it (`graphics-assets.md` §5, measurement at `:952`, rule at `:961`). **Never
substitute a white colour key for a mask.** Only the colour-key path (strategy C, the 21
colour-keyed objects = 20 fixed-ID sans-white types + `kCustomPict`) decides alpha on a palette
index — white is `index == 0`; `graphics-assets.md` §7.4 — and do not substitute an RGB test for
that index test either.

### 5.3 Room background composition

`DrawLocale` (`rendering.md` §6.2) rebuilds `backSrcMap` for the whole 3x3 neighbourhood on
every room change. Per room, `DrawRoomBackground` decodes the whole 512 x 322 background `PICT`
(`kBaseBackgroundID` 2000 .. 2017, 18 built-ins; user art at `kUserBackground` 3000+, which in a
house file is fetched as resource type **`'Date'`, not `'PICT'`** — `LoadGraphicSpecial`,
`RoomGraphics.c:134-158`, `:142`; `rendering.md` §6.5 and §20 item 19) into **`workSrcMap` as
pure scratch** (`RoomGraphics.c:238-239`) and then blits the 8 tile columns from `workSrcMap`
into `backSrcMap` — **the background is never blitted as a single piece** — followed by that
room's non-animated objects (`DrawARoomsObjects`).

Two orderings a port must not rearrange (`rendering.md` §6.2, `:1795-1805`):

* The **central room is drawn last** of the nine (`RoomGraphics.c:118-121`), so a neighbour's
  overhanging objects can never cover the playfield.
* The **floor-support band is drawn last of all** — `DrawFloorSupport()` is called only at
  `RoomGraphics.c:123-124`, after all nine rooms *and* all their objects, and only when
  `numNeighbors > 3` — so it overlays the ceilings and floors of the rooms above and below.

The **tile mechanism** is a column index, not a tile ID (`RoomGraphics.c:241-251`):

```c
QSetRect(&src, 0, 0, kTileWide, kTileHigh);
src.left  = tiles[i] * kTileWide;    /* tiles[] are COLUMN INDICES into the background PICT */
src.right = src.left + kTileWide;
QOffsetRect(&dest, kTileWide, 0);
```

So `roomType.tiles[8]` selects, for each of the 8 columns of the room, which 64-px column of
the 512-px background art to use.

Tile *values* also carry collision semantics, but **only in `kRoof` rooms** — and `kRoof` is
background `PICT` **2014** (`GliderDefines.h:241`), not a tile code. `CheckRoofCollision`
(`Interactions.c:450-505`, reached only from `Interactions.c:722` when
`thisBackground == kRoof`) is an `if / else if` chain with sloped-roof line equations for
`tiles[i]` values **1, 2, 5 and 6** and a final `else` that kills the glider — which is why tile
value **4**, present in real roof rooms, is fatal (see §20 row 7). Tile values are also read for
the `kDirt` ceiling/floor escapes (`Interactions.c:263`, `:396`). `constants.md` §5626-5673.

`DrawLighting` is a **no-op stub** (`rendering.md` §6.9, `progression.md` §15.5); `numLights`
therefore barely matters. `RedrawRoomLighting` is the only partial background rebuild.

### 5.4 Room transitions

Three exist, **two are reachable** (`rendering.md` §14). All three read `workSrcMap` and write
the window port with `srcRect == dstRect`; none of them touches `backSrcMap`.

| Transition | Status |
|---|---|
| `DumpScreenOn` | hard cut — **live**: the initial reveal at the start of every game, `Play.c:172`, `:177`, `:181`, always with `&justRoomsRect` |
| `WipeScreenOn` | 4-px band sweep (`kWipeRectThick` 4) — **the only *animated* transition the player ever sees**, `Transit.c:300`, `:338`, `:377`, `:416` |
| `PourScreenOn` | dead code, called from nowhere, documented |
| four dissolves | commented out |

---

## 6. The player: state machine and physics

Authority: `player-physics.md` (whole doc). This is the single most fidelity-critical
subsystem.

### 6.1 The 24 modes

`GliderDefines.h:571-594`:

| # | Mode | # | Mode |
|---:|---|---:|---|
| 0 | `kGliderNormal` | 12 | `kGliderDuctingUp` |
| 1 | `kGliderFadingIn` | 13 | `kGliderDuctingIn` |
| 2 | `kGliderFadingOut` | 14 | `kGliderMailInLeft` |
| 3 | `kGliderGoingUp` | 15 | `kGliderMailOutLeft` |
| 4 | `kGliderComingUp` | 16 | `kGliderMailInRight` |
| 5 | `kGliderGoingDown` | 17 | `kGliderMailOutRight` |
| 6 | `kGliderComingDown` | 18 | `kGliderGoingFoil` |
| 7 | `kGliderFaceLeft` | 19 | `kGliderLosingFoil` |
| 8 | `kGliderFaceRight` | 20 | `kGliderShredding` |
| 9 | `kGliderBurning` | 21 | `kGliderInLimbo` |
| 10 | `kGliderTransporting` | 22 | `kGliderIdle` |
| 11 | `kGliderDuctingDown` | 23 | `kGliderTransportingIn` |

Two traps: the single `gliderType.frame` field is **reused for five different meanings**
depending on mode (fade index, about-face index, shred countdown, transport countdown, duct
progress), and `wasMode` carries **four** distinct meanings. `player-physics.md` §7 has the
per-mode table: which fields are live, what `frame` counts, what the exit condition is, and
which renderer draws it.

Related constants: `kFirstAboutFaceFrame` 18, `kLastAboutFaceFrame` 20, `kLeftFadeOffset` 7,
`kLastFadeSequence` 16, `kNumGliderSrcRects` 31, `kNumShadowSrcRects` 2, `kWasBurning` 2,
`kGliderStartsDown` 32, `kShredderCountdown` -68, `kLengthOfZap` 30
(`GliderDefines.h:546-569`, except `kShredderCountdown` -68 which is a file-local `#define` in
`Player.c:17`, used only at `Player.c:1303`).

### 6.2 The integrator

`MoveGlider` (`GliderPRO/Sources/Player.c:64-147`) is the entire physics model. Structure:

```c
/* 1. ramp actual velocity toward desired, then RESET desired */
if (hVel > hDesiredVel) { hVel -= kHImpulse; if (hVel < hDesiredVel) hVel = hDesiredVel; }
else if (hVel < hDesiredVel) { hVel += kHImpulse; if (hVel > hDesiredVel) hVel = hDesiredVel; }
hDesiredVel = 0;
/* ...identical block vertically... */
vDesiredVel = kGravity;         /* 3, every single frame */
/* 2. move: BOTH blocks are `if (vel < 0) {...} else {...}` -- never `else if (vel > 0)`.
      The +/-kMaxHVel clamp is NOT a separate step: it is the first line of each
      horizontal branch (clamp to -16 in the `< 0` arm at Player.c:96-97, to +16 in
      the `else` arm at :113-114), and `wasHVel` is then written from the already-
      clamped value (:99, :116). There is no vertical clamp at all. */
```

| Constant | Value | Citation | Effect |
|---|---:|---|---|
| `kGravity` | 3 | `Player.c:13` | `vDesiredVel` reset target every frame |
| `kHImpulse` | 2 | `Player.c:14` | horizontal ramp rate, px/frame² |
| `kVImpulse` | 2 | `Player.c:15` | vertical ramp rate |
| `kMaxHVel` | 16 | `Player.c:16` | horizontal clamp, ±. **There is no vertical clamp** |
| `kNormalThrust` | 5 | `Input.c:14` | left/right sets `hDesiredVel = ±5` |
| `kHyperThrust` | 8 | `Input.c:15` | battery adds **directly** to `hVel`, bypassing the ramp |
| `kHeliumLift` | 4 | `Input.c:16` | helium sets `vDesiredVel = -4` |
| `kFloorVentLift` | -6 | `Interactions.c:13` | |
| `kCeilingVentDrop` | 8 | `Interactions.c:14` | |
| `kFanStrength` | 12 | `Interactions.c:15` | uses `+=`, so **fans stack with key input** |
| `kShoveVelocity` | 8 | | dynamic-object shove |
| `kRubberBandVelocity` | 20 | `RubberBands.c:12` | |

Derived speeds a porter should sanity-check against: terminal fall **3 px/frame** (90 px/s,
emergent because `vDesiredVel` is re-set to exactly `kGravity` each frame and the ramp
converges); cruise **5 px/frame** (150 px/s); battery **16 px/frame** (480 px/s); helium rise
**4 px/frame**. From rest, holding Right ramps `hVel` 0 → 2 → 4 → 5 over three frames (100 ms).

**The three most-missed details:**

1. Because `hDesiredVel` is zeroed every frame, the `+=` in `Input.c:313` behaves as `=` for
   player input; the `+=` form exists so input can compose with other same-frame writers
   (`Dynamics.c:54-58`). `input.md` §4.5.
2. Both move blocks are `if (vel < 0) {…} else {…}`, so a **zero-velocity axis takes the
   positive branch**, and `wasHVel`/`wasVVel` and all four edges of `whole` are rewritten on
   every call regardless of motion. `player-physics.md` §3.
3. `wasVVel` is written (`Player.c:131`, `:140`) and **never read** anywhere in the tree;
   `wasHVel` *is* read by `GliderHitTop` (`Interactions.c:65`, `:66`, `:86`, `:87`).

`gliderType` is **110 bytes packed** / 112 natural and needs a 4-byte `long`; full offset table
at `player-physics.md` §2 ("The `gliderType` record") and `structs.md`.

### 6.3 Per-frame order inside `PlayGame`

`GliderPRO/Sources/Play.c:430-599` — the authoritative order, reproduced in full at
`player-physics.md` §17 ("The authoritative per-frame update order") and `progression.md` §18.3.
The two-player and one-player arms of `PlayGame` are structurally identical; both are listed:

1. `HandleTelephone()` (`Play.c:445`).
2. `HandleDynamics()` (`Play.c:449` two-player / `:475` one-player) — **before input**, so
   enemies and dynamic objects move against *last* frame's glider position.
3. If `!gameOver`: `GetInput(&theGlider)` (`:452` / `:481`; `GetDemoInput(&theGlider)` at `:479`
   when `demoGoing`) then, if two-player, `GetInput(&theGlider2)` (`:453`) — **player 1 first,
   always**. Both read the same `KeyMap` snapshot.
4. `HandleInteraction()` (`:454` / `:482`), still inside the same `!gameOver` guard — one call
   resolving **both** gliders against shared mutable room state (`Interactions.c:1691-1711`):
   one hot-spot loop tests `theGlider` at `:1639` and `theGlider2` at `:1657` with a shared
   `stillOver` latch; prize pickup is first-caller-wins in `SetObjectState`
   (`Objects.c:460-462`); then `CheckGliderInRoom(&theGlider)` at `:1705`, then `(&theGlider2)`
   at `:1706` — a full room rebuild can happen between them.
5. `HandleTriggers()` (`:456` / `:484`) then `HandleBands()` (`:457` / `:485`) — **after** the
   interaction pass, and **not** guarded by `!gameOver`.
6. If `!gameOver`: `HandleGlider(&theGlider)` (`:460` / `:487`) then `HandleGlider(&theGlider2)`
   (`:461`) — this is where `MoveGlider` is actually invoked.
7. If `playing`: `MoviesTask` (`:467` / `:492`, `COMPILEQT`), `RenderFrame()` (`:469` / `:494` —
   contains the pacer), `HandleDynamicScoreboard()` (`:470` / `:495`).
8. If `gameOver` (`:499`): `countDown--` (`:501`); at `<= 0`,
   `mortals < 0 ? DoDiedGameOver() : DoGameOver()` (`:546-549`).

**Preserve player-1-first ordering everywhere, and keep dynamics before input and `HandleGlider`
after `HandleBands`.** `determinism.md` porting notes 1 and 4.

---

## 7. Input

Authority: `input.md` (whole doc).

### 7.1 Model

During play, input is **polled level state**, never events: one `GetKeys(theKeys)` per frame
into a single file-scope 16-byte `KeyMap` (`Input.c:31`), refreshed only when processing
player 1 (`Input.c:283-288`). Consequences: no key repeat (holding Left thrusts every frame);
no edge information (the one edge-triggered action, firing a band, is synthesised by the
`fireHeld` latch — `input.md` §9.1); modifier keys are ordinary bits; the map is indexed by
**physical ADB scan code**, so it is keyboard-layout independent.

`BitTst` numbers bits **MSB-first from the first byte**; the raw-virtual-key ↔ offset transform
is `KeyMapOffsetFromRawKey` with a verified table at `input.md` §3.4.

By default `doBackground = false` (`Main.c:186`), so during a game the app services **zero**
events: no window updates, no menus, no Apple events, no cursor tracking (`Play.c:437-443`).

### 7.2 Default key map — player 1 (remappable)

| Action | Default key | KeyMap offset | Raw VK | `gliderType` field | Prefs offset |
|---|---|---:|---|---|---|
| Left | Left arrow | 124 | 0x7B | `leftKey` | +146 |
| Right | Right arrow | 123 | 0x7C | `rightKey` | +150 |
| Battery / Helium | Down arrow | 122 | 0x7D | `battKey` | +154 |
| Rubber bands | Up arrow | 121 | 0x7E | `bandKey` | +158 |

Established at `Main.c:135-138`, `:184`, and again by the prefs "Defaults" buttons
(`Settings.c:335-343`, `:1233-1241`) — the two must agree.

### 7.3 Player 2 — hard-coded, not remappable

`InterfaceInit.c:147-152`: `leftKey = kControlKeyMap` (60 / 0x3B), `rightKey = kCommandKeyMap`
(48 / 0x37), `battKey = kOptionKeyMap` (61 / 0x3A), `bandKey = kShiftKeyMap` (63 / 0x38),
`which = kPlayer2`. Never overwritten, no UI, not persisted.

Because player 2's "right" is Command, and player 1's `DoCommandKey` (`Input.c:286`) fires on
the same bit, this matters — but **Command-Q is not gated**: `Input.c:54-57` sets
`playing = false; paused = false;` unconditionally, so Command-Q always ends the game, including
in a two-player game where player 2's "right" key *is* Command. Only the *save-first offer*
(`Input.c:59`) and Command-S (`Input.c:65`) carry `!twoPlayerGame`. So a two-player game **can**
be quit from the keyboard but can never be saved. (`input.md` §11 and porting note 11;
`input.md` §1.2's summary sentence says the quit is gated and is **wrong** — do not follow it.)

### 7.4 Fixed system keys

| Key | Action |
|---|---|
| Tab (default) **or** Esc | Pause, per `prefsInfo.wasEscPauseKey` @ +218; placards are `PICT` 1016 / 1015 |
| Command-Q | Quit — **always**, ungated (`Input.c:54-57`). The `ALRT` 1041 "save first" offer is the only one-player-only part (`Input.c:59`) |
| Command-S | Save game in place (one-player only, `Input.c:65`) |
| Delete | Two-player only, from **player 1's keyboard only**, and only while the other player is already in limbo. All four conditions must hold: `otherPlayerEscaped != kNoOneEscaped`, Delete down, `thisGlider->which` (`kPlayer1 == TRUE`), `!onePlayerLeft` (`Input.c:366-368`). `ForceKillGlider` (`Transit.c:449-469`) then fades out whichever glider is *not* in `kGliderInLimbo` and plays `kFadeOutSound`. Level-triggered and unlatched; repeats are suppressed by the `mode != kGliderFadingOut` test |

### 7.5 Emergent fifth action

Holding **Left and Right simultaneously** performs an about-face rather than applying thrust
(`Input.c:306-310`), driving modes 18-20 of the about-face animation.

### 7.6 Prefs

226 bytes in this build (`prefsInfo`), **234 in retail** with `prefVersion` at +172 instead of
+164, because `long encrypted, fakeLong` is commented out of `Externs.h:240` in this drop.
Full field map at `ui-dialogs.md` §6.1 and `input.md` §21.2; the retail difference is
`format-decisions.md` D6. Key display names are persisted, not recomputed
(`input.md` §20.4).

### 7.7 The recorded demo

`'demo'` 128 is **6702 bytes = 1117 six-byte `demoType` records**, frames 46..3414
(`format-decisions.md` D7). Key codes are **0 = right, 1 = left, 2 = battery/helium,
3 = rubber band** — the source comments on cases 0 and 1 in `GetDemoInput` are **swapped**; the
recorder call sites are authoritative because they produced the shipped bytes (`LogDemoKey(0)`
in the rightKey branch at `Input.c:304`, `(1)` in leftKey at `:321`). Playback is terminated by
the *game* (death → `countDown` 16 → `DoDiedGameOver` → `playing = false`), not by the stream,
and the original reads one record **past the end** — benign on Mac `NewPtr` slack, a panic in
Go. A port must add `demoIndex < len(demo)`.

`DoDemoGame` (`Play.c:282-303`) sets `thisHouseIndex = demoHouseIndex`, which is **-1 when no
house named `Demo House` is in the search path** — an unguarded negative index a port must
guard (`unresolved-format-decisions.md` §4.1).

---

## 8. Objects: the complete class / type table

Authority: `object-taxonomy.md` (per-type reference + 22-house census), `constants.md` §6.

117 defined codes in 9 classes, inside a **144-entry** (`kNumSrcRects` = 0x90) index space. All
from `GliderPRO/Headers/GliderDefines.h:311-437`. Gaps between classes are deliberate padding.

### 8.1 Blowers — 0x01..0x10 (`:311-326`)

| ID | Constant | ID | Constant |
|---|---|---|---|
| 0x01 | `kFloorVent` | 0x09 | `kCandle` |
| 0x02 | `kCeilingVent` | 0x0A | `kStubby` |
| 0x03 | `kFloorBlower` | 0x0B | `kTiki` |
| 0x04 | `kCeilingBlower` | 0x0C | `kBBQ` |
| 0x05 | `kSewerGrate` | 0x0D | `kInvisBlower` |
| 0x06 | `kLeftFan` | 0x0E | `kGrecoVent` |
| 0x07 | `kRightFan` | 0x0F | `kSewerBlower` |
| 0x08 | `kTaper` | 0x10 | `kLiftArea` |

### 8.2 Furniture — 0x11..0x1F (`:328-342`)

| ID | Constant | ID | Constant |
|---|---|---|---|
| 0x11 | `kTable` | 0x19 | `kDeckTable` |
| 0x12 | `kShelf` | 0x1A | `kStool` |
| 0x13 | `kCabinet` | 0x1B | `kTrunk` |
| 0x14 | `kFilingCabinet` | 0x1C | `kInvisObstacle` |
| 0x15 | `kWasteBasket` | 0x1D | `kManhole` |
| 0x16 | `kMilkCrate` | 0x1E | `kBooks` |
| 0x17 | `kCounter` | 0x1F | `kInvisBounce` |
| 0x18 | `kDresser` | | |

### 8.3 Prizes — 0x21..0x2F (`:344-358`)

| ID | Constant | Points | ID | Constant | Points |
|---|---|---:|---|---|---:|
| 0x21 | `kRedClock` | 100 | 0x29 | `kGreaseLf` | 0 |
| 0x22 | `kBlueClock` | 300 | 0x2A | `kFoil` | 0 |
| 0x23 | `kYellowClock` | 500 | 0x2B | `kInvisBonus` | 100/300/500 per-object |
| 0x24 | `kCuckoo` | 1000 | 0x2C | `kStar` | 5000 |
| 0x25 | `kPaper` | 0 (**+1 life**) | 0x2D | `kSparkle` | 0 |
| 0x26 | `kBattery` | 0 | 0x2E | `kHelium` | 0 |
| 0x27 | `kBands` | 0 | 0x2F | `kSlider` | 0 |
| 0x28 | `kGreaseRt` | 0 | | | |

### 8.4 Transport — 0x31..0x40 (`:360-375`)

| ID | Constant | ID | Constant |
|---|---|---|---|
| 0x31 | `kUpStairs` | 0x39 | `kDoorExRt` |
| 0x32 | `kDownStairs` | 0x3A | `kDoorExLf` |
| 0x33 | `kMailboxLf` | 0x3B | `kWindowInLf` |
| 0x34 | `kMailboxRt` | 0x3C | `kWindowInRt` |
| 0x35 | `kFloorTrans` | 0x3D | `kWindowExRt` |
| 0x36 | `kCeilingTrans` | 0x3E | `kWindowExLf` |
| 0x37 | `kDoorInLf` | 0x3F | `kInvisTrans` |
| 0x38 | `kDoorInRt` | 0x40 | `kDeluxeTrans` |

### 8.5 Switches — 0x41..0x49 (`:377-385`)

`0x41 kLightSwitch`, `0x42 kMachineSwitch`, `0x43 kThermostat`, `0x44 kPowerSwitch`,
`0x45 kKnifeSwitch`, `0x46 kInvisSwitch`, `0x47 kTrigger`, `0x48 kLgTrigger`,
`0x49 kSoundTrigger`.

### 8.6 Lights — 0x51..0x58 (`:387-394`)

`0x51 kCeilingLight`, `0x52 kLightBulb`, `0x53 kTableLamp`, `0x54 kHipLamp`, `0x55 kDecoLamp`,
`0x56 kFlourescent`, `0x57 kTrackLight`, `0x58 kInvisLight`.

### 8.7 Appliances — 0x61..0x6E (`:396-409`)

| ID | Constant | ID | Constant |
|---|---|---|---|
| 0x61 | `kShredder` | 0x68 | `kVCR` |
| 0x62 | `kToaster` | 0x69 | `kStereo` |
| 0x63 | `kMacPlus` | 0x6A | `kMicrowave` |
| 0x64 | `kGuitar` | 0x6B | `kCinderBlock` |
| 0x65 | `kTV` | 0x6C | `kFlowerBox` |
| 0x66 | `kCoffee` | 0x6D | `kCDs` |
| 0x67 | `kOutlet` | 0x6E | `kCustomPict` |

### 8.8 Enemies — 0x71..0x79 (`:411-419`)

`0x71 kBalloon`, `0x72 kCopterLf`, `0x73 kCopterRt`, `0x74 kDartLf`, `0x75 kDartRt`,
`0x76 kBall`, `0x77 kDrip`, `0x78 kFish`, `0x79 kCobweb`.
Frame counts: balloon 8, copter 10, dart 4, ball 2, drip 6, fish 8 (`:452-458`).

### 8.9 Clutter — 0x81..0x8F (`:421-435`)

| ID | Constant | ID | Constant |
|---|---|---|---|
| 0x81 | `kOzma` | 0x89 | `kVase1` |
| 0x82 | `kMirror` (procedural) | 0x8A | `kVase2` |
| 0x83 | `kMousehole` | 0x8B | `kBulletin` |
| 0x84 | `kFireplace` | 0x8C | `kCloud` |
| 0x85 | `kFlower` (6 frames) | 0x8D | `kFaucet` |
| 0x86 | `kWallWindow` (procedural) | 0x8E | `kRug` |
| 0x87 | `kBear` | 0x8F | `kChimes` |
| 0x88 | `kCalendar` | | |

### 8.10 The `srcRects[144]` bug a port must decide about

`srcRects` is allocated with `NewPtr` and **not cleared** (`StructuresInit2.c:271`).
`InitSrcRects()` (`:306-475`) assigns only **116** of 144 entries. Twenty-seven of the 28 gaps
are class padding, but **`0x85 kFlower` is a real placeable type whose `srcRects` entry is
never written** — flowers are drawn from a separate `flowerSrc[6]` array instead. A port that
zero-initialises gets a degenerate rect where the original got heap garbage.
`constants.md` §6, `graphics-assets.md` §6.5, open question in `graphics-assets.md` OQ2.

### 8.11 The 12-byte `objectType` tagged union

`object-taxonomy.md` §1.2 is the byte-by-byte authority. Shape: `short what` (2 bytes, the ID
above) + `Rect`/payload discriminated by **`what` range, not by a high nibble** — nine 10-byte
variants (`blowerType`, `furnitureType`, `bonusType`, `transportType`, `switchType`,
`lightType`, `applianceType`, `enemyType`, `clutterType`; `GliderStructs.h:90-105`).

Four traps, each of which silently corrupts a house if got wrong:

* In `bonusType`, **`state` precedes `initial`** — the opposite order to blower / light /
  appliance / enemy.
* In `enemyType`, **`delay` is at U6 and `byte0` at U7 — the reverse of `applianceType`.**
* `kDeluxeTrans` (0x40) packs four values into two fields: `tall` hi/lo = width/4 and
  height/4; `wide` hi/lo nibbles = `initial` and `state`.
* `blowerType.vector` low nibble is 1=up, 2=right, 4=down, 8=left and **must be masked with
  `0x0F`** — real houses carry high bits (max observed 0x14).

---

## 9. Interactions, collision and room boundaries

Authority: `interactions.md`; `object-draw-all.md` for how hot spots get emitted.

### 9.1 Hot spots are the only channel

`hotSpots[kMaxHotSpots]`, `kMaxHotSpots` = **56** (`GliderDefines.h:259`). Objects register
rects during the drawing pass (`DrawARoomsObjects`); the interaction pass tests the glider
against them and dispatches on a 28-value action code. `doScrutinize` selects the precise
`SectGlider` test rather than a plain rect intersection.

### 9.2 The 28 action codes (`GliderDefines.h:282-309`)

| # | Action | # | Action |
|---:|---|---:|---|
| 0 | `kIgnoreIt` | 14 | `kSlideIt` |
| 1 | `kLiftIt` | 15 | `kTransportIt` |
| 2 | `kDropIt` | 16 | `kIgnoreLeftWall` |
| 3 | `kPushItLeft` | 17 | `kIgnoreRightWall` |
| 4 | `kPushItRight` | 18 | `kMailItLeft` |
| 5 | `kDissolveIt` | 19 | `kMailItRight` |
| 6 | `kRewardIt` | 20 | `kDuctItDown` |
| 7 | `kMoveItUp` | 21 | `kDuctItUp` |
| 8 | `kMoveItDown` | 22 | `kMicrowaveIt` |
| 9 | `kSwitchIt` | 23 | `kIgnoreGround` |
| 10 | `kShredIt` | 24 | `kBounceIt` |
| 11 | `kStrumIt` | 25 | `kChimeIt` |
| 12 | `kTriggerIt` | 26 | `kWebIt` |
| 13 | `kBurnIt` | 27 | `kSoundIt` |

### 9.3 Room-boundary limits (`GliderDefines.h:503-511`)

| Constant | Value | Meaning |
|---|---:|---|
| `kCeilingLimit` | 8 | solid ceiling y |
| `kFloorLimit` | 312 | solid floor y |
| `kRoofLimit` | 122 | outdoor roof y |
| `kLeftWallLimit` | 12 | solid left wall x |
| `kRightWallLimit` | 500 | solid right wall x |
| `kNoLeftWallLimit` | **-24** | open-left escape x (`= -kGliderWide/2`) |
| `kNoRightWallLimit` | **536** | open-right escape x (`= kRoomWide + kGliderWide/2`) |
| `kNoCeilingLimit` | **-10** | open-ceiling escape y |
| `kNoFloorLimit` | **332** | open-floor escape y |

### 9.4 `CheckGliderInRoom` — the exact rules

`GliderPRO/Sources/Interactions.c:689`. `interactions.md` §12 is the authority. Load-bearing
details:

1. **Only modes 0, 7, 8 and 9** (`kGliderNormal`, `kGliderFaceLeft`, `kGliderFaceRight`,
   `kGliderBurning`) are subject to room boundaries at all.
2. Vertical checks are one `if / else if / else if` chain — ceiling (`:696`), floor (`:709`),
   then `thisBackground == kRoof && dest.bottom > kRoofLimit` (`:722`), so a glider past
   `kFloorLimit` gets **floor** handling even in a roof room; **horizontal is a separate
   `if / else if` chain** (`:725`, `:738`), so one vertical and one horizontal resolution can
   both fire in the same frame. The two chains are **asymmetric**: the vertical tests use the
   fixed constants of §9.3 and defer openness to the escape functions, while the horizontal
   tests compare against the room-dependent globals `leftThresh` / `rightThresh` (assigned in
   `Room.c:835-878`) — 12 / 500 for a walled side but **-24 / 536 for an open one**. A port that
   hard-codes 12 / 500 here changes rooms 36 px early in open-sided rooms
   (`interactions.md` §12, `:2673-2696`).
2a. Items 4-6 below are **not** in `CheckGliderInRoom`; it only dispatches. They live in
   `CheckEscapeLeft` / `Right` / `Down` (`Interactions.c:378`, `:572`, `:662`) and their
   two-player variants (`:283`, `:509`, `:599`).
3. A **burning** glider dies at *every* boundary — and `wasMode` is set to 0 first.
4. Wall bounce is reflection **amplified by penetration depth**:
   `offset = kLeftWallLimit - dest.left; hVel = -hVel + offset;`
5. **Landing on a solid floor is fatal** unless `ignoreGround` (a manhole) was set this frame
   *and* `dest.bottom > kNoFloorLimit` (332).
6. Walking through a door works because there is **no clamp at all** while `ignoreLeft` is set
   and `dest.left` is in `[-24, 12)`.

### 9.5 Everything that kills the glider

`progression.md` §10.2 has the complete list. All deaths funnel through **`OffAMortal`**
(`Player.c:1484-1604`), reached from exactly two places:

| Path | Trigger |
|---|---|
| `FadeGliderOut` (`Player.c:295`) | `frame--` reaches < 0 after `StartGliderFadingOut` set `frame = 15`. **30 call sites** across `Interactions.c`, `Dynamics.c`, `Player.c`, `Transit.c` — the generic "you touched something lethal" death. 16 frames = 0.53 s |
| `MoveGliderShredding` (`Player.c:1317`) | `frame++` reaches ≥ 0 from `kShredderCountdown` -68 |

---

## 10. Room transitions and progression

Authority: `progression.md` §3-§9.

### 10.1 Addressing

Three levels of "where": **house** (a file), **room** (`floor`, `suite`), **room-space pixel**.
`thisRoom` is a *copy*, not a pointer into the house handle (`progression.md` §3.1) — a port
must reproduce the copy-back discipline or lose state. Room lookups are linear scans
everywhere. Address space ceiling is **8192** rooms (`kMaxNumRoomsV` 64 floors x
`kMaxNumRoomsH` 128 suites); corpus max is 531. `kRoomIsEmpty` = -1 marks tombstones left by
room deletion.

### 10.2 `MoveRoomToRoom` — side and vertical transitions

`GliderPRO/Sources/Transit.c:151`, 22 steps, enumerated at `progression.md` §8.3. Key facts:

* `HandleRoomVisitation()` is the **first statement** (`Transit.c:155`), so the 100-point
  room-visit award happens **on departure**, not arrival.
* `OffsetGlider` applies exactly `±kRoomWide` (512) horizontally / `±kTileHigh` (322) vertically
  — note it uses `kTileHigh`, **not** `kVertLocalOffset`, even though both are 322
  (`Player.c:1469`, `:1474`; `progression.md` §8.8). `kVertLocalOffset` is used by
  `localRoomsDest` instead (`InterfaceInit.c:211-218`).
* Entry Y for a side transition is `kGliderStartsDown (32) + leftStart/rightStart - 2`.
* `DrawLocale` rebuilds the whole 3x3 view; `WipeScreenOn` animates the change.

### 10.3 Escape codes (`GliderDefines.h:596-608`)

Negative sentinels in the two-player protocol:

| Value | Constant | Value | Constant |
|---:|---|---:|---|
| -1 | `kNoOneEscaped` | -8 | `kPlayerEscapingUpStairs` |
| -2 | `kPlayerEscapedRight` | -9 | `kPlayerEscapingDownStairs` |
| -3 | `kPlayerEscapedLeft` | -10 | `kPlayerTransportedOut` |
| -4 | `kPlayerEscapedUp` | -11 | `kPlayerDuckedOut` |
| -5 | `kPlayerEscapedDown` | -12 | `kPlayerMailedOut` |
| -6 | `kPlayerEscapedUpStairs` | -69 | `kPlayerIsDeadForever` |
| -7 | `kPlayerEscapedDownStairs` | | |

### 10.4 Two-player "escape / limbo / follow the leader"

A three-way handshake: the leading player escapes, goes into `kGliderInLimbo`, the second
player is dragged along by `FollowTheLeader`. `mortals` and the whole inventory are **shared
globals**. `progression.md` §9. Delete forces a kill on the straggler (§7.4;
`Input.c:366-371` → `ForceKillGlider`, `Transit.c:449-469`).

### 10.5 Winning and losing

* **Win**: leave the house through the designated exit condition (`progression.md` §11.1) →
  `DoGameOver` → an angel/star animation sequence → `TestHighScore()`.
* **Lose**: `mortals < 0` → `InitDiedGameOver` → the eight-page "GAME OVER" paper flutter
  (`HandlePages`, `DrawPages`) → `DoDiedGameOver`. `progression.md` §12, `scoring.md` §8.6.
* Both are preceded by a **16-frame countdown** during which the simulation keeps running
  (`scoring.md` §8.3).

### 10.6 Dynamic-layer hard limits

`kMaxDynamicObs` 18 (`dinahs[]`), `kMaxSavedMaps` 24, `kMaxMasterObjects` 216
(= 24 x 9), `kMaxRoomObs` 24, `kMaxSparkles` 3, `kMaxFlyingPts` 3, `kMaxCandles` 20,
`kMaxTikis` 8, `kMaxCoals` 8, `kMaxPendulums` 8, `kMaxRubberBands` 2, `kMaxGrease` 16,
`kMaxStars` 4, `kMaxShredded` 4 (`GliderDefines.h:246-266`). **Exhaustion is observable
behaviour**, not an error: `savedMaps` running out changes what animates
(`progression.md` §15.3, `object-draw-all.md` on budget exhaustion).

---

## 11. Scoring, lives and inventory

Authority: `scoring.md`.

### 11.1 Points — the entire table

`GliderDefines.h:536-541`. There is no multiplier, no combo, no difficulty scaling.

| Event | Points | Awarded at |
|---|---:|---|
| Leave a room for the first time | 100 (`kRoomVisitScore`) | `Transit.c:442` |
| `kRedClock` 0x21 | 100 | `Interactions.c:776` |
| `kBlueClock` 0x22 | 300 | `Interactions.c:792` |
| `kYellowClock` 0x23 | 500 | `Interactions.c:808` |
| `kCuckoo` 0x24 | 1000 | `Interactions.c:825` |
| `kInvisBonus` 0x2B | 100 / 300 / 500 (per-object `data.c.points`) | `Interactions.c:928` |
| `kStar` 0x2C | 5000 | `Interactions.c:948` |

Independently corroborated by the editor's `CountTotalHousePoints()`
(`HouseInfo.c:51-105`) — six cases, matching exactly. Note it scans all 24 object slots
regardless of `numObjects`.

The room-visit award fires **on departure**, so the starting room is not scored until you
leave it. `scoring.md` §2.3. The flag is written in two places: the authoritative copy inside
the locked house handle and the working copy `thisRoom->visited`.

Score display **rolls** toward the true value at `kScoreRollAmount` = **13 points per frame**
(`Scoreboard.c:15-21`). Reproducing the roll is a feel requirement (`scoring.md` §B).

### 11.2 Lives (`mortals`) — a count of *spares*

`kInitialGliders` = 2 (`Play.c:19`).

| Mode | `mortals` after init | Deaths available |
|---|---:|---:|
| One player | 2 | **3** |
| Two players | 4 | 5 |

The assignment-then-add order in `InitGlider` (`Play.c:338-352`) matters: `mortals = 2;
if (twoPlayerGame) mortals += 2;`, and `InitGlider` is called twice in a two-player game
(`Play.c:122-123`), so the result is 4, not 6.

**The only extra life is `kPaper` (0x25)** (`Interactions.c:831-848`), worth zero points:
`mortals++`, and `mortals++` again if `twoPlayerGame && !onePlayerLeft`. There is **no cap**
(Leviathan has 46 papers) and **no score-threshold extra life anywhere** — `theScore` is
compared only against `displayedScore` in the scoreboard roll and against the high-score
table. `scoring.md` §3.3.

`onePlayerLeft` (`Player.c:53`, set at `:1507`) reverts every doubled pickup — paper, battery,
bands, foil, helium — to single value once one player is out.

### 11.3 The four consumables (`Interactions.c:13-19`)

| Consumable | Counter | Sign | Per pickup | Unit |
|---|---|---|---:|---|
| Battery | `batteryTotal` | positive | 50 | frames of thrust |
| Helium | `batteryTotal` | **negative** | 150 | frames of lift |
| Rubber bands | `bandsTotal` | positive | 8 | bands |
| Foil | `foilTotal` | positive | 8 | hits absorbed |

**Battery and helium share one signed counter**, with asymmetric, destructive top-up rules
(`Interactions.c:860-865` and `:966-971`):

```c
/* battery */  if (batteryTotal > 0) batteryTotal += 50;  else batteryTotal =  50;
/* helium  */  if (batteryTotal < 0) batteryTotal -= 150; else batteryTotal = -150;
```

so picking up a battery while holding helium **destroys** the helium and vice versa. This is
why the two badges share one scoreboard slot (§3.5). The microwave (`kMicrowave` 0x6A) is the
only bulk wipe (`scoring.md` §4.5).

### 11.4 High scores

`kMaxScores` = 10 (`GliderDefines.h:249`). The real store is `scoresType`, **292 bytes at
`houseType` offset 528** — i.e. inside the house file itself. The `'gliS'` side-car under
`Preferences:G-PRO Scores {f}:` is dead code because `IsFileReadOnly` is an unconditional
`false` stub (`HouseIO.c:663`). `TestHighScore` is the qualification gate; `SortHighScores` is
a destructive in-place selection sort with a padding hazard. `scoring.md` §7,
`ui-dialogs.md` §9.

### 11.5 Saved games: a complete and completely dead subsystem

* `gameType` — 40-byte in-house snapshot at `houseType` +820, guarded by `hasGame` at +860.
  `SaveGame` **refuses two-player games** (`SavedGames.c:309-310`).
* `game2Type` — the standalone `'gliG'` document header, **110 bytes (0x6E), not 114**. The
  header comment's `// total = 114` counts a flexible array member's phantom `// 4`.
  `format-decisions.md` **D1**.
* Every read path is dead: `SaveGame2`'s body is commented out (`SavedGames.c:33-147`) and
  `OpenSavedGame` returns `false` at `:169` before its body. `QueryResumeGame` (`Menu.c:710`)
  has **zero** call sites. `progression.md` §16, `scoring.md` §9.

---

## 12. Audio

Authority: `audio.md`.

### 12.1 Engine

Four `SndChannel`s: three for effects plus one for music, all created with
`sampledSynth | initNoInterp | initMono` (`Sound.c:379`, `:387`, `:395`, `Music.c:286-288`).
The game is **unambiguously mono**: `UnivSetSoundVolume` writes the same value into both halves
of the long (`Utilities.c:782`). Playback is `bufferCmd` + `callBackCmd`; priority arbitration
(`PlayPrioritySound`) evicts the lowest-priority voice.

The application's own samples — all 64 effects and all 7 music pieces, 70 resources — are
**8-bit unsigned mono PCM at exactly 22254.5455 Hz** (`sampleRate` Fixed `0x56EE8BA3`, the
classic Mac rate). **House-supplied `'snd '` resources are not:** nine distinct rates occur
across the 63 of them — 22254.5455, 22254.5454, 22255.0, 22050.0, 11127.2727, 11127.5, 9779.0,
7418.1818 and 5563.6364 Hz (`audio.md` §3.5) — so a port must read `sampleRate` per resource and
resample, not assume one rate. (For the five `cmpSH` house sounds the header's frame count is
*packets*, not samples — sample count is `packets * 6`; `audio.md` §5.) Conversion to the
port's format is
`s16 = (int16(b) - 128) << 8`. Resampling by point-sampling is a deliberate fidelity choice.
`audio.md` §11.

`'snd '` parsing skips **+20 bytes** to reach the sample header (`Sound.c:286`, `:296`),
assuming format-1 with one synth, one command and one `soundHeader` prefix.

### 12.2 The 64 sound effects

Index = the constant's value; **`'snd '` resource ID = index + 1000**.
`GliderDefines.h:55-118` for names, `:120-181` for priorities.

| # | Constant | # | Constant | # | Constant | # | Constant |
|---:|---|---:|---|---:|---|---:|---|
| 0 | `kHitWallSound` | 16 | `kCaughtFireSound` | 32 | `kTVOnSound` | 48 | `kPaper1Sound` |
| 1 | `kFadeInSound` | 17 | `kScoreTikSound` | 33 | `kTVOffSound` | 49 | `kPaper2Sound` |
| 2 | `kFadeOutSound` | 18 | `kThrustSound` | 34 | `kCoffeeSound` | 50 | `kPaper3Sound` |
| 3 | `kBeepsSound` | 19 | `kFizzleSound` | 35 | `kMysticSound` | 51 | `kPaper4Sound` |
| 4 | `kBuzzerSound` | 20 | `kFireBandSound` | 36 | `kZapSound` | 52 | `kTypingSound` |
| 5 | `kDingSound` | 21 | `kBandReboundSound` | 37 | `kPopSound` | 53 | `kCarriageSound` |
| 6 | `kEnergizeSound` | 22 | `kGreaseSpillSound` | 38 | `kEnemyInSound` | 54 | `kChord2Sound` |
| 7 | `kFollowSound` | 23 | `kChordSound` | 39 | `kEnemyOutSound` | 55 | `kPhoneRingSound` |
| 8 | `kMicrowavedSound` | 24 | `kVCRSound` | 40 | `kPaperCrunchSound` | 56 | `kChime1Sound` |
| 9 | `kSwitchSound` | 25 | `kFoilHitSound` | 41 | `kBounceSound` | 57 | `kChime2Sound` |
| 10 | `kBirdSound` | 26 | `kShredSound` | 42 | `kDripSound` | 58 | `kWebTwangSound` |
| 11 | `kCuckooSound` | 27 | `kToastLaunchSound` | 43 | `kDropSound` | 59 | `kTransOutSound` |
| 12 | `kTikSound` | 28 | `kToastLandSound` | 44 | `kFishOutSound` | 60 | `kTransInSound` |
| 13 | `kTokSound` | 29 | `kMacOnSound` | 45 | `kFishInSound` | 61 | `kBonusSound` |
| 14 | `kBlowerOn` | 30 | `kMacBeepSound` | 46 | `kDontExitSound` | 62 | `kHissSound` |
| 15 | `kBlowerOff` | 31 | `kMacOffSound` | 47 | `kSizzleSound` | 63 | `kTriggerSound` |

Per-sound frame counts, durations in ms, purge flags and exact priorities are tabulated in
`audio.md` §5.

### 12.3 Priority bands (`GliderDefines.h:120-181`)

| Band | Members (examples) |
|---|---|
| 100-103 | `kHitWall` 100, `kScoreTik` 101, `kBandRebound` 102, `kDontExit` 103 |
| 200-204 | `kTik`, `kTok`, `kMystic`, `kChime1`, `kChime2` |
| 300-311 | `kThrust` 300, `kFireBand` 301, `kChord`, `kVCR`, `kToastLaunch`, `kToastLand`, `kCoffee`, `kBounce`, `kDrip`, `kDrop`, `kWebTwang`, `kHiss` 311 |
| 400-413 | `kFoilHit` 400 … `kSizzle` 413 |
| 500 | `kPhoneRing` |
| 700-703 | `kSwitch`, `kBlowerOn`, `kBlowerOff`, `kFizzle` |
| 800-812 | `kBeeps` 800 … `kBonus` 812 |
| 900-906 | `kFadeIn` 900, `kFadeOut`, `kCaughtFire`, `kShred`, `kFollow`, `kTransIn`, `kTransOut` 906 |
| 999 | `kTrigger` |

There is **no 600 band.**

### 12.4 Music

Seven pieces, `'snd '` **2000-2006** (`kBaseBufferMusicID` 2000, `kMaxMusic` 7):
`Refrain1.22`, `Refrain2.22`, `Refrain3.22`, `Refrain4.22`, `Chorus.22` (about double the
length of the others), `RefrainSparse1.22`, `RefrainSparse2.22`.

The idle/menu score sequence is `musicScore[16] = {0,1,2,3,4,4,0,1,2,3,4,4,5,6,4,4}`, with
`kLastMusicPiece` = 16 and `kLastGamePiece` = 6 (`Music.c:17-18`).

**`musicScore` is only the idle/menu score.** During a game `musicMode` is
`kPlayGameScoreMode` (-2) and the engine walks a *second* array,
`gameScore[6] = {6,5,-1,6,4,4}` (`Music.c:349-354`) = RefrainSparse2, RefrainSparse1,
back-jump sentinel, RefrainSparse2, Chorus, Chorus. The `-1` at index 2 is **not** a
terminator — `musicCursor += musicSoundID` makes the cursor jump *back* one
(`Music.c:186-189`) — and the game-score wrap is to **1**, not 0 (`Music.c:183-184`), so
`gameScore[0]` plays only once per session. A port built from `musicScore` alone plays the
wrong music during gameplay (`audio.md` §7.4, `:2094`, `:2144-2152`).

Mode constants: `kProdGameScoreMode` -4,
`kKickGameScoreMode` -3, `kPlayGameScoreMode` -2, `kPlayWholeScoreMode` -1, `kPlayChorus` 4,
`kPlayRefrainSparse1` 5, `kPlayRefrainSparse2` 6 (`GliderDefines.h:47-53`).
`audio.md` §7.

### 12.5 House sounds

Houses may carry their own `'snd '` resources; 13 of the 22 do. **Five are MACE 6:1 compressed
(`cmpSH`, `compressionID` 4, `packetSize` 8) and cannot be decoded from anything in this tree**
— the quantisation tables live in the Sound Manager. Affected: `CD Demo House` 3007 "Door
Chime", `Demo House` 3011 "Meow", `Nemo's Market` 3001 "Door Chime", 3003 "Cash Register",
3004 "Cat Meow". Ship them as silence until the tables are obtained. `audio.md` OQ3.

---

## 13. The house (level) file format — summary

**Byte-exact authority: [`analysis/house-format.md`](analysis/house-format.md).** Normative
rulings on the open edges: [`analysis/format-decisions.md`](analysis/format-decisions.md)
D1-D7.

### 13.1 Shape

A house file's data fork **is the raw big-endian struct image of `houseType`**, with no
container, no chunking and no framing:

```
file = houseType (866 bytes) followed by nRooms x roomType (348 bytes each)
total = 866 + 348 * nRooms
```

Type `'gliH'`, creator `'ozm5'`. All 22 shipped houses have `version == 0x0200`
(`kHouseVersion`).

Object *k* of room *r* is at byte offset `866 + 348*r + 60 + 12*k`.

Demo House = **16526 bytes** = `866 + 45*348`, asserted by the `COMPILEDEMO` build
(`HouseIO.c:349`, `:382`) — though `COMPILEDEMO` is commented out in this drop, so both asserts
are dead code here (§2).

### 13.2 `houseType` — 866 bytes

| Offset | Field | Notes |
|---:|---|---|
| 0 | `version` | 0x0200 |
| 2 | `unusedShort` | **uninitialised garbage. Preserve verbatim** (D4) |
| 4 | `timeStamp` | a date **and** bit 0 = the house **locked** flag. `WriteHouse` stores it as `timeStamp & 0x7FFFFFFF` (`HouseIO.c:478`), and Mac seconds passed 2^31 in 1972, so **every** stored house date is ~68 years early: read it as `stored \| 0x80000000`. Restored, the 22 houses fall in 1995-06..1995-12 (Sampler 2000-05) and 11 of them match, to the day, the newest *unmasked* high-score stamp in the same file. `house-format.md` §3.3.1; `internal/house/mactime.go` |
| 8 | `flags` | observed {0x00 x14, 0x02 x7, 0x06 x1 (`Art Museum`)}. bit 0 `wardBitSet`, bit 1 `phoneBitSet` (the House Info **No Phone** checkbox), bit 2 **inverted** `bannerStarCountOn` (`HouseIO.c:416-418`). So 8 houses suppress the ringing phone |
| 12 / 14 | `initial.v` / `initial.h` | start point |
| 16 | `banner` | Str255 |
| 272 | `trailer` | Str255 |
| 528 | `highScores` | `scoresType`, 292 bytes |
| 820 | `savedGame` | `gameType`, 40 bytes — **non-zero residue in 22 of 22** houses; stale-with-`hasGame == 0` in 20, `hasGame == 1` in 2 (`ImagineHouse PRO II`, `Titanic`). `savedGame.version` is 0x0100 in 17 houses, 0x0000 in 2 (`Davis Station`, `Fun House`) and 0xF6F6 / 0x00A9 / 0x0001 in `Art Museum` / `Grand Prix` / `Sampler` — **never** `kSavedGameVersion` 0x0200, so never validate it. **Preserve verbatim** (D4) |
| 860 | `hasGame` | |
| 861 | `unusedBoolean` | heap residue: observed 255, 30, 37, 185, 30, 14, 2 |
| 862 | `firstRoom` | |
| 864 | `nRooms` | |

On `flags` bit 1: it is **house-scoped, not per-room**, and its sense is negative. It is the House
Info dialog's `No Phone` checkbox (`kNoPhoneCheck` = 14, `HouseInfo.c:19`, `:120`, `:294-297`;
`DITL` 1001 item 14 is literally labelled "No Phone", `Glider PRO.r:4376-4377`), decoded once per
house into `phoneBitSet` at `HouseIO.c:417`. `phoneBitSet == true` therefore means *"this house
has **no** phone"*, so `HandleTelephone`'s `if (!phoneBitSet)` (`Play.c:748`) is **correct and not
inverted** — an earlier "inverted gate" reading was withdrawn (`determinism-networking.md`
porting-risk item 10, RESOLVED; `house-format.md:569`). Because the flag changes which random
events run, it must be inside any determinism hash — which it is, being inside the house bytes.
One genuine editor-only latent bug worth **not** copying: the clear path masks with `0xFFFFDFFD`
(`HouseInfo.c:272`), which also clears bit 13.

### 13.3 `roomType` — 348 bytes

| Offset | Field | Offset | Field |
|---:|---|---:|---|
| 0 | `name` (`Str27` — 1 length byte **clamped to ≤ 27** by `HouseLegal.c:871-873`, + 27 MacRoman bytes = 28 total; residue after `name[0]` bytes) | 34 | `background` (`short`, `PICT` resource ID: built-ins 2000-2017, user art 3000-3799 — see §5.3, §14.2) |
| 28 | `bounds` | 36 | `tiles[8]` (column indices, §5.3) |
| 30 | `leftStart` | 52 | `floor` |
| 31 | `rightStart` | 54 | `suite` (-1 = `kRoomIsEmpty`) |
| 32 | `unusedByte` | 56 | `openings` (**dead**) |
| 33 | `visited` | 58 | `numObjects` |
| | | 60 | `objects[24]` x 12 bytes |

`roomType.unusedByte` is the one reserved field the original **force-zeroes** (`Room.c:173`,
`HouseLegal.c:868`) and it is 0 in all 4,070 corpus rooms. Every *other* reserved field, every
`StrNN` tail, `hasGame`, and the whole `savedGame` block are nondeterministic residue that must
be **preserved byte-for-byte** for a round-trip (D4). A naive Pascal-string-to-Go-`string`
round-trip fails **22 of 22** houses.

### 13.4 Sizing and bounds (D2, D3, D5)

* Alignment model is `align=mac68k` (== GCC `pack(2)`): `sizeof(houseType)` **866**,
  `sizeof(roomType)` **348**, `sizeof(demoType)` **6**. Every *field offset* is identical under
  both models; the only three records whose **size** changes are `houseType` 866 → 868,
  `game2Type` 110 → 112 and `demoType` 6 → 8. `roomType` is **348 under both** (all its members
  are 1 or 2 bytes wide), as are `objectType` 12, `gameType` 40, `scoresType` 292 and
  `savedRoom` 292. **Parse at `866 + 348*r`, never at `sizeof(struct)*r`.**
* Load-time bound: `nRooms = min(headerNRooms, (fileLen - 866) / 348)`, `1 <= nRooms <= 32767`.
  There is no `kMaxRooms`. Semantic ceiling is 8192 addressable rooms.
* **21 of 22** data forks are exactly `866 + 348*nRooms`; `Sampler` alone is **+2 bytes**
  (trailing `01 01`), which is allocation slack from a natural-alignment (868-byte header)
  build. **Accept, preserve, never reject, and never derive `nRooms` from file length alone.**

### 13.5 The BinHex container

The vendored houses are `GliderPRO/Houses/*.binhex` — BinHex 4.0, which preserves both Mac
forks. Alphabet is the standard 64 characters, RLE marker is `0x90 <n>`, and there are **three
CRC-16/XMODEM checks** (header, data fork, resource fork). All **66 checks pass across all 22
houses** — but note that `tools/probe_house.py` with `check_crc=True` verifies only the two
*fork* CRCs (`:137`, `:140-142`), i.e. 44 of the 66; the 22 **header** CRCs (stored at payload
offset +18, after type 4 + creator 4 + flags 2 + dataLen 4 + rsrcLen 4) are parsed but never
checked by the tool and must be checked separately. Also: `binhex_decode(path)` defaults to
`check_crc=False` — pass `check_crc=True` — and the returned dict's resource-fork key is
`rsrc`, not `res`.

### 13.6 Corpus invariants any loader should reproduce

22 houses; all `version == 0x0200`; **4,070 rooms total**; `nRooms` from 2 (`Sampler`) to 531
(`Teddy World`); 7 unlocked / 15 locked; two pairs share an **identical 40-byte `savedGame`
residue block at +820** — `CD Demo House` / `Slumberland` and `Nemo's Market` / `SpacePods`.
**The files themselves are all distinct** (72,554 vs 134,150 bytes; 44,018 vs 140,762; all 22
data-fork md5s differ): each pair descends from a common template that was then edited heavily
and the residue block was never rewritten (`unresolved-format-decisions.md` `:99`, closing
`scoring.md` OQ5). Per-house detail in `houses-inventory.md` and `original-houses.md`.

### 13.7 Links

Room links are encoded in a packed field; the codec (`MergeFloorSuite` /
`ExtractFloorSuite`) is at `progression.md` §5.2, with a full audit of every link in all 22
houses at §5.5. `CheckDuplicateFloorSuite`'s `bitPlace = ((floor + 7) * 128) + suite` spans
exactly 0..8191 over the legal domain and is **not** an off-by-one; the real bug is the call
order (`HouseLegal.c:1089` before `:1110`). `constants.md` OQ1, closed by D5's corollary.

---

## 14. Resources and assets

Authority: `resource-fork.md` (inventory), `graphics-assets.md` (formats + pipeline).

### 14.1 `Glider PRO.r`

| Metric | Value |
|---|---:|
| File size | 15,475,666 bytes |
| Lines (LF-normalised) | 199,843 |
| Resources | **538** |
| Types | **35** |
| Total payload | 3,168,309 bytes |
| Text encoding | MacRoman |

It is a **DeRez text dump**, not a binary fork: `tools/probe_rez.py` walks the header
character-by-character (`:78`) and must read the file in **binary** mode (`:145`). There is no
`MBAR`, there are no font resources, and there is no Balloon Help.

### 14.2 The 152 `PICT`s, partitioned by pipeline role

`graphics-assets.md` §7.8. Both this partition and `resource-fork.md` §3.1.1's ID-range
partition were computed programmatically and reconcile exactly.

| Bucket | Count | IDs |
|---|---:|---|
| `sheet_art` | 14 | 4000-4005, 4008, 4011-4016, 4018 |
| `sheet_mask` | 13 | 5000-5002, 5004, 5005, 5008, 5011-5016, 5018 |
| `strip_art` | 14 | 1019, 1996, 1997, 1999, 3963, 3974, 3976, 3998, 3999, 4006, 4007, 4009, 4010, 4017 |
| `strip_mask` | 8 | 1020, 4998, 4999, 5006, 5007, 5009, 5010, 5017 |
| `obj_art` | 38 | 21 colour-keyed + 9 opaque + 8 mask-paired |
| `obj_mask` | 8 | 3903, 3904, 3912-3915, 3921, 3927 |
| `bg` | 18 | 2000-2017 |
| `misc` | 1 | 3957 (`kManholeThruFloor`) |
| `ui` | 38 | 150, 151, 153, 1000-1018, 1021-1023, 1202, 1211, 1216, 1217, 1988-1995, 1998 |
| **Total** | **152** | |

### 14.3 The PICT dialect actually used

Nine opcodes, and **all 152 pictures decode with zero rejections** under that allow-list
(`graphics-assets.md` §7.2):

| Opcode | Name | Handling |
|---|---|---|
| 0x0011 | `VersionOp` | detect v1/v2; in v2 consume 2 bytes then align every read to an even offset |
| 0x0C00 | `HeaderOp` | consume 24 bytes and ignore them **whatever the `version` field says** — both `-1` (95 pictures) and `-2` (21 pictures, incl. the glider sheet 3999, the splash 1000, the scoreboard 1997 and backgrounds 2002, 2016, 2017) occur, and both payloads are 24 bytes. **Never reject `-2`** |
| 0x0001 | `Clip` | read `uint16 rgnSize`, skip it (always 10 = a bare rect) |
| 0x0090 | `BitsRect` | rows **unpacked**; 17 occurrences |
| 0x0098 | `PackBitsRect` | the workhorse, 185 occurrences |
| 0x0099 | `PackBitsRgn` | as 0x0098 + read and **ignore** the mask region (3 occurrences) |
| 0x001E | `DefHilite` | skip 0 bytes |
| 0x00A0 | `ShortComment` | skip 2 bytes |
| 0x00FF | `OpEndPic` | stop |

A Glider-specific decoder should treat everything else as fatal: **not one pixel of shipped art
is drawn by a vector opcode**, there is zero text, no `DirectBits`, no QuickTime payload, no
pattern PixMaps. 204 of 205 raster ops have `packType == 0` (the exception is `packType == 1`
in `PICT` 1003); depths are 8 bpp (167 ops) and 1 bpp (38 ops), nothing else; all 205 use
`srcCopy` — **transparency is never inside a PICT**.

### 14.4 Palette

`clut` 128, 256 entries. 168 of 205 raster ops carry an embedded `ColorTable` and **167 of the
168 resolve byte-identically to `clut` 128**. Resolution order (`graphics-assets.md` §7.5):
embedded table first (honouring `ctFlags` bit 15), then `clut` 128, then — crucially —
**1-bit data is not palette data**: for `pixelSize == 1`, value 1 is opaque and value 0 is
transparent; running a mask through `clut` 128 yields plausible-looking nonsense. `clut` 128 is
a safety net for the shipped art but a requirement for user house art.

`kRedOrangeColor8` = 23, with the source comment "actually, 18"
(`GliderDefines.h:542`). `kGrayBackgroundColor` = 251, plate grey `#555555`.

### 14.5 Movies

15 per-house QuickTime `.mov` files play on `kTV` screens. **14 of the 15 have lost their
`moov` atom**, so their frame rate is unrecoverable; only `Demo House.mov` is known
(`timeScale / sampleDuration` = 600/121 = **4.9587 fps**). `quicktime-movies.md` §19.5 recommends
5 fps uniformly as a considered guess. Codecs are **`rle `** (10 movies), **`raw `** (4 —
`Art Museum`, `Castle o' the Air`, `Davis Station`, `Slumberland`) and **`smc `** (1 —
`Grand Prix`); a pipeline built for only `rle `/`smc ` silently loses 4 of the 15
(`quicktime-movies.md` §10). `Grand Prix.mov`'s custom ~59-entry colour table is unrecoverable.

---

## 15. The UI / menu shell

Authority: `ui-dialogs.md`.

### 15.1 Modes and boot order

`theMode` (`MainWindow.c:41`) is `kSplashMode` 0, `kEditMode` 1, `kPlayMode` 2. The outer loop
is literally `while (!quitting) HandleEvent();` (`Main.c:363-364`); when a game starts,
`PlayGame` takes over and spins its own loop.

Startup order (23 steps, `Main.c:284-370`, full list at `ui-dialogs.md` §2.2). The ordering
consequence worth remembering: `ReadInPrefs()` runs **before** the depth switch and before the
window exists but **after** `CheckOurEnvirons`, and `InitializeMenus()` is step **21** — the
splash is drawn and a house is opened before any menu can be pulled down. `menusUp`
(`Menu.c:33`) guards every menu-update function against being called early.

### 15.2 Menu bar

| Menu | `MENU` | Items |
|---|---:|---|
| Apple | 128 | `About Glider PRO…`; plus `AppendResMenu('DRVR')` items whose **handler is commented out** (`Menu.c:281-296`) — a port should not append them |
| Game | 129 | `New Game` ⌘N (the resource text carries a **trailing NUL**), `Two Player Game` ⌘2, `Open Saved Game…` ⌘O, `Load House…` ⌘L, `Quit` ⌘Q |
| Options | 130 | `Room Editor` ⌘E (checkmarked in edit mode), `High Scores…` ⌘H, `Preferences…` ⌘P, `Demo…` ⌘D |
| House | 131 | Inserted **only in `kEditMode`**; 21 items, all gated on `houseUnlocked`. `Paste Room` is permanently disabled and renamed `Nothing To Paste` |

Genuine 1.0.4 conflict: `Options▸Demo…` (⌘D) and `House▸Duplicate Object` (⌘D) are both
enabled in edit mode with an object selected; `MenuKey` scans in bar order and Options wins, so
⌘D runs the demo. `ui-dialogs.md` §3.3.

Editor pop-ups (not in the bar): `MENU` 140 `Rooms` (18 background names mapping 1:1 to
`PICT` 2000-2017, plus a runtime-enabled `Original Artwork` item), `MENU` 141 `Tools`
(9 tool-mode constants 1-9: Blowers, Furniture, Prizes, Transport, Switches, Lighting,
Appliances Etc., Enemies, Clutter).

### 15.3 Dialog inventory

28 `DLOG`, 26 `ALRT`, with complete byte-decoded `DITL` dumps at `ui-dialogs.md` §5.3-5.4.
Preferences is four panes: main (`DLOG` 1012), Display (1017), Sound (1018), Controls (1023),
plus the Brains easter-egg pane (1024). House selection is `DITL` 1000's paged grid with
keyboard navigation in `LoadFilter` (`SelectHouse.c:198-349`).

Fonts: **there are no font resources in the fork.** `TextFont(applFont)` resolves to the
system's application font (Geneva on the era's Macs). `toolbox-primitives.md` R-FONT-1 gives
the normative substitution: Chicago 12 → Liberation Sans Bold @ 13 px; Geneva 9 → Liberation
Sans Regular @ 9 px; Geneva 12 → @ 11 px; Geneva 14 → @ 13 px, with bold synthesised as +1 px
advance per character.

---

## 16. Determinism and the RNG

Authority: `determinism.md`, with the normative ruling in `toolbox-primitives.md` Part 1.

* **R-RNG-1**: `Random()` is the Lehmer / Park-Miller minimal-standard generator on
  `qd.randSeed`, `a = 16807`, `m = 2147483647`, computed with Schrage decomposition, returning
  the **low 16 bits of the new state reinterpreted as `int16`**. Initial seed is **1** (Carbon
  leaves `randSeed == 1`). A verified seed-1 stream is tabulated at `toolbox-primitives.md`
  §1.6.
* **R-RNG-2**: the shipped `demo` 128 replay is **RNG-independent**. R-RNG-1 is needed only for
  pixel-exact *animation* (candle flames, page flutter, telephone chimes), never for
  trajectory. Containment proof at `toolbox-primitives.md` §1.13.
* `RandomInt` (`Utilities.c:72-82`) is `abs(Random()) * range / 32768`. Its **upper bound is
  inclusive** — `RandomInt(6)` can return 6 — and its distribution is skewed. Reimplement it
  exactly; do not substitute a modular reduction.
* **R-XOR-1**: the single `srcXor` in the tree (`Marquee.c:438`) is `dest[i] ^= 0xFF` where the
  1-bit source pixel is set, `^= 0x00` where clear.
* **R-CLUT4-1**: the depth-4 device CLUT is `entry[i] = ((15-i)*0x1111, ...)` for i in 0..15.

---

## 17. Persistence map

`progression.md` §17 is the authority. One-line summary:

| Lives in | Contents |
|---|---|
| The house image in memory (survives room change, death, and process exit if saved) | `visited` flags, object `state` bytes, `highScores`, `savedGame` |
| Globals rebuilt on every room change | `thisRoom` copy, `masterObjects[216]`, `hotSpots[56]`, `dinahs[18]`, `localNumbers[9]`, the animation arrays |
| Globals persisting across rooms for the whole game | `theScore`, `mortals`, `batteryTotal`, `bandsTotal`, `foilTotal`, `numStarsRemaining`, `showFoil`, `onePlayerLeft` |

`SetObjectsToDefaults` (`Play.c:603-708`) is the state reset; note what it implies for "opened
doors" and "lit lights" (`progression.md` §6.5-6.6). Two special cases a port must not smooth
over (`progression.md` §17.1):

* A `kStereo` (0x69) is reset to **`isPlayMusicGame` — the player's *music preference* — not
  `data.g.initial`** (`Play.c:675-677`), unlike the whole
  `kShredder`..`kMicrowave` group at `:679-690`. A user preference therefore leaks into reset
  object state; §16's determinism guarantees do not cover it, so a networked or replayed port
  must transmit or pin this flag.
* A `kDeluxeTrans` (0x40) restores its low (live) nibble from the **high (initial) nibble** of
  `data.d.wide` (`Play.c:657-661`), not from a separate `initial` field.

---

## 18. Original bugs a faithful port must decide about

These are not port bugs; they are shipped behaviour. Reproduce, clamp, or guard — but decide
deliberately.

| Bug | Effect | Citation |
|---|---|---|
| `srcRects[0x85 kFlower]` never assigned, array not zeroed | Heap garbage where a rect should be | `StructuresInit2.c:271`, `:306-475` |
| `DoGameOverStarAnimation` computes `which = (angelDest.left / 32) % 5` while `left` is negative (`angelDest` is offset by -96 at `:146-147`) | `which` = -3, -2, -1 on the first three firings, and `ZeroRectCorner(&pages[which].dest)` / `QOffsetRect` then write **below** `pages[]` (`:161-162`). Visible effect: no star until x reaches 0. **Clamp; do not reproduce the overrun** | `GameOver.c:146-147`, `:159-162` (in `DoGameOverStarAnimation`, `:136-230`); `scoring.md` bug 14 — which carries the same wrong `:326` line number and should be fixed there too |
| `InitDiedGameOver` uses screen **width** where it wants height | "GAME OVER" pages start at a vertical offset derived from horizontal resolution; visible on non-4:3 screens | `GameOver.c:290-291` |
| `FireTrigger`'s `else` branch indexes `masterObjects[-1]` | The `else` branch is **entered** by exactly one shipped trigger (Land of Illusion room 1 object 16 → room 174 object 15), but the `masterObjects[-1]` read lives inside its `kGreaseRt`/`kGreaseLf` case (`Triggers.c:187-188`) and that target slot is **empty** (`what == -1`), so the wild read never actually happens with shipped data. **Early-return when `localLink == -1`** — behaviour-preserving for the shipped corpus | `Triggers.c:174-193`; `object-dynamics.md` §8.12 (which supersedes its own weaker OQ3) |
| `SetObjectState` has no `case kKnifeSwitch:` and no `default:`, with `changed` uninitialised | Two shipped links hit it; believed unobservable | `Objects.c:369`, `:533-542`, `:695` |
| Demo playback reads one record past the end of `'demo'` 128 | Benign on `NewPtr` slack; a panic in Go | `format-decisions.md` D7 |
| `RoomInfo.c:514` and `:555` omit the `tempBack - 800` thumbnail substitution that `:416-420` applies | Re-picking background 2002/2011/2016/2017 silently switches to the scaled full-size thumbnail. **Apply uniformly** | `graphics-assets.md` §7.8 |
| Two-player win bug: `FlagGameOver()` fires the moment `numStarsRemaining` hits 0, **regardless of `mortals`** | If the surviving player takes the last star after `mortals` has already reached -1, the dispatch at `Play.c:546` sees `mortals < 0` and runs `DoDiedGameOver()` — the losing "GAME OVER" flutter on what is actually a victory, with no trailer text and no angel. The high score is still recorded | `Interactions.c:933-951`, `Play.c:546`; `scoring.md` §8.9 |
| `ReadScoresFromDisk` sizes the read with `GetEOF` (`:823`) and `FSRead`s that many bytes straight into the 292-byte `highScores` field (`:841`) | Would overrun the locked house handle for any side-car larger than 292 bytes. Guarded by `houseIsReadOnly` (`HouseIO.c:428-434`), which is always false because `IsFileReadOnly` is an unconditional `false` stub (`HouseIO.c:659-663`), so unreachable in this build — reinstating the stub's commented-out body makes it live. Bound the read | `HighScores.c:801-855`, `:823`, `:841`; `HouseIO.c:187`, `:428-434` |
| Mask `PICT` 5005 is one row shorter than art 4005 (80x268 vs 80x269) | Only consumer of row 268 is `tvScreen2`, blitted opaque. Treat a missing mask row as transparent | `graphics-assets.md` §7.6 |

---

## 19. Fidelity contract

The behaviours that **must** match the original or the port will not feel like Glider PRO.
Ranked: #1 is the thing that makes it unplayable if wrong, #20 is the thing a fan would notice
on a second playthrough.

| # | Must match | Why it matters | Specified by |
|---:|---|---|---|
| 1 | **Fixed 30.07 Hz integer step, no delta time, no catch-up.** Every velocity is literally px/frame; `nextFrame = TickCount() + 2` recomputed from now | Any interpolation or variable timestep changes every trajectory in the game | `architecture.md` §7; `determinism.md` §1 |
| 2 | **`MoveGlider`'s desired-velocity ramp**: `hDesiredVel` → 0 and `vDesiredVel` → `kGravity` (3) *every* frame; ramp ±2; horizontal clamp ±16 applied **inside each horizontal move branch**, with `wasHVel` written from the clamped value; **no vertical clamp** | This is the entire feel of the glider. Terminal fall (3 px/f) and cruise (5 px/f) are emergent, not constants | `player-physics.md` §4, §7; `Player.c:64-147` |
| 3 | **Battery bypasses the ramp** (`hVel += 8` directly, clamped to ±16) while helium and keys set targets | Battery is the only "burst" in the game; ramping it makes it feel dead | `input.md` §4.5 |
| 4 | **Room geometry is exactly 512 x 322, unscaled, 3x3 neighbourhood, 20-px scoreboard**; `playOriginV` derives from `screen`, not `houseRect`, so the room sits 10 px below centre. The window's 20-px menu-bar loss and the offscreen's 20-px `kScoreboardTall` loss are **unrelated quantities that happen to be equal** | Every hard-coded object y (`kFloorVentTop` 305, `kShadowTop` 306, …) assumes it | `rendering.md` §2, §2.2 |
| 5 | **The nine boundary limits and `CheckGliderInRoom`'s exact structure**: modes 0/7/8/9 only; separate vertical and horizontal if-chains, the horizontal one against the **room-dependent** `leftThresh`/`rightThresh` (-24/536 open, 12/500 walled); bounce reflection amplified by penetration depth; solid-floor landing is fatal unless a manhole set `ignoreGround` this frame | Doorways, bounces, and every death-by-floor depend on it; hard-coding 12/500 changes rooms 36 px early | `interactions.md` §12 |
| 6 | **The exact per-frame order of `PlayGame`** — telephone, **dynamics before input**, player-1 input then player-2 input, one `HandleInteraction` for both, triggers and bands **after** interaction and outside the `!gameOver` guard, then `HandleGlider` (the only caller of `MoveGlider`), then `RenderFrame`. Player-1-first everywhere: input, hot-spot testing, `CheckGliderInRoom`, and prize pickup (first-caller-wins in `SetObjectState`) | Enemies move against last frame's glider position; reordering changes every trajectory, and two-player outcomes diverge | §6.3; `Play.c:445-497`; `player-physics.md` §17; `determinism.md` porting notes 1, 4 |
| 7 | **Level-polled input at exactly one sample per frame, shared between both players**, with `fireHeld` as the only synthesised edge | No key repeat, no queue latency; bands fire once per press | `input.md` §2, §9.1 |
| 8 | **Battery and helium share one signed counter, with destructive asymmetric top-up** | Picking up the wrong consumable is a real tactical loss in the original | `scoring.md` §4.2 |
| 9 | **Lives semantics**: `mortals` counts *spares* (2 → 3 deaths); the only extra life is `kPaper`; no cap; no score-threshold life; doubling only while `!onePlayerLeft` | Difficulty pacing | `scoring.md` §3 |
| 10 | **Room-visit scoring fires on departure**, writing both copies of `visited` | The starting room's 100 points arrive late; total-score parity with `CountTotalHousePoints` depends on it | `scoring.md` §2.3 |
| 11 | **The 28 hot-spot actions are the only channel by which the world affects the player**, with `hotSpots[56]` rebuilt during the drawing pass | Getting the emission order wrong changes which object wins a contested overlap | `interactions.md`; `object-draw-all.md` |
| 12 | **Present-before-erase dirty-rect discipline** and the exact `RenderFrame` layer order (reflections, grease, pendulums, flames-or-stars by `evenFrame`, dynamics, flying points, sparkles, glider(s), shreds, bands) | Z-order and one-frame trails are visible | `architecture.md` §7; `rendering.md` §10 |
| 13 | **`evenFrame` half-rate gating** (flames on even, stars on odd) | Animation cadence | `Render.c:639-671` |
| 14 | **Score roll at 13 points/frame** | "Reproduce the score roll exactly or the game feels wrong" | `scoring.md` §5.1, §B |
| 15 | **The four masking strategies**: a 1-bit mask `PICT` wins wherever one exists (bit 1 = opaque; the mask carries information the colour plane does not — 681 pixels of the angel 1019 are white *and* opaque), the white colour key (**palette index 0**, not an RGB test) applies only to the 21 colour-keyed objects, and the `switch` sheet 4003, the backgrounds, the scoreboard and the 9 opaque objects use plain `srcCopy`; plus the three procedurally-drawn objects (`kMirror`, `kCounter`, `kWallWindow`) | Wrong transparency is instantly visible; substituting a colour key for a mask punches holes in the art | `rendering.md` §8; `graphics-assets.md` §5, §7.4 |
| 16 | **Sound priority arbitration across 3 effect channels**, with the exact band values, and mono output | Audio behaves as a fixed hierarchy, not a mixer | `audio.md` §4-5 |
| 17 | **Both music sequences** over `'snd '` 2000-2006: the idle/menu `musicScore[16] = {0,1,2,3,4,4,0,1,2,3,4,4,5,6,4,4}` **and** the in-game `gameScore[6] = {6,5,-1,6,4,4}`, with the `-1` back-jump and the wrap to index **1** | Long-session texture; `musicScore` alone plays the wrong music during gameplay | `audio.md` §7; `Music.c:349-354`, `:183-189` |
| 18 | **Byte-exact house round-trip**: preserve every reserved field, every `StrNN` tail, `Sampler`'s +2 slack; parse at `866 + 348*r` | The port must not corrupt user houses | `house-format.md`; `format-decisions.md` D2-D4 |
| 19 | **Hard budget limits are behaviour**: 24 objects/room, 18 dynamic, 24 saved maps, 56 hot spots, 2 bands, 3 sparkles, 3 flying points, 4 stars. Exhaustion is observable, not an error | Busy rooms in shipped houses actually hit these | `progression.md` §15.3; `object-draw-all.md` |
| 20 | **R-RNG-1's exact stream and `RandomInt`'s inclusive, skewed range** | Only pixel-level animation depends on it, but that includes candle flames in most houses | `toolbox-primitives.md` Part 1 |

Explicitly **not** in the contract (safe to fix or drop): the `masterObjects[-1]` wild reads,
the `pages[]` underflow, the demo's read-past-end, the `Random()` seeding alternatives, the
`AppendResMenu('DRVR')` items, the dead saved-game subsystem, `DrawLighting`'s no-op stub, and
the `idleMode` variable.

### 19.1 The contract, audited (stage 1.8c)

Stage 1.8's acceptance criterion is that every row above is either satisfied or has a written
exception. This is that audit. It is a *documentation* pass over work the earlier stages did, not
a claim that 1.8c implemented twenty things: the point of writing it down is that "the port is
faithful" stops being an assertion and becomes twenty citations a reader can check, and that the
five places where the port knowingly differs are on the record instead of in someone's head.

The rule for "held" is deliberately strict — a row counts as held only if a test fails when the
behaviour changes, *naming what changed*. A transcription with a good comment and no test is not
held, and neither is one whose only witness is the pixel corpus: the corpus reports that 600 frame
hashes moved and does not say why. Applying that rule found one row not held. Row 4's
`playOriginV` derivation had no assertion anywhere — every other test in the repository takes the
origin *from* the view rather than stating it, which is the right dependency direction and is
exactly why nothing checked the view itself. `internal/render/view_test.go` is 1.8c's answer, and
it was verified to fail (with the 10-px message) on the one-word change from `Screen` to `House`
before being kept.

| # | Row | Held by | Pinned by |
|---:|---|---|---|
| 1 | 30.07 Hz integer step, no catch-up | `internal/game/render_frame.go` (`awaitFrame`) | `game.TestPlayGameAdvancesTheTwoClocks`, `game.TestRenderFrameOrder` (`awaitFrame`'s position), `fidelity.TestFramesAreStableWithinARun` — **exception (a)** |
| 2 | `MoveGlider`'s desired-velocity ramp | `internal/game/player/glider.go` | `player.TestFreeFallTrace`, `TestHoldRightTrace`, `TestHeliumTrace`, `TestNoVerticalClamp`, `TestHorizontalClampInsideSignBranches`, `TestWasVelWrittenUnconditionally`, `TestWholeIsSweptNotAccumulated` |
| 3 | Battery bypasses the ramp | `internal/game/player/input.go` | `player.TestBatteryFollowsTheBank`, `TestHeliumIsTheNegativeHalf`, `TestBatteryFizzlesAtZero`, `TestThrustSoundEveryFourthFrame` |
| 4 | 512x322 unscaled, 3x3, 20-px board | `internal/render/view.go` | `render.TestTheOriginComesFromTheScreenAndNotTheHouseRect`, `TestARoomIsFiveHundredAndTwelveByThreeHundredAndTwentyTwo`, `TestTheHouseRectIsTheScreenLessTheScoreboard` (**new in 1.8c** — this was the one row the audit found *un*held), `TestScoreboardGeometry640`, `TestScoreboardTwoOffsets`, `TestComposeEveryRoom` |
| 5 | The nine limits, `CheckGliderInRoom` | `internal/game/player/escape.go` | `player.TestTwoThresholdsPerBoundary`, `TestWallBouncesToTheWallFace`, `TestOpenSideNeedsNoClearance`, `TestFloorIsLethalWithoutPermission`, `TestIgnoreGroundFallsThrough`, `TestNonInRoomModesAreNotBounded`, `TestCornerExitGetsTwoVerdicts` |
| 6 | `PlayGame`'s per-frame order | `internal/game/play.go` | `game.TestPlayGameOrder` — the call sequence *and* each call's guard set, read out of the AST (**new in 1.8c**) |
| 7 | One input sample per frame, shared | `internal/game/play.go` (`GetInput`), `internal/game/player/input.go` | `game.TestPlayGameOrder` (one poll per glider per frame), `player.TestBandDebounce`, `TestRefusedBandCostsNothing`, `game.TestGetDemoInputOnlyPollsPlayerOne` — **exception (b)** |
| 8 | One signed battery/helium counter | `internal/game/rewards.go` | `game.TestBatteryAndHeliumShareOneSignedCounter`, `TestFifteenRewardArms` |
| 9 | Lives semantics (`mortals` = spares) | `internal/game/play.go`, `internal/game/mortal.go` | `game.TestNewGameSetsUpTheGliderAndTheGame`, `TestTwoPlayerInitGliderDoesNotDoubleTheLives`, `TestTwoPlayerDoublesTheFiveSupplies`, `TestGliderCountClamp`, `player.TestIdleCountdownLivesInHVel` |
| 10 | Room-visit scoring on departure | `internal/game/transit.go` | `game.TestTheRoomCountAgreesWithTheScore`, `TestCountRoomsVisitedCountsTheFlagAndNothingElse`, `replay`'s duct trace (score 100 → 200 on the transition) — **exception (c)** |
| 11 | 28 hot-spot actions, `hotSpots[56]` | `internal/game/objects.go`, `internal/render/locale.go` | `game.TestHotSpotTypeCoverage`, `TestSetObjectStateEveryType`, `TestScrutinizedActions`, `TestObjectGraphEveryRoom`, `TestHotSpotTableOverflowReturnsMinusOne` |
| 12 | Present-before-erase, `RenderFrame` order | `internal/game/render_frame.go` | `game.TestRenderFrameOrder` (the layer order out of the AST, plus the flames-or-stars parity branch and player 1's z-order over player 2), `TestPlayGameKeepsPublishingFrames`, `fidelity.TestFrames` (600 hashed frames) — **exception (d)** |
| 13 | `evenFrame` half-rate gating | `internal/game/render_frame.go`, the movers | `game.TestEvenFrameGatingIsPerHandler`, `TestBallAndFishRegistrationSetEvenFrame`, `TestRenderFlamesEntersForAnyOfItsThreeTables` |
| 14 | Score roll at 13 points/frame | `internal/game/scoreboard.go` | `game.TestScoreRoll`, `TestScoreRollJumpsWhenDisarmed`, `TestScoreRollDrawsTheIntermediateNumber`, `TestRefreshScoreboardArmsAndEmptiesTheRoll` |
| 15 | The four masking strategies | `internal/render/locale.go`, `internal/render/objectdraw2.go` | `render.TestTheMaskingStrategyOfEveryObjectType` — the 21/9/2/3 census read out of the dispatch (**new in 1.8c**) |
| 16 | Sound priority over 3 channels, mono | `internal/audio/engine.go` | `audio.TestLowestPriorityWins`, `TestRepeatedSoundSpreadsAcrossChannels`, `TestTriggerExclusivity`, `TestCompletionCallbackFreesTheChannel`, `TestMixSumsAndClips` |
| 17 | Both music sequences | `internal/game/music.go`, `internal/audio/music.go` | `game.TestScoreTables`, `TestWholeScoreWrapsEarly`, `TestGameScoreSettlesOnOneRefrain`, `TestProdGameScoreMode`, `audio.TestMusicQueueAndChain` |
| 18 | Byte-exact house round-trip | `internal/house` | `house.TestCorpusRoundTrip` (every shipped house, byte for byte), `TestCorpusSlack`, `TestCorpusNonCompacted`, `TestCorpusRoomInvariants` |
| 19 | The hard budget limits | `internal/game/objects.go`, `internal/render/anim.go` | `game.TestTablesRespectTheirCaps`, `TestHotSpotTableOverflowReturnsMinusOne`, `render.TestEachFamilyIsRefusedAtItsOwnCap`, `TestTheSavedMapBudgetSaturatesInShippedContent`, `TestASaturatedTableDoesNotConsumeARandomDraw` |
| 20 | R-RNG-1's stream, `RandomInt`'s range | `internal/game/rand.go` | `game.TestRandomMatchesTheVerifiedSeed1Stream`, `TestRandomIntUpperBoundIsInclusive`, `TestRandomIntSkewOverEveryRawWord`, `TestNaiveInt32ModularWouldHaveDiverged`, `TestAdvanceRandSeedIsOneDraw`, `TestThePhysicsNeverDrawsFromTheRNG` (**all new in 1.8c**) — **exception (e)** |

#### The five exceptions

**(a) Row 1 — the wait is slept, not spun.** The C's limiter is `while (TickCount() < nextFrame)
{ }`. `awaitFrame` keeps the condition and the reseed-from-now (so there is still no catch-up) but
takes a `World.WaitTick` hook for the loop *body*: nil is the C's empty body and is what the
fidelity and replay builds use, and `cmd/glidergo` installs a 1 ms sleep. That changes how the
wait is spent, not when it ends. The open half of IMPROVEMENTS 2.17 — a "keep real time" setting
that skips frames on a stall — would be a real contract change, which is why `prefs.KeepRealTime`
is declared, defaults to false, and is still read by nothing.

**(b) Row 7 — the shared key map is a hook, not a file-scope global.** The C polls
`GetKeys(theKeys)` once into a file-scope `KeyMap` and both players read the same buffer.
`World.KeyPoll` is called once per glider and returns that glider's already-resolved `player.Keys`
(the three rebindable keys resolved on the way through), so the port has two calls where the C has
one buffer. The observable behaviour is the same — one sample per frame per glider, no repeat, no
queue — but a host that returned *different* key states to the two calls in one frame would
diverge from the original in a way no test here can see, so the contract lives in the hook's
documented obligation rather than in the type.

**(c) Row 10 — one `visited` write, not two.** The C writes `visited` into the house handle and
again into the `thisRoom` working copy, because those are two structures with the same field.
`World.ThisRoom()` returns a pointer into the house, so the port has nothing to keep in sync and
the two writes collapse into one that cannot go stale. Both of the C's indices name the same room
at every call site (`DrawLocale` sets `localNumbers[kCentralRoom]` from `thisRoom` and nothing
between them can run), so this is unobservable — but the row says "both copies", so it is written
down.

**(d) Row 12 — the quit path does not repaint the splash into the work map.** `NewGame`'s tail
(`Play.c:257-273`) ends a quit game by scaling the splash art into `workSrcMap` and invalidating
the window, so the Mac's next update event paints the title screen out of it. Here the title
screen belongs to `internal/shell` and is composed on the shell's own surface, so not one of those
294,400 pixels is reachable; the write is dropped deliberately (IMPROVEMENTS 2.62). It buys back a
harness invariant worth more than the pixels — with the work map left as the last frame drew it,
`replay`'s plane hashes can insist that a still room's work map equals its background map, which
catches exactly the draws that forget a back rect. The two *ending* paths keep their restore,
because there the splash is on screen a moment later. Nothing in the frame order moves; the
omission is one call after the loop has ended.

**(e) Row 20 — three residual differences in the random stream.** The generator, the seed and the
scope all match (`rand.go`, and `cmd/glidergo`'s process-lifetime `app.randSeed` with
`game.AdvanceRandSeed` accounting for `VariableInit`'s launch draw), but:

1. `WriteOutPrefs`'s `thePrefs.fakeLong = Random()` (`Main.c:232`) is **not** emulated. It only
   runs at startup when the copy-protection check rewrote the prefs file (`Main.c:315`
   `didValidation`), so whether the original's first game starts one draw further along depends on
   state that is not in this tree. A port cannot be faithful to both branches; it is faithful to
   the one that does not require guessing.
2. `PourScreenOn`'s `RandomInt(colWide)` rejection loop (`Transitions.c:44`) is dead code in the
   shipped build and is deliberately not transcribed (`internal/game/screen.go`), so it consumes
   no draws here. If it were ever reached in the original it would consume an unbounded number.
3. Whether Apple's `Random()` trap really was Park–Miller-by-Schrage is **unverifiable without a
   Mac.** §1.5 of `toolbox-primitives.md` argues it from Apple's own documentation and §1.13
   bounds the cost of being wrong: animation phase, never trajectory. R-RNG-2 proves the shipped
   demo replay is RNG-independent, and `game.TestThePhysicsNeverDrawsFromTheRNG` proves the same
   thing about this port from the other end — the physics package contains no draw at all. See
   IMPROVEMENTS 2.18.

---

## 20. Known unknowns

Merged from the `## Open questions` section of all 28 docs (27 actually have one;
`unresolved-format-decisions.md` is the redirect index and has none), deduplicated against
`unresolved-format-decisions.md` (which closes 30 items, confirms 10, narrows 4 and corrects
2), and ranked by how much each blocks work.

### Tier 1 — blocks a subsystem until decided

| # | Unknown | What would settle it | Interim rule |
|---:|---|---|---|
| 1 | **MACE 6:1 quantisation tables.** 5 house sounds are `cmpSH` / `compressionID` 4 and cannot be decoded from anything in this tree | Obtain the published MACE tables from an open-source Mac audio decoder, or re-record | Ship those 5 as silence (`audio.md` OQ3) |
| 2 | **Frame rate of the 14 headerless `.mov` files.** Their `mvhd`/`mdhd` `timeScale` and `stts` died with the lost `moov` | An original copy of any of the 14 with its resource fork | 5 fps uniformly (`quicktime-movies.md` §19.5, OQ1) |
| 3 | **`Grand Prix.mov`'s ~59-entry custom colour table**, which lived in the `ImageDescription` | An original file, or a screenshot of the running game | Fall back to `clut` 128 (`quicktime-movies.md` OQ2) |
| 4 | **Application font metrics.** No font resources exist in the fork; every pixel position in `scoring.md` §5/§7 was derived from code, not a screenshot. "The largest single source of visual divergence in the whole subsystem" | A screenshot of a running 1.1.2, or Geneva 9/12 metrics | R-FONT-1's Liberation Sans substitution, with the measured cost bounded in `scoring.md` Appendix Z |
| 5 | **`hotSpots[]` index stability across a `DrawLocale` / `RedrawRoomLighting` redraw.** `activeRectEscaped` saves an index in one frame and compares it in a later one (`Interactions.c:42`, `:1405`, `:1452`, `:1495`, `:1534`, `:1569`) | Tracing whether `RedrawRoomLighting` rebuilds hot spots | **Key the transporter handshake on `(roomNum, objectNum)` instead of an index** (`interactions.md` OQ1) |
| 6 | **When `masterObjects[].dynaNum` becomes valid.** Filled during the drawing pass (`ObjectDrawAll.c:952-960`), so it is -1 until `DrawLocale` completes; `SpillGrease(dynaNum, …)` on a room's first frame would index `grease[-1]` | Confirming `DrawLocale` always precedes the first `HandleInteraction` | Guard the index (`interactions.md` OQ9) |

### Tier 2 — affects correctness in identifiable cases

| # | Unknown | Interim rule |
|---:|---|---|
| 7 | **`tiles[i] == 4` in `kRoof` rooms** (background `PICT` 2014) is present in real roof rooms (CD Demo House r138, r148, r149) yet `CheckRoofCollision`'s chain covers only 1, 2, 5 and 6, so its final `else` kills the glider (`Interactions.c:498-503`) | Inspect the tile art; treat as solid-and-avoidable pending evidence (`interactions.md` OQ3) |
| 8 | **`kRoof` rooms have a floor but no shadow.** `DoesRoomHaveFloor` returns false only for sky/stratosphere/stars, but `IsShadowVisible` also excludes `kRoof`, so a roof glider casts no shadow yet still dies at `kFloorLimit` | Reproduce as-is (`player-physics.md` OQ4) |
| 9 | **Does `flushCmd` discard an already-queued `callBackCmd`?** Decides whether `FlushAnyTriggerPlaying` permanently wedges a voice at priority 999 | Reset the priority in the flush (`audio.md` OQ1) |
| 10 | **What sets `hotSpots[].stillOver` for `kMicrowaveIt`?** Nothing found; the guard in `HandleMicrowaveAction` may be dead | Either reading differs only in repeated `kMicrowavedSound` (`enemies.md` OQ1) |
| 11 | **Is `kSlider` (0x2F) ever a live hot spot?** `HandleRewards` has a bare `case kSlider: break;` and `SetObjectState` falls through with `changed` uninitialised. No sampled house has one | Exhaustive scan before implementing (`interactions.md` OQ4) |
| 12 | **Two-player `stillOver` semantics**: `CheckForHotSpots` clears it only when *neither* glider overlaps, so player 2 standing in a switch blocks player 1 from re-triggering it | Reproduce; verify against a recording if one ever appears (`enemies.md` OQ8) |
| 13 | **`kChimeIt` (action 25)** has a `HandleHotSpotCollision` case but no located `AddActiveRect` emitter | Implement the case; treat as unreachable (`interactions.md` OQ7) |
| 14 | **Two-player stairs code transition -8 → -6** happens somewhere in `Player.c`'s stairs mover; the exact line was not located | `interactions.md` OQ8 |
| 15 | **Authoritative meaning of `roomType.floor` / `roomType.suite`** — used for map placement, no collision consumer found | Treat as purely navigational (`interactions.md` OQ2) |
| 16 | **The bare literals `315` (`Player.c:704`) and `29` (`Player.c:336`)** in duct/stairs movers. `29` is exactly `kGliderStartsDown - 3`; `315` matches no constant | Transcribe verbatim, do not "fix" (`player-physics.md` OQ2-3) |
| 17 | **`kUserBackground` upper bound**: `DITL` 1016 says 3499, the constants say 3300, the validation allows 3000-3799 | **Accept 3000-3799**, matching the code that runs (`resource-fork.md` OQ4) |
| 18 | **Does any shipped house use `kCustomPict` or `kUserBackground`?** Not measured per-object | Implement the `LoadGraphicSpecial` fallback chain to `PICT` 2000 (`graphics-assets.md` OQ6) |
| 19 | **The `'Date'` resource type** is accepted in place of a `PICT` by `LoadGraphicSpecial` and `PictIDExists`, but no such resource exists anywhere in this tree | Treat as "ID exists, no pixels" (`graphics-assets.md` OQ1) |
| 20 | **`RectWide(&dartSrc[0])` vs `srcRects[kDartLf]`** — both 64, but respawn uses one and the editor clamp the other | Would only diverge with custom enemy art, for which no mechanism was found (`enemies.md` OQ4) |
| 21 | **`kFish.delay == 0` reachable?** Shipped fish have `delay` down to 0, giving `timer = 0` at spawn — a fish that leaps every frame | Believed intended ("continuous fish") (`enemies.md` OQ5) |
| 22 | **`CenterRectInRect`'s negative division**: is `(49-62)/2` −6 or −7 in the shipped binary? | CodeWarrior and MPW C truncate → −6, and Go matches (`quicktime-movies.md` OQ5) |

### Tier 3 — cosmetic, vestigial, or historical; does not block anything

| # | Unknown |
|---:|---|
| 23 | Fields with **no reader anywhere**: `blowerType.tall`, `furnitureType.pict`, `data.f.byte0/byte1`, `data.h.byte0`, `dinahs[].byte1`, `triggers[].what`, `soundLoaded[]`/`numSoundsLoaded`, `wasVVel`, `demoType.padding`, `nullCmd`'s `param1 = 1964`. Drop the ones that would cause out-of-bounds reads (`triggers[].what`); preserve the on-disk ones verbatim |
| 24 | Why `srcRects[kFlower]` is the only gap *inside* a class; whether the `switch` sheet was ever masked; why 20 mask `#define`s (3900-3926) survive with no resources (the 20 are exactly the 20 colour-keyed objects) |
| 25 | `PICT` 10000 (4,182 bytes) — no `#define` refers to it; presumed placeholder. `PICT` 1994, 1998, 1021, and the `ozm5` resource are bucketed by elimination, not by a located call site |
| 26 | `'DLOG'` 150's three extra bytes (`00 02 2F`) and `'DLGX'` 150's 190-byte per-item table — both inert |
| 27 | Whether the `PackBitsRgn` mask region is ever non-trivial (all 3 occurrences equal the picture rect) |
| 28 | The badge blank strip's last row is palette index 250 in one place and 172 in another |
| 29 | `Environ.c`'s memory estimate produces 6396 where 6024 is the true pixel count — a "do we have RAM" approximation with no behavioural effect, but it feeds screen metrics |
| 30 | Historical residue formally closed by `format-decisions.md` but never *explained*: what `houseType.unusedShort` (+2) was intended for; why `roomType.unusedByte` needed force-zeroing in two places; which of the 22 houses shipped on the retail disk; which compiler produced `Sampler`'s 868-byte header; whether `Demo House.binhex` is the exact file `'demo'` 128 was recorded against; whether the demo's `padding` residue encodes anything |
| 31 | No **version-1 house** exists in the corpus, so `ConvertHouseVer1To2` (`House.c:746-818`) and the inferred v1 `MergeFloorSuite` = `(floor + 8) * 100 + suite` are verified only by algebra |
| 32 | `kNewHouseVersion` (0x0300) is only ever compared against (`HouseIO.c:395`), never written. No 3.0 format exists to document |
| 33 | `Fun House` room 29 has `where = 5812`, an anomalous link encoding (`original-houses.md` OQ4) |
| 34 | Does any shipped house depend on the RNG for a *reachable* outcome? `HandleCoffee` and `HandleSparkleObject` are RNG-timed in principle; unsurveyed (`determinism.md` OQ3) |
| 35 | The **exact frame at which the demo glider dies**. Frame 3414 is the last *input*, not the death. This is a port **acceptance test**, not a prerequisite |
| 36 | Whether all 13 sound-bearing houses conform to the `'snd '` +20-byte skip assumption; a malformed one would produce noise rather than an error |
| 37 | Whether the four glider atlases (`PICT` 3963/3974/3976/3999) truly have pixel-identical silhouettes, since `InitGliderMap` shares one mask (4999) across all four |

---

## 21. Asset pipeline

What must be extracted, in what order, into what target formats, and what already exists in
`tools/`.

### 21.1 The extraction scripts

`tools/extract_all.py` is the build step; everything else is an inspection CLI that also
exposes its extractor as a function the driver calls. One command produces the whole tree:

```
python3 tools/extract_all.py            # or: make assets
```

→ `assets/extracted/{res,art,sound,houses,movie}/` + `manifest.json`, **908 files, 36 MB, 16 s**.
(1.3 added `houseart/`; the tree is now 1,899 files and about 70 s.) It is derived data and this
script is the derivation, but it is *committed* anyway, so that a clone can play without running
it — see docs/IMPROVEMENTS.md 1.2. Two runs over the same `GliderPRO/` produce byte-identical
output, verified: no wall-clock anywhere in the manifest, and `make assets-check` re-extracts to
a temp directory and compares every file against the committed tree.

| Script | Size | What it does |
|---|---:|---|
| `tools/extract_all.py` | 13,170 B / 324 lines | The driver. Runs the five buckets in dependency order, hashes every input out of `GliderPRO/` for provenance, records a `tree_sha256` per bucket, and checks 9 counts (538 resources / 35 types / 152 PICTs / 18 backgrounds / 70 app sounds / 58 house sounds / 22 houses / 4,070 rooms / 15 movies) against the numbers `docs/analysis/` counted independently. A mismatch exits non-zero rather than quietly shipping fewer assets |
| `tools/probe_rez.py` | 11,794 B | Parses the `Glider PRO.r` DeRez dump into typed resources. Character-walking header parser at `:78`; **must read in binary mode** (`:145`). Sub-command `extract` writes `<out>/<TYPE>/<id>.bin` |
| `tools/probe_house.py` | 21,042 B | BinHex 4.0 decode + Mac resource-map parsing. `binhex_crc` alphabet assert at `:29`; fork CRC checks at `:137`/`:140`. **Pass `check_crc=True`** (defaults to False); resource key is `rsrc` |
| `tools/probe_pict.py` | 37,019 B | The PICT decoder. `Picture.parse` → `rasterise` (`:702`) returns `(W, H, RGB, info)` with `info["idxmap"]` the **palette-index plane**. `clut` parsing at `:824`, 256-entry RGB conversion at `:417`, defensive grey ramp at `:728-731` |
| `tools/extract_art.py` | 39,221 B / 652 lines | The whole art pipeline. Embeds the sheet table (`SHEETS`), all 116 atlas rows (`ATLAS`), the strip/background/UI/misc tables, `ALLOWED_OPS` as a frozenset and the four alpha functions (`rgba_from_mask`, `rgba_from_key`, `rgba_opaque`, `crop`). `check_accounting()` partitions all 152 `PICT`s into the §7.8 buckets and raises on a duplicate, an unclassified id, a phantom id or a wrong total |
| `tools/probe_snd.py` | 20,735 B / 499 lines | Sub-commands `show`, `list`, `houses`, `extract`, `extract-houses`, `wav`. `extract` produces **70 `.pcm` files, 1,126,404 bytes**; `extract-houses` produces **58 `.pcm` files, 2,133,612 bytes** and records the 5 MACE resources as explicit skips |
| `tools/probe_houses_inventory.py` | 38,758 B | Generates `docs/analysis/houses-inventory.md` from the 22 house files |
| `tools/probe_mov.py` | 59,692 B | QuickTime container + `rle `/`raw `/`smc ` decoding for the 15 house movies |

### 21.2 Ordered extraction steps

**Step 0 — normalise the source tree for citation.**
`for f in GliderPRO/Sources/*.c GliderPRO/Headers/*.h; do tr '\r' '\n' < "$f" > /tmp/norm/$(basename $f); done`
Not an asset step, but every citation in this document and in `docs/analysis/` is a line number
in that copy.

**Step 1 — split the resource fork.**
`python3 tools/probe_rez.py extract /tmp/gp/res GliderPRO/'Glider PRO.r'`
(**note the argument order: destination first, then the dump.** `cmd_extract(path, outdir)` is
reached as `extract <outdir> [<rez>]` — `main` passes `argv[3]` as the path and `argv[2]` as the
output directory, `probe_rez.py:322`, and the rez path defaults to `DEFAULT_REZ` if omitted.
Getting it backwards fails with `FileNotFoundError` on the output directory.)
→ `<res>/<TYPE>/<id>.bin`, one file per resource, 538 files across 35 type directories. Types
with filename-illegal characters are sanitised: **`'snd '` becomes the directory `snd_`**.
Expect 3,168,309 bytes of payload total. Authority: `resource-fork.md` Part 2.

**Step 2 — art: PICT → RGBA PNG + manifest.**
`python3 tools/extract_art.py /tmp/gp/res /tmp/gp/art`
Internally three stages: decode each `PICT` to an index plane (9 opcodes, everything else
fatal); apply one of four alpha rules; crop by the atlas table. Measured output: **179 PNGs plus
`manifest.json`** — `sheet/` 14, `object/` 94, `strip/` 14, `bg/` 18, `ui/` 38, `misc/` 1.
The run begins with `check_accounting()`, which partitions all 152 `PICT`s in the fork into the
§7.8 buckets (`sheet_art` 14, `sheet_mask` 13, `strip_art` 14, `strip_mask` 8, `obj_art` 38,
`obj_mask` 8, `bg` 18, `ui` 38, `misc` 1) and raises if any id is unclassified, double-claimed or
absent. That check is what makes "every PICT extracts or is deliberately skipped" a fact rather
than a claim — it found the one real gap, `PICT` 10000 (the `kCustomPict` placeholder, reached
through a *runtime* atlas key rather than a numeric one, so the branch that emits it never ran).

Naming: `object/<WHAT>_<constant>.png` where `<WHAT>` is the **two-digit
uppercase hex object code** (so a room dump joins against the sprite directory with no lookup
table) and `<constant>` is the exact C identifier. Flowers get six files
`85_kFlower_0..5.png`. `manifest.json`'s `stats` block is the pipeline's self-test and must
read `{"sheet":50,"key":21,"none":13,"tmpl":12,"opaq":9,"pair":8,"proc":3,"frames":1}` plus the
non-object counters `{"strip":14,"bg":18,"ui":38,"misc":1}` — 116
`srcRects` entries + 1 `kFlower` frames entry = 117 object records, of which 88 carry a file.
Backgrounds are asserted to be exactly 512x322 and strips to match their `GliderDefines.h`
GWorld dimensions, so a decoder regression fails here instead of surfacing as a skewed room.
Authority: `graphics-assets.md` §7.

Do **not** emit PNGs for the 12 `kind == "tmpl"` entries: their `srcRects` value is a size/hit
template, not an art rect, and cropping it produces 12 sprites that look almost right
(e.g. `srcRects[kTable]` = (0,0,64,8) would crop the top 8 rows of the furniture sheet instead
of `tableSrc`, 64x22). `graphics-assets.md` §7.7.

**Step 3 — palette.** Emit `clut` 128 as a 256-entry RGB table (dumped in
`graphics-assets.md` §4.2 and `rendering.md` §5.6). Hard-coding it is valid for the shipped art
because all 167 embedded tables agree, but the loader must still honour an embedded
`ColorTable` for **user house art**.

**Step 4 — audio: `'snd '` → PCM.**
`probe_snd.py extract` → 70 `.pcm` files (1,126,404 bytes) + `manifest.tsv`, every one of them
`stdSH` at 22254.5455 Hz. `probe_snd.py extract-houses` → the houses' custom sound-trigger
resources: **58 of 63 extract** (2,133,612 bytes); the other **5 are MACE 6:1 (`cmpSH`,
`compressionID` 4)** and appear in the manifest as `SKIPPED` rows naming the reason —
`CD Demo House` 3007 *Door Chime*, `Demo House` 3011 *Meow*, `Nemo's Market` 3001 *Door Chime*,
3003 *Cash Register*, 3004 *Cat Meow*. Those five play as silence until a MACE decoder exists;
nothing in Stage 1 depends on them. Unlike the application sounds, house sounds carry a
**per-resource sample rate**: **nine distinct values** occur across the 58 —
5563.6364, 7418.1818, 9779.0000, 11127.2727, 11127.5000, 22050.0000, 22254.5454, 22254.5455 and
22255.0000 Hz. Three of those are the same intended rate authored three ways (22254.5454 /
22254.5455 / 22255.0000) and two are exact divisions of it (÷4 and ÷2), but a player must honour
the stored `Fixed` rather than assume the application's single rate, so the manifest's `rate_hz`
column is load-bearing, not decoration.
Target format for the port: `int16` mono at each resource's **native rate**, converted with
`s16 = (int16(b) - 128) << 8`, point-sampled if the audio backend needs another rate (a
deliberate fidelity choice, not laziness). Layout: `sounds [64][]int16` indexed by the sound
constant, `music [7][]int16` for `'snd '` 2000-2006. Authority: `audio.md` §11.

**Step 5 — houses: BinHex → data fork + resource fork.**
`extract_all.py` emits `houses/<name>.house` (data fork) and `houses/<name>.rsrc` (resource fork)
so that **nothing in the Go port ever sees BinHex**. It re-derives `nRooms` from each header and
refuses any house whose data fork is neither exactly `866 + 348·nRooms` nor that plus the
documented 2-byte PowerPC slack (§12.4 of `house-format.md`; only `Sampler` has it), then checks
the corpus totals: 22 houses, 4,070 rooms, all `version == 0x0200`.
For deeper inspection, `tools/probe_house.py` with `check_crc=True` over
`GliderPRO/Houses/*.binhex`. Emit, per house:
the raw data fork **byte-for-byte** (this is the `houseType` image — do not normalise, do not
re-pack, do not trim `Sampler`'s +2 bytes), plus the resource fork split by type (house
`PICT`s, house `'snd '`s). Verify the corpus invariants of §13.6 as a regression test: 22
houses, 4,070 rooms, all `version == 0x0200`, 66/66 CRCs (remembering that `probe_house.py`
itself only checks 44 of them — §13.5), and the two shared 40-byte `savedGame` blocks at +820
(`CD Demo House`/`Slumberland`, `Nemo's Market`/`SpacePods`). Do **not** assert that any two
files are byte-identical: all 22 data forks are distinct, and such a test can never pass.

**Step 6 — movies.**
Planned as "optional, last"; it turned out to cost **0.4 s and 1.1 MB**, so it is simply part of
the pipeline. `probe_mov.py extract` writes one `movie/<name>.idx8` per film — every frame's
8-bit palette indices back to back, no codec at runtime — plus a `manifest.tsv` giving codec,
depth, size, frame count and fps. Whether Stage 1 *draws* them is a separate 1.5 decision; the
data is ready either way.
For reference, `tools/probe_mov.py` over the 15 `.mov` files also renders per-frame RGBA at 5 fps
(see unknown #2).
`kTV`'s screen rect is 64 x 49; `Demo House.mov` is 82 x 62 and gets cropped by
`CenterRectInRect` — reproduce the crop, including the −6 vs −7 truncation
(`quicktime-movies.md` §7.1, OQ5).

**Step 7 — embed.** All of the above is pre-converted at build time and embedded with
`go:embed`; nothing in the shipping port parses PICT, BinHex or DeRez. The one exception is
**user house files**, which must be parsed at runtime and where the decoder must be permissive
(log and fall back to `PICT` 2000, mirroring `LoadGraphicSpecial`,
`GliderPRO/Sources/RoomGraphics.c:134-158`) rather than fatal.

---

## 22. Quick-reference constant appendix

Everything a porter looks up ten times a day. All from `GliderPRO/Headers/GliderDefines.h`
unless noted.

```
TIMING      kTicksPerFrame 2 (:533)        -> 30.07 fps       kIdleSplashTicks 7200L (:197)
GEOMETRY    kRoomWide 512 (:499)  kTileHigh 322 (:498)  kNumTiles 8 (:496)  kTileWide 64 (:497)
            kVertLocalOffset 322 (:501)    kFloorSupportTall 44 (:500)
            kScoreboardTall 20 (:515)      kMaxViewWidth 1536 (:267)  kMaxViewHeight 1026 (:268)
GLIDER      kGliderWide 48  kGliderHigh 20  kHalfGliderWide 24  kGliderBurningHigh 26
            kShadowHigh 9   kShadowTop 306  kGliderStartsDown 32  kNumGliderSrcRects 31
PHYSICS     kGravity 3  kHImpulse 2  kVImpulse 2  kMaxHVel 16          (Player.c:13-16)
            kNormalThrust 5  kHyperThrust 8  kHeliumLift 4             (Input.c:14-16)
            kFloorVentLift -6  kCeilingVentDrop 8  kFanStrength 12     (Interactions.c:13-15)
            kRubberBandVelocity 20 (RubberBands.c:12)  kShredderCountdown -68 (Player.c:17)
LIMITS      kCeilingLimit 8   kFloorLimit 312   kRoofLimit 122         (:503-505)
            kLeftWallLimit 12  kRightWallLimit 500                     (:506, :508)
            kNoLeftWallLimit -24  kNoRightWallLimit 536                (:507, :509)
            kNoCeilingLimit -10   kNoFloorLimit 332                    (:510-511)
BUDGETS     kMaxRoomObs 24  kMaxHotSpots 56  kMaxDynamicObs 18  kMaxSavedMaps 24
            kMaxMasterObjects 216  kMaxRubberBands 2  kMaxSparkles 3  kMaxFlyingPts 3
            kMaxCandles 20  kMaxTikis 8  kMaxCoals 8  kMaxPendulums 8  kMaxGrease 16
            kMaxStars 4  kMaxShredded 4  kMaxScores 10  kNumSrcRects 0x90 (144)
SCORING     kRoomVisitScore 100  kRedClockPoints 100  kBlueClockPoints 300
            kYellowClockPoints 500  kCuckooClockPoints 1000  kStarPoints 5000  (:536-541)
            kInitialGliders 2 (Play.c:19)   kScoreRollAmount 13 (Scoreboard.c:15-21)
SUPPLIES    kBatterySupply 50  kHeliumSupply 150  kBandsSupply 8  kFoilSupply 8
                                                                  (Interactions.c:16-19)
FORMAT      kHouseVersion 0x0200  kNewHouseVersion 0x0300  kRoomIsEmpty -1
            sizeof houseType 866   roomType 348   objectType 12   demoType 6   game2Type 110
ROOMS       kBaseBackgroundID 2000  kFirstOutdoorBack 2009  kNumBackgrounds 18
            kUserBackground 3000  kUserStructureRange 3300  kSplash8BitPICT 1000
            kMaxNumRoomsH 128  kMaxNumRoomsV 64  -> 8192 addressable rooms
COLOUR      kPreferredDepth 8 (Externs.h:15)  kRedOrangeColor8 23 ("actually, 18")
            kGrayBackgroundColor 251  kGrayBackgroundColor4 10  (Scoreboard.c:15-21)
PLACEMENT   kFloorVentTop 305  kCeilingVentTop 8  kFloorBlowerTop 304  kCeilingBlowerTop 5
            kSewerGrateTop 303  kCeilingTransTop 6  kFloorTransTop 302  kStairsTop 28
            kCounterBottom 304  kDresserBottom 293  kCeilingLightTop 4  kHipLampTop 23
            kDecoLampTop 91  kFlourescentTop 12  kTrackLightTop 5
            kDoorInRtLeft 368  kDoorExRtLeft 496  kWindowInTop 64  kWindowInRtLeft 492
            kWindowExTop 64  kWindowExRtLeft 496  kTableThick 8  kShelfThick 6
SWITCHES    kToggle 0  kForceOn 1  kForceOff 2  kOneShot 3
FRAMES      kNumTrackLights 3  kNumOutletPicts 4  kNumCandleFlames 5  kNumTikiFlames 5
            kNumBBQCoals 4  kNumPendulums 3  kNumBreadPicts 6  kNumBalloonFrames 8
            kNumCopterFrames 10  kNumDartFrames 4  kNumBallFrames 2  kNumDripFrames 6
            kNumFishFrames 8  kNumFlowers 6  kNumMarqueePats 7  kLastFadeSequence 16
BOOLEANS    kFaceRight TRUE  kFaceLeft FALSE  kPlayer1 TRUE  kPlayer2 FALSE
```
