# Glider PRO 1.0.4 — Player (Glider) State Machine and Physics

## Scope

This document is a complete, implementation-level specification of the **player
entity** in Glider PRO 1.0.4 — the paper glider the user flies. It covers:

* the `gliderType` record (every field, type, byte offset, and observed `sizeof`),
* the position/velocity representation and its exact units,
* the integrator (`MoveGlider`) including gravity, impulse ramping, clamping, and
  the swept "dirty rectangle" bookkeeping that is fused into the same function,
* every one of the **24 glider modes** and the handler that runs for each,
* every mode-entry ("`Flag…`" / "`Start…`") function and the exact field values it
  installs,
* sprite/frame selection for all 31 source rectangles, including the fade
  sequence table and the foil (aluminium foil) atlas swap,
* hit boxes: the four distinct box computations (`SectGlider`, `GliderInRect`,
  `GliderHitTop`, and the raw `dest` tests in `CheckGliderInRoom`),
* facing, tipping (banking), about-face tumble, sliding (grease), webbing,
  burning, shredding, mail/duct/transporter travel, stairs,
* input: key maps, thrust, battery (hyper-thrust), helium, rubber bands, and the
  recoil they apply to the player,
* death, the 68-frame shredder death delay, respawn position (`enteredRect`),
  and the absence of invincibility frames,
* the authoritative per-frame update order and the render order,
* every literal physics constant with its numeric value and every place it is
  applied.

It deliberately **does not** specify: room/house file format beyond the fields
that determine where the player spawns, object behaviour beyond the effect the
object has on the player, sound synthesis, the level editor, or the scoreboard.

All line numbers refer to the original sources with **CR line endings converted
to LF** (`tr '\r' '\n'`). Paths are relative to the repository root.

## Sources read

Read in full:

| File | Lines | Why |
| --- | --- | --- |
| `GliderPRO/Sources/Player.c` | 1605 | The integrator and all 24 mode handlers, `OffsetGlider`, `OffAMortal` |
| `GliderPRO/Sources/Modes.c` | 640 | Every mode-entry function |
| `GliderPRO/Sources/Interactions.c` | 1777 | Hit boxes, wall/room escape, hot-spot effects, rewards, webbing |
| `GliderPRO/Sources/Input.c` | 398 | Key polling, thrust, battery, helium, band firing |
| `GliderPRO/Sources/Transit.c` | 558 | Room-to-room, transporter, mail, duct transit and respawn routing |
| `GliderPRO/Sources/Grease.c` | 302 | The `kSlideIt` hot spot that makes the player slide |
| `GliderPRO/Headers/GliderDefines.h` | 625 | Every constant and enum |
| `GliderPRO/Headers/GliderStructs.h` | 347 | Every struct |

Read in relevant part:

| File | Section | Why |
| --- | --- | --- |
| `GliderPRO/Sources/Play.c` | 19, 74–278, 305–366, 387–599, 712–729 | `NewGame`, `InitGlider`, `PlayGame` frame order, `HideGlider` |
| `GliderPRO/Sources/Render.c` | 136–189, 452–530, 559–612, 639–671 | `DrawReflection`, `RenderGlider`, `RenderShreds`, `RenderFrame` |
| `GliderPRO/Sources/Room.c` | 816–933, 1103–1205 | `DetermineRoomOpenings`, `IsShadowVisible`, floor/ceiling tests |
| `GliderPRO/Sources/RoomGraphics.c` | 44–130, 400–418 | `DrawLocale` (clears `takingTheStairs`), `ReadyLevel` |
| `GliderPRO/Sources/RectUtils.c` | 53–216 | `ZeroRectCorner`, `RectWide/Tall`, `CenterRectInRect`, `IsRectLeftOfRect`, `QOffsetRect`, `QSetRect` |
| `GliderPRO/Sources/StructuresInit.c` | 19, 27, 169–238 | The glider sprite atlas and the `gliderSrc[31]` table |
| `GliderPRO/Sources/StructuresInit2.c` | 255–257, 383–392, 430 | `shreds` allocation; object source rects used by stairs/mail/duct/shredder geometry |
| `GliderPRO/Sources/InterfaceInit.c` | 147–219 | Player-2 key map, `fadeInSequence[16]`, `playOriginH/V` |
| `GliderPRO/Sources/Main.c` | 69–72, 135–138, 225–228 | Player-1 default key map and its persistence |
| `GliderPRO/Sources/House.c` | 218–240 | `WhereDoesGliderBegin` |
| `GliderPRO/Sources/Environ.c` | 654 | Memory budget line that pins `shreds` to `kMaxShredded` elements |
| `GliderPRO/Sources/ObjectRects.c` | 680–806, 930–950, 1133–1187 | Hot-spot rect geometry for slide/stairs/mail/duct/shredder, `GetUpStairsRightEdge`, `GetDownStairsLeftEdge` |
| `GliderPRO/Sources/Dynamics.c` | 17, 34–74 | `CheckDynamicCollision` (foil shove) |
| `GliderPRO/Sources/RubberBands.c` | 12–14, 140–290 | Band-vs-player impulse, `AddBand` recoil |
| `GliderPRO/Sources/DynamicMaps.c` | 29, 726–795 | `shreds` declaration, `AddAShreddedGlider`, `RemoveShreds`, `ZeroShreds` |
| `GliderPRO/Sources/GameOver.c` | 17, 45, 237–240 | `FlagGameOver`, `countDown` |
| `GliderPRO/Sources/SavedGames.c` | 101–106, 261–275, 330–335 | Which player fields persist across a save |
| `GliderPRO/Headers/Externs.h` | 129–163 | Raw Mac key-map bit offsets |
| `GliderPRO/Headers/GliderVars.h` | 56–58 | `batteryTotal`, `bandsTotal`, `foilTotal`, `mortals`, `showFoil` |
| `GliderPRO/Headers/Player.h` | 1–13 | (contains only two `GWorldPtr` externs) |
| `GliderPRO/Glider PRO.r` | resource index | PICT frame sizes for the glider atlases (verified with python3) |
| `GliderPRO/Houses/*.binhex` | data forks | `houseType.initial` and `roomType.leftStart/rightStart` (verified with python3) |

---

## 1. Model summary

The glider is a **48 x 20 pixel axis-aligned rectangle** that moves in **whole
pixels per frame** at a nominal **30 frames per second**.

Four facts dominate any port:

1. **Everything is integer.** There is no fixed point anywhere in the player
   code. `hVel`, `vVel`, `hDesiredVel`, `vDesiredVel`, `wasHVel`, `wasVVel` are
   all `short` (signed 16-bit) and are added directly to rectangle edges. There
   is no `<<4`, no sub-pixel accumulator, no `Fixed`. The only shifts applied to
   player state are `>> 6` (divide by the 64-pixel tile width, in
   `GliderPRO/Sources/Interactions.c:200`, `:454`) and `>> 3` (an averaging
   divide-by-8 in `WebGlider`, `GliderPRO/Sources/Interactions.c:1750`).

2. **There is no separate x/y.** Position lives entirely in QuickDraw
   rectangles. Movement mutates `dest.left/right/top/bottom` (and the shadow's
   `destShadow`) in place. `whole` and `wholeShadow` are *swept* rectangles that
   union the pre-move and post-move extents so the dirty-rect renderer can
   restore the background. Movement and dirty-rect bookkeeping are fused into
   the same statements (`GliderPRO/Sources/Player.c:101`–`:145`).

3. **Desired velocities are reset every frame by the integrator.**
   `MoveGlider` sets `hDesiredVel = 0` and `vDesiredVel = kGravity` at
   `GliderPRO/Sources/Player.c:78` and `:92`. So every accelerating influence
   (keys, fans, vents, helium) must be re-applied on *every* frame; nothing
   persists. Current velocity `hVel`/`vVel` *does* persist and is ramped toward
   the desired value by at most 2 px/frame.

4. **Damping happens only through the desired-velocity ramp.** There is no
   friction term, no drag coefficient. Horizontal "friction" is simply the
   `±kHImpulse` ramp pulling `hVel` back toward the per-frame default of 0.
   Vertical "gravity" is the same ramp pulling `vVel` toward `+3`.

Terminal velocity is asymmetric: horizontal speed is clamped to ±16, **vertical
speed is not clamped at all** — the vertical terminal speed is an emergent
consequence of `vDesiredVel` defaulting to `+3` (so free fall settles at exactly
+3 px/frame, reached from 0 in two frames).

---

## 2. The `gliderType` record

Declared at `GliderPRO/Headers/GliderStructs.h:200`–`:216`. Verbatim:

```c
typedef struct {
	Rect		src, mask, dest, whole;
	Rect		destShadow, wholeShadow;
	Rect		clip, enteredRect;
	long		leftKey, rightKey;
	long		battKey, bandKey;
	short		hVel, vVel;
	short		wasHVel, wasVVel;
	short		vDesiredVel, hDesiredVel;
	short		mode, frame, wasMode;
	Boolean		facing, tipped;
	Boolean		sliding, ignoreLeft, ignoreRight;
	Boolean		fireHeld, which;
	Boolean		heldLeft, heldRight;
	Boolean		dontDraw, ignoreGround;
} gliderType, *gliderPtr;
```

`Rect` is the QuickDraw rectangle: four `SInt16` in the order
`top, left, bottom, right` (`GliderPRO/Sources/RectUtils.c:210`–`:216` shows
member-by-member assignment, so the order matters only for on-disk data, not for
this in-memory struct).

### 2.1 Verified layout

Compiled with gcc on this machine using the exact field order above, once with
natural alignment (PowerPC-like) and once with `#pragma pack(2)` (68k-like).
**Observed output:**

```
sizeof(Rect)=8 sizeof(gliderType)=112       <- natural alignment
sizeof(Rect)=8 sizeof(gliderType)=110       <- #pragma pack(2)
```

Both give **identical member offsets**; only the trailing pad differs (112 is
110 rounded up to a multiple of 4). Observed offsets:

| Offset | Size | Field | Type | Meaning |
| --- | --- | --- | --- | --- |
| 0 | 8 | `src` | Rect | source rect inside the 48x668 glider atlas (which sprite to blit) |
| 8 | 8 | `mask` | Rect | source rect inside the 1-bit mask atlas; **always assigned the same value as `src`** everywhere in the code base |
| 16 | 8 | `dest` | Rect | the glider's position, in room-local pixels |
| 24 | 8 | `whole` | Rect | swept union of pre-move and post-move `dest`, for dirty-rect restore |
| 32 | 8 | `destShadow` | Rect | the ground shadow's position (48 x 9, top fixed at 306 unless moved) |
| 40 | 8 | `wholeShadow` | Rect | swept union for the shadow |
| 48 | 8 | `clip` | Rect | mode-specific clipping/target rect (mail slot, duct, etc.) |
| 56 | 8 | `enteredRect` | Rect | **the respawn position**: where `dest` was when the player last became "normal" in this room |
| 64 | 4 | `leftKey` | long | raw key-map bit index for "turn/thrust left" |
| 68 | 4 | `rightKey` | long | raw key-map bit index for "turn/thrust right" |
| 72 | 4 | `battKey` | long | raw key-map bit index for battery / helium |
| 76 | 4 | `bandKey` | long | raw key-map bit index for firing a rubber band |
| 80 | 2 | `hVel` | short | current horizontal velocity, px/frame; **overloaded as an idle countdown** in `kGliderIdle` mode |
| 82 | 2 | `vVel` | short | current vertical velocity, px/frame (positive = down) |
| 84 | 2 | `wasHVel` | short | `hVel` as of the last completed move; used to "un-sweep" the hit box in `GliderHitTop` |
| 86 | 2 | `wasVVel` | short | `vVel` as of the last completed move; **written but never read anywhere** |
| 88 | 2 | `vDesiredVel` | short | target vertical velocity for this frame; reset to `kGravity` (3) each frame |
| 90 | 2 | `hDesiredVel` | short | target horizontal velocity for this frame; reset to 0 each frame |
| 92 | 2 | `mode` | short | the state-machine state, 0..23 (see §5) |
| 94 | 2 | `frame` | short | mode-local counter; **meaning differs per mode** (see §5.2) |
| 96 | 2 | `wasMode` | short | **triple-purpose**: saved mode for limbo/idle, burn countdown, web countdown (see §5.3) |
| 98 | 1 | `facing` | Boolean | `kFaceRight` = TRUE (1), `kFaceLeft` = FALSE (0) |
| 99 | 1 | `tipped` | Boolean | banking: player is holding the key *opposite* to `facing` |
| 100 | 1 | `sliding` | Boolean | set for one frame when standing on grease/slider |
| 101 | 1 | `ignoreLeft` | Boolean | one-frame flag: a door/window lets the player leave through the left wall |
| 102 | 1 | `ignoreRight` | Boolean | one-frame flag: same for the right wall |
| 103 | 1 | `fireHeld` | Boolean | band-key debounce (edge trigger) |
| 104 | 1 | `which` | Boolean | `kPlayer1` = TRUE (1) or `kPlayer2` = FALSE (0) |
| 105 | 1 | `heldLeft` | Boolean | left key was down this frame (blocks down-stairs) |
| 106 | 1 | `heldRight` | Boolean | right key was down this frame (blocks up-stairs) |
| 107 | 1 | `dontDraw` | Boolean | suppress rendering (in limbo, in transit, dead) |
| 108 | 1 | `ignoreGround` | Boolean | one-frame flag: a manhole/hole in the floor lets the player fall through |

There are exactly **two** instances: `gliderType theGlider, theGlider2;`
(`GliderPRO/Sources/Player.c:43`). `theGlider` is player 1; `theGlider2` is
player 2 and is only active in a two-player game (it is also used as a scratch
glider in one-player mode only insofar as `glid2SrcMap` is reused for the foil
atlas — see §18.3).

### 2.2 Aliasing note for a Go port

`src` and `mask` are always set to the same value. Every assignment in
`Player.c` and `Modes.c` is of the form

```c
thisGlider->src  = gliderSrc[n];
thisGlider->mask = gliderSrc[n];
```

and every clip operation touches both in lockstep (e.g.
`GliderPRO/Sources/Player.c:341`–`:343`). They are separate fields only because
`CopyMask` takes independent source and mask rectangles. A Go port may collapse
them into one rect **provided** it reproduces every place where only one edge of
each is trimmed (`Player.c:342`, `:375`, `:411`, `:469`, `:503`, `:540`, `:746`,
`:843`, `:873`, `:911`, `:944`, `:1042`, `:1133`, `:1308`).

---

## 3. Coordinate systems and units

### 3.1 Room-local coordinates

The player's `dest` is in **room-local** pixels. A room is
`kRoomWide` = **512** wide (`kNumTiles` 8 x `kTileWide` 64) and `kTileHigh` =
**322** tall (`GliderPRO/Headers/GliderDefines.h:496`–`:501`). Local x runs
0..511 left to right; local y runs 0..321 top to bottom. Values outside that
range are legal and meaningful — they are how the player leaves a room.

### 3.2 Screen coordinates

Rendering adds a global origin: `playOriginH = (screenWidth - kRoomWide) / 2`
and `playOriginV = (screenHeight - kTileHigh) / 2`
(`GliderPRO/Sources/InterfaceInit.c:203`–`:204`). Every blit does
`QOffsetRect(&dest, playOriginH, playOriginV)`
(`GliderPRO/Sources/Render.c:503`). The mirror reflection uses
`playOriginH - 20, playOriginV - 16` (`GliderPRO/Sources/Render.c:151`).

### 3.3 Room boundaries and thresholds

| Constant | Value | Where defined | Meaning |
| --- | --- | --- | --- |
| `kRoomWide` | 512 | `GliderDefines.h:499` | room width |
| `kTileHigh` | 322 | `GliderDefines.h:498` | room height |
| `kTileWide` | 64 | `GliderDefines.h:497` | tile width; `>> 6` converts x to tile index |
| `kNumTiles` | 8 | `GliderDefines.h:496` | tiles per room |
| `kVertLocalOffset` | 322 | `GliderDefines.h:501` | vertical spacing between stacked rooms on screen |
| `kCeilingLimit` | 8 | `GliderDefines.h:503` | `dest.top` below this triggers the ceiling test |
| `kFloorLimit` | 312 | `GliderDefines.h:504` | `dest.bottom` above this triggers the floor test |
| `kRoofLimit` | 122 | `GliderDefines.h:505` | on `kRoof` backgrounds, `dest.bottom` above this triggers roof collision |
| `kLeftWallLimit` | 12 | `GliderDefines.h:506` | left wall plane for walled rooms |
| `kNoLeftWallLimit` | -24 | `GliderDefines.h:507` | `0 - kGliderWide/2`: x at which the player has fully left through the left side |
| `kRightWallLimit` | 500 | `GliderDefines.h:508` | right wall plane for walled rooms |
| `kNoRightWallLimit` | 536 | `GliderDefines.h:509` | `kRoomWide + kGliderWide/2` |
| `kNoCeilingLimit` | -10 | `GliderDefines.h:510` | `dest.top` at which the player has fully left upward |
| `kNoFloorLimit` | 332 | `GliderDefines.h:511` | `dest.bottom` at which the player has fully left downward |
| `kShadowTop` | 306 | `GliderDefines.h:553` | fixed y of `destShadow.top` |
| `kGliderStartsDown` | 32 | `GliderDefines.h:569` | base y for a side-entry respawn rect |

`leftThresh` and `rightThresh` are per-room globals set by
`DetermineRoomOpenings` (`GliderPRO/Sources/Room.c:816`–`:933`). They take the
value `kLeftWallLimit`/`kRightWallLimit` in a walled room, or
`kNoLeftWallLimit`/`kNoRightWallLimit` in an open room. **The player code tests
`leftThresh == kLeftWallLimit` to decide "is there a wall here"**
(`GliderPRO/Sources/Interactions.c:513`, `:576`, `:603`, `:666`) — an
open room is one whose threshold is *not* the wall value.

### 3.4 Glider geometry

| Constant | Value | Where | Note |
| --- | --- | --- | --- |
| `kGliderWide` | 48 | `GliderDefines.h:548` | *nominal* `dest` width, installed at mode entry; reduced by the reveal/hide clipping in the stairs, mail and duct handlers (§8.3–§8.17) |
| `kGliderHigh` | 20 | `GliderDefines.h:549` | *nominal* `dest` height (26 while burning); reduced by the up-stairs, duct and shredder clipping (§8.3–§8.20) |
| `kHalfGliderWide` | 24 | `GliderDefines.h:550` | used for the roof tile test and band spawn |
| `kGliderBurningHigh` | 26 | `GliderDefines.h:551` | `dest` height while burning (flame plume above) |
| `kShadowHigh` | 9 | `GliderDefines.h:552` | `destShadow` height |

The burning glider is 26 tall, anchored at the bottom:
`dest.top = dest.bottom - kGliderBurningHigh`
(`GliderPRO/Sources/Modes.c:413`). This is why `SectGlider` adds 6 to
`glideBounds.top` for burning gliders (`GliderPRO/Sources/Interactions.c:107`–
`:108`) — 26 − 20 = 6, restoring the *body* box and excluding the flame.

The word *nominal* in the first two rows is load-bearing. 48 × 20 is the size the
mode-entry functions install (`FlagGliderNormal` at `Modes.c:336`–`:337`,
`StartGliderTransporting` at `:311`–`:312`) and the threshold the reveal/hide
tests compare against; it is not an invariant of `dest`. Twelve handlers
deliberately shrink `dest` frame by frame, so that a glider entering a stairwell,
a duct or a mail slot appears to slide behind the wall instead of popping in at
full size: up-stairs (`GliderPRO/Sources/Player.c:341`, height 0, and `:374`,
height 1..19), `FinishGliderUpStairs` (`:410`, width 4, 8, … 44 — successive
multiples of |`kHClimbStairsSpeed`| = 4, starting at 4 because
`ReadyGliderForTripUpStairs` (`Modes.c:519`) parks `dest.left` exactly on
`rightClip` (`:539`) and `FinishGliderUpStairs` offsets by −4 before computing
`hNotClipped`), down-stairs (`Player.c:468`, width 0, and `:502`, plus `:539` in
`FinishGliderDownStairs`, which trims `dest.left` rather than `dest.right`), the
three duct handlers (`:745`, `:842`, `:943`), mail in and out (`:872` and
`:1041`; `:910` and `:1132` trim `dest.left`) and the shredder (`:1307`). The
clipped edge is recomputed from the *unclipped* opposite edge every frame, so the
clip never accumulates -- but `dest` stays sub-nominal for the whole sequence,
and only the `FlagGliderNormal`/`FlagGliderBurning` call on the mode-exit path
restores it.

Three details are lost by treating 48 × 20 as an invariant and deriving
`dest.right = dest.left + 48` or `dest.bottom = dest.top + 20` each frame. First,
`src` and `mask` are clipped in lockstep with `dest` on every one of those lines
(`:411`–`:412`, `:375`–`:376`, and so on for the rest), so a port that clips only
`dest` scales the sprite instead of cropping it. Second, `destShadow` is slaved to
the clipped edge — `destShadow.right = dest.right` at `:413`, `:471`, `:505`,
`:875` and `:1044`, `destShadow.left = dest.left` at `:542`, `:913` and `:1135` —
so the shadow is not 48 wide in these modes either; `kShadowHigh` is safe only
because nothing ever clips `destShadow.top` or `.bottom`. Third, `whole` and
`wholeShadow` are written from the *unclipped* pre-move and post-move edges
*before* the clipping runs (`:331` `whole.bottom = dest.bottom`, then `:334`
`whole.top = dest.top`, and only then does `:341`/`:374` move `dest.top`, leaving
`whole.top` alone), so the swept dirty rect stays full size while `dest` does not.
That last one matters to any port that keeps `whole`, and specifically to
`StartGliderFadingOut`'s `RectTall(&dest) > kGliderHigh` test (`Modes.c:87`) and
`FlagGliderShredding`'s `dest.left > whole.left` test (`Modes.c:372`), both of
which read these rects as state. The authoritative per-mode rect assignments are
in §8.3–§8.20.

---

## 4. Master constant table

Every literal that affects player motion, in one place. "Applied at" gives the
statement that consumes it.

### 4.1 Core integrator (`GliderPRO/Sources/Player.c:13`–`:17`)

| Constant | Value | Hex | Applied at | Effect |
| --- | --- | --- | --- | --- |
| `kGravity` | 3 | 0x0003 | `Player.c:92` | `vDesiredVel = kGravity` every frame — the free-fall terminal speed |
| `kHImpulse` | 2 | 0x0002 | `Player.c:68`, `:74` | max change in `hVel` per frame |
| `kVImpulse` | 2 | 0x0002 | `Player.c:82`, `:88` | max change in `vVel` per frame |
| `kMaxHVel` | 16 | 0x0010 | `Player.c:96`–`:97`, `:113`–`:114` | horizontal speed clamp (±16) |
| `kShredderCountdown` | -68 | 0xFFBC | `Player.c:17` (used at `:1303`) | frames of "dead in the shredder" delay before `OffAMortal` |

### 4.2 Input (`GliderPRO/Sources/Input.c:14`–`:16`)

| Constant | Value | Applied at | Effect |
| --- | --- | --- | --- |
| `kNormalThrust` | 5 | `Input.c:293`, `:295`, `:313`, `:323` | `hDesiredVel ±= 5` from a held direction key, or forced while burning |
| `kHyperThrust` | 8 | `Input.c:126`, `:128`, `:133`, `:135` | battery adds ±8 **directly to `hVel`**, bypassing the ramp |
| `kHeliumLift` | 4 | `Input.c:162` | `vDesiredVel = -4` while helium is engaged |

### 4.3 Blowers, fans and shoves

| Constant | Value | Where defined | Applied at | Effect |
| --- | --- | --- | --- | --- |
| `kFloorVentLift` | -6 | `Interactions.c:13` | `Interactions.c:1203`, `:1362` | `vDesiredVel = -6` on `kLiftIt`; also the "foil saves you from fire" bounce |
| `kCeilingVentDrop` | 8 | `Interactions.c:14` | `Interactions.c:1207` | `vDesiredVel = 8` on `kDropIt` |
| `kFanStrength` | 12 | `Interactions.c:15` | `Interactions.c:1211`, `:1215` | `hDesiredVel += ∓12` on `kPushItLeft` / `kPushItRight` (note `+=`, so fans stack with keys) |
| `kShoveVelocity` | 8 | `Dynamics.c:17` | `Dynamics.c:54`, `:56` | `hDesiredVel = ±8` when a foil-clad player hits a dynamic object |

### 4.4 Transit speeds (all in `GliderPRO/Sources/Player.c`)

| Constant | Value | Line | Used by |
| --- | --- | --- | --- |
| `kClimbStairsSpeed` | -4 | `:317` | `MoveGliderUpStairs` vertical |
| `kVClimbStairsSpeed` | -4 | `:385` | `FinishGliderUpStairs` vertical |
| `kHClimbStairsSpeed` | -4 | `:386` | `FinishGliderUpStairs` horizontal, and seeded into `hVel` on completion |
| `kVDropStairsSpeed` | 4 | `:433`, `:514`, `:929` | `MoveGliderDownStairs`, `FinishGliderDownStairs` vertical (also redefined, unused, in `FinishGliderDuctingIn`) |
| `kHDropStairsSpeed` | 4 | `:434`, `:515` | down-stairs horizontal, and seeded into `hVel` on completion |
| `kVDropDuctSpeed` | 4 | `:659` | `MoveGliderDownDuct`, and `FinishGliderDuctingIn` (`:936`) |
| `kVRiseDuctSpeed` | -4 | `:756` | `MoveGliderUpDuct` |
| `kHPushMailSpeed` | -4 | `:853` | `FinishGliderMailingLeft` |
| `kHPushMailRtSpeed` | 4 | `:891` | `FinishGliderMailingRight` |
| `kHMailPullSpeed` | 4 | `:962` | `MoveGliderInMailLeft` |
| `kHMailPullRtSpeed` | -4 | `:1053` | `MoveGliderInMailRight` |
| `kVMailDropSpeed` | 2 | `:963`, `:1054` | max px/frame the player settles down inside a mail slot |
| `kDropShredSlow` | 1 | `:1262` | shredding descent while the shredder is still eating |
| `kDropShredFast` | 4 | `:1263` | shredding descent before reaching the shredder |

### 4.5 Mode timing

| Constant | Value | Where | Meaning |
| --- | --- | --- | --- |
| `kLastFadeSequence` | 16 | `GliderDefines.h:564` | length of `fadeInSequence[]`; fade-in ends at `frame >= 16`, fade-out starts at `frame = 15` |
| `kLeftFadeOffset` | 7 | `GliderDefines.h:563` | added to a fade sprite index for the left-facing variant |
| `kFirstAboutFaceFrame` | 18 | `GliderDefines.h:560` | first about-face sprite index (also the `frame` seed for facing right) |
| `kLastAboutFaceFrame` | 20 | `GliderDefines.h:561` | last about-face sprite index (also the `frame` seed for facing left) |
| `kWasBurning` | 2 | `GliderDefines.h:562` | sentinel *intended* to be stored in `frame` so a stairs or mail transition could restore burning; never actually written — its only writers (`Modes.c:128`, `:145`) sit behind `mode == kGliderBurning`, and every hot spot that would reach them kills a burning glider first. See §5.2 |
| `kFramesToBurn` | 60 | `Modes.c:408` | frames a burning glider survives (2 s at 30 fps) |
| `kKillWebbedGlider` | 150 | `Interactions.c:1738` | frames stuck in a spider web before dying (5 s) |
| `kVGliderAppearsComingUp` | 100 | `Modes.c:521` | y at which the player appears when arriving up a staircase |
| `kVGliderAppearsComingDown` | 100 | `Modes.c:552` | y at which the player appears when arriving down a staircase |
| `kNumCountDownFrames` | 16 | `GameOver.c:17` | frames between `FlagGameOver()` and the game-over screen |
| (idle delay) | 30 | `Modes.c:638` | `hVel = 30` in `TagGliderIdle`, a 30-frame freeze for the non-leading player |
| `kTicksPerFrame` | 2 | `GliderDefines.h:533` | 2 ticks = 1/30 s target frame time |
| `kDemoLength` | 6702 | `GliderDefines.h:625` | **bytes**, not frames — the size of the `demoData` buffer and of the shipped `'demo'` 128 resource, i.e. 1117 six-byte `{frame, key}` records spanning game frames 46–3414. See §10.3 |

### 4.6 Supplies and scoring that touch player velocity

| Constant | Value | Where | Effect on the player |
| --- | --- | --- | --- |
| `kBatterySupply` | 50 | `Interactions.c:16` | `batteryTotal += 50` (or `= 50` if it was helium) |
| `kHeliumSupply` | 150 | `Interactions.c:17` | `batteryTotal -= 150` (or `= -150` if it was battery) |
| `kBandsSupply` | 8 | `Interactions.c:18` | `bandsTotal += 8` |
| `kFoilSupply` | 8 | `Interactions.c:19` | `foilTotal += 8`, then `StartGliderFoilGoing` |
| `kInitialGliders` | 2 | `Play.c:19` | `mortals = 2` (4 in a two-player game) |
| `kRubberBandVelocity` | 20 | `RubberBands.c:12` | band `hVel = ±20`; player recoil is `-hVel/2` = ∓10 |
| `kBandFallCount` | 4 | `RubberBands.c:13` | band gains `vVel++` every 4 frames |

`batteryTotal`, `bandsTotal`, `foilTotal`, `mortals`, `showFoil` are **globals,
not per-glider fields** (`GliderPRO/Headers/GliderVars.h:56`–`:58`). In a
two-player game both players share one pool of energy, bands, foil and lives —
which is exactly why every reward doubles itself when
`twoPlayerGame && !onePlayerLeft` (`GliderPRO/Sources/Interactions.c:842`,
`:864`, `:883`, `:911`, `:970`).
---

## 5. The mode enum and dispatch table

### 5.1 Mode values

From `GliderPRO/Headers/GliderDefines.h:571`–`:594`. These are a plain
`#define` block with explicit consecutive values, so the numbers are stable and
must be preserved (they are written to disk in the saved game — see §16.4).

| # | Symbol | Handler | `dontDraw`? | Notes |
| --- | --- | --- | --- | --- |
| 0 | `kGliderNormal` | `MoveGliderNormal` | no | The only mode in which the player is fully interactive |
| 1 | `kGliderFadingIn` | `FadeGliderIn` | no | Materialising after respawn |
| 2 | `kGliderFadingOut` | `FadeGliderOut` | no | Dying |
| 3 | `kGliderGoingUp` | `MoveGliderUpStairs` | no | Walking up out of the top of the room |
| 4 | `kGliderComingUp` | `FinishGliderUpStairs` | no | Emerging at the bottom-right of the room above |
| 5 | `kGliderGoingDown` | `MoveGliderDownStairs` | no | Walking down out of the room |
| 6 | `kGliderComingDown` | `FinishGliderDownStairs` | no | Emerging at the left of the room below |
| 7 | `kGliderFaceLeft` | `MoveGliderFaceLeft` | no | About-face tumble, 3 frames |
| 8 | `kGliderFaceRight` | `MoveGliderFaceRight` | no | About-face tumble, 3 frames |
| 9 | `kGliderBurning` | `MoveGliderBurning` | no | On fire; 60-frame fuse |
| 10 | `kGliderTransporting` | `TransportGliderOut` | becomes yes | Dissolving into a transporter |
| 11 | `kGliderDuctingDown` | `MoveGliderDownDuct` | becomes yes | Sliding into a floor duct |
| 12 | `kGliderDuctingUp` | `MoveGliderUpDuct` | becomes yes | Rising into a ceiling duct |
| 13 | `kGliderDuctingIn` | `FinishGliderDuctingIn` | no | Dropping out of a ceiling duct |
| 14 | `kGliderMailInLeft` | `MoveGliderInMailLeft` | becomes yes | Being sucked into a left-facing mail slot |
| 15 | `kGliderMailOutLeft` | `FinishGliderMailingLeft` | no | Being pushed out of a left-facing mail slot |
| 16 | `kGliderMailInRight` | `MoveGliderInMailRight` | becomes yes | Being sucked into a right-facing mail slot |
| 17 | `kGliderMailOutRight` | `FinishGliderMailingRight` | no | Being pushed out of a right-facing mail slot |
| 18 | `kGliderGoingFoil` | `MoveGliderFoilGoing` | no | Dissolve-out / dissolve-in to gain foil |
| 19 | `kGliderLosingFoil` | `MoveGliderFoilLosing` | no | Dissolve-out / dissolve-in to lose foil |
| 20 | `kGliderShredding` | `MoveGliderShredding` | no | Being eaten by a paper shredder |
| 21 | `kGliderInLimbo` | (none — empty case) | unchanged — see §8.21 | Two-player only: waiting for the other player |
| 22 | `kGliderIdle` | `HandleIdleGlider` | no (true only for the `NewGame` spawn, `Play.c:190`) | Two-player only: 30-frame freeze after a room change |
| 23 | `kGliderTransportingIn` | `TransportGliderIn` | no | Materialising out of a transporter |

Only modes 0, 7, 8, 9 count as "in the room" for the wall/floor/ceiling logic
(`GliderPRO/Sources/Interactions.c:691`–`:694`) and only 0, 7, 8, 9, 18, 19
count for dynamic-object collision (`GliderPRO/Sources/Dynamics.c:45`–`:50`).

The `dontDraw?` column records what a mode's own entry function or handler writes
to the field, and nothing more. Visibility is never a function of `mode`:
`RenderGlider` gates on `if (thisGlider->dontDraw) return;` alone
(`GliderPRO/Sources/Render.c:457`–`:458`). Neither of the two-player parking modes
touches the flag — `FlagGliderInLimbo` (`GliderPRO/Sources/Modes.c:458`–`:468`)
writes only `wasMode`, `mode` and `firstPlayer`, and `TagGliderIdle`
(`Modes.c:631`–`:639`) writes only `wasMode`, `mode` and `hVel`. So a limbo glider
is hidden only when the mode it came *from* hid it, and an idle glider is drawn.
A port that early-returns from its render path on mode 21 or 22 is wrong; see
§8.21 and §8.22.

### 5.2 What `frame` means in each mode

`frame` is a single `short` reused as five completely different things.
Getting this wrong is one of the easiest ways to break a port.

| Mode(s) | `frame` holds | Range / direction |
| --- | --- | --- |
| 1 `kGliderFadingIn`, 23 `kGliderTransportingIn` | index into `fadeInSequence[16]` | 0 → 16, then mode 0 |
| 2 `kGliderFadingOut`, 10 `kGliderTransporting` | index into `fadeInSequence[16]` | 15 → −1, then death / transit |
| 3, 5 (stairs out) | **`kWasBurning` (2) sentinel or 0** — carries "I was on fire" across the room boundary | set once at mode entry |
| 4, 6, 13, 15, 17 (arrivals) | same sentinel, tested with `frame == kWasBurning` | read once on completion |
| 7 `kGliderFaceLeft` | sprite index, counting down | 20 → 17, exits below 18 |
| 8 `kGliderFaceRight` | sprite index, counting up | 18 → 21, exits above 20 |
| 9 `kGliderBurning` | flame animation phase | 0,1,2,3,0,1,… (`wasMode` is the fuse) |
| 18, 19 (foil) | dissolve phase | 0 → 9, exits above 8 |
| 20 `kGliderShredding` | **a y coordinate** (`bounds->bottom - 3`) while positive, then a **negative countdown** from `kShredderCountdown` (−68) to 0 | see §8.20 |
| 11, 12, 14, 16 (duct departures, mail-in) | zeroed at mode entry, then never read | `Modes.c:236`, `:269`, `:173` |
| 21, 22 | unused | — |

Mode 13 sits on the arrivals row rather than with the other duct modes because
`FinishGliderDuctingIn` runs the same sentinel test as the stairs and mail arrivals — `if
(thisGlider->frame == kWasBurning) FlagGliderBurning(thisGlider); else
FlagGliderNormal(thisGlider);` (`GliderPRO/Sources/Player.c:949`–`:952`). It is
also the one mode that really does inherit `frame` rather than set it:
`StartGliderDuctingIn` (`Modes.c:279`–`:284`) writes `mode`, `whole` and
`dontDraw` and nothing else, so mode 13 sees whatever the mode-11 or mode-12
starter left, which is always 0 (`Modes.c:236`, `:269`).

The branch is nevertheless dead in the original, for two independent reasons: a
burning glider can never reach a duct at all — every hot-spot case kills it first
with `wasMode = 0; StartGliderFadingOut(...)`
(`GliderPRO/Sources/Interactions.c:1425`–`:1430`, `:1468`–`:1473`,
`:1511`–`:1516`, `:1544`–`:1549`) — and even if it could, the duct starters have
already zeroed `frame`. The trap for a porter is to read "unused" here, drop the
`frame = 0` writes at duct entry as dead stores, and keep the `:949` test that
§8.13 correctly describes: any residual 2 left over from the mode-9 flame
animation (`Player.c:205`–`:207`, and `kWasBurning` is itself 2) then drops the
glider out of the ceiling duct on fire with a fresh 60-frame fuse.

The sentinel is in fact dead on every route, not just the duct one, so no arrival
in the original ever resumes burning. `kWasBurning` has exactly two writers,
`StartGliderGoingUpStairs` and `StartGliderGoingDownStairs` (`Modes.c:128`,
`:145`), and both sit inside `if (thisGlider->mode == kGliderBurning)` — yet the
`kMoveItUp` and `kMoveItDown` cases already kill a burning glider before they can
reach those starters (`Interactions.c:1253`–`:1258`, `:1288`–`:1293`). Nothing else
in the program assigns 2 to `frame` as a sentinel, so all five tests
(`Player.c:417`, `:546`, `:879`, `:917`, `:949`) always fall through to
`FlagGliderNormal`. Transcribe them for fidelity; do not build fire propagation
across stairs, mailboxes or ducts on top of them.

### 5.3 What `wasMode` means

| Situation | Meaning | Set at | Read at |
| --- | --- | --- | --- |
| `kGliderBurning` | remaining frames before burning to death; counts **down** from `kFramesToBurn` (60) | `Modes.c:432` | `Player.c:220`–`:225` |
| webbed (any mode) | frames stuck in the web; counts **up** to `kKillWebbedGlider` (150) | `Interactions.c:1769` | `Interactions.c:1770` |
| `kGliderInLimbo` | the mode to restore when limbo ends | `Modes.c:460` | `Modes.c:478` |
| `kGliderIdle` | the mode to restore when the 30-frame freeze ends | `Modes.c:636` | `Player.c:1328` |
| entering a wall/floor/ceiling while burning | zeroed so the burn timer cannot also fire | `Interactions.c:700`, `:713`, `:729`, `:742` | — |

