# Glider PRO 1.0.4 — Determinism, RNG and Simulation Model (Multiplayer Feasibility)

## Scope

This document answers one question exhaustively: **if two machines run the Glider PRO
simulation from the same starting state and feed it the same inputs, do they stay in
sync?** To answer it, the document enumerates:

1. The RNG algorithm, where it is seeded, and **every** place randomness is consumed.
2. Every dependence on wall-clock time and on frame timing.
3. Every dependence on Mac-specific machine state (screen size, colour depth, memory
   pressure, user preferences).
4. The exact per-frame update order of every entity class in the simulation.
5. The exact ordering of the *room-load* code path, which is not per-frame but which
   consumes RNG and therefore matters just as much.

It then delivers a verdict on lockstep determinism, and concrete recommendations for a
Go port supporting **(a)** a shared-fate 2-player race mode where each machine simulates
its own player and exchanges progress, and **(b)** possible future full lockstep.

**Result on (a), stated up front:** the source rules it out. `HandleInteraction`
(`GliderPRO/Sources/Interactions.c:1691-1711`) resolves both gliders against shared mutable
room state in one call — one hotspot loop testing both gliders with a shared `stillOver`
latch (`:1632-1675`), a first-caller-wins prize rule in `SetObjectState`
(`GliderPRO/Sources/Objects.c:460-462`), and a room rebuild that can occur between the two
`CheckGliderInRoom` calls (`:1705`, `:1706`). §9.8 states the problem with the evidence and
§10.5.2 (`docs/analysis/determinism-networking.md:1701`) gives the resolution: every peer
simulates every player, only input crosses the wire, which is option (b).

Out of scope: rendering fidelity, sound, the level editor, house file authoring, the
scoreboard's pixel layout. Those are covered by sibling documents. Where the editor
touches something relevant to determinism (e.g. `ObjectAdd.c` consuming RNG when the
*author* places an object) it is called out so a porter does not mistake it for
gameplay RNG.

### Conventions

* All citations are of the form `GliderPRO/Sources/Play.c:434`, path relative to the
  repository root.
* **Line numbers are from a CR→LF converted copy of the file.** The originals are
  Classic Mac text (bare `\r`). The conversion used was:
  ```
  for f in Sources/*.c Headers/*.h; do tr '\r' '\n' < "$f" > /tmp/wf-determinism/$(basename "$f"); done
  ```
  This is a byte-for-byte 1:1 substitution, so line *numbers* are identical to what any
  other tool would report after the same conversion.
* Constants are given in the form the source uses, with the decimal and hex equivalent
  where the hex form is load-bearing (object type codes, bit masks, event masks).
* Pseudocode keeps the original C identifier in parentheses, e.g.
  "the frame counter (`gameFrame`)", so a porter can diff against the C line by line.

### A tooling warning that affects reproducing this analysis

In the shell used for this analysis, the bare command `grep` is a wrapper **function**
that execs `ugrep` with `-G --ignore-files --hidden -I --exclude-dir=...`. The
`--ignore-files` flag applies gitignore-style filtering and **silently omitted whole
source files** from search results. An early RNG sweep with bare `grep` reported *zero*
`RandomInt` hits in `Sources/Dynamics.c`, which in fact contains three
(`GliderPRO/Sources/Dynamics.c:304`, `:492`, `:506`). Every search underpinning this
document was re-run with the absolute path `/usr/bin/grep`. **Any bare-`grep` result in
this environment must be treated as untrustworthy for source-tree sweeps.**

## Sources read

Read in full (line counts are of the LF-converted copies):

| File | Lines | Why |
| --- | --- | --- |
| `GliderPRO/Sources/Utilities.c` | 788 | `RandomInt`, `RandomLong`, `RandomLongQUS`, `ToolBoxInit` seeding, `TickCount` helpers, sound-volume scaling |
| `GliderPRO/Sources/Play.c` | 821 | `NewGame`, `PlayGame` main loop, `InitGlider`, `SetObjectsToDefaults`, telephone/chimes RNG |
| `GliderPRO/Sources/Dynamics.c` | 776 | `CheckDynamicCollision`, `DidBandHitDynamic`, per-appliance handlers, sparkle/coffee RNG |
| `GliderPRO/Sources/Dynamics2.c` | 589 | `HandleBalloon`/`HandleCopter`/`HandleDart`/`HandleBall`/`HandleDrip`/`HandleFish` |
| `GliderPRO/Sources/Dynamics3.c` | 555 | `HandleDynamics` dispatch, `RenderDynamics`, `ZeroDinahs`, `AddDynamicObject` |
| `GliderPRO/Sources/Render.c` | 772 | `RenderFrame` order, frame pacing, `InitGarbageRects`, dirty-rect queues |
| `GliderPRO/Sources/ObjectDrawAll.c` | 966 | `DrawARoomsObjects` — the room-load RNG consumer |
| `GliderPRO/Sources/Triggers.c` | 205 | trigger timers and firing order |
| `GliderPRO/Sources/RubberBands.c` | 318 | band integration, collision, swap-with-last removal |
| `GliderPRO/Sources/Transit.c` | 558 | room transitions, two-player limbo handshake, `FollowTheLeader` |
| `GliderPRO/Sources/Events.c` | 573 | `WaitNextEvent`, suspend/resume, idle-demo trigger |
| `GliderPRO/Sources/Main.c` | 386 | init order, prefs defaults, the compiled-out `Random()` call |
| `GliderPRO/Sources/Room.c` | 1206 | `ForceThisRoom`, `GetNeighborRoomNumber`, `DetermineRoomOpenings`, `GetNumberOfLights` |
| `GliderPRO/Headers/GliderDefines.h` | 625 | every constant |
| `GliderPRO/Headers/GliderStructs.h` | 347 | every struct |

Read in the parts that matter (function-level, with the surrounding context):

| File | Lines | Regions read |
| --- | --- | --- |
| `GliderPRO/Sources/Player.c` | 1605 | `MoveGlider` :64, `MoveGliderNormal` :151, `HandleIdleGlider` :1323, `HandleGlider` :1335, `OffsetGlider` :1443, `OffAMortal` :1484 |
| `GliderPRO/Sources/Interactions.c` | 1777 | `SectGlider` :101, `CheckEscapeUpTwo` :171, `CheckGliderInRoom` :689, `HandleRewards` :756, `HandleSwitches` :985, `HandleMicrowaveAction` :1159, `HandleHotSpotCollision` :1198, `CheckForHotSpots` :1627, `HandleInteraction` :1691, `FlagStillOvers` :1715, `WebGlider` :1736 |
| `GliderPRO/Sources/DynamicMaps.c` | 798 | `NilSavedMaps` :46, `BackUpToSavedMap` :70, `ReBackUpSavedMap` :100, `AddSparkle` :169, `AddFlyingPoint` :199, `AddCandleFlame` :316, `AddTikiFlame` :400, `AddBBQCoals` :486, `AddPendulum` :570, `AddStar` :662, `AddAShreddedGlider` :730, `RemoveShreds` :748, `ZeroFlamesAndTheLike` :787 |
| `GliderPRO/Sources/RoomGraphics.c` | 462 | `DrawLocale` :44, `ReadyLevel` :402, `DrawLighting` :422, `RedrawRoomLighting` :434 |
| `GliderPRO/Sources/Grease.c` | 302 | `HandleGrease` :43, `AddGrease` :206, `SpillGrease` :257 |
| `GliderPRO/Sources/Modes.c` | 640 | `FlagGliderInLimbo` :458, `UndoGliderLimbo` :472, `TagGliderIdle` :631 |
| `GliderPRO/Sources/Input.c` | 398 | `LogDemoKey` :44, `DoPause` :77, `DoBatteryEngaged` :121, `DoHeliumEngaged` :160, `GetDemoInput` :186, `GetInput` :281 |
| `GliderPRO/Sources/Objects.c` | 1001 | `IsThisValid` :89, `ListAllLocalObjects` :300, `SetObjectState` :366, `GetObjectState` :703 |
| `GliderPRO/Sources/Scoreboard.c` | 457 | `HandleDynamicScoreboard` :72 |
| `GliderPRO/Sources/SavedGames.c` | 352 | `SaveGame` :303 — the canonical "progress" field list |
| `GliderPRO/Sources/Transitions.c` | 145 | `PourScreenOn` :18 (dead), `WipeScreenOn` :72 |
| `GliderPRO/Sources/StructuresInit2.c` | — | `demoData` allocation and `GetResource('demo',128)` :280-297 |
| `GliderPRO/Sources/Environ.c` | 734 | `thisMac` population :440-560, `CheckMemorySize` :580-660 |
| `GliderPRO/Headers/Externs.h` | 393 | `kPreferredDepth` :15, `prefsInfo` :232-267 |
| `GliderPRO/Headers/Environ.h` | 37 | `macEnviron` :11-32 |

Also parsed with `python3` for empirical verification (results reported inline):

* `GliderPRO/Houses/Empty House.binhex` — BinHex 4.0 decode, then byte-level parse of
  the `houseType` data fork.
* `GliderPRO/Glider PRO.r:199389` — the `data 'demo' (128)` resource, byte-level parse of
  the `demoType` replay stream.
* A `gcc`-compiled transcription of `GliderPRO/Headers/GliderStructs.h` under
  `#pragma pack(2)` to confirm every struct size and field offset.

---

# Part 1 — The simulation clock and the frame model

## 1.1 There is exactly one integer timestep, and no delta time anywhere

Glider PRO's simulation is a **fixed-timestep, integer-only** simulation. There is no
floating point anywhere in the gameplay path, no `dt`, and no velocity scaling by elapsed
time. Every velocity is a `short` measured in **pixels per frame**, and every position is
a `Rect` of four `short`s measured in pixels.

Evidence: `gliderType` (`GliderPRO/Headers/GliderStructs.h:200-216`) declares
`short hVel, vVel; short wasHVel, wasVVel; short vDesiredVel, hDesiredVel;` — all
integers. `MoveGlider` (`GliderPRO/Sources/Player.c:64-147`) integrates by direct
addition of `hVel`/`vVel` onto the four `Rect` members with no scaling:

```
thisGlider->whole.right = thisGlider->dest.right;  // Player.c:101
thisGlider->dest.left  += thisGlider->hVel;        // Player.c:102
thisGlider->dest.right += thisGlider->hVel;        // Player.c:103
thisGlider->whole.left  = thisGlider->dest.left;   // Player.c:104
```

(`whole` is the union of the pre- and post-move rects, used as the dirty rect; it is
maintained by the same statements that move `dest`.)

`dynaType` (`GliderPRO/Headers/GliderStructs.h:310-320`) is the same: `short hVel, vVel`.
`bandType` (`:274-279`) is the same. There is no `float` or `double` declared in any
gameplay struct in `GliderStructs.h`.

**Consequence for a port:** the physics is trivially bit-reproducible across
architectures. There is no IEEE-754 rounding hazard, no `x87` 80-bit intermediate hazard,
no fused-multiply-add hazard, no compiler-reassociation hazard. This is the single
largest piece of good news in this document.

## 1.2 The two frame counters

| Global | Type | Declared | Reset | Mutated |
| --- | --- | --- | --- | --- |
| `gameFrame` | `long` | **defined** (not extern) at `GliderPRO/Sources/Play.c:51` | `gameFrame = 0L;` at `GliderPRO/Sources/Play.c:112` | `gameFrame++;` at `GliderPRO/Sources/Play.c:434` |
| `evenFrame` | `Boolean` | `GliderPRO/Sources/Play.c:53` | not reset in `NewGame` | `evenFrame = !evenFrame;` at `GliderPRO/Sources/Play.c:435`; **also force-set `true`** at `GliderPRO/Sources/Dynamics2.c:420`, `GliderPRO/Sources/Dynamics3.c:474`, `GliderPRO/Sources/Dynamics3.c:524` |

`gameFrame` is a monotone frame index used for three purposes:

1. Demo replay indexing: `if (gameFrame == (long)demoData[demoIndex].frame)`
   (`GliderPRO/Sources/Input.c:224`).
2. Demo recording: `demoData[demoIndex].frame = gameFrame;`
   (`GliderPRO/Sources/Input.c:46`).
3. Scoreboard work-slicing: `whosTurn = gameFrame & 0x00000007;`
   (`GliderPRO/Sources/Scoreboard.c:95`) — spreads the six scoreboard-blink refreshes
   across an 8-frame cycle.

`evenFrame` is a half-rate gate used pervasively as "do this every other frame". It is
toggled once per frame at `GliderPRO/Sources/Play.c:435`, which means its value is a pure
function of frame parity — **except** that three places overwrite it:

* `GliderPRO/Sources/Dynamics2.c:420` — `HandleBall`, when an inactive ball becomes
  active, executes `evenFrame = true;`.
* `GliderPRO/Sources/Dynamics3.c:474` — `AddDynamicObject`, in the `kBall` branch, sets
  `evenFrame = true;` before its reverse-integration loop.
* `GliderPRO/Sources/Dynamics3.c:524` — `AddDynamicObject`, in the `kFish` branch, ditto.

The `AddDynamicObject` writes happen during room load, so they only perturb the parity of
the first post-load frame. The `HandleBall` write happens *mid-frame, inside the dynamics
pass*, and therefore changes the behaviour of every later phase of that same frame
(`RenderFrame` picks flames-vs-stars off `evenFrame` at
`GliderPRO/Sources/Render.c:649`). This is almost certainly an author bug — `evenFrame` is
a global being abused as scratch state for the ball's reverse-integration loop — but it is
observable behaviour and a port must reproduce it exactly or the two implementations will
diverge in animation phase and in `HandleToast`/`HandleDrip`/`HandleFish` gravity timing.

### Everything gated on `evenFrame`

| Site | Effect |
| --- | --- |
| `GliderPRO/Sources/Dynamics.c:60` | `if ((evenFrame) && (foilTotal > 0)) foilTotal--;` — aluminium foil decays at half rate while colliding with a dynamic |
| `GliderPRO/Sources/Dynamics.c:327` | `HandleToast` advances the toast animation at half rate |
| `GliderPRO/Sources/Dynamics2.c:38`, `:77` | `HandleBalloon` frame advance (rising and falling variants) |
| `GliderPRO/Sources/Dynamics2.c:409` | `HandleBall` gravity: `if (evenFrame) dinahs[who].vVel++;` |
| `GliderPRO/Sources/Dynamics2.c:433` | `HandleDrip` frame flip `frame = 9 - frame` |
| `GliderPRO/Sources/Dynamics2.c:471` | `HandleDrip` gravity `vVel++` |
| `GliderPRO/Sources/Dynamics2.c:551` | `HandleFish` gravity `vVel++` |
| `GliderPRO/Sources/Render.c:649-652` | `if (evenFrame) RenderFlames(); else RenderStars();` |
| `GliderPRO/Sources/Interactions.c` (`WebGlider`, from :1736) | web damage accrues on even frames only |

Note the asymmetry: balls, drips and fish gain one unit of downward velocity **every
other frame**, while the glider gains `kGravity` **every frame**
(`vDesiredVel = kGravity;` at `GliderPRO/Sources/Player.c:91`, with `kGravity` = 3 at
`GliderPRO/Sources/Player.c:13`). A port that "cleans this up" into a uniform gravity
model will produce visibly different, and different-in-detail, trajectories.

## 1.3 Frame pacing: `kTicksPerFrame`, and why it does not affect step count

`kTicksPerFrame` is **2** (`GliderPRO/Headers/GliderDefines.h:533`). A Mac "tick" is
1/60 s, so the nominal frame rate is 30 Hz.

Pacing is implemented as a busy-wait at the *end* of `RenderFrame`:

```
1. while (TickCount() < nextFrame) { }               // Render.c:662-664
2. nextFrame = TickCount() + kTicksPerFrame;         // Render.c:665
```

and initialised in `InitGarbageRects`:

```
nextFrame = TickCount() + kTicksPerFrame;            // Render.c:690
```

Two details matter enormously:

* **Line 665 assigns, it does not accumulate.** It is `nextFrame = TickCount() + 2`, not
  `nextFrame += 2`. There is no catch-up, no frame dropping, and no variable-step
  fallback. If a frame takes 10 ticks of real work, the next deadline is simply 2 ticks
  after that frame finished. **A slow machine runs the game in slow motion; it does not
  skip simulation steps.**
* Therefore **`TickCount()` never influences how many simulation steps are executed, nor
  what any step computes.** It only influences *when* steps happen in wall-clock time.
  This is the second largest piece of good news in this document: the busy-wait is a pure
  pacer and can be replaced with anything (a Go ticker, vsync, an unbounded loop for
  headless replay) without changing simulation results.

There is one caveat: `TickCount()` is called *inside* `RenderFrame`, i.e. inside the
simulation call graph. A port that wants a headless deterministic simulator must lift the
pacer out of the step function (Part 10 is the sibling file `determinism-networking.md`;
this is §10.1, "The simulation/render cut", at `determinism-networking.md:64`).

### Every `TickCount()` call site in the tree

Verified with `/usr/bin/grep -n "TickCount" Sources/*.c`:

| Site | Purpose | Affects simulation results? |
| --- | --- | --- |
| `GliderPRO/Sources/Render.c:662` | frame-pacing busy-wait | No |
| `GliderPRO/Sources/Render.c:665` | set next frame deadline | No |
| `GliderPRO/Sources/Render.c:690` | initialise deadline in `InitGarbageRects` | No |
| `GliderPRO/Sources/Utilities.c:446` | `WaitForInputEvent` timeout base (`TickCount() + 60L * seconds`) | No — UI only |
| `GliderPRO/Sources/Utilities.c:474` | `WaitForInputEvent` timeout test | No |
| `GliderPRO/Sources/Play.c:277` | reset idle-demo timer after `NewGame` returns | No |
| `GliderPRO/Sources/Play.c:302` | reset idle-demo timer after `DoDemoGame` | No |
| `GliderPRO/Sources/Events.c:427` | reset idle timer on resume from suspend | No |
| `GliderPRO/Sources/Events.c:538` | `if (TickCount() >= incrementModeTime) DoDemoGame();` — the auto-demo trigger | **Only decides whether the attract-mode demo starts.** Once started, the demo is frame-driven |
| `GliderPRO/Sources/Menu.c:336`, `:402`, `:419`, `:424` | reset idle timer on menu activity | No |
| `GliderPRO/Sources/AppleEvents.c:111` | reset idle timer on Apple Event | No |
| `GliderPRO/Sources/GameOver.c:149`, `:219`, `:220`, `:456`, `:477`, `:478` | pacing for the game-over page-flutter animation (`TickCount() + 2`) | No — post-game cutscene |
| `GliderPRO/Sources/InterfaceInit.c:163` | initialise `incrementModeTime` at launch | No |

`kIdleSplashTicks` is `7200L` (`GliderPRO/Headers/GliderDefines.h:197`) = 120 s at 60
ticks/s, the idle timeout before attract mode.

## 1.4 Wall-clock dependencies that *do* reach the screen

`GetTime()` (Mac Toolbox, fills a `DateTimeRec` from the system clock) is called at five
sites, all in object *drawing*:

| Site | Object |
| --- | --- |
| `GliderPRO/Sources/ObjectDraw.c:970` | clock face variant 1 |
| `GliderPRO/Sources/ObjectDraw.c:1012` | clock face variant 2 |
| `GliderPRO/Sources/ObjectDraw.c:1033` | clock face variant 3 |
| `GliderPRO/Sources/ObjectDraw.c:1054` | clock face variant 4 |
| `GliderPRO/Sources/ObjectDraw2.c:1160` | clock face (custom/user picture path) |

These draw **real-world hour and minute hands** onto wall clocks in the room. They are
purely cosmetic: they write into an offscreen GWorld and never feed back into any
gameplay variable. They are, however, a *visual* desync between two peers whose system
clocks differ — worth noting because a naive "screenshot comparison" desync detector will
fire on them.

`GetDateTime()` is called at six sites:

| Site | Purpose | Determinism impact |
| --- | --- | --- |
| `GliderPRO/Sources/Utilities.c:61` | `GetDateTime((UInt32 *)&qd.randSeed);` — **seeds the RNG, once per launch** | **Total.** See Part 2 |
| `GliderPRO/Sources/Utilities.c:117` | `InitRandomLongQUS` seeds `theSeed` | None — dead code, see §2.4 |
| `GliderPRO/Sources/HighScores.c:411` | high-score timestamp | None |
| `GliderPRO/Sources/HouseIO.c:477` | house-file timestamp on write | None |
| `GliderPRO/Sources/SavedGames.c:320` | saved-game timestamp | None |
| `GliderPRO/Sources/Validate.c:109`, `GliderPRO/Sources/DebugUtilities.c:268`, `:328` | validation / debug | None |

---

# Part 2 — The random number generator

## 2.1 `RandomInt` is the only live RNG in the shipped game

The complete implementation (`GliderPRO/Sources/Utilities.c:72-82`):

```c
short RandomInt (short range)
{
	register long	rawResult;

	rawResult = Random();                              // :76
	if (rawResult < 0L)                                // :77
		rawResult *= -1L;                              // :78
	rawResult = (rawResult * (long)range) / 32768L;     // :79

	return ((short)rawResult);                          // :81
}
```

As numbered pseudocode, preserving names:

```
RandomInt(range):
  1. raw (rawResult) <- Random()            // Toolbox trap; signed 16-bit result widened to long
  2. if raw < 0 then raw <- -raw            // absolute value; folds the sign
  3. raw <- (raw * range) / 32768           // C integer division: truncation toward zero
  4. return (short) raw
```

**`Random()` is a QuickDraw Toolbox trap. Its algorithm is not present anywhere in this
source tree.** Verified with `/usr/bin/grep -rn "16807\|randSeed\|Lehmer" Sources/*.c
Headers/*.h`: the only hit outside `RandomInt` itself is the seeding line
`GliderPRO/Sources/Utilities.c:61`. A search of the 199,843-line Rez dump
`GliderPRO/Glider PRO.r` for `16807` and `randSeed` also returns nothing. **A Go port
must supply its own generator; there is no reference implementation to transcribe from
this repository.** This is recorded as an open question in the final section.

### 2.1.1 Empirically derived properties of the `RandomInt` mapping

The mapping in steps 2–4 is fully determined by the source, independent of what `Random()`
does internally, so it can be characterised exhaustively. Enumerating all 65,536 possible
`Random()` return values (`python3`, results observed, not inferred):

```
range=2      -> values 0..2 ; count(0)=32767 count(2)=1 ; total=65536
range=3      -> values 0..3 ; count(0)=21845 count(3)=1 ; total=65536
range=4      -> values 0..4 ; count(0)=16383 count(4)=1 ; total=65536
range=5      -> values 0..5 ; count(0)=13107 count(5)=1 ; total=65536
range=6      -> values 0..6 ; count(0)=10923 count(6)=1 ; total=65536
range=60     -> values 0..60 ; count(0)=1093  count(60)=1 ; total=65536
range=180    -> values 0..180 ; count(0)=365  count(180)=1 ; total=65536
range=200    -> values 0..200 ; count(0)=327  count(200)=1 ; total=65536
range=240    -> values 0..240 ; count(0)=273  count(240)=1 ; total=65536
range=25000  -> values 0..25000 ; count(0)=3  count(25000)=1 ; total=65536
```

Three findings, all load-bearing:

**Finding 1 — `RandomInt(range)` can return `range` itself, not `range-1`.** If `Random()`
returns `-32768` (the most negative 16-bit value), step 2 computes `-(-32768) = +32768`
and step 3 computes `(32768 * range) / 32768 = range` exactly. Observed:

```
RandomInt(2)  with raw=-32768 returns 2      (== range, OUT OF [0, range-1])
RandomInt(3)  with raw=-32768 returns 3
RandomInt(4)  with raw=-32768 returns 4
RandomInt(5)  with raw=-32768 returns 5
RandomInt(6)  with raw=-32768 returns 6
RandomInt(240) with raw=-32768 returns 240
RandomInt(25000) with raw=-32768 returns 25000
```

Probability 1/65536 per call, *if* the underlying `Random()` can emit `-32768`. Where this
matters:

* `flames[numFlames].mode = RandomInt(kNumCandleFlames);`
  (`GliderPRO/Sources/DynamicMaps.c:338`) with `kNumCandleFlames` = 5
  (`GliderPRO/Headers/GliderDefines.h:447`) — mode 5 indexes past the 5 candle frames
  (0..4) and the renderer reads one strip of source pixels beyond the backed-up map.
  `RenderFlames` (`GliderPRO/Sources/Render.c:193`) does bound the *subsequent* advance
  with `if (mode >= kNumCandleFlames) mode = 0`, so the out-of-range value survives for
  exactly one frame.
* `theStars[numStars].mode = RandomInt(6);` (`GliderPRO/Sources/DynamicMaps.c:685`) — mode
  6 with only 6 star frames (0..5).
* `if (RandomInt(2) == 0) toOrFro = true; else toOrFro = false;`
  (`GliderPRO/Sources/DynamicMaps.c:594`) — harmless, the `else` absorbs both 1 and 2.
* `if (RandomInt(2) == 0)` at `GliderPRO/Sources/Play.c:775` (chime selection) — harmless
  for the same reason.

A Go port using `rand.Intn(range)` will produce results in `[0, range-1]` and will
therefore **never** reproduce this. That is fine for a fresh port, but it means the Go
port is not bit-identical to the original in the 1-in-65536 case, and it means a porter
must **explicitly clamp** the flame/star mode if they choose to mimic the original
mapping.

**Finding 2 — the distribution is not uniform.** For `range = 2`:

```
RandomInt(2): #(returns 0)=32767  #(returns 1)=32768  #(returns 2)=1   (of 65536 raws)
```

The sign-folding at step 2 makes `0` one count short of `1`, because `raw = 0` and
`raw = -0` are the same value; there is no `-0` in two's complement, so the negative half
of the range contributes 32768 distinct magnitudes (1..32768) while the non-negative half
contributes 32768 (0..32767) — the magnitudes 1..32767 are reachable twice, `0` once, and
`32768` once. A port aiming for statistical rather than bit fidelity can ignore this;
a port aiming for bit fidelity cannot.

**Finding 3 — no overflow is possible.** `range` is a `short`, maximum 32767. The
intermediate `raw * range` peaks at `32768 * 32767 = 1,073,709,056`, comfortably inside
the signed 32-bit range `2,147,483,647`. Confirmed numerically. So the largest live call,
`RandomInt(kRingSpread)` with `kRingSpread` = 25000
(`GliderPRO/Sources/Play.c:21`, used at `:735` and `:759`), peaks at
`32768 * 25000 = 819,200,000` — safe. **A Go port must still use `int32`/`int64`
arithmetic here, not `int16`**, or the multiply will wrap.

## 2.2 Seeding: exactly once per process launch, from wall-clock seconds

```c
void ToolBoxInit (void)
{
#if !TARGET_CARBON
	InitGraf(&qd.thePort);                                  // Utilities.c:46
	InitFonts();
	FlushEvents(everyEvent, 0);
	InitWindows();
	InitMenus();
	TEInit();
	InitDialogs(nil);

	MaxApplZone();                                          // :54
	MoreMasters(); MoreMasters(); MoreMasters(); MoreMasters();  // :56-59

	GetDateTime((UInt32 *)&qd.randSeed);                     // :61  <== THE ONLY SEEDING
#endif
	InitCursor();                                            // :65
	switchedOut = false;                                     // :66
}
```

`ToolBoxInit` is the very first call in `main` (`GliderPRO/Sources/Main.c:284`-region).
`GetDateTime` yields **seconds since the Mac epoch (1904-01-01 00:00:00 local time)** as a
`UInt32`, written directly into the QuickDraw global `qd.randSeed`.

Consequences:

1. **Seed granularity is one second.** Two launches within the same second get the same
   seed and therefore the same entire random sequence.
2. **The seed is never re-set.** There is no reseeding at `NewGame`
   (`GliderPRO/Sources/Play.c:74`; the reset block at `:112-118` sets `gameFrame`,
   `numBands`, `demoIndex`, `saidFollow`, `otherPlayerEscaped`, `onePlayerLeft`,
   `playerSuicide` and nothing else), no reseeding at room load
   (`DrawLocale`, `GliderPRO/Sources/RoomGraphics.c:44`), and no reseeding on death
   (`OffAMortal`, `GliderPRO/Sources/Player.c:1484`).
3. **The RNG stream is therefore global and cumulative across the entire session** —
   splash screen, menus, every room you have ever entered, every death, and every
   attract-mode demo all draw from the same stream in the order they happen.
   This is the single most important structural fact about the RNG for multiplayer
   purposes.
4. `qd.randSeed` is a `long` inside the QuickDraw globals block. Under Carbon
   (`TARGET_CARBON`) the whole seeding block is `#if`'d out
   (`GliderPRO/Sources/Utilities.c:45`, `:63`), so a Carbon build would use whatever
   `Random()` defaults to. The shipped 1.0.4 arcade build is the non-Carbon path.

## 2.3 The build configuration that decides which RNG code is live

`GliderPRO/Headers/GliderDefines.h:11-16`:

```c
//#define CREATEDEMODATA        // NOT defined
//#define COMPILEDEMO           // NOT defined
//#define CAREFULDEBUG          // NOT defined
#define COMPILENOCP             // DEFINED
#define COMPILEQT               // DEFINED
#define BUILD_ARCADE_VERSION 1  // DEFINED, value 1
```

These four states matter for this document:

| Flag | State | Effect on determinism |
| --- | --- | --- |
| `COMPILENOCP` | defined | Removes the copy-protection block in `WriteOutPrefs`, which contains a `Random()` call — see §2.4 |
| `CREATEDEMODATA` | not defined | `demoData` is loaded from the `'demo'` resource rather than recorded; `LogDemoKey` is still compiled but its callers in `GetInput` are only reached in record mode |
| `COMPILEDEMO` | not defined | Full game, not the crippled demo |
| `BUILD_ARCADE_VERSION` | 1 | `GetDemoInput` aborts the attract demo on *any* game key (`GliderPRO/Sources/Input.c:191-203`), rather than checking for Command |

## 2.4 Dead RNG code — do not port these

Four RNG-related code paths exist in the tree and are **never executed** in the shipped
build. Each was verified by a repo-wide `/usr/bin/grep` for its identifier.

| Function / site | Location | Why dead |
| --- | --- | --- |
| `RandomLong(long range)` | `GliderPRO/Sources/Utilities.c:88-109` | Declared at `GliderPRO/Headers/Externs.h:343`-region; **zero call sites**. Would consume *two* `Random()` values (high word `:96`, low word `:101`) and combine with `rawResultHi = (rawResultHi << 16) + rawResultLo;` at `:106` |
| `InitRandomLongQUS(void)` | `GliderPRO/Sources/Utilities.c:115-118` | `GetDateTime(&theSeed);` — zero call sites |
| `RandomLongQUS(void)` | `GliderPRO/Sources/Utilities.c:124-128` | A textbook LCG: `theSeed = theSeed * 1103515245 + 12345; return (theSeed);`. **Zero call sites.** This is a trap for a porter: it *looks* like the game's RNG and it is fully specified in the source, but nothing calls it |
| `thePrefs.fakeLong = Random();` | `GliderPRO/Sources/Main.c:232` | Sits inside `#ifndef COMPILEDEMO` / `#ifndef COMPILENOCP` (`GliderPRO/Sources/Main.c:229-234`). `COMPILENOCP` **is** defined at `GliderPRO/Headers/GliderDefines.h:14`, so this is compiled out. Independently confirmed: the `prefsInfo` struct's matching field is itself commented out — `// long encrypted, fakeLong;` at `GliderPRO/Headers/Externs.h:239` — so the code would not even compile if the guard were removed |
| `PourScreenOn(Rect *)` | `GliderPRO/Sources/Transitions.c:18-68` | Contains `do { i = RandomInt(colWide); } while (columnProgress[i] >= rowTall);` at `:42-45` — a rejection-sampling loop, i.e. a *variable* number of RNG draws. **Zero call sites** repo-wide. Only `WipeScreenOn` (`GliderPRO/Sources/Transitions.c:72`, no RNG, `kWipeRectThick` = 4) and `DumpScreenOn` are used |

`RandomLongQUS`'s LCG constants (multiplier `1103515245` = `0x41C64E6D`, increment
`12345` = `0x3039`) are the well-known ANSI-C `rand()` parameters. **Do not mistake them
for Glider PRO's actual generator.**

## 2.5 Complete enumeration of live RNG consumption

