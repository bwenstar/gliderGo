# Stage 1.5 — objects, collision, room transitions: the port specification

**What this is.** The document Stage 1.5's Go is transcribed from. It is the union of eight
subsystem passes over the 1994 Glider PRO C sources plus three adversarial critiques of those
passes, re-verified against the C and assembled into six sections. It explains *why* the C is as it
is and what a porter would get wrong; it is not a paraphrase of the code. Citations are
`File.c:line`, or bare `:line` once the file is established, in the line-number space of the
CR→LF-converted sources (`tr '\r' '\n' < Sources/X.c`), which is the space every other document in
`docs/analysis` uses. The eight raw reports this was built from are in
`docs/analysis/stage15-raw/<name>.md` and are working notes, not deliverables — see §6.8 before
copying anything out of them.

**What Stage 1.5 covers.** Everything between the house data (Stage 1.2, `internal/house`), the room
compositor (1.3, `internal/render`) and the glider state machine (1.4, `internal/game/player`): the
object graph and the hot-spot table, the 28-action collision dispatcher, the frame loop, the dynamics
table with its appliances and enemies, rewards and switches and triggers, rubber bands and grease,
background animations, and room transitions. It does not cover sound (1.6), QuickTime movies, demo
playback, or the editor halves of `Link.c` / `Room.c` / `Objects.c` — see §5.5.

**Size.** ~9,145 lines of C ported, out of ~17,400 that have to be read to port them (§5.1). Files
ported: `Play.c`, `Interactions.c:756-1777`, `Objects.c`, `ObjectRects.c`, `Room.c` (runtime half),
`RoomGraphics.c`, `Render.c`, `DynamicMaps.c`, `Dynamics.c`, `Dynamics2.c`, `Dynamics3.c`, `Trip.c`,
`Triggers.c`, `RubberBands.c`, `Grease.c`, `Transit.c`, `Transitions.c`, `Scoreboard.c`,
`Banner.c:205-236`, `Link.c:34-53`, `Player.c:1443-1604`, `GameOver.c:237-242`, plus registration
calls added *into* `ObjectDrawAll.c`'s existing 1.3 sweep. Stage 1.4 already owns `Player.c` (less
that tail), `Modes.c`, `Input.c` and `Interactions.c:54-752`.

**How to read it.** §1 is the contract; read it first and in full. §2 says where every piece of state
lives and for how long, and ends with the verdict on `internal/game/player/env.go`. §3 is the
per-object-type behaviour table — a lookup, not a read. §4 is 165 numbered traps in nine
cost-ordered tiers; read the tier headings now and the tier itself before writing the code it
covers. §5 is the sub-stage plan. §6 is what is still open, plus a correction batch against the
existing `docs/analysis` corpus and against comments in already-committed Go. Anything the assembly
could not verify carries a literal `UNVERIFIED:` inline; those are not facts, and none of them was
promoted during assembly.

**Contradictions between the six sections had to be resolved against the C.** They are listed
immediately below, before the table of contents, so they can be re-checked.

### Contradictions resolved during assembly

1. **Section numbering.** §1 was drafted against a *guessed* numbering in which §2–§9 meant the
   eight raw subsystem reports, and §4's preamble mis-numbered the same way ("§2 (the pipelines) and
   §3 (live state)"). Resolved by rewriting §1's owner column, §1.0.1's mapping table and §2's two
   stray references to name the **raw report** (`hotspot-pipeline`, `rewards-switches-triggers`,
   `dynamics-core`, `dynamics-movers`, `bands-grease`, `background-animations`, `room-transitions`,
   `live-state-inventory`), and by repointing §4's preamble at §1 and §2. No §-reference in this
   document now means anything other than a section of this document, except where the target
   document is named — inline (`interactions.md` §19.2) or, in §6.6's correction tables, in the
   caption above the table. §7, §8 and §9 do not exist here; every surviving `§7`/`§8`/`§9` is a
   section of `interactions.md`, `enemies.md` or `progression.md`.
2. **`numChimes` division by zero.** `live-state-inventory` T25 and its openQuestion 1 want a guard;
   the whole chime block is `if (numChimes > 0)` at `Play.c:771` and the division at `:780` is inside
   it, so it is unreachable. Fullest treatment §6.1 item 1; also §1.1 F4 and §4 trap 156.
3. **The double `RenderFrame`.** `background-animations` asserts no animation family ever advances
   twice in a row; `room-transitions` has the four `Transit.c` call sites and does not follow them
   through. The C sides with `room-transitions`. Fullest treatment §1.5; policy in §6.4 item 1;
   traps 24 and 158.
4. **Hot spots per object: two or three?** `live-state-inventory`'s "24 × 2 = 48 < 56, therefore
   safe" is void — a tall flame emits three (`ObjectRects.c:410`, `:413`, `:421`). The bound still
   holds on shipped data, with a corpus peak of 53. Fullest treatment §6.1 item 4; also §2.2, §3.1
   and trap 76, whose `UNVERIFIED:` markers are kept.
5. **A star's `savedMaps` cost.** `background-animations` says two slots and derives an exhaustion
   threshold of 12; `AddStar` calls `BackUpToSavedMap` once (`DynamicMaps.c:679-680`), so the strip
   is one slot and the threshold is 24. Trap 157; §5.3 (1.5f); §6.2 item 2.
6. **`kInvisObstacle`.** `hotspot-pipeline` files it under `kIgnoreIt` as a placeholder;
   `ObjectRects.c:629-631` gives it `kDissolveIt`. Trap 160; §3.3.
7. **How many appliance handlers share the `timer > 0` skeleton.** One report says eight and seven in
   the same paragraph; the C says six. §3.8; §5.3 (1.5c); §6.8.
8. **`vDesiredVel = kGravity`.** `dynamics-core` cites `Player.c:90`, which is the clamp inside the
   ramp and asserts the opposite; it is `:92`. §1.2 LB2; §6.8.
9. **`FlagStillOvers`.** `env.go:103-105`, `interactions.md` and `enemies.md` all say it runs on room
   entry and makes a glider dropped onto a switch trigger it; it has one caller
   (`FinishGliderDuctingIn`, `Player.c:954`) and it *suppresses* the switch. Fullest treatment §2.12;
   trap 18; §6.6 and §6.7.
10. **Draw order is not the reverse of build order.** §6.8 item 4 gives both orders; the only true
    statement is that central is first in one and last in the other.
11. **`bands-grease`'s `RoomGraphics.c` line numbers** are uniformly one low and two of them name the
    wrong call. Corrected order in §5.2 item 4 and §6.8 item 1; §1.4.4 uses the corrected numbers.
12. **`objDataType` is 26 bytes, not 36** (`GliderStructs.h:105`); 36 is `dynaType`. Trap 165; §6.8
    item 2.
13. **`DrawLocale` has three call sites and `ReadyLevel` four.** `room-transitions` and
    `hotspot-pipeline` were each right about a different function; §1.4.3 and §1.4.4 now say which.
14. **"The 117-row type → hot-spot-action table is specified nowhere."** True of the eight raw
    reports, and §5.3, §5.4 and §6.3 item 8 all say so; but §3.2-§3.10 of this document does specify
    it, for all 117 types, with `ObjectRects.c` citations. All three places now say so. What is still
    outstanding is a verification read of `ObjectRects.c:296-1063` against §3's tables.
15. **Left open on purpose:** whether any shipped 3×3 view over-subscribes the 18-slot `dinahs`
    table. `live-state-inventory`'s census says no; `dynamics-core` and `dynamics-movers` argue yes
    by arithmetic. The censuses are not comparable and the answer is screen-size dependent, so this
    is *not* resolved here — see §6.2 item 1 for the census that would settle it.

---

## Table of contents

**§1 The frame, end to end**
- 1.0 Build configuration, and what that adds to the frame
- 1.0.1 Owner labels
- 1.1 The frame, unmodified — F1..F11, with `HandleInteraction` (F6c) and `RenderFrame` (F10b) expanded
- 1.2 Where evaluation order is load-bearing — LB1..LB13
- 1.3 The one-line summary of the contract
- 1.4 The room-change path — 1.4.1 entry points, 1.4.2 `MoveRoomToRoom`, 1.4.3 `ReadyLevel`, 1.4.4 `DrawLocale`, 1.4.5 what the ordering guarantees
- 1.5 `RenderFrame` runs twice on a transition frame
- 1.6 The four steps no subsystem report owns — 1.6.1 `HandlePlayEvent`, 1.6.2 `HandleDynamicScoreboard`, 1.6.3 the game-over tail, 1.6.4 two things that DO recompose the room mid-sweep
- 1.7 What this means for the Go surface
- 1.8 Not settled here

**§2 The state, and who owns it**
- 2.1 The five lifetimes, and the exact C site that ends each
- 2.2 The bounds, and the four `#define`s that are not where you expect
- 2.3 Room-scoped: the tables `DrawLocale` rebuilds
- 2.4 Room-scoped: the scalars
- 2.5 House-scoped
- 2.6 Game-scoped
- 2.7 New-life scope: `OffAMortal`
- 2.8 Never reset
- 2.9 One package: `internal/game`
- 2.10 The two types
- 2.11 How the two gliders share what they share
- 2.12 Verdict on `internal/game/player/env.go`

**§3 Behaviour by object class**
- 3.0 How to read the tables
- 3.1 Rules that apply across every group — 3.1.1 `SetObjectState`, 3.1.2 what contributes to `numLights`
- 3.2 Blowers — `blowerType`, 0x01-0x10, 16 types
- 3.3 Furniture — `furnitureType`, 0x11-0x1F, 15 types
- 3.4 Bonuses and prizes — `bonusType`, 0x21-0x2F, 15 types
- 3.5 Transports — `transportType`, 0x31-0x40, 16 types
- 3.6 Switches and triggers — `switchType`, 0x41-0x49, 9 types
- 3.7 Lights — `lightType`, 0x51-0x58, 8 types
- 3.8 Appliances — `applianceType`, 0x61-0x6E, 14 types
- 3.9 Enemies — `enemyType`, 0x71-0x79, 9 types
- 3.10 Clutter — `clutterType`, 0x81-0x8F, 15 types
- 3.11 The no-behaviour census
- 3.12 Open questions and unverified claims

**§4 The traps** (165, numbered continuously across the tiers)
- 4.A Crashes and out-of-bounds reads that the C survived and Go will not (1-15)
- 4.B Inverted outcomes (16-23)
- 4.C Statement order and control-flow shape (24-60)
- 4.D Index spaces, overloaded fields and aliased unions (61-75)
- 4.E Capacity, ordering and RNG determinism (76-87)
- 4.F Two-player asymmetries (88-95)
- 4.G Arithmetic, coordinate spaces and geometry (96-113)
- 4.H Original bugs and quirks that are visible and must be a conscious decision (114-155)
- 4.I Refuted, or checked and not a trap — do not re-add these (156-165)

**§5 Sub-stage decomposition**
- 5.1 How big Stage 1.5 actually is
- 5.2 The four things that must be settled before 1.5a starts
- 5.3 The sub-stages — 1.5a world/room/object graph, 1.5b frame and traversal, 1.5c dynamics, 1.5d rewards and switches, 1.5e bands and grease, 1.5f background animations
- 5.4 Build 1.5a first — and why the tempting alternatives are wrong
- 5.5 What Stage 1.5 does not cover

**§6 Open questions**
- 6.1 Settled during this pass — do not re-open
- 6.2 Still open: data questions a corpus census would settle
- 6.3 Still open: C reads nobody has done
- 6.4 Still open: decisions, not discoveries
- 6.5 Permanently unverifiable from the source
- 6.6 Corrections to apply to `docs/analysis/*.md` (batch — do not apply from here)
- 6.7 Corrections to comments in already-committed Go
- 6.8 The eight raw reports, and the four disagreements not to re-import

---

## 1. The frame, end to end

This section is the contract every other section serves. It is the single authoritative ordered
list of everything that happens in one Stage 1.5 frame, the single authoritative list of the points
where evaluation order is observable, and the owner of the four frame steps that no subsystem report
claimed. Everything else in this document specifies the *inside* of one of the calls listed here.

Where a fact below contradicts a subsystem report, the C was re-read and the C wins; those cases are
flagged inline as **CORRECTION**.

### 1.0 Build configuration, and what that adds to the frame

Two `#define`s change the shape of the frame and no report stated which configuration is normative.
Both are on in the shipped source, so **both are normative for Stage 1.5**:

- `#define COMPILEQT` — `GliderDefines.h:15`. Adds two `MoviesTask` calls to the frame
  (`Play.c:466-467`, `:491-492`) and **four unconditional `RenderFrame()` calls** inside the four
  Transit.c transition functions (`Transit.c:303`, `:341`, `:380`, `:419`). See §1.5 — this is the
  most expensive consequence of the build config and it is not cosmetic.
- `#define BUILD_ARCADE_VERSION 1` — `GliderDefines.h:16`. Compiles in the scoreboard-blackout
  block at `Play.c:512-541`, which runs once, on the last frame of the game-over countdown.
  The other three `BUILD_ARCADE_VERSION` sites in the frame path (`Play.c:138`, `:806`, and the
  `#else` at `:542-544`) contain only commented-out `HideMenuBarOld`/`ShowMenuBarOld` calls and have
  no behaviour.

Nominal frame period: `kTicksPerFrame = 2` (`GliderDefines.h:533`), i.e. 30 Hz on a 60.15 Hz
tick clock, enforced by the busy-wait at `Render.c:662-664`. There is a **second, independent**
pacing source stacked on top of it — see step F3.

### 1.0.1 Owner labels

Each call below names the subsystem that specifies its interior, by the name of the raw report whose
pass covered it (`docs/analysis/stage15-raw/<name>.md`). The assembled section that carries the
material is given here once and not repeated in the tables below. Stage 1.4's own steps are marked
**1.4**.

| raw report | subsystem | assembled section |
|---|---|---|
| (this section) | the frame itself; `HandlePlayEvent`, `MoviesTask`, `HandleDynamicScoreboard`, the game-over tail | §1.1, §1.6 |
| `hotspot-pipeline` | hot-spot construction and the 28-action dispatcher | §3.1, §3.2-§3.10 |
| `rewards-switches-triggers` | rewards, switches, triggers | §3.4, §3.6 |
| `dynamics-core` | dynamics core: registration, appliances, `CheckDynamicCollision` | §3.8 |
| `dynamics-movers` | dynamics movers: toast, balloon, copter, dart, ball, drip, fish | §3.9 |
| `bands-grease` | rubber bands and grease | §3.4 (grease jars), §5.3 (1.5e) |
| `background-animations` | background animations: flames, coals, pendulums, stars, sparkles, flying points, shreds, the phone and chimes | §3.2 (flames), §5.3 (1.5f) |
| `room-transitions` | room transitions | §1.4, §3.5 |
| `live-state-inventory` | live-state inventory and scoping | §2 |


Stage 1.4 (`internal/game/player`) is already done and is referred to as **1.4** below. Its entry
points are `Glider.HandleGlider(Env)` (`handle.go:727`), `Input.GetInput(*Glider, Env, Keys)`
(`input.go:81`), `Glider.CheckGliderInRoom(Env)` (`escape.go:443`) and `Glider.MoveGlider()`
(`glider.go:220`). Stage 1.5 owns everything else on this list and owns the implementation of
`player.Env`.

### 1.1 The frame, unmodified

`PlayGame`, `Play.c:430-599`. The loop is `while ((playing) && (!quitting))` at `Play.c:432`.

The one-player and two-player arms are **separate code**, not a parameterised loop: `Play.c:447`
splits into `:447-472` (two-player) and `:473-497` (one-player). They are identical in order. Both
line numbers are given below as `2P/1P`.

| # | call | lines (2P / 1P) | owner | notes |
|---|---|---|---|---|
| **F1** | `gameFrame++` | `:434` | §1 | `long`. Read by `HandleDynamicScoreboard` (F10c) as a blink phase, and by nothing else in the frame. |
| **F2** | `evenFrame = !evenFrame` | `:435` | §1 | **The only assignment to `evenFrame` anywhere in the program that is part of the schedule.** Read at `Render.c:649` (F10b/R4), `Dynamics.c:60` (foil gate, `dynamics-core`), and `Interactions.c:1756` (`WebGlider`'s twang, `hotspot-pipeline`). `Dynamics2.c:420` stomps it to `true` from inside a ball handler (`dynamics-movers`). |
| **F3** | `if (doBackground) { do { HandlePlayEvent(); } while (switchedOut); }` | `:437-443` | §1, spec'd in §1.6.1 | Unowned by every report. Not gated on `gameOver` or `playing`. |
| **F4** | `HandleTelephone()` | `:445` | `background-animations` | `Play.c:744-789`. Two **sequential, non-exclusive** `if`s: the phone block `if (!phoneBitSet)` `:748-768` and the chime block `if (numChimes > 0)` `:771-788`. Both can play a sound in one frame. **CORRECTION to `live-state-inventory` T25 and `live-state-inventory` openQuestion 1:** the `delayTime = kChimeDelay / numChimes` at `:780` is *inside* `if (numChimes > 0)` at `:771`, so division by zero is unreachable and needs no guard. The clamp `if (delayTime < 2) delayTime = 2;` at `:781-782` is real and must be ported. |
| | `if (twoPlayerGame)` | `:447` | §1 | The split. |
| **F5** | `HandleDynamics()` | `:449 / :475` | `dynamics-core` dispatch, `dynamics-core`+`dynamics-movers` handlers | **NOT gated on `gameOver`.** `Dynamics3.c:31-105`, a flat `for (i = 0; i < numDynamics; i++)` over `switch (dinahs[i].type)`, no `active` test, no room filter. Slot order is `DrawLocale`'s draw order — see §1.4 step T13 and `live-state-inventory`. |
| | `if (!gameOver) {` | `:450 / :476` | §1 | |
| **F6a** | `GetInput(&theGlider)` | `:452 / :481` | **1.4** | 1P arm is `if (demoGoing) GetDemoInput(&theGlider); else GetInput(&theGlider);` `:478-481`. `GetDemoInput` and `demoIndex`/`demoData` are **out of scope for Stage 1.5**; see §1.8. |
| **F6b** | `GetInput(&theGlider2)` | `:453` | **1.4** | 2P only. No 1P equivalent. |
| **F6c** | `HandleInteraction()` | `:454 / :482` | `hotspot-pipeline` (sweep + dispatcher), `rewards-switches-triggers`, `bands-grease`, `room-transitions`, **1.4** | Expands to F6c-i .. F6c-iv below. |
| | `}` | `:455 / :483` | | |
| **F7** | `HandleTriggers()` | `:456 / :484` | `rewards-switches-triggers` | **OUTSIDE the `!gameOver` guard.** `Triggers.c:79-96`. |
| **F8** | `HandleBands()` | `:457 / :485` | `bands-grease` | **OUTSIDE the `!gameOver` guard.** `RubberBands.c:208-252`. Early-returns on `numBands == 0` (`:213-214`). |
| | `if (!gameOver) {` | `:458 / :486` | §1 | 2P opens a block at `:459`; 1P has no braces. |
| **F9a** | `HandleGlider(&theGlider)` | `:460 / :487` | **1.4**, plus `room-transitions` for the transitions it reaches | `Player.c:1335-1439`. 24-case `switch (mode)`, then unconditionally `ignoreLeft = ignoreRight = ignoreGround = false;` at `:1436-1438`. |
| **F9b** | `HandleGlider(&theGlider2)` | `:461` | **1.4** | 2P only. |
| | `}` | `:462` | | 2P only. |
| | `if (playing) {` | `:463 / :488` | §1 | Note the guard is `playing`, not `!gameOver`: the render still runs during the game-over countdown. |
| **F10a** | `#ifdef COMPILEQT if ((thisMac.hasQT) && (hasMovie) && (tvInRoom) && (tvOn)) MoviesTask(theMovie, 0);` | `:465-468 / :490-493` | §1 | Live. **Stage 1.5 stubs this**: no QuickTime. Record the stub so the next audit does not read the omission as an oversight. `tvOn` staleness is a real `live-state-inventory` issue, not a §1 one. |
| **F10b** | `RenderFrame()` | `:469 / :494` | §1 spine, per-step owners below | `Render.c:639-671`. Expands to R1..R14 below. **Not idempotent, and not the only call site — see §1.5.** |
| **F10c** | `HandleDynamicScoreboard()` | `:470 / :495` | §1, spec'd in §1.6.2 | Unowned by every report, and `dynamics-core`'s leaked-port trap depends on it. |
| | `}` | `:471 / :496` | | |
| **F11** | `if (gameOver) { countDown--; ... }` | `:499-553` | §1, spec'd in §1.6.3 | **Outside both arms**, at loop-body level. Transcribed by three reports, specified by none. |

#### F6c expanded — `HandleInteraction`, `Interactions.c:1691-1711`

```
F6c-i    CheckForHotSpots()                                  // :1693  ALWAYS first
F6c-ii   if (twoPlayerGame)                                  // :1694
             if (onePlayerLeft)                              // :1696
                 if (playerDead == kPlayer1)                 // :1698
                     CheckGliderInRoom(&theGlider2)          // :1699
                 else
                     CheckGliderInRoom(&theGlider)           // :1701
             else
F6c-iii          CheckGliderInRoom(&theGlider)               // :1705
F6c-iv           CheckGliderInRoom(&theGlider2)              // :1706
         else
F6c-iii      CheckGliderInRoom(&theGlider)                   // :1710
```

`CheckForHotSpots` (`Interactions.c:1627-1687`) is a single ascending pass
`for (i = 0; i < nHotSpots; i++)`, no early exit, no sorting, no nearest-wins. In the two-player arm
the second `SectGlider` (glider 2, `:1657-1658`) is **not** in an `else`, so one rect can fire for
both gliders in one iteration, glider 1 first. Full nesting is `hotspot-pipeline`'s; it must not be fused.

`CheckGliderInRoom` is **1.4** (`escape.go:443`), and its shape is quoted here only because
§1.2 LB6 depends on it. `Interactions.c:689-752`:

```
if (mode is one of kGliderNormal, kGliderFaceLeft, kGliderFaceRight, kGliderBurning)   // :691-694
    if (dest.top < kCeilingLimit)                                    // :696
        if (mode == kGliderBurning) { wasMode = 0; StartGliderFadingOut; PlayPrioritySound }
        else if ((twoPlayerGame) && (!onePlayerLeft)) CheckEscapeUpTwo                  // :704
        else CheckEscapeUp                                                             // :707
    else if (dest.bottom > kFloorLimit)                              // :709
        (same three-way, CheckEscapeDownTwo / CheckEscapeDown)                          // :717/:720
    else if ((thisBackground == kRoof) && (dest.bottom > kRoofLimit))// :722
        CheckRoofCollision                                                             // :723
    // ---- a NEW statement, not an else if ----
    if (dest.left < leftThresh)                                      // :725
        (same three-way, CheckEscapeLeftTwo / CheckEscapeLeft)                          // :733/:736
    else if (dest.right > rightThresh)                               // :738
        (same three-way, CheckEscapeRightTwo / CheckEscapeRight)                        // :746/:749
```

**CORRECTION to `room-transitions` §B1:** the report anchors the new `if` at `Interactions.c:724`; `:724` is a blank
line. The new statement is `:725`. The claim it supports (one glider can chain two transitions in
one frame) is correct and is LB7 below.

#### F10b expanded — `RenderFrame`, `Render.c:639-671`

Verified line by line against `grep -an`; four reports transcribe this exactly and they are right.

| # | call | line | owner |
|---|---|---|---|
| **R1** | `if (hasMirror) { DrawReflection(&theGlider, true); if (twoPlayerGame) DrawReflection(&theGlider2, false); }` | `:641-646` | `background-animations` (mirror region), draws the 1.4 glider |
| **R2** | `HandleGrease()` | `:647` | `bands-grease` | 
| **R3** | `RenderPendulums()` | `:648` | `background-animations` |
| **R4** | `if (evenFrame) RenderFlames(); else RenderStars();` | `:649-652` | `background-animations` |
| **R5** | `RenderDynamics()` | `:653` | `dynamics-movers` (only 7 case groups; the appliances and `kSparkle` are deliberately absent — `dynamics-core`) |
| **R6** | `RenderFlyingPoints()` | `:654` | `background-animations` |
| **R7** | `RenderSparkles()` | `:655` | `background-animations` |
| **R8** | `RenderGlider(&theGlider, true)` | `:656` | §1.5-adjacent; a pure draw of 1.4 state |
| **R9** | `if (twoPlayerGame) RenderGlider(&theGlider2, false)` | `:657-658` | as R8 |
| **R10** | `RenderShreds()` | `:659` | `background-animations` |
| **R11** | `RenderBands()` | `:660` | `bands-grease` |
| **R12** | `while (TickCount() < nextFrame) { }` | `:662-664` | §1 — the frame limiter |
| **R13** | `nextFrame = TickCount() + kTicksPerFrame;` | `:665` | §1 |
| **R14** | `CopyRectsQD()` | `:667` | §1 — `Render.c:616-635`: work→main **first** (`:620-626`), back→work **second** (`:628-634`) |
| **R15** | `numWork2Main = 0; numBack2Work = 0;` | `:669`, `:670` | §1 |

**CORRECTION to `bands-grease` §B and §G:** `RenderFrame` is `Render.c:639-671`, not `:639-673`; the two
zeroing statements are `:669` and `:670`, not `:670-671`; `CopyRectsQD`'s second loop is `:628-634`,
not `:625-632`.

R2 is the one that surprises porters: `HandleGrease` is **gameplay running inside the renderer**. It
rewrites `hotSpots[].action`, `.isOn` and `.bounds` in place (`Grease.c:60`, `:61`, `:68`, `:95`,
`:102`) *after* F6c has already swept the table this frame. See LB9.

Note R3's early return: `clockFrame++` is at `Render.c:270`, **after**
`if (numPendulums == 0) return;` at `:267-268`. So `clockFrame` does not advance at all in a room
with no cuckoo clock. This matters to §1.5's double-step accounting.

The two dirty-rect adders are **not** interchangeable, and `live-state-inventory` T9's "identical shape" is wrong:
`AddRectToWorkRects` (`Render.c:65-80`) clamps to `justRoomsRect` (screen coordinates);
`AddRectToBackRects` (`:84-99`) clamps to `0`/`workSrcRect` (work-map coordinates). Both drop
silently once the count reaches `kMaxGarbageRects - 1` = 47 (`Render.c:67`, `:86`), not 48
(`kMaxGarbageRects = 48`, `Render.c:20`).

### 1.2 Where evaluation order is load-bearing

Each of these is observable. Reordering any of them changes gameplay, not just pixels.

**LB1 — `HandleInteraction` (F6c) runs BEFORE `HandleGlider` (F9). `Play.c:482` then `Play.c:487`
(2P: `:454` then `:460`).** This is the single most important ordering in Stage 1.5.

Every velocity write the world performs — a fan's `hDesiredVel += -kFanStrength` / `+= kFanStrength`
(`Interactions.c:1211`, `:1215`; `kFanStrength = 12`, `:15`), a mover's `hDesiredVel = ±kShoveVelocity` (`Dynamics.c:54`,
`:56`), a bounce, a slide — lands in `hDesiredVel`/`vDesiredVel`, which are **requests**, not
displacements. Nothing displaces the glider until `MoveGlider` runs inside `HandleGlider`, and
`MoveGlider` (`Player.c:64-149`) approaches the request one `kHImpulse`/`kVImpulse` step at a time
and clamps to `±kMaxHVel`/`kMaxVVel`. So a world force applied on frame N is **attenuated by the
ramp in the same frame** and never teleports the glider.

The whole accumulation window is therefore, in order:

```
F5  HandleDynamics    -> hDesiredVel = ±kShoveVelocity      ASSIGNMENT (Dynamics.c:54, :56)
F6a GetInput          -> hDesiredVel += / -=                ADDITION   (Input.c:214, :216, :229,
                                                                        :236, :293, :295, :313, :323)
F6c HandleInteraction -> hDesiredVel += / -=                ADDITION   (Interactions.c:1211, :1215)
F9  HandleGlider      -> MoveGlider ramps, clamps, displaces, then
                         hDesiredVel = 0          (Player.c:78)
                         vDesiredVel = kGravity   (Player.c:92)
```

**CORRECTION to `dynamics-core`:** the re-seed is `Player.c:92`, not `:90`. `Player.c:90` is
`thisGlider->vVel = thisGlider->vDesiredVel;` — the clamp *inside* the ramp, which asserts the
opposite of the claim. `dynamics-movers`'s `:92` is right. `hDesiredVel = 0` at `:78` is agreed by both and correct.

Three consequences a port must preserve:
- A mover's shove is an **assignment**, so two movers hitting the same glider in one frame do not
  add — the later slot wins. A shove plus the player's own thrust **do** add.
- The reset happens at the *end* of the frame that consumes it, so `hDesiredVel` is zero at the top
  of every frame and the accumulation window never leaks across frames.
- `MoveGlider` is reached from only **six** of the 24 modes: `MoveGliderNormal` (`Player.c:198`),
  `MoveGliderBurning` (`:226`), `MoveGliderFaceLeft` (`:565`), `MoveGliderFaceRight` (`:582`),
  `MoveGliderFoilGoing` (`:1197`), `MoveGliderFoilLosing` (`:1255`). In the other 18 modes a world
  force written in F6c is simply discarded when the next `MoveGlider` re-seeds — it is not banked.

**LB2 — `HandleDynamics` (F5) runs BEFORE `GetInput` (F6a). `Play.c:475` then `:481` (2P `:449`
then `:452`).** Two effects. First, the priority in LB1: the shove is the assignment and the keys
add to it, so a player holding left into rising toast gets `-kShoveVelocity - kSlowGliderVel`, not
one or the other. Second, `CheckDynamicCollision` compares the mover's *freshly moved* rect against
the glider's **end-of-previous-frame** `dest`, because the glider has not moved yet this frame.

**LB3 — `HandleDynamics` (F5), `HandleTriggers` (F7) and `HandleBands` (F8) are OUTSIDE the
`!gameOver` guard; `GetInput`, `HandleInteraction` and `HandleGlider` are inside it.**
(`Play.c:475` and `:484-485` vs `:476` and `:486`.) During the 16-frame game-over countdown
(`kNumCountDownFrames = 16`, `GameOver.c:17`; set by `FlagGameOver`, `GameOver.c:237-242`):
appliances keep animating and keep consuming RNG; `CheckDynamicCollision` keeps firing and keeps
draining the shared `foilTotal`; already-armed triggers keep counting down and keep firing, so
appliances toggle, outlets zap and grease spills; bands keep flying and keep actuating hot spots
against a stale `theGlider.dest`. A shove written into `hDesiredVel` in this window is never
consumed, because `HandleGlider` no longer runs. Do not "tidy" any of the three inside the guard.

**LB4 — `HandleTriggers` (F7) runs AFTER `HandleInteraction` (F6c), in the same frame.**
`ArmTrigger` sets `timer = data.e.delay * 3` (`Triggers.c:49`); `HandleTriggers` decrements **then**
tests (`:87-88`). So a trigger armed at F6c with `delay == 0` fires at F7 in the **same frame**, and
`delay = d` fires on the `max(1, 3d)`-th call, i.e. `3d - 1` frames later. Verified.

**LB5 — `HandleBands` (F8) runs AFTER `HandleTriggers` (F7).** `Play.c:485` follows `:484`. A
trigger armed by a *band* (`CheckBandCollision` → `ArmTrigger`, `RubberBands.c:137`) is therefore
first decremented on the **next** frame, one frame later than the same trigger armed by the glider
at F6c. LB4 and LB5 are about different edges and are both true; `rewards-switches-triggers` claim 3 and `dynamics-movers` claim 4 look
contradictory and are not — the former is arm→fire, the latter is fire→`dinahs[]` animation.

**LB6 — inside F6c, the whole hot-spot sweep runs before any boundary test.** `CheckForHotSpots`
(F6c-i) sets `ignoreLeft`/`ignoreRight`/`ignoreGround` at `Interactions.c:1417`, `:1421`, `:1587`,
and `CheckGliderInRoom`'s escape checks read them in the **same** `HandleInteraction` call
(`:515`, `:578`, `:605`, `:668` for the walls; `:346`, `:362`, `:404`, `:419`, `:434` for the
ground). They are cleared at the end of `HandleGlider` (`Player.c:1436-1438`) and by
`FlagGliderNormal` (`Modes.c:356-358`). So a doorway rect at index 40 still grants permission for
a boundary test that logically happened "earlier" — the sweep is all-then-test, not interleaved.

**LB7 — one glider can complete two room changes in one frame.** The horizontal test at
`Interactions.c:725` is a **new statement**, not an `else if` chained to the vertical tests at
`:696`/`:709`/`:722`. A glider in a corner exits upward, `MoveRoomToRoom` runs to completion
(including `ReadyLevel`, so `leftThresh` is now the *new* room's), and then `:725` tests the
glider's unchanged X against the new threshold and can fire a second `MoveRoomToRoom`. See §1.4.

**LB8 — in a two-player game, one frame can straddle two rooms.** `CheckGliderInRoom(&theGlider)`
at `:1705` can complete a room change; then `:1706` and both `HandleGlider` calls at
`Play.c:460-461` run against the **new** room's globals. This is precisely why `MoveRoomToRoom`
calls `OffsetGlider` on *both* gliders unconditionally.

**LB9 — `HandleGrease` (R2) runs inside `RenderFrame` (F10b), i.e. after F6c and after F9.** A jar
that lands on frame N promotes its hot spot to `kSlideIt` and turns it on at R2 of frame N, which is
after that frame's sweep. So the glider cannot slip on it until frame N+1, and each `bounds.right +=
2` / `bounds.left -= 2` growth step (`Grease.c:95`, `:102`) is likewise first seen a frame later.

**LB10 — F6c-i (`kSlideIt`) runs before F6c-ii (`CheckRoofCollision`).**
`HandleHotSpotCollision`'s `kSlideIt` sets `sliding = true` (`Interactions.c:1377`) and
`CheckRoofCollision` reads `!sliding` (`:455`). Same frame, in that order. Reverse them and grease
on a roof becomes lethal. Only `MoveGliderNormal` clears `sliding`, and only in `kGliderNormal`.

**LB11 — inside `RenderFrame`, R14 blits work→main before back→work.** Therefore
`AddRectToWorkRects(r)` means "show r on screen at the end of this frame" and
`AddRectToBackRects(r)` means "erase r in the work map *after* it has been shown". Every renderer
follows the pair. `dynamics-core`'s two-frame appliance handshake (`timer == 1` writes `backSrcMap` and queues
back→work; `timer == 0` queues work→main) exists only because of this ordering, and collapsing the
two timer values draws a stale frame.

**LB12 — F10c runs after F10b, and leaves the graphics port set.** `HandleDynamicScoreboard`'s
`QuickScoreRefresh()` (`Scoreboard.c:92`) is the last port-setting call of the frame. `dynamics-core`'s
`HandleOutlet` leaked-port trap depends on that being true; if F10c is dropped or reordered, that
trap's premise disappears. See §1.6.2.

**LB13 — F2 (`evenFrame = !evenFrame`) happens once per frame and only at `Play.c:435`.** Every
half-rate schedule in the game — the flames/stars alternation at R4, the foil-spend gate at
`Dynamics.c:60`, `WebGlider`'s twang at `Interactions.c:1756` — hangs off that one flip. The
consequence is §1.5.

### 1.3 The one-line summary of the contract

```
per frame:  clocks -> events -> ambient sound -> WORLD MOVES -> INPUT -> WORLD ACTS ON GLIDER
            -> triggers -> bands -> GLIDER MOVES -> render (which also runs grease) -> scoreboard
            -> game-over countdown
```

The glider moves **last** among the simulation steps and **first** among nothing. Everything that
wants to influence it has already run, and influenced it only through `hDesiredVel`/`vDesiredVel`.

### 1.4 The room-change path

A room change is not a step in `PlayGame`. Nothing in `PlayGame` mentions rooms. It is a re-entrant
side effect of F6c or F9, and it reorders the frame around itself.

#### 1.4.1 The three places a transit function is entered

1. **From F6c**, via `CheckGliderInRoom` → `CheckEscape*`/`CheckRoofCollision`. All 21
   `MoveRoomToRoom` call sites in `Interactions.c` (`:188`, `:222`, `:252`, `:269`, `:300`, `:334`,
   `:349`, `:365`, `:386`, `:400`, `:407`, `:422`, `:437`, `:528`, `:559`, `:581`, `:594`, `:618`,
   `:649`, `:671`, `:684`) are inside a `CheckEscape*` or `CheckRoofCollision`, i.e. reached from
   `CheckGliderInRoom` (F6c-ii..iv), **never** from `HandleHotSpotCollision` (F6c-i). There are zero
   `ForceThisRoom` and zero `ReadyLevel` calls in `Interactions.c`. So **no hot-spot action changes
   the room synchronously** — every transit hot spot (`kTransportIt` `:1381`, `kMailItLeft` `:1424`,
   `kMailItRight` `:1467`, `kDuctItDown` `:1510`, `kDuctItUp` `:1543`, `kMoveItUp` `:1249`,
   `kMoveItDown` `:1284`) only sets a glider *mode*, and the room change happens on a later frame
   inside F9. `room-transitions`'s claim here is confirmed. **But see §1.6.4 — two *other* things do recompose the
   room from inside the F6c-i sweep, and `room-transitions`'s blanket reassurance would let a porter skip both.**
2. **From F9**, via the mode handlers: `MoveGliderUpStairs` (`Player.c:349`/`:351`/`:358`/`:370`),
   `MoveGliderDownStairs` (`:478`/`:480`/`:487`/`:498`), `TransportGliderOut`
   (`:614`/`:616`/`:623`/`:635`), `MoveGliderDownDuct` (`:721`/`:723`/`:730`/`:741`),
   `MoveGliderUpDuct` (`:818`/`:820`/`:827`/`:838`), `MoveGliderInMailLeft`
   (`:1017`/`:1019`/`:1026`/`:1037`), `MoveGliderInMailRight` (`:1108`/`:1110`/`:1117`/`:1128`).
   This is where most transitions actually fire.
3. **From `OffAMortal`**, which is itself reached only from F9 — from `FadeGliderOut`
   (`Player.c:295`, when `frame < 0`) and from `MoveGliderShredding` (`:1317`). `OffAMortal`
   (`:1484-1604`) either calls `FollowTheLeader` (`:1530`, its only call site, →
   `Transit.c:473-557`) or, on the `mortals == -1 && onePlayerLeft && !gameOver` path (`:1540-1603`),
   re-dispatches a transit function on the **surviving** glider using the **dead** glider's
   destination. That is why the pending-transit triple must be world-scoped, not per-glider — `live-state-inventory`.

#### 1.4.2 `MoveRoomToRoom`, `Transit.c:151-310` — the full-transition ordered list

| # | call | line | owner |
|---|---|---|---|
| **T1** | `HandleRoomVisitation()` | `:155` | `room-transitions` |
| **T2** | `switch (where)` — one of `kToRight :158`, `kToLeft :190`, `kAbove :222`, `kBelow :254`, `default :286` | `:156-288` | `room-transitions` |
| **T3** | tail: `if ((twoPlayerGame) && (!onePlayerLeft)) { if (firstPlayer == kPlayer1) TagGliderIdle(&theGlider2); else TagGliderIdle(&theGlider); }` | `:290-296` | `room-transitions` |
| **T4** | `ReadyLevel()` | `:298` | `room-transitions` → expands to T10..T14 |
| **T5** | `RefreshScoreboard(kNormalTitleMode)` | `:299` | §1 (a pure draw) |
| **T6** | `WipeScreenOn(where, &justRoomsRect)` | `:300` | §1 (a pure draw) |
| **T7** | `#ifdef COMPILEQT RenderFrame();` | `:303` | §1 — **see §1.5** |
| **T8** | `if ((thisMac.hasQT) && (hasMovie) && (tvInRoom) && (tvOn)) { GoToBeginningOfMovie; StartMovie; }` | `:304-308` | §1, stubbed |

Three orderings inside T2 that must not be rearranged (all `room-transitions`'s, all re-verified):
- T1 precedes the `ForceThisRoom` in every T2 branch, so `HandleRoomVisitation` credits the room
  being **left**.
- In `kToRight`/`kToLeft`, `ForceThisRoom` is at `:169`/`:201`, and the `enteredRect` computation at
  `:174-176`/`:183-185` reads `thisRoom->leftStart`/`rightStart` **after** it — so the entry Y comes
  from the **destination** room.
- In `kAbove`/`kBelow`, `ForceThisRoom` (`:224`/`:256`) runs **before** the `if (!takingTheStairs)`
  test (`:225`/`:257`), so `ReadyGliderForTripUpStairs` → `GetUpStairsRightEdge()` (`Modes.c:536`)
  sees the new room. And `takingTheStairs` must be read here, because T12's `DrawLocale` clears it
  (`RoomGraphics.c:127`).

#### 1.4.3 `ReadyLevel`, `RoomGraphics.c:402-418` — the room load

| # | call | line |
|---|---|---|
| **T10** | `NilSavedMaps()` | `:404` |
| **T11** | `#ifdef COMPILEQT if ((thisMac.hasQT) && (hasMovie) && (tvInRoom)) { tvInRoom = false; tvWithMovieNumber = -1; StopMovie(theMovie); }` | `:406-413` |
| **T12** | `DetermineRoomOpenings()` | `:415` |
| **T13** | `DrawLocale()` | `:416` |
| **T14** | `InitGarbageRects()` | `:417` |

**CORRECTION to `bands-grease` §F**, which drops two of the five steps and misplaces a third. `bands-grease` says
"`ReadyLevel` calls `NilSavedMaps()` :404 **then** `DrawLocale()` :405". `DrawLocale()` is at
`:416`, `DetermineRoomOpenings()` sits between them at `:415`, and `InitGarbageRects()` follows at
`:417`. Both dropped calls are load-bearing for this stage: `DetermineRoomOpenings` (`Room.c:816-933`)
publishes `leftThresh`, `rightThresh`, `leftOpen`, `rightOpen`, `topOpen`, `bottomOpen`, which the
very next frame's `CheckGliderInRoom` reads at `Interactions.c:725` and `:738`; `InitGarbageRects`
(`Render.c:675-691`) zeroes `numWork2Main`, `numBack2Work`, `numSparkles`, `numFlyingPts`, sets
`sparkles[i].mode = flyingPoints[i].mode = -1`, and re-seeds `nextFrame`.

**CORRECTION to `live-state-inventory`'s `InitGarbageRects` transcription:** `Render.c:690` is
`nextFrame = TickCount() + kTicksPerFrame;`, not `nextFrame = TickCount();`. Drop the addend and the
first frame after every room load busy-waits zero ticks and runs early.

`ReadyLevel` has **exactly four** call sites, all in `Transit.c`: `:298`, `:335`, `:374`, `:413`.
`DrawLocale` has **exactly three**: `RoomGraphics.c:416`, `Play.c:163`, `Play.c:814`. `room-transitions`'s
"four sites" and `hotspot-pipeline`'s "three sites" are both right, about different functions, and nothing in the
union said so. See §1.6.4 for `Play.c:814`.

#### 1.4.4 `DrawLocale`, `RoomGraphics.c:44-130` — the room-scope reset and rebuild

| # | call | line | owner |
|---|---|---|---|
| **T13a** | `ZeroFlamesAndTheLike()` | `:51` | `background-animations` — `DynamicMaps.c:787-797`: `numFlames :789`, `numTikiFlames :790`, `numCoals :791`, `numPendulums :792`, `numGrease :793`, `numStars :794`, `numShredded :795`, `numChimes :796` |
| **T13b** | `ZeroDinahs()` | `:52` | `dynamics-core` |
| **T13c** | `KillAllBands()` | `:53` | `bands-grease` |
| **T13d** | `ZeroMirrorRegion()` | `:54` | `background-animations` |
| **T13e** | `ZeroTriggers()` | `:55` | `rewards-switches-triggers` |
| **T13f** | `numTempManholes = 0` | `:56` | render-side (see §1.8) |
| **T13g** | `FlushAnyTriggerPlaying()` | `:57` | `rewards-switches-triggers` |
| **T13h** | `DumpTriggerSound()` | `:58` | `rewards-switches-triggers` |
| **T13i** | `tvInRoom = false; tvWithMovieNumber = -1` | `:59-60` | `live-state-inventory`. Note `tvOn` is **not** cleared here. |
| **T13j** | `roomV = rooms[thisRoomNumber].floor` | `:62-65` | `room-transitions` |
| **T13k** | `for (i = 0; i < 9; i++) { localNumbers[i] = GetNeighborRoomNumber(i); isStructure[i] = IsRoomAStructure(localNumbers[i]); }` | `:67-71` | `room-transitions` |
| **T13l** | `ListAllLocalObjects()` | `:72` | `hotspot-pipeline` — the only place `masterObjects[]` and `hotSpots[]` are built; `nHotSpots = 0` at `Objects.c:307` |
| **T13m** | `SetGWorld(backSrcMap); PaintRect(&backSrcRect)` | `:74-76` | render |
| **T13n** | `if (numNeighbors > 3)` → NW, NE, N (at `roomV + 1`), then SW, SE, S (at `roomV - 1`); each: `numLights = GetNumberOfLights(...)`, `DrawRoomBackground(...)`, `DrawARoomsObjects(slot, false)`. **No `DrawLighting` for these six.** | `:78-103` | `background-animations` registration, `dynamics-core` registration, render |
| **T13o** | `if (numNeighbors > 1)` → W then E, each also `DrawLighting()` | `:105-116` | as above |
| **T13p** | central room: `numLights`, `DrawRoomBackground`, `DrawARoomsObjects(kCentralRoom, false)`, `DrawLighting()` | `:118-121` | as above |
| **T13q** | `if (numNeighbors > 3) DrawFloorSupport()` | `:123-124` | render |
| **T13r** | `RestoreWorkMap()` | `:125` | render — whole back over whole work |
| **T13s** | `shadowVisible = IsShadowVisible()` | `:126` | `live-state-inventory` |
| **T13t** | `takingTheStairs = false` | `:127` | `room-transitions` |

**CORRECTIONS to `bands-grease` §F**, whose `RoomGraphics.c` citations are uniformly one low while its
`Grease.c`/`RubberBands.c` citations are exact, so two of its anchors name the wrong call: its
"`KillAllBands() :52`" is actually `ZeroDinahs()`, and its "`ZeroTriggers() :54`" is actually
`ZeroMirrorRegion()`. It also omits `ZeroMirrorRegion` entirely, gives `ListAllLocalObjects` as
`:71` (it is `:72`), the central room as `:117-120` (it is `:118-121`), `DrawLocale` as `:43-129`
(it is `:44-130`), and `ZeroFlamesAndTheLike` as `:50` (it is `:51`). Take `RoomGraphics.c` and
`Render.c` numbers from `hotspot-pipeline`/`rewards-switches-triggers`/`background-animations`/`room-transitions`/`live-state-inventory` and `Grease.c`/`RubberBands.c` numbers from `bands-grease`.

Two orderings inside T13 are load-bearing:
- **T13a before T13l.** `ZeroFlamesAndTheLike` zeroes `numChimes` (`DynamicMaps.c:796`) and
  `ListAllLocalObjects` → `ListOneRoomsObjects` → `CreateActiveRects` re-increments it
  (`ObjectRects.c:1052`). Reverse them and the wind chimes go permanently silent.
  **CORRECTION to `background-animations`:** `numChimes` counts **central-room** chimes only, not all nine rooms —
  `CreateActiveRects` is called under `if ((where == kCentralRoom) && (IsThisValid(roomNum, n)))`
  at `Objects.c:283-284`. `hotspot-pipeline` is right.
- **T13l before T13n/o/p.** `ListAllLocalObjects` builds `masterObjects[]` before any
  `DrawARoomsObjects` call, so the `dynaNum` write-back at `ObjectDrawAll.c:953-961` has a list to
  write into.

**CORRECTION to `live-state-inventory`'s summary of the two orders.** `masterObjects[]` is built Central, E, W, N, NE,
SE, S, SW, NW (`Objects.c:312-328`); the draw order is NW, NE, N, SW, SE, S, W, E, Central
(`RoomGraphics.c:80-121`). `live-state-inventory` calls the second the "**reverse**" of the first. It is not: reversing
the build order gives NW, SW, S, SE, NE, N, W, E, Central. The only true statement is "central is
first in the build order and last in the draw order". This matters because draw order fixes
`dinahs[]`/`savedMaps[]`/`grease[]` slot indices, and slot index decides both `HandleDynamics`'
iteration order (which mover wins the `hDesiredVel` assignment race, LB1) and which entries are
dropped at saturation. Adopt `background-animations`'s framing: state the two orders separately.

#### 1.4.5 What the ordering guarantees

`ReadyLevel` is **T4, at the end of `MoveRoomToRoom`**. Therefore:

- `topOpen`, `bottomOpen`, `leftThresh`, `rightThresh`, `thisBackground`, `thisTiles[]`,
  `shadowVisible`, `localNumbers[]`, `isStructure[]`, `masterObjects[]`, `hotSpots[]`, `nHotSpots`,
  `dinahs[]`, `grease[]`, `savedMaps[]` and the animation tables are **all correct before the next
  frame's F6c**. That is the guarantee 1.4 needs and it holds.
- They are **not** correct *during* `MoveRoomToRoom`. `OffsetGlider` and the `enteredRect`
  assignments in T2 run against the old room's thresholds — which is harmless only because neither
  reads them.
- F7 and F8 sit *between* F6c and F9, so they see the post-transition room if the transition came
  from an escape check, and the pre-transition room if it is about to come from F9.

### 1.5 `RenderFrame` runs twice on a transition frame

No report follows this through, one report states the opposite of the truth, and it must be settled
before anyone writes the render loop.

**The fact.** `COMPILEQT` is defined (`GliderDefines.h:15`), so all four `RenderFrame()` calls in
`Transit.c` are live: `:303` (`MoveRoomToRoom`), `:341` (`TransportRoomToRoom`), `:380`
(`MoveDuctToDuct`), `:419` (`MoveMailToMail`). Each sits **before**, not inside, the
`if ((thisMac.hasQT) && (hasMovie) && (tvInRoom) && (tvOn))` that follows it, so each is
unconditional. In the three link transitions it is also **outside** the `if (!sameRoom)` guards
(`:334-338` vs `:341`), so it fires even on a same-room transport. And `RenderFrame` runs again at
F10b (`Play.c:469`/`:494`) later in the same frame.

**Why it is not idempotent, and what double-steps.** Because `evenFrame` flips only at
`Play.c:435` (LB13), the second call takes the **same** branch of `Render.c:649-652` as the first:

- R4: `RenderFlames` (or `RenderStars`) executes **twice in a row**, and the other **zero** times.
  All three flame loops mutate `mode` **and** `src.top`/`src.bottom` before wrapping
  (`Render.c:200-256`), so this is real state, not a redraw.
- R3: `clockFrame++` (`Render.c:270`) executes twice, so pendulum phase and the `kTikSound`/
  `kTokSound` schedule drift on every room change — **but only in a room that has a pendulum**,
  because `clockFrame++` is after `if (numPendulums == 0) return;` at `:267-268`.
- R2: `HandleGrease` runs twice.
- R12: the busy-wait runs twice, so a transition frame costs two frame budgets.
- R10/R7/R6: `RenderShreds`, `RenderSparkles`, `RenderFlyingPoints` **would** double-step, but on
  the `MoveRoomToRoom` path T13a sets `numShredded = 0` (`DynamicMaps.c:795`) and T14 sets
  `numSparkles = numFlyingPts = 0` (`Render.c:682-688`) first. **On a same-room transport, duct or
  mail trip `ReadyLevel` is skipped entirely** (`Transit.c:334-335`, `:337-338`, `:373-374`,
  `:412-413`) while `RenderFrame` at `:341`/`:380`/`:419` still runs — so there shreds, sparkles and
  flying points **do** double-step.
- None of the four `Transit.c` `RenderFrame` calls is followed by `HandleDynamicScoreboard`, so
  `displayedScore` does not roll on the transition render.

**CORRECTION to `background-animations` §B**, which asserts the opposite: "*a candle and a star in the same room never
update on the same frame and **neither ever updates twice in a row**.*" The second clause is false,
in exactly the subsystem that owns it. Delete it. The first clause is true.

**NORMATIVE DECISION for Stage 1.5: reproduce it.** Both `RenderFrame` call sites are in the port,
in the same positions. The behaviour is observable in animation phase, in the pendulum sound
schedule and in frame pacing, and a fidelity test that walks a glider between rooms will see it.
The Go `RenderFrame` must therefore be a plain method with no "already rendered this frame" guard,
and the transition functions must call it exactly where the C does.

### 1.6 The four steps no subsystem report owns

§1 owns these. They are specified here rather than cross-referenced.

#### 1.6.1 F3 — `HandlePlayEvent` and `doBackground`

`doBackground` is a user preference, loaded from prefs at `Main.c:118`, forced false at `Main.c:186`
and `Settings.c:1228`, saved back at `Main.c:271`. **When it is false the game processes no events
at all during play.** F3 is not gated on `gameOver` or `playing`.

`HandlePlayEvent` (`Play.c:387-426`) handles exactly two event kinds:

```
if (WaitNextEvent(everyEvent, &theEvent, sleep, nil))            // :393, sleep = 2 (:391)
    if ((theEvent.what == updateEvt) && ((WindowPtr)theEvent.message == mainWindow))  // :395-396
        GetPort; SetPortWindowPort(mainWindow); BeginUpdate(mainWindow);
        CopyBits(workSrcMap -> mainWindow, &justRoomsRect, &justRoomsRect, srcCopy, nil);  // :401-403
        RefreshScoreboard(kNormalTitleMode);                                               // :404
        EndUpdate(mainWindow); SetPort(wasPort);
    else if ((theEvent.what == osEvt) && (theEvent.message & 0x01000000))                 // :408
        if (theEvent.message & 0x00000001)      // resume
            switchedOut = false; ToggleMusicWhilePlaying(); HideCursor();                  // :412-414
        else                                    // suspend
            InitCursor(); switchedOut = true; ToggleMusicWhilePlaying();                   // :419-421
```

Three facts a porter needs:
- **No `keyDown` handling at all.** Keyboard state is read by `GetInput` via `GetKeys`, not through
  the event queue. Pause and command-key handling live inside `GetInput` (1.4's `DoPause`,
  `DoCommandKey`).
- `sleep = 2` makes `WaitNextEvent` a **second, independent 2-tick pacing source** stacked on R12's
  busy-wait. With `doBackground` true the frame can therefore be paced by either.
- The `do { ... } while (switchedOut)` halts the simulation without advancing `gameFrame` or
  `evenFrame`, because F1 and F2 already ran.

Stage 1.5 has no Mac event queue. Implement F3 as: if `doBackground` is false, nothing; if true, a
single hook that (a) may set `switchedOut`, (b) spins while `switchedOut`, (c) on the equivalent of
`updateEvt` re-blits `justRoomsRect` from the work map. Record the divergence in the pacing model
explicitly, since the port's frame clock will not be tick-based.

#### 1.6.2 F10c — `HandleDynamicScoreboard`, `Scoreboard.c:72-132`

Mutates live state that no report's `liveState` section claims.

```
if (theScore > displayedScore)                                   // :80
    if (doRollScore)                                             // :82
        displayedScore += kScoreRollAmount;                      // :84
        if (displayedScore > theScore) displayedScore = theScore; // :85-86
    else
        displayedScore = theScore;                               // :89
    PlayPrioritySound(kScoreTikSound, kScoreTikPriority);        // :91  -- OUTSIDE the inner if/else
    QuickScoreRefresh();                                         // :92  -- OUTSIDE, and last port set
whosTurn = gameFrame & 0x00000007;                               // :95
switch (whosTurn)                                                // :96
    case 0: if ((foilTotal > 0) && (foilTotal < kFoilLow)) QuickFoilRefresh(false);        // :98-101
    case 1: if ((batteryTotal > 0) && (batteryTotal < kBatteryLow)) QuickBatteryRefresh(true);
            else if ((batteryTotal < 0) && (batteryTotal > kHeliumLow)) QuickBatteryRefresh(true);
    case 2: if ((bandsTotal > 0) && (bandsTotal < kBandsLow)) QuickBandsRefresh(false);    // :110-113
    case 4: (battery, as case 1 but QuickBatteryRefresh(false))                            // :115-120
    case 5: if ((foilTotal > 0) && (foilTotal < kFoilLow)) QuickFoilRefresh(true);         // :122-125
    case 7: if ((bandsTotal > 0) && (bandsTotal < kBandsLow)) QuickBandsRefresh(true);     // :127-130
    // no case 3, no case 6, no default -- deliberately
```

Live state: `displayedScore`, plus a sound every rolling frame. Thresholds are local `#define`s at
`:74-77`: `kFoilLow 2`, `kBatteryLow 17`, `kHeliumLow -38`, `kBandsLow 2`. The gaps at
`whosTurn == 3` and `6` are what make the low-supply indicators blink rather than flicker; do not
fill them in. LB12 depends on `QuickScoreRefresh` at `:92` being the frame's last port set.

#### 1.6.3 F11 — the game-over tail, `Play.c:499-553`

```
if (gameOver)                                                    // :499  -- at loop-body level,
{                                                                //          OUTSIDE both arms
    countDown--;                                                 // :501  -- AFTER F10b/F10c, so the
    if (countDown <= 0)                                          // :502     final frame still renders
    {
        GetGWorld(&wasCPort, &wasWorld);                          // :507
        HideGlider(&theGlider);                                   // :509
        RefreshScoreboard(kNormalTitleMode);                      // :510
#if BUILD_ARCADE_VERSION                                          // :512  -- compiled IN
        SetGWorld(boardSrcMap, nil); PaintRect(&boardSrcRect);    // :515-516
        CopyBits(boardSrcMap -> mainWindow, &boardSrcRect, &boardDestRect, srcCopy, 0L);  // :518-520
        hOffset = (boardSrcRect.right >= 640)
                    ? (RectWide(&boardSrcRect) - kMaxViewWidth) / 2 : -576;               // :527-530
        DrawPicture(GetPicture(kScoreboardPictID), <picFrame at hOffset>);                // :531-540
#endif
        if (mortals < 0) DoDiedGameOver();                        // :546-547
        else             DoGameOver();                            // :548-549
        SetGWorld(wasCPort, wasWorld);                            // :551
    }
}
```

- `countDown` is seeded by `FlagGameOver` (`GameOver.c:237-242`): `gameOver = true;`
  `countDown = kNumCountDownFrames;` (`= 16`, `GameOver.c:17`) `SetMusicalMode(kPlayWholeScoreMode);`.
  `countDown` is **not** in `NewGame`'s reset block (`Play.c:112-118`) — see `live-state-inventory`.
- `countDown--` runs **after** F10b and F10c in the same frame, so the last countdown frame still
  renders. `HideGlider` and `RefreshScoreboard` run inside a `GetGWorld`/`SetGWorld` save-restore
  pair (`:507`, `:551`).
- The terminal branch is **`mortals < 0` means *died*, `mortals >= 0` means *won***. Because
  `OffAMortal` decrements to `-1` on the last death, `mortals == -1` is the died case.
- `DoDiedGameOver`/`DoGameOver` end the game; both are out of Stage 1.5's scope, but `playing` and
  `quitting` are the loop's exit condition and must be in the port.

#### 1.6.4 Two things DO recompose the room from inside the F6c-i sweep

`room-transitions`'s "no hot-spot action changes the room synchronously" (§1.4.1 item 1) is correct about *transit*,
and it is exactly the sentence that would let a porter miss both of these.

**(a) `RedrawRoomLighting()`, `RoomGraphics.c:434-461`, called from `Interactions.c:1083`.** That
call is inside `HandleSwitches` ← `HandleHotSpotCollision` ← the `for (i = 0; i < nHotSpots; i++)`
sweep. It performs a full central-room recomposition **mid-sweep, before F10b**:

```
roomV  = rooms[thisRoomNumber].floor                             // :440-443
wasLit = numLights > 0                                           // :445  (still the CENTRAL room's,
                                                                 //        left there by T13p)
numLights = GetNumberOfLights(localNumbers[kCentralRoom])        // :446
isLit  = numLights > 0                                           // :447
if (wasLit != isLit)                                             // :448  -- only on a 0 <-> non-0 flip
    DrawRoomBackground(localNumbers[kCentralRoom], kCentralRoom, roomV)   // :450
    DrawARoomsObjects(kCentralRoom, true)                        // :451  -- the ONLY redraw==true site
    DrawLighting()                                               // :452
    UpdateOutletsLighting(localNumbers[kCentralRoom], numLights) // :453  -- its ONLY call site
    if (numNeighbors > 3) DrawFloorSupport()                     // :455-456
    RestoreWorkMap()                                             // :457
    AddRectToWorkRects(&localRoomsDest[kCentralRoom])            // :458
    shadowVisible = IsShadowVisible()                            // :459
```

It is **not** a table-rebuild hazard, and the reason is `redraw == true`: every `Add*` registration
and the `dynaNum` correlation loop in `DrawARoomsObjects` sits behind a `!redraw` test — 19 of them,
from `if ((!redraw) && (neighbor == kCentralRoom))` at `ObjectDrawAll.c:452` to `if (!redraw)` at
`:953` (the correlation loop, which writes `masterObjects[n].dynaNum` at `:959`). So `nHotSpots`,
`hotSpots[]` and `dinahs[]` are untouched and the sweep's `i` stays valid. Note the two guard forms
are not interchangeable: the `(!redraw) && (neighbor == kCentralRoom)` form (`:452`, `:648`, `:797`,
`:808`, `:819`, `:830`, `:841`, `:852`, `:868`, `:884`, `:907`) means those object classes register
**only** for the central room, while the bare `!redraw` form (`:664`, `:697`, `:721`, `:738`, `:754`,
`:770`, `:786`) registers for all nine. Which class falls in which group is `dynamics-core`'s and `background-animations`'s to state. But `RestoreWorkMap` at `:457` wipes every animation out of the work map
mid-frame, and only the central rect is pushed to screen, so a neighbour-room flame stays erased on
screen until its next even frame. `room-transitions`'s reassurance is right for a reason it does not give; state the
reason.

`shadowVisible` therefore has **four** write sites, not three: `DrawLocale` `:126`,
`RedrawRoomLighting` `:459`, `FlagGliderNormal` (`Modes.c:361`), `MoveGliderShredding`
(`Player.c:1286`). `room-transitions` is right and `live-state-inventory` is missing `:459` — and `:459` matters because it means a
light switch re-derives `shadowVisible` mid-play, not only a room change or a mode change.

**(b) `DrawLocale()` at `Play.c:814`, reachable mid-sweep.** The chain is:
`HandleRewards` `case kStar` → `numStarsRemaining--` (`Interactions.c:942`) →
`if (numStarsRemaining <= 0) FlagGameOver(); else DisplayStarsRemaining();` (`:943-946`) →
`Banner.c:233-234`: `if (WaitForInputEvent(30)) RestoreEntireGameScreen();` → `Play.c:800-820`,
which does `HideCursor`, `SetPort(mainWindow)`, `PaintRect(thisMac.screen)`, **`DrawLocale()`**
`:814`, `RefreshScoreboard` `:815`.

So `DrawLocale` — hence T13l, hence `nHotSpots = 0` and a full `masterObjects[]`/`hotSpots[]`
rebuild — **can run underneath `CheckForHotSpots`' `for (i = 0; i < nHotSpots; i++)` loop**, at the
iteration that collected the star. And `who->isOn = false;` at `Interactions.c:950` then writes
through a `hotSpots[]` pointer into the **rebuilt** table at the same index. The star's own hot spot
no longer exists after the rebuild (its `data.c.state` is now 0, so `IsThisValid` rejects it), so the
indices shift.

**This is a `DrawLocale` without `NilSavedMaps` (T10), without `DetermineRoomOpenings` (T12) and
without `InitGarbageRects` (T14)** — a genuinely different reset from `ReadyLevel`'s. Consequences:
the `savedMaps` slots the old tables held are leaked, and the pending dirty rects survive.
So `Room` cannot be constructor-only in Go: the object-graph rebuild must be a method separable
from construction, and `live-state-inventory` must enumerate which fields survive it.

**CORRECTION to both critics on the trigger.** `WaitForInputEvent` (`Utilities.c:439-479`) returns
`didResume`, which is set **only** in the resume arm of an `osEvt` (`:465-466`) — a keypress or
mouse click at `:459-460` sets `waiting = false` but leaves `didResume` false, as does the 30-second
timeout at `:474-475`. So `RestoreEntireGameScreen` fires **only if the player switches out of and
back into the application during the stars-remaining banner**, not on a keypress. It is rare, not
unreachable, and it is reachable during normal play. Also note `DelayTicks(60)` at `Banner.c:232`
precedes the wait, so this path blocks the frame for at least one second and up to thirty; when it
returns, `nextFrame` is far in the past and R12 waits zero ticks.

### 1.7 What this means for the Go surface

Stage 1.5 must expose one function shaped like F1..F11, and it must be a single flat method whose
statement order is this list. Concretely:

- The frame is a method on the world (`func (w *World) Tick(k Keys)`), not a pipeline of independent
  systems, because F5's writes are consumed by F9 and F6c's writes are consumed by F9, and no
  ordering-agnostic scheduler can express that.
- `*World` implements `player.Env` directly. The 1.4 entry points are called from `Tick` at exactly
  F6a, F6b, F6c-ii..iv (via `HandleInteraction`) and F9a, F9b.
- `RenderFrame` is a method with **no** re-entrancy guard (§1.5), and the four transition functions
  call it where the C does.
- `HandleGrease` is called from inside `RenderFrame`, not from `Tick` (LB9). Resist moving it.
- F7 and F8 are called from `Tick` unconditionally, outside the `gameOver` test (LB3).
- `Env` needs one addition no report lists: `WebGlider` reads the global `evenFrame` at
  `Interactions.c:1756` to gate the twang, so if `hotspot-pipeline`'s recommendation to move `WebGlider` into
  `player` is taken, `Env` needs an `EvenFrame() bool`. (`dynamics-movers` cites the read as `:1755`; `:1755` is
  the brace, `:1756` is the read.)

### 1.8 Not settled here

- **`GetDemoInput` / demo playback is declared OUT OF SCOPE for Stage 1.5**, in writing, because it
  sits on the same line of the frame as `GetInput` (`Play.c:478-479`) and would otherwise look like
  an oversight. `demoIndex` is reset at `Play.c:114`, `demoGoing` gates the branch, `demoData` is
  allocated at `StructuresInit2.c:287`. The 1P arm ports as `GetInput` unconditionally, with the
  `demoGoing` branch left as a documented hole.
- **`MoviesTask` (F10a) and the movie restarts (T8, T11) are stubs.** No QuickTime. `tvInRoom`,
  `tvWithMovieNumber` and `hasMovie` are still tracked because T13i and T11 write them; `tvOn` is
  never reset by `DrawLocale` and its staleness is `live-state-inventory`'s problem, not §1's.
- **`numTempManholes` (T13f) is render-side, not Stage 1.5.** It is written by
  `AddTempManholeRect` (`Objects.c:349-362`) from `ObjectDrawAll.c:277` and consumed only by
  `RoomGraphics.c:277-371`. Recorded here so the next audit does not re-derive it.
- **The pacing model.** The C has two pacing sources (R12's tick busy-wait, and F3's
  `WaitNextEvent(sleep = 2)` when `doBackground` is true) and Stage 1.5 will have neither. §1.6.1
  specifies the divergence but does not choose the port's clock; that belongs in whatever section
  owns the host loop. **UNVERIFIED:** I did not measure how much the two sources actually interact
  on real hardware, only that both are present and both nominally 2 ticks.
- **`rightClip`/`leftClip` are one shared global pair in the C** (`short rightClip, leftClip,
  transRoom;`, `Player.c:52`), written at `Modes.c:150`, `:536`, `:567` and read at `Player.c:407`,
  `:463`, `:536`. Stage 1.4 made them per-glider (`internal/game/player/glider.go:87`, whose comment
  admits it). Whether the sharing is observable — two gliders on staircases in the same room — is a
  decision `room-transitions`/`live-state-inventory` must record; §1 only notes that both writes and both reads happen inside F9, so
  the frame ordering does not by itself resolve it.

---

## 2. The state, and who owns it

Stage 1.5 is the stage where the port stops being a pure function of a house file. Everything in
this section is *live* state: memory that exists only while a game is being played, that some
function writes and some other function reads a frame or a room later. The original keeps all of
it in file-scope globals, and the single most important fact about those globals is that **their
reset scopes are not uniform and are not documented anywhere in the source**. Two counters
declared on the same line of `DynamicMaps.c` are reset by different functions at different
lifetimes. A porter who groups state by where it is declared will get it wrong; the only reliable
grouping is *by the function that clears it*.

So this section inventories the live state by reset site, then assigns ownership in Go on that
basis. The reset site is the design input; the declaration site is a red herring.

### 2.1 The five lifetimes, and the exact C site that ends each

There are exactly five, and each has one function that is the authority on it.

| Lifetime | The C site | What it means |
|---|---|---|
| session | `CreatePointers` (`StructuresInit2.c:187-299`) | allocated once at launch, never freed |
| new game | `PlayGame`'s prologue (`Play.c:102-118`) plus `InitGlider` (`Play.c:307-366`) | one press of "New Game" |
| new house | `SetObjectsToDefaults` (`Play.c:603-708`), `InitHouseData`-side clears (`House.c:142-151`), `ReadHouse`'s flag decode (`HouseIO.c:416-424`) | one house file being adopted |
| room change | `DrawLocale`'s reset block (`RoomGraphics.c:51-60`) reached through `ReadyLevel` (`RoomGraphics.c:402-418`) | the glider crossing into a new room |
| new life | `OffAMortal` (`Player.c:1484-1604`) | one glider being spent |
| never | BSS zero at launch, then whatever wrote last | the trap category |

Three of these need their exact shape stated, because the shape is what a porter gets wrong.

`PlayGame`'s prologue is not one block. `SetObjectsToDefaults` is *guarded*:

```
if (mode != kResumeGameMode)          // Play.c:102
    SetObjectsToDefaults()            // :103
HideCursor()                          // :104
if (mode == kResumeGameMode)          // :105
    SetHouseToSavedRoom()             // :106
else if (mode == kNewGameMode)        // :107
    SetHouseToFirstRoom()             // :108
DetermineRoomOpenings()               // :109
NilSavedMaps()                        // :110

gameFrame = 0                         // :112
numBands = 0                          // :113
demoIndex = 0                         // :114
saidFollow = 0                        // :115
otherPlayerEscaped = kNoOneEscaped    // :116
onePlayerLeft = false                 // :117
playerSuicide = false                 // :118
```

Note the asymmetry: `:112-118` run unconditionally, on a resumed game as well as a new one, but
`SetObjectsToDefaults` does not — a resumed game keeps the object states saved in `smallGame`.
Note also what is *absent* from `:112-118`: `playerDead` is never reset here. It is only ever
written at `Player.c:1508`, and `onePlayerLeft` at `:1507` is what gates every read of it, so the
stale value is unobservable — but a Go port that asserts on `playerDead` outside an
`OnePlayerLeft()` guard will see garbage from the previous game.

`ReadyLevel` is the whole of the room-change lifetime, and it is only four statements:

```
void ReadyLevel (void)                          // RoomGraphics.c:402
{
    NilSavedMaps()                              // :404
#ifdef COMPILEQT
    if ((thisMac.hasQT) && (hasMovie) && (tvInRoom))   // :407
    {
        tvInRoom = false                        // :409
        tvWithMovieNumber = -1                  // :410
        StopMovie(theMovie)                     // :411
    }
#endif
    DetermineRoomOpenings()                     // :415
    DrawLocale()                                // :416
    InitGarbageRects()                          // :417
}
```

`COMPILEQT` is defined (`GliderDefines.h:15`), so the movie arm is live. `DrawLocale`'s own
prologue is the ten statements that do the real clearing:

```
void DrawLocale (void)          // RoomGraphics.c:43
{
    ZeroFlamesAndTheLike()      // :51
    ZeroDinahs()                // :52
    KillAllBands()              // :53
    ZeroMirrorRegion()          // :54
    ZeroTriggers()              // :55
    numTempManholes = 0         // :56
    FlushAnyTriggerPlaying()    // :57
    DumpTriggerSound()          // :58
    tvInRoom = false            // :59
    tvWithMovieNumber = -1      // :60
```

`:57` matters more than it looks: entering a room silences whichever of the three sound channels
is holding a trigger sound (`Sound.c:87-127`). The sound throttle therefore has room-change-scoped
state, which is why it cannot be a package-level singleton in the port (see §2.12).

The `DrawLocale`-clears-it list is not the same as the `ReadyLevel`-clears-it list, and neither is
the same as "room-scoped", because **`DrawLocale` is called from a second place that is not
`ReadyLevel`**. `RestoreEntireGameScreen` (`Play.c:800-820`, sole caller `Banner.c:234`) calls
`DrawLocale` at `Play.c:814` with no `NilSavedMaps` before it and no `InitGarbageRects` after it.
That single fact decides the shape of the Go API in §2.10: the locale rebuild has to be a separable
method, not something a constructor does.

### 2.2 The bounds, and the four `#define`s that are not where you expect

Every table is fixed-capacity, and `NewPtr` does not zero, so every capacity is observable
behaviour rather than an implementation detail: the eleventh grease patch in a room simply does
not exist, and `AddGrease` returning `-1` is a documented outcome the callers branch on.

`GliderDefines.h:249-266` holds most of them:

| `#define` | Value | Site | Table |
|---|---|---|---|
| `kMaxRoomObs` | 24 | `GliderDefines.h:250` | objects per room in the house file |
| `kMaxSparkles` | 3 | `:251` | `sparkles` |
| `kMaxFlyingPts` | 3 | `:253` | `flyingPoints` |
| `kMaxCandles` | 20 | `:255` | `flames` |
| `kMaxTikis` | 8 | `:256` | `tikiFlames` |
| `kMaxCoals` | 8 | `:257` | `bbqCoals` |
| `kMaxPendulums` | 8 | `:258` | `pendulums` |
| `kMaxHotSpots` | 56 | `:259` | `hotSpots` |
| `kMaxSavedMaps` | 24 | `:260` | `savedMaps` |
| `kMaxRubberBands` | 2 | `:261` | `bands` |
| `kMaxGrease` | 16 | `:262` | `grease` |
| `kMaxStars` | 4 | `:263` | `theStars` |
| `kMaxShredded` | 4 | `:264` | `shreds` |
| `kMaxDynamicObs` | 18 | `:265` | `dinahs` |
| `kMaxMasterObjects` | 216 | `:266` | `masterObjects` (the comment says `kMaxRoomObs * 9`) |

Four more are file-local, which is exactly why a porter working from that header alone will miss
them:

| `#define` | Value | Site | Table |
|---|---|---|---|
| `kMaxGarbageRects` | 48 | `Render.c:20` | `work2MainRects`, `back2WorkRects` |
| `kMaxTriggers` | 16 | `Triggers.c:12` | `triggers` |
| `kMaxTempManholes` | 8 | `Objects.c:12` | `tempManholes` |
| `kInitialGliders` | 2 | `Play.c:19` | seeds `mortals` |

And one cap is a bare literal with no `#define` at all: `if ((sayIt) && (saidFollow < 3))` at
`Modes.c:462`. Stage 1.4 already named it `MaxSaidFollow` (`internal/game/player/consts.go:221-222`),
which is the right call, but the doc comment there must keep saying the C has a literal, or a
later reader will go looking for a header constant that does not exist.

Two of these bounds are not the safe kind.

**`kMaxHotSpots` 56 is not provably sufficient.** `kMaxRoomObs` is 24, so an obvious argument says
24 objects cannot fill 56 slots. The argument fails: a tall `kTaper` produces **three** hot spots,
not one. `ObjectRects.c:407-421`:

```
if (<taller than kDeadlyFlameHeight>)     // :409-ish
{
    AddHotSpot(..., kLiftIt)              // :410
    AddHotSpot(..., kBurnIt)              // :413
}
else
    AddHotSpot(..., kBurnIt)              // :416
AddHotSpot(..., kDissolveIt)              // :421
```

The `kDissolveIt` spot at `:421` is outside the if/else, so the tall case emits three. 24 tapers
is 72 > 56. Whether any shipped house has 24 tall tapers in one room is a different question and
the answer is almost certainly no — but the port must reproduce `AddHotSpot`'s
return-`-1`-when-full contract rather than growing a slice, because the fifty-seventh hot spot
being silently absent is the original's behaviour. UNVERIFIED: I did not census the 22 shipped
houses for a room that actually exceeds 56.

**`kMaxDynamicObs` 18 versus `kMaxRoomObs` 24.** Nine local rooms each contribute their dynamic
objects to one 18-slot table, so overflow is trivially reachable in principle. UNVERIFIED: I did
not establish whether `AddDynamicObject` is called for the eight neighbour rooms or only the
central one, which is what decides how easily the 18 fills; the `dynamics-movers` and `bands-grease` reports own that call graph. §6.2 item 1 records the contradiction between the three censuses and the composition census that would settle it.

### 2.3 Room-scoped: the tables `DrawLocale` rebuilds

These are the ones `DrawLocale`'s prologue clears at `RoomGraphics.c:51-60` and its body refills.
"Builder" below is the function that appends; "readers" is who consumes it during the frame loop.

| Table | C type and size | Bound | Built by | Cleared by | Principal readers |
|---|---|---|---|---|---|
| `hotSpots` | `hotObject *` — `Rect bounds; short action, who; Boolean isOn, stillOver, doScrutinize` = 16 bytes (`GliderStructs.h:218-225`) | 56 | `AddHotSpot` out of `ObjectRects.c`, counted by `nHotSpots` | `nHotSpots = 0` (`Objects.c:307`, inside `ListAllLocalObjects`) | `CheckForHotSpots`/`HandleHotSpotCollision` (`Interactions.c`), `HandleGrease` (`Grease.c:60,:61,:68,:95,:102` — it *writes* them), `TriggerSwitch` (`Trip.c:146-149`) |
| `masterObjects` | `objDataType *` — 7 shorts + `objectType` (12 bytes, `GliderStructs.h:105`) = **26 bytes** (`:322-332`) | 216 | `ListAllLocalObjects` (`Objects.c`) | `numMasterObjects = 0` (`Objects.c:305`) | the link resolution in the four transit starts, `ArmTrigger` (`Triggers.c:46-50`), `FireTrigger` (`:110`) |
| `dinahs` | `dynaType *` — 2 Rects + 8 shorts + 2 Bytes + 2 Booleans = 36 bytes (`:310-320`) | 18 | `AddDynamicObject` (`Dynamics3.c:187`) | `ZeroDinahs` (`Dynamics3.c:160-180`) | the six `Dynamics*.c` handlers, the fourteen `Toggle*`/eight `Trigger*` in `Trip.c` |
| `savedMaps` | `savedType[24]` — `Rect dest; GWorldPtr map; short where, who` = 16 bytes (`:227-233`) | 24 | `BackUpToSavedMap` (`DynamicMaps.c:70-93`), `ReBackUpSavedMap` (`:100`) | `NilSavedMaps` (`DynamicMaps.c:46-62`), called from `ReadyLevel:404` and `Play.c:110`/`:253` | `RestoreFromSavedMap`, every flame/star/grease renderer |
| `flames`, `tikiFlames`, `bbqCoals`, `pendulums`, `theStars` | `flamePtr`/`pendulumPtr`/`starPtr` | 20, 8, 8, 8, 4 | the `Add*` in `ObjectDrawAll.c` | `ZeroFlamesAndTheLike` (`DynamicMaps.c:787-797`) | the animation pass (`background-animations`) |
| `grease` | `greaseType *` — `Rect dest; 9 shorts; Boolean isRight` = 28 bytes (`:287-295`) | 16 | `AddGrease` (`Grease.c:206`), returns `-1` when full | `numGrease = 0` (`DynamicMaps.c:793`) | `HandleGrease` (`Grease.c`), `RenderGrease` (`:282`) |
| `shreds` | `shredType *` — `Rect bounds; short frame` = 10 bytes (`:304-308`) | 4 | `AddAShreddedGlider` | `numShredded = 0` (`DynamicMaps.c:795`) | the shred renderer |
| `bands` | `bandType *` — `Rect dest; short mode, count, hVel, vVel` = 16 bytes (`:274-279`) | 2 | `AddBand` (`RubberBands.c:256`) | `KillAllBands` (`RubberBands.c:307-318`) — zeroes only `.mode` and `numBands` | `HandleBands`, `RenderBands` |
| `triggers` | `trigType[16]` — `short object, room, index, timer; short what; Boolean armed` (`Triggers.c:14-19`) | 16 | `ArmTrigger` (`Triggers.c:38-52`) | `ZeroTriggers` (`Triggers.c:198-205`) — clears only `.armed`, and there is **no** `numTriggers` | `HandleTriggers` (`:85-92`), `FireTrigger` (`:105-190`) |
| `tempManholes` | `Rect[8]` (`Objects.c:72`) | 8 | `Objects.c:355` under `if (numTempManholes < kMaxTempManholes)` | `numTempManholes = 0` (`RoomGraphics.c:56`) | the floor renderer |
| `mirrorRgn` | `RgnHandle` | — | `AddToMirrorRegion` (`Render.c:745-761`) | `ZeroMirrorRegion` (`Render.c:765-771`) | the mirror double-blit |
| `work2MainRects`, `back2WorkRects` | `Rect[48]` each (`Render.c:35-36`) | 48 | `AddRectToWorkRects`, `AddRectToBackRects` | `InitGarbageRects` (`Render.c:675-691`) | `CopyRectsQD` at end of frame |

Four notes on these that a straight transcription loses.

**`ZeroDinahs` does not zero a `dinahs` entry.** It writes `type`, `dest`, `whole`, `hVel`, `vVel`,
`count`, `frame`, `timer`, `position`, `room`, `byte0`, `active` (`Dynamics3.c:164-176`) and leaves
`byte1` and `moving` alone. That is safe only because every one of `AddDynamicObject`'s case
bodies sets both explicitly (`Dynamics3.c:212-213`, `:239-240`, `:259-260`, `:279-280`, `:302-303`,
`:322-323`, `:345-346`, `:365-366`, `:386-387`, `:407-408`, `:432-433`, `:460-461`, `:485` and
`:492`, `:511-512`, `:541-542`). A Go port that clears the slice with `Dynamic{}` is *more* correct
than the C and no behaviour changes; a Go port that reuses a slot without going through
`AddDynamicObject` would diverge.

**`triggers[].what` is dead, which makes a real indexing bug unobservable.**
`Triggers.c:47` sets `triggers[where].object = masterObjects[whoLinked].objectLink` — a
*house room object number*, 0..23. `Triggers.c:50` then does
`triggers[where].what = masterObjects[triggers[where].object].theObject.what`, indexing the
216-slot master table with that 0..23 object number. It is in bounds but it is the wrong slot
whenever the trigger's target is not in the central room. It does not matter: `.what` is written
at `:50` and read nowhere — `HandleTriggers` reads `.armed`/`.timer` (`:85-91`) and `FireTrigger`
reads `.index`, `.room`, `.object` (`:105`, `:114`, `:179-184`). **The Go `Trigger` struct should
omit the field entirely**, with a comment citing `:50` so nobody re-adds it from the C.

**`bands[].mode` has two dead values.** `KillAllBands` writes `0` (`RubberBands.c:317`) but
`kKillBandMode` is `-1` (`RubberBands.c:14`). So `0` is "slot never used" and `-1` is "retire this
band during this frame's sweep"; they are not interchangeable and a Go `bool Active` loses the
distinction.

**`savedMaps` has a genuine cross-translation-unit type mismatch in the original.** `Objects.c:73`
defines it as `savedType savedMaps[kMaxSavedMaps]` — a 24-element BSS array. `StructuresInit2.c:40`
declares it `extern savedPtr savedMaps` — a pointer. `Grease.c:30`, `Render.c:54` and
`DynamicMaps.c:36` all declare it correctly as `extern savedType savedMaps[]`. The 68K linker
resolves by name without type checking, so `CreatePointers` at `StructuresInit2.c:233` writes a
`NewPtr` block address into the first four bytes of the real array — which, because `dest` is the
first member of `savedType`, is `savedMaps[0].dest.top` and `.left` — and then
`StructuresInit2.c:237-238` nils the 24 `.map` fields of the *leaked heap block* rather than the
array. The damage is confined to `savedMaps[0].dest.top/.left`, and `BackUpToSavedMap` overwrites
`.dest` before any read (`DynamicMaps.c:79`), so nothing observable happens. **The port must not
treat `savedMaps` as session-allocated**: it is a plain array with a BSS lifetime, and the first
`NilSavedMaps` (`Play.c:110`) sees all 24 `.map` already nil.

### 2.4 Room-scoped: the scalars

| Global | C type and decl | Written by | Reset at room change by | Read by |
|---|---|---|---|---|
| `nHotSpots` | `short` `Objects.c:76` | `AddHotSpot` | `Objects.c:307` | every hot-spot sweep |
| `numMasterObjects`, `numLocalMasterObjects` | `short` `Objects.c:76` | `ListAllLocalObjects` | `Objects.c:305` | link resolution |
| `numDynamics` | `short` `Dynamics3.c:19` | `AddDynamicObject` | `Dynamics3.c:179` | the dynamics frame loop |
| `numFlames`, `numTikiFlames`, `numCoals`, `numPendulums`, `numGrease`, `numStars`, `numShredded`, `numChimes` | `short` — `DynamicMaps.c:32-33` for six, `Grease.c:27` for `numGrease`, `DynamicMaps.c:31` for `numChimes` | the `Add*` | `ZeroFlamesAndTheLike`: `numFlames:789`, `numTikiFlames:790`, `numCoals:791`, `numPendulums:792`, `numGrease:793`, `numStars:794`, `numShredded:795`, `numChimes:796` | the animation and render passes |
| `numSavedMaps` | `short` `DynamicMaps.c:32` | `BackUpToSavedMap:91` | `NilSavedMaps` (`DynamicMaps.c:61`) | the full check at `:75` |
| `numTempManholes` | `short` `Objects.c:77` | `Objects.c:355` | `RoomGraphics.c:56` | floor render |
| `numBands` | `short` `RubberBands.c:26` | `AddBand` | `KillAllBands` (`RubberBands.c:317`) **and** `Play.c:113` | `Render.c:539`, `RubberBands.c:213` |
| `hasMirror` | `Boolean` `Render.c:42` | `Render.c:760` (`= true`) | `ZeroMirrorRegion` (`Render.c:770`) | `Env.HasMirror`, every dirty-rect add |
| `takingTheStairs` | `Boolean` `Transit.c:18` | `Player.c:344`, `Player.c:472` (`= true`) | `RoomGraphics.c:127` | the stairs branch of the transit code |
| `numLights` | `short` `RoomGraphics.c:31` | recomputed per room inside `DrawLocale` | — | gates most object draws |
| `thisTiles[kNumTiles]`, `thisBackground` | `short` `RoomGraphics.c:31-32` | `DrawRoomBackground` | — | `Env.Tile`, `Env.Background` |
| `localNumbers[9]`, `isStructure[9]`, `localRoomsDest[9]` | `RoomGraphics.c:29-33` | the locale loop | — | neighbour composition |
| `leftThresh`, `rightThresh`, `topOpen`, `bottomOpen`, `leftOpen`, `rightOpen` | `short`/`Boolean` `Room.c:28-30` | `DetermineRoomOpenings` | — | the escape checks; `leftThresh`/`rightThresh` are *both* a coordinate and a wall flag (`Interactions.c:513`, `:603`) |
| `shadowVisible` | `Boolean` `Player.c:53` | four sites: `RoomGraphics.c:126`, `RoomGraphics.c:459`, `Modes.c:361`, `Player.c:1286` (`= false`) | `RoomGraphics.c:126` | the shadow draw |
| `tvInRoom`, `tvWithMovieNumber` | `Boolean` `HouseIO.c:36`, `short` `Objects.c:77` | `ObjectDrawAll.c:708` | `RoomGraphics.c:59-60` and again at `ReadyLevel:409-410` | `Play.c:202`/`:224`/`:466`/`:491`, `Dynamics.c:437`/`:449`, `Transit.c:304`/`:342`/`:381`/`:420`, `Trip.c:43` |
| `nextFrame` | `long` `Render.c:40` | `InitGarbageRects` at `Render.c:690`: `nextFrame = TickCount() + kTicksPerFrame` | same | the frame pacer |
| `numWork2Main`, `numBack2Work` | `short` `Render.c:41` | the two adders | `InitGarbageRects` | `CopyRectsQD` |

`shadowVisible` is the trap in this table and it is the reason `env.go` needs a doc fix rather than
a signature change. There are two different things called "is the shadow visible":
`IsShadowVisible()` at `Room.c:1103` is a *pure predicate* recomputed from the room, and
`shadowVisible` at `Player.c:53` is a *cached flag* that `Modes.c:361` and `Player.c:1286` write
for reasons that have nothing to do with the room. `NopEnv` conflates them
(`internal/game/player/env.go:194-195` reads and writes one field). See §2.13.

`nextFrame` is worth one line of warning: `InitGarbageRects` is the *frame pacer* reset as well as
the dirty-rect reset, so entering a room resets the tick budget. `RestoreEntireGameScreen` does not
call it (`Play.c:814` has no `InitGarbageRects`), so returning from the banner does not.

### 2.5 House-scoped

Cleared when a house is adopted, not when a room changes and not when a game starts.

| Global | C type | Set by | Cleared by | Read by |
|---|---|---|---|---|
| `wardBitSet` | `Boolean` `RoomGraphics.c:33` | `HouseIO.c:416` from `flags & 0x1` | `House.c:142` | `RoomGraphics.c:195` — makes off-the-map rooms black instead of dirt/meadow/sky |
| `phoneBitSet` | `Boolean` `Play.c:54` | `HouseIO.c:417` from `flags & 0x2` | `House.c:143` | `Play.c:748` |
| `previousRoom` | `short` `Room.c:27` | `Room.c:212`, `:217`, `:384` | `House.c:151`, `HouseIO.c:424` (`= -1`), also `InterfaceInit.c:156` at launch | `House.c:684` |
| `hasMovie` | `Boolean` `HouseIO.c:36` | `HouseIO.c:139` | `HouseIO.c:158`, `:196` | the four `(hasQT && hasMovie && tvInRoom)` guards |
| `numberRooms`, `thisRoomNumber` | `short` `Room.c:27` | house load / `ForceThisRoom` | — | everything |
| object states inside `thisHouse` | the house handle itself | `SetObjectsToDefaults` (`Play.c:603-708`), then `SetObjectState` (`Objects.c:366`) | `SetObjectsToDefaults`, skipped on resume (`Play.c:102`) | `GetObjectState` (`Objects.c:703`), `IsThisValid` (`Objects.c:89-122`) |

The last row is the one that breaks any attempt to make the house immutable in Stage 1.5. A switch
thrown in room 4 has to be visible when the glider walks back into room 4 ten rooms later, and the
original stores that in the house handle. `SetObjectState` (`Objects.c:366-…`) writes **three
places in one call** — the house handle, `masterObjects[...].theObject`, and
`hotSpots[...].isOn` — at `:408`/`:416`, `:465`/`:470`, `:523`/`:529`, `:620`/`:625`. That single
function is the strongest argument in this whole section for one package rather than two (§2.12).

`IsThisValid` (`Objects.c:89-122`) is the read side and its exact shape matters:

```
itsGood = true
switch (<object what>)
{
    case kObjectIsEmpty:
        itsGood = false
        break
    case kRedClock: ... case kHelium:      // the twelve grouped kinds, kSparkle among them
        itsGood = data.c.state
        break
}
// no default
return itsGood
```

There is no `default`, so every kind outside the twelve is unconditionally valid. A porter who
adds a `default: itsGood = false` to satisfy a linter deletes most of the game.

### 2.6 Game-scoped

| Global | C type | Set by | Reset at new game by | Read by |
|---|---|---|---|---|
| `theScore` | `long` `Player.c:50` | `Interactions.c` reward handlers | `InitGlider`: `Play.c:317` from `smallGame.score` on resume, `Play.c:339` `= 0` otherwise | the scoreboard |
| `mortals` | `short` `Play.c:52` | `OffAMortal` | `Play.c:318` on resume; `Play.c:341-343` otherwise: `mortals = kInitialGliders; if (twoPlayerGame) mortals += kInitialGliders` | `OffAMortal`'s game-over test, the scoreboard |
| `batteryTotal` | `short` `Play.c:52` | the pickups, `Input.c:138`/`:163` | `Play.c:319` / `Play.c:344` (`= 0`) | `Env.BatteryTotal`, the thrust/lift branch |
| `bandsTotal` | `short` `Play.c:52` | pickups, `AddBand` | `Play.c:321` / `Play.c:345` | `Env.BandsTotal` |
| `foilTotal` | `short` `Play.c:52` | pickups, `Interactions.c:82`, `Dynamics.c:51`/`:62-63` | `Play.c:322` / `Play.c:346` | `Env.FoilTotal` |
| `showFoil` | `Boolean` `Play.c:53` | `Player.c:1144-1152` | `Play.c:325` / `Play.c:350` | the sprite-sheet swap |
| `numStarsRemaining` | `short` | `Play.c:312` `= smallGame.wasStarsLeft` on resume, `:314` `= CountStarsInHouse()` on new | as set | the banner |
| `gameFrame` | `long` `Play.c:51` | the frame loop | `Play.c:112` | the demo recorder, various timers |
| `saidFollow` | `short` `Modes.c:13` | `Modes.c:465` `saidFollow++`, guarded by `saidFollow < 3` at `:462` | `Play.c:115` | `Modes.c:462` |
| `otherPlayerEscaped` | `short` `Interactions.c:42` | twelve sites in `Player.c` (`:358`, `:363`, `:486`, `:491`, `:622`, `:627`, `:729`, `:734`, `:826`, `:831`, `:1025`, `:1030`) | `Play.c:116` (`= kNoOneEscaped`, `-1`, `GliderDefines.h:608`) | the same twelve blocks |
| `onePlayerLeft` | `Boolean` `Player.c:53` | `Player.c:1507` (`= true`) | `Play.c:117` | `Player.c:347`, `:475`, `:611`, `:718`, `:815`, `:1014`, `:1105`, `:1445`, and the `!onePlayerLeft` conjunct in the four two-player escape dispatches |
| `playerSuicide` | `Boolean` `Play.c:54` | the suicide key | `Play.c:118` | `OffAMortal` |
| `twoPlayerGame` | `Boolean` `Play.c:53` | the mode dialog | — (a game setting, fixed for the game) | everywhere |
| `evenFrame` | `Boolean` `Play.c:53` | flipped each frame | — | `Interactions.c:1756` (`WebGlider`) and the alternating renderers |

`InitGlider` is the shape problem here. It takes a `gliderPtr` (`Play.c:307`) but nine of the
statements in it write **game-scoped** globals: `numStarsRemaining` (`:312`, `:314`), `theScore`,
`mortals`, `batteryTotal`, `bandsTotal`, `foilTotal`, `showFoil` (`:317-325` / `:339-350`). In a
two-player game it is called twice (`Play.c:121-122`), so all nine are written twice. That happens
to be idempotent — `mortals = kInitialGliders; if (twoPlayerGame) mortals += kInitialGliders`
gives `2*kInitialGliders` on both calls because the first statement is an assignment, not a `+=` —
but it is idempotent by luck. **In Go `InitGlider` must be a method on the game-scoped type taking
a `*player.Glider`, never a method on `Glider`**, or the second call will double `mortals`.

### 2.7 New-life scope: `OffAMortal`

`OffAMortal` (`Player.c:1484-1604`) is the only new-life reset site, and it is the function three
of the eight reports transcribe with fused conditions. The exact control flow of its two-player
head:

```
void OffAMortal (gliderPtr thisGlider)
{
    ...
    HideGlider(thisGlider)                          // :1495
    if (twoPlayerGame)                              // :1496
    {
        if (<the other glider is also gone>)        // inner test
        {
            ...                                     // both dead -> game over path
        }
        if (<...>)
        {
            onePlayerLeft = true                    // :1507
            playerDead = thisGlider->which          // :1508
        }
    }
    else                                            // :1511
    {
        ...                                         // :1511-1515, one-player path
    }
```

The `else` at `:1511` belongs to `if (twoPlayerGame)` at `:1496`, and the inner tests have no
`else` of their own. Any transcription that renders this as
`if (twoPlayerGame && bothGone) ... else if (twoPlayerGame) ... else ...` changes which arms run.
`onePlayerLeft` is `:1507` and `playerDead` is `:1508` — several reports say `:1509`/`:1510`.

`playerDead` is a `Boolean` (`Player.c:53`) holding a *player identity*, because
`kPlayer1 == TRUE` and `kPlayer2 == FALSE` (`GliderDefines.h:556-557`). `Player.c:1508` assigns
`thisGlider->which` into it and `Player.c:349` compares it against `kPlayer1`. So the type is
faithful and the *name* is the problem: `playerDead` reads as "is a player dead", but it means
"which player is dead", and it is only meaningful when `onePlayerLeft` is set. `Player.c:1445`
shows the intended use exactly: `if ((twoPlayerGame) && (onePlayerLeft) && (thisGlider->which == playerDead))`.

### 2.8 Never reset

The trap category. Each of these is BSS-zero at launch and then holds whatever wrote it last,
across rooms, lives, games and houses.

| Global | Decl | Written | Read | Why it is contained (or is not) |
|---|---|---|---|---|
| `firstPlayer` | `Boolean` `Transit.c:18` | `Modes.c:467` only | `Transit.c:292` only | one writer, one reader, and the writer always runs immediately before the reader in the limbo handshake. Nothing else can observe it — which is why `Env.SetFirstPlayer` is redundant (§2.13). |
| `bandHitLast` | `short` `RubberBands.c:26` | `:88`, `:146` | `:86` | read before either write in the same sweep; the stale value from the previous room is compared once. Genuinely leaky, and the leak is observable in principle: the first band fired in a new room can be suppressed by what the last band in the previous room hit. UNVERIFIED: I did not construct the input that demonstrates it. |
| `tvOn` | `Boolean` `Play.c:54` | `ObjectDrawAll.c:693`, `Trip.c:49`, `Trip.c:54` | `Play.c:205`, `:466`, `:491`; `Transit.c:304`, `:342`, `:381`, `:420` | **not contained.** Nothing clears it, so walking out of a room with the TV on into a room with no TV leaves `tvOn` true. Every reader also tests `tvInRoom`, which *is* room-scoped (`RoomGraphics.c:59`), so the stale value is masked — but a port that drops the `tvInRoom` conjunct anywhere gets a phantom TV. |
| `playerDead` | `Boolean` `Player.c:53` | `Player.c:1508` | eleven sites | gated by `onePlayerLeft`, which *is* reset (`Play.c:117`). |
| `nLocalObj` | `short` `Objects.c:76` | never | never | fully dead. `Objects.c:76` is its only occurrence in the tree. Do not port it. |
| `lastBackground` | `short` `Room.c:28` | `InterfaceInit.c:159` at launch, `RoomInfo.c:481` in the editor | the editor | editor state, not gameplay. Out of scope for 1.5. |
| `newState` | `Boolean` `Objects.c:78` | `SetObjectState`, on some paths only | the `SetObjectState` callers | an out-parameter smuggled through a global; it *survives* calls that do not write it. In Go this must be a second return value, with the not-written case given a documented value. |
| `priority0/1/2` | `short` `Sound.c:31` | `PlaySound0/1/2` (`:144`, `:172`, `:210`), zeroed by the async completion callbacks `CallBack0/1/2` (`Sound.c:225`, `:241`, `:257`) and by the sound init at `Sound.c:451-453` | `PlayPrioritySound` (`:47-68`) | intentionally persistent — this *is* the throttle. `FlushAnyTriggerPlaying` (`Sound.c:87-127`) quiets whichever channel holds `kTriggerPriority`, and `DrawLocale:57` calls it. |

### 2.9 One package: `internal/game`

The reports disagree four ways, and one of them (`background-animations.md:441`) argues for no new
package at all:

- `internal/game/world` with `World` implementing `player.Env` — `hotspot-pipeline.md:766-768`,
  `bands-grease.md:432`, `live-state-inventory.md`'s `goSurface`, and `dynamics-movers.md:625`
  (as the thing that satisfies a *second* interface).
- `internal/game/room` with `Room` implementing `player.Env` — `room-transitions.md:860-864`.
- `internal/game` with `*Game` implementing `player.Env` — `rewards-switches-triggers.md:519`.
- `internal/game/dynamics` as its own package behind its own `dynamics.Env` —
  `dynamics-core.md:709`, `dynamics-movers.md:625-628`.
- extend `internal/render` for the animation tables, adding files rather than a package —
  `background-animations.md:441`.

**Recommendation: one new package, `internal/game`, holding two types, `World` and `Room`. Reject
`internal/game/world`, `internal/game/room` and `internal/game/dynamics` as separate packages.
Accept `background-animations.md`'s point in full: the flame/star/coal/pendulum/grease *tables*
stay in `internal/render` where `Scene` already declares them.**

The reasons are all reasons from the C, not taste.

**1. `SetObjectState` writes three owners in one call.** `Objects.c:408`/`:416`, `:465`/`:470`,
`:523`/`:529`, `:620`/`:625` write the house handle, `masterObjects[...]` and `hotSpots[...].isOn`
together. If `hotSpots` lives in package A and the house-object writer in package B, that one
function needs either a back-pointer or a second interface. Both are the tangle
`internal/game/player/env.go:3-15` was written to avoid, and `env.go` avoids it *once*, at the
player boundary, deliberately. A second seam is not a second win; it is the first mistake.

**2. `HandleGrease` mutates `hotSpots` from a frame-level driver.** `Grease.c:60`, `:61`, `:68`,
`:95`, `:102` assign into `hotSpots[grease[i].hotNum]` — action, isOn and bounds. Grease is a
room-scoped table and the hot-spot table is a room-scoped table, but the *driver* is the per-frame
loop, which also needs the house, `numLights` and the RNG. An `internal/game/room` package would
have to expose write access to `Room.Hot` to whatever holds the frame loop, or hold the frame loop
itself and then need the house — which is `room-transitions.md`'s own position collapsing into
`internal/game`.

**3. `DrawLocale` is a World-level operation that *constructs* a Room.** It reads
`thisHouse`, `numNeighbors`, `wardBitSet` and the RNG stream, and it produces `hotSpots`,
`masterObjects`, `dinahs`, the flame tables and the openings. A `room` package cannot own its own
constructor; the thing that owns the house has to. So `Room` is a value the World builds, not a
peer of the World.

**4. A separate `dynamics` package buys a second `Env` and nothing else.** I checked whether the
dynamics code reaches sideways into the room tables the way grease does:
`grep -n 'hotSpots\[|masterObjects\[|SetObjectState|thisHouse' Dynamics.c Dynamics2.c Dynamics3.c`
finds nothing. So `dynamics-core.md:709` is right that the coupling is looser there than
elsewhere. But its handlers still spend foil (`Dynamics.c:51`, `:62-63`), play sounds, and call
`StartGliderFoilLosing`/`StartGliderFadingOut` — so it needs `player.Env`'s inventory and sound
methods reflected into a `dynamics.Env`, plus `TwoPlayerGame`/`OnePlayerLeft`/`Survivor` for the
four-way block. That is a near-copy of a dozen `Env` methods to buy a directory. Put
`Dynamics*.c`, `Trip.c` and `Grease.c` in files inside `internal/game` instead — same file
granularity, no second interface, no second test double.

**5. `internal/game` is currently empty of everything but `player/`.** `rewards-switches-triggers.md:519`
notes this correctly. Naming the package `world` gives `world.World`, which stutters, and leaves
`internal/game` a directory with no code. Naming the type `Game` and the package `game` gives
`game.Game`, which stutters too. `internal/game` with `game.World` and `game.Room` reads correctly
at every use site and matches how the C actually splits: the house-and-session half and the
one-room half.

`internal/render` keeps what it has. `render.Scene` (`internal/render/locale.go`) already declares
`SavedMaps`, `Flames`, `TikiFlames`, `Coals`, `Pendulums`, `Stars`, `Dynamics`, `TempManholes`,
`MirrorRects` and `numGrease`, already reproduces `DrawLocale`'s clear-everything prologue in the
right order, and already has the `-1`-when-full contract. Duplicating those in `internal/game`
would give two tables where the C has one.

### 2.10 The two types

```go
// internal/game

// World is one game in progress: the house, the score, the inventory, both
// gliders, the sound mixer, and the room the gliders are in. It is the C's
// Play.c / Player.c / Interactions.c file-scope globals, minus the ones that
// DrawLocale rebuilds -- those are on Room.
//
// A World must not be copied after Start: &w.P1 and &w.P2 are handed to the
// player code as *player.Glider and to Env methods that compare identities.
type World struct {
    H *house.House          // thisHouse
    R Room                  // the current locale; rebuilt, not replaced
    M Mixer                 // priority0/1/2 and the trigger flush
    Rand *rand.Rand         // the RandomInt stream; order is observable

    P1, P2 player.Glider    // theGlider, theGlider2 -- values, so &w.P1 is stable
    TwoPlayer bool          // twoPlayerGame
    OneLeft   bool          // onePlayerLeft
    DeadWhich bool          // playerDead: kPlayer1 == true
    FirstPlayer bool        // firstPlayer (Transit.c:18)
    Escaped   int16         // otherPlayerEscaped
    Suicide   bool          // playerSuicide

    Score    int32          // theScore (long)
    Mortals  int16          // mortals
    Battery  int16          // batteryTotal -- signed; negative is helium
    Bands    int16          // bandsTotal
    Foil     int16          // foilTotal
    ShowFoil bool           // showFoil
    StarsLeft int16         // numStarsRemaining
    SaidFollow int16        // saidFollow

    Frame     int64         // gameFrame
    EvenFrame bool          // evenFrame
    NextFrame int64         // nextFrame (Render.c:40)

    Pending player.Link     // transRect + transRoom + linkedToWhat, as one value
    PrevRoom int16          // previousRoom
    Ward, Phone bool        // wardBitSet, phoneBitSet -- house scope
    HasMovie bool           // hasMovie
    TVOn bool               // tvOn: never reset in the C; see below
}

// Room is what DrawLocale produces: everything with a room-change lifetime.
type Room struct {
    *render.Scene           // the composition, the surfaces, and the anim tables

    Hot    []HotObject      // hotSpots, cap MaxHotSpots
    Master []MasterObject   // masterObjects, cap MaxMasterObjects
    Trig   [MaxTriggers]Trigger
    Bands  [MaxRubberBands]Band
    Grease []Grease         // cap MaxGrease

    LeftThresh, RightThresh int16
    TopOpen, BottomOpen, LeftOpen, RightOpen bool
    ShadowVisible   bool    // the cached flag, not the predicate
    TakingTheStairs bool
    HasMirror       bool
    TVInRoom        bool
    TVMovieNumber   int16

    Work2Main [MaxGarbageRects]Rect
    Back2Work [MaxGarbageRects]Rect
    NumWork2Main, NumBack2Work int
}
```

Three shape decisions follow directly from §2.1 and §2.4.

**`Room` is a field, not a pointer, and it is rebuilt in place.** `World.ReadyLevel()` reproduces
`RoomGraphics.c:402-418`; `World.DrawLocale()` reproduces `RoomGraphics.c:43` onward and is
callable on its own, because `RestoreEntireGameScreen` calls it on its own (`Play.c:814`, with no
`NilSavedMaps` before and no `InitGarbageRects` after). So the rebuild cannot be
`w.R = NewRoom(...)`: it must be a method whose doc comment enumerates the fields that survive it.
Those fields are exactly the ones §2.4 does not list as reset by `RoomGraphics.c:51-60` — in
particular `NextFrame`, `NumWork2Main` and `NumBack2Work`, which `InitGarbageRects` owns and
`DrawLocale` does not touch.

**Capacity is expressed as a cap, not a length.** `Hot`, `Master` and `Grease` are slices with
`cap == the bound` and the append helpers return `-1` when full, matching `AddGrease`
(`Grease.c:206`) and `AddHotSpot`. `Trig` and `Bands` are arrays because the C clears one field of
every slot rather than truncating a count (`Triggers.c:203`, `RubberBands.c:317`) and there is no
`numTriggers` at all.

**`Room` embeds `*render.Scene` rather than duplicating it**, so `w.R.Flames`, `w.R.SavedMaps`,
`w.R.Back` and `w.R.NumLights` are reachable without a second copy and `internal/render`'s golden
tests keep working untouched.

`TVOn` deserves the comment it will not otherwise get:

```go
// TVOn is tvOn (Play.c:54). It is never reset anywhere in the original -- written
// at ObjectDrawAll.c:693 and Trip.c:49/:54, read at Play.c:205/:466/:491 and
// Transit.c:304/:342/:381/:420 -- so it survives a room change with a stale value
// from the previous room's television. Every reader also tests TVInRoom, which IS
// room-scoped (RoomGraphics.c:59), which is the only reason nothing shows. Keep
// both conjuncts at every one of the seven read sites.
```

### 2.11 How the two gliders share what they share

The question is: `batteryTotal`, `foilTotal`, `bandsTotal`, the sound throttle and
`otherPlayerEscaped` are single globals read and written by handlers that run once per glider. How
does that work in Go with no package-level state and no `*World` back-pointer inside `Glider`?

It already works, and the answer is that Stage 1.4 built the mechanism:

**1. The gliders are values inside the `World`, and `*World` is the `Env`.** `player.Glider` holds
no back-pointer (`env.go:3-15`); `Env` is threaded as the *first parameter* of every handler. So a
frame is:

```go
func (w *World) doFrame() {
    ...
    w.P1.MoveGlider(w)      // w is the Env; &w.P1 is the glider
    if w.TwoPlayer {
        w.P2.MoveGlider(w)
    }
}
```

`w.P1.SomeHandler(w)` gives the handler a receiver that is `&w.P1` and an `Env` that is `w`. The
shared counters are ordinary fields of `w`, so `w.SetBatteryTotal(n)` from inside a P1 handler and
the same call from inside a P2 handler hit the same `int16`. This is not a trick; it is the exact
aliasing the C has, expressed with one object instead of a global segment. And because `*World`
implements `Env` directly there is no adapter type, no closure struct, and no allocation on the
interface conversion.

**2. The signedness of `batteryTotal` is a property of the shared field, not of the accessor.**
`env.go:23-27` already documents it: one counter, positive is battery, negative is helium, and
`DoBatteryEngaged` decrementing while `DoHeliumEngaged` increments (`Input.c:138`, `:163`) are both
walking the same number toward zero. Nothing about two players changes that. What two players do
change is that P1 can spend P2's helium, and the port must not "fix" it — the C has one counter and
so must the port, with a `World.Battery` comment saying so.

**3. `otherPlayerEscaped` is a rendezvous slot, so it must not become a per-glider field.** All
twelve sites are the same two-line pattern, e.g. `Player.c:356-363`:

```
if (otherPlayerEscaped == kPlayerEscapedUpStairs)   // :356
{
    otherPlayerEscaped = kNoOneEscaped              // :358
    ... // both are out: actually change room
}
else
{
    otherPlayerEscaped = kPlayerEscapedUpStairs     // :363
    ... // I am first: go to limbo and wait
}
```

The whole meaning is "did the *other* one get here first", so the field has to be one field with
`World` as its owner and `kNoOneEscaped` (`-1`, `GliderDefines.h:608`) as the empty value.
Per-glider storage makes the first branch unreachable.

**4. The sound throttle is a `Mixer` value on the `World`, not a package singleton.** `priority0`,
`priority1`, `priority2` (`Sound.c:31`) plus the three "what is playing" slots are the throttle;
`PlayPrioritySound` (`Sound.c:40-84`) picks the lowest-priority channel and only speaks if
`priority >= lowestPriority` (`:68`). Two things force it onto the `World`:
`FlushAnyTriggerPlaying` is called from `DrawLocale:57`, so the throttle has room-change-scoped
behaviour; and the trigger-priority early-out at `Sound.c:47-51` means the *result* of a sound call
depends on what the other glider just played. So:

```go
type Mixer struct {
    Priority [3]int16   // priority0, priority1, priority2 (Sound.c:31)
    Playing  [3]int16   // soundPlaying0/1/2
}
func (m *Mixer) PlayPriority(sound, priority int16)  // Sound.c:40-84
func (m *Mixer) FlushAnyTriggerPlaying()             // Sound.c:87-127
```

and `func (w *World) PlayPrioritySound(s, p int16) { w.M.PlayPriority(s, p) }` satisfies
`env.go:20` with no signature change. The `CallBack0/1/2` zeroing at `Sound.c:225`/`:241`/`:257` is
asynchronous in the original; in the port it is the audio backend telling the mixer a slot freed,
and the *order* in which slots free is observable through `PlayPrioritySound`'s
lowest-priority-wins choice. UNVERIFIED: whether a deterministic port can reproduce that ordering
exactly, since the original's timing came from the Sound Manager's own callbacks.

**5. `Survivor()` needs both gliders, which is why it belongs on the `World`.**
`Player.c:349-352` moves the glider that `playerDead` does *not* name. With `P1`/`P2` as `World`
fields that is three lines and no pointer bookkeeping:

```go
func (w *World) Survivor() *player.Glider {
    if w.DeadWhich == player.Player1 {   // kPlayer1 == TRUE (GliderDefines.h:556)
        return &w.P2
    }
    return &w.P1
}
```

**6. The pending-transit triple is `World`-scoped, and `g.Transit` is a shadow copy.** `transRect`
(`Player.c:49`), `transRoom` (`Player.c:52`) and `linkedToWhat` (`Transit.c:17`) are *single*
globals that both gliders' transit handlers read. Stage 1.4 moved the resolution to the caller and
put the answer on the glider as `Transit Link` (`internal/game/player/glider.go:93`,
`internal/game/player/rect.go:36-40`). That is correct for `StartGliderMailingOut`, which reads the
departing glider's own link (`internal/game/player/modes.go:162`). It is *not* the source of truth:
`World.Pending player.Link` is, because the C's globals are shared. `room-transitions.md:702`
reaches the same conclusion and it is right. No `Env` method is needed to publish it — the
publisher is `HandleHotSpotCollision`, which lives in `internal/game` and can assign `w.Pending`
directly before calling `g.StartGliderTransporting(w, link)`.

The one cost of all this is that `World` is not copyable, and Go will not enforce it. The doc
comment has to say so, and a `go vet`-visible guard (an embedded `noCopy sync.Locker` or just a
`_ [0]sync.Mutex`) is worth the ugliness given that `&w.P1` escapes into `Env` calls.

### 2.12 Verdict on `internal/game/player/env.go`

**It needs extending. It does not need replacing, and no existing method needs a changed
signature.** Every one of the 46 methods currently declared maps onto something in §2.10's `World`
or `Room`, and `*World` can implement all 46 as one- or two-line bodies. The seam itself is sound:
the anti-tangle argument at `env.go:3-15` is exactly the argument §2.9 reaches from the C, and
`NopEnv`'s embed-and-override design (`env.go:141-147`) is what makes an additive extension not
break Stage 1.4's tests.

One correction to the record first: `live-state-inventory.md` says the interface's ~30 methods each
correspond to a C free function. There are **46**
(`sed -n '16,139p' internal/game/player/env.go | grep -cE '^\t[A-Z][A-Za-z]*\('`), and about
twenty of them — `BatteryTotal`, `SetBatteryTotal`, `BandsTotal`, `SetBandsTotal`, `FoilTotal`,
`SetFoilTotal`, `TopOpen`, `BottomOpen`, `LeftThresh`, `RightThresh`, `Background`, `Tile`,
`TwoPlayerGame`, `OnePlayerLeft`, `PlayerDead`, `OtherPlayerEscaped`, `SetOtherPlayerEscaped`,
`SaidFollow`, `SetSaidFollow`, `SetFirstPlayer` — are accessors over a global with no C function
behind them at all. That distinction matters for this verdict: an accessor can be renamed freely,
whereas a method named after a C function should keep the name.

#### Doc-comment corrections (no signature change)

**`FlagStillOvers` — `env.go:103-105` states the effect backwards.** It reads "re-tests what the
glider is standing on … so a glider dropped onto a switch triggers it." The opposite is true.
`FlagStillOvers` latches `stillOver` on every rect the glider overlaps
(`Interactions.c:1725` onward), and `HandleSwitches` (`Interactions.c:990`), `ArmTrigger`
(`Triggers.c:38`) and `HandleMicrowaveAction` (`Interactions.c:1164`) all *early-return* when it is
set. Its purpose is to stop a glider that materialises inside a switch from firing it. Replace with:

```go
// FlagStillOvers latches hotSpots[].stillOver on every rect the glider currently
// overlaps, after it arrives somewhere without moving. It SUPPRESSES: HandleSwitches
// (Interactions.c:990), ArmTrigger (Triggers.c:38) and HandleMicrowaveAction
// (Interactions.c:1164) each early-return while the flag is set, so a glider that
// materialises inside a switch does not fire it.
FlagStillOvers(g *Glider)
```

**`IsShadowVisible` — the name covers two different things and `NopEnv` conflates them.**
`Room.c:1103`'s `IsShadowVisible` is a pure predicate recomputed from the room; `shadowVisible`
(`Player.c:53`) is a cached flag written at `RoomGraphics.c:126`, `RoomGraphics.c:459`,
`Modes.c:361` and `Player.c:1286`. The player code reads the *flag*. `NopEnv:194-195` makes the
getter and setter the same field, which is right for the flag and wrong for the predicate. Keep
both methods, and say which one they are:

```go
// IsShadowVisible and SetShadowVisible are the cached flag shadowVisible
// (Player.c:53), written at RoomGraphics.c:126 (cleared on every DrawLocale),
// RoomGraphics.c:459, Modes.c:361 and Player.c:1286. They are NOT the predicate
// IsShadowVisible() at Room.c:1103, which recomputes from the room; the
// implementation calls that only to seed the flag.
```

**`SetFirstPlayer` — mark it redundant, keep it.** `firstPlayer` (`Transit.c:18`) has exactly one
writer, `Modes.c:467`, and one reader, `Transit.c:292`, and the writer is inside
`FlagGliderInLimbo`, which is `player` code. So the method is real. Add: "sole writer is
`FlagGliderInLimbo` (`Modes.c:467`); sole reader is `Transit.c:292`; never reset, which is safe
only because those two always run in that order."

**`Survivor` — add the precondition.** It is meaningful only when `OnePlayerLeft()` is true,
because `playerDead` is never reset (§2.8). Add: "Valid only when `OnePlayerLeft()`; outside that
guard `playerDead` holds a value from a previous game (it has no reset site — `Play.c:112-118`
resets `onePlayerLeft` but not `playerDead`)."

**The four transit methods — say that they may move the *other* glider.** `MoveRoomToRoom`,
`MoveDuctToDuct`, `MoveMailToMail` and `TransportRoomToRoom` take a `*Glider` but the
one-player-left branch moves `Survivor()` instead (`Player.c:347-352`), and the normal two-player
branch moves both. Add one comment above the group: "the `g` passed in is the glider that
triggered the transition; the implementation may move the other one instead
(`Player.c:349-352`) or both."

**`Tile` — say that the caller range-checks.** `env.go:83-86` already says it. Keep it, and add
that this is deliberate mirroring of the C, which does not check either.

#### Rename (one)

`PlayerDead() bool` → `DeadPlayer() bool`. It returns *which* player is dead
(`kPlayer1 == TRUE`, `GliderDefines.h:556-557`; assigned from `thisGlider->which` at
`Player.c:1508`), not whether one is. The current name has already misled two of the eight reports.
It is an accessor with no C function behind it, so renaming costs nothing but the `NopEnv` line and
`dynamics-movers.md:680`'s reference to it. Keep the `bool` type — it is faithful to
`Boolean playerDead`.

#### Additions

All additive, so `NopEnv` absorbs them and no Stage 1.4 test breaks.

```go
// ---- frame parity -----------------------------------------------------
// EvenFrame is the global evenFrame (Play.c:53). WebGlider reads it directly
// (Interactions.c:1756) to alternate the web's grab, and several renderers
// alternate on it. It has no reset site.
EvenFrame() bool
SetEvenFrame(v bool)

// ---- transit ----------------------------------------------------------
// PublishTransitLink records where the transit object the glider just entered
// leads. In the C this is three separate globals -- transRect (Player.c:49),
// transRoom (Player.c:52) and linkedToWhat (Transit.c:17) -- written by
// StartGliderTransporting and its three siblings and read by the matching
// MoveXToY. They are SHARED by both gliders, so the glider's own Transit field
// (glider.go:93) is a shadow copy and this is the source of truth.
PublishTransitLink(l Link)
TransitLink() Link

// ---- objects ----------------------------------------------------------
// AddSparkle queues a sparkle at a rect, bounded at MaxSparkles = 3
// (GliderDefines.h:251); it is silently dropped when the table is full.
AddSparkle(r Rect)
// NumShredded and RemoveShreds let OffAMortal wait for the shred animation to
// finish; the table is bounded at MaxShredded = 4 (GliderDefines.h:264) and
// cleared on every room change (DynamicMaps.c:795).
NumShredded() int16
RemoveShreds()

// ---- dirty rects ------------------------------------------------------
// AddRectToBackRects is the other half of the pair whose work side is already
// here: back2WorkRects, bounded at kMaxGarbageRects = 48 (Render.c:20), reset by
// InitGarbageRects (Render.c:675-691).
AddRectToBackRects(r Rect)
// QuickGlidersRefresh redraws the remaining-lives badges after OffAMortal.
QuickGlidersRefresh(force bool)

// ---- game over --------------------------------------------------------
// GameOver ends the game from inside OffAMortal when mortals reaches zero.
// mortals is game-scoped and seeded from kInitialGliders = 2 (Play.c:19), doubled
// for a two-player game (Play.c:341-343).
GameOver()

// ---- music ------------------------------------------------------------
// ToggleMusicWhilePlaying is the command-key music toggle; it is here because
// DoCommandKey (already declared) reaches it.
ToggleMusicWhilePlaying()
```

Every addition is one field or one table on §2.10's `World`/`Room`, so `*World` implements each in
one line, and `NopEnv` needs eleven trivial bodies plus one field (`Shredded` already exists at
`env.go:161`, so `NumShredded` can return `int16(len(e.Shredded))`).

#### What does not change

`Env`'s 46 existing signatures, the first-parameter threading, the "the glider holds no
back-pointer" rule, `NopEnv`'s embeddability, and `player.Rect` staying its own struct rather than
becoming an alias of `house.Rect`. The last is load-bearing: `render.Rect` *is*
`house.Rect` (`internal/render/rect.go:17`) while `player.Rect` is a distinct struct with the same
layout (`internal/game/player/glider.go:28`), so `internal/game` converts with `player.Rect(r)`
at the boundary at zero cost. Unifying them would drag `house` into `player` and undo the seam.

Two Stage 1.4 deviations from the C stay as they are and stay documented as deviations:
`RightClip`/`LeftClip` as per-glider fields (`internal/game/player/glider.go:87`) where the C has
shared globals `rightClip`/`leftClip` (`Player.c:52`), and `Transit` as a per-glider `Link`
(`:93`). UNVERIFIED: whether the C's sharing of `rightClip`/`leftClip` between two gliders on the
same staircase is observable. Both are only ever written immediately before the read in the same
handler, so the shared-versus-per-glider difference should not be, but constructing the two-player
staircase case to prove it is Stage 1.5 work.

---

## 3. Behaviour by object class

All 117 object types are covered below, grouped by the nine `objectType.data` union members
(`GliderStructs.h:11-105`), which is also how every switch in the C is grouped. A type with no
runtime behaviour is listed with the reason; nothing is omitted. The census of "no behaviour" types
was cross-checked three ways — no `AddActiveRect` in `CreateActiveRects`, no case in
`AddDynamicObject`, no `Add*` in `DrawARoomsObjects` — and is reproduced in §3.11 so the next audit
does not have to re-derive it.

### 3.0 How to read the tables

Six passes touch objects. Their order inside one frame, and the frame loop itself, are §1's; the
tables here name the pass only.

| tag | pass | entry point |
|---|---|---|
| P0 | compose (room-change only) | `DrawLocale` → `ListAllLocalObjects` → `CreateActiveRects`, and `DrawARoomsObjects` → the `Add*` registrars |
| P1 | dynamics | `HandleDynamics` (`Play.c:449` 2P / `:475` 1P) over `dinahs[0..numDynamics-1]` |
| P2 | interact | `HandleInteraction` → `CheckForHotSpots` → `HandleHotSpotCollision` |
| P3 | triggers | `HandleTriggers` (`Play.c:456`/`:484`) → `FireTrigger` |
| P4 | bands | `HandleBands` (`:457`/`:485`) → `CheckBandCollision` |
| P5 | render | `RenderFrame`: `HandleGrease`, the four animation strip families, sparkles, flying points, shreds |

`X` marks `SetObjectState` (`Objects.c:366-698`), which is not a pass — it is the shared state-writing
subroutine reached from P2 (`HandleRewards`, `HandleSwitches`), P3 (`FireTrigger`) and P4. It is
specified once in §3.1 rather than repeated per type.

"Live state" names the tables in §2 the type needs a slot in. `—` means the type needs no live state
at all: everything about it is either in the house handle or in the composed background bitmap.

### 3.1 Rules that apply across every group

**Hot spots exist only for the central room.** `ListOneRoomsObjects` calls `CreateActiveRects(n)` only
under `if ((where == kCentralRoom) && (IsThisValid(roomNum, n)))` (`Objects.c:283-284`); every other
room's `masterObjects[].hotNum` is set to `-1` at `:286`. So every P2 and P4 behaviour below is
central-room-only, while several P0 and P1 behaviours reach all nine rooms. The `numChimes++` at
`ObjectRects.c:1052` lives inside `CreateActiveRects`, so the chime interval counts **central-room
chimes only** — not, as one report has it, chimes across all nine listed rooms.

**`IsThisValid(where, who)` (`Objects.c:89-122`)** gates both hot-spot creation and every P0
registration:

```c
itsGood = true;
switch ((*thisHouse)->rooms[where].objects[who].what)
{
	case kObjectIsEmpty:
	itsGood = false;
	break;

	case kRedClock: case kBlueClock: case kYellowClock: case kCuckoo:
	case kPaper: case kBattery: case kBands: case kFoil:
	case kInvisBonus: case kStar: case kSparkle: case kHelium:
	itsGood = (*thisHouse)->rooms[where].objects[who].data.c.state;
	break;
}
// no default: every other kind is unconditionally valid
```

Note `kGreaseRt`/`kGreaseLf`/`kSlider` are **not** in the state-gated list even though they share
`bonusType` — a fallen grease jar is still "valid" and still composes (as a `kSlideIt` rect). A
consumed `kSparkle` stops registering, which is the only way a sparkle ever goes away.

**`AddActiveRect` (`ObjectRects.c:277-292`)** appends to `hotSpots[]` and returns the index, or `-1`
when `nHotSpots >= kMaxHotSpots` (56). `masterObjects[who].hotNum` keeps only the **last** rect an
object creates, which matters for the fans and the five flames (§3.2). The maximum any single type
creates is **three** (a tall candle: `kLiftIt` + `kBurnIt` + `kDissolveIt`), so the crude bound is
24 × 3 = 72 > 56 and overflow is reachable in principle; it degrades gracefully to `hotNum == -1`.
UNVERIFIED: whether any shipped room actually exceeds 56.

`doScrutinize` (the 5 px inset on all four sides inside `SectGlider`) is `true` for exactly five
actions — `kDissolveIt`, `kBounceIt`, `kShredIt`, `kMicrowaveIt`, `kWebIt` — and `false` at every
other `AddActiveRect` site. Hazards use the tight box; everything else uses the loose one.

**Geometry constants**, local to `ObjectRects.c:11-17` and used nowhere else:
`kFloorColumnWide 4`, `kCeilingColumnWide 24`, `kFanColumnThick 16`, `kFanColumnDown 20`,
`kDeadlyFlameHeight 24`, `kStoolThick 25`, `kShredderActiveHigh 40`. A floor vent's lift column is
**four pixels wide**, not the width of the vent.

#### 3.1.1 `SetObjectState(room, object, action, local)` — the state engine

Nine families, keyed on the house handle's `what`. `action` is `kToggle` 0, `kForceOn` 1,
`kForceOff` 2. Returns `changed`, a local declared at `:369` and **never initialised**.

| family | union member written | honours `action` | writes on `changed && local != -1` |
|---|---|---|---|
| the 11 blowers (`:376-418`) | `data.a.state` | yes, all three | master `data.a.state`; `thisRoom` copy if `room == thisRoomNumber`; **`kBlowerOn`/`kBlowerOff` sound**; `hotSpots[hotNum].isOn = newState` guarded by `hotNum != -1` |
| the 5 flames (`:420-425`) | none | — | `changed = false; // Cannot switch on/off these` |
| the 15 furniture (`:427-443`) | none | — | `changed = false` |
| the 14 prizes (`:445-473`) | `data.c.state` | **no** — always forces `false`, `changed = (was true)` | master `data.a.state = false` (**wrong union member**, see below); inside `if (room == thisRoomNumber)`: `thisRoom` copy and `hotSpots[hotNum].isOn = false` guarded by `hotNum != -1` |
| `kSlider` (`:475-476`) | none | — | bare `break` — returns uninitialised `changed` |
| the 15 non-deluxe transports (`:478-494`) | none | — | `changed = false` |
| `kDeluxeTrans` (`:496-531`) | low nibble of `data.d.wide` | yes | master and `thisRoom` copies of the whole `wide` byte; `hotSpots[hotNum].isOn` guarded by `hotNum != -1` |
| the 8 switch/trigger kinds (`:533-540`) | none | — | `changed = false`. **`kKnifeSwitch` is not in this list** — see below |
| the 8 lights (`:542-577`) | `data.f.state` | yes | master and `thisRoom` copies. No `isOn`, no sound |
| `kGuitar` (`:579-580`) | none | — | `changed = false` |
| `kStereo` (`:582-586`) | **no object state at all** | ignores `action` | `newState = !isPlayMusicGame; isPlayMusicGame = newState; changed = true;` — a stereo switch toggles the game's global music flag |
| the 8 timer appliances (`:588-625`) | `data.g.state` | yes | master and `thisRoom` copies; and **only for `kShredder`** `hotSpots[masterObjects[local].hotNum].isOn = newState` with **no `hotNum != -1` guard** |
| `kCinderBlock`, `kFlowerBox`, `kCDs`, `kCustomPict` (`:627-631`) | none | — | `changed = false` |
| the 8 mobile enemies (`:633-671`) | `data.h.state` | yes | master and `thisRoom` copies |
| `kCobweb` (`:673-675`) | none | — | `changed = false` |
| the 15 clutter (`:677-693`) | none | — | `changed = false` |

Four things a porter must reproduce or consciously diverge from:

1. **`kKnifeSwitch` (0x45) reaches no case at all.** The switch family at `:533-540` lists
   `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kInvisSwitch`, `kTrigger`,
   `kLgTrigger`, `kSoundTrigger` — eight labels, no `kKnifeSwitch`. It is the only code in
   `0x01..0x8F` that no label matches (mechanically diffed: all 116 non-action `case k*` labels in
   `Objects.c:372-696` against the 117 `#define`s at `GliderDefines.h:311-435`). There is no
   `default:` in the outer switch and no `default:` in any of the five inner `switch (action)`
   blocks (`:386`, `:497`, `:552`, `:598`, `:645`), so a switch whose **target** is a knife switch
   returns uninitialised stack. Go must pick a value; return `false` and record the deviation.
2. **`kSlider` and every `changed = false` family return early-out `false`** — except `kSlider`,
   which returns garbage the same way (`:475-476`). Three uninitialised-return paths total.
3. **The prize family writes the wrong union member into the master copy**: `:465` is
   `masterObjects[local].theObject.data.a.state = false;`, and `blowerType.state` is at payload
   offset 7 while `bonusType.state` is at offset 8. Offset 7 is the **low byte of
   `bonusType.points`**. So collecting any prize zeroes the master copy's `points` low byte and
   leaves its `state` byte set. Harmless in the C because nothing reads either afterwards
   (`kInvisBonus` reads `points` at `Interactions.c:920`, *before* the call), and the house handle
   and `hotSpots[].isOn` are what actually make the prize go away.
4. **No bounds test on `room` or `object`.** `:373` indexes `(*thisHouse)->rooms[room].objects[object]`
   directly. A `roomLink` of `-1` cannot reach here (`HandleSwitches` guards), but nothing checks the
   upper bound.

`SetObjectsToDefaults` (`Play.c:602-704`) is the **only** reset of stored object state, called from
`NewGame` under `if (mode != kResumeGameMode)` (`Play.c:102-103`). It is a hand-written case list,
not a per-union sweep, and the omissions are byte-observable: the blower case lists 11 types and
**omits the 5 flames**; the prize case lists 14 and **omits `kSlider`**; the appliance case lists 9
(`kShredder`, `kToaster`, `kMacPlus`, `kGuitar`, `kTV`, `kCoffee`, `kOutlet`, `kVCR`, `kMicrowave`)
with `kStereo` split out to `data.g.state = isPlayMusicGame` and **omits `kCinderBlock`,
`kFlowerBox`, `kCDs`, `kCustomPict`**; the enemy case lists 8 and **omits `kCobweb`**; furniture,
switches, transports other than `kDeluxeTrans`, and clutter appear not at all. `kDeluxeTrans` is
restored from the *high* nibble: `initState = (wide & 0xF0) >> 4; wide &= 0xF0; wide += initState;`
(`:655-659`). It also clears `rooms[r].visited`.

#### 3.1.2 What contributes to `numLights`

Because ~30 draw sites and the whole-room blackout at `RoomGraphics.c:179-191` key off it,
`GetNumberOfLights(where)`'s play-mode branch (`Room.c:1033-1097`) is part of this section's
contract. An outdoor background (`kGarden`, `kSkywalk`, `kMeadow`, `kField`, `kRoof`, `kSky`,
`kStratosphere`, `kStars`) counts 1; `kDirt` counts 1 only if all eight tiles are 0; every other
background counts 0. **Only if the count is still 0** does it scan the 24 object slots, adding 1 per
`kDoorInLf`, `kDoorInRt`, `kWindowInLf`, `kWindowInRt`, `kWallWindow` unconditionally, and 1 per
light type whose `data.f.state` is true. The four *exterior* door/window types
(`kDoorExLf`/`kDoorExRt`/`kWindowExLf`/`kWindowExRt`) contribute **nothing**. Edit mode
(`:976-1031`) is the same logic on `data.f.initial`.

`isLit` is not a per-object flag: it is a local recomputed as `isLit = (numLights > 0)` at the top of
every `DrawARoomsObjects` call (`ObjectDrawAll.c:38`), from the single global `numLights` that
`DrawLocale` sets to the count of whichever room it is about to draw.

---

### 3.2 Blowers — `blowerType`, 0x01-0x10, 16 types

`blowerType` is `topLeft(4) distance(2) initial(1) state(1) vector(1) tall(1)`.

| code | type | pass | trigger | effect | live state |
|---|---|---|---|---|---|
| 0x01 | kFloorVent | P0, P2, X | glider in the 4 × `distance` column above `topLeft`, `isOn = data.a.state` | `kLiftIt`: `vDesiredVel = kFloorVentLift` (-6), an **assignment** | — |
| 0x02 | kCeilingVent | P0, P2, X | glider in the 24 × `distance` column below `topLeft` | `kDropIt`: `vDesiredVel = kCeilingVentDrop` (+8) | — |
| 0x03 | kFloorBlower | P0, P2, X | as kFloorVent | `kLiftIt` | — |
| 0x04 | kCeilingBlower | P0, P2, X | as kCeilingVent | `kDropIt` | — |
| 0x05 | kSewerGrate | P0, P2, X | as kFloorVent | `kLiftIt` | — |
| 0x06 | kLeftFan | P0, P2, X | two rects: 13 × 43 blade at `topLeft + (16,12)`, always on; `distance` × 16 wind column extending **left** at `topLeft + (-distance, 20)`, `isOn = state` | blade `kDissolveIt` (lethal), column `kPushItLeft`: `hDesiredVel += -kFanStrength` (-12), an **accumulation** | — |
| 0x07 | kRightFan | P0, P2, X | mirror; column offset by `RectWide(srcRects[kRightFan])` | blade `kDissolveIt`, column `kPushItRight` (+12) | — |
| 0x08 | kTaper | P0, P2, P5 | up to 3 rects, see below; flame strip registered in all nine rooms | `kLiftIt`+`kBurnIt` or `kBurnIt`; body 7 × 48 at `+(6,11)` `kDissolveIt` | `flames[]` slot + one 16×75 `savedMaps` slot |
| 0x09 | kCandle | P0, P2, P5 | as kTaper; column offset by `topLeft.h - 2` | body 8 × 20 at `+(9,11)` | `flames[]` + `savedMaps` |
| 0x0A | kStubby | P0, P2, P5 | as kTaper; column offset by an extra -1 | body 15 × 26 at `+(1,11)` | `flames[]` + `savedMaps` |
| 0x0B | kTiki | P0, P2, P5 | as kTaper, but the P0 registration has **no `SectRect` visibility test** | body 15 × 14 at `+(6,6)` | `tikiFlames[]` + one 8×50 `savedMaps` slot |
| 0x0C | kBBQ | P0, P2, P5 | as kTaper; flame column is `QSetRect(0, -distance, 4, 8)` — bottom **+8**, i.e. it reaches 8 px *below* `topLeft.v` | body 52 × 17 at `+(6,8)` | `bbqCoals[]` + one 32×36 `savedMaps` slot |
| 0x0D | kInvisBlower | P0, P2, X | **exactly one** rect, or **none**; selected by `switch (data.a.vector & 0x0F)` | 1 → `kLiftIt`, 2 → `kPushItRight`, 4 → `kDropIt`, 8 → `kPushItLeft`; any other low nibble → no hot spot | — |
| 0x0E | kGrecoVent | P0, P2, X | as kFloorVent | `kLiftIt` | — |
| 0x0F | kSewerBlower | P0, P2, X | as kFloorVent | `kLiftIt` | — |
| 0x10 | kLiftArea | P0, P2, X | one rect `distance` × `(tall * 2)` at `topLeft`, or none; same `vector` selector | as kInvisBlower | — |

**The vector selector**, resolving `hotspot-pipeline`'s open question 5 (which guessed at a field
name `data.a.byte0`): the field is `vector`, a `Byte` at payload offset 8, and the bit meaning is
documented in the struct comment (`GliderStructs.h:16`: `| x | x | x | x | 8 | 4 | 2 | 1 |`,
"F. lf. dn. rt. up"). The switch is at `ObjectRects.c:523` (`kInvisBlower`) and `:595` (`kLiftArea`),
with cases 1/2/4/8 and **no `default`**. A low nibble of 0, 3, 5, 6, 7 or 9-15 produces no hot spot,
and for `kInvisBlower` also leaves `bounds` uninitialised — never read, because the `AddActiveRect`
call is inside the case. `kInvisBlower`'s columns are `distance + 24` long and offset by
`12 - kFloorColumnWide/2` (up/down) or `12 - kFanColumnThick/2` (left/right), with an extra `+24` to
`topLeft.v` in the *up* case only; `kLiftArea` uses `distance × tall*2` with no adjustment.
`data.a.tall` is read at exactly two sites, both `kLiftArea` (`ObjectRects.c:64`, `:591`).

**The flames' hot spots are unconditionally on and cannot be switched.** All five flame types pass
the literal `true` for `isOn`, not `data.a.state` (`:410`, `:413`, `:416`, and the four siblings);
`SetObjectState` gives them `changed = false` (`Objects.c:420-425`); and they are absent from
`SetObjectsToDefaults`' blower list. Consequence: **an "unlit" candle still burns the glider and no
switch, trigger or band can ever turn it off.** The `isLit`-gated call at
`ObjectDrawAll.c:76-83` draws only the candle *body*; the `AddCandleFlame` call is its sibling
statement inside the same `SectRect`, so the flame is registered and animated either way.

The three-rect shape, transcribed as written (`ObjectRects.c:399-422`, and identically at `:424-447`,
`:449-472`, `:474-497`, `:499-520`):

```c
QSetRect(&bounds, 0, -theObject.data.a.distance, kFloorColumnWide, 0);
QOffsetRect(&bounds, HalfRectWide(&srcRects[kTaper]) - kFloorColumnWide / 2, 0);
QOffsetRect(&bounds, theObject.data.a.topLeft.h, theObject.data.a.topLeft.v);
if ((bounds.bottom - bounds.top) > kDeadlyFlameHeight)      // 24
{
	bounds.bottom -= kDeadlyFlameHeight;
	hotSpotNumber = AddActiveRect(&bounds, kLiftIt, who, true, false);
	bounds.bottom += kDeadlyFlameHeight;
	bounds.top = bounds.bottom - kDeadlyFlameHeight + 2;
	hotSpotNumber = AddActiveRect(&bounds, kBurnIt, who, true, false);
}
else
	hotSpotNumber = AddActiveRect(&bounds, kBurnIt, who, true, false);
QSetRect(&bounds, 0, 0, 7, 48);
QOffsetRect(&bounds, theObject.data.a.topLeft.h + 6, theObject.data.a.topLeft.v + 11);
hotSpotNumber = AddActiveRect(&bounds, kDissolveIt, who, true, true);
```

A tall flame is therefore an updraught with a lethal 22 px cap; a short one is lethal all the way
down. `hotNum` ends up on the **body**, not the flame, which is why `SetObjectState`'s `isOn` write
(if it could fire) would target the solid.

**P5, the flame animation.** `AddCandleFlame(room, i, h, v)` (`DynamicMaps.c:316-344`) is called from
`ObjectDrawAll.c:83`/`:98` (kTaper, anchors `+10,+7`), `:117`/`:132` (kCandle, `+14,+7`),
`:151`/`:166` (kStubby, `+9,+7`) — two sites each, the central-room branch and the neighbour-room
branch. It bails on `numFlames >= kMaxCandles (20) || h < 16 || v < 15`; `src` is 16 × 15 anchored
**bottom-centre** at `(h-8, v-15)`; it claims a 16 × 75 `savedMaps` slot holding 5 pre-composited
cells; `mode = RandomInt(5)`. `AddTikiFlame` uses a **top-left** anchor and an 8 × 50 slot
(`kMaxTikis` 8); `AddBBQCoals` a 32 × 36 slot with 4 cells (`kMaxCoals` 8). The neighbour-room branch
for the three candles additionally suppresses registration when the flame rect intersects the
central room grown vertically by `kFloorSupportTall` (44) — `kTiki` and `kBBQ` have no such test, and
`kTiki` has no `SectRect` test at all.

Every `RandomInt` draw in P0 is **inside** `if (savedNum != -1)`, so a saturated 24-slot `savedMaps`
table skips the draw and shifts the entire downstream RNG stream (`DynamicMaps.c:333-338`,
`:417-422`, `:503-508`, `:580-594`, `:680-685`). Since `savedMaps` saturation is screen-size
dependent (the `SectRect` gate) this makes the RNG stream a function of window size.

---

### 3.3 Furniture — `furnitureType`, 0x11-0x1F, 15 types

`furnitureType` is `bounds(8) pict(2)`: an authored rect, used verbatim. No state byte at all, so
`SetObjectState` returns `false` for every one of them and no switch, trigger or band can change
anything. All 15 are P0 + P2 only, and 12 of them share one case
(`ObjectRects.c:619-632`).

| code | type | hot spot | rect | notes |
|---|---|---|---|---|
| 0x11 | kTable | `kDissolveIt`, `isOn = true`, scrutinize | `data.b.bounds` verbatim | |
| 0x12 | kShelf | `kDissolveIt` | verbatim | |
| 0x13 | kCabinet | `kDissolveIt` | verbatim | |
| 0x14 | kFilingCabinet | `kDissolveIt` | verbatim | |
| 0x15 | kWasteBasket | `kDissolveIt` | verbatim | |
| 0x16 | kMilkCrate | `kDissolveIt` | verbatim | |
| 0x17 | kCounter | `kDissolveIt` | verbatim | |
| 0x18 | kDresser | `kDissolveIt` | verbatim | |
| 0x19 | kDeckTable | `kDissolveIt` | verbatim | |
| 0x1A | kStool | `kDissolveIt` (`:654-659`) | `InsetRect(bounds,1,1)` then `bottom = top + kStoolThick` (25) | only the **seat** is solid |
| 0x1B | kTrunk | `kDissolveIt` | verbatim | |
| 0x1C | kInvisObstacle | `kDissolveIt` (label at `:629`, inside the 12-type group) | verbatim | **This is a lethal invisible wall, not an inert placeholder.** Drawn: nothing (`ObjectDrawAll.c:269-270`) |
| 0x1D | kManhole | `kIgnoreGround` (`:640-647`) | `bounds` with `left += kGliderWide+3`, `right -= kGliderWide+3`, `top = kFloorLimit-1`, `bottom = kTileHigh`; scrutinize **false** | `ignoreGround = true` lets the glider fall through the floor. P0 also calls `AddTempManholeRect(&itsRect)` (`ObjectDrawAll.c:277`) for the floor-support art — consumed only by `RoomGraphics.c:277-371`, so `tempManholes[]` is Stage 1.3's, out of 1.5 scope |
| 0x1E | kBooks | `kDissolveIt` (`:634-638`) | `bounds` with `right -= 2` | |
| 0x1F | kInvisBounce | `kBounceIt` (`:649-652`), scrutinize | verbatim | `BounceGlider(g, &bounds)` **unconditionally** — no mode gate, so it fires on a fading-out, shredding or limbo glider |

`kDissolveIt` (`Interactions.c:1218-1244`) is the generic lethal solid, and its foil arithmetic is
the one thing to get right — note the else-less inner guard:

```c
case kDissolveIt:
if (thisGlider->mode != kGliderFadingOut)                           // :1219
{
	if ((foilTotal > 0) || (thisGlider->mode == kGliderLosingFoil))  // :1221
	{
		if (GliderHitTop(thisGlider, &(who->bounds)))                // :1223
		{
			StartGliderFadingOut(thisGlider);                        // :1225
			PlayPrioritySound(kFadeOutSound, kFadeOutPriority);      // :1226
		}
		else
		{
			if (foilTotal > 0)                                       // :1230  <-- no else
			{
				foilTotal--;                                         // :1232
				if (foilTotal <= 0)                                  // :1233
					StartGliderFoilLosing(thisGlider);               // :1234
			}
		}
	}
	else
	{
		StartGliderFadingOut(thisGlider);                            // :1240
		PlayPrioritySound(kFadeOutSound, kFadeOutPriority);          // :1241
	}
}
break;
```

`GliderHitTop` true means a vertical or engulfing contact, which is **fatal even with foil**; false
means a side impact, which `GliderHitTop` has already handled itself (sound, `foilTotal--`,
`StartGliderFoilLosing`, `hVel = -hVel - offset` at `Interactions.c:81-93`) and the caller then
decrements a **second** time.

---

### 3.4 Bonuses and prizes — `bonusType`, 0x21-0x2F, 15 types

`bonusType` is `topLeft(4) length(2) points(2) state(1) initial(1)`. Eleven of the fifteen share one
`CreateActiveRects` case (`ObjectRects.c:661-679`): `bounds = srcRects[what]`, `ZeroRectCorner`,
offset by `data.c.topLeft`, one `kRewardIt` rect with `isOn = data.c.state`, scrutinize false.

| code | type | pass | trigger | effect | live state |
|---|---|---|---|---|---|
| 0x21 | kRedClock | P0, P2, P4, X | `kRewardIt` hit, no mode gate | `kBeepsSound`; `RestoreFromSavedMap(...,false)`; `AddFlyingPoint(bounds, 100, hVel/2, vVel/2)`; `hVel /= 4; vVel /= 4`; `theScore += 100`; `RedrawAllGrease()` | one `savedMaps` slot; a `flyingPoints[]` slot |
| 0x22 | kBlueClock | " | " | as above with `kBuzzerSound` / 300 | " |
| 0x23 | kYellowClock | " | " | `kDingSound` / 500 | " |
| 0x24 | kCuckoo | " | " | `kCuckooSound` / 1000; **plus `StopPendulum(thisRoomNumber, objectNum)`** | **two** `savedMaps` slots (clock face + 32×84 pendulum strip); a `pendulums[]` slot; a `flyingPoints[]` slot |
| 0x25 | kPaper | " | " | `kEnergizeSound`; `RestoreFromSavedMap`; `AddSparkle`; `hVel /= 2; vVel /= 2`; `mortals++`, **and a second `mortals++` if `(twoPlayerGame) && (!onePlayerLeft)`**; `QuickGlidersRefresh()`; `RedrawAllGrease()` | `savedMaps`; `sparkles[]` |
| 0x26 | kBattery | " | " | as kPaper but `if (batteryTotal > 0) batteryTotal += 50; else batteryTotal = 50;` then a **separate** `if 2P` `+= 50`; `QuickBatteryRefresh(false)` | `savedMaps`; `sparkles[]` |
| 0x27 | kBands | " | " | `bandsTotal += 8`, doubled in 2P; `QuickBandsRefresh(false)` | " |
| 0x28 | kGreaseRt | P0, P2, P3, P4, P5, X | rect's **action depends on state**: `kRewardIt` if `data.c.state` (jar upright), else `kSlideIt` | `if (SetObjectState(...)) SpillGrease(dynaNum, hotNum)` — **no sound, no sparkle, no damping, no `RedrawAllGrease`** | `grease[]` slot + one 32×108 `savedMaps` slot, but **only when upright** |
| 0x29 | kGreaseLf | " | mirror | " | " |
| 0x2A | kFoil | P0, P2, P4, X | `kRewardIt` | `kEnergizeSound`; `RestoreFromSavedMap`; `AddSparkle`; `/= 2`; `foilTotal += 8` doubled in 2P; **`StartGliderFoilGoing(thisGlider)`**; `RedrawAllGrease()` | `savedMaps`; `sparkles[]` |
| 0x2B | kInvisBonus | " | `kRewardIt`, `isOn = state`; **nothing is drawn** (`ObjectDrawAll.c:422-424`) | `points = data.c.points` read **before** the `SetObjectState` call; `kBonusSound`; `AddFlyingPoint(bounds, points, hVel/2, vVel/2)`; `/= 4`; `theScore += points`. **No `RestoreFromSavedMap`, no `RedrawAllGrease`** | `flyingPoints[]` |
| 0x2C | kStar | P0, P2, P4, P5, X | `kRewardIt` | `StopStar`; `kEnergizeSound`; `RestoreFromSavedMap`; `AddSparkle`; **no damping**; `numStarsRemaining--`; `if (numStarsRemaining <= 0) FlagGameOver(); else DisplayStarsRemaining();`; `RedrawAllGrease()`; `theScore += kStarPoints` (5000) **after** the grease redraw | **two** `savedMaps` slots (prize backup + one 32 × 186 six-frame strip); a `stars[]` slot; `sparkles[]` |
| 0x2D | kSparkle | P0, P1 | **no hot spot at all** — `ObjectRects.c:719-725` computes a rect and never calls `AddActiveRect`. Registers a dinah, **central room only**, gated on `SectRect` and `data.c.state` | P1 `HandleSparkleObject`: idle countdown then `AddSparkle` + `kMysticSound`, see below | a `dinahs[]` slot; `sparkles[]` |
| 0x2E | kHelium | P0, P2, P4, X | `kRewardIt` | as kBattery but `if (batteryTotal < 0) batteryTotal -= 150; else batteryTotal = -150;` then a separate `if 2P` `-= 150` | `savedMaps`; `sparkles[]` |
| 0x2F | kSlider | P0, P2 | **one `kSlideIt` rect, always** — `QSetRect(0,0,data.c.length,16)` at `topLeft`, `isOn` hardcoded `true` (`ObjectRects.c:727-732`). Draws nothing | `sliding = true; vVel = bounds.top - dest.bottom` | — |

Two things about `kSlider`. It is **dead in `HandleRewards`** (`Interactions.c:978-979` is an empty
`break`) and it is **live as a slide surface** — a bare "dead" note in the reward table would drop it
from the port. Its 16 px-tall rect makes the `vVel` snap much stronger than grease's 2 px line, and
it is not restored by `SetObjectsToDefaults`, which is inert because nothing consumes its state.

**Reward damping is not uniform.** The four clocks and `kInvisBonus` use `AddFlyingPoint` and damp to
a **quarter**; the five supplies use `AddSparkle` and damp to a **half**; `kStar` uses `AddSparkle`
and does **not** damp at all; the two greases do neither. `AddFlyingPoint`'s sprite is chosen by a
`switch` on the literal `points` value (`DynamicMaps.c:222-248`): 100 → frames 12..14, 250 → 9..11,
300 → 6..8, 500 → 3..5, **default → 0..2**. So `kCuckoo`'s 1000 renders the default digit pair — the
"1000" label is not something the code can draw — and 250 is reachable only through `kInvisBonus`.

**The battery/helium sign rule.** One signed counter: `> 0` is battery, `< 0` is helium, `0` is empty.
Both tests are strict, so at exactly 0 the `else` branch runs and *replaces* rather than adds — the
same value the add would have produced, so no visible difference at 0. Picking up a battery while
holding helium **discards the helium**, and vice versa. The two-player doubling is a separate
statement, not an `else`, so in 2P with both alive a battery from empty gives 100 and helium from
empty gives -300. Blink thresholds (`Scoreboard.c:74-77`): `kBatteryLow 17`, `kHeliumLow -38`,
`kFoilLow 2`, `kBandsLow 2`.

**`who->isOn = false` is unconditional** after every non-dead reward case (`:779`, `:795`, `:811`,
`:828`, `:847`, `:869`, `:888`, `:897`, `:916`, `:930`, `:950`, `:975`) — it sits *outside* the
`if (SetObjectState(...))` block, so a second glider touching an already-taken prize in the same
frame still disarms the rect. In the P4 band path the polarity is the opposite: `isOn = false` is
*inside* the grease-only branch (`RubberBands.c:128`).

**A band can only knock over grease.** `CheckBandCollision`'s `kRewardIt` arm
(`RubberBands.c:118-130`) reads `whoLinked`, tests `what == kGreaseRt || what == kGreaseLf`, and does
nothing for the other twelve. No clock, battery, foil, bands or star is band-collectable.

**Persistence.** Consumption writes the live house handle (`Objects.c:462`), so the prize is gone for
the whole game: re-entering the room re-runs `DrawLocale`, `IsThisValid` rejects it, no art is drawn
and no hot spot is made. Dying does not restore it. Only `SetObjectsToDefaults` does.

**`HandleSparkleObject` (`Dynamics.c:293-317`)** — the only P1 handler keyed on `active` rather than
`timer`, and note the empty `else`:

```c
if (dinahs[who].active)                          // :297
{
	if (dinahs[who].frame <= 0)                  // :299   idle?
	{
		dinahs[who].timer--;                     // :301
		if (dinahs[who].timer <= 0)              // :302
		{
			dinahs[who].timer = RandomInt(240) + 60;   // :304
			dinahs[who].frame = kNumSparkleModes;      // :305   = 5, a LOCKOUT not a sprite
			tempRect = dinahs[who].dest;               // :306
			AddSparkle(&tempRect);                     // :307
			PlayPrioritySound(kMysticSound, kMysticPriority);   // :308
		}
	}
	else
		dinahs[who].frame--;                     // :312
}
else
{                                                // :314-316   deliberately empty
}
```

The initial timer is `RandomInt(60)+15` (`Dynamics3.c:208`) but the re-arm is `RandomInt(240)+60`, so
the first sparkle of a room comes within 15..75 frames and later ones every 60..300. If the 3-slot
`sparkles[]` array is full, `AddSparkle` silently does nothing and the lockout and sound still happen.

**Grease geometry.** A *statically* spilled jar (`state == false` at compose time) has no `grease[]`
entry, no `savedMaps` slot and no cap: the trail is painted into `backSrcMap` once and one `kSlideIt`
rect is registered. A *runtime* spill is a `grease[]` entry with a 32 × 108 four-frame strip whose
hot spot is mutated in place from `kRewardIt` to `kSlideIt` by `HandleGrease` in P5 — which means the
promotion happens **after** that frame's P2, so the first frame the player can be caught is the
fifth. During the four fall frames the rect is still `kRewardIt` with `isOn == false`: neither
collectable nor slippery. `SpillGrease(who, index)` takes a **grease index** and a **hot-spot index**
(`masterObjects[].dynaNum` and `.hotNum`) and no-ops unless `mode == kGreaseIdle`. Five call sites:
`RubberBands.c:126`, `Interactions.c:895` (glider), `Interactions.c:1060` (a switch's link),
`Triggers.c:117` and `:187`. Only three of the five gate on `SetObjectState`, so **the switch path
spills visually and mechanically but leaves the house saying "upright"** — re-entering the room
restores the jar whole.

`kSlideIt` (`Interactions.c:1376-1379`) is two lines: `sliding = true;` and
`vVel = who->bounds.top - thisGlider->dest.bottom;` — the only direct `vVel` write in the whole
dispatcher, bypassing the desired-velocity ramp. It is not a hard snap: `MoveGlider` then steps
`vVel` by `kVImpulse` (2) toward `kGravity` (3) before applying it, so the glider settles with its
feet exactly 2 px below the line and stays overlapping the 2 px rect frame after frame. `sliding`
does exactly two things: it selects sprite 30/29 in `MoveGliderNormal` (which is also its only
clear, `Player.c:159`/`:181`) and it makes `CheckRoofCollision` skip its always-fatal landing test
(`Interactions.c:455`). There is no friction, traction or `hVel` effect anywhere.

---

### 3.5 Transports — `transportType`, 0x31-0x40, 16 types

`transportType` is `topLeft(4) tall(2) where(2) who(1) wide(1)`. `where` is a packed floor/suite
link, `who` the destination object index with 255 meaning unlinked. All sixteen are `changed = false`
in `SetObjectState` except `kDeluxeTrans`.

| code | type | pass | hot spot / trigger | effect | live state |
|---|---|---|---|---|---|
| 0x31 | kUpStairs | P0, P2 | `kMoveItUp`, 112 × 32 at `topLeft`, always on. **Not gated on `who != 255`** — the stairs carry no link | `if (!heldRight && GliderInRect(...))` then the burning / two-player / plain three-way, ending in `StartGliderGoingUpStairs` | — |
| 0x32 | kDownStairs | P0, P2 | `kMoveItDown`, 80 × 56 anchored at the sprite's right edge, y offset 170 | `!heldLeft`, mirror of the above | — |
| 0x33 | kMailboxLf | P0, P2 | `kMailItLeft`, 72 × 40 spanning `topLeft + (-42,16)`, **only if `data.d.who != 255`** | five-level gate incl. a facing term, then `StartGliderMailingIn` and the dispatcher assigns `mode = kGliderMailInLeft` | pending-transit triple; `activeRectEscaped` in 2P |
| 0x34 | kMailboxRt | P0, P2 | `kMailItRight`, 72 × 40 at `topLeft + (79,16)`, same gate | mirror, `kGliderMailInRight` | " |
| 0x35 | kFloorTrans | P0, P2 | `kDuctItDown`, 76 × 48 **above** the object, `QOffsetRect(-8, RectTall(srcRects[kFloorTrans]))`, gated on `who != 255` | four-level gate, `StartGliderDuctingDown` | " |
| 0x36 | kCeilingTrans | P0, P2 | `kDuctItUp`, 76 × 48 **below** the object, `QOffsetRect(-8, 0)`, gated on `who != 255` | the only case with **five** gate terms, and the one-player arm alone latches `who->stillOver = true` | " |
| 0x37 | kDoorInLf | P0, P2 | `kIgnoreLeftWall`, 16 × 240 at `topLeft + (0,52)` | `ignoreLeft = true` | — |
| 0x38 | kDoorInRt | P0, P2 | `kIgnoreRightWall`, 16 × 240 at `+(128,52)` | `ignoreRight = true` | — |
| 0x39 | kDoorExRt | P0, P2 | `kIgnoreRightWall`, 16 × 240 at `+(0,52)` | `ignoreRight = true` | — |
| 0x3A | kDoorExLf | P0, P2 | `kIgnoreLeftWall`, 16 × 240 at `+(0,52)` | `ignoreLeft = true` | — |
| 0x3B | kWindowInLf | P0, P2 | `kIgnoreLeftWall`, 16 × 44 at `+(0,96)` | " | — |
| 0x3C | kWindowInRt | P0, P2 | `kIgnoreRightWall`, 16 × 44 at `+(4,96)` | " | — |
| 0x3D | kWindowExRt | P0, P2 | `kIgnoreRightWall`, 16 × 44 at `+(0,96)` | " | — |
| 0x3E | kWindowExLf | P0, P2 | `kIgnoreLeftWall`, 16 × 44 at `+(0,96)` | " | — |
| 0x3F | kInvisTrans | P0, P2 | `kTransportIt`, gated on `who != 255`: `QSetRect(0,0,64,32)` offset by `topLeft`, **then** `bottom = top + data.d.tall; right += (short)data.d.wide;`. `isOn` hardcoded `true` | the transport five-level gate, `StartGliderTransporting` | pending-transit triple; `activeRectEscaped` |
| 0x40 | kDeluxeTrans | P0, P2, X | `kTransportIt`, gated on `who != 255`: `wide = (data.d.tall & 0xFF00) >> 8; tall = data.d.tall & 0x00FF; QSetRect(0,0,wide*4,tall*4)` offset by `topLeft`; **`isOn = data.d.wide & 0x0F`** | as kInvisTrans. The only switchable transport: `SetObjectState` flips the low nibble of `wide` and writes `hotSpots[hotNum].isOn` | " |

Four `kIgnore*Wall` types on each side. `hotspot-pipeline` names the four left-hand ones and gives
only bare line numbers for the right-hand ones; the two right-hand *exterior* types
(`kDoorExRt` 0x39, `kWindowExRt` 0x3D) appear nowhere in the eight reports at all. Note the
asymmetry: `kDoorInLf` and `kDoorExLf` produce the same rect at the same offset and the same action,
whereas `kDoorInRt` is offset `+128` and `kDoorExRt` is offset `+0`. That is not a typo — an interior
right-hand door's frame is 128 px right of its `topLeft`, an exterior one's is not.

**The `who != 255` creation gate** is shared by `kMailboxLf`, `kMailboxRt`, `kFloorTrans`,
`kCeilingTrans`, `kInvisTrans` and `kDeluxeTrans` (`ObjectRects.c:751`, `:763`, `:774`, `:786`, `:872`,
`:885`). An unlinked mailbox, duct or transporter is inert scenery. `DoLink`/`DoUnlink`
(`Link.c:270-319`, `:325+`) always set or clear `data.d.where` (-1) and `data.d.who` (255) together,
so either test is a reliable "is linked".

The five link-bearing types are *departure* points. The **arrival** side is a different list:
`UpdateLinkControl`'s `kTransportLinkOnly` case (`Link.c:162-186`) permits `kMailboxLf`,
`kMailboxRt`, `kCeilingTrans`, `kInvisTrans`, `kDeluxeTrans`, `kInvisLight`, `kOzma`, `kMirror`,
`kFireplace`, `kWallWindow`, `kCalendar`, `kBulletin`, `kCloud`. **`kFloorTrans` is absent**, so a
floor duct is a one-way exit and you always arrive at a ceiling duct — which is why
`WhatAreWeLinkedTo` has no `kFloorTrans` case and `kLinkedToFloorDuct` is unreachable. The nine
non-mailbox non-ceiling-duct entries all fall into that function's `default` and are pure "arrive
here" markers: the glider materialises centred in the target's `GetObjectRect` in
`kGliderTransportingIn`.

The four transport-family P2 cases (`kTransportIt`, `kMailItLeft`, `kMailItRight`, `kDuctItDown`,
`kDuctItUp`) share one five-level shape. Transcribed once, for `kTransportIt`
(`Interactions.c:1381-1414`); the others differ only in the excluded modes, the sentinel, and (for
the mailboxes) an extra facing term:

```c
if (thisGlider->mode == kGliderBurning)                              // :1382
	{ wasMode = 0; StartGliderFadingOut; kFadeOutSound; }             // :1384-1386
else if ((GliderInRect(thisGlider, &who->bounds)) &&                 // :1388
		(thisGlider->mode != kGliderTransporting) &&                  // :1389
		(thisGlider->mode != kGliderFadingOut))                       // :1390
{
	if ((twoPlayerGame) && (!onePlayerLeft))                          // :1392
	{
		if (otherPlayerEscaped == kNoOneEscaped)                      // :1394
		{
			if (thisGlider->mode != kGliderInLimbo)                   // :1396
			{
				activeRectEscaped = index;                            // :1398
				StartGliderTransporting(thisGlider, who);             // :1399
			}
		}
		else if (otherPlayerEscaped == kPlayerTransportedOut)         // :1402
		{
			if ((thisGlider->mode != kGliderInLimbo) &&
					(activeRectEscaped == index))                     // :1404-1405
				StartGliderTransporting(thisGlider, who);             // :1407
		}
	}
	else
		StartGliderTransporting(thisGlider, who);                     // :1412
}
```

The mailbox facing term (`:1435-1436`) is
`((facing == kFaceRight) && !tipped) || ((facing == kFaceLeft) && tipped)` — literally
`facing == kFaceRight XOR tipped`, which given `Input.c:314`/`:324`/`:328` means "the glider is
thrusting rightward this frame", or, with no direction key held, "the glider faces right".
`kMailItRight` is the mirror. `StartGliderMailingIn` does **not** set `mode`
(`Modes.c:155-177`); the dispatcher assigns it at `:1446`/`:1455`/`:1462`.

The stairs cases (`kMoveItUp` `:1250-1283`, `kMoveItDown` `:1285-1318`) have no
`activeRectEscaped` check, so **two up-staircases in one room cross-talk** in two-player mode.
`heldRight` is set at `Input.c:315` only when the right key alone is held; holding both keys sets
`heldLeft` only (`:309`).

`kDeluxeTrans` is a `HandleSwitches` link target that does nothing visible
(`Interactions.c:1068-1069` is an empty break) yet `SetObjectState` **did** change its state and its
hot spot's `isOn` — which arms or disarms the transporter. A porter reading the switch dispatch alone
would conclude a switch linked to a deluxe transporter is inert.

---

### 3.6 Switches and triggers — `switchType`, 0x41-0x49, 9 types

`switchType` is `topLeft(4) delay(2) where(2) who(1) type(1)`. Six kinds emit `kSwitchIt`, two emit
`kTriggerIt`, one emits `kSoundIt`. All nine are `changed = false` targets in `SetObjectState` (bar
the `kKnifeSwitch` hole), so a switch cannot switch a switch — but a *trigger* can fire one
(`FireTrigger`'s switch case includes `kKnifeSwitch`).

| code | type | pass | hot spot | effect | live state |
|---|---|---|---|---|---|
| 0x41 | kLightSwitch | P0, P2, P3, P4 | `kSwitchIt`, `srcRects[what]` zero-cornered at `data.e.topLeft`, **only if `data.e.where != -1`** | `HandleSwitches(who)` | `hotSpots[].stillOver` |
| 0x42 | kMachineSwitch | " | " | " | " |
| 0x43 | kThermostat | " | " | " | " |
| 0x44 | kPowerSwitch | " | " | " | " |
| 0x45 | kKnifeSwitch | " | " | " | " |
| 0x46 | kInvisSwitch | " | " ; **draws nothing** (`ObjectDrawAll.c:573-575` is just the `dynaNum` assignment) | `HandleSwitches`, but **plays no sound** (`Interactions.c:1030-1031`) while still doing the `CopyRectBackToWork`/`AddRectToWorkRects` pair | " |
| 0x47 | kTrigger | P0, P2, P4, P3 | `kTriggerIt`, same rect and same `where != -1` gate | `ArmTrigger(who)` | a `triggers[]` slot; `hotSpots[].stillOver` |
| 0x48 | kLgTrigger | " | " | " | " |
| 0x49 | kSoundTrigger | P0, P2, P3 | `kSoundIt`, a **hardcoded 48 × 48** rect at `topLeft`, created only `if (LoadTriggerSound(data.e.where) == noErr)` | `if (!who->stillOver) { PlayPrioritySound(kTriggerSound, kTriggerPriority); who->stillOver = true; }` | `hotSpots[].stillOver` |

**`kSoundTrigger` is not a trigger.** Its `data.e.where` is a *sound resource ID*, not a floor/suite
link, which is why `GetRoomLinked`/`GetObjectLinked` (`Objects.c:126-215`) do not list it and return
-1 for it via `default`, and why its rect is 48 × 48 rather than the object's 32 × 32 `srcRects`
entry. It never enqueues anything; it just plays a sound behind a `stillOver` latch. It is, however,
a legal *target*: both `HandleSwitches` (`:1072`) and `FireTrigger` play `kChordSound`/`kTriggerSound`
when a switch or trigger points at one.

**`HandleSwitches` (`Interactions.c:985-1155`).** `if (who->stillOver) return;` (`:990-991`), then
`whoLinked = who->who`, `roomLinked = masterObjects[whoLinked].roomLink`,
`objectLinked = ...objectLink`, `linkIndex = ...localLink`. Then
`if (SetObjectState(roomLinked, objectLinked, masterObjects[whoLinked].theObject.data.e.type, linkIndex))`:
build `newRect` from the bounds offset by `playOrigin`, redraw the switch art in the position of the
**global `newState`** (i.e. the *target's* new state), play `kSwitchSound`, `CopyRectBackToWork`,
`AddRectToWorkRects`. Then, as its own nested `if (linkIndex != -1)` (`:1036`), a switch on
`masterObjects[linkIndex].theObject.what`:

- nine prizes (`kRedClock`, `kBlueClock`, `kYellowClock`, `kPaper`, `kBattery`, `kBands`, `kFoil`,
  `kStar`, `kHelium`) → `RestoreFromSavedMap(roomLinked, objectLinked, true)` then
  `AddSparkle(&bounds)`. Because `doSparkle = true` already adds a correctly-placed sparkle and plays
  `kFadeOutSound` (`DynamicMaps.c:153-159`), `:1050` adds a **second** one from an uninitialised
  rect. A switch destroys a prize the player never touched, for no score and no supply.
- `kCuckoo` → `RestoreFromSavedMap(...,true)` then `StopPendulum`. No `AddSparkle`.
- `kGreaseRt`/`kGreaseLf` → `SpillGrease(dynaNum, hotNum)` with **no `SetObjectState`**.
- all eight light types → `RedrawRoomLighting()`.
- `kToaster`→`ToggleToaster`, `kMacPlus`→`ToggleMacPlus`, `kGuitar`→`PlayPrioritySound(kChordSound)`,
  `kTV`→`ToggleTV`, `kCoffee`→`ToggleCoffee`, `kOutlet`→`ToggleOutlet`, `kVCR`→`ToggleVCR`,
  `kStereo`→`ToggleStereos`, `kMicrowave`→`ToggleMicrowave`, `kBalloon`→`ToggleBalloon`,
  `kCopterLf`/`Rt`→`ToggleCopter`, `kDartLf`/`Rt`→`ToggleDart`, `kBall`→`ToggleBall`,
  `kDrip`→`ToggleDrip`, `kFish`→`ToggleFish`, all on `masterObjects[linkIndex].dynaNum`.
- `kSoundTrigger` → `PlayPrioritySound(kTriggerSound, kTriggerPriority)`.
- `kInvisBonus`, `kSlider`, `kDeluxeTrans`, `kShredder` → nothing here, though `SetObjectState`
  already changed their state.
- **No `default`.** A switch linked to furniture, a transport other than `kDeluxeTrans`, clutter or
  `kCobweb` produces no visible effect beyond the switch art — but a switch linked to a **blower**
  very much does act: the work happens one level up, inside `SetObjectState`'s blower family, which
  plays `kBlowerOn`/`kBlowerOff` (the only `PlayPrioritySound` calls in all of `Objects.c`, at
  `:411-414`) and writes `hotSpots[hotNum].isOn`. That is the entire on/off mechanism for every vent
  and fan in the game.

Finally `who->stillOver = true;` at `:1154`, **outside** the `SetObjectState` guard.

The switch's own state is never stored: `DrawLightSwitch(&newRect, newState)` draws the *target's*
new state. A switch linked to a prize therefore always draws OFF, because the prize branch forces
`newState = false`. A switch linked to something `SetObjectState` never writes `newState` for draws
whatever the last unrelated call left in the global. `ObjectDrawAll.c:508-570` composes the same art
from `GetObjectState(room, obj)` — the target's stored state — confirming the intent: **a switch is a
remote indicator, not a stateful object.**

**`ArmTrigger` (`Triggers.c:34-55`)**:

```c
if (who->stillOver) return;                 // :38-39
where = FindEmptyTriggerSlot();             // :41   first i in 0..15 with !armed, else -1
if (where != -1) {                          // :43
	whoLinked = who->who;
	triggers[where].room   = masterObjects[whoLinked].roomLink;
	triggers[where].object = masterObjects[whoLinked].objectLink;
	triggers[where].index  = whoLinked;
	triggers[where].timer  = masterObjects[whoLinked].theObject.data.e.delay * 3;
	triggers[where].what   = masterObjects[triggers[where].object].theObject.what;   // :50
	triggers[where].armed  = true;
}
who->stillOver = true;                      // :54   OUTSIDE the if
```

Queue size 16. When it is full nothing is enqueued, no sound, no feedback, and `stillOver` is
**still** latched — the glider must leave the rect and come back for another attempt. Triggers are
freed only by firing (`:91`) or by `ZeroTriggers` on a room change.

`:50` deserves a decision. `triggers[].what` is written and **never read anywhere**, but the write
itself indexes `masterObjects[triggers[where].object]` where `.object` is a **room-object slot 0..23**
(or `-1` when `GetObjectLinked` returned -1 because `data.e.who == 255`). In C that is a garbage read
of a field nothing reads; in Go it panics. **Drop the write.** Record the deviation.

Timer arithmetic: `timer = delay * 3`; `HandleTriggers` decrements then tests (`:87-88`); so delay
`d` fires on the `max(1, 3d)`-th call, i.e. `3d - 1` frames later, and **`d == 0` arms and fires in
the same frame**. A trigger armed by a *band* costs one extra frame because `HandleBands`
(`Play.c:485`) follows `HandleTriggers` (`:484`); and because `HandleTriggers` runs after
`HandleDynamics`, a trigger that fires on frame N takes visible effect from frame N+1.

**`FireTrigger` (`Triggers.c:100-193`)** dispatches on `masterObjects[localLink].theObject.what`,
under `if (masterObjects[triggerIs].localLink != -1)`. Its target list is **shorter** than
`HandleSwitches`':

| target | action |
|---|---|
| `kGreaseRt`, `kGreaseLf` | `if (SetObjectState(room, object, kForceOn, triggeredIs)) SpillGrease(dynaNum, hotNum)` |
| `kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, **`kKnifeSwitch`**, `kInvisSwitch` | `TriggerSwitch(masterObjects[triggeredIs].dynaNum)` = `HandleSwitches(&hotSpots[dynaNum])` |
| `kSoundTrigger` | `PlayPrioritySound(kChordSound, kChordPriority)` — with a `// Change me` comment |
| `kToaster` | `TriggerToast(dynaNum)` |
| `kGuitar` | `kChordSound` |
| `kCoffee` | `kCoffeeSound` (a sound only — **not** `ToggleCoffee`) |
| `kOutlet` | `TriggerOutlet(dynaNum)` |
| `kBalloon` | `TriggerBalloon(dynaNum)` |
| `kCopterLf`, `kCopterRt` | `TriggerCopter(dynaNum)` |
| `kDartLf`, `kDartRt` | `TriggerDart(dynaNum)` |
| `kDrip` | `TriggerDrip(dynaNum)` |
| `kFish` | `TriggerFish(dynaNum)` |

**There is no `kBall` case, and no light, prize, `kTV`, `kMacPlus`, `kVCR`, `kStereo`, `kMicrowave`
or `kDeluxeTrans` case.** A trigger cannot turn a light on, start a TV, or launch a ball; only a
switch can. `TriggerSwitch` works only because `DrawARoomsObjects` writes
`dynamicNum = masterObjects[i].hotNum` for the six switch kinds (`ObjectDrawAll.c:518`, `:531`,
`:544`, `:557`, `:570`, `:574`) — so for a switch, `dynaNum` **is** its hot-spot index. Fragile, and
that assignment sits *outside* the `SectRect`, unlike every other registration.

The `else` arm (`localLink == -1`, `Triggers.c:172-192`) sets `triggeredIs = -1` and then, for grease
only, calls `SetObjectState(..., -1)` and `SpillGrease(masterObjects[-1].dynaNum, masterObjects[-1].hotNum)`.
That is an out-of-bounds read in C. Go must guard it; record the deviation.

`ObjectAdd.c:503-506` sets `data.e.type = kOneShot` (3) for `kTrigger`/`kLgTrigger` and `kToggle` (0)
for every other switch at creation. `type` is read only as `SetObjectState`'s `action`
(`Interactions.c:999`), and triggers never reach `HandleSwitches` with their own type, so `kOneShot`
normally never reaches `SetObjectState` — unless a hand-edited house sets type 3 on a real switch,
in which case none of the three inner `switch (action)` blocks matches and `changed` is garbage.

**`FlagStillOvers` (`Interactions.c:1715-1732`)**, already in `player.Env`, is the third entry point:
on arrival-without-motion it sets `stillOver = true` for every `isOn` hot spot the glider intersects
and false otherwise, and unlike `CheckForHotSpots` it also *clears* `stillOver` on hot spots whose
`isOn` is false. Its effect here is to **suppress** the switch or trigger the glider materialises on
top of.

---

### 3.7 Lights — `lightType`, 0x51-0x58, 8 types

`lightType` is `topLeft(4) length(2) byte0(1) byte1(1) initial(1) state(1)`.

**No light type creates a hot spot** (`ObjectRects.c:930-938` is a bare `break` for all eight) and
none registers a dinah. Their entire runtime behaviour is: (a) contributing to `numLights` when
`data.f.state` is true, and (b) being a `HandleSwitches` link target whose effect is
`RedrawRoomLighting()`. They are **not** decorative — a light switch is the only thing in the game
that can black out or reveal a whole room.

| code | type | pass | trigger | effect | live state |
|---|---|---|---|---|---|
| 0x51 | kCeilingLight | P0, X | a switch linked to it | `SetObjectState` writes `data.f.state` in the house and both copies; `HandleSwitches` then calls `RedrawRoomLighting()` | — (state lives in the house) |
| 0x52 | kLightBulb | " | " | " | — |
| 0x53 | kTableLamp | " | " | " | — |
| 0x54 | kHipLamp | " | " | " | — |
| 0x55 | kDecoLamp | " | " | " | — |
| 0x56 | kFlourescent | " | " | " ; `GetObjectRect` additionally does `itsRect->right = data.f.length` (`ObjectRects.c:180-193`) | — |
| 0x57 | kTrackLight | " | " | " ; same `length` override | — |
| 0x58 | kInvisLight | " | " ; **also a legal transport arrival marker** (`Link.c:162-186`) | " | — |

So the group is **not** uniform, and a spec that says "the eight lights behave identically" is wrong
on three counts: two of them stretch their rect from `length`, one is an arrival marker, and only
these eight plus five door/window types feed `numLights`.

**`RedrawRoomLighting` (`RoomGraphics.c:434-461`)** is re-entrant into the middle of P2: it is called
from `HandleSwitches` (`Interactions.c:1083`), which is called from `HandleHotSpotCollision`, which is
called from the `for (i = 0; i < nHotSpots; i++)` sweep in `CheckForHotSpots`. Mid-sweep it performs
a full central-room recomposition — `DrawRoomBackground`, `DrawARoomsObjects(central, true)`,
`RestoreWorkMap()` — plus `UpdateOutletsLighting(localNumbers[kCentralRoom], numLights)` at `:453`
and a fourth write of `shadowVisible` at `:459`. It is **not** a table-rebuild hazard: `redraw = true`
suppresses every `Add*` and the `dynaNum` correlation loop (all guards at `ObjectDrawAll.c:452`-`:953`
include `!redraw`), substituting `ReBackUp*` calls, so `nHotSpots`, `hotSpots[]`, `dinahs[]`,
`grease[]` and `masterObjects[]` are all stable across it. It is guarded by `wasLit != isLit` at
`:448`, so turning off one of two lights does not fire it at all — which is why an outlet's baked-in
light count can go stale.

---

### 3.8 Appliances — `applianceType`, 0x61-0x6E, 14 types

`applianceType` is `topLeft(4) height(2) byte0(1) delay(1) initial(1) state(1)`. Nine of the
fourteen register a `dinahs[]` entry; three are static lethal solids with no dynamic at all; one is
a sound-only guitar; one is a picture.

| code | type | pass | hot spot | dinah / P1 handler | live state |
|---|---|---|---|---|---|
| 0x61 | kShredder | P0, P2, X | `kShredIt`, `srcRects[kShredder]` with `bottom = top + kShredderActiveHigh (40)`, `right += 48`, offset `(-24,-36)`; `isOn = data.g.state`, scrutinize | **none** — a shredder has no dinah | — |
| 0x62 | kToaster | P0, P1, P2, P3, X | `kDissolveIt` on the body (shared appliance case) | `HandleToast`. Registered **central room only** | `dinahs[]` |
| 0x63 | kMacPlus | P0, P1, P2, X | `kDissolveIt` | `HandleMacPlus`. All nine rooms | `dinahs[]` |
| 0x64 | kGuitar | P0, P2, P3, X | `kStrumIt`, a hardcoded 8 × 96 at `topLeft + (34,32)`, always on, scrutinize false | none | `hotSpots[].stillOver` |
| 0x65 | kTV | P0, P1, P2, X | `kDissolveIt` | `HandleTV`. All nine rooms | `dinahs[]`; the QuickTime globals `tvInRoom`, `tvWithMovieNumber`, `tvOn`, `hasMovie` |
| 0x66 | kCoffee | P0, P1, P2, P3, X | `kDissolveIt` | `HandleCoffee`. All nine rooms | `dinahs[]` |
| 0x67 | kOutlet | P0, P1, P2, P3, X | **`kIgnoreIt`**, `srcRects[kOutlet]` at `topLeft`, `isOn = data.g.state` — inert for the glider | `HandleOutlet`. All nine rooms | `dinahs[]` |
| 0x68 | kVCR | P0, P1, P2, X | `kDissolveIt` | `HandleVCR`. All nine rooms | `dinahs[]` |
| 0x69 | kStereo | P0, P1, P2, X | `kDissolveIt` | `HandleStereo`. All nine rooms | `dinahs[]`; the global `isPlayMusicGame` |
| 0x6A | kMicrowave | P0, P1, P2, X | **two** rects: `kDissolveIt` on the body, then `bounds.bottom = bounds.top; bounds.top = 0;` and `kMicrowaveIt` — the full-height column **above** the appliance. Both scrutinize | `HandleMicrowave`. All nine rooms | `dinahs[]` |
| 0x6B | kCinderBlock | P0, P2 | `kDissolveIt` (shares the appliance case at `:981-996`) | **none** | — |
| 0x6C | kFlowerBox | P0, P2 | `kDissolveIt` | none | — |
| 0x6D | kCDs | P0, P2 | `kDissolveIt` | none | — |
| 0x6E | kCustomPict | P0 | **none** (`:998-999`) | none | — (draw-only; `GetObjectRect` reads the PICT's `picFrame` and rewrites `data.g.height = 10000` if the resource is missing) |

`kCinderBlock`, `kFlowerBox` and `kCDs` are the trap in this group: they are grouped with the
toggleable machines in `CreateActiveRects` and so are **lethal solids**, but they have no
`AddDynamicObject` case, no `Toggle*`, no `SetObjectState` case (`changed = false`) and no entry in
`SetObjectsToDefaults`. Static killers that look like appliances.

**Registration shape** (from `DrawARoomsObjects`): the seven timer appliances use the
visibility-gated all-nine-rooms shape — `GetObjectRect`, `OffsetRectRoomRelative`,
`if (SectRect(&itsRect, &testRect, &whoCares)) { Draw<Kind>(...); if (!redraw) { rectA = itsRect;
QOffsetRect(&rectA, -playOriginH, -playOriginV); dynamicNum = AddDynamicObject(...); } }` — so an
appliance in a **neighbour** room gets a dinah if a sliver of it is on screen, and gets **none** if
the window is too small. `kToaster` uses the same shape with the inner guard tightened to
`if ((!redraw) && (neighbor == kCentralRoom))`.

**Six of the nine P1 handlers share one skeleton**, not eight as one report has it:

```c
if (dinahs[who].timer > 0)          // timer == 0 means fully inert
{
	dinahs[who].timer--;            // decrement FIRST
	if (dinahs[who].active) { ...on-transition phases... }
	else                    { ...off-transition phases... }
}
```

The six are `HandleMacPlus` (`Dynamics.c:390`), `HandleTV` (`:430`), `HandleCoffee` (`:484`),
`HandleVCR` (`:605`), `HandleStereo` (`:673`), `HandleMicrowave` (`:718`). The three that do not are
`HandleSparkleObject` (opens `if (dinahs[who].active)`, §3.4), `HandleToast` (opens
`if (dinahs[who].moving)`) and `HandleOutlet` (opens `if (dinahs[who].position != 0)`).

Per-handler phase tables and reachability, which is what a porter actually needs:

- **`HandleMacPlus`** (`:388-424`). `timer == 0` → `AddRectToWorkRects`; `timer == 1` → sound +
  `CopyBits` into `backSrcMap` + `AddRectToBackRects`; active-only `timer == 30` → `kMacOnSound`.
  Init `timer = 0` → inert. `ToggleMacPlus` sets **40 on, 10 off**, so switching *off* never plays
  `kMacOnSound` (from 10 the `timer == 30` case is unreachable). On art `plusScreen2`, off
  `plusScreen1`. The two-frame handshake (`timer == 1` writes `backSrcMap`, `timer == 0` queues
  work→main) depends on `AddRectToWorkRects` meaning "blit r to the screen" and
  `AddRectToBackRects` meaning "erase r in `workSrcMap` afterwards" — verified at
  `Render.c:620-626` then `:628-634`, in that order. Note the two adders clamp against **different**
  rectangles (`Render.c:70-77` clamps to `justRoomsRect` in global coords; `:89-96` clamps to
  `0/workSrcRect`), so they are not one helper.
- **`HandleTV`** (`:428-478`). Same skeleton, no `timer == 30`. `ToggleTV` always sets `timer = 4`,
  both directions. Both *active* phases are wrapped in a QuickTime test with an **empty then-branch**
  and the real work in the `else`
  (`if ((thisMac.hasQT) && (hasMovie) && (tvInRoom) && (who == tvWithMovieNumber)) { } else { ... }`),
  i.e. when this dinah is the movie TV the still art is suppressed and the movie draws instead. The
  *inactive* branch has no such test and always paints `tvScreen1`.
- **`HandleCoffee`** (`:482-524`). Init `timer = isOn ? 200 : 0`. Active phases: `timer == 0` →
  `AddRectToWorkRects` **and `timer = 200 + RandomInt(200)`**; `timer == 1` → `kMacOnSound` +
  `coffeeLight2`; `timer == 100` → `kCoffeeSound` **and `timer = 200 + RandomInt(200)`**. An
  initially-on coffee maker therefore loops 200→100→gurgle→reload and **never reaches 1 or 0**, so
  the handler never draws the lit light — that is `DrawCoffee`'s job from `data.g.state`. The
  `timer == 1`/`0` branches are reached only via `ToggleCoffee` (`timer = 4`). Two `RandomInt(200)`
  draws per gurgle: a determinism-relevant RNG consumer.
- **`HandleOutlet`** (`:528-599`). The only handler keyed on `position` and the only one that draws
  into `workSrcMap`. Zap length is exactly 30 frames; `frame` cycles `1,2,3,1,2,3,…` (index 0 is
  reserved for the idle plug); `kZapSound` plays at launch plus `timer` 25, 20, 15, 10, 5 — five
  repeats, and never at 0 because the `% 5` test lives in the `else` of `timer <= 0`. The final frame
  is the interesting one: `position` is cleared at `:555` and then **read** at `:567`, so a lit room
  (`hVel > 0`) redraws `outletSrc[0]` while a dark room `PaintRect`s the outlet away — necessary
  because `DrawOutlet` is `isLit`-gated so the back map has no outlet to restore from. Its idle half
  has the same else-less inner guard as the toast:

  ```c
  else                                             // :582   idle
  {
      if (dinahs[who].active)                      // :584
          dinahs[who].timer--;                     // :585
      if (dinahs[who].timer <= 0)                  // :587   NOT an else
      {
          if (dinahs[who].active)                  // :589
          {
              dinahs[who].position = 1;            // :591
              dinahs[who].timer = kLengthOfZap;    // :592   = 30
              PlayPrioritySound(kZapSound, kZapPriority);
          }
          else
              dinahs[who].timer = dinahs[who].count;   // :596
      }
  }
  ```

  **The hit box is only the 16 × 24 socket** — `dest` never grows with the zap, and the outlet's hot
  spot is `kIgnoreIt`, so the dynamic channel is the outlet's *only* damage path. With
  `data.g.delay == 0`, `count == 0` and an active outlet re-zaps on the very next frame.
  `UpdateOutletsLighting(room, nLights)` (`Trip.c:235-244`) writes the **light count into
  `dinahs[i].hVel`** for every `kOutlet` in `room`; nothing else reads a kOutlet's `hVel`.
  `AddDynamicObject` seeds it from `numLights` at creation (`Dynamics3.c:314`), so a
  *neighbour-room* outlet gets its own room's count correctly at compose time and is then never
  refreshed.
- **`HandleVCR`** (`:603-667`). Init `timer = isOn ? 115 : 0`. Phases: `timer == 0` →
  `AddRectToWorkRects` + `timer = 115`; `timer == 5` → `kMacOnSound`; `timer == 1` → `kVCRSound` +
  `vcrTime2`; `timer == 100` → `AddRectToWorkRects` + `timer = 115` + **`frame = 1 - frame`**;
  `timer == 101` → draw `vcrTime2` if `frame == 0` else `vcrTime1`. An initially-on VCR loops
  115→101→100 forever: a **15-frame blinking clock colon**. **`timer == 5` is dead code** — from 115
  the reload at 100 pre-empts it and from 4 the countdown skips 5 — so `kMacOnSound` never plays for
  a VCR.
- **`HandleStereo`** (`:671-711`). Init `timer = 0` **always** (`Dynamics3.c:361`), regardless of
  `data.g.state` → inert until toggled. Both branches call `ToggleMusicWhilePlaying()` at
  `timer == 0`, so switching a stereo **off** starts the music if it was stopped. `AddDynamicObject`
  gets `data.g.state` as `isOn` while `DrawStereo` is handed `isPlayMusicGame`, so the drawn light
  and `active` can disagree.
- **`HandleMicrowave`** (`:715-775`). Init `timer = 0` → inert until `ToggleMicrowave` sets 4. At
  `timer == 1` it tiles the 16 × 35 `microOn` strip three times across the 48-wide `dest` and then
  `AddRectToBackRects` on the full 48. The microwave's danger is entirely in its two hot spots, not
  in this handler.
- **`HandleToast`** (`:321-384`) is a mover, specified with the enemies in §3.9 because it shares
  their apex→velocity idiom.

**The fourteen `Toggle*` (`Trip.c:18-141`)** — each flips `dinahs[index].active`, plus:
`ToggleToaster` nothing else; `ToggleMacPlus` `timer = 40` if now active else `10`;
`ToggleTV` the QuickTime start/stop block including the `tvOn` write, then `timer = 4`
**unconditionally, outside the QT `if`**; `ToggleCoffee` `timer = 4`; `ToggleOutlet` nothing else, so
a zapping outlet keeps zapping until its current cycle ends; `ToggleVCR` `timer = 4`;
**`ToggleStereos` is the only one with a guard** — `if (dinahs[index].timer == 0)` wraps both the
flip and `timer = 4`, so a stereo toggled twice within four frames swallows the second;
`ToggleMicrowave` `timer = 4`; `ToggleBalloon`/`Copter`/`Dart`/`Ball`/`Drip`/`Fish` nothing else.
**No `Toggle*` redraws anything**; the visible change happens on the next `HandleDynamics`.

`tvOn` has no reset anywhere. It is written at `ObjectDrawAll.c:693` (inside the shape-1 all-nine-rooms
`kTV` case, so with two TVs the last drawn wins and central is drawn ninth) and by `ToggleTV`
(`Trip.c:49`/`:54`). `DrawLocale` clears `tvInRoom` and `tvWithMovieNumber` but **not `tvOn`**, so it
is stale from the previous room in any room without a TV.

**`kShredIt` (`Interactions.c:1324-1341`)** requires **full containment** — a glider half in the
shredder is safe — and spends only one foil sheet (no `GliderHitTop`):

```c
if ((thisGlider->mode != kGliderShredding) &&
		(GliderInRect(thisGlider, &who->bounds)))                     // :1325-1326
{
	if ((foilTotal > 0) || (thisGlider->mode == kGliderLosingFoil))    // :1328
	{
		PlayPrioritySound(kFoilHitSound, kFoilHitPriority);            // :1330
		if (foilTotal > 0)                                             // :1331   <-- no else
		{
			foilTotal--;                                               // :1333
			if (foilTotal <= 0)  StartGliderFoilLosing(thisGlider);     // :1334-1335
		}
	}
	else
		FlagGliderShredding(thisGlider, &who->bounds);                 // :1339
}
```

**`kMicrowaveIt` (`Interactions.c:1581-1584` → `HandleMicrowaveAction :1159-1194`)** is three
independent `if`s, not `else if`s, and the battery test is `!= 0` (helium is negative) while the
other two are `> 0`:

```c
if (who->stillOver) return;                                          // :1164-1165
killed = false;
whoLinked = who->who;
if (masterObjects[whoLinked].theObject.data.g.state)                 // :1169
{
	kills = (short)masterObjects[whoLinked].theObject.data.g.byte0;   // :1171
	if (((kills & 0x0001) == 0x0001) && (bandsTotal > 0))            // :1172
	{ bandsTotal = 0; killed = true; QuickBandsRefresh(false); }
	if (((kills & 0x0002) == 0x0002) && (batteryTotal != 0))         // :1178
	{ batteryTotal = 0; killed = true; QuickBatteryRefresh(false); }
	if (((kills & 0x0004) == 0x0004) && (foilTotal > 0))             // :1184
	{ foilTotal = 0; killed = true; StartGliderFoilLosing(thisGlider); }
}
if (killed)  PlayPrioritySound(kMicrowavedSound, kMicrowavedPriority);
```

`stillOver` is never set, so the microwave re-zaps every frame the glider stays in the column.

**`kStrumIt` (`:1343-1349`)** is sound only:
`if (!who->stillOver) { PlayPrioritySound(kChordSound, kChordPriority); who->stillOver = true; }`.
Guitar strings are not solid.

**`kIgnoreIt` has no case label** in `HandleHotSpotCollision` at all — it falls out of the switch. It
is created for exactly two things, `kOutlet` (`isOn = data.g.state`) and the seven mobile enemies of
§3.9. It is also not one of the five actions a band can see. So it is genuinely inert: it costs one
of the 56 `hotSpots[]` slots and nothing else. It is **not** created for `kInvisObstacle`, which is a
lethal `kDissolveIt`.

---

### 3.9 Enemies — `enemyType`, 0x71-0x79, 9 types

`enemyType` is `topLeft(4) length(2) delay(1) byte0(1) initial(1) state(1)`. Eight are movers with a
`dinahs[]` entry; one (`kCobweb`) is a static trap with a hot spot and no dinah.

| code | type | pass | hot spot | dinah / P1 | lethal when | live state |
|---|---|---|---|---|---|---|
| 0x71 | kBalloon | P0, P1, P3, P4, X | `kIgnoreIt` (inert) | `Dynamics2.c:30-130`. Registered **central room only, no `SectRect`** | only while `moving` **and** `vVel < 0` (rising) | `dinahs[]`; `sparkles[]` |
| 0x72 | kCopterLf | " | `kIgnoreIt` | `:134-238` | only while `moving` **and** `hVel != 0` (not shot) | " |
| 0x73 | kCopterRt | " | `kIgnoreIt` | `:134-238` | " | " |
| 0x74 | kDartLf | " | `kIgnoreIt` | `:242-354` | " | " |
| 0x75 | kDartRt | " | `kIgnoreIt` | `:242-354` | " | " |
| 0x76 | kBall | P0, P1, X | `kIgnoreIt` | `:358-423`. **No `TriggerBall`** — `FireTrigger` has no `kBall` case, so a ball can only be toggled | **always**, even idle / switched off / come to rest | `dinahs[]` |
| 0x77 | kDrip | P0, P1, P3, X | `kIgnoreIt` | `:427-497`. Registered central room only, `SectRect`-gated | only while `moving` | `dinahs[]` |
| 0x78 | kFish | P0, P1, P3, X | **`kDissolveIt`** on the 36 × 33 bowl, scrutinize — the only mover with a lethal solid | `:501-588`. Central room only, `SectRect`-gated | only while `moving` | `dinahs[]` |
| 0x79 | kCobweb | P0, P2 | `kWebIt`, `srcRects[kCobweb]` (54 × 45) grown by `InsetRect(-24,-10)` → 102 × 65, scrutinize | none | — | `wasMode` on the glider |

Seven types take `kIgnoreIt` (`ObjectRects.c:1001-1013`), not six. `kFish` takes `kDissolveIt`
(`:1016-1022`) and `kCobweb` takes `kWebIt` (`:1025-1032`).

**Registration shape 3** — the six of `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`, `kDartRt`,
`kBall` use the **not visibility-gated, central-room-only** form
(`if ((neighbor == kCentralRoom) && (!redraw)) { GetObjectRect; OffsetRectRoomRelative;
QOffsetRect(&itsRect, -playOriginH, -playOriginV); dynamicNum = AddDynamicObject(...); }`), offsetting
`itsRect` in place rather than copying. So **enemies always animate regardless of screen size while
appliances do not**. `kDrip` and `kFish` use shape 2 (central room only *and* `SectRect`-gated).

**No randomness in any mover.** `RandomInt` appears in `Dynamics.c`/`Dynamics2.c`/`Dynamics3.c` at
exactly four sites, none of them a mover: `Dynamics3.c:208` and `Dynamics.c:304` (`kSparkle`), and
`Dynamics.c:492`/`:506` (`kCoffee`). No floating point and no sub-pixel accumulator: everything is
`short` arithmetic on `Rect` fields. The only fractional effect is the half-rate
`if (evenFrame) vVel++` used by the ball, drip and fish — and **not** by the toast.

**The two apex→velocity solvers** in `AddDynamicObject` recover a launch velocity from an authored
apex height by summing the arithmetic series. Full-rate (toast, `Dynamics3.c:224-235`) gives
`velocity` = the smallest `v` with `v(v+1)/2 >= height` and `count = +velocity`. Half-rate (ball
`:472-486`, fish `:522-535`) increments only on alternate iterations and stores `count = -velocity`
— **the opposite sign convention** — and both **stomp the global `evenFrame = true`** mid room-load
(`:474`, `:524`). That third `evenFrame` write is at `Dynamics2.c:420`, on a ball's launch frame.

Per-mover laws, all verified by simulation:

- **Toast** (`kToaster`, in the appliance group but a mover). `vVel` walks `-count … 0 … +count`
  inclusive, `2*count+1` frames, displacement summing to exactly **zero**, so the toast lands
  precisely where it launched, forever, with no drift. Apex `count*(count+1)/2` px. Gravity is
  **unconditional** (`vVel++` every frame at `:356`), unlike the ball/drip/fish. Collision is tested
  against the **previous** frame's position (`:349` precedes `:350`). Bands do not test it. The idle
  half has the else-less inner guard that is the easiest thing in this file to get wrong:

  ```c
  else                                             // :367
  {
      if (dinahs[who].active)                      // :369
          dinahs[who].frame--;                     // :370
      if (dinahs[who].frame <= 0)                  // :371   NOT an else, NOT &&
      {
          if (dinahs[who].active)                  // :373
          {
              dinahs[who].vVel = (short)-dinahs[who].count;   // :375
              dinahs[who].frame = 0;
              dinahs[who].moving = true;
              PlayPrioritySound(kToastLaunchSound, kToastLaunchPriority);
          }
          else
              dinahs[who].frame = dinahs[who].timer;          // :381   re-arm and stay idle
      }
  }
  ```

  Fusing `:369` and `:371` into `if (active && frame <= 0)` breaks the inactive case. `RenderToast`
  clips the toast into the slot: `vClip = dest.bottom - dinahs[who].hVel` (`hVel` is the slot line,
  seeded as `where->top + 2`), removing rows from the **bottom** of the sprite; at rest `vClip == 27`
  so only 2 px show.
- **Balloon.** 2 px/frame up, full-rate; horizontal position never changes; 280 → 8 in 136 frames.
  Popped: constant 8 px/frame down with **no acceleration**, and the popped arm calls neither
  `CheckDynamicCollision` nor `DidBandHitDynamic` — **a popped balloon is harmless.** Recycles on
  `dest.top <= 8` or `dest.bottom >= 310`. Sparkle-in fires at `timer == kStartSparkle` (4), or at
  the moment of appearing when `count < 4`; `count` is always a multiple of 3 so `count == 4` is
  unreachable and there is no silent no-sparkle case. `delay == 0` gives a continuous stream.
- **Copter.** (±1, +2) px/frame, full-rate, constant; `hVel = -1` for Lf, `+1` for Rt; sprite
  advances **every** frame, no `evenFrame` gate. **No horizontal bound** — a copter drifts off the
  side of the room and is clipped only by `AddRectToWorkRects`; only the vertical test recycles it,
  restoring `dest.left = position` (the spawn column). Shot: `hVel = 0` is the flag, and the shot arm
  calls neither collision function, so a shot copter is harmless.
- **Dart.** (±6, +2) px/frame, constant, full-rate, and it **never animates while flying** — one
  static sprite per direction. **Both** horizontal bounds are evaluated for **both** directions
  (`:296-297`), plus the floor line. `dest.top = position` restores the spawn row. `topLeft.h` is
  ignored at spawn (Lf starts at x=448, Rt at x=0). `.room` is **never assigned** in the dart's
  registration case (`Dynamics3.c:437-463`).
- **Ball.** The collision block at `Dynamics2.c:360-376` is **outside** `if (moving)`, so **a
  never-launched, switched-off or come-to-rest ball still kills** — the only mover with this
  property. Apex is `v*(v+1)` px, **not** `length`; flight `4v + 2` frames; because `evenFrame` is
  forced true on the launch frame every arc is identical while the ball stays active. Switched off,
  the bounce damps `-((vVel*3)/4)` with C truncation: from `v = 8` the impacts are 8, 6, 4, 3, 2, 1,
  then 0 → stop, about 108 frames. `data.h.delay` is never read: a ball has no cadence.
  `Dynamics2.c:396-397` is dead code.
- **Drip.** The swell is **one if/else-if chain**, not four ifs (`:481-494`), keyed on **absolute**
  `timer` values 6, 4 and 2 — so `count < 7` skips swell frames and `count == 0` detaches every
  frame. At detach `vVel = 0`, so the **first moving frame moves 0 px**; the 3 px nudge plus the
  12 px sprite means actual travel is `length - 15`, **not** `length`. Any `length <= 15` splashes on
  the first moving frame. Gravity is half-rate and the parity at detach is not forced anywhere.
  `hVel` holds the remembered ceiling row; `position` is the absolute splash row.
- **Fish.** Idle bob fires once every four `timer` decrements, walking `frame` 0→1→2→3→0 and moving
  ±1 px so the fish dips 2 px below the water line; full cycle 16 frames. `frame` advances **only
  while descending** (`if ((vVel >= 0) && (frame < 7)) frame++`). The splash plays **two** sounds in
  the same frame (`kDropSound` then `kFishInSound`) and pre-loads the next jump's `vVel = count`.
  `hVel` is the reload period, and `who->data.g.height` and `data.h.length` are the **same bytes**
  (both at payload offset 4). Because nothing forces `evenFrame` at a leap, **the fish alternates
  between two leap heights whenever `delay` is odd or zero, and is constant only when `delay` is a
  non-zero even number** — e.g. `length = 64` (`v = 8`) with `delay = 11` alternates 64 px / 72 px
  and 32 / 34 frames forever. A port that hard-codes one apex is wrong half the time. `RenderFish`
  uses `CopyMask` while moving but a plain opaque `CopyBits` with `srcCopy` while idle, so the idle
  fish's white pixels are stamped over the water.

**`CheckDynamicCollision(who, thisGlider, doOffset)` (`Dynamics.c:34-74`)** is the shared kill path.
Six-mode whitelist (`kGliderNormal`, `kGliderFaceLeft`, `kGliderFaceRight`, `kGliderBurning`,
`kGliderGoingFoil`, `kGliderLosingFoil`); survivability gate `(foilTotal > 0) || mode == kGliderLosingFoil`;
otherwise `StartGliderFadingOut` + `kFadeOutSound`. With foil: `hDesiredVel = ±kShoveVelocity` (8) as
an **assignment**, so two dinahs colliding in one frame means the last one wins; then, **only if the
mover is moving up** (`vVel < 0`), `vDesiredVel = dinahs[who].vVel` — a falling mover never pushes
the glider down. The foil spend is `evenFrame`-gated (`if ((evenFrame) && (foilTotal > 0))`, a real
`&&` in the C, unlike the nested-`if` shapes elsewhere in the same file), so continuous contact costs
one sheet per **two** frames. `SectGlider(..., true)` insets by 5 on all four sides *after* adding 6
to `top` for a burning glider, and uses strict `<`/`>`, so **touching edges collide** —
`image.Rectangle.Overlaps` is half-open and gives the opposite answer.
`IsRectLeftOfRect` (`RectUtils.c:181-190`) is **not** a centre-of-mass test: C precedence makes
`offset = width1 - width2/2`, not `(width1 - width2)/2`, and the second argument is the glider's raw
`dest`, not the inset box.

**`DidBandHitDynamic(who)` (`:80-106`)** is a plain AABB of every band against `dinahs[who].dest`
with **no `playOrigin` offset**, breaking on the first hit, and it does **not** kill or slow the
band — one band can pop several enemies as it flies through. `Boolean collided;` at `:84` is
uninitialised and would be returned if `numBands == 0`; all three callers guard with
`(numBands > 0) &&` (`Dynamics2.c:62`, `:162`, `:267`), which short-circuits. Return `false`
explicitly in Go and note why the C got away with it. Only three types are band-sensitive: balloon
(`frame = 6`, `vVel = 8`, `kPopSound`), copter (`frame = 8`, `hVel = 0`, `vVel = 8`,
`kPaperCrunchSound`) and dart (`frame = 1` or `3`, `hVel = 0`, `vVel = 8`, `kPaperCrunchSound`). In
all three the hit is the `if` arm of an `if/else` whose `else` performs that frame's movement, so a
hit enemy **does not move on the hit frame** and its erase rect is not updated. Nothing else is
band-sensitive — no ball, drip, fish, toast, appliance or furniture.

**The eight `Trigger*` (`Trip.c:145-231`)**: `TriggerToast` is `HandleToast`'s launch block minus the
`frame--`, guarded by `if (!moving)`; `TriggerOutlet` is the idle block guarded by
`if (position == 0)`; `TriggerDrip` is `if ((!moving) && (timer > 7)) timer = 7;` — a **clamp**, not
an assignment; `TriggerFish` is `if ((active) && (!moving)) { whole = dest; moving = true;
frame = 4; kFishOutSound }`, bypassing the timer; `TriggerBalloon`/`Copter`/`Dart` are all
`if (!moving) timer = kStartSparkle + 1;` and **none of the three checks `active`**;
`TriggerSwitch(who)` is `HandleSwitches(&hotSpots[who])`. Only `TriggerToast`, `TriggerOutlet` and
`TriggerFish` make a sound.

**`kWebIt` (`Interactions.c:1602-1613`)** has two arms and the second **re-tests** `GliderInRect`:

```c
case kWebIt:
if ((GliderInRect(thisGlider, &who->bounds)) &&
		(thisGlider->mode != kGliderBurning))                       // :1603-1604
	WebGlider(thisGlider, &who->bounds);                            // :1605
else if ((thisGlider->mode == kGliderBurning) &&
		(GliderInRect(thisGlider, &who->bounds)))                    // :1606-1607
{
	thisGlider->wasMode = 0;
	StartGliderFadingOut(thisGlider);
	PlayPrioritySound(kFadeOutSound, kFadeOutPriority);
}
break;
```

`WebGlider` (`:1736-1777`) opens with a burning test that is therefore **dead**, then:
`hDist = ((bounds->right - dest.right) + (bounds->left - dest.left)) >> 3` and the vertical
equivalent; `if (hDesiredVel != 0) { if (evenFrame) { hVel = hDist; vVel = vDist; kWebTwangSound } }`
else `{ hDesiredVel = 0; vDesiredVel = 0; }`. So **struggling does not free you** — it hauls you back
toward the web's centre and costs a twang — while the passive branch zeroes `vDesiredVel` so the
glider hangs. Then `wasMode++` **unconditionally**, and at `>= kKillWebbedGlider` (150) the glider
fades out. The `evenFrame` read at `:1756` means `Env` needs an `EvenFrame()` accessor if `WebGlider`
moves into `player`.

---

### 3.10 Clutter — `clutterType`, 0x81-0x8F, 15 types

`clutterType` is `bounds(8) pict(2)`, and `GetObjectRect` returns `*itsRect = who->data.i.bounds`
verbatim for all fifteen (`ObjectRects.c:258-267`). Fourteen of them share one bare `break` in
`CreateActiveRects` (`:1035-1049`); only `kChimes` gets a hot spot.

| code | type | pass | behaviour | live state |
|---|---|---|---|---|
| 0x81 | kOzma | P0 | **transport arrival marker** (`Link.c:162-186`). Drawn via the furniture case (`ObjectDrawAll.c:226`, `DrawPictSansWhiteObject`) but **not solid** — no hot spot | — |
| 0x82 | kMirror | P0 | arrival marker; and the only clutter type with a live registration: `if ((neighbor == kCentralRoom) && (!redraw)) { InsetRect(&itsRect, 4, 4); AddToMirrorRegion(&itsRect); }` at `ObjectDrawAll.c:907-911`, **outside** the `SectRect` that gates `DrawMirror`, so a central mirror always contributes even when entirely offscreen. `InsetRect` mutates `itsRect` | the mirror region |
| 0x83 | kMousehole | P0 | **decorative.** No hot spot, no dinah, not a legal link target, draw-only (`ObjectDrawAll.c:914`) | — |
| 0x84 | kFireplace | P0 | arrival marker; otherwise draw-only | — |
| 0x85 | kFlower | P0 | **decorative** (`DrawFlower`, `:922`) | — |
| 0x86 | kWallWindow | P0 | arrival marker; **and it counts as a light** in `GetNumberOfLights` | — |
| 0x87 | kBear | P0 | **decorative** | — |
| 0x88 | kCalendar | P0 | arrival marker; otherwise draw-only | — |
| 0x89 | kVase1 | P0 | **decorative** | — |
| 0x8A | kVase2 | P0 | **decorative** | — |
| 0x8B | kBulletin | P0 | arrival marker; otherwise draw-only | — |
| 0x8C | kCloud | P0 | arrival marker; otherwise draw-only | — |
| 0x8D | kFaucet | P0 | **decorative** | — |
| 0x8E | kRug | P0 | **decorative** | — |
| 0x8F | kChimes | P0, P2 | `numChimes++` then a `kChimeIt` rect: `srcRects[kChimes]` zero-cornered and offset by `data.i.bounds.left/top`, always on. P2: `if (!who->stillOver) { StrikeChime(); who->stillOver = true; }` | `hotSpots[].stillOver`; `numChimes`; `theChimes` |

`StrikeChime` (`Play.c:793-796`) only sets `theChimes.nextRing = 0`; the sound is emitted by the
**next** frame's `HandleTelephone`, which picks `kChime1Sound` or `kChime2Sound` by `RandomInt(2)`
and reschedules with `delayTime = kChimeDelay (180) / numChimes`, then `if (delayTime < 2)
delayTime = 2;` and `theChimes.nextRing = RandomInt(delayTime) + 1;` (`Play.c:771-788`). **There is
no division-by-zero path**: the whole chime block is inside `if (numChimes > 0)` at `:771`. One
report's open question about guarding `kChimeDelay / numChimes` should be closed; a Go guard is
harmless but does not deserve a spec paragraph. Note also that the phone block (`if (!phoneBitSet)`
at `:748`) and the chime block are **sequential ifs, not else-if**, so both can sound in one frame.

Because `numChimes` is zeroed on every room change (`ZeroFlamesAndTheLike`, `DynamicMaps.c:796`) and
gates the chime block, `theChimes.nextRing` does **not** decrement in a chime-less room and resumes
from a stale countdown on re-entry — a cross-scope interaction between a game-scope struct
(`InitTelephone`, `Play.c:733-740`, once per new game) and a room-scope count.

---

### 3.11 The no-behaviour census

Types with **no hot spot** (24, from `CreateActiveRects`): `kSparkle` (`:719-725` computes a rect and
discards it), the eight lights (`:930-938`), `kCustomPict` (`:998-999`), and the fourteen clutter
types `kOzma`…`kRug` (`:1035-1049`). Of these, ten still have behaviour: `kSparkle` has a dinah, the
eight lights feed `numLights` and are switch targets, `kMirror` feeds the mirror region.

Types with **no runtime behaviour at all** — no hot spot, no dinah, no `Add*`, no `SetObjectState`
case that changes anything, and not a legal link target — are exactly **eight**:

| type | reason |
|---|---|
| kCustomPict 0x6E | decorative; render-only, `GetObjectRect` reads the PICT's `picFrame` (Stage 1.3) |
| kMousehole 0x83 | decorative |
| kFlower 0x85 | decorative |
| kBear 0x87 | decorative |
| kVase1 0x89 | decorative |
| kVase2 0x8A | decorative |
| kFaucet 0x8D | decorative |
| kRug 0x8E | decorative |

Each verified three ways: bare `break` in `CreateActiveRects`; absent from `AddDynamicObject`'s
switch (`Dynamics3.c:197-549`) and from `UpdateLinkControl`'s `kTransportLinkOnly` list
(`Link.c:162-186`); draw-only in `ObjectDrawAll.c:591-607` (`DrawPictSansWhiteObject`) and
`:914-926` (`DrawSimpleClutter`/`DrawFlower`), all already in `internal/render`.

Near misses that are **not** in that list, recorded so a reader does not have to re-derive them:
`kInvisObstacle` (lethal `kDissolveIt`), `kInvisBounce` (`kBounceIt`), `kInvisBonus` (`kRewardIt`),
`kInvisTrans` (`kTransportIt`), `kInvisSwitch` (`kSwitchIt`, silent), `kInvisLight` (arrival marker),
`kInvisBlower` (vector-selected wind), `kSlider` (`kSlideIt`), `kCinderBlock`/`kFlowerBox`/`kCDs`
(lethal `kDissolveIt`), `kOzma` (arrival marker), `kMirror` (mirror region), `kWallWindow` (arrival
marker **and** a light source), `kSparkle` (a dinah), `kOutlet` (a dinah; its hot spot is inert).
**Every `kInvis*` type behaves exactly like its visible sibling; none of them is a placeholder.**

Types whose `SetObjectState` returns `false` unconditionally, i.e. that cannot be switched or
triggered at all: the 5 flames, all 15 furniture, `kSlider` (uninitialised), the 15 non-deluxe
transports, the 8 switch/trigger kinds (bar the `kKnifeSwitch` hole), `kGuitar`, `kCinderBlock`,
`kFlowerBox`, `kCDs`, `kCustomPict`, `kCobweb` and all 15 clutter — 68 of 117.

Types dropped by `SetObjectsToDefaults`, so a hand-edited house whose `state != initial` survives a
new game: the 5 flames, `kSlider`, `kCinderBlock`, `kFlowerBox`, `kCDs`, `kCustomPict`, `kCobweb`,
plus all furniture, all switches, all non-deluxe transports and all clutter. Behaviourally inert
(nothing consumes those bytes) but **not** byte-inert, so a port that implements "for each object,
`state = initial` by union variant" will overwrite bytes the C leaves alone.

`nLocalObj` should have no field in the port: `LC_ALL=C grep -an nLocalObj *.c` returns exactly one
line, `Objects.c:76` — the declaration. Dead in every path, not just the gameplay path.
`tempManholes[]` is `kManhole`'s only extra state and is consumed solely by
`RoomGraphics.c:277-371`, so it belongs to Stage 1.3, not here.

### 3.12 Open questions and unverified claims

1. UNVERIFIED: whether any shipped room's registerable object set actually overflows the 56-slot
   `hotSpots[]` table. The crude bound is 24 × 3 = 72 (three rects for a tall candle), so it is not
   provably safe; `AddActiveRect` degrades gracefully to `hotNum == -1`. A corpus census over the 22
   houses' 4,070 rooms would settle it, and is cheap now that `internal/house` parses them all.
2. UNVERIFIED: whether any `kTrigger`/`kLgTrigger` in the corpus has `data.e.who == 255` while
   `data.e.where != -1`. If none does, `ArmTrigger`'s `:50` write and `FireTrigger`'s
   `localLink == -1` arm are both unreachable on shipped data and the Go guards are pure insurance.
   The same census answers both.
3. UNVERIFIED: whether the 18-slot `dinahs[]` table is ever exhausted on shipped data. One report's
   census claims no room's registerable set exceeds 18 and that `AddDynamicObject`'s `return (-1)`
   (`Dynamics3.c:193-194`) is therefore never taken; two others assert reachability by arithmetic
   without a census. Keep the cap and the registration order either way — the order is what decides
   which mover wins the `hDesiredVel` assignment race and which entries get dropped at saturation.
   Do not write a fidelity test that *asserts* drops until this is settled.
4. UNVERIFIED: the exact set of `kInvisBlower`/`kLiftArea` `vector` low nibbles present in the
   corpus. If any is not 1/2/4/8 the object silently has no hot spot, which is worth a test.

---

## 4. The traps

Every entry below is something a competent porter reading the C would plausibly get wrong: an
inverted outcome, a statement order that looks redundant, an index that is not the index it appears
to be, or an original bug that is load-bearing. Restatements of what the code plainly does are in
§1 (the frame and the pipelines) and §2 (live state); this section only carries the places where the obvious
reading is the wrong one.

Ordering is by cost of the mistake, in tiers. Within a tier, the entries a porter is most likely to
hit first come first. Citations are to the CR-converted sources (`tr '\r' '\n'`), which is the
convention the rest of these documents use. `Grease.c` and `RubberBands.c` citations are taken from
`bands-grease.md`, which is exact for those two files; `RoomGraphics.c`, `Render.c` and `Play.c`
citations are taken from the other reports, because `bands-grease.md`'s numbers for those files run
one low.

Anything I could not confirm in the C myself is marked `UNVERIFIED:` inline.

---

### 4.A Crashes and out-of-bounds reads that the C survived and Go will not

These are the ones that turn a working port into a panicking port on shipped house data. Every one
of them is reachable in one of the 22 original houses, or is reachable in principle with no guard
anywhere in the C to stop it. In each case the recommendation is the same shape: guard it, return
the C's observable answer, and record the deviation in a comment — do not transcribe the raw
dereference, and do not "fix" the semantics while you are there.

**1. `SetObjectState` has no bounds test on `room` or `object`, no `default:` in its outer switch,
and shipped houses call it with `-1`.** `Objects.c:366-373` opens
`switch ((*thisHouse)->rooms[room].objects[object].what)` with nothing checked, and the switch ends
at `:695` with no default, so an unmatched `what` returns the uninitialised local `changed`
declared at `:369`. `HandleSwitches` feeds it exactly what `masterObjects` holds
(`Interactions.c:993-999`): `GetRoomLinked` returns `-1` when `data.e.where == -1` or when
`GetRoomNumber` finds no such room (`Objects.c:157-164`), and `GetObjectLinked` returns `-1` when
`data.e.who == 255` (`:203-206`) — while `CreateActiveRects` only gates hot-spot creation on
`where != -1` (`ObjectRects.c:918`), so both failure modes still get a live `kSwitchIt` rect.
`live-state-inventory`'s census finds five switch objects and twenty `kTrigger` objects across the
22 houses that reach it with a `-1`. The C's out-of-bounds reads are not even deterministic: with
`roomType` at 348 bytes (`GliderStructs.h:166-180`), `objects[-1].what` aliases `tiles[6]` and
`objects[-1].data[7]` aliases the low byte of `openings`; with `houseType` putting `rooms[]` at
+866, `rooms[-1].objects[0].what` lands at house byte 578, inside the second high-score name, which
changes when the player sets a record.
*Symptom:* index-out-of-range panic on Land of Illusion / "Window Illusion" and four other rooms
the moment the player flips a switch, or — if you add a `default: return false` and stop there — a
silent behaviour change everywhere, because Go zero-initialises `changed` to `false` while the C
returned whatever was in that stack slot (see trap 4).

**2. `FireTrigger`'s `else` arm dereferences `masterObjects[-1]` by construction.**
`Triggers.c:107` branches on `if (masterObjects[triggerIs].localLink != -1)`, so inside the `else`
at `:174-193` that value *is* `-1`, and `:178` assigns it straight to `triggeredIs`. The grease case
then reads `masterObjects[triggeredIs].dynaNum` and `.hotNum` at `:187-188`. The branch is entered
whenever a trigger's target room is outside the current 3x3 neighbourhood, which the twenty
Leviathan trigger sites in trap 1 do. Note the `SetObjectState(..., local = -1)` at `:184-185` is
itself safe — `if ((changed) && (local != -1))` at `Objects.c:406` covers the master write — and it
*does* flip the house record, so the grease is marked fallen even though nothing local can spill.
*Symptom:* panic in Leviathan's Grand Courtyard. Correct port: perform the `SetObjectState`, skip
the `SpillGrease`.

**3. `triggers[].what` is written from a room-object slot used as a `masterObjects` index, and the
write is the hazard, not the read.** `Triggers.c:50` is
`triggers[where].what = masterObjects[triggers[where].object].theObject.what;` and `.object` was
assigned `masterObjects[whoLinked].objectLink` three lines earlier at `:47` — a room object slot
`0..23`, or `-1` whenever `GetObjectLinked` failed. The field is never read anywhere (`FireTrigger`
recomputes the target type at `:110`), so `live-state-inventory` and `rewards-switches-triggers`
both correctly call it dead — but a literal transcription still performs the indexed load.
*Symptom:* panic on the first arming of a trigger whose target has `who == 255`. Drop the field
entirely; do not "fix" it to read the right object, because nothing would consume the value.

**4. `SetObjectState` can return an uninitialised `changed` by three distinct routes, and Go's
`false` is a behaviour change.** `Objects.c:369` declares `Boolean changed;` with no initialiser.
(a) `case kSlider: break;` at `:475-476` assigns nothing — unreachable in shipped data
(`kSlider` is never a link target) but present. (b) `kKnifeSwitch` (0x45) is **absent from the
outer switch entirely**: the switch-family label list at `:533-540` is `kLightSwitch`,
`kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kInvisSwitch`, `kTrigger`, `kLgTrigger`,
`kSoundTrigger`, and a mechanical diff of all 116 `case k*` labels in `:372-700` against all 117
type `#define`s in `GliderDefines.h:311-435` leaves exactly one type uncovered — this one.
(c) Five *inner* `switch (action)` blocks — blowers `:386`, `kDeluxeTrans` `:497`, lights `:552`,
appliances `:598`, enemies `:645` — have exactly three cases (`kToggle`, `kForceOn`, `kForceOff`)
and no default, and `kOneShot` (3) is stored in `data.e.type` by `ObjectAdd.c:503-506` for
`kTrigger`/`kLgTrigger`. In C the garbage is usually non-zero, so `HandleSwitches` usually takes the
true branch and draws the switch from an equally stale global `newState`.
*Symptom:* a knife switch that visibly toggled in the original never toggles in the port, or vice
versa. Pick `changed = false`, say so in a comment, and note that the C did not.

**5. `grease[].hotNum` is never initialised, and `RedrawAllGrease` reads it before every guard.**
`grease` comes from `NewPtr`, not `NewPtrClear` (`StructuresInit2.c:246`), and `AddGrease` writes
eight fields but not `hotNum` (`Grease.c:227-243`); only `SpillGrease` writes it (`:262`).
`RedrawAllGrease:282` is `src = hotSpots[grease[i].hotNum].bounds;` — above all three of its guards
at `:283-285` — and it runs on nearly every prize pickup (ten of the fourteen reward cases).
*Symptom:* panic on the first prize collected in any room containing an unspilled grease jar.
Initialise `HotNum = -1` and hoist the read below the `mode != kGreaseIdle` guard, which is
behaviour-preserving because that clause implies `SpillGrease` has run.

**6. `SpillGrease` is reachable with `-1` in *either* argument, from two different directions.**
`Grease.c:257-265` reads `grease[who].mode` with no bounds check. `who` is `masterObjects[].dynaNum`,
which is `-1` whenever `AddGrease` refused — `numGrease >= kMaxGrease` (16) at `:212`, or
`BackUpToSavedMap` returned `-1` because `savedMaps` was full at 24 (`:220`). Meanwhile `hotNum` is
`-1` for every *neighbour-room* jar (`Objects.c:283-286` only creates hot spots for the central
room) while `dynaNum` is perfectly valid for one, because `AddGrease` runs for all nine rooms — so a
switch (`Interactions.c:1057-1061`) or trigger (`Triggers.c:117-118`) linked to an adjacent room's
jar sets `mode = kGreaseFalling` with `hotNum == -1`, and four frames later `HandleGrease` **writes**
`hotSpots[-1].action`, `.isOn` and `.bounds` (`Grease.c:60`, `:61`, `:68`).
*Symptom:* the first case panics on a dense room's jar; the second is heap corruption in the
original and a panic four frames after a switch throw in the port. Guard both ends.

**7. `masterObjects[].dynaNum` is `-1` far more often than the dispatchers check for.** The
dispatchers guard `linkIndex != -1` (`Interactions.c:1036`) and `localLink != -1`
(`Triggers.c:107`) and **never** `dynaNum != -1`, then call e.g.
`ToggleToaster(masterObjects[linkIndex].dynaNum)` (`Interactions.c:1090`) whose body is
`dinahs[index].active = !dinahs[index].active;` (`Trip.c:24`). `dynaNum` is `-1` both when
`AddDynamicObject` overflowed the 18 slots and — much more commonly — when the linked object is one
of the types that only register for the central room (`kSparkle`, `kToaster`, `kBalloon`,
`kCopterLf/Rt`, `kDartLf/Rt`, `kBall`, `kDrip`, `kFish`), i.e. whenever the switch's target is in a
neighbour room. `Triggers.c:151-170` passes it to `TriggerBalloon` et al. the same way.
*Symptom:* panic, or a one-byte write before the array in the original. There is no meaningful way
to reproduce it; guard, log, and state the divergence in the spec rather than leaving it silent.
`UNVERIFIED:` whether the House editor can author a cross-room trigger link at all —
`live-state-inventory`'s census says exactly one `kTrigger` in the corpus has a cross-room target
and that target is an empty slot, while `dynamics-movers` could not determine it. The guard costs
nothing either way.

**8. `TriggerSwitch` receives a `dynaNum` and uses it as a `hotSpots` index — and for a
neighbour-room switch that index is nonsense.** `Triggers.c:128` is
`TriggerSwitch(masterObjects[triggeredIs].dynaNum)` and `Trip.c:146-149` is
`void TriggerSwitch (short who) { HandleSwitches(&hotSpots[who]); }`. This is correct, not a type
confusion, because for the six switch types `DrawARoomsObjects` assigns
`dynamicNum = masterObjects[i].hotNum` (`ObjectDrawAll.c:518`, `:531`, `:544`, `:557`, `:570`,
`:574`) before the write-back at `:959` — but `i` there is the **room object slot** from
`for (i = 0; i < kMaxRoomObs; i++)` at `:43`, not a master index. For a central-room switch the two
coincide; for a neighbour-room switch it reads the central room's slot-`i` `hotNum`, which is an
unrelated hot spot or `-1`. `dynamicNum` *is* reset to `-1` per slot at `:45`, so this is genuine
index-space confusion, not carry-over.
*Symptom:* a trigger wired to a switch in the next room flips the wrong object, or panics.

**9. The appliance branch of `SetObjectState` is the one `isOn` write with no `hotNum != -1`
guard.** Compare `Objects.c:415-416` (blowers, guarded), `:469-470` (prizes, guarded) and
`:528-529` (`kDeluxeTrans`, guarded) against `:624-625`:
```c
if ((*thisHouse)->rooms[room].objects[object].what == kShredder)
    hotSpots[masterObjects[local].hotNum].isOn = newState;
```
and note the shredder test is itself nested inside `if (room == thisRoomNumber)` at `:621`.
`hotNum` is `-1` whenever `AddActiveRect` refused, which happens once `nHotSpots >= kMaxHotSpots`
(56, `ObjectRects.c:280-281`).
*Symptom:* panic on switching a shredder in a hot-spot-saturated room; in the C it wrote into the
16 bytes before `hotSpots`. Add the guard, record the deviation.

**10. `HandleSwitches` passes a never-assigned `Rect` to `AddSparkle`, and it fires constantly.**
`Interactions.c:987` declares `Rect newRect, bounds;`. `newRect` is assigned at `:1001-1002`;
`bounds` is assigned nowhere in the function. At `:1050`, inside the bonus-target case, it is
`AddSparkle(&bounds)` — passed to a function that both reads it and mutates it in place
(`DynamicMaps.c:169-193`). `live-state-inventory` counts 145 shipped switch links whose target is
one of those bonus kinds, so this is normal play, not a corner. It is also the *second* sparkle:
`:1049`'s `RestoreFromSavedMap(..., true)` already produced a correctly positioned one, so the
garbage one burns one of only three sparkle slots (`kMaxSparkles = 3`, `GliderDefines.h:251`).
*Symptom:* a sparkle at a random screen position on every switch-triggered prize dissolve, plus a
missing sparkle elsewhere from slot starvation. Not reproducible — the C's value is whatever the
previous callee left on the stack next to `newRect`. Recommendation: omit the second `AddSparkle`
and document the choice; a zero `Rect` in Go puts it at `(playOriginH-10, playOriginV-9)`, which is
*a* choice but not *the* choice.

**11. `ForceThisRoom` advances `thisRoomNumber` even when it refuses to load the room, and has no
negative guard below `-1`.**
```c
if (roomNumber == -1) return;                          // Room.c:373-374
...
if (roomNumber < (*thisHouse)->nRooms)
    *thisRoom = (*thisHouse)->rooms[roomNumber];       // :379-380
else
    YellowAlert(kYellowIllegalRoomNum, 0);             // :381-382
...
previousRoom = thisRoomNumber;                         // :384
thisRoomNumber = roomNumber;                           // :385
```
The two assignments are **outside** the if/else, and `-2 < nRooms` passes, so `ForceThisRoom(-2)`
copies `rooms[-2]`. Note the `== -1` early return happens *before* `previousRoom`/`thisRoomNumber`
are touched — that is what makes the "no room there" case leave room state alone.
*Symptom:* out-of-bounds house reads from everything keyed on `thisRoomNumber` (`DrawLocale:64`,
`GetNeighborRoomNumber:622-623`, `GetUpStairsRightEdge` `ObjectRects.c:1147`) combined with a
`thisRoom` describing the *previous* room. Guard `roomNumber < 0` in Go and record it.

**12. `AddAShreddedGlider` uses `>` where all five siblings use `>=`.** `DynamicMaps.c:732` is
`if (numShredded > kMaxShredded) return;` with `kMaxShredded = 4` and `shreds` allocated with
exactly 4 elements (`StructuresInit2.c:256`). At `numShredded == 4` the test is `4 > 4`, false, so
the C writes `shreds[4]` and leaves `numShredded == 5`, which `RenderShreds` then reads every
frame. Compare `:321` (`numFlames >= kMaxCandles`), `:405`, `:491`, `:575`, `:667`.
*Symptom:* panic, needing four simultaneous shredder deaths. Use `>=`, or use a 5-element backing
array if you want the original's exact pixels; either way note the divergence.

**13. `DidBandHitDynamic` and `FollowTheLeader` both return/dispatch on uninitialised locals, and
the paper over them lives in the caller.** `Dynamics.c:84` declares `Boolean collided;` and `:105`
returns it with the loop skipped when `numBands == 0`; all three call sites are
`if ((numBands > 0) && (DidBandHitDynamic(who)))` (`Dynamics2.c:62`, `:162`, `:267`) — a single
`&&`, not a nested if, and the `&&` is the load-bearing part. `FollowTheLeader`
(`Transit.c:482-497`) tests `if (theGlider.mode == kGliderInLimbo)` then
`else if (theGlider2.mode == kGliderInLimbo)` — **not** `else` — and dispatches `switch (oneOrTwo)`
at `:499` on an uninitialised `Boolean` when neither is in limbo.
*Symptom:* nondeterministic band hits, or a transit dispatched for the wrong glider. Keep the `&&`;
for `FollowTheLeader`, treat "neither in limbo" as a no-op return and say so in the comment,
because it is the only choice that cannot crash.

**14. `GetNumberOfLights(kRoomIsEmpty)` reads `rooms[-1]`, and the result is genuinely unused.**
`Room.c:1038` indexes `thisHousePtr->rooms[where]` with no check, and `DrawLocale` passes
`localNumbers[i]`, which is `-1` for a missing neighbour (`Room.c:617`). The garbage `numLights` is
never consumed: `DrawRoomBackground` bails for `kRoomIsEmpty` (`RoomGraphics.c:179`),
`DrawARoomsObjects` returns at its first statement (`ObjectDrawAll.c:33-34`), and the next
iteration overwrites it.
*Symptom:* none, and that is the point — `internal/render/locale.go:1079-1084` already returns 0
for a nil room. Listed so that nobody "restores fidelity" here.

**15. Defaultless switches whose out-of-range argument is only unreachable by call-site
discipline.** `WipeScreenOn` (`Transitions.c:74-135`) sets `hOffset`, `vOffset` and `count` in four
cases with no `default`, and `MoveRoomToRoom`'s own switch has `default: break;` (`:286-287`) and
then falls into the unconditional tail that calls `WipeScreenOn(where, ...)` at `:300` with that
same value. `GetNeighborRoomNumber` (`Room.c:571-615`) has nine cases and no default, so
`hDelta`/`vDelta` are uninitialised outside `0..8`. `GetObjectRect` is the third.
*Symptom:* `MoveRoomToRoom(g, junk)` is not a harmless no-op — it loops an arbitrary number of
times over uninitialised offsets. The port must not add a fifth direction constant.

---

### 4.B Inverted outcomes

This is the class the Stage 1.4 audit found most of. Each of these reads, from the name or the
shape, as the opposite of what it does.

**16. `GliderHitTop` returning TRUE kills the player, and it is not about landing on top.** Its
single call site is `Interactions.c:1223`, and the true branch is `StartGliderFadingOut` +
`kFadeOutSound` (`:1225-1226`). What the function actually does (`:52-98`): inset the glider's
`dest` by 5 on all four sides, subtract `wasHVel` from `left`/`right` to *rewind* last frame's
horizontal motion, and test overlap. Still overlapping with the horizontal motion removed means the
approach was vertical or engulfing — fatal. Not overlapping means the contact was caused purely by
horizontal motion — survivable, and the false branch bounces `hVel` and charges foil.
*Symptom:* with the polarity flipped, every side scrape kills and every landing on a table is
survivable. Stage 1.4's audit already caught this inverted once.

**17. Suppressing `CheckRoofCollision` suppresses a *kill*, not a clamp.** Every branch of
`Interactions.c:450-504`, including the final unconditional `else`, does
`vVel = kFloorLimit - dest.bottom; StartGliderFadingOut(...); PlayPrioritySound(kFadeOutSound, ...)`.
`sliding` gates it at `:455` (`!thisGlider->sliding`).
*Symptom:* get the polarity backwards and grease on a `kRoof` room becomes lethal instead of a
survivability *gain*.

**18. `stillOver` suppresses; it does not trigger. `FlagStillOvers` sets it TRUE.**
`FlagStillOvers` (`Interactions.c:1715-1732`) latches every rect the glider currently overlaps;
`HandleSwitches` (`:990`), `ArmTrigger` (`Triggers.c:38`) and `HandleMicrowaveAction` (`:1164`) all
early-return when it is set. Its purpose is to stop a glider that *materialises* inside a switch
from firing it.
*Symptom:* every duct arrival instantly re-ducts. Note
`internal/game/player/env.go:103-105` currently documents the exact
opposite ("so a glider dropped onto a switch triggers it") and must be corrected as part of this
stage. (Fullest treatment §2.12, which carries the `env.go:103-105` doc-comment correction; also §6.6 and §6.7.)

**19. `playerDead` names *which* player died; it is not "is a player dead".** `Player.c:53`
declares `Boolean shadowVisible, onePlayerLeft, playerDead;`, `kPlayer1 == TRUE` and
`kPlayer2 == FALSE` (`GliderDefines.h:556-557`), and the single write is
`playerDead = thisGlider->which;` at `Player.c:1508`. It is compared against a glider's `which`
(`Modes.c:474`) and against the player constants (`Player.c:349`, `RubberBands.c:163`, `:186`,
`Dynamics.c:337`, `:538`, `Dynamics2.c:364`, `:439`, `:513`). `onePlayerLeft` is the flag that says
someone is dead, and `playerDead` is only meaningful while it is true.
*Symptom:* every two-player collision, band push and follow-the-leader dispatch picks the wrong
glider. `env.go`'s `PlayerDead() bool` has the right shape and a name that invites the wrong use.

**20. `GetUpStairsRightEdge` searches for `kDownStairs`, and `GetDownStairsLeftEdge` for
`kUpStairs`.** `ObjectRects.c:1147` and `:1177`. The names describe the *trip*, not the object: you
arrive in the destination room at its opposite staircase. Both also read
`(*thisHouse)->rooms[thisRoomNumber]` — the house, not `thisRoom` — so both are wrong if
`ForceThisRoom` took trap 11's illegal path. Fallbacks are `kRoomWide` (512) and 0.
*Symptom:* both gliders walk out of the wrong side of the destination room.

**21. `IsShadowVisible` and `DoesRoomHaveFloor` are identical except for `kRoof`.**
`Room.c:1103-1134` vs `:1138-1168`. Both test bit `0x0008` for user backgrounds; the built-in lists
differ by exactly one entry — `kRoof` is in `IsShadowVisible`'s no-floor list and absent from
`DoesRoomHaveFloor`'s. On a roof: no shadow, but `bottomOpen == false`.
*Symptom:* implement one and call it from both places and every roof room either grows a shadow or
becomes a fall-through.

**22. `IsShadowVisible()` is both a pure predicate and a cached global, and the port currently
conflates them.** The recompute is `Room.c:1103`; the cache is `shadowVisible` at `Player.c:53`,
written at four sites — `DrawLocale` `RoomGraphics.c:126`, `FlagGliderNormal` `Modes.c:361`,
`MoveGliderShredding` `Player.c:1286`, and `RedrawRoomLighting` `RoomGraphics.c:459`. That fourth
one matters: a light switch re-derives `shadowVisible` mid-play, so it is not only a room-change or
mode-change event. `NopEnv` in `env.go:194-195` returns whatever `SetShadowVisible` stored.
*Symptom:* a real implementation that recomputes and a test that stores disagree. Split the method
or document which one it is.

**23. `kStar` is missing its `StopStar` on the switch path.** `HandleSwitches` groups `kStar` with
the ordinary prizes at `Interactions.c:1040-1051` and calls only `RestoreFromSavedMap`, while
`kCuckoo` at `:1053-1055` does get its `StopPendulum`. `HandleRewards` *does* call `StopStar`
(`:941`).
*Symptom:* a switch that dissolves a star erases it for one frame, then `RenderStars` blits the
next cell of a strip that still contains the star — so the star reappears and twinkles forever,
uncollectable, because `SetObjectState` already cleared `data.c.state`. After a subsequent light
switch it also freezes the *old* lighting inside its 32x31 rect, because `DrawARoomsObjects` now
skips the invalid star and `ReBackUpStar` never refreshes the strip. Reproduce it; it is the most
visible original bug in the animation subsystem.

---

### 4.C Statement order and control-flow shape

Nothing here changes what the code does when read line by line; everything here changes what it
does when tidied. Preserve the C's exact nesting — never fuse a nested `if` into `&&`, never turn
an else-less inner guard into `else if`.

**24. `RenderFrame` is not idempotent, and it is called twice on any frame in which a transition
completes.** Call sites: `Play.c:469` and `:494` (the two-player and one-player arms of the frame
loop) plus `Transit.c:303`, `:341`, `:380`, `:419`. `COMPILEQT` **is** defined
(`GliderDefines.h:15`), so all four Transit sites are live; each sits *before*, not inside, the
`if ((thisMac.hasQT) && (hasMovie) && (tvInRoom) && (tvOn))` that follows it, i.e. unconditional;
and in `TransportRoomToRoom`/`MoveDuctToDuct`/`MoveMailToMail` it is also **outside** the
`if (!sameRoom)` guards (`Transit.c:334-338` vs `:341`), so it fires even on a same-room transport.
Because `evenFrame` flips only at `Play.c:435`, the second call takes the same branch of
`Render.c:649-652` as the first. Confirmed double-steps: `clockFrame++` (`Render.c:270`) advances
twice, so pendulum phase and the `kTikSound`/`kTokSound` schedule drift on every room change; the
flame/tiki/coal strips advance two cells and the stars none, or the reverse, and all three loops
mutate `mode` *and* `src.top`/`src.bottom` (`Render.c:200-256`), so this is state, not a redraw;
`HandleGrease` runs twice; and the `while (TickCount() < nextFrame)` limiter at `Render.c:662-664`
waits twice, so a transition frame costs two frame budgets. On the `MoveRoomToRoom` path
`ZeroFlamesAndTheLike` (`numShredded = 0`, `DynamicMaps.c:795`) and `InitGarbageRects`
(`numSparkles = numFlyingPts = 0`, `Render.c:682-688`) run first, so shreds, sparkles and flying
points are spared — but on a **same-room** transport/duct/mail trip `ReadyLevel` is skipped
entirely while the `RenderFrame` at `:341` still runs, so there they double-step too. None of the
four Transit calls is followed by `HandleDynamicScoreboard`, so `displayedScore` does not roll on
the transition render.
*Symptom:* if you call the render once, every room change shifts pendulum phase and candle
animation by one cell relative to the original; if you call it twice without knowing why, you will
hunt the "double" `HandleGrease` as a bug. Decide explicitly, in writing, whether the Go port
reproduces it. Note that `background-animations.md` asserts the opposite ("neither ever updates
twice in a row"); the C above refutes it and that sentence must not survive into the spec. (Fullest treatment §1.5; the port decision is §6.4 item 1; the claim it refutes is trap 158.)

**25. `HandleGrease` mutates hot spots from inside `RenderFrame`, after `HandleInteraction` has
already run.** `Render.c:647` → `Grease.c:60-68` changes `action` to `kSlideIt`, forces
`isOn = true`, and **replaces** `bounds` with a 2x2 room-local rect; `:95`/`:102` then widen it by
2 px per frame.
*Symptom:* a port that treats `hotSpots[].action` and `.bounds` as immutable after construction
cannot express grease at all; and because this happens after `HandleGlider`, the player is first
caught on the frame *after* the slick appears.

**26. `HandleGrease`'s promotion block is not an `else`, and the fall is four frames, not three.**
`frame` starts at `-1` (`Grease.c:232`); `HandleGrease` does `frame++` (`:55`) then
`if (frame >= 3)` (`:56`), and the `CopyBits`/`AddRectToWorkRects` at `:71-80` and the ±2 `dest`
advance at `:83-86` sit **outside** that inner block and run unconditionally. Rows 0/1/2/3 are drawn
at `dest + 0/+2/+4/+6`. The left-spreading `stop` test and the promotion to `kGreaseSpiltIdle` are
at `:126-130`; the function closes at `:133`.
*Symptom:* fuse them into if/else and you drop the last frame of the animation and leave `dest` at
+6 instead of +8. `docs/analysis/interactions.md` §19.1 calls it a 3-frame animation.

**27. `CheckForHotSpots`' second `SectGlider` is a new `if`, not an `else`.**
`Interactions.c:1627-1687`: in the two-player arm, `hitObject = false` then
`if (SectGlider(&theGlider, ...))` then a **separate** `if (SectGlider(&theGlider2, ...))`, then
`if (!hitObject) hotSpots[i].stillOver = false;` at `:1674-1675`. The one-player arm is
`if (SectGlider(...)) Handle...; else stillOver = false;`.
*Symptom:* both gliders legitimately fire the same hot spot in one frame — a prize can be collected
twice, a fan pushes twice. Also, because `hitObject` is only set inside the arms that actually
called the handler, one player standing on a switch prevents the other from ever re-triggering it,
and in `onePlayerLeft` mode the dead player's stale `dest` satisfies `SectGlider` but fails the
`playerDead` guard, so the latch clears while a glider rect is still sitting on the rect.

**28. `CheckForHotSpots` clears `stillOver` only for `isOn` spots; `FlagStillOvers` clears it for
all.** `Interactions.c:1633` gates the whole body on `hotSpots[i].isOn`, whereas `FlagStillOvers`
additionally sets `stillOver = false` for `isOn == false` spots (`:1729-1730`).
*Symptom:* call one where the C calls the other and a different set of switches becomes re-firable.

**29. Two `stillOver` latches sit deliberately outside their "something happened" guards.**
`HandleSwitches`' `who->stillOver = true;` at `Interactions.c:1154` is outside the
`if (SetObjectState(...))` block that closes at `:1152`, so a switch whose target refuses to change
still consumes the entry. `ArmTrigger`'s `who->stillOver = true;` at `Triggers.c:54` is outside
`if (where != -1)` (`:43-52`), so when all 16 trigger slots are armed the trigger is silently
swallowed **and** debounced.
*Symptom:* guard either one with its function's success and a player standing on a switch
re-triggers it every frame. `UNVERIFIED:` `live-state-inventory` reports one shipped room (The
Asylum Pro, "Rockin' In The Free World?") with exactly 16 trigger objects, which would make the
`ArmTrigger` case reachable; I did not re-census.

**30. `HandleMicrowaveAction` reads `stillOver` and never sets it.** `Interactions.c:1164` is the
only reference, and nothing else latches a `kMicrowaveIt` rect except `FlagStillOvers`.
*Symptom:* a microwave re-zaps and replays `kMicrowavedSound` on every frame the glider is fully
inside its column. That is the original behaviour; reproduce it.

**31. `FlagStillOvers` has exactly one caller, it takes one glider, and it is not "room entry".**
`Player.c:954`, in the `else` branch of `FinishGliderDuctingIn` — the frame a ceiling-duct arrival
finishes emerging. No other finisher calls it: `FinishGliderMailingLeft` (`:883`) and `...Right`
(`:921`) set `enteredRect` and stop, and stairs, transporters, floor ducts
(`kLinkedToFloorDuct` is an empty case, `Transit.c:138-139`) and respawns get no pre-latch. On a
cross-room transit `AddActiveRect` already zeroed every `stillOver` (`ObjectRects.c:287`) and this
re-latches; on a **same-room** duct (`Transit.c:359`) the table was not rebuilt and this is the only
thing preventing an instant re-duct. In two-player only the arriving glider's position seeds the
latch.
*Symptom:* a glider dropped out of a ceiling duct onto a switch does not fire it, while a glider
mailed onto the same switch does. Do not "harmonise" the finishers.

**32. `kDuctItUp` latches `stillOver` only in the one-player arm.** `Interactions.c:1576`, inside
the `else`; neither two-player arm (`:1565`, `:1573`) latches, and `stillOver` is one of the five
gate terms at `:1554`.
*Symptom:* two-player ceiling ducts behave differently from one-player ones — a glider hovering in
the hot spot can re-trigger.

**33. `HandleTriggers` and `HandleBands` are outside the `!gameOver` guard, and `HandleDynamics` is
too.** `Play.c:456-457` (two-player) and `:484-485` (one-player) sit after the
`if (!gameOver) { GetInput; HandleInteraction; }` block closes.
*Symptom:* after the last star, armed triggers keep firing and bands keep flying and keep testing
against a stale glider rect for the whole `countDown`. This is exactly the guard a porter tidies
into the block above it.

**34. `HandleBands` re-enters the hot-spot pipeline after the sweep, in the same frame.**
`CheckBandCollision` calls `HandleSwitches` (`RubberBands.c:133`) and `ArmTrigger` (`:137`), both of
which set `stillOver = true`.
*Symptom:* a band hitting a switch the glider is also standing on locks the glider out next frame,
and a band can arm a trigger the glider's own contact was debounced out of. Note also that a
band-armed trigger costs one extra frame relative to a glider-armed one, because `HandleBands`
(`:485`) runs after `HandleTriggers` (`:484`).

**35. `CheckBandCollision` clamps at the wall *before* it kills out-of-bounds bands, and the two
tests use different expressions.** Phase 1 (`RubberBands.c:44-61`) clamps `dest.left` to exactly
`kLeftWallLimit`, gated on `(leftThresh == kLeftWallLimit)` at `:44`; phase 5 (`:195-203`) kills on
a bare `dest.left < kLeftWallLimit`, and is an if / else-if pair. `12 < 12` is false, so the clamped
band lives — but only in a room whose left threshold *is* the wall.
*Symptom:* run phase 5 first, or clamp to `kLeftWallLimit - 1`, and every band that touches a wall
dies. "Simplify" both to the same expression and bands stop bouncing in rooms with side openings.

**36. `HandleBands`' compaction loop is a `while` inside a `do`, and terminates only because of a
store that looks dead.**
```c
count = 0;
do {
    while (bands[count].mode == kKillBandMode) { bands[count].mode = 0; KillBand(count); }
    count++;
} while (count < numBands);            // RubberBands.c:241-251
```
`KillBand` is swap-with-last (`bands[which] = bands[lastBand]`, `:300`), so the `mode = 0` write
must happen *before* the kill or it clobbers the swapped-in band's mode; and when
`which == lastBand` there is no copy, so that same write is the only thing that breaks the loop.
The `while` is also what lets two consecutive dying bands both be reaped. The `do` evaluates
`bands[0]` once even after `numBands` hits 0 — in bounds because the array has 2 slots, and safe
because `mode` was just zeroed.
*Symptom:* omit the store and you get an infinite loop; move it after `KillBand` and a live band is
silently killed.

**37. `AddBand` reads `bands[numBands]` before the increment.** `RubberBands.c:285` then `:286`.
The recoil is `glider.hVel -= band.hVel / 2` — sign-opposite to the shot, magnitude 10.
*Symptom:* write `numBands++` first and index `numBands` after and you compute the recoil from a
garbage slot.

**38. Two else-less inner guards in the dynamics handlers, both of which re-arm a counter for an
*inactive* object.**
```c
if (dinahs[who].active)          // Dynamics.c:369, HandleToast
    dinahs[who].frame--;
if (dinahs[who].frame <= 0)      // :371 — NOT `else if`, NOT `&&`
{ ... dinahs[who].frame = ... }  // :381
```
```c
if (dinahs[who].active)          // Dynamics.c:584, HandleOutlet
    dinahs[who].timer--;
if (dinahs[who].timer <= 0)      // :587
{ ... }                          // :596
```
*Symptom:* fuse them and an inactive toaster or outlet drifts its counter negative and fires the
instant it is switched on, instead of waiting a full delay.

**39. `CheckDynamicCollision`'s foil gate is genuinely a single `&&`, while the visually similar
`kDissolveIt` handler nests.** `if ((evenFrame) && (foilTotal > 0))` at `Dynamics.c:60` versus
`Interactions.c:1221-1235`, which is `if ((foilTotal > 0) || (mode == kGliderLosingFoil)) { if
(GliderHitTop(...)) {...} else { if (foilTotal > 0) {...} } }`.
*Symptom:* merge them into one shared helper and either the dynamic channel starts double-charging
foil (which only the hot-spot channel does — see trap 40) or the hot-spot channel starts skipping
odd frames.

**40. `kDissolveIt` charges foil twice per frame, and the inner decrement is unconditional.**
`GliderHitTop` decrements at `Interactions.c:82` on its `!hitTop` path, and the caller decrements
again at `:1230-1235`. One frame of survivable side contact therefore costs two sheets, halving the
effective `kFoilSupply` of 8 to four side impacts. Worse, `:82` is unconditional while the caller's
outer gate is `(foilTotal > 0) || (mode == kGliderLosingFoil)` (`:1221`), so a glider already at 0
and in `kGliderLosingFoil` walks in and leaves `foilTotal` at `-1`.
*Symptom:* do **not** clamp `foilTotal` at 0 — later reads compare `<= 0` and `> 0`, and clamping
changes when `StartGliderFoilLosing` fires.

**41. Statement order inside the handlers is mutate-then-test and test-then-move, consistently.**
`HandleToast` calls `CheckDynamicCollision` at `Dynamics.c:349` **before** `VOffsetRect` at `:350`,
so the collision uses last frame's position, and `whole` is computed after the move and before the
`vVel++` (`:351-356`), so the swept rect spans old→new using the pre-increment velocity.
`HandleOutlet` does `timer--` (`:532`) → collision (`:534-550`) → `timer <= 0` reset (`:552`), and
clears `position` at `:555` then reads it at `:567` in the same pass. All the `timer`-driven
appliances decrement before testing, so `timer = 4` yields four observable phases (3, 2, 1, 0), not
five.
*Symptom:* reorder `:567` above `:552` and a dark room stops erasing its outlet. Test bounds before
moving and see trap 42.

**42. All three of balloon, copter and dart spawn exactly *on* their own recycle boundary.**
Balloon: `dest.bottom = kBalloonStart` (310) at `Dynamics3.c:395` and `Dynamics2.c:101`, tested
`dest.bottom >= 310` at `:90`. Copter: `dest.top = kCopterStart` (8) at `Dynamics3.c:417` and
`Dynamics2.c:207`, tested `<= 8` at `:191`. `kDartLf`: `dest.right = kRoomWide` (512) at
`Dynamics3.c:444`, tested `>= 512` at `:297`. `kDartRt`: `dest.left = 0` at `:450`, tested `<= 0` at
`:296`. The only thing that saves them is that the boundary test is placed **after** the move, and
the move happens on the first `moving` frame.
*Symptom:* test bounds before moving, or test on the frame `moving` is set, and every balloon,
copter and dart despawns instantly and never appears.

**43. The ball's collision test is unconditional; every other mover's is inside `if (moving)`.**
`Dynamics2.c:360-376` sits entirely outside the `if (moving)` at `:378`.
*Symptom:* hoist it inside to "tidy up" and a switched-off ball, a never-launched ball and a ball
that has damped to a stop all stop being lethal, when in the original they are lethal.

**44. A shot enemy is harmless, and the proof is an absence.** The popped-balloon arm
(`Dynamics2.c:75-87`), the shot-copter arm (`:181-189`) and the falling-dart arm (`:289-294`) call
neither `CheckDynamicCollision` nor `DidBandHitDynamic`. There is no flag; the behaviour is purely
which statements are in which branch. Relatedly, a band-hit enemy does not move that frame — in all
three sites the hit is the `if` arm of an if/else whose `else` holds the `HOffsetRect`/`VOffsetRect`
and the `whole` recomputation — and `DidBandHitDynamic` neither consumes the band nor zeroes its
`hVel`, so one band can pop several enemies as it flies through.
*Symptom:* lift the collision call to the top of the handler (as the ball legitimately does) and
crumpled darts and popped balloons become lethal.

**45. `CheckGliderInRoom` runs the vertical and horizontal boundary tests as two independent
`if`-chains, so one glider can chain two transitions in one frame.** `Interactions.c:696` /
`else if :709` / `else if :722`, then a **new statement** at `:725`
(`if (thisGlider->dest.left < leftThresh)`) with `else if (:738)`. (`:724` is blank —
`room-transitions.md` cites `:724` for the new `if` and `:738` correctly two lines later; the anchor
is `:725`.)
*Symptom:* fuse them into one chain with `else` and a glider in a corner can no longer leave
diagonally, which changes which room it lands in.

**46. The four two-player departure handlers have no "don't exit" third branch, and that is how
both gliders end up in limbo.** Compare `CheckEscapeUpTwo` (`Interactions.c:179-196`), which is
three-way with a `kDontExitSound` else, against `MoveGliderUpStairs` (`Player.c:351-363`):
```c
if (otherPlayerEscaped == kPlayerEscapedUpStairs) { reset; MoveRoomToRoom; }
else { otherPlayerEscaped = kPlayerEscapedUpStairs; RefreshScoreboard; FlagGliderInLimbo; }
```
There is no third arm, so a *pending non-matching* escape code is silently overwritten and the
second glider is limboed too. `MoveGliderDownStairs`, `TransportGliderOut` (`:618-631`) and
`MoveGliderDownDuct`/`MoveGliderUpDuct` (`:727-740`) have the same two-way shape.
*Symptom:* "harmonise" these into the three-way shape and the deadlock that the original produces —
both gliders in limbo, no transition — becomes impossible, which changes the recovery path in
trap 47.

**47. `takingTheStairs` is a one-shot cleared only by the function that runs *after* its readers,
and one path never clears it at all.** Set at `Player.c:344` and `:472` — **before** the
`if (twoPlayerGame)` fan-out at `:345` — read at `Transit.c:225` and `:257`, cleared at
`RoomGraphics.c:127` inside the `DrawLocale` that `ReadyLevel` (`:416`) invokes after those reads.
If the glider becomes the leader instead of transitioning (`Player.c:359-362`), no
`MoveRoomToRoom` runs, so no `DrawLocale` runs, so the flag stays true indefinitely.
*Symptom:* clear it at the top of `MoveRoomToRoom` and both stair transitions break. Leave the
original's leak and the documented failure reproduces: player A limbos on the stairs, the player
hits Delete, `ForceKillGlider` → `OffAMortal` → `FollowTheLeader` → `MoveRoomToRoom(&theGlider2,
kAbove)`, where `if (!takingTheStairs)` at `Transit.c:225` is false and both gliders walk out of the
destination's down-staircase. `docs/analysis/progression.md` §9.5 asserts the opposite.

**48. `HandleRoomVisitation` is addressed by `localNumbers[kCentralRoom]`, not `thisRoomNumber`, and
is called first.** `Transit.c:440`. Equivalent only because `localNumbers` still holds the *previous*
load's central room at that point — which is the room the original wanted to mark.
*Symptom:* modernise the addressing **and** move the call and you mark the wrong room. Doing either
one alone is invisible. Keep the call first and keep the addressing.

**49. `DrawRoomBackground` publishes `thisBackground`/`thisTiles` before both of its early
returns.** `RoomGraphics.c:169-177` runs, then `:179-191` (the `numLights == 0` blackout) and
`:193-207` (the `kRoomIsEmpty` + `wardBitSet` blackout) can return.
*Symptom:* move the publish below the returns and every unlit dirt room becomes solid, because
Stage 1.4's `CheckEscapeUp`/`CheckEscapeDown` read `thisTiles` to find the dirt openings.
`internal/render/locale.go:359` already has this function; Stage 1.5 must confirm the ordering
survived that port rather than assume it.

**50. `ZeroFlamesAndTheLike` must run before `ListAllLocalObjects`.** In `DrawLocale` that is
`RoomGraphics.c:51` before `:72`. It zeroes `numChimes` (`DynamicMaps.c:796`) and the listing
re-counts it (`ObjectRects.c:1052`, inside `CreateActiveRects`).
*Symptom:* reverse them and every wind chime in the house goes silent. Note the counter's exact
lines, because two reports disagree: `numGrease = 0` is `DynamicMaps.c:793` and `numChimes = 0` is
`:796`, in the function at `:787-797`.

**51. `DrawCuckoo` runs *before* `AddPendulum` but `AddStar` runs *before* `DrawSimplePrizes`.**
`ObjectDrawAll.c:339` vs `:343`, and `:440` vs `:442`. The pendulum swings entirely inside the
clock's 40x80 rect (`StructuresInit2.c:362`, at `(left+4, top+46)` size 32x28), so its strip **must**
contain the clock body or the clock is erased two to four times a second. The star's strip must
**not** contain the star.
*Symptom:* two adjacent cases with opposite orders, both mandatory. Get either backwards and you
either erase the cuckoo clock or leave a permanent ghost star.

**52. Star and cuckoo each occupy two `savedMaps` slots under the *same* `(where, who)` key, and the
claim order decides which one lookups find.** The object-rect slot is claimed first
(`ObjectDrawAll.c:434` for stars, `:336` for cuckoos), the strip slot second (`:440`, `:343`).
`RestoreFromSavedMap` and `ReBackUpSavedMap` both return the **first** match, which is therefore
always the object-rect slot.
*Symptom:* reverse the two registrations and collecting a star blits a 32x186 strip into the room.
Note `RestoreFromSavedMap`'s `break` is inside its `if` (`DynamicMaps.c:160`), so a slot with a nil
`map` is skipped and the loop *continues* — which can fall through to the strip slot with its
garbage `dest`. Reachable only if `CreateOffScreenGWorld` failed, whose error code at `:82` is
captured and never checked.

**53. Flames are registered in unlit rooms; only the candle *body* is gated on `isLit`.**
`ObjectDrawAll.c:76-83` has `if (isLit) DrawSimpleBlowers(...)` as a *sibling statement* of the
`AddCandleFlame` call, both inside the `SectRect` guard. Same for tiki and BBQ.
*Symptom:* fuse the two ifs and unlit rooms lose their disembodied floating flames, which is
original behaviour — `DrawRoomBackground` paints an unlit room solid black
(`RoomGraphics.c:179-191`), so the strip's baked background is black.

**54. `AddPendulum` writes `clockFrame = 10` before it claims its slot.** `DynamicMaps.c:578` vs
`:580`.
*Symptom:* a cuckoo that fails to get a slot still resets the global clock for every clock in the
room, even though it will never swing.

**55. `RenderPendulums`' phase test has a nested reset, not a modulo.**
```c
clockFrame++;
if ((clockFrame == 10) || (clockFrame == 15))
{
    if (clockFrame >= 15) clockFrame = 0;
    ...
}
```
`Render.c:270` onward, with `AddPendulum` seeding 10 (`DynamicMaps.c:578`), so the first increment
gives 11 and matches neither.
*Symptom:* write `clockFrame %= 15` or `>= 10` and the swing rhythm changes from 5, 10, 5, 10 to
uniform. Sounds land on every second swing, exactly 15 frames apart; the pendulum dwells 10 frames
at each extreme and 5 at centre; the first swing is 5 frames after the seed, not next frame.

**56. `RenderShreds` runs after `RenderSparkles`.** `Render.c:659` vs `:655`.
*Symptom:* the sparkle a shred spawns on its frame 20 is not drawn until the next frame.

**57. The two-player doubling in `HandleRewards` is a separate statement, not an `else`.**
`Interactions.c:842-843`, `:864-865`, `:883-884`, `:911-912`, `:970-971`.
*Symptom:* fuse it into the sign test's `else` and every two-player pickup from an empty counter is
halved.

**58. `who->isOn = false` is outside the `SetObjectState` guard in every reward case, and inside a
narrower guard on the band path.** `Interactions.c:779`, `:795`, `:811`, `:828`, `:847`, `:869`,
`:888`, `:897`, `:916`, `:930`, `:950`, `:975` versus `RubberBands.c:128`.
*Symptom:* a prize whose `SetObjectState` returns false still has its hot spot turned off, so it
becomes uncollectable for the rest of the room's life while still being drawn. Do not unify the two
paths.

**59. `ReadyGliderFromTransit` calls `FlagGliderNormal` before it measures.** `Transit.c:72`, ahead
of `tempRect = thisGlider->dest` in every arm, so the mailbox arms' `RectTall(&tempRect)` is always
the normalised 20. `FlagGliderNormal` (`Modes.c:334-362`) normalises `dest` and `destShadow` from
their **top-left corners** — preserving `destShadow.top`, which is why the shadow stays on the floor
plane through a transit — and ends by recomputing `shadowVisible = IsShadowVisible()`, i.e. from
the *destination* room, before the glider is positioned.
*Symptom:* measure before normalising and every mailbox arrival is offset by the difference between
the pre- and post-transit glider height.

**60. `OffsetGlider`'s vertical cases deliberately do not touch the shadow, and there is no shared
epilogue.** `Player.c:1443-1479`: `kToRight`/`kToLeft` offset `destShadow.left`/`.right` and assign
`wholeShadow = destShadow`; `kAbove`/`kBelow` touch neither.
*Symptom:* factor the four cases into "offset by (dh, dv) then assign both wholes" and the shadow
leaves the floor plane on every vertical transition.

---

### 4.D Index spaces, overloaded fields and aliased unions

**61. `masterObjects[i]` equals central-room slot `i` only by construction, and three separate
pieces of code depend on it.** `CreateActiveRects(short who)` reads `masterObjects[who].theObject`
(`ObjectRects.c:304`); `hotSpots[].who` stores that same index (`:284`), later dereferenced as
`masterObjects[who->who].roomLink` (`Modes.c:164`); and `ObjectDrawAll.c:518` uses the room slot as
a master index. It works only because (a) `ListAllLocalObjects` lists `kCentralRoom` **first**
(`Objects.c:312`) and (b) `ListOneRoomsObjects` appends an entry for all 24 slots including empty
ones (`:266-295`, no `kObjectIsEmpty` skip, `numMasterObjects` incremented at `:289`).
`CreateActiveRects` is only *called* for the central room (`:283`), which is what keeps it honest.
*Symptom:* compact `masterObjects` to skip empty slots and every switch link, every transporter
destination and every trigger-to-switch chain points at the wrong object. Storing an object pointer
instead of the slot number loses nothing; storing a slot number and then indexing a
differently-ordered list breaks everything.

**62. `masterObjects[].hotNum` is the **last** rect an object created, not all of them.**
`CreateActiveRects` returns `hotSpotNumber` (`ObjectRects.c:1062`), overwritten by each
`AddActiveRect`. The multi-rect types: `kLeftFan`/`kRightFan` (blade `kDissolveIt` then wind
`kPushIt*`, `:374`/`:380` and `:389`/`:395`), the **five** flame types `kTaper`/`kCandle`/`kStubby`/
`kTiki`/`kBBQ` (`kLiftIt` and `kBurnIt` when
`(bounds.bottom - bounds.top) > kDeadlyFlameHeight` (24), else `kBurnIt` alone, then always a
`kDissolveIt` body rect — `:407-421`, `:435-446`, `:460-471`, `:485-496`, `:508-519`), and
`kMicrowave` (`kDissolveIt` then `kMicrowaveIt`, `:975`/`:978`).
*Symptom:* toggling a fan toggles only its wind column and leaves the blade permanently lethal;
toggling a microwave toggles only the beam and leaves the body lethal; a candle's `hotNum` is its
body, not its flame. A port that keeps a per-object list of hot-spot indices and toggles them all
makes fan blades and microwave bodies stop killing. (Note: `kInvisBlower` and `kLiftArea` are
**not** multi-rect — see trap 96.)

**63. `masterObjects[].dynaNum` is three different index spaces, all written by the same
statement.** `dinahs[]` for dynamic objects (from `AddDynamicObject`'s return,
`ObjectDrawAll.c:652-888`); `grease[]` for `kGreaseRt`/`kGreaseLf` (from `AddGrease`'s return, which
is `numGrease - 1`, `:376`/`:397` via `Grease.c:247`); `hotSpots[]` for the six switch types
(`:518-574`); `-1` for everything else. All of them are copied into `masterObjects[n].dynaNum` by
the single statement at `:959`. `GliderStructs.h:330` says "index into dinahs (if any)", which is
wrong for two of the three.
*Symptom:* give `dynaNum` one Go type and you write the wrong dereference in `FireTrigger`'s
`kLightSwitch..kInvisSwitch` case (`Triggers.c:122-129`). If you split it into two fields, both must
be written from the same source line so the ordering cannot drift.
`docs/analysis/enemies.md:1097-1098` documents the chain as always ending in `dinahs[dynaNum]`,
which is wrong for switches.

**64. `dinahs[].hVel` is a horizontal velocity for exactly two of the sixteen types.** Toaster =
the clip line (`Dynamics3.c:223`, with the C's own comment "hVel used as clip"); outlet =
`numLights` (`:314`); drip = the remembered rest `top` (`:502`); fish = the idle-delay reload
(`:521`). Only copters and darts use it as velocity. `position` is likewise a coordinate for five
enemy types and a boolean for the outlet (`:429`, `:458`, `:489`, `:538`); `count` is an initial
velocity or a reload delay; `bands[].mode` is an animation frame 0..2 or the `-1` kill flag
(`RubberBands.c:14`); `leftThresh`/`rightThresh` are both a coordinate and a wall flag (compared
against `kLeftWallLimit`/`kRightWallLimit` at `Interactions.c:513`, `:603`);
`thisGlider->wasMode` is a saved mode, a burn fuse **and** the spider-web timer; `newState` is a
global out-parameter (`Objects.c:78`).
*Symptom:* name the Go field `HVel` and write generic movement code over it and four of the six
affected types corrupt. This is systemic — do not give these fields meaningful names.

**65. `frame` and `timer` swap roles between the toaster's two phases, and the VCR stores a boolean
in `frame`.** While `moving`, the toaster's `frame` is the 0..5 bread sprite index and `timer` is a
constant; on landing, `frame = timer` (`Dynamics.c:363`) turns `frame` into the reload countdown —
and `timer = data.g.delay * 3` with `delay` a `Byte`, so up to 765. `timer` is never decremented for
a toaster anywhere in the program. The VCR does
`dinahs[who].frame = 1 - dinahs[who].frame;` (`:630`) read as `if (frame == 0)` (`:634`).
`TriggerToast`'s else does `frame = timer` for an inactive toaster (`Trip.c:165`) and `TriggerFish`
repurposes `whole` as a launch anchor (`:200`).
*Symptom:* only `RenderToast`'s `if (moving)` gate (`Dynamics.c:117`) keeps 765 out of
`breadSrc[6]`; drop it for symmetry with `RenderBall`/`RenderDrip` and you index out of bounds
immediately. A `frame` field typed as a sprite index and range-checked against a sheet length
rejects the VCR's value.

**66. `who` means different things in the five animation structs, and the Go `Anim` already swaps
two of them.** `flameType.who`, `pendulumType.who` and `starType.who` are `savedMaps` **slot**
indices; `pendulumType.link` and `starType.link` are **object** indices; `greaseType.who` is the
object index while `greaseType.mapNum` is the slot. `StopPendulum`/`StopStar` match on
`(link, where)`. `internal/render`'s `Anim.SavedMap` holds the C's `who` and `Anim.Who` holds the
C's `link`.
*Symptom:* one transposition and `StopStar` silently matches nothing, which is exactly the shape of
trap 23's symptom arriving for the wrong reason.

**67. `grease[].where` is a real room number, not a 0..8 local index.** Assigned
`localNumbers[neighbor]` (`ObjectDrawAll.c:376`), compared against `thisRoomNumber`
(`Grease.c:283`) and matched in `ReBackUpGrease` (`:187`). `grease[].who` is the object slot within
that room.
*Symptom:* treat it as a local index and every neighbour-room jar's redraw and re-backup silently
no-ops.

**68. `SetObjectState`'s prize branch writes the wrong union member, at a byte offset that
matters.** `Objects.c:465` is `masterObjects[local].theObject.data.a.state = false;` inside the
prize case. `blowerType.state` is at data offset **7** (`GliderStructs.h:11-19`);
`bonusType.state` is at offset **8** (`:27-34`). On big-endian 68k, offset 7 is the low byte of the
`short points`. So the master copy's `data.c.state` is never cleared and a `kInvisBonus`'s `points`
is corrupted — 0x01F4 (500) becomes 0x0100 (256). Unobservable in normal play only because
`HandleRewards` reads `points` at `Interactions.c:920` *before* calling `SetObjectState` at `:921`,
and because `isOn` is cleared through a separate path. The same branch also ignores its `action`
argument entirely (so the `0` passed at `RubberBands.c:125` and `Interactions.c:893` is dead) and
returns `changed = (rooms[room].objects[object].data.c.state == true)` with `newState = false` —
which is precisely what stops a second band re-spilling a jar.
*Symptom:* `internal/house/object.go` stores `Data` as `[10]byte`, so writing `Data[7] = 0`
reproduces the C exactly. Writing `Bonus().State` is the tidy choice and a divergence; pick one and
record it. Note also that the `hotSpots[...].isOn = false` at `:468` is nested inside
`if (room == thisRoomNumber)` **and** `if (masterObjects[local].hotNum != -1)` — two separate inner
ifs, not a `&&`.

**69. The fish reads `data.g.height` for an object whose payload is `enemyType`, and gets the right
bytes by luck.** `Dynamics3.c:522`. `applianceType.height` and `enemyType.length` are both the
`short` at bytes 4-5 (`GliderStructs.h:67`, `:77`), so the value is correct. The **delay** fields
are not interchangeable: `applianceType.delay` is byte 7 and `enemyType.delay` is byte 6
(`:69` vs `:78`) — the two structs order `byte0` and `delay` oppositely.
*Symptom:* route the fish's leap height through an Appliance decoder and you also read the wrong
`delay`. `internal/house/object.go:424-443` models both orderings correctly; use `Enemy().Length`
and `Enemy().Delay` and note why the C's member name differs.

**70. `distance` in the grease path is `data.c.length`, not `data.a.distance`.**
`ObjectDrawAll.c:378`/`:399` pass `thisObject.data.c.length`. Both live at data offset 4-5, so the C
is indifferent; a typed Go house model is not.
*Symptom:* read `Blower().Distance` and grease trails get whatever the bonus `length` happens not to
be. Use `Object.Bonus().Length`.

**71. `kSoundTrigger`'s `data.e.where` is a sound resource ID, not a room link.**
`ObjectRects.c:926` gates the hot spot on `LoadTriggerSound(...) == noErr`.
*Symptom:* feed it to `ExtractFloorSuite`/`GetRoomNumber` and you get a nonsense room, and the
`where != -1` gate that every other switch type uses does not apply.

**71a. That gate is *stateful*, and it silently caps a room at one working sound trigger.**
Added after 1.5a; not in the original assembly. `LoadTriggerSound` (`Sound.c:265-303`) loads into a
**single reserved slot**, `theSoundData[kMaxSounds - 1]`, and its first line is

```c
if ((dontLoadSounds) || (theSoundData[kMaxSounds - 1] != nil))
    theErr = -1;
```

so it fails outright when the slot is already occupied. `DumpTriggerSound()` (`:305-310`) is the only
thing that frees it, and it is called from **`DrawLocale`'s reset head** (`RoomGraphics.c:58`, twelve
lines before `ListAllLocalObjects()` at `:72`). Three consequences, all observable:

- A room's **first** `kSoundTrigger` takes the slot and gets a `kSoundIt` rect. Every later one in
  the same room fails the `!= nil` test and gets **no hot spot at all** — it is not a silent
  trigger, it is not a trigger. This is a real per-room limit, not an artefact of a port.
- The limit is per *locale*, not per game: the slot is re-armed on every room change, so the same
  authored object works again when the player comes back.
- A trigger whose `snd ` resource does not exist leaves the slot **free** (the `theSound == nil`
  branch returns -1 without allocating). So a room whose first sound trigger names a missing sound
  and whose second names a real one gets a working rect on the *second* — order matters, and it is
  the order of the `objects[]` slots, not of anything visible in the editor.

*Symptom of getting it wrong:* model the gate as a pure `soundExists(id)` predicate and every sound
trigger in a room composes a rect, so a room authored with three of them fires three chords at once
and its hot-spot indices are shifted against the original's for every object listed after them —
which moves every `hotNum` in the room and therefore what every switch in it switches.

*Port:* `World.TriggerSoundExists func(soundID int16) bool` is the resource lookup (nil = the C's
`dontLoadSounds` short circuit) and `Room.TriggerSoundHeld` is the slot. `Room`, not `World`,
precisely because `DumpTriggerSound` runs in `DrawLocale`'s head, so the slot's lifetime *is* a
locale's even though the C's array is a file-scope global. `game.loadTriggerSound` is the two lines
that join them; `TestSoundTriggerIsOnePerRoom` pins all three consequences.

**72. `case kLgTrigger:` in the hot-spot action dispatcher is an object type, not an action.**
`Interactions.c:1352`, inside `switch (who->action)`. `kLgTrigger` is 0x48 = 72
(`GliderDefines.h:384`) and the 28 actions run 0..27 (`:282-309`). No hot spot is ever created with
action 72; the `kLgTrigger` *object* correctly produces `action = kTriggerIt`
(`ObjectRects.c:911-914`).
*Symptom:* a Go typed action enum will not compile it. Delete the label; do not add an action 72.

**73. `byte0` and `byte1` are write-only across the entire program, and the dart never writes
`.room` at all.** Every `byte0`/`byte1` write is in `AddDynamicObject` or `ZeroDinahs`; there is no
reader. `Dynamics3.c:437-463` (`kDartLf`/`kDartRt`) goes from `position` (`:458`) straight to
`byte0` (`:459`), skipping `dinahs[].room`, which reads back as 0 because `ZeroDinahs` (`:175`) ran
first; the only reader is `UpdateOutletsLighting` (`Trip.c:241`), which filters `type == kOutlet`
first.
*Symptom:* build the object-index lookup on `byte0` and it always misses — the live mapping is
`masterObjects[].dynaNum`, assigned by the correlation loop at `ObjectDrawAll.c:953-961` after
`dynamicNum` is reset to `-1` per object at `:45`. And any room-filtered iteration over dynamics a
port adds will diverge unless the dart's omission is reproduced or proved unreachable again.

**74. `numLights` is scratch that ends on the central room's value, and it is baked into
`dinahs[]`.** `DrawLocale` reassigns it before each of the nine rooms (`RoomGraphics.c:80`, `:84`,
`:88`, `:92`, `:96`, `:100`, `:107`, `:112`, `:118`) and finishes with the central room's.
`AddDynamicObject` stores it into `dinahs[].hVel` for outlets (`Dynamics3.c:314`) — the fourth
overload of that field. `UpdateOutletsLighting` (`Trip.c:235-244`) repairs it and is the only reader
of `dinahs[].room`. `RedrawRoomLighting` also relies on `numLights` still holding the central room's
count (`RoomGraphics.c:445-448`).
*Symptom:* recompute `numLights` per room at use time and outlets in a multi-room locale light from
the wrong room's count.

**75. `thisRoom` is a read cache; every persistent write goes to the house.**
`CopyThisRoomToRoom` (`Room.c:354-365`) is called only from `CopyRoomToThisRoom` (`:346`) and the
editor. `SetObjectState` writes the house (`Objects.c:366-699`). `HandleRoomVisitation` writes both
(`Transit.c:440` and `:443`) and the `thisRoom` write is dead.
*Symptom:* make `thisRoom` a pointer into the house — which is the tempting thing to do in Go — and
every write the original discarded becomes persistent. It happens to be invisible for `visited`; it
is not invisible in general.

---

### 4.E Capacity, ordering and RNG determinism

**76. `AddActiveRect` fails silently at 56, and the "can't happen" bound is wrong.**
`if (nHotSpots >= kMaxHotSpots) return (-1);` (`ObjectRects.c:280-281`, `kMaxHotSpots = 56`,
`GliderDefines.h:259`). `CreateActiveRects` never checks the return except to pass it along, so
every rect past the 56th is dropped: the object still **draws** but has no collision, no reward and
no switch. `live-state-inventory` argues the table cannot overflow because no type creates more than
two rects, giving `24 * 2 = 48 < 56` — that premise is false. The five flame types create **three**
(trap 62), so the crude bound is `24 * 3 = 72`.
*Symptom:* nineteen tall tapers in one room and later objects lose their hot spots. Keep the `-1`
return, keep trap 9's guard, and either census the corpus or pin the behaviour with a test.
`UNVERIFIED:` whether any shipped room actually exceeds 56. (§6.1 item 4 runs the census this asks
for and gets a peak of 53, in Slumberland room 256 — but that figure is a generous upper-bound
estimate scored from the object mix, not a build of the real table, so it narrows the `UNVERIFIED:`
above rather than closing it. It also independently confirms the three-rect finding.)

**77. The `dinahs` table holds 18, the budget is shared across all nine rooms, and the central room
registers LAST.** `kMaxDynamicObs = 18` (`GliderDefines.h:265`); `AddDynamicObject` returns `-1` at
`Dynamics3.c:193-194` with no error, no art and no hazard. `DrawLocale` registers NW, NE, N, SW, SE,
S, W, E, **Central** (`RoomGraphics.c:80-121`) while `ListAllLocalObjects` builds `masterObjects`
Central-first (`Objects.c:312-328`) — two different orders, and registration follows the *draw*
order. The seven back-map appliances register for all nine rooms; the movers register only for the
central room. So an appliance-heavy neighbourhood starves the player's own room's balloons, darts
and drips.
*Symptom:* a growable Go slice silently diverges in exactly the rooms the original was tuned in.
Keep the cap and keep the order. `UNVERIFIED:` whether the drop is *reachable* in shipped data —
`live-state-inventory`'s census finds three rooms at exactly 18 and none above, which if correct
means the `-1` return is never taken on the original houses; `dynamics-core` and `dynamics-movers`
both assert reachability by arithmetic without a census. Do not write a fidelity test that asserts
drops until this is settled.

**78. `savedMaps` has 24 slots for the whole nine-room locale, the central room is composed last,
and `legit != -1` gates the DRAW as well as the animation.** `kMaxSavedMaps = 24`
(`GliderDefines.h:260`); `kMaxCandles` (20) is unreachable because the shared pool binds first. One
slot per clock/paper/battery/bands/foil/helium/grease jar, **two** per cuckoo and **two** per star,
summed over all nine rooms in `DrawARoomsObjects` order. `ObjectDrawAll.c:295-296` and its eight
siblings skip the *draw* when the slot claim failed.
*Symptom:* the 25th backed-up object is invisible and still collectible, because
`ListAllLocalObjects` builds hot spots on a completely independent path. A full-recompose renderer
must therefore *count slot occupancy* even if it never stores a pixel. For grease the consequence is
worse: `ObjectDrawAll.c:379` skips `DrawGreaseRt` and the jar is invisible with a live `kRewardIt`
rect, whose collection calls `SpillGrease(-1, hotNum)` (trap 6).

**79. `RestoreEntireGameScreen` calls `DrawLocale` bare, and it is reachable during normal play.**
`Play.c:800-820`, `DrawLocale` at `:814`, with **no** `NilSavedMaps` and **no**
`DetermineRoomOpenings` and **no** `InitGarbageRects`. Its only caller is `Banner.c:234`, under
`if (WaitForInputEvent(30))` in the stars-remaining banner — i.e. pressing a key during that pause.
`DrawLocale` has exactly three call sites (`RoomGraphics.c:416`, `Play.c:163`, `Play.c:814`) while
`ReadyLevel` has exactly four (`Transit.c:298`, `:335`, `:374`, `:413`); the two three/four counts
in the reports are about different functions and do not conflict.
*Symptom:* `numSavedMaps` roughly doubles, so a room needing more than 12 slots loses its later
prizes and animations for the rest of its life. It also silently clears `takingTheStairs`
(`RoomGraphics.c:127`), which can rescue trap 47's stuck flag. Structurally: the locale rebuild must
be a **method** separable from room construction, because this path reconstructs `masterObjects`,
`hotSpots`, `dinahs`, `grease` and `triggers` while leaving `savedMaps`, sparkles, flying points and
the dirty-rect queues alone. `internal/render/locale.go:227` currently truncates `SavedMaps` inside
`DrawLocale`, conflating `NilSavedMaps` into it, and so does not reproduce this.

**80. Every room-load `RandomInt` is on the success path only, and slot exhaustion therefore shifts
the whole RNG stream.** `DynamicMaps.c:338` (flame mode), `:422` (tiki), `:508` (coals), `:594`
(pendulum), `:685` (star) — each **inside** `if (savedNum != -1)` after
`savedNum = BackUpToSavedMap(...)` (verbatim at `:333-338` and `:680-685`) — plus
`Dynamics3.c:208` (`dinahs[].timer = RandomInt(60) + 15` for `kSparkle`). Two more during play:
`Dynamics.c:304` (`RandomInt(240) + 60`), `:492` and `:506` (`200 + RandomInt(200)`). The room-load
calls are interleaved in the order objects are visited across all nine rooms. And `RandomInt` is
**inclusive** of its argument (`Utilities.c:72-82`), so `RandomInt(60) + 15` is 15..75.
*Symptom:* consume the RNG in a post-pass and every subsequent random draw in the game diverges.
Consume it at the registration site, in `DrawARoomsObjects` order, only when the slot was granted.
`internal/render`'s `Anim` and `Dynamic` structs (`locale.go:115-121`, `:124-131`) have no field for
these seeded values yet.

**81. Screen size and `numNeighbors` change gameplay, not just framing.** `houseRect` and
`playOriginH`/`V` come from `thisMac.screen` (`InterfaceInit.c:196-204`), and
`testRect = houseRect; ZeroRectCorner(&testRect);` (`ObjectDrawAll.c:36-37`) is what every
`SectRect` registration gate compares against. So *which* appliances, flames, tikis, coals,
pendulums, stars and grease jars register is resolution-dependent — and because `savedMaps` is
capped it is visible as missing art. `numNeighbors` is a user preference (`Main.c:111` from
`thePrefs.wasNumNeighbors`, values 1/3/9 via `Settings.c:1146-1176`, forced to 1 on a screen
<= 512 px at `Main.c:191-192`) and it gates both the drawn set (`RoomGraphics.c:78`, `:105`) and the
object index (`Objects.c:311`, `:320`).
*Symptom:* combined with trap 80, the RNG stream is a function of screen resolution and of whether
the player pressed a key at the stars banner. And at `numNeighbors == 1` a switch linked outside the
central room gets `localLink = -1`, i.e. a *different game*. `internal/render/view.go` already pins
640x480 via `DefaultView()`; Stage 1.5 must not introduce a second source of geometry.

**82. `ListAllLocalObjects` visits the neighbours in a non-numeric order and its correlation loop
has no `break`.** `Objects.c:311-330`: central, then East, then West (if `numNeighbors > 1`), then
North, NorthEast, SouthEast, South, SouthWest, NorthWest (if `> 3`). That is **not**
`kCentralRoom..kNorthWestRoom` order, and `localLink` stores indices into this list, so the order is
observable. The correlation loop at `:332-346` has no `break`, so with duplicate
`(roomNum, objectNum)` pairs the **last** match wins — harmless only because
`GetNeighborRoomNumber` and `GetRoomNumber` both return the *first* match (`:629`, `:753`), so no
room number can occupy two of the nine slots. `localLink` stays `-1` when the target is outside the
visible set (`:278`).
*Symptom:* reorder the visit and every `localLink` in the house shifts.

**83. The build order and the draw order are different, and one is not the reverse of the other.**
Build (`masterObjects`, `Objects.c:312-328`): Central, E, W, N, NE, SE, S, SW, NW. Draw
(`RoomGraphics.c:80-120`): NW, NE, N, SW, SE, S, W, E, Central. Reversing the build order gives NW,
SW, S, SE, NE, N, W, E, Central — **not** the draw order. The only true summary is "central is first
in the build order and last in the draw order".
*Symptom:* implement "reverse" literally and you get wrong slot indices in every multi-room locale,
which decides `dinahs[]`/`savedMaps[]`/`grease[]` slot numbers, and therefore both
`HandleDynamics`' iteration order (which mover wins the `hDesiredVel` **assignment** race, trap 89)
and which entries are dropped at saturation.
`live-state-inventory`'s one-word "reverse" is wrong; `background-animations` states the two orders
separately and correctly.

**84. Both dirty-rect queues cap at 47 of 48 and clamp against **different** rectangles, with an
if/else-if chain.** `if (numWork2Main < (kMaxGarbageRects - 1))` (`Render.c:67`) and the same at
`:86`, with `kMaxGarbageRects = 48` (`Render.c:20`). `AddRectToWorkRects` clamps to
`justRoomsRect` in global coordinates (`:70-77`); `AddRectToBackRects` clamps to `0`/`workSrcRect`
(`:89-96`). In both, the second clamp is an `else if`, so a rect overhanging on both sides keeps one
overhang. `AddRectToWorkRectsWhole` (`:103-132`) additionally rejects fully-outside (`:107-111`) and
degenerate (`:124-128`) rects.
*Symptom:* fuse the two clamps into independent ifs — the exact class the Stage 1.4 audit found 57
of — or write one shared helper for both adders, and rects land in the wrong place. Note also that
`Env.AddRectToWorkRects` as the player uses it takes **room-local** coordinates while every
appliance handler passes a **global** rect. 18 dynamics contribute up to 36 rects, so saturation is
reachable, and past it rects are dropped and stale pixels persist.

**85. `savedMaps` is declared two incompatible ways across translation units, and it is benign only
by accident.** `Objects.c:73` defines `savedType savedMaps[kMaxSavedMaps];` (the array everything
renders from, via `extern savedType savedMaps[]` at `DynamicMaps.c:36` and `Render.c:54`), while
`StructuresInit2.c:40` declares `extern savedPtr savedMaps;` and `:233` does
`savedMaps = (savedPtr)NewPtr(...)`. Same linker symbol, so the `NewPtr` result is written into the
first four bytes of the array — which by `savedType`'s layout is `savedMaps[0].dest.top` and
`.left`, not `.map` — and the `savedMaps[i].map = nil` loop at `:237-238` nils the map fields of the
*leaked* block. BSS already zeroed the real array's `map` fields, so `NilSavedMaps` disposes
nothing bogus, and `BackUpToSavedMap` overwrites slot 0's `dest` before anyone reads it.
(`RoomGraphics.c:35` adds a third, also-incompatible `extern CGrafPtr savedMaps[];`, unused.)
*Symptom:* none. Do not reproduce; collapse to one slice, and do not go looking for the
initialisation that "must" be somewhere.

**86. `ZeroTriggers` discards pending triggers on every room change, and `ZeroDinahs` discards every
mover's mid-flight state.** `RoomGraphics.c:55` and `:52`, both the first things `DrawLocale` does.
*Symptom:* gameplay-visible and correct: a trigger armed with a two-second delay and then walked
away from never fires. Do not add a cache keyed on room number. The sole exception is a same-room
transport, which skips `ReadyLevel` and therefore skips the whole rebuild (trap 92).

**87. Sparkle and flying-point capacity is 3 each, and one switch throw can consume two.**
`kMaxSparkles = kMaxFlyingPts = 3` (`GliderDefines.h:251`, `:253`).
`RestoreFromSavedMap(doSparkle = true)` produces one and trap 10's garbage `AddSparkle` produces
another.
*Symptom:* several enemies recycling on one frame silently lose their sparkles. `AddSparkle` drops
silently when three are live.

---

### 4.F Two-player asymmetries

**88. The three transit globals are shared by both gliders, and Stage 1.4's per-glider `Transit`
is not a drop-in replacement.** `transRect` (`Player.c:49`), `transRoom` (`Player.c:52`) and
`linkedToWhat` (`Transit.c:17`) are written by whichever glider most recently entered a transit
object (`Modes.c:164-170`, `:227-233`, `:260-266`, `:302-308`) and read by
`ReadyGliderFromTransit` for **both** gliders (`Transit.c:328-329`). `FollowTheLeader`
(`Transit.c:533-551`) and `OffAMortal`'s `mortals == -1` re-dispatch (`Player.c:1575-1595`) call
`TransportRoomToRoom`/`MoveMailToMail`/`MoveDuctToDuct` with the **surviving** glider, whose own
`Transit` was never set for this transit — the C reads the globals, which still hold the **dead
leader's** destination. That is the entire point: the survivor follows the leader.
*Symptom:* read `g.Transit` there and the survivor goes to the wrong room, or to `{0,0,0,0}`. Fix:
World keeps a single pending-transit `Link` mirroring the three globals, published by the hot-spot
dispatcher; the four transit functions read **that**. `g.Transit` (`glider.go:93`) stays as a shadow
copy, which `StartGliderMailingOut` (`modes.go:162`) legitimately reads for the *departing* glider.

**89. `foilTotal`, `bandsTotal`, `batteryTotal`, `mortals` and `theScore` are one shared pool, and
the collision call order decides who dies.** One declaration:
`short batteryTotal, bandsTotal, foilTotal, mortals;` at `Play.c:52`, plus `theScore`
(`Player.c:50`), `numStarsRemaining` (`Banner.c:28`), `otherPlayerEscaped` (`Interactions.c:42`).
`mortals = kInitialGliders; if (twoPlayerGame) mortals += kInitialGliders;` (`Play.c:341-343`), and
`Player.c:1498` tests `mortals < -1` for the both-dead case. The four-way two-player block (e.g.
`Dynamics2.c:44-60`) always calls glider 1 first:
```c
if (twoPlayerGame) {
    if (onePlayerLeft) {
        if (playerDead == theGlider.which) CheckDynamicCollision(who, &theGlider2, false);
        else                               CheckDynamicCollision(who, &theGlider,  false);
    } else {
        CheckDynamicCollision(who, &theGlider,  false);
        CheckDynamicCollision(who, &theGlider2, false);
    }
} else
    CheckDynamicCollision(who, &theGlider, false);
```
*Symptom:* with `foilTotal == 1` and both gliders touching the same mover on an even frame, glider 1
is shoved, spends the last sheet (`Dynamics.c:62`) and enters `kGliderLosingFoil`; glider 2 then
fails both arms of `(foilTotal > 0) || (mode == kGliderLosingFoil)` at `:51` and takes
`StartGliderFadingOut` at `:69` — it **dies**. Player 1 always wins that race. Preserve the order.
Also: `Scoreboard.c:209-211` clamps the *displayed* `mortals` to 0, but `QuickGlidersRefresh`
(`:268-299`) does not — `NumToString((long)mortals, ...)` at `:284` — so a negative count is briefly
drawn by the quick path.

**90. `HandleOutlet`'s `onePlayerLeft` arm passes the wrong `doOffset`, and the fix is a
divergence.** All seven movers store `dest` room-local and pass `false`; the appliances store `dest`
in screen space and pass `true`. `HandleOutlet` (`Dynamics.c:528-599`) passes `false` at `:539` and
`:541` (the `onePlayerLeft` arm) and `true` at `:545`, `:546`, `:550`. Since the outlet's `dest` is
global, `false` is wrong.
*Symptom:* in a two-player game after a death, the surviving glider is tested against a rect offset
by `+playOriginH/V` (+64, +79), so the zap misses entirely. Factor the four-way idiom into one
shared helper and you silently fix it. Neither `enemies.md` nor `object-dynamics.md` records this.

**91. `MoveRoomToRoom` writes BOTH gliders' `enteredRect`, with no dead-player guard.**
`Transit.c:177-178`, `:186`, `:209-210`, `:218`, `:233-234`, `:239`, `:265-266`, `:271`.
`OffsetGlider` and `UndoGliderLimbo` *do* skip the permanently dead glider; the `enteredRect`
assignment does not.
*Symptom:* any Go transition method that takes a single `*Glider` and touches only that glider is
wrong for two players. Conversely, adding the dead-player guard for symmetry diverges on any test
that inspects `enteredRect`.

**92. Same-room transits skip the entire rebuild while still rendering a frame.**
`TransportRoomToRoom` (`Transit.c:314`), `MoveDuctToDuct` (`:352`) and `MoveMailToMail` (`:391`) each
compute `sameRoom = (transRoom == thisRoomNumber)` and guard `ForceThisRoom`, `ReadyLevel` and
`WipeScreenOn` with it; `MoveRoomToRoom` (`:151`) always calls `ReadyLevel()` at `:298`.
*Symptom:* a duct that leads back into the same room preserves `nHotSpots`, every `stillOver`, every
`isOn` mutated by `SetObjectState`, every grease rect, and every mid-flight mover — and the
unconditional `RenderFrame` at `:341` then double-steps shreds, sparkles and flying points because
nothing zeroed them (trap 24).

**93. `kMoveItUp`/`kMoveItDown` have no index check in the two-player handshake, while the other
five transit actions do.** The staircase cases compare only `otherPlayerEscaped`
(`Interactions.c:1271`, `:1306`); the mail/duct/transport cases additionally require
`activeRectEscaped == index` (`:1405`, `:1452`, `:1495`, `:1534`, `:1569`).
*Symptom:* two up-staircases in one room and the second player can use the wrong one.

**94. `kPlayerEscapingUpStairs` (-8) and `kPlayerEscapedUpStairs` (-6) are a deliberate two-state
gate.** The dispatcher writes the *-ing* form at `Interactions.c:1266`; `MoveGliderUpStairs` writes
the *-ed* form at `Player.c:363` when the climb completes; the second glider's gate at `:1271`
accepts only the *-ed* form. Both forms are read again at `Player.c:1546`/`:1555` in the
"other player died mid-transit" recovery.
*Symptom:* fuse the two sentinels and the second glider can enter the stairs while the first is
still climbing.

**95. `activeRectEscaped`, `firstPlayer` and `playerDead` are never initialised and never reset, and
`NewGame` does not touch them.** `NewGame` (`Play.c:112-118`) initialises `gameFrame`, `numBands`,
`demoIndex`, `saidFollow`, `otherPlayerEscaped`, `onePlayerLeft`, `playerSuicide` — not
`activeRectEscaped` (`Interactions.c:42`, five writes at `:1398`, `:1444`, `:1487`, `:1527`, `:1562`,
five reads at `:1405`, `:1452`, `:1495`, `:1534`, `:1569`), not `firstPlayer` (`Transit.c:18`, one
writer at `Modes.c:467` inside `FlagGliderInLimbo`, two readers at `Transit.c:145` and `:292`), not
`playerDead`, not `takingTheStairs`.
*Symptom:* `activeRectEscaped` is a hot-spot **index** into a table that is rebuilt on every room
change, so it is only ever meaningful within one generation; World must document that it is compared
for equality and never dereferenced. Do **not** initialise it to `-1` — the C starts it at 0 (BSS),
and the guarded reads behave differently. A fidelity test comparing back-to-back games in one
session must not reset any of these.

---

### 4.G Arithmetic, coordinate spaces and geometry

**96. `kInvisBlower` and `kLiftArea` select **one** rect from a bitfield, or none at all.**
`ObjectRects.c:523` and `:595` are each a single `switch (theObject.data.a.vector & 0x0F)` with one
`AddActiveRect` per case and a `break` after each: `1` = up → `kLiftIt`, `2` = right →
`kPushItRight`, `4` = down → `kDropIt`, `8` = left → `kPushItLeft`, matching the bit comment at
`GliderStructs.h:16`. There is **no default**, so a low nibble of 0, 3, 5, 6, 7 or 9-15 yields no hot
spot at all (and for `kInvisBlower` an uninitialised `bounds` that is simply never used).
`kInvisBlower` uses `distance + 24` and ignores `tall`; `kLiftArea` uses `distance x (tall * 2)`.
*Symptom:* `hotspot-pipeline.md` lists these two among the multi-rect `hotNum`-aliasing hazards,
which sends a porter hunting a hazard that does not exist while hiding the real one — a malformed
`vector` produces a completely inert object. This closes that report's openQuestion 5, which
guessed the field was `data.a.byte0`.

**97. `IsRectLeftOfRect` has an operator-precedence bug that must be reproduced.**
`RectUtils.c:185`: `offset = (rect1->right - rect1->left) - (rect2->right - rect2->left) / 2;` —
`/` binds tighter, so this is `wide1 - (wide2 / 2)`, not a half-width comparison. With
`kGliderWide = 48`: balloon (24 wide) → 0, copter/ball/toast (32) → +8, drip/fish (16) → -8, dart
(64) → **+40**. The test is `rect1->left < rect2->left + offset`, and the second argument is the
glider's raw `dest`, not the inset box the intersection test used.
*Symptom:* for a dart, "left of the glider" is true unless `dart.left >= glider.left + 40`, and with
`SectGlider(scrutinize = true)` insetting 5 px per side the overlap requires
`dart.left < glider.left + 43` — a 3-pixel band in which the answer is false. So a dart almost
always shoves the glider **rightward regardless of which way it is flying**, including into itself.
Transcribe the expression verbatim; do not add the parentheses.

**98. `CenterRectInRect` divides by 2 with a numerator that can be negative.**
`rectA->top = rectB->top + (RectTall(rectB) - tallA) / 2;` (`RectUtils.c`).
`srcRects[kCeilingTrans]` is 56x15 and the glider is 20 tall, so the numerator is `-5`, and C89
truncates toward zero: `-5/2 == -2`.
*Symptom:* write `>> 1` and you get `-3`, and every ceiling-duct arrival is one pixel off. Go's `/`
truncates toward zero too — but only if you write `/2`.

**99. `WebGlider` uses `>> 3`, not `/ 8`.** `Interactions.c:1750`, `:1752`. Arithmetic right shift
floors toward negative infinity; Go's `/` truncates toward zero.
*Symptom:* for a glider left of the web's centre the numerator is positive and the two agree; right
of centre it is negative and they differ by one pixel per frame, which the twang timing makes
visible. Use `>> 3` on a signed type. Everywhere *else* in this subsystem the C uses division and
you must **not** shift: `thisGlider->hVel /= 4` and `/= 2` (`Interactions.c:773-774`, `:839-840`)
would round the wrong way for a left-moving glider under `>> 2`.

**100. `-((vVel * 3) / 4)` relies on truncation toward zero on a positive `vVel`.**
`Dynamics2.c:392`, with the stop condition `vVel == 0` tested **after** the negation (`:393`), so an
impact velocity of 1 gives `-(3/4) = 0` and the ball stops.
*Symptom:* a float implementation with `math.Floor` never lets the ball come to rest.

**101. `QSetRect`'s parameter order is `(rect, left, top, right, bottom)`.**
`RectUtils.c:210-216` — not the `Rect` struct's own declaration order of `top, left, bottom, right`.
Every seed rect in `Grease.c` uses it (`:63`, `:65`, `:71`, `:92`, `:99`, `:146`, `:215`, `:218`), so
`QSetRect(&src, 0, -2, 2, 0)` is a 2-wide, 2-tall rect sitting *above* the anchor.
*Symptom:* `ZeroDinahs`' `QSetRect(&dinahs[i].dest, 0,0,0,0)` hides the mistake, so you will not
notice until the sprite tables in `StructuresInit.c` and every grease rect come out transposed.

**102. All four overlap tests are inclusive on every edge, and `GliderInRect` is the mirror.**
`GliderHitTop` (`Interactions.c:68-77`), `SectGlider` (`:118-127`) and the band test
(`RubberBands.c:72-81`) use `theRect->bottom < glideBounds.top` — strict `<`, so
`bottom == top` counts as a hit. `GliderInRect` (`:140-149`) uses
`glideBounds.top < theRect->top` → false, so exact edge equality is **contained**.
*Symptom:* Go's `image.Rectangle.Overlaps` is half-open and gives the opposite answer on shared
edges — every "resting on" contact stops registering.

**103. `SectGlider`'s burning inset and `GliderInRect`'s lack of one do not agree.**
`SectGlider` raises the glider's `top` by 6 for `kGliderBurning` at `Interactions.c:107-108`
**before** applying the `scrutinize` inset of 5, so a burning glider's effective top is
`dest.top + 11`. `GliderInRect` (`:138`) uses the raw `dest` with no inset and no burning trim.
*Symptom:* on a `doScrutinize = true` rect (`kShredIt`, `kMicrowaveIt`, `kWebIt`) the sweep admits
the glider using the tight 38x10 box and the case then demands containment of the full 48x20 box —
the tight box gets you in and the loose box decides the outcome. And a burning glider needs 6 px
more headroom to satisfy `GliderInRect` than a normal one.

**104. Every velocity write in the interaction phase is a *request*, attenuated the same frame.**
`MoveGlider` (`Player.c:64-147`) ramps `hVel` toward `hDesiredVel` by `kHImpulse` = 2 and then sets
`hDesiredVel = 0` (`:78`); it ramps `vVel` toward `vDesiredVel` by `kVImpulse` = 2 and then sets
`vDesiredVel = kGravity` = 3 (`:92` — **not** `:90`, which is the clamp *inside* the ramp and says
the opposite); it clamps `hVel` to ±`kMaxHVel` = 16 and leaves `vVel` **unclamped**. Because
`HandleInteraction` (`Play.c:482`) precedes `HandleGlider` (`:487`), a `kLiftIt` writing
`vDesiredVel = -6` moves the glider by at most 2 px of new velocity that frame, and the vent must
re-assert every frame. The exceptions that bypass the ramp are `kSlideIt`'s `vVel =`
(`Interactions.c:1378`) and `BounceGlider`/`GliderHitTop`'s `hVel =` (`:160`, `:162`, `:93`).
*Symptom:* `AddBand`'s -10 recoil displaces 8 on the first frame, a band's push of 10 displaces 8,
and a `kSlideIt` snap of -22 displaces -20 and can move the glider arbitrarily far because `vVel` is
never clamped. Write `vDesiredVel` where the C writes `vVel` and the glider falls straight through
the slide line.

**105. `kLiftIt`/`kDropIt` assign; `kPushItLeft`/`kPushItRight` accumulate; and
`CheckDynamicCollision` assigns.** `Interactions.c:1203` and `:1207` use `=`; `:1211` and `:1215`
use `+=`; `Dynamics.c:54` and `:56` use `=`. `vDesiredVel` in the dynamic channel is written only
when `dinahs[who].vVel < 0` (`:57-58`).
*Symptom:* the sweep has no early exit, so the **last** vent in index order wins outright while all
fans sum: two fans facing each other cancel exactly, and a ceiling vent below a floor vent in index
order cancels the floor vent completely. And because `MoveGlider` zeroes `hDesiredVel` at the end of
the previous frame while `HandleDynamics` runs first, a mover's shove and the player's own thrust
**add**, but a second mover's shove in the same frame **overwrites** the first.

**106. Grease's equilibrium is 2 px below the line, not on it.** The applied displacement is
`min(2 - d, 3)` for `d = dest.bottom - bounds.top`, so `d` converges to 2 and the 2-px-tall rect
keeps overlapping the glider box (inclusive edges, no inset — trap 102).
*Symptom:* snap `dest.bottom` to `bounds.top` and the glider is *not* overlapping next frame, so the
slide flickers on and off.

**107. Grease lives in two coordinate spaces at once and the conversion happens exactly once.**
`grease[].dest`, `.start` and `.stop` are in work-map space; `hotSpots[].bounds` is room-local. The
falling promotion converts once with `QOffsetRect(&src, -playOriginH, -playOriginV)`
(`Grease.c:66`) before storing the seed bounds; the spreading branch then advances the *work-space*
`start` (`:94`/`:101`) and the *room-local* `bounds` edge (`:95`/`:102`) independently by the same
±2; `RedrawAllGrease` converts back with `+playOrigin` (`:287`).
*Symptom:* mix the two and the slide rect sits a whole room-origin away from the paint, silently.
The same split runs through the whole renderer: everything in the five strip tables (`dest`, `src`)
is in composed back-map coordinates because `OffsetRectRoomRelative`
(`ObjectRects.c:1093-1131`) already added `playOrigin` plus the neighbour offset, while shreds,
bands, dynamics and the arguments to `AddSparkle`/`AddFlyingPoint`/`AddAShreddedGlider` are
room-relative and get `+playOrigin` at draw time.

**108. `AddSparkle` and `AddFlyingPoint` mutate the caller's `Rect` through the pointer — and only
when there was a free slot.** `DynamicMaps.c:176-179` and `:206-209` add `playOriginH`/`V` in place,
**inside** the `if (numSparkles < kMaxSparkles)` / `if (numFlyingPts < kMaxFlyingPts)` guard. The C
defends by copying first at `Dynamics2.c:95-96`, `:117-118`, `:124`, `:197-198`, `:225`, `:232`,
`:303-304`, `:341`, `:348`, and `Dynamics.c:306-307`; `Render.c:603` does **not** — it passes
`&shreds[i].bounds`, so a finished shred's bounds is permanently shifted, harmless only because
`frame == 20` means `RenderShreds` never touches it again. `RestoreFromSavedMap` relies on the
convention in the other direction, converting `savedMaps[i].dest` back to room-local with
`QOffsetRect(&bounds, -playOriginH, -playOriginV)` at `:156` before calling `AddSparkle` at `:157`.
*Symptom:* `AddSparkle(&dinahs[who].dest)` drifts the object's own position by one `playOrigin` per
sparkle, and only on the sparkles that got a slot. A Go signature taking `Rect` by value silently
fixes the bug and diverges; taking `*Rect` preserves it. Whichever you choose, match the copies at
all fourteen call sites exactly.

**109. `AddGrease`'s ∓8 undoes `BackupGrease`'s ±8; it is not an art offset.** `BackupGrease` walks
the caller's `src` by +2 four times, net +8, for a right-spilling jar (`Grease.c:159`/`:167`);
`AddGrease` then subtracts 8 (`:223-226`), so `grease[].dest` equals the jar's original `itsRect`.
*Symptom:* "fix" the mutation by passing a copy into `BackupGrease` and you must also delete the
∓8, or the whole animation is displaced. `docs/analysis/interactions.md` §19.2 describes this
backwards.

**110. The slide rect leads the painted trail by 2 px permanently, and the runtime and static
spills have different geometry.** Runtime hot rect `[h+36, h+34+distance)`, paint ending at
`h+32+distance`; static hot rect `[h+31, h+26+length)` (`ObjectRects.c:692-694`, offset by `32-1`)
with paint starting at `h+37` (`ObjectDraw.c:1332`). Two independent code paths, deliberately not
shared.
*Symptom:* walking out of a room and back changes both the visible line and the slippery region by a
few pixels. A fidelity test that compares the two paths fails for the right reason. Reproduce; do
not unify.

**111. The four grease frame source rects are not a vertical strip, and the 4-bit even-x fixups
differ between the candle and everything else.** `greaseSrcRt[3]` is at (32,297) and
`greaseSrcLf[1]` at (32,324) (`StructuresInit.c:371-387`), so `y = 243 + 27*i` gets the wrong art
for right frame 3 and left frames 1 and 3. Separately, `AddCandleFlame:326-331` adjusts `src.left`
*after* offsetting (`-1`, then `+2` if that went negative) while `AddTikiFlame:409-414`,
`AddBBQCoals:495-500`, `AddPendulum:584-589` and `AddStar:671-676` adjust `h` *before*; `AddGrease`
has none. All five are gated on `thisMac.isDepth == 4`.
*Symptom:* the fixups are dead on any modern surface — drop them, but know they exist so you do not
mistake the candle's variant for a bug.

**112. Hardcoded sprite dimensions sit next to `RectTall`/`RectWide` calls for the same sprite.**
`Dynamics2.c:387` uses a literal `32` for the ball, `:460` a literal `12` for the drip, `:534` a
literal `16` for the fish, `:211` a literal `32` for the copter's width, while `:102-103`,
`:208-209`, `:313-314`, `:326-327` use `RectTall`/`RectWide` on the same tables. They agree today.
*Symptom:* reproduce the literals so a future asset change diverges the same way the original
would.

**113. `HouseLegal.c`'s parity forcing is an editor pass, not a runtime invariant.**
`topLeft.h` and `length` are forced even at `HouseLegal.c:268-278`, which is why grease's ±2
spreading walk terminates exactly on `stop`.
*Symptom:* an odd `length` overshoots `stop` by 1 px and a `length < 5` gives the *static* rect
negative width. Do not assume evenness and do not add a clamp the original does not have.

---

### 4.H Original bugs and quirks that are visible and must be a conscious decision

**114. `HandleOutlet`'s bare `PaintRect` inherits whatever port the previous frame left set, and
the `SetPort` that would have fixed it is commented out one line above.** `Dynamics.c:578` paints
with no `SetPort`, relying on `Play.c:154`'s `SetPort((GrafPtr)workSrcMap)`; `:577` is the commented
one. Three reachable functions set the port and never restore it: `QuickScoreRefresh` →
`boardPSrcMap` (`Scoreboard.c:308`), called from `HandleDynamicScoreboard` at `:92` whenever
`theScore > displayedScore`, which runs at the **end** of the frame (`Play.c:470`/`:495`) so the
*next* frame's `HandleOutlet` inherits it; and `QuickGlidersRefresh` → `boardGSrcMap`
(`Scoreboard.c:273`), called from `Interactions.c:844` (extra-glider pickup) and `Player.c:1519`
(death). `QuickBatteryRefresh`, `QuickBandsRefresh` and `QuickFoilRefresh` only `CopyBits` and are
safe.
*Symptom:* after a score roll, a pickup or a death, a dark-room outlet's final-frame erase paints
into a scoreboard map and the last zap frame stays on screen. A Go port that always targets the
work surface is *more* correct and will diverge from a pixel-exact reference, so the choice must be
conscious. Note the dependency: this trap is why `HandleDynamicScoreboard` must be owned by
somebody — §1.6.2 specifies the function and §2 must scope `displayedScore` — even though no report's `liveState` claims `displayedScore`.

**115. Every strip's baked background is frozen at `DrawLocale` time, and the drawing order decides
what is in it.** Rooms compose NW, NE, N, SW, SE, S, W, E, Central; within a room, objects in slot
order. Anything drawn *after* a strip was baked and overlapping its rect is punched out on every
animation frame. `DrawFloorSupport` (`RoomGraphics.c:123-124`) paints the inter-floor beams **after**
every strip was baked — which is precisely why the candle family has a neighbour cull
(`ObjectDrawAll.c:88-101`, registering a non-central candle flame only if its 16x15 rect misses
`localRoomsDest[kCentralRoom]` expanded by `kFloorSupportTall` (44) top and bottom) and `kTiki`
(`:173-183`) and `kBBQ` (`:185-198`) do **not**.
*Symptom:* a tiki torch near the top of the south room erases a strip of beam every even frame — an
original artefact. A full-recompose port silently *fixes* all of this; expect the fidelity pass to
diff exactly here. What the cull actually catches: for N/S neighbours, flames within ~44 px of the
shared floor; for W/E, within 8 px of the shared wall; for the four diagonals, never.

**116. The five strip renderers enqueue only `AddRectToWorkRects`, never `AddRectToBackRects`, and
`CopyRectsQD` blits work rects first and erases back rects second.** `Render.c:620-626` (work→main)
then `:628-634` (back→work), in that order, so `AddRectToWorkRects(r)` means "blit r to the screen"
and `AddRectToBackRects(r)` means "erase r in `workSrcMap` afterwards". The strips carry their own
background, so there is no erase step.
*Symptom:* the animation persists in Work between its own update frames, and anything that *does*
enqueue a back-rect over it (glider, shadow, mirror reflection, band, flying point, sparkle,
dynamic, shred) restores clean background there at the **end** of the frame. Net effect: a glider
crossing a candle makes the flame blink at 15 Hz, and a glider crossing a pendulum can hide it for
up to 10 frames. `dynamics-core`'s two-frame appliance handshake (`timer == 1` writes `backSrcMap`,
`timer == 0` queues work→main) depends on exactly this polarity.

**117. The drip's `whole` is never refreshed in its idle branch, so the swell animation is
invisible after the first cycle.** `RenderDrip` (`Dynamics.c:236-253`) has no `moving` gate: it
always draws `dripSrc[frame]` at `dest`, queues `AddRectToBackRects(dest)` and queues
`AddRectToWorkRects(whole)`. The splash at `Dynamics2.c:454-466` teleports `dest` back to the
ceiling but leaves `whole` holding the last aerial sweep, and the idle branch at `:475-496` never
assigns `whole` — only the detach at `:490` does. Given trap 116's ordering, the swell frames are
composited into `workSrcMap` at the ceiling and then erased without ever being blitted, because the
work rect points near the floor. The first cycle is fine (`whole == dest` from
`Dynamics3.c:501`).
*Symptom:* from the second drip onward the ceiling shows only the baked `dripSrc[3]`. The drip is
uniquely exposed because it is one of only two ungated renderers and the only one whose idle branch
does not refresh `whole` (the fish's does, at `Dynamics2.c:557`). `UNVERIFIED:` how visible this is
depends on how similar `dripSrc[0..2]` and `dripSrc[3]` are, which nobody inspected.

**118. Five of the seven appliances start inert regardless of their authored state, and three
sounds are unreachable.** Stereo and microwave get `timer = 0` unconditionally
(`Dynamics3.c:361`, `:382`) and MacPlus/TV get `timer = 0` too (`:255`, `:275`); only coffee
(`:296`) and VCR (`:339`) get a non-zero timer when `isOn`. Since the outer gate is
`if (timer > 0)`, five of seven do nothing until a switch touches them and their visible state comes
purely from the static draw. The coffee maker's active loop oscillates 200↔(200..400) via
`Dynamics.c:506` and never reaches `timer == 1` or `0` except through `ToggleCoffee`; the VCR's
oscillates 115↔100 via `:629`, so **`timer == 5` at `:615-616` is unreachable and `kMacOnSound` is
dead code for the VCR**; the MacPlus `timer == 30` chime (`:406-407`) is reachable only from
`ToggleMacPlus`'s on-value 40, never from its off-value 10.
*Symptom:* seed a timer from `isOn` "helpfully" and every room entry plays boot chimes. A fidelity
test that asserts these three sounds fails against the real game.

**119. `Trigger*` for balloon, copter, dart and drip do not check `active`, and that is
observable.** `Trip.c:209-231` and `:188-192` write `timer` unconditionally, while every
decrement is inside `if (active)` (`Dynamics2.c:109-111`, `:217-219`, `:333-335`, `:477-479`).
`TriggerFish` (`:196-205`) *does* check.
*Symptom:* the trigger looks inert but has **armed** the object: when a switch later turns it on it
appears 5 frames later, or drips almost immediately, instead of waiting a full period. Related:
`TriggerDrip` is a clamp — `if ((!moving) && (timer > 7)) timer = 7;` (`Trip.c:190-191`) — so
writing `timer = 7` unconditionally speeds up a drip that was already about to fall. And
`ToggleStereos` is the only `Toggle*` that can refuse, gated on `timer == 0` (`Trip.c:87-91`).

**120. `SetObjectState`'s `kStereo` case flips a global settings flag and never touches the
object.** `Objects.c:584-588`: `newState = !isPlayMusicGame; isPlayMusicGame = newState;
changed = true;`. The house record's stereo `data.g.state` is written only by
`SetObjectsToDefaults` (`Play.c:676`, `state = isPlayMusicGame`).
*Symptom:* a switch on a stereo toggles the whole game's music preference and the object's stored
state never follows.

**121. Blower switches are not inert, and `Objects.c:411-414` is the only sound the file plays.**
`rewards-switches-triggers` closes its switch section with "a switch linked to furniture, a blower,
a transport, a clutter item or `kCobweb` ... produces no visible effect beyond the switch art".
True for four of the five; false for blowers. `HandleSwitches`' `linkIndex` dispatch has no blower
case, but `SetObjectState`'s blower family (eleven labels: `kFloorVent`, `kCeilingVent`,
`kFloorBlower`, `kCeilingBlower`, `kLeftFan`, `kRightFan`, `kSewerGrate`, `kInvisBlower`,
`kGrecoVent`, `kSewerBlower`, `kLiftArea`) does the work at `Objects.c:406-417`: it writes the
master and (if `room == thisRoomNumber`) the `thisRoom` copy, plays
`kBlowerOn`/`kBlowerOnPriority` or `kBlowerOff`/`kBlowerOffPriority`, and — guarded by
`if (masterObjects[local].hotNum != -1)` — sets `hotSpots[hotNum].isOn = newState`. That is the
entire on/off mechanism for every vent and fan in the game.
*Symptom:* believe the "inert" reading and no vent or fan in the house can ever be switched, and no
switch makes a sound.

**122. `HandleSwitches`' grease case does not call `SetObjectState`, so a switch-spilled jar is not
persisted.** Compare `Interactions.c:1057-1061` with `:891-897`, `RubberBands.c:124-127` and
`Triggers.c:114-119`.
*Symptom:* leave the room and come back and the jar is upright again. The only guard against a
switch re-spilling it is `SpillGrease`'s own `mode == kGreaseIdle` test plus the `stillOver`
debounce.

**123. Lighting changes interact badly with an in-progress spill, in two directions.**
`ReBackUpGrease`'s guard is `if ((mode == kGreaseIdle) || (mode == kGreaseFalling))`
(`Grease.c:189`), so a spill in the *spreading* stage is not re-backed-up and `savedMaps` keeps the
pre-lighting-change frames; the `redraw` arm of `ObjectDrawAll.c:373-374` does not repaint the
trail; and `RedrawRoomLighting` (`RoomGraphics.c:434-461`) does **not** call `RedrawAllGrease`.
*Symptom:* flipping the lights mid-spill leaves the trail visually erased while the `kSlideIt` rect
keeps growing.

**124. `RedrawRoomLighting` recomposes the central room from inside the hot-spot sweep.**
`Interactions.c:1083` calls it from `HandleSwitches` ← `HandleHotSpotCollision` ← the
`for (i = 0; i < nHotSpots; i++)` loop in `CheckForHotSpots`. It performs `DrawRoomBackground`,
`DrawARoomsObjects(central, true)` and `RestoreWorkMap()` — a whole-back-over-work blit —
mid-sweep, before `RenderFrame`. It is **not** a table-rebuild hazard, because `redraw = true`
suppresses every `Add*` and the `dynaNum` correlation loop (all guards at
`ObjectDrawAll.c:452`-`:953` include `!redraw`), so `nHotSpots` and `hotSpots[]` cannot change under
the loop. Its `RestoreWorkMap()` at `RoomGraphics.c:457` wipes every animation from Work, but only
`localRoomsDest[kCentralRoom]` is enqueued to the screen.
*Symptom:* neighbour-room flames stay erased on screen until their next even frame. And a porter who
reads `room-transitions`' blanket "no hot-spot action changes the room synchronously" — which is
otherwise **correct**: every `MoveRoomToRoom` in `Interactions.c` is inside a
`CheckEscape*`/`CheckRoofCollision`, reached from `CheckGliderInRoom` after the sweep, and there are
zero `ForceThisRoom`/`ReadyLevel` calls in that file — will skip this recomposition entirely.

**125. The four `ReBackUp*` functions can silently find nothing, and no `Add*` runs on the redraw
path.** They require both a `savedMaps` slot matching `(where, who)` **and** an element whose `who`
equals that slot index (`DynamicMaps.c:292-310` and siblings).
*Symptom:* a flame that failed to register when the room was composed can never appear later, and a
light switch can neither create nor destroy an animation — only refresh one.

**126. `AddStar` has no `h`/`v` guard, unlike all four of its siblings.**
`DynamicMaps.c:667-668`.
*Symptom:* a star whose composed rect is partly off the back map gets a strip whose uncovered cells
are never written by `BackUpStar`'s `CopyBits` (QuickDraw clips src and dest together), so it
animates over whatever a fresh GWorld contained. `SectRect` at `ObjectDrawAll.c:429` makes this hard
to reach but not impossible near the map edge.

**127. `RemoveShreds` uses strict `>` against `largest = 0`, and it fires on every death.**
`DynamicMaps.c:752`, `:756`, called from `Player.c:1489` for *all* deaths, not just shredder ones.
A shred still in its growth phase (`frame == 0`) is never selected, so `who` stays `-1` and
`numShredded` is not decremented. And `numShredded` is zeroed by any room change.
*Symptom:* unreachable today (68 countdown frames vs 54 animation frames) but a leak the moment
either constant moves; and in two-player, a surviving player who leaves the room silently cancels
the pending cleanup.

**128. `RenderShreds` plays `kShredSound` on all 35 growth frames.** `Render.c:585`. The sound
priority system is the rate limiter.
*Symptom:* a naive Go mixer machine-guns.

**129. `clockFrame` and `playedTikTok` are global, so all clocks in a room are locked together and
only one can be heard.** `playedTikTok` (`Render.c:265`) is set false *before* the
`numPendulums == 0` early return and allows exactly one sound per frame, won by the lowest array
index.
*Symptom:* with two out-of-phase clocks you hear one clock's tik/tok and never the other's.

**130. `kGliderInLimbo` freezes `dest` and keeps colliding.** `HandleGlider`'s
`case kGliderInLimbo: break;` (`Player.c:1423-1424`) moves nothing, so a limbo glider's rect sits
still and re-fires every overlapping hot spot every frame. The transit cases all exclude limbo
(`Interactions.c:1263`, `:1298`, `:1396`, `:1442`, `:1485`, `:1525`, `:1560`), but the hazard cases
do not: `kBurnIt`'s gate (`:1357-1358`) admits a limbo glider and `FlagGliderBurning` destroys the
limbo state.
*Symptom:* do **not** "tidy" this by adding a global limbo gate to the dispatcher.

**131. Most actions have no mode gate at all.** `kRewardIt`, `kLiftIt`, `kDropIt`, `kPushItLeft`,
`kPushItRight`, `kSwitchIt`, `kStrumIt`, `kTriggerIt`, `kSlideIt`, `kIgnoreLeftWall`,
`kIgnoreRightWall`, `kMicrowaveIt`, `kIgnoreGround`, `kBounceIt`, `kChimeIt` and `kSoundIt` never
look at `thisGlider->mode`. Only `kDissolveIt`, `kShredIt`, `kBurnIt`, `kMoveItUp`/`Down`,
`kTransportIt`, the mail cases, the duct cases and `kWebIt` do — and those exist precisely because
`HandleInteraction` is guarded only by `if (!gameOver)` (`Play.c:476-483`), not by the glider's
mode, and `CheckForHotSpots` tests `&theGlider` unconditionally.
*Symptom:* a glider that is fading out, shredding, ducting, mailing or in limbo still collects
prizes, flips switches, arms triggers, gets blown around and gets bounced. That is the original.

**132. `WebGlider`'s burning early-out is dead code, and its timer advances unconditionally at
double the twang rate.** `Interactions.c:1741-1747` tests `mode == kGliderBurning`, but the only
caller reaches `WebGlider` through `(GliderInRect(...)) && (mode != kGliderBurning)` at
`:1603-1604` and routes burning gliders to its own `else if` at `:1606-1612`. And `wasMode++` at
`:1769` is outside both branches of `:1754`, while the twang/recentre is gated on `evenFrame`
(`:1756`).
*Symptom:* the 150-frame death timer advances whether or not the player struggles, at twice the rate
of the visible twang. `wasMode` is also the burn countdown (`Modes.c:432`, `Player.c:220`) and the
limbo mode save (`Modes.c:460`), so a glider that was webbed and then burned inherits a corrupted
timer. Transcribe the dead early-out or drop it, but do not "fix" the caller to reach it. Note
`WebGlider` reads the global `evenFrame` at `:1756`, which means `Env` needs an `EvenFrame()`
accessor that no report lists.

**133. `StartGliderMailingIn` does not set `mode`; the dispatcher does, six times.**
`Modes.c:155-177` sets `frame` and `clip` only; `mode = kGliderMailInLeft/Right` is assigned
immediately after every call (`Interactions.c:1446`, `:1455`, `:1462`, `:1489`, `:1498`, `:1505`).
*Symptom:* omit the assignments and the glider stays in its previous mode. If the Go
`StartGliderMailingIn` ever grows a mode assignment, all six become redundant — decide once.

**134. `FinishGliderDuctingIn` hard-codes the duct's position as 7.**
`vNotClipped = thisGlider->dest.bottom - (kCeilingTransTop + 1);` (`Player.c:941`) with
`kCeilingTransTop = 6` (`GliderDefines.h:472`), so the emergence target is `dest.bottom >= 27`
regardless of where the duct actually is. The arrival starts at `dest.top = tempRect.top - 20` with
`tempRect.top = transRect.top - 2`.
*Symptom:* place a ceiling duct halfway down the wall and the glider takes the `else` branch on the
very first frame: it never animates, and it is 4 px lower than `ReadyGliderFromTransit` put it
because `:936-939` already ran. The function also `#define`s `kVDropStairsSpeed 4` at `:930` and
never uses it — the `+= 4` at `:937-938` is `kVDropDuctSpeed`, leaking in from
`MoveGliderDownDuct`'s `#define` at `:659`.

**135. `kLinkedToFloorDuct` is dead by design, and a floor-duct destination drops you through the
floor.** `WhatAreWeLinkedTo` has no `kFloorTrans` case (`Transit.c:41-58`), so `kLinkedToFloorDuct`
(4) is never produced and the arm at `:138-139` is unreachable. The reason is editor-side:
`UpdateLinkControl`'s `kTransportLinkOnly` list (`Link.c:162-186`) does not include `kFloorTrans`,
so you cannot author a link *to* a floor duct. Floor ducts are exits only.
*Symptom:* a hand-authored or converted house that does it anyway falls into
`default → kLinkedToOther` and the glider is centred in the floor duct's 56x15 rect in
`kGliderTransportingIn` mode; a floor transporter sits at `topLeft.v` around 297, so `dest.bottom`
lands past `kFloorLimit` (312), `CheckGliderInRoom` ignores it while the mode is
`kGliderTransportingIn` (`Interactions.c:691-694` lists only Normal/FaceLeft/FaceRight/Burning), and
`CheckEscapeDown` fires on the first frame after arrival. **Do not "fix" this by clamping.**

**136. `WipeScreenOn`'s clamp only touches `top` and `bottom`.** `Transitions.c:124-133`, never
`left`/`right`, so the horizontal wipes walk the band off the rect's edge and rely on `CopyBits`
clipping.
*Symptom:* a Go blitter that panics or wraps on an out-of-range source rect crashes on every
left/right room change.

**137. `DetermineRoomOpenings`' `kDirt` case tests two different tile values for the same tile, and
the inconsistency is load-bearing.** `Room.c:869-882`: `leftThresh` is chosen on `leftTile == 1`
(`:871`) but `leftOpen` on `leftTile != 0` (`:879`). `kMeadow` (`:883-894`) uses 6/7; five
backgrounds are fully open (`:896-905`); the tail (`:924-932`) is
`bottomOpen = !DoesRoomHaveFloor(); topOpen = !DoesRoomHaveCeiling();`.
*Symptom:* `tiles[0] == 0` in a dirt room yields `leftThresh = kNoLeftWallLimit` (-24, walkable off
frame) together with `leftOpen = false` — every other case in the function keeps the two
consistent. `leftThresh`/`rightThresh` are read by `CheckGliderInRoom` (`Interactions.c:725`,
`:738`); `leftOpen`/`rightOpen` by the editor and the `CheckEscape*` companions. Reproduce as
written.

**138. `SetObjectsToDefaults` is a hand-written case list, not a per-union-group sweep.**
`Play.c:603-708`: blowers `:622-633` (11 of 16 types, **omitting** `kTaper`, `kCandle`, `kStubby`,
`kTiki`, `kBBQ`), bonuses `:635-653` (14 of 15, omitting `kSlider`), `kDeluxeTrans` `:655-661`,
lights `:663-673`, `kStereo` `:675-677` (`state = isPlayMusicGame`), appliances `:679-690` (9,
omitting `kCinderBlock`, `kFlowerBox`, `kCDs`, `kCustomPict`), enemies `:692-702` (8, omitting
`kCobweb`). Furniture, switches and clutter do not appear at all.
*Symptom:* implement "for each object, state = initial by variant" and you overwrite bytes the C
leaves alone in any hand-edited house whose flame or slider `state != initial`. The omissions are
behaviourally inert — `Objects.c:420-425` gives the five flames
`changed = false; // Cannot switch on/off these`, and nothing consumes a slider's state — but they
are not byte-inert.

**139. Flame burn rects and fan blades are unconditionally on, so an "unlit" candle still burns and
can never be switched off.** `ObjectRects.c:410`, `:413`, `:416` and the four sibling flame cases
pass `isOn = true` **literally**, not `data.a.state`; `:374` and `:389` do the same for the fan
blade (13x43 at `topLeft+(16,12)` for `kLeftFan`, `+(6,12)` for `kRightFan`). Combined with
`Objects.c:420-425` and trap 138's omission of the five flames from `SetObjectsToDefaults`, there is
no path that turns a flame off.
*Symptom:* derive the burn rect's `isOn` from the object's state and unlit candles stop being
lethal. This is also the *simpler* cause of trap 62's fan symptom: the blade's `isOn` was never
state-derived in the first place.

**140. `kInvisTrans` and `kDeluxeTrans` hot spots are gated on `data.d.who != 255`, and
`kInvisTrans` overrides its own rect.** `ObjectRects.c:871-882` and `:884-896`, sharing the gate
with the mailboxes (`:752`, `:764`) and ducts (`:776`, `:788`). `kInvisTrans`:
`QSetRect(&bounds,0,0,64,32)`, offset by `topLeft`, **then**
`bounds.bottom = bounds.top + data.d.tall; bounds.right += (short)data.d.wide;`. `kDeluxeTrans`:
`wide = (data.d.tall & 0xFF00) >> 8; tall = data.d.tall & 0x00FF; QSetRect(&bounds,0,0,wide*4,
tall*4)`, offset by `topLeft`, `isOn = data.d.wide & 0x0F`.
*Symptom:* miss the gate and an unlinked transporter becomes a live hot spot that sends the glider
to room `-1`.

**141. `IsThisValid` decides whether a hot spot exists at all, and `kSparkle` is in its
state-gated list.** `Objects.c:89-122`: `itsGood = true`; `false` for `kObjectIsEmpty`;
`= (*thisHouse)->rooms[where].objects[who].data.c.state` for `kRedClock`, `kBlueClock`,
`kYellowClock`, `kCuckoo`, `kPaper`, `kBattery`, `kBands`, `kFoil`, `kInvisBonus`, `kStar`,
`kSparkle`, `kHelium`; no `default`, so every other kind is unconditionally valid.
*Symptom:* an already-collected prize produces **no** hot spot and `hotNum = -1`, which is why a
consumed sparkle stops registering. Grease jars are **not** in that list, so they always get one,
and `CreateActiveRects` branches on state to give either `kRewardIt` (upright,
`ObjectRects.c:688`) or `kSlideIt` (already spilled, `:696`). This closes
`live-state-inventory`'s openQuestion 7.

**142. A grease object's hot-spot action depends on its state, and the two representations can
disagree.** `ObjectRects.c:681-717` gives `kRewardIt` if `data.c.state` and `kSlideIt` otherwise,
while `AddGrease` is gated on the same state (`ObjectDrawAll.c:369`).
*Symptom:* a central-room jar that was already spilled at room-entry has a live `kSlideIt` rect and
**no** `grease[]` entry, so `RedrawAllGrease` will not find it — and if a neighbouring prize
pickup's `RestoreFromSavedMap` splats over its painted trail, nothing repaints it and the invisible
slick remains. This is also why trap 5's unguarded read is *accidentally* harmless in the original.

**143. `kSlideIt` and non-grease `kRewardIt` are invisible to bands, and the reward case does
almost nothing.** `CheckBandCollision:68-70` lists only `kDissolveIt`, `kRewardIt`, `kSwitchIt`,
`kTriggerIt`, `kBounceIt` — so a band flies straight through a grease slick and through a
`kSlider`. And `RubberBands.c:121-122` narrows the reward case to `kGreaseRt`/`kGreaseLf`, with
`hotSpots[i].isOn = false` (`:128`) inside that narrowing.
*Symptom:* a band that hits a clock's `kRewardIt` rect does nothing at all — not even turn it off —
but it *does* set `bandHitLast` (`:88`) and `nothingCollided = false` (`:85`), which suppresses the
next band.

**144. `bandHitLast` is one global for both in-flight bands, is never reset, and one band clears the
other's suppression.** `RubberBands.c:26`, read at `:86`, written at `:88` and `:146`. Not reset by
`KillAllBands` (`:307-317` touches only `mode` and `numBands`), not by `DrawLocale`
(`RoomGraphics.c:53` calls `KillAllBands` and nothing else), not by `NewGame`.
`CheckBandCollision` sets `nothingCollided = true` at `:42` and neither wall branch clears it.
*Symptom:* three live consequences. (a) BSS-zero means the very first band of a session that
overlaps hot spot 0 is silently ignored. (b) Band 0 hits hot spot 5 and sets `bandHitLast = 5`; band
1, processed after it in the same `HandleBands` loop, hits nothing and resets it to `-1` at `:146`,
so band 0 re-actuates hot spot 5 next frame — exactly the "rapid on/off toggle" the comment at `:87`
says they were preventing. (c) It suppresses only the *same* index, so one band grazing spot 3 then
spot 7 triggers both. Reproduce; do not make it per-band. Note only the `kDissolveIt`/`kBounceIt`
case has a `break` (`:116`), so one band can actuate several overlapping
`kRewardIt`/`kSwitchIt`/`kTriggerIt` spots in one frame.

**145. `sliding` has exactly two effects and neither is friction, and it is not a one-frame flag.**
Exhaustive grep: `Interactions.c:455`, `Player.c:155`/`:159`/`:177`/`:181`, `Interactions.c:1377`,
`Play.c:364`. Sprite selection plus `CheckRoofCollision` suppression (trap 17). It is cleared only
inside `MoveGliderNormal` (`Player.c:1339-1340` is the only path there).
*Symptom:* add drag, zero `hVel` or "make the controls slippery" and grease plays wrong. Clear it
alongside `ignoreLeft`/`ignoreRight`/`ignoreGround` at the end of `HandleGlider`
(`Player.c:1437-1439`) and a glider that enters a non-normal mode while sliding loses its roof
immunity, which the original keeps until it returns to normal mode. Note `kSlideIt` sets
`sliding = true` at `Interactions.c:1376-1379` and `CheckRoofCollision` reads
`!thisGlider->sliding` at `:455`, in that order within the frame.

**146. The 4-frame grease dead window.** From `SpillGrease` until the fourth fall frame the hot spot
is still `kRewardIt` with `isOn == false`, so the jar can neither be collected again nor be
slippery; only `Grease.c:60-61` flips `action` to `kSlideIt` and `isOn` back to true. And because
`HandleGrease` runs inside `RenderFrame`, the player is first caught the frame *after* that.
*Symptom:* collapse the window and a jar becomes slippery on the frame it is knocked over.

**147. `AddGrease`'s `BackUpToSavedMap` deliberately copies the wrong pixels, and leaves a bogus
`dest`.** It reads the `(0,0,32,108)` corner of `backSrcMap` — the top-left of the whole nine-room
composite, not the jar — and `BackupGrease` overwrites all of it immediately; the call exists only
to allocate a slot and a GWorld. `savedMaps[savedNum].dest` is left as `(0,0,32,108)`. Nothing ever
calls `RestoreFromSavedMap` for a grease jar (verified: `Interactions.c:1057-1061`, the only
grease-adjacent site, does not).
*Symptom:* do not "correct" the rect — `savedMaps[].where`/`.who` and the slot accounting are the
only outputs that matter. But a Go implementation that reuses a general `RestoreFromSavedMap` and
iterates all slots must not accidentally restore a grease slot.

**148. Animation rates are not uniform, and three movers' timing is parity-dependent with nothing
resetting the parity.** The copter's and shot-copter's frame animations are **not** `evenFrame`-gated
(`Dynamics2.c:142-144`, `:183-185`) while the balloon's and the toast's are (`:38-43`, `:77-82`,
`Dynamics.c:327-332`); the dart never animates. The copter has no horizontal bound at all
(`:191-192` tests only `dest.top`/`dest.bottom`), so copters drift off the side of the room and are
clipped by the work-rect clamp — do not add an `hVel` reversal. The drip is the only mover with
half-rate gravity: `length = 64` falls in 14 frames if `evenFrame` is false at the detach and 15 if
true. The fish's leap height alternates with parity (71 px every leap at `delay = 10`; 64/72 at
`delay = 11`), and nothing forces `evenFrame` at a leap — only at spawn.
*Symptom:* a fidelity test that asserts one apex passes on half the houses, and any ball or fish
spawning in the room perturbs the drip's fall duration (trap 149).

**149. Three places write the global `evenFrame = true` out of band, two of them during room
composition.** `Dynamics3.c:474` (`kBall` spawn) and `:524` (`kFish` spawn), both inside
`AddDynamicObject` inside `DrawLocale`; plus `Dynamics2.c:420`, a ball launching from idle inside
`HandleDynamics`. `evenFrame` is read by `HandleToast` (`Dynamics.c:327`),
`CheckDynamicCollision` (`:60`), `RenderFrame`'s flames-vs-stars choice (`Render.c:649`), and the
drip/fish/ball gravity gates (`Dynamics2.c:410`, `:472`, `:551`). It is initialised `false` once at
`InterfaceInit.c:131`, toggled only at `Play.c:435`, and never reset by `NewGame`.
*Symptom:* (a) entering any room containing a ball or a fish forces the parity, so that frame's
`RenderFrame` runs `RenderFlames()` instead of `RenderStars()` and every subsequent
parity-dependent behaviour in the room is phase-locked to it; (b) a ball launching from idle flips
the parity **mid-frame**, so every `dinahs` entry with a higher index sees the flipped value in the
same `HandleDynamics` pass — the drip's frame toggle, the fish's gravity and the foil decrement all
key off it; (c) the phase at the start of a game depends on how many frames the previous game ran.
Reproduce the writes literally; do not clean up the local `lilFrame` loop into something that leaves
the global alone. `docs/analysis/interactions.md` 22.12 attributes this only to `HandleBall` and
misses the two composition-time sites.

**150. `count`'s sign flips between the toaster and the ball/fish, and the two launch solvers are
different `do`/`while` loops.** Toaster: `count = +velocity` (`Dynamics3.c:233`), consumer negates
(`vVel = (short)-dinahs[who].count`, `Dynamics.c:375`; `Trip.c:159`), and the stop test
`if (dinahs[who].vVel > dinahs[who].count)` (`:357`) compares against the positive magnitude, which
is what makes the arc symmetric. Ball and fish: `count = -velocity` (`Dynamics3.c:486`, `:535`),
consumer assigns directly (`Dynamics2.c:389`, `:418`, `:538`). Solvers — full rate, toaster only
(`Dynamics3.c:226-231`): `do { velocity++; position -= velocity; } while (position > 0);`; half
rate, ball and fish (`:476-483`, `:526-533`):
`do { if (lilFrame) velocity++; lilFrame = !lilFrame; position -= velocity; } while (position > 0);`.
*Symptom:* normalise `count` to a magnitude and the ball and fish launch *downward*; normalise to
negative and the toast does. Being `do`/`while`, `height == 0` or `length == 0` still yields
velocity 1 — and `data.h.length == 0` **does** occur in shipped houses for the fish (observed range
0..261), so a `while`-loop transcription gives 0 and a dead object.

**151. The drip's swell keys off absolute `timer` values, not a counter, and its frame flip is only
safe because of where the two assignments sit.** `Dynamics2.c:481-486` is an
`if / else if / else if / else if` chain on `timer == 6`, `4`, `2`, not four `if`s; so `count < 7`
skips swell frames and `count == 0` (`delay == 0`) makes the drip detach every frame. And
`if (evenFrame) frame = 9 - frame;` (`:433-434`) is safe only because the drip enters `moving` with
`frame == 4` (`:492`) and leaves with `frame == 3` set in the same statement group as
`moving = false` (`:464-465`) — `dripSrc` has 6 entries, so a `frame == 3` reaching that line
indexes `dripSrc[6]`.
*Symptom:* separate those two assignments and you index out of bounds.

**152. The fish's idle bob is tested outside `if (active)` but keyed on a `timer` only an active
fish decrements.** `Dynamics2.c:558` tests `(timer & 0x0003) == 0x0003`; `:578`, inside
`if (active)` at `:576`, is the only decrement.
*Symptom:* an inactive fish's timer is frozen, so the bob either fires on **every** frame — a 1 px
vibration walking frames 0→1→2→3 with net zero displacement — or never fires at all, depending on
the frozen residue. At spawn `timer = delay*3`, so the vibrating case is `delay ≡ 1 (mod 4)`.
`docs/analysis/enemies.md` §11.6 asserts the opposite ("a switched-off fish still bobs").

**153. `RenderFish` is opaque while idle and masked while moving.** `Dynamics.c:278-280`
(`CopyBits`/`srcCopy`) vs `:267-270` (`CopyMask`). The idle 16x16 rect is stamped over the
background, white pixels included.
*Symptom:* alpha-blend the idle fish and it grows a halo the original does not have.

**154. `HandleDynamics` has no `active` check and no room filter; `RenderDynamics` deliberately
omits nine types.** `Dynamics3.c:35-37` dispatches every slot every frame, including appliances
belonging to the eight neighbour rooms. `RenderDynamics` (`:112-154`) has no case for `kSparkle` or
any of the seven appliances (`:150-151` `default`) — those draw from inside `HandleDynamics`, at a
different point in the frame, into a different surface (`backSrcMap` for six, `workSrcMap` for the
outlet).
*Symptom:* adding a filter to `HandleDynamics` changes RNG consumption and `evenFrame` parity;
adding render cases double-draws; moving the appliance drawing into the render phase breaks the
two-frame `timer == 1`/`timer == 0` handshake.

**155. `Dynamics2.c:396-397` is unreachable dead code.** `:385` sets `whole.bottom = position` and
`:386` sets `dest.bottom = position`, so `whole.bottom < dest.bottom` can never hold at `:396`.
*Symptom:* transcribe it if you want a line-for-line port, but do not let it convince you that
`whole` gets extended there.

---

### 4.I Refuted, or checked and not a trap — do not re-add these

Listed so that the next audit does not spend a pass re-deriving them. Each was asserted somewhere in
the raw reports, or is the obvious suspicion, and each is settled.

**156. `HandleTelephone` cannot divide by zero.** `live-state-inventory`'s T25 and its
openQuestion 1 both flag `delayTime = kChimeDelay / numChimes;` at `Play.c:780` as needing a guard.
The whole chime block is `if (numChimes > 0)` at `:771`, spanning `:772-788`, and the division is
inside it. What the report *should* have recorded instead: the clamp `if (delayTime < 2)
delayTime = 2;` (`:781-782`) and `theChimes.nextRing = RandomInt(delayTime) + 1;` (`:784`). Note
also that `:748`'s `if (!phoneBitSet)` phone block and the chime block are **sequential ifs, not
else-if**, so both can play a sound on one frame. (Same conclusion, with the clamp, in §1.1 F4 and §6.1 item 1.)

**157. A star consumes ONE `savedMaps` slot for its strip, not two.** `background-animations`
claims two, and derives an exhaustion threshold of 12 from it. `AddStar`
(`DynamicMaps.c:662-694`) calls `BackUpToSavedMap` exactly once, at `:680`, with
`QSetRect(&bounds, 0, 0, 32, 31 * 6)` — one slot holding a six-frame strip. The star's *total* is
still two slots (object rect + strip, trap 52) and the threshold is 24, not 12. (Consequences: §5.3 sub-stage 1.5f, and §6.2 item 2.)

**158. "Flames and stars never update twice in a row" is false.** See trap 24. Delete the sentence. Also §1.5 and §6.4 item 1.

**159. `numChimes` counts central-room chimes only.** `background-animations` says "across all
listed rooms". `ListOneRoomsObjects` calls `CreateActiveRects` only under
`if ((where == kCentralRoom) && (IsThisValid(roomNum, n)))` (`Objects.c:283-284`), and `numChimes++`
is inside `CreateActiveRects` (`ObjectRects.c:1052`). So the chime interval depends on the central
room's chimes alone.

**160. `kInvisObstacle` is a lethal solid, not an inert placeholder.** `hotspot-pipeline` lists it
under `kIgnoreIt` as an "`kInvisObstacle`-class placeholder". `ObjectRects.c:629` puts
`case kInvisObstacle:` in the furniture group whose single `AddActiveRect` at `:631` is
`AddActiveRect(&bounds, kDissolveIt, who, true, true)`, and `Objects.c:437` puts it in the same
family in `SetObjectState`. Every `kInvis*` type behaves like its visible sibling; none is a
placeholder. (`kIgnoreIt` is created for exactly two things: `kOutlet` with `isOn = data.g.state`
(`:959-966`), and the **seven** mobile enemies `kBalloon`, `kCopterLf`, `kCopterRt`, `kDartLf`,
`kDartRt`, `kBall`, `kDrip` sharing the rect at `:1013` — not six. `kFish` takes `kDissolveIt` at
`:1016-1022`.) (§3.3's furniture table is the authority for which types emit `kDissolveIt`.)

**161. There are five flame types, not six** — `kTaper`, `kCandle`, `kStubby`, `kTiki`, `kBBQ`.
`kBurnIt` is emitted at exactly `ObjectRects.c:413`/`:416`, `:438`/`:441`, `:463`/`:466`,
`:488`/`:491`, `:511`/`:514`.

**162. `CheckBandCollision`'s `collided` is uninitialised but never read as such, and both wall-phase
stores are dead.** `RubberBands.c:40`; the hot-spot loop assigns it only inside a triple-nested
guard, and the glider phase (`:150-159`) reassigns it unconditionally before its own
`if (collided)`. If `hVel == 0` the glider phase is skipped and it is never read at all. So the two
`collided = true` stores at `:51` and `:60` are dead in every reachable path.
`docs/analysis/interactions.md` says it is overwritten "on the first iteration of the hot-spot
loop", which is the wrong reason for the right conclusion.

**163. `ZeroDinahs` omits `byte1` and `moving`, harmlessly.** `Dynamics3.c:166-177`. All 15
committing cases of `AddDynamicObject` write `moving` (including `kBall`, out of sequence at `:485`
between `vVel` and `count`), nothing reads `byte1`, and `default: return -1` (`:546-548`) does not
increment `numDynamics`. Note `dinahs[].type` *is* written at `:196` before the switch, so the slot
past the end carries a stale type — the next successful `Add` overwrites it. If a Go port ever adds
a type that forgets to set `Moving`, the C-faithful reset will not save it.

**164. Miscellaneous absences of bugs, all grep-verified:** `CreateActiveRects`' `hotSpotNumber` **is**
initialised to `-1` (`ObjectRects.c:303`). `ListAllLocalObjects`' `localLink` loop has no `break`
but cannot see a duplicate key (trap 82). `RenderDynamics` has no case for `kSparkle` or the seven
appliances, and that is correct (trap 154). `doRollScore` (`Scoreboard.c:42`) is only ever set true,
so the `else displayedScore = theScore;` at `:88-89` is dead after the first `RefreshScoreboard`.
`RenderGlider`, `RenderToast`, `RenderBall` and `RenderDrip` are pure draws with no state mutation —
checked specifically because a second `HandleGrease`-class surprise would have been expensive, and
there is not one. `nLocalObj` is dead everywhere, not merely in the gameplay path: `Objects.c:76` is
its only occurrence in the tree, so do not port a field for it. `stopPages`/`pagesStuck` **do** have
an init, at `GameOver.c:308-309`. `wardBitSet` is **house** scope, not room scope: declared
`RoomGraphics.c:33`, cleared `House.c:142`, assigned from the house flags at `HouseIO.c:416`, read
at `RoomGraphics.c:195`.

**165. `objDataType` is 26 bytes, not 36.** Seven shorts (`roomNum`, `objectNum`, `roomLink`,
`objectLink`, `localLink`, `hotNum`, `dynaNum`) = 14, plus `objectType`, which
`GliderStructs.h:105` annotates "total = 12". 36 is `dynaType`'s size
(`GliderStructs.h:310-320`). Matters only because a wrong size invites a wrong byte-offset
derivation of the kind trap 1 depends on. (Also §6.8 item 2.)

---

## 5. Sub-stage decomposition

Stage 1.5 is one four-line bullet in `docs/PLAN.md` and it is the largest stage in the project by
a factor of three. This section splits it into six sub-stages in strict dependency order, sizes
each one, and says what each unblocks.

Cross-references below name the owning section by subject rather than by number, because the
numbering is only fixed when this document is assembled. Where a claim in this section came out of
an adversarial critique of the eight subsystem reports rather than out of a report, it is marked.

### 5.1 How big Stage 1.5 actually is

Two numbers, and they are different by a factor of two, which is why "Stage 1.5" has looked
deceptively small.

**Lines of C that must be *read* to write Stage 1.5: ~17,400.** That is the live-state inventory's
grand total across the twenty-five files that declare or touch the shared globals. It includes
`Player.c`, `Modes.c` and `Input.c`, all already ported in 1.4, and it includes ~1,900 lines of
editor code in `Link.c` and `Room.c` that Stage 1.5 does not port at all but does have to read to
know what the runtime half is.

**Lines of C that Stage 1.5 *ports*: ~9,100.** Derived by adding up the function ranges each
sub-stage owns, after subtracting what 1.3 and 1.4 already did:

| Area | Files and ranges | C lines |
|---|---|---|
| Object graph and hot-spot construction | `Objects.c:89-122`, `:126-250`, `:254-348`, `:351-362`, `:366-699`; `ObjectRects.c:277-292`, `:296-1063` | 1,384 |
| Room addressing and openness predicates | `Room.c` runtime half (`:343-435`, `:562-635`, `:711-1205`) | 430 |
| Room-load spine and the reset stubs | `RoomGraphics.c:44-60`, `:389-461`; `DynamicMaps.c:46-163`, `:787-797`; `Dynamics3.c:160-180`; `RubberBands.c:307-317`; `Triggers.c:198-204`; `Render.c:765-770`; `Play.c:307-366`, `:603-708` | 420 |
| Frame loop, event pump, game-over tail | `Play.c:74-278`, `:387-425`, `:430-554`, `:744-796`, `:800-820` | 455 |
| Render spine, dirty rects, glider compositing | `Render.c` (whole file minus the six animation passes and `RenderBands`) | 379 |
| Hot-spot dispatcher and the sweep | `Interactions.c:1198-1777` | 568 |
| Room transitions | `Transit.c` (all), `Transitions.c` (all), `Link.c:34-53`, `Player.c:1443-1604`, `GameOver.c:237-242` | 915 |
| Scoreboard's per-frame half | `Scoreboard.c:53-132`, `:268-299`, `:338-408` | 190 |
| Dynamics: table, appliances, movers | `Dynamics3.c:31-154`, `:187-554`; `Dynamics.c` (all); `Dynamics2.c` (all) | 1,899 |
| Toggles and triggers | `Trip.c` (all), `Triggers.c:12-194` | 443 |
| Sparkles and flying points | `DynamicMaps.c:169-254`; `Render.c:325-416` | 170 |
| Rewards and switches | `Interactions.c:756-1194`; `Banner.c:205-236` | 465 |
| Bands and grease | `RubberBands.c:12-303`, `Grease.c` (all), `Render.c:534-556` | 630 |
| Background animations and the saved-map economy | `DynamicMaps.c:263-778`; `Render.c:193-321`, `:420-448`, `:559-612` | 797 |
| | **total** | **9,145** |

For calibration: Stage 1.4 ported ~3,200 lines of C (`Player.c` less its tail, `Modes.c`, `Input.c`,
`Interactions.c:54-752`) into 3,140 lines of Go plus 1,533 of tests. At the same ratio Stage 1.5 is
about 9,000 lines of Go and 4,000 of tests — roughly twice everything committed so far.

Two things the table deliberately does **not** charge to Stage 1.5, because 1.3 already ported
them and the registration work hangs off the existing code: `ObjectRects.c`'s `GetObjectRect`
(`:120-268`), `VerticalRoomOffset`/`OffsetRectRoomRelative` (`:1067-1131`) and the 117-case draw
switch of `DrawARoomsObjects` (`ObjectDrawAll.c:23-965`) are in `internal/render` today
(`objectrect.go`, `view.go`, `locale.go`). Stage 1.5 adds the registration calls *into* that sweep
— `BackUpToSavedMap`, `AddCandleFlame`, `AddPendulum`, `AddStar`, `AddGrease`, `AddDynamicObject`,
the `dynamicNum` write and the `dynaNum` correlation pass at `ObjectDrawAll.c:953-961` — rather
than re-transcribing it. That is 1.3's largest gift to 1.5 and it is why the object-graph row above
is 1,384 lines and not 2,350.

### 5.2 The four things that must be settled before 1.5a starts

All three critiques converge on the same four unresolved decisions, and each one has an owner in
the decomposition below rather than being left floating:

1. **The build configuration.** `COMPILEQT` and `BUILD_ARCADE_VERSION` are both defined
   (`GliderDefines.h:15-16`), which between them add four `RenderFrame` call sites, two
   `MoviesTask` calls and the whole `Play.c:512-541` scoreboard-blackout path to the frame. Owned
   by **1.5b**, which cannot write the frame loop without it.
2. **The double `RenderFrame`.** `RenderFrame` is not idempotent and is called twice on any frame a
   transition completes — once from `Transit.c:303`/`:341`/`:380`/`:419` (unconditional under
   `COMPILEQT`) and once from `Play.c:469`/`:494`. Because `evenFrame` flips only at `Play.c:435`,
   the second call takes the same branch of `Render.c:649-652` as the first. Owned by **1.5b**;
   its acceptance criterion pins it per frame. (Critique 1, C1/G1. It also settles the flat
   contradiction between the animations report, which claims no family ever advances twice in a
   row, and the transitions report, which has the fact and does not follow it through: the C sides
   with the transitions report.)
3. **The pending-transit triple.** `transRect` (`Player.c:49`), `transRoom` (`Player.c:52`) and
   `linkedToWhat` (`Transit.c:17`) are single globals read by both gliders, and `FollowTheLeader`
   plus `OffAMortal`'s `mortals == -1` re-dispatch read the *dead* leader's destination. They must
   live on `World`, with 1.4's per-glider `g.Transit` (`internal/game/player/glider.go:93`) demoted
   to the shadow copy `StartGliderMailingOut` legitimately reads. Owned by **1.5b**. The same
   decision has to be taken for `rightClip`/`leftClip` (`Player.c:52`), which 1.4 also moved
   per-glider (`glider.go:87`).
4. **The line-number convention in the source reports.** One report carries `RoomGraphics.c`
   citations that are uniformly one low next to `Grease.c` citations that are exact, and two of its
   anchors name the wrong call (`KillAllBands` where the C has `ZeroDinahs`, `ZeroTriggers` where
   the C has `ZeroMirrorRegion`). Correct order for `DrawLocale`'s reset head, from
   `RoomGraphics.c:51-60`: `ZeroFlamesAndTheLike` `:51`, `ZeroDinahs` `:52`, `KillAllBands` `:53`,
   `ZeroMirrorRegion` `:54`, `ZeroTriggers` `:55`. Owned by **1.5a**, which is the sub-stage that
   transcribes them.

### 5.3 The sub-stages

Paste-ready for `docs/PLAN.md`, replacing the current single 1.5 bullet.

---

**1.5 Objects, collision, room transitions** — ~9,100 lines of C, six sub-stages. This is the
largest stage in the project; the decomposition and its dependency argument are in
`docs/analysis/stage-15-spec.md` §5.

**1.5a The world and the room: live state, object graph, hot-spot table, room load** — ~2,230
lines of C

- `internal/game` gains `World` (session-, game- and house-scope state) and `Room` (the state
  `DrawLocale` rebuilds), scoped on the C's own reset sites rather than on intuition:
  session = `CreatePointers` (`StructuresInit2.c:187-299`), game = `Play.c:112-118` + `InitGlider`,
  room = `DrawLocale` (`RoomGraphics.c:51-60`). `Room` embeds `*render.Scene`.
- Ports `Objects.c`'s `IsThisValid`, the four link resolvers (`GetRoomLinked`, `GetObjectLinked`,
  `ObjectIsLinkTransport`, `ObjectIsLinkSwitch`), `ListOneRoomsObjects`, `ListAllLocalObjects`,
  `AddTempManholeRect` and `SetObjectState` (`:89-699`, 600 lines); `ObjectRects.c`'s
  `AddActiveRect` and all 117 cases of `CreateActiveRects` (`:277-1063`, 784 lines);
  `Room.c`'s runtime half — `ForceThisRoom`, `RoomExists`, `GetNeighborRoomNumber`,
  `IsRoomAStructure`, `DetermineRoomOpenings`, `GetNumberOfLights`, `IsShadowVisible`,
  `DoesRoomHaveFloor`/`Ceiling` (430 lines); `RoomGraphics.c`'s `ReadyLevel`, `RestoreWorkMap`,
  `RedrawRoomLighting` and `DrawLocale`'s reset head; the four `savedMaps` primitives
  (`DynamicMaps.c:46-163`); the five reset stubs `DrawLocale` calls; and `Play.c`'s
  `SetObjectsToDefaults` and `InitGlider`.
- `CreateActiveRects` is the single largest unwritten piece of the spec and gets a section of its
  own: 93 of the 117 types produce at least one hot spot, 24 produce none, and the union of the
  eight source reports specified the 28 *actions* without ever saying which *types* emit them.
  Fifteen of the undocumented types are lethal `kDissolveIt` solids — twelve furniture kinds
  including `kTable`, `kCounter` and `kInvisObstacle`. A port that omits them ships a house full
  of pass-through tables. (Critique 3's structural gap.) **Closed during assembly:** §3.2-§3.10 of
  this document now carries the per-type table — code, action, `isOn`, `doScrutinize` and the rect
  derivation for all 117 types, with `ObjectRects.c` line citations — so 1.5a transcribes from §3
  and not from the raw reports.
- **Unblocks everything.** No other sub-stage has anything to iterate over until `hotSpots[]` and
  `masterObjects[]` exist, and `Rebuild` — the `DrawLocale`-scope reset, which must be a method on
  `World` and not a constructor, because `RestoreEntireGameScreen` (`Play.c:800-820`) calls
  `DrawLocale` at `:814` *without* `NilSavedMaps` or `InitGarbageRects` — is the seam every later
  sub-stage registers into.
- *Acceptance:* **all 4,070 rooms of all 22 houses build a `masterObjects`/`hotSpots` pair**, each
  pinned by a per-room hash over `numMasterObjects`, `numLocalMasterObjects`, `nHotSpots`, every
  master entry's `(roomNum, objectNum, roomLink, objectLink, localLink, hotNum, dynaNum)` and every
  hot spot's `(bounds, action, who, isOn, doScrutinize)`. The type→action table is proved covered:
  each of the 93 hot-spot-producing types is exercised by at least one shipped room, or appears in
  a checked-in list of types the corpus never instantiates. `DetermineRoomOpenings` and
  `GetNumberOfLights` are asserted against hand-derived values for one named room per house (22
  assertions each). `SetObjectState` is driven with all four of `kToggle`/`kForceOn`/`kForceOff`/
  `kOneShot` against all 117 `what` codes without panicking, which is what pins the three
  uninitialised-`changed` returns and the missing `room`/`object` bounds check as recorded
  deviations rather than crashes.

**1.5b The frame and the traversal: frame loop, hot-spot dispatcher, room transitions** — ~2,510
lines of C

- Ports `Play.c`'s `PlayGame` loop with all twelve steps and the `gameOver` tail (`:430-554`),
  `HandlePlayEvent` and the `doBackground` pump (`:387-425`), `NewGame`'s reset block, the
  telephone and chimes (`:744-796`), `RestoreEntireGameScreen`; `Render.c`'s frame spine —
  `RenderFrame`, `CopyRectsQD`, the two dirty-rect adders with their two *different* clamp
  rectangles, `RenderGlider`, `DrawReflection`, `InitGarbageRects`, the five `CopyRect*` helpers
  and the mirror region; `Interactions.c:1198-1777` — `HandleHotSpotCollision`'s 28 cases,
  `CheckForHotSpots`, `HandleInteraction`, `FlagStillOvers`, `WebGlider`; all of `Transit.c` and
  `Transitions.c`; `Player.c:1443-1604` (`OffsetGlider`, `OffAMortal`); and `Scoreboard.c`'s
  per-frame half.
- Land it in two commits, in this order: (i) the frame loop, the dirty-rect protocol, `RenderGlider`
  and the dispatcher's movement actions (`kLiftIt`, `kDropIt`, `kPushIt*`, the four `kIgnore*Wall`/
  `kIgnoreGround`, `kBounceIt`, `kMoveItUp`/`Down`, `kTransportIt`, `kMailIt*`, `kDuctIt*`) plus
  `Transit.c` — which is the point the game becomes playable; then (ii) `kDissolveIt`, `kBurnIt`,
  `kShredIt`, `kChimeIt`, `kWebIt`, `kSoundIt`, `kMicrowaveIt`, the game-over tail and the
  scoreboard. `kRewardIt`, `kSwitchIt` and `kTriggerIt` stay stubbed until 1.5d.
- **This is the sub-stage that makes the game playable**, because it is the one that finishes
  `internal/game/player`'s `Env` (46 methods, `env.go`) with real implementations. Everything after
  it can be tested by hand as well as headlessly.
- Owns the three settlements listed in §5.2 items 1-3: the build configuration, the double
  `RenderFrame`, and hoisting `transRect`/`transRoom`/`linkedToWhat` (and `rightClip`/`leftClip`)
  out of the glider onto `World`.
- *Acceptance:* `*game.World` implements `player.Env` in full and `NopEnv` no longer appears
  outside `_test.go` files. **A scripted input trace leaves Demo House's first room by each of the
  seven exit kinds** — left door, right door, up staircase, down staircase, transporter, ceiling
  duct, mailbox — as seven replays, each asserting the destination room number, the arrival mode
  and the arrival rect against numbers hand-derived from the C before the test was run. Every one
  of the 22 houses' start room is entered and one exit taken without a panic. A 600-frame headless
  replay pins, per frame, `gameFrame`, `evenFrame`, `numWork2Main`, `numBack2Work` and
  `clockFrame`, which is what pins the double `RenderFrame` on a transition frame (`clockFrame`
  advances twice and the frame limiter waits twice) and the `sameRoom` transport that skips
  `ReadyLevel` while still rendering.

**1.5c Dynamics: the `dinahs` table, the appliances, the movers, the toggles and the triggers** —
~2,510 lines of C

- Ports `Dynamics3.c`'s `AddDynamicObject` (17 registration cases, 368 lines), `HandleDynamics` and
  `RenderDynamics`; all of `Dynamics.c` — `CheckDynamicCollision`, `DidBandHitDynamic`, the nine
  appliance handlers and the seven `Render*` mover draws; all of `Dynamics2.c` — the six mover
  handlers; all of `Trip.c` — fourteen `Toggle*`, eight `Trigger*`, `UpdateOutletsLighting`; all of
  `Triggers.c` — `ArmTrigger`, `FindEmptyTriggerSlot`, `HandleTriggers`, `FireTrigger`; and the
  sparkle/flying-point pair (`DynamicMaps.c:169-254`, `Render.c:325-416`).
- Placed before rewards and switches on purpose, and it is the one place the dependency graph has a
  genuine cycle to cut. `HandleSwitches`' `linkIndex` dispatch calls into 21 `Toggle*`/`Trigger*`
  cases; `Trip.c`'s `TriggerSwitch` (`:146-149`) calls back into `HandleSwitches`. Doing dynamics
  first leaves exactly one stub (`TriggerSwitch`); doing switches first would leave 21.
- `CheckDynamicCollision` (`Dynamics.c:34-74`) is a second, earlier collision channel and must not
  share an implementation with `kDissolveIt`: different mode gate, different foil cost (one sheet
  per two frames under `(evenFrame) && (foilTotal > 0)` versus two sheets every frame), different
  effect. Note also that six of the nine appliance handlers use the `timer > 0` skeleton and three
  do not (`HandleSparkleObject`, `HandleToast`, `HandleOutlet`) — one report says eight and then
  says seven in the same paragraph; the C says six. (Critique 3, C4.)
- **Unblocks** rewards and switches (which toggle appliances), the `dinahs` slot the trigger table
  addresses, and the enemies that make a house dangerous.
- *Acceptance:* each of the 17 registrable types has a test pinning its post-`AddDynamicObject`
  slot fields and one frame of its handler against hand-derived numbers; `RenderDynamics`'
  deliberate omission of `kSparkle` and all seven appliances is pinned by a test that would fail on
  a double-draw. All fourteen `Toggle*` and all eight `Trigger*` have a test, and the seven target
  types `FireTrigger` silently ignores (`kMacPlus`, `kTV`, `kVCR`, `kStereo`, `kMicrowave`,
  `kBall`, `kShredder`) are pinned as no-ops. Trigger timing is asserted at three delays: `delay`
  0 fires on the arming frame, 1 fires two frames later, 7 fires twenty frames later. The
  `kMaxDynamicObs` 18-slot cap is asserted on the busiest room in the corpus, identified by a
  census over all 4,070 rooms that is checked in beside the test — which also settles whether the
  saturation is reachable in shipped data at all, a point on which three of the eight source
  reports disagree.

**1.5d Rewards, switches, and per-room persistence** — ~465 lines of C

- Ports `Interactions.c:756-1194` — `HandleRewards` (fifteen prize cases), `HandleSwitches` (the
  `SetObjectState` call, the six switch-art cases, the 21-case `linkIndex` dispatch) and
  `HandleMicrowaveAction`; plus `Banner.c`'s `DisplayStarsRemaining`. Unstubs `kRewardIt`,
  `kSwitchIt`, `kTriggerIt` from 1.5b and `TriggerSwitch` from 1.5c.
- Small in lines and large in surface: this is where per-room persistence becomes observable,
  because every prize and every switch writes a `state` byte back into the house handle through
  `SetObjectState` and that write is what survives leaving and re-entering a room. It is also where
  the fidelity traps cluster — `who->isOn = false` sitting *outside* the `if` in every prize case,
  `who->stillOver = true` sitting outside the `if` in `HandleSwitches`, the `if (linkIndex != -1)`
  being a separate nested `if` and not an `&&`, and the blower family of `SetObjectState` playing
  `kBlowerOn`/`kBlowerOff` (`Objects.c:411-414`, the only sound in `Objects.c` and in none of the
  eight source reports — critique 3, C5).
- **Unblocks** finishing a house at all: the last star fires `FlagGameOver`, and a grease jar
  prize is the only producer of grease.
- *Acceptance:* **each of the fifteen `HandleRewards` cases and each of the 21 `HandleSwitches`
  `linkIndex` cases has a test pinning its trigger condition and its effect** — the score delta,
  the inventory delta, the sound played, and the `state` byte written back into the house — which
  is the per-object-class criterion the original 1.5 bullet asked for. Taking the last star in a
  house sets `gameOver` within one frame, and `countDown` runs 16 further frames before dispatch.
  Leaving a room after collecting a prize and re-entering it finds the prize gone, for one prize
  of each of the twelve state-gated types. `kKnifeSwitch` — the one `what` code in `0x01..0x8F`
  that `SetObjectState`'s outer switch has no case for, verified by a mechanical diff of all 116
  case labels against all 117 `#define`s — has a test pinning the uninitialised return as a
  deviation.

**1.5e Bands and grease** — ~630 lines of C

- Ports all of `RubberBands.c` — `CheckBandCollision`'s five phases in order, `HandleBands`,
  `AddBand`, `KillBand`; all of `Grease.c` — `HandleGrease`, `BackupGrease`, `ReBackUpGrease`,
  `AddGrease`, `SpillGrease`, `RedrawAllGrease`; and `Render.c`'s `RenderBands`.
- Last of the three writers into `hotSpots[]`, and the one that makes the table mutable
  mid-frame: `HandleGrease` runs *inside* `RenderFrame` (`Render.c:647`) and rewrites
  `hotSpots[].action`, `.isOn` and `.bounds` after `HandleInteraction` has already swept, so a
  slide rect created or widened on frame N is not collidable until N+1. `CheckBandCollision` is
  the second consumer, filtering to five of the 28 actions and re-entering `HandleSwitches` and
  `ArmTrigger` after the sweep. Both depend on 1.5d existing.
- **Unblocks** nothing further; it is the last gameplay sub-stage.
- *Acceptance:* a band fired into each of the five actions `CheckBandCollision` filters for
  (`kDissolveIt`, `kRewardIt`, `kSwitchIt`, `kTriggerIt`, `kBounceIt`) produces the C's effect and
  no other; the `bandHitLast` debounce, the two-band cap, the wall bounce and the floor kill each
  have a test, including the ordering assertion that a band clamped to `kLeftWallLimit` by phase 1
  **survives** phase 5. A grease jar is stepped frame by frame through all four modes with the
  `kSlideIt` rect's `bounds.left`/`bounds.right` pinned per frame, and the frame on which it first
  becomes collidable is asserted to be one later than the frame it lands.

**1.5f Background animations and the saved-map economy** — ~800 lines of C

- Ports the rest of `DynamicMaps.c` — the `BackUp`/`ReBackUp`/`Add` triples for candle flames,
  tiki flames, BBQ coals, pendulums and stars, `StopPendulum`, `StopStar`,
  `AddAShreddedGlider`, `RemoveShreds`, `ZeroFlamesAndTheLike` — and `Render.c`'s `RenderFlames`,
  `RenderPendulums`, `RenderStars` and `RenderShreds`.
- Deliberately last, and it is the only sub-stage that is purely cosmetic — with one exception
  that makes it worth doing carefully rather than quickly. The five room-load `RandomInt` draws
  (`DynamicMaps.c:333-338`, `:417-422`, `:503-508`, `:580-594`, `:680-685`) are each *inside*
  `if (savedNum != -1)`, so when the 24-slot `savedMaps` table saturates the draw does not happen
  and the entire downstream RNG stream shifts. Registration is also screen-size dependent through
  `SectRect`, and `RestoreEntireGameScreen` leaks slots. The RNG stream is therefore a function of
  resolution and of whether the player pressed a key at the stars banner. (Critique 2's most
  determinism-critical omission; two source reports each hold half of it.)
- Correct the two size claims the source reports get wrong before writing this: a star consumes
  **one** `savedMaps` slot (a 32 × 186 six-frame strip, `DynamicMaps.c:679-680`), not two, so
  exhaustion bites above 24 registrations and not above 12; and `numGrease` is zeroed at
  `DynamicMaps.c:793`, `numChimes` at `:796`.
- *Acceptance:* each of the six animated families has a test stepping a full wrap cycle with the
  strip index and `src` rect pinned per frame — candles 15 px stride over 5 frames, tikis 10 over
  5, coals 9 over 4, pendulums 28 px per step on the `clockFrame == 10 || clockFrame == 15`
  schedule, stars 31 over 6, shreds their own strip — and the pendulum's `kTikSound`/`kTokSound`
  alternation is asserted to fire at most once per frame across all pendulums in a room.
  `savedMaps` saturation is asserted on the room a checked-in census over all 4,070 rooms names as
  needing more than 24 slots, with the dropped registrations enumerated. **Two headless replays of
  1,200 frames from the same seed produce byte-identical index planes**, which is what proves the
  conditional-on-a-free-slot `RandomInt` order is reproduced.

---

### 5.4 Build 1.5a first — and why the tempting alternatives are wrong

Strict dependency order already puts 1.5a first; the question worth answering is why that is also
the right answer once "something playable early" is a goal, since 1.5a puts nothing on screen.

**Build 1.5a first.** Four reasons, in order of weight.

1. **It is the only sub-stage whose acceptance criterion can be *proved* rather than sampled.**
   1.5a is a pure function of the house data: given a room, build `masterObjects[]` and
   `hotSpots[]`. That runs over all 4,070 rooms of all 22 houses in seconds, exactly as 1.2's
   loader and 1.3's compositor do, and it pins every table entry by hash. Every later sub-stage's
   criterion is a per-behaviour or per-trace assertion over a handful of hand-derived cases,
   because behaviour over time cannot be exhausted. If the table is wrong, every one of those later
   assertions is measuring the wrong thing, and 1.5a is the last point at which corpus-scale proof
   is available. This is the same argument that put 1.2 before 1.3 and it held.
2. **The largest unwritten piece of spec lives here, so this is where the risk is.**
   `CreateActiveRects` is 768 lines and 79 `AddActiveRect` sites, and all four source reports that
   touch it list it as *supporting* rather than primary. The consequence is that 24 types with live
   hot spots — twelve furniture kinds, seven vents and blowers, three static appliances,
   `kDoorExRt`, `kWindowExRt` — are specified nowhere in 680 KB of reports, and fifteen of them are
   lethal solids. §3.2-§3.10 of this document fills that gap, so the spec risk is retired and only
   the transcription risk remains. That is not a documentation nicety: a porter who builds the frame loop first and
   comes back to the table later will spend the intervening weeks playing a house whose tables are
   silently wrong and whose bugs all *look* like frame-order bugs.
3. **It is testable with no clock at all.** No frame loop, no timing, no input, no rendering. Every
   ordering question in Stage 1.5 — the double `RenderFrame`, the accumulation window between
   `HandleDynamics` and `MoveGlider`, whether `HandleGrease` runs before or after the sweep — is a
   *frame-order* question, and none of them can be reasoned about until the thing the frame
   iterates over is fixed and pinned. Settling the table first turns those into one-variable
   problems.
4. **`Rebuild` is a design decision, not an implementation detail, and getting it wrong is
   expensive.** `RestoreEntireGameScreen` (`Play.c:800-820`) reaches `DrawLocale` at `:814` from
   the stars-remaining banner *during normal play*, without `NilSavedMaps` and without
   `InitGarbageRects`. So the room rebuild has to be a method on `World` that reconstructs the
   object graph and the `DrawLocale`-zeroed tables while leaving `savedMaps`, sparkles, flying
   points and the dirty-rect queues alone — which means the two field sets must be enumerated in
   1.5a, before any code depends on a constructor-shaped `Room`. Discovering this in 1.5f, where
   the saved-map leak becomes visible, would mean re-plumbing every sub-stage in between.

**Then build 1.5b immediately, and treat 1.5a+1.5b as one push.** Together they are ~4,700 lines of
C, a bit over half the stage, and they are the point at which a glider walks through a house — the
earliest state in which every remaining sub-stage can be checked by hand as well as headlessly, and
the earliest state in which the project has something to show. 1.5b is also the sub-stage that
retires `NopEnv`, so it is the moment the 1.4/1.5 seam stops being a fiction.

**The two tempting alternatives, and why to refuse them.**

*Starting with 1.5c (dynamics) or 1.5f (animations) because they look self-contained.* They are the
two sub-stages with the cleanest file boundaries — `Dynamics*.c` and `DynamicMaps.c` are almost
entirely one subsystem each — and that is exactly the trap. Both register into tables 1.5a owns
(`dinahs[]` is indexed by `DrawLocale`'s draw order; `savedMaps[]` slot order decides which
animations exist at all), and both are dispatched from a frame loop 1.5b owns. Built first, each
would need a scaffold that mimics the registration order and the frame order, which is precisely
the work that then has to be thrown away — and a scaffold that guesses the draw order wrong would
produce tests that pass against the wrong slot indices and then fail mysteriously when the real
`DrawLocale` arrives. Neither is playable in any sense: an animated candle in a room the glider
cannot enter is a screenshot, not a game.

*Starting with 1.5b (the frame loop) because that is where "playable" lives.* Superficially the
strongest alternative, and it fails on the acceptance criterion. 1.5b's criterion is "leave the
first room by each of the seven exit kinds", and every one of those seven depends on a hot spot
that `CreateActiveRects` builds — `kMoveItUp`, `kTransportIt`, `kMailItLeft`, `kDuctItDown` and the
four `kIgnore*Wall` rects. Without 1.5a there is nothing to sweep, so 1.5b would have to be
accepted against a hand-written table, which is a fixture that proves the dispatcher matches the
fixture. Ordering it second costs one sub-stage of delay and buys a criterion that means something.

### 5.5 What Stage 1.5 does not cover

Recorded so the next audit does not have to re-derive the negatives.

- **Sound is 1.6.** Stage 1.5 records every `PlayPrioritySound` call site with its sound and
  priority constants and routes them through `Env`, and nothing more. The one exception worth
  flagging is `kSoundTrigger`, whose hot spot is created only when `LoadTriggerSound(...) == noErr`
  (`ObjectRects.c:923-928`) — so in 1.5 the rect must be created on the assumption the load
  succeeds, and 1.6 makes that real.
- **QuickTime movies are stubbed.** `MoviesTask` and the four `StartMovie`/`StopMovie` sites are
  no-ops with a comment; 1.1 already extracted all 15 movies as flat 8-bit index buffers, so this
  is a scheduling decision and not a data problem. `tvOn` still has to be modelled, because it has
  no reset point anywhere in the C (written at `ObjectDrawAll.c:693` and by `ToggleTV`, and
  `DrawLocale` clears `tvInRoom` and `tvWithMovieNumber` but not `tvOn`), so it is stale from the
  previous room in any room without a TV.
- **`tempManholes` is render-side**, consumed only by `RoomGraphics.c:277-371`'s six
  floor-support variants, and belongs to 1.3. Stage 1.5 ports `AddTempManholeRect` and
  `numTempManholes = 0` because `DrawLocale` and `ObjectDrawAll.c:277` do, and nothing else.
- **Demo playback** (`demoIndex`, `demoGoing`, `GetDemoInput`, `demoData`) sits on the same line of
  the frame loop as `GetInput` and is **out of scope for 1.5**, stated here so the omission is a
  decision rather than an oversight. It returns with the shell in 1.7, which is what starts it.
- **The editor halves** of `Link.c` (`:57-395`) and `Room.c` (room create/delete) and
  `Objects.c`'s `GetObjectState` (`:703-871`) are stage 5. `UpdateLinkControl`'s
  `kTransportLinkOnly` case (`Link.c:162-186`) is read in 1.5a for one reason only: it is the
  authoritative list of legal link targets, and it is what makes `kLinkedToFloorDuct` dead code.
- **`nLocalObj` gets no field.** `Objects.c:76` is its only occurrence in the tree; it is dead
  everywhere, not merely in the gameplay path. (Critique 2, C17.)

---

## 6. Open questions

This section is the residue: everything the eight subsystem passes and the three adversarial
critiques left unsettled, plus the batch of corrections they found in the already-written
`docs/analysis/*.md` corpus. Nothing here is a restatement of the specification proper —
where an item depends on material that is specified earlier, it names the owning topic (the
frame loop, the hot-spot pipeline, the dynamics table, room transitions, the live-state
inventory) rather than a section number, because §6 was drafted in parallel with the sections
above it and a wrong `§n` would be worse than a named topic.

Every item is one of four kinds, and the kind decides who can close it:

- **Settled here** (§6.1) — the C or the shipped corpus answers it, and this pass did the
  reading or ran the census. Do not re-open these; they are cited.
- **A data question** (§6.2) — the mechanism is certain from the C and only a corpus census
  says whether it fires. Each carries the exact predicate to count.
- **A C read nobody has done** (§6.3) — a specific file and line range, and what it decides.
- **A decision, not a discovery** (§6.4) — the C is understood and the port has to choose.
  Each carries the choice and the test that should pin it.

§6.5 lists the handful of things that are permanently unverifiable from the source. §6.6 is
the correction batch against `docs/analysis/*.md`, §6.7 the same against comments in
already-committed Go, and §6.8 says what to do with the eight raw reports themselves.

A census was run during this pass over `assets/extracted/houses/*.house` (22 files) reading
the binary directly: house header 866 bytes, rooms of 348, `floor` at room offset 52, `suite`
at 54, `objects[24]` at 60, each object `what` at +0 and its 10 data bytes at +2. Switch and
trigger links were resolved the way `GetRoomLinked` does it (`Objects.c:148-161`): decode
`data.e.where` with `ExtractFloorSuite` for `version >= 0x0200` — `suite = combo / 100`,
`floor = combo % 100 - kNumUndergroundFloors(8)` (`Link.c:49-51`, `GliderDefines.h:535`) —
then look the pair up in the house's floor/suite index. Totals, for scale: 1,563 switch-family
link fields, of which 1,253 resolve same-room, 278 resolve cross-room, 21 name a floor/suite
that does not exist in the house (so `GetRoomLinked` returns -1) and 11 are unlinked
(`data.e.who == 255`). Rerunning any of the counts below means reproducing that decode; the
better long-term home for it is a `glidertool house census` subcommand (`cmd/glidertool/house.go`,
built with `$HOME/.local/opt/go/bin/go`) driving `internal/house` instead of raw offsets.

### 6.1 Settled during this pass — do not re-open

1. **`HandleTelephone` cannot divide by zero.** `live-state-inventory` §openQuestions 1 and
   its trap T25 declined to claim it. The chime block is `if (numChimes > 0)` at `Play.c:771`
   and spans `:772-788`, so the `delayTime = kChimeDelay / numChimes;` at `:780` is inside the
   guard. It is followed by `if (delayTime < 2) delayTime = 2;` (`:781-782`) and
   `theChimes.nextRing = RandomInt(delayTime) + 1;` (`:784`). The phone block above it is a
   separate `if (!phoneBitSet)` (`:748`), not an `else if`, so both can sound on one frame.
   Delete T25; keep the floor of 2, which is a real behaviour.
2. **`IsThisValid` is six lines and its predicate is now written down.** `Objects.c:89-122`:
   `itsGood = true`; `case kObjectIsEmpty: itsGood = false;`; one grouped case —
   `kRedClock`, `kBlueClock`, `kYellowClock`, `kCuckoo`, `kPaper`, `kBattery`, `kBands`,
   `kFoil`, `kInvisBonus`, `kStar`, `kSparkle`, `kHelium` — doing
   `itsGood = (*thisHouse)->rooms[where].objects[who].data.c.state;`. **No `default`**, so
   every other kind is unconditionally valid. `kSparkle`'s membership is why a consumed
   sparkle stops registering. Closes `live-state-inventory` §openQuestions 7.
3. **The `kInvisBlower`/`kLiftArea` direction selector is `theObject.data.a.vector & 0x0F`.**
   `ObjectRects.c:523` (verified) and `:595` (UNVERIFIED: read by critic 3, not re-read here),
   with `case 1` up → `kLiftIt`, `2` right → `kPushItRight`, `4` down → `kDropIt`, `8` left →
   `kPushItLeft`, matching the bit comment at `GliderStructs.h:16`. There is **no `default`**,
   so a low nibble outside {1,2,4,8} produces **no hot spot at all**, and each type produces
   **one** rect, never two. Closes `hotspot-pipeline` §openQuestions 5 and corrects its trap 4.
4. **`hotSpots` cannot overflow on the shipped corpus, but the margin is three slots.** The
   per-object maximum is **three**, not two: a tall `kTaper` emits `kLiftIt` (`ObjectRects.c:410`)
   and `kBurnIt` (`:413`) inside `if ((bounds.bottom - bounds.top) > kDeadlyFlameHeight)`
   (`:407`), then `kDissolveIt` unconditionally at `:421` — verified by reading `:399-422`, and
   the same shape holds for `kCandle`, `kStubby`, `kTiki`, `kBBQ`. So `live-state-inventory`'s
   `24 * 2 = 48 < 56` argument is void. Census result: scoring 3 per tall flame, 2 per flame
   with `distance <= 24`, 2 per fan, 2 per microwave and a generous 1 for every other live
   object, the worst room in the 22 houses is **Slumberland room 256 at 53** — and that is a
   strict upper bound, because the lights, `kSparkle`, `kCustomPict` and the fourteen clutter
   types were counted as 1 when `CreateActiveRects` gives them none. The largest tall-flame
   population in one room is 16. `kMaxHotSpots` is 56, so `AddActiveRect`'s
   `return (-1)` (`ObjectRects.c:280-281`) is never taken on shipped data. Keep the -1 path and
   keep the `hotNum != -1` guards; a third-party house can still reach it. Closes
   `live-state-inventory` §openQuestions 4.
5. **Real houses do set `data.e.type` to all three switch actions, and never to a fourth.**
   Census of the `type` byte (`data.e` offset 9): `kLightSwitch` 61/30/22, `kMachineSwitch`
   32/22/25, `kThermostat` 55/33/20, `kPowerSwitch` 62/6/10, `kKnifeSwitch` 110/43/77,
   `kInvisSwitch` 167/259/209 for `kToggle`/`kForceOn`/`kForceOff` (0/1/2,
   `GliderDefines.h:441-443`); `kTrigger` 239 and `kLgTrigger` 81, **all** with `type == 3`
   (`kOneShot`, `:444`). `HandleSwitches` passes `data.e.type` to `SetObjectState`
   (`Interactions.c:998-999`) and every family's inner `switch (action)` has cases for
   `kToggle`, `kForceOn`, `kForceOff` (e.g. the blower family, `Objects.c:386-400`), so the
   *out-of-range-action* route to the uninitialised `changed` is unreachable through a switch on
   shipped data. `FireTrigger` passes a literal `kForceOn` (`Triggers.c:113`), so `kOneShot`
   never reaches `SetObjectState` at all. Closes `rewards-switches-triggers` §openQuestions 1.
   This does **not** close the other two uninitialised-`changed` routes (`case kSlider: break;`
   at `Objects.c:475-476`, and the missing `kKnifeSwitch` label — see item 6).
6. **The `kKnifeSwitch` gap in `SetObjectState` IS reachable in shipped data — two links.**
   `kKnifeSwitch` (0x45) has no label in `SetObjectState`'s outer switch (occurrences in
   `Objects.c` are `:153`, `:199`, `:243`, `:967`, all outside `:366-699`), so a switch or
   trigger whose **target** is a knife switch returns the uninitialised `changed`. Census: two
   such links exist — Teddy World room 163 slots 16 and 17, both `kInvisSwitch`, both targeting
   room 202. Neither is filtered out by the `linkIndex != -1` guard, because `SetObjectState`
   is called at `Interactions.c:998` *before* that guard and with the real `roomLink`. So
   flipping either switch reads an uninitialised local in the original. In Go the zero value
   makes `changed` deterministically `false` (switch art does not redraw, `stillOver` still
   latches at `Interactions.c:1154`); that is a divergence to record, not a bug to fix.
7. **No shipped `kTrigger`/`kLgTrigger` has `who == 255`.** All 11 unlinked switch-family
   objects are switches (`kLightSwitch` ×2, `kMachineSwitch` ×1, `kThermostat` ×1,
   `kPowerSwitch` ×1, `kKnifeSwitch` ×5, `kInvisSwitch` ×1). `ArmTrigger`'s
   `triggers[where].what = masterObjects[triggers[where].object].theObject.what;`
   (`Triggers.c:50`) therefore never indexes `masterObjects[-1]` on shipped data, so the Go
   panic critic 2 predicted cannot fire on the original 22 houses. Drop the write anyway —
   `triggers[].what` has one writer and no readers.
8. **`dinahs[-1]` IS reachable in shipped data — fourteen links.** The mechanism: neighbour-room
   balloons, copters, darts, balls, drips, fish, toasters and sparkles are registered only for
   the central room (`if ((neighbor == kCentralRoom) && (!redraw))`, e.g. `ObjectDrawAll.c:797`
   for `kBalloon`, `:808`/`:818` for the copters), so their `masterObjects[].dynaNum` stays -1;
   `HandleSwitches` guards on `linkIndex != -1` (`Interactions.c:1036`) and never on `dynaNum`,
   passing it straight through at `:1090` (`ToggleToaster`), `:1124` (`ToggleBalloon`), `:1136`
   (`ToggleDart`); `ToggleToaster`/`ToggleBalloon` are one line each,
   `dinahs[index].active = !dinahs[index].active;` (`Trip.c:24`, `:106`). Census: 14 links whose
   target is a central-only dynamic in a room **adjacent** to the switch's own room, i.e. in the
   3×3 view when the player is standing on the switch — 12 `kKnifeSwitch` and 2
   `kMachineSwitch`, targeting 7 `kBalloon`, 5 `kToaster`, 2 `kDartRt`, in Slumberland (rooms
   45, 249, 308), Teddy World (rooms 69, 202) and The Asylum Pro (rooms 68, 69). A further 51
   cross-room links to central-only dynamics resolve to rooms outside the view, where
   `localLink == -1` filters them. This settles the `dynamics-core`/`dynamics-movers` gap in the
   direction the two dynamics reports guessed: the `dynaNum != -1` guard is **mandatory**, not
   defensive, and a fidelity test should cover Teddy World room 202's balloon switches.
9. **No shipped switch or trigger links to a grease jar at all.** Zero of the 1,531 resolvable
   links target `kGreaseRt`/`kGreaseLf`. So on shipped data: `HandleSwitches`' grease case
   (`Interactions.c:1058-1061`), `FireTrigger`'s grease case (`Triggers.c:110-118`) and
   `FireTrigger`'s `localLink == -1` else arm with its `masterObjects[-1]` read
   (`:174-192`, guard at `:107`) are all dead, and with them `bands-grease`' T22 `grease[-1]`
   and T23 `hotSpots[-1]` by that route. Closes `bands-grease` §openQuestions 1. Caveat, and it
   is the reason the guards must stay: the 21 links whose floor/suite is absent from the house
   give `triggers[].room == -1`, and `FireTrigger`'s else arm then reads
   `(*thisHouse)->rooms[-1].objects[who].what` (`:177-179`) to choose its case label — so
   `rooms[-1]` is genuinely read on shipped data (corroborating `live-state-inventory`'s T1),
   and only the garbage value it returns keeps the grease arm from firing.
10. **No shipped `kInvisBonus` carries `points == 250`.** Census over `data.c.points`: zero
    occurrences. `pointsSrc`'s 250 group (indices 9..11) is therefore unreachable in the
    original houses, though `Interactions.c:920-925` still selects it for any third-party house
    that sets the value. Closes `background-animations` §openQuestions 5.
11. **`nLocalObj` is dead everywhere, not just in the gameplay path.** One occurrence in the
    tree: the declaration at `Objects.c:76`. No field for it.
12. **The build configuration is not in doubt.** `COMPILEQT` is defined and
    `BUILD_ARCADE_VERSION` is `1` (`GliderDefines.h:14-16`, verified). That is a fact, so
    `dynamics-movers` §openQuestions 4 is closed as a question about the C; what remains is the
    port-policy half, in §6.4.
13. **`SectGlider` and `GliderInRect` are already ported.** `room-transitions`
    §openQuestions 6 lists them as unread. They are Stage 1.4 work — `internal/game/player`
    (hit box, `GliderInRect`, `GliderHitTop`, `BounceGlider`) with the C walked in
    `docs/analysis/player-physics.md` §11. Read those rather than re-deriving.

### 6.2 Still open: data questions a corpus census would settle

Each of these has a certain mechanism and an unknown incidence. None blocks writing the Go —
all of them decide whether a fidelity test must assert the degenerate path.

1. **Does any shipped 3×3 view over-subscribe the 18-slot `dinahs` table?** This is a live
   contradiction in the union: `live-state-inventory`'s census says nothing exceeds 18 (so
   `AddDynamicObject`'s `return (-1)` at `Dynamics3.c:193-194` is never taken), while
   `dynamics-core` and `dynamics-movers` both assert reachability by arithmetic. The censuses
   are not comparable, because the deciding factors are (a) the nine-room registration and the
   central-room-last order (`RoomGraphics.c:80-121`), (b) the seven appliances registering for
   all nine rooms while the enemies register central-only, and (c) the `SectRect` visibility
   gate that the appliances have and the enemies do not — which makes the answer
   screen-size dependent. *Next action:* a census that composes each room the way `DrawLocale`
   does — `internal/render.Scene` already has the neighbourhood arithmetic
   (`internal/render/locale.go:1000`, `GetNeighborRoomNumber`) — walks the nine rooms in draw
   order, applies the same `SectRect` gate as `ObjectDrawAll.c:658-794`, and reports the maximum
   registration count and the first dropped object per room, at each supported screen height.
   Until it is run, keep the cap, keep the draw order, and do not write a test that asserts
   drops.
2. **Does any shipped 3×3 view exhaust the 24-slot `savedMaps` table?** This decides whether
   `background-animations`' traps 4 and 19 are theoretical. It is worth more than it looks:
   critic 2 established that the five room-load `RandomInt` draws sit **inside**
   `if (savedNum != -1)` (`DynamicMaps.c:333-338`, `:417-422`, `:503-508`, `:580-594`,
   `:680-685`), so a saturated table does not merely drop art — it shifts the whole RNG stream,
   and it does so as a function of screen size (the `SectRect` gate) and of whether
   `RestoreEntireGameScreen` has leaked slots. *Next action:* same composition harness as item
   1, costing 1 slot per prize, per grease jar and per star (a star is **one** 32×186 strip,
   `DynamicMaps.c:679-680`, not two) and 2 per cuckoo, over all nine rooms.
   `docs/analysis/houses-inventory.md` and `original-houses.md` may already hold the per-type
   room census this needs.
3. **Is there a `kRoof` room in the shipped houses containing a grease jar or a `kSlider`?**
   Decides whether `bands-grease`' roof-immunity behaviour (its T12/G3 — `kSlideIt` sets
   `sliding`, which suppresses `CheckRoofCollision` at `Interactions.c:455`) is exercised by
   real content. *Next action:* one pass over the corpus for rooms whose background is `kRoof`
   holding `what` in {0x28, 0x29, 0x2F}.
4. **Are the three `dinahs`-saturated rooms `live-state-inventory` names (Teddy World "Moving
   walls", Teddy World "Get out of here!", California or Bust! "And the Pets, Too") reachable
   in play?** Only affects the priority of the fidelity test, and cannot be answered by a
   census — it needs a scripted traversal, which is Stage 1.8's harness. Leave it until then.
5. **Where the corpus answer is already known, prefer it to a re-census.** `enemies.md`
   Appendix A A.3-A.4's 22-house trigger-link census (171 trigger→switch links, all same-room;
   one cross-room trigger targeting an empty slot) was independently reproduced twice, by
   `live-state-inventory` through `internal/house` and by this pass through raw offsets. Treat
   that appendix as the model for how these results should be written up: mechanism, census,
   verdict, in that order.

### 6.3 Still open: C reads nobody has done

| # | Question | Read | What it decides |
|---|----------|------|-----------------|
| 1 | Do the copter and dart band-hit guards allow a second hit on a falling enemy? | `Dynamics2.c:30-355` (`HandleBalloon`, `HandleCopter`, `HandleDart`) | Closes both `bands-grease` §openQuestions 5 and `dynamics-core` §openQuestions 4 — the same read. The balloon's guard is known (`Dynamics2.c:32-35`: `moving` and `vVel < 0`, which makes a re-hit impossible); the other two are not. Also cross-checks the `hVel`/`position`/`count` rows of the dynamics field table against their consumers. |
| 2 | After `SetObjectState` returns, does the file-scope `newState` describe the switch or its target? | `Objects.c:533-600` (the switch family) plus the draw calls at `Interactions.c:1007-1028` | `hotspot-pipeline` §openQuestions 3. It decides which state the switch **art** is drawn in, which is visible in every room with a light switch. |
| 3 | Which coordinate space do `grease[].dest` and `grease[].start` live in? | `Grease.c:180-265` (`AddGrease`, `BackupGrease`) and `ObjectDrawAll.c:360-405` | `hotspot-pipeline` §openQuestions 2. `HandleGrease` offsets by `-playOriginH/-playOriginV` before storing the `kSlideIt` bounds (`Grease.c:66`, `:68`) but the spreading branch (`:93`, `:100`) does not, and the two are consistent only if `start`/`dest.bottom` are screen-space. Wrong answer here misplaces every slide rect by one room origin. |
| 4 | What happens to the *other* glider's sprite sheet when `foilTotal` goes positive in a two-player game? | `Player.c:1140-1160`, `Modes.c:453-480` | `hotspot-pipeline` §openQuestions 1. `HandleRewards` adds `kFoilSupply` twice (`Interactions.c:910-912`) into a single global but calls `StartGliderFoilGoing` for one glider (`:913`). |
| 5 | What does each type's `dinahs[].byte1` mean, and does anything read it? | `Dynamics.c`, `Dynamics2.c`, `Dynamics3.c`, `Trip.c` | `live-state-inventory` §openQuestions 3. Two reports independently found every `AddDynamicObject` case writes 0 and nothing reads it; the residual question is documentary, so this is a low-priority read that should end in one sentence in the dynamics field table, not in a Go field. |
| 6 | Does anything read `roomType.openings` at run time? | the rest of `Room.c` and `HouseLegal.c` | `live-state-inventory` §openQuestions 2. It matters only because the `who == 255` write path lands on `objects[-1].data.a.state`, which is the low byte of `openings`. `DetermineRoomOpenings` recomputes the four `*Open` flags from the tiles and background rather than from `openings`, so the corruption is believed invisible; proving it needs the grep. |
| 7 | Does `internal/render`'s `GetNumberOfLights` match `Room.c:970-1099`? | `Room.c:970-1099` against `internal/render/locale.go:1079` | `room-transitions` §openQuestions 4. `numLights` gates `DrawRoomBackground`'s blackout early return (`RoomGraphics.c:179-191`), i.e. whether a room draws at all. Presumed already ported and never cross-checked. |
| 8 | Is the 117-row type → hot-spot-action table complete? | `ObjectRects.c:296-1063` in full, all 79 `AddActiveRect` sites | Critic 3's headline gap: all four reports that touch `CreateActiveRects` list it as *supporting*, so 24 types with live hot spots (12 furniture `kDissolveIt` solids, 7 vents/blowers, 3 static appliances, `kDoorExRt`, `kWindowExRt`) were never specified by anyone *in the raw reports*. **Resolved during assembly:** §3.2-§3.10 does contain the table, for all 117 types, so this is no longer the blocking item; what remains is a verification read of `ObjectRects.c:296-1063` against §3's tables, which nobody has done end to end. The geometry constants at `ObjectRects.c:11-17` (`kFloorColumnWide 4`, `kCeilingColumnWide 24`, `kFanColumnThick 16`, `kFanColumnDown 20`, `kDeadlyFlameHeight 24`, `kStoolThick 25`, `kShredderActiveHigh 40`) belong with it. |
| 9 | Is `kBasement`'s absence from `IsRoomAStructure`'s list deliberate? | `Room.c:790-800` vs `:850` | `room-transitions` §openQuestions 7. Authorial intent, so unanswerable; it affects `DrawFloorSupport` only. Reproduce the asymmetry and note it. |
| 10 | What was the `quickerTransitions` preference meant to select between? | `Transitions.c:18-70`, `:139-144`, and the three commented-out sites at `Play.c:156-168`, `:816-819` | `room-transitions` §openQuestions 8. Not used by any room transition (`MoveRoomToRoom` uses `WipeScreenOn`, `NewGame` uses `DumpScreenOn`), so out of scope for Stage 1.5; recorded so the next reader does not chase it. |

### 6.4 Still open: decisions, not discoveries

1. **Does the port reproduce the double `RenderFrame` on a transition frame?** `RenderFrame`
   has five call sites, not one: `Play.c:469`, `:494`, and `Transit.c:303`, `:341`, `:380`,
   `:419` (verified by grep; all four Transit sites are live because `COMPILEQT` is defined, and
   all four sit before the movie `if`, outside the `if (!sameRoom)` guards). `RenderFrame` is not
   idempotent: `evenFrame` flips only at `Play.c:435`, so the second call takes the *same*
   branch of `Render.c:649-652` — flame/tiki/coal strips advance twice and the stars not at all,
   or the reverse; `clockFrame` advances twice; the `while (TickCount() < nextFrame)` limiter
   waits twice; `HandleGrease` runs twice. On a **same-room** transport, duct or mail trip
   `ReadyLevel` is skipped (`Transit.c:334-338`) while the `RenderFrame` at `:341` still runs, so
   shreds, sparkles and flying points double-step too. *Decision:* reproduce and pin it with a
   test, or diverge and record it. It cannot stay as a one-clause aside, and
   `background-animations`' "neither ever updates twice in a row" must go either way.
2. **What does the Go frame loop do about events?** The C's step 3 is
   `if (doBackground) { do { HandlePlayEvent(); } while (switchedOut); }` (`Play.c:437-443`).
   `HandlePlayEvent` (`:387-425`) handles exactly two event kinds — `updateEvt` for `mainWindow`
   and `osEvt` suspend/resume — has no `keyDown` handling at all, and its
   `WaitNextEvent(everyEvent, &theEvent, sleep, nil)` with `sleep = 2` is a second, independent
   2-tick pacing source stacked on `RenderFrame`'s busy-wait; the `do/while (switchedOut)` halts
   the simulation without advancing `gameFrame` or `evenFrame`. UNVERIFIED (critic 1's claim, not
   re-read here): `doBackground` is a preference read at `Main.c:118` and forced false at
   `Main.c:186` and `Settings.c:1228`, which would mean the shipped game processes no events
   during play. *Decision:* state whether the port has an event step at all, and if the answer is
   "no", say that the two pacing sources collapse to one.
3. **Reproduce or guard the negative indices?** Six of them, with what §6.1 now knows about
   reachability: `dinahs[-1]` (**reachable**, 14 links), `rooms[-1]` (**reachable**, 21 links),
   `masterObjects[-1]` via `ArmTrigger` (unreachable on shipped data), `grease[-1]` and
   `hotSpots[-1]` via the grease paths (unreachable — no shipped link targets a jar),
   `hotSpots[uninitialised]`, and `objects[-1]` via `SetObjectState`'s `who == 255` path. In C
   these read and write junk; in Go they panic. The first two are not hypothetical, so this is a
   decision the port must take before a headless traversal of Teddy World: guard with a recorded
   deviation, or reproduce the read against a scratch slot. Precedent exists both ways —
   `internal/render` reproduces two original bugs deliberately.
4. **`rightClip`/`leftClip`: restore the global or keep Stage 1.4's per-glider copy?** They are
   one shared pair in the C (`short rightClip, leftClip, transRoom;`, `Player.c:52`), written at
   `Modes.c:150`, `:536`, `:567` and read at `Player.c:407`, `:463`, `:536`; Stage 1.4 made them
   per-glider (`internal/game/player/glider.go:87`, whose comment admits it). *Decision needs an
   observability test:* two gliders on staircases in the same room, which is the only state where
   the sharing can differ. Nobody has run it.
5. **Where does the pending transit live?** `transRect` (`Player.c:49`), `transRoom`
   (`Player.c:52`) and `linkedToWhat` (`Transit.c:17`) are single globals that both gliders read,
   and `FollowTheLeader` (`Transit.c:533-551`) plus `OffAMortal`'s `mortals == -1` re-dispatch
   (`Player.c:1575-1595`) dispatch the *survivor* against the *dead leader's* destination.
   Stage 1.4's per-glider `Transit Link` (`glider.go:93`) cannot express that. If the live-state
   design in the sections above does not already put a `Pending Link` on the world and say which
   of the four transit functions reads it rather than `g.Transit`, that is a defect to fix, not an
   open question.
6. **Is the room a constructor-only value?** `DrawLocale` has exactly three call sites —
   `RoomGraphics.c:416` (from `ReadyLevel`), `Play.c:163` (`NewGame`) and `Play.c:814` — and the
   third is inside `RestoreEntireGameScreen` (`:800-820`), whose only caller is `Banner.c:234`
   under `if (WaitForInputEvent(30))` in the stars-remaining banner. That path rebuilds the whole
   locale **without** `NilSavedMaps`, `DetermineRoomOpenings` or `InitGarbageRects`, i.e. it
   leaks `savedMaps` slots and leaves the dirty-rect state alone. *Decision:* whether the port
   reproduces it, and whether the locale rebuild is therefore a method separable from
   construction. `background-animations` §openQuestions 3 also noticed that
   `RestoreEntireGameScreen` paints the window black (`Play.c:810-812`) and never blits work to
   the screen, so the play area would stay black until the dirty rects repaint it — UNVERIFIED,
   and it belongs to whoever owns `DisplayStarsRemaining`, not to Stage 1.5.
7. **What does the port do with a modal called from inside a frame handler?**
   `DisplayStarsRemaining` does `DelayTicks(60)` and `WaitForInputEvent(30)` (`Banner.c:232-233`)
   from inside `HandleRewards`, i.e. from inside `HandleInteraction`, i.e. mid-frame. This
   interacts with whatever Stage 1.5 does for `DoPause` (`internal/game/player/env.go:137`) and
   with `docs/analysis/determinism.md`. Unanswerable from the C; it is an architecture call.
8. **Partially off-map animation strips.** `BackUpFlames` and friends `CopyBits` a `src` that can
   extend past `backSrcRect` — reachable for `AddStar` (no `h`/`v` guard) and for tikis (no
   `SectRect` at all). QuickDraw clips src and dest together, so the uncovered part of the cell
   shows whatever `CreateOffScreenGWorld` left there, and the C does not zero its GWorlds.
   *Recommended decision:* clamp the copy in Go, leave the remainder black, record the
   divergence.
9. **Byte-exactness of `SetObjectsToDefaults`.** It is a hand-written case list, not a per-union
   sweep (`Play.c:603-708`): the blower case lists 11 of 16 types and omits the five flames, the
   bonus case omits `kSlider`, the appliance case omits `kCinderBlock`, `kFlowerBox`, `kCDs`,
   `kCustomPict`, the enemy case omits `kCobweb`, and furniture, switches and clutter appear not
   at all. The omissions are behaviourally inert (`Objects.c:420-425` gives the flames
   `changed = false`) but not byte-inert. *Decision:* transcribe the case list, or sweep by
   variant and accept that a hand-edited house whose flame `state != initial` diverges in the
   saved bytes.
10. **`IsShadowVisible()` in the `Env` interface conflates two things** — the pure recompute
    (`Room.c:1103`) and the cached global `shadowVisible` (`Player.c:53`), where `NopEnv` returns
    whatever `SetShadowVisible` stored (`internal/game/player/env.go:194-195`). A real
    implementation that recomputes and a test that stores will disagree. *Decision:* split the
    method or document which one it is. Related: `shadowVisible` has **four** writers, not three —
    `RoomGraphics.c:126` (`DrawLocale`), `Modes.c:361`, `Player.c:1286` and
    `RoomGraphics.c:459` (`RedrawRoomLighting`), so a light switch re-derives it mid-play.
11. **Demo playback has no owner.** `demoIndex` is in `NewGame`'s reset block (`Play.c:114`),
    `demoGoing`/`GetDemoInput` are on the frame loop (`Play.c:478-479`) and `demoData` is
    allocated at `StructuresInit2.c:287`. It sits on the same line of the frame as `GetInput`.
    *Decision:* declare it out of scope for Stage 1.5 in writing, or give it a home.
12. **`WebGlider` needs `EvenFrame()` on `Env`** if the hot-spot lookup moves into
    `internal/game/player` as the live-state pass recommends: `WebGlider` reads the global
    `evenFrame` to gate the twang at `Interactions.c:1756` (`:1755` is the brace). Nobody listed
    it as a required `Env` addition. Decide when the method list is frozen.

### 6.5 Permanently unverifiable from the source

Keep these hedged in the prose; no future pass can close them by reading the C.

1. **What a fresh `CreateOffScreenGWorld` contains.** `Environ.c` does not zero them, so the
   uncovered part of an off-map strip is undefined on real hardware. Feeds §6.4 item 8.
2. **The 68k/PPC layout consequence of the `savedMaps` declaration mismatch.**
   `background-animations`' conclusion that it is benign rests on `savedType` beginning with
   `Rect dest` (`GliderStructs.h:229-232`), so the `NewPtr` value lands in `dest.top`/`dest.left`
   rather than in `map`. If `Rect` were laid out differently, the first `NilSavedMaps` would
   `DisposeGWorld` a `NewPtr` block.
3. **How the 1994 linker resolved `savedMaps`** — array at `Objects.c:73`, `extern savedPtr` at
   `StructuresInit2.c:40`, `CGrafPtr[]` at `RoomGraphics.c:35`. The claim that
   `StructuresInit2.c:232-233` clobbers `savedMaps[0].dest.top`/`.left` and leaks 384 bytes is an
   inference about tentative definitions. The conclusion (benign; the real array is BSS-zero and
   `.dest` is overwritten at `DynamicMaps.c:80` before any read) holds either way.
4. **Whether the shipped 1.0.4 binary matches these sources**, which is what would settle
   `dynamics-core` §openQuestions 1 (the `HandleOutlet` `doOffset` asymmetry: `false` at
   `Dynamics.c:539`/`:541` versus `true` at `:545`/`:546`/`:550`, against an outlet whose `Dest`
   is global) as *bug* versus *intent*. Reproduce the asymmetry and mark it; do not unify it.
5. **Sprite pixel semantics** — which balloon frame looks like what, whether `dripSrc[0..2]` nest
   inside `dripSrc[3]`, whether `fishSrc[0..3]` differ visibly, and whether the microwave's
   `microOn`/`microOff` is one 48-wide strip or a 16-wide cell meant to tile three times
   (`Dynamics3.c:376` widens to `left + 48`; `Dynamics.c:729-744` fills it with three 16-wide
   blits). Not settleable from the C, but settleable from the assets: render PICTs
   4009/4012/4014/4015/4016/4017 and their 5000-series masks, and inspect the decoded appliance
   atlas in `internal/render`. Only affects how strongly a test may assert.
6. **`srcRects[]` provenance.** `dynamics-movers` cross-checked its four values against
   `internal/render/srcrects.go:311-318` rather than a second C table, so if that Go table was
   itself transcribed from `StructuresInit2.c:431-457` the check is not independent.

### 6.6 Corrections to apply to `docs/analysis/*.md` (batch — do not apply from here)

Every row below was found by one of the eight subsystem passes and, where a critic disputed it,
re-verified against the C. Apply as one editing pass. Rows are grouped by target file; the
caption names the pass that found them.

**`rendering.md`** (found by `background-animations`)

| Where | Wrong claim | Correct claim | Citation |
|---|---|---|---|
| `:3455-3456` | `AddPendulum` seeds `clockFrame = 10` so the first swing happens on the very next rendered frame | `RenderPendulums` increments first, so `clockFrame` becomes 11, matching neither `== 10` nor `== 15`; the first swing is five frames later | `Render.c:270` |
| `:3457-3458` | "roughly 2 ticks per second with deliberately uneven spacing — the tick…tock cadence" | The *swings* are unevenly spaced (5, 10, 5, 10) but a sound fires only on the direction flip, so `kTikSound`/`kTokSound` are exactly 15 frames apart; what is uneven is the dwell — 10 frames at each extreme, 5 at centre | `Render.c:260-290` |
| `:3213-3216` | The pointer version at `StructuresInit2.c:232-233` *shadows* the `savedMaps` array in that translation unit | `StructuresInit2.c:40` is `extern savedPtr savedMaps;` — the same linker symbol, so there is no shadowing: the `NewPtr` writes a pointer into the first four bytes of the real array and the `map = nil` loop nils a block nothing else sees. Harmless, for a different reason than stated | `Objects.c:73`, `StructuresInit2.c:40`, `:232-233` |
| `:2496-2498` | Gate column reads "central-only" for `kTaper`/`kCandle`/`kStubby` | The neighbour path exists; a neighbour flame registers whenever its rect misses the central room expanded by `kFloorSupportTall`. §12's prose at `:2575-2581` already states this correctly | `ObjectDrawAll.c:83`, `:98`, `:117`, `:132`, `:151`, `:166` |
| `:2499` | Gate for `kTiki` is "isLit" | `isLit` gates only `DrawTiki`; the tiki flame registration is unconditional and, uniquely, has no `SectRect` test | `ObjectDrawAll.c:179-182` |

**`interactions.md`** (found by `bands-grease`, `hotspot-pipeline`, `rewards-switches-triggers`,
`live-state-inventory`)

| Where | Wrong claim | Correct claim | Citation |
|---|---|---|---|
| §19.1 | The grease jar's fall is a 3-frame animation | Four frames: `frame` starts at -1, `frame++` precedes the `>= 3` test, and the `>= 3` block does not `else` out the blit | `Grease.c:55-80` |
| §19.2 | The ±8 in `AddGrease` shifts the jar art 8 px away from the spill direction so the tipped silhouette lines up | It exactly cancels `BackupGrease`'s four ±2 offsets, leaving `grease[].dest` equal to the jar's original `itsRect` | `Grease.c:159`, `:167` |
| §19.2 | The spill length is `data.a.distance` | The call site passes `data.c.length` — same byte offset, wrong union group | `ObjectDrawAll.c:378`, `:399` |
| §19.3 | Four `SpillGrease` call sites | Five; the missing one is the `localLink == -1` arm, which additionally reads `masterObjects[-1]` | `Triggers.c:187` |
| §19.4 | `RedrawAllGrease` is called from the clock-pickup paths (`:777`, `:793`, `:809`, `:826`) | Ten sites | `Interactions.c:777`, `:793`, `:809`, `:826`, `:845`, `:867`, `:886`, `:914`, `:947`, `:973` |
| §9.12 | The `kSlideIt` velocity write is "a hard snap, not a spring" | It is attenuated by `MoveGlider`'s ±2 ramp before it displaces anything; `internal/game/player/glider.go:189-201` already says so, i.e. the doc contradicts the committed Go | `Player.c:79-92` |
| §9.12 | `sliding` "disables the normal ground handling for one frame" in `MoveGliderNormal` | `MoveGliderNormal` contains only sprite selection, the flag clear and `MoveGlider()` — there is no ground handling there to disable | `Player.c:151-200` |
| §9.12 | "so you can slide off a roof" | Every branch of `CheckRoofCollision` kills the glider, so suppressing it means you do **not** die — you skate along the shingles | `Interactions.c:455`, `:1376-1379` |
| §14.5 | Overlapping the trail "snaps the glider's bottom exactly to the trail's top" | Same ramp attenuation; the equilibrium is 2 px below the line top | `Player.c:79-92`, `Grease.c:63-68` |
| §22.15, §18.4 | The dead `collided = true` stores are explained by the hot-spot loop "overwriting `collided` on its first iteration" | The loop assigns `collided` only inside a triple-nested guard, so with no on/relevant hot spot it is never assigned at all. The stores are dead for a different reason: the glider phase reassigns it, and if `hVel == 0` it is never read | `Interactions.c` sweep, `RubberBands.c:145-146` |
| §18.4 | "One band grazing two hot spots in successive frames will suppress the second" | `bandHitLast != i` suppresses only the **same** index; and the reachable inverse is missing — a second band that hits nothing resets `bandHitLast` to -1 and defeats the first band's debounce | `RubberBands.c:86`, `:88`, `:146` |
| §17.4 | `DidBandHitDynamic` has four call sites / four enemy types | Three | `Dynamics2.c:62`, `:162`, `:267` |
| §17.5 | Band-hit list of enemies | Omits the copter entirely: frame 8, `hVel = 0`, `vVel = 8`, `kPaperCrunchSound` | `Dynamics2.c:162-168` |
| §9.10 | A `delay == 0` trigger "fires one frame later" (and the parenthetical contradicts the sentence) | It fires in the **same** frame: `ArmTrigger` runs inside `HandleInteraction` and `HandleTriggers` follows in the same iteration. General rule: it fires on the `max(1, 3d)`-th call, i.e. `3d - 1` frames later for `d >= 1` and 0 frames later for `d == 0` | `Play.c:482`, `:484`, `Triggers.c:49`, `:87-88` |
| §11.2, last table row | "lights, appliances, enemies — `hotSpots[hotNum].isOn` sync at `:529`, `:625`" | `:529` is the `kDeluxeTrans` sync and belongs to the row above; of the three families named only the appliances sync `isOn`, and that sync is narrowed twice — by `if (room == thisRoomNumber)` and by `if (what == kShredder)` — so it is shredder-only | `Objects.c:521-530`, `:572-577`, `:621-625`, `:665-670` |
| §11.2, appliance row | (omission) | The shredder `isOn` sync is the only one **without** a `hotNum != -1` guard, so a shredder whose hot spot was refused writes `hotSpots[-1].isOn` | `Objects.c:415-416`, `:469-470`, `:528-529` vs `:624-625`; `ObjectRects.c:280-281` |
| §11.2, §22 | (omission) | A second uninitialised-`changed` path: `kKnifeSwitch` (0x45) has no label in `SetObjectState`'s outer switch, so any switch or trigger whose target is a knife switch returns garbage. Reachable — two shipped links (§6.1 item 6) | `Objects.c:366-699`, switch-family labels at `:533-540` |
| §10.1, "Extra" column | (omission) | Omits `RedrawAllGrease()`, which ten of the fourteen reward cases call and which `kInvisBonus` and `kGreaseRt`/`kGreaseLf` pointedly do not — the asymmetry matters because `RedrawAllGrease` is where the `hotSpots[grease[i].hotNum]` out-of-range read lives | `Interactions.c:777`…`:973`; `Grease.c:282` |
| §11.1 | "`ToggleMacPlus` sets `timer = 10`, `ToggleToaster` sets `timer = 40`" | `ToggleToaster` sets no timer — it is one line, `active = !active`. `ToggleMacPlus` sets both numbers (40 when it has just become active, 10 when it has not). Also omits `ToggleTV`'s unconditional `timer = 4` outside the QuickTime branch | `Trip.c:22-25`, `:29-36`, `:57` |
| §9.10, FireTrigger table | "six switch types — `TriggerSwitch(dynaNum)` → `HandleSwitches(&hotSpots[who])`" is left unexplained | Literally correct but it passes a `dynaNum` into a `hotSpots` index: for the six switch types `dynamicNum` is assigned `masterObjects[i].hotNum`, which coincides with the master index only for the central room, so a trigger targeting a neighbour-room switch reaches an unrelated hot spot or `hotSpots[-1]`. Belongs in §22 | `ObjectDrawAll.c:518`, `:531`, `:544`, `:557`, `:570`, `:574`; `Trip.c:148` |
| §22.4 | `triggers[].what` "is never actually consumed… so the bug is inert" | Inert as stated (one writer, no readers, confirmed by grep), but the *write* is the hazard: `Triggers.c:50` indexes `masterObjects` with a room-object slot, and with -1 when `GetObjectLinked` returned -1. In Go it panics. Drop the write | `Triggers.c:47`, `:50`; `Objects.c:203-206` |
| §22.12 | The `evenFrame` clobber is `HandleBall` at `Dynamics2.c:420` alone | Two more sites, both inside `AddDynamicObject`, i.e. during room **composition** rather than play: entering a room containing a ball or a fish desynchronises the flame/star alternation before the first frame | `Dynamics3.c:474`, `:524` |
| §1746, `:1748-1751`, `:1758`, `:3300-3302` | `FlagStillOvers` runs on room entry | It has exactly one caller — `FinishGliderDuctingIn`, the frame a ceiling-duct arrival finishes emerging, which may not even change room. Consequence for `:1758`: on a cross-room entry `AddActiveRect` already zeroed `stillOver`, so nothing latches a microwave at all unless the arrival happens to be a ceiling duct. `object-dynamics.md:329` states it correctly, so the docs contradict each other | `Player.c:954`; `Transit.c:359`; `Interactions.c:1725`, `:990` |
| §1739 vs §3253 vs §1774 | The two-player `stillOver` clear is cited as `:1675`, `:1673-:1674` and `:1674` in three places | `if (!hitObject)` is at `:1674`, the assignment at `:1675` | `Interactions.c:1674-1675` |
| §19.4, §18.4, §14.5, §17.4, §22.11, §22.16 | Systematic citation drift into `Grease.c` (low by 2, because line 1 of that file is blank) and a scatter elsewhere | `Grease.c`: falling branch `:53-87` not `:51-85`; `:58-59`, `:60`, `:61`, `:63`/`:65`, `:66`, `:67`, `:68`, `:71-80`, `:83-86` for the statements cited as `:55-57`, `:58`, `:59`, `:61`/`:63`, `:64`, `:65`, `:66`, `:68-78`, `:81-84`; `RedrawAllGrease` `:270` not `:265`, its `== 2` test `:284`, its playOrigin offset `:287`; the 2-px seed rect `:63-68`; `bandHitLast != i` `RubberBands.c:86` not `:83`; the dissolve/bounce block `:89-117`; the glider push `:165`/`:166`; `KillAllBands` `:307`; `AddBand`'s initial `vVel` `:263-266`; `DidBandHitDynamic` `Dynamics2.c:62`, `:162`, `:267`; the overlap idiom `Dynamics.c:87-100` | as listed |

**`object-dynamics.md` §2.4-§2.5** (found by `dynamics-core`)

| Where | Wrong claim | Correct claim | Citation |
|---|---|---|---|
| `:360` | `byte1` is "never read or written anywhere" | Written to 0 by **all fifteen** `AddDynamicObject` cases; never read. The write half matters because the next paragraph correctly notes `ZeroDinahs` does not clear it | `Dynamics3.c:212`, `:239`, `:259`, `:279`, `:302`, `:322`, `:345`, `:365`, `:386`, `:407`, `:432`, `:460`, `:492`, `:511`, `:541` |
| `:359` | `byte0` is the "room object index" | Also write-only: written by all fifteen cases, read nowhere. The live mapping is the reverse one, `masterObjects[].dynaNum` | `ObjectDrawAll.c:959` |
| `:354` | `count` is the "launch velocity magnitude" for toast/ball/fish | Magnitude only for the toast (`count = velocity`, positive, consumer negates). Ball and fish store `count = -velocity`, **negative**, and the consumer assigns it directly — a port taking the row at face value launches both downward. For the ball, `count` is also the per-bounce reload | `Dynamics3.c:233`, `:486`, `:535`; `Dynamics.c:375`; `Trip.c:159`; `Dynamics2.c:389`, `:418`, `:538` |
| `:355` | `frame`'s overload list is "toast idle: countdown" | Four more: the VCR stores a one-bit blink parity; the sparkle a 5-frame re-trigger lockout, not a sprite index; the dart a *facing* selector (0 for `kDartLf`, 2 for `kDartRt`); the outlet cycles 1,2,3 reserving 0 for the idle plug; the drip seeds 3 | `Dynamics.c:630`, `:634`, `:305`, `:312`, `:554`, `:562-564`; `Dynamics3.c:207`, `:446`, `:452`, `:505` |
| `:356` | `timer` has no overloads | The toaster's `timer` is a **constant**, written once and thereafter only read as the reload for `frame`; nothing decrements it. The fish reloads its timer from `hVel`, not `count`; the outlet's serves as both zap-length and idle countdown, distinguished by `position`; the coffee and VCR self-reload from hard-coded constants | `Dynamics3.c:235`, `:537`; `Dynamics.c:363`, `:381`, `:492`, `:506`, `:613`, `:629`; `Trip.c:165`; `Dynamics2.c:539` |
| `:357` | `position` is the outlet's state flag | It is a *coordinate* in five more types — copter `dest.left`, dart `dest.top`, ball and fish `dest.bottom`, drip `dest.top + data.h.length` — and a boolean only for the outlet | `Dynamics3.c:429`, `:458`, `:489`, `:538`, `:507-508` |
| `:351` | (correct, strengthen) | `hVel` is a genuine horizontal velocity only for copter (±1) and dart (±`kDartVelocity`): four overloads against two real uses | `Dynamics3.c` seeds as cited in the row |
| `:358` | "never set for darts" (correct, strengthen) | `dinahs[].room` has exactly one reader in the program, and it filters `type == kOutlet` first, so the dart's permanent 0 is provably unobservable | `Trip.c:241` |
| §2.5 | (omission) | `Toggle*`/`Trigger*` are invoked under a `linkIndex != -1` / `localLink != -1` guard only, never a `dynaNum != -1` guard, so a switch wired to a neighbour-room central-only dynamic calls e.g. `ToggleToaster(-1)`. Reachable — 14 shipped links (§6.1 item 8) | `Interactions.c:1036`, `:1090`, `:1124`, `:1136`; `Triggers.c:107`; `Trip.c:24`, `:106` |
| §2.4, §2.5 | (omission) | `DrawLocale` registers the central room **last** while the seven back-map appliances register for all nine rooms, so neighbour-room appliances can consume all 18 slots and silently drop central-room enemies | `RoomGraphics.c:80-121`; `ObjectDrawAll.c:664`, `:697`, `:721`, `:738`, `:754`, `:770`, `:786` |
| §2.4 | (omission) | The two-frame `timer == 1` / `timer == 0` back-map handshake, and that `RenderDynamics` deliberately omits `kSparkle` and all seven appliances — without which a porter puts the appliance drawing in the render phase | `Dynamics3.c:150-151` |

**`enemies.md`** (found by `dynamics-movers`, `hotspot-pipeline`)

| Where | Wrong claim | Correct claim | Citation |
|---|---|---|---|
| §11.6 (~`:1953`) | "The bob runs whether or not the fish is active… so a switched-off fish still bobs" | Premise right, conclusion inverted: the only place `timer` changes is inside `if (active)`, so an inactive fish's timer is frozen and it either bobs on **every** frame (a 1 px vibration) or never, decided by the frozen residue. At spawn `timer = delay*3`, so the vibrating case is `delay ≡ 1 (mod 4)` | `Dynamics2.c:558`, `:576`, `:578` |
| §11.6 | One leap height per fish | The apex depends on the global parity at the launch frame, and nothing forces parity at a leap (only at spawn). `length = 64, delay = 10` → constant 71 px every 34 frames; `delay = 11` → alternates 64/72 px and 32/34 frames forever; `length = 100, delay = 7` → alternates 99/109. It alternates whenever `delay` is odd or zero | `Dynamics3.c:524`; `Dynamics2.c:501-588` |
| §11.5 (~`:1868`) | "Fall time for distance `d` at half gravity is ≈`2*sqrt(d)`: default `length = 64` → ≈16 frames" | The travel is `length - 15` px (3 px detach plus the 12 px sprite already put `dest.bottom` at `spawnTop + 15`), so 49 px and 14 or 15 frames, parity-dependent. Missing degenerate case: any `length <= 15` splashes on the first moving frame with no visible fall, and the shipped range starts at 1 | `Dynamics2.c:489`; `HouseLegal.c` clamps `length` only from above |
| §11.5 | The swell is rendered as four independent `if`s (B3/B4/B5/B6) | The C is one `if / else if / else if / else if` chain. Behaviour-identical here, but it is the transcription slip the Stage 1.4 audit called the commonest error class, and the ball section three pages earlier does preserve its chains | `Dynamics2.c:481-494` |
| §11.5 | The swell is a visible "3-frame swell animation" | True only for the first drip after a room build: the idle branch never refreshes `whole`, so `RenderDrip`'s `AddRectToWorkRects(whole)` points at the stale aerial sweep and frames 0/1/2 are composited into `workSrcMap` and erased without being blitted | `Dynamics2.c:475-496` |
| §11.4 | Ball period ≈`4v` frames, `length = 64` → ≈32 frames ≈1.06 s | The flight is exactly `4v + 2` = 34 frames ≈1.13 s and the apex is `v*(v+1)` = 72 px, not the requested 64 | `Dynamics2.c:358-423` |
| §8.3 | The half-rate solver table starts at `length = 1` | Omits `length = 0`, which the do-while still resolves to velocity 1 — and §11.6 records the shipped `kFish.length` range as 0..261, so zero occurs in real houses. The full-rate table likewise omits `height <= 0` (not a shipped concern) | `Dynamics3.c` solvers |
| §11.2, §11.1 | (omission) | The spawn-on-recycle-boundary trap is found for the dart but not noted for the balloon (`dest.bottom == kBalloonStart == 310`) or the copter (`dest.top == kCopterStart == 8`), where the identical move-before-test argument is load-bearing | `Dynamics3.c:429`, `:458`; `Dynamics2.c:30-355` |
| `:1097-1098` | The switch→hazard chain always ends in `dinahs[dynaNum]` | For the six switch types `dynamicNum` is `masterObjects[i].hotNum`, so `dynaNum` is a **hot-spot** index there and `TriggerSwitch(dynaNum)` → `hotSpots[dynaNum]` is correct rather than a bug. No doc states the overload, which invites a porter to "fix" it | `ObjectDrawAll.c:518`, `:531`, `:544`, `:557`, `:570`, `:574`; `Trip.c:148` |
| `:458`, `:1415` | `FlagStillOvers` runs on room entry | Same correction as the `interactions.md` row | `Player.c:954` |

**`determinism.md` §4.1** (found by `live-state-inventory`)

| Where | Wrong claim | Correct claim | Citation |
|---|---|---|---|
| §4.1 "Declared" column | `onePlayerLeft`, `playerDead`, `shadowVisible` listed as "(Play)"/"(Room)" | All three are `Player.c:53` | `Player.c:53` |
| §4.1 | `theScore` listed as "(Play)" | `Player.c:50` | `Player.c:50` |
| §4.1 | `numStarsRemaining` listed as "(Play)" | `Banner.c:28` | `Banner.c:28` |
| §4.1 | `numLights`, `localNumbers[9]`, `isStructure[9]` listed as "(Room)" and `numNeighbors` as "(Main/prefs)" | All four are `RoomGraphics.c:31-33` | `RoomGraphics.c:31-33` |
| §4.1 reset row | `numShredded`'s reset names only `RemoveShreds` | Omits `ZeroFlamesAndTheLike`, which is the room-change reset and therefore the one that matters for scoping | `DynamicMaps.c:795` |

**`progression.md` §8-§17** (found by `room-transitions`)

| Where | Wrong claim | Correct claim | Citation |
|---|---|---|---|
| §9.5 (`:3450-3455`) | In `FollowTheLeader`, `takingTheStairs` is false, so the arrival uses the plain `OffsetGlider` path, "a visible difference from a normal both-players-took-the-stairs transition" | Backwards. The leader never changed rooms — it went into limbo — so no `MoveRoomToRoom`, no `ReadyLevel`, no `DrawLocale`, so `takingTheStairs` is still true and the arrival **does** take `ReadyGliderForTripUpStairs`. There is no difference; it is the same path | `Player.c:344`, `:359-362`; `Transit.c:225` |
| §8.11 (`:3167`) | "In two-player mode, the glider that did *not* initiate the transit is put to sleep for 30 frames" | The test is `(twoPlayerGame) && (thisGlider->which != firstPlayer)`, and `firstPlayer` is the glider that went into limbo first, so the glider idled is the follower — which **is** the one that initiated the actual transit | `Transit.c:145-146`; `Modes.c:467` |
| §8.1 (`:2337-2352`) | The two-player dispatch is `elif twoPlayerGame: CheckEscapeUpTwo(...)` | The C is `else if ((twoPlayerGame) && (!onePlayerLeft))` in all four branches. Dropping `!onePlayerLeft` traps the survivor in a room forever; Stage 1.4's `escape.go:447` already has it right | `Interactions.c:701`, `:714`, `:731`, `:745` |
| §8.3 (`:2504-2506`) | `QSetRect(&enterRect, …)`/`QOffsetRect(…)` placed after the two-player `if`/`else` as a shared step | The C duplicates the pair inside both branches, twice over. Semantically equivalent, but it is why §8.3 cannot be used to reconstruct the C's shape | `Transit.c:174-176`, `:183-185`, `:206-208`, `:215-217` |
| §17.1 | `SetObjectsToDefaults` "is called from exactly one place: `NewGame(kNewGameMode)`" | The C tests the negation: `if (mode != kResumeGameMode) SetObjectsToDefaults();`. True in practice, but state the guard, not the conclusion — `NewGame` has a third branch that implies the author expected more modes | `Play.c:102-103`, `:179-182`; `GliderDefines.h:616-617` |
| §17.2 | `numHotSpots` | The global is `nHotSpots` | `Objects.c:76`; `ObjectRects.c:279-288`; `Interactions.c:1632`, `:1719` |
| §10.3 | "glider `enteredRect` — Room change: set" | True only for `MoveRoomToRoom`; for the three link transitions it is set only in the `kLinkedToOther` arm and otherwise deferred to `FinishGliderMailingLeft`/`Right`/`DuctingIn`. §8.11 says this correctly, so §10.3 contradicts §8.11 | `Transit.c:88`; `Player.c:883`, `:921`, `:953` |
| §10.3 | "`otherPlayerEscaped` — cleared to `kNoOneEscaped` on a successful move" | Cleared by the *caller* before the call, never by the transit functions; and in `OffAMortal`'s `mortals == -1` re-dispatch it is set to `kPlayerIsDeadForever` after the move | `Interactions.c:188`; `Player.c:357`, `:627`, `:735`, `:1602` |
| §10.3 | "glider `facing` — forced by `Insure*` on side exits" | Incomplete: `ReadyGliderForTripUpStairs` forces `kFaceLeft`, `…DownStairs` forces `kFaceRight`, and `StartGliderMailingOut` picks facing from the destination slot type | `Modes.c:526`, `:557`, `:188-206` |
| §10.3 | (missing row) | `activeRectEscaped` appears in §17.3's "persists for the whole game" list but has no §10.3 row, so the doc never says `NewGame` fails to initialise it — as it also fails to initialise `firstPlayer`, `playerDead` and `takingTheStairs` | `Play.c:112-118` |
| §8.9 | The three link transitions are "byte-for-byte identical" | Functionally identical, not textually: `TransportRoomToRoom` has no blank line where the other two do. Immaterial; fix so a reader who diffs them does not distrust the doc | `Transit.c:323-324`, `:361-363`, `:400-402` |
| §8.6 | (omission) | `DrawRoomBackground`'s `where == kCentralRoom` block runs **before** both of the function's early returns, so `thisBackground` and `thisTiles` are correct even for an unlit or empty central room — load-bearing for the dirt-room escape checks, and its absence is how a port ends up "tidying" the block after the returns | `RoomGraphics.c:169-177`, `:179-191`, `:193-207` |
| §8.11 | (omission) | `CenterRectInRect` divides a possibly-negative numerator by 2 with C truncation toward zero, which is the difference between `/2` and `>>1` for the ceiling-duct arm | `RectUtils.c` |

**Gaps rather than errors** — material that is in the C and in no `docs/analysis` file. Worth
adding to the named document when the batch is applied, since the Stage 1.5 spec is not where a
future reader will look for them: the missing `StopStar` on the switch-dissolve path, the
"strip slots have a bogus `dest`" invariant, the reason the floor-support cull exists, and the
`RestoreEntireGameScreen` slot-doubling (all `rendering.md`); the 4-frame dead window with
`isOn == false`, the 2-px lead of the hot rect over the paint, the 6-px displacement between the
static hot rect and the static paint, the uninitialised `grease[].hotNum` and
`RedrawAllGrease`'s unguarded read of it, the dead `kGreaseFalling` arm of `ReBackUpGrease`'s
guard, that a lighting change does not re-back-up a spreading spill, that `HandleSwitches`'
grease case never calls `SetObjectState` and so does not persist, that `grease[]` holds only
jars upright at room entry, that `HandleBands`/`HandleTriggers` run while `gameOver`, and
`HouseLegal.c:268-278`'s parity forcing (all `interactions.md` §18-§19); the enemies'
registration **without** a `SectRect` gate versus the appliances' **with** one, which is the whole
reason the screen-size dependence affects appliances and not enemies (`object-draw-all.md`,
`ObjectDrawAll.c:658-794` vs `:796-860`); the `SetObjectState` negative-index reachability result
with its five switch and twenty trigger sites and the `roomType`/`houseType` byte-offset
derivation of what those reads hit (no owner — `determinism.md` §4 is the closest); the
`hotNum != -1` guard missing from the shredder write, `who->isOn = false` sitting outside the
`SetObjectState` guard in all ten live reward cases, `HandleSwitches` setting `stillOver` outside
its guard, `ArmTrigger` latching even when `FindEmptyTriggerSlot` returns -1, and
`kGliderInLimbo` being a no-op in `HandleGlider` so a limbo glider keeps colliding
(`interactions.md` §22); and `Objects.c:411-414`'s `kBlowerOn`/`kBlowerOff` — the only
`PlayPrioritySound` in `Objects.c`, and the audible feedback for every vent and fan switch in the
game — which appears in no report and no document.

### 6.7 Corrections to comments in already-committed Go

Not code defects; comments that will mislead the Stage 1.5 transcription, which is the failure
mode Stage 1.4's audit hit thirteen times.

| File | Wrong claim | Correct claim | Citation |
|---|---|---|---|
| `internal/game/player/env.go:103-105` | "`FlagStillOvers` re-tests what the glider is standing on after it arrives somewhere without moving, so a glider dropped onto a switch triggers it." | It sets `stillOver = true` and `HandleSwitches` early-returns on `stillOver`, so it **suppresses** the switch. It also has exactly one caller, a ceiling-duct arrival — not room entry | `Interactions.c:1725`, `:990`; `Player.c:954` |
| `internal/game/player/env.go:194-195` | `IsShadowVisible()` returns whatever `SetShadowVisible` stored | Conflates the pure recompute (`Room.c:1103`) with the cached global `shadowVisible` (`Player.c:53`). Split or document — see §6.4 item 10 | `Room.c:1103`; `Player.c:53` |
| `internal/game/player/glider.go:133` | `Sliding` is "set for one frame when standing on grease" | The following sentence saves it and `handle.go:157-159` is precise, but the summary line is loose: the flag survives indefinitely if the glider is not in `kGliderNormal` | `Interactions.c:1376-1379`; `Player.c:151-200` |
| `internal/game/player/glider.go:87` | (deviation, admitted in the comment) | `rightClip`/`leftClip` are one shared pair in the C. Either restore the sharing or expand the comment to say the deviation is unobservable and why — see §6.4 item 4 | `Player.c:52` |
| `internal/game/player/env.go` (whole) | — | Not an error, but the claim that "every one of the ~30 method names corresponds to a real C free function" does not hold for the interface as it stands: 46 methods, of which roughly twenty are global reads and writes with no C function behind them (`BatteryTotal`/`SetBatteryTotal`, `Tile`, `TopOpen`, `LeftThresh`, `SetTakingTheStairs`, `TwoPlayerGame`, …). State it as "the function-shaped methods match a C free function; these ~20 accessors stand in for direct global access" | `env.go` |

**All five rows are resolved as of Stage 1.5a.** `FlagStillOvers` and the `env.go` interface
preamble were rewritten during 1.5's implementation; `IsShadowVisible`/`SetShadowVisible` were
split out into their own comment naming all four writers of the cached global, including the one
that forces it false (`Player.c:1286` — the shredder, *not* a transport, which is what the first
draft of that comment got wrong); `RightClip`/`LeftClip` states the deviation, why it is
unobservable in the original's configuration, and why Stage 3 needs it anyway. The `Sliding`
summary line was replaced with the two facts that make it not a one-frame flag — only mode Normal
clears it, and `CheckRoofCollision` runs in four modes and reads it — pinned by
`TestSlidingOutlivesItsFrame`.

`internal/render/locale.go:1079-1084` was checked and is **correct**: it returns 0 from
`GetNumberOfLights` for a nil room, with a comment naming the C's `rooms[-1]`. The C really does
read out of bounds there (`Room.c:1038` with `where == kRoomIsEmpty` from `Room.c:617`) and the Go
behaviour is both safe and unobservable, because `DrawRoomBackground` and `DrawARoomsObjects` bail
for `kRoomIsEmpty` before the count can be read. Leave it alone.

### 6.8 The eight raw reports, and the four disagreements not to re-import

`docs/analysis/stage15-raw/*.md` are working notes, not deliverables. The three critiques'
correction lists were applied while the sections above were written; the raw files are left as
they are. Four items are recorded here because they are the ones most likely to be re-imported by
someone who opens a raw report later and copies from it:

1. **`bands-grease.md` carries two line-number conventions.** Its `Grease.c` and
   `RubberBands.c` citations are exact; its `RoomGraphics.c` citations are uniformly one low, and
   two of them land on the **wrong call** — its "`KillAllBands() :52`" is `ZeroDinahs()` and its
   "`ZeroTriggers() :54`" is `ZeroMirrorRegion()`. Correct: `DrawLocale` `:44-130`,
   `ZeroFlamesAndTheLike()` `:51`, `ZeroDinahs()` `:52`, `KillAllBands()` `:53`,
   `ZeroMirrorRegion()` `:54`, `ZeroTriggers()` `:55`, `ListAllLocalObjects()` `:72`, central room
   `:118-121`, `ReadyLevel` `:402-418`, `RedrawRoomLighting` `:434-461`. It also drops two of
   `ReadyLevel`'s five steps: the order is `NilSavedMaps()` `:404` → [`COMPILEQT` `StopMovie`
   block `:406-413`] → `DetermineRoomOpenings()` `:415` → `DrawLocale()` `:416` →
   `InitGarbageRects()` `:417`. Take `RoomGraphics.c` and `Render.c` numbers from the other six
   reports, `Grease.c`/`RubberBands.c` from this one.
2. **`live-state-inventory.md`'s `objDataType` is 36 bytes; it is 26** (7 shorts plus a 12-byte
   `objectType`, `GliderStructs.h:105`). 36 is `dynaType`. `room-transitions.md`'s "8 shorts" is
   also wrong — seven.
3. **`live-state-inventory.md`'s `InitGarbageRects` ends with `nextFrame = TickCount()`; the C is
   `nextFrame = TickCount() + kTicksPerFrame;`** (`Render.c:690`, verified). Without the addend the
   first frame after every room load busy-waits zero ticks.
4. **`live-state-inventory.md` calls the draw order "the reverse of the build order"; it is not.**
   Reversing the build order (`Objects.c:312-328`) gives NW, SW, S, SE, NE, N, W, E, Central,
   while the draw order (`RoomGraphics.c:80-120`) is NW, NE, N, SW, SE, S, W, E, Central. The only
   true statement is that central is first in the build order and last in the draw order. This
   matters because the draw order fixes `dinahs[]`/`savedMaps[]`/`grease[]` slot indices, and the
   slot index decides both the iteration order in `HandleDynamics` and which entries are dropped
   at saturation.

Two smaller ones, for completeness: `dynamics-core.md` cites `Player.c:90` for
`vDesiredVel = kGravity` (it is `:92`; `:90` is the clamp inside the ramp, which asserts the
opposite), and `dynamics-core.md` says "nine appliance handlers, eight share one skeleton, two do
not" — the timer skeleton is shared by **six** (`HandleMacPlus`, `HandleTV`, `HandleCoffee`,
`HandleVCR`, `HandleStereo`, `HandleMicrowave`), while `HandleSparkleObject` opens on `active`,
`HandleToast` on `moving` and `HandleOutlet` on `position != 0`.