Note the collision between "burning fuse" and "web counter": a **burning glider
caught in a web** would have both meanings applied to `wasMode`. The code avoids
this by special-casing it — `WebGlider` immediately kills a burning glider that
is fully inside the web (`GliderPRO/Sources/Interactions.c:1741`–`:1747`).

### 5.4 `HandleGlider` — the dispatcher

`GliderPRO/Sources/Player.c:1335`–`:1439`. Pseudocode, in the original switch
order:

```
 1  HandleGlider(thisGlider):
 2      switch (thisGlider->mode):               // Player.c:1337, no default: case
 3        case kGliderNormal:            MoveGliderNormal(thisGlider)        // :1339
 4        case kGliderFadingIn:          FadeGliderIn(thisGlider)            // :1343
 5        case kGliderFadingOut:         FadeGliderOut(thisGlider)           // :1347
 6        case kGliderGoingUp:           MoveGliderUpStairs(thisGlider)      // :1351
 7        case kGliderComingUp:          FinishGliderUpStairs(thisGlider)    // :1355
 8        case kGliderGoingDown:         MoveGliderDownStairs(thisGlider)    // :1359
 9        case kGliderComingDown:        FinishGliderDownStairs(thisGlider)  // :1363
10        case kGliderFaceLeft:          MoveGliderFaceLeft(thisGlider)      // :1367
11        case kGliderFaceRight:         MoveGliderFaceRight(thisGlider)     // :1371
12        case kGliderBurning:           MoveGliderBurning(thisGlider)       // :1375
13        case kGliderTransporting:      TransportGliderOut(thisGlider)      // :1379
14        case kGliderDuctingDown:       MoveGliderDownDuct(thisGlider)      // :1383
15        case kGliderDuctingUp:         MoveGliderUpDuct(thisGlider)        // :1387
16        case kGliderDuctingIn:         FinishGliderDuctingIn(thisGlider)   // :1391
17        case kGliderMailInLeft:        MoveGliderInMailLeft(thisGlider)    // :1395
18        case kGliderMailOutLeft:       FinishGliderMailingLeft(thisGlider) // :1399
19        case kGliderMailInRight:       MoveGliderInMailRight(thisGlider)   // :1403
20        case kGliderMailOutRight:      FinishGliderMailingRight(thisGlider)// :1407
21        case kGliderGoingFoil:         MoveGliderFoilGoing(thisGlider)     // :1411
22        case kGliderLosingFoil:        MoveGliderFoilLosing(thisGlider)    // :1415
23        case kGliderShredding:         MoveGliderShredding(thisGlider)     // :1419
24        case kGliderInLimbo:           /* nothing */                       // :1423-1424
25        case kGliderIdle:              HandleIdleGlider(thisGlider)        // :1426
26        case kGliderTransportingIn:    TransportGliderIn(thisGlider)       // :1430
27
28      thisGlider->ignoreLeft   = false          // Player.c:1436
29      thisGlider->ignoreRight  = false          // Player.c:1437
30      thisGlider->ignoreGround = false          // Player.c:1438
```

The switch is in exact mode-enum order 0–23, with every mode present. Every case
`break`s and there is no `default:`, so an unrecognised mode is a silent no-op
followed by the `ignore*` clears — a Go port using a 24-entry table indexed by
mode must therefore bounds-check rather than panic, and must keep the clears
outside the dispatch.

What *is* out of enum order in `Player.c` is the forward-declaration block
(`:20`–`:40`) and the order the functions are defined in: `MoveGliderNormal` (0)
at `:151`, `MoveGliderBurning` (9) at `:203`, `FadeGliderIn` (1) at `:231`,
`TransportGliderIn` (23) at `:261`, `FadeGliderOut` (2) at `:291`. That is a
reading-order curiosity with no bearing on dispatch, and it is easy to mistake
for a dispatch-order subtlety that does not exist.

Lines 28–30 are the critical part: **`ignoreLeft`, `ignoreRight` and
`ignoreGround` are one-frame flags cleared unconditionally at the end of
`HandleGlider`, for every mode.** They are set earlier in the same frame by
`HandleHotSpotCollision` (`GliderPRO/Sources/Interactions.c:1417`, `:1421`,
`:1587`), which runs before `HandleGlider` in the frame order (§17). So "I am
standing in a doorway" is communicated from the interaction pass to the movement
pass through these three booleans and is forgotten immediately afterwards.

`sliding` is *not* cleared here; it is cleared inside `MoveGliderNormal`
(`GliderPRO/Sources/Player.c:159`, `:181`) after it has been used to pick the
sprite, so it too survives exactly one frame. See §8.0.

---

## 6. The sprite atlas and `gliderSrc[31]`

### 6.1 The atlas resources

`InitGliderMap` (`GliderPRO/Sources/StructuresInit.c:169`–`:238`) builds three
offscreen GWorlds from PICT resources and one shared 1-bit mask:

| Global | PICT ID | Symbol | Contents |
| --- | --- | --- | --- |
| `glidSrcMap` | 3999 | `kGliderPictID` (`GliderDefines.h:568`) | plain glider, player 1 colours |
| `glid2SrcMap` | 3974 | `kGlider2PictID` (`GliderDefines.h:566`) | plain glider, player 2 colours |
| (loaded on demand) | 3976 | `kGliderFoilPictID` (`GliderDefines.h:567`) | foil-clad glider, player 1 colours |
| (loaded on demand) | 3963 | `kGliderFoil2PictID` (`GliderDefines.h:565`) | foil-clad glider, player 2 colours |
| `glidMaskMap` | **4999** = `kGliderPictID + 1000` | — | 1-bit mask, depth 1 |
| `shadowSrcMap` | 3998 | `kShadowPictID` (`StructuresInit.c:19`) | the two ground-shadow sprites |
| `shadowMaskMap` | 4998 | `kShadowPictID + 1000` | 1-bit shadow mask |
| `bandsSrcMap` | 4007 | `kRubberBandsPictID` (`StructuresInit.c:27`) | the three rubber-band frames |
| `bandsMaskMap` | 5007 | — | 1-bit band mask |

**Verified from the resource dump.** Parsing `data 'PICT' (id, …)` blocks out of
`GliderPRO/Glider PRO.r` with python3 and unpacking the `picSize` +
`picFrame` header (`>Hhhhh`) gives, observed:

```
PICT 3999 reslen=18688 picSize=18688 frame=(t=0 l=0 b=668 r=48) -> 48x668  ver=0x0011
PICT 4999 reslen= 4102 picSize= 4102 frame=(t=0 l=0 b=668 r=48) -> 48x668  ver=0x1101
PICT 3974 reslen=17604 picSize=17604 frame=(t=0 l=0 b=668 r=48) -> 48x668  ver=0x0011
PICT 3976 reslen=14846 picSize=14846 frame=(t=0 l=0 b=668 r=48) -> 48x668  ver=0x0011
PICT 3963 reslen=15258 picSize=15258 frame=(t=0 l=0 b=668 r=48) -> 48x668  ver=0x0011
PICT 3998 reslen=  167 picSize=  167 frame=(t=0 l=0 b= 18 r=48) -> 48x18   ver=0x1101
PICT 4998 reslen=  167 picSize=  167 frame=(t=0 l=0 b= 18 r=48) -> 48x18
PICT 4007 reslen= 2384 picSize= 2384 frame=(t=0 l=0 b= 18 r=16) -> 16x18
PICT 5007 reslen=   95 picSize=   95 frame=(t=0 l=0 b= 18 r=16) -> 16x18
```

and **PICT 4974, 4976 and 4963 do not exist** — one shared mask (4999) serves
all four 48x668 glider atlases. (The natural explanation is that all four atlases
have pixel-identical silhouettes, but that is an inference: it was not checked by
decompressing the PackBits pixel data. (unverified)) The load-bearing part is
mechanical and does hold: `CopyMask` is always called with `glidMaskMap` as the
mask regardless of which atlas is the source, so a Go port that renders with
per-atlas alpha must derive alpha from the 4999 mask, not from each atlas.

Two further observations from the same parse: `Glider PRO.r` declares exactly
**152** `PICT` resources in total, and **PICT 3998 and PICT 4998 are byte-for-byte
identical** (both 167 bytes) — the ground shadow is its own mask, so a port needs
to decode only one of the two.

`kGliderWide` = 48 (`GliderDefines.h:548`) and `kGliderHigh` = 20 (`:549`), so the
48-pixel atlas width and the 20-pixel normal sprite height in §6.2 both come
straight from the headers.

`glidSrcRect` is declared as `QSetRect(&glidSrcRect, 0, 0, kGliderWide, 668);`
with the comment `// 32112 pixels` (`GliderPRO/Sources/StructuresInit.c:178`),
and 48 x 668 = 32064 — the comment is off by 48, i.e. it counts one extra row of
48. Harmless, but it confirms 48 x 668 was hand-computed.

### 6.2 The 31 source rectangles

`GliderPRO/Sources/StructuresInit.c:191`–`:205`, `kNumGliderSrcRects` = 31
(`GliderDefines.h:558`):

```c
for (i = 0; i <= 20; i++)          // 21 rects, 20 px tall     :191-195
{
    QSetRect(&gliderSrc[i], 0, 0, kGliderWide, kGliderHigh);
    QOffsetRect(&gliderSrc[i], 0, kGliderHigh * i);
}
for (i = 21; i <= 28; i++)         // 8 rects, 26 px tall (burning)  :196-200
{
    QSetRect(&gliderSrc[i], 0, 0, kGliderWide, kGliderBurningHigh);
    QOffsetRect(&gliderSrc[i], 0, 420 + (kGliderBurningHigh * (i - 21)));
}
QSetRect(&gliderSrc[29], 0, 0, kGliderWide, kGliderHigh);   // :202
QOffsetRect(&gliderSrc[29], 0, 628);                        // :203
QSetRect(&gliderSrc[30], 0, 0, kGliderWide, kGliderHigh);   // :204
QOffsetRect(&gliderSrc[30], 0, 648);                        // :205
```

The arithmetic closes exactly: 21 x 20 = **420**, 8 x 26 = **208**, 2 x 20 =
**40**; 420 + 208 + 40 = **668**, matching the observed PICT frame height.

| Index | y range | Height | Sprite | Evidence |
| --- | --- | --- | --- | --- |
| 0 | 0–20 | 20 | right-facing, level | `Player.c:192` (facing right, not tipped), `Modes.c:348` |
| 1 | 20–40 | 20 | right-facing, tipped (banked) | `Player.c:187` (facing right, tipped), also up-stairs right (`Player.c:327`) |
| 2 | 40–60 | 20 | left-facing, level | `Player.c:170`, `Modes.c:343`, all stairs/mail neutral poses |
| 3 | 60–80 | 20 | left-facing, tipped | `Player.c:165` |
| 4 | 80–100 | 20 | right-facing dissolve, **faintest** | `fadeInSequence[0] == 4` |
| 5 | 100–120 | 20 | right dissolve 2 | |
| 6 | 120–140 | 20 | right dissolve 3 | |
| 7 | 140–160 | 20 | right dissolve 4 | |
| 8 | 160–180 | 20 | right dissolve 5 | |
| 9 | 180–200 | 20 | right dissolve 6 | |
| 10 | 200–220 | 20 | right dissolve 7, **most solid** | `fadeInSequence[15] == 10` |
| 11 | 220–240 | 20 | left dissolve, faintest (= 4 + `kLeftFadeOffset`) | `Player.c:246`–`:249` |
| 12 | 240–260 | 20 | left dissolve 2 | |
| 13 | 260–280 | 20 | left dissolve 3 | |
| 14 | 280–300 | 20 | left dissolve 4 | |
| 15 | 300–320 | 20 | left dissolve 5 | |
| 16 | 320–340 | 20 | left dissolve 6 | |
| 17 | 340–360 | 20 | left dissolve 7, most solid | |
| 18 | 360–380 | 20 | about-face frame A | `kFirstAboutFaceFrame`, `Modes.c:452` |
| 19 | 380–400 | 20 | about-face frame B | |
| 20 | 400–420 | 20 | about-face frame C | `kLastAboutFaceFrame`, `Modes.c:442` |
| 21 | 420–446 | **26** | burning, right-facing, phase 0 | `Modes.c:424`, `Player.c:216` |
| 22 | 446–472 | 26 | burning, right, phase 1 | |
| 23 | 472–498 | 26 | burning, right, phase 2 | |
| 24 | 498–524 | 26 | burning, right, phase 3 | |
| 25 | 524–550 | 26 | burning, left-facing, phase 0 | `Modes.c:419`, `Player.c:211` |
| 26 | 550–576 | 26 | burning, left, phase 1 | |
| 27 | 576–602 | 26 | burning, left, phase 2 | |
| 28 | 602–628 | 26 | burning, left, phase 3 | |
| 29 | 628–648 | 20 | **sliding, right-facing** | `Player.c:179` |
| 30 | 648–668 | 20 | **sliding, left-facing** | `Player.c:157` |

The dissolve direction is proved by `MoveGliderFoilGoing`
(`GliderPRO/Sources/Player.c:1170`–`:1198`): frames 1..4 use `10 - frame` =
9, 8, 7, 6 (fading away), then frames 5..8 route to `DeckGliderInFoil` which uses
`frame + 2` = 7, 8, 9, 10 (fading back in, now from the foil atlas). So the
sequence is 9, 8, 7, 6, 7, 8, 9, 10 — a symmetric dissolve through the faintest
frame and back, with the atlas swapped at the midpoint. Index 4 is therefore the
faintest and 10 the most solid.

### 6.3 `fadeInSequence[16]`

`GliderPRO/Sources/InterfaceInit.c:167`–`:182`, observed values:

| index | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| value | 4 | 5 | 6 | 7 | 5 | 6 | 7 | 8 | 6 | 7 | 8 | 9 | 7 | 8 | 9 | 10 |

This is a 4-frame flicker cycle whose base creeps up by 1 every 4 frames
(4,5,6,7 / 5,6,7,8 / 6,7,8,9 / 7,8,9,10), producing a shimmering
materialisation over 16 frames (0.53 s). The **left-facing** variant adds
`kLeftFadeOffset` = 7 to the table value, giving 11..17
(`GliderPRO/Sources/Player.c:246`–`:249`).

Fade-out replays the same table backwards from index 15
(`GliderPRO/Sources/Player.c:293`, `:300`–`:308`), so a dying glider goes
10, 9, 8, 7, 9, 8, 7, 6, … 4 and then vanishes.

### 6.4 The shadow sprites

`shadowSrc[2]`, each 48 x 9, at y = 0 and y = 9
(`GliderPRO/Sources/StructuresInit.c:216`–`:220`;
`kNumShadowSrcRects` = 2, `GliderDefines.h:559`). Index selection is
`which = (facing == kFaceRight) ? 0 : 1`
(`GliderPRO/Sources/Render.c:459`–`:462`), i.e. index 0 is the right-facing
shadow, index 1 the left-facing one.

---

## 7. `MoveGlider` — the integrator

`GliderPRO/Sources/Player.c:64`–`:147`. This single function does **all** of:
ramping `hVel`/`vVel` toward the desired velocities, resetting the desired
velocities, clamping horizontal speed, snapshotting `wasHVel`/`wasVVel`, moving
`dest` and `destShadow`, and computing the swept `whole`/`wholeShadow`.

It is called at the end of `MoveGliderNormal`, `MoveGliderBurning`,
`MoveGliderFaceLeft`, `MoveGliderFaceRight` and `MoveGliderFoilGoing` /
`MoveGliderFoilLosing` — i.e. exactly the six modes in which the player has
momentum. Every other mode moves `dest` by a hard-coded speed instead.

### 7.1 Full pseudocode

```
 1  MoveGlider(thisGlider):
 2      // ---- horizontal ramp (Player.c:66-78) ----
 3      if (hVel > hDesiredVel):                     // Player.c:66 -- ">" is tested FIRST
 4          hVel -= kHImpulse                        // -2
 5          if (hVel < hDesiredVel): hVel = hDesiredVel      // never overshoot
 6      else if (hVel < hDesiredVel):                // Player.c:72
 7          hVel += kHImpulse                        // +2
 8          if (hVel > hDesiredVel): hVel = hDesiredVel
 9      hDesiredVel = 0                              // Player.c:78  <-- reset
10
11      // ---- vertical ramp (Player.c:80-92) ----
12      if (vVel > vDesiredVel):                     // Player.c:80
13          vVel -= kVImpulse                        // -2
14          if (vVel < vDesiredVel): vVel = vDesiredVel
15      else if (vVel < vDesiredVel):                // Player.c:86
16          vVel += kVImpulse                        // +2
17          if (vVel > vDesiredVel): vVel = vDesiredVel
18      vDesiredVel = kGravity                       // Player.c:92 = 3  <-- reset
19
20      // ---- horizontal move + sweep (Player.c:94-127) ----
21      if (hVel < 0):                               // Player.c:94, moving left
22          if (hVel < -kMaxHVel): hVel = -kMaxHVel  // clamp to -16
23          wasHVel = hVel                           // Player.c:99
24          whole.right = dest.right                 // pre-move right edge
25          dest.left  += hVel
26          dest.right += hVel
27          whole.left  = dest.left                  // post-move left edge
28          wholeShadow.right = destShadow.right
29          destShadow.left  += hVel
30          destShadow.right += hVel
31          wholeShadow.left  = destShadow.left
32      else:                                        // Player.c:111 -- plain else, so
33                                                   //   hVel == 0 lands HERE
34          if (hVel > kMaxHVel): hVel = kMaxHVel    // clamp to +16
35          wasHVel = hVel                           // Player.c:116 (runs even for 0)
36          whole.left = dest.left                   // pre-move left edge
37          dest.left  += hVel
38          dest.right += hVel
39          whole.right = dest.right
40          wholeShadow.left = destShadow.left
41          destShadow.left  += hVel
42          destShadow.right += hVel
43          wholeShadow.right = destShadow.right
44
45      // ---- vertical move + sweep (Player.c:129-146) ----
46      if (vVel < 0):                               // Player.c:129, moving up
47          wasVVel = vVel                           // Player.c:131
48          whole.bottom = dest.bottom
49          dest.top    += vVel
50          dest.bottom += vVel
51          whole.top    = dest.top
52      else:                                        // Player.c:138 -- plain else, so
53                                                   //   vVel == 0 lands HERE
54          wasVVel = vVel                           // Player.c:140 (runs even for 0)
55          whole.top = dest.top
56          dest.top    += vVel
57          dest.bottom += vVel
58          whole.bottom = dest.bottom
59      // NOTE: destShadow is NEVER moved vertically here.
```

Both move blocks use `if (vel < 0) { … } else { … }` — **not** `else if (vel > 0)`.
That is load-bearing: a zero-velocity axis still takes the positive branch, so
`wasHVel`/`wasVVel` are refreshed to 0 and all four edges of `whole` are
rewritten every single call.

### 7.2 Consequences a port must reproduce exactly

1. **No vertical clamp.** Lines 22 and 34 clamp `hVel` to ±`kMaxHVel` (16), but
   there is no equivalent for `vVel`. The vertical clamp is *emergent*: because
   `vDesiredVel` is reset to `+3` every frame and the ramp is ±2, a falling
   glider converges on exactly `vVel = 3`. The trap is to conclude from the
   absence of a clamp that a large world force gets one uncapped frame at full
   strength. It does not: `HandleInteraction` runs *before* `HandleGlider` in
   the same frame (`GliderPRO/Sources/Play.c:482` then `:487`), so everything
   the world writes is filtered by the ramp on its way into `dest`, and the two
   kinds of write are filtered differently. `kDropIt` writes a *target*, not a
   velocity — `thisGlider->vDesiredVel = kCeilingVentDrop;`
   (`GliderPRO/Sources/Interactions.c:1207`) — which the ramp then walks towards
   at `kVImpulse` = 2 per frame (`GliderPRO/Sources/Player.c:86`–`:91`), so a
   glider that enters a ceiling blower already falling at `vVel = 3` is
   displaced 5, then 7, then 8, then 8, and *holds* at 8 for as long as the hot
   spot keeps overlapping: `CheckForHotSpots`
   (`GliderPRO/Sources/Interactions.c:1627`–`:1687`) rewrites `vDesiredVel = 8`
   again every frame after `MoveGlider` has reset it to `kGravity`
   (`GliderPRO/Sources/Player.c:92`), so the decay back to `+3` only begins on
   the first frame the hot spot no longer overlaps. `kSlideIt` does write `vVel`
   directly (`GliderPRO/Sources/Interactions.c:1378`), but the vertical ramp
   (`GliderPRO/Sources/Player.c:80`–`:91`) still runs before the move
   (`:129`–`:146`), so the magnitude that actually reaches `dest` is smaller by
   up to `kVImpulse` = 2 — see §13.2 for where that leaves the glider. The same
   attenuation applies to every other direct velocity write in `Interactions.c`
   that leaves the mode alone: `BounceGlider` (`:160`, `:162`), the ceiling stops
   in `CheckEscapeUp` (`:272`, `:275`, `:278`) and `WebGlider` (`:1758`,
   `:1759`) all set velocity and never touch `dest`, so each loses up to 2 px to
   the ramp on the frame it fires. The floor and roof writes (`:353`, `:369`,
   `:411`, `:426`, `:441`, `:463`, `:473`, `:483`, `:493`, `:500`) go further
   still: each is followed immediately by `StartGliderFadingOut`, and
   `FadeGliderOut` (`GliderPRO/Sources/Player.c:291`–`:311`) never calls
   `MoveGlider`, so those `vVel` values are never fed to the integrator at all.
   A Go port that adds a symmetric `vVel` clamp, or that applies any of these
   writes straight to position, will change the feel of ceiling blowers and
   grease and will shift every collision resting position by 2 px.

2. **The clamp is inside the sign branches.** If `hVel` were somehow ±20 while
   `hDesiredVel` had the same sign, the clamp still fires because the branch is
   chosen on the *sign of `hVel`*, not on whether the ramp moved. The split is
   `if (hVel < 0) … else …` (`GliderPRO/Sources/Player.c:94`, `:111`), so
   `hVel == 0` takes the "moving right" branch: it is compared against
   `+kMaxHVel` (a no-op) and, importantly, **`wasHVel` *is* refreshed to 0**
   (`GliderPRO/Sources/Player.c:116`). `GliderHitTop` reads `wasHVel`
   (`GliderPRO/Sources/Interactions.c:65`–`:66`), so a glider that is stationary
   *at the moment `MoveGlider` last ran* un-sweeps its hit box by 0, not by the
   last non-zero horizontal velocity. The same holds for `wasVVel` via
   `:129`/`:138`. A port must write `wasHVel`/`wasVVel` unconditionally.

3. **Battery thrust bypasses the ramp.** `DoBatteryEngaged` writes
   `hVel += ±kHyperThrust` directly (`GliderPRO/Sources/Input.c:126`–`:135`),
   before `MoveGlider` runs. So on a battery frame `hVel` can be up to 8 beyond
   whatever the ramp would allow, and the clamp then trims it to ±16. Combined
   with a held direction key the practical result is: hold key → `hDesiredVel` =
   ±5, so the ramp pulls toward ±5; battery adds ±8 per frame on top; the clamp
   pins the result at ±16. Steady-state battery flight is therefore exactly
   ±16 px/frame.

4. **The sweep is one-sided per axis.** `whole` is *not* a union of the old and
   new rects computed by min/max; it is built by writing the trailing edge before
   the move and the leading edge after. That happens to equal the union **only
   because** `dest` never changes size during a `MoveGlider` call. If a port
   changes the glider's height mid-move (as burning does at mode entry, not
   during movement) the naive union and this algorithm diverge. Reproduce the
   statements, not the intent.

5. **`whole` is fully rewritten by every `MoveGlider` call.** Because both move
   blocks are `if (vel < 0) … else …`, all four edges of `whole` are assigned on
   every call: the horizontal block writes `.left` and `.right`, the vertical
   block writes `.top` and `.bottom`. It therefore does **not** accumulate across
   frames, and for a glider with `hVel == vVel == 0` it collapses to exactly
   `dest`. The `whole = dest` assignments in the mode-entry functions
   (`GliderPRO/Sources/Modes.c:31`, `:56`, `:102`, `:282`, `:316`, `:540`,
   `:571`, `:589`, `:615`) matter only for the modes that *don't* call
   `MoveGlider` — for the six momentum modes they are redundant with the first
   `MoveGlider` call.

6. **The shadow only tracks horizontally.** `destShadow.top`/`.bottom` are never
   touched by `MoveGlider`. They are set once by `InitGlider`
   (`GliderPRO/Sources/Play.c:354`–`:356`, `QSetRect(…, kShadowHigh)` then
   `QOffsetRect(…, dest.left, kShadowTop)`), and thereafter only
   `destShadow.bottom = destShadow.top + kShadowHigh` is re-imposed by the
   mode-entry functions (e.g. `GliderPRO/Sources/Modes.c:339`) — note they
   re-derive `.bottom` from `.top` but never rewrite `.top` itself. The shadow is
   a fixed-height blob on the floor that slides left and right under the glider.
   There is no scaling or fading with altitude.

7. **`wasVVel` is written but appears to be read nowhere.** It is assigned at
   `Player.c:131` and `:140`. A port may omit it (but keep it if
   byte-compatibility of a saved struct matters — it does not, `gliderType` is
   never serialised; see §16.4).
   > [verified: `wasHVel` is read by `GliderHitTop`; no read of `wasVVel` was
   > found, but the search was over the `Sources/` and `Headers/` trees only]

### 7.3 Worked traces

Free fall from rest, no input (`hDesiredVel` 0, `vDesiredVel` 3 every frame):

| frame | `vVel` before | ramp | `vVel` after | `dest.top` delta |
| --- | --- | --- | --- | --- |
| 1 | 0 | +2 | 2 | +2 |
| 2 | 2 | +2 → clamp to 3 | 3 | +3 |
| 3 | 3 | none | 3 | +3 |
| n≥3 | 3 | none | 3 | +3 |

Terminal fall speed is therefore **3 px/frame = 90 px/s**, and it is reached in
two frames. A room is 322 px tall, so a full-height fall takes about 108 frames
(3.6 s).

Holding "right" from rest (`hDesiredVel` = +5 each frame):

| frame | `hVel` | note |
| --- | --- | --- |
| 1 | 2 | +kHImpulse |
| 2 | 4 | |
| 3 | 5 | clamped to `hDesiredVel`, not overshooting to 6 |
| n≥3 | 5 | |

Releasing the key: `hDesiredVel` becomes 0, so `hVel` decays 5 → 3 → 1 → 0 over
three frames. Total horizontal cruise speed is **5 px/frame = 150 px/s**; with
the battery, **16 px/frame = 480 px/s**.

Helium (`vDesiredVel` = −4 each frame) from terminal fall: `vVel` goes
3 → 1 → −1 → −3 → −4, i.e. four frames to reach full lift. Rise speed is
**4 px/frame**, slightly faster than the fall speed of 3.

---

## 8. Every mode handler

Ordered by mode number. All line references are `GliderPRO/Sources/Player.c`
unless stated.

### 8.0 Mode 0 — `kGliderNormal` / `MoveGliderNormal` (`:151`–`:199`)

This handler does **only two things**: pick the sprite, then integrate.

```
 1  MoveGliderNormal(thisGlider):
 2      if (facing == kFaceLeft):                    // :153
 3          if (sliding):                            // :155
 4              src = mask = gliderSrc[30]           // :157-158  sliding-left
 5              sliding = false                      // :159      <- consumed
 6          else if (tipped):
 7              src = mask = gliderSrc[3]            // :165-166  left, banked
 8          else:
 9              src = mask = gliderSrc[2]            // :170-171  left, level
10      else:                                        // :175
11          if (sliding):
12              src = mask = gliderSrc[29]           // :179-180  sliding-right
13              sliding = false                      // :181
14          else if (tipped):
15              src = mask = gliderSrc[1]            // :187-188  right, banked
16          else:
17              src = mask = gliderSrc[0]            // :192-193  right, level
18      MoveGlider(thisGlider)                       // :198
```

Points that matter:

* **`facing` is never written here.** Sprite selection reads `facing`, `sliding`
  and `tipped`; it does not touch them (other than consuming `sliding`).
  `facing` is written at exactly six places in the whole game:
  `GliderPRO/Sources/Player.c:571` (about-face-left completed),
  `GliderPRO/Sources/Player.c:588` (about-face-right completed),
  `GliderPRO/Sources/Modes.c:190` / `:197` (`StartGliderMailingOut`, by slot
  direction), `GliderPRO/Sources/Modes.c:526` (`ReadyGliderForTripUpStairs`
  forces `kFaceLeft`), `GliderPRO/Sources/Modes.c:557`
  (`ReadyGliderForTripDownStairs` forces `kFaceRight`), plus
  `GliderPRO/Sources/Play.c:324` / `:348` in `InitGlider`. **The only way the
  player can turn around during normal flight is the about-face tumble** (modes
  7/8), which is triggered by pressing both direction keys at once
  (`GliderPRO/Sources/Input.c:308`) or by `InsureGliderFacing*` on a horizontal
  room change (`GliderPRO/Sources/Transit.c:164`–`:168`, `:196`–`:200`).
* **`sliding` is consumed unconditionally** by whichever facing branch runs, so it
  lasts exactly one frame of normal flight. The `kSlideIt` hot spot re-sets it
  every frame the glider stays on the grease
  (`GliderPRO/Sources/Interactions.c:1377`).
* There is no `else` for "no sprite change" — a normal glider's sprite is
  recomputed from scratch every frame, so it is fully determined by
  (`facing`, `sliding`, `tipped`).

The complete normal-flight sprite table:

| `facing` | `sliding` | `tipped` | sprite | atlas y |
| --- | --- | --- | --- | --- |
| left (0) | true | (ignored) | `gliderSrc[30]` | 648–668 |
| left (0) | false | true | `gliderSrc[3]` | 60–80 |
| left (0) | false | false | `gliderSrc[2]` | 40–60 |
| right (1) | true | (ignored) | `gliderSrc[29]` | 628–648 |
| right (1) | false | true | `gliderSrc[1]` | 20–40 |
| right (1) | false | false | `gliderSrc[0]` | 0–20 |

### 8.1 Mode 1 — `kGliderFadingIn` / `FadeGliderIn` (`:231`–`:257`)

```
 1  FadeGliderIn(thisGlider):
 2      if (frame == 0): PlayPrioritySound(kFadeInSound, kFadeInPriority)   // :233-234
 3      frame++                                                            // :236
 4      if (frame >= kLastFadeSequence):        // 16
 5          FlagGliderNormal(thisGlider)                                   // :239
 6          enteredRect = dest                                             // :240
 7      else:
 8          if (facing == kFaceLeft):
 9              src = mask = gliderSrc[fadeInSequence[frame] + kLeftFadeOffset]   // +7
10          else:
11              src = mask = gliderSrc[fadeInSequence[frame]]
12      // NOTE: no MoveGlider call - the glider is motionless while fading in
```

`enteredRect = dest` at line 6 is the respawn anchor: **the place you fade in is
the place you will fade in again next time you die in this room.**

### 8.2 Mode 2 — `kGliderFadingOut` / `FadeGliderOut` (`:291`–`:311`)

```
 1  FadeGliderOut(thisGlider):
 2      frame--                                     // :293
 3      if (frame < 0):
 4          OffAMortal(thisGlider)                  // :295  -> lose a life
 5      else:
 6          if (facing == kFaceLeft):
 7              src = mask = gliderSrc[fadeInSequence[frame] + kLeftFadeOffset]
 8          else:
 9              src = mask = gliderSrc[fadeInSequence[frame]]
10      // no MoveGlider - motionless
```

Entry sets `frame = kLastFadeSequence - 1` = 15 (`GliderPRO/Sources/Modes.c:103`),
so the fade-out is 16 frames and `OffAMortal` fires on the 17th.

### 8.3 Mode 3 — `kGliderGoingUp` / `MoveGliderUpStairs` (`:315`–`:379`)

`kClimbStairsSpeed` = **−4** (`:317`).

```
 1  MoveGliderUpStairs(thisGlider):
 2      if (facing == kFaceLeft): src = mask = gliderSrc[2]
 3      else:                     src = mask = gliderSrc[1]
 4      whole.bottom = dest.bottom                  // :331 sweep trailing edge
 5      dest.top    += kClimbStairsSpeed            // -4
 6      dest.bottom += kClimbStairsSpeed
 7      whole.top    = dest.top                     // :334
 8      vNotClipped = dest.bottom - 29              // :336  how much is still visible
 9      if (vNotClipped < kGliderHigh):             // 20 - :337, outer test
10          if (vNotClipped <= 0):                  // :339  fully off the top
11              dest.top = dest.bottom              // :341  collapse to zero height
12              src.top  = src.bottom               // :342
13              mask.top = mask.bottom              // :343
14              takingTheStairs = true              // :344
15              if (twoPlayerGame):                 // :345
16                  if (onePlayerLeft):             // :347
17                      MoveRoomToRoom(the *surviving* glider, kAbove)   // :349-352
18                  else if (otherPlayerEscaped == kPlayerEscapedUpStairs):  // :356
19                      otherPlayerEscaped = kNoOneEscaped              // :358
20                      MoveRoomToRoom(thisGlider, kAbove)              // :359
21                  else:
22                      otherPlayerEscaped = kPlayerEscapedUpStairs  // -6  :363
23                      RefreshScoreboard(kEscapedTitleMode)            // :364
24                      FlagGliderInLimbo(thisGlider, true)             // :365
25              else:
26                  MoveRoomToRoom(thisGlider, kAbove)                  // :370
27          else:                                   // partly hidden
28              dest.top = dest.bottom - vNotClipped    // :374
29              src.top  = src.bottom  - vNotClipped    // :375
30              mask.top = mask.bottom - vNotClipped    // :376
```

The magic number **29** at line 8 is the y of the top of the staircase opening
(`kStairsTop` is 28, `GliderDefines.h:474`; the code uses 29 = one below it).
Lines 27–30 clip the sprite so the glider appears to walk up behind the wall.
Note the two-player handshake test is `== kPlayerEscapedUpStairs` — the code this
exit itself latches — **not** `== kNoOneEscaped`, and the waiting player is *not*
given `dontDraw = true` here (`FlagGliderInLimbo` does not set it either; the
limbo glider keeps rendering its last frame until something else hides it).
`RefreshScoreboard(kEscapedTitleMode)` at line 23 is what puts the "waiting for
the other player" banner on the scoreboard.

### 8.4 Mode 4 — `kGliderComingUp` / `FinishGliderUpStairs` (`:383`–`:427`)

`kVClimbStairsSpeed` = **−4** (`:385`), `kHClimbStairsSpeed` = **−4** (`:386`).
This is the arrival at the *bottom right* of the room above: the glider walks
diagonally up-and-left out of the staircase mouth.

```
 1  FinishGliderUpStairs(thisGlider):
 2      src = mask = gliderSrc[2]                   // always faces left  :389-390
 3      whole.bottom = dest.bottom                  // :392
 4      dest.top    += kVClimbStairsSpeed           // -4  :393
 5      dest.bottom += kVClimbStairsSpeed           // :394
 6      whole.top    = dest.top                     // :395
 7      whole.right  = dest.right                   // :397
 8      dest.left  += kHClimbStairsSpeed            // -4  :398
 9      dest.right += kHClimbStairsSpeed            // :399
10      whole.left   = dest.left                    // :400
11      wholeShadow.right = destShadow.right        // :402  the shadow moves too
12      destShadow.left  += kHClimbStairsSpeed      // :403
13      destShadow.right += kHClimbStairsSpeed      // :404
14      wholeShadow.left  = destShadow.left         // :405
15      hNotClipped = rightClip - dest.left         // :407
16      if (hNotClipped < kGliderWide):             // 48 - still emerging  :408
17          dest.right  = dest.left  + hNotClipped  // :410
18          src.right   = src.left   + hNotClipped  // :411
19          mask.right  = mask.left  + hNotClipped  // :412
20          destShadow.right = dest.right           // :413
21      else:                                       // fully emerged
22          if (frame == kWasBurning):  FlagGliderBurning(thisGlider)   // :417-418
23          else:                       FlagGliderNormal(thisGlider)    // :420
24          hVel = kHClimbStairsSpeed               // -4   :421
25          hDesiredVel = kHClimbStairsSpeed        // -4   :422
26          vVel = kVClimbStairsSpeed               // -4   :423
27          vDesiredVel = kVClimbStairsSpeed        // -4   :424
28          enteredRect = dest                      // :425
```

`rightClip` is a file-scope global (`Player.c:52`) set to
`GetUpStairsRightEdge()` at `GliderPRO/Sources/Modes.c:150` and `:536`.
`GetUpStairsRightEdge` (`GliderPRO/Sources/ObjectRects.c:1135`–`:1159`) defaults
to `kRoomWide` (512) and otherwise returns
`data.d.topLeft.h + srcRects[kDownStairs].right - 1` for the first `kDownStairs`
object in the room, i.e. **the right edge of the down-staircase graphic minus 1**
(the graphic is 160 wide — `GliderPRO/Sources/StructuresInit2.c:385`).

Lines 24–27 are important: the arrival **seeds momentum**, so you emerge from a
staircase already drifting up and to the left at 4 px/frame on each axis. The
seeded `hDesiredVel`/`vDesiredVel` are consumed by the *first* `MoveGlider` call
of the next frame, which ramps toward them and only then resets them to
`0`/`kGravity`. They do **not** affect sprite selection — `MoveGliderNormal`
reads only `facing`, `sliding` and `tipped` (§8.0). Their single effect is that
the first post-arrival frame ramps toward −4 instead of toward 0/+3, so `vVel`
stays at −4 for one extra frame rather than immediately decaying.

### 8.5 Mode 5 — `kGliderGoingDown` / `MoveGliderDownStairs` (`:431`–`:508`)

`kVDropStairsSpeed` = **4** (`:433`), `kHDropStairsSpeed` = **4** (`:434`).