Every `RandomInt` call site in the tree, from
`/usr/bin/grep -n "RandomInt" Sources/*.c`, classified. **There are 32 call sites** (re-counted:
`DynamicMaps.c` 5, `Dynamics.c` 3, `Dynamics3.c` 1, `GameOver.c` 9, `InterfaceInit.c` 1,
`ObjectAdd.c` 5, `Play.c` 7, `Transitions.c` 1 — excluding the definition at
`Utilities.c:72`); **16 of them are reachable during gameplay or room load** (Group A's 10
plus Group B's 6). Appendix Z uses the same total of 32.

### Group A — per-frame gameplay RNG (10 sites, 4 distinct functions)

| # | Site | Code | When |
| --- | --- | --- | --- |
| A1 | `GliderPRO/Sources/Play.c:735` | `thePhone.nextRing = RandomInt(kRingSpread) + kRingBaseDelay;` | `InitTelephone`, once per `NewGame` |
| A2 | `GliderPRO/Sources/Play.c:736` | `thePhone.rings = RandomInt(3) + 3;` | `InitTelephone`, once per `NewGame` |
| A3 | `GliderPRO/Sources/Play.c:739` | `theChimes.nextRing = RandomInt(kChimeDelay) + 1;` | `InitTelephone`, once per `NewGame` |
| A4 | `GliderPRO/Sources/Play.c:759` | `thePhone.nextRing = RandomInt(kRingSpread) + kRingBaseDelay;` | `HandleTelephone`, when a ring cycle completes |
| A5 | `GliderPRO/Sources/Play.c:760` | `thePhone.rings = RandomInt(3) + 3;` | same frame as A4 |
| A6 | `GliderPRO/Sources/Play.c:775` | `if (RandomInt(2) == 0)` — picks `kChime1Sound` vs `kChime2Sound` | `HandleTelephone`, on each chime strike, only if `numChimes > 0` |
| A7 | `GliderPRO/Sources/Play.c:784` | `theChimes.nextRing = RandomInt(delayTime) + 1;` | same frame as A6 |
| A8 | `GliderPRO/Sources/Dynamics.c:304` | `dinahs[who].timer = RandomInt(240) + 60;` | `HandleSparkleObject`, each time a sparkle object fires |
| A9 | `GliderPRO/Sources/Dynamics.c:492` | `dinahs[who].timer = 200 + RandomInt(200);` | `HandleCoffee`, branch 1 |
| A10 | `GliderPRO/Sources/Dynamics.c:506` | `dinahs[who].timer = 200 + RandomInt(200);` | `HandleCoffee`, branch 2 |

(Ten call sites in four distinct source functions: `InitTelephone`, `HandleTelephone`,
`HandleSparkleObject`, `HandleCoffee`.)

### Group B — room-load RNG (6 sites)

These fire from `DrawARoomsObjects` (`GliderPRO/Sources/ObjectDrawAll.c:23`) during
`DrawLocale` (`GliderPRO/Sources/RoomGraphics.c:44`), i.e. on **every room transition**
and on `NewGame`.

| # | Site | Code | Guarded by |
| --- | --- | --- | --- |
| B1 | `GliderPRO/Sources/Dynamics3.c:208` | `dinahs[numDynamics].timer = RandomInt(60) + 15;` | `AddDynamicObject`, `kSparkle` branch only; requires `numDynamics < kMaxDynamicObs` (18) at `:193` |
| B2 | `GliderPRO/Sources/DynamicMaps.c:338` | `flames[numFlames].mode = RandomInt(kNumCandleFlames);` | `AddCandleFlame`: requires `numFlames < kMaxCandles` (20), `h >= 16`, `v >= 15`, **and** `BackUpToSavedMap` returning `!= -1` |
| B3 | `GliderPRO/Sources/DynamicMaps.c:422` | `tikiFlames[numTikiFlames].mode = RandomInt(kNumTikiFlames);` | `AddTikiFlame`: `numTikiFlames < kMaxTikis` (8), `h >= 8`, `v >= 10`, `savedNum != -1` |
| B4 | `GliderPRO/Sources/DynamicMaps.c:508` | `bbqCoals[numCoals].mode = RandomInt(kNumBBQCoals);` | `AddBBQCoals`: `numCoals < kMaxCoals` (8), `h >= 32`, `v >= 9`, `savedNum != -1` |
| B5 | `GliderPRO/Sources/DynamicMaps.c:594` | `if (RandomInt(2) == 0) toOrFro = true; else toOrFro = false;` | `AddPendulum`: `numPendulums < kMaxPendulums` (8), `h >= 32`, `v >= 28`, `savedNum != -1` |
| B6 | `GliderPRO/Sources/DynamicMaps.c:685` | `theStars[numStars].mode = RandomInt(6);` | `AddStar`: `numStars < kMaxStars` (4), `savedNum != -1` |

### Group C — non-gameplay RNG (never during `PlayGame`)

| # | Site | Code | Context |
| --- | --- | --- | --- |
| C1 | `GliderPRO/Sources/InterfaceInit.c:160` | `wasFlower = RandomInt(kNumFlowers);` | Once at launch; picks which of 6 flowers appears on the splash screen. **Advances the shared stream before any game starts** |
| C2 | `GliderPRO/Sources/GameOver.c:123` | `workSrcRect.right + RandomInt(workSrcRect.right / 5) + ...` | Game-over page-flutter animation |
| C3 | `GliderPRO/Sources/GameOver.c:125` | `RandomInt(workSrcRect.bottom) - workSrcRect.bottom / 2` | ditto |
| C4 | `GliderPRO/Sources/GameOver.c:127` | `pages[i].frame = RandomInt(6);` | ditto |
| C5 | `GliderPRO/Sources/GameOver.c:296` | `pages[i].counter = RandomInt(32);` | ditto |
| C6 | `GliderPRO/Sources/GameOver.c:322` | `if ((pages[i].dest.bottom + RandomInt(8)) > stopPages)` | ditto |
| C7 | `GliderPRO/Sources/GameOver.c:346` | `if (RandomInt(2) == 0)` | ditto |
| C8 | `GliderPRO/Sources/GameOver.c:369` | `pages[i].counter = RandomInt(4) + 4;` | ditto |
| C9 | `GliderPRO/Sources/GameOver.c:384` | `pages[i].counter = RandomInt(8) + 8;` | ditto |
| C10 | `GliderPRO/Sources/GameOver.c:385` | `if (RandomInt(2) == 0)` | ditto |
| C11 | `GliderPRO/Sources/ObjectAdd.c:621` | `data.g.delay = 10 + RandomInt(10);` | **Editor only** — assigns a default delay when the *author* places an appliance |
| C12 | `GliderPRO/Sources/ObjectAdd.c:626` | `data.g.delay = 10 + RandomInt(10);` | editor only |
| C13 | `GliderPRO/Sources/ObjectAdd.c:689` | `data.h.delay = 10 + RandomInt(10);` | editor only |
| C14 | `GliderPRO/Sources/ObjectAdd.c:715` | `data.h.delay = 10 + RandomInt(10);` | editor only |
| C15 | `GliderPRO/Sources/ObjectAdd.c:742` | `wasFlower = RandomInt(kNumFlowers);` | editor only |
| C16 | `GliderPRO/Sources/Transitions.c:44` | `i = RandomInt(colWide);` | **dead** — `PourScreenOn` is never called |

Group C is critical to understand for multiplayer: **C1..C10 draw from the same stream as
Group A and B.** If peer 1 has watched an attract demo and peer 2 has not, or one peer
saw a game-over animation, their streams are at different positions. Group C11..C15 only
run inside the level editor and are therefore irrelevant at runtime, but they do mean the
*house file's* `delay` bytes were themselves chosen randomly at authoring time — which is
fine, because they are then baked into the file and read deterministically
(§4.4 verifies this against real bytes).

## 2.6 The telephone and chimes: the only RNG-driven gameplay timers

`GliderPRO/Sources/Play.c:18-31`:

```c
#define kHouseBannerAlert   1009
#define kInitialGliders     2
#define kRingDelay          90
#define kRingSpread         25000
#define kRingBaseDelay      5000
#define kChimeDelay         180

typedef struct
{
	short   nextRing;
	short   rings;
	short   delay;
} phoneType, *phonePtr;
```

`InitTelephone` (`GliderPRO/Sources/Play.c:733-740`):

```
1. thePhone.nextRing  <- RandomInt(25000) + 5000        // Play.c:735 ; range [5000, 30000]
2. thePhone.rings     <- RandomInt(3) + 3               // Play.c:736 ; range [3, 5] (6 in the 1/65536 case)
3. thePhone.delay     <- kRingDelay (90)                // Play.c:737
4. theChimes.nextRing <- RandomInt(180) + 1             // Play.c:739 ; range [1, 180]
```

Note the scale: `nextRing` starts between 5000 and 30000 **frames**, i.e. between roughly
2.8 and 16.7 minutes of play at 30 Hz. `thePhone.nextRing` is a `short`, and 30000 fits
(max 32767) — but only just.

`HandleTelephone` (`GliderPRO/Sources/Play.c:744-789`), as pseudocode:

```
HandleTelephone():
  1. if (!phoneBitSet) then                                       // :748
  2.     if thePhone.nextRing == 0 then                            // :750  EQUALITY, not <=
  3.         if thePhone.delay == 0 then                           // :752  EQUALITY, not <=
  4.             thePhone.delay <- kRingDelay (90)                  // :754  set BEFORE the sound
  5.             play kPhoneRingSound at kPhoneRingPriority         // :755
  6.             thePhone.rings--                                   // :756
  7.             if thePhone.rings == 0 then                        // :757  EQUALITY, not <=
  8.                 thePhone.nextRing <- RandomInt(25000) + 5000   // :759
  9.                 thePhone.rings    <- RandomInt(3) + 3          // :760
 10.         else
 11.             thePhone.delay--                                   // :764  decrement is in the ELSE
 12.     else
 13.         thePhone.nextRing--                                    // :767  decrement is in the ELSE
 14. if numChimes > 0 then                        // chimes object(s) present in this room
 15.     if theChimes.nextRing == 0 then                            // :773  EQUALITY, not <=
 16.         if RandomInt(2) == 0 then play kChime1Sound else play kChime2Sound   // :775-778
 17.         delayTime <- kChimeDelay / numChimes                    // :780
 18.         if delayTime < 2 then delayTime <- 2                    // :781-782
 19.         theChimes.nextRing <- RandomInt(delayTime) + 1          // :784
 20.     else
 21.         theChimes.nextRing--                                    // :787  decrement is in the ELSE
```

**Three structural details a port must not paraphrase away.** (a) Every test is `== 0`,
never `<= 0`; combined with (b) the decrement living in the `else` branch, a counter that
somehow went negative would never recover — the countdown is a strict "decrement while
non-zero, then act while zero" machine, and it spends a whole frame *at* zero before acting.
(c) `thePhone.delay` is reassigned to 90 **before** `PlayPrioritySound`, so the ring burst is
`rings` rings spaced 90 frames apart with the first ring firing on the frame `nextRing`
reaches 0 *and* `delay` reaches 0.

`StrikeChime()` (`GliderPRO/Sources/Play.c:793`) forces `theChimes.nextRing = 0;`, i.e.
"chime on the next frame" — called when the glider strikes a wind-chimes object via the
`kChimeIt` hotspot action (`kChimeIt` = 25,
`GliderPRO/Headers/GliderDefines.h:307`).

**`phoneBitSet` and `numChimes` have different scopes — this was previously stated
incorrectly and is worth getting right.**

* `phoneBitSet` is **per house, not per room.** It is decoded once, at house load, from bit 1
  of `houseType.flags`: `phoneBitSet = (((*thisHouse)->flags & 0x00000002) == 0x00000002);`
  (`GliderPRO/Sources/HouseIO.c:417`). It is otherwise only written by `CreateNewHouse`
  (`GliderPRO/Sources/House.c:143`, `false`) and by the House Info editor dialog
  (`GliderPRO/Sources/HouseInfo.c:266-269`). Nothing at room-transition time touches it.
  Note also the **inverted sense**: the countdown runs when `!phoneBitSet`, i.e. when the
  house flag is *clear* — so a house that sets the "phone bit" is a house whose phone never
  rings. See Open question 2.
* `numChimes` **is** room-scoped: zeroed by `ZeroFlamesAndTheLike`
  (`GliderPRO/Sources/DynamicMaps.c:796`) and incremented per chimes object by
  `ObjectRects.c:1052` during room load.

So the RNG-consumption consequence is narrower than "which rooms you visited": the phone's
draws (A4/A5) are gated by a **whole-house constant**, and only the chimes' draws (A6/A7)
depend on the current room. Both still make total stream position depend on play history —
A4/A5 on elapsed frames, A6/A7 on which rooms have chimes and how long you stayed.

## 2.7 The RNG consumption profile of a room load, in exact order

This is the mechanism that makes naive shared-RNG lockstep unsafe, so it is spelled out
completely.

`DrawLocale()` (`GliderPRO/Sources/RoomGraphics.c:44-130` — the function header is at `:44`
and the closing brace at `:130`), numbered:

```
DrawLocale():
   1. ZeroFlamesAndTheLike()        // :51  numFlames=numTikiFlames=numCoals=numPendulums=numStars=0
   2. ZeroDinahs()                  // :52  numDynamics=0, all 18 slots type=kObjectIsEmpty
   3. KillAllBands()                // :53  numBands=0
   4. ZeroMirrorRegion()            // :54
   5. ZeroTriggers()                // :55  all 16 triggers[].armed=false
   6. numTempManholes <- 0          // :56
   7. FlushAnyTriggerPlaying(); DumpTriggerSound()   // :57-58
   8. tvInRoom <- false; tvWithMovieNumber <- -1     // :59-60
   9. roomV <- thisHouse->rooms[thisRoomNumber].floor  // :64
  10. for i = 0..8:                                   // :67-71
  11.     localNumbers[i] <- GetNeighborRoomNumber(i)   // :69
  12.     isStructure[i]  <- IsRoomAStructure(localNumbers[i])   // :70
  13. ListAllLocalObjects()                           // :72   builds masterObjects[], numMasterObjects
  14. GetGWorld(...); SetGWorld(backSrcMap); PaintRect(&backSrcRect)  // :74-76
  15. if numNeighbors > 3 then                        // :78   <== USER PREFERENCE
  16.     for W in [NorthWest, NorthEast, North, SouthWest, SouthEast, South]:   // :80-102
  17.         numLights <- GetNumberOfLights(localNumbers[W])
  18.         DrawRoomBackground(localNumbers[W], W, roomV +/- 1)   // +1 for N*, -1 for S*
  19.         DrawARoomsObjects(W, false)              // <== CONSUMES RNG
  20. if numNeighbors > 1 then                        // :105  <== USER PREFERENCE
  21.     for W in [West, East]:                       // :107-115
  22.         numLights <- GetNumberOfLights(localNumbers[W])
  23.         DrawRoomBackground(localNumbers[W], W, roomV)
  24.         DrawARoomsObjects(W, false)              // <== CONSUMES RNG
  25.         DrawLighting()                           // :110 and :115 — once per side room
  26. numLights <- GetNumberOfLights(localNumbers[kCentralRoom])   // :118
  27. DrawRoomBackground(localNumbers[kCentralRoom], kCentralRoom, roomV)  // :119
  28. DrawARoomsObjects(kCentralRoom, false)          // :120  <== CONSUMES RNG
  29. DrawLighting()                                  // :121
  30. if numNeighbors > 3 then DrawFloorSupport()     // :123-124
  31. RestoreWorkMap()                                // :125
  32. shadowVisible <- IsShadowVisible()              // :126
  33. takingTheStairs <- false                        // :127
  34. SetGWorld(wasCPort, wasWorld)                   // :129
```

The neighbour order is **fixed and non-obvious**: NW, NE, N, SW, SE, S, then W, E, then
Central. Any port must reproduce that order exactly if it wants the RNG stream to match,
because each of those nine calls can draw from the stream.

`numNeighbors` comes from the user's preferences file:

* Loaded: `numNeighbors = thePrefs.wasNumNeighbors;` (`GliderPRO/Sources/Main.c:111`).
* Default when no prefs exist: `numNeighbors = 9;` (`GliderPRO/Sources/Main.c:154`).
* **Forced down by screen size:**
  `if ((numNeighbors > 1) && (thisMac.screen.right <= 512)) numNeighbors = 1;`
  (`GliderPRO/Sources/Main.c:191-192`).

So `numNeighbors` is one of `{1, 3-or-less, 9}` in practice, and the branch tests are
`> 3` and `> 1`. **Two machines with different `numNeighbors` — because the user changed a
setting, or because one has a 512-pixel-wide screen — draw a different number of
`RandomInt` values on every single room load.** After the first room transition their
streams are permanently offset.

### What `DrawARoomsObjects` does, and its RNG sites

`DrawARoomsObjects(short neighbor, Boolean redraw)`
(`GliderPRO/Sources/ObjectDrawAll.c:23-965`):

```
DrawARoomsObjects(neighbor, redraw):
   1. if localNumbers[neighbor] == kRoomIsEmpty (-1) then return    // :33-34
   2. testRect <- houseRect ; ZeroRectCorner(&testRect)             // :36-37
   3. isLit <- (numLights > 0)                                      // :38
   4. for i = 0 .. kMaxRoomObs-1 (0..23):                           // :43
   5.     dynamicNum <- -1 ; legit <- -1                            // :45-46
   6.     if IsThisValid(localNumbers[neighbor], i) then            // :48
   7.         switch on object type -> per-type draw + per-type registration
   8.     if !redraw then                                            // :953   <-- INSIDE the i loop
   9.         for n = 0 .. numMasterObjects-1:                       // :955
  10.             if masterObjects[n].objectNum == i and
  11.                masterObjects[n].roomNum == localNumbers[neighbor] then
  12.                 masterObjects[n].dynaNum <- dynamicNum          // :959
  13. HSetState((Handle)thisHouse, wasState)                         // :964
```

**The link fixup at step 8 is inside the `for i` loop but *outside* the `IsThisValid` test**
(the `if (IsThisValid(...))` block closes at `:951`, the fixup runs at `:953-961`, the loop
closes at `:962`). Consequence: for an object slot that fails `IsThisValid`, `dynamicNum` is
still `-1` from step 5 and the fixup **still runs**, writing `dynaNum = -1` into every
`masterObjects[]` entry that happens to match `(objectNum == i, roomNum == …)`. A port that
hoists the fixup out of the loop, or that puts it inside the validity test, will produce a
different `masterObjects[].dynaNum` table.

The registration calls that consume RNG, and their positional arguments:

| Object type (hex) | Registration call | Line | Offsets | Guard |
| --- | --- | --- | --- | --- |
| `kTaper` 0x08 | `AddCandleFlame` | `:83` (central) / `:98` (neighbour) | `itsRect.left+10, itsRect.top+7` | see two-branch note below |
| `kCandle` 0x09 | `AddCandleFlame` | `:117` / `:132` | `+14, +7` | ditto |
| `kStubby` 0x0A | `AddCandleFlame` | `:151` / `:166` | `+9, +7` | ditto |
| `kTiki` 0x0B | `AddTikiFlame` | `:181` | `+10, -9` | any neighbour, **no `SectRect` visibility guard at all** (`:173-183`) — an off-screen tiki still draws RNG |
| `kBBQ` 0x0C | `AddBBQCoals` | `:195` | `+16, +9` | any neighbour, `SectRect(&itsRect,&testRect)` at `:188` |
| `kCuckoo` 0x24 | `AddPendulum` | `:343` | `+4, +46` | any neighbour, `SectRect` at `:331` **and** its own `BackUpToSavedMap` at `:336` returning `legit != -1` (`:337`) |
| `kStar` 0x2C | `AddStar` | `:440` | — | any neighbour, `SectRect` at `:429` **and** its own `BackUpToSavedMap` at `:434` returning `legit != -1` (`:435`) |
| `kSparkle` 0x2D | `AddDynamicObject` | `:456` | — | `SectRect` at `:450` and `(!redraw) && (neighbor == kCentralRoom)` at `:452` — **`kCentralRoom` only** |

**The three candle types take one of two registration paths depending on the neighbour, and
this was previously mis-stated as "any".** The structure (kTaper shown; kCandle and kStubby
are identical with different offsets) is:

```
case kTaper:                                      // :73
  if (isLit) DrawTaper(...)                       // :76
  if (neighbor == kCentralRoom)                   // :78
      if (redraw) ReBackUpFlames(...) else AddCandleFlame(...)   // :80 / :83
  else
      rectA = itsRect ; rectB = localRoomsDest[kCentralRoom]     // :89-90
      if (!SectRect(&rectA, &rectB, &whoCares))   // :93   <== NOTE THE NEGATION
          if (redraw) ReBackUpFlames(...) else AddCandleFlame(...)  // :95 / :98
```

So a candle in a *neighbour* room registers its flame — and therefore draws from the RNG —
**only if its rect does not overlap the central room's on-screen rect**. This is an
anti-double-draw guard, but it makes RNG consumption depend on the geometric relationship
between a neighbour room's candle positions and the central room's viewport. A port must
reproduce the negation and `localRoomsDest[kCentralRoom]`.

Note also that `kCuckoo` and `kStar` each consume **two** saved-map slots: one from their own
`BackUpToSavedMap` call in `DrawARoomsObjects` (`:336`, `:434`) and one from the
`BackUpToSavedMap` inside `AddPendulum`/`AddStar`. That doubles their cost against the
24-slot budget below.

And the registration calls that do **not** consume RNG but which do allocate saved maps
(and therefore change whether *later* calls can consume RNG):

| Object type | Call | Line | Neighbour restriction |
| --- | --- | --- | --- |
| `kGreaseRt` 0x28 | `AddGrease` | `:376` | any |
| `kGreaseLf` 0x29 | `AddGrease` | `:397` | any |
| `kToaster` 0x62 | `AddDynamicObject` | `:652` | `kCentralRoom` only |
| `kMacPlus` 0x63 | `AddDynamicObject` | `:668` | any |
| `kTV` 0x65 | `AddDynamicObject` | `:701` | any |
| `kCoffee` 0x66 | `AddDynamicObject` | `:725` | any |
| `kOutlet` 0x67 | `AddDynamicObject` | `:742` | any |
| `kVCR` 0x68 | `AddDynamicObject` | `:758` | any |
| `kStereo` 0x69 | `AddDynamicObject` | `:774` | any |
| `kMicrowave` 0x6A | `AddDynamicObject` | `:790` | any |
| `kBalloon` 0x71 | `AddDynamicObject` | `:802` | `kCentralRoom` only (`:797`) |
| `kCopterLf` 0x72 | `AddDynamicObject` | `:813` | **`kCentralRoom` only** (`:808`) |
| `kCopterRt` 0x73 | `AddDynamicObject` | `:824` | **`kCentralRoom` only** (`:819`) |
| `kDartLf` 0x74 | `AddDynamicObject` | `:835` | **`kCentralRoom` only** (`:830`) |
| `kDartRt` 0x75 | `AddDynamicObject` | `:846` | **`kCentralRoom` only** (`:841`) |
| `kBall` 0x76 | `AddDynamicObject` | `:857` | **`kCentralRoom` only** (`:852`) |
| `kDrip` 0x77 | `AddDynamicObject` | `:872` | `kCentralRoom` only (`:868`) |
| `kFish` 0x78 | `AddDynamicObject` | `:888` | `kCentralRoom` only (`:884`) |

(The five rows in bold were previously listed as "any". All five are guarded by
`if ((neighbor == kCentralRoom) && (!redraw))` — so the moving enemies exist only in the
room the glider is actually in, and neighbour rooms show their static sprite. This matters
for the `kMaxDynamicObs` = 18 budget as well as for `evenFrame`: the `kBall` branch's
`evenFrame = true;` write at `Dynamics3.c:474` therefore fires at most once per room load
per ball in the *central* room.)

**When `redraw == true`, every one of the RNG-consuming calls is replaced by its
`ReBackUp*` counterpart, which consumes no RNG.** `RedrawRoomLighting`
(`GliderPRO/Sources/RoomGraphics.c:434`) calls `DrawARoomsObjects(kCentralRoom, true)`, so
flipping a light switch does *not* advance the RNG. That asymmetry must be preserved.

### The saved-map budget, and why it gates RNG

Five of the six Group-B RNG sites only fire if `BackUpToSavedMap` succeeded:

```c
short BackUpToSavedMap (Rect *theRect, short where, short who)   // DynamicMaps.c:70
{
	Rect		mapRect;
	OSErr		theErr;

	if (numSavedMaps >= kMaxSavedMaps)               // DynamicMaps.c:75
		return(-1);                                  // :76  <== THE ONLY -1 RETURN

	mapRect = *theRect;
	ZeroRectCorner(&mapRect);
	savedMaps[numSavedMaps].dest = *theRect;
	theErr = CreateOffScreenGWorld(&savedMaps[numSavedMaps].map,
	                               &mapRect, kPreferredDepth);   // :82  theErr IS DISCARDED
	CopyBits(...);                                   // :84-86
	savedMaps[numSavedMaps].where = where;           // :88
	savedMaps[numSavedMaps].who = who;               // :89
	numSavedMaps++;                                  // :90
	return (numSavedMaps - 1);                       // :92
}
```

**`theErr` is assigned at `:82` and never read.** There is exactly one `return(-1)` in the
function, at `:76`, and it is reached only by the `numSavedMaps >= kMaxSavedMaps` cap. A
failed `NewGWorld` therefore does **not** make `BackUpToSavedMap` return -1: the function
carries on, `CopyBits` through a nil/garbage GWorld, still increments `numSavedMaps`, and
still returns a valid slot index — so **the RNG draw still happens**. Free memory does not
gate the RNG stream. (This corrects hazard 6 in §7.2 and the "whether a `NewGWorld`
allocation succeeded" bullet below.)

`kMaxSavedMaps` = **24** (`GliderPRO/Headers/GliderDefines.h:260`). `kPreferredDepth` =
**8** (`GliderPRO/Headers/Externs.h:15`).

`AddCandleFlame` (`GliderPRO/Sources/DynamicMaps.c:316-343`) in full pseudocode:

```
AddCandleFlame(where, who, h, v):
   1. if (numFlames >= kMaxCandles (20)) or (h < 16) or (v < 15) then return
   2. QSetRect(&src, 0, 0, 16, 15) ; QOffsetRect(&src, h-8, v-15)
   3. if (thisMac.isDepth == 4) and ((src.left % 2) == 1) then       // 4-bit colour fixup
   4.     QOffsetRect(&src, -1, 0)
   5.     if src.left < 0 then QOffsetRect(&src, 2, 0)
   6. QSetRect(&bounds, 0, 0, 16, 15 * kNumCandleFlames (5))          // 16 x 75
   7. savedNum <- BackUpToSavedMap(&bounds, where, who)
   8. if savedNum != -1 then
   9.     BackUpFlames(&src, savedNum)
  10.     flames[numFlames].dest <- src                              // :337
  11.     flames[numFlames].mode <- RandomInt(kNumCandleFlames)       // :338 <== RNG
  12.     QSetRect(&flames[n].src, 0,0,16,15)                         // :339
  13.     QOffsetRect(&flames[n].src, 0, flames[n].mode * 15)         // :340  mode selects the sprite row
  14.     flames[numFlames].who  <- savedNum                          // :341
  15.     numFlames++                                                 // :342
```

Function body is `DynamicMaps.c:316-344`; the RNG draw at `:338` is the only one.

So RNG consumption during a room load depends on:

* the user preference `numNeighbors` (how many neighbour rooms get drawn at all),
* the machine's colour depth `thisMac.isDepth` (which shifts `src.left` and can push
  `h`/`v` across the `h < 16` / `v < 15` thresholds),
* whether `numSavedMaps` has hit 24 — which depends on the *order* in which objects were
  encountered, which depends on `numNeighbors` and on the house's per-room object arrays,
* ~~whether a `NewGWorld` allocation succeeded~~ — **not a factor.** `CreateOffScreenGWorld`
  does first try `useTempMem` and then retry with `0`
  (`GliderPRO/Sources/Utilities.c:270-273`), but `BackUpToSavedMap` discards its `OSErr`
  (`DynamicMaps.c:82`), so allocation failure is invisible to the caller and cannot suppress
  an RNG draw. Available memory does **not** affect the stream.
* the per-type caps: `kMaxCandles` 20, `kMaxTikis` 8, `kMaxCoals` 8, `kMaxPendulums` 8,
  `kMaxStars` 4, `kMaxDynamicObs` 18.

**This is the core finding of Part 2: room-load RNG consumption is a function of the
machine, not just of the game state.**

### `AddPendulum` additionally resets a *global* animation counter

```
AddPendulum(where, who, h, v):
   1. if (numPendulums >= kMaxPendulums (8)) or (h < 32) or (v < 28) then return
   2. clockFrame <- 10                                              // DynamicMaps.c:578
   3. ... 32x28 rect, isDepth==4 shift ...
   4. savedNum <- BackUpToSavedMap(...)
   5. if savedNum != -1 then
   6.     pendulums[numPendulums].mode <- 1
   7.     if RandomInt(2) == 0 then toOrFro <- true else toOrFro <- false   // :594 <== RNG
   8.     pendulums[numPendulums].active <- true
   9.     src offset by 28
  10.     numPendulums++
```

Step 2's placement is exact and matters: `clockFrame = 10;` at `DynamicMaps.c:578` sits
*after* the early-return guard at `:575-576` but *before* the `savedNum != -1` test at
`:581`. So a cuckoo clock that passes the cap and size guards but then fails to get a
saved-map slot (because `numSavedMaps` has hit 24) **still resets the global `clockFrame`
to 10 without adding a pendulum** — and, because the guard already passed, without consuming
the `RandomInt(2)` at `:594`. `clockFrame` then drives `RenderPendulums` (§3.9.2).

---

# Part 3 — The exact per-frame update order

## 3.1 `PlayGame` — the authoritative loop

`GliderPRO/Sources/Play.c:430-556`. This is the single source of truth for update order.
Transcribed as numbered pseudocode with exact line citations; the two-player and
one-player arms are listed separately because they differ.

```
PlayGame():
   1. while (playing) and (!quitting):                            // Play.c:432
   2.     gameFrame++                                             // Play.c:434
   3.     evenFrame <- !evenFrame                                  // Play.c:435
   4.     if doBackground then                                     // Play.c:437
   5.         do { HandlePlayEvent() } while (switchedOut)          // Play.c:439-442
   6.     HandleTelephone()                                        // Play.c:445
   7.     if twoPlayerGame then                                    // Play.c:447
   8.         HandleDynamics()                                     // Play.c:449
   9.         if !gameOver then                                    // Play.c:450
  10.             GetInput(&theGlider)                             // Play.c:452
  11.             GetInput(&theGlider2)                            // Play.c:453
  12.             HandleInteraction()                              // Play.c:454
  13.         HandleTriggers()                                     // Play.c:456
  14.         HandleBands()                                        // Play.c:457
  15.         if !gameOver then                                    // Play.c:458
  16.             HandleGlider(&theGlider)                         // Play.c:460
  17.             HandleGlider(&theGlider2)                        // Play.c:461
  18.         if playing then                                      // Play.c:463
  19.             [COMPILEQT: MoviesTask(...) if tvInRoom]          // Play.c:465-468
  20.             RenderFrame()                                    // Play.c:469
  21.             HandleDynamicScoreboard()                        // Play.c:470
  22.     else                                                     // Play.c:473
  23.         HandleDynamics()                                     // Play.c:475
  24.         if !gameOver then                                    // Play.c:476
  25.             if demoGoing then GetDemoInput(&theGlider)       // Play.c:478-479
  26.             else GetInput(&theGlider)                        // Play.c:480-481
  27.             HandleInteraction()                              // Play.c:482
  28.         HandleTriggers()                                     // Play.c:484
  29.         HandleBands()                                        // Play.c:485
  30.         if !gameOver then HandleGlider(&theGlider)           // Play.c:486-487
  31.         if playing then                                      // Play.c:488
  32.             [COMPILEQT: MoviesTask(...) if tvInRoom]          // Play.c:490-493
  33.             RenderFrame()                                    // Play.c:494
  34.             HandleDynamicScoreboard()                        // Play.c:495
  35.     if gameOver then                                         // Play.c:499
  36.         countDown--                                          // Play.c:501
  37.         if countDown <= 0 then                               // Play.c:502
  38.             HideGlider(&theGlider)                            // Play.c:504
  39.             RefreshScoreboard(kNormalTitleMode)               // Play.c:505
  40.             [BUILD_ARCADE_VERSION: black out + redraw kScoreboardPictID (1997)]
  41.             if mortals < 0 then DoDiedGameOver() else DoGameOver()
```

### The canonical phase order

| Phase | Function | Two-player | One-player | Notes |
| --- | --- | --- | --- | --- |
| 0 | frame counters | `Play.c:434-435` | same | `gameFrame++`, `evenFrame` toggle |
| 1 | OS event pump | `Play.c:439` | same | **only if `doBackground`** |
| 2 | telephone / chimes | `Play.c:445` | `Play.c:445` | RNG consumer |
| 3 | dynamic objects | `Play.c:449` | `Play.c:475` | `HandleDynamics` — 17 entity types (15 `case` groups), see §3.3 |
| 4 | input sampling | `Play.c:452-453` | `Play.c:478-481` | **P1 then P2**; skipped if `gameOver` |
| 5 | interactions | `Play.c:454` | `Play.c:482` | hotspots, then room bounds |
| 6 | triggers | `Play.c:456` | `Play.c:484` | 16 timers, always runs even when `gameOver` |
| 7 | rubber bands | `Play.c:457` | `Play.c:485` | always runs even when `gameOver` |
| 8 | glider mode machine | `Play.c:460-461` | `Play.c:486-487` | **P1 then P2**; skipped if `gameOver` |
| 9 | render (mutates sim) | `Play.c:469` | `Play.c:494` | see §3.9 |
| 10 | scoreboard | `Play.c:470` | `Play.c:495` | `gameFrame & 7` slicing |
| 11 | game-over countdown | `Play.c:501` | same | |

Two structural observations a porter must not miss:

* **Phases 6 and 7 (`HandleTriggers`, `HandleBands`) run even when `gameOver` is set**,
  while phases 4, 5, 8 do not. So during the death countdown, triggers keep firing and
  rubber bands keep flying, but no input is read and no glider moves.
* **`HandleDynamics` runs before input is read.** Enemies move against the *previous*
  frame's glider position for collision purposes within `CheckDynamicCollision`, then the
  glider moves in phase 8 against the *new* enemy positions in the *next* frame's phase
  5. There is a systematic one-frame lag baked in and it is asymmetric between the two
  entity classes.

## 3.2 Phase 2 — `HandleTelephone`

Covered in §2.6. Consumes 0, 2, or 4 `RandomInt` draws per frame depending on whether the
ring burst completes and whether a chime strikes.

## 3.3 Phase 3 — `HandleDynamics` and the 17 dynamic-object types

**There are 17 dynamic (animated) object types, dispatched by 15 `case` groups** —
`kCopterLf`/`kCopterRt` share `HandleCopter` and `kDartLf`/`kDartRt` share `HandleDart`, so
the group count is two fewer than the type count. Both counts are pinned by the source: the
`HandleDynamics` switch at `GliderPRO/Sources/Dynamics3.c:39-99` lists 17 `case` labels in
15 groups, and `HowManyDynamicObjects` (`GliderPRO/Sources/ObjectAdd.c:1045-1071`) tests the
same 17 `what` codes in one `if` at `:1051-1067`. A port must size its dispatch table for 17
types, not 16.

`GliderPRO/Sources/Dynamics3.c:31-104`:

```
HandleDynamics():
   1. for i = 0 .. numDynamics-1:                        // Dynamics3.c:35
   2.     switch (dinahs[i].type):
   3.         kSparkle   (0x2D) -> HandleSparkleObject(i)  // :39-40
   4.         kToaster   (0x62) -> HandleToast(i)          // :43-44
   5.         kMacPlus   (0x63) -> HandleMacPlus(i)        // :47-48
   6.         kTV        (0x65) -> HandleTV(i)             // :51-52
   7.         kCoffee    (0x66) -> HandleCoffee(i)         // :55-56
   8.         kOutlet    (0x67) -> HandleOutlet(i)         // :59-60
   9.         kVCR       (0x68) -> HandleVCR(i)            // :63-64
  10.         kStereo    (0x69) -> HandleStereo(i)         // :67-68
  11.         kMicrowave (0x6A) -> HandleMicrowave(i)      // :71-72
  12.         kBalloon   (0x71) -> HandleBalloon(i)        // :75-76
  13.         kCopterLf  (0x72),
  14.         kCopterRt  (0x73) -> HandleCopter(i)         // :79-81
  15.         kDartLf    (0x74),
  16.         kDartRt    (0x75) -> HandleDart(i)           // :84-86
  17.         kBall      (0x76) -> HandleBall(i)           // :89-90
  18.         kDrip      (0x77) -> HandleDrip(i)           // :93-94
  19.         kFish      (0x78) -> HandleFish(i)           // :97-98
  20.         default -> break                             // :100-101
```

**Iteration order is array order, `0 .. numDynamics-1`.** The array (`dinahs`) is filled
by `AddDynamicObject` during room load in `DrawARoomsObjects`' object-index order (0..23)
for each neighbour room in `DrawLocale`'s neighbour order (NW, NE, N, SW, SE, S, W, E,
Central). So the dynamics array order is a deterministic function of (house data,
`numNeighbors`, allocation success). `kMaxDynamicObs` = 18
(`GliderPRO/Headers/GliderDefines.h:265`); overflow is silently dropped by
`if (numDynamics >= kMaxDynamicObs) return (-1);` at `GliderPRO/Sources/Dynamics3.c:193`.

Note that `HandleDynamics` **has no `active` check** — the per-type handler is responsible
for checking `dinahs[i].active` and `dinahs[i].moving`.

### 3.3.1 `dynaType` — the dynamic-object state record

Verified by compiling a transcription of the struct under `#pragma pack(2)` with a 32-bit
`long`. Observed size: **36 bytes**.

| Offset | Field | Type | Size | Meaning |
| --- | --- | --- | --- | --- |
| 0 | `dest` | `Rect` | 8 | current on-screen destination (`top,left,bottom,right` as `short`) |
| 8 | `whole` | `Rect` | 8 | union of old+new dest, used to build dirty rects |
| 16 | `hVel` | `short` | 2 | horizontal velocity, px/frame |
| 18 | `vVel` | `short` | 2 | vertical velocity, px/frame |
| 20 | `type` | `short` | 2 | object type code (`kBalloon` etc.) or `kObjectIsEmpty` (-1) |
| 22 | `count` | `short` | 2 | per-type reload/launch counter |
| 24 | `frame` | `short` | 2 | animation frame index |
| 26 | `timer` | `short` | 2 | per-type countdown |
| 28 | `position` | `short` | 2 | per-type spawn/limit coordinate |
| 30 | `room` | `short` | 2 | owning room number |
| 32 | `byte0` | `Byte` | 1 | per-type flag byte |
| 33 | `byte1` | `Byte` | 1 | per-type flag byte |
| 34 | `moving` | `Boolean` | 1 | is this object currently in motion |
| 35 | `active` | `Boolean` | 1 | is this object switched on |

Declared at `GliderPRO/Headers/GliderStructs.h:310-320`. `dinahs` is the global array,
declared at `GliderPRO/Sources/Dynamics3.c:18`.

### 3.3.2 `ZeroDinahs` does not zero everything

`GliderPRO/Sources/Dynamics3.c:160-180`:

```
ZeroDinahs():
   1. for i = 0 .. kMaxDynamicObs-1 (0..17):
   2.     dinahs[i].type <- kObjectIsEmpty (-1)
   3.     dinahs[i].dest <- {0,0,0,0}
   4.     dinahs[i].whole <- {0,0,0,0}
   5.     dinahs[i].hVel <- 0 ; dinahs[i].vVel <- 0
   6.     dinahs[i].count <- 0 ; dinahs[i].frame <- 0
   7.     dinahs[i].timer <- 0 ; dinahs[i].position <- 0
   8.     dinahs[i].room <- 0 ; dinahs[i].byte0 <- 0
   9.     dinahs[i].active <- false
  10. numDynamics <- 0                                  // Dynamics3.c:179
```

