# Glider PRO 1.0.4 — Part 10: Networking a Deterministic Go Port

**This file is Part 10 of `docs/analysis/determinism.md`.** That document ends at §9.8 and
forward-references five sections that live here:

| Forward reference | From | Lands at |
| --- | --- | --- |
| "split sim/render (§10.1)" | `determinism.md:3138` (hazard 13) | [§10.1](#101-the-simulationrender-cut) |
| "see §10.3" / "split into named streams (§10.3)" / "Fixing it is not optional for lockstep, and it is not hard (§10.3)" | `determinism.md:3126`, `:3128`, `:3133`, `:3177` (hazards 1, 3, 8; §7.3) | [§10.3](#103-rng-own-prng-named-streams-and-decoupling-from-rendering) |
| "Use a per-frame bitmask instead (§10.4)" | `determinism.md:3351` (§8.4.1) | [§10.4](#104-the-wire-format) |
| "A networked progress packet is exactly the missing piece, and §10.4 specifies one" | `determinism.md:3679` (§9.7) | [§10.4.8](#1048-the-progress-packet--the-two-player-save-the-original-never-had) |
| "…'each machine simulates its own player' (mode (a)) is not a straight code split. §10.5 addresses that." | `determinism.md:3707` (§9.8) | [§10.5](#105-the-authoritative-fix-for-the-shared-handleinteraction-problem) |

Section numbering continues `determinism.md`'s scheme: §10.x here, §1..§9 there. Bare
section references without a file name (e.g. "§3.9") mean `determinism.md`. Citations use
the same convention as the rest of the analysis: `GliderPRO/Sources/File.c:NNN` with line
numbers taken from a CR→LF-converted copy of the original file.

Everything in Part 10 is a *design* derived from evidence in Parts 1–9 plus additional
source reading done specifically for this part. **There is no networking code anywhere in
the Glider PRO source tree** — no sockets, no AppleTalk, no ADSP, no PPC toolbox, no
`OpenTransport`. The two-player mode is shared-keyboard, single-process, one simulation
(§9). Every wire format, message and packet in this document is therefore *new*, and is
justified only by how well it matches structures the original already has.

## Sources read for Part 10

Beyond the files listed in `determinism.md`'s "Sources read", Part 10 rests on a full
re-reading of:

| File | Converted lines | What was used |
| --- | --- | --- |
| `GliderPRO/Sources/Play.c` | 821 | `PlayGame` :430-599, `InitTelephone` :733-740, `HandleTelephone` :744-789, `NewGame` reset block :112-118, `DoDemoGame` :282-303 |
| `GliderPRO/Sources/Input.c` | 398 | `LogDemoKey` :44-49, `DoPause` :77-117, `DoBatteryEngaged` :121-156, `DoHeliumEngaged` :160-182, `GetDemoInput` :186-277, `GetInput` :281-379 |
| `GliderPRO/Sources/Interactions.c` | 1777 | `CheckEscapeUpTwo` :171-240, `HandleSwitches` :985-1156, hotspot dispatch (`HandleHotSpotCollision`) :1198-1623, `CheckForHotSpots` :1627-1687, `HandleInteraction` :1691-1711, `HandleRewards` :756-981 |
| `GliderPRO/Sources/Player.c` | 1605 | stairs/transport/duct/mail completions, `HandleGlider` :1335-1439, `OffAMortal` :1484-1604 |
| `GliderPRO/Sources/Transit.c` | 558 | `MoveRoomToRoom` :151-310, `ForceKillGlider` :449-469, `FollowTheLeader` :473-557 |
| `GliderPRO/Sources/Render.c` | 772 | `RenderFrame` :639-671, `InitGarbageRects` :675-691 |
| `GliderPRO/Sources/Grease.c` | 302 | `HandleGrease` :43-133, `AddGrease` :206-251, `RedrawAllGrease` :270-301 |
| `GliderPRO/Sources/DynamicMaps.c` | 798 | `BackUpToSavedMap` :70-93, `RestoreFromSavedMap` :131-163, the five `Add*` RNG sites |
| `GliderPRO/Sources/ObjectDrawAll.c` | 966 | `DrawARoomsObjects` :23-966, all 7 `BackUpToSavedMap` sites, the `isLit` gates |
| `GliderPRO/Sources/RoomGraphics.c` | 462 | `DrawLocale` :44-130, `DrawRoomBackground` :162-253, `RedrawRoomLighting` :434-461 |
| `GliderPRO/Sources/Objects.c` | 1001 | `IsThisValid` :89-122, `SetObjectState` :366-699 |
| `GliderPRO/Sources/Dynamics.c` | 776 | the appliance handlers' direct blits |
| `GliderPRO/Sources/Triggers.c` | 205 | `trigType` declaration :15-21 |
| `GliderPRO/Headers/GliderStructs.h` | 347 | `gliderType` :200-216, `demoType` :334-339, `hotObject` :218-225 |
| `GliderPRO/Headers/GliderDefines.h` | 625 | glider modes :571-594, escape states :596-608, `kDemoLength` :625 |
| `GliderPRO/Glider PRO.r` | 199843 | `data 'demo' (128)` block beginning at line **199389**, re-parsed with python3 |

## 10.0 The seven questions Part 10 must answer

| # | Question | Section | One-line answer |
| --- | --- | --- | --- |
| 1 | Lock-step or rollback? | §10.2 | **Input-delay lock-step, 2 frames of delay.** Rollback is impossible without rewriting the room-load path. |
| 2 | Where is the sim/render cut? | §10.1 | Not at `RenderFrame`. Cut *inside* it, and also inside `HandleDynamics`, `HandleInteraction` and `HandleGlider`, all four of which blit. |
| 3 | How to decouple RNG from `GetNumberOfLights` (hazard 3/25)? | §10.3.3 | `numLights` is *not* the mechanism — `IsThisValid` and the 24-slot saved-map budget are. Make the room-load draw schedule a pure function of the room's *logical* contents. |
| 4 | How to decouple RNG from render-time decisions? | §10.3.4-10.3.5 | Draw first, allocate second: every `Add*` must consume its RNG before any resource guard can fail. |
| 5 | Concrete input-frame wire format? | §10.4.2-10.4.4 | 1 byte per player per frame (4 live bits), `gameFrame` as the sequence number, no per-record frame field. 3369 bytes for the whole shipped demo vs 6702. |
| 6 | `otherPlayerEscaped` rendezvous → messages? | §10.6 | Under lock-step: **zero messages.** It is derived state. Only a non-lock-step topology needs `Rendezvous` messages, and §10.6.6 specifies them. |
| 7 | Authoritative fix for shared `HandleInteraction`? | §10.5 | Do **not** split it. Keep one whole-world simulation on every peer and make the *inputs* the only thing that crosses the wire. §10.5.4 gives the reordering that makes it safe. |

---

# 10.1 The simulation/render cut

## 10.1.1 The problem, restated with its full extent

§3.9 establishes that `RenderFrame` mutates simulation state. That is true but it
understates the problem. The correct statement is:

> **Glider PRO has no simulation/render boundary at all.** Four of the eleven phases in
> `PlayGame` (§3.1) contain both simulation writes and framebuffer writes, interleaved
> statement by statement. Phase 9 (`RenderFrame`) is the worst but it is not unique.

Measured by call sites, from `/usr/bin/grep -n` over CR→LF copies:

| Phase | Function | Blit/graphics calls inside it | Evidence |
| --- | --- | --- | --- |
| 3 | `HandleDynamics` → `Dynamics.c` handlers | **51** (`CopyBits`, `CopyMask`, `PaintRect`, `AddRectToWorkRects`) | `GliderPRO/Sources/Dynamics.c:129`–`:767` |
| 3 | `HandleDynamics` → `Dynamics2.c` handlers | 5 (`AddRectToWorkRects`) | `GliderPRO/Sources/Dynamics2.c:94`, `:196`, `:302`, `:458`, `:532` |
| 5 | `HandleInteraction` → `HandleRewards` | 10 × `RestoreFromSavedMap` + 10 × `RedrawAllGrease` | `GliderPRO/Sources/Interactions.c:771`–`:973` |
| 5 | `HandleInteraction` → `HandleHotSpotCollision` → `HandleSwitches` | `CopyRectBackToWork` + `AddRectToWorkRects` + 2 × `RestoreFromSavedMap(..., true)` | `GliderPRO/Sources/Interactions.c:1033`, `:1034`, `:1049`, `:1054` — all inside **`HandleSwitches` (`:985-1156`)**, not `HandleHotSpotCollision` itself; reached from the `kSwitchIt` case (label `:1320`, call `:1321`) |
| 7 | `HandleBands` → `HandleSwitches` | same four calls as above | `GliderPRO/Sources/RubberBands.c:133` — a band hitting a `kSwitchIt` hotspot runs the *same* switch body one phase earlier, so the port must place `HandleSwitches` where both callers can reach it |
| 8 | `HandleGlider` → transporter/duct/mail | **10** × `CopyRectWorkToMain` | `GliderPRO/Sources/Player.c:603`, `:606`, `:711`, `:714`, `:808`, `:811`, `:1007`, `:1010`, `:1098`, `:1101` |
| 8 | `HandleGlider` → `Modes.c` (`StartGliderFadingOut`, `:75`) | 2 × `AddRectToWorkRects` | `GliderPRO/Sources/Modes.c:91`, `:96` |
| 7 | `HandleBands` | 1 × `AddRectToWorkRects` | `GliderPRO/Sources/RubberBands.c:231` |
| 9 | `RenderFrame` | everything (§3.9) | `GliderPRO/Sources/Render.c:639-671` |
| 10 | `HandleDynamicScoreboard` | 28 | `GliderPRO/Sources/Scoreboard.c` |
| — | `MoveRoomToRoom` (called from phases 5 and 8) | `WipeScreenOn` + an extra whole `RenderFrame()` | `GliderPRO/Sources/Transit.c:300`, `:303` |

`Dynamics3.c` (the dispatcher, `GliderPRO/Sources/Dynamics3.c:31-105`) and `Triggers.c`
contain **zero** graphics calls. `Objects.c` has exactly one, `DrawThisRoomsObjects()` at
`GliderPRO/Sources/Objects.c:995`, inside `BringSendFrontBack` — which sits in the
`#ifndef COMPILEDEMO` editor-only block (`:875-1000`) and also calls `UpdateMenus`,
`InvalWindowRect` and `DeselectObject`, so it is unreachable during play. Those three files
are already clean for porting purposes.

## 10.1.2 The two directions of coupling, and only one of them matters

There are two distinct couplings, and a port must treat them completely differently.

**Coupling A — simulation writes inside render code.** `RenderFrame` advances animation
state machines and can spawn entities (§3.9.1). This is the one that matters, because it
means *skipping the render changes the simulation*.

**Coupling B — render writes inside simulation code.** `HandleToast` blits a toaster
sprite from inside phase 3. This is *harmless to determinism*: the blit does not feed
back. The only exception is `BackUpToSavedMap`, which reads pixels back
(`GliderPRO/Sources/DynamicMaps.c:84-86`), and it is a room-load operation, never
per-frame (§3.9.3 makes the same point).

So the port's job is:

* **Coupling A: must be fixed.** Move the mutations out of the draw path.
* **Coupling B: just delete the graphics calls from the sim functions** and re-derive the
  draw from the state they already wrote. No behaviour changes because nothing reads pixels.

## 10.1.3 Complete inventory of Coupling A

Every simulation write reachable from `RenderFrame` (`GliderPRO/Sources/Render.c:639-671`),
in execution order, with the exact state written. This extends §3.9.1 with the sites §3.9.1
summarised.

| Order | Call | Line | Simulation state written | Can it spawn? |
| --- | --- | --- | --- | --- |
| 1 | `DrawReflection(&theGlider, true)` | `Render.c:643` | none (pure draw) | no |
| 2 | `DrawReflection(&theGlider2, false)` | `Render.c:644-645` | none | no |
| 3 | `HandleGrease()` | `Render.c:647` | `grease[i].frame`, `.mode`, `.start`, `.dest`; **`hotSpots[grease[i].hotNum].action = kSlideIt`, `.isOn = true`, `.bounds`**, and `.bounds.right += 2` / `.bounds.left -= 2` per frame | **yes — collision geometry** |
| 4 | `RenderPendulums()` | `Render.c:648` | global `clockFrame`; `pendulums[i].mode`, `.src`, `.toOrFro`; global `playedTikTok` | no |
| 5a | `RenderFlames()` (even frames) | `Render.c:650` | `flames[i].mode`/`.src`, `tikiFlames[i].mode`/`.src`, `bbqCoals[i].mode`/`.src` | no |
| 5b | `RenderStars()` (odd frames) | `Render.c:652` | `theStars[i].mode`, `.src` | no |
| 6 | `RenderDynamics()` | `Render.c:653` | none (pure draw, `Dynamics3.c:112-154`) | no |
| 7 | `RenderFlyingPoints()` | `Render.c:654` | `flyingPoints[i].dest`, `.mode`, `.loops`, `.whole`; global `numFlyingPts` | no (retires only) |
| 8 | `RenderSparkles()` | `Render.c:655` | `sparkles[i].mode`, `.bounds`; global `numSparkles` | no (retires only) |
| 9 | `RenderGlider(&theGlider, true)` | `Render.c:656` | `theGlider.frame` for animated modes | no |
| 10 | `RenderGlider(&theGlider2, false)` | `Render.c:657-658` | `theGlider2.frame` | no |
| 11 | `RenderShreds()` | `Render.c:659` | `shreds[i].frame`, `.bounds`; **calls `AddSparkle(&shreds[i].bounds)`** at `GliderPRO/Sources/Render.c:603` | **yes — spawns a sparkle** |
| 12 | `RenderBands()` | `Render.c:660` | none (pure draw) | no |
| 13 | `while (TickCount() < nextFrame) {}` | `Render.c:662-664` | none — pacing | no |
| 14 | `nextFrame = TickCount() + kTicksPerFrame` | `Render.c:665` | `nextFrame` (**not** sim state) | no |
| 15 | `CopyRectsQD()` | `Render.c:667` | none | no |
| 16-17 | `numWork2Main = 0; numBack2Work = 0` | `Render.c:669-670` | dirty-rect counters (render state) | no |

**Nine of the seventeen steps write simulation state.** Two can spawn entities. One
(`HandleGrease`) creates collision geometry that phase 5 of the *next* frame will read.

## 10.1.4 `HandleGrease` is the load-bearing case

`GliderPRO/Sources/Grease.c:43-133` is the single reason you cannot simply "not call
`RenderFrame` on a headless peer". Transcribed, with the simulation writes marked:

```
HandleGrease():                                        // Grease.c:43
   1. if numGrease == 0 then return                    // :48-49
   2. for i = 0 .. numGrease-1:                        // :51
   3.     if grease[i].mode == kGreaseFalling (1) then  // :53
   4.         grease[i].frame++                                       // SIM  :55
   5.         if grease[i].frame >= 3 then                            //      :56
   6.             grease[i].frame <- 3                                // SIM  :58
   7.             grease[i].mode  <- kGreaseSpreading (2)             // SIM  :59
   8.             hotSpots[grease[i].hotNum].action <- kSlideIt        // SIM  :60
   9.             hotSpots[grease[i].hotNum].isOn   <- true            // SIM  :61
  10.             if grease[i].isRight then src <- (0,-2,2,0)          //      :63
  11.             else                      src <- (-2,-2,0,0)         //      :65
  12.             QOffsetRect(&src, -playOriginH, -playOriginV)        //      :66
  13.             QOffsetRect(&src, grease[i].start, grease[i].dest.bottom)  // :67
  14.             hotSpots[grease[i].hotNum].bounds <- src             // SIM  :68
  15.         (blit savedMaps[grease[i].mapNum] frame -> work and back) // DRAW :71-80
  16.         AddRectToWorkRects(&grease[i].dest)                       // DRAW :82
  17.         grease[i].dest offset by ±2 horizontally                 // SIM  :83-86
  18.     else if grease[i].mode == kGreaseSpreading then               //      :88
  19.         if grease[i].isRight then
  20.             src <- (0,-2,2,0) offset by (start, dest.bottom)      //      :92-93
  21.             grease[i].start += 2                                  // SIM  :94
  22.             hotSpots[grease[i].hotNum].bounds.right += 2           // SIM  :95
  23.         else
  24.             src <- (-2,-2,0,0) offset by (start, dest.bottom)     //      :99-100
  25.             grease[i].start -= 2                                  // SIM  :101
  26.             hotSpots[grease[i].hotNum].bounds.left -= 2            // SIM  :102
  27.         PaintRect(&src) into backSrcMap and workSrcMap            // DRAW :111-115
  28.         AddRectToWorkRects(&src)                                  // DRAW :116
  29.         if isRight  and start >= stop then mode <- kGreaseSpiltIdle (3)  // SIM :123-124
  30.         if !isRight and start <= stop then mode <- kGreaseSpiltIdle      // SIM :128-129
```

Grease mode constants (`GliderPRO/Sources/Grease.c:17-20`):

| Constant | Value |
| --- | --- |
| `kGreaseIdle` | 0 |
| `kGreaseFalling` | 1 |
| `kGreaseSpreading` | 2 |
| `kGreaseSpiltIdle` | 3 |

The spill spreads **2 pixels per frame** in a fixed direction until `start` crosses `stop`,
where `stop = src.right + distance` (rightward) or `src.left - distance` (leftward) and
`distance` comes from the object's `data.c.length` byte in the house file
(`GliderPRO/Sources/Grease.c:236-243`). A glider that touches the resulting hotspot gets
`action == kSlideIt`, which sets `sliding = true` and snaps `vVel` to the hotspot's top
(`GliderPRO/Sources/Interactions.c:1376-1378`).

**Therefore a peer that does not render does not grow the grease, and the two peers
disagree about where the slippery floor is within two frames of a jar tipping.** That is a
hard desync, not a cosmetic one.

Note also `RedrawAllGrease` (`GliderPRO/Sources/Grease.c:270-301`), called from ten places
in `HandleRewards`: it is pure draw (it reads `hotSpots[grease[i].hotNum].bounds` and paints
it black) and can be moved wholesale into the renderer. Its guard is a **three-part**
condition worth preserving for a port that reuses the code
(`GliderPRO/Sources/Grease.c:283-285`): it paints only when
`grease[i].where == thisRoomNumber` **and** `(src.bottom - src.top) == 2` **and**
`grease[i].mode != kGreaseIdle`. There is also an early `if (numGrease == 0) return;`
at `:277-278`.

## 10.1.5 The cut

Split `RenderFrame` into two functions and call them from different places:

```
// PHASE 9a — part of the simulation step. Deterministic. No I/O. No pixels.
SimStepPart2():
   1. AdvanceGrease()        // Grease.c:43-133 with steps 15,16,27,28 deleted
   2. AdvancePendulums()     // Render.c:260-320 minus the CopyBits
   3. if evenFrame then AdvanceFlames() else AdvanceStars()
   4. AdvanceFlyingPoints()  // Render.c:325-380 minus the CopyBits
   5. AdvanceSparkles()      // Render.c:384-418 minus the CopyBits
   6. AdvanceGliderAnim(&theGlider)
   7. if twoPlayerGame then AdvanceGliderAnim(&theGlider2)
   8. AdvanceShreds()        // Render.c:559-614; still calls AddSparkle at frame 20

// PHASE 9b — presentation only. May be skipped, throttled, or run on another goroutine
// against a snapshot. Writes nothing the simulation reads.
Draw():
   1. if hasMirror then DrawReflection(p1); DrawReflection(p2)
   2. DrawGrease() ; DrawPendulums() ; DrawFlamesOrStars()
   3. DrawDynamics() ; DrawFlyingPoints() ; DrawSparkles()
   4. DrawGlider(p1) ; DrawGlider(p2) ; DrawShreds() ; DrawBands()
   5. DrawScoreboard()
   6. Present()
```

**The order inside `SimStepPart2` must be exactly the order in
`GliderPRO/Sources/Render.c:641-660`.** It is observable: `AdvanceShreds` can call
`AddSparkle`, and `AdvanceSparkles` has already run this frame, so a sparkle spawned by a
shred is first animated on the *next* frame. Reordering steps 5 and 8 changes sparkle
lifetimes by one frame.

`kTicksPerFrame` pacing (steps 13-14) moves out of both functions and into the loop
(§10.2.5). `CopyRectsQD` and the two dirty-rect counter resets move into `Draw`.

## 10.1.6 Why the mechanical split works

In every one of the nine mutating steps, the mutation and the blit are *adjacent but
independent*: the blit reads `src`/`dest` rects that the mutation just computed, and nothing
in the mutation reads a pixel. Concretely, for `RenderPendulums`
(`GliderPRO/Sources/Render.c:260-320`) the pattern is

```
clockFrame++;                      // mutation
...
pendulums[i].src advanced ±28;     // mutation
CopyBits(savedMaps[...], ..., &pendulums[i].src, &pendulums[i].dest, ...);   // draw
AddRectToWorkRects(&pendulums[i].dest);                                      // draw
```

so `Draw()` can recompute `src` from `pendulums[i].mode` (the sprite index) and `dest`
(fixed at room load) with no additional state. The same holds for flames, coals, tikis,
stars, flying points and sparkles: each keeps a `mode` integer that *is* the sprite index,
and a `dest` rect that never moves except for flying points, whose `dest` is simulation
state anyway.

`RenderShreds` is the one place where the port must be careful: the sparkle spawn at
`GliderPRO/Sources/Render.c:603` is inside the drawing loop and is conditional on
`shreds[i].frame` reaching 20. Keep the spawn in `AdvanceShreds` and keep it conditional on
exactly the same frame number.

## 10.1.7 The three other cuts

**Phase 3 (`HandleDynamics`).** Delete the graphics calls from the 16 handlers. The
appliance handlers (`HandleToast` `GliderPRO/Sources/Dynamics.c:321`, `HandleMacPlus`
`:388`, `HandleTV` `:428`, `HandleCoffee` `:482`, `HandleOutlet` `:528`, `HandleVCR` `:603`,
`HandleStereo` `:671`, `HandleMicrowave` `:715`) each blit `applianceSrcMap → backSrcMap` or
`→ workSrcMap` inline. What determines the sprite is `dinahs[who].frame` and
`dinahs[who].active`, both already simulation state, so the renderer can reproduce the blit
from `dinahs[]` alone. The 5 `AddRectToWorkRects` calls in `Dynamics2.c` are pure dirty-rect
bookkeeping.

**Phase 5 (`HandleInteraction`).** `HandleRewards` calls
`RestoreFromSavedMap(thisRoomNumber, objectNum, doSparkle)`
(`GliderPRO/Sources/DynamicMaps.c:131-163`) ten times. That function does three things:

1. blits the saved background over the prize (draw),
2. `AddRectToWorkRects` (draw),
3. **if `doSparkle` then `AddSparkle(&bounds)` and `PlayPrioritySound(kFadeOutSound, …)`**
   (`GliderPRO/Sources/DynamicMaps.c:153-159`) — a *simulation* spawn.

All ten `HandleRewards` calls pass `doSparkle = false`
(`GliderPRO/Sources/Interactions.c:771`, `:787`, `:803`, `:819`, `:836`, `:855`, `:877`,
`:905`, `:938`, `:961`), so for rewards it is pure draw. The two calls that pass `true` are
in **`HandleSwitches`** (`GliderPRO/Sources/Interactions.c:1049`, `:1054`) — *not* in
`HandleHotSpotCollision`, which only reaches them via the `kSwitchIt` case (`:1320-1322`) —
and those *do* spawn. Note that `HandleSwitches` is also called from `HandleBands`
(`GliderPRO/Sources/RubberBands.c:133`) and from `TriggerSwitch`
(`GliderPRO/Sources/Trip.c:148`), so both spawn sites are reachable from phase 7 as well as
phase 5. Split
`RestoreFromSavedMap` into `EraseObject()` (draw) and an explicit `AddSparkle` at the two
`true` call sites.

Note the loop in `RestoreFromSavedMap` has a `break` at
`GliderPRO/Sources/DynamicMaps.c:160`, so it restores **only the first** saved-map slot
matching `(where, who)`. §10.3.4 shows that stars occupy *two* slots with the same key, so
the second is never restored. Replicate that or fix it, but pick one.

**Phase 8 (`HandleGlider`).** The ten `CopyRectWorkToMain` calls in `Player.c` are
immediate-present calls used by the transporter / duct / mailbox fade animations to punch
the glider out of the visible framebuffer at the exact instant the mode changes. They are
pure presentation; delete them and let `Draw()` handle it, accepting that a port with a
one-frame-later present shows the glider for one extra frame during those transitions.
Cross-check that against §3.8's mode table before deciding it is acceptable — it is a
visible difference, not a simulation one.

## 10.1.8 The extra `RenderFrame()` inside room transitions

`MoveRoomToRoom` ends with

```c
	ReadyLevel();                                     // Transit.c:298
	RefreshScoreboard(kNormalTitleMode);              // Transit.c:299
	WipeScreenOn(where, &justRoomsRect);              // Transit.c:300

#ifdef COMPILEQT
	RenderFrame();                                    // Transit.c:303
	...
#endif
```

and the other three transition functions do the same (`TransportRoomToRoom`
`GliderPRO/Sources/Transit.c:314-348`, `MoveDuctToDuct` `:352-387`, `MoveMailToMail`
`:391-426`). Because `COMPILEQT` **is** defined (§2.3), that `RenderFrame()` is live, so a
room transition executes **one extra `SimStepPart2` worth of animation advance** —
`clockFrame++`, flame modes advance, sparkles age, and `AdvanceGrease` runs (harmlessly,
since `ZeroFlamesAndTheLike` has just set `numGrease = 0` via `DrawLocale`
→ `GliderPRO/Sources/RoomGraphics.c:51`).

This is hazard 14 in §7.2 and is marked BENIGN-but-must-be-replicated. In the split design
it becomes explicit and easy to get right:

```
MoveRoomToRoom(g, where):
   ...
   ReadyLevel()
   SimStepPart2()      // the extra advance the original performs via RenderFrame()
   Draw()              // WipeScreenOn + present
```

Do **not** put the extra `SimStepPart2()` behind a "if rendering enabled" test. That is
precisely the bug this whole section exists to prevent.

## 10.1.9 What must stay on the render side

For completeness, the state that is *only* render state and must be kept out of any
checksum (§10.4.6) and out of any rollback snapshot:

| State | Where | Why it is render-only |
| --- | --- | --- |
| `work2Main[]`, `back2Work[]`, `numWork2Main`, `numBack2Work` | `Render.c:20`-region | dirty rectangles; `kMaxGarbageRects` = 48 (`Render.c:20`), append guarded by `< (kMaxGarbageRects - 1)` i.e. < 47 (`Render.c:67`, `:86`, `:105`) |
| `nextFrame` | `Render.c` | wall-clock pacing deadline (`Render.c:665`, `:690`) |
| `savedMaps[i].map` (`GWorldPtr`) | `GliderStructs.h:230` | Mac offscreen buffer handle |
| `savedMaps[i].dest`, `.where`, `.who` | `GliderStructs.h:229-232` | **NOT render-only** — `numSavedMaps` gates RNG (§10.3.4) |
| `theGlider.src`, `.mask`, `.whole`, `.wholeShadow` | `GliderStructs.h:202-205` | derived from `mode`/`frame`/`dest` (§4.3.6) |
| `playedTikTok` | `Render.c:265` | a once-per-frame sound latch |
| `shadowVisible`, `hasMirror`, `tvInRoom`, `tvWithMovieNumber` | `DrawLocale` `RoomGraphics.c:59`, `:126` | drawing decisions |
| `displayedScore` | Scoreboard | the animated readout; `theScore` is the real value |
| `numTempManholes`, `tempManholes[]` | `RoomGraphics.c:36-37` | floor-support graphics only |

## 10.1.10 Verifying the cut

The split is only correct if it is bit-identical. The test is available for free, and it is
the shipped demo:

1. Implement `SimStepPart2` + `Draw`.
2. Run the `'demo' 128` stream (§10.4.9 gives the transcoding) through the sim with `Draw`
   enabled, hashing the sim state (§10.4.6) every frame; record 3369 hashes.
3. Run it again with `Draw` **never called**.
4. The two hash sequences must be identical.

If they differ, a mutation is still inside `Draw`. This test catches every Coupling-A leak
mechanically and requires no network at all. `determinism.md` §8.7 already argues the demo
proves replay determinism; this turns it into a regression test for the cut.

---

# 10.2 Lock-step vs rollback: the decision

## 10.2.1 What the eleven-phase order actually tells us

The frame order in §3.1 is not arbitrary; it has four properties that decide the networking
model.

**(i) Input is read in the middle of the frame, not at the top.** Phase 4
(`GliderPRO/Sources/Play.c:452-453`) runs *after* phase 3 `HandleDynamics`
(`:449`). Enemies have already moved by the time the glider's keys are sampled. §3.1's own
observation: "There is a systematic one-frame lag baked in and it is asymmetric between the
two entity classes." For a rollback implementation this is the worst possible layout,
because re-simulating a frame with corrected input requires re-running phase 3 as well —
phase 3 is not input-dependent, but it *is* RNG-dependent (`HandleCoffee`'s
`RandomInt(200)`, `GliderPRO/Sources/Dynamics.c:492`, `:506`; `HandleSparkleObject`'s
`RandomInt(240)`, `:304`), so a rollback must also rewind the RNG.

**(ii) Phases 6 and 7 run even when `gameOver` is set; phases 4, 5 and 8 do not.**
(`GliderPRO/Sources/Play.c:450`, `:456`, `:457`, `:458`.) A frame is therefore not a
uniform function — it has two shapes. Any rollback that re-simulates must reproduce the
right shape for the right frame, which means `gameOver` and `countDown` are part of the
rollback state.

**(iii) A single frame can trigger a room transition, and a room transition is a
multi-hundred-line synchronous operation that reallocates offscreen buffers, re-reads the
house handle, consumes RNG, and calls `RenderFrame()` recursively.** `CheckGliderInRoom`
(phase 5) can call `MoveRoomToRoom` (`GliderPRO/Sources/Interactions.c:188`), which calls
`ReadyLevel()` → `NilSavedMaps()` → `DisposeGWorld` for up to 24 GWorlds
(`GliderPRO/Sources/DynamicMaps.c:46-62`) → `DrawLocale()` → up to nine
`GetNumberOfLights` + `DrawRoomBackground` + `DrawARoomsObjects` passes
(`GliderPRO/Sources/RoomGraphics.c:78-121`) → up to 24 `CreateOffScreenGWorld` calls
(`GliderPRO/Sources/DynamicMaps.c:82`). **This is the rollback killer.** Rolling back
across a room transition means undoing all of that, including the RNG draws it made and the
`house` handle mutations `SetObjectsToDefaults`/`SetObjectState` performed.

**(iv) `gameFrame` is a monotone `long` incremented exactly once per iteration**
(`GliderPRO/Sources/Play.c:434`) and there is no catch-up loop — `nextFrame` is *assigned*,
not accumulated (`GliderPRO/Sources/Render.c:665`, hazard 10 in §7.2). So the simulation is
already a pure discrete-time system with a natural sequence number, which is exactly what
lock-step wants.

## 10.2.2 What a rollback port would cost

Rollback (predict remote input, re-simulate on misprediction) requires an
`O(1)`-ish snapshot/restore of the entire simulation state. Part 4 enumerates it. Summing
the array state from §4.2 with the sizes verified under `#pragma pack(2)` and 32-bit `long`
(re-verified for Part 10 with `gcc`):

| Component | Count | Element size | Bytes |
| --- | --- | --- | --- |
| `theGlider` + `theGlider2` | 2 | 110 | 220 |
| `dinahs` | 18 | 36 | 648 |
| `hotSpots` | 56 | 16 | 896 |
| `triggers` | 16 | 12 | 192 |
| `bands` | 2 | 16 | 32 |
| `grease` | 16 | 26 | 416 |
| `flames` | 20 | 20 | 400 |
| `tikiFlames` | 8 | 20 | 160 |
| `bbqCoals` | 8 | 20 | 160 |
| `pendulums` | 8 | 26 | 208 |
| `theStars` | 4 | 24 | 96 |
| `sparkles` | 3 | 10 | 30 |
| `flyingPoints` | 3 | 28 | 84 |
| `shreds` | 4 | 10 | 40 |
| `masterObjects` | 216 | 26 | 5616 |
| `savedMaps` (metadata only) | 24 | 12 | 288 |
| ~60 global scalars (§4.1) | — | — | ~180 |
| **subtotal** | | | **9 666 B ≈ 9.4 KB** |
| `thisRoom` (by-value `roomType` copy; `CopyRoomToThisRoom` `Room.c:343-350`, `CopyThisRoomToRoom` `:354-365`, `ForceThisRoom` `:369-386` with the assignment at `:379`) | 1 | 348 | 348 |
| **the loaded house** (`houseType` header + 348 × `nRooms`) | 1 | 866 + 348·n | 866 + 348·n |

`sizeof(gliderType)` = **110**, verified: `offsetof(dest)` = 16, `offsetof(hVel)` = 80,
`offsetof(mode)` = 92, `offsetof(facing)` = 98, `offsetof(ignoreGround)` = 108. Field list
in §4.3.6; declaration at `GliderPRO/Headers/GliderStructs.h:200-216`.

Element sizes re-verified with `gcc -fpack-struct=2` against the `GliderStructs.h`
declarations: `dynaType` 36 (`:310-320`), `hotObject` 16 (`:218-225`), `trigType` 12
(`Triggers.c:15-21`), `bandType` 16 (`:274-279`), `greaseType` 26 (`:287-295`), `flameType`
20 (`:251-256`), `pendulumType` **26** (`:258-264` — 2 `Rect` + 4 `short` + 2 `Boolean`),
`starType` 24 (`:297-302`), `sparkleType` 10 (`:235-239`), `flyingPtType` 28 (`:241-249`),
`shredType` 10 (`:304-308`), `objDataType` 26 (`:322-332`), `savedType` 16 of which only
**12** bytes are metadata (`dest` 8 + `where` 2 + `who` 2; the 4-byte `GWorldPtr` at `:230`
is a host resource, not state), `roomType` 348 (`:166-180`), `houseType` header 866
(`:182-198`).

The ~10 KB of fixed arrays is trivially snapshottable. **The house is not.** For a
100-room house that is 866 + 34 800 = 35 666 bytes, and it is *mutable game state*, not an
asset: `SetObjectsToDefaults` writes it at game start (`GliderPRO/Sources/Play.c:603-708`,
hazard 24), `SetObjectState` writes `data.c.state` / `data.f.state` on every prize pickup
and switch flip (`GliderPRO/Sources/Objects.c:366-698`), and `HandleRoomVisitation` writes
`rooms[n].visited` (`GliderPRO/Sources/Transit.c:430-445`). It is also the thing
`GetNumberOfLights` and `IsThisValid` read at room load, so its contents change the RNG
draw schedule (§10.3.3).

So a rollback snapshot is ~45 KB for a typical house, which at 8 frames of rollback depth
and 30 Hz is 1.35 MB/s of memcpy — actually fine on modern hardware. **Cost is not the
objection.** The objections are structural.

## 10.2.3 The five structural blockers to rollback

| # | Blocker | Evidence | Why it blocks rollback |
| --- | --- | --- | --- |
| 1 | Room transitions do heavyweight, side-effecting work inside a frame | `GliderPRO/Sources/Transit.c:151-310`; `ReadyLevel` `RoomGraphics.c:402-418`; `NilSavedMaps` `DynamicMaps.c:46-62` | re-simulating a frame that transitioned means disposing and recreating up to 24 offscreen buffers and re-running `DrawLocale`; the RNG draws it makes must also be rewound |
| 2 | The RNG is a single global stream with no rewind | `Random()` is an OS trap with no accessible state (`GliderPRO/Sources/Utilities.c:76`, hazard 1); seeded into `qd.randSeed` (`:61`) | rollback needs `Save()`/`Restore()` on the generator; the original literally cannot provide it |
| 3 | The house handle is mutable and large | hazards 23, 24; `Objects.c:366-698` | it must be in the snapshot, and it is the only variable-size component |
| 4 | `SetObjectState`'s one-winner rule is order-dependent | `GliderPRO/Sources/Objects.c:460-462`: `changed = (…data.c.state == true); newState = false; …data.c.state = newState;` | whichever glider is tested first wins the prize (§10.5.2); a mispredicted input flips *which player scored*, which is far more visible than a position correction |
| 5 | Sound is fired from inside the simulation with no de-duplication | `PlayPrioritySound` throughout; e.g. `Grease.c:263`, `Interactions.c:769` | re-simulating replays sounds; rollback needs a sound-suppression pass, which the original has no hook for |

Blocker 4 deserves emphasis. In a rollback netcode, mispredicting the remote player's input
by one frame can change *who collected the battery*. Both peers converge, but one of them
saw the wrong player's score go up and then saw it taken away. In a shared-fate game where
`theScore`, `batteryTotal`, `bandsTotal`, `foilTotal` and `mortals` are all shared globals
(§9.4, §9.8), that visible flicker buys nothing: **the pooled resources mean the players do
not care who picked it up.** Rollback's entire value proposition — hiding latency on your
*own* actions — is already mostly satisfied here, because your own glider's motion is
integer, low-inertia, and 30 Hz.

## 10.2.4 Decision

> **Use input-delay lock-step.** Both peers run the identical, complete, whole-world
> simulation. Neither peer simulates "its own player" — each simulates *both*. The only
> thing that crosses the wire during play is a per-frame input word per player (§10.4.2).
> The simulation for frame `N` does not begin until both peers' inputs for frame `N` are in
> hand. Local input is applied `kInputDelay` frames in the future.

Recommended parameters:

| Parameter | Value | Justification |
| --- | --- | --- |
| `kInputDelay` | **2 frames** | 2 frames × `kTicksPerFrame` (2 ticks = 2/60 s) ≈ **66 ms** of tolerated one-way jitter. The original already freezes the second arriving glider for 30 frames after every room change (`TagGliderIdle` → `hVel = 30`, `GliderPRO/Sources/Modes.c:631`, `:638`), so a 2-frame input delay is 1/15th of a grace period the game already imposes. |
| `kMaxInputDelay` | 6 frames (200 ms) | negotiated up at `MatchStart` (§10.4.7) if RTT demands it; must be agreed, never unilateral |
| Redundancy | resend the last **8** input words in every packet | 8 frames × 1 byte × 2 players = 16 bytes; makes single packet loss free |
| Send rate | 1 packet per simulated frame (30/s) | see §10.4.4 for the resulting 360–480 B/s |
| Checksum cadence | every **8** frames, on `gameFrame & 0x00000007 == 0` | reuses the phase-10 slicing the game already does (`whosTurn = gameFrame & 0x00000007` at `GliderPRO/Sources/Scoreboard.c:95`, dispatched at `:96`; hazard 31) |
| Desync policy | hard stop with a diagnostic dump | see §10.4.6 |

**Why 2 and not 0.** With zero delay, lock-step means "wait for the remote packet before
simulating", which converts every network hiccup into a visible stall. With 2 frames of
delay, a packet may arrive up to 66 ms late and the local player never notices, because the
local player's own input for frame `N` was decided at frame `N-2` and is already in the
buffer.

**Why not more than 2 by default.** The glider's response to input is a velocity change of
`kNormalThrust` = 5 or `kHyperThrust` = 8 applied on the *same* frame
(`GliderPRO/Sources/Input.c:14-15`, applied at `:313` (right) and `:323` (left), and
`DoBatteryEngaged` `:126`, `:128`, `:133`, `:135`). Input latency is directly felt. Two
frames is 66 ms, which is
already at the edge of noticeable for a game this responsive.

## 10.2.5 The restructured frame loop

`PlayGame` (`GliderPRO/Sources/Play.c:430-599`; the frame loop itself is `:432-554`) becomes:

```
NetPlayGame():
   1. while playing and !quitting:
   2.     // ---- collect and send local input for the FUTURE frame ----
   3.     localWord <- SampleLocalInput()                  // replaces GetKeys(theKeys)
   4.     inputBuffer[myIndex][gameFrame + kInputDelay] <- localWord
   5.     SendInputFrames(gameFrame + kInputDelay, last 8 words)      // §10.4.4
   6.
   7.     // ---- barrier: do not advance without both peers' inputs ----
   8.     PumpNetwork()                                     // non-blocking; fills inputBuffer
   9.     if inputBuffer[peerIndex][gameFrame + 1] is absent then
  10.         stallFrames++
  11.         Draw()                                        // keep the window alive
  12.         if stallFrames > kStallTimeout then EnterDisconnectFlow()
  13.         continue                                      // do NOT advance gameFrame
  14.     stallFrames <- 0
  15.
  16.     // ---- the simulation step: byte-identical on both peers ----
  17.     gameFrame++                                       // Play.c:434
  18.     evenFrame <- !evenFrame                            // Play.c:435
  19.     HandleTelephone()                                  // Play.c:445  (phase 2)
  20.     HandleDynamics()                                   // Play.c:449  (phase 3)
  21.     if !gameOver then
  22.         ApplyInput(&theGlider,  inputBuffer[p1][gameFrame])   // phase 4, replaces GetInput
  23.         ApplyInput(&theGlider2, inputBuffer[p2][gameFrame])
  24.         HandleInteraction()                            // Play.c:454  (phase 5) UNCHANGED
  25.     HandleTriggers()                                   // Play.c:456  (phase 6)
  26.     HandleBands()                                      // Play.c:457  (phase 7)
  27.     if !gameOver then
  28.         HandleGlider(&theGlider)                       // Play.c:460  (phase 8)
  29.         HandleGlider(&theGlider2)                      // Play.c:461
  30.     if playing then
  31.         SimStepPart2()                                 // phase 9a  (§10.1.5)
  32.         AdvanceScoreboardState()                       // phase 10, sim part only
  33.     if gameOver then
  34.         countDown--                                    // Play.c:501
  35.         if countDown <= 0 then EndOfGame()
  36.
  37.     // ---- checksum exchange ----
  38.     if (gameFrame & 0x7) == 0 then
  39.         SendChecksum(gameFrame, HashSimState())         // §10.4.6
  40.         VerifyPendingChecksums()
  41.
  42.     // ---- presentation, decoupled ----
  43.     Draw()                                             // phase 9b
  44.     PaceToNextFrame()                                  // replaces Render.c:662-665
```

Differences from the original that are *required*, each traceable to a hazard in §7.2:

| Change | Original | Hazard closed |
| --- | --- | --- |
| `gameFrame++` moved *after* the network barrier | `Play.c:434`, top of loop | new: makes the barrier the only place the clock can stall |
| `GetKeys(theKeys)` removed from the sim; `ApplyInput` reads a buffer | `Input.c:285`, one global `KeyMap` shared by both gliders (§9.3) | — (this is the seam §9.3 identifies) |
| no `HandlePlayEvent()` / `switchedOut` loop inside the step | `Play.c:437-443` | 11 |
| `DoPause()` cannot be called from `ApplyInput` | `Input.c:77-117`, `:373-377` | 12 |
| `SimStepPart2` separated from `Draw` | `Play.c:469`, `Render.c:639-671` | 13 |
| `TickCount` pacing moved out of the render function | `Render.c:662-665` | 10 (already benign) |
| `DoCommandKey()` cannot be called from `ApplyInput` | `Input.c:287`, `:53-73` | new (see §10.2.7) |

Everything else — phase order, `evenFrame`, the `gameOver` gating of phases 4/5/8, the
`countDown` tail — is **unchanged**, deliberately (§7.4).

## 10.2.6 Stall policy

Under lock-step the local frame cannot advance without the peer's word. That must be
handled explicitly, not by blocking on a socket read.

| State | Condition | Behaviour |
| --- | --- | --- |
| Running | peer word for `gameFrame + 1` present | advance |
| Stalling | absent, `stallFrames <= 15` (0.5 s) | do not advance; keep drawing the last frame; keep sending |
| Warning | `15 < stallFrames <= 150` (0.5–5 s) | show a "waiting for player 2" overlay; keep sending; keep buffering local input at `gameFrame + kInputDelay` (the buffer must hold at least `kStallTimeout + kInputDelay` words) |
| Disconnected | `stallFrames > 150` | `EnterDisconnectFlow()` |

`EnterDisconnectFlow()` has a natural implementation the original already provides:
**convert the match to a one-player game using the death cascade.** `OffAMortal` sets
`onePlayerLeft = true` and `playerDead = thisGlider->which`
(`GliderPRO/Sources/Player.c:1507-1508`), after which roughly forty guard clauses across
the codebase (the exhaustive list is in §9.5) suppress all interaction for the departed
glider. So a disconnect is representable *as an existing game state*. The port should:

1. synthesise `otherPlayerEscaped = kPlayerIsDeadForever` (-69)
   (`GliderPRO/Headers/GliderDefines.h:596`) — the value `OffAMortal` itself writes at
   `GliderPRO/Sources/Player.c:1602`,
2. set `onePlayerLeft = true`, `playerDead = <the departed player's which>`,
3. `FlagGliderInLimbo(departed, false)` and `departed->dontDraw = true`
   (`GliderPRO/Sources/Player.c:1505-1506`),
4. continue the simulation single-player, with `twoPlayerGame` still `true` so the doubled
   pickups (§10.5.3) correctly stop doubling — note `HandleRewards` gates the doubling on
   `(twoPlayerGame) && (!onePlayerLeft)`, so this happens automatically
   (`GliderPRO/Sources/Interactions.c:842`, `:864`, `:883`, `:911`, `:970`).

That is a graceful degradation with **zero new game logic**.

## 10.2.7 The complete blocking-call inventory a networked build must remove

§6.4 covers this generally. Concretely, every place the original blocks or spins inside the
game loop:

| Site | Code | Effect | Networked replacement |
| --- | --- | --- | --- |
| `GliderPRO/Sources/Render.c:662-664` | `while (TickCount() < nextFrame) {}` | busy-wait until the frame deadline | move to `PaceToNextFrame()`; sleep, and skip the wait entirely if the network barrier already consumed the time |
| `GliderPRO/Sources/Play.c:439-442` | `do { HandlePlayEvent(); } while (switchedOut);` | halts the sim while the app is backgrounded (hazard 11) | never gate the sim on focus; pump events outside the barrier |
| `GliderPRO/Sources/Input.c:88-93` | `do { GetKeys(theKeys); } while (pause key still down);` | spin until key release | forbidden |
| `GliderPRO/Sources/Input.c:96-104` | `paused = true; while (paused) { GetKeys(...) }` | the whole pause implementation is a spin loop inside phase 4 (hazard 12) | pause becomes a *networked agreed state* — see below |
| `GliderPRO/Sources/Input.c:112-118` | second key-release spin after unpausing | | forbidden |
| `GliderPRO/Sources/Input.c:56-73` | `DoCommandKey()` → `Alert`, `QuerySaveGame` (`:383-397`) | modal dialog inside phase 4 | route to a UI queue handled outside the barrier |
| `GliderPRO/Sources/Transit.c:300` | `WipeScreenOn(...)` | multi-frame screen wipe animation inside phase 5/8 | move into `Draw`; the *sim* transition must be instantaneous |

**Networked pause.** Pause must be a simulation input, not a control-flow hijack. The clean
design, and the one that costs nothing:

* Reserve bit 6 of the input word (§10.4.2) as `pauseToggle`.
* `ApplyInput` observing `pauseToggle` sets a global `netPaused = !netPaused`.
* When `netPaused`, the loop still runs the barrier and still exchanges input words (all
  zero except pause bits), but skips phases 2–10 entirely. `gameFrame` **still increments**,
  so both peers stay on the same clock and the checksum cadence keeps working.

This makes pause deterministic, symmetric, and free of a special protocol message.

## 10.2.8 Why not a client/server authoritative model

An authoritative-server design would sidestep desyncs entirely: one peer simulates, the
other renders received state. It is rejected for this port because:

1. **The state to ship is large and unstructured.** §10.2.2 gives ~10 KB of arrays plus a
   mutable house. There is no existing serialisation: `SaveGame` refuses to run in
   two-player mode (`GliderPRO/Sources/SavedGames.c:309-310`) and `gameType` is 40 bytes
   describing *one* glider (§4.3.4). Building a full state-sync format is strictly more work
   than building the 1-byte input word.
2. **The game is already structurally deterministic** (§7.3) — integer physics, fixed
   timestep, no floats, no hash iteration, no threads. Throwing that away to ship state
   would be perverse.
3. **The shipped `'demo' 128` resource is proof that input-replay works** and is a free
   1117-record test vector (§8.5, §10.4.9). There is no equivalent test asset for state
   replication.
4. **Latency would be worse for the non-authoritative player**, who would see their own
   glider respond at RTT rather than at `kInputDelay`.

Lock-step's one real weakness — a single divergence is fatal — is exactly what §10.3 and
§10.4.6 exist to eliminate.

---

# 10.3 RNG: own PRNG, named streams, and decoupling from rendering

This section closes hazards 1, 2, 3, 4, 5, 6, 8 and 25 from §7.2.

## 10.3.1 Replace `Random()`, keep `RandomInt`'s arithmetic

`RandomInt` (`GliderPRO/Sources/Utilities.c:72-82`):

```c
short RandomInt (short range)
{
	register long	rawResult;

	rawResult = Random();                              // Utilities.c:76  QuickDraw trap
	if (rawResult < 0L)
		rawResult *= -1L;                              // Utilities.c:78
	rawResult = (rawResult * (long)range) / 32768L;    // Utilities.c:79
	return ((short)rawResult);
}
```

`Random()` is a Toolbox trap; its algorithm is not in the tree (hazard 1). It returns a
signed 16-bit value in `[-32768, 32767]` seeded from `qd.randSeed`, which
`ToolBoxInit` fills once per launch with `GetDateTime((UInt32 *)&qd.randSeed)`
(`GliderPRO/Sources/Utilities.c:61`, hazard 2).

**What must be preserved is the mapping, not the generator.** Three properties, all
established empirically in §2.1.1 and all load-bearing:

| Property | Consequence | Hazard |
| --- | --- | --- |
| `abs()` then `* range / 32768` | the upper bound is **inclusive**: `RandomInt(6)` can return 6 (`AddStar`, `GliderPRO/Sources/DynamicMaps.c:685`, indexing a 6-entry sprite strip 0..5), and `RandomInt(kNumCandleFlames)` can return 5 where `kNumCandleFlames` = 5 (`GliderPRO/Headers/GliderDefines.h:447`) and valid indices are 0..4 | 29 |
| `abs(-32768)` on a 16-bit-then-widened value | the extreme negative maps to 32768, which after `* range / 32768` yields exactly `range` | 29 |
| the distribution is not uniform | `RandomInt(2)` yields 0 for 32767 inputs, 1 for 32768, and 2 for 1 | 30 |

So the Go replacement is:

```go
// RandomInt reproduces GliderPRO's Utilities.c:72-82 mapping exactly, including the
// inclusive upper bound and the non-uniformity. Do NOT use rand.Intn here.
func (s *Stream) RandomInt(rng int16) int16 {
    raw := int32(s.next16())        // signed 16-bit, [-32768, 32767]
    if raw < 0 {
        raw = -raw                  // note: -(-32768) == 32768 in int32, as on 68k
    }
    return int16((raw * int32(rng)) / 32768)
}
```

`next16()` must be a *specified*, seedable, deterministic 16-bit generator. Any of these is
fine; what matters is that it is written down in the protocol version:

| Option | Notes |
| --- | --- |
| xorshift32, take the high 16 bits | 4-byte state, trivially serialisable, good enough for candle flames |
| PCG32 truncated to 16 bits | better statistically; 8-byte state |
| the classic Mac `Random()` LCG, if a peer ever wants to look like 1994 | irreproducible without the ROM; **do not** attempt |

**Do not** use `math/rand` or `math/rand/v2` global functions: they are not
version-stable across Go releases and `rand.Intn` has a different (correct, exclusive)
bound. The generator is part of the wire protocol; pin it in your own code.

Also relevant: `RandomLong` exists (`GliderPRO/Sources/Utilities.c:88`) but §2.4
classifies it as dead. Do not port it unless a call site is found.

## 10.3.2 Named streams

Hazard 8: one global stream mixes gameplay timers, cosmetics, the splash screen and the
game-over animation, and §2.5's Group C observation is decisive —

> "If peer 1 has watched an attract demo and peer 2 has not, or one peer saw a game-over
> animation, their streams are at different positions."

The fix is to give every consumer its own independently-seeded stream, so that drawing a
flower on a splash screen cannot move a telephone timer. Derive each stream from a single
`matchSeed` (§10.3.6) with a fixed, documented label:

| Stream | Label | Consumers (from §2.5) | Sync class |
| --- | --- | --- | --- |
| `StreamPhone` | `"phone"` | A1 `Play.c:735`, A2 `:736`, A4 `:759`, A5 `:760` | **synchronised** — gameplay |
| `StreamChimes` | `"chimes"` | A3 `Play.c:739`, A6 `:775`, A7 `:784` | **synchronised** — audio only, but drawn per frame from a shared clock; keep it synchronised so a desync check on it catches drift early |
| `StreamDynamics` | `"dyna"` | A8 `Dynamics.c:304` (sparkle object), A9 `:492`, A10 `:506` (coffee) | **synchronised** — gameplay; `HandleCoffee`'s timer changes when a cup of coffee launches, which moves the glider |
| `StreamRoomLoad` | `"roomload"` | B1 `Dynamics3.c:208`, B2 `DynamicMaps.c:338`, B3 `:422`, B4 `:508`, B5 `:594`, B6 `:685` | **synchronised** — see §10.3.5; only B5 (`toOrFro`) and B1 (`kSparkle` timer) are gameplay-visible, but all six must be consumed on a fixed schedule so the stream position stays aligned |
| `StreamCosmetic` | `"cosmetic"` | C1 `InterfaceInit.c:160` (splash flower), C2–C10 `GameOver.c:123`–`:385` (page flutter) | **local, never synchronised** — reseed freely from wall clock |
| — | — | C11–C15 `ObjectAdd.c:621`–`:742` | **editor only**; never runs during `PlayGame` |
| — | — | C16 `Transitions.c:44` | **dead code**; `PourScreenOn` is never called (§2.4) |

Stream derivation, so that the set of streams is extensible without renumbering:

```
streamSeed(label) = SHA256(matchSeed_le64 || 0x00 || label)[0:8] interpreted little-endian
```

Each stream also carries a **draw counter** (`uint64`, incremented on every `next16()`).
The counters are part of the checksum (§10.4.6). This is the single most valuable
diagnostic in the whole design: if two peers' `StreamRoomLoad` counters differ after a room
transition, you know instantly that the *room-load draw schedule* diverged, which localises
the bug to §10.3.5 rather than to physics.

## 10.3.3 Decoupling RNG from `GetNumberOfLights` — and a correction to hazard 25

§5.6 and hazard 25 assert:

> "…in an unlit room some object branches skip their registration calls, so **turning a
> light off can change how much RNG a room load consumes.**"

**The stated mechanism is wrong.** Verified by reading `DrawARoomsObjects`
(`GliderPRO/Sources/ObjectDrawAll.c:22`+) in full:

```c
	isLit = (numLights > 0);                           // ObjectDrawAll.c:38
	...
		case kTaper:
		GetObjectRect(&thisObject, &itsRect);
		OffsetRectRoomRelative(&itsRect, neighbor);
		if (SectRect(&itsRect, &testRect, &whoCares))
		{
			if (isLit)
				DrawSimpleBlowers(thisObject.what, &itsRect);   // <-- ONLY the draw is gated
			if (neighbor == kCentralRoom)
			{
				if (redraw)
					ReBackUpFlames(localNumbers[neighbor], i);
				else
					AddCandleFlame(localNumbers[neighbor], i,   // <-- NOT gated on isLit
							itsRect.left + 10, itsRect.top + 7);
			}
			...
```

`isLit` gates **only** `Draw*` calls. Every RNG-consuming registration —
`AddCandleFlame` (`:83`, `:98`, `:117`, `:132`, `:151`, `:166`), `AddTikiFlame` (`:181`),
`AddBBQCoals` (`:195`), `AddPendulum` (`:343`), `AddGrease` (`:376`, `:397`),
`AddStar` (`:440`), `AddDynamicObject(kSparkle, …)` (`:456`) — is outside the `isLit`
test. Likewise all seven `BackUpToSavedMap` calls in the file (`:294`, `:308`, `:322`,
`:336`, `:360`, `:416`, `:434`) are ungated. The enemy `AddDynamicObject` calls (`:802`,
`:813`, `:824`, `:835`, `:846`, `:857`, `:872`, `:888`) are gated on
`(neighbor == kCentralRoom) && (!redraw)` — a positional test, not a lighting one.

`numLights == 0` *does* short-circuit `DrawRoomBackground`
(`GliderPRO/Sources/RoomGraphics.c:179-191`) — it paints the room's rect black and returns
early. That changes the *pixels* that `BackUpToSavedMap` subsequently snapshots (a dark
snapshot instead of a background one), which is why turning the lights on triggers the whole
`ReBackUp*` family (`RedrawRoomLighting` `GliderPRO/Sources/RoomGraphics.c:434-461`). It
does not change RNG consumption.

**The real mechanism by which gameplay history changes RNG consumption is `IsThisValid`.**
`GliderPRO/Sources/Objects.c:89-122` (`HGetState`/`HLock`/`HSetState` elided):

```c
Boolean IsThisValid (short where, short who)
{
	itsGood = true;                                                      // :94
	switch ((*thisHouse)->rooms[where].objects[who].what)                // :98
	{
		case kObjectIsEmpty:                                             // :100
		itsGood = false;
		break;

		case kRedClock:  case kBlueClock: case kYellowClock: case kCuckoo:
		case kPaper:     case kBattery:   case kBands:       case kFoil:
		case kInvisBonus:case kStar:      case kSparkle:     case kHelium:
		itsGood = (*thisHouse)->rooms[where].objects[who].data.c.state;   // :116
		break;
	}
	return (itsGood);                                                    // :121
}
```

(The twelve prize `case` labels are `Objects.c:104-115`, in that order.)
`DrawARoomsObjects` skips the whole object when it returns false
(`GliderPRO/Sources/ObjectDrawAll.c:48`). `SetObjectState` clears `data.c.state` on pickup
(`GliderPRO/Sources/Objects.c:460-462`). So:

1. A player collects a **star**. `data.c.state ← false`.
2. On every later load of that room, `IsThisValid` returns false, the `kStar` case is
   skipped, and **`AddStar`'s `RandomInt(6)` (B6, `DynamicMaps.c:685`) is never drawn.**
3. A player collects a **sparkle** prize (`kSparkle`, one of the twelve `IsThisValid`
   types). On every later load, `AddDynamicObject(kSparkle, …)` is skipped and
   **B1's `RandomInt(60) + 15` (`Dynamics3.c:208`) is never drawn.**
4. Two saved-map slots per collected star, and one per collected prize, are freed from the
   24-slot budget — which can let a *previously-suppressed* `AddCandleFlame` succeed and
   **add** a B2 draw (§10.3.4).

That is the coupling: **prize-collection history changes the room-load draw schedule.** It
is exactly as serious as §7.3 says, but it lives in `IsThisValid` and `numSavedMaps`, not in
`numLights`.

**The fix.** Make the room-load draw schedule a pure function of the room's *logical*
contents, independent of both resource limits and collection state:

```
LoadRoom(roomNumber):
   1. objects <- rooms[roomNumber].objects[0..23]        // ALL 24 slots, in index order
   2. for i = 0 .. 23:
   3.     if objects[i].what == kObjectIsEmpty then continue
   4.     // draw the RNG this object type would draw, ALWAYS, before any guard:
   5.     switch objects[i].what:
   6.         kTaper, kCandle, kStubby:  mode <- StreamRoomLoad.RandomInt(kNumCandleFlames)
   7.         kTiki:                     mode <- StreamRoomLoad.RandomInt(kNumTikiFlames)
   8.         kBBQ:                      mode <- StreamRoomLoad.RandomInt(kNumBBQCoals)
   9.         kCuckoo:                   toOrFro <- StreamRoomLoad.RandomInt(2) == 0
  10.         kStar:                     mode <- StreamRoomLoad.RandomInt(6)
  11.         kSparkle:                  timer <- StreamRoomLoad.RandomInt(60) + 15
  12.     // THEN decide whether the entity actually exists:
  13.     if !IsThisValid(roomNumber, i) then continue     // collected prize -> no entity
  14.     if <per-class cap reached> then continue         // kMaxCandles etc.
  15.     if <geometry guard fails> then continue          // h < 16, v < 15, ...
  16.     register the entity with the mode/timer drawn in step 5
```

Steps 5–11 run for **every non-empty object of a drawing type, in room-object index
order 0..23**, regardless of collection state, caps, geometry guards, or free memory. The
schedule is then a function of `(house file bytes, roomNumber, neighbour set)` only.

The cost is that a room whose star has been collected still burns one `RandomInt(6)`. That
is intentional: it is what makes the stream position independent of gameplay history.

**Note on `kNumCandleFlames` and the three candle types.** `AddCandleFlame` is called from
six `case` labels — `kTaper` (`:83`, `:98`), `kCandle` (`:117`, `:132`), `kStubby` (`:151`,
`:166`) — twice each, because the central-room path and the neighbour-room path are separate
branches and the neighbour path has an extra `SectRect` overlap test against
`localRoomsDest[kCentralRoom]` expanded by `kFloorSupportTall`
(`GliderPRO/Sources/ObjectDrawAll.c:88-93` for `kTaper`, repeated verbatim at `:122-127`
for `kCandle` and `:156-161` for `kStubby`; only the sprite hot-spot offset differs — +10,
+14, +9 respectively). At most one of the two fires per object.
A port must draw **one** flame RNG per candle object, and must reproduce the overlap test if
it wants the *entity* (not the draw) to appear in the same cases.

## 10.3.4 Decoupling from the saved-map budget — and a correction to hazard 6

Hazard 6 says `BackUpToSavedMap` can return -1 for "24-slot cap **or GWorld allocation
failure**", and concludes "**free memory affects the RNG stream**". The first half is right;
the parenthetical is wrong. `GliderPRO/Sources/DynamicMaps.c:70-93`:

```c
short BackUpToSavedMap (Rect *theRect, short where, short who)
{
	Rect		mapRect;
	OSErr		theErr;

	if (numSavedMaps >= kMaxSavedMaps)                                       // :75
		return(-1);                                                          // :76

	mapRect = *theRect;
	ZeroRectCorner(&mapRect);
	savedMaps[numSavedMaps].dest = *theRect;
	theErr = CreateOffScreenGWorld(&savedMaps[numSavedMaps].map,
			&mapRect, kPreferredDepth);                                      // :82

	CopyBits(... backSrcMap ..., savedMaps[numSavedMaps].map, ...);           // :84-86

	savedMaps[numSavedMaps].where = where;
	savedMaps[numSavedMaps].who = who;
	numSavedMaps++;
	return (numSavedMaps - 1);                                               // :92
}
```

**`theErr` is assigned and never tested.** The function returns -1 on exactly one
condition: `numSavedMaps >= kMaxSavedMaps` (`kMaxSavedMaps` = 24,
`GliderPRO/Headers/GliderDefines.h:260`). An allocation failure produces a nil `GWorldPtr`
that is then passed straight to `CopyBits`, and the slot is still consumed. So:

> **Corrected hazard 6: free memory does *not* affect the RNG stream. The 24-slot budget
> does, and that budget is affected by gameplay history.** Severity stays FIXABLE; the
> mechanism is the cap alone.

That is *better* news than the original hazard, because a cap is deterministic. But the cap
is still coupled to gameplay through `IsThisValid` (§10.3.3), and there is a second, sharper
finding:

**A `kStar` object consumes two saved-map slots.** `DrawARoomsObjects` calls
`BackUpToSavedMap(&itsRect, localNumbers[neighbor], i)` directly at
`GliderPRO/Sources/ObjectDrawAll.c:434`, and then calls `AddStar(...)`
(`:440`, under the `legit != -1` guard at `:435` and the `else` of the `redraw` test at
`:437-439`), whose body calls `BackUpToSavedMap(&bounds, where, who)` again at
`GliderPRO/Sources/DynamicMaps.c:680` — **with the same `(where, who)` key**. Both
succeed (nothing dedupes), so one star burns 2 of the 24 slots. With `kMaxStars` = 4
(`GliderPRO/Headers/GliderDefines.h:263`), four stars in view consume **8 slots**, a third of
the budget.

Consequences a port must decide about explicitly:

1. `RestoreFromSavedMap` has a `break` after the first match
   (`GliderPRO/Sources/DynamicMaps.c:160`), so only the *first* of the two duplicate slots
   is ever restored. The second is dead from creation.
2. `ReBackUpSavedMap` likewise returns on the first match
   (`GliderPRO/Sources/DynamicMaps.c:119`), so a lighting change re-snapshots only the first.
3. The doubled consumption tightens the budget, which — per §10.3.3 — is the very thing that
   makes `AddCandleFlame` fail and drop a B2 RNG draw in cluttered rooms.

The `kFoil` case does the same *shape* of thing (`ObjectDrawAll.c:416`) but with only one
slot, because `DrawFoil` does not call `AddStar`-style registration.

The complete saved-map consumer list, with slot counts per object, verified by grep:

| Consumer | Site | Slots | Notes |
| --- | --- | --- | --- |
| clocks / prizes (5 branches) | `ObjectDrawAll.c:294`, `:308`, `:322`, `:336`, `:360` | 1 each | `legit != -1` gates only the `Draw*` call |
| `kFoil` | `ObjectDrawAll.c:416` | 1 | |
| `kStar` (outer) | `ObjectDrawAll.c:434` | 1 | **plus** the inner one below |
| `AddStar` (inner) | `DynamicMaps.c:680` | 1 | same `(where, who)`; **duplicate** |
| `AddCandleFlame` | `DynamicMaps.c:333` | 1 | gated on `numFlames < 20`, `h >= 16`, `v >= 15` |
| `AddTikiFlame` | `DynamicMaps.c:417` | 1 | `numTikiFlames < 8`, `h >= 8`, `v >= 10` |
| `AddBBQCoals` | `DynamicMaps.c:503` | 1 | `numCoals < 8`, `h >= 32`, `v >= 9` |
| `AddPendulum` | `DynamicMaps.c:580` | 1 | cap/geometry guard `numPendulums >= kMaxPendulums (8) \|\| h < 32 \|\| v < 28` returns early at `:575`; **`clockFrame = 10`** at `:578` sits *after* that guard but *before* the slot check at `:580`, so the global is perturbed when the pendulum is dropped for lack of a **slot**, but not when it is dropped by the cap or the geometry test |
| `AddGrease` | `Grease.c:219` | 1 | `numGrease < 16` |

`AddPendulum`'s ordering is worth a second look: the cap-and-geometry guard returns at
`GliderPRO/Sources/DynamicMaps.c:575`, then `clockFrame = 10` at `:578` executes *before*
`savedNum = BackUpToSavedMap(...)` at `:580`. So a pendulum that is dropped for lack of a
slot still resets the global clock phase for every other pendulum in the room, while a
pendulum dropped by the cap or the `h < 32 / v < 28` test does not. Hazard 28 notes the
reset; the ordering relative to the two *different* rejection points is the part a port will
get wrong.

**The fix**, consistent with §10.3.3: the RNG draw happens before any guard, and the
saved-map budget becomes a pure rendering concern.

```
AddCandleFlame(where, who, h, v):
   1. mode <- StreamRoomLoad.RandomInt(kNumCandleFlames)     // ALWAYS, first
   2. if numFlames >= kMaxCandles then return                 // then the caps
   3. if h < 16 or v < 15 then return
   4. slot <- renderer.SnapshotBackground(where, who)         // may fail; irrelevant to sim
   5. flames[numFlames] = {dest, mode, src: rect(0, mode*15, 16, mode*15+15), who: slot}
   6. numFlames++
```

Under §10.1's split, `SnapshotBackground` lives entirely on the render side, so a headless
peer never calls it, never needs a 24-slot cap, and never diverges. **The `savedMaps`
array leaves the simulation state entirely.** (Consequence: drop it from the checksum in
§10.4.6 — and note that this is a deliberate divergence from the original, which will
produce different candle-flame *sprite phases* than 1994 in cluttered rooms. Cosmetic.)

## 10.3.5 The room-load draw contract

Combining §10.3.3 and §10.3.4, the port owes a written contract for room loading, because
this is where lock-step will break if anywhere does.

**Contract.** For a given `(house bytes, roomNumber, protocolVersion)`, the sequence of
`StreamRoomLoad` draws produced by a room load is identical on every peer, for all time,
regardless of: which prizes have been collected, which lights are on, how much memory is
free, the screen size, the colour depth, the `numNeighbors` preference, or whether the peer
is rendering at all.

Enforcement rules:

| Rule | Rationale | Hazard closed |
| --- | --- | --- |
| **R1.** Iterate room objects by index `0..kMaxRoomObs-1`, never by a filtered or sorted list | `DrawARoomsObjects` already does (`ObjectDrawAll.c:43`); makes order independent of contents | — |
| **R2.** Draw RNG before every guard (caps, geometry, `IsThisValid`, snapshot) | §10.3.3 step 5, §10.3.4 | 6, 25 |
| **R3.** The neighbour-room set used for RNG purposes is **fixed at 9**, never the `numNeighbors` preference | `DrawLocale` gates neighbour passes on `numNeighbors > 3` / `> 1` (`RoomGraphics.c:78`, `:105`) and `numNeighbors` is a user preference forced to 1 when `thisMac.screen.right <= 512` (`Main.c:191-192`) | **3, 4** |
| **R4.** Never let colour depth shift a geometry guard | `thisMac.isDepth == 4` decrements `h` before the `h % 2` alignment fix in `AddTikiFlame` (`DynamicMaps.c:409-414`), `AddBBQCoals` (`:495-500`), `AddPendulum` (`:584-589`), `AddStar` (`:671-676`), and offsets `src` in `AddCandleFlame` (`:326-331`) — any of which can cross the `h < 8` / `h < 32` thresholds | **5** |
| **R5.** Room-load RNG draws happen only inside `LoadRoom`, never from a draw call | `RedrawRoomLighting` (`RoomGraphics.c:434-461`) calls `DrawARoomsObjects(kCentralRoom, true)` with `redraw = true`, which routes every `Add*` to its `ReBackUp*` twin and therefore consumes **no** RNG — verified; a port must preserve that property | 3 |
| **R6.** `clockFrame = 10` happens if and only if the room contains at least one `kCuckoo`, and happens before any per-pendulum guard | `DynamicMaps.c:578` vs `:580` | 28 |
| **R7.** `evenFrame` writes from `AddDynamicObject` must be replicated | `kBall` and `kFish` set the global `evenFrame = true` at `Dynamics3.c:474`, `:524` | **16** |

R3 is the most important and the least obvious. §7.3 calls hazard 3 "the worst one". The
fix is not to make `numNeighbors` a synchronised setting — it is to stop the *simulation*
from caring about it at all. Under R3, `numNeighbors` becomes a pure view setting: a peer
running a 512-pixel-wide window sees one room, its opponent sees nine, and both simulate
nine rooms' worth of entities. That is the correct semantics anyway, because entities in
neighbour rooms are registered into the *same* `flames`/`dinahs`/`hotSpots` arrays
(`DrawARoomsObjects(kNorthWestRoom, false)` etc., `RoomGraphics.c:82`–`:120`) and therefore
already participate in the central room's simulation.

Note the consequence for `numLights`: under R3 the *nine* `GetNumberOfLights` calls
(`RoomGraphics.c:80`, `:84`, `:88`, `:92`, `:96`, `:100`, `:107`, `:112`, `:118`) all still
happen, in that exact order (NW, NE, N, SW, SE, S, W, E, central). `GetNumberOfLights`
consumes no RNG itself (`GliderPRO/Sources/Room.c:970-1099`), so its only remaining role is
to decide sprite variants — i.e. it becomes render-only, which is what hazard 25 wanted.

## 10.3.6 Seeding

Hazard 2: the original seeds once per launch from `GetDateTime`
(`GliderPRO/Sources/Utilities.c:61`). For a match:

```
MatchStart negotiation (§10.4.7):
   1. each peer generates a 64-bit nonce
   2. matchSeed = SHA256(min(nonceA,nonceB) || max(nonceA,nonceB))[0:8]  (little-endian)
   3. each synchronised stream is seeded from streamSeed(label) (§10.3.2)
   4. StreamCosmetic is seeded locally, from anything
```

Using `min`/`max` rather than "host's nonce" makes the seed symmetric and removes any
question of which side is authoritative. Both peers can compute it without an extra
round-trip once they have exchanged nonces.

Reseeding rules:

| Event | Reseed? | Evidence / rationale |
| --- | --- | --- |
| `NewGame` | no — the streams are created at `MatchStart` and never reseeded | `InitTelephone` (`Play.c:733-740`) *draws* from `StreamPhone` at game start; that draw must be part of the synchronised sequence |
| room transition | no | draws continue from wherever `StreamRoomLoad` is |
| player death / respawn | no | |
| game over → new game in the same match | yes, derive `matchSeed' = SHA256(matchSeed || gameIndex_le32)` | keeps successive games in a session independent but reproducible |
| attract demo, splash screen, game-over flutter | `StreamCosmetic` only | this is precisely what §2.5's Group C warning is about |

## 10.3.7 Hazard closure summary

| Hazard | §7.2 verdict | Closed by | Residual |
| --- | --- | --- | --- |
| 1 `Random()` is an OS trap | BLOCKER for byte-exact emulation | §10.3.1 — own PRNG | sequences differ from 1994; accepted |
| 2 wall-clock seed | FIXABLE | §10.3.6 — `matchSeed` | none |
| 3 `numNeighbors` changes RNG consumption | FIXABLE, worst | §10.3.5 **R3** — fixed 9-neighbour logical set | `numNeighbors` becomes view-only |
| 4 `numNeighbors` forced by screen width | FIXABLE | §10.3.5 R3 | none |
| 5 `isDepth == 4` shifts geometry guards | FIXABLE | §10.3.5 R4 + R2 | drop 4-bit paths; document |
| 6 `BackUpToSavedMap` -1 suppresses a draw | FIXABLE | §10.3.4 — draw first; `savedMaps` leaves the sim. **Hazard text corrected** | candle sprite phases differ in cluttered rooms |
| 7 exact caps | BENIGN, replicate | keep every cap value from `GliderDefines.h:249-266` | none |
| 8 one shared stream | FIXABLE | §10.3.2 — named streams | none |
| 25 object state affects `GetNumberOfLights` → RNG | FIXABLE | §10.3.3 — **mechanism corrected**: it is `IsThisValid` + the slot budget, not `numLights` | none |
| 29 inclusive upper bound | BENIGN, replicate | §10.3.1 | none |
| 30 non-uniform | BENIGN, replicate | §10.3.1 | none |

---

# 10.4 The wire format

## 10.4.1 What `demoType` teaches, positively and negatively

`demoType` (`GliderPRO/Headers/GliderStructs.h:334-339`) is the only input-serialisation
format the original has, and §8 establishes that it *works*: 1117 records replay 3369 frames
of physics correctly. Its design choices, and what to keep:

| `demoType` property | Evidence | Keep? |
| --- | --- | --- |
| `gameFrame` is the sequence number | `if (gameFrame == (long)demoData[demoIndex].frame)` (`GliderPRO/Sources/Input.c:224`); §8.7 "a reliable, monotone logical clock" | **Keep.** This is the single most valuable idea. |
| big-endian, 2-byte-packed, `long` = 32 bits | verified `sizeof(demoType)` = 6 with `#pragma pack(2)` and 32-bit `long`; §4.3.3 note on the 64-bit trap | **Keep big-endian** (it is also network byte order) and **keep explicit widths**. |
| one *action* per record, not one *state* per frame | `switch (demoData[demoIndex].key)` has exactly one arm per record (`Input.c:226-265`) | **Reject.** §8.4.1 asymmetry #1: two records on one frame permanently strand the cursor. |
| sparse: only frames with input get a record | observed 1117 records over 3369 frames | **Reject** for a live protocol — sparseness needs a per-record frame field (4 bytes for 1 bit of information) and makes loss detection ambiguous. Keep it as an *archival* format. |
| a `padding` byte that is never written | `LogDemoKey` writes only `.frame` and `.key` (`Input.c:46-47`); observed pad values are ASCII heap garbage (§8.5 finding 6) | **Reject.** No uninitialised bytes on the wire, ever. |
| no length, no version, no checksum | `BlockMove(*tempHandle, demoData, kDemoLength)` trusts a compile-time constant (`StructuresInit2.c:295`) | **Reject.** Version every message. |
| no bounds check when recording | `LogDemoKey` (`Input.c:44-49`) increments `demoIndex` with no cap; the buffer is `sizeof(demoType) * 2000` = 12 000 bytes under `CREATEDEMODATA` (`StructuresInit2.c:282`) and `NewPtr(kDemoLength)` otherwise (`:287`), where `kDemoLength` is a **byte** count of 6702 (`GliderPRO/Headers/GliderDefines.h:625`) = 1117 × 6-byte records | **Reject.** A recording longer than 2000 actions overruns the heap. |

Re-verified independently for Part 10 by parsing the `data 'demo' (128)` block starting at
`GliderPRO/Glider PRO.r:199389` with python3:

```
resource bytes: 6702   (kDemoLength = 6702)  match: True
records of 6 bytes: 1117   remainder: 0
min frame = 46, max frame = 3414
strictly increasing: True    duplicate frames: 0
key counts: {0: 910, 1: 198, 3: 9}      (key 2 never appears)
distinct padding values: 109
first 6 records: (46,0,114) (47,0,114) (48,0,114) (49,0,69) (56,0,6) (57,0,114)
last 5 records:  (3410,0,-1) (3411,0,-2) (3412,0,-8) (3413,0,7) (3414,0,1)
frame-delta histogram (top 8): 1:1009  7:8  4:6  8:6  3:6  10:6  14:4  13:4
```

This reproduces §8.5 exactly, including the 109 distinct garbage `padding` values.

## 10.4.2 `InputWord` — one byte per player per frame

The replacement for `demoType.key` is a **held-state bitmask**, not an action code.
Justification from the source: `GetInput` reads four *held* keys with `BitTst`
(`GliderPRO/Sources/Input.c:301`, `:318`, `:330`, `:344`) and writes `heldLeft`,
`heldRight`, `tipped`, `fireHeld` — i.e. the simulation's real input is four booleans, not
one enum.

| Bit | Mask | Name | Original key | Set when | Consumed at |
| --- | --- | --- | --- | --- | --- |
| 0 | `0x01` | `kInRight` | `thisGlider->rightKey` | right key held | `GliderPRO/Sources/Input.c:301` |
| 1 | `0x02` | `kInLeft` | `thisGlider->leftKey` | left key held | `GliderPRO/Sources/Input.c:318` |
| 2 | `0x04` | `kInBattery` | `thisGlider->battKey` | battery/helium key held | `GliderPRO/Sources/Input.c:330` |
| 3 | `0x08` | `kInBand` | `thisGlider->bandKey` | rubber-band key held | `GliderPRO/Sources/Input.c:344` |
| 4 | `0x10` | `kInSuicide` | `kDeleteKeyMap` | delete held, for `ForceKillGlider` | `GliderPRO/Sources/Input.c:368-371` |
| 5 | `0x20` | *reserved, must be 0* | — | — | — |
| 6 | `0x40` | `kInPauseToggle` | Esc or Tab, **edge-triggered** | pause key pressed this frame | `GliderPRO/Sources/Input.c:373-377`; see §10.2.7 |
| 7 | `0x80` | *reserved, must be 0* | — | — | — |

Bits 0–4 are **level-triggered** (held state, resent every frame). Bit 6 is
**edge-triggered** (set on the frame the key transitions down, cleared otherwise) — because
the original's pause is a key-press with two spin-wait release loops around it
(`GliderPRO/Sources/Input.c:89-94` before, `:111-116` after, with the `while (paused)`
poll loop at `:97-105` in between) and level-triggering it would toggle pause
30 times a second.

Bit ordering note: the *right* key is bit 0 because `GetInput` tests `rightKey` first and
`LogDemoKey(0)` is emitted from the right-key branch (§8.3). This keeps the numbering
consistent with the shipped demo's key codes: **0 = right, 1 = left, 2 = battery,
3 = band**, exactly as §8.3's note establishes (and note that the *comments* in
`GetDemoInput` at `GliderPRO/Sources/Input.c:228` and `:235` have them backwards).

**The input word is read by phase 5 as well as phase 4**, which a port must not overlook.
`heldRight` and `heldLeft` are written by `GetInput`
(`GliderPRO/Sources/Input.c:299-300`, `:309`, `:315`, `:325` — note `:324` writes `tipped`,
not `heldLeft`) and then *read* by `HandleHotSpotCollision`:

| Hotspot action | Guard | Line |
| --- | --- | --- |
| `kMoveItUp` (up stairs) | `if (!thisGlider->heldRight && GliderInRect(thisGlider, &who->bounds))` | `GliderPRO/Sources/Interactions.c:1251` |
| `kMoveItDown` (down stairs) | `if (!thisGlider->heldLeft && GliderInRect(thisGlider, &who->bounds))` | `GliderPRO/Sources/Interactions.c:1286` |

Those two lines are the **only** reads of `heldLeft`/`heldRight` in the entire tree; the
other nine occurrences are the writes in `Input.c` (`:220`, `:221`, `:231`, `:238` in
`GetDemoInput`; `:299`, `:300`, `:309`, `:315`, `:325` in `GetInput`). So **holding the
right-arrow key suppresses the up-staircase and holding left suppresses the down-staircase**:
a player standing on the stairs with bit 0 set never triggers the rendezvous. This is a deliberate
"walk past the stairs" affordance and it means the input word participates in the escape
protocol, not just in physics. It is another reason the input word must be exchanged before
phase 5 rather than merged into phase 4's output (§10.2.5 lines 22-24).

**`ApplyInput` — the replacement for `GetInput`.** Numbered to mirror
`GliderPRO/Sources/Input.c:281-379` so a porter can diff:

```
ApplyInput(thisGlider, word):                                 // replaces GetInput
   1. // NO GetKeys, NO DoCommandKey, NO DoPause here.        // cf. Input.c:283-287
   2. if thisGlider->mode == kGliderBurning (9) then          //     Input.c:290
   3.     if thisGlider->facing == kFaceLeft then
   4.         thisGlider->hDesiredVel -= kNormalThrust (5)     //     Input.c:293
   5.     else
   6.         thisGlider->hDesiredVel += kNormalThrust         //     Input.c:295
   7.     return                                              // burning ignores input entirely
   8. thisGlider->heldLeft  <- false                          //     Input.c:299
   9. thisGlider->heldRight <- false                          //     Input.c:300
  10. if word & kInRight then                                 //     Input.c:301
  11.     if word & kInLeft then                              //     Input.c:306
  12.         ToggleGliderFacing(thisGlider)                   //     Input.c:308
  13.         thisGlider->heldLeft <- true                     //     Input.c:309
  14.     else
  15.         thisGlider->hDesiredVel += kNormalThrust         //     Input.c:313
  16.         thisGlider->tipped   <- (facing == kFaceLeft)     //     Input.c:314
  17.         thisGlider->heldRight<- true                      //     Input.c:315
  18. else if word & kInLeft then                              //     Input.c:318
  19.     thisGlider->hDesiredVel -= kNormalThrust             //     Input.c:323
  20.     thisGlider->tipped   <- (facing == kFaceRight)        //     Input.c:324
  21.     thisGlider->heldLeft <- true                          //     Input.c:325
  22. else
  23.     thisGlider->tipped <- false                          //     Input.c:328
  24. if (word & kInBattery) and (batteryTotal != 0) and
  25.        (thisGlider->mode == kGliderNormal) then           //     Input.c:330-331
  26.     if batteryTotal > 0 then DoBatteryEngaged(thisGlider) //     Input.c:336-337
  27.     else                    DoHeliumEngaged(thisGlider)   //     Input.c:339
  28. else
  29.     batteryWasEngaged <- false                            //     Input.c:342
  30. if (word & kInBand) and (bandsTotal > 0) and
  31.        (thisGlider->mode == kGliderNormal) then           //     Input.c:344-345
  32.     if !thisGlider->fireHeld then                         //     Input.c:350
  33.         if AddBand(thisGlider, dest.left+24, dest.top+10, facing) then   // Input.c:352
  34.             bandsTotal--                                  //     Input.c:355
  35.             if bandsTotal <= 0 then QuickBandsRefresh(false)  // Input.c:356-357
  36.             thisGlider->fireHeld <- true                  //     Input.c:359
  37. else
  38.     thisGlider->fireHeld <- false                         //     Input.c:364
  39. if (otherPlayerEscaped != kNoOneEscaped) and (word & kInSuicide) and
  40.        (thisGlider->which) and (!onePlayerLeft) then       //     Input.c:366-368
  41.     ForceKillGlider()                                     //     Input.c:370
  42. if word & kInPauseToggle then netPaused <- !netPaused      // replaces Input.c:373-376
```

Two subtleties a port must preserve verbatim:

* **Line 39–41.** `ForceKillGlider` is guarded on `(thisGlider->which)`, and `which` is a
  `Boolean` with `kPlayer1 == TRUE` (`GliderPRO/Headers/GliderDefines.h:556`). So **only
  player 1 can trigger the suicide**, even though `ForceKillGlider`
  (`GliderPRO/Sources/Transit.c:449-469`) kills whichever glider is *not* in limbo. Bit 4
  of player 2's word is therefore inert. Do not "fix" this without deciding it is a rules
  change.
* **Lines 24–29 and 30–38.** The `else` arms fire whenever the condition is false *for any
  reason*, including `batteryTotal == 0` or a non-normal mode. So holding the battery key
  with an empty battery clears `batteryWasEngaged` every frame — which resets
  `batteryFrame` to 0 on the next successful engage
  (`GliderPRO/Sources/Input.c:147-148` in `DoBatteryEngaged`, `:173-174` in
  `DoHeliumEngaged`) and hence retriggers the thrust sound.
  This is exactly the guard-asymmetry §8.4.1 flags in `GetDemoInput`, and the bitmask format
  fixes it for free by re-deriving the guards from the same code path both players use.

## 10.4.3 Empirical validation of the format against the shipped demo

The shipped demo was transcoded into the `InputWord` format with python3 (bit 0 = key 0,
bit 1 = key 1, bit 2 = key 2, bit 3 = key 3) and measured:

```
span: frames 46..3414 = 3369 frames
raw 1 byte/frame:                    3369 bytes   (vs demoType stream 6702 bytes, ratio 0.50)
nibble-packed (4 bits/frame):        1685 bytes
byte-RLE (mask,count) pairs:          215 runs -> 430 bytes
distinct masks present: [0, 1, 2, 8]
mask histogram: 0:2252  1:910  2:198  8:9
frames with more than one bit set: 0
frames with mask 3 (both directions = about-face): 0
```

Findings:

1. **The bitmask format is half the size of `demoType` for the same content**, because it
   drops the 4-byte frame field. 3369 bytes vs 6702.
2. **Only four masks occur, and never more than one bit at a time.** That confirms §8.5
   finding 3 (key 2 unused) *and* §8.4.1 asymmetry #2 (the about-face combination
   `kInRight | kInLeft` = 3 is not representable in `demoType` and never occurs in the
   shipped data). The bitmask format represents it natively — see `ApplyInput` step 11.
3. **RLE gets to 430 bytes** for the archival case, so a demo-file format can be **15.6×**
   smaller than the original's 6702-byte `demo` resource (7.8× smaller than the 3369-byte
   raw one-byte-per-frame stream) while being strictly more expressive. Not needed for the
   live protocol.
4. The mask-0 count (2252 of 3369 frames, 67 %) shows that most frames have no input at all,
   which is why the redundancy scheme in §10.4.4 is cheap.

## 10.4.4 `MsgInputFrames` — the live packet

One message per simulated frame per peer. All multi-byte fields **big-endian** (matching the
original's on-disk convention, §6.3.2, and network byte order).

| Offset | Field | Type | Bytes | Value / meaning |
| --- | --- | --- | --- | --- |
| 0 | `magic` | `uint16` | 2 | `0x474C` (`"GL"`) |
| 2 | `version` | `uint8` | 1 | protocol version; **1** for this document |
| 3 | `msgType` | `uint8` | 1 | `0x01` = `MsgInputFrames` |
| 4 | `matchID` | `uint32` | 4 | low 32 bits of `matchSeed`; rejects stale packets from a previous match |
| 8 | `senderSlot` | `uint8` | 1 | 0 = player 1 (`kPlayer1`), 1 = player 2 (`kPlayer2`) |
| 9 | `count` | `uint8` | 1 | number of input words that follow, 1..32 |
| 10 | `firstFrame` | `uint32` | 4 | `gameFrame` of `words[0]`; the target frame of the newest word is `firstFrame + count - 1` |
| 14 | `ackFrame` | `uint32` | 4 | highest contiguous frame this peer has received from the other |
| 18 | `words[count]` | `uint8[]` | `count` | `InputWord` per §10.4.2, oldest first |

Fixed header = **18 bytes**. With `count` = 8 (the recommended redundancy) the packet is
26 bytes, sent 30×/s → **780 B/s each way**, plus the checksum message every 8 frames
(§10.4.5, 22 bytes at 3.75/s ≈ 83 B/s). Under 1 KB/s per direction. For comparison, at
`count` = 2 (no redundancy beyond the delay window) it is 20 bytes → 600 B/s.

Rules:

* `firstFrame` must always be `>= gameFrame + 1` for at least one word — i.e. never send
  only history.
* A receiver stores `words[i]` at frame `firstFrame + i` **only if that slot is empty**.
  Retransmissions must never overwrite, because a resend of an already-consumed frame with
  a different value is a protocol violation, and silently accepting it turns a bug into a
  desync.
* A word for a frame already simulated is discarded, **but if it differs from what was
  simulated, that is a fatal protocol error** — log and abort, do not paper over it.
* `count` must be `>= kInputDelay` so the delay window is always fully covered by the newest
  packet.
* This is the *complete* per-frame protocol. **No entity state, no positions, no velocities,
  no scores are ever sent during play.**

## 10.4.5 `MsgChecksum` and `MsgAck`

```
MsgChecksum   (msgType 0x02, 22 bytes)
  0  magic uint16   0x474C
  2  version uint8  1
  3  msgType uint8  0x02
  4  matchID uint32
  8  senderSlot uint8
  9  (pad) uint8    0
 10  frame uint32   gameFrame the hash was taken at; always (frame & 7) == 0
 14  hash uint64    §10.4.6
```

`MsgAck` is unnecessary as a separate message because `MsgInputFrames.ackFrame` carries the
acknowledgement. A dedicated `MsgAck` (msgType `0x03`, the 18-byte header with `count` = 0)
is worth defining anyway for the paused and stalled states, where a peer has no new input to
send but must keep the liveness signal flowing.

`MsgBye` (msgType `0x04`, 18-byte header, `count` = 0) — voluntary disconnect. On receipt,
run `EnterDisconnectFlow()` (§10.2.6) immediately rather than waiting out
`kStallTimeout`.

## 10.4.6 What to hash, and in what order

The checksum must cover everything the simulation reads and nothing it does not. Getting
this wrong in either direction is costly: too little and desyncs go undetected until they
are visible; too much and you get false positives from render state.

**Hash input, in exactly this order** (FNV-1a 64 or xxhash64; the algorithm just has to be
pinned in `version`):

```
HashSimState():
   1. gameFrame (uint32), evenFrame (uint8)
   2. thisRoomNumber, previousRoom, leftThresh, rightThresh          (int16 each)
   3. theScore (int32), mortals, batteryTotal, bandsTotal, foilTotal,
      numStarsRemaining                                              (int16 each)
   4. otherPlayerEscaped, activeRectEscaped (int16), onePlayerLeft, playerDead,
      playerSuicide, firstPlayer, takingTheStairs (uint8 each), saidFollow (int16)
   5. gameOver (uint8), countDown (int16), netPaused (uint8)
   6. for g in [theGlider, theGlider2]:            // player 1 FIRST, always
   7.     g.dest, g.destShadow, g.enteredRect      (4 × int16 each, top/left/bottom/right)
   8.     g.hVel, g.vVel, g.wasHVel, g.wasVVel, g.vDesiredVel, g.hDesiredVel  (int16)
   9.     g.mode, g.frame, g.wasMode               (int16)
  10.     g.facing, g.tipped, g.sliding, g.fireHeld, g.which,
          g.heldLeft, g.heldRight, g.dontDraw, g.ignoreLeft, g.ignoreRight,
          g.ignoreGround                            (uint8 each)
  11. numDynamics (int16); dinahs[0..numDynamics-1]: type, count, frame, timer,
      position, room, hVel, vVel (int16), byte0, byte1 (uint8), moving, active (uint8),
      dest, whole (4 × int16 each)
  12. nHotSpots (int16); hotSpots[0..nHotSpots-1]: bounds (4 × int16), action, who (int16),
      isOn, stillOver, doScrutinize (uint8)
  13. numBands (int16), bandHitLast (int16); bands[0..1]: dest, mode, count, hVel, vVel
  14. triggers[0..15]: object, room, index, timer, what (int16), armed (uint8)
  15. numGrease (int16); grease[0..numGrease-1]: dest, mapNum, mode, who, where,
      start, stop, frame, hotNum (int16), isRight (uint8)
  16. clockFrame (int16); numPendulums; pendulums[...]: mode, where, who, link (int16),
      toOrFro, active (uint8)
  17. numFlames, numTikiFlames, numCoals (int16) and each array's mode fields
  18. numStars (int16); theStars[...]: mode, who, link, where (int16)
  19. numSparkles (int16); sparkles[0..2]: mode (int16), bounds
  20. numFlyingPts (int16); flyingPoints[0..2]: mode, start, stop, loops, hVel, vVel, dest
  21. numShredded (int16); shreds[0..3]: frame (int16), bounds
  22. thePhone.nextRing, .rings, .delay; theChimes.nextRing (int16 each)
  23. numMasterObjects (int16); masterObjects[0..numMasterObjects-1]:
      roomNum, objectNum, roomLink, objectLink, localLink, hotNum, dynaNum (int16)
      and theObject (12 bytes, §4.3.1)
  24. for each synchronised stream in fixed label order
      (chimes, dyna, phone, roomload): the stream's draw counter (uint64)
  25. a rolling hash of the mutable house: for every room, `visited` (uint8) plus every
      object's 12 bytes (§4.3.1) — see note below
```

**Explicitly excluded**, per §10.1.9:

`src`, `mask`, `whole`, `wholeShadow`, `clip` on the gliders; `savedMaps[]` in its entirety
(§10.3.4 removes it from the simulation); `numSavedMaps`; `work2Main[]`, `back2Work[]`,
`numWork2Main`, `numBack2Work`; `nextFrame`; `displayedScore`; `playedTikTok`;
`shadowVisible`, `hasMirror`, `tvInRoom`, `tvWithMovieNumber`; `numTempManholes`,
`tempManholes[]`; `numLights`, `numNeighbors`, `thisTiles[]`, `localRoomsDest[]`,
`isStructure[]`; `StreamCosmetic`'s counter; `theKeys`.

**Step 25 needs care.** The house is the largest component and hashing all of it every 8
frames is wasteful (§10.2.2: 866 + 348·n bytes). Two acceptable strategies:

* **Dirty-tracked incremental hash.** Maintain a running XOR-of-per-room-hashes, and
  recompute a room's hash only when `SetObjectState`, `SetObjectsToDefaults` or
  `HandleRoomVisitation` writes to it. The three write sites are
  `GliderPRO/Sources/Objects.c:366-698`, `GliderPRO/Sources/Play.c:603-708` and
  `GliderPRO/Sources/Transit.c:430-445`. This is cheap and exact.
* **Hash only the current room plus its eight neighbours** in the per-8-frame checksum, and
  hash the whole house at every room transition. Cheaper still, slightly weaker.

Note the trap hazard 23 identifies: `thisRoom` is a **by-value 348-byte copy**
(`ForceThisRoom`, `GliderPRO/Sources/Room.c:369-386`; the assignment is `*thisRoom = (*thisHouse)->rooms[roomNumber];` at `:379`; the reverse copy is `CopyThisRoomToRoom` at `:354-365`), not a pointer into the house. `SetObjectState` writes
*both* the house handle and `thisRoom` when `room == thisRoomNumber`
(`GliderPRO/Sources/Objects.c:465-471`), and `HandleRoomVisitation` writes both
(`GliderPRO/Sources/Transit.c:439-444`). **Hash both**, or a port that mirrors only one will
pass its own checksum while being wrong.

**Desync response.** On mismatch: stop the simulation immediately, do not attempt to
resynchronise. Lock-step has no recovery path — the states have already diverged and every
subsequent frame compounds it. Dump, on both peers: `gameFrame`, the last 64 input words
for both players, all stream draw counters, and a component-wise hash breakdown (one hash
per numbered step above) so the first differing component identifies the subsystem. In
practice the stream counters (step 24) and `numDynamics`/`nHotSpots` (steps 11–12) identify
almost every real bug.

## 10.4.7 `MsgMatchStart` — the setup handshake

```
MsgHello      (msgType 0x10)
  0..7   header (magic, version, msgType, matchID = 0)
  8      senderSlot uint8       // proposed; resolved below
  9      (pad) uint8 0
 10      nonce uint64
 18      protocolVersions uint16 bitmap of supported versions
 20      houseNameLen uint8
 21      houseName[houseNameLen]  // Pascal-ish; the original stores a Str255 (Main.c prefs)
 ..      houseHash uint64         // hash of the entire house file bytes
 ..      kInputDelay uint8        // proposed
 ..      numNeighborsView uint8   // INFORMATIONAL ONLY -- see §10.3.5 R3

MsgMatchStart (msgType 0x11)
  0..7   header (matchID = low 32 bits of matchSeed)
  8      slotAssignment uint8     // 0: sender is player 1; 1: sender is player 2
  9      kInputDelay uint8        // agreed = max(proposed_A, proposed_B)
 10      matchSeed uint64
 18      houseHash uint64         // echoed for confirmation
 26      startFrame uint32        // both peers begin at this gameFrame (normally 0)
```

`houseHash` is mandatory. The house file **is** simulation input: §4.4 verifies the on-disk
layout, and every object's `data` bytes (blower directions, appliance `delay`, grease
`length`, `initial` states) come straight from it. Two peers running different builds of the
same-named house desync on the first room load, and the only way to make that a clean error
instead of a mystery is to compare hashes up front.

`numNeighborsView` is deliberately informational. Per §10.3.5 R3 the simulation always
behaves as if all nine neighbours exist, so peers may legitimately disagree about what they
render. Sending it lets the UI warn ("your opponent can see more of the house than you
can") without making it a compatibility gate.

Slot assignment must be deterministic and symmetric: the peer with the numerically smaller
nonce becomes player 1 (`kPlayer1`, `which == TRUE`). Ties are impossible in practice with
64-bit nonces; if they occur, both peers re-nonce.

**Why slot assignment matters more than it looks.** Player 2's identity is not cosmetic:

| Asymmetry | Evidence |
| --- | --- |
| Only player 1 can trigger `ForceKillGlider` | `GliderPRO/Sources/Input.c:368` `(thisGlider->which)` |
| Player 2 is idled for 30 frames at game start | `GliderPRO/Sources/Play.c:189` `TagGliderIdle(&theGlider2)` |
| Player 1 is checked first for every hotspot, so player 1 wins ties | `GliderPRO/Sources/Interactions.c:1639` vs `:1657`; §3.5.1 |
| Player 1 is checked first in `HandleInteraction` and in `HandleGlider` | `GliderPRO/Sources/Play.c:452-453`, `:460-461`; `GliderPRO/Sources/Interactions.c:1705-1706` |
| The rubber-band recoil transfer differs per player | `GliderPRO/Sources/RubberBands.c:163-166` vs `:186-193` |
| Player 2's keys are hard-coded modifiers and unremappable | `GliderPRO/Sources/InterfaceInit.c:148-151`; §9.2 |

The last one dissolves in a networked port (each peer maps its own physical keys into a
word), but the first four are **rules**, and both peers must agree on who is who before
frame 1.

## 10.4.8 `MsgProgress` — the two-player save the original never had

§9.7's conclusion: `SaveGame` returns immediately in two-player mode
(`GliderPRO/Sources/SavedGames.c:309-310`), and `gameType` (40 bytes, §4.3.4) describes
**one** glider. A networked port needs a progress record for two reasons: resuming a
dropped match, and letting a rejoining peer catch up without replaying thousands of frames.

Modelled on `gameType` (`GliderPRO/Headers/GliderStructs.h:116-134`) and `game2Type`
(`:144-164`), keeping the original field order so the mapping is obvious:

```
MsgProgress (msgType 0x20)
  --- header ---
  0   magic uint16      0x474C
  2   version uint8     1
  3   msgType uint8     0x20
  4   matchID uint32
  8   senderSlot uint8
  9   (pad) uint8 0
 10   frame uint32      the gameFrame this snapshot was taken at

  --- shared game state (the fields gameType has, promoted to shared) ---
 14   recVersion int16      cf. gameType.version = kSavedGameVersion (0x0200), SavedGames.c:318
 16   wasStarsLeft int16    cf. gameType.wasStarsLeft <- numStarsRemaining, :319
 18   score int32           cf. gameType.score       <- theScore, :324
 22   energy int16          cf. gameType.energy      <- batteryTotal, :327
 24   bands int16           cf. gameType.bands       <- bandsTotal, :328
 26   foil int16            cf. gameType.foil        <- foilTotal, :332
 28   numGliders int16      cf. gameType.numGliders  <- mortals, :331   (SHARED, §9.4)
 30   roomNumber int16      cf. gameType.roomNumber  <- thisRoomNumber, :329
 32   previousRoom int16    (no gameType equivalent; needed by DetermineRoomOpenings)

  --- two-player rendezvous state (NOTHING in gameType covers this) ---
 34   otherPlayerEscaped int16     GliderDefines.h:596-608
 36   activeRectEscaped  int16     Interactions.c:42
 38   onePlayerLeft uint8
 39   playerDead uint8
 40   playerSuicide uint8
 41   firstPlayer uint8
 42   saidFollow int16             capped at 3, Modes.c:13
 44   takingTheStairs uint8
 45   (pad) uint8 0

  --- per-player, TWICE: player 1 (which == kPlayer1) first ---
 46   whereH int16          cf. gameType.where.h <- theGlider.dest.left,  :322
 48   whereV int16          cf. gameType.where.v <- theGlider.dest.top,   :323
 50   enteredH int16        theGlider.enteredRect.left   (respawn point, Player.c:1534)
 52   enteredV int16        theGlider.enteredRect.top
 54   hVel int16
 56   vVel int16
 58   gliderState int16     cf. gameType.gliderState <- theGlider.mode, :330
 60   wasMode int16
 62   frame int16
 64   facing uint8          cf. gameType.facing <- theGlider.facing, :334
 65   dontDraw uint8
 (66..85 = the same 20 bytes for player 2)

  --- stream positions ---
 86   matchSeed uint64
 94   drawCount[4] uint64   chimes, dyna, phone, roomload -- in that fixed label order

  --- house state (the part gameType cannot express; cf. game2Type.savedData[]) ---
102   nRooms uint16
104   rooms[nRooms]: { visited uint8 ; pad uint8 ; objects[24] × 12 bytes }
      // = 2 + 288 = 290 bytes per room; cf. savedRoom (292 bytes, §4.3.5) minus its
      // unusedShort/unusedByte
```

Fixed part = **102 bytes** + 8 (matchSeed) + 32 (counters) is already counted; the variable
part is 290 × `nRooms`. For a 100-room house that is 29 102 bytes total. That is a one-shot
message, not a per-frame one.

Design notes:

* **`numGliders` is shared, not per-player.** `mortals` is one global counter; two-player
  games start with `kInitialGliders` (2) **plus** another `kInitialGliders`
  (`GliderPRO/Sources/Play.c:341-343`) for 4 shared lives, and `OffAMortal` decrements the
  single counter (`GliderPRO/Sources/Player.c:1492`). §9.4.
* **`gameType.timeStamp` is dropped.** It is `GetDateTime` (`SavedGames.c:320-321`) — a
  wall-clock value with no simulation meaning. Keep it in a file header if you want, never
  in the simulation.
* **`gameType.unusedLong`, `unusedLong2`, `unusedShort` are dropped.** They are written as
  0 (`SavedGames.c:325`, `:326`, `:333`).
* **`showFoil` is a global, not a glider field.** §4.3.4's note is right: `gliderType` has
  no `showFoil` member (`GliderPRO/Headers/GliderStructs.h:200-216`); `SaveGame` reads the
  global at `SavedGames.c:335`. Put it in the shared block if you need it; it is a
  scoreboard flag (`Play.c:53`).
* `savedRoom` is 292 bytes because of its `unusedShort` and `unusedByte`
  (`GliderPRO/Headers/GliderStructs.h:137-138`, §4.3.5). Do not carry them.
* **Resuming from `MsgProgress` is not the same as resuming a lock-step match.** All
  transient per-room state — `dinahs`, `hotSpots`, `flames`, `grease`, `bands`, `triggers`,
  `sparkles`, `masterObjects` — is *not* in the packet, because it is regenerated
  deterministically by `LoadRoom` (§10.3.5) from `(house bytes, roomNumber, stream
  positions)`. That is only true if §10.3.5's contract holds. **`MsgProgress` is therefore
  a test of §10.3.5**: if resuming from a progress packet produces a different room than
  the peer that never dropped, the room-load contract is broken.

## 10.4.9 The demo as a protocol test vector

The `'demo' 128` resource can be transcoded into `MsgInputFrames` streams and used as a
network-layer test with no second human. The transcoding is exact and reversible for this
particular resource because it contains no duplicate frames (§8.5 finding 2):

```
DemoToWire():
   1. read 6702 bytes from the 'demo' 128 resource        // kDemoLength, GliderDefines.h:625
   2. parse as 1117 big-endian records {int32 frame; int8 key; int8 pad}   // pad IGNORED
   3. words <- array of 3369 zero bytes, indexed by frame 46..3414
   4. for each record: words[frame - 46] |= (1 << key)
   5. emit MsgInputFrames packets of 8 words each for slot 0
   6. emit all-zero words for slot 1 (the demo is single-player, Play.c:479)
```

Three tests it enables, in increasing strength:

| Test | Setup | Passes if |
| --- | --- | --- |
| T1 sim/render cut | one process, `Draw` on vs `Draw` never called | the 3369 per-frame hashes are identical (§10.1.10) |
| T2 loopback lock-step | two processes on one machine, real sockets, `kInputDelay` = 2 | both produce the same 3369 hashes as T1, and no checksum mismatch fires |
| T3 adversarial network | T2 with injected 200 ms jitter, 5 % loss, and reordering | identical hashes; stall counter rises but never reaches `kStallTimeout` |

T2 and T3 are the only tests that exercise the barrier, the redundancy window and the
retransmit-must-not-overwrite rule. They cost nothing to run continuously in CI.

Caveat: the demo drives `theGlider` in *single-player* mode
(`GliderPRO/Sources/Play.c:478-479`), which takes the `else` arm of `PlayGame` and the
`else` arm of `HandleInteraction` (`GliderPRO/Sources/Interactions.c:1710`). So T1–T3
validate the transport, the cut and the RNG contract, **but not the two-player rendezvous**.
For that, hand-author a second input stream; §10.6.5 lists the state transitions it must
cover.

## 10.4.10 Encoding rules

| Rule | Reason |
| --- | --- |
| All integers big-endian | matches the original's on-disk convention (§6.3.2) and network byte order |
| All widths explicit (`int16`, `uint32`, …) | the original's `long` is **32 bits**; a 64-bit `long` silently changes the size of every struct that *contains* one — `demoType`, `gameType`, `scoresType`, `houseType`, `gliderType` (porting note 8, §4.3.3) |
| No padding bytes with undefined contents | `demoType.padding` is uninitialised heap and made the shipped resource non-reproducible (§8.5 finding 7) |
| Explicit `count`/`nRooms` before every variable-length array | the original relies on `kDemoLength` being a compile-time constant (`StructuresInit2.c:295`) |
| Every message carries `version` and `matchID` | the original carries neither |
| `Boolean` on the wire is `0` or `1` only | Mac `Boolean` is a `Byte`; the original stores `kFaceRight = TRUE` and `kPlayer1 = TRUE` (`GliderDefines.h:554-556`), and C `TRUE` is 1 here, but a hash over a `Boolean` field must not see 0xFF |
| Reject unknown `msgType` silently; reject wrong `version` loudly | forward compatibility vs. safety |

---

# 10.5 The authoritative fix for the shared-`HandleInteraction` problem

`determinism.md` §9.8 names this as the one property working against a networked port:

> "The one property working *against* a networked port is that both gliders' interactions
> are resolved inside a single `HandleInteraction()` call
> (`GliderPRO/Sources/Interactions.c:1691-1711`) against shared mutable room state — so
> 'each machine simulates its own player' (mode (a)) is not a straight code split."

## 10.5.1 The function, and precisely why it resists splitting

`GliderPRO/Sources/Interactions.c:1691-1711` — transcribed line for line from the
CR→LF-converted source, so the `//` labels are the real line numbers (the function is
21 lines: `void` at `:1691`, closing brace at `:1711`; `:1712` is blank and `:1713` is the
`FlagStillOvers` banner comment):

```c
void HandleInteraction (void)                                  // :1691
{                                                              // :1692
	CheckForHotSpots();                                    // :1693
	if (twoPlayerGame)                                     // :1694
	{
		if (onePlayerLeft)                                 // :1696
		{
			if (playerDead == kPlayer1)                    // :1698
				CheckGliderInRoom(&theGlider2);            // :1699
			else
				CheckGliderInRoom(&theGlider);             // :1701
		}
		else
		{
			CheckGliderInRoom(&theGlider);                 // :1705
			CheckGliderInRoom(&theGlider2);                // :1706
		}
	}
	else                                                   // :1709
		CheckGliderInRoom(&theGlider);                     // :1710
}                                                              // :1711
```

Four distinct entanglements, each independently fatal to a per-player code split:

**(E1) `CheckForHotSpots` is a single loop over shared state that tests both gliders per
iteration.** `GliderPRO/Sources/Interactions.c:1627-1687`, §3.5.1. The loop body writes
`hotSpots[i].stillOver`, which is a *shared* anti-retrigger latch: it is cleared only if
**neither** glider is over the hotspot (`hitObject = false;` at `:1638`, then
`if (!hitObject) hotSpots[i].stillOver = false;` at `:1674-1675`).
Splitting the loop per player would mean each half sees a different `stillOver` history, so
switches and triggers would fire at different times.

**(E2) `SetObjectState` implements a one-winner rule.** `GliderPRO/Sources/Objects.c:460-462`
(the arm shared by **fourteen** `case` labels, `:446-459` — `IsThisValid`'s twelve plus
`kGreaseRt` and `kGreaseLf`; this is the file's only
`data.c.state` arm, so the range is unambiguous):

```c
		changed = ((*thisHouse)->rooms[room].objects[object].data.c.state == true);  // :460
		newState = false;                                                           // :461
		(*thisHouse)->rooms[room].objects[object].data.c.state = newState;           // :462
```

and every reward in `HandleRewards` is wrapped in `if (SetObjectState(...))`
(`GliderPRO/Sources/Interactions.c:767`, `:783`, `:799`, `:815`, `:832`, `:851`, `:873`,
`:893`, `:901`, `:921`, `:934`, `:957`; a thirteenth call sits in `HandleHotSpotCollision`
at `:998`). So the *first* caller in a frame gets `true` and the prize; the second gets
`false` and nothing. Whoever is tested first — player 1, always, because
`CheckForHotSpots` tests `theGlider` at `Interactions.c:1639` and `theGlider2` only at
`:1657` inside the same `for (i = 0; i < nHotSpots; i++)` (`:1632`) — wins ties. **This is the original's built-in conflict
resolution and it must be preserved exactly**, because it is what makes simultaneous pickup
of one battery award one battery.

**(E3) The rewards mutate shared scalars, and double them in two-player mode.**
Enumerated:

| Reward | Shared write | Doubling site |
| --- | --- | --- |
| `kPaper` | `mortals++` | `mortals++` again at `GliderPRO/Sources/Interactions.c:842-843` |
| `kBattery` | `batteryTotal += kBatterySupply` (50) or `= kBatterySupply` | `+= kBatterySupply` again at `:864-865` |
| `kBands` | `bandsTotal += kBandsSupply` (8) | `+= kBandsSupply` again at `:883-884` |
| `kFoil` | `foilTotal += kFoilSupply` (8) | `+= kFoilSupply` again at `:911-912` |
| `kHelium` | `batteryTotal -= kHeliumSupply` (150) or `= -kHeliumSupply` | `-= kHeliumSupply` again at `:970-971` |
| `kRedClock` / `kBlueClock` / `kYellowClock` / `kCuckoo` / `kInvisBonus` / `kStar` | `theScore +=` 100 / 300 / 500 / 1000 / `data.c.points` / 5000 | **not doubled** |
| `kStar` | `numStarsRemaining--`; `FlagGameOver()` if `<= 0` | not doubled |

All five doublings are gated on `(twoPlayerGame) && (!onePlayerLeft)`, so they stop
automatically when a player is out — which is what makes the disconnect flow in §10.2.6
free. Supply constants from §3.5.5 / `GliderPRO/Sources/Interactions.c:13-19`.

**(E4) `CheckGliderInRoom` can perform a room transition mid-loop.**
`GliderPRO/Sources/Interactions.c:689-752` → `CheckEscapeUpTwo` → `MoveRoomToRoom`
(`:188`) → `ReadyLevel()` → `DrawLocale()` → the entire room is torn down and rebuilt
(§10.2.1(iii)) — **between the `CheckGliderInRoom(&theGlider)` at `:1705` and the
`CheckGliderInRoom(&theGlider2)` at `:1706`.** So player 2's boundary check may run against
a *completely different room* than player 1's did, one line earlier. This is not a bug; it
is how the rendezvous works (the second arriver's call is what completes the transition).
But it means the two calls are not independent even in principle.

## 10.5.2 The fix: do not split it

> **Authoritative fix: keep `HandleInteraction` exactly as written, and give every peer the
> whole world.** Do not adopt "each machine simulates its own player". Adopt "every machine
> simulates every player". The only thing that crosses the wire is input (§10.4.2), and the
> barrier in §10.2.5 guarantees both peers enter phase 5 with identical state and identical
> input.

Under that model E1–E4 stop being problems and become *requirements*:

| Entanglement | Under lock-step |
| --- | --- |
| E1 shared `stillOver` latch | correct on both peers automatically, because both run the same loop over the same array |
| E2 one-winner `SetObjectState` | correct and identical, because the call order is fixed (player 1 first) and both peers execute it |
| E3 shared/doubled scalars | correct, because both peers apply both players' pickups |
| E4 mid-loop room transition | correct, because both peers transition on the same frame at the same statement |

**The `HandleInteraction` "problem" is not a problem for lock-step. It is a problem only for
the topology `determinism.md` §9.8 was contemplating (mode (a), per-player simulation), and
the correct response is to reject that topology.** Concretely, the assignment's mode (a) —
"each machine simulates its own player" — is unimplementable here without redesigning the
prize, hotspot and transition rules, and would change the game.

## 10.5.3 What still has to change, and it is small

Keeping `HandleInteraction` verbatim is not the same as keeping the file verbatim. Three
edits are required, all traceable to §10.1's Coupling B.

**(1) Remove the graphics from `HandleRewards`.** Each reward currently does
`RestoreFromSavedMap(thisRoomNumber, objectNum, false)` then `RedrawAllGrease()`
(the 20 sites listed in §10.1.1). Replace with a render-side event:

```
   ...if SetObjectState(...) then
        PlaySound(...)                                   // fire-and-forget, hazard 34
        renderEvents.append(EraseObject{room, objectNum}) // was RestoreFromSavedMap
        AddFlyingPoint(&bounds, points, hVel/2, vVel/2)   // SIM: keep
        thisGlider->hVel /= 4 ; thisGlider->vVel /= 4     // SIM: keep exact integer division
        theScore += <points>                              // SIM: keep
        renderEvents.append(RedrawGrease{})               // was RedrawAllGrease
```

The `/= 2` and `/= 4` velocity dampings are **simulation**, and they are C integer division
truncating toward zero — for negative velocities, `-7 / 4 == -1`, not `-2`. Go's `/` on
`int16` behaves identically, so this ports cleanly, but a port that "improves" it to a
shift (`-7 >> 2 == -2`) changes the physics.

Which rewards use `/= 2` vs `/= 4`: score prizes (clocks, cuckoo, invisible bonus) use
`/= 4` (`GliderPRO/Sources/Interactions.c:774-775`, `:790-791`, `:806-807`, `:823-824`,
`:926-927`); supply prizes (paper, battery, bands, foil, helium) use `/= 2` (`:839-840`,
`:858-859`, `:880-881`, `:908-909`, `:964-965`); `kStar` damps **neither**
(`:934-953` has no velocity write). `kGreaseRt`/`kGreaseLf` and `kSparkle` and `kSlider` do
nothing to velocity.

**(2) Split the two `doSparkle = true` calls.** `HandleHotSpotCollision`'s
`RestoreFromSavedMap(roomLinked, objectLinked, true)` at
`GliderPRO/Sources/Interactions.c:1049` and `:1054` spawns a sparkle and plays a sound
from inside a "restore graphics" helper (§10.1.7). Make the spawn explicit at the call site
and leave the erase on the render side.

**(3) Move `CopyRectBackToWork` + `AddRectToWorkRects` out.**
`GliderPRO/Sources/Interactions.c:1033-1034`. Pure draw.

Everything else in `Interactions.c` — the hotspot loop, the escape state machine, the
one-winner rule, the doubling, `SectGlider`'s 5-pixel `scrutinize` inset (§3.5.2),
`FlagStillOvers` (`:1715`), `WebGlider` (`:1736`) with `kKillWebbedGlider` = 150 defined
*inside the function body* at `:1738` and tested against `thisGlider->wasMode` at `:1770` — stays
byte-for-byte.

## 10.5.4 The one reordering worth considering, and why to reject it

There is an argument for making `HandleInteraction` order-independent, so that "who was
checked first" stops mattering: collect both gliders' hotspot intersections into a list,
then resolve. It is tempting because E2's player-1-wins rule looks arbitrary.

**Reject it.** Three reasons, in order of weight:

1. **It changes the game.** Player 1 currently wins every tie. Making ties fair (say,
   awarding to the glider whose centre is closer) is a rules change, and this document's
   job is a faithful port (§7.4: "What is *already* right, and must not be 'improved'").
2. **It does not help lock-step at all.** Lock-step needs *determinism*, not *fairness*.
   Player-1-wins is perfectly deterministic.
3. **It would need to reorder E4.** The mid-loop room transition means the second glider's
   check legitimately runs against a new room. A "collect then resolve" design has to
   decide what a collected intersection means after the room changed underneath it. That is
   a new and subtle correctness problem in exchange for nothing.

The only ordering change that *is* required is the trivial one already in §10.2.5: input for
both players must be applied (phase 4) before `HandleInteraction` (phase 5), which is what
`PlayGame` already does (`GliderPRO/Sources/Play.c:452-454`).

## 10.5.5 A checklist for the port's `Interactions.c`

| Property | Must hold | Evidence |
| --- | --- | --- |
| Hotspot loop runs `i = 0 .. nHotSpots-1` ascending, body gated on `hotSpots[i].isOn` | yes | `Interactions.c:1632`, `:1634` |
| Player 1 tested before player 2 for every hotspot | yes | `Interactions.c:1639` (P1) then `:1657` (P2) |
| `stillOver` cleared only when neither glider hit | yes | `Interactions.c:1674-1675`, via `hitObject`; single-player equivalent at `:1683` |
| Dead player skipped via `playerDead == kPlayer2` / `== kPlayer1` inside each arm | yes | `Interactions.c:1644` (P1 arm), `:1662` (P2 arm) |
| `CheckGliderInRoom` runs only for modes `kGliderNormal`, `kGliderFaceLeft`, `kGliderFaceRight`, `kGliderBurning` | yes | `Interactions.c:691-694` |
| A burning glider fades out instead of escaping, at all four boundaries | yes | `Interactions.c:698-702`, `:711-715`, `:727-731`, `:740-744` |
| `CheckGliderInRoom(&theGlider)` before `(&theGlider2)` | yes | `Interactions.c:1705-1706` |
| Vertical escape checked before horizontal, in **separate** `if` chains | yes | §3.5.3; a corner can trigger both in one frame |
| `SetObjectState` returns `changed` and gates the reward | yes | `Objects.c:460` (set), `:698` (return) |
| Supply doubling gated on `(twoPlayerGame) && (!onePlayerLeft)` | yes | `Interactions.c:842`, `:864`, `:883`, `:911`, `:970` |
| Velocity damping is C integer division, not a shift | yes | §10.5.3 |
| `SectGlider` does **not** call `SectRect`. It hand-rolls four **closed / inclusive** edge tests, so rects that merely *touch* DO intersect | yes | `Interactions.c:118-127`: `if (theRect->bottom < glideBounds.top) itHit = false; else if (theRect->top > glideBounds.bottom) … else if (theRect->right < glideBounds.left) … else if (theRect->left > glideBounds.right) … else itHit = true;` — strict `<`/`>`, so equality counts as a hit. (§3.5.2 says `SectRect`/`InsetRect`; that is wrong and should be corrected there too.) |
| `SectGlider` adds `+6` to `gliderRect.top` when burning, and insets 5 px when `scrutinize` | yes | `Interactions.c:101-131`; the burning adjust is `glideBounds.top += 6` at `:107-108`, and the inset is written out by hand (`left += 5; top += 5; right -= 5; bottom -= 5`) at `:112-115`, **not** via `InsetRect` |

## 10.5.6 The `newState` global — a latent trap in `SetObjectState`

Worth flagging here because a Go port's zero-initialisation changes it.
`GliderPRO/Sources/Objects.c:78` declares a **file-scope** `Boolean newState;`, and the
prize branch writes `newState = false;` before using it
(`GliderPRO/Sources/Objects.c:461`, then stores it into the house at `:462`). Other branches
read and write it too. Because it is a
file-scope global it is zero-initialised in C as well, so this one is benign — but the
neighbouring hazard is not: `SetObjectState`'s `switch` has **no outer `default:` label**,
and `changed` is a plain local (`GliderPRO/Sources/Objects.c:369`). An object `what` value
that matches no `case` falls through to `return (changed)` at `:698` with `changed`
**uninitialised**. In the shipped build the value is whatever was on the stack; in Go it is
`false`.

Go's behaviour (`false` = "nothing changed" = "no reward") is the safe reading and matches
what every reachable `case` that means "not a prize" does explicitly (`changed = false` at
`:692`). Adopt it, and record the divergence. The same class of bug appears in
`FollowTheLeader`'s `oneOrTwo` (§10.6.7) and in `DidBandHitDynamic` (hazard 17), and in
`HandleSwitches`'s `bounds` (§10.5.7, which unlike the other three is reachable).

## 10.5.7 `HandleSwitches` reads an uninitialised `Rect` — and it *is* reachable

The three uninitialised-local bugs above are unreachable in the shipped game. This one is
not, and it was missed by Part 7's hazard table. `HandleSwitches`
(`GliderPRO/Sources/Interactions.c:985-1156`) opens with

```c
985  void HandleSwitches (hotPtr who)
986  {
987      Rect      newRect, bounds;      // `bounds` is NEVER assigned anywhere in the function
...
1001     newRect = who->bounds;          // the only assignment: to `newRect`, from `who->bounds`
...
1049     RestoreFromSavedMap(roomLinked, objectLinked, true);
1050     AddSparkle(&bounds);            // reads the uninitialised local
```

The `switch (masterObjects[linkIndex].theObject.what)` at `:1038` reaches `:1049-1050` for
nine prize types (`kRedClock`, `kBlueClock`, `kYellowClock`, `kPaper`, `kBattery`, `kBands`,
`kFoil`, `kStar`, `kHelium`, `:1040-1048`), i.e. on **every** switch or knife-switch wired to
a prize — ordinary gameplay, not a corner case. Two consequences:

1. **The sparkle is at a garbage rectangle.** `AddSparkle`
   (`GliderPRO/Sources/DynamicMaps.c:169-193`) offsets `*theRect` by `playOriginH/V` and
   centres `sparkleSrc[0]` in it, so a stack-garbage `Rect` places the sparkle anywhere,
   including far off-screen.
2. **It is a *second*, redundant sparkle.** `RestoreFromSavedMap(..., true)` at `:1049` has
   already emitted a correctly-placed one from the saved map's own `dest`
   (`GliderPRO/Sources/DynamicMaps.c:155-157`: `bounds = savedMaps[i].dest;
   QOffsetRect(&bounds, -playOriginH, -playOriginV); AddSparkle(&bounds);`). The `:1050` call
   is almost certainly a copy-paste leftover; the author's intent was probably
   `AddSparkle(&newRect)` (the switch's own rect) or nothing at all.

For the port: **drop the `:1050` call entirely** and let `RestoreFromSavedMap`'s `doSparkle`
arm be the only sparkle. Do not "fix" it to `AddSparkle(&newRect)` — that would add a
sparkle on the switch that the original never reliably drew. This is cosmetic and so does
not enter the checksum (§10.4.6), but it *does* consume a `sparkles[]` slot
(`numSparkles`, capped at `kMaxSparkles`), so replicating or dropping it changes which
later sparkles find a free slot. Since `numSparkles` is cosmetic-only in this design, that
is safe; if a port ever checksums it, this call must be replicated bug-for-bug instead,
which is impossible (the value is undefined). One more reason to keep cosmetic state out
of the checksum.

---

# 10.6 The `otherPlayerEscaped` rendezvous on the wire

§3.5.4 documents the rendezvous as "already a distributed-agreement protocol implemented in
shared memory" and §9.8 says mapping it onto messages "is nearly mechanical". This section
does the mapping — and reaches a conclusion that saves a great deal of work.

## 10.6.1 There are two rendezvous variables, not one

§3.5.4 documents `otherPlayerEscaped`. There is a second one that the analysis so far has
not covered:

```c
short		otherPlayerEscaped, activeRectEscaped;      // Interactions.c:42
```

`activeRectEscaped` records **which hotspot index** the first arriver used, and the second
arriver must match it. Its five write sites and five matching read sites:

| Hotspot action | Write | Read (must match) |
| --- | --- | --- |
| `kTransportIt` | `Interactions.c:1398` | `Interactions.c:1405` |
| `kMailItLeft` | `Interactions.c:1444` | `Interactions.c:1452` |
| `kMailItRight` | `Interactions.c:1487` | `Interactions.c:1495` |
| `kDuctItDown` | `Interactions.c:1527` | `Interactions.c:1534` |
| `kDuctItUp` | `Interactions.c:1562` | `Interactions.c:1569` |

The read is always of the form

```c
			else if (otherPlayerEscaped == kPlayerTransportedOut)
				{
					if ((thisGlider->mode != kGliderInLimbo) &&
							(activeRectEscaped == index))          // Interactions.c:1405
					{
						StartGliderTransporting(thisGlider, who);
					}
				}
```

so the pair must use the **same physical transporter / mailbox / duct**, not merely the same
*kind* of exit. `activeRectEscaped` is never reset to a sentinel — it simply retains the
last value, which is harmless because it is only consulted while
`otherPlayerEscaped` names the matching class. **It is simulation state and belongs in the
checksum** (§10.4.6 step 4) and in `MsgProgress` (§10.4.8 offset 36).

Note there is no `activeRectEscaped` check for stairs (`kMoveItUp` / `kMoveItDown`,
`Interactions.c:1250-1283` and `:1285-1318`): two gliders may use two *different* staircases in the same room
and still rendezvous. That asymmetry is deliberate-looking and must be preserved.

## 10.6.2 The three-state variant — room-boundary escapes

For the four room boundaries, the entire protocol lives in one function per direction and
completes within phase 5. `CheckEscapeUpTwo` (`GliderPRO/Sources/Interactions.c:171-240`),
the canonical form:

```
   // reached only via CheckGliderInRoom's dest.top < kCeilingLimit test    // :696
   if topOpen:                                                   // :175
     if thisGlider->dest.top < kNoCeilingLimit:                  // :177
       if otherPlayerEscaped == kNoOneEscaped (-1):              // :179   FIRST ARRIVER
           otherPlayerEscaped <- kPlayerEscapedUp (-4)           // :181
           RefreshScoreboard(kEscapedTitleMode)                  // :182
           FlagGliderInLimbo(thisGlider, true)                   // :183   -> firstPlayer <- which
       else if otherPlayerEscaped == kPlayerEscapedUp:           // :185   SECOND ARRIVER, same exit
           otherPlayerEscaped <- kNoOneEscaped                   // :187
           MoveRoomToRoom(thisGlider, kAbove)                    // :188   BOTH move
       else:                                                     // :190   WRONG EXIT
           PlayPrioritySound(kDontExitSound, kDontExitPriority)  // :192
           offset <- kNoCeilingLimit (-10) - thisGlider->dest.top // :193
           thisGlider->vVel <- -thisGlider->vVel + offset        // :194   bounce back
```

The four direction functions and their state values. The *outer* trigger is in
`CheckGliderInRoom` (`GliderPRO/Sources/Interactions.c:696`, `:709`, `:725`, `:738`); the
*inner* escape test is inside each `…Two` function:

| Direction | Function | Line | Sets | Outer trigger | Inner test | Bounce formula |
| --- | --- | --- | --- | --- | --- | --- |
| Up | `CheckEscapeUpTwo` | `:171` | `kPlayerEscapedUp` (-4) | `dest.top < kCeilingLimit` (`:696`) | `topOpen` (`:175`) and `dest.top < kNoCeilingLimit` (-10) (`:177`) | `vVel <- -vVel + (kNoCeilingLimit - dest.top)` (`:193-194`) |
| Down | `CheckEscapeDownTwo` | `:283` | `kPlayerEscapedDown` (-5) | `dest.bottom > kFloorLimit` (`:709`) | `bottomOpen` (`:287`) and `dest.bottom > kNoFloorLimit` (`:289`) | symmetric, `:291-300`, second copy `:325-333` |
| Left | `CheckEscapeLeftTwo` | `:509` | `kPlayerEscapedLeft` (-3) | `dest.left < leftThresh` (`:725`) | `leftThresh == kLeftWallLimit` (`:513`) + `ignoreLeft` (`:515`) + `dest.left < kNoLeftWallLimit` (`:517`), **or** the no-wall `else` at `:548` | `hVel <- -hVel + (kNoLeftWallLimit - dest.left)` (`:533-535`, `:564-565`) |
| Right | `CheckEscapeRightTwo` | `:599` | `kPlayerEscapedRight` (-2) | `dest.right > rightThresh` (`:738`) | `rightThresh == kRightWallLimit` (`:603`) + `ignoreRight` (`:605`) + `dest.right > kNoRightWallLimit` (`:607`), **or** the no-wall `else` at `:638` | `hVel <- -hVel + (kNoRightWallLimit - dest.right)` (`:623-624`) |

**All four functions contain the handshake twice.** Verified by grepping
`otherPlayerEscaped` over the file:

| Function | First copy (set / clear) | Second copy (set / clear) | Why there are two |
| --- | --- | --- | --- |
| `CheckEscapeUpTwo` | `:181` / `:187` | `:215` / `:221` | second copy is the `thisBackground == kDirt` (2011) dirt-hole path (`:198-239`) |
| `CheckEscapeDownTwo` | `:293` / `:299` | `:327` / `:333` | same dirt-hole duplication |
| `CheckEscapeLeftTwo` | `:521` / `:527` | `:552` / `:558` | second copy is the `leftThresh != kLeftWallLimit` (no wall at all) path — note it has **no position test**, so it fires as soon as `CheckGliderInRoom`'s outer `dest.left < leftThresh` is true |
| `CheckEscapeRightTwo` | `:611` / `:617` | `:642` / `:648` | same, for `rightThresh != kRightWallLimit` |

The dirt-hole path is guarded by a tile test — the glider must be under a tile whose index
is 5 or 6, computed as `dest.left >> 6` and `dest.right >> 6` with both in `[0, 8)`
(`GliderPRO/Sources/Interactions.c:200-209`). A port must not factor the copies into one
helper without checking that the `else` branches differ: the *tile-miss* branch sets
`vVel <- kCeilingLimit - dest.top` (`:233`, `:236`, `:239`) while the *wrong-exit* branch
sets `vVel <- -vVel + offset` (`:194`, `:228`). Those are different formulas.

## 10.6.3 The four-state variant — stairs, transporters, mail, ducts

For animated exits the handshake is split across **two phases and two files**, because the
glider has to finish an animation before it is actually "out". The states come in
`Escaping` / `Escaped` pairs:

| Exit | Phase-5 write (intent) | Where | Phase-8 write (completion) | Where |
| --- | --- | --- | --- | --- |
| Up stairs | `kPlayerEscapingUpStairs` (-8) | `Interactions.c:1266` | `kPlayerEscapedUpStairs` (-6) | `Player.c:363` |
| Down stairs | `kPlayerEscapingDownStairs` (-9) | `Interactions.c:1301` | `kPlayerEscapedDownStairs` (-7) | `Player.c:491` |
| Transporter | *(none — writes `activeRectEscaped` only)* | `Interactions.c:1398` | `kPlayerTransportedOut` (-10) | `Player.c:627` |
| Mailbox | *(none — `activeRectEscaped` only)* | `Interactions.c:1444`, `:1487` | `kPlayerMailedOut` (-12) | `Player.c:1030`, `:1121` |
| Duct | *(none — `activeRectEscaped` only)* | `Interactions.c:1527`, `:1562` | `kPlayerDuckedOut` (-11) | `Player.c:734`, `:831` |

The up-stairs completion, `GliderPRO/Sources/Player.c:341-370`:

```c
			takingTheStairs = true;                                  // :344
			if (twoPlayerGame)                                        // :345
			{
				if (onePlayerLeft)                                    // :347
				{
					if (playerDead == kPlayer1)                       // :349
						MoveRoomToRoom(&theGlider2, kAbove);          // :350
					else
						MoveRoomToRoom(&theGlider, kAbove);           // :352
				}
				else
				{
					if (otherPlayerEscaped == kPlayerEscapedUpStairs) // :356  SECOND
					{
						otherPlayerEscaped = kNoOneEscaped;           // :358
						MoveRoomToRoom(thisGlider, kAbove);           // :359
					}
					else                                              // FIRST
					{
						otherPlayerEscaped = kPlayerEscapedUpStairs;  // :363
						RefreshScoreboard(kEscapedTitleMode);          // :364
						FlagGliderInLimbo(thisGlider, true);          // :365
					}
				}
			}
			else
				MoveRoomToRoom(thisGlider, kAbove);                   // :370
```

Note the inversion relative to §10.6.2: the *second*-arriver test comes first here, and the
first-arriver case is the `else`. Same semantics, different shape. The transporter
completion (`GliderPRO/Sources/Player.c:609-636`, tests `:620`, clears `:622`, sets `:627`)
has the §10.6.2 shape again. **Do not
assume a uniform template.**

So the four-state sequence for stairs is:

```
frame N   : glider A touches the up-stairs hotspot (phase 5)
            requires !A.heldRight and GliderInRect(A, bounds)     Interactions.c:1251
            requires A.mode not in {kGliderGoingUp, kGliderInLimbo}   Interactions.c:1263-1264
            otherPlayerEscaped <- kPlayerEscapingUpStairs (-8)   Interactions.c:1266
            RefreshScoreboard(kEscapedTitleMode)                 Interactions.c:1267
            StartGliderGoingUpStairs(A)                          Interactions.c:1268
                                                                 -> mode kGliderGoingUp (3)
frame N..M: A animates up the stairs (phase 8, MoveGliderUpStairs)
frame M   : A's animation completes (phase 8)
            otherPlayerEscaped is NOT kPlayerEscapedUpStairs, so:
            otherPlayerEscaped <- kPlayerEscapedUpStairs (-6)     Player.c:363
            FlagGliderInLimbo(A, true) -> mode kGliderInLimbo (21), firstPlayer <- A.which
frame P   : glider B touches the same or another up-stairs hotspot
            otherPlayerEscaped == kPlayerEscapedUpStairs, so:
            StartGliderGoingUpStairs(B)                          Interactions.c:1276
            (otherPlayerEscaped stays -6)
frame P..Q: B animates
frame Q   : B's animation completes
            otherPlayerEscaped == kPlayerEscapedUpStairs, so:
            otherPlayerEscaped <- kNoOneEscaped                   Player.c:358
            MoveRoomToRoom(B, kAbove)                             Player.c:359
              -> TagGliderIdle(the non-firstPlayer)               Transit.c:290-296
              -> ReadyLevel(); WipeScreenOn(); RenderFrame()      Transit.c:298-303
```

The intermediate `kPlayerEscaping*` state exists precisely so that glider B touching the
hotspot while A is still walking (frames N..M) is treated as "A has claimed the exit" and B
is allowed to start walking too (the `else if` at `Interactions.c:1271` requires
`otherPlayerEscaped == kPlayerEscapedUpStairs`, so during `Escaping` B is simply ignored —
the `else if` does not match `-8`). **Verify that reading against your port**: during the
`Escaping` window, B's hotspot touch does nothing at all, because neither branch matches.

## 10.6.4 Complete state table

| Value | Constant | Set by | Cleared by | Meaning |
| --- | --- | --- | --- | --- |
| -1 | `kNoOneEscaped` | `Play.c:116` (`NewGame`); every second-arriver path; `FollowTheLeader` `Transit.c:480` | — | nobody waiting |
| -2 | `kPlayerEscapedRight` | `Interactions.c:611`, `:642` | second arriver `:617`, `:648`; `OffAMortal` `Player.c:1543-1600` | one glider in limbo, exited right |
| -3 | `kPlayerEscapedLeft` | `Interactions.c:521`, `:552` | `:527`, `:558`; `Player.c:1543-1600` | … left |
| -4 | `kPlayerEscapedUp` | `Interactions.c:181`, `:215` | `:187`, `:221`; `Player.c:1543-1600` | … up through the ceiling / dirt hole |
| -5 | `kPlayerEscapedDown` | `Interactions.c:293`, `:327` | `:299`, `:333`; `Player.c:1543-1600` | … down through the floor |
| -6 | `kPlayerEscapedUpStairs` | `Player.c:363` | `Player.c:358` | … finished walking up the stairs |
| -7 | `kPlayerEscapedDownStairs` | `Player.c:491` | `Player.c:486` | … finished walking down |
| -8 | `kPlayerEscapingUpStairs` | `Interactions.c:1266` | superseded by -6 at `Player.c:363` | still walking up |
| -9 | `kPlayerEscapingDownStairs` | `Interactions.c:1301` | superseded by -7 | still walking down |
| -10 | `kPlayerTransportedOut` | `Player.c:627` | `Player.c:622` | in limbo via a transporter; pair on `activeRectEscaped` |
| -11 | `kPlayerDuckedOut` | `Player.c:734` (down), `:831` (up) | `Player.c:729`, `:826` | via a duct |
| -12 | `kPlayerMailedOut` | `Player.c:1030` (left), `:1121` (right) | `Player.c:1025`, `:1116` | via a mailbox |
| -69 | `kPlayerIsDeadForever` | `Player.c:1602` (end of `OffAMortal`) | `NewGame` only | the other player is permanently out |

(`GliderPRO/Headers/GliderDefines.h:596-608`.)

`kPlayerIsDeadForever` also gates the suicide key: `ApplyInput` step 39 tests
`otherPlayerEscaped != kNoOneEscaped` (`GliderPRO/Sources/Input.c:366`), which is true for
`-69`, but the same condition also requires `!onePlayerLeft`, which is false once a player
is permanently out. So the delete key is live exactly while the *other* player is waiting in
limbo — which is the intent: "I am stuck, kill me so we can move on."

## 10.6.5 Under lock-step, the rendezvous needs **zero** messages

This is the section's punchline.

> `otherPlayerEscaped`, `activeRectEscaped`, `firstPlayer`, `onePlayerLeft`, `playerDead`,
> `playerSuicide` and `saidFollow` are **derived state**. Every one is written by
> deterministic simulation code from deterministic inputs. Under lock-step both peers
> compute identical values for all of them on the same frame. **No rendezvous message
> exists in this protocol.**

What §9.8 calls "already a two-phase protocol" is prior art for the *game rules*, not for
the network layer. The rendezvous is not a distributed agreement problem once both peers run
the same simulation; it is a local state machine that happens to look like one.

The corresponding obligation is that the rendezvous variables must be **in the checksum**
(§10.4.6 step 4) so that a divergence is caught within 8 frames, and in `MsgProgress`
(§10.4.8) so a resumed match reconstructs them. Both are specified.

The two-player test stream (§10.4.9's caveat) must cover, at minimum:

| Case | Why |
| --- | --- |
| both gliders exit the **same** boundary | the happy path, `Interactions.c:187-188` |
| second glider reaches a **different** boundary | the bounce, `Interactions.c:191-194` |
| both gliders take the **same** staircase | four-state path, `Player.c:356-359` |
| both gliders take **different** staircases in one room | no `activeRectEscaped` check for stairs (§10.6.1) |
| both gliders use the **same** transporter | `activeRectEscaped` match, `Interactions.c:1405` |
| second glider tries a **different** transporter | `activeRectEscaped` mismatch → nothing happens |
| glider B touches the stairs while A is still `Escaping` | neither branch matches (§10.6.3) |
| one glider dies while the other is in limbo | `OffAMortal`'s dispatch, `Player.c:1543-1600` |
| player 1 presses delete while player 2 waits in limbo | `ForceKillGlider` → `playerSuicide` → `FollowTheLeader` |
| both players die on the same frame | `mortals < -1` → `FlagGameOver`, `Player.c:1500` |
| a star is collected, ending the game, on the same frame as a room transition | `numStarsRemaining <= 0` → `FlagGameOver` inside phase 5 |

## 10.6.6 If you *must* build a non-lock-step topology

For completeness — a peer-authority design (each peer owns its glider's *position* and the
other peer applies it) still cannot avoid agreeing on the rendezvous, and would need these
messages. This is documented so the choice in §10.2.4 is an informed one, not so it is
implemented.

```
MsgRendezvous (msgType 0x30, 24 bytes)
  0..9  standard header (magic, version, msgType, matchID, senderSlot, pad)
 10     frame uint32                  the frame the transition is claimed for
 14     newState int16                one of the 13 kPlayer* / kNoOneEscaped values
 16     activeRect int16              the hotspot index, or -1
 18     firstPlayer uint8             which glider is now in limbo
 19     claim uint8                   0 = FIRST_ARRIVER, 1 = SECOND_ARRIVER_COMMIT
 20     roomAfter int16               thisRoomNumber the sender will be in
 22     crc16 uint16                  over bytes 0..21
```

and it would need, at minimum:

* a tie-break rule for both peers claiming FIRST_ARRIVER on the same frame (use
  `senderSlot`: player 1 wins, matching E2's existing bias),
* an idempotence rule so a retransmitted SECOND_ARRIVER_COMMIT does not transition twice,
* a rule for what happens when a peer's glider is bounced (`Interactions.c:191-194`) by a
  state the other peer has not yet applied — the bounce changes `vVel`, so it is a physics
  divergence, not just a bookkeeping one,
* and a resolution for E4: the two peers' `MoveRoomToRoom` calls must land on the same frame
  or the room contents (and therefore the RNG stream, §10.3.5) diverge.

That last bullet is unbounded work. It is the concrete reason §10.2.4 chooses lock-step.

## 10.6.7 Two traps in the death cascade

**Trap 1 — `FollowTheLeader` reads an uninitialised local.**
`GliderPRO/Sources/Transit.c:473-557`:

```c
void FollowTheLeader (void)
{
	short		wasEscaped;
	Boolean		oneOrTwo;                        // Transit.c:476  UNINITIALISED

	playerSuicide = false;                        // :478
	wasEscaped = otherPlayerEscaped;              // :479
	otherPlayerEscaped = kNoOneEscaped;           // :480

	if (theGlider.mode == kGliderInLimbo)         // :482
	{
		oneOrTwo = true;
		... copy theGlider's rects into theGlider2 ...
	}
	else if (theGlider2.mode == kGliderInLimbo)   // :490
	{
		oneOrTwo = false;
		... copy theGlider2's rects into theGlider ...
	}
	// NO else -- oneOrTwo can still be garbage here

	switch (wasEscaped)                           // :499
	{
		case kPlayerEscapedUp: ...
		if (oneOrTwo) MoveRoomToRoom(&theGlider2, kAbove);   // reads it
		else          MoveRoomToRoom(&theGlider,  kAbove);
		...
	}
}
```

`FollowTheLeader` is called from `OffAMortal` when `playerSuicide` is set
(`GliderPRO/Sources/Player.c:1529-1530`), and `playerSuicide` is set only by
`ForceKillGlider` (`playerSuicide = true` at `GliderPRO/Sources/Transit.c:457`, `:466`;
those are the only two writes of `true` in the tree, the only other write being
`playerSuicide = false` at `Play.c:118` and `Transit.c:478`), which itself only fires when
one of the two gliders **is** in `kGliderInLimbo`
(`GliderPRO/Sources/Transit.c:451`, `:460`). So in the shipped game one of the two branches
always runs and `oneOrTwo` is always assigned. It is reachable-in-principle UB, not
reachable-in-practice UB.

In Go, `oneOrTwo` is `false`, meaning "move `theGlider`". Since the branch is unreachable,
the choice is free — but **write it down and add a panic/assert**, because if a port ever
changes `ForceKillGlider`'s guards this becomes a silent wrong-glider-moves bug.

**Trap 2 — `OffAMortal`'s dispatch duplicates `FollowTheLeader`'s, with different
selectors.** Compare `GliderPRO/Sources/Player.c:1543-1600` (selector: `playerDead ==
kPlayer1`) with `GliderPRO/Sources/Transit.c:499-556` (selector: `oneOrTwo`). Same seven
cases, same seven transition calls, and both take the `theGlider2` arm when the selector is
true — but the selectors mean *different things about different gliders*: `playerDead ==
kPlayer1` is "glider 1 has just died, so glider 2 is the survivor to move", whereas
`oneOrTwo` is set at `Transit.c:484` when `theGlider.mode == kGliderInLimbo`, i.e. "glider 1
is the one waiting in limbo, so glider 2 is the freshly-killed one to move". One is reached
via `mortals == -1 && onePlayerLeft`, the other via `playerSuicide`. They
must not be factored together without proving the selectors agree. They also differ in one
respect: `OffAMortal` ends with `otherPlayerEscaped = kPlayerIsDeadForever`
(`GliderPRO/Sources/Player.c:1602`) while `FollowTheLeader` sets `kNoOneEscaped` up front
(`GliderPRO/Sources/Transit.c:480`).

Both dispatches map the same seven states to the same four transition functions:

| `wasEscaped` / `otherPlayerEscaped` | Transition |
| --- | --- |
| `kPlayerEscapedUp` (-4), `kPlayerEscapingUpStairs` (-8), `kPlayerEscapedUpStairs` (-6) | `MoveRoomToRoom(survivor, kAbove)` |
| `kPlayerEscapedDown` (-5), `kPlayerEscapingDownStairs` (-9), `kPlayerEscapedDownStairs` (-7) | `MoveRoomToRoom(survivor, kBelow)` |
| `kPlayerEscapedLeft` (-3) | `MoveRoomToRoom(survivor, kToLeft)` |
| `kPlayerEscapedRight` (-2) | `MoveRoomToRoom(survivor, kToRight)` |
| `kPlayerTransportedOut` (-10) | `TransportRoomToRoom(survivor)` |
| `kPlayerMailedOut` (-12) | `MoveMailToMail(survivor)` |
| `kPlayerDuckedOut` (-11) | `MoveDuctToDuct(survivor)` |
| anything else (incl. `-1`, `-69`) | `default: break` — no transition |

Note that both the `Escaping` and `Escaped` stair states map to the same move, so a player
who dies while their partner is still *walking* up the stairs still gets pulled upward.

---

# 10.7 Corrections and additions to Part 7's hazard table

Part 10's source reading produced four corrections and four additions to §7.2. They are
recorded here rather than by editing §7.2, so the audit trail is visible.

| # | §7.2 as written | Correction |
| --- | --- | --- |
| 6 | "`BackUpToSavedMap` returning -1 (24-slot cap **or GWorld allocation failure**) suppresses the `RandomInt` draw" — "**free memory affects the RNG stream**" | **`theErr` from `CreateOffScreenGWorld` is assigned and never tested** (`GliderPRO/Sources/DynamicMaps.c:82`). The only -1 path is `numSavedMaps >= kMaxSavedMaps` (`:75-76`). Free memory does **not** affect the stream; the 24-slot cap does. Severity unchanged (FIXABLE). §10.3.4. |
| 25 | "in an unlit room some object branches skip their registration calls, so turning a light off can change how much RNG a room load consumes" | **`isLit` gates only `Draw*` calls.** All RNG-consuming `Add*` calls and all seven `BackUpToSavedMap` calls in `ObjectDrawAll.c` are outside the `isLit` test (verified over the whole file). The real mechanism is `IsThisValid` returning `data.c.state` for the twelve prize types (`GliderPRO/Sources/Objects.c:116`), which removes collected prizes from the room-load walk entirely, plus the saved-map budget. §10.3.3. |
| 13 | "Render pass mutates simulation state … `Render.c:639-671`" | Understated. **Four of eleven phases blit**, and phases 3, 5 and 8 also contain simulation-relevant graphics calls (`RestoreFromSavedMap` with `doSparkle = true` actually *spawns*). §10.1.1, §10.1.7. |
| 3 | "never consume gameplay RNG from a rendering path" | Correct but incomplete as a fix. The actionable rule is **R3 in §10.3.5**: the *logical* neighbour set is always 9, independent of the `numNeighbors` preference, so `numNeighbors` becomes view-only. |

New hazards found while writing Part 10:

| # | Hazard | Severity | Evidence | Effect | Fix |
| --- | --- | --- | --- | --- | --- |
| 35 | A `kStar` object consumes **two** saved-map slots with the same `(where, who)` key | FIXABLE | `GliderPRO/Sources/ObjectDrawAll.c:434` and `GliderPRO/Sources/DynamicMaps.c:680` | tightens the 24-slot budget by 1 per star (up to 8 of 24 with `kMaxStars` = 4), which changes which `AddCandleFlame` calls fail; `RestoreFromSavedMap`'s `break` (`DynamicMaps.c:160`) means the second slot is never restored | §10.3.4 removes `savedMaps` from the simulation entirely |
| 36 | `AddPendulum` writes the global `clockFrame = 10` **before** the slot check | BENIGN but must be replicated | `GliderPRO/Sources/DynamicMaps.c:578` vs `:580` | a pendulum dropped for lack of a slot still resets every clock's phase in the room | replicate the ordering (§10.3.5 R6) |
| 37 | `LogDemoKey` has no bounds check against the buffer | BENIGN in the shipped build (recording is `#ifdef CREATEDEMODATA`-only) | `GliderPRO/Sources/Input.c:44-49`; buffer is 2000 records (`StructuresInit2.c:282`) | a recording longer than 2000 actions overruns the heap | any port's recorder must bound-check |
| 38 | `HandleSwitches` calls `AddSparkle` on an **uninitialised** `Rect`, and it is **reachable** | COSMETIC but real UB (the only reachable uninitialised-local read found in the tree) | `Rect newRect, bounds;` at `GliderPRO/Sources/Interactions.c:987`; `bounds` never assigned; `AddSparkle(&bounds);` at `:1050`, reached for nine prize types from the `switch` at `:1038` | a second, garbage-positioned sparkle on top of the correct one `RestoreFromSavedMap(..., true)` already spawned at `:1049`; also consumes a `sparkles[]` slot | drop the `:1050` call; §10.5.7 |

---

## Open questions

1. **What exactly does `Random()` return?** The trap's algorithm is not in the tree
   (hazard 1, `GliderPRO/Sources/Utilities.c:76`). §2.1.1's empirical properties were
   derived from the *mapping* in `RandomInt`, not from the generator. This is unanswerable
   from this source tree and is why §10.3.1 substitutes a specified PRNG. Consequence: a Go
   port cannot reproduce a 1994 session frame-for-frame, only its own sessions.

2. **Is the `activeRectEscaped` asymmetry for stairs intentional?** Transporters, mailboxes
   and ducts require the pair to use the *same* hotspot index
   (`GliderPRO/Sources/Interactions.c:1405`, `:1452`, `:1495`, `:1534`, `:1569`) but stairs
   do not (`kMoveItUp` `:1250-1283` and `kMoveItDown` `:1285-1318` have no such test). Two gliders can rendezvous via two different
   staircases in the same room. Nothing in the source explains why. A port must choose to
   preserve it (recommended) or unify it (a rules change).

3. **What happens if `activeRectEscaped` names a hotspot index that no longer exists after a
   room reload?** It is never reset to a sentinel. In the original the read is always
   guarded by `otherPlayerEscaped` naming a matching class, and a transition clears
   `otherPlayerEscaped`, so the stale value is unreachable. This was reasoned about, not
   tested. A port should assert it.

4. **Is one frame of animation the right amount for the transition `RenderFrame()`?**
   `GliderPRO/Sources/Transit.c:303` is inside `#ifdef COMPILEQT`, and `COMPILEQT` **is**
   defined unconditionally at `GliderPRO/Headers/GliderDefines.h:15` (§2.3), so it is live — but that means the animation advance is
   *incidentally* coupled to a QuickTime build flag. Was that intended, or is the extra
   advance an artefact of where the movie restart had to go? A port must pick one and
   document it (§10.1.8 recommends keeping it).

5. **Should `kInputDelay` be 2?** §10.2.4 argues from `kNormalThrust` being applied
   same-frame (`GliderPRO/Sources/Input.c:313`) and from the existing 30-frame
   `TagGliderIdle` grace period (`GliderPRO/Sources/Modes.c:638`). It has not been measured
   with human players. It is the one parameter in this document that wants playtesting
   rather than source reading.

6. **How large is a real house?** §10.2.2 and §10.4.8 size `MsgProgress` as
   102 + 290 × `nRooms` bytes using a hypothetical 100 rooms. The actual room counts of the
   shipped houses are in `docs/analysis/original-houses.md` and
   `docs/analysis/houses-inventory.md`; Part 10 did not re-derive them, and the packet
   sizing should be re-checked against the largest real house before fixing a buffer size.

7. **Does any shipped house actually depend on the RNG for a *reachable* outcome?** §8.7
   observes that the demo replays correctly despite a different RNG stream every time, and
   names `HandleCoffee`'s `RandomInt(200)` (`GliderPRO/Sources/Dynamics.c:492`, `:506`) and
   `HandleSparkleObject`'s `RandomInt(240)` (`:304`) as counterexamples in principle. Nobody
   has surveyed the shipped houses to find a room where an RNG-driven timer changes whether
   a room is passable. If none exists, §10.3's synchronised streams are belt-and-braces; if
   one exists, they are essential. Worth a survey.

8. **Is the player-1-wins tie-break visible in practice?** E2 (§10.5.1) means player 1 gets
   every simultaneously-touched prize. With pooled resources (§9.4) this should be
   invisible, but `theScore` is not doubled while supplies are, so a contested *clock*
   awards 100/300/500/1000 points once regardless of who touches it — meaning ties do have a
   measurable effect on nothing at all. Confirming that with a two-player test stream would
   retire the concern.

9. **What is the right behaviour when the two peers' houses have the same name but different
   bytes?** §10.4.7 makes `houseHash` mandatory and aborts. Should the port instead offer to
   transfer the house? That is a product decision with a protocol consequence (a
   `MsgHouseTransfer`), deliberately left out of version 1.

10. ~~**`numChimes` gates `HandleTelephone`'s chime arm, but the phone arm runs when
    `!phoneBitSet`**, which reads as inverted.~~ **RESOLVED — the polarity is correct.**
    `phoneBitSet` is not a room-level "there is a phone here" flag at all: it is a
    **house-wide** flag, bit 1 (`0x00000002`) of `houseType.flags`, loaded once per house at
    `GliderPRO/Sources/HouseIO.c:417` and initialised to `false` for brand-new houses only
    (`InitializeEmptyHouse`, `House.c:143`). It is
    edited in the House Info dialog by a checkbox whose item constant is **`kNoPhoneCheck`**
    (`GliderPRO/Sources/HouseInfo.c:120`, `:296-297`; written back to `flags` at `:266-273`).
    So `phoneBitSet == true` means *"this house has **no** phone"*, and `if (!phoneBitSet)`
    at `GliderPRO/Sources/Play.c:748` correctly reads "this house has a phone, so run the
    ring countdown". There is no bug and no inversion. Consequence for §10.3.2: `StreamPhone`
    consumes A1/A2 (`Play.c:735`, `:736`) unconditionally at `InitTelephone`, but A4/A5
    (`:759`, `:760`) only in houses **without** the no-phone flag — so the flag must be part
    of the `houseHash` (§10.4.7), which it already is, being inside the house bytes.
    (Aside, editor-only: the clear path at `HouseInfo.c:272` masks with `0xFFFFDFFD`, i.e.
    `~0x00002002`, so unchecking the box also clears bit 13. Irrelevant to play; do not
    replicate.)

---

## Porting notes

Ordered by the cost of getting them wrong.

1. **Do not adopt "each machine simulates its own player."** It is the natural reading of
   the two-player mode and it is wrong here. `HandleInteraction` resolves both gliders
   against shared mutable room state within one call
   (`GliderPRO/Sources/Interactions.c:1691-1711`), `SetObjectState` has a first-caller-wins
   rule (`GliderPRO/Sources/Objects.c:460-462`), and a room transition can happen *between*
   the two `CheckGliderInRoom` calls (`Interactions.c:1705-1706`). Every peer simulates every player;
   only input crosses the wire. §10.5.2.

2. **The sim/render cut is not at `RenderFrame`.** Nine of `RenderFrame`'s seventeen steps
   write simulation state, and phases 3, 5 and 8 also blit. Split `RenderFrame` into
   `SimStepPart2` + `Draw`, preserving the exact order of
   `GliderPRO/Sources/Render.c:641-660`, and prove the split with the demo (§10.1.10). If
   you skip that proof you will ship a peer that desyncs the moment someone tabs away.

3. **`HandleGrease` is the trap inside the trap.** It runs from `RenderFrame`
   (`GliderPRO/Sources/Render.c:647`) and creates collision geometry:
   `hotSpots[...].action = kSlideIt`, `.isOn = true`, `.bounds` growing 2 px/frame
   (`GliderPRO/Sources/Grease.c:60-68`, `:94-102`). A peer that does not render does not
   grow the grease. §10.1.4.

4. **`RandomInt`'s upper bound is inclusive and its distribution is skewed.** Reimplement
   `abs(raw) * range / 32768` exactly (`GliderPRO/Sources/Utilities.c:72-82`, the
   `Random()` draw at `:76` and the scaling at `:79`). `RandomInt(6)`
   can return 6; `RandomInt(2)` returns 2 for exactly one of 65536 inputs. `rand.Intn` is
   not a substitute. §10.3.1.

5. **Draw the RNG before every guard.** Every `Add*` must consume its draw before the cap
   check, the geometry check, the `IsThisValid` check and the snapshot attempt. Otherwise
   prize-collection history, screen size, colour depth and the 24-slot budget all move the
   stream. §10.3.3, §10.3.5 R2.

6. **`numNeighbors` must not reach the simulation.** It is a user preference
   (`GliderPRO/Sources/Main.c:111`, `:154`) forced to 1 on narrow screens
   (`:191-192`) and it currently changes how many rooms `DrawLocale` walks
   (`GliderPRO/Sources/RoomGraphics.c:78`, `:105`) and therefore how much RNG a room load
   consumes. Always walk all nine logically; render however many you like. §10.3.5 R3.

7. **Hash both `thisRoom` and the house handle.** `thisRoom` is a by-value 348-byte copy
   (`GliderPRO/Sources/Room.c:379`, hazard 23) and `SetObjectState` /
   `HandleRoomVisitation` write both. A checksum over only one will agree with itself while
   being wrong. §10.4.6 step 25.

8. **`long` is 32 bits and structs are 2-byte packed.** `#pragma options align=mac68k`
   (`GliderPRO/Headers/Externs.h:231`, reset at `:269`). Verified sizes: `demoType` 6, `hotObject` 16,
   `bandType` 16, `greaseType` 26, `dynaType` 36, `gliderType` 110, `gameType` 40,
   `scoresType` 292, `savedRoom` 292, `houseType` header 866. Keep the 2-byte packing but
   widen `long` to 64 bits and the five structs that *contain* a `long` all change size —
   `demoType` 6 → **10**, `gameType` 40 → **56**, `scoresType` 292 → **372**, `houseType`
   header 866 → **970**, `gliderType` 110 → **126** (four longs: `leftKey`, `rightKey`,
   `battKey`, `bandKey`, `GliderStructs.h:205-206`). A 64-bit *pointer* additionally takes
   `savedType` 16 → **20** (`GWorldPtr map`, `GliderStructs.h:230`). `hotObject`, `bandType`, `greaseType`,
   `dynaType`, `objectType` and `savedRoom` contain neither and are unaffected. (All eleven
   figures re-derived with gcc under `#pragma pack(2)`.) §4.3.3 — all wrong, all silently.

9. **All velocity damping is C integer division.** `hVel /= 2`, `hVel /= 4` truncate toward
   zero: `-7 / 4 == -1`. Go matches. A shift does not (`-7 >> 2 == -2`). §10.5.3.

10. **Preserve the player-1-first ordering everywhere.** Input (`Play.c:452-453`), hotspots
    (`Interactions.c:1639` before `:1657`), room-boundary checks (`:1705-1706`), glider updates
    (`Play.c:460-461`). And remember `kPlayer1 == TRUE` and `kFaceRight == TRUE`
    (`GliderPRO/Headers/GliderDefines.h:554-556`) — a Go port using `Which int` with 0 = P1
    inverts every `thisGlider->which` test in the tree.

11. **Bit 4 of player 2's input word is inert.** `ForceKillGlider` is guarded on
    `(thisGlider->which)` (`GliderPRO/Sources/Input.c:368`), so only player 1 can trigger
    the shared suicide. Replicate it; do not "fix" it silently. §10.4.2.

12. **Pause and focus loss must be simulation inputs, not control flow.** `DoPause` is three
    *sequential* (not nested) spin loops inside phase 4
    (`GliderPRO/Sources/Input.c:77-117`: `:89-94` waits for the pause key to be released,
    `:97-105` is the pause loop proper, `:111-116` waits for the release again) and
    `switchedOut` halts the whole loop (`GliderPRO/Sources/Play.c:437-443`). Both are
    per-peer and both stop the clock. §10.2.7.

13. **Do not send state during play.** The per-frame protocol is 18 header bytes plus one
    byte per player per frame (§10.4.4). Under 1 KB/s each way with 8 frames of redundancy.
    Any temptation to "just send the position to be safe" reintroduces every problem
    lock-step exists to avoid.

14. **Never send uninitialised bytes.** `demoType.padding` is heap garbage — 109 distinct
    values observed in the shipped resource (§10.4.1) — which is why the shipped demo is not
    byte-reproducible even from identical gameplay (§8.5 finding 7).

15. **Zero-initialisation changes five latent C bugs, and one of them is reachable.**
    `oneOrTwo` in `FollowTheLeader` (`GliderPRO/Sources/Transit.c:476`, §10.6.7), `changed`
    in `SetObjectState` (`GliderPRO/Sources/Objects.c:369`, §10.5.6), the local in
    `DidBandHitDynamic` (hazard 17), and `masterObjects[-1]` in `FireTrigger` (hazard 18,
    which *panics* in Go) are all unreachable-in-practice — assert on them rather than
    relying on that. The fifth is **not** unreachable: `bounds` in `HandleSwitches`
    (`GliderPRO/Sources/Interactions.c:987`, read by `AddSparkle(&bounds)` at `:1050`) is
    hit by every switch wired to a prize. Drop that call; see §10.5.7 for why fixing it is
    wrong too.

16. **Replicate the transition-time extra animation step.** `MoveRoomToRoom` and its three
    siblings call `RenderFrame()` after `WipeScreenOn` (`GliderPRO/Sources/Transit.c:303`,
    `:341`, `:380`, `:419`), which under the split becomes an explicit extra
    `SimStepPart2()`. Do not gate it on whether rendering is enabled. §10.1.8.

17. **The demo is a free CI test.** 1117 records, 3369 frames, frames 46–3414, keys
    {0: 910, 1: 198, 3: 9}, strictly increasing with no duplicates. Transcode it to input
    words (§10.4.9) and run T1/T2/T3 on every commit. It will catch the sim/render cut, the
    RNG contract and the transport layer. It will **not** catch the two-player rendezvous —
    hand-author a second stream covering the eleven cases in §10.6.5.

18. **Disconnect is an existing game state, not a new one.** Synthesise
    `otherPlayerEscaped = kPlayerIsDeadForever` (-69), `onePlayerLeft = true`,
    `playerDead = <departed>`, `FlagGliderInLimbo(departed, false)`, `dontDraw = true` —
    exactly what `OffAMortal` does at `GliderPRO/Sources/Player.c:1505-1508`, `:1602` — and
    the forty-odd existing guard clauses (§9.5) do the rest, including switching off the
    supply doubling. §10.2.6.

19. **Keep `savedMaps` out of the simulation, and accept the cosmetic divergence.**
    §10.3.4's fix means candle-flame sprite phases in cluttered rooms will differ from the
    original, because the original suppressed the RNG draw when the 24-slot budget ran out.
    That is the price of determinism and it is worth paying. Write it down so nobody
    "fixes" it later by restoring the guard.

20. **Version the protocol from day one, and put the PRNG choice, the hash algorithm, the
    stream label list and the room-load draw schedule *inside* the version.** All four are
    protocol, not implementation. Changing any of them silently is a desync that will look
    like a physics bug.