```
 1  MoveGliderDownStairs(thisGlider):
 2      if (facing == kFaceLeft): src = mask = gliderSrc[2]   // :439-440
 3      else:                     src = mask = gliderSrc[0]   // :444-445
 4      // HORIZONTAL FIRST (:448-451), then the shadow, then vertical
 5      whole.left = dest.left                      // :448
 6      dest.left  += kHDropStairsSpeed             // +4  :449
 7      dest.right += kHDropStairsSpeed             // :450
 8      whole.right = dest.right                    // :451
 9      wholeShadow.left = destShadow.left          // :453
10      destShadow.left  += kHDropStairsSpeed       // :454
11      destShadow.right += kHDropStairsSpeed       // :455
12      wholeShadow.right = destShadow.right        // :456
13      whole.top = dest.top                        // :458
14      dest.top    += kVDropStairsSpeed            // +4  :459
15      dest.bottom += kVDropStairsSpeed            // :460
16      whole.bottom = dest.bottom                  // :461
17      hNotClipped = rightClip - dest.left         // :463
18      if (hNotClipped < kGliderWide):             // :464 outer test
19          if (hNotClipped <= 0):                  // :466  fully inside the stairwell
20              dest.right = dest.left              // :468  collapse to zero width
21              src.right  = src.left               // :469
22              mask.right = mask.left              // :470
23              destShadow.right = dest.right       // :471
24              takingTheStairs = true              // :472
25              if (twoPlayerGame):                 // :473
26                  if (onePlayerLeft):
27                      MoveRoomToRoom(the *surviving* glider, kBelow)   // :477-480
28                  else if (otherPlayerEscaped == kPlayerEscapedDownStairs):  // :484
29                      otherPlayerEscaped = kNoOneEscaped              // :486
30                      MoveRoomToRoom(thisGlider, kBelow)              // :487
31                  else:
32                      otherPlayerEscaped = kPlayerEscapedDownStairs  // -7  :491
33                      RefreshScoreboard(kEscapedTitleMode)            // :492
34                      FlagGliderInLimbo(thisGlider, true)             // :493
35              else:
36                  MoveRoomToRoom(thisGlider, kBelow)                  // :498
37          else:
38              dest.right = dest.left + hNotClipped    // :502
39              src.right  = src.left  + hNotClipped    // :503
40              mask.right = mask.left + hNotClipped    // :504
41              destShadow.right = dest.right           // :505
```

So going *down* the stairs the player walks right and down and is progressively
clipped on the right against `rightClip`.

### 8.6 Mode 6 — `kGliderComingDown` / `FinishGliderDownStairs` (`:512`–`:556`)

Same speeds (`+4`, `+4`, `:514`–`:515`). Arrival at the top-left of the room
below; the glider walks right and down out of the staircase.

```
 1  FinishGliderDownStairs(thisGlider):
 2      src = mask = gliderSrc[0]                   // always faces right  :518-519
 3      (move dest +4 horizontally :521-524, destShadow +4 horizontally :526-529,
 4       then dest +4 vertically :531-534, updating whole/wholeShadow as above)
 5      hNotClipped = dest.right - leftClip         // :536
 6      if (hNotClipped < kGliderWide):             // 48 - still emerging  :537
 7          dest.left = dest.right - hNotClipped    // :539  trims the LEFT edge
 8          src.left  = src.right  - hNotClipped    // :540
 9          mask.left = mask.right - hNotClipped    // :541
10          destShadow.left = dest.left             // :542
11      else:                                       // fully emerged
12          if (frame == kWasBurning): FlagGliderBurning else FlagGliderNormal  // :546-549
13          hVel = hDesiredVel = kHDropStairsSpeed  // +4  :550-551
14          vVel = vDesiredVel = kVDropStairsSpeed  // +4  :552-553
15          enteredRect = dest                      // :554
```

`leftClip` (`Player.c:52`) is `GetDownStairsLeftEdge()`
(`GliderPRO/Sources/Modes.c:567`), which defaults to 0 and otherwise returns
`data.d.topLeft.h + 1` of the first `kUpStairs` object
(`GliderPRO/Sources/ObjectRects.c:1163`–`:1187`).

### 8.7 Mode 7 — `kGliderFaceLeft` / `MoveGliderFaceLeft` (`:560`–`:573`)

```
 1  MoveGliderFaceLeft(thisGlider):
 2      src = mask = gliderSrc[frame]               // :562-563
 3      MoveGlider(thisGlider)                      // :565  <- still flies!
 4      frame--                                     // :567
 5      if (frame < kFirstAboutFaceFrame):          // < 18
 6          mode = kGliderNormal                    // :570
 7          facing = kFaceLeft                      // :571
```

Entry (`FlagGliderFaceLeft`, `GliderPRO/Sources/Modes.c:438`–`:444`) sets
`frame = kLastAboutFaceFrame` = 20 and the sprite to `gliderSrc[20]`. The
decrement and the exit test sit at the *end* of the same invocation that assigned
the sprite (`:567`–`:571`), so the sequence is sprites 20, 19, 18 over three
frames (0.1 s), and on the third of those frames — the one that renders sprite
18 — `frame` becomes 17 after `MoveGlider` has run and the mode flips to normal
with `facing = kFaceLeft`. There is no fourth invocation of the handler; the very
next frame is already `MoveGliderNormal`. Nor is the entry frame free: `GetInput`
calls `ToggleGliderFacing` (`GliderPRO/Sources/Input.c:308`) before `HandleGlider`
in the frame loop (`GliderPRO/Sources/Play.c:452`–`:461` two-player,
`:481`–`:487` one-player), so the frame on which the toggle is detected is already
handler call 1, not a preparation frame.

Crucially `MoveGlider` still runs, so physics continues during the tumble; and
because `mode != kGliderNormal`, `MoveGliderNormal` does *not* run, so
`facing`/`tipped`/`sliding` sprite selection is suspended — but
`CheckGliderInRoom` **does** include modes 7 and 8
(`GliderPRO/Sources/Interactions.c:691`–`:694`), so you can still die by hitting
a wall mid-tumble. (Do not look for the mode gate at `:698`–`:701`: that is the
burning-glider fade-out inside the ceiling branch, one of four identical copies at
`:698`, `:711`, `:727` and `:740`.)

### 8.8 Mode 8 — `kGliderFaceRight` / `MoveGliderFaceRight` (`:577`–`:590`)

Mirror image: entry sets `frame = kFirstAboutFaceFrame` = 18
(`GliderPRO/Sources/Modes.c:448`–`:454`), the handler increments, and exits when
`frame > kLastAboutFaceFrame` (> 20) with `facing = kFaceRight`. Sprites 18, 19,
20.

### 8.9 Mode 9 — `kGliderBurning` / `MoveGliderBurning` (`:203`–`:227`)

```
 1  MoveGliderBurning(thisGlider):
 2      frame++                                     // :205
 3      if (frame > 3): frame = 0                   // :206-207
 4      if (facing == kFaceLeft):
 5          src = mask = gliderSrc[25 + frame]      // :211-212  (25..28)
 6      else:
 7          src = mask = gliderSrc[21 + frame]      // :216-217  (21..24)
 8      wasMode--                                   // :220  the fuse
 9      if (wasMode <= 0):                          // :221
10          StartGliderFadingOut(thisGlider)        // :223
11          PlayPrioritySound(kFadeOutSound, kFadeOutPriority)  // :224
12      MoveGlider(thisGlider)                      // :226  runs unconditionally
```

Note there is **no** `wasMode = 0` reset when the fuse expires: `wasMode` simply
keeps going negative on any further burning frames. Nothing reads it after
`StartGliderFadingOut` has switched the mode to 2, so this is harmless — but a
port must not "helpfully" clamp it, and must not skip the final `MoveGlider` call
on the frame the fuse expires.

Entry (`FlagGliderBurning`, `GliderPRO/Sources/Modes.c:406`–`:434`) sets
`wasMode = kFramesToBurn` = **60**, `frame = 0`, all four velocities 0,
`tipped = false`, and plays `kCaughtFireSound` (`:410`). It also renormalises
**four** rect edges, not just the one that makes the glider 26 tall:
`dest.right = dest.left + kGliderWide` (`:412`),
`dest.top = dest.bottom - kGliderBurningHigh` (`:413`),
`destShadow.right = destShadow.left + kGliderWide` (`:414`) and
`destShadow.bottom = destShadow.top + kShadowHigh` (`:415`). Those are the same
four statements §9.1 calls load-bearing for `FlagGliderNormal`, which is the
sibling arm of the same `if`/`else` at five arrival sites (`Player.c:417`–`:420`,
`:546`–`:549`, `:879`–`:882`, `:917`–`:920`, `:949`–`:952`), and transcribing only
`:413` is the easy mistake because the other three are no-ops for the common case
of a floor vent catching a glider in flight. They are not no-ops when the glider
being set alight still has clipped rects: `kBurnIt` gates on nothing but
`mode != kGliderBurning && mode != kGliderFadingOut`
(`GliderPRO/Sources/Interactions.c:1356`–`:1358`) and `CheckForHotSpots` runs every
frame regardless of mode, so a glider still emerging from a staircase, a mailbox or
a duct — `dest.right`, `src.right` and `destShadow.right` all pulled in to a
part-width `hNotClipped`, e.g. `Player.c:410`–`:413` — can burn mid-emergence.
Without `:412` and `:414` that glider stays part-width for the rest of its life,
since `MoveGlider` only ever offsets `dest` afterwards, and `CopyMask` stretches
the full-width 48 x 26 flame sprite and the shadow into it. So a burning glider has
exactly **60 frames (2 s)** to reach
water/a vent, and it is forced to fly in its `facing` direction the whole time
(`GetInput` overrides all input for a burning glider —
`GliderPRO/Sources/Input.c:290`–`:296`).

### 8.10 Mode 10 — `kGliderTransporting` / `TransportGliderOut` (`:594`–`:653`)

```
 1  TransportGliderOut(thisGlider):
 2      frame--                                     // :598
 3      if (frame < 0):                             // :599
 4          // erase the glider from the screen immediately
 5          tempRect = whole;  QOffsetRect(&tempRect, playOriginH, playOriginV)  // :601-602
 6          CopyRectWorkToMain(&tempRect)                                        // :603
 7          tempRect = wholeShadow; QOffsetRect(...); CopyRectWorkToMain(&tempRect)  // :604-606
 8          dontDraw = true                                                      // :607
 9          if (twoPlayerGame):                                                  // :609
10              if (onePlayerLeft):                                              // :611
11                  TransportRoomToRoom(the *surviving* glider)                  // :613-616
12              else if (otherPlayerEscaped == kPlayerTransportedOut):           // :620
13                  otherPlayerEscaped = kNoOneEscaped                           // :622
14                  TransportRoomToRoom(thisGlider)     // second player: both go // :623
15              else:                                                            // :626
16                  otherPlayerEscaped = kPlayerTransportedOut  // = -10          // :627
17                  RefreshScoreboard(kEscapedTitleMode)                          // :628
18                  FlagGliderInLimbo(thisGlider, true)  // first player waits    // :629
19          else:
20              TransportRoomToRoom(thisGlider)                                   // :635
21      else:
22          src = mask = gliderSrc[fadeInSequence[frame] (+7 if facing left)]     // :640-651
```

The handshake is the *first* glider to reach `frame < 0` sets
`otherPlayerEscaped` and goes to limbo; the *second* one sees its own code
already latched, resets `otherPlayerEscaped` to `kNoOneEscaped`, and performs the
actual transit for both. Note the latched-code test is `== kPlayerTransportedOut`
(the code this exit writes), **not** `== kNoOneEscaped`. The same shape is used
by the stair, duct and mail exits with their own codes; see §16.5.

Entry is `StartGliderTransporting` (`GliderPRO/Sources/Modes.c:288`–`:330`), and
it does far more than seed the fade counter. It plays
`PlayPrioritySound(kTransOutSound, kTransOutPriority)` (`:294`) and applies the
same foil fix-up as every other exit — `DeckGliderInFoil` if the *old* mode was
18, `RemoveFoilFromGlider` if it was 19 (`:296`–`:299`). It then resolves the
destination link, in a block that is byte-for-byte the one in
`StartGliderMailingIn` at `:163`–`:171`:

```
whoLinked    = who->who;                                          // :301
transRoom    = masterObjects[whoLinked].roomLink;                 // :302
objLinked    = masterObjects[whoLinked].objectLink;               // :303
linkedToWhat = WhatAreWeLinkedTo(transRoom, objLinked);           // :304
GetObjectRect(&(*thisHouse)->rooms[transRoom].objects[objLinked], // :308
              &transRect);
```

That block is the load-bearing part of the routine and the easiest thing to lose
in a port. `transRoom`, `linkedToWhat` and `transRect` are process-lifetime
globals shared with the mailbox and duct paths (§23 item 10), and this is the only
place they are written on the transporter route — `StartGliderTransporting` is
reached solely from the `kTransportIt` case
(`GliderPRO/Sources/Interactions.c:1399`, `:1407`, `:1412`). When `frame` goes
negative, `TransportRoomToRoom` reads `transRoom` (`Transit.c:321`–`:323`) and
hands `linkedToWhat` to `ReadyGliderFromTransit` (`:332`), whose `kLinkedToOther`
arm positions the glider with `CenterRectInRect(&tempRect, &transRect)` (`:79`).
Omit the block and the failure is not a clean crash on the first transporter: the
globals still hold whatever the previous mailbox or duct trip left, so the player
lands in a plausible-looking wrong room, in the wrong arrival mode, centred on a
stale rect — a fault that is intermittent and house-dependent.

Only after that does the routine touch the glider: it forces `dest` to exactly
48 x 20 and `destShadow` to 48 x 9 (`:311`–`:314`, in case the glider was burning
and therefore 26 tall), sets `mode = kGliderTransporting` (`:315`),
`whole = dest` (`:316`) and `frame = kLastFadeSequence - 1` = 15 (`:317`), and
installs `gliderSrc[fadeInSequence[15]]` into both `src` and `mask`, with
`+ kLeftFadeOffset` when `facing == kFaceLeft` (`:318`–`:329`).

### 8.11 Mode 11 — `kGliderDuctingDown` / `MoveGliderDownDuct` (`:657`–`:750`)

`kVDropDuctSpeed` = **4** (`:659`). The glider centres itself horizontally on
the duct mouth at **1 pixel per frame** while descending 4 px/frame.

```
 1  MoveGliderDownDuct(thisGlider):
 2      src = mask = gliderSrc[0]  (or [2] facing left)
 3      // horizontal creep toward clip.left, 1 px per frame; dest and
 4      // destShadow move in lockstep (:674-697)
 5      if (dest.left < clip.left):                 // :674
 6          whole.left = dest.left
 7          dest.left++;  dest.right++
 8          whole.right = dest.right
 9          wholeShadow.left = destShadow.left      // :681
10          destShadow.left++;  destShadow.right++
11          wholeShadow.right = destShadow.right
12      else if (dest.left > clip.left):            // :686
13          whole.right = dest.right
14          dest.left--;  dest.right--
15          whole.left = dest.left
16          wholeShadow.right = destShadow.right    // :693
17          destShadow.left--;  destShadow.right--
18          wholeShadow.left = destShadow.left
19      // vertical
20      whole.top = dest.top
21      dest.top    += kVDropDuctSpeed              // +4
22      dest.bottom += kVDropDuctSpeed
23      whole.bottom = dest.bottom
24      vNotClipped = 315 - dest.top                // :704
25      if (vNotClipped <= 0):
26          (erase via CopyRectWorkToMain as in TransportGliderOut)
27          dontDraw = true
28          (two-player handshake with kPlayerDuckedOut = -11, else)
29          MoveDuctToDuct(thisGlider)
30      else if (vNotClipped < kGliderHigh):        // 20
31          src.bottom  = src.top  + vNotClipped
32          mask.bottom = mask.top + vNotClipped
33          dest.bottom = dest.top + vNotClipped
```

Each creep branch is eight statements, not four: the shadow rect is stepped by the
same pixel as `dest` and its `wholeShadow` bookkeeping mirrors `whole`'s. It is
easy to drop the second half of each branch as redundant, and it is not —
`RenderGlider` draws the shadow from `destShadow` on every frame in which the
global `shadowVisible` is set, and mode 11 falls through to the unclipped branch
that does so (`GliderPRO/Sources/Render.c:491`–`:495`), while `wholeShadow` is the
rect it feeds to `AddRectToWorkRects` (`Render.c:496`–`:498`) and the one
`MoveGliderDownDuct` later hands to `CopyRectWorkToMain` to erase the shadow on
the way out (`:712`–`:714`). Skip those four lines and the shadow simply stays
put while the glider slides to the duct centre, then fails to be erased. The
branches are `if` / `else if`, not `if` / `else`: at `dest.left == clip.left`
neither fires and the creep is finished.

**315** at line 24 is a bare literal. It is `kShadowTop + kShadowHigh`
(306 + 9) numerically, but semantically it is the floor plane the duct sits in;
`kSewerGrateTop` is 303 and `kFloorTransTop` is 302 (`GliderDefines.h:471`,
`:473`) and the floor transporter graphic is 15 tall
(`GliderPRO/Sources/StructuresInit2.c:389`), giving 302 + 15 = 317. Treat 315 as
an unexplained tuned constant and copy it verbatim.

`clip` is set at mode entry by `StartGliderDuctingDown`
(`GliderPRO/Sources/Modes.c:213`–`:242`) with
`clip.left = bounds->left + ((RectWide(bounds) - kGliderWide) / 2)` (`:238`–`:239`),
i.e. the glider aims for the horizontal centre of the duct. That routine also does
the rest of the exit bookkeeping, which is easy to overlook precisely because none
of it concerns mode 11's own motion:
`PlayPrioritySound(kTransOutSound, kTransOutPriority)` (`:219`); the foil fix-up,
`DeckGliderInFoil` if the old mode was 18 and `RemoveFoilFromGlider` if it was 19
(`:221`–`:224`, and reachable — the `kDuctItDown` guard excludes only
`kGliderDuctingDown` and `kGliderFadingOut`,
`GliderPRO/Sources/Interactions.c:1517`–`:1519`); `frame = 0` (`:236`); and the
link resolution into `transRoom`, `linkedToWhat` and `transRect` (`:226`–`:234`,
the identical block quoted in §8.10, with `GetObjectRect` at `:233`).

`transRect` is the one that will bite. It is the *only* input that positions the
far end of the trip: `ReadyGliderFromTransit`'s `kLinkedToCeilingDuct` arm centres
the arriving glider in it (`Transit.c:126`). Because it is a shared global (§23
item 10), a port that writes it on the mailbox path only will emerge from every
duct at whatever rect the previous mailbox trip pointed to, rather than at the
linked duct.

`StartGliderDuctingUp` (`Modes.c:246`–`:275`) is byte-identical to
`StartGliderDuctingDown` apart from the mode constant: sound at
`:252`, foil at `:254`–`:257`, the link block at `:259`–`:267` with `transRect` at
`:266`, `frame = 0` at `:269`, `mode = kGliderDuctingUp` written last at `:274`.
"Upward" describes only the mode; nothing in the routine is mirrored vertically, and
in particular `leftSought` at `:271`–`:272` is the same
`bounds->left + ((RectWide(bounds) - kGliderWide) / 2)` centring expression, with no
vertical counterpart to `StartGliderMailingIn`'s `clip.top`. The `frame = 0` is inert on this route — neither
duct handler ever reads `frame`, and the arrival runs `FlagGliderNormal`, which
zeroes it again, before `StartGliderDuctingIn` — but transcribe it anyway; §5.2
explains what dropping it costs.

### 8.12 Mode 12 — `kGliderDuctingUp` / `MoveGliderUpDuct` (`:754`–`:847`)

`kVRiseDuctSpeed` = **−4** (`:756`). Identical structure with the vertical
direction reversed — including the horizontal creep, whose two branches are
byte-identical to mode 11's, shadow statements and all (`dest` at `:773`–`:776`
and `:785`–`:788`, `destShadow` at `:778`–`:781` and `:790`–`:793`) — and

```
vNotClipped = dest.bottom - (kCeilingTransTop + 1)      // :801  = dest.bottom - 7
```

`kCeilingTransTop` = 6 (`GliderDefines.h:472`). Completion routes to
`MoveDuctToDuct` with the same `kPlayerDuckedOut` handshake. Neither duct handler
ever moves `destShadow` vertically, so the shadow stays on the floor plane and only
creeps sideways; mode 12 is where dropping that creep is most visible, because the
climb from floor level to `dest.bottom <= 7` takes on the order of seventy-five
frames and the horizontal creep finishes in the first few, leaving a stranded
shadow on screen for the whole ascent. Mode 11 only has to fall to `dest.top` 315,
so there the lag lasts a handful of frames.

### 8.13 Mode 13 — `kGliderDuctingIn` / `FinishGliderDuctingIn` (`:927`–`:956`)

Arrival out of a ceiling duct, dropping at `kVDropDuctSpeed` = 4 (`:936`).

```
 1  FinishGliderDuctingIn(thisGlider):
 2      // NOTE: no sprite assignment at all - src/mask are whatever the
 3      //       previous mode left (set by StartGliderDuctingIn's caller)
 4      if (dest.top == dest.bottom):               // :932  first frame, zero-height
 5          PlayPrioritySound(kTransInSound, kTransInPriority)   // :933
 6      whole.top = dest.top                        // :935
 7      dest.top    += kVDropDuctSpeed              // +4  :936
 8      dest.bottom += kVDropDuctSpeed              // :937
 9      whole.bottom = dest.bottom                  // :938
10      vNotClipped = dest.bottom - (kCeilingTransTop + 1)     // 7  :940
11      if (vNotClipped < kGliderHigh):             // 20 - still emerging  :941
12          dest.top = dest.bottom - vNotClipped    // :943
13          src.top  = src.bottom  - vNotClipped    // :944
14          mask.top = mask.bottom - vNotClipped    // :945
15      else:                                       // fully out
16          if (frame == kWasBurning): FlagGliderBurning(thisGlider)  // :949-950
17          else:                      FlagGliderNormal(thisGlider)   // :952
18          enteredRect = dest                      // :953
19          FlagStillOvers(thisGlider)              // :954
```

Lines 13 and 14 are destructive and self-referential: every frame they recompute
`src.top` from an `src.bottom` that this handler never writes. Mode 13 therefore
renders correctly only because the sprite it inherits is the full-height
`gliderSrc[0]` / `[2]` that `FlagGliderNormal` installed on the way in
(`GliderPRO/Sources/Modes.c:343`–`:349`, reached unconditionally from
`ReadyGliderFromTransit` at `Transit.c:72`, i.e. before the sole call to
`StartGliderDuctingIn` at `:124`). `StartGliderDuctingIn` itself writes only
`mode`, `whole` and `dontDraw` (`Modes.c:279`–`:284`) and assigns no sprite at all;
see the §9 row. The trap is to read that as an oversight and "fix" it by copying
`StartGliderTransporting` and seeding a `fadeInSequence` frame here — the resulting
short `src.bottom` clips the glider wrongly for the whole duct-in animation.

Line 16 matters, but not for the reason it appears to. Mode 13 is one of the arrival
handlers that honours the `frame == kWasBurning` sentinel, and yet the branch is
unreachable in the original: a burning glider is killed outright at every duct and
mailbox hot spot before it can enter (`wasMode = 0; StartGliderFadingOut(...)` at
`GliderPRO/Sources/Interactions.c:1511`–`:1516` and `:1544`–`:1549`), and the duct
starters zero `frame` on the way out anyway (`Modes.c:236`, `:269`). Transcribe the
test regardless: dropping the `frame = 0` writes as dead stores while keeping this
test is precisely how a port ends up with a glider dropping out of a ceiling duct on
fire with a fresh 60-frame fuse (§5.2). The `#define kVDropStairsSpeed 4` at `:929`
is dead — the
body uses `kVDropDuctSpeed` from `MoveGliderDownDuct`'s earlier `#define`.

Line 19 is unique to this handler: `FlagStillOvers`
(`GliderPRO/Sources/Interactions.c:1715`–`:1732`) pre-marks every hot spot the
glider currently overlaps as `stillOver = true`, so the glider does **not**
immediately re-trigger the duct it just came out of. A port that omits this will
create an infinite duct loop.

### 8.14 Mode 14 — `kGliderMailInLeft` / `MoveGliderInMailLeft` (`:960`–`:1047`)

`kHMailPullSpeed` = **4** (`:962`), `kVMailDropSpeed` = **2** (`:963`).

```
 1  MoveGliderInMailLeft(thisGlider):
 2      src = mask = gliderSrc[0] / [2]             // :967-976
 3      // settle downward into the slot, max 2 px per frame (:978-988)
 4      if (dest.top < clip.top):                   // :978
 5          drop = clip.top - dest.top              // :980
 6          if (drop > kVMailDropSpeed): drop = kVMailDropSpeed     // clamp to 2  :981-982
 7          whole.top = dest.top                    // :984
 8          dest.top += drop;  dest.bottom += drop  // :985-986
 9          whole.bottom = dest.bottom              // :987
10      // pulled right into the slot
11      whole.left = dest.left                      // :990
12      dest.left  += kHMailPullSpeed               // +4  :991
13      dest.right += kHMailPullSpeed               // :992
14      whole.right = dest.right                    // :993
15      wholeShadow.left = destShadow.left          // :995  the shadow is pulled too
16      destShadow.left  += kHMailPullSpeed         // :996
17      destShadow.right += kHMailPullSpeed         // :997
18      wholeShadow.right = destShadow.right        // :998
19      hNotClipped = clip.right - dest.left        // :1000
20      if (hNotClipped < kGliderWide):             // :1001 outer test
21          if (hNotClipped <= 0):                  // :1003
22              (erase whole + wholeShadow via CopyRectWorkToMain  :1005-1010)
23              dontDraw = true                     // :1011
24              if (twoPlayerGame):                 // :1012
25                  if (onePlayerLeft):    MoveMailToMail(the survivor)   // :1014-1019
26                  else if (otherPlayerEscaped == kPlayerMailedOut):     // :1023
27                      otherPlayerEscaped = kNoOneEscaped               // :1025
28                      MoveMailToMail(thisGlider)                       // :1026
29                  else:
30                      otherPlayerEscaped = kPlayerMailedOut  // -12  :1030
31                      RefreshScoreboard(kEscapedTitleMode)             // :1031
32                      FlagGliderInLimbo(thisGlider, true)              // :1032
33              else:
34                  MoveMailToMail(thisGlider)                           // :1037
35          else:
36              dest.right = dest.left + hNotClipped    // :1041
37              src.right  = src.left  + hNotClipped    // :1042
38              mask.right = mask.left + hNotClipped    // :1043
39              destShadow.right = dest.right           // :1044
```

`clip` comes from `StartGliderMailingIn`
(`GliderPRO/Sources/Modes.c:155`–`:177`): `clip = *bounds` then
`clip.top = bounds->bottom - RectTall(&dest)`, i.e. the glider settles to sit on
the bottom of the mailbox's hot rect.

### 8.15 Mode 15 — `kGliderMailOutLeft` / `FinishGliderMailingLeft` (`:851`–`:885`)

`kHPushMailSpeed` = **−4** (`:853`). The glider is pushed *left* out of the slot.

```
 1  FinishGliderMailingLeft(thisGlider):
 2      if (dest.left == dest.right):               // first frame, zero-width  :856
 3          PlayPrioritySound(kTransInSound, kTransInPriority)   // :857
 4      whole.right = dest.right                    // :859
 5      dest.left  += kHPushMailSpeed               // -4  :860
 6      dest.right += kHPushMailSpeed               // :861
 7      whole.left = dest.left                      // :862
 8      wholeShadow.right = destShadow.right        // :864  the shadow is pushed too
 9      destShadow.left  += kHPushMailSpeed         // :865
10      destShadow.right += kHPushMailSpeed         // :866
11      wholeShadow.left = destShadow.left          // :867
12      hNotClipped = clip.right - dest.left        // :869
13      if (hNotClipped < kGliderWide):             // 48  :870
14          dest.right = dest.left + hNotClipped    // :872
15          src.right  = src.left  + hNotClipped    // :873
16          mask.right = mask.left + hNotClipped    // :874
17          destShadow.right = dest.right           // :875
18      else:
19          if (frame == kWasBurning) FlagGliderBurning else FlagGliderNormal  // :879-882
20          enteredRect = dest                      // :883
```

The handler never assigns a sprite. Unlike modes 14 and 16, which re-pick
`gliderSrc[0]`/`[2]` from `facing` on every frame (`:967`–`:976`, `:1058`–`:1067`),
the only writes
to `src`/`mask` here are the width clamps at `:873`–`:874`; the left-facing sprite
was established once at mode entry by `StartGliderMailingOut`
(`GliderPRO/Sources/Modes.c:190`–`:193`, which also forces `facing = kFaceLeft`).
Nothing during the mode can change `facing`, because the handler runs neither
input nor `MoveGlider`, so re-picking the sprite each frame would be inert -- but a
porter who assumes this mode owns the sprite and therefore leaves the assignment
out of mode entry gets a glider that emerges from a left-hand mailbox wearing
whatever sprite mail-in left it with, facing backwards for the whole 12-frame
push-out.

Lines 8-11 are the load-bearing part. `destShadow.left` is never re-derived from
`dest.left` anywhere in normal flight -- `MoveGlider` only integrates it
incrementally (`:106`–`:109` / `:123`–`:126`) and `FlagGliderNormal` only rewrites
the far edges from the near ones (`GliderPRO/Sources/Modes.c:338`–`:339`) -- so a
port that pushes `dest` out of the mailbox without pushing `destShadow` with it
leaves the shadow pinned at the mailbox slot while `dest` travels 48 px (12 frames
x 4 px), and the shadow stays 48 px displaced from the glider for the rest of its
life in that room: 48 px to the right after a mail-out-left, 48 px to the left
after a mail-out-right. Line 17 is the shadow's half of the emergence clip: it
keeps `destShadow.right` pinned to the already-clipped `dest.right` so the shadow
widens in lockstep with the glider instead of snapping to full width, and it must
run *after* line 14 because it reads the clipped value, not the unclipped one (the
mirror at `:913` likewise follows `:910`).

Transcribe lines 4-11 in statement order rather than "intent" order.
`wholeShadow.right` is read from the *old* `destShadow.right` before the shadow is
advanced (`:864` precedes `:865`–`:866`) and `wholeShadow.left` from the *new*
value after (`:867`) -- the same read-before-update sweep idiom as `whole` at
`:859`–`:862`. A port that moves `destShadow` first and then unions the pre- and
post-move rects, or that reorders the four assignments, computes a different erase
rect. In the C the result is exactly right: `ReadyGliderFromTransit` seeds
`destShadow` as a zero-width rect at the slot and copies it into `wholeShadow`
(`GliderPRO/Sources/Transit.c:101`–`:104`, and `:117`–`:120` for the right
mailbox), and thereafter each frame's `wholeShadow` spans from the new
`destShadow.left` to the previous `destShadow.right`, which is precisely the ground
the widening shadow covers. Drop lines 8-11 and nothing writes `wholeShadow` at all
for the duration of the mode, so it stays that zero-width seed rect: the shadow's
dirty rect covers nothing, and the stale shadow smears until the first `MoveGlider`
call rewrites `wholeShadow`.

### 8.16 Mode 16 — `kGliderMailInRight` / `MoveGliderInMailRight` (`:1051`–`:1138`)

`kHMailPullRtSpeed` = **−4** (`:1053`), `kVMailDropSpeed` = 2 (`:1054`).
Mirror of §8.14 with `hNotClipped = dest.right - clip.left` (`:1091`), the shadow
carried along by the mirrored sweep at `:1086`–`:1089` (`wholeShadow.right` read
from the old `destShadow.right` first, `wholeShadow.left` written last), and the
clipped branch working from the far edge inwards: `dest.left = dest.right - hNotClipped`,
`src`/`mask` likewise, `destShadow.left = dest.left` (`:1132`–`:1135`).

### 8.17 Mode 17 — `kGliderMailOutRight` / `FinishGliderMailingRight` (`:889`–`:923`)

`kHPushMailRtSpeed` = **4** (`:891`). Mirror of §8.15 with
`hNotClipped = dest.right - clip.left` (`:907`). The mirroring is exact, so the
shadow block matters here for the same reason: `wholeShadow.left = destShadow.left;
destShadow.left += kHPushMailRtSpeed; destShadow.right += kHPushMailRtSpeed;
wholeShadow.right = destShadow.right;` (`:902`–`:905`), with
`destShadow.left = dest.left` in the clipped branch (`:913`, after the `dest.left`
clip at `:910`). Like §8.15 this handler does not assign a sprite -- `facing` and
`gliderSrc[0]` are set once by `StartGliderMailingOut`
(`GliderPRO/Sources/Modes.c:197`–`:200`) -- and its only `src`/`mask` writes are the
clamps at `:911`–`:912`.

### 8.18 Mode 18 — `kGliderGoingFoil` / `MoveGliderFoilGoing` (`:1170`–`:1198`)

```
 1  MoveGliderFoilGoing(thisGlider):
 2      frame++                                     // :1172
 3      if (frame > 8):
 4          FlagGliderNormal(thisGlider)            // :1175
 5      else if (frame < 5):
 6          src = mask = gliderSrc[10 - frame (+7 if facing left)]   // 9,8,7,6
 7      else:
 8          DeckGliderInFoil(thisGlider)            // 7,8,9,10 from the foil atlas
 9      MoveGlider(thisGlider)                      // :1197  physics continues
```

`DeckGliderInFoil` (`:1142`–`:1166`):

```
 1  DeckGliderInFoil(thisGlider):
 2      showFoil = true                             // :1144  GLOBAL
 3      if (twoPlayerGame):                         // :1146
 4          LoadGraphic(kGliderFoilPictID)  into glidSrcMap   // 3976
 5          LoadGraphic(kGliderFoil2PictID) into glid2SrcMap  // 3963
 6      src = mask = gliderSrc[frame + 2 (+kLeftFadeOffset if facing left)]
```

In a **one-player** game the atlas is *not* reloaded; instead `glid2SrcMap`
already holds the foil graphic (loaded at `NewGame` time,
`GliderPRO/Sources/Play.c:132`–`:135`) and `RenderGlider` picks between the two
maps with `if ((!twoPlayerGame) && showFoil)`
(`GliderPRO/Sources/Render.c:507`–`:512`). In a two-player game both players
share `showFoil`, so both atlases have to be swapped in place. **This is the
single most confusing piece of glider rendering in the code base.** See §18.3.

### 8.19 Mode 19 — `kGliderLosingFoil` / `MoveGliderFoilLosing` (`:1230`–`:1256`)

Identical to §8.18 but calls `RemoveFoilFromGlider` (`:1202`–`:1226`) at
frame ≥ 5, which sets `showFoil = false` and, in a two-player game, reloads
PICT 3999 into `glidSrcMap` and PICT 3974 into `glid2SrcMap`.

### 8.20 Mode 20 — `kGliderShredding` / `MoveGliderShredding` (`:1260`–`:1319`)

`kDropShredSlow` = **1** (`:1262`), `kDropShredFast` = **4** (`:1263`). This is
the most intricate handler because `frame` changes meaning halfway through.

```
 1  MoveGliderShredding(thisGlider):
 2      if (frame > 0):                             // frame is a TARGET Y
 3          src = mask = gliderSrc[0] / [2]         // :1268-1277  by facing, EVERY frame
 4          vNotClipped = frame - dest.top          // :1279  distance to shredder
 5          if (vNotClipped < kGliderHigh):         // 20 - already being eaten
 6              whole.top = dest.top
 7              dest.top    += kDropShredSlow       // +1 px/frame  :1283
 8              dest.bottom += kDropShredSlow       // :1284
 9              whole.bottom = dest.bottom
10              shadowVisible = false               // :1286  GLOBAL, hide the shadow
11              PlayPrioritySound(kShredSound, kShredPriority)   // every frame
12          else:                                   // still falling toward it
13              whole.top = dest.top
14              dest.top    += kDropShredFast       // +4 px/frame
15              dest.bottom += kDropShredFast
16              whole.bottom = dest.bottom
17          vNotClipped = frame - dest.top          // :1297 recompute after moving
18          if (vNotClipped > 0):
19              if (vNotClipped < kGliderHigh):
20                  src.bottom  = src.top  + vNotClipped     // :1308
21                  mask.bottom = mask.top + vNotClipped
22                  dest.bottom = dest.top + vNotClipped
23          else:
24              AddAShreddedGlider(&dest)           // :1302  spawn the confetti
25              frame = kShredderCountdown          // :1303  = -68
26      else:                                       // frame <= 0: death delay
27          frame++                                 // :1315
28          if (frame >= 0):
29              OffAMortal(thisGlider)              // :1317
```

Entry (`FlagGliderShredding`, `GliderPRO/Sources/Modes.c:366`–`:402`) sets
`frame = bounds->bottom - 3` — the y at which the glider is fully consumed — and
`dest.left = bounds->left + 36` (the glider is dragged to a fixed x inside the
shredder), width 48, height 20 measured from the top, all four velocities 0,
sprite `gliderSrc[2]`/`[0]` by facing (same function, `:386`–`:395`), and plays
`kCaughtFireSound`. Do not read that as the only place the sprite is chosen: the
handler re-asserts the same choice from `facing` on every frame of the grind
(`Player.c:1268`–`:1277`, pseudocode line 3), which matters on exactly one frame —
see below.

So the total shredding animation is: fall at 4 px/frame while the distance to
`frame` is 20 px or more, then grind down 1 px/frame — `kShredSound` on every one
of those frames, the sprite clipped to `vNotClipped` px tall — then
`AddAShreddedGlider` and a **68-frame (2.27 s) pause** before the life is
actually lost. The grind is *not* a fixed 20 frames and can never be 20: it lasts
exactly as many frames as `vNotClipped` holds when the slow branch is first
taken, which is always 16 to 19. Two facts pin it there. The initial distance is
`(bounds->bottom - 3) - dest.top` (`GliderPRO/Sources/Modes.c:400`, with
`dest.bottom = dest.top + kGliderHigh` at `:371`), and the only caller of
`FlagGliderShredding` is the `kShredIt` case
(`GliderPRO/Sources/Interactions.c:1339`), which fires only when `GliderInRect`
succeeds (`:1326`, definition `:140`–`:147`); that confines `dest.top` to
`[bounds.top, bounds.bottom - 20]` for the shredder's 40-px active rect
(`GliderPRO/Sources/ObjectRects.c:19`, `:942`) and so the initial distance to 17
to 37. The fast branch then subtracts 4 only while the distance is 20 or more, so
the slow branch is always entered at 16 to 19 — 17 to 19 when the glider was
caught close enough that no fast frame ran at all. A porter who hard-codes a
20-frame grind timer instead of driving the loop off `vNotClipped` spawns the
confetti, and starts the 68-frame countdown, one to four frames late.