**`byte1` and `moving` are NOT reset.** They therefore retain whatever the *previous*
room's object at the same array index left there, until the per-type `AddDynamicObject`
branch overwrites them. Every `AddDynamicObject` branch that uses `moving` does set it,
so this is latent rather than live — but a Go port that zero-initialises everything (as Go
naturally does) is technically *diverging* from the C, and a Go port that faithfully
reproduces "don't reset these" would be reproducing a bug. Recommendation: zero
everything; note the divergence.

### 3.3.3 `AddDynamicObject` — per-type initialisation, in full

`GliderPRO/Sources/Dynamics3.c:187-554`. Signature:

```c
short AddDynamicObject (short what, Rect *where, objectType *who,
                        short room, short index, Boolean isOn)
```

Guard: `if (numDynamics >= kMaxDynamicObs) return (-1);`
(`GliderPRO/Sources/Dynamics3.c:193`). Tail: `numDynamics++;` at `:551`,
`return (numDynamics - 1);` at `:553`.

| Type | Branch line | Distinctive initialisation |
| --- | --- | --- |
| `kSparkle` 0x2D | `:199` | `timer = RandomInt(60) + 15;` at `:208` — the only RNG in this function |
| `kToaster` 0x62 | `:217` | reverse-integrates the toast launch velocity: `do { velocity++; position -= velocity; } while (position > 0);` at `:226-231`, then `vVel = -velocity`, `count = velocity`, `frame = (short)who->data.g.delay * 3` |
| `kMacPlus` 0x63 | `:244` | animation frame from `data.g.delay` |
| `kTV` 0x65 | `:264` | movie/`tvWithMovieNumber` wiring |
| `kCoffee` 0x66 | `:284` | |
| `kOutlet` 0x67 | `:307` | `count = ((short)who->data.g.delay * 6) / kTicksPerFrame;` at `:316` — note **`data.g`**, not `data.h` |
| `kVCR` 0x68 | `:327` | |
| `kStereo` 0x69 | `:350` | |
| `kMicrowave` 0x6A | `:370` | |
| `kBalloon` 0x71 | `:391` | `count = ((short)who->data.h.delay * 6) / kTicksPerFrame;` at `:401` |
| `kCopterLf`/`kCopterRt` 0x72/0x73 | `:412`/`:413` | `count = ((short)who->data.h.delay * 6) / kTicksPerFrame;` at `:426` |
| `kDartLf`/`kDartRt` 0x74/0x75 | `:437`/`:438` | `count = ((short)who->data.h.delay * 6) / kTicksPerFrame;` at `:456`. **This branch never assigns `.room = room;`** — the only branch that omits it (see §7.4) |
| `kBall` 0x76 | `:465` | `evenFrame = true;` at `:474`; half-rate reverse-integration `do { if (lilFrame) velocity++; lilFrame = !lilFrame; position -= velocity; } while (position > 0);` at `:476-483` |
| `kDrip` 0x77 | `:496` | `count = ((short)who->data.h.delay * 6) / kTicksPerFrame;` at `:504` |
| `kFish` 0x78 | `:516` | `hVel = ((short)who->data.h.delay * 6) / kTicksPerFrame;` at `:521`; `evenFrame = true;` at `:524`; half-rate reverse-integration at `:526-533` |

The reverse-integration loops are the mechanism by which an enemy's launch velocity is
derived from its *target apex*: the code integrates gravity backwards from rest until it
has "fallen" past the spawn `position`, and uses the accumulated velocity as the launch
impulse. Two variants exist: full-rate (`kToaster`) and half-rate (`kBall`, `kFish`,
matching their `if (evenFrame) vVel++` gravity). A port must use exactly the same loop
structure — an analytical `sqrt(2*g*h)` will differ by one unit in many cases.

`kTicksPerFrame` appears in **six** initialisation formulas in this function
(`GliderPRO/Sources/Dynamics3.c:316`, `:401`, `:426`, `:456`, `:504`, `:521`), all dividing
`delay * 6` by 2. So these are `delay * 3` in the shipped build. Five assign `.count`; the
sixth (`:521`, `kFish`) assigns `.hVel`. Five read `who->data.h.delay`; the first (`:316`,
`kOutlet`) reads `who->data.g.delay`. **A port that changes the frame rate must not change
`kTicksPerFrame` in these six formulas** or enemy reload timings shift. (This matches
Porting note 3, which enumerates the same six sites.)

### 3.3.4 Per-type dynamics handlers — the constants and the gates

Constants in `GliderPRO/Sources/Dynamics2.c:13-19`:

| Constant | Value | Used by |
| --- | --- | --- |
| `kBalloonStop` | 8 | `HandleBalloon` ceiling |
| `kBalloonStart` | 310 | `HandleBalloon` respawn floor (also `Dynamics3.c:13`) |
| `kCopterStart` | 8 | `HandleCopter` respawn ceiling (also `Dynamics3.c:14`) |
| `kCopterStop` | 310 | `HandleCopter` floor |
| `kDartVelocity` | 6 | `HandleDart` horizontal speed (also `Dynamics3.c:15`) |
| `kDartStop` | 310 | `HandleDart` floor |
| `kEnemyDropSpeed` | 8 | enemy drop speed |

`kShoveVelocity` = 8 (`GliderPRO/Sources/Dynamics.c:17`).

| Handler | Location | Frame/velocity rule | Respawn rule |
| --- | --- | --- | --- |
| `HandleSparkleObject` | `Dynamics.c:293` | if `frame <= 0`: `timer = RandomInt(240)+60` (`:304`), `frame = kNumSparkleModes` (5) (`:305`), `AddSparkle(&tempRect)` (`:307`), play `kMysticSound` (`:308`); else `frame--` (`:311`) | n/a |
| `HandleToast` | `Dynamics.c:321` | animation advances only `if (evenFrame)` (`:327`) | `kNumBreadPicts` = 6 frames |
| `HandleMacPlus` | `Dynamics.c:388` | | |
| `HandleTV` | `Dynamics.c:428` | | |
| `HandleCoffee` | `Dynamics.c:482` | `timer = 200 + RandomInt(200)` at both `:492` and `:506` | |
| `HandleOutlet` | `Dynamics.c:528` | `kNumOutletPicts` = 4 | |
| `HandleVCR` | `Dynamics.c:603` | | |
| `HandleStereo` | `Dynamics.c:671` | | |
| `HandleMicrowave` | `Dynamics.c:715` | | |
| `HandleBalloon` | `Dynamics2.c:30` | frame advance gated on `evenFrame` (`:38`, `:77`); band-hit check `(numBands > 0) && DidBandHitDynamic(who)` (`:62`) | bounds `:89-90`; respawn `vVel = -2`, `timer = count`, `dest.bottom = kBalloonStart` (`:99-104`); idle `timer--` (`:111`); pre-spawn sparkle at `timer == kStartSparkle` (4) (`:122-127`) |
| `HandleCopter` | `Dynamics2.c:134` | frame wraps at 8 (`:143`); shot frames 8..10 (`:183-185`) | bounds `:191-192`; respawn `vVel = 2`, `hVel = ±1`, `dest.left = position` (`:200-212`) |
| `HandleDart` | `Dynamics2.c:242` | | bounds `left <= 0 \|\| right >= kRoomWide (512) \|\| bottom >= kDartStop (310)` (`:296-298`); respawn frame 0 or 2, `hVel = ±kDartVelocity`, `dest.top = position` (`:306-328`) |
| `HandleBall` | `Dynamics2.c:358` | collision first (`:360-376`); if `moving`: `VOffsetRect(&dest, vVel)` (`:380`), bounce when `dest.bottom >= position` (`:381`) with `vVel = count` if active else `vVel = -((vVel*3)/4)` and stop at 0 (`:388-395`); else `if (evenFrame) vVel++` (`:409-410`) | idle+active branch **sets global `evenFrame = true`** (`:420`) |
| `HandleDrip` | `Dynamics2.c:427` | `if (evenFrame) frame = 9 - frame` (`:433-434`); `if (evenFrame) vVel++` (`:471-472`) | landing `:454-466`; idle timer triggers at 6, 4, 2, `<= 0` (`:481-494`) |
| `HandleFish` | `Dynamics2.c:501` | `if ((vVel >= 0) && (frame < 7)) frame++` (`:507-508`); `if (evenFrame) vVel++` (`:551-552`) | splashdown `:528-543` sets `vVel = count`, `timer = hVel`; idle bob when `(timer & 0x0003) == 0x0003` (`:558-575`); leap when `timer <= 0` (`:579`) |

### 3.3.5 `CheckDynamicCollision` and the foil-decay coupling

`GliderPRO/Sources/Dynamics.c:34-78`:

```
CheckDynamicCollision(who, thisGlider, doOffset):
   1. if glider mode not in {kGliderNormal, kGliderFaceLeft, kGliderFaceRight,
                             kGliderBurning} -> return false
   2. if !SectGlider(thisGlider, &dinahs[who].dest, false) -> return false
   3. if showFoil then
   4.     if dinahs[who].dest is left of glider centre then
   5.         thisGlider->hVel <- +kShoveVelocity (8)
   6.     else
   7.         thisGlider->hVel <- -kShoveVelocity (8)
   8.     if (evenFrame) and (foilTotal > 0) then foilTotal--       // Dynamics.c:60
   9.     play kFoilHitSound
  10. else
  11.     (kill the glider)
```

`foilTotal` — the aluminium-foil hit counter — decays **once per two frames while in
contact**, not once per collision. This makes foil duration a function of frame parity at
the moment of contact.

### 3.3.6 `DidBandHitDynamic` reads uninitialised memory when there are no bands

`GliderPRO/Sources/Dynamics.c:80-106`:

```c
Boolean DidBandHitDynamic (short who)
{
	short		i;
	Boolean		collided;                    // :84  <== NEVER INITIALISED

	for (i = 0; i < numBands; i++)           // :86
	{
		...
		collided = true;   /* or false */    // :91,:93,:95,:97,:99
		...
	}

	return (collided);                        // :105  <== reads garbage if numBands == 0
}
```

All three call sites short-circuit on `numBands`:

* `GliderPRO/Sources/Dynamics2.c:62` — `if ((numBands > 0) && (DidBandHitDynamic(who)))`
* `GliderPRO/Sources/Dynamics2.c:162` — same pattern
* `GliderPRO/Sources/Dynamics2.c:267` — same pattern

so the bug is **latent, not live**. A Go port will initialise `collided` to `false` and be
correct; a Go port that faithfully replicates undefined behaviour cannot. Record it and
move on.

## 3.4 Phase 4 — input sampling

### 3.4.1 Both players read the same physical keyboard

```c
void GetInput (gliderPtr thisGlider)
{
	...
	if (thisGlider->which == kPlayer1)     // Input.c:283
		GetKeys(theKeys);                  // Input.c:285  <== one global KeyMap
	...
}
```

`theKeys` is a single global `KeyMap` (`GliderPRO/Sources/Input.c:31`). `GetKeys` is
called **only** when `thisGlider->which == kPlayer1`
(`GliderPRO/Sources/Input.c:283-287`). `kPlayer1` is `TRUE` and `kPlayer2` is `FALSE`
(`GliderPRO/Headers/GliderDefines.h:556-557`), so `which` is a `Boolean` used as a player
index.

Because `PlayGame` calls `GetInput(&theGlider)` before `GetInput(&theGlider2)`
(`GliderPRO/Sources/Play.c:452-453`), player 1's call refreshes the shared `theKeys`
snapshot and player 2's call reads that same snapshot. **Two-player Glider PRO is a
same-keyboard local-multiplayer game**: each glider has its own four key *bindings*
(`leftKey`, `rightKey`, `battKey`, `bandKey` — four `long`s in `gliderType`,
`GliderPRO/Headers/GliderStructs.h:205-206`) tested against the shared KeyMap.

For a network port this is actually convenient: the per-glider input is already
parameterised by a key-binding record, so replacing "test bit in local KeyMap" with
"consume a 4-bit input word from the network" is a small change.

### 3.4.2 Input constants

| Constant | Value | Location |
| --- | --- | --- |
| `kNormalThrust` | 5 | `GliderPRO/Sources/Input.c:14` |
| `kHyperThrust` | 8 | `GliderPRO/Sources/Input.c:15` |
| `kHeliumLift` | 4 | `GliderPRO/Sources/Input.c:16` |

### 3.4.3 The demo replay path

`GetDemoInput` (`GliderPRO/Sources/Input.c:186-278`) is the *existing, shipped proof* that
this simulation is input-deterministic. Pseudocode:

```
GetDemoInput(thisGlider):
   1. if thisGlider->which == kPlayer1 then
   2.     GetKeys(theKeys)
   3. #if BUILD_ARCADE_VERSION                                   // Input.c:191
   4.     if any of leftKey/rightKey/battKey/bandKey is held then
   5.         playing <- false ; paused <- false                  // abort attract mode
   6. #else
   7.     if BitTst(&theKeys, kCommandKeyMap) then DoCommandKey()
   8. #endif
   9. if glider is burning then (special-case handling)
  10. else
  11.     heldLeft <- false ; heldRight <- false ; tipped <- false
  12.     if gameFrame == (long)demoData[demoIndex].frame then     // Input.c:224
  13.         switch (demoData[demoIndex].key):                    // Input.c:226
  14.             case 0: heldLeft <- true ; hDesiredVel += kNormalThrust (5)
  15.             case 1: heldRight <- true ; hDesiredVel -= kNormalThrust (5)
  16.             case 2: (battery / helium engage)
  17.             case 3: AddBand(thisGlider, dest.left + 24, dest.top + 10, facing)
  18.                     bandsTotal--
  19.         demoIndex++                                          // Input.c:266
  20.     else
  21.         fireHeld <- false
  22. (Esc / Tab pause check)
```

Note the sign convention: **`hDesiredVel += kNormalThrust` for LEFT** and
`hDesiredVel -= kNormalThrust` for RIGHT. This is inverted relative to screen coordinates
because `MoveGlider` applies `hVel` directly to `dest.left`/`dest.right`, and the sign is
flipped somewhere in the mode handlers. A porter must transcribe the signs literally; do
not "fix" them.

`LogDemoKey` (`GliderPRO/Sources/Input.c:44-49`), the recording counterpart:

```c
void LogDemoKey (char keyIs)
{
	demoData[demoIndex].frame = gameFrame;   // Input.c:46
	demoData[demoIndex].key = keyIs;         // Input.c:47
	demoIndex++;                             // Input.c:48
}
```

It never writes `.padding` — which is why the shipped `'demo'` resource contains
uninitialised garbage in every padding byte (verified empirically in §8).

### 3.4.4 `DoPause` blocks the entire loop

`DoPause` (`GliderPRO/Sources/Input.c:77-118`) is a **blocking spin** on `GetKeys` until
the pause key is released and pressed again. While paused, `gameFrame` does not advance,
`RenderFrame` is not called, and `nextFrame` is stale — which the assign-not-accumulate
pacer at `GliderPRO/Sources/Render.c:665` absorbs harmlessly.

For a networked port, a local blocking pause is a desync source (peer A pauses, peer B
does not). See Part 10 in `determinism-networking.md` — §10.2.7 lists every blocking call a
networked build must remove.

## 3.5 Phase 5 — `HandleInteraction`

`GliderPRO/Sources/Interactions.c:1691-1711` (the declaration is at `:1691`, the closing
brace at `:1711`; `:1712` is the blank line before the `FlagStillOvers` banner at `:1713`):

```
HandleInteraction():
   1. CheckForHotSpots()                                      // Interactions.c:1693
   2. if twoPlayerGame then                                   // :1694
   3.     if onePlayerLeft then                               // :1696
   4.         CheckGliderInRoom(surviving glider only)        // :1699 / :1701
   5.     else
   6.         CheckGliderInRoom(&theGlider)                    // :1705
   7.         CheckGliderInRoom(&theGlider2)                   // :1706
   8. else
   9.     CheckGliderInRoom(&theGlider)                        // :1710
```

### 3.5.1 `CheckForHotSpots` — hotspot iteration order

`GliderPRO/Sources/Interactions.c:1627-1687`:

```
CheckForHotSpots():
   1. for i = 0 .. nHotSpots-1:                              // :1632
   2.     if !hotSpots[i].isOn then continue
   3.     if twoPlayerGame then
   4.         hitObject <- false                              // inside the twoPlayerGame branch
   5.         if SectGlider(&theGlider, &hotSpots[i].bounds, hotSpots[i].doScrutinize) then
   6.             if onePlayerLeft then
   7.                 if playerDead == kPlayer2 then
   8.                     HandleHotSpotCollision(&theGlider, &hotSpots[i], i) ; hitObject <- true
   9.             else
  10.                 HandleHotSpotCollision(&theGlider, &hotSpots[i], i) ; hitObject <- true
  11.         if SectGlider(&theGlider2, &hotSpots[i].bounds, hotSpots[i].doScrutinize) then
  12.             if onePlayerLeft then
  13.                 if playerDead == kPlayer1 then
  14.                     HandleHotSpotCollision(&theGlider2, &hotSpots[i], i) ; hitObject <- true
  15.             else
  16.                 HandleHotSpotCollision(&theGlider2, &hotSpots[i], i) ; hitObject <- true
  17.         if !hitObject then hotSpots[i].stillOver <- false      // :1674-1675
  18.     else
  19.         if SectGlider(&theGlider, ...) then HandleHotSpotCollision(&theGlider, &hotSpots[i], i)
  20.         else hotSpots[i].stillOver <- false                    // :1683
```

Note the nesting: the `SectGlider` test is the **outer** condition and the
`onePlayerLeft`/`playerDead` test is nested **inside** it, not the other way round.
`SectGlider` is a pure predicate so this does not change behaviour, but a port that hoists
the aliveness test outward must keep the collision handler's guard semantics identical.
Note also the signature: `HandleHotSpotCollision(gliderPtr, hotPtr, short index)` takes
**three** arguments — the hotspot pointer *and* its index.

`kMaxHotSpots` = **56** (`GliderPRO/Headers/GliderDefines.h:259`). Iteration is strictly
by increasing index, **player 1 checked before player 2 for each hotspot**. The `hotSpots`
array is populated by `DrawARoomsObjects` in room-load order, so hotspot index order is
part of the deterministic room-load state.

`hotObject` is a `Boolean` used to clear `stillOver`. `stillOver` is the anti-retrigger
latch: it prevents a switch or trigger from firing every frame while the glider sits on
it. `FlagStillOvers` (`GliderPRO/Sources/Interactions.c:1715`) is the bulk reset.

### 3.5.2 `SectGlider` — the collision predicate

`GliderPRO/Sources/Interactions.c:101-130`:

```
SectGlider(thisGlider, theRect, scrutinize):
   1. glideBounds <- thisGlider->dest
   2. if thisGlider->mode == kGliderBurning then glideBounds.top += 6
   3. if scrutinize then                                      // :110-116, hand-rolled inset
   4.     glideBounds.left += 5; glideBounds.top += 5
   5.     glideBounds.right -= 5; glideBounds.bottom -= 5
   6. if      theRect->bottom < glideBounds.top    then itHit <- false   // :118
   7. else if theRect->top    > glideBounds.bottom then itHit <- false   // :120
   8. else if theRect->right  < glideBounds.left   then itHit <- false   // :122
   9. else if theRect->left   > glideBounds.right  then itHit <- false   // :124
  10. else                                              itHit <- true    // :126-127
  11. return itHit
```

The 5-pixel `scrutinize` inset makes tight-fit hotspots (transports, mailboxes) require
the glider to be properly centred. `doScrutinize` is a per-hotspot `Boolean`
(`GliderPRO/Headers/GliderStructs.h:224`), set during room load.

**`SectGlider` does not call `SectRect`.** It hand-rolls the test with four
non-strict comparisons, which makes it a **CLOSED-interval** predicate: rects that merely
*touch* count as intersecting. Concretely, `theRect->bottom == glideBounds.top` is a hit,
and a degenerate/empty rect (e.g. `left == right`) still collides. This is the opposite
of QuickDraw `SectRect`, which uses strict inequality on half-open intervals
(`left < other.right && other.left < right && top < other.bottom && other.top < bottom`)
and returns `false` for empty rects.

A Go port must therefore implement **two distinct rect predicates** and never substitute
one for the other:

| Predicate | Semantics | Used for |
| --- | --- | --- |
| `SectGlider` (`Interactions.c:101`) | closed interval; touching edges collide; empty rects can still hit | all glider-vs-object gameplay collision |
| `SectRect` (QuickDraw) | half-open, strict inequality; empty rects never intersect | dirty-rect culling, room-visibility tests, the candle registration test in `ObjectDrawAll.c` |

Using `SectRect` for gameplay collision would shrink every hitbox by one pixel on each
side and desynchronise the port immediately.

### 3.5.3 `CheckGliderInRoom` — the room-boundary state machine

`GliderPRO/Sources/Interactions.c:689-752` (header `:689`, closing brace `:752`):

```
CheckGliderInRoom(thisGlider):
   1. if thisGlider->mode not in {kGliderNormal, kGliderFaceLeft,
                                  kGliderFaceRight, kGliderBurning} then return
   2. if thisGlider->dest.top < kCeilingLimit (8) then
   3.     if mode == kGliderBurning then StartGliderFadingOut(thisGlider)
   4.     else if (twoPlayerGame) and (!onePlayerLeft) then CheckEscapeUpTwo(thisGlider)  // :704-705
   5.     else CheckEscapeUp(thisGlider)                                                  // :706-707
   6. else if thisGlider->dest.bottom > kFloorLimit (312) then
   7.     (same three-way dispatch for Down)                                              // :717-720
   8. else if (thisBackground == kRoof (2014)) and
              (thisGlider->dest.bottom > kRoofLimit (122)) then
   9.     CheckRoofCollision(thisGlider)
  10. // SEPARATE if-chain, not else-if:
  11. if thisGlider->dest.left < leftThresh then
  12.     (three-way dispatch for Left)                                                   // :733-736
  13. else if thisGlider->dest.right > rightThresh then
  14.     (three-way dispatch for Right)                                                  // :746-749
```

**Vertical and horizontal escapes are checked in two separate `if` chains**, so a glider
in a corner can trigger both a vertical and a horizontal escape in the same frame. The
vertical one runs first.

Boundary constants (`GliderPRO/Headers/GliderDefines.h:503-511`):

| Constant | Value | Meaning |
| --- | --- | --- |
| `kCeilingLimit` | 8 | glider `dest.top` above this → ceiling event |
| `kFloorLimit` | 312 | glider `dest.bottom` **greater** than this → floor event |
| `kRoofLimit` | 122 | roof-background collision line |
| `kLeftWallLimit` | 12 | `leftThresh` when the room has a left wall |
| `kNoLeftWallLimit` | -24 | `leftThresh` when it does not (= `0 - kGliderWide/2`) |
| `kRightWallLimit` | 500 | `rightThresh` with a right wall |
| `kNoRightWallLimit` | 536 | without (= `kRoomWide + kGliderWide/2` = 512 + 24) |
| `kNoCeilingLimit` | -10 | escape-up trigger line when the ceiling is open |
| `kNoFloorLimit` | 332 | escape-down trigger line when the floor is open |

`leftThresh` and `rightThresh` are globals (`GliderPRO/Sources/Room.c:30`) set by
`DetermineRoomOpenings` (`GliderPRO/Sources/Room.c:816-933`) on every room load.

### 3.5.4 The two-player escape handshake — the original's rendezvous protocol

This is the single most important piece of prior art for a networked port, because it is
already a distributed-agreement protocol implemented in shared memory.

`CheckEscapeUpTwo` (`GliderPRO/Sources/Interactions.c:171-240`), transcribed:

```
CheckEscapeUpTwo(thisGlider):
   1. if topOpen then                                                    // :175
   2.     if thisGlider->dest.top < kNoCeilingLimit (-10) then           // :177
   3.         if otherPlayerEscaped == kNoOneEscaped (-1) then           // :179
   4.             otherPlayerEscaped <- kPlayerEscapedUp (-4)             // :181
   5.             RefreshScoreboard(kEscapedTitleMode)                    // :182
   6.             FlagGliderInLimbo(thisGlider, true)                     // :183
   7.         else if otherPlayerEscaped == kPlayerEscapedUp then         // :185
   8.             otherPlayerEscaped <- kNoOneEscaped                     // :187
   9.             MoveRoomToRoom(thisGlider, kAbove)                      // :188
  10.         else
  11.             PlayPrioritySound(kDontExitSound, kDontExitPriority)     // :192
  12.             offset <- kNoCeilingLimit - thisGlider->dest.top         // :193
  13.             thisGlider->vVel <- -thisGlider->vVel + offset           // :194
  14. else if thisBackground == kDirt (2011) then                         // :198
  15.     leftTile  <- thisGlider->dest.left  >> 6                        // :200, tile index
  16.     rightTile <- thisGlider->dest.right >> 6                        // :201
  17.     if both tile indices are in [0,8) then                          // :203-204
  18.         if thisTiles[leftTile] and thisTiles[rightTile] are each 5 or 6 then   // :206-209
  19.             (identical three-way handshake, :211-229; limbo at :217)
  20.         else thisGlider->vVel <- kCeilingLimit - thisGlider->dest.top  // :233
  21.     else thisGlider->vVel <- kCeilingLimit - thisGlider->dest.top      // :236
  22. else thisGlider->vVel <- kCeilingLimit - thisGlider->dest.top          // :239
```

Note there are **three** separate `vVel <- kCeilingLimit - dest.top` fallbacks (`:233`,
`:236`, `:239`), one for each way the ceiling can turn out to be solid: wrong dirt tile,
tile index out of range, and a non-dirt closed ceiling. All three are the same expression,
so a port may collapse them, but the *dirt tile bounds check* (`:203-204`) must be kept —
without it a glider straddling the room edge would index `thisTiles` out of range.

The protocol is a **three-state rendezvous on a single shared variable**
`otherPlayerEscaped`:

| State value | Constant | Meaning |
| --- | --- | --- |
| -1 | `kNoOneEscaped` | nobody is waiting |
| -2 | `kPlayerEscapedRight` | one player is in limbo, having exited right |
| -3 | `kPlayerEscapedLeft` | ... left |
| -4 | `kPlayerEscapedUp` | ... up through the ceiling |
| -5 | `kPlayerEscapedDown` | ... down through the floor |
| -6 | `kPlayerEscapedUpStairs` | ... up the stairs |
| -7 | `kPlayerEscapedDownStairs` | ... down the stairs |
| -8 | `kPlayerEscapingUpStairs` | in transit up the stairs |
| -9 | `kPlayerEscapingDownStairs` | in transit down the stairs |
| -10 | `kPlayerTransportedOut` | ... via a transporter |
| -11 | `kPlayerDuckedOut` | ... via a duct |
| -12 | `kPlayerMailedOut` | ... via a mailbox |
| -69 | `kPlayerIsDeadForever` | the other player is permanently gone |

(`GliderPRO/Headers/GliderDefines.h:596-608`.)

Semantics: the **first** glider to reach an exit sets `otherPlayerEscaped` to the exit it
used and goes into limbo (`kGliderInLimbo` = 21). The **second** glider must reach the
*same* exit; when it does, the flag is cleared and *both* gliders move to the new room.
If the second glider reaches a *different* exit, it is bounced back with `kDontExitSound`
— the game refuses to split the pair. This is a **shared-fate** design, which maps
directly onto requirement (a) in the assignment.

The four `CheckEscape*Two` functions and their `FlagGliderInLimbo` sites:

| Direction | Function | Line | `FlagGliderInLimbo` at |
| --- | --- | --- | --- |
| Up | `CheckEscapeUpTwo` | `Interactions.c:171` | `:183` (open ceiling), `:217` (dirt-hole path) |
| Down | `CheckEscapeDownTwo` | `Interactions.c:283` | `:295`, `:329` |
| Left | `CheckEscapeLeftTwo` | `Interactions.c:509` | `:523` (`leftThresh == kLeftWallLimit` and `ignoreLeft`), `:554` (`leftThresh != kLeftWallLimit`) |
| Right | `CheckEscapeRightTwo` | `Interactions.c:599` | `:613` (`rightThresh == kRightWallLimit` and `ignoreRight`), `:644` (otherwise) |

Left and Right each have **two** limbo sites, not one: the `*Thresh == *WallLimit` path
(the room has a wall, but this glider has `ignoreLeft`/`ignoreRight` set because it came
through a door) and the plain-open-side path in the outer `else`. Both run the same
three-way handshake.

And their single-player counterparts, which move the room immediately with no handshake:
`CheckEscapeUp` `:244`, `CheckEscapeDown` `:378`, `CheckEscapeLeft` `:572`,
`CheckEscapeRight` `:662`.

`FlagGliderInLimbo` (`GliderPRO/Sources/Modes.c:458-468`):

```
FlagGliderInLimbo(thisGlider, sayIt):
   1. thisGlider->wasMode <- thisGlider->mode
   2. thisGlider->mode <- kGliderInLimbo (21)
   3. if (sayIt) and (saidFollow < 3) then
   4.     PlayPrioritySound(kFollowSound (7), kFollowPriority (904))
   5.     saidFollow++
   6. firstPlayer <- thisGlider->which
```

`saidFollow` (`GliderPRO/Sources/Modes.c:13`) caps the "follow me" voice sample at 3 plays
per game; it is reset by `saidFollow = 0;` at `GliderPRO/Sources/Play.c:115`.
`firstPlayer` records who is waiting — used by `MoveRoomToRoom`'s tail to decide which
glider gets idled on arrival.

`UndoGliderLimbo` (`GliderPRO/Sources/Modes.c:472-480`):

```
UndoGliderLimbo(thisGlider):
   1. if (twoPlayerGame) and (onePlayerLeft) and
        (thisGlider->which == playerDead) then return
   2. if thisGlider->mode == kGliderInLimbo then thisGlider->mode <- thisGlider->wasMode
   3. thisGlider->dontDraw <- false
```

`TagGliderIdle` (`GliderPRO/Sources/Modes.c:631-639`):

```
TagGliderIdle(thisGlider):
   1. if (twoPlayerGame) and (onePlayerLeft) and
        (thisGlider->which == playerDead) then return
   2. thisGlider->wasMode <- thisGlider->mode
   3. thisGlider->mode <- kGliderIdle (22)
   4. thisGlider->hVel <- 30            // reused as a 30-frame delay counter
```

`hVel` is **repurposed as a countdown** while in `kGliderIdle` mode. `HandleIdleGlider`
(`GliderPRO/Sources/Player.c:1323-1331`):

```
HandleIdleGlider(thisGlider):
   1. thisGlider->hVel--
   2. if thisGlider->hVel <= 0 then
   3.     thisGlider->mode <- thisGlider->wasMode
   4.     thisGlider->dontDraw <- false
```

So the second glider to arrive in a new room is frozen for exactly **30 frames** (1 s at
30 Hz) before it becomes controllable. That is a deterministic, frame-counted grace period
— useful as a natural sync window for a networked port.

### 3.5.5 Interaction constants

`GliderPRO/Sources/Interactions.c:13-19`:

| Constant | Value | Effect |
| --- | --- | --- |
| `kFloorVentLift` | -6 | upward velocity imparted by a floor vent |
| `kCeilingVentDrop` | 8 | downward velocity from a ceiling vent |
| `kFanStrength` | 12 | horizontal velocity from a fan |
| `kBatterySupply` | 50 | units added by a battery pickup |
| `kHeliumSupply` | 150 | units added by a helium pickup |
| `kBandsSupply` | 8 | rubber bands added by a bands pickup |
| `kFoilSupply` | 8 | foil hits added by a foil pickup |

`kKillWebbedGlider` = 150 (`WebGlider`, `GliderPRO/Sources/Interactions.c:1736`-region) —
the cobweb struggle counter, advanced on `evenFrame` only, with a `>> 3` divisor on the
struggle amount.

`HandleMicrowaveAction` (`GliderPRO/Sources/Interactions.c:1159`) uses three bit masks to
decide what a microwave kills: `0x0001`, `0x0002`, `0x0004`.

## 3.6 Phase 6 — `HandleTriggers`

`GliderPRO/Sources/Triggers.c:79-96`:

```
HandleTriggers():
   1. for i = 0 .. kMaxTriggers-1 (0..15):
   2.     if triggers[i].armed then
   3.         triggers[i].timer--
   4.         if triggers[i].timer <= 0 then
   5.             triggers[i].timer <- 0
   6.             triggers[i].armed <- false
   7.             FireTrigger(i)
```

`kMaxTriggers` = **16** (`GliderPRO/Sources/Triggers.c:12`). `trigType`
(`GliderPRO/Sources/Triggers.c:15-21`):

| Field | Type | Meaning |
| --- | --- | --- |
| `object` | `short` | index of the triggered object within its room |
| `room` | `short` | room number of the triggered object |
| `index` | `short` | index into `masterObjects` of the *trigger* |
| `timer` | `short` | frames remaining |
| `what` | `short` | object type code of the triggered object |
| `armed` | `Boolean` | slot in use |

`ArmTrigger` (`GliderPRO/Sources/Triggers.c:34-55`):

```
ArmTrigger(who):
   1. if who->stillOver then return                                  // :38-39
   2. where <- FindEmptyTriggerSlot()                                // :41
   3. if where != -1 then
   4.     whoLinked <- who->who
   5.     triggers[where].room   <- masterObjects[whoLinked].roomLink
   6.     triggers[where].object <- masterObjects[whoLinked].objectLink
   7.     triggers[where].index  <- whoLinked
   8.     triggers[where].timer  <- masterObjects[whoLinked].theObject.data.e.delay * 3   // :49
   9.     triggers[where].what   <- masterObjects[triggers[where].object].theObject.what
  10.     triggers[where].armed  <- true
  11. who->stillOver <- true                                         // :54
```

Note `timer = delay * 3` where `delay` is a `short` in `switchType`
(`GliderPRO/Headers/GliderStructs.h:48`) read straight from the house file. The `* 3`
factor is a hard-coded frame conversion.

`FindEmptyTriggerSlot` (`GliderPRO/Sources/Triggers.c:59-75`) returns the **lowest** free
index, or -1. **This is order-dependent state**: which slot a trigger lands in determines
its position in `HandleTriggers`' iteration, which determines the order two triggers
expiring on the same frame fire in. A port must use the same lowest-free-index policy.

`FireTrigger` (`GliderPRO/Sources/Triggers.c:100-194`) dispatches on the *triggered*
object's type. Complete dispatch table:

| Triggered type | Action | Line |
| --- | --- | --- |
| `kGreaseRt` 0x28, `kGreaseLf` 0x29 | `SetObjectState(room, object, kForceOn, triggeredIs)`, and if it returns true, `SpillGrease(dynaNum, hotNum)` | `:112-120` |
| `kLightSwitch` 0x41, `kMachineSwitch` 0x42, `kThermostat` 0x43, `kPowerSwitch` 0x44, `kKnifeSwitch` 0x45, `kInvisSwitch` 0x46 | `TriggerSwitch(dynaNum)` | `:122-129` |
| `kSoundTrigger` 0x49 | `PlayPrioritySound(kChordSound, kChordPriority)` | `:131-133` |
| `kToaster` 0x62 | `TriggerToast(dynaNum)` | `:135-137` |
| `kGuitar` 0x64 | `PlayPrioritySound(kChordSound, kChordPriority)` | `:139-141` |
| `kCoffee` 0x66 | `PlayPrioritySound(kCoffeeSound, kCoffeePriority)` | `:143-145` |
| `kOutlet` 0x67 | `TriggerOutlet(dynaNum)` | `:147-149` |
| `kBalloon` 0x71 | `TriggerBalloon(dynaNum)` | `:151-153` |
| `kCopterLf` 0x72, `kCopterRt` 0x73 | `TriggerCopter(dynaNum)` | `:155-158` |
| `kDartLf` 0x74, `kDartRt` 0x75 | `TriggerDart(dynaNum)` | `:160-163` |
| `kDrip` 0x77 | `TriggerDrip(dynaNum)` | `:165-167` |
| `kFish` 0x78 | `TriggerFish(dynaNum)` | `:169-171` |