The per-frame sprite reset at `:1268`–`:1277` looks redundant, and on every frame
but one it is: `src.top` is never written inside the handler, so
`src.bottom = src.top + vNotClipped` (`:1308`) lands on the same value whether or
not the rect was just restored, and `facing` cannot change mid-shred because every
assignment to it belongs to some other mode (`:571` and `:588` in the about-face
handlers; `GliderPRO/Sources/Modes.c:190`, `:197`, `:526`, `:557`). The frame
where it matters is the terminal one. Once clipping has engaged, every frame ends
with `dest.bottom` pinned to `frame`, so when the recomputed distance finally
reaches 0 or less (`:1300`) the clipping `else` at `:1305`–`:1310` does not run:
the C is left holding a full 20-px-tall `src`/`mask` against a `dest` that the
previous frame pinned at `dest.bottom = frame` and this frame's `+1`
(`:1283`–`:1284`) has left just 1 px tall, while a port that omits the reset is
still holding the previous frame's 1-px `src`. Nothing touches any of those rects
again — `FlagGliderShredding` never sets `dontDraw` and the `frame <= 0` branch
only counts up — so `RenderGlider` keeps drawing (it early-returns on `dontDraw`
alone, `GliderPRO/Sources/Render.c:457`) and passes `src`, `mask` and `dest`
straight to `CopyMask` (`:505`–`:524`), which scales when source and destination
differ in size. The original therefore squashes a 48-by-20 source *and its mask*
into a 48-by-1 line; the port blits the glider's top row 1:1. The wrong line then
persists frozen for the terminal frame plus all 68 countdown frames
(`kShredderCountdown` = −68, `:17`), about 2.3 s at 30 Hz, and because the mask is
squashed too the two can differ by a solid 48-px line versus almost nothing rather
than by a shade. There is no benign path: the initial
distance is always at least 17, so the clipping branch has always run at least
once by the time the terminal frame arrives.

`AddAShreddedGlider` (`GliderPRO/Sources/DynamicMaps.c:726`–`:746`):

```
 1  AddAShreddedGlider(bounds):
 2      if (numShredded > kMaxShredded): return     // kMaxShredded = 4
 3      shreds[numShredded].bounds.left   = bounds->left + 4
 4      shreds[numShredded].bounds.right  = shreds[numShredded].bounds.left + 40
 5      shreds[numShredded].bounds.top    = bounds->top + 14
 6      shreds[numShredded].bounds.bottom = shreds[numShredded].bounds.top
 7      shreds[numShredded].frame = 0
 8      numShredded++
```

Note the off-by-one guard: `> kMaxShredded` rather than `>=`, so up to **5**
shreds can exist in a `kMaxShredded`-sized array. Whether the array is
oversized is a question for the dynamic-maps document; for the player, the
relevant fact is that `OffAMortal` calls `RemoveShreds()` if `numShredded > 0`
(`GliderPRO/Sources/Player.c:1489`–`:1490`).

`RenderShreds` (`GliderPRO/Sources/Render.c:559`–`:612`) grows the shred rect's
bottom by 1 px per frame until `high >= 35`, then sets `frame = 1` and thereafter
moves the confetti down 4 px/frame for 20 frames.

### 8.21 Mode 21 — `kGliderInLimbo` (no handler)

`GliderPRO/Sources/Player.c:1423`–`:1424` is an empty `case`. A glider in limbo is
frozen — no handler runs, so every rect keeps whatever the previous mode's last
frame left in it — but it is *not* automatically invisible. `FlagGliderInLimbo`
never touches `dontDraw`, and `RenderGlider` gates on that flag alone
(`GliderPRO/Sources/Render.c:457`–`:458`), so whether the parked glider is hidden
depends entirely on the mode it came from. This mode exists only for the
two-player "one player has left the room and is waiting" state.

`FlagGliderInLimbo` (`GliderPRO/Sources/Modes.c:458`–`:468`):

```
 1  FlagGliderInLimbo(thisGlider, sayIt):
 2      wasMode = mode                              // :460
 3      mode = kGliderInLimbo                       // :461
 4      if (sayIt && saidFollow < 3):               // :462
 5          PlayPrioritySound(kFollowSound, kFollowPriority)   // :464
 6          saidFollow++                            // :465
 7      firstPlayer = thisGlider->which             // :467
```

`saidFollow` (`GliderPRO/Sources/Modes.c:13`) is reset to 0 by `NewGame`
(`GliderPRO/Sources/Play.c:116`), so the "follow me" prompt plays at most
**three times per game**.

Line 7 is counter-intuitive and worth pausing on: `firstPlayer` names the glider
that has just *stopped* and is waiting, not the one still flying. That is what makes
the `TagGliderIdle` calls in `MoveRoomToRoom`
(`GliderPRO/Sources/Transit.c:292`–`:295`) freeze the *other* glider — the trailing
one that actually triggered the room change.

On the transporter, duct and mail paths a limbo glider genuinely is invisible, but
only because the departing handler set the flag a few lines before it called
`FlagGliderInLimbo`: `Player.c:607` (transport out), `:715` (duct down), `:812`
(duct up), `:1011` and `:1102` (mail in). `OffAMortal` sets it too, immediately
after the call (`Player.c:1505`–`:1506`). On the two staircase paths
(`FlagGliderInLimbo` at `Player.c:365` and `:493`) `dontDraw` stays **false** and
`RenderGlider` runs every frame of the wait. The glider's own blit is a visual
no-op there because `dest`, `src` and `mask` were collapsed a few lines earlier —
`dest.top = dest.bottom` at `:341` on the way up, `dest.right = dest.left` at
`:468` on the way down — not because the mode suppressed the draw. Two things
therefore survive the freeze and a port must keep both. The shadow: the up path never
collapses `destShadow` (the down path does, at `:471`), so while `shadowVisible` is
set a stationary shadow goes on being blitted at the last position for the whole
wait. And the dirty-rect bookkeeping: `whole` and `dest` are still handed to
`AddRectToWorkRects`/`AddRectToBackRects` unconditionally
(`GliderPRO/Sources/Render.c:526`–`:529`), and `wholeShadow`/`destShadow` likewise
inside the `shadowVisible` guard (`:496`–`:499`). A render function that
early-returns on mode 21 loses the shadow and leaves the background unrestored.

`UndoGliderLimbo` (`GliderPRO/Sources/Modes.c:472`–`:480`) restores `mode = wasMode`
and clears `dontDraw`, and it is deliberately not a clean inverse of the entry
function. The `dontDraw = false` at `:479` sits *outside* the
`if (mode == kGliderInLimbo)` test at `:477` but *inside* the dead-player early
return at `:474`–`:475`. So it un-hides any glider it is handed, in limbo or not,
while doing nothing whatsoever for the dead player of a one-player-left game.
Tidying the assignment into the `if` is the obvious refactor and it is wrong.

### 8.22 Mode 22 — `kGliderIdle` / `HandleIdleGlider` (`:1323`–`:1331`)

```
 1  HandleIdleGlider(thisGlider):
 2      hVel--                                      // :1325  hVel IS the countdown
 3      if (hVel <= 0):
 4          mode = wasMode                          // :1328
 5          dontDraw = false                        // :1329
```

Entry (`TagGliderIdle`, `GliderPRO/Sources/Modes.c:631`–`:639`):

```
 1  TagGliderIdle(thisGlider):
 2      if (twoPlayerGame && onePlayerLeft && thisGlider->which == playerDead):
 3          return                                  // :633-634  dead player stays dead
 4      wasMode = mode
 5      mode = kGliderIdle
 6      hVel = 30                                   // :638  "used for 30 frame delay"
```

**`hVel` is repurposed as a frame counter.** A port that keeps velocity in a
float or in a separate struct must replicate the aliasing, because when the idle
ends `hVel` is exactly 0 and the glider starts from rest.

Note that `TagGliderIdle` does *not* set `dontDraw`, and neither do three of its
four callers. Only `NewGame` does, at `GliderPRO/Sources/Play.c:190`, for the
second glider's initial spawn; the three `Transit.c` calls
(`GliderPRO/Sources/Transit.c:146`, `:293`, `:295`) leave the flag exactly as they
found it, and in `MoveRoomToRoom` that means **false**, because the immediately
preceding `UndoGliderLimbo(&theGlider); UndoGliderLimbo(&theGlider2);`
(`Transit.c:162`–`:163`) has just executed `dontDraw = false` (`Modes.c:479`). An
idle glider is therefore *drawn*, frozen at its new-room entry position, for all 30
frames, and it is drawn at full size: a side transition translates the existing rect
by a room width (`OffsetGlider`, `Player.c:1443`–`:1480`) and a stair transition
rebuilds it from the sprite (`ReadyGliderForTripUpStairs` / `…DownStairs`,
`Modes.c:537`–`:543` and `:568`–`:574`), so nothing is left collapsed. A port that keys
visibility off mode 22 makes the second player vanish for the whole 30-frame freeze
after every two-player room change. `HandleIdleGlider` clears `dontDraw` when the
countdown reaches 0 whatever set it (`Player.c:1329`), and that is what un-hides the
`NewGame` spawn.

### 8.23 Mode 23 — `kGliderTransportingIn` / `TransportGliderIn` (`:261`–`:287`)

Byte-for-byte identical to `FadeGliderIn` (§8.1) except the sound is
`kTransInSound`/`kTransInPriority` instead of `kFadeInSound`
(`GliderPRO/Sources/Player.c:263`–`:264`). Same 16-frame fade table, same
`FlagGliderNormal` + `enteredRect = dest` on completion.
---

## 9. Mode-entry functions (`Modes.c`)

Every state transition goes through one of these. The table gives the exact
field writes; the numbers in the "Line" column are `GliderPRO/Sources/Modes.c`.

| Function | Line | Sets `mode` | Other writes |
| --- | --- | --- | --- |
| `StartGliderFadingIn` | 25 | 1 | `if (foilTotal <= 0) showFoil = false;` `whole = dest;` `frame = 0;` `dontDraw = false;` sprite `fadeInSequence[0]` (+7 left) |
| `StartGliderTransportingIn` | 50 | 23 | same as above |
| `StartGliderFadingOut` | 75 | 2 | returns immediately if already mode 2; `DeckGliderInFoil` if mode was 18, `RemoveFoilFromGlider` if mode was 19; if `RectTall(dest) > kGliderHigh` adds the old rect to the dirty lists (and the mirror rect at `playOrigin−20,−16`) and then forces `dest.right = dest.left + kGliderWide; dest.top = dest.bottom - kGliderHigh;`; `whole = dest;` `frame = kLastFadeSequence − 1` (15) |
| `StartGliderGoingUpStairs` | 120 | 3 | `DeckGliderInFoil` if mode was 18, `RemoveFoilFromGlider` if mode was 19 (`:122`–`:125`); `frame = (mode == kGliderBurning) ? kWasBurning : 0` (`:127`–`:130`, *after* the foil fix-up has already read the old `frame`) |
| `StartGliderGoingDownStairs` | 137 | 5 | same foil fix-up (`:139`–`:142`); same `frame` rule (`:144`–`:147`); `rightClip = GetUpStairsRightEdge()` |
| `StartGliderMailingIn` | 155 | 14 or 16 (by caller) | plays `kTransOutSound`; `transRoom`, `objLinked`, `linkedToWhat` and `transRect` from the linked object via `GetObjectRect`; `frame = 0`; `clip = *bounds`; `clip.top = bounds->bottom − RectTall(dest)` |
| `StartGliderMailingOut` | 181 | 15 or 17 | `DeckGliderInFoil` if mode was 18, `RemoveFoilFromGlider` if mode was 19 (`:183`–`:186`); `facing` and `mode` chosen by `linkedToWhat`; sprite `[2]` (left slot) or `[0]`; `hVel = vVel = hDesiredVel = vDesiredVel = 0`; `tipped = false`; `dontDraw = false` |
| `StartGliderDuctingDown` | 213 | 11, written **last** (`:241`) | plays `kTransOutSound`; `DeckGliderInFoil` if mode was 18, `RemoveFoilFromGlider` if mode was 19 (`:221`–`:224`, both reading the *old* mode); `transRoom`, `objLinked`, `linkedToWhat` and `transRect` from the linked object via `GetObjectRect` (`:226`–`:234`, `GetObjectRect` at `:233`); `frame = 0` (`:236`); `clip = *bounds` (`:237`) with `clip.left = bounds->left + ((RectWide(bounds) − kGliderWide) / 2)` (`:238`–`:239`) |
| `StartGliderDuctingUp` | 246 | 12, written **last** (`:274`) | byte-identical to `StartGliderDuctingDown` bar the mode constant — nothing is mirrored vertically: `kTransOutSound` at `:252`, foil transitions at `:254`–`:257`, the link block including `transRect` at `:259`–`:267` (`GetObjectRect` at `:266`), `frame = 0` at `:269`, the *same* `clip.left` expression at `:271`–`:272` |
| `StartGliderDuctingIn` | 279 | 13 | `whole = dest`; `dontDraw = false` (`:281`–`:283`). No sprite assignment and no `frame` write — mode 13 inherits `frame` from the mode-11/12 starter, which is why the `kWasBurning` test at `Player.c:949` sees 0 |
| `StartGliderTransporting` | 288 | 10 (`:315`) | plays `kTransOutSound` (`:294`); `DeckGliderInFoil` if mode was 18, `RemoveFoilFromGlider` if mode was 19 (`:296`–`:299`); `transRoom`, `objLinked`, `linkedToWhat` and `transRect` from the linked object via `GetObjectRect` (`:301`–`:309`, `GetObjectRect` at `:308`) — the same block as `StartGliderMailingIn`, and the *only* place the transporter route writes those globals; forces `dest` to 48 x 20 and `destShadow` to 48 x 9 (`:311`–`:314`); `whole = dest` (`:316`); `frame = kLastFadeSequence − 1` = 15 (`:317`); sprite `fadeInSequence[15]` (+`kLeftFadeOffset` if facing left) into both `src` and `mask` (`:318`–`:329`) |
| **`FlagGliderNormal`** | 334 | 0 | `dest.right = dest.left + kGliderWide; dest.bottom = dest.top + kGliderHigh;` `destShadow.right = destShadow.left + kGliderWide; destShadow.bottom = destShadow.top + kShadowHigh;` (does **not** write `whole`, `wholeShadow`, `destShadow.left` or `destShadow.top`); sprite `[2]`/`[0]` by facing; `hVel = vVel = hDesiredVel = vDesiredVel = 0`; `tipped = ignoreLeft = ignoreRight = ignoreGround = dontDraw = false`; `frame = 0`; `shadowVisible = IsShadowVisible()` |
| `FlagGliderShredding` | 366 | 20 | `kCaughtFireSound`; `dest.left = bounds->left + 36`, width 48, height 20 from the top; extends `whole`/`wholeShadow` toward whichever side the glider jumped to (tested with `dest.left > whole.left`); `destShadow` follows `dest.left`; sprite `[2]`/`[0]`; velocities 0; `frame = bounds->bottom − 3`; `tipped = false` |
| `FlagGliderBurning` | 406 | 9 | `kCaughtFireSound` (`:410`); **four** rect writes, the same normalisation as `FlagGliderNormal` but with `dest` anchored at the *bottom* — `dest.right = dest.left + kGliderWide` (`:412`), `dest.top = dest.bottom − kGliderBurningHigh` (26, `:413`), `destShadow.right = destShadow.left + kGliderWide` (`:414`), `destShadow.bottom = destShadow.top + kShadowHigh` (`:415`); sprite `[25]`/`[21]`; velocities 0; `frame = 0`; `wasMode = kFramesToBurn` (60); `tipped = false` |
| `FlagGliderFaceLeft` | 438 | 7 | `frame = kLastAboutFaceFrame` (20); sprite `[20]` |
| `FlagGliderFaceRight` | 448 | 8 | `frame = kFirstAboutFaceFrame` (18); sprite `[18]` |
| `FlagGliderInLimbo` | 458 | 21 | `wasMode = mode`; optional `kFollowSound` (max 3 per game); `firstPlayer = which` |
| `UndoGliderLimbo` | 472 | `wasMode`, but **only if** currently mode 21 (`:477`–`:478`) | returns for the dead player of a `onePlayerLeft` game (`:474`–`:475`); `dontDraw = false` (`:479`) sits outside the mode-21 test, so it fires for any glider that gets past the early return |
| `ToggleGliderFacing` | 484 | 7 or 8 | **returns unless `mode == kGliderNormal`**; calls `FlagGliderFaceRight` if `facing == kFaceLeft`, else `FlagGliderFaceLeft` |
| `InsureGliderFacingRight` | 497 | 8 (maybe) | no-op for the dead player in a two-player game with `onePlayerLeft` (`:499`–`:500`); no-op if `mode == kGliderBurning`; else `FlagGliderFaceRight` if currently facing left |
| `InsureGliderFacingLeft` | 508 | 7 (maybe) | mirror |
| `ReadyGliderForTripUpStairs` | 519 | 4 | `facing = kFaceLeft`; sprite `[2]`; velocities 0; `rightClip = GetUpStairsRightEdge()`; `dest = src` then `ZeroRectCorner(&dest)` then `QOffsetRect(&dest, rightClip, kVGliderAppearsComingUp /*100*/)`; `whole = dest` (`:540`); `destShadow.left/.right = dest.left/.right`; `wholeShadow = destShadow` (`:541`–`:543`); then calls `FinishGliderUpStairs(thisGlider)` once immediately. Also returns early for the dead player in a two-player game (`:523`–`:524`) |
| `ReadyGliderForTripDownStairs` | 550 | 6 | `facing = kFaceRight`; sprite `[0]`; `leftClip = GetDownStairsLeftEdge()`; `dest` offset to `(leftClip − kGliderWide, kVGliderAppearsComingDown /*100*/)`; `whole = dest` (`:571`); `destShadow.left/.right = dest.left/.right`; `wholeShadow = destShadow` (`:572`–`:574`); then calls `FinishGliderDownStairs` once. Also returns early for the dead player in a two-player game (`:554`–`:555`) |
| `StartGliderFoilGoing` | 581 | 18 | returns if mode is already 18 or 21; `QuickFoilRefresh(false)`; `whole = dest`; `frame = 0`; sprite `[10 − frame]` (+7 left) |
| `StartGliderFoilLosing` | 605 | 19 | returns if mode is already 19 or 21; `QuickFoilRefresh(false)`; `kFizzleSound`; `whole = dest`; `frame = 0`; sprite `[10 − frame]` |
| `TagGliderIdle` | 631 | 22 | returns for the dead player; `wasMode = mode`; `hVel = 30` |

The foil fix-up that keeps recurring in that table is one copy-pasted prologue,

```
if (thisGlider->mode == kGliderGoingFoil)
    DeckGliderInFoil(thisGlider);
else if (thisGlider->mode == kGliderLosingFoil)
    RemoveFoilFromGlider(thisGlider);
```

and it opens **seven** of these functions: `StartGliderFadingOut` (`:82`–`:85`),
`StartGliderGoingUpStairs` (`:122`–`:125`), `StartGliderGoingDownStairs`
(`:139`–`:142`), `StartGliderMailingOut` (`:183`–`:186`),
`StartGliderDuctingDown` (`:221`–`:224`), `StartGliderDuctingUp` (`:254`–`:257`)
and `StartGliderTransporting` (`:296`–`:299`). The four arrival starters —
`StartGliderFadingIn` (`:25`), `StartGliderTransportingIn` (`:50`),
`StartGliderMailingIn` (`:155`) and `StartGliderDuctingIn` (`:279`) — genuinely
lack it, so the pattern is "every routine that can be entered *out of* the 9-frame
foil dissolve commits that dissolve first". It is reachable in ordinary play:
`CheckForHotSpots` runs every frame from `HandleInteraction`
(`GliderPRO/Sources/Interactions.c:1693`) with no mode filter, and
`MoveGliderFoilGoing` keeps calling `MoveGlider` (`Player.c:1197`), so a glider
touching a staircase, mailbox, duct or transporter hot spot during modes 18/19 is
routine.

Two details a transcription has to preserve. First, the prologue reads the glider
*before* the function rewrites `frame`, so `DeckGliderInFoil`/`RemoveFoilFromGlider`
index the atlas with the stale dissolve frame — `src = mask =
gliderSrc[frame + 2]` (+`kLeftFadeOffset` facing left), `frame` in 0..8 giving
indices 2..10 (`Player.c:1154`–`:1165`, `:1214`–`:1225`). In
`StartGliderGoingUpStairs` the `frame` reset sits at `:127`–`:130`, after the
prologue has already consumed the old value. Second, whose sprite survives varies:
`StartGliderMailingOut` (`:192`–`:193` / `:199`–`:200`) and
`StartGliderTransporting` (`:318`–`:329`) overwrite
`src`/`mask` a few lines later, so only the global `showFoil` and the two-player atlas
reload outlive the prologue there, whereas in both stairs starters and both ducting
starters the prologue's write is the last sprite write the function makes.

Dropping the prologue is a rendering and save-state bug rather than a physics one.
`showFoil` is a global (`GliderPRO/Headers/GliderVars.h:58`); `RenderGlider`
selects the atlas with `if ((!twoPlayerGame) && (showFoil))`
(`GliderPRO/Sources/Render.c:507`) and
`DrawReflection` tests `if (showFoil)` with no `twoPlayerGame` guard (`:163`), so a
stale value draws the glider plain while foiled or foil-clad after the foil is
gone, for the remainder of the foil supply; it is also written into save games
(`GliderPRO/Sources/SavedGames.c:106`, `:335`), and in a two-player game the wrong PICTs stay loaded
in `glidSrcMap`/`glid2SrcMap`. `showFoil` is never consulted for damage immunity —
every read is render, save or init — so this does not make the glider invulnerable.

### 9.1 `FlagGliderNormal` in detail (`Modes.c:334`–`:362`)

This is the canonical "reset to a flyable state" routine and every arrival path
ends here. Pseudocode:

```
 1  FlagGliderNormal(thisGlider):
 2      dest.right        = dest.left + kGliderWide          // 48   :336
 3      dest.bottom       = dest.top  + kGliderHigh          // 20   :337
 4      destShadow.right  = destShadow.left + kGliderWide    // 48   :338
 5      destShadow.bottom = destShadow.top + kShadowHigh     //  9   :339
 6      mode = kGliderNormal                                 //      :340
 7      if (facing == kFaceLeft): src = mask = gliderSrc[2]   //     :343-344
 8      else:                     src = mask = gliderSrc[0]   //     :348-349
 9      hVel = 0;  vVel = 0                                  //      :351-352
10      hDesiredVel = 0;  vDesiredVel = 0                    //      :353-354
11      tipped = false                                       //      :355
12      ignoreLeft = false;  ignoreRight = false;  ignoreGround = false  // :356-358
13      dontDraw = false                                     //      :359
14      frame = 0                                            //      :360
15      shadowVisible = IsShadowVisible()                    //      :361  GLOBAL, per room
```

That is the **whole** function — 26 statements, nothing else. In particular
`FlagGliderNormal` does **not** touch `whole`, `wholeShadow`, `destShadow.left`
or `destShadow.top`: it only re-derives the two *far* edges of each rect from the
near edges it leaves alone. Callers that need `whole`/`wholeShadow` re-anchored
do it themselves afterwards (e.g. `OffAMortal`, `GliderPRO/Sources/Player.c:1535`–`:1538`);
for the momentum modes the next `MoveGlider` call rewrites all four edges of
`whole` anyway (§7.2.5).

Note lines 2–3: **`dest` is re-anchored at its top-left.** A glider that was 26
tall (burning, `kGliderBurningHigh` against `kGliderHigh` 20 —
`GliderPRO/Headers/GliderDefines.h:549`, `:551`) and then runs `FlagGliderNormal`
therefore has its bottom edge jump *up* by 6 px, because the height is re-derived
from the top. `StartGliderFadingOut` does the opposite, anchoring at the bottom
(`dest.top = dest.bottom - kGliderHigh`, `Modes.c:99`, and only inside the
`RectTall(&dest) > kGliderHigh` guard at `:87`). The asymmetry is real, and both
halves of it must be transcribed exactly as written.

The route to it, however, is *not* the staircase. A burning glider cannot take the
stairs at all: `case kMoveItUp:` and `case kMoveItDown:` both intercept
`mode == kGliderBurning` first and fade the glider out instead
(`GliderPRO/Sources/Interactions.c:1253`–`:1258` and `:1288`–`:1293`), and those two
cases are the only callers of `StartGliderGoingUpStairs`/`DownStairs`, so
`frame = kWasBurning` (`Modes.c:128`, `:145`) never fires and the
`frame == kWasBurning` arms at `Player.c:417`–`:418` and `:546`–`:547` are dead.
Even if they were reached, they would not show the 6 px step: the arrival rebuilds `dest` from
the sprite (`dest = src; ZeroRectCorner(&dest); QOffsetRect(&dest, rightClip, 100)`,
`Modes.c:537`–`:539`), so the glider is 20 tall on arrival, and the `kWasBurning`
arm calls `FlagGliderBurning`, not `FlagGliderNormal`.

The reachable route is the foil pickup. `case kFoil:`
(`Interactions.c:900`–`:913`, reached from `case kRewardIt:` at `:1246`–`:1247` via
`HandleRewards`) has no mode gate, and neither do `HandleHotSpotCollision`,
`CheckForHotSpots` or `HandleInteraction`. `StartGliderFoilGoing` returns only for
modes 18 and 21 (`Modes.c:583`–`:584`) and never resizes `dest`, so a burning
26-tall glider enters `kGliderGoingFoil` still 26 tall; `HandleGlider` then
dispatches `MoveGliderFoilGoing` rather than `MoveGliderBurning`
(`Player.c:1411`–`:1412`), which freezes the `wasMode` fuse at `Player.c:220` so the
glider survives all nine dissolve frames, and frame 9 calls `FlagGliderNormal`
(`Player.c:1175`) — bottom edge up 6 px. The sibling
`MoveGliderFoilLosing` → `FlagGliderNormal` (`Player.c:1234`) is reachable the same
way. A fidelity test for this asymmetry has to be written against the foil path;
one written against the staircase can never run.

Do not, on the strength of that, delete the dead `frame == kWasBurning` branches or
the `frame = kWasBurning` writes that feed them. `frame` survives the room
transition untouched — `StartGliderGoingUpStairs` sets it, `MoveGliderUpStairs`
never writes it, and `ReadyGliderForTripUpStairs` (`Modes.c:519`–`:546`) does not
reset it before calling `FinishGliderUpStairs` — so the unreachability rests
entirely on the two interaction guards above, not on anything in the stairs code
itself. Transcribe both arms.

One further trap in this neighbourhood, and it is the one that makes the asymmetry
easy to lose in a port. `FlagGliderNormal` and `FlagGliderBurning` rebuild `dest`
from *opposite* corners but rebuild `destShadow` from the same corner in both cases
(`destShadow.bottom = destShadow.top + kShadowHigh`, `Modes.c:339` and `:415`), so
factoring the two into one shared resize helper parameterised on an anchor silently
breaks the shadow.

`shadowVisible` is a **global** (`GliderPRO/Sources/Player.c:52`), not per-glider.
In a two-player game whichever glider most recently ran `FlagGliderNormal` — or
started shredding (`Player.c:1286`) — controls whether *both* shadows draw.

---

## 10. Input

### 10.1 Key maps

Input is read with the Toolbox `GetKeys(theKeys)` into a `KeyMap` (a 128-bit
bitmap of physical keys) and tested with `BitTst(&theKeys, offset)`
(`GliderPRO/Sources/Input.c:286`, `:301`). The four per-glider key fields hold
raw **virtual key-code bit offsets**, not characters.

Offsets, from `GliderPRO/Headers/Externs.h:129`–`:163`:

| Symbol | Offset | Key |
| --- | --- | --- |
| `kUpArrowKeyMap` | 121 | Up arrow |
| `kDownArrowKeyMap` | 122 | Down arrow |
| `kRightArrowKeyMap` | 123 | Right arrow |
| `kLeftArrowKeyMap` | 124 | Left arrow |
| `kCommandKeyMap` | 48 | Command |
| `kEscKeyMap` | 50 | Escape |
| `kDeleteKeyMap` | 52 | Delete |
| `kTabKeyMap` | 55 | Tab |
| `kControlKeyMap` | 60 | Control |
| `kOptionKeyMap` | 61 | Option |
| `kCapsLockKeyMap` | 62 | Caps Lock |
| `kShiftKeyMap` | 63 | Shift |
| `kQKeyMap` | 11 | Q |
| `kSKeyMap` | 6 | S |

Defaults:

| Glider | `leftKey` | `rightKey` | `battKey` | `bandKey` | Set at |
| --- | --- | --- | --- | --- | --- |
| `theGlider` (player 1) | 124 (Left) | 123 (Right) | 122 (Down) | 121 (Up) | `GliderPRO/Sources/Main.c:135`–`:138` (also restored from prefs at `:69`–`:72`, saved at `:225`–`:228`, remappable via `GliderPRO/Sources/Settings.c:527`–`:556` and defaulted at `:1237`–`:1240`) |
| `theGlider2` (player 2) | 60 (Control) | 48 (Command) | 61 (Option) | 63 (Shift) | `GliderPRO/Sources/InterfaceInit.c:148`–`:151` — **hard-coded, not remappable** |

`theGlider.which = kPlayer1` (TRUE) at `GliderPRO/Sources/InterfaceInit.c:147`;
`theGlider2.which = kPlayer2` (FALSE) at `:152`.

Player 2 using Command as "right" is why the Command-key check in `GetInput` is
guarded by `if (thisGlider->which == kPlayer1)`
(`GliderPRO/Sources/Input.c:283`–`:288`) — otherwise player 2 moving right would
trigger command-key handling.

### 10.2 `GetInput` (`GliderPRO/Sources/Input.c:281`–`:379`)

```
  1  GetInput(thisGlider):
  2      if (thisGlider->which == kPlayer1):
  3          GetKeys(theKeys)                                    // :285
  4          if (BitTst(&theKeys, kCommandKeyMap)):              // :286
  5              DoCommandKey()                                  // :287
  6
  7      // ---- BURNING OVERRIDE (:290-296) ----
  8      if (thisGlider->mode == kGliderBurning):
  9          if (thisGlider->facing == kFaceLeft):
 10              thisGlider->hDesiredVel -= kNormalThrust        // -5
 11          else:
 12              thisGlider->hDesiredVel += kNormalThrust        // +5
 13          // and NOTHING else is read this frame
 14      else:
 15          thisGlider->heldLeft  = false                       // :299
 16          thisGlider->heldRight = false                       // :300
 17
 18          if (BitTst(&theKeys, thisGlider->rightKey)):         // :301
 19              LogDemoKey(0)      // #ifdef CREATEDEMODATA only  :303-305
 20              if (BitTst(&theKeys, thisGlider->leftKey)):      // BOTH held
 21                  ToggleGliderFacing(thisGlider)              // :308
 22                  thisGlider->heldLeft = true                 // :309
 23              else:
 24                  thisGlider->hDesiredVel += kNormalThrust    // +5   :313
 25                  thisGlider->tipped = (thisGlider->facing == kFaceLeft)   // :314
 26                  thisGlider->heldRight = true                // :315
 27          else if (BitTst(&theKeys, thisGlider->leftKey)):     // :318
 28              LogDemoKey(1)      // #ifdef CREATEDEMODATA only  :320-322
 29              thisGlider->hDesiredVel -= kNormalThrust        // -5   :323
 30              thisGlider->tipped = (thisGlider->facing == kFaceRight)      // :324
 31              thisGlider->heldLeft = true                     // :325
 32          else:
 33              thisGlider->tipped = false                      // :328
 34
 35          // ---- battery / helium (:330-:342) ----
 36          if (BitTst(&theKeys, thisGlider->battKey) &&
 37              batteryTotal != 0 && thisGlider->mode == kGliderNormal):
 38              LogDemoKey(2)      // #ifdef CREATEDEMODATA only  :333-335
 39              if (batteryTotal > 0): DoBatteryEngaged(thisGlider)   // :337
 40              else:                  DoHeliumEngaged(thisGlider)    // :339
 41          else:
 42              batteryWasEngaged = false                       // :342
 43
 44          // ---- rubber band (:344-:364) ----
 45          if (BitTst(&theKeys, thisGlider->bandKey) &&
 46              bandsTotal > 0 && thisGlider->mode == kGliderNormal):
 47              LogDemoKey(3)      // #ifdef CREATEDEMODATA only  :347-349
 48              if (!thisGlider->fireHeld):                      // edge trigger  :350
 49                  if (AddBand(thisGlider, thisGlider->dest.left + 24,
 50                                          thisGlider->dest.top + 10,
 51                                          thisGlider->facing)):     // :352-353
 52                      bandsTotal--                                  // :355
 53                      if (bandsTotal <= 0) QuickBandsRefresh(false) // :356-357
 54                      thisGlider->fireHeld = true                   // :359
 55          else:
 56              thisGlider->fireHeld = false                    // :364
 57
 58          // ---- suicide / pause (:366-:377) -- STILL INSIDE the non-burning else ----
 59          if (otherPlayerEscaped != kNoOneEscaped &&
 60              BitTst(&theKeys, kDeleteKeyMap) &&
 61              thisGlider->which && !onePlayerLeft):            // :366-368
 62              ForceKillGlider()                                // :370
 63          if ((isEscPauseKey  && BitTst(&theKeys, kEscKeyMap)) ||
 64              (!isEscPauseKey && BitTst(&theKeys, kTabKeyMap))):    // :373-374
 65              DoPause()                                        // :376  (takes no args)
```

Two structural points the indentation is load-bearing for: the suicide and pause
tests are **inside** the non-burning `else` block (it closes at `:378`), so a
burning glider can neither commit suicide nor pause; and the pause key is chosen
by the global `isEscPauseKey` preference rather than both keys being live.

Observations a port must honour:

* **`GetKeys` is only called for player 1** (line 3). Player 2's input is read
  from the *same* `theKeys` snapshot, which is a file-scope global
  (`GliderPRO/Sources/Input.c:31`). `PlayGame` calls `GetInput(&theGlider)`
  before `GetInput(&theGlider2)` (`GliderPRO/Sources/Play.c:452`–`:453`,
  immediately before `HandleInteraction` at `:454`), so the ordering is: sample once, then both gliders read the sample. A Go port
  must sample the keyboard exactly once per frame and share it.
* **The burning override is total** (lines 8–13). A burning glider cannot be
  steered, cannot use the battery, cannot fire bands, and `heldLeft`/`heldRight`
  are *not cleared* — they keep whatever value they had when the fire started.
  Since `heldRight` blocks the up-stairs hot spot
  (`if (!thisGlider->heldRight && GliderInRect(...))`,
  `GliderPRO/Sources/Interactions.c:1251`) and `heldLeft` blocks the down-stairs
  one (`:1286`) — the only two reads of either flag outside `Input.c` — a glider
  that catches fire while holding right can be
  permanently unable to take the up-stairs for the whole 60-frame burn.
* **Pressing both direction keys turns the glider around** (lines 20–22) but only
  sets `heldLeft`, never `heldRight`. And `ToggleGliderFacing` silently returns
  unless `mode == kGliderNormal` (`GliderPRO/Sources/Modes.c:486`–`:487`), so the
  about-face cannot be triggered mid-tumble, mid-fade, on the stairs, etc.
* **`tipped` is derived, not stored.** `tipped = (facing == kFaceLeft)` when
  pushing right and `(facing == kFaceRight)` when pushing left: it means "I am
  being pushed against the way I am pointing", i.e. flying backwards. It is
  cleared whenever no direction key is held (line 33). `tipped` selects the
  banked sprite (`gliderSrc[1]`/`[3]`) and, more importantly, **reverses the
  battery thrust direction** (§10.4) and **decides which mail slot you can
  enter** (§13.5).
* **The bands/battery keys require `mode == kGliderNormal`.** You cannot thrust
  while tumbling, fading, on the stairs, in a duct, in the mail, gaining foil, or
  shredding.
* The Delete-key suicide (lines 59–62) is only available to **player 1**
  (`thisGlider->which` is TRUE only for player 1) when the *other* player has
  already escaped the room and both are still alive. It calls `ForceKillGlider`
  (`GliderPRO/Sources/Transit.c:449`–`:469`), which fades out whichever glider is
  **not** in limbo and sets the global `playerSuicide = true`.

### 10.3 `LogDemoKey` and the demo

`LogDemoKey(char keyIs)` (`GliderPRO/Sources/Input.c:44`–`:49`) appends
`{ gameFrame, key }` to `demoData[]` when recording. Every call site is wrapped in
`#ifdef CREATEDEMODATA`, so in a normal build `GetInput` does not log at all. `GetDemoInput`
(`:186`–`:277`) replays it, switching on the logged key:

| logged key | effect in `GetDemoInput` | line |
| --- | --- | --- |
| 0 | `hDesiredVel += kNormalThrust`, `tipped = (facing == kFaceLeft)`, `heldRight = true`, `fireHeld = false` | `:228`–`:233` |
| 1 | `hDesiredVel -= kNormalThrust`, `tipped = (facing == kFaceRight)`, `heldLeft = true`, `fireHeld = false` | `:235`–`:240` |
| 2 | battery / helium, then `fireHeld = false` | `:242`–`:248` |
| 3 | rubber band — the only case that does **not** clear `fireHeld`; a successful `AddBand` sets it | `:250`–`:263` |

The source comments on cases 0 and 1 say `// left key` and `// right key`
respectively, which is **backwards** relative to what the code does (case 0
thrusts right). Follow the code, not the comments.

The switch is only half the story, and the half around it is what a replay
transcribed from the table alone gets wrong. `GetDemoInput` clears three flags
before it even looks at the demo record — `thisGlider->heldLeft = false;
thisGlider->heldRight = false; thisGlider->tipped = false;` (`:220`–`:222`) —
which is *not* what `GetInput` does: `GetInput` clears only
`heldLeft`/`heldRight` up front (`:299`–`:300`), clears `tipped` in the `else` of
the direction tests (`:328`), and leaves `tipped` untouched altogether on the
both-keys-held path (`:306`–`:310`). Then the record is consumed only if
`gameFrame == (long)demoData[demoIndex].frame` (`:224`), with `demoIndex++`
*inside* that branch (`:266`) and `else thisGlider->fireHeld = false;`
(`:268`–`:269`) on every frame that has no matching record.

Those `fireHeld` clears exist because a replay has no key state to release. In
`GetInput` the band key's own `else` rearms the edge trigger (`:363`–`:364`); in
the demo the three non-band cases and the no-record path have to do it by hand,
which is why case 3 is the only one that leaves the flag alone. Transcribe the
table without them and `fireHeld` latches true after the first band, so the
`if (!thisGlider->fireHeld)` guard at `:251` swallows every later recorded shot;
drop the `tipped` reset and `tipped` latches true after the first backwards push,
which silently reverses battery thrust (§10.4) and changes which mail slot the
glider may enter (§13.5) for the remainder of the replay.

Three further traps surround the switch. The burning override sits *before* the
reset block (`:211`–`:217`), so a burning glider keeps its old
`heldLeft`/`heldRight`/`tipped`/`fireHeld` and consumes no record at all —
`demoIndex` does not advance, and because the frame test at `:224` is an equality
rather than a `>=`, the pending record is stranded until `gameFrame` happens to
reach it. `LogDemoKey` is called once per pressed key per frame (`:304`, `:321`,
`:334`, `:348`), so one recorded frame can hold several records, yet
`GetDemoInput` consumes at most one per frame; a port that drains every record
matching the current frame will not replay the same run. And `theKeys` is still
read during playback for the pause key (`:271`–`:272`) even though the demo
drives movement, while it is only refreshed by `GetKeys` when
`thisGlider->which == kPlayer1` (`:188`–`:190`).

`kDemoLength` = 6702 (`GliderDefines.h:625`) is a **byte** count, not a frame
count: it is the argument to `NewPtr` and to `BlockMove` for the whole buffer
(`GliderPRO/Sources/StructuresInit2.c:287`, `:295`) and is summed as bytes in the
memory estimate at `GliderPRO/Sources/Environ.c:657`. Since `demoType` is
`{ long frame; char key; char padding; }` = 6 bytes
(`GliderPRO/Headers/GliderStructs.h:334`–`:339`) under the 68k's two-byte
alignment — 6702 divides by 6 but not by 8, which rules out a padded eight-byte
record — the shipped `'demo'` 128 resource's 6702 bytes are exactly 1117 records;
parsing it out of `GliderPRO/Glider PRO.r:199389` gives `frame` rising
monotonically from 46 to 3414 and only keys 0 (910 records), 1 (198) and 3 (9):
the recorded run never touches the battery. The event count therefore has to be
derived from the resource's own length divided by 6, never from `kDemoLength`
read as a count, which would claim 6702 events, six times the truth, and walk
5585 records of heap garbage past the end of the buffer. The recording
build sizes the buffer differently again,
`NewPtr(sizeof(demoType) * 2000)` (`StructuresInit2.c:282`), and dumps
`sizeof(demoType) * demoIndex` bytes (`GliderPRO/Sources/Play.c:217`), which is
where the 6702 = 6 × 1117 figure came from in the first place.

Because `LogDemoKey` assigns only `frame` and `key` (`Input.c:44`–`:49`), the record's
third field is never initialised, and the shipped resource duly carries whatever
was in that heap block when the demo was recorded: mostly zeroes, but also 0xFF
and fragments of ASCII text in no pattern. Read six bytes per record and ignore
the sixth; a port that compares, hashes or checksums whole records will disagree
with the resource for reasons that have nothing to do with the demo.

`demoIndex` is a `short` zeroed in `NewGame` (`Play.c:114`), and `GetDemoInput`
never tests it against `kDemoLength` or against any record count
(`Input.c:224`, `:266`),
so the original has no end-of-demo condition at all: past the 1117th record it
keeps reading off the end of the buffer and simply never sees another `gameFrame`
match. Playback ends only when the frame loop itself ends,
`while ((playing) && (!quitting))` (`Play.c:432`), or when `gameOver` closes the
guard around the call, `if (!gameOver) { if (demoGoing) GetDemoInput(&theGlider); … }`
(`Play.c:476`–`:479`) — in practice when the demo house kills the glider. A Go
port does need a bounds check to avoid a panic, but it must only stop the port
consuming records; clearing `playing` or setting `gameOver` when the records run
out would end the attract mode earlier than the original does.

Under `#if BUILD_ARCADE_VERSION` the demo aborts on any player key press
(`GliderPRO/Sources/Input.c:192`–`:201`).

### 10.4 `DoBatteryEngaged` (`GliderPRO/Sources/Input.c:121`–`:156`)

```
 1  DoBatteryEngaged(thisGlider):
 2      if (thisGlider->facing == kFaceLeft):
 3          if (thisGlider->tipped): thisGlider->hVel += kHyperThrust   // +8  :126
 4          else:                    thisGlider->hVel -= kHyperThrust   // -8  :128
 5      else:
 6          if (thisGlider->tipped): thisGlider->hVel -= kHyperThrust   // -8  :133
 7          else:                    thisGlider->hVel += kHyperThrust   // +8  :135
 8      batteryTotal--                                                  // :138
 9      if (batteryTotal == 0):                                         // :140
10          QuickBatteryRefresh(false)                                  // :142
11          PlayPrioritySound(kFizzleSound, kFizzlePriority)             // :143
12      else:
13          if (!batteryWasEngaged): batteryFrame = 0                   // :147-148
14          if (batteryFrame == 0):
15              PlayPrioritySound(kThrustSound, kThrustPriority)        // :149-150
16          batteryFrame++                                              // :151  unconditional
17          if (batteryFrame >= 4): batteryFrame = 0                    // :152-153
18          batteryWasEngaged = true                                    // :154
```

Note the order at lines 13–17: the sound test happens **before** the increment,
and the increment is not inside an `else`. Copy the statements verbatim; a
restructured version that increments first gets the sound phase wrong.

The truth table for direction is `facing XOR tipped`:

| `facing` | `tipped` | `hVel` change | Interpretation |
| --- | --- | --- | --- |
| left (0) | false | **−8** | pointing left, flying left → thrust left |
| left (0) | true | **+8** | pointing left, flying right (backwards) → thrust right |
| right (1) | false | **+8** | pointing right, flying right → thrust right |
| right (1) | true | **−8** | pointing right, flying left (backwards) → thrust left |

So the battery always pushes **in the direction of travel**, which is exactly the
direction the held key is pushing (`tipped` is only true when the key opposes
`facing`). Thrust is applied **directly to `hVel`**, bypassing the `kHImpulse`
ramp, which is why the battery feels instantaneous.

`kThrustSound` plays on `batteryFrame == 0` of a 0..3 cycle, so in a one-player
game it sounds every 4th frame of a sustained hold and immediately on a fresh
press. That cadence does not survive a second player. `batteryFrame` and
`batteryWasEngaged` are file-scope globals shared by both gliders
(`GliderPRO/Sources/Input.c:33`–`:34`), not glider fields, and the reset
`else batteryWasEngaged = false;` (`:341`–`:342`) is the `else` of the *whole*
battery test at `:330`–`:331`, so it fires for any glider that did not thrust this
frame rather than only when the key is released. Because `PlayGame` calls
`GetInput(&theGlider); GetInput(&theGlider2);` back to back
(`GliderPRO/Sources/Play.c:452`–`:453`), there are three cadences, not one: one
player, every 4th frame; two players with only one thrusting, `batteryWasEngaged`
is false at every `DoBatteryEngaged` call, so `batteryFrame` is forced back to 0
and the sound plays **every** frame; two players both thrusting,
`DoBatteryEngaged` runs twice per frame, so `batteryFrame` advances by 2, the
sound plays every 2nd frame and `batteryTotal` drains at 2/frame. The one
exception is a burning partner: the reset at `:342` sits inside the non-burning
`else` that opens at `:298`, so while the other glider is in `kGliderBurning`
`GetInput` takes the override at `:290`–`:296`, never touches
`batteryWasEngaged`, and the thrusting player gets the plain 4-frame cadence back
for the length of the burn. `DoHeliumEngaged` and `kHissSound` behave identically
(`:173`–`:180`). A port that moves `batteryFrame`/`batteryWasEngaged` into the
per-glider struct — the obvious Go design — cannot reproduce any of this; keep
them package-level (§23 item 10).

### 10.5 `DoHeliumEngaged` (`GliderPRO/Sources/Input.c:160`–`:182`)

```
 1  DoHeliumEngaged(thisGlider):
 2      thisGlider->vDesiredVel = -kHeliumLift          // -4   :162
 3      batteryTotal++                                  // :163  (negative -> toward 0)
 4      if (batteryTotal == 0):                         // :165
 5          QuickBatteryRefresh(false)                  // :167
 6          PlayPrioritySound(kFizzleSound, kFizzlePriority)   // :168
 7          batteryWasEngaged = false                   // :169
 8      else:
 9          if (!batteryWasEngaged): batteryFrame = 0   // :173-174
10          if (batteryFrame == 0):
11              PlayPrioritySound(kHissSound, kHissPriority)   // :175-176
12          batteryFrame++                              // :177  unconditional
13          if (batteryFrame >= 4): batteryFrame = 0     // :178-179
14          batteryWasEngaged = true                    // :180
```

`batteryTotal` is a **signed** counter shared by battery and helium: positive
means battery charges, negative means helium units. Both count *toward* zero.
Helium sets `vDesiredVel = −4` — the ramp then lifts `vVel` by 2/frame, so the
glider rises at 4 px/frame after 4 frames (§7.3). Helium does **not** bypass the
ramp the way the battery does.

The energy pickups are mutually destructive: taking a battery when you hold
helium sets `batteryTotal = kBatterySupply` (50) outright rather than adding
(`GliderPRO/Sources/Interactions.c:860`–`:863`), and taking helium when you hold
battery sets `batteryTotal = -kHeliumSupply` (−150)
(`GliderPRO/Sources/Interactions.c:966`–`:969`). Both sign tests are strict
(`batteryTotal > 0` and `batteryTotal < 0`), so an exhausted `batteryTotal == 0`
takes the assignment branch — which happens to give the same answer as
accumulating, but only by coincidence. Note also that both cases halve `hVel` and
`vVel` two lines earlier (`:858`–`:859`, `:964`–`:965`), as every supply pickup
does; see §14.

### 10.6 Rubber bands and recoil

`AddBand` (`GliderPRO/Sources/RubberBands.c:256`–`:290`):

```
 1  AddBand(thisGlider, h, v, direction):
 2      if (numBands >= kMaxRubberBands): return         // kMaxRubberBands = 2
 3      bands[numBands].mode  = 0
 4      bands[numBands].count = 0
 5      bands[numBands].vVel  = thisGlider->tipped ? -2 : 0
 6      QSetRect(&bands[numBands].dest, h - 8, v - 3, h + 8, v + 3)
 7      if (direction == kFaceRight):
 8          QOffsetRect(&bands[numBands].dest, 32, 0)    // muzzle offset
 9          bands[numBands].hVel = kRubberBandVelocity   // +20
10      else:
11          QOffsetRect(&bands[numBands].dest, -32, 0)
12          bands[numBands].hVel = -kRubberBandVelocity  // -20
13      thisGlider->hVel -= (bands[numBands].hVel / 2)   // RECOIL: -/+10
14      numBands++
15      PlayPrioritySound(kFireBandSound, kFireBandPriority)
```

Called from `GetInput` with `h = dest.left + 24` (the glider's horizontal
centre — `kHalfGliderWide`) and `v = dest.top + 10` (its vertical centre)
(`GliderPRO/Sources/Input.c:352`–`:353`).

Line 13 is the player-facing part: **firing a band applies exactly 10 px/frame of
recoil** in the opposite direction, added straight to `hVel` with no ramp. With
`kMaxRubberBands` = 2 (`GliderDefines.h:261`) you can only have two in flight.
Line 5 gives a band fired while flying backwards a slight upward velocity (−2).

Bands can also hit *you*. `CheckBandCollision` is
`GliderPRO/Sources/RubberBands.c:37`–`:204`, of which the glider-overlap test is
only the tail, `:148`–`:194`:

```
 1  // only if bands[who].hVel != 0
 2  if (band overlaps theGlider.dest on all four edges):
 3      theGlider.hVel += (bands[who].hVel / 2)          // +/-10
 4      bands[who].hVel = 0
 5      PlayPrioritySound(kHitWallSound, kHitWallPriority)
```

guarded by `(!twoPlayerGame) || (!onePlayerLeft) || (playerDead == kPlayer2)` for
player 1 and the mirrored condition for player 2. So in a two-player game you can
shoot your partner for a 10 px/frame shove, and the band stops dead (its `hVel`
becomes 0 so it cannot hit twice).

What that tail hides is that the function *opens* with a rebound, not a kill. If
the room has a real left wall — `leftThresh == kLeftWallLimit`, set per room at
`GliderPRO/Sources/Room.c:837`, `:859`, `:872`, `:885` and `:909` — and
`dest.left < kLeftWallLimit`, then `hVel` is negated **only if the band is still
moving into the wall** (`if (bands[who].hVel < 0)`), `dest.left` is snapped to
exactly `kLeftWallLimit` with `dest.right = dest.left + 16`, `kBandReboundSound`
plays at `kBandReboundPriority` and `collided` is set (`:44`–`:52`). The right
wall is the mirror image, snapping `dest.right = kRightWallLimit` and
`dest.left = dest.right - 16` (`:53`–`:61`). The two are `if` / `else if`, so at
most one fires per frame.

Between that and the glider test sits a pass over every `isOn` hot spot
(`:63`–`:143`), which is the second, object-driven way a band comes back at the
player. For `kDissolveIt` and `kBounceIt` the band is reversed only if the
lookahead `(dest.right - bands[who].hVel) < bounds.left` (mirrored for leftward
bands) says it began the frame clear of the object, and is otherwise given
`mode = kKillBandMode` (−1) (`:89`–`:117`); `kRewardIt` on a `kGreaseRt`/`kGreaseLf`
object calls `SetObjectState` then `SpillGrease` and clears the hot spot's `isOn`
(`:118`–`:130`); `kSwitchIt` calls `HandleSwitches` and `kTriggerIt` calls
`ArmTrigger` (`:131`–`:138`). The whole pass is debounced by the file-scope
`bandHitLast`, so a band overlapping one object toggles it once rather than once
per frame.

`HandleBands` (`GliderPRO/Sources/RubberBands.c:208`–`:252`) advances each band:
`mode++` (wrapping 0,1,2 for the 3 sprite frames), `count++` and every 4 frames
(`kBandFallCount`) `vVel++` — so bands accelerate downward at 1 px per 4 frames,
i.e. 0.25 px/frame², a much gentler gravity than the glider's.

The kill tests close `CheckBandCollision`: `dest.left < kLeftWallLimit` or
`dest.right > kRightWallLimit` (`:195`–`:199`), else
`dest.bottom > kFloorLimit` (312) (`:200`–`:203`), each setting
`mode = kKillBandMode` with no sound. They read the absolute constants 12 and 500,
**not** `leftThresh`/`rightThresh`, and it is that pairing with the rebound above
that decides the behaviour. In a walled room the rebound has already snapped
`dest.left` to exactly 12 (or `dest.right` to exactly 500) and the tests are
strict, so neither side test can fire: bands ping-pong horizontally and die only
on the floor. In an open-sided room, where `leftThresh` is `kNoLeftWallLimit`
(−24) and `rightThresh` is `kNoRightWallLimit` (536), the rebound is skipped and
the band vanishes silently at the 12/500 plane — a band cannot follow the player
out of the room the way the glider does.

---

## 11. Hit boxes

There are **four distinct** hit-box computations. Using the wrong one is a
gameplay-visible bug.

### 11.1 `SectGlider` — the general overlap test (`GliderPRO/Sources/Interactions.c:101`–`:130`)

```
 1  SectGlider(thisGlider, theRect, scrutinize) -> Boolean:      // :101
 2      glideBounds = thisGlider->dest                           // :106
 3      if (thisGlider->mode == kGliderBurning):                 // :107
 4          glideBounds.top += 6                                 // :108  26 -> 20
 5      if (scrutinize):                                         // :110
 6          glideBounds.left   += 5                              // :112
 7          glideBounds.top    += 5                              // :113
 8          glideBounds.right  -= 5                              // :114
 9          glideBounds.bottom -= 5                              // :115
10      if      (theRect->bottom < glideBounds.top):    itHit = false  // :118-119
11      else if (theRect->top    > glideBounds.bottom): itHit = false  // :120-121
12      else if (theRect->right  < glideBounds.left):   itHit = false  // :122-123
13      else if (theRect->left   > glideBounds.right):  itHit = false  // :124-125
14      else:                                           itHit = true   // :126-127
15      return itHit                                                   // :129
```

With `scrutinize`, the box shrinks to **38 x 10** centred in the 48 x 20 sprite
(and 38 x 10 for a burning glider too, since the +6 is applied first).

The test is written as a chain of `else if`s that *reject*, so the effective
predicate is

```
itHit == (theRect->bottom >= glideBounds.top)    &&
         (theRect->top    <= glideBounds.bottom) &&
         (theRect->right  >= glideBounds.left)   &&
         (theRect->left   <= glideBounds.right)
```

— i.e. all four comparisons are **non-strict**, so two rects that merely *touch*
along an edge (`theRect->bottom == glideBounds.top`) **do** count as a hit. A Go
port that writes the idiomatic half-open `image.Rectangle.Overlaps` (which uses
strict `<`) will miss exactly the one-pixel-touching case, and that case happens
constantly because the integrator moves in whole pixels.

`scrutinize` comes from the hot spot's `doScrutinize` flag. Four call sites pass
it: three in `CheckForHotSpots` — `GliderPRO/Sources/Interactions.c:1639`–`:1640`
and `:1657`–`:1658` for the two gliders of a two-player game, `:1679`–`:1680` for
the one-player case — and one in `FlagStillOvers` (`:1723`–`:1724`). The flag is
set per-object in `ObjectRects.c` (e.g. the shredder sets it true,
`GliderPRO/Sources/ObjectRects.c:949`). Dynamic-object collisions always
scrutinize (`GliderPRO/Sources/Dynamics.c:42`).

### 11.2 `GliderInRect` — full containment (`GliderPRO/Sources/Interactions.c:134`–`:150`)

```
 1  GliderInRect(thisGlider, theRect) -> Boolean:
 2      return (thisGlider->dest.left   >= theRect->left)  &&
 3             (thisGlider->dest.right  <= theRect->right) &&
 4             (thisGlider->dest.top    >= theRect->top)   &&
 5             (thisGlider->dest.bottom <= theRect->bottom)
```

No burning adjustment, no scrutinize. It is used wherever the glider must be
*entirely* inside a rect before the object is allowed to act on it, and that list
is much longer than the handful of objects one would guess. Twelve call sites in
all: the stairs (`:1251`, `:1286`), the shredder (`:1326`), the transporter
(`:1388`), both mailboxes (`:1431`, `:1474`), both transporter ducts (`:1517`,
`:1550`), the microwave (`:1582`), the web (`:1603` for a live glider and `:1607`
for a burning one), and the vestigial test inside `WebGlider` itself (`:1741`).

That completeness is the most important thing in this section for a porter,
because the sections that describe those objects each name only the *other* half
of their gate: §13.5 is about the mailbox's facing and `tipped` combination, §13.7
about the duct's `!who->stillOver` latch, §14 about the microwave's. A
port that gates those six objects on the `SectGlider(..., doScrutinize)` overlap
`CheckForHotSpots` performs (§13.8) will swallow the glider the moment its nose
touches the 72 x 40 mailbox rect or the 76 x 48 duct rect, rather than waiting
until the whole 48 x 20 `dest` box is inside it, and will web or microwave it on
mere contact. Mail and duct entry positions come out visibly wrong and every one
of those objects becomes far easier to trigger than it should be.

The web case deserves spelling out, because the C is redundant there and the
redundancy is easy to copy the wrong way round. `WebGlider` opens with
`if ((thisGlider->mode == kGliderBurning) && (GliderInRect(thisGlider,
webBounds)))` (`:1741`), but it has exactly one caller (`:1605`) and that call is
already guarded by `mode != kGliderBurning` (`:1604`), so `:1741` can never be
true in practice. The branch that actually burns a webbed glider to death is the
caller's `else if ((mode == kGliderBurning) && GliderInRect(...))` at
`:1606`–`:1611`. Observable behaviour is the same either way, but a port that
implements only the inner test puts the burning check in the wrong function *and*
loses the containment gate on the ordinary webbing path.

### 11.3 `GliderHitTop` — the "did I hit the underside" test (`GliderPRO/Sources/Interactions.c:54`–`:97`)

Used only for `kDissolveIt` objects, and only once the glider has been found to
have foil (`GliderPRO/Sources/Interactions.c:1218`–`:1244`). The name invites the
reading that foil lets you bounce off things that would otherwise dissolve you,
but the return value is used the other way round: `true` means you die anyway.
See the discussion after the pseudocode.

```
 1  GliderHitTop(thisGlider, theRect) -> Boolean:            // :54
 2      glideBounds.left   = thisGlider->dest.left   + 5      // :60
 3      glideBounds.top    = thisGlider->dest.top    + 5      // :61
 4      glideBounds.right  = thisGlider->dest.right  - 5      // :62
 5      glideBounds.bottom = thisGlider->dest.bottom - 5      // :63
 6      glideBounds.left  -= thisGlider->wasHVel              // :65   UN-SWEEP
 7      glideBounds.right -= thisGlider->wasHVel              // :66
 8      if      (theRect->bottom < glideBounds.top):    hitTop = false   // :68-69
 9      else if (theRect->top    > glideBounds.bottom): hitTop = false   // :70-71
10      else if (theRect->right  < glideBounds.left):   hitTop = false   // :72-73
11      else if (theRect->left   > glideBounds.right):  hitTop = false   // :74-75
12      else:                                           hitTop = true    // :76-77
13      if (!hitTop):                              // :79  -- NOT a hit vertically
14          PlayPrioritySound(kFoilHitSound, kFoilHitPriority)   // :81
15          foilTotal--                                          // :82
16          if (foilTotal <= 0): StartGliderFoilLosing(thisGlider)  // :83-84
17          glideBounds.left  += thisGlider->wasHVel   // :86  re-sweep
18          glideBounds.right += thisGlider->wasHVel   // :87
19          if (thisGlider->hVel > 0):                                 // :88
20              offset = 2 + glideBounds.right - theRect->left         // :89
21          else:                                                      // :90
22              offset = 2 + glideBounds.left - theRect->right         // :91
23          thisGlider->hVel = -thisGlider->hVel - offset              // :93
24      return hitTop                                                  // :96
```

The un-sweep at lines 7–8 rewinds the box by last frame's horizontal velocity so
the test asks "was I already overlapping this object *before* I moved
horizontally?" If yes, `hitTop` is true, the contact is not attributable to this
frame's horizontal motion — you landed on it, rose into it, or were engulfed by
it — and the caller **kills you**: `StartGliderFadingOut` plus `kFadeOutSound`
(`:1225`–`:1226`), foil or no foil. If no, you ran into its *side*, and lines
19–23 reflect `hVel` and push you out by `offset` (the penetration depth plus 2).
Only that side case is survivable, so foil is not a shield against a
`kDissolveIt` object at all; it is a shield against flying into one sideways.
A port that treats the vertical case as absorbed turns foil into
near-invincibility against every stool, table and microwave body in the game.

Note that the four-way test is written as a chain of `else if`s that set
`hitTop = false`, i.e. the *negation* of a rect intersection, and that the
function returns `hitTop` unconditionally (`:96`) — the `!hitTop` block falls
through to the same return. Note `-hVel - offset`: for `hVel > 0` and a penetration of `p`,
`offset = 2 + p` and the new velocity is `-hVel - 2 - p`, i.e. reflected **and
amplified** by the penetration. This is a spring-like ejection, not an elastic
bounce.

A survivable side hit costs **two** foil, not one. Line 15 decrements
unconditionally inside `GliderHitTop`, and the caller decrements a second time in
its `else` branch (`:1230`–`:1235`, that one guarded by `if (foilTotal > 0)`). So
`kFoilSupply` = 8 (`:19`) buys four side hits rather than eight, and only the last
unit costs 1, because the caller's guard sees the value line 15 has already taken
down. Worse, the cost is per *frame* of contact and not per bounce:
`CheckForHotSpots` re-invokes `HandleHotSpotCollision` every frame the scrutinized
box still overlaps (`:1679`–`:1681`) and `kDissolveIt` never consults
`stillOver`, so a glider pressed against the side of a stool burns 2 foil a frame.
Two edge cases follow from line 15 being unguarded: entering with `foilTotal == 1`
costs 1 and fires `StartGliderFoilLosing` from line 16, and entering with
`foilTotal == 0` and `mode == kGliderLosingFoil` — which the outer guard at
`:1221` admits — drives `foilTotal` to −1 and calls `StartGliderFoilLosing` a
second time, harmless only because that function returns early when the mode is
already `kGliderLosingFoil`. A Go port that clamps `foilTotal` at zero will
diverge from anything that later tests its value or sign.

### 11.4 Raw `dest` edge tests

`CheckGliderInRoom` and the four `CheckEscape*` functions compare `dest` edges
directly against the room thresholds with **no inset and no burning
adjustment** (`GliderPRO/Sources/Interactions.c:696`, `:709`, `:722`, `:725`,
`:738`). A burning glider's `dest` really is 6 px taller at the top —
`FlagGliderBurning` sets `dest.top = dest.bottom - kGliderBurningHigh`, 26 rather
than `kGliderHigh` 20 (`GliderPRO/Sources/Modes.c:413`) — so its flame plume
*does* trip the `dest.top < kCeilingLimit` test (8) six pixels earlier than an
unlit glider at the same altitude.