The `else` arm (`GliderPRO/Sources/Triggers.c:174-193`, taken when
`masterObjects[triggerIs].localLink == -1`) handles only grease, reading the object type
directly out of the house handle under `HLock`/`HSetState`.

`ZeroTriggers` (`GliderPRO/Sources/Triggers.c:198-204`) sets `armed = false` on all 16 and
nothing else — the other fields are stale until reused. Called from `DrawLocale`
(`GliderPRO/Sources/RoomGraphics.c:55`), so **all pending triggers are cancelled by a room
transition.**

## 3.7 Phase 7 — `HandleBands`

`GliderPRO/Sources/RubberBands.c:208-252`. Constants at `:12-14`:

| Constant | Value |
| --- | --- |
| `kRubberBandVelocity` | 20 |
| `kBandFallCount` | 4 |
| `kKillBandMode` | -1 |

`kMaxRubberBands` = **2** (`GliderPRO/Headers/GliderDefines.h:261`) — only two bands can
be airborne at once.

```
HandleBands():
   1. if numBands == 0 then return                                  // :213-214
   2. for i = 0 .. numBands-1:                                       // :216
   3.     bands[i].mode++ ; if bands[i].mode > 2 then bands[i].mode <- 0   // :218-220
   4.     bands[i].count++                                           // :222
   5.     if bands[i].count >= kBandFallCount (4) then               // :223
   6.         bands[i].vVel++ ; bands[i].count <- 0                   // :225-226
   7.     dest <- bands[i].dest ; QOffsetRect(&dest, playOriginH, playOriginV)
   8.     AddRectToWorkRects(&dest)                                   // :229-231, old rect dirty
   9.     bands[i].dest.left/.right += bands[i].hVel                  // :233-234
  10.     bands[i].dest.top/.bottom += bands[i].vVel                  // :235-236
  11.     CheckBandCollision(i)                                       // :238
  12. // compaction pass, :241-251
  13. count <- 0
  14. do
  15.     while bands[count].mode == kKillBandMode:                   // :244  NOTE: while, not if
  16.         bands[count].mode <- 0 ; KillBand(count)                 // :246-247
  17.     count++
  18. while count < numBands
```

The compaction pass is subtle and must be copied literally. The inner construct at `:244`
is a **`while`, not an `if`**: because `KillBand` moves the *last* band into slot `count`
(swap-with-last), the freshly moved band may itself be marked `kKillBandMode`, so the same
slot is re-tested until it holds a live band. An `if` would leak a dead band for one frame.
The `while (count < numBands)` re-reads `numBands` each iteration, so the loop bound
shrinks as bands die.

`AddBand` (`GliderPRO/Sources/RubberBands.c:256-290`) — note it returns `Boolean`:

```
Boolean AddBand(thisGlider, h, v, direction):
   1. if numBands >= kMaxRubberBands (2) then return false           // :258-259
   2. bands[numBands].mode <- 0 ; count <- 0                         // :261-262
   3. bands[numBands].vVel <- (thisGlider->tipped) ? -2 : 0          // :263-266
   4. dest.left <- h-8 ; right <- h+8 ; top <- v-3 ; bottom <- v+3   // :267-270 (16x6, centred)
   5. if direction == kFaceLeft then                                 // :272  tests LEFT first
   6.     dest.left -= 32 ; dest.right -= 32                          // :274-275
   7.     bands[numBands].hVel <- -kRubberBandVelocity (-20)          // :276
   8. else
   9.     dest.left += 32 ; dest.right += 32                          // :280-281
  10.     bands[numBands].hVel <- +kRubberBandVelocity (+20)          // :282
  11. thisGlider->hVel -= (bands[numBands].hVel / 2)                 // :285  RECOIL
  12. numBands++                                                     // :286
  13. PlayPrioritySound(kFireBandSound, kFireBandPriority)           // :288
  14. return true                                                    // :289
```

Step 11 is the **recoil**: firing a band shoves the glider 10 px/frame the other way
(integer division of ±20 by 2). This is a real gameplay mechanic and directly couples input
to glider velocity. Note the rect is built by direct field assignment, not `QSetRect`, and
the direction branch tests `kFaceLeft`, taking the negative-velocity arm first.

`CheckBandCollision` (`GliderPRO/Sources/RubberBands.c:37-204`), the interesting parts:

```
CheckBandCollision(who):
   1. nothingCollided <- true                                                  // :42
   2. // wall rebound — note the second arm is `else if`, and both compare against
   3. // the CONSTANTS kLeftWallLimit/kRightWallLimit, not against leftThresh/rightThresh
   4. if (leftThresh == kLeftWallLimit) and (dest.left < kLeftWallLimit) then   // :44
   5.     if hVel < 0 then hVel <- -hVel                                        // :46-47 conditional!
   6.     dest.left <- kLeftWallLimit ; dest.right <- dest.left + 16             // :48-49
   7.     PlayPrioritySound(kBandReboundSound, kBandReboundPriority)             // :50
   8. else if (rightThresh == kRightWallLimit) and (dest.right > kRightWallLimit) then  // :53
   9.     if hVel > 0 then hVel <- -hVel                                        // :55-56
  10.     dest.right <- kRightWallLimit ; dest.left <- dest.right - 16           // :57-58
  11.     PlayPrioritySound(kBandReboundSound, kBandReboundPriority)             // :59
  12. // hotspot interaction
  13. for i = 0 .. nHotSpots-1:                                                 // :63
  14.     if !hotSpots[i].isOn then continue                                     // :65
  15.     action <- hotSpots[i].action                                           // :67
  16.     if action not in {kDissolveIt, kRewardIt, kSwitchIt, kTriggerIt, kBounceIt}
  17.         then continue                                                      // :68-70 filter FIRST
  18.     collided <- closed-interval overlap(bands[who].dest, hotSpots[i].bounds) // :72-81
  19.     if !collided then continue
  20.     nothingCollided <- false                                               // :85
  21.     if bandHitLast == i then continue    // :86 — LATCH GATES ALL FIVE ACTIONS
  22.     bandHitLast <- i                                                       // :88
  23.     if action is kDissolveIt or kBounceIt then                             // :89 SHARED branch
  24.         if hVel > 0 then
  25.             if (dest.right - hVel) < bounds.left then                       // :93-94
  26.                 hVel <- -hVel ; dest.right <- bounds.left ; dest.left <- right-16
  27.             else bands[who].mode <- kKillBandMode                           // :101
  28.         else
  29.             if (dest.left - hVel) > bounds.right then                       // :105-106
  30.                 hVel <- -hVel ; dest.left <- bounds.right ; dest.right <- left+16
  31.             else bands[who].mode <- kKillBandMode                           // :113
  32.         PlayPrioritySound(kBandReboundSound, kBandReboundPriority)          // :115
  33.         break                            // :116 — ABORTS THE HOTSPOT LOOP
  34.     else if action == kRewardIt then                                        // :118
  35.         whoLinked <- hotSpots[i].who
  36.         if linked object is kGreaseRt or kGreaseLf then                     // :121-122
  37.             if SetObjectState(thisRoomNumber, ..., 0, whoLinked) then
  38.                 SpillGrease(dynaNum, hotNum)                                // :126
  39.             hotSpots[i].isOn <- false                                       // :128
  40.     else if action == kSwitchIt  then HandleSwitches(&hotSpots[i])          // :133
  41.     else if action == kTriggerIt then ArmTrigger(&hotSpots[i])              // :137
  42. if nothingCollided then bandHitLast <- -1                                  // :145-146
  43. // glider momentum transfer — GATED ON hVel != 0
  44. if bands[who].hVel != 0 then                                              // :148
  45.     collided <- closed-interval overlap(bands[who].dest, theGlider.dest)   // :150-159
  46.     if collided then
  47.         if ((!twoPlayerGame) or (!onePlayerLeft) or (playerDead == kPlayer2)) then // :163
  48.             theGlider.hVel += (bands[who].hVel / 2)                        // :165
  49.             bands[who].hVel <- 0                                           // :166
  50.             PlayPrioritySound(kHitWallSound, kHitWallPriority)              // :167
  51.     if twoPlayerGame then                                                  // :171
  52.         collided <- closed-interval overlap(bands[who].dest, theGlider2.dest) // :173-182
  53.         if collided then
  54.             if ((!onePlayerLeft) or (playerDead == kPlayer1)) then          // :186
  55.                 theGlider2.hVel += (bands[who].hVel / 2)                    // :188
  56.                 bands[who].hVel <- 0 ; PlayPrioritySound(kHitWallSound, ...) // :189-190
  57. // kill conditions
  58. if (dest.left < kLeftWallLimit) or (dest.right > kRightWallLimit) then     // :195-196
  59.     bands[who].mode <- kKillBandMode                                       // :198
  60. else if dest.bottom > kFloorLimit then                                    // :200
  61.     bands[who].mode <- kKillBandMode                                       // :202
```

Four things here are easy to get wrong and all of them change gameplay:

1. **Neither the hotspot test nor the glider test calls `SectRect` or `SectGlider`.** Both
   are hand-rolled **closed-interval** overlap tests (`:72-81` and `:150-159`/`:173-182`),
   structurally identical to `SectGlider` (§3.5.2) but written out inline. Touching edges
   count as a hit.
2. **`bandHitLast` gates all five actions, not just `kSwitchIt`.** The `if (bandHitLast != i)`
   at `:86` encloses `:89-138` — dissolve, reward, switch and trigger are *all* suppressed
   when the band hit the same hotspot index on the previous frame.
3. **`kDissolveIt` and `kBounceIt` share one branch** (`:89-117`) that does *not* dissolve
   anything: it either rebounds the band off the hotspot's near edge (if the band had not
   already passed through it last frame) or kills the band, then `break`s out of the hotspot
   loop entirely — so **at most one dissolve/bounce hotspot is processed per frame**, and no
   later hotspot in the array is examined.
4. **`kRewardIt` only handles grease.** For any linked object other than `kGreaseRt`/
   `kGreaseLf` the branch does nothing at all — it does not award a prize.
5. **Momentum transfer is gated on `bands[who].hVel != 0`** (`:148`), so a band already
   stopped by one glider cannot push the other, and the player-2 block additionally requires
   `twoPlayerGame` (`:171`).
6. **The kill conditions are purely positional** (`:195-203`): off the left/right wall
   limits, or below `kFloorLimit` (312). There is **no** "hVel and vVel both zero" kill.

`bandHitLast` (`GliderPRO/Sources/RubberBands.c:26`) is a global anti-double-toggle latch:
without it, a band overlapping a light switch for two consecutive frames would flip it
twice. It is a **single** global, not per-band — so with two bands airborne, band 2 hitting
the same hotspot index band 1 just hit will be suppressed, and `nothingCollided` is
per-*call*, so the second band's miss resets the latch to -1 after the first band set it.
That is observable, order-dependent behaviour and must be reproduced exactly.

`KillBand` (`GliderPRO/Sources/RubberBands.c:294-303`):

```
KillBand(which):
   1. lastBand <- numBands - 1
   2. if which != lastBand then bands[which] <- bands[lastBand]     // SWAP WITH LAST
   3. numBands--
```

**Swap-with-last removal.** A band's array index is not stable across removals. Combined
with the `do/while` compaction loop in `HandleBands` (`:241-251`), the surviving band's
index after a removal depends on which one died. A Go port using `append(s[:i], s[i+1:]...)`
(order-preserving removal) **will produce different indices** and therefore a different
`bandHitLast` interaction and a different iteration order. Use swap-with-last.

`KillAllBands` (`GliderPRO/Sources/RubberBands.c:307-317`) sets `numBands = 0` and is
called from `DrawLocale` (`GliderPRO/Sources/RoomGraphics.c:53`) — **room transitions
destroy in-flight bands.**

## 3.8 Phase 8 — `HandleGlider` and the 24-mode state machine

`GliderPRO/Sources/Player.c:1335-1439`. A `switch (thisGlider->mode)` over all 24 glider
modes, followed unconditionally by three flag resets.

| Mode | Value | Handler | Handler line |
| --- | --- | --- | --- |
| `kGliderNormal` | 0 | `MoveGliderNormal` | `Player.c:151` |
| `kGliderFadingIn` | 1 | `FadeGliderIn` | `Player.c:231` |
| `kGliderFadingOut` | 2 | `FadeGliderOut` | `Player.c:291` |
| `kGliderGoingUp` | 3 | `MoveGliderUpStairs` | `Player.c:315` |
| `kGliderComingUp` | 4 | `FinishGliderUpStairs` | `Player.c:383` |
| `kGliderGoingDown` | 5 | `MoveGliderDownStairs` | `Player.c:431` |
| `kGliderComingDown` | 6 | `FinishGliderDownStairs` | `Player.c:512` |
| `kGliderFaceLeft` | 7 | `MoveGliderFaceLeft` | `Player.c:560` |
| `kGliderFaceRight` | 8 | `MoveGliderFaceRight` | `Player.c:577` |
| `kGliderBurning` | 9 | `MoveGliderBurning` | `Player.c:203` |
| `kGliderTransporting` | 10 | `TransportGliderOut` | `Player.c:594` |
| `kGliderDuctingDown` | 11 | `MoveGliderDownDuct` | `Player.c:657` |
| `kGliderDuctingUp` | 12 | `MoveGliderUpDuct` | `Player.c:754` |
| `kGliderDuctingIn` | 13 | `FinishGliderDuctingIn` | `Player.c:927` |
| `kGliderMailInLeft` | 14 | `MoveGliderInMailLeft` | `Player.c:960` |
| `kGliderMailOutLeft` | 15 | `FinishGliderMailingLeft` | `Player.c:851` |
| `kGliderMailInRight` | 16 | `MoveGliderInMailRight` | `Player.c:1051` |
| `kGliderMailOutRight` | 17 | `FinishGliderMailingRight` | `Player.c:889` |
| `kGliderGoingFoil` | 18 | `MoveGliderFoilGoing` | `Player.c:1170` |
| `kGliderLosingFoil` | 19 | `MoveGliderFoilLosing` | `Player.c:1230` |
| `kGliderShredding` | 20 | `MoveGliderShredding` | `Player.c:1260` |
| `kGliderInLimbo` | 21 | (no-op) | — |
| `kGliderIdle` | 22 | `HandleIdleGlider` | `Player.c:1323` |
| `kGliderTransportingIn` | 23 | `TransportGliderIn` | `Player.c:261` |

The unconditional tail (`GliderPRO/Sources/Player.c:1436-1438`):

```c
thisGlider->ignoreLeft = false;
thisGlider->ignoreRight = false;
thisGlider->ignoreGround = false;
```

These three flags are set by the `kIgnoreLeftWall` (16), `kIgnoreRightWall` (17) and
`kIgnoreGround` (23) hotspot actions during phase 5 and consumed in phase 8, then cleared.
They are **single-frame** flags with a strict producer→consumer ordering across phases.
A port that reorders phases 5 and 8 breaks them.

### 3.8.1 `MoveGlider` — the integrator, in full

`GliderPRO/Sources/Player.c:64-146`. Constants at `:13-17`:

| Constant | Value |
| --- | --- |
| `kGravity` | 3 |
| `kHImpulse` | 2 |
| `kVImpulse` | 2 |
| `kMaxHVel` | 16 |
| `kShredderCountdown` | -68 |

```
MoveGlider(thisGlider):
   1. // horizontal velocity ramp toward the desired velocity
   2. if thisGlider->hVel > thisGlider->hDesiredVel then
   3.     thisGlider->hVel -= kHImpulse (2)
   4.     if thisGlider->hVel < thisGlider->hDesiredVel then
   5.         thisGlider->hVel <- thisGlider->hDesiredVel
   6. else if thisGlider->hVel < thisGlider->hDesiredVel then
   7.     thisGlider->hVel += kHImpulse (2)
   8.     if thisGlider->hVel > thisGlider->hDesiredVel then
   9.         thisGlider->hVel <- thisGlider->hDesiredVel
  10. thisGlider->hDesiredVel <- 0                                  // Player.c:78
  11. // vertical velocity ramp
  12. if thisGlider->vVel > thisGlider->vDesiredVel then
  13.     thisGlider->vVel -= kVImpulse (2) ; clamp
  14. else if thisGlider->vVel < thisGlider->vDesiredVel then
  15.     thisGlider->vVel += kVImpulse (2) ; clamp
  16. thisGlider->vDesiredVel <- kGravity (3)                        // Player.c:92
  17. // horizontal integration with velocity clamp and dirty-rect union
  18. if thisGlider->hVel < 0 then
  19.     if thisGlider->hVel < -kMaxHVel (16) then thisGlider->hVel <- -kMaxHVel
  20.     thisGlider->wasHVel <- thisGlider->hVel
  21.     thisGlider->whole.right <- thisGlider->dest.right
  22.     thisGlider->dest.left  += thisGlider->hVel
  23.     thisGlider->dest.right += thisGlider->hVel
  24.     thisGlider->whole.left <- thisGlider->dest.left
  25.     (identical block for destShadow / wholeShadow)
  26. else
  27.     if thisGlider->hVel > kMaxHVel then thisGlider->hVel <- kMaxHVel
  28.     thisGlider->wasHVel <- thisGlider->hVel
  29.     thisGlider->whole.left <- thisGlider->dest.left
  30.     thisGlider->dest.left  += thisGlider->hVel
  31.     thisGlider->dest.right += thisGlider->hVel
  32.     thisGlider->whole.right <- thisGlider->dest.right
  33.     (identical block for destShadow / wholeShadow)
  34. // vertical integration (NO vertical velocity clamp)
  35. if thisGlider->vVel < 0 then
  36.     thisGlider->wasVVel <- thisGlider->vVel
  37.     thisGlider->whole.bottom <- thisGlider->dest.bottom
  38.     thisGlider->dest.top    += thisGlider->vVel
  39.     thisGlider->dest.bottom += thisGlider->vVel
  40.     thisGlider->whole.top <- thisGlider->dest.top
  41. else
  42.     thisGlider->wasVVel <- thisGlider->vVel
  43.     thisGlider->whole.top <- thisGlider->dest.top
  44.     thisGlider->dest.top    += thisGlider->vVel
  45.     thisGlider->dest.bottom += thisGlider->vVel
  46.     thisGlider->whole.bottom <- thisGlider->dest.bottom
```

Four things to notice:

* **`hDesiredVel` is reset to 0 and `vDesiredVel` to `kGravity` every frame** (lines 10
  and 16). Input in phase 4 sets `hDesiredVel` for *one* frame only; the ramp in lines
  2-9 then chases it. This is a "thrust while held" model implemented without any held-key
  state in the integrator.
* **Horizontal velocity is clamped to ±16 (`kMaxHVel`); vertical velocity is not
  clamped at all.** A long fall accumulates unbounded `vVel`. Nothing in `MoveGlider`
  limits it. The only bound is that `dest` leaves the room and an escape fires.
* `whole` is the **union of the pre- and post-move rect** — the dirty rectangle. It is
  built by the asymmetric assignments in lines 21/24 vs 29/32. This is *rendering* state
  computed inside the *physics* function; a Go port that separates sim from render must
  either keep computing it or recompute it in the renderer from `wasHVel`/`wasVVel`.
* `wasHVel`/`wasVVel` preserve the applied velocity for the mode handlers and the
  renderer (sprite tilt selection).

`MoveGliderNormal` (`GliderPRO/Sources/Player.c:151-200`) selects the sprite index before
calling `MoveGlider`:

| `facing` | condition | `gliderSrc[]` index |
| --- | --- | --- |
| `kFaceLeft` | `sliding` | 30 |
| `kFaceLeft` | `tipped` | 3 |
| `kFaceLeft` | otherwise | 2 |
| `kFaceRight` | `sliding` | 29 |
| `kFaceRight` | `tipped` | 1 |
| `kFaceRight` | otherwise | 0 |

`kNumGliderSrcRects` = 31 (`GliderPRO/Headers/GliderDefines.h:558`).

### 3.8.2 `OffsetGlider` — the room-transition teleport

`GliderPRO/Sources/Player.c:1443-1480`:

```
OffsetGlider(thisGlider, where):
   1. if (twoPlayerGame) and (onePlayerLeft) and (thisGlider->which == playerDead)
        then return                                                       // :1445-1446
   2. switch where:                                                        // :1448
   3.     kToRight:                                                        // :1450
   4.         dest.left/.right       += kRoomWide (512)                     // :1451-1452
   5.         destShadow.left/.right += kRoomWide                           // :1453-1454
   6.         whole <- dest ; wholeShadow <- destShadow                     // :1455-1456
   7.     kToLeft:  same with -= kRoomWide                                  // :1459-1466
   8.     kAbove:                                                          // :1468
   9.         dest.top/.bottom -= kTileHigh (322)                           // :1469-1470
  10.         whole <- dest                                                 // :1471
  11.     kBelow:   same with += kTileHigh                                  // :1474-1478
```

`kRoomWide` = 512 = `kNumTiles (8) * kTileWide (64)`
(`GliderPRO/Headers/GliderDefines.h:499`). `kTileHigh` = 322
(`GliderPRO/Headers/GliderDefines.h:498`), and `kVertLocalOffset` is also 322 (`:501`).

**The four cases are not symmetric, and this is easy to get wrong.** Only `.left`/`.right`
are adjusted horizontally and only `.top`/`.bottom` vertically — the function never touches
the other axis. More importantly, **the vertical cases (`kAbove`, `kBelow`) do not move
`destShadow` or `wholeShadow` at all**; the horizontal cases do. So after a vertical room
transition the glider's shadow rect still holds the *previous* room's horizontal-shadow
state until the shadow logic recomputes it. A port that implements this as a single rigid
"translate all four rects" helper will diverge on vertical transitions. Note also this
function moves four rects (`dest`, `destShadow`, `whole`, `wholeShadow`), not eight.

### 3.8.3 `OffAMortal` — the death path

`GliderPRO/Sources/Player.c:1484-1604`. This is the most intricate control flow in the
game and the natural death sync point.

```
OffAMortal(thisGlider):
   1. if gameOver then return                                      // :1486-1487
   2. if numShredded > 0 then RemoveShreds()                       // :1489-1490
   3. mortals--                                                    // :1492
   4. if mortals < 0 then
   5.     HideGlider(thisGlider)
   6.     if twoPlayerGame then
   7.         if mortals < -1 then
   8.             FlagGameOver() ; thisGlider->dontDraw <- true
   9.         else
  10.             FlagGliderInLimbo(thisGlider, false)                // :1505
  11.             thisGlider->dontDraw <- true                        // :1506
  12.             onePlayerLeft <- true                               // :1507
  13.             playerDead <- thisGlider->which                     // :1508
  14.     else                                                        // single player
  15.         FlagGameOver() ; thisGlider->dontDraw <- true              // :1513-1514
  16. else
  17.     QuickGlidersRefresh() ; HideGlider(thisGlider)                 // :1519-1520
  18. if mortals >= 0 then                                              // :1523
  19.     if thisGlider->mode == kGliderGoingFoil then DeckGliderInFoil(thisGlider)  // :1525
  20.     FlagGliderNormal(thisGlider)                                   // :1528
  21.     if playerSuicide then
  22.         FollowTheLeader()                                          // :1530
  23.     else
  24.         StartGliderFadingIn(thisGlider)                            // :1533
  25.         thisGlider->dest  <- thisGlider->enteredRect               // :1534
  26.         thisGlider->whole <- thisGlider->dest                      // :1535
  27.         thisGlider->destShadow.left  <- thisGlider->dest.left      // :1536
  28.         thisGlider->destShadow.right <- thisGlider->dest.right     // :1537
  29.         thisGlider->wholeShadow <- thisGlider->destShadow          // :1538
  30. else if (mortals == -1) and (onePlayerLeft) and (!gameOver) then   // :1541
  31.     survivor <- (playerDead == kPlayer1) ? &theGlider2 : &theGlider   // per case, :1548 etc.
  32.     switch otherPlayerEscaped:                                     // :1543
  33.         kPlayerEscapedUp (-4), kPlayerEscapingUpStairs (-8),
  34.           kPlayerEscapedUpStairs (-6)    -> MoveRoomToRoom(survivor, kAbove)   // :1545-1552
  35.         kPlayerEscapedDown (-5), kPlayerEscapingDownStairs (-9),
  36.           kPlayerEscapedDownStairs (-7)  -> MoveRoomToRoom(survivor, kBelow)   // :1554-1561
  37.         kPlayerEscapedLeft  (-3) -> MoveRoomToRoom(survivor, kToLeft)          // :1563
  38.         kPlayerEscapedRight (-2) -> MoveRoomToRoom(survivor, kToRight)         // :1570
  39.         kPlayerTransportedOut (-10) -> TransportRoomToRoom(survivor)           // :1577
  40.         kPlayerMailedOut    (-12) -> MoveMailToMail(survivor)                  // :1584
  41.         kPlayerDuckedOut    (-11) -> MoveDuctToDuct(survivor)                  // :1591
  42.         default -> nothing                                                     // :1598
  43.     otherPlayerEscaped <- kPlayerIsDeadForever (-69)               // :1602
```

Two details the earlier draft of this section got wrong:

- The foil restore at `:1525` is conditioned on **`thisGlider->mode == kGliderGoingFoil`**,
  not on `foilTotal > 0`. Only a glider that died *while wrapping itself in foil* gets the
  foil back on respawn; a glider that merely had foil in the room does not.
- The `kAbove` and `kBelow` cases are **three-label fallthrough groups**: the four stairs
  constants `kPlayerEscapedUpStairs` (-6), `kPlayerEscapedDownStairs` (-7),
  `kPlayerEscapingUpStairs` (-8) and `kPlayerEscapingDownStairs` (-9) share the plain
  up/down cases, so a stairs escape resolves to an ordinary vertical `MoveRoomToRoom`. A port
  that gives the stairs constants their own handler will send the survivor to the wrong room.

Line 39 is important: once one of two players is permanently out, the escape flag is
parked at -69 forever, which makes every subsequent `CheckEscape*Two`
`otherPlayerEscaped == kNoOneEscaped` test fail — but `CheckGliderInRoom` also stops
calling the `*Two` variants once `onePlayerLeft` is true: all four dispatches read
`else if ((twoPlayerGame) && (!onePlayerLeft))`, at `GliderPRO/Sources/Interactions.c:704`
(up), `:717` (down), `:733` (left) and `:746` (right), so the survivor plays with
single-player exit rules. That transition is a hard behavioural change that must be reproduced.

`mortals` starts at `kInitialGliders` = 2 (`#define kInitialGliders 2`,
`GliderPRO/Sources/Play.c:19`; assigned at `Play.c:341`), doubled to 4 for two-player games
(`GliderPRO/Sources/Play.c:342-343`).

## 3.9 Phase 9 — `RenderFrame` mutates the simulation

This is the biggest structural obstacle to a clean Go port. `RenderFrame` is not a pure
function of state; it advances several animation state machines and can spawn entities.

`GliderPRO/Sources/Render.c:639-671`, transcribed:

```
RenderFrame():
   1. if hasMirror then                                            // :641
   2.     DrawReflection(&theGlider, true)                          // :643
   3.     if twoPlayerGame then DrawReflection(&theGlider2, false)   // :644-645
   4. HandleGrease()                                               // :647  MUTATES hotSpots[]
   5. RenderPendulums()                                            // :648  MUTATES clockFrame
   6. if evenFrame then RenderFlames() else RenderStars()           // :649-652  MUTATES flame/star modes
   7. RenderDynamics()                                             // :653
   8. RenderFlyingPoints()                                         // :654  MUTATES flyingPoints[]
   9. RenderSparkles()                                             // :655  MUTATES sparkles[]
  10. RenderGlider(&theGlider, true)                                // :656
  11. if twoPlayerGame then RenderGlider(&theGlider2, false)         // :657-658
  12. RenderShreds()                                               // :659  CAN CALL AddSparkle
  13. RenderBands()                                                // :660
  14. while (TickCount() < nextFrame) { }                            // :662-664
  15. nextFrame <- TickCount() + kTicksPerFrame (2)                  // :665
  16. CopyRectsQD()                                                // :667
  17. numWork2Main <- 0                                            // :669
  18. numBack2Work <- 0                                            // :670
```

### 3.9.1 What each render call mutates

| Call | Simulation state it writes |
| --- | --- |
| `HandleGrease` (`Grease.c:43`) | `grease[i].frame`, `grease[i].mode`, `grease[i].start`, `grease[i].dest`, **and `hotSpots[grease[i].hotNum].action`/`.isOn`/`.bounds`** — i.e. it creates and grows collision geometry |
| `RenderPendulums` (`Render.c:260`) | global `clockFrame`; per-pendulum `mode`, `src.top`/`src.bottom` (±28), `toOrFro`. `playedTikTok` is a **local** (`Render.c:263`), re-initialised to `false` on every call (`:265`) — it is *not* persistent state |
| `RenderFlames` (`Render.c:193`) | `flames[i].mode`, `tikiFlames[i].mode`, `bbqCoals[i].mode` (each wrapping at its own count) |
| `RenderStars` (`Render.c:420`) | `theStars[i].mode` (advance +31 in source rect, wrap when `mode >= 6`) |
| `RenderDynamics` (`Dynamics3.c:112`) | nothing (pure draw) |
| `RenderFlyingPoints` (`Render.c:325`) | `flyingPoints[i].dest`, `.mode`, `.loops`, `numFlyingPts` |
| `RenderSparkles` (`Render.c:384`) | `sparkles[i].mode`, `numSparkles` |
| `RenderGlider` (`Render.c:452`) | glider `frame` for animated modes |
| `RenderShreds` (`Render.c:559`) | `shreds[i].frame`, `shreds[i].bounds`, **and calls `AddSparkle(&shreds[i].bounds)` at frame 20** plus plays `kFadeOutSound` |
| `RenderBands` (`Render.c:534`) | nothing (pure draw) |

**Verdict: render and simulation are not separable in the original.** A Go port must
either (a) keep a combined step that does both, or (b) split it into
`SimStepPart2()` (the mutating parts, in exactly this order) plus `Draw()` (the blitting).
Option (b) is strongly recommended and is mechanically achievable because in every case
the mutation and the blit are adjacent but independent.

### 3.9.2 `RenderPendulums` and `clockFrame`

`GliderPRO/Sources/Render.c:260-321`:

```
RenderPendulums():
   1. playedTikTok <- false            // LOCAL, :263 declaration, :265 init
   2. if numPendulums == 0 then return                            // :267-268
   3. clockFrame++                                                // :270
   4. if (clockFrame == 10) or (clockFrame == 15) then             // :271 — GATES EVERYTHING BELOW
   5.     if clockFrame >= 15 then clockFrame <- 0                 // :273-274
   6.     for i = 0 .. numPendulums-1:                            // :276
   7.         if !pendulums[i].active then continue                // :278
   8.         if pendulums[i].toOrFro then                         // :280
   9.             pendulums[i].mode++                              // :282
  10.             pendulums[i].src.top += 28 ; src.bottom += 28     // :283-284
  11.             if pendulums[i].mode >= 2 then                   // :286
  12.                 toOrFro <- !toOrFro                          // :288
  13.                 if !playedTikTok then PlayPrioritySound(kTikSound, kTikPriority)
  14.                                        playedTikTok <- true   // :289-293
  15.         else
  16.             pendulums[i].mode--                              // :298
  17.             pendulums[i].src.top -= 28 ; src.bottom -= 28     // :299-300
  18.             if pendulums[i].mode <= 0 then                   // :302
  19.                 toOrFro <- !toOrFro                          // :304
  20.                 if !playedTikTok then PlayPrioritySound(kTokSound, kTokPriority)
  21.                                        playedTikTok <- true   // :305-309
  22.         CopyBits(savedMaps[pendulums[i].who].map -> workSrcMap, src, dest)  // :313-315
  23.         AddRectToWorkRects(&pendulums[i].dest)                // :317
```

Three structural points the earlier draft of this section got wrong:

- The `clockFrame == 10 || clockFrame == 15` test at `:271` is a **gate around the whole
  remainder of the function**, not a standalone "tick boundary" observation. The reset at
  `:273-274` *and* the entire pendulum loop `:276-319` are inside it, so pendulums only step
  on those two `clockFrame` values — 2 steps per 15 frames, not one step per frame.
- Each pendulum is additionally gated on `pendulums[i].active` (`:278`); inactive pendulums
  are neither stepped nor drawn.
- The animation advances `src.top`/`src.bottom` by ±28 — a **vertical** walk down the sprite
  strip, not a horizontal one.

Note the early return at step 2: **`clockFrame` only advances when at least one pendulum
exists in the current room.** And `AddPendulum` sets `clockFrame = 10`
(`GliderPRO/Sources/DynamicMaps.c:578`) at room load, so the first render call takes it to
11 and the first *step* happens 4 frames later at 15. The starting phase is therefore
deterministic, but the `toOrFro` direction is `RandomInt(2)`-chosen
(`GliderPRO/Sources/DynamicMaps.c:594`).

The 15-frame cycle with steps at 10 and 15 gives the clock a 2:1 asymmetric tick-tock
(10 frames from the wrap to the first step, 5 from there to the second).
`kNumPendulums` = 3 (`GliderPRO/Headers/GliderDefines.h:450`) is the number of *sprite*
frames, distinct from `kMaxPendulums` = 8 (`:258`), the array cap.

### 3.9.3 Dirty-rect queues silently drop rectangles

`kMaxGarbageRects` = **48** (`GliderPRO/Sources/Render.c:20`). Three add functions:

| Function | Line | Clamp target | Overflow behaviour |
| --- | --- | --- | --- |
| `AddRectToWorkRects` | `Render.c:65` | `justRoomsRect` | silently returns if count `>= 47` |
| `AddRectToBackRects` | `Render.c:84` | `justRoomsRect` | silently returns if count `>= 47` |
| `AddRectToWorkRectsWhole` | `Render.c:103` | `workSrcRect` | silently returns if count `>= 47` |

The guard is `>= 47`, not `>= 48` — one slot is left unused. This is purely a visual
artefact (a dropped update leaves stale pixels), not a simulation divergence, **unless** a
port reads pixels back. Glider PRO does read pixels back in one place: `BackUpToSavedMap`
snapshots the offscreen map so flames/prizes can be erased later. That is a load-time
operation, not per-frame, so the dirty-rect overflow does not feed back into simulation.

### 3.9.4 `InitGarbageRects` — the per-room render reset

`GliderPRO/Sources/Render.c:675-691`:

```
InitGarbageRects():
   1. numWork2Main <- 0 ; numBack2Work <- 0
   2. numSparkles <- 0
   3. for i = 0 .. kMaxSparkles-1 (0..2): sparkles[i].mode <- -1
   4. numFlyingPts <- 0
   5. for i = 0 .. kMaxFlyingPts-1 (0..2): flyingPoints[i].mode <- -1
   6. nextFrame <- TickCount() + kTicksPerFrame                     // :690
```

`kMaxSparkles` = 3, `kNumSparkleModes` = 5, `kMaxFlyingPts` = 3,
`kMaxFlyingPointsLoop` = 24 (`GliderPRO/Headers/GliderDefines.h:251-254`).

`AddSparkle` (`GliderPRO/Sources/DynamicMaps.c:169-193`) finds the **first slot with
`mode == -1`** (`:184-185`) rather than appending — so with only 3 slots, sparkle indices are
reused in lowest-free order. The whole body is wrapped in `if (numSparkles < kMaxSparkles)`
(`:174`), so a fourth simultaneous sparkle is silently dropped.

**`AddSparkle` mutates its argument.** Lines `:176-179` add `playOriginH`/`playOriginV` to
`*theRect` in place before centring the sparkle sprite in it. Every caller that passes a
pointer to *persistent* state therefore has that state shifted by the play origin as a side
effect — most notably `RenderShreds`, which passes `&shreds[i].bounds`
(`GliderPRO/Sources/Render.c:603`). `AddFlyingPoint` has the identical in-place mutation at
`:206-209`. A Go port that takes the rect by value here will diverge.

There are **19** `AddSparkle` call sites, not nine:

| Site | Context |
| --- | --- |
| `GliderPRO/Sources/DynamicMaps.c:157` | `RestoreFromSavedMap(..., doSparkle=true)` — object erased from the saved map |
| `GliderPRO/Sources/Dynamics2.c:96` | balloon burst |
| `GliderPRO/Sources/Dynamics2.c:118` | balloon respawn |
| `GliderPRO/Sources/Dynamics2.c:125` | balloon pre-spawn (`kStartSparkle` = 4) |
| `GliderPRO/Sources/Dynamics2.c:198` | copter destroyed |
| `GliderPRO/Sources/Dynamics2.c:226` | copter respawn |
| `GliderPRO/Sources/Dynamics2.c:233` | copter pre-spawn |
| `GliderPRO/Sources/Dynamics2.c:304` | dart leaves the room / hits the floor |
| `GliderPRO/Sources/Dynamics2.c:342` | dart respawn |
| `GliderPRO/Sources/Dynamics2.c:349` | dart pre-spawn |
| `GliderPRO/Sources/Dynamics.c:307` | sparkle object fires |
| `GliderPRO/Sources/Interactions.c:838`, `:857`, `:879`, `:907`, `:940`, `:963`, `:1050` | prize pickups, switch flips, microwave kills (7 sites) |
| `GliderPRO/Sources/Render.c:603` | inside `RenderShreds` |

(Note `Dynamics2.c:304` and `Dynamics.c:307` are two *different files*; an earlier draft of
this document conflated them into a single "`Dynamics.c:304`-region" entry and dropped
`DynamicMaps.c:157`.)

`AddFlyingPoint` (`GliderPRO/Sources/DynamicMaps.c:199-254`) — the floating score numbers.
Same lowest-free-slot allocation (`:214-215`) under `if (numFlyingPts < kMaxFlyingPts)`
(`:204`), and it finishes with `mode <- start` (`:249`). Its start/stop animation table, by
point value:

| `points` | `start` | `stop` |
| --- | --- | --- |
| 100 | 12 | 14 |
| 250 | 9 | 11 |
| 300 | 6 | 8 |
| 500 | 3 | 5 |
| (default) | 0 | 2 |

## 3.10 Phase 10 — `HandleDynamicScoreboard`

`GliderPRO/Sources/Scoreboard.c:72-132`. Function-local `#define`s at `:74-77` (all
commented "25%"):

| Constant | Value | Meaning |
| --- | --- | --- |
| `kFoilLow` | 2 | foil-gauge low-water mark |
| `kBatteryLow` | 17 | battery-gauge low-water mark |
| `kHeliumLow` | -38 | helium-gauge low-water mark (negative = helium is stored as negative battery) |
| `kBandsLow` | 2 | bands-gauge low-water mark |

`kScoreRollAmount` = **13** (`GliderPRO/Sources/Scoreboard.c:21`) — the score odometer
climbs 13 points per frame, so a 5000-point star takes 385 frames (~12.8 s) to finish
rolling.

```
HandleDynamicScoreboard():
   1. if theScore > displayedScore then                            // :80
   2.     if doRollScore then                                      // :82
   3.         displayedScore += kScoreRollAmount (13)               // :84
   4.         if displayedScore > theScore then displayedScore <- theScore   // :85-86
   5.     else
   6.         displayedScore <- theScore                            // :89
   7.     PlayPrioritySound(kScoreTikSound (17), kScoreTikPriority (101))    // :91
   8.     QuickScoreRefresh()                                      // :92
   9. whosTurn <- gameFrame & 0x00000007      // long, :78 decl; :95 assignment
  10. switch whosTurn:                                             // :96
  11.     0 -> if (foilTotal > 0) and (foilTotal < kFoilLow) then QuickFoilRefresh(false)      // :98-101
  12.     1 -> if (batteryTotal > 0) and (batteryTotal < kBatteryLow) then QuickBatteryRefresh(true)
  13.          else if (batteryTotal < 0) and (batteryTotal > kHeliumLow) then QuickBatteryRefresh(true)  // :103-108
  14.     2 -> if (bandsTotal > 0) and (bandsTotal < kBandsLow) then QuickBandsRefresh(false)  // :110-113
  15.     4 -> same as case 1 but QuickBatteryRefresh(false)          // :115-120
  16.     5 -> same as case 0 but QuickFoilRefresh(true)              // :122-125
  17.     7 -> same as case 2 but QuickBandsRefresh(true)             // :127-130
  18.     (3 and 6 have no case labels at all)
```

Each case is **guarded by its low-water mark** — the gauge is only touched when the supply
is non-zero *and* below 25%. The paired show/hide cases (0/5 foil, 1/4 battery, 2/7 bands)
are what makes a nearly-empty gauge blink; a full gauge is never redrawn here.

The `gameFrame & 7` slicing is a load-spreading trick: only one gauge is considered per
frame, on an 8-frame cycle. It reads `gameFrame` but writes nothing simulation-relevant.
**Helium is represented as negative `batteryTotal`** — that is why `kHeliumLow` is -38 and
why `kHeliumSupply` (150) is added as a negative somewhere in the pickup path.

`kRoomVisitScore` = 100, `kRedClockPoints` = 100, `kBlueClockPoints` = 300,
`kYellowClockPoints` = 500, `kCuckooClockPoints` = 1000, `kStarPoints` = 5000
(`GliderPRO/Headers/GliderDefines.h:536-541`).

## 3.11 Phase 11 — the game-over countdown

`GliderPRO/Sources/Play.c:499-553`:

```
if gameOver then
    countDown--                                                    // :501
    if countDown <= 0 then
        GetGWorld(&wasCPort, &wasWorld)                            // :507
        HideGlider(&theGlider)                                     // :509
        RefreshScoreboard(kNormalTitleMode)                        // :510
        #if BUILD_ARCADE_VERSION
            (black out the scoreboard region; redraw kScoreboardPictID = 1997)   // :512-544
        #endif
        if mortals < 0 then DoDiedGameOver() else DoGameOver()     // :546-549
        SetGWorld(wasCPort, wasWorld)                              // :551
```

`countDown` is set by `FlagGameOver` (`GliderPRO/Sources/GameOver.c:237-242`):

```c
void FlagGameOver (void)
{
	gameOver = true;                              // GameOver.c:239
	countDown = kNumCountDownFrames;              // GameOver.c:240
	SetMusicalMode(kPlayWholeScoreMode);          // GameOver.c:241
}
```

with `kNumCountDownFrames` = **16** (`GliderPRO/Sources/GameOver.c:17`) and
`kPlayWholeScoreMode` = -1 (`GliderPRO/Headers/GliderDefines.h:50`). So the death→game-over
transition is a fixed **16-frame** delay. During it, `HandleTelephone` (`Play.c:445`),
`HandleDynamics` (`:449`/`:475`), `HandleTriggers` (`:456`/`:484`), `HandleBands`
(`:457`/`:485`), `RenderFrame` and `HandleDynamicScoreboard` all keep running; only the
three `if (!gameOver)`-guarded groups stop — `GetInput`/`GetDemoInput`,
`HandleInteraction` and `HandleGlider` (`:450-455`, `:458-462` two-player;
`:476-483`, `:486-487` single-player).

`FlagGameOver` is called from exactly three sites:
`GliderPRO/Sources/Interactions.c:944` (the win condition — inside the `kStar` hotspot case,
fired when `numStarsRemaining <= 0` after `numStarsRemaining--` at `:942`, i.e. **all stars
collected**, not "reaching a goal room"),
`GliderPRO/Sources/Player.c:1500` (two-player, `mortals < -1`),
`GliderPRO/Sources/Player.c:1513` (single-player, `mortals < 0`).


---

# Part 4 — Complete simulation state inventory

For lockstep or for progress-packet checksumming, you need to know exactly what "the
simulation state" is. Glider PRO keeps it in **globals**, not in a state struct. This part
enumerates them.

## 4.1 Global scalar state

| Global | Type | Declared | Reset where | Role |
| --- | --- | --- | --- | --- |
| `gameFrame` | `long` | extern `Play.c:62`-region | `= 0L` `Play.c:112` | frame index |
| `evenFrame` | `Boolean` | `Play.c:53` | never reset in `NewGame` | frame parity gate |
| `theKeys` | `KeyMap` (4 × `UInt32`) | `Input.c:31` | — | current keyboard snapshot |
| `numDynamics` | `short` | `Dynamics3.c:19` | `= 0` `Dynamics3.c:179` | live dynamic-object count |
| `numBands` | `short` | `RubberBands.c:26`-region | `= 0` `Play.c:113`, `KillAllBands` | live rubber-band count |
| `bandHitLast` | `short` | `RubberBands.c:26`-region | `= -1` in `CheckBandCollision` | last hotspot a band toggled |
| `numSparkles` | `short` | `DynamicMaps.c:31` | `= 0` `Render.c:682` (then `sparkles[i].mode = -1` `:683-684`) | live sparkle count |
| `numFlyingPts` | `short` | `DynamicMaps.c:31` | `= 0` `Render.c:686` (then `flyingPoints[i].mode = -1` `:687-688`) | live flying-score count |
| `numFlames` | `short` | `DynamicMaps.c:32` | `ZeroFlamesAndTheLike` | candle count |
| `numTikiFlames` | `short` | `DynamicMaps.c:32` | ditto | tiki-torch count |
| `numCoals` | `short` | `DynamicMaps.c:32` | ditto | BBQ count |
| `numPendulums` | `short` | `DynamicMaps.c:33` | ditto | clock count |
| `numStars` | `short` | `DynamicMaps.c:33` | ditto | star count |
| `clockFrame` | `short` | `DynamicMaps.c:33` | `= 10` in `AddPendulum` | pendulum animation phase |
| `numShredded` | `short` | `DynamicMaps.c:33` | `RemoveShreds` | shredded-glider effect count |
| `numSavedMaps` | `short` | `DynamicMaps.c:32` | `= 0` in `NilSavedMaps` | offscreen backup count |
| `numChimes` | `short` | `DynamicMaps.c:31` | room load | wind-chime object count in room |
| `phoneBitSet` | `Boolean` | (Play, `Play.c:54`) | **house load**, `HouseIO.c:417` (bit 1 of `houseType.flags`) | whole-house phone suppression flag — **not** per room, and inverted: the phone rings only while `!phoneBitSet` (§2.6) |
| `numGrease` | `short` | `Grease.c` | room load | grease-spill count |
| `nHotSpots` | `short` | (Objects) | room load | active hotspot count |
| `numMasterObjects` | `short` | (Objects) | `ListAllLocalObjects` | flattened 9-room object list length |
| `numLights` | `short` | (Room) | per-neighbour in `DrawLocale` | lights in the room being drawn |
| `numNeighbors` | `short` | (Main/prefs) | `ReadInPrefs` | **user preference**, 1/3/9 |
| `localNumbers[9]` | `short[9]` | (Room) | `DrawLocale:67-70` | the nine neighbour room numbers |
| `isStructure[9]` | `Boolean[9]` | (Room) | `DrawLocale:67-70` | per-neighbour structure flag |
| `thisRoomNumber` | `short` | `Room.c:29` | `ForceThisRoom` | current room |
| `previousRoom` | `short` | `Room.c:29` | `ForceThisRoom` | room we came from |
| `numberRooms` | `short` | `Room.c:29` | house load | rooms in the house |
| `leftThresh` | `short` | `Room.c:30` | `DetermineRoomOpenings` | left escape line |
| `rightThresh` | `short` | `Room.c:30` | `DetermineRoomOpenings` | right escape line |
| `lastBackground` | `short` | `Room.c:31` | room load | background PICT ID cache |
| `foilTotal` | `short` | (Play) | `InitGlider` | foil hits remaining |
| `batteryTotal` | `short` | (Play) | `InitGlider` | battery (positive) / helium (negative) |
| `bandsTotal` | `short` | (Play) | `InitGlider` | rubber bands in inventory |
| `mortals` | `short` | (Play) | `= kInitialGliders` `Play.c:341` | lives remaining |
| `theScore` | `long` | (Play) | `= 0L` `Play.c:340` | score |
| `displayedScore` | `long` | (Scoreboard) | | animated score readout |
| `numStarsRemaining` | `short` | (Play) | house load | win condition counter |
| `otherPlayerEscaped` | `short` | (Play) | `= kNoOneEscaped` `Play.c:116` | 2P rendezvous state |
| `onePlayerLeft` | `Boolean` | (Play) | `= false` `Play.c:117` | one of two players is out |
| `playerDead` | `Boolean` | (Play) | `OffAMortal:1508` | which player is out |
| `playerSuicide` | `Boolean` | (Play) | `= false` `Play.c:118` | death was self-inflicted (`ForceKillGlider`) |
| `firstPlayer` | `Boolean` | `Transit.c:18` | `FlagGliderInLimbo` | who reached the exit first |
| `saidFollow` | `short` | `Modes.c:13` | `= 0` `Play.c:115` | "follow me" voice-sample counter (max 3) |
| `twoPlayerGame` | `Boolean` | (Play) | menu | mode flag |
| `gameOver` | `Boolean` | (Play) | `FlagGameOver` | countdown active |
| `countDown` | `short` | `GameOver.c:45` | `= 16` `GameOver.c:240` | frames until the game-over screen |
| `playing` | `Boolean` | (Play) | `= true` `Play.c:213` | main-loop condition |
| `demoGoing` | `Boolean` | (Play) | `DoDemoGame` | attract mode active |
| `demoIndex` | `short` | `Input.c:33` | `= 0` `Play.c:114` | replay cursor |
| `batteryFrame` | `short` | `Input.c:33` | | helium animation 0..3 |
| `takingTheStairs` | `Boolean` | `Transit.c:18` | `= false` `DrawLocale:127` | stair-transition flag |
| `linkedToWhat` | `short` | `Transit.c:17` | `WhatAreWeLinkedTo` | transport link classification |
| `nextFrame` | `long` | (Render) | `InitGarbageRects:690` | pacing deadline (**not sim state**) |
| `switchedOut` | `Boolean` | `Events.c:31` | `ToolBoxInit:66` | app is suspended |
| `shadowVisible` | `Boolean` | (Room) | `DrawLocale:126` | draw the glider's shadow |
| `hasMirror` | `Boolean` | (Render) | room load | is there a mirror in this room |
| `tvInRoom` | `Boolean` | (RoomGraphics) | `DrawLocale:59` | QuickTime TV present |
| `tvWithMovieNumber` | `short` | (RoomGraphics) | `= -1` `DrawLocale:60` | which movie |
| `numTempManholes` | `short` | (RoomGraphics) | `= 0` `DrawLocale:56` | manhole rect count |

## 4.2 Global array state

| Array | Element type | Cap | Constant | Cap value |
| --- | --- | --- | --- | --- |
| `dinahs` | `dynaType` (36 B) | `kMaxDynamicObs` | `GliderDefines.h:265` | 18 |
| `bands` | `bandType` | `kMaxRubberBands` | `GliderDefines.h:261` | 2 |
| `triggers` | `trigType` | `kMaxTriggers` | `Triggers.c:12` | 16 |
| `hotSpots` | `hotObject` | `kMaxHotSpots` | `GliderDefines.h:259` | 56 |
| `sparkles` | `sparkleType` | `kMaxSparkles` | `GliderDefines.h:251` | 3 |
| `flyingPoints` | `flyingPtType` | `kMaxFlyingPts` | `GliderDefines.h:253` | 3 |
| `flames` | `flameType` | `kMaxCandles` | `GliderDefines.h:255` | 20 |
| `tikiFlames` | `flameType` | `kMaxTikis` | `GliderDefines.h:256` | 8 |
| `bbqCoals` | `flameType` | `kMaxCoals` | `GliderDefines.h:257` | 8 |
| `pendulums` | `pendulumType` | `kMaxPendulums` | `GliderDefines.h:258` | 8 |
| `theStars` | `starType` | `kMaxStars` | `GliderDefines.h:263` | 4 |
| `shreds` | `shredType` | `kMaxShredded` | `GliderDefines.h:264` | 4 |
| `grease` | `greaseType` | `kMaxGrease` | `GliderDefines.h:262` | 16 |
| `savedMaps` | `savedType` | `kMaxSavedMaps` | `GliderDefines.h:260` | 24 |
| `masterObjects` | `objDataType` | `kMaxMasterObjects` | `GliderDefines.h:266` | 216 (= 24 × 9) |
| `demoData` | `demoType` (6 B) | `kDemoLength` bytes | `GliderDefines.h:625` | 6702 B = 1117 records |

The two gliders are the globals `theGlider` and `theGlider2`, both of type `gliderType`.

## 4.3 Struct layouts, empirically verified