What the plume does not buy is an early escape. Each of the four threshold
branches tests `mode == kGliderBurning` first and answers with
`wasMode = 0; StartGliderFadingOut(); PlayPrioritySound(kFadeOutSound, ...)` —
`:698`–`:703` at the ceiling, `:711`–`:716` at the floor, `:727`–`:732` at the
left threshold, `:740`–`:745` at the right — pre-empting the
`CheckEscapeUpTwo`/`CheckEscapeUp` calls at `:705`/`:707`, which are the only call
sites of those functions anywhere in the source. So a burning glider fades out 6 px
"early" at the ceiling and can never transit to the room above (§12.1 describes
the same behaviour from the driver's side). The roof is the single exception: the
`(thisBackground == kRoof) && (dest.bottom > kRoofLimit)` branch at `:722`–`:723`
carries no burning guard, and because it sits after the `kFloorLimit` (312) test in
the same `else if` chain while `kRoofLimit` is only 122, a burning glider in a roof
room does reach `CheckRoofCollision`.

### 11.5 `BounceGlider` (`GliderPRO/Sources/Interactions.c:154`–`:167`)

```
 1  BounceGlider(thisGlider, theRect):
 2      glideBounds = thisGlider->dest
 3      if ((theRect->right - glideBounds.left) < (glideBounds.right - theRect->left)):
 4          thisGlider->hVel = theRect->right - glideBounds.left     // push right
 5      else:
 6          thisGlider->hVel = theRect->left - glideBounds.right     // push left
 7      if (foilTotal > 0): PlayPrioritySound(kFoilHitSound, kFoilHitPriority)
 8      else:               PlayPrioritySound(kHitWallSound, kHitWallPriority)
```

The new `hVel` is the **overlap distance** — set, not added. The obvious reading
is that the next `MoveGlider` then teleports the glider exactly clear of the
obstacle in one frame. It does not, because the decay happens *before* the move
rather than after it. `HandleInteraction` runs ahead of `HandleGlider` in the same
frame (`GliderPRO/Sources/Play.c:482` then `:487`, and `:454` then `:460`–`:461`
in the two-player branch), and `MoveGlider` ramps `hVel` toward `hDesiredVel` by
`kHImpulse` = 2 (`GliderPRO/Sources/Player.c:66`–`:77`, clamped so it cannot
overshoot the target) and clamps to ±`kMaxHVel` = 16 (`:96`–`:97`, `:113`–`:114`)
*before* adding it to `dest` (`:102`–`:103`, `:119`–`:120`). With
`hDesiredVel == 0` the displacement on the bounce frame is therefore
`max(min(|overlap|, 18) − 2, 0)` — two pixels short of exactly clear — and an
overlap wider than 18 is spread over several frames at 16 px each, with
`BounceGlider` re-firing and replaying `kFoilHitSound`/`kHitWallSound` on every one
of them. Wide overlaps are not hypothetical: a `kInvisBounce` rect is whatever size
the level author drew, so the push can be as large as about `(rectWidth + 48) / 2`.
Nor is `hDesiredVel` necessarily 0 on the bounce frame — a held direction key
contributes ±5 (`GliderPRO/Sources/Input.c:313`, `:323`) and a fan ±12
(`GliderPRO/Sources/Interactions.c:1211`, `:1215`) — so the ramp target, and hence
the ejection distance, depends on what the player and the room are doing that
frame.

The 2 px shortfall does not normally leave the glider pinned, because
`kInvisBounce` registers with `doScrutinize` = true
(`GliderPRO/Sources/ObjectRects.c:649`–`:651`) and `SectGlider` insets 5 px per
side (`GliderPRO/Sources/Interactions.c:110`–`:115`), so a 2 px residual falls
inside the inset and next frame's overlap test fails. What a port that implements
the stated intent loses is the 2 px offset at every resting position and the extra
`kHitWallSound`/`kFoilHitSound` plays while a wide rect ejects. Used for
`kBounceIt` hot spots (`GliderPRO/Sources/Interactions.c:1591`).

---

## 12. Walls, floor, ceiling and the roof

### 12.1 `CheckGliderInRoom` — the driver (`GliderPRO/Sources/Interactions.c:689`–`:752`)

```
 1  CheckGliderInRoom(thisGlider):
 2      if (mode != kGliderNormal && mode != kGliderFaceLeft &&
 3          mode != kGliderFaceRight && mode != kGliderBurning): return    // :691-694
 4
 5      // ---- vertical chain (first match wins) ----
 6      if (dest.top < kCeilingLimit):                     // 8    :696
 7          if (mode == kGliderBurning):
 8              wasMode = 0;  StartGliderFadingOut;  kFadeOutSound   // :700-702
 9          else if (twoPlayerGame && !onePlayerLeft): CheckEscapeUpTwo(thisGlider)
 10         else:                                     CheckEscapeUp(thisGlider)
 11     else if (dest.bottom > kFloorLimit):               // 312  :709
 12         if (mode == kGliderBurning): ...die...
 13         else if (twoPlayerGame && !onePlayerLeft): CheckEscapeDownTwo else CheckEscapeDown
 14     else if (thisBackground == kRoof && dest.bottom > kRoofLimit):  // 122  :722
 15         CheckRoofCollision(thisGlider)
 16
 17     // ---- horizontal chain (INDEPENDENT of the above) ----
 18     if (dest.left < leftThresh):                       // :725
 19         if (mode == kGliderBurning): ...die...
 20         else if (twoPlayerGame && !onePlayerLeft): CheckEscapeLeftTwo else CheckEscapeLeft
 21     else if (dest.right > rightThresh):                // :738
 22         if (mode == kGliderBurning): ...die...
 23         else if (twoPlayerGame && !onePlayerLeft): CheckEscapeRightTwo else CheckEscapeRight
```

The two chains are **separate `if` statements**, so a glider in a corner is
processed both vertically and horizontally in the same frame — but each chain
only fires its first matching branch. A burning glider dies on contact with four
of these five boundaries — the ceiling, the floor and both side thresholds — since
each of those branches tests `mode == kGliderBurning` before anything else, and
clears `wasMode` first so the burn timer cannot also fire. The roof (line 14) is
the exception: that branch carries no burning guard, so a burning glider in a roof
room reaches `CheckRoofCollision` in the ordinary way (§11.4).

### 12.2 `CheckEscapeUp` (one player) (`GliderPRO/Sources/Interactions.c:244`–`:279`)

```
 1  CheckEscapeUp(thisGlider):
 2      if (topOpen):                                      // :248
 3          if (dest.top < kNoCeilingLimit):               // -10   :250
 4              MoveRoomToRoom(thisGlider, kAbove)         // :252
 5          // no else — keep rising
 6      else if (thisBackground == kDirt):                 // :255
 7          leftTile  = dest.left  >> 6                    // :257  / kTileWide
 8          rightTile = dest.right >> 6                    // :258
 9          if (leftTile >= 0 && leftTile < 8 &&
 10             rightTile >= 0 && rightTile < 8):          // :260-261
 11             if ((tiles[leftTile] == 5 || tiles[leftTile] == 6) &&
 12                 (tiles[rightTile] == 5 || tiles[rightTile] == 6)):   // :263-266
 13                 if (dest.top < kNoCeilingLimit):       // :268
 14                     MoveRoomToRoom(thisGlider, kAbove) // :269
 15                 // no else
 16             else:
 17                 vVel = kCeilingLimit - dest.top        // :272  snap back
 18         else:
 19             vVel = kCeilingLimit - dest.top            // :275  snap back
 20     else:
 21         vVel = kCeilingLimit - dest.top                // :278  snap back
```

The outer chain dispatches on the **room**, not on where the glider is: `topOpen`
(`:248`), else `kDirt` (`:255`), else everything else (`:277`). Only that third arm
is an ordinary solid ceiling. The `dest.top < kNoCeilingLimit` test is an *inner*
guard that appears once inside each of the first two arms (`:250`, `:268`) and has
no `else` in either place, so when the ceiling is open and `dest.top` is between
`kCeilingLimit` (8) and `kNoCeilingLimit` (−10) this function does nothing at all
and the glider simply keeps rising — that 18-px band is the last stretch it coasts
before the room changes. Fusing the two tests into a single
`if (topOpen && dest.top < kNoCeilingLimit)` and hanging the snap-back on *its*
`else` is the one mistake that must not be made: `CheckGliderInRoom` re-enters here
on every frame that `dest.top < 8` (`:696`), so the glider would be re-snapped to
the ceiling line each frame and could never reach −10. Every open-ceilinged room —
`kGarden`, `kMeadow`, `kField`, `kRoof`, `kSky`, `kStratosphere`, `kStars`
(`GliderPRO/Sources/Room.c:1189`–`:1197`, via `topOpen = !DoesRoomHaveCeiling()` at
`Room.c:929`–`:932`) — would be sealed at the top.

The `kDirt` arm is an `else if` on `topOpen`, so it is reached only when the room
*has* a ceiling. That is not a contradiction: `kDirt` (2011) is a built-in
background and is absent from the `DoesRoomHaveCeiling` exception list, so a dirt
room always reports `topOpen == false` and the tile test is exactly what stands in
for its ceiling. Nesting the `kDirt` test *inside* the `topOpen` branch instead
makes the whole dug-tunnel path unreachable. Tiles **5 and 6** are the passable
ceiling tiles for going up, and both the left and right edge tiles must qualify.

The tile-index range check at `:260`–`:261` is load-bearing rather than defensive.
`dest.left` is legitimately negative whenever the glider straddles an open left
side (`leftThresh` is then `kNoLeftWallLimit` = −24, `Room.c:908`–`:919`), and
`dest.right` reaches `kNoRightWallLimit` = 536 on the other side; `>> 6` is an
arithmetic shift on a signed `short`, so those become −1 and 8. The C diverts both
to the snap-back at `:275`; a port that omits the guard indexes `thisTiles[-1]` or
`thisTiles[8]`, and one that "repairs" the omission by clamping to 0..7 reads the
wrong tile and can pass straight through solid dirt.

`vVel = kCeilingLimit - dest.top` is an assignment, not an accumulation, and it
looks like a hard positional snap to `dest.top = 8` — but it is not one, because
`MoveGlider` ramps `vVel` toward `vDesiredVel` *before* adding it to `dest.top`
(`GliderPRO/Sources/Player.c:80`–`:92`, then `:143`). With the usual
`vDesiredVel = kGravity` = 3 and `kVImpulse` = 2, a snap of `s = 8 - dest.top`
lands the glider at `dest.top` = 6 for any `s >= 5`, 7 for `s == 4`, 8 for
`s == 3` (the only exact case), 9 for `s == 2` and 10 for `s == 1`. So for a
glider at `dest.top = -5` the snap of 13 is decayed to 11 and `dest.top` becomes 6,
not 8; 6 is still under the ceiling line, so `CheckEscapeUp` fires again next frame
with `s = 2` and pushes it to 9. Ceiling contact is a two-frame event filtered
through the same ±2 ramp as everything else. A port that writes
`dest.top = kCeilingLimit` directly is off by up to 2 px per contact and loses the
residual `vVel` (11, then 3) that persists into later frames.

### 12.3 `CheckEscapeDown` (`GliderPRO/Sources/Interactions.c:378`–`:446`)

```
 1  CheckEscapeDown(thisGlider):
 2      if (bottomOpen):                                   // :382
 3          if (dest.bottom > kNoFloorLimit):              // 332   :384
 4              MoveRoomToRoom(thisGlider, kBelow)         // :386
 5          // no else — keep falling
 6      else if (thisBackground == kDirt):                 // :389
 7          leftTile  = dest.left  >> 6                    // :391
 8          rightTile = dest.right >> 6                    // :392
 9          if (leftTile >= 0 && leftTile < 8 &&
 10             rightTile >= 0 && rightTile < 8):          // :394
 11             if ((tiles[leftTile] == 2 || tiles[leftTile] == 3) &&
 12                 (tiles[rightTile] == 2 || tiles[rightTile] == 3)):   // :396-397
 13                 if (dest.bottom > kNoFloorLimit):      // :399
 14                     MoveRoomToRoom(thisGlider, kBelow) // :400
 15                 // no else
 16             else:
 17                 groundOrDie()                          // :404-414
 18         else:
 19             groundOrDie()                              // :419-429
 20     else:
 21         groundOrDie()                                  // :434-444
 22
 23  groundOrDie():           // written out in full at all three sites, not a call
 24      if (ignoreGround):                                // :434  a manhole
 25          if (dest.bottom > kNoFloorLimit):             // :436
 26              MoveRoomToRoom(thisGlider, kBelow)        // :437
 27          // no else — keep falling
 28      else:
 29          vVel = kFloorLimit - dest.bottom              // :441
 30          StartGliderFadingOut;  kFadeOutSound          // :442-443
```

The shape is `CheckEscapeUp` (§12.2) with the snap-back replaced by
`groundOrDie`: a three-way `else if` chain on the room — `bottomOpen` (`:382`),
else `kDirt` (`:389`), else solid floor (`:432`) — with the
`dest.bottom > kNoFloorLimit` test as an else-less inner guard at `:384` and `:399`.
`bottomOpen` comes from `bottomOpen = !DoesRoomHaveFloor()`
(`GliderPRO/Sources/Room.c:924`–`:927`), true only for `kSky`, `kStratosphere`,
`kStars` and user rooms carrying the no-floor bounds bit (`Room.c:1143`–`:1163`),
so a `kDirt` room is always `bottomOpen == false` and its dug tunnels live in the
second arm.

Getting this chain wrong is fatal in the literal sense. Terminal free-fall speed is
`kGravity` = 3 px/frame (`GliderPRO/Sources/Player.c:13`, `:92`), so `dest.bottom`
crosses `kFloorLimit` (312) at about 314 and needs roughly another seven frames to
pass `kNoFloorLimit` (332). Through all of those frames `CheckGliderInRoom` calls
this function (`:709`) and the C does *nothing*: the glider visibly sinks through
the floor line and only then transitions. A port that fuses `bottomOpen` with the
332 test and hangs `groundOrDie` on the fused `else` fades the player out on the
first frame past 312, which turns every downward transition through an open floor
into a death.

**Hitting the floor kills you.** There is no landing. The only exceptions are an
open-bottomed room, a `kDirt` tunnel with tiles 2 or 3 under both edges (versus 5
and 6 going up), or the `ignoreGround` flag from a manhole hot spot. Note that in
the `ignoreGround` case the glider must still be past `kNoFloorLimit` (332) — from
312 to 332 it simply keeps falling.

`ignoreGround` is what makes the dirt arm's tile-mismatch `else` non-fatal: at
`:404` the C runs the same `groundOrDie` pair as the solid-floor arm, so a manhole
placed in a dirt room over tiles that are not 2 or 3 still drops the player to the
room below rather than killing them. The tile-index range check at `:394` matters
for the same reason as in §12.2, except that here its `else` at `:417`–`:430` is a
third copy of `groundOrDie` — out of range the glider dies (or falls through, with
`ignoreGround`), it does not snap.

### 12.4 `CheckEscapeLeft` (`GliderPRO/Sources/Interactions.c:572`–`:595`)

```
 1  CheckEscapeLeft(thisGlider):
 2      if (leftThresh == kLeftWallLimit):                 // 12: there IS a wall  :576
 3          if (ignoreLeft):                               // :578
 4              if (dest.left < kNoLeftWallLimit):         // -24   :580
 5                  MoveRoomToRoom(thisGlider, kToLeft)    // :581
 6              // no else — the wall is inert this frame
 7          else:
 8              if (foilTotal > 0): kFoilHitSound          // :585-586
 9              else:               kHitWallSound          // :587-588
 10             offset = kLeftWallLimit - dest.left        // :589  penetration
 11             hVel = -hVel + offset                     // :590  reflect + eject
 12     else:                                              // open room  :593
 13         MoveRoomToRoom(thisGlider, kToLeft)            // :594
```

The bounce is the `else` of `ignoreLeft` alone (`:583`), not of a fused
`ignoreLeft && dest.left < kNoLeftWallLimit`, and the positional test at `:580` has
no `else`. That distinction is what makes doors and windows work. `ignoreLeft` is
re-set every frame the glider overlaps a `kIgnoreLeftWall` hot spot (§13), and the
glider has to carry `dest.left` all the way from 11 down past −24 before the exit
fires; through that whole run the wall must be inert — no sound, no reflection.
Fuse the two tests and the first frame at `dest.left = 10` plays `kHitWallSound`
and sets `hVel = -hVel + 2`, throwing the glider back out of the doorway it is
standing in, so left and right exits through doors and windows never fire at all.

Note `hVel = -hVel + offset` (not `-hVel - offset` as in `GliderHitTop`): for a
leftward `hVel = -6` at `dest.left = 8`, `offset = 4`, giving `hVel = 10`. So the
bounce reflects the speed and adds the penetration depth. **You bounce off walls
faster than you hit them.** There is no wall damage — walls are perfectly safe
unless you are on fire.

### 12.5 `CheckEscapeRight` (`GliderPRO/Sources/Interactions.c:662`–`:685`)

Mirror, with `kRightWallLimit` (500) and `kNoRightWallLimit` (536), and
`offset = kRightWallLimit - dest.right; hVel = -hVel + offset;` (`:679`–`:680`).
"Mirror" includes the nesting: `if (ignoreRight)` at `:668` wraps an else-less
`if (dest.right > kNoRightWallLimit)` at `:670`, and the sound-and-bounce block is
the `else` of `ignoreRight` at `:673`, so the right wall is equally inert while the
glider carries `dest.right` from 501 out past 536.

### 12.6 The two-player variants

`CheckEscapeUpTwo` (`:171`–`:240`), `CheckEscapeDownTwo` (`:283`–`:374`),
`CheckEscapeLeftTwo` (`:509`–`:568`), `CheckEscapeRightTwo` (`:599`–`:658`) add
the limbo handshake. Each keeps its one-player sibling's structure *unchanged* —
the same room-type dispatch, the same else-less `kNo*Limit` inner guards, the same
tile-range checks, the same snap-back and `groundOrDie` arms — and substitutes a
three-armed handshake for each `MoveRoomToRoom` call. Using Up as the example, the
substitution is:

```
 1  // in place of each  MoveRoomToRoom(thisGlider, kAbove)  in §12.2:
 2  if (otherPlayerEscaped == kNoOneEscaped):         // :179  nobody waiting yet
 3      otherPlayerEscaped = kPlayerEscapedUp         // -4     :181
 4      RefreshScoreboard(kEscapedTitleMode)          // :182
 5      FlagGliderInLimbo(thisGlider, true)           // :183  plays "follow me"
 6  else if (otherPlayerEscaped == kPlayerEscapedUp): // :185  waiting at THIS exit
 7      otherPlayerEscaped = kNoOneEscaped            // :187  clear the latch
 8      MoveRoomToRoom(thisGlider, kAbove)            // :188  both go
 9  else:                                             // :190  waiting somewhere else
10      PlayPrioritySound(kDontExitSound, kDontExitPriority)   // :192
11      offset = kNoCeilingLimit - dest.top           // :193
12      vVel = -vVel + offset                         // :194
```

So in a two-player game the **first** glider to reach an exit goes into limbo at
the boundary and the room does not change; the **second** glider to reach the
same exit triggers the transition for both. A glider that reaches a *different*
exit while the partner waits gets a `kDontExitSound` and bounces off an invisible
barrier at the `kNo*Limit` plane.

Three details of that chain are easy to lose. The middle arm tests the exit's *own*
code rather than being a bare `else`, which is what makes `kPlayerIsDeadForever`
(−69, set by `OffAMortal` at `GliderPRO/Sources/Player.c:1602`) and the stairs,
transporter, duct and mail codes (§15.6) fall through to the third arm instead of
teleporting a lone glider out of the room; each function substitutes its own
constant (`kPlayerEscapedUp` −4, `kPlayerEscapedDown` −5,
`kPlayerEscapedLeft` −3, `kPlayerEscapedRight` −2). The latch is cleared at `:187`
*before* the transition, and dropping that line leaves `otherPlayerEscaped` armed
into the next room, where the first glider to touch any exit takes the middle arm
and leaves without its partner. And the third arm is the "wrong exit" bounce, not
the "has not crossed the plane yet" case — it sits *inside* the `kNo*Limit` guard,
so `offset` always points back into the room (positive going up and left, negative
going down and right) and the glider really is thrown off the barrier; below the
plane the C is silent and does nothing, exactly as in §12.2.

There is no `onePlayerLeft` test in any of the four functions. That guard lives in
the caller: `CheckGliderInRoom` picks the `*Two` variant only under
`(twoPlayerGame) && (!onePlayerLeft)` (`:704`, `:717`, `:733`, `:746`), so a
`onePlayerLeft` arm transcribed into the handshake would be dead code.

"Same structure as the one-player sibling" is exact for Up, but the horizontal pair
carry the handshake **twice**: once inside `leftThresh == kLeftWallLimit` under
`ignoreLeft`, gated on `dest.left < kNoLeftWallLimit` (`:519`–`:536`), and once in
the open-side `else` with no positional gate at all (`:550`–`:566`), mirrored at
`:609`–`:626` and `:640`–`:656`. The ordinary wall arm in between (`:538`–`:546`,
`:628`–`:636`) is untouched, and still uses `kLeftWallLimit`/`kRightWallLimit` with
`kFoilHitSound`/`kHitWallSound` — do not let the handshake's `kNoLeftWallLimit` and
`kDontExitSound` bleed into it. `CheckEscapeUpTwo` and `CheckEscapeDownTwo` carry
the handshake twice as well, once in the open arm and once in the `kDirt` arm
(`:211`–`:230`, `:323`–`:342`). One asymmetry survives all of this and must not be
factored away: `CheckEscapeDownTwo` has the tile-range check at `:315`–`:316` but
**no `else` for it**, so out of range the glider just keeps falling, where
`CheckEscapeDown` runs `groundOrDie` (`:417`–`:430`).

`otherPlayerEscaped` is a single global that is read and written within the frame,
and §12.1's two chains are separate `if` statements, so a glider in a corner can
run `CheckEscapeUpTwo` and then `CheckEscapeLeftTwo` in the same frame with the
second call seeing the first call's write. The vertical-then-horizontal order is
load-bearing.

Note what the limbo branch does *not* contain: there is no `dontDraw` write in it.
No line of `Interactions.c` assigns that field at all, so the glider that reaches
a wall exit first goes on being rendered for the entire wait, frozen at the
position that tripped the test — and since the `kNo*Limit` planes sit half a
glider beyond the wall (`kNoCeilingLimit` −10, `kNoFloorLimit` 332,
`kNoLeftWallLimit` −24, `kNoRightWallLimit` 536, against a 48 x 20 glider) roughly
half of it is still inside the room, shadow included. A port that suppresses
drawing on mode 21 makes the waiting player vanish from the room edge; see §8.21.

### 12.7 `CheckRoofCollision` (`GliderPRO/Sources/Interactions.c:450`–`:505`)

Only on `thisBackground == kRoof`, only when `dest.bottom > kRoofLimit` (122).

```
 1  CheckRoofCollision(thisGlider):
 2      offset = (dest.left + kHalfGliderWide) >> 6        // :454  centre tile index
 3      if (offset < 0 || offset > 7 || sliding): return   // :455 (inverted)
 4      tileOver = tiles[offset]
 5      switch (tileOver):
 6        case 1:   // roof sloping one way
 7            if (((dest.left + kHalfGliderWide) - (offset << 6)) > (250 - dest.bottom)):
 8                die                                       // :463
 9        case 2:
10            if (((dest.left + kHalfGliderWide) - (offset << 6)) > (186 - dest.bottom)):
11                die                                       // :473
12        case 5:
13            if ((64 - ((dest.left + kHalfGliderWide) - (offset << 6))) > (186 - dest.bottom)):
14                die                                       // :483
15        case 6:
16            if ((64 - ((dest.left + kHalfGliderWide) - (offset << 6))) > (250 - dest.bottom)):
17                die                                       // :493
18        default:
19            die                                           // :500  flat roof = solid
20      // "die" == vVel = kFloorLimit - dest.bottom; StartGliderFadingOut; kFadeOutSound
```

In the C this is an `else if` chain on `tileOver`, not a `switch`, and there is no
named temporary for the in-tile x — the expression
`(thisGlider->dest.left + kHalfGliderWide) - (offset << 6)` is written out in
full in each branch. Call that value *xInTile* (my name, not the original's): it
runs 0..63 across the 64-px-wide tile, and each branch asks
`xInTile > (pitch - dest.bottom)` (or `64 - xInTile > (pitch - dest.bottom)` for
the mirrored tiles).

| `tileOver` | Test | Pitch constant | Slope |
| --- | --- | --- | --- |
| 1 | `xInTile > (250 - dest.bottom)` | 250 (`:461`) | down to the right |
| 2 | `xInTile > (186 - dest.bottom)` | 186 (`:471`) | down to the right, steeper start |
| 5 | `(64 - xInTile) > (186 - dest.bottom)` | 186 (`:481`) | down to the left |
| 6 | `(64 - xInTile) > (250 - dest.bottom)` | 250 (`:491`) | down to the left |
| anything else (including 0, 3, 4, 7) | unconditional | — | flat / solid: kills on contact |

`sliding` grants immunity (the `!thisGlider->sliding` term at `:455`), which is
how a glider on a greased roof slides along it instead of dying.
---

## 13. Hot spots: everything the world does to the player

Objects act on the player through the `hotSpots[]` array of `hotObject` records
(`GliderPRO/Headers/GliderStructs.h:218`–`:225`):

```c
typedef struct {
	Rect		bounds;			// room-local (same space as gliderType.dest); add playOrigin only to draw
	short		action;			// one of the kIgnoreIt..kSoundIt action codes
	short		who;			// index of the owning object
	Boolean		isOn, stillOver;
	Boolean		doScrutinize;	// use the 5-px-inset hit box
} hotObject, *hotPtr;
```

(The struct in `GliderStructs.h` carries no comments of its own; those are ours.)
The coordinate space of `bounds` is the one thing here a port can get
catastrophically wrong, so it is worth the evidence. `AddActiveRect` stores the
rect verbatim (`GliderPRO/Sources/ObjectRects.c:283`) from geometry built purely
out of `theObject.data.*.topLeft`, and `SectGlider`, `GliderInRect` and
`GliderHitTop` compare it against an un-offset copy of `thisGlider->dest`
(`GliderPRO/Sources/Interactions.c:101`, `:134`, `:54`). `playOrigin` is added only
when a hot rect is turned into pixels (`GliderPRO/Sources/Interactions.c:1001`–`:1002`,
`GliderPRO/Sources/Grease.c:282`–`:287`), and `Grease.c:66` correspondingly
*subtracts* it to convert a screen-space spill rect back into hot-spot space. A
port that offsets hot rects by `playOrigin` before testing them shifts the entire
interaction layer by one origin, and nothing in the room can be triggered at all.

`kMaxHotSpots` = **56** (`GliderPRO/Headers/GliderDefines.h:259`).

### 13.1 The action enum

`GliderPRO/Headers/GliderDefines.h:282`–`:309`. Every value, with whether it
touches the player's motion:

| # | Symbol | Effect on the player |
| --- | --- | --- |
| 0 | `kIgnoreIt` | nothing |
| 1 | `kLiftIt` | `vDesiredVel = kFloorVentLift` (−6) |
| 2 | `kDropIt` | `vDesiredVel = kCeilingVentDrop` (+8) |
| 3 | `kPushItLeft` | `hDesiredVel += −kFanStrength` (−12) |
| 4 | `kPushItRight` | `hDesiredVel += +kFanStrength` (+12) |
| 5 | `kDissolveIt` | death; with foil, death anyway unless the contact is a side hit, which reflects `hVel` (§11.3) |
| 6 | `kRewardIt` | `HandleRewards` (see §14) |
| 7 | `kMoveItUp` | up the stairs; requires `!heldRight` and `GliderInRect` |
| 8 | `kMoveItDown` | down the stairs; requires `!heldLeft` and `GliderInRect` |
| 9 | `kSwitchIt` | nothing to the player |
| 10 | `kShredIt` | `FlagGliderShredding`, unless foil absorbs it |
| 11 | `kStrumIt` | nothing (sound only) |
| 12 | `kTriggerIt` | nothing |
| 13 | `kBurnIt` | `FlagGliderBurning`, unless foil (then `vDesiredVel = −6`) |
| 14 | `kSlideIt` | `sliding = true; vVel = who->bounds.top − dest.bottom;` |
| 15 | `kTransportIt` | `StartGliderTransporting` |
| 16 | `kIgnoreLeftWall` | `ignoreLeft = true` (one frame) |
| 17 | `kIgnoreRightWall` | `ignoreRight = true` (one frame) |
| 18 | `kMailItLeft` | `StartGliderMailingIn`, mode 14; requires `GliderInRect` and a `facing`/`tipped` combination |
| 19 | `kMailItRight` | `StartGliderMailingIn`, mode 16; requires `GliderInRect` and the opposite combination |
| 20 | `kDuctItDown` | `StartGliderDuctingDown`, mode 11; requires `GliderInRect` |
| 21 | `kDuctItUp` | `StartGliderDuctingUp`, mode 12; requires `GliderInRect` |
| 22 | `kMicrowaveIt` | destroys bands / battery / foil; requires `GliderInRect` |
| 23 | `kIgnoreGround` | `ignoreGround = true` (one frame) |
| 24 | `kBounceIt` | `BounceGlider` |
| 25 | `kChimeIt` | nothing |
| 26 | `kWebIt` | `WebGlider`; requires `GliderInRect` |
| 27 | `kSoundIt` | nothing |

(`kLgTrigger` shares the `kTriggerIt` case at `GliderPRO/Sources/Interactions.c:1351`–`:1352`.)

### 13.2 `kSlideIt` — the grease/slider effect (`GliderPRO/Sources/Interactions.c:1376`–`:1379`)

```
1  case kSlideIt:
2      thisGlider->sliding = true                                  // :1377
3      thisGlider->vVel = who->bounds.top - thisGlider->dest.bottom  // :1378
```

Two frames of consequence:

1. `vVel` is **set** (not added) to the signed distance from the glider's bottom
   edge to the top of the slide surface. It is tempting to read that as a
   teleport onto the surface, but it is not one, for two separate reasons.
   First, the write is attenuated: `HandleInteraction` runs before `HandleGlider`
   in the same frame (`GliderPRO/Sources/Play.c:482` then `:487`), and
   `MoveGlider` runs its vertical ramp (`GliderPRO/Sources/Player.c:80`–`:91`)
   before it adds `vVel` to `dest` (`:129`–`:146`), so with the usual
   `vDesiredVel` of `+3` a `vVel` of `-d` is ramped to `2 - d` before it is
   applied. That arithmetic has a fixed point: for a glider whose bottom sits `d`
   px below the surface top the displacement is `2 - d` for *every* `d`, so the
   bottom always lands exactly 2 px below `who->bounds.top`, and it then holds
   there, because the next frame writes `vVel = -2`, the ramp takes that to 0,
   and nothing moves. (If something else wrote `vDesiredVel` that frame — helium
   `-4`, `kLiftIt` `-6`, `kDropIt` `+8` — the 2 px is applied towards *that*
   value instead, so the resting offset flips sign for a negative
   `vDesiredVel`.) Second, the snap is never downwards. `kSlideIt` rects are
   registered with `doScrutinize = false`
   (`GliderPRO/Sources/ObjectRects.c:696`, `:715`, `:731`, against the signature
   at `:277`), so `SectGlider` (`GliderPRO/Sources/Interactions.c:118`–`:127`)
   only reports a hit once `theRect->top <= glideBounds.bottom` — that is, once
   the glider's bottom is already at or below the surface top. The expression
   `who->bounds.top - thisGlider->dest.bottom` is therefore always `<= 0`, and a
   glider falling onto grease from 40 px up does not see the hot spot at all
   until it has arrived.
2. `sliding = true` does three things: selects the sliding sprite
   (`gliderSrc[30]` facing left at `GliderPRO/Sources/Player.c:157`,
   `gliderSrc[29]` facing right at `:179`), grants immunity from
   `CheckRoofCollision` (`GliderPRO/Sources/Interactions.c:457`), and is consumed
   by `MoveGliderNormal` on the next frame (`Player.c:159`, `:181`).

Slide surfaces come from three object classes:

| Object | Hot rect | Where |
| --- | --- | --- |
| `kGreaseRt` (0x28) unspilled | `QSetRect(&bounds, 0, -2, length - 5, 0)` then offset `(31, 27)` then `+ topLeft` | `GliderPRO/Sources/ObjectRects.c:681`–`:698` |
| `kGreaseLf` (0x29) unspilled | `QSetRect(&bounds, -length + 5, -2, 0, 0)` then offset `(1, 27)` then `+ topLeft` | `GliderPRO/Sources/ObjectRects.c:700`–`:717` |
| `kSlider` (0x2F) | `QSetRect(&bounds, 0, 0, length, 16)` then `+ topLeft` | `GliderPRO/Sources/ObjectRects.c:727`–`:732` |
| spilled grease (dynamic) | a 2 x 2 rect grown 2 px/frame — see below | `GliderPRO/Sources/Grease.c:60`–`:68`, `:121`–`:130` |

**Note the 2-pixel-tall slide rect** for unspilled grease: it sits at
`topLeft.v + 25 .. +27`, i.e. the *lip* of the grease jar. `length` is
`objectType.data.c.length` (`GliderPRO/Headers/GliderStructs.h:30`, commented
"grease spill").

Spilled grease is installed dynamically by `HandleGrease`
(`GliderPRO/Sources/Grease.c:43`–`:133`):

```
 1  // when a falling grease jar finishes its 4-frame tip animation (frame >= 3):
 2  grease[i].mode = kGreaseSpreading
 3  hotSpots[grease[i].hotNum].action = kSlideIt                   // Grease.c:66
 4  hotSpots[grease[i].hotNum].isOn   = true
 5  if (grease[i].isRight): QSetRect(&src,  0, -2, 2, 0)           // 2x2 seed
 6  else:                   QSetRect(&src, -2, -2, 0, 0)
 7  QOffsetRect(&src, -playOriginH, -playOriginV)
 8  QOffsetRect(&src, grease[i].start, grease[i].dest.bottom)
 9  hotSpots[grease[i].hotNum].bounds = src
10  // then, each frame in kGreaseSpreading mode:
11  if (isRight): bounds.right += 2;  start += 2;  until start >= stop   // Grease.c:121-127
12  else:         bounds.left  -= 2;  start -= 2;  until start <= stop
```

so a grease slick extends at **2 px/frame** in the spill direction until it
reaches `stop`. The jar's tip animation is 4 frames of a 32 x 27 sprite from
`savedMaps`, and the jar itself shifts ±2 px/frame while tipping
(`GliderPRO/Sources/Grease.c:83`–`:86`).

### 13.3 `kWebIt` — spider webs (`GliderPRO/Sources/Interactions.c:1736`–`:1776`)

```
 1  WebGlider(thisGlider, webBounds):
 2      if (mode == kGliderBurning && GliderInRect(thisGlider, webBounds)):
 3          wasMode = 0                                  // :1743
 4          StartGliderFadingOut(thisGlider);  kFadeOutSound
 5          return                                       // burning + inside = instant death
 6
 7      hDist = ((webBounds->right  - dest.right)  + (webBounds->left - dest.left)) >> 3   // :1749-1750
 8      vDist = ((webBounds->bottom - dest.bottom) + (webBounds->top  - dest.top))  >> 3   // :1751-1752
 9
10      if (hDesiredVel != 0):                           // :1754  player is struggling
11          if (evenFrame):                              // :1756  only every other frame
12              hVel = hDist                             // :1758
13              vVel = vDist                             // :1759
14              PlayPrioritySound(kWebTwangSound, kWebTwangPriority)
15      else:                                            // :1763  player is passive
16          hDesiredVel = 0                              // :1765  (already 0)
17          vDesiredVel = 0                              // :1766  <-- CANCELS GRAVITY
18
19      wasMode++                                        // :1769
20      if (wasMode >= kKillWebbedGlider):               // 150
21          wasMode = 0
22          StartGliderFadingOut(thisGlider);  kFadeOutSound   // :1773-1774
```

Lines 2–5 are dead code. `WebGlider` has exactly one caller, `:1605`, and that
call is already guarded by `mode != kGliderBurning` at `:1604`, so the burning
branch inside the function can never be entered. A burning glider in a web is
actually killed by the caller's own `else if ((mode == kGliderBurning) &&
GliderInRect(...))` at `:1606`–`:1611`; the effect is identical but the
containment gate that matters for the *live* glider lives at `:1603`, not here
(§11.2).

`hDist` and `vDist` are **twice the offset from the glider's centre to the web's
centre, divided by 8** — i.e. `((webCentre - gliderCentre) * 2) >> 3` =
`(webCentre - gliderCentre) / 4`. So a struggling glider is snapped a quarter of
the way toward the web's centre every *even* frame, producing the twanging
oscillation. A passive glider has gravity cancelled (line 17) and hangs
motionless. Either way `wasMode` counts up and the glider dies after **150
frames (5 s)**.

`evenFrame` is a global toggled once per frame at
`GliderPRO/Sources/Play.c:435`.

### 13.4 `kBurnIt`, `kDissolveIt`, `kShredIt` — the three deadly actions

| Action | Without foil | With foil |
| --- | --- | --- |
| `kBurnIt` (`Interactions.c:1356`) | `FlagGliderBurning` | `vDesiredVel = kFloorVentLift` (−6), `kSizzleSound`, `foilTotal--` |
| `kDissolveIt` (`Interactions.c:1218`) | `StartGliderFadingOut` + `kFadeOutSound` | `GliderHitTop` — a contact still overlapping after the `wasHVel` un-sweep counts as vertical and is **fatal even with foil** (`StartGliderFadingOut` + `kFadeOutSound`, `Interactions.c:1225`–`:1226`); only a side hit is survivable, reflecting `hVel` at a cost of 2 foil per frame of contact (§11.3) |
| `kShredIt` (`Interactions.c:1324`) | `FlagGliderShredding` (requires `GliderInRect` and `mode != kGliderShredding`) | one foil is consumed instead |

`foilTotal` reaching 0 triggers `StartGliderFoilLosing`
(`GliderPRO/Sources/Interactions.c:83`–`:84` inside `GliderHitTop`, `:1233`–`:1234`
in the `kDissolveIt` caller, `:1334`–`:1335` in `kShredIt`).

The shredder hot rect (`GliderPRO/Sources/ObjectRects.c:940`–`:950`):

```
1  bounds = srcRects[kShredder]                     // 0,0,73,22
2  bounds.bottom = bounds.top + kShredderActiveHigh // 40   ObjectRects.c:19
3  bounds.right += kGliderWide                      // +48 -> 121 wide
4  ZeroRectCorner(&bounds)
5  bounds += topLeft
6  QOffsetRect(&bounds, -24, -36)
7  action = kShredIt;  isOn = data.g.state;  doScrutinize = true
```

so the shredder's active zone is **121 x 40** and offset up-left by (24, 36) from
the graphic — it reaches above and to the left of the visible shredder so the
glider is caught as it approaches.

### 13.5 `kMailItLeft` / `kMailItRight` — the facing/tipped gate

`GliderPRO/Sources/Interactions.c:1424` (comment: "mailbox open to right") and
`:1467` ("mailbox open to left"):

```
kMailItLeft  requires: GliderInRect(thisGlider, &who->bounds)                      // :1431
                       && (facing == kFaceRight && !tipped) || (facing == kFaceLeft  && tipped)
kMailItRight requires: GliderInRect(thisGlider, &who->bounds)                      // :1474
                       && (facing == kFaceLeft  && !tipped) || (facing == kFaceRight && tipped)
```

The `GliderInRect` term is easy to miss and gameplay-visible: the glider must be
*wholly* inside the 72 x 40 rect before it is mailed, not merely overlapping it
(§11.2).

Both conditions reduce to "**you must be travelling in the correct direction**",
because `tipped` means "flying opposite to `facing`". `kMailItLeft` (a mailbox
whose slot opens rightward) needs you moving right; `kMailItRight` needs you
moving left. A glider drifting with no key held has `tipped == false`, so its
`facing` alone decides.

The mailbox hot rects (`GliderPRO/Sources/ObjectRects.c:751`–`:773`) are 72 x 40:
`kMailboxLf` → `QSetRect(&bounds, -72, 0, 0, 40)` offset `(30, 16)`;
`kMailboxRt` → `QSetRect(&bounds, 0, 0, 72, 40)` offset `(79, 16)`. Both are only
installed when `data.d.who != 255`, i.e. when the mailbox is actually linked to
another one.

### 13.6 `kMoveItUp` / `kMoveItDown` — stairs

```
kMoveItUp   (Interactions.c:1251): requires !heldRight && GliderInRect(thisGlider, &who->bounds)
kMoveItDown (Interactions.c:1286): requires !heldLeft  && GliderInRect(thisGlider, &who->bounds)
```

The `heldRight`/`heldLeft` guards let you fly *past* a staircase without being
grabbed: to go up you must not be pressing right, to go down you must not be
pressing left. Hot rects: `kUpStairs` → **112 x 32** at `topLeft`
(`GliderPRO/Sources/ObjectRects.c:734`–`:740`); `kDownStairs` →
`QSetRect(&bounds, -80, -56, 0, 0)` offset `(srcRects[kDownStairs].right, 170)`
then `+ topLeft` = **80 x 56** (`GliderPRO/Sources/ObjectRects.c:742`–`:749`).

### 13.7 `kDuctItDown` / `kDuctItUp` — transporter ducts

Hot rects (`GliderPRO/Sources/ObjectRects.c:775`–`:797`):
`kFloorTrans` → `QSetRect(&bounds, 0, -48, 76, 0)` offset
`(-8, RectTall(srcRects[kFloorTrans]))` — 76 x 48 sitting *above* the floor duct;
`kCeilingTrans` → `QSetRect(&bounds, 0, 0, 76, 48)` offset `(-8, 0)` — 76 x 48
below the ceiling duct. Both actions require `GliderInRect`
(`GliderPRO/Sources/Interactions.c:1517`, `:1550`) — the glider must be wholly
inside the 76 x 48 rect, not merely overlapping it (§11.2) — and `kDuctItUp`
additionally requires `!who->stillOver` and sets `stillOver = true` in the
one-player branch (`:1543`+).

### 13.8 `CheckForHotSpots` and `stillOver`

`GliderPRO/Sources/Interactions.c:1627`–`:1687`:

```
1  CheckForHotSpots():
2      for i in 0 .. nHotSpots-1:
3          if (!hotSpots[i].isOn): continue
4          hit = false
5          (for the live glider(s):)
6              if (SectGlider(glider, &hotSpots[i].bounds, hotSpots[i].doScrutinize)):
7                  HandleHotSpotCollision(glider, &hotSpots[i], i)
8                  hit = true
9          if (!hit): hotSpots[i].stillOver = false
```

`stillOver` is a **latch**: actions that must fire only once per entry
(`kSwitchIt`, `kStrumIt`, `kChimeIt`, `kSoundIt`, `kMicrowaveIt`, `kDuctItUp`)
check it at the top of their case and set it at the end. It is cleared the first
frame the glider stops overlapping. `FlagStillOvers`
(`GliderPRO/Sources/Interactions.c:1715`–`:1732`) force-sets it for every hot spot
the glider currently overlaps and is called on duct arrival
(`GliderPRO/Sources/Player.c:954`) to prevent immediate re-triggering.

`HandleInteraction` (`GliderPRO/Sources/Interactions.c:1691`–`:1711`):

```
1  HandleInteraction():
2      CheckForHotSpots()
3      (for each live glider) CheckGliderInRoom(glider)
```

Order matters: hot spots run **before** the wall checks, so `ignoreLeft` etc. are
already set when `CheckEscapeLeft` reads them in the same frame.

### 13.9 Dynamic objects (`GliderPRO/Sources/Dynamics.c:34`–`:74`)

```
 1  CheckDynamicCollision(thisGlider, who, offset):
 2      dinahRect = dinahs[who].dest
 3      if (offset): QOffsetRect(&dinahRect, -playOriginH, -playOriginV)
 4      if (!SectGlider(thisGlider, &dinahRect, true)): return      // always scrutinize
 5      if (mode not in {Normal, FaceLeft, FaceRight, Burning, GoingFoil, LosingFoil}): return
 6      if (foilTotal > 0 || mode == kGliderLosingFoil):
 7          if (IsRectLeftOfRect(&dinahRect, &thisGlider->dest)):
 8              thisGlider->hDesiredVel =  kShoveVelocity      // +8   Dynamics.c:54
 9          else:
10              thisGlider->hDesiredVel = -kShoveVelocity      // -8   Dynamics.c:56
11          if (dinahs[who].vVel < 0):
12              thisGlider->vDesiredVel = dinahs[who].vVel     // inherit upward motion
13          PlayPrioritySound(kFoilHitSound, kFoilHitPriority)
14          if (evenFrame && foilTotal > 0):
15              foilTotal--
16              if (foilTotal <= 0): StartGliderFoilLosing(thisGlider)
17      else:
18          StartGliderFadingOut(thisGlider);  kFadeOutSound
```

Two things worth flagging:

* Foil is consumed only on **even frames**, so a foil-clad glider pinned against a
  moving object loses 1 foil per 2 frames (15/s), i.e. `kFoilSupply` (8) buys
  about half a second of grinding.
* Line 12 lets an *ascending* dynamic object carry the glider up by writing its
  velocity into `vDesiredVel` — a descending one does not push you down.

`IsRectLeftOfRect` (`GliderPRO/Sources/RectUtils.c:181`–`:190`) has a C
precedence quirk that changes the shove direction:

```c
offset = (rect1->right - rect1->left) - (rect2->right - rect2->left) / 2;
if ((rect1->left) < (rect2->left + offset))
	return (true);
```

`/ 2` binds only to the second width, so `offset = width1 - width2/2`, almost
certainly intended as `(width1 - width2) / 2`. For the glider (width 48) versus a
dinah of width `w`, the actual test is `dinah.left < glider.left + w - 24`.
**This is a bug in the original and a Go port must reproduce it verbatim** or
foil shoves will point the wrong way for some object widths.

---

## 14. Rewards and inventory

`HandleRewards` (`GliderPRO/Sources/Interactions.c:756`–`:981`). Only the
player-visible effects are listed; the object bookkeeping (`SetObjectState`,
`AddSparkle`, etc.) belongs to another document.

| Object (case at line) | Score | Player-state effect |
| --- | --- | --- |
| `kRedClock` (`:766`) | `theScore += 100` (`kRedClockPoints`) | `AddFlyingPoint(&bounds, 100, hVel/2, vVel/2)` then `hVel /= 4; vVel /= 4;` — `kBeepsSound` |
| `kBlueClock` (`:782`) | `+= 300` (`kBlueClockPoints`) | same `/4` velocity damping — `kBuzzerSound` |
| `kYellowClock` (`:798`) | `+= 500` (`kYellowClockPoints`) | same — `kDingSound` |
| `kCuckoo` (`:814`) | `+= 1000` (`kCuckooClockPoints`) | same — `kCuckooSound`, `StopPendulum` |
| `kPaper` (`:831`) | — | `hVel /= 2; vVel /= 2;` and **`mortals++`** (a second `mortals++` if `twoPlayerGame && !onePlayerLeft`) |
| `kBattery` (`:850`) | — | `hVel /= 2; vVel /= 2;` then `batteryTotal += kBatterySupply` if already positive, else `= kBatterySupply` (50); +50 again in two-player |
| `kBands` (`:872`) | — | `hVel /= 2; vVel /= 2;` then `bandsTotal += kBandsSupply` (8); doubled in two-player |
| `kGreaseRt` / `kGreaseLf` (`:891`–`:892`) | — | `SpillGrease(...)` — starts the slick |
| `kFoil` (`:900`) | — | `hVel /= 2; vVel /= 2;` then `foilTotal += kFoilSupply` (8) then `StartGliderFoilGoing(thisGlider)`; doubled in two-player |
| `kInvisBonus` (`:919`) | `+= data.c.points` | same `/4` velocity damping as the clocks |
| `kStar` (`:933`) | `+= 5000` (`kStarPoints`) | `StopStar`, `numStarsRemaining--`, `FlagGameOver()` at ≤ 0 |
| `kSparkle` (`:953`) | — | nothing |
| `kHelium` (`:956`) | — | `hVel /= 2; vVel /= 2;` then `batteryTotal -= kHeliumSupply` if already negative, else `= -kHeliumSupply` (−150); −150 again in two-player |
| `kSlider` (`:978`) | — | nothing (handled as `kSlideIt` elsewhere) |

Score constants: `kRoomVisitScore` 100 (`GliderDefines.h:536`),
`kRedClockPoints` 100, `kBlueClockPoints` 300, `kYellowClockPoints` 500,
`kCuckooClockPoints` 1000, `kStarPoints` 5000 (`GliderDefines.h:537`–`:541`).

The velocity damping on a clock pickup is a deliberate "thunk": grabbing a clock
divides both velocity components by 4 (integer division, so it truncates toward
zero), and half of the pre-damping velocity is transferred to the flying score
number: in `kCuckoo`, `AddFlyingPoint(&bounds, 1000, hVel / 2, vVel / 2)` (`:822`)
runs *before* `hVel /= 4; vVel /= 4;` (`:823`–`:824`). Paper (an extra life)
divides by 2 instead.

So do all four supply pickups, and this is the easiest effect in the section to
overlook because nothing else in those cases hints at it: `kBattery`
(`:858`–`:859`), `kBands` (`:880`–`:881`), `kFoil` (`:908`–`:909`) and `kHelium`
(`:964`–`:965`) each halve `hVel` and `vVel` inside the same
`if (SetObjectState(...))` guard, in the fixed order `PlayPrioritySound` →
`RestoreFromSavedMap` → `AddSparkle(&bounds)` → `hVel /= 2` → `vVel /= 2` →
inventory update. Unlike the clocks they discard the other half; there is no
`AddFlyingPoint` to receive it. Collecting a battery while battery-thrusting
therefore halves your speed on the pickup frame — a thunk that shapes the
trajectory for many frames afterwards, and one that interacts with the ±16
`kMaxHVel` clamp. The division is C `/` on a `short` and truncates toward zero, so
`hVel == -1` becomes `0`; Go's `int16(-1) / 2` matches, but "tidying" it into
`>> 1` gives −1 and a permanent drift.

`HandleMicrowaveAction` (`GliderPRO/Sources/Interactions.c:1159`–`:1194`) is
reached only when `GliderInRect` holds (`:1582`), i.e. when the glider is wholly
inside the microwave's rect and not merely touching it (§11.2), and then returns
immediately unless `!who->stillOver` (`:1164`–`:1165`). It reads a bitmask:

| Bit | Effect |
| --- | --- |
| `0x0001` | `bandsTotal = 0` |
| `0x0002` | `batteryTotal = 0` |
| `0x0004` | `foilTotal = 0` and `StartGliderFoilLosing(thisGlider)` |

and plays `kMicrowavedSound` if anything was destroyed.

---

## 15. Death and respawn

### 15.1 The three ways to die

| Cause | Path | Frames until the life is lost |
| --- | --- | --- |
| Anything that calls `StartGliderFadingOut` (floor, wall while burning, dissolve, roof, web timeout, burn timeout, dynamic object, suicide) | mode 2, `frame` 15 → −1 | **17** |
| Shredder | mode 20 | fall + 20 grind frames + **68** delay |
| Star collected / both players dead | `FlagGameOver()` directly | `kNumCountDownFrames` = 16 |

There is no health, no damage, no hit points, and **no invincibility period after
respawning**. The only thing resembling invincibility is that during the 16-frame
fade-in the mode is 1, and mode 1 is excluded from `CheckGliderInRoom`
(`GliderPRO/Sources/Interactions.c:691`–`:694`) and from
`CheckDynamicCollision` (`GliderPRO/Sources/Dynamics.c:45`–`:50`) — but hot spots
are **not** mode-gated in `CheckForHotSpots`, so a glider fading in on top of a
shredder or a burner is affected immediately. (`kShredIt` and `kTransportIt` do
gate on mode; `kBurnIt` and `kDissolveIt` do not.)

### 15.2 `OffAMortal` (`GliderPRO/Sources/Player.c:1484`–`:1604`)

```
  1  OffAMortal(thisGlider):
  2      if (gameOver): return                                    // :1486-1487
  3      if (numShredded > 0): RemoveShreds()                     // :1489-1490
  4      mortals--                                                // :1492
  5
  6      if (mortals < 0):                                        // :1493
  7          HideGlider(thisGlider)                               // :1495
  8          if (twoPlayerGame):
  9              if (mortals < -1):            // both players dead
 10                  FlagGameOver()                               // :1500
 11                  dontDraw = true
 12              else:
 13                  FlagGliderInLimbo(thisGlider, false)         // :1505
 14                  dontDraw = true
 15                  onePlayerLeft = true                         // :1507
 16                  playerDead = thisGlider->which               // :1508
 17          else:
 18              FlagGameOver()                                   // :1513
 19              dontDraw = true
 20      else:
 21          QuickGlidersRefresh()                                // :1519
 22          HideGlider(thisGlider)                               // :1520
 23
 24      if (mortals >= 0):                                       // :1523
 25          if (mode == kGliderGoingFoil): DeckGliderInFoil(thisGlider)   // :1525-1526
 26          FlagGliderNormal(thisGlider)                         // :1528
 27          if (playerSuicide):
 28              FollowTheLeader()                                // :1530
 29          else:
 30              StartGliderFadingIn(thisGlider)                  // :1533
 31              dest = enteredRect                               // :1534   <-- RESPAWN
 32              whole = dest                                     // :1535
 33              destShadow.left  = dest.left                     // :1536
 34              destShadow.right = dest.right                    // :1537
 35              wholeShadow = destShadow                         // :1538
 36      else if (mortals == -1 && onePlayerLeft && !gameOver):    // :1541
 37          switch (otherPlayerEscaped):
 38            kPlayerEscapedUp / kPlayerEscapingUpStairs / kPlayerEscapedUpStairs:
 39                MoveRoomToRoom(survivor, kAbove)               // :1545-1552
 40            kPlayerEscapedDown / kPlayerEscapingDownStairs / kPlayerEscapedDownStairs:
 41                MoveRoomToRoom(survivor, kBelow)               // :1554-1561
 42            kPlayerEscapedLeft:   MoveRoomToRoom(survivor, kToLeft)     // :1563-1568
 43            kPlayerEscapedRight:  MoveRoomToRoom(survivor, kToRight)    // :1570-1575
 44            kPlayerTransportedOut: TransportRoomToRoom(survivor)        // :1577-1582
 45            kPlayerMailedOut:      MoveMailToMail(survivor)             // :1584-1589
 46            kPlayerDuckedOut:      MoveDuctToDuct(survivor)             // :1591-1596
 47            default: nothing
 48          otherPlayerEscaped = kPlayerIsDeadForever             // :1602  = -69
 49      // note: "survivor" is theGlider2 if playerDead == kPlayer1, else theGlider
```

Line 25 is subtle: if you die *while gaining foil*, the foil is applied anyway so
the inventory and the sprite atlas stay consistent.

Line 31 is the respawn: **`dest = enteredRect`**. `enteredRect` is written at
exactly six places, all of them "I have just become normal in this room":

| Where | When |
| --- | --- |
| `GliderPRO/Sources/Player.c:240` | fade-in completed |
| `GliderPRO/Sources/Player.c:270` | transporter arrival completed |
| `GliderPRO/Sources/Player.c:425` | up-stairs arrival completed |
| `GliderPRO/Sources/Player.c:554` | down-stairs arrival completed |
| `GliderPRO/Sources/Player.c:883` / `:921` | mail arrival completed (left / right) |
| `GliderPRO/Sources/Player.c:953` | duct arrival completed |
| `GliderPRO/Sources/Transit.c:88` | `ReadyGliderFromTransit` (transporter arrival) |
| `GliderPRO/Sources/Transit.c:177`, `:178`, `:186` | entering a room from the left (`kToRight`) |
| `GliderPRO/Sources/Transit.c:209`, `:210`, `:218` | entering a room from the right (`kToLeft`) |
| `GliderPRO/Sources/Transit.c:233`, `:234`, `:239` | entering a room from below (`kAbove`) |
| `GliderPRO/Sources/Transit.c:265`, `:266`, `:271` | entering a room from above (`kBelow`) |

So **you respawn where you entered the room**, not where you died and not at the
house's start point. Note `enteredRect` is *not* re-anchored — `FlagGliderNormal`
runs first (line 26), setting `dest` to 48 x 20 from wherever `dest` currently
is, and *then* `dest` is overwritten wholesale by `enteredRect` (which is always
a 48 x 20 rect because it was captured from a normal-sized `dest`).

`mortals` starts at `kInitialGliders` = 2, or 4 in a two-player game
(`GliderPRO/Sources/Play.c:19`, `:341`–`:343`). Since `mortals--` happens
*before* the `< 0` test, you get **3 deaths** in one-player (`mortals` 2 → 1 → 0 →
−1 game over) and **6 shared deaths** in two-player: deaths 1–4 leave `mortals` at
3/2/1/0 and respawn normally, death 5 takes it to −1 (that glider goes to limbo
forever, `onePlayerLeft = true`), and death 6 takes it to −2, which is the
`mortals < -1` "both players are now dead" branch at `GliderPRO/Sources/Player.c:1498`.
`kPaper` pickups increment it — by 1 normally, and by **2** in a two-player game
while both players are still alive (`GliderPRO/Sources/Interactions.c:841`–`:843`).

### 15.3 `HideGlider` (`GliderPRO/Sources/Play.c:712`–`:729`)

```
1  HideGlider(thisGlider):
2      tempRect = thisGlider->whole
3      QOffsetRect(&tempRect, playOriginH, playOriginV)
4      CopyRectWorkToMain(&tempRect)
5      if (hasMirror):
6          tempRect = thisGlider->whole
7          QOffsetRect(&tempRect, playOriginH - 20, playOriginV - 16)
8          CopyRectWorkToMain(&tempRect)
9      tempRect = thisGlider->wholeShadow
10     QOffsetRect(&tempRect, playOriginH, playOriginV)
11     CopyRectWorkToMain(&tempRect)
```

Pure dirty-rect restoration: blit the clean background over the glider's swept
rect, its mirror reflection, and its shadow. In a Go port with a full-frame
redraw this function disappears entirely.

### 15.4 `ForceKillGlider` (`GliderPRO/Sources/Transit.c:449`–`:469`)

```
1  ForceKillGlider():
2      if (theGlider.mode == kGliderInLimbo):
3          StartGliderFadingOut(&theGlider2);  kFadeOutSound
4      else:
5          StartGliderFadingOut(&theGlider);   kFadeOutSound
6      playerSuicide = true
```

The glider that is **not** in limbo is the one killed, and `playerSuicide` makes
`OffAMortal` call `FollowTheLeader()` instead of the normal respawn — i.e. the
suicide teleports you to your partner rather than back to `enteredRect`.

### 15.5 `FollowTheLeader` (`GliderPRO/Sources/Transit.c:473`–`:557`)

```
 1  FollowTheLeader():
 2      playerSuicide = false
 3      wasEscaped = otherPlayerEscaped
 4      otherPlayerEscaped = kNoOneEscaped
 5      // copy the limbo glider's position onto the other glider
 6      (other).dest        = (limbo).dest
 7      (other).whole       = (other).dest
 8      (other).destShadow  = (limbo).destShadow
 9      (other).wholeShadow = (other).destShadow
10      switch (wasEscaped):
11        kPlayerEscapedUp   / kPlayerEscaped*UpStairs:   MoveRoomToRoom(kAbove)
12        kPlayerEscapedDown / kPlayerEscaped*DownStairs: MoveRoomToRoom(kBelow)
13        kPlayerEscapedLeft:      MoveRoomToRoom(kToLeft)
14        kPlayerEscapedRight:     MoveRoomToRoom(kToRight)
15        kPlayerTransportedOut:   TransportRoomToRoom(...)
16        kPlayerMailedOut:        MoveMailToMail(...)
17        kPlayerDuckedOut:        MoveDuctToDuct(...)
```

### 15.6 The escape codes

`GliderPRO/Headers/GliderDefines.h:596`–`:608`. All negative so they can share a
single `short` with `kNoOneEscaped`:

| Symbol | Value |
| --- | --- |
| `kPlayerIsDeadForever` | −69 |
| `kPlayerMailedOut` | −12 |
| `kPlayerDuckedOut` | −11 |
| `kPlayerTransportedOut` | −10 |
| `kPlayerEscapingDownStairs` | −9 |
| `kPlayerEscapingUpStairs` | −8 |
| `kPlayerEscapedDownStairs` | −7 |
| `kPlayerEscapedUpStairs` | −6 |
| `kPlayerEscapedDown` | −5 |
| `kPlayerEscapedUp` | −4 |
| `kPlayerEscapedLeft` | −3 |
| `kPlayerEscapedRight` | −2 |
| `kNoOneEscaped` | −1 |

`otherPlayerEscaped` is a single global (`GliderPRO/Sources/Interactions.c:42`),
so **only one player can be waiting at a time**.
---

## 16. Room transitions and spawn positions

### 16.1 `OffsetGlider` (`GliderPRO/Sources/Player.c:1443`–`:1480`)

The player does not have a world position — the *room* changes and the glider is
translated by exactly one room's worth of pixels.

```
 1  OffsetGlider(thisGlider, where):
 2      if (twoPlayerGame && onePlayerLeft && thisGlider->which == playerDead):
 3          return                                        // dead player is not moved
 4      switch (where):
 5        case kToRight:                                     // :1450
 6            dest.left       += kRoomWide                   // :1451  +512
 7            dest.right      += kRoomWide                   // :1452
 8            destShadow.left += kRoomWide                   // :1453
 9            destShadow.right+= kRoomWide                   // :1454
10            whole       = dest                             // :1455
11            wholeShadow = destShadow                       // :1456
12        case kToLeft:                                      // :1459
13            dest.left       -= kRoomWide                   // :1460  -512
14            dest.right      -= kRoomWide                   // :1461
15            destShadow.left -= kRoomWide                   // :1462
16            destShadow.right-= kRoomWide                   // :1463
17            whole       = dest;  wholeShadow = destShadow  // :1464-1465
18        case kAbove:                                       // :1468
19            dest.top    -= kTileHigh                       // :1469  -322
20            dest.bottom -= kTileHigh                       // :1470
21            whole = dest                                   // :1471
22            // destShadow / wholeShadow are NOT touched
23        case kBelow:                                       // :1474
24            dest.top    += kTileHigh                       // :1475  +322
25            dest.bottom += kTileHigh                       // :1476
26            whole = dest                                   // :1477
```

Note that the function writes the four rect edges directly rather than calling
`QOffsetRect`, and note the asymmetry at lines 22/26: vertical room changes do
**not** move `destShadow`, because the shadow's y is a fixed room-local constant (306) and
must stay there. Horizontal changes do move it, because its x tracks the glider.

Also note the direction inversion at the call sites: moving to the room on the
**east** calls `OffsetGlider(kToLeft)` (`GliderPRO/Sources/Transit.c:172`, `:182`)
— the glider slides 512 px left in the *new* room's coordinates, which places it
at the new room's left edge. Similarly `kAbove` → `OffsetGlider(kBelow)`
(`GliderPRO/Sources/Transit.c:231`, `:238`).

### 16.2 `MoveRoomToRoom` (`GliderPRO/Sources/Transit.c:151`–`:310`)

```
 1  MoveRoomToRoom(thisGlider, where):
 2      HandleRoomVisitation()                                  // :155  award 100 pts once
 3      switch (where):
 4        case kToRight:                                        // :158
 5            SetMusicalMode(kProdGameScoreMode)
 6            if (twoPlayerGame):
 7                UndoGliderLimbo(&theGlider); UndoGliderLimbo(&theGlider2)
 8                InsureGliderFacingRight(&theGlider); InsureGliderFacingRight(&theGlider2)
 9            else:
10                InsureGliderFacingRight(thisGlider)           // :168
11            ForceThisRoom(localNumbers[kEastRoom])            // :169
12            (for each moving glider) OffsetGlider(g, kToLeft) // :172-173 / :182
13            QSetRect(&enterRect, 0, 0, 48, 20)                // :174 / :183
14            QOffsetRect(&enterRect, 0,
15                        kGliderStartsDown + (short)thisRoom->leftStart - 2)   // :175-176
16            (each glider).enteredRect = enterRect             // :177-178 / :186
17        case kToLeft:                                         // :190
18            ... InsureGliderFacingLeft ... ForceThisRoom(localNumbers[kWestRoom])
19            OffsetGlider(g, kToRight)                         // :204-205 / :214
20            QSetRect(&enterRect, 0, 0, 48, 20)
21            QOffsetRect(&enterRect, kRoomWide - 48,           // 464
22                        kGliderStartsDown + (short)thisRoom->rightStart - 2)  // :207-208
23            (each glider).enteredRect = enterRect             // :209-210 / :218
24        case kAbove:                                          // :222
25            SetMusicalMode(kKickGameScoreMode)
26            ForceThisRoom(localNumbers[kNorthRoom])           // :224
27            if (!takingTheStairs):                            // :225
28                UndoGliderLimbo both (two-player)
29                OffsetGlider(g, kBelow)                       // :231-232 / :238
30                (each glider).enteredRect = (that glider).dest  // :233-234 / :239
31            else:
32                ReadyGliderForTripUpStairs(g)                 // :246-247 / :250
33        case kBelow:                                          // :254
34            ForceThisRoom(localNumbers[kSouthRoom])
35            if (!takingTheStairs):
36                OffsetGlider(g, kAbove)                       // :263-264 / :270
37                (each glider).enteredRect = (that glider).dest  // :265-266 / :271
38            else:
39                ReadyGliderForTripDownStairs(g)               // :278-279 / :282
40
41      if (twoPlayerGame && !onePlayerLeft):                    // :290
42          TagGliderIdle(the glider that is NOT firstPlayer)     // :292-295
43      ReadyLevel()                                            // :298  <- clears takingTheStairs
44      RefreshScoreboard(kNormalTitleMode)                      // :299
45      WipeScreenOn(where, &justRoomsRect)                      // :300
46      RenderFrame()                                            // :303
47      (restart any QuickTime movie in the new room)
```

Four things a port must get right:

1. **A horizontal room change forces the facing.** Going right forces
   `kFaceRight` and going left forces `kFaceLeft`, via `InsureGliderFacing*`,
   which is a *no-op if the glider is burning* (`GliderPRO/Sources/Modes.c:502`,
   `:513`; `:499`/`:510` are the separate dead-player early-outs) and otherwise triggers a full about-face tumble (mode 7/8). So the
   glider arrives in the next room mid-tumble.

2. **`enteredRect` for a horizontal entry is not where the glider actually is.**
   The glider is placed by `OffsetGlider` (translated by 512), but `enteredRect`
   is set to a *canonical* 48 x 20 rect flush against the entry wall at
   `y = kGliderStartsDown + leftStart − 2` (or `rightStart`). So if you die in the
   new room you respawn at that canonical spot, not at your actual entry point.
   `kGliderStartsDown` = 32, so with `leftStart == 0` the respawn `dest.top` is
   **30**, and with `leftStart == 32` it is **62**.

3. **A vertical room change sets `enteredRect = dest`** (the true translated
   position), not a canonical rect.

4. **`takingTheStairs` lifecycle.** It is set to `true` only at
   `GliderPRO/Sources/Player.c:344` (up) and `:472` (down), read at
   `GliderPRO/Sources/Transit.c:225` and `:257`, and cleared at
   `GliderPRO/Sources/RoomGraphics.c:127` inside `DrawLocale`, which is called
   from `ReadyLevel` (`GliderPRO/Sources/RoomGraphics.c:402`–`:418`) at
   `GliderPRO/Sources/Transit.c:298` — i.e. **after** the
   `ReadyGliderForTrip*Stairs` call. A Go port that clears the flag earlier will
   send stair-walkers to the wrong arrival position.

`ReadyLevel` order (`GliderPRO/Sources/RoomGraphics.c:402`–`:418`):
`NilSavedMaps()` → (QuickTime teardown) → `DetermineRoomOpenings()` →
`DrawLocale()` → `InitGarbageRects()`. `DrawLocale` ends with
`shadowVisible = IsShadowVisible();` (`:126`) and `takingTheStairs = false;`
(`:127`).

### 16.3 `TransportRoomToRoom`, `MoveDuctToDuct`, `MoveMailToMail`

`GliderPRO/Sources/Transit.c:314`–`:348`, `:352`–`:387`, `:391`–`:426`. All three
are structurally identical:

```
 1  SetMusicalMode(kKickGameScoreMode)                            // :318
 2  HandleRoomVisitation()                                        // :319
 3  sameRoom = (transRoom == thisRoomNumber)                      // :321
 4  if (!sameRoom): ForceThisRoom(transRoom)                      // :322-323
 5  if (twoPlayerGame):                                           // :324
 6      UndoGliderLimbo(&theGlider)                               // :326
 7      UndoGliderLimbo(&theGlider2)                              // :327
 8      ReadyGliderFromTransit(&theGlider,  linkedToWhat)         // :328
 9      ReadyGliderFromTransit(&theGlider2, linkedToWhat)         // :329
10  else:
11      ReadyGliderFromTransit(thisGlider, linkedToWhat)          // :332
12  if (!sameRoom): ReadyLevel()                                  // :334-335
13  RefreshScoreboard(kNormalTitleMode)                           // :336  unconditional
14  if (!sameRoom): WipeScreenOn(kAbove, &justRoomsRect)          // :337-338
15  RenderFrame()                                                 // :341, in #ifdef COMPILEQT
16  if (thisMac.hasQT && hasMovie && tvInRoom && tvOn):           // :342
17      GoToBeginningOfMovie(theMovie);  StartMovie(theMovie)     // :344-345
```

The `twoPlayerGame` test at line 5 is not decoration: in a one-player game only
`thisGlider` is readied and neither glider is taken out of limbo. Note also that the
three `!sameRoom` tests are separate, with the unconditional
`RefreshScoreboard(kNormalTitleMode)` sitting between `ReadyLevel` and
`WipeScreenOn` — the scoreboard is refreshed even for a transit that lands back in
the room it started from.

`transRoom`, `linkedToWhat` and `transRect` are globals filled at mode entry by
`StartGliderMailingIn`, `StartGliderDuctingDown`, `StartGliderDuctingUp` **and
`StartGliderTransporting`** from the linked object's record — all four run the
identical resolution block, and the transporter is the one a porter forgets,
because the fade counter is the visible part of that routine (§8.10).
`WhatAreWeLinkedTo`
(`GliderPRO/Sources/Transit.c:31`–`:61`):

| Destination object class | `linkedToWhat` | Value |
| --- | --- | --- |
| `kMailboxLf` (0x33) | `kLinkedToLeftMailbox` | 1 |
| `kMailboxRt` (0x34) | `kLinkedToRightMailbox` | 2 |
| `kCeilingTrans` (0x36) | `kLinkedToCeilingDuct` | 3 |
| anything else | `kLinkedToOther` | 0 |

(`kLinkedToOther` 0, `kLinkedToLeftMailbox` 1, `kLinkedToRightMailbox` 2,
`kLinkedToCeilingDuct` 3, `kLinkedToFloorDuct` 4 —
`GliderPRO/Headers/GliderDefines.h:610`–`:614`.)

`ReadyGliderFromTransit` (`GliderPRO/Sources/Transit.c:65`–`:147`):

```
 1  ReadyGliderFromTransit(thisGlider, toWhat):                       // :65
 2      if (twoPlayerGame && onePlayerLeft && which == playerDead): return  // :70-71
 3      FlagGliderNormal(thisGlider)                                  // :73  baseline
 4      switch (toWhat):
 5        case kLinkedToOther:                          // :76  a transporter pad
 6            StartGliderTransportingIn(thisGlider)                   // :77
 7            tempRect = dest                                         // :78
 8            CenterRectInRect(&tempRect, &transRect)                 // :79
 9            dest.left/right/top/bottom = tempRect.*                 // :80-83
10            destShadow.left = tempRect.left                         // :84
11            destShadow.right = tempRect.right                       // :85
12            whole = dest;  wholeShadow = destShadow                 // :86-87
13            enteredRect = dest                                      // :88
14        case kLinkedToLeftMailbox:                     // :91  slot opens leftward
15            StartGliderMailingOut(thisGlider)                       // :92
16            clip = transRect                                        // :93
17            clip.right  -= 64                                       // :94
18            clip.bottom -= 25                                       // :95
19            tempRect = dest                                         // :96
20            dest.left   = clip.right                                 // :97
21            dest.right  = dest.left            // :98  zero width, will grow
22            dest.bottom = clip.bottom - 4                            // :99
23            dest.top    = dest.bottom - RectTall(&tempRect)          // :100
24            destShadow.left  = dest.left                             // :101
25            destShadow.right = dest.right                            // :102
26            whole = dest;  wholeShadow = destShadow                  // :103-104
27        case kLinkedToRightMailbox:                                  // :107
28            StartGliderMailingOut(thisGlider)                        // :108
29            clip = transRect                                         // :109
30            clip.left   += 79                                        // :110
31            clip.bottom -= 25                                        // :111
32            tempRect = dest                                          // :112
33            dest.right  = clip.left                                  // :113
34            dest.left   = dest.right                    // :114  zero width
35            dest.bottom = clip.bottom - 4                            // :115
36            dest.top    = dest.bottom - RectTall(&tempRect)          // :116
37            destShadow.left/right = dest.left/right                  // :117-118
38            whole = dest;  wholeShadow = destShadow                  // :119-120
39        case kLinkedToCeilingDuct:                                   // :123
40            StartGliderDuctingIn(thisGlider)                         // :124
41            tempRect = dest                                          // :125
42            CenterRectInRect(&tempRect, &transRect)                  // :126
43            dest.left/right/top = tempRect.left/right/top            // :127-129
44            dest.bottom = dest.top                                   // :130
45            QOffsetRect(&dest, 0, -RectTall(&tempRect))  // :131  start fully above
46            destShadow.left/right = tempRect.left/right              // :132-133
47            whole = dest;  wholeShadow = destShadow                  // :134-135
48        case kLinkedToFloorDuct:                                     // :138
49            (nothing)
50      if (twoPlayerGame && which != firstPlayer):                    // :145
51          TagGliderIdle(thisGlider)                                  // :146
```

The mail/duct arrivals deliberately start with a **degenerate (zero-width or
zero-height) `dest`** so the `Finish…` handler can grow it from nothing,
producing the "pushed out of the slot" effect. That is also why those handlers
test `dest.left == dest.right` / `dest.top == dest.bottom` to fire the arrival
sound (`kTransInSound`) exactly once
(`GliderPRO/Sources/Player.c:856`, `:894`, `:932`).

`CenterRectInRect` (`GliderPRO/Sources/RectUtils.c:142`–`:154`):

```c
rectA->left   = rectB->left + ((RectWide(rectB) - RectWide(rectA)) / 2);
rectA->right  = rectA->left + RectWide(rectA);      // computed before left changed
rectA->top    = rectB->top  + ((RectTall(rectB) - RectTall(rectA)) / 2);
rectA->bottom = rectA->top  + RectTall(rectA);
```

(the original caches the widths in locals first, so the arithmetic is correct).

### 16.4 The house's initial spawn point

`WhereDoesGliderBegin` (`GliderPRO/Sources/House.c:224`–`:240`):

```
1  WhereDoesGliderBegin(theRect, mode):
2      if (mode == kResumeGameMode): initialPt = smallGame.where
3      else:                         initialPt = (*thisHouse)->initial
4      QSetRect(theRect, 0, 0, kGliderWide, kGliderHigh)     // 48 x 20
5      QOffsetRect(theRect, initialPt.h, initialPt.v)
```

So the very first spawn `dest` is `(left=initial.h, top=initial.v,
right=initial.h+48, bottom=initial.v+20)`.

`InitGlider` (`GliderPRO/Sources/Play.c:307`–`:366`) then:

| Field | New game | Resumed game |
| --- | --- | --- |
| `dest` | `WhereDoesGliderBegin(..., kNewGameMode)` | `WhereDoesGliderBegin(..., kResumeGameMode)` |
| `theScore` | 0 | `smallGame.score` |
| `mortals` | `kInitialGliders` (2), +2 if `twoPlayerGame` | `smallGame.numGliders` |
| `batteryTotal` | 0 | `smallGame.energy` |
| `bandsTotal` | 0 | `smallGame.bands` |
| `foilTotal` | 0 | `smallGame.foil` |
| `mode` | `kGliderNormal` | `smallGame.gliderState`, then `FlagGliderBurning` if that was `kGliderBurning` else `FlagGliderNormal` |
| `facing` | `kFaceRight` | `smallGame.facing` |
| `showFoil` | false | `smallGame.showFoil` |
| `src`/`mask` | `gliderSrc[0]` | (from the Flag call) |
| `destShadow` | `QSetRect(0,0,48,9)` then `QOffsetRect(dest.left, kShadowTop /*306*/)`; `wholeShadow = destShadow` | same |
| `hVel`,`vVel`,`hDesiredVel`,`vDesiredVel` | 0 | 0 |
| `tipped`, `sliding`, `dontDraw` | false | false |

`InitGlider` line numbers: `WhereDoesGliderBegin` `GliderPRO/Sources/Play.c:309`,
`numStarsRemaining` `:311`–`:314`, the `kResumeGameMode` block `:316`–`:337`,
the `kNewGameMode` block `:338`–`:352`, `mortals = kInitialGliders` `:341` with
`mortals += kInitialGliders` for two players `:343`, the shadow setup
`:354`–`:356`, the four velocity zeroes `:358`–`:361`, and
`tipped`/`sliding`/`dontDraw` `:363`–`:365`.
`kInitialGliders` is **2**, defined locally at `GliderPRO/Sources/Play.c:19`
(not in `GliderDefines.h`).

Two things `InitGlider` does **not** do:

* It never sets `whole`. The first `MoveGlider` (or the first `Start…` mode
  entry) writes it, so a Go port must not rely on `whole` being valid before the
  first frame of movement.
* It never sets `enteredRect`. `theGlider` / `theGlider2` are file-scope globals
  and therefore zero-filled, so `enteredRect` is `(0,0,0,0)` at this point. What
  saves the game is `NewGame`'s call to `StartGliderFadingIn(&theGlider)` at
  `GliderPRO/Sources/Play.c:185` (and `:188` for player 2): the glider starts in
  `kGliderFadingIn` (mode 1), and when `FadeGliderIn` completes it executes
  `FlagGliderNormal` then `enteredRect = dest`
  (`GliderPRO/Sources/Player.c:240`). **A Go port that skips the 16-frame
  spawn fade must set `enteredRect = dest` explicitly, or the first death will
  respawn the player at (0,0).**

`NewGame`'s setup order around this (`GliderPRO/Sources/Play.c:74`–`:214`):
`NilSavedMaps()` `:110`, `gameFrame = 0` `:112`, `numBands = 0` `:113`,
`demoIndex = 0` `:114`, `saidFollow = 0` `:115`,
`otherPlayerEscaped = kNoOneEscaped` `:116`, `onePlayerLeft = false` `:117`,
`playerSuicide = false` `:118`, the `InitGlider` + atlas loads `:120`–`:136`,
`DrawLocale()` `:163`, `RefreshScoreboard(kNormalTitleMode)` `:164`,
`InitGarbageRects()` `:184`, `StartGliderFadingIn(&theGlider)` `:185`,
(two-player) `StartGliderFadingIn(&theGlider2)` `:188` +
`TagGliderIdle(&theGlider2)` `:189` + `theGlider2.dontDraw = true` `:190`,
`playing = true` `:213`, `PlayGame()` `:214`.

Note `:190`: in a two-player game player 2 starts with `dontDraw = true` and
idle, so player 1 materialises alone and player 2 pops in when the idle counter
expires.

The saved-game record (`gameType`, `GliderPRO/Headers/GliderStructs.h:116`–`:134`)
persists only: `where` (a `Point`), `score`, `energy`, `bands`, `roomNumber`,
`gliderState` (= `theGlider.mode`), `numGliders` (= `mortals`), `foil`, `facing`,
`showFoil`, `wasStarsLeft` (`GliderPRO/Sources/SavedGames.c:101`–`:106`,
`:261`–`:275`, `:330`–`:335`). **`gliderType` itself is never serialised** — no
velocities, no `enteredRect`, no `frame`. A resumed game therefore always starts
from rest.

### 16.5 Two-player limbo state machine

Globals involved:

| Global | Declared | Meaning |
| --- | --- | --- |
| `otherPlayerEscaped` | `GliderPRO/Sources/Interactions.c:42` | which exit the waiting player took, or `kNoOneEscaped` |
| `onePlayerLeft` | `GliderPRO/Sources/Player.c:53` | one player is permanently dead |
| `playerDead` | `GliderPRO/Sources/Player.c:53` | `which` of the dead player |
| `firstPlayer` | `GliderPRO/Sources/Transit.c:18` | `which` of the player who reached the exit first |
| `playerSuicide` | (extern at `GliderPRO/Sources/Player.c:58`) | the Delete-key kill happened |
| `takingTheStairs` | `GliderPRO/Sources/Transit.c:18` | this transition is a staircase walk |
| `saidFollow` | `GliderPRO/Sources/Modes.c:13` | how many times the "follow me" sound has played (max 3) |

Sequence for a normal two-player room change:

1. Player A reaches an exit. For the four walls that is `CheckEscape*Two`
   (`GliderPRO/Sources/Interactions.c:171`, `:283`, `:509`, `:599`); the
   staircases, transporters, ducts and mail slots run the same handshake inline in
   their `Player.c` handlers. Either way it sees
   `otherPlayerEscaped == kNoOneEscaped`, records the exit code, and calls
   `FlagGliderInLimbo(A, true)` — saving A's mode in `wasMode`, playing
   `kFollowSound` up to 3 times, and setting `firstPlayer = A.which`. It does
   **not** set `A.dontDraw`: nothing in `Interactions.c` writes that field at all.
   A is hidden only on the exits whose handler had already hidden it a few lines
   earlier (`Player.c:607`, `:715`, `:812`, `:1011`, `:1102`); on a wall exit A
   stays fully drawn, frozen at the room edge, and on a staircase exit only its
   shadow survives, because `dest` was collapsed rather than the flag set — see
   §8.21. **The room does not change.**
2. Player B reaches the *same* exit. Now `otherPlayerEscaped != kNoOneEscaped`,
   so `MoveRoomToRoom` runs for both.
3. `MoveRoomToRoom` calls `UndoGliderLimbo` on both gliders (restoring A's mode),
   moves both, then `TagGliderIdle` on whichever glider is **not** `firstPlayer`
   — giving player B a 30-frame freeze so player A (who waited) gets a head
   start.
4. If player B reaches a *different* exit instead, `CheckEscape*Two` plays
   `kDontExitSound` and bounces B off an invisible barrier at the `kNo*Limit`
   plane with `vVel = -vVel + offset` (or `hVel`).
5. `otherPlayerEscaped` is reset to `kNoOneEscaped` **by the second player's own
   escape test**, immediately before it calls `MoveRoomToRoom` — e.g.
   `GliderPRO/Sources/Interactions.c:187` for the up exit, `:299` for down,
   `:527`/`:558` for left, `:617`/`:648` for right, and
   `GliderPRO/Sources/Player.c:358` / `:486` for the staircases,
   `:622` for transporters, `:729`/`:826` for ducts, `:1025`/`:1116` for
   mailboxes. It is also reset in `NewGame`
   (`GliderPRO/Sources/Play.c:116`) and in `FollowTheLeader`
   (`GliderPRO/Sources/Transit.c:480`, after saving the old value into
   `wasEscaped` at `:479`), and it is set to `kPlayerIsDeadForever` (−69) in
   `OffAMortal` (`GliderPRO/Sources/Player.c:1602`). So the reset always lives
   next to the `MoveRoomToRoom` call, never inside it — a port must not move it
   into the transition function, because the `kPlayerIsDeadForever` and
   `kPlayerEscaping*Stairs` states deliberately fall through to the *third*
   ("wrong exit") branch instead.

---

## 17. The authoritative per-frame update order

`PlayGame` (`GliderPRO/Sources/Play.c:430`–`:599`). The two-player and
one-player paths are separate but identical in order.

```
  1  while (playing && !quitting):
  2      gameFrame++                                      // :434
  3      evenFrame = !evenFrame                           // :435
  4      if (doBackground):                               // :437
  5          do { HandlePlayEvent(); } while (switchedOut)
  6      HandleTelephone()                                // :445
  7      HandleDynamics()                                 // :449 / :475
  8      if (!gameOver):
  9          GetInput(&theGlider)                         // :452 / :481
 10          GetInput(&theGlider2)                        // :453   (two-player only)
 11          //   (one-player: GetDemoInput(&theGlider) at :479 if demoGoing)
 12          HandleInteraction()                          // :454 / :482
 13      HandleTriggers()                                 // :456 / :484
 14      HandleBands()                                    // :457 / :485
 15      if (!gameOver):
 16          HandleGlider(&theGlider)                     // :460 / :487
 17          HandleGlider(&theGlider2)                    // :461   (two-player only)
 18      if (playing):
 19          MoviesTask(theMovie, 0)                      // :467 / :492  (QuickTime)
 20          RenderFrame()                                // :469 / :494
 21          HandleDynamicScoreboard()                    // :470 / :495
 22      if (gameOver):                                   // :499
 23          countDown--                                  // :501
 24          if (countDown <= 0):                         // :502
 25              HideGlider(...); RefreshScoreboard(...)
 26              mortals < 0 ? DoDiedGameOver() : DoGameOver()
```

The ordering constraints this creates:

* **Input runs before interaction, which runs before movement.** So within one
  frame: keys set `hDesiredVel`/`tipped`; hot spots add to `hDesiredVel`,
  overwrite `vDesiredVel`, and set the one-frame `ignore*` flags; then
  `HandleGlider` → `MoveGliderNormal` → `MoveGlider` consumes and resets them.
* **Wall checks (`CheckGliderInRoom`) run inside `HandleInteraction`, i.e. BEFORE
  the glider moves.** So a wall bounce reacts to *last* frame's position. This is
  why the bounce formulas add the penetration depth: the glider has already
  overshot when it is detected.
* **A room transition happens inside `HandleGlider`** (from
  `MoveGliderUpStairs`, `TransportGliderOut`, etc.) or inside
  `HandleInteraction` (from `CheckEscape*`). Either way `MoveRoomToRoom` runs a
  full `ReadyLevel` + `RenderFrame` *inside* the current frame, and then the
  outer loop's `RenderFrame` runs again.
* **`HandleBands` runs before `HandleGlider`**, so a band hit adds to `hVel`
  before the integrator sees it.
* **`HandleDynamics` runs first**, so dynamic objects have already moved when
  `CheckDynamicCollision` (called from within `HandleDynamics` itself for each
  dinah) tests them.
* Player 1 is always processed before player 2, in input, in interaction (via
  `CheckForHotSpots`' inner loop) and in movement.

Frame pacing is a **busy-wait** at the end of `RenderFrame`
(`GliderPRO/Sources/Render.c:662`–`:665`):

```c
while (TickCount() < nextFrame) { }
nextFrame = TickCount() + kTicksPerFrame;      // kTicksPerFrame = 2
```

A Mac tick is 1/60.15 s, so `kTicksPerFrame` = 2 targets **30.07 fps** and the
whole simulation is frame-locked: all velocities are literally pixels per frame.
`HandlePlayEvent` uses `WaitNextEvent` with `sleep = 2`
(`GliderPRO/Sources/Play.c:387`–`:426`).

---

## 18. Rendering the player

### 18.1 `RenderGlider` (`GliderPRO/Sources/Render.c:452`–`:530`)

```
 1  RenderGlider(thisGlider, oneOrTwo):                              // :452
 2      if (thisGlider->dontDraw): return                            // :457-458
 3      which = (thisGlider->facing == kFaceRight) ? 0 : 1            // :460-463
 4      if (shadowVisible):                                          // :465
 5          dest = thisGlider->destShadow                            // :467
 6          QOffsetRect(&dest, playOriginH, playOriginV)             // :468
 7          if (mode == kGliderComingUp || mode == kGliderGoingDown): // :470-471
 8              src = shadowSrc[which]                               // :473
 9              src.right = src.left + (dest.right - dest.left)      // :474  trim RIGHT edge
10              CopyMask(shadowSrcMap, shadowMaskMap, workSrcMap,
11                       &src, &src, &dest)                          // :476-479
12          else if (mode == kGliderComingDown):                     // :481
13              src = shadowSrc[which]                               // :483
14              src.left = src.right - (dest.right - dest.left)      // :484  trim LEFT edge
15              CopyMask(..., &src, &src, &dest)                     // :486-489
16          else:
17              CopyMask(shadowSrcMap, shadowMaskMap, workSrcMap,
18                       &shadowSrc[which], &shadowSrc[which], &dest) // :492-495
19          src = thisGlider->wholeShadow                             // :496
20          QOffsetRect(&src, playOriginH, playOriginV)               // :497
21          AddRectToWorkRects(&src)                                  // :498
22          AddRectToBackRects(&dest)     // :499  the OFFSET shadow rect
23      dest = thisGlider->dest                                       // :502
24      QOffsetRect(&dest, playOriginH, playOriginV)                  // :503
25      if (oneOrTwo):                                                // :505  player 1
26          if ((!twoPlayerGame) && showFoil):                        // :507
27              CopyMask(glid2SrcMap, glidMaskMap, workSrcMap,
28                       &thisGlider->src, &thisGlider->mask, &dest)  // :508-511
29          else:
30              CopyMask(glidSrcMap,  glidMaskMap, workSrcMap,
31                       &thisGlider->src, &thisGlider->mask, &dest)  // :513-516
32      else:                                                         // player 2
33          CopyMask(glid2SrcMap, glidMaskMap, workSrcMap,
34                   &thisGlider->src, &thisGlider->mask, &dest)      // :520-523
35      src = thisGlider->whole                                       // :526
36      QOffsetRect(&src, playOriginH, playOriginV)                   // :527
37      AddRectToWorkRects(&src)                                      // :528
38      AddRectToBackRects(&dest)         // :529  the OFFSET dest rect
```

Lines 7–15 keep the shadow the same width as the (clipped) glider while it walks
in or out of a staircase, trimming from whichever side the glider is being
revealed from — and note that the width used is `dest.right - dest.left` of the
**shadow** rect (`destShadow`, which the stair handlers clip in lockstep with
`dest`), not of `thisGlider->dest`.

Two other details that are easy to miss:

* `which` is computed from `facing` and indexes `shadowSrc[]`, so there are
  exactly **two** shadow sprites: index 0 = facing right, index 1 = facing left.
* Both `AddRectToBackRects` calls (lines 22 and 38) are passed the
  **screen-offset** rect, whereas both `AddRectToWorkRects` calls are passed
  `whole`/`wholeShadow` also offset. So every dirty rect in both lists is in
  screen coordinates. (`AddRectToBackRects` is what erases the glider from
  `workSrcMap` next frame by copying from `backSrcMap`.)

### 18.2 `DrawReflection` (`GliderPRO/Sources/Render.c:136`–`:189`)

```
 1  DrawReflection(thisGlider, oneOrTwo):                           // :136
 2      if (thisGlider->dontDraw): return                           // :142-143
 3      which = (facing == kFaceRight) ? 0 : 1     // :145-148  computed but UNUSED
 4      dest = thisGlider->dest                                     // :150
 5      QOffsetRect(&dest, playOriginH - 20, playOriginV - 16)       // :151
 6      wasClip = NewRgn();  if (wasClip == nil) return              // :153-155
 7      SetPort((GrafPtr)workSrcMap)                                // :157
 8      GetClip(wasClip);  SetClip(mirrorRgn)                       // :158-159
 9      if (oneOrTwo):                                              // :161  player 1
10          if (showFoil):                                          // :163
11              CopyMask(glid2SrcMap, glidMaskMap, workSrcMap,
12                       &src, &mask, &dest)                        // :164-167
13          else:
14              CopyMask(glidSrcMap, glidMaskMap, workSrcMap, ...)  // :169-172
15      else:                                                       // player 2
16          CopyMask(glid2SrcMap, glidMaskMap, workSrcMap, ...)     // :176-179
17      SetClip(wasClip);  DisposeRgn(wasClip)                      // :182-183
18      src = thisGlider->whole                                     // :185
19      QOffsetRect(&src, playOriginH - 20, playOriginV - 16)       // :186
20      AddRectToWorkRects(&src)                                    // :187
21      AddRectToBackRects(&dest)                                   // :188
```

Line 3 is dead code: `which` is computed exactly as in `RenderGlider` but never
read, because the reflection has no shadow.

The mirror offset is a fixed **(−20, −16)** and is applied in `DrawReflection`,
`HideGlider` (`GliderPRO/Sources/Play.c:719`–`:722`) and
`StartGliderFadingOut` (`GliderPRO/Sources/Modes.c:95`).

Note line 10: the reflection tests `showFoil` **without** the `!twoPlayerGame`
guard that `RenderGlider` uses at `:507`, so in a two-player game with foil the reflection
of player 1 is drawn from `glid2SrcMap` — which at that moment holds
`kGliderFoil2PictID` (player 2's foil colours). This is a cosmetic
inconsistency in the original.

### 18.3 The four-atlas / two-GWorld trick

This is worth stating plainly because it is easy to get wrong:

| Game mode | `glidSrcMap` holds | `glid2SrcMap` holds | Loaded at |
| --- | --- | --- | --- |
| one player, start of game | PICT 3999 (plain, P1) | PICT **3976** (foil, P1) | `GliderPRO/Sources/Play.c:132`–`:135` |
| two players, start of game | PICT 3999 (plain, P1) | PICT 3974 (plain, P2) | `GliderPRO/Sources/Play.c:124`–`:127` |
| two players, foil active | PICT **3976** (foil, P1) | PICT **3963** (foil, P2) | `DeckGliderInFoil`, `GliderPRO/Sources/Player.c:1146`–`:1160` |
| two players, foil lost | PICT 3999 | PICT 3974 | `RemoveFoilFromGlider`, `GliderPRO/Sources/Player.c:1206`–`:1220` |

So in one-player mode "foil" is a *map selection* at draw time; in two-player mode
it is a *resource reload*. A Go port with four separate textures and a per-player
`hasFoil` flag will look correct and will not need the reload, but must remember
that `showFoil` is shared between both players.

### 18.4 `RenderShreds` — the confetti that outlives the glider (`GliderPRO/Sources/Render.c:559`–`:612`)

Not strictly the player, but it is the visual half of the shredder death and it
holds the timing.

```
 1  RenderShreds():
 2      for i in 0 .. numShredded-1:                             // :566
 3          if (shreds[i].frame == 0):            // :568  still growing
 4              shreds[i].bounds.bottom += 1                     // :570
 5              high = bounds.bottom - bounds.top                // :571
 6              if (high >= 35): shreds[i].frame = 1             // :572-573
 7              src = shredSrcRect;  src.top = src.bottom - high // :574-575
 8              blit clipped to `high`                           // :576-581
 9              AddRectToBackRects(&dest)                        // :582
10              dest.top--;  AddRectToWorkRects(&dest)           // :583-584
11              PlayPrioritySound(kShredSound, kShredPriority)   // :585
12          else if (shreds[i].frame < 20):       // :587  falling
13              bounds.top += 4;  bounds.bottom += 4             // :589-590
14              shreds[i].frame++                                // :593
15              if (frame < 20): blit whole sprite               // :594-599
16              else:  AddSparkle(&bounds)                       // :603
17                     PlayPrioritySound(kFadeOutSound, ...)     // :604
```

So each confetti pile grows 1 px/frame until it is **35 px** tall (about 35
frames, 1.16 s), then falls **4 px/frame** for 19 frames, then sparkles and
vanishes. Meanwhile the glider's own `frame` is counting up from
`kShredderCountdown` (−68) toward 0 in `MoveGliderShredding`, so the life is not
actually lost until the confetti is nearly done — the two timings are tuned
against each other but are *not* coupled in code.

`RemoveShreds` is called from `OffAMortal`
(`GliderPRO/Sources/Player.c:1489`–`:1490`) when `numShredded > 0`, which is what
stops the confetti when the respawn finally happens.

### 18.5 `RenderFrame` order (`GliderPRO/Sources/Render.c:639`–`:671`)

```
 1  RenderFrame():
 2      if (hasMirror):  DrawReflection(&theGlider, true)
 3                       DrawReflection(&theGlider2, false)   (two-player)
 4      HandleGrease()
 5      RenderPendulums()
 6      evenFrame ? RenderFlames() : RenderStars()      // alternate every frame
 7      RenderDynamics()
 8      RenderFlyingPoints()
 9      RenderSparkles()
10      RenderGlider(&theGlider, true)
11      RenderGlider(&theGlider2, false)                (two-player)
12      RenderShreds()
13      RenderBands()
14      while (TickCount() < nextFrame) { }              // busy-wait pacing
15      nextFrame = TickCount() + kTicksPerFrame          // 2
16      CopyRectsQD()                                    // blit the dirty rects
17      numWork2Main = 0;  numBack2Work = 0
```

The player is drawn **after** all scenery and dynamics but **before** shreds and
rubber bands. Note line 4: `HandleGrease` is a *simulation* step (it grows the
slide hot rect) that runs inside the render pass, so a grease slick grows once
per rendered frame, and it grows **after** `HandleInteraction` has already tested
against it.

Line 6 halves the effective frame rate of flames and stars to 15 fps each.

---

## 19. Empirical verification appendix

Everything in this section is **observed output** from commands run on this
machine against the read-only original tree, not inference.

### 19.1 `gliderType` size and offsets (gcc)

A probe file replicating the struct verbatim, compiled twice:

```
sizeof(Rect)=8 sizeof(gliderType)=112          # natural (PowerPC-like) alignment
sizeof(Rect)=8 sizeof(gliderType)=110          # with #pragma pack(2)  (68k-like)
```

Offsets (identical under both alignments):

```
src 0   mask 8   dest 16  whole 24  destShadow 32  wholeShadow 40
clip 48  enteredRect 56
leftKey 64  rightKey 68  battKey 72  bandKey 76
hVel 80  vVel 82  wasHVel 84  wasVVel 86  vDesiredVel 88  hDesiredVel 90
mode 92  frame 94  wasMode 96
facing 98  tipped 99  sliding 100  ignoreLeft 101  ignoreRight 102
fireHeld 103  which 104  heldLeft 105  heldRight 106  dontDraw 107
ignoreGround 108
```

The struct is never written to disk, so the 110-vs-112 difference is irrelevant
to compatibility; the offsets are given so a port can be checked field-for-field
against a 68k memory dump if one is ever available.

> [verified] These numbers require a **4-byte `long`**, as on the classic Mac.
> Re-running the same probe on a modern LP64 host (where `long` is 8 bytes) yields
> `sizeof = 128 / 126` and shifts everything from `leftKey` onward: `leftKey 64,
> rightKey 72, battKey 80, bandKey 88, hVel 96 … ignoreGround 124`. Use `int32`
> for the four key fields in the port.

### 19.2 On-disk record sizes (python3 `struct`)

```
gameType   = 40 bytes   (where @ 8, gliderState @ 30, numGliders @ 32,
                         foil @ 34, facing @ 38, showFoil @ 39)
roomType   = 348 bytes  (name @ 0, bounds @ 28, leftStart @ 30, rightStart @ 31,
                         unusedByte @ 32, visited @ 33, background @ 34,
                         tiles @ 36, floor @ 52, suite @ 54, openings @ 56,
                         numObjects @ 58, objects @ 60)
houseType header = 866  (version @ 0, unusedShort @ 2, timeStamp @ 4, flags @ 8,
                         initial @ 12, banner @ 16, trailer @ 272,
                         highScores @ 528, savedGame @ 820, hasGame @ 860,
                         firstRoom @ 862, nRooms @ 864)
```

matching the `// total =` comments in `GliderPRO/Headers/GliderStructs.h:134`,
`:180`, `:198`. All multi-byte fields are **big-endian**.

### 19.3 A real house file: `Empty House.binhex`

The task briefing said the `.binhex` files hold "a resource fork with the level
data". **That is not what the bytes say.** Decoding BinHex 4.0 and parsing the
resource map shows the resource fork contains only Finder icon resources
(`ICN#`, `icl8`, `icl4`, `ics#`, `ics8`, `ics4`, all id −16455). The **data fork**
is the `houseType` record. Observed:

```
data fork = 13046 bytes; (13046 - 866) / 348 = 35.0
raw bytes 0..16 : 02 00 00 00 2c 31 30 ec 00 00 00 00 00 40 00 53
version    = 0x0200
timeStamp  = 741421292
flags      = 0x00000000
initial    = Point{v=64, h=83}   (raw 00 40 00 53)
  => glider start dest = (l=83, t=64, r=131, b=84)
hasGame=0 firstRoom=0 nRooms=35
room  0 'Untitled Room' bounds=0 leftStart=  0 rightStart=  0 vis=1 back=2000
        tiles=[0, 1, 1, 1, 1, 1, 1, 1] floor=1 suite=64 openings=0x0000 numObjects=4
        entry-from-left  dest.top = kGliderStartsDown(32) + 0 - 2 = 30
        entry-from-right dest.top = kGliderStartsDown(32) + 0 - 2 = 30
        raw room bytes 28..60:
          00 00 00 00 00 01 07 d0 00 00 00 01 00 01 00 01 00 01 00 01
          00 01 00 01 00 01 00 40 00 00 00 04
```

`(13046 − 866) / 348 = 35.0` exactly, confirming the 866-byte header and
348-byte room stride, and `nRooms = 35` agrees. `initial = {v=64, h=83}` gives the
first-ever glider `dest` of `(83, 64, 131, 84)` via `WhereDoesGliderBegin`.
`kHouseVersion` is `0x0200` (`GliderPRO/Headers/GliderDefines.h:517`) and
`kNewHouseVersion` is `0x0300` (`:518`); this file is 0x0200.

### 19.4 Survey of the shipped houses

All 22 `.binhex` houses in `GliderPRO/Houses/`, BinHex-decoded and parsed with
python3. Every one is version 0x0200. Observed:

| House | `initial` (h, v) | `firstRoom` | `nRooms` | distinct `leftStart` | distinct `rightStart` |
| --- | --- | --- | --- | --- | --- |
| Art Museum | 231, 56 | 91 | 109 | {32} | {0, 32, 101} |
| CD Demo House | 223, 105 | 70 | 206 | {0, 22, 32, 33, 69, 113, 129, 148, 170, 173} | {0, 20, 32, 33, 112, 113, 169} |
| California or Bust! | 308, 62 | 14 | 16 | {32} | {32} |
| Castle o' the Air | 239, 124 | 33 | 85 | {2, 6, 32, 48, 119, 122, 154} | {6, 32, 42, 83, 121, 122} |
| Davis Station | 189, 95 | 4 | 65 | {0, 3, 7, 22, 32, 53, 77, 94, 111, 125} | {0, 2, 19, 32, 43, 95, 122, 142} |
| Demo House | 49, 107 | 0 | 45 | {0} | {0} |
| Empty House | 83, 64 | 0 | 35 | {0} | {0} |
| Fun House | 42, 78 | 29 | 43 | {14, 32, 66, 106} | {0, 15, 26, 32, 141} |
| Grand Prix | 268, 58 | 127 | 175 | {0, 3, 32, 90} | {28, 30, 32, 120, 131, 174, 190} |
| ImagineHouse PRO II | 211, 35 | 1 | 279 | 12 distinct, incl. {0, 32, 33} | 13 distinct, incl. {0, 32, 33} |
| In The Mirror | 424, 50 | 6 | 97 | {2, 32, 46, 56} | {0, 3, 21, 32, 33, 120} |
| Land of Illusion | 245, 163 | 43 | 303 | 25 distinct, incl. {0, 31, 32} | 19 distinct, incl. {0, 31, 32} |
| Leviathan | 229, 44 | 39 | 472 | 14 distinct, incl. {0, 32, 33} | 19 distinct, incl. {0, 32, 33} |
| Metropolis | 245, 24 | 8 | 127 | {0, 32, 69, 93, 106, 115, 148, 173, 249} | {0, 32, 55, 92, 115} |
| Nemo's Market | 342, 51 | 0 | 124 | {3, 32, 64, 66, 82, 109, 120} | {0, 31, 32, 107, 124, 168} |
| Rainbow's End | 237, 200 | 30 | 223 | 19 distinct, incl. {0, 32} | 19 distinct, incl. {0, 32} |
| Sampler | 384, 85 | 1 | 2 | {32} | {27, 32} |
| Slumberland | 361, 7 | 126 | 383 | 14 distinct, incl. {0, 32} | 19 distinct, incl. {0, 32} |
| SpacePods | 30, 72 | 259 | 402 | 24 distinct, incl. {0, 32, 33} | 26 distinct, incl. {0, 32} |
| Teddy World | 362, 41 | 0 | 531 | {0, 31, 32, 33, 51, 52, 56} | {0, 29, 31, 32, 55, 57, 74, 81} |
| The Asylum Pro | 78, 27 | 20 | 140 | {0, 32, 127, 128} | {0, 18, 127} |
| Titanic | 39, 97 | 92 | 208 | {32, 40, 77, 86, 87, 170, 184, 213, 215, 216, 217} | {16, 32, 78, 87, 213, 216, 217} |

`(dataForkLen - 866) / 348` is an exact integer equal to `nRooms` for all 22
houses **except `Sampler`**, so the 866-byte header and 348-byte room stride are
confirmed against every shipped file.

The practical result for the player spec: across all 8140 `leftStart`/`rightStart`
values in the 22 houses, **32 accounts for 78.9% and 0 for 15.7% (94.6% combined)**,
so the canonical horizontal-entry respawn `dest.top` is almost always **30** or
**62** — but a port must still read the byte, because the remaining 5.4% spans
values up to 255.

One anomaly: `Sampler` does not have an exact room count.

```
Sampler: dataFork 1564   (1564-866) = 698   /348 = 2.0057471264367814
```

866 + 2 x 348 = 1562, so there are **2 trailing bytes** past the last complete
room. See §22.

### 19.5 Glider PICT frames

See §6.1 for the observed table. Additional observation: a resource-type census of
`GliderPRO/Glider PRO.r` gives PICT 152, snd 70, DITL 54, cicn 44, ICON 35,
DLOG 28, ALRT 26, dctb 20, CURS 16, crsr 12, STR# 10, FREF 6, ICN# 6, icl4 6,
icl8 6, MENU 6, CNTL 5, mctb 5, ics# 4, ics4 4, ics8 4, WIND 3, clut 2, vers 2,
WDEF 2, and one each of BNDL, CDEF, DLGX, PAT#, acur, cctb, demo, ictb, ozm5,
wctb. The single `demo` resource is the recorded demo key stream consumed by
`GetDemoInput`; the single `ozm5` is a house embedded in the application.

**Caveat for tooling:** in a UTF-8 locale, `grep` silently produces **no output
at all** (not even a `-c` count of 0, and exit status 1) for any of these files
that contain Mac-Roman high bytes, because the invalid multibyte sequences make it
treat the file as binary. Measured: `Headers/GliderDefines.h` has 550 high bytes
and `Sources/Interactions.c` has 9, and `grep -c kTicksPerFrame
Headers/GliderDefines.h` prints nothing even though the token really is there (at
`:533`). Files that happen to be pure ASCII (`Player.c`, `Modes.c`, `Input.c`)
grep correctly, which makes the failure mode especially treacherous — it looks
like a confirmed absence. `Glider PRO.r` fails the same way and needs `-a`. All
searching for this document was therefore done with a python3 helper that reads
bytes and decodes `mac-roman` (sources) or `latin-1` (the `.r` dump) per line.

---

## 20. Mac Toolbox dependencies and their Go replacements

| Original mechanism | Where it appears in the player code | What a Go port needs |
| --- | --- | --- |
| QuickDraw `Rect` (`top,left,bottom,right` `SInt16`) | every position field | An `image.Rectangle`-like struct, but **keep integer pixels** and keep the field order for on-disk parsing |
| QuickDraw `Point` (`v,h` — **v first**) | `houseType.initial`, `gameType.where` | Parse as `v` then `h`; do not assume `(x, y)` |
| `CopyMask(srcMap, maskMap, dstMap, &src, &mask, &dst)` | `RenderGlider`, `DrawReflection` | Alpha-masked blit from a sprite atlas; the mask is a **separate 1-bit PICT** (4999) shared by all four glider atlases |
| GWorlds (offscreen `PixMap`s): `glidSrcMap`, `glid2SrcMap`, `glidMaskMap`, `shadowSrcMap`, `shadowMaskMap`, `workSrcMap`, `backSrcMap` | atlas storage and the double buffer | Textures / image buffers. `workSrcMap` is the composited frame; `backSrcMap` is the clean background used to erase dirty rects |
| Dirty-rectangle lists (`AddRectToWorkRects`, `AddRectToBackRects`, `CopyRectsQD`, `CopyRectWorkToMain`) and the `whole`/`wholeShadow` swept rects | `MoveGlider`, `HideGlider`, every `Start…` | **Delete all of it** and redraw the full frame. But keep `whole` if you want bit-exact behaviour of `StartGliderFadingOut`'s `RectTall(dest) > kGliderHigh` test and `FlagGliderShredding`'s `dest.left > whole.left` test, which read `whole` as state |
| Resource Manager (`GetPicture`, PICT IDs, `mask = PICT + 1000`) | `InitGliderMap`, `DeckGliderInFoil` | Extract PICTs once to PNG/atlas at build time; the +1000 mask convention is only a naming rule |
| `SetClip(mirrorRgn)` / `RgnHandle` | `DrawReflection` | A scissor rectangle or stencil for the mirror area |
| Sound Manager `PlayPrioritySound(id, priority)` | ~40 call sites in the player code | A mixer with a priority-preemption policy; the 62 sound IDs (spanning 0 = `kHitWallSound` to 63 = `kTriggerSound`) and the 61 priorities (spanning 100 = `kHitWallPriority` to 999 = `kTriggerPriority`) are tabulated at `GliderPRO/Headers/GliderDefines.h:55`–`:180` |
| `GetKeys(KeyMap)` + `BitTst(&map, bitOffset)` with raw virtual key codes | `GetInput` | A per-frame keyboard snapshot indexed by scancode; the four `*Key` fields store the raw offsets (see §10.1) |
| `TickCount()` busy-wait, `kTicksPerFrame = 2` | `RenderFrame` | A fixed 30 Hz (strictly 60.15/2 = 30.07 Hz) simulation step. **Do not** convert velocities to per-second units — everything is per-frame integers |
| `WaitNextEvent(..., sleep = 2)` | `HandlePlayEvent` | Event pumping; irrelevant to physics |
| 8-bit indexed colour (`kPreferredDepth`), `clut`/`cctb`/`dctb` resources | atlas depth | Convert to RGBA at extraction time |
| Big-endian on-disk data (`houseType`, `roomType`, `gameType`) | house loading | `binary.BigEndian` everywhere; `Str27`/`Str31`/`Str255` are Pascal strings (length byte first) in **Mac Roman** |
| `Handle`, `HLock`/`HGetState`/`HSetState`, `**houseHand` | house access | Plain slices |
| QuickTime `MoviesTask` | `PlayGame`, `MoveRoomToRoom` | Optional; unrelated to physics |
| 68k/PPC `short` = 16 bits with silent wraparound | every velocity and counter | Use `int16` if you want bit-exact overflow, or `int` if you have proved no field can exceed ±32767. `wasMode` counting to 150, `frame` to 322-ish and `hVel` clamped to ±16 are all safe; the shredder `frame` holds a y coordinate and the idle `hVel` holds 30, also safe |

---

## 21. Quick reference: the complete physics pipeline for one frame

For a glider in `kGliderNormal` with no special objects nearby:

```
 1  evenFrame ^= 1
 2  // --- GetInput ---
 3  keys = sample()
 4  heldLeft = heldRight = false
 5  if right && left:      ToggleGliderFacing()          // -> mode 7 or 8
 6  elif right:            hDesiredVel += 5; tipped = (facing == left);  heldRight = true
 7  elif left:             hDesiredVel -= 5; tipped = (facing == right); heldLeft  = true
 8  else:                  tipped = false
 9  if battKey && batteryTotal > 0 && mode == kGliderNormal:  hVel += (facing^tipped ? +8 : -8); batteryTotal--
10  if battKey && batteryTotal < 0 && mode == kGliderNormal:  vDesiredVel = -4; batteryTotal++
11  if bandKey && bandsTotal > 0 && !fireHeld:  AddBand(...); hVel -= bandHVel/2
12  // --- HandleInteraction ---
13  for each active hot spot overlapping the glider: apply its action
14      (may add to hDesiredVel, overwrite vDesiredVel, set sliding/ignore*, or change mode)
15  CheckGliderInRoom()                                  // walls/floor/ceiling/roof
16  // --- HandleBands ---
17  advance bands; a band hitting the glider does hVel += bandHVel/2
18  // --- HandleGlider -> MoveGliderNormal ---
19  pick sprite from (facing, sliding, tipped);  sliding = false
20  // --- MoveGlider ---
21  hVel  = approach(hVel, hDesiredVel, 2);   hDesiredVel = 0
22  vVel  = approach(vVel, vDesiredVel, 2);   vDesiredVel = 3
23  clamp |hVel| <= 16   (only inside the sign branches; no vertical clamp)
24  wasHVel = hVel;  wasVVel = vVel   // both unconditional -- a zero-velocity axis takes the else branch and is reset to 0
25  dest += (hVel, vVel);  destShadow += (hVel, 0);  update whole/wholeShadow
26  // --- end of HandleGlider ---
27  ignoreLeft = ignoreRight = ignoreGround = false
28  // --- RenderFrame ---
29  draw reflection, scenery, dynamics, then the glider, then shreds and bands
30  busy-wait to 2 ticks
```

Steps 9 and 10 are one statement in the C, not two: the guard is
`battKey && batteryTotal != 0 && mode == kGliderNormal`
(`GliderPRO/Sources/Input.c:330`–`:331`) and the sign of `batteryTotal` then
picks `DoBatteryEngaged` over `DoHeliumEngaged` inside it (`:336`–`:339`). The
`else` on that guard sets `batteryWasEngaged = false` (`:341`–`:342`), which is
what makes the next engaged frame restart the `kThrustSound`/`kHissSound` phase;
see §10.4 and §10.5. Note also that steps 13–15 run *before* step 21, so
everything the world writes in `HandleInteraction` is filtered by the ramp on
its way into `dest` — see §7.2 item 1.

---

## 22. Open questions

These are the points I could not settle from the source alone. Each is phrased so
that a single behavioural test against the original binary would answer it.

1. **`Sampler.binhex` has 2 trailing bytes.** Observed:
   `len 1564, (len − 866) = 698, / 348 = 2.0057471264367814`. 866 + 2 x 348 =
   1562, so there are two bytes past the last complete room. Either the format
   allows trailing padding, or `Sampler` was written by a different tool version.
   A loader should ignore a short trailing remainder rather than reject the file.
   Every other house I decoded divided exactly.

2. **The bare literal `315`** in `MoveGliderDownDuct`
   (`GliderPRO/Sources/Player.c:704`: `vNotClipped = 315 - thisGlider->dest.top;`).
   It is not any named constant. `kFloorTransTop` (302) plus the floor
   transporter graphic height (15) is 317, not 315; `kShadowTop + kShadowHigh`
   is 315 but that is almost certainly coincidence. Is 315 the intended plane, or
   a two-pixel fudge?

3. **The bare literal `29`** in `MoveGliderUpStairs`
   (`GliderPRO/Sources/Player.c:336`: `vNotClipped = thisGlider->dest.bottom - 29;`).
   Same question. 29 is not `kCeilingTransTop`, not `kCeilingLimit` (8) and not
   `kGliderStartsDown − 3` (29 *is* `kGliderStartsDown − 3`, so this may simply be
   an unrolled `kGliderStartsDown - 3`; it is worth confirming which of the two
   readings the author intended before renaming it in a port).

4. **`kRoof` rooms have a floor but no shadow.** `DoesRoomHaveFloor`
   (`GliderPRO/Sources/Room.c:1138`–`:1168`) returns false only for `kSky`,
   `kStratosphere` and `kStars`, so `kRoof` has `bottomOpen == false`; but
   `IsShadowVisible` (`GliderPRO/Sources/Room.c:1103`–`:1134`) returns false for
   `kRoof` as well as those three. So on a roof the glider casts no shadow yet
   `dest.bottom > kFloorLimit` still kills it. Intentional (you are meant to fall
   through a roof tile via `CheckRoofCollision` long before reaching
   `kFloorLimit`) or an oversight?

5. **`wasVVel` is written and never read.** `MoveGlider` maintains it at
   `GliderPRO/Sources/Player.c:131` and `:140`, and an exhaustive search of the
   tree finds no reader — whereas `wasHVel` (written at `:99` and `:116`) is read
   by `GliderHitTop` at `GliderPRO/Sources/Interactions.c:65`, `:66`, `:86`,
   `:87`. Was a vertical "un-sweep" planned and never implemented?

6. **The swapped comments in `GetDemoInput`** (`GliderPRO/Sources/Input.c:228`
   says `// left key` on the branch that presses right, and `:236` says
   `// right key` on the branch that presses left). Harmless in the C, but it
   means the recorded `demo` resource's key numbering should be verified by
   actually replaying it before trusting a Go reimplementation of the demo.

7. **Are PICT 4974 / 4976 / 4963 genuinely absent, or stripped from the derez
   dump?** The resource census of `GliderPRO/Glider PRO.r` found no such
   resources, and `InitGliderMap` only ever loads mask PICT 4999, so the code
   agrees that one mask serves all four glider atlases. Still worth confirming
   the four atlases really do have pixel-identical silhouettes before sharing one
   alpha channel in a port.

8. **`FinishGliderDuctingIn` declares a macro it never uses.**
   `GliderPRO/Sources/Player.c:929` is `#define kVDropStairsSpeed 4` inside
   `FinishGliderDuctingIn`, but the body then uses `kVDropDuctSpeed`
   (`:936`–`:937`). Both are 4, so the behaviour is identical, but this is the
   third textual redefinition of `kVDropStairsSpeed` in the file (`:433`, `:514`,
   `:929`). Was a different duct speed intended?

9. **`otherPlayerEscaped` semantics for `kPlayerIsDeadForever`.** `OffAMortal`
   sets it to −69 (`GliderPRO/Sources/Player.c:1602`) so that the surviving
   player's escape tests always take the third ("wrong exit") branch and the
   room never changes. But `FollowTheLeader` also reads and clears it
   (`GliderPRO/Sources/Transit.c:479`–`:480`) and switches on the saved value.
   What does `FollowTheLeader` do when `wasEscaped == kPlayerIsDeadForever`? Its
   switch has no such case, so it falls through with no room change — probably
   correct, but unverified.

10. **Frame pacing when the machine is too slow.** `RenderFrame` busy-waits until
    `TickCount() >= nextFrame` and then sets `nextFrame = TickCount() + 2`. If a
    frame overruns, the deadline slips rather than catching up, so the game slows
    down instead of dropping frames. A Go port that instead runs a fixed-step
    accumulator with catch-up will behave differently under load (and in demos).
    Which behaviour is "correct" for the port is a design decision, not a fact
    recoverable from the source.

## 23. Porting notes

Ordered by how likely each is to be got wrong.

1. **Keep everything as 16-bit integers in pixels-per-frame.** Do not introduce
   floats, delta-time, or sub-pixel accumulation. The game is frame-locked at 2
   Mac ticks (30.07 fps) and every constant is tuned to integer arithmetic. If you
   want variable refresh, run the simulation in fixed 30 Hz steps and interpolate
   only for display.

2. **Reproduce `MoveGlider` statement for statement**, including: the reset of
   `hDesiredVel` to 0 and `vDesiredVel` to `kGravity` *inside* the mover; the
   overshoot clamps in the ramp; the `kMaxHVel` clamp living inside the sign
   branches; the **absence of any vertical clamp**; the fact that `wasHVel` /
   `wasVVel` are rewritten on **every** call, including a frame the glider spent
   stationary; and the one-sided `whole` edge writes.

   The last of those is easy to get backwards, because the assignments sit inside
   the two halves of a sign test and look conditional. They are not: the split is
   `if (vel < 0) {…} else {…}` (Player.c:96-146), so a velocity of zero takes the
   positive branch and the assignment still runs. It matters because
   `GliderHitTop` rewinds the hit box by `wasHVel` to decide whether the glider
   landed on an object or ran into its side (Interactions.c:65) — a glider that was
   stationary last frame must rewind by 0, not by its last non-zero velocity, or
   every landing after a stop is misread as a side impact.

3. **`vVel` is used as a positional snap in three places** — `kSlideIt`
   (`vVel = surfaceTop - dest.bottom`), `CheckEscapeUp`/`Down`
   (`vVel = limit - dest.edge`), and `BounceGlider`
   (`hVel = overlapDistance`). These can be arbitrarily large. Any clamp you add
   for "safety" will break grease, wall bounces and ceiling snaps.

4. **`frame`, `wasMode` and `hVel` are each overloaded.** `frame` is a fade index,
   a sprite index, an animation phase, a burning sentinel, a y coordinate *and* a
   negative countdown depending on mode. `wasMode` is a burn fuse, a web counter
   and a saved mode. `hVel` is the idle countdown. Either replicate the aliasing
   or write a very careful mapping and test every transition.

5. **The one-frame flags.** `ignoreLeft`, `ignoreRight`, `ignoreGround` are set by
   `HandleInteraction` and cleared at the very end of `HandleGlider`; `sliding`
   is set by `HandleInteraction` and cleared inside `MoveGliderNormal`; `tipped`
   is recomputed by `GetInput` every frame; `heldLeft`/`heldRight` are recomputed
   by `GetInput` **except while burning**, when they retain their old values.

6. **Preserve the frame order exactly**: input → hot spots → wall checks →
   triggers → bands → glider movement → render. In particular the wall checks run
   *before* the move, so all wall/ceiling/floor reactions are one frame late and
   the bounce formulas compensate by adding the penetration depth.

7. **`facing` only changes through the about-face tumble.** `MoveGliderNormal`
   reads `facing` and never writes it. Pressing both direction keys is the only
   in-flight way to turn, and it costs 3 frames in mode 7/8 during which
   `MoveGlider` still runs but sprite/state selection is suspended.

8. **Respawn is `dest = enteredRect`**, and `enteredRect` is a *canonical* rect
   for horizontal room entries (flush against the wall at
   `32 + leftStart − 2`) but the *actual* position for vertical entries and for
   every arrival animation. Get this wrong and death loops become possible.

9. **The shredder's 68-frame delay** (`kShredderCountdown` = −68, applied at
   `GliderPRO/Sources/Player.c:1303`) sits between the confetti spawning and the
   life being lost. During those 68 frames the glider is still in mode 20, still
   drawn (clipped to zero height), and `shadowVisible` has been forced to `false`
   globally — which suppresses the *other* player's shadow too in a two-player
   game until someone runs `FlagGliderNormal`.

10. **Shared globals, not per-player state.** `batteryTotal`, `bandsTotal`,
    `foilTotal`, `mortals`, `showFoil`, `shadowVisible`, `theScore`,
    `otherPlayerEscaped`, `takingTheStairs`, `firstPlayer`, `rightClip`,
    `leftClip`, `transRoom`, `linkedToWhat`, `transRect`, `batteryFrame` and
    `batteryWasEngaged` are all single globals shared by both gliders. Moving any
    of them into a per-player struct will change behaviour. The last two look most
    like per-glider state and are not: they are the battery/helium sound phase
    (`GliderPRO/Sources/Input.c:33`–`:34`), and putting them in the struct silently
    changes the two-player thrust cadence (§10.4).

11. **`GetKeys` is sampled once per frame for player 1 only**, and player 2 reads
    the same snapshot. Player 2's keys are hard-coded modifiers (Control,
    Command, Option, Shift) that a modern OS may swallow — plan a remap.

12. **Reproduce (or consciously fix) the two confirmed original bugs.**
    (a) `IsRectLeftOfRect` (`GliderPRO/Sources/RectUtils.c:185`) computes
    `offset = (rect1->right - rect1->left) - (rect2->right - rect2->left) / 2;`
    — that is `width1 − width2/2`, not `(width1 − width2)/2`, because `/` binds
    tighter than `-`. It decides which way a foil shove pushes the glider in
    `CheckDynamicCollision`, so the bias is gameplay-visible.
    (b) `AddAShreddedGlider` (`GliderPRO/Sources/DynamicMaps.c:732`) guards with
    `if (numShredded > kMaxShredded) return;` — `>` rather than `>=`. `shreds`
    is a `shredPtr` allocated as exactly `sizeof(shredType) * kMaxShredded` with
    `kMaxShredded == 4` (`GliderPRO/Headers/GliderDefines.h:264`,
    `GliderPRO/Sources/StructuresInit2.c:256`), so a fifth simultaneous shredding
    writes `shreds[4]`, one element past the end of the block. In Go this is a
    panic rather than silent corruption, so the bound **must** be `>=`.