The layouts below were verified by transcribing `GliderPRO/Headers/GliderStructs.h` into a
standalone C file, compiling it with `gcc` under `#pragma pack(2)` (the GCC equivalent of
the original's `#pragma options align=mac68k`, `GliderPRO/Headers/Externs.h:230`), with
`long` mapped to a 32-bit type to match 68k/PPC Mac, and printing `sizeof` and
`offsetof` for every field. **Every observed size matched the author's own comment in the
header**, which is a strong cross-check that both the transcription and the original
comments are right.

The `long` mapping matters. A first run using native 64-bit `long` produced *wrong*
answers, which is worth recording as a trap:

| Struct | Observed with 64-bit `long` (WRONG) | Observed with 32-bit `long` (CORRECT) | Author's comment |
| --- | --- | --- | --- |
| `scoresType` | 372 | **292** | 292 (`GliderStructs.h:114`) |
| `gameType` | 56 | **40** | 40 (`GliderStructs.h:134`) |
| `demoType` | 10 | **6** | (implied) |
| `houseType` header | 970 | **866** | 866 (`GliderStructs.h:198`) |

### 4.3.1 `objectType` — 12 bytes

`GliderPRO/Headers/GliderStructs.h:90-105`. Observed `sizeof` = **12**.

| Offset | Field | Type | Size |
| --- | --- | --- | --- |
| 0 | `what` | `short` | 2 |
| 2 | `data` | `union {a..i}` | 10 |

The union has nine members, all exactly 10 bytes:

| Tag | Type | Fields (offset within the union) |
| --- | --- | --- |
| `a` | `blowerType` (`:11-19`) | `Point topLeft` @0 (4), `short distance` @4 (2), `Boolean initial` @6 (1), `Boolean state` @7 (1), `Byte vector` @8 (1), `Byte tall` @9 (1) |
| `b` | `furnitureType` (`:21-25`) | `Rect bounds` @0 (8), `short pict` @8 (2) |
| `c` | `bonusType` (`:27-34`) | `Point topLeft` @0, `short length` @4 (grease spill), `short points` @6 (invis bonus), `Boolean state` @8, `Boolean initial` @9 |
| `d` | `transportType` (`:36-43`) | `Point topLeft` @0, `short tall` @4 (invis transport), `short where` @6, `Byte who` @8, `Byte wide` @9 |
| `e` | `switchType` (`:45-52`) | `Point topLeft` @0, `short delay` @4, `short where` @6, `Byte who` @8, `Byte type` @9 |
| `f` | `lightType` (`:54-62`) | `Point topLeft` @0, `short length` @4, `Byte byte0` @6, `Byte byte1` @7, `Boolean initial` @8, `Boolean state` @9 |
| `g` | `applianceType` (`:64-72`) | `Point topLeft` @0, `short height` @4 (toaster height / PICT ID), `Byte byte0` @6, `Byte delay` @7, `Boolean initial` @8, `Boolean state` @9 |
| `h` | `enemyType` (`:74-82`) | `Point topLeft` @0, `short length` @4, `Byte delay` @6, `Byte byte0` @7, `Boolean initial` @8, `Boolean state` @9 |
| `i` | `clutterType` (`:84-88`) | `Rect bounds` @0 (8), `short pict` @8 (2) |

Note the `delay`/`byte0` **swap between `g` (appliance) and `h` (enemy)**: appliance is
`byte0` then `delay`, enemy is `delay` then `byte0`. A port that models this union as a
single flat struct will silently mis-read one of the two.

`blowerType.vector` is a bitfield with the author's own diagram at
`GliderPRO/Headers/GliderStructs.h:16-17`:

```
              F. lf. dn. rt. up
| x | x | x | x | 8 | 4 | 2 | 1 |
```

so bit 0 (0x01) = up, bit 1 (0x02) = right, bit 2 (0x04) = down, bit 3 (0x08) = left, and
the high nibble is unused.

Mac `Point` is `struct { short v; short h; }` — **vertical first**. Mac `Rect` is
`struct { short top, left, bottom, right; }` — **top-left-bottom-right**, not
x/y/w/h. Both are big-endian on disk. Getting the field order wrong is the single most
common house-file parsing bug.

### 4.3.2 `roomType` — 348 bytes

`GliderPRO/Headers/GliderStructs.h:166-180`. Observed `sizeof` = **348**, and observed
offsets:

| Offset | Field | Type | Size |
| --- | --- | --- | --- |
| 0 | `name` | `Str27` | 28 (1 length byte + 27) |
| 28 | `bounds` | `short` | 2 (opening bitmask; see §5.4) |
| 30 | `leftStart` | `Byte` | 1 |
| 31 | `rightStart` | `Byte` | 1 |
| 32 | `unusedByte` | `Byte` | 1 |
| 33 | `visited` | `Boolean` | 1 |
| 34 | `background` | `short` | 2 (PICT ID, `kBaseBackgroundID` 2000 … or `kUserBackground` 3000+) |
| 36 | `tiles[8]` | `short[8]` | 16 |
| 52 | `floor` | `short` | 2 |
| 54 | `suite` | `short` | 2 |
| 56 | `openings` | `short` | 2 |
| 58 | `numObjects` | `short` | 2 |
| 60 | `objects[24]` | `objectType[24]` | 288 |

`kNumTiles` = 8, `kTileWide` = 64, so a room is 8 × 64 = 512 px wide
(`GliderPRO/Headers/GliderDefines.h:496-499`). `kMaxRoomObs` = 24
(`GliderPRO/Headers/GliderDefines.h:250`).

### 4.3.3 `houseType` — 866-byte header + 348 × `nRooms`

`GliderPRO/Headers/GliderStructs.h:182-198`. Observed header `sizeof` = **866**, with
observed offsets:

| Offset | Field | Type | Size |
| --- | --- | --- | --- |
| 0 | `version` | `short` | 2 — always `kHouseVersion` 0x0200 (`GliderDefines.h:517`). `kNewHouseVersion` 0x0300 (`:518`) is *rejected* by the loader: `HouseIO.c:395` bails out on `version >= kNewHouseVersion` |
| 2 | `unusedShort` | `short` | 2 |
| 4 | `timeStamp` | `long` | 4 |
| 8 | `flags` | `long` | 4 (**bit 0 = `wardBit`**, per the author's comment at `:187`) |
| 12 | `initial` | `Point` | 4 (`v`, `h`) |
| 16 | `banner` | `Str255` | 256 |
| 272 | `trailer` | `Str255` | 256 |
| 528 | `highScores` | `scoresType` | 292 |
| 820 | `savedGame` | `gameType` | 40 |
| 860 | `hasGame` | `Boolean` | 1 |
| 861 | `unusedBoolean` | `Boolean` | 1 |
| 862 | `firstRoom` | `short` | 2 |
| 864 | `nRooms` | `short` | 2 |
| 866 | `rooms[]` | `roomType[]` | 348 × `nRooms` |

### 4.3.4 `gameType` — 40 bytes — the canonical progress record

`GliderPRO/Headers/GliderStructs.h:116-134`. Observed `sizeof` = **40**, offsets:

| Offset | Field | Type | Size | Filled from (`SaveGame`) |
| --- | --- | --- | --- | --- |
| 0 | `version` | `short` | 2 | `kSavedGameVersion` (`SavedGames.c:311`) |
| 2 | `wasStarsLeft` | `short` | 2 | `numStarsRemaining` (`:312`) |
| 4 | `timeStamp` | `long` | 4 | `GetDateTime(&stamp)` (`:320-321`) |
| 8 | `where` | `Point` | 4 | `.h = theGlider.dest.left`, `.v = theGlider.dest.top` (`:322-323`) |
| 12 | `score` | `long` | 4 | `theScore` (`:324`) |
| 16 | `unusedLong` | `long` | 4 | `0L` (`:325`) |
| 20 | `unusedLong2` | `long` | 4 | `0L` (`:326`) |
| 24 | `energy` | `short` | 2 | `batteryTotal` (`:327`) |
| 26 | `bands` | `short` | 2 | `bandsTotal` (`:328`) |
| 28 | `roomNumber` | `short` | 2 | `thisRoomNumber` (`:329`) |
| 30 | `gliderState` | `short` | 2 | `theGlider.mode` (`:330`) |
| 32 | `numGliders` | `short` | 2 | `mortals` (`:331`) |
| 34 | `foil` | `short` | 2 | `foilTotal` (`:332`) |
| 36 | `unusedShort` | `short` | 2 | `0` (`:333`) |
| 38 | `facing` | `Boolean` | 1 | `theGlider.facing` (`:334`) |
| 39 | `showFoil` | `Boolean` | 1 | `theGlider.showFoil`… see note (`:335`) |

Note: `gliderType` has no `showFoil` member in
`GliderPRO/Headers/GliderStructs.h:200-216`; `showFoil` is a separate global. `SaveGame`
(`GliderPRO/Sources/SavedGames.c:335`) reads that global.

`SaveGame` refuses to run in two-player mode:

```c
if (twoPlayerGame)                    // SavedGames.c:309
	return;                            // SavedGames.c:310
```

so the original has **no** two-player save/resume. That is exactly the gap a networked
progress packet has to fill.

### 4.3.5 `savedRoom` — 292 bytes, and `game2Type` — 110-byte header

`savedRoom` (`GliderPRO/Headers/GliderStructs.h:136-142`), observed `sizeof` = **292**:

| Offset | Field | Type | Size |
| --- | --- | --- | --- |
| 0 | `unusedShort` | `short` | 2 |
| 2 | `unusedByte` | `Byte` | 1 |
| 3 | `visited` | `Boolean` | 1 |
| 4 | `objects[24]` | `objectType[24]` | 288 |

`game2Type` (`GliderPRO/Headers/GliderStructs.h:144-164`) is the *external* save-file
format: a fixed header followed by a flexible `savedRoom savedData[]` array. Re-derived
offsets under `#pragma options align=mac68k` (≡ `pack(2)`), with `FSSpec` = 70 bytes:

| Offset | Field | Type | Size |
| --- | --- | --- | --- |
| 0 | `house` | `FSSpec` | 70 |
| 70 | `version` | `short` | 2 |
| 72 | `wasStarsLeft` | `short` | 2 |
| 74 | `timeStamp` | `long` | 4 |
| 78 | `where` | `Point` | 4 |
| 82 | `score` | `long` | 4 |
| 86 | `unusedLong` | `long` | 4 |
| 90 | `unusedLong2` | `long` | 4 |
| 94 | `energy` | `short` | 2 |
| 96 | `bands` | `short` | 2 |
| 98 | `roomNumber` | `short` | 2 |
| 100 | `gliderState` | `short` | 2 |
| 102 | `numGliders` | `short` | 2 |
| 104 | `foil` | `short` | 2 |
| 106 | `nRooms` | `short` | 2 |
| 108 | `facing` | `Boolean` | 1 |
| 109 | `showFoil` | `Boolean` | 1 |
| 110 | `savedData[]` | `savedRoom[]` | flexible |

**The header is 110 bytes, not 114.** The source comment at `:164` says "total = 114"
because it counts the flexible array member `savedData[]` as 4 bytes (`:163`); a compiled
`offsetof(game2Type, savedData)` is 110, and `sizeof(game2Type)` is likewise 110. A file of
*n* rooms is therefore `110 + 292*n` bytes. Getting this wrong by 4 would misalign every
saved room.

This is the format a "full snapshot" progress packet should be modelled on if you want to
resume mid-house rather than restart. Note `house` is an `FSSpec` — a Mac volume-reference /
directory-ID / Pascal-name triple that has no meaning off a Mac filesystem, so a port must
substitute its own house identifier there and cannot use the layout byte-for-byte.

### 4.3.6 `gliderType` — the per-player state

`GliderPRO/Headers/GliderStructs.h:200-216`, in declaration order:

| Field | Type | Purpose |
| --- | --- | --- |
| `src` | `Rect` | source rect in the glider sprite sheet |
| `mask` | `Rect` | source rect in the mask sheet |
| `dest` | `Rect` | **position** — the authoritative sim position |
| `whole` | `Rect` | dirty-rect union (render) |
| `destShadow` | `Rect` | shadow position |
| `wholeShadow` | `Rect` | shadow dirty rect (render) |
| `clip` | `Rect` | clipping rect (mailbox/duct animations) |
| `enteredRect` | `Rect` | where to respawn after a death |
| `leftKey` | `long` | key binding (KeyMap offset) |
| `rightKey` | `long` | key binding |
| `battKey` | `long` | key binding |
| `bandKey` | `long` | key binding |
| `hVel` | `short` | horizontal velocity (**or the idle countdown in `kGliderIdle`**) |
| `vVel` | `short` | vertical velocity |
| `wasHVel` | `short` | last applied `hVel` |
| `wasVVel` | `short` | last applied `vVel` |
| `vDesiredVel` | `short` | target vertical velocity (reset to `kGravity` each frame) |
| `hDesiredVel` | `short` | target horizontal velocity (reset to 0 each frame) |
| `mode` | `short` | one of the 24 `kGlider*` modes |
| `frame` | `short` | animation frame |
| `wasMode` | `short` | mode to restore from limbo/idle |
| `facing` | `Boolean` | `kFaceRight` (TRUE) / `kFaceLeft` (FALSE) |
| `tipped` | `Boolean` | glider is nose-down |
| `sliding` | `Boolean` | on grease |
| `ignoreLeft` | `Boolean` | single-frame flag from `kIgnoreLeftWall` |
| `ignoreRight` | `Boolean` | single-frame flag from `kIgnoreRightWall` |
| `fireHeld` | `Boolean` | band key was held last frame |
| `which` | `Boolean` | `kPlayer1` (TRUE) / `kPlayer2` (FALSE) |
| `heldLeft` | `Boolean` | left key held this frame |
| `heldRight` | `Boolean` | right key held this frame |
| `dontDraw` | `Boolean` | suppress rendering (limbo / dead) |
| `ignoreGround` | `Boolean` | single-frame flag from `kIgnoreGround` |

**The minimal per-player sim state is: `dest`, `destShadow`, `enteredRect`, `hVel`, `vVel`,
`wasHVel`, `wasVVel`, `vDesiredVel`, `hDesiredVel`, `mode`, `frame`, `wasMode`, `facing`,
`tipped`, `sliding`, `fireHeld`, `which`, `heldLeft`, `heldRight`, `dontDraw`, plus the
three single-frame `ignore*` flags.** `src`, `mask`, `whole`, `wholeShadow` and `clip` are
render-derived. The four `*Key` longs are configuration.

### 4.3.7 `hotObject`, `bandType`, `greaseType`, and the effect records

| Struct | Location | Fields |
| --- | --- | --- |
| `hotObject` | `GliderStructs.h:218-225` | `Rect bounds; short action; short who; Boolean isOn, stillOver; Boolean doScrutinize;` |
| `savedType` | `GliderStructs.h:227-233` | `Rect dest; GWorldPtr map; short where; short who;` — **`GWorldPtr` is a Mac offscreen-buffer handle; a port replaces it with an image buffer** |
| `sparkleType` | `GliderStructs.h:235-239` | `Rect bounds; short mode;` (mode -1 = free slot) |
| `flyingPtType` | `GliderStructs.h:241-249` | `Rect dest, whole; short start; short stop; short mode; short loops; short hVel, vVel;` |
| `flameType` | `GliderStructs.h:251-256` | `Rect dest, src; short mode; short who;` (`who` = saved-map index) |
| `pendulumType` | `GliderStructs.h:258-264` | `Rect dest, src; short mode, where; short who, link; Boolean toOrFro, active;` |
| `boundsType` | `GliderStructs.h:266-272` | `Boolean left, top, right, bottom;` — **this is the `'bnds'` resource layout** |
| `bandType` | `GliderStructs.h:274-279` | `Rect dest; short mode, count; short hVel, vVel;` |
| `linksType` | `GliderStructs.h:281-285` | `short srcRoom, srcObj; short destRoom, destObj;` |
| `greaseType` | `GliderStructs.h:287-295` | `Rect dest; short mapNum, mode; short who, where; short start, stop; short frame, hotNum; Boolean isRight;` |
| `starType` | `GliderStructs.h:297-302` | `Rect dest, src; short mode, who; short link, where;` |
| `shredType` | `GliderStructs.h:304-308` | `Rect bounds; short frame;` |
| `objDataType` | `GliderStructs.h:322-332` | `short roomNum, objectNum, roomLink, objectLink, localLink, hotNum, dynaNum; objectType theObject;` |
| `demoType` | `GliderStructs.h:334-339` | `long frame; char key; char padding;` — observed `sizeof` = **6** |
| `retroLink` | `GliderStructs.h:341-345` | `short room; short object;` |

## 4.4 Empirical verification of the on-disk house format

To confirm the `houseType`/`roomType`/`objectType` layouts against real data, the shipped
house `GliderPRO/Houses/Empty House.binhex` was decoded and parsed.

**Step 1 — BinHex 4.0 decode.** BinHex 4.0 uses a 64-character alphabet and 0x90-prefixed
RLE. The alphabet, transcribed from the standard:

```
!"#$%&'()*+,-012345689@ABCDEFGHIJKLMNPQRSTUVXYZ[`abcdefhijklmpqr
```

A Python decoder was written that: skips to the `:` delimiter, maps alphabet characters to
6-bit values, packs to bytes, expands 0x90 RLE (`0x90 0x00` is a literal 0x90; `0x90 n`
repeats the previous byte `n` times), then splits the header (name length, name, type,
creator, flags, data-fork length, resource-fork length) and the two forks with their CRCs.

**Step 2 — observed fork sizes.**

```
name:     'Empty House'
type:     'gliH'     creator: 'ozm5'
data fork:     13046 bytes
resource fork:  2670 bytes
finder flags:   0x0500
```

**Step 3 — the arithmetic check.** `866 + 348 * n = 13046` gives `348 n = 12180`,
`n = 35` exactly. So the data fork is a `houseType` header followed by **35** `roomType`
records with zero slack. This confirms both the 866 and the 348 figures against real
bytes.

**Step 4 — header field values, read big-endian at the offsets from §4.3.3.**

```
offset   0  version        = 0x0200  (= kHouseVersion, GliderDefines.h:517)
offset   2  unusedShort    = 0
offset   4  timeStamp      = 0x2C3130EC
offset   8  flags          = 0x00000000   (wardBit clear, phoneBit clear, bannerStars 0)
offset  12  initial.v/.h   = 64 / 83      (the Point is v-first)
offset 860  hasGame        = 0
offset 861  unusedBoolean  = 0
offset 862  firstRoom      = 0
offset 864  nRooms         = 35            <== matches the arithmetic exactly
```

**The version is `kHouseVersion` = 0x0200, not `kNewHouseVersion` = 0x0300.** This matters:
`HouseIO.c:394-400` reads the version and **refuses to load any house whose version is
`>= kNewHouseVersion`** (`if (wasHouseVersion >= kNewHouseVersion) { YellowAlert(
kYellowNewerVersion, 0); ...; return(false); }`). 0x0300 is a *future* format that the
shipped 1994 build rejects, and every writer in the tree stamps 0x0200
(`House.c:127`, `House.c:814`). A port must therefore treat 0x0200 as the only loadable
version and 0x0300 as a hard rejection, not as an alternative.

Incidentally `timeStamp` doubles as the house lock flag: `houseUnlocked = (((*thisHouse)
->timeStamp & 0x00000001) == 0)` (`HouseIO.c:402`). 0x2C3130EC is even, so Empty House is
unlocked.

**Step 5 — room 0, read at offset 866.**

```
name        (Str27,  offset  +0)  len 13, 'Untitled Room'
bounds      (short,  offset +28)  = 0      (no 'bnds' override; see §5.4)
leftStart   (Byte,   offset +30)  = 0
rightStart  (Byte,   offset +31)  = 0
unusedByte  (Byte,   offset +32)  = 0
visited     (Bool,   offset +33)  = 1      <== room 0 is marked ALREADY VISITED on disk
background  (short,  offset +34)  = 2000   (kBaseBackgroundID, a built-in background)
tiles[8]    (shorts, offset +36)  = 0,1,1,1,1,1,1,1
floor       (short,  offset +52)  = 1
suite       (short,  offset +54)  = 64
openings    (ushort, offset +56)  = 0
numObjects  (short,  offset +58)  = 4
objects[24] (offset +60), 12 bytes each
```

Note `visited` is **1**, not 0, for room 0 of the shipped Empty House. A port that asserts
"a fresh house has no visited rooms" will trip on real data, and the room-visitation scoring
path (§5.7.4) will award the first-room bonus differently depending on whether this byte is
honoured. `bounds` is 0 here, which is exactly the case that §5.4 says falls back to the
per-background derivation rather than the `'bnds'` mask.

**Step 6 — object decoding, cross-checked against `GliderDefines.h` constants.** All 35
rooms' `objects[]` were walked and `what` interpreted with the appropriate union member. The
observed `what` codes are 0x01, 0x05, 0x0D, 0x1D, 0x2C, 0x31, 0x32, 0x37, 0x39, 0x3C, 0x3E,
0x51, 0x52, 0x53, 0x56, and the `topLeft.v` values matched the hard-coded placement
constants exactly:

| Observed `what` | Constant | Where first seen | Observed `topLeft.v` | Expected constant | Value |
| --- | --- | --- | --- | --- | --- |
| 0x51 | `kCeilingLight` (`GliderDefines.h:387`) | room 0, obj 0 | 4 | `kCeilingLightTop` (`:477`) | 4 |
| 0x37 | `kDoorInLf` (`:366`) | room 0, obj 1 | 0 | `kDoorInTop` (`:483`) | 0 |
| 0x01 | `kFloorVent` (`:311`) | room 0, obj 2 | 305 | `kFloorVentTop` (`:467`) | 305 |
| 0x32 | `kDownStairs` (`:361`) | room 2, obj 3 | 28 | `kStairsTop` (`:474`) | 28 |

(Room 0 holds only four objects — 0x51, 0x37 and two 0x01s — so the `kDownStairs` sample
comes from room 2; `kUpStairs` 0x31 appears at `topLeft.v` 28 as well.)

**Four independent constants matching four independently-read bytes** is conclusive: the
struct layout, the field order, the big-endian encoding, and the `#pragma pack(2)`
alignment are all correct as documented.

**Step 7 — resource fork.** For completeness, the 2670-byte resource fork was parsed with a
Mac resource-map parser. It contains **only icon resources**: `ICN#`, `icl8`, `icl4`,
`ics#`, `ics8`, `ics4`, all with resource ID **-16455** (the standard custom-file-icon ID).
There is **no gameplay data in the resource fork of a house file** — a porter looking for
room data there will find nothing. (This was in fact the first wrong assumption made
during this analysis and is worth flagging.)


---

# Part 5 — The room load / teardown state machine

Room transitions are the biggest single state change in the game, they consume RNG, and
they are the natural network sync point. This part documents the whole sequence.

## 5.1 `NewGame` — the full game-start sequence

`GliderPRO/Sources/Play.c:74-280`, ordered with citations:

```
NewGame(mode):     // mode is kResumeGameMode (0) or kNewGameMode (1)
   1. if mode != kResumeGameMode then SetObjectsToDefaults()        // Play.c:102-103
   2. HideCursor()
   3. if mode == kResumeGameMode then SetHouseToSavedRoom()          // Play.c:105-106
      else if mode == kNewGameMode then SetHouseToFirstRoom()        // Play.c:107-108
      // note: `else if`, not `else` — a third mode value would set neither
   4. DetermineRoomOpenings()                                       // Play.c:109
   5. NilSavedMaps()                                                // Play.c:110
   6. gameFrame           <- 0L                                     // Play.c:112
   7. numBands            <- 0                                      // Play.c:113
   8. demoIndex           <- 0                                      // Play.c:114
   9. saidFollow          <- 0                                      // Play.c:115
  10. otherPlayerEscaped  <- kNoOneEscaped (-1)                     // Play.c:116
  11. onePlayerLeft       <- false                                  // Play.c:117
  12. playerSuicide       <- false                                  // Play.c:118
  13. if twoPlayerGame then                                         // Play.c:120-128
  14.     InitGlider(&theGlider,  kNewGameMode)                      // Play.c:122
  15.     InitGlider(&theGlider2, kNewGameMode)                      // Play.c:123
  16.     load kGliderPictID into glidSrcMap, kGlider2PictID into glid2SrcMap  // :124-127
  17. else                                                          // Play.c:129-136
  18.     InitGlider(&theGlider, mode)                               // Play.c:131
  19.     load kGliderPictID into glidSrcMap, kGliderFoilPictID into glid2SrcMap  // :132-135
  20. ... (window / offscreen setup) ...
  21. DrawLocale()                                                  // Play.c:163  <== CONSUMES RNG
  22. RefreshScoreboard(kNormalTitleMode)                           // Play.c:164
  23. ... (fade in / transition) ...
  24. InitGarbageRects()                                            // Play.c:184
  25. StartGliderFadingIn(&theGlider)                               // Play.c:185
  26. if twoPlayerGame then                                         // Play.c:186
  27.     StartGliderFadingIn(&theGlider2)                           // Play.c:188
  28.     TagGliderIdle(&theGlider2)                                 // Play.c:189
  29.     theGlider2.dontDraw <- true                                // Play.c:190
  30. InitTelephone()                                               // Play.c:192  <== CONSUMES RNG (3 draws)
  31. ...
  32. playing <- true                                               // Play.c:213
  33. PlayGame()                                                    // Play.c:214
  34. ... (teardown) ...
  35. incrementModeTime <- TickCount() + kIdleSplashTicks (7200)      // Play.c:277
```

**RNG order at game start is: `DrawLocale()` first (variable number of draws), then
`InitTelephone()` (exactly 3 draws).** Note that step 28 idles glider 2 for 30 frames at
game start too, so player 2 is frozen for the first second of a two-player game.

**The two-player branch discards `mode`.** Both `InitGlider` calls at `Play.c:122-123` pass
the literal `kNewGameMode`, whatever `mode` the caller supplied; only the single-player
branch (`:131`) forwards it. So a two-player game **cannot resume a saved position** — the
glider is always initialised as if starting fresh, even though step 3 has already pointed
the house at the saved room. A port must replicate the asymmetry (or consciously fix it and
accept that saved-game behaviour will differ from the original).

## 5.2 `SetObjectsToDefaults` — resetting the whole house

`GliderPRO/Sources/Play.c:603-708`. Walks every room and every object in the house handle
and copies each object class's `initial` field into its `state` field, plus:

* `(*thisHouse)->rooms[i].visited = false;` (`GliderPRO/Sources/Play.c:619`) — clears the
  room-visit score flags.
* `kDeluxeTrans` (0x40) gets special nibble handling at
  `GliderPRO/Sources/Play.c:657-661`.
* `kStereo` (0x69) **reads** the global preference *into* the object:
  `objects[i].data.g.state = isPlayMusicGame;` (`GliderPRO/Sources/Play.c:675-677`). The data
  flows preference → object state, not the other way round, so whether a house's stereos
  start playing depends on a **user preference**, and `SetObjectsToDefaults` writes that
  preference into the house handle. For a lockstep port this is a divergence source: two
  peers with different `isPlayMusicGame` settings get different initial object state.
* The inner loop runs over **all `kMaxRoomObs` = 24 object slots**
  (`GliderPRO/Sources/Play.c:620`), not `numObjects` — empty slots go through the `switch`
  too and fall out of the `default`.

This is the "reset the world" function. **It mutates the in-memory house handle**, so the
house data is *not* read-only during play — a port must copy-on-load or version the house
state. It also means `visited` flags and object `state` flags are part of the simulation
state that a lockstep peer must agree on.

## 5.3 `ReadyLevel` — the room-transition core

`GliderPRO/Sources/RoomGraphics.c:402-418`:

```
ReadyLevel():
   1. NilSavedMaps()                                       // frees all 24 saved-map slots
   2. #ifdef COMPILEQT
   3.     tvInRoom <- false
   4.     tvWithMovieNumber <- -1
   5.     StopMovie(...) if a movie was playing
   6. #endif
   7. DetermineRoomOpenings()                              // sets leftThresh/rightThresh
   8. DrawLocale()                                         // <== CONSUMES RNG
   9. InitGarbageRects()                                   // resets sparkles/flyingPoints, nextFrame
```

`NilSavedMaps` (`GliderPRO/Sources/DynamicMaps.c:46-62`) walks all `kMaxSavedMaps` = 24
slots, disposes any non-nil GWorld (`:52-57`), sets `savedMaps[i].where = -1` and
`savedMaps[i].who = -1` (`:58-59`) and finally `numSavedMaps = 0` (`:61`). So the 24-slot
budget is **per room**, freshly allocated on every transition, and the `where`/`who` sentinels
are -1 rather than 0 — `RestoreFromSavedMap` matches on them, so a port that zero-fills
instead of writing -1 will match slot 0's room/object by accident.

## 5.4 `DetermineRoomOpenings` — the escape thresholds

`GliderPRO/Sources/Room.c:816-933`. Sets five globals: `leftThresh`, `rightThresh`,
`leftOpen`, `rightOpen`, and (via helpers) `topOpen`/`bottomOpen`.

```
DetermineRoomOpenings():
   1. whichBack <- thisRoom->background                         // :821
   2. leftTile  <- thisRoom->tiles[0]                           // :822
   3. rightTile <- thisRoom->tiles[kNumTiles - 1]  // tiles[7]  // :823
   4. if whichBack >= kUserBackground (3000) then               // :825  USER BACKGROUND PATH
   5.     if thisRoom->bounds != 0 then boundsCode <- thisRoom->bounds >> 1     // :827-828
   6.     else                          boundsCode <- GetOriginalBounding(whichBack)  // :830
   7.     leftOpen  <- ((boundsCode & 0x0001) == 0x0001)          // :831
   8.     rightOpen <- ((boundsCode & 0x0004) == 0x0004)          // :832
   9.     leftThresh  <- leftOpen  ? kNoLeftWallLimit  : kLeftWallLimit    // :834-837
  10.     rightThresh <- rightOpen ? kNoRightWallLimit : kRightWallLimit   // :839-842
  11. else                                                      // :844  BUILT-IN PATH
  12.     switch whichBack:                                      // :846
  13.       kSimpleRoom, kPaneledRoom, kBasement, kChildsRoom, kAsianRoom,
  14.       kUnfinishedRoom, kSwingersRoom, kBathroom, kLibrary, kSky:      // :848-857
  15.         leftThresh  <- (leftTile  == 0) ? kLeftWallLimit  : kNoLeftWallLimit   // :858-861
  16.         rightThresh <- (rightTile == 7) ? kRightWallLimit : kNoRightWallLimit  // :862-865
  17.         leftOpen  <- (leftTile  != 0) ; rightOpen <- (rightTile != 7)          // :866-867
  18.       kDirt:                                                                   // :870
  19.         leftThresh  <- (leftTile  == 1) ? kLeftWallLimit  : kNoLeftWallLimit   // :871-874
  20.         rightThresh <- (rightTile == 7) ? kRightWallLimit : kNoRightWallLimit  // :875-878
  21.         leftOpen  <- (leftTile  != 0) ; rightOpen <- (rightTile != 7)          // :879-880
  22.       kMeadow:                                                                 // :883
  23.         leftThresh  <- (leftTile  == 6) ? kLeftWallLimit  : kNoLeftWallLimit   // :884-887
  24.         rightThresh <- (rightTile == 7) ? kRightWallLimit : kNoRightWallLimit  // :888-891
  25.         leftOpen  <- (leftTile  != 6) ; rightOpen <- (rightTile != 7)          // :892-893
  26.       kGarden, kSkywalk, kField, kStratosphere, kStars:                        // :896-900
  27.         leftThresh <- kNoLeftWallLimit ; rightThresh <- kNoRightWallLimit      // :901-902
  28.         leftOpen <- true ; rightOpen <- true                                   // :903-904
  29.       default:  same as the kSimpleRoom group                                  // :907-920
  30. bottomOpen <- !DoesRoomHaveFloor()                          // :924-927
  31. topOpen    <- !DoesRoomHaveCeiling()                        // :929-932
```

**The earlier draft of this section had the two paths backwards.** The `'bnds'`-style
bitmask is used **only for user backgrounds** (PICT ID >= `kUserBackground` = 3000).
Built-in backgrounds (2000..2017) never consult `'bnds'` at all — they decide openness from
**tile indices** via a per-background `switch`, and the sentinel tile value differs per
background family (0 for the room group, 1 for `kDirt`'s left, 6 for `kMeadow`'s left, 7 for
everyone's right, and `kGarden`/`kSkywalk`/`kField`/`kStratosphere`/`kStars` are
unconditionally open on both sides).

Within the user path, note the fallback order: `thisRoom->bounds >> 1` is used **only when
`bounds != 0`**; a zero `bounds` falls back to `GetOriginalBounding(whichBack)`. (Room 0 of
the shipped Empty House has `bounds == 0`, §4.4.) `GetOriginalBounding`
(`GliderPRO/Sources/Room.c:937-966`) loads a **`'bnds'` resource** with
`GetResource('bnds', theID)` (`:942`); the resource is four `Boolean` bytes (`boundsType`,
`GliderPRO/Headers/GliderStructs.h:266-272`), recombined as
`boundCode = left*1 + top*2 + right*4 + bottom*8` (`:953-960`). If the resource is missing
it returns **0** and raises `kYellowNoBoundsRes` only when a PICT with that ID exists
(`:943-948`) — so a missing `'bnds'` silently means "closed on all sides".

The opening bitmask:

| Bit | Hex | Meaning |
| --- | --- | --- |
| 0 | 0x0001 | left is open |
| 1 | 0x0002 | top is open |
| 2 | 0x0004 | right is open |
| 3 | 0x0008 | bottom is open |

**Only bits 0 and 2 are read by `DetermineRoomOpenings`.** Bits 1 and 3 (top/bottom) are
ignored here; vertical openness comes from `DoesRoomHaveFloor`
(`GliderPRO/Sources/Room.c:1138`) and `DoesRoomHaveCeiling`
(`GliderPRO/Sources/Room.c:1172`), which inspect object types.

**Original bug to preserve (or consciously fix): `kDirt` is self-inconsistent.** Its
`leftThresh` is chosen on `leftTile == 1` (`:871`) but its `leftOpen` is computed as
`leftTile != 0` (`:879`) — copied verbatim from the `kSimpleRoom` group. For `leftTile == 1`
the room therefore reports `leftOpen == true` while `leftThresh` is set to the *walled* value
`kLeftWallLimit`. Since `CheckEscapeLeftTwo` branches on `leftThresh == kLeftWallLimit`
(§3.5.4) and other code reads `leftOpen`, the two disagree. Reproduce the code as written
rather than "fixing" it, or dirt rooms will behave differently.

## 5.5 `GetNeighborRoomNumber` — the nine-neighbour lookup

`GliderPRO/Sources/Room.c:562-635`:

```
GetNeighborRoomNumber(which):
   1. map `which` (0..8) to (hDelta, vDelta):
         kCentralRoom   (0) -> ( 0,  0)
         kNorthRoom     (1) -> ( 0, +1)
         kNorthEastRoom (2) -> (+1, +1)
         kEastRoom      (3) -> (+1,  0)
         kSouthEastRoom (4) -> (+1, -1)
         kSouthRoom     (5) -> ( 0, -1)
         kSouthWestRoom (6) -> (-1, -1)
         kWestRoom      (7) -> (-1,  0)
         kNorthWestRoom (8) -> (-1, +1)
   2. target = (thisRoom->suite + hDelta, thisRoom->floor + vDelta)
   3. linear search i = 0 .. numberRooms-1 for the first room whose
      (suite, floor) equals target
   4. return that index, or kRoomIsEmpty (-1) if none
```

(The `vDelta` signs above were read off `GliderPRO/Sources/Room.c:562` and verified: the
north row is `+1`, the south row is `-1`, i.e. `floor` increases *upward*. What matters for
determinism is that the lookup is a **linear search returning the first match**, so duplicate (suite, floor) pairs in a malformed house resolve
by lowest room index. That is deterministic but house-dependent.)

Rooms are addressed by a 2-D `(suite, floor)` grid, with `kMaxNumRoomsH` = 128 and
`kMaxNumRoomsV` = 64 (`GliderPRO/Headers/GliderDefines.h:543-544`).
`kNumUndergroundFloors` = 8 (`GliderPRO/Headers/GliderDefines.h:535`).

## 5.6 `GetNumberOfLights` — why lighting affects RNG indirectly

`GliderPRO/Sources/Room.c:970-1099`:

```
GetNumberOfLights(where):
   1. if theMode == kEditMode then use thisRoom (:976-1032)
      else use (*thisHouse)->rooms[where] under HLock (:1033-1097)
      // the two branches are otherwise structurally identical
   2. switch background:                                       // :1038 (play)
   3.   kGarden (2009), kSkywalk (2010), kMeadow (2012), kField (2013),
        kRoof (2014), kSky (2015), kStratosphere (2016), kStars (2017):
   4.       count <- 1                                          // :1048
   5.   kDirt (2011):
   6.       count <- 0                                          // :1052
   7.       if all eight tiles[] are 0 then count <- 1           // :1053-1061
   8.   default:
   9.       count <- 0                                          // :1065
  10. if count == 0 then                                        // :1068  NOT an early return
  11.     for i = 0 .. kMaxRoomObs-1 (all 24 slots):             // :1070
  12.         kDoorInLf, kDoorInRt, kWindowInLf, kWindowInRt, kWallWindow -> count++  // :1074-1080
  13.         kCeilingLight, kLightBulb, kTableLamp, kHipLamp, kDecoLamp,
              kFlourescent, kTrackLight, kInvisLight -> if data.f.state then count++  // :1082-1092
  14. return count                                              // :1098
```

Two corrections to an earlier draft of this section: the outdoor/dirt switch does **not
return early** — it sets `count`, and the object-counting loop runs whenever `count` is
still 0. So a `kDirt` room whose tiles are *not* all zero (`count` stays 0) **falls through
and counts its doors, windows and lit lamps** rather than reporting 0 lights. Only the eight
outdoor backgrounds short-circuit, because they set `count` to 1. Note also the object loop
walks all 24 slots, and the lit test reads `data.f.state` in the play-mode branch and
`data.f.initial` in the edit-mode branch.

`numLights` gates `isLit = (numLights > 0)` in `DrawARoomsObjects`
(`GliderPRO/Sources/ObjectDrawAll.c:38`), which changes **which sprite variant** is drawn
— and in an unlit room some object branches skip their registration calls, so
**turning a light off can change how much RNG a room load consumes.** Since light state is
part of the house's object `state` fields, and those are mutated by
`SetObjectState`/`TriggerSwitch` during play, RNG consumption at room load depends on
gameplay history.

`kFirstOutdoorBack` = 2009, `kNumBackgrounds` = 18, `kBaseBackgroundID` = 2000,
`kUserBackground` = 3000, `kUserStructureRange` = 3300
(`GliderPRO/Headers/GliderDefines.h:519-523`).

## 5.7 The four transition functions

`GliderPRO/Sources/Transit.c`. All four share a skeleton; `MoveRoomToRoom` is the most
complex because it handles four directions.

### 5.7.1 `MoveRoomToRoom(thisGlider, where)` — `Transit.c:151-310`

```
MoveRoomToRoom(thisGlider, where):
   1. HandleRoomVisitation()                                        // Transit.c:155
   2. switch where:
   3.   kToRight (2):
   4.       SetMusicalMode(kProdGameScoreMode (-4))
   5.       if twoPlayerGame: UndoGliderLimbo(&theGlider); UndoGliderLimbo(&theGlider2)
   6.                         InsureGliderFacingRight(&theGlider); ...(&theGlider2)
   7.          else:          UndoGliderLimbo(thisGlider); InsureGliderFacingRight(thisGlider)
   8.       ForceThisRoom(localNumbers[kEastRoom])
   9.       OffsetGlider(<gliders>, kToLeft)
  10.       QSetRect(&enterRect, 0, 0, 48, 20)
  11.       QOffsetRect(&enterRect, 0, kGliderStartsDown (32)
                                       + (short)thisRoom->leftStart - 2)
  12.       enteredRect <- enterRect
  13.   kToLeft (4):  mirror image, using (kRoomWide - 48) and thisRoom->rightStart
  14.   kAbove (1):
  15.       SetMusicalMode(kKickGameScoreMode (-3))
  16.       ForceThisRoom(localNumbers[kNorthRoom])
  17.       if takingTheStairs then ReadyGliderForTripUpStairs(...)
  18.       else OffsetGlider(..., kBelow); enteredRect <- dest
  19.   kBelow (3):  mirror, kSouthRoom, ReadyGliderForTripDownStairs / OffsetGlider(kAbove)
  20. // common tail, Transit.c:290-309
  21. if (twoPlayerGame) and (!onePlayerLeft) then
  22.     if firstPlayer == kPlayer1 then TagGliderIdle(&theGlider2)
  23.     else                            TagGliderIdle(&theGlider)
  24. ReadyLevel()                                                    // <== CONSUMES RNG
  25. RefreshScoreboard(kNormalTitleMode)
  26. WipeScreenOn(where, &justRoomsRect)
  27. #ifdef COMPILEQT
  28.     RenderFrame()                                               // <== ADVANCES ANIMATION
  29.     (restart movie if tvInRoom)
  30. #endif
```

`kGliderStartsDown` = 32 (`GliderPRO/Headers/GliderDefines.h:569`). `leftStart` and
`rightStart` are the per-room `Byte` fields at offsets 30 and 31 of `roomType`, so the
horizontal-entry vertical position is house data.

Steps 21-23: **the player who did NOT reach the exit first gets idled for 30 frames on
arrival.** `firstPlayer` was set by `FlagGliderInLimbo`.

Step 28 is significant: **a room transition calls `RenderFrame` once, outside the main
loop.** That advances every animation state machine and the pendulum `clockFrame` by one
extra step, and (via `RenderShreds`) can even spawn a sparkle. It also runs the
`TickCount()` busy-wait and resets `nextFrame`. A port must include this extra render step
or the animation phase after every transition will be off by one.

### 5.7.2 `TransportRoomToRoom`, `MoveDuctToDuct`, `MoveMailToMail`

`GliderPRO/Sources/Transit.c:314-348`, `:352-387`, `:391-426`. All three have the identical
shape:

```
<Transition>(thisGlider):
   1. SetMusicalMode(kKickGameScoreMode (-3))
   2. HandleRoomVisitation()
   3. sameRoom <- (transRoom == thisRoomNumber)
   4. if !sameRoom then ForceThisRoom(transRoom)
   5. if twoPlayerGame then
   6.     UndoGliderLimbo(&theGlider) ; UndoGliderLimbo(&theGlider2)
   7.     ReadyGliderFromTransit(&theGlider, linkedToWhat)
   8.     ReadyGliderFromTransit(&theGlider2, linkedToWhat)
   9. else
  10.     ReadyGliderFromTransit(thisGlider, linkedToWhat)
  11. if !sameRoom then ReadyLevel()                        // <== CONSUMES RNG only if room changed
  12. RefreshScoreboard(kNormalTitleMode)
  13. if !sameRoom then WipeScreenOn(kAbove, &justRoomsRect)
  14. RenderFrame()                                        // <== ALWAYS, even for same-room
  15. (restart movie)
```

**`sameRoom` matters for RNG**: a transporter that lands you in the *same* room skips
`ReadyLevel()` entirely, so it consumes zero RNG, but still calls `RenderFrame()`.

### 5.7.3 `ReadyGliderFromTransit` and the four link classes

`WhatAreWeLinkedTo(where, who)` (`GliderPRO/Sources/Transit.c:31-61`) classifies the
destination object into one of:

| Constant | Value | Location |
| --- | --- | --- |
| `kLinkedToOther` | 0 | `GliderPRO/Headers/GliderDefines.h:610` |
| `kLinkedToLeftMailbox` | 1 | `:611` |
| `kLinkedToRightMailbox` | 2 | `:612` |
| `kLinkedToCeilingDuct` | 3 | `:613` |
| `kLinkedToFloorDuct` | 4 | `:614` |

`ReadyGliderFromTransit(thisGlider, toWhat)`
(`GliderPRO/Sources/Transit.c:65-147`):

```
ReadyGliderFromTransit(thisGlider, toWhat):
   1. if (twoPlayerGame) and (onePlayerLeft) and
        (thisGlider->which == playerDead) then return               // Transit.c:69-70
   2. FlagGliderNormal(thisGlider)                                  // Transit.c:72
   3. switch toWhat:
   4.   kLinkedToOther:        StartGliderTransportingIn(thisGlider)
                              CenterRectInRect(&dest, &transRect)
   5.   kLinkedToLeftMailbox:  clip.right -= 64 ; clip.bottom -= 25
                              dest.bottom = clip.bottom - 4
   6.   kLinkedToRightMailbox: clip.left += 79 ; clip.bottom -= 25
   7.   kLinkedToCeilingDuct:  StartGliderDuctingIn(thisGlider)
                              centre, then offset up by RectTall(dest)
   8. if (twoPlayerGame) and (thisGlider->which != firstPlayer) then
   9.     TagGliderIdle(thisGlider)                                  // Transit.c:145-146
```

The magic numbers 64, 25, 4, 79 are mailbox sprite geometry.

### 5.7.4 `HandleRoomVisitation` — the score-once-per-room latch

`GliderPRO/Sources/Transit.c:430-445`:

```
HandleRoomVisitation():
   1. if !thisRoom->visited then
   2.     HLock((Handle)thisHouse)
   3.     thisHousePtr->rooms[localNumbers[kCentralRoom]].visited <- true
   4.     HSetState((Handle)thisHouse, ...)
   5.     theScore += kRoomVisitScore (100)
   6.     thisRoom->visited <- true
```

**Note it writes the flag in two places** — the house handle *and* the local `thisRoom`
copy — because `thisRoom` is a by-value copy made by `ForceThisRoom`
(`GliderPRO/Sources/Room.c:369-386`):

```c
void ForceThisRoom (short roomNumber)      // Room.c:369-386
{
	char		tagByte;

	if (roomNumber == -1)                          // Room.c:373-374  EARLY RETURN
		return;

	tagByte = HGetState((Handle)thisHouse);
	HLock((Handle)thisHouse);
	if (roomNumber < (*thisHouse)->nRooms)         // Room.c:378  BOUNDS CHECK
		*thisRoom = (*thisHouse)->rooms[roomNumber];   // Room.c:379  BY-VALUE COPY
	else
		YellowAlert(kYellowIllegalRoomNum, 0);     // Room.c:381
	HSetState((Handle)thisHouse, tagByte);

	previousRoom = thisRoomNumber;                 // Room.c:384
	thisRoomNumber = roomNumber;                   // Room.c:385
}
```

Two guards an earlier draft of this snippet omitted, both of which change behaviour:

- **`roomNumber == -1` returns immediately** (`:373-374`) *without* touching `previousRoom`
  or `thisRoomNumber`. `kRoomIsEmpty` is -1 and `GetNeighborRoomNumber` returns it for a
  missing neighbour, so walking into a non-existent room leaves the room pointers unchanged.
- **`roomNumber >= nRooms` skips the copy but still updates `previousRoom`/`thisRoomNumber`**
  (`:378-385`). `thisRoom` therefore keeps the *previous* room's 348 bytes while
  `thisRoomNumber` names a different room — the two are out of sync, and the only signal is a
  `YellowAlert`. A port should reproduce the -1 early return exactly; the out-of-range case is
  unreachable with well-formed houses but must not silently index out of bounds.

**`thisRoom` is a 348-byte snapshot, not a pointer into the house.** Any mid-room mutation
that is written only to `thisRoom` is lost on the next transition, and any written only to
the house handle is invisible until the next `ForceThisRoom`. This dual-write pattern is
a real correctness hazard in a port. `HandleRoomVisitation` gets it right; a porter must
check every other `thisRoom->` write for the same treatment.

### 5.7.5 `ForceKillGlider` and `FollowTheLeader` — the shared-fate suicide path

`ForceKillGlider()` (`GliderPRO/Sources/Transit.c:449-469`):

```
ForceKillGlider():
   1. if one glider is in kGliderInLimbo and the other is not already fading out then
   2.     StartGliderFadingOut(the non-limbo glider)
   3.     PlayPrioritySound(kFadeOutSound (2), kFadeOutPriority (901))
   4.     playerSuicide <- true
```

This is the "I'm stuck, kill me so we can move on" escape hatch: if one player is waiting
in limbo at an exit and the other cannot reach it, the waiting player can force the other
to die. `playerSuicide` then routes `OffAMortal` into `FollowTheLeader` instead of a normal
respawn (`GliderPRO/Sources/Player.c:1529-1530`). Note `playerSuicide` is *not* cleared in
`OffAMortal`; the only reset is at the top of `FollowTheLeader` (`Transit.c:478`).

`FollowTheLeader()` (`GliderPRO/Sources/Transit.c:473-557`):

```
FollowTheLeader():
   1. playerSuicide      <- false                                   // Transit.c:478
   2. wasEscaped         <- otherPlayerEscaped                      // Transit.c:479
   3. otherPlayerEscaped <- kNoOneEscaped (-1)                      // Transit.c:480
   4. // move the NON-limbo glider onto the limbo glider's position
   5. if theGlider.mode == kGliderInLimbo then                      // Transit.c:482
   6.     oneOrTwo <- true                                          // Transit.c:484
   7.     theGlider2.dest       <- theGlider.dest                    // Transit.c:485
   8.     theGlider2.destShadow <- theGlider.destShadow              // Transit.c:486
   9.     theGlider2.whole       <- theGlider2.dest                  // Transit.c:487
  10.     theGlider2.wholeShadow <- theGlider2.destShadow            // Transit.c:488
  11. else if theGlider2.mode == kGliderInLimbo then                 // Transit.c:490
  12.     oneOrTwo <- false ; mirror of steps 7-10                   // Transit.c:492-496
  13. // NO else — see the warning below
  14. switch wasEscaped:                                            // Transit.c:499-556
  15.     kPlayerEscapedUp (-4), kPlayerEscapingUpStairs (-8),
          kPlayerEscapedUpStairs (-6)   -> MoveRoomToRoom(oneOrTwo ? &theGlider2 : &theGlider, kAbove)   // :501-508
  16.     kPlayerEscapedDown (-5), kPlayerEscapingDownStairs (-9),
          kPlayerEscapedDownStairs (-7) -> MoveRoomToRoom(..., kBelow)     // :510-517
  17.     kPlayerEscapedLeft  (-3)  -> MoveRoomToRoom(..., kToLeft)         // :519-524
  18.     kPlayerEscapedRight (-2)  -> MoveRoomToRoom(..., kToRight)        // :526-531
  19.     kPlayerTransportedOut (-10) -> TransportRoomToRoom(...)           // :533-538
  20.     kPlayerMailedOut    (-12) -> MoveMailToMail(...)                  // :540-545
  21.     kPlayerDuckedOut    (-11) -> MoveDuctToDuct(...)                  // :547-552
  22.     default -> nothing                                               // :554
```

This is the shared-fate mechanic in its purest form: the force-killed player is **teleported
onto the waiting (limbo) player's position** and both proceed together. For a networked race
mode, this is exactly the semantics you want to reproduce. Four details an earlier draft got
wrong:

- **The glider passed to `MoveRoomToRoom` is the *non*-limbo one.** When `theGlider` is in
  limbo, `oneOrTwo` is `true` and every case moves **`&theGlider2`** (`:505`, `:514`, `:521`,
  `:528`, `:535`, `:542`, `:549`). The limbo glider is left for `UndoGliderLimbo` inside the
  transition to release.
- **`whole`/`wholeShadow` are not copied from the limbo glider** — they are assigned from the
  target's own freshly written `dest`/`destShadow` (`:487-488`, `:495-496`). The values coincide
  here, but the dependency order differs.
- **The four stairs constants fall through** into the plain `kAbove`/`kBelow` cases
  (`:501-503`, `:510-512`), exactly as in `OffAMortal` (§3.8.3).
- **`oneOrTwo` is left uninitialised when neither glider is in `kGliderInLimbo`.** There is no
  `else` after `:490`, yet `oneOrTwo` is read at `:504`/`:513`/`:520`/`:527`/`:534`/`:541`/`:548`.
  In practice `FollowTheLeader` is only reached via `playerSuicide`, which `ForceKillGlider`
  only sets when one glider *is* in limbo, so the path is unreachable — but a port must not
  rely on whatever a stack-garbage `Boolean` happens to be. Pick a defined behaviour and
  document it; do not translate this as a real branch.

## 5.8 `SetObjectState` — the object-state mutator

`GliderPRO/Sources/Objects.c:366-700` with signature
`SetObjectState(short room, short object, short action, short local)`. `action` is one of:

| Constant | Value | Location |
| --- | --- | --- |
| `kToggle` | 0 | `GliderPRO/Headers/GliderDefines.h:441` |
| `kForceOn` | 1 | `:442` |
| `kForceOff` | 2 | `:443` |
| `kOneShot` | 3 | `:444` |

`SetObjectState` writes into the **house handle**, so object state persists across room
transitions and is part of the world state. `GetObjectState(room, object)`
(`GliderPRO/Sources/Objects.c:703`) reads it back.

`ListAllLocalObjects()` (`GliderPRO/Sources/Objects.c:300-348`) flattens the nine
neighbour rooms' 24-slot object arrays into `masterObjects[]` (cap 216) with
`numMasterObjects`, in neighbour-index-then-object-index order. `ListOneRoomsObjects(where)`
(`:254`) is the per-room helper. `IsThisValid(where, who)` (`:89`) is the per-object
validity predicate used by both `ListOneRoomsObjects` and `DrawARoomsObjects`.

`masterObjects[n].dynaNum` is assigned at the *end* of `DrawARoomsObjects`
(`GliderPRO/Sources/ObjectDrawAll.c:944-962`) by a linear back-search matching
`objectNum` and `roomNum`. So the object→dynamic-object mapping is established after all
the `AddDynamicObject` calls, and it depends on `numMasterObjects` ordering.

---

# Part 6 — Mac-specific and machine-specific dependencies

## 6.1 The `thisMac` environment record

`GliderPRO/Headers/Environ.h:11-32`, complete:

```c
typedef struct
{
	Rect		screen, gray;            // :13
	long		dirID;                   // :14
	short		wasDepth, isDepth;       // :15
	short		thisResFile;             // :16
	short		numScreens;              // :17
	short		vRefNum;                 // :18
	Boolean		can1Bit, can4Bit, can8Bit;   // :20
	Boolean		wasColorOrGray;          // :22
	Boolean		hasWNE;                  // :23
	Boolean		hasSystem7, hasColor, hasGestalt, canSwitch, canColor;
	Boolean		hasSM3;                  // :29
	Boolean		hasQT;                   // :30
	Boolean		hasDrag;                 // :31
} macEnviron;
extern macEnviron thisMac;
```

Populated by `CheckOurEnvirons` in `GliderPRO/Sources/Environ.c`:

| Field | Set at | Value / source |
| --- | --- | --- |
| `hasWNE` | `Environ.c:447` | hard-coded `true` with a `// TEMP` comment |
| `hasSM3` | `Environ.c:451` | hard-coded `true` with a `// TEMP` comment |
| `hasQT` | `Environ.c:452` | `DoWeHaveQuickTime()` |
| `numScreens` | `Environ.c:462` | `HowManyUsableScreens(false, true, true)` |
| `screen` | `Environ.c:463` | `GetDeviceRect(&thisMac.screen)` |
| `wasColorOrGray` | `Environ.c:466` | `AreWeColorOrGrayscale()` |
| `isDepth` | `Environ.c:542` | `WhatsOurDepth()` |

`WhatsOurDepth()` (`GliderPRO/Sources/Environ.c:232-253`) reads
`(**(**thisGDevice).gdPMap).pixelSize` at `:243`, defaulting to 1.

### 6.1.1 Which of these affect the simulation

| Field | Affects simulation? | How |
| --- | --- | --- |
| `screen.right` | **YES** | `if ((numNeighbors > 1) && (thisMac.screen.right <= 512)) numNeighbors = 1;` at `GliderPRO/Sources/Main.c:191-192` → changes room-load RNG consumption |
| `isDepth` | **no** (rendering only) | `if (thisMac.isDepth == 4)` nudges the *destination* rect by one pixel in `AddCandleFlame` (`DynamicMaps.c:326`), `AddTikiFlame` (`:409`), `AddBBQCoals` (`:495`), `AddPendulum` (`:584`), `AddStar` (`:671`). Re-verified: the RNG draw in each of those five functions is gated only on `BackUpToSavedMap(&bounds, …) != -1`, and `bounds` is built by a depth-independent `QSetRect(&bounds, 0, 0, w, h)` — so **depth does not change RNG consumption**, only pixel positions |
| `hasQT` | **YES (indirectly)** | gates `tvInRoom`, `MoviesTask` and the extra `RenderFrame()` in transitions |
| memory availability | **YES** | `CreateOffScreenGWorld` failure makes `BackUpToSavedMap` return -1, suppressing an RNG draw |
| `numScreens`, `hasColor`, `hasSystem7` | no | startup gating only (`RedAlert` if missing) |
| `hasSM3`, `hasDrag`, `wasDepth`, `can*Bit`, `dirID`, `vRefNum`, `thisResFile`, `gray` | no | |

## 6.2 `numNeighbors` — the most dangerous preference

Full derivation from `GliderPRO/Sources/Main.c:52-201`:

* If a prefs file loaded successfully: `numNeighbors = thePrefs.wasNumNeighbors;`
  (`GliderPRO/Sources/Main.c:111`).
* Otherwise the defaults block runs, including `numNeighbors = 9;`
  (`GliderPRO/Sources/Main.c:154`), `isDepthPref = kSwitchIfNeeded (0);`
  (`GliderPRO/Sources/Main.c:146`), volume clamped to 1..3
  (`GliderPRO/Sources/Main.c:140-144`), `maxFiles = 48; willMaxFiles = 48;`,
  `doAutoDemo = true;`, `doBackground = false;`.
* Then unconditionally:
  `if ((numNeighbors > 1) && (thisMac.screen.right <= 512)) numNeighbors = 1;`
  (`GliderPRO/Sources/Main.c:191-192`).
* Then `UnivGetSoundVolume(&wasVolume, thisMac.hasSM3); UnivSetSoundVolume(isVolume, ...);`
  (`GliderPRO/Sources/Main.c:194-195`).

`wasNumNeighbors` is a `short` in `prefsInfo` at `GliderPRO/Headers/Externs.h:253`.
`kPrefsVersion` = **0x0034** (`GliderPRO/Sources/Main.c:16`).

`prefsInfo` sits inside `#pragma options align=mac68k` … `#pragma options align=reset`
(`GliderPRO/Headers/Externs.h:230`, `:269`) — i.e. 2-byte packing, the same as everything
else on disk. Its declaration order (`GliderPRO/Headers/Externs.h:232-267`):

```
Str32 wasDefaultName; Str15 wasLeftName, wasRightName; Str15 wasBattName, wasBandName;
Str15 wasHighName; Str31 wasHighBanner;
/* long encrypted, fakeLong;   <-- COMMENTED OUT at Externs.h:239 */
long wasLeftMap, wasRightMap; long wasBattMap, wasBandMap;
short wasVolume; short prefVersion; short wasMaxFiles;
short wasEditH, wasEditV; short wasMapH, wasMapV; short wasMapWide, wasMapHigh;
short wasToolsH, wasToolsV; short wasLinkH, wasLinkV; short wasCoordH, wasCoordV;
short isMapLeft, isMapTop; short wasNumNeighbors;        // :253
short wasDepthPref; short wasToolGroup; short smWarnings; short wasFloor, wasSuite;
Boolean wasZooms, wasMusicOn; Boolean wasAutoEdit, wasDoColorFade;
Boolean wasMapOpen, wasToolsOpen; Boolean wasCoordOpen, wasQuickTrans;
Boolean wasIdleMusic, wasGameMusic; Boolean wasEscPauseKey;
Boolean wasDoAutoDemo, wasScreen2; Boolean wasDoBackground, wasHouseChecks;
Boolean wasPrettyMap, wasBitchDialogs;
```

The four `long` key maps (`wasLeftMap`, `wasRightMap`, `wasBattMap`, `wasBandMap`) are the
KeyMap offsets stored in `gliderType.leftKey` etc.

## 6.3 The complete Mac Toolbox surface a port must replace

| Toolbox area | Calls used | What the Go port needs |
| --- | --- | --- |
| **QuickDraw drawing** | `CopyBits`, `CopyMask`, `PaintRect`, `EraseRect`, `DrawPicture`, `ForeColor`/`BackColor`, `ClipRect`, `RectRgn` | a software blitter over an indexed or RGBA framebuffer; `srcCopy`/`transparent`/`srcXor` transfer modes |
| **QuickDraw geometry** | `SectRect`, `OffsetRect`, `InsetRect`, `SetRect`, `UnionRect`, plus the game's own `QSetRect`/`QOffsetRect`/`HOffsetRect`/`VOffsetRect` | integer `Rect{Top,Left,Bottom,Right int16}`. `SectRect` is **strict-inequality, half-open** (touching edges do *not* intersect; empty rects never intersect) — but see the row below: it is **not** what gameplay collision uses |
| **Gameplay collision (not QuickDraw)** | `SectGlider` (`GliderPRO/Sources/Interactions.c:101-130`) and the inline tests in `CheckBandCollision` (`GliderPRO/Sources/RubberBands.c:72-81`, `:150-159`, `:173-182`) | hand-rolled **closed-interval** overlap (`<` / `>`, not `<=` / `>=`), so **touching edges collide and degenerate/empty rects can still register a hit**. Write this predicate yourself in Go; do **not** reuse the `SectRect` port. `SectRect` appears only in culling, room-visibility and the candle registration test in `ObjectDrawAll.c` (§3.5.2) |
| **QuickDraw RNG** | `Random()`, `qd.randSeed` | **your own PRNG** — the algorithm is not in this tree (see §2.1) |
| **GWorlds / offscreen** | `NewGWorld(..., useTempMem)` then retry with `0`, `LockPixels`, `GetGWorldPixMap`, `SetGWorld`/`GetGWorld`, `DisposeGWorld` | plain image buffers; **note that allocation failure is a real, RNG-affecting code path** in the original that a Go port will simply never hit — see §7.2 |
| **Resource Manager** | `GetPicture` (PICTs), `GetResource('bnds', id)`, `GetResource('Date', id)` (background tiles, `Room.c:272`), `GetResource('demo', 128)`, `GetIndString`, `GetCIcon`, `GetIconSuite`, `ReleaseResource`, `HLock`/`HGetState`/`HSetState` | an asset bundle keyed by (four-char type, int16 ID); PICT decoding; **big-endian** payloads |
| **Sound Manager** | `GetDefaultOutputVolume`, `SetDefaultOutputVolume` with `/0x0024` and `*0x0025` scaling (`Utilities.c:751`, `:779`), volume capped at `0x00000100`, doubled into both channels via `longVol + (longVol << 16)` (`Utilities.c:782`) | any audio backend; the 0..7 volume scale is the game's own |
| **Event Manager** | `WaitNextEvent(everyEvent, &theEvent, sleep=2, nil)` (`Play.c:393`, `Events.c:500`), `GetNextEvent`, `FlushEvents(everyEvent, 0)`, `osEvt` with suspend/resume mask `0x01000000` and resume bit `0x00000001` | a windowing event loop; **the suspend/resume path sets `switchedOut`, which blocks the sim loop** |
| **Keyboard** | `GetKeys(theKeys)` into a 128-bit `KeyMap`, tested with `BitTst(&theKeys, offset)` | a 128-bit keyboard bitmap indexed by the same virtual key offsets, or a remapping layer |
| **Time** | `TickCount()` (1/60 s since boot), `GetDateTime` (seconds since 1904), `GetTime` (calendar), `Delay(ticks, &junk)` | monotonic clock for pacing; wall clock only for the cosmetic wall clocks and timestamps |
| **Memory Manager** | `MaxApplZone`, `MoreMasters` ×4, `NewPtr`, `DisposePtr`, `HandToHand`, `MemError` | Go GC; note `CheckMemorySize` (`Environ.c:585-660`) computes a byte budget scaled by `thisMac.isDepth`, including `bytesNeeded += kDemoLength;` at `Environ.c:657` |
| **Regions** | `RgnHandle`, `AddToMirrorRegion` (`Render.c:740`), `ZeroMirrorRegion` (`Render.c:765`) | a rect list or a 1-bit mask |
| **Process** | `ExitToShell` | `os.Exit` |

### 6.3.1 Colour depth and 8-bit indexed colour

`kPreferredDepth` = **8** (`GliderPRO/Headers/Externs.h:15`) — every offscreen GWorld the
game creates is 8-bit indexed:

* `CreateOffScreenGWorld(&workSrcMap, &workSrcRect, kPreferredDepth)`
  (`GliderPRO/Sources/StructuresInit2.c:158`)
* `CreateOffScreenGWorld(&backSrcMap, &backSrcRect, kPreferredDepth)`
  (`GliderPRO/Sources/StructuresInit2.c:162`)
* `CreateOffScreenGWorld(&savedMaps[numSavedMaps].map, &mapRect, kPreferredDepth)`
  (`GliderPRO/Sources/DynamicMaps.c:82`)

`kRedOrangeColor8` = 23, with the author's own correction comment "actually, 18"
(`GliderPRO/Headers/GliderDefines.h:542`) — a raw palette index into the Mac 8-bit system
palette. A Go port must either carry the Mac 256-colour palette or resolve such indices to
RGB at asset-conversion time.

The `thisMac.isDepth == 4` (16-colour) special cases in `DynamicMaps.c` exist because at
4 bits per pixel two pixels share a byte, so odd source-`left` values would need a nibble
shift the blitter cannot do — the code nudges the rect to an even boundary instead. **A Go
port running at 8-bit or RGBA never takes these branches**, which is a one-pixel *rendering*
difference from a 16-colour Mac. It does **not** affect the simulation or the RNG stream: in
all five call sites the nudge is applied to the destination rect only, while the RNG draw is
gated on `BackUpToSavedMap(&bounds, …)` with a depth-independent `bounds` (see hazard 5 in
§7.2, corrected).

### 6.3.2 Big-endian on-disk data

Everything in a house file, a `'bnds'` resource, a `'demo'` resource and a prefs file is
**big-endian**, 2-byte aligned, with Pascal strings (leading length byte, fixed-size
buffer). Specifically:

* `short` → `int16` big-endian.
* `long` → `int32` big-endian (**not** 64-bit; verified empirically in §4.3).
* `Point` → `{v int16; h int16}`, **v first**.
* `Rect` → `{top, left, bottom, right int16}`.
* `Str15`/`Str27`/`Str31`/`Str32`/`Str255` → 1 length byte + N bytes, total N+1 (so `Str27`
  occupies 28 bytes, `Str255` occupies 256).
* `Boolean` → 1 byte, non-zero = true.
* Text encoding is **Mac Roman**, not UTF-8 or Latin-1.

## 6.4 Blocking and re-entrancy hazards in the main loop

| Site | Behaviour | Network impact |
| --- | --- | --- |
| `GliderPRO/Sources/Play.c:437-443` | `if (doBackground) { do { HandlePlayEvent(); } while (switchedOut); }` — spins the OS event loop until the app is foregrounded again | **a peer that loses focus stops simulating entirely** |
| `GliderPRO/Sources/Play.c:387-426` (`HandlePlayEvent`) | `WaitNextEvent` with `sleep = 2` ticks; handles update events by redrawing; `osEvt` toggles `switchedOut` | |
| `GliderPRO/Sources/Input.c:77-117` (`DoPause`) | blocking `GetKeys` spin (`do { GetKeys(theKeys); } while (…)` `:111-116`) | **a local pause stalls only one peer** |
| `GliderPRO/Sources/Render.c:662-664` | `while (TickCount() < nextFrame) { }` busy-wait | burns CPU; harmless for correctness |
| `GliderPRO/Sources/Utilities.c:485-499` (`WaitCommandQReleased`) | blocking `GetKeys` spin | quit path only |
| `GliderPRO/Sources/Utilities.c:439-479` (`WaitForInputEvent`) | blocking with a `TickCount` timeout | UI only |
| `GliderPRO/Sources/Events.c:479-541` (`HandleEvent`) | the outer non-gameplay loop; `long sleep = 2;` at `:483`, passed to `WaitNextEvent` at `:500` (with a `GetNextEvent` fallback at `:504` when `!thisMac.hasWNE`); Cmd+Opt / Opt handling at `:486-497` | |
| `GliderPRO/Sources/Events.c:390-442` (`HandleOSEvent`) | suspend/resume: on resume, re-checks the colour depth against `thisMac.isDepth`, may call `BitchAboutColorDepth()`, restarts music, and resets `incrementModeTime` at `:427` | a depth change mid-game alters which `isDepth == 4` rendering branches run. Re-verified: that is **cosmetic only** — those branches never gate an RNG draw (hazard 5) |


---

# Part 7 — Determinism verdict

## 7.1 The question, restated precisely

Two questions must be separated:

* **Q1 (replay determinism):** on *one* machine with *one* build, does replaying the same
  input stream from the same start state produce the same simulation?
* **Q2 (lockstep determinism):** on *two different* machines, given an identical RNG seed
  and identical input streams, does the simulation evolve identically?

**Answer to Q1: yes, and the game ships proof of it** — the `'demo' 128` resource is a
recorded input stream that replays correctly (Part 8).

**Answer to Q2: no, not as written.** The simulation is *structurally* deterministic —
integer-only, fixed-timestep, no floating point, no delta-time, no hash-order iteration —
but it has a small number of specific inputs from outside the simulation that differ between
machines. Every one of them is enumerated below. All are fixable in a port; none is
fundamental.

## 7.2 Hazard table

Severity key: **BLOCKER** = breaks lockstep and cannot be papered over;
**FIXABLE** = breaks lockstep as written but has a clean deterministic replacement;
**BENIGN** = looks dangerous, actually is not.

| # | Hazard | Severity | Evidence | Effect | Fix in a Go port |
| --- | --- | --- | --- | --- | --- |
| 1 | `Random()` is an OS trap; its algorithm is not in this source tree | **BLOCKER** for byte-exact emulation, trivial for a port | `GliderPRO/Sources/Utilities.c:76` | cannot reproduce the original's exact sequence | define your own PRNG (see §10.3); accept that sequences differ from 1994 |
| 2 | Seeded from wall clock, once per launch | FIXABLE | `GliderPRO/Sources/Utilities.c:61` `GetDateTime((UInt32 *)&qd.randSeed)` | two machines always start with different streams | seed per match from a value both peers agree on |
| 3 | `numNeighbors` preference (1 or 9) changes how many rooms `DrawLocale` draws, and neighbour rooms consume RNG | **FIXABLE, and the worst one** | `GliderPRO/Sources/RoomGraphics.c:78` (`numNeighbors > 3`), `:105` (`> 1`), `:123` (`> 3`, floor support); `GliderPRO/Sources/Main.c:111`, `:154` | different draw counts per room load → permanent stream divergence | never consume gameplay RNG from a rendering path; see §10.3 |
| 4 | `numNeighbors` forced to 1 when `thisMac.screen.right <= 512` | FIXABLE | `GliderPRO/Sources/Main.c:191-192` | screen size silently changes RNG consumption | same as 3 |
| 5 | `thisMac.isDepth == 4` nudges the *destination* rect of a registered object by one pixel (odd `h` → `h-1`, clamped back up by `+2` if negative) | **BENIGN** — re-verified: it does **not** change RNG consumption | `GliderPRO/Sources/DynamicMaps.c:326` (`AddCandleFlame`), `:409` (`AddTikiFlame`), `:495` (`AddBBQCoals`), `:584` (`AddPendulum`), `:671` (`AddStar`) | render position only. In all five functions the RNG draw (`:338`, `:422`, `:508`, `:594`, `:685`) is gated solely on `BackUpToSavedMap(&bounds, …) != -1`, and `bounds` is a depth-independent `QSetRect(&bounds, 0, 0, w, h)` — never the shifted rect | replicate the shift for pixel-exact rendering, or drop the 4-bit path; either way the stream is unaffected |
| 6 | `BackUpToSavedMap` returning -1 (24-slot cap **only**) suppresses the `RandomInt` draw in `AddCandleFlame` | FIXABLE | cap check `GliderPRO/Sources/DynamicMaps.c:75-76` `if (numSavedMaps >= kMaxSavedMaps) return(-1);`; success `return (numSavedMaps - 1);` at `:92`; guarded draw at `:338` | slot exhaustion affects the RNG stream. Note: **allocation failure does *not* — `theErr` from `CreateOffScreenGWorld` (`:82`) is assigned and then discarded**, so a failed GWorld still returns a valid index | make the draw unconditional, or move it before the registration |
| 7 | Per-class caps silently drop registrations: `kMaxCandles` 20, `kMaxTikis` 8, `kMaxCoals` 8, `kMaxPendulums` 8, `kMaxStars` 4, `kMaxDynamicObs` 18, `kMaxSavedMaps` 24, `kMaxHotSpots` 56, `kMaxTriggers` 16, `kMaxGrease` 16, `kMaxSparkles` 3, `kMaxFlyingPts` 3, `kMaxShredded` 4, `kMaxRubberBands` 2, `kMaxMasterObjects` 216 | BENIGN (deterministic) but must be replicated exactly | `GliderPRO/Headers/GliderDefines.h:249-266`, `GliderPRO/Sources/Triggers.c:12` | a port with larger caps diverges from the original | keep the exact caps |
| 8 | One shared RNG stream mixes gameplay timers, cosmetics, splash screen and game-over animation | FIXABLE | §2.5 enumeration | a cosmetic draw shifts a gameplay draw | split into named streams (§10.3) |
| 9 | Wall-clock `GetTime()` drives the analog clock hand sprites | BENIGN (cosmetic only) | `GliderPRO/Sources/ObjectDraw.c:970`, `:1012`, `:1033`, `:1054`; `GliderPRO/Sources/ObjectDraw2.c:1160` | clock faces differ between peers; nothing reads them back | render-only; keep out of the sim |
| 10 | `TickCount()` frame pacing | **BENIGN** | `GliderPRO/Sources/Render.c:662-665` | `nextFrame = TickCount() + kTicksPerFrame` is an **assignment**, not `+=`, so a slow machine runs *slower in real time* but executes the *same number of steps* per input event | replace with your own pacing; step count is unaffected |
| 11 | `switchedOut` blocks the whole loop while the app is in the background | FIXABLE | `GliderPRO/Sources/Play.c:437-443` | a peer that loses focus stops simulating | never block the sim on focus in a networked build |
| 12 | `DoPause()` blocks locally | FIXABLE | `GliderPRO/Sources/Input.c:77-118` | one peer pauses, the other does not | pause must be a networked, agreed-upon state |
| 13 | Render pass mutates simulation state | FIXABLE (structural) | `GliderPRO/Sources/Render.c:639-671`; §3.9 mutation table | you cannot skip rendering on a headless peer without changing the sim | split sim/render (§10.1) |
| 14 | The extra `RenderFrame()` inside room transitions | BENIGN but must be replicated | `GliderPRO/Sources/Transit.c:303`, `:341`, `:380`, `:419` (the only four `RenderFrame()` calls in the file) | one extra animation step per transition | keep it, or move animation into the sim step |
| 15 | `HandleBall` writes the *global* `evenFrame = true` | BENIGN but must be replicated | `GliderPRO/Sources/Dynamics2.c:420` | a ball can flip the global parity mid-frame, affecting every later entity in the same frame | replicate exactly, including the ordering |
| 16 | `AddDynamicObject` writes the global `evenFrame = true` for `kBall` and `kFish` | BENIGN but must be replicated | `GliderPRO/Sources/Dynamics3.c:474`, `:524` | room load perturbs the parity | replicate exactly |
| 17 | `DidBandHitDynamic` returns an **uninitialized local** when `numBands == 0` | BENIGN in the shipped build; a real UB bug | `GliderPRO/Sources/Dynamics.c:84`, `:105`; guarded at `GliderPRO/Sources/Dynamics2.c:62`, `:162`, `:267` | all three call sites pre-check `numBands > 0`, so the garbage is never read | in Go, `collided` defaults to `false` — matches the guarded behaviour |
| 18 | `FireTrigger`'s else-branch indexes `masterObjects[-1]` | BENIGN-ish; a real out-of-bounds read | `GliderPRO/Sources/Triggers.c:178`, `:187-188` | that branch is only reached when `localLink == -1`, then `triggeredIs = -1` and `masterObjects[-1].dynaNum` / `.hotNum` are read | in Go this panics; guard it and treat as "no grease to spill" |
| 19 | Swap-with-last removal reorders live entities | BENIGN (deterministic) but order-sensitive | `KillBand` `GliderPRO/Sources/RubberBands.c:294-303`; `RemoveShreds` `GliderPRO/Sources/DynamicMaps.c:748` | index identity is not stable across removals | replicate the swap-with-last exactly; do not use a stable-order slice delete |
| 20 | `FindEmptyTriggerSlot` returns the lowest free index | BENIGN (deterministic) | `GliderPRO/Sources/Triggers.c:59-75` | slot reuse order is load-bearing for `HandleTriggers` iteration | replicate lowest-free-index |
| 21 | `AddSparkle` uses the first slot with `mode == -1` | BENIGN (deterministic) | `GliderPRO/Sources/DynamicMaps.c:169-193` | same as 20 | replicate |
| 22 | `bandHitLast` is a single global shared by both rubber bands | BENIGN (deterministic) but a coupling | `GliderPRO/Sources/RubberBands.c:26`, `:86-88` | band A's last hit suppresses band B's re-hit | replicate as a global, not per-band |
| 23 | `thisRoom` is a **by-value 348-byte copy**, not a pointer into the house | FIXABLE (correctness) | `GliderPRO/Sources/Room.c:369-386` | writes to `thisRoom` are lost on transition unless mirrored into the house handle | mirror both, exactly as `HandleRoomVisitation` does |
| 24 | `SetObjectsToDefaults` mutates the loaded house handle | FIXABLE (structural) | `GliderPRO/Sources/Play.c:603-708` | the "asset" is also mutable game state | deep-copy the house on game start |
| 25 | Object `state` flags and `visited` flags live in the house and affect `GetNumberOfLights`, hence RNG consumption at room load | FIXABLE | `GliderPRO/Sources/Room.c:970-1099` | RNG consumption depends on gameplay history | same as 3 — decouple RNG from rendering |
| 26 | `kDart` branch of `AddDynamicObject` never sets `.room` | BENIGN in practice; latent | `GliderPRO/Sources/Dynamics3.c:437-463` | `.room` keeps whatever `ZeroDinahs` left (0), unlike every other type | replicate the omission or fix it — but pick one and document it |
| 27 | `ZeroDinahs` does not clear `byte1` or `moving` | BENIGN-ish; latent | `GliderPRO/Sources/Dynamics3.c:160-180` | stale values survive a room change for slots that a new room does not re-initialise | Go zeroes the whole struct; note the (harmless) divergence |
| 28 | `AddPendulum` sets the global `clockFrame = 10` | BENIGN but must be replicated | `GliderPRO/Sources/DynamicMaps.c:578` | adding one clock resets the phase of all clocks in the room | replicate |
| 29 | `RandomInt(range)` can return `range` itself (inclusive upper bound) | BENIGN but must be replicated exactly | `GliderPRO/Sources/Utilities.c:79`; §2.1 empirical | `RandomInt(kNumCandleFlames)` can return 5 for a 5-element table; `RandomInt(6)` can return 6 for stars | replicate the formula, not `rand.Intn` |
| 30 | `RandomInt` is not uniform | BENIGN but must be replicated | §2.1 empirical: `RandomInt(2)` gives 32767 / 32768 / 1 | | replicate the formula |
| 31 | `HandleDynamicScoreboard` keys off `gameFrame & 0x00000007` | BENIGN (deterministic) | `GliderPRO/Sources/Scoreboard.c:95` | scoreboard refresh phase is tied to absolute frame number | replicate; note cases 3 and 6 are unused |
| 32 | 68k/PPC `short` overflow in the glider velocity integrator | BENIGN but must be replicated | `GliderPRO/Sources/Player.c:64-146`; vertical velocity is **not clamped** | a long fall can in principle overflow `short` | use `int16` arithmetic in Go, not `int` |
| 33 | Sparse-key demo format only records *changes*, and `GetDemoInput` clears `heldLeft`/`heldRight` every frame for player 2 | BENIGN | `GliderPRO/Sources/Input.c:186-278` | | see §8.3 |
| 34 | Sound playback is fire-and-forget with priorities; no gameplay reads sound state | BENIGN | `PlayPrioritySound(sound, priority)` throughout | sound divergence cannot desync the sim | keep sound out of the sim |

## 7.3 Verdict

> **The Glider PRO simulation is deterministic in structure and non-deterministic in
> practice.** Every source of non-determinism is an *input from outside the simulation*
> (wall-clock seed, user preferences, screen size, colour depth, free memory, window focus)
> rather than anything intrinsic to the physics. There is no floating point, no
> hash-map iteration, no pointer-value-dependent ordering, no threading, and no
> accumulating delta-time. The per-frame update order is a fixed, hand-written sequence
> (§3.1).
>
> With the RNG seeded identically **and** the RNG decoupled from the rendering path
> (hazards 3, 4, 6, 8, 25 — hazard 5, colour depth, turns out **not** to affect the
> stream), a Go port is fully lockstep-deterministic.

The single most important structural change is hazard 3/25: **in the original, how much
randomness a room consumes depends on how many neighbour rooms you draw and how they are
lit.** That is a rendering concern leaking into the simulation. Fixing it is not optional
for lockstep, and it is not hard (§10.3).

## 7.4 What is *already* right, and must not be "improved"

A porter's instinct will be to modernise these. Do not.

| Property | Where | Why it must stay |
| --- | --- | --- |
| Integer-only physics | `GliderPRO/Sources/Player.c:64-147` (`MoveGlider`) and all of `Dynamics*.c` | no float determinism problems at all; converting to float breaks lockstep for free |
| One sim step per frame, no delta-time | `GliderPRO/Sources/Play.c:432-554` (`while ((playing) && (!quitting))` at `:432`, closing brace `:554`) | frame-rate independence is *not* wanted; the physics constants are per-frame |
| `nextFrame` **assigned**, not accumulated | `GliderPRO/Sources/Render.c:665` | prevents catch-up loops, so a slow peer executes the same number of steps |
| `evenFrame` half-rate gating | `GliderPRO/Sources/Play.c:435` and ~40 read sites | the "30 Hz" entities are a deliberate design, not a bug |
| Fixed hand-written update order | §3.1 | any reordering changes behaviour |
| Swap-with-last removal | `GliderPRO/Sources/RubberBands.c:294`, `GliderPRO/Sources/DynamicMaps.c:748` | order-visible; a stable delete changes behaviour |
| Exact array caps | `GliderPRO/Headers/GliderDefines.h:249-266` (`:267-268` are `kMaxViewWidth`/`kMaxViewHeight`, not entity caps) | overflow-drop behaviour is observable |


---

# Part 8 — The demo replay system: shipped proof of input-determinism

Glider PRO ships a recorded input stream and replays it as an attract-mode demo. This is
the strongest available evidence that the simulation is replay-deterministic, and it is
also a complete worked example of an input-stream protocol — exactly what a lockstep port
needs. It is worth studying closely, including its bugs.

## 8.1 The record format

`GliderPRO/Headers/GliderStructs.h:334-339`:

```c
typedef struct
{
	long		frame;      // 4 bytes, big-endian
	char		key;        // 1 byte
	char		padding;    // 1 byte  -- NEVER WRITTEN
} demoType, *demoPtr;       // total = 6
```

| Field | Type | Offset | Size |
| --- | --- | --- | --- |
| `frame` | `long` | 0 | 4 |
| `key` | `char` | 4 | 1 |
| `padding` | `char` | 5 | 1 |
| | | | **6 total** |

`kDemoLength` = **6702** (`GliderPRO/Headers/GliderDefines.h:625`).
6702 / 6 = **1117 records exactly**.

## 8.2 Loading

`GliderPRO/Sources/StructuresInit2.c:280-297`:

```c
#ifdef CREATEDEMODATA
	demoData = nil;
	demoData = (demoPtr)NewPtr(sizeof(demoType) * 2000);        // :282  recording buffer
	if (demoData == nil) ...
#else
	demoData = nil;
	demoData = (demoPtr)NewPtr(kDemoLength);                    // :287
	if (demoData == nil) ...
	else
	{
		tempHandle = GetResource('demo', 128);
		...
		BlockMove(*tempHandle, demoData, kDemoLength);           // :295
		ReleaseResource(tempHandle);
	}
#endif
```

`CREATEDEMODATA` is **not** defined (`GliderPRO/Headers/GliderDefines.h:11`), so the
shipped build is replay-only. The recording build dumps the buffer via
`DumpToResEditFile((Ptr)demoData, sizeof(demoType) * (long)demoIndex)`
(`GliderPRO/Sources/Play.c:217`).

`CheckMemorySize` adds `bytesNeeded += kDemoLength;` at `GliderPRO/Sources/Environ.c:657`.

## 8.3 Recording — `LogDemoKey`

`GliderPRO/Sources/Input.c:44-49`:

```c
void LogDemoKey (char keyIs)
{
	demoData[demoIndex].frame = gameFrame;   // :46
	demoData[demoIndex].key = keyIs;         // :47
	demoIndex++;                             // :48
}
```

**`padding` is never assigned** — it retains whatever was in the freshly-`NewPtr`'d buffer.

The four call sites, all inside `GetInput` and all guarded by `#ifdef CREATEDEMODATA`:

| Call | Line | Emitted when |
| --- | --- | --- |
| `LogDemoKey(0)` | `GliderPRO/Sources/Input.c:304` | the **right** key is down (`thisGlider->rightKey`) |
| `LogDemoKey(1)` | `GliderPRO/Sources/Input.c:321` | the right key is up and the **left** key is down |
| `LogDemoKey(2)` | `GliderPRO/Sources/Input.c:334` | `battKey` down **and** `batteryTotal != 0` **and** `mode == kGliderNormal` |
| `LogDemoKey(3)` | `GliderPRO/Sources/Input.c:348` | `bandKey` down **and** `bandsTotal > 0` **and** `mode == kGliderNormal` |

> **Note the numbering is the opposite of the source comments.** `GetDemoInput` labels
> `case 0` as "left key" (`GliderPRO/Sources/Input.c:228`) and `case 1` as "right key"
> (`:235`), but `case 0` performs `hDesiredVel += kNormalThrust` and sets
> `heldRight = true`, which is rightward motion, and `LogDemoKey(0)` is emitted from the
> **right**-key branch. The comments are wrong; the behaviour is self-consistent:
> **0 = right, 1 = left.**

Because a single frame can hit up to three of these branches (a direction, plus battery,
plus band), the recorder can emit **multiple records with the same `frame` value**.

## 8.4 Replay — `GetDemoInput`

`GliderPRO/Sources/Input.c:186-278`:

```
GetDemoInput(thisGlider):
   1. if thisGlider->which == kPlayer1 then
   2.     GetKeys(theKeys)
   3.     #if BUILD_ARCADE_VERSION                                    // Input.c:192
   4.         if any of leftKey / rightKey / battKey / bandKey is down then
   5.             playing <- false ; paused <- false                   // Input.c:199-200
   6.     #else
   7.         if BitTst(&theKeys, kCommandKeyMap) then DoCommandKey()
   8.     #endif
   9. if thisGlider->mode == kGliderBurning then
  10.     hDesiredVel -= kNormalThrust if facing left, else += kNormalThrust
  11.     return                                                       // burning ignores the demo stream
  12. heldLeft <- false ; heldRight <- false ; tipped <- false          // Input.c:220-222
  13. if gameFrame == (long)demoData[demoIndex].frame then              // Input.c:224
  14.     switch demoData[demoIndex].key:                              // Input.c:226
  15.       case 0:  hDesiredVel += kNormalThrust
  16.                tipped   <- (facing == kFaceLeft)
  17.                heldRight<- true
  18.                fireHeld <- false
  19.       case 1:  hDesiredVel -= kNormalThrust
  20.                tipped   <- (facing == kFaceRight)
  21.                heldLeft <- true
  22.                fireHeld <- false
  23.       case 2:  if batteryTotal > 0 then DoBatteryEngaged(thisGlider)
  24.                else                    DoHeliumEngaged(thisGlider)
  25.                fireHeld <- false
  26.       case 3:  if !fireHeld then
  27.                    if AddBand(thisGlider, dest.left + 24, dest.top + 10, facing) then
  28.                        bandsTotal--
  29.                        if bandsTotal <= 0 then QuickBandsRefresh(false)
  30.                        fireHeld <- true
  31.     demoIndex++                                                   // Input.c:266
  32. else
  33.     fireHeld <- false
  34. if (isEscPauseKey and Esc down) or (!isEscPauseKey and Tab down) then DoPause()
```

`kNormalThrust` = 5, `kHyperThrust` = 8, `kHeliumLift` = 4
(`GliderPRO/Sources/Input.c:14-16`).

### 8.4.1 Four record/replay asymmetries (latent bugs)

| # | Asymmetry | Evidence | Consequence |
| --- | --- | --- | --- |
| 1 | `demoIndex++` runs **once** per frame, inside an `if`, not a `while` | `GliderPRO/Sources/Input.c:224`, `:266` | if the recording contains two records for the same frame, the second is stranded and **every subsequent record replays one frame late or never** |
| 2 | Both-keys-held about-face is not representable | recorder emits `LogDemoKey(0)` then calls `ToggleGliderFacing` at `GliderPRO/Sources/Input.c:308`; replay `case 0` just thrusts right | an about-face during recording replays as a plain right thrust |
| 3 | `case 2` omits the recorder's `batteryTotal != 0` and `mode == kGliderNormal` guards | recorder `GliderPRO/Sources/Input.c:332-333` vs replay `:242-249` | replay can call `DoHeliumEngaged` when `batteryTotal == 0`, and can fire the battery in a non-normal glider mode |
| 4 | `case 3` omits the recorder's `bandsTotal > 0` and `mode == kGliderNormal` guards | recorder `GliderPRO/Sources/Input.c:346-347` vs replay `:250-265` | replay can attempt `AddBand` with zero bands; `bandsTotal` can go negative |
| 5 | `tipped <- false` is unconditional in replay (`:222`) but only in the `else` branch of the recorder (`:328`) | | a frame with a battery-only record clears `tipped` in replay but not in recording |

Also: `GetDemoInput` never runs for player 2 — `PlayGame` only calls it in the
single-player branch (`GliderPRO/Sources/Play.c:479`), and `demoGoing` is only set by
`DoDemoGame` (`GliderPRO/Sources/Play.c:294`), which is single-player.

These asymmetries did not bite in practice because the shipped demo happens to avoid the
problem cases (see §8.5). **A Go port must not copy this format.** It is a lossy,
one-action-per-frame encoding. Use a per-frame bitmask instead (§10.4).

## 8.5 Empirical parse of the shipped `'demo' 128` resource

Extracted from the `data 'demo' (128)` block of `GliderPRO/Glider PRO.r` and parsed as
1117 big-endian `demoType` records. Observed:

```
demo resource: 6702 bytes  (kDemoLength = 6702)  match=True
entries of sizeof(demoType)=6: 1117  (6702/6 = 1117.0)
frames non-decreasing: True ; min=46 max=3414 ; distinct keys=[0, 1, 3]
frames STRICTLY increasing: True
duplicate-frame records: 0
key distribution: Counter({0: 910, 1: 198, 3: 9})
frame-delta histogram (top 8): [(1, 1009), (7, 8), (4, 6), (8, 6), (3, 6), (10, 6), (14, 4), (13, 4)]
total sim frames covered: 46..3414
```

First and last records, showing the uninitialised `padding`:

```
  [   0] frame=46     key=0 pad=114
  [   1] frame=47     key=0 pad=114
  [   2] frame=48     key=0 pad=114
  [   3] frame=49     key=0 pad=69
  [   4] frame=56     key=0 pad=6
  [   5] frame=57     key=0 pad=114
  [   6] frame=58     key=0 pad=111
  [   7] frame=59     key=0 pad=114
  [   8] frame=60     key=0 pad=106
  [   9] frame=74     key=0 pad=97
  ...
  [1112] frame=3410   key=0 pad=-1
  [1113] frame=3411   key=0 pad=-2
  [1114] frame=3412   key=0 pad=-8
  [1115] frame=3413   key=0 pad=7
  [1116] frame=3414   key=0 pad=1
```

Findings, all directly observed:

1. **Exactly 1117 records**, so `kDemoLength` = 6702 is `1117 * 6` with no slack — the
   resource is exactly full.
2. **Frames are strictly increasing** with **zero duplicates**, which is precisely why
   asymmetry #1 in §8.4.1 never fires in the shipped data.
3. **Keys observed: only 0 (right, 910×), 1 (left, 198×), 3 (band, 9×).** Key 2
   (battery/helium) **never appears** — the demo run never used a battery, so
   asymmetry #3 is never exercised either.
4. **1009 of 1116 frame deltas are 1**, i.e. the recording is dominated by continuously
   held direction keys. The format records the *held state every frame*, not edges, so
   holding "right" for 100 frames costs 100 records (600 bytes). This is a very
   inefficient encoding.
5. The demo covers game frames **46 through 3414** — 3369 frames, about 112 seconds at
   `kTicksPerFrame` = 2 (30 fps). Frames 0..45 have no records: the glider is fading in
   (`StartGliderFadingIn`, `GliderPRO/Sources/Play.c:185`) and idle.
6. **`padding` is garbage.** The observed values 114, 111, 97, 32, 104, 116 are the ASCII
   codes for `r`, `o`, `a`, space, `h`, `t` — leftover text from whatever occupied that
   heap block before `NewPtr`. The tail values (-1, -2, -8, 7, 1) are unrelated bytes.
   This confirms `LogDemoKey` never writes the field, and confirms the record is 6 bytes
   with mac68k 2-byte alignment (no padding *between* fields — a 4-byte `long` followed by
   two `char`s, total 6, not 8).
7. Because the padding is uninitialised heap, **the shipped resource is not byte-reproducible**
   even from the same gameplay. Only bytes 0..4 of each record are meaningful.

## 8.6 How the demo is launched

`DoDemoGame()` (`GliderPRO/Sources/Play.c:282-303`):

```
DoDemoGame():
   1. save the current house index
   2. thisHouseIndex <- demoHouseIndex                          // Play.c:289
   3. (open that house)
   4. demoGoing <- true                                         // Play.c:294
   5. NewGame(kNewGameMode)
   6. restore the original house index
   7. demoGoing <- false                                        // Play.c:276 (inside NewGame's tail)
   8. incrementModeTime <- TickCount() + kIdleSplashTicks        // Play.c:302
```

`demoHouseIndex` is found by `BuildHouseList`-time scanning in
`GliderPRO/Sources/SelectHouse.c:636-641` (initialised to -1, set to the matching index).
`Menu.c:88` disables the demo menu item when `demoHouseIndex == -1`.

The trigger is in the idle path:
`if ((theMode == kSplashMode) && doAutoDemo && !switchedOut) { if (TickCount() >= incrementModeTime) DoDemoGame(); }`
(`GliderPRO/Sources/Events.c:536-540`), with `kIdleSplashTicks` = **7200L** = 2 minutes
(`GliderPRO/Headers/GliderDefines.h:197`).

## 8.7 What the demo proves, and what it does not

**Proves:**

* The simulation is a pure function of (start state, per-frame input) on one machine and
  one build. 3369 frames of physics, dynamics, triggers, bands and room transitions replay
  correctly from 1117 sparse input records.
* `gameFrame` is a reliable, monotone logical clock that both a recorder and a replayer can
  agree on — the natural sequence number for a lockstep protocol.
* Wall-clock time does not feed the simulation: the demo replays identically regardless of
  when it is played, despite `qd.randSeed` being reseeded from the clock at every launch.

**Does not prove:**

* Cross-machine determinism. The demo runs on the machine that plays it, with that
  machine's `numNeighbors`, screen size and colour depth. Nothing compares two machines.
* That the RNG stream is irrelevant. It is not: the demo run's random draws (candle modes,
  telephone timers, coffee timers, sparkle re-arms) differ on every playback because the
  seed is `GetDateTime`. **The demo replays correctly *in spite of* a different RNG stream
  each time**, which tells us something important: in this house, over these 3369 frames,
  no RNG-driven event altered the glider's trajectory. That is a property of the demo
  house, not a general guarantee — `HandleCoffee`'s `RandomInt(200)` timer
  (`GliderPRO/Sources/Dynamics.c:492`, `:506`) and the enemy respawn timers absolutely can
  change gameplay outcomes elsewhere.

That last point is the single most useful empirical observation in this document: **the
shipped demo is evidence that input-determinism holds, and simultaneously evidence that
RNG-determinism does *not* hold, in the original.**


---

# Part 9 — The original's two-player model

Glider PRO already has a two-player mode. It is **shared-keyboard, single-process, single
simulation** — but its *rules* are the ones a networked port must reproduce, and they were
clearly designed with a "shared fate" philosophy. This part documents the mechanism as a
whole; the escape/limbo handshake state machine itself is in §3.5.4 and is not repeated.

## 9.1 The two gliders

`theGlider` and `theGlider2` are two separate `gliderType` globals, defined at
`GliderPRO/Sources/Player.c:43` (`gliderType theGlider, theGlider2;`) and declared
`extern` at `GliderPRO/Headers/GliderVars.h:46` — **not** in `Play.c`, as an earlier draft
said. `gliderType.which` is a `Boolean`
(`GliderPRO/Headers/GliderStructs.h:213`) with:

| Constant | Value | Location |
| --- | --- | --- |
| `kPlayer1` | `TRUE` (1) | `GliderPRO/Headers/GliderDefines.h:556` |
| `kPlayer2` | `FALSE` (0) | `GliderPRO/Headers/GliderDefines.h:557` |

**Note the inversion**: player 1 is `true`. Any Go port using `Which int` with 0/1 must be
careful — the source compares `playerDead == theGlider.which` and
`thisGlider->which != firstPlayer` all over `Dynamics*.c` and `Interactions.c`.

## 9.2 Key bindings

Player 1's keys are stored as four `long` KeyMap offsets in the glider record and are
**user-remappable** via preferences:

| Field | Default | Constant | Value | Location |
| --- | --- | --- | --- | --- |
| `theGlider.leftKey` | left arrow | `kLeftArrowKeyMap` | 124 | `GliderPRO/Sources/Main.c:135`, `GliderPRO/Headers/Externs.h:132` |
| `theGlider.rightKey` | right arrow | `kRightArrowKeyMap` | 123 | `GliderPRO/Sources/Main.c:136`, `Externs.h:131` |
| `theGlider.battKey` | down arrow | `kDownArrowKeyMap` | 122 | `GliderPRO/Sources/Main.c:137`, `Externs.h:130` |
| `theGlider.bandKey` | up arrow | `kUpArrowKeyMap` | 121 | `GliderPRO/Sources/Main.c:138`, `Externs.h:129` |

Loaded from prefs at `GliderPRO/Sources/Main.c:69-72`, written back at
`GliderPRO/Sources/Main.c:225+`, reset to defaults at
`GliderPRO/Sources/Settings.c:1237-1240`.

Player 2's keys are **hard-coded modifier keys and cannot be remapped**
(`GliderPRO/Sources/InterfaceInit.c:148-151`):

| Field | Key | Constant | Value |
| --- | --- | --- | --- |
| `theGlider2.leftKey` | Control | `kControlKeyMap` | 60 |
| `theGlider2.rightKey` | Command | `kCommandKeyMap` | 48 |
| `theGlider2.battKey` | Option | `kOptionKeyMap` | 61 |
| `theGlider2.bandKey` | Shift | `kShiftKeyMap` | 63 |

**Player 2's "right" key is Command**, which is also the game's own command-key modifier.
`GetInput` only tests `kCommandKeyMap` for `DoCommandKey()` when
`thisGlider->which == kPlayer1` (`GliderPRO/Sources/Input.c:283-287`), so pressing player
2's right key does not trigger Cmd-Q — but `DoCommandKey` itself
(`GliderPRO/Sources/Input.c:53-73`) is reached from player 1's pass on the *same* frame if
player 1 also holds Command. Both `DoCommandKey` branches are further gated on
`!twoPlayerGame` for the save paths, so in a two-player game only Cmd-Q (quit) is live.

## 9.3 Input collection is per-glider but the KeyMap is global

`GliderPRO/Sources/Input.c:281-287`:

```c
void GetInput (gliderPtr thisGlider)
{
	if (thisGlider->which == kPlayer1)
	{
		GetKeys(theKeys);                             // Input.c:285  ONE global snapshot
		if (BitTst(&theKeys, kCommandKeyMap))
			DoCommandKey();
	}
	...
	if (BitTst(&theKeys, thisGlider->rightKey))       // reads the SAME theKeys
```

`theKeys` is a single global `KeyMap` (`GliderPRO/Sources/Input.c:31`). It is refreshed
**only** on player 1's call. `PlayGame` calls `GetInput(&theGlider)` then
`GetInput(&theGlider2)` (`GliderPRO/Sources/Play.c:452-453`), so both players read the same
128-bit snapshot taken at the top of the frame. **This is the natural seam for a network
port**: replace the single `GetKeys` with two independently-sourced input records.

## 9.4 Life sharing

`mortals` is a **single global counter shared by both players**, not per-player:

* `InitGlider` new-game: `mortals = kInitialGliders (2);`
  (`GliderPRO/Sources/Play.c:341`; `kInitialGliders` is `#define`d as 2 at `Play.c:19`), then
  `if (twoPlayerGame) mortals += kInitialGliders;` (`GliderPRO/Sources/Play.c:342-343`)
  → **4 total lives** shared between the two players.
* `OffAMortal` decrements the shared counter (`GliderPRO/Sources/Player.c:1492`).

So one player burning through lives directly costs the other. This is the core of
"shared fate".

## 9.5 The death cascade

`OffAMortal(thisGlider)` (`GliderPRO/Sources/Player.c:1484-1604`):

```
OffAMortal(thisGlider):
   1. if gameOver then return                                    // Player.c:1486-1487
   2. if numShredded > 0 then RemoveShreds()                      // Player.c:1489-1490
   3. mortals--                                                   // Player.c:1492
   4. if twoPlayerGame then
   5.     if mortals < -1 then FlagGameOver()                     // Player.c:1500
   6.     else
   7.         FlagGliderInLimbo(thisGlider, false)                // Player.c:1505
   8.         thisGlider->dontDraw <- true                         // Player.c:1506
   9.         onePlayerLeft <- true                                // Player.c:1507
  10.         playerDead    <- thisGlider->which                   // Player.c:1508
  11. else
  12.     if mortals < 0 then ... else FlagGameOver()              // Player.c:1513
  13. ... respawn, or dispatch on otherPlayerEscaped, ending with
  14.     otherPlayerEscaped <- kPlayerIsDeadForever (-69)
```

Note the asymmetric thresholds: **single player dies out at `mortals < 0`, two players at
`mortals < -1`** — i.e. two players get one extra "free" death because both can be dead
simultaneously.

Once `onePlayerLeft` is true and `playerDead` identifies the corpse, **roughly forty guard
clauses** across the codebase suppress interaction for the dead player. The exhaustive
list of `(twoPlayerGame) && (onePlayerLeft)` / `playerDead == ...` guard sites:

| File | Lines |
| --- | --- |
| `GliderPRO/Sources/Dynamics.c` | 335, 337, 536, 538 |
| `GliderPRO/Sources/Dynamics2.c` | 46, 48, 147, 149, 252, 254, 362, 364, 437, 439, 511, 513 |
| `GliderPRO/Sources/Input.c` | 368 |
| `GliderPRO/Sources/Interactions.c` | 704, 717, 733, 746, 842, 864, 883, 911, 970, 1259, 1294, 1392, 1438, 1481, 1521, 1556, 1642, 1644, 1660, 1662, 1696, 1698 |
| `GliderPRO/Sources/Modes.c` | 474 (`UndoGliderLimbo`), 499 (`InsureGliderFacingRight`), 510 (`InsureGliderFacingLeft`), 523 (`ReadyGliderForTripUpStairs`), 554 (`ReadyGliderForTripDownStairs`), 633 (`TagGliderIdle`) |
| `GliderPRO/Sources/Player.c` | 1443+ (`OffsetGlider` early return) |
| `GliderPRO/Sources/Transit.c` | 69-70 (`ReadyGliderFromTransit`) |

**A port that forgets any of these will have a ghost glider interacting with the world.**
The idiom is always the same:

```c
if ((twoPlayerGame) && (onePlayerLeft) && (thisGlider->which == playerDead))
	return;
```

## 9.6 The limbo / idle / follow primitives

`FlagGliderInLimbo(thisGlider, sayIt)` (`GliderPRO/Sources/Modes.c:458-468`):

```
   1. thisGlider->wasMode <- thisGlider->mode
   2. thisGlider->mode    <- kGliderInLimbo (21)
   3. if sayIt and (saidFollow < 3) then
   4.     PlayPrioritySound(kFollowSound (7), kFollowPriority (904))
   5.     saidFollow++
   6. firstPlayer <- thisGlider->which
```

`saidFollow` (`GliderPRO/Sources/Modes.c:13`) caps the "follow me!" voice sample at **3
plays per game**; reset to 0 in `NewGame` (`GliderPRO/Sources/Play.c:115`). `firstPlayer`
records **who got to the exit first** — it is read by `MoveRoomToRoom`'s tail
(`GliderPRO/Sources/Transit.c:290-309`) and `ReadyGliderFromTransit`
(`GliderPRO/Sources/Transit.c:145-146`) to decide which player gets idled on arrival.

`UndoGliderLimbo(thisGlider)` (`GliderPRO/Sources/Modes.c:472-480`):

```
   1. if (twoPlayerGame) and (onePlayerLeft) and (which == playerDead) then return
   2. if mode == kGliderInLimbo then mode <- wasMode
   3. dontDraw <- false
```

`TagGliderIdle(thisGlider)` (`GliderPRO/Sources/Modes.c:631-639`):

```
   1. if (twoPlayerGame) and (onePlayerLeft) and (which == playerDead) then return
   2. wasMode <- mode
   3. mode    <- kGliderIdle (22)
   4. hVel    <- 30            // reused as a 30-frame countdown, NOT a velocity
```

`HandleIdleGlider(thisGlider)` (`GliderPRO/Sources/Player.c:1323-1331`):

```
   1. hVel--
   2. if hVel <= 0 then
   3.     mode     <- wasMode
   4.     dontDraw <- false
```

So an idled glider is frozen for exactly **30 sim frames = 1 second at 30 fps**. This is a
*deterministic* grace period, which makes it perfectly reproducible over a network. It is
applied:

| Site | Who gets idled |
| --- | --- |
| `GliderPRO/Sources/Play.c:189` | glider 2, at game start |
| `GliderPRO/Sources/Transit.c:292-297` | whichever glider is **not** `firstPlayer`, after `MoveRoomToRoom` |
| `GliderPRO/Sources/Transit.c:145-146` | whichever glider is **not** `firstPlayer`, in `ReadyGliderFromTransit` |

## 9.7 Feature restrictions in two-player mode

| Restriction | Evidence |
| --- | --- |
| **Games cannot be saved.** `SaveGame` returns immediately. | `GliderPRO/Sources/SavedGames.c:309-310` `if (twoPlayerGame) return;` |
| Cmd-S save is disabled | `GliderPRO/Sources/Input.c:66` `else if ((BitTst(&theKeys, kSKeyMap)) && (!twoPlayerGame))` |
| Cmd-Q does not offer to save | `GliderPRO/Sources/Input.c:59` `if ((!twoPlayerGame) && (!demoGoing))` |
| The demo/attract mode is single-player only | `GliderPRO/Sources/Play.c:479` — `GetDemoInput` is only called in the `else` (single-player) branch |
| The band-fired-recoil transfer differs per player | `GliderPRO/Sources/RubberBands.c:163-166` (player 1) vs `:186-193` (player 2) |

**The save-game restriction is the single most important fact for a networked port**: there
is no existing serialisation of a two-player game state. `gameType` (40 bytes) describes
one glider. A networked progress packet is exactly the missing piece, and §10.4 specifies
one.

## 9.8 Why the original's design is *already* net-friendly

Four properties of the existing two-player mode make a networked version tractable:

1. **The two gliders never physically collide with each other.** There is no
   glider-vs-glider collision test anywhere in `Interactions.c` — each glider is tested
   against room objects, hot spots, dynamics and bands independently. Their only
   interactions are through *shared scalars* (`mortals`, `theScore`, `batteryTotal`,
   `bandsTotal`, `foilTotal`, `numStarsRemaining`) and through the *room they are both in*.
2. **They are always in the same room.** Room transitions are a joint operation
   (§5.7): one player reaches an exit, goes into limbo, and the room only changes when the
   other joins (or dies). There is never a state where the two gliders are simulating
   different rooms. **This is enormous** — it means the entire dynamic-object set, hot-spot
   set, trigger set and band set is common to both peers, and the sync point is naturally
   a barrier.
3. **The escape handshake is already a two-phase protocol.** `otherPlayerEscaped` is a
   rendezvous variable: first arriver sets it, second arriver clears it and performs the
   transition (§3.5.4). Mapping that onto messages is nearly mechanical.
4. **`gameFrame` is a monotone logical clock** shared by both gliders
   (`GliderPRO/Sources/Play.c:434`), and the demo system proves it is a valid sequence
   number for input.

The one property working *against* a networked port is that both gliders' interactions are
resolved inside a single `HandleInteraction()` call
(`GliderPRO/Sources/Interactions.c:1691-1711` — declaration at `:1691`, closing brace at
`:1711`) against shared mutable room state — so "each machine simulates its own player"
(mode (a)) is not a straight code split. Re-verified against the source, the entanglement
is fourfold and every strand is in that one call tree:

| Strand | Evidence in the original |
| --- | --- |
| One loop over the shared `hotSpots[]` array tests **both** gliders per iteration | `CheckForHotSpots`, `GliderPRO/Sources/Interactions.c:1627-1687`: `SectGlider(&theGlider, …)` at `:1639`, `SectGlider(&theGlider2, …)` at `:1657`, inside the same `for (i = 0; i < nHotSpots; i++)` at `:1632` |
| The anti-retrigger latch is shared, and cleared only if **neither** glider is over the hotspot | `:1638` `hitObject = false;` … `:1674-1675` `if (!hitObject) hotSpots[i].stillOver = false;` |
| Prize pickup is first-caller-wins on shared house state | `SetObjectState`, `GliderPRO/Sources/Objects.c:460-462`: `changed = (… data.c.state == true); newState = false; … data.c.state = newState;` — the second caller in the same frame sees `changed == false` |
| A room transition can happen *between* the two boundary checks | `CheckGliderInRoom(&theGlider)` at `:1705`, `CheckGliderInRoom(&theGlider2)` at `:1706`; the first can reach `CheckEscapeUpTwo` (`:171`) → `MoveRoomToRoom` (`:188`) → `ReadyLevel` and rebuild the whole room before player 2's check runs |

`§10.5` — **`docs/analysis/determinism-networking.md:1605`, "10.5 The authoritative fix for
the shared-`HandleInteraction` problem"** — addresses that, in the only way the source
allows: keep `HandleInteraction` verbatim and have *every* peer simulate *every* player
(§10.5.2). See the cross-reference table below for where the rest of Part 10 lives.

---

<!-- BEGIN appended cross-reference (Part 10 lives in a separate file) -->

## Part 10 — see `determinism-networking.md`

Part 10 is **not in this file**. It was written as a separate document to keep this one
navigable:

> `docs/analysis/determinism-networking.md`

Every forward reference from Parts 1–9 resolves there, at the same section numbers:

| Referenced as | Section in `determinism-networking.md` | Subject |
|---|---|---|
| §10.1 | `10.1 The simulation/render cut` | where to cut so `RenderFrame` stops mutating simulation state (the problem stated in §3.9) |
| §10.2 | `10.2 Lock-step vs rollback: the decision` | the decision and its justification from the eleven-phase frame order of Part 3 |
| §10.3 | `10.3 RNG: own PRNG, named streams, and decoupling from rendering` | decoupling RNG consumption from `GetNumberOfLights` and from render-time decisions (hazards 3 and 25 of §7.2) |
| §10.4 | `10.4 The wire format` | the input-frame wire format modelled on `demoType` (§8.1) |
| §10.5 | `10.5 The authoritative fix for the shared-HandleInteraction problem` | the paragraph immediately above this one |
| §10.6 | `10.6 The otherPlayerEscaped rendezvous on the wire` | how the two-phase rendezvous of §3.5.4 maps onto messages |
| §10.7 | `10.7 Corrections and additions to Part 7's hazard table` | corrections to hazards 3, 6, 13, 25 and three new hazards 35–37 |

That file also carries the master `## Open questions` (10 items) and `## Porting notes`
(20 items) lists for the whole determinism analysis, Parts 1–10. This file's own
`## Open questions` / `## Porting notes` sections, at the very end, restate the items that
are derived purely from Parts 1–9 and point at the master lists.

<!-- END appended cross-reference -->

---

# Appendix Z. Correction: the shipped demo does not depend on `Random()`

*Appended section. Added by the toolbox-primitives investigation; nothing above this line was
modified. Full evidence in `docs/analysis/toolbox-primitives.md` Part 1 (see §1.3, §1.10-§1.14).*

## Z.1 What Part 2 gets right

Part 2 correctly identifies `RandomInt` as the only live RNG entry point and correctly transcribes
it from `GliderPRO/Sources/Utilities.c:72-82`:

```
RandomInt(range) = (|Random()| * range) / 32768
```

It is also right that `Main.c:232`'s bare `thePrefs.fakeLong = Random();` is inert - it sits inside
`#ifndef COMPILEDEMO`-adjacent conditional compilation and `COMPILENOCP` **is** defined
(`GliderPRO/Headers/GliderDefines.h:14`). Confirmed: the only live `Random()` call in the entire
tree is `Utilities.c:76`, inside `RandomInt` (`:72-82`); the only other `Random()` call sites
in the tree are `Utilities.c:96` and `:101` in the caller-less `RandomLong`, plus the
compiled-out `Main.c:232`. `RandomLong` (`Utilities.c:88-109`, two draws),
`InitRandomLongQUS` (`:115-118`) and `RandomLongQUS` (`:124-128`, the 1103515245/12345 LCG) all
have **zero callers**, and so does `PourScreenOn` (`Transitions.c`), which means the unbounded
rejection loop at `Transitions.c:44` is dead code.

## Z.2 Correction to §2.2 point 4 - the seeding call is compiled out

`GliderPRO/Sources/Utilities.c:61` reads `GetDateTime((UInt32 *)&qd.randSeed);` but it sits inside
`#if !TARGET_CARBON` ... `#endif` (`Utilities.c:45-64`). `GliderPRO/Prefix.h` - all seven lines of
it - sets:

```c
#define TARGET_CARBON               1
#define ACCESSOR_CALLS_ARE_FUNCTIONS 1
#define OPAQUE_TOOLBOX_STRUCTS      1
#define OPAQUE_UPP_TYPES            1
#define forCarbon                   1
#define BUILDING_RUN_LINKED_IN      0
#define DEBUG                       1
```

So **the shipped Carbon build never seeds `qd.randSeed`**, and classic QuickDraw's documented
initial value is 1. Every launch of 1.0.4 therefore produces an identical RNG stream.
`GliderPRO/Glider PRO CW5.mcp` does contain a non-Carbon target as well, so a 68k/classic-PPC
build *would* have been date-seeded; `toolbox-primitives.md` §1.3 normatively selects the Carbon
behaviour (fixed seed 1) and §1.15 enumerates what changes if you choose the other.

A second, independent reason not to model the date-seeded path: `GetDateTime` returns seconds
since 1904-01-01, which exceeds `0x7FFFFFFF` for every date after 1972-02-11, so reinterpreted as
the `SInt32` that `qd.randSeed` is, it is **negative** throughout 1994-2040. A Lehmer generator
requires state in [1, m-1] and Apple's handling of a negative seed is not recoverable offline
(`GliderPRO/CarbonLib` exports `Random` but has **zero code sections** - all four PEF containers
have `sectionCount` 1 and that section is kind 4, loader).

## Z.3 Correction to Part 2's central claim - the demo replay is RNG-independent

Part 2 asserts that the shipped `demo` 128 replay depends on `Random()`. It does not. Four
independent steps:

**1. The demo's input stream is a pure function of `gameFrame`.** `GetDemoInput`
(`GliderPRO/Sources/Input.c:186-275`) dispatches on
`if (gameFrame == (long)demoData[demoIndex].frame)` and switches on the record's `key` byte
(0 = left, 1 = right, 2 = battery, 3 = rubber band). No RNG. The frame loop selects it at
`GliderPRO/Sources/Play.c:478-481`.

**2. The physics has no RNG.** `GliderPRO/Sources/Player.c` and
`GliderPRO/Sources/Interactions.c` contain **no** reference to `Random`, `RandomInt` or
`RandomLong` (`/usr/bin/grep -c Random` returns 0 for both). There is no `PlayerControl.c` in
this tree; an earlier draft cited one. All 32 live `RandomInt` call sites break down as
animation **18** (`DynamicMaps.c` 5, `Dynamics.c` 3, `Dynamics3.c` 1, `GameOver.c` 9), the
editor 5 (`ObjectAdd.c`), the launch path 1 (`InterfaceInit.c`), audio scheduling 7
(`Play.c` - the telephone and chimes), and dead code 1 (`Transitions.c`, inside the
caller-less `PourScreenOn`) - 18+5+1+7+1 = **32**, matching §2.5. Enumerated in
`toolbox-primitives.md` §1.10.

**3. Demo House contains almost nothing that consumes the RNG.** Re-parsed with
`tools/probe_house.py`: 45 rooms, 1080 objects, and exactly **five**
RNG-consuming object instances (1 `kCandle`, 2 `kTaper`, 1 `kCuckoo`, 1 `kStar`; zero `kStubby`,
zero `kTiki`, zero `kBBQ`). Critically it has **zero** `kChimes` (0x8F), **zero** `kSparkle`
(0x2D) and **zero** `kCoffee` (0x66), with `phoneBit` False, `wardBit` False, `flags` 0,
`initial` (49, 107), `bannerStarCountOn` True, and `fileSize == expectedSize == 16526`.

**4. Zero RNG draws occur inside the demo frame loop.** The only per-frame RNG consumer that
could fire during a demo is the telephone. `InitTelephone` (`Play.c:733-740`) is called **once**,
at `Play.c:192` in `NewGame`, consuming 3 draws, and sets
`nextRing = RandomInt(kRingSpread) + kRingBaseDelay` with `kRingSpread` = 25000 and
`kRingBaseDelay` = 5000 (`Play.c:21-22`). So `nextRing >= 5000` frames while the demo spans frames
**46..3414**: the phone can never ring during the demo. `numChimes` is 0 because the house has no
`kChimes`, killing the chime branch. And even if the phone did ring, `HandleTelephone`
(`Play.c:744-789`) only calls `PlayPrioritySound` - it touches no physics variable.

Total RNG consumption for a full demo playthrough: **at most 8 draws**, none of which reaches
gameplay state.

## Z.4 The containment proof

Every array written by an RNG-consuming registrar, with every reader enumerated:

| Array | Written | Read | What the read does |
|---|---|---|---|
| `flames[]` | `DynamicMaps.c:337-341` | `Render.c:202-216` | advance mode, pick src rect, `CopyBits(..., srcCopy, nil)` |
| `tikiFlames[]` | `DynamicMaps.c:421-426` | `Render.c:221-235` | same |
| `bbqCoals[]` | `DynamicMaps.c:507-511` | `Render.c:240-254` | same |
| `pendulums[]` | `DynamicMaps.c:592-603` | `Render.c:278-317` | same |
| `theStars[]` | `DynamicMaps.c:684-690` | `Render.c:429-445` | same |

There are **no other readers of any of the five arrays anywhere in the tree**. No collision rect,
no `IsThisValid` test, no hot-spot list and no player field ever consults them.

**Conclusion.** A port can replay `demo` 128 frame-exactly with `Random()` stubbed to return 0
forever and the glider will follow the identical path. The RNG is needed only for pixel-exact
animation - flame/coal/pendulum phase, flower species, the game-over page flutter, the telephone
and chime schedule, and the launch-time splash variant.

## Z.5 `demo` 128, re-verified byte-exactly

`demoType` = `{long frame; char key; char padding;}`, **6 bytes**, big-endian `frame`.
`kDemoLength` = 6702 (`GliderPRO/Headers/GliderDefines.h:625`) = **1117 records**. Loaded by
`GliderPRO/Sources/StructuresInit2.c:281-297` via `NewPtr(kDemoLength)` +
`GetResource('demo', 128)` + `BlockMove`.

Observed:

```
length 6702 bytes = 1117 records of 6
first 36 bytes: 00 00 00 2E 00 72 | 00 00 00 2F 00 72 | 00 00 00 30 00 72
                00 00 00 31 00 45 | 00 00 00 38 00 06 | 00 00 00 39 00 72
frame range 46..3414, strictly increasing: True, span 3369
key histogram {0: 910, 1: 198, 3: 9}         <- key 2 (battery) never appears
gap histogram (top 8) {1:1009, 7:8, 4:6, 8:6, 3:6, 10:6, 14:4, 13:4}, max gap 100
padding: 109 distinct values, 407 zeros, top 5 = (0,407) (255,42) (32,39) (114,33) (111,24)
key-3 (banding) records at indices 438, 439, 440, 459, 460, 461, 467, 468, 469
```

Three port-relevant facts:

1. **`padding` is uninitialised 1994 garbage.** `LogDemoKey` (`Input.c:44-49`) writes only `frame`
   and `key`. Nothing ever reads `padding`. Ignore byte 5 but keep the 6-byte stride.
2. **`demoIndex` is unbounded** - `GetDemoInput` increments it with no `< 1117` guard, so after
   frame 3414 the original reads 6 bytes past the `NewPtr` block. Bounds-check in a port; C
   silently read garbage where Go will panic.
3. **`BUILD_ARCADE_VERSION` is 1** (`GliderDefines.h:16`), and `GetDemoInput` contains a block that
   aborts the demo on any of the four gameplay keys, so the demo is interruptible at any frame.

## Z.6 The normative generator, for reference

Specified in full in `toolbox-primitives.md` §1.5-§1.8. Summary: Lehmer / Park-Miller minimal
standard, `a` = 16807 (0x41A7), `m` = 2147483647 (0x7FFFFFFF), Schrage `q` = 127773 / `r` = 2836,
return = low 16 bits of the new state reinterpreted as `int16`, seed 1, period 2147483646, with
states 0 and 0x7FFFFFFF absorbing. First 24 `Random()` values from seed 1:

```
16807, 15089, -21287, 3114, -18558, -9528, -28968, 2558, 12099, 1101, -26472, 15445,
4748, -9246, -11085, 14151, 14615, 16657, -15464, 18772, 11823, 11025, -27091, -29947
```

Two traps: `RandomInt(range)` is **inclusive** of `range` (it returns `range` when the raw word is
`0x8000`, 1 in 65536), so `rand.Intn` is not equivalent; and a naive `int32`
`(state * 16807) % 2147483647` overflows for `state > 127773` and diverges from Schrage within two
draws (172226 of the 299999 seeds in [1, 299999] differ).

---

## Open questions

The determinism analysis' open questions are consolidated with Part 10's, in
`docs/analysis/determinism-networking.md` § "Open questions" (10 numbered items). The three
that belong to Parts 1–9 of *this* file, restated so this document is not silent on them:

1. **What exactly does `Random()` return?** The trap is not in the tree; the only live call
   site is `GliderPRO/Sources/Utilities.c:76` inside `RandomInt` (`:72-82`). Answered
   *for the shipped build* by Appendix Z and `toolbox-primitives.md` §1.5-§1.8 (Lehmer /
   Park-Miller, `a` = 16807, `m` = 2147483647, seed 1); item 1 there.
2. **Is `HandleTelephone`'s `!phoneBitSet` gate inverted?** Re-verified and narrowed:
   `phoneBitSet` is a **per-house** flag, bit 1 of `houseType.flags`, decoded once at
   `GliderPRO/Sources/HouseIO.c:417`; it is *not* a per-room "this room has a phone" flag, as
   an earlier draft of §4.2 said. The phone countdown therefore advances for every room of
   every house whose flag is **clear** (`GliderPRO/Sources/Play.c:748`), while the chime arm
   is separately gated on the genuinely room-scoped `numChimes > 0` (`:771`). Whether the
   author intended `phoneBitSet` to mean "this house has a phone" (in which case `:748` is
   inverted) or "suppress the phone in this house" (in which case it is correct) is not
   decidable from the source; the House Info dialog is the only writer
   (`GliderPRO/Sources/HouseInfo.c:266-273`) and its resource label is not in the `.c` file.
   **Port the observed behaviour, not the guessed intent.** Related latent bug in the same
   writer: clearing the bit uses `flags & 0xFFFFDFFD` (`HouseInfo.c:272`), which also clears
   bit 13 — harmless today because only bits 0-2 are defined (`HouseIO.c:416-418`), but do not
   copy the mask. §2.6; item 10 there.
3. **Does any shipped house depend on the RNG for a *reachable* outcome?** §8.7 shows the
   demo replays correctly under a fresh stream every launch, yet `HandleCoffee`
   (`GliderPRO/Sources/Dynamics.c:492`, `:506`) and `HandleSparkleObject` (`:304`) are
   RNG-timed in principle. Unsurveyed; item 7 there.

## Porting notes

The full ordered list — 20 items, Parts 1–10, ranked by the cost of getting them wrong — is
in `docs/analysis/determinism-networking.md` § "Porting notes". The four that are derived
purely from Parts 1–9 of this file:

1. **Do not adopt "each machine simulates its own player."** `HandleInteraction`
   (`GliderPRO/Sources/Interactions.c:1691-1711`) resolves both gliders against shared
   mutable room state in one call: one hotspot loop over `hotSpots[]` testing `theGlider`
   at `:1639` and `theGlider2` at `:1657` with a shared `stillOver` latch (`:1674-1675`),
   first-caller-wins prize pickup in `SetObjectState`
   (`GliderPRO/Sources/Objects.c:460-462`), and a full room rebuild possible between
   `CheckGliderInRoom(&theGlider)` at `:1705` and `CheckGliderInRoom(&theGlider2)` at
   `:1706`. §9.8, resolved in §10.5.2.
2. **`RandomInt`'s upper bound is inclusive and its distribution is skewed.** Reimplement
   `abs(Random()) * range / 32768` exactly (`GliderPRO/Sources/Utilities.c:72-82`);
   `RandomInt(6)` can return 6. §2.1.1.
3. **One integer timestep, no delta time.** Every velocity, gravity and timer constant is a
   per-frame integer. `kTicksPerFrame` (= 2, `GliderPRO/Headers/GliderDefines.h:533`) only
   sets the wall-clock deadline (`GliderPRO/Sources/Render.c:662-665`, `:690`) — never the
   step count — and appears in the simulation solely as the compile-time divisor that
   converts an author's tick delay to frames (`GliderPRO/Sources/Dynamics3.c:316`, `:401`,
   `:426`, `:456`, `:504`, `:521`), which is constant-folded and therefore harmless. §1.1,
   §1.3.
4. **Preserve the player-1-first ordering everywhere:** input
   (`GliderPRO/Sources/Play.c:452-453`), hotspots (`Interactions.c:1639` before `:1657`),
   room-boundary checks (`:1705` before `:1706`), glider updates (`Play.c:460-461`). §3.5,
   §9.3.