13. **Integer division truncates toward zero** in C for the values used here
    (`hVel /= 4`, `vVel /= 2`, `bands[i].hVel / 2`, `hDist >> 3`). Note the
    asymmetry: `/` truncates toward zero but `>>` floors. `WebGlider` uses `>> 3`
    on a value that can be negative, so `-9 >> 3 == -2` whereas `-9 / 8 == -1`.
    Go's `>>` on a signed integer also floors, so a direct transcription is
    correct — but do not "clean it up" into a division.

14. **Sprite atlas layout is load-bearing.** 48 x 668 with rows of 20 px for
    indices 0–20, 26 px for 21–28 (starting at y = 420) and 20 px for 29–30
    (628 and 648). The burning sprites being 6 px taller is why `dest` changes
    height between modes 9 and 0, and why `SectGlider` adds 6 to `glideBounds.top`
    for burning gliders.

15. **Sound cadence is gameplay-visible.** `kShredSound` plays *every frame*
    while the glider is still being pulled into the shredder
    (`GliderPRO/Sources/Player.c:1287`) and again *every frame* while each
    confetti pile grows (`GliderPRO/Sources/Render.c:585`, ~35 frames per pile).
    `kThrustSound` / `kHissSound` play only on `batteryFrame == 0` of a 0..3 cycle
    (`GliderPRO/Sources/Input.c:147`–`:153`, `:173`–`:179`), which in a one-player
    game means every 4th frame of a sustained hold and immediately on a fresh
    press. In a two-player game the cycle collapses, because `batteryFrame` and
    `batteryWasEngaged` are globals and the reset at `:342` fires for whichever
    glider did *not* thrust this frame: one thruster then sounds every frame, two
    thrusters every 2nd frame (§10.4).
    `kFollowSound` plays at most 3 times per game
    (`saidFollow < 3` at `GliderPRO/Sources/Modes.c:462`, incremented at `:465`,
    zeroed by `NewGame` at `GliderPRO/Sources/Play.c:115`). A port that
    de-duplicates or rate-limits sounds generically will feel wrong.
